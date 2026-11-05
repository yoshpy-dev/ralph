package cli

import (
	"bytes"
	"context"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/yoshpy-dev/ralph/internal/org/driver"
)

// The tests in this file drive the CLI tests' fake herdr (herdrStub in
// org_test.go) through driver.Herdr and the real ExecRunner, so the pane
// operations the org runtime sends once it puts seats in split panes (plan
// 2026-11-06-org-herdr-single-tab, D1 and D8) get the replies herdr 0.7.5
// gives (the plan's Assumptions) before any CLI test depends on them.

// stubHerdr runs setupOrgStubPATH and returns a Herdr that runs the stub, and
// the stub's argv log.
func stubHerdr(t *testing.T) (driver.Herdr, string) {
	t.Helper()
	herdrLog, _ := setupOrgStubPATH(t)
	return driver.Herdr{R: driver.ExecRunner{}}, herdrLog
}

// mustPaneList returns the panes the stub lists for workspaceID.
func mustPaneList(t *testing.T, h driver.Herdr, workspaceID string) []driver.HerdrPane {
	t.Helper()
	panes, err := h.PaneList(context.Background(), workspaceID)
	if err != nil {
		t.Fatalf("pane list --workspace %s: %v", workspaceID, err)
	}
	return panes
}

// mustSplit splits paneID in direction and returns the new pane.
func mustSplit(t *testing.T, h driver.Herdr, paneID, direction, cwd string) driver.HerdrPaneRef {
	t.Helper()
	ref, err := h.PaneSplit(context.Background(), paneID, direction, cwd)
	if err != nil {
		t.Fatalf("pane split %s --direction %s: %v", paneID, direction, err)
	}
	return ref
}

// workspace create records the root pane its reply names, in a tab labelled
// "1", so pane get, tab get, and pane list know it, and takes --env.
func TestOrgHerdrStub_WorkspaceCreateRecordsTheRootPane(t *testing.T) {
	h, herdrLog := stubHerdr(t)
	ctx := context.Background()
	cwd := t.TempDir()

	ws, err := h.WorkspaceCreate(ctx, cwd, "org-a", "RALPH_PRIMARY_CLI=claude")
	if err != nil {
		t.Fatalf("workspace create: %v", err)
	}
	if want := (driver.HerdrWorkspace{WorkspaceID: "ws-stub-1", TabID: "ws-stub-1:t1", RootPaneID: "ws-stub-1:p1"}); ws != want {
		t.Fatalf("workspace create = %+v; want %+v", ws, want)
	}
	root := driver.HerdrPane{PaneID: "ws-stub-1:p1", TabID: "ws-stub-1:t1", WorkspaceID: "ws-stub-1"}
	if got := mustPaneList(t, h, ws.WorkspaceID); !reflect.DeepEqual(got, []driver.HerdrPane{root}) {
		t.Fatalf("pane list = %+v; want only the root pane %+v", got, root)
	}
	if got, err := h.PaneGet(ctx, ws.RootPaneID); err != nil || got != root {
		t.Fatalf("pane get %s = %+v, %v; want %+v with no label", ws.RootPaneID, got, err, root)
	}
	if label, err := h.TabGet(ctx, ws.TabID); err != nil || label != "1" {
		t.Fatalf("tab get %s = %q, %v; want \"1\"", ws.TabID, label, err)
	}
	if label, err := h.WorkspaceGet(ctx, ws.WorkspaceID); err != nil || label != "org-a" {
		t.Fatalf("workspace get %s = %q, %v; want \"org-a\"", ws.WorkspaceID, label, err)
	}
	// The --help of the --env check is not logged, as for tab create.
	lines := readLogLines(t, herdrLog)
	if want := "workspace create --cwd " + cwd + " --label org-a --env RALPH_PRIMARY_CLI=claude"; len(lines) == 0 || lines[0] != want {
		t.Fatalf("herdr calls = %q; want the first to be %q", lines, want)
	}
}

