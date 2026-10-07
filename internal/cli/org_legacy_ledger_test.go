package cli

import (
	"bytes"
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/yoshpy-dev/ralph/internal/org"
)

// Tests for guardLegacyOrgStateDir (plan 2026-10-07-org-state-dir-common,
// AC5-AC7). Every test chdirs into a linked worktree of a temp repository,
// so none of them runs in parallel.

// legacyNoteMarker is a phrase only guardLegacyOrgStateDir's note prints.
const legacyNoteMarker = "older org ledger"

// runRootCmdSplit runs `ralph <args...>` in-process with stdout and stderr
// captured separately (cmd.OutOrStdout / cmd.ErrOrStderr).
func runRootCmdSplit(t *testing.T, args ...string) (stdout, stderr string, err error) {
	t.Helper()
	root := NewRootCmd()
	var outBuf, errBuf bytes.Buffer
	root.SetOut(&outBuf)
	root.SetErr(&errBuf)
	root.SetArgs(args)
	err = root.Execute()
	return outBuf.String(), errBuf.String(), err
}

// legacyLedgerRepo is a temp repository with one commit and one linked
// worktree. All paths are symlink-resolved, matching what git prints.
type legacyLedgerRepo struct {
	mainRoot  string
	worktree  string
	sharedDir string // <mainRoot>/.harness/state/org, what the resolver returns
	legacyDir string // <worktree>/.harness/state/org, the per-worktree ledger
}

// newLegacyLedgerRepo builds a legacyLedgerRepo and chdirs into its
// worktree. It isolates git from the developer's config and from GIT_DIR-style
// variables a git hook may export, and clears RALPH_ORG_STATE_DIR so the
// git tiers are the ones under test.
func newLegacyLedgerRepo(t *testing.T) legacyLedgerRepo {
	t.Helper()
	isolateGitConfig(t)
	for _, key := range []string{"GIT_DIR", "GIT_WORK_TREE", "GIT_COMMON_DIR", "GIT_INDEX_FILE"} {
		t.Setenv(key, "")
		if err := os.Unsetenv(key); err != nil {
			t.Fatalf("unsetenv %s: %v", key, err)
		}
	}
	t.Setenv(org.EnvOrgStateDir, "")

	tmp, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatalf("eval symlinks: %v", err)
	}
	r := legacyLedgerRepo{mainRoot: filepath.Join(tmp, "main"), worktree: filepath.Join(tmp, "wt")}
	r.sharedDir = filepath.Join(r.mainRoot, ".harness", "state", "org")
	r.legacyDir = filepath.Join(r.worktree, ".harness", "state", "org")

	runLedgerTestGit(t, "init", "--quiet", r.mainRoot)
	runLedgerTestGit(t, "-C", r.mainRoot, "-c", "user.name=t", "-c", "user.email=t@example.com", "-c", "commit.gpgsign=false",
		"commit", "--quiet", "--allow-empty", "-m", "init")
	runLedgerTestGit(t, "-C", r.mainRoot, "worktree", "add", "--quiet", r.worktree, "-b", "wt")
	t.Chdir(r.worktree)
	return r
}

func runLedgerTestGit(t *testing.T, args ...string) {
	t.Helper()
	if out, err := exec.Command("git", args...).CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
}

// spawnLedgerSeat spawns seat into org-a in the ledger at stateDir through
// the stub herdr/agmsg (setupOrgStubPATH). --state-dir picks the ledger, so
// the guard stays out of the way.
func spawnLedgerSeat(t *testing.T, stateDir, seat string) {
	t.Helper()
	out, err := runOrgCmd(t,
		"spawn", "--org-id", "org-a", "--id", seat, "--role", "worker",
		"--driver", "claude", "--model", "sonnet", "--cwd", t.TempDir(),
		"--scope", "test-scope", "--state-dir", stateDir,
	)
	if err != nil {
		t.Fatalf("seed spawn %s into %s: %v (output: %s)", seat, stateDir, err, out)
	}
}

func readFileBytes(t *testing.T, path string) []byte {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return data
}

