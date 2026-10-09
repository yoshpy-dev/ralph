package org

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// `ralph org start --plan <split plan> --feature <slug>` (plan
// docs/plans/active/2026-10-09-org-feature-worktree.md, spec FR-4) starts one
// org for one feature of an approved split plan (split.go): a worktree and a
// branch for the feature, made by scripts/ralph-worktree.sh, and a leader
// seat whose cwd is that worktree, bound to the feature (FeatureBinding,
// reserve.go). StartFeature runs it in this order:
//
//  1. the split plan and the feature, with no side effect;
//  2. the leader spawn's own checks on an unlocked read of the manifest
//     (checkSpawnInput and spawnPrecheckErr, spawn.go): max_orgs,
//     max_total_seats, the reservation and its binding, and the rest of what
//     Spawn refuses;
//  3. the ralph-worktree.sh record of the worktree: an existing one is reused
//     only when it was made for the same split plan feature, at the same path
//     on the same branch, and that branch is checked out there;
//  4. `ralph-worktree.sh ensure` in the main worktree, which makes the
//     worktree and the branch from the clean default branch, or returns the
//     recorded worktree;
//  5. Spawn of the leader, which decides 2 again under the manifest lock.
//
// Nothing is written to the manifest before 5, and a start refused before 4
// makes no worktree. A start refused in 4 has the script's message; one that
// fails in 5 says that the worktree and the branch stay.

const (
	// featureWorktreeDir holds every feature worktree, relative to the repo
	// root (`.claude/worktrees/` is gitignored, so the main worktree stays
	// clean for the next start).
	featureWorktreeDir = ".claude/worktrees"
	// featureWorktreeKind and featureWorktreeCleanupPolicy are the --kind
	// and --cleanup-policy StartFeature passes to ralph-worktree.sh ensure.
	// The worktree outlives disband and the PR, so nothing removes it but
	// `ralph-worktree.sh cleanup`.
	featureWorktreeKind          = "org"
	featureWorktreeCleanupPolicy = "manual"
	// worktreeScriptRel is ralph-worktree.sh relative to the repo root.
	worktreeScriptRel = "scripts/ralph-worktree.sh"
)

// StartFeatureParams describes one `ralph org start --plan` invocation.
type StartFeatureParams struct {
	StateDir  string // resolved org state dir (the manifest's dir); split plans are in SplitPlansDirIn(StateDir)
	RepoRoot  string // root of the main worktree, where scripts/ralph-worktree.sh runs
	SplitPath string // --plan as given
	Feature   string // --feature, the feature's slug
	OrgID     string // --org-id; "" means the feature's slug
	// Driver, Model, LeaderDriver and TimeoutMS go to the leader spawn as
	// the SpawnParams fields of the same names.
	Driver, Model, LeaderDriver string
	TimeoutMS                   int
}

// StartFeatureResult is StartFeature's return value. Spawn is the leader
// spawn's result; a start refused before the spawn has only Outcome and Err
// set there (SpawnOutcomeRejected, or SpawnOutcomeFailed when the manifest
// cannot be read or ensure printed another worktree path).
type StartFeatureResult struct {
	Spawn    SpawnResult
	OrgID    string     // the org_id; "" when the start was refused before a valid one was known
	Worktree string     // absolute path of the feature worktree; "" when none was made or reused
	Branch   string     // the feature branch, set together with Worktree
	Split    *SplitPlan // the split plan read; nil when it could not be read
}

// FeatureWorktrees is what StartFeature needs from scripts/ralph-worktree.sh
// and git. Org.Worktrees holds one; nil means scriptFeatureWorktrees.
type FeatureWorktrees interface {
	// Lookup returns the ralph-worktree.sh state record for id, read in
	// repoRoot; ok is false when there is none. A record that exists but
	// cannot be read is an error.
	Lookup(repoRoot, id string) (rec WorktreeRecord, ok bool, err error)
	// Ensure runs `ralph-worktree.sh ensure` for req in repoRoot and returns
	// the worktree path it printed.
	Ensure(repoRoot string, req EnsureWorktree) (string, error)
	// CurrentBranch returns the branch the worktree at path has checked out
	// (`git -C <path> branch --show-current`), "" for a detached HEAD.
	CurrentBranch(path string) (string, error)
}

