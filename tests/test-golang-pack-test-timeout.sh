#!/usr/bin/env sh
# Regression tests for the Go pack's default test timeout.
#
# run_tests runs go test with -timeout=30m first in GOFLAGS, followed by the
# effective GOFLAGS (go env GOFLAGS: the variable, else the go env -w value),
# so a timeout the caller set comes later and wins. Part A records what a
# fake go receives; part B reads the deadline a test sees under the real go.
#
# GOLANG_VERIFY chooses the script under test (a scratch copy for a mutant).

set -eu

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
GOLANG_VERIFY="${GOLANG_VERIFY:-$PROJECT_ROOT/packs/languages/golang/verify.sh}"
case "$GOLANG_VERIFY" in
  /*) ;;
  *) GOLANG_VERIFY="$PWD/$GOLANG_VERIFY" ;;
esac

# Find the real go and its build cache before any PATH holds the fake go.
# The cache is passed on so part B reuses it (the pack sets GOCACHE only
# when it is unset).
real_go="$(command -v go 2>/dev/null || true)"
real_gocache=""
real_path=""
if [ -n "$real_go" ]; then
  real_gocache="$("$real_go" env GOCACHE 2>/dev/null || true)"
  real_path="$(dirname "$real_go"):$PATH"
fi

_pass=0
_fail=0
_total=0

record_pass() {
  _pass=$((_pass + 1))
  _total=$((_total + 1))
  printf '  PASS: %s\n' "$1"
}

record_fail() {
  _fail=$((_fail + 1))
  _total=$((_total + 1))
  printf '  FAIL: %s\n' "$1"
}

assert_exit() {
  _desc="$1"
  _expected="$2"
  _actual="$3"
  _output="$4"
  if [ "$_expected" = "$_actual" ]; then
    record_pass "$_desc (exit $_actual)"
  else
    record_fail "$_desc (expected exit $_expected, got $_actual)"
    sed 's/^/    out: /' "$_output" 2>/dev/null || true
  fi
}

assert_line() {
  _desc="$1"
  _needle="$2"
  _log="$3"
  if grep -Fqx -- "$_needle" "$_log" 2>/dev/null; then
    record_pass "$_desc"
  else
    record_fail "$_desc (missing line: $_needle)"
    sed 's/^/    log: /' "$_log" 2>/dev/null || true
  fi
}

assert_count() {
  _desc="$1"
  _expected="$2"
  _needle="$3"
  _log="$4"
  _actual="$(grep -Fc -- "$_needle" "$_log" 2>/dev/null || true)"
  if [ "$_expected" = "$_actual" ]; then
    record_pass "$_desc"
  else
    record_fail "$_desc (expected $_expected lines with $_needle, got $_actual)"
    sed 's/^/    log: /' "$_log" 2>/dev/null || true
  fi
}

workdir="$(mktemp -d "${TMPDIR:-/tmp}/golang-pack-test-timeout.XXXXXX")"
workdir="$(cd "$workdir" && pwd -P)"
cleanup() {
  rm -rf "$workdir"
}
trap cleanup EXIT HUP INT TERM

# case_file <label>: a file name under workdir for a case's logs.
case_file() {
  printf '%s/%s' "$workdir" "$(printf '%s' "$1" | tr -c 'A-Za-z0-9' '-')"
}

# ─── Part A: a fake go records what go test gets ──────────────────────────

fake_dir="$workdir/fake-bin"
fake_path="$fake_dir:$PATH"
mkdir -p "$fake_dir"

# The fake go logs every call as GOFLAGS=<value>|<args>|<dir> and answers
# go env GOFLAGS the way go does: the go env -w value stands in through
# FAKE_GO_ENV_GOFLAGS, and FAKE_GO_ENV_FAIL=1 makes go env fail.
cat >"$fake_dir/go" <<'STUB'
#!/usr/bin/env sh
printf 'GOFLAGS=%s|%s|%s\n' "${GOFLAGS:-}" "$*" "$(pwd)" >>"$FAKE_GO_LOG"
if [ "${1:-}" = env ] && [ "${2:-}" = GOFLAGS ]; then
  if [ "${FAKE_GO_ENV_FAIL:-}" = 1 ]; then
    echo "go: fake go env failure" >&2
    exit 1
  fi
  if [ -n "${FAKE_GO_ENV_GOFLAGS+set}" ]; then
    printf '%s\n' "$FAKE_GO_ENV_GOFLAGS"
  else
    printf '%s\n' "${GOFLAGS:-}"
  fi
fi
exit 0
STUB
chmod +x "$fake_dir/go"

# Static mode runs these too; stubs keep a real one on PATH out of the run.
for _tool in gofmt golangci-lint staticcheck; do
  cat >"$fake_dir/$_tool" <<'STUB'
#!/usr/bin/env sh
printf '%s|%s|%s\n' "${0##*/}" "$*" "$(pwd)" >>"$FAKE_GO_LOG"
exit 0
STUB
  chmod +x "$fake_dir/$_tool"