func TestOrgLegacyLedger_MutatingVerbsRefusedWhileLegacySeatsActive(t *testing.T) {
	cases := []struct {
		name string
		args func(seatCwd string) []string
	}{
		{"spawn", func(seatCwd string) []string {
			return []string{"spawn", "--org-id", "org-a", "--id", "seat-2", "--role", "worker",
				"--driver", "claude", "--model", "sonnet", "--cwd", seatCwd, "--scope", "test-scope"}
		}},
		{"spawn_dry_run", func(seatCwd string) []string {
			return []string{"spawn", "--org-id", "org-a", "--id", "seat-2", "--role", "worker",
				"--driver", "claude", "--model", "sonnet", "--cwd", seatCwd, "--scope", "test-scope", "--dry-run"}
		}},
		{"start", func(seatCwd string) []string {
			return []string{"start", "--org-id", "org-a", "--cwd", seatCwd, "do the task"}
		}},
		{"send", func(string) []string {
			return []string{"send", "--org-id", "org-a", "--to", "seat-1", "--text", "TYPE: HEARTBEAT"}
		}},
		{"stop", func(string) []string { return []string{"stop", "--org-id", "org-a", "--seat", "seat-1"} }},
		{"stop_all", func(string) []string { return []string{"stop", "--all"} }},
		{"stop_all_dry_run", func(string) []string { return []string{"stop", "--all", "--dry-run"} }},
		{"disband", func(string) []string { return []string{"disband", "--org-id", "org-a"} }},
		{"disband_all", func(string) []string { return []string{"disband", "--all"} }},
		{"watch", func(string) []string { return []string{"watch", "--org-id", "org-a", "--once"} }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			herdrLog, agmsgLog := setupOrgStubPATH(t)
			r := newLegacyLedgerRepo(t)
			spawnLedgerSeat(t, r.legacyDir, "seat-1")
			legacyManifest := org.ManifestPathIn(r.legacyDir)
			before := readFileBytes(t, legacyManifest)
			herdrBefore, agmsgBefore := len(readLogLines(t, herdrLog)), len(readLogLines(t, agmsgLog))

			stdout, stderr, err := runRootCmdSplit(t, append([]string{"org"}, tc.args(t.TempDir())...)...)
			if err == nil {
				t.Fatalf("expected %s to be refused while the legacy ledger has an active seat; stdout: %s stderr: %s", tc.name, stdout, stderr)
			}
			msg := err.Error()
			for _, want := range []string{
				r.legacyDir, r.sharedDir, "1 active seat(s)",
				"--state-dir " + shellQuoteIfNeeded(r.legacyDir),
				"--state-dir " + shellQuoteIfNeeded(r.sharedDir),
			} {
				if !strings.Contains(msg, want) {
					t.Errorf("refusal message missing %q: %s", want, msg)
				}
			}
			if stdout != "" {
				t.Errorf("expected nothing on stdout for a refused verb, got: %s", stdout)
			}
			if after := readFileBytes(t, legacyManifest); !bytes.Equal(before, after) {
				t.Errorf("legacy manifest changed:\nbefore:\n%s\nafter:\n%s", before, after)
			}
			if _, statErr := os.Stat(r.sharedDir); !errors.Is(statErr, fs.ErrNotExist) {
				t.Errorf("expected the shared ledger dir %s not to be created, stat err: %v", r.sharedDir, statErr)
			}
			if got := len(readLogLines(t, herdrLog)); got != herdrBefore {
				t.Errorf("herdr was called %d time(s) after the refusal, want 0", got-herdrBefore)
			}
			if got := len(readLogLines(t, agmsgLog)); got != agmsgBefore {
				t.Errorf("agmsg was called %d time(s) after the refusal, want 0", got-agmsgBefore)
			}
		})
	}
}

func TestOrgLegacyLedger_ExplicitStateDirSkipsGuard(t *testing.T) {
	cases := []struct {
		name   string
		choose func(t *testing.T, r legacyLedgerRepo) []string
	}{
		{"flag", func(t *testing.T, r legacyLedgerRepo) []string {
			return []string{"--state-dir", r.legacyDir}
		}},
		{"env", func(t *testing.T, r legacyLedgerRepo) []string {
			t.Setenv(org.EnvOrgStateDir, r.legacyDir)
			return nil
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			setupOrgStubPATH(t)
			r := newLegacyLedgerRepo(t)
			spawnLedgerSeat(t, r.legacyDir, "seat-1")

			args := append([]string{"org", "stop", "--org-id", "org-a", "--seat", "seat-1"}, tc.choose(t, r)...)
			stdout, stderr, err := runRootCmdSplit(t, args...)
			if err != nil {
				t.Fatalf("expected stop with the ledger chosen by %s to run, got: %v (stdout: %s stderr: %s)", tc.name, err, stdout, stderr)
			}
			if strings.Contains(stderr, legacyNoteMarker) {
				t.Errorf("expected no legacy-ledger note when %s chose the ledger, stderr: %s", tc.name, stderr)
			}
			events := readManifestEvents(t, org.ManifestPathIn(r.legacyDir))
			if last := events[len(events)-1]; last.Event != org.EventStopped || last.SeatID != "seat-1" {
				t.Errorf("expected the legacy ledger to record seat-1 stopped, last event: %+v", last)
			}
		})
	}
}

