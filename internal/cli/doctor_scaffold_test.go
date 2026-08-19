package cli

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/yoshpy-dev/ralph/internal/scaffold"
)

// findScaffoldResult returns the checkResult named name, failing the test if
// absent — every scenario below asserts against a specific FR-9 sub-check by
// name rather than by slice position, since checkScaffoldIntegrity's
// ordering is an implementation detail, not a contract.
func findScaffoldResult(t *testing.T, results []checkResult, name string) checkResult {
	t.Helper()
	for _, r := range results {
		if r.Name == name {
			return r
		}
	}
	t.Fatalf("no checkResult named %q in %+v", name, results)
	return checkResult{}
}

// assertAllScaffoldChecksPass is the AC-9 regression shape: exactly the five
// FR-9 sub-checks are present and every one reports "pass", both under
// --strict and without it — a converged (or freshly initialized) v2 project
// must never produce a false positive regardless of the flag.
func assertAllScaffoldChecksPass(t *testing.T, target string) {
	t.Helper()
	wantNames := []string{
		"Scaffold: core file hashes",
		"Scaffold: managed blocks",
		"Scaffold: settings.json owned keys",
		"Scaffold: conflict markers",
		"Scaffold: manifest/disk consistency",
	}
	for _, strict := range []bool{false, true} {
		results := checkScaffoldIntegrity(target, strict)
		if len(results) != len(wantNames) {
			t.Fatalf("strict=%v: got %d results, want %d: %+v", strict, len(results), len(wantNames), results)
		}
		for _, name := range wantNames {
			r := findScaffoldResult(t, results, name)
			if r.Status != "pass" {
				t.Errorf("strict=%v: %s: Status = %q, want \"pass\" (Detail: %s)", strict, name, r.Status, r.Detail)
			}
		}
	}
}

// TestCheckScaffoldIntegrity_FreshInit_AllPass is AC-9 handoff test 1: a
// fresh `ralph init` v2 project must be green on all five FR-9 sub-checks,
// under both --strict and without it.
func TestCheckScaffoldIntegrity_FreshInit_AllPass(t *testing.T) {
	target := initV2Project(t, gen1(), "1.0.0-test")
	assertAllScaffoldChecksPass(t, target)
}

// TestCheckScaffoldIntegrity_ConvergedUpgrade_AllPass is AC-9 handoff test 2:
// a project that has been cleanly upgraded to the current template
// generation (no user edits, no partial application) must also be green on
// all five checks — a completed `ralph upgrade` is not itself a scaffold
// integrity violation.
func TestCheckScaffoldIntegrity_ConvergedUpgrade_AllPass(t *testing.T) {
	target := initV2Project(t, gen1(), "1.0.0-test")

	scaffold.EmbeddedFS = gen2().build()
	Version = "1.1.0-test"

	var out, errOut bytes.Buffer
	if err := runUpgradeIOWithOptions(target, upgradeOptions{Pager: pagerNever}, strings.NewReader(""), &out, &errOut, false); err != nil {
		t.Fatalf("runUpgradeIOWithOptions: %v\nstderr:\n%s", err, errOut.String())
	}

	assertAllScaffoldChecksPass(t, target)
}

// TestCheckScaffoldIntegrity_EjectedFork_CoreHashesPass is AC-9 handoff test
// 3: ejecting a modified core path records it as owner=fork, and FR-9(a)
// must not flag it — plan.Drift never contains fork paths (they are
// classified into plan.Advisories instead), matching the spec's "fork 除く"
// carve-out. Every other check also stays green: the fork's content is
// still present on disk (satisfies (e)), is not a v2 exception face
// (irrelevant to (b)/(c)), and contains no conflict markers (satisfies (d)).
func TestCheckScaffoldIntegrity_EjectedFork_CoreHashesPass(t *testing.T) {
	target := initV2Project(t, gen1(), "1.0.0-test")

	writeMigrationDiskFile(t, target, "scripts/run-verify.sh", "#!/bin/sh\necho user-owned-now\n")

	var out bytes.Buffer
	if err := runEjectIO(target, "scripts/run-verify.sh", &out); err != nil {
		t.Fatalf("runEjectIO: %v", err)
	}

	assertAllScaffoldChecksPass(t, target)
}

// TestCheckScaffoldIntegrity_DriftedCoreFile_CoreHashesFail is AC-8 handoff
// test 4: a core path modified on disk with no fork record (unresolved
// drift) must fail FR-9(a) under --strict and warn without it.
func TestCheckScaffoldIntegrity_DriftedCoreFile_CoreHashesFail(t *testing.T) {
	target := initV2Project(t, gen1(), "1.0.0-test")
	writeMigrationDiskFile(t, target, "scripts/run-verify.sh", "#!/bin/sh\necho drifted-by-user\n")

	strictResults := checkScaffoldIntegrity(target, true)
	r := findScaffoldResult(t, strictResults, "Scaffold: core file hashes")
	if r.Status != "fail" {
		t.Errorf("strict: Status = %q, want \"fail\" (Detail: %s)", r.Status, r.Detail)
	}
	if !strings.Contains(r.Detail, "scripts/run-verify.sh") {
		t.Errorf("Detail must name the drifted path: %s", r.Detail)
	}

	warnResults := checkScaffoldIntegrity(target, false)
	r = findScaffoldResult(t, warnResults, "Scaffold: core file hashes")
	if r.Status != "warn" {
		t.Errorf("non-strict: Status = %q, want \"warn\" (Detail: %s)", r.Status, r.Detail)
	}
}

