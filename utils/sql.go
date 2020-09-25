package utils

import (
	"database/sql"
)

func BeginTx(db *sql.DB) (*sql.Tx, error) {
	tx, err := db.Begin()
	if err != nil {
		return nil, err
	}

	return tx, nil
}

func ExecTx(tx *sql.Tx, sql string, args ...interface{}) error {
	_, err := tx.Exec(sql, args...)
	if err != nil {
		return err
	}

	return nil
}

func CommitTx(tx *sql.Tx) error {
	if err := tx.Commit(); err != nil {
		return err
	}

	return nil
}

func RollbackTx(tx *sql.Tx) error {
	if err := tx.Rollback(); err != nil {
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

func Exec(db *sql.DB, qStr string, args ...interface{}) (sql.Result, error) {
	return db.Exec(qStr, args...)
}
