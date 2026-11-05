package cli

import (
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/yoshpy-dev/ralph/internal/org"
)

// `ralph org spawn`, `ralph org start <task>`, and `ralph org start --plan`
// re-run for a seat already spawned (plan 2026-11-05-org-role-models, D4 and
// D9; cross-review cycle 1): org.ResolveRoleModel only decides the driver
// and the model, so the re-run reaches Spawn, whose idempotent return comes
// before the driver_pool and model-name checks, and the seat is returned
// with exit 0 although driver_pool was narrowed since. A new seat with the
// same request is refused by Spawn and recorded as rejected, like any
// envelope refusal; start --plan refuses a new leader before it makes
// anything, with nothing recorded.

// retryCodexPoolToml has codex in driver_pool, for the first spawn;
// retryClaudePoolToml has only claude, for the re-runs after it. Both run the
// seats guarded, so a codex seat needs no codex_verified.
const (
	retryCodexPoolToml  = "[org]\ndriver_pool = [\"claude\", \"codex\"]\n\n[org.permissions]\ndefault = \"guarded\"\n"
	retryClaudePoolToml = "[org]\ndriver_pool = [\"claude\"]\n\n[org.permissions]\ndefault = \"guarded\"\n"
)

// retryCodexFlags are the explicit flags of the first spawn and of the
// re-runs that repeat it.
var retryCodexFlags = []string{"--driver", "codex", "--model", "gpt-6.1-sol"}

// orgRetrySnapshot is what a re-run must leave as it was: the number of
// manifest events and receipts in the ledger, and of herdr and agmsg calls.
type orgRetrySnapshot struct{ events, receipts, herdr, agmsg int }

func takeOrgRetrySnapshot(t *testing.T, stateDir, herdrLog, agmsgLog string) orgRetrySnapshot {
	t.Helper()
	rr, err := org.NewReceiptStoreAtPath(org.ReceiptsPathIn(stateDir)).Read()
	if err != nil {
		t.Fatalf("read receipts: %v", err)
	}
	return orgRetrySnapshot{
		events:   len(readManifestEvents(t, org.ManifestPathIn(stateDir))),
		receipts: len(rr.Receipts),
		herdr:    len(readLogLines(t, herdrLog)),
		agmsg:    len(readLogLines(t, agmsgLog)),
	}
}

// retryCase is one re-run of a seat already spawned: flags the run passes
// on top of the verb's own.
type retryCase struct {
	name  string
	flags []string
}

// retryCases are the re-runs both tests make with driver_pool narrowed to
// claude and the role's entry on codex: the same explicit flags, no flags
// (the driver from the entry), and a --model that is not one word.
var retryCases = []retryCase{
	{"the same --driver codex --model", retryCodexFlags},
	{"no flags, the role's codex entry", nil},
	{"a --model that is not one word", []string{"--driver", "codex", "--model", "not a real model"}},
}

// refusalCase is one new seat (or org) with a request the narrowed config
// refuses, and the text the refusal must contain.
type refusalCase struct {
	name  string
	flags []string
	want  string
}

// refusalCases are the new seats both tests start with the narrowed config.
// role is the seat's role, which the entry's refusal names.
func refusalCases(role string) []refusalCase {
	return []refusalCase{
		{"--driver codex --model", retryCodexFlags, `org: driver "codex" not in [org].driver_pool [claude]`},
		{"no flags, the role's codex entry", nil,
			`org: role "` + role + `" runs on driver "codex" in [org.role_models].` + role + `, which is not in [org].driver_pool [claude]: ` +
				"add codex to driver_pool, change the entry, or pass both --driver and --model"},
		{"a --model that is not one word", []string{"--driver", "claude", "--model", "not a real model"}, `org: model "not a real model" must be ""`},
	}
}

