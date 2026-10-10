package org

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

// `ralph org start --plan` at the org layer (plan
// docs/plans/active/2026-10-09-org-feature-worktree.md, AC3, AC4, AC6 to AC9,
// and plan docs/plans/active/2026-10-10-org-feature-worktree-ownership.md,
// AC1 to AC3b: a feature worktree stays with the split plan that made it).
// splitPlanFixture, approveSplitPlan and writeSplitPlan live in
// split_test.go; testFeature, reserveEvent in reserve_test.go;
// featureLeaderParams, spawnFeatureLeaderIn, assertActiveFeature and
// takeSpawnSnapshot in spawn_feature_test.go.

// fakeWorktrees is a FeatureWorktrees that keeps ralph-worktree.sh's records
// and each worktree's checked-out branch in memory. Ensure behaves like the
// script: it returns a recorded worktree with the same path, branch and kind
// as it is (canonical_ref untouched), refuses a record that differs, and
// otherwise records a new one and makes its directory for real, so
// StartFeature's directory check sees it.
type fakeWorktrees struct {
	records   map[string]WorktreeRecord // state id -> record
	checkouts map[string]string         // worktree path -> branch checked out ("" detached)

	lookupErr error
	ensureErr error
	branchErr error
	// ensureHook, when non-nil, runs first in Ensure: a racing spawn that
	// changes the manifest after StartFeature's unlocked pre-check.
	ensureHook func()
	// ensurePath, when non-empty, is what Ensure returns for a new worktree
	// instead of its path.
	ensurePath string
	// ensureDrops makes Ensure forget the record of a new worktree before it
	// returns, as a cleanup run right after it would.
	ensureDrops bool

	calls   []string // "lookup:<id>", "ensure:<id>", "branch:<path>", in order
	ensured []EnsureWorktree
	roots   []string // repoRoot of each Lookup and Ensure call, in order
}

func newFakeWorktrees() *fakeWorktrees {
	return &fakeWorktrees{records: map[string]WorktreeRecord{}, checkouts: map[string]string{}}
}

func (f *fakeWorktrees) Lookup(repoRoot, id string) (WorktreeRecord, bool, error) {
	f.calls = append(f.calls, "lookup:"+id)
	f.roots = append(f.roots, repoRoot)
	if f.lookupErr != nil {
		return WorktreeRecord{}, false, f.lookupErr
	}
	rec, ok := f.records[id]
	return rec, ok, nil
}

func (f *fakeWorktrees) Ensure(repoRoot string, req EnsureWorktree) (string, error) {
	f.calls = append(f.calls, "ensure:"+req.ID)
	f.ensured = append(f.ensured, req)
	f.roots = append(f.roots, repoRoot)
	if f.ensureHook != nil {
		f.ensureHook()
	}
	if f.ensureErr != nil {
		return "", f.ensureErr
	}
	path := filepath.Join(repoRoot, filepath.FromSlash(req.Path))
	if rec, ok := f.records[req.ID]; ok {
		if rec.WorktreePath != path || rec.Branch != req.Branch || rec.Kind != req.Kind {
			return "", fmt.Errorf("ralph-worktree: state collision for id '%s'", req.ID)
		}
		return path, nil
	}
	if err := os.MkdirAll(path, 0o755); err != nil {
		return "", err
	}
	f.records[req.ID] = WorktreeRecord{WorktreePath: path, Branch: req.Branch, Kind: req.Kind, CanonicalRef: req.CanonicalRef}
	f.checkouts[path] = req.Branch
	if f.ensureDrops {
		delete(f.records, req.ID)
	}
	if f.ensurePath != "" {
		return f.ensurePath, nil
	}
	return path, nil
}

func (f *fakeWorktrees) CurrentBranch(path string) (string, error) {
	f.calls = append(f.calls, "branch:"+path)
	if f.branchErr != nil {
		return "", f.branchErr
	}
	branch, ok := f.checkouts[path]
	if !ok {
		return "", fmt.Errorf("fatal: not a git repository: %s", path)
	}
	return branch, nil
}

// featureStart is a StartFeature fixture: an Org on fake drivers and fake
// worktrees, its state dir holding the approved split plan auth-split
// (splitPlanFixture: auth-core, Type fix, and auth-docs, depending on it),
// and a repo root.
type featureStart struct {
	o        *Org
	h        *fakeHerdr
	a        *fakeAgmsg
	wt       *fakeWorktrees
	stateDir string
	root     string // symlinks resolved, as StartFeature resolves RepoRoot
	plan     string
}

func newFeatureStart(t *testing.T) *featureStart {
	t.Helper()
	return newFeatureStartIn(t, "")
}

// newFeatureStartIn is newFeatureStart with its ledger, the manifest, the
// receipts and splits/, in stateDir; "" leaves it in testOrg's directory.
func newFeatureStartIn(t *testing.T, stateDir string) *featureStart {
	t.Helper()
	o, h, a := testOrg(t)
	if stateDir != "" {
		o.Manifest = NewManifestStoreAtPath(ManifestPathIn(stateDir))
		o.Receipts = NewReceiptStoreAtPath(filepath.Join(stateDir, "receipts.jsonl"))
	}
	wt := newFakeWorktrees()
	o.Worktrees = wt
	st := &featureStart{o: o, h: h, a: a, wt: wt, stateDir: filepath.Dir(o.Manifest.Path()), root: resolvedOrClean(t.TempDir())}
	if err := os.MkdirAll(SplitPlansDirIn(st.stateDir), 0o755); err != nil {
		t.Fatal(err)
	}
	st.plan = st.writePlan(t, "auth-split", approveSplitPlan(t, splitPlanFixture))
	return st
}

// writePlan writes content as split plan id and returns its path.
func (st *featureStart) writePlan(t *testing.T, id, content string) string {
	t.Helper()
	return writeSplitPlan(t, SplitPlansDirIn(st.stateDir), id+".md", content)
}

func (st *featureStart) params(feature string) StartFeatureParams {
	return StartFeatureParams{
		StateDir: st.stateDir, RepoRoot: st.root, SplitPath: st.plan, Feature: feature,
		Driver: "claude", Model: "sonnet", TimeoutMS: 5000,
	}
}

// worktree is the feature worktree StartFeature uses for orgID.
func (st *featureStart) worktree(orgID string) string {
	return filepath.Join(st.root, ".claude", "worktrees", "org-"+orgID)
}

// ref is the canonical_ref StartFeature records for feature slug of st.plan.
func (st *featureStart) ref(slug string) string {
	return splitRef(st.plan, slug)
}

// splitRef is the canonical_ref of feature slug of the split plan at path:
// the path with its symlinks resolved, as ResolveSplitPlanPath gives it.
func splitRef(path, slug string) string {
	return "split:" + resolvedOrClean(path) + "#" + slug
}

// planDigestOf is the current digest of the split plan at path.
func planDigestOf(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return PlanDigest(data)
}

// mustStart runs StartFeature and fails unless the leader spawn ended
// wantOutcome with no error.
func (st *featureStart) mustStart(t *testing.T, p StartFeatureParams, wantOutcome SpawnOutcome) StartFeatureResult {
	t.Helper()
	res := st.o.StartFeature(p)
	if res.Spawn.Outcome != wantOutcome || res.Spawn.Err != nil {
		t.Fatalf("expected StartFeature to end %s, got %+v", wantOutcome, res.Spawn)
	}
	return res
}

// assertStartRefused checks that res is a rejection whose error contains every
// string in want, with no worktree.
func assertStartRefused(t *testing.T, res StartFeatureResult, want ...string) {
	t.Helper()
	if res.Spawn.Outcome != SpawnOutcomeRejected || res.Spawn.Err == nil {
		t.Fatalf("expected a rejection, got %+v", res.Spawn)
	}
	for _, w := range want {
		if !strings.Contains(res.Spawn.Err.Error(), w) {
			t.Errorf("error %q does not contain %q", res.Spawn.Err, w)
		}
	}
	if res.Worktree != "" || res.Branch != "" {
		t.Errorf("expected no worktree in the result, got %q on %q", res.Worktree, res.Branch)
	}
}

// wantCalls checks the fake worktrees' calls.
func (st *featureStart) wantCalls(t *testing.T, want ...string) {
	t.Helper()
	if !slices.Equal(st.wt.calls, want) {
		t.Fatalf("worktree calls = %q, want %q", st.wt.calls, want)
	}
}

// eventOf returns the latest event of o's manifest named name for
// orgID/seatID.
func eventOf(t *testing.T, o *Org, orgID, seatID, name string) ManifestEvent {
	t.Helper()
	events := mustReadEvents(t, o)
	for i := len(events) - 1; i >= 0; i-- {
		if ev := events[i]; ev.OrgID == orgID && ev.SeatID == seatID && ev.Event == name {
			return ev
		}
	}
	t.Fatalf("no %s event for %s/%s", name, orgID, seatID)
	return ManifestEvent{}
}

