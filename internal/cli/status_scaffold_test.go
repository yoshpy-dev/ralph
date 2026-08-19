package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/yoshpy-dev/ralph/internal/scaffold"
)

// statusScaffoldPayload is the subset of `ralph status --json` this file
// asserts against, decoded via the same anonymous/partial-struct pattern
// status_test.go's existing JSON tests use.
type statusScaffoldPayload struct {
	Orgs     []json.RawMessage `json:"orgs"`
	Scaffold *struct {
		Layout string `json:"layout"`
		Files  []struct {
			Path              string `json:"path"`
			Owner             string `json:"owner"`
			ForkedFromVersion string `json:"forked_from_version"`
			Drift             bool   `json:"drift"`
		} `json:"files"`
		Drift []string `json:"drift"`
	} `json:"scaffold"`
}

func decodeStatusScaffoldPayload(t *testing.T, out string) statusScaffoldPayload {
	t.Helper()
	var payload statusScaffoldPayload
	if err := json.Unmarshal([]byte(out), &payload); err != nil {
		t.Fatalf("json.Unmarshal: %v\noutput:\n%s", err, out)
	}
	return payload
}

// TestStatusCmd_ScaffoldSection_V2Project_NoOwnershipIssues is the AC-10
// baseline matrix cell: a freshly initialized v2 project (all paths
// owner=core, zero drift) with no org runtime state at all -- both the text
// "Scaffold ownership" section and the --json "scaffold" key must render,
// and no "scaffold" key literal should be missing.
func TestStatusCmd_ScaffoldSection_V2Project_NoOwnershipIssues(t *testing.T) {
	target := initV2Project(t, gen1(), "1.0.0-test")
	t.Chdir(target)

	orgStateDir := t.TempDir() // no org manifest here -- empty-roster path

	out, err := runStatusCmd(t, "--state-dir", orgStateDir)
	if err != nil {
		t.Fatalf("status: %v (output: %s)", err, out)
	}
	if !strings.Contains(out, "Scaffold ownership:") {
		t.Errorf("expected the scaffold ownership section, got:\n%s", out)
	}
	if !strings.Contains(out, "Unresolved drift: none") {
		t.Errorf("expected \"Unresolved drift: none\" for a clean fresh init, got:\n%s", out)
	}

	jsonOut, err := runStatusCmd(t, "--state-dir", orgStateDir, "--json")
	if err != nil {
		t.Fatalf("status --json: %v (output: %s)", err, jsonOut)
	}
	payload := decodeStatusScaffoldPayload(t, jsonOut)
	if payload.Scaffold == nil {
		t.Fatalf("expected a non-nil scaffold key, got none:\n%s", jsonOut)
	}
	if payload.Scaffold.Layout != scaffold.LayoutV2 {
		t.Errorf("scaffold.layout = %q, want %q", payload.Scaffold.Layout, scaffold.LayoutV2)
	}
	if len(payload.Scaffold.Files) == 0 {
		t.Errorf("expected a non-empty scaffold.files for a fresh v2 init:\n%s", jsonOut)
	}
	if len(payload.Scaffold.Drift) != 0 {
		t.Errorf("expected zero scaffold.drift entries for a clean fresh init, got: %+v", payload.Scaffold.Drift)
	}
	for _, f := range payload.Scaffold.Files {
		if f.Owner == "" {
			t.Errorf("file %s has no owner recorded: %+v", f.Path, f)
		}
	}
}

