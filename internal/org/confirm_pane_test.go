package org

import (
	"cmp"
	"errors"
	"slices"
	"strings"
	"testing"
)

// The check before a C-c or a close of a recorded pane, by the pane's own
// label (plan 2026-11-06-org-herdr-single-tab, D4 and D5; AC5, AC6, AC7):
// confirmSeatPane for Stop, Disband, CloseDeferredSelfPane, and the
// director, and confirmStaleSeatPane, which accepts one pane more, for
// compensateStale.

// confirmPaneHerdr is a fakeHerdr holding pane p1 in tab t1 of workspace
// ws-1, labelled as given ("" for no label).
func confirmPaneHerdr(paneLabel, tabLabel, workspaceLabel string) *fakeHerdr {
	h := &fakeHerdr{}
	setLabel(&h.paneTabs, "p1", "t1")
	setLabel(&h.paneWorkspaces, "p1", "ws-1")
	if paneLabel != "" {
		setLabel(&h.paneLabels, "p1", paneLabel)
	}
	setLabel(&h.tabLabels, "t1", tabLabel)
	setLabel(&h.workspaceLabels, "ws-1", workspaceLabel)
	return h
}

// TestConfirmSeatPane_TheRule pins D4 on the check itself, for org-a's
// seat-1: a pane labelled seat-1 is confirmed with no tab get; a pane with
// no label is confirmed only when its tab is labelled seat-1 (a seat an
// earlier version made in a tab of its own), so not in the org's tab; a pane
// labelled anything else is not, whatever its tab; and the workspace must be
// org-a's every time. With the tab a stale spawn recorded (confirmStaleSeatPane),
// a pane with no label in that tab is confirmed too, with no tab get, and one
// in another tab is not unless that tab is labelled seat-1. For a seat whose
// id is the org_id (org-a), a tab labelled org-a is the org's tab, so a pane
// with no label in it is not confirmed (a pane labelled org-a still is, and
// so is a stale spawn's pane in the tab it recorded). A mismatch names the
// label it found.
func TestConfirmSeatPane_TheRule(t *testing.T) {
	paneWS := []string{"pane_get", "workspace_get"}
	paneTabWS := []string{"pane_get", "tab_get", "workspace_get"}
	for _, tc := range []struct {
		name               string
		seat               string // seat-1 when empty
		paneLabel, tab, ws string
		createdTab         string // confirmStaleSeatPane's when set
		wantErr            bool   // an errLabelMismatch
		wantText           string
		wantCalls          []string
	}{
		{name: "pane labelled with the seat id", paneLabel: "seat-1", tab: "org-a", ws: "org-a", wantCalls: paneWS},
		{name: "pane labelled with the seat id, another org's workspace", paneLabel: "seat-1", tab: "org-a", ws: "org-z", wantErr: true,
			wantText: `workspace "ws-1" is not org_id "org-a"'s: herdr labels it "org-z"`, wantCalls: paneWS},
		{name: "pane labelled another seat in a tab labelled seat-1", paneLabel: "seat-9", tab: "seat-1", ws: "org-a", wantErr: true,
			wantText: `pane "p1" is not seat "seat-1"'s: herdr labels it "seat-9"`, wantCalls: []string{"pane_get"}},
		{name: "no label, tab labelled with the seat id (an earlier version)", tab: "seat-1", ws: "org-a", wantCalls: paneTabWS},
		{name: "no label, tab labelled with the seat id, another org's workspace", tab: "seat-1", ws: "org-z", wantErr: true,
			wantText: `herdr labels it "org-z"`, wantCalls: paneTabWS},
		{name: "no label in the org's tab", tab: "org-a", ws: "org-a", wantErr: true,
			wantText: `pane "p1" is not seat "seat-1"'s: it has no label, and herdr has it in tab "t1" labelled "org-a"`, wantCalls: []string{"pane_get", "tab_get"}},
		{name: "no label in a new workspace's tab", tab: "1", ws: "org-a", wantErr: true,
			wantText: `it has no label, and herdr has it in tab "t1" labelled "1"`, wantCalls: []string{"pane_get", "tab_get"}},

		{name: "stale: no label in the tab its spawn made it in", tab: "1", ws: "org-a", createdTab: "t1", wantCalls: paneWS},
		{name: "stale: no label in that tab, another org's workspace", tab: "org-z", ws: "org-z", createdTab: "t1", wantErr: true,
			wantText: `herdr labels it "org-z"`, wantCalls: paneWS},
		{name: "stale: labelled another seat in that tab", paneLabel: "seat-9", tab: "org-a", ws: "org-a", createdTab: "t1", wantErr: true,
			wantText: `herdr labels it "seat-9"`, wantCalls: []string{"pane_get"}},
		{name: "stale: no label in another tab", tab: "org-a", ws: "org-a", createdTab: "t0", wantErr: true,
			wantText:  `it has no label, and herdr has it in tab "t1" labelled "org-a", not in tab "t0", where its spawn made it`,
			wantCalls: []string{"pane_get", "tab_get"}},
		{name: "stale: no label in another tab labelled with the seat id", tab: "seat-1", ws: "org-a", createdTab: "t0", wantCalls: paneTabWS},

		{name: "seat id is the org_id: pane labelled with it", seat: "org-a", paneLabel: "org-a", tab: "org-a", ws: "org-a", wantCalls: paneWS},
		{name: "seat id is the org_id: no label in the org's tab", seat: "org-a", tab: "org-a", ws: "org-a", wantErr: true,
			wantText:  `pane "p1" is not seat "org-a"'s: it has no label, and herdr has it in tab "t1" labelled "org-a", which is also the workspace's label`,
			wantCalls: []string{"pane_get", "tab_get"}},
		{name: "seat id is the org_id, stale: no label in the tab its spawn made it in", seat: "org-a", tab: "org-a", ws: "org-a", createdTab: "t1",
			wantCalls: paneWS},
		{name: "seat id is the org_id, stale: no label in another tab labelled with it", seat: "org-a", tab: "org-a", ws: "org-a", createdTab: "t0",
			wantErr:   true,
			wantText:  `which is also the workspace's label: the label of a tab its panes share, not of a tab of the pane's own, not in tab "t0", where its spawn made it`,
			wantCalls: []string{"pane_get", "tab_get"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			o, _, _ := testOrg(t)
			h := confirmPaneHerdr(tc.paneLabel, tc.tab, tc.ws)
			o.Herdr = h
			seat := cmp.Or(tc.seat, "seat-1")
			var gone bool
			var err error
			if tc.createdTab == "" {
				gone, err = o.confirmSeatPane("org-a", seat, "p1", seatLabelWords)
			} else {
				gone, err = o.confirmStaleSeatPane("org-a", seat, "p1", tc.createdTab)
			}
			if gone {
				t.Fatalf("gone = true for a pane herdr has")
			}
			switch {
			case !tc.wantErr && err != nil:
				t.Fatalf("err = %v, want the pane confirmed", err)
			case tc.wantErr && (!errors.Is(err, errLabelMismatch) || !strings.Contains(err.Error(), tc.wantText)):
				t.Fatalf("err = %v, want a label mismatch saying %q", err, tc.wantText)
			}
			if !slices.Equal(h.calls, tc.wantCalls) {
				t.Fatalf("herdr calls = %v, want %v", h.calls, tc.wantCalls)
			}
		})
	}
	t.Run("a pane herdr does not know is gone", func(t *testing.T) {
		o, h, _ := testOrg(t)
		for _, createdTab := range []string{"", "t1"} {
			gone, err := o.confirmStaleSeatPane("org-a", "seat-1", "p1", createdTab)
			if !gone || err != nil {
				t.Fatalf("createdTab %q: gone, err = %v, %v; want gone", createdTab, gone, err)
			}
		}
		if !slices.Equal(h.calls, []string{"pane_get", "pane_get"}) {
			t.Fatalf("herdr calls = %v, want the two pane gets only", h.calls)
		}
	})
}