// WorktreeRecord is the part of a ralph-worktree.sh state record
// (`<git common dir>/ralph/worktrees/<id>.json`) that StartFeature compares.
type WorktreeRecord struct {
	WorktreePath string `json:"worktree_path"`
	Branch       string `json:"branch"`
	Kind         string `json:"kind"`
	CanonicalRef string `json:"canonical_ref"`
}

// EnsureWorktree holds the options of one `ralph-worktree.sh ensure`, each
// passed as the option of the same name. Path is relative to the repo root.
type EnsureWorktree struct {
	ID, Kind, Branch, Path, CanonicalRef, CleanupPolicy string
}

// featureWorktreeID is the ralph-worktree.sh state id of orgID's feature
// worktree.
func featureWorktreeID(orgID string) string {
	return "org-" + orgID
}

// featureWorktreeRelPath is orgID's feature worktree, relative to the repo
// root.
func featureWorktreeRelPath(orgID string) string {
	return featureWorktreeDir + "/" + featureWorktreeID(orgID)
}

// featureCanonicalRef is the canonical_ref recorded for the worktree of
// feature slug of split plan splitID.
func featureCanonicalRef(splitID, slug string) string {
	return "split:" + splitID + "#" + slug
}

// featureWorktreeCleanup is the command that removes orgID's feature
// worktree, its branch and its record.
func featureWorktreeCleanup(orgID string) string {
	return "./" + worktreeScriptRel + " cleanup --id " + featureWorktreeID(orgID)
}

// StartFeature starts the org for feature p.Feature of the split plan at
// p.SplitPath (see the comment at the top of this file for the order). The
// org_id is p.OrgID or the slug, the branch `<type>/<slug>`, and the
// worktree `.claude/worktrees/org-<org_id>` under the main worktree (its
// symlinks resolved, as git prints it). The leader is spawned with the
// worktree as its cwd, the feature's `- Reserve:` paths, the binding (split
// plan id, slug, approval digest, branch, worktree), a one-line scope, and a
// task made of what the org was started for and the feature's body
// (featureLeaderTask). Running the same start again reuses the worktree and,
// while the leader is active, records nothing.
func (o *Org) StartFeature(p StartFeatureParams) StartFeatureResult {
	var res StartFeatureResult
	refuse := func(err error) StartFeatureResult {
		res.Spawn = SpawnResult{Outcome: SpawnOutcomeRejected, Err: err}
		return res
	}

	plan, feature, orgID, err := readStartFeature(p)
	res.Split = plan
	if err != nil {
		return refuse(err)
	}
	res.OrgID = orgID
	root := resolvedOrClean(mustAbs(p.RepoRoot))
	worktree := filepath.Join(root, filepath.FromSlash(featureWorktreeRelPath(orgID)))
	branch := feature.Type + "/" + feature.Slug
	leader := startFeatureLeaderParams(p, plan, feature, orgID, worktree, branch)

	checked, err := checkSpawnInput(leader)
	if err != nil {
		return refuse(err)
	}
	rr, err := o.Manifest.Read()
	if err != nil {
		res.Spawn = SpawnResult{Outcome: SpawnOutcomeFailed, Err: fmt.Errorf("org: read manifest: %w", err)}
		return res
	}
	if err := spawnPrecheckErr(o.Config, checked, rr.Events); err != nil {
		return refuse(err)
	}

	worktrees := o.worktrees()
	want := WorktreeRecord{
		WorktreePath: worktree, Branch: branch, Kind: featureWorktreeKind,
		CanonicalRef: featureCanonicalRef(plan.ID, feature.Slug),
	}
	if err := checkFeatureWorktreeReuse(worktrees, root, orgID, want); err != nil {
		return refuse(err)
	}
	made, err := worktrees.Ensure(root, EnsureWorktree{
		ID: featureWorktreeID(orgID), Kind: want.Kind, Branch: branch, Path: featureWorktreeRelPath(orgID),
		CanonicalRef: want.CanonicalRef, CleanupPolicy: featureWorktreeCleanupPolicy,
	})
	if err != nil {
		return refuse(fmt.Errorf("%w; make the main worktree %s a clean checkout of the default branch and run start again", err, root))
	}
	if !samePath(made, worktree) {
		res.Spawn = SpawnResult{Outcome: SpawnOutcomeFailed, Err: fmt.Errorf(
			"org: %s ensure --id %s printed the worktree %q, not %s; the leader is not started",
			worktreeScriptRel, featureWorktreeID(orgID), made, worktree)}
		return res
	}
	res.Worktree, res.Branch = worktree, branch

	r := o.Spawn(leader)
	if r.Outcome != SpawnOutcomeSpawned && r.Outcome != SpawnOutcomeIdempotent && r.Err != nil {
		r.Err = fmt.Errorf("%w; the worktree %s and its branch %s stay: running the same start again reuses them, and %s removes them if they are not needed",
			r.Err, worktree, branch, featureWorktreeCleanup(orgID))
	}
	res.Spawn = r
	return res
}

