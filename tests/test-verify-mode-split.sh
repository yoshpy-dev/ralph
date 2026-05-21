#!/usr/bin/env sh
# Regression tests for HARNESS_VERIFY_MODE routing in language verifiers.
#
# These tests use fake language tools so they verify command dispatch without
# depending on Go, Rust, Python, Dart, or Node toolchains being installed.

set -eu

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

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
  if [ "$_expected" = "$_actual" ]; then
    record_pass "$_desc (exit $_actual)"
  else
    record_fail "$_desc (expected exit $_expected, got $_actual)"
  fi
}

assert_called() {
  _desc="$1"
  _needle="$2"
  _log="$3"
  if grep -Fqx -- "$_needle" "$_log" 2>/dev/null; then
    record_pass "$_desc"
  else
    record_fail "$_desc (missing call: $_needle)"
    if [ -f "$_log" ]; then
      sed 's/^/    call: /' "$_log"
    fi
  fi
}

assert_not_called() {
  _desc="$1"
  _needle="$2"
  _log="$3"
  if grep -Fqx -- "$_needle" "$_log" 2>/dev/null; then
    record_fail "$_desc (unexpected call: $_needle)"
  else
    record_pass "$_desc"
  fi
}

workdir="$(mktemp -d "${TMPDIR:-/tmp}/verify-mode-split.XXXXXX")"
cleanup() { rm -rf "$workdir"; }
trap cleanup EXIT

stub_dir="$workdir/stubs"
mkdir -p "$stub_dir"

make_stub() {
  _name="$1"
  _stub="$stub_dir/$_name"
  cat >"$_stub" <<'STUB'
#!/usr/bin/env sh
{
  printf '%s' "${0##*/}"
  for arg in "$@"; do
    printf ' %s' "$arg"
  done
  printf '\n'
} >>"$COMMAND_LOG"
exit 0
STUB
  chmod +x "$_stub"
}

for _tool in go gofmt golangci-lint staticcheck npm pnpm yarn python3 ruff mypy pytest cargo dart flutter; do
  make_stub "$_tool"
done

run_verify() {
  _verify="$1"
  _project="$2"
  _mode="$3"
  _calls="$4"
  _out="$5"
  : >"$_calls"
  (
    cd "$_project"
    COMMAND_LOG="$_calls" \
      HARNESS_VERIFY_MODE="$_mode" \
      PATH="$stub_dir:$PATH" \
      "$_verify"
  ) >"$_out" 2>&1
}

make_project_dir() {
  _name="$1"
  _dir="$workdir/$_name"
  mkdir -p "$_dir"
  printf '%s\n' "$_dir"
}

run_mode_case() {
  _label="$1"
  _verify="$2"
  _project="$3"
  _mode="$4"
  _expected_exit="$5"
  _calls="$workdir/$_label-$_mode.calls"
  _out="$workdir/$_label-$_mode.out"
  set +e
  run_verify "$_verify" "$_project" "$_mode" "$_calls" "$_out"
  _rc=$?
  set -e
  assert_exit "$_label mode=$_mode" "$_expected_exit" "$_rc"
  MODE_CALLS="$_calls"
}

