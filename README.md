# Tiny Timer CLI

A simple command-line application to run a [Pomodoro](https://www.pomodorotechnique.com) timer with an animated progress bar using the [Bubble Tea](https://github.com/charmbracelet/bubbletea) framework.

![Tiny Timer application demo](https://vhs.charm.sh/vhs-Bc93HZoHL5g7S16LmkgB9.gif)

## Features

- Animated progress bar
- Countdown timer (defaults to `25` minutes)
- Customizable timer duration

## Installation

### Homebrew (macOS/Linux)

Install from the topfunky tap:

```bash
brew install topfunky/tap/tiny-timer
```

### From Source

1. Clone the repository:

```bash
git clone https://github.com/topfunky/tiny-timer.git
cd tiny-timer
```

2. Build and install the application to your `GOPATH/bin`:

```bash
make install
```

## Usage

Run the application with the default duration (25 minutes):

```bash
tiny-timer
```

Or specify a custom duration in minutes:

```bash
tiny-timer 5
```

## CLI Flags

The following command-line flags are available:

### Positional Arguments

- **`[minutes]`** - Optional duration in minutes for the timer. If not specified, defaults to 25 minutes. Can be placed before or after flags.

  ```bash
  tiny-timer 5
  tiny-timer -count-up 10
  ```

### Flags

- **`-title <string>`** - Set an optional title for the timer session. Useful for labeling your work sessions.

  ```bash
  tiny-timer -title "Writing documentation"
  tiny-timer 30 -title "Code review"
  ```

- **`-count-up`** - Enable count-up mode instead of countdown. In this mode, the timer tracks elapsed time and allows you to log tasks to the SQLite database. Default duration is 1 hour (for progress bar scaling).

  ```bash
  tiny-timer -count-up
  tiny-timer -count-up -title "Project planning"
  ```

- **`-clean`** - Delete the SQLite database and exit. Useful for resetting your session history.

  ```bash
  tiny-timer -clean
  ```

- **`-debug`** - Enable debug logging to `debug.log` file. All log output will be written to this file for troubleshooting.

  ```bash
  tiny-timer -debug
  ```

## Development

### Testing Releases with GoReleaser

To test the release process without creating an actual release, use the make target:

```bash
make release-dry-run
```

Or run GoReleaser directly:

```bash
goreleaser release --snapshot --skip-publish --clean
```

This will:
- Build binaries for all configured platforms (Linux, macOS, Windows)
- Create archives and checksums
- Generate the Homebrew formula
- Skip publishing to GitHub
- Clean up previous build artifacts

For a full dry-run that also validates publishing (without actually publishing):

```bash
goreleaser release --snapshot --skip-publish
```

### Record demo video with VHS

```nushell
docker run --rm -v ($env.PWD):/vhs ghcr.io/charmbracelet/vhs vhs/basic.tape
```

## License

This project is licensed under the MIT License. See [LICENSE](LICENSE) for details.

It includes sample code from the Bubbles project.

[Pomodoro](https://www.pomodorotechnique.com) is a trademark of Francesco Cirillo. The Pomodoro Technique is a time management method developed by Cirillo in the late 1980s.
