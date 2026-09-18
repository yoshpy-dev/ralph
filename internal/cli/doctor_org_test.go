package cli

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"

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

	noSuchHome := filepath.Join(t.TempDir(), "no-such-agmsg-home")
	ar := checkAgmsgAvailable(noSuchHome)
	if ar.Status != "info" {
		t.Errorf("agmsg status = %q, want info", ar.Status)
	}
	if !strings.Contains(ar.Detail, "agmsg not installed") {
		t.Errorf("agmsg detail = %q, want mention of 'agmsg not installed'", ar.Detail)
	}
	// AC-10a (tech-debt fix): the not-installed detail must also name the
	// resolved home directory doctor actually checked, so a misconfigured
	// agmsg_home is diagnosable from the doctor output alone.
	if !strings.Contains(ar.Detail, noSuchHome) {
		t.Errorf("agmsg detail = %q, want it to mention the resolved home %q", ar.Detail, noSuchHome)
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
	// AC-10a: the "available" detail must also name the resolved home, not
	// just report the pass/fail status.
	if r := checkAgmsgAvailable(agmsgHome); !strings.Contains(r.Detail, agmsgHome) {
		t.Errorf("agmsg detail = %q, want it to mention the resolved home %q", r.Detail, agmsgHome)
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

// writeCodexModelsCache writes a minimal codex models_cache.json fixture at
// dir/models_cache.json containing one entry per slug (mirroring the
// codex-cli 0.149.1 shape: {"models": [{"slug": "...", ...}, ...]}).
func writeCodexModelsCache(t *testing.T, dir string, slugs ...string) {
	t.Helper()
	var b strings.Builder
	b.WriteString(`{"models": [`)
	for i, slug := range slugs {
		if i > 0 {
			b.WriteString(",")
		}
		b.WriteString(`{"slug": "` + slug + `", "display_name": "` + slug + `", "visibility": "list"}`)
	}
	b.WriteString(`]}`)
	if err := os.WriteFile(filepath.Join(dir, "models_cache.json"), []byte(b.String()), 0o644); err != nil {
		t.Fatal(err)
	}
}

// TestCheckCodexModelSlugs_AllPresent_Pass covers AC-4(a): every codex
// model_pool slug is present in the cache -> pass, listing the count.
func TestCheckCodexModelSlugs_AllPresent_Pass(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("CODEX_HOME", dir)
	writeCodexModelsCache(t, dir, "gpt-5.5", "gpt-5.5-codex")

	cfg := config.Config{Org: config.OrgConfig{ModelPool: []config.OrgModelPoolEntry{
		{Driver: "claude", Model: "sonnet"},
		{Driver: "codex", Model: "gpt-5.5"},
		{Driver: "codex", Model: "gpt-5.5-codex"},
	}}}

	r := checkCodexModelSlugs(cfg)
	if r.Status != "pass" {
		t.Fatalf("status = %q, want pass (detail=%q)", r.Status, r.Detail)
	}
	if !strings.Contains(r.Detail, "2 codex model_pool slug(s)") {
		t.Errorf("detail %q should report the codex slug count (2)", r.Detail)
	}
}

// TestCheckCodexModelSlugs_SomeMissing_WarnNamesEach covers AC-4(b): one or
// more pool codex slugs absent from the cache -> warn, naming each missing
// slug.
func TestCheckCodexModelSlugs_SomeMissing_WarnNamesEach(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("CODEX_HOME", dir)
	writeCodexModelsCache(t, dir, "gpt-5.5")

	cfg := config.Config{Org: config.OrgConfig{ModelPool: []config.OrgModelPoolEntry{
		{Driver: "codex", Model: "gpt-5.5"},
		{Driver: "codex", Model: "gpt-9000-retired"},
	}}}

	r := checkCodexModelSlugs(cfg)
	if r.Status != "warn" {
		t.Fatalf("status = %q, want warn (detail=%q)", r.Status, r.Detail)
	}
	if !strings.Contains(r.Detail, "gpt-9000-retired") {
		t.Errorf("detail %q should name the missing slug", r.Detail)
	}
	if strings.Contains(r.Detail, "gpt-5.5,") || strings.Contains(r.Detail, "gpt-5.5 ") {
		t.Errorf("detail %q should not name the present slug as missing", r.Detail)
	}
}

// TestCheckCodexModelSlugs_CacheMissing_InfoNamesPath covers AC-4(c): the
// cache file does not exist -> info skip naming the path it looked for.
func TestCheckCodexModelSlugs_CacheMissing_InfoNamesPath(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("CODEX_HOME", dir) // no models_cache.json written.

	cfg := config.Config{Org: config.OrgConfig{ModelPool: []config.OrgModelPoolEntry{
		{Driver: "codex", Model: "gpt-5.5"},
	}}}

	r := checkCodexModelSlugs(cfg)
	if r.Status != "info" {
		t.Fatalf("status = %q, want info (detail=%q)", r.Status, r.Detail)
	}
	wantPath := filepath.Join(dir, "models_cache.json")
	if !strings.Contains(r.Detail, wantPath) {
		t.Errorf("detail %q should name the cache path it looked for (%s)", r.Detail, wantPath)
	}
}

// TestCheckCodexModelSlugs_NoCodexEntries_Info covers the no-codex-entries
// branch: model_pool has entries, but none for driver codex -> info,
// without ever touching the cache path.
func TestCheckCodexModelSlugs_NoCodexEntries_Info(t *testing.T) {
	cfg := config.Config{Org: config.OrgConfig{ModelPool: []config.OrgModelPoolEntry{
		{Driver: "claude", Model: "sonnet"},
		{Driver: "claude", Model: "opus"},
	}}}

	r := checkCodexModelSlugs(cfg)
	if r.Status != "info" {
		t.Fatalf("status = %q, want info (detail=%q)", r.Status, r.Detail)
	}
	if r.Detail != "no codex entries in model_pool" {
		t.Errorf("detail = %q, want exact %q", r.Detail, "no codex entries in model_pool")
	}
}

// TestCheckCodexModelSlugs_UnparsableCache_InfoNotWarn covers AC-4's
// unparsable-JSON case: an invalid cache file is skipped with "info" (the
// cache is best-effort data owned by codex, not a ralph-level warning).
func TestCheckCodexModelSlugs_UnparsableCache_InfoNotWarn(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("CODEX_HOME", dir)
	if err := os.WriteFile(filepath.Join(dir, "models_cache.json"), []byte("{not valid json"), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg := config.Config{Org: config.OrgConfig{ModelPool: []config.OrgModelPoolEntry{
		{Driver: "codex", Model: "gpt-5.5"},
	}}}

	r := checkCodexModelSlugs(cfg)
	if r.Status != "info" {
		t.Fatalf("status = %q, want info (a malformed cache must never be a warn — it is codex-owned best-effort data)", r.Status)
	}
	if !strings.Contains(r.Detail, "not valid JSON") {
		t.Errorf("detail %q should surface the parse error", r.Detail)
	}
}

// TestCheckCodexModelSlugs_CodexHomeUsedLiterally_TrailingSpace is a
// regression guard for the "never trim or normalize a non-empty CODEX_HOME"
// contract (codex-rs/utils/home-dir/src/lib.rs, find_codex_home, identical
// at rust-v0.149.1 and rust-v0.154.0, checked 2026-09-17: CODEX_HOME is
// filtered only on emptiness and then used as-is -- never trimmed). The
// pre-fix resolver also joined CODEX_HOME's raw value, so a trailing-space
// CODEX_HOME alone does not distinguish old from new behavior; this test
// pins the contract against any future cleanup that reintroduces a
// TrimSpace, using two sibling directories that differ only by a trailing
// space. A directory whose name carries a trailing space holds a cache with
// every codex slug the test's cfg uses (so reading it would pass), while
// the trimmed-name sibling holds a cache missing those slugs (so reading it
// would warn). Setting CODEX_HOME to the untrimmed literal name and
// asserting pass proves codexModelsCachePath read the literal directory,
// not a trimmed one.
func TestCheckCodexModelSlugs_CodexHomeUsedLiterally_TrailingSpace(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("trailing-space directory names are not creatable on Windows")
	}

	literal := filepath.Join(t.TempDir(), "codex ")
	trimmed := strings.TrimRight(literal, " ")
	if err := os.MkdirAll(literal, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(trimmed, 0o755); err != nil {
		t.Fatal(err)
	}
	writeCodexModelsCache(t, literal, "gpt-5.5")
	writeCodexModelsCache(t, trimmed /* no matching slugs */)

	t.Setenv("CODEX_HOME", literal)

	cfg := config.Config{Org: config.OrgConfig{ModelPool: []config.OrgModelPoolEntry{
		{Driver: "codex", Model: "gpt-5.5"},
	}}}

	r := checkCodexModelSlugs(cfg)
	if r.Status != "pass" {
		t.Fatalf("status = %q, want pass (detail=%q) -- CODEX_HOME must be read literally, not trimmed", r.Status, r.Detail)
	}
}

// TestCheckCodexModelSlugs_WhitespaceOnlyCodexHome_UsedLiterally covers the
// one input where the literal-CODEX_HOME contract changes doctor's
// behavior: the pre-fix resolver treated a whitespace-only CODEX_HOME as
// unset and fell back to $HOME/.codex, while codex's find_codex_home
// filters only on emptiness and would try to use " " as a directory. The
// resolver now joins the literal value, so the check reports the cache as
// missing at that literal path (info, never a hard failure) instead of
// silently reading a different directory.
func TestCheckCodexModelSlugs_WhitespaceOnlyCodexHome_UsedLiterally(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("relies on a whitespace-only path component resolving to fs.ErrNotExist, which Windows does not guarantee")
	}
	t.Setenv("CODEX_HOME", " ")

	cfg := config.Config{Org: config.OrgConfig{ModelPool: []config.OrgModelPoolEntry{
		{Driver: "codex", Model: "gpt-5.5"},
	}}}

	r := checkCodexModelSlugs(cfg)
	if r.Status != "info" {
		t.Fatalf("status = %q, want info (detail=%q)", r.Status, r.Detail)
	}
	wantPath := filepath.Join(" ", "models_cache.json")
	if !strings.Contains(r.Detail, "not found at "+wantPath) {
		t.Errorf("detail %q should name the literal path %q", r.Detail, wantPath)
	}
}

// TestCheckCodexModelSlugs_HomeUnresolvable_Info covers AC-1(b): when
// CODEX_HOME is empty and os.UserHomeDir cannot resolve a home directory,
// checkCodexModelSlugs reports status info (a best-effort check, never a
// hard failure) with the resolution error surfaced in Detail.
func TestCheckCodexModelSlugs_HomeUnresolvable_Info(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("os.UserHomeDir falls back to USERPROFILE/HOMEDRIVE+HOMEPATH on Windows, not HOME")
	}

	t.Setenv("CODEX_HOME", "")
	t.Setenv("HOME", "")

	cfg := config.Config{Org: config.OrgConfig{ModelPool: []config.OrgModelPoolEntry{
		{Driver: "codex", Model: "gpt-5.5"},
	}}}

	r := checkCodexModelSlugs(cfg)
	if r.Status != "info" {
		t.Fatalf("status = %q, want info (detail=%q)", r.Status, r.Detail)
	}
	if !strings.Contains(r.Detail, "could not resolve codex models cache path") {
		t.Errorf("detail %q should name the resolution failure", r.Detail)
	}
}

// TestCheckCodexModelSlugs_DetailCarriesExactCacheMtime pins AC-1: the
// rendered timestamp is the cache file's actual mtime, not the wall clock at
// check time. os.Chtimes sets a known, second-aligned mtime five minutes in
// the past; an implementation that rendered time.Now() instead would not
// produce this exact RFC3339 substring.
func TestCheckCodexModelSlugs_DetailCarriesExactCacheMtime(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("CODEX_HOME", dir)
	writeCodexModelsCache(t, dir, "gpt-5.5")

	known := time.Now().Add(-5 * time.Minute).Truncate(time.Second)
	if err := os.Chtimes(filepath.Join(dir, "models_cache.json"), known, known); err != nil {
		t.Fatal(err)
	}

	cfg := config.Config{Org: config.OrgConfig{ModelPool: []config.OrgModelPoolEntry{
		{Driver: "codex", Model: "gpt-5.5"},
	}}}

	r := checkCodexModelSlugs(cfg)
	if r.Status != "pass" {
		t.Fatalf("status = %q, want pass (detail=%q)", r.Status, r.Detail)
	}
	wantStamp := "cache written " + known.UTC().Format(time.RFC3339)
	if !strings.Contains(r.Detail, wantStamp) {
		t.Errorf("detail %q should contain the exact mtime stamp %q", r.Detail, wantStamp)
	}
	if strings.Contains(r.Detail, "cache may be stale") {
		t.Errorf("detail %q should not carry a stale note for a 5-minute-old cache", r.Detail)
	}
	if !strings.Contains(r.Detail, "5m ago") {
		t.Errorf("detail %q should report the age as 5m ago", r.Detail)
	}
}

// TestCheckCodexModelSlugs_StaleCache_WarnCarriesExactMtimeAndStaleNote
// covers AC-1/AC-2 on the warn path: a 48-hour-old cache still names the
// missing slug, carries the exact mtime, and carries the stale note.
func TestCheckCodexModelSlugs_StaleCache_WarnCarriesExactMtimeAndStaleNote(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("CODEX_HOME", dir)
	writeCodexModelsCache(t, dir, "gpt-5.5")

	known := time.Now().Add(-48 * time.Hour).Truncate(time.Second)
	if err := os.Chtimes(filepath.Join(dir, "models_cache.json"), known, known); err != nil {
		t.Fatal(err)
	}

	cfg := config.Config{Org: config.OrgConfig{ModelPool: []config.OrgModelPoolEntry{
		{Driver: "codex", Model: "gpt-5.5"},
		{Driver: "codex", Model: "gpt-9000-retired"},
	}}}

	r := checkCodexModelSlugs(cfg)
	if r.Status != "warn" {
		t.Fatalf("status = %q, want warn (detail=%q)", r.Status, r.Detail)
	}
	if !strings.Contains(r.Detail, "gpt-9000-retired") {
		t.Errorf("detail %q should name the missing slug", r.Detail)
	}
	wantStamp := "cache written " + known.UTC().Format(time.RFC3339)
	if !strings.Contains(r.Detail, wantStamp) {
		t.Errorf("detail %q should contain the exact mtime stamp %q", r.Detail, wantStamp)
	}
	if !strings.Contains(r.Detail, "cache may be stale") {
		t.Errorf("detail %q should carry a stale note for a 48-hour-old cache", r.Detail)
	}
	if !strings.Contains(r.Detail, "2d ago") {
		t.Errorf("detail %q should report the age as 2d ago", r.Detail)
	}
}

// TestCheckCodexModelSlugs_StaleCache_PassAlsoCarriesStaleNote covers AC-2:
// stale notes apply to pass too, because an old cache is weak evidence a
// slug still exists.
func TestCheckCodexModelSlugs_StaleCache_PassAlsoCarriesStaleNote(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("CODEX_HOME", dir)
	writeCodexModelsCache(t, dir, "gpt-5.5")

	known := time.Now().Add(-48 * time.Hour).Truncate(time.Second)
	if err := os.Chtimes(filepath.Join(dir, "models_cache.json"), known, known); err != nil {
		t.Fatal(err)
	}

	cfg := config.Config{Org: config.OrgConfig{ModelPool: []config.OrgModelPoolEntry{
		{Driver: "codex", Model: "gpt-5.5"},
	}}}

	r := checkCodexModelSlugs(cfg)
	if r.Status != "pass" {
		t.Fatalf("status = %q, want pass (detail=%q)", r.Status, r.Detail)
	}
	if !strings.Contains(r.Detail, "cache may be stale") {
		t.Errorf("detail %q should carry a stale note for a 48-hour-old cache even on pass", r.Detail)
	}
}

// TestFormatCacheAge covers AC-3: the age formatter's granularity switches
// (minutes below an hour, hours below a day, days otherwise) and clamps
// negative ages (clock skew or a future mtime) to 0m.
func TestFormatCacheAge(t *testing.T) {
	tests := []struct {
		name string
		d    time.Duration
		want string
	}{
		{"negative clamps to 0m", -time.Minute, "0m"},
		{"sub-minute rounds to 0m", 30 * time.Second, "0m"},
		{"90 minutes is 1h", 90 * time.Minute, "1h"},
		{"47 hours is 1d", 47 * time.Hour, "1d"},
		{"49 hours is 2d", 49 * time.Hour, "2d"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := formatCacheAge(tt.d); got != tt.want {
				t.Errorf("formatCacheAge(%v) = %q, want %q", tt.d, got, tt.want)
			}
		})
	}
}

