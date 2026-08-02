package cli

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/yoshpy-dev/ralph/internal/config"
	"github.com/yoshpy-dev/ralph/internal/org/driver"
)

// writeStubBin writes an executable shell script named bin in dir that
// always exits 0, optionally logging its invocation to logPath so tests can
// assert whether a subprocess ran at all.
func writeStubBin(t *testing.T, dir, bin, logPath string) {
	t.Helper()
	var body string
	if logPath != "" {
		body = "#!/bin/sh\necho \"$0 $*\" >> " + logPath + "\necho pong\nexit 0\n"
	} else {
		body = "#!/bin/sh\necho pong\nexit 0\n"
	}
	if err := os.WriteFile(filepath.Join(dir, bin), []byte(body), 0o755); err != nil {
		t.Fatal(err)
	}
}

// writeFailingStubBin writes an executable shell script that always exits 1
// with detail on stderr, simulating a CLI rejecting an unknown model id.
func writeFailingStubBin(t *testing.T, dir, bin, stderrMsg string) {
	t.Helper()
	body := "#!/bin/sh\necho '" + stderrMsg + "' >&2\nexit 1\n"
	if err := os.WriteFile(filepath.Join(dir, bin), []byte(body), 0o755); err != nil {
		t.Fatal(err)
	}
}

