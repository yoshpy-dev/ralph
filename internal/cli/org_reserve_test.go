package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/yoshpy-dev/ralph/internal/org"
)

// Tests for --reserve on `ralph org start` and `ralph org spawn`, the
// reservation line of `ralph org status`, and where spawn reads the org-wide
// limits from (plan docs/plans/active/2026-10-08-org-limits-reserve.md, the
// CLI side of AC5, AC7 and AC10, and AC13). A refusal reaches the operator as
// the returned error, which cmd/ralph prints on stderr before exiting 1.

// startOrgLeader runs `ralph org start` for orgID through the stub
// herdr/agmsg (setupOrgStubPATH), with extra flags before the task argument.
func startOrgLeader(t *testing.T, orgID string, extra ...string) (stdout, stderr string, err error) {
	t.Helper()
	args := append([]string{"start", "--org-id", orgID, "--driver", "claude", "--model", "sonnet", "--cwd", t.TempDir()}, extra...)
	return runOrgCmdStreams(t, append(args, "task text")...)
}

// scopeReservedEvents returns orgID's scope_reserved events in the manifest
// under stateDir, in manifest order.
func scopeReservedEvents(t *testing.T, stateDir, orgID string) []org.ManifestEvent {
	t.Helper()
	var reserved []org.ManifestEvent
	for _, ev := range readManifestEvents(t, org.ManifestPathIn(stateDir)) {
		if ev.OrgID == orgID && ev.Event == org.EventScopeReserved {
			reserved = append(reserved, ev)
		}
	}
	return reserved
}

func writeTestFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

// TestOrgStart_Reserve_RepeatedFlagRecordsOneReservation covers the
// repeatable flag: each --reserve value is one path, the paths are
// normalized into one org-level scope_reserved recorded before
// spawn_started, and the reservation alone satisfies the autonomous --scope
// gate (no --scope is passed).
func TestOrgStart_Reserve_RepeatedFlagRecordsOneReservation(t *testing.T) {
	setupOrgStubPATH(t)
	stateDir := filepath.Join(t.TempDir(), "state")

	_, stderr, err := startOrgLeader(t, "org-a", "--state-dir", stateDir,
		"--reserve", "internal/auth/", "--reserve", "docs/x.md", "--reserve", "./internal//auth/")
	if err != nil {
		t.Fatalf("start with --reserve: %v (stderr: %s)", err, stderr)
	}

	reserved := scopeReservedEvents(t, stateDir, "org-a")
	if len(reserved) != 1 {
		t.Fatalf("expected one scope_reserved for org-a, got %+v", reserved)
	}
	if got := reserved[0]; got.SeatID != "" || got.Details != "paths=docs/x.md,internal/auth/" || got.DryRun {
		t.Errorf("scope_reserved = %+v, want an org-level real event with Details %q", got, "paths=docs/x.md,internal/auth/")
	}
	events := readManifestEvents(t, org.ManifestPathIn(stateDir))
	if types := eventTypes(events); len(types) < 2 || types[0] != org.EventScopeReserved || types[1] != org.EventSpawnStarted {
		t.Errorf("expected scope_reserved right before spawn_started, got %v", types)
	}
	if last := events[len(events)-1]; last.Event != org.EventSpawned || last.SeatID != org.LeaderIdentity {
		t.Errorf("expected the leader spawned without --scope, last event: %+v", last)
	}
}

