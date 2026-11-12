package cli

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/yoshpy-dev/ralph/internal/config"
)

// Tests for checkCodexHookTrust, `ralph doctor`'s "Codex hook trust" check
// (plan 2026-11-12-codex-hook-trust-check, AC1-AC5, AC9; issue #310). Each
// passes its own codex config path, so none reads the real ~/.codex. None
// runs in parallel (newLegacyLedgerRepo chdirs and sets environment
// variables).

// codexHookTrustDoctorHooks is a hooks.json with two handlers codex asks
// approval for: pre_tool_use:0:0 and session_start:0:0.
const codexHookTrustDoctorHooks = `{"hooks": {
  "PreToolUse": [{"matcher": "Bash", "hooks": [{"type": "command", "command": "guard.sh"}]}],
  "SessionStart": [{"hooks": [{"type": "command", "command": "start.sh"}]}]
}}`

// codexHookTrustWithCodex is an [org] whose driver_pool has codex, so a codex
// seat can be spawned and the check reads the files.
var codexHookTrustWithCodex = config.OrgConfig{DriverPool: []string{"claude", "codex"}}

// codexHookTrustDoctorRepo is a main checkout with codexHookTrustDoctorHooks
// and a codex home of its own, both under a symlink-resolved temp dir.
type codexHookTrustDoctorRepo struct {
	tmp        string
	repo       string // the checkout, a .git directory holding HEAD
	hooksPath  string // <repo>/.codex/hooks.json
	home       string // the home codexConfigDisplayPath shortens to "~"
	configPath string // <home>/.codex/config.toml, not written yet
}

func newCodexHookTrustDoctorRepo(t *testing.T) codexHookTrustDoctorRepo {
	t.Helper()
	tmp := codexHookTrustTempDir(t)
	f := codexHookTrustDoctorRepo{tmp: tmp, repo: filepath.Join(tmp, "repo"), home: filepath.Join(tmp, "home")}
	f.hooksPath = filepath.Join(f.repo, ".codex", "hooks.json")
	f.configPath = filepath.Join(f.home, ".codex", "config.toml")
	makeCodexHookTrustCheckout(t, f.repo)
	writeCodexHookTrustFile(t, f.hooksPath, codexHookTrustDoctorHooks)
	return f
}

// resolveEnv is the check's resolveEnv for f's codex home.
func (f codexHookTrustDoctorRepo) resolveEnv() (codexSandboxEnv, error) {
	return codexSandboxEnv{Home: f.home, ConfigPath: f.configPath}, nil
}

// codexHookTrustApprovals renders a [hooks.state] table with a trusted_hash
// for each key.
func codexHookTrustApprovals(keys ...string) string {
	var b strings.Builder
	for _, key := range keys {
		b.WriteString(codexHookTrustStateTable(key, `trusted_hash = "sha256:a"`))
	}
	return b.String()
}

// assertCodexHookTrustResult checks the check's name and status and that the
// detail holds each of want and none of notWant.
func assertCodexHookTrustResult(t *testing.T, r checkResult, status string, want, notWant []string) {
	t.Helper()
	if r.Name != "Codex hook trust" {
		t.Errorf("Name = %q, want %q", r.Name, "Codex hook trust")
	}
	if r.Status != status {
		t.Errorf("Status = %q, want %q (Detail: %s)", r.Status, status, r.Detail)
	}
	for _, s := range want {
		if !strings.Contains(r.Detail, s) {
			t.Errorf("Detail lacks %q:\n%s", s, r.Detail)
		}
	}
	for _, s := range notWant {
		if strings.Contains(r.Detail, s) {
			t.Errorf("Detail holds %q:\n%s", s, r.Detail)
		}
	}
	if strings.Contains(r.Detail, "\n") {
		t.Errorf("Detail spans more than one line:\n%s", r.Detail)
	}
}

func TestCheckCodexHookTrust_NotNeededWithoutCodexReadsNothing(t *testing.T) {
	f := newCodexHookTrustDoctorRepo(t)
	// Every input is broken: had the check read anything, it would not pass.
	mkdirCodexHookTrust(t, f.configPath)
	writeCodexHookTrustFile(t, f.hooksPath, "{")
	resolved := false
	resolveEnv := func() (codexSandboxEnv, error) {
		resolved = true
		return f.resolveEnv()
	}

	r := checkCodexHookTrust(config.OrgConfig{DriverPool: []string{"claude"}}, f.repo, resolveEnv)

	assertCodexHookTrustResult(t, r, "pass", []string{"not needed: codex is not in [org].driver_pool, so no codex seat can be spawned"}, nil)
	if resolved {
		t.Error("the check resolved the codex config path with no codex in driver_pool")
	}
}

