package org

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"
	"unicode"
)

// The fakes and fixtures that the tests of `ralph org merge` at the org layer
// share (plan docs/plans/active/2026-10-17-org-auto-merge.md, S9b): mxExec, a
// fake MergeOrderExec standing in for git and gh, and mxFixture, a ledger in
// a temp state dir, for checkMerge (merge_test.go); mrExec and mrFixture,
// which answer the calls Merge adds after checkMerge's (merge_run_test.go);
// and mwClock and mwFixture, a fake clock and the fixture of the wait of
// --wait-checks (merge_wait_test.go). No test runs gh or reaches a network.

// The keys mxExec answers by (mxKey).
const (
	mxGetURL   = "git remote get-url origin"
	mxLsRemote = "git ls-remote"
	mxPRList   = "gh pr list"
	mxPRView   = "gh pr view"
	mxPRChecks = "gh pr checks"
	mxRepoAPI  = "gh api repos/o/r"
	mxQueue    = "gh api graphql"
	mxRules    = "gh api repos/o/r/rules/branches/main"
	mxRuleset  = "gh api repos/o/r/rulesets/1"
	mxRepoView = "gh repo view"
)

// mxAll are the commands checkMerge runs when every check passes, in order.
var mxAll = []string{mxGetURL, mxLsRemote, mxPRList, mxPRView, mxPRChecks, mxRepoAPI, mxQueue, mxRules, mxRuleset, mxRepoView, mxLsRemote}

// Parts of the refusals of the default fixture.
const (
	mxRepoArg = "github.com/o/r"
	mxAgain   = "run ralph org merge --branch feat/a again"
	mxAccess  = "check that gh can reach github.com and is logged in to it (gh auth status --hostname github.com), and " + mxAgain
	mxProtect = "org: merge: GitHub's protection of origin's default branch was not confirmed, so nothing was merged: "
	mxSubject = "main, the default branch of origin's repository o/r"
)

// mxExec is a fake MergeOrderExec for checkMerge. Every call is recorded with
// its key (mxKey) and its ctx's deadline. `git remote get-url origin`
// answers originErr, else origin; the i-th ls-remote answers ls[i], the last
// one again after them; gh answers the reply under its key, and fails when it
// has none. With upstream, gh pr and gh repo view without origin's -R fail,
// as gh would read the upstream remote then. The call whose key is block
// waits for its ctx instead. after runs once the n-th call (from 0) has its
// answer, before it returns.
type mxExec struct {
	mu        sync.Mutex
	calls     []mxCall
	origin    string
	originErr error
	ls        []mgReply
	lsCalls   int
	replies   map[string]mgReply
	upstream  bool
	block     string
	after     func(n int)
}

type mxCall struct {
	key      string
	dir      string
	env      []string
	name     string
	args     []string
	deadline time.Time // ctx's, zero when it has none
}

// mxKey is the key of a call: the git command, `gh api <endpoint>`, or `gh
// <command> <subcommand>`.
func mxKey(name string, args []string) string {
	switch {
	case name == "git" && len(args) > 0 && args[0] == "ls-remote":
		return mxLsRemote
	case name == "gh" && len(args) > 0 && args[0] == "api":
		return "gh api " + args[len(args)-1]
	case name == "gh" && len(args) > 1:
		return "gh " + args[0] + " " + args[1]
	}
	return strings.Join(append([]string{name}, args...), " ")
}

