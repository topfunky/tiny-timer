package main

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	_ "github.com/mattn/go-sqlite3"
)

var (
	// Global database connection - initialized once and reused
	dbConnection *sql.DB
	dbMutex      sync.Mutex
)

// initDBConnection initializes the global database connection
// This should be called once at application startup
func initDBConnection() error {
	dbMutex.Lock()
	defer dbMutex.Unlock()

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

	// Use DELETE journal mode with FULL synchronous for immediate write visibility
	connStr := dbPath + "?_journal_mode=DELETE&_synchronous=FULL"
	db, err := sql.Open("sqlite3", connStr)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}

	// Set connection pool settings
	db.SetMaxOpenConns(1) // Single connection to avoid timing issues
	db.SetMaxIdleConns(1)

	// Ensure DELETE journal mode
	_, err = db.Exec("PRAGMA journal_mode=DELETE")
	if err != nil {
		db.Close()
		return fmt.Errorf("failed to set journal mode: %w", err)
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
	dbMutex.Lock()
	defer dbMutex.Unlock()
	if dbConnection == nil {
		panic("database connection not initialized - call initDBConnection first")
	}
	return dbConnection
}

// closeDBConnection closes the global database connection
// Should be called on application exit
func closeDBConnection() error {
	dbMutex.Lock()
	defer dbMutex.Unlock()

	if dbConnection == nil {
		return nil
	}

	err := dbConnection.Close()
	dbConnection = nil
	return err
}