func TestCheckCodexHookTrust_InfoWhenTheConfigPathCannotBeResolved(t *testing.T) {
	f := newCodexHookTrustDoctorRepo(t)
	resolveEnv := func() (codexSandboxEnv, error) { return codexSandboxEnv{}, errors.New("no home here") }

	r := checkCodexHookTrust(codexHookTrustWithCodex, f.repo, resolveEnv)

	assertCodexHookTrustResult(t, r, "info", []string{"could not resolve the codex config path: no home here", "hook approvals not checked"}, nil)
}

// The hooks.json codex reads is the one Check 3 checks, so this check says
// only that it did not check, naming that file (AC4).
func TestCheckCodexHookTrust_InfoWhenItsOwnHooksJSONCannotBeRead(t *testing.T) {
	cases := []struct {
		name    string
		hooks   string // "" removes hooks.json; "<dir>" makes it a directory
		problem string
	}{
		{"missing", "", "does not exist"},
		{"a directory", "<dir>", "could not be read (not a regular file)"},
		{"not JSON", "{", "is not valid (not valid JSON (at byte 1))"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := newCodexHookTrustDoctorRepo(t)
			if err := os.Remove(f.hooksPath); err != nil {
				t.Fatal(err)
			}
			switch tc.hooks {
			case "":
			case "<dir>":
				mkdirCodexHookTrust(t, f.hooksPath)
			default:
				writeCodexHookTrustFile(t, f.hooksPath, tc.hooks)
			}
			want := []string{"hook approvals not checked: " + f.hooksPath + " " + tc.problem, "(the same file Codex effective config checks)"}

			r := checkCodexHookTrust(codexHookTrustWithCodex, f.repo, f.resolveEnv)
			assertCodexHookTrustResult(t, r, "info", want, nil)

			// doctor passes "." from the repository's root: the same file.
			t.Chdir(f.repo)
			r = checkCodexHookTrust(codexHookTrustWithCodex, ".", f.resolveEnv)
			assertCodexHookTrustResult(t, r, "info", want, nil)
		})
	}
}

// From a linked worktree codex reads the main checkout's hooks.json, which
// Check 3 does not read, so this check warns when it is missing or broken
// (AC4, plan advisory finding 2).
func TestCheckCodexHookTrust_WarnsFromALinkedWorktreeWhenTheMainCheckoutsHooksCannotBeRead(t *testing.T) {
	cases := []struct {
		name    string
		hooks   string // "" leaves the main checkout without hooks.json
		problem string
	}{
		{"missing", "", "which does not exist"},
		{"not of codex's shape", `{"hooks": []}`, `which is not valid (unexpected JSON array at "hooks")`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := newLegacyLedgerRepo(t)
			writeCodexHookTrustFile(t, filepath.Join(r.worktree, ".codex", "hooks.json"), codexHookTrustDoctorHooks)
			mainHooks := filepath.Join(r.mainRoot, ".codex", "hooks.json")
			if tc.hooks != "" {
				writeCodexHookTrustFile(t, mainHooks, tc.hooks)
			}
			resolveEnv := func() (codexSandboxEnv, error) {
				return codexSandboxEnv{ConfigPath: filepath.Join(t.TempDir(), "config.toml")}, nil
			}

			for _, dir := range []string{r.worktree, "."} {
				result := checkCodexHookTrust(codexHookTrustWithCodex, dir, resolveEnv)
				assertCodexHookTrustResult(t, result, "warn",
					[]string{"codex seats read hooks from " + mainHooks + ", " + tc.problem, "so no hook runs in a codex seat"},
					[]string{"Codex effective config"})
			}
		})
	}
}

