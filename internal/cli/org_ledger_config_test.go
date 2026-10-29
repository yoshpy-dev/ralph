package cli

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/yoshpy-dev/ralph/internal/config"
	"github.com/yoshpy-dev/ralph/internal/org"
)

// Tests for the personal file's [org].state_dir as a tier of the ledger
// resolution, through the real CLI path (resolveOrgLedger), for plan
// 2026-10-30-org-state-dir-local (AC1, AC2, AC3, AC5). TestMain turns the
// personal file off (config.LocalConfigEnv), so each test turns it back on.
// Every test chdirs, so none of them runs in parallel.

// newOrgLedgerConfigRepo is newLegacyLedgerRepo (the test runs in its linked
// worktree) with the personal file at the main worktree's root naming
// ledger, an absolute directory outside the repository that holds
// seedTwoOrgManifest's orgs. The default ledger (r.sharedDir) does not
// exist, so a verb that reads it finds no org.
func newOrgLedgerConfigRepo(t *testing.T) (r legacyLedgerRepo, ledger string) {
	t.Helper()
	r = newLegacyLedgerRepo(t)
	t.Setenv(config.LocalConfigEnv, "")
	ledger = filepath.Join(t.TempDir(), "ledger")
	seedTwoOrgManifest(t, ledger)
	writeTestFile(t, filepath.Join(r.mainRoot, config.LocalConfigFileName), "[org]\nstate_dir = \""+ledger+"\"\n")
	return r, ledger
}

// orgStatusLeaderRow is the row `ralph org status --org-id org-a` prints for
// seedTwoOrgManifest's active leader.
const orgStatusLeaderRow = "leader\tleader\tclaude\topus\tspawned (active)\t"

// TestOrgLedger_PersonalFileNamesTheLedgerFromAnyWorktree covers AC1 and
// AC2: `ralph status` and `ralph org status` read the ledger the personal
// file at the main worktree's root names, with source config, from the main
// checkout, a subdirectory of it, the linked worktree, and a subdirectory of
// that. The linked worktree's own ralph.local.toml, naming another ledger,
// is not read.
func TestOrgLedger_PersonalFileNamesTheLedgerFromAnyWorktree(t *testing.T) {
	r, ledger := newOrgLedgerConfigRepo(t)
	writeTestFile(t, filepath.Join(r.worktree, config.LocalConfigFileName),
		"[org]\nstate_dir = \""+filepath.Join(t.TempDir(), "worktree-ledger")+"\"\n")

	for _, dir := range []string{
		r.mainRoot,
		filepath.Join(r.mainRoot, "pkg", "sub"),
		r.worktree,
		filepath.Join(r.worktree, "pkg", "sub"),
	} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		t.Chdir(dir)

		stdout, stderr, err := runRootCmdSplit(t, "status", "--json")
		if err != nil {
			t.Fatalf("in %s: ralph status --json: %v\nstderr: %s", dir, err, stderr)
		}
		var got struct {
			StateDir       string          `json:"state_dir"`
			StateDirSource string          `json:"state_dir_source"`
			Orgs           []statusOrgJSON `json:"orgs"`
		}
		if err := json.Unmarshal([]byte(stdout), &got); err != nil {
			t.Fatalf("in %s: ralph status --json does not parse: %v\n%s", dir, err, stdout)
		}
		if got.StateDir != ledger || got.StateDirSource != "config" || len(got.Orgs) != 2 {
			t.Errorf("in %s: ralph status --json = state_dir %q, source %q, %d orgs; want %q, config, 2 orgs",
				dir, got.StateDir, got.StateDirSource, len(got.Orgs), ledger)
		}

		stdout, stderr, err = runRootCmdSplit(t, "org", "status", "--org-id", "org-a")
		if err != nil {
			t.Fatalf("in %s: ralph org status: %v\nstderr: %s", dir, err, stderr)
		}
		if !strings.Contains(stdout, orgStatusLeaderRow) {
			t.Errorf("in %s: ralph org status shows no leader of org-a from %s:\n%s", dir, ledger, stdout)
		}
	}
}

