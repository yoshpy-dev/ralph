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
