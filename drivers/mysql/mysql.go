package mysql

import (
	"database/sql"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
)

const DRIVER_MYSQL = "mysql"
const PLACEHOLDER = "?"

type MysqlDriver struct {
	DriverName string
	DSN        string
	DB         *sql.DB
}

func parseConfig(config interface{}) string {
	config_map := config.(map[string]interface{})

	return fmt.Sprintf("%s:%s@%s(%s:%s)/%s",
		config_map["username"],
		config_map["password"],
		config_map["protocol"],
		config_map["host"],
		config_map["port"],
		config_map["dbname"],
	)
}

func (md *MysqlDriver) InitDriver(config interface{}) error {
	md.DSN = parseConfig(config)
	md.DriverName = DRIVER_MYSQL

	db, err := sql.Open(md.DriverName, md.DSN)
	if err != nil {
		return err
	}
	md.DB = db
	return nil
}

func (md *MysqlDriver) GetDB() *sql.DB {
	return md.DB
}

func (md *MysqlDriver) GetSqlPlaceholder() string {
	return PLACEHOLDER
}

func GetDriver(config interface{}) (*MysqlDriver, error) {
	driver := new(MysqlDriver)

	if err := driver.InitDriver(config); err != nil {
		return nil, err
	}

	return driver, nil
}
