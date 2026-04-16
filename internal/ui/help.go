package ui

import (
	"strings"

	"charm.land/lipgloss/v2"
)

// helpEntry is a single keybinding entry for the help overlay.
type helpEntry struct {
	key  string
	desc string
}

// helpEntries returns all keybindings as display entries.
func helpEntries() []helpEntry {
	return []helpEntry{
		{"h / \u2190", "Move to left pane"},
		{"l / \u2192", "Move to right pane"},
		{"Tab", "Next pane"},
		{"S-Tab", "Previous pane"},
		{"j / \u2193", "Scroll down"},
		{"k / \u2191", "Scroll up"},
		{"Space", "Select item"},
		{"Enter", "Confirm"},
		{"r", "Retry slice"},
		{"a", "Abort slice"},
		{"A", "Abort all slices"},
		{"L", "View full log"},
		{"e", "Open editor"},
		{"d", "Focus deps"},
		{"/", "Search"},
		{"?", "Toggle help"},
		{"q", "Quit"},
	}
}

// RenderHelp renders the help overlay centered in the given dimensions.
func RenderHelp(width, height int) string {
	entries := helpEntries()

	var sb strings.Builder
	sb.WriteString(TitleStyle().Render("Keybindings"))
	sb.WriteString("\n\n")

	keyStyle := lipgloss.NewStyle().Width(14).Bold(true)
	for _, e := range entries {
		sb.WriteString(keyStyle.Render(e.key))
		sb.WriteString(e.desc)
		sb.WriteString("\n")
	}
	sb.WriteString("\nPress ? to close")

	overlay := HelpOverlayStyle().Render(sb.String())

	return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, overlay)
}
