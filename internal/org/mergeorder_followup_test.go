package org

import (
	"context"
	"encoding/json"
	"errors"
	"maps"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"
)

// The follow-up section of `ralph org merge-order` at the org layer (issue
// #270; plan docs/plans/active/2026-10-30-org-followups.md, D5, AC3 and AC8):
// MergeOrder with MergeOrderParams.Followups on a follow-up record and the
// fake MergeOrderExec of mergeorder_test.go (moExec), whose gh answers per
// --head branch whatever fields it is asked for.

// moFollowupAddedAt is when the fixtures' follow-ups are recorded, an hour
// before mergeOrderTestClock.
var moFollowupAddedAt = mergeOrderTestClock.Add(-time.Hour)

// moFollowupGHArgs is a follow-up's gh pr list for branch, as merge-order
// runs it without -R.
func moFollowupGHArgs(branch string) []string {
	return []string{"pr", "list", "--head", branch, "--state", "all", "--limit", "1000", "--json",
		"number,state,url,headRefOid,baseRefName,isCrossRepository,mergedAt"}
}

// moFollowupPR is one PR of a follow-up's gh pr list output: moPR's fields,
// with mergedAt in the embedded ghPullRequest's MergedAt. It declares no
// field of its own: one with the JSON name mergedAt would hide the embedded
// one, and a time set there would be left out of the output (self-review
// R2-4).
type moFollowupPR struct {
	ghPullRequest
}

// moFPR is moPR(n, state, "main") with no merge time.
func moFPR(n int, state string) moFollowupPR {
	return moFollowupPR{ghPullRequest: moPR(n, state, "main")}
}

// moAt is moFollowupAddedAt moved by d, as RFC3339.
func moAt(d time.Duration) string {
	return moFollowupAddedAt.Add(d).Format(time.RFC3339)
}

// moMergedPR is PR n merged d after moFollowupAddedAt, before it when d is
// negative.
func moMergedPR(n int, d time.Duration) moFollowupPR {
	pr := moPR(n, "MERGED", "main")
	pr.MergedAt = moAt(d)
	return moFollowupPR{ghPullRequest: pr}
}

// moFollowupPRs is `gh pr list --json` output holding prs.
func moFollowupPRs(prs ...moFollowupPR) string {
	b, err := json.Marshal(prs)
	if err != nil {
		panic(err)
	}
	return string(b)
}

// addFollowup records a follow-up of issue after branch in fx's state dir at
// moFollowupAddedAt and returns its ID.
func (fx *moFixture) addFollowup(t *testing.T, issue int, branch string) string {
	t.Helper()
	s := NewFollowupStore(fx.stateDir)
	s.Now = func() time.Time { return moFollowupAddedAt }
	id, err := s.Add(FollowupAdd{Issue: issue, AfterBranch: branch})
	if err != nil {
		t.Fatal(err)
	}
	return id
}

