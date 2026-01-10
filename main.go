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

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/progress"
	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	title, countUp, clean, debug := parseFlags()

	configureLogging(debug)

	if clean {
		handleCleanFlag()
		return
	}

	if err := initDB(); err != nil {
		fmt.Println("Error initializing database:", err)
		os.Exit(1)
	}

	defer closeDBConnection()

	targetDuration := calculateTargetDuration(countUp)
	keys := createKeyBindings()
	m := createModel(title, countUp, targetDuration, keys)

	if _, err := tea.NewProgram(m).Run(); err != nil {
		fmt.Println("Oh no!", err)
		os.Exit(1)
	}
}

func parseFlags() (title string, countUp bool, clean bool, debug bool) {
	titleFlag := flag.String("title", "", "Optional title for the timer session")
	countUpFlag := flag.Bool("count-up", false, "Enable count-up mode (logs task time after completion)")
	cleanFlag := flag.Bool("clean", false, "Delete the database and exit")
	debugFlag := flag.Bool("debug", false, "Enable debug logging to debug.log")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [minutes] [flags]\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Positional arguments:\n")
		fmt.Fprintf(os.Stderr, "  minutes\n")
		fmt.Fprintf(os.Stderr, "    \tDuration in minutes for the timer (default: 25)\n\n")
		fmt.Fprintf(os.Stderr, "Flags:\n")
		flag.PrintDefaults()
	}

	preprocessArgs()
	flag.Parse()

	return *titleFlag, *countUpFlag, *cleanFlag, *debugFlag
}

func preprocessArgs() {
	args := os.Args[1:]
	if len(args) > 0 {
		if _, err := strconv.ParseInt(args[0], 10, 64); err == nil {
			minutes := args[0]
			newArgs := append(args[1:], minutes)
			os.Args = append([]string{os.Args[0]}, newArgs...)
		}
	}
}

func configureLogging(debug bool) {
	if debug {
		f, err := tea.LogToFile("debug.log", "debug")
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: Failed to enable debug logging: %v\n", err)
		} else {
			defer f.Close()
		}
	} else {
		log.SetOutput(io.Discard)
	}
}

func handleCleanFlag() {
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
}

func calculateTargetDuration(countUp bool) int64 {
	var targetDurationInMinutes int64 = defaultDurationInMinutes
	if flag.NArg() > 0 {
		if arg, err := strconv.ParseInt(flag.Arg(0), 10, 64); err == nil && arg > 0 {
			targetDurationInMinutes = arg
		}
	}

	targetDuration := targetDurationInMinutes * 60
	if countUp && flag.NArg() == 0 {
		targetDuration = defaultCountUpDuration
	}

	return targetDuration
}

func createKeyBindings() keyMap {
	return keyMap{
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
}

func createModel(title string, countUp bool, targetDuration int64, keys keyMap) model {
	prog := progress.New(progress.WithGradient(colorMontezumaGold, colorCream), progress.WithoutPercentage())
	if countUp {
		prog.SetPercent(0)
	} else {
		prog.SetPercent(1.0)
	}

	return model{
		progress:       prog,
		startTime:      time.Now().Unix(),
		targetDuration: targetDuration,
		title:          title,
		countUpMode:    countUp,
		help:           newHelpModel(),
		keys:           keys,
	}
}
