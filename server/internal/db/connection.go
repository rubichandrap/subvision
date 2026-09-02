// Package db opens the SQLite database backing the server's job state.
package db

import (
	"database/sql"

	_ "github.com/mattn/go-sqlite3"
)

// Open opens the SQLite database at dsn (a file path or ":memory:" for tests).
// SQLite allows one writer, so the pool is capped at a single connection.
func Open(dsn string) (*sql.DB, error) {
	handle, err := sql.Open("sqlite3", dsn)
	if err != nil {
		return nil, err
	}
	handle.SetMaxOpenConns(1)
	return handle, nil
}
