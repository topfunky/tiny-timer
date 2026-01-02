package main

import (
	"fmt"
	"os/exec"
	"runtime"
	"testing"
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
