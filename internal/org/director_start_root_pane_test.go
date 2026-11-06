package org

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"slices"
	"strings"
	"testing"

	"github.com/yoshpy-dev/ralph/internal/org/driver"
)

// The tests of the headless director in the root pane of its workspace
// (plan 2026-11-06-org-herdr-single-tab, D6, D7, and AC8): the order of
// start's herdr calls and records, a pane rename that fails, a tab rename
// that fails, and a workspace reply with no root pane. The stop side of
// AC8 (this version's pane, an earlier version's, an unlabelled pane) is
// TestStopDirector_ConfirmsThePaneByItsLabel in confirm_pane_test.go.

// logDirectorWrites makes every write to f's director.jsonl log, into the
// fake herdr's calls, "write <event>" for each line it writes, so a test
// reads the herdr calls and the records in one order.
func logDirectorWrites(t *testing.T, f *dirStartFixture) {
	t.Helper()
	f.store.write = func(file *os.File, b []byte) (int, error) {
		for _, line := range strings.Split(strings.TrimSuffix(string(b), "\n"), "\n") {
			var ev DirectorEvent
			if err := json.Unmarshal([]byte(line), &ev); err != nil {
				t.Errorf("director.jsonl line %q: %v", line, err)
			}
			f.h.mu.Lock()
			f.h.calls = append(f.h.calls, "write "+ev.Event)
			f.h.mu.Unlock()
		}
		return file.Write(b)
	}
}

// AC8, the start: the workspace create, with no extraEnv (driver.Herdr adds
// its PaneEnv for the root pane), then its record, then the record of the
// root pane, then the pane's label DirectorTabLabel and its tab's label
// DirectorWorkspaceLabel, then the agent start in that pane, then launched.
// No tab create.
func TestStartDirector_RecordsTheRootPaneThenLabelsIt(t *testing.T) {
	const token = directorTestToken
	f := newDirStartFixture(t, token)
	logDirectorWrites(t, f)
	res := f.mustStart(t, DirectorDriverClaude, "opus")

	want := []string{"write attached", "workspace_create", "write workspace_created", "write pane_created",
		"pane_rename", "tab_rename", "agent_start", "write launched"}
	if !slices.Equal(f.h.calls, want) {
		t.Fatalf("herdr calls and records = %q, want %q", f.h.calls, want)
	}
	if len(f.h.workspaceCreateEnvs) != 1 || f.h.workspaceCreateEnvs[0] != nil || len(f.h.tabCreateWorkspaces) != 0 {
		t.Fatalf("workspace create extraEnv %q, tab creates %v; want one workspace create with none and no tab create",
			f.h.workspaceCreateEnvs, f.h.tabCreateWorkspaces)
	}
	if reg := res.Registration; reg.PaneID != "pane-1" || reg.WorkspaceID != "ws-1" || f.h.paneWorkspaces["pane-1"] != "ws-1" ||
		!slices.Equal(f.h.agentStartPanes, []string{"pane-1"}) {
		t.Fatalf("registration %+v, agent started in %v; want pane-1, the root pane of ws-1", reg, f.h.agentStartPanes)
	}
	if !slices.Equal(f.h.paneRenameCalls, []renameCall{{id: "pane-1", label: DirectorTabLabel(token)}}) ||
		!slices.Equal(f.h.tabRenameCalls, []renameCall{{id: fakeTabID("pane-1"), label: DirectorWorkspaceLabel(token)}}) {
		t.Fatalf("pane renames %v, tab renames %v; want pane-1 labelled %s and its tab %s",
			f.h.paneRenameCalls, f.h.tabRenameCalls, DirectorTabLabel(token), DirectorWorkspaceLabel(token))
	}
}

