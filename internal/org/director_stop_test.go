package org

import (
	"context"
	"errors"
	"os"
	"slices"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/yoshpy-dev/ralph/internal/org/driver"
)

// The StopDirector tests. The fake herdr (directorHerdr) and the fixture
// (dirStartFixture) they share with the StartDirector tests are in
// director_start_test.go.

// TestStopDirector_ClosesThePaneAndWorkspaceAndRecordsTheStop pins AC8: a
// confirmed pane gets C-c and is closed, then the workspace; a current
// registration gets detached and stopped, one already detached stopped only.
func TestStopDirector_ClosesThePaneAndWorkspaceAndRecordsTheStop(t *testing.T) {
	for _, detachFirst := range []bool{false, true} {
		name := "current"
		if detachFirst {
			name = "detached but open"
		}
		t.Run(name, func(t *testing.T) {
			f := newDirStartFixture(t, directorTestToken)
			f.mustStart(t, DirectorDriverClaude, "opus")
			trail := []string{"attached d1", "workspace_created d1", "pane_created d1", "launched d1"}
			if detachFirst {
				if _, err := f.store.Detach(DirectorDetach{ID: "d1"}); err != nil {
					t.Fatalf("Detach: %v", err)
				}
				trail = append(trail, "detached d1")
			}
			n := len(f.h.calls)
			res, err := f.stop("d1")
			if err != nil {
				t.Fatalf("stop: %v", err)
			}
			if res.WasCurrent == detachFirst || res.Registration.ID != "d1" || res.Registration.State != DirectorStateRunning ||
				res.CtrlCNote != "ctrl_c=ok" || res.PaneNote != "pane=closed" || res.WorkspaceNote != "workspace=closed" || res.UnrecordedWorkspaceLabel != "" {
				t.Fatalf("stop result = %+v", res)
			}
			if got := herdrCallsSince(f.h, n); !slices.Equal(got, directorCloseCalls) {
				t.Fatalf("herdr calls = %v, want %v", got, directorCloseCalls)
			}
			if len(f.h.sendKeysKeys) != 1 || !slices.Equal(f.h.sendKeysKeys[0], []string{"C-c"}) || !slices.Equal(f.h.sendKeysCalls, []string{"pane-1"}) ||
				!slices.Equal(f.h.paneCloseCalls, []string{"pane-1"}) || !slices.Equal(f.h.workspaceCloseCalls, []string{"ws-1"}) {
				t.Fatalf("C-c %v, pane closes %v, workspace closes %v", f.h.sendKeysKeys, f.h.paneCloseCalls, f.h.workspaceCloseCalls)
			}
			if !detachFirst {
				trail = append(trail, "detached d1")
			}
			wantTrail(t, f.store, append(trail, "stopped d1")...)
			wantNoDirectorOpen(t, f.store)
		})
	}
}

// TestStopDirector_WhatHerdrNoLongerKnowsCountsAsClosed pins AC8's last
// sentence: a pane or workspace herdr does not know, at the check or at the
// close, counts as closed and the stop is recorded.
func TestStopDirector_WhatHerdrNoLongerKnowsCountsAsClosed(t *testing.T) {
	t.Run("at the check", func(t *testing.T) {
		f := newDirStartFixture(t, directorTestToken)
		f.mustStart(t, DirectorDriverClaude, "opus")
		delete(f.h.paneTabs, "pane-1")
		delete(f.h.workspaceLabels, "ws-1")
		n := len(f.h.calls)
		res, err := f.stop("d1")
		if err != nil || res.PaneNote != "pane=already closed" || res.WorkspaceNote != "workspace=already closed" || res.CtrlCNote != "ctrl_c=skipped" {
			t.Fatalf("stop = %+v, %v", res, err)
		}
		if got := herdrCallsSince(f.h, n); !slices.Equal(got, []string{"pane_get", "workspace_get"}) {
			t.Fatalf("herdr calls = %v, want the two checks only", got)
		}
		wantNoDirectorOpen(t, f.store)
	})
	t.Run("at the close", func(t *testing.T) {
		f := newDirStartFixture(t, directorTestToken)
		f.mustStart(t, DirectorDriverClaude, "opus")
		f.h.paneCloseErrs = map[string]error{"pane-1": driver.NewHerdrError(driver.HerdrCodePaneNotFound, "gone")}
		f.h.workspaceCloseErrs = map[string]error{"ws-1": driver.NewHerdrError(driver.HerdrCodeWorkspaceNotFound, "gone")}
		res, err := f.stop("d1")
		if err != nil || res.PaneNote != "pane=already closed" || res.WorkspaceNote != "workspace=already closed" {
			t.Fatalf("stop = %+v, %v", res, err)
		}
		wantNoDirectorOpen(t, f.store)
	})
}

