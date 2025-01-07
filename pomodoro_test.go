package main

import (
	"database/sql"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFormatDurationAsMMSS(t *testing.T) {
	tests := []struct {
		duration int64
		expected string
	}{
		{0, "00:00"},
		{59, "00:59"},
		{60, "01:00"},
		{61, "01:01"},
		{3600, "60:00"},
	}

	for _, test := range tests {
		result := formatDurationAsMMSS(test.duration)
		assert.Equal(t, test.expected, result, "formatDurationAsMMSS(%d)", test.duration)
	}
}

func TestSaveSessionToDB(t *testing.T) {
	// Set up a temporary database path
	tempDBPath := os.TempDir() + sqlite_db_file_path
	os.Setenv("HOME", os.TempDir())

	// Clean up the temporary database file after the test
	defer os.Remove(tempDBPath)

	// Test data.
	// Assumes that each tuple of (duration, completed) is unique.
	tests := []struct {
		duration  int64
		completed bool
	}{
		{60, true},
		{120, false},
		{180, true},
	}

	for _, test := range tests {
		err := saveSessionToDB(test.duration, test.completed)
		assert.NoError(t, err, "saveSessionToDB(%d, %v)", test.duration, test.completed)

		// Verify the session was saved correctly
		db, err := sql.Open("sqlite3", tempDBPath)
		assert.NoError(t, err)
		defer db.Close()

		var count int
		err = db.QueryRow("SELECT COUNT(*) FROM sessions WHERE duration = ? AND completed = ?", test.duration, test.completed).Scan(&count)
		assert.NoError(t, err)
		assert.Equal(t, 1, count, "Expected one session with duration %d and completed %v", test.duration, test.completed)
	}
}
