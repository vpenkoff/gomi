package main

import (
	"io/ioutil"
	"database/sql"
	_ "github.com/go-sql-driver/mysql"
	"log"
	"flag"
	"fmt"
)

func main() {

	flag_init := flag.Bool("init", false, "gomi init")
	flag_new := flag.String("new", "", "gomi new `migration_name`")
	flag_migrate := flag.String("migrate", "", "gomi migrate")
	flag.Parse()

	fmt.Println(flag_init)
	fmt.Println(flag_new)
	fmt.Println(flag_migrate)

	flag.PrintDefaults()




	dsn := "debi:t00r@/test"

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	sql, err := ioutil.ReadFile("./dump.sql")
	if err != nil {
		log.Fatal(err)
	}

	tx, err := db.Begin()
	if err != nil {
		log.Fatal(err)
	}

	_, err = tx.Exec(string(sql))
	if err != nil {
		if rollbackErr := tx.Rollback(); rollbackErr != nil {
			log.Fatalf("update drivers: unable to rollback: %v", rollbackErr)
		}
		log.Fatal(err)
	}

	if err := tx.Commit(); err != nil {
		log.Fatal(err)
	}
}