// TestRunDoctorFull_StrictFlipsExitCode_DriftedCore is the AC-8 exit-code
// integration pin: the same unresolved-drift project must exit 0 from
// runDoctorFull without --strict and non-zero (non-nil error) with it, while
// every other doctor check (claude/codex/go, herdr/agmsg) stays green/info
// via stubbed PATH entries so the drift finding is what determines the
// outcome.
func TestRunDoctorFull_StrictFlipsExitCode_DriftedCore(t *testing.T) {
	target := initV2Project(t, gen1(), "1.0.0-test")
	writeMigrationDiskFile(t, target, "scripts/run-verify.sh", "#!/bin/sh\necho drifted-by-user\n")

	binDir := filepath.Join(target, "bin")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, bin := range []string{"claude", "codex", "go"} {
		writeStubBin(t, binDir, bin, "")
	}
	t.Setenv("PATH", binDir)
	t.Setenv("RALPH_ORG_AGMSG_HOME", filepath.Join(target, "no-such-agmsg-home"))

	nonStrictErr := captureStdout(t, func() error { return runDoctorFull(target, false, false) })
	if nonStrictErr != nil {
		t.Errorf("non-strict: err = %v, want nil (drift is a warning without --strict)", nonStrictErr)
	}

	var strictOut string
	strictErr := captureStdoutText(t, &strictOut, func() error { return runDoctorFull(target, false, true) })
	if strictErr == nil {
		t.Fatalf("strict: err = nil, want non-nil (unresolved drift must fail --strict)\noutput:\n%s", strictOut)
	}
	if !strings.Contains(strictOut, "Scaffold: core file hashes: fail") {
		t.Errorf("expected a core-file-hashes fail line in strict output:\n%s", strictOut)
	}
}

// TestCheckScaffoldIntegrity_BrokenBlockMarker_ManagedBlocksFail is AC-8
// handoff test 5: deleting AGENTS.md's END marker line leaves an
// unmatched BEGIN marker, which upgrade.UpdateManagedBlockStyled classifies
// as BlockMalformed — FR-9(b) must fail under --strict.
func TestCheckScaffoldIntegrity_BrokenBlockMarker_ManagedBlocksFail(t *testing.T) {
	target := initV2Project(t, gen1(), "1.0.0-test")

	agentsPath := filepath.Join(target, "AGENTS.md")
	content := mustReadFile(t, agentsPath)
	mutated := bytes.Replace(content, []byte("<!-- END RALPH MANAGED -->\n"), []byte(""), 1)
	if bytes.Equal(mutated, content) {
		t.Fatal("test setup: END marker line was not found/removed")
	}
	if err := os.WriteFile(agentsPath, mutated, 0644); err != nil {
		t.Fatalf("write AGENTS.md: %v", err)
	}

	results := checkScaffoldIntegrity(target, true)
	r := findScaffoldResult(t, results, "Scaffold: managed blocks")
	if r.Status != "fail" {
		t.Errorf("Status = %q, want \"fail\" (Detail: %s)", r.Status, r.Detail)
	}
	if !strings.Contains(r.Detail, "AGENTS.md") {
		t.Errorf("Detail must name AGENTS.md: %s", r.Detail)
	}
}

// TestCheckScaffoldIntegrity_BlockContentMutated_ManagedBlocksFail is AC-8
// handoff test 6: editing content strictly inside AGENTS.md's well-formed
// marker pair (markers untouched) makes the block's interior diverge from
// the expected managed content — BlockUpdated, not BlockUnchanged — which
// FR-9(b) must also treat as a violation.
func TestCheckScaffoldIntegrity_BlockContentMutated_ManagedBlocksFail(t *testing.T) {
	target := initV2Project(t, gen1(), "1.0.0-test")

	agentsPath := filepath.Join(target, "AGENTS.md")
	content := mustReadFile(t, agentsPath)
	mutated := bytes.Replace(content, []byte("Old mission text.\n"), []byte("Mutated mission text by user.\n"), 1)
	if bytes.Equal(mutated, content) {
		t.Fatal("test setup: managed interior text was not found/replaced")
	}
	if err := os.WriteFile(agentsPath, mutated, 0644); err != nil {
		t.Fatalf("write AGENTS.md: %v", err)
	}

	results := checkScaffoldIntegrity(target, true)
	r := findScaffoldResult(t, results, "Scaffold: managed blocks")
	if r.Status != "fail" {
		t.Errorf("Status = %q, want \"fail\" (Detail: %s)", r.Status, r.Detail)
	}
	if !strings.Contains(r.Detail, "AGENTS.md") {
		t.Errorf("Detail must name AGENTS.md: %s", r.Detail)
	}
}

