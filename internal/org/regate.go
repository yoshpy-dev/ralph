package org

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
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
	// (AC3), whose log RegateResult.LogPath names. Nothing was run, recorded,
	// or pushed.
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
	// regateGitWaitDelay is regateBoundedGit's exec.Cmd.WaitDelay: how long
	// after the kill, or after git exited, it waits for the output pipes,
	// which a child (ssh, a credential helper, a hook) can hold.
	regateGitWaitDelay = 2 * time.Second
	// regateStderrTailLines is how many of git's last stderr lines a refused
	// fetch, merge, or push quotes.
	regateStderrTailLines = 5
	// regateRemoteNoRef is what git prints on stderr when origin has no ref
	// a fetch names (LC_ALL=C, regateFetchEnv).
	regateRemoteNoRef = "couldn't find remote ref"
)

// regateGitTimeout bounds each git command Regate runs with
// regateBoundedGit: the fetches and the push against origin (D8), the check
// of the identity the merge commit needs, and the merge, its abort, and its
// undo (regateUncancelledGit). Tests shorten it.
var regateGitTimeout = 120 * time.Second

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
	// gateScriptEnv. The git commands get it with only what Regate needs
	// after it: the fetches GIT_TERMINAL_PROMPT=0 and LC_ALL=C
	// (regateFetchEnv), the push GIT_TERMINAL_PROMPT=0 (regateRemoteEnv), the
	// merge and the git merge-tree that checks what it writes
	// (regateMergeWrites) GIT_MERGE_AUTOEDIT=no (regateMergeEnv), and the
	// identity check, the merge's abort, and its undo nothing, so the merge
	// commit has the caller's identity and the merge's hooks run in the
	// caller's locale; the record's by= and herdr_pane= come from its USER and
	// HERDR_PANE_ID, as Gate's do.
	Env []string
	// Stderr gets the one line Regate writes when ctx is done while a git it
	// does not stop runs, the merge, its abort, or its undo: that it waits
	// for that git to finish (regateUncancelledGit). nil uses os.Stderr.
	Stderr io.Writer
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
	// (RegateOutcomeGated, RegateOutcomeMerged); with RegateOutcomeUpToDate
	// and RegateOutcomePushOnly, the log the earlier gate_passed of Head
	// names.
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
	// regateGitTimeout: why, the end of git's stderr, and the next step,
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
// with the worktree as it was when something ignored is in the merge's way
// (regateCheckInTheWay, D3) or git makes none (regateMergeTip, AC5), then
// runs and records the gate on the merge commit the same way and pushes a
// pass, or undoes the merge commit when nothing or a script failure was
// recorded (regateMerged: RegateOutcomeMerged). regate.lock is released when
// Regate returns, whatever it returns; its marker stays.
//
// The error is for what is not a refusal: the manifest, the lock, or a git
// command that fails, a log that cannot be written, a record that cannot be
// appended, and an interruption: when ctx is done, a running fetch, script,
// or push is stopped (a merge, its abort, and its undo are not: they finish
// first, in a process group of their own, which the Ctrl-C or the hangup a
// terminal sends its foreground group does not reach either, and p.Stderr
// gets a line that says Regate waits for one, regateUncancelledGit). Up to
// the moment the event is appended nothing is recorded, and the merge commit
// is undone (regateInterrupted, regateMerged); after it, the record and the
// merge commit are kept and nothing more is pushed (regatePush).
func (o *Org) Regate(ctx context.Context, p RegateParams) (RegateResult, error) {
	if p.Branch == "" {
		return RegateResult{Refused: "org: regate: no branch named: name the feature branch to regate (ralph org merge-order " +
			"lists the ledger's branches)"}, nil
	}
	env := p.Env
	if env == nil {
		env = os.Environ()
	}
	stderr := p.Stderr
	if stderr == nil {
		stderr = os.Stderr
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
		return o.regateMerged(ctx, b, t, env, stderr, words, res)
	case b.passedOn(t):
		res.Outcome, res.LogPath = RegateOutcomeUpToDate, b.gate.record.Log
		return res, nil
	}
	if res, err = o.regateGate(ctx, b, t, env, res); err == nil && res.Refused == "" {
		res.Outcome = RegateOutcomeGated
	}
	return res, err
}

// regatePrefix starts every error Regate returns, including those of the git
// helpers it shares with Gate (gateGit and the helpers built on it).
const regatePrefix = "org: regate: "

