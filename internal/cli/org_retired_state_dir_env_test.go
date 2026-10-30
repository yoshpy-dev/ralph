package cli

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/yoshpy-dev/ralph/internal/config"
	"github.com/yoshpy-dev/ralph/internal/org"
)

// Tests for checkRetiredOrgStateDirEnv (through resolveOrgLedger) and for
// doctor's warn about RALPH_ORG_STATE_DIR (checkRetiredEnv), plan
// 2026-10-30-org-state-dir-local, D5 and AC6. TestMain unsets the variable,
// so each test sets it with t.Setenv. The verbs run in the linked worktree
// of a temp repository (newLegacyLedgerRepo), where the default ledger is
// the main worktree's (sharedDir); every such test chdirs, so none of them
// runs in parallel.

const (
	// retiredEnvNote starts both notes checkRetiredOrgStateDirEnv prints.
	retiredEnvNote = "note: RALPH_ORG_STATE_DIR is set ("
	// retiredEnvNoteRefusal is in the note of a read-only verb whose
	// default ledger is not the variable's, and in no other.
	retiredEnvNoteRefusal = "the commands that change the ledger are refused while the variable names another one"
	// retiredEnvNoteTail ends both notes.
	retiredEnvNoteTail = "Set [org].state_dir in ralph.local.toml or pass --state-dir to choose the ledger, and unset RALPH_ORG_STATE_DIR."
)

// retiredEnvRepo is a newLegacyLedgerRepo (the test runs in its linked
// worktree) with setupOrgStubPATH's herdr and agmsg, and oldLedger, a
// directory outside the repository holding seedTwoOrgManifest's orgs: the
// ledger a user named with RALPH_ORG_STATE_DIR before ralph stopped reading
// it. The variable is not set yet.
type retiredEnvRepo struct {
	legacyLedgerRepo
	oldLedger          string
	herdrLog, agmsgLog string
}

func newRetiredEnvRepo(t *testing.T) retiredEnvRepo {
	t.Helper()
	herdrLog, agmsgLog := setupOrgStubPATH(t)
	r := retiredEnvRepo{legacyLedgerRepo: newLegacyLedgerRepo(t), herdrLog: herdrLog, agmsgLog: agmsgLog}
	r.oldLedger = filepath.Join(t.TempDir(), "old ledger")
	seedTwoOrgManifest(t, r.oldLedger)
	return r
}

// retiredEnvSpawnArgs is `ralph org spawn` of seat-1 into org-n, an org in
// neither ledger, with extra flags after it.
func retiredEnvSpawnArgs(t *testing.T, extra ...string) []string {
	return append([]string{"org", "spawn", "--org-id", "org-n", "--id", "seat-1", "--role", "worker",
		"--driver", "claude", "--model", "sonnet", "--cwd", t.TempDir(), "--scope", "test-scope"}, extra...)
}

// checkRetiredEnvNote checks that stderr holds exactly one note of
// checkRetiredOrgStateDirEnv, on one line naming envDir and ledger, with the
// refusal sentence when refused is true and without it otherwise.
func checkRetiredEnvNote(t *testing.T, what, stderr, envDir, ledger string, refused bool) {
	t.Helper()
	if n := strings.Count(stderr, retiredEnvNote); n != 1 {
		t.Errorf("%s: stderr has the RALPH_ORG_STATE_DIR note %d times; want once:\n%s", what, n, stderr)
		return
	}
	var line string
	for _, l := range strings.Split(stderr, "\n") {
		if strings.HasPrefix(l, retiredEnvNote) {
			line = l
		}
	}
	ledgerPart := "using the org ledger at " + ledger + ". "
	if refused {
		ledgerPart = "showing the org ledger at " + ledger + " instead, "
	}
	for _, want := range []string{retiredEnvNote + envDir + ")", ledgerPart, retiredEnvNoteTail} {
		if !strings.Contains(line, want) {
			t.Errorf("%s: the note %q does not contain %q", what, line, want)
		}
	}
	if got := strings.Contains(line, retiredEnvNoteRefusal); got != refused {
		t.Errorf("%s: the note %q says the changing commands are refused: %t; want %t", what, line, got, refused)
	}
}

