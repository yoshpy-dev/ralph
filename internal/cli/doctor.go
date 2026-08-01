package cli

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	toml "github.com/pelletier/go-toml/v2"
	"github.com/spf13/cobra"

	"github.com/yoshpy-dev/ralph/internal/config"
	"github.com/yoshpy-dev/ralph/internal/org/driver"
	"github.com/yoshpy-dev/ralph/internal/scaffold"
)

func newDoctorCmd() *cobra.Command {
	var probeModels bool
	cmd := &cobra.Command{
		Use:   "doctor",
		Short: "Check environment and project setup",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runDoctorOpts(".", probeModels)
		},
	}
	cmd.Flags().BoolVar(&probeModels, "probe-models", false,
		"probe every [org].model_pool entry by launching a minimal CLI invocation per model (slower; requires claude/codex on PATH)")
	return cmd
}

type checkResult struct {
	Name   string `json:"name"`
	Status string `json:"status"` // pass, info, warn, fail
	Detail string `json:"detail,omitempty"`
}

// runDoctor is the pre-org-runtime entry point, preserved so existing
// callers/tests that don't know about --probe-models keep working unchanged.
// It never runs model-pool probes (equivalent to --probe-models=false).
func runDoctor(targetDir string) error {
	return runDoctorOpts(targetDir, false)
}

// runDoctorOpts is runDoctor plus the --probe-models opt-in: when true, every
// [org].model_pool entry is probed via internal/org/driver.ProbeModel (Check
// 13). Model probes are the only check in this function that spawn CLI
// subprocesses beyond the pre-existing --version probes, so they stay
// opt-in and off by default.
func runDoctorOpts(targetDir string, probeModels bool) error {
	cfg, cfgErr := config.Load(filepath.Join(targetDir, "ralph.toml"))
	var results []checkResult

	if cfgErr != nil && !errors.Is(cfgErr, fs.ErrNotExist) {
		results = append(results, checkResult{
			Name:   "ralph.toml",
			Status: "warn",
			Detail: fmt.Sprintf("parse error: %v — using defaults", cfgErr),
		})
	}

	// Check 1: Claude Code CLI.
	results = append(results, checkClaudeCLI(cfg))

	// Check 2: Codex.
	results = append(results, checkCodexCLI(cfg))

	// Check 3: Codex effective config (project trust + hooks feature + at least one hook).
	results = append(results, checkCodexEffectiveConfig(targetDir))

	// Check 4: Hooks integrity.
	results = append(results, checkHooks(targetDir))

	// Check 5: Manifest version.
	results = append(results, checkManifestVersion(targetDir))

	// Check 6: Language pack verify.sh (checks project's installed packs via manifest).
	results = append(results, checkInstalledPacks(targetDir)...)

	// Check 7: Loop driver effective value (env > TOML > default).
	results = append(results, checkLoopDriver(cfg, os.Getenv))

	// Check 8: Go availability.
	results = append(results, checkGo(cfg))

	// Check 9: Stale orchestrator state.
	results = append(results, checkStaleOrchestratorState(targetDir))

	// Check 10: herdr availability (org runtime driver adapter).
	results = append(results, checkHerdrAvailable())

	// Check 11: agmsg availability (org runtime driver adapter).
	results = append(results, checkAgmsgAvailable())

	// Check 12: [org] envelope summary (pool size / max_seats).
	results = append(results, checkOrgEnvelope(cfg))

	// Check 13: optional model-pool probes (--probe-models).
	if probeModels {
		results = append(results, checkOrgModelProbes(cfg, driver.ExecRunner{})...)
	}

	// Print results.
	fmt.Println("ralph doctor")
	fmt.Println()

	allPass := true
	for _, r := range results {
		icon := "✓"
		switch r.Status {
		case "info":
			icon = "ℹ"
		case "warn":
			icon = "⚠"
		case "fail":
			icon = "✗"
			allPass = false
		}
		fmt.Printf("  %s %s: %s", icon, r.Name, r.Status)
		if r.Detail != "" {
			fmt.Printf(" — %s", r.Detail)
		}
		fmt.Println()
	}

	fmt.Println()
	if allPass {
		fmt.Println("All checks passed.")
		return nil
	}
	fmt.Println("Some checks failed. Fix the issues above.")
	return fmt.Errorf("doctor: %d check(s) failed", countFailed(results))
}

