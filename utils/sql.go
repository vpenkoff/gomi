package utils

import (
	"database/sql"
	"log"
)

func ExecTx(db *sql.DB, sql string, args ...interface{}) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}

	_, err = tx.Exec(sql, args...)
	if err != nil {
		if rollbackErr := tx.Rollback(); rollbackErr != nil {
			log.Printf("Unable to rollback: %v", rollbackErr)
		}
		return err
	}

	if err := tx.Commit(); err != nil {
		return err
	}
	return nil
}

func QuerySingle(db *sql.DB, qStr string, args ...interface{}) *sql.Row {
	return db.QueryRow(qStr, args...)
}

func QueryAll(db *sql.DB, qStr string, args ...interface{}) (*sql.Rows, error) {
	rows, err := db.Query(qStr, args...)
	if err != nil {
		return nil, err
	}

	defer rows.Close()
	return rows, nil
}
