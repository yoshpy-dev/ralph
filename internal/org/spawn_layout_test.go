package org

import (
	"context"
	"errors"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/yoshpy-dev/ralph/internal/org/driver"
)

// Tests of where Spawn puts a seat's pane (placeSeatPane; plan
// 2026-11-06-org-herdr-single-tab, D1 to D3, D5 and D6, AC1 to AC4, AC7 and
// AC9). The fake's root pane ids are fakeRootPaneID's (each test clears
// testOrg's nextPaneID) and its split ids are named per test.

// layoutOrg is testOrg with nextPaneID cleared, so a new workspace's root
// pane is fakeRootPaneID, and the given ids for the splits, in order.
func layoutOrg(t *testing.T, splits ...string) (*Org, *fakeHerdr, *fakeAgmsg) {
	t.Helper()
	o, h, a := testOrg(t)
	h.nextPaneID = ""
	h.splitPaneIDs = splits
	return o, h, a
}

// mustSpawnLayout spawns p and fails the test unless it reached spawned on
// wantPane.
func mustSpawnLayout(t *testing.T, o *Org, p SpawnParams, wantPane string) {
	t.Helper()
	if r := o.Spawn(p); r.Outcome != SpawnOutcomeSpawned || r.Seat.PaneID != wantPane {
		t.Fatalf("spawn %s/%s = %+v; want spawned on %s", p.OrgID, p.SeatID, r, wantPane)
	}
}

// paneCreatedStep returns the Details of the pane_created spawn_step of
// orgID/seatID's latest spawn in o's manifest ("" for none).
func paneCreatedStep(t *testing.T, o *Org, orgID, seatID string) string {
	t.Helper()
	details := ""
	for _, ev := range mustReadEvents(t, o) {
		if ev.OrgID != orgID || ev.SeatID != seatID {
			continue
		}
		switch {
		case ev.Event == EventSpawnStarted:
			details = ""
		case ev.Event == EventSpawnStep && strings.HasPrefix(ev.Details, paneCreatedEvent+" "):
			details = ev.Details
		}
	}
	return details
}

// unlabelledPanes returns the panes herdr's view has in workspaceID with no
// label, sorted.
func unlabelledPanes(h *fakeHerdr, workspaceID string) []string {
	h.mu.Lock()
	defer h.mu.Unlock()
	var out []string
	for id, ws := range h.paneWorkspaces {
		if _, inTab := h.paneTabs[id]; ws == workspaceID && inTab && h.paneLabels[id] == "" {
			out = append(out, id)
		}
	}
	slices.Sort(out)
	return out
}

// seedOpenOrgWorkspace records workspaceID open for orgID, as an
// org_workspace_created an earlier spawn wrote, and puts it in herdr's view
// labelled orgID, with the root pane (fakeRootPaneID) an earlier version
// left in its tab "1" and no pane of a seat, as after its seats stopped.
func seedOpenOrgWorkspace(t *testing.T, o *Org, h *fakeHerdr, orgID, workspaceID string) {
	t.Helper()
	if err := o.Manifest.Append(ManifestEvent{TS: "2026-08-01T00:00:00Z", OrgID: orgID, Event: EventOrgWorkspaceCreated, PaneID: workspaceID, Worktree: "/tmp/seat"}); err != nil {
		t.Fatalf("seed org_workspace_created: %v", err)
	}
	root := fakeRootPaneID(workspaceID)
	setLabel(&h.workspaceLabels, workspaceID, orgID)
	setLabel(&h.paneTabs, root, fakeTabID(root))
	setLabel(&h.paneWorkspaces, root, workspaceID)
	setLabel(&h.tabLabels, fakeTabID(root), "1")
}

// TestOrgSpawn_Layout_LeaderImplementerReviewer pins AC1: a new org's
// leader, implementer and reviewer, spawned in that order, make exactly
// these herdr calls: the leader's workspace create, with the leader's
// RALPH_PRIMARY_CLI (driver.Herdr adds the fixed PaneEnv before it), the
// rename of its tab to the org_id and of its root pane to leader; then, for
// each next seat, the check of the workspace's label, the list of its panes,
// a split of the leader's pane to the right for the implementer and of the
// implementer's pane down for the reviewer, and the rename of the new pane.
// No tab is created, all three panes are in the root pane's tab, and each
// seat's pane_created step says how its pane was made and in which tab.
func TestOrgSpawn_Layout_LeaderImplementerReviewer(t *testing.T) {
	o, h, _ := layoutOrg(t, "p-impl", "p-rev")
	root, tab := fakeRootPaneID("ws-1"), fakeRootTabID("ws-1")
	mustSpawnLayout(t, o, leaderParams("org-a"), root)
	mustSpawnLayout(t, o, implementerParams("org-a"), "p-impl")
	mustSpawnLayout(t, o, reviewerParams("org-a"), "p-rev")

	next := []string{"workspace_get", "pane_list", "pane_split", "pane_rename", "agent_start"}
	wantCalls := append(append([]string{"workspace_create", "tab_rename", "pane_rename", "agent_start"}, next...), next...)
	if !slices.Equal(h.calls, wantCalls) {
		t.Fatalf("herdr calls = %v\nwant %v", h.calls, wantCalls)
	}
	if want := [][]string{{"RALPH_PRIMARY_CLI=claude"}}; !slices.EqualFunc(h.workspaceCreateEnvs, want, slices.Equal) {
		t.Fatalf("WorkspaceCreate extraEnv = %q, want %q", h.workspaceCreateEnvs, want)
	}
	if want := []renameCall{{tab, "org-a"}}; !slices.Equal(h.tabRenameCalls, want) {
		t.Fatalf("TabRename calls = %+v, want %+v", h.tabRenameCalls, want)
	}
	if want := []renameCall{{root, LeaderIdentity}, {"p-impl", implementerSeatID}, {"p-rev", "reviewer"}}; !slices.Equal(h.paneRenameCalls, want) {
		t.Fatalf("PaneRename calls = %+v, want %+v", h.paneRenameCalls, want)
	}
	wantSplits := []paneSplitCall{
		{paneID: root, direction: driver.HerdrSplitRight, cwd: "/tmp/seat", extraEnv: []string{"RALPH_PRIMARY_CLI=claude"}},
		{paneID: "p-impl", direction: driver.HerdrSplitDown, cwd: "/tmp/seat", extraEnv: []string{"RALPH_PRIMARY_CLI=claude"}},
	}
	if !slices.EqualFunc(h.paneSplitCalls, wantSplits, func(a, b paneSplitCall) bool {
		return a.paneID == b.paneID && a.direction == b.direction && a.cwd == b.cwd && slices.Equal(a.extraEnv, b.extraEnv)
	}) {
		t.Fatalf("PaneSplit calls = %+v, want %+v", h.paneSplitCalls, wantSplits)
	}
	if !slices.Equal(h.workspaceGetCalls, []string{"ws-1", "ws-1"}) || !slices.Equal(h.paneListCalls, []string{"ws-1", "ws-1"}) {
		t.Fatalf("workspace gets %v, pane lists %v; want ws-1 twice each", h.workspaceGetCalls, h.paneListCalls)
	}
	for _, pane := range []string{root, "p-impl", "p-rev"} {
		if h.paneTabs[pane] != tab || h.paneWorkspaces[pane] != "ws-1" {
			t.Fatalf("pane %s in tab %q of %q, want %s of ws-1", pane, h.paneTabs[pane], h.paneWorkspaces[pane], tab)
		}
	}
	if h.tabLabels[tab] != "org-a" || len(h.tabLabels) != 1 {
		t.Fatalf("tabs %v, want the one tab, labelled org-a", h.tabLabels)
	}
	for seat, want := range map[string]string{
		LeaderIdentity:    paneCreatedDetails(paneHowRoot, tab),
		implementerSeatID: paneCreatedDetails(paneHowSplitRight, tab),
		"reviewer":        paneCreatedDetails(paneHowSplitDown, tab),
	} {
		if got := paneCreatedStep(t, o, "org-a", seat); got != want {
			t.Fatalf("%s's pane_created step = %q, want %q", seat, got, want)
		}
	}
}