// A main checkout's hooks.json that exists but cannot be read (a directory,
// or a file of mode 000) is a warning from a linked worktree too, and info
// from the main checkout itself, where it is the file Check 3 reads (AC4).
func TestCheckCodexHookTrust_WarnsFromALinkedWorktreeWhenTheMainCheckoutsHooksAreUnreadable(t *testing.T) {
	cases := []struct {
		name       string
		breakHooks func(t *testing.T, mainHooks string) // makes mainHooks exist but unreadable
	}{
		{"a directory", func(t *testing.T, p string) { mkdirCodexHookTrust(t, p) }},
		{"mode 000", func(t *testing.T, p string) {
			if os.Geteuid() == 0 {
				t.Skip("root can read a mode-000 file")
			}
			writeCodexHookTrustFile(t, p, codexHookTrustDoctorHooks)
			if err := os.Chmod(p, 0); err != nil {
				t.Fatalf("chmod %s: %v", p, err)
			}
			t.Cleanup(func() { _ = os.Chmod(p, 0o644) })
			if f, err := os.Open(p); err == nil {
				_ = f.Close()
				t.Skip("mode 000 does not deny reads here")
			}
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := newLegacyLedgerRepo(t)
			writeCodexHookTrustFile(t, filepath.Join(r.worktree, ".codex", "hooks.json"), codexHookTrustDoctorHooks)
			mainHooks := filepath.Join(r.mainRoot, ".codex", "hooks.json")
			tc.breakHooks(t, mainHooks)
			resolveEnv := func() (codexSandboxEnv, error) {
				return codexSandboxEnv{ConfigPath: filepath.Join(t.TempDir(), "config.toml")}, nil
			}

			for _, dir := range []string{r.worktree, "."} {
				result := checkCodexHookTrust(codexHookTrustWithCodex, dir, resolveEnv)
				assertCodexHookTrustResult(t, result, "warn",
					[]string{"codex seats read hooks from " + mainHooks + ", which could not be read (", "so no hook runs in a codex seat"}, nil)
			}
			result := checkCodexHookTrust(codexHookTrustWithCodex, r.mainRoot, resolveEnv)
			assertCodexHookTrustResult(t, result, "info", []string{"could not be read (", "the same file Codex effective config checks"}, nil)
		})
	}
}

func TestCheckCodexHookTrust_PassWhenNoHookNeedsApproval(t *testing.T) {
	f := newCodexHookTrustDoctorRepo(t)
	writeCodexHookTrustFile(t, f.hooksPath, `{"hooks": {"Stop": [{"hooks": [{"type": "command", "command": " "}]}]}}`)
	// No config at all: with nothing to approve it does not matter.

	r := checkCodexHookTrust(codexHookTrustWithCodex, f.repo, f.resolveEnv)

	assertCodexHookTrustResult(t, r, "pass", []string{"no hook in " + f.hooksPath + " needs approval"}, nil)
}

func TestCheckCodexHookTrust_WarnsWhenTheCodexConfigIsMissing(t *testing.T) {
	f := newCodexHookTrustDoctorRepo(t)

	r := checkCodexHookTrust(codexHookTrustWithCodex, f.repo, f.resolveEnv)

	assertCodexHookTrustResult(t, r, "warn", []string{
		"no codex user config at ~/.codex/config.toml, so none of the 2 hooks in " + f.hooksPath + " is approved",
		"codex skips them, so they do not run in codex seats",
		"open codex in " + f.repo + " and trust the hooks (Hooks need review → Trust all and continue)",
	}, nil)
}

func TestCheckCodexHookTrust_WarnsWhenTheCodexConfigIsUnreadable(t *testing.T) {
	f := newCodexHookTrustDoctorRepo(t)
	mkdirCodexHookTrust(t, f.configPath)

	r := checkCodexHookTrust(codexHookTrustWithCodex, f.repo, f.resolveEnv)

	assertCodexHookTrustResult(t, r, "warn", []string{
		"could not read ~/.codex/config.toml (not a regular file)",
		"the approvals of the 2 hooks in " + f.hooksPath + " were not checked",
	}, nil)
}

func TestCheckCodexHookTrust_WarnsWithOnlyThePositionOfAnUndecodableConfig(t *testing.T) {
	const marker = "config-content-marker-4be1"
	f := newCodexHookTrustDoctorRepo(t)
	writeCodexHookTrustFile(t, f.configPath, codexHookTrustApprovals(f.hooksPath+":pre_tool_use:0:0")+marker+" = = 1\n")

	r := checkCodexHookTrust(codexHookTrustWithCodex, f.repo, f.resolveEnv)

	assertCodexHookTrustResult(t, r, "warn", []string{
		"~/.codex/config.toml could not be decoded as a codex config (line 3, column ",
		"the approvals of the 2 hooks in " + f.hooksPath + " were not checked",
	}, []string{marker, "trusted_hash", "sha256:a"})
}

