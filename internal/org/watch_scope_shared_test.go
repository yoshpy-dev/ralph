package org

import (
	"context"
	"slices"
	"strings"
	"testing"
)

// The scope-change condition of a feature org whose split plan has shared
// paths (plan 2026-11-22-split-shared-paths, AC4): a
// change to a shared path is not outside the reservation. The fixture is
// reservedScopeFixture (watch_scope_test.go), with org-a's leader bound to a
// split plan feature.

// sharedScopeBinding is testFeature's binding with the shared paths shared,
// in FeatureBinding.Shared's form (normalized, joined with commas; "" none).
func sharedScopeBinding(shared string) *FeatureBinding {
	f := testFeature("split-1", "auth", "0123456789ab")
	f.Shared = shared
	return f
}

// newSharedScopeFixture is newReservedScopeFixture for org-a's leader bound
// to a split plan feature that reserves reserve and whose plan shares
// shared, with the org's inbox wired, so a scope_change ALERT about the
// leader is an inbox item at once. It fails unless the org reads back the
// shared paths it was given.
func newSharedScopeFixture(t *testing.T, repo, shared string, reserve ...string) (*reservedScopeFixture, *InboxStore) {
	t.Helper()
	f := newBoundScopeFixture(t, repo, sharedScopeBinding(shared), reserve...)
	inbox := NewInboxStore(f.dir)
	inbox.Now = f.clk.Now
	f.o.Inbox = inbox
	f.o.EscalationsPath = EscalationsPathIn(f.dir)
	f.o.DesktopNotify = func(context.Context, string) error { return nil }
	var want []string
	if shared != "" {
		want = strings.Split(shared, ",")
	}
	if got := ActiveFeature(mustReadEvents(t, f.o), "org-a").SharedPaths(); !slices.Equal(got, want) {
		t.Fatalf("org-a's shared paths = %q; want %q", got, want)
	}
	return f, inbox
}

// inboxItems returns the number of items in inbox.
func inboxItems(t *testing.T, inbox *InboxStore) int {
	t.Helper()
	rr, err := inbox.Read()
	if err != nil {
		t.Fatalf("read the inbox: %v", err)
	}
	return len(rr.Items)
}

// AC4: in a feature org whose split plan shares a file and a directory,
// changes to them raise no scope_change and put nothing in the inbox: a new
// shared file, a new file deep under the shared directory, their commit, a
// rename under the shared directory, the shared file edited and committed in
// one cycle, and a change to the reservation and the shared file in one
// cycle. A change outside both afterwards raises one, which is an inbox item
// at once, so the inbox the test reads is the one the watch writes.
func TestWatch_SharedScope_ChangesToTheSharedPathsRaiseNothing(t *testing.T) {
	repo := scopeGitRepo(t, "main")
	f, inbox := newSharedScopeFixture(t, repo, "packages/shared/,pnpm-lock.yaml", "src/")
	f.cycle(t) // the baseline
	steps := []struct {
		name string
		do   func()
	}{
		{"a new shared file", func() { scopeWrite(t, repo, "pnpm-lock.yaml", "v1") }},
		{"a new file deep under the shared directory", func() { scopeWrite(t, repo, "packages/shared/deep/a.ts", "new") }},
		{"their commit", func() { scopeCommitAll(t, repo, "shared") }},
		{"a rename under the shared directory", func() { scopeGit(t, repo, "mv", "packages/shared/deep/a.ts", "packages/shared/b.ts") }},
		{"the shared file edited and committed in one cycle", func() {
			scopeWrite(t, repo, "pnpm-lock.yaml", "v2")
			scopeCommitAll(t, repo, "lock")
		}},
		{"the reservation and the shared file in one cycle", func() {
			scopeWrite(t, repo, "src/keep.go", "edited")
			scopeWrite(t, repo, "pnpm-lock.yaml", "v3")
		}},
	}
	for _, s := range steps {
		s.do()
		f.cycle(t)
		if alerts := f.scopeAlerts(); len(alerts) != 0 {
			t.Fatalf("after %s: scope_change ALERTs = %q; want none", s.name, alerts)
		}
		if n := inboxItems(t, inbox); n != 0 {
			t.Fatalf("after %s: the inbox has %d items; want none", s.name, n)
		}
	}
	if snap := f.status.SeatSnapshots[LeaderIdentity]; snap == nil || !snap.ScopeOutsidePathsSeen || len(snap.ScopeOutsidePaths) != 0 {
		t.Fatalf("the leader's snapshot = %+v; want the baseline seen, with nothing outside", snap)
	}

	scopeWrite(t, repo, "other/x.go", "edited")
	f.cycle(t)
	f.wantOneScopeAlert(t, "other/x.go")
	if n := inboxItems(t, inbox); n != 1 {
		t.Fatalf("the inbox has %d items; want the one scope_change item", n)
	}
}