// TestOrgReserveFlag_ValueIsOnePath pins StringArray over StringSlice: a
// comma inside one --reserve value is not split into two paths, so the value
// is refused as a path with a comma, before anything is recorded.
func TestOrgReserveFlag_ValueIsOnePath(t *testing.T) {
	cases := map[string]func(t *testing.T, stateDir string) (string, string, error){
		"start": func(t *testing.T, stateDir string) (string, string, error) {
			return startOrgLeader(t, "org-a", "--state-dir", stateDir, "--reserve", "internal/auth/,docs/")
		},
		"spawn": func(t *testing.T, stateDir string) (string, string, error) {
			return runOrgCmdStreams(t, "spawn", "--org-id", "org-a", "--id", org.LeaderIdentity, "--role", org.LeaderIdentity,
				"--driver", "claude", "--model", "sonnet", "--cwd", t.TempDir(),
				"--state-dir", stateDir, "--reserve", "internal/auth/,docs/")
		},
	}
	for verb, run := range cases {
		t.Run(verb, func(t *testing.T) {
			herdrLog, _ := setupOrgStubPATH(t)
			stateDir := filepath.Join(t.TempDir(), "state")

			_, _, err := run(t, stateDir)
			if err == nil || !strings.Contains(err.Error(), `reserve path "internal/auth/,docs/" has a comma`) {
				t.Fatalf("expected the comma-containing value refused as one path, got: %v", err)
			}
			if events := readManifestEvents(t, org.ManifestPathIn(stateDir)); len(events) != 0 {
				t.Errorf("expected nothing recorded, got %v", eventTypes(events))
			}
			if lines := readLogLines(t, herdrLog); len(lines) != 0 {
				t.Errorf("expected no herdr call, got %v", lines)
			}
		})
	}
}

// TestOrgSpawn_Reserve_LeaderOnly covers `ralph org spawn --reserve`: the
// leader seat (--id leader, the session-promoted leader's own registration)
// records the reservation, and any other seat is refused before anything is
// recorded or called.
func TestOrgSpawn_Reserve_LeaderOnly(t *testing.T) {
	herdrLog, _ := setupOrgStubPATH(t)
	stateDir := filepath.Join(t.TempDir(), "state")

	stdout, stderr, err := runOrgCmdStreams(t, "spawn", "--org-id", "org-a", "--id", org.LeaderIdentity, "--role", org.LeaderIdentity,
		"--driver", "claude", "--model", "sonnet", "--cwd", t.TempDir(),
		"--state-dir", stateDir, "--reserve", "internal/")
	if err != nil {
		t.Fatalf("spawn --id leader --reserve: %v (stdout: %s stderr: %s)", err, stdout, stderr)
	}
	if reserved := scopeReservedEvents(t, stateDir, "org-a"); len(reserved) != 1 || reserved[0].Details != "paths=internal/" {
		t.Fatalf("expected org-a's scope_reserved paths=internal/, got %+v", reserved)
	}

	eventsBefore := len(readManifestEvents(t, org.ManifestPathIn(stateDir)))
	herdrBefore := len(readLogLines(t, herdrLog))
	stdout, _, err = runOrgCmdStreams(t, "spawn", "--org-id", "org-a", "--id", "seat-1", "--role", "worker",
		"--driver", "claude", "--model", "sonnet", "--cwd", t.TempDir(), "--scope", "test-scope",
		"--state-dir", stateDir, "--reserve", "docs/")
	if err == nil {
		t.Fatalf("expected a non-leader --reserve refused, stdout: %s", stdout)
	}
	if want := `only the leader seat reserves paths for its org; seat_id "seat-1" cannot`; !strings.Contains(err.Error(), want) {
		t.Errorf("expected the refusal to contain %q, got: %v", want, err)
	}
	if got := len(readManifestEvents(t, org.ManifestPathIn(stateDir))); got != eventsBefore {
		t.Errorf("expected no manifest event for the refusal, %d -> %d", eventsBefore, got)
	}
	if got := len(readLogLines(t, herdrLog)); got != herdrBefore {
		t.Errorf("expected no herdr call for the refusal, %d -> %d", herdrBefore, got)
	}
}