func TestCheckCodexHookTrust_WarnsNamingTheHookNotApproved(t *testing.T) {
	f := newCodexHookTrustDoctorRepo(t)
	writeCodexHookTrustFile(t, f.configPath, codexHookTrustApprovals(f.hooksPath+":session_start:0:0"))

	r := checkCodexHookTrust(codexHookTrustWithCodex, f.repo, f.resolveEnv)

	assertCodexHookTrustResult(t, r, "warn", []string{
		"1 of 2 hooks in " + f.hooksPath + " is not approved in ~/.codex/config.toml (pre_tool_use:0:0) — codex skips it, so it does not run in codex seats",
		"open codex in " + f.repo + " and trust the hooks (Hooks need review → Trust all and continue)",
	}, []string{"session_start", "turned off", "enabled = false"})
}

func TestCheckCodexHookTrust_WarnsWhenAHookIsTurnedOff(t *testing.T) {
	cases := []struct {
		name          string
		config        func(f codexHookTrustDoctorRepo) string
		want, notWant []string
	}{
		{
			"turned off and approved",
			func(f codexHookTrustDoctorRepo) string {
				return codexHookTrustApprovals(f.hooksPath+":pre_tool_use:0:0") +
					codexHookTrustStateTable(f.hooksPath+":session_start:0:0", "trusted_hash = \"sha256:b\"\nenabled = false")
			},
			[]string{
				"1 of 2 hooks in <hooks> is turned off (enabled = false) in ~/.codex/config.toml (session_start:0:0) — codex skips it",
				"remove enabled = false from the [hooks.state] entries of the hooks turned off in ~/.codex/config.toml",
			},
			[]string{"not approved", "Trust all and continue"},
		},
		{
			"one not approved and one turned off",
			func(f codexHookTrustDoctorRepo) string {
				return codexHookTrustStateTable(f.hooksPath+":session_start:0:0", "enabled = false")
			},
			[]string{
				"1 of 2 hooks in <hooks> is not approved in ~/.codex/config.toml, 1 turned off (enabled = false) (pre_tool_use:0:0, session_start:0:0) — codex skips them",
				"trust the hooks (Hooks need review → Trust all and continue), and remove enabled = false from the [hooks.state] entries",
			},
			nil,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := newCodexHookTrustDoctorRepo(t)
			writeCodexHookTrustFile(t, f.configPath, tc.config(f))
			var want []string
			for _, w := range tc.want {
				want = append(want, strings.ReplaceAll(w, "<hooks>", f.hooksPath))
			}

			r := checkCodexHookTrust(codexHookTrustWithCodex, f.repo, f.resolveEnv)

			assertCodexHookTrustResult(t, r, "warn", want, tc.notWant)
		})
	}
}

func TestCheckCodexHookTrust_NamesAtMostThreeHooks(t *testing.T) {
	f := newCodexHookTrustDoctorRepo(t)
	writeCodexHookTrustFile(t, f.hooksPath, `{"hooks": {
  "PreToolUse": [{"hooks": [{"type": "command", "command": "a"}, {"type": "command", "command": "b"}]}],
  "PostToolUse": [{"hooks": [{"type": "command", "command": "c"}]}],
  "Stop": [{"hooks": [{"type": "command", "command": "d"}]}]
}}`)
	writeCodexHookTrustFile(t, f.configPath, "model = \"gpt-x\"\n")

	r := checkCodexHookTrust(codexHookTrustWithCodex, f.repo, f.resolveEnv)

	assertCodexHookTrustResult(t, r, "warn", []string{
		"4 of 4 hooks in " + f.hooksPath + " are not approved in ~/.codex/config.toml (post_tool_use:0:0, pre_tool_use:0:0, pre_tool_use:0:1, …) — codex skips them",
	}, []string{"stop:0:0"})
}

func TestCheckCodexHookTrust_PassWhenEveryHookIsApproved(t *testing.T) {
	f := newCodexHookTrustDoctorRepo(t)
	writeCodexHookTrustFile(t, f.configPath, codexHookTrustApprovals(f.hooksPath+":pre_tool_use:0:0", f.hooksPath+":session_start:0:0"))

	r := checkCodexHookTrust(codexHookTrustWithCodex, f.repo, f.resolveEnv)

	assertCodexHookTrustResult(t, r, "pass", []string{
		"2 hooks in " + f.hooksPath + " approved in ~/.codex/config.toml",
		"keys only; the hash is not checked",
		"makes codex show Hooks need review in the seat's pane",
	}, nil)
}

