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
	"github.com/yoshpy-dev/ralph/internal/scaffold"
)

func newDoctorCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "doctor",
		Short: "Check environment and project setup",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runDoctor(".")
		},
	}
}

type checkResult struct {
	Name   string `json:"name"`
	Status string `json:"status"` // pass, warn, fail
	Detail string `json:"detail,omitempty"`
}

func runDoctor(targetDir string) error {
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

	// Check 2: Codex CLI.
	results = append(results, checkCodexCLI(cfg))

	// Check 3: Codex effective config (project trust + codex_hooks + at least one hook).
	results = append(results, checkCodexEffectiveConfig(targetDir))

	// Check 4: Hooks integrity.
	results = append(results, checkHooks(targetDir))

	// Check 5: Manifest version.
	results = append(results, checkManifestVersion(targetDir))

	// Check 6: Language pack verify.sh (checks project's installed packs via manifest).
	results = append(results, checkInstalledPacks(targetDir)...)

	// Check 7: Go availability.
	results = append(results, checkGo(cfg))

	// Print results.
	fmt.Println("ralph doctor")
	fmt.Println()

	allPass := true
	for _, r := range results {
		icon := "✓"
		switch r.Status {
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
	r := checkResult{Name: "Codex CLI"}
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
// `[features] codex_hooks = true` plus at least one [hooks.<event>] entry.
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

	codexHooks := false
	if features, ok := raw["features"].(map[string]any); ok {
		if v, ok := features["codex_hooks"].(bool); ok {
			codexHooks = v
		}
	}

	hookEntries := 0
	if hooks, ok := raw["hooks"].(map[string]any); ok {
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

	switch {
	case !codexHooks:
		r.Status = "warn"
		r.Detail = "[features] codex_hooks = true is not set; project hooks will be ignored"
	case hookEntries == 0:
		r.Status = "warn"
		r.Detail = "no [hooks.*] entries — codex_hooks enabled but nothing wired up. Run `codex trust .` once configured"
	default:
		r.Status = "pass"
		r.Detail = fmt.Sprintf("codex_hooks=true, %d hook entry(ies). Confirm `codex trust .` ran for this project", hookEntries)
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

	// Detect installed packs by checking which pack files appear in the manifest.
	availPacks, err := scaffold.AvailablePacks()
	if err != nil {
		return []checkResult{{Name: "Language packs", Status: "warn", Detail: "could not list packs"}}
	}

	installedPacks := make(map[string]bool)
	for _, p := range availPacks {
		packFS, pErr := scaffold.PackFS(p)
		if pErr != nil {
			continue
		}
		// If any file from this pack is in the manifest, the pack is installed.
		_ = fs.WalkDir(packFS, ".", func(path string, d fs.DirEntry, err error) error {
			if err != nil || d.IsDir() {
				return err
			}
			if _, ok := m.Files[path]; ok {
				installedPacks[p] = true
				return fs.SkipAll
			}
			return nil
		})
	}

	if len(installedPacks) == 0 {
		return []checkResult{{Name: "Language packs", Status: "pass", Detail: "none installed"}}
	}

	var results []checkResult
	for p := range installedPacks {
		r := checkResult{Name: fmt.Sprintf("Pack: %s", p)}
		// Check that verify.sh is executable on disk.
		verifyPath := filepath.Join(targetDir, "verify.sh")
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
			r.Detail = "verify.sh not found on disk"
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
