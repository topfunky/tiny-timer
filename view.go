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
		var promptText string
		if m.promptType == 2 {
			promptText = "Enter duration in minutes: " + m.inputBuffer
		} else {
			promptText = "Enter task title: " + m.inputBuffer
		}
		return "\n" + pad + promptText + "\n\n" + pad + helpStyle("Press Enter to confirm, Esc to cancel")
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
		helpText = "'d' done • 'h' history • 't' title • 'm' minutes • 'r' reset • any other key to quit"
	} else {
		helpText = "'d' done • 'h' history • 't' title • 'm' minutes • 'r' reset • any other key to quit"
	}

	return "\n" +
		titleLine +
		pad + m.progress.View() + fmt.Sprintf(" %s \n\n", formatDurationAsMMSS(remaining)) +
		pad + renderHelpText(helpText)
}
