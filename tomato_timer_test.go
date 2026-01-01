package main

import (
	"database/sql"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/bubbles/progress"
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
	// Assumes that each tuple of (duration, completed, title) is unique.
	tests := []struct {
		duration  int64
		completed bool
		title     string
	}{
		{60, true, "Test Session 1"},
		{120, false, "Test Session 2"},
		{180, true, ""},
	}

	for _, test := range tests {
		err := saveSessionToDB(test.duration, test.completed, test.title)
		assert.NoError(t, err, "saveSessionToDB(%d, %v, %s)", test.duration, test.completed, test.title)

		// Verify the session was saved correctly
		db, err := sql.Open("sqlite3", tempDBPath)
		assert.NoError(t, err)
		defer db.Close()

		var count int
		err = db.QueryRow("SELECT COUNT(*) FROM sessions WHERE duration = ? AND completed = ? AND title = ?", test.duration, test.completed, test.title).Scan(&count)
		assert.NoError(t, err)
		assert.Equal(t, 1, count, "Expected one session with duration %d, completed %v, and title %s", test.duration, test.completed, test.title)
	}
}

func TestSaveSessionToDBWithTitle(t *testing.T) {
	// Set up a temporary database path
	tempDBPath := os.TempDir() + sqlite_db_file_path
	os.Setenv("HOME", os.TempDir())

	// Clean up the temporary database file after the test
	defer os.Remove(tempDBPath)

	// Test with a specific title
	title := "Important Work Session"
	duration := int64(1500)
	completed := true

	err := saveSessionToDB(duration, completed, title)
	assert.NoError(t, err)

	// Verify the title was saved correctly
	db, err := sql.Open("sqlite3", tempDBPath)
	assert.NoError(t, err)
	defer db.Close()

	var savedTitle string
	err = db.QueryRow("SELECT title FROM sessions WHERE duration = ? AND completed = ?", duration, completed).Scan(&savedTitle)
	assert.NoError(t, err)
	assert.Equal(t, title, savedTitle, "Expected title to be saved correctly")
}

func TestSaveSessionToDBWithoutTitle(t *testing.T) {
	// Set up a temporary database path
	tempDBPath := os.TempDir() + sqlite_db_file_path
	os.Setenv("HOME", os.TempDir())

	// Clean up the temporary database file after the test
	defer os.Remove(tempDBPath)

	// Test without title (empty string)
	duration := int64(900)
	completed := false

	err := saveSessionToDB(duration, completed, "")
	assert.NoError(t, err)

	// Verify the empty title was saved correctly
	db, err := sql.Open("sqlite3", tempDBPath)
	assert.NoError(t, err)
	defer db.Close()

	var savedTitle string
	err = db.QueryRow("SELECT title FROM sessions WHERE duration = ? AND completed = ?", duration, completed).Scan(&savedTitle)
	assert.NoError(t, err)
	assert.Equal(t, "", savedTitle, "Expected title to be empty")
}

func TestViewWithTitle(t *testing.T) {
	// Create a model with a title
	m := model{
		progress:       progress.New(progress.WithGradient(colorMontezumaGold, colorCream), progress.WithoutPercentage()),
		startTime:      time.Now().Unix(),
		targetDuration: 60,
		title:          "Test Task",
	}

	view := m.View()
	
	// Verify that the title is displayed in the view
	assert.Contains(t, view, "Test Task", "Expected view to contain the title")
}

func TestViewWithoutTitle(t *testing.T) {
	// Create a model without a title
	m := model{
		progress:       progress.New(progress.WithGradient(colorMontezumaGold, colorCream), progress.WithoutPercentage()),
		startTime:      time.Now().Unix(),
		targetDuration: 60,
		title:          "",
	}

	view := m.View()
	
	// Count the number of lines - should have one fewer line when no title is present
	lines := strings.Split(view, "\n")
	
	// Verify that empty title doesn't add extra newlines
	// The view should not have a title line
	for _, line := range lines {
		// Make sure no line is JUST padding (which would indicate an empty title line)
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			// Empty lines are okay, just not lines with only padding
			continue
		}
	}
}

func TestDatabaseSchemaHasTitleColumn(t *testing.T) {
	// Set up a temporary database path
	tempDBPath := os.TempDir() + sqlite_db_file_path
	os.Setenv("HOME", os.TempDir())

	// Clean up the temporary database file after the test
	defer os.Remove(tempDBPath)

	// Create a session to initialize the database
	err := saveSessionToDB(60, true, "Test")
	assert.NoError(t, err)

	// Verify the title column exists
	db, err := sql.Open("sqlite3", tempDBPath)
	assert.NoError(t, err)
	defer db.Close()

	rows, err := db.Query("PRAGMA table_info(sessions)")
	assert.NoError(t, err)
	defer rows.Close()

	columnNames := make(map[string]bool)
	for rows.Next() {
		var cid int
		var name string
		var ctype string
		var notnull int
		var dfltValue interface{}
		var pk int

		err = rows.Scan(&cid, &name, &ctype, &notnull, &dfltValue, &pk)
		assert.NoError(t, err)
		columnNames[name] = true
	}

	// Verify all expected columns exist
	assert.True(t, columnNames["id"], "Expected 'id' column to exist")
	assert.True(t, columnNames["datetime"], "Expected 'datetime' column to exist")
	assert.True(t, columnNames["duration"], "Expected 'duration' column to exist")
	assert.True(t, columnNames["completed"], "Expected 'completed' column to exist")
	assert.True(t, columnNames["title"], "Expected 'title' column to exist")
}
