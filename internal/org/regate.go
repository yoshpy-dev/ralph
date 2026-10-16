package org

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"time"
)

// The `ralph org regate` verb (Regate), stage 8a-3b of plan
// docs/plans/active/2026-10-16-org-regate.md (FR-7 of
// docs/specs/2026-10-07-org-multi-org-director.md). While a feature org runs,
// its leader runs ralph org gate; once the org has opened its PR and been
// disbanded, Gate refuses it, so a branch that ralph org merge-order lists as
// rebase or regate after another feature's PR was merged had no ralph verb to
// bring it back to ready. Regate is that verb, named by the branch, as
// merge-order lists them (D1):
//
//   - under regate.lock, which one Regate of a ledger holds at a time (D7),
//     it reads the branch's latest complete feature binding and latest gate
//     record under the manifest lock (D2), refuses a branch no complete
//     binding names or whose org runs, and marks the lock with the org it
//     works on, so a spawn of that org is refused until it ends (D9);
//   - in the binding's feature worktree it makes Gate's checks of the branch,
//     the tree, and the scripts (regateCheckWorktree);
//   - it fetches origin's default branch and the branch (regateFetch, D8),
//     and refuses a HEAD that is not origin's tip of the branch (D4);
//   - when HEAD contains origin's default branch tip and the branch's latest
//     gate record is a readable gate_passed of HEAD on that tip, it stops
//     there (RegateOutcomeUpToDate, AC3); when HEAD contains the tip
//     otherwise, it runs the gate scripts on HEAD as Gate does (gate_run.go)
//     and records the outcome as an org-level gate_passed or gate_failed of
//     the binding's org with via=regate, after it reads the ledger again
//     (recordRegate, D6, RegateOutcomeGated, AC9). A HEAD that does not
//     contain the tip is refused.
//
// A refusal records nothing and changes neither the worktree nor origin; the
// fetches change only refs/remotes/origin/<name>. Regate does not push.
// Neither the watch gate (watchgate.go) nor the approval check (approval.go)
// applies to it, since it starts no new work (D1).

// The outcomes of a Regate that was not refused (RegateResult.Outcome).
const (
	// RegateOutcomeUpToDate: HEAD contains origin's default branch tip and the
	// branch's latest gate record is a gate_passed of HEAD on that tip
	// (AC3). Nothing was run, recorded, or pushed.
	RegateOutcomeUpToDate = "up_to_date"
	// RegateOutcomeGated: HEAD contains the tip but has no such gate_passed
	// (AC9). The scripts ran on HEAD and a gate_passed or gate_failed was
	// recorded (RegateResult.Passed tells which). Nothing was pushed.
	RegateOutcomeGated = "gated"
)

const (
	// regateLockFile is the lock in the state dir that Regate holds from its
	// first check to its end (D7). Its content is the marker of the Regate
	// that holds it, or that last held it (regateMarker, D9).
	regateLockFile = "regate.lock"
	// regateLockWait is how long Regate retries regate.lock before it
	// refuses: a spawn's check holds the lock shared for a moment (D9), which
	// must not make a Regate fail.
	regateLockWait = time.Second
	// regateRemoteWaitDelay is regateRemoteGit's exec.Cmd.WaitDelay: how long
	// after the kill, or after git exited, it waits for the output pipes,
	// which a child (ssh, a credential helper) can hold.
	regateRemoteWaitDelay = 2 * time.Second
	// regateStderrTailLines is how many of git's last stderr lines a refused
	// fetch quotes.
	regateStderrTailLines = 5
	// regateRemoteNoRef is what git prints on stderr when origin has no ref
	// a fetch names (LC_ALL=C, regateRemoteEnv).
	regateRemoteNoRef = "couldn't find remote ref"
)

// regateRemoteTimeout bounds each git command Regate runs against origin
// (D8). Tests shorten it.
var regateRemoteTimeout = 120 * time.Second

// regateTestHooks are the points where a test stops Regate. Each is nil
// outside tests.
var regateTestHooks struct {
	// afterFirstCheck runs right after the first check of the ledger passed
	// (regateFirstCheck), regate.lock held with its marker written.
	afterFirstCheck func()
	// beforeRecord runs right before the record (recordRegate), once the
	// scripts have run.
	beforeRecord func()
	// afterRecord runs right after the record was appended, regate.lock
	// still held.
	afterRecord func()
}

