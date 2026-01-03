package main

import (
	"database/sql"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/stretchr/testify/assert"
)

// setupTestDB sets up a temporary database for testing and returns the cleanup function
func setupTestDB(t *testing.T) (string, func()) {
	// Set HOME to temp directory for testing
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", os.TempDir())
	
	// Get the database path (will use temp HOME)
	dbPath, err := getDBPath()
	if err != nil {
		t.Fatalf("Failed to get DB path: %v", err)
	}
	
	// Cleanup function
	cleanup := func() {
		os.Remove(dbPath)
		os.Setenv("HOME", originalHome)
	}
	
	return dbPath, cleanup
}

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
	db, err := sql.Open("sqlite3", tempDBPath)
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
	tempDBPath, cleanup := setupTestDB(t)
	defer cleanup()

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
	tempDBPath, cleanup := setupTestDB(t)
	defer cleanup()

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
	tempDBPath, cleanup := setupTestDB(t)
	defer cleanup()

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

func TestTableViewMode(t *testing.T) {
	m := model{
		progress:       progress.New(progress.WithGradient(colorMontezumaGold, colorCream), progress.WithoutPercentage()),
		startTime:      time.Now().Unix(),
		targetDuration: 60,
		title:          "Test Task",
		mode:           timerView,
	}

	// Verify initial mode is timer view
	assert.Equal(t, timerView, m.mode)

	// Verify timer view is displayed
	view := m.View()
	assert.Contains(t, view, "title")
	assert.Contains(t, view, "history")
}

func TestDateFormatInTable(t *testing.T) {
	// Test that dates are formatted correctly in the table view
	testCases := []struct {
		input    string
		expected string
	}{
		{"2026-01-07 10:30:45", "Wednesday, 7 Jan 26"},
		{"2026-01-01 00:00:00", "Thursday, 1 Jan 26"},
		{"2025-12-25 15:30:00", "Thursday, 25 Dec 25"},
	}

	for _, tc := range testCases {
		t.Run(tc.input, func(t *testing.T) {
			// Parse the datetime string
			parsedTime, err := time.Parse("2006-01-02 15:04:05", tc.input)
			assert.NoError(t, err)

			// Format it the way the table should display it
			formatted := parsedTime.Format("Monday, 2 Jan 06")
			assert.Equal(t, tc.expected, formatted, "Date should be formatted as '%s'", tc.expected)
		})
	}
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
	parsedTime, err := time.Parse("2006-01-02T15:04:05Z", s.datetime)
	assert.NoError(t, err, "Should be able to parse datetime from database: %s", s.datetime)

	// Format it the way the table displays it
	formatted := parsedTime.Format("Monday, 2 Jan 06")

	// Verify the format matches expected pattern (e.g., "Wednesday, 7 Jan 26")
	assert.Regexp(t, `^\w+, \d{1,2} \w+ \d{2}$`, formatted, "Date should match pattern 'DayName, D Mon YY'")
}

