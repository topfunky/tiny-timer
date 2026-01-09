package main

import (
	"database/sql"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/charmbracelet/bubbles/progress"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
)

func TestInitDB(t *testing.T) {
	// Set up a temporary database path
	tempDBPath, cleanup := setupTestDB(t)
	defer cleanup()

	// Initialize the database
	err := initDB()
	assert.NoError(t, err, "initDB should not return an error")

	// Verify the database file was created
	_, err = os.Stat(tempDBPath)
	assert.NoError(t, err, "Database file should exist after initDB")

	// Verify the sessions table was created
	db, err := sql.Open("sqlite", tempDBPath)
	assert.NoError(t, err)
	defer db.Close()

	// Query the table to ensure it exists
	var tableName string
	err = db.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name='sessions'").Scan(&tableName)
	assert.NoError(t, err, "sessions table should exist")
	assert.Equal(t, "sessions", tableName, "Table name should be 'sessions'")
}

func TestInitDBIdempotent(t *testing.T) {
	// Set up a temporary database path
	_, cleanup := setupTestDB(t)
	defer cleanup()

	// Initialize the database twice
	err := initDB()
	assert.NoError(t, err, "First initDB should not return an error")

	err = initDB()
	assert.NoError(t, err, "Second initDB should not return an error (idempotent)")
}

func TestGetRecentSessionsOnEmptyDB(t *testing.T) {
	// Set up a temporary database path
	_, cleanup := setupTestDB(t)
	defer cleanup()

	// Initialize the database
	err := initDB()
	assert.NoError(t, err)

	// Fetch recent sessions from empty database
	sessions, err := getRecentSessions(10)
	assert.NoError(t, err, "getRecentSessions should not error on empty database")
	assert.Empty(t, sessions, "Expected no sessions in empty database")
}

func TestSaveSessionToDB(t *testing.T) {
	// Set up a temporary database path
	tempDBPath, cleanup := setupTestDB(t)
	defer cleanup()

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
		db, err := sql.Open("sqlite", tempDBPath)
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
	tempDBPath, cleanup := setupTestDB(t)
	defer cleanup()

	// Test with a specific title
	title := "Important Work Session"
	duration := int64(1500)
	completed := true

	err := saveSessionToDB(duration, completed, title)
	assert.NoError(t, err)

	// Verify the title was saved correctly
	db, err := sql.Open("sqlite", tempDBPath)
	assert.NoError(t, err)
	defer db.Close()

	var savedTitle string
	err = db.QueryRow("SELECT title FROM sessions WHERE duration = ? AND completed = ?", duration, completed).Scan(&savedTitle)
	assert.NoError(t, err)
	assert.Equal(t, title, savedTitle, "Expected title to be saved correctly")
}

func TestSaveSessionToDBWithoutTitle(t *testing.T) {
	// Set up a temporary database path
	tempDBPath, cleanup := setupTestDB(t)
	defer cleanup()

	// Test without title (empty string)
	duration := int64(900)
	completed := false

	err := saveSessionToDB(duration, completed, "")
	assert.NoError(t, err)

	// Verify the empty title was saved correctly
	db, err := sql.Open("sqlite", tempDBPath)
	assert.NoError(t, err)
	defer db.Close()

	var savedTitle string
	err = db.QueryRow("SELECT title FROM sessions WHERE duration = ? AND completed = ?", duration, completed).Scan(&savedTitle)
	assert.NoError(t, err)
	assert.Equal(t, "", savedTitle, "Expected title to be empty")
}