// TestStartFeature_StartsLeaderInFeatureWorktree covers plan AC3 at the org
// layer: ensure is asked for the worktree org-<org_id> on <type>/<slug> with
// the split plan feature (its plan's resolved path and the slug) as
// canonical_ref, in the repo root, after the record lookup, and the record is
// read again after it; the leader is spawned with that worktree as its cwd,
// the feature's paths reserved together with the binding, the scope line,
// and a task holding the ledger and the feature's body.
func TestStartFeature_StartsLeaderInFeatureWorktree(t *testing.T) {
	st := newFeatureStart(t)
	res := st.mustStart(t, st.params("auth-core"), SpawnOutcomeSpawned)

	wt, digest := st.worktree("auth-core"), planDigestOf(t, st.plan)
	if res.OrgID != "auth-core" || res.Worktree != wt || res.Branch != "fix/auth-core" || res.Split == nil ||
		res.Split.ID != "auth-split" || res.Split.Digest != digest {
		t.Fatalf("unexpected result: org %q worktree %q branch %q split %+v", res.OrgID, res.Worktree, res.Branch, res.Split)
	}
	st.wantCalls(t, "lookup:org-auth-core", "ensure:org-auth-core", "lookup:org-auth-core", "branch:"+wt)
	wantEnsure := EnsureWorktree{
		ID: "org-auth-core", Kind: "org", Branch: "fix/auth-core", Path: ".claude/worktrees/org-auth-core",
		CanonicalRef: "split:" + resolvedOrClean(st.plan) + "#auth-core", CleanupPolicy: "manual",
	}
	if !slices.Equal(st.wt.ensured, []EnsureWorktree{wantEnsure}) || !slices.Equal(st.wt.roots, []string{st.root, st.root, st.root}) {
		t.Fatalf("ensure = %+v in %q, want %+v in %s", st.wt.ensured, st.wt.roots, wantEnsure, st.root)
	}

	events := mustReadEvents(t, st.o)
	wantDetails := "paths=docs/auth.md,internal/auth/ split=auth-split feature=auth-core digest=" + digest + " branch=fix/auth-core"
	if ev := events[0]; ev.Event != EventScopeReserved || ev.OrgID != "auth-core" || ev.Details != wantDetails || ev.Worktree != wt {
		t.Fatalf("expected the bound scope_reserved first (Details %q, Worktree %q), got %+v", wantDetails, wt, ev)
	}
	assertActiveFeature(t, st.o, "auth-core", &FeatureBinding{
		Split: "auth-split", Feature: "auth-core", Digest: digest, Branch: "fix/auth-core", Worktree: wt,
	})
	spawned := eventOf(t, st.o, "auth-core", LeaderIdentity, EventSpawned)
	if spawned.Worktree != wt || spawned.Role != LeaderIdentity || spawned.Driver != "claude" || spawned.Model != "sonnet" ||
		!strings.Contains(spawned.Details, "scope=split auth-split feature auth-core (reserve: docs/auth.md, internal/auth/)") {
		t.Fatalf("expected the leader spawned in %s with the scope line, got %+v", wt, spawned)
	}
	if len(st.a.joinCalls) != 1 || st.a.joinCalls[0].projectPath != wt || st.a.joinCalls[0].agentID != LeaderIdentity {
		t.Fatalf("expected the leader to join agmsg from %s, got %+v", wt, st.a.joinCalls)
	}

	feature, _ := res.Split.Feature("auth-core")
	task := featureLeaderTask(res.Split, feature, st.stateDir, wt, "fix/auth-core")
	if !strings.HasSuffix(task, "\n\nObjective: add the token store.\n\n- [ ] AC1: tokens persist") {
		t.Fatalf("expected the task to end with the feature's body, got %q", task)
	}
	promptPath, err := st.o.promptFilePath("auth-core", LeaderIdentity)
	if err != nil {
		t.Fatal(err)
	}
	prompt, err := os.ReadFile(promptPath)
	if err != nil {
		t.Fatalf("read the leader's prompt: %v", err)
	}
	if !strings.Contains(string(prompt), "## タスク\n\n"+task+"\n") || !strings.Contains(string(prompt), "- scope: split auth-split feature auth-core") {
		t.Fatalf("expected the leader's prompt to hold the task and the scope, got:\n%s", prompt)
	}
	// The spawned task names the ledger start recorded the leader in, the
	// state dir of st.o's manifest.
	if ledger := "\n- 台帳: " + st.stateDir + "(ralph org のコマンドには必ず --state-dir '" + st.stateDir + "' を付ける)\n"; !strings.Contains(string(prompt), ledger) {
		t.Fatalf("expected the leader's task to name the ledger with %q, got:\n%s", ledger, prompt)
	}
}

// TestStartFeature_OrgIDAndTypeDefault: --org-id names the org, the state id
// and the worktree, while the branch and the binding stay the feature's; a
// feature without `- Type:` gets a feat/ branch; RepoRoot is used with its
// symlinks resolved (given here through a symlink to the repo).
func TestStartFeature_OrgIDAndTypeDefault(t *testing.T) {
	st := newFeatureStart(t)
	link := filepath.Join(t.TempDir(), "repo-link")
	if err := os.Symlink(st.root, link); err != nil {
		t.Fatal(err)
	}
	p := st.params("auth-docs")
	p.RepoRoot, p.OrgID = link, "docs-x"
	res := st.mustStart(t, p, SpawnOutcomeSpawned)

	wt := st.worktree("docs-x")
	if res.OrgID != "docs-x" || res.Worktree != wt || res.Branch != "feat/auth-docs" {
		t.Fatalf("unexpected result: org %q worktree %q branch %q", res.OrgID, res.Worktree, res.Branch)
	}
	if e := st.wt.ensured[0]; e.ID != "org-docs-x" || e.Path != ".claude/worktrees/org-docs-x" || e.Branch != "feat/auth-docs" ||
		e.CanonicalRef != st.ref("auth-docs") {
		t.Fatalf("unexpected ensure %+v", e)
	}
	assertActiveFeature(t, st.o, "docs-x", &FeatureBinding{
		Split: "auth-split", Feature: "auth-docs", Digest: res.Split.Digest, Branch: "feat/auth-docs", Worktree: wt,
	})
}

// TestStartFeature_CanonicalRefIsTheResolvedPlanPath covers the shape in
// AC1 of plan docs/plans/active/2026-10-10-org-feature-worktree-ownership.md:
// the canonical_ref is split:<the split plan's absolute path>#<slug>, the
// path with its symlinks resolved, so a start that names the plan through a
// symlink to its ledger records the plan's own path, and the same start that
// names the plan directly reuses that worktree.
func TestStartFeature_CanonicalRefIsTheResolvedPlanPath(t *testing.T) {
	if got, want := featureCanonicalRef("/repo/.harness/state/org/splits/auth-split.md", "auth-core"),
		"split:/repo/.harness/state/org/splits/auth-split.md#auth-core"; got != want {
		t.Fatalf("featureCanonicalRef = %q, want %q", got, want)
	}
	st := newFeatureStart(t)
	link := filepath.Join(t.TempDir(), "ledger-link")
	if err := os.Symlink(st.stateDir, link); err != nil {
		t.Fatal(err)
	}
	p := st.params("auth-core")
	p.SplitPath = filepath.Join(link, "splits", "auth-split.md")
	st.mustStart(t, p, SpawnOutcomeSpawned)
	if got := st.wt.ensured[0].CanonicalRef; got != "split:"+resolvedOrClean(st.plan)+"#auth-core" || strings.Contains(got, link) {
		t.Fatalf("expected the canonical_ref to hold the plan's resolved path, not %s, got %q", link, got)
	}
	st.mustStart(t, st.params("auth-core"), SpawnOutcomeIdempotent)
	wt := st.worktree("auth-core")
	st.wantCalls(t, "lookup:org-auth-core", "ensure:org-auth-core", "lookup:org-auth-core", "branch:"+wt,
		"lookup:org-auth-core", "branch:"+wt, "ensure:org-auth-core", "lookup:org-auth-core", "branch:"+wt)
}

// TestStartFeature_OrgIDAtTheLengthLimit pins maxFeatureOrgIDLen at 20,
// herdr's 32-character agent name less the `_` and "implementer": a feature
// org with an --org-id of 20 characters starts its leader, and the leader
// can then spawn the implementer seat in it. One character more is refused
// (TestStartFeature_RefusedBeforeAnyRecord).
func TestStartFeature_OrgIDAtTheLengthLimit(t *testing.T) {
	if maxFeatureOrgIDLen != 20 || implementerSeatID != "implementer" {
		t.Fatalf("maxFeatureOrgIDLen, implementerSeatID = %d, %q, want 20, implementer", maxFeatureOrgIDLen, implementerSeatID)
	}
	st := newFeatureStart(t)
	p := st.params("auth-core")
	p.OrgID = "a" + strings.Repeat("b", maxFeatureOrgIDLen-1)
	if res := st.mustStart(t, p, SpawnOutcomeSpawned); res.OrgID != p.OrgID || res.Worktree != st.worktree(p.OrgID) {
		t.Fatalf("unexpected result: org %q worktree %q", res.OrgID, res.Worktree)
	}
	st.h.paneID = "pane-2"
	impl := mustSpawnParams(p.OrgID, implementerSeatID)
	impl.Role = implementerSeatID
	if r := st.o.Spawn(impl); r.Outcome != SpawnOutcomeSpawned {
		t.Fatalf("expected the leader of a %d-character org_id to spawn %s, got %+v", len(p.OrgID), implementerSeatID, r)
	}
}

// TestStartFeature_SlugAtTheLengthLimit is the default org_id's side of
// maxFeatureOrgIDLen: without --org-id, a feature whose slug has 20
// characters starts its leader in the org named by the slug, and the leader
// can then spawn the implementer seat in it. A slug of 21 characters is
// refused when the split plan is read, before the worktree record is looked
// up, with nothing recorded.
func TestStartFeature_SlugAtTheLengthLimit(t *testing.T) {
	splitWith := func(slug string) string {
		return "# long slug\n\n- Status: Approved\n- Approved: 2026-10-09 sha256:000000000000\n\n## Features\n\n### " +
			slug + "\n\n- Reserve: internal/long/\n\nObjective: a long slug.\n"
	}
	t.Run("20 characters", func(t *testing.T) {
		st := newFeatureStart(t)
		slug := "a" + strings.Repeat("b", maxFeatureOrgIDLen-1)
		st.plan = st.writePlan(t, "long-split", approveSplitPlan(t, splitWith(slug)))
		res := st.mustStart(t, st.params(slug), SpawnOutcomeSpawned)
		if res.OrgID != slug || res.Worktree != st.worktree(slug) || res.Branch != "feat/"+slug {
			t.Fatalf("unexpected result: org %q worktree %q branch %q", res.OrgID, res.Worktree, res.Branch)
		}
		st.h.paneID = "pane-2"
		impl := mustSpawnParams(slug, implementerSeatID)
		impl.Role = implementerSeatID
		if r := st.o.Spawn(impl); r.Outcome != SpawnOutcomeSpawned {
			t.Fatalf("expected the leader of a %d-character slug to spawn %s, got %+v", len(slug), implementerSeatID, r)
		}
	})
	t.Run("21 characters", func(t *testing.T) {
		st := newFeatureStart(t)
		slug := "a" + strings.Repeat("b", maxFeatureOrgIDLen)
		st.plan = st.writePlan(t, "long-split", approveSplitPlan(t, splitWith(slug)))
		res := st.o.StartFeature(st.params(slug))
		assertStartRefused(t, res, `feature slug "`+slug+`" is 21 characters`, "use a shorter slug")
		if res.Split != nil {
			t.Errorf("expected the split plan not read, got %+v", res.Split)
		}
		st.wantCalls(t)
		if after := takeSpawnSnapshot(t, st.o, st.h, st.a); after != (spawnSnapshot{}) {
			t.Fatalf("expected nothing recorded or called, got %+v", after)
		}
	})
}

