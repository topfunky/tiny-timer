# Development Journal

## 2026-01-02 17:15 - Fixed Database Initialization on First Run

### Problem

On first run, pressing 't' to view task history would fail with an error because the database file and sessions table didn't exist yet. The app only created the database when saving a session, not on launch. This meant users couldn't view the (empty) task list on first run.

### The Fix

Added database initialization on application startup:

**1. Created `initDB()` function:**
```go
func initDB() error {
    dbPath, err := getDBPath()
    if err != nil {
        return err
    }
    dbDir := filepath.Dir(dbPath)
    if err := os.MkdirAll(dbDir, os.ModePerm); err != nil {
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
    return err
}
```

**2. Called `initDB()` in `main()`:**
- Runs before parsing flags or creating the model
- Exits with error message if initialization fails
- Ensures database is ready before any user interaction

### Benefits

- **Better UX**: Users can press 't' on first run without errors
- **Consistent state**: Database always exists when app is running
- **Fail-fast**: Database errors are caught at startup, not during usage
- **Idempotent**: Safe to call multiple times (uses `CREATE TABLE IF NOT EXISTS`)

### Testing

Added 3 new tests:
1. `TestInitDB` - Verifies database and table creation
2. `TestInitDBIdempotent` - Ensures safe to call multiple times
3. `TestGetRecentSessionsOnEmptyDB` - Verifies querying empty database works

Manual testing confirmed:
- Database directory created on first launch
- Database file created with correct schema
- Pressing 't' on first run shows empty task list (no errors)

All 31 tests pass successfully. Build succeeds with no errors.

## 2026-01-02 15:45 - Fixed Database Path Resolution for Cross-Platform Compatibility

### Problem