// readStartFeature is StartFeature's first step, which has no side effect:
// the repo root is given, the split plan at p.SplitPath is in the state
// dir's splits/, reads and is approved, it has the feature p.Feature, and
// the org_id (p.OrgID, or the slug) is valid. plan is returned whenever it
// was read, also with an error.
func readStartFeature(p StartFeatureParams) (plan *SplitPlan, feature SplitFeature, orgID string, err error) {
	if p.RepoRoot == "" {
		return nil, feature, "", errors.New("org: start --plan needs the root of the main worktree, where " + worktreeScriptRel + " runs")
	}
	path, _, err := ResolveSplitPlanPath(p.StateDir, p.SplitPath)
	if err != nil {
		return nil, feature, "", err
	}
	if plan, err = LoadSplitPlan(path); err != nil {
		return nil, feature, "", err
	}
	if err := plan.CheckApproved(); err != nil {
		return plan, feature, "", err
	}
	feature, ok := plan.Feature(p.Feature)
	if !ok {
		slugs := make([]string, len(plan.Features))
		for i, f := range plan.Features {
			slugs[i] = f.Slug
		}
		return plan, feature, "", fmt.Errorf("org: split plan %s has no feature %q: its features are %s", plan.Path, p.Feature, strings.Join(slugs, ", "))
	}
	orgID = p.OrgID
	if orgID == "" {
		orgID = feature.Slug
	}
	if err := ValidateIdentifier("org_id", orgID); err != nil {
		return plan, feature, "", err
	}
	return plan, feature, orgID, nil
}

// startFeatureLeaderParams is the leader spawn StartFeature asks for feature
// f of plan in orgID (see StartFeature).
func startFeatureLeaderParams(p StartFeatureParams, plan *SplitPlan, f SplitFeature, orgID, worktree, branch string) SpawnParams {
	return SpawnParams{
		OrgID: orgID, SeatID: LeaderIdentity, Role: LeaderIdentity,
		Driver: p.Driver, Model: p.Model, LeaderDriver: p.LeaderDriver, TimeoutMS: p.TimeoutMS,
		Cwd:     worktree,
		Scope:   fmt.Sprintf("split %s feature %s (reserve: %s)", plan.ID, f.Slug, strings.Join(f.Reserve, ", ")),
		Reserve: f.Reserve,
		Feature: &FeatureBinding{Split: plan.ID, Feature: f.Slug, Digest: plan.Digest, Branch: branch, Worktree: worktree},
		Task:    featureLeaderTask(plan, f, worktree, branch),
	}
}

