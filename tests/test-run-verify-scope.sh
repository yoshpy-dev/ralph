#!/usr/bin/env sh
set -eu

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

# Cases pick their scope explicitly; do not inherit it from the caller.
unset RALPH_VERIFY_SCOPE

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

# Pin git away from the developer's global and system configuration.
hermetic_home="$workdir/.home"
mkdir -p "$hermetic_home"
HOME="$hermetic_home"
GIT_CONFIG_GLOBAL="$hermetic_home/.gitconfig"
GIT_CONFIG_SYSTEM=/dev/null
GIT_CONFIG_NOSYSTEM=1
GIT_TERMINAL_PROMPT=0
export HOME GIT_CONFIG_GLOBAL GIT_CONFIG_SYSTEM GIT_CONFIG_NOSYSTEM GIT_TERMINAL_PROMPT

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

# Pristine copy of the fixture scripts and stub packs for the remote-backed cases.
skel="$workdir/skel"
cp -R "$repo" "$skel"

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

# Remote-backed fixtures: the diff base comes from the default branch, not from
# the branch's own upstream.

# Fixture on main whose only remote <remote> is a bare repo under $workdir, with
# main pushed and tracked. Prints the fixture path.
make_remote_fixture() {
  _name="$1"
  _remote="$2"
  _bare="$workdir/$_name.git"
  _fixture="$workdir/$_name"
  git init -q --bare "$_bare"
  cp -R "$skel" "$_fixture"
  (
    cd "$_fixture"
    git init -q
    git checkout -q -B main
    git config user.name "Ralph Test"
    git config user.email "ralph-test@example.com"
    printf '# test repo\n' > README.md
    git add .
    git commit -q -m "init"
    git remote add "$_remote" "$_bare"
    git push -q -u "$_remote" main
  )
  printf '%s\n' "$_fixture"
}

# Commit a Go module under service/ in the current directory.
commit_go_service() {
  mkdir -p service
  printf 'module example.com/service\n\ngo 1.22\n' > service/go.mod
  printf 'package main\n' > service/main.go
  git add service
  git commit -q -m "add go service"
}

# Pushed feature branch (upstream == HEAD) with a committed Go change: the
# /verify wrapper must still select the golang pack.
fixture="$(make_remote_fixture pushed origin)"
(
  cd "$fixture"
  git checkout -q -B feature
  commit_go_service
  git push -q -u origin feature
)
_head="$(cd "$fixture" && git rev-parse HEAD)"
_upstream="$(cd "$fixture" && git rev-parse '@{upstream}')"
if [ "$_head" = "$_upstream" ]; then
  record_pass "pushed fixture has upstream equal to HEAD"
else
  record_fail "pushed fixture has upstream equal to HEAD (HEAD=$_head upstream=$_upstream)"
fi
calls="$workdir/pushed.calls"
: > "$calls"
(
  cd "$fixture"
  COMMAND_LOG="$calls" ./scripts/run-static-verify.sh >/dev/null
)
assert_called "pushed branch static wrapper runs local gate in changed scope" "local:static:changed" "$calls"
assert_called "pushed branch static wrapper runs golang pack for committed change" "golang:static:changed:service" "$calls"
assert_not_called "pushed branch static wrapper skips unrelated python pack" "python:static:changed:" "$calls"

# Only remote is not named origin: an unpushed Go commit on main (tracking
# central/main) must still select the golang pack.
fixture="$(make_remote_fixture central central)"
(cd "$fixture" && commit_go_service)
calls="$workdir/central.calls"
: > "$calls"
(
  cd "$fixture"
  COMMAND_LOG="$calls" ./scripts/run-static-verify.sh >/dev/null
)
assert_called "central-only remote static wrapper runs local gate in changed scope" "local:static:changed" "$calls"
assert_called "central-only remote static wrapper runs golang pack for unpushed commit" "golang:static:changed:service" "$calls"
assert_not_called "central-only remote static wrapper skips unrelated python pack" "python:static:changed:" "$calls"

