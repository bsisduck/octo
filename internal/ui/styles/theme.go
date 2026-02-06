package styles

import (
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
)

// Color palette -- single source of truth
var (
	ColorPrimary     = lipgloss.Color("69")
	ColorSuccess     = lipgloss.Color("42")
	ColorWarning     = lipgloss.Color("214")
	ColorError       = lipgloss.Color("196")
	ColorMuted       = lipgloss.Color("241")
	ColorText        = lipgloss.Color("255")
	ColorHighlight   = lipgloss.Color("62")
	ColorHighlightBg = lipgloss.Color("237")
	ColorNormal      = lipgloss.Color("252")
)

// Title styles
var (
	Title = lipgloss.NewStyle().Bold(true).Foreground(ColorPrimary)

	TitleWithMargin = lipgloss.NewStyle().
		Bold(true).
		Foreground(ColorPrimary).
		MarginBottom(1)

	Logo = lipgloss.NewStyle().Foreground(ColorPrimary)

	Tagline = lipgloss.NewStyle().
		Foreground(ColorMuted).
		Italic(true)
)

// Section and category styles
var (
	Section = lipgloss.NewStyle().Bold(true).Foreground(ColorSuccess)

	Category = lipgloss.NewStyle().
		Bold(true).
		Foreground(ColorSuccess).
		PaddingLeft(1)
)

// Status and state styles
var (
	Success = lipgloss.NewStyle().Foreground(ColorSuccess)

	Running = lipgloss.NewStyle().Foreground(ColorSuccess)

	Warning = lipgloss.NewStyle().Foreground(ColorWarning)

	Warn = lipgloss.NewStyle().Bold(true).Foreground(ColorWarning)

	Error = lipgloss.NewStyle().Foreground(ColorError)

	Stopped = lipgloss.NewStyle().Foreground(ColorWarning)

	Info = lipgloss.NewStyle().Foreground(ColorMuted)
)

// Label and value styles
var (
	Label = lipgloss.NewStyle().Foreground(ColorMuted)

	LabelWithWidth = lipgloss.NewStyle().
		Foreground(ColorMuted).
		Width(14)

	StatLabel = lipgloss.NewStyle().
		Foreground(ColorMuted).
		Width(12)

	Value = lipgloss.NewStyle().Foreground(ColorText)

	StatValue = lipgloss.NewStyle().Foreground(ColorSuccess)

	Size = lipgloss.NewStyle().
		Foreground(ColorMuted).
		Width(10).
		Align(lipgloss.Right)
)

// Selection and interaction styles
var (
	Selected = lipgloss.NewStyle().
		Foreground(ColorText).
		Background(ColorHighlight).
		Bold(true).
		Padding(0, 1)

	Normal = lipgloss.NewStyle().
		Foreground(ColorNormal).
		Padding(0, 1)

	SelectedAnalyze = lipgloss.NewStyle().
		Background(ColorHighlightBg).
		Foreground(ColorText)

	NormalAnalyze = lipgloss.NewStyle().PaddingLeft(3)

	Unused = lipgloss.NewStyle().Foreground(ColorWarning)
)

// Help and instructional styles
var (
	Help = lipgloss.NewStyle().Foreground(ColorMuted)

	Subtitle = lipgloss.NewStyle().Foreground(ColorMuted)

	DeleteConfirm = lipgloss.NewStyle().
		Foreground(ColorError).
		Bold(true)
)

// DisableColors forces all Lipgloss rendering to produce plain text.
// Call once at startup from cmd/root.go based on --no-color flag.
func DisableColors() {
	lipgloss.SetColorProfile(termenv.Ascii)
}