func TestDatabaseSchemaHasTitleColumn(t *testing.T) {
	// Set up a temporary database path
	tempDBPath, cleanup := setupTestDB(t)
	defer cleanup()

	// Create a session to initialize the database
	err := saveSessionToDB(60, true, "Test")
	assert.NoError(t, err)

	// Verify the title column exists
	db, err := sql.Open("sqlite", tempDBPath)
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

func TestGetRecentSessions(t *testing.T) {
	// Set up a temporary database path
	_, cleanup := setupTestDB(t)
	defer cleanup()

	// Create several test sessions
	sessions := []struct {
		duration  int64
		completed bool
		title     string
	}{
		{1500, true, "Task 1"},
		{900, false, "Task 2"},
		{1800, true, "Task 3"},
		{600, true, "Task 4"},
	}

	for _, s := range sessions {
		err := saveSessionToDB(s.duration, s.completed, s.title)
		assert.NoError(t, err)
	}

	// Fetch recent sessions
	recentSessions, err := getRecentSessions(10)
	assert.NoError(t, err)
	assert.Equal(t, len(sessions), len(recentSessions), "Expected to retrieve all sessions")

	// Verify that all session titles are present
	titles := make(map[string]bool)
	for _, s := range recentSessions {
		titles[s.title] = true
	}
	for _, s := range sessions {
		assert.True(t, titles[s.title], "Expected session with title '%s' to be in results", s.title)
	}
}

func TestGetRecentSessionsWithLimit(t *testing.T) {
	// Set up a temporary database path
	_, cleanup := setupTestDB(t)
	defer cleanup()

	// Create several test sessions
	for i := 0; i < 5; i++ {
		err := saveSessionToDB(1500, true, fmt.Sprintf("Task %d", i))
		assert.NoError(t, err)
		time.Sleep(10 * time.Millisecond)
	}

	// Fetch only 3 most recent sessions
	recentSessions, err := getRecentSessions(3)
	assert.NoError(t, err)
	assert.Equal(t, 3, len(recentSessions), "Expected to retrieve only 3 sessions")
}

func TestDateFormatInTableFromDatabase(t *testing.T) {
	// Set up a temporary database path
	_, cleanup := setupTestDB(t)
	defer cleanup()

	// Save a session to the database
	err := saveSessionToDB(1500, true, "Test Task")
	assert.NoError(t, err)

	// Retrieve the session
	sessions, err := getRecentSessions(1)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(sessions), "Expected one session")

	// Verify the datetime can be parsed and formatted correctly
	s := sessions[0]
	// SQLite DATETIME can store dates in different formats, try both
	var parsedTime time.Time
	if parsedTime, err = time.Parse("2006-01-02 15:04:05", s.datetime); err != nil {
		parsedTime, err = time.Parse("2006-01-02T15:04:05Z", s.datetime)
	}
	assert.NoError(t, err, "Should be able to parse datetime from database: %s", s.datetime)

	// Format it the way the table displays it
	formatted := parsedTime.Format("Monday, 2 Jan 06")

	// Verify the format matches expected pattern (e.g., "Wednesday, 7 Jan 26")
	assert.Regexp(t, `^\w+, \d{1,2} \w+ \d{2}$`, formatted, "Date should match pattern 'DayName, D Mon YY'")
}

func TestFirstSaveThenImmediateHistoryRead(t *testing.T) {
	_, cleanup := setupTestDB(t)
	defer cleanup()

	// Initialize database (creates new empty DB)
	err := initDB()
	assert.NoError(t, err)

	startTime := time.Now().Unix() - 120
	m := model{
		progress:       progress.New(progress.WithGradient(colorMontezumaGold, colorCream), progress.WithoutPercentage()),
		startTime:      startTime,
		targetDuration: 3600,
		countUpMode:    true,
		mode:           timerView,
		title:          "First Task",
	}

	// Press 'd' to mark done and activate prompt
	keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}}
	newModel, _ := m.Update(keyMsg)
	modelTyped := newModel.(model)

	assert.True(t, modelTyped.promptActive, "Expected prompt to be active")
	assert.Equal(t, promptLogAndReset, modelTyped.promptType, "Expected promptType to be promptLogAndReset")

	// Complete the prompt (this saves to DB and refreshes table)
	newModel, _ = modelTyped.Update(promptMsg{title: "First Task", logDB: true})
	modelTyped = newModel.(model)

	// Immediately press 'h' to show history (this is the bug scenario)
	keyMsg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'h'}}
	newModel, _ = modelTyped.Update(keyMsg)
	modelTyped = newModel.(model)

	// Verify we're in table view
	assert.Equal(t, tableView, modelTyped.mode, "Expected to switch to table view")

	// Verify the table has rows (this would fail with the bug)
	tableRows := modelTyped.table.Rows()
	assert.NotEmpty(t, tableRows, "Expected table to have rows after saving first task")
	assert.GreaterOrEqual(t, len(tableRows), 1, "Expected at least one row in history table")

	// Verify the saved task appears in the table
	found := false
	for _, row := range tableRows {
		if len(row) > 0 && row[0] == "First Task" {
			found = true
			break
		}
	}
	assert.True(t, found, "Expected 'First Task' to appear in history table")
}
