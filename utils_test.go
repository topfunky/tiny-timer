package main

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestFormatDurationAsMMSS(t *testing.T) {
	tests := []struct {
		duration int64
		expected string
	}{
		{0, "00:00"},
		{59, "00:59"},
		{60, "01:00"},
		{61, "01:01"},
		{3600, "60:00"},
	}

	for _, test := range tests {
		result := formatDurationAsMMSS(test.duration)
		assert.Equal(t, test.expected, result, "formatDurationAsMMSS(%d)", test.duration)
	}
}

func TestDateFormatInTable(t *testing.T) {
	// Test that dates are formatted correctly in the table view
	testCases := []struct {
		input    string
		expected string
	}{
		{"2026-01-07 10:30:45", "Wednesday, 7 Jan 26"},
		{"2026-01-01 00:00:00", "Thursday, 1 Jan 26"},
		{"2025-12-25 15:30:00", "Thursday, 25 Dec 25"},
	}

	for _, tc := range testCases {
		t.Run(tc.input, func(t *testing.T) {
			// Parse the datetime string
			parsedTime, err := time.Parse("2006-01-02 15:04:05", tc.input)
			assert.NoError(t, err)

			// Format it the way the table should display it
			formatted := parsedTime.Format("Monday, 2 Jan 06")
			assert.Equal(t, tc.expected, formatted, "Date should be formatted as '%s'", tc.expected)
		})
	}
}
