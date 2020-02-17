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
	"strings"
	"time"
)

const MIGRATIONS_DIR = "migrations"

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
	case "mysql":
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
	InitMigration	bool
	NewMigration	string
	Migrate		*flag.FlagSet
	MigrateSingle	string
	MigrateAll	bool
}

func initFlags() *InputFlags {
	var flags InputFlags

	flag.BoolVar(&flags.InitMigration, "init", false, "gomi init")
	flag.StringVar(&flags.NewMigration, "new", "", "gomi new `migration_name`")
	flag.Parse()

	migrate_fs := flag.NewFlagSet("migrate", flag.ExitOnError)
	migrate_fs.StringVar(&flags.MigrateSingle, "name", "", "migration name")
	migrate_fs.BoolVar(&flags.MigrateAll, "all", false, "migrate all")
	flags.Migrate.Parse()

	return &flags
}

func (flags *InputFlags) CheckParams() {
	if len(os.Args) < 2 {
		log.Fatal("Invalid command line arguments")
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

	content := fmt.Sprintf("--Migration name: %s", migration_name)
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

	if flags.InitMigration {
		qStr := initMigrationsSql()
		if err := execSql(db, string(qStr)); err != nil {
			log.Fatal(err)
		}

	}

	if flags.NewMigration != "" {
		migration_name := flags.NewMigration
		if err := generateMigration(migration_name); err != nil {
			log.Fatal(err)
		}
	}

	if flags.MigrateSingle != "" {
		migration_name := strings.Split(flags.MigrateSingle, "/")[1]
		migrated, err := checkMigrated(migration_name, db)
		if err != nil {
			log.Fatal(err)
		}

		if !migrated {

			sql, err := ioutil.ReadFile(flags.MigrateSingle)
			if err != nil {
				log.Fatal(err)
			}

			if err := execSql(db, string(sql)); err != nil {
				log.Fatal(err)
			}

			if err := trackMigration(migration_name, db); err != nil {
				log.Fatal(err)
			}
			log.Println("Migration completed")
		} else {
			log.Println("Migration already done")
		}
	}
}