func (e *mxExec) run(ctx context.Context, dir string, env []string, name string, args ...string) (string, error) {
	key := mxKey(name, args)
	deadline, _ := ctx.Deadline()
	e.mu.Lock()
	n := len(e.calls)
	e.calls = append(e.calls, mxCall{key: key, dir: dir, env: slices.Clone(env), name: name, args: slices.Clone(args), deadline: deadline})
	var r mgReply
	known := true
	switch key {
	case mxGetURL:
		r = mgReply{out: e.origin, err: e.originErr}
	case mxLsRemote:
		r = e.ls[min(e.lsCalls, len(e.ls)-1)]
		e.lsCalls++
	default:
		r, known = e.replies[key]
	}
	after := e.after
	e.mu.Unlock()
	if after != nil {
		defer after(n)
	}
	if err := ctx.Err(); err != nil {
		return "", err
	}
	if key == e.block {
		<-ctx.Done()
		return "", ctx.Err()
	}
	if e.upstream && name == "gh" && args[0] != "api" && !slices.Contains(args, mxRepoArg) {
		return "", errors.New("mxExec: gh read the upstream remote")
	}
	if !known {
		return "", fmt.Errorf("mxExec: unexpected %s %q", name, args)
	}
	return r.out, r.err
}

// keys are the keys of the calls, in order.
func (e *mxExec) keys() []string {
	e.mu.Lock()
	defer e.mu.Unlock()
	var out []string
	for _, c := range e.calls {
		out = append(out, c.key)
	}
	return out
}

func (e *mxExec) snapshot() []mxCall {
	e.mu.Lock()
	defer e.mu.Unlock()
	return slices.Clone(e.calls)
}

// mxListing is origin with main at moMain and feat/a at moHeadA, where the
// gate left them.
func mxListing() string { return moListing("main", "main", moMain, "feat/a", moHeadA) }

// mxPRViewJSON is a `gh pr view --json` answer; autoMerge is the
// autoMergeRequest as JSON.
func mxPRViewJSON(state, head, base, url, autoMerge string) string {
	return fmt.Sprintf(`{"autoMergeRequest":%s,"baseRefName":%q,"headRefOid":%q,"mergeCommit":null,"state":%q,"url":%q}`,
		autoMerge, base, head, state, url)
}

// mxChecks is a `gh pr checks --json` answer.
func mxChecks(checks ...ghPRCheck) string { return mgJSON(append([]ghPRCheck{}, checks...)) }

// mxPass is a check of the workflow ci that passed.
func mxPass(name string) ghPRCheck {
	return ghPRCheck{Bucket: "pass", Name: name, State: "SUCCESS", Workflow: "ci"}
}

// newMXExec answers every check of the branch feat/a of o/r on github.com as
// one that passes: PR #7, open, at moHeadA into main, with the check verify
// passed, a ruleset (id 1) the caller cannot bypass, no merge queue, and the
// merge method MERGE.
func newMXExec() *mxExec {
	return &mxExec{origin: moOriginURL + "\n", ls: []mgReply{{out: mxListing()}}, replies: map[string]mgReply{
		mxPRList:   {out: moPRs(moPR(7, "OPEN", "main"))},
		mxPRView:   {out: mxPRViewJSON("OPEN", moHeadA, "main", "https://github.com/o/r/pull/7", "null")},
		mxPRChecks: {out: mxChecks(mxPass("verify"))},
		mxRepoAPI:  {out: meRepoJSON("main", "o/r", nil, false)},
		mxQueue:    {out: meQueueJSON("null")},
		mxRules:    {out: mgRules(meChecksRule(1, true, "verify"))},
		mxRuleset:  {out: meRuleset(1, "protect main", "never")},
		mxRepoView: {out: `{"viewerDefaultMergeMethod":"MERGE"}`},
	}}
}

// mxReadyLedger binds feat/a to org-a, records a gate_passed of moHeadA on
// moMain, and disbands org-a.
func mxReadyLedger() []any {
	return []any{moBind("org-a", "a"), moPassed("org-a", "feat/a", moHeadA, moMain), moDisbanded("org-a")}
}

// mxFixture is a ledger in a temp state dir and the fake exec checkMerge runs
// through.
type mxFixture struct {
	mo   *moFixture
	exec *mxExec
}

// newMXFixture is the ledger lines (mxReadyLedger when nil) with newMXExec.
func newMXFixture(t *testing.T, lines []any) *mxFixture {
	t.Helper()
	fx := &mxFixture{mo: newMOFixture(t), exec: newMXExec()}
	if lines == nil {
		lines = mxReadyLedger()
	}
	fx.mo.add(t, lines...)
	return fx
}

