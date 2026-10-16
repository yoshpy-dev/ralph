package org

import (
	"cmp"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"
)

// `ralph org merge-order` at the org layer (plan
// docs/plans/active/2026-10-14-org-merge-order.md, the org-layer half of AC1
// to AC7, AC9 and AC11): MergeOrder on manifest fixtures, with a fake
// MergeOrderExec standing in for git ls-remote, git remote get-url origin and
// gh, so no test reaches a network. realMergeOrderExec runs real local
// processes only.

// mergeOrderTestClock is the fixtures' clock: ListedAt reads it.
var mergeOrderTestClock = time.Date(2026, 10, 14, 9, 0, 0, 0, time.UTC)

const mergeOrderTestListedAt = "2026-10-14T09:00:00Z"

// The commits of the fixtures: moMain is origin/main's tip when the gates
// ran, moMain2 its tip after it moved; moHeadA and moHeadB are feature heads.
var (
	moMain  = strings.Repeat("1", 40)
	moMain2 = strings.Repeat("2", 40)
	moHeadA = strings.Repeat("a", 40)
	moHeadB = strings.Repeat("b", 40)
)

// moFixture is a state dir under a temp dir with an empty manifest path, an
// Org reading it with mergeOrderTestClock, and the fake exec MergeOrder runs
// through.
type moFixture struct {
	o        *Org
	root     string // the temp dir holding stateDir
	stateDir string
	path     string
	exec     *moExec
	n        int // events add wrote, for their TS
}

func newMOFixture(t *testing.T) *moFixture {
	t.Helper()
	root := t.TempDir()
	stateDir := filepath.Join(root, "state")
	mkdirAll(t, stateDir)
	path := ManifestPathIn(stateDir)
	return &moFixture{
		o:    &Org{Manifest: NewManifestStoreAtPath(path), Now: func() time.Time { return mergeOrderTestClock }},
		root: root, stateDir: stateDir, path: path, exec: &moExec{},
	}
}

// add appends each line to the manifest: a ManifestEvent as a JSON line,
// given a TS a minute after the previous one when it has none, and a string
// as raw bytes, as they are (a corrupt line ends with "\n" only when the
// test writes it).
func (fx *moFixture) add(t *testing.T, lines ...any) {
	t.Helper()
	for _, line := range lines {
		switch l := line.(type) {
		case ManifestEvent:
			fx.n++
			if l.TS == "" {
				l.TS = mergeOrderTestClock.Add(time.Duration(fx.n-120) * time.Minute).Format(time.RFC3339)
			}
			if err := fx.o.Manifest.Append(l); err != nil {
				t.Fatal(err)
			}
		case string:
			f, err := os.OpenFile(fx.path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := f.WriteString(l); err != nil {
				t.Fatal(err)
			}
			if err := f.Close(); err != nil {
				t.Fatal(err)
			}
		default:
			t.Fatalf("add: a %T is neither a ManifestEvent nor a raw line", line)
		}
	}
}

func (fx *moFixture) params(all bool) MergeOrderParams {
	return MergeOrderParams{Dir: "/repo", All: all, Exec: fx.exec.run}
}

// run is MergeOrder with fx.params(all), failing the test on an error.
func (fx *moFixture) run(t *testing.T, all bool) MergeOrderResult {
	t.Helper()
	res, err := fx.o.MergeOrder(context.Background(), fx.params(all))
	if err != nil {
		t.Fatalf("MergeOrder: %v", err)
	}
	return res
}

// moExec is a fake MergeOrderExec. `git ls-remote` answers lsOut and lsErr
// (lsBlock: it waits for ctx instead), and `git remote get-url origin`
// answers originErr, else origin's URL in origin, else moOriginURL. gh
// answers, per branch (`--head <branch>`), the error in ghErr, else the
// output in gh, else "[]"; with ghBlock it waits for ctx, and with ghMissing
// every gh call fails as a binary not on PATH.
type moExec struct {
	mu        sync.Mutex
	calls     []moCall
	lsOut     string
	lsErr     error
	lsBlock   bool
	origin    string
	originErr error
	ghMissing bool
	gh        map[string]string
	ghErr     map[string]error
	ghBlock   map[string]bool
}

// moOriginURL is origin's URL by default: the repository moPR's urls name.
const moOriginURL = "git@github.com:o/r.git"

type moCall struct {
	dir  string
	env  []string
	name string
	args []string
}

func (e *moExec) run(ctx context.Context, dir string, env []string, name string, args ...string) (string, error) {
	e.mu.Lock()
	e.calls = append(e.calls, moCall{dir: dir, env: slices.Clone(env), name: name, args: slices.Clone(args)})
	e.mu.Unlock()
	if err := ctx.Err(); err != nil {
		return "", err
	}
	switch name {
	case "git":
		if slices.Equal(args, []string{"remote", "get-url", "origin"}) {
			if e.originErr != nil {
				return "", e.originErr
			}
			return cmp.Or(e.origin, moOriginURL) + "\n", nil
		}
		if e.lsBlock {
			<-ctx.Done()
			return "", ctx.Err()
		}
		return e.lsOut, e.lsErr
	case "gh":
		if e.ghMissing {
			return "", &exec.Error{Name: "gh", Err: exec.ErrNotFound}
		}
		branch := args[3]
		if e.ghBlock[branch] {
			<-ctx.Done()
			return "", ctx.Err()
		}
		if err := e.ghErr[branch]; err != nil {
			return "", err
		}
		if out, ok := e.gh[branch]; ok {
			return out, nil
		}
		return "[]", nil
	}
	return "", fmt.Errorf("moExec: unexpected command %s", name)
}

// callsOf returns the calls of the command name.
func (e *moExec) callsOf(name string) []moCall {
	e.mu.Lock()
	defer e.mu.Unlock()
	var out []moCall
	for _, c := range e.calls {
		if c.name == name {
			out = append(out, c)
		}
	}
	return out
}

func (e *moExec) setPR(branch, out string) {
	if e.gh == nil {
		e.gh = make(map[string]string)
	}
	e.gh[branch] = out
}

// moListing is ls-remote output: origin's HEAD naming refs/heads/<def> (and
// its tip when pairs hold def), then one refs/heads/<name> line per
// name, sha pair.
func moListing(def string, pairs ...string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "ref: refs/heads/%s\tHEAD\n", def)
	for i := 0; i+1 < len(pairs); i += 2 {
		if pairs[i] == def {
			fmt.Fprintf(&b, "%s\tHEAD\n", pairs[i+1])
		}
	}
	for i := 0; i+1 < len(pairs); i += 2 {
		fmt.Fprintf(&b, "%s\trefs/heads/%s\n", pairs[i+1], pairs[i])
	}
	return b.String()
}

// moBind is org orgID's complete binding to feature slug of split plan
// split-1, on the branch feat/<slug>.
func moBind(orgID, slug string) ManifestEvent {
	return boundReserveEvent(orgID, testFeature("split-1", slug, "0123456789ab"), "internal/"+slug+"/")
}

// moBindTo is moBind for split plan split on branch.
func moBindTo(orgID, split, slug, branch string) ManifestEvent {
	f := testFeature(split, slug, "0123456789ab")
	f.Branch = branch
	return boundReserveEvent(orgID, f, "internal/"+slug+"/")
}

// moGate is orgID's gate record on branch, a gate_passed when reason is
// empty and a gate_failed otherwise.
func moGate(orgID, branch, head, base, baseRef, reason string) ManifestEvent {
	r := gateRecord{Reason: reason, Head: head, Base: base, BaseRef: baseRef, Branch: branch, By: "alice", Log: "/state/gates/" + orgID + ".log"}
	return ManifestEvent{OrgID: orgID, Event: r.event(), Details: r.details()}
}

// moPassed is a gate_passed on branch that checked head against
// refs/remotes/origin/main at base.
func moPassed(orgID, branch, head, base string) ManifestEvent {
	return moGate(orgID, branch, head, base, "refs/remotes/origin/main", "")
}

// moFailed is moPassed as a gate_failed for reason.
func moFailed(orgID, branch, head, base, reason string) ManifestEvent {
	return moGate(orgID, branch, head, base, "refs/remotes/origin/main", reason)
}

// moGarbled is a gate event of orgID whose Details gateRecordFromDetails
// cannot read.
func moGarbled(orgID, event string) ManifestEvent {
	return ManifestEvent{OrgID: orgID, Event: event, Details: "head=not-a-sha garbled"}
}

func moDisbanded(orgID string) ManifestEvent {
	return ManifestEvent{OrgID: orgID, Event: EventDisbanded}
}

// moCorruptLine is a manifest line that is not JSON: a gate record cut off
// while it was written.
const moCorruptLine = `{"ts":"2026-10-14T08:30:00Z","org_id":"org-a","event":"gate_fa` + "\n"

// moPRs is `gh pr list --json` output holding prs.
func moPRs(prs ...ghPullRequest) string {
	b, err := json.Marshal(prs)
	if err != nil {
		panic(err)
	}
	return string(b)
}

// moPR is an open same-repository PR number n of the branch with head
// moHeadA, merging into base.
func moPR(n int, state, base string) ghPullRequest {
	return ghPullRequest{Number: n, State: state, URL: fmt.Sprintf("https://github.com/o/r/pull/%d", n), HeadRefOid: moHeadA, BaseRefName: base}
}

// moEntry returns res's entry for branch, failing the test when there is
// none.
func moEntry(t *testing.T, res MergeOrderResult, branch string) MergeOrderEntry {
	t.Helper()
	for _, e := range res.Entries {
		if e.Branch == branch {
			return e
		}
	}
	t.Fatalf("no entry for %s in %+v", branch, res.Entries)
	return MergeOrderEntry{}
}

// moOrder is res's entries as "<rank> <class> <branch>".
func moOrder(res MergeOrderResult) []string {
	var out []string
	for _, e := range res.Entries {
		out = append(out, fmt.Sprintf("%d %s %s", e.Rank, e.Class, e.Branch))
	}
	return out
}

// moNoteWith returns the first note of res holding want, "" when none does.
func moNoteWith(res MergeOrderResult, want string) string {
	for _, n := range res.Notes {
		if strings.Contains(n, want) {
			return n
		}
	}
	return ""
}

// TestMergeOrder_ClassifiesAGatePassedAgainstOrigin covers AC1 and AC11: a
// gate_passed is ready only when its head is origin's branch tip, its
// base_ref names origin's default branch now and its base is that branch's
// tip, and a known PR merges into it; otherwise regate, wrong_base or
// rebase, in that order, and no_remote when origin has no such branch.
func TestMergeOrder_ClassifiesAGatePassedAgainstOrigin(t *testing.T) {
	onMain := moListing("main", "main", moMain, "feat/auth", moHeadA)
	tests := []struct {
		name    string
		baseRef string
		listing string
		pr      string // gh's output for feat/auth; "" for no PR
		want    string
		detail  []string
	}{
		{"ready: the head and the base are origin's tips", "refs/remotes/origin/main", onMain, "",
			MergeClassReady, []string{"aaaaaaaaaaaa", "111111111111", "origin/main"}},
		{"ready: a base_ref under refs/heads, from a gate run with no origin", "refs/heads/main", onMain, "",
			MergeClassReady, nil},
		{"ready: an open PR into the default branch", "refs/remotes/origin/main", onMain, moPRs(moPR(7, "OPEN", "main")),
			MergeClassReady, nil},
		{"rebase: origin/main moved after the gate", "refs/remotes/origin/main",
			moListing("main", "main", moMain2, "feat/auth", moHeadA), "",
			MergeClassRebase, []string{"origin/main moved", "gate base 111111111111, now 222222222222"}},
		{"rebase: origin's default branch is now trunk and the old main is still at the gate's base (AC11)", "refs/remotes/origin/main",
			moListing("trunk", "trunk", moMain2, "main", moMain, "feat/auth", moHeadA), "",
			MergeClassRebase, []string{"refs/remotes/origin/main", "now trunk", "origin/trunk"}},
		{"rebase: trunk is at the gate's base too, but the gate checked main", "refs/remotes/origin/main",
			moListing("trunk", "trunk", moMain, "main", moMain, "feat/auth", moHeadA), "",
			MergeClassRebase, []string{"now trunk"}},
		{"rebase: a base_ref that is neither origin's nor a local branch", "refs/tags/main", onMain, "",
			MergeClassRebase, []string{"refs/tags/main"}},
		{"wrong_base: the PR merges into another branch", "refs/remotes/origin/main", onMain, moPRs(moPR(7, "OPEN", "develop")),
			MergeClassWrongBase, []string{"PR #7", "develop", "not origin's default branch main"}},
		{"wrong_base: the PR still merges into main after the default became trunk (AC11)", "refs/remotes/origin/main",
			moListing("trunk", "trunk", moMain2, "main", moMain, "feat/auth", moHeadA), moPRs(moPR(7, "OPEN", "main")),
			MergeClassWrongBase, []string{"merges into main", "trunk"}},
		{"regate: pushed after the gate", "refs/remotes/origin/main",
			moListing("main", "main", moMain, "feat/auth", moHeadB), "",
			MergeClassRegate, []string{"origin has bbbbbbbbbbbb", "the gate checked aaaaaaaaaaaa"}},
		{"regate comes before wrong_base and rebase", "refs/remotes/origin/main",
			moListing("main", "main", moMain2, "feat/auth", moHeadB), moPRs(moPR(7, "OPEN", "develop")),
			MergeClassRegate, nil},
		{"no_remote: origin has no such branch", "refs/remotes/origin/main", moListing("main", "main", moMain), "",
			MergeClassNoRemote, []string{"origin has no branch feat/auth"}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			fx := newMOFixture(t)
			fx.add(t, moBind("org-a", "auth"), moGate("org-a", "feat/auth", moHeadA, moMain, tc.baseRef, ""))
			fx.exec.lsOut = tc.listing
			if tc.pr != "" {
				fx.exec.setPR("feat/auth", tc.pr)
			}
			res := fx.run(t, false)
			if len(res.Entries) != 1 {
				t.Fatalf("entries = %+v, want one", res.Entries)
			}
			e := res.Entries[0]
			if e.Class != tc.want || e.Rank != 1 || e.Branch != "feat/auth" {
				t.Fatalf("entry = %+v, want rank 1 %s feat/auth", e, tc.want)
			}
			for _, want := range tc.detail {
				if !strings.Contains(e.Detail, want) {
					t.Errorf("detail %q does not say %q", e.Detail, want)
				}
			}
			heads := make(map[string]string)
			parseLsRemote(tc.listing, heads)
			if e.RemoteHead != heads["feat/auth"] {
				t.Errorf("remote head = %q, want %q", e.RemoteHead, heads["feat/auth"])
			}
			if e.Gate == nil {
				t.Fatalf("entry %+v has no gate record", e)
			}
			wantGate := MergeOrderGate{Event: EventGatePassed, Readable: true, Head: moHeadA, Base: moMain, BaseRef: tc.baseRef, TS: e.Gate.TS}
			if *e.Gate != wantGate || e.Gate.TS == "" {
				t.Errorf("gate = %+v, want %+v with a TS", e.Gate, wantGate)
			}
			if e.Binding == nil || e.Binding.OrgID != "org-a" || e.Binding.Feature != "auth" || e.Binding.Split != "split-1" || e.Binding.TS == "" {
				t.Errorf("binding = %+v, want org-a's auth of split-1 with a TS", e.Binding)
			}
			if (tc.pr == "") != (e.PR == nil) {
				t.Errorf("PR = %+v, want one only when gh reports one", e.PR)
			}
			if res.ListedAt != mergeOrderTestListedAt || !res.GHAvailable {
				t.Errorf("listed at %q, gh available %v; want %s and true", res.ListedAt, res.GHAvailable, mergeOrderTestListedAt)
			}
			if moNoteWith(res, "branches are ready") != "" {
				t.Errorf("notes = %q, want no note about several ready branches for one branch", res.Notes)
			}
			if moNoteWith(res, "the order is as of the ls-remote at "+mergeOrderTestListedAt) == "" {
				t.Errorf("notes = %q, want the as-of note", res.Notes)
			}
		})
	}
}