done

# Two Go roots, so each case checks one go test per root.
repo_a="$workdir/fake-repo"
mkdir -p "$repo_a/svc"
printf 'module example.com/root\n\ngo 1.21\n' >"$repo_a/go.mod"
printf 'module example.com/svc\n\ngo 1.21\n' >"$repo_a/svc/go.mod"

# run_fake <label> <mode> [VAR=value ...]: run the pack over repo_a with the
# fake go first on PATH and nothing of the caller's GOFLAGS; the case's
# assignments are applied last.
run_fake() {
  _label="$1"
  _mode="$2"
  shift 2
  FAKE_LOG="$(case_file "fake $_label").log"
  _out="$(case_file "fake $_label").out"
  : >"$FAKE_LOG"
  set +e
  (
    cd "$repo_a"
    unset GOFLAGS GOCACHE STATICCHECK_CACHE RALPH_VERIFY_PROJECT_ROOTS \
      FAKE_GO_ENV_FAIL FAKE_GO_ENV_GOFLAGS
    PATH="$fake_path"
    FAKE_GO_LOG="$FAKE_LOG"
    HARNESS_VERIFY_MODE="$_mode"
    export PATH FAKE_GO_LOG HARNESS_VERIFY_MODE
    env "$@" "$GOLANG_VERIFY"
  ) >"$_out" 2>&1
  _rc=$?
  set -e
  assert_exit "$_label: pack runs" 0 "$_rc" "$_out"
}

# expect_go_test <label> <goflags>: go test ran once in each root, with the
# args test ./... and exactly this GOFLAGS.
expect_go_test() {
  _label="$1"
  _want="$2"
  assert_line "$_label: go test in . gets GOFLAGS=$_want" \
    "GOFLAGS=$_want|test ./...|$repo_a" "$FAKE_LOG"
  assert_line "$_label: go test in svc gets GOFLAGS=$_want" \
    "GOFLAGS=$_want|test ./...|$repo_a/svc" "$FAKE_LOG"
  assert_count "$_label: one go test per root" 2 "|test " "$FAKE_LOG"
}

run_fake "no GOFLAGS" test
expect_go_test "no GOFLAGS" "-timeout=30m"

run_fake "GOFLAGS=-count=1" test GOFLAGS=-count=1
expect_go_test "GOFLAGS=-count=1" "-timeout=30m -count=1"

run_fake "quoted GOFLAGS entry" test 'GOFLAGS="-timeout=45m"'
expect_go_test "quoted GOFLAGS entry" '-timeout=30m "-timeout=45m"'

run_fake "go env -w value" test FAKE_GO_ENV_GOFLAGS=-timeout=45m
expect_go_test "go env -w value" "-timeout=30m -timeout=45m"

run_fake "go env fails" test FAKE_GO_ENV_FAIL=1 GOFLAGS=-count=1
expect_go_test "go env fails" "-timeout=30m -count=1"

run_fake "static mode" static
assert_line "static mode: go vet runs in ." "GOFLAGS=|vet ./...|$repo_a" "$FAKE_LOG"
assert_count "static mode: no go test" 0 "|test " "$FAKE_LOG"
assert_count "static mode: no go env" 0 "|env " "$FAKE_LOG"

# ─── Part B: the real go reports the deadline a test sees ─────────────────

if [ -z "$real_go" ]; then
  printf '  SKIP: real go cases (go is not on PATH)\n'