// TestFeatureLeaderTask pins the leader's task text: the lines naming the
// split plan, feature, worktree, branch, reserved paths, dependencies,
// procedure and ledger, then the feature's body after a blank line; no
// dependency reads なし, and an empty body adds nothing. The ledger line's
// --state-dir is one POSIX shell word, also for a path with a space or a
// single quote.
func TestFeatureLeaderTask(t *testing.T) {
	plan := &SplitPlan{ID: "auth-split", Path: "/state/splits/auth-split.md", Digest: "0123456789ab"}
	head := func(slug, worktree, branch, reserve, deps, ledger string) string {
		return "- 分割計画: auth-split(/state/splits/auth-split.md、承認の digest 0123456789ab)\n" +
			"- 機能: " + slug + "\n" +
			"- worktree: " + worktree + "(leader の cwd)\n" +
			"- ブランチ: " + branch + "\n" +
			"- 予約したパス: " + reserve + "\n" +
			"- 依存する機能: " + deps + "\n" +
			"- 進め方: `/org` skill の「機能ごとの org」の手順に従う\n" +
			"- 台帳: " + ledger
	}
	withBody := SplitFeature{
		Slug: "auth-docs", Type: "feat", Reserve: []string{"docs/auth/", "docs/x.md"},
		DependsOn: []string{"auth-core", "auth-api"}, Body: "Objective: document it.\n\n- [ ] AC1",
	}
	got := featureLeaderTask(plan, withBody, "/repo/.harness/state/org", "/repo/.claude/worktrees/org-auth-docs", "feat/auth-docs")
	want := head("auth-docs", "/repo/.claude/worktrees/org-auth-docs", "feat/auth-docs", "docs/auth/, docs/x.md", "auth-core, auth-api",
		"/repo/.harness/state/org(ralph org のコマンドには必ず --state-dir '/repo/.harness/state/org' を付ける)") +
		"\n\nObjective: document it.\n\n- [ ] AC1"
	if got != want {
		t.Errorf("task =\n%s\nwant\n%s", got, want)
	}
	bare := SplitFeature{Slug: "auth-core", Type: "fix", Reserve: []string{"internal/auth/"}}
	for _, c := range []struct{ stateDir, ledger string }{
		{"/my ledgers/org", "/my ledgers/org(ralph org のコマンドには必ず --state-dir '/my ledgers/org' を付ける)"},
		{"/it's/org", `/it's/org(ralph org のコマンドには必ず --state-dir '/it'\''s/org' を付ける)`},
	} {
		if got, want := featureLeaderTask(plan, bare, c.stateDir, "/wt", "fix/auth-core"), head("auth-core", "/wt", "fix/auth-core", "internal/auth/", "なし", c.ledger); got != want {
			t.Errorf("task =\n%s\nwant\n%s", got, want)
		}
	}
}

// TestFeatureLeaderTask_StateDirIsOneShellWord: the --state-dir value of the
// ledger line, run through sh, is the state dir as one argument, for paths
// with a space, a single quote, and characters the shell would expand.
func TestFeatureLeaderTask_StateDirIsOneShellWord(t *testing.T) {
	plan := &SplitPlan{ID: "s", Path: "/state/splits/s.md", Digest: "0123456789ab"}
	f := SplitFeature{Slug: "a", Type: "feat", Reserve: []string{"a/"}}
	for _, dir := range []string{"/plain/org", "/my ledgers/org", "/it's/org", "/a'b'/c", `/$HOME/*/"q"/\x/` + "`id`"} {
		task := featureLeaderTask(plan, f, dir, "/wt", "feat/a")
		_, rest, ok := strings.Cut(task, "--state-dir ")
		word, _, ok2 := strings.Cut(rest, " を付ける)")
		if !ok || !ok2 {
			t.Fatalf("no --state-dir <word> を付ける in the task:\n%s", task)
		}
		got, err := exec.Command("sh", "-c", "set -- "+word+`; printf '%s|%s' "$#" "$1"`).Output()
		if err != nil {
			t.Fatalf("sh with %s: %v", word, err)
		}
		if want := "1|" + dir; string(got) != want {
			t.Errorf("--state-dir %s reads in sh as %q, want %q", word, got, want)
		}
	}
}

// TestStartFeature_RefusedBeforeAnyRecord covers plan AC4 at the org layer,
// and the other refusals that need neither the manifest nor a worktree: an
// unapproved plan, one without an approval digest, one changed after its
// approval, one with a `- Branch:` line added after its approval, an unknown
// feature, a plan outside splits/, a bad org_id, an --org-id one character
// longer than maxFeatureOrgIDLen, a model outside the pool (the pre-check),
// and no repo root. None looks up or makes a worktree, or writes, receipts
// or calls anything.
func TestStartFeature_RefusedBeforeAnyRecord(t *testing.T) {
	approved := approveSplitPlan(t, splitPlanFixture)
	for _, tc := range []struct {
		name      string
		content   string // the split plan auth-split; "" leaves the approved fixture
		change    func(st *featureStart, p *StartFeatureParams)
		want      []string
		wantSplit bool // whether the plan was read
	}{
		{"not approved", approveSplitPlan(t, strings.Replace(splitPlanFixture, "- Status: Approved", "- Status: Draft", 1)), nil,
			[]string{`not approved: - Status: is "Draft", want Approved`}, true},
		{"no approval digest", strings.Replace(splitPlanFixture, "- Approved: 2026-10-09 sha256:000000000000", "- Approved: 2026-10-09", 1), nil,
			[]string{"no approval digest"}, true},
		{"changed after its approval", strings.Replace(approved, "Objective: add the token store.", "Objective: add the token store, then drop the old one.", 1), nil,
			[]string{"changed after its approval"}, true},
		{"a - Branch: line added after its approval", strings.Replace(approved, "- Depends on: none\n", "- Depends on: none\n- Branch: feat/elsewhere\n", 1), nil,
			[]string{"a - Branch: line is not allowed in a split plan"}, false},
		{"an unknown feature", "", func(_ *featureStart, p *StartFeatureParams) { p.Feature = "nope" },
			[]string{`has no feature "nope": its features are auth-core, auth-docs`}, true},
		{"a plan outside splits/", "", func(st *featureStart, p *StartFeatureParams) {
			p.SplitPath = writeSplitPlan(t, st.stateDir, "auth-split.md", approved)
		}, []string{"not directly in", "a split plan is a file <id>.md directly in"}, false},
		{"a bad --org-id", "", func(_ *featureStart, p *StartFeatureParams) { p.OrgID = "Auth" },
			[]string{`invalid org_id "Auth"`}, true},
		{"an --org-id of 21 characters", "", func(_ *featureStart, p *StartFeatureParams) { p.OrgID = "a" + strings.Repeat("b", 20) },
			[]string{`org: org_id "a` + strings.Repeat("b", 20) + `" is 21 characters: a feature org's org_id is at most 20 characters, ` +
				`so that its leader can spawn the seat "implementer" within herdr's 32-character agent name <org_id>_<seat_id>; use a shorter --org-id`}, true},
		{"a model outside the pool", "", func(_ *featureStart, p *StartFeatureParams) { p.Model = "gpt-9" },
			[]string{`model "gpt-9" not in [org].model_pool for driver "claude"`}, true},
		{"no repo root", "", func(_ *featureStart, p *StartFeatureParams) { p.RepoRoot = "" },
			[]string{"needs the root of the main worktree"}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			st := newFeatureStart(t)
			if tc.content != "" {
				st.writePlan(t, "auth-split", tc.content)
			}
			p := st.params("auth-core")
			if tc.change != nil {
				tc.change(st, &p)
			}
			res := st.o.StartFeature(p)
			assertStartRefused(t, res, tc.want...)
			if (res.Split != nil) != tc.wantSplit {
				t.Errorf("Split = %+v, want read=%v", res.Split, tc.wantSplit)
			}
			st.wantCalls(t)
			if after := takeSpawnSnapshot(t, st.o, st.h, st.a); after != (spawnSnapshot{}) {
				t.Fatalf("expected nothing recorded or called, got %+v", after)
			}
		})
	}
}

// TestStartFeature_PrecheckRefusesWithoutWorktree covers plan AC7 and the
// first half of AC8: the unlocked pre-check refuses a start into a running
// org that was not started for the feature (a promoted session's leader's
// org, the org of `ralph org start <task>`, an org with a plain --reserve),
// and one that max_orgs, max_total_seats or another org's overlapping
// reservation stops, before the worktree record is even looked up, and
// without the `rejected` record a Spawn of the leader would write.
func TestStartFeature_PrecheckRefusesWithoutWorktree(t *testing.T) {
	for _, tc := range []struct {
		name  string
		setup func(t *testing.T, st *featureStart)
		want  string
	}{
		{"a promoted leader's org", func(t *testing.T, st *featureStart) {
			spawnSeatIn(t, st.o, st.h, "auth-core", "seat-1", "ws-1", "pane-1")
		}, `org_id "auth-core" is running without a split plan`},
		{"ralph org start <task>'s org", func(t *testing.T, st *featureStart) {
			spawnLeaderIn(t, st.o, st.h, "auth-core", "ws-1", "pane-1")
		}, `org_id "auth-core" is running without a split plan`},
		{"an org with a plain --reserve", func(t *testing.T, st *featureStart) {
			spawnLeaderIn(t, st.o, st.h, "auth-core", "ws-1", "pane-1", "internal/auth/", "docs/auth.md")
		}, `org_id "auth-core" was started without a split plan`},
		{"max_orgs", func(t *testing.T, st *featureStart) {
			st.o.Config.MaxOrgs = 1
			spawnLeaderIn(t, st.o, st.h, "other", "ws-1", "pane-1", "web/")
		}, `max_orgs 1 reached: org_id "auth-core" is not running`},
		{"max_total_seats", func(t *testing.T, st *featureStart) {
			st.o.Config.MaxTotalSeats = 1
			spawnSeatIn(t, st.o, st.h, "other", "seat-1", "ws-1", "pane-1")
		}, "max_total_seats 1 reached"},
		{"an overlapping reservation", func(t *testing.T, st *featureStart) {
			spawnLeaderIn(t, st.o, st.h, "other", "ws-1", "pane-1", "internal/")
		}, `reservation overlaps the reservation of running org_id "other": internal/auth/ overlaps internal/`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			st := newFeatureStart(t)
			tc.setup(t, st)
			before := takeSpawnSnapshot(t, st.o, st.h, st.a)
			assertStartRefused(t, st.o.StartFeature(st.params("auth-core")), tc.want)
			st.wantCalls(t)
			if after := takeSpawnSnapshot(t, st.o, st.h, st.a); after != before {
				t.Fatalf("expected nothing recorded or called, %+v -> %+v", before, after)
			}
			assertActiveFeature(t, st.o, "auth-core", nil)
		})
	}
}

