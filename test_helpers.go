package main

import (
	"os"
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/progress"
	"tiny-timer/status"
)

// newTestModel creates a model with default test values, including help and keys
func newTestModel() model {
	keys := keyMap{
		Done: key.NewBinding(
			key.WithKeys("d"),
			key.WithHelp("d", "done"),
		),
		History: key.NewBinding(
			key.WithKeys("h"),
			key.WithHelp("h", "history"),
		),
		Title: key.NewBinding(
			key.WithKeys("t"),
			key.WithHelp("t", "title"),
		),
		Minutes: key.NewBinding(
			key.WithKeys("m"),
			key.WithHelp("m", "minutes"),
		),
		Reset: key.NewBinding(
			key.WithKeys("r"),
			key.WithHelp("r", "reset"),
		),
		Quit: key.NewBinding(
			key.WithKeys("q", "esc", "ctrl+c"),
			key.WithHelp("q/esc", "quit"),
		),
		Confirm: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "confirm"),
		),
		Cancel: key.NewBinding(
			key.WithKeys("esc"),
			key.WithHelp("esc", "cancel"),
		),
		Backspace: key.NewBinding(
			key.WithKeys("backspace"),
			key.WithHelp("backspace", "delete"),
		),
	}
	statusCmp := status.NewStatusCmp()
	statusCmp.SetKeyMap(keys)
	return model{
		progress:       progress.New(progress.WithGradient(colorMontezumaGold, colorCream), progress.WithoutPercentage()),
		startTime:      time.Now().Unix(),
		targetDuration: 60,
		help:           newHelpModel(),
		keys:           keys,
		status:         statusCmp,
	}
}

// setupTestDB sets up a temporary database for testing and returns the cleanup function
func setupTestDB(t *testing.T) (string, func()) {
	// Close any existing database connection
	closeDBConnection()

	// Set HOME to temp directory for testing
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", os.TempDir())

	// Get the database path (will use temp HOME)
	dbPath, err := getDBPath()
	if err != nil {
		t.Fatalf("Failed to get DB path: %v", err)
	}

	// Initialize the database connection for tests
	if err := initDBConnection(); err != nil {
		t.Fatalf("Failed to initialize DB connection: %v", err)
	}

	// Cleanup function
	cleanup := func() {
		closeDBConnection()
		os.Remove(dbPath)
		os.Setenv("HOME", originalHome)
	}

	return dbPath, cleanup
}

// extractColorCode extracts the ANSI color code from a string
func extractColorCode(s string) string {
	// Look for ANSI escape sequences: \x1b[38;5;XXXm or \x1b[38;2;R;G;Bm
	start := strings.Index(s, "\x1b[")
	if start == -1 {
		return ""
	}
	end := strings.Index(s[start:], "m")
	if end == -1 {
		return ""
	}
	return s[start : start+end+1]
}