// AC8 and D6: a pane rename that fails fails the start, which then closes
// the workspace it made, a close that takes the root pane with it, and
// records the stop. The unlabelled pane is not checked or sent anything on
// its own: no pane get, no C-c, no pane close. herdr has neither the pane
// nor the workspace afterwards. When that workspace close fails, start
// records the registration detached only and names the workspace close
// alone as the herdr command, which closes the pane too.
func TestStartDirector_APaneRenameThatFailsClosesTheWorkspaceItMade(t *testing.T) {
	boom := errors.New("herdr: connection refused")
	label := DirectorTabLabel(directorTestToken)
	t.Run("the workspace closes", func(t *testing.T) {
		f := newDirStartFixture(t, directorTestToken)
		f.h.closeRemoves = true
		f.h.paneRenameErr = boom
		_, err := f.o.StartDirector(f.params(DirectorDriverClaude, "opus"))
		var ce *DirectorCleanupError
		if !errors.Is(err, boom) || errors.As(err, &ce) {
			t.Fatalf("err = %v, want the pane rename's error and a cleanup that finished", err)
		}
		for _, want := range []string{"org: director: start d1: herdr pane rename pane-1 " + label + ": ", "d1 is recorded stopped"} {
			if !strings.Contains(err.Error(), want) {
				t.Fatalf("err = %v, want it to say %q", err, want)
			}
		}
		if want := []string{"workspace_create", "pane_rename", "workspace_get", "workspace_close"}; !slices.Equal(f.h.calls, want) {
			t.Fatalf("herdr calls = %v, want %v", f.h.calls, want)
		}
		if _, ok := f.h.paneTabs["pane-1"]; ok {
			t.Fatalf("herdr still has pane-1 (in %q), want it closed with its workspace", f.h.paneTabs["pane-1"])
		}
		if _, ok := f.h.workspaceLabels["ws-1"]; ok || !slices.Equal(f.h.workspaceCloseCalls, []string{"ws-1"}) {
			t.Fatalf("herdr still has ws-1 (closes %v), want it closed", f.h.workspaceCloseCalls)
		}
		wantTrail(t, f.store, "attached d1", "workspace_created d1", "pane_created d1", "detached d1", "stopped d1")
		var paneCreated DirectorEvent
		if err := json.Unmarshal([]byte(directorFileLines(t, f.store)[2]), &paneCreated); err != nil || paneCreated.PaneID != "pane-1" {
			t.Fatalf("pane_created = %+v (%v), want the root pane pane-1, recorded before its label", paneCreated, err)
		}
		wantNoDirectorOpen(t, f.store)
	})
	t.Run("the workspace close fails", func(t *testing.T) {
		f := newDirStartFixture(t, directorTestToken)
		f.h.paneRenameErr = boom
		closeErr := errors.New("herdr workspace close: connection refused")
		f.h.workspaceCloseErrs = map[string]error{"ws-1": closeErr}
		_, err := f.o.StartDirector(f.params(DirectorDriverClaude, "opus"))
		var ce *DirectorCleanupError
		if !errors.As(err, &ce) || !errors.Is(err, boom) || !errors.Is(err, closeErr) || ce.Unconfirmed || !ce.Detached ||
			ce.PaneID != "pane-1" || !slices.Equal(ce.HerdrCommands, []string{"herdr workspace close ws-1"}) ||
			ce.StopCommand != "ralph org director stop --id d1" {
			t.Fatalf("err = %#v, want a cleanup error with the workspace close alone as the herdr command", err)
		}
		if want := []string{"workspace_create", "pane_rename", "workspace_get", "workspace_close"}; !slices.Equal(f.h.calls, want) {
			t.Fatalf("herdr calls = %v, want %v", f.h.calls, want)
		}
		wantTrail(t, f.store, "attached d1", "workspace_created d1", "pane_created d1", "detached d1")
	})
}

