<<<<<<< HEAD
package ui

import "charm.land/lipgloss/v2"

// Color constants used across the TUI.
var (
	ColorFocusBorder   = lipgloss.Color("62")  // purple
	ColorNormalBorder   = lipgloss.Color("240") // gray
	ColorTitle          = lipgloss.Color("170") // pink
	ColorStatusOK       = lipgloss.Color("42")  // green
	ColorStatusFail     = lipgloss.Color("196") // red
	ColorStatusRunning  = lipgloss.Color("214") // orange
	ColorStatusPending  = lipgloss.Color("245") // dim gray
	ColorProgress       = lipgloss.Color("62")  // purple
)

// PaneStyle returns the border style for a pane.
// width and height are the content dimensions (border adds 2 to each).
func PaneStyle(focused bool, width, height int) lipgloss.Style {
	c := ColorNormalBorder
	if focused {
		c = ColorFocusBorder
	}
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(c).
		Width(width).
		Height(height)
}

// TitleStyle returns the style for pane titles.
func TitleStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		Bold(true).
		Foreground(ColorTitle)
}

// StatusStyle returns a style colored by status.
func StatusStyle(status string) lipgloss.Style {
	switch status {
	case "complete", "passed":
		return lipgloss.NewStyle().Foreground(ColorStatusOK)
	case "failed", "error":
		return lipgloss.NewStyle().Foreground(ColorStatusFail)
	case "running":
		return lipgloss.NewStyle().Foreground(ColorStatusRunning)
	default:
		return lipgloss.NewStyle().Foreground(ColorStatusPending)
	}
}

// HelpOverlayStyle returns the style for the help overlay box.
func HelpOverlayStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ColorFocusBorder).
		Padding(1, 2)
}
||||||| 085ae31
=======
package ui

import (
	"image/color"

	"charm.land/lipgloss/v2"
)

// StatusIcon returns the icon character for a given slice status.
func StatusIcon(status string) string {
	switch status {
	case "complete":
		return "+"
	case "running":
		return "*"
	case "pending":
		return "-"
	case "failed", "stuck", "repair_limit", "aborted", "config_error", "max_retries":
		return "!"
	default:
		return "?"
	}
}

// StatusColor returns the color for a given slice status.
func StatusColor(status string) color.Color {
	switch status {
	case "complete":
		return lipgloss.Color("#00FF00")
	case "running":
		return lipgloss.Color("#00FFFF")
	case "pending":
		return lipgloss.Color("#808080")
	case "failed", "stuck", "repair_limit", "aborted", "config_error", "max_retries":
		return lipgloss.Color("#FF0000")
	default:
		return lipgloss.Color("#808080")
	}
}

// PhaseColor returns the color for a given pipeline phase.
func PhaseColor(phase string) color.Color {
	switch phase {
	case "inner":
		return lipgloss.Color("#00FFFF")
	case "outer":
		return lipgloss.Color("#FFFF00")
	case "done":
		return lipgloss.Color("#00FF00")
	default:
		return lipgloss.Color("#808080")
	}
}
>>>>>>> slice/2026-04-15-ralph-tui/4-ralph-tui