// TestCheckScaffoldIntegrity_SettingsOwnedKeyRemoved_Fails is AC-8 handoff
// test 7: removing a ralph-owned settings.json entry (here,
// permissions.allow's template-shipped "Bash(git status:*)" — gen1's
// fixture settings.json does not include a "hooks" key at all, so this
// covers the same owned-array-pruning mechanism the handoff's "delete hooks
// key" scenario exercises) makes upgrade.MergeOwnedSettings report
// Changed=true, since the merge would re-add the missing template entry.
func TestCheckScaffoldIntegrity_SettingsOwnedKeyRemoved_Fails(t *testing.T) {
	target := initV2Project(t, gen1(), "1.0.0-test")

	settingsPath := filepath.Join(target, ".claude", "settings.json")
	mutated := []byte(`{
  "customUserSetting": true,
  "env": {
    "FOO": "v1"
  },
  "permissions": {
    "allow": [
      "Bash(old-owned:*)"
    ]
  }
}
`)
	if err := os.WriteFile(settingsPath, mutated, 0644); err != nil {
		t.Fatalf("write settings.json: %v", err)
	}

	results := checkScaffoldIntegrity(target, true)
	r := findScaffoldResult(t, results, "Scaffold: settings.json owned keys")
	if r.Status != "fail" {
		t.Errorf("Status = %q, want \"fail\" (Detail: %s)", r.Status, r.Detail)
	}
}

// TestCheckScaffoldIntegrity_ConflictMarkers_Fails is AC-8 handoff test 8:
// unresolved git conflict markers written into a manifest-tracked file must
// fail FR-9(d).
func TestCheckScaffoldIntegrity_ConflictMarkers_Fails(t *testing.T) {
	target := initV2Project(t, gen1(), "1.0.0-test")

	notesPath := filepath.Join(target, "docs", "notes.md")
	conflicted := "<<<<<<< HEAD\nmine\n=======\ntheirs\n>>>>>>> branch\n"
	if err := os.WriteFile(notesPath, []byte(conflicted), 0644); err != nil {
		t.Fatalf("write docs/notes.md: %v", err)
	}

	results := checkScaffoldIntegrity(target, true)
	r := findScaffoldResult(t, results, "Scaffold: conflict markers")
	if r.Status != "fail" {
		t.Errorf("Status = %q, want \"fail\" (Detail: %s)", r.Status, r.Detail)
	}
	if !strings.Contains(r.Detail, "docs/notes.md") {
		t.Errorf("Detail must name docs/notes.md: %s", r.Detail)
	}
}

// TestCheckScaffoldIntegrity_UnreadableTrackedFile_StrictFails pins the
// gate-level guarantee for a manifest-tracked file that exists but cannot
// be read (permission denied): SOME strict-eligible check must fail — in
// the current flow the ownership-planning pass reads the path first and
// fails there; the conflict-marker scan's own unreadable-file branch is
// defense in depth behind it (cycle-3 warn-path audit alongside AR#1/AR#2,
// cycle 2, docs/reports/cross-review-triage-overlay-scaffold-v2-p5.md).
// Asserting at the gate level (any fail naming the path) rather than on a
// specific check name keeps the pin valid if the internal check ordering
// changes.
func TestCheckScaffoldIntegrity_UnreadableTrackedFile_StrictFails(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("running as root: chmod 0000 does not make files unreadable")
	}
	target := initV2Project(t, gen1(), "1.0.0-test")

	notesPath := filepath.Join(target, "docs", "notes.md")
	if err := os.Chmod(notesPath, 0000); err != nil {
		t.Fatalf("chmod docs/notes.md: %v", err)
	}
	t.Cleanup(func() { _ = os.Chmod(notesPath, 0644) })

	strictResults := checkScaffoldIntegrity(target, true)
	foundFail := false
	for _, r := range strictResults {
		if r.Status == "fail" && strings.Contains(r.Detail, "docs/notes.md") {
			foundFail = true
		}
	}
	if !foundFail {
		t.Errorf("strict: want at least one fail result naming docs/notes.md, got %+v", strictResults)
	}

	nonStrict := checkScaffoldIntegrity(target, false)
	for _, r := range nonStrict {
		if r.Status == "fail" {
			t.Errorf("non-strict: no result may be fail, got %+v", r)
		}
	}
}

// TestCheckScaffoldIntegrity_ManifestTrackedFileDeleted_Fails is AC-8
// handoff test 9: a manifest-tracked path (docs/notes.md, owner=seed) that
// is deleted from disk must fail FR-9(e).
func TestCheckScaffoldIntegrity_ManifestTrackedFileDeleted_Fails(t *testing.T) {
	target := initV2Project(t, gen1(), "1.0.0-test")

	notesPath := filepath.Join(target, "docs", "notes.md")
	if err := os.Remove(notesPath); err != nil {
		t.Fatalf("remove docs/notes.md: %v", err)
	}

	results := checkScaffoldIntegrity(target, true)
	r := findScaffoldResult(t, results, "Scaffold: manifest/disk consistency")
	if r.Status != "fail" {
		t.Errorf("Status = %q, want \"fail\" (Detail: %s)", r.Status, r.Detail)
	}
	if !strings.Contains(r.Detail, "docs/notes.md") {
		t.Errorf("Detail must name docs/notes.md: %s", r.Detail)
	}
}

