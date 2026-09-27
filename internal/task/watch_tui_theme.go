// Colors and lipgloss styles for the watch TUI. Kept in one place so the
// palette can be reworked without touching layout or event logic.
package task

import (
	"github.com/charmbracelet/lipgloss"
)

// Muted palette for agent colors — no neon
var agentPalette = []lipgloss.Color{
	lipgloss.Color("73"),  // steel teal
	lipgloss.Color("140"), // mauve
	lipgloss.Color("107"), // sage
	lipgloss.Color("179"), // amber
	lipgloss.Color("110"), // slate blue
	lipgloss.Color("174"), // dusty rose
	lipgloss.Color("115"), // seafoam
	lipgloss.Color("183"), // lavender
}

var (
	colorSep       = lipgloss.Color("237")
	colorMuted     = lipgloss.Color("241")
	colorDim       = lipgloss.Color("245")
	colorText      = lipgloss.Color("251")
	colorAccent    = lipgloss.Color("73")
	colorSelBg     = lipgloss.Color("238")
	colorBarBg     = lipgloss.Color("232")
	colorPayloadBg = lipgloss.Color("233")

	tuiBarStyle       = lipgloss.NewStyle().Foreground(colorDim).Background(colorBarBg).Padding(0, 1)
	tuiSectionTitle   = lipgloss.NewStyle().Foreground(colorAccent).Bold(true)
	tuiSepStyle       = lipgloss.NewStyle().Foreground(colorSep)
	tuiHeaderRow      = lipgloss.NewStyle().Foreground(colorAccent).Bold(true)
	tuiSelectedRow    = lipgloss.NewStyle().Foreground(lipgloss.Color("255")).Background(colorSelBg).Bold(true)
	tuiSelectionStyle = lipgloss.NewStyle().Foreground(colorAccent).Bold(true)
	tuiMuted          = lipgloss.NewStyle().Foreground(colorMuted)
	tuiLabel          = lipgloss.NewStyle().Foreground(colorDim)
	tuiBorderColor    = colorSep // kept for compatibility
	tuiOkStyle        = lipgloss.NewStyle().Foreground(lipgloss.Color("71"))
	tuiErrStyle       = lipgloss.NewStyle().Foreground(lipgloss.Color("167"))
	tuiRunningStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("136"))
	tuiDefaultStatus  = lipgloss.NewStyle().Foreground(colorMuted)
	tuiDefaultCell    = lipgloss.NewStyle().Foreground(colorText)
	tuiPayloadLine    = lipgloss.NewStyle().Foreground(colorText).Background(colorPayloadBg)
)
