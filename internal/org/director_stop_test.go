package org

import (
	"errors"
	"slices"
	"strings"
	"testing"

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
			for _, want := range append(tc.wantText, "nothing was recorded", "herdr workspace list") {
				if !strings.Contains(err.Error(), want) {
					t.Fatalf("err = %v, want it to say %q", err, want)
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
			if !errors.As(err, &ce) || ce.Unconfirmed || !errors.Is(err, closeErr) || !slices.Equal(ce.HerdrCommands, tc.wantCommands) {
				t.Fatalf("err = %#v, want a close failure with commands %q", err, tc.wantCommands)
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