// appendFollowupRaw appends raw to fx's follow-up record as it is.
func (fx *moFixture) appendFollowupRaw(t *testing.T, raw string) {
	t.Helper()
	f, err := os.OpenFile(FollowupsPathIn(fx.stateDir), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.WriteString(raw); err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
}

// followupParams is fx.params(false) with Followups.
func (fx *moFixture) followupParams() MergeOrderParams {
	p := fx.params(false)
	p.Followups = true
	return p
}

// runFollowups is MergeOrder with fx.followupParams(), failing the test on an
// error.
func (fx *moFixture) runFollowups(t *testing.T) MergeOrderResult {
	t.Helper()
	res, err := fx.o.MergeOrder(context.Background(), fx.followupParams())
	if err != nil {
		t.Fatalf("MergeOrder: %v", err)
	}
	return res
}

// moFollowupGHCalls are the follow-up lookups among e's gh calls: those that
// ask for mergedAt.
func moFollowupGHCalls(e *moExec) []moCall {
	var out []moCall
	for _, c := range e.callsOf("gh") {
		if strings.HasSuffix(c.args[9], ",mergedAt") {
			out = append(out, c)
		}
	}
	return out
}

// moFollowupStates is each follow-up of res as its ID and state.
func moFollowupStates(res MergeOrderResult) map[string]string {
	out := make(map[string]string)
	for _, f := range res.Followups.Items {
		out[f.ID] = f.State
	}
	return out
}

// checkFollowupDetail fails the test unless f's detail holds every phrase of
// want.
func checkFollowupDetail(t *testing.T, f MergeOrderFollowup, want ...string) {
	t.Helper()
	for _, w := range want {
		if !strings.Contains(f.Detail, w) {
			t.Errorf("%s detail = %q; want it to contain %q", f.ID, f.Detail, w)
		}
	}
}

// TestMergeOrderFollowup_StateComesFromTheAfterBranchsPRs covers D5, AC3 and
// AC8: a PR of the after-branch merged at or after the record makes the
// follow-up ready, whichever PR is the newest; a PR merged before it does
// not, alone or with a newer open PR, and the detail says it was merged
// before the follow-up was recorded; the newest PR closed without a merge is
// closed with the reason; no PR is waiting; a fork's PR is never read; a
// merged PR with no merge time, a state gh has no other of, and gh's limit of
// PRs with none merged after are unknown. Every follow-up has source gh and
// MergeOrder returns no error.
func TestMergeOrderFollowup_StateComesFromTheAfterBranchsPRs(t *testing.T) {
	added := moAt(0)
	forkMerged := moMergedPR(9, time.Minute)
	forkMerged.IsCrossRepository = true
	limit := make([]moFollowupPR, 0, mergeOrderGHLimit)
	for i := range mergeOrderGHLimit {
		limit = append(limit, moFPR(mergeOrderGHLimit-i, "OPEN"))
	}
	limit[len(limit)-1] = moMergedPR(1, -time.Minute)
	for _, tc := range []struct {
		name     string
		prs      []moFollowupPR
		state    string
		pr       int // the PR reported, 0 for none
		mergedAt string
		detail   []string
	}{
		{"merged after the record is ready", []moFollowupPR{moMergedPR(7, 30*time.Minute)}, FollowupMergeReady, 7, moAt(30 * time.Minute),
			[]string{"PR #7 was merged at " + moAt(30*time.Minute) + ", after the follow-up was recorded at " + added,
				"ask the human about issue #270 now, then run ralph org followup done f1"}},
		{"merged at the time of the record is ready", []moFollowupPR{moMergedPR(7, 0)}, FollowupMergeReady, 7, added,
			[]string{"after the follow-up was recorded"}},
		{"merged after the record under a newer open PR is ready", []moFollowupPR{moFPR(8, "OPEN"), moMergedPR(7, time.Minute)},
			FollowupMergeReady, 7, moAt(time.Minute), []string{"PR #7 was merged at"}},
		{"merged before the record is waiting", []moFollowupPR{moMergedPR(7, -time.Hour)}, FollowupMergeWaiting, 7, moAt(-time.Hour),
			[]string{"PR #7 was merged before the follow-up was recorded (merged at " + moAt(-time.Hour) + ", recorded at " + added + ")",
				"waits for a new PR of feat/a to be merged", "if none is coming, ask the human about issue #270 now and run ralph org followup done f1"}},
		{"an older merged PR and a newer open one is waiting (AC8)", []moFollowupPR{moFPR(9, "OPEN"), moMergedPR(7, -time.Hour)},
			FollowupMergeWaiting, 9, "", []string{"PR #9 is open: the follow-up waits for its merge",
				"PR #7 was merged before the follow-up was recorded (at " + moAt(-time.Hour) + ") and does not count"}},
		{"closed without a merge is closed", []moFollowupPR{moFPR(7, "CLOSED")}, FollowupMergeClosed, 7, "",
			[]string{"PR #7 was closed without a merge, so the merge the follow-up waits for may not come",
				"may not come: reopen PR #7 or open a new PR of feat/a, or ask the human about issue #270 and run ralph org followup done f1"}},
		// The clause about the older merge comes between the closed PR and
		// "reopen", so the PR to reopen is named by its number (self-review
		// L5).
		{"closed after an older merged one is closed", []moFollowupPR{moMergedPR(6, -time.Hour), moFPR(7, "CLOSED")}, FollowupMergeClosed, 7, "",
			[]string{"PR #7 was closed without a merge", "PR #6 was merged before the follow-up was recorded (at " + moAt(-time.Hour) +
				") and does not count: reopen PR #7 or open a new PR of feat/a"}},
		{"no PR is waiting", nil, FollowupMergeWaiting, 0, "", []string{"feat/a has no PR yet: the follow-up waits for one to be opened and merged"}},
		{"a fork's PR merged after is not read", []moFollowupPR{forkMerged}, FollowupMergeWaiting, 0, "", []string{"feat/a has no PR yet"}},
		{"a fork's PR does not hide the branch's open PR", []moFollowupPR{forkMerged, moFPR(3, "OPEN")}, FollowupMergeWaiting, 3, "",
			[]string{"PR #3 is open"}},
		{"a merged PR with no merge time is unknown", []moFollowupPR{moFPR(7, "MERGED")}, FollowupMergeUnknown, 7, "",
			[]string{`PR #7 is merged, but gh gave its merge time as "", which cannot be compared with the follow-up's added_at ` + added}},
		{"a state gh has no other of is unknown", []moFollowupPR{moFPR(7, "DRAFT")}, FollowupMergeUnknown, 7, "",
			[]string{`gh gave PR #7 the state "DRAFT", which is none of OPEN, MERGED and CLOSED`}},
		{"gh's limit of PRs with none merged after is unknown", limit, FollowupMergeUnknown, 0, "",
			[]string{"gh returned 1000 PRs for feat/a, its limit, and none of them was merged after the follow-up was recorded"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fx := newMOFixture(t)
			if id := fx.addFollowup(t, 270, "feat/a"); id != "f1" {
				t.Fatalf("id = %q, want f1", id)
			}
			fx.exec.setPR("feat/a", moFollowupPRs(tc.prs...))
			res := fx.runFollowups(t)
			if len(res.Followups.Items) != 1 {
				t.Fatalf("followups = %+v, want one", res.Followups)
			}
			f := res.Followups.Items[0]
			if f.ID != "f1" || f.Issue != 270 || f.AfterBranch != "feat/a" || f.AddedAt != added || f.Source != FollowupMergeSourceGH {
				t.Errorf("follow-up = %+v, want f1 for issue 270 after feat/a, added at %s, from gh", f, added)
			}
			if f.State != tc.state || f.MergedAt != tc.mergedAt {
				t.Errorf("state %q merged_at %q (detail %q), want %q and %q", f.State, f.MergedAt, f.Detail, tc.state, tc.mergedAt)
			}
			switch {
			case tc.pr == 0 && f.PR != nil:
				t.Errorf("PR = %+v, want none", f.PR)
			case tc.pr != 0 && (f.PR == nil || *f.PR != MergeOrderFollowupPR{Number: tc.pr, URL: "https://github.com/o/r/pull/" + strconv.Itoa(tc.pr)}):
				t.Errorf("PR = %+v, want #%d with its url", f.PR, tc.pr)
			}
			checkFollowupDetail(t, f, tc.detail...)
		})
	}
}

// TestMergeOrderFollowup_OneLookupPerAfterBranch covers AC3's lookup: two
// follow-ups on one branch share one gh pr list, in the order the branches
// first appear, of the shape lookupMergePR runs (--state all, --limit 1000,
// GH_PROMPT_DISABLED=1, p.Dir) with mergedAt added to the fields. Both get the
// branch's state.
func TestMergeOrderFollowup_OneLookupPerAfterBranch(t *testing.T) {
	fx := newMOFixture(t)
	fx.addFollowup(t, 1, "feat/a")
	fx.addFollowup(t, 2, "feat/b")
	fx.addFollowup(t, 3, "feat/a")
	fx.exec.setPR("feat/a", moFollowupPRs(moMergedPR(7, time.Minute)))
	res := fx.runFollowups(t)
	want := map[string]string{"f1": FollowupMergeReady, "f2": FollowupMergeWaiting, "f3": FollowupMergeReady}
	if got := moFollowupStates(res); !maps.Equal(got, want) {
		t.Fatalf("states = %v, want %v", got, want)
	}
	if ids := []string{res.Followups.Items[0].ID, res.Followups.Items[1].ID, res.Followups.Items[2].ID}; !slices.Equal(ids, []string{"f1", "f2", "f3"}) {
		t.Errorf("follow-ups in the order %q, want ID order", ids)
	}
	gh := fx.exec.callsOf("gh")
	if len(gh) != 2 {
		t.Fatalf("gh ran %d times (%+v), want once per after-branch", len(gh), gh)
	}
	for i, branch := range []string{"feat/a", "feat/b"} {
		if c := gh[i]; !slices.Equal(c.args, moFollowupGHArgs(branch)) || c.dir != "/repo" || !slices.Equal(c.env, []string{"GH_PROMPT_DISABLED=1"}) {
			t.Errorf("gh call %d = %+v, want %q in /repo with GH_PROMPT_DISABLED=1", i, c, moFollowupGHArgs(branch))
		}
	}
}

// TestMergeOrderFollowup_EntriesAndTheirLookupsStayAsTheyWere covers AC3's
// "the entries do not change": with follow-ups, the entries, the notes, gh's
// availability and the ranked lookups are those of the run without them, and
// the follow-up lookups come after the ranked ones. A ranked lookup's PR list
// that holds mergedAt still parses (feat/a's merged PR hides it).
func TestMergeOrderFollowup_EntriesAndTheirLookupsStayAsTheyWere(t *testing.T) {
	fx := moLookupFixture(t, moOriginURL, "a", "b")
	fx.exec.setPR("feat/a", moFollowupPRs(moMergedPR(7, time.Minute)))
	fx.addFollowup(t, 270, "feat/a")
	fx.addFollowup(t, 271, "feat/z")
	without := fx.run(t, false)
	rankedCalls := slices.Clone(fx.exec.calls)
	fx.exec.calls = nil
	with := fx.runFollowups(t)
	if !reflect.DeepEqual(with.Entries, without.Entries) || !slices.Equal(with.Notes, without.Notes) || with.GHAvailable != without.GHAvailable {
		t.Fatalf("with follow-ups: entries %+v notes %q gh %v\nwithout: entries %+v notes %q gh %v", with.Entries, with.Notes,
			with.GHAvailable, without.Entries, without.Notes, without.GHAvailable)
	}
	if got := moOrder(with); !slices.Equal(got, []string{"1 ready feat/b"}) {
		t.Errorf("order = %q, want feat/b alone (feat/a merged)", got)
	}
	if len(without.Followups.Items) != 0 || without.Followups.Items == nil {
		t.Errorf("without Followups: followups = %#v, want an empty list", without.Followups)
	}
	if !reflect.DeepEqual(fx.exec.calls[:len(rankedCalls)], rankedCalls) {
		t.Errorf("calls = %+v\nwant them to start with the ranked run's %+v", fx.exec.calls, rankedCalls)
	}
	var followupGH []string
	for _, c := range fx.exec.calls[len(rankedCalls):] {
		if c.name == "gh" {
			followupGH = append(followupGH, strings.Join(c.args, " "))
		}
	}
	if want := []string{strings.Join(moFollowupGHArgs("feat/a"), " "), strings.Join(moFollowupGHArgs("feat/z"), " ")}; !slices.Equal(followupGH, want) {
		t.Errorf("follow-up gh calls = %q, want %q", followupGH, want)
	}
	if got, want := moFollowupStates(with), map[string]string{"f1": FollowupMergeReady, "f2": FollowupMergeWaiting}; !maps.Equal(got, want) {
		t.Errorf("states = %v, want %v", got, want)
	}
}

// TestMergeOrderFollowup_GHMissingMakesEveryFollowupUnknown covers AC3's
// unknown for a gh that is not on PATH: every follow-up is unknown with the
// reason, gh is not run for the second branch, and MergeOrder returns no
// error.
func TestMergeOrderFollowup_GHMissingMakesEveryFollowupUnknown(t *testing.T) {
	fx := newMOFixture(t)
	fx.addFollowup(t, 1, "feat/a")
	fx.addFollowup(t, 2, "feat/b")
	fx.exec.ghMissing = true
	res := fx.runFollowups(t)
	for _, f := range res.Followups.Items {
		if f.State != FollowupMergeUnknown || f.PR != nil || f.MergedAt != "" {
			t.Errorf("%s = %+v, want unknown with no PR", f.ID, f)
		}
		checkFollowupDetail(t, f, "gh was not found, so no PR was read", "ralph status tells it from the ledger's records of merges, without gh")
	}
	if len(res.Followups.Items) != 2 || len(fx.exec.callsOf("gh")) != 1 {
		t.Errorf("followups %+v, gh calls %d; want two follow-ups and one gh call", res.Followups, len(fx.exec.callsOf("gh")))
	}
}

// TestMergeOrderFollowup_GHFailureMakesOnlyItsBranchUnknown covers AC3's
// unknown for a lookup that fails: gh exiting non-zero, timing out, or
// printing something that is not a PR list makes only that branch's
// follow-up unknown with the reason; the other branch is still read.
func TestMergeOrderFollowup_GHFailureMakesOnlyItsBranchUnknown(t *testing.T) {
	for _, tc := range []struct {
		name   string
		setup  func(fx *moFixture, p *MergeOrderParams)
		detail string
	}{
		{"an exit", func(fx *moFixture, _ *MergeOrderParams) {
			fx.exec.ghErr = map[string]error{"feat/a": errors.New("exit status 1: HTTP 502")}
		}, "gh pr list --head feat/a failed (exit status 1: HTTP 502): its PR is unknown"},
		{"a timeout", func(fx *moFixture, p *MergeOrderParams) {
			fx.exec.ghBlock = map[string]bool{"feat/a": true}
			p.GHTimeout = 20 * time.Millisecond
		}, "gh pr list --head feat/a failed (timed out after 20ms: context deadline exceeded): its PR is unknown"},
		{"output that is not a PR list", func(fx *moFixture, _ *MergeOrderParams) {
			fx.exec.setPR("feat/a", "not json")
		}, "gh pr list --head feat/a failed (its output is not a JSON list of PRs: "},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fx := newMOFixture(t)
			fx.addFollowup(t, 1, "feat/a")
			fx.addFollowup(t, 2, "feat/b")
			fx.exec.setPR("feat/b", moFollowupPRs(moMergedPR(8, time.Minute)))
			p := fx.followupParams()
			tc.setup(fx, &p)
			res, err := fx.o.MergeOrder(context.Background(), p)
			if err != nil {
				t.Fatalf("MergeOrder: %v", err)
			}
			if got, want := moFollowupStates(res), map[string]string{"f1": FollowupMergeUnknown, "f2": FollowupMergeReady}; !maps.Equal(got, want) {
				t.Fatalf("states = %v, want %v", got, want)
			}
			checkFollowupDetail(t, res.Followups.Items[0], tc.detail)
			if len(moFollowupGHCalls(fx.exec)) != 2 {
				t.Errorf("gh calls = %+v, want one per branch", fx.exec.callsOf("gh"))
			}
		})
	}
}