// From a linked worktree the check reads the main checkout's hooks.json and
// looks for approvals under the main checkout's path (AC5).
func TestCheckCodexHookTrust_LinkedWorktreeUsesTheMainCheckoutsKeys(t *testing.T) {
	r := newLegacyLedgerRepo(t)
	mainHooks := filepath.Join(r.mainRoot, ".codex", "hooks.json")
	wtHooks := filepath.Join(r.worktree, ".codex", "hooks.json")
	writeCodexHookTrustFile(t, mainHooks, codexHookTrustDoctorHooks)
	writeCodexHookTrustFile(t, wtHooks, codexHookTrustDoctorHooks)
	configPath := filepath.Join(t.TempDir(), "config.toml")
	resolveEnv := func() (codexSandboxEnv, error) { return codexSandboxEnv{ConfigPath: configPath}, nil }

	writeCodexHookTrustFile(t, configPath, codexHookTrustApprovals(mainHooks+":pre_tool_use:0:0", mainHooks+":session_start:0:0"))
	result := checkCodexHookTrust(codexHookTrustWithCodex, r.worktree, resolveEnv)
	assertCodexHookTrustResult(t, result, "pass", []string{"2 hooks in " + mainHooks + " approved in " + configPath}, []string{wtHooks})

	// Approvals under the worktree's own path are not the ones codex looks up.
	writeCodexHookTrustFile(t, configPath, codexHookTrustApprovals(wtHooks+":pre_tool_use:0:0", wtHooks+":session_start:0:0"))
	result = checkCodexHookTrust(codexHookTrustWithCodex, r.worktree, resolveEnv)
	assertCodexHookTrustResult(t, result, "warn", []string{
		"2 of 2 hooks in " + mainHooks + " are not approved in " + configPath,
		"open codex in " + r.mainRoot + " and trust the hooks",
	}, []string{wtHooks})
}

// AC11: a hooks.json codex refuses for a null or a missing field loads no
// hook, so it is never a pass: from a linked worktree, whose own hooks.json
// is valid, it is a warning, and from the main checkout, where it is the
// file Check 3 reads, it is info.
func TestCheckCodexHookTrust_NullHookStructuresAreNeverAPass(t *testing.T) {
	for _, tc := range codexHookTrustInvalidHooks {
		t.Run(tc.name, func(t *testing.T) {
			r := newLegacyLedgerRepo(t)
			writeCodexHookTrustFile(t, filepath.Join(r.worktree, ".codex", "hooks.json"), codexHookTrustDoctorHooks)
			mainHooks := filepath.Join(r.mainRoot, ".codex", "hooks.json")
			writeCodexHookTrustFile(t, mainHooks, tc.data)
			resolveEnv := func() (codexSandboxEnv, error) {
				return codexSandboxEnv{ConfigPath: filepath.Join(t.TempDir(), "config.toml")}, nil
			}

			result := checkCodexHookTrust(codexHookTrustWithCodex, r.worktree, resolveEnv)
			assertCodexHookTrustResult(t, result, "warn",
				[]string{"codex seats read hooks from " + mainHooks + ", which is not valid (" + tc.want + "), so no hook runs in a codex seat"}, nil)

			result = checkCodexHookTrust(codexHookTrustWithCodex, r.mainRoot, resolveEnv)
			assertCodexHookTrustResult(t, result, "info",
				[]string{"hook approvals not checked: " + mainHooks + " is not valid (" + tc.want + ") (the same file Codex effective config checks)"}, nil)
		})
	}
}

