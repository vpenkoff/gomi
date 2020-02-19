package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const MIGRATIONS_DIR = "./migrations"
const DRIVER_MYSQL = "mysql"

type Config struct {
	Driver string `json:"driver"`
	DSN    string `json:"dsn"`
}

var SqlPlaceholder string

func readConfig() (*Config, error) {
	config, err := ioutil.ReadFile("./config.json")
	if err != nil {
		fmt.Println(err)
		return nil, err
	}

	reader := bytes.NewReader(config)

	var decoded interface{}

	if err := json.NewDecoder(reader).Decode(&decoded); err != nil {
		fmt.Println(err)
		return nil, err
	}

	driver := decoded.(map[string]interface{})["driver"].(string)
	config_map := decoded.(map[string]interface{})

	switch driver {
	case DRIVER_MYSQL:
		SqlPlaceholder = "?"
		dsn := fmt.Sprintf("%s:%s@%s(%s:%s)/%s",
			config_map["username"],
			config_map["password"],
			config_map["protocol"],
			config_map["host"],
			config_map["port"],
			config_map["dbname"],
		)
		return &Config{Driver: driver, DSN: dsn}, nil
	}

	return nil, errors.New("Could not load config")
}

type InputFlags struct {
	InitCmd           *flag.FlagSet
	MigrationCmd      *flag.FlagSet
	MigrateCmdNew     *string
	MigrateCmdAll     *bool
	MigrateCmdSingle  *string
}

func initFlags() *InputFlags {
	var inputFlags InputFlags

	inputFlags.InitCmd = flag.NewFlagSet("init", flag.ExitOnError)
	inputFlags.MigrationCmd = flag.NewFlagSet("migrate", flag.ExitOnError)

	inputFlags.MigrateCmdNew = inputFlags.MigrationCmd.String("new", "", "migration name")
	inputFlags.MigrateCmdAll = inputFlags.MigrationCmd.Bool("all", false, "migrate all from migrations")
	inputFlags.MigrateCmdSingle = inputFlags.MigrationCmd.String("name", "", "migrate migration")
	return &inputFlags
}

func PrintUsage() {
	fmt.Println(`
Usage:
gomi init - init migrations table
gomi migrate -new [name] - create migration with name
gomi migrate -all - migrate all migrations
gomi migrate -name [name] - migrate migration with name
`)
}

func (flags *InputFlags) CheckParams() {
	if len(os.Args) < 2 {
		PrintUsage()
		os.Exit(1)
	}

	switch os.Args[1] {
	case "init":
		flags.InitCmd.Parse(os.Args[1:])
	case "migrate":
		flags.MigrationCmd.Parse(os.Args[2:])
	default:
		PrintUsage()
		os.Exit(1)
	}
}

func getDB(config *Config) (*sql.DB, error) {
	return sql.Open(config.Driver, config.DSN)
}

func initMigrationsSql() string {
	return `
		CREATE TABLE migrations(
			id INT NOT NULL AUTO_INCREMENT PRIMARY KEY,
			name VARCHAR(255) NOT NULL,
			created_at TIMESTAMP NOT NULL
		);
	`
}

func execSql(db *sql.DB, sql string, args ...interface{}) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}

	_, err = tx.Exec(sql, args...)
	if err != nil {
		if rollbackErr := tx.Rollback(); rollbackErr != nil {
			return err
		}
		return err
	}

	if err := tx.Commit(); err != nil {
		return err
	}
	return nil
}

func querySingleSql(db *sql.DB, qStr string, args ...interface{}) *sql.Row {
	return db.QueryRow(qStr, args...)
}

func querySql(db *sql.DB, qStr string, args ...interface{}) (*sql.Rows, error) {
	rows, err := db.Query(qStr, args...)
	if err != nil {
		return nil, err
	}

	defer rows.Close()
	return rows, nil
}

func generateMigrationName(name string) string {
	today := time.Now().UTC()
	return fmt.Sprintf("%d_%s.sql", today.Unix(), name)
}

func existsMigrationDir() bool {
	if _, err := os.Stat(MIGRATIONS_DIR); os.IsNotExist(err) {
		return false
	}

	return true
}

func generateMigration(name string) error {
	migration_name := generateMigrationName(name)

	if !existsMigrationDir() {
		if err := os.Mkdir(MIGRATIONS_DIR, 0755); err != nil {
			return err
		}
	}

	content := fmt.Sprintf("-- Migration name: %s", migration_name)
	byte_content := []byte(content)
	file_name := fmt.Sprintf("%s/%s", MIGRATIONS_DIR, migration_name)
	if err := ioutil.WriteFile(file_name, byte_content, 0644); err != nil {
		return err
	}

	return nil
}

func checkMigrated(migration string, db *sql.DB) (bool, error) {
	qStr := `
		SELECT 1
		FROM migrations
		WHERE name = %s;
	`

	query := fmt.Sprintf(qStr, SqlPlaceholder)
	var migrated int

	if err := querySingleSql(db, query, migration).Scan(&migrated); err != nil {
		switch {
		case err == sql.ErrNoRows:
			return false, nil
		case err != nil:
			return false, err
		}
	}
	return true, nil
}

func trackMigration(migration string, db *sql.DB) error {
	qStr := `
		INSERT INTO migrations(
			name,
			created_at
		)
		VALUES (%s, now())
	`
	query := fmt.Sprintf(qStr, SqlPlaceholder)
	return execSql(db, query, migration)
}

func migrate(migration_path string, db *sql.DB) error {
	migration_name := strings.Split(migration_path, "/")[1]
	sql, err := ioutil.ReadFile(migration_path)
	if err != nil {
		log.Println(err)
		return err
	}

	if err := execSql(db, string(sql)); err != nil {
		log.Println(err)
		return err
	}

	if err := trackMigration(migration_name, db); err != nil {
		log.Println(err)
		return err
	}

	log.Printf("Migration %s completed", migration_name)
	return nil
}

func main() {
	config, err := readConfig()
	if err != nil {
		log.Fatal(err)
	}

	flags := initFlags()
	flags.CheckParams()

	db, err := getDB(config)
	if err != nil {
		log.Fatal(err)
	}

	defer db.Close()

	if flags.InitCmd.Parsed() {
		qStr := initMigrationsSql()
		if err := execSql(db, string(qStr)); err != nil {
			log.Fatal(err)
		}

	}

	if flags.MigrationCmd.Parsed() {
		if *flags.MigrateCmdNew != "" {
			migration_name := *flags.MigrateCmdNew
			if err := generateMigration(migration_name); err != nil {
				log.Fatal(err)
			}
			log.Printf("Migration %s created!\n", migration_name)
		}

		if *flags.MigrateCmdSingle != "" {
			migration_name := strings.Split(*flags.MigrateCmdSingle, "/")[1]
			migrated, err := checkMigrated(migration_name, db)
			if err != nil {
				log.Fatal(err)
			}
			if !migrated {
				migrate(*flags.MigrateCmdSingle, db)
			} else {
				log.Println("Migration already done")
			}
		}

		if *flags.MigrateCmdAll {
			migrations, err := filepath.Glob(MIGRATIONS_DIR + "/" + "*.sql")
			if err != nil {
				log.Fatal(err)
			}

			for _, migration := range migrations {
				migration_name := strings.Split(migration, "/")[1]
				migrated, err := checkMigrated(migration_name, db)
				if err != nil {
					log.Fatal(err)
				}

				if !migrated {
					migrate(migration, db)
				} else {
					log.Printf("Migration %s already done.Skipping...", migration_name)
				}
			}
		}
	}
}