// TestMergeOrder_DefaultBranchComesFromOriginsHEAD covers AC11's other half:
// the result names the branch origin's HEAD points to, and its tip, not
// main.
func TestMergeOrder_DefaultBranchComesFromOriginsHEAD(t *testing.T) {
	fx := newMOFixture(t)
	fx.add(t, moBind("org-a", "auth"), moGate("org-a", "feat/auth", moHeadA, moMain2, "refs/remotes/origin/trunk", ""))
	fx.exec.lsOut = moListing("trunk", "main", moMain, "trunk", moMain2, "feat/auth", moHeadA)
	res := fx.run(t, false)
	if res.DefaultBranch != "trunk" || res.DefaultHead != moMain2 {
		t.Fatalf("default = %s %s, want trunk %s", res.DefaultBranch, res.DefaultHead, moMain2)
	}
	if e := moEntry(t, res, "feat/auth"); e.Class != MergeClassReady {
		t.Fatalf("entry = %+v, want ready against trunk", e)
	}
}

// TestMergeOrder_FailedNoGateAndNoRemote covers AC2: a gate_failed is failed
// with its reason, a binding with no gate record is no_gate, and both come
// before no_remote, so a branch origin does not have still says why it
// cannot be merged.
func TestMergeOrder_FailedNoGateAndNoRemote(t *testing.T) {
	fx := newMOFixture(t)
	fx.add(t,
		moBind("org-f", "fail"), moFailed("org-f", "feat/fail", moHeadA, moMain, GateReasonTestFailed),
		moBind("org-n", "none"),
		moBind("org-g", "gone"), moPassed("org-g", "feat/gone", moHeadA, moMain),
		moBind("org-s", "static"), moFailed("org-s", "feat/static", moHeadA, moMain, GateReasonStaticFailed),
	)
	fx.exec.lsOut = moListing("main", "main", moMain, "feat/static", moHeadA)
	res := fx.run(t, false)
	for branch, want := range map[string]struct{ class, detail string }{
		"feat/fail":   {MergeClassFailed, GateReasonTestFailed},
		"feat/static": {MergeClassFailed, GateReasonStaticFailed},
		"feat/none":   {MergeClassNoGate, "no gate record"},
		"feat/gone":   {MergeClassNoRemote, "origin has no branch feat/gone"},
	} {
		e := moEntry(t, res, branch)
		if e.Class != want.class || !strings.Contains(e.Detail, want.detail) {
			t.Errorf("%s = %s (%q), want %s saying %q", branch, e.Class, e.Detail, want.class, want.detail)
		}
	}
	if e := moEntry(t, res, "feat/fail"); e.Gate == nil || e.Gate.Event != EventGateFailed || e.Gate.Reason != GateReasonTestFailed || e.RemoteHead != "" {
		t.Errorf("feat/fail = %+v, want its gate_failed with the reason and no remote head", e)
	}
	if e := moEntry(t, res, "feat/none"); e.Gate != nil || e.Binding == nil {
		t.Errorf("feat/none = %+v, want a binding and no gate", e)
	}
}

// moGateClause is how every Detail and note that says to run the gate ends.
const moGateClause = " (the gate runs only while the feature org runs)"

// TestMergeOrder_DetailsNameTheGateCommandWithTheOrg: every Detail that
// tells the reader to run the gate (unreadable, no_gate, failed, regate,
// rebase) names it with the --org-id of the branch's latest binding and ends
// with the clause that the gate runs only while the feature org runs; a
// branch only a gate record names gets the --org-id of that record, which
// ralph org gate requires; a branch bound by two orgs gets the latest
// binding's org even when its gate record is the other's; and the note about
// several ready branches ends with the same clause.
func TestMergeOrder_DetailsNameTheGateCommandWithTheOrg(t *testing.T) {
	bind := moBind("org-a", "auth")
	onMain := moListing("main", "main", moMain, "feat/auth", moHeadA)
	for _, tc := range []struct {
		name    string
		lines   []any
		listing string
		want    string
	}{
		{"unreadable: a record that cannot be read", []any{bind, moGarbled("org-a", EventGatePassed)}, onMain, MergeClassUnreadable},
		{"unreadable: a corrupt line after the record", []any{bind, moPassed("org-a", "feat/auth", moHeadA, moMain), moCorruptLine},
			onMain, MergeClassUnreadable},
		{"no_gate", []any{bind}, onMain, MergeClassNoGate},
		{"failed", []any{bind, moFailed("org-a", "feat/auth", moHeadA, moMain, GateReasonTestFailed)}, onMain, MergeClassFailed},
		{"regate", []any{bind, moPassed("org-a", "feat/auth", moHeadA, moMain)},
			moListing("main", "main", moMain, "feat/auth", moHeadB), MergeClassRegate},
		{"rebase: the gate checked another branch", []any{bind, moGate("org-a", "feat/auth", moHeadA, moMain, "refs/remotes/origin/develop", "")},
			onMain, MergeClassRebase},
		{"rebase: origin/main moved", []any{bind, moPassed("org-a", "feat/auth", moHeadA, moMain)},
			moListing("main", "main", moMain2, "feat/auth", moHeadA), MergeClassRebase},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fx := newMOFixture(t)
			fx.add(t, tc.lines...)
			fx.exec.lsOut = tc.listing
			e := moEntry(t, fx.run(t, false), "feat/auth")
			if e.Class != tc.want || !strings.Contains(e.Detail, "run ralph org gate --org-id org-a ") || !strings.HasSuffix(e.Detail, moGateClause) {
				t.Fatalf("feat/auth = %s (%q), want %s naming ralph org gate --org-id org-a and ending with %q", e.Class, e.Detail, tc.want, moGateClause)
			}
		})
	}

	fx := newMOFixture(t)
	fx.add(t, moFailed("org-x", "feat/orphan", moHeadA, moMain, GateReasonTestFailed),
		moBind("org-r", "r1"), moPassed("org-r", "feat/r1", moHeadA, moMain),
		moBind("org-s", "r2"), moPassed("org-s", "feat/r2", moHeadA, moMain))
	fx.exec.lsOut = moListing("main", "main", moMain, "feat/orphan", moHeadA, "feat/r1", moHeadA, "feat/r2", moHeadA)
	res := fx.run(t, false)
	if e := moEntry(t, res, "feat/orphan"); e.Class != MergeClassFailed || e.Binding != nil ||
		!strings.Contains(e.Detail, "run ralph org gate --org-id org-x again") || !strings.HasSuffix(e.Detail, moGateClause) {
		t.Errorf("feat/orphan = %s (%q), want failed with no binding, naming ralph org gate --org-id org-x of its gate record", e.Class, e.Detail)
	}
	if note := moNoteWith(res, "2 branches are ready"); !strings.HasSuffix(note, "run ralph org gate again"+moGateClause) {
		t.Errorf("notes = %q, want the several-ready note to end with the gate and its clause", res.Notes)
	}

	twice := newMOFixture(t)
	twice.add(t, bind, moFailed("org-a", "feat/auth", moHeadA, moMain, GateReasonTestFailed),
		moBindTo("org-b", "split-2", "auth2", "feat/auth"))
	twice.exec.lsOut = onMain
	if e := moEntry(t, twice.run(t, false), "feat/auth"); e.Class != MergeClassFailed ||
		!strings.Contains(e.Detail, "run ralph org gate --org-id org-b again") {
		t.Errorf("feat/auth = %s (%q), want failed naming org-b, its latest binding's org, not org-a of its gate record", e.Class, e.Detail)
	}

	// collectMergeTargets attaches no record without an org_id, so only a
	// target built by hand has no org to name: the command is then bare.
	noOrg := &mergeTarget{branch: "feat/x", gate: &mergeGateRecord{ev: ManifestEvent{Event: EventGateFailed}}}
	if got := mergeGateCommand(noOrg); got != "ralph org gate" {
		t.Errorf("gate command for a gate record with no org_id = %q, want the bare ralph org gate", got)
	}
}

// moRegateGate is moGate as ralph org regate records it: via=regate, with
// the log moRegateLog.
func moRegateGate(orgID, branch, head, base, reason string) ManifestEvent {
	r := gateRecord{Reason: reason, Head: head, Base: base, BaseRef: "refs/remotes/origin/main", Branch: branch, By: "alice",
		Via: gateViaRegate, Log: moRegateLog}
	return ManifestEvent{OrgID: orgID, Event: r.event(), Details: r.details()}
}

// moRegateLog is the log of moRegateGate's records.
const moRegateLog = "/state/gates/org-a-20261014T080000Z.log"

// TestMergeOrder_DetailsNameRegateForABranchWhoseOrgIsNotRunning covers
// AC10 of plan docs/plans/active/2026-10-16-org-regate.md: for a branch whose
// latest binding's org was disbanded, every Detail that tells the reader to
// gate the branch (unreadable, no_gate, failed, regate, rebase) names ralph
// org regate --branch <branch>, not ralph org gate, without "in the feature
// worktree" or the clause that the gate runs only while the org runs. regate
// also says the gate's head may not have been pushed, rebase that regate
// merges origin's default branch in, and failed, for a record regate wrote,
// on which head the gate ran (origin's tip, or the merge commit regate made,
// undone or kept by the reason) and its log. A gate record before the
// disband and one after it (regate's) read the same; ready keeps its text.
func TestMergeOrder_DetailsNameRegateForABranchWhoseOrgIsNotRunning(t *testing.T) {
	bind, gone := moBind("org-a", "auth"), moDisbanded("org-a")
	onMain := moListing("main", "main", moMain, "feat/auth", moHeadA)
	moved := moListing("main", "main", moMain2, "feat/auth", moHeadA)
	const next = "fix it in the feature worktree, push it, and run ralph org regate --branch feat/auth"
	merged := func(reason, kept string) string {
		return "the gate ralph org regate ran failed (" + reason + ") on bbbbbbbbbbbb, the merge commit it made of origin/main into " +
			"the branch, which it " + kept + "; its log is " + moRegateLog + ": " + next
	}
	for _, tc := range []struct {
		name    string
		lines   []any
		listing string
		class   string
		detail  string // %s is the TS of the branch's latest gate record
	}{
		{"unreadable: a record that cannot be read", []any{bind, gone, moGarbled("org-a", EventGatePassed)}, onMain, MergeClassUnreadable,
			"the latest gate record (gate_passed at %s) cannot be read: run ralph org regate --branch feat/auth to record a new one"},
		{"unreadable: a corrupt line after the record", []any{bind, moPassed("org-a", "feat/auth", moHeadA, moMain), gone, moCorruptLine},
			onMain, MergeClassUnreadable, "a corrupt manifest line follows the latest gate record (gate_passed at %s) and may be a newer " +
				"one: run ralph org regate --branch feat/auth to record a new one"},
		{"no_gate", []any{bind, gone}, onMain, MergeClassNoGate,
			"no gate record: run ralph org regate --branch feat/auth before the PR is merged"},
		{"failed: the leader's gate before the disband", []any{bind, moFailed("org-a", "feat/auth", moHeadA, moMain, GateReasonTestFailed), gone},
			onMain, MergeClassFailed, "the gate failed (test_failed): " + next},
		{"failed: regate's gate on origin's tip", []any{bind, gone, moRegateGate("org-a", "feat/auth", moHeadA, moMain, GateReasonTestFailed)},
			onMain, MergeClassFailed, "the gate ralph org regate ran on origin's tip aaaaaaaaaaaa of the branch failed (test_failed); its log " +
				"is " + moRegateLog + ": " + next},
		{"failed: regate's merge commit, static_failed", []any{bind, gone, moRegateGate("org-a", "feat/auth", moHeadB, moMain2, GateReasonStaticFailed)},
			moved, MergeClassFailed, merged(GateReasonStaticFailed, "undid")},
		{"failed: regate's merge commit, test_failed", []any{bind, gone, moRegateGate("org-a", "feat/auth", moHeadB, moMain2, GateReasonTestFailed)},
			moved, MergeClassFailed, merged(GateReasonTestFailed, "undid")},
		{"failed: regate's merge commit, head_moved", []any{bind, gone, moRegateGate("org-a", "feat/auth", moHeadB, moMain2, GateReasonHeadMoved)},
			moved, MergeClassFailed, merged(GateReasonHeadMoved, "kept in the feature worktree")},
		{"failed: regate's merge commit, tree_changed", []any{bind, gone, moRegateGate("org-a", "feat/auth", moHeadB, moMain2, GateReasonTreeChanged)},
			moved, MergeClassFailed, merged(GateReasonTreeChanged, "kept in the feature worktree")},
		{"regate: origin's tip is not the gate's head", []any{bind, gone, moRegateGate("org-a", "feat/auth", moHeadB, moMain, "")}, onMain,
			MergeClassRegate, "pushed after the gate, or the gate's head was not pushed: origin has aaaaaaaaaaaa, the gate checked " +
				"bbbbbbbbbbbb: run ralph org regate --branch feat/auth"},
		{"rebase: the gate checked another branch", []any{bind, moGate("org-a", "feat/auth", moHeadA, moMain, "refs/remotes/origin/develop", ""), gone},
			onMain, MergeClassRebase, "the gate checked against refs/remotes/origin/develop, but origin's default branch is now main: run " +
				"ralph org regate --branch feat/auth, which merges origin/main in and runs the gate again"},
		{"rebase: origin/main moved", []any{bind, moPassed("org-a", "feat/auth", moHeadA, moMain), gone}, moved, MergeClassRebase,
			"origin/main moved: gate base 111111111111, now 222222222222: run ralph org regate --branch feat/auth, which merges " +
				"origin/main in and runs the gate again"},
		{"ready keeps its text", []any{bind, gone, moRegateGate("org-a", "feat/auth", moHeadA, moMain, "")}, onMain, MergeClassReady,
			"the gate passed on aaaaaaaaaaaa, origin's tip of the branch, with base 111111111111, origin/main's tip"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fx := newMOFixture(t)
			fx.add(t, tc.lines...)
			fx.exec.lsOut = tc.listing
			e := moEntry(t, fx.run(t, false), "feat/auth")
			want := tc.detail
			if e.Gate != nil && strings.Contains(want, "%s") {
				want = fmt.Sprintf(want, e.Gate.TS)
			}
			if e.Class != tc.class || e.Detail != want {
				t.Fatalf("feat/auth = %s\n%q\nwant %s\n%q", e.Class, e.Detail, tc.class, want)
			}
		})
	}
}