// TestStopDirector_ALabelNotThisLaunchsSendsNothing pins AC9's label rule:
// when the tab, the pane's workspace, or the recorded workspace is not
// labelled with this launch's names, stop sends no C-c, closes nothing,
// writes nothing, and names the labels it expected and found.
func TestStopDirector_ALabelNotThisLaunchsSendsNothing(t *testing.T) {
	cases := []struct {
		name     string
		relabel  func(h *directorHerdr)
		wantText []string
	}{
		{"the tab", func(h *directorHerdr) { setLabel(&h.tabLabels, fakeTabID("pane-1"), "director_ffffffffffff") },
			[]string{"a tab labelled director_" + directorTestToken, `labelled "director_ffffffffffff"`}},
		{"the pane's workspace", func(h *directorHerdr) { setLabel(&h.workspaceLabels, "ws-1", "ralph_director_ffffffffffff") },
			[]string{"a workspace labelled ralph_director_" + directorTestToken, `labels it "ralph_director_ffffffffffff"`}},
		{"the recorded workspace, its pane gone", func(h *directorHerdr) {
			delete(h.paneTabs, "pane-1")
			setLabel(&h.workspaceLabels, "ws-1", "org-a")
		}, []string{`workspace "ws-1" is not confirmed as d1's (labelled ralph_director_` + directorTestToken + ")", `labels it "org-a"`}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := newDirStartFixture(t, directorTestToken)
			f.mustStart(t, DirectorDriverClaude, "opus")
			tc.relabel(f.h)
			before := directorFileState(t, f.store)
			_, err := f.stop("d1")
			var ce *DirectorCleanupError
			if !errors.As(err, &ce) || !ce.Unconfirmed || ce.Verb != "stop" || ce.Err != nil || ce.Detached {
				t.Fatalf("err = %#v, want an unconfirmed stop", err)
			}
			// The ids stay unconfirmed on a retry, so the way out records the
			// stop with --force (D11).
			if ce.StopCommand != "ralph org director stop --id d1 --force" {
				t.Fatalf("StopCommand = %q, want the stop with --force", ce.StopCommand)
			}
			for _, want := range append(tc.wantText, "nothing was recorded", "herdr workspace list", "afterwards record the stop with ralph org director stop --id d1 --force") {
				if !strings.Contains(err.Error(), want) {
					t.Fatalf("err = %v, want it to say %q", err, want)
				}
			}
			// The label checks are the seats' (confirmSeatPane,
			// confirmOrgWorkspace), but a director is no seat and has no org:
			// the mismatch names the labels only (self-review L4).
			for _, word := range []string{"seat", "org_id"} {
				if strings.Contains(err.Error(), word) {
					t.Fatalf("err = %v, want it not to say %q", err, word)
				}
			}
			if len(f.h.sendKeysCalls) != 0 || len(f.h.paneCloseCalls) != 0 || len(f.h.workspaceCloseCalls) != 0 {
				t.Fatalf("C-c %v, pane closes %v, workspace closes %v, want none", f.h.sendKeysCalls, f.h.paneCloseCalls, f.h.workspaceCloseCalls)
			}
			if after := directorFileState(t, f.store); after != before {
				t.Fatalf("director.jsonl changed:\n%s", after)
			}
		})
	}
}

// TestStopDirector_ACloseThatFailsRecordsNothing pins AC9's close rule: a
// pane or workspace herdr cannot close leaves the registration as it was,
// and the error names the herdr commands; a retry once herdr answers stops
// it.
func TestStopDirector_ACloseThatFailsRecordsNothing(t *testing.T) {
	closeErr := errors.New("herdr: connection refused")
	cases := []struct {
		name         string
		inject       func(h *directorHerdr)
		wantCommands []string
		wantWsCloses []string
	}{
		{"the pane", func(h *directorHerdr) { h.paneCloseErrs = map[string]error{"pane-1": closeErr} },
			[]string{"herdr pane close pane-1", "herdr workspace close ws-1"}, nil},
		{"the workspace", func(h *directorHerdr) { h.workspaceCloseErrs = map[string]error{"ws-1": closeErr} },
			[]string{"herdr workspace close ws-1"}, []string{"ws-1"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := newDirStartFixture(t, directorTestToken)
			f.mustStart(t, DirectorDriverClaude, "opus")
			tc.inject(f.h)
			before := directorFileState(t, f.store)
			_, err := f.stop("d1")
			var ce *DirectorCleanupError
			if !errors.As(err, &ce) || ce.Unconfirmed || !errors.Is(err, closeErr) || !slices.Equal(ce.HerdrCommands, tc.wantCommands) ||
				ce.StopCommand != "ralph org director stop --id d1" {
				t.Fatalf("err = %#v, want a close failure with commands %q and the stop without --force", err, tc.wantCommands)
			}
			if !slices.Equal(f.h.workspaceCloseCalls, tc.wantWsCloses) {
				t.Fatalf("workspace closes = %v, want %v", f.h.workspaceCloseCalls, tc.wantWsCloses)
			}
			if after := directorFileState(t, f.store); after != before {
				t.Fatalf("director.jsonl changed:\n%s", after)
			}
			f.h.paneCloseErrs, f.h.workspaceCloseErrs = nil, nil
			if _, err := f.stop("d1"); err != nil {
				t.Fatalf("retry: %v", err)
			}
			wantNoDirectorOpen(t, f.store)
		})
	}
}