// TestMergeOrderFollowup_PRFromAnotherRepositoryMakesEveryFollowupUnknown
// covers AC3's unknown for the origin check: without a pin, a PR whose url
// names another repository than origin's makes every follow-up unknown with
// lookupMergePRs's note, one read before it included, and gh is not run for
// the branches after it; with p.Origin, a PR of another repository does the
// same with foreignPRWhy's reason, -R is passed, and no get-url runs.
func TestMergeOrderFollowup_PRFromAnotherRepositoryMakesEveryFollowupUnknown(t *testing.T) {
	merged := func(owner string) string {
		pr := moRepoPR(7, "MERGED", "main", owner, "app")
		pr.MergedAt = moAt(time.Minute)
		return moFollowupPRs(moFollowupPR{ghPullRequest: pr})
	}
	t.Run("origin's URL", func(t *testing.T) {
		fx := newMOFixture(t)
		fx.exec.origin = "git@github.com:acme/app.git"
		fx.addFollowup(t, 1, "feat/a")
		fx.addFollowup(t, 2, "feat/b")
		fx.addFollowup(t, 3, "feat/c")
		fx.exec.setPR("feat/a", merged("acme"))
		fx.exec.setPR("feat/b", merged("upstream-org"))
		res := fx.runFollowups(t)
		for _, f := range res.Followups.Items {
			if f.State != FollowupMergeUnknown || f.PR != nil {
				t.Errorf("%s = %+v, want unknown with no PR", f.ID, f)
			}
			checkFollowupDetail(t, f, "gh reads upstream-org/app, not origin's acme/app: no PR is used", "run gh repo set-default acme/app")
		}
		if got := moGHBranches(fx); !slices.Equal(got, []string{"feat/a", "feat/b"}) {
			t.Errorf("gh asked about %q, want feat/a and feat/b only", got)
		}
		if git := moGitArgs(fx.exec); !slices.Equal(git, []string{"remote get-url origin"}) {
			t.Errorf("git calls = %q, want one get-url", git)
		}
	})
	t.Run("a pinned origin", func(t *testing.T) {
		fx := newMOFixture(t)
		fx.addFollowup(t, 1, "feat/a")
		fx.addFollowup(t, 2, "feat/b")
		fx.exec.setPR("feat/a", moFollowupPRs(moMergedPR(7, time.Minute)))
		fx.exec.setPR("feat/b", merged("x"))
		p := fx.followupParams()
		p.Origin = moPinnedOrigin
		res, err := fx.o.MergeOrder(context.Background(), p)
		if err != nil {
			t.Fatalf("MergeOrder: %v", err)
		}
		why := "gh pr list --head feat/b -R github.com/o/r returned PR #7 with the url " + strconv.Quote("https://github.com/x/app/pull/7") +
			", which is not a PR of origin's repository github.com/o/r; check origin (git remote get-url origin)"
		for _, f := range res.Followups.Items {
			if f.State != FollowupMergeUnknown || f.Detail != why {
				t.Errorf("%s = %+v, want unknown with %q", f.ID, f, why)
			}
		}
		for _, c := range fx.exec.callsOf("gh") {
			if want := append(moFollowupGHArgs(c.args[3]), "-R", "github.com/o/r"); !slices.Equal(c.args, want) {
				t.Errorf("gh %q, want %q", c.args, want)
			}
		}
		if git := fx.exec.callsOf("git"); len(git) != 0 {
			t.Errorf("git calls = %+v, want none", git)
		}
	})
}

