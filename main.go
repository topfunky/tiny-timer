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
	// Initialize database on launch
	if err := initDB(); err != nil {
		fmt.Println("Error initializing database:", err)
		os.Exit(1)
	}

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
