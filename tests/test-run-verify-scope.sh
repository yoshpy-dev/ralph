#!/usr/bin/env sh
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

assert_called() {
  _desc="$1"
  _needle="$2"
  _log="$3"
  if grep -Fqx -- "$_needle" "$_log" 2>/dev/null; then
    record_pass "$_desc"
  else
    record_fail "$_desc (missing call: $_needle)"
    [ -f "$_log" ] && sed 's/^/    call: /' "$_log"
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

workdir="$(mktemp -d "${TMPDIR:-/tmp}/run-verify-scope.XXXXXX")"
cleanup() { rm -rf "$workdir"; }
trap cleanup EXIT HUP INT TERM

repo="$workdir/repo"
mkdir -p "$repo/scripts" "$repo/packs/languages/golang" "$repo/packs/languages/python"
cp "$PROJECT_ROOT/scripts/run-verify.sh" "$repo/scripts/run-verify.sh"
cp "$PROJECT_ROOT/scripts/run-static-verify.sh" "$repo/scripts/run-static-verify.sh"
cp "$PROJECT_ROOT/scripts/run-test.sh" "$repo/scripts/run-test.sh"
cp "$PROJECT_ROOT/scripts/detect-changed-languages.sh" "$repo/scripts/detect-changed-languages.sh"
chmod +x "$repo/scripts/"*.sh

cat > "$repo/scripts/detect-languages.sh" <<'SH'
#!/usr/bin/env sh
printf 'golang\npython\n'
SH
chmod +x "$repo/scripts/detect-languages.sh"

cat > "$repo/scripts/verify.local.sh" <<'SH'
#!/usr/bin/env sh
printf 'local:%s:%s\n' "$HARNESS_VERIFY_MODE" "$RALPH_VERIFY_SCOPE" >> "$COMMAND_LOG"
SH
chmod +x "$repo/scripts/verify.local.sh"

cat > "$repo/packs/languages/golang/verify.sh" <<'SH'
#!/usr/bin/env sh
printf 'golang:%s:%s:%s\n' "$HARNESS_VERIFY_MODE" "$RALPH_VERIFY_SCOPE" "${RALPH_VERIFY_PROJECT_ROOTS:-}" >> "$COMMAND_LOG"
SH
chmod +x "$repo/packs/languages/golang/verify.sh"

cat > "$repo/packs/languages/python/verify.sh" <<'SH'
#!/usr/bin/env sh
printf 'python:%s:%s:%s\n' "$HARNESS_VERIFY_MODE" "$RALPH_VERIFY_SCOPE" "${RALPH_VERIFY_PROJECT_ROOTS:-}" >> "$COMMAND_LOG"
SH
chmod +x "$repo/packs/languages/python/verify.sh"

(
  cd "$repo"
  git init -q
  git checkout -q -B main
  git config user.name "Ralph Test"
  git config user.email "ralph-test@example.com"
  printf '# test repo\n' > README.md
  git add .
  git commit -q -m "init"
)

# Changed scope: local gate always runs; only the changed language pack runs.
mkdir -p "$repo/service"
printf 'module example.com/service\n\ngo 1.22\n' > "$repo/service/go.mod"
printf 'package main\n' > "$repo/service/main.go"
calls="$workdir/changed.calls"
: > "$calls"
(
  cd "$repo"
  COMMAND_LOG="$calls" RALPH_VERIFY_SCOPE=changed HARNESS_VERIFY_MODE=static ./scripts/run-verify.sh >/dev/null
)
assert_called "changed scope runs local static gate" "local:static:changed" "$calls"
assert_called "changed scope runs changed language pack root" "golang:static:changed:service" "$calls"
assert_not_called "changed scope skips unrelated language pack" "python:static:changed:" "$calls"

# Wrapper default: /test wrapper defaults RALPH_VERIFY_SCOPE to changed.
calls="$workdir/wrapper.calls"
: > "$calls"
(
  cd "$repo"
  unset RALPH_VERIFY_SCOPE
  COMMAND_LOG="$calls" ./scripts/run-test.sh >/dev/null
)
assert_called "test wrapper defaults local gate to changed" "local:test:changed" "$calls"
assert_called "test wrapper defaults language pack to changed root" "golang:test:changed:service" "$calls"
assert_not_called "test wrapper skips unrelated pack by default" "python:test:changed:" "$calls"

# Wrapper override: explicit full scope is honored, as used by CI.
calls="$workdir/wrapper-full.calls"
: > "$calls"
(
  cd "$repo"
  COMMAND_LOG="$calls" RALPH_VERIFY_SCOPE=full ./scripts/run-test.sh >/dev/null
)
assert_called "test wrapper honors explicit full local gate" "local:test:full" "$calls"
assert_called "test wrapper honors explicit full golang pack" "golang:test:full:" "$calls"
assert_called "test wrapper honors explicit full python pack" "python:test:full:" "$calls"

# Shared changes fall back to full and run every detected language pack.
(
  cd "$repo"
  git add service
  git commit -q -m "add go"
  mkdir -p .github/workflows
  printf 'name: verify\n' > .github/workflows/verify.yml
)
calls="$workdir/full.calls"
: > "$calls"
(
  cd "$repo"
  COMMAND_LOG="$calls" RALPH_VERIFY_SCOPE=changed HARNESS_VERIFY_MODE=test ./scripts/run-verify.sh >/dev/null
)
assert_called "full fallback runs local test gate" "local:test:changed" "$calls"
assert_called "full fallback runs golang pack" "golang:test:changed:" "$calls"
assert_called "full fallback runs python pack" "python:test:changed:" "$calls"

printf '\n-- Summary --\n'
printf '  PASS: %d / %d\n' "$_pass" "$_total"
printf '  FAIL: %d\n' "$_fail"

if [ "$_fail" -gt 0 ]; then
  exit 1
fi
