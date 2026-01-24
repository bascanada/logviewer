// SPDX-License-Identifier: GPL-3.0-only
package tui

import "github.com/charmbracelet/lipgloss"

// Color palette
var (
	ColorPrimary   = lipgloss.Color("#7C3AED") // Purple
	ColorSecondary = lipgloss.Color("#3B82F6") // Blue
	ColorSuccess   = lipgloss.Color("#22C55E") // Green
	ColorWarning   = lipgloss.Color("#F59E0B") // Amber
	ColorError     = lipgloss.Color("#EF4444") // Red
	ColorMuted     = lipgloss.Color("#6B7280") // Gray
	ColorBorder    = lipgloss.Color("#374151") // Dark gray
	ColorBg        = lipgloss.Color("#1F2937") // Dark background
	ColorBgActive  = lipgloss.Color("#374151") // Active background
	ColorText      = lipgloss.Color("#F9FAFB") // Light text
	ColorTextMuted = lipgloss.Color("#9CA3AF") // Muted text
)

// Log level colors
var LogLevelColors = map[string]lipgloss.Color{
	"ERROR":   ColorError,
	"WARN":    ColorWarning,
	"WARNING": ColorWarning,
	"INFO":    ColorSuccess,
	"DEBUG":   ColorSecondary,
	"TRACE":   ColorMuted,
}

// Styles contains all UI styles
type Styles struct {
	// Base styles
	App        lipgloss.Style
	Header     lipgloss.Style
	Footer     lipgloss.Style
	MainView   lipgloss.Style
	StatusBar  lipgloss.Style
	HelpBar    lipgloss.Style

	// Tab styles
	TabActive   lipgloss.Style
	TabInactive lipgloss.Style
	TabBar      lipgloss.Style

	// Log view styles
	LogList      lipgloss.Style
	LogEntry     lipgloss.Style
	LogSelected  lipgloss.Style
	LogTimestamp lipgloss.Style
	LogLevel     lipgloss.Style
	LogMessage   lipgloss.Style
	LogContext   lipgloss.Style

	// Sidebar styles
	Sidebar       lipgloss.Style
	SidebarTitle  lipgloss.Style
	SidebarKey    lipgloss.Style
	SidebarValue  lipgloss.Style
	SidebarBorder lipgloss.Style

	// Search input styles
	SearchInput       lipgloss.Style
	SearchInputActive lipgloss.Style
	SearchPrompt      lipgloss.Style

	// Border styles
	BorderVertical   lipgloss.Style
	BorderHorizontal lipgloss.Style
}

// DefaultStyles creates the default style set
func DefaultStyles() Styles {
	return Styles{
		App: lipgloss.NewStyle(),

		Header: lipgloss.NewStyle().
			Background(ColorBg).
			Foreground(ColorText).
			Padding(0, 1),

		Footer: lipgloss.NewStyle().
			Background(ColorBg).
			Foreground(ColorTextMuted).
			Padding(0, 1),

		MainView: lipgloss.NewStyle(),

		StatusBar: lipgloss.NewStyle().
			Background(ColorBg).
			Foreground(ColorTextMuted).
			Padding(0, 1),

		HelpBar: lipgloss.NewStyle().
			Background(ColorBg).
			Foreground(ColorMuted).
			Padding(0, 1),

		// Tabs
		TabActive: lipgloss.NewStyle().
			Background(ColorPrimary).
			Foreground(ColorText).
			Bold(true).
			Padding(0, 2).
			MarginRight(1),

		TabInactive: lipgloss.NewStyle().
			Background(ColorBorder).
			Foreground(ColorTextMuted).
			Padding(0, 2).
			MarginRight(1),

		TabBar: lipgloss.NewStyle().
			Background(ColorBg).
			Padding(0, 1),

		// Log view
		LogList: lipgloss.NewStyle(),

		LogEntry: lipgloss.NewStyle().
			Foreground(ColorText),

		LogSelected: lipgloss.NewStyle().
			Background(ColorBgActive).
			Foreground(ColorText).
			Bold(true),

		LogTimestamp: lipgloss.NewStyle().
			Foreground(ColorMuted),

		LogLevel: lipgloss.NewStyle().
			Bold(true).
			Width(5),

		LogMessage: lipgloss.NewStyle().
			Foreground(ColorText),

		LogContext: lipgloss.NewStyle().
			Foreground(ColorSecondary).
			Italic(true),

		// Sidebar
		Sidebar: lipgloss.NewStyle().
			BorderLeft(true).
			BorderStyle(lipgloss.NormalBorder()).
			BorderForeground(ColorBorder).
			Padding(0, 1),

		SidebarTitle: lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorPrimary).
			MarginBottom(1),

		SidebarKey: lipgloss.NewStyle().
			Foreground(ColorSecondary).
			Bold(true),

		SidebarValue: lipgloss.NewStyle().
			Foreground(ColorText),

		SidebarBorder: lipgloss.NewStyle().
			Foreground(ColorBorder),

		// Search
		SearchInput: lipgloss.NewStyle().
			Background(ColorBg).
			Foreground(ColorText).
			Padding(0, 1),

		SearchInputActive: lipgloss.NewStyle().
			Background(ColorBgActive).
			Foreground(ColorText).
			Padding(0, 1).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ColorPrimary),

		SearchPrompt: lipgloss.NewStyle().
			Foreground(ColorPrimary).
			Bold(true),

		// Borders
		BorderVertical: lipgloss.NewStyle().
			Foreground(ColorBorder),

		BorderHorizontal: lipgloss.NewStyle().
			Foreground(ColorBorder),
	}
}

// GetLevelStyle returns a style for the given log level
func GetLevelStyle(level string) lipgloss.Style {
	color, ok := LogLevelColors[level]
	if !ok {
		color = ColorMuted
	}
	return lipgloss.NewStyle().
		Foreground(color).
		Bold(true).
		Width(7).
		Align(lipgloss.Center)
}