// TestOrgRetiredStateDirEnv_ReadOnlyVerbsShowTheDefaultLedgerWithANote
// covers AC6 for the verbs that only read the ledger: with
// RALPH_ORG_STATE_DIR naming another ledger than the default one, `ralph
// status` and `ralph org status` show the default ledger (source
// git-main-worktree, none of the variable's orgs), exit 0, and print one
// note on stderr that names both directories and says the verbs that change
// the ledger are refused. stdout stays one JSON document with --json.
func TestOrgRetiredStateDirEnv_ReadOnlyVerbsShowTheDefaultLedgerWithANote(t *testing.T) {
	r := newRetiredEnvRepo(t)
	t.Setenv(retiredOrgStateDirEnv, r.oldLedger)

	stdout, stderr, err := runRootCmdSplit(t, "status", "--json")
	if err != nil {
		t.Fatalf("ralph status --json: %v\nstderr: %s", err, stderr)
	}
	var got struct {
		StateDir       string          `json:"state_dir"`
		StateDirSource string          `json:"state_dir_source"`
		Orgs           []statusOrgJSON `json:"orgs"`
	}
	if err := json.Unmarshal([]byte(stdout), &got); err != nil {
		t.Fatalf("ralph status --json does not parse: %v\n%s", err, stdout)
	}
	if got.StateDir != r.sharedDir || got.StateDirSource != "git-main-worktree" || len(got.Orgs) != 0 {
		t.Errorf("ralph status --json = state_dir %q, source %q, %d orgs; want %q, git-main-worktree, no org",
			got.StateDir, got.StateDirSource, len(got.Orgs), r.sharedDir)
	}
	checkRetiredEnvNote(t, "ralph status --json", stderr, r.oldLedger, r.sharedDir, true)

	stdout, stderr, err = runRootCmdSplit(t, "org", "status", "--org-id", "org-a")
	if err != nil {
		t.Fatalf("ralph org status: %v\nstderr: %s", err, stderr)
	}
	if strings.Contains(stdout, orgStatusLeaderRow) {
		t.Errorf("ralph org status shows org-a's leader from the variable's ledger %s:\n%s", r.oldLedger, stdout)
	}
	checkRetiredEnvNote(t, "ralph org status", stderr, r.oldLedger, r.sharedDir, true)
}

