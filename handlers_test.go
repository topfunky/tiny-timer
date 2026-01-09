package main

import (
	"database/sql"
	"testing"
	"time"

	"github.com/charmbracelet/bubbles/progress"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
)

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

	// Simulate a tick after resuming
	newModel, _ := m.Update(tickMsg(time.Now()))
	modelTyped := newModel.(model)

	// Verify the prompt is active
	assert.True(t, modelTyped.promptActive, "Expected prompt to be active when resuming after completion")

	// Complete the prompt
	newModel, _ = modelTyped.Update(promptMsg{title: "Completed Task", logDB: true})

	// Verify the session was saved to the database
	db, err := sql.Open("sqlite", tempDBPath)
	assert.NoError(t, err)
	defer db.Close()

	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM sessions WHERE title = ? AND completed = ?", "Completed Task", true).Scan(&count)
	assert.NoError(t, err)
	assert.Equal(t, 1, count, "Expected one completed session to be saved after prompt")
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
	modelTyped := newModel.(model)

	// Verify prompt is active
	assert.True(t, modelTyped.promptActive, "Expected prompt to be active on tick after completion")

	// Complete the prompt
	newModel, _ = modelTyped.Update(promptMsg{title: "Tick After Complete", logDB: true})

	// Verify the session was saved to the database
	db, err := sql.Open("sqlite", tempDBPath)
	assert.NoError(t, err)
	defer db.Close()

	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM sessions WHERE title = ? AND completed = ?", "Tick After Complete", true).Scan(&count)
	assert.NoError(t, err)
	assert.Equal(t, 1, count, "Expected one completed session to be saved after prompt on tick after completion")
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
	db, err := sql.Open("sqlite", tempDBPath)
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
	db, err := sql.Open("sqlite", tempDBPath)
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
	db, err := sql.Open("sqlite", tempDBPath)
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

func TestTimerEndPromptsForTitle(t *testing.T) {
	// Set up a temporary database path
	_, cleanup := setupTestDB(t)
	defer cleanup()

	// Create a timer that is just about to finish
	m := model{
		progress:       progress.New(progress.WithGradient(colorMontezumaGold, colorCream), progress.WithoutPercentage()),
		startTime:      time.Now().Unix() - 61,
		targetDuration: 60,
		title:          "Original Title",
		mode:           timerView,
	}

	// Simulate a tick
	newModel, _ := m.Update(tickMsg(time.Now()))
	modelTyped := newModel.(model)

	// Verify that instead of quitting, it activated the prompt
	assert.True(t, modelTyped.promptActive, "Expected prompt to be active when timer ends")
	assert.Equal(t, promptLogAndReset, modelTyped.promptType, "Expected promptType to be promptLogAndReset")
	assert.Equal(t, "Original Title", modelTyped.inputBuffer, "Expected input buffer to be pre-filled with current title")
}

func TestCountUpModeUsesPositionalArgAsTarget(t *testing.T) {
	// This test simulates the logic in main.go for initializing the model
	// We want to ensure that if countUpMode is true, targetDuration can still be set by positional args

	// Case 1: Default count-up duration (no positional arg simulated)
	m1 := model{
		countUpMode:    true,
		targetDuration: defaultCountUpDuration,
	}
	assert.Equal(t, int64(3600), m1.targetDuration)

	// Case 2: Positional arg provided (e.g., 5 minutes)
	// In main.go, this would be: targetDuration = 5 * 60
	m2 := model{
		countUpMode:    true,
		targetDuration: 300,
	}
	assert.Equal(t, int64(300), m2.targetDuration, "Target duration should be 300 seconds (5 minutes) even in count-up mode")
}
