# Agent Guidelines

## Project Overview

**Tomato Timer CLI** is a command-line Pomodoro timer written in Go. It features:
- Countdown timer mode (default): tracks time remaining with an animated progress bar
- Count-up mode (`--count-up`): tracks elapsed time and allows logging tasks to SQLite
- Task history: view recent logged tasks with 't' key
- TUI built with Bubble Tea framework (bubbletea + bubbles + lipgloss)
- SQLite persistence for task logging
- Cross-platform support (macOS notifications on darwin)

## Version Control

See [`.ai/agents/VERSION_CONTROL.md`](.ai/agents/VERSION_CONTROL.md) for comprehensive version control guidelines using Jujutsu (jj).

## Essential Commands

### Build & Install
```bash
make build      # Build binary (outputs: tomato-timer)
make install    # Install to GOPATH/bin
make clean      # Clean build cache and test cache
```

### Testing
```bash
make test       # Run all tests (go test ./...)
go test -v      # Run with verbose output
```

### Development
```bash
go run .        # Run directly without building
make list       # Show all available make targets
```

## Code Organization

- **`tomato_timer.go`**: Main file containing all application logic (single-file structure)
  - Model struct and state management
  - Event handlers (Update function with separate update* handlers)
  - View rendering logic
  - Database operations
  - Helper functions (formatting, notifications)
- **`tomato_timer_test.go`**: Comprehensive test suite
  - Follows Go testing conventions (`*_test.go` suffix)
  - Uses `testify` assertions
  - Tests organized by feature with clear names

## Architecture & State Management

Uses Bubble Tea MVC pattern:
- **Model**: `type model struct` holds all state (progress, timer, UI mode, input buffer, etc.)
- **Update**: `func (m model) Update(msg tea.Msg)` handles all events and state changes
  - Dispatches to specialized handlers: `updateKey()`, `updatePercent()`, `updateWindowSize()`, `handlePromptInput()`
  - Returns new model and command (e.g., `tickCmd()`)
- **View**: `func (m model) View()` renders UI based on current state

### Key State Fields
```go
type model struct {
  progress       progress.Model      // Animated progress bar
  startTime      int64               // Unix timestamp (never changes, used for all time calculations)
  targetDuration int64               // Countdown target or count-up max (in seconds)
  title          string              // Task title (displayed at top, set by prompt or --title flag)
  mode           viewMode            // timerView or tableView
  countUpMode    bool                // true = count elapsed time, false = countdown
  inputBuffer    string              // Text being typed in prompt
  promptActive   bool                // Whether prompt is currently visible
  promptType     int                 // 0 = log+reset (d), 1 = title-only (D)
  table          table.Model         // Recent sessions display
}
```

### Critical: Time Calculation Pattern

**Always use absolute time, never cached progress**:
```go
elapsed := time.Now().Unix() - m.startTime  // ✓ Correct - always current
remaining := m.targetDuration - elapsed      // ✓ Correct - calculated fresh
```

This is critical because:
- Timer can be suspended (Ctrl-Z) and resumed later
- startTime is set once and never changes
- Progress percentage is cached for UI animation but cannot be relied upon for logic

## Code Patterns & Conventions

### File Structure
- Single monolithic `.go` file with all logic
- No separate packages or subdirectories
- All types and functions are at package level

### Naming Conventions
- **Types**: PascalCase (`viewMode`, `model`, `session`, `promptMsg`, `tickMsg`)
- **Constants**: UPPER_SNAKE_CASE with descriptive names (`defaultDurationInMinutes`, `defaultCountUpDuration`, `sqlite_db_file_path`)
- **Functions**: camelCase, descriptive names indicating behavior (`updatePercent`, `handlePromptInput`, `getRecentSessions`)
- **UI strings**: In constants at top (color codes, default values)

### Go Idioms Used
- Error handling: `if err := ...; err != nil { return err }` pattern
- Defer for cleanup: `defer db.Close()`, `defer rows.Close()`
- Interface satisfaction: `model` implements tea.Model implicitly
- Return error as last value: standard Go convention
- Unix timestamps (`int64`) for all time values to avoid timezone issues

### Color & Styling
- Uses lipgloss for terminal styling
- Three main colors defined as constants:
  - `colorGrey = "#626262"` - help text, headers
  - `colorCream = "#fefdbc"` - progress/highlights
  - `colorMontezumaGold = "#f0c442"` - progress bar gradient
- Help text always uses `helpStyle()` which is `lipgloss.NewStyle().Foreground(...).Render`

## Testing Approach

### Test Patterns
- Each feature has multiple test cases with descriptive names
- Test-driven approach: create model with specific state, call Update, assert result
- Uses `testify/assert` for readable assertions
- **Database tests**: Create temp SQLite DB in temp dir for isolation
- **Time-based tests**: Use `testing.Testing()` to skip platform-specific code (e.g., macOS notifications)

### Test Examples
```go
// Setup model with specific state
m := model{
  progress:       progress.New(...),
  startTime:      time.Now().Unix(),
  targetDuration: 3600,
  countUpMode:    true,
}

// Trigger action
newModel, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
modelTyped := newModel.(model)

// Assert outcomes
assert.Equal(t, "", modelTyped.title, "Title should be blank after logging")
assert.NoError(t, err)
```

### Important: Test Always for User-Facing Changes
- Any change to key handling must include key binding tests
- Any change to UI rendering must verify View() output
- Any database changes must verify save/read operations
- All tests must pass before considering work complete: `make test`

## Database Schema

SQLite database stored at `~/.config/tomato-timer/tomato-timer.db`:
```sql
CREATE TABLE sessions (
  id        INTEGER PRIMARY KEY AUTOINCREMENT,
  datetime  DATETIME DEFAULT CURRENT_TIMESTAMP,
  duration  INTEGER,      -- elapsed seconds
  completed BOOLEAN,
  title     TEXT           -- task title (can be empty string)
)
```

