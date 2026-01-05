package main

import (
	"time"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
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

// keyMap defines a set of keybindings. To work for help it must satisfy
// key.Map. It could also very easily be a map[string]key.Binding.
type keyMap struct {
	Done     key.Binding
	History  key.Binding
	Title    key.Binding
	Minutes  key.Binding
	Reset    key.Binding
	Quit     key.Binding
	Confirm  key.Binding
	Cancel   key.Binding
	Backspace key.Binding
}

// ShortHelp returns keybindings to be shown in the mini help view. It's part
// of the key.Map interface.
func (k keyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Done, k.History, k.Title, k.Minutes, k.Reset, k.Quit}
}

// FullHelp returns keybindings for the expanded help view. It's part of the
// key.Map interface.
func (k keyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Done, k.History, k.Title, k.Minutes, k.Reset}, // first column
		{k.Quit}, // second column
	}
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
	help           help.Model
	keys           keyMap
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