// AC10: run in a subdirectory, the check also counts the hooks of every
// .codex/hooks.json from the root down to it, keyed under that file's path,
// so approved root hooks do not hide an unapproved nested one. A nested
// .codex without hooks.json adds nothing, and from the root the nested layer
// is not on the way.
func TestCheckCodexHookTrust_CountsNestedLayersDownToTheDirectory(t *testing.T) {
	f := newCodexHookTrustDoctorRepo(t)
	sub := filepath.Join(f.repo, "sub")
	subHooks := filepath.Join(sub, ".codex", "hooks.json")
	writeCodexHookTrustFile(t, subHooks, codexHookTrustStopHooks)
	rootKeys := []string{f.hooksPath + ":pre_tool_use:0:0", f.hooksPath + ":session_start:0:0"}
	writeCodexHookTrustFile(t, f.configPath, codexHookTrustApprovals(rootKeys...))

	r := checkCodexHookTrust(codexHookTrustWithCodex, sub, f.resolveEnv)
	assertCodexHookTrustResult(t, r, "warn", []string{
		"1 of 3 hooks in " + f.hooksPath + " and " + subHooks + " is not approved in ~/.codex/config.toml (sub/.codex/hooks.json:stop:0:0) — codex skips it, so it does not run in codex seats",
		"open codex in " + sub + " and trust the hooks (Hooks need review → Trust all and continue)",
	}, nil)

	r = checkCodexHookTrust(codexHookTrustWithCodex, f.repo, f.resolveEnv)
	assertCodexHookTrustResult(t, r, "pass", []string{"2 hooks in " + f.hooksPath + " approved in ~/.codex/config.toml"}, []string{subHooks})

	writeCodexHookTrustFile(t, f.configPath, codexHookTrustApprovals(rootKeys[0], rootKeys[1], subHooks+":stop:0:0"))
	r = checkCodexHookTrust(codexHookTrustWithCodex, sub, f.resolveEnv)
	assertCodexHookTrustResult(t, r, "pass", []string{"3 hooks in " + f.hooksPath + " and " + subHooks + " approved in ~/.codex/config.toml"}, nil)

	writeCodexHookTrustFile(t, f.configPath, codexHookTrustApprovals(rootKeys...))
	if err := os.Remove(subHooks); err != nil {
		t.Fatal(err)
	}
	r = checkCodexHookTrust(codexHookTrustWithCodex, sub, f.resolveEnv)
	assertCodexHookTrustResult(t, r, "pass", []string{"2 hooks in " + f.hooksPath + " approved in ~/.codex/config.toml"}, []string{subHooks})
}

// A nested hooks.json codex cannot load is reported like the root's: a
// warning naming it when it is not the file Check 3 reads, in which case the
// root's hooks still run, and info when it is (AC10, AC11).
func TestCheckCodexHookTrust_ReportsANestedHooksJSONItCannotRead(t *testing.T) {
	f := newCodexHookTrustDoctorRepo(t)
	sub := filepath.Join(f.repo, "sub")
	subHooks := filepath.Join(sub, ".codex", "hooks.json")
	writeCodexHookTrustFile(t, subHooks, `{"hooks": {"Stop": [null]}}`)
	deeper := filepath.Join(sub, "deeper")
	mkdirCodexHookTrust(t, deeper)
	writeCodexHookTrustFile(t, f.configPath, codexHookTrustApprovals(f.hooksPath+":pre_tool_use:0:0", f.hooksPath+":session_start:0:0"))
	problem := `is not valid (unexpected JSON null at "hooks.Stop.0")`

	r := checkCodexHookTrust(codexHookTrustWithCodex, deeper, f.resolveEnv)
	assertCodexHookTrustResult(t, r, "warn", []string{"codex seats read hooks from " + subHooks + ", which " + problem + ", so no hook from it runs in a codex seat"}, nil)

	r = checkCodexHookTrust(codexHookTrustWithCodex, sub, f.resolveEnv)
	assertCodexHookTrustResult(t, r, "info", []string{"hook approvals not checked: " + subHooks + " " + problem + " (the same file Codex effective config checks)"}, nil)

	// With the root's hooks.json broken too, neither file's hooks run.
	writeCodexHookTrustFile(t, f.hooksPath, "{")
	r = checkCodexHookTrust(codexHookTrustWithCodex, deeper, f.resolveEnv)
	assertCodexHookTrustResult(t, r, "warn", []string{
		"codex seats read hooks from " + f.hooksPath + ", which is not valid (not valid JSON (at byte 1)), and from " + subHooks + ", which " + problem + ", so no hook runs in a codex seat",
	}, nil)
}