// TestStopDirector_Refusals pins AC9's refusals: each writes nothing and
// calls no herdr.
func TestStopDirector_Refusals(t *testing.T) {
	cases := []struct {
		name      string
		setup     func(t *testing.T, f *dirStartFixture) DirectorStopParams
		wantErrIs []error
		wantText  string
	}{
		{"no id", func(_ *testing.T, f *dirStartFixture) DirectorStopParams { return f.stopParams("") }, nil, "needs the id"},
		{"not an id", func(_ *testing.T, f *dirStartFixture) DirectorStopParams { return f.stopParams("x1") }, nil, "not a registration id"},
		{"no store", func(*testing.T, *dirStartFixture) DirectorStopParams { return DirectorStopParams{ID: "d1"} }, nil, "no director store"},
		{"no director.jsonl", func(_ *testing.T, f *dirStartFixture) DirectorStopParams { return f.stopParams("d1") },
			[]error{ErrDirectorNoOpenHeadless}, "no registration d1"},
		{"an unknown id", func(t *testing.T, f *dirStartFixture) DirectorStopParams {
			f.mustStart(t, DirectorDriverClaude, "opus")
			return DirectorStopParams{Store: f.store, ID: "d9"}
		}, []error{ErrDirectorNoOpenHeadless}, "no registration d9"},
		{"a promoted session's registration", func(t *testing.T, f *dirStartFixture) DirectorStopParams {
			mustAttachDirector(t, f.store, DirectorAttach{Driver: DirectorDriverClaude})
			return DirectorStopParams{Store: f.store, ID: "d1"}
		}, []error{ErrDirectorNoOpenHeadless, ErrDirectorNotHeadless}, "not started in herdr"},
		{"already stopped", func(t *testing.T, f *dirStartFixture) DirectorStopParams {
			f.mustStart(t, DirectorDriverClaude, "opus")
			if _, err := f.stop("d1"); err != nil {
				t.Fatalf("first stop: %v", err)
			}
			return DirectorStopParams{Store: f.store, ID: "d1"}
		}, []error{ErrDirectorNoOpenHeadless, ErrDirectorStopped}, "already stopped"},
		{"inside the director's own pane", func(t *testing.T, f *dirStartFixture) DirectorStopParams {
			f.mustStart(t, DirectorDriverClaude, "opus")
			f.env[herdrPaneIDEnv] = "pane-1"
			return DirectorStopParams{Store: f.store, ID: "d1"}
		}, []error{ErrDirectorStopSelfPane}, "outside the director's pane"},
		{"inside the director's own pane, with force", func(t *testing.T, f *dirStartFixture) DirectorStopParams {
			f.mustStart(t, DirectorDriverClaude, "opus")
			f.env[herdrPaneIDEnv] = "pane-1"
			return DirectorStopParams{Store: f.store, ID: "d1", Force: true}
		}, []error{ErrDirectorStopSelfPane}, "outside the director's pane"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := newDirStartFixture(t, directorTestToken)
			p := tc.setup(t, f)
			before, n := directorFileState(t, f.store), len(f.h.calls)
			entries := dirEntryNames(t, f.dir)
			_, err := f.o.StopDirector(p)
			if err == nil || !strings.Contains(err.Error(), tc.wantText) {
				t.Fatalf("err = %v, want a refusal saying %q", err, tc.wantText)
			}
			for _, target := range tc.wantErrIs {
				if !errors.Is(err, target) {
					t.Fatalf("err = %v, want it to wrap %v", err, target)
				}
			}
			if got := herdrCallsSince(f.h, n); len(got) != 0 {
				t.Fatalf("a refusal called herdr: %v", got)
			}
			if after := directorFileState(t, f.store); after != before {
				t.Fatalf("director.jsonl changed from %q to %q", before, after)
			}
			if got := dirEntryNames(t, f.dir); !slices.Equal(got, entries) {
				t.Fatalf("ledger entries changed from %v to %v", entries, got)
			}
		})
	}
}

// TestDirector_TwoLedgersOnOneHerdrServerNeverCloseEachOther pins AC5: two
// ledgers start directors on one herdr server with different tokens, names,
// and labels; after herdr renumbers its ids so ledger A's recorded pane and
// workspace name ledger B's, A's stop sends nothing to them and writes
// nothing, and B's stop still works.
func TestDirector_TwoLedgersOnOneHerdrServerNeverCloseEachOther(t *testing.T) {
	a := newDirStartFixture(t)
	b := newDirStartFixture(t)
	b.o.Herdr = a.h
	startA := a.mustStart(t, DirectorDriverClaude, "opus")

	// herdr loses its state and hands out the same ids again.
	h := a.h
	h.paneTabs, h.paneWorkspaces, h.tabLabels, h.workspaceLabels = nil, nil, nil, nil
	startB := b.mustStart(t, DirectorDriverClaude, "opus")

	ra, rb := startA.Registration, startB.Registration
	if ra.ID != "d1" || rb.ID != "d1" || ra.PaneID != rb.PaneID || ra.WorkspaceID != rb.WorkspaceID {
		t.Fatalf("fixture: A %+v, B %+v, want both d1 on the same renumbered ids", ra, rb)
	}
	if ra.Token == rb.Token || startA.AgentName == startB.AgentName || startA.WorkspaceLabel == startB.WorkspaceLabel || startA.TabLabel == startB.TabLabel {
		t.Fatalf("A %+v and B %+v share a token or a name", startA, startB)
	}
	if !slices.Equal(h.agentStartNames, []string{startA.AgentName, startB.AgentName}) {
		t.Fatalf("agent names = %v", h.agentStartNames)
	}

	beforeA, beforeB := directorFileState(t, a.store), directorFileState(t, b.store)
	_, err := a.stop("d1")
	var ce *DirectorCleanupError
	if !errors.As(err, &ce) || !ce.Unconfirmed || !strings.Contains(err.Error(), startB.TabLabel) {
		t.Fatalf("A's stop: %v, want an unconfirmed refusal naming B's label as found", err)
	}
	if len(h.sendKeysCalls) != 0 || len(h.paneCloseCalls) != 0 || len(h.workspaceCloseCalls) != 0 {
		t.Fatalf("A's stop sent C-c %v, pane closes %v, workspace closes %v to B's resources", h.sendKeysCalls, h.paneCloseCalls, h.workspaceCloseCalls)
	}
	if directorFileState(t, a.store) != beforeA || directorFileState(t, b.store) != beforeB {
		t.Fatal("A's refused stop changed a director.jsonl")
	}
	if _, err := b.stop("d1"); err != nil {
		t.Fatalf("B's stop: %v", err)
	}
	wantNoDirectorOpen(t, b.store)
}

// stopWithForce is f's stop of id with Force set.
func stopWithForce(f *dirStartFixture, id string) (DirectorStopResult, error) {
	p := f.stopParams(id)
	p.Force = true
	return f.o.StopDirector(p)
}