// TestMergeOrderFollowup_PinnedOriginReadsItsOwnPRs: with p.Origin, a PR
// whose url names the pinned repository is read, so a merge after the record
// is ready.
func TestMergeOrderFollowup_PinnedOriginReadsItsOwnPRs(t *testing.T) {
	fx := newMOFixture(t)
	fx.addFollowup(t, 1, "feat/a")
	fx.exec.setPR("feat/a", moFollowupPRs(moMergedPR(7, time.Minute)))
	p := fx.followupParams()
	p.Origin = moPinnedOrigin
	res, err := fx.o.MergeOrder(context.Background(), p)
	if err != nil {
		t.Fatalf("MergeOrder: %v", err)
	}
	if len(res.Followups.Items) != 1 || res.Followups.Items[0].State != FollowupMergeReady {
		t.Errorf("followups = %+v, want f1 ready", res.Followups)
	}
}

// TestMergeOrderFollowup_InterruptedLookupIsNoError: ctx done while a
// follow-up's gh runs makes that branch and the ones after it unknown, runs
// nothing more, keeps what was read before, and is no error, since the
// entries are classified by then.
func TestMergeOrderFollowup_InterruptedLookupIsNoError(t *testing.T) {
	fx := newMOFixture(t)
	fx.addFollowup(t, 1, "feat/a")
	fx.addFollowup(t, 2, "feat/b")
	fx.addFollowup(t, 3, "feat/c")
	fx.exec.setPR("feat/a", moFollowupPRs(moMergedPR(7, time.Minute)))
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	p := fx.followupParams()
	p.Exec = func(c context.Context, dir string, env []string, name string, args ...string) (string, error) {
		if name == "gh" && args[3] == "feat/b" {
			cancel()
			return "", c.Err()
		}
		return fx.exec.run(c, dir, env, name, args...)
	}
	res, err := fx.o.MergeOrder(ctx, p)
	if err != nil {
		t.Fatalf("MergeOrder: %v", err)
	}
	want := map[string]string{"f1": FollowupMergeReady, "f2": FollowupMergeUnknown, "f3": FollowupMergeUnknown}
	if got := moFollowupStates(res); !maps.Equal(got, want) {
		t.Fatalf("states = %v, want %v", got, want)
	}
	for _, f := range res.Followups.Items[1:] {
		checkFollowupDetail(t, f, "interrupted (context canceled) while gh ran, so the after-branch's PRs were not read")
	}
	if got := moGHBranches(fx); !slices.Equal(got, []string{"feat/a"}) {
		t.Errorf("gh asked about %q, want feat/a only (feat/b's call cancelled)", got)
	}
}