// TestCheckScaffoldIntegrity_LegacyManifest_InfoNeverFailsStrict is AC-9
// handoff test 10: a legacy (pre-v2) manifest is not itself an FR-9
// violation — checkScaffoldIntegrity must return exactly one "info" result
// advising `ralph upgrade`, never "fail", even under --strict.
func TestCheckScaffoldIntegrity_LegacyManifest_InfoNeverFailsStrict(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, ".ralph"), 0755); err != nil {
		t.Fatalf("MkdirAll .ralph: %v", err)
	}
	legacy := scaffold.NewManifest("0.9.0-test")
	legacy.SetFile("AGENTS.md", "sha256:legacy")
	if err := legacy.Write(filepath.Join(dir, ".ralph", "manifest.toml")); err != nil {
		t.Fatalf("writing legacy manifest: %v", err)
	}

	for _, strict := range []bool{false, true} {
		results := checkScaffoldIntegrity(dir, strict)
		if len(results) != 1 {
			t.Fatalf("strict=%v: got %d results, want 1: %+v", strict, len(results), results)
		}
		if results[0].Status != "info" {
			t.Errorf("strict=%v: Status = %q, want \"info\" (Detail: %s)", strict, results[0].Status, results[0].Detail)
		}
		if !strings.Contains(results[0].Detail, "ralph upgrade") {
			t.Errorf("strict=%v: Detail must point at `ralph upgrade`: %s", strict, results[0].Detail)
		}
	}
}

// TestCheckScaffoldIntegrity_NoManifest_ReturnsNil confirms a non-project
// directory (no .ralph/manifest.toml at all) produces zero scaffold
// results, so pre-existing doctor behavior for non-ralph directories is
// unaffected.
func TestCheckScaffoldIntegrity_NoManifest_ReturnsNil(t *testing.T) {
	dir := t.TempDir()
	results := checkScaffoldIntegrity(dir, true)
	if results != nil {
		t.Errorf("results = %+v, want nil", results)
	}
}

// TestCheckScaffoldIntegrity_CorruptManifest_StrictFails is the M2
// regression guard: a `.ralph/manifest.toml` that exists but fails to
// parse (corrupt TOML) must NOT be treated the same as "no manifest at
// all" -- it is a strict-eligible violation in its own right. Before the
// fix, checkScaffoldIntegrity returned nil (zero results) for any
// ReadManifest error, which made --strict exit 0 on a corrupted manifest:
// the integrity gate failed open exactly when the manifest was least
// trustworthy.
func TestCheckScaffoldIntegrity_CorruptManifest_StrictFails(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, ".ralph"), 0o755); err != nil {
		t.Fatalf("MkdirAll .ralph: %v", err)
	}
	corrupt := []byte("this is not valid TOML: [[[\n")
	if err := os.WriteFile(filepath.Join(dir, ".ralph", "manifest.toml"), corrupt, 0o644); err != nil {
		t.Fatalf("writing corrupt manifest.toml: %v", err)
	}

	strictResults := checkScaffoldIntegrity(dir, true)
	if len(strictResults) != 1 {
		t.Fatalf("strict: got %d results, want 1: %+v", len(strictResults), strictResults)
	}
	if strictResults[0].Status != "fail" {
		t.Errorf("strict: Status = %q, want \"fail\" (Detail: %s)", strictResults[0].Status, strictResults[0].Detail)
	}

	warnResults := checkScaffoldIntegrity(dir, false)
	if len(warnResults) != 1 {
		t.Fatalf("non-strict: got %d results, want 1: %+v", len(warnResults), warnResults)
	}
	if warnResults[0].Status != "warn" {
		t.Errorf("non-strict: Status = %q, want \"warn\" (Detail: %s)", warnResults[0].Status, warnResults[0].Detail)
	}

	// Integration pin: runDoctorFull's exit code must actually flip under
	// --strict for a corrupt manifest, the same AC-8 shape
	// TestRunDoctorFull_StrictFlipsExitCode_DriftedCore pins for drift.
	binDir := filepath.Join(dir, "bin")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, bin := range []string{"claude", "codex", "go"} {
		writeStubBin(t, binDir, bin, "")
	}
	t.Setenv("PATH", binDir)
	t.Setenv("RALPH_ORG_AGMSG_HOME", filepath.Join(dir, "no-such-agmsg-home"))

	nonStrictErr := captureStdout(t, func() error { return runDoctorFull(dir, false, false) })
	if nonStrictErr != nil {
		t.Errorf("non-strict: err = %v, want nil (a corrupt manifest is a warning without --strict)", nonStrictErr)
	}
	strictErr := captureStdout(t, func() error { return runDoctorFull(dir, false, true) })
	if strictErr == nil {
		t.Fatal("strict: err = nil, want non-nil (a corrupt manifest must fail --strict)")
	}
}