// TestOrgSpawn_Layout_ReviewerBeforeImplementer pins AC2: with the reviewer
// spawned before the implementer, the reviewer splits the leader's pane to
// the right and the implementer the reviewer's pane down, all in one tab
// and with no tab created. A fourth seat splits the last spawned non-leader
// seat's pane down (R6).
func TestOrgSpawn_Layout_ReviewerBeforeImplementer(t *testing.T) {
	o, h, _ := layoutOrg(t, "p-rev", "p-impl", "p-4")
	o.Config.MaxSeats = 4
	root, tab := fakeRootPaneID("ws-1"), fakeRootTabID("ws-1")
	mustSpawnLayout(t, o, leaderParams("org-a"), root)
	mustSpawnLayout(t, o, reviewerParams("org-a"), "p-rev")
	mustSpawnLayout(t, o, implementerParams("org-a"), "p-impl")
	mustSpawnLayout(t, o, mustSpawnParams("org-a", "seat-4"), "p-4")

	got := make([]string, 0, len(h.paneSplitCalls))
	for _, c := range h.paneSplitCalls {
		got = append(got, c.paneID+" "+c.direction)
	}
	if want := []string{root + " right", "p-rev down", "p-impl down"}; !slices.Equal(got, want) {
		t.Fatalf("splits = %v, want %v", got, want)
	}
	if n := countString(h.calls, "tab_create"); n != 0 {
		t.Fatalf("tab_create called %d times, want none", n)
	}
	for _, pane := range []string{root, "p-rev", "p-impl", "p-4"} {
		if h.paneTabs[pane] != tab {
			t.Fatalf("pane %s in tab %q, want %s", pane, h.paneTabs[pane], tab)
		}
	}
}

// TestOrgSpawn_Layout_StopImplementerOfThree pins AC3: in a three-seat org
// whose seats share one tab, a stop of the implementer sends C-c to and
// closes the implementer's pane alone.
func TestOrgSpawn_Layout_StopImplementerOfThree(t *testing.T) {
	o, h, _ := layoutOrg(t, "p-impl", "p-rev")
	mustSpawnLayout(t, o, leaderParams("org-a"), fakeRootPaneID("ws-1"))
	mustSpawnLayout(t, o, implementerParams("org-a"), "p-impl")
	mustSpawnLayout(t, o, reviewerParams("org-a"), "p-rev")
	keysBefore := len(h.sendKeysCalls)

	if r := o.Stop(StopParams{OrgID: "org-a", Seat: implementerSeatID}); r.Err != nil || r.CloseErr != nil {
		t.Fatalf("stop the implementer: %+v", r)
	}
	if got := h.sendKeysCalls[keysBefore:]; !slices.Equal(got, []string{"p-impl"}) {
		t.Fatalf("C-c sent to %v, want p-impl alone", got)
	}
	if !slices.Equal(h.paneCloseCalls, []string{"p-impl"}) {
		t.Fatalf("panes closed = %v, want p-impl alone", h.paneCloseCalls)
	}
	if active := activeRealSeats(t, o); !slices.Equal(active, []OrgSeat{{"org-a", LeaderIdentity}, {"org-a", "reviewer"}}) {
		t.Fatalf("active seats = %v, want the leader and the reviewer", active)
	}
}

