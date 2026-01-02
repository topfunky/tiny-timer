package main

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

const (
	padding                  = 2
	maxWidth                 = 80
	defaultDurationInMinutes = 25
	defaultCountUpDuration   = 3600
	colorGrey                = "#626262"
	colorLightGrey           = "#a0a0a0"
	colorCream               = "#fefdbc"
	colorMontezumaGold       = "#f0c442"
	sqliteDBFilePath         = "/.config/tomato-timer/tomato-timer.db"
)

var helpStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(colorGrey)).Render

// renderHelpText styles help text with two-tone formatting: shortcut letters in light grey, rest in grey
func renderHelpText(text string) string {
	lightGreyStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(colorLightGrey))
	greyStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(colorGrey))

	var result strings.Builder
	i := 0
	for i < len(text) {
		if text[i] == '\'' && i+1 < len(text) {
			// Found opening quote, find closing quote
			start := i + 1
			end := start
			for end < len(text) && text[end] != '\'' {
				end++
			}
			if end < len(text) {
				// Found closing quote, style the shortcut letter(s) in light grey
				shortcut := text[start:end]
				result.WriteString(lightGreyStyle.Render(shortcut))
				i = end + 1
				continue
			}
		}
		// Collect regular characters until we hit a quote or end of string
		regularStart := i
		for i < len(text) && text[i] != '\'' {
			i++
		}
		if regularStart < i {
			result.WriteString(greyStyle.Render(text[regularStart:i]))
		}
	}
	return result.String()
}
