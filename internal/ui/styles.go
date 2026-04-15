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