// pane split makes a new pane in the tab and the workspace of the pane it
// splits, and pane list reports every pane of the workspace in the order the
// stub made them (tab create's included), then a pane a test recorded
// itself, and none of another workspace.
func TestOrgHerdrStub_PaneSplitAddsPanesToTheTabOfThePaneItSplits(t *testing.T) {
	h, herdrLog := stubHerdr(t)
	ctx := context.Background()
	cwd := t.TempDir()

	ws, err := h.WorkspaceCreate(ctx, cwd, "org-a")
	if err != nil {
		t.Fatalf("workspace create: %v", err)
	}
	right := mustSplit(t, h, ws.RootPaneID, driver.HerdrSplitRight, cwd)
	down := mustSplit(t, h, right.PaneID, driver.HerdrSplitDown, cwd)
	if want := (driver.HerdrPaneRef{PaneID: "pane-split-1", TabID: ws.TabID}); right != want {
		t.Fatalf("split right = %+v; want %+v", right, want)
	}
	if want := (driver.HerdrPaneRef{PaneID: "pane-split-2", TabID: ws.TabID}); down != want {
		t.Fatalf("split down = %+v; want %+v", down, want)
	}
	inTab := func(paneID, tabID string) driver.HerdrPane {
		return driver.HerdrPane{PaneID: paneID, TabID: tabID, WorkspaceID: ws.WorkspaceID}
	}
	if got, want := mustPaneList(t, h, ws.WorkspaceID), []driver.HerdrPane{
		inTab(ws.RootPaneID, ws.TabID), inTab(right.PaneID, ws.TabID), inTab(down.PaneID, ws.TabID),
	}; !reflect.DeepEqual(got, want) {
		t.Fatalf("pane list after two splits = %+v; want three panes in tab %s: %+v", got, ws.TabID, want)
	}
	if got, err := h.PaneGet(ctx, down.PaneID); err != nil || got != inTab(down.PaneID, ws.TabID) {
		t.Fatalf("pane get %s = %+v, %v; want it in tab %s", down.PaneID, got, err, ws.TabID)
	}

	// A pane tab create makes, and one split from it, are in their own tab.
	tab, err := h.TabCreate(ctx, ws.WorkspaceID, cwd, "seat-1")
	if err != nil {
		t.Fatalf("tab create: %v", err)
	}
	fromTab := mustSplit(t, h, tab.PaneID, driver.HerdrSplitRight, cwd)
	if want := (driver.HerdrPaneRef{PaneID: "pane-split-3", TabID: "tab-stub-1"}); fromTab != want {
		t.Fatalf("split of the tab create pane = %+v; want %+v", fromTab, want)
	}
	// A pane recorded the way seedSeat records one, and a pane of another
	// workspace.
	recordStubHerdr(t, "pane-p-seeded", "tab-p-seeded "+ws.WorkspaceID)
	other, err := h.WorkspaceCreate(ctx, cwd, "org-b")
	if err != nil {
		t.Fatalf("workspace create org-b: %v", err)
	}
	if got, want := mustPaneList(t, h, ws.WorkspaceID), []driver.HerdrPane{
		inTab(ws.RootPaneID, ws.TabID), inTab(right.PaneID, ws.TabID), inTab(down.PaneID, ws.TabID),
		inTab(tab.PaneID, "tab-stub-1"), inTab(fromTab.PaneID, "tab-stub-1"), inTab("p-seeded", "tab-p-seeded"),
	}; !reflect.DeepEqual(got, want) {
		t.Fatalf("pane list = %+v; want %+v", got, want)
	}
	if got, want := mustPaneList(t, h, other.WorkspaceID), []driver.HerdrPane{
		{PaneID: other.RootPaneID, TabID: other.TabID, WorkspaceID: other.WorkspaceID},
	}; !reflect.DeepEqual(got, want) {
		t.Fatalf("pane list of %s = %+v; want only its root pane %+v", other.WorkspaceID, got, want)
	}

	wantSplit := "pane split " + ws.RootPaneID + " --direction right --ratio 0.5 --cwd " + cwd + " --no-focus"
	if lines := readLogLines(t, herdrLog); !slices.Contains(lines, wantSplit) {
		t.Fatalf("herdr calls = %q; want %q among them", lines, wantSplit)
	}
}

// pane rename and tab rename set the labels pane get, pane list, and tab get
// report.
func TestOrgHerdrStub_RenameSetsTheLabels(t *testing.T) {
	h, _ := stubHerdr(t)
	ctx := context.Background()
	cwd := t.TempDir()

	ws, err := h.WorkspaceCreate(ctx, cwd, "org-a")
	if err != nil {
		t.Fatalf("workspace create: %v", err)
	}
	split := mustSplit(t, h, ws.RootPaneID, driver.HerdrSplitRight, cwd)
	if err := h.PaneRename(ctx, ws.RootPaneID, "leader"); err != nil {
		t.Fatalf("pane rename: %v", err)
	}
	if err := h.TabRename(ctx, ws.TabID, "org-a"); err != nil {
		t.Fatalf("tab rename: %v", err)
	}

	root := driver.HerdrPane{PaneID: ws.RootPaneID, TabID: ws.TabID, WorkspaceID: ws.WorkspaceID, Label: "leader"}
	if got, err := h.PaneGet(ctx, ws.RootPaneID); err != nil || got != root {
		t.Fatalf("pane get %s = %+v, %v; want %+v", ws.RootPaneID, got, err, root)
	}
	if got, want := mustPaneList(t, h, ws.WorkspaceID), []driver.HerdrPane{
		root, {PaneID: split.PaneID, TabID: ws.TabID, WorkspaceID: ws.WorkspaceID},
	}; !reflect.DeepEqual(got, want) {
		t.Fatalf("pane list = %+v; want the label on the root pane only: %+v", got, want)
	}
	if label, err := h.TabGet(ctx, ws.TabID); err != nil || label != "org-a" {
		t.Fatalf("tab get %s = %q, %v; want \"org-a\"", ws.TabID, label, err)
	}
}

