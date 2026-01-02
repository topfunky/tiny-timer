# Development Journal

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