// writeAgmsgHome creates dir/scripts/send.sh (executable) so dir counts as a
// usable agmsg home per driver.AgmsgAvailable. When version is non-empty, it
// also writes dir/VERSION with that content.
func writeAgmsgHome(t *testing.T, dir, version string) {
	t.Helper()
	scriptsDir := filepath.Join(dir, "scripts")
	if err := os.MkdirAll(scriptsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(scriptsDir, "send.sh"), []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	if version != "" {
		if err := os.WriteFile(filepath.Join(dir, "VERSION"), []byte(version+"\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

// TestCheckHerdrAgmsgAvailable_AbsentIsInfo pins AC-9: with herdr absent from
// PATH and agmsg's home missing scripts/send.sh, both checks must report
// "info" (not "warn"/"fail") so runDoctorOpts' exit code stays unaffected.
func TestCheckHerdrAgmsgAvailable_AbsentIsInfo(t *testing.T) {
	t.Setenv("PATH", t.TempDir()) // empty PATH — nothing resolvable.

	hr := checkHerdrAvailable()
	if hr.Status != "info" {
		t.Errorf("herdr status = %q, want info", hr.Status)
	}
	if !strings.Contains(hr.Detail, "herdr not installed") {
		t.Errorf("herdr detail = %q, want mention of 'herdr not installed'", hr.Detail)
	}

	ar := checkAgmsgAvailable(filepath.Join(t.TempDir(), "no-such-agmsg-home"))
	if ar.Status != "info" {
		t.Errorf("agmsg status = %q, want info", ar.Status)
	}
	if !strings.Contains(ar.Detail, "agmsg not installed") {
		t.Errorf("agmsg detail = %q, want mention of 'agmsg not installed'", ar.Detail)
	}
}

// TestCheckHerdrAgmsgAvailable_PresentIsPass covers the counterpart: a stub
// herdr binary on PATH and an agmsg home with scripts/send.sh must both be
// reported as available (pass).
func TestCheckHerdrAgmsgAvailable_PresentIsPass(t *testing.T) {
	dir := t.TempDir()
	writeStubBin(t, dir, "herdr", "")
	t.Setenv("PATH", dir)

	agmsgHome := t.TempDir()
	writeAgmsgHome(t, agmsgHome, "")

	if r := checkHerdrAvailable(); r.Status != "pass" {
		t.Errorf("herdr status = %q, want pass (detail=%q)", r.Status, r.Detail)
	}
	if r := checkAgmsgAvailable(agmsgHome); r.Status != "pass" {
		t.Errorf("agmsg status = %q, want pass (detail=%q)", r.Status, r.Detail)
	}
}

// TestCheckAgmsgAvailable_PathBootstrapperNotHome pins AC-1's explicit
// requirement at the doctor layer: an `agmsg` executable on PATH (the npm
// bootstrapper) must not be mistaken for a real agmsg home. Only a home
// directory with scripts/send.sh counts.
func TestCheckAgmsgAvailable_PathBootstrapperNotHome(t *testing.T) {
	dir := t.TempDir()
	writeStubBin(t, dir, "agmsg", "") // npm-bootstrapper-shaped binary on PATH.
	t.Setenv("PATH", dir)

	emptyHome := t.TempDir() // no scripts/send.sh.
	if r := checkAgmsgAvailable(emptyHome); r.Status != "info" {
		t.Errorf("agmsg status = %q, want info despite agmsg on PATH (detail=%q)", r.Status, r.Detail)
	}
}

// TestCheckAgmsgAvailable_VersionShown confirms an available home's VERSION
// file content is surfaced in the check detail.
func TestCheckAgmsgAvailable_VersionShown(t *testing.T) {
	agmsgHome := t.TempDir()
	writeAgmsgHome(t, agmsgHome, agmsgTestedVersion)

	r := checkAgmsgAvailable(agmsgHome)
	if r.Status != "pass" {
		t.Errorf("status = %q, want pass (detail=%q)", r.Status, r.Detail)
	}
	if !strings.Contains(r.Detail, agmsgTestedVersion) {
		t.Errorf("detail %q should mention version %s", r.Detail, agmsgTestedVersion)
	}
}

// TestCheckAgmsgAvailable_VersionMismatchIsInfo confirms a VERSION differing
// from agmsgTestedVersion is surfaced as an informational note, never a
// warn/fail (doctor's exit code must stay unaffected by version drift).
func TestCheckAgmsgAvailable_VersionMismatchIsInfo(t *testing.T) {
	agmsgHome := t.TempDir()
	writeAgmsgHome(t, agmsgHome, "9.9.9")

	r := checkAgmsgAvailable(agmsgHome)
	if r.Status != "info" {
		t.Errorf("status = %q, want info for version mismatch (detail=%q)", r.Status, r.Detail)
	}
	if !strings.Contains(r.Detail, "9.9.9") || !strings.Contains(r.Detail, agmsgTestedVersion) {
		t.Errorf("detail %q should mention both the found version and the tested version %s", r.Detail, agmsgTestedVersion)
	}
}

// TestCheckOrgEnvelope_ReportsPoolSizeAndMaxSeats confirms the envelope
// summary reflects the loaded config's model_pool/max_seats without
// re-loading or re-validating config itself.
func TestCheckOrgEnvelope_ReportsPoolSizeAndMaxSeats(t *testing.T) {
	cfg := config.Default()
	r := checkOrgEnvelope(cfg)
	if r.Status != "info" {
		t.Errorf("status = %q, want info", r.Status)
	}
	wantPoolSize := len(cfg.Org.ModelPool)
	if !strings.Contains(r.Detail, "model_pool: "+strconv.Itoa(wantPoolSize)) {
		t.Errorf("detail %q missing pool size %d", r.Detail, wantPoolSize)
	}
	if !strings.Contains(r.Detail, "max_seats: "+strconv.Itoa(cfg.Org.MaxSeats)) {
		t.Errorf("detail %q missing max_seats %d", r.Detail, cfg.Org.MaxSeats)
	}
}

// TestRunDoctorOpts_HerdrAgmsgAbsent_ExitCodeUnaffected is the integration
// pin for AC-9: a fully-initialized project with claude/codex/go stubbed on
// PATH (so the pre-existing required checks pass) but herdr/agmsg absent
// must still return a nil error from runDoctorOpts — informational findings
// must not flip the doctor exit code.
func TestRunDoctorOpts_HerdrAgmsgAbsent_ExitCodeUnaffected(t *testing.T) {
	setupTestEmbedFS(t)
	Version = "0.1.0-test"

	dir := t.TempDir()
	cfg := initConfig{ProjectName: "test", Packs: []string{"golang"}}
	if err := executeInit(dir, cfg, false); err != nil {
		t.Fatalf("init: %v", err)
	}

	binDir := filepath.Join(dir, "bin")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		t.Fatal(err)
	}
	// claude/codex/go present (required-by-default checks); herdr/agmsg
	// deliberately absent.
	for _, bin := range []string{"claude", "codex", "go"} {
		writeStubBin(t, binDir, bin, "")
	}
	t.Setenv("PATH", binDir)
	// Pin agmsg home to a guaranteed-empty directory so this test is
	// deterministic regardless of whether the machine running it happens to
	// have a real agmsg install at the default ~/.agents/skills/agmsg.
	t.Setenv("RALPH_ORG_AGMSG_HOME", filepath.Join(dir, "no-such-agmsg-home"))

	origStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	err := runDoctorOpts(dir, false)
	_ = w.Close()
	os.Stdout = origStdout
	out, _ := io.ReadAll(r)

	if err != nil {
		t.Fatalf("runDoctorOpts returned error with only informational herdr/agmsg findings: %v\noutput:\n%s", err, out)
	}
	if !strings.Contains(string(out), "herdr: info") {
		t.Errorf("expected herdr info line in output:\n%s", out)
	}
	if !strings.Contains(string(out), "agmsg: info") {
		t.Errorf("expected agmsg info line in output:\n%s", out)
	}
}

// TestRunDoctorOpts_ProbeModelsFalse_NoSubprocess confirms that without
// --probe-models, no model-probe subprocess is ever launched. The stub
// claude binary logs every invocation's argv; runDoctorOpts' pre-existing
// Check 1 (checkClaudeCLI) legitimately invokes `claude --version`, so the
// assertion targets the "--model" flag specifically — that flag only ever
// appears in a driver.ProbeModel call, never in the --version probe.
func TestRunDoctorOpts_ProbeModelsFalse_NoSubprocess(t *testing.T) {
	setupTestEmbedFS(t)
	Version = "0.1.0-test"

	dir := t.TempDir()
	cfg := initConfig{ProjectName: "test", Packs: []string{"golang"}}
	if err := executeInit(dir, cfg, false); err != nil {
		t.Fatalf("init: %v", err)
	}

	binDir := filepath.Join(dir, "bin")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		t.Fatal(err)
	}
	logPath := filepath.Join(dir, "claude-invocations.log")
	writeStubBin(t, binDir, "claude", logPath)
	writeStubBin(t, binDir, "codex", "")
	writeStubBin(t, binDir, "go", "")
	t.Setenv("PATH", binDir)

	if err := runDoctorOpts(dir, false); err != nil {
		t.Fatalf("runDoctorOpts: %v", err)
	}

	logBytes, statErr := os.ReadFile(logPath)
	if statErr != nil {
		if !os.IsNotExist(statErr) {
			t.Fatalf("unexpected read error: %v", statErr)
		}
		// No invocation at all (e.g. Check 1 skipped) — also satisfies the
		// "no probe ran" assertion.
		return
	}
	if strings.Contains(string(logBytes), "--model") {
		t.Errorf("claude invocation log contains --model — a probe subprocess ran despite --probe-models not being set:\n%s", logBytes)
	}
}

// TestCheckOrgModelProbes_SucceedsAndFails covers the --probe-models path:
// a known model succeeds, an unknown model fails with detail carried
// through, and the codex failure message is labeled advisory.
func TestCheckOrgModelProbes_SucceedsAndFails(t *testing.T) {
	dir := t.TempDir()
	writeStubBin(t, dir, "claude", "")
	writeFailingStubBin(t, dir, "codex", "error: unknown model")
	t.Setenv("PATH", dir)

	cfg := config.Config{
		Org: config.OrgConfig{
			DriverPool: []string{"claude", "codex"},
			ModelPool: []config.OrgModelPoolEntry{
				{Driver: "claude", Model: "sonnet"},
				{Driver: "codex", Model: "not-a-real-model"},
			},
			MaxSeats: 5,
		},
	}

	results := checkOrgModelProbes(cfg, driver.ExecRunner{})
	if len(results) != 2 {
		t.Fatalf("expected 2 probe results, got %d: %+v", len(results), results)
	}

	claudeResult := results[0]
	if claudeResult.Status != "pass" {
		t.Errorf("claude probe status = %q, want pass (detail=%q)", claudeResult.Status, claudeResult.Detail)
	}

	codexResult := results[1]
	if codexResult.Status != "warn" {
		t.Errorf("codex probe status = %q, want warn", codexResult.Status)
	}
	if !strings.Contains(codexResult.Detail, "advisory") {
		t.Errorf("codex probe detail %q should be labeled advisory", codexResult.Detail)
	}
	if !strings.Contains(codexResult.Detail, "unknown model") {
		t.Errorf("codex probe detail %q should carry the CLI's stderr detail", codexResult.Detail)
	}
}

// TestCheckOrgModelProbes_SkipsWhenBinaryMissing confirms that a driver
// with no CLI on PATH produces one informational skip line instead of
// per-model failures, and that no subprocess is attempted.
func TestCheckOrgModelProbes_SkipsWhenBinaryMissing(t *testing.T) {
	t.Setenv("PATH", t.TempDir()) // neither claude nor codex resolvable.

	cfg := config.Config{
		Org: config.OrgConfig{
			DriverPool: []string{"claude"},
			ModelPool: []config.OrgModelPoolEntry{
				{Driver: "claude", Model: "opus"},
				{Driver: "claude", Model: "sonnet"},
			},
			MaxSeats: 5,
		},
	}

	results := checkOrgModelProbes(cfg, driver.ExecRunner{})
	if len(results) != 1 {
		t.Fatalf("expected 1 skip result grouped by driver, got %d: %+v", len(results), results)
	}
	if results[0].Status != "info" {
		t.Errorf("skip status = %q, want info", results[0].Status)
	}
	if !strings.Contains(results[0].Detail, "skipping 2 model probe(s)") {
		t.Errorf("skip detail %q should mention skipping 2 model probe(s)", results[0].Detail)
	}
}

// TestCheckOrgModelProbes_RespectsContextBudget is a smoke check that
// ProbeModel is invoked with a bounded (non-background) context, so a hung
// CLI cannot wedge `ralph doctor --probe-models` indefinitely.
func TestCheckOrgModelProbes_RespectsContextBudget(t *testing.T) {
	dir := t.TempDir()
	writeStubBin(t, dir, "claude", "")
	t.Setenv("PATH", dir)

	cfg := config.Config{
		Org: config.OrgConfig{
			DriverPool: []string{"claude"},
			ModelPool:  []config.OrgModelPoolEntry{{Driver: "claude", Model: "sonnet"}},
			MaxSeats:   5,
		},
	}

	results := checkOrgModelProbes(cfg, recordingRunner{t: t})
	if len(results) != 1 || results[0].Status != "pass" {
		t.Fatalf("unexpected results: %+v", results)
	}
}

// recordingRunner wraps driver.ExecRunner but asserts the context passed by
// probeOrgModel carries a deadline (i.e. is not context.Background()).
type recordingRunner struct {
	t *testing.T
}

func (r recordingRunner) Run(ctx context.Context, name string, args ...string) (string, error) {
	if _, ok := ctx.Deadline(); !ok {
		r.t.Errorf("expected ctx passed to Runner.Run to carry a deadline (30s probe budget)")
	}
	return driver.ExecRunner{}.Run(ctx, name, args...)
}
