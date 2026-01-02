package main

import (
	"fmt"
	"time"

	"github.com/charmbracelet/bubbles/progress"
	tea "github.com/charmbracelet/bubbletea"
)

// Top level event handler that is called each time the screen is updated
func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		return updateKey(m, msg)

	case tea.WindowSizeMsg:
		return updateWindowSize(m, msg)

	case tickMsg:
		return updatePercent(m)

	case promptMsg:
		return handlePromptInput(m, msg)

	// FrameMsg is sent when the progress bar wants to animate itself
	case progress.FrameMsg:
		progressModel, cmd := m.progress.Update(msg)
		m.progress = progressModel.(progress.Model)
		return m, cmd

	default:
		return m, nil
	}
}

func updatePercent(m model) (tea.Model, tea.Cmd) {
	elapsed := time.Now().Unix() - m.startTime

	if m.countUpMode {
		// In count-up mode, just update the progress bar to show elapsed time
		percentCompleted := float64(elapsed) / float64(m.targetDuration)
		if percentCompleted > 1.0 {
			percentCompleted = 1.0
		}
		cmd := m.progress.SetPercent(percentCompleted)
		return m, tea.Batch(tickCmd(), cmd)
	}

	percentCompleted := float64(elapsed) / float64(m.targetDuration)

	// Check for completion based on actual elapsed time
	if percentCompleted >= 1.0 {
		// Ensure progress is set to 100% for final display
		m.progress.SetPercent(1.0)

		if err := sendNotification("Pomodoro CLI", "Timer has finished"); err != nil {
			fmt.Println("Error sending notification:", err)
		}

		// Save the session to the SQLite DB on completion
		if err := saveSessionToDB(m.targetDuration, true, m.title); err != nil {
			fmt.Println("Error saving session to DB:", err)
		}

		return m, tea.Quit
	}

	cmd := m.progress.SetPercent(percentCompleted)
	return m, tea.Batch(tickCmd(), cmd)
}

func updateWindowSize(m model, msg tea.WindowSizeMsg) (tea.Model, tea.Cmd) {
	m.progress.Width = msg.Width - padding*2 - 4
	if m.progress.Width > maxWidth {
		m.progress.Width = maxWidth
	}
	return m, nil
}

func handlePromptInput(m model, msg promptMsg) (tea.Model, tea.Cmd) {
	m.promptActive = false

	if m.promptType == 0 {
		// Log session to DB and start new count
		elapsed := time.Now().Unix() - m.startTime
		if err := saveSessionToDB(elapsed, true, msg.title); err != nil {
			fmt.Println("Error saving session to DB:", err)
		}
		// Reset timer for new session
		m.startTime = time.Now().Unix()
		cmd := m.progress.SetPercent(0)
		m.title = ""
		return m, tea.Batch(tickCmd(), cmd)
	} else {
		// Just update title without logging
		m.title = msg.title
		return m, tickCmd()
	}
}

// handlePromptKeyInput handles key input when prompt is active
func handlePromptKeyInput(m model, msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if msg.Type == tea.KeyEnter {
		return handlePromptInput(m, promptMsg{title: m.inputBuffer, logDB: m.promptType == 0})
	} else if msg.Type == tea.KeyEsc {
		m.promptActive = false
		return m, nil
	} else if msg.Type == tea.KeyBackspace {
		if len(m.inputBuffer) > 0 {
			m.inputBuffer = m.inputBuffer[:len(m.inputBuffer)-1]
		}
		return m, nil
	} else if msg.Type == tea.KeySpace {
		m.inputBuffer += " "
		return m, nil
	} else if msg.Type == tea.KeyRunes {
		for _, r := range msg.Runes {
			m.inputBuffer += string(r)
		}
		return m, nil
	}
	return m, nil
}

// handleTableViewKey handles key input when in table view mode
func handleTableViewKey(m model, msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Any key exits table view
	m.mode = timerView
	return m, nil
}

// handleCountUpModeKey handles key input in count-up mode
func handleCountUpModeKey(m model, msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()
	switch key {
	case "d":
		// Prompt for title, log to DB, and start new session
		m.promptActive = true
		m.promptType = 0
		m.inputBuffer = m.title
		return m, nil
	case "D":
		// Prompt for title only, continue timer without logging
		m.promptActive = true
		m.promptType = 1
		m.inputBuffer = m.title
		return m, nil
	case "r":
		// Reset timer in count-up mode
		m.startTime = time.Now().Unix()
		cmd := m.progress.SetPercent(0)
		return m, tea.Batch(tickCmd(), cmd)
	case "t":
		// Show table view
		t, err := buildTableView(10)
		if err != nil {
			fmt.Println("Error fetching sessions:", err)
			return m, nil
		}
		m.table = t
		m.mode = tableView
		return m, nil
	default:
		// Quit on other keys
		return m, tea.Quit
	}
}

// handleTimerModeKey handles key input in timer mode (non count-up)
func handleTimerModeKey(m model, msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()
	switch key {
	case "D":
		// Prompt for title only
		m.promptActive = true
		m.promptType = 1
		m.inputBuffer = m.title
		return m, nil
	case "r":
		// Reset timer
		m.startTime = time.Now().Unix()
		cmd := m.progress.SetPercent(0)
		return m, tea.Batch(tickCmd(), cmd)
	case "t":
		// Show table view of recent sessions
		t, err := buildTableView(10)
		if err != nil {
			fmt.Println("Error fetching sessions:", err)
			return m, nil
		}
		m.table = t
		m.mode = tableView
		return m, nil
	default:
		// Quit if any other key is pressed
		return m, tea.Quit
	}
}

func updateKey(m model, msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Handle Ctrl-Z to suspend in all modes
	if msg.Type == tea.KeyCtrlZ {
		return m, tea.Suspend
	}

	// Handle prompt input mode
	if m.promptActive {
		return handlePromptKeyInput(m, msg)
	}

	// Handle table view mode
	if m.mode == tableView {
		return handleTableViewKey(m, msg)
	}

	// Handle count-up mode keys
	if m.countUpMode {
		return handleCountUpModeKey(m, msg)
	}

	// Handle timer view mode (non count-up)
	return handleTimerModeKey(m, msg)
}
