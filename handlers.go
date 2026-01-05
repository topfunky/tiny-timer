package main

import (
	"fmt"
	"log"
	"strconv"
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
	m.help.Width = msg.Width
	return m, nil
}

func handlePromptInput(m model, msg promptMsg) (tea.Model, tea.Cmd) {
	m.promptActive = false

	switch m.promptType {
	case promptLogAndReset:
		// Log session to DB and start new session
		elapsed := time.Now().Unix() - m.startTime
		log.Printf("handlePromptInput: Saving session, elapsed=%d, title=%q", elapsed, msg.title)
		if err := saveSessionToDB(elapsed, true, msg.title); err != nil {
			log.Printf("handlePromptInput: Error saving session: %v", err)
			fmt.Println("Error saving session to DB:", err)
		}
		// Refresh history table if we are logging
		log.Printf("handlePromptInput: Building table view after save")
		if t, err := buildTableView(10); err == nil {
			m.table = t
			log.Printf("handlePromptInput: Table view built with %d rows", len(t.Rows()))
		} else {
			log.Printf("handlePromptInput: Error building table view: %v", err)
		}
		// Reset timer for new session
		m.startTime = time.Now().Unix()
		cmd := m.progress.SetPercent(0)
		m.title = ""
		return m, tea.Batch(tickCmd(), cmd)
	case promptEditTitle:
		// Just update title without logging
		m.title = msg.title
		return m, tickCmd()
	case promptSetDuration:
		// Set duration in minutes
		var minutes int64
		if n, err := strconv.ParseInt(msg.title, 10, 64); err == nil && n > 0 {
			minutes = n
		} else {
			// Invalid input, keep current duration
			return m, tickCmd()
		}
		m.targetDuration = minutes * 60
		// Reset timer with new duration
		m.startTime = time.Now().Unix()
		cmd := m.progress.SetPercent(0)
		return m, tea.Batch(tickCmd(), cmd)
	}
	return m, tickCmd()
}

// handlePromptKeyInput handles key input when prompt is active
func handlePromptKeyInput(m model, msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if msg.Type == tea.KeyEnter {
		return handlePromptInput(m, promptMsg{title: m.inputBuffer, logDB: m.promptType == promptLogAndReset})
	} else if msg.Type == tea.KeyEsc {
		m.promptActive = false
		return m, nil
	} else if msg.Type == tea.KeyBackspace {
		if len(m.inputBuffer) > 0 {
			m.inputBuffer = m.inputBuffer[:len(m.inputBuffer)-1]
		}
		return m, nil
	} else if msg.Type == tea.KeySpace {
		// Only allow space for title prompts, not duration
		if m.promptType != promptSetDuration {
			m.inputBuffer += " "
		}
		return m, nil
	} else if msg.Type == tea.KeyRunes {
		for _, r := range msg.Runes {
			// For duration prompts, only allow numeric characters
			if m.promptType == promptSetDuration {
				if r >= '0' && r <= '9' {
					m.inputBuffer += string(r)
				}
			} else {
				m.inputBuffer += string(r)
			}
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
		// Mark task as done: prompt for title, log to DB, and start new session
		m.promptActive = true
		m.promptType = promptLogAndReset
		m.inputBuffer = m.title
		return m, nil
	case "h":
		// Show history (table view)
		log.Printf("handleCountUpModeKey: 'h' pressed, fetching history")
		t, err := buildTableView(10)
		if err != nil {
			log.Printf("handleCountUpModeKey: Error fetching sessions: %v", err)
			fmt.Println("Error fetching sessions:", err)
			return m, nil
		}
		log.Printf("handleCountUpModeKey: History fetched, %d rows", len(t.Rows()))
		m.table = t
		m.mode = tableView
		return m, nil
	case "t":
		// Edit current task title
		m.promptActive = true
		m.promptType = promptEditTitle
		m.inputBuffer = m.title
		return m, nil
	case "m":
		// Set target duration in minutes
		m.promptActive = true
		m.promptType = promptSetDuration
		// Pre-fill with current duration in minutes
		currentMinutes := m.targetDuration / 60
		m.inputBuffer = strconv.FormatInt(currentMinutes, 10)
		return m, nil
	case "r":
		// Reset timer in count-up mode
		m.startTime = time.Now().Unix()
		cmd := m.progress.SetPercent(0)
		return m, tea.Batch(tickCmd(), cmd)
	default:
		// Quit on other keys
		return m, tea.Quit
	}
}

// handleTimerModeKey handles key input in timer mode (non count-up)
func handleTimerModeKey(m model, msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()
	switch key {
	case "d":
		// Mark task as done: prompt for title, log to DB, and reset timer
		m.promptActive = true
		m.promptType = promptLogAndReset
		m.inputBuffer = m.title
		return m, nil
	case "h":
		// Show history (table view)
		log.Printf("handleCountUpModeKey: 'h' pressed, fetching history")
		t, err := buildTableView(10)
		if err != nil {
			log.Printf("handleCountUpModeKey: Error fetching sessions: %v", err)
			fmt.Println("Error fetching sessions:", err)
			return m, nil
		}
		log.Printf("handleCountUpModeKey: History fetched, %d rows", len(t.Rows()))
		m.table = t
		m.mode = tableView
		return m, nil
	case "t":
		// Edit current task title
		m.promptActive = true
		m.promptType = promptEditTitle
		m.inputBuffer = m.title
		return m, nil
	case "m":
		// Set target duration in minutes
		m.promptActive = true
		m.promptType = promptSetDuration
		// Pre-fill with current duration in minutes
		currentMinutes := m.targetDuration / 60
		m.inputBuffer = strconv.FormatInt(currentMinutes, 10)
		return m, nil
	case "r":
		// Reset timer
		m.startTime = time.Now().Unix()
		cmd := m.progress.SetPercent(0)
		return m, tea.Batch(tickCmd(), cmd)
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
