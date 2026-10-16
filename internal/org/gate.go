package org

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
)

// The `ralph org gate` verb (Gate), stage 7b of plan
// docs/plans/active/2026-10-13-org-gate.md (FR-7 of
// docs/specs/2026-10-07-org-multi-org-director.md). A feature org's leader
// rebases its branch on the default branch and runs the gate in the feature
// worktree, on the head it pushes, before it opens the PR. The gate checks
// the org's binding, the worktree, and that HEAD contains the default
// branch's tip, runs scripts/run-static-verify.sh and then
// scripts/run-test.sh there (gate_run.go), and records the outcome with the
// head and base SHAs as one org-level gate_passed or gate_failed event
// (gateRecord). Stage 7c's merge order reads that record.
//
// A refusal before the scripts run records nothing and writes no log (B7).
// The gate never fetches, pushes, or calls herdr or agmsg. Neither the watch
// gate (watchgate.go) nor the approval check (approval.go) applies to it,
// since it starts no new autonomous work (AC7).

const (
	// GateStaticScript and GateTestScript are the scripts Gate runs, in this
	// order, relative to the feature worktree's root.
	GateStaticScript = "scripts/run-static-verify.sh"
	GateTestScript   = "scripts/run-test.sh"
	// GateDirtyListLines caps the porcelain lines a dirty-tree refusal
	// lists; the CLI caps its tree_changed list the same.
	GateDirtyListLines = 20
)

// The reasons a gate_failed records, first in its Details.
const (
	// GateReasonStaticFailed: scripts/run-static-verify.sh exited non-zero,
	// so scripts/run-test.sh did not run.
	GateReasonStaticFailed = "static_failed"
	// GateReasonTestFailed: scripts/run-test.sh exited non-zero.
	GateReasonTestFailed = "test_failed"
	// GateReasonHeadMoved: both scripts passed, but HEAD was another commit
	// after them than before.
	GateReasonHeadMoved = "head_moved"
	// GateReasonTreeChanged: both scripts passed on the same HEAD, but the
	// worktree had changes or untracked files after them (B6).
	GateReasonTreeChanged = "tree_changed"
)

// gitObjectPattern is the shape of a commit SHA the gate records: 40
// lowercase hex digits, or 64 in a SHA-256 repository.
var gitObjectPattern = regexp.MustCompile(`^[0-9a-f]{40}([0-9a-f]{24})?$`)

// The keys of an EventGatePassed's or EventGateFailed's Details, written in
// this order (reason only on gate_failed, via only on a record ralph org
// regate wrote), each as one space-separated `key=value` token. log comes
// last because its value is strconv.Quote'd and may hold spaces, the rule of
// plan_approved's cwd (planApproval.details): gateRecordFromDetails reads
// everything after " log=" as the log, so a key written after it would be
// lost.
const (
	gateRecordReasonKey    = "reason"
	gateRecordHeadKey      = "head"
	gateRecordBaseKey      = "base"
	gateRecordBaseRefKey   = "base_ref"
	gateRecordBranchKey    = "branch"
	gateRecordByKey        = "by"
	gateRecordHerdrPaneKey = "herdr_pane"
	gateRecordViaKey       = "via"
	gateRecordLogKey       = "log"
)

// gateViaRegate is the via= of a gate record ralph org regate wrote
// (regate.go); a record ralph org gate writes has no via=.
const gateViaRegate = "regate"

// GateParams describes one `ralph org gate` call.
type GateParams struct {
	// OrgID is the feature org whose worktree the gate checks.
	OrgID string
	// Dir is the directory the gate was run in, the feature worktree or a
	// directory under it; empty means os.Getwd().
	Dir string
	// Env is the caller's environment, NAME=value entries (the CLI passes
	// os.Environ()); nil means os.Environ(). The scripts get it without the
	// RALPH_VERIFY_* and HARNESS_VERIFY_* variables and with
	// RALPH_VERIFY_SCOPE=full (gateScriptEnv). The record's by= and
	// herdr_pane= come from its USER and HERDR_PANE_ID.
	Env []string
}