// TestOrgSpawn_RoleModels_RetryAfterDriverPoolNarrowed_ReturnsTheSeat: a
// reviewer seat spawned with --driver codex --model, then re-run after
// driver_pool was narrowed to claude (with the reviewer's entry on codex),
// returns the seat with exit 0 and the already-spawned line, writing no
// manifest event or receipt and calling no herdr or agmsg, with the same
// flags, with no flags, and with a --model that is not one word. A new seat
// with each request is refused by Spawn: recorded as rejected, in the plain
// driver_pool wording for --driver codex, naming the entry for no flags,
// and naming the model for a model that is not one word.
func TestOrgSpawn_RoleModels_RetryAfterDriverPoolNarrowed_ReturnsTheSeat(t *testing.T) {
	herdrLog, agmsgLog := setupOrgStubPATH(t)
	t.Chdir(t.TempDir())
	stateDir := filepath.Join(t.TempDir(), "state")
	cwd := t.TempDir()
	spawn := func(configPath, seat string, flags []string) (string, error) {
		args := []string{"spawn", "--org-id", "org-a", "--id", seat, "--role", "reviewer", "--cwd", cwd,
			"--scope", "test-scope", "--state-dir", stateDir, "--config", configPath}
		return runOrgCmd(t, append(args, flags...)...)
	}
	if out, err := spawn(roleModelsConfig(t, retryCodexPoolToml), "seat-1", retryCodexFlags); err != nil {
		t.Fatalf("first spawn: %v (output: %s)", err, out)
	}
	narrowed := roleModelsConfig(t, retryClaudePoolToml+"\n[org.role_models]\nreviewer = { driver = \"codex\", model = \"gpt-6.1-sol\" }\n")
	before := takeOrgRetrySnapshot(t, stateDir, herdrLog, agmsgLog)

	for _, tc := range retryCases {
		t.Run("re-run: "+tc.name, func(t *testing.T) {
			out, err := spawn(narrowed, "seat-1", tc.flags)
			if err != nil {
				t.Fatalf("re-run %v: %v (output: %s)", tc.flags, err, out)
			}
			if want := `seat "seat-1" already spawned (org_id=org-a driver=codex model=gpt-6.1-sol pane_id=`; !strings.Contains(out, want) {
				t.Errorf("output = %q; want it to contain %q", out, want)
			}
			if after := takeOrgRetrySnapshot(t, stateDir, herdrLog, agmsgLog); after != before {
				t.Errorf("after the re-run %+v; want it as before %+v", after, before)
			}
		})
	}

	for i, tc := range refusalCases("reviewer") {
		t.Run("new seat: "+tc.name, func(t *testing.T) {
			seat := "seat-new-" + itoa(i)
			out, err := spawn(narrowed, seat, tc.flags)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("spawn %v = %v; want refused with %q (output: %s)", tc.flags, err, tc.want, out)
			}
			if last := lastSeatEvent(t, stateDir, "org-a", seat); last.Event != org.EventRejected || last.Details != err.Error() {
				t.Errorf("last event of %s = %+v; want rejected carrying the error", seat, last)
			}
			if lines := readLogLines(t, herdrLog); len(lines) != before.herdr {
				t.Errorf("herdr calls %d -> %d; want none for a refusal", before.herdr, len(lines))
			}
		})
	}
}

// TestOrgStart_RoleModels_LeaderRetryAfterDriverPoolNarrowed_ReturnsTheLeader
// is the same for `ralph org start <task>`: a leader started on codex, then
// started again for the same org after driver_pool was narrowed to claude
// (with the leader's entry on codex), returns the leader with exit 0 and
// writes and calls nothing, with the same flags, with no flags, and with a
// --model that is not one word; a start for a new org with each request is
// refused by Spawn and recorded as rejected.
func TestOrgStart_RoleModels_LeaderRetryAfterDriverPoolNarrowed_ReturnsTheLeader(t *testing.T) {
	herdrLog, agmsgLog := setupOrgStubPATH(t)
	t.Chdir(t.TempDir())
	stateDir := filepath.Join(t.TempDir(), "state")
	cwd := t.TempDir()
	start := func(configPath, orgID string, flags []string) (string, error) {
		args := append([]string{"start", "--org-id", orgID, "--cwd", cwd, "--scope", orgID,
			"--state-dir", stateDir, "--config", configPath}, flags...)
		return runOrgCmd(t, append(args, "task text")...)
	}
	if out, err := start(roleModelsConfig(t, retryCodexPoolToml), "org-a", retryCodexFlags); err != nil {
		t.Fatalf("first start: %v (output: %s)", err, out)
	}
	narrowed := roleModelsConfig(t, retryClaudePoolToml+"\n[org.role_models]\nleader = { driver = \"codex\", model = \"gpt-6.1-sol\" }\n")
	before := takeOrgRetrySnapshot(t, stateDir, herdrLog, agmsgLog)

	for _, tc := range retryCases {
		t.Run("re-run: "+tc.name, func(t *testing.T) {
			out, err := start(narrowed, "org-a", tc.flags)
			if err != nil {
				t.Fatalf("start again %v: %v (output: %s)", tc.flags, err, out)
			}
			if want := `seat "leader" already spawned (org_id=org-a driver=codex model=gpt-6.1-sol pane_id=`; !strings.Contains(out, want) {
				t.Errorf("output = %q; want it to contain %q", out, want)
			}
			if after := takeOrgRetrySnapshot(t, stateDir, herdrLog, agmsgLog); after != before {
				t.Errorf("after starting again %+v; want it as before %+v", after, before)
			}
		})
	}

	for i, tc := range refusalCases(org.LeaderIdentity) {
		t.Run("new org: "+tc.name, func(t *testing.T) {
			orgID := "org-new-" + itoa(i)
			out, err := start(narrowed, orgID, tc.flags)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("start %v = %v; want refused with %q (output: %s)", tc.flags, err, tc.want, out)
			}
			if last := lastSeatEvent(t, stateDir, orgID, org.LeaderIdentity); last.Event != org.EventRejected || last.Details != err.Error() {
				t.Errorf("last event of %s's leader = %+v; want rejected carrying the error", orgID, last)
			}
			if lines := readLogLines(t, herdrLog); len(lines) != before.herdr {
				t.Errorf("herdr calls %d -> %d; want none for a refusal", before.herdr, len(lines))
			}
		})
	}
}