// From a linked worktree a nested layer whose .codex is in the worktree is
// read from, and keyed under, the same place in the main checkout (AC10).
func TestCheckCodexHookTrust_LinkedWorktreeChecksNestedLayersUnderTheMainCheckout(t *testing.T) {
	r := newLegacyLedgerRepo(t)
	mainHooks := filepath.Join(r.mainRoot, ".codex", "hooks.json")
	mainSubHooks := filepath.Join(r.mainRoot, "sub", ".codex", "hooks.json")
	writeCodexHookTrustFile(t, mainHooks, codexHookTrustDoctorHooks)
	writeCodexHookTrustFile(t, mainSubHooks, codexHookTrustStopHooks)
	wtHooks := filepath.Join(r.worktree, ".codex", "hooks.json")
	wtSubHooks := filepath.Join(r.worktree, "sub", ".codex", "hooks.json")
	writeCodexHookTrustFile(t, wtHooks, codexHookTrustDoctorHooks)
	writeCodexHookTrustFile(t, wtSubHooks, codexHookTrustFixtureHooks)
	configPath := filepath.Join(t.TempDir(), "config.toml")
	resolveEnv := func() (codexSandboxEnv, error) { return codexSandboxEnv{ConfigPath: configPath}, nil }
	wtSub := filepath.Join(r.worktree, "sub")
	mainKeys := []string{mainHooks + ":pre_tool_use:0:0", mainHooks + ":session_start:0:0"}

	writeCodexHookTrustFile(t, configPath, codexHookTrustApprovals(mainKeys...))
	result := checkCodexHookTrust(codexHookTrustWithCodex, wtSub, resolveEnv)
	assertCodexHookTrustResult(t, result, "warn", []string{
		"1 of 3 hooks in " + mainHooks + " and " + mainSubHooks + " is not approved in " + configPath + " (sub/.codex/hooks.json:stop:0:0)",
		"open codex in " + filepath.Join(r.mainRoot, "sub") + " and trust the hooks",
	}, []string{wtHooks, wtSubHooks})

	writeCodexHookTrustFile(t, configPath, codexHookTrustApprovals(mainKeys[0], mainKeys[1], mainSubHooks+":stop:0:0"))
	result = checkCodexHookTrust(codexHookTrustWithCodex, wtSub, resolveEnv)
	assertCodexHookTrustResult(t, result, "pass", []string{"3 hooks in " + mainHooks + " and " + mainSubHooks + " approved in " + configPath}, nil)

	// Approvals under the worktree's own nested path are not the ones codex
	// looks up.
	writeCodexHookTrustFile(t, configPath, codexHookTrustApprovals(mainKeys[0], mainKeys[1], wtSubHooks+":stop:0:0"))
	result = checkCodexHookTrust(codexHookTrustWithCodex, wtSub, resolveEnv)
	assertCodexHookTrustResult(t, result, "warn", []string{"1 of 3 hooks in " + mainHooks + " and " + mainSubHooks + " is not approved"}, nil)
}

// A main checkout reached through a symlink has two spellings, and codex keys
// an approval by the one it was opened under, so the check asks for both,
// in either direction (AC9).
func TestCheckCodexHookTrust_SymlinkedMainCheckoutNeedsBothSpellings(t *testing.T) {
	f := newCodexHookTrustDoctorRepo(t)
	link := filepath.Join(f.tmp, "link-repo")
	symlinkCodexHookTrust(t, f.repo, link)
	linkHooks := filepath.Join(link, ".codex", "hooks.json")
	realKeys := []string{f.hooksPath + ":pre_tool_use:0:0", f.hooksPath + ":session_start:0:0"}
	linkKeys := []string{linkHooks + ":pre_tool_use:0:0", linkHooks + ":session_start:0:0"}
	bothRoots := "open codex in " + link + " and again in " + f.repo + " and trust the hooks each time (Hooks need review → Trust all and continue)"

	cases := []struct {
		name   string
		keys   []string
		status string
		want   []string
	}{
		{"approved under the real path only", realKeys, "warn", []string{"2 of 2 hooks in " + linkHooks + " are not approved", bothRoots}},
		{"approved under the symlink path only", linkKeys, "warn", []string{"2 of 2 hooks in " + linkHooks + " are not approved", bothRoots}},
		{"approved under both", append(append([]string{}, realKeys...), linkKeys...), "pass", []string{"2 hooks in " + linkHooks + " approved"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			writeCodexHookTrustFile(t, f.configPath, codexHookTrustApprovals(tc.keys...))
			r := checkCodexHookTrust(codexHookTrustWithCodex, link, f.resolveEnv)
			assertCodexHookTrustResult(t, r, tc.status, tc.want, nil)
		})
	}

	// With no config at all the approval step names both spellings too.
	if err := os.Remove(f.configPath); err != nil {
		t.Fatal(err)
	}
	r := checkCodexHookTrust(codexHookTrustWithCodex, link, f.resolveEnv)
	assertCodexHookTrustResult(t, r, "warn", []string{"no codex user config at ~/.codex/config.toml", bothRoots}, nil)
}