func countFailed(results []checkResult) int {
	n := 0
	for _, r := range results {
		if r.Status == "fail" {
			n++
		}
	}
	return n
}

// probeBinary runs `<bin> --version` to confirm the binary on PATH is actually
// callable. A bare exec.LookPath success is not enough — stale or broken
// shims (npm-installed CLIs that lost their entry script, version managers
// pointing at a removed install) appear on PATH but blow up at runtime,
// which lets `ralph doctor` report `pass` while every subsequent /work or
// /cross-review fails.
//
// Bounded by a 5-second timeout so a hung CLI cannot wedge `ralph doctor`.
// Returns the first non-empty line of the version output so multi-line
// banners do not break the doctor table layout.
func probeBinary(bin string) (version string, err error) {
	if _, lookErr := exec.LookPath(bin); lookErr != nil {
		return "", fmt.Errorf("%s not found in PATH: %w", bin, lookErr)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	out, runErr := exec.CommandContext(ctx, bin, "--version").CombinedOutput()
	if ctx.Err() == context.DeadlineExceeded {
		return "", fmt.Errorf("%s --version timed out after 5s", bin)
	}
	if runErr != nil {
		return "", fmt.Errorf("%s --version failed: %w", bin, runErr)
	}
	for _, line := range strings.Split(string(out), "\n") {
		if trimmed := strings.TrimSpace(line); trimmed != "" {
			return trimmed, nil
		}
	}
	return "", fmt.Errorf("%s --version produced no output", bin)
}

func checkClaudeCLI(cfg config.Config) checkResult {
	r := checkResult{Name: "Claude Code CLI"}
	version, err := probeBinary("claude")
	if err != nil {
		if cfg.Doctor.RequireClaudeCLI {
			r.Status = "fail"
			r.Detail = fmt.Sprintf("claude unusable: %v", err)
		} else {
			r.Status = "warn"
			r.Detail = fmt.Sprintf("claude unusable (not required): %v", err)
		}
		return r
	}
	r.Status = "pass"
	r.Detail = version
	return r
}

func checkCodexCLI(cfg config.Config) checkResult {
	r := checkResult{Name: "Codex"}
	version, err := probeBinary("codex")
	if err != nil {
		if cfg.Doctor.RequireCodexCLI {
			r.Status = "fail"
			r.Detail = fmt.Sprintf("codex unusable: %v", err)
		} else {
			r.Status = "warn"
			r.Detail = fmt.Sprintf("codex unusable (not required): %v", err)
		}
		return r
	}
	r.Status = "pass"
	r.Detail = version
	return r
}

// checkCodexEffectiveConfig confirms that .codex/config.toml is present and
// carries the bits Codex actually loads from a project-level config:
// `[features] hooks = true` plus at least one [hooks.<event>] entry.
// We cannot probe Codex's trust state from Go, so the result stays a warning
// when the file is structurally fine — the user has to confirm trust via
// `codex trust .` and the .codex/README.md guidance.
func checkCodexEffectiveConfig(targetDir string) checkResult {
	r := checkResult{Name: "Codex effective config"}
	cfgPath := filepath.Join(targetDir, ".codex", "config.toml")
	data, err := os.ReadFile(cfgPath)
	if errors.Is(err, fs.ErrNotExist) {
		r.Status = "warn"
		r.Detail = ".codex/config.toml not found"
		return r
	}
	if err != nil {
		r.Status = "warn"
		r.Detail = fmt.Sprintf("could not read .codex/config.toml: %v", err)
		return r
	}

	var raw map[string]any
	if err := toml.Unmarshal(data, &raw); err != nil {
		r.Status = "fail"
		r.Detail = fmt.Sprintf("invalid .codex/config.toml: %v", err)
		return r
	}

	hooksFeatureEnabled := false
	if features, ok := raw["features"].(map[string]any); ok {
		if v, ok := features["hooks"].(bool); ok {
			hooksFeatureEnabled = v
		}
	}

	hookEntries := 0
	hasInlineHookRepresentation := false
	if hooks, ok := raw["hooks"].(map[string]any); ok {
		hasInlineHookRepresentation = true
		for _, eventHooks := range hooks {
			switch v := eventHooks.(type) {
			case []any:
				hookEntries += len(v)
			case map[string]any:
				if len(v) > 0 {
					hookEntries++
				}
			}
		}
	}

	hooksJSONPath := filepath.Join(targetDir, ".codex", "hooks.json")
	if hasInlineHookRepresentation {
		if _, err := os.Stat(hooksJSONPath); err == nil {
			r.Status = "fail"
			r.Detail = "both .codex/config.toml [hooks] and .codex/hooks.json exist; remove hooks.json because this project uses config.toml as the Codex hook source of truth"
			return r
		} else if !errors.Is(err, fs.ErrNotExist) {
			r.Status = "warn"
			r.Detail = fmt.Sprintf("could not inspect .codex/hooks.json: %v", err)
			return r
		}
	}

	switch {
	case !hooksFeatureEnabled:
		r.Status = "warn"
		r.Detail = "[features] hooks = true is not set; project hooks will be ignored"
	case hookEntries == 0:
		r.Status = "warn"
		r.Detail = "no [hooks.*] entries — hooks feature enabled but nothing wired up. Run `codex trust .` once configured"
	default:
		r.Status = "pass"
		r.Detail = fmt.Sprintf("hooks=true, %d hook entry(ies). Confirm `codex trust .` ran for this project", hookEntries)
	}
	return r
}

func checkHooks(targetDir string) checkResult {
	r := checkResult{Name: "Hooks integrity"}
	settingsPath := filepath.Join(targetDir, ".claude", "settings.json")
	data, err := os.ReadFile(settingsPath)
	if err != nil {
		r.Status = "warn"
		r.Detail = "settings.json not found"
		return r
	}

	var settings map[string]any
	if err := json.Unmarshal(data, &settings); err != nil {
		r.Status = "fail"
		r.Detail = "invalid settings.json"
		return r
	}

	hooks, ok := settings["hooks"]
	if !ok {
		r.Status = "warn"
		r.Detail = "no hooks configured"
		return r
	}

	// Check that hook script files exist.
	hooksMap, ok := hooks.(map[string]any)
	if !ok {
		r.Status = "pass"
		return r
	}

	missing := 0
	for _, eventHooks := range hooksMap {
		eventList, ok := eventHooks.([]any)
		if !ok {
			continue
		}
		for _, eh := range eventList {
			ehMap, ok := eh.(map[string]any)
			if !ok {
				continue
			}
			hooksList, ok := ehMap["hooks"].([]any)
			if !ok {
				continue
			}
			for _, h := range hooksList {
				hMap, ok := h.(map[string]any)
				if !ok {
					continue
				}
				cmd, ok := hMap["command"].(string)
				if !ok {
					continue
				}
				if _, err := os.Stat(filepath.Join(targetDir, cmd)); errors.Is(err, fs.ErrNotExist) {
					missing++
				}
			}
		}
	}

	if missing > 0 {
		r.Status = "fail"
		r.Detail = fmt.Sprintf("%d hook script(s) missing", missing)
	} else {
		r.Status = "pass"
	}
	return r
}

func checkManifestVersion(targetDir string) checkResult {
	r := checkResult{Name: "Manifest version"}
	manifestPath := filepath.Join(targetDir, ".ralph", "manifest.toml")
	m, err := scaffold.ReadManifest(manifestPath)
	if err != nil {
		r.Status = "warn"
		r.Detail = "no manifest found"
		return r
	}

	if m.Meta.Version == Version {
		r.Status = "pass"
		r.Detail = m.Meta.Version
	} else {
		r.Status = "warn"
		r.Detail = fmt.Sprintf("manifest %s ≠ CLI %s — run 'ralph upgrade'", m.Meta.Version, Version)
	}
	return r
}

// checkInstalledPacks checks packs that are actually installed in the project
// (detected from manifest), not just what's available in embedded templates.
func checkInstalledPacks(targetDir string) []checkResult {
	manifestPath := filepath.Join(targetDir, ".ralph", "manifest.toml")
	m, err := scaffold.ReadManifest(manifestPath)
	if err != nil {
		// No manifest — fall back to checking embedded packs.
		return checkEmbeddedPacks()
	}

	// Use Meta.Packs as the authoritative installed-pack list.
	// Both `ralph init` (init.go) and `ralph pack add` (pack.go) maintain this
	// field, so it reliably reflects which packs were installed.
	// The previous approach walked PackFS and probed m.Files[pack-root-relative
	// path], but manifest keys are namespaced (e.g. "packs/languages/golang/…"),
	// so the lookup always missed and reported "none installed".
	if len(m.Meta.Packs) == 0 {
		return []checkResult{{Name: "Language packs", Status: "pass", Detail: "none installed"}}
	}

	var results []checkResult
	for _, p := range m.Meta.Packs {
		r := checkResult{Name: fmt.Sprintf("Pack: %s", p)}
		// verify.sh lives under packs/languages/<lang>/verify.sh on disk.
		// The previous code probed filepath.Join(targetDir, "verify.sh") (project
		// root), which always produced a misleading "not found on disk" warning.
		verifyPath := filepath.Join(targetDir, packRelDir(p), "verify.sh")
		packFS, pErr := scaffold.PackFS(p)
		if pErr != nil {
			r.Status = "warn"
			r.Detail = "pack not found in templates"
			results = append(results, r)
			continue
		}
		if _, fErr := packFS.Open("verify.sh"); fErr != nil {
			r.Status = "warn"
			r.Detail = "verify.sh missing in template"
		} else if _, sErr := os.Stat(verifyPath); errors.Is(sErr, fs.ErrNotExist) {
			r.Status = "warn"
			r.Detail = fmt.Sprintf("verify.sh not found on disk: %s", verifyPath)
		} else {
			r.Status = "pass"
		}
		results = append(results, r)
	}
	return results
}

// checkEmbeddedPacks is the fallback when no manifest exists.
func checkEmbeddedPacks() []checkResult {
	packs, err := scaffold.AvailablePacks()
	if err != nil {
		return []checkResult{{Name: "Language packs", Status: "warn", Detail: "could not list packs"}}
	}

	var results []checkResult
	for _, p := range packs {
		r := checkResult{Name: fmt.Sprintf("Pack: %s", p)}
		packFS, pErr := scaffold.PackFS(p)
		if pErr != nil {
			r.Status = "warn"
			r.Detail = "pack not found"
			results = append(results, r)
			continue
		}
		if _, fErr := packFS.Open("verify.sh"); fErr != nil {
			r.Status = "warn"
			r.Detail = "verify.sh missing"
		} else {
			r.Status = "pass"
		}
		results = append(results, r)
	}
	return results
}

// checkLoopDriver reports the effective Ralph Loop driver — the value
// ralph-pipeline.sh will actually see — and which source it came from
// (env > TOML > default). Implements AC-6 of issue #44. The lookup function
// is injected so the test can supply a deterministic env without
// monkey-patching os.Getenv.
//
// Issue #44 cycle-3 cross-review hardenings:
//   - When driver=codex is effective but the codex binary is absent, return
//     fail. Otherwise doctor reports pass while the next `ralph run`
//     preflight blocks immediately on the missing required CLI.
//   - Sandbox / approval / reviewer-model values are resolved through the
//     same env > TOML > default priority before display, so an operator
//     who exports RALPH_CODEX_SANDBOX=danger-full-access does not see
//     `sandbox: workspace-write` in doctor and assume the safer default.
func checkLoopDriver(cfg config.Config, getenv func(string) string) checkResult {
	r := checkResult{Name: "Loop driver"}

	defaults := config.Default().Loop
	pick := func(envKey, tomlVal, defaultVal string) (string, string) {
		if v := getenv(envKey); v != "" {
			return v, "env"
		}
		if tomlVal != "" {
			// TOML matching default still reports toml as the source so users
			// who explicitly write `driver = "claude"` see their choice acknowledged.
			return tomlVal, "toml"
		}
		return defaultVal, "default"
	}

	effective, source := pick("RALPH_LOOP_DRIVER", cfg.Loop.Driver, defaults.Driver)
	sandbox, _ := pick("RALPH_CODEX_SANDBOX", cfg.Loop.CodexSandbox, defaults.CodexSandbox)
	approval, _ := pick("RALPH_CODEX_APPROVAL_POLICY", cfg.Loop.CodexApprovalPolicy, defaults.CodexApprovalPolicy)
	reviewer, _ := pick("RALPH_CLAUDE_REVIEWER_MODEL", cfg.Loop.ClaudeReviewerModel, defaults.ClaudeReviewerModel)

	if effective == "codex" {
		if _, err := exec.LookPath("codex"); err != nil {
			r.Status = "fail"
			r.Detail = fmt.Sprintf("%s (source: %s) — codex binary not found in PATH; `ralph run` preflight will fail", effective, source)
			return r
		}
	}

	r.Status = "pass"
	if effective == "codex" {
		r.Detail = fmt.Sprintf("%s (source: %s, sandbox: %s, approval: %s, reviewer: claude/%s)",
			effective, source, sandbox, approval, reviewer)
	} else {
		r.Detail = fmt.Sprintf("%s (source: %s)", effective, source)
	}
	return r
}

// checkStaleOrchestratorState warns when .harness/state/orchestrator/orchestrator.json
// has status "running" but the file has not been updated in more than 24 hours,
// indicating a crashed or abandoned run that never cleaned up its state.
func checkStaleOrchestratorState(targetDir string) checkResult {
	r := checkResult{Name: "Orchestrator state"}

	orchPath := filepath.Join(targetDir, ".harness", "state", "orchestrator", "orchestrator.json")
	fi, err := os.Stat(orchPath)
	if errors.Is(err, fs.ErrNotExist) {
		// No state file — nothing to warn about.
		r.Status = "pass"
		return r
	}
	if err != nil {
		r.Status = "warn"
		r.Detail = fmt.Sprintf("could not stat orchestrator.json: %v", err)
		return r
	}

	data, err := os.ReadFile(orchPath)
	if err != nil {
		r.Status = "warn"
		r.Detail = fmt.Sprintf("could not read orchestrator.json: %v", err)
		return r
	}

	var raw struct {
		Status  string `json:"status"`
		Started string `json:"started"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		// Malformed JSON — not our concern here, pass silently.
		r.Status = "pass"
		return r
	}

	if raw.Status != "running" {
		r.Status = "pass"
		return r
	}

	mtime := fi.ModTime()
	age := time.Since(mtime)
	if age <= 24*time.Hour {
		// File updated recently — running state is likely live.
		r.Status = "pass"
		return r
	}

	// Stale running state: file untouched for >24h.
	ageH := int(age.Hours())
	detail := fmt.Sprintf(
		"stale orchestrator state (running since %s, file untouched for %dh); "+
			"a crashed run may have left it behind — run './scripts/ralph abort' or "+
			"delete .harness/state/orchestrator/orchestrator.json",
		raw.Started, ageH,
	)
	r.Status = "warn"
	r.Detail = detail
	return r
}

func checkGo(cfg config.Config) checkResult {
	r := checkResult{Name: "Go"}
	_, err := exec.LookPath("go")
	if err != nil {
		if cfg.Doctor.RequireGo {
			r.Status = "fail"
			r.Detail = "go not found in PATH"
		} else {
			r.Status = "pass"
			r.Detail = "not required"
		}
	} else {
		r.Status = "pass"
	}
	return r
}

// checkHerdrAvailable reports whether the herdr CLI (org runtime seat driver)
// is on PATH. Org runtime is purely additive (AC-9): a project that never
// runs `ralph org` has no reason to install herdr, so absence is reported as
// "info", not "warn"/"fail" — it must never change runDoctorOpts' exit code.
func checkHerdrAvailable() checkResult {
	r := checkResult{Name: "herdr"}
	if err := driver.HerdrAvailable(); err != nil {
		r.Status = "info"
		r.Detail = "herdr not installed — org runtime seats unavailable (solo execution unaffected)"
		return r
	}
	r.Status = "pass"
	r.Detail = "available"
	return r
}

// checkAgmsgAvailable is checkHerdrAvailable's counterpart for the agmsg CLI.
func checkAgmsgAvailable() checkResult {
	r := checkResult{Name: "agmsg"}
	if err := driver.AgmsgAvailable(); err != nil {
		r.Status = "info"
		r.Detail = "agmsg not installed — org runtime seats unavailable (solo execution unaffected)"
		return r
	}
	r.Status = "pass"
	r.Detail = "available"
	return r
}

// checkOrgEnvelope summarizes the loaded [org] envelope (model_pool size,
// max_seats) as an informational line. It never re-loads config — the caller
// already resolved cfg via config.Load, and a load/parse failure is already
// surfaced by the pre-existing "ralph.toml" check in runDoctorOpts, so this
// check does not duplicate that reporting.
func checkOrgEnvelope(cfg config.Config) checkResult {
	return checkResult{
		Name:   "Org envelope",
		Status: "info",
		Detail: fmt.Sprintf("model_pool: %d entries, max_seats: %d", len(cfg.Org.ModelPool), cfg.Org.MaxSeats),
	}
}

// checkOrgModelProbes runs driver.ProbeModel for every [org].model_pool
// entry, grouped by driver so a single missing CLI produces one skip line
// instead of one per model. Each probe is bounded by a 30s context so a hung
// CLI cannot wedge `ralph doctor --probe-models`.
//
// codex `--model` support on `codex exec` is best-effort upstream (see
// driver.ProbeModel), so a codex probe failure is reported as "warn" with an
// explicit "advisory" label rather than being treated as a hard rejection.
func checkOrgModelProbes(cfg config.Config, runner driver.Runner) []checkResult {
	var results []checkResult

	byDriver := make(map[string][]string, len(cfg.Org.DriverPool))
	var driverOrder []string
	for _, entry := range cfg.Org.ModelPool {
		if _, seen := byDriver[entry.Driver]; !seen {
			driverOrder = append(driverOrder, entry.Driver)
		}
		byDriver[entry.Driver] = append(byDriver[entry.Driver], entry.Model)
	}

	for _, drv := range driverOrder {
		models := byDriver[drv]
		if _, err := exec.LookPath(drv); err != nil {
			results = append(results, checkResult{
				Name:   fmt.Sprintf("Org model probe (%s)", drv),
				Status: "info",
				Detail: fmt.Sprintf("%s not installed — skipping %d model probe(s)", drv, len(models)),
			})
			continue
		}
		for _, model := range models {
			results = append(results, probeOrgModel(runner, drv, model))
		}
	}
	return results
}

// probeOrgModel runs a single ProbeModel call under a bounded timeout and
// translates the result into a checkResult.
func probeOrgModel(runner driver.Runner, drv, model string) checkResult {
	r := checkResult{Name: fmt.Sprintf("Org model probe (%s/%s)", drv, model)}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := driver.ProbeModel(ctx, runner, drv, model); err != nil {
		r.Status = "warn"
		if drv == "codex" {
			r.Detail = fmt.Sprintf("advisory: codex --model support on exec is best-effort upstream; probe failed: %v", err)
		} else {
			r.Detail = fmt.Sprintf("probe failed: %v", err)
		}
		return r
	}
	r.Status = "pass"
	return r
}