// TestOrgSpawn_Layout_RecordedWorkspaceNotUsable pins AC4 and D2: when the
// workspace the manifest records open for the org is gone from herdr, or
// herdr labels it with something else, the next spawn records
// org_workspace_closed for it, saying why, then creates a new workspace,
// and the seat runs in its root pane. A workspace get that fails another
// way fails the spawn at workspace_get, with nothing created or recorded
// closed.
func TestOrgSpawn_Layout_RecordedWorkspaceNotUsable(t *testing.T) {
	for _, tc := range []struct {
		name        string
		change      func(h *fakeHerdr)
		wantDetails string
	}{
		{"gone", func(h *fakeHerdr) { delete(h.workspaceLabels, "ws-1") }, "not found at spawn"},
		{"relabelled", func(h *fakeHerdr) { h.workspaceLabels["ws-1"] = "org-z" }, `label "org-z" at spawn`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			o, h, _ := layoutOrg(t)
			mustSpawnLayout(t, o, leaderParams("org-a"), fakeRootPaneID("ws-1"))
			tc.change(h)
			h.workspaceID = "ws-2"
			before := len(mustReadEvents(t, o))
			mustSpawnLayout(t, o, implementerParams("org-a"), fakeRootPaneID("ws-2"))

			events := mustReadEvents(t, o)[before:]
			closed := eventsNamed(events, EventOrgWorkspaceClosed)
			created := eventsNamed(events, EventOrgWorkspaceCreated)
			if len(closed) != 1 || closed[0].PaneID != "ws-1" || closed[0].SeatID != "" || closed[0].Details != tc.wantDetails ||
				len(created) != 1 || created[0].PaneID != "ws-2" {
				t.Fatalf("workspace events = closed %+v, created %+v; want ws-1 closed (%q), then ws-2 created", closed, created, tc.wantDetails)
			}
			if i, j := slices.Index(events, closed[0]), slices.Index(events, created[0]); i > j {
				t.Fatalf("org_workspace_closed at %d after org_workspace_created at %d", i, j)
			}
			if len(h.paneListCalls) != 0 || len(h.paneSplitCalls) != 0 {
				t.Fatalf("pane lists %v, splits %+v; want none in the unusable ws-1", h.paneListCalls, h.paneSplitCalls)
			}
			if got := paneCreatedStep(t, o, "org-a", implementerSeatID); got != paneCreatedDetails(paneHowRoot, fakeRootTabID("ws-2")) {
				t.Fatalf("implementer's pane_created step = %q, want the root pane of ws-2", got)
			}
			if open := openOrgWorkspaces(mustReadEvents(t, o), "org-a"); !slices.Equal(open, []string{"ws-2"}) {
				t.Fatalf("open workspaces = %v, want ws-2 alone", open)
			}
		})
	}
	t.Run("the get fails", func(t *testing.T) {
		o, h, _ := layoutOrg(t)
		mustSpawnLayout(t, o, leaderParams("org-a"), fakeRootPaneID("ws-1"))
		refused := errors.New("herdr workspace get: connect: connection refused")
		h.getErrs = map[string]error{"ws-1": refused}
		r := o.Spawn(implementerParams("org-a"))
		if r.Outcome != SpawnOutcomeFailed || !errors.Is(r.Err, refused) || !strings.Contains(r.Err.Error(), "spawn step workspace_get failed") {
			t.Fatalf("spawn = %+v; want a workspace_get failure", r)
		}
		last := lastEvent(t, o)
		if last.Event != EventSpawnFailed || last.PaneID != "" {
			t.Fatalf("last event = %+v; want spawn_failed with no pane", last)
		}
		assertDetailsContains(t, last.Details, "step=workspace_get ", "compensation=no pane to compensate")
		events := mustReadEvents(t, o)
		if n := countString(h.calls, "workspace_create"); n != 1 || len(eventsNamed(events, EventOrgWorkspaceClosed)) != 0 {
			t.Fatalf("workspace creates %d, closed events %d; want the leader's create alone and none closed", n, len(eventsNamed(events, EventOrgWorkspaceClosed)))
		}
	})
}

// staleLayout is the state a spawn of org-a's seat leaves when its process
// dies right after the pane_created spawn_step (D5), before the pane is
// labelled: the manifest records and herdr's view, built by seed. before
// lists the unlabelled panes of ws-1 that were there before that spawn made
// its pane (an earlier version's root shell and seat panes).
type staleLayout struct {
	seat   string
	pane   string
	seed   func(t *testing.T, o *Org, h *fakeHerdr)
	before []string
}

// seedLayoutEvents appends evs to o's manifest.
func seedLayoutEvents(t *testing.T, o *Org, evs ...ManifestEvent) {
	t.Helper()
	for _, ev := range evs {
		if ev.TS == "" {
			ev.TS = "2026-11-06T00:00:00Z"
		}
		if err := o.Manifest.Append(ev); err != nil {
			t.Fatalf("seed %s: %v", ev.Event, err)
		}
	}
}

// seedHerdrPane puts pane in tab of workspace in herdr's view, labelled
// paneLabel ("" for none), with the tab labelled tabLabel.
func seedHerdrPane(h *fakeHerdr, pane, tab, workspace, paneLabel, tabLabel string) {
	setLabel(&h.paneTabs, pane, tab)
	setLabel(&h.paneWorkspaces, pane, workspace)
	if paneLabel != "" {
		setLabel(&h.paneLabels, pane, paneLabel)
	}
	setLabel(&h.tabLabels, tab, tabLabel)
}

// staleSpawnStarted is the spawn_started of org-a/seatID.
func staleSpawnStarted(seatID string) ManifestEvent {
	return ManifestEvent{OrgID: "org-a", SeatID: seatID, Event: EventSpawnStarted, Role: "worker", Driver: "claude", Model: "sonnet", Worktree: "/tmp/seat"}
}