// TestOrgStart_Reserve_OverlapWithRunningOrgRejected is AC5 through the CLI:
// with org-a holding internal/auth/, org-b's start is refused for a file
// under it, a parent directory, and the whole repo, naming org-a and the
// overlapping paths, while internal/authz/ (a different path segment)
// passes.
func TestOrgStart_Reserve_OverlapWithRunningOrgRejected(t *testing.T) {
	setupOrgStubPATH(t)
	stateDir := filepath.Join(t.TempDir(), "state")
	if _, stderr, err := startOrgLeader(t, "org-a", "--state-dir", stateDir, "--reserve", "internal/auth/"); err != nil {
		t.Fatalf("start org-a: %v (stderr: %s)", err, stderr)
	}

	for _, path := range []string{"internal/auth/token.go", "internal/", "."} {
		t.Run(path, func(t *testing.T) {
			stdout, _, err := startOrgLeader(t, "org-b", "--state-dir", stateDir, "--reserve", path)
			if err == nil {
				t.Fatalf("expected org-b's start refused, stdout: %s", stdout)
			}
			if want := `running org_id "org-a": ` + path + " overlaps internal/auth/"; !strings.Contains(err.Error(), want) {
				t.Errorf("expected the refusal to contain %q, got: %v", want, err)
			}
			if !strings.HasPrefix(stdout, "rejected: ") {
				t.Errorf("expected the rejected line on stdout, got: %s", stdout)
			}
			if got := scopeReservedEvents(t, stateDir, "org-b"); len(got) != 0 {
				t.Errorf("expected no reservation for org-b, got %+v", got)
			}
			if last := lastSeatEvent(t, stateDir, "org-b", org.LeaderIdentity); last.Event != org.EventRejected {
				t.Errorf("expected org-b's leader recorded rejected, got %+v", last)
			}
		})
	}

	if _, stderr, err := startOrgLeader(t, "org-b", "--state-dir", stateDir, "--reserve", "internal/authz/"); err != nil {
		t.Fatalf("expected internal/authz/ not to overlap internal/auth/: %v (stderr: %s)", err, stderr)
	}
	if got := scopeReservedEvents(t, stateDir, "org-b"); len(got) != 1 || got[0].Details != "paths=internal/authz/" {
		t.Errorf("expected org-b's scope_reserved paths=internal/authz/, got %+v", got)
	}
}

// TestOrgStart_Reserve_SameListPassesDifferentListRejected is AC7 through the
// CLI: starting a running org again with the same paths (spelled
// differently) returns the existing leader with no new record, different
// paths are refused with the leader left as it is, and a start without
// --reserve still returns the leader.
func TestOrgStart_Reserve_SameListPassesDifferentListRejected(t *testing.T) {
	setupOrgStubPATH(t)
	stateDir := filepath.Join(t.TempDir(), "state")
	if _, stderr, err := startOrgLeader(t, "org-a", "--state-dir", stateDir, "--reserve", "internal/", "--reserve", "docs/"); err != nil {
		t.Fatalf("start org-a: %v (stderr: %s)", err, stderr)
	}

	stdout, stderr, err := startOrgLeader(t, "org-a", "--state-dir", stateDir, "--reserve", "docs/", "--reserve", "./internal/")
	if err != nil {
		t.Fatalf("expected the same paths to pass: %v (stderr: %s)", err, stderr)
	}
	if !strings.Contains(stdout, `seat "leader" already spawned`) {
		t.Errorf("expected the existing leader returned, stdout: %s", stdout)
	}
	if got := scopeReservedEvents(t, stateDir, "org-a"); len(got) != 1 {
		t.Errorf("expected still one scope_reserved, got %d", len(got))
	}

	_, _, err = startOrgLeader(t, "org-a", "--state-dir", stateDir, "--reserve", "docs/")
	if err == nil || !strings.Contains(err.Error(), `org_id "org-a" already reserves docs/,internal/`) {
		t.Fatalf("expected a different list refused, got: %v", err)
	}
	if last := lastSeatEvent(t, stateDir, "org-a", org.LeaderIdentity); last.Event != org.EventSpawned {
		t.Errorf("expected the leader left spawned, got %+v", last)
	}

	if _, stderr, err := startOrgLeader(t, "org-a", "--state-dir", stateDir); err != nil {
		t.Fatalf("expected a start without --reserve to return the leader: %v (stderr: %s)", err, stderr)
	}
	if got := scopeReservedEvents(t, stateDir, "org-a"); len(got) != 1 {
		t.Errorf("expected still one scope_reserved, got %d", len(got))
	}
}