The database path was constructed using `os.Getenv("HOME")` which had several issues:
1. Could return an empty string if `$HOME` environment variable is not set
2. Not cross-platform (doesn't work properly on Windows)
3. No error handling if home directory can't be determined
4. Would silently create invalid paths like `/.config/tomato-timer/tomato-timer.db` (in root directory)

### The Fix

Replaced `os.Getenv("HOME")` with Go's standard `os.UserHomeDir()` function, which:
- Returns a proper error if the home directory can't be determined
- Works cross-platform (Linux, macOS, Windows, Plan 9)
- Uses platform-specific methods to find the home directory
- Follows Go best practices

### Implementation Details

**1. Created `getDBPath()` helper function:**
```go
func getDBPath() (string, error) {
    homeDir, err := os.UserHomeDir()
    if err != nil {
        return "", err
    }
    return filepath.Join(homeDir, ".config", "tomato-timer", "tomato-timer.db"), nil
}
```

**2. Updated database functions:**
- `saveSessionToDB()` now calls `getDBPath()` and handles errors properly
- `getRecentSessions()` now calls `getDBPath()` and handles errors properly
- Used `filepath.Join()` for cross-platform path construction
- Used `filepath.Dir()` to get the directory for `MkdirAll()`

**3. Cleaned up constants:**
- Removed obsolete `sqliteDBFilePath` constant from `constants.go`
- Centralized path logic in the `getDBPath()` function

**4. Updated test infrastructure:**
- Created `setupTestDB()` helper function for consistent test database setup
- Updated all 10 database-related tests to use the new helper
- Tests properly restore original `$HOME` after completion

### Benefits

- **More robust**: Proper error handling when home directory can't be determined
- **Cross-platform**: Works on Windows (`%USERPROFILE%`), macOS/Linux (`$HOME`), and Plan 9 (`$home`)
- **Cleaner code**: Centralized database path logic in one function
- **Better testing**: Simplified test setup with reusable helper function
- **Follows Go best practices**: Uses standard library functions designed for this purpose

### Testing

All 28 tests pass successfully:
- Database save/retrieve operations
- Schema validation
- Table view rendering
- Timer functionality
- Count-up mode features

Build succeeds with no errors or warnings.

## 2026-01-07 14:30 - Code Refactoring: Split Monolithic File into Separate Modules

### Overview

Refactored the codebase from a single monolithic `tomato_timer.go` file (562 lines) into a modular structure with separate files for better readability and discoverability.

### File Structure

The codebase is now organized into 7 focused files:

1. **`constants.go`** - All constants (padding, colors, durations, DB path) and styling helpers (`renderHelpText`, `helpStyle`)
2. **`model.go`** - Model struct, types (`tickMsg`, `promptMsg`, `viewMode`, `session`), and initialization (`Init`, `tickCmd`)
3. **`utils.go`** - Helper functions (`formatDurationAsMMSS`, `sendNotification`)
4. **`database.go`** - Database operations (`saveSessionToDB`, `getRecentSessions`, `buildTableView`)
5. **`handlers.go`** - All event handlers (`Update`, `updatePercent`, `updateKey`, and specialized key handlers)
6. **`view.go`** - UI rendering logic (`View` method)
7. **`main.go`** - Entry point and CLI flag parsing

### Benefits

- **Improved discoverability**: Each file has a clear, single responsibility making it easier to find specific functionality
- **Better organization**: Related code is grouped together logically
- **Easier maintenance**: Changes to specific features (e.g., database operations) are isolated to their respective files
- **Preserved functionality**: All existing behavior maintained - no functional changes

### Testing

All 28 existing tests pass successfully after refactoring:
- Database operations tests
- View rendering tests
- Key handling tests
- Count-up mode tests
- Timer functionality tests

Build succeeds with no errors or warnings. The refactoring maintains 100% backward compatibility.

## 2026-01-02 10:15 - Count-Up Mode with Task Logging

### Feature Overview

Added a new count-up mode (`--count-up` flag) that tracks elapsed time instead of counting down. This mode starts with a 1-hour default duration and allows logging of tasks as they complete.

### Key Features

1. **Command-line flag**: `--count-up` enables the mode (default: off)
2. **Default duration**: 1 hour (3600 seconds) for count-up sessions
3. **Task logging**: Press 'd' to log current session and start a new 1-hour count
   - Prompts user for task title
   - Saves title and elapsed time to SQLite database
   - Resets timer for next session
4. **Title-only mode**: Press 'D' to change the current task title without logging
   - Prompts user for new title
   - Updates display title
   - Continues current elapsed time (no reset)
5. **Updated help text**: Context-sensitive help for count-up vs countdown mode
6. **Database persistence**: All logged tasks stored in SQLite with timestamps

### Implementation Details

- Added `countUpMode` field to model to track mode state
- Added `promptMsg` message type for title input handling
- Added prompt input UI with text buffer and backspace support
- Modified `updatePercent()` to continuously show elapsed time in count-up mode
- Modified `View()` to display elapsed time and appropriate help text based on mode
- Added `handlePromptInput()` to process task logging or title-only updates

### Testing

Added 9 comprehensive tests covering:
- Count-up mode initialization with correct defaults
- Elapsed time tracking
- Key bindings (d and D)
- Prompt activation and input handling
- Session logging to database
- Title-only mode without logging
- UI display of help text and elapsed time

All existing tests continue to pass. Build succeeds with no errors.

## 2026-01-01 19:30 - Fixed Ctrl-Z Suspend/Resume Support

### Problem

The app was dying when pressing Ctrl-Z instead of suspending to the background. The issue was that the `updateKey()` function had a catch-all handler that quit the app on any key press except 'r' and 't':

```go
} else {
    // Quit if any key is pressed
    return m, tea.Quit
}
```

When Ctrl-Z was pressed, bubbletea intercepted it as a key event before the OS could handle it, causing the app to quit instead of suspending.

### The Fix

Added explicit handling for Ctrl-Z **before** the generic "quit on any key" logic:

```go
// Allow Ctrl-Z to suspend the process in timer view
if msg.Type == tea.KeyCtrlZ {
    return m, tea.Suspend
}
```

This tells bubbletea to suspend the process properly, allowing normal terminal suspend/resume behavior:
1. Press **Ctrl-Z** to suspend the app
2. Run **`fg`** to resume it
3. The timer continues accurately from where it left off (using absolute time tracking)

### Key Implementation Details

- The timer already used absolute time calculations (`time.Now().Unix() - m.startTime`), so no changes were needed to the timing logic
- Changed the completion check in `updatePercent()` to use calculated elapsed time instead of cached progress percentage, ensuring immediate detection when resuming after completion
- Added Ctrl-Z handling in both timer view and table view modes

### Testing

Added comprehensive tests to verify:
- Ctrl-Z suspends in timer view
- Ctrl-Z suspends in table view (doesn't exit the table)
- Other keys still quit as expected
- Timer continues accurately after resume
- Completion is detected and saved to DB even if you resume after the timer finished

All tests pass successfully.
