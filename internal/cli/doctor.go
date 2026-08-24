package cli

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	toml "github.com/pelletier/go-toml/v2"
	"github.com/spf13/cobra"

	"github.com/yoshpy-dev/ralph/internal/config"
	"github.com/yoshpy-dev/ralph/internal/org/driver"
	"github.com/yoshpy-dev/ralph/internal/scaffold"
	"github.com/yoshpy-dev/ralph/internal/upgrade"
)

func newDoctorCmd() *cobra.Command {
	var probeModels, strict bool
	cmd := &cobra.Command{
		Use:   "doctor",
		Short: "Check environment and project setup",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runDoctorFull(".", probeModels, strict)
		},
	}
	cmd.Flags().BoolVar(&probeModels, "probe-models", false,
		"probe every [org].model_pool entry by launching a minimal CLI invocation per model (slower; requires claude/codex on PATH)")
	cmd.Flags().BoolVar(&strict, "strict", false,
		"fail (exit 1) on FR-9 scaffold-integrity violations (core file hashes, managed block health, "+
			"settings.json owned keys, conflict markers, manifest/disk consistency — plus the meta-failures "+
			"that make any of those checks impossible to run: an unparseable manifest, an ownership-planning "+
			"error such as an unreadable tracked file, or a per-check computation error (e.g. an unreadable "+
			"block surface, an invalid-JSON settings.json)); without --strict the "+
			"same findings are printed as warnings and doctor's exit code is unaffected. --strict only elevates "+
			"these scaffold checks — every other doctor check (e.g. missing claude/codex CLI) keeps its "+
			"existing pass/warn/fail semantics. Note: (b) managed blocks and (c) settings.json owned keys "+
			"compare disk content against the CURRENT BINARY's embedded templates, so after upgrading the "+
			"ralph binary itself, run `ralph upgrade` first, then `doctor --strict` — otherwise a pending, "+
			"not-yet-applied template update fails --strict even though nothing is actually broken. (a) core "+
			"file hashes deliberately tolerates that same pending-update state (FR-4) and does not flag it; "+
			"this asymmetry between (a) and (b)/(c) is intentional, not a bug")
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
// 11). Model probes are the only check in this function that spawn CLI
// subprocesses beyond the pre-existing --version probes, so they stay
// opt-in and off by default.
//
// Signature/behavior preserved exactly (never runs under --strict, always
// strict=false) so pre-existing callers/tests that predate FR-9 doctor
// --strict (docs/specs/2026-08-17-overlay-scaffold-v2.md) keep working
// unchanged, mirroring how runDoctor itself wraps this function.
// runDoctorFull is the --strict-aware superset newDoctorCmd calls directly.
func runDoctorOpts(targetDir string, probeModels bool) error {
	return runDoctorFull(targetDir, probeModels, false)
}

