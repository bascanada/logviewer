// Package tui provides the terminal user interface components.
package tui

import "github.com/charmbracelet/bubbles/key"

// KeyMap defines all keyboard shortcuts
type KeyMap struct {
	// Navigation
	Up       key.Binding
	Down     key.Binding
	PageUp   key.Binding
	PageDown key.Binding
	Home     key.Binding
	End      key.Binding

	// Tab navigation
	NextTab key.Binding
	PrevTab key.Binding
	NewTab  key.Binding
	CloseTab key.Binding

	// Sidebar
	ToggleSidebar   key.Binding
	ExpandSidebar   key.Binding
	ShrinkSidebar   key.Binding

	// Display
	ToggleWrap key.Binding

	// Search
	Search      key.Binding
	ClearSearch key.Binding

	// Actions
	Refresh key.Binding
	Copy    key.Binding
	Help    key.Binding
	Quit    key.Binding
}

// DefaultKeyMap returns the default keybindings
func DefaultKeyMap() KeyMap {
	return KeyMap{
		Up: key.NewBinding(
			key.WithKeys("up", "k"),
			key.WithHelp("↑/k", "up"),
		),
		Down: key.NewBinding(
			key.WithKeys("down", "j"),
			key.WithHelp("↓/j", "down"),
		),
		PageUp: key.NewBinding(
			key.WithKeys("pgup", "ctrl+u"),
			key.WithHelp("PgUp", "page up"),
		),
		PageDown: key.NewBinding(
			key.WithKeys("pgdown", "ctrl+d"),
			key.WithHelp("PgDn", "page down"),
		),
		Home: key.NewBinding(
			key.WithKeys("home", "g"),
			key.WithHelp("Home/g", "go to top"),
		),
		End: key.NewBinding(
			key.WithKeys("end", "G"),
			key.WithHelp("End/G", "go to bottom"),
		),
		NextTab: key.NewBinding(
			key.WithKeys("tab", "l"),
			key.WithHelp("Tab/l", "next tab"),
		),
		PrevTab: key.NewBinding(
			key.WithKeys("shift+tab", "h"),
			key.WithHelp("S-Tab/h", "prev tab"),
		),
		NewTab: key.NewBinding(
			key.WithKeys("ctrl+t"),
			key.WithHelp("Ctrl+t", "new tab"),
		),
		CloseTab: key.NewBinding(
			key.WithKeys("ctrl+w", "q"),
			key.WithHelp("Ctrl+w/q", "close tab"),
		),
		ToggleSidebar: key.NewBinding(
			key.WithKeys("enter", "d"),
			key.WithHelp("Enter/d", "toggle details"),
		),
		ExpandSidebar: key.NewBinding(
			key.WithKeys("]", "alt+right"),
			key.WithHelp("]", "expand sidebar"),
		),
		ShrinkSidebar: key.NewBinding(
			key.WithKeys("[", "alt+left"),
			key.WithHelp("[", "shrink sidebar"),
		),
		ToggleWrap: key.NewBinding(
			key.WithKeys("w"),
			key.WithHelp("w", "toggle wrap"),
		),
		Search: key.NewBinding(
			key.WithKeys("/"),
			key.WithHelp("/", "search"),
		),
		ClearSearch: key.NewBinding(
			key.WithKeys("esc"),
			key.WithHelp("Esc", "clear/cancel"),
		),
		Refresh: key.NewBinding(
			key.WithKeys("r", "ctrl+r"),
			key.WithHelp("r", "refresh"),
		),
		Copy: key.NewBinding(
			key.WithKeys("c", "y"),
			key.WithHelp("c/y", "copy line"),
		),
		Help: key.NewBinding(
			key.WithKeys("?"),
			key.WithHelp("?", "help"),
		),
		Quit: key.NewBinding(
			key.WithKeys("ctrl+c"),
			key.WithHelp("Ctrl+c", "quit"),
		),
	}
}

// ShortHelp returns keybindings to be shown in the mini help view
func (k KeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Up, k.Down, k.Search, k.ToggleSidebar, k.NextTab, k.Help, k.Quit}
}

// FullHelp returns keybindings for the expanded help view
func (k KeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Up, k.Down, k.PageUp, k.PageDown, k.Home, k.End},
		{k.NextTab, k.PrevTab, k.NewTab, k.CloseTab},
		{k.ToggleSidebar, k.ExpandSidebar, k.ShrinkSidebar},
		{k.ToggleWrap, k.Search, k.ClearSearch, k.Refresh, k.Copy},
		{k.Help, k.Quit},
	}
}
