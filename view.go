package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
)

// promptKeyMap defines keybindings for prompt mode
type promptKeyMap struct {
	Confirm key.Binding
	Cancel  key.Binding
}

func (k promptKeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Confirm, k.Cancel}
}

func (k promptKeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Confirm, k.Cancel},
	}
}

// tableKeyMap defines keybindings for table view mode
type tableKeyMap struct {
	Quit key.Binding
}

func (k tableKeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Quit}
}

func (k tableKeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Quit},
	}
}

// Handler that draws the UI of the application
func (m model) View() string {
	if m.promptActive {
		pad := strings.Repeat(" ", padding)
		var promptText string
		if m.promptType == 2 {
			promptText = "Enter duration in minutes: " + m.inputBuffer
		} else {
			promptText = "Enter task title: " + m.inputBuffer
		}
		promptKeys := promptKeyMap{
			Confirm: m.keys.Confirm,
			Cancel:  m.keys.Cancel,
		}
		return "\n" + pad + promptText + "\n\n" + pad + m.help.View(promptKeys)
	}

	if m.mode == tableView {
		pad := strings.Repeat(" ", padding)
		// Apply padding to each line of the table
		tableLines := strings.Split(m.table.View(), "\n")
		paddedTable := make([]string, len(tableLines))
		for i, line := range tableLines {
			paddedTable[i] = pad + line
		}
		tableKeys := tableKeyMap{
			Quit: m.keys.Quit,
		}
		return "\n" +
			strings.Join(paddedTable, "\n") + "\n\n" +
			pad + m.help.View(tableKeys)
	}

	elapsed := time.Now().Unix() - m.startTime
	remaining := m.targetDuration - elapsed

	if m.countUpMode {
		remaining = elapsed
	}

	if remaining < 0 {
		// When it completes, display the original duration of the timer
		remaining = 0
	}

	pad := strings.Repeat(" ", padding)

	// Display title if provided
	titleLine := ""
	if m.title != "" {
		titleLine = pad + m.title + "\n\n"
	}

	return "\n" +
		titleLine +
		pad + m.progress.View() + fmt.Sprintf(" %s \n\n", formatDurationAsMMSS(remaining)) +
		pad + m.help.View(m.keys)
}