// TestCheckScaffoldIntegrity_OwnershipPlanningError_StrictFails is the AR#1
// (cycle 1, docs/reports/cross-review-triage-overlay-scaffold-v2-p5.md)
// regression guard: when resolveOwnershipPlan itself fails (here, a
// manifest-tracked core path replaced by a directory, so
// PlanCoreReplaceDesired's disk read errors "is a directory"),
// checkScaffoldIntegrity must report a strict-eligible violation naming the
// underlying error, not the pre-fix hardcoded "warn" that let a corrupted
// scaffold pass --strict silently.
func TestCheckScaffoldIntegrity_OwnershipPlanningError_StrictFails(t *testing.T) {
	target := initV2Project(t, gen1(), "1.0.0-test")

	verifyPath := filepath.Join(target, "scripts", "run-verify.sh")
	if err := os.Remove(verifyPath); err != nil {
		t.Fatalf("removing scripts/run-verify.sh: %v", err)
	}
	if err := os.Mkdir(verifyPath, 0o755); err != nil {
		t.Fatalf("replacing scripts/run-verify.sh with a directory: %v", err)
	}

	strictResults := checkScaffoldIntegrity(target, true)
	if len(strictResults) != 1 {
		t.Fatalf("strict: got %d results, want 1: %+v", len(strictResults), strictResults)
	}
	if strictResults[0].Status != "fail" {
		t.Errorf("strict: Status = %q, want \"fail\" (Detail: %s)", strictResults[0].Status, strictResults[0].Detail)
	}
	if !strings.Contains(strictResults[0].Detail, "scripts/run-verify.sh") {
		t.Errorf("Detail must name the unreadable path: %s", strictResults[0].Detail)
	}

	warnResults := checkScaffoldIntegrity(target, false)
	if len(warnResults) != 1 {
		t.Fatalf("non-strict: got %d results, want 1: %+v", len(warnResults), warnResults)
	}
	if warnResults[0].Status != "warn" {
		t.Errorf("non-strict: Status = %q, want \"warn\" (Detail: %s)", warnResults[0].Status, warnResults[0].Detail)
	}

	// Integration pin, mirroring TestCheckScaffoldIntegrity_CorruptManifest_StrictFails.
	binDir := filepath.Join(target, "bin")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, bin := range []string{"claude", "codex", "go"} {
		writeStubBin(t, binDir, bin, "")
	}
	t.Setenv("PATH", binDir)
	t.Setenv("RALPH_ORG_AGMSG_HOME", filepath.Join(target, "no-such-agmsg-home"))

	nonStrictErr := captureStdout(t, func() error { return runDoctorFull(target, false, false) })
	if nonStrictErr != nil {
		t.Errorf("non-strict: err = %v, want nil (a planning error is a warning without --strict)", nonStrictErr)
	}
	strictErr := captureStdout(t, func() error { return runDoctorFull(target, false, true) })
	if strictErr == nil {
		t.Fatal("strict: err = nil, want non-nil (an ownership-planning error must fail --strict)")
	}
}

// TestCheckScaffoldIntegrity_SettingsSnapshotDeleted_StrictFails is the AR#2
// (cycle 1, docs/reports/cross-review-triage-overlay-scaffold-v2-p5.md)
// regression guard: deleting the manifest-tracked settings snapshot
// (.ralph/core/settings.ralph.json) while leaving its manifest entry in
// place must fail FR-9(e) under --strict. Before the fix, the snapshot was
// excluded from checkManifestConsistency's existence sweep via the full
// v2SkipPaths() set, and checkSettingsOwnedKeys treats a missing snapshot as
// a legitimate "{}" oldOwned fallback rather than a violation — so deleting
// the snapshot passed both checks and --strict exited 0 on a manifest/disk
// mismatch.
func TestCheckScaffoldIntegrity_SettingsSnapshotDeleted_StrictFails(t *testing.T) {
	target := initV2Project(t, gen1(), "1.0.0-test")

	snapshotPath := filepath.Join(target, ".ralph", "core", "settings.ralph.json")
	if _, err := os.Stat(snapshotPath); err != nil {
		t.Fatalf("test setup: settings snapshot not found at %s: %v", snapshotPath, err)
	}
	if err := os.Remove(snapshotPath); err != nil {
		t.Fatalf("removing settings snapshot: %v", err)
	}

	results := checkScaffoldIntegrity(target, true)
	r := findScaffoldResult(t, results, "Scaffold: manifest/disk consistency")
	if r.Status != "fail" {
		t.Errorf("Status = %q, want \"fail\" (Detail: %s)", r.Status, r.Detail)
	}
	if !strings.Contains(r.Detail, ".ralph/core/settings.ralph.json") {
		t.Errorf("Detail must name the deleted snapshot path: %s", r.Detail)
	}

	// checkSettingsOwnedKeys's "{}" fallback for a missing snapshot means it
	// must NOT independently flag the deletion — confirming FR-9(e) is the
	// check actually carrying this regression, not FR-9(c) by accident.
	ownedKeys := findScaffoldResult(t, results, "Scaffold: settings.json owned keys")
	if ownedKeys.Status != "pass" {
		t.Errorf("Scaffold: settings.json owned keys: Status = %q, want \"pass\" (a missing snapshot falls back to \"{}\" oldOwned, not a violation) (Detail: %s)", ownedKeys.Status, ownedKeys.Detail)
	}

	nonStrictResults := checkScaffoldIntegrity(target, false)
	r = findScaffoldResult(t, nonStrictResults, "Scaffold: manifest/disk consistency")
	if r.Status != "warn" {
		t.Errorf("non-strict: Status = %q, want \"warn\" (Detail: %s)", r.Status, r.Detail)
	}
}

