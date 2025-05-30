package db

import (
	"database/sql"
	"path/filepath"

	_ "github.com/mattn/go-sqlite3"
)

func Connect() (*sql.DB, error) {
	dbPath := filepath.Join("data", "subvision.db")
	return sql.Open("sqlite3", dbPath)
}
