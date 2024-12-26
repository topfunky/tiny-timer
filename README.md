# Pomodoro CLI

A simple command-line application to run a [Pomodoro](https://www.pomodorotechnique.com) timer with an animated progress bar using the [Bubble Tea](https://github.com/charmbracelet/bubbletea) framework.

## Features

- Animated progress bar
- Customizable timer duration

## Installation

1. Clone the repository:

```bash
git clone https://github.com/yourusername/pomodoro-cli.git
cd pomodoro-cli
```

1. Build the application:

```bash
go build -o pomodoro
go install
```

## Usage

Run the application with the default duration (25 minutes):

```bash
pomodoro
```

Or specify a custom duration in minutes:

```bash
pomodoro 5
```

## Dependencies

This project uses the following Go packages:

- [github.com/charmbracelet/bubbles](https://github.com/charmbracelet/bubbles)
- [github.com/charmbracelet/bubbletea](https://github.com/charmbracelet/bubbletea)
- [github.com/charmbracelet/lipgloss](https://github.com/charmbracelet/lipgloss)

## License

This project is currently private and has no license. It includes sample code from the Bubbles project.