// pane close forgets the pane, then its tab once no pane is left in it, and
// the workspace once no pane is left there, as herdr 0.7.5 does; it keeps a
// pane tab create made, as the stub did before (see herdrStub), so the
// workspace that holds one stays.
func TestOrgHerdrStub_PaneCloseForgetsThePaneThenItsTabAndWorkspace(t *testing.T) {
	h, _ := stubHerdr(t)
	ctx := context.Background()
	cwd := t.TempDir()

	mustClose := func(paneID string) {
		t.Helper()
		if err := h.PaneClose(ctx, paneID); err != nil {
			t.Fatalf("pane close %s: %v", paneID, err)
		}
	}
	mustBeGone := func(paneID string) {
		t.Helper()
		if _, err := h.PaneGet(ctx, paneID); !driver.IsNotFound(err) {
			t.Fatalf("pane get %s after its close: err = %v; want not found", paneID, err)
		}
	}
	paneIDs := func(workspaceID string) []string {
		t.Helper()
		var ids []string
		for _, p := range mustPaneList(t, h, workspaceID) {
			ids = append(ids, p.PaneID)
		}
		return ids
	}

	// A workspace with a pane tab create made: closing every other pane
	// forgets them and their tab, and the workspace stays.
	ws, err := h.WorkspaceCreate(ctx, cwd, "org-a")
	if err != nil {
		t.Fatalf("workspace create: %v", err)
	}
	split := mustSplit(t, h, ws.RootPaneID, driver.HerdrSplitRight, cwd)
	tab, err := h.TabCreate(ctx, ws.WorkspaceID, cwd, "seat-1")
	if err != nil {
		t.Fatalf("tab create: %v", err)
	}

	mustClose(split.PaneID)
	mustBeGone(split.PaneID)
	if label, err := h.TabGet(ctx, ws.TabID); err != nil || label != "1" {
		t.Fatalf("tab get %s with a pane left in it = %q, %v; want \"1\"", ws.TabID, label, err)
	}
	if got, want := paneIDs(ws.WorkspaceID), []string{ws.RootPaneID, tab.PaneID}; !slices.Equal(got, want) {
		t.Fatalf("pane list = %q; want %q", got, want)
	}

	mustClose(tab.PaneID)
	kept := driver.HerdrPane{PaneID: tab.PaneID, TabID: tab.TabID, WorkspaceID: ws.WorkspaceID}
	if got, err := h.PaneGet(ctx, tab.PaneID); err != nil || got != kept {
		t.Fatalf("pane get %s after its close = %+v, %v; want the tab create pane kept: %+v", tab.PaneID, got, err, kept)
	}
	if label, err := h.TabGet(ctx, tab.TabID); err != nil || label != "seat-1" {
		t.Fatalf("tab get %s after its pane's close = %q, %v; want \"seat-1\"", tab.TabID, label, err)
	}

	mustClose(ws.RootPaneID)
	mustBeGone(ws.RootPaneID)
	if _, err := h.TabGet(ctx, ws.TabID); !driver.IsNotFound(err) {
		t.Fatalf("tab get %s after its last pane closed: err = %v; want not found", ws.TabID, err)
	}
	if label, err := h.WorkspaceGet(ctx, ws.WorkspaceID); err != nil || label != "org-a" {
		t.Fatalf("workspace get %s with a pane left in it = %q, %v; want \"org-a\"", ws.WorkspaceID, label, err)
	}
	if got, want := paneIDs(ws.WorkspaceID), []string{tab.PaneID}; !slices.Equal(got, want) {
		t.Fatalf("pane list = %q; want %q", got, want)
	}

	// A workspace of a root pane and a split: closing both forgets it.
	other, err := h.WorkspaceCreate(ctx, cwd, "org-b")
	if err != nil {
		t.Fatalf("workspace create org-b: %v", err)
	}
	down := mustSplit(t, h, other.RootPaneID, driver.HerdrSplitDown, cwd)
	mustClose(other.RootPaneID)
	if label, err := h.WorkspaceGet(ctx, other.WorkspaceID); err != nil || label != "org-b" {
		t.Fatalf("workspace get %s with a pane left in it = %q, %v; want \"org-b\"", other.WorkspaceID, label, err)
	}
	mustClose(down.PaneID)
	mustBeGone(down.PaneID)
	if _, err := h.WorkspaceGet(ctx, other.WorkspaceID); !driver.IsNotFound(err) {
		t.Fatalf("workspace get %s after its last pane closed: err = %v; want not found", other.WorkspaceID, err)
	}
	if _, err := h.TabGet(ctx, other.TabID); !driver.IsNotFound(err) {
		t.Fatalf("tab get %s after its last pane closed: err = %v; want not found", other.TabID, err)
	}
	if _, err := h.PaneList(ctx, other.WorkspaceID); !driver.IsNotFound(err) {
		t.Fatalf("pane list --workspace %s after its last pane closed: err = %v; want not found", other.WorkspaceID, err)
	}
	if label, err := h.WorkspaceGet(ctx, ws.WorkspaceID); err != nil || label != "org-a" {
		t.Fatalf("workspace get %s after another workspace's last pane closed = %q, %v; want \"org-a\"", ws.WorkspaceID, label, err)
	}
}