// TestConfirmSeatPane_StopDisbandAndOwnPaneClose pins D4 through its seat
// callers (AC5, AC6), for seat-1 spawned on pane-1 in tab-pane-1 of ws-1
// (org-a), herdr's view then changed per row: Stop, Disband, and the close
// of the caller's own pane (CloseDeferredSelfPane) send C-c to and close a
// pane labelled seat-1 (this version's seat, its tab labelled org-a) and an
// unlabelled pane in a tab labelled seat-1 (an earlier version's seat), and
// leave alone, with nothing sent, an unlabelled pane in the org's tab, a
// pane labelled another seat, and a pane labelled seat-1 in another org's
// workspace. Disband then leaves the org's workspace open too.
func TestConfirmSeatPane_StopDisbandAndOwnPaneClose(t *testing.T) {
	runSeatPaneCloses(t, "seat-1", []seatPaneCloseCase{
		{name: "this version's seat", setup: func(h *fakeHerdr) {
			setLabel(&h.paneLabels, "pane-1", "seat-1")
			setLabel(&h.tabLabels, fakeTabID("pane-1"), "org-a")
		}, wantClosed: true},
		{name: "an earlier version's seat", setup: func(h *fakeHerdr) {
			delete(h.paneLabels, "pane-1")
			setLabel(&h.tabLabels, fakeTabID("pane-1"), "seat-1")
		}, wantClosed: true},
		{name: "an unlabelled pane in the org's tab", setup: func(h *fakeHerdr) {
			delete(h.paneLabels, "pane-1")
			setLabel(&h.tabLabels, fakeTabID("pane-1"), "org-a")
		}, wantText: `pane "pane-1" is not seat "seat-1"'s: it has no label, and herdr has it in tab "tab-pane-1" labelled "org-a"`},
		{name: "a pane labelled another seat", setup: func(h *fakeHerdr) {
			setLabel(&h.paneLabels, "pane-1", "seat-9")
		}, wantText: `pane "pane-1" is not seat "seat-1"'s: herdr labels it "seat-9"`},
		{name: "a pane of another org's workspace", setup: func(h *fakeHerdr) {
			setLabel(&h.paneLabels, "pane-1", "seat-1")
			setLabel(&h.paneWorkspaces, "pane-1", "ws-other")
			setLabel(&h.workspaceLabels, "ws-other", "org-b")
		}, wantText: `workspace "ws-other" is not org_id "org-a"'s: herdr labels it "org-b"`},
	})
}

