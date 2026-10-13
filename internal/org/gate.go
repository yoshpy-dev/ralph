package org

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync/atomic"
	"syscall"
	"time"
)

// The `ralph org gate` verb (Gate), stage 7b of plan
// docs/plans/active/2026-10-13-org-gate.md (FR-7 of
// docs/specs/2026-10-07-org-multi-org-director.md). A feature org's leader
// rebases its branch on the default branch and runs the gate in the feature
// worktree, on the head it pushes, before it opens the PR. The gate checks
// the org's binding, the worktree, and that HEAD contains the default
// branch's tip, runs scripts/run-static-verify.sh and then
// scripts/run-test.sh there, and records the outcome with the head and base
// SHAs as one org-level gate_passed or gate_failed event (gateRecord). Stage
// 7c's merge order reads that record.
//
// A refusal before the scripts run records nothing and writes no log (B7).
// The gate never fetches, pushes, or calls herdr or agmsg. Neither the watch
// gate (watchgate.go) nor the approval check (approval.go) applies to it,
// since it starts no new autonomous work (AC7).

const (
	// gateStaticScript and gateTestScript are the scripts Gate runs, in this
	// order, relative to the feature worktree's root.
	gateStaticScript = "scripts/run-static-verify.sh"
	gateTestScript   = "scripts/run-test.sh"
	// gateLogDir is the directory under the org state dir that holds the
	// gate logs, <org_id>-<UTC time>.log (createGateLog).
	gateLogDir = "gates"
	// gateLogTimeLayout is the UTC time in a gate log's name.
	gateLogTimeLayout = "20060102T150405Z"
	// gateLogMaxSuffix bounds the -2, -3, ... suffixes createGateLog tries
	// when a log of the same org and second exists.
	gateLogMaxSuffix = 100
	// gateTailLines is how many of the failing script's last output lines
	// GateResult.Tail keeps.
	gateTailLines = 20
	// gateTailMaxLineBytes caps one line of the tail; a line without a line
	// break grows no further.
	gateTailMaxLineBytes = 4096
	// gateRefusedListLines caps the porcelain lines a dirty-tree refusal
	// lists.
	gateRefusedListLines = 20
	// gateScopeEnv is the one verification variable the scripts get (B9):
	// the full language scope, never narrowed by a diff.
	gateScopeEnv = "RALPH_VERIFY_SCOPE=full"
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

// gateStrippedEnvPrefixes are the prefixes of the variable names Gate drops
// from the caller's environment before it runs the scripts (B9): a
// RALPH_VERIFY_BASE=HEAD or a RALPH_VERIFY_PROJECT_ROOTS could otherwise make
// the language verifiers skip the change.
var gateStrippedEnvPrefixes = []string{"RALPH_VERIFY_", "HARNESS_VERIFY_"}

// gateKillGrace is how long an interrupted gate waits for a script's process
// group to exit after SIGTERM before it sends SIGKILL to the group. Tests
// shorten it.
var gateKillGrace = 5 * time.Second

// gitObjectPattern is the shape of a commit SHA the gate records: 40
// lowercase hex digits, or 64 in a SHA-256 repository.
var gitObjectPattern = regexp.MustCompile(`^[0-9a-f]{40}([0-9a-f]{24})?$`)

// The keys of an EventGatePassed's or EventGateFailed's Details, written in
// this order (reason only on gate_failed), each as one space-separated
// `key=value` token. log comes last because its value is strconv.Quote'd and
// may hold spaces, the rule of plan_approved's cwd (planApproval.details).
const (
	gateRecordReasonKey    = "reason"
	gateRecordHeadKey      = "head"
	gateRecordBaseKey      = "base"
	gateRecordBaseRefKey   = "base_ref"
	gateRecordBranchKey    = "branch"
	gateRecordByKey        = "by"
	gateRecordHerdrPaneKey = "herdr_pane"
	gateRecordLogKey       = "log"
)

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
	Log       string // the absolute path of the gate log
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
	return d + gateRecordHeadKey + "=" + r.Head +
		" " + gateRecordBaseKey + "=" + r.Base +
		" " + gateRecordBaseRefKey + "=" + r.BaseRef +
		" " + gateRecordBranchKey + "=" + r.Branch +
		" " + gateRecordByKey + "=" + r.By +
		" " + gateRecordHerdrPaneKey + "=" + pane +
		" " + gateRecordLogKey + "=" + strconv.Quote(r.Log)
}