// regateWords is Regate's wording of the worktree checks it shares with
// Gate (gateWording), for branch: a commit it asks for is to be pushed too,
// since Regate refuses a HEAD that origin's tip of the branch is not (D4). A
// stopped rebase is only to be undone: a finished one rewrites the branch, so
// origin would take it only with a force push, which Regate does not do (D3).
func regateWords(branch string) gateWording {
	return gateWording{prefix: regatePrefix, again: "run ralph org regate --branch " + branch + " again", verb: "ralph org regate",
		afterCommit: ", push the commit to origin's " + branch,
		rebaseSteps: "undo it with git rebase --abort (regate merges origin's default branch instead, and a finished rebase " +
			"would need a force push)"}
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
	// runsOn: running, and orgID's latest complete binding is of another
	// branch (mergeTarget.orgRunsOn): that branch. "" otherwise.
	runsOn string
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
			b.orgID, b.pos, b.running, b.runsOn = t.binding.OrgID, t.bindingPos, t.orgRunning, t.orgRunsOn
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
// names or whose org runs; for an org that runs another feature now
// (regateBranch.runsOn), the refusal names that feature's branch instead of
// the leader's ralph org gate. When the checks pass it writes lock's marker
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
		case b.running && b.runsOn != "":
			refused = fmt.Sprintf("org: regate: the branch %s is bound to org_id %q, which is running, but now for another feature, "+
				"on the branch %s: wait for the org to be disbanded and run ralph org regate --branch %s then", branch, b.orgID,
				b.runsOn, branch)
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
	r, err := gateGit(ctx, w.prefix, f.Worktree, "rev-parse", "--show-toplevel")
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
	if refused, err = regateCheckInProgress(ctx, root, branch, w); err != nil || refused != "" {
		return gateTarget{}, refused, err
	}
	dirty, err := gateDirtyLines(ctx, w.prefix, root)
	if err != nil {
		return gateTarget{}, "", err
	}
	if len(dirty) > 0 {
		return gateTarget{}, gateDirtyRefusal(root, dirty, w), nil
	}
	t := gateTarget{root: root, branch: branch}
	if t.head, err = gateHead(ctx, w.prefix, root); err != nil {
		return gateTarget{}, "", err
	}
	base, found, err := resolveGateBase(ctx, w.prefix, root)
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
// with a detached HEAD has been refused by gateCheckBranch already. A merge
// finished with git commit is to be pushed to origin's branch too, since
// Regate gates only what origin has (D4); Gate has no such check, so the
// wording is Regate's alone.
func regateCheckInProgress(ctx context.Context, root, branch string, w gateWording) (string, error) {
	rebasing, err := gateRebaseInProgress(ctx, w.prefix, root)
	if err != nil {
		return "", err
	}
	if rebasing {
		return w.prefix + fmt.Sprintf("a rebase is in progress in %s: %s, and %s", root, w.rebaseSteps, w.again), nil
	}
	merging, err := gateGitPathExists(ctx, w.prefix, root, "a merge", "MERGE_HEAD")
	if err != nil {
		return "", err
	}
	if merging {
		return w.prefix + fmt.Sprintf("a merge is in progress in %s: finish it with git commit and push the commit to origin's "+
			"%s, or undo it with git merge --abort, and %s", root, branch, w.again), nil
	}
	return "", nil
}

// regateGate runs the gate on t's HEAD, origin's tip of the branch (AC9) or
// the merge commit Regate made (regateMerged): it runs the scripts on t as
// Gate does, with a log of b's org, and records the outcome with via=regate
// (recordRegate). res holds what the checks found; the scripts' outcome is
// added to it, and Passed once a gate_passed is recorded. Nothing is recorded
// when it returns an error or sets res.Refused.
func (o *Org) regateGate(ctx context.Context, b regateBranch, t gateTarget, env []string, res RegateResult) (RegateResult, error) {
	w := regateWords(t.branch)
	logPath, logFile, err := o.createGateLog(b.orgID, w)
	if err != nil {
		return res, err
	}
	res.LogPath = logPath
	run, err := runGateScripts(ctx, t, gateScriptEnv(env), logFile, w)
	if closeErr := logFile.Close(); err == nil && closeErr != nil {
		err = fmt.Errorf("org: regate: write the log %s: %w", logPath, closeErr)
	}
	if err != nil {
		return res, regateInterrupted(ctx, err, logPath)
	}
	res.Reason, res.Tail = run.failed, run.tail
	if res.Reason == "" {
		if res.Reason, res.Dirty, err = gateCheckAfter(ctx, regatePrefix, t); err != nil {
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