// GateResult is Gate's outcome when it returns no error.
type GateResult struct {
	// Refused is set when the gate refused: the reason and one next step.
	// Nothing was recorded. A refusal before the scripts leaves every other
	// field empty, since no script ran and no log was written. A refusal at
	// the record (the org changed while the scripts ran, recordGate) keeps
	// the other fields as the scripts left them, LogPath included.
	Refused string
	// Passed reports a recorded gate_passed; false with Refused empty is a
	// recorded gate_failed.
	Passed bool
	// Reason is the gate_failed reason (GateReason*), empty on a pass.
	Reason string
	// Head is HEAD before the scripts ran; Base is BaseRef's tip, which
	// HEAD contains; BaseRef is refs/remotes/origin/<default branch> or
	// refs/heads/<default branch> (resolveGateBase); Branch is the feature
	// branch.
	Head, Base, BaseRef, Branch string
	// LogPath is the absolute path of the log under <state-dir>/gates/
	// holding both scripts' output.
	LogPath string
	// Tail is the failing script's last gateTailLines output lines
	// (static_failed and test_failed).
	Tail []string
	// Dirty is `git status --porcelain --untracked-files=all` after the
	// scripts (tree_changed), one line per entry.
	Dirty []string
}

// gateRecord is one EventGatePassed or EventGateFailed: what the gate ran on
// and who ran it.
type gateRecord struct {
	Reason  string // GateReason*; "" on gate_passed
	Head    string // HEAD before the scripts ran
	Base    string // the base ref's tip
	BaseRef string // refs/remotes/origin/<name> or refs/heads/<name>
	Branch  string // the feature branch
	By      string // USER, or approverUnknown (approverName)
	// HerdrPane: HERDR_PANE_ID was set in the caller's environment.
	HerdrPane bool
	// Via is the verb that ran the gate when it is not ralph org gate
	// (gateViaRegate), "" for ralph org gate.
	Via string
	Log string // the absolute path of the gate log
}

// event is the manifest event r is recorded as: gate_passed when Reason is
// empty, gate_failed otherwise. A reader pairs it with the event's name.
func (r gateRecord) event() string {
	if r.Reason == "" {
		return EventGatePassed
	}
	return EventGateFailed
}

// details is the Details of r's event (see the gateRecord*Key constants).
func (r gateRecord) details() string {
	pane := "unset"
	if r.HerdrPane {
		pane = "set"
	}
	var d string
	if r.Reason != "" {
		d = gateRecordReasonKey + "=" + r.Reason + " "
	}
	d += gateRecordHeadKey + "=" + r.Head +
		" " + gateRecordBaseKey + "=" + r.Base +
		" " + gateRecordBaseRefKey + "=" + r.BaseRef +
		" " + gateRecordBranchKey + "=" + r.Branch +
		" " + gateRecordByKey + "=" + r.By +
		" " + gateRecordHerdrPaneKey + "=" + pane
	if r.Via != "" {
		d += " " + gateRecordViaKey + "=" + r.Via
	}
	return d + " " + gateRecordLogKey + "=" + strconv.Quote(r.Log)
}

// gateRecordFromDetails reads a gate event's Details back (stage 7c reads
// the records with it). ok is false when a key before log is given twice or a
// token is not `key=value`, when head or base is not a commit SHA
// (gitObjectPattern), base_ref is not under refs/, branch is empty, or reason
// is set to anything but a GateReason* value. Keys it does not know are
// skipped. by, herdr_pane, via and log are read on a best-effort basis: they
// are for an audit. A record whose Reason is empty is a gate_passed one
// (gateRecord.event); a reader checks that against the event's name.
func gateRecordFromDetails(details string) (r gateRecord, ok bool) {
	head, logValue, hasLog := strings.Cut(details, " "+gateRecordLogKey+"=")
	values := make(map[string]string)
	for _, token := range strings.Fields(head) {
		key, value, found := strings.Cut(token, "=")
		if _, dup := values[key]; !found || dup {
			return gateRecord{}, false
		}
		values[key] = value
	}
	r = gateRecord{
		Reason: values[gateRecordReasonKey], Head: values[gateRecordHeadKey], Base: values[gateRecordBaseKey],
		BaseRef: values[gateRecordBaseRefKey], Branch: values[gateRecordBranchKey], By: values[gateRecordByKey],
		HerdrPane: values[gateRecordHerdrPaneKey] == "set", Via: values[gateRecordViaKey],
	}
	if _, hasReason := values[gateRecordReasonKey]; hasReason && !isGateReason(r.Reason) {
		return gateRecord{}, false
	}
	if !gitObjectPattern.MatchString(r.Head) || !gitObjectPattern.MatchString(r.Base) ||
		!strings.HasPrefix(r.BaseRef, "refs/") || r.Branch == "" {
		return gateRecord{}, false
	}
	if hasLog {
		if s, err := strconv.Unquote(logValue); err == nil {
			r.Log = s
		}
	}
	return r, true
}

// isGateReason reports whether reason is one of the GateReason* values.
func isGateReason(reason string) bool {
	switch reason {
	case GateReasonStaticFailed, GateReasonTestFailed, GateReasonHeadMoved, GateReasonTreeChanged:
		return true
	}
	return false
}