// RegateParams describes one `ralph org regate` call.
type RegateParams struct {
	// Branch is the feature branch to regate.
	Branch string
	// Env is the caller's environment, NAME=value entries (the CLI passes
	// os.Environ()); nil means os.Environ(). The scripts get it fixed by
	// gateScriptEnv, the git commands against origin get it with
	// regateRemoteEnv's variables after it, and the record's by= and
	// herdr_pane= come from its USER and HERDR_PANE_ID, as Gate's do.
	Env []string
}

// RegateResult is Regate's outcome when it returns no error.
type RegateResult struct {
	// Refused is set when Regate refused: the reason and one next step.
	// Nothing was recorded, and neither the worktree nor origin was changed.
	// A refusal before the scripts leaves every other field empty. A refusal
	// at the record (the branch's binding or its org changed while the
	// scripts ran, recordRegate) keeps the other fields as the scripts left
	// them, LogPath included, with Outcome empty.
	Refused string
	// Outcome is RegateOutcomeUpToDate or RegateOutcomeGated; empty when
	// Refused is set.
	Outcome string
	// Passed reports a recorded gate_passed (RegateOutcomeGated); false with
	// Outcome RegateOutcomeGated is a recorded gate_failed.
	Passed bool
	// Reason is the gate_failed reason (GateReason*), empty on a pass.
	Reason string
	// OrgID is the org of the branch's binding, which the record names;
	// Worktree is the binding's feature worktree; Branch is the branch.
	OrgID, Worktree, Branch string
	// Head is HEAD, origin's tip of the branch; Base is origin's default
	// branch tip after the fetch, which HEAD contains; BaseRef is
	// refs/remotes/origin/<default branch>.
	Head, Base, BaseRef string
	// LogPath is the absolute path of the gate log under <state-dir>/gates/
	// (RegateOutcomeGated).
	LogPath string
	// Tail is the failing script's last gateTailLines output lines
	// (static_failed and test_failed).
	Tail []string
	// Dirty is `git status --porcelain --untracked-files=all` after the
	// scripts (tree_changed), one line per entry.
	Dirty []string
}