// gateRecordFromDetails reads a gate event's Details back (stage 7c reads
// the records with it). ok is false when a key before log is given twice or a
// token is not `key=value`, when head or base is not a commit SHA
// (gitObjectPattern), base_ref is not under refs/, branch is empty, or reason
// is set to anything but a GateReason* value. Keys it does not know are
// skipped. by, herdr_pane and log are read on a best-effort basis: they are
// for an audit. A record whose Reason is empty is a gate_passed one
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
		HerdrPane: values[gateRecordHerdrPaneKey] == "set",
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
//     branch --show-current`; a detached HEAD is refused);
//   - the worktree is clean: `git status --porcelain --untracked-files=all`
//     prints nothing (ignored files do not count);
//   - a default branch is found and its base ref's tip is an ancestor of
//     HEAD (resolveGateBase; the gate never fetches);
//   - both scripts exist in the worktree's root as executable regular files.
//
// It then writes a log under <state-dir>/gates/ (createGateLog) and runs
// ./scripts/run-static-verify.sh and, when that exits 0,
// ./scripts/run-test.sh in the worktree's root, with the caller's
// environment fixed by gateScriptEnv, each in its own process group
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
// stopped and nothing is recorded.
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

	logPath, logFile, err := o.createGateLog(p.OrgID)
	if err != nil {
		return GateResult{}, err
	}
	res := GateResult{Head: target.head, Base: target.base.sha, BaseRef: target.base.ref, Branch: target.branch, LogPath: logPath}
	run, err := runGateScripts(ctx, target, gateScriptEnv(env), logFile)
	if closeErr := logFile.Close(); err == nil && closeErr != nil {
		err = fmt.Errorf("org: gate: write the log %s: %w", logPath, closeErr)
	}
	if err != nil {
		return res, gateInterrupted(ctx, err, logPath)
	}
	res.Reason, res.Tail = run.failed, run.tail
	if res.Reason == "" {
		if res.Reason, res.Dirty, err = gateCheckAfter(ctx, target); err != nil {
			return res, gateInterrupted(ctx, err, logPath)
		}
	}

	record := gateRecord{
		Reason: res.Reason, Head: res.Head, Base: res.Base, BaseRef: res.BaseRef, Branch: res.Branch,
		By: approverName(gateEnvValue(env, "USER")), HerdrPane: gateEnvValue(env, herdrPaneIDEnv) != "", Log: logPath,
	}
	if res.Refused, err = o.recordGate(p.OrgID, life, record); err != nil {
		return res, err
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
			"./%s and ./%s yourself", orgID, gateStaticScript, gateTestScript)
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

// gateCheckWorktree runs Gate's checks of the worktree dir is in against f,
// in Gate's order, and returns what they found, or the refusal of the first
// that fails. The error is a git command that could not run or failed in a
// way no check expects.
func gateCheckWorktree(ctx context.Context, orgID, dir string, f *FeatureBinding) (gateTarget, string, error) {
	r, err := gateGit(ctx, dir, "rev-parse", "--show-toplevel")
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
	t := gateTarget{root: root}

	branch, err := gateGitMust(ctx, root, "branch", "--show-current")
	if err != nil {
		return gateTarget{}, "", err
	}
	switch t.branch = strings.TrimSpace(branch); {
	case t.branch == "":
		return gateTarget{}, fmt.Sprintf("org: gate: HEAD is detached in %s, not on the feature branch %s: run git switch %s "+
			"and run the gate again", root, f.Branch, f.Branch), nil
	case t.branch != f.Branch:
		return gateTarget{}, fmt.Sprintf("org: gate: %s has the branch %s checked out, not the feature branch %s: run git switch "+
			"%s and run the gate again", root, t.branch, f.Branch, f.Branch), nil
	}

	dirty, err := gateDirtyLines(ctx, root)
	if err != nil {
		return gateTarget{}, "", err
	}
	if len(dirty) > 0 {
		return gateTarget{}, gateDirtyRefusal(root, dirty), nil
	}
	if t.head, err = gateHead(ctx, root); err != nil {
		return gateTarget{}, "", err
	}

	base, found, err := resolveGateBase(ctx, root)
	if err != nil {
		return gateTarget{}, "", err
	}
	if !found {
		return gateTarget{}, fmt.Sprintf("org: gate: no default branch found in %s: refs/remotes/origin/HEAD names none that "+
			"exists, and neither main nor master exists as refs/remotes/origin/<name> or refs/heads/<name>: run git fetch "+
			"origin (and git remote set-head origin --auto) and run the gate again", root), nil
	}
	t.base = base
	if refused, err := gateCheckAncestor(ctx, t); err != nil || refused != "" {
		return gateTarget{}, refused, err
	}
	for _, rel := range []string{gateStaticScript, gateTestScript} {
		if refused := gateCheckScript(root, rel); refused != "" {
			return gateTarget{}, refused, nil
		}
	}
	return t, "", nil
}