func (fx *mxFixture) params() MergeParams {
	return MergeParams{Branch: "feat/a", Dir: "/repo", Exec: fx.exec.run, LookPath: mgFound}
}

// check runs checkMerge with p and fails the test unless the temp dir, the
// manifest included, is as it was before, and every command run is read-only
// (mxAssertReadOnly). checkKind returns the kind of the refusal too.
func (fx *mxFixture) check(ctx context.Context, t *testing.T, p MergeParams) (mergeReady, string, error) {
	t.Helper()
	r, refused, _, err := fx.checkKind(ctx, t, p)
	return r, refused, err
}

func (fx *mxFixture) checkKind(ctx context.Context, t *testing.T, p MergeParams) (mergeReady, string, mergeRefusalKind, error) {
	t.Helper()
	before := moSnapshot(t, fx.mo.root)
	r, refused, kind, err := fx.mo.o.checkMerge(ctx, p)
	if after := moSnapshot(t, fx.mo.root); !slices.Equal(after, before) {
		t.Errorf("the temp dir changed:\nbefore %q\nafter  %q", before, after)
	}
	mxAssertReadOnly(t, fx.exec)
	return r, refused, kind, err
}

// mxLsRemoteArgs are the ls-remote's arguments.
var mxLsRemoteArgs = []string{"ls-remote", "--symref", "origin", "HEAD", "refs/heads/*"}

// mxAssertReadOnly fails the test unless every call is one that changes
// nothing: git remote get-url origin, the ls-remote, gh pr list, view and
// checks, gh repo view, and gh api with no method and no input, with -f fields
// only on the GraphQL query; so never gh pr merge. No call names an upstream
// remote either.
func mxAssertReadOnly(t *testing.T, e *mxExec) {
	t.Helper()
	for _, c := range e.snapshot() {
		var readOnly bool
		switch {
		case c.name == "git":
			readOnly = slices.Equal(c.args, []string{"remote", "get-url", "origin"}) || slices.Equal(c.args, mxLsRemoteArgs)
		case slices.Contains([]string{mxPRList, mxPRView, mxPRChecks, mxRepoView}, c.key):
			readOnly = true
		case c.name == "gh" && c.args[0] == "api":
			writes := slices.ContainsFunc(c.args, func(a string) bool {
				return a == "-X" || a == "--input" || strings.HasPrefix(a, "--method") || strings.HasPrefix(a, "-X")
			})
			fields := slices.Contains(c.args, "-f") || slices.Contains(c.args, "-F")
			readOnly = !writes && (!fields || c.key == mxQueue && slices.Contains(c.args, "query="+mergeQueueQuery))
		}
		if !readOnly {
			t.Errorf("%s %q is not a read-only command", c.name, c.args)
		}
		if slices.ContainsFunc(c.args, func(a string) bool { return strings.Contains(a, "upstream") }) {
			t.Errorf("%s %q names the upstream remote", c.name, c.args)
		}
	}
}

// mxAssertPinned fails the test unless every call runs in /repo, git with
// GIT_TERMINAL_PROMPT=0, gh with GH_PROMPT_DISABLED=1 and pinned to origin's
// repository: `-R <repoArg>` on gh pr, repoArg as gh repo view's argument, and
// `--hostname <apiHost>` on gh api.
func mxAssertPinned(t *testing.T, e *mxExec, apiHost, repoArg string) {
	t.Helper()
	for _, c := range e.snapshot() {
		env := []string{"GH_PROMPT_DISABLED=1"}
		var pinned bool
		switch {
		case c.name == "git":
			env, pinned = []string{"GIT_TERMINAL_PROMPT=0"}, true
		case c.args[0] == "pr":
			i := slices.Index(c.args, "-R")
			pinned = i > 0 && i+1 < len(c.args) && c.args[i+1] == repoArg
		case c.args[0] == "repo":
			pinned = len(c.args) > 2 && c.args[2] == repoArg
		case c.args[0] == "api":
			pinned = slices.Equal(c.args[1:3], []string{"--hostname", apiHost})
		}
		if !pinned || c.dir != "/repo" || !slices.Equal(c.env, env) {
			t.Errorf("%s %q in %q with %q is not pinned to %s (%s) in /repo with %q", c.name, c.args, c.dir, c.env, repoArg, apiHost, env)
		}
	}
}