// TestConfirmSeatPane_SeatIDIsTheOrgID_StopDisbandAndOwnPaneClose pins the
// one case where the tab label rule of D4 would meet this version's org tab
// (cross-review of plan 2026-11-06-org-herdr-single-tab, ACTION_REQUIRED 1):
// seat org-a of org org-a, spawned on pane-1 in tab-pane-1, which spawn
// labels org-a as it labels every org's tab. Stop, Disband, and the close of
// the caller's own pane still send C-c to and close the pane labelled org-a
// (this version's seat), and leave alone, with nothing sent, an unlabelled
// pane in that tab (a shell a person opened there, or another pane herdr
// gave the recorded id): its tab's label is the seat id, but it is the
// workspace's label too.
func TestConfirmSeatPane_SeatIDIsTheOrgID_StopDisbandAndOwnPaneClose(t *testing.T) {
	runSeatPaneCloses(t, "org-a", []seatPaneCloseCase{
		{name: "this version's seat", setup: func(h *fakeHerdr) {
			setLabel(&h.paneLabels, "pane-1", "org-a")
			setLabel(&h.tabLabels, fakeTabID("pane-1"), "org-a")
		}, wantClosed: true},
		{name: "an unlabelled pane in the org's tab", setup: func(h *fakeHerdr) {
			delete(h.paneLabels, "pane-1")
			setLabel(&h.tabLabels, fakeTabID("pane-1"), "org-a")
		}, wantText: `pane "pane-1" is not seat "org-a"'s: it has no label, and herdr has it in tab "tab-pane-1" labelled "org-a", which is also the workspace's label`},
	})
}