// TestStartFeature_SameStartAgainReusesWorktree covers the first sentence of
// plan AC6 (and of AC1 of plan
// docs/plans/active/2026-10-10-org-feature-worktree-ownership.md): the same
// start again reuses the worktree (ensure returns it, and the record read
// again after ensure is still the feature's) and, with the leader active,
// records and calls nothing; after the leader stopped it spawns a new leader
// in the same worktree with no second reservation record.
func TestStartFeature_SameStartAgainReusesWorktree(t *testing.T) {
	st := newFeatureStart(t)
	first := st.mustStart(t, st.params("auth-core"), SpawnOutcomeSpawned)

	before := takeSpawnSnapshot(t, st.o, st.h, st.a)
	again := st.mustStart(t, st.params("auth-core"), SpawnOutcomeIdempotent)
	if after := takeSpawnSnapshot(t, st.o, st.h, st.a); after != before {
		t.Fatalf("expected nothing recorded or called, %+v -> %+v", before, after)
	}
	if again.Worktree != first.Worktree || again.Branch != first.Branch {
		t.Fatalf("expected the same worktree, got %q on %q, then %q on %q", first.Worktree, first.Branch, again.Worktree, again.Branch)
	}
	wt := st.worktree("auth-core")
	st.wantCalls(t, "lookup:org-auth-core", "ensure:org-auth-core", "lookup:org-auth-core", "branch:"+wt,
		"lookup:org-auth-core", "branch:"+wt, "ensure:org-auth-core", "lookup:org-auth-core", "branch:"+wt)

	if r := st.o.Stop(StopParams{OrgID: "auth-core", Seat: LeaderIdentity}); r.Err != nil {
		t.Fatalf("stop the leader: %v", r.Err)
	}
	st.h.paneID = "pane-2"
	st.mustStart(t, st.params("auth-core"), SpawnOutcomeSpawned)
	if n := countString(eventNames(t, st.o), EventScopeReserved); n != 1 {
		t.Fatalf("expected one scope_reserved, got %d", n)
	}
	if ev := eventOf(t, st.o, "auth-core", LeaderIdentity, EventSpawned); ev.PaneID != "pane-2" || ev.Worktree != wt {
		t.Fatalf("expected the new leader in %s, got %+v", wt, ev)
	}
	if len(st.wt.records) != 1 {
		t.Fatalf("expected one worktree record, got %+v", st.wt.records)
	}
}

// TestStartFeature_OtherBindingRefusedWithoutWorktree covers the second
// sentence of plan AC6: into an org bound to a feature, another feature (by
// --org-id) and the same feature from the plan approved again (another
// digest) are refused by the pre-check, before the worktree record is looked
// up, with nothing recorded.
func TestStartFeature_OtherBindingRefusedWithoutWorktree(t *testing.T) {
	st := newFeatureStart(t)
	st.mustStart(t, st.params("auth-core"), SpawnOutcomeSpawned)
	oldDigest := planDigestOf(t, st.plan)
	calls := slices.Clone(st.wt.calls)
	const boundTo = `org_id "auth-core" is bound to feature "auth-core" of split plan "auth-split"`

	before := takeSpawnSnapshot(t, st.o, st.h, st.a)
	other := st.params("auth-docs")
	other.OrgID = "auth-core"
	assertStartRefused(t, st.o.StartFeature(other), boundTo, `for feature "auth-docs" of split plan "auth-split"`)

	st.writePlan(t, "auth-split", approveSplitPlan(t, strings.Replace(splitPlanFixture, "Objective: add the token store.", "Objective: add the token store v2.", 1)))
	newDigest := planDigestOf(t, st.plan)
	assertStartRefused(t, st.o.StartFeature(st.params("auth-core")), boundTo+` (digest "`+oldDigest+`"`, `(digest "`+newDigest+`"`)

	if after := takeSpawnSnapshot(t, st.o, st.h, st.a); after != before {
		t.Fatalf("expected nothing recorded or called, %+v -> %+v", before, after)
	}
	st.wantCalls(t, calls...)
}

// TestStartFeature_WorktreeRecordMismatchRefused covers the reuse check
// (plan AC6, Codex plan advisory finding 1): a worktree record for the
// org_id is reused only when its canonical_ref, worktree path, branch and
// kind are this start's and that branch is checked out in it. Every other
// record, a record that cannot be read, and a worktree whose branch cannot
// be read are refused before ensure, naming what differs. A record with
// another canonical_ref, which another split plan's feature or a task may
// still be using (also the record PR #216 wrote for this feature, before the
// canonical_ref held the plan's path), points first at another slug and at
// the cleanup only for a worktree that is no longer needed, and not at
// another --org-id, which would keep the branch (AC3b of plan
// docs/plans/active/2026-10-10-org-feature-worktree-ownership.md); the other
// refusals name the cleanup. A missing ralph-worktree.sh comes back as it is.
func TestStartFeature_WorktreeRecordMismatchRefused(t *testing.T) {
	const cleanup = "so start does not reuse it: remove the worktree with ./scripts/ralph-worktree.sh cleanup --id org-auth-core " +
		"if it is no longer needed, or use another --org-id"
	const otherRefHint = "so start does not reuse it: it was made for something other than this split plan's feature, " +
		"which may still be using that worktree and its branch; change this feature's slug in the split plan and approve it again " +
		"(the slug is the default org_id and part of the branch fix/auth-core), or, if that worktree and its branch are no longer needed, " +
		"remove them with ./scripts/ralph-worktree.sh cleanup --id org-auth-core"
	boom := errors.New("boom")
	for _, tc := range []struct {
		name     string
		change   func(st *featureStart, rec *WorktreeRecord) // edits the matching record before it is stored; nil: no directory for it
		want     func(st *featureStart) string
		otherRef bool // the record has another canonical_ref
	}{
		{"another split plan", func(_ *featureStart, rec *WorktreeRecord) {
			rec.CanonicalRef = "split:/other/org/splits/auth-split.md#auth-core"
		}, func(st *featureStart) string {
			return `records canonical_ref "split:/other/org/splits/auth-split.md#auth-core", not "` + st.ref("auth-core") + `"`
		}, true},
		{"this feature's record from PR #216", func(_ *featureStart, rec *WorktreeRecord) { rec.CanonicalRef = "split:auth-split#auth-core" },
			func(st *featureStart) string {
				return `records canonical_ref "split:auth-split#auth-core", not "` + st.ref("auth-core") + `"`
			}, true},
		{"a task worktree", func(_ *featureStart, rec *WorktreeRecord) { rec.CanonicalRef, rec.Kind = "", "task" },
			func(st *featureStart) string {
				return `canonical_ref "", not "` + st.ref("auth-core") + `", kind "task", not "org"`
			}, true},
		{"another path", func(st *featureStart, rec *WorktreeRecord) { rec.WorktreePath = filepath.Join(st.root, "elsewhere") },
			func(*featureStart) string { return `records worktree_path "` }, false},
		{"another branch recorded", func(_ *featureStart, rec *WorktreeRecord) { rec.Branch = "feat/auth-core" },
			func(*featureStart) string { return `records branch "feat/auth-core", not "fix/auth-core"` }, false},
		{"another branch checked out", func(st *featureStart, rec *WorktreeRecord) { st.wt.checkouts[rec.WorktreePath] = "main" },
			func(*featureStart) string { return `which has branch "main" checked out, not "fix/auth-core"` }, false},
		{"a detached HEAD", func(st *featureStart, rec *WorktreeRecord) { st.wt.checkouts[rec.WorktreePath] = "" },
			func(*featureStart) string { return `which has a detached HEAD checked out, not "fix/auth-core"` }, false},
		{"the worktree directory is gone", nil, func(*featureStart) string { return "which is not an existing directory" }, false},
		{"the branch cannot be read", func(st *featureStart, _ *WorktreeRecord) { st.wt.branchErr = boom },
			func(*featureStart) string { return "whose checked-out branch cannot be read (boom)" }, false},
		{"the record cannot be read", func(st *featureStart, _ *WorktreeRecord) { st.wt.lookupErr = boom },
			func(*featureStart) string { return "org: the worktree record org-auth-core cannot be read (boom)" }, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			st := newFeatureStart(t)
			wt := st.worktree("auth-core")
			rec := WorktreeRecord{WorktreePath: wt, Branch: "fix/auth-core", Kind: "org", CanonicalRef: st.ref("auth-core")}
			st.wt.checkouts[wt] = rec.Branch
			if tc.change != nil {
				if err := os.MkdirAll(wt, 0o755); err != nil {
					t.Fatal(err)
				}
				tc.change(st, &rec)
			}
			st.wt.records["org-auth-core"] = rec
			hint := cleanup
			if tc.otherRef {
				hint = otherRefHint
			}
			res := st.o.StartFeature(st.params("auth-core"))
			assertStartRefused(t, res, tc.want(st), hint)
			if tc.otherRef && strings.Contains(res.Spawn.Err.Error(), "--org-id") {
				t.Errorf("expected no --org-id for a record of something else, got %v", res.Spawn.Err)
			}
			if slices.ContainsFunc(st.wt.calls, func(c string) bool { return strings.HasPrefix(c, "ensure:") }) {
				t.Fatalf("expected no ensure, got calls %q", st.wt.calls)
			}
			if after := takeSpawnSnapshot(t, st.o, st.h, st.a); after != (spawnSnapshot{}) {
				t.Fatalf("expected nothing recorded or called, got %+v", after)
			}
		})
	}
	t.Run("the matching record is reused", func(t *testing.T) {
		st := newFeatureStart(t)
		wt := st.worktree("auth-core")
		if err := os.MkdirAll(wt, 0o755); err != nil {
			t.Fatal(err)
		}
		st.wt.records["org-auth-core"] = WorktreeRecord{WorktreePath: wt, Branch: "fix/auth-core", Kind: "org", CanonicalRef: st.ref("auth-core")}
		st.wt.checkouts[wt] = "fix/auth-core"
		if res := st.mustStart(t, st.params("auth-core"), SpawnOutcomeSpawned); res.Worktree != wt {
			t.Fatalf("expected the recorded worktree, got %q", res.Worktree)
		}
		st.wantCalls(t, "lookup:org-auth-core", "branch:"+wt, "ensure:org-auth-core", "lookup:org-auth-core", "branch:"+wt)
	})
	t.Run("a missing ralph-worktree.sh", func(t *testing.T) {
		st := newFeatureStart(t)
		st.wt.lookupErr = fmt.Errorf("org: /x/scripts/ralph-worktree.sh does not exist: start --plan makes the feature worktree with it; %w", errWorktreeScriptMissing)
		res := st.o.StartFeature(st.params("auth-core"))
		assertStartRefused(t, res, "/x/scripts/ralph-worktree.sh does not exist", "ralph init ships it")
		if !errors.Is(res.Spawn.Err, errWorktreeScriptMissing) || strings.Contains(res.Spawn.Err.Error(), "cleanup") {
			t.Fatalf("expected the missing script's error as it is, got %v", res.Spawn.Err)
		}
	})
}