// TestOrgLedger_ConfigFlagDoesNotChooseThePersonalFile covers AC2 (D1): a
// --config naming a ralph.toml in another directory, whose ralph.local.toml
// names another ledger, does not change the ledger: it is still the one the
// personal file of the repository the command runs in names.
func TestOrgLedger_ConfigFlagDoesNotChooseThePersonalFile(t *testing.T) {
	newOrgLedgerConfigRepo(t)
	other := t.TempDir()
	otherToml := filepath.Join(other, "ralph.toml")
	writeTestFile(t, otherToml, "")
	writeTestFile(t, filepath.Join(other, config.LocalConfigFileName),
		"[org]\nstate_dir = \""+filepath.Join(t.TempDir(), "other-ledger")+"\"\n")

	stdout, stderr, err := runRootCmdSplit(t, "org", "status", "--org-id", "org-a", "--config", otherToml)
	if err != nil {
		t.Fatalf("ralph org status --config: %v\nstderr: %s", err, stderr)
	}
	if !strings.Contains(stdout, orgStatusLeaderRow) {
		t.Errorf("ralph org status --config %s shows no leader of org-a, so it did not read the repository's personal file:\n%s", otherToml, stdout)
	}
}

// TestOrgLedger_RelativePersonalStateDirFailsUntilStateDirIsPassed covers
// AC3 and AC5 (D2, D4): a relative [org].state_dir makes every verb that
// resolves the ledger without --state-dir exit 1, naming the command, the
// file, and the value, before it reads or writes a ledger. With --state-dir
// the file is not read for the ledger, and config.Load, which `ralph org
// stop` runs (newOrgRuntimeAt) and which reads the same file, does not check
// the value, so the verb runs.
func TestOrgLedger_RelativePersonalStateDirFailsUntilStateDirIsPassed(t *testing.T) {
	r := newLegacyLedgerRepo(t)
	t.Setenv(config.LocalConfigEnv, "")
	path := filepath.Join(r.mainRoot, config.LocalConfigFileName)
	writeTestFile(t, path, "[org]\nstate_dir = \"ledger\"\n")

	for _, tc := range []struct {
		command string
		args    []string
	}{
		{"ralph org status", []string{"org", "status", "--org-id", "org-a"}},
		{"ralph org stop", []string{"org", "stop", "--all"}},
		{"ralph status", []string{"status"}},
		{"ralph insights", []string{"insights"}},
	} {
		_, _, err := runRootCmdSplit(t, tc.args...)
		want := tc.command + ": cannot resolve the org ledger: " + path + `: [org].state_dir "ledger" must be an absolute path or start with ~/`
		if code := ExitCode(err); code != 1 || !strings.Contains(err.Error(), want) {
			t.Errorf("ralph %s = exit %d, %v; want exit 1 with %q", strings.Join(tc.args, " "), code, err, want)
		}
	}
	if _, err := os.Stat(r.sharedDir); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("the default ledger %s exists after the refusals (stat: %v), want nothing written", r.sharedDir, err)
	}

	setupOrgStubPATH(t)
	stateDir := filepath.Join(t.TempDir(), "state")
	stdout, stderr, err := runRootCmdSplit(t, "org", "stop", "--all", "--state-dir", stateDir)
	if err != nil || strings.TrimSpace(stdout) != "no active seats" {
		t.Errorf("ralph org stop --all --state-dir = (stdout %q, stderr %q, %v); want \"no active seats\" and no error", stdout, stderr, err)
	}
	seedTwoOrgManifest(t, stateDir)
	stdout, stderr, err = runRootCmdSplit(t, "org", "status", "--org-id", "org-a", "--state-dir", stateDir)
	if err != nil || !strings.Contains(stdout, orgStatusLeaderRow) {
		t.Errorf("ralph org status --state-dir = (stdout %q, stderr %q, %v); want org-a's leader and no error", stdout, stderr, err)
	}
}