// A split, rename, or list of an id the stub does not know answers herdr's
// not-found envelope, and the failed split hands out no pane id.
func TestOrgHerdrStub_UnknownIDsAreNotFound(t *testing.T) {
	h, _ := stubHerdr(t)
	ctx := context.Background()
	cwd := t.TempDir()

	for _, tc := range []struct {
		name string
		call func() error
	}{
		{"pane split", func() error {
			_, err := h.PaneSplit(ctx, "no-such-pane", driver.HerdrSplitRight, cwd)
			return err
		}},
		{"pane rename", func() error { return h.PaneRename(ctx, "no-such-pane", "seat-1") }},
		{"tab rename", func() error { return h.TabRename(ctx, "no-such-tab", "org-a") }},
		{"pane list", func() error {
			_, err := h.PaneList(ctx, "no-such-workspace")
			return err
		}},
	} {
		if err := tc.call(); !driver.IsNotFound(err) {
			t.Errorf("%s of an unknown id: err = %v; want not found", tc.name, err)
		}
	}

	ws, err := h.WorkspaceCreate(ctx, cwd, "org-a")
	if err != nil {
		t.Fatalf("workspace create: %v", err)
	}
	if got := mustSplit(t, h, ws.RootPaneID, driver.HerdrSplitDown, cwd); got.PaneID != "pane-split-1" {
		t.Fatalf("first split after a failed one = %+v; want pane-split-1", got)
	}
}

// workspace create --help and pane split --help list --env, as tab create's
// does, so a Herdr with PaneEnv adds it; ORG_STUB_TAB_CREATE_HELP=noenv takes
// it out of all three helps, so the calls carry no --env and the Herdr warns
// once per subcommand.
func TestOrgHerdrStub_HelpsListEnvUnlessNoenv(t *testing.T) {
	for _, tc := range []struct {
		help    string
		wantEnv bool
	}{
		{"", true},
		{"noenv", false},
	} {
		t.Run("help="+tc.help, func(t *testing.T) {
			_, herdrLog := stubHerdr(t)
			t.Setenv("ORG_STUB_TAB_CREATE_HELP", tc.help)
			var warn bytes.Buffer
			h := driver.NewHerdr(driver.ExecRunner{}, "AGMSG_SELF_NAME=off")
			h.Warn = &warn
			ctx := context.Background()
			cwd := t.TempDir()

			ws, err := h.WorkspaceCreate(ctx, cwd, "org-a", "RALPH_PRIMARY_CLI=claude")
			if err != nil {
				t.Fatalf("workspace create: %v", err)
			}
			mustSplit(t, h, ws.RootPaneID, driver.HerdrSplitRight, cwd)
			if _, err := h.TabCreate(ctx, ws.WorkspaceID, cwd, "seat-1"); err != nil {
				t.Fatalf("tab create: %v", err)
			}

			lines := readLogLines(t, herdrLog)
			if len(lines) != 3 {
				t.Fatalf("herdr calls = %q; want the three calls, no --help", lines)
			}
			for _, l := range lines {
				if got := strings.Contains(l, " --env AGMSG_SELF_NAME=off"); got != tc.wantEnv {
					t.Errorf("herdr call %q has --env: %v; want %v", l, got, tc.wantEnv)
				}
			}
			for _, sub := range []string{"workspace create", "pane split", "tab create"} {
				warned := strings.Contains(warn.String(), "herdr "+sub+" has no --env flag")
				if warned == tc.wantEnv {
					t.Errorf("warning for %s: %v; want %v (warnings %q)", sub, warned, !tc.wantEnv, warn.String())
				}
			}
		})
	}
}
