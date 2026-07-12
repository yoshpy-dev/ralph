#!/usr/bin/env sh
set -eu

# Tests for the legacy shell CLI deprecation notice (AC4).
# The notice must appear on stderr when a Go 'ralph' binary is on PATH,
# be absent when it is not, be suppressed by RALPH_NO_DEPRECATION=1,
# and must not affect stdout.

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
RALPH="${PROJECT_ROOT}/scripts/ralph"

pass=0
fail=0

check_eq() {
  label="$1"
  want="$2"
  got="$3"
  if [ "$want" = "$got" ]; then
    printf '  PASS: %s\n' "$label"
    pass=$((pass + 1))
  else
    printf '  FAIL: %s\n' "$label"
    printf '    want: %s\n' "$want"
    printf '    got:  %s\n' "$got"
    fail=$((fail + 1))
  fi
}

check_contains() {
  label="$1"
  needle="$2"
  haystack="$3"
  if printf '%s' "$haystack" | grep -qF "$needle"; then
    printf '  PASS: %s\n' "$label"
    pass=$((pass + 1))
  else
    printf '  FAIL: %s\n' "$label"
    printf '    missing: %s\n' "$needle"
    fail=$((fail + 1))
  fi
}

check_not_contains() {
  label="$1"
  needle="$2"
  haystack="$3"
  if printf '%s' "$haystack" | grep -qF "$needle"; then
    printf '  FAIL: %s\n' "$label"
    printf '    unexpected: %s\n' "$needle"
    fail=$((fail + 1))
  else
    printf '  PASS: %s\n' "$label"
    pass=$((pass + 1))
  fi
}

# Set up a minimal git repo so ralph sourcing does not fail
work_dir="$(mktemp -d "${TMPDIR:-/tmp}/ralph-deprecation.XXXXXX")"
trap 'rm -rf "$work_dir"' EXIT HUP INT TERM

cd "$work_dir"
git init -q -b main
git config user.email test@example.com
git config user.name "Test User"
printf 'seed\n' > README.md
git add README.md
git commit -q -m 'chore: seed'

# Make a fake plan so 'ralph status' has something to work with (avoids
# early-exit errors unrelated to the deprecation notice under test)
fake_bin_dir="${work_dir}/fake-bin"
mkdir -p "$fake_bin_dir"

echo "==> deprecation notice tests"

# ─── 1. fake 'ralph' binary present → notice appears on stderr ───
printf '#!/usr/bin/env sh\necho go-ralph-stub\n' > "${fake_bin_dir}/ralph"
chmod +x "${fake_bin_dir}/ralph"

stderr_out="$(PATH="${fake_bin_dir}:${PATH}" RALPH_NO_DEPRECATION="" \
  "$RALPH" status --no-color 2>&1 >/dev/null || true)"
check_contains \
  "notice appears on stderr when Go binary is on PATH" \
  "this shell entrypoint is legacy" \
  "$stderr_out"

# ─── 2. No 'ralph' binary on a stripped PATH → no notice ───
# Use a minimal PATH that contains only the bare essentials (no installed 'ralph').
_minimal_path="/usr/bin:/bin"
stderr_out="$(PATH="$_minimal_path" RALPH_NO_DEPRECATION="" \
  "$RALPH" status --no-color 2>&1 >/dev/null || true)"
check_not_contains \
  "notice absent when Go binary is not on PATH" \
  "this shell entrypoint is legacy" \
  "$stderr_out"

# ─── 3. RALPH_NO_DEPRECATION=1 → suppressed ───
stderr_out="$(PATH="${fake_bin_dir}:${PATH}" RALPH_NO_DEPRECATION=1 \
  "$RALPH" status --no-color 2>&1 >/dev/null || true)"
check_not_contains \
  "notice suppressed by RALPH_NO_DEPRECATION=1" \
  "this shell entrypoint is legacy" \
  "$stderr_out"

# ─── 4. stdout is unaffected (notice goes to stderr only) ───
stdout_out="$(PATH="${fake_bin_dir}:${PATH}" RALPH_NO_DEPRECATION="" \
  "$RALPH" status --no-color 2>/dev/null || true)"
check_not_contains \
  "stdout unaffected by notice" \
  "this shell entrypoint is legacy" \
  "$stdout_out"

printf '\nralph deprecation notice tests: %d passed, %d failed\n' "$pass" "$fail"
[ "$fail" -eq 0 ]
