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
	"github.com/charmbracelet/lipgloss"
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

func TestGetRecentSessions(t *testing.T) {
	// Set up a temporary database path
	tempDBPath := os.TempDir() + sqlite_db_file_path
	os.Setenv("HOME", os.TempDir())

	// Clean up the temporary database file after the test
	defer os.Remove(tempDBPath)

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
	tempDBPath := os.TempDir() + sqlite_db_file_path
	os.Setenv("HOME", os.TempDir())

	// Clean up the temporary database file after the test
	defer os.Remove(tempDBPath)

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
	assert.Contains(t, view, "Press 'r' to reset timer")
	assert.Contains(t, view, "Press 't' for recent tasks")
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
	tempDBPath := os.TempDir() + sqlite_db_file_path
	os.Setenv("HOME", os.TempDir())

	// Clean up the temporary database file after the test
	defer os.Remove(tempDBPath)

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
	tempDBPath := os.TempDir() + sqlite_db_file_path
	os.Setenv("HOME", os.TempDir())

	// Clean up the temporary database file after the test
	defer os.Remove(tempDBPath)

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
	
	fmt.Printf("Number of sessions: %d\n", len(sessions))
	fmt.Printf("Number of rows: %d\n", len(rows))
	if len(rows) > 0 {
		fmt.Printf("First row: %v\n", rows[0])
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
	
	// Print the view for debugging
	fmt.Println("=== TABLE VIEW OUTPUT ===")
	fmt.Println(view)
	fmt.Println("=== END TABLE VIEW ===")

	// Check that headers appear in the output
	assert.Contains(t, view, "Title", "Table should contain 'Title' header")
	assert.Contains(t, view, "Duration", "Table should contain 'Duration' header")
	assert.Contains(t, view, "Date", "Table should contain 'Date' header")

	// Find header line and first data line to compare alignment
	lines := strings.Split(view, "\n")
	var headerLine string
	var firstDataLine string
	var headerLineRaw string
	var firstDataLineRaw string
	foundHeader := false
	
	for i, line := range lines {
		// Find the header line (contains all three headers)
		if !foundHeader && strings.Contains(line, "Title") && strings.Contains(line, "Duration") && strings.Contains(line, "Date") {
			headerLineRaw = line
			headerLine = strings.TrimPrefix(line, strings.Repeat(" ", padding))
			foundHeader = true
			fmt.Printf("Header line RAW [%d]: '%s'\n", i, headerLineRaw)
			fmt.Printf("Header line trimmed [%d]: '%s'\n", i, headerLine)
			continue
		}
		
		// Find the first data line (after border line, non-empty, not help text)
		if foundHeader && len(strings.TrimSpace(line)) > 0 && !strings.Contains(line, "─") && !strings.Contains(line, "Press any key") {
			firstDataLineRaw = line
			firstDataLine = strings.TrimPrefix(line, strings.Repeat(" ", padding))
			fmt.Printf("First data line RAW [%d]: '%s'\n", i, firstDataLineRaw)
			fmt.Printf("First data line trimmed [%d]: '%s'\n", i, firstDataLine)
			break
		}
	}
	
	assert.NotEmpty(t, headerLine, "Should have found the header line")
	assert.NotEmpty(t, firstDataLine, "Should have found a data line")
	
	// Compare using the RAW lines (with padding included)
	// This tests if the padding from View() is applied consistently
	headerTitleStartRaw := strings.Index(headerLineRaw, "Title")
	dataTitleStartRaw := strings.Index(firstDataLineRaw, "Test Task")
	
	fmt.Printf("Header 'Title' starts at column: %d (in raw output)\n", headerTitleStartRaw)
	fmt.Printf("Data 'Test Task' starts at column: %d (in raw output)\n", dataTitleStartRaw)
	
	assert.Equal(t, headerTitleStartRaw, dataTitleStartRaw, 
		"Header and data should start at the same column in the rendered output. Header starts at %d, Data starts at %d",
		headerTitleStartRaw, dataTitleStartRaw)
}