// Gate runs `ralph org gate` for p.OrgID. It checks, in this order, and
// refuses (GateResult.Refused, nothing run or recorded, no log) when a check
// fails:
//
//   - the org's current life holds a complete split plan feature binding
//     (ActiveFeature) and the org is running (RunningOrgs);
//   - p.Dir is in the binding's worktree (`git rev-parse --show-toplevel`,
//     both sides with symlinks resolved), on the binding's branch (`git
//     branch --show-current`; a detached HEAD is refused, with its own next
//     step when a rebase is in progress: gateCheckBranch);
//   - the worktree is clean: `git status --porcelain --untracked-files=all`
//     prints nothing (ignored files do not count);
//   - a default branch is found and its base ref's tip is an ancestor of
//     HEAD (resolveGateBase; the gate never fetches);
//   - both scripts exist in the worktree's root as executable regular files.
//
// It then writes a log under <state-dir>/gates/ (createGateLog) and runs
// ./scripts/run-static-verify.sh and, when that exits 0,
// ./scripts/run-test.sh in the worktree's root, with the caller's
// environment fixed by gateScriptEnv, each in its own process group, which
// is stopped when the script exits and leaves processes in it
// (runGateScript). The manifest lock is not held while they run (AC5). When
// both pass it reads HEAD and the worktree again: another HEAD is
// head_moved, a dirty worktree tree_changed. Under the manifest lock it
// reads the manifest again and records one org-level event only when the org
// still has the same binding in the same life (recordGate), so a disband or
// a new start during the run records nothing and refuses.
//
// The error is for what is not a refusal: the manifest or a git command that
// fails, a log that cannot be written, a record that cannot be appended, and
// an interruption: when ctx is done, the running script's process group is
// stopped and nothing is recorded. ctx done after the scripts finished, while
// Gate reads the worktree again or waits for the manifest lock, also records
// nothing, up to the moment the event is appended (recordGate).
func (o *Org) Gate(ctx context.Context, p GateParams) (GateResult, error) {
	if err := ValidateIdentifier("org_id", p.OrgID); err != nil {
		return GateResult{Refused: "org: gate: " + strings.TrimPrefix(err.Error(), "org: ")}, nil
	}
	env := p.Env
	if env == nil {
		env = os.Environ()
	}
	dir := p.Dir
	if dir == "" {
		wd, err := os.Getwd()
		if err != nil {
			return GateResult{}, fmt.Errorf("org: gate: read the working directory: %w", err)
		}
		dir = wd
	}

	rr, err := o.Manifest.Read()
	if err != nil {
		return GateResult{}, fmt.Errorf("org: gate: read manifest: %w", err)
	}
	life, refused := gateOrgLife(rr.Events, p.OrgID)
	if refused != "" {
		return GateResult{Refused: refused}, nil
	}
	target, refused, err := gateCheckWorktree(ctx, p.OrgID, dir, life.binding)
	if err != nil {
		return GateResult{}, gateInterrupted(ctx, err, "")
	}
	if refused != "" {
		return GateResult{Refused: refused}, nil
	}

	logPath, logFile, err := o.createGateLog(p.OrgID, gateWords)
	if err != nil {
		return GateResult{}, err
	}
	res := GateResult{Head: target.head, Base: target.base.sha, BaseRef: target.base.ref, Branch: target.branch, LogPath: logPath}
	run, err := runGateScripts(ctx, target, gateScriptEnv(env), logFile, gateWords)
	if closeErr := logFile.Close(); err == nil && closeErr != nil {
		err = fmt.Errorf("org: gate: write the log %s: %w", logPath, closeErr)
	}
	if err != nil {
		return res, gateInterrupted(ctx, err, logPath)
	}
	res.Reason, res.Tail = run.failed, run.tail
	if res.Reason == "" {
		if res.Reason, res.Dirty, err = gateCheckAfter(ctx, gateWords.prefix, target); err != nil {
			return res, gateInterrupted(ctx, err, logPath)
		}
	}

	record := gateRecord{
		Reason: res.Reason, Head: res.Head, Base: res.Base, BaseRef: res.BaseRef, Branch: res.Branch,
		By: approverName(gateEnvValue(env, "USER")), HerdrPane: gateEnvValue(env, herdrPaneIDEnv) != "", Log: logPath,
	}
	if res.Refused, err = o.recordGate(ctx, p.OrgID, life, record); err != nil {
		return res, gateInterrupted(ctx, err, logPath)
	}
	res.Passed = res.Refused == "" && res.Reason == ""
	return res, nil
}