// TestOrgRetiredStateDirEnv_MutatingVerbsRefuseAnotherLedger covers AC6 for
// the verbs that change the ledger: with RALPH_ORG_STATE_DIR naming another
// ledger than the default one, `ralph org start`, `ralph org spawn`, and
// `ralph org send` of a TASK exit 1, with no note, and an error, marked
// errOrgLedgerUnresolved, that names both directories and the ways out.
// Neither ledger gains anything (the default one is not even created), no
// watch is launched, and neither herdr nor agmsg is called. The real watch
// gate is on, and with the variable naming the default ledger the same
// start launches the watch once and spawns the leader, so the launch seam
// is reachable.
func TestOrgRetiredStateDirEnv_MutatingVerbsRefuseAnotherLedger(t *testing.T) {
	r := newRetiredEnvRepo(t)
	useRealOrgWatchGate(t)
	refuseOrgWatchLaunch(t)
	t.Setenv(retiredOrgStateDirEnv, r.oldLedger)
	before := readFileBytes(t, org.ManifestPathIn(r.oldLedger))
	startArgs := []string{"org", "start", "--org-id", "org-n", "--driver", "claude", "--model", "sonnet",
		"--cwd", t.TempDir(), "--scope", "org-wide", "do the task"}

	ways := "to keep using " + r.oldLedger + ", write [org].state_dir = " + strconv.Quote(r.oldLedger) +
		" in " + filepath.Join(r.mainRoot, config.LocalConfigFileName) + " or pass --state-dir " + shellQuoteIfNeeded(r.oldLedger) +
		", and unset RALPH_ORG_STATE_DIR; to use " + r.sharedDir + ", unset RALPH_ORG_STATE_DIR"
	for _, tc := range []struct {
		command string
		args    []string
	}{
		{"ralph org start", startArgs},
		{"ralph org spawn", retiredEnvSpawnArgs(t)},
		{"ralph org send", []string{"org", "send", "--org-id", "org-a", "--to", "leader", "--text", "TYPE: TASK\nTASK_ID: t-1\n\ndo it"}},
	} {
		stdout, stderr, err := runRootCmdSplit(t, tc.args...)
		want := tc.command + ": cannot resolve the org ledger: RALPH_ORG_STATE_DIR names " + r.oldLedger +
			", which ralph no longer reads, and the ledger ralph finds by default is " + r.sharedDir + ", so nothing was changed; " + ways
		if code := ExitCode(err); code != 1 || !errors.Is(err, errOrgLedgerUnresolved) || err == nil || err.Error() != want {
			t.Errorf("%s = exit %d, %v\nwant exit 1 with\n%s", tc.command, code, err, want)
		}
		if stdout != "" || strings.Contains(stderr, retiredEnvNote) {
			t.Errorf("%s printed stdout %q, stderr %q; want nothing beside the error", tc.command, stdout, stderr)
		}
	}
	if _, err := os.Stat(r.sharedDir); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("the default ledger %s exists after the refusals (stat: %v); want nothing written", r.sharedDir, err)
	}
	if after := readFileBytes(t, org.ManifestPathIn(r.oldLedger)); !bytes.Equal(before, after) {
		t.Errorf("the variable's ledger changed:\nbefore:\n%s\nafter:\n%s", before, after)
	}
	for name, log := range map[string]string{"herdr": r.herdrLog, "agmsg": r.agmsgLog} {
		if lines := readLogLines(t, log); len(lines) != 0 {
			t.Errorf("%s calls = %v; want none", name, lines)
		}
	}

	t.Setenv(retiredOrgStateDirEnv, r.sharedDir)
	var launches []orgWatchLaunchRequest
	setOrgWatchLaunch(t, lateMarkLauncher(t, r.sharedDir, 4501, 10*time.Millisecond, &launches))
	stdout, stderr, err := runRootCmdSplit(t, startArgs...)
	if err != nil || !strings.HasPrefix(stdout, `spawned seat "leader" `) {
		t.Fatalf("start with the variable naming the default ledger = %v; want the leader spawned (stdout: %s stderr: %s)", err, stdout, stderr)
	}
	if len(launches) != 1 {
		t.Errorf("launches = %d; want 1", len(launches))
	}
	checkRetiredEnvNote(t, "ralph org start", stderr, r.sharedDir, r.sharedDir, false)
}