// TestMergeOrder_DetailsOfARunningOrgOrNoBindingAreUnchanged covers AC10's
// other half: the Details of a branch whose latest binding's org runs bound
// to it (also when the org_id was disbanded and bound again to the branch,
// after another feature or not), and of a branch only gate records name, are
// what they were before regate, byte for byte: ralph org gate --org-id <org>
// and the clause that the gate runs only while the org runs. A gate_failed
// regate wrote reads as any other then. A running org bound to another
// branch since is TestMergeOrder_DetailsOfABranchWhoseOrgRunsAnotherFeature.
func TestMergeOrder_DetailsOfARunningOrgOrNoBindingAreUnchanged(t *testing.T) {
	const clause = " (the gate runs only while the feature org runs)"
	onMain := moListing("main", "main", moMain, "feat/auth", moHeadA)
	for _, who := range []struct {
		name           string
		before, after  func(orgID string) []any
		orgID          string
		noGateWithNone bool // the branch is listed with no gate record
	}{
		{"a running org", func(o string) []any { return []any{moBind(o, "auth")} }, func(string) []any { return nil }, "org-a", true},
		{"an org_id bound again to the same feature, which runs", func(o string) []any { return []any{moBind(o, "auth")} },
			func(o string) []any { return []any{moDisbanded(o), moBind(o, "auth")} }, "org-a", true},
		{"an org_id bound to another feature and then to this one again, which runs", func(o string) []any {
			return []any{moBind(o, "auth")}
		}, func(o string) []any {
			return []any{moDisbanded(o), moBindTo(o, "split-1", "other", "feat/other"), moDisbanded(o), moBind(o, "auth")}
		}, "org-a", true},
		{"no binding", func(o string) []any { return []any{moPassed(o, "feat/auth", moHeadA, moMain)} }, func(string) []any { return nil },
			"org-x", false},
	} {
		gate := "ralph org gate --org-id " + who.orgID
		for _, tc := range []struct {
			name    string
			lines   func(orgID string) []any
			listing string
			class   string
			detail  string // %s is the TS of the branch's latest gate record
		}{
			{"unreadable", func(o string) []any { return []any{moGarbled(o, EventGatePassed)} }, onMain, MergeClassUnreadable,
				"the latest gate record (gate_passed at %s) cannot be read: run " + gate + " again in the feature worktree" + clause},
			{"unreadable: a corrupt line", func(o string) []any { return []any{moPassed(o, "feat/auth", moHeadA, moMain), moCorruptLine} }, onMain,
				MergeClassUnreadable, "a corrupt manifest line follows the latest gate record (gate_passed at %s) and may be a newer one: run " +
					gate + " again in the feature worktree" + clause},
			{"no_gate", func(string) []any { return nil }, onMain, MergeClassNoGate,
				"no gate record: run " + gate + " in the feature worktree before the PR is merged" + clause},
			{"failed", func(o string) []any { return []any{moFailed(o, "feat/auth", moHeadA, moMain, GateReasonTestFailed)} }, onMain,
				MergeClassFailed, "the gate failed (test_failed): fix it and run " + gate + " again" + clause},
			{"failed: a record regate wrote", func(o string) []any {
				return []any{moRegateGate(o, "feat/auth", moHeadB, moMain, GateReasonTestFailed)}
			}, onMain, MergeClassFailed, "the gate failed (test_failed): fix it and run " + gate + " again" + clause},
			{"regate", func(o string) []any { return []any{moPassed(o, "feat/auth", moHeadA, moMain)} },
				moListing("main", "main", moMain, "feat/auth", moHeadB), MergeClassRegate,
				"pushed after the gate: origin has bbbbbbbbbbbb, the gate checked aaaaaaaaaaaa: run " + gate + " again" + clause},
			{"rebase: another branch", func(o string) []any {
				return []any{moGate(o, "feat/auth", moHeadA, moMain, "refs/remotes/origin/develop", "")}
			}, onMain, MergeClassRebase, "the gate checked against refs/remotes/origin/develop, but origin's default branch is now main: " +
				"rebase on origin/main and run " + gate + " again" + clause},
			{"rebase: origin/main moved", func(o string) []any { return []any{moPassed(o, "feat/auth", moHeadA, moMain)} },
				moListing("main", "main", moMain2, "feat/auth", moHeadA), MergeClassRebase,
				"origin/main moved: gate base 111111111111, now 222222222222: rebase on origin/main and run " + gate + " again" + clause},
		} {
			if tc.class == MergeClassNoGate && !who.noGateWithNone {
				continue
			}
			t.Run(who.name+"/"+tc.name, func(t *testing.T) {
				fx := newMOFixture(t)
				fx.add(t, who.before(who.orgID)...)
				fx.add(t, tc.lines(who.orgID)...)
				fx.add(t, who.after(who.orgID)...)
				fx.exec.lsOut = tc.listing
				e := moEntry(t, fx.run(t, false), "feat/auth")
				want := tc.detail
				if e.Gate != nil && strings.Contains(want, "%s") {
					want = fmt.Sprintf(want, e.Gate.TS)
				}
				if e.Class != tc.class || e.Detail != want {
					t.Fatalf("feat/auth = %s\n%q\nwant %s\n%q", e.Class, e.Detail, tc.class, want)
				}
			})
		}
	}
}

// TestMergeOrder_DetailsOfABranchWhoseOrgRunsAnotherFeature covers a branch
// whose latest binding's org_id runs, but was started again for another
// feature since (its latest complete binding is of another branch): ralph
// org gate --org-id <org> would gate that feature's worktree, so every Detail
// that tells the reader to gate the branch (unreadable, no_gate, failed,
// regate, rebase) names the org_id and the branch it runs now, and says to
// run ralph org regate --branch <branch> once the org is disbanded, with
// regate's wording of each class, as for a branch whose org is not running.
// ready keeps its text, and the entry's JSON is the one of the same branch
// while the org runs bound to it, apart from Detail.
func TestMergeOrder_DetailsOfABranchWhoseOrgRunsAnotherFeature(t *testing.T) {
	onMain := moListing("main", "main", moMain, "feat/auth", moHeadA)
	moved := moListing("main", "main", moMain2, "feat/auth", moHeadA)
	const runsOn = "org_id org-a now runs another feature, on the branch feat/other: "
	const regate = "once org-a is disbanded, run ralph org regate --branch feat/auth"
	const next = runsOn + "fix it in the feature worktree, push it, and " + regate
	for _, tc := range []struct {
		name    string
		lines   []any
		listing string
		class   string
		detail  string // %s is the TS of the branch's latest gate record
	}{
		{"unreadable: a record that cannot be read", []any{moGarbled("org-a", EventGatePassed)}, onMain, MergeClassUnreadable,
			"the latest gate record (gate_passed at %s) cannot be read: " + runsOn + regate + " to record a new one"},
		{"unreadable: a corrupt line after the record", []any{moPassed("org-a", "feat/auth", moHeadA, moMain), moCorruptLine}, onMain,
			MergeClassUnreadable, "a corrupt manifest line follows the latest gate record (gate_passed at %s) and may be a newer one: " +
				runsOn + regate + " to record a new one"},
		{"no_gate", nil, onMain, MergeClassNoGate, "no gate record: " + runsOn + regate + " before the PR is merged"},
		{"failed: the leader's gate", []any{moFailed("org-a", "feat/auth", moHeadA, moMain, GateReasonTestFailed)}, onMain,
			MergeClassFailed, "the gate failed (test_failed): " + next},
		{"failed: regate's gate on origin's tip", []any{moRegateGate("org-a", "feat/auth", moHeadA, moMain, GateReasonTestFailed)},
			onMain, MergeClassFailed, "the gate ralph org regate ran on origin's tip aaaaaaaaaaaa of the branch failed (test_failed); " +
				"its log is " + moRegateLog + ": " + next},
		{"failed: regate's merge commit", []any{moRegateGate("org-a", "feat/auth", moHeadB, moMain2, GateReasonTreeChanged)}, moved,
			MergeClassFailed, "the gate ralph org regate ran failed (tree_changed) on bbbbbbbbbbbb, the merge commit it made of " +
				"origin/main into the branch, which it kept in the feature worktree; its log is " + moRegateLog + ": " + next},
		{"regate", []any{moPassed("org-a", "feat/auth", moHeadB, moMain)}, onMain, MergeClassRegate,
			"pushed after the gate, or the gate's head was not pushed: origin has aaaaaaaaaaaa, the gate checked bbbbbbbbbbbb: " +
				runsOn + regate},
		{"rebase: the gate checked another branch", []any{moGate("org-a", "feat/auth", moHeadA, moMain, "refs/remotes/origin/develop", "")},
			onMain, MergeClassRebase, "the gate checked against refs/remotes/origin/develop, but origin's default branch is now main: " +
				runsOn + regate + ", which merges origin/main in and runs the gate again"},
		{"rebase: origin/main moved", []any{moPassed("org-a", "feat/auth", moHeadA, moMain)}, moved, MergeClassRebase,
			"origin/main moved: gate base 111111111111, now 222222222222: " + runsOn + regate +
				", which merges origin/main in and runs the gate again"},
		{"ready keeps its text", []any{moPassed("org-a", "feat/auth", moHeadA, moMain)}, onMain, MergeClassReady,
			"the gate passed on aaaaaaaaaaaa, origin's tip of the branch, with base 111111111111, origin/main's tip"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			// entry is feat/auth's entry after org-a's binding to it, tc.lines,
			// and after.
			entry := func(after ...any) MergeOrderEntry {
				fx := newMOFixture(t)
				fx.add(t, moBind("org-a", "auth"))
				fx.add(t, tc.lines...)
				fx.add(t, after...)
				fx.exec.lsOut = tc.listing
				return moEntry(t, fx.run(t, false), "feat/auth")
			}
			e := entry(moDisbanded("org-a"), moBindTo("org-a", "split-1", "other", "feat/other"))
			want := tc.detail
			if e.Gate != nil && strings.Contains(want, "%s") {
				want = fmt.Sprintf(want, e.Gate.TS)
			}
			if e.Class != tc.class || e.Detail != want {
				t.Fatalf("feat/auth = %s\n%q\nwant %s\n%q", e.Class, e.Detail, tc.class, want)
			}

			// The same branch while org-a runs bound to it (feat/other is
			// another org's): its Detail names ralph org gate, and the rest of
			// the entry is the same.
			bound := entry(moBindTo("org-b", "split-1", "other", "feat/other"))
			if tc.class != MergeClassReady && !strings.Contains(bound.Detail, "run ralph org gate --org-id org-a ") {
				t.Fatalf("feat/auth while org-a runs bound to it = %q; want ralph org gate --org-id org-a", bound.Detail)
			}
			e.Detail, bound.Detail = "", ""
			got, err := json.Marshal(e)
			if err != nil {
				t.Fatal(err)
			}
			wantJSON, err := json.Marshal(bound)
			if err != nil {
				t.Fatal(err)
			}
			if string(got) != string(wantJSON) {
				t.Errorf("feat/auth's JSON apart from Detail =\n%s\nwant the one while org-a runs bound to it\n%s", got, wantJSON)
			}
		})
	}
}

// TestMergeOrder_SeveralReadyNoteNamesRegateForADisbandedOrg covers AC10 in
// the note about several ready branches: it is what it was while every ready
// branch's org runs (a disbanded org's branch that is not ready does not
// count), and adds ralph org regate when the org of one of them is
// disbanded.
func TestMergeOrder_SeveralReadyNoteNamesRegateForADisbandedOrg(t *testing.T) {
	const old = "2 branches are ready, but merging one moves origin/main: the other ready ones then become rebase, so rebase them " +
		"on origin/main and run ralph org gate again (the gate runs only while the feature org runs)"
	for _, tc := range []struct {
		name string
		gone []any
		want string
	}{
		{"both orgs run", nil, old},
		{"the disbanded org's branch is not ready", []any{moBind("org-c", "c"), moDisbanded("org-c")}, old},
		{"one org is disbanded", []any{moDisbanded("org-b")},
			old + "; a branch whose org is disbanded is brought up to date with ralph org regate --branch <branch> instead"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fx := newMOFixture(t)
			fx.add(t, moBind("org-a", "a"), moPassed("org-a", "feat/a", moHeadA, moMain), moBind("org-b", "b"),
				moPassed("org-b", "feat/b", moHeadB, moMain))
			fx.add(t, tc.gone...)
			fx.exec.lsOut = moListing("main", "main", moMain, "feat/a", moHeadA, "feat/b", moHeadB, "feat/c", moHeadA)
			if note := moNoteWith(fx.run(t, false), "branches are ready"); note != tc.want {
				t.Fatalf("note = %q\nwant %q", note, tc.want)
			}
		})
	}
}

// TestMergeOrder_GateViaIsInTheEntryAndItsJSON: a gate record ralph org
// regate wrote shows via=regate in the entry's gate and as "via" in its
// JSON; one ralph org gate wrote has no via, and its JSON no "via" key.
func TestMergeOrder_GateViaIsInTheEntryAndItsJSON(t *testing.T) {
	fx := newMOFixture(t)
	fx.add(t, moBind("org-a", "a"), moPassed("org-a", "feat/a", moHeadA, moMain),
		moBind("org-b", "b"), moDisbanded("org-b"), moRegateGate("org-b", "feat/b", moHeadB, moMain, ""))
	fx.exec.lsOut = moListing("main", "main", moMain, "feat/a", moHeadA, "feat/b", moHeadB)
	res := fx.run(t, false)
	for branch, want := range map[string]string{"feat/a": "", "feat/b": gateViaRegate} {
		e := moEntry(t, res, branch)
		if e.Gate == nil || e.Gate.Via != want || e.Class != MergeClassReady {
			t.Fatalf("%s = %+v; want a ready entry whose gate has via %q", branch, e, want)
		}
		b, err := json.Marshal(e.Gate)
		if err != nil {
			t.Fatal(err)
		}
		var keys map[string]json.RawMessage
		if err := json.Unmarshal(b, &keys); err != nil {
			t.Fatal(err)
		}
		if via, ok := keys["via"]; (want == "") == ok || (ok && string(via) != `"regate"`) {
			t.Errorf("%s's gate JSON = %s; want \"via\":\"regate\" only for the record regate wrote", branch, b)
		}
	}
}

// TestMergeOrder_LatestGateRecordWins covers AC2 and AC4 (C8): a branch's
// class reads its latest gate record. One that cannot be read (Details
// gateRecordFromDetails refuses, or an event name that disagrees with the
// reason), or a corrupt manifest line after the latest readable one, makes
// the branch unreadable and never falls back to an earlier gate_passed. A
// corrupt line before the latest record does not matter, a readable record
// after it clears it, and of several corrupt lines the last one decides.
func TestMergeOrder_LatestGateRecordWins(t *testing.T) {
	pass := moPassed("org-a", "feat/auth", moHeadA, moMain)
	fail := moFailed("org-a", "feat/auth", moHeadA, moMain, GateReasonTestFailed)
	passNamedFailed := ManifestEvent{OrgID: "org-a", Event: EventGateFailed, Details: pass.Details}
	failNamedPassed := ManifestEvent{OrgID: "org-a", Event: EventGatePassed, Details: fail.Details}
	other := moBind("org-b", "other")
	tests := []struct {
		name     string
		lines    []any
		want     string
		detail   string
		readable bool // the entry's Gate.Readable
	}{
		{"a pass then a fail is failed", []any{pass, fail}, MergeClassFailed, GateReasonTestFailed, true},
		{"a fail then a pass is ready", []any{fail, pass}, MergeClassReady, "", true},
		{"a pass then a record that cannot be read is unreadable", []any{pass, moGarbled("org-a", EventGateFailed)},
			MergeClassUnreadable, "cannot be read", false},
		{"a pass then a gate_passed holding a reason is unreadable", []any{pass, failNamedPassed},
			MergeClassUnreadable, "(gate_passed at", false},
		{"a pass then a gate_failed without a reason is unreadable", []any{pass, passNamedFailed},
			MergeClassUnreadable, "(gate_failed at", false},
		{"a record that cannot be read then a pass is ready", []any{moGarbled("org-a", EventGatePassed), pass},
			MergeClassReady, "", true},
		{"a pass then a corrupt line is unreadable", []any{pass, moCorruptLine},
			MergeClassUnreadable, "corrupt manifest line follows", true},
		{"a pass, a corrupt line and another org's binding is still unreadable", []any{pass, moCorruptLine, other},
			MergeClassUnreadable, "corrupt manifest line follows", true},
		{"a corrupt line before the only gate record does not matter", []any{moCorruptLine, pass},
			MergeClassReady, "", true},
		{"a newer pass after a corrupt line clears it", []any{pass, moCorruptLine, pass},
			MergeClassReady, "", true},
		{"a newer fail after a corrupt line is failed", []any{pass, moCorruptLine, fail},
			MergeClassFailed, GateReasonTestFailed, true},
		{"of corrupt lines before and after the only gate record, the last one decides", []any{moCorruptLine, pass, moCorruptLine},
			MergeClassUnreadable, "corrupt manifest line follows", true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			fx := newMOFixture(t)
			fx.add(t, moBind("org-a", "auth"))
			fx.add(t, tc.lines...)
			fx.exec.lsOut = moListing("main", "main", moMain, "feat/auth", moHeadA, "feat/other", moHeadB)
			res := fx.run(t, false)
			e := moEntry(t, res, "feat/auth")
			if e.Class != tc.want || !strings.Contains(e.Detail, tc.detail) {
				t.Fatalf("feat/auth = %s (%q), want %s saying %q", e.Class, e.Detail, tc.want, tc.detail)
			}
			if e.Gate == nil || e.Gate.Readable != tc.readable {
				t.Fatalf("gate = %+v, want readable %v", e.Gate, tc.readable)
			}
			if !tc.readable && (e.Gate.Head != "" || e.Gate.Base != "" || e.Gate.Reason != "") {
				t.Errorf("gate = %+v, want nothing read from a record that cannot be read", e.Gate)
			}
			corrupt := 0
			for _, line := range tc.lines {
				if s, isRaw := line.(string); isRaw && s == moCorruptLine {
					corrupt++
				}
			}
			switch note := moNoteWith(res, "corrupt line(s)"); {
			case corrupt == 0 && note != "":
				t.Errorf("notes = %q, want no corrupt-line note without a corrupt line", res.Notes)
			case corrupt > 0 && note != fmt.Sprintf("the manifest has %d corrupt line(s) that cannot be read: a branch whose latest "+
				"gate record comes before one is unreadable until ralph org gate (while the branch's org runs) or ralph org regate "+
				"--branch <branch> (once that org is disbanded) records a new one", corrupt):
				t.Errorf("notes = %q, want the corrupt-line note counting %d and naming the gate and regate", res.Notes, corrupt)
			}
		})
	}
}