// TestOrgLedger_UnparsablePersonalFileNamesItsPath covers AC5: a personal
// file that is not TOML makes a verb exit 1 naming the file. Without
// --state-dir the ledger resolution refuses it; with --state-dir config.Load
// does (since A1), so the file has to be fixed.
func TestOrgLedger_UnparsablePersonalFileNamesItsPath(t *testing.T) {
	r := newLegacyLedgerRepo(t)
	t.Setenv(config.LocalConfigEnv, "")
	path := filepath.Join(r.mainRoot, config.LocalConfigFileName)
	writeTestFile(t, path, "[org\nstate_dir = \"/ledger\"\n")
	stateDir := filepath.Join(t.TempDir(), "state")

	for _, tc := range []struct {
		args []string
		want string
	}{
		{[]string{"org", "status", "--org-id", "org-a"}, "ralph org status: cannot resolve the org ledger: " + path + ": "},
		{[]string{"status"}, "ralph status: cannot resolve the org ledger: " + path + ": "},
		{[]string{"org", "status", "--org-id", "org-a", "--state-dir", stateDir}, path + ": "},
		{[]string{"org", "stop", "--all", "--state-dir", stateDir}, path + ": "},
	} {
		_, _, err := runRootCmdSplit(t, tc.args...)
		if code := ExitCode(err); code != 1 || !strings.Contains(err.Error(), tc.want) {
			t.Errorf("ralph %s = exit %d, %v; want exit 1 with %q", strings.Join(tc.args, " "), code, err, tc.want)
		}
	}
}

// TestOrgEscalate_LedgerThatCannotBeResolvedStillReachesTheHuman covers S2b:
// without --state-dir, a personal file that is not TOML, or whose
// [org].state_dir is relative, leaves `ralph org escalate` with no ledger
// (D4), and the item still reaches the human as not recorded: exit 1, no
// ID, the banner naming the cause, and one desktop notification. A message
// escalate refuses is refused first, with no banner and no notification.
// Neither writes an inbox.jsonl or an escalations.jsonl anywhere in the
// repository's temp tree.
func TestOrgEscalate_LedgerThatCannotBeResolvedStillReachesTheHuman(t *testing.T) {
	for _, tc := range []struct {
		name, body, causeTail string
	}{
		{"not TOML", "[org\nstate_dir = \"/ledger\"\n", ": "},
		{"relative state_dir", "[org]\nstate_dir = \"ledger\"\n", `: [org].state_dir "ledger" must be an absolute path or start with ~/`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := newLegacyLedgerRepo(t)
			t.Setenv(config.LocalConfigEnv, "")
			path := filepath.Join(r.mainRoot, config.LocalConfigFileName)
			writeTestFile(t, path, tc.body)
			cause := "ralph org escalate: cannot resolve the org ledger: " + path

			rec := recordDesktopNotify(t, nil)
			stdout, stderr, err := runRootCmdSplit(t, "org", "escalate", "--org-id", "org-a", "--text", "TYPE: BLOCKED\nTASK_ID: org-a\n\nstuck")
			if code := ExitCode(err); code != 1 || !errors.Is(err, errOrgLedgerUnresolved) ||
				!strings.Contains(err.Error(), "org: escalate: the item could not be recorded: "+cause) {
				t.Errorf("escalate = exit %d, %v; want exit 1, the item could not be recorded, naming %q", code, err, cause)
			}
			if stdout != "" {
				t.Errorf("stdout = %q; want no ID for an item that was not recorded", stdout)
			}
			banner := "ORG ESCALATION (NOT RECORDED): org=org-a type=BLOCKED -- the item is not in any inbox: " + cause + tc.causeTail
			if !strings.Contains(stderr, banner) || strings.Count(stderr, "ORG ESCALATION") != 1 {
				t.Errorf("stderr = %q; want one not-recorded banner with %q", stderr, banner)
			}
			if calls := rec.calls(); len(calls) != 1 || !strings.Contains(calls[0], "org-a") || strings.Contains(calls[0], "stuck") {
				t.Errorf("desktop notifications = %q; want one naming org-a without the body", calls)
			}

			rec = recordDesktopNotify(t, nil)
			stdout, stderr, err = runRootCmdSplit(t, "org", "escalate", "--org-id", "org-a", "--text", "TYPE: ALERT\n\nstuck")
			if code := ExitCode(err); code != 1 || !strings.Contains(err.Error(), "TYPE ALERT cannot be escalated") {
				t.Errorf("escalate of an ALERT = exit %d, %v; want exit 1 with the TYPE refusal", code, err)
			}
			if stdout != "" || strings.Contains(stderr, "ORG ESCALATION") || len(rec.calls()) != 0 {
				t.Errorf("a refusal printed stdout %q, stderr %q, notified %q", stdout, stderr, rec.calls())
			}

			if _, err := os.Stat(r.sharedDir); !errors.Is(err, os.ErrNotExist) {
				t.Errorf("the default ledger %s exists (stat: %v), want nothing written", r.sharedDir, err)
			}
			inboxName, escalationsName := filepath.Base(org.InboxPathIn(r.sharedDir)), filepath.Base(org.EscalationsPathIn(r.sharedDir))
			err = filepath.WalkDir(filepath.Dir(r.mainRoot), func(p string, d os.DirEntry, err error) error {
				if err != nil {
					return err
				}
				if d.Name() == inboxName || d.Name() == escalationsName {
					t.Errorf("escalate wrote %s, want no ledger file anywhere", p)
				}
				return nil
			})
			if err != nil {
				t.Fatalf("walk the temp tree: %v", err)
			}
		})
	}
}