// mxGHMissing is the refusal of a gh that is not on PATH.
const mxGHMissing = `org: merge: gh is not on PATH (exec: "gh": executable file not found in $PATH), and ralph org merge reads ` +
	"the PR and merges it with gh, so nothing was merged: install gh, log in to github.com (gh auth login --hostname " +
	"github.com), and " + mxAgain

// mxNotFound is how a gh that is not installed fails.
var mxNotFound = mgReply{err: &exec.Error{Name: "gh", Err: exec.ErrNotFound}}

// mxForeign is the refusal of a PR whose url u names another repository than
// origin's, as cmd returned it.
func mxForeign(cmd, u string) string {
	return "org: merge: " + cmd + " returned PR #7 with the url " + u + ", which is not a PR of origin's repository " + mxRepoArg +
		", so nothing was merged: ralph org merge merges only origin's PRs; check origin (git remote get-url origin), and merge " +
		"the PR yourself if it is the one to merge"
}

// The keys of the calls Merge adds (mrExec).
const (
	mrMerge   = "gh pr merge"
	mrDisable = "gh pr merge --disable-auto"
	mrReread  = "gh pr view (read back)"
)

// Parts of the texts of the default fixture.
const (
	mrMergeCmd   = "gh pr merge 7 -R " + mxRepoArg
	mrViewCmd    = "gh pr view 7 -R " + mxRepoArg
	mrDisableCmd = "gh pr merge 7 --disable-auto -R " + mxRepoArg
	mrRequest    = `{"enabledAt":"2026-10-17T00:00:00Z","mergeMethod":"MERGE"}`
	mrProtection = `the ruleset "protect main" (id 1) of ` + mxSubject + ", requires branches to be up to date before merging " +
		"with its 1 required status check, and you cannot bypass it (current_user_can_bypass: never)"
)

// mrOID is the merge commit the fixtures' read-back gives.
var mrOID = strings.Repeat("c", 40)

// mrMerged are the commands of a merge that went through: checkMerge's, gh
// pr merge, and the read-back; mrDisabled adds the removal of a request.
var (
	mrMerged   = slices.Concat(mxAll, []string{mrMerge, mrReread})
	mrDisabled = slices.Concat(mrMerged, []string{mrDisable})
)

// mrExec is mxExec with the calls Merge adds answered by their own replies:
// gh pr merge (merge), gh pr merge --disable-auto (disable), and gh pr view
// with mergeReadBackFields (reread). They are recorded with mxExec's calls.
// In gh pr merge, inMerge runs with its ctx first, when set; then, with
// blockMerge, it waits for its ctx. A call whose ctx is done fails with ctx's
// error, as a gh that is killed does.
type mrExec struct {
	*mxExec
	merge, disable, reread mgReply
	inMerge                func(ctx context.Context)
	blockMerge             bool
}

