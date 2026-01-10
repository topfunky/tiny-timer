// Package status implements the StatusCmp component that displays a temporary alert.
//
// Modified slightly from https://github.com/charmbracelet/crush/blob/main/internal/tui/components/core/status/status.go
package status

import (
	"time"

	"github.com/charmbracelet/bubbles/help"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

type InfoType int

const (
	InfoTypeInfo InfoType = iota
	InfoTypeSuccess
	InfoTypeWarn
	InfoTypeError
	InfoTypeUpdate
)

type InfoMsg struct {
	Type InfoType
	Msg  string
	TTL  time.Duration
}

type ClearStatusMsg struct{}

type Theme struct {
	Name      string
	Green     lipgloss.Color
	GreenDark lipgloss.Color
	BgSubtle  lipgloss.Color
	Red       lipgloss.Color
	Error     lipgloss.Color
	White     lipgloss.Color
	Yellow    lipgloss.Color
	Warning   lipgloss.Color
	BgOverlay lipgloss.Color
	Primary   lipgloss.Color
	Border    lipgloss.Color
}

func NewTheme() *Theme {
	return &Theme{
		Name:      "tinytimer",
		Green:     lipgloss.Color("#00d787"), // Julep
		GreenDark: lipgloss.Color("#00a86b"), // Guac
		BgSubtle:  lipgloss.Color("#313244"), // Charcoal
		Red:       lipgloss.Color("#ff5555"), // Coral
		Error:     lipgloss.Color("#ff6b6b"), // Sriracha
		White:     lipgloss.Color("#ffffff"), // Butter
		Yellow:    lipgloss.Color("#ffff00"), // Mustard
		Warning:   lipgloss.Color("#ffd700"), // Zest
		BgOverlay: lipgloss.Color("#1e1e2e"), // Iron
		Primary:   lipgloss.Color("#6c5ce7"), // Charple
		Border:    lipgloss.Color("#313244"), // Charcoal
	}
}

type Styles struct {
	Base lipgloss.Style
	Help help.Styles
}

func (t *Theme) S() *Styles {
	base := lipgloss.NewStyle()
	return &Styles{
		Base: base,
		Help: help.Styles{
			ShortKey:       base.Foreground(lipgloss.Color("#a0a0a0")),
			ShortDesc:      base.Foreground(lipgloss.Color("#626262")),
			ShortSeparator: base.Foreground(lipgloss.Color("#313244")),
		},
	}
}

type StatusCmp struct {
	info       InfoMsg
	width      int
	messageTTL time.Duration
	help       help.Model
	keyMap     help.KeyMap
}

func NewStatusCmp() *StatusCmp {
	t := NewTheme()
	h := help.New()
	h.Styles = t.S().Help
	return &StatusCmp{
		messageTTL: 3 * time.Second,
		help:       h,
	}
}

func (m *StatusCmp) SetKeyMap(keyMap help.KeyMap) {
	m.keyMap = keyMap
}

func (m *StatusCmp) clearMessageCmd(ttl time.Duration) tea.Cmd {
	return tea.Tick(ttl, func(time.Time) tea.Msg {
		return ClearStatusMsg{}
	})
}

func (m *StatusCmp) Update(msg tea.Msg) (*StatusCmp, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.help.Width = msg.Width - 2
		return m, nil

	case InfoMsg:
		m.info = msg
		ttl := msg.TTL
		if ttl == 0 {
			ttl = m.messageTTL
		}
		return m, m.clearMessageCmd(ttl)

	case ClearStatusMsg:
		m.info = InfoMsg{}
		return m, nil
	}
	return m, nil
}

func (m *StatusCmp) View() string {
	if m.info.Msg != "" {
		return m.infoMsg()
	}
	return NewTheme().S().Base.Padding(0, 1, 1, 1).Render(m.help.View(m.keyMap))
}

func (m *StatusCmp) infoMsg() string {
	t := NewTheme()

	// Determine styling based on info type
	var infoTypeLabel string
	var infoTypeStyle lipgloss.Style
	var messageStyle lipgloss.Style
	var messageFg lipgloss.Color

	switch m.info.Type {
	case InfoTypeError:
		infoTypeLabel = "ERROR"
		infoTypeStyle = t.S().Base.Background(t.Red).Padding(0, 1)
		messageStyle = t.S().Base.Background(t.Error).Padding(0, 1)
		messageFg = t.White
	case InfoTypeWarn:
		infoTypeLabel = "WARNING"
		infoTypeStyle = t.S().Base.Foreground(t.BgOverlay).Background(t.Yellow).Padding(0, 1)
		messageStyle = t.S().Base.Foreground(t.BgOverlay).Background(t.Warning).Padding(0, 1)
	default:
		if m.info.Type == InfoTypeUpdate {
			infoTypeLabel = "HEY!"
		} else {
			infoTypeLabel = "OKAY!"
		}
		infoTypeStyle = t.S().Base.Foreground(t.BgSubtle).Background(t.Green).Padding(0, 1).Bold(true)
		messageStyle = t.S().Base.Background(t.GreenDark).Padding(0, 1)
		messageFg = t.BgOverlay
	}

	// Render info type label
	infoType := infoTypeStyle.Render(infoTypeLabel)

	// Calculate available width and truncate message
	widthLeft := m.width - (lipgloss.Width(infoType) + 2)
	info := ansi.Truncate(m.info.Msg, widthLeft, "…")

	// Render message with calculated width
	if messageFg != "" {
		messageStyle = messageStyle.Foreground(messageFg)
	}
	message := messageStyle.Width(widthLeft + 2).Render(info)

	return ansi.Truncate(infoType+message, m.width, "…")
}
