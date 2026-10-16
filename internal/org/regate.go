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
//     and refuses a HEAD that is not origin's tip of the branch (D4), but
//     for the retry of a push that failed: a HEAD with origin's tip of the
//     branch under it whose gate_passed on origin's default branch tip is
//     the branch's latest record is pushed, and nothing else is done
//     (regateHeadNotOnOrigin, RegateOutcomePushOnly);
//   - when HEAD contains origin's default branch tip and the branch's latest
//     gate record is a readable gate_passed of HEAD on that tip, it stops
//     there (RegateOutcomeUpToDate, AC3); when HEAD contains the tip
//     otherwise, it runs the gate scripts on HEAD as Gate does (gate_run.go)
//     and records the outcome as an org-level gate_passed or gate_failed of
//     the binding's org with via=regate, after it reads the ledger again
//     (recordRegate, D6, RegateOutcomeGated, AC9);
//   - when HEAD does not contain the tip, it merges the tip in with a merge
//     commit (regateMergeTip, D3), runs the scripts on the merge commit and
//     records the outcome the same way, and pushes a pass to origin's branch
//     without force (regatePush, RegateOutcomeMerged, AC4). Until a
//     gate_passed is recorded, it undoes its merge commit when a script
//     fails, the record is refused, or it is interrupted (regateUndoMerge,
//     D5); once one is recorded, it keeps the record and the merge commit
//     whatever the push does, and running it again retries the push.
//
// A refusal before the merge records nothing and changes neither the worktree
// nor origin; the fetches change only refs/remotes/origin/<name>. Neither the
// watch gate (watchgate.go) nor the approval check (approval.go) applies to
// it, since it starts no new work (D1).

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
	// RegateOutcomeMerged: HEAD did not contain the tip (AC4). Regate merged
	// the tip in with a merge commit (RegateResult.Merge), ran the scripts on
	// it, and recorded a gate_passed or gate_failed of it (Passed tells
	// which). A pass was pushed (Pushed, or PushFailed); a static_failed or
	// test_failed merge commit was undone (Undone, or NotUndone), and a
	// head_moved or tree_changed one was kept (NotUndone).
	RegateOutcomeMerged = "merged"
	// RegateOutcomePushOnly: HEAD was not origin's tip of the branch but had
	// it under it, and the branch's latest gate record is a gate_passed of
	// HEAD on origin's default branch tip (D4's retry of a push that failed).
	// Nothing was run or recorded; HEAD was pushed (Pushed, or PushFailed).
	RegateOutcomePushOnly = "push_only"
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
	// fetch, merge, or push quotes.
	regateStderrTailLines = 5
	// regateRemoteNoRef is what git prints on stderr when origin has no ref
	// a fetch names (LC_ALL=C, regateRemoteEnv).
	regateRemoteNoRef = "couldn't find remote ref"
	// regateNoIdentity is in what git prints on stderr when it has no
	// committer or author name and email to make a commit with (LC_ALL=C):
	// "Committer identity unknown", "Author identity unknown".
	regateNoIdentity = "identity unknown"
)

// regateRemoteTimeout bounds each git command Regate runs against origin
// (D8), and each one regateUncancelledGit runs. Tests shorten it.
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
	// still held, before a push.
	afterRecord func()
}

// RegateParams describes one `ralph org regate` call.
type RegateParams struct {
	// Branch is the feature branch to regate.
	Branch string
	// Env is the caller's environment, NAME=value entries (the CLI passes
	// os.Environ()); nil means os.Environ(). The scripts get it fixed by
	// gateScriptEnv; the git commands against origin, and the merge, its
	// abort, and its undo, get it with regateRemoteEnv's variables after it
	// (the merge also with GIT_MERGE_AUTOEDIT=no, regateMergeEnv), so the
	// merge commit has the caller's identity; the record's by= and
	// herdr_pane= come from its USER and HERDR_PANE_ID, as Gate's do.
	Env []string
}