// TestOrgRetiredStateDirEnv_OnlyANoteWhenTheLedgerIsChosenOrTheSame covers
// the rest of AC6: with RALPH_ORG_STATE_DIR set, `ralph org spawn` runs and
// records the seat in the ledger it uses, with one note naming the
// variable's value and that ledger, when --state-dir names the ledger
// (the variable's or another), when the personal file's [org].state_dir
// names it (another than the variable's), and when the default ledger is
// the directory the variable names, also as a path relative to the working
// directory, which is made absolute from it. A blank value prints no note.
func TestOrgRetiredStateDirEnv_OnlyANoteWhenTheLedgerIsChosenOrTheSame(t *testing.T) {
	cases := []struct {
		name string
		// setup sets the variable and returns spawn's extra flags, the
		// ledger the seat goes to, and the directory the note names ("" for
		// no note).
		setup func(t *testing.T, r retiredEnvRepo) (extra []string, ledger, envDir string)
	}{
		{"--state-dir naming the variable's ledger", func(t *testing.T, r retiredEnvRepo) ([]string, string, string) {
			t.Setenv(retiredOrgStateDirEnv, r.oldLedger)
			return []string{"--state-dir", r.oldLedger}, r.oldLedger, r.oldLedger
		}},
		{"--state-dir naming another ledger", func(t *testing.T, r retiredEnvRepo) ([]string, string, string) {
			t.Setenv(retiredOrgStateDirEnv, r.oldLedger)
			other := filepath.Join(t.TempDir(), "other")
			return []string{"--state-dir", other}, other, r.oldLedger
		}},
		{"[org].state_dir naming another ledger", func(t *testing.T, r retiredEnvRepo) ([]string, string, string) {
			t.Setenv(retiredOrgStateDirEnv, r.oldLedger)
			other := filepath.Join(t.TempDir(), "other")
			t.Setenv(config.LocalConfigEnv, "")
			writeTestFile(t, filepath.Join(r.mainRoot, config.LocalConfigFileName), fmt.Sprintf("[org]\nstate_dir = %q\n", other))
			return nil, other, r.oldLedger
		}},
		{"the default ledger", func(t *testing.T, r retiredEnvRepo) ([]string, string, string) {
			t.Setenv(retiredOrgStateDirEnv, r.sharedDir)
			return nil, r.sharedDir, r.sharedDir
		}},
		{"the default ledger as a relative path", func(t *testing.T, r retiredEnvRepo) ([]string, string, string) {
			rel, err := filepath.Rel(r.worktree, r.sharedDir)
			if err != nil {
				t.Fatal(err)
			}
			t.Setenv(retiredOrgStateDirEnv, rel)
			return nil, r.sharedDir, r.sharedDir
		}},
		{"a blank value", func(t *testing.T, r retiredEnvRepo) ([]string, string, string) {
			t.Setenv(retiredOrgStateDirEnv, " \t")
			return nil, r.sharedDir, ""
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := newRetiredEnvRepo(t)
			extra, ledger, envDir := tc.setup(t, r)

			stdout, stderr, err := runRootCmdSplit(t, retiredEnvSpawnArgs(t, extra...)...)
			if err != nil || !strings.HasPrefix(stdout, `spawned seat "seat-1" `) {
				t.Fatalf("spawn = %v; want seat-1 spawned (stdout: %s stderr: %s)", err, stdout, stderr)
			}
			events := readManifestEvents(t, org.ManifestPathIn(ledger))
			if last := events[len(events)-1]; last.Event != org.EventSpawned || last.OrgID != "org-n" || last.SeatID != "seat-1" {
				t.Errorf("last event in %s = %+v; want org-n/seat-1 spawned", ledger, last)
			}
			if envDir == "" {
				if strings.Contains(stderr, retiredEnvNote) {
					t.Errorf("stderr = %q; want no note for a blank value", stderr)
				}
				return
			}
			checkRetiredEnvNote(t, "ralph org spawn", stderr, envDir, ledger, false)
		})
	}
}

// TestOrgRetiredStateDirEnv_EscalateStillReachesTheHuman: a leader runs
// `ralph org escalate` without --state-dir, so with RALPH_ORG_STATE_DIR
// naming another ledger than the default one the refusal leaves it with no
// ledger, and the item still reaches the human as not recorded, as for a
// personal file that cannot be read (S2b): exit 1, no ID, one banner naming
// the cause, one desktop notification, and no inbox or escalations file in
// either temp tree. With the variable naming the default ledger, the item
// is recorded there, with the note.
func TestOrgRetiredStateDirEnv_EscalateStillReachesTheHuman(t *testing.T) {
	r := newRetiredEnvRepo(t)
	t.Setenv(retiredOrgStateDirEnv, r.oldLedger)
	rec := recordDesktopNotify(t, nil)
	text := "TYPE: BLOCKED\nTASK_ID: org-a\n\nstuck"

	stdout, stderr, err := runRootCmdSplit(t, "org", "escalate", "--org-id", "org-a", "--text", text)
	cause := "ralph org escalate: cannot resolve the org ledger: RALPH_ORG_STATE_DIR names " + r.oldLedger +
		", which ralph no longer reads, and the ledger ralph finds by default is " + r.sharedDir
	if code := ExitCode(err); code != 1 || !errors.Is(err, errOrgLedgerUnresolved) ||
		!strings.Contains(err.Error(), "org: escalate: the item could not be recorded: "+cause) {
		t.Errorf("escalate = exit %d, %v; want exit 1, the item could not be recorded, naming %q", code, err, cause)
	}
	if stdout != "" {
		t.Errorf("stdout = %q; want no ID for an item that was not recorded", stdout)
	}
	banner := "ORG ESCALATION (NOT RECORDED): org=org-a type=BLOCKED -- the item is not in any inbox: " + cause
	if !strings.Contains(stderr, banner) || strings.Count(stderr, "ORG ESCALATION") != 1 {
		t.Errorf("stderr = %q; want one not-recorded banner with %q", stderr, banner)
	}
	if calls := rec.calls(); len(calls) != 1 || !strings.Contains(calls[0], "org-a") {
		t.Errorf("desktop notifications = %q; want one naming org-a", calls)
	}
	inboxName, escalationsName := filepath.Base(org.InboxPathIn(r.sharedDir)), filepath.Base(org.EscalationsPathIn(r.sharedDir))
	for _, root := range []string{filepath.Dir(r.mainRoot), r.oldLedger} {
		err := filepath.WalkDir(root, func(p string, d os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.Name() == inboxName || d.Name() == escalationsName {
				t.Errorf("escalate wrote %s; want no ledger file anywhere", p)
			}
			return nil
		})
		if err != nil {
			t.Fatalf("walk %s: %v", root, err)
		}
	}

	t.Setenv(retiredOrgStateDirEnv, r.sharedDir)
	setEscalateWatch(t, acceptingWatchLiveness())
	stdout, stderr, err = runRootCmdSplit(t, "org", "escalate", "--org-id", "org-a", "--text", text)
	if err != nil || stdout != "escalated e1 (org=org-a type=BLOCKED)\n" {
		t.Fatalf("escalate with the variable naming the default ledger = %q, %v; want e1 recorded (stderr: %s)", stdout, err, stderr)
	}
	if item := cliInboxItem(t, r.sharedDir, "e1"); item.OrgID != "org-a" {
		t.Errorf("e1 in %s = %+v; want org-a's item", r.sharedDir, item)
	}
	checkRetiredEnvNote(t, "ralph org escalate", stderr, r.sharedDir, r.sharedDir, false)
}