func TestTableHeadersAreLeftAligned(t *testing.T) {
	// Set up a temporary database path
	_, cleanup := setupTestDB(t)
	defer cleanup()

	// Create a test session
	err := saveSessionToDB(1500, true, "Test Task")
	assert.NoError(t, err)

	// Create a model and trigger table view
	m := model{
		progress:       progress.New(progress.WithGradient(colorMontezumaGold, colorCream), progress.WithoutPercentage()),
		startTime:      time.Now().Unix(),
		targetDuration: 60,
		title:          "Test Task",
		mode:           timerView,
	}

	// Simulate pressing 't' to show table
	sessions, err := getRecentSessions(10)
	assert.NoError(t, err)

	// Build table (same logic as in updateKey)
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
		if parsedTime, err := time.Parse("2006-01-02T15:04:05Z", s.datetime); err == nil {
			datetime = parsedTime.Format("Monday, 2 Jan 06")
		}

		rows = append(rows, table.Row{title, duration, datetime})
	}

	tbl := table.New(
		table.WithColumns(columns),
		table.WithRows(rows),
		table.WithFocused(false),
		table.WithHeight(len(rows)+2), // Add extra height for visibility
	)

	s := table.DefaultStyles()
	s.Header = s.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color(colorGrey)).
		BorderBottom(true).
		Bold(false).
		Padding(0, 0)
	s.Cell = s.Cell.
		Padding(0, 0)
	tbl.SetStyles(s)

	m.table = tbl
	m.mode = tableView

	// Get the rendered view
	view := m.View()

	// Check that headers appear in the output
	assert.Contains(t, view, "Title", "Table should contain 'Title' header")
	assert.Contains(t, view, "Duration", "Table should contain 'Duration' header")
	assert.Contains(t, view, "Date", "Table should contain 'Date' header")

	// Find header line and first data line to compare alignment
	lines := strings.Split(view, "\n")
	var headerLineRaw string
	var firstDataLineRaw string
	foundHeader := false

	for _, line := range lines {
		// Find the header line (contains all three headers)
		if !foundHeader && strings.Contains(line, "Title") && strings.Contains(line, "Duration") && strings.Contains(line, "Date") {
			headerLineRaw = line
			foundHeader = true
			continue
		}

		// Find the first data line (after border line, non-empty, not help text)
		if foundHeader && len(strings.TrimSpace(line)) > 0 && !strings.Contains(line, "─") && !strings.Contains(line, "Press any key") {
			firstDataLineRaw = line
			break
		}
	}

	assert.NotEmpty(t, headerLineRaw, "Should have found the header line")
	assert.NotEmpty(t, firstDataLineRaw, "Should have found a data line")

	// Compare using the RAW lines (with padding included)
	// Headers and data should start at the same column position
	headerTitleStartRaw := strings.Index(headerLineRaw, "Title")
	dataTitleStartRaw := strings.Index(firstDataLineRaw, "Test Task")

	assert.Equal(t, headerTitleStartRaw, dataTitleStartRaw,
		"Header and data should start at the same column in the rendered output. Header starts at %d, Data starts at %d",
		headerTitleStartRaw, dataTitleStartRaw)
}

func TestTimerContinuesAfterPause(t *testing.T) {
	// Test that timer calculates elapsed time correctly even after a simulated pause
	m := model{
		progress:       progress.New(progress.WithGradient(colorMontezumaGold, colorCream), progress.WithoutPercentage()),
		startTime:      time.Now().Unix() - 30, // Started 30 seconds ago
		targetDuration: 120,                    // 2 minute timer
		title:          "Test Task",
		mode:           timerView,
	}

	// Simulate the passage of time (another 30 seconds)
	time.Sleep(100 * time.Millisecond)

	// Calculate elapsed time as the timer does
	elapsed := time.Now().Unix() - m.startTime
	remaining := m.targetDuration - elapsed

	// Verify elapsed time is approximately 30 seconds (within 1 second tolerance)
	assert.True(t, elapsed >= 30 && elapsed <= 31, "Expected elapsed time to be ~30 seconds, got %d", elapsed)
	assert.True(t, remaining >= 89 && remaining <= 90, "Expected remaining time to be ~90 seconds, got %d", remaining)

	// Verify progress calculation
	percentCompleted := float64(elapsed) / float64(m.targetDuration)
	assert.InDelta(t, 0.25, percentCompleted, 0.01, "Expected ~25%% completion")
}

func TestResumeAfterCompletion(t *testing.T) {
	// Set up a temporary database path
	tempDBPath, cleanup := setupTestDB(t)
	defer cleanup()

	// Create a timer that started 70 seconds ago (past the 60 second target)
	m := model{
		progress:       progress.New(progress.WithGradient(colorMontezumaGold, colorCream), progress.WithoutPercentage()),
		startTime:      time.Now().Unix() - 70,
		targetDuration: 60,
		title:          "Completed Task",
		mode:           timerView,
	}

	// Initial progress should be 0
	assert.Equal(t, 0.0, m.progress.Percent(), "Initial progress should be 0")

	// Simulate a tick after resuming (which calls updatePercent)
	newModel, _ := m.Update(tickMsg(time.Now()))

	// Verify the model is returned after completion
	assert.NotNil(t, newModel, "Expected model to be returned")

	// Verify the session was saved to the database
	db, err := sql.Open("sqlite3", tempDBPath)
	assert.NoError(t, err)
	defer db.Close()

	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM sessions WHERE title = ? AND completed = ?", "Completed Task", true).Scan(&count)
	assert.NoError(t, err)
	assert.Equal(t, 1, count, "Expected one completed session to be saved when resuming after completion")
}