// TestOrgSpawn_Layout_InterruptedAfterPaneCreated pins AC7 from the state a
// dead spawn leaves right after its pane_created spawn_step, in four ways of
// making the pane: the root pane of the workspace it made (its tab still
// "1"), a split in this version's tab, a split inside an earlier version's
// seat's own tab, and the pane of a tab it made. A respawn of that seat
// closes exactly that pane (compensateStale), then places the seat again,
// and afterwards the org's workspace holds no unlabelled pane but those that
// were there before the dead spawn. herdr's view drops what is closed, as
// herdr does (closeRemoves), so closing the root pane closes its workspace
// and the respawn makes a new one.
func TestOrgSpawn_Layout_InterruptedAfterPaneCreated(t *testing.T) {
	leaderSpawned := ManifestEvent{OrgID: "org-a", SeatID: LeaderIdentity, Event: EventSpawned, Role: LeaderIdentity, Driver: "claude", Model: "sonnet",
		Worktree: "/tmp/seat", PaneID: "p-leader", HerdrAgentName: herdrAgentName("org-a", LeaderIdentity)}
	wsCreated := ManifestEvent{OrgID: "org-a", Event: EventOrgWorkspaceCreated, PaneID: "ws-1", Worktree: "/tmp/seat"}
	created := func(seat, pane, how, tab string) ManifestEvent {
		return ManifestEvent{OrgID: "org-a", SeatID: seat, Event: EventSpawnStep, PaneID: pane, Details: paneCreatedDetails(how, tab)}
	}
	for _, tc := range []struct {
		name      string
		stale     staleLayout
		wantWS    string // the workspace the respawned seat runs in
		wantHow   string
		wantNewWS bool
	}{
		{name: "the root pane of the workspace it made", wantWS: "ws-2", wantHow: paneHowRoot, wantNewWS: true, stale: staleLayout{
			seat: LeaderIdentity, pane: "p-root",
			seed: func(t *testing.T, o *Org, h *fakeHerdr) {
				st := staleSpawnStarted(LeaderIdentity)
				st.Role = LeaderIdentity
				seedLayoutEvents(t, o, st, wsCreated, created(LeaderIdentity, "p-root", paneHowRoot, "t-root"))
				setLabel(&h.workspaceLabels, "ws-1", "org-a")
				seedHerdrPane(h, "p-root", "t-root", "ws-1", "", "1")
			}}},
		{name: "a split in this version's tab", wantWS: "ws-1", wantHow: paneHowSplitRight, stale: staleLayout{
			seat: implementerSeatID, pane: "p-split",
			seed: func(t *testing.T, o *Org, h *fakeHerdr) {
				seedLayoutEvents(t, o, wsCreated, leaderSpawned, staleSpawnStarted(implementerSeatID), created(implementerSeatID, "p-split", paneHowSplitRight, "t-org"))
				setLabel(&h.workspaceLabels, "ws-1", "org-a")
				seedHerdrPane(h, "p-leader", "t-org", "ws-1", LeaderIdentity, "org-a")
				seedHerdrPane(h, "p-split", "t-org", "ws-1", "", "org-a")
			}}},
		{name: "a split inside an earlier version's seat's tab", wantWS: "ws-1", wantHow: paneHowSplitRight, stale: staleLayout{
			seat: implementerSeatID, pane: "p-split", before: []string{"p-leader", "p-shell"},
			seed: func(t *testing.T, o *Org, h *fakeHerdr) {
				seedLayoutEvents(t, o, wsCreated, leaderSpawned, staleSpawnStarted(implementerSeatID), created(implementerSeatID, "p-split", paneHowSplitRight, "t-leader"))
				setLabel(&h.workspaceLabels, "ws-1", "org-a")
				seedHerdrPane(h, "p-shell", "t-1", "ws-1", "", "1")
				seedHerdrPane(h, "p-leader", "t-leader", "ws-1", "", LeaderIdentity)
				seedHerdrPane(h, "p-split", "t-leader", "ws-1", "", LeaderIdentity)
			}}},
		{name: "the pane of a tab it made", wantWS: "ws-1", wantHow: paneHowTab, stale: staleLayout{
			seat: implementerSeatID, pane: "p-tab", before: []string{"p-shell"},
			seed: func(t *testing.T, o *Org, h *fakeHerdr) {
				seedLayoutEvents(t, o, wsCreated, staleSpawnStarted(implementerSeatID), created(implementerSeatID, "p-tab", paneHowTab, "t-new"))
				setLabel(&h.workspaceLabels, "ws-1", "org-a")
				seedHerdrPane(h, "p-shell", "t-1", "ws-1", "", "1")
				seedHerdrPane(h, "p-tab", "t-new", "ws-1", "", implementerSeatID)
			}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			o, h, _ := layoutOrg(t, "p-new")
			h.closeRemoves = true
			h.workspaceID, h.paneID = "ws-2", "p-new"
			tc.stale.seed(t, o, h)

			p := mustSpawnParams("org-a", tc.stale.seat)
			if tc.stale.seat == LeaderIdentity {
				p = leaderParams("org-a")
			}
			r := o.Spawn(p)
			wantPane := "p-new"
			if tc.wantNewWS {
				wantPane = fakeRootPaneID("ws-2")
			}
			if r.Outcome != SpawnOutcomeSpawned || r.Seat.PaneID != wantPane {
				t.Fatalf("respawn = %+v; want spawned on %s", r, wantPane)
			}
			if !slices.Equal(h.paneCloseCalls, []string{tc.stale.pane}) || !slices.Contains(h.sendKeysCalls, tc.stale.pane) {
				t.Fatalf("panes closed %v, C-c %v; want %s alone closed", h.paneCloseCalls, h.sendKeysCalls, tc.stale.pane)
			}
			failed := eventsNamed(mustReadEvents(t, o), EventSpawnFailed)
			if len(failed) != 1 || failed[0].PaneID != tc.stale.pane {
				t.Fatalf("spawn_failed events = %+v; want one for %s", failed, tc.stale.pane)
			}
			assertDetailsContains(t, failed[0].Details, "superseded by respawn (previous_event=spawn_step compensation=ctrl_c=ok pane=closed)")
			if h.paneWorkspaces[wantPane] != tc.wantWS || h.paneLabels[wantPane] != tc.stale.seat {
				t.Fatalf("the new pane %s is in %q labelled %q; want %s labelled %s", wantPane, h.paneWorkspaces[wantPane], h.paneLabels[wantPane], tc.wantWS, tc.stale.seat)
			}
			if got := paneCreatedStep(t, o, "org-a", tc.stale.seat); !strings.HasPrefix(got, paneCreatedEvent+" how="+tc.wantHow+" ") {
				t.Fatalf("the respawn's pane_created step = %q, want how=%s", got, tc.wantHow)
			}
			for _, ws := range []string{"ws-1", "ws-2"} {
				want := []string(nil)
				if ws == "ws-1" {
					want = tc.stale.before
				}
				if got := unlabelledPanes(h, ws); !slices.Equal(got, want) {
					t.Fatalf("unlabelled panes of %s after the respawn = %v, want %v", ws, got, want)
				}
			}
			if tc.wantNewWS {
				if closed := eventsNamed(mustReadEvents(t, o), EventOrgWorkspaceClosed); len(closed) != 1 || closed[0].PaneID != "ws-1" || closed[0].Details != "not found at spawn" {
					t.Fatalf("org_workspace_closed = %+v; want ws-1 not found at spawn", closed)
				}
			}
		})
	}
}

// splitEnteredHerdr is a fakeHerdr whose first PaneSplit signals entered
// and then waits for release before it answers: a spawn held in the middle
// of making its pane, under the org's layout lock.
type splitEnteredHerdr struct {
	*fakeHerdr
	once    sync.Once
	entered chan struct{}
	release chan struct{}
}

func (h *splitEnteredHerdr) PaneSplit(ctx context.Context, paneID, direction, cwd string, extraEnv ...string) (driver.HerdrPaneRef, error) {
	first := false
	h.once.Do(func() { first = true })
	if first {
		close(h.entered)
		<-h.release
	}
	return h.fakeHerdr.PaneSplit(ctx, paneID, direction, cwd, extraEnv...)
}

