package drivers

import (
	"errors"
	"gitlab.com/vpenkoff/gomi/drivers/mysql"
	"gitlab.com/vpenkoff/gomi/drivers/postgres"
)

const DRIVER_MYSQL = "mysql"
const DRIVER_PGSQL = "postgres"

type Driver interface {
	InitMigrationTable() error
	CheckMigrated(string) (bool, error)
	TrackMigration(string) error
	Migrate(string) error
}

func GetDriverFromConfig(config interface{}) (Driver, error) {
	driver := config.(map[string]interface{})["driver"].(string)

	switch driver {
	case DRIVER_MYSQL:
		return mysql.GetDriver(config)
	case DRIVER_PGSQL:
		return postgres.GetDriver(config)
	}
	return nil, errors.New("Could not get the driver specified in the config")
}
