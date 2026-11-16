package tui

import "github.com/charmbracelet/lipgloss"

var (
	// FireOS-inspired color palette
	colorOrange = lipgloss.Color("#FF9900")
	colorCyan   = lipgloss.Color("#00A8E1")
	colorGreen  = lipgloss.Color("#2ECC71")
	colorYellow = lipgloss.Color("#F1C40F")
	colorRed    = lipgloss.Color("#E74C3C")
	colorMuted  = lipgloss.Color("#7F8C8D")
	colorWhite  = lipgloss.Color("#ECF0F1")
	colorDarkBg = lipgloss.Color("#1E272E")

	// Frame & Header styles
	boxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorOrange).
			Padding(1, 2)

	headerStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorOrange)

	subHeaderStyle = lipgloss.NewStyle().
			Foreground(colorCyan).
			Bold(true)

	categoryHeaderStyle = lipgloss.NewStyle().
				Foreground(colorOrange).
				Bold(true).
				MarginTop(1)

	// Menu item styles
	cursorStyle = lipgloss.NewStyle().
			Foreground(colorOrange).
			Bold(true)

	selectedItemStyle = lipgloss.NewStyle().
				Foreground(colorWhite).
				Bold(true)

	normalItemStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#BDC3C7"))

	checkedBoxStyle = lipgloss.NewStyle().
			Foreground(colorGreen).
			Bold(true)

	uncheckedBoxStyle = lipgloss.NewStyle().
				Foreground(colorMuted)

	descriptionBoxStyle = lipgloss.NewStyle().
				Foreground(colorCyan).
				Italic(true).
				MarginTop(1)

	// Execute button styles
	executeBtnIdleStyle = lipgloss.NewStyle().
				Foreground(colorWhite).
				Background(lipgloss.Color("#34495E")).
				Padding(0, 2).
				Bold(true)

	executeBtnActiveStyle = lipgloss.NewStyle().
				Foreground(colorDarkBg).
				Background(colorOrange).
				Padding(0, 2).
				Bold(true)

	// Status badge styles
	badgeOKStyle = lipgloss.NewStyle().
			Foreground(colorGreen).
			Bold(true)

	badgeSkipStyle = lipgloss.NewStyle().
			Foreground(colorYellow).
			Bold(true)

	badgeErrStyle = lipgloss.NewStyle().
			Foreground(colorRed).
			Bold(true)

	footerHelpStyle = lipgloss.NewStyle().
			Foreground(colorMuted).
			MarginTop(1)
)