// wantNothingSent asserts h got no C-c and no pane or workspace close.
func wantNothingSent(t *testing.T, h *directorHerdr) {
	t.Helper()
	if len(h.sendKeysCalls) != 0 || len(h.paneCloseCalls) != 0 || len(h.workspaceCloseCalls) != 0 {
		t.Fatalf("C-c %v, pane closes %v, workspace closes %v, want none", h.sendKeysCalls, h.paneCloseCalls, h.workspaceCloseCalls)
	}
}

// findWorkspaceCommands are the herdr commands a forced stop of the launch
// with token gives for what it left open: the label, never a recorded id.
func findWorkspaceCommands(token string) []string {
	return []string{"herdr workspace list", "herdr workspace close <the id of the workspace labelled ralph_director_" + token + ">"}
}

// TestStopDirector_ForceRecordsAStopWhoseLabelsAreAnotherLaunchs pins AC14
// on AC5's two ledgers: after herdr renumbered its ids, ledger A's stop with
// Force sends no C-c and no close to the pane and workspace its ids now
// name (ledger B's), records stopped with force, detached first since A's
// registration is current, and finds A's workspace by its label. B's
// director.jsonl stays as it was, and B's own stop still closes B's.
func TestStopDirector_ForceRecordsAStopWhoseLabelsAreAnotherLaunchs(t *testing.T) {
	const tokenB = "ba0000000000"
	a := newDirStartFixture(t, directorTestToken)
	b := newDirStartFixture(t, tokenB)
	b.o.Herdr = a.h
	a.mustStart(t, DirectorDriverClaude, "opus")
	h := a.h
	h.paneTabs, h.paneWorkspaces, h.tabLabels, h.workspaceLabels = nil, nil, nil, nil
	b.mustStart(t, DirectorDriverClaude, "opus")
	beforeB, n := directorFileState(t, b.store), len(h.calls)

	res, err := stopWithForce(a, "d1")
	if err != nil {
		t.Fatalf("A's stop --force: %v", err)
	}
	if !res.Forced || !res.WasCurrent || res.CloseErr == nil || !strings.Contains(res.CloseErr.Error(), `labelled "director_`+tokenB+`"`) ||
		res.CtrlCNote != "ctrl_c=skipped: not confirmed" || res.PaneNote != "pane=not closed: not confirmed" || res.WorkspaceNote != "workspace=not closed: not confirmed" {
		t.Fatalf("A's stop --force = %+v, want a forced stop that names B's label as found and closed nothing", res)
	}
	if !slices.Equal(res.HerdrCommands, findWorkspaceCommands(directorTestToken)) {
		t.Fatalf("HerdrCommands = %q, want A's workspace found by its label", res.HerdrCommands)
	}
	if got := herdrCallsSince(h, n); !slices.Equal(got, []string{"pane_get", "tab_get"}) {
		t.Fatalf("herdr calls = %v, want the pane check only", got)
	}
	wantNothingSent(t, h)
	wantTrail(t, a.store, "attached d1", "workspace_created d1", "pane_created d1", "launched d1", "detached d1", "stopped d1+force")
	wantNoDirectorOpen(t, a.store)
	if directorFileState(t, b.store) != beforeB {
		t.Fatal("A's forced stop changed B's director.jsonl")
	}

	if _, err := b.stop("d1"); err != nil {
		t.Fatalf("B's stop: %v", err)
	}
	if !slices.Equal(h.paneCloseCalls, []string{"pane-1"}) || !slices.Equal(h.workspaceCloseCalls, []string{"ws-1"}) {
		t.Fatalf("B's stop: pane closes %v, workspace closes %v, want B's pane-1 and ws-1", h.paneCloseCalls, h.workspaceCloseCalls)
	}
	wantTrail(t, b.store, "attached d1", "workspace_created d1", "pane_created d1", "launched d1", "detached d1", "stopped d1")
}

// TestStopDirector_ForceWithLabelsThatMatchClosesAsWithoutIt pins AC14's
// last rule: with this launch's labels, Force changes nothing: C-c, the
// pane and workspace closes, a stopped without force, and Forced false.
func TestStopDirector_ForceWithLabelsThatMatchClosesAsWithoutIt(t *testing.T) {
	f := newDirStartFixture(t, directorTestToken)
	f.mustStart(t, DirectorDriverClaude, "opus")
	n := len(f.h.calls)
	res, err := stopWithForce(f, "d1")
	if err != nil || res.Forced || res.CloseErr != nil || res.HerdrCommands != nil || !res.WasCurrent ||
		res.CtrlCNote != "ctrl_c=ok" || res.PaneNote != "pane=closed" || res.WorkspaceNote != "workspace=closed" {
		t.Fatalf("stop --force = %+v, %v; want the plain stop", res, err)
	}
	if got := herdrCallsSince(f.h, n); !slices.Equal(got, directorCloseCalls) {
		t.Fatalf("herdr calls = %v, want %v", got, directorCloseCalls)
	}
	wantTrail(t, f.store, "attached d1", "workspace_created d1", "pane_created d1", "launched d1", "detached d1", "stopped d1")
	wantNoDirectorOpen(t, f.store)
}

