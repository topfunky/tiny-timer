package main

import (
	"fmt"
	"os/exec"
	"runtime"
	"testing"

	"charm.land/bubbles/v2/help"
	"charm.land/lipgloss/v2"
)

// A display helper for formatting the time remaining in the timer
func formatDurationAsMMSS(duration int64) string {
	hours := duration / 60
	minutes := duration % 60
	return fmt.Sprintf("%02d:%02d", hours, minutes)
}

// Trigger a macOS notification
func sendNotification(title, message string) error {
	if testing.Testing() {
		return nil
	}
	if runtime.GOOS != "darwin" {
		return nil
	}
	cmd := exec.Command("osascript", "-e", fmt.Sprintf(`display notification "%s" with title "%s" sound name "Bottle"`, message, title))
	return cmd.Run()
}

// newHelpModel creates a help model with custom two-tone grey styling:
// - Keys (single letters) render in light grey (#a0a0a0)
// - Descriptions render in dark grey (#626262)
func newHelpModel() help.Model {
	h := help.New()
	
	// Customize styles for two-tone grey
	keyStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(colorLightGrey))
	descStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(colorGrey))
	sepStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(colorGrey))
	
	h.Styles.ShortKey = keyStyle
	h.Styles.ShortDesc = descStyle
	h.Styles.ShortSeparator = sepStyle
	h.Styles.FullKey = keyStyle
	h.Styles.FullDesc = descStyle
	h.Styles.FullSeparator = sepStyle
	h.Styles.Ellipsis = sepStyle
	
	return h
}