// TestOrgSpawn_Layout_ConcurrentSpawnsShareOneTab pins AC9 and D3: two
// seats of one org spawned at once both land in the org's one tab and do
// not split the same pane. One spawn is held inside its PaneSplit, under
// the org's layout lock, until the other has recorded its spawn_started
// and had time to reach the lock; the second then waits, and once the
// first has recorded and labelled its pane, splits that pane down instead
// of the leader's. Two first spawns of a new org make one workspace.
func TestOrgSpawn_Layout_ConcurrentSpawnsShareOneTab(t *testing.T) {
	t.Run("after the leader", func(t *testing.T) {
		o, fh, _ := layoutOrg(t, "p-a", "p-b")
		mustSpawnLayout(t, o, leaderParams("org-a"), fakeRootPaneID("ws-1"))
		h := &splitEnteredHerdr{fakeHerdr: fh, entered: make(chan struct{}), release: make(chan struct{})}
		o.Herdr = h

		results := make(chan SpawnResult, 2)
		go func() { results <- o.Spawn(implementerParams("org-a")) }()
		<-h.entered
		go func() { results <- o.Spawn(reviewerParams("org-a")) }()
		waitForEvent(t, o, "reviewer", EventSpawnStarted)
		time.Sleep(100 * time.Millisecond)
		close(h.release)
		for range 2 {
			if r := <-results; r.Outcome != SpawnOutcomeSpawned {
				t.Fatalf("spawn = %+v; want spawned", r)
			}
		}

		if len(fh.paneSplitCalls) != 2 || fh.paneSplitCalls[0].paneID == fh.paneSplitCalls[1].paneID {
			t.Fatalf("splits = %+v; want two splits of different panes", fh.paneSplitCalls)
		}
		if got := []string{fh.paneSplitCalls[0].paneID + " " + fh.paneSplitCalls[0].direction, fh.paneSplitCalls[1].paneID + " " + fh.paneSplitCalls[1].direction}; !slices.Equal(got, []string{fakeRootPaneID("ws-1") + " right", "p-a down"}) {
			t.Fatalf("splits = %v; want the leader's pane right, then the implementer's down", got)
		}
		tab := fakeRootTabID("ws-1")
		for _, pane := range []string{"p-a", "p-b"} {
			if fh.paneTabs[pane] != tab {
				t.Fatalf("pane %s in tab %q, want %s", pane, fh.paneTabs[pane], tab)
			}
		}
		if n := countString(fh.calls, "tab_create"); n != 0 {
			t.Fatalf("tab_create called %d times, want none", n)
		}
	})
	t.Run("two first spawns", func(t *testing.T) {
		o, h, _ := layoutOrg(t, "p-b")
		results := make(chan SpawnResult, 2)
		var start sync.WaitGroup
		start.Add(1)
		for _, p := range []SpawnParams{leaderParams("org-a"), implementerParams("org-a")} {
			go func() {
				start.Wait()
				results <- o.Spawn(p)
			}()
		}
		start.Done()
		for range 2 {
			if r := <-results; r.Outcome != SpawnOutcomeSpawned {
				t.Fatalf("spawn = %+v; want spawned", r)
			}
		}
		if n := countString(h.calls, "workspace_create"); n != 1 || len(h.paneSplitCalls) != 1 || h.paneSplitCalls[0].paneID != fakeRootPaneID("ws-1") {
			t.Fatalf("workspace creates %d, splits %+v; want one workspace and the second seat split from its root pane", n, h.paneSplitCalls)
		}
		if open := openOrgWorkspaces(mustReadEvents(t, o), "org-a"); !slices.Equal(open, []string{"ws-1"}) {
			t.Fatalf("open workspaces = %v, want ws-1 alone", open)
		}
	})
}

// renameEnteredHerdr is a fakeHerdr whose first PaneRename that labels a
// pane with seat signals entered and then waits for release before it
// answers: a spawn that has recorded its pane and not yet labelled it,
// still under the org's layout lock.
type renameEnteredHerdr struct {
	*fakeHerdr
	seat    string
	once    sync.Once
	entered chan struct{}
	release chan struct{}
}

func (h *renameEnteredHerdr) PaneRename(ctx context.Context, paneID, label string) error {
	if label == h.seat {
		first := false
		h.once.Do(func() { first = true })
		if first {
			close(h.entered)
			<-h.release
		}
	}
	return h.fakeHerdr.PaneRename(ctx, paneID, label)
}

// TestOrgSpawn_Layout_LockHeldUntilThePaneIsLabelled pins the end of D3's
// lock: a spawn holds the org's layout lock until it has labelled its pane.
// The implementer's spawn is held inside its PaneRename, after its
// pane_created step; a reviewer spawn of the same org started then makes
// none of the layout calls (workspace_get, pane_list, pane_split,
// tab_create, workspace_create) within one second, and once released splits
// the implementer's pane down. In the /test report of plan
// 2026-11-06-org-herdr-single-tab, this test failed 10 of 10 runs against
// each of M1 (no lock between the org's seats), M2 (the lock released
// before the spawn_step), and M3 (released after the spawn_step and before
// PaneRename), and passed 10 of 10 with the lock as it is. With the lock no
// layout call can come in that second, so only a failing run depends on
// timing.
func TestOrgSpawn_Layout_LockHeldUntilThePaneIsLabelled(t *testing.T) {
	o, fh, _ := layoutOrg(t, "p-a", "p-b")
	mustSpawnLayout(t, o, leaderParams("org-a"), fakeRootPaneID("ws-1"))
	h := &renameEnteredHerdr{fakeHerdr: fh, seat: implementerSeatID, entered: make(chan struct{}), release: make(chan struct{})}
	o.Herdr = h

	results := make(chan SpawnResult, 2)
	go func() { results <- o.Spawn(implementerParams("org-a")) }()
	<-h.entered
	fh.mu.Lock()
	base := len(fh.calls)
	fh.mu.Unlock()
	go func() { results <- o.Spawn(reviewerParams("org-a")) }()
	waitForEvent(t, o, "reviewer", EventSpawnStarted)
	layoutCalls := []string{"workspace_get", "pane_list", "pane_split", "tab_create", "workspace_create"}
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		fh.mu.Lock()
		since := slices.Clone(fh.calls[base:])
		fh.mu.Unlock()
		for _, c := range since {
			if slices.Contains(layoutCalls, c) {
				close(h.release)
				t.Fatalf("herdr calls while the implementer's pane is unlabelled = %v; want no layout call", since)
			}
		}
		time.Sleep(5 * time.Millisecond)
	}
	close(h.release)
	for range 2 {
		if r := <-results; r.Outcome != SpawnOutcomeSpawned {
			t.Fatalf("spawn = %+v; want spawned", r)
		}
	}
	got := []string{}
	for _, c := range fh.paneSplitCalls {
		got = append(got, c.paneID+" "+c.direction)
	}
	if !slices.Equal(got, []string{fakeRootPaneID("ws-1") + " right", "p-a down"}) {
		t.Fatalf("splits = %v; want the leader's pane right, then the implementer's down", got)
	}
}