// Regate runs `ralph org regate` for p.Branch. It checks, in this order, and
// refuses (RegateResult.Refused, nothing recorded, the worktree and origin
// unchanged) when a check fails:
//
//   - p.Branch is not empty;
//   - no other Regate of the ledger holds regate.lock after regateLockWait
//     (D7); once it holds the lock, it empties the marker an earlier Regate
//     left there (D9);
//   - under the manifest lock, a complete feature binding names p.Branch,
//     and the org of the latest one is not running (regateFirstCheck, D2);
//   - the binding's worktree passes regateCheckWorktree;
//   - both fetches pass (regateFetch, D8), and HEAD is origin's tip of the
//     branch (regateCheckOriginHead, D4);
//   - HEAD contains origin's default branch tip.
//
// When the branch's latest gate record is a readable gate_passed of HEAD on
// that tip with no corrupt line after it (regateBranch.upToDate), it returns
// RegateOutcomeUpToDate. Otherwise it writes a log under <state-dir>/gates/
// named after the binding's org (createGateLog) and runs the scripts as Gate
// does, without the manifest lock, reads HEAD and the tree again after a
// pass (gateCheckAfter), and records the outcome with via=regate under the
// manifest lock only when the branch's binding and its org are as the first
// check found them (recordRegate, D6): RegateOutcomeGated. regate.lock is
// released when Regate returns, whatever it returns; its marker stays.
//
// The error is for what is not a refusal: the manifest, the lock, or a git
// command that fails, a log that cannot be written, a record that cannot be
// appended, and an interruption: when ctx is done, a running fetch or script
// is stopped and nothing is recorded, up to the moment the event is appended
// (regateInterrupted).
func (o *Org) Regate(ctx context.Context, p RegateParams) (RegateResult, error) {
	if p.Branch == "" {
		return RegateResult{Refused: "org: regate: no branch named: name the feature branch to regate (ralph org merge-order " +
			"lists the ledger's branches)"}, nil
	}
	env := p.Env
	if env == nil {
		env = os.Environ()
	}
	words := regateWords(p.Branch)

	stateDir := filepath.Dir(o.Manifest.Path())
	lock, err := acquireHeldFileLock(stateDir, regateLockFile, "regate", regateLockWait)
	switch {
	case errors.Is(err, errFileLockHeld):
		return RegateResult{Refused: fmt.Sprintf("org: regate: another ralph org regate runs on this ledger (%s is held): "+
			"one regate runs at a time, so wait for it to end and %s", regateLockPathIn(stateDir), words.again)}, nil
	case err != nil:
		return RegateResult{}, err
	}
	defer func() { _ = lock.release() }()
	// The marker an earlier Regate left goes at once, so that a spawn of its
	// org is not refused while this one waits for the manifest lock (D9).
	if err := writeRegateMarker(lock.f, nil); err != nil {
		return RegateResult{}, err
	}

	b, refused, err := o.regateFirstCheck(ctx, p.Branch, lock)
	if err != nil {
		return RegateResult{}, regateInterrupted(ctx, err, "")
	}
	if refused != "" {
		return RegateResult{Refused: refused}, nil
	}
	if hook := regateTestHooks.afterFirstCheck; hook != nil {
		hook()
	}

	t, refused, err := regateCheckWorktree(ctx, b.orgID, b.binding, words)
	if err != nil {
		return RegateResult{}, regateInterrupted(ctx, err, "")
	}
	if refused != "" {
		return RegateResult{Refused: refused}, nil
	}
	defaultTip, branchTip, refused, err := regateFetch(ctx, t, env, words)
	if err != nil {
		return RegateResult{}, regateInterrupted(ctx, err, "")
	}
	if refused != "" {
		return RegateResult{Refused: refused}, nil
	}
	if refused, err = regateCheckOriginHead(ctx, t, branchTip, words); err != nil {
		return RegateResult{}, regateInterrupted(ctx, err, "")
	}
	if refused != "" {
		return RegateResult{Refused: refused}, nil
	}
	t.base = gateBase{branch: t.base.branch, ref: "refs/remotes/origin/" + t.base.branch, sha: defaultTip}
	contains, err := regateIsAncestor(ctx, t.root, t.base.sha, t.head)
	if err != nil {
		return RegateResult{}, regateInterrupted(ctx, err, "")
	}
	if !contains {
		return RegateResult{Refused: words.prefix + fmt.Sprintf("HEAD %s of %s in %s does not contain origin's tip %s of %s, "+
			"and regate does not bring it in: merge it yourself (git -C %s merge %s), push the merge, and %s",
			t.head, t.branch, t.root, t.base.sha, t.base.branch, t.root, t.base.ref, words.again)}, nil
	}

	res := RegateResult{OrgID: b.orgID, Worktree: b.binding.Worktree, Branch: t.branch, Head: t.head, Base: t.base.sha, BaseRef: t.base.ref}
	if b.upToDate(t) {
		res.Outcome = RegateOutcomeUpToDate
		return res, nil
	}
	return o.regateGateOnly(ctx, b, t, env, res)
}

// regateWords is Regate's wording of the worktree checks it shares with
// Gate (gateWording), for branch.
func regateWords(branch string) gateWording {
	return gateWording{prefix: "org: regate: ", again: "run ralph org regate --branch " + branch + " again"}
}

// regateLockPathIn is the regate.lock of the state dir stateDir.
func regateLockPathIn(stateDir string) string {
	return filepath.Join(stateDir, regateLockFile)
}

// regateInterrupted is the error Regate returns for err: when ctx is done,
// one that says Regate was interrupted and recorded nothing, and names the
// log when the scripts had started (logPath set); err itself otherwise.
func regateInterrupted(ctx context.Context, err error, logPath string) error {
	if ctx.Err() == nil {
		return err
	}
	if logPath == "" {
		return fmt.Errorf("org: regate: interrupted (%v) before the scripts ran: nothing was recorded", context.Cause(ctx))
	}
	return fmt.Errorf("org: regate: interrupted (%v): the scripts were stopped and nothing was recorded; the output so far is in %s",
		context.Cause(ctx), logPath)
}

// regateBranch is a branch as one read of the ledger finds it
// (readRegateBranch).
type regateBranch struct {
	// binding is the branch's latest complete feature binding, nil for none;
	// orgID is its org and pos its index in the ledger's events.
	binding *FeatureBinding
	orgID   string
	pos     int
	// running: orgID is in RunningOrgs.
	running bool
	// gate is the branch's latest gate record (collectMergeTargets), nil for
	// none; corruptAfterGate: it is readable and a corrupt line follows it.
	gate             *mergeGateRecord
	corruptAfterGate bool
}