// TestStopDirector_ForceAfterACloseThatFailsRecordsWhatWasLeftOpen pins
// AC14's close rule: when the pane close fails, Force sends no workspace
// close; when the workspace close fails, the pane it closed stays closed.
// Either way stopped is recorded with force and the result says what was
// left open and how to find it.
func TestStopDirector_ForceAfterACloseThatFailsRecordsWhatWasLeftOpen(t *testing.T) {
	closeErr := errors.New("herdr: connection refused")
	cases := []struct {
		name                         string
		inject                       func(h *directorHerdr)
		wantCalls                    []string
		wantPaneNote, wantWorkspNote string
		wantWsCloses                 []string
	}{
		{"the pane", func(h *directorHerdr) { h.paneCloseErrs = map[string]error{"pane-1": closeErr} },
			[]string{"pane_get", "tab_get", "workspace_get", "workspace_get", "pane_send_keys", "pane_close"},
			"pane=close failed: herdr: connection refused", "workspace=not closed: the pane was left open", nil},
		{"the workspace", func(h *directorHerdr) { h.workspaceCloseErrs = map[string]error{"ws-1": closeErr} },
			directorCloseCalls, "pane=closed", "workspace=close failed: herdr: connection refused", []string{"ws-1"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := newDirStartFixture(t, directorTestToken)
			f.mustStart(t, DirectorDriverClaude, "opus")
			tc.inject(f.h)
			n := len(f.h.calls)
			res, err := stopWithForce(f, "d1")
			if err != nil || !res.Forced || !res.WasCurrent || !errors.Is(res.CloseErr, closeErr) ||
				res.CtrlCNote != "ctrl_c=ok" || res.PaneNote != tc.wantPaneNote || res.WorkspaceNote != tc.wantWorkspNote ||
				!slices.Equal(res.HerdrCommands, findWorkspaceCommands(directorTestToken)) {
				t.Fatalf("stop --force = %+v, %v", res, err)
			}
			if got := herdrCallsSince(f.h, n); !slices.Equal(got, tc.wantCalls) {
				t.Fatalf("herdr calls = %v, want %v", got, tc.wantCalls)
			}
			if !slices.Equal(f.h.paneCloseCalls, []string{"pane-1"}) || !slices.Equal(f.h.workspaceCloseCalls, tc.wantWsCloses) {
				t.Fatalf("pane closes %v, workspace closes %v, want pane-1 once and %v", f.h.paneCloseCalls, f.h.workspaceCloseCalls, tc.wantWsCloses)
			}
			wantTrail(t, f.store, "attached d1", "workspace_created d1", "pane_created d1", "launched d1", "detached d1", "stopped d1+force")
			wantNoDirectorOpen(t, f.store)
		})
	}
}

// TestStopDirector_ForceWhenAGetTimesOutSendsNothing pins AC14 for a
// recorded id herdr does not answer about: the pane check times out, so
// nothing is sent, and Force records stopped with force alone for a
// registration already detached.
func TestStopDirector_ForceWhenAGetTimesOutSendsNothing(t *testing.T) {
	f := newDirStartFixture(t, directorTestToken)
	f.mustStart(t, DirectorDriverClaude, "opus")
	if _, err := f.store.Detach(DirectorDetach{ID: "d1"}); err != nil {
		t.Fatalf("Detach: %v", err)
	}
	f.o.DriverCallTimeout = 20 * time.Millisecond
	f.h.getBlock = true
	n := len(f.h.calls)
	res, err := stopWithForce(f, "d1")
	if err != nil || !res.Forced || res.WasCurrent || !errors.Is(res.CloseErr, context.DeadlineExceeded) ||
		res.PaneNote != "pane=not closed: not confirmed" || !slices.Equal(res.HerdrCommands, findWorkspaceCommands(directorTestToken)) {
		t.Fatalf("stop --force = %+v, %v; want a forced stop after the timed-out check", res, err)
	}
	if got := herdrCallsSince(f.h, n); !slices.Equal(got, []string{"pane_get"}) {
		t.Fatalf("herdr calls = %v, want the pane check only", got)
	}
	wantNothingSent(t, f.h)
	wantTrail(t, f.store, "attached d1", "workspace_created d1", "pane_created d1", "launched d1", "detached d1", "stopped d1+force")
	wantNoDirectorOpen(t, f.store)
}