// TestStartFeature_AfterDisband covers the last two cases of plan AC6:
// disband leaves the worktree, so another split plan with the same slug is
// refused by the reuse check (no ensure, nothing recorded), while the same
// split plan feature, approved again with another digest, reuses it and
// binds the org to the new digest (AC1 of plan
// docs/plans/active/2026-10-10-org-feature-worktree-ownership.md). Another
// Type alone gives the other plan's feature another branch but the same
// org_id, so the same worktree record still refuses it, which is why the
// refusal points at another slug.
func TestStartFeature_AfterDisband(t *testing.T) {
	disbanded := func(t *testing.T) *featureStart {
		t.Helper()
		st := newFeatureStart(t)
		st.mustStart(t, st.params("auth-core"), SpawnOutcomeSpawned)
		if d := st.o.Disband(DisbandParams{OrgID: "auth-core"}); len(d.Errs) != 0 || !d.Disbanded {
			t.Fatalf("disband auth-core: %+v", d)
		}
		st.wt.calls = nil
		return st
	}
	wantRecord := func(t *testing.T, st *featureStart) {
		t.Helper()
		want := WorktreeRecord{WorktreePath: st.worktree("auth-core"), Branch: "fix/auth-core", Kind: "org", CanonicalRef: st.ref("auth-core")}
		if len(st.wt.records) != 1 || st.wt.records["org-auth-core"] != want {
			t.Fatalf("worktree records = %+v, want only %+v", st.wt.records, want)
		}
	}

	for _, tc := range []struct {
		name, content, branch string
	}{
		{"another split plan with the same slug", splitPlanFixture, "fix/auth-core"},
		{"another split plan with the same slug and another Type", strings.Replace(splitPlanFixture, "- Type: fix\n", "- Type: feat\n", 1), "feat/auth-core"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			st := disbanded(t)
			p := st.params("auth-core")
			p.SplitPath = st.writePlan(t, "other-split", approveSplitPlan(t, tc.content))
			before := takeSpawnSnapshot(t, st.o, st.h, st.a)
			assertStartRefused(t, st.o.StartFeature(p),
				`records canonical_ref "`+st.ref("auth-core")+`", not "`+splitRef(p.SplitPath, "auth-core")+`"`,
				"which may still be using that worktree and its branch; change this feature's slug in the split plan and approve it again "+
					"(the slug is the default org_id and part of the branch "+tc.branch+")",
				"if that worktree and its branch are no longer needed, remove them with ./scripts/ralph-worktree.sh cleanup --id org-auth-core")
			st.wantCalls(t, "lookup:org-auth-core")
			if after := takeSpawnSnapshot(t, st.o, st.h, st.a); after != before {
				t.Fatalf("expected nothing recorded or called, %+v -> %+v", before, after)
			}
			wantRecord(t, st)
		})
	}
	t.Run("the same feature approved again", func(t *testing.T) {
		st := disbanded(t)
		st.writePlan(t, "auth-split", approveSplitPlan(t, strings.Replace(splitPlanFixture, "Objective: add the token store.", "Objective: add the token store v2.", 1)))
		st.h.workspaceID, st.h.paneID = "ws-2", "pane-2"
		res := st.mustStart(t, st.params("auth-core"), SpawnOutcomeSpawned)
		wt := st.worktree("auth-core")
		st.wantCalls(t, "lookup:org-auth-core", "branch:"+wt, "ensure:org-auth-core", "lookup:org-auth-core", "branch:"+wt)
		if res.Worktree != wt {
			t.Fatalf("expected the same worktree, got %q", res.Worktree)
		}
		wantRecord(t, st)
		assertActiveFeature(t, st.o, "auth-core", &FeatureBinding{
			Split: "auth-split", Feature: "auth-core", Digest: planDigestOf(t, st.plan), Branch: "fix/auth-core", Worktree: wt,
		})
	})
}

// TestStartFeature_EnsureFailureRefused covers plan AC9 at the org layer: an
// ensure failure is refused with the script's message, wrapping the error,
// with nothing recorded and no worktree in the result. The fix that follows
// depends on the failure: a main checkout that is not the clean default
// branch is to be made one; a state record or a directory in the way is
// removed or avoided with another --org-id; a branch in the way, which
// another org or worktree may be using and another --org-id keeps, is
// avoided first by another slug or Type, and renamed or deleted only once
// nothing uses it (AC3b of plan
// docs/plans/active/2026-10-10-org-feature-worktree-ownership.md). The
// .codex/config.toml rewrite message, which says itself how to restore the
// file, and any other failure come back with nothing added.
func TestStartFeature_EnsureFailureRefused(t *testing.T) {
	const prefix = "org: scripts/ralph-worktree.sh ensure: ralph-worktree: "
	for _, tc := range []struct {
		name, script string
		fix          func(st *featureStart) string // "" for nothing added
	}{
		{"main has uncommitted changes", "base branch 'main' has uncommitted changes", func(st *featureStart) string {
			return "make the main worktree " + st.root + " a clean checkout of the default branch and run start again"
		}},
		{"main is on another branch", "must start from clean default branch 'main' (current: feat/x)", func(st *featureStart) string {
			return "make the main worktree " + st.root + " a clean checkout of the default branch and run start again"
		}},
		{"a state record in the way", "state collision for id 'org-auth-core': /repo/.git/ralph/worktrees/org-auth-core.json", func(*featureStart) string {
			return "remove the worktree with ./scripts/ralph-worktree.sh cleanup --id org-auth-core if it is no longer needed, or use another --org-id"
		}},
		{"a directory in the way", "worktree path already exists without matching state: /repo/.claude/worktrees/org-auth-core", func(st *featureStart) string {
			return "remove " + st.worktree("auth-core") + " if it is no longer needed (git worktree remove for a worktree), or use another --org-id"
		}},
		{"a branch in the way", "branch already exists without matching state: fix/auth-core", func(*featureStart) string {
			return branchInTheWayHint("fix/auth-core")
		}},
		{"the .codex/config.toml rewrite", "base branch 'main' has uncommitted changes only in .codex/config.toml, and they look like the known external rewrite " +
			"(see docs/recipes/codex-setup.md).\nIf that is the only change, restore it:\n  git -C '/repo' checkout -- .codex/config.toml", nil},
		{"no jq", "jq not found", nil},
		{"no default branch", "base branch not found: main", nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			st := newFeatureStart(t)
			st.wt.ensureErr = errors.New(prefix + tc.script + " (exit status 1)")
			res := st.o.StartFeature(st.params("auth-core"))
			want := st.wt.ensureErr.Error()
			if tc.fix != nil {
				want += "; " + tc.fix(st)
			}
			assertStartRefused(t, res)
			if res.Spawn.Err.Error() != want {
				t.Errorf("error =\n%s\nwant\n%s", res.Spawn.Err, want)
			}
			if !errors.Is(res.Spawn.Err, st.wt.ensureErr) {
				t.Fatalf("expected the ensure error wrapped, got %v", res.Spawn.Err)
			}
			if after := takeSpawnSnapshot(t, st.o, st.h, st.a); after != (spawnSnapshot{}) {
				t.Fatalf("expected nothing recorded or called, got %+v", after)
			}
		})
	}
}

// branchInTheWayHint is ensureFailureErr's fix for a branch in the way.
func branchInTheWayHint(branch string) string {
	return "another org or worktree may be using the branch " + branch + ", which is the feature's branch whatever the --org-id: " +
		"change this feature's slug (or Type) in the split plan and approve it again; only once you have made sure " +
		"nothing uses that branch, rename it (git branch -m) or, if its commits are not needed, delete it (git branch -D), " +
		"and run start again"
}

