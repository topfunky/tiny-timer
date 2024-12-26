package main

import (
	"testing"

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