// TestStatusCmd_ScaffoldSection_ForkAndDrift is the AC-10 matrix cell
// covering an ejected fork (owner=fork, forked_from_version populated) and
// an unresolved-drift owner=core path, in both text and --json.
func TestStatusCmd_ScaffoldSection_ForkAndDrift(t *testing.T) {
	target := initV2Project(t, gen1(), "1.0.0-test")

	// Eject scripts/run-verify.sh: modify disk first (mirrors
	// doctor_scaffold_test.go's TestCheckScaffoldIntegrity_EjectedFork_CoreHashesPass),
	// then eject it into owner=fork with forked_from_version=1.0.0-test.
	writeMigrationDiskFile(t, target, "scripts/run-verify.sh", "#!/bin/sh\necho user-owned-now\n")
	var ejectOut bytes.Buffer
	if err := runEjectIO(target, "scripts/run-verify.sh", &ejectOut); err != nil {
		t.Fatalf("runEjectIO: %v (output: %s)", err, ejectOut.String())
	}

	// Drift scripts/old-tool.sh: modify disk with no eject, leaving it
	// owner=core (ownerForScaffoldPath's catch-all -- "docs/" is seed, so a
	// second core path outside that prefix is needed here) but diverging
	// from both the recorded and current template hash (mirrors
	// TestCheckScaffoldIntegrity_DriftedCoreFile_CoreHashesFail).
	writeMigrationDiskFile(t, target, "scripts/old-tool.sh", "#!/bin/sh\necho user-modified-with-no-fork-record\n")

	t.Chdir(target)
	orgStateDir := t.TempDir()

	out, err := runStatusCmd(t, "--state-dir", orgStateDir)
	if err != nil {
		t.Fatalf("status: %v (output: %s)", err, out)
	}
	if !strings.Contains(out, "scripts/run-verify.sh: fork (forked_from_version=1.0.0-test)") {
		t.Errorf("expected the fork row with forked_from_version in text output, got:\n%s", out)
	}
	if !strings.Contains(out, "Unresolved drift: 1 path(s)") || !strings.Contains(out, "scripts/old-tool.sh") {
		t.Errorf("expected scripts/old-tool.sh listed as unresolved drift, got:\n%s", out)
	}

	jsonOut, err := runStatusCmd(t, "--state-dir", orgStateDir, "--json")
	if err != nil {
		t.Fatalf("status --json: %v (output: %s)", err, jsonOut)
	}
	payload := decodeStatusScaffoldPayload(t, jsonOut)
	if payload.Scaffold == nil {
		t.Fatalf("expected a non-nil scaffold key, got none:\n%s", jsonOut)
	}

	var sawFork, sawDriftedCore bool
	for _, f := range payload.Scaffold.Files {
		switch f.Path {
		case "scripts/run-verify.sh":
			sawFork = true
			if f.Owner != scaffold.OwnerFork {
				t.Errorf("scripts/run-verify.sh owner = %q, want %q", f.Owner, scaffold.OwnerFork)
			}
			if f.ForkedFromVersion != "1.0.0-test" {
				t.Errorf("scripts/run-verify.sh forked_from_version = %q, want %q", f.ForkedFromVersion, "1.0.0-test")
			}
		case "scripts/old-tool.sh":
			sawDriftedCore = true
			if f.Owner != scaffold.OwnerCore {
				t.Errorf("scripts/old-tool.sh owner = %q, want %q", f.Owner, scaffold.OwnerCore)
			}
			if !f.Drift {
				t.Errorf("scripts/old-tool.sh must carry drift=true in the per-file entry")
			}
		}
	}
	if !sawFork {
		t.Errorf("scaffold.files missing scripts/run-verify.sh:\n%s", jsonOut)
	}
	if !sawDriftedCore {
		t.Errorf("scaffold.files missing scripts/old-tool.sh:\n%s", jsonOut)
	}
	if len(payload.Scaffold.Drift) != 1 || payload.Scaffold.Drift[0] != "scripts/old-tool.sh" {
		t.Errorf("scaffold.drift = %+v, want exactly [\"scripts/old-tool.sh\"]", payload.Scaffold.Drift)
	}
}

// TestStatusCmd_ScaffoldSection_LegacyManifest is the AC-10 matrix cell for
// a legacy (pre-v2) manifest: text prints a one-line migration advisory,
// and --json's scaffold key carries only {"layout":"legacy"} (no files
// list, per the handoff's documented shape).
func TestStatusCmd_ScaffoldSection_LegacyManifest(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, ".ralph"), 0o755); err != nil {
		t.Fatalf("MkdirAll .ralph: %v", err)
	}
	legacy := scaffold.NewManifest("0.9.0-test")
	legacy.SetFile("AGENTS.md", "sha256:legacy")
	if err := legacy.Write(filepath.Join(dir, ".ralph", "manifest.toml")); err != nil {
		t.Fatalf("writing legacy manifest: %v", err)
	}
	t.Chdir(dir)
	orgStateDir := t.TempDir()

	out, err := runStatusCmd(t, "--state-dir", orgStateDir)
	if err != nil {
		t.Fatalf("status: %v (output: %s)", err, out)
	}
	if !strings.Contains(out, "Scaffold ownership: legacy manifest layout") {
		t.Errorf("expected the legacy migration advisory, got:\n%s", out)
	}
	if !strings.Contains(out, "ralph upgrade") {
		t.Errorf("expected the legacy advisory to point at `ralph upgrade`, got:\n%s", out)
	}

	jsonOut, err := runStatusCmd(t, "--state-dir", orgStateDir, "--json")
	if err != nil {
		t.Fatalf("status --json: %v (output: %s)", err, jsonOut)
	}
	payload := decodeStatusScaffoldPayload(t, jsonOut)
	if payload.Scaffold == nil {
		t.Fatalf("expected a non-nil scaffold key for a legacy manifest, got none:\n%s", jsonOut)
	}
	if payload.Scaffold.Layout != "legacy" {
		t.Errorf("scaffold.layout = %q, want \"legacy\"", payload.Scaffold.Layout)
	}
	if len(payload.Scaffold.Files) != 0 {
		t.Errorf("expected no scaffold.files for a legacy manifest, got: %+v", payload.Scaffold.Files)
	}
	if strings.Contains(jsonOut, `"files"`) {
		t.Errorf("expected the legacy scaffold shape to omit \"files\" entirely, got:\n%s", jsonOut)
	}
}