// TestMergeOrder_UnreadableRecordAttachesToTheBindingBeforeIt covers AC4's
// last sentence (C8): a gate record that cannot be read attaches to the
// branch of its org's latest complete binding before it, even when the org
// was bound to another feature later, and one whose org has neither a
// complete binding nor a readable gate record before it attaches to no
// branch and is only counted in a note.
func TestMergeOrder_UnreadableRecordAttachesToTheBindingBeforeIt(t *testing.T) {
	fx := newMOFixture(t)
	half := testFeature("split-1", "half", "0123456789ab")
	half.Worktree = ""
	fx.add(t,
		moBind("org-a", "x"), moPassed("org-a", "feat/x", moHeadA, moMain),
		moGarbled("org-a", EventGateFailed),
		moDisbanded("org-a"),
		moBind("org-a", "y"),
		moGarbled("org-b", EventGatePassed), // org-b has no binding yet
		moBind("org-b", "z"),
		boundReserveEvent("org-c", half, "internal/half/"), // incomplete
		moGarbled("org-c", EventGatePassed),
	)
	fx.exec.lsOut = moListing("main", "main", moMain, "feat/x", moHeadA, "feat/y", moHeadA, "feat/z", moHeadA)
	res := fx.run(t, false)
	got := map[string]string{}
	for _, e := range res.Entries {
		got[e.Branch] = e.Class
	}
	want := map[string]string{"feat/x": MergeClassUnreadable, "feat/y": MergeClassNoGate, "feat/z": MergeClassNoGate}
	if fmt.Sprint(got) != fmt.Sprint(want) {
		t.Fatalf("classes = %v, want %v", got, want)
	}
	if moNoteWith(res, "2 gate record(s) cannot be read") == "" {
		t.Fatalf("notes = %q, want org-b's and org-c's records counted as attached to no branch", res.Notes)
	}
}

// TestMergeOrder_UnreadableRecordAttachesToEveryBranchOfItsOrg covers C8
// when one org_id was bound to two branches before a gate record that cannot
// be read (plan docs/plans/active/2026-10-16-org-regate.md, D2): ralph org
// regate records under the org of a branch's binding after the org_id was
// bound to another branch, so the record may be either branch's, and it
// attaches to both, so neither falls back to its older pass. org-a is bound
// to auth and disbanded, then bound to billing, with a pass of feat/auth that
// regate recorded after both disbands, or a pass of each branch while its
// binding ran (and org-a running billing still). Both branches are
// unreadable, and the Detail says the record may be the other branch's.
func TestMergeOrder_UnreadableRecordAttachesToEveryBranchOfItsOrg(t *testing.T) {
	const ts = "2026-10-14T09:00:00Z"
	why := "the latest gate record (gate_failed at " + ts + ") cannot be read and may be a record of another branch of the same " +
		"org, org-a, which has had 2 branches: "
	for _, tc := range []struct {
		name  string
		lines []any
		want  map[string]string // branch -> its Detail
	}{
		{"regate's pass of the earlier binding's branch", []any{moBind("org-a", "auth"), moDisbanded("org-a"), moBind("org-a", "billing"),
			moDisbanded("org-a"), moRegateGate("org-a", "feat/auth", moHeadA, moMain, "")}, map[string]string{
			"feat/auth":    why + "run ralph org regate --branch feat/auth to record a new one",
			"feat/billing": why + "run ralph org regate --branch feat/billing to record a new one"}},
		{"a pass of each branch while its binding ran", []any{moBind("org-a", "auth"), moPassed("org-a", "feat/auth", moHeadA, moMain),
			moDisbanded("org-a"), moBind("org-a", "billing"), moPassed("org-a", "feat/billing", moHeadA, moMain)}, map[string]string{
			"feat/auth": why + "org_id org-a now runs another feature, on the branch feat/billing: once org-a is disbanded, run ralph " +
				"org regate --branch feat/auth to record a new one",
			"feat/billing": why + "run ralph org gate --org-id org-a again in the feature worktree " + mergeGateRunsWhileOrgRuns}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fx := newMOFixture(t)
			fx.add(t, tc.lines...)
			garbled := moGarbled("org-a", EventGateFailed)
			garbled.TS = ts
			fx.add(t, garbled)
			fx.exec.lsOut = moListing("main", "main", moMain, "feat/auth", moHeadA, "feat/billing", moHeadA)
			res := fx.run(t, false)
			for branch, want := range tc.want {
				e := moEntry(t, res, branch)
				if e.Class != MergeClassUnreadable || e.Gate == nil || e.Gate.Readable || e.Gate.Event != EventGateFailed || e.Gate.TS != ts {
					t.Errorf("%s = %s, gate %+v; want unreadable, with the gate_failed at %s that cannot be read", branch, e.Class, e.Gate, ts)
				}
				if e.Detail != want {
					t.Errorf("%s detail:\n%q\nwant\n%q", branch, e.Detail, want)
				}
			}
			if len(res.Entries) != len(tc.want) || moNoteWith(res, "gate record(s) cannot be read") != "" {
				t.Errorf("entries = %+v, notes = %q; want feat/auth and feat/billing only, and the record attached", res.Entries, res.Notes)
			}
		})
	}
}

// TestMergeOrder_UnreadableRecordWithNoBindingAttachesToItsOrgsGatedBranches
// covers cross-review cycle 3's finding (C8) of org-merge-order (triage:
// docs/reports/cross-review-triage-org-merge-order.md): an org with no
// complete binding (a ledger whose binding line was lost or written by hand)
// gates a branch only its gate records name. A gate_failed of that org that
// cannot be read attaches to the branches of the org's readable gate records
// before it, so feat/c is unreadable instead of falling back to its older
// gate_passed, and both of org-e's gated branches, feat/e and feat/e2, take
// it (plan docs/plans/active/2026-10-16-org-regate.md, D2). The note counts
// only org-d's record, whose org has neither a binding nor a readable gate
// record before it.
func TestMergeOrder_UnreadableRecordWithNoBindingAttachesToItsOrgsGatedBranches(t *testing.T) {
	fx := newMOFixture(t)
	fx.add(t,
		moPassed("org-c", "feat/c", moHeadA, moMain), moGarbled("org-c", EventGateFailed),
		moGarbled("org-d", EventGatePassed),
		moPassed("org-e", "feat/e", moHeadA, moMain), moPassed("org-e", "feat/e2", moHeadA, moMain), moGarbled("org-e", EventGateFailed),
	)
	fx.exec.lsOut = moListing("main", "main", moMain, "feat/c", moHeadA, "feat/e", moHeadA, "feat/e2", moHeadA)
	res := fx.run(t, false)
	got := map[string]string{}
	for _, e := range res.Entries {
		got[e.Branch] = e.Class
	}
	want := map[string]string{"feat/c": MergeClassUnreadable, "feat/e": MergeClassUnreadable, "feat/e2": MergeClassUnreadable}
	if fmt.Sprint(got) != fmt.Sprint(want) {
		t.Fatalf("classes = %v, want %v", got, want)
	}
	e := moEntry(t, res, "feat/c")
	if e.Gate == nil || e.Gate.Readable || e.Gate.Event != EventGateFailed || e.Binding != nil {
		t.Fatalf("feat/c = %+v, want the gate_failed that cannot be read and no binding", e)
	}
	if !strings.Contains(e.Detail, "(gate_failed at ") || !strings.Contains(e.Detail, "run ralph org gate --org-id org-c again") {
		t.Errorf("feat/c detail = %q, want the unreadable gate_failed and the gate command for org-c", e.Detail)
	}
	note := moNoteWith(res, "gate record(s) cannot be read")
	if !strings.HasPrefix(note, "1 gate record(s) cannot be read and their org had neither a feature binding nor a readable gate record "+
		"before them") {
		t.Errorf("notes = %q, want only org-d's record counted as attached to no branch", res.Notes)
	}
}

// TestMergeOrder_UnreadableRecordAttachesToTheBoundAndTheGatedBranch covers
// C8's two places for a gate record that cannot be read, the branches of its
// org's complete bindings and of its readable gate records (plan
// docs/plans/active/2026-10-16-org-regate.md, D2): org-b's readable record
// names feat/x, its binding feat/b, and the record that cannot be read goes
// to both, whichever of the two came first, so feat/x does not keep its pass.
func TestMergeOrder_UnreadableRecordAttachesToTheBoundAndTheGatedBranch(t *testing.T) {
	for _, tc := range []struct {
		name  string
		lines []any
	}{
		{"the gate record before the binding", []any{moPassed("org-b", "feat/x", moHeadA, moMain), moBind("org-b", "b")}},
		{"the binding before the gate record", []any{moBind("org-b", "b"), moPassed("org-b", "feat/x", moHeadA, moMain)}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fx := newMOFixture(t)
			fx.add(t, tc.lines...)
			fx.add(t, moGarbled("org-b", EventGateFailed))
			fx.exec.lsOut = moListing("main", "main", moMain, "feat/b", moHeadA, "feat/x", moHeadA)
			res := fx.run(t, false)
			got := map[string]string{}
			for _, e := range res.Entries {
				got[e.Branch] = e.Class
			}
			if want := map[string]string{"feat/b": MergeClassUnreadable, "feat/x": MergeClassUnreadable}; fmt.Sprint(got) != fmt.Sprint(want) {
				t.Fatalf("classes = %v, want %v", got, want)
			}
			if note := moNoteWith(res, "gate record(s) cannot be read"); note != "" {
				t.Errorf("notes = %q, want no record counted as attached to no branch", res.Notes)
			}
		})
	}
}

// TestMergeOrder_TargetsAreTheBranchesOfEveryBinding covers AC3 (C3): the
// branches of a disbanded org and of both features one org_id was started
// for are listed, each with the binding it was started for, and a branch
// only a readable gate record names is listed with no binding.
func TestMergeOrder_TargetsAreTheBranchesOfEveryBinding(t *testing.T) {
	fx := newMOFixture(t)
	fx.add(t,
		moBind("org-a", "one"), moPassed("org-a", "feat/one", moHeadA, moMain), moDisbanded("org-a"),
		moBind("org-a", "two"),
		moBind("org-b", "three"), moDisbanded("org-b"),
		moPassed("org-c", "feat/orphan", moHeadA, moMain),
	)
	fx.exec.lsOut = moListing("main", "main", moMain, "feat/one", moHeadA, "feat/orphan", moHeadA)
	res := fx.run(t, false)
	for branch, want := range map[string]*MergeOrderBinding{
		"feat/one":    {OrgID: "org-a", Feature: "one", Split: "split-1"},
		"feat/two":    {OrgID: "org-a", Feature: "two", Split: "split-1"},
		"feat/three":  {OrgID: "org-b", Feature: "three", Split: "split-1"},
		"feat/orphan": nil,
	} {
		e := moEntry(t, res, branch)
		switch {
		case want == nil && e.Binding != nil:
			t.Errorf("%s binding = %+v, want none", branch, e.Binding)
		case want != nil && (e.Binding == nil || e.Binding.OrgID != want.OrgID || e.Binding.Feature != want.Feature || e.Binding.Split != want.Split):
			t.Errorf("%s binding = %+v, want %+v", branch, e.Binding, want)
		}
	}
	if e := moEntry(t, res, "feat/one"); e.Class != MergeClassReady {
		t.Errorf("feat/one of the disbanded life = %+v, want ready", e)
	}
	if e := moEntry(t, res, "feat/orphan"); e.Class != MergeClassReady {
		t.Errorf("feat/orphan = %+v, want ready from its gate record alone", e)
	}
	if len(res.Entries) != 4 {
		t.Fatalf("entries = %v, want the four branches", moOrder(res))
	}
}

// TestMergeOrder_SameBranchBoundTwiceShowsTheLatestBinding covers C3's
// supplement: a branch two bindings name is one entry, with the latest
// binding.
func TestMergeOrder_SameBranchBoundTwiceShowsTheLatestBinding(t *testing.T) {
	fx := newMOFixture(t)
	fx.add(t,
		moBindTo("org-a", "split-1", "auth", "feat/auth"), moDisbanded("org-a"),
		ManifestEvent{TS: "2026-10-14T08:59:00Z", OrgID: "org-b", Event: EventScopeReserved, Worktree: "/wt/auth2",
			Details: moBindTo("org-b", "split-2", "auth2", "feat/auth").Details},
	)
	fx.exec.lsOut = moListing("main", "main", moMain)
	res := fx.run(t, false)
	if len(res.Entries) != 1 {
		t.Fatalf("entries = %v, want one for feat/auth", moOrder(res))
	}
	want := MergeOrderBinding{OrgID: "org-b", Feature: "auth2", Split: "split-2", TS: "2026-10-14T08:59:00Z"}
	if b := res.Entries[0].Binding; b == nil || *b != want {
		t.Fatalf("binding = %+v, want the latest %+v", b, want)
	}
}

// TestMergeOrder_DryRunSeatAndIncompleteRecordsDoNotCount covers AC4's
// first sentence: dry-run bindings and gate records, gate records a seat
// carries, plain reservations, and incomplete bindings (no worktree, or a
// key given twice) are not read.
func TestMergeOrder_DryRunSeatAndIncompleteRecordsDoNotCount(t *testing.T) {
	dry := func(ev ManifestEvent) ManifestEvent {
		ev.DryRun = true
		return ev
	}
	noWorktree := testFeature("split-1", "half", "0123456789ab")
	noWorktree.Worktree = ""
	seatPass := moPassed("org-a", "feat/auth", moHeadA, moMain)
	seatPass.SeatID = LeaderIdentity
	fx := newMOFixture(t)
	fx.add(t,
		moBind("org-a", "auth"), moFailed("org-a", "feat/auth", moHeadA, moMain, GateReasonTestFailed),
		dry(moPassed("org-a", "feat/auth", moHeadA, moMain)),
		seatPass,
		dry(moBind("org-d", "dry")),
		dry(moPassed("org-d", "feat/dryonly", moHeadA, moMain)),
		reserveEvent("org-p", "docs/"),
		boundReserveEvent("org-h", noWorktree, "internal/half/"),
		ManifestEvent{OrgID: "org-t", Event: EventScopeReserved, Worktree: "/wt/twice",
			Details: "paths=internal/twice/ split=split-1 feature=twice digest=0123456789ab branch=feat/twice branch=feat/again"},
	)
	fx.exec.lsOut = moListing("main", "main", moMain, "feat/auth", moHeadA)
	res := fx.run(t, false)
	if got := moOrder(res); !slices.Equal(got, []string{"1 failed feat/auth"}) {
		t.Fatalf("entries = %q, want only feat/auth, still failed", got)
	}
}

// TestMergeOrder_RecordsWithoutAnOrgIDAreNotRead: a complete binding and a
// readable gate record whose org_id is empty name no branch, as reserve.go
// does not read such a binding either.
func TestMergeOrder_RecordsWithoutAnOrgIDAreNotRead(t *testing.T) {
	noOrgBind := moBind("org-z", "noorg")
	noOrgBind.OrgID = ""
	noOrgGate := moPassed("", "feat/gateonly", moHeadA, moMain)
	fx := newMOFixture(t)
	fx.add(t, moBind("org-a", "auth"), noOrgBind, noOrgGate)
	fx.exec.lsOut = moListing("main", "main", moMain, "feat/gateonly", moHeadA)
	res := fx.run(t, false)
	if got := moOrder(res); !slices.Equal(got, []string{"1 no_gate feat/auth"}) {
		t.Fatalf("order = %q, want only feat/auth", got)
	}
}

