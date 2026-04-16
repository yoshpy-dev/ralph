package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	tea "charm.land/bubbletea/v2"

	"github.com/yoshpy-dev/harness-engineering-scaffolding-template/internal/action"
	"github.com/yoshpy-dev/harness-engineering-scaffolding-template/internal/state"
	"github.com/yoshpy-dev/harness-engineering-scaffolding-template/internal/ui"
	"github.com/yoshpy-dev/harness-engineering-scaffolding-template/internal/ui/panes"
	"github.com/yoshpy-dev/harness-engineering-scaffolding-template/internal/watcher"
)

func main() {
	orchDir := flag.String("orch-dir", "", "path to orchestrator state directory")
	worktreeBase := flag.String("worktree-base", "", "path to worktree base directory")
	planDir := flag.String("plan-dir", "", "path to plan directory (for dependency graph)")
	showVersion := flag.Bool("version", false, "print version and exit")
	flag.Parse()

	if *showVersion {
		fmt.Println(versionString())
		os.Exit(0)
	}

	if *orchDir == "" {
		log.Fatal("--orch-dir is required")
	}
	if *worktreeBase == "" {
		log.Fatal("--worktree-base is required")
	}

	// Resolve to absolute paths for reliable file watching.
	absOrchDir, err := filepath.Abs(*orchDir)
	if err != nil {
		log.Fatalf("resolving orch-dir: %v", err)
	}
	absWorktreeBase, err := filepath.Abs(*worktreeBase)
	if err != nil {
		log.Fatalf("resolving worktree-base: %v", err)
	}

	absPlanDir := ""
	if *planDir != "" {
		absPlanDir, err = filepath.Abs(*planDir)
		if err != nil {
			log.Fatalf("resolving plan-dir: %v", err)
		}
	}

	// Read initial state.
	status, err := state.ReadFullStatus(absOrchDir, absWorktreeBase, absPlanDir)
	if err != nil {
		log.Fatalf("reading initial state: %v", err)
	}

	// Create file watcher.
	w, err := watcher.New(absOrchDir, absWorktreeBase)
	if err != nil {
		log.Fatalf("creating watcher: %v", err)
	}
	defer func() { _ = w.Stop() }()

	// Create action executor (repo root is two levels up from orch-dir:
	// .harness/state/loop/ → repo root).
	repoRoot := resolveRepoRoot(absOrchDir)
	var executor *action.Executor
	if exec, err := action.NewExecutor(repoRoot); err == nil {
		executor = exec
	}

	// Build the TUI model.
	model := newAppModel(status, w, executor, absOrchDir, absWorktreeBase, absPlanDir)

	p := tea.NewProgram(model)
	if _, err := p.Run(); err != nil {
		log.Fatalf("TUI error: %v", err)
	}
}