// waitForEvent waits up to 10 s for orgID "org-a"'s seatID to have event in
// o's manifest.
func waitForEvent(t *testing.T, o *Org, seatID, event string) {
	t.Helper()
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		for _, ev := range mustReadEvents(t, o) {
			if ev.OrgID == "org-a" && ev.SeatID == seatID && ev.Event == event {
				return
			}
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatalf("no %s of %s within 10s", event, seatID)
}

// TestOrgSpawn_Layout_RootPaneRenameFails pins D6 for the root pane: a
// failed pane rename closes the pane the spawn made, with no label check,
// and records spawn_failed at pane_rename with it. Closing the root pane
// closes the new workspace (closeRemoves), and the next spawn of the org
// records that workspace closed and makes a new one.
func TestOrgSpawn_Layout_RootPaneRenameFails(t *testing.T) {
	o, h, _ := layoutOrg(t)
	h.closeRemoves = true
	renameErr := errors.New("herdr pane rename: connect: connection refused")
	h.paneRenameErr = renameErr
	root := fakeRootPaneID("ws-1")

	r := o.Spawn(leaderParams("org-a"))
	if r.Outcome != SpawnOutcomeFailed || !errors.Is(r.Err, renameErr) || !strings.Contains(r.Err.Error(), "spawn step pane_rename failed") {
		t.Fatalf("spawn = %+v; want a pane_rename failure", r)
	}
	if !slices.Equal(h.sendKeysCalls, []string{root}) || !slices.Equal(h.paneCloseCalls, []string{root}) {
		t.Fatalf("C-c %v, closes %v; want the root pane %s", h.sendKeysCalls, h.paneCloseCalls, root)
	}
	if want := []string{"workspace_create", "tab_rename", "pane_rename", "pane_send_keys", "pane_close"}; !slices.Equal(h.calls, want) {
		t.Fatalf("herdr calls = %v, want %v (no label check before the close)", h.calls, want)
	}
	last := lastEvent(t, o)
	if last.Event != EventSpawnFailed || last.PaneID != root {
		t.Fatalf("last event = %+v; want spawn_failed recording %s", last, root)
	}
	assertDetailsContains(t, last.Details, "step=pane_rename ", "compensation=ctrl_c=ok pane=closed")

	h.paneRenameErr = nil
	h.workspaceID = "ws-2"
	mustSpawnLayout(t, o, leaderParams("org-a"), fakeRootPaneID("ws-2"))
	closed := eventsNamed(mustReadEvents(t, o), EventOrgWorkspaceClosed)
	if len(closed) != 1 || closed[0].PaneID != "ws-1" || closed[0].Details != "not found at spawn" {
		t.Fatalf("org_workspace_closed = %+v; want ws-1 not found at spawn", closed)
	}
}

// TestOrgSpawn_Layout_TabRenameFailsSpawnGoesOn pins D6 for the tab: a
// failed tab rename of a new workspace's tab does not fail the spawn; the
// pane_created step notes it after its three words, which paneCreatedTab
// still reads.
func TestOrgSpawn_Layout_TabRenameFailsSpawnGoesOn(t *testing.T) {
	o, h, _ := layoutOrg(t)
	h.tabRenameErr = errors.New("herdr: tab_not_found")
	root, tab := fakeRootPaneID("ws-1"), fakeRootTabID("ws-1")
	mustSpawnLayout(t, o, leaderParams("org-a"), root)

	got := paneCreatedStep(t, o, "org-a", LeaderIdentity)
	if want := paneCreatedDetails(paneHowRoot, tab) + " tab_rename=failed: herdr: tab_not_found"; got != want {
		t.Fatalf("pane_created step = %q, want %q", got, want)
	}
	if read, ok := paneCreatedTab(got); !ok || read != tab {
		t.Fatalf("paneCreatedTab = %q, %v; want %s", read, ok, tab)
	}
	if h.paneLabels[root] != LeaderIdentity || h.tabLabels[tab] != "1" {
		t.Fatalf("pane label %q, tab label %q; want leader and herdr's 1", h.paneLabels[root], h.tabLabels[tab])
	}
}

// TestOrgSpawn_Layout_FailuresBeforeThePane pins D6 for the steps that fail
// before a pane exists: each records spawn_failed at its step with no pane
// to close and sends nothing to a pane. The split's pane closed between the
// list and the split (pane_not_found, R3) is one of them; a workspace
// create reply with no root pane, or with a root pane and no tab, leaves the
// workspace recorded open, for a disband to close.
func TestOrgSpawn_Layout_FailuresBeforeThePane(t *testing.T) {
	refused := errors.New("herdr: connect: connection refused")
	for _, tc := range []struct {
		name   string
		leader bool // fail the leader's spawn (the org's first); else the implementer's
		inject func(o *Org, h *fakeHerdr)
		step   string
	}{
		{"pane list fails", false, func(_ *Org, h *fakeHerdr) { h.paneListErr = refused }, "pane_list"},
		{"the pane to split vanished", false, func(_ *Org, h *fakeHerdr) {
			h.paneSplitErr = driver.NewHerdrError(driver.HerdrCodePaneNotFound, "pane "+fakeRootPaneID("ws-1")+" not found")
		}, "pane_split"},
		{"workspace create fails", true, func(_ *Org, h *fakeHerdr) { h.workspaceCreateErr = refused }, "workspace_create"},
		{"workspace create names no root pane", true, func(o *Org, h *fakeHerdr) { o.Herdr = &noRootPaneHerdr{h} }, "workspace_create"},
		{"workspace create names a root pane and no tab", true, func(o *Org, h *fakeHerdr) { o.Herdr = &noTabHerdr{h} }, "workspace_create"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			o, h, _ := layoutOrg(t)
			p := leaderParams("org-a")
			if !tc.leader {
				mustSpawnLayout(t, o, p, fakeRootPaneID("ws-1"))
				p = implementerParams("org-a")
			}
			tc.inject(o, h)
			keysBefore := len(h.sendKeysCalls)
			r := o.Spawn(p)
			if r.Outcome != SpawnOutcomeFailed || !strings.Contains(r.Err.Error(), "spawn step "+tc.step+" failed") {
				t.Fatalf("spawn = %+v; want a %s failure", r, tc.step)
			}
			last := lastEvent(t, o)
			if last.Event != EventSpawnFailed || last.SeatID != p.SeatID || last.PaneID != "" {
				t.Fatalf("last event = %+v; want spawn_failed of %s with no pane", last, p.SeatID)
			}
			assertDetailsContains(t, last.Details, "step="+tc.step+" ", "compensation=no pane to compensate")
			if len(h.sendKeysCalls) != keysBefore || len(h.paneCloseCalls) != 0 {
				t.Fatalf("C-c %v, closes %v; want nothing sent", h.sendKeysCalls[keysBefore:], h.paneCloseCalls)
			}
			if strings.HasPrefix(tc.name, "workspace create names ") {
				if open := openOrgWorkspaces(mustReadEvents(t, o), "org-a"); !slices.Equal(open, []string{"ws-1"}) {
					t.Fatalf("open workspaces = %v, want ws-1 recorded open", open)
				}
			}
		})
	}
}