// TestMergeOrder_OrdersByClassThenGatePosition covers AC6: the classes come
// in their order, ready ones by the position of their gate record (not their
// binding), branches with no gate record by their binding, and merged and
// closed only with All, merged first.
func TestMergeOrder_OrdersByClassThenGatePosition(t *testing.T) {
	fx := newMOFixture(t)
	fx.add(t,
		moBind("org-r2", "r2"), moBind("org-r1", "r1"), moBind("org-ng1", "ng1"),
		moPassed("org-r1", "feat/r1", moHeadA, moMain),
		moBind("org-rb", "rb"), moGate("org-rb", "feat/rb", moHeadA, moMain, "refs/remotes/origin/develop", ""),
		moBind("org-wb", "wb"), moPassed("org-wb", "feat/wb", moHeadA, moMain),
		moBind("org-rg", "rg"), moPassed("org-rg", "feat/rg", moHeadA, moMain),
		moBind("org-un", "un"), moGarbled("org-un", EventGatePassed),
		moBind("org-fa", "fa"), moFailed("org-fa", "feat/fa", moHeadA, moMain, GateReasonHeadMoved),
		moBind("org-nr", "nr"), moPassed("org-nr", "feat/nr", moHeadA, moMain),
		moPassed("org-r2", "feat/r2", moHeadA, moMain),
		moBind("org-ng2", "ng2"),
		moBind("org-cl", "cl"), moPassed("org-cl", "feat/cl", moHeadA, moMain),
		moBind("org-me", "me"), moPassed("org-me", "feat/me", moHeadA, moMain),
	)
	pairs := []string{"main", moMain}
	for _, b := range []string{"r1", "r2", "rb", "wb", "un", "fa", "ng1", "ng2", "cl", "me"} {
		pairs = append(pairs, "feat/"+b, moHeadA)
	}
	fx.exec.lsOut = moListing("main", append(pairs, "feat/rg", moHeadB)...)
	fx.exec.setPR("feat/wb", moPRs(moPR(3, "OPEN", "develop")))
	fx.exec.setPR("feat/cl", moPRs(moPR(4, "CLOSED", "main")))
	fx.exec.setPR("feat/me", moPRs(moPR(5, "MERGED", "main")))

	shown := []string{
		"1 ready feat/r1", "2 ready feat/r2", "3 rebase feat/rb", "4 wrong_base feat/wb", "5 regate feat/rg",
		"6 unreadable feat/un", "7 failed feat/fa", "8 no_remote feat/nr", "9 no_gate feat/ng1", "10 no_gate feat/ng2",
	}
	res := fx.run(t, false)
	if got := moOrder(res); !slices.Equal(got, shown) {
		t.Fatalf("order =\n%s\nwant\n%s", strings.Join(got, "\n"), strings.Join(shown, "\n"))
	}
	if moNoteWith(res, "2 branch(es) whose PR is merged or closed are not shown") == "" {
		t.Errorf("notes = %q, want the hidden merged and closed counted", res.Notes)
	}
	if moNoteWith(res, "2 branches are ready, but merging one moves origin/main") == "" {
		t.Errorf("notes = %q, want the note that merging one ready branch makes the other rebase", res.Notes)
	}

	all := fx.run(t, true)
	if got, want := moOrder(all), append(slices.Clone(shown), "11 merged feat/me", "12 closed feat/cl"); !slices.Equal(got, want) {
		t.Fatalf("order with All =\n%s\nwant\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
	}
	if moNoteWith(all, "are not shown") != "" {
		t.Errorf("notes with All = %q, want no hidden count", all.Notes)
	}
	if e := moEntry(t, all, "feat/me"); e.PR == nil || e.PR.Number != 5 || !strings.Contains(e.Detail, "PR #5 is merged") {
		t.Errorf("feat/me = %+v, want PR #5 merged", e)
	}
}

// TestMergeOrder_MergedOrClosedComesFirst covers the first rule of the
// classification: a merged or closed PR hides the branch whatever its gate
// records say, a merged one that origin deleted included.
func TestMergeOrder_MergedOrClosedComesFirst(t *testing.T) {
	fx := newMOFixture(t)
	fx.add(t,
		moBind("org-a", "gone"), moPassed("org-a", "feat/gone", moHeadA, moMain),
		moBind("org-b", "bad"), moGarbled("org-b", EventGatePassed),
		moBind("org-c", "nogate"),
	)
	fx.exec.lsOut = moListing("main", "main", moMain2)
	fx.exec.setPR("feat/gone", moPRs(moPR(1, "MERGED", "main")))
	fx.exec.setPR("feat/bad", moPRs(moPR(2, "CLOSED", "main")))
	fx.exec.setPR("feat/nogate", moPRs(moPR(3, "MERGED", "main")))
	if res := fx.run(t, false); len(res.Entries) != 0 || moNoteWith(res, "3 branch(es)") == "" {
		t.Fatalf("entries %v, notes %q; want all three hidden and counted", moOrder(res), res.Notes)
	}
	res := fx.run(t, true)
	want := []string{"1 merged feat/gone", "2 merged feat/nogate", "3 closed feat/bad"}
	if got := moOrder(res); !slices.Equal(got, want) {
		t.Fatalf("order = %q, want %q", got, want)
	}
}

// TestMergeOrder_PicksTheHighestSameRepoPR covers C5: of a branch's PRs, the
// one with the highest number that is not from a fork, so a fork's merged PR
// of the same branch name neither hides the branch nor stands for its PR.
func TestMergeOrder_PicksTheHighestSameRepoPR(t *testing.T) {
	fork := moPR(9, "MERGED", "main")
	fork.IsCrossRepository = true
	fx := newMOFixture(t)
	fx.add(t,
		moBind("org-a", "auth"), moPassed("org-a", "feat/auth", moHeadA, moMain),
		moBind("org-b", "forked"), moPassed("org-b", "feat/forked", moHeadA, moMain),
	)
	fx.exec.lsOut = moListing("main", "main", moMain, "feat/auth", moHeadA, "feat/forked", moHeadA)
	fx.exec.setPR("feat/auth", moPRs(moPR(3, "CLOSED", "main"), fork, moPR(7, "OPEN", "main"), moPR(5, "CLOSED", "main")))
	fx.exec.setPR("feat/forked", moPRs(fork))
	res := fx.run(t, false)
	want := MergeOrderPR{Number: 7, State: "OPEN", URL: "https://github.com/o/r/pull/7", HeadRefOid: moHeadA, BaseRefName: "main"}
	if e := moEntry(t, res, "feat/auth"); e.Class != MergeClassReady || e.PR == nil || *e.PR != want {
		t.Fatalf("feat/auth = %+v (PR %+v), want ready with %+v", e, e.PR, want)
	}
	if e := moEntry(t, res, "feat/forked"); e.Class != MergeClassReady || e.PR != nil {
		t.Fatalf("feat/forked = %+v, want ready with no PR (only a fork's)", e)
	}
}

// moRepoPR is moPR whose url names owner/repo.
func moRepoPR(n int, state, base, owner, repo string) ghPullRequest {
	pr := moPR(n, state, base)
	pr.URL = fmt.Sprintf("https://github.com/%s/%s/pull/%d", owner, repo, n)
	return pr
}

// moRepoCheckFixture is three branches, feat/a, feat/b and feat/c, each
// bound and gated on origin's tips, so each is ready on origin's evidence,
// with origin's URL origin.
func moRepoCheckFixture(t *testing.T, origin string) *moFixture {
	t.Helper()
	fx := newMOFixture(t)
	for _, b := range []string{"a", "b", "c"} {
		fx.add(t, moBind("org-"+b, b), moPassed("org-"+b, "feat/"+b, moHeadA, moMain))
	}
	fx.exec.lsOut = moListing("main", "main", moMain, "feat/a", moHeadA, "feat/b", moHeadA, "feat/c", moHeadA)
	fx.exec.origin = origin
	return fx
}

// moGHBranches is the --head value of each gh call fx's exec saw.
func moGHBranches(fx *moFixture) []string {
	var out []string
	for _, c := range fx.exec.callsOf("gh") {
		out = append(out, c.args[3])
	}
	return out
}

// TestMergeOrder_PRIsUsedWhenItsURLNamesOriginsRepository covers the
// repository check's match: origin's owner/repo read from an scp-like, an
// https, an ssh URL with a trailing ".git/", an ssh host alias with another
// case, and the other URL shapes all match a PR url of github.com/acme/app,
// so the PRs are used: feat/a's open PR stays on its ready row and feat/b's
// merged PR hides it. `git remote get-url origin` runs once, in p.Dir with
// GIT_TERMINAL_PROMPT=0, however many PRs gh returns.
func TestMergeOrder_PRIsUsedWhenItsURLNamesOriginsRepository(t *testing.T) {
	for _, origin := range []string{
		"git@github.com:acme/app.git",
		"https://github.com/acme/app",
		"ssh://git@github.com/acme/app.git/",
		"git@github.com.emu:Acme/App.git",
		"github.com:acme/app",
		"http://github.com/acme/app.git",
		"git://github.com/acme/app.git",
		"ssh://git@github.com:22/acme/app.git",
		"https://user@github.example.com/team/acme/app.git",
	} {
		t.Run(origin, func(t *testing.T) {
			fx := moRepoCheckFixture(t, origin)
			fx.exec.setPR("feat/a", moPRs(moRepoPR(7, "OPEN", "main", "acme", "app")))
			fx.exec.setPR("feat/b", moPRs(moRepoPR(8, "MERGED", "main", "acme", "app")))
			res := fx.run(t, false)
			if got, want := moOrder(res), []string{"1 ready feat/a", "2 ready feat/c"}; !slices.Equal(got, want) {
				t.Fatalf("order = %q, want %q (feat/b hidden as merged)", got, want)
			}
			if e := moEntry(t, res, "feat/a"); e.PR == nil || e.PR.Number != 7 {
				t.Errorf("feat/a PR = %+v, want #7", e.PR)
			}
			if moNoteWith(res, "gh reads") != "" || moNoteWith(res, "no PR is used") != "" {
				t.Errorf("notes = %q, want no repository note", res.Notes)
			}
			var getURL []moCall
			for _, c := range fx.exec.callsOf("git") {
				if c.args[0] == "remote" {
					getURL = append(getURL, c)
				}
			}
			if len(getURL) != 1 || !slices.Equal(getURL[0].args, []string{"remote", "get-url", "origin"}) || getURL[0].dir != "/repo" ||
				!slices.Equal(getURL[0].env, []string{"GIT_TERMINAL_PROMPT=0"}) {
				t.Errorf("get-url calls = %+v, want one git remote get-url origin in /repo with GIT_TERMINAL_PROMPT=0", getURL)
			}
			if got := moGHBranches(fx); !slices.Equal(got, []string{"feat/a", "feat/b", "feat/c"}) {
				t.Errorf("gh asked about %q, want every branch", got)
			}
		})
	}
}

// TestMergeOrder_PRFromAnotherRepositoryThanOriginsIsLeftOut covers the
// cross-review finding: gh reads the repository `gh repo set-default` (or the
// remotes' names) picks, which in a fork checkout is upstream, not origin. A
// PR whose url names another owner/repo than origin's, or that names none,
// leaves every PR out: a merged PR of the same branch name in upstream does
// not hide origin's ready branch, a PR into another branch does not make it
// wrong_base, and a PR of origin's found before is dropped too. One note says
// why (for another repository, the two fixes that
// TestMergeOrder_RepositoryNoteNamesTheFixForARenameAndForAnotherRepository
// pins), gh is not asked about the branches after it, and the run succeeds
// with gh available.
func TestMergeOrder_PRFromAnotherRepositoryThanOriginsIsLeftOut(t *testing.T) {
	upstream := func(n int, state, base string) string {
		return moPRs(moRepoPR(n, state, base, "upstream-org", "app"))
	}
	unparsable := moPR(7, "MERGED", "main")
	unparsable.URL = "https://github.example.com/pull/7"
	for _, tc := range []struct {
		name    string
		prs     map[string]string
		ghCalls []string
		note    []string
	}{
		{"a merged PR of upstream does not hide origin's ready branch", map[string]string{"feat/a": upstream(7, "MERGED", "main")},
			[]string{"feat/a"}, []string{"gh reads upstream-org/app, not origin's acme/app: no PR is used", "git remote set-url origin",
				"run gh repo set-default acme/app"}},
		{"a PR of upstream into another branch is not wrong_base", map[string]string{"feat/a": upstream(7, "OPEN", "develop")},
			[]string{"feat/a"}, []string{"gh reads upstream-org/app, not origin's acme/app"}},
		{"a PR of the same name under another owner", map[string]string{"feat/a": moPRs(moRepoPR(7, "CLOSED", "main", "someone", "app"))},
			[]string{"feat/a"}, []string{"gh reads someone/app, not origin's acme/app"}},
		{"a PR of another repository after one of origin's drops both", map[string]string{
			"feat/a": moPRs(moRepoPR(7, "OPEN", "main", "acme", "app")), "feat/b": upstream(8, "MERGED", "main")},
			[]string{"feat/a", "feat/b"}, []string{"gh reads upstream-org/app, not origin's acme/app"}},
		{"a PR url that names no repository", map[string]string{"feat/a": moPRs(unparsable)},
			[]string{"feat/a"}, []string{`gh returned PR #7 for feat/a with the url "https://github.example.com/pull/7", which names no ` +
				"repository to check against origin's acme/app: no PR is used"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fx := moRepoCheckFixture(t, "git@github.com:acme/app.git")
			for branch, out := range tc.prs {
				fx.exec.setPR(branch, out)
			}
			res := fx.run(t, false)
			if got, want := moOrder(res), []string{"1 ready feat/a", "2 ready feat/b", "3 ready feat/c"}; !slices.Equal(got, want) {
				t.Fatalf("order = %q, want %q on origin's evidence alone", got, want)
			}
			for _, e := range res.Entries {
				if e.PR != nil {
					t.Errorf("%s PR = %+v, want none", e.Branch, e.PR)
				}
			}
			if got := moGHBranches(fx); !slices.Equal(got, tc.ghCalls) {
				t.Errorf("gh asked about %q, want %q and no branch after", got, tc.ghCalls)
			}
			var repoNotes []string
			for _, n := range res.Notes {
				if strings.Contains(n, "no PR is used") {
					repoNotes = append(repoNotes, n)
				}
			}
			if len(repoNotes) != 1 {
				t.Fatalf("notes = %q, want one note that no PR is used", res.Notes)
			}
			for _, want := range append(tc.note, "merged and closed PRs are not told apart") {
				if !strings.Contains(repoNotes[0], want) {
					t.Errorf("note %q does not say %q", repoNotes[0], want)
				}
			}
			if !res.GHAvailable {
				t.Errorf("GHAvailable = false, want true: gh was there")
			}
		})
	}
}

// TestMergeOrder_RepositoryNoteNamesTheFixForARenameAndForAnotherRepository
// covers cross-review cycle 3's finding of org-merge-order (triage:
// docs/reports/cross-review-triage-org-merge-order.md): after origin's
// repository is renamed on GitHub, git still reaches it at origin's old URL
// through GitHub's redirect and gh returns PR urls with the new name, so the
// check leaves the PRs out, and gh repo set-default with origin's old name
// would not help. merge-order cannot tell a rename from another repository,
// so the note names both fixes, git remote set-url origin for a rename and gh
// repo set-default with origin's owner/repo otherwise, whichever case it is.
// The PRs are still left out and gh is not asked again.
func TestMergeOrder_RepositoryNoteNamesTheFixForARenameAndForAnotherRepository(t *testing.T) {
	for _, tc := range []struct {
		name, owner, repo string
	}{
		{"origin's repository renamed on GitHub", "acme", "app-v2"},
		{"gh reads another repository", "upstream-org", "app"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fx := moRepoCheckFixture(t, "https://github.com/acme/app.git")
			fx.exec.setPR("feat/a", moPRs(moRepoPR(7, "MERGED", "main", tc.owner, tc.repo)))
			res := fx.run(t, false)
			if got, want := moOrder(res), []string{"1 ready feat/a", "2 ready feat/b", "3 ready feat/c"}; !slices.Equal(got, want) {
				t.Fatalf("order = %q, want %q with no PR used", got, want)
			}
			if e := moEntry(t, res, "feat/a"); e.PR != nil {
				t.Errorf("feat/a PR = %+v, want none", e.PR)
			}
			ghRepo := tc.owner + "/" + tc.repo
			want := "gh reads " + ghRepo + ", not origin's acme/app: no PR is used, so " + mergeOrderNoPRs + "; if acme/app was renamed to " +
				ghRepo + ", point origin at it (git remote set-url origin <its URL>); otherwise run gh repo set-default acme/app in this repository"
			if note := moNoteWith(res, "no PR is used"); note != want {
				t.Fatalf("note = %q,\nwant %q", note, want)
			}
			if got := moGHBranches(fx); !slices.Equal(got, []string{"feat/a"}) {
				t.Errorf("gh asked about %q, want feat/a only", got)
			}
		})
	}
}