func TestOrgLegacyLedger_MutatingVerbWithoutActiveLegacySeats(t *testing.T) {
	dryRunSpawn := func(seatCwd string) []string {
		return []string{"org", "spawn", "--org-id", "org-a", "--id", "seat-2", "--role", "worker",
			"--driver", "claude", "--model", "sonnet", "--cwd", seatCwd, "--scope", "test-scope", "--dry-run"}
	}

	t.Run("legacy_ledger_without_active_seats_notes_once", func(t *testing.T) {
		setupOrgStubPATH(t)
		r := newLegacyLedgerRepo(t)
		spawnLedgerSeat(t, r.legacyDir, "seat-1")
		if out, err := runOrgCmd(t, "stop", "--org-id", "org-a", "--seat", "seat-1", "--state-dir", r.legacyDir); err != nil {
			t.Fatalf("seed stop: %v (output: %s)", err, out)
		}
		legacyManifest := org.ManifestPathIn(r.legacyDir)
		before := readFileBytes(t, legacyManifest)

		stdout, stderr, err := runRootCmdSplit(t, dryRunSpawn(t.TempDir())...)
		if err != nil {
			t.Fatalf("expected spawn to run when the legacy ledger has no active seat, got: %v (stderr: %s)", err, stderr)
		}
		if n := strings.Count(stderr, legacyNoteMarker); n != 1 {
			t.Errorf("expected the legacy-ledger note once on stderr, got %d time(s): %s", n, stderr)
		}
		for _, want := range []string{r.legacyDir, r.sharedDir, "0 active seat(s)", "--state-dir " + shellQuoteIfNeeded(r.legacyDir)} {
			if !strings.Contains(stderr, want) {
				t.Errorf("note missing %q: %s", want, stderr)
			}
		}
		if strings.Contains(stdout, legacyNoteMarker) {
			t.Errorf("expected the note on stderr only, stdout: %s", stdout)
		}
		var found bool
		for _, ev := range readManifestEvents(t, org.ManifestPathIn(r.sharedDir)) {
			found = found || (ev.SeatID == "seat-2" && ev.Event == org.EventSpawned)
		}
		if !found {
			t.Errorf("expected seat-2 to be recorded in the shared ledger %s", r.sharedDir)
		}
		if after := readFileBytes(t, legacyManifest); !bytes.Equal(before, after) {
			t.Errorf("legacy manifest changed:\nbefore:\n%s\nafter:\n%s", before, after)
		}
	})

	t.Run("no_legacy_ledger_prints_no_note", func(t *testing.T) {
		setupOrgStubPATH(t)
		newLegacyLedgerRepo(t)

		_, stderr, err := runRootCmdSplit(t, dryRunSpawn(t.TempDir())...)
		if err != nil {
			t.Fatalf("spawn --dry-run: %v (stderr: %s)", err, stderr)
		}
		if strings.Contains(stderr, legacyNoteMarker) {
			t.Errorf("expected no legacy-ledger note without a legacy ledger, stderr: %s", stderr)
		}
	})
}

func TestOrgLegacyLedger_ReadOnlyVerbsNoteAndContinue(t *testing.T) {
	setupOrgStubPATH(t)
	r := newLegacyLedgerRepo(t)
	spawnLedgerSeat(t, r.legacyDir, "seat-1")
	spawnLedgerSeat(t, r.sharedDir, "seat-1")
	legacyManifest := org.ManifestPathIn(r.legacyDir)
	before := readFileBytes(t, legacyManifest)

	stateDirHint := "--state-dir " + shellQuoteIfNeeded(r.legacyDir)
	cases := []struct {
		name     string
		args     []string
		jsonOut  bool
		wantHint string
	}{
		{"status", []string{"status"}, false, stateDirHint},
		{"status_json", []string{"status", "--json"}, true, stateDirHint},
		{"org_status_json", []string{"org", "status", "--org-id", "org-a", "--json"}, true, stateDirHint},
		{"org_read", []string{"org", "read", "--org-id", "org-a", "--seat", "seat-1"}, false, stateDirHint},
		{"org_wait", []string{"org", "wait", "--org-id", "org-a", "--seat", "seat-1"}, false, stateDirHint},
		{"org_report", []string{"org", "report", "--org-id", "org-a", "--out", t.TempDir()}, false, stateDirHint},
		// insights has no --state-dir flag; the note names the env var.
		{"insights_json", []string{"insights", "--json", "--events-dir", t.TempDir()}, true,
			org.EnvOrgStateDir + "=" + shellQuoteIfNeeded(r.legacyDir)},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			stdout, stderr, err := runRootCmdSplit(t, tc.args...)
			if err != nil {
				t.Fatalf("expected %v to run with a note, got: %v (stderr: %s)", tc.args, err, stderr)
			}
			if n := strings.Count(stderr, legacyNoteMarker); n != 1 {
				t.Errorf("expected the legacy-ledger note once on stderr, got %d time(s): %s", n, stderr)
			}
			for _, want := range []string{r.legacyDir, "1 active seat(s)", tc.wantHint} {
				if !strings.Contains(stderr, want) {
					t.Errorf("note missing %q: %s", want, stderr)
				}
			}
			if strings.Contains(stdout, legacyNoteMarker) {
				t.Errorf("expected the note on stderr only, stdout: %s", stdout)
			}
			if tc.jsonOut && !json.Valid([]byte(stdout)) {
				t.Errorf("expected valid JSON on stdout, got: %s", stdout)
			}
		})
	}

	if after := readFileBytes(t, legacyManifest); !bytes.Equal(before, after) {
		t.Errorf("legacy manifest changed by a read-only verb:\nbefore:\n%s\nafter:\n%s", before, after)
	}
}

