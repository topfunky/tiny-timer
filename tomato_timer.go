package main

// A TUI timer displaying a countdown in minutes and seconds.
//
// Accepts a single command line argument to set the timer duration in minutes.

import (
	"database/sql"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	_ "github.com/mattn/go-sqlite3"
)

const (
	padding                  = 2
	maxWidth                 = 80
	defaultDurationInMinutes = 25
	colorGrey                = "#626262"
	colorCream               = "#fefdbc"
	colorMontezumaGold       = "#f0c442"
	sqlite_db_file_path      = "/.config/tomato-timer/tomato-timer.db"
)

var helpStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(colorGrey)).Render

func main() {
	// Read CLI args, or use defaults
	var targetDurationInMinutes int64 = defaultDurationInMinutes
	if len(os.Args) > 1 {
		if arg, err := strconv.ParseInt(os.Args[1], 10, 64); err == nil && arg > 0 {
			targetDurationInMinutes = arg
		}
	}

	m := model{
		progress:       progress.New(progress.WithGradient(colorMontezumaGold, colorCream), progress.WithoutPercentage()),
		startTime:      time.Now().Unix(),
		targetDuration: targetDurationInMinutes * 60,
	}

	if _, err := tea.NewProgram(m).Run(); err != nil {
		fmt.Println("Oh no!", err)
		os.Exit(1)
	}
}

type tickMsg time.Time

type model struct {
	progress       progress.Model
	startTime      int64
	targetDuration int64
	table          table.Model
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

// Top level event handler that is called each time the screen is updated
func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		return updateKey(m, msg)

	case tea.WindowSizeMsg:
		return updateWindowSize(m, msg)

	case tickMsg:
		return updatePercent(m)

	// FrameMsg is sent when the progress bar wants to animate itself
	case progress.FrameMsg:
		progressModel, cmd := m.progress.Update(msg)
		m.progress = progressModel.(progress.Model)
		return m, cmd

	default:
		return m, nil
	}
}

// All individual event update handlers
// ---

func updatePercent(m model) (tea.Model, tea.Cmd) {
	if m.progress.Percent() == 1.0 {
		if err := sendNotification("Pomodoro CLI", "Timer has finished"); err != nil {
			fmt.Println("Error sending notification:", err)
		}

		// Save the session to the SQLite DB on completion
		if err := saveSessionToDB(m.targetDuration, true); err != nil {
			fmt.Println("Error saving session to DB:", err)
		}

		return m, tea.Quit
	}

	elapsed := time.Now().Unix() - m.startTime
	percentCompleted := float64(elapsed) / float64(m.targetDuration)

	cmd := m.progress.SetPercent(percentCompleted)
	return m, tea.Batch(tickCmd(), cmd)
}

func updateWindowSize(m model, msg tea.WindowSizeMsg) (tea.Model, tea.Cmd) {
	m.progress.Width = msg.Width - padding*2 - 4
	if m.progress.Width > maxWidth {
		m.progress.Width = maxWidth
	}
	return m, nil
}

func updateKey(m model, msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if msg.String() == "r" {
		// Reset timer
		m.startTime = time.Now().Unix()
		cmd := m.progress.SetPercent(0)
		return m, tea.Batch(tickCmd(), cmd)
	} else {
		// Quit if any key is pressed

		// Save the uncompleted session to the SQLite DB
		if err := saveSessionToDB(m.targetDuration, false); err != nil {
			fmt.Println("Error saving session to DB:", err)
		}

		return m, tea.Quit
	}
}

// Handler that draws the UI of the application
func (m model) View() string {
	remaining := m.targetDuration - (time.Now().Unix() - m.startTime)
	if remaining <= 0 {
		// When it completes, display the original duration of the timer
		remaining = m.targetDuration
	}

	pad := strings.Repeat(" ", padding)
	return "\n" +
		pad + m.progress.View() + fmt.Sprintf(" %s \n\n", formatDurationAsMMSS(remaining)) +
		pad + helpStyle("Press 'r' to reset timer • Press any other key to quit")

	// return baseStyle.Render(m.table.View()) + "\n  " + m.table.HelpView() + "\n"
}

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

// Save a record to SQLite DB that represents a working session as counted by the timer
func saveSessionToDB(duration int64, completed bool) error {
	dbPath := os.Getenv("HOME") + sqlite_db_file_path
	if err := os.MkdirAll(os.Getenv("HOME")+"/.config/tomato-timer", os.ModePerm); err != nil {
		return err
	}
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return err
	}
	defer db.Close()

	createTableSQL := `CREATE TABLE IF NOT EXISTS sessions (
		"id" INTEGER PRIMARY KEY AUTOINCREMENT,
		"logged_at" DATETIME DEFAULT CURRENT_TIMESTAMP,
		"duration" INTEGER,
		"completed" BOOLEAN
	);`

	_, err = db.Exec(createTableSQL)
	if err != nil {
		return err
	}

	insertSessionSQL := `INSERT INTO sessions (duration, completed) VALUES (?, ?)`
	_, err = db.Exec(insertSessionSQL, duration, completed)
	if err != nil {
		return err
	}

	return nil
}

// Generate a report as a Bubble table
//
// https://github.com/charmbracelet/bubbletea/blob/master/examples/table/main.go
//
// select datetime, completed, (duration / 60) as duration_minutes from sessions;
//
// select
//   case cast (strftime('%w', '2022-01-01') as integer)
//   when 0 then 'Sunday'
//   when 1 then 'Monday'
//   when 2 then 'Tuesday'
//   when 3 then 'Wednesday'
//   when 4 then 'Thursday'
//   when 5 then 'Friday'
//   else 'Saturday' end as weekday