// noRootPaneHerdr is a fakeHerdr whose WorkspaceCreate reply names no root
// pane and no tab, as driver.Herdr's does for a reply without them.
type noRootPaneHerdr struct{ *fakeHerdr }

func (h *noRootPaneHerdr) WorkspaceCreate(ctx context.Context, cwd, label string, extraEnv ...string) (driver.HerdrWorkspace, error) {
	ws, err := h.fakeHerdr.WorkspaceCreate(ctx, cwd, label, extraEnv...)
	return driver.HerdrWorkspace{WorkspaceID: ws.WorkspaceID}, err
}

// noTabHerdr is a fakeHerdr whose WorkspaceCreate reply names the root pane
// and no tab, so a check of the root pane alone would let the spawn go on
// (P5 of the /test report of plan 2026-11-06-org-herdr-single-tab).
type noTabHerdr struct{ *fakeHerdr }

func (h *noTabHerdr) WorkspaceCreate(ctx context.Context, cwd, label string, extraEnv ...string) (driver.HerdrWorkspace, error) {
	ws, err := h.fakeHerdr.WorkspaceCreate(ctx, cwd, label, extraEnv...)
	ws.TabID = ""
	return ws, err
}

// TestOrgSpawn_Layout_NoLiveSeatPaneMakesATab pins D1's last case: in a
// workspace the manifest records open and herdr labels with the org_id but
// that holds no live seat's pane -- an earlier version's workspace whose
// seats have stopped, with its root shell left in tab "1", and a seat the
// manifest still has active whose pane herdr no longer lists -- the spawn
// makes a tab for the seat, labelled with the seat id, with the seat's
// RALPH_PRIMARY_CLI, and labels its pane.
func TestOrgSpawn_Layout_NoLiveSeatPaneMakesATab(t *testing.T) {
	o, h, _ := layoutOrg(t)
	seedOpenOrgWorkspace(t, o, h, "org-a", "ws-1")
	seedSeat(t, o, "org-a", "seat-gone", "worker", "p-gone")
	h.paneID = "p-tab"
	p := mustSpawnParams("org-a", "seat-1")
	p.Driver, p.Model = "codex", "gpt-5-codex"
	o.Config.Permissions.CodexVerified = true
	mustSpawnLayout(t, o, p, "p-tab")

	if want := []string{"workspace_get", "pane_list", "tab_create", "pane_rename", "agent_start"}; !slices.Equal(h.calls, want) {
		t.Fatalf("herdr calls = %v, want %v", h.calls, want)
	}
	if !slices.Equal(h.tabCreateWorkspaces, []string{"ws-1"}) || !slices.EqualFunc(h.tabCreateEnvs, [][]string{{"RALPH_PRIMARY_CLI=codex"}}, slices.Equal) {
		t.Fatalf("TabCreate workspaces %v, extraEnv %q; want ws-1 with RALPH_PRIMARY_CLI=codex", h.tabCreateWorkspaces, h.tabCreateEnvs)
	}
	if h.tabLabels[fakeTabID("p-tab")] != "seat-1" || h.paneLabels["p-tab"] != "seat-1" {
		t.Fatalf("tab label %q, pane label %q; want seat-1 on both", h.tabLabels[fakeTabID("p-tab")], h.paneLabels["p-tab"])
	}
	if got := paneCreatedStep(t, o, "org-a", "seat-1"); got != paneCreatedDetails(paneHowTab, fakeTabID("p-tab")) {
		t.Fatalf("pane_created step = %q, want how=tab in %s", got, fakeTabID("p-tab"))
	}
}

