package org

import (
	"slices"
	"testing"

	"github.com/yoshpy-dev/ralph/internal/config"
)

// Tests of the claude retry entries of a seat's pane (spec
// docs/specs/2026-11-07-org-seat-resume.md, FR-8 and FR-9; plan
// 2026-11-07-org-seat-resume, AC-8 and S3). testOrgConfig sets no
// [org].api_wait_minutes, so each test sets it.

// claudeRetryEnv30 is driver.ClaudeRetryEnv(30), for [org].api_wait_minutes
// 30, the default, written out.
var claudeRetryEnv30 = []string{"CLAUDE_CODE_RETRY_WATCHDOG=1", "CLAUDE_CODE_RETRY_WATCHDOG_MAX_WAIT_MS=1800000", "CLAUDE_CODE_MAX_RETRIES=55"}

// seatPaneEnvWant is the extraEnv a seat of driverName gets: its
// RALPH_PRIMARY_CLI, then retry.
func seatPaneEnvWant(driverName string, retry []string) []string {
	return append([]string{"RALPH_PRIMARY_CLI=" + driverName}, retry...)
}

// TestOrgSpawn_ClaudeRetryEnvFollowsEachSeatsDriver pins AC-8 at the calls
// that make a seat's pane: the org's first seat, a leader, in the root pane
// (WorkspaceCreate), and the next seat in a split (PaneSplit). With
// [org].api_wait_minutes 30 a claude seat's pane gets RALPH_PRIMARY_CLI and
// then the three claude retry entries, and a codex seat's RALPH_PRIMARY_CLI
// alone, decided per seat, so an org that mixes the drivers gives each pane
// its own; with 0 no pane gets them.
func TestOrgSpawn_ClaudeRetryEnvFollowsEachSeatsDriver(t *testing.T) {
	for _, tc := range []struct {
		name                     string
		minutes                  int
		leaderDriver, seatDriver string
		wantRoot, wantSplit      []string
	}{
		{"claude leader and claude seat", 30, "claude", "claude",
			seatPaneEnvWant("claude", claudeRetryEnv30), seatPaneEnvWant("claude", claudeRetryEnv30)},
		{"codex leader and codex seat", 30, "codex", "codex",
			seatPaneEnvWant("codex", nil), seatPaneEnvWant("codex", nil)},
		{"claude leader and codex seat", 30, "claude", "codex",
			seatPaneEnvWant("claude", claudeRetryEnv30), seatPaneEnvWant("codex", nil)},
		{"codex leader and claude seat", 30, "codex", "claude",
			seatPaneEnvWant("codex", nil), seatPaneEnvWant("claude", claudeRetryEnv30)},
		{"claude seats with the setting 0", 0, "claude", "claude",
			seatPaneEnvWant("claude", nil), seatPaneEnvWant("claude", nil)},
		{"claude seats with the setting 1", 1, "claude", "claude",
			seatPaneEnvWant("claude", []string{"CLAUDE_CODE_RETRY_WATCHDOG=1", "CLAUDE_CODE_RETRY_WATCHDOG_MAX_WAIT_MS=60000", "CLAUDE_CODE_MAX_RETRIES=6"}),
			seatPaneEnvWant("claude", []string{"CLAUDE_CODE_RETRY_WATCHDOG=1", "CLAUDE_CODE_RETRY_WATCHDOG_MAX_WAIT_MS=60000", "CLAUDE_CODE_MAX_RETRIES=6"})},
	} {
		t.Run(tc.name, func(t *testing.T) {
			o, h, _ := testOrg(t)
			o.Config.Permissions = config.OrgPermissionsConfig{Default: PermissionModeAutonomous, CodexVerified: true}
			o.Config.APIWaitMinutes = tc.minutes
			models := map[string]string{"claude": "sonnet", "codex": "gpt-5-codex"}

			leader := mustSpawnParams("org-a", LeaderIdentity)
			leader.Role, leader.Driver, leader.Model = LeaderIdentity, tc.leaderDriver, models[tc.leaderDriver]
			if r := o.Spawn(leader); r.Outcome != SpawnOutcomeSpawned {
				t.Fatalf("spawn of the leader failed: %+v", r)
			}
			if len(h.workspaceCreateEnvs) != 1 || !slices.Equal(h.workspaceCreateEnvs[0], tc.wantRoot) {
				t.Fatalf("WorkspaceCreate extraEnv per call = %q; want one call with %q", h.workspaceCreateEnvs, tc.wantRoot)
			}

			seat := mustSpawnParams("org-a", "seat-2")
			seat.Driver, seat.Model = tc.seatDriver, models[tc.seatDriver]
			if r := o.Spawn(seat); r.Outcome != SpawnOutcomeSpawned {
				t.Fatalf("spawn of seat-2 failed: %+v", r)
			}
			if len(h.paneSplitCalls) != 1 || !slices.Equal(h.paneSplitCalls[0].extraEnv, tc.wantSplit) || len(h.tabCreateEnvs) != 0 {
				t.Fatalf("PaneSplit calls = %+v, TabCreate extraEnv = %q; want one split with %q and no tab", h.paneSplitCalls, h.tabCreateEnvs, tc.wantSplit)
			}
		})
	}
}