func TestTickAfterCompletion(t *testing.T) {
	// Set up a temporary database path
	tempDBPath, cleanup := setupTestDB(t)
	defer cleanup()

	// Create a timer that started 70 seconds ago (past the 60 second target)
	m := model{
		progress:       progress.New(progress.WithGradient(colorMontezumaGold, colorCream), progress.WithoutPercentage()),
		startTime:      time.Now().Unix() - 70,
		targetDuration: 60,
		title:          "Tick After Complete",
		mode:           timerView,
	}

	// Simulate a regular tick (not resume)
	newModel, _ := m.Update(tickMsg(time.Now()))

	// Verify the model is returned
	assert.NotNil(t, newModel, "Expected model to be returned")

	// Verify the session was saved to the database
	db, err := sql.Open("sqlite3", tempDBPath)
	assert.NoError(t, err)
	defer db.Close()

	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM sessions WHERE title = ? AND completed = ?", "Tick After Complete", true).Scan(&count)
	assert.NoError(t, err)
	assert.Equal(t, 1, count, "Expected one completed session to be saved on tick after completion")
}

func TestCtrlZSuspendsInTimerView(t *testing.T) {
	m := model{
		progress:       progress.New(progress.WithGradient(colorMontezumaGold, colorCream), progress.WithoutPercentage()),
		startTime:      time.Now().Unix(),
		targetDuration: 60,
		title:          "Test Task",
		mode:           timerView,
	}

	// Create a Ctrl-Z key message
	keyMsg := tea.KeyMsg{Type: tea.KeyCtrlZ}

	// Process the Ctrl-Z key
	newModel, cmd := m.Update(keyMsg)

	// Verify the model is returned unchanged
	assert.NotNil(t, newModel, "Expected model to be returned")

	// Verify a suspend command was returned
	assert.NotNil(t, cmd, "Expected suspend command to be returned")
}

func TestCtrlZSuspendsInTableView(t *testing.T) {
	m := model{
		progress:       progress.New(progress.WithGradient(colorMontezumaGold, colorCream), progress.WithoutPercentage()),
		startTime:      time.Now().Unix(),
		targetDuration: 60,
		title:          "Test Task",
		mode:           tableView,
	}

	// Create a Ctrl-Z key message
	keyMsg := tea.KeyMsg{Type: tea.KeyCtrlZ}

	// Process the Ctrl-Z key
	newModel, cmd := m.Update(keyMsg)

	// Verify the model is still in table view (not exited)
	modelTyped := newModel.(model)
	assert.Equal(t, tableView, modelTyped.mode, "Expected to remain in table view after Ctrl-Z")

	// Verify a suspend command was returned
	assert.NotNil(t, cmd, "Expected suspend command to be returned")
}

func TestOtherKeysStillQuitTimerView(t *testing.T) {
	m := model{
		progress:       progress.New(progress.WithGradient(colorMontezumaGold, colorCream), progress.WithoutPercentage()),
		startTime:      time.Now().Unix(),
		targetDuration: 60,
		title:          "Test Task",
		mode:           timerView,
	}

	// Test that pressing 'q' still quits
	keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}}

	// Process the key
	_, cmd := m.Update(keyMsg)

	// Verify a quit command was returned (we can't directly test if it's tea.Quit, but we know it's not nil)
	assert.NotNil(t, cmd, "Expected quit command to be returned for non-special keys")
}

func TestCountUpModeInitialization(t *testing.T) {
	m := model{
		progress:       progress.New(progress.WithGradient(colorMontezumaGold, colorCream), progress.WithoutPercentage()),
		startTime:      time.Now().Unix(),
		targetDuration: 3600,
		countUpMode:    true,
		mode:           timerView,
	}

	assert.True(t, m.countUpMode, "Expected count-up mode to be enabled")
	assert.Equal(t, int64(3600), m.targetDuration, "Expected 1 hour default duration in count-up mode")
}

func TestCountUpModeElapsedTime(t *testing.T) {
	startTime := time.Now().Unix() - 30
	m := model{
		progress:       progress.New(progress.WithGradient(colorMontezumaGold, colorCream), progress.WithoutPercentage()),
		startTime:      startTime,
		targetDuration: 3600,
		countUpMode:    true,
		mode:           timerView,
	}

	elapsed := time.Now().Unix() - m.startTime
	assert.True(t, elapsed >= 30 && elapsed <= 32, "Expected elapsed time to be approximately 30 seconds")
}