// orgStatusPayload decodes `ralph org status --json` keeping the top-level
// keys, so a test can tell an omitted reservation from an empty one.
func orgStatusPayload(t *testing.T, stdout string) map[string]json.RawMessage {
	t.Helper()
	var payload map[string]json.RawMessage
	if err := json.Unmarshal([]byte(stdout), &payload); err != nil {
		t.Fatalf("status --json is not a JSON object: %v\n%s", err, stdout)
	}
	return payload
}

func payloadKeys(payload map[string]json.RawMessage) []string {
	keys := make([]string, 0, len(payload))
	for k := range payload {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	return keys
}

// TestOrgStatus_ShowsReservation is AC10: `ralph org status` prints the
// org's reservation as one `reserved:` line and as the JSON `reservation`
// array, and an org without one prints neither (its JSON keeps exactly the
// keys it had before reservations).
func TestOrgStatus_ShowsReservation(t *testing.T) {
	setupOrgStubPATH(t)
	stateDir := filepath.Join(t.TempDir(), "state")
	if _, stderr, err := startOrgLeader(t, "org-a", "--state-dir", stateDir, "--reserve", "internal/auth/", "--reserve", "docs/x.md"); err != nil {
		t.Fatalf("start org-a: %v (stderr: %s)", err, stderr)
	}
	if _, stderr, err := startOrgLeader(t, "org-b", "--state-dir", stateDir, "--scope", "org-b"); err != nil {
		t.Fatalf("start org-b: %v (stderr: %s)", err, stderr)
	}

	stdout, _, err := runOrgCmdStreams(t, "status", "--org-id", "org-a", "--state-dir", stateDir)
	if err != nil {
		t.Fatalf("status org-a: %v", err)
	}
	if !slices.Contains(outputLines(stdout), "reserved: docs/x.md, internal/auth/") {
		t.Errorf("expected the line %q in org-a's status, got:\n%s", "reserved: docs/x.md, internal/auth/", stdout)
	}

	stdout, _, err = runOrgCmdStreams(t, "status", "--org-id", "org-a", "--state-dir", stateDir, "--json")
	if err != nil {
		t.Fatalf("status org-a --json: %v", err)
	}
	var reservation []string
	if err := json.Unmarshal(orgStatusPayload(t, stdout)["reservation"], &reservation); err != nil ||
		!slices.Equal(reservation, []string{"docs/x.md", "internal/auth/"}) {
		t.Errorf("reservation = %q (err %v), want [docs/x.md internal/auth/]; stdout:\n%s", reservation, err, stdout)
	}

	stdout, _, err = runOrgCmdStreams(t, "status", "--org-id", "org-b", "--state-dir", stateDir)
	if err != nil {
		t.Fatalf("status org-b: %v", err)
	}
	if strings.Contains(stdout, "reserved:") {
		t.Errorf("expected no reserved line for org-b, got:\n%s", stdout)
	}
	stdout, _, err = runOrgCmdStreams(t, "status", "--org-id", "org-b", "--state-dir", stateDir, "--json")
	if err != nil {
		t.Fatalf("status org-b --json: %v", err)
	}
	if keys := payloadKeys(orgStatusPayload(t, stdout)); !slices.Equal(keys, []string{"corrupt_lines", "seats"}) {
		t.Errorf("org-b's status --json keys = %v, want [corrupt_lines seats]", keys)
	}
}

// TestOrgStatus_ReservationWithoutSeatsAndAfterDisband covers the edges of
// AC10: a reservation left by a spawn that failed afterwards shows next to
// "no seats", a disbanded org's earlier reservation and a dry-run
// reservation show nothing.
func TestOrgStatus_ReservationWithoutSeatsAndAfterDisband(t *testing.T) {
	t.Setenv("PATH", "")
	stateDir := t.TempDir()
	appendSeedEvent(t, stateDir, org.ManifestEvent{OrgID: "org-a", Event: org.EventScopeReserved, Details: "paths=internal/"})
	appendSeedEvent(t, stateDir, org.ManifestEvent{OrgID: "org-b", Event: org.EventScopeReserved, Details: "paths=docs/"})
	appendSeedEvent(t, stateDir, org.ManifestEvent{OrgID: "org-b", Event: org.EventDisbanded})
	appendSeedEvent(t, stateDir, org.ManifestEvent{OrgID: "org-c", Event: org.EventScopeReserved, Details: "paths=cmd/", DryRun: true})

	stdout, _, err := runOrgCmdStreams(t, "status", "--org-id", "org-a", "--state-dir", stateDir)
	if err != nil {
		t.Fatalf("status org-a: %v", err)
	}
	if got, want := outputLines(stdout), []string{"no seats", "reserved: internal/"}; !slices.Equal(got, want) {
		t.Errorf("org-a's status = %q, want %q", got, want)
	}
	stdout, _, err = runOrgCmdStreams(t, "status", "--org-id", "org-a", "--state-dir", stateDir, "--json")
	if err != nil {
		t.Fatalf("status org-a --json: %v", err)
	}
	if got := string(orgStatusPayload(t, stdout)["reservation"]); strings.Join(strings.Fields(got), "") != `["internal/"]` {
		t.Errorf("org-a's reservation = %s, want [\"internal/\"]", got)
	}

	for _, orgID := range []string{"org-b", "org-c"} {
		stdout, _, err := runOrgCmdStreams(t, "status", "--org-id", orgID, "--state-dir", stateDir)
		if err != nil {
			t.Fatalf("status %s: %v", orgID, err)
		}
		if got := outputLines(stdout); !slices.Equal(got, []string{"no seats"}) {
			t.Errorf("%s's status = %q, want only \"no seats\"", orgID, got)
		}
		stdout, _, err = runOrgCmdStreams(t, "status", "--org-id", orgID, "--state-dir", stateDir, "--json")
		if err != nil {
			t.Fatalf("status %s --json: %v", orgID, err)
		}
		if keys := payloadKeys(orgStatusPayload(t, stdout)); !slices.Equal(keys, []string{"corrupt_lines", "seats"}) {
			t.Errorf("%s's status --json keys = %v, want [corrupt_lines seats]", orgID, keys)
		}
	}
}

// newOrgLimitsRepo builds a repository with a linked worktree
// (newLegacyLedgerRepo), writes mainToml as the main checkout's ralph.toml,
// and starts org-a from the main checkout through the stub herdr/agmsg, so
// org-a runs in the ledger every worktree shares.
func newOrgLimitsRepo(t *testing.T, mainToml string) legacyLedgerRepo {
	t.Helper()
	setupOrgStubPATH(t)
	r := newLegacyLedgerRepo(t)
	writeTestFile(t, filepath.Join(r.mainRoot, "ralph.toml"), mainToml)
	t.Chdir(r.mainRoot)
	if _, stderr, err := startOrgLeader(t, "org-a", "--scope", "org-a"); err != nil {
		t.Fatalf("start org-a from the main checkout: %v (stderr: %s)", err, stderr)
	}
	return r
}

// TestOrgStart_OrgWideLimits_ReadFromMainWorktreeRalphToml is AC13: with
// max_orgs = 1 in the main checkout's ralph.toml and org-a running, a second
// org is refused from a subdirectory of the main checkout that has no
// ralph.toml and from a linked worktree whose own ralph.toml says
// max_orgs = 10. A --state-dir (absolute or relative) or RALPH_ORG_STATE_DIR
// naming the main checkout's ledger from a linked worktree whose ralph.toml
// says max_orgs = 99, as a feature org's leader passes it, is refused the
// same way. --config, and a --state-dir naming another ledger, keep using the
// caller's config, which allows it.
func TestOrgStart_OrgWideLimits_ReadFromMainWorktreeRalphToml(t *testing.T) {
	// worktreeAllowing99 makes the linked worktree, with max_orgs = 99 in its
	// own ralph.toml, the cwd.
	worktreeAllowing99 := func(t *testing.T, r legacyLedgerRepo) {
		t.Helper()
		writeTestFile(t, filepath.Join(r.worktree, "ralph.toml"), "[org]\nmax_orgs = 99\n")
		t.Chdir(r.worktree)
	}
	cases := []struct {
		name  string
		setup func(t *testing.T, r legacyLedgerRepo) (extra []string, ledger string) // ledger "" is the shared one
		// refused: org-b is refused by the main checkout's max_orgs = 1.
		refused bool
	}{
		{"main checkout subdirectory without ralph.toml", func(t *testing.T, r legacyLedgerRepo) ([]string, string) {
			sub := filepath.Join(r.mainRoot, "pkg", "sub")
			if err := os.MkdirAll(sub, 0o755); err != nil {
				t.Fatal(err)
			}
			t.Chdir(sub)
			return nil, ""
		}, true},
		{"linked worktree whose ralph.toml says max_orgs = 10", func(t *testing.T, r legacyLedgerRepo) ([]string, string) {
			writeTestFile(t, filepath.Join(r.worktree, "ralph.toml"), "[org]\nmax_orgs = 10\n")
			t.Chdir(r.worktree)
			return nil, ""
		}, true},
		{"--config with max_orgs = 10", func(t *testing.T, r legacyLedgerRepo) ([]string, string) {
			path := filepath.Join(t.TempDir(), "ralph.toml")
			writeTestFile(t, path, "[org]\nmax_orgs = 10\n")
			t.Chdir(r.worktree)
			return []string{"--config", path}, ""
		}, false},
		{"--state-dir naming the shared ledger", func(t *testing.T, r legacyLedgerRepo) ([]string, string) {
			worktreeAllowing99(t, r)
			return []string{"--state-dir", r.sharedDir}, ""
		}, true},
		{"relative --state-dir naming the shared ledger", func(t *testing.T, r legacyLedgerRepo) ([]string, string) {
			worktreeAllowing99(t, r)
			rel, err := filepath.Rel(r.worktree, r.sharedDir)
			if err != nil {
				t.Fatal(err)
			}
			return []string{"--state-dir", rel}, ""
		}, true},
		{"RALPH_ORG_STATE_DIR naming the shared ledger", func(t *testing.T, r legacyLedgerRepo) ([]string, string) {
			t.Setenv(org.EnvOrgStateDir, r.sharedDir)
			worktreeAllowing99(t, r)
			return nil, ""
		}, true},
		{"--state-dir naming another ledger with org-a running", func(t *testing.T, r legacyLedgerRepo) ([]string, string) {
			worktreeAllowing99(t, r)
			other := filepath.Join(t.TempDir(), "org")
			if _, stderr, err := startOrgLeader(t, "org-a", "--scope", "org-a", "--state-dir", other); err != nil {
				t.Fatalf("start org-a in %s: %v (stderr: %s)", other, err, stderr)
			}
			return []string{"--state-dir", other}, other
		}, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := newOrgLimitsRepo(t, "[org]\nmax_orgs = 1\n")
			extra, ledger := tc.setup(t, r)
			if ledger == "" {
				ledger = r.sharedDir
			}

			stdout, stderr, err := startOrgLeader(t, "org-b", append([]string{"--scope", "org-b"}, extra...)...)
			last := lastSeatEvent(t, ledger, "org-b", org.LeaderIdentity)
			if !tc.refused {
				if err != nil || last.Event != org.EventSpawned {
					t.Fatalf("expected org-b started under the caller's config, got err %v, last event %+v (stdout: %s stderr: %s)", err, last, stdout, stderr)
				}
				return
			}
			if err == nil {
				t.Fatalf("expected org-b refused by the main checkout's max_orgs = 1, stdout: %s", stdout)
			}
			for _, want := range []string{`max_orgs 1 reached: org_id "org-b" is not running and 1 orgs are (org-a)`, "ralph org disband --org-id"} {
				if !strings.Contains(err.Error(), want) {
					t.Errorf("expected the refusal to contain %q, got: %v", want, err)
				}
			}
			if last.Event != org.EventRejected {
				t.Errorf("expected org-b's leader recorded rejected in the shared ledger, got %+v", last)
			}
		})
	}
}