// readRegateBranch reads branch from l the way merge-order does
// (collectMergeTargets): its latest complete binding, read back in full from
// its event, and its latest gate record. A branch l does not name is the
// zero regateBranch.
func readRegateBranch(l mergeLedger, branch string) regateBranch {
	targets, _ := collectMergeTargets(l)
	for _, t := range targets {
		if t.branch != branch {
			continue
		}
		b := regateBranch{gate: t.gate, corruptAfterGate: t.corruptAfterGate}
		if t.binding != nil {
			b.binding = reservationFromEvent(l.events[t.bindingPos]).Feature
			b.orgID, b.pos = t.binding.OrgID, t.bindingPos
			b.running = slices.Contains(RunningOrgs(l.events), b.orgID)
		}
		return b
	}
	return regateBranch{}
}

// upToDate reports whether b's latest gate record lets Regate stop before the
// scripts (AC3): it is readable, no corrupt line follows it (either could
// hide a newer record), and it is a gate_passed of t's HEAD whose base is t's
// base tip and whose base_ref names t's default branch.
func (b regateBranch) upToDate(t gateTarget) bool {
	g := b.gate
	if g == nil || !g.readable || b.corruptAfterGate || g.record.Reason != "" {
		return false
	}
	name, named := gateBaseBranchName(g.record.BaseRef)
	return g.record.Head == t.head && g.record.Base == t.base.sha && named && name == t.base.branch
}

// regateFirstCheck is Regate's first check, made under the manifest lock with
// regate.lock (lock) held and its marker emptied (D2, D9): it reads the
// ledger line by line (readMergeLedger), finds branch in it
// (readRegateBranch), and returns the refusal of a branch no complete binding
// names or whose org runs. When the checks pass it writes lock's marker
// before the manifest lock is released: the binding's org, branch, and
// worktree, so a spawn of that org that comes after it refuses (D9). A
// refusal leaves the marker empty.
func (o *Org) regateFirstCheck(ctx context.Context, branch string, lock *heldFileLock) (regateBranch, string, error) {
	var b regateBranch
	var refused string
	err := withManifestLockContext(ctx, filepath.Dir(o.Manifest.Path()), func() error {
		l, err := readMergeLedger(o.Manifest.Path())
		if err != nil {
			return fmt.Errorf("org: regate: read manifest: %w", err)
		}
		b = readRegateBranch(l, branch)
		switch {
		case b.binding == nil:
			refused = fmt.Sprintf("org: regate: no complete feature binding in the ledger names the branch %s: regate works on "+
				"the branch of a feature org that ralph org start --plan started; ralph org merge-order lists the ledger's "+
				"branches", branch)
			return nil
		case b.running:
			refused = fmt.Sprintf("org: regate: the branch %s is bound to org_id %q, which is running: while it runs, its leader "+
				"runs ralph org gate --org-id %s in the feature worktree, so wait for the org to be disbanded and run ralph org "+
				"regate --branch %s then", branch, b.orgID, b.orgID, branch)
			return nil
		}
		return writeRegateMarker(lock.f, &regateMarker{OrgID: b.orgID, Branch: branch, Worktree: b.binding.Worktree, PID: os.Getpid()})
	})
	return b, refused, err
}

// regateMarker is what regate.lock holds once a Regate's first check passed
// (D9): the org, branch, and feature worktree it works on, and its pid, as
// one JSON line. A spawn that finds the lock held reads its org_id
// (readRegateMarker). Regate empties it right after it takes the lock and
// leaves it when it releases the lock: emptied at the end, it would let a
// spawn read nothing and go on before the end.
type regateMarker struct {
	OrgID    string `json:"org_id"`
	Branch   string `json:"branch"`
	Worktree string `json:"worktree"`
	PID      int    `json:"pid"`
}

// writeRegateMarker replaces the content of f, the open regate.lock, with m
// as a JSON line, or with nothing when m is nil.
func writeRegateMarker(f *os.File, m *regateMarker) error {
	var data []byte
	if m != nil {
		line, err := json.Marshal(m)
		if err != nil {
			return fmt.Errorf("org: regate: encode the marker of %s: %w", f.Name(), err)
		}
		data = append(line, '\n')
	}
	if err := f.Truncate(0); err != nil {
		return fmt.Errorf("org: regate: write the marker of %s: %w", f.Name(), err)
	}
	if _, err := f.WriteAt(data, 0); err != nil {
		return fmt.Errorf("org: regate: write the marker of %s: %w", f.Name(), err)
	}
	return nil
}

// readRegateMarker returns the org_id of the marker in the regate.lock at
// path (regateMarker). ok is false when the file cannot be read, is empty, or
// holds no JSON marker with an org_id.
func readRegateMarker(path string) (orgID string, ok bool) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", false
	}
	var m regateMarker
	if err := json.Unmarshal(bytes.TrimSpace(data), &m); err != nil || m.OrgID == "" {
		return "", false
	}
	return m.OrgID, true
}