// doctor passes "." and the shell's $PWD may spell the main checkout through
// a symlink. The check still asks for both spellings, in either direction
// (AC9): with only the symlink's spelling, approvals under it alone would pass
// although a seat started under the real path sees the hooks unapproved.
func TestCheckCodexHookTrust_SymlinkedMainCheckoutFromDotNeedsBothSpellings(t *testing.T) {
	f := newCodexHookTrustDoctorRepo(t)
	link := filepath.Join(f.tmp, "link-repo")
	symlinkCodexHookTrust(t, f.repo, link)
	chdirCodexHookTrustAs(t, link)
	linkHooks := filepath.Join(link, ".codex", "hooks.json")
	realKeys := []string{f.hooksPath + ":pre_tool_use:0:0", f.hooksPath + ":session_start:0:0"}
	linkKeys := []string{linkHooks + ":pre_tool_use:0:0", linkHooks + ":session_start:0:0"}
	bothRoots := "open codex in " + link + " and again in " + f.repo + " and trust the hooks each time"

	cases := []struct {
		name   string
		keys   []string
		status string
		want   []string
	}{
		{"approved under the symlink path only", linkKeys, "warn", []string{"2 of 2 hooks in " + linkHooks + " are not approved", bothRoots}},
		{"approved under the real path only", realKeys, "warn", []string{"2 of 2 hooks in " + linkHooks + " are not approved", bothRoots}},
		{"approved under both", append(append([]string{}, realKeys...), linkKeys...), "pass", []string{"2 hooks in " + linkHooks + " approved"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			writeCodexHookTrustFile(t, f.configPath, codexHookTrustApprovals(tc.keys...))
			r := checkCodexHookTrust(codexHookTrustWithCodex, ".", f.resolveEnv)
			assertCodexHookTrustResult(t, r, tc.status, tc.want, nil)
		})
	}
}

func TestCheckCodexEffectiveConfig_PassPointsAtCodexHookTrust(t *testing.T) {
	dir := t.TempDir()
	writeCodexConfigToml(t, dir, "model = \"gpt-5.5\"\n\n[features]\nhooks = true\n")
	writeCodexHooksJSON(t, dir, validHooksJSON)

	r := checkCodexEffectiveConfig(dir)

	if r.Status != "pass" {
		t.Fatalf("Status = %q, want pass (Detail: %s)", r.Status, r.Detail)
	}
	want := "hooks.json wired; confirm `codex trust .` ran for this project (hook approvals: see Codex hook trust)"
	if r.Detail != want {
		t.Errorf("Detail = %q, want %q", r.Detail, want)
	}
}

// TestRunDoctor_CodexHookTrustFollowsCodexEffectiveConfig runs the doctor
// entry point and checks that the "Codex hook trust" line comes right after
// "Codex effective config", through the doctorCodexSandboxEnv seam.
func TestRunDoctor_CodexHookTrustFollowsCodexEffectiveConfig(t *testing.T) {
	f := newCodexHookTrustDoctorRepo(t)
	writeCodexConfigToml(t, f.repo, "model = \"gpt-5.5\"\n\n[features]\nhooks = true\n")
	writeCodexHooksJSON(t, f.repo, validHooksJSON)
	var keys []string
	for _, tail := range []string{"post_tool_use:0:0", "pre_tool_use:0:0", "session_start:0:0", "user_prompt_submit:0:0"} {
		keys = append(keys, f.hooksPath+":"+tail)
	}
	writeCodexHookTrustFile(t, f.configPath, codexHookTrustApprovals(keys...))

	binDir := filepath.Join(f.tmp, "bin")
	mkdirCodexHookTrust(t, binDir)
	for _, bin := range []string{"claude", "codex", "go"} {
		writeStubBin(t, binDir, bin, "")
	}
	t.Setenv("PATH", binDir)
	setAgmsgHomeOverride(t, filepath.Join(f.tmp, "no-such-agmsg-home"))
	orig := doctorCodexSandboxEnv
	doctorCodexSandboxEnv = f.resolveEnv
	t.Cleanup(func() { doctorCodexSandboxEnv = orig })

	// The bare repo has no scaffold, so other checks may fail; only the
	// order of the two lines matters here.
	var out string
	runErr := captureStdoutText(t, &out, func() error { return runDoctor(f.repo) })

	lines := strings.Split(out, "\n")
	for i, line := range lines {
		if !strings.Contains(line, "Codex effective config:") {
			continue
		}
		if !strings.Contains(line, "(hook approvals: see Codex hook trust)") {
			t.Errorf("the Codex effective config line does not point at Codex hook trust: %s", line)
		}
		want := "  ✓ Codex hook trust: pass — 4 hooks in " + f.hooksPath + " approved in ~/.codex/config.toml"
		if i+1 >= len(lines) || !strings.HasPrefix(lines[i+1], want) {
			t.Errorf("the line after Codex effective config is not %q:\n%s", want, out)
		}
		return
	}
	t.Fatalf("no Codex effective config line (runDoctor: %v):\n%s", runErr, out)
}