check_go() {
  _verify="$PROJECT_ROOT/packs/languages/golang/verify.sh"

  _dir="$(make_project_dir go-static)"
  printf 'module example.com/go-static\n\ngo 1.22\n' >"$_dir/go.mod"
  run_mode_case go "$_verify" "$_dir" static 0
  assert_called "Go static runs gofmt" "gofmt -l ." "$MODE_CALLS"
  assert_called "Go static runs vet" "go vet ./..." "$MODE_CALLS"
  assert_called "Go static runs golangci-lint" "golangci-lint run ./..." "$MODE_CALLS"
  assert_called "Go static runs staticcheck" "staticcheck ./..." "$MODE_CALLS"
  assert_not_called "Go static skips tests" "go test ./..." "$MODE_CALLS"

  _dir="$(make_project_dir go-test)"
  printf 'module example.com/go-test\n\ngo 1.22\n' >"$_dir/go.mod"
  run_mode_case go "$_verify" "$_dir" test 0
  assert_called "Go test runs tests" "go test ./..." "$MODE_CALLS"
  assert_not_called "Go test skips gofmt" "gofmt -l ." "$MODE_CALLS"
  assert_not_called "Go test skips vet" "go vet ./..." "$MODE_CALLS"
  assert_not_called "Go test skips staticcheck" "staticcheck ./..." "$MODE_CALLS"

  _dir="$(make_project_dir go-all)"
  printf 'module example.com/go-all\n\ngo 1.22\n' >"$_dir/go.mod"
  run_mode_case go "$_verify" "$_dir" all 0
  assert_called "Go all runs gofmt" "gofmt -l ." "$MODE_CALLS"
  assert_called "Go all runs tests" "go test ./..." "$MODE_CALLS"

  _dir="$(make_project_dir go-unknown)"
  printf 'module example.com/go-unknown\n\ngo 1.22\n' >"$_dir/go.mod"
  run_mode_case go "$_verify" "$_dir" invalid 2
}

check_typescript() {
  _verify="$PROJECT_ROOT/packs/languages/typescript/verify.sh"

  _dir="$(make_project_dir ts-static)"
  cat >"$_dir/package.json" <<'JSON'
{"scripts":{"lint":"lint","typecheck":"typecheck","test":"test"}}
JSON
  run_mode_case typescript "$_verify" "$_dir" static 0
  assert_called "TypeScript static runs lint" "npm run lint --if-present" "$MODE_CALLS"
  assert_called "TypeScript static runs typecheck" "npm run typecheck --if-present" "$MODE_CALLS"
  assert_not_called "TypeScript static skips tests" "npm run test --if-present" "$MODE_CALLS"

  _dir="$(make_project_dir ts-test)"
  cat >"$_dir/package.json" <<'JSON'
{"scripts":{"lint":"lint","typecheck":"typecheck","test":"test"}}
JSON
  run_mode_case typescript "$_verify" "$_dir" test 0
  assert_called "TypeScript test runs tests" "npm run test --if-present" "$MODE_CALLS"
  assert_not_called "TypeScript test skips lint" "npm run lint --if-present" "$MODE_CALLS"
  assert_not_called "TypeScript test skips typecheck" "npm run typecheck --if-present" "$MODE_CALLS"

  _dir="$(make_project_dir ts-all)"
  cat >"$_dir/package.json" <<'JSON'
{"scripts":{"lint":"lint","typecheck":"typecheck","test":"test"}}
JSON
  run_mode_case typescript "$_verify" "$_dir" all 0
  assert_called "TypeScript all runs lint" "npm run lint --if-present" "$MODE_CALLS"
  assert_called "TypeScript all runs tests" "npm run test --if-present" "$MODE_CALLS"
}

check_python() {
  _verify="$PROJECT_ROOT/packs/languages/python/verify.sh"

  _dir="$(make_project_dir py-static)"
  printf '[project]\nname = "py-static"\n' >"$_dir/pyproject.toml"
  run_mode_case python "$_verify" "$_dir" static 0
  assert_called "Python static runs ruff" "ruff check ." "$MODE_CALLS"
  assert_called "Python static runs mypy" "mypy ." "$MODE_CALLS"
  assert_not_called "Python static skips pytest" "pytest -q" "$MODE_CALLS"

  _dir="$(make_project_dir py-test)"
  printf '[project]\nname = "py-test"\n' >"$_dir/pyproject.toml"
  run_mode_case python "$_verify" "$_dir" test 0
  assert_called "Python test runs pytest" "pytest -q" "$MODE_CALLS"
  assert_not_called "Python test skips ruff" "ruff check ." "$MODE_CALLS"
  assert_not_called "Python test skips mypy" "mypy ." "$MODE_CALLS"

  _dir="$(make_project_dir py-all)"
  printf '[project]\nname = "py-all"\n' >"$_dir/pyproject.toml"
  run_mode_case python "$_verify" "$_dir" all 0
  assert_called "Python all runs ruff" "ruff check ." "$MODE_CALLS"
  assert_called "Python all runs pytest" "pytest -q" "$MODE_CALLS"
}