// TestMergeOrderFollowup_DoneFollowupsAreLeftOut: a done follow-up is neither
// reported nor looked up.
func TestMergeOrderFollowup_DoneFollowupsAreLeftOut(t *testing.T) {
	fx := newMOFixture(t)
	fx.addFollowup(t, 1, "feat/a")
	fx.addFollowup(t, 2, "feat/b")
	if _, err := NewFollowupStore(fx.stateDir).Done("f1", ""); err != nil {
		t.Fatal(err)
	}
	res := fx.runFollowups(t)
	if len(res.Followups.Items) != 1 || res.Followups.Items[0].ID != "f2" {
		t.Errorf("followups = %+v, want f2 alone", res.Followups)
	}
	if got := moGHBranches(fx); !slices.Equal(got, []string{"feat/b"}) {
		t.Errorf("gh asked about %q, want feat/b only", got)
	}
}

// TestMergeOrderFollowup_WithoutTheParamNothingIsRead: without
// MergeOrderParams.Followups (ralph org cleanup's call), the follow-ups are
// an empty list, the record is not reported, and nothing runs for them.
func TestMergeOrderFollowup_WithoutTheParamNothingIsRead(t *testing.T) {
	fx := newMOFixture(t)
	fx.addFollowup(t, 1, "feat/a")
	res := fx.run(t, false)
	if want := (MergeOrderFollowups{Items: []MergeOrderFollowup{}}); !reflect.DeepEqual(res.Followups, want) {
		t.Errorf("followups %#v; want %#v, an empty list and no record", res.Followups, want)
	}
	if len(fx.exec.calls) != 0 {
		t.Errorf("calls = %+v, want none", fx.exec.calls)
	}
}

