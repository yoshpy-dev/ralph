package cli

import (
	"fmt"
	"os"
	"path/filepath"

	tea "charm.land/bubbletea/v2"
	"github.com/spf13/cobra"

	"github.com/yoshpy-dev/harness-engineering-scaffolding-template/internal/action"
	"github.com/yoshpy-dev/harness-engineering-scaffolding-template/internal/state"
	"github.com/yoshpy-dev/harness-engineering-scaffolding-template/internal/ui"
	"github.com/yoshpy-dev/harness-engineering-scaffolding-template/internal/ui/panes"
	"github.com/yoshpy-dev/harness-engineering-scaffolding-template/internal/watcher"
)

func newStatusCmd() *cobra.Command {
	var (
		orchDir      string
		worktreeBase string
		planDir      string
		jsonMode     bool
		noTUI        bool
	)

	cmd := &cobra.Command{
		Use:   "status",
		Short: "Show pipeline/orchestrator progress",
		Long:  "Display the current pipeline status. Launches TUI when available, otherwise falls back to table or JSON output.",
		RunE: func(cmd *cobra.Command, args []string) error {
			if orchDir == "" {
				orchDir = ".harness/state/orchestrator"
			}
			if worktreeBase == "" {
				worktreeBase = ".claude/worktrees"
			}

			if jsonMode {
				return runStatusJSON(orchDir, worktreeBase, planDir)
			}

			if noTUI || !isTTY() {
				return runStatusTable(orchDir, worktreeBase, planDir)
			}

			return runStatusTUI(orchDir, worktreeBase, planDir)
		},
	}

	cmd.Flags().StringVar(&orchDir, "orch-dir", "", "path to orchestrator state directory (default: .harness/state/orchestrator)")
	cmd.Flags().StringVar(&worktreeBase, "worktree-base", "", "path to worktree base directory (default: .claude/worktrees)")
	cmd.Flags().StringVar(&planDir, "plan-dir", "", "path to plan directory (for dependency graph)")
	cmd.Flags().BoolVar(&jsonMode, "json", false, "output machine-readable JSON")
	cmd.Flags().BoolVar(&noTUI, "no-tui", false, "force table output (skip TUI)")

	return cmd
}

func isTTY() bool {
	fi, err := os.Stdout.Stat()
	if err != nil {
		return false
	}
	return fi.Mode()&os.ModeCharDevice != 0
}

func runStatusJSON(orchDir, worktreeBase, planDir string) error {
	absOrchDir, err := filepath.Abs(orchDir)
	if err != nil {
		return fmt.Errorf("resolving orch-dir: %w", err)
	}
	absWorktreeBase, err := filepath.Abs(worktreeBase)
	if err != nil {
		return fmt.Errorf("resolving worktree-base: %w", err)
	}
	absPlanDir := ""
	if planDir != "" {
		absPlanDir, err = filepath.Abs(planDir)
		if err != nil {
			return fmt.Errorf("resolving plan-dir: %w", err)
		}
	}

	status, err := state.ReadFullStatus(absOrchDir, absWorktreeBase, absPlanDir)
	if err != nil {
		return fmt.Errorf("reading state: %w", err)
	}

	// Simple JSON-like output for now; will be replaced with proper JSON marshaling.
	fmt.Printf("Slices: %d\n", len(status.Slices))
	for _, s := range status.Slices {
		fmt.Printf("  %s: %s\n", s.Name, s.Status)
	}
	return nil
}

func runStatusTable(orchDir, worktreeBase, planDir string) error {
	// Fallback table output — delegates to the same state reader.
	return runStatusJSON(orchDir, worktreeBase, planDir)
}

func runStatusTUI(orchDir, worktreeBase, planDir string) error {
	absOrchDir, err := filepath.Abs(orchDir)
	if err != nil {
		return fmt.Errorf("resolving orch-dir: %w", err)
	}
	absWorktreeBase, err := filepath.Abs(worktreeBase)
	if err != nil {
		return fmt.Errorf("resolving worktree-base: %w", err)
	}
	absPlanDir := ""
	if planDir != "" {
		absPlanDir, err = filepath.Abs(planDir)
		if err != nil {
			return fmt.Errorf("resolving plan-dir: %w", err)
		}
	}

	status, err := state.ReadFullStatus(absOrchDir, absWorktreeBase, absPlanDir)
	if err != nil {
		return fmt.Errorf("reading initial state: %w", err)
	}

	w, err := watcher.New(absOrchDir, absWorktreeBase)
	if err != nil {
		return fmt.Errorf("creating watcher: %w", err)
	}
	defer func() { _ = w.Stop() }()

	repoRoot := resolveRepoRoot(absOrchDir)
	var executor *action.Executor
	if exec, err := action.NewExecutor(repoRoot); err == nil {
		executor = exec
	}

	model := newAppModel(status, w, executor, absOrchDir, absWorktreeBase, absPlanDir)

	p := tea.NewProgram(model)
	if _, err := p.Run(); err != nil {
		return fmt.Errorf("TUI error: %w", err)
	}
	return nil
}

