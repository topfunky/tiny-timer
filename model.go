package main

import (
	"time"

	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
)

type tickMsg time.Time

type promptMsg struct {
	title string
	logDB bool
}

type viewMode int

const (
	timerView viewMode = iota
	tableView
)

type promptType int

const (
	promptLogAndReset promptType = iota // Log session to DB and reset timer (d key)
	promptEditTitle                     // Edit title without logging (t key)
	promptSetDuration                   // Set target duration in minutes (m key)
)

type session struct {
	id        int
	datetime  string
	duration  int64
	completed bool
	title     string
}

type model struct {
	progress       progress.Model
	startTime      int64
	targetDuration int64
	title          string
	mode           viewMode
	table          table.Model
	countUpMode    bool
	inputBuffer    string
	promptActive   bool
	promptType     promptType
}

// Start the event loop
func (m model) Init() tea.Cmd {
	return tickCmd()
}

// Configure the event loop to run
func tickCmd() tea.Cmd {
	return tea.Tick(time.Second*1, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}
