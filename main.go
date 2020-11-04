package main

import (
	"errors"
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	"gitlab.com/vpenkoff/gomi/drivers"
	"gitlab.com/vpenkoff/gomi/utils"
	"io/ioutil"
	"log"
	"path/filepath"
	"text/tabwriter"
	"os"
)

const DEFAULT_MIGRATIONS_DIR = "./migrations"
const DEFAULT_CFG_PATH = "./config.json"

var CONFIG_FILE_FIELDS = []string{"host", "port", "protocol", "driver", "username", "password", "dbname"}

// flags
var f_cfg_path string
var f_init bool
var f_migrate bool
var f_migrate_all bool
var f_migrate_new bool
var f_migration_name string
var f_migration_dir string
var f_help bool

type MigrationAction int
const (
	MigrationInit MigrationAction = iota
	MigrationMigrateAll
	MigrationMigrateSingle
	MigrationNew
	Help
)

func init() {
	flag.StringVar(&f_cfg_path, "config", DEFAULT_CFG_PATH, "config file")
	flag.BoolVar(&f_init, "init", false, "init migrations table")
	flag.BoolVar(&f_migrate, "migrate", false, "do migration")
	flag.BoolVar(&f_migrate_all, "all", false, "migrate all migrations")
	flag.BoolVar(&f_migrate_new, "new", false, "create new migration")
	flag.StringVar(&f_migration_name, "name", "", "migration name")
	flag.StringVar(&f_migration_dir, "dir", DEFAULT_MIGRATIONS_DIR, "migration directory")
	flag.BoolVar(&f_help, "help", false, "help message")
}

func main() {
	flag.Parse()

	config, err := readConfig(f_cfg_path)
	if err != nil {
		log.Fatal(err)
	}

	dbDriver, err := drivers.GetDriverFromConfig(config)
	if err != nil {
		log.Fatal(err)
	}

	defer dbDriver.CloseConn()

	action, err := getMigrationAction()
	if err != nil {
		log.Fatal(err)
	}

	if err := executeMigrationAction(action, dbDriver); err != nil {
		log.Fatal(err)
	}
}

func readConfig(cfg_path string) (interface{}, error) {
	var path string
	if cfg_path != "" {
		path = cfg_path
	} else {
		path = DEFAULT_CFG_PATH
	}
	config, err := ioutil.ReadFile(path)

	if err != nil {
		return nil, err
	}

	reader := bytes.NewReader(config)

	var decoded interface{}

	if err := json.NewDecoder(reader).Decode(&decoded); err != nil {
		return nil, err
	}

	if valid := validateConfig(decoded); !valid {
		return nil, errors.New("Config is invalid")
	}

	return decoded, nil
}

func validateConfig(config interface{}) bool {
	c, ok := config.(map[string]interface{})
	if !ok {
		return false
	}

	if len(c) != len(CONFIG_FILE_FIELDS) {
		return false
	}

	valid := false

	for field ,_ := range c {
		valid = checkConfigFieldValid(field)
	}

	return valid
}

func checkConfigFieldValid(field string) bool {
	valid := false

	for _, f := range CONFIG_FILE_FIELDS {
		if field == f {
			valid = true
		}
	}

	return valid
}

func getMigrationAction() (MigrationAction, error) {
	if f_init {
		return MigrationInit, nil
	}

	if f_migrate && f_migrate_all {
		return MigrationMigrateAll, nil
	}

	if f_migrate && f_migration_name != "" {
		return MigrationMigrateSingle, nil
	}

	if f_migrate_new && f_migration_name != "" {
		return MigrationNew, nil
	}

	if f_help {
		return Help, nil
	}

	return Help, nil
}

func executeMigrationAction(action MigrationAction, db drivers.Driver) error {
	switch action {
	case MigrationInit:
		return exec_migration_init(db)
	case MigrationMigrateAll:
		return exec_migration_migrate_all(db)
	case MigrationMigrateSingle:
		return exec_migration_migrate_single(db)
	case MigrationNew:
		return exec_migration_new()
	case Help:
		return printDefaults()
	default:
		return printDefaults()
	}
}

func exec_migration_init(db drivers.Driver) error {
	if err := db.InitMigrationTable(); err != nil {
		return err
	}
	log.Printf("Init migration table completed")
	return nil
}

func exec_migration_migrate_all(db drivers.Driver) error {
	if !utils.ExistsDir(f_migration_dir) {
		return fmt.Errorf("Directory %s does not exist!", f_migration_dir)
	}

	abs_dir, err := filepath.Abs(f_migration_dir)
	if err != nil {
		return err
	}

	migrations, err := filepath.Glob(abs_dir + "/" + "*.sql")
	if err != nil {
		return err
	}

	for _, migration := range migrations {
		if err := db.Migrate(migration); err != nil {
			return fmt.Errorf("Migration: %s || error: %s\n", migration, err)
		}
	}
	log.Printf("All migrations done")
	return nil
}

func exec_migration_migrate_single(db drivers.Driver) error {
	if err := db.Migrate(f_migration_name); err != nil {
		return err
	}
	log.Printf("Migration %s completed", f_migration_name)
	return nil
}

func exec_migration_new() error {
	migrations_dir, err := utils.SetupMigrationsDir(f_migration_dir)
	if err != nil {
		return err
	}

	if err := utils.GenerateMigration(migrations_dir, f_migration_name); err != nil {
		return err
	}

	log.Printf("Migration %s created!\n", f_migration_name)
	return nil
}

func printDefaults() error {
	w := tabwriter.NewWriter(os.Stderr, 0, 8, 1, '\t', 0)
	fmt.Fprintln(w, "Usage: gomi -config=CONFIG_FILE [ACTION...]")
	fmt.Fprintln(w, "-config\t\t\tconfiguration file for connecting to database")
	fmt.Fprintln(w, "ACTIONS:")
	fmt.Fprintln(w, "-init\tcreate migrations table into the specified database to track migrations")
	fmt.Fprintln(w, "-migrate -all [-dir]\tmigrate all migrations from the specified directory [-dir]." +
					"\n\tBy default the directory is 'migrations' in the current working directory.")
	fmt.Fprintln(w, "-migrate -name [-dir]\tmigrate migration with name [-name] from directory [-dir]." +
					"\n\tBy default the directory is 'migrations' in the current working directory.")
	fmt.Fprintln(w, "-new -name [-dir]\tcreate new migration [-name] in the specified directory [-dir]." +
					"\n\tBy default the directory is 'migrations' in the current working directory.")
	fmt.Fprintln(w, "-help\tprint this message")
	w.Flush()
	return nil
}
