# Tomato Timer CLI

A simple command-line application to run a [Pomodoro](https://www.pomodorotechnique.com) timer with an animated progress bar using the [Bubble Tea](https://github.com/charmbracelet/bubbletea) framework.

## Features

- Animated progress bar
- Countdown timer
- Customizable timer duration

## Installation

### Homebrew (macOS/Linux)

```bash
brew tap topfunky/tap
brew install tomato-timer
```

### From Source

1. Clone the repository:

```bash
git clone https://github.com/topfunky/tomato-timer.git
cd tomato-timer
```

2. Build and install the application to your `GOPATH/bin`:

```bash
make install
```

### Download Binary

Download the latest release for your platform from the [releases page](https://github.com/topfunky/tomato-timer/releases).

## Usage

Run the application with the default duration (25 minutes):

```bash
tomato-timer
```

Or specify a custom duration in minutes:

```bash
tomato-timer 5
```

## Dependencies

This project uses the following Go packages:

- [github.com/charmbracelet/bubbles](https://github.com/charmbracelet/bubbles)
- [github.com/charmbracelet/bubbletea](https://github.com/charmbracelet/bubbletea)
- [github.com/charmbracelet/lipgloss](https://github.com/charmbracelet/lipgloss)

## License

This project is currently private and has no license. It includes sample code from the Bubbles project.

[Pomodoro](https://www.pomodorotechnique.com) is a trademark of Francesco Cirillo. The Pomodoro Technique is a time management method developed by Cirillo in the late 1980s.