// TestCheckScaffoldIntegrity_SettingsJSONDeleted_OwnedKeysCatchesIt is the
// exception-face audit behind AR#2's narrowed checkManifestConsistency skip
// set: settings.json (v2SettingsPath) stays excluded from FR-9(e)'s
// existence sweep because FR-9(c) (checkSettingsOwnedKeys) already fails
// when the file is missing entirely — readFinalDiskContent returns nil for
// an absent file, and MergeOwnedSettings treats that nil as "{}", which
// diverges from a non-empty owned template (env/permissions/hooks).
func TestCheckScaffoldIntegrity_SettingsJSONDeleted_OwnedKeysCatchesIt(t *testing.T) {
	target := initV2Project(t, gen1(), "1.0.0-test")

	settingsPath := filepath.Join(target, ".claude", "settings.json")
	if err := os.Remove(settingsPath); err != nil {
		t.Fatalf("removing settings.json: %v", err)
	}

	results := checkScaffoldIntegrity(target, true)
	r := findScaffoldResult(t, results, "Scaffold: settings.json owned keys")
	if r.Status != "fail" {
		t.Errorf("Status = %q, want \"fail\" (Detail: %s)", r.Status, r.Detail)
	}

	consistency := findScaffoldResult(t, results, "Scaffold: manifest/disk consistency")
	if consistency.Status != "pass" {
		t.Errorf("Scaffold: manifest/disk consistency: Status = %q, want \"pass\" (settings.json stays excluded from FR-9(e); FR-9(c) already caught it) (Detail: %s)", consistency.Status, consistency.Detail)
	}
}

// TestCheckScaffoldIntegrity_AgentsMdDeleted_ManagedBlocksCatchesIt is the
// AGENTS.md counterpart of the settings.json audit above: a missing
// managed-block surface stays excluded from FR-9(e) because FR-9(b)
// (checkManagedBlocks) already fails on it — updateOneBlockV2 returns a
// zero-value, ok=false outcome for a missing surface, which
// checkManagedBlocks treats as an offender.
func TestCheckScaffoldIntegrity_AgentsMdDeleted_ManagedBlocksCatchesIt(t *testing.T) {
	target := initV2Project(t, gen1(), "1.0.0-test")

	agentsPath := filepath.Join(target, "AGENTS.md")
	if err := os.Remove(agentsPath); err != nil {
		t.Fatalf("removing AGENTS.md: %v", err)
	}

	results := checkScaffoldIntegrity(target, true)
	r := findScaffoldResult(t, results, "Scaffold: managed blocks")
	if r.Status != "fail" {
		t.Errorf("Status = %q, want \"fail\" (Detail: %s)", r.Status, r.Detail)
	}

	consistency := findScaffoldResult(t, results, "Scaffold: manifest/disk consistency")
	if consistency.Status != "pass" {
		t.Errorf("Scaffold: manifest/disk consistency: Status = %q, want \"pass\" (AGENTS.md stays excluded from FR-9(e); FR-9(b) already caught it) (Detail: %s)", consistency.Status, consistency.Detail)
	}
}

// TestCheckScaffoldIntegrity_SettingsJSONInvalidJSON_StrictFails is the AR#1
// (cycle 2, docs/reports/cross-review-triage-overlay-scaffold-v2-p5.md)
// regression guard: an unparseable .claude/settings.json makes
// upgrade.MergeOwnedSettings itself fail (parseSettingsDoc rejects invalid
// JSON), and that failure must be a strict-eligible violation, not the
// pre-fix hardcoded "warn" that let --strict exit 0 while FR-9(c) could not
// even be evaluated.
func TestCheckScaffoldIntegrity_SettingsJSONInvalidJSON_StrictFails(t *testing.T) {
	target := initV2Project(t, gen1(), "1.0.0-test")

	settingsPath := filepath.Join(target, ".claude", "settings.json")
	if err := os.WriteFile(settingsPath, []byte("this is not valid JSON: [[[\n"), 0o644); err != nil {
		t.Fatalf("writing invalid settings.json: %v", err)
	}

	strictResults := checkScaffoldIntegrity(target, true)
	r := findScaffoldResult(t, strictResults, "Scaffold: settings.json owned keys")
	if r.Status != "fail" {
		t.Errorf("strict: Status = %q, want \"fail\" (Detail: %s)", r.Status, r.Detail)
	}
	if !strings.Contains(r.Detail, v2SettingsPath) {
		t.Errorf("Detail must name %s: %s", v2SettingsPath, r.Detail)
	}

	nonStrictResults := checkScaffoldIntegrity(target, false)
	r = findScaffoldResult(t, nonStrictResults, "Scaffold: settings.json owned keys")
	if r.Status != "warn" {
		t.Errorf("non-strict: Status = %q, want \"warn\" (Detail: %s)", r.Status, r.Detail)
	}

	// No runDoctorFull integration pin here (unlike the sibling AR fixes
	// above): checkHooks (doctor.go, an unrelated pre-existing check) also
	// reads .claude/settings.json via encoding/json and reports a hardcoded
	// "fail" -- not gated by --strict at all -- for ANY invalid JSON there,
	// so doctor's overall exit code would already be non-zero without
	// --strict regardless of this fix, confounding what a pin would prove.
	// TestCheckScaffoldIntegrity_SettingsSnapshotCorrupt_StrictFails below
	// exercises the identical MergeOwnedSettings error branch via the
	// settings *snapshot* instead, which checkHooks never reads, and carries
	// the doctor-level integration pin instead.
}

