package org

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/yoshpy-dev/ralph/internal/config"
)

// AC11 of plan docs/plans/active/2026-10-13-org-gate.md with the shipped
// scripts: the gate runs the repo's own scripts/run-static-verify.sh,
// scripts/run-test.sh and scripts/run-verify.sh (and the two language
// detectors they call) in a scratch repository holding a go.mod, whose Go
// language pack verifier is a stub. The caller's environment carries the
// variables, and a stand-in ralph the personal ralph.local.toml values, that
// would let run-verify.sh skip that verifier and the branch secret scan.

// gateRunVerifyScripts are the repo scripts the scratch repository gets as
// they are: the gate's two scripts and what they call, ralph-config.sh, which
// reads the settings, included.
var gateRunVerifyScripts = []string{
	"run-static-verify.sh", "run-test.sh", "run-verify.sh", "detect-languages.sh", "detect-changed-languages.sh",
	"ralph-config.sh",
}

// gateGolangVerifyStub is packs/languages/golang/verify.sh: it appends one
// line with the mode, scope and project roots it got to $GATE_TEST_OUT/calls
// and exits with the number in $GATE_TEST_OUT/exit-golang (0 when there is
// none).
const gateGolangVerifyStub = `#!/bin/sh
printf 'golang:mode=%s:scope=%s:roots=%s\n' "${HARNESS_VERIFY_MODE:-}" "${RALPH_VERIFY_SCOPE:-}" "${RALPH_VERIFY_PROJECT_ROOTS:-}" >> "$GATE_TEST_OUT/calls"
echo "golang verifier ran"
exit "$(cat "$GATE_TEST_OUT/exit-golang" 2>/dev/null || echo 0)"
`

// gateSecretScanStub is scripts/secret-scan-branch.sh: it records that the
// branch secret scan ran and passes.
const gateSecretScanStub = `#!/bin/sh
echo secret-scan >> "$GATE_TEST_OUT/calls"
`

// gateCallerVerifyEnv is the caller's verification environment of AC11.
// No variable sets the diff base or skips the branch secret scan any more:
// ralph.local.toml's [verify].base and skip_branch_secret_scan do, and
// gateRalphStub answers them (plan 2026-10-30-script-local-config).
var gateCallerVerifyEnv = []string{"RALPH_VERIFY_SCOPE=changed", "RALPH_VERIFY_PROJECT_ROOTS=nowhere"}

// gateRalphStub is a stand-in ralph for the scratch repository's
// scripts/ralph-config.sh. It answers `ralph config get` as ralph does for a
// developer whose ralph.local.toml sets [verify].base = "HEAD" and
// skip_branch_secret_scan = true, values that let run-verify.sh skip the Go
// verifier and the branch secret scan, and as ralph does with
// RALPH_LOCAL_CONFIG=off, which keeps it off that file: the defaults "" and
// false. Any other call exits 2. Each call appends
// <key>:<RALPH_LOCAL_CONFIG> to $GATE_TEST_OUT/ralph-calls.
const gateRalphStub = `#!/bin/sh
printf '%s:%s\n' "$3" "${RALPH_LOCAL_CONFIG-}" >> "$GATE_TEST_OUT/ralph-calls"
case "$1 $2 $3:${RALPH_LOCAL_CONFIG-}" in
"config get verify.base:off") echo ;;
"config get verify.base:"*) echo HEAD ;;
"config get verify.skip_branch_secret_scan:off") echo false ;;
"config get verify.skip_branch_secret_scan:"*) echo true ;;
*) exit 2 ;;
esac
`

// gateRalphStubEnv is env with gateRalphStub, in a directory of its own,
// first on PATH and RALPH_LOCAL_CONFIG empty, as in a developer's shell,
// where ralph reads the personal file: env's PATH and RALPH_LOCAL_CONFIG
// entries (TestMain sets it to off) are replaced.
func gateRalphStubEnv(t *testing.T, env []string) []string {
	t.Helper()
	bin := t.TempDir()
	writeGateFile(t, bin, "ralph", gateRalphStub, 0o755)
	var out []string
	for _, kv := range env {
		if name, _, _ := strings.Cut(kv, "="); name != "PATH" && name != config.LocalConfigEnv {
			out = append(out, kv)
		}
	}
	return append(out, "PATH="+bin+string(os.PathListSeparator)+os.Getenv("PATH"), config.LocalConfigEnv+"=")
}

// gateRalphCalls returns what gateRalphStub was asked, in order: nil when
// it was not run.
func gateRalphCalls(t *testing.T, fx *gateFixture) []string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(fx.out, "ralph-calls"))
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		t.Fatal(err)
	}
	return strings.Fields(string(data))
}

// writeRunVerifyRepo writes, in dir, the repo scripts gateRunVerifyScripts,
// the secret scan stub, a go.mod, and the Go verifier stub.
func writeRunVerifyRepo(t *testing.T, dir string) {
	t.Helper()
	for _, name := range gateRunVerifyScripts {
		data, err := os.ReadFile(filepath.Join(splitTestRepoRoot(t), "scripts", name))
		if err != nil {
			t.Fatal(err)
		}
		writeGateFile(t, dir, "scripts/"+name, string(data), 0o755)
	}
	writeGateFile(t, dir, "scripts/secret-scan-branch.sh", gateSecretScanStub, 0o755)
	writeGateFile(t, dir, "go.mod", "module example.com/gate\n\ngo 1.25\n", 0o644)
	writeGateFile(t, dir, "packs/languages/golang/verify.sh", gateGolangVerifyStub, 0o755)
}