// TestMergeOrderFollowup_RecordDamageIsCountedAndAnUnreadableRecordIsANote:
// the record's corrupt lines and ignored events are counted in Followups,
// with its path, while its open follow-ups are still reported; a record that
// cannot be read (a directory) leaves the follow-ups empty with its error in
// Followups.Error and in a note, runs nothing, and is no error. A follow-up
// whose added_at does not parse is unknown.
func TestMergeOrderFollowup_RecordDamageIsCountedAndAnUnreadableRecordIsANote(t *testing.T) {
	t.Run("damage", func(t *testing.T) {
		fx := newMOFixture(t)
		fx.addFollowup(t, 1, "feat/a")
		fx.appendFollowupRaw(t, `{"ts":"2026-10-14T08:10:00Z","id":"f2","eve`+"\n"+`{"ts":"2026-10-14T08:11:00Z","id":"f9","event":"done"}`+"\n")
		res := fx.runFollowups(t)
		if got := res.Followups; got.Path != FollowupsPathIn(fx.stateDir) || got.CorruptLines != 1 || got.IgnoredEvents != 1 || got.Error != "" {
			t.Errorf("record = path %q, corrupt %d, ignored %d, error %q; want %q, 1, 1, none", got.Path, got.CorruptLines,
				got.IgnoredEvents, got.Error, FollowupsPathIn(fx.stateDir))
		}
		if len(res.Followups.Items) != 1 || res.Followups.Items[0].ID != "f1" {
			t.Errorf("followups = %+v, want f1", res.Followups)
		}
	})
	t.Run("an unreadable record", func(t *testing.T) {
		fx := newMOFixture(t)
		mkdirAll(t, FollowupsPathIn(fx.stateDir))
		res := fx.runFollowups(t)
		if res.Followups.Items == nil || len(res.Followups.Items) != 0 || len(fx.exec.calls) != 0 {
			t.Errorf("followups %#v, calls %+v; want none of either", res.Followups, fx.exec.calls)
		}
		got := res.Followups
		if !strings.HasPrefix(got.Error, "org: followup: read "+FollowupsPathIn(fx.stateDir)+": ") || got.CorruptLines != 0 ||
			got.IgnoredEvents != 0 || got.Path != FollowupsPathIn(fx.stateDir) {
			t.Errorf("record = %+v; want the read error, no counts, and the path", got)
		}
		if note := moNoteWith(res, "the follow-up record cannot be read ("); note != "the follow-up record cannot be read ("+got.Error+
			"): no follow-up is shown" {
			t.Errorf("notes = %q, want the unreadable record's, with its error", res.Notes)
		}
	})
	t.Run("an added_at that does not parse", func(t *testing.T) {
		fx := newMOFixture(t)
		fx.appendFollowupRaw(t, `{"ts":"yesterday","id":"f1","event":"added","issue":5,"after_branch":"feat/a"}`+"\n")
		fx.exec.setPR("feat/a", moFollowupPRs(moMergedPR(7, time.Minute)))
		res := fx.runFollowups(t)
		if len(res.Followups.Items) != 1 || res.Followups.Items[0].State != FollowupMergeUnknown || res.Followups.Items[0].PR != nil {
			t.Fatalf("followups = %+v, want f1 unknown with no PR", res.Followups)
		}
		checkFollowupDetail(t, res.Followups.Items[0], `the follow-up's added_at "yesterday" is not an RFC3339 time, so no merge can be compared with it`)
	})
}