func (e *mrExec) run(ctx context.Context, dir string, env []string, name string, args ...string) (string, error) {
	var key string
	var r mgReply
	switch {
	case name != "gh" || len(args) < 2 || args[0] != "pr":
		return e.mxExec.run(ctx, dir, env, name, args...)
	case args[1] == "merge" && slices.Contains(args, "--disable-auto"):
		key, r = mrDisable, e.disable
	case args[1] == "merge":
		key, r = mrMerge, e.merge
	case args[1] == "view" && slices.Contains(args, mergeReadBackFields):
		key, r = mrReread, e.reread
	default:
		return e.mxExec.run(ctx, dir, env, name, args...)
	}
	deadline, _ := ctx.Deadline()
	e.mu.Lock()
	e.calls = append(e.calls, mxCall{key: key, dir: dir, env: slices.Clone(env), name: name, args: slices.Clone(args), deadline: deadline})
	e.mu.Unlock()
	if key == mrMerge && e.inMerge != nil {
		e.inMerge(ctx)
	}
	if key == mrMerge && e.blockMerge {
		<-ctx.Done()
	}
	if err := ctx.Err(); err != nil {
		return "", err
	}
	return r.out, r.err
}

// mrReadBack is a `gh pr view --json
// state,headRefOid,baseRefName,mergeCommit,autoMergeRequest` answer whose
// head is the gate's, moHeadA, and whose base is main, the fixture's default
// branch; an empty oid is a null mergeCommit, and autoMerge is the
// autoMergeRequest as JSON.
func mrReadBack(state, oid, autoMerge string) string {
	return mrReadBackAt(state, moHeadA, oid, autoMerge)
}

// mrReadBackAt is mrReadBack with head as headRefOid.
func mrReadBackAt(state, head, oid, autoMerge string) string {
	return mrReadBackInto(state, head, "main", oid, autoMerge)
}

// mrReadBackInto is mrReadBack with head as headRefOid and base as
// baseRefName, each written as JSON writes it, control characters included.
func mrReadBackInto(state, head, base, oid, autoMerge string) string {
	commit := "null"
	if oid != "" {
		commit = `{"oid":"` + oid + `"}`
	}
	h, err := json.Marshal(head)
	if err != nil {
		panic(err)
	}
	b, err := json.Marshal(base)
	if err != nil {
		panic(err)
	}
	return fmt.Sprintf(`{"autoMergeRequest":%s,"baseRefName":%s,"headRefOid":%s,"mergeCommit":%s,"state":%q}`, autoMerge, b, h,
		commit, state)
}

// mrFixture is mxFixture with mrExec in front of its mxExec, which answers
// the read-back with PR #7 MERGED as mrOID; manifest is the manifest as it
// was when merge started.
type mrFixture struct {
	*mxFixture
	ex       *mrExec
	manifest string
}

// newMRFixture is newMXFixture(t, lines) with mrExec.
func newMRFixture(t *testing.T, lines []any) *mrFixture {
	t.Helper()
	mx := newMXFixture(t, lines)
	return &mrFixture{mxFixture: mx, ex: &mrExec{mxExec: mx.exec, reread: mgReply{out: mrReadBack("MERGED", mrOID, "null")}}}
}

func (fx *mrFixture) params() MergeParams {
	p := fx.mxFixture.params()
	p.Exec = fx.ex.run
	return p
}

// merge runs Merge with p and fails the test unless every command ran in
// /repo pinned to origin's repository (mxAssertPinned) and none carries
// --auto, --admin, --delete-branch, or -d.
func (fx *mrFixture) merge(ctx context.Context, t *testing.T, p MergeParams) (MergeResult, error) {
	t.Helper()
	fx.manifest = mrRead(t, fx.mo.path)
	res, err := fx.mo.o.Merge(ctx, p)
	mxAssertPinned(t, fx.exec, "github.com", mxRepoArg)
	for _, c := range fx.exec.snapshot() {
		for _, flag := range []string{"--auto", "--admin", "--delete-branch", "-d"} {
			if slices.Contains(c.args, flag) {
				t.Errorf("%s %q carries %s", c.name, c.args, flag)
			}
		}
	}
	return res, err
}

// appended is what was appended to the manifest since merge started,
// failing the test unless the manifest then is a prefix of it now.
func (fx *mrFixture) appended(t *testing.T) string {
	t.Helper()
	now := mrRead(t, fx.mo.path)
	if !strings.HasPrefix(now, fx.manifest) {
		t.Fatalf("the manifest was rewritten:\nbefore %q\nafter  %q", fx.manifest, now)
	}
	return now[len(fx.manifest):]
}

