package panes

import (
	"fmt"
	"sort"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/yoshpy-dev/harness-engineering-scaffolding-template/internal/state"
	"github.com/yoshpy-dev/harness-engineering-scaffolding-template/internal/ui"
)

// DepsModel shows a dependency tree for slices.
type DepsModel struct {
	deps     []state.SliceDependency
	slices   []state.SliceState
	selected string
	width    int
	height   int
	focused  bool
}

// NewDeps creates a new DepsModel.
func NewDeps(deps []state.SliceDependency, slices []state.SliceState, width, height int) DepsModel {
	return DepsModel{
		deps:   deps,
		slices: slices,
		width:  width,
		height: height,
	}
}

// SetDeps updates the dependency data.
func (m *DepsModel) SetDeps(deps []state.SliceDependency, slices []state.SliceState) {
	m.deps = deps
	m.slices = slices
}

// SetSelected sets which slice is highlighted in the tree.
func (m *DepsModel) SetSelected(name string) {
	m.selected = name
}

// SetSize updates the pane dimensions.
func (m *DepsModel) SetSize(width, height int) {
	m.width = width
	m.height = height
}

// SetFocused sets the focus state.
func (m *DepsModel) SetFocused(focused bool) {
	m.focused = focused
}

// Init implements tea.Model.
func (m DepsModel) Init() tea.Cmd {
	return nil
}

// Update implements tea.Model.
func (m DepsModel) Update(msg tea.Msg) (DepsModel, tea.Cmd) {
	switch msg := msg.(type) {
	case ui.SliceSelectedMsg:
		m.selected = msg.Slice.Name
	case ui.StateUpdatedMsg:
		m.deps = msg.Status.Dependencies
		m.slices = msg.Status.Slices
	}
	return m, nil
}

// View renders the dependency tree.
func (m DepsModel) View() string {
	if len(m.deps) == 0 && len(m.slices) == 0 {
		return "No dependencies"
	}

	statusMap := make(map[string]state.SliceStatus)
	for _, s := range m.slices {
		statusMap[s.Name] = s.Status
	}

	// Build adjacency: parent -> children.
	children := make(map[string][]string)
	hasParent := make(map[string]bool)
	allNodes := make(map[string]bool)

	for _, d := range m.deps {
		children[d.From] = append(children[d.From], d.To)
		hasParent[d.To] = true
		allNodes[d.From] = true
		allNodes[d.To] = true
	}

	// Also include slices without dependencies.
	for _, s := range m.slices {
		allNodes[s.Name] = true
	}

	// Find roots (nodes with no parent).
	var roots []string
	for name := range allNodes {
		if !hasParent[name] {
			roots = append(roots, name)
		}
	}
	sort.Strings(roots)

	// Sort children for deterministic output.
	for k := range children {
		sort.Strings(children[k])
	}

	var b strings.Builder
	for i, root := range roots {
		isLast := i == len(roots)-1
		m.renderNode(&b, root, "", isLast, statusMap, children)
	}

	return strings.TrimRight(b.String(), "\n")
}

func (m DepsModel) renderNode(b *strings.Builder, name, prefix string, isLast bool, statusMap map[string]state.SliceStatus, children map[string][]string) {
	connector := "├── "
	if isLast {
		connector = "└── "
	}
	// Root-level nodes have no prefix connector.
	if prefix == "" {
		connector = ""
	}

	status := statusMap[name]
	clr := ui.StatusColor(string(status))
	icon := ui.StatusIcon(string(status))

	nodeStyle := lipgloss.NewStyle().Foreground(clr)
	label := fmt.Sprintf("%s %s", icon, name)

	if name == m.selected {
		label = nodeStyle.Bold(true).Render(label)
	} else {
		label = nodeStyle.Render(label)
	}

	fmt.Fprintf(b, "%s%s%s\n", prefix, connector, label)

	kids := children[name]
	for i, child := range kids {
		childIsLast := i == len(kids)-1
		childPrefix := prefix
		if prefix != "" {
			if isLast {
				childPrefix += "    "
			} else {
				childPrefix += "│   "
			}
		} else {
			// First level: add indentation for children.
			if isLast {
				childPrefix = "    "
			} else {
				childPrefix = "│   "
			}
		}
		m.renderNode(b, child, childPrefix, childIsLast, statusMap, children)
	}
}