// regateCheckWorktree runs Regate's checks of f's feature worktree and
// returns what they found, or the refusal of the first that fails: the
// worktree is a directory, and the root of a git worktree; then, with Gate's
// checks and Regate's wording w, it is on f's branch (gateCheckBranch: not
// detached, no rebase stopped there), no rebase or merge is in progress
// (regateCheckInProgress), the tree is clean, a default branch is found
// (resolveGateBase, whose base ref and tip Regate replaces after the fetch),
// and both scripts are there (gateCheckScripts). Gate's check that HEAD
// contains the base is left out: Regate fetches first. The error is a git
// command that could not run or failed in a way no check expects.
func regateCheckWorktree(ctx context.Context, orgID string, f *FeatureBinding, w gateWording) (gateTarget, string, error) {
	info, err := os.Stat(f.Worktree)
	switch {
	case errors.Is(err, fs.ErrNotExist):
		return gateTarget{}, w.prefix + fmt.Sprintf("org_id %q's feature worktree %s of the branch %s does not exist, and regate "+
			"runs only there: when the branch's PR is merged or closed, clean up what is left with %s in the main checkout; "+
			"otherwise make the worktree again (git worktree prune, then git worktree add %s %s, in the main checkout) and %s",
			orgID, f.Worktree, f.Branch, featureWorktreeCleanup(orgID), f.Worktree, f.Branch, w.again), nil
	case err != nil:
		return gateTarget{}, w.prefix + fmt.Sprintf("org_id %q's feature worktree %s cannot be read (%s): restore it and %s",
			orgID, f.Worktree, pathErrorCause(err), w.again), nil
	case !info.IsDir():
		return gateTarget{}, w.prefix + fmt.Sprintf("org_id %q's feature worktree %s is not a directory: restore the worktree "+
			"of the branch %s there and %s", orgID, f.Worktree, f.Branch, w.again), nil
	}
	r, err := gateGit(ctx, f.Worktree, "rev-parse", "--show-toplevel")
	if err != nil {
		return gateTarget{}, "", err
	}
	if r.code != 0 {
		return gateTarget{}, w.prefix + fmt.Sprintf("org_id %q's feature worktree %s is not a git worktree (git rev-parse "+
			"--show-toplevel: %s): restore the worktree of the branch %s there and %s", orgID, f.Worktree, r.stderr, f.Branch, w.again), nil
	}
	root, err := filepath.EvalSymlinks(strings.TrimSuffix(r.stdout, "\n"))
	if err != nil {
		return gateTarget{}, "", fmt.Errorf("org: regate: resolve the worktree root %s: %w", strings.TrimSuffix(r.stdout, "\n"), err)
	}
	want, err := filepath.EvalSymlinks(f.Worktree)
	if err != nil {
		return gateTarget{}, w.prefix + fmt.Sprintf("org_id %q's feature worktree %s cannot be read (%s): restore it and %s",
			orgID, f.Worktree, pathErrorCause(err), w.again), nil
	}
	if root != want {
		return gateTarget{}, w.prefix + fmt.Sprintf("org_id %q's feature worktree %s is not the root of a git worktree but a "+
			"directory in %s: restore the worktree of the branch %s there and %s", orgID, f.Worktree, root, f.Branch, w.again), nil
	}

	branch, refused, err := gateCheckBranch(ctx, root, f, w)
	if err != nil || refused != "" {
		return gateTarget{}, refused, err
	}
	if refused, err = regateCheckInProgress(ctx, root, w); err != nil || refused != "" {
		return gateTarget{}, refused, err
	}
	dirty, err := gateDirtyLines(ctx, root)
	if err != nil {
		return gateTarget{}, "", err
	}
	if len(dirty) > 0 {
		return gateTarget{}, gateDirtyRefusal(root, dirty, w), nil
	}
	t := gateTarget{root: root, branch: branch}
	if t.head, err = gateHead(ctx, root); err != nil {
		return gateTarget{}, "", err
	}
	base, found, err := resolveGateBase(ctx, root)
	if err != nil {
		return gateTarget{}, "", err
	}
	if !found {
		return gateTarget{}, w.prefix + fmt.Sprintf("no default branch found in %s: refs/remotes/origin/HEAD names none that "+
			"exists, and neither main nor master exists as refs/remotes/origin/<name> or refs/heads/<name>: run git remote "+
			"set-head origin --auto there and %s", root, w.again), nil
	}
	t.base = base
	if refused = gateCheckScripts(root, w); refused != "" {
		return gateTarget{}, refused, nil
	}
	return t, "", nil
}