// gateCheckAncestor refuses t when its base's tip is not an ancestor of
// HEAD (`git merge-base --is-ancestor`, exit 1): the branch needs a rebase.
func gateCheckAncestor(ctx context.Context, t gateTarget) (string, error) {
	r, err := gateGit(ctx, t.root, "merge-base", "--is-ancestor", t.base.ref, "HEAD")
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

// gateCheckScript refuses rel in root when it does not exist or is not an
// executable regular file (symlinks followed).
func gateCheckScript(root, rel string) string {
	path := filepath.Join(root, filepath.FromSlash(rel))
	info, err := os.Stat(path)
	switch {
	case errors.Is(err, fs.ErrNotExist):
		return fmt.Sprintf("org: gate: %s does not exist: the gate runs ./%s in the feature worktree, so add it, commit it, "+
			"and run the gate again", path, rel)
	case err != nil:
		return fmt.Sprintf("org: gate: %s cannot be read (%s): fix it, commit it, and run the gate again", path, pathErrorCause(err))
	case !info.Mode().IsRegular():
		return fmt.Sprintf("org: gate: %s is not a regular file: the gate runs ./%s, so make it the script, commit it, and run "+
			"the gate again", path, rel)
	case info.Mode().Perm()&0o111 == 0:
		return fmt.Sprintf("org: gate: %s is not executable: run chmod +x %s, commit the change, and run the gate again", path, rel)
	}
	return ""
}

// gateDirtyRefusal is the refusal of a worktree with the porcelain lines
// dirty, the first gateRefusedListLines of them listed.
func gateDirtyRefusal(root string, dirty []string) string {
	shown := dirty[:min(len(dirty), gateRefusedListLines)]
	list := "  " + strings.Join(shown, "\n  ")
	if more := len(dirty) - len(shown); more > 0 {
		list += fmt.Sprintf("\n  ... and %d more", more)
	}
	return fmt.Sprintf("org: gate: the feature worktree %s has %d uncommitted change(s) or untracked file(s): commit or "+
		"remove them (a file the scripts write belongs in .gitignore) and run the gate again:\n%s", root, len(dirty), list)
}

// gateCheckAfter reads t's worktree again after both scripts passed and
// returns the gate_failed reason, or "" when HEAD is the same commit and the
// worktree is still clean. dirty is set for tree_changed.
func gateCheckAfter(ctx context.Context, t gateTarget) (reason string, dirty []string, err error) {
	head, err := gateHead(ctx, t.root)
	if err != nil {
		return "", nil, err
	}
	if head != t.head {
		return GateReasonHeadMoved, nil, nil
	}
	if dirty, err = gateDirtyLines(ctx, t.root); err != nil {
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
// when no candidate exists. It reads only local refs, never the network.
func resolveGateBase(ctx context.Context, root string) (gateBase, bool, error) {
	candidates := []string{"main", "master"}
	r, err := gateGit(ctx, root, "symbolic-ref", "--quiet", "refs/remotes/origin/HEAD")
	switch {
	case err != nil:
		return gateBase{}, false, err
	case r.code == 0:
		if name, ok := strings.CutPrefix(strings.TrimSpace(r.stdout), "refs/remotes/origin/"); ok && name != "" && name != "HEAD" {
			candidates = append([]string{name}, candidates...)
		}
	case r.code != 1:
		return gateBase{}, false, fmt.Errorf("org: gate: git symbolic-ref --quiet refs/remotes/origin/HEAD in %s: exit %d: %s", root, r.code, r.stderr)
	}
	for _, name := range candidates {
		for _, ref := range []string{"refs/remotes/origin/" + name, "refs/heads/" + name} {
			exists, err := gateRefExists(ctx, root, ref)
			if err != nil {
				return gateBase{}, false, err
			}
			if !exists {
				continue
			}
			sha, err := gateGitMust(ctx, root, "rev-parse", "--verify", "--quiet", ref+"^{commit}")
			if err != nil {
				return gateBase{}, false, err
			}
			return gateBase{branch: name, ref: ref, sha: strings.TrimSpace(sha)}, true, nil
		}
	}
	return gateBase{}, false, nil
}

// gateRefExists reports whether ref exists in root's repository (`git
// show-ref --verify --quiet`, exit 1 for no).
func gateRefExists(ctx context.Context, root, ref string) (bool, error) {
	r, err := gateGit(ctx, root, "show-ref", "--verify", "--quiet", ref)
	switch {
	case err != nil:
		return false, err
	case r.code == 0:
		return true, nil
	case r.code == 1:
		return false, nil
	}
	return false, fmt.Errorf("org: gate: git show-ref --verify %s in %s: exit %d: %s", ref, root, r.code, r.stderr)
}

// gateHead returns root's HEAD commit.
func gateHead(ctx context.Context, root string) (string, error) {
	out, err := gateGitMust(ctx, root, "rev-parse", "--verify", "HEAD^{commit}")
	return strings.TrimSpace(out), err
}

// gateDirtyLines returns `git status --porcelain --untracked-files=all` in
// root, one entry per line: nil for a clean worktree. Ignored files are not
// listed.
func gateDirtyLines(ctx context.Context, root string) ([]string, error) {
	out, err := gateGitMust(ctx, root, "status", "--porcelain", "--untracked-files=all")
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
// run or was stopped by ctx; a non-zero exit is not one.
func gateGit(ctx context.Context, dir string, args ...string) (gitResult, error) {
	cmd := exec.CommandContext(ctx, "git", append([]string{"--no-optional-locks", "-C", dir}, args...)...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	err := cmd.Run()
	r := gitResult{stdout: stdout.String(), stderr: strings.TrimSpace(stderr.String())}
	var exitErr *exec.ExitError
	switch {
	case ctx.Err() != nil:
		return r, fmt.Errorf("org: gate: git %s in %s: %w", strings.Join(args, " "), dir, context.Cause(ctx))
	case err == nil:
	case errors.As(err, &exitErr) && exitErr.ExitCode() > 0:
		r.code = exitErr.ExitCode()
	default:
		return r, fmt.Errorf("org: gate: git %s in %s: %w", strings.Join(args, " "), dir, err)
	}
	return r, nil
}

// gateGitMust is gateGit with a non-zero exit as an error naming git's
// stderr, and returns stdout.
func gateGitMust(ctx context.Context, dir string, args ...string) (string, error) {
	r, err := gateGit(ctx, dir, args...)
	if err != nil {
		return "", err
	}
	if r.code != 0 {
		return "", fmt.Errorf("org: gate: git %s in %s: exit %d: %s", strings.Join(args, " "), dir, r.code, r.stderr)
	}
	return r.stdout, nil
}

// gateScriptEnv is the environment the gate scripts get (B9): env without
// every variable whose name starts with RALPH_VERIFY_ or HARNESS_VERIFY_,
// then RALPH_VERIFY_SCOPE=full.
func gateScriptEnv(env []string) []string {
	out := make([]string, 0, len(env)+1)
	for _, kv := range env {
		name, _, _ := strings.Cut(kv, "=")
		stripped := false
		for _, prefix := range gateStrippedEnvPrefixes {
			stripped = stripped || strings.HasPrefix(name, prefix)
		}
		if !stripped {
			out = append(out, kv)
		}
	}
	return append(out, gateScopeEnv)
}

// gateEnvValue returns key's value in env, the last entry winning as it does
// for a child process, and "" when env has none.
func gateEnvValue(env []string, key string) string {
	var value string
	for _, kv := range env {
		if name, v, ok := strings.Cut(kv, "="); ok && name == key {
			value = v
		}
	}
	return value
}

// createGateLog creates the log of a gate of orgID, empty, under
// <state-dir>/gates/ (state-dir is the manifest's directory, made absolute):
// <org_id>-<UTC time, gateLogTimeLayout>.log, or the same name with -2, -3,
// ... before .log when one exists, created with O_EXCL so two gates never
// share a log. It returns the log's absolute path and the open file.
func (o *Org) createGateLog(orgID string) (string, *os.File, error) {
	stateDir, err := absPath(filepath.Dir(o.Manifest.Path()))
	if err != nil {
		return "", nil, fmt.Errorf("org: gate: resolve the state dir for the gate log: %w", err)
	}
	dir := filepath.Join(stateDir, gateLogDir)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", nil, fmt.Errorf("org: gate: create the gate log dir %s: %w", dir, err)
	}
	name := orgID + "-" + o.nowTime().UTC().Format(gateLogTimeLayout)
	for n := 1; n <= gateLogMaxSuffix; n++ {
		path := filepath.Join(dir, name+".log")
		if n > 1 {
			path = filepath.Join(dir, name+"-"+strconv.Itoa(n)+".log")
		}
		f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
		if err == nil {
			return path, f, nil
		}
		if !errors.Is(err, fs.ErrExist) {
			return "", nil, fmt.Errorf("org: gate: create the gate log %s: %w", path, err)
		}
	}
	return "", nil, fmt.Errorf("org: gate: create the gate log: %d logs named %s already exist in %s", gateLogMaxSuffix, name, dir)
}

// gateRun is what runGateScripts found.
type gateRun struct {
	failed string   // GateReasonStaticFailed or GateReasonTestFailed; "" when both passed
	tail   []string // the failing script's last output lines
}

// runGateScripts runs the static script and, when it exits 0, the test
// script in t.root (runGateScript), with env, writing a header line before
// each and its exit status after it to log, and their output in between.
// The error is ctx's when it is done, or a log write that failed.
func runGateScripts(ctx context.Context, t gateTarget, env []string, log io.Writer) (gateRun, error) {
	header := func(format string, args ...any) error {
		if _, err := fmt.Fprintf(log, format, args...); err != nil {
			return fmt.Errorf("org: gate: write the log: %w", err)
		}
		return nil
	}
	if err := header("# ralph org gate\n# worktree: %s\n# branch: %s\n# head: %s\n# base: %s %s\n",
		t.root, t.branch, t.head, t.base.ref, t.base.sha); err != nil {
		return gateRun{}, err
	}
	for _, s := range []struct{ rel, reason string }{
		{gateStaticScript, GateReasonStaticFailed},
		{gateTestScript, GateReasonTestFailed},
	} {
		if err := header("\n==> ralph org gate: running ./%s\n", s.rel); err != nil {
			return gateRun{}, err
		}
		out := &gateOutput{log: log}
		code, err := runGateScript(ctx, t.root, s.rel, env, out)
		if err != nil {
			_ = header("\n==> ralph org gate: ./%s interrupted\n", s.rel)
			return gateRun{}, err
		}
		if out.err != nil {
			return gateRun{}, fmt.Errorf("org: gate: write the log: %w", out.err)
		}
		if err := header("\n==> ralph org gate: ./%s exited %d\n", s.rel, code); err != nil {
			return gateRun{}, err
		}
		if code != 0 {
			return gateRun{failed: s.reason, tail: out.tail()}, nil
		}
	}
	return gateRun{}, nil
}

// runGateScript runs ./rel in root with env and both output streams going
// to out, its stdin empty, in a process group of its own, and returns its
// exit code (-1 when a signal ended it or it could not start; the reason then
// goes to out). When ctx is done the group gets SIGTERM, and whatever of it
// is still there gateKillGrace later gets SIGKILL (stopGateGroup); the error
// is then ctx's. Release builds target darwin and linux only, where
// Setpgid and kill(-pgid) exist.
func runGateScript(ctx context.Context, root, rel string, env []string, out io.Writer) (int, error) {
	cmd := exec.CommandContext(ctx, filepath.Join(root, filepath.FromSlash(rel)))
	cmd.Args[0] = "./" + rel
	cmd.Dir, cmd.Env, cmd.Stdout, cmd.Stderr = root, env, out, out
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	// cancelledAt is when Cancel sent SIGTERM (UnixNano), set from the
	// goroutine os/exec watches ctx in.
	var cancelledAt atomic.Int64
	cmd.Cancel = func() error {
		cancelledAt.Store(time.Now().UnixNano())
		_ = syscall.Kill(-cmd.Process.Pid, syscall.SIGTERM)
		return nil
	}
	cmd.WaitDelay = gateKillGrace
	err := cmd.Run()
	if ctx.Err() != nil {
		if cmd.Process != nil {
			since := time.Now()
			if at := cancelledAt.Load(); at != 0 {
				since = time.Unix(0, at)
			}
			stopGateGroup(cmd.Process.Pid, since.Add(gateKillGrace))
		}
		return 0, context.Cause(ctx)
	}
	if cmd.ProcessState == nil {
		_, _ = fmt.Fprintf(out, "ralph org gate: ./%s could not run: %v\n", rel, err)
		return -1, nil
	}
	if errors.Is(err, exec.ErrWaitDelay) {
		_, _ = fmt.Fprintf(out, "ralph org gate: a process ./%s started still held its output %s after it exited\n", rel, gateKillGrace)
	}
	return cmd.ProcessState.ExitCode(), nil
}

// stopGateGroup waits until deadline for process group pgid to have no
// process left (kill(-pgid, 0) fails), then sends SIGKILL to the group. A
// group whose processes are gone answers ESRCH and is left alone.
func stopGateGroup(pgid int, deadline time.Time) {
	for time.Now().Before(deadline) {
		if syscall.Kill(-pgid, 0) != nil {
			return
		}
		time.Sleep(manifestLockPollInterval)
	}
	_ = syscall.Kill(-pgid, syscall.SIGKILL)
}

// gateOutput is the writer a gate script's stdout and stderr both go to
// (os/exec calls Write from one goroutine at a time for the two when they
// are the same writer): every byte goes to log, and the last gateTailLines
// lines are kept for the tail. The first log write error is kept in err and
// later writes skip the log, so the script is not stopped by it.
type gateOutput struct {
	log     io.Writer
	err     error
	lines   []string
	partial []byte
}

func (w *gateOutput) Write(b []byte) (int, error) {
	if w.err == nil {
		if _, err := w.log.Write(b); err != nil {
			w.err = err
		}
	}
	rest := b
	for len(rest) > 0 {
		i := bytes.IndexByte(rest, '\n')
		if i < 0 {
			w.partial = append(w.partial, rest[:min(len(rest), gateTailMaxLineBytes-len(w.partial))]...)
			break
		}
		w.partial = append(w.partial, rest[:min(i, gateTailMaxLineBytes-len(w.partial))]...)
		w.push(string(w.partial))
		w.partial = w.partial[:0]
		rest = rest[i+1:]
	}
	return len(b), nil
}

// push keeps line as the latest complete line, a trailing carriage return
// dropped, and the oldest beyond gateTailLines dropped.
func (w *gateOutput) push(line string) {
	w.lines = append(w.lines, strings.TrimSuffix(line, "\r"))
	if len(w.lines) > gateTailLines {
		w.lines = w.lines[len(w.lines)-gateTailLines:]
	}
}

// tail returns the last gateTailLines lines written, a last line with no
// line break included.
func (w *gateOutput) tail() []string {
	lines := append([]string(nil), w.lines...)
	if len(w.partial) > 0 {
		lines = append(lines, strings.TrimSuffix(string(w.partial), "\r"))
	}
	if len(lines) > gateTailLines {
		lines = lines[len(lines)-gateTailLines:]
	}
	return lines
}

// recordGate appends record as orgID's gate event under the manifest lock,
// after it reads the manifest again: when the org's binding or life is no
// longer life's (it was disbanded, started again, or bound to another
// feature while the scripts ran), it records nothing and returns the
// refusal. The error is a lock, read, or append that failed; nothing is
// recorded then either.
func (o *Org) recordGate(orgID string, life gateLife, record gateRecord) (string, error) {
	var refused string
	err := withManifestLock(filepath.Dir(o.Manifest.Path()), func() error {
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
		if err := o.appendEvent(ev); err != nil {
			return fmt.Errorf("org: gate: record %s: %w", ev.Event, err)
		}
		return nil
	})
	if err != nil {
		return "", fmt.Errorf("%w; nothing was recorded, so run the gate again", err)
	}
	return refused, nil
}
