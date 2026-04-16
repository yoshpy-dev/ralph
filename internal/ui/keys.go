package ui

import "charm.land/bubbles/v2/key"

// KeyMap defines all keybindings for the TUI.
type KeyMap struct {
	Quit     key.Binding
	Help     key.Binding
	Left     key.Binding
	Right    key.Binding
	Up       key.Binding
	Down     key.Binding
	NextPane key.Binding
	PrevPane key.Binding
	Select   key.Binding
	Retry    key.Binding
	Abort    key.Binding
	AbortAll key.Binding
	ViewLog  key.Binding
	Editor   key.Binding
	FocusDep key.Binding
	Search   key.Binding
	Enter    key.Binding
}

// DefaultKeyMap returns the default keybinding configuration.
func DefaultKeyMap() KeyMap {
	return KeyMap{
		Quit: key.NewBinding(
			key.WithKeys("q"),
			key.WithHelp("q", "quit"),
		),
		Help: key.NewBinding(
			key.WithKeys("?"),
			key.WithHelp("?", "toggle help"),
		),
		Left: key.NewBinding(
			key.WithKeys("h", "left"),
			key.WithHelp("h/\u2190", "left pane"),
		),
		Right: key.NewBinding(
			key.WithKeys("l", "right"),
			key.WithHelp("l/\u2192", "right pane"),
		),
		Up: key.NewBinding(
			key.WithKeys("k", "up"),
			key.WithHelp("k/\u2191", "up"),
		),
		Down: key.NewBinding(
			key.WithKeys("j", "down"),
			key.WithHelp("j/\u2193", "down"),
		),
		NextPane: key.NewBinding(
			key.WithKeys("tab"),
			key.WithHelp("Tab", "next pane"),
		),
		PrevPane: key.NewBinding(
			key.WithKeys("shift+tab"),
			key.WithHelp("S-Tab", "prev pane"),
		),
		Select: key.NewBinding(
			key.WithKeys(" "),
			key.WithHelp("Space", "select"),
		),
		Retry: key.NewBinding(
			key.WithKeys("r"),
			key.WithHelp("r", "retry"),
		),
		Abort: key.NewBinding(
			key.WithKeys("a"),
			key.WithHelp("a", "abort slice"),
		),
		AbortAll: key.NewBinding(
			key.WithKeys("A"),
			key.WithHelp("A", "abort all"),
		),
		ViewLog: key.NewBinding(
			key.WithKeys("L"),
			key.WithHelp("L", "view log"),
		),
		Editor: key.NewBinding(
			key.WithKeys("e"),
			key.WithHelp("e", "open editor"),
		),
		FocusDep: key.NewBinding(
			key.WithKeys("d"),
			key.WithHelp("d", "focus deps"),
		),
		Search: key.NewBinding(
			key.WithKeys("/"),
			key.WithHelp("/", "search"),
		),
		Enter: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("Enter", "confirm"),
		),
	}
}