// RegateResult is Regate's outcome when it returns no error. With an error
// once it has made its merge commit or come to the push, Regate returns it
// too, as far as it got.
type RegateResult struct {
	// Refused is set when Regate refused: the reason and one next step.
	// Nothing was recorded or pushed. A refusal before the scripts leaves
	// every other field empty, and the worktree and origin as they were: when
	// the merge could not make its merge commit, a merge it left in progress
	// was undone with git merge --abort (regateMergeTip). A refusal at the
	// record (the branch's binding or its org changed while the scripts ran,
	// recordRegate) keeps the other fields as the scripts left them, LogPath
	// included, with Outcome empty; after a merge, the merge commit was then
	// undone (Undone), or the refusal and NotUndone say why not.
	Refused string
	// Outcome is RegateOutcomeUpToDate, RegateOutcomeGated,
	// RegateOutcomeMerged, or RegateOutcomePushOnly; empty when Refused is
	// set.
	Outcome string
	// Passed reports a recorded gate_passed of Head (RegateOutcomeGated,
	// RegateOutcomeMerged), or, with RegateOutcomePushOnly, the earlier
	// gate_passed of Head that let Regate push it; false with Outcome
	// RegateOutcomeGated or RegateOutcomeMerged is a recorded gate_failed.
	Passed bool
	// Reason is the gate_failed reason (GateReason*), empty on a pass.
	Reason string
	// OrgID is the org of the branch's binding, which the record names;
	// Worktree is the binding's feature worktree; Branch is the branch.
	OrgID, Worktree, Branch string
	// Head is the head the gate checked, in this Regate or in the earlier
	// gate_passed that lets it stop or push only: HEAD, origin's tip of the
	// branch (RegateOutcomeUpToDate, RegateOutcomeGated); the merge commit
	// Regate made (RegateOutcomeMerged, and a refusal at the record after a
	// merge); or HEAD with origin's tip of the branch under it
	// (RegateOutcomePushOnly). Base is origin's default branch tip after the
	// fetch, which Head contains; BaseRef is
	// refs/remotes/origin/<default branch>.
	Head, Base, BaseRef string
	// Merge is the merge commit Regate made of Base into the branch (D3),
	// which Head names too, also once it was undone, and PreMergeHead the
	// HEAD it was made on, origin's tip of the branch; both are empty when
	// Regate made none.
	Merge, PreMergeHead string
	// Undone reports that Regate undid Merge with git reset --keep
	// PreMergeHead (D5), since nothing was recorded (the record was refused,
	// or Regate was interrupted or failed before it) or a static_failed or
	// test_failed was recorded. Merge is then reachable only from the reflog.
	Undone bool
	// NotUndone is set when Merge was kept after a gate_failed or with
	// nothing recorded: with head_moved and tree_changed (D5 leaves those
	// worktrees as they are), and when the undo could not run (HEAD was no
	// longer Merge, the tree was dirty, git reset --keep failed). It says how
	// the worktree is and the command that undoes the merge.
	NotUndone string
	// LogPath is the absolute path of the gate log under <state-dir>/gates/
	// (RegateOutcomeGated, RegateOutcomeMerged); with RegateOutcomePushOnly,
	// the log the pushed head's gate_passed names.
	LogPath string
	// Tail is the failing script's last gateTailLines output lines
	// (static_failed and test_failed).
	Tail []string
	// Dirty is `git status --porcelain --untracked-files=all` after the
	// scripts (tree_changed), one line per entry.
	Dirty []string
	// Pushed reports that Regate pushed Head to origin's branch without force
	// (a RegateOutcomeMerged pass, RegateOutcomePushOnly).
	Pushed bool
	// PushFailed is set when that push failed or ran past
	// regateRemoteTimeout: why, the end of git's stderr, and the next step,
	// running Regate again, which pushes only (D4). The record and Head are
	// kept as they are (D5).
	PushFailed string
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
//     branch (regateCheckOriginHead, D4), unless the branch's latest gate
//     record lets Regate push HEAD and nothing else (regateHeadNotOnOrigin:
//     RegateOutcomePushOnly, or a refusal that names the reset of a merge
//     commit an earlier Regate did not push).
//
// When HEAD contains origin's default branch tip and the branch's latest gate
// record is a readable gate_passed of HEAD on that tip with no corrupt line
// after it (regateBranch.passedOn), it returns RegateOutcomeUpToDate. When
// HEAD contains the tip otherwise, it writes a log under <state-dir>/gates/
// named after the binding's org (createGateLog) and runs the scripts as Gate
// does, without the manifest lock, reads HEAD and the tree again after a pass
// (gateCheckAfter), and records the outcome with via=regate under the
// manifest lock only when the branch's binding and its org are as the first
// check found them (recordRegate, D6): RegateOutcomeGated. When HEAD does not
// contain the tip, it first makes D3's merge commit of the tip, or refuses
// with the worktree as it was when git makes none (regateMergeTip, AC5), then
// runs and records the gate on the merge commit the same way and pushes a
// pass, or undoes the merge commit when nothing or a script failure was
// recorded (regateMerged: RegateOutcomeMerged). regate.lock is released when
// Regate returns, whatever it returns; its marker stays.
//
// The error is for what is not a refusal: the manifest, the lock, or a git
// command that fails, a log that cannot be written, a record that cannot be
// appended, and an interruption: when ctx is done, a running fetch, script,
// or push is stopped (a merge, its abort, and its undo are not: they finish
// first). Up to the moment the event is appended nothing is recorded, and the
// merge commit is undone (regateInterrupted, regateMerged); after it, the
// record and the merge commit are kept and nothing more is pushed
// (regatePush).
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
	t.base = gateBase{branch: t.base.branch, ref: "refs/remotes/origin/" + t.base.branch, sha: defaultTip}
	res := RegateResult{OrgID: b.orgID, Worktree: b.binding.Worktree, Branch: t.branch, Head: t.head, Base: t.base.sha, BaseRef: t.base.ref}
	if t.head != branchTip {
		return regateHeadNotOnOrigin(ctx, b, t, branchTip, env, words, res)
	}
	contains, err := regateIsAncestor(ctx, t.root, t.base.sha, t.head)
	switch {
	case err != nil:
		return RegateResult{}, regateInterrupted(ctx, err, "")
	case !contains:
		return o.regateMerged(ctx, b, t, env, words, res)
	case b.passedOn(t):
		res.Outcome = RegateOutcomeUpToDate
		return res, nil
	}
	if res, err = o.regateGate(ctx, b, t, env, res); err == nil && res.Refused == "" {
		res.Outcome = RegateOutcomeGated
	}
	return res, err
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

// passedOn reports whether b's latest gate record is a pass of t's HEAD on t's
// base tip, which lets Regate stop before the scripts (AC3) and, for a HEAD
// that is not origin's tip of the branch, push it and nothing else (D4): it is
// readable, no corrupt line follows it (either could hide a newer record), and
// it is a gate_passed of t's HEAD whose base is t's base tip and whose
// base_ref names t's default branch.
func (b regateBranch) passedOn(t gateTarget) bool {
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
// that talks to origin (or one regateUncancelledGit runs), with env and
// regateRemoteEnv's variables, its stdin empty, stopped after
// regateRemoteTimeout (D8); timedOut reports that the bound stopped it.
// WaitDelay closes the output pipes a child still holds after the kill or
// after git exited, so a child cannot hold the bound up; an exit 0 whose pipe
// a child held is a success, the way realMergeOrderExec reads it. The error
// is for a git that could not run or ctx done; a non-zero exit is not one.
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

// regateUncancelledGit runs git args in dir as regateRemoteGit does, with
// env, on a context ctx's cancellation does not reach (context.WithoutCancel),
// so only regateRemoteTimeout stops it: the merge, its abort, and its undo
// change the worktree, and one stopped half-way would leave it in neither
// state. Regate looks at ctx after it.
func regateUncancelledGit(ctx context.Context, dir string, env []string, args ...string) (gitResult, bool, error) {
	return regateRemoteGit(context.WithoutCancel(ctx), dir, env, args...)
}

// regateMergeEnv is env, copied, with GIT_MERGE_AUTOEDIT=no after it, so the
// merge opens no editor whatever the caller sets; regateUncancelledGit adds
// regateRemoteEnv's LC_ALL=C, so regateMergeTip reads git's messages
// untranslated (regateNoIdentity).
func regateMergeEnv(env []string) []string {
	return append(slices.Clip(env), "GIT_MERGE_AUTOEDIT=no")
}

// regateHeadNotOnOrigin is Regate's path for a HEAD that is not originHead,
// origin's tip of the branch after the fetch (D4). Regate gates only a head
// origin has, with one exception, the retry of a push that failed: when
// originHead is under HEAD and the branch's latest gate record is a pass of
// HEAD on t's base tip (regateBranch.passedOn), it pushes HEAD and does
// nothing else (regatePush, RegateOutcomePushOnly). Otherwise it refuses with
// regateCheckOriginHead's reason and regateResetHint's.
func regateHeadNotOnOrigin(ctx context.Context, b regateBranch, t gateTarget, originHead string, env []string, w gateWording,
	res RegateResult) (RegateResult, error) {
	if b.passedOn(t) {
		under, err := regateIsAncestor(ctx, t.root, originHead, t.head)
		if err != nil {
			return RegateResult{}, regateInterrupted(ctx, err, "")
		}
		if under {
			res.Outcome, res.Passed, res.LogPath = RegateOutcomePushOnly, true, b.gate.record.Log
			return regatePush(ctx, t, env, res)
		}
	}
	refused, err := regateCheckOriginHead(ctx, t, originHead, w)
	if err != nil {
		return RegateResult{}, regateInterrupted(ctx, err, "")
	}
	hint, err := regateResetHint(ctx, b, t, originHead, w)
	if err != nil {
		return RegateResult{}, regateInterrupted(ctx, err, "")
	}
	return RegateResult{Refused: refused + hint}, nil
}

// regateResetHint is what the refusal of a HEAD that is not originHead adds
// when HEAD is a merge commit an earlier Regate made and could not push (D4):
// b's latest gate record is readable with no corrupt line after it, is one
// Regate wrote (via=regate) of HEAD, and HEAD's first parent is originHead.
// The hint is the git reset --keep that drops that commit, and running Regate
// again, which merges anew. For any other HEAD it is "": the reset could drop
// commits nobody pushed, a leader's gated head among them.
func regateResetHint(ctx context.Context, b regateBranch, t gateTarget, originHead string, w gateWording) (string, error) {
	g := b.gate
	if g == nil || !g.readable || b.corruptAfterGate || g.record.Head != t.head || g.record.Via != gateViaRegate {
		return "", nil
	}
	parents, err := regateParents(ctx, t.root, t.head)
	if err != nil || len(parents) == 0 || parents[0] != originHead {
		return "", err
	}
	return fmt.Sprintf("; or, since HEAD is the merge commit an earlier regate made on origin's tip of the branch and did not "+
		"push, drop it with git -C %s reset --keep refs/remotes/origin/%s and %s, which merges origin's %s anew",
		t.root, t.branch, w.again, t.base.branch), nil
}

// regateParents returns commit's parents in root's repository, the first
// parent first (`git rev-list --parents -n 1`): none for a root commit.
func regateParents(ctx context.Context, root, commit string) ([]string, error) {
	out, err := gateGitOutput(ctx, root, "rev-list", "--parents", "-n", "1", commit)
	if err != nil {
		return nil, err
	}
	fields := strings.Fields(out)
	if len(fields) == 0 {
		return nil, fmt.Errorf("org: regate: git rev-list --parents -n 1 %s in %s printed nothing", commit, root)
	}
	return fields[1:], nil
}

// regateMerged is Regate's path for a HEAD that does not contain origin's
// default branch tip (AC4): it merges the tip in (regateMergeTip, D3), runs
// and records the gate on the merge commit (regateGate), and pushes a
// recorded gate_passed (regatePush). Until a gate_passed is recorded, the
// merge commit is undone (regateUndoMerge, D5): when Regate is interrupted
// before the scripts, when nothing is recorded (the record is refused, or
// Regate is interrupted or fails before the append), and after a recorded
// static_failed or test_failed. After a recorded head_moved or tree_changed
// it is kept, and res.NotUndone says how to undo it. An error or refusal
// after the merge says what became of the merge commit (regateUndoneText).
func (o *Org) regateMerged(ctx context.Context, b regateBranch, t gateTarget, env []string, w gateWording, res RegateResult) (RegateResult, error) {
	merge, refused, err := regateMergeTip(ctx, t, env, w)
	if err != nil {
		// The merge ran on a context the interrupt does not reach, so err
		// is not ctx's: it says how the worktree is.
		return RegateResult{}, err
	}
	if refused != "" {
		return RegateResult{Refused: refused}, nil
	}
	res.Head, res.Merge, res.PreMergeHead = merge, merge, t.head
	t.head = merge
	if ctx.Err() != nil {
		res = regateUndoMerge(ctx, t, env, res)
		return res, fmt.Errorf("%w; %s", regateInterrupted(ctx, ctx.Err(), ""), regateUndoneText(res))
	}

	res, err = o.regateGate(ctx, b, t, env, res)
	switch {
	case err != nil:
		res = regateUndoMerge(ctx, t, env, res)
		return res, fmt.Errorf("%w; %s", err, regateUndoneText(res))
	case res.Refused != "":
		res = regateUndoMerge(ctx, t, env, res)
		res.Refused += "; " + regateUndoneText(res)
		return res, nil
	}
	res.Outcome = RegateOutcomeMerged
	switch res.Reason {
	case "":
		return regatePush(ctx, t, env, res)
	case GateReasonStaticFailed, GateReasonTestFailed:
		return regateUndoMerge(ctx, t, env, res), nil
	case GateReasonHeadMoved:
		why := "HEAD moved off it while the scripts ran (head_moved), and git reset --keep would drop what came after it"
		if head, err := gateHead(context.WithoutCancel(ctx), t.root); err == nil {
			why = "HEAD moved off it to " + head + " while the scripts ran (head_moved), and git reset --keep would drop what came after it"
		}
		res.NotUndone = regateMergeLeft(t.root, res, why, "")
	default:
		res.NotUndone = regateMergeLeft(t.root, res, fmt.Sprintf("the scripts left %d uncommitted change(s) or untracked file(s) "+
			"in the worktree (tree_changed), which git reset --keep stops on or carries along", len(res.Dirty)), "")
	}
	return res, nil
}

// regateMergeTip merges t's base tip into the branch checked out in t.root
// with D3's merge commit, `git merge --no-ff --no-edit -m 'Merge <origin/name>
// <12 hex> into <branch>' <sha>`, run with env, so the commit has the
// caller's identity and the caller's hooks run, and GIT_MERGE_AUTOEDIT=no
// (regateMergeEnv), on a context ctx's cancellation does not reach
// (regateUncancelledGit). It returns the merge commit once it has checked
// that the commit's parents are HEAD and the tip. When git makes no merge
// commit (a conflict, a pre-merge-commit hook that fails, no user.name or
// user.email, a merge past regateRemoteTimeout), it undoes a merge left in
// progress with git merge --abort, checks that HEAD and the tree are as they
// were, and returns the refusal, which names the conflicting files for a
// conflict (AC5). The error is for a git that cannot run, a worktree it
// could not put back as it was, and a merge commit with other parents: it
// says how the worktree is and what restores it. Nothing is recorded then.
func regateMergeTip(ctx context.Context, t gateTarget, env []string, w gateWording) (merge, refused string, err error) {
	uncancelled := context.WithoutCancel(ctx)
	msg := fmt.Sprintf("Merge %s %s into %s", strings.TrimPrefix(t.base.ref, "refs/remotes/"), t.base.sha[:12], t.branch)
	command := fmt.Sprintf("git merge --no-ff --no-edit -m '%s' %s", msg, t.base.sha)
	r, timedOut, err := regateUncancelledGit(ctx, t.root, regateMergeEnv(env), "merge", "--no-ff", "--no-edit", "-m", msg, t.base.sha)
	if err != nil {
		return "", "", err
	}
	if !timedOut && r.code == 0 {
		head, err := gateHead(uncancelled, t.root)
		if err != nil {
			return "", "", err
		}
		parents, err := regateParents(uncancelled, t.root, head)
		if err != nil {
			return "", "", err
		}
		if !slices.Equal(parents, []string{t.head, t.base.sha}) {
			return "", "", fmt.Errorf("org: regate: %s in %s left HEAD at %s, whose parents are %q, not HEAD %s and origin's tip %s "+
				"of %s (a hook may have made another commit), so regate ran nothing and recorded nothing: check git -C %s log, "+
				"and to undo the merge run git -C %s reset --keep %s", command, t.root, head, parents, t.head, t.base.sha,
				t.base.branch, t.root, t.root, t.head)
		}
		return head, "", nil
	}

	// The conflicting files are read before the abort clears them; a failure
	// to read them only leaves them out of the refusal, and the checks after
	// the abort read the worktree again.
	files, _ := gateGitOutput(uncancelled, t.root, "diff", "--name-only", "--diff-filter=U")
	merging, err := gateGitPathExists(uncancelled, t.root, "a merge", "MERGE_HEAD")
	if err != nil {
		return "", "", regateMergeNotBack(t, command, fmt.Sprintf("whether a merge is in progress cannot be read (%v)", err))
	}
	aborted := ""
	if merging {
		ar, abortTimedOut, err := regateUncancelledGit(ctx, t.root, env, "merge", "--abort")
		switch {
		case err != nil:
			return "", "", regateMergeNotBack(t, command, fmt.Sprintf("a merge is in progress, and git merge --abort could not run (%v)", err))
		case abortTimedOut || ar.code != 0:
			return "", "", regateMergeNotBack(t, command, "a merge is in progress, and git merge --abort failed"+regateStderrTail(ar.stderr))
		}
		aborted = " (git merge --abort undid the merge it left in progress)"
	}
	head, err := gateHead(uncancelled, t.root)
	if err != nil {
		return "", "", regateMergeNotBack(t, command, fmt.Sprintf("HEAD cannot be read (%v)", err))
	}
	dirty, err := gateDirtyLines(uncancelled, t.root)
	if err != nil {
		return "", "", regateMergeNotBack(t, command, fmt.Sprintf("the worktree cannot be read (%v)", err))
	}
	if head != t.head || len(dirty) > 0 {
		return "", "", regateMergeNotBack(t, command, fmt.Sprintf("HEAD is %s (it was %s), and the worktree has %d uncommitted "+
			"change(s) or untracked file(s)", head, t.head, len(dirty)))
	}

	said := r.stderr
	if said == "" {
		said = r.stdout
	}
	switch conflicted := strings.Fields(files); {
	case timedOut:
		return "", w.prefix + fmt.Sprintf("%s in %s ran past %s and was stopped%s, so nothing was recorded: check the hooks a "+
			"merge runs there (pre-merge-commit, prepare-commit-msg, commit-msg) and %s%s", command, t.root, regateRemoteTimeout,
			aborted, w.again, regateStderrTail(said)), nil
	case len(conflicted) > 0:
		return "", w.prefix + fmt.Sprintf("merging origin's tip %s of %s into %s in %s conflicts in %s, and regate does not "+
			"resolve conflicts%s, so nothing was recorded: merge it yourself (git -C %s merge %s), resolve the conflicts, "+
			"commit, push, and %s", t.base.sha, t.base.branch, t.branch, t.root, strings.Join(conflicted, ", "), aborted, t.root,
			t.base.ref, w.again), nil
	case strings.Contains(r.stderr, regateNoIdentity):
		return "", w.prefix + fmt.Sprintf("%s in %s could not make the merge commit, since git has no user.name and user.email "+
			"to make it with%s, so nothing was recorded: set them (git -C %s config user.name <name> and git -C %s config "+
			"user.email <email>, or in your global git config) and %s%s", command, t.root, aborted, t.root, t.root, w.again,
			regateStderrTail(said)), nil
	}
	return "", w.prefix + fmt.Sprintf("%s in %s made no merge commit (exit %d)%s, so nothing was recorded: a hook such as "+
		"pre-merge-commit may have stopped it; fix what git said and %s%s", command, t.root, r.code, aborted, w.again,
		regateStderrTail(said)), nil
}

// regateMergeNotBack is regateMergeTip's error for a merge that made no merge
// commit and left the worktree in state, not as it was: nothing was
// recorded, and what puts the worktree back.
func regateMergeNotBack(t gateTarget, command, state string) error {
	return fmt.Errorf("org: regate: %s in %s made no merge commit, and the worktree is not as it was before: %s; nothing was "+
		"recorded: check it with git -C %s status, undo a merge in progress with git -C %s merge --abort, put HEAD back with "+
		"git -C %s reset --keep %s, and run ralph org regate --branch %s again", command, t.root, state, t.root, t.root, t.root,
		t.head, t.branch)
}

// regateUndoMerge undoes res.Merge, the merge commit Regate made on
// res.PreMergeHead in t.root (D5), with git reset --keep res.PreMergeHead,
// and sets res.Undone. It undoes only a HEAD that is still res.Merge in a
// clean tree: a commit made after it would go with it, and reset --keep stops
// on a dirty tree's changes or carries them along. Its git commands run on a
// context ctx's cancellation does not reach, so an interrupt does not stop
// the undo half-way (regateUncancelledGit). When it does not undo, it sets
// res.NotUndone (regateMergeLeft).
func regateUndoMerge(ctx context.Context, t gateTarget, env []string, res RegateResult) RegateResult {
	uncancelled := context.WithoutCancel(ctx)
	head, err := gateHead(uncancelled, t.root)
	if err != nil {
		res.NotUndone = regateMergeLeft(t.root, res, fmt.Sprintf("HEAD cannot be read (%v)", err), "")
		return res
	}
	if head != res.Merge {
		res.NotUndone = regateMergeLeft(t.root, res, "HEAD is now "+head+", not the merge commit", "")
		return res
	}
	dirty, err := gateDirtyLines(uncancelled, t.root)
	switch {
	case err != nil:
		res.NotUndone = regateMergeLeft(t.root, res, fmt.Sprintf("the worktree cannot be read (%v)", err), "")
		return res
	case len(dirty) > 0:
		res.NotUndone = regateMergeLeft(t.root, res, fmt.Sprintf("the worktree has %d uncommitted change(s) or untracked file(s)", len(dirty)), "")
		return res
	}
	r, timedOut, err := regateUncancelledGit(ctx, t.root, env, "reset", "--keep", res.PreMergeHead)
	switch {
	case err != nil:
		res.NotUndone = regateMergeLeft(t.root, res, fmt.Sprintf("git reset --keep %s could not run (%v)", res.PreMergeHead, err), "")
	case timedOut:
		res.NotUndone = regateMergeLeft(t.root, res, fmt.Sprintf("git reset --keep %s ran past %s and was stopped", res.PreMergeHead,
			regateRemoteTimeout), regateStderrTail(r.stderr))
	case r.code != 0:
		res.NotUndone = regateMergeLeft(t.root, res, fmt.Sprintf("git reset --keep %s failed (exit %d)", res.PreMergeHead, r.code),
			regateStderrTail(r.stderr))
	default:
		res.Undone = true
	}
	return res
}

// regateMergeLeft is res.NotUndone for res.Merge kept because of why: how to
// look at the worktree and the git reset --keep that undoes the merge, then
// tail (regateStderrTail).
func regateMergeLeft(root string, res RegateResult, why, tail string) string {
	return fmt.Sprintf("the merge commit %s was not undone, since %s: check the worktree (git -C %s status, git -C %s log), "+
		"and to undo the merge run git -C %s reset --keep %s, which also drops every commit made after it%s",
		res.Merge, why, root, root, root, res.PreMergeHead, tail)
}

// regateUndoneText says what became of res.Merge when Regate was to undo it
// (regateUndoMerge): undone, or res.NotUndone.
func regateUndoneText(res RegateResult) string {
	if res.Undone {
		return fmt.Sprintf("the merge commit %s was undone (git reset --keep %s), so only the reflog has it now", res.Merge,
			res.PreMergeHead)
	}
	return res.NotUndone
}

// regatePush pushes t's HEAD, the head of a gate_passed, to origin's branch
// without force (`git push origin <sha>:refs/heads/<branch>`, regateRemoteGit,
// D8), sets res.Pushed, and then points refs/remotes/origin/<branch> at it
// when the push has not (regateSetTrackingRef). A push that fails or runs past
// regateRemoteTimeout sets res.PushFailed. The record and HEAD stay as they
// are either way (D5): running Regate again finds the push landed (AC3) or
// pushes only (D4). The error is ctx done before or during the push
// (regatePushInterrupted), a git that cannot run, or a remote-tracking ref
// that cannot be updated after the push.
func regatePush(ctx context.Context, t gateTarget, env []string, res RegateResult) (RegateResult, error) {
	if ctx.Err() != nil {
		return res, regatePushInterrupted(ctx, t, false)
	}
	spec := t.head + ":refs/heads/" + t.branch
	command := "git push origin " + spec
	kept := fmt.Sprintf("the gate_passed record of %s and HEAD are kept: check origin (git remote get-url origin), its "+
		"credentials, and whether someone pushed to %s after the fetch, and run ralph org regate --branch %s again, which "+
		"pushes only, or says what changed", t.head, t.branch, t.branch)
	r, timedOut, err := regateRemoteGit(ctx, t.root, env, "push", "origin", spec)
	switch {
	case err != nil && ctx.Err() != nil:
		return res, regatePushInterrupted(ctx, t, true)
	case err != nil:
		return res, fmt.Errorf("%w; %s", err, kept)
	case timedOut:
		res.PushFailed = fmt.Sprintf("org: regate: %s in %s ran past %s and was stopped, so origin may not have it: %s%s",
			command, t.root, regateRemoteTimeout, kept, regateStderrTail(r.stderr))
		return res, nil
	case r.code != 0:
		res.PushFailed = fmt.Sprintf("org: regate: %s in %s failed (exit %d), so origin's %s is not %s: %s%s",
			command, t.root, r.code, t.branch, t.head, kept, regateStderrTail(r.stderr))
		return res, nil
	}
	res.Pushed = true
	return res, regateSetTrackingRef(context.WithoutCancel(ctx), t)
}

// regatePushInterrupted is regatePush's error for ctx done before the push or
// while it ran (during): the record and HEAD are kept, and running Regate
// again pushes only, or finds the push landed.
func regatePushInterrupted(ctx context.Context, t gateTarget, during bool) error {
	when := fmt.Sprintf("before %s was pushed to origin's %s, so nothing was pushed", t.head, t.branch)
	if during {
		when = fmt.Sprintf("while %s was pushed to origin's %s, so origin may or may not have it", t.head, t.branch)
	}
	return fmt.Errorf("org: regate: interrupted (%v) %s: the gate_passed record of it and HEAD are kept; run ralph org regate "+
		"--branch %s again, which pushes it, or finds it pushed", context.Cause(ctx), when, t.branch)
}

// regateSetTrackingRef points refs/remotes/origin/<branch> at t's HEAD, which
// Regate pushed to origin's branch, when the push has not: git updates it
// only when remote.origin.fetch maps the branch there, which Regate does not
// count on. It updates the ref from the value it read (`git update-ref`
// with the old value), so a fetch in between is not overwritten.
func regateSetTrackingRef(ctx context.Context, t gateTarget) error {
	ref := "refs/remotes/origin/" + t.branch
	r, err := gateGit(ctx, t.root, "rev-parse", "--verify", "--quiet", ref)
	if err != nil {
		return fmt.Errorf("%w; %s was pushed to origin's %s", err, t.head, t.branch)
	}
	old := strings.TrimSpace(r.stdout)
	if old == t.head {
		return nil
	}
	if _, err := gateGitOutput(ctx, t.root, "update-ref", ref, t.head, old); err != nil {
		return fmt.Errorf("%w; %s was pushed to origin's %s, and git fetch origin %s brings %s up to date", err, t.head, t.branch,
			t.branch, ref)
	}
	return nil
}

// regateGate runs the gate on t's HEAD, origin's tip of the branch (AC9) or
// the merge commit Regate made (regateMerged): it runs the scripts on t as
// Gate does, with a log of b's org, and records the outcome with via=regate
// (recordRegate). res holds what the checks found; the scripts' outcome is
// added to it, and Passed once a gate_passed is recorded. Nothing is recorded
// when it returns an error or sets res.Refused.
func (o *Org) regateGate(ctx context.Context, b regateBranch, t gateTarget, env []string, res RegateResult) (RegateResult, error) {
	logPath, logFile, err := o.createGateLog(b.orgID)
	if err != nil {
		return res, err
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
	res.Passed = res.Reason == ""
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
