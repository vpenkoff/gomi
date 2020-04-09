package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
	"github.com/vpenkoff/gomi/drivers"
	"github.com/vpenkoff/gomi/utils"
	"io/ioutil"
	"log"
	"path/filepath"
)

const DEFAULT_MIGRATIONS_DIR = "./migrations"
const DEFAULT_CFG_PATH = "./config.json"

func readConfig(cfg_path string) (interface{}, error) {
	var path string
	if cfg_path != "" {
		path = cfg_path
	} else {
		path = DEFAULT_CFG_PATH
	}
	config, err := ioutil.ReadFile(path)

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

	return decoded, nil
}

func main() {

	var cfg_path_flag = flag.String("cfg", DEFAULT_CFG_PATH, "config file")
	var init_flag = flag.Bool("init", false, "init migration table")
	var migrate_flag = flag.Bool("migrate", false, "do migration")
	var migrate_all_flag = flag.Bool("all", false, "migrate all")
	var migrate_new_flag = flag.Bool("new", false, "create new migration")
	var migration_name_flag = flag.String("name", "", "migration name")
	var migration_dir_flag = flag.String("dir", DEFAULT_MIGRATIONS_DIR, "migration directory")
	flag.Parse()

	config, err := readConfig(*cfg_path_flag)
	if err != nil {
		log.Fatal(err)
	}

	dbDriver, err := drivers.GetDriverFromConfig(config)
	if err != nil {
		log.Fatal(err)
	}

	if *init_flag {
		if err := dbDriver.InitMigrationTable(); err != nil {
			log.Printf("Init migration table failed")
			log.Fatal(err)
		}
		log.Printf("Init migration table completed")
		return
	}

	if *migrate_new_flag && *migration_name_flag != "" {
		migrations_dir, err := utils.SetupMigrationsDir(*migration_dir_flag)
		if err != nil {
			log.Fatal(err)
		}

		if err := utils.GenerateMigration(migrations_dir, *migration_name_flag); err != nil {
			log.Fatal(err)
		}

		log.Printf("Migration %s created!\n", *migration_name_flag)
	}

	if *migrate_flag && *migration_name_flag != "" {
		if err := dbDriver.Migrate(*migration_name_flag); err != nil {
			log.Printf("Init migration table failed")
			log.Fatal(err)
		}
		log.Printf("Migration %s completed", *migration_name_flag)
	}

	if *migrate_flag && *migrate_all_flag {
		if !utils.ExistsDir(*migration_dir_flag) {
			log.Fatalf("Directory %s does not exist!", *migration_dir_flag)
		}

		abs_dir, err := filepath.Abs(*migration_dir_flag)
		if err != nil {
			log.Fatal(err)
		}

		migrations, err := filepath.Glob(abs_dir + "/" + "*.sql")
		if err != nil {
			log.Fatal(err)
		}

		for _, migration := range migrations {
			if err := dbDriver.Migrate(migration); err != nil {
				log.Printf("Init migration table failed")
				log.Fatal(err)
			}
			log.Printf("Migration %s completed", migration)
		}
		log.Printf("All migrations done")
	}
	log.Printf("Bye bye...")
}