// TestStopDirector_RefusesInsideTheDirectorsWorkspace pins AC9's workspace
// rule: once herdr's label confirms the recorded workspace as this
// launch's, a stop whose HERDR_WORKSPACE_ID is that workspace is refused,
// with or without Force, with the checks only: no C-c, no close, nothing
// written. A workspace recorded without a pane (a start that died before
// its tab) is refused the same way. HERDR_WORKSPACE_ID of another workspace
// does not refuse.
func TestStopDirector_RefusesInsideTheDirectorsWorkspace(t *testing.T) {
	cases := []struct {
		name      string
		setup     func(t *testing.T, f *dirStartFixture) string // returns the recorded workspace
		wantCalls []string
	}{
		{"running", func(t *testing.T, f *dirStartFixture) string {
			f.mustStart(t, DirectorDriverClaude, "opus")
			return "ws-1"
		}, []string{"pane_get", "tab_get", "workspace_get", "workspace_get"}},
		{"only the workspace recorded", func(t *testing.T, f *dirStartFixture) string {
			if _, err := f.store.AttachHeadless(testHeadlessAttach()); err != nil {
				t.Fatalf("AttachHeadless: %v", err)
			}
			if err := f.store.RecordWorkspace("d1", "ws-7", testDirectorWho); err != nil {
				t.Fatalf("RecordWorkspace: %v", err)
			}
			setLabel(&f.h.workspaceLabels, "ws-7", DirectorWorkspaceLabel(directorTestToken))
			return "ws-7"
		}, []string{"workspace_get"}},
	}
	for _, tc := range cases {
		for _, force := range []bool{false, true} {
			name := tc.name
			if force {
				name += ", with force"
			}
			t.Run(name, func(t *testing.T) {
				f := newDirStartFixture(t, directorTestToken)
				f.env[herdrWorkspaceIDEnv] = tc.setup(t, f)
				before, n := directorFileState(t, f.store), len(f.h.calls)
				p := f.stopParams("d1")
				p.Force = force
				_, err := f.o.StopDirector(p)
				if !errors.Is(err, ErrDirectorStopSelfWorkspace) || !strings.Contains(err.Error(), "outside the director's workspace") ||
					!strings.Contains(err.Error(), "labelled ralph_director_"+directorTestToken) {
					t.Fatalf("err = %v, want ErrDirectorStopSelfWorkspace naming the confirmed label", err)
				}
				var ce *DirectorCleanupError
				if errors.As(err, &ce) {
					t.Fatalf("err = %v, want a refusal, not a cleanup error", err)
				}
				if got := herdrCallsSince(f.h, n); !slices.Equal(got, tc.wantCalls) {
					t.Fatalf("herdr calls = %v, want the checks only: %v", got, tc.wantCalls)
				}
				wantNothingSent(t, f.h)
				if after := directorFileState(t, f.store); after != before {
					t.Fatalf("director.jsonl changed from %q to %q", before, after)
				}
			})
		}
	}
	t.Run("another workspace's id", func(t *testing.T) {
		f := newDirStartFixture(t, directorTestToken)
		f.mustStart(t, DirectorDriverClaude, "opus")
		f.env[herdrWorkspaceIDEnv] = "ws-9"
		if _, err := f.stop("d1"); err != nil {
			t.Fatalf("stop: %v", err)
		}
		wantNoDirectorOpen(t, f.store)
	})
	t.Run("a recorded workspace herdr no longer knows", func(t *testing.T) {
		// Nothing is sent to it, so nothing can end this command.
		f := newDirStartFixture(t, directorTestToken)
		f.mustStart(t, DirectorDriverClaude, "opus")
		delete(f.h.paneTabs, "pane-1")
		delete(f.h.workspaceLabels, "ws-1")
		f.env[herdrWorkspaceIDEnv] = "ws-1"
		if res, err := f.stop("d1"); err != nil || res.WorkspaceNote != "workspace=already closed" {
			t.Fatalf("stop = %+v, %v; want the workspace counted as closed", res, err)
		}
		wantNoDirectorOpen(t, f.store)
	})
}

// TestStopDirector_ARenumberedWorkspaceIDIsTheLabelMismatch pins that the
// workspace refusal waits for the label: when HERDR_WORKSPACE_ID is the
// recorded id but herdr now gives that id to another launch (AC5's
// renumbering), the stop is the label mismatch instead: refused with
// nothing written without Force, recorded with force with it, and nothing
// sent either way.
func TestStopDirector_ARenumberedWorkspaceIDIsTheLabelMismatch(t *testing.T) {
	for _, force := range []bool{false, true} {
		name := "without force"
		if force {
			name = "with force"
		}
		t.Run(name, func(t *testing.T) {
			f := newDirStartFixture(t, directorTestToken)
			f.mustStart(t, DirectorDriverClaude, "opus")
			// herdr renumbered: pane-1 is gone and ws-1 is another launch's.
			delete(f.h.paneTabs, "pane-1")
			setLabel(&f.h.workspaceLabels, "ws-1", DirectorWorkspaceLabel("ffffffffffff"))
			f.env[herdrWorkspaceIDEnv] = "ws-1"
			before := directorFileState(t, f.store)
			p := f.stopParams("d1")
			p.Force = force
			res, err := f.o.StopDirector(p)
			if errors.Is(err, ErrDirectorStopSelfWorkspace) {
				t.Fatalf("err = %v, want the label mismatch, not the workspace refusal", err)
			}
			wantNothingSent(t, f.h)
			if !force {
				var ce *DirectorCleanupError
				if !errors.As(err, &ce) || !ce.Unconfirmed {
					t.Fatalf("err = %#v, want an unconfirmed stop", err)
				}
				if after := directorFileState(t, f.store); after != before {
					t.Fatalf("director.jsonl changed:\n%s", after)
				}
				return
			}
			if err != nil || !res.Forced || !strings.Contains(res.CloseErr.Error(), `labels it "ralph_director_ffffffffffff"`) {
				t.Fatalf("stop --force = %+v, %v; want a forced stop over the mismatched workspace", res, err)
			}
			wantTrail(t, f.store, "attached d1", "workspace_created d1", "pane_created d1", "launched d1", "detached d1", "stopped d1+force")
		})
	}
}

// TestStopDirector_AForcedStopThatCannotRecordSaysToRetryWithForce pins the
// forced stop whose record fails: nothing is recorded, the
// *DirectorCleanupError joins why nothing was closed with why nothing was
// recorded, and its StopCommand carries --force, since a retry meets the
// same pane and workspace.
func TestStopDirector_AForcedStopThatCannotRecordSaysToRetryWithForce(t *testing.T) {
	closeErr := errors.New("herdr: connection refused")
	cases := []struct {
		name         string
		inject       func(h *directorHerdr)
		unconfirmed  bool
		wantCommands []string
	}{
		{"a label that is another launch's", func(h *directorHerdr) { setLabel(&h.tabLabels, fakeTabID("pane-1"), "director_ffffffffffff") },
			true, findWorkspaceCommands(directorTestToken)},
		{"a pane close that fails", func(h *directorHerdr) { h.paneCloseErrs = map[string]error{"pane-1": closeErr} },
			false, []string{"herdr pane close pane-1", "herdr workspace close ws-1"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := newDirStartFixture(t, directorTestToken)
			f.mustStart(t, DirectorDriverClaude, "opus")
			tc.inject(f.h)
			before := directorFileState(t, f.store)
			f.store.write = func(*os.File, []byte) (int, error) { return 0, syscall.ENOSPC }
			_, err := stopWithForce(f, "d1")
			var ce *DirectorCleanupError
			if !errors.As(err, &ce) || ce.Unconfirmed != tc.unconfirmed || !errors.Is(err, syscall.ENOSPC) ||
				ce.StopCommand != "ralph org director stop --id d1 --force" || !slices.Equal(ce.HerdrCommands, tc.wantCommands) {
				t.Fatalf("err = %#v, want a cleanup error that retries with --force", err)
			}
			if !tc.unconfirmed && !errors.Is(err, closeErr) {
				t.Fatalf("err = %v, want it to keep the close failure", err)
			}
			if !strings.Contains(err.Error(), "nothing was recorded") {
				t.Fatalf("err = %v, want it to say nothing was recorded", err)
			}
			if after := directorFileState(t, f.store); after != before {
				t.Fatalf("director.jsonl changed:\n%s", after)
			}
		})
	}
}

