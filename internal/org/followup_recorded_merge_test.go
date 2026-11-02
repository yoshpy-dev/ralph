package org

import (
	"context"
	"testing"
)

// followupTestHead is the head a test's PR merged; followupTestOtherHead
// the head of a later PR of the same branch name, the branch used again.
const (
	followupTestHead      = "0123456789abcdef0123456789abcdef01234567"
	followupTestOtherHead = "fedcba9876543210fedcba9876543210fedcba98"
)

// prMergedEvent is a pr_merged of branch at head as ralph org merge writes
// it: recordMerge appends it to a manifest of its own, and the event read
// back is given ts, so its Details are recordMerge's own.
func prMergedEvent(t *testing.T, ts, branch, head string) ManifestEvent {
	t.Helper()
	o := &Org{Manifest: NewManifestStoreAtPath(ManifestPathIn(t.TempDir()))}
	res := MergeResult{Branch: branch, OrgID: "org-a", PRNumber: 12, Method: "merge", Head: head}
	if err := o.recordMerge(context.Background(), res, ""); err != nil {
		t.Fatal(err)
	}
	read, err := o.Manifest.Read()
	if err != nil || len(read.Events) != 1 || read.Events[0].Event != EventPRMerged {
		t.Fatalf("the pr_merged recordMerge wrote reads back as %+v, %v", read, err)
	}
	ev := read.Events[0]
	ev.TS = ts
	return ev
}

// worktreeCleanedEvent is a worktree_cleaned of branch at head as the
// cleanup writes it (cleanupDetails, via cleanup), with worktree as its
// quoted path.
func worktreeCleanedEvent(ts, branch, head, worktree string) ManifestEvent {
	return ManifestEvent{TS: ts, OrgID: "org-a", Event: EventWorktreeCleaned,
		Details: cleanupDetails(branch, head, CleanupViaCleanup, worktree)}
}