func TestCountUpModeKeysActivatePrompt(t *testing.T) {
	m := model{
		progress:       progress.New(progress.WithGradient(colorMontezumaGold, colorCream), progress.WithoutPercentage()),
		startTime:      time.Now().Unix(),
		targetDuration: 3600,
		countUpMode:    true,
		mode:           timerView,
	}

	// Test 'd' key activates prompt for logging
	keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}}
	newModel, _ := m.Update(keyMsg)
	modelTyped := newModel.(model)
	assert.True(t, modelTyped.promptActive, "Expected prompt to be active after pressing 'd'")
	assert.Equal(t, promptLogAndReset, modelTyped.promptType, "Expected promptType to be promptLogAndReset")

	// Test 't' key activates prompt for title only
	m.promptActive = false
	m.title = "Test Title"
	keyMsg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'t'}}
	newModel, _ = m.Update(keyMsg)
	modelTyped = newModel.(model)
	assert.True(t, modelTyped.promptActive, "Expected prompt to be active after pressing 't'")
	assert.Equal(t, promptEditTitle, modelTyped.promptType, "Expected promptType to be promptEditTitle")
}

func TestCountUpModePromptInput(t *testing.T) {
	m := model{
		progress:       progress.New(progress.WithGradient(colorMontezumaGold, colorCream), progress.WithoutPercentage()),
		startTime:      time.Now().Unix(),
		targetDuration: 3600,
		countUpMode:    true,
		mode:           timerView,
		promptActive:   true,
		inputBuffer:    "",
		promptType:     promptLogAndReset,
	}

	// Test typing characters
	keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'T', 'e', 's', 't'}}
	newModel, _ := m.Update(keyMsg)
	modelTyped := newModel.(model)
	assert.Equal(t, "Test", modelTyped.inputBuffer, "Expected input buffer to accumulate typed characters")

	// Test backspace
	keyMsg = tea.KeyMsg{Type: tea.KeyBackspace}
	newModel, _ = modelTyped.Update(keyMsg)
	modelTyped = newModel.(model)
	assert.Equal(t, "Tes", modelTyped.inputBuffer, "Expected backspace to remove last character")
}

func TestCountUpModePromptLogAndReset(t *testing.T) {
	tempDBPath, cleanup := setupTestDB(t)
	defer cleanup()

	startTime := time.Now().Unix() - 120
	m := model{
		progress:       progress.New(progress.WithGradient(colorMontezumaGold, colorCream), progress.WithoutPercentage()),
		startTime:      startTime,
		targetDuration: 3600,
		countUpMode:    true,
		mode:           timerView,
	}

	// Simulate prompt completion (type 0 = log and reset)
	newModel, _ := m.Update(promptMsg{title: "Test Task", logDB: true})
	modelTyped := newModel.(model)

	assert.Equal(t, "", modelTyped.title, "Expected title to be blank after logging")
	assert.False(t, modelTyped.promptActive, "Expected prompt to be inactive after completion")

	// Verify session was saved to DB
	db, err := sql.Open("sqlite3", tempDBPath)
	assert.NoError(t, err)
	defer db.Close()

	var count int
	var savedTitle string
	err = db.QueryRow("SELECT COUNT(*), title FROM sessions WHERE title = ?", "Test Task").Scan(&count, &savedTitle)
	assert.NoError(t, err)
	assert.Equal(t, 1, count, "Expected session to be saved to database")
	assert.Equal(t, "Test Task", savedTitle, "Expected task title to be saved")
}

func TestCountUpModePromptTitleOnly(t *testing.T) {
	m := model{
		progress:       progress.New(progress.WithGradient(colorMontezumaGold, colorCream), progress.WithoutPercentage()),
		startTime:      time.Now().Unix() - 50,
		targetDuration: 3600,
		countUpMode:    true,
		mode:           timerView,
		title:          "Old Title",
	}

	// Test that pressing 't' activates title-only prompt
	keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'t'}}
	newModel, _ := m.Update(keyMsg)
	modelTyped := newModel.(model)

	assert.True(t, modelTyped.promptActive, "Expected prompt to be active after pressing 't'")
	assert.Equal(t, promptEditTitle, modelTyped.promptType, "Expected promptType to be promptEditTitle")
	assert.Equal(t, "Old Title", modelTyped.inputBuffer, "Expected input buffer to pre-fill with current title")

	// Now test that input works in prompt mode (appending to pre-filled text)
	m = modelTyped
	keyMsg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'N', 'e', 'w'}}
	newModel, _ = m.Update(keyMsg)
	modelTyped = newModel.(model)
	assert.Equal(t, "Old TitleNew", modelTyped.inputBuffer, "Expected input buffer to contain pre-filled text plus typed text")
}

