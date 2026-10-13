package org

import (
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
// MergeOrderExec standing in for git ls-remote and gh, so no test reaches a
// network. realMergeOrderExec runs real local processes only.

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

// moExec is a fake MergeOrderExec. git answers lsOut and lsErr (lsBlock:
// it waits for ctx instead). gh answers, per branch (`--head <branch>`), the
// error in ghErr, else the output in gh, else "[]"; with ghBlock it waits for
// ctx, and with ghMissing every gh call fails as a binary not on PATH.
type moExec struct {
	mu        sync.Mutex
	calls     []moCall
	lsOut     string
	lsErr     error
	lsBlock   bool
	ghMissing bool
	gh        map[string]string
	ghErr     map[string]error
	ghBlock   map[string]bool
}

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

// TestMergeOrder_LatestGateRecordWins covers AC2 and AC4 (C8): a branch's
// class reads its latest gate record. One that cannot be read (Details
// gateRecordFromDetails refuses, or an event name that disagrees with the
// reason), or a corrupt manifest line after the latest readable one, makes
// the branch unreadable and never falls back to an earlier gate_passed. A
// corrupt line before the latest record does not matter, and a readable
// record after it clears it.
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
			corrupt := slices.Contains(tc.lines, any(moCorruptLine))
			if note := moNoteWith(res, "the manifest has 1 corrupt line(s)"); corrupt != (note != "") {
				t.Errorf("notes = %q, want the corrupt-line note only when a corrupt line is there (%v)", res.Notes, corrupt)
			}
		})
	}
}

// TestMergeOrder_UnreadableRecordAttachesToTheBindingBeforeIt covers AC4's
// last sentence (C8): a gate record that cannot be read attaches to the
// branch of its org's latest complete binding before it, even when the org
// was bound to another feature later, and one with no complete binding of
// its org before it attaches to no branch and is only counted in a note.
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
// then one gh pr list per branch with GH_PROMPT_DISABLED=1, all in p.Dir.
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
		want := []string{"pr", "list", "--head", branch, "--state", "all", "--json", "number,state,url,headRefOid,baseRefName,isCrossRepository"}
		if c := gh[i]; !slices.Equal(c.args, want) || c.dir != "/repo" || !slices.Equal(c.env, []string{"GH_PROMPT_DISABLED=1"}) {
			t.Errorf("gh call %d = %+v, want %q in /repo with GH_PROMPT_DISABLED=1", i, c, want)
		}
	}
}

// TestMergeOrder_NoTargetsRunsNoGit covers C2's early return: with no
// feature branch in the ledger (no manifest, an empty one, or only records
// that do not count) the result is empty with a note, and no git or gh runs.
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
// were (no lock file, no log, no cache), and needs no watch.
func TestMergeOrder_WritesNothing(t *testing.T) {
	fx := newMOFixture(t)
	fx.add(t,
		moBind("org-a", "a"), moPassed("org-a", "feat/a", moHeadA, moMain),
		moBind("org-b", "b"), moGarbled("org-b", EventGateFailed), moCorruptLine,
		moBind("org-c", "c"),
	)
	fx.exec.lsOut = moListing("main", "main", moMain, "feat/a", moHeadA)
	fx.exec.setPR("feat/a", moPRs(moPR(1, "OPEN", "main")))
	before := moSnapshot(t, fx.root)
	manifest, err := os.ReadFile(fx.path)
	if err != nil {
		t.Fatal(err)
	}
	fx.run(t, false)
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
// empty lists as [] and the absent PR, gate and binding left out, so the
// CLI can print it as it is for --json.
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
		{"result", keys(got.Top), []string{"default_branch", "default_head", "entries", "gh_available", "listed_at", "notes"}},
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