// TestEnsureFailureMessages_InWorktreeScript keeps the message parts
// ensureFailureErr reads in step with scripts/ralph-worktree.sh: each is in
// the script's text.
func TestEnsureFailureMessages_InWorktreeScript(t *testing.T) {
	data, err := os.ReadFile(filepath.Join(splitTestRepoRoot(t), filepath.FromSlash(worktreeScriptRel)))
	if err != nil {
		t.Fatal(err)
	}
	for _, msg := range []string{
		ensureNotDefaultBranchMsg, ensureUncommittedMsg, ensureCodexRewriteMsg,
		ensureStateCollisionMsg, ensurePathExistsMsg, ensureBranchExistsMsg,
	} {
		if !strings.Contains(string(data), msg) {
			t.Errorf("%s no longer says %q, which ensureFailureErr reads to choose its hint", worktreeScriptRel, msg)
		}
	}
}

// TestStartFeature_EnsurePrintsAnotherPath: a worktree path from ensure that
// is not the one asked for fails the start before the spawn.
func TestStartFeature_EnsurePrintsAnotherPath(t *testing.T) {
	st := newFeatureStart(t)
	st.wt.ensurePath = filepath.Join(st.root, "elsewhere")
	res := st.o.StartFeature(st.params("auth-core"))
	if res.Spawn.Outcome != SpawnOutcomeFailed || res.Spawn.Err == nil ||
		!strings.Contains(res.Spawn.Err.Error(), `printed the worktree "`+st.wt.ensurePath+`", not `+st.worktree("auth-core")) {
		t.Fatalf("expected a failure naming both paths, got %+v", res.Spawn)
	}
	if res.Worktree != "" {
		t.Fatalf("expected no worktree in the result, got %q", res.Worktree)
	}
	if after := takeSpawnSnapshot(t, st.o, st.h, st.a); after != (spawnSnapshot{}) {
		t.Fatalf("expected nothing recorded or called, got %+v", after)
	}
}

// TestStartFeature_RecordChangedDuringEnsureRefused covers AC3 of plan
// docs/plans/active/2026-10-10-org-feature-worktree-ownership.md: ensure
// returns a recorded worktree with the same path, branch and kind whatever
// its canonical_ref, so when another start makes the record for another
// split plan's feature while this start runs (here inside ensure), start
// reads the record again after ensure and refuses before the spawn: no
// leader (no driver call), nothing in this ledger, no worktree in the
// result, and the other start's record untouched. The message points at
// another slug or at waiting, and at neither the cleanup nor another
// --org-id. A record that cannot be read again, that is gone, or that no
// longer matches otherwise is refused the same way, with the hint to run
// start again.
func TestStartFeature_RecordChangedDuringEnsureRefused(t *testing.T) {
	boom := errors.New("boom")
	const otherRef = "split:/other/org/splits/auth-split.md#auth-core"
	const runAgain = ", so this start does not use that worktree and starts no leader: something changed the worktree " +
		"or its record while this start ran; run start again, which checks the record before it makes or reuses a worktree"
	for _, tc := range []struct {
		name  string
		setup func(t *testing.T, st *featureStart) // runs inside ensure, before it records the worktree
		want  func(st *featureStart) string
		other bool // another start's record: the message has no runAgain, and the record must stay
	}{
		{"another split plan's start made the record", func(t *testing.T, st *featureStart) {
			wt := st.worktree("auth-core")
			if err := os.MkdirAll(wt, 0o755); err != nil {
				t.Error(err)
			}
			st.wt.records["org-auth-core"] = WorktreeRecord{WorktreePath: wt, Branch: "fix/auth-core", Kind: "org", CanonicalRef: otherRef}
			st.wt.checkouts[wt] = "fix/auth-core"
		}, func(st *featureStart) string {
			return `org: another start made the worktree record org-auth-core for "` + otherRef + `" (this start's is "` + st.ref("auth-core") +
				`") while this start ran, so this start does not use that worktree and starts no leader: ` +
				"change this feature's slug in the split plan and approve it again (the slug is the default org_id and part of the branch fix/auth-core), " +
				"or wait until the other org (org_id auth-core) is done and its worktree is removed, and run start again"
		}, true},
		{"the record cannot be read back", func(_ *testing.T, st *featureStart) { st.wt.lookupErr = boom }, func(*featureStart) string {
			return "org: right after scripts/ralph-worktree.sh ensure returned the worktree, its record org-auth-core cannot be read (boom)" + runAgain
		}, false},
		{"the record is gone", func(_ *testing.T, st *featureStart) { st.wt.ensureDrops = true }, func(*featureStart) string {
			return "its record org-auth-core is gone" + runAgain
		}, false},
		{"the branch cannot be read", func(_ *testing.T, st *featureStart) { st.wt.branchErr = boom }, func(st *featureStart) string {
			return "its record org-auth-core names " + st.worktree("auth-core") + ", whose checked-out branch cannot be read (boom)" + runAgain
		}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			st := newFeatureStart(t)
			st.wt.ensureHook = func() { tc.setup(t, st) }
			res := st.o.StartFeature(st.params("auth-core"))
			assertStartRefused(t, res, tc.want(st))
			for _, banned := range []string{"cleanup", "--org-id"} {
				if strings.Contains(res.Spawn.Err.Error(), banned) {
					t.Errorf("expected no %q in the refusal, got %v", banned, res.Spawn.Err)
				}
			}
			if len(st.wt.calls) < 3 || !slices.Equal(st.wt.calls[:3], []string{"lookup:org-auth-core", "ensure:org-auth-core", "lookup:org-auth-core"}) {
				t.Fatalf("expected the record read again after ensure, got calls %q", st.wt.calls)
			}
			if after := takeSpawnSnapshot(t, st.o, st.h, st.a); after != (spawnSnapshot{}) {
				t.Fatalf("expected no leader and nothing recorded, got %+v", after)
			}
			if tc.other && st.wt.records["org-auth-core"].CanonicalRef != otherRef {
				t.Fatalf("expected the other start's record untouched, got %+v", st.wt.records["org-auth-core"])
			}
		})
	}
}

// TestStartFeature_SpawnRefusedAfterEnsure covers the second half of plan
// AC8: when another org reserves an overlapping path between the pre-check
// and the spawn, the leader spawn refuses it under the lock, and the error
// keeps Spawn's refusal and adds that the worktree and the branch stay, that
// the same start reuses them and how to remove them. While that org runs,
// the same start is refused by the pre-check; once it is disbanded, the same
// start reuses the worktree and spawns the leader.
func TestStartFeature_SpawnRefusedAfterEnsure(t *testing.T) {
	st := newFeatureStart(t)
	st.wt.ensureHook = func() {
		if err := st.o.Manifest.Append(reserveEvent("other", "internal/")); err != nil {
			t.Errorf("append the racing reservation: %v", err)
		}
	}
	res := st.o.StartFeature(st.params("auth-core"))
	wt := st.worktree("auth-core")
	rejected := lastEvent(t, st.o)
	hint := "; the worktree " + wt + " and its branch fix/auth-core stay: running the same start again reuses them, " +
		"and ./scripts/ralph-worktree.sh cleanup --id org-auth-core removes them if they are not needed"
	if res.Spawn.Outcome != SpawnOutcomeRejected || rejected.Event != EventRejected || rejected.SeatID != LeaderIdentity ||
		!strings.Contains(rejected.Details, `reservation overlaps the reservation of running org_id "other"`) ||
		res.Spawn.Err == nil || res.Spawn.Err.Error() != rejected.Details+hint {
		t.Fatalf("expected Spawn's refusal %q followed by the hint, got %+v", rejected.Details, res.Spawn)
	}
	if res.Worktree != wt || res.Branch != "fix/auth-core" {
		t.Fatalf("expected the worktree in the result, got %q on %q", res.Worktree, res.Branch)
	}

	st.wt.ensureHook = nil
	assertStartRefused(t, st.o.StartFeature(st.params("auth-core")), `reservation overlaps the reservation of running org_id "other"`)
	if err := st.o.Manifest.Append(ManifestEvent{OrgID: "other", Event: EventDisbanded}); err != nil {
		t.Fatal(err)
	}
	again := st.mustStart(t, st.params("auth-core"), SpawnOutcomeSpawned)
	if again.Worktree != wt || len(st.wt.records) != 1 {
		t.Fatalf("expected the same worktree reused, got %q and records %+v", again.Worktree, st.wt.records)
	}
	st.wantCalls(t, "lookup:org-auth-core", "ensure:org-auth-core", "lookup:org-auth-core", "branch:"+wt,
		"lookup:org-auth-core", "branch:"+wt, "ensure:org-auth-core", "lookup:org-auth-core", "branch:"+wt)
}

// TestStartFeature_SpawnFailureKeepsWorktree: a leader spawn that fails in a
// driver call after the worktree was made wraps that failure with the same
// hint.
func TestStartFeature_SpawnFailureKeepsWorktree(t *testing.T) {
	st := newFeatureStart(t)
	sentinel := errors.New("herdr: tab create refused")
	st.h.tabCreateErr = sentinel
	res := st.o.StartFeature(st.params("auth-core"))
	if res.Spawn.Outcome != SpawnOutcomeFailed || !errors.Is(res.Spawn.Err, sentinel) ||
		!strings.Contains(res.Spawn.Err.Error(), "spawn step tab_create failed") ||
		!strings.Contains(res.Spawn.Err.Error(), "and its branch fix/auth-core stay: running the same start again reuses them") {
		t.Fatalf("expected the spawn failure wrapped with the hint, got %+v", res.Spawn)
	}
	if res.Worktree != st.worktree("auth-core") {
		t.Fatalf("expected the worktree in the result, got %q", res.Worktree)
	}
}

