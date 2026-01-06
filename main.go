package main

// A TUI timer displaying a countdown in minutes and seconds.
//
// Accepts a single command line argument to set the timer duration in minutes.

import (
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/progress"
	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	// Parse CLI flags
	titleFlag := flag.String("title", "", "Optional title for the timer session")
	countUpFlag := flag.Bool("count-up", false, "Enable count-up mode (logs task time after completion)")
	cleanFlag := flag.Bool("clean", false, "Delete the database and exit")
	debugFlag := flag.Bool("debug", false, "Enable debug logging to debug.log")

	// Pre-process arguments to allow positional argument (minutes) before flags
	// flag.Parse() stops at the first non-flag argument.
	// We check if the first argument is a number and move it to the end if so.
	args := os.Args[1:]
	if len(args) > 0 {
		if _, err := strconv.ParseInt(args[0], 10, 64); err == nil {
			// First arg is a number, move it to the end so flag.Parse() can see the flags
			minutes := args[0]
			newArgs := append(args[1:], minutes)
			os.Args = append([]string{os.Args[0]}, newArgs...)
		}
	}

	flag.Parse()

	// Enable debug logging if flag is set
	// tea.LogToFile configures the standard log package to write to debug.log
	// All log.Printf() calls throughout the codebase will write to this file
	if *debugFlag {
		f, err := tea.LogToFile("debug.log", "debug")
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: Failed to enable debug logging: %v\n", err)
		} else {
			defer f.Close()
		}
	} else {
		// Silence all log output in normal mode
		log.SetOutput(io.Discard)
	}

	if *cleanFlag {
		dbPath, err := getDBPath()
		if err != nil {
			fmt.Println("Error getting database path:", err)
			os.Exit(1)
		}
		if _, err := os.Stat(dbPath); err == nil {
			if err := os.Remove(dbPath); err != nil {
				fmt.Println("Error deleting database:", err)
				os.Exit(1)
			}
			fmt.Println("Database deleted successfully.")
		} else if os.IsNotExist(err) {
			fmt.Println("Database does not exist.")
		} else {
			fmt.Println("Error checking database:", err)
			os.Exit(1)
		}
		os.Exit(0)
	}

	// Initialize database on launch
	if err := initDB(); err != nil {
		fmt.Println("Error initializing database:", err)
		os.Exit(1)
	}

	// Read positional arg for duration, or use default
	var targetDurationInMinutes int64 = defaultDurationInMinutes
	if flag.NArg() > 0 {
		if arg, err := strconv.ParseInt(flag.Arg(0), 10, 64); err == nil && arg > 0 {
			targetDurationInMinutes = arg
		}
	}

	targetDuration := targetDurationInMinutes * 60
	if *countUpFlag && flag.NArg() == 0 {
		targetDuration = defaultCountUpDuration
	}

	keys := keyMap{
		Done: key.NewBinding(
			key.WithKeys("d"),
			key.WithHelp("d", "done"),
		),
		History: key.NewBinding(
			key.WithKeys("h"),
			key.WithHelp("h", "history"),
		),
		Title: key.NewBinding(
			key.WithKeys("t"),
			key.WithHelp("t", "title"),
		),
		Minutes: key.NewBinding(
			key.WithKeys("m"),
			key.WithHelp("m", "minutes"),
		),
		Reset: key.NewBinding(
			key.WithKeys("r"),
			key.WithHelp("r", "reset"),
		),
		Quit: key.NewBinding(
			key.WithKeys("q", "esc", "ctrl+c"),
			key.WithHelp("q/esc", "quit"),
		),
		Confirm: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "confirm"),
		),
		Cancel: key.NewBinding(
			key.WithKeys("esc"),
			key.WithHelp("esc", "cancel"),
		),
		Backspace: key.NewBinding(
			key.WithKeys("backspace"),
			key.WithHelp("backspace", "delete"),
		),
	}

	m := model{
		progress:       progress.New(progress.WithGradient(colorMontezumaGold, colorCream), progress.WithoutPercentage()),
		startTime:      time.Now().Unix(),
		targetDuration: targetDuration,
		title:          *titleFlag,
		countUpMode:    *countUpFlag,
		help:           help.New(),
		keys:           keys,
	}

	// Ensure database connection is closed on exit
	defer closeDBConnection()

	if _, err := tea.NewProgram(m).Run(); err != nil {
		fmt.Println("Oh no!", err)
		os.Exit(1)
	}
}
