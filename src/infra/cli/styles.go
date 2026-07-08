package cli

import "github.com/charmbracelet/lipgloss"

var (
	Title = lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#C678DD")).
		Align(lipgloss.Center)

	Header = lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#61AFEF"))

	Label = lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#AAAAAA"))

	Value = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#98C379"))

	Error = lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#E06C75"))

	Warning = lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#E5C07B"))

	Footer = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#888888"))

	Panel = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#C678DD")).
		Padding(1, 2).
		Width(72)
)