// regateCheckInProgress refuses root, worded with w, when a rebase or a
// merge is stopped there: the merge Regate does not start on one, and a
// rebase stopped on the branch would move it (gateGitPathExists). A rebase
// with a detached HEAD has been refused by gateCheckBranch already.
func regateCheckInProgress(ctx context.Context, root string, w gateWording) (string, error) {
	rebasing, err := gateRebaseInProgress(ctx, root)
	if err != nil {
		return "", err
	}
	if rebasing {
		return w.prefix + fmt.Sprintf("a rebase is in progress in %s: finish it with git rebase --continue, or undo it with "+
			"git rebase --abort, and %s", root, w.again), nil
	}
	merging, err := gateGitPathExists(ctx, root, "a merge", "MERGE_HEAD")
	if err != nil {
		return "", err
	}
	if merging {
		return w.prefix + fmt.Sprintf("a merge is in progress in %s: finish it with git commit, or undo it with git merge "+
			"--abort, and %s", root, w.again), nil
	}
	return "", nil
}

// regateFetch fetches origin's default branch t.base.branch and then the
// branch t.branch in t.root, each with its own refspec into
// refs/remotes/origin/ (D8), so that origin having no t.branch is told apart
// from every other failure, and returns the two tips it then reads there.
// The refusal says why a fetch failed: origin has no t.branch, the fetch ran
// past regateRemoteTimeout, or it failed another way, quoting the end of
// git's stderr. The error is for ctx done or a git that could not run.
func regateFetch(ctx context.Context, t gateTarget, env []string, w gateWording) (defaultTip, branchTip, refused string, err error) {
	var tips []string
	for _, name := range []string{t.base.branch, t.branch} {
		spec := "+refs/heads/" + name + ":refs/remotes/origin/" + name
		command := "git fetch origin " + spec
		r, timedOut, err := regateRemoteGit(ctx, t.root, env, "fetch", "origin", spec)
		switch {
		case err != nil:
			return "", "", "", err
		case timedOut:
			return "", "", w.prefix + fmt.Sprintf("%s in %s ran past %s and was stopped, so nothing was recorded: check the "+
				"network and origin's credentials and %s%s", command, t.root, regateRemoteTimeout, w.again, regateStderrTail(r.stderr)), nil
		case r.code != 0 && name == t.branch && strings.Contains(r.stderr, regateRemoteNoRef):
			return "", "", w.prefix + fmt.Sprintf("origin has no branch %s: it was deleted after the merge, or never pushed "+
				"(%s in %s failed), so nothing was recorded: when its PR is merged there is nothing to regate; otherwise push "+
				"it (git -C %s push origin %s) and %s", t.branch, command, t.root, t.root, t.branch, w.again), nil
		case r.code != 0:
			return "", "", w.prefix + fmt.Sprintf("%s in %s failed (exit %d), so nothing was recorded: check origin (git "+
				"remote get-url origin) and its credentials and %s%s", command, t.root, r.code, w.again, regateStderrTail(r.stderr)), nil
		}
		tip, err := gateGitOutput(ctx, t.root, "rev-parse", "--verify", "refs/remotes/origin/"+name+"^{commit}")
		if err != nil {
			return "", "", "", err
		}
		tips = append(tips, strings.TrimSpace(tip))
	}
	return tips[0], tips[1], "", nil
}

// regateStderrTail is the end of git's stderr for a refusal: "" when it
// printed nothing, else "; git said:" and its last regateStderrTailLines
// lines, each on a line of its own.
func regateStderrTail(stderr string) string {
	var lines []string
	for line := range strings.SplitSeq(stderr, "\n") {
		if line = strings.TrimRight(line, "\r \t"); line != "" {
			lines = append(lines, line)
		}
	}
	if len(lines) == 0 {
		return ""
	}
	return "; git said:\n  " + strings.Join(lines[max(0, len(lines)-regateStderrTailLines):], "\n  ")
}