// TestCheckCodexModelSlugs_CacheChangedWhileReading_FreshnessUnknown covers
// AC-4: readCodexModelsCache's before/after Stat disagree when the cache is
// rewritten in place while being read. writeCodexModelsCache uses
// os.WriteFile, which truncates and rewrites the same inode, so the open
// handle's second Stat observes the new size/mtime (this holds on
// darwin/linux, the CI platforms this repo targets). The slug verdict must
// still come from the bytes read before the rewrite (the original,
// missing-slug cache), while the Detail reports the freshness as unknown
// instead of presenting a stale or fresh timestamp for a version of the
// cache that isn't the one the verdict was computed from.
func TestCheckCodexModelSlugs_CacheChangedWhileReading_FreshnessUnknown(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("CODEX_HOME", dir)
	writeCodexModelsCache(t, dir, "gpt-5.5")

	known := time.Now().Add(-48 * time.Hour).Truncate(time.Second)
	if err := os.Chtimes(filepath.Join(dir, "models_cache.json"), known, known); err != nil {
		t.Fatal(err)
	}

	codexCacheBetweenReadAndStat = func() {
		writeCodexModelsCache(t, dir, "gpt-5.5", "gpt-9000-retired")
	}
	t.Cleanup(func() { codexCacheBetweenReadAndStat = nil })

	cfg := config.Config{Org: config.OrgConfig{ModelPool: []config.OrgModelPoolEntry{
		{Driver: "codex", Model: "gpt-5.5"},
		{Driver: "codex", Model: "gpt-9000-retired"},
	}}}

	r := checkCodexModelSlugs(cfg)
	if r.Status != "warn" {
		t.Fatalf("status = %q, want warn (verdict must come from the bytes read before the in-place rewrite; detail=%q)", r.Status, r.Detail)
	}
	if !strings.Contains(r.Detail, "gpt-9000-retired") {
		t.Errorf("detail %q should name the slug missing from the pre-rewrite cache", r.Detail)
	}
	if !strings.Contains(r.Detail, "freshness unknown") {
		t.Errorf("detail %q should report freshness unknown after an in-place rewrite during read", r.Detail)
	}
	if strings.Contains(r.Detail, "cache may be stale") {
		t.Errorf("detail %q should not carry a stale note when freshness is unknown", r.Detail)
	}
	if strings.Contains(r.Detail, "cache written") {
		t.Errorf("detail %q should not carry a cache written timestamp when freshness is unknown", r.Detail)
	}
}