// TestOrgRetiredStateDirEnv_NamingTheLegacyLedgerRefusesStop: before ralph
// stopped reading RALPH_ORG_STATE_DIR, naming a linked worktree's older
// ledger with it skipped the legacy-ledger guard, as --state-dir does. Now
// the default ledger is the main worktree's, so `ralph org stop` is refused
// by the variable's check, before the guard (no legacy-ledger note), and
// the older ledger keeps its active seat with no herdr call.
func TestOrgRetiredStateDirEnv_NamingTheLegacyLedgerRefusesStop(t *testing.T) {
	herdrLog, _ := setupOrgStubPATH(t)
	r := newLegacyLedgerRepo(t)
	spawnLedgerSeat(t, r.legacyDir, "seat-1")
	before := readFileBytes(t, org.ManifestPathIn(r.legacyDir))
	herdrBefore := len(readLogLines(t, herdrLog))
	t.Setenv(retiredOrgStateDirEnv, r.legacyDir)

	_, stderr, err := runRootCmdSplit(t, "org", "stop", "--org-id", "org-a", "--seat", "seat-1")
	want := "ralph org stop: cannot resolve the org ledger: RALPH_ORG_STATE_DIR names " + r.legacyDir +
		", which ralph no longer reads, and the ledger ralph finds by default is " + r.sharedDir
	if code := ExitCode(err); code != 1 || !errors.Is(err, errOrgLedgerUnresolved) || !strings.Contains(err.Error(), want) {
		t.Errorf("stop = exit %d, %v; want exit 1 with %q", code, err, want)
	}
	if strings.Contains(stderr, legacyNoteMarker) {
		t.Errorf("stderr = %q; want no legacy-ledger note before the refusal", stderr)
	}
	if after := readFileBytes(t, org.ManifestPathIn(r.legacyDir)); !bytes.Equal(before, after) {
		t.Errorf("the older ledger changed:\nbefore:\n%s\nafter:\n%s", before, after)
	}
	if got := len(readLogLines(t, herdrLog)); got != herdrBefore {
		t.Errorf("herdr was called %d time(s) after the refusal; want 0", got-herdrBefore)
	}
}