// wantChangedDuringStop asserts err is stop's refusal of a registration that
// a start recorded more for after stop read it (plan D14): a
// *DirectorCleanupError wrapping ErrDirectorChangedDuringStop with no herdr
// command and the stop to run again, whose text is want. notes are what
// this stop did to the pane and the workspace: res carries them, and the
// text does not repeat them (the CLI prints them on stderr).
func wantChangedDuringStop(t *testing.T, res DirectorStopResult, err error, want, notes string) {
	t.Helper()
	var ce *DirectorCleanupError
	if !errors.As(err, &ce) || !errors.Is(err, ErrDirectorChangedDuringStop) || ce.Verb != "stop" || ce.Detached || ce.Unconfirmed ||
		ce.HerdrCommands != nil || ce.StopCommand != "ralph org director stop --id d1" {
		t.Fatalf("err = %#v, want a stop refused with ErrDirectorChangedDuringStop that names the stop to run again", err)
	}
	if err.Error() != want {
		t.Fatalf("err =\n%v\nwant\n%v", err, want)
	}
	if got := res.CtrlCNote + ", " + res.PaneNote + ", " + res.WorkspaceNote; got != notes {
		t.Fatalf("stop result notes = %q, want %q", got, notes)
	}
}

// changedDuringStopText is the text of stop's refusal of d1 (plan D14), with
// what the ids were when stop read d1 and are now: "org: director: stop d1:"
// once, and none of this stop's notes.
func changedDuringStopText(then, now string) string {
	return "org: director: stop d1: record d1 stopped: a start in progress recorded a workspace or pane after stop read the registration: " +
		"d1 had " + then + " when stop read it, and has " + now + " now; this stop closed only what it had read; " +
		"nothing was recorded; run ralph org director stop --id d1 again to close what start recorded"
}