// call is the one call with key, failing the test unless there is exactly
// one.
func (fx *mrFixture) call(t *testing.T, key string) mxCall {
	t.Helper()
	var found []mxCall
	for _, c := range fx.exec.snapshot() {
		if c.key == key {
			found = append(found, c)
		}
	}
	if len(found) != 1 {
		t.Fatalf("%d calls of %s, want 1: %q", len(found), key, fx.exec.keys())
	}
	return found[0]
}

// mrRead is the file at path, "" when it does not exist.
func mrRead(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		t.Fatal(err)
	}
	return string(b)
}

// mrRecord is the manifest line of the fixture's pr_merged with details.
func mrRecord(t *testing.T, details string) string {
	t.Helper()
	b, err := json.Marshal(ManifestEvent{TS: mergeOrderTestListedAt, OrgID: "org-a", Event: EventPRMerged, Details: details})
	if err != nil {
		t.Fatal(err)
	}
	return string(b) + "\n"
}

// mrResult is the MergeResult fields the checks of the default fixture set,
// with Method set to method.
func mrResult(method string) MergeResult {
	return MergeResult{Branch: "feat/a", OrgID: "org-a", PRNumber: 7, PRURL: "https://github.com/o/r/pull/7", Method: method,
		Head: moHeadA, Protection: mrProtection}
}

// mrAssertLockFree fails the test unless regate.lock in stateDir can be
// taken at once, so Merge released it.
func mrAssertLockFree(t *testing.T, stateDir string) {
	t.Helper()
	lock, err := acquireHeldFileLock(stateDir, regateLockFile, "test", 0)
	if err != nil {
		t.Fatalf("regate.lock after Merge: %v", err)
	}
	if err := lock.release(); err != nil {
		t.Fatal(err)
	}
}

// mrNoRawControl fails the test when s holds a control character other than
// the line breaks of a gh said tail.
func mrNoRawControl(t *testing.T, s string) {
	t.Helper()
	if strings.ContainsFunc(s, func(r rune) bool { return unicode.IsControl(r) && r != '\n' }) {
		t.Errorf("%q holds a raw control character", s)
	}
}

// mwMaxSleeps is how many sleeps mwClock allows before it ends the wait.
const mwMaxSleeps = 50

// mwClock is a fake clock for a wait: now moves only when sleep sleeps. Each
// sleep is recorded; onSleep, when set, runs after it with the sleep's
// number (from 1), as what happens while Merge sleeps. A sleep returns ctx's
// error. After mwMaxSleeps sleeps it fails the test and stops ctx, so a wait
// that does not end ends.
type mwClock struct {
	t       *testing.T
	now     time.Time
	sleeps  []time.Duration
	onSleep func(n int)
	stop    context.CancelCauseFunc
}

func (c *mwClock) Now() time.Time { return c.now }

func (c *mwClock) sleep(ctx context.Context, d time.Duration) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	c.now = c.now.Add(d)
	c.sleeps = append(c.sleeps, d)
	if len(c.sleeps) > mwMaxSleeps {
		c.t.Errorf("more than %d sleeps: the wait does not end", mwMaxSleeps)
		c.stop(errors.New("too many sleeps"))
	}
	if c.onSleep != nil {
		c.onSleep(len(c.sleeps))
	}
	return ctx.Err()
}

// mwFixture is mrFixture with gh pr checks answering checks, Merge's
// params with WaitChecks and Progress, a ctx the test stops, and mwClock,
// whose every sleep takes regate.lock and releases it (lockTaken counts
// them) before onSleep.
type mwFixture struct {
	*mrFixture
	clock     *mwClock
	progress  *strings.Builder
	ctx       context.Context
	cancel    context.CancelCauseFunc
	p         MergeParams
	onSleep   func(n int)
	lockTaken int
}