// TestCheckScaffoldIntegrity_SettingsSnapshotCorrupt_StrictFails is AR#1's
// doctor-level integration pin, isolated from checkHooks' unrelated
// unconditional-fail-on-invalid-JSON behavior (see the comment on
// TestCheckScaffoldIntegrity_SettingsJSONInvalidJSON_StrictFails above):
// corrupting the settings *snapshot* (oldOwned) drives MergeOwnedSettings
// into the exact same error branch as a corrupt settings.json (current)
// would, without checkHooks ever touching that file.
func TestCheckScaffoldIntegrity_SettingsSnapshotCorrupt_StrictFails(t *testing.T) {
	target := initV2Project(t, gen1(), "1.0.0-test")

	snapshotPath := filepath.Join(target, ".ralph", "core", "settings.ralph.json")
	if err := os.WriteFile(snapshotPath, []byte("this is not valid JSON: [[[\n"), 0o644); err != nil {
		t.Fatalf("writing invalid settings snapshot: %v", err)
	}

	strictResults := checkScaffoldIntegrity(target, true)
	r := findScaffoldResult(t, strictResults, "Scaffold: settings.json owned keys")
	if r.Status != "fail" {
		t.Errorf("strict: Status = %q, want \"fail\" (Detail: %s)", r.Status, r.Detail)
	}
	if !strings.Contains(r.Detail, "merging "+v2SettingsPath) {
		t.Errorf("Detail must name the merge failure: %s", r.Detail)
	}

	nonStrictResults := checkScaffoldIntegrity(target, false)
	r = findScaffoldResult(t, nonStrictResults, "Scaffold: settings.json owned keys")
	if r.Status != "warn" {
		t.Errorf("non-strict: Status = %q, want \"warn\" (Detail: %s)", r.Status, r.Detail)
	}

	// Integration pin, mirroring TestCheckScaffoldIntegrity_CorruptManifest_StrictFails.
	binDir := filepath.Join(target, "bin")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, bin := range []string{"claude", "codex", "go"} {
		writeStubBin(t, binDir, bin, "")
	}
	t.Setenv("PATH", binDir)
	t.Setenv("RALPH_ORG_AGMSG_HOME", filepath.Join(target, "no-such-agmsg-home"))

	nonStrictErr := captureStdout(t, func() error { return runDoctorFull(target, false, false) })
	if nonStrictErr != nil {
		t.Errorf("non-strict: err = %v, want nil (a corrupt settings snapshot is a warning without --strict)", nonStrictErr)
	}
	strictErr := captureStdout(t, func() error { return runDoctorFull(target, false, true) })
	if strictErr == nil {
		t.Fatal("strict: err = nil, want non-nil (a corrupt settings snapshot must fail --strict)")
	}
}

// TestCheckScaffoldIntegrity_GitignoreTemplateMarkersBroken_ManagedBlocksStrictFails
// is the AR#2 (cycle 2,
// docs/reports/cross-review-triage-overlay-scaffold-v2-p5.md) regression
// guard: applyBlockUpdatesV2 itself returning a genuine error (as opposed to
// a malformed/missing-surface outcome, which is already covered by
// TestCheckScaffoldIntegrity_AgentsMdDeleted_ManagedBlocksCatchesIt and the
// pre-existing "!oc.ok" offender logic) must be a strict-eligible violation.
// The current embedded .gitignore template is swapped for one missing its
// managed-block BEGIN/END markers entirely, which makes
// extractBlockInterior fail inside applyBlockUpdatesV2's .gitignore branch
// -- a genuine err != nil return, not an outcome.
func TestCheckScaffoldIntegrity_GitignoreTemplateMarkersBroken_ManagedBlocksStrictFails(t *testing.T) {
	target := initV2Project(t, gen1(), "1.0.0-test")

	broken := gen1().build()
	broken["templates/base/.gitignore"] = &fstest.MapFile{Data: []byte("# no managed block markers at all\n")}
	scaffold.EmbeddedFS = broken

	strictResults := checkScaffoldIntegrity(target, true)
	r := findScaffoldResult(t, strictResults, "Scaffold: managed blocks")
	if r.Status != "fail" {
		t.Errorf("strict: Status = %q, want \"fail\" (Detail: %s)", r.Status, r.Detail)
	}
	if !strings.Contains(r.Detail, "computing managed block outcomes") {
		t.Errorf("Detail must name the computation failure: %s", r.Detail)
	}

	nonStrictResults := checkScaffoldIntegrity(target, false)
	r = findScaffoldResult(t, nonStrictResults, "Scaffold: managed blocks")
	if r.Status != "warn" {
		t.Errorf("non-strict: Status = %q, want \"warn\" (Detail: %s)", r.Status, r.Detail)
	}

	// Integration pin, mirroring TestCheckScaffoldIntegrity_CorruptManifest_StrictFails.
	binDir := filepath.Join(target, "bin")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, bin := range []string{"claude", "codex", "go"} {
		writeStubBin(t, binDir, bin, "")
	}
	t.Setenv("PATH", binDir)
	t.Setenv("RALPH_ORG_AGMSG_HOME", filepath.Join(target, "no-such-agmsg-home"))

	nonStrictErr := captureStdout(t, func() error { return runDoctorFull(target, false, false) })
	if nonStrictErr != nil {
		t.Errorf("non-strict: err = %v, want nil (a block-computation error is a warning without --strict)", nonStrictErr)
	}
	strictErr := captureStdout(t, func() error { return runDoctorFull(target, false, true) })
	if strictErr == nil {
		t.Fatal("strict: err = nil, want non-nil (a block-computation error must fail --strict)")
	}
}

