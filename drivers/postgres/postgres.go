package postgres

import (
	"database/sql"
	"errors"
	"fmt"
	_ "github.com/jackc/pgx/stdlib"
	"gitlab.com/vpenkoff/gomi/utils"
	"log"
	"strings"
	"time"
)

const DRIVER_POSTGRES = "pgx"

type driver struct {
	DriverName string
	DSN        string
	DB         *sql.DB
}

type PGError struct {
	Msg  string
	Type string
}

func (e *PGError) Error() string {
	return fmt.Sprintf("PG Error: %s", e.Msg)
}

var PGDriver driver

func parseConfig(config interface{}) string {
	config_map := config.(map[string]interface{})

	return fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		config_map["host"],
		config_map["port"],
		config_map["username"],
		config_map["password"],
		config_map["dbname"],
		config_map["sslmode"],
	)
}

func (d *driver) InitDriver(config interface{}) error {
	d.DSN = parseConfig(config)
	d.DriverName = DRIVER_POSTGRES

	db, err := sql.Open(d.DriverName, d.DSN)
	if err != nil {
		return err
	}

	db.SetMaxIdleConns(1)
	db.SetMaxOpenConns(100)
	db.SetConnMaxLifetime(3600 * time.Second)

	d.DB = db
	PGDriver = *d
	return nil
}

func GetDriver(config interface{}) (*driver, error) {
	if err := PGDriver.InitDriver(config); err != nil {
		return nil, err
	}

	return &PGDriver, nil
}

func (d *driver) InitMigrationTable() error {
	qStr := `
		CREATE TABLE migrations(
			id SERIAL PRIMARY KEY,
			name VARCHAR(255) NOT NULL,
			created_at TIMESTAMP NOT NULL
		);
	`

	if err := utils.ExecTx(d.DB, qStr); err != nil {
		log.Println(&PGError{err.Error(), "custom"})
		return err
	}
	return nil
}

func (d *driver) CheckMigrated(migration_name string) (bool, error) {
	qStr := `
		SELECT 1
		FROM migrations
		WHERE name = $1;
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

func (d *driver) TrackMigration(migration_name string) error {
	qStr := `
		INSERT INTO migrations(
			name,
			created_at
		)
		VALUES ($1, now())
	`
	return utils.ExecTx(d.DB, qStr, migration_name)
}

func (d *driver) Migrate(migration_path string) error {
	migration_name := utils.GetMigrationName(migration_path)
	migration, err := utils.ReadMigration(migration_path)
	if err != nil {
		return err
	}

	migrated, err := d.CheckMigrated(migration_name)
	if err != nil {
		fmt.Println(err)
		return err
	}

	if migrated {
		return errors.New(fmt.Sprintf("Migration %s already migrated", migration_name))
	}

	statements := strings.Split(string(migration), ";")

	for _, sql := range statements {
		if err := utils.ExecTx(d.DB, sql); err != nil {
			return err
		}
	}

	if err := d.TrackMigration(migration_name); err != nil {
		return err
	}
	return nil
}