// TestMergeOrderFollowup_JSONCarriesEveryFieldAndANullPR covers AC3's JSON:
// followups is always there, the object `ralph status --json` gives (items,
// [] with none, corrupt_lines, ignored_events and error, not the path), and
// each follow-up has exactly id, issue, after_branch, added_at, state,
// source, pr (number and url, or null), merged_at and detail.
func TestMergeOrderFollowup_JSONCarriesEveryFieldAndANullPR(t *testing.T) {
	top := func(t *testing.T, res MergeOrderResult) map[string]json.RawMessage {
		t.Helper()
		b, err := json.Marshal(res)
		if err != nil {
			t.Fatal(err)
		}
		var keys map[string]json.RawMessage
		if err := json.Unmarshal(b, &keys); err != nil {
			t.Fatal(err)
		}
		return keys
	}
	empty := newMOFixture(t)
	if got, want := string(top(t, empty.runFollowups(t))["followups"]), `{"items":[],"corrupt_lines":0,"ignored_events":0,"error":""}`; got != want {
		t.Errorf("followups with none = %s, want %s", got, want)
	}

	fx := newMOFixture(t)
	fx.addFollowup(t, 270, "feat/a")
	fx.addFollowup(t, 271, "feat/b")
	fx.exec.setPR("feat/a", moFollowupPRs(moMergedPR(7, time.Minute)))
	keys := top(t, fx.runFollowups(t))
	if len(keys) != 8 {
		t.Errorf("top-level keys = %v, want the 7 of before and followups", slices.Sorted(maps.Keys(keys)))
	}
	var followups map[string]json.RawMessage
	if err := json.Unmarshal(keys["followups"], &followups); err != nil {
		t.Fatalf("followups = %s (%v), want an object", keys["followups"], err)
	}
	if got, want := slices.Sorted(maps.Keys(followups)), []string{"corrupt_lines", "error", "ignored_events", "items"}; !slices.Equal(got, want) {
		t.Errorf("followups keys = %q, want %q", got, want)
	}
	var items []map[string]json.RawMessage
	if err := json.Unmarshal(followups["items"], &items); err != nil || len(items) != 2 {
		t.Fatalf("followups items = %s (%v), want two", followups["items"], err)
	}
	want := []string{"added_at", "after_branch", "detail", "id", "issue", "merged_at", "pr", "source", "state"}
	for i, item := range items {
		if got := slices.Sorted(maps.Keys(item)); !slices.Equal(got, want) {
			t.Errorf("follow-up %d keys = %q, want %q", i, got, want)
		}
	}
	for key, value := range map[string]string{"id": `"f1"`, "issue": "270", "state": `"ready"`, "source": `"gh"`,
		"pr": `{"number":7,"url":"https://github.com/o/r/pull/7"}`, "merged_at": strconv.Quote(moAt(time.Minute))} {
		if got := string(items[0][key]); got != value {
			t.Errorf("f1 %s = %s, want %s", key, got, value)
		}
	}
	if pr, mergedAt := string(items[1]["pr"]), string(items[1]["merged_at"]); pr != "null" || mergedAt != `""` {
		t.Errorf("f2 pr = %s, merged_at = %s; want null and \"\"", pr, mergedAt)
	}
}

// TestMergeOrderFollowup_ReportedWithTargetsAndWithout: the follow-ups come
// with the entries when the ledger has feature branches, and on their own
// (no ls-remote) when it has none; both paths read the record in the
// manifest's state dir.
func TestMergeOrderFollowup_ReportedWithTargetsAndWithout(t *testing.T) {
	withTargets := moLookupFixture(t, moOriginURL, "a")
	withTargets.addFollowup(t, 1, "feat/x")
	if res := withTargets.runFollowups(t); len(res.Followups.Items) != 1 || len(res.Entries) != 1 || res.DefaultBranch != "main" {
		t.Errorf("with a target: followups %+v, entries %+v, default %q", res.Followups, res.Entries, res.DefaultBranch)
	}
	noTargets := newMOFixture(t)
	noTargets.addFollowup(t, 1, "feat/x")
	res := noTargets.runFollowups(t)
	if len(res.Followups.Items) != 1 || len(res.Entries) != 0 || res.DefaultBranch != "" || moNoteWith(res, "no feature branches in the ledger") == "" {
		t.Errorf("with no target: followups %+v, entries %+v, default %q, notes %q", res.Followups, res.Entries, res.DefaultBranch, res.Notes)
	}
	if git := moGitArgs(noTargets.exec); slices.Contains(git, "ls-remote --symref origin HEAD refs/heads/*") {
		t.Errorf("git calls = %q, want no ls-remote with no target", git)
	}
	if p := filepath.Dir(res.Followups.Path); p != noTargets.stateDir {
		t.Errorf("record read in %s, want the manifest's state dir %s", p, noTargets.stateDir)
	}
}