// runDoctorFull is runDoctorOpts plus the --strict opt-in (FR-9, Phase 5):
// when true, violations found by checkScaffoldIntegrity's five FR-9
// sub-checks are reported as "fail" (and therefore flip doctor's exit code)
// instead of "warn". strict never affects any other check in this function.
func runDoctorFull(targetDir string, probeModels, strict bool) error {
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

	// Check 3: Codex hook wiring ([features] hooks; hooks.json schema + dispatcher routing; stale config.toml [hooks] table).
	results = append(results, checkCodexEffectiveConfig(targetDir))

	// Check 4: Hooks integrity.
	results = append(results, checkHooks(targetDir))

	// Check 5: Manifest version.
	results = append(results, checkManifestVersion(targetDir))

	// Check 6: Language pack verify.sh (checks project's installed packs via manifest).
	results = append(results, checkInstalledPacks(targetDir)...)

	// Check 7: Go availability.
	results = append(results, checkGo(cfg))

	// Check 8: herdr availability (org runtime driver adapter).
	results = append(results, checkHerdrAvailable())

	// Check 9: agmsg availability (org runtime driver adapter).
	results = append(results, checkAgmsgAvailable(driver.ResolveAgmsgHome(cfg.Org.AgmsgHome)))

	// Check 10: [org] envelope summary (pool size / max_seats).
	results = append(results, checkOrgEnvelope(cfg))

	// Check 11: optional model-pool probes (--probe-models).
	if probeModels {
		results = append(results, checkOrgModelProbes(cfg, driver.ExecRunner{})...)
	}

	// Check 12: FR-9 scaffold integrity (core hashes, managed blocks,
	// settings.json owned keys, conflict markers, manifest/disk
	// consistency). Always runs (findings are warnings by default); strict
	// controls only whether a violation is reported as "fail". See
	// checkScaffoldIntegrity's doc comment for the no-manifest/legacy-manifest
	// short-circuits.
	results = append(results, checkScaffoldIntegrity(targetDir, strict)...)

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
	for line := range strings.SplitSeq(string(out), "\n") {
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

// checkCodexEffectiveConfig confirms that Codex's project-level hook wiring
// is structurally sound. `.codex/hooks.json` is the source of truth for
// Codex hooks (live-fire evidence:
// docs/evidence/codex-hooks-livefire-slice1-2026-08-20.md — on the codex-cli
// release this was verified against, inline `.codex/config.toml`
// `[[hooks.*]]` entries never fired while the equivalent hooks.json entry
// did). This check therefore validates hooks.json's schema and dispatcher
// routing, and flags a config.toml that still carries `[hooks]`/
// `[[hooks.*]]` tables as a stale duplicate representation. We cannot probe
// Codex's trust state from Go, so a structurally sound result still reminds
// the operator to confirm `codex trust .` via the .codex/README.md
// guidance.
//
// This is an environment check (warn-level findings), not part of FR-9
// scaffold integrity. `--strict` never escalates these findings; the only
// `fail` this check produces is an unparseable `config.toml`.
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

	var findings []string

	// [features] hooks: an explicit `false` disables Codex project hooks
	// outright, worth a warn. An absent key is left lenient — the official
	// hooks doc does not specify a default when the key is omitted, so
	// doctor does not assume either value (leniency decision recorded in
	// docs/evidence/codex-hooks-livefire-slice1-2026-08-20.md). A present
	// but non-boolean value (e.g. a quoted "false") is a third, distinct
	// state — also a warn, since Codex may not enable hooks with a
	// malformed value and silently passing here would misreport
	// enablement.
	if features, ok := raw["features"].(map[string]any); ok {
		if hv, present := features["hooks"]; present {
			if b, isBool := hv.(bool); isBool {
				if !b {
					findings = append(findings, "[features] hooks = false — Codex project hooks are disabled")
				}
			} else {
				// Present but not a boolean (e.g. hooks = "false" typo):
				// distinct from the lenient absent-key case — Codex may
				// not enable hooks with a malformed value, so silently
				// passing here would misreport enablement (cross-review
				// AR#1, cycle 1,
				// docs/reports/cross-review-triage-codex-hooks-json-wiring.md).
				// No %T in the message: raw Go type names leak internals,
				// the same readability call validateCodexHooksJSON makes.
				findings = append(findings, "[features] hooks must be a boolean — a quoted or otherwise non-boolean value may leave hooks disabled; use `hooks = true`")
			}
		}
	}

	// A surviving config.toml [hooks]/[[hooks.*]] table is a stale
	// duplicate representation now that hooks.json is the source of truth.
	if _, ok := raw["hooks"].(map[string]any); ok {
		findings = append(findings, "both representations exist; hooks.json is the source of truth — remove the config.toml [hooks] entries")
	}

	hooksJSONPath := filepath.Join(targetDir, ".codex", "hooks.json")
	hjData, err := os.ReadFile(hooksJSONPath)
	switch {
	case errors.Is(err, fs.ErrNotExist):
		findings = append(findings, ".codex/hooks.json not found — Codex project hooks are not wired")
	case err != nil:
		findings = append(findings, fmt.Sprintf("could not read .codex/hooks.json: %v", err))
	default:
		findings = append(findings, validateCodexHooksJSON(hjData)...)
	}

	if len(findings) > 0 {
		r.Status = "warn"
		r.Detail = strings.Join(findings, "; ")
		return r
	}

	r.Status = "pass"
	r.Detail = "hooks.json wired; confirm `codex trust .` ran for this project"
	return r
}

// hookScriptBasenameRe extracts "*.sh" basenames out of a hooks.json command
// string, mirroring tests/test-hook-wiring.sh's
// check_no_direct_hook_scripts_in_hooks_json grep -oE pattern — used by
// validateCodexHooksJSON to flag a command that bypasses ralph-dispatch.sh
// by calling another hook script directly (C3-M1, cycle 3,
// docs/reports/self-review-2026-08-20-codex-hooks-json-wiring.md). Keeping
// this check in Go (not just the shell test) means a scaffolded project —
// which does not ship tests/ — still gets it from `ralph doctor`.
var hookScriptBasenameRe = regexp.MustCompile(`[A-Za-z0-9_.-]+\.sh`)

// codexShippedHookEvents is the set of Codex hooks.json events ralph ships
// dispatcher routing for (.codex/hooks.json + templates/base/.codex/hooks.json).
// validateCodexHooksJSON checks that each of these has at least one handler
// routed through ralph-dispatch.sh. Keep in sync with the shipped hooks.json
// event set (docs/plans/active/2026-08-24-codex-hooks-multi-event.md) —
// SessionEnd/PreCompact are deliberately not included (see that plan's
// non-goals and docs/tech-debt/README.md).
var codexShippedHookEvents = []string{"PostToolUse", "PreToolUse", "SessionStart", "UserPromptSubmit"}

// dispatchEventArgRe holds one compiled regexp per codexShippedHookEvents
// entry, each requiring "ralph-dispatch.sh <event>" with the event name
// immediately followed by a word boundary (non-alphanumeric/underscore, or
// end of string). A per-event "routed" flag must confirm the dispatcher was
// invoked with the MATCHING event argument, not just referenced by basename
// — a PreToolUse entry whose command ends in "ralph-dispatch.sh PostToolUse"
// is a mis-wiring, not valid routing for PreToolUse, and the boundary check
// keeps "ralph-dispatch.sh PostToolUse" from falsely matching a command that
// actually invokes "ralph-dispatch.sh PostToolUseExtra".
var dispatchEventArgRe = func() map[string]*regexp.Regexp {
	res := make(map[string]*regexp.Regexp, len(codexShippedHookEvents))
	for _, ev := range codexShippedHookEvents {
		res[ev] = regexp.MustCompile(`ralph-dispatch\.sh\s+` + regexp.QuoteMeta(ev) + `(?:[^A-Za-z0-9_]|$)`)
	}
	return res
}()

// validateCodexHooksJSON checks hooksJSONData against the official Codex
// hooks.json schema (top-level "hooks" -> event-name keys -> matcher-group
// arrays -> {"type":"command","command":<string>} handlers), confirms that
// every event in codexShippedHookEvents has at least one handler routed
// through ralph-dispatch.sh AND invoked with that event's own name as the
// dispatcher argument (the layered .d dispatcher Claude Code also uses; see
// dispatchEventArgRe), and flags any command that references a *.sh script
// other than ralph-dispatch.sh directly (a dispatcher bypass — C3-M1). Every
// defect is reported as a distinct finding string; callers treat a
// non-empty result as a warn, never a fail — this mirrors
// checkCodexEffectiveConfig's warn-level environment-check contract (not
// part of FR-9 scaffold integrity, so never --strict-eligible).
func validateCodexHooksJSON(hooksJSONData []byte) []string {
	var doc map[string]any
	if err := json.Unmarshal(hooksJSONData, &doc); err != nil {
		return []string{fmt.Sprintf("invalid JSON in .codex/hooks.json: %v", err)}
	}

	hooksRaw, ok := doc["hooks"]
	if !ok {
		return []string{`hooks.json is missing the top-level "hooks" key (schema: {"hooks": {"<event>": [...]}})`}
	}
	events, ok := hooksRaw.(map[string]any)
	if !ok {
		return []string{`hooks.json "hooks" value must be an object keyed by event name`}
	}

	var findings []string
	dispatcherRoutedByEvent := make(map[string]bool, len(codexShippedHookEvents))
	for _, eventName := range codexShippedHookEvents {
		dispatcherRoutedByEvent[eventName] = false
	}

	for eventName, groupsRaw := range events {
		groups, ok := groupsRaw.([]any)
		if !ok {
			findings = append(findings, fmt.Sprintf("hooks.json %q must be an array of matcher groups", eventName))
			continue
		}
		for i, groupRaw := range groups {
			group, ok := groupRaw.(map[string]any)
			if !ok {
				findings = append(findings, fmt.Sprintf("hooks.json %s[%d] must be an object with a \"hooks\" array", eventName, i))
				continue
			}
			handlersRaw, ok := group["hooks"]
			if !ok {
				findings = append(findings, fmt.Sprintf("hooks.json %s[%d] is missing its \"hooks\" array", eventName, i))
				continue
			}
			handlers, ok := handlersRaw.([]any)
			if !ok {
				findings = append(findings, fmt.Sprintf("hooks.json %s[%d].hooks must be an array", eventName, i))
				continue
			}
			for j, handlerRaw := range handlers {
				handler, ok := handlerRaw.(map[string]any)
				if !ok {
					findings = append(findings, fmt.Sprintf("hooks.json %s[%d].hooks[%d] must be an object", eventName, i, j))
					continue
				}
				typeVal, hasType := handler["type"].(string)
				if !hasType || typeVal != "command" {
					findings = append(findings, fmt.Sprintf(`hooks.json %s[%d].hooks[%d] is missing "type": "command"`, eventName, i, j))
					continue
				}
				cmdRaw, hasCmd := handler["command"]
				if !hasCmd {
					findings = append(findings, fmt.Sprintf(`hooks.json %s[%d].hooks[%d] is missing "command"`, eventName, i, j))
					continue
				}
				cmdVal, isString := cmdRaw.(string)
				if !isString {
					kind := fmt.Sprintf("%T", cmdRaw)
					if _, isArr := cmdRaw.([]any); isArr {
						kind = "array"
					}
					findings = append(findings, fmt.Sprintf(`hooks.json %s[%d].hooks[%d] "command" must be a string (got %s)`, eventName, i, j, kind))
					continue
				}
				if cmdVal == "" {
					findings = append(findings, fmt.Sprintf(`hooks.json %s[%d].hooks[%d] "command" must not be empty`, eventName, i, j))
					continue
				}
				if re, shipped := dispatchEventArgRe[eventName]; shipped && re.MatchString(cmdVal) {
					dispatcherRoutedByEvent[eventName] = true
				}
				for _, name := range hookScriptBasenameRe.FindAllString(cmdVal, -1) {
					if name == "ralph-dispatch.sh" {
						continue
					}
					findings = append(findings, fmt.Sprintf(
						"hooks.json %s[%d].hooks[%d] \"command\" references %q directly instead of routing through ralph-dispatch.sh",
						eventName, i, j, name))
				}
			}
		}
	}

	for _, eventName := range codexShippedHookEvents {
		if !dispatcherRoutedByEvent[eventName] {
			findings = append(findings, fmt.Sprintf("hooks.json %s has no handler routed through ralph-dispatch.sh", eventName))
		}
	}

	return findings
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
				// The command may carry arguments (e.g. "./path/to/script.sh
				// EventName"); only the first whitespace-separated token is
				// the executable, so stat that instead of the full string.
				fields := strings.Fields(cmd)
				if len(fields) == 0 {
					continue
				}
				exe := fields[0]
				if _, err := os.Stat(filepath.Join(targetDir, exe)); errors.Is(err, fs.ErrNotExist) {
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

// agmsgTestedVersion is the agmsg script interface version this adapter
// (internal/org/driver/agmsg.go) was verified against. A different VERSION
// found at runtime is surfaced as an informational note only -- doctor never
// fails or warns on a version mismatch, since the script interface's actual
// stability is unknown for versions ralph hasn't been tested against.
const agmsgTestedVersion = "1.1.13"

// checkAgmsgAvailable is checkHerdrAvailable's counterpart for the agmsg
// script collection. Unlike herdr, agmsg is NOT a single binary on PATH --
// home is the resolved agmsg home directory (driver.ResolveAgmsgHome), and
// availability is decided by the presence of home's scripts/send.sh.
// checkAgmsgAvailable's Detail always names home explicitly (both the
// not-installed and available branches) -- tech-debt fix: this check used
// to discard driver.AgmsgAvailable's resolved home in favor of a fixed
// string, so a misconfigured 3-surface `agmsg_home` (env/config/default)
// gave no clue which directory doctor actually checked
// (docs/tech-debt/README.md, "checkAgmsgAvailable discards the resolved
// agmsg home from driver.AgmsgAvailable's error").
func checkAgmsgAvailable(home string) checkResult {
	r := checkResult{Name: "agmsg"}
	if err := driver.AgmsgAvailable(home); err != nil {
		r.Status = "info"
		r.Detail = fmt.Sprintf("agmsg not installed at %s — org runtime seats unavailable (solo execution unaffected)", home)
		return r
	}
	r.Status = "pass"
	r.Detail = fmt.Sprintf("available at %s", home)
	version, err := driver.AgmsgVersion(home)
	if err != nil {
		return r
	}
	r.Detail = fmt.Sprintf("available at %s (version %s)", home, version)
	if version != agmsgTestedVersion {
		r.Status = "info"
		r.Detail = fmt.Sprintf("available at %s (version %s; tested against %s — behavior differences are possible)",
			home, version, agmsgTestedVersion)
	}
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

// --- FR-9 scaffold integrity (docs/specs/2026-08-17-overlay-scaffold-v2.md,
// FR-9; Phase 5 plan Scope item 3) ---
//
// scaffoldViolationStatus maps a boolean violation outcome to the checkResult
// status every FR-9 sub-check below uses: a clean check is always "pass"; a
// violation is "fail" under --strict (elevates doctor's exit code) and
// "warn" otherwise (findings still print, exit code untouched) — this is
// the single place that encodes the --strict elevation boundary so it can
// never drift between the five sub-checks.
func scaffoldViolationStatus(violated, strict bool) string {
	switch {
	case !violated:
		return "pass"
	case strict:
		return "fail"
	default:
		return "warn"
	}
}

// checkScaffoldIntegrity runs the FR-9 scaffold-integrity checks: (a) core
// file hashes vs the embedded templates (fork paths excluded),
// (b) managed-block marker/content health, (c) settings.json ralph-owned
// key sync, (d) absence of unresolved conflict markers in tracked files, and
// (e) manifest/disk consistency.
//
// Two short-circuits keep this additive over pre-FR-9 doctor behavior:
//   - No manifest at all (not a ralph project — os.IsNotExist) returns nil:
//     zero results, so non-project directories are unaffected.
//   - A legacy (pre-v2) manifest returns exactly one "info"-status result
//     advising `ralph upgrade`. This is deliberately never a --strict
//     failure: a legacy layout is not itself a violation of the v2
//     ownership contracts FR-9 checks against (the same distinction
//     eject/adopt/pack add draw via requireV2ManifestForOwnership's
//     errLegacyLayoutFailClosed) — it just means FR-9's checks do not apply
//     yet.
//
// A manifest that exists but fails to parse (corrupt TOML, truncated
// write) is neither of the above: it is a strict-eligible violation in its
// own right, reported via scaffoldViolationStatus like every other FR-9
// sub-check below. Folding this case into the "no manifest" nil-return (as
// an earlier version of this function did, reasoning that the pre-existing
// "Manifest version" check already surfaces it) made --strict fail open
// exactly when the manifest is least trustworthy: that check reports
// "warn" with the misleading detail "no manifest found" and never affects
// doctor's exit code, so a corrupted manifest silently disabled every one
// of FR-9's five checks under --strict.
func checkScaffoldIntegrity(targetDir string, strict bool) []checkResult {
	manifestPath := filepath.Join(targetDir, ".ralph", "manifest.toml")
	manifest, err := scaffold.ReadManifest(manifestPath)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil
		}
		return []checkResult{{
			Name:   "Scaffold integrity",
			Status: scaffoldViolationStatus(true, strict),
			Detail: fmt.Sprintf(".ralph/manifest.toml exists but could not be read: %v", err),
		}}
	}
	if manifest.Meta.Layout != scaffold.LayoutV2 {
		return []checkResult{{
			Name:   "Scaffold integrity",
			Status: "info",
			Detail: "legacy (pre-v2) manifest layout; run `ralph upgrade` to migrate before FR-9 scaffold checks apply",
		}}
	}

	// A filepath.Abs failure (only possible when os.Getwd itself fails) is
	// the same class of meta-failure as the corrupt-manifest and
	// ownership-planning-error cases above: every one of FR-9's five
	// sub-checks needs absDir, so this is verification-impossible, not a
	// reason to fail open (class-closing audit, cycle 2,
	// docs/reports/cross-review-triage-overlay-scaffold-v2-p5.md).
	absDir, err := filepath.Abs(targetDir)
	if err != nil {
		return []checkResult{{
			Name:   "Scaffold integrity",
			Status: scaffoldViolationStatus(true, strict),
			Detail: fmt.Sprintf("resolving target dir: %v", err),
		}}
	}

	// resolveOwnershipPlan (adopt.go) runs the exact same read-only
	// PlanCoreReplaceDesired classification `ralph upgrade`/eject/adopt use,
	// so FR-9's notion of "drift"/"fork" can never drift out of sync with
	// those commands. Warnings buildDesiredStateV2 may emit (e.g. an
	// unavailable pack) are discarded here — doctor's own checks below
	// report their own findings at the right granularity instead.
	// A planning-error here (e.g. a tracked core path replaced by a
	// directory, so PlanCoreReplaceDesired's disk read fails with "is a
	// directory") is a strict-eligible violation in its own right, reported
	// via scaffoldViolationStatus like every other FR-9 sub-check below —
	// the scaffold cannot even be classified, which is the worst case FR-9
	// is meant to catch, not a reason to fail open. This mirrors the
	// corrupt-manifest handling above (M2). Contrast with status.go's own
	// resolveOwnershipPlan call (buildScaffoldStatus, M7): that caller
	// degrades gracefully and keeps rendering the rest of `ralph status`
	// because status is a best-effort report, whereas `doctor --strict` is a
	// gate — the same failure means two different things depending on which
	// command hit it (AR#1, cycle 1,
	// docs/reports/cross-review-triage-overlay-scaffold-v2-p5.md).
	desired, plan, err := resolveOwnershipPlan(absDir, manifest, io.Discard)
	if err != nil {
		return []checkResult{{
			Name:   "Scaffold integrity",
			Status: scaffoldViolationStatus(true, strict),
			Detail: fmt.Sprintf("computing ownership plan: %v", err),
		}}
	}

	return []checkResult{
		checkScaffoldCoreHashes(plan, strict),
		checkManagedBlocks(absDir, desired, strict),
		checkSettingsOwnedKeys(absDir, desired, strict),
		checkConflictMarkers(absDir, manifest, strict),
		checkManifestConsistency(absDir, manifest, strict),
	}
}

// checkScaffoldCoreHashes is FR-9(a): every core-owned path's disk content
// must match the embedded template it was generated from, with fork paths
// excluded. plan.Drift is exactly this population — PlanCoreReplaceDesired
// (internal/upgrade/replaceplan.go) only ever puts a path in Drift when its
// disk content diverges from both the manifest-recorded hash and the
// current template hash with no fork record to explain the divergence; a
// forked path is classified into plan.Advisories instead (see
// driftPathSet's doc comment, adopt.go), so fork content is never flagged
// here — matching FR-9(a)'s "fork 除く". plan.Drift also carries untracked
// collisions (no manifest entry), annotated separately below.
//
// A pending-but-expected update (unmodified core file, template changed
// upstream — plan.Ops OpUpdate) is not a violation: disk still matches its
// manifest-recorded hash exactly, so there is nothing unresolved, only a
// `ralph upgrade` waiting to be run.
func checkScaffoldCoreHashes(plan upgrade.ReplacePlan, strict bool) checkResult {
	r := checkResult{Name: "Scaffold: core file hashes"}
	violated := len(plan.Drift) > 0
	r.Status = scaffoldViolationStatus(violated, strict)
	if !violated {
		return r
	}
	paths := make([]string, 0, len(plan.Drift))
	hasUntracked := false
	for _, d := range plan.Drift {
		// plan.Drift can include genuinely untracked paths (no manifest
		// entry — classifyUntracked's core branch); eject/adopt reject
		// those, so the guidance must distinguish them the same way the
		// upgrade-side writeDriftGuidanceV2 does (C4-M1, self-review
		// cycle 4, extending AR#1, cycle 3).
		if d.RecordedHash == "" {
			hasUntracked = true
			paths = append(paths, d.Path+" (untracked)")
			continue
		}
		paths = append(paths, d.Path)
	}
	sort.Strings(paths)
	detail := fmt.Sprintf(
		"%d core path(s) diverge from both the recorded and current template hash with no fork record (unresolved drift): %s — resolve with `ralph eject` (keep the change) or `ralph adopt` (discard it)",
		len(paths), strings.Join(paths, ", "))
	if hasUntracked {
		detail += "; paths marked (untracked) have no manifest entry — eject/adopt reject them, move the local file aside (or merge manually) and re-run `ralph upgrade` instead"
	}
	r.Detail = detail
	return r
}

// checkManagedBlocks is FR-9(b): each managed-block surface (AGENTS.md,
// .gitignore) must have well-formed BEGIN/END markers whose interior
// already equals the expected managed content. applyBlockUpdatesV2 (write
// mode is upgrade's; false here) is the exact same computation `ralph
// upgrade` and its --dry-run preview use, so "healthy" here means
// byte-identical to what the block engine would compute — a surface whose
// current content is up to date is upgrade.BlockUnchanged (blockUpdateOutcome.ok==true);
// anything else (a pending BlockUpdated/BlockAppended change, a
// BlockMalformed marker pair, a missing/symlinked/non-regular surface file)
// is a violation.
//
// Unlike checkScaffoldCoreHashes's FR-9(a), which deliberately tolerates a
// pending-but-not-yet-applied template update, this check has no such
// tolerance: a pending BlockUpdated/BlockAppended change against the
// current binary's embedded template IS flagged here. This asymmetry
// between (a) and (b) is intentional (see the --strict flag help and
// checkScaffoldCoreHashes's doc comment) — run `ralph upgrade` before
// `doctor --strict` right after upgrading the ralph binary itself, so a
// version-skew pending update does not read as scaffold damage.
//
// An applyBlockUpdatesV2 error itself (e.g. os.Lstat failing on a block
// surface with something other than ErrNotExist — a permission error, a
// non-directory path component) is a strict-eligible violation in its own
// right, not a warn-and-move-on: FR-9(b) cannot be verified at all when the
// block-outcome computation itself fails, which is the same
// verification-impossible class as the corrupt-manifest and
// ownership-planning-error meta-failures above (AR#2, cycle 2,
// docs/reports/cross-review-triage-overlay-scaffold-v2-p5.md).
func checkManagedBlocks(absDir string, desired map[string][]byte, strict bool) checkResult {
	r := checkResult{Name: "Scaffold: managed blocks"}
	outcomes, notes, err := applyBlockUpdatesV2(absDir, desired, io.Discard, false)
	if err != nil {
		r.Status = scaffoldViolationStatus(true, strict)
		r.Detail = fmt.Sprintf("computing managed block outcomes: %v", err)
		return r
	}

	var offenders []string
	for _, bs := range blockSurfaces {
		oc, ok := outcomes[bs.path]
		if !ok || !oc.ok || oc.outcome != upgrade.BlockUnchanged {
			offenders = append(offenders, bs.path)
		}
	}

	violated := len(offenders) > 0
	r.Status = scaffoldViolationStatus(violated, strict)
	if !violated {
		return r
	}
	detail := fmt.Sprintf("%d block surface(s) unhealthy: %s", len(offenders), strings.Join(offenders, ", "))
	if len(notes) > 0 {
		detail += " — " + strings.Join(notes, "; ")
	}
	r.Detail = detail
	return r
}

// checkSettingsOwnedKeys is FR-9(c): .claude/settings.json's ralph-owned
// keys (upgrade.OwnedSettingsPaths: env, permissions.allow,
// permissions.deny, hooks) must be in sync with the current template.
//
// Source of truth: the exact 3-way merge `ralph upgrade` performs —
// upgrade.MergeOwnedSettings(current disk content, oldOwned snapshot,
// newOwned template) — rather than a second, parallel definition of "owned
// key present". oldOwned is the settings snapshot
// (.ralph/core/settings.ralph.json, upgrade.LoadSettingsSnapshot), falling
// back to "{}" when absent (pre-Phase-3 init, mirroring runUpgradeV2's own
// fallback); newOwned is desired[v2SettingsPath], the current embedded
// template. mergeResult.Changed==true means the merge would rewrite
// settings.json — an owned key is missing, stale (ralph-owned in oldOwned
// but dropped from newOwned, so it should have been pruned), or otherwise
// out of sync — which is exactly FR-9(c)'s "所有キーの健在" violated. A
// key present with extra user-added entries is never flagged: those are
// preserved untouched by the merge (Changed stays false for them).
//
// Like checkManagedBlocks (FR-9(b)) and unlike checkScaffoldCoreHashes
// (FR-9(a)), this check compares against the current binary's embedded
// template with no pending-update tolerance — the same intentional
// asymmetry documented on checkManagedBlocks and in the --strict flag
// help.
//
// Every error path below (snapshot read, missing desired-state template,
// disk read, or the merge itself) is a strict-eligible violation, not a
// warn-and-move-on: each one means FR-9(c) cannot be verified at all, the
// same verification-impossible class as checkManagedBlocks' computation
// error and the corrupt-manifest/ownership-planning-error meta-failures in
// checkScaffoldIntegrity above (AR#1 + class-closing audit, cycle 2,
// docs/reports/cross-review-triage-overlay-scaffold-v2-p5.md). Before this
// fix, an invalid-JSON settings.json made MergeOwnedSettings fail, and
// --strict reported a hardcoded "warn" that never flipped doctor's exit
// code — the gate failed open on the exact input FR-9(c) exists to catch.
func checkSettingsOwnedKeys(absDir string, desired map[string][]byte, strict bool) checkResult {
	r := checkResult{Name: "Scaffold: settings.json owned keys"}

	oldOwnedSnapshot, found, err := upgrade.LoadSettingsSnapshot(absDir)
	if err != nil {
		r.Status = scaffoldViolationStatus(true, strict)
		r.Detail = fmt.Sprintf("reading settings snapshot: %v", err)
		return r
	}
	oldOwned := oldOwnedSnapshot
	if !found {
		oldOwned = []byte("{}")
	}

	newOwned, ok := desired[v2SettingsPath]
	if !ok {
		r.Status = scaffoldViolationStatus(true, strict)
		r.Detail = fmt.Sprintf("current templates do not ship %s; cannot verify owned keys", v2SettingsPath)
		return r
	}

	current, err := readFinalDiskContent(absDir, v2SettingsPath)
	if err != nil {
		r.Status = scaffoldViolationStatus(true, strict)
		r.Detail = fmt.Sprintf("reading %s: %v", v2SettingsPath, err)
		return r
	}

	mergeResult, err := upgrade.MergeOwnedSettings(current, oldOwned, newOwned)
	if err != nil {
		r.Status = scaffoldViolationStatus(true, strict)
		r.Detail = fmt.Sprintf("merging %s: %v", v2SettingsPath, err)
		return r
	}

	violated := mergeResult.Changed
	r.Status = scaffoldViolationStatus(violated, strict)
	if !violated {
		return r
	}
	r.Detail = fmt.Sprintf(
		"%s: ralph-owned keys (%s) are out of sync with the current template — run `ralph upgrade` to reconcile",
		v2SettingsPath, strings.Join(upgrade.OwnedSettingsPaths[:], ", "))
	return r
}

// conflictMarkerRe matches a git conflict-marker line: exactly seven
// `<`/`=`/`>` characters, optionally followed by a label (e.g. "<<<<<<<
// HEAD"), matching the same shape git itself writes on an unresolved merge.
var conflictMarkerRe = regexp.MustCompile(`(?m)^(<{7}(\s.*)?|={7}|>{7}(\s.*)?)$`)

// checkConflictMarkers is FR-9(d): no manifest-tracked file may contain an
// unresolved git conflict marker. Every manifest.Files path is scanned
// (including v2 exception faces — a conflict marker is a violation
// regardless of which mechanism owns the surrounding content); a path
// missing from disk is silently skipped here since that is FR-9(e)'s
// (checkManifestConsistency) finding to report, not this check's.
func checkConflictMarkers(absDir string, manifest *scaffold.Manifest, strict bool) checkResult {
	r := checkResult{Name: "Scaffold: conflict markers"}

	paths := make([]string, 0, len(manifest.Files))
	for p := range manifest.Files {
		paths = append(paths, p)
	}
	sort.Strings(paths)

	var offenders []string
	var unreadable []string
	for _, p := range paths {
		full := filepath.Join(absDir, filepath.FromSlash(p))
		data, err := os.ReadFile(full)
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				// FR-9(e)'s (checkManifestConsistency) finding, not ours.
				continue
			}
			// Exists but cannot be read (e.g. permission denied): the
			// check is impossible to run for this path. Defense in depth:
			// in practice checkScaffoldIntegrity's ownership-planning
			// pass reads most tracked paths first and already fails
			// strict on such an error, but any path the planner did not
			// itself read would otherwise be silently skipped here while
			// FR-9(e)'s os.Stat existence sweep misses it too (fail-open
			// class; cycle-3 warn-path audit alongside AR#1/AR#2, cycle
			// 2, docs/reports/cross-review-triage-overlay-scaffold-v2-p5.md).
			unreadable = append(unreadable, fmt.Sprintf("%s (%v)", p, err))
			continue
		}
		if conflictMarkerRe.Match(data) {
			offenders = append(offenders, p)
		}
	}
	// Both fragments are reported together (no early return): an operator
	// with one unreadable file and one real conflict marker should learn
	// about both in a single run (self-review C3-L5, cycle 3).
	if len(unreadable) > 0 {
		r.Status = scaffoldViolationStatus(true, strict)
		detail := fmt.Sprintf("cannot scan tracked file(s) for conflict markers: %s", strings.Join(unreadable, "; "))
		if len(offenders) > 0 {
			detail += fmt.Sprintf("; conflict markers found in: %s", strings.Join(offenders, ", "))
		}
		r.Detail = detail
		return r
	}

	violated := len(offenders) > 0
	r.Status = scaffoldViolationStatus(violated, strict)
	if !violated {
		return r
	}
	r.Detail = fmt.Sprintf("%d tracked file(s) contain unresolved conflict markers: %s", len(offenders), strings.Join(offenders, ", "))
	return r
}