// TestSpawnPrecheckErr_MatchesSpawn keeps spawnPrecheckErr in step with
// Spawn: on the same ledger, the pre-check returns nil exactly when Spawn
// goes on (spawned or idempotent), and otherwise Spawn's own refusal. The
// cases cover each check of Spawn's locked sections and each path into them:
// a new seat, the idempotent return (active, and legacy inactive), and a
// stale in-flight seat, which Spawn compensates before it counts seats.
func TestSpawnPrecheckErr_MatchesSpawn(t *testing.T) {
	f := testFeature("split-1", "auth", "0123456789ab")
	bound := featureLeaderParams("org-a", f, "internal/auth/")
	with := func(p SpawnParams, change func(*SpawnParams)) SpawnParams {
		change(&p)
		return p
	}
	for _, tc := range []struct {
		name  string
		setup func(t *testing.T, o *Org, h *fakeHerdr)
		p     SpawnParams
		want  string // "" when Spawn goes on
	}{
		{"a new org", nil, bound, ""},
		{"max_seats of the org", func(t *testing.T, o *Org, h *fakeHerdr) {
			for i := 1; i <= 3; i++ {
				spawnSeatIn(t, o, h, "org-a", fmt.Sprintf("seat-%d", i), "ws-a", fmt.Sprintf("pane-%d", i))
			}
		}, bound, `max_seats 3 reached for org_id "org-a"`},
		{"max_orgs", func(t *testing.T, o *Org, h *fakeHerdr) {
			o.Config.MaxOrgs = 1
			spawnLeaderIn(t, o, h, "org-b", "ws-b", "pane-b", "web/")
		}, bound, "max_orgs 1 reached"},
		{"max_total_seats", func(t *testing.T, o *Org, h *fakeHerdr) {
			o.Config.MaxTotalSeats = 1
			spawnSeatIn(t, o, h, "org-b", "seat-1", "ws-b", "pane-b")
		}, bound, "max_total_seats 1 reached"},
		{"an overlapping reservation", func(t *testing.T, o *Org, h *fakeHerdr) {
			spawnLeaderIn(t, o, h, "org-b", "ws-b", "pane-b", "internal/")
		}, bound, `reservation overlaps the reservation of running org_id "org-b"`},
		{"a promoted leader's org", func(t *testing.T, o *Org, h *fakeHerdr) {
			spawnSeatIn(t, o, h, "org-a", "seat-1", "ws-a", "pane-1")
		}, bound, "is running without a split plan"},
		{"ralph org start <task>'s org", func(t *testing.T, o *Org, h *fakeHerdr) {
			spawnLeaderIn(t, o, h, "org-a", "ws-a", "pane-1")
		}, bound, "is running without a split plan"},
		{"a plain --reserve org", func(t *testing.T, o *Org, h *fakeHerdr) {
			spawnLeaderIn(t, o, h, "org-a", "ws-a", "pane-1", "internal/auth/")
		}, bound, "was started without a split plan"},
		{"the same binding, leader active", func(t *testing.T, o *Org, h *fakeHerdr) {
			spawnFeatureLeaderIn(t, o, h, "org-a", "ws-a", "pane-1", f, "internal/auth/")
		}, bound, ""},
		{"another digest, leader stopped", func(t *testing.T, o *Org, h *fakeHerdr) {
			spawnFeatureLeaderIn(t, o, h, "org-a", "ws-a", "pane-1", f, "internal/auth/")
			if r := o.Stop(StopParams{OrgID: "org-a", Seat: LeaderIdentity}); r.Err != nil {
				t.Fatalf("stop: %v", r.Err)
			}
		}, featureLeaderParams("org-a", testFeature("split-1", "auth", "ba9876543210"), "internal/auth/"), `org_id "org-a" is bound to`},
		{"a legacy inactive leader past max_orgs", func(t *testing.T, o *Org, h *fakeHerdr) {
			spawnLeaderIn(t, o, h, "org-a", "ws-a", "pane-1")
			if err := o.Manifest.Append(ManifestEvent{OrgID: "org-a", Event: EventDisbanded}); err != nil {
				t.Fatal(err)
			}
			o.Config.MaxOrgs = 1
			spawnLeaderIn(t, o, h, "org-b", "ws-b", "pane-b", "web/")
		}, bound, `max_orgs 1 reached: org_id "org-a" is not running`},
		{"a stale leader is compensated first", func(t *testing.T, o *Org, _ *fakeHerdr) {
			o.Config.MaxTotalSeats = 1
			if err := o.Manifest.Append(ManifestEvent{OrgID: "org-a", SeatID: LeaderIdentity, Event: EventSpawnStarted,
				Role: LeaderIdentity, Driver: "claude", Model: "sonnet"}); err != nil {
				t.Fatal(err)
			}
		}, bound, ""},
		{"a retired role key", func(_ *testing.T, o *Org, _ *fakeHerdr) {
			o.Config.Roles = map[string][]string{oldLeaderName: {"opus"}}
		}, bound, "ralph.toml has retired role key(s)"},
		{"a model outside the pool", nil, with(bound, func(p *SpawnParams) { p.Model = "gpt-9" }), `model "gpt-9" not in [org].model_pool`},
		{"an unverified codex leader", nil, with(bound, func(p *SpawnParams) { p.Driver, p.Model = "codex", "gpt-5-codex" }), "requires [org.permissions].codex_verified=true"},
		{"an autonomous worker without scope", nil, with(mustSpawnParams("org-a", "seat-1"), func(p *SpawnParams) { p.Scope = "" }),
			"autonomous permission mode requires --scope or --reserve"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			o, h, _ := testOrg(t)
			if tc.setup != nil {
				tc.setup(t, o, h)
			}
			checked, err := checkSpawnInput(tc.p)
			if err != nil {
				t.Fatalf("checkSpawnInput: %v", err)
			}
			pre := spawnPrecheckErr(o.Config, checked, mustReadEvents(t, o))
			r := o.Spawn(tc.p)
			if tc.want == "" {
				if pre != nil || (r.Outcome != SpawnOutcomeSpawned && r.Outcome != SpawnOutcomeIdempotent) {
					t.Fatalf("expected both to go on, pre-check %v, Spawn %+v", pre, r)
				}
				return
			}
			if pre == nil || !strings.Contains(pre.Error(), tc.want) {
				t.Fatalf("expected the pre-check to refuse with %q, got %v", tc.want, pre)
			}
			if r.Outcome != SpawnOutcomeRejected || r.Err == nil || r.Err.Error() != pre.Error() {
				t.Fatalf("expected Spawn to refuse with the pre-check's error %q, got %+v", pre, r)
			}
		})
	}
}

// --- the script-backed FeatureWorktrees (scriptFeatureWorktrees) ---

// worktreeScriptRepo makes a git repo on main whose one commit holds the
// repo's scripts/ralph-worktree.sh and scripts/ralph-common.sh and a
// .gitignore for .claude/worktrees/, and returns its root (symlinks
// resolved). It skips the test when bash, git or jq is missing.
func worktreeScriptRepo(t *testing.T) string {
	t.Helper()
	for _, tool := range []string{"bash", "git", "jq"} {
		if _, err := exec.LookPath(tool); err != nil {
			t.Skipf("%s is needed to run scripts/ralph-worktree.sh: %v", tool, err)
		}
	}
	isolateGitEnv(t)
	root := resolvedOrClean(t.TempDir())
	runGit(t, "init", "--quiet", "-b", "main", root)
	if err := os.MkdirAll(filepath.Join(root, "scripts"), 0o755); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"ralph-worktree.sh", "ralph-common.sh"} {
		data, err := os.ReadFile(filepath.Join(splitTestRepoRoot(t), "scripts", name))
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(root, "scripts", name), data, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(root, ".gitignore"), []byte(".claude/worktrees/\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	runGit(t, "-C", root, "add", ".")
	runGit(t, "-C", root, "-c", "user.name=t", "-c", "user.email=t@example.com", "-c", "commit.gpgsign=false",
		"commit", "--quiet", "-m", "init")
	return root
}

// TestStartFeature_RealWorktreeScript runs StartFeature with Org.Worktrees
// nil, so through scriptFeatureWorktrees and the real
// scripts/ralph-worktree.sh (one test, since each script run costs a second
// or more on macOS): the first start makes the worktree and the branch from
// main, records canonical_ref, and spawns the leader in it; the same start
// again passes the reuse check on the real record and checkout and records
// nothing; with main dirty, a start for another feature is refused with the
// script's message and records nothing; with main clean again but that
// feature's branch already made by hand, it is refused with the script's
// message and the branch fix, which leads with another slug, and makes no
// worktree. A repo without the script says so.
func TestStartFeature_RealWorktreeScript(t *testing.T) {
	bare := t.TempDir()
	if _, _, err := (scriptFeatureWorktrees{}).Lookup(bare, "org-a"); !errors.Is(err, errWorktreeScriptMissing) ||
		!strings.Contains(err.Error(), filepath.Join(bare, "scripts", "ralph-worktree.sh")+" does not exist") {
		t.Fatalf("expected the missing script named, got %v", err)
	}

	root := worktreeScriptRepo(t)
	st := newFeatureStart(t)
	st.o.Worktrees = nil
	st.root = root

	res := st.mustStart(t, st.params("auth-core"), SpawnOutcomeSpawned)
	wt := st.worktree("auth-core")
	if res.Worktree != wt {
		t.Fatalf("expected the worktree %s, got %q", wt, res.Worktree)
	}
	rec, ok, err := (scriptFeatureWorktrees{}).Lookup(root, "org-auth-core")
	if want := (WorktreeRecord{WorktreePath: wt, Branch: "fix/auth-core", Kind: "org", CanonicalRef: st.ref("auth-core")}); err != nil || !ok || rec != want {
		t.Fatalf("worktree record = %+v (found %v, err %v), want %+v", rec, ok, err, want)
	}

	before := takeSpawnSnapshot(t, st.o, st.h, st.a)
	st.mustStart(t, st.params("auth-core"), SpawnOutcomeIdempotent)
	if after := takeSpawnSnapshot(t, st.o, st.h, st.a); after != before {
		t.Fatalf("expected the same start to record nothing, %+v -> %+v", before, after)
	}

	untracked := filepath.Join(root, "untracked.txt")
	if err := os.WriteFile(untracked, []byte("x\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	assertStartRefused(t, st.o.StartFeature(st.params("auth-docs")),
		"org: scripts/ralph-worktree.sh ensure: ralph-worktree: base branch 'main' has uncommitted changes",
		"; make the main worktree "+root+" a clean checkout of the default branch and run start again")
	if after := takeSpawnSnapshot(t, st.o, st.h, st.a); after != before {
		t.Fatalf("expected nothing recorded or called, %+v -> %+v", before, after)
	}

	if err := os.Remove(untracked); err != nil {
		t.Fatal(err)
	}
	runGit(t, "-C", root, "branch", "feat/auth-docs")
	res = st.o.StartFeature(st.params("auth-docs"))
	assertStartRefused(t, res,
		"org: scripts/ralph-worktree.sh ensure: ralph-worktree: branch already exists without matching state: feat/auth-docs",
		"; "+branchInTheWayHint("feat/auth-docs"))
	if msg := res.Spawn.Err.Error(); strings.Index(msg, "change this feature's slug (or Type)") > strings.Index(msg, "git branch -m") {
		t.Errorf("expected the slug change before renaming the branch, got %v", res.Spawn.Err)
	}
	if strings.Contains(res.Spawn.Err.Error(), "clean checkout of the default branch") {
		t.Errorf("expected no clean-checkout hint for a branch in the way, got %v", res.Spawn.Err)
	}
	if _, err := os.Stat(st.worktree("auth-docs")); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("expected no worktree at %s, got %v", st.worktree("auth-docs"), err)
	}
	if after := takeSpawnSnapshot(t, st.o, st.h, st.a); after != before {
		t.Fatalf("expected nothing recorded or called, %+v -> %+v", before, after)
	}
}

// TestStartFeature_RealWorktreeScript_CheckoutsMoved runs the two checkouts
// start --plan reads through the real scripts/ralph-worktree.sh and git: a
// feature worktree whose checkout moved to another branch fails the reuse
// check on the branch git reports, with nothing recorded, and is reused again
// once it is back on the feature's branch; a directory left at a feature's
// worktree path without a record is refused by ensure with the script's
// message and the hint to remove it; a main checkout on a branch other than
// the default is refused by ensure with the script's message and the
// clean-checkout hint, and makes no worktree for the feature.
func TestStartFeature_RealWorktreeScript_CheckoutsMoved(t *testing.T) {
	root := worktreeScriptRepo(t)
	st := newFeatureStart(t)
	st.o.Worktrees = nil
	st.root = root
	st.mustStart(t, st.params("auth-core"), SpawnOutcomeSpawned)
	wt := st.worktree("auth-core")
	before := takeSpawnSnapshot(t, st.o, st.h, st.a)

	runGit(t, "-C", wt, "checkout", "--quiet", "-b", "moved")
	assertStartRefused(t, st.o.StartFeature(st.params("auth-core")),
		`which has branch "moved" checked out, not "fix/auth-core"`, "cleanup --id org-auth-core")
	if after := takeSpawnSnapshot(t, st.o, st.h, st.a); after != before {
		t.Fatalf("expected nothing recorded or called, %+v -> %+v", before, after)
	}
	runGit(t, "-C", wt, "checkout", "--quiet", "fix/auth-core")
	if res := st.mustStart(t, st.params("auth-core"), SpawnOutcomeIdempotent); res.Worktree != wt {
		t.Fatalf("expected the worktree %s reused, got %q", wt, res.Worktree)
	}

	left := st.worktree("auth-docs")
	if err := os.MkdirAll(left, 0o755); err != nil {
		t.Fatal(err)
	}
	assertStartRefused(t, st.o.StartFeature(st.params("auth-docs")),
		"org: scripts/ralph-worktree.sh ensure: ralph-worktree: worktree path already exists without matching state: "+left,
		"; remove "+left+" if it is no longer needed (git worktree remove for a worktree), or use another --org-id")
	if after := takeSpawnSnapshot(t, st.o, st.h, st.a); after != before {
		t.Fatalf("expected nothing recorded or called, %+v -> %+v", before, after)
	}
	if err := os.Remove(left); err != nil {
		t.Fatal(err)
	}

	runGit(t, "-C", root, "checkout", "--quiet", "-b", "side")
	assertStartRefused(t, st.o.StartFeature(st.params("auth-docs")),
		"org: scripts/ralph-worktree.sh ensure: ralph-worktree: must start from clean default branch 'main' (current: side)",
		"; make the main worktree "+root+" a clean checkout of the default branch and run start again")
	if _, err := os.Stat(st.worktree("auth-docs")); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("expected no worktree at %s, got %v", st.worktree("auth-docs"), err)
	}
	if after := takeSpawnSnapshot(t, st.o, st.h, st.a); after != before {
		t.Fatalf("expected nothing recorded or called, %+v -> %+v", before, after)
	}
}