// AC4: a change outside both the reservation and the shared paths still
// raises scope_change, and the ALERT names only the paths outside both,
// with the words of an org that shares nothing. A shared directory covers
// only what is under it and a shared file only itself, and a move out of a
// shared path names the path it moved to.
func TestWatch_SharedScope_AnAlertNamesOnlyThePathsOutsideTheReservationAndTheSharedPaths(t *testing.T) {
	tests := []struct {
		name  string
		do    func(t *testing.T, repo string)
		clean bool // the worktree is clean afterwards: only the commit shows the change
		want  []string
	}{
		{"edits of a shared file, a shared directory, the reservation, and outside", func(t *testing.T, repo string) {
			scopeWrite(t, repo, "pnpm-lock.yaml", "edited")
			scopeWrite(t, repo, "packages/shared/a.ts", "edited")
			scopeWrite(t, repo, "src/keep.go", "edited")
			scopeWrite(t, repo, "other/x.go", "edited")
		}, false, []string{"other/x.go"}},
		{"the same committed", func(t *testing.T, repo string) {
			scopeWrite(t, repo, "pnpm-lock.yaml", "edited")
			scopeWrite(t, repo, "packages/shared/a.ts", "edited")
			scopeWrite(t, repo, "src/keep.go", "edited")
			scopeWrite(t, repo, "other/x.go", "edited")
			scopeCommitAll(t, repo, "shared, reserved and outside")
		}, true, []string{"other/x.go"}},
		{"a shared directory covers only what is under it", func(t *testing.T, repo string) {
			scopeWrite(t, repo, "packages/shared/a.ts", "new")
			scopeWrite(t, repo, "packages/sharedx/a.ts", "new")
			scopeWrite(t, repo, "packages/shared.ts", "new")
		}, false, []string{"packages/shared.ts", "packages/sharedx/a.ts"}},
		{"a shared file covers only itself", func(t *testing.T, repo string) {
			scopeWrite(t, repo, "pnpm-lock.yaml", "new")
			scopeWrite(t, repo, "pnpm-lock.yaml.bak", "new")
			scopeWrite(t, repo, "sub/pnpm-lock.yaml", "new")
		}, false, []string{"pnpm-lock.yaml.bak", "sub/pnpm-lock.yaml"}},
		{"a move out of the shared directory", func(t *testing.T, repo string) {
			scopeWrite(t, repo, "packages/shared/a.ts", "new")
			scopeCommitAll(t, repo, "shared")
			scopeGit(t, repo, "mv", "packages/shared/a.ts", "other/a.ts")
		}, false, []string{"other/a.ts"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := scopeGitRepo(t, "main")
			f, _ := newSharedScopeFixture(t, repo, "packages/shared/,pnpm-lock.yaml", "src/")
			f.cycle(t)
			tt.do(t, repo)
			if dirty := gitOut(t, repo, "status", "--porcelain"); tt.clean && dirty != "" {
				t.Fatalf("the worktree is not clean after the commit:\n%s", dirty)
			}
			f.cycle(t)
			f.wantOneScopeAlert(t, tt.want...)
		})
	}
}

// AC4's other half: only the shared paths of the org's own split plan count.
// A feature org whose plan has no shared paths, and an org with a plain
// reservation, raise scope_change for a change to pnpm-lock.yaml as before.
func TestWatch_SharedScope_AnOrgWithoutSharedPathsAlertsOnThemAsBefore(t *testing.T) {
	for _, tc := range []struct {
		name    string
		fixture func(t *testing.T, repo string) *reservedScopeFixture
	}{
		{"a feature org whose plan shares nothing", func(t *testing.T, repo string) *reservedScopeFixture {
			f, _ := newSharedScopeFixture(t, repo, "", "src/")
			if ActiveFeature(mustReadEvents(t, f.o), "org-a") == nil {
				t.Fatal("expected org-a bound to a split plan feature")
			}
			return f
		}},
		{"a plain reservation", func(t *testing.T, repo string) *reservedScopeFixture {
			return newReservedScopeFixture(t, repo, "src/")
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			repo := scopeGitRepo(t, "main")
			f := tc.fixture(t, repo)
			f.cycle(t)
			scopeWrite(t, repo, "pnpm-lock.yaml", "new")
			scopeWrite(t, repo, "src/keep.go", "edited")
			f.cycle(t)
			f.wantOneScopeAlert(t, "pnpm-lock.yaml")
		})
	}
}
