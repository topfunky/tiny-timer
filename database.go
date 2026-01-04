package main

import (
	"database/sql"
	"os"
	"path/filepath"
	"time"

	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/lipgloss"
)

// getDBPath returns the full path to the SQLite database file
func getDBPath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(homeDir, ".config", "tomato-timer", "tomato-timer.db"), nil
}

// ensureSessionsTable creates the sessions table if it doesn't exist
// This helper function is used by both initDB() and saveSessionToDB()
func ensureSessionsTable(db *sql.DB) error {
	createTableSQL := `CREATE TABLE IF NOT EXISTS sessions (
		"id" INTEGER PRIMARY KEY AUTOINCREMENT,
		"datetime" DATETIME DEFAULT CURRENT_TIMESTAMP,
		"duration" INTEGER,
		"completed" BOOLEAN,
		"title" TEXT
	);`

	_, err := db.Exec(createTableSQL)
	return err
}

// initDB initializes the database and creates the sessions table if it doesn't exist
// This is now a wrapper that calls initDBConnection for backward compatibility
func initDB() error {
	return initDBConnection()
}

// Save a record to SQLite DB that represents a working session as counted by the timer
func saveSessionToDB(duration int64, completed bool, title string) error {
	db := getDB()
	debugLog("saveSessionToDB: duration=%d, completed=%v, title=%q", duration, completed, title)

	// Use explicit transaction with commit to ensure write is fully committed
	tx, err := db.Begin()
	if err != nil {
		debugLog("saveSessionToDB: Begin transaction failed: %v", err)
		return err
	}

	insertSessionSQL := `INSERT INTO sessions (duration, completed, title) VALUES (?, ?, ?)`
	result, err := tx.Exec(insertSessionSQL, duration, completed, title)
	if err != nil {
		debugLog("saveSessionToDB: Exec failed: %v", err)
		tx.Rollback()
		return err
	}

	rowsAffected, _ := result.RowsAffected()
	debugLog("saveSessionToDB: Inserted %d row(s)", rowsAffected)

	// Commit the transaction explicitly
	err = tx.Commit()
	if err != nil {
		debugLog("saveSessionToDB: Commit failed: %v", err)
		return err
	}

	debugLog("saveSessionToDB: Successfully saved session")
	return nil
}

// Fetch recent sessions from the database
func getRecentSessions(limit int) ([]session, error) {
	db := getDB()
	debugLog("getRecentSessions: limit=%d", limit)

	query := `SELECT id, datetime, duration, completed, title FROM sessions ORDER BY datetime DESC, id DESC LIMIT ?`
	rows, err := db.Query(query, limit)
	if err != nil {
		debugLog("getRecentSessions: Query failed: %v", err)
		return nil, err
	}
	defer rows.Close()

	var sessions []session
	count := 0
	for rows.Next() {
		var s session
		err := rows.Scan(&s.id, &s.datetime, &s.duration, &s.completed, &s.title)
		if err != nil {
			debugLog("getRecentSessions: Scan failed: %v", err)
			return nil, err
		}
		sessions = append(sessions, s)
		count++
	}

	debugLog("getRecentSessions: Retrieved %d session(s)", count)
	return sessions, nil
}

// buildTableView creates a table model from recent sessions
func buildTableView(limit int) (table.Model, error) {
	sessions, err := getRecentSessions(limit)
	if err != nil {
		return table.Model{}, err
	}

	columns := []table.Column{
		{Title: "Title", Width: 40},
		{Title: "Duration", Width: 10},
		{Title: "Date", Width: 20},
	}

	rows := []table.Row{}
	for _, s := range sessions {
		title := s.title
		if title == "" {
			title = "(no title)"
		}
		duration := formatDurationAsMMSS(s.duration)
		datetime := s.datetime
		if t, err := time.Parse("2006-01-02T15:04:05Z", s.datetime); err == nil {
			datetime = t.Format("Monday, 2 Jan 06")
		}
		rows = append(rows, table.Row{title, duration, datetime})
	}

	t := table.New(
		table.WithColumns(columns),
		table.WithRows(rows),
		table.WithFocused(false),
		table.WithHeight(len(rows)),
	)

	s := table.DefaultStyles()
	s.Header = s.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color(colorGrey)).
		BorderBottom(true).
		Bold(false).
		Foreground(lipgloss.Color(colorGrey)).
		Padding(0, 0)
	s.Cell = s.Cell.
		Padding(0, 0)
	s.Selected = s.Selected.
		Foreground(lipgloss.Color(colorCream)).
		Background(lipgloss.Color(colorGrey)).
		Bold(false)
	t.SetStyles(s)

	return t, nil
}
