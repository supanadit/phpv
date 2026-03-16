package ui

import (
	"github.com/charmbracelet/lipgloss"
)

var (
	PrimaryColor   = lipgloss.Color("#6C5CE7")
	SecondaryColor = lipgloss.Color("#A29BFE")
	SuccessColor   = lipgloss.Color("#00B894")
	WarningColor   = lipgloss.Color("#FDCB6E")
	ErrorColor     = lipgloss.Color("#E17055")
	InfoColor      = lipgloss.Color("#74B9FF")

	DimColor    = lipgloss.Color("#636E72")
	BrightColor = lipgloss.Color("#DFE6E9")

	HeaderStyle = lipgloss.NewStyle().
			Foreground(PrimaryColor).
			Bold(true).
			Padding(0, 1)

	SubheaderStyle = lipgloss.NewStyle().
			Foreground(SecondaryColor).
			Bold(true)

	SuccessStyle = lipgloss.NewStyle().
			Foreground(SuccessColor).
			Bold(true)

	WarningStyle = lipgloss.NewStyle().
			Foreground(WarningColor)

	ErrorStyle = lipgloss.NewStyle().
			Foreground(ErrorColor).
			Bold(true)

	InfoStyle = lipgloss.NewStyle().
			Foreground(InfoColor)

	DimStyle = lipgloss.NewStyle().
			Foreground(DimColor)

	CheckMarkStyle = lipgloss.NewStyle().
			Foreground(SuccessColor).
			Bold(true)

	ArrowStyle = lipgloss.NewStyle().
			Foreground(InfoColor)

	BoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(PrimaryColor).
			Padding(1, 2)

	CodeStyle = lipgloss.NewStyle().
			Background(lipgloss.Color("#2D3436")).
			Foreground(lipgloss.Color("#DFE6E9")).
			Padding(0, 1)
)

type Theme string

type OutputMode int

const (
	ModeQuiet OutputMode = iota
	ModeAnimation
	ModeVerbose
)

const (
	ThemeAuto  Theme = "auto"
	ThemeDark  Theme = "dark"
	ThemeLight Theme = "light"
)

func (m OutputMode) String() string {
	switch m {
	case ModeQuiet:
		return "quiet"
	case ModeAnimation:
		return "animation"
	case ModeVerbose:
		return "verbose"
	default:
		return "unknown"
	}
}

func GetThemeStyles(theme Theme) map[string]lipgloss.Style {
	if theme == ThemeAuto {
		theme = ThemeDark
	}

	return map[string]lipgloss.Style{
		"header":    HeaderStyle,
		"subheader": SubheaderStyle,
		"success":   SuccessStyle,
		"warning":   WarningStyle,
		"error":     ErrorStyle,
		"info":      InfoStyle,
		"dim":       DimStyle,
		"checkmark": CheckMarkStyle,
		"arrow":     ArrowStyle,
		"box":       BoxStyle,
		"code":      CodeStyle,
	}
}
