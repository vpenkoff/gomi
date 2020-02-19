package drivers

import (
	"database/sql"
	"errors"
	"github.com/vpenkoff/gomi/drivers/mysql"
)

const DRIVER_MYSQL = "mysql"

type Driver interface {
	InitDriver(interface{}) error
	GetDB() *sql.DB
	GetSqlPlaceholder() string
}

func GetDriverFromConfig(config interface{}) (Driver, error) {
	driver := config.(map[string]interface{})["driver"].(string)

	switch driver {
	case DRIVER_MYSQL:
		return mysql.GetDriver(config)
	}
	return nil, errors.New("Could not get the driver specified in the config")
}