// D2, D5, AC4, and AC8: the ledger records the after-branch as merged only
// by a pr_merged or a worktree_cleaned that names the same branch exactly at
// or after the follow-up was added, and not by a worktree_cleaned that only
// cleans up a merge from before it (the same head as an earlier pr_merged).
func TestFollowupRecordedMerge(t *testing.T) {
	const added = "2026-10-30T12:00:00Z"
	const head, other = followupTestHead, followupTestOtherHead
	followup := Followup{ID: "f1", Issue: 270, AfterBranch: "feat/a", State: FollowupStateOpen, AddedAt: added}
	for _, tc := range []struct {
		name      string
		f         Followup
		events    []ManifestEvent
		wantAt    string
		wantEvent string // "" means not merged
	}{
		{name: "no events", f: followup},
		{
			name: "a pr_merged after the add", f: followup,
			events: []ManifestEvent{prMergedEvent(t, "2026-10-30T13:00:00Z", "feat/a", head)},
			wantAt: "2026-10-30T13:00:00Z", wantEvent: EventPRMerged,
		},
		{
			name: "a pr_merged in the second of the add", f: followup,
			events: []ManifestEvent{prMergedEvent(t, added, "feat/a", head)},
			wantAt: added, wantEvent: EventPRMerged,
		},
		{
			// Both times are whole seconds, so a merge recorded in the second
			// the follow-up was added counts even when it came a fraction of a
			// second before the add: the comparison is at or after.
			name: "a worktree_cleaned in the second of the add", f: followup,
			events: []ManifestEvent{worktreeCleanedEvent(added, "feat/a", head, "/repo/wt")},
			wantAt: added, wantEvent: EventWorktreeCleaned,
		},
		{
			name: "a pr_merged before the add (a branch name used again)", f: followup,
			events: []ManifestEvent{prMergedEvent(t, "2026-10-30T11:59:59Z", "feat/a", head)},
		},
		{
			name: "a worktree_cleaned after the add", f: followup,
			events: []ManifestEvent{worktreeCleanedEvent("2026-10-30T14:00:00Z", "feat/a", head, "/repo/.claude/worktrees/a")},
			wantAt: "2026-10-30T14:00:00Z", wantEvent: EventWorktreeCleaned,
		},
		{
			// ralph org merge merged and recorded the PR before the add, its
			// own cleanup was refused, and ralph org cleanup ran after the add.
			name: "a pr_merged before the add and the worktree_cleaned of its head after it", f: followup,
			events: []ManifestEvent{
				prMergedEvent(t, "2026-10-29T09:00:00Z", "feat/a", head),
				worktreeCleanedEvent("2026-10-30T12:30:00Z", "feat/a", head, "/repo/wt"),
			},
		},
		{
			name: "the earlier merge's pr_merged after its worktree_cleaned in file order", f: followup,
			events: []ManifestEvent{
				worktreeCleanedEvent("2026-10-30T12:30:00Z", "feat/a", head, "/repo/wt"),
				prMergedEvent(t, "2026-10-29T09:00:00Z", "feat/a", head),
			},
		},
		{
			name: "a pr_merged before the add and a worktree_cleaned of another head after it (the branch used again)", f: followup,
			events: []ManifestEvent{
				prMergedEvent(t, "2026-10-29T09:00:00Z", "feat/a", head),
				worktreeCleanedEvent("2026-10-30T12:30:00Z", "feat/a", other, "/repo/wt"),
			},
			wantAt: "2026-10-30T12:30:00Z", wantEvent: EventWorktreeCleaned,
		},
		{
			name: "the earlier merge's worktree_cleaned, then a pr_merged of another head after the add", f: followup,
			events: []ManifestEvent{
				prMergedEvent(t, "2026-10-29T09:00:00Z", "feat/a", head),
				worktreeCleanedEvent("2026-10-30T12:30:00Z", "feat/a", head, "/repo/wt"),
				prMergedEvent(t, "2026-10-30T15:00:00Z", "feat/a", other),
			},
			wantAt: "2026-10-30T15:00:00Z", wantEvent: EventPRMerged,
		},
		{
			name: "the first of two in file order", f: followup,
			events: []ManifestEvent{
				prMergedEvent(t, "2026-10-30T13:00:00Z", "feat/a", head),
				worktreeCleanedEvent("2026-10-30T13:00:05Z", "feat/a", head, "/repo/wt"),
			},
			wantAt: "2026-10-30T13:00:00Z", wantEvent: EventPRMerged,
		},
		{
			name: "another branch", f: followup,
			events: []ManifestEvent{
				prMergedEvent(t, "2026-10-30T13:00:00Z", "feat/b", head),
				worktreeCleanedEvent("2026-10-30T13:00:00Z", "feat/b", head, "/repo/wt"),
			},
		},
		{
			name: "a branch the after-branch is a prefix of", f: followup,
			events: []ManifestEvent{
				prMergedEvent(t, "2026-10-30T13:00:00Z", "feat/ab", head),
				worktreeCleanedEvent("2026-10-30T13:00:00Z", "feat/a/b", head, "/repo/wt"),
			},
		},
		{
			name: "a branch that is a prefix of the after-branch",
			f:    Followup{ID: "f1", Issue: 270, AfterBranch: "feat/ab", State: FollowupStateOpen, AddedAt: added},
			events: []ManifestEvent{
				prMergedEvent(t, "2026-10-30T13:00:00Z", "feat/a", head),
				worktreeCleanedEvent("2026-10-30T13:00:00Z", "feat/a", head, "/repo/wt"),
			},
		},
		{
			name: "another branch's worktree_cleaned whose quoted worktree holds the branch", f: followup,
			events: []ManifestEvent{worktreeCleanedEvent("2026-10-30T13:00:00Z", "feat/b", head, "/repo/x branch=feat/a")},
		},
		{
			// The quoted worktree names the earlier merge's head, but the
			// event's own head is another one.
			name: "a worktree_cleaned of another head whose quoted worktree holds the earlier head", f: followup,
			events: []ManifestEvent{
				prMergedEvent(t, "2026-10-29T09:00:00Z", "feat/a", head),
				worktreeCleanedEvent("2026-10-30T12:30:00Z", "feat/a", other, "/repo/x head="+head),
			},
			wantAt: "2026-10-30T12:30:00Z", wantEvent: EventWorktreeCleaned,
		},
		{
			name: "a gate record and a scope reservation that name the branch", f: followup,
			events: []ManifestEvent{
				{TS: "2026-10-30T13:00:00Z", OrgID: "org-a", Event: EventGatePassed, Details: "branch=feat/a"},
				{TS: "2026-10-30T13:00:00Z", OrgID: "org-a", Event: EventScopeReserved, Details: "branch=feat/a"},
			},
		},
		{
			name: "times compare as instants, not as text", f: followup,
			events: []ManifestEvent{prMergedEvent(t, "2026-10-30T20:30:00+09:00", "feat/a", head)},
		},
		{
			name: "an event ts that does not parse", f: followup,
			events: []ManifestEvent{prMergedEvent(t, "yesterday", "feat/a", head)},
		},
		{
			name:   "an added_at that does not parse",
			f:      Followup{ID: "f1", Issue: 270, AfterBranch: "feat/a", State: FollowupStateOpen, AddedAt: "t"},
			events: []ManifestEvent{prMergedEvent(t, "2026-10-30T13:00:00Z", "feat/a", head)},
		},
		{
			name:   "a Details with an empty branch, for a follow-up with none",
			f:      Followup{ID: "f1", Issue: 270, State: FollowupStateOpen, AddedAt: added},
			events: []ManifestEvent{{TS: "2026-10-30T13:00:00Z", OrgID: "org-a", Event: EventPRMerged, Details: "branch= pr=12"}},
		},
		{
			name: "a Details whose first token is not branch=", f: followup,
			events: []ManifestEvent{{TS: "2026-10-30T13:00:00Z", OrgID: "org-a", Event: EventPRMerged, Details: "pr=12 branch=feat/a"}},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			merged, at, event := FollowupRecordedMerge(tc.events, tc.f)
			if merged != (tc.wantEvent != "") || at != tc.wantAt || event != tc.wantEvent {
				t.Fatalf("FollowupRecordedMerge = %v, %q, %q; want %v, %q, %q", merged, at, event, tc.wantEvent != "", tc.wantAt, tc.wantEvent)
			}
		})
	}
}