// gateInterrupted is the error Gate returns for err: when ctx is done, one
// that says the gate was interrupted and recorded nothing, and names the log
// when the scripts had started (logPath set); err itself otherwise.
func gateInterrupted(ctx context.Context, err error, logPath string) error {
	if ctx.Err() == nil {
		return err
	}
	if logPath == "" {
		return fmt.Errorf("org: gate: interrupted (%v) before the scripts ran: nothing was recorded", context.Cause(ctx))
	}
	return fmt.Errorf("org: gate: interrupted (%v): the scripts were stopped and nothing was recorded; the output so far is in %s",
		context.Cause(ctx), logPath)
}

// gateLife is the org a gate checks, as one read of the manifest finds it:
// its binding, and the index of its latest real `disbanded` (-1 for none),
// which tells a later life with the same binding apart.
type gateLife struct {
	binding     *FeatureBinding
	lastDisband int
}

// gateOrgLife reads orgID's binding from events and returns it, or the
// refusal of an org that holds none, holds an incomplete one, or is not
// running.
func gateOrgLife(events []ManifestEvent, orgID string) (gateLife, string) {
	life := gateLife{lastDisband: -1}
	if d, ok := lastDisbandedIndexes(events)[orgID]; ok {
		life.lastDisband = d
	}
	running := false
	for _, id := range RunningOrgs(events) {
		running = running || id == orgID
	}
	f := ActiveFeature(events, orgID)
	switch {
	case f == nil && running:
		return life, fmt.Sprintf("org: gate: org_id %q holds no split plan feature binding (ralph org start <task> or a promoted "+
			"session's leader started it): the gate checks only an org that ralph org start --plan started, so run "+
			"./%s and ./%s yourself", orgID, GateStaticScript, GateTestScript)
	case f == nil && life.lastDisband >= 0:
		return life, fmt.Sprintf("org: gate: org_id %q was disbanded and has not started again: the gate checks a running "+
			"feature org; start it with ralph org start --plan, or name a running org_id (ralph org status lists them)", orgID)
	case f == nil:
		return life, fmt.Sprintf("org: gate: org_id %q is not running and holds no split plan feature binding: name the org_id "+
			"of a feature org that ralph org start --plan started (ralph org status lists them)", orgID)
	case !f.Complete():
		return life, fmt.Sprintf("org: gate: org_id %q is bound to %s: disband it and start it again with ralph org start --plan", orgID, f)
	case !running:
		// A binding is part of a reservation, which makes an org count as
		// running (RunningOrgs), so this guards a later change of either.
		return life, fmt.Sprintf("org: gate: org_id %q is bound to %s but is not running: start it again with ralph org start --plan", orgID, f)
	}
	life.binding = f
	return life, ""
}

// gateTarget is what the checks before the scripts found.
type gateTarget struct {
	root   string // the feature worktree's root, symlinks resolved
	branch string // the feature branch, checked out there
	head   string // HEAD's commit
	base   gateBase
}

// gateBase is the base a gate checks HEAD against (resolveGateBase).
type gateBase struct {
	branch string // the default branch's name
	ref    string // refs/remotes/origin/<branch> or refs/heads/<branch>
	sha    string // ref's tip
}

// gateWording is how the worktree checks that ralph org regate shares with
// Gate (gateCheckBranch, gateDirtyRefusal, gateCheckScripts) name the verb
// in a refusal: the two verbs' refusals differ only in these. The shared
// helpers that only fail with an error (gateGit and the ones built on it)
// take the prefix alone, so the error names the verb that ran them. The log
// and the scripts' runs (createGateLog, runGateScripts) name the verb too.
type gateWording struct {
	prefix string // what the refusal starts with: "org: gate: "
	again  string // its last step: "run the gate again"
	verb   string // the command in the log's lines: "ralph org gate"
	// afterCommit follows a commit the refusal asks for, before again: ""
	// for Gate, which gates the worktree's HEAD; ralph org regate gates only
	// what origin has, so it asks for a push too.
	afterCommit string
}

// gateWords is Gate's wording.
var gateWords = gateWording{prefix: "org: gate: ", again: "run the gate again", verb: "ralph org gate"}