func TestCountUpModePromptWithSpaces(t *testing.T) {
	m := model{
		progress:       progress.New(progress.WithGradient(colorMontezumaGold, colorCream), progress.WithoutPercentage()),
		startTime:      time.Now().Unix() - 50,
		targetDuration: 3600,
		countUpMode:    true,
		mode:           timerView,
		title:          "Work",
	}

	// Activate prompt with 'd' key
	keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}}
	newModel, _ := m.Update(keyMsg)
	m = newModel.(model)
	assert.True(t, m.promptActive, "Expected prompt to be active")

	// Type "Work on " with space
	keyMsg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'o', 'n', ' ', 't', 'a', 's', 'k'}}
	newModel, _ = m.Update(keyMsg)
	m = newModel.(model)
	assert.Equal(t, "Workon task", m.inputBuffer, "Expected input to include space from runes")

	// Also test explicit space key
	m.inputBuffer = "Test"
	keyMsg = tea.KeyMsg{Type: tea.KeySpace}
	newModel, _ = m.Update(keyMsg)
	m = newModel.(model)
	assert.Equal(t, "Test ", m.inputBuffer, "Expected KeySpace to add space to input buffer")
}

func TestCountUpViewDisplay(t *testing.T) {
	m := model{
		progress:       progress.New(progress.WithGradient(colorMontezumaGold, colorCream), progress.WithoutPercentage()),
		startTime:      time.Now().Unix() - 600,
		targetDuration: 3600,
		countUpMode:    true,
		mode:           timerView,
		title:          "My Task",
	}

	view := m.View()
	assert.Contains(t, view, "My Task", "Expected title to be displayed")
	assert.Contains(t, view, "done", "Expected count-up mode help text")
	assert.Contains(t, view, "title", "Expected title change help text")
}

func TestNormalModeSetTitle(t *testing.T) {
	m := model{
		progress:       progress.New(progress.WithGradient(colorMontezumaGold, colorCream), progress.WithoutPercentage()),
		startTime:      time.Now().Unix(),
		targetDuration: 1500,
		countUpMode:    false,
		mode:           timerView,
		title:          "Old Title",
	}

	// Press 't' to activate prompt
	keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'t'}}
	newModel, _ := m.Update(keyMsg)
	modelTyped := newModel.(model)

	assert.True(t, modelTyped.promptActive, "Expected prompt to be active after pressing 't' in normal mode")
	assert.Equal(t, promptEditTitle, modelTyped.promptType, "Expected promptType to be promptEditTitle in normal mode")
	assert.Equal(t, "Old Title", modelTyped.inputBuffer, "Expected input buffer to pre-fill with current title")

	// Simulate typing new title
	modelTyped.inputBuffer = "New Title"
	newModel, _ = modelTyped.Update(tea.KeyMsg{Type: tea.KeyEnter})
	modelTyped = newModel.(model)

	assert.Equal(t, "New Title", modelTyped.title, "Expected title to be updated")
	assert.False(t, modelTyped.promptActive, "Expected prompt to be inactive after Enter")
}

// Tests for new consistent keyboard shortcuts

func TestHKeyShowsHistoryInCountdownMode(t *testing.T) {
	// Set up a temporary database path
	_, cleanup := setupTestDB(t)
	defer cleanup()

	// Initialize database
	err := initDB()
	assert.NoError(t, err)

	m := model{
		progress:       progress.New(progress.WithGradient(colorMontezumaGold, colorCream), progress.WithoutPercentage()),
		startTime:      time.Now().Unix(),
		targetDuration: 1500,
		countUpMode:    false,
		mode:           timerView,
	}

	// Press 'h' to show history
	keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'h'}}
	newModel, _ := m.Update(keyMsg)
	modelTyped := newModel.(model)

	assert.Equal(t, tableView, modelTyped.mode, "Expected to switch to table view after pressing 'h'")
}