// featureLeaderTask is the leader's {{TASK}} (prompts/leader.md) for feature
// f of plan: one line each for the split plan, the feature, the worktree,
// the branch, the reserved paths, the features it depends on, and the
// procedure to follow, then a blank line and f's body as the plan has it.
func featureLeaderTask(plan *SplitPlan, f SplitFeature, worktree, branch string) string {
	deps := "なし"
	if len(f.DependsOn) > 0 {
		deps = strings.Join(f.DependsOn, ", ")
	}
	task := strings.Join([]string{
		fmt.Sprintf("- 分割計画: %s(%s、承認の digest %s)", plan.ID, plan.Path, plan.Digest),
		"- 機能: " + f.Slug,
		fmt.Sprintf("- worktree: %s(leader の cwd)", worktree),
		"- ブランチ: " + branch,
		"- 予約したパス: " + strings.Join(f.Reserve, ", "),
		"- 依存する機能: " + deps,
		"- 進め方: `/org` skill の「機能ごとの org」の手順に従う",
	}, "\n")
	if f.Body != "" {
		task += "\n\n" + f.Body
	}
	return task
}

// checkFeatureWorktreeReuse looks up the ralph-worktree.sh record of orgID's
// feature worktree and returns nil when there is none or it is want: the same
// canonical_ref (split plan id and slug), worktree path (compared with
// symlinks resolved), branch and kind, with want's branch checked out in that
// directory. Anything else, and a record that cannot be read, is refused
// (plan: Codex plan advisory finding 1): the worktree outlives disband, so
// another split plan with the same slug must not take over its commits, and
// a worktree whose checkout moved to another branch is not the feature's.
// The same split plan feature approved again is still the same record.
func checkFeatureWorktreeReuse(worktrees FeatureWorktrees, root, orgID string, want WorktreeRecord) error {
	id := featureWorktreeID(orgID)
	refuse := func(what string) error {
		return fmt.Errorf("org: the worktree record %s %s, so start does not reuse it: "+
			"remove the worktree with %s if it is no longer needed, or use another --org-id",
			id, what, featureWorktreeCleanup(orgID))
	}
	rec, ok, err := worktrees.Lookup(root, id)
	switch {
	case errors.Is(err, errWorktreeScriptMissing):
		return err
	case err != nil:
		return refuse(fmt.Sprintf("cannot be read (%v)", err))
	case !ok:
		return nil
	}
	var diffs []string
	if rec.CanonicalRef != want.CanonicalRef {
		diffs = append(diffs, fmt.Sprintf("canonical_ref %q, not %q", rec.CanonicalRef, want.CanonicalRef))
	}
	if !samePath(rec.WorktreePath, want.WorktreePath) {
		diffs = append(diffs, fmt.Sprintf("worktree_path %q, not %q", rec.WorktreePath, want.WorktreePath))
	}
	if rec.Branch != want.Branch {
		diffs = append(diffs, fmt.Sprintf("branch %q, not %q", rec.Branch, want.Branch))
	}
	if rec.Kind != want.Kind {
		diffs = append(diffs, fmt.Sprintf("kind %q, not %q", rec.Kind, want.Kind))
	}
	if len(diffs) > 0 {
		return refuse("records " + strings.Join(diffs, ", "))
	}
	if info, err := os.Stat(want.WorktreePath); err != nil || !info.IsDir() {
		return refuse(fmt.Sprintf("names %s, which is not an existing directory", want.WorktreePath))
	}
	current, err := worktrees.CurrentBranch(want.WorktreePath)
	if err != nil {
		return refuse(fmt.Sprintf("names %s, whose checked-out branch cannot be read (%v)", want.WorktreePath, err))
	}
	if current != want.Branch {
		checkedOut := fmt.Sprintf("branch %q", current)
		if current == "" {
			checkedOut = "a detached HEAD"
		}
		return refuse(fmt.Sprintf("names %s, which has %s checked out, not %q", want.WorktreePath, checkedOut, want.Branch))
	}
	return nil
}

// worktrees returns o.Worktrees, or scriptFeatureWorktrees when it is nil.
func (o *Org) worktrees() FeatureWorktrees {
	if o.Worktrees != nil {
		return o.Worktrees
	}
	return scriptFeatureWorktrees{}
}

// errWorktreeScriptMissing is wrapped by scriptFeatureWorktrees' errors when
// the repo has no scripts/ralph-worktree.sh.
var errWorktreeScriptMissing = errors.New("ralph init ships it into the repo; restore it and run start again")