// TestMergeOrderFollowup_GHAvailableWithNoFeatureBranch covers cross-review
// cycle 1's ACTION_REQUIRED #1: with no feature branch, a follow-up's
// after-branch is a branch to ask gh about, so GHAvailable is true when the
// follow-ups' lookup found gh, as lookupMergePRs's available is: gh answered
// with the branch's PRs, or failed for the branch. It is false when gh is
// missing, also after it answered for an earlier branch, and when there is no
// branch to ask about (no record, or only a done follow-up), with no gh call.
func TestMergeOrderFollowup_GHAvailableWithNoFeatureBranch(t *testing.T) {
	for _, tc := range []struct {
		name    string
		setup   func(t *testing.T, fx *moFixture, p *MergeOrderParams)
		gh      bool // GHAvailable
		states  map[string]string
		ghCalls int
	}{
		{"gh answers", func(t *testing.T, fx *moFixture, _ *MergeOrderParams) {
			fx.addFollowup(t, 1, "feat/a")
			fx.exec.setPR("feat/a", moFollowupPRs(moMergedPR(7, time.Minute)))
		}, true, map[string]string{"f1": FollowupMergeReady}, 1},
		{"gh fails for the branch", func(t *testing.T, fx *moFixture, _ *MergeOrderParams) {
			fx.addFollowup(t, 1, "feat/a")
			fx.exec.ghErr = map[string]error{"feat/a": errors.New("exit status 1: HTTP 502")}
		}, true, map[string]string{"f1": FollowupMergeUnknown}, 1},
		{"gh is missing", func(t *testing.T, fx *moFixture, _ *MergeOrderParams) {
			fx.addFollowup(t, 1, "feat/a")
			fx.exec.ghMissing = true
		}, false, map[string]string{"f1": FollowupMergeUnknown}, 1},
		{"gh goes missing after an answer", func(t *testing.T, fx *moFixture, p *MergeOrderParams) {
			fx.addFollowup(t, 1, "feat/a")
			fx.addFollowup(t, 2, "feat/b")
			fx.exec.setPR("feat/a", moFollowupPRs(moMergedPR(7, time.Minute)))
			p.Exec = func(c context.Context, dir string, env []string, name string, args ...string) (string, error) {
				out, err := fx.exec.run(c, dir, env, name, args...)
				if name == "gh" && args[3] == "feat/a" {
					fx.exec.ghMissing = true
				}
				return out, err
			}
		}, false, map[string]string{"f1": FollowupMergeUnknown, "f2": FollowupMergeUnknown}, 2},
		{"no record", func(*testing.T, *moFixture, *MergeOrderParams) {}, false, map[string]string{}, 0},
		{"only a done follow-up", func(t *testing.T, fx *moFixture, _ *MergeOrderParams) {
			fx.addFollowup(t, 1, "feat/a")
			if _, err := NewFollowupStore(fx.stateDir).Done("f1", ""); err != nil {
				t.Fatal(err)
			}
		}, false, map[string]string{}, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fx := newMOFixture(t)
			p := fx.followupParams()
			tc.setup(t, fx, &p)
			res, err := fx.o.MergeOrder(context.Background(), p)
			if err != nil {
				t.Fatalf("MergeOrder: %v", err)
			}
			if res.GHAvailable != tc.gh || len(res.Entries) != 0 {
				t.Errorf("GHAvailable = %v with entries %+v; want %v and no entry", res.GHAvailable, res.Entries, tc.gh)
			}
			if got := moFollowupStates(res); !maps.Equal(got, tc.states) {
				t.Errorf("states = %v, want %v", got, tc.states)
			}
			if got := len(fx.exec.callsOf("gh")); got != tc.ghCalls {
				t.Errorf("gh ran %d times, want %d", got, tc.ghCalls)
			}
		})
	}
}

// TestMergeOrderFollowup_LookupNeverClearsTheEntriesGHAvailable: with a
// feature branch whose lookup found gh, a follow-up lookup that finds gh
// missing leaves GHAvailable true; the follow-up is unknown with the reason.
func TestMergeOrderFollowup_LookupNeverClearsTheEntriesGHAvailable(t *testing.T) {
	fx := moLookupFixture(t, moOriginURL, "a")
	fx.addFollowup(t, 1, "feat/z")
	p := fx.followupParams()
	p.Exec = func(c context.Context, dir string, env []string, name string, args ...string) (string, error) {
		if name == "gh" && strings.HasSuffix(args[9], ",mergedAt") {
			fx.exec.ghMissing = true
		}
		return fx.exec.run(c, dir, env, name, args...)
	}
	res, err := fx.o.MergeOrder(context.Background(), p)
	if err != nil {
		t.Fatalf("MergeOrder: %v", err)
	}
	if !res.GHAvailable || len(res.Entries) != 1 {
		t.Errorf("GHAvailable = %v with entries %+v; want true and feat/a", res.GHAvailable, res.Entries)
	}
	if len(res.Followups.Items) != 1 || res.Followups.Items[0].State != FollowupMergeUnknown {
		t.Fatalf("followups = %+v, want f1 unknown", res.Followups)
	}
	checkFollowupDetail(t, res.Followups.Items[0], "gh was not found, so no PR was read")
}