// spyWorktrees is scriptFeatureWorktrees with its calls recorded the way
// fakeWorktrees records them.
type spyWorktrees struct {
	scriptFeatureWorktrees
	calls []string
}

func (s *spyWorktrees) Lookup(repoRoot, id string) (WorktreeRecord, bool, error) {
	s.calls = append(s.calls, "lookup:"+id)
	return s.scriptFeatureWorktrees.Lookup(repoRoot, id)
}

func (s *spyWorktrees) Ensure(repoRoot string, req EnsureWorktree) (string, error) {
	s.calls = append(s.calls, "ensure:"+req.ID)
	return s.scriptFeatureWorktrees.Ensure(repoRoot, req)
}

func (s *spyWorktrees) CurrentBranch(path string) (string, error) {
	s.calls = append(s.calls, "branch:"+path)
	return s.scriptFeatureWorktrees.CurrentBranch(path)
}

// featureWorktreeState is what the org of a feature worktree owns, read
// through the real script and git: the ralph-worktree.sh record id as its
// file holds it, the commit of branch, and the branch and the commit checked
// out in wt.
func featureWorktreeState(t *testing.T, root, id, wt, branch string) string {
	t.Helper()
	file, err := runWorktreeScript(root, "state-path", id)
	if err != nil {
		t.Fatal(err)
	}
	record, err := os.ReadFile(strings.TrimSpace(file))
	if err != nil {
		t.Fatal(err)
	}
	git := func(args ...string) string {
		out, err := exec.Command("git", args...).Output()
		if err != nil {
			t.Fatalf("git %v: %v", args, err)
		}
		return strings.TrimSpace(string(out))
	}
	return strings.Join([]string{string(record), git("-C", root, "rev-parse", "refs/heads/"+branch),
		git("-C", wt, "branch", "--show-current"), git("-C", wt, "rev-parse", "HEAD")}, "\n")
}

// TestStartFeature_RealWorktreeScript_OtherLedger covers AC2 and AC3b of plan
// docs/plans/active/2026-10-10-org-feature-worktree-ownership.md through the
// real scripts/ralph-worktree.sh: two ledgers of one repo each hold a split
// plan auth-split with the feature auth-core (the same slug and Type). After
// ledger A's org is started and disbanded, its worktree stays, and ledger
// B's start of the same feature is refused by the reuse check on the
// canonical_ref, which holds each plan's path: only the record is looked up
// (no ensure), nothing is recorded in ledger B or called, and the refusal
// leads with another slug. Following it, ledger B's plan with the feature
// renamed and approved again starts in a worktree and on a branch of its
// own; the same start again reads that canonical_ref back, with the space in
// ledger B's path, and reuses the worktree; and ledger A's worktree, branch
// and record are as they were.
func TestStartFeature_RealWorktreeScript_OtherLedger(t *testing.T) {
	root := worktreeScriptRepo(t)
	a := newFeatureStart(t)
	a.o.Worktrees, a.root = nil, root
	a.mustStart(t, a.params("auth-core"), SpawnOutcomeSpawned)
	if d := a.o.Disband(DisbandParams{OrgID: "auth-core"}); len(d.Errs) != 0 || !d.Disbanded {
		t.Fatalf("disband ledger A's auth-core: %+v", d)
	}
	wtA := a.worktree("auth-core")
	ownedByA := featureWorktreeState(t, root, "org-auth-core", wtA, "fix/auth-core")

	b := newFeatureStartIn(t, filepath.Join(t.TempDir(), "ledger b", "org"))
	spy := &spyWorktrees{}
	b.o.Worktrees, b.root = spy, root
	res := b.o.StartFeature(b.params("auth-core"))
	assertStartRefused(t, res,
		`org: the worktree record org-auth-core records canonical_ref "`+a.ref("auth-core")+`", not "`+b.ref("auth-core")+`", so start does not reuse it: `,
		"which may still be using that worktree and its branch; change this feature's slug in the split plan and approve it again",
		"or, if that worktree and its branch are no longer needed, remove them with ./scripts/ralph-worktree.sh cleanup --id org-auth-core")
	if msg := res.Spawn.Err.Error(); strings.Index(msg, "change this feature's slug") > strings.Index(msg, "cleanup") || strings.Contains(msg, "--org-id") {
		t.Errorf("expected the slug change first and no --org-id, got %v", res.Spawn.Err)
	}
	if !slices.Equal(spy.calls, []string{"lookup:org-auth-core"}) {
		t.Fatalf("expected only the record looked up, no ensure, got calls %q", spy.calls)
	}
	if after := takeSpawnSnapshot(t, b.o, b.h, b.a); after != (spawnSnapshot{}) {
		t.Fatalf("expected nothing recorded in ledger B or called, got %+v", after)
	}

	b.writePlan(t, "auth-split", approveSplitPlan(t, strings.ReplaceAll(splitPlanFixture, "auth-core", "auth-store")))
	started := b.mustStart(t, b.params("auth-store"), SpawnOutcomeSpawned)
	wtB := b.worktree("auth-store")
	if started.Worktree != wtB || started.Branch != "fix/auth-store" {
		t.Fatalf("expected ledger B's own worktree %s on fix/auth-store, got %q on %q", wtB, started.Worktree, started.Branch)
	}
	rec, ok, err := (scriptFeatureWorktrees{}).Lookup(root, "org-auth-store")
	want := WorktreeRecord{WorktreePath: wtB, Branch: "fix/auth-store", Kind: "org", CanonicalRef: b.ref("auth-store")}
	if err != nil || !ok || rec != want || !strings.Contains(rec.CanonicalRef, "/ledger b/org/splits/auth-split.md#auth-store") {
		t.Fatalf("worktree record = %+v (found %v, err %v), want %+v", rec, ok, err, want)
	}
	b.mustStart(t, b.params("auth-store"), SpawnOutcomeIdempotent)
	if after := featureWorktreeState(t, root, "org-auth-core", wtA, "fix/auth-core"); after != ownedByA {
		t.Fatalf("expected ledger A's worktree, branch and record unchanged:\n%s\n->\n%s", ownedByA, after)
	}
}