check_rust() {
  _verify="$PROJECT_ROOT/packs/languages/rust/verify.sh"

  _dir="$(make_project_dir rust-static)"
  printf '[package]\nname = "rust_static"\nversion = "0.1.0"\n' >"$_dir/Cargo.toml"
  run_mode_case rust "$_verify" "$_dir" static 0
  assert_called "Rust static runs fmt" "cargo fmt --all --check" "$MODE_CALLS"
  assert_called "Rust static runs clippy" "cargo clippy --all-targets --all-features -- -D warnings" "$MODE_CALLS"
  assert_not_called "Rust static skips tests" "cargo test --all-features" "$MODE_CALLS"

  _dir="$(make_project_dir rust-test)"
  printf '[package]\nname = "rust_test"\nversion = "0.1.0"\n' >"$_dir/Cargo.toml"
  run_mode_case rust "$_verify" "$_dir" test 0
  assert_called "Rust test runs tests" "cargo test --all-features" "$MODE_CALLS"
  assert_not_called "Rust test skips fmt" "cargo fmt --all --check" "$MODE_CALLS"
  assert_not_called "Rust test skips clippy" "cargo clippy --all-targets --all-features -- -D warnings" "$MODE_CALLS"

  _dir="$(make_project_dir rust-all)"
  printf '[package]\nname = "rust_all"\nversion = "0.1.0"\n' >"$_dir/Cargo.toml"
  run_mode_case rust "$_verify" "$_dir" all 0
  assert_called "Rust all runs fmt" "cargo fmt --all --check" "$MODE_CALLS"
  assert_called "Rust all runs tests" "cargo test --all-features" "$MODE_CALLS"
}

check_dart() {
  _verify="$PROJECT_ROOT/packs/languages/dart/verify.sh"

  _dir="$(make_project_dir dart-static)"
  printf 'name: dart_static\n' >"$_dir/pubspec.yaml"
  run_mode_case dart "$_verify" "$_dir" static 0
  assert_called "Dart static runs format" "dart format --output=none --set-exit-if-changed ." "$MODE_CALLS"
  assert_called "Dart static runs analyze" "dart analyze --fatal-infos" "$MODE_CALLS"
  assert_not_called "Dart static skips tests" "dart test" "$MODE_CALLS"

  _dir="$(make_project_dir dart-test)"
  printf 'name: dart_test\n' >"$_dir/pubspec.yaml"
  mkdir -p "$_dir/test"
  run_mode_case dart "$_verify" "$_dir" test 0
  assert_called "Dart test runs tests" "dart test" "$MODE_CALLS"
  assert_not_called "Dart test skips format" "dart format --output=none --set-exit-if-changed ." "$MODE_CALLS"
  assert_not_called "Dart test skips analyze" "dart analyze --fatal-infos" "$MODE_CALLS"

  _dir="$(make_project_dir dart-all)"
  printf 'name: dart_all\n' >"$_dir/pubspec.yaml"
  mkdir -p "$_dir/test"
  run_mode_case dart "$_verify" "$_dir" all 0
  assert_called "Dart all runs format" "dart format --output=none --set-exit-if-changed ." "$MODE_CALLS"
  assert_called "Dart all runs tests" "dart test" "$MODE_CALLS"
}

check_go
check_typescript
check_python
check_rust
check_dart

printf '\n-- Summary --\n'
printf '  PASS: %d / %d\n' "$_pass" "$_total"
printf '  FAIL: %d\n' "$_fail"

if [ "$_fail" -gt 0 ]; then
  exit 1
fi