// TestOrgSpawn_Layout_TabOfAClaudeSeatGetsTheClaudeRetryEnv covers the third
// call that makes a seat's pane, TabCreate, for a workspace that holds no
// live seat's pane (TestOrgSpawn_Layout_NoLiveSeatPaneMakesATab): with
// [org].api_wait_minutes 30 a claude seat's tab gets the claude retry
// entries after its RALPH_PRIMARY_CLI, and a codex seat's does not.
func TestOrgSpawn_Layout_TabOfAClaudeSeatGetsTheClaudeRetryEnv(t *testing.T) {
	for _, tc := range []struct {
		driver, model string
		want          []string
	}{
		{"claude", "sonnet", seatPaneEnvWant("claude", claudeRetryEnv30)},
		{"codex", "gpt-5-codex", seatPaneEnvWant("codex", nil)},
	} {
		t.Run(tc.driver, func(t *testing.T) {
			o, h, _ := layoutOrg(t)
			o.Config.Permissions.CodexVerified = true
			o.Config.APIWaitMinutes = 30
			seedOpenOrgWorkspace(t, o, h, "org-a", "ws-1")
			seedSeat(t, o, "org-a", "seat-gone", "worker", "p-gone")
			h.paneID = "p-tab"
			p := mustSpawnParams("org-a", "seat-1")
			p.Driver, p.Model = tc.driver, tc.model
			mustSpawnLayout(t, o, p, "p-tab")

			if !slices.EqualFunc(h.tabCreateEnvs, [][]string{tc.want}, slices.Equal) || len(h.paneSplitCalls) != 0 || len(h.workspaceCreateEnvs) != 0 {
				t.Fatalf("TabCreate extraEnv %q, splits %+v, workspace creates %q; want one tab with %q and nothing else",
					h.tabCreateEnvs, h.paneSplitCalls, h.workspaceCreateEnvs, tc.want)
			}
		})
	}
}

// TestSeatPaneExtraEnv_ClaudeRetryEnvIsANewSliceEachCall: placeSeatPane's
// extraEnv is its own, so a change to one call's slice reaches no later
// call; and claudeRetryPaneEnv gives a codex driver nil.
func TestSeatPaneExtraEnv_ClaudeRetryEnvIsANewSliceEachCall(t *testing.T) {
	cfg := config.OrgConfig{APIWaitMinutes: 30}
	got := seatPaneExtraEnv(cfg, "claude")
	for i := range got {
		got[i] = "CHANGED=1"
	}
	if again, want := seatPaneExtraEnv(cfg, "claude"), seatPaneEnvWant("claude", claudeRetryEnv30); !slices.Equal(again, want) {
		t.Fatalf("seatPaneExtraEnv after a caller changed its slice = %q; want %q", again, want)
	}
	if got := claudeRetryPaneEnv(cfg, "codex"); got != nil {
		t.Fatalf("claudeRetryPaneEnv for codex = %q; want nil", got)
	}
}