func newMWFixture(t *testing.T, checks mgReply, wait time.Duration) *mwFixture {
	t.Helper()
	return newMWFixtureOn(t, nil, checks, wait)
}

// newMWFixtureOn is newMWFixture on the ledger lines (mxReadyLedger when
// nil).
func newMWFixtureOn(t *testing.T, lines []any, checks mgReply, wait time.Duration) *mwFixture {
	t.Helper()
	fx := &mwFixture{mrFixture: newMRFixture(t, lines), progress: &strings.Builder{}}
	fx.ctx, fx.cancel = context.WithCancelCause(context.Background())
	t.Cleanup(func() { fx.cancel(nil) })
	fx.clock = &mwClock{t: t, now: time.Date(2026, 10, 17, 0, 0, 0, 0, time.UTC), stop: fx.cancel, onSleep: func(n int) {
		lock, err := acquireHeldFileLock(fx.mo.stateDir, regateLockFile, "test", 0)
		if err != nil {
			t.Errorf("regate.lock while Merge waits: %v; want it free", err)
		} else {
			fx.lockTaken++
			_ = lock.release()
		}
		if fx.onSleep != nil {
			fx.onSleep(n)
		}
	}}
	fx.exec.replies[mxPRChecks] = checks
	fx.p = fx.params()
	fx.p.WaitChecks, fx.p.Progress = wait, fx.progress
	fx.p.waitTest = mergeWaitTest{now: fx.clock.Now, sleep: fx.clock.sleep}
	return fx
}

// setChecks makes gh pr checks answer r from now on.
func (fx *mwFixture) setChecks(r mgReply) {
	fx.exec.mu.Lock()
	defer fx.exec.mu.Unlock()
	fx.exec.replies[mxPRChecks] = r
}

// afterCall runs f once the call with index n (from 0) has its answer.
func (fx *mwFixture) afterCall(n int, f func()) {
	fx.exec.after = func(i int) {
		if i == n {
			f()
		}
	}
}

// lines are the lines Merge wrote to Progress.
func (fx *mwFixture) lines() []string {
	var out []string
	for line := range strings.Lines(fx.progress.String()) {
		out = append(out, strings.TrimSuffix(line, "\n"))
	}
	return out
}

// check fails the test unless the commands, the sleeps, and the lines are
// keys, sleeps, and lines, and regate.lock was free in every sleep and is
// free now.
func (fx *mwFixture) check(t *testing.T, keys []string, sleeps []time.Duration, lines []string) {
	t.Helper()
	if got := fx.exec.keys(); !slices.Equal(got, keys) {
		t.Errorf("commands = %q\nwant %q", got, keys)
	}
	if !slices.Equal(fx.clock.sleeps, sleeps) {
		t.Errorf("sleeps = %v, want %v", fx.clock.sleeps, sleeps)
	}
	if fx.lockTaken != len(fx.clock.sleeps) {
		t.Errorf("regate.lock was taken in %d of the %d sleeps; want every one", fx.lockTaken, len(fx.clock.sleeps))
	}
	if got := fx.lines(); !slices.Equal(got, lines) {
		t.Errorf("Progress =\n%s\nwant\n%s", strings.Join(got, "\n"), strings.Join(lines, "\n"))
	}
	mrAssertLockFree(t, fx.mo.stateDir)
}

// fx.run runs Merge with fx.p and fx.ctx.
func (fx *mwFixture) run(t *testing.T) (MergeResult, error) {
	t.Helper()
	return fx.merge(fx.ctx, t, fx.p)
}

// addDuring appends lines to the ledger in the sleep numbered n (from 1)
// and returns a pointer to what that appended, as the manifest holds it.
func (fx *mwFixture) addDuring(t *testing.T, n int, lines ...any) *string {
	var added string
	fx.onSleep = func(i int) {
		if i == n {
			before := mrRead(t, fx.mo.path)
			fx.mo.add(t, lines...)
			added = strings.TrimPrefix(mrRead(t, fx.mo.path), before)
		}
	}
	return &added
}