// TestOrgSpawnAndStartHelp_OrgWideLimitsSource: `ralph org spawn --help` and
// `ralph org start --help` say what withMainWorktreeOrgLimits does, that a
// --state-dir or RALPH_ORG_STATE_DIR naming the main worktree's ledger reads
// the main worktree's ralph.toml and one naming another ledger does not.
func TestOrgSpawnAndStartHelp_OrgWideLimitsSource(t *testing.T) {
	for _, verb := range []string{"spawn", "start"} {
		out, err := runOrgCmd(t, verb, "--help")
		if err != nil {
			t.Fatalf("%s --help: %v", verb, err)
		}
		help := strings.Join(strings.Fields(out), " ")
		for _, want := range []string{
			"the ledger is the main worktree's .harness/state/org, whether found by default or named with --state-dir or RALPH_ORG_STATE_DIR",
			"In every other case (--config, a --state-dir or RALPH_ORG_STATE_DIR naming another ledger,",
		} {
			if !strings.Contains(help, want) {
				t.Errorf("%s --help does not say %q:\n%s", verb, want, out)
			}
		}
	}
}

// TestOrgSpawn_MainWorktreeRalphToml_OnlyOrgWideLimitsTaken checks that only
// max_orgs and max_total_seats come from the main checkout's ralph.toml: its
// max_seats = 1 does not limit a spawn from a subdirectory without
// ralph.toml (built-in max_seats 5), while its max_total_seats = 1 does.
func TestOrgSpawn_MainWorktreeRalphToml_OnlyOrgWideLimitsTaken(t *testing.T) {
	spawnSeat1 := func(t *testing.T, r legacyLedgerRepo) error {
		t.Helper()
		sub := filepath.Join(r.mainRoot, "pkg")
		if err := os.MkdirAll(sub, 0o755); err != nil {
			t.Fatal(err)
		}
		t.Chdir(sub)
		_, _, err := runOrgCmdStreams(t, "spawn", "--org-id", "org-a", "--id", "seat-1", "--role", "worker",
			"--driver", "claude", "--model", "sonnet", "--cwd", t.TempDir(), "--scope", "test-scope")
		return err
	}

	t.Run("max_seats stays the caller's", func(t *testing.T) {
		r := newOrgLimitsRepo(t, "[org]\nmax_seats = 1\n")
		if err := spawnSeat1(t, r); err != nil {
			t.Fatalf("expected seat-1 to fit the built-in max_seats 5, got: %v", err)
		}
	})
	t.Run("max_total_seats comes from the main checkout", func(t *testing.T) {
		r := newOrgLimitsRepo(t, "[org]\nmax_total_seats = 1\n")
		err := spawnSeat1(t, r)
		if err == nil || !strings.Contains(err.Error(), "max_total_seats 1 reached: 1 seats are active across all orgs") {
			t.Fatalf("expected seat-1 refused by the main checkout's max_total_seats = 1, got: %v", err)
		}
	})
}

// TestOrgStart_MainWorktreeRalphTomlLoadError covers a main checkout
// ralph.toml that fails to load: a start from a linked worktree is refused
// with an error naming that file, not run with the built-in limits.
func TestOrgStart_MainWorktreeRalphTomlLoadError(t *testing.T) {
	setupOrgStubPATH(t)
	r := newLegacyLedgerRepo(t)
	mainToml := filepath.Join(r.mainRoot, "ralph.toml")
	writeTestFile(t, mainToml, "[org]\nmax_orgs = 0\n")

	_, _, err := startOrgLeader(t, "org-a", "--scope", "org-a")
	if err == nil {
		t.Fatal("expected start refused when the main checkout's ralph.toml fails to load")
	}
	for _, want := range []string{mainToml, "[org].max_orgs must be >= 1, got 0"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("expected the error to contain %q, got: %v", want, err)
		}
	}
	if events := readManifestEvents(t, org.ManifestPathIn(r.sharedDir)); len(events) != 0 {
		t.Errorf("expected nothing recorded, got %v", eventTypes(events))
	}
}
