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
	"reflect"
	"time"
)

type Config struct {
	Driver string `json:"driver"`
	DSN    string `json:"dsn"`
}

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
	InitMigration bool
	NewMigration  string
	Migrate       string
}

func initFlags() *InputFlags {
	var flags InputFlags

	flag.BoolVar(&flags.InitMigration, "init", false, "gomi init")
	flag.StringVar(&flags.NewMigration, "new", "", "gomi new `migration_name`")
	flag.StringVar(&flags.Migrate, "migrate", "", "gomi migrate")
	flag.Parse()

	return &flags
}

func (flags *InputFlags) CheckParams() error {
	opts_count := 0

	fields := reflect.TypeOf(*flags)
	values := reflect.ValueOf(*flags)

	for i := 0; i < fields.NumField(); i++ {
		if !values.Field(i).IsZero() {
			opts_count++
		}
	}
	if opts_count != 1 {
		return errors.New("Invalid command line arguments")
	}
	return nil
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

func execSql(sql string, db *sql.DB) {
	tx, err := db.Begin()
	if err != nil {
		log.Fatal(err)
	}

	_, err = tx.Exec(string(sql))
	if err != nil {
		if rollbackErr := tx.Rollback(); rollbackErr != nil {
			log.Fatalf("update drivers: unable to rollback: %v", rollbackErr)
		}
		log.Fatal(err)
	}

	if err := tx.Commit(); err != nil {
		log.Fatal(err)
	}
}

func querySingleSql(db *sql.DB, qStr string, args ...interface{}) *sql.Row {
	return db.QueryRow(qStr, args)
}

func querySql(db *sql.DB, qStr string, args ...interface{}) (*sql.Rows, error) {
	rows, err := db.Query(qStr, args)
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
	dir_name := "migrations"
	if _, err := os.Stat(dir_name); os.IsNotExist(err) {
		return false
	}

	return true
}

func generateMigration(name string) error {
	migration_name := generateMigrationName(name)

	if !existsMigrationDir() {
		if err := os.Mkdir("migrations", 0755); err != nil {
			return err
		}
	}

	content := fmt.Sprintf("--Migration name: %s", migration_name)
	byte_content := []byte(content)
	file_name := fmt.Sprintf("migrations/%s", migration_name)
	if err := ioutil.WriteFile(file_name, byte_content, 0644); err != nil {
		return err
	}

	return nil
}

func checkIfMigrated(migration string, db *sql.DB) (bool, error) {
	qStr := `
		SELECT 1
		FROM migrations
		WHERE name = $1
	`
	var migrated int

	if err := querySingleSql(db, qStr).Scan(&migrated); err != nil {
		switch {
		case err == sql.ErrNoRows:
			return false, nil
		case err != nil:
			return false, err
		}
	}
	return true, nil
}

func main() {
	config, err := readConfig()
	if err != nil {
		log.Fatal(err)
	}

	flags := initFlags()

	if err := flags.CheckParams(); err != nil {
		flag.PrintDefaults()
		log.Fatal(err)
	}

	db, err := getDB(config)
	if err != nil {
		log.Fatal(err)
	}

	defer db.Close()

	if flags.InitMigration {
		qStr := initMigrationsSql()
		execSql(qStr, db)

	}

	if flags.NewMigration != "" {
		migration_name := flags.NewMigration
		if err := generateMigration(migration_name); err != nil {
			log.Fatal(err)
		}
	}

	if flags.Migrate != "" {
		// check if migration exists
		// check if migration is migrated
		// execute migration
	}

	/*

		sql, err := ioutil.ReadFile("./dump.sql")
		if err != nil {
			log.Fatal(err)
		}

	*/
}
