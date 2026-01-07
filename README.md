# Tiny Timer CLI

A simple command-line application to run a [Pomodoro](https://www.pomodorotechnique.com) timer with an animated progress bar using the [Bubble Tea](https://github.com/charmbracelet/bubbletea) framework.

## Features

- Animated progress bar
- Countdown timer
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

## Dependencies

This project uses the following Go packages:

- [github.com/charmbracelet/bubbles](https://github.com/charmbracelet/bubbles)
- [github.com/charmbracelet/bubbletea](https://github.com/charmbracelet/bubbletea)
- [github.com/charmbracelet/lipgloss](https://github.com/charmbracelet/lipgloss)

## License

This project is licensed under the MIT License. See [LICENSE](LICENSE) for details.

It includes sample code from the Bubbles project.

[Pomodoro](https://www.pomodorotechnique.com) is a trademark of Francesco Cirillo. The Pomodoro Technique is a time management method developed by Cirillo in the late 1980s.
