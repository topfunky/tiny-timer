package main

// A TUI timer displaying a countdown in minutes and seconds.
//
// Accepts a single command line argument to set the timer duration in minutes.

import (
	"database/sql"
	"flag"
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
	defaultCountUpDuration   = 3600
	colorGrey                = "#626262"
	colorCream               = "#fefdbc"
	colorMontezumaGold       = "#f0c442"
	sqlite_db_file_path      = "/.config/tomato-timer/tomato-timer.db"
)

var helpStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(colorGrey)).Render

func main() {
	// Parse CLI flags
	titleFlag := flag.String("title", "", "Optional title for the timer session")
	countUpFlag := flag.Bool("count-up", false, "Enable count-up mode (logs task time after completion)")
	flag.Parse()

	// Read positional arg for duration, or use default
	var targetDurationInMinutes int64 = defaultDurationInMinutes
	if flag.NArg() > 0 {
		if arg, err := strconv.ParseInt(flag.Arg(0), 10, 64); err == nil && arg > 0 {
			targetDurationInMinutes = arg
		}
	}

	targetDuration := targetDurationInMinutes * 60
	if *countUpFlag {
		targetDuration = defaultCountUpDuration
	}

	m := model{
		progress:       progress.New(progress.WithGradient(colorMontezumaGold, colorCream), progress.WithoutPercentage()),
		startTime:      time.Now().Unix(),
		targetDuration: targetDuration,
		title:          *titleFlag,
		countUpMode:    *countUpFlag,
	}

	if _, err := tea.NewProgram(m).Run(); err != nil {
		fmt.Println("Oh no!", err)
		os.Exit(1)
	}
}

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
	promptType     int // 0 = new session (d), 1 = title only (D)
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

	case promptMsg:
		return handlePromptInput(m, msg)

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
	elapsed := time.Now().Unix() - m.startTime

	if m.countUpMode {
		// In count-up mode, just update the progress bar to show elapsed time
		percentCompleted := float64(elapsed) / float64(m.targetDuration)
		if percentCompleted > 1.0 {
			percentCompleted = 1.0
		}
		cmd := m.progress.SetPercent(percentCompleted)
		return m, tea.Batch(tickCmd(), cmd)
	}

	percentCompleted := float64(elapsed) / float64(m.targetDuration)

	// Check for completion based on actual elapsed time
	if percentCompleted >= 1.0 {
		// Ensure progress is set to 100% for final display
		m.progress.SetPercent(1.0)

		if err := sendNotification("Pomodoro CLI", "Timer has finished"); err != nil {
			fmt.Println("Error sending notification:", err)
		}

		// Save the session to the SQLite DB on completion
		if err := saveSessionToDB(m.targetDuration, true, m.title); err != nil {
			fmt.Println("Error saving session to DB:", err)
		}

		return m, tea.Quit
	}

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

func handlePromptInput(m model, msg promptMsg) (tea.Model, tea.Cmd) {
	m.promptActive = false

	if m.promptType == 0 {
		// Log session to DB and start new count
		elapsed := time.Now().Unix() - m.startTime
		if err := saveSessionToDB(elapsed, true, msg.title); err != nil {
			fmt.Println("Error saving session to DB:", err)
		}
		// Reset timer for new session
		m.startTime = time.Now().Unix()
		cmd := m.progress.SetPercent(0)
		m.title = ""
		return m, tea.Batch(tickCmd(), cmd)
	} else {
		// Just update title without logging
		m.title = msg.title
		return m, tickCmd()
	}
}

