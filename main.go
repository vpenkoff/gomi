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
	"reflect"
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
	Action        string
}

func initFlags() *InputFlags {
	var flags InputFlags

	flag.BoolVar(&flags.InitMigration, "init", false, "gomi init")
	flag.StringVar(&flags.NewMigration, "new", "", "gomi new `migration_name`")
	flag.StringVar(&flags.Action, "migrate", "", "gomi migrate")
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

func runSql(sql string, db *sql.DB) {
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
		runSql(qStr, db)

	}

	/*


		sql, err := ioutil.ReadFile("./dump.sql")
		if err != nil {
			log.Fatal(err)
		}

	*/
}