// TestOrgSpawn_Layout_SplitParent pins D1's choice of the pane to split,
// from the roster and herdr's pane list: none with no candidate, the one
// candidate to the right, and, of two or more, the last spawned non-leader
// seat's pane down -- the leader's only when every candidate is the
// leader's. A seat whose pane herdr does not list, a stopped seat, the
// spawning seat itself, another org's seat, and a dry-run seat are no
// candidates.
func TestOrgSpawn_Layout_SplitParent(t *testing.T) {
	started := func(org, seat string) ManifestEvent {
		return ManifestEvent{OrgID: org, SeatID: seat, Event: EventSpawnStarted}
	}
	on := func(org, seat, event, pane string) ManifestEvent {
		return ManifestEvent{OrgID: org, SeatID: seat, Event: event, PaneID: pane}
	}
	listed := func(ids ...string) []driver.HerdrPane {
		out := make([]driver.HerdrPane, 0, len(ids))
		for _, id := range ids {
			out = append(out, driver.HerdrPane{PaneID: id, TabID: "t", WorkspaceID: "ws-1"})
		}
		return out
	}
	dryRun := on("org-a", "seat-dry", EventSpawned, "p-dry")
	dryRun.DryRun = true
	for _, tc := range []struct {
		name     string
		events   []ManifestEvent
		panes    []driver.HerdrPane
		wantPane string
		wantHow  string
	}{
		{"no seat", nil, listed("p-shell"), "", ""},
		{"one candidate", []ManifestEvent{started("org-a", LeaderIdentity), on("org-a", LeaderIdentity, EventSpawned, "p-l")},
			listed("p-l"), "p-l", paneHowSplitRight},
		{"one non-leader candidate", []ManifestEvent{started("org-a", "reviewer"), on("org-a", "reviewer", EventSpawned, "p-r")},
			listed("p-r"), "p-r", paneHowSplitRight},
		{"two: the non-leader", []ManifestEvent{
			started("org-a", LeaderIdentity), on("org-a", LeaderIdentity, EventSpawned, "p-l"),
			started("org-a", "reviewer"), on("org-a", "reviewer", EventSpawned, "p-r"),
		}, listed("p-l", "p-r"), "p-r", paneHowSplitDown},
		{"two: the non-leader, the leader spawned again after it", []ManifestEvent{
			started("org-a", "reviewer"), on("org-a", "reviewer", EventSpawned, "p-r"),
			started("org-a", LeaderIdentity), on("org-a", LeaderIdentity, EventSpawned, "p-l"),
		}, listed("p-l", "p-r"), "p-r", paneHowSplitDown},
		{"three: the last spawned non-leader, by spawn_started", []ManifestEvent{
			started("org-a", LeaderIdentity), started("org-a", "zz-first"), started("org-a", "aa-second"),
			on("org-a", "aa-second", EventSpawned, "p-2"), on("org-a", LeaderIdentity, EventSpawned, "p-l"), on("org-a", "zz-first", EventSpawned, "p-1"),
		}, listed("p-l", "p-1", "p-2"), "p-2", paneHowSplitDown},
		{"an in-flight seat counts", []ManifestEvent{
			started("org-a", LeaderIdentity), on("org-a", LeaderIdentity, EventSpawned, "p-l"),
			started("org-a", "seat-2"), on("org-a", "seat-2", EventSpawnStep, "p-2"),
		}, listed("p-l", "p-2"), "p-2", paneHowSplitDown},
		{"a recorded pane herdr does not list", []ManifestEvent{
			started("org-a", LeaderIdentity), on("org-a", LeaderIdentity, EventSpawned, "p-l"),
			started("org-a", "seat-2"), on("org-a", "seat-2", EventSpawned, "p-gone"),
		}, listed("p-l"), "p-l", paneHowSplitRight},
		{"no candidate but itself, a stopped seat, another org's, a dry run", []ManifestEvent{
			started("org-a", "seat-1"), on("org-a", "seat-1", EventSpawnStep, "p-self"),
			started("org-a", "seat-stopped"), on("org-a", "seat-stopped", EventStopped, "p-stopped"),
			started("org-b", "seat-b"), on("org-b", "seat-b", EventSpawned, "p-b"),
			dryRun,
		}, listed("p-self", "p-stopped", "p-b", "p-dry"), "", ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			pane, how := splitParent(tc.events, "org-a", "seat-1", tc.panes)
			if pane != tc.wantPane || how != tc.wantHow {
				t.Fatalf("splitParent = %q, %q; want %q, %q", pane, how, tc.wantPane, tc.wantHow)
			}
		})
	}
}

// TestOrgSpawn_Layout_LockHeldPastTheDeadline pins D3's wait: while another
// holder keeps the org's layout lock past the spawn's deadline, the spawn
// fails at layout_lock with the deadline hint and spawn_failed, having
// asked herdr nothing; a spawn of another org takes its own lock and goes
// on.
func TestOrgSpawn_Layout_LockHeldPastTheDeadline(t *testing.T) {
	o, h, _ := layoutOrg(t)
	lock, err := acquireHeldFileLock(filepath.Dir(o.Manifest.Path()), layoutLockName("org-a"), "layout", 0)
	if err != nil {
		t.Fatalf("take the layout lock: %v", err)
	}
	defer func() { _ = lock.release() }()

	p := leaderParams("org-a")
	p.TimeoutMS = 300
	r := o.Spawn(p)
	if r.Outcome != SpawnOutcomeFailed || !errors.Is(r.Err, errFileLockHeld) || !strings.Contains(r.Err.Error(), "spawn step layout_lock failed") ||
		!strings.Contains(r.Err.Error(), spawnDeadlineHint(300)) {
		t.Fatalf("spawn = %+v; want a layout_lock failure with the deadline hint", r)
	}
	if len(h.calls) != 0 {
		t.Fatalf("herdr calls = %v, want none", h.calls)
	}
	last := lastEvent(t, o)
	if last.Event != EventSpawnFailed || last.SeatID != LeaderIdentity {
		t.Fatalf("last event = %+v; want the leader's spawn_failed", last)
	}
	assertDetailsContains(t, last.Details, "step=layout_lock ", "compensation=no pane to compensate")

	mustSpawnLayout(t, o, leaderParams("org-b"), fakeRootPaneID("ws-1"))
}

// TestOrgSpawn_Layout_DryRunAsksHerdrNothing: a dry run takes no layout
// lock and calls no herdr, and its trail's step says pane_created alone,
// which names no tab.
func TestOrgSpawn_Layout_DryRunAsksHerdrNothing(t *testing.T) {
	o, h, _ := layoutOrg(t)
	lock, err := acquireHeldFileLock(filepath.Dir(o.Manifest.Path()), layoutLockName("org-a"), "layout", 0)
	if err != nil {
		t.Fatalf("take the layout lock: %v", err)
	}
	defer func() { _ = lock.release() }()

	p := leaderParams("org-a")
	p.DryRun = true
	if r := o.Spawn(p); r.Outcome != SpawnOutcomeSpawned || !r.Seat.DryRun {
		t.Fatalf("dry run = %+v; want a dry-run spawned", r)
	}
	if len(h.calls) != 0 {
		t.Fatalf("herdr calls = %v, want none", h.calls)
	}
	var steps []string
	for _, ev := range mustReadEvents(t, o) {
		if ev.Event == EventSpawnStep && ev.DryRun {
			steps = append(steps, ev.Details)
		}
	}
	if len(steps) == 0 || steps[0] != paneCreatedEvent {
		t.Fatalf("dry-run steps = %q, want pane_created first", steps)
	}
	if _, ok := paneCreatedTab(steps[0]); ok {
		t.Fatalf("paneCreatedTab(%q) reads a tab, want none", steps[0])
	}
}