// TestCheckRetiredEnv covers AC6 for doctor at the check: a warn for each
// retired variable set to a non-empty value, RALPH_ORG_AGMSG_HOME first,
// and none for one that is empty or unset. The RALPH_ORG_STATE_DIR line
// names the value, says ralph no longer reads it and that the verbs that
// change the ledger are refused while it names another one, and says where
// the value goes now.
func TestCheckRetiredEnv(t *testing.T) {
	getenv := func(env map[string]string) func(string) string {
		return func(name string) string { return env[name] }
	}
	stateDirRow := checkResult{
		Name:   "RALPH_ORG_STATE_DIR",
		Status: "warn",
		Detail: `set to "/old ledger", but ralph no longer reads it, and while it names another directory than the ledger ralph finds by default, ` +
			"the org verbs that change the ledger are refused; write the value in [org].state_dir of ralph.local.toml " +
			"(next to ralph.toml in the main worktree root, ignored by git), as an absolute path or one that starts with ~/, " +
			"or pass --state-dir, and unset RALPH_ORG_STATE_DIR",
	}
	for _, tc := range []struct {
		name string
		env  map[string]string
		want []string
	}{
		{"none set", nil, nil},
		{"both empty", map[string]string{"RALPH_ORG_AGMSG_HOME": "", "RALPH_ORG_STATE_DIR": ""}, nil},
		{"state dir only", map[string]string{"RALPH_ORG_STATE_DIR": "/old ledger"}, []string{"RALPH_ORG_STATE_DIR"}},
		{"both", map[string]string{"RALPH_ORG_STATE_DIR": "/old ledger", "RALPH_ORG_AGMSG_HOME": "/agmsg"}, []string{"RALPH_ORG_AGMSG_HOME", "RALPH_ORG_STATE_DIR"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := checkRetiredEnv(getenv(tc.env))
			var names []string
			for _, r := range got {
				names = append(names, r.Name)
				if r.Name == "RALPH_ORG_STATE_DIR" && r != stateDirRow {
					t.Errorf("RALPH_ORG_STATE_DIR row = %+v\nwant %+v", r, stateDirRow)
				}
			}
			if strings.Join(names, ",") != strings.Join(tc.want, ",") {
				t.Errorf("rows = %q; want %q", names, tc.want)
			}
		})
	}
}

// TestRunDoctorOpts_RetiredOrgStateDirEnv covers AC6 for doctor through
// runDoctorOpts on an initialized project with claude, codex, and go
// stubbed: with RALPH_ORG_STATE_DIR set, doctor prints its warn line and
// still exits 0; unset or empty, no line names it.
func TestRunDoctorOpts_RetiredOrgStateDirEnv(t *testing.T) {
	setupTestEmbedFS(t)
	Version = "0.1.0-test"
	dir := t.TempDir()
	if err := executeInit(dir, initConfig{ProjectName: "test", Packs: []string{"golang"}}, false); err != nil {
		t.Fatalf("init: %v", err)
	}
	binDir := filepath.Join(dir, "bin")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, bin := range []string{"claude", "codex", "go"} {
		writeStubBin(t, binDir, bin, "")
	}
	t.Setenv("PATH", binDir)
	runDoctor := func() (string, error) {
		t.Helper()
		origStdout := os.Stdout
		r, w, err := os.Pipe()
		if err != nil {
			t.Fatal(err)
		}
		os.Stdout = w
		runErr := runDoctorOpts(dir, false)
		_ = w.Close()
		os.Stdout = origStdout
		out, _ := io.ReadAll(r)
		return string(out), runErr
	}

	t.Run("set", func(t *testing.T) {
		t.Setenv(retiredOrgStateDirEnv, "/old ledger")
		out, err := runDoctor()
		if err != nil {
			t.Fatalf("a RALPH_ORG_STATE_DIR warning must not fail doctor: %v\noutput:\n%s", err, out)
		}
		if want := `⚠ RALPH_ORG_STATE_DIR: warn — set to "/old ledger", but ralph no longer reads it, and while it names another directory`; !strings.Contains(out, want) {
			t.Errorf("doctor output does not contain %q:\n%s", want, out)
		}
	})
	for _, tc := range []struct {
		name  string
		unset bool
	}{{"unset", true}, {"empty", false}} {
		t.Run(tc.name, func(t *testing.T) {
			setOrUnsetEnv(t, retiredOrgStateDirEnv, "", tc.unset)
			out, err := runDoctor()
			if err != nil {
				t.Fatalf("doctor: %v\noutput:\n%s", err, out)
			}
			if strings.Contains(out, retiredOrgStateDirEnv) {
				t.Errorf("doctor output names %s while it is %s:\n%s", retiredOrgStateDirEnv, tc.name, out)
			}
		})
	}
}