### Database Operations
- **Save**: `saveSessionToDB(duration, completed, title)` - creates table if missing
- **Read**: `getRecentSessions(limit)` - returns sorted by datetime DESC
- Accessed via SQLite with `database/sql` and `mattn/go-sqlite3` driver

## User Interaction Modes

### Countdown Mode (default)
- **Start**: `tomato-timer [minutes]` (default 25)
- **Keys**: 
  - 'r' = reset timer
  - 't' = show recent sessions table
  - Ctrl-Z = suspend/resume
  - Any other key = quit
- **On completion**: Saves session to DB with elapsed duration

### Count-Up Mode (`--count-up` flag)
- **Start**: `tomato-timer --count-up [--title "Task"]`
- **Default duration**: 1 hour (3600s) - just for progress bar scaling
- **Keys**:
  - 'd' = prompt for title, log session, reset timer to 0:00
  - 'D' = prompt for title only (continue counting without logging)
  - 'r' = reset timer to 0:00 without logging
  - 't' = show recent sessions table
  - Ctrl-Z = suspend/resume
  - Any other key = quit
- **Prompt handling**: Text input with backspace, Enter to confirm, Esc to cancel

## Important Gotchas & Non-Obvious Patterns

### 1. Time Calculation and Suspension
- `startTime` is **never** modified except at initialization
- When resuming from Ctrl-Z suspend, elapsed time is recalculated: `time.Now().Unix() - m.startTime`
- This automatically handles gaps from suspension without special logic
- **Gotcha**: Don't cache elapsed time - always recalculate in handlers

### 2. Modal vs Mutable Patterns
- All handlers return `(tea.Model, tea.Cmd)` - old model is discarded
- Updates are "modal" functional style, not mutating: `m.title = ""; return m, cmd`
- **Gotcha**: Don't modify m, then use old references - always work with returned model

### 3. Platform-Specific Notification Code
- Notifications only work on macOS (darwin)
- Code checks `runtime.GOOS != "darwin"` to skip on other platforms
- Tests check `testing.Testing()` to avoid trying notifications during test runs
- **Gotcha**: Notification failures are logged but don't crash the app

### 4. Prompt Type Encoding
- `promptType: 0` = task logging (d key) - saves to DB and resets
- `promptType: 1` = title-only (D key) - no DB save, timer continues
- Both activate prompt UI, but `handlePromptInput` behaves differently
- **Gotcha**: Easy to mix up which type does what - check code carefully

### 5. View Mode State
- `mode: timerView` (default) = shows timer and progress bar
- `mode: tableView` = shows recent sessions table (read-only, any key exits)
- Only 'r', 't', and Ctrl-Z are handled in timer view
- All other keys quit the app (design choice: keep app lightweight)
- **Gotcha**: Adding new features shouldn't break this "few keys" philosophy

### 6. Database Path
- Stored at `os.Getenv("HOME")/.config/tomato-timer/tomato-timer.db`
- Directory is created automatically if missing: `os.MkdirAll(..., os.ModePerm)`
- **Gotcha**: Tests use temp db path (set in test via os.Setenv) to avoid polluting user DB

### 7. Progress Bar in Count-Up Mode
- Progress bar still shows 0-100%, but capped at 1 hour by default
- Elapsed time displayed as MM:SS regardless of mode
- **Gotcha**: Progress bar will show "full" after 1 hour in count-up mode (visual limitation)

## Code Quality & Review Checklist

Before submitting changes:
- [ ] All tests pass: `make test`
- [ ] Code builds: `make build`
- [ ] Changes follow Conventional Commits format
- [ ] New features have corresponding tests
- [ ] Database changes verified (save/read operations tested)
- [ ] UI changes verified visually (View() logic tested)
- [ ] No hardcoded paths (use constants like `sqlite_db_file_path`)
- [ ] Time calculations use `time.Now().Unix() - m.startTime` pattern
- [ ] Error handling present and logged (don't silently fail)
- [ ] JOURNAL.md updated for significant work

## Dependencies

### Core TUI Framework
- `github.com/charmbracelet/bubbletea` v1.2.4+ - Event loop framework
- `github.com/charmbracelet/bubbles` v0.20.0+ - UI components (progress, table)
- `github.com/charmbracelet/lipgloss` v1.0.0+ - Terminal styling

### Database
- `github.com/mattn/go-sqlite3` v1.14.24+ - SQLite driver

### Testing
- `github.com/stretchr/testify` v1.10.0+ - Assertions and test helpers

### Language
- Go 1.23.4+ (check go.mod for exact version)

## Documentation

- **JOURNAL.md**: Historical record of significant work (debugging, features, fixes)
  - Use "## YYYY-MM-DD HH:MM - Title" headers
  - Document problem, solution, and testing
  - Helpful for understanding why decisions were made
- **README.md**: User-facing documentation (installation, usage, features)
- **AGENTS.md** (this file): Developer guidelines for working in the codebase

## Debugging Tips

### Logging
- Print errors with `fmt.Println()` - they appear in terminal
- Use `fmt.Printf()` for debugging during development (remove before committing)
- **Gotcha**: Error handling might hide issues - always check returned errors

### Testing Specific Features
```bash
# Run single test
go test -run TestCountUpModePromptLogAndReset -v

# Run with timeout and race detector
go test -race -timeout 30s -v
```

### Common Issues
- **Panic on nil**: Always check error returns and nil assertions
- **Timer seems wrong**: Verify `startTime` isn't being modified
- **Database errors**: Check directory exists and file permissions
- **UI not updating**: Verify `cmd := m.progress.SetPercent(...)` or `tea.Batch()` returns
