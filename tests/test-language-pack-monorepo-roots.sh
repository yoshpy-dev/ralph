#!/usr/bin/env sh
# Regression tests for language pack verifier execution in nested monorepo roots.

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
    sed 's/^/    call: /' "$_log"
  else
    record_pass "$_desc"
  fi
}

assert_contains() {
  _desc="$1"
  _needle="$2"
  _haystack="$3"
  case " $_haystack " in
    *" $_needle "*) record_pass "$_desc" ;;
    *)
      record_fail "$_desc (expected output to contain $_needle, got $_haystack)"
      ;;
  esac
}

assert_not_contains() {
  _desc="$1"
  _needle="$2"
  _haystack="$3"
  case " $_haystack " in
    *" $_needle "*) record_fail "$_desc (unexpected output: $_needle in $_haystack)" ;;
    *) record_pass "$_desc" ;;
  esac
}

workdir="$(mktemp -d "${TMPDIR:-/tmp}/language-pack-roots.XXXXXX")"
workdir="$(cd "$workdir" && pwd -P)"
cleanup() {
  rm -rf "$workdir"
}
trap cleanup EXIT HUP INT TERM

stub_dir="$workdir/stubs"
mkdir -p "$stub_dir"

make_stub() {
  _name="$1"
  _stub="$stub_dir/$_name"
  cat >"$_stub" <<'STUB'
#!/usr/bin/env sh
{
  printf '%s|%s|' "${0##*/}" "$(pwd)"
  first=1
  for arg in "$@"; do
    if [ "$first" -eq 1 ]; then
      first=0
    else
      printf ' '
    fi
    printf '%s' "$arg"
  done
  printf '\n'
} >>"$COMMAND_LOG"
exit 0
STUB
  chmod +x "$_stub"
}

for _tool in npm pnpm yarn python3 ruff mypy pytest cargo go gofmt golangci-lint staticcheck dart flutter terraform tofu tflint tfsec trivy; do
  make_stub "$_tool"
done

run_verify() {
  _label="$1"
  _verify="$2"
  _repo="$3"
  _mode="$4"
  _calls="$workdir/$_label.calls"
  _out="$workdir/$_label.out"
  : >"$_calls"

  set +e
  (
    cd "$_repo"
    COMMAND_LOG="$_calls" \
      HARNESS_VERIFY_MODE="$_mode" \
      PATH="$stub_dir:$PATH" \
      "$_verify"
  ) >"$_out" 2>&1
  _rc=$?
  set -e

  assert_exit "$_label verifier" 0 "$_rc"
  CALL_LOG="$_calls"
}

# TypeScript: repo root has no package.json, but two nested packages run.
repo_ts="$workdir/typescript"
mkdir -p "$repo_ts/packages/web" "$repo_ts/packages/admin"
cat >"$repo_ts/packages/web/package.json" <<'JSON'
{"scripts":{"lint":"lint","typecheck":"typecheck","test":"test"}}
JSON
cat >"$repo_ts/packages/admin/package.json" <<'JSON'
{"scripts":{"lint":"lint","typecheck":"typecheck","test":"test"}}
JSON
run_verify "typescript" "$PROJECT_ROOT/packs/languages/typescript/verify.sh" "$repo_ts" static
assert_called "TypeScript runs lint in nested web root" "npm|$repo_ts/packages/web|run lint --if-present" "$CALL_LOG"
assert_called "TypeScript runs typecheck in nested admin root" "npm|$repo_ts/packages/admin|run typecheck --if-present" "$CALL_LOG"

filtered_calls="$workdir/typescript-filtered.calls"
: >"$filtered_calls"
(
  cd "$repo_ts"
  COMMAND_LOG="$filtered_calls" \
    HARNESS_VERIFY_MODE=static \
    RALPH_VERIFY_PROJECT_ROOTS=packages/web \
    PATH="$stub_dir:$PATH" \
    "$PROJECT_ROOT/packs/languages/typescript/verify.sh"
) >/dev/null 2>&1
assert_called "TypeScript root filter keeps selected root" "npm|$repo_ts/packages/web|run lint --if-present" "$filtered_calls"
if grep -Fq "packages/admin" "$filtered_calls" 2>/dev/null; then
  record_fail "TypeScript root filter excludes unselected root"
else
  record_pass "TypeScript root filter excludes unselected root"
fi

# Python: nested pyproject.toml dispatches static tools from that root.
repo_py="$workdir/python"
mkdir -p "$repo_py/services/api"
printf '[project]\nname = "api"\n' >"$repo_py/services/api/pyproject.toml"
run_verify "python" "$PROJECT_ROOT/packs/languages/python/verify.sh" "$repo_py" static
assert_called "Python runs ruff in nested root" "ruff|$repo_py/services/api|check ." "$CALL_LOG"
assert_called "Python runs mypy in nested root" "mypy|$repo_py/services/api|." "$CALL_LOG"

