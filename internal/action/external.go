package action

import (
	"os"
	"os/exec"

	tea "charm.land/bubbletea/v2"
)

// OpenPager opens the log file in the user's preferred pager ($PAGER, default: less).
// Uses tea.ExecProcess to suspend the TUI while the pager runs.
func (e *Executor) OpenPager(logPath string) tea.Cmd {
	pager := os.Getenv("PAGER")
	if pager == "" {
		pager = "less"
	}
	c := exec.Command(pager, logPath)
	return tea.ExecProcess(c, func(err error) tea.Msg {
		return ExternalDoneMsg{Action: "pager", Err: err}
	})
}

// OpenEditor opens the worktree path in the user's preferred editor ($EDITOR, default: vi).
// Uses tea.ExecProcess to suspend the TUI while the editor runs.
func (e *Executor) OpenEditor(worktreePath string) tea.Cmd {
	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = "vi"
	}
	c := exec.Command(editor, worktreePath)
	return tea.ExecProcess(c, func(err error) tea.Msg {
		return ExternalDoneMsg{Action: "editor", Err: err}
	})
}