// TestRunDoctorFull_PersonalFileThatCannotResolveTheLedgerIsAFailLine covers
// AC5 for doctor (D4): a personal file whose [org].state_dir is relative, or
// that is not TOML, gives one fail line naming the file, doctor runs its
// other checks, the merge guarantee still runs, and no merge-guarantee.json
// is written into the ledger the default tiers would name. With no personal
// file the cache is written there, which is what makes the absence mean
// something.
func TestRunDoctorFull_PersonalFileThatCannotResolveTheLedgerIsAFailLine(t *testing.T) {
	dir, _ := doctorMGProject(t)
	t.Chdir(dir)
	t.Setenv(config.LocalConfigEnv, "")
	path := config.LocalPath("ralph.toml")
	ledger, _, err := org.ResolveOrgStateDir("", false)
	if err != nil {
		t.Fatalf("resolve the default ledger: %v", err)
	}
	if err := os.MkdirAll(ledger, 0o755); err != nil {
		t.Fatal(err)
	}
	e := newDoctorMGExec(t, org.MergeGuaranteeGitHubProtection)
	fake := doctorMGEnv(e, true, "")
	useDoctorMGEnv(t, func(targetDir string) (org.MergeGuaranteeParams, string, error) {
		params, _, _ := fake(targetDir)
		_, stateDir, err := mergeGuaranteeEnvFromOS(targetDir)
		return params, stateDir, err
	})

	for _, tc := range []struct {
		name, body, wantLedgerLine string
	}{
		{"relative state_dir", "[org]\nstate_dir = \"ledger\"\n",
			"  ✗ Org ledger: fail — cannot resolve the org ledger: " + path + `: [org].state_dir "ledger" must be an absolute path or start with ~/`},
		{"not TOML", "[org\n", "  ✗ Org ledger: fail — cannot resolve the org ledger: " + path + ": "},
		{"no personal file", "", ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_ = os.Remove(org.MergeGuaranteePathIn(ledger))
			if tc.body == "" {
				if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
					t.Fatal(err)
				}
			} else {
				writeTestFile(t, path, tc.body)
			}
			var out string
			err := captureStdoutText(t, &out, func() error { return runDoctorFull(dir, false, false) })
			_, cacheErr := os.Stat(org.MergeGuaranteePathIn(ledger))
			for _, want := range []string{"Org envelope:", "Merge guarantee: pass — github_protection: ", "Scaffold: core file hashes:"} {
				if !strings.Contains(out, want) {
					t.Errorf("output has no %q line, so doctor did not go on with its checks:\n%s", want, out)
				}
			}
			if tc.wantLedgerLine == "" {
				if err != nil || strings.Contains(out, "Org ledger") || cacheErr != nil {
					t.Errorf("runDoctorFull = %v, cache stat %v; want no error, no Org ledger line, and the cache in %s:\n%s", err, cacheErr, ledger, out)
				}
				return
			}
			if err == nil || err.Error() != "doctor: 1 check(s) failed" {
				t.Errorf("runDoctorFull = %v; want doctor: 1 check(s) failed", err)
			}
			if !strings.Contains(out, tc.wantLedgerLine) {
				t.Errorf("output has no line with %q:\n%s", tc.wantLedgerLine, out)
			}
			if !strings.Contains(out, "; not cached: the org ledger could not be resolved\n") {
				t.Errorf("the merge guarantee line does not say it was not cached:\n%s", out)
			}
			if !errors.Is(cacheErr, os.ErrNotExist) {
				t.Errorf("merge-guarantee.json was written into %s (stat: %v), want no cache", ledger, cacheErr)
			}
		})
	}
}
