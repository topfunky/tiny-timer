package main

// A TUI timer displaying a countdown in minutes and seconds.
//
// Accepts a single command line argument to set the timer duration in minutes.

import (
	"flag"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/charmbracelet/bubbles/progress"
	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	// Parse CLI flags
	titleFlag := flag.String("title", "", "Optional title for the timer session")
	countUpFlag := flag.Bool("count-up", false, "Enable count-up mode (logs task time after completion)")
	cleanFlag := flag.Bool("clean", false, "Delete the database and exit")
	debugFlag := flag.Bool("debug", false, "Enable debug logging to tomato-timer-debug.log")
	flag.Parse()

	// Enable debug logging if flag is set
	if *debugFlag {
		if err := setDebugEnabled(true); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: Failed to enable debug logging: %v\n", err)
		} else {
			defer closeDebugLog()
		}
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

	// Ensure database connection is closed on exit
	defer closeDBConnection()

	if _, err := tea.NewProgram(m).Run(); err != nil {
		fmt.Println("Oh no!", err)
		os.Exit(1)
	}
}