// TestStatusCmd_ScaffoldSection_NoManifest_AbsentBothFormats is the AC-10
// matrix cell for a directory that is not a ralph project at all: the text
// output must never mention "Scaffold ownership", and --json must omit the
// "scaffold" key entirely (the chosen representation for "absent", pinned
// here so it cannot silently flip to `"scaffold": null` later).
func TestStatusCmd_ScaffoldSection_NoManifest_AbsentBothFormats(t *testing.T) {
	dir := t.TempDir() // no .ralph/manifest.toml at all
	t.Chdir(dir)
	orgStateDir := t.TempDir()
	seedTwoOrgManifest(t, orgStateDir)

	out, err := runStatusCmd(t, "--state-dir", orgStateDir)
	if err != nil {
		t.Fatalf("status: %v (output: %s)", err, out)
	}
	if strings.Contains(out, "Scaffold ownership") {
		t.Errorf("expected no scaffold section for a non-ralph directory, got:\n%s", out)
	}
	// Pre-existing org output must be completely unaffected.
	if !strings.Contains(out, "org_id: org-a") || !strings.Contains(out, "org_id: org-b") {
		t.Errorf("expected unaffected org output, got:\n%s", out)
	}

	jsonOut, err := runStatusCmd(t, "--state-dir", orgStateDir, "--json")
	if err != nil {
		t.Fatalf("status --json: %v (output: %s)", err, jsonOut)
	}
	if strings.Contains(jsonOut, `"scaffold"`) {
		t.Errorf("expected the \"scaffold\" key omitted entirely for a non-ralph directory, got:\n%s", jsonOut)
	}
}

// TestStatusCmd_ScaffoldSection_EmptyOrgRosterStillRendersScaffold covers
// the "scaffold present, org state absent" matrix cell against the
// printStatusEmpty path: an org-less state dir must still show the
// scaffold-ownership section and must not error.
func TestStatusCmd_ScaffoldSection_EmptyOrgRosterStillRendersScaffold(t *testing.T) {
	target := initV2Project(t, gen1(), "1.0.0-test")
	t.Chdir(target)
	orgStateDir := t.TempDir() // no org manifest ever written here

	out, err := runStatusCmd(t, "--state-dir", orgStateDir)
	if err != nil {
		t.Fatalf("status: %v (output: %s)", err, out)
	}
	if !strings.Contains(out, "Scaffold ownership:") {
		t.Errorf("expected the scaffold section on the empty-roster path, got:\n%s", out)
	}
	if !strings.Contains(out, "no org runtime state found") {
		t.Errorf("expected the pre-existing empty-roster message unaffected, got:\n%s", out)
	}

	jsonOut, err := runStatusCmd(t, "--state-dir", orgStateDir, "--json")
	if err != nil {
		t.Fatalf("status --json: %v (output: %s)", err, jsonOut)
	}
	payload := decodeStatusScaffoldPayload(t, jsonOut)
	if payload.Scaffold == nil || payload.Scaffold.Layout != scaffold.LayoutV2 {
		t.Errorf("expected scaffold.layout=%q on the empty-roster JSON path, got: %+v", scaffold.LayoutV2, payload.Scaffold)
	}
	if len(payload.Orgs) != 0 {
		t.Errorf("expected orgs=[] on the empty-roster JSON path, got: %+v", payload.Orgs)
	}
}

// TestStatusCmd_ScaffoldSection_OrgAndScaffoldCombined_FlagsScopedToOrgOnly
// is the AC-10 matrix cell combining real org state with a v2 scaffold
// project, and pins that --org-id / --state-dir only scope the org portion:
// the scaffold section (and its content) must be identical regardless of
// which --org-id filter is applied.
func TestStatusCmd_ScaffoldSection_OrgAndScaffoldCombined_FlagsScopedToOrgOnly(t *testing.T) {
	target := initV2Project(t, gen1(), "1.0.0-test")
	t.Chdir(target)
	orgStateDir := t.TempDir()
	seedTwoOrgManifest(t, orgStateDir)

	outAll, err := runStatusCmd(t, "--state-dir", orgStateDir)
	if err != nil {
		t.Fatalf("status: %v (output: %s)", err, outAll)
	}
	outFiltered, err := runStatusCmd(t, "--state-dir", orgStateDir, "--org-id", "org-a")
	if err != nil {
		t.Fatalf("status --org-id org-a: %v (output: %s)", err, outFiltered)
	}

	for _, out := range []string{outAll, outFiltered} {
		if !strings.Contains(out, "Scaffold ownership:") {
			t.Errorf("expected the scaffold section regardless of --org-id, got:\n%s", out)
		}
		if !strings.Contains(out, "Unresolved drift: none") {
			t.Errorf("expected a clean-drift scaffold section regardless of --org-id, got:\n%s", out)
		}
	}
	// org-b must be filtered out of the roster by --org-id, same as the
	// pre-existing TestStatusCmd_OrgIDFilterShowsOnlyThatOrg contract --
	// unaffected by the scaffold section's presence.
	if strings.Contains(outFiltered, "org_id: org-b") {
		t.Errorf("--org-id filter leaked org-b into output despite the scaffold section addition:\n%s", outFiltered)
	}
	if !strings.Contains(outAll, "org_id: org-b") {
		t.Errorf("expected org-b present in the unfiltered run:\n%s", outAll)
	}
}