// TestStatusJSON_FromLinkedWorktreeReportsSharedLedger is AC7: `ralph status
// --json` from a linked worktree reports the main checkout's ledger.
func TestStatusJSON_FromLinkedWorktreeReportsSharedLedger(t *testing.T) {
	type statusPayload struct {
		StateDir       string `json:"state_dir"`
		StateDirSource string `json:"state_dir_source"`
	}
	cases := []struct {
		name       string
		seedLegacy bool
		fromMain   bool
		wantNote   bool
	}{
		{"worktree_without_legacy_ledger", false, false, false},
		{"worktree_with_legacy_ledger", true, false, true},
		{"main_checkout_with_legacy_ledger_in_worktree", true, true, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			setupOrgStubPATH(t)
			r := newLegacyLedgerRepo(t)
			if tc.seedLegacy {
				spawnLedgerSeat(t, r.legacyDir, "seat-1")
			}
			if tc.fromMain {
				t.Chdir(r.mainRoot)
			}

			stdout, stderr, err := runRootCmdSplit(t, "status", "--json")
			if err != nil {
				t.Fatalf("status --json: %v (stderr: %s)", err, stderr)
			}
			var payload statusPayload
			if err := json.Unmarshal([]byte(stdout), &payload); err != nil {
				t.Fatalf("stdout is not JSON: %v\n%s", err, stdout)
			}
			if payload.StateDir != r.sharedDir {
				t.Errorf("state_dir = %q, want the main checkout's %q", payload.StateDir, r.sharedDir)
			}
			if payload.StateDirSource != "git-main-worktree" {
				t.Errorf("state_dir_source = %q, want %q", payload.StateDirSource, "git-main-worktree")
			}
			if gotNote := strings.Contains(stderr, legacyNoteMarker); gotNote != tc.wantNote {
				t.Errorf("legacy-ledger note on stderr = %t, want %t; stderr: %s", gotNote, tc.wantNote, stderr)
			}
		})
	}
}

// TestOrgLegacyLedger_UnreadableLegacyLedger covers a legacy manifest that
// exists but cannot be read: a mutating verb cannot tell whether seats are
// still active there, so it is refused; a read-only verb notes it and runs.
func TestOrgLegacyLedger_UnreadableLegacyLedger(t *testing.T) {
	setupOrgStubPATH(t)
	r := newLegacyLedgerRepo(t)
	// A directory where manifest.jsonl should be: stat succeeds, the read
	// fails.
	if err := os.MkdirAll(org.ManifestPathIn(r.legacyDir), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	_, _, err := runRootCmdSplit(t, "org", "stop", "--org-id", "org-a", "--seat", "seat-1")
	if err == nil {
		t.Fatal("expected stop to be refused when the legacy ledger cannot be read")
	}
	if !strings.Contains(err.Error(), "--state-dir "+shellQuoteIfNeeded(r.sharedDir)) {
		t.Errorf("refusal message should say how to choose a ledger with --state-dir: %v", err)
	}

	stdout, stderr, err := runRootCmdSplit(t, "status", "--json")
	if err != nil {
		t.Fatalf("status --json: %v (stderr: %s)", err, stderr)
	}
	if !strings.Contains(stderr, "cannot read this linked worktree's older org ledger") {
		t.Errorf("expected a note about the unreadable legacy ledger, stderr: %s", stderr)
	}
	if !json.Valid([]byte(stdout)) {
		t.Errorf("expected valid JSON on stdout, got: %s", stdout)
	}
}