# Rust: nested Cargo.toml dispatches cargo commands from that crate root.
repo_rust="$workdir/rust"
mkdir -p "$repo_rust/crates/core"
printf '[package]\nname = "core"\nversion = "0.1.0"\n' >"$repo_rust/crates/core/Cargo.toml"
run_verify "rust" "$PROJECT_ROOT/packs/languages/rust/verify.sh" "$repo_rust" static
assert_called "Rust runs fmt in nested crate root" "cargo|$repo_rust/crates/core|fmt --all --check" "$CALL_LOG"
assert_called "Rust runs clippy in nested crate root" "cargo|$repo_rust/crates/core|clippy --all-targets --all-features -- -D warnings" "$CALL_LOG"

# Go: nested go.mod dispatches go commands from that module root.
repo_go="$workdir/golang"
mkdir -p "$repo_go/services/worker"
printf 'module example.com/worker\n\ngo 1.22\n' >"$repo_go/services/worker/go.mod"
run_verify "golang" "$PROJECT_ROOT/packs/languages/golang/verify.sh" "$repo_go" static
assert_called "Go runs gofmt in nested module root" "gofmt|$repo_go/services/worker|-l ." "$CALL_LOG"
assert_called "Go runs vet in nested module root" "go|$repo_go/services/worker|vet ./..." "$CALL_LOG"

# Dart: nested pubspec.yaml dispatches non-mutating format and analysis.
repo_dart="$workdir/dart"
mkdir -p "$repo_dart/apps/mobile"
printf 'name: mobile\n' >"$repo_dart/apps/mobile/pubspec.yaml"
run_verify "dart" "$PROJECT_ROOT/packs/languages/dart/verify.sh" "$repo_dart" static
assert_called "Dart runs non-mutating format in nested root" "dart|$repo_dart/apps/mobile|format --output=none --set-exit-if-changed ." "$CALL_LOG"
assert_called "Dart runs analysis in nested root" "dart|$repo_dart/apps/mobile|analyze --fatal-infos" "$CALL_LOG"

# Terraform: nested module dispatches IaC CLI from that root and prunes cache noise.
repo_tf="$workdir/terraform"
mkdir -p "$repo_tf/infra/prod" "$repo_tf/infra/stage" "$repo_tf/.terraform/modules/noise"
printf 'terraform {}\n' >"$repo_tf/infra/prod/main.tf"
printf 'terraform {}\n' >"$repo_tf/infra/stage/main.tf"
printf 'terraform {}\n' >"$repo_tf/.terraform/modules/noise/main.tf"
run_verify "terraform" "$PROJECT_ROOT/packs/languages/terraform/verify.sh" "$repo_tf" static
assert_called "Terraform runs fmt in nested root" "tofu|$repo_tf/infra/prod|fmt -check" "$CALL_LOG"
assert_called "Terraform runs fmt in each nested root" "tofu|$repo_tf/infra/stage|fmt -check" "$CALL_LOG"
assert_not_called "Terraform default skips backendless init" "tofu|$repo_tf/infra/prod|init -backend=false -input=false" "$CALL_LOG"
assert_not_called "Terraform default skips validate without .terraform" "tofu|$repo_tf/infra/prod|validate" "$CALL_LOG"

terraform_opt_calls="$workdir/terraform-opt-in.calls"
: >"$terraform_opt_calls"
(
  cd "$repo_tf"
  COMMAND_LOG="$terraform_opt_calls" \
    HARNESS_VERIFY_MODE=static \
    RALPH_TERRAFORM_INIT_VALIDATE=true \
    PATH="$stub_dir:$PATH" \
    "$PROJECT_ROOT/packs/languages/terraform/verify.sh"
) >/dev/null 2>&1
assert_called "Terraform opt-in runs backendless init in nested root" "tofu|$repo_tf/infra/prod|init -backend=false -input=false" "$terraform_opt_calls"
assert_called "Terraform opt-in runs validate after init in nested root" "tofu|$repo_tf/infra/prod|validate" "$terraform_opt_calls"
assert_called "Terraform opt-in runs backendless init in each nested root" "tofu|$repo_tf/infra/stage|init -backend=false -input=false" "$terraform_opt_calls"
assert_called "Terraform opt-in runs validate in each nested root" "tofu|$repo_tf/infra/stage|validate" "$terraform_opt_calls"
if grep -Fq ".terraform/modules/noise" "$terraform_opt_calls" 2>/dev/null; then
  record_fail "Terraform opt-in prunes cache roots"
else
  record_pass "Terraform opt-in prunes cache roots"
fi

# Detection: nested go.mod is detected, JVM markers are intentionally not emitted.
repo_detect="$workdir/detect"
mkdir -p "$repo_detect/backend"
printf 'module example.com/backend\n\ngo 1.22\n' >"$repo_detect/backend/go.mod"
printf 'plugins {}\n' >"$repo_detect/build.gradle"
detect_out="$(cd "$repo_detect" && "$PROJECT_ROOT/scripts/detect-languages.sh" 2>/dev/null)"
assert_contains "detect-languages finds nested Go modules" "golang" "$detect_out"
assert_not_contains "detect-languages does not emit jvm without a shipped pack" "jvm" "$detect_out"

printf '\n-- Summary --\n'
printf '  PASS: %d / %d\n' "$_pass" "$_total"
printf '  FAIL: %d\n' "$_fail"

if [ "$_fail" -gt 0 ]; then
  exit 1
fi