func updateKey(m model, msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Handle prompt input mode
	if m.promptActive {
		if msg.Type == tea.KeyEnter {
			return handlePromptInput(m, promptMsg{title: m.inputBuffer, logDB: m.promptType == 0})
		} else if msg.Type == tea.KeyEsc {
			m.promptActive = false
			return m, nil
		} else if msg.Type == tea.KeyBackspace {
			if len(m.inputBuffer) > 0 {
				m.inputBuffer = m.inputBuffer[:len(m.inputBuffer)-1]
			}
			return m, nil
		} else if msg.Type == tea.KeySpace {
			m.inputBuffer += " "
			return m, nil
		} else if msg.Type == tea.KeyRunes {
			for _, r := range msg.Runes {
				m.inputBuffer += string(r)
			}
			return m, nil
		}
		return m, nil
	}

	// Handle table view mode
	if m.mode == tableView {
		// Allow Ctrl-Z to suspend in table view
		if msg.Type == tea.KeyCtrlZ {
			return m, tea.Suspend
		}
		// Any other key exits table view
		m.mode = timerView
		return m, nil
	}

	// Allow Ctrl-Z to suspend the process in timer view
	if msg.Type == tea.KeyCtrlZ {
		return m, tea.Suspend
	}

	// Handle count-up mode keys
	if m.countUpMode {
		if msg.String() == "d" {
			// Prompt for title, log to DB, and start new session
			m.promptActive = true
			m.promptType = 0
			m.inputBuffer = m.title
			return m, nil
		} else if msg.String() == "D" {
			// Prompt for title only, continue timer without logging
			m.promptActive = true
			m.promptType = 1
			m.inputBuffer = m.title
			return m, nil
		} else if msg.String() == "r" {
			// Reset timer in count-up mode
			m.startTime = time.Now().Unix()
			cmd := m.progress.SetPercent(0)
			return m, tea.Batch(tickCmd(), cmd)
		} else if msg.String() == "t" {
			// Show table view
			sessions, err := getRecentSessions(10)
			if err != nil {
				fmt.Println("Error fetching sessions:", err)
				return m, nil
			}

			columns := []table.Column{
				{Title: "Title", Width: 40},
				{Title: "Duration", Width: 10},
				{Title: "Date", Width: 20},
			}

			rows := []table.Row{}
			for _, s := range sessions {
				title := s.title
				if title == "" {
					title = "(no title)"
				}
				duration := formatDurationAsMMSS(s.duration)
				datetime := s.datetime
				if t, err := time.Parse("2006-01-02T15:04:05Z", s.datetime); err == nil {
					datetime = t.Format("Monday, 2 Jan 06")
				}
				rows = append(rows, table.Row{title, duration, datetime})
			}

			t := table.New(
				table.WithColumns(columns),
				table.WithRows(rows),
				table.WithFocused(false),
				table.WithHeight(len(rows)),
			)

			s := table.DefaultStyles()
			s.Header = s.Header.
				BorderStyle(lipgloss.NormalBorder()).
				BorderForeground(lipgloss.Color(colorGrey)).
				BorderBottom(true).
				Bold(false).
				Foreground(lipgloss.Color(colorGrey)).
				Padding(0, 0)
			s.Cell = s.Cell.
				Padding(0, 0)
			s.Selected = s.Selected.
				Foreground(lipgloss.Color(colorCream)).
				Background(lipgloss.Color(colorGrey)).
				Bold(false)
			t.SetStyles(s)

			m.table = t
			m.mode = tableView
			return m, nil
		}
		// Quit on other keys
		return m, tea.Quit
	}

	// Handle timer view mode (non count-up)
	if msg.String() == "D" {
		// Prompt for title only
		m.promptActive = true
		m.promptType = 1
		m.inputBuffer = m.title
		return m, nil
	} else if msg.String() == "r" {
		// Reset timer
		m.startTime = time.Now().Unix()
		cmd := m.progress.SetPercent(0)
		return m, tea.Batch(tickCmd(), cmd)
	} else if msg.String() == "t" {
		// Show table view of recent sessions
		sessions, err := getRecentSessions(10)
		if err != nil {
			fmt.Println("Error fetching sessions:", err)
			return m, nil
		}

		// Build table with left-aligned columns
		columns := []table.Column{
			{Title: "Title", Width: 40},
			{Title: "Duration", Width: 10},
			{Title: "Date", Width: 20},
		}

		rows := []table.Row{}
		for _, s := range sessions {
			title := s.title
			if title == "" {
				title = "(no title)"
			}
			duration := formatDurationAsMMSS(s.duration)
			// Parse and format the datetime to be more readable
			datetime := s.datetime
			if t, err := time.Parse("2006-01-02T15:04:05Z", s.datetime); err == nil {
				datetime = t.Format("Monday, 2 Jan 06")
			}

			rows = append(rows, table.Row{title, duration, datetime})
		}

		t := table.New(
			table.WithColumns(columns),
			table.WithRows(rows),
			table.WithFocused(false),
			table.WithHeight(len(rows)),
		)

		s := table.DefaultStyles()
		s.Header = s.Header.
			BorderStyle(lipgloss.NormalBorder()).
			BorderForeground(lipgloss.Color(colorGrey)).
			BorderBottom(true).
			Bold(false).
			Foreground(lipgloss.Color(colorGrey)).
			Padding(0, 0)
		s.Cell = s.Cell.
			Padding(0, 0)
		s.Selected = s.Selected.
			Foreground(lipgloss.Color(colorCream)).
			Background(lipgloss.Color(colorGrey)).
			Bold(false)
		t.SetStyles(s)

		m.table = t
		m.mode = tableView
		return m, nil
	} else {
		// Quit if any key is pressed
		return m, tea.Quit
	}
}

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
		pad + helpStyle(helpText)
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
func saveSessionToDB(duration int64, completed bool, title string) error {
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
		"datetime" DATETIME DEFAULT CURRENT_TIMESTAMP,
		"duration" INTEGER,
		"completed" BOOLEAN,
		"title" TEXT
	);`

	_, err = db.Exec(createTableSQL)
	if err != nil {
		return err
	}

	insertSessionSQL := `INSERT INTO sessions (duration, completed, title) VALUES (?, ?, ?)`
	_, err = db.Exec(insertSessionSQL, duration, completed, title)
	if err != nil {
		return err
	}

	return nil
}

// Fetch recent sessions from the database
func getRecentSessions(limit int) ([]session, error) {
	dbPath := os.Getenv("HOME") + sqlite_db_file_path
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, err
	}
	defer db.Close()

	query := `SELECT id, datetime, duration, completed, title FROM sessions ORDER BY datetime DESC LIMIT ?`
	rows, err := db.Query(query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sessions []session
	for rows.Next() {
		var s session
		err := rows.Scan(&s.id, &s.datetime, &s.duration, &s.completed, &s.title)
		if err != nil {
			return nil, err
		}
		sessions = append(sessions, s)
	}

	return sessions, nil
}
