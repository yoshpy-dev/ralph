package ui

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
)

// PaneContents holds the rendered text for each pane.
// Sub-models render their output into these strings via appModel.
type PaneContents struct {
	Slices   string
	Detail   string
	Deps     string
	Actions  string
	Logs     string
	Progress string
}

// MinWidth is the minimum terminal width for the full layout.
const MinWidth = 60

// MinHeight is the minimum terminal height for the full layout.
const MinHeight = 15

// RenderLayout renders the full TUI layout.
func RenderLayout(width, height int, panes PaneContents, focused Pane, progress string) string {
	if width < MinWidth || height < MinHeight {
		return renderCompact(width, height, panes, focused, progress)
	}

	// Reserve 1 line for the progress bar.
	contentHeight := max(height-1, 6)

	// Upper row: 60% height, Lower row: 40% height.
	upperHeight := contentHeight * 60 / 100
	lowerHeight := contentHeight - upperHeight

	// Upper row widths: slices 30%, detail 35%, deps 35%.
	slicesW := width * 30 / 100
	detailW := width * 35 / 100
	depsW := width - slicesW - detailW

	// Lower row widths: actions 30%, logs 70%.
	actionsW := width * 30 / 100
	logsW := width - actionsW

	slicesPane := renderPane("Slices", panes.Slices, focused == PaneSlices, slicesW, upperHeight)
	detailPane := renderPane("Detail", panes.Detail, focused == PaneDetail, detailW, upperHeight)
	depsPane := renderPane("Dependencies", panes.Deps, focused == PaneDeps, depsW, upperHeight)
	actionsPane := renderPane("Actions", panes.Actions, focused == PaneActions, actionsW, lowerHeight)
	logsPane := renderPane("Logs", panes.Logs, focused == PaneLogs, logsW, lowerHeight)

	upperRow := lipgloss.JoinHorizontal(lipgloss.Top, slicesPane, detailPane, depsPane)
	lowerRow := lipgloss.JoinHorizontal(lipgloss.Top, actionsPane, logsPane)

	body := lipgloss.JoinVertical(lipgloss.Left, upperRow, lowerRow)

	progressLine := lipgloss.NewStyle().Width(width).Render(progress)

	return lipgloss.JoinVertical(lipgloss.Left, body, progressLine)
}

// renderPane renders a single pane with a title and bordered box.
func renderPane(title, content string, focused bool, totalWidth, totalHeight int) string {
	// Content dimensions = total - border (1 char each side).
	contentW := totalWidth - 2
	contentH := totalHeight - 2
	if contentW < 1 {
		contentW = 1
	}
	if contentH < 1 {
		contentH = 1
	}

	titleLine := TitleStyle().Render(title)
	body := fmt.Sprintf("%s\n%s", titleLine, content)

	return PaneStyle(focused, contentW, contentH).Render(body)
}

// renderCompact renders a simplified single-pane view for small terminals.
func renderCompact(width, _ int, panes PaneContents, focused Pane, progress string) string {
	var sb strings.Builder
	sb.WriteString(TitleStyle().Render(fmt.Sprintf("ralph-tui [%s]", focused)))
	sb.WriteString("\n")

	switch focused {
	case PaneSlices:
		sb.WriteString(panes.Slices)
	case PaneDetail:
		sb.WriteString(panes.Detail)
	case PaneDeps:
		sb.WriteString(panes.Deps)
	case PaneActions:
		sb.WriteString(panes.Actions)
	case PaneLogs:
		sb.WriteString(panes.Logs)
	}
	sb.WriteString("\n")
	sb.WriteString(lipgloss.NewStyle().Width(width).Render(progress))
	return sb.String()
}