// seatPaneCloseCase is one row of runSeatPaneCloses: setup changes herdr's
// view after the spawn; wantText is in the error when the pane is not
// closed.
type seatPaneCloseCase struct {
	name       string
	setup      func(h *fakeHerdr)
	wantClosed bool
	wantText   string
}

// runSeatPaneCloses spawns seat of org-a on pane-1 (the root pane of ws-1,
// in tab-pane-1) for each row and each of Stop, Disband, and
// CloseDeferredSelfPane, applies the row's setup, and checks that the verb
// sends C-c to and closes pane-1 when wantClosed (Disband then closes ws-1
// and reports the org disbanded), and otherwise fails with a label mismatch
// saying wantText, sends nothing, and leaves the workspace open.
func runSeatPaneCloses(t *testing.T, seat string, cases []seatPaneCloseCase) {
	t.Helper()
	for _, tc := range cases {
		for _, verb := range []string{"stop", "disband", "own pane close"} {
			t.Run(tc.name+"/"+verb, func(t *testing.T) {
				o, h, _ := testOrg(t)
				if r := o.Spawn(mustSpawnParams("org-a", seat)); r.Outcome != SpawnOutcomeSpawned || r.Seat.PaneID != "pane-1" {
					t.Fatalf("spawn: %+v", r)
				}
				tc.setup(h)
				keysBefore := len(h.sendKeysCalls)

				var err error
				switch verb {
				case "stop":
					res := o.Stop(StopParams{OrgID: "org-a", Seat: seat})
					err = res.Err
					wantLastEvent(t, o, seat, tc.wantClosed, EventStopped, EventStopFailed)
				case "disband":
					res := o.Disband(DisbandParams{OrgID: "org-a"})
					if len(res.Errs) > 0 {
						err = res.Errs[0]
					}
					wantWorkspaces := []string(nil)
					if tc.wantClosed {
						wantWorkspaces = []string{"ws-1"}
					}
					if res.Disbanded != tc.wantClosed || !slices.Equal(h.workspaceCloseCalls, wantWorkspaces) {
						t.Fatalf("disbanded %v, workspace closes %v; want %v, %v", res.Disbanded, h.workspaceCloseCalls, tc.wantClosed, wantWorkspaces)
					}
				case "own pane close":
					err = o.CloseDeferredSelfPane("pane-1", false)
				}

				if tc.wantClosed {
					if err != nil {
						t.Fatalf("err = %v, want the pane closed", err)
					}
					if !slices.Equal(h.paneCloseCalls, []string{"pane-1"}) {
						t.Fatalf("pane closes = %v, want pane-1", h.paneCloseCalls)
					}
					if verb != "own pane close" && !slices.Equal(h.sendKeysCalls[keysBefore:], []string{"pane-1"}) {
						t.Fatalf("C-c sent to %v, want pane-1", h.sendKeysCalls[keysBefore:])
					}
					return
				}
				if !errors.Is(err, errLabelMismatch) || !strings.Contains(err.Error(), tc.wantText) {
					t.Fatalf("err = %v, want a label mismatch saying %q", err, tc.wantText)
				}
				if len(h.paneCloseCalls) != 0 || len(h.sendKeysCalls) != keysBefore {
					t.Fatalf("pane closes %v, C-c %v; want nothing sent", h.paneCloseCalls, h.sendKeysCalls[keysBefore:])
				}
			})
		}
	}
}