else
  mod_dir="$workdir/real-module"
  mkdir -p "$mod_dir"
  printf 'module example.com/deadline\n\ngo 1.21\n' >"$mod_dir/go.mod"
  printf 'package deadline\n' >"$mod_dir/deadline.go"
  # The test writes the minutes to its deadline, or none, to $DEADLINE_OUT.
  cat >"$mod_dir/deadline_test.go" <<'GO'
package deadline

import (
	"os"
	"strconv"
	"testing"
	"time"
)

func TestDeadline(t *testing.T) {
	out := "none"
	if d, ok := t.Deadline(); ok {
		out = strconv.Itoa(int(time.Until(d).Round(time.Minute).Minutes()))
	}
	if err := os.WriteFile(os.Getenv("DEADLINE_OUT"), []byte(out+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
}
GO

  # Scratch GOENV files, so the user's go.env is never read or written.
  goenv_empty="$workdir/goenv-empty"
  goenv_45="$workdir/goenv-45"
  : >"$goenv_empty"
  set +e
  (
    cd "$workdir"
    unset GOFLAGS
    GOENV="$goenv_45" GOTOOLCHAIN=local "$real_go" env -w GOFLAGS=-timeout=45m
  ) >"$workdir/goenv-45.out" 2>&1
  _rc=$?
  set -e
  assert_exit "go env -w GOFLAGS=-timeout=45m into a scratch GOENV" 0 "$_rc" "$workdir/goenv-45.out"
  assert_line "the scratch GOENV holds the value" "GOFLAGS=-timeout=45m" "$goenv_45"

  # run_real <label> [VAR=value ...]: run the pack over mod_dir with the
  # real go and nothing of the caller's GOFLAGS; the case's assignments are
  # applied last. Each case has its own DEADLINE_OUT: go caches a test's
  # result by the environment variables the test reads.
  run_real() {
    _label="$1"
    shift
    DEADLINE_FILE="$(case_file "$_label").deadline"
    _out="$(case_file "$_label").out"
    set +e
    (
      cd "$mod_dir"
      unset GOFLAGS GOCACHE STATICCHECK_CACHE RALPH_VERIFY_PROJECT_ROOTS
      if [ -n "$real_gocache" ]; then
        GOCACHE="$real_gocache"
        export GOCACHE
      fi
      PATH="$real_path"
      GOENV="$goenv_empty"
      GOTOOLCHAIN=local
      GOWORK=off
      GOPROXY=off
      HARNESS_VERIFY_MODE="test"
      DEADLINE_OUT="$DEADLINE_FILE"
      export PATH GOENV GOTOOLCHAIN GOWORK GOPROXY HARNESS_VERIFY_MODE DEADLINE_OUT
      env "$@" "$GOLANG_VERIFY"
    ) >"$_out" 2>&1
    _rc=$?
    set -e
    assert_exit "$_label: pack runs with the real go" 0 "$_rc" "$_out"
  }

  expect_deadline() {
    _label="$1"
    _want="$2"
    _got="$(sed -n '1p' "$DEADLINE_FILE" 2>/dev/null || true)"
    if [ "$_got" = "$_want" ]; then
      record_pass "$_label: the test's deadline is $_want minutes away"
    else
      record_fail "$_label: expected a deadline $_want minutes away, got ${_got:-no file}"
    fi
  }

  run_real "real go, no GOFLAGS"
  expect_deadline "real go, no GOFLAGS" 30

  run_real "real go, GOFLAGS=-timeout=45m" GOFLAGS=-timeout=45m
  expect_deadline "real go, GOFLAGS=-timeout=45m" 45

  run_real "real go, quoted GOFLAGS entry" 'GOFLAGS="-timeout=45m"'
  expect_deadline "real go, quoted GOFLAGS entry" 45

  run_real "real go, go env -w value" GOENV="$goenv_45"
  expect_deadline "real go, go env -w value" 45
fi

printf '\n-- Summary --\n'
printf '  PASS: %d / %d\n' "$_pass" "$_total"
printf '  FAIL: %d\n' "$_fail"

if [ "$_fail" -gt 0 ]; then
  exit 1
fi