// resolveRepoRoot walks up from orchDir to find the repo root.
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
	return filepath.Join(orchDir, "..", "..", "..")
}

// appModel wraps the ui.Model with sub-model composition, watcher integration,
// and state refresh logic. This bridges ui and ui/panes to avoid import cycles.
type appModel struct {
	ui       ui.Model
	watcher  *watcher.Watcher
	executor *action.Executor
	orchDir  string
	wtBase   string
	planDir  string
	tailer   *watcher.Tailer

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

	m.sliceList.SetFocused(true)
	if s, ok := m.sliceList.SelectedSlice(); ok {
		m.detail.SetSlice(&s)
		m.deps.SetSelected(s.Name)
	}
	m.syncPaneContents()
	return m
}

func (m *appModel) Init() tea.Cmd {
	cmds := []tea.Cmd{m.watcher.Watch()}
	if s, ok := m.sliceList.SelectedSlice(); ok && s.LogPath != "" {
		if t, err := watcher.NewTailer(s.Name, s.LogPath); err == nil {
			m.tailer = t
			cmds = append(cmds, t.Tail())
		}
	} else {
		if t, err := watcher.NewTailer("", os.DevNull); err == nil {
			m.tailer = t
		}
	}
	return tea.Batch(cmds...)
}

func (m *appModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		innerModel, cmd := m.ui.Update(msg)
		m.ui = innerModel.(ui.Model)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
		m.resizePanes()
		m.syncPaneContents()
		return m, tea.Batch(cmds...)

	case watcher.StateChangedMsg:
		if status, err := state.ReadFullStatus(m.orchDir, m.wtBase, m.planDir); err == nil {
			m.sliceList.SetSlices(status.Slices)
			m.deps.SetDeps(status.Dependencies, status.Slices)
			m.progress.SetSlices(status.Slices)
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
		if m.tailer != nil {
			cmds = append(cmds, m.tailer.Tail())
		}
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
		m.actions, _ = m.actions.Update(msg)
		if s.LogPath != "" && m.tailer != nil {
			_ = m.tailer.SwitchFile(s.Name, s.LogPath)
		}
		m.syncPaneContents()
		return m, nil
	}

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

	if kmsg, ok := msg.(tea.KeyPressMsg); ok {
		if m.ui.Confirm.Visible {
			innerModel, cmd := m.ui.Update(kmsg)
			m.ui = innerModel.(ui.Model)
			if cmd != nil {
				cmds = append(cmds, cmd)
			}
			return m, tea.Batch(cmds...)
		}

		switch m.ui.Focused {
		case ui.PaneSlices:
			var cmd tea.Cmd
			m.sliceList, cmd = m.sliceList.Update(kmsg)
			if cmd != nil {
				cmds = append(cmds, cmd)
			}
		case ui.PaneLogs:
			var cmd tea.Cmd
			m.logView, cmd = m.logView.Update(kmsg)
			if cmd != nil {
				cmds = append(cmds, cmd)
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
			_ = consumed
		}

		prevFocused := m.ui.Focused
		innerModel, cmd := m.ui.Update(kmsg)
		m.ui = innerModel.(ui.Model)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
		if m.ui.Focused != prevFocused {
			m.syncFocus()
		}
		m.syncPaneContents()

		if m.ui.Quitting {
			return m, tea.Quit
		}
		return m, tea.Batch(cmds...)
	}

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

func (m *appModel) syncFocus() {
	m.sliceList.SetFocused(m.ui.Focused == ui.PaneSlices)
	m.detail.SetFocused(m.ui.Focused == ui.PaneDetail)
	m.deps.SetFocused(m.ui.Focused == ui.PaneDeps)
	m.logView.SetFocused(m.ui.Focused == ui.PaneLogs)
	m.actions = m.actions.SetFocused(m.ui.Focused == ui.PaneActions)
}

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

	m.sliceList.SetSize(slicesW-2, upperHeight-2)
	m.detail.SetSize(detailW-2, upperHeight-2)
	m.deps.SetSize(depsW-2, upperHeight-2)
	m.actions = m.actions.SetSize(actionsW-2, lowerHeight-2)
	m.logView.SetSize(logsW-2, lowerHeight-2)
	m.progress.SetWidth(w)
}