// AC8 and D6: a tab rename that fails changes only what herdr shows, so
// start goes on: the director starts in its labelled pane, launched is
// recorded, and start writes one line on Notice (none when Notice is nil).
// stop then confirms the pane by its own label, its tab unread, and closes
// it and the workspace.
func TestStartDirector_ATabRenameThatFailsStillStartsTheDirector(t *testing.T) {
	const token = directorTestToken
	boom := errors.New("herdr: connection refused")
	for _, withNotice := range []bool{true, false} {
		name := "with a notice writer"
		if !withNotice {
			name = "with no notice writer"
		}
		t.Run(name, func(t *testing.T) {
			f := newDirStartFixture(t, token)
			f.h.tabRenameErr = boom
			p := f.params(DirectorDriverClaude, "opus")
			var notice bytes.Buffer
			if withNotice {
				p.Notice = &notice
			}
			res, err := f.o.StartDirector(p)
			if err != nil || res.Registration.State != DirectorStateRunning || res.Registration.PaneID != "pane-1" {
				t.Fatalf("StartDirector = %+v, %v; want d1 running in pane-1", res.Registration, err)
			}
			if want := withAgentStart(); !slices.Equal(f.h.calls, want) {
				t.Fatalf("herdr calls = %v, want %v", f.h.calls, want)
			}
			if f.h.paneLabels["pane-1"] != DirectorTabLabel(token) || f.h.tabLabels[fakeTabID("pane-1")] != "1" {
				t.Fatalf("pane label %q, tab label %q; want the pane labelled %s and the tab as herdr labelled it",
					f.h.paneLabels["pane-1"], f.h.tabLabels[fakeTabID("pane-1")], DirectorTabLabel(token))
			}
			wantNotice := ""
			if withNotice {
				wantNotice = "d1: herdr tab rename " + fakeTabID("pane-1") + " " + DirectorWorkspaceLabel(token) +
					" failed, so the tab of its pane pane-1 keeps herdr's label; the director starts anyway: herdr: connection refused\n"
			}
			if notice.String() != wantNotice {
				t.Fatalf("notice = %q, want %q", notice.String(), wantNotice)
			}
			wantTrail(t, f.store, "attached d1", "workspace_created d1", "pane_created d1", "launched d1")

			n := len(f.h.calls)
			if stopped, err := f.stop("d1"); err != nil || stopped.PaneNote != "pane=closed" || stopped.WorkspaceNote != "workspace=closed" {
				t.Fatalf("stop = %+v, %v; want the pane and the workspace closed", stopped, err)
			}
			if got := herdrCallsSince(f.h, n); !slices.Equal(got, directorCloseCalls) {
				t.Fatalf("stop's herdr calls = %v, want %v", got, directorCloseCalls)
			}
		})
	}
}

// rootlessHerdr is a directorHerdr whose WorkspaceCreate answers with the
// ids strip leaves, as a herdr reply that names no root pane or no tab.
type rootlessHerdr struct {
	*directorHerdr
	strip func(ws *driver.HerdrWorkspace)
}

func (h *rootlessHerdr) WorkspaceCreate(ctx context.Context, cwd, label string, extraEnv ...string) (driver.HerdrWorkspace, error) {
	ws, err := h.directorHerdr.WorkspaceCreate(ctx, cwd, label, extraEnv...)
	h.strip(&ws)
	return ws, err
}

// D7: a workspace reply that names no root pane or no tab fails the start
// after the workspace is recorded and before any pane is: start labels
// nothing, records no pane, closes the workspace, and records the stop.
func TestStartDirector_AWorkspaceReplyWithNoRootPaneOrTabFails(t *testing.T) {
	for name, strip := range map[string]func(ws *driver.HerdrWorkspace){
		"no root pane": func(ws *driver.HerdrWorkspace) { ws.RootPaneID = "" },
		"no tab":       func(ws *driver.HerdrWorkspace) { ws.TabID = "" },
	} {
		t.Run(name, func(t *testing.T) {
			f := newDirStartFixture(t, directorTestToken)
			f.o.Herdr = &rootlessHerdr{directorHerdr: f.h, strip: strip}
			_, err := f.o.StartDirector(f.params(DirectorDriverClaude, "opus"))
			if err == nil || !strings.Contains(err.Error(), `herdr's reply for workspace "ws-1" names no root pane or no tab`) ||
				!strings.Contains(err.Error(), "d1 is recorded stopped") {
				t.Fatalf("err = %v, want the reply's refusal and a cleanup that finished", err)
			}
			if want := []string{"workspace_create", "workspace_get", "workspace_close"}; !slices.Equal(f.h.calls, want) {
				t.Fatalf("herdr calls = %v, want %v", f.h.calls, want)
			}
			wantTrail(t, f.store, "attached d1", "workspace_created d1", "detached d1", "stopped d1")
		})
	}
}