// scriptFeatureWorktrees is the FeatureWorktrees StartFeature uses when
// Org.Worktrees is nil: the repo's scripts/ralph-worktree.sh, run with bash
// in the repo root (ensure checks that the checkout there is the clean
// default branch), and git.
type scriptFeatureWorktrees struct{}

// Lookup runs `ralph-worktree.sh state-path <id>` and reads the JSON record
// at the path it prints, if there is one.
func (scriptFeatureWorktrees) Lookup(repoRoot, id string) (WorktreeRecord, bool, error) {
	out, err := runWorktreeScript(repoRoot, "state-path", id)
	if err != nil {
		return WorktreeRecord{}, false, err
	}
	file := strings.TrimSpace(out)
	if file == "" {
		return WorktreeRecord{}, false, fmt.Errorf("org: %s state-path %s printed no path", worktreeScriptRel, id)
	}
	data, err := os.ReadFile(file)
	switch {
	case errors.Is(err, fs.ErrNotExist):
		return WorktreeRecord{}, false, nil
	case err != nil:
		return WorktreeRecord{}, false, fmt.Errorf("org: read %s: %w", file, pathErrorCause(err))
	}
	var rec WorktreeRecord
	if err := json.Unmarshal(data, &rec); err != nil {
		return WorktreeRecord{}, false, fmt.Errorf("org: %s is not a JSON record: %w", file, err)
	}
	return rec, true, nil
}

// Ensure runs `ralph-worktree.sh ensure` with req's options and returns the
// last line it prints on stdout, the worktree path (git's own output goes to
// stderr).
func (scriptFeatureWorktrees) Ensure(repoRoot string, req EnsureWorktree) (string, error) {
	out, err := runWorktreeScript(repoRoot, "ensure", "--id", req.ID, "--kind", req.Kind, "--branch", req.Branch,
		"--path", req.Path, "--canonical-ref", req.CanonicalRef, "--cleanup-policy", req.CleanupPolicy)
	if err != nil {
		return "", err
	}
	lines := strings.Split(strings.TrimSpace(out), "\n")
	path := strings.TrimSpace(lines[len(lines)-1])
	if path == "" {
		return "", fmt.Errorf("org: %s ensure --id %s printed no worktree path", worktreeScriptRel, req.ID)
	}
	return path, nil
}

// CurrentBranch runs `git -C <path> branch --show-current`.
func (scriptFeatureWorktrees) CurrentBranch(path string) (string, error) {
	out, err := runCommand(exec.Command("git", "-C", path, "branch", "--show-current"))
	if err != nil {
		return "", fmt.Errorf("org: git -C %s branch --show-current: %w", path, err)
	}
	return strings.TrimSpace(out), nil
}

// runWorktreeScript runs `bash <repoRoot>/scripts/ralph-worktree.sh <args>`
// in repoRoot and returns its stdout. A missing script wraps
// errWorktreeScriptMissing.
func runWorktreeScript(repoRoot string, args ...string) (string, error) {
	script := filepath.Join(repoRoot, filepath.FromSlash(worktreeScriptRel))
	if info, err := os.Stat(script); err != nil || !info.Mode().IsRegular() {
		return "", fmt.Errorf("org: %s does not exist: start --plan makes the feature worktree with it; %w", script, errWorktreeScriptMissing)
	}
	cmd := exec.Command("bash", append([]string{script}, args...)...)
	cmd.Dir = repoRoot
	out, err := runCommand(cmd)
	if err != nil {
		return "", fmt.Errorf("org: %s %s: %w", worktreeScriptRel, args[0], err)
	}
	return out, nil
}

// runCommand runs cmd and returns its stdout. A failure is an error with
// what cmd printed on stderr and its exit status.
func runCommand(cmd *exec.Cmd) (string, error) {
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	if err := cmd.Run(); err != nil {
		if msg := strings.TrimSpace(stderr.String()); msg != "" {
			return "", fmt.Errorf("%s (%w)", msg, err)
		}
		return "", err
	}
	return stdout.String(), nil
}
