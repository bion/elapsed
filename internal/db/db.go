package db

import (
	"database/sql"
	_ "github.com/mattn/go-sqlite3"
	"log"
	"os"
)

var Db *sql.DB

func Teardown() {
	Db.Close()
	log.Print("closed db connection")
}

func Init() {
	dbPath, exists := os.LookupEnv("DB_FILE_PATH")
	if !exists {
		dbPath = "./elapsed.db"
	}

	var err error
	Db, err = sql.Open("sqlite3", dbPath)

	if err != nil {
		log.Fatal(err)
	}

	Db.Exec("PRAGMA foreign_keys = ON;")
	log.Printf("db connection opened with file %s", dbPath)
}
