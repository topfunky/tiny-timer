package main

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"
)

var (
	// Global database connection - initialized once and reused
	dbConnection *sql.DB
)

// initDBConnection initializes the global database connection
// This should be called once at application startup
func initDBConnection() error {
	if dbConnection != nil {
		return nil // Already initialized
	}

	dbPath, err := getDBPath()
	if err != nil {
		return fmt.Errorf("failed to get DB path: %w", err)
	}

	dbDir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dbDir, os.ModePerm); err != nil {
		return fmt.Errorf("failed to create DB directory: %w", err)
	}

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}

	// Create sessions table if it doesn't exist
	if err := ensureSessionsTable(db); err != nil {
		db.Close()
		return fmt.Errorf("failed to create sessions table: %w", err)
	}

	dbConnection = db
	return nil
}

// getDB returns the global database connection
// Panics if called before initDBConnection
func getDB() *sql.DB {
	if dbConnection == nil {
		panic("database connection not initialized - call initDBConnection first")
	}
	return dbConnection
}

// closeDBConnection closes the global database connection
// Should be called on application exit
func closeDBConnection() error {
	if dbConnection == nil {
		return nil
	}

	err := dbConnection.Close()
	dbConnection = nil
	return err
}
