package main

// A simple example that shows how to render an animated progress bar. In this
// example we bump the progress by 25% every two seconds, animating our
// progress bar to its new target state.
//
// It's also possible to render a progress bar in a more static fashion without
// transitions. For details on that approach see the progress-static example.

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/progress"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const (
	padding                  = 2
	maxWidth                 = 80
	defaultDurationInMinutes = 2
	colorGrey                = "#626262"
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
		progress:       progress.New(progress.WithDefaultGradient(), progress.WithoutPercentage()),
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
}

func (m model) Init() tea.Cmd {
	return tickCmd()
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		return m, tea.Quit

	case tea.WindowSizeMsg:
		m.progress.Width = msg.Width - padding*2 - 4
		if m.progress.Width > maxWidth {
			m.progress.Width = maxWidth
		}
		return m, nil

	case tickMsg:
		if m.progress.Percent() == 1.0 {
			return m, tea.Quit
		}

		// Note that you can also use progress.Model.SetPercent to set the
		// percentage value explicitly, too.
		elapsed := time.Now().Unix() - m.startTime
		percentCompleted := float64(elapsed) / float64(m.targetDuration)

		cmd := m.progress.SetPercent(percentCompleted)
		return m, tea.Batch(tickCmd(), cmd)

	// FrameMsg is sent when the progress bar wants to animate itself
	case progress.FrameMsg:
		progressModel, cmd := m.progress.Update(msg)
		m.progress = progressModel.(progress.Model)
		return m, cmd

	default:
		return m, nil
	}
}

func (m model) View() string {
	remaining := m.targetDuration - (time.Now().Unix() - m.startTime)
	if remaining <= 0 {
		// When it completes, display the original duration of the timer
		remaining = m.targetDuration
	}

	pad := strings.Repeat(" ", padding)
	return "\n" +
		pad + m.progress.View() + fmt.Sprintf(" %s\n\n", formatDurationAsMMSS(remaining)) +
		pad + helpStyle("Press any key to quit")
}

func tickCmd() tea.Cmd {
	return tea.Tick(time.Second*1, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func formatDurationAsMMSS(duration int64) string {
	hours := duration / 60
	minutes := duration % 60
	return fmt.Sprintf("%02d:%02d", hours, minutes)
}
