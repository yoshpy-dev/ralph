package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/yoshpy-dev/harness-engineering-scaffolding-template/internal/config"
	"github.com/yoshpy-dev/harness-engineering-scaffolding-template/internal/scaffold"
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

	if cfgErr != nil && !os.IsNotExist(cfgErr) {
		results = append(results, checkResult{
			Name:   "ralph.toml",
			Status: "warn",
			Detail: fmt.Sprintf("parse error: %v — using defaults", cfgErr),
		})
	}

	// Check 1: Claude Code CLI.
	results = append(results, checkClaudeCLI(cfg))

	// Check 2: Hooks integrity.
	results = append(results, checkHooks(targetDir))

	// Check 3: Manifest version.
	results = append(results, checkManifestVersion(targetDir))

	// Check 4: Language pack verify.sh.
	results = append(results, checkLanguagePacks()...)

	// Check 5: Go availability.
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

func checkClaudeCLI(cfg config.Config) checkResult {
	r := checkResult{Name: "Claude Code CLI"}
	_, err := exec.LookPath("claude")
	if err != nil {
		if cfg.Doctor.RequireClaudeCLI {
			r.Status = "fail"
			r.Detail = "claude not found in PATH"
		} else {
			r.Status = "warn"
			r.Detail = "claude not found (not required)"
		}
	} else {
		r.Status = "pass"
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
				if _, err := os.Stat(filepath.Join(targetDir, cmd)); os.IsNotExist(err) {
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

func checkLanguagePacks() []checkResult {
	packs, err := scaffold.AvailablePacks()
	if err != nil {
		return []checkResult{{Name: "Language packs", Status: "warn", Detail: "could not list packs"}}
	}

	var results []checkResult
	for _, p := range packs {
		r := checkResult{Name: fmt.Sprintf("Pack: %s", p)}
		// Check if verify.sh exists in the pack.
		packFS, err := scaffold.PackFS(p)
		if err != nil {
			r.Status = "warn"
			r.Detail = "pack not found"
			results = append(results, r)
			continue
		}
		if _, err := packFS.Open("verify.sh"); err != nil {
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
