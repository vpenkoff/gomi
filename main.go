package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	"github.com/vpenkoff/gomi/drivers"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const DEFAULT_MIGRATIONS_DIR = "./migrations"
const DEFAULT_CFG_PATH = "./config.json"

var SqlPlaceholder string

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

func existsDir(dir string) bool {
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return false
	}

	return true
}

func generateMigration(dir, name string) error {
	migration_name := generateMigrationName(name)

	content := fmt.Sprintf("-- Migration name: %s", migration_name)
	byte_content := []byte(content)
	file_name := fmt.Sprintf("%s/%s", dir, migration_name)
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
	tmp := strings.Split(migration_path, "/")
	migration_name := tmp[len(tmp)-1]
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

func setup_migrations_dir(dir string) (string, error) {
	if dir != "" {
		abs_dir, err := filepath.Abs(dir)
		if err != nil {
			return "", err
		}

		if !existsDir(abs_dir) {
			return createDir(abs_dir)
		}
		return abs_dir, nil
	} else {
		if !existsDir(dir) {
			return createDir(DEFAULT_MIGRATIONS_DIR)
		}
		return DEFAULT_MIGRATIONS_DIR, nil
	}
}

func createDir(dir string) (string, error) {
	if err := os.Mkdir(dir, 0755); err != nil {
		return "", err
	}

	return dir, nil
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

	SqlPlaceholder = dbDriver.GetSqlPlaceholder()

	db := dbDriver.GetDB()
	defer db.Close()

	if *init_flag {
		qStr := initMigrationsSql()
		if err := execSql(db, string(qStr)); err != nil {
			log.Fatal(err)
		}
		log.Printf("Init migration table completed")
		return
	}

	if *migrate_new_flag && *migration_name_flag != "" {
		migrations_dir, err := setup_migrations_dir(*migration_dir_flag)
		if err != nil {
			log.Fatal(err)
		}

		if err := generateMigration(migrations_dir, *migration_name_flag); err != nil {
			log.Fatal(err)
		}

		log.Printf("Migration %s created!\n", migration_name_flag)
	}

	if *migrate_flag && *migration_name_flag != "" {
		tmp := strings.Split(*migration_name_flag, "/")
		migration_name := tmp[len(tmp)-1]
		migrated, err := checkMigrated(migration_name, db)
		if err != nil {
			log.Fatal(err)
		}
		if !migrated {
			migrate(*migration_name_flag, db)
		} else {
			log.Println("Migration already done")
		}
	}

	if *migrate_flag && *migrate_all_flag {
		if !existsDir(*migration_dir_flag) {
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
			tmp := strings.Split(migration, "/")
			migration_name := tmp[len(tmp)-1]
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
		log.Printf("All migrations done")
	}

	log.Printf("Bye bye...")
}