// regateCheckOriginHead refuses t when HEAD is not originHead, origin's tip
// of the branch after the fetch (D4): Regate gates only a head origin has, so
// it neither takes in commits that were not pushed nor goes past what
// someone pushed. The refusal says which side has commits the other does
// not, and how to make them the same.
func regateCheckOriginHead(ctx context.Context, t gateTarget, originHead string, w gateWording) (string, error) {
	if t.head == originHead {
		return "", nil
	}
	originRef := "refs/remotes/origin/" + t.branch
	headHasOrigin, err := regateIsAncestor(ctx, t.root, originHead, t.head)
	if err != nil {
		return "", err
	}
	originHasHead, err := regateIsAncestor(ctx, t.root, t.head, originHead)
	if err != nil {
		return "", err
	}
	switch {
	case headHasOrigin:
		return w.prefix + fmt.Sprintf("HEAD %s of %s in %s has commits that origin's tip %s of the branch does not: regate "+
			"gates only what origin has, so push them (git -C %s push origin %s) and %s",
			t.head, t.branch, t.root, originHead, t.root, t.branch, w.again), nil
	case originHasHead:
		return w.prefix + fmt.Sprintf("origin's tip %s of %s has commits that HEAD %s in %s does not (they were pushed from "+
			"elsewhere): bring the worktree up to date (git -C %s merge --ff-only %s) and %s",
			originHead, t.branch, t.head, t.root, t.root, originRef, w.again), nil
	}
	return w.prefix + fmt.Sprintf("HEAD %s of %s in %s and origin's tip %s of the branch have diverged: reconcile them in the "+
		"worktree, push the result, and %s", t.head, t.branch, t.root, originHead, w.again), nil
}

// regateIsAncestor reports whether commit a is an ancestor of commit b, or
// b itself, in root's repository (`git merge-base --is-ancestor`, exit 1
// for no).
func regateIsAncestor(ctx context.Context, root, a, b string) (bool, error) {
	r, err := gateGit(ctx, root, "merge-base", "--is-ancestor", a, b)
	switch {
	case err != nil:
		return false, err
	case r.code == 0:
		return true, nil
	case r.code == 1:
		return false, nil
	}
	return false, fmt.Errorf("org: regate: git merge-base --is-ancestor %s %s in %s: exit %d: %s", a, b, root, r.code, r.stderr)
}

// regateRemoteEnv is env, copied, with LC_ALL=C and GIT_TERMINAL_PROMPT=0
// after it, so both win (D8): git asks for no credentials on the terminal,
// and prints its messages untranslated, which regateFetch reads
// (regateRemoteNoRef). ssh's own prompt for a key is left as it is.
func regateRemoteEnv(env []string) []string {
	return append(slices.Clip(env), "LC_ALL=C", "GIT_TERMINAL_PROMPT=0")
}

// regateRemoteGit runs `git --no-optional-locks -C dir args...`, a command
// that talks to origin, with env and regateRemoteEnv's variables, its stdin
// empty, stopped after regateRemoteTimeout (D8); timedOut reports that the
// bound stopped it. WaitDelay closes the output pipes a child still holds
// after the kill or after git exited, so a child cannot hold the bound up;
// an exit 0 whose pipe a child held is a success, the way realMergeOrderExec
// reads it. The error is for a git that could not run or ctx done; a
// non-zero exit is not one.
func regateRemoteGit(ctx context.Context, dir string, env []string, args ...string) (r gitResult, timedOut bool, err error) {
	remoteCtx, cancel := context.WithTimeout(ctx, regateRemoteTimeout)
	defer cancel()
	cmd := exec.CommandContext(remoteCtx, "git", append([]string{"--no-optional-locks", "-C", dir}, args...)...)
	cmd.Env = regateRemoteEnv(env)
	cmd.WaitDelay = regateRemoteWaitDelay
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	runErr := cmd.Run()
	r = gitResult{stdout: stdout.String(), stderr: strings.TrimSpace(stderr.String())}
	var exitErr *exec.ExitError
	switch {
	case ctx.Err() != nil:
		return r, false, fmt.Errorf("org: regate: git %s in %s: %w", strings.Join(args, " "), dir, context.Cause(ctx))
	case remoteCtx.Err() != nil:
		return r, true, nil
	case runErr == nil, errors.Is(runErr, exec.ErrWaitDelay):
	case errors.As(runErr, &exitErr) && exitErr.ExitCode() > 0:
		r.code = exitErr.ExitCode()
	default:
		return r, false, fmt.Errorf("org: regate: git %s in %s: %w", strings.Join(args, " "), dir, runErr)
	}
	return r, false, nil
}