// TestMergeOrder_LocalPathOriginSkipsTheRepositoryCheck: an origin that is a
// local path (absolute, relative, file://) or a URL of another scheme names
// no GitHub repository to check against, so gh's PRs are used as they are,
// a url that names no repository included.
func TestMergeOrder_LocalPathOriginSkipsTheRepositoryCheck(t *testing.T) {
	unparsable := moPR(8, "MERGED", "main")
	unparsable.URL = "https://example.test/pull/8"
	for _, origin := range []string{"/srv/git/app.git", "../origin.git", "origin.git", "./a:b/app.git", "file:///srv/git/acme/app.git",
		"ftp://github.com/acme/app.git"} {
		t.Run(origin, func(t *testing.T) {
			fx := moRepoCheckFixture(t, origin)
			fx.exec.setPR("feat/a", moPRs(moRepoPR(7, "OPEN", "develop", "upstream-org", "app")))
			fx.exec.setPR("feat/b", moPRs(unparsable))
			res := fx.run(t, false)
			if got, want := moOrder(res), []string{"1 ready feat/c", "2 wrong_base feat/a"}; !slices.Equal(got, want) {
				t.Fatalf("order = %q, want %q (feat/b hidden as merged)", got, want)
			}
			if moNoteWith(res, "no PR is used") != "" {
				t.Errorf("notes = %q, want no repository note", res.Notes)
			}
		})
	}
}

// TestMergeOrder_ReadsOwnerAndRepoFromOriginAndPRURLs covers the two URL
// readers: origin's owner/repo only from a URL with a host, and a PR's only
// from https://<host>/<owner>/<repo>/pull/<number>.
func TestMergeOrder_ReadsOwnerAndRepoFromOriginAndPRURLs(t *testing.T) {
	for in, want := range map[string]string{
		"git@github.com:acme/app.git":               "acme/app",
		"ssh://git@github.com/acme/app.git/":        "acme/app",
		"https://github.com/acme/app.git.git":       "acme/app.git",
		"git@github.com:/acme/app":                  "acme/app",
		"https://github.com/acme":                   "",
		"https://github.com/acme/":                  "",
		"https://github.com//app":                   "",
		"https:///acme/app":                         "",
		":acme/app":                                 "",
		"git@:acme/app":                             "",
		"/abs/acme/app.git":                         "",
		"rel/acme/app":                              "",
		"file:///acme/app":                          "",
		"":                                          "",
		"git@github.com:app.git":                    "",
		"https://github.com/acme/app\x7f/../x":      "",
		"HTTPS://github.com/acme/app":               "acme/app",
		"ssh://[::1]:22/acme/app.git":               "acme/app",
		"git+ssh://git@github.com/acme/app.git":     "",
		"https://github.com/acme/app?tab=readme#go": "acme/app",
	} {
		repo, ok := mergeRepoFromOriginURL(in)
		if got := map[bool]string{true: repo.String(), false: ""}[ok]; got != want {
			t.Errorf("origin %q = %q (%v), want %q", in, repo, ok, want)
		}
	}
	for in, want := range map[string]string{
		"https://github.com/acme/app/pull/7":       "acme/app",
		"https://ghe.example.com/Acme/App/pull/12": "Acme/App",
		"http://github.com/acme/app/pull/7":        "",
		"https://github.com/acme/app/pull/x":       "",
		"https://github.com/acme/app/pull/0":       "",
		"https://github.com/acme/app/issues/7":     "",
		"https://github.com/acme/app/pull/7/files": "",
		"https://github.com/pull/7":                "",
		"https:///acme/app/pull/7":                 "",
		"https://example.test/pull/1\tINJECT":      "",
		"":                                         "",
	} {
		repo, ok := mergeRepoFromPRURL(in)
		if got := map[bool]string{true: repo.String(), false: ""}[ok]; got != want {
			t.Errorf("PR url %q = %q (%v), want %q", in, repo, ok, want)
		}
	}
}

// TestMergeOrder_GetURLRunsOnlyOnceGHReturnsAPR: `git remote get-url origin`
// does not run when gh returns no PR for any branch (none at all, only a
// fork's, or a failure), nor when gh is missing; only the ls-remote runs.
func TestMergeOrder_GetURLRunsOnlyOnceGHReturnsAPR(t *testing.T) {
	fork := moRepoPR(9, "MERGED", "main", "fork", "app")
	fork.IsCrossRepository = true
	for _, tc := range []struct {
		name  string
		setup func(e *moExec)
	}{
		{"gh returns no PR", func(*moExec) {}},
		{"gh returns only a fork's PR", func(e *moExec) { e.setPR("feat/a", moPRs(fork)) }},
		{"gh fails", func(e *moExec) {
			e.ghErr = map[string]error{"feat/a": errors.New("exit status 1"), "feat/b": errors.New("exit status 1"), "feat/c": errors.New("exit status 4")}
		}},
		{"gh is missing", func(e *moExec) { e.ghMissing = true }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fx := moRepoCheckFixture(t, "git@github.com:acme/app.git")
			tc.setup(fx.exec)
			fx.run(t, false)
			if git := fx.exec.callsOf("git"); len(git) != 1 || git[0].args[0] != "ls-remote" {
				t.Fatalf("git calls = %+v, want the ls-remote only", git)
			}
		})
	}
}

