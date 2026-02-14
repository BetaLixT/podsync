package tui

import "github.com/charmbracelet/lipgloss"

var (
	// Colors
	primaryColor   = lipgloss.Color("39")  // Blue
	secondaryColor = lipgloss.Color("245") // Gray
	successColor   = lipgloss.Color("42")  // Green
	warningColor   = lipgloss.Color("214") // Orange
	errorColor     = lipgloss.Color("196") // Red

	// Base styles
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(primaryColor)

	subtitleStyle = lipgloss.NewStyle().
			Foreground(secondaryColor)

	// Status styles
	connectedStyle = lipgloss.NewStyle().
			Foreground(successColor).
			Bold(true)

	disconnectedStyle = lipgloss.NewStyle().
				Foreground(warningColor).
				Bold(true)

	// Episode list styles
	selectedStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("15")).
			Background(primaryColor)

	normalStyle = lipgloss.NewStyle()

	podcastNameStyle = lipgloss.NewStyle().
				Foreground(primaryColor)

	durationStyle = lipgloss.NewStyle().
			Foreground(secondaryColor)

	// Help bar style
	helpStyle = lipgloss.NewStyle().
			Foreground(secondaryColor)

	keyStyle = lipgloss.NewStyle().
			Foreground(primaryColor).
			Bold(true)

	// Box styles
	boxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(primaryColor)

	// Status messages
	statusInfoStyle = lipgloss.NewStyle().
			Foreground(primaryColor)

	statusErrorStyle = lipgloss.NewStyle().
				Foreground(errorColor)

	statusSuccessStyle = lipgloss.NewStyle().
				Foreground(successColor)

	// Syncing indicator
	syncingStyle = lipgloss.NewStyle().
			Foreground(warningColor).
			Bold(true)

	// Tab styles
	activeTabStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("15")).
			Background(primaryColor)

	inactiveTabStyle = lipgloss.NewStyle().
				Foreground(secondaryColor)

	tabGapStyle = lipgloss.NewStyle().
			Foreground(secondaryColor)
)