// regateGateOnly is Regate's path for a HEAD that contains origin's default
// branch tip but has no gate_passed on it (AC9): it runs the scripts on t
// as Gate does, with a log of b's org, and records the outcome with
// via=regate (recordRegate). res holds what the checks found; the scripts'
// outcome is added to it.
func (o *Org) regateGateOnly(ctx context.Context, b regateBranch, t gateTarget, env []string, res RegateResult) (RegateResult, error) {
	logPath, logFile, err := o.createGateLog(b.orgID)
	if err != nil {
		return RegateResult{}, err
	}
	res.LogPath = logPath
	run, err := runGateScripts(ctx, t, gateScriptEnv(env), logFile)
	if closeErr := logFile.Close(); err == nil && closeErr != nil {
		err = fmt.Errorf("org: regate: write the log %s: %w", logPath, closeErr)
	}
	if err != nil {
		return res, regateInterrupted(ctx, err, logPath)
	}
	res.Reason, res.Tail = run.failed, run.tail
	if res.Reason == "" {
		if res.Reason, res.Dirty, err = gateCheckAfter(ctx, t); err != nil {
			return res, regateInterrupted(ctx, err, logPath)
		}
	}

	record := gateRecord{
		Reason: res.Reason, Head: res.Head, Base: res.Base, BaseRef: res.BaseRef, Branch: res.Branch,
		By: approverName(gateEnvValue(env, "USER")), HerdrPane: gateEnvValue(env, herdrPaneIDEnv) != "", Via: gateViaRegate, Log: logPath,
	}
	if hook := regateTestHooks.beforeRecord; hook != nil {
		hook()
	}
	if res.Refused, err = o.recordRegate(ctx, b, t.branch, record); err != nil {
		return res, regateInterrupted(ctx, err, logPath)
	}
	if res.Refused != "" {
		return res, nil
	}
	if hook := regateTestHooks.afterRecord; hook != nil {
		hook()
	}
	res.Outcome, res.Passed = RegateOutcomeGated, res.Reason == ""
	return res, nil
}

// recordRegate appends record as the gate event of was's org under the
// manifest lock, after it reads the ledger again (D6). The record of a
// running org is recordGate's, so Regate checks with its own rule: only when
// branch's latest complete binding is still the one the first check found
// (the same event, org, and binding: sameFeature) and its org is not running
// is anything recorded; otherwise it records nothing and returns the
// refusal. The error is a lock, read, or append that failed, or ctx's cause
// when ctx is done before the append; nothing is recorded then either.
func (o *Org) recordRegate(ctx context.Context, was regateBranch, branch string, record gateRecord) (string, error) {
	// As in recordGate, ctx is checked before the lock and again under it
	// right before the append.
	if ctx.Err() != nil {
		return "", context.Cause(ctx)
	}
	var refused string
	err := withManifestLockContext(ctx, filepath.Dir(o.Manifest.Path()), func() error {
		l, err := readMergeLedger(o.Manifest.Path())
		if err != nil {
			return fmt.Errorf("org: regate: read manifest: %w", err)
		}
		now := readRegateBranch(l, branch)
		var changed string
		switch {
		case now.binding == nil:
			changed = "no complete feature binding names it any more"
		case now.orgID != was.orgID || !sameFeature(now.binding, was.binding):
			changed = fmt.Sprintf("it is now bound to org_id %q's %s", now.orgID, now.binding)
		case now.pos != was.pos:
			changed = fmt.Sprintf("org_id %q bound it again", now.orgID)
		case now.running:
			changed = fmt.Sprintf("org_id %q is running again", now.orgID)
		}
		if changed != "" {
			refused = fmt.Sprintf("org: regate: nothing was recorded, since the branch %s changed while the scripts ran (%s): "+
				"check it with ralph org merge-order and ralph org status, and run ralph org regate --branch %s again once "+
				"its org is disbanded", branch, changed, branch)
			return nil
		}
		ev := ManifestEvent{TS: o.now(), OrgID: was.orgID, SeatID: "", Event: record.event(), Details: record.details()}
		if ctx.Err() != nil {
			return context.Cause(ctx)
		}
		if err := o.appendEvent(ev); err != nil {
			return fmt.Errorf("org: regate: record %s: %w", ev.Event, err)
		}
		return nil
	})
	switch {
	case err != nil && ctx.Err() != nil:
		// Interrupted: Regate says so (regateInterrupted).
		return "", err
	case err != nil:
		return "", fmt.Errorf("%w; nothing was recorded, so run ralph org regate --branch %s again", err, branch)
	}
	return refused, nil
}