// wantLastEvent fails the test unless the last manifest event is seat's of
// org-a and is closedEvent when closed and notClosedEvent otherwise.
func wantLastEvent(t *testing.T, o *Org, seat string, closed bool, closedEvent, notClosedEvent string) {
	t.Helper()
	events := mustReadEvents(t, o)
	want := notClosedEvent
	if closed {
		want = closedEvent
	}
	if last := events[len(events)-1]; last.OrgID != "org-a" || last.SeatID != seat || last.Event != want {
		t.Fatalf("last event = %+v, want %s's %s", last, seat, want)
	}
}

// TestCompensateStale_PaneCreatedDetails pins D5's Details: what
// paneCreatedDetails writes for each way of making a pane, paneCreatedTab
// reads the tab back from, with a note after the three words too; any other
// Details reads as no tab.
func TestCompensateStale_PaneCreatedDetails(t *testing.T) {
	for _, how := range []string{paneHowRoot, paneHowSplitRight, paneHowSplitDown, paneHowTab} {
		details := paneCreatedDetails(how, "w1:t2")
		if want := "pane_created how=" + how + " tab=w1:t2"; details != want {
			t.Fatalf("paneCreatedDetails(%q) = %q, want %q", how, details, want)
		}
		for _, d := range []string{details, details + " tab_rename=failed: herdr: tab_not_found"} {
			if tab, ok := paneCreatedTab(d); !ok || tab != "w1:t2" {
				t.Fatalf("paneCreatedTab(%q) = %q, %v; want w1:t2", d, tab, ok)
			}
		}
	}
	for _, d := range []string{
		"", "tab_created", "agent_started prompt_file=/tmp/p.md", "pane_created",
		"pane_created how=root", "pane_created how=root tab=", "pane_created how=sideways tab=t1",
		"pane_created tab=t1 how=root", "pane_created how= tab=t1", "pane_createdx how=root tab=t1",
	} {
		if tab, ok := paneCreatedTab(d); ok || tab != "" {
			t.Fatalf("paneCreatedTab(%q) = %q, %v; want no tab", d, tab, ok)
		}
	}
}

// staleCreatedPane is a stale spawn of org-a/seat-1 for
// TestCompensateStale_ClosesOnlyThePaneItsSpawnMade: its manifest records
// (spawn_started, then each spawn_step's Details, every one on pane) and
// herdr's view of ws-stale (labelled wsLabel), which holds pane in tab with
// paneLabel ("" for none), and the panes of other.
type staleCreatedPane struct {
	steps              []string
	pane, tab          string
	paneLabel, wsLabel string
	tabLabels          map[string]string // tab id -> label
	other              map[string]string // other pane id -> its tab id, no label
}

func (s staleCreatedPane) seed(t *testing.T, o *Org, h *fakeHerdr) {
	t.Helper()
	events := []ManifestEvent{{TS: "2026-11-06T00:00:00Z", OrgID: "org-a", SeatID: "seat-1", Event: EventSpawnStarted,
		Role: "worker", Driver: "claude", Model: "sonnet", Worktree: "/tmp/seat"}}
	for _, d := range s.steps {
		events = append(events, ManifestEvent{TS: "2026-11-06T00:00:01Z", OrgID: "org-a", SeatID: "seat-1", Event: EventSpawnStep, PaneID: s.pane, Details: d})
	}
	for _, ev := range events {
		if err := o.Manifest.Append(ev); err != nil {
			t.Fatalf("seed %s: %v", ev.Event, err)
		}
	}
	panes := map[string]string{s.pane: s.tab}
	for id, tab := range s.other {
		panes[id] = tab
	}
	for id, tab := range panes {
		setLabel(&h.paneTabs, id, tab)
		setLabel(&h.paneWorkspaces, id, "ws-stale")
	}
	if s.paneLabel != "" {
		setLabel(&h.paneLabels, s.pane, s.paneLabel)
	}
	for tab, label := range s.tabLabels {
		setLabel(&h.tabLabels, tab, label)
	}
	setLabel(&h.workspaceLabels, "ws-stale", s.wsLabel)
}