// checkManifestConsistency is FR-9(e): every manifest-tracked path must
// still exist on disk. Three of v2SkipPaths()'s four exception faces are
// excluded here — settings.json (v2SettingsPath) and the two managed-block
// surfaces (AGENTS.md, .gitignore) — because their presence/health is
// FR-9(b)/(c)'s job (checkManagedBlocks, checkSettingsOwnedKeys), and both
// of those checks are verified to already fail on a missing surface:
// checkManagedBlocks flags it because updateOneBlockV2 returns a zero-value,
// ok=false outcome for a missing block surface, which checkManagedBlocks
// treats as an offender; checkSettingsOwnedKeys flags it because
// readFinalDiskContent returns nil for a missing settings.json,
// MergeOwnedSettings treats that nil/absent current as "{}", and merging
// "{}" against a non-empty owned template (env/permissions/hooks) reports
// Changed=true. Re-checking bare existence for either of these two faces
// here would only duplicate those findings under a different check name.
//
// The settings snapshot (.ralph/core/settings.ralph.json,
// upgrade.SettingsSnapshotRelPath) is a v2SkipPaths() member too, but it is
// deliberately NOT excluded here: checkSettingsOwnedKeys's oldOwned falls
// back to "{}" when upgrade.LoadSettingsSnapshot reports found=false (the
// same non-destructive fallback runUpgradeV2 itself uses for a
// pre-Phase-3-init project that never had a snapshot) — that fallback
// treats a missing snapshot as legitimate "no prior owned state", not a
// violation, so a snapshot deleted out from under a manifest that still
// tracks it passes checkSettingsOwnedKeys silently. Nothing else checks the
// snapshot's existence, so it stays in this sweep (AR#2, cycle 1,
// docs/reports/cross-review-triage-overlay-scaffold-v2-p5.md). Do not widen
// this back to the full v2SkipPaths() set without re-verifying that
// whichever face is added is still guaranteed by its own dedicated check.
//
// Existence-only (not a hash comparison) is deliberate: a fork's disk
// content is expected to diverge from its recorded DiskHash over time as
// the user keeps editing it post-eject, and that is not an integrity
// violation — only a manifest entry with nothing behind it on disk at all
// is.
func checkManifestConsistency(absDir string, manifest *scaffold.Manifest, strict bool) checkResult {
	r := checkResult{Name: "Scaffold: manifest/disk consistency"}
	// Derived from the shared v2 exception-face set minus the settings
	// snapshot: settings.json's existence is guaranteed by FR-9(c) and the
	// block faces' by FR-9(b), but nothing else guarantees the snapshot
	// exists, so it stays in this check's existence sweep (AR#2, cycle 1,
	// docs/reports/cross-review-triage-overlay-scaffold-v2-p5.md;
	// derivation form per self-review C2-L3, cycle 2).
	skip := v2SkipPaths()
	delete(skip, upgrade.SettingsSnapshotRelPath)

	paths := make([]string, 0, len(manifest.Files))
	for p := range manifest.Files {
		paths = append(paths, p)
	}
	sort.Strings(paths)

	var missing []string
	for _, p := range paths {
		if skip[p] {
			continue
		}
		full := filepath.Join(absDir, filepath.FromSlash(p))
		if _, statErr := os.Stat(full); errors.Is(statErr, fs.ErrNotExist) {
			missing = append(missing, p)
		}
	}

	violated := len(missing) > 0
	r.Status = scaffoldViolationStatus(violated, strict)
	if !violated {
		return r
	}
	r.Detail = fmt.Sprintf("%d manifest-tracked path(s) missing from disk: %s", len(missing), strings.Join(missing, ", "))
	return r
}