// TestMergeOrder_OriginURLThatCannotBeReadLeavesPRsOut: a `git remote get-url
// origin` that fails or times out leaves every PR out with one note naming
// the failure, and gh is not asked again; ctx done while it runs is an error.
func TestMergeOrder_OriginURLThatCannotBeReadLeavesPRsOut(t *testing.T) {
	fx := moRepoCheckFixture(t, "")
	fx.exec.originErr = errors.New("exit status 2: error: No such remote 'origin'")
	fx.exec.setPR("feat/a", moPRs(moRepoPR(7, "MERGED", "main", "acme", "app")))
	res := fx.run(t, false)
	if got, want := moOrder(res), []string{"1 ready feat/a", "2 ready feat/b", "3 ready feat/c"}; !slices.Equal(got, want) {
		t.Fatalf("order = %q, want %q with no PR", got, want)
	}
	if note := moNoteWith(res, "git remote get-url origin failed (exit status 2: error: No such remote 'origin')"); !strings.Contains(note, "no PR is used") {
		t.Errorf("notes = %q, want the get-url failure noted with no PR used", res.Notes)
	}
	if got := moGHBranches(fx); !slices.Equal(got, []string{"feat/a"}) {
		t.Errorf("gh asked about %q, want feat/a only", got)
	}

	timeout := moRepoCheckFixture(t, "")
	timeout.exec.setPR("feat/a", moPRs(moRepoPR(7, "MERGED", "main", "acme", "app")))
	p := timeout.params(false)
	p.GHTimeout = 20 * time.Millisecond
	p.Exec = func(c context.Context, dir string, env []string, name string, args ...string) (string, error) {
		if name == "git" && args[0] == "remote" {
			<-c.Done()
			return "", c.Err()
		}
		return timeout.exec.run(c, dir, env, name, args...)
	}
	res, err := timeout.o.MergeOrder(context.Background(), p)
	if err != nil || moNoteWith(res, "git remote get-url origin failed (timed out after 20ms") == "" {
		t.Fatalf("MergeOrder = %q, %v; want the get-url timeout noted", res.Notes, err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	cancelled := moRepoCheckFixture(t, "")
	cancelled.exec.setPR("feat/a", moPRs(moRepoPR(7, "MERGED", "main", "acme", "app")))
	p = cancelled.params(false)
	p.Exec = func(c context.Context, dir string, env []string, name string, args ...string) (string, error) {
		if name == "git" && args[0] == "remote" {
			cancel()
			return "", c.Err()
		}
		return cancelled.exec.run(c, dir, env, name, args...)
	}
	if res, err := cancelled.o.MergeOrder(ctx, p); err == nil || !strings.Contains(err.Error(), "interrupted") ||
		!strings.Contains(err.Error(), "while git remote get-url origin ran") || len(res.Entries) != 0 {
		t.Fatalf("MergeOrder = %+v, %v; want an interrupted error and nothing classified", res, err)
	}
}

// TestMergeOrder_GHMissingNotesAndIsNotRunAgain covers AC5's gh half: with
// no gh, PRs stay empty, one note says merged, closed and PR bases are not
// told apart, gh is tried once only, and the run succeeds.
func TestMergeOrder_GHMissingNotesAndIsNotRunAgain(t *testing.T) {
	fx := newMOFixture(t)
	fx.add(t,
		moBind("org-a", "a"), moPassed("org-a", "feat/a", moHeadA, moMain),
		moBind("org-b", "b"), moBind("org-c", "c"),
	)
	fx.exec.lsOut = moListing("main", "main", moMain, "feat/a", moHeadA)
	fx.exec.ghMissing = true
	res := fx.run(t, false)
	if calls := fx.exec.callsOf("gh"); len(calls) != 1 {
		t.Fatalf("gh ran %d times, want once", len(calls))
	}
	if res.GHAvailable {
		t.Errorf("GHAvailable = true, want false")
	}
	if moNoteWith(res, "gh was not found") == "" || moNoteWith(res, "not told apart") == "" {
		t.Errorf("notes = %q, want the gh-missing note", res.Notes)
	}
	for _, e := range res.Entries {
		if e.PR != nil {
			t.Errorf("%s PR = %+v, want none", e.Branch, e.PR)
		}
	}
	if got, want := moOrder(res), []string{"1 ready feat/a", "2 no_gate feat/b", "3 no_gate feat/c"}; !slices.Equal(got, want) {
		t.Errorf("order = %q, want %q", got, want)
	}
}

// TestMergeOrder_GHFailureForOneBranchLeavesItsPROut covers AC5's failure
// half: a gh that exits non-zero, times out, or prints something that is not
// a PR list for one branch leaves that branch's PR out with a note naming
// it, and the other branches keep theirs.
func TestMergeOrder_GHFailureForOneBranchLeavesItsPROut(t *testing.T) {
	fx := newMOFixture(t)
	for _, b := range []string{"a", "b", "c", "d"} {
		fx.add(t, moBind("org-"+b, b), moPassed("org-"+b, "feat/"+b, moHeadA, moMain))
	}
	fx.exec.lsOut = moListing("main", "main", moMain, "feat/a", moHeadA, "feat/b", moHeadA, "feat/c", moHeadA, "feat/d", moHeadA)
	fx.exec.ghErr = map[string]error{"feat/a": errors.New("exit status 1: HTTP 401: Bad credentials")}
	fx.exec.setPR("feat/b", "not json")
	fx.exec.ghBlock = map[string]bool{"feat/c": true}
	fx.exec.setPR("feat/d", moPRs(moPR(4, "MERGED", "main")))
	p := fx.params(false)
	p.GHTimeout = 20 * time.Millisecond
	res, err := fx.o.MergeOrder(context.Background(), p)
	if err != nil {
		t.Fatalf("MergeOrder: %v, want a gh failure not to fail the run", err)
	}
	if !res.GHAvailable || len(fx.exec.callsOf("gh")) != 4 {
		t.Fatalf("GHAvailable %v after %d gh calls, want true after 4", res.GHAvailable, len(fx.exec.callsOf("gh")))
	}
	for branch, why := range map[string]string{
		"feat/a": "exit status 1: HTTP 401: Bad credentials",
		"feat/b": "its output is not a JSON list of PRs",
		"feat/c": "timed out after 20ms",
	} {
		if moNoteWith(res, "gh pr list --head "+branch+" failed ("+why) == "" {
			t.Errorf("notes = %q, want one naming %s and %q", res.Notes, branch, why)
		}
		if e := moEntry(t, res, branch); e.PR != nil || e.Class != MergeClassReady {
			t.Errorf("%s = %+v, want ready with no PR", branch, e)
		}
	}
	if got := moOrder(res); len(got) != 3 || moNoteWith(res, "1 branch(es) whose PR is merged or closed") == "" {
		t.Errorf("order %q, notes %q; want feat/d hidden as merged", got, res.Notes)
	}
}

// TestMergeOrder_GHFailingTheSameWayForSeveralBranchesIsOneNote covers
// self-review L2: gh failing with the same error for several branches (an
// unauthenticated gh fails for all) is one note giving their count and naming
// at most five, a reason met once keeps the one-branch note, and gh is still
// asked about every branch.
func TestMergeOrder_GHFailingTheSameWayForSeveralBranchesIsOneNote(t *testing.T) {
	const auth, network = "exit status 1: HTTP 401: Bad credentials", "exit status 1: error connecting to api.github.com"
	fx := newMOFixture(t)
	fx.exec.lsOut = moListing("main", "main", moMain)
	fx.exec.ghErr = map[string]error{}
	var branches []string
	for i := 1; i <= 10; i++ {
		b := fmt.Sprintf("feat/b%02d", i)
		branches = append(branches, b)
		fx.add(t, moBind(fmt.Sprintf("org-%02d", i), fmt.Sprintf("b%02d", i)))
		switch {
		case i <= 7:
			fx.exec.ghErr[b] = errors.New(auth)
		case i <= 9:
			fx.exec.ghErr[b] = errors.New(network)
		default:
			fx.exec.ghErr[b] = errors.New("exit status 2: something else")
		}
	}
	res := fx.run(t, false)
	if got := moGHBranches(fx); !slices.Equal(got, branches) {
		t.Fatalf("gh asked about %q, want every branch", got)
	}
	want := []string{
		"gh pr list failed for 7 branches (feat/b01, feat/b02, feat/b03, feat/b04, feat/b05 and 2 more) with the same error (" + auth +
			"): their PRs are unknown, so merged, closed and the PR base are not told apart for them",
		"gh pr list failed for 2 branches (feat/b08, feat/b09) with the same error (" + network + "): their PRs are unknown",
		"gh pr list --head feat/b10 failed (exit status 2: something else): its PR is unknown",
	}
	var ghNotes []string
	for _, n := range res.Notes {
		if strings.HasPrefix(n, "gh pr list") {
			ghNotes = append(ghNotes, n)
		}
	}
	if len(ghNotes) != len(want) {
		t.Fatalf("gh notes = %q, want %d", ghNotes, len(want))
	}
	for i, w := range want {
		if !strings.HasPrefix(ghNotes[i], w) {
			t.Errorf("gh note %d = %q, want it to start with %q", i+1, ghNotes[i], w)
		}
	}
}

// moForkPRs is n merged PRs from forks of the branch name, newest first as
// gh lists them, numbered down to first. A fork's PR has the url of the
// repository it merges into, here acme/app.
func moForkPRs(first, n int) []ghPullRequest {
	prs := make([]ghPullRequest, 0, n)
	for i := range n {
		pr := moRepoPR(first+n-1-i, "MERGED", "main", "acme", "app")
		pr.IsCrossRepository = true
		prs = append(prs, pr)
	}
	return prs
}

// moLimitNotes are the notes of res that say gh returned its limit.
func moLimitNotes(res MergeOrderResult) []string {
	var out []string
	for _, n := range res.Notes {
		if strings.Contains(n, "its limit") {
			out = append(out, n)
		}
	}
	return out
}

// TestMergeOrder_GHLimitFullOfForksLeavesThePROut covers cross-review cycle 2
// of org-merge-order: gh returns a branch's PRs newest first up to --limit,
// so newer fork PRs of the same branch name can fill the limit and push the
// branch's own PR out of the result. When gh returns mergeOrderGHLimit PRs,
// all from forks, the branch gets no PR and is classified on the gate and
// origin alone, and a note says why (one note for several branches); one PR
// fewer is no PR with no note; and a same-repository PR among the limit's
// PRs is used, its repository checked as any other.
func TestMergeOrder_GHLimitFullOfForksLeavesThePROut(t *testing.T) {
	limit := strconv.Itoa(mergeOrderGHLimit)
	full := moForkPRs(100, mergeOrderGHLimit)
	ownAmongFull := func(own ghPullRequest) string {
		prs := slices.Clone(full)
		prs[len(prs)/2] = own
		return moPRs(prs...)
	}
	noGetURL := func(t *testing.T, fx *moFixture) {
		t.Helper()
		if git := fx.exec.callsOf("git"); len(git) != 1 || git[0].args[0] != "ls-remote" {
			t.Errorf("git calls = %+v, want the ls-remote only: no PR was returned to check", git)
		}
	}

	t.Run("one branch full of forks", func(t *testing.T) {
		fx := moRepoCheckFixture(t, "git@github.com:acme/app.git")
		fx.exec.setPR("feat/a", moPRs(full...))
		fx.exec.setPR("feat/b", moPRs(moForkPRs(100, mergeOrderGHLimit-1)...))
		res := fx.run(t, false)
		if got, want := moOrder(res), []string{"1 ready feat/a", "2 ready feat/b", "3 ready feat/c"}; !slices.Equal(got, want) {
			t.Fatalf("order = %q, want %q", got, want)
		}
		for _, e := range res.Entries {
			if e.PR != nil {
				t.Errorf("%s PR = %+v, want none: only forks' PRs", e.Branch, e.PR)
			}
		}
		want := "gh returned " + limit + " PRs for feat/a, its limit, all from forks, and the branch's own PR may be past them: " +
			"no PR is used for it, so " + mergeOrderNoPRs
		if got := moLimitNotes(res); !slices.Equal(got, []string{want}) {
			t.Errorf("limit notes = %q, want only %q", got, want)
		}
		if note := moNoteWith(res, "feat/b"); note != "" {
			t.Errorf("note %q names feat/b, want none: gh returned fewer PRs than its limit", note)
		}
		noGetURL(t, fx)
	})

	t.Run("several branches full of forks are one note", func(t *testing.T) {
		fx := moRepoCheckFixture(t, "git@github.com:acme/app.git")
		fx.exec.setPR("feat/a", moPRs(full...))
		fx.exec.setPR("feat/c", moPRs(full...))
		res := fx.run(t, false)
		if got, want := moOrder(res), []string{"1 ready feat/a", "2 ready feat/b", "3 ready feat/c"}; !slices.Equal(got, want) {
			t.Fatalf("order = %q, want %q", got, want)
		}
		want := "gh returned " + limit + " PRs, its limit, for each of 2 branches (feat/a, feat/c), all from forks, and their own " +
			"PRs may be past them: no PR is used for them, so " + mergeOrderNoPRs
		if got := moLimitNotes(res); !slices.Equal(got, []string{want}) {
			t.Errorf("limit notes = %q, want only %q", got, want)
		}
		if got := moGHBranches(fx); !slices.Equal(got, []string{"feat/a", "feat/b", "feat/c"}) {
			t.Errorf("gh asked about %q, want every branch", got)
		}
		noGetURL(t, fx)
	})

	t.Run("a same-repository PR among the limit's is used", func(t *testing.T) {
		fx := moRepoCheckFixture(t, "git@github.com:acme/app.git")
		fx.exec.setPR("feat/a", ownAmongFull(moRepoPR(5, "MERGED", "main", "acme", "app")))
		fx.exec.setPR("feat/b", ownAmongFull(moRepoPR(6, "OPEN", "develop", "acme", "app")))
		res := fx.run(t, false)
		if got, want := moOrder(res), []string{"1 ready feat/c", "2 wrong_base feat/b"}; !slices.Equal(got, want) {
			t.Fatalf("order = %q, want %q (feat/a hidden as merged)", got, want)
		}
		if e := moEntry(t, res, "feat/b"); e.PR == nil || e.PR.Number != 6 {
			t.Errorf("feat/b PR = %+v, want #6, the one PR not from a fork", e.PR)
		}
		if got := moLimitNotes(res); len(got) != 0 {
			t.Errorf("limit notes = %q, want none: a PR not from a fork was found", got)
		}
		if moNoteWith(res, "1 branch(es) whose PR is merged or closed") == "" {
			t.Errorf("notes = %q, want feat/a counted as hidden", res.Notes)
		}
	})

	t.Run("the repository check runs on the PR picked among the limit's", func(t *testing.T) {
		fx := moRepoCheckFixture(t, "git@github.com:acme/app.git")
		fx.exec.setPR("feat/a", ownAmongFull(moRepoPR(5, "MERGED", "main", "upstream-org", "app")))
		fx.exec.setPR("feat/b", moPRs(moRepoPR(6, "MERGED", "main", "acme", "app")))
		res := fx.run(t, false)
		if got, want := moOrder(res), []string{"1 ready feat/a", "2 ready feat/b", "3 ready feat/c"}; !slices.Equal(got, want) {
			t.Fatalf("order = %q, want %q with no PR", got, want)
		}
		if moNoteWith(res, "gh reads upstream-org/app, not origin's acme/app: no PR is used") == "" {
			t.Errorf("notes = %q, want the repository note", res.Notes)
		}
		if got := moGHBranches(fx); !slices.Equal(got, []string{"feat/a"}) {
			t.Errorf("gh asked about %q, want feat/a only", got)
		}
	})
}

// TestMergeOrder_ReadsTheLsRemoteListing covers the ls-remote half of C7:
// branch names with several slashes, a branch called HEAD
// (refs/heads/HEAD, which the pattern HEAD matches too) that is not taken
// for origin's HEAD, a symbolic branch's `ref:` line, and refs outside
// refs/heads.
func TestMergeOrder_ReadsTheLsRemoteListing(t *testing.T) {
	listing := strings.Join([]string{
		"ref: refs/heads/HEAD-target\trefs/heads/HEAD",
		moHeadB + "\trefs/heads/HEAD",
		"ref: refs/heads/main\tHEAD",
		moMain + "\tHEAD",
		moMain + "\trefs/heads/alias",
		"ref: refs/heads/main\trefs/heads/alias", // after the SHA, so a parser that stored it would keep it
		moMain + "\trefs/heads/main",
		moHeadA + "\trefs/heads/feat/a/b/c",
		"ref: refs/remotes/origin/main\trefs/remotes/origin/HEAD",
		moHeadB + "\trefs/remotes/origin/HEAD",
		moHeadB + "\trefs/tags/HEAD",
		"",
	}, "\n")
	heads := make(map[string]string)
	if symref := parseLsRemote(listing, heads); symref != "refs/heads/main" {
		t.Fatalf("symref = %q, want refs/heads/main", symref)
	}
	want := map[string]string{"HEAD": moHeadB, "alias": moMain, "main": moMain, "feat/a/b/c": moHeadA}
	if fmt.Sprint(heads) != fmt.Sprint(want) {
		t.Fatalf("heads = %v, want %v", heads, want)
	}

	fx := newMOFixture(t)
	fx.add(t, moBindTo("org-a", "split-1", "abc", "feat/a/b/c"), moPassed("org-a", "feat/a/b/c", moHeadA, moMain))
	fx.exec.lsOut = listing
	res := fx.run(t, false)
	if res.DefaultBranch != "main" || res.DefaultHead != moMain {
		t.Fatalf("default = %s %s, want main %s", res.DefaultBranch, res.DefaultHead, moMain)
	}
	if got := moOrder(res); !slices.Equal(got, []string{"1 ready feat/a/b/c"}) {
		t.Fatalf("order = %q, want feat/a/b/c ready", got)
	}

	crlf := strings.ReplaceAll(moListing("main", "main", moMain, "feat/a/b/c", moHeadA), "\n", "\r\n")
	heads = make(map[string]string)
	if symref := parseLsRemote(crlf, heads); symref != "refs/heads/main" || heads["feat/a/b/c"] != moHeadA {
		t.Fatalf("CRLF listing read as %q, %v", symref, heads)
	}
}

// TestMergeOrder_LsRemoteProblemsClassifyNothing covers AC7: an ls-remote
// that fails, times out, prints nothing, prints no HEAD symref (only a
// branch called HEAD has one), or names a HEAD that is not a listed branch
// is an error that says why, with nothing classified, and so is ctx done.
func TestMergeOrder_LsRemoteProblemsClassifyNothing(t *testing.T) {
	tests := []struct {
		name   string
		out    string
		err    error
		block  bool
		cancel bool
		want   []string
	}{
		{name: "the command fails", err: errors.New("exit status 128: fatal: 'origin' does not appear to be a git repository"),
			want: []string{"failed", "'origin' does not appear to be a git repository", "nothing was classified"}},
		{name: "the command times out", block: true, want: []string{"timed out after 20ms", "nothing was classified"}},
		{name: "empty output", out: "", want: []string{"printed no HEAD symref", "default branch is unknown"}},
		{name: "no HEAD symref, as an old protocol prints", out: moMain + "\tHEAD\n" + moMain + "\trefs/heads/main\n",
			want: []string{"printed no HEAD symref"}},
		{name: "only refs/heads/HEAD is a symref", out: "ref: refs/heads/main\trefs/heads/HEAD\n" + moMain + "\trefs/heads/HEAD\n" +
			moMain + "\trefs/heads/main\n", want: []string{"printed no HEAD symref"}},
		{name: "HEAD names a tag", out: "ref: refs/tags/v1\tHEAD\n" + moMain + "\trefs/heads/main\n",
			want: []string{"origin's HEAD names refs/tags/v1, which is not a branch"}},
		{name: "HEAD names a branch the listing lacks", out: "ref: refs/heads/trunk\tHEAD\n" + moMain + "\trefs/heads/main\n",
			want: []string{"origin's HEAD names refs/heads/trunk, which", "did not list"}},
		{name: "ctx is done", cancel: true, out: moListing("main", "main", moMain), want: []string{"interrupted", "nothing was classified"}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			fx := newMOFixture(t)
			fx.add(t, moBind("org-a", "auth"), moPassed("org-a", "feat/auth", moHeadA, moMain))
			fx.exec.lsOut, fx.exec.lsErr, fx.exec.lsBlock = tc.out, tc.err, tc.block
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			if tc.cancel {
				cancel()
			}
			p := fx.params(false)
			p.LsRemoteTimeout = 20 * time.Millisecond
			res, err := fx.o.MergeOrder(ctx, p)
			if err == nil {
				t.Fatalf("MergeOrder = %+v, want an error", res)
			}
			for _, want := range tc.want {
				if !strings.Contains(err.Error(), want) {
					t.Errorf("error %q does not say %q", err, want)
				}
			}
			if !strings.HasPrefix(err.Error(), "org: merge-order: ") {
				t.Errorf("error %q does not start with org: merge-order:", err)
			}
			if len(res.Entries) != 0 || res.DefaultBranch != "" {
				t.Errorf("result = %+v, want nothing classified", res)
			}
			if calls := fx.exec.callsOf("gh"); len(calls) != 0 {
				t.Errorf("gh ran %d times, want none after the ls-remote failed", len(calls))
			}
		})
	}
}

// TestMergeOrder_InterruptedWhileGHRunsIsAnError: ctx done while gh runs is
// an error, not a gh failure to note.
func TestMergeOrder_InterruptedWhileGHRunsIsAnError(t *testing.T) {
	fx := newMOFixture(t)
	fx.add(t, moBind("org-a", "a"), moBind("org-b", "b"))
	fx.exec.lsOut = moListing("main", "main", moMain)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	p := fx.params(false)
	p.Exec = func(c context.Context, dir string, env []string, name string, args ...string) (string, error) {
		if name == "gh" {
			cancel()
			return "", c.Err()
		}
		return fx.exec.run(c, dir, env, name, args...)
	}
	res, err := fx.o.MergeOrder(ctx, p)
	if err == nil || !strings.Contains(err.Error(), "interrupted") || !strings.Contains(err.Error(), "while gh ran") {
		t.Fatalf("MergeOrder = %+v, %v; want an interrupted error", res, err)
	}
}

// TestMergeOrder_RunsGitAndGHWithTheirArgsDirAndEnv covers C2, C5 and C6:
// one ls-remote with the pattern as one argument and GIT_TERMINAL_PROMPT=0,
// then one gh pr list per branch with --limit 1000 (gh's default is 30) and
// GH_PROMPT_DISABLED=1, all in p.Dir.
func TestMergeOrder_RunsGitAndGHWithTheirArgsDirAndEnv(t *testing.T) {
	fx := newMOFixture(t)
	fx.add(t, moBind("org-a", "a"), moBind("org-b", "b"), moPassed("org-a", "feat/a", moHeadA, moMain))
	fx.exec.lsOut = moListing("main", "main", moMain)
	fx.run(t, false)
	git := fx.exec.callsOf("git")
	if len(git) != 1 {
		t.Fatalf("git ran %d times, want once", len(git))
	}
	if want := []string{"ls-remote", "--symref", "origin", "HEAD", "refs/heads/*"}; !slices.Equal(git[0].args, want) ||
		git[0].dir != "/repo" || !slices.Equal(git[0].env, []string{"GIT_TERMINAL_PROMPT=0"}) {
		t.Fatalf("git call = %+v, want %q in /repo with GIT_TERMINAL_PROMPT=0", git[0], want)
	}
	gh := fx.exec.callsOf("gh")
	if len(gh) != 2 {
		t.Fatalf("gh ran %d times, want once per branch", len(gh))
	}
	for i, branch := range []string{"feat/a", "feat/b"} {
		want := []string{"pr", "list", "--head", branch, "--state", "all", "--limit", "1000", "--json",
			"number,state,url,headRefOid,baseRefName,isCrossRepository"}
		if c := gh[i]; !slices.Equal(c.args, want) || c.dir != "/repo" || !slices.Equal(c.env, []string{"GH_PROMPT_DISABLED=1"}) {
			t.Errorf("gh call %d = %+v, want %q in /repo with GH_PROMPT_DISABLED=1", i, c, want)
		}
	}
}

// TestMergeOrder_NoTargetsRunsNoGit covers C2's early return: with no
// feature branch in the ledger (no manifest, an empty one, or only records
// that do not count) and no merge-guarantee cache, the result is empty with
// a note and the guarantee not_checked, and no git or gh runs.
func TestMergeOrder_NoTargetsRunsNoGit(t *testing.T) {
	for _, tc := range []struct {
		name  string
		lines []any
		notes []string
	}{
		{name: "no manifest"},
		{name: "an empty manifest", lines: []any{"\n\n"}},
		{name: "a plain reservation and a seat", lines: []any{reserveEvent("org-p", "docs/"),
			ManifestEvent{OrgID: "org-p", SeatID: LeaderIdentity, Event: EventSpawned}}},
		{name: "a corrupt line and a record attached to no branch", lines: []any{moCorruptLine, moGarbled("org-x", EventGatePassed)},
			notes: []string{"1 corrupt line(s)", "1 gate record(s) cannot be read"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fx := newMOFixture(t)
			fx.add(t, tc.lines...)
			res := fx.run(t, false)
			if len(fx.exec.calls) != 0 {
				t.Fatalf("calls = %+v, want none", fx.exec.calls)
			}
			if res.Entries == nil || len(res.Entries) != 0 || res.DefaultBranch != "" || res.ListedAt != "" || res.GHAvailable {
				t.Fatalf("result = %+v, want an empty one", res)
			}
			if g := res.MergeGuarantee; g.Kind != MergeGuaranteeUnknown || g.Reason != MergeGuaranteeReasonNotChecked {
				t.Fatalf("merge guarantee = %+v, want unknown (not_checked) with no cache", g)
			}
			if len(res.Notes) != 1+len(tc.notes) || !strings.Contains(res.Notes[0], "no feature branches in the ledger") {
				t.Fatalf("notes = %q, want the no-branch note and %q", res.Notes, tc.notes)
			}
			for _, want := range tc.notes {
				if moNoteWith(res, want) == "" {
					t.Errorf("notes = %q, want one saying %q", res.Notes, want)
				}
			}
		})
	}
}

// moGuarantee is a github_protection cache of moOriginURL's repository o/r,
// for its default branch branch, checked an hour before mergeOrderTestClock.
func moGuarantee(branch string) MergeGuaranteeCheck {
	return MergeGuaranteeCheck{
		Kind: MergeGuaranteeGitHubProtection, Detail: "a ruleset requires up-to-date branches with 1 required check",
		Host: "github.com", Repo: "o/r", APIRepo: "o/r", DefaultBranch: branch,
		CheckedAt: mergeOrderTestClock.Add(-time.Hour).Format(time.RFC3339),
	}
}

// saveGuarantee writes c as the merge-guarantee cache in fx's state dir.
func (fx *moFixture) saveGuarantee(t *testing.T, c MergeGuaranteeCheck) {
	t.Helper()
	if s := SaveMergeGuarantee(fx.stateDir, c); !s.Written || s.WriteErr != nil {
		t.Fatalf("save = %+v", s)
	}
}

// moGitArgs is each git call of e as its arguments joined by spaces.
func moGitArgs(e *moExec) []string {
	var out []string
	for _, c := range e.callsOf("git") {
		out = append(out, strings.Join(c.args, " "))
	}
	return out
}

// TestMergeOrder_FillsTheMergeGuaranteeFromTheLedgersCache covers AC7 and
// AC10 at the org layer: MergeOrder reads the cache in the manifest's state
// dir with its own exec in p.Dir and o.Now, after the ls-remote. A fresh
// cache of origin's repository and default branch gives its kind after one
// `git remote get-url origin`; no cache is not_checked with no git call but
// the ls-remote; a cache of another default branch than the ls-remote names
// is default_branch_changed; another origin is origin_changed; and a cache
// stale by o.Now is stale with no get-url, while one fresh by o.Now is used
// even though it is years old by the wall clock.
func TestMergeOrder_FillsTheMergeGuaranteeFromTheLedgersCache(t *testing.T) {
	const lsRemote, getURL = "ls-remote --symref origin HEAD refs/heads/*", "remote get-url origin"
	old := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	for _, tc := range []struct {
		name    string
		setup   func(fx *moFixture, c *MergeGuaranteeCheck)
		save    bool
		listing string
		kind    string
		reason  string
		text    string
		git     []string
	}{
		{name: "fresh cache of origin's default branch", save: true, kind: MergeGuaranteeGitHubProtection,
			text: "github_protection (main of github.com/o/r, checked 2026-10-14T08:00:00Z): a ruleset requires up-to-date " +
				"branches with 1 required check", git: []string{lsRemote, getURL}},
		{name: "no cache", kind: MergeGuaranteeUnknown, reason: MergeGuaranteeReasonNotChecked,
			text: "unknown (not checked: run ralph doctor)", git: []string{lsRemote}},
		{name: "cache of another default branch", save: true, listing: moListing("trunk", "trunk", moMain, "feat/a", moHeadA),
			kind: MergeGuaranteeUnknown, reason: MergeGuaranteeReasonDefaultBranchChanged,
			text: "unknown (doctor checked main; origin's default branch is now trunk)", git: []string{lsRemote, getURL}},
		{name: "origin points to another repository", save: true,
			setup: func(fx *moFixture, _ *MergeGuaranteeCheck) { fx.exec.origin = "git@github.com:o/other.git" },
			kind:  MergeGuaranteeUnknown, reason: MergeGuaranteeReasonOriginChanged,
			text: "unknown (doctor checked github.com/o/r; origin is now github.com/o/other)", git: []string{lsRemote, getURL}},
		{name: "stale by the org's clock", save: true,
			setup: func(_ *moFixture, c *MergeGuaranteeCheck) {
				c.CheckedAt = mergeOrderTestClock.Add(-25 * time.Hour).Format(time.RFC3339)
			},
			kind: MergeGuaranteeUnknown, reason: MergeGuaranteeReasonStale,
			text: "unknown (stale: last github_protection at 2026-10-13T08:00:00Z; run ralph doctor)", git: []string{lsRemote}},
		{name: "fresh by the org's clock, years old by the wall clock", save: true,
			setup: func(fx *moFixture, c *MergeGuaranteeCheck) {
				fx.o.Now = func() time.Time { return old }
				c.CheckedAt = old.Add(-time.Hour).Format(time.RFC3339)
			},
			kind: MergeGuaranteeGitHubProtection,
			text: "github_protection (main of github.com/o/r, checked 2019-12-31T23:00:00Z): a ruleset requires up-to-date " +
				"branches with 1 required check", git: []string{lsRemote, getURL}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fx := newMOFixture(t)
			fx.add(t, moBind("org-a", "a"), moPassed("org-a", "feat/a", moHeadA, moMain))
			fx.exec.lsOut = cmp.Or(tc.listing, moListing("main", "main", moMain, "feat/a", moHeadA))
			c := moGuarantee("main")
			if tc.setup != nil {
				tc.setup(fx, &c)
			}
			if tc.save {
				fx.saveGuarantee(t, c)
			}
			g := fx.run(t, false).MergeGuarantee
			if g.Kind != tc.kind || g.Reason != tc.reason || g.Text() != tc.text {
				t.Errorf("merge guarantee = %+v\ntext %q\nwant kind %s, reason %q, text %q", g, g.Text(), tc.kind, tc.reason, tc.text)
			}
			if tc.save && (g.Host != "github.com" || g.Repo != "o/r" || g.DefaultBranch != "main" || g.CheckedAt != c.CheckedAt) {
				t.Errorf("merge guarantee = %+v, want the cache's host, repo, branch and checked_at", g)
			}
			if tc.reason != "" && tc.save && g.LastKind != MergeGuaranteeGitHubProtection {
				t.Errorf("last kind = %q, want the cache's github_protection", g.LastKind)
			}
			if got := moGitArgs(fx.exec); !slices.Equal(got, tc.git) {
				t.Errorf("git calls = %q, want %q", got, tc.git)
			}
			for _, call := range fx.exec.callsOf("git") {
				if call.dir != "/repo" || !slices.Equal(call.env, []string{"GIT_TERMINAL_PROMPT=0"}) {
					t.Errorf("git call %+v, want it in /repo with GIT_TERMINAL_PROMPT=0", call)
				}
			}
		})
	}
}

