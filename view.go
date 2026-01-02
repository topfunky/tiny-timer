package main

import (
	"fmt"
	"strings"
	"time"
)

// Handler that draws the UI of the application
func (m model) View() string {
	if m.promptActive {
		pad := strings.Repeat(" ", padding)
		promptText := "Enter task title: " + m.inputBuffer
		return "\n" + pad + promptText + "\n\n" + pad + helpStyle("Press Enter to confirm")
	}

	if m.mode == tableView {
		pad := strings.Repeat(" ", padding)
		// Apply padding to each line of the table
		tableLines := strings.Split(m.table.View(), "\n")
		paddedTable := make([]string, len(tableLines))
		for i, line := range tableLines {
			paddedTable[i] = pad + line
		}
		return "\n" +
			strings.Join(paddedTable, "\n") + "\n\n" +
			pad + helpStyle("Press any key to return to timer")
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

	var helpText string
	if m.countUpMode {
		helpText = "Press 'd' to log task • 'D' to change title • 'r' to reset • 't' for history"
	} else {
		helpText = "Press 'D' to set title • 'r' to reset • 't' for history • any other key to quit"
	}

	return "\n" +
		titleLine +
		pad + m.progress.View() + fmt.Sprintf(" %s \n\n", formatDurationAsMMSS(remaining)) +
		pad + renderHelpText(helpText)
}