// gateCheckWorktree runs Gate's checks of the worktree dir is in against f,
// in Gate's order, and returns what they found, or the refusal of the first
// that fails. The error is a git command that could not run or failed in a
// way no check expects.
func gateCheckWorktree(ctx context.Context, orgID, dir string, f *FeatureBinding) (gateTarget, string, error) {
	r, err := gateGit(ctx, gateWords.prefix, dir, "rev-parse", "--show-toplevel")
	if err != nil {
		return gateTarget{}, "", err
	}
	if r.code != 0 {
		return gateTarget{}, fmt.Sprintf("org: gate: %s is not in a git worktree (git rev-parse --show-toplevel: %s): cd into "+
			"org_id %q's feature worktree %s and run the gate again", dir, r.stderr, orgID, f.Worktree), nil
	}
	root, err := filepath.EvalSymlinks(strings.TrimSuffix(r.stdout, "\n"))
	if err != nil {
		return gateTarget{}, "", fmt.Errorf("org: gate: resolve the worktree root %s: %w", strings.TrimSuffix(r.stdout, "\n"), err)
	}
	want, err := filepath.EvalSymlinks(f.Worktree)
	if err != nil {
		return gateTarget{}, fmt.Sprintf("org: gate: org_id %q's feature worktree %s cannot be read (%s): restore it, or disband "+
			"the org and start it again with ralph org start --plan", orgID, f.Worktree, pathErrorCause(err)), nil
	}
	if root != want {
		return gateTarget{}, fmt.Sprintf("org: gate: the gate ran in the git worktree %s, not in org_id %q's feature worktree %s: "+
			"cd into %s and run the gate again", root, orgID, f.Worktree, f.Worktree), nil
	}
	branch, refused, err := gateCheckBranch(ctx, root, f, gateWords)
	if err != nil || refused != "" {
		return gateTarget{}, refused, err
	}
	t := gateTarget{root: root, branch: branch}

	dirty, err := gateDirtyLines(ctx, gateWords.prefix, root)
	if err != nil {
		return gateTarget{}, "", err
	}
	if len(dirty) > 0 {
		return gateTarget{}, gateDirtyRefusal(root, dirty, gateWords), nil
	}
	if t.head, err = gateHead(ctx, gateWords.prefix, root); err != nil {
		return gateTarget{}, "", err
	}

	base, found, err := resolveGateBase(ctx, gateWords.prefix, root)
	if err != nil {
		return gateTarget{}, "", err
	}
	if !found {
		refused, err = gateNoBaseRefusal(ctx, root)
		return gateTarget{}, refused, err
	}
	t.base = base
	if refused, err = gateCheckAncestor(ctx, t); err != nil || refused != "" {
		return gateTarget{}, refused, err
	}
	if refused = gateCheckScripts(root, gateWords); refused != "" {
		return gateTarget{}, refused, nil
	}
	return t, "", nil
}

// gateCheckBranch returns the branch checked out in root, or the refusal of
// a worktree that is not on f's branch, worded with w: another branch, a
// detached HEAD, or a rebase in progress (HEAD is detached then, and `git
// switch` fails until the rebase is finished or undone,
// gateRebaseInProgress).
func gateCheckBranch(ctx context.Context, root string, f *FeatureBinding, w gateWording) (string, string, error) {
	out, err := gateGitOutput(ctx, w.prefix, root, "branch", "--show-current")
	if err != nil {
		return "", "", err
	}
	switch branch := strings.TrimSpace(out); branch {
	case f.Branch:
		return branch, "", nil
	case "":
	default:
		return "", w.prefix + fmt.Sprintf("%s has the branch %s checked out, not the feature branch %s: run git switch "+
			"%s and %s", root, branch, f.Branch, f.Branch, w.again), nil
	}
	rebasing, err := gateRebaseInProgress(ctx, w.prefix, root)
	if err != nil {
		return "", "", err
	}
	if rebasing {
		return "", w.prefix + fmt.Sprintf("a rebase is in progress in %s, so HEAD is detached and not on the feature branch %s: "+
			"finish it with git rebase --continue, or undo it with git rebase --abort, and %s", root, f.Branch, w.again), nil
	}
	return "", w.prefix + fmt.Sprintf("HEAD is detached in %s, not on the feature branch %s: run git switch %s "+
		"and %s", root, f.Branch, f.Branch, w.again), nil
}

// gateRebaseInProgress reports whether a rebase is stopped in root: git's
// rebase-merge or rebase-apply directory for root's worktree exists
// (gateGitPathExists). An error starts with prefix ("org: gate: ").
func gateRebaseInProgress(ctx context.Context, prefix, root string) (bool, error) {
	return gateGitPathExists(ctx, prefix, root, "a rebase", "rebase-merge", "rebase-apply")
}