// TestCompensateStale_ClosesOnlyThePaneItsSpawnMade pins D4's one acceptance
// for compensateStale (AC7, AC5), from the state a spawn leaves when it
// stops right after its pane_created spawn_step (D5): its pane, with no
// label, in the tab that record names, in org-a's workspace, gets C-c and is
// closed, with no tab get, whatever the tab's label: the root pane of a new
// workspace (tab "1"), a split in this version's tab (org-a), a split in an
// earlier version's tab (leader), and a new tab's pane. Only that pane: the
// other panes of the tab are not touched. An unlabelled pane in another tab,
// one labelled another seat, and one in another org's workspace are left
// untouched, and so is an unlabelled pane in its tab when the last record is
// a later spawn_step (by then a spawn has labelled its pane). Today's
// tab_created step keeps the rule of Stop. Each spawn_failed says what was
// done, and the new spawn goes on.
func TestCompensateStale_ClosesOnlyThePaneItsSpawnMade(t *testing.T) {
	closed := "compensation=ctrl_c=ok pane=closed)"
	for _, tc := range []struct {
		name        string
		stale       staleCreatedPane
		wantClosed  bool
		wantDetails string
		wantChecks  []string // herdr calls before the C-c, or before the new spawn
	}{
		{name: "the root pane of a new workspace", wantClosed: true, wantDetails: closed, stale: staleCreatedPane{
			steps: []string{paneCreatedDetails(paneHowRoot, "tab-root")}, pane: "root-pane", tab: "tab-root", wsLabel: "org-a",
			tabLabels: map[string]string{"tab-root": "1"}}},
		{name: "a split in this version's tab", wantClosed: true, wantDetails: closed, stale: staleCreatedPane{
			steps: []string{paneCreatedDetails(paneHowSplitRight, "tab-org")}, pane: "split-pane", tab: "tab-org", wsLabel: "org-a",
			tabLabels: map[string]string{"tab-org": "org-a"}, other: map[string]string{"leader-pane": "tab-org"}}},
		{name: "a split in an earlier version's tab", wantClosed: true, wantDetails: closed, stale: staleCreatedPane{
			steps: []string{paneCreatedDetails(paneHowSplitDown, "tab-leader")}, pane: "split-pane", tab: "tab-leader", wsLabel: "org-a",
			tabLabels: map[string]string{"tab-leader": "leader"}, other: map[string]string{"leader-pane": "tab-leader"}}},
		{name: "a new tab's pane", wantClosed: true, wantDetails: closed, stale: staleCreatedPane{
			steps: []string{paneCreatedDetails(paneHowTab, "tab-new")}, pane: "tab-pane", tab: "tab-new", wsLabel: "org-a",
			tabLabels: map[string]string{"tab-new": "seat-1"}}},

		{name: "an unlabelled pane in another tab", stale: staleCreatedPane{
			steps: []string{paneCreatedDetails(paneHowSplitRight, "tab-made")}, pane: "split-pane", tab: "tab-org", wsLabel: "org-a",
			tabLabels: map[string]string{"tab-made": "org-a", "tab-org": "org-a"}},
			wantDetails: `left untouched, labelled differently: pane "split-pane" is not seat "seat-1"'s: it has no label, and herdr has it in tab "tab-org" labelled "org-a", not in tab "tab-made", where its spawn made it)`,
			wantChecks:  []string{"pane_get", "tab_get"}},
		{name: "a pane labelled another seat in its tab", stale: staleCreatedPane{
			steps: []string{paneCreatedDetails(paneHowSplitRight, "tab-org")}, pane: "split-pane", tab: "tab-org", paneLabel: "seat-9", wsLabel: "org-a",
			tabLabels: map[string]string{"tab-org": "org-a"}},
			wantDetails: `left untouched, labelled differently: pane "split-pane" is not seat "seat-1"'s: herdr labels it "seat-9")`,
			wantChecks:  []string{"pane_get"}},
		{name: "an unlabelled pane in its tab, another org's workspace", stale: staleCreatedPane{
			steps: []string{paneCreatedDetails(paneHowRoot, "tab-root")}, pane: "root-pane", tab: "tab-root", wsLabel: "org-z",
			tabLabels: map[string]string{"tab-root": "1"}},
			wantDetails: `left untouched, labelled differently: pane "root-pane" is not confirmed in org_id "org-a"'s workspace: workspace "ws-stale" is not org_id "org-a"'s: herdr labels it "org-z")`,
			wantChecks:  []string{"pane_get", "workspace_get"}},
		{name: "an unlabelled pane in its tab, a later step last", stale: staleCreatedPane{
			steps: []string{paneCreatedDetails(paneHowRoot, "tab-root"), "agent_started prompt_file=/tmp/p.md"}, pane: "root-pane", tab: "tab-root", wsLabel: "org-a",
			tabLabels: map[string]string{"tab-root": "1"}},
			wantDetails: `previous_event=spawn_step compensation=ctrl_c=skipped pane=left untouched, labelled differently: pane "root-pane" is not seat "seat-1"'s: it has no label, and herdr has it in tab "tab-root" labelled "1")`,
			wantChecks:  []string{"pane_get", "tab_get"}},

		{name: "tab_created, its tab labelled with the seat id", wantClosed: true, wantDetails: closed, stale: staleCreatedPane{
			steps: []string{"tab_created"}, pane: "stale-pane", tab: "tab-stale", wsLabel: "org-a",
			tabLabels: map[string]string{"tab-stale": "seat-1"}},
			wantChecks: []string{"pane_get", "tab_get", "workspace_get"}},
		{name: "tab_created, an unlabelled pane in the org's tab", stale: staleCreatedPane{
			steps: []string{"tab_created"}, pane: "stale-pane", tab: "tab-org", wsLabel: "org-a",
			tabLabels: map[string]string{"tab-org": "org-a"}},
			wantDetails: `left untouched, labelled differently: pane "stale-pane" is not seat "seat-1"'s: it has no label, and herdr has it in tab "tab-org" labelled "org-a")`,
			wantChecks:  []string{"pane_get", "tab_get"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			o, h, _ := testOrg(t)
			tc.stale.seed(t, o, h)

			r := o.Spawn(mustSpawnParams("org-a", "seat-1"))
			if r.Outcome != SpawnOutcomeSpawned || r.Seat.PaneID != "pane-1" {
				t.Fatalf("expected the new spawn to reach spawned on pane-1, got %+v", r)
			}
			checks := tc.wantChecks
			if tc.wantClosed && checks == nil {
				checks = []string{"pane_get", "workspace_get"}
			}
			want := slices.Clone(checks)
			if tc.wantClosed {
				want = append(want, "pane_send_keys", "pane_close")
			}
			if len(h.calls) < len(want) || !slices.Equal(h.calls[:len(want)], want) {
				t.Fatalf("herdr calls = %v, want them to start %v", h.calls, want)
			}
			wantClosed := []string(nil)
			if tc.wantClosed {
				wantClosed = []string{tc.stale.pane}
			}
			if !slices.Equal(h.paneCloseCalls, wantClosed) || slices.Contains(h.sendKeysCalls, tc.stale.pane) != tc.wantClosed {
				t.Fatalf("pane closes %v, C-c %v; want only %v closed", h.paneCloseCalls, h.sendKeysCalls, wantClosed)
			}
			for other := range tc.stale.other {
				if slices.Contains(h.sendKeysCalls, other) {
					t.Fatalf("C-c sent to %s, another pane of the tab: %v", other, h.sendKeysCalls)
				}
			}
			failed := eventsNamed(mustReadEvents(t, o), EventSpawnFailed)
			if len(failed) != 1 || failed[0].PaneID != tc.stale.pane {
				t.Fatalf("expected one spawn_failed recording %s, got %+v", tc.stale.pane, failed)
			}
			assertDetailsContains(t, failed[0].Details, "superseded by respawn (previous_event=spawn_step ", tc.wantDetails)
		})
	}
}