// TestCheckScaffoldIntegrity_SettingsSnapshotUnreadable_OwnedKeysStrictFails
// is the class-closing audit (cycle 2,
// docs/reports/cross-review-triage-overlay-scaffold-v2-p5.md) regression
// guard for checkSettingsOwnedKeys' first error path: a settings snapshot
// (.ralph/core/settings.ralph.json) that exists but cannot be read as a
// file at all (here, replaced by a directory, so os.ReadFile fails with
// something other than ErrNotExist) is distinct from
// TestCheckScaffoldIntegrity_SettingsSnapshotDeleted_StrictFails' missing-
// snapshot case (which legitimately falls back to "{}" oldOwned) -- an
// unreadable snapshot must fail FR-9(c) itself, not silently degrade.
func TestCheckScaffoldIntegrity_SettingsSnapshotUnreadable_OwnedKeysStrictFails(t *testing.T) {
	target := initV2Project(t, gen1(), "1.0.0-test")

	snapshotPath := filepath.Join(target, ".ralph", "core", "settings.ralph.json")
	if err := os.Remove(snapshotPath); err != nil {
		t.Fatalf("removing settings snapshot: %v", err)
	}
	if err := os.Mkdir(snapshotPath, 0o755); err != nil {
		t.Fatalf("replacing settings snapshot with a directory: %v", err)
	}

	strictResults := checkScaffoldIntegrity(target, true)
	r := findScaffoldResult(t, strictResults, "Scaffold: settings.json owned keys")
	if r.Status != "fail" {
		t.Errorf("strict: Status = %q, want \"fail\" (Detail: %s)", r.Status, r.Detail)
	}
	if !strings.Contains(r.Detail, "reading settings snapshot") {
		t.Errorf("Detail must name the snapshot read failure: %s", r.Detail)
	}

	nonStrictResults := checkScaffoldIntegrity(target, false)
	r = findScaffoldResult(t, nonStrictResults, "Scaffold: settings.json owned keys")
	if r.Status != "warn" {
		t.Errorf("non-strict: Status = %q, want \"warn\" (Detail: %s)", r.Status, r.Detail)
	}
}

// TestCheckScaffoldIntegrity_SettingsJSONReplacedByDirectory_OwnedKeysStrictFails
// is the class-closing audit counterpart for checkSettingsOwnedKeys'
// readFinalDiskContent error path: .claude/settings.json replaced by a
// directory makes os.ReadFile fail with something other than ErrNotExist,
// distinct from TestCheckScaffoldIntegrity_SettingsJSONDeleted_OwnedKeysCatchesIt's
// missing-file case (which readFinalDiskContent treats as legitimate
// absence, "") -- an unreadable settings.json must fail FR-9(c) itself.
func TestCheckScaffoldIntegrity_SettingsJSONReplacedByDirectory_OwnedKeysStrictFails(t *testing.T) {
	target := initV2Project(t, gen1(), "1.0.0-test")

	settingsPath := filepath.Join(target, ".claude", "settings.json")
	if err := os.Remove(settingsPath); err != nil {
		t.Fatalf("removing settings.json: %v", err)
	}
	if err := os.Mkdir(settingsPath, 0o755); err != nil {
		t.Fatalf("replacing settings.json with a directory: %v", err)
	}

	strictResults := checkScaffoldIntegrity(target, true)
	r := findScaffoldResult(t, strictResults, "Scaffold: settings.json owned keys")
	if r.Status != "fail" {
		t.Errorf("strict: Status = %q, want \"fail\" (Detail: %s)", r.Status, r.Detail)
	}
	if !strings.Contains(r.Detail, "reading "+v2SettingsPath) {
		t.Errorf("Detail must name the settings.json read failure: %s", r.Detail)
	}

	nonStrictResults := checkScaffoldIntegrity(target, false)
	r = findScaffoldResult(t, nonStrictResults, "Scaffold: settings.json owned keys")
	if r.Status != "warn" {
		t.Errorf("non-strict: Status = %q, want \"warn\" (Detail: %s)", r.Status, r.Detail)
	}
}

// captureStdout runs fn with os.Stdout redirected to a pipe (discarding the
// captured output) and returns fn's error, mirroring doctor_org_test.go's
// stdout-capture pattern for tests that only care about the returned error.
func captureStdout(t *testing.T, fn func() error) error {
	t.Helper()
	var discard string
	return captureStdoutText(t, &discard, fn)
}

// captureStdoutText is captureStdout plus capturing the printed text into
// *out, for assertions that need to inspect doctor's report lines.
func captureStdoutText(t *testing.T, out *string, fn func() error) error {
	t.Helper()
	origStdout := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe: %v", err)
	}
	os.Stdout = w
	fnErr := fn()
	_ = w.Close()
	os.Stdout = origStdout
	data, _ := io.ReadAll(r)
	*out = string(data)
	return fnErr
}