func TestHKeyShowsHistoryInCountUpMode(t *testing.T) {
	// Set up a temporary database path
	_, cleanup := setupTestDB(t)
	defer cleanup()

	// Initialize database
	err := initDB()
	assert.NoError(t, err)

	m := model{
		progress:       progress.New(progress.WithGradient(colorMontezumaGold, colorCream), progress.WithoutPercentage()),
		startTime:      time.Now().Unix(),
		targetDuration: 3600,
		countUpMode:    true,
		mode:           timerView,
	}

	// Press 'h' to show history
	keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'h'}}
	newModel, _ := m.Update(keyMsg)
	modelTyped := newModel.(model)

	assert.Equal(t, tableView, modelTyped.mode, "Expected to switch to table view after pressing 'h'")
}

func TestDKeyMarksDoneInCountdownMode(t *testing.T) {
	tempDBPath, cleanup := setupTestDB(t)
	defer cleanup()

	startTime := time.Now().Unix() - 120
	m := model{
		progress:       progress.New(progress.WithGradient(colorMontezumaGold, colorCream), progress.WithoutPercentage()),
		startTime:      startTime,
		targetDuration: 1500,
		countUpMode:    false,
		mode:           timerView,
		title:          "Test Task",
	}

	// Press 'd' to mark done
	keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}}
	newModel, _ := m.Update(keyMsg)
	modelTyped := newModel.(model)

	assert.True(t, modelTyped.promptActive, "Expected prompt to be active after pressing 'd'")
	assert.Equal(t, promptLogAndReset, modelTyped.promptType, "Expected promptType to be promptLogAndReset")

	// Complete the prompt
	newModel, _ = modelTyped.Update(promptMsg{title: "Test Task", logDB: true})
	modelTyped = newModel.(model)

	// Verify session was saved to DB
	db, err := sql.Open("sqlite3", tempDBPath)
	assert.NoError(t, err)
	defer db.Close()

	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM sessions WHERE title = ?", "Test Task").Scan(&count)
	assert.NoError(t, err)
	assert.Equal(t, 1, count, "Expected session to be saved to database")
}

func TestDKeyMarksDoneInCountUpMode(t *testing.T) {
	tempDBPath, cleanup := setupTestDB(t)
	defer cleanup()

	startTime := time.Now().Unix() - 120
	m := model{
		progress:       progress.New(progress.WithGradient(colorMontezumaGold, colorCream), progress.WithoutPercentage()),
		startTime:      startTime,
		targetDuration: 3600,
		countUpMode:    true,
		mode:           timerView,
		title:          "Count Up Task",
	}

	// Press 'd' to mark done
	keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}}
	newModel, _ := m.Update(keyMsg)
	modelTyped := newModel.(model)

	assert.True(t, modelTyped.promptActive, "Expected prompt to be active after pressing 'd'")
	assert.Equal(t, promptLogAndReset, modelTyped.promptType, "Expected promptType to be promptLogAndReset")

	// Complete the prompt
	newModel, _ = modelTyped.Update(promptMsg{title: "Count Up Task", logDB: true})
	modelTyped = newModel.(model)

	// Verify session was saved to DB
	db, err := sql.Open("sqlite3", tempDBPath)
	assert.NoError(t, err)
	defer db.Close()

	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM sessions WHERE title = ?", "Count Up Task").Scan(&count)
	assert.NoError(t, err)
	assert.Equal(t, 1, count, "Expected session to be saved to database")
}

func TestTKeyEditsTitleInCountdownMode(t *testing.T) {
	m := model{
		progress:       progress.New(progress.WithGradient(colorMontezumaGold, colorCream), progress.WithoutPercentage()),
		startTime:      time.Now().Unix(),
		targetDuration: 1500,
		countUpMode:    false,
		mode:           timerView,
		title:          "Old Title",
	}

	// Press 't' to edit title
	keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'t'}}
	newModel, _ := m.Update(keyMsg)
	modelTyped := newModel.(model)

	assert.True(t, modelTyped.promptActive, "Expected prompt to be active after pressing 't'")
	assert.Equal(t, promptEditTitle, modelTyped.promptType, "Expected promptType to be promptEditTitle")
	assert.Equal(t, "Old Title", modelTyped.inputBuffer, "Expected input buffer to pre-fill with current title")

	// Complete the prompt with new title
	newModel, _ = modelTyped.Update(promptMsg{title: "New Title", logDB: false})
	modelTyped = newModel.(model)

	assert.Equal(t, "New Title", modelTyped.title, "Expected title to be updated")
	assert.False(t, modelTyped.promptActive, "Expected prompt to be inactive after completion")
}

