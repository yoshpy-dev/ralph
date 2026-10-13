package org

import (
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

// AC11 of plan docs/plans/active/2026-10-13-org-gate.md with the shipped
// scripts: the gate runs the repo's own scripts/run-static-verify.sh,
// scripts/run-test.sh and scripts/run-verify.sh (and the two language
// detectors they call) in a scratch repository holding a go.mod, whose Go
// language pack verifier is a stub. The caller's environment carries the
// variables that would let run-verify.sh skip that verifier.

// gateRunVerifyScripts are the repo scripts the scratch repository gets as
// they are: the gate's two scripts and what they call.
var gateRunVerifyScripts = []string{
	"run-static-verify.sh", "run-test.sh", "run-verify.sh", "detect-languages.sh", "detect-changed-languages.sh",
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
var gateCallerVerifyEnv = []string{
	"RALPH_VERIFY_BASE=HEAD", "RALPH_VERIFY_SCOPE=changed", "RALPH_VERIFY_PROJECT_ROOTS=nowhere", "RALPH_VERIFY_SKIP_BRANCH_SECRET_SCAN=1",
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
// RALPH_VERIFY_BASE=HEAD, RALPH_VERIFY_SCOPE=changed,
// RALPH_VERIFY_PROJECT_ROOTS and RALPH_VERIFY_SKIP_BRANCH_SECRET_SCAN=1 in
// the caller's environment, the shipped run-static-verify.sh run directly
// skips the Go verifier and the branch secret scan and passes (the control),
// while the gate's run gets none of them: the Go verifier runs in the full
// scope with no project roots, in the static and then the test mode, and the
// branch secret scan runs in the static mode. A failing verifier is
// static_failed; a passing one passes, with the worktree still clean (what
// run-verify.sh writes is ignored).
func TestGate_RealRunVerify_FullScopeRunsTheLanguageVerifier(t *testing.T) {
	fx := newGateFixtureWith(t, gateFixtureOptions{populate: writeRunVerifyRepo})
	fx.env = append(fx.env, gateCallerVerifyEnv...)
	fx.setExit(t, "golang", 1)

	control := exec.Command("./" + gateStaticScript)
	control.Dir, control.Env = fx.worktree, fx.env
	if out, err := control.CombinedOutput(); err != nil || !strings.Contains(string(out), "==> Language packs selected: none") {
		t.Fatalf("control: run-static-verify.sh with the caller's environment = %v; want it to pass with no language pack:\n%s", err, out)
	}
	if calls := fx.calls(t); calls != nil {
		t.Fatalf("control ran %v; want neither the Go verifier nor the secret scan", calls)
	}
	if st := gitOut(t, fx.worktree, "status", "--porcelain", "--untracked-files=all"); st != "" {
		t.Fatalf("the control run left the worktree dirty:\n%s", st)
	}

	res := fx.gate(t)
	if res.Passed || res.Reason != GateReasonStaticFailed || !slices.Contains(res.Tail, "==> Some verifiers failed.") {
		t.Fatalf("gate with a failing Go verifier = %+v; want static_failed with run-verify.sh's summary in the tail", res)
	}
	staticCalls := []string{"golang:mode=static:scope=full:roots=", "secret-scan"}
	if calls := fx.calls(t); !slices.Equal(calls, staticCalls) {
		t.Fatalf("calls = %q; want %q", calls, staticCalls)
	}
	if events := fx.gateEvents(t); len(events) != 1 || events[0].Event != EventGateFailed ||
		!strings.HasPrefix(events[0].Details, "reason=static_failed ") {
		t.Fatalf("gate events = %+v; want one gate_failed static_failed", events)
	}

	if err := os.Remove(filepath.Join(fx.out, "calls")); err != nil {
		t.Fatal(err)
	}
	fx.setExit(t, "golang", 0)
	res = fx.gate(t)
	if !res.Passed {
		t.Fatalf("gate with a passing Go verifier = %+v; want a pass", res)
	}
	if want := append(staticCalls, "golang:mode=test:scope=full:roots="); !slices.Equal(fx.calls(t), want) {
		t.Fatalf("calls = %q; want %q", fx.calls(t), want)
	}
	if st := gitOut(t, fx.worktree, "status", "--porcelain", "--untracked-files=all"); st != "" {
		t.Fatalf("the gate's run left the worktree dirty:\n%s", st)
	}
}
