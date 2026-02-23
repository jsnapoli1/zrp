package main

import (
	"database/sql"

	"zrp/internal/database"

	_ "modernc.org/sqlite"
)

var db *sql.DB

func initDB(path string) error {
	var err error
	db, err = database.InitDB(db, path)
	if err != nil {
		return err
	}
	return runMigrations()
}

func runMigrations() error {
	return database.RunMigrations(db, InitSearchTables)
}

func seedDB() {
	database.SeedDB(db)
}

func nextID(prefix string, table string, digits int) string {
	return database.NextID(db, prefix, table, digits)
}

func ns(s *string) sql.NullString {
	return database.NS(s)
}

func sp(n sql.NullString) *string {
	return database.SP(n)
}