// TestStopDirector_RefusesWhenAStartRecordsMoreWhileItRuns pins AC9's last
// sentence (plan D14): when a start still running records a workspace or
// pane of the registration after stop read it, stop closes only what it read,
// writes neither stopped nor detached, leaves the director running and
// current, and its error says what it closed and to run stop again. Force
// changes nothing when the labels match. The stop run again is recorded.
func TestStopDirector_RefusesWhenAStartRecordsMoreWhileItRuns(t *testing.T) {
	for _, force := range []bool{false, true} {
		name := "the pane, while stop closes the workspace"
		if force {
			name += ", with force"
		}
		t.Run(name, func(t *testing.T) {
			f := newDirStartFixture(t, directorTestToken)
			if _, err := f.store.AttachHeadless(testHeadlessAttach()); err != nil {
				t.Fatalf("AttachHeadless: %v", err)
			}
			if err := f.store.RecordWorkspace("d1", "ws-7", testDirectorWho); err != nil {
				t.Fatalf("RecordWorkspace: %v", err)
			}
			setLabel(&f.h.workspaceLabels, "ws-7", DirectorWorkspaceLabel(directorTestToken))
			var paneID string
			f.h.workspaceCloseHook = func() {
				f.h.workspaceCloseHook = nil
				// Start's step 3 (its tab and pane) and step 6 (launched),
				// its agent start left out.
				var err error
				if paneID, err = f.h.TabCreate(context.Background(), "ws-7", f.cwd, DirectorTabLabel(directorTestToken)); err != nil {
					t.Errorf("TabCreate: %v", err)
					return
				}
				recordPaneAndLaunched(t, f.store, paneID)
			}
			p := f.stopParams("d1")
			p.Force = force
			res, err := f.o.StopDirector(p)

			wantChangedDuringStop(t, res, err, changedDuringStopText(`workspace "ws-7" and no pane`, `workspace "ws-7" and pane "pane-1"`),
				"ctrl_c=skipped, pane=none recorded, workspace=closed")
			if res.Registration.PaneID != "" || res.WorkspaceNote != "workspace=closed" || res.WasCurrent || res.Forced {
				t.Fatalf("stop result = %+v, want the registration as read and the workspace closed", res)
			}
			if !slices.Equal(f.h.calls, []string{"workspace_get", "workspace_close", "tab_create"}) || !slices.Equal(f.h.workspaceCloseCalls, []string{"ws-7"}) {
				t.Fatalf("herdr calls = %v (workspace closes %v), want the check and the close of ws-7, then start's tab", f.h.calls, f.h.workspaceCloseCalls)
			}
			wantTrail(t, f.store, "attached d1", "workspace_created d1", "pane_created d1", "launched d1")
			if got := mustReadDirector(t, f.store); currentDirectorID(got) != "d1" ||
				!slices.Equal(openHeadlessSummary(got), []string{"d1 running ws=ws-7 pane=" + paneID}) {
				t.Fatalf("Read = current %+v, open %q; want d1 running and current with ws-7 and %s", got.Current, openHeadlessSummary(got), paneID)
			}

			// herdr closed the new tab with its workspace; the stop run again
			// finds both gone and records the stop.
			delete(f.h.workspaceLabels, "ws-7")
			delete(f.h.paneTabs, paneID)
			again, err := f.stop("d1")
			if err != nil || !again.WasCurrent || again.PaneNote != "pane=already closed" || again.WorkspaceNote != "workspace=already closed" {
				t.Fatalf("stop again = %+v, %v; want the stop of d1 recorded", again, err)
			}
			wantTrail(t, f.store, "attached d1", "workspace_created d1", "pane_created d1", "launched d1", "detached d1", "stopped d1")
		})
	}
	t.Run("everything, before stop asks herdr anything", func(t *testing.T) {
		// With nothing recorded at the read, stop calls no herdr, so the
		// start lands in its only other look outside the store: its read of
		// HERDR_PANE_ID, right after the registration's.
		f := newDirStartFixture(t, directorTestToken)
		if _, err := f.store.AttachHeadless(testHeadlessAttach()); err != nil {
			t.Fatalf("AttachHeadless: %v", err)
		}
		landed := false
		f.o.Getenv = func(key string) string {
			if key == herdrPaneIDEnv && !landed {
				landed = true
				// Start's steps 2, 3, and 6, its agent start left out.
				ctx := context.Background()
				ws, err := f.h.WorkspaceCreate(ctx, f.cwd, DirectorWorkspaceLabel(directorTestToken))
				if err == nil {
					err = f.store.RecordWorkspace("d1", ws, testDirectorWho)
				}
				var pane string
				if err == nil {
					pane, err = f.h.TabCreate(ctx, ws, f.cwd, DirectorTabLabel(directorTestToken))
				}
				if err != nil {
					t.Errorf("start's steps: %v", err)
					return ""
				}
				recordPaneAndLaunched(t, f.store, pane)
			}
			return f.env[key]
		}
		res, err := f.stop("d1")

		wantChangedDuringStop(t, res, err, changedDuringStopText("no workspace and no pane", `workspace "ws-1" and pane "pane-1"`),
			"ctrl_c=skipped, pane=none recorded, workspace=none recorded")
		if res.UnrecordedWorkspaceLabel != DirectorWorkspaceLabel(directorTestToken) || res.WasCurrent {
			t.Fatalf("stop result = %+v, want nothing recorded at the read", res)
		}
		if !slices.Equal(f.h.calls, []string{"workspace_create", "tab_create"}) {
			t.Fatalf("herdr calls = %v, want start's two creates only", f.h.calls)
		}
		wantTrail(t, f.store, "attached d1", "workspace_created d1", "pane_created d1", "launched d1")

		// The stop run again closes what start recorded.
		n := len(f.h.calls)
		again, err := f.stop("d1")
		if err != nil || !again.WasCurrent || again.PaneNote != "pane=closed" || again.WorkspaceNote != "workspace=closed" {
			t.Fatalf("stop again = %+v, %v; want start's pane and workspace closed", again, err)
		}
		if got := herdrCallsSince(f.h, n); !slices.Equal(got, directorCloseCalls) {
			t.Fatalf("herdr calls = %v, want %v", got, directorCloseCalls)
		}
		wantTrail(t, f.store, "attached d1", "workspace_created d1", "pane_created d1", "launched d1", "detached d1", "stopped d1")
		wantNoDirectorOpen(t, f.store)
	})
}

// recordPaneAndLaunched records paneID and launched for d1, as start does
// once herdr made its tab and its agent started.
func recordPaneAndLaunched(t *testing.T, s *DirectorStore, paneID string) {
	t.Helper()
	if err := s.RecordPane("d1", paneID, testDirectorWho); err != nil {
		t.Errorf("RecordPane: %v", err)
		return
	}
	if still, err := s.MarkLaunched("d1", testDirectorWho); err != nil || !still {
		t.Errorf("MarkLaunched = %v, %v; want true, nil", still, err)
	}
}

// TestStopDirector_AForcedStopRecordsWhatStartRecordedMeanwhile pins that
// D14's check leaves the forced stop alone: its stopped says nothing more was
// closed, which stays true when a start records a pane after stop's read, so
// a forced stop over a workspace whose label is not this launch's still
// records stopped with force (and detached), sending nothing.
func TestStopDirector_AForcedStopRecordsWhatStartRecordedMeanwhile(t *testing.T) {
	f := newDirStartFixture(t, directorTestToken)
	if _, err := f.store.AttachHeadless(testHeadlessAttach()); err != nil {
		t.Fatalf("AttachHeadless: %v", err)
	}
	if err := f.store.RecordWorkspace("d1", "ws-7", testDirectorWho); err != nil {
		t.Fatalf("RecordWorkspace: %v", err)
	}
	setLabel(&f.h.workspaceLabels, "ws-7", "org-a")
	landed := false
	f.o.Getenv = func(key string) string {
		if key == herdrPaneIDEnv && !landed {
			landed = true
			recordPaneAndLaunched(t, f.store, "pane-9")
		}
		return f.env[key]
	}
	res, err := stopWithForce(f, "d1")
	if err != nil || !res.Forced || !res.WasCurrent || res.Registration.PaneID != "" {
		t.Fatalf("stop --force = %+v, %v; want a forced stop of the registration as read", res, err)
	}
	if !slices.Equal(f.h.calls, []string{"workspace_get"}) {
		t.Fatalf("herdr calls = %v, want the workspace check only", f.h.calls)
	}
	wantNothingSent(t, f.h)
	wantTrail(t, f.store, "attached d1", "workspace_created d1", "pane_created d1", "launched d1", "detached d1", "stopped d1+force")
	wantNoDirectorOpen(t, f.store)
}