// gateGitPathExists reports whether any of names, paths in git's directory
// for root's worktree, exists (`git rev-parse --git-path` prints each,
// relative to root unless it is absolute, as it is in a linked worktree).
// what names the state they mark in the error ("a rebase"), which starts
// with prefix ("org: gate: ").
func gateGitPathExists(ctx context.Context, prefix, root, what string, names ...string) (bool, error) {
	args := []string{"rev-parse"}
	for _, name := range names {
		args = append(args, "--git-path", name)
	}
	out, err := gateGitOutput(ctx, prefix, root, args...)
	if err != nil {
		return false, err
	}
	for path := range strings.SplitSeq(strings.TrimSpace(out), "\n") {
		if !filepath.IsAbs(path) {
			path = filepath.Join(root, path)
		}
		_, err := os.Stat(path)
		switch {
		case err == nil:
			return true, nil
		case !errors.Is(err, fs.ErrNotExist):
			return false, fmt.Errorf("%scheck for %s in progress in %s: %w", prefix, what, root, err)
		}
	}
	return false, nil
}

// gateNoBaseRefusal is the refusal of root's repository when resolveGateBase
// found no default branch: with an origin remote, fetch it; with none, there
// is nothing to fetch, so create the default branch locally.
func gateNoBaseRefusal(ctx context.Context, root string) (string, error) {
	remotes, err := gateGitOutput(ctx, gateWords.prefix, root, "remote")
	if err != nil {
		return "", err
	}
	if slices.Contains(strings.Fields(remotes), "origin") {
		return fmt.Sprintf("org: gate: no default branch found in %s: refs/remotes/origin/HEAD names none that exists, and "+
			"neither main nor master exists as refs/remotes/origin/<name> or refs/heads/<name>: run git fetch origin (and git "+
			"remote set-head origin --auto) and run the gate again", root), nil
	}
	return fmt.Sprintf("org: gate: no default branch found in %s: the repository has no origin remote, and neither main nor "+
		"master exists as refs/heads/<name>: create the default branch locally as main or master (git branch main <commit>) "+
		"and run the gate again", root), nil
}

// gateCheckAncestor refuses t when its base's tip is not an ancestor of
// HEAD (`git merge-base --is-ancestor`, exit 1): the branch needs a rebase.
func gateCheckAncestor(ctx context.Context, t gateTarget) (string, error) {
	r, err := gateGit(ctx, gateWords.prefix, t.root, "merge-base", "--is-ancestor", t.base.ref, "HEAD")
	switch {
	case err != nil:
		return "", err
	case r.code == 0:
		return "", nil
	case r.code != 1:
		return "", fmt.Errorf("org: gate: git merge-base --is-ancestor %s HEAD in %s: exit %d: %s", t.base.ref, t.root, r.code, r.stderr)
	}
	next := fmt.Sprintf("run `git fetch origin %s` and `git rebase %s`", t.base.branch, t.base.ref)
	if !strings.HasPrefix(t.base.ref, "refs/remotes/") {
		next = fmt.Sprintf("run `git rebase %s`", t.base.ref)
	}
	return fmt.Sprintf("org: gate: HEAD %s of %s does not contain %s's tip %s: %s, then run the gate again",
		t.head, t.branch, t.base.ref, t.base.sha, next), nil
}

// gateCheckScripts refuses root, worded with w, when one of the two gate
// scripts is not there to run (gateCheckScript), the static script first.
func gateCheckScripts(root string, w gateWording) string {
	for _, rel := range []string{GateStaticScript, GateTestScript} {
		if refused := gateCheckScript(root, rel, w); refused != "" {
			return refused
		}
	}
	return ""
}

// gateCheckScript refuses rel in root, worded with w, when it does not exist
// or is not an executable regular file (symlinks followed).
func gateCheckScript(root, rel string, w gateWording) string {
	path := filepath.Join(root, filepath.FromSlash(rel))
	info, err := os.Stat(path)
	switch {
	case errors.Is(err, fs.ErrNotExist):
		return w.prefix + fmt.Sprintf("%s does not exist: the gate runs ./%s in the feature worktree, so add it, commit it%s, "+
			"and %s", path, rel, w.afterCommit, w.again)
	case err != nil:
		return w.prefix + fmt.Sprintf("%s cannot be read (%s): fix it, commit it%s, and %s", path, pathErrorCause(err),
			w.afterCommit, w.again)
	case !info.Mode().IsRegular():
		return w.prefix + fmt.Sprintf("%s is not a regular file: the gate runs ./%s, so make it the script, commit it%s, and %s",
			path, rel, w.afterCommit, w.again)
	case info.Mode().Perm()&0o111 == 0:
		return w.prefix + fmt.Sprintf("%s is not executable: run chmod +x %s, commit the change%s, and %s", path, rel,
			w.afterCommit, w.again)
	}
	return ""
}

