<<<<<<< HEAD
package ui

// Pane identifies one of the TUI panes.
type Pane int

const (
	PaneSlices  Pane = iota // upper-left: slice list
	PaneDetail              // upper-center: slice detail
	PaneDeps                // upper-right: dependency graph
	PaneActions             // lower-left: actions
	PaneLogs                // lower-right: logs
)

// PaneCount is the total number of navigable panes.
const PaneCount = 5

// String returns the display name for the pane.
func (p Pane) String() string {
	switch p {
	case PaneSlices:
		return "Slices"
	case PaneDetail:
		return "Detail"
	case PaneDeps:
		return "Dependencies"
	case PaneActions:
		return "Actions"
	case PaneLogs:
		return "Logs"
	default:
		return "Unknown"
	}
}

// NextPane returns the next pane in tab order (wrapping).
func NextPane(current Pane) Pane {
	return Pane((int(current) + 1) % PaneCount)
}

// PrevPane returns the previous pane in tab order (wrapping).
func PrevPane(current Pane) Pane {
	return Pane((int(current) + PaneCount - 1) % PaneCount)
}

// RightPane returns the pane to the right of current.
// Upper row: Slices -> Detail -> Deps (stops at edge).
// Lower row: Actions -> Logs (stops at edge).
func RightPane(current Pane) Pane {
	switch current {
	case PaneSlices:
		return PaneDetail
	case PaneDetail:
		return PaneDeps
	case PaneActions:
		return PaneLogs
	default:
		return current
	}
}

// LeftPane returns the pane to the left of current.
// Upper row: Deps -> Detail -> Slices (stops at edge).
// Lower row: Logs -> Actions (stops at edge).
func LeftPane(current Pane) Pane {
	switch current {
	case PaneDetail:
		return PaneSlices
	case PaneDeps:
		return PaneDetail
	case PaneLogs:
		return PaneActions
	default:
		return current
	}
}
||||||| 085ae31
=======
package ui

// Pane identifies a pane in the TUI layout.
type Pane int

const (
	PaneSlices Pane = iota
	PaneDetail
	PaneDeps
	PaneLogs
)

// PaneCount is the total number of navigable panes.
const PaneCount = 4

// NextPane returns the next pane in tab order.
func NextPane(current Pane) Pane {
	return (current + 1) % Pane(PaneCount)
}

// PrevPane returns the previous pane in tab order.
func PrevPane(current Pane) Pane {
	return (current - 1 + Pane(PaneCount)) % Pane(PaneCount)
}
>>>>>>> slice/2026-04-15-ralph-tui/4-ralph-tui
