package mysql

import (
	"database/sql"
	"errors"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	"gitlab.com/vpenkoff/gomi/utils"
	"log"
)

const DRIVER_MYSQL = "mysql"

type driver struct {
	DriverName string
	DSN        string
	DB         *sql.DB
}

var MySQLDriver driver

func parseConfig(config interface{}) string {
	config_map := config.(map[string]interface{})

	return fmt.Sprintf("%s:%s@%s(%s:%s)/%s?multiStatements=true",
		config_map["username"],
		config_map["password"],
		config_map["protocol"],
		config_map["host"],
		config_map["port"],
		config_map["dbname"],
	)
}

func (d *driver) CloseConn() error {
	return d.DB.Close()
}

func (d *driver) InitDriver(config interface{}) error {
	d.DSN = parseConfig(config)
	d.DriverName = DRIVER_MYSQL

	db, err := sql.Open(d.DriverName, d.DSN)
	if err != nil {
		return err
	}
	d.DB = db
	MySQLDriver = *d
	return nil
}

func GetDriver(config interface{}) (*driver, error) {
	if err := MySQLDriver.InitDriver(config); err != nil {
		return nil, err
	}

	return &MySQLDriver, nil
}

func (d *driver) InitMigrationTable() error {
	qStr := `
		CREATE TABLE migrations(
			id INT NOT NULL AUTO_INCREMENT PRIMARY KEY,
			name VARCHAR(255) NOT NULL,
			created_at TIMESTAMP NOT NULL
		);`
	if _, err := utils.Exec(d.DB, qStr); err != nil {
		return err
	}
	return nil
}

func (d *driver) CheckMigrated(migration_name string) (bool, error) {
	qStr := `
		SELECT 1
		FROM migrations
		WHERE name = ?;
	`

	var migrated int

	if err := utils.QuerySingle(d.DB, qStr, migration_name).Scan(&migrated); err != nil {
		switch {
		case err == sql.ErrNoRows:
			return false, nil
		case err != nil:
			return false, err
		}
	}
	return true, nil
}

func (d *driver) TrackMigration(tx *sql.Tx, migration_name string) error {
	qStr := `
		INSERT INTO migrations(
			name,
			created_at
		)
		VALUES (?, now())
	`
	return utils.ExecTx(tx, qStr, migration_name)
}

func (d *driver) Migrate(migration_path string) error {
	migration_name := utils.GetMigrationName(migration_path)
	migration, err := utils.ReadMigration(migration_path)
	if err != nil {
		return err
	}

	migrated, err := d.CheckMigrated(migration_name)
	if err != nil {
		return err
	}

	if migrated {
		return errors.New("Migration already migrated")
	}

	tx, err := utils.BeginTx(d.DB)
	if err != nil {
		log.Printf("Unable to start tx: %v\n", err)
		return err
	}

	if err := utils.ExecTx(tx, string(migration)); err != nil {
		log.Printf("Error executing tx: %v\n", err)
		log.Println("Rolling back...")

		if err := utils.RollbackTx(tx); err != nil {
			log.Printf("Unable to rollback: %v\n", err)
			return err
		}

		return err

	}

	if err := d.TrackMigration(tx, migration_name); err != nil {
		if err := utils.RollbackTx(tx); err != nil {
			log.Printf("Unable to rollback: %v\n", err)
			return err
		}
		return err
	}

	if err := utils.CommitTx(tx); err != nil {
		log.Printf("Unable to commit tx: %v\n", err)
		return err
	}

	return nil
}