// gateDirtyRefusal is the refusal of a worktree with the porcelain lines
// dirty, worded with w, the first GateDirtyListLines of them listed.
func gateDirtyRefusal(root string, dirty []string, w gateWording) string {
	shown := dirty[:min(len(dirty), GateDirtyListLines)]
	list := "  " + strings.Join(shown, "\n  ")
	if more := len(dirty) - len(shown); more > 0 {
		list += fmt.Sprintf("\n  ... and %d more", more)
	}
	return w.prefix + fmt.Sprintf("the feature worktree %s has %d uncommitted change(s) or untracked file(s): commit or "+
		"remove them (a file the scripts write belongs in .gitignore)%s and %s:\n%s", root, len(dirty), w.afterCommit, w.again, list)
}

// gateCheckAfter reads t's worktree again after both scripts passed and
// returns the gate_failed reason, or "" when HEAD is the same commit and the
// worktree is still clean. dirty is set for tree_changed. An error starts
// with prefix ("org: gate: ").
func gateCheckAfter(ctx context.Context, prefix string, t gateTarget) (reason string, dirty []string, err error) {
	head, err := gateHead(ctx, prefix, t.root)
	if err != nil {
		return "", nil, err
	}
	if head != t.head {
		return GateReasonHeadMoved, nil, nil
	}
	if dirty, err = gateDirtyLines(ctx, prefix, t.root); err != nil {
		return "", nil, err
	}
	if len(dirty) > 0 {
		return GateReasonTreeChanged, dirty, nil
	}
	return "", nil, nil
}

// resolveGateBase finds the default branch of the repository root is in and
// its base ref (AC4). The candidates, in order, are the branch
// refs/remotes/origin/HEAD points to (`git symbolic-ref --quiet`, read as a
// full ref name so a local branch called origin/<name> cannot shorten it),
// then main, then master; the first for which refs/remotes/origin/<name> or
// refs/heads/<name> exists is the default branch, and its base ref is the
// remote ref when that exists and the local one otherwise. found is false
// when no candidate exists. It reads only local refs, never the network. An
// error starts with prefix ("org: gate: ").
func resolveGateBase(ctx context.Context, prefix, root string) (gateBase, bool, error) {
	candidates := []string{"main", "master"}
	r, err := gateGit(ctx, prefix, root, "symbolic-ref", "--quiet", "refs/remotes/origin/HEAD")
	switch {
	case err != nil:
		return gateBase{}, false, err
	case r.code == 0:
		if name, ok := strings.CutPrefix(strings.TrimSpace(r.stdout), "refs/remotes/origin/"); ok && name != "" && name != "HEAD" {
			candidates = append([]string{name}, candidates...)
		}
	case r.code != 1:
		return gateBase{}, false, fmt.Errorf("%sgit symbolic-ref --quiet refs/remotes/origin/HEAD in %s: exit %d: %s", prefix, root, r.code, r.stderr)
	}
	for _, name := range candidates {
		for _, ref := range []string{"refs/remotes/origin/" + name, "refs/heads/" + name} {
			exists, err := gateRefExists(ctx, prefix, root, ref)
			if err != nil {
				return gateBase{}, false, err
			}
			if !exists {
				continue
			}
			sha, err := gateGitOutput(ctx, prefix, root, "rev-parse", "--verify", "--quiet", ref+"^{commit}")
			if err != nil {
				return gateBase{}, false, err
			}
			return gateBase{branch: name, ref: ref, sha: strings.TrimSpace(sha)}, true, nil
		}
	}
	return gateBase{}, false, nil
}

// gateRefExists reports whether ref exists in root's repository (`git
// show-ref --verify --quiet`, exit 1 for no). An error starts with prefix
// ("org: gate: ").
func gateRefExists(ctx context.Context, prefix, root, ref string) (bool, error) {
	r, err := gateGit(ctx, prefix, root, "show-ref", "--verify", "--quiet", ref)
	switch {
	case err != nil:
		return false, err
	case r.code == 0:
		return true, nil
	case r.code == 1:
		return false, nil
	}
	return false, fmt.Errorf("%sgit show-ref --verify %s in %s: exit %d: %s", prefix, ref, root, r.code, r.stderr)
}

// gateHead returns root's HEAD commit. An error starts with prefix ("org:
// gate: ").
func gateHead(ctx context.Context, prefix, root string) (string, error) {
	out, err := gateGitOutput(ctx, prefix, root, "rev-parse", "--verify", "HEAD^{commit}")
	return strings.TrimSpace(out), err
}

// gateDirtyLines returns `git status --porcelain --untracked-files=all` in
// root, one entry per line: nil for a clean worktree. Ignored files are not
// listed. An error starts with prefix ("org: gate: ").
func gateDirtyLines(ctx context.Context, prefix, root string) ([]string, error) {
	out, err := gateGitOutput(ctx, prefix, root, "status", "--porcelain", "--untracked-files=all")
	if err != nil {
		return nil, err
	}
	var lines []string
	for line := range strings.SplitSeq(out, "\n") {
		if line != "" {
			lines = append(lines, line)
		}
	}
	return lines, nil
}

