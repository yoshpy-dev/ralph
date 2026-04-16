package action

import (
	"os"
	"os/exec"
	"strings"

	tea "charm.land/bubbletea/v2"
)

// OpenPager opens the log file in the user's preferred pager ($PAGER, default: less).
// Uses tea.ExecProcess to suspend the TUI while the pager runs.
func (e *Executor) OpenPager(logPath string) tea.Cmd {
	pager := strings.TrimSpace(os.Getenv("PAGER"))
	if pager == "" {
		pager = "less"
	}
	parts := strings.Fields(pager)
	c := exec.Command(parts[0], append(parts[1:], logPath)...)
	return tea.ExecProcess(c, func(err error) tea.Msg {
		return ExternalDoneMsg{Action: "pager", Err: err}
	})
}

// OpenEditor opens the worktree path in the user's preferred editor ($EDITOR, default: vi).
// Uses tea.ExecProcess to suspend the TUI while the editor runs.
func (e *Executor) OpenEditor(worktreePath string) tea.Cmd {
	editor := strings.TrimSpace(os.Getenv("EDITOR"))
	if editor == "" {
		editor = "vi"
	}
	parts := strings.Fields(editor)
	c := exec.Command(parts[0], append(parts[1:], worktreePath)...)
	return tea.ExecProcess(c, func(err error) tea.Msg {
		return ExternalDoneMsg{Action: "editor", Err: err}
	})
}