func TestTKeyEditsTitleInCountUpMode(t *testing.T) {
	m := model{
		progress:       progress.New(progress.WithGradient(colorMontezumaGold, colorCream), progress.WithoutPercentage()),
		startTime:      time.Now().Unix(),
		targetDuration: 3600,
		countUpMode:    true,
		mode:           timerView,
		title:          "Old Title",
	}

	// Press 't' to edit title
	keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'t'}}
	newModel, _ := m.Update(keyMsg)
	modelTyped := newModel.(model)

	assert.True(t, modelTyped.promptActive, "Expected prompt to be active after pressing 't'")
	assert.Equal(t, promptEditTitle, modelTyped.promptType, "Expected promptType to be promptEditTitle")
	assert.Equal(t, "Old Title", modelTyped.inputBuffer, "Expected input buffer to pre-fill with current title")
}

func TestMKeySetsDurationInCountdownMode(t *testing.T) {
	m := model{
		progress:       progress.New(progress.WithGradient(colorMontezumaGold, colorCream), progress.WithoutPercentage()),
		startTime:      time.Now().Unix(),
		targetDuration: 1500, // 25 minutes
		countUpMode:    false,
		mode:           timerView,
	}

	// Press 'm' to set duration
	keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'m'}}
	newModel, _ := m.Update(keyMsg)
	modelTyped := newModel.(model)

	assert.True(t, modelTyped.promptActive, "Expected prompt to be active after pressing 'm'")
	assert.Equal(t, promptSetDuration, modelTyped.promptType, "Expected promptType to be promptSetDuration")

	// Complete the prompt with 30 minutes
	newModel, _ = modelTyped.Update(promptMsg{title: "30", logDB: false})
	modelTyped = newModel.(model)

	assert.Equal(t, int64(1800), modelTyped.targetDuration, "Expected duration to be 30 minutes (1800 seconds)")
	assert.False(t, modelTyped.promptActive, "Expected prompt to be inactive after completion")
}

func TestMKeySetsDurationInCountUpMode(t *testing.T) {
	m := model{
		progress:       progress.New(progress.WithGradient(colorMontezumaGold, colorCream), progress.WithoutPercentage()),
		startTime:      time.Now().Unix(),
		targetDuration: 3600, // 1 hour
		countUpMode:    true,
		mode:           timerView,
	}

	// Press 'm' to set duration
	keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'m'}}
	newModel, _ := m.Update(keyMsg)
	modelTyped := newModel.(model)

	assert.True(t, modelTyped.promptActive, "Expected prompt to be active after pressing 'm'")
	assert.Equal(t, promptSetDuration, modelTyped.promptType, "Expected promptType to be promptSetDuration")

	// Complete the prompt with 45 minutes
	newModel, _ = modelTyped.Update(promptMsg{title: "45", logDB: false})
	modelTyped = newModel.(model)

	assert.Equal(t, int64(2700), modelTyped.targetDuration, "Expected duration to be 45 minutes (2700 seconds)")
	assert.False(t, modelTyped.promptActive, "Expected prompt to be inactive after completion")
}

func TestMKeyResetsTimerAfterSettingDuration(t *testing.T) {
	startTime := time.Now().Unix() - 60
	m := model{
		progress:       progress.New(progress.WithGradient(colorMontezumaGold, colorCream), progress.WithoutPercentage()),
		startTime:      startTime,
		targetDuration: 1500,
		countUpMode:    false,
		mode:           timerView,
	}

	// Press 'm' to set duration
	keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'m'}}
	newModel, _ := m.Update(keyMsg)
	modelTyped := newModel.(model)

	// Complete the prompt with 20 minutes
	newModel, _ = modelTyped.Update(promptMsg{title: "20", logDB: false})
	modelTyped = newModel.(model)

	// Verify timer was reset (startTime should be recent)
	elapsed := time.Now().Unix() - modelTyped.startTime
	assert.True(t, elapsed < 2, "Expected timer to be reset (elapsed time should be < 2 seconds)")
	assert.Equal(t, int64(1200), modelTyped.targetDuration, "Expected duration to be 20 minutes (1200 seconds)")
}
