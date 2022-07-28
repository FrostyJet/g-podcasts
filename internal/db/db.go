package db

import (
	"database/sql" // add this
	"fmt"
	"log"

	_ "github.com/lib/pq" // add this
)

var db *sql.DB

func Init() {
	var err error

	connStr := getConnStr("almighty", "secret", "localhost", "5432", "google_podcasts_db")
	db, err = sql.Open("postgres", connStr)

	// Connect to database
	if err != nil {
		log.Fatal(err)
	}

	err = db.Ping()
	if err != nil {
		log.Fatal(err)
	}

	log.Println("Connected to database")
}

func GetDB() *sql.DB {
	return db
}

func getConnStr(username, password, host, port, dbName string) string {
	return fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		host, port, username, password, dbName,
	)
}