# D7 of plan 2026-10-30-script-local-config: a settings error must not let the
# changed scope pass with nothing verified. The fixture is on main with no
# remote and a clean tree (what run-verify.sh writes is ignored), so the
# automatic base is main itself and the changed scope finds no_changes; it
# holds scripts/ralph-config.sh, and its golang verifier fails. A stand-in
# ralph answers verify.base: with "" the run passes without the golang
# verifier (the control); with exit 3 the detector takes the full scope and
# the failing verifier fails the run.
# shellcheck source=lib/fake-ralph.sh
. "$PROJECT_ROOT/tests/lib/fake-ralph.sh"
fake_bin="$workdir/fake-bin"
fake_ralph_install "$fake_bin"
fixture="$workdir/config-error"
cp -R "$skel" "$fixture"
cp "$PROJECT_ROOT/scripts/ralph-config.sh" "$fixture/scripts/ralph-config.sh"
cat > "$fixture/packs/languages/golang/verify.sh" <<'SH'
#!/usr/bin/env sh
printf 'golang:%s:%s:%s\n' "$HARNESS_VERIFY_MODE" "$RALPH_VERIFY_SCOPE" "${RALPH_VERIFY_PROJECT_ROOTS:-}" >> "$COMMAND_LOG"
exit 1
SH
chmod +x "$fixture/packs/languages/golang/verify.sh"
printf '.harness/\ndocs/evidence/\n' > "$fixture/.gitignore"
(
  cd "$fixture"
  git init -q
  git checkout -q -B main
  git config user.name "Ralph Test"
  git config user.email "ralph-test@example.com"
  printf '# test repo\n' > README.md
  git add .
  git commit -q -m "init"
  commit_go_service
)

# run_config_case <answer> -- runs run-verify.sh in the changed scope and the
# test mode in the fixture, with the stand-in first on PATH answering
# verify.base with "" (answer "empty") or exit 3 (answer "error"); sets
# config_rc, and leaves the output in $config_out and the calls in $calls.
config_out="$workdir/config-error.out"
run_config_case() {
  fake_ralph_reset "$fake_bin"
  case "$1" in
    empty) fake_ralph_value "$fake_bin" verify.base "" ;;
    error) fake_ralph_fail "$fake_bin" verify.base 3 "config get: ralph.local.toml: [plan_visual].browser must be an absolute path" ;;
  esac
  calls="$workdir/config-$1.calls"
  : > "$calls"
  set +e
  (
    cd "$fixture"
    COMMAND_LOG="$calls" PATH="$fake_bin:$PATH" RALPH_VERIFY_SCOPE=changed HARNESS_VERIFY_MODE=test ./scripts/run-verify.sh
  ) > "$config_out" 2>&1
  config_rc=$?
  set -e
}

run_config_case empty
if [ "$config_rc" -eq 0 ] && grep -Fq '==> Language scope: changed (no_changes)' "$config_out"; then
  record_pass "control: with [verify].base empty, clean main passes as no_changes"
else
  record_fail "control: with [verify].base empty, clean main passes as no_changes (exit $config_rc)"
  sed 's/^/    out: /' "$config_out"
fi
assert_not_called "control: the failing golang pack does not run" "golang:test:changed:" "$calls"

run_config_case error
if [ "$config_rc" -ne 0 ]; then
  record_pass "settings error: the run fails (exit $config_rc)"
else
  record_fail "settings error: the run fails (exit 0)"
  sed 's/^/    out: /' "$config_out"
fi
if grep -Fq '==> Language scope: full fallback (config_error:verify.base)' "$config_out" &&
  ! grep -Fq 'no_changes' "$config_out"; then
  record_pass "settings error: full fallback with config_error:verify.base, not no_changes"
else
  record_fail "settings error: full fallback with config_error:verify.base, not no_changes"
  sed 's/^/    out: /' "$config_out"
fi
assert_called "settings error: the failing golang pack runs" "golang:test:changed:" "$calls"
assert_called "settings error: the python pack runs too" "python:test:changed:" "$calls"

printf '\n-- Summary --\n'
printf '  PASS: %d / %d\n' "$_pass" "$_total"
printf '  FAIL: %d\n' "$_fail"

if [ "$_fail" -gt 0 ]; then
  exit 1
fi