// gitResult is one git command's output and exit code.
type gitResult struct {
	stdout string
	stderr string // trimmed
	code   int
}

// gateGit runs `git --no-optional-locks -C dir args...` (so a status does not
// take the index lock from a git command the leader runs at the same time)
// and returns its output and exit code. The error is for a git that could not
// run or was stopped by ctx; a non-zero exit is not one. It starts with
// prefix, the verb that ran git: gateWords.prefix ("org: gate: ") for Gate,
// regatePrefix for Regate.
func gateGit(ctx context.Context, prefix, dir string, args ...string) (gitResult, error) {
	cmd := exec.CommandContext(ctx, "git", append([]string{"--no-optional-locks", "-C", dir}, args...)...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	err := cmd.Run()
	r := gitResult{stdout: stdout.String(), stderr: strings.TrimSpace(stderr.String())}
	var exitErr *exec.ExitError
	switch {
	case ctx.Err() != nil:
		return r, fmt.Errorf("%sgit %s in %s: %w", prefix, strings.Join(args, " "), dir, context.Cause(ctx))
	case err == nil:
	case errors.As(err, &exitErr) && exitErr.ExitCode() > 0:
		r.code = exitErr.ExitCode()
	default:
		return r, fmt.Errorf("%sgit %s in %s: %w", prefix, strings.Join(args, " "), dir, err)
	}
	return r, nil
}

// gateGitOutput is gateGit with a non-zero exit as an error naming git's
// stderr, and returns stdout. An error starts with prefix ("org: gate: ").
func gateGitOutput(ctx context.Context, prefix, dir string, args ...string) (string, error) {
	r, err := gateGit(ctx, prefix, dir, args...)
	if err != nil {
		return "", err
	}
	if r.code != 0 {
		return "", fmt.Errorf("%sgit %s in %s: exit %d: %s", prefix, strings.Join(args, " "), dir, r.code, r.stderr)
	}
	return r.stdout, nil
}

// recordGate appends record as orgID's gate event under the manifest lock,
// after it reads the manifest again: when the org's binding or life is no
// longer life's (it was disbanded, started again, or bound to another
// feature while the scripts ran), it records nothing and returns the
// refusal. The error is a lock, read, or append that failed, or ctx's cause
// when ctx is done before the append; nothing is recorded then either.
func (o *Org) recordGate(ctx context.Context, orgID string, life gateLife, record gateRecord) (string, error) {
	// ctx is checked twice (cross-review cycle 1 #1), so an interrupt that
	// lands after the scripts finished records nothing either: here, before
	// the lock is taken, and under the lock right before the append. The wait
	// for a held lock ends when ctx is done (withManifestLockContext), but
	// taking a free lock does not look at ctx, and the interrupt can also land
	// while the manifest is read under the lock.
	if ctx.Err() != nil {
		return "", context.Cause(ctx)
	}
	var refused string
	err := withManifestLockContext(ctx, filepath.Dir(o.Manifest.Path()), func() error {
		rr, err := o.Manifest.Read()
		if err != nil {
			return fmt.Errorf("org: gate: read manifest: %w", err)
		}
		now, notBound := gateOrgLife(rr.Events, orgID)
		var changed string
		switch {
		case notBound != "" && now.lastDisband != life.lastDisband:
			changed = "it was disbanded"
		case notBound != "":
			changed = "it no longer holds a complete feature binding"
		case now.lastDisband != life.lastDisband:
			changed = "it was disbanded and started again"
		case !sameFeature(now.binding, life.binding):
			changed = "it is now bound to " + now.binding.String()
		}
		if changed != "" {
			refused = fmt.Sprintf("org: gate: nothing was recorded, since org_id %q changed while the scripts ran (%s): check the "+
				"org with ralph org status and run the gate again in the feature worktree of its current binding", orgID, changed)
			return nil
		}
		ev := ManifestEvent{TS: o.now(), OrgID: orgID, SeatID: "", Event: record.event(), Details: record.details()}
		if ctx.Err() != nil {
			return context.Cause(ctx)
		}
		if err := o.appendEvent(ev); err != nil {
			return fmt.Errorf("org: gate: record %s: %w", ev.Event, err)
		}
		return nil
	})
	switch {
	case err != nil && ctx.Err() != nil:
		// Interrupted: Gate says so (gateInterrupted).
		return "", err
	case err != nil:
		return "", fmt.Errorf("%w; nothing was recorded, so run the gate again", err)
	}
	return refused, nil
}