// TestStopDirector_ConfirmsThePaneByItsLabel pins D4 for the headless
// director (AC8's stop side): director stop closes a pane labelled
// DirectorTabLabel (this version's director, its tab labelled
// DirectorWorkspaceLabel) with no tab get, and an unlabelled pane in a tab
// labelled DirectorTabLabel (an earlier version's); it sends nothing to an
// unlabelled pane in the director's tab or to a pane labelled for another
// director, and names the label it found, with no word of a seat or an org.
func TestStopDirector_ConfirmsThePaneByItsLabel(t *testing.T) {
	label, wsLabel := DirectorTabLabel(directorTestToken), DirectorWorkspaceLabel(directorTestToken)
	for _, tc := range []struct {
		name      string
		setup     func(h *directorHerdr)
		wantCalls []string // when closed
		wantText  string   // when not closed
	}{
		{name: "this version's director", setup: func(h *directorHerdr) {
			setLabel(&h.paneLabels, "pane-1", label)
			setLabel(&h.tabLabels, fakeTabID("pane-1"), wsLabel)
		}, wantCalls: []string{"pane_get", "workspace_get", "workspace_get", "pane_send_keys", "pane_close", "workspace_close"}},
		{name: "an earlier version's director", setup: func(h *directorHerdr) {
			delete(h.paneLabels, "pane-1")
			setLabel(&h.tabLabels, fakeTabID("pane-1"), label)
		}, wantCalls: legacyDirectorCloseCalls},
		{name: "an unlabelled pane in the director's tab", setup: func(h *directorHerdr) {
			delete(h.paneLabels, "pane-1")
			setLabel(&h.tabLabels, fakeTabID("pane-1"), wsLabel)
		}, wantText: `pane "pane-1" is not labelled "` + label + `": it has no label, and herdr has it in tab "tab-pane-1" labelled "` + wsLabel + `"`},
		{name: "a pane labelled for another director", setup: func(h *directorHerdr) {
			setLabel(&h.paneLabels, "pane-1", DirectorTabLabel("ffffffffffff"))
		}, wantText: `pane "pane-1" is not labelled "` + label + `": herdr labels it "director_ffffffffffff"`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := newDirStartFixture(t, directorTestToken)
			f.mustStart(t, DirectorDriverClaude, "opus")
			tc.setup(f.h)
			n := len(f.h.calls)
			res, err := f.stop("d1")
			if tc.wantText == "" {
				if err != nil || res.PaneNote != "pane=closed" || res.WorkspaceNote != "workspace=closed" {
					t.Fatalf("stop = %+v, %v; want the pane and the workspace closed", res, err)
				}
				if got := herdrCallsSince(f.h, n); !slices.Equal(got, tc.wantCalls) {
					t.Fatalf("herdr calls = %v, want %v", got, tc.wantCalls)
				}
				wantNoDirectorOpen(t, f.store)
				return
			}
			var ce *DirectorCleanupError
			if !errors.As(err, &ce) || !ce.Unconfirmed || !strings.Contains(err.Error(), tc.wantText) {
				t.Fatalf("err = %v, want an unconfirmed stop saying %q", err, tc.wantText)
			}
			for _, word := range []string{"seat", "org_id"} {
				if strings.Contains(err.Error(), word) {
					t.Fatalf("err = %v, want it not to say %q", err, word)
				}
			}
			if len(f.h.sendKeysCalls) != 0 || len(f.h.paneCloseCalls) != 0 || len(f.h.workspaceCloseCalls) != 0 {
				t.Fatalf("C-c %v, pane closes %v, workspace closes %v, want none", f.h.sendKeysCalls, f.h.paneCloseCalls, f.h.workspaceCloseCalls)
			}
		})
	}
}