// TestOrgStartPlan_RoleModels_LeaderEntryOutsideDriverPool is the same for
// `ralph org start --plan`, whose leader spawn org.StartFeature builds from
// StartFeatureParams, so DriverFromRoleEntry reaches it too: with the
// leader's entry on codex and driver_pool = ["claude"], start --plan with no
// --driver or --model is refused naming the role, its entry, and
// driver_pool, before anything is made (no feature worktree, branch, or
// ralph-worktree.sh record, no manifest event, no herdr or agmsg call). A
// leader started for the feature while codex was in driver_pool is returned
// by the same start --plan after driver_pool was narrowed, with no flags and
// with --driver codex --model, and nothing is recorded or called.
func TestOrgStartPlan_RoleModels_LeaderEntryOutsideDriverPool(t *testing.T) {
	const leaderOnCodex = "\n[org.permissions]\ndefault = \"guarded\"\n\n[org.role_models]\nleader = { driver = \"codex\", model = \"gpt-6.1-sol\" }\n"
	herdrLog, agmsgLog := setupOrgStubPATH(t)
	root, sub := newFeatureRepoSub(t, ".claude/worktrees/\n.harness/\n")
	stateDir := filepath.Join(root, ".harness", "state", "org")
	plan := writeFeatureSplitPlan(t, stateDir, approvedFeatureSplitPlan(t))
	// Both files are under .harness/, which the repository ignores, so the
	// main checkout stays clean for scripts/ralph-worktree.sh.
	wide := filepath.Join(root, ".harness", "wide.toml")
	writeTestFile(t, wide, "[org]\ndriver_pool = [\"claude\", \"codex\"]\n"+leaderOnCodex)
	narrowed := filepath.Join(root, ".harness", "narrowed.toml")
	writeTestFile(t, narrowed, "[org]\ndriver_pool = [\"claude\"]\n"+leaderOnCodex)
	t.Chdir(sub)
	startPlan := func(configPath string, flags []string) (stdout, stderr string, err error) {
		return runOrgCmdStreams(t, append([]string{"start", "--plan", plan, "--feature", "a", "--config", configPath}, flags...)...)
	}

	stdout, _, err := startPlan(narrowed, nil)
	want := `org: role "leader" runs on driver "codex" in [org.role_models].leader, which is not in [org].driver_pool [claude]: ` +
		"add codex to driver_pool, change the entry, or pass both --driver and --model"
	if err == nil || err.Error() != want {
		t.Fatalf("start --plan = %v; want refused with %q (stdout: %s)", err, want, stdout)
	}
	assertNoFeatureStart(t, root, "a")
	if events := readManifestEvents(t, org.ManifestPathIn(stateDir)); len(events) != 0 {
		t.Errorf("manifest = %v; want nothing recorded", eventTypes(events))
	}
	if lines := slices.Concat(readLogLines(t, herdrLog), readLogLines(t, agmsgLog)); len(lines) != 0 {
		t.Errorf("herdr and agmsg calls = %v; want none", lines)
	}

	stdout, stderr, err := startPlan(wide, nil)
	if err != nil {
		t.Fatalf("start --plan with codex in driver_pool: %v (stdout: %s stderr: %s)", err, stdout, stderr)
	}
	if want := `spawned seat "leader" (org_id=a driver=codex model=gpt-6.1-sol `; !strings.Contains(stdout, want) {
		t.Fatalf("stdout = %q; want it to contain %q", stdout, want)
	}
	before := takeOrgRetrySnapshot(t, stateDir, herdrLog, agmsgLog)
	for _, tc := range []retryCase{{"no flags, the leader's codex entry", nil}, {"the same --driver codex --model", retryCodexFlags}} {
		t.Run("re-run: "+tc.name, func(t *testing.T) {
			stdout, stderr, err := startPlan(narrowed, tc.flags)
			if err != nil {
				t.Fatalf("start --plan again %v: %v (stdout: %s stderr: %s)", tc.flags, err, stdout, stderr)
			}
			if want := `seat "leader" already spawned (org_id=a driver=codex model=gpt-6.1-sol pane_id=`; !strings.Contains(stdout, want) {
				t.Errorf("stdout = %q; want it to contain %q", stdout, want)
			}
			if after := takeOrgRetrySnapshot(t, stateDir, herdrLog, agmsgLog); after != before {
				t.Errorf("after starting again %+v; want it as before %+v", after, before)
			}
		})
	}
}