// TestMergeOrder_NoTargetsReadsTheGuaranteeWithoutTheLsRemote: with no
// feature branch, a fresh cache still gives its kind after the one local
// `git remote get-url origin`, with no ls-remote and no gh, and the cache's
// branch is kept as it is, since no ls-remote read the default branch.
func TestMergeOrder_NoTargetsReadsTheGuaranteeWithoutTheLsRemote(t *testing.T) {
	fx := newMOFixture(t)
	fx.saveGuarantee(t, moGuarantee("main"))
	res := fx.run(t, false)
	if g := res.MergeGuarantee; g.Kind != MergeGuaranteeGitHubProtection || g.Reason != "" || g.DefaultBranch != "main" {
		t.Fatalf("merge guarantee = %+v, want the cache's github_protection for main", g)
	}
	if got := moGitArgs(fx.exec); !slices.Equal(got, []string{"remote get-url origin"}) || len(fx.exec.calls) != 1 {
		t.Fatalf("calls = %+v, want the get-url only", fx.exec.calls)
	}
	if res.DefaultBranch != "" || len(res.Entries) != 0 || !strings.Contains(res.Notes[0], "no feature branches in the ledger") {
		t.Fatalf("result = %+v, want the empty one with the no-branch note", res)
	}
}

// TestMergeOrder_TrailingLineWithoutNewlineIsCorrupt: a last line cut off
// with no newline that is not JSON counts as a corrupt line after the gate
// record, so the branch is unreadable.
func TestMergeOrder_TrailingLineWithoutNewlineIsCorrupt(t *testing.T) {
	fx := newMOFixture(t)
	fx.add(t, moBind("org-a", "auth"), moPassed("org-a", "feat/auth", moHeadA, moMain),
		`{"ts":"2026-10-14T08:59:00Z","org_id":"org-a","event":"gate_passed","details":"head=`)
	fx.exec.lsOut = moListing("main", "main", moMain, "feat/auth", moHeadA)
	res := fx.run(t, false)
	if got := moOrder(res); !slices.Equal(got, []string{"1 unreadable feat/auth"}) {
		t.Fatalf("order = %q, want feat/auth unreadable", got)
	}
	if moNoteWith(res, "the manifest has 1 corrupt line(s)") == "" {
		t.Fatalf("notes = %q, want the corrupt line counted", res.Notes)
	}
}

// TestMergeOrder_ManifestThatCannotBeReadIsAnError covers AC7's last
// sentence: a manifest that cannot be read is an error, before git runs.
func TestMergeOrder_ManifestThatCannotBeReadIsAnError(t *testing.T) {
	fx := newMOFixture(t)
	mkdirAll(t, fx.path) // a directory where the manifest should be
	res, err := fx.o.MergeOrder(context.Background(), fx.params(false))
	if err == nil || !strings.Contains(err.Error(), "org: merge-order: read manifest") {
		t.Fatalf("MergeOrder = %+v, %v; want a read manifest error", res, err)
	}
	if len(fx.exec.calls) != 0 {
		t.Fatalf("calls = %+v, want none", fx.exec.calls)
	}
}

// TestMergeOrder_WritesNothing covers AC9: MergeOrder leaves the manifest's
// bytes and every file and directory under the state dir's parent as they
// were (no lock file, no log, no cache), and needs no watch. The
// merge-guarantee cache it reads stays as it was too.
func TestMergeOrder_WritesNothing(t *testing.T) {
	fx := newMOFixture(t)
	fx.add(t,
		moBind("org-a", "a"), moPassed("org-a", "feat/a", moHeadA, moMain),
		moBind("org-b", "b"), moGarbled("org-b", EventGateFailed), moCorruptLine,
		moBind("org-c", "c"),
	)
	fx.saveGuarantee(t, moGuarantee("main"))
	fx.exec.lsOut = moListing("main", "main", moMain, "feat/a", moHeadA)
	fx.exec.setPR("feat/a", moPRs(moPR(1, "OPEN", "main")))
	before := moSnapshot(t, fx.root)
	manifest, err := os.ReadFile(fx.path)
	if err != nil {
		t.Fatal(err)
	}
	if g := fx.run(t, false).MergeGuarantee; g.Kind != MergeGuaranteeGitHubProtection {
		t.Fatalf("merge guarantee = %+v, want the cache read", g)
	}
	fx.run(t, true)
	if after := moSnapshot(t, fx.root); !slices.Equal(after, before) {
		t.Fatalf("the temp dir changed:\nbefore %q\nafter  %q", before, after)
	}
	if again, err := os.ReadFile(fx.path); err != nil || string(again) != string(manifest) {
		t.Fatalf("the manifest changed (%v)", err)
	}
}

// moSnapshot lists every path under root with its mode, size and content
// hash.
func moSnapshot(t *testing.T, root string) []string {
	t.Helper()
	var out []string
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		line := fmt.Sprintf("%s %v %d", strings.TrimPrefix(path, root), info.Mode(), info.Size())
		if info.Mode().IsRegular() {
			b, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			sum := sha256.Sum256(b)
			line += " " + hex.EncodeToString(sum[:])
		}
		out = append(out, line)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return out
}

// TestMergeOrder_ResultJSONShape: the result marshals with snake_case keys,
// empty lists as [] and the absent PR, gate and binding left out, and the
// merge guarantee as an object, so the CLI can print it as it is for --json.
func TestMergeOrder_ResultJSONShape(t *testing.T) {
	fx := newMOFixture(t)
	fx.add(t, moBind("org-a", "a"), moPassed("org-a", "feat/a", moHeadA, moMain), moPassed("org-x", "feat/x", moHeadA, moMain))
	fx.exec.lsOut = moListing("main", "main", moMain, "feat/a", moHeadA)
	fx.exec.setPR("feat/a", moPRs(moPR(1, "OPEN", "main")))
	b, err := json.Marshal(fx.run(t, false))
	if err != nil {
		t.Fatal(err)
	}
	var got struct {
		Top     map[string]json.RawMessage
		Entries []map[string]json.RawMessage `json:"entries"`
	}
	if err := json.Unmarshal(b, &got.Top); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatal(err)
	}
	keys := func(m map[string]json.RawMessage) []string {
		var k []string
		for key := range m {
			k = append(k, key)
		}
		slices.Sort(k)
		return k
	}
	sub := func(raw json.RawMessage) map[string]json.RawMessage {
		var m map[string]json.RawMessage
		if err := json.Unmarshal(raw, &m); err != nil {
			t.Fatalf("%s: %v", raw, err)
		}
		return m
	}
	for _, tc := range []struct {
		name string
		got  []string
		want []string
	}{
		{"result", keys(got.Top), []string{"default_branch", "default_head", "entries", "gh_available", "listed_at",
			"merge_guarantee", "notes"}},
		{"merge guarantee without a cache", keys(sub(got.Top["merge_guarantee"])), []string{"detail", "kind", "reason"}},
		{"full entry", keys(got.Entries[0]), []string{"binding", "branch", "class", "detail", "gate", "pr", "rank", "remote_head"}},
		{"pr", keys(sub(got.Entries[0]["pr"])), []string{"base_ref_name", "head_ref_oid", "number", "state", "url"}},
		{"gate", keys(sub(got.Entries[0]["gate"])), []string{"base", "base_ref", "event", "head", "readable", "reason", "ts"}},
		{"binding", keys(sub(got.Entries[0]["binding"])), []string{"feature", "org_id", "split", "ts"}},
		{"entry without a PR or binding", keys(got.Entries[1]), []string{"branch", "class", "detail", "gate", "rank", "remote_head"}},
	} {
		if !slices.Equal(tc.got, tc.want) {
			t.Errorf("%s keys = %q, want %q", tc.name, tc.got, tc.want)
		}
	}

	empty, err := json.Marshal(newMOFixture(t).run(t, false))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(empty), `"entries":[]`) || !strings.Contains(string(empty), `"notes":["no feature branches`) {
		t.Fatalf("empty result = %s, want entries [] and the no-branch note", empty)
	}
}

// TestMergeOrder_RealExecRunsInDirWithEnvAndStderr: realMergeOrderExec runs
// in dir, adds env over the inherited environment, returns stdout, puts the
// stderr of a failing command into the error, and reports a missing binary
// as exec.ErrNotFound.
func TestMergeOrder_RealExecRunsInDirWithEnvAndStderr(t *testing.T) {
	dir := resolved(t, t.TempDir())
	t.Setenv("MERGE_ORDER_TEST", "inherited")
	out, err := realMergeOrderExec(context.Background(), dir, []string{"MERGE_ORDER_TEST=added"}, "sh", "-c", `pwd -P; printf '%s\n' "$MERGE_ORDER_TEST"`)
	if err != nil || out != dir+"\nadded\n" {
		t.Fatalf("out = %q, %v; want %q", out, err, dir+"\nadded\n")
	}
	out, err = realMergeOrderExec(context.Background(), dir, nil, "sh", "-c", "echo partial; echo '  boom  ' >&2; exit 3")
	if err == nil || !strings.HasSuffix(err.Error(), "exit status 3: boom") || out != "partial\n" {
		t.Fatalf("out = %q, err = %v; want partial and exit status 3: boom", out, err)
	}
	if _, err = realMergeOrderExec(context.Background(), dir, nil, "ralph-merge-order-test-no-such-binary"); !errors.Is(err, exec.ErrNotFound) {
		t.Fatalf("err = %v, want exec.ErrNotFound", err)
	}
}

// TestMergeOrder_RealExecIsNotHeldByAChildHoldingThePipe: a child that keeps
// the output pipe open (ssh under git) holds neither a timeout nor an exit 0
// past WaitDelay; the exit 0 is still a success.
func TestMergeOrder_RealExecIsNotHeldByAChildHoldingThePipe(t *testing.T) {
	dir := t.TempDir()
	t.Cleanup(func() {
		for _, name := range []string{"timeout.pid", "exit0.pid"} {
			if b, err := os.ReadFile(filepath.Join(dir, name)); err == nil {
				if pid, err := strconv.Atoi(strings.TrimSpace(string(b))); err == nil {
					_ = syscall.Kill(pid, syscall.SIGKILL)
				}
			}
		}
	})
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	start := time.Now()
	_, err := realMergeOrderExec(ctx, dir, nil, "sh", "-c", "sleep 60 & echo $! > timeout.pid; wait")
	if elapsed := time.Since(start); err == nil || elapsed > 20*time.Second {
		t.Fatalf("timeout run returned %v after %s, want an error within WaitDelay of the timeout", err, elapsed)
	}
	start = time.Now()
	out, err := realMergeOrderExec(context.Background(), dir, nil, "sh", "-c", "sleep 60 & echo $! > exit0.pid; echo done")
	if elapsed := time.Since(start); err != nil || out != "done\n" || elapsed > 20*time.Second {
		t.Fatalf("exit 0 run = %q, %v after %s; want done with no error within WaitDelay", out, err, elapsed)
	}
}