// resolveRepoRoot walks up from orchDir to find the repo root.
// orchDir is typically .harness/state/loop/ so we go up 3 levels.
func resolveRepoRoot(orchDir string) string {
	dir := orchDir
	for range 10 {
		if _, err := os.Stat(filepath.Join(dir, ".git")); err == nil {
			return dir
		}
		if _, err := os.Stat(filepath.Join(dir, "scripts", "ralph")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	// Fallback: assume 3 levels up
	return filepath.Join(orchDir, "..", "..", "..")
}

// appModel wraps the ui.Model with sub-model composition, watcher integration,
// and state refresh logic. This layer bridges ui and ui/panes to avoid import cycles.
type appModel struct {
	ui       ui.Model
	watcher  *watcher.Watcher
	executor *action.Executor
	orchDir  string
	wtBase   string
	planDir  string

	// Sub-models (owned here to avoid ui ← ui/panes cycle).
	sliceList panes.SliceListModel
	detail    panes.DetailModel
	deps      panes.DepsModel
	actions   panes.ActionsModel
	logView   panes.LogViewModel
	progress  panes.ProgressModel
}

func newAppModel(status *state.FullStatus, w *watcher.Watcher, exec *action.Executor, orchDir, wtBase, planDir string) *appModel {
	m := &appModel{
		ui:       ui.New(),
		watcher:  w,
		executor: exec,
		orchDir:  orchDir,
		wtBase:   wtBase,
		planDir:  planDir,

		sliceList: panes.NewSliceList(status.Slices, 0, 0),
		detail:    panes.NewDetail(0, 0),
		deps:      panes.NewDeps(status.Dependencies, status.Slices, 0, 0),
		actions:   panes.NewActionsModel(exec),
		logView:   panes.NewLogView(0, 0),
		progress:  panes.NewProgress(status.Slices, 0),
	}

	// Set initial focus on slices pane.
	m.sliceList.SetFocused(true)

	// Select the first slice if available.
	if s, ok := m.sliceList.SelectedSlice(); ok {
		m.detail.SetSlice(&s)
		m.deps.SetSelected(s.Name)
	}

	// Render initial pane contents.
	m.syncPaneContents()

	return m
}

func (m *appModel) Init() tea.Cmd {
	return m.watcher.Watch()
}

func (m *appModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		// Forward to ui.Model for size tracking, then resize sub-models.
		innerModel, cmd := m.ui.Update(msg)
		m.ui = innerModel.(ui.Model)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
		m.resizePanes()
		m.syncPaneContents()
		return m, tea.Batch(cmds...)

	case watcher.StateChangedMsg:
		// Re-read full state on any file change.
		if status, err := state.ReadFullStatus(m.orchDir, m.wtBase, m.planDir); err == nil {
			m.sliceList.SetSlices(status.Slices)
			m.deps.SetDeps(status.Dependencies, status.Slices)
			m.progress.SetSlices(status.Slices)
			// Refresh detail for current selection.
			if s, ok := m.sliceList.SelectedSlice(); ok {
				m.detail.SetSlice(&s)
				m.deps.SetSelected(s.Name)
			}
			m.syncPaneContents()
		}
		cmds = append(cmds, m.watcher.Watch())
		return m, tea.Batch(cmds...)

	case watcher.LogLineMsg:
		m.logView.AppendLine(msg.Line)
		m.syncPaneContents()
		cmds = append(cmds, m.watcher.Watch())
		return m, tea.Batch(cmds...)

	case watcher.WatcherErrorMsg:
		return m, nil

	case ui.ConfirmYesMsg:
		cmd := m.actions.ExecuteConfirmed(msg.Tag)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
		return m, tea.Batch(cmds...)

	case ui.ConfirmNoMsg:
		return m, nil

	case ui.SliceSelectedMsg:
		s := msg.Slice
		m.detail.SetSlice(&s)
		m.deps.SetSelected(s.Name)
		m.syncPaneContents()
		return m, nil
	}

	// Check for action result messages and forward to actions pane.
	switch msg.(type) {
	case action.RetryResultMsg, action.AbortResultMsg, action.ExternalDoneMsg:
		var cmd tea.Cmd
		m.actions, cmd = m.actions.Update(msg)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
		m.syncPaneContents()
		return m, tea.Batch(cmds...)
	}

	// Key events: dispatch to focused sub-model first, then to ui.Model.
	if kmsg, ok := msg.(tea.KeyPressMsg); ok {
		// If confirmation dialog is visible, let ui.Model handle it.
		if m.ui.Confirm.Visible {
			innerModel, cmd := m.ui.Update(kmsg)
			m.ui = innerModel.(ui.Model)
			if cmd != nil {
				cmds = append(cmds, cmd)
			}
			return m, tea.Batch(cmds...)
		}

		// Try focused sub-model first.
		subConsumed := false
		switch m.ui.Focused {
		case ui.PaneSlices:
			var cmd tea.Cmd
			m.sliceList, cmd = m.sliceList.Update(kmsg)
			if cmd != nil {
				cmds = append(cmds, cmd)
				subConsumed = true
			}

		case ui.PaneLogs:
			var cmd tea.Cmd
			m.logView, cmd = m.logView.Update(kmsg)
			if cmd != nil {
				cmds = append(cmds, cmd)
				subConsumed = true
			}

		case ui.PaneActions:
			cmd, confirmReq, consumed := m.actions.HandleKey(kmsg)
			if confirmReq != nil {
				m.ui.ShowConfirm(confirmReq.Message, confirmReq.Tag)
				m.syncPaneContents()
				return m, nil
			}
			if cmd != nil {
				cmds = append(cmds, cmd)
			}
			if consumed {
				subConsumed = true
			}
		}

		// Always pass to ui.Model for global keys (quit, help, pane nav, etc.).
		prevFocused := m.ui.Focused
		innerModel, cmd := m.ui.Update(kmsg)
		m.ui = innerModel.(ui.Model)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}

		// If focus changed, sync focus state on sub-models.
		if m.ui.Focused != prevFocused {
			m.syncFocus()
		}

		if subConsumed || len(cmds) > 0 {
			m.syncPaneContents()
		}

		if m.ui.Quitting {
			return m, tea.Quit
		}

		return m, tea.Batch(cmds...)
	}

	// Pass everything else to the inner UI model.
	innerModel, cmd := m.ui.Update(msg)
	m.ui = innerModel.(ui.Model)
	if cmd != nil {
		cmds = append(cmds, cmd)
	}

	if m.ui.Quitting {
		return m, tea.Quit
	}

	return m, tea.Batch(cmds...)
}

func (m *appModel) View() tea.View {
	m.syncPaneContents()
	v := m.ui.View()
	v.AltScreen = true
	return v
}

// syncPaneContents renders all sub-models into the ui.Model's PaneContents.
func (m *appModel) syncPaneContents() {
	m.ui.Panes = ui.PaneContents{
		Slices:   m.sliceList.View(),
		Detail:   m.detail.View(),
		Deps:     m.deps.View(),
		Actions:  m.actions.View(),
		Logs:     m.logView.View(),
		Progress: m.progress.View(),
	}
}

// syncFocus updates the focused state on all sub-models.
func (m *appModel) syncFocus() {
	m.sliceList.SetFocused(m.ui.Focused == ui.PaneSlices)
	m.detail.SetFocused(m.ui.Focused == ui.PaneDetail)
	m.deps.SetFocused(m.ui.Focused == ui.PaneDeps)
	m.logView.SetFocused(m.ui.Focused == ui.PaneLogs)
	m.actions = m.actions.SetFocused(m.ui.Focused == ui.PaneActions)
}

// resizePanes recalculates pane dimensions from the terminal size.
func (m *appModel) resizePanes() {
	w, h := m.ui.Width, m.ui.Height
	if w == 0 || h == 0 {
		return
	}

	contentHeight := max(h-1, 6)
	upperHeight := contentHeight * 60 / 100
	lowerHeight := contentHeight - upperHeight

	slicesW := w * 30 / 100
	detailW := w * 35 / 100
	depsW := w - slicesW - detailW

	actionsW := w * 30 / 100
	logsW := w - actionsW

	// Subtract border (2) for content dimensions.
	m.sliceList.SetSize(slicesW-2, upperHeight-2)
	m.detail.SetSize(detailW-2, upperHeight-2)
	m.deps.SetSize(depsW-2, upperHeight-2)
	m.actions = m.actions.SetSize(actionsW-2, lowerHeight-2)
	m.logView.SetSize(logsW-2, lowerHeight-2)
	m.progress.SetWidth(w)
}