// TestGate_RealRunVerify_FullScopeRunsTheLanguageVerifier covers AC11: with
// RALPH_VERIFY_SCOPE=changed and RALPH_VERIFY_PROJECT_ROOTS in the caller's
// environment, and a personal ralph.local.toml whose [verify].base is HEAD
// and whose skip_branch_secret_scan is true (gateRalphStub), the shipped
// run-static-verify.sh run directly skips the Go verifier, although the
// feature branch commits a Go file, and the branch secret scan, and passes
// (the control). The gate's run gets none of them: with RALPH_LOCAL_CONFIG=off
// ralph answers the skip with its default false (AC4 of plan
// 2026-10-30-script-local-config), the Go verifier runs in the full scope with
// no project roots, in the static and then the test mode, and the branch
// secret scan runs in the static mode. A failing verifier is static_failed; a
// passing one passes, with the worktree still clean (what run-verify.sh writes
// is ignored).
func TestGate_RealRunVerify_FullScopeRunsTheLanguageVerifier(t *testing.T) {
	fx := newGateFixtureWith(t, gateFixtureOptions{populate: writeRunVerifyRepo})
	fx.env = append(gateRalphStubEnv(t, fx.env), gateCallerVerifyEnv...)
	fx.setExit(t, "golang", 1)
	writeGateFile(t, fx.worktree, "gate.go", "package gate\n", 0o644)
	runGit(t, "-C", fx.worktree, "add", "gate.go")
	gateCommit(t, fx.worktree, "add gate.go")

	control := exec.Command("./" + GateStaticScript)
	control.Dir, control.Env = fx.worktree, fx.env
	if out, err := control.CombinedOutput(); err != nil || !strings.Contains(string(out), "==> Language packs selected: none") ||
		!strings.Contains(string(out), "==> Skipping branch secret scan ([verify].skip_branch_secret_scan = true)") {
		t.Fatalf("control: run-static-verify.sh with the caller's environment = %v; want it to pass with no language pack and the scan skipped:\n%s", err, out)
	}
	if calls := fx.calls(t); calls != nil {
		t.Fatalf("control ran %v; want neither the Go verifier nor the secret scan", calls)
	}
	if got, want := gateRalphCalls(t, fx), []string{"verify.base:", "verify.skip_branch_secret_scan:"}; !slices.Equal(got, want) {
		t.Fatalf("the control asked ralph %q; want %q", got, want)
	}
	if st := gitOut(t, fx.worktree, "status", "--porcelain", "--untracked-files=all"); st != "" {
		t.Fatalf("the control run left the worktree dirty:\n%s", st)
	}

	// The gate's static run asks ralph for the skip with RALPH_LOCAL_CONFIG=off
	// and gets false; it runs no changed scope, so it never asks for the base.
	gateRalph := []string{"verify.skip_branch_secret_scan:off"}
	for _, name := range []string{"calls", "ralph-calls"} {
		if err := os.Remove(filepath.Join(fx.out, name)); err != nil && !errors.Is(err, os.ErrNotExist) {
			t.Fatal(err)
		}
	}
	res := fx.gate(t)
	if res.Passed || res.Reason != GateReasonStaticFailed || !slices.Contains(res.Tail, "==> Some verifiers failed.") {
		t.Fatalf("gate with a failing Go verifier = %+v; want static_failed with run-verify.sh's summary in the tail", res)
	}
	staticCalls := []string{"golang:mode=static:scope=full:roots=", "secret-scan"}
	if calls := fx.calls(t); !slices.Equal(calls, staticCalls) {
		t.Fatalf("calls = %q; want %q", calls, staticCalls)
	}
	if got := gateRalphCalls(t, fx); !slices.Equal(got, gateRalph) {
		t.Fatalf("the gate asked ralph %q; want %q", got, gateRalph)
	}
	if events := fx.gateEvents(t); len(events) != 1 || events[0].Event != EventGateFailed ||
		!strings.HasPrefix(events[0].Details, "reason=static_failed ") {
		t.Fatalf("gate events = %+v; want one gate_failed static_failed", events)
	}

	for _, name := range []string{"calls", "ralph-calls"} {
		if err := os.Remove(filepath.Join(fx.out, name)); err != nil {
			t.Fatal(err)
		}
	}
	fx.setExit(t, "golang", 0)
	res = fx.gate(t)
	if !res.Passed {
		t.Fatalf("gate with a passing Go verifier = %+v; want a pass", res)
	}
	if want := append(staticCalls, "golang:mode=test:scope=full:roots="); !slices.Equal(fx.calls(t), want) {
		t.Fatalf("calls = %q; want %q", fx.calls(t), want)
	}
	if got := gateRalphCalls(t, fx); !slices.Equal(got, gateRalph) {
		t.Fatalf("the gate asked ralph %q; want %q", got, gateRalph)
	}
	if st := gitOut(t, fx.worktree, "status", "--porcelain", "--untracked-files=all"); st != "" {
		t.Fatalf("the gate's run left the worktree dirty:\n%s", st)
	}
}
