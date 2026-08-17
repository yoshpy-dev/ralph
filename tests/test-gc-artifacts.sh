#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
GC_SCRIPT="${PROJECT_ROOT}/scripts/gc-artifacts.sh"

_pass=0
_fail=0
_total=0
_tmp=""

cleanup() {
  [ -n "$_tmp" ] && rm -rf "$_tmp"
}
trap cleanup EXIT HUP INT TERM

assert_eq() {
  local desc="$1" expected="$2" actual="$3"
  _total=$((_total + 1))
  if [ "$expected" = "$actual" ]; then
    _pass=$((_pass + 1))
    printf '  PASS: %s\n' "$desc"
  else
    _fail=$((_fail + 1))
    printf '  FAIL: %s\n    expected: %s\n    actual:   %s\n' "$desc" "$expected" "$actual"
  fi
}

assert_exit() {
  local desc="$1" expected="$2"
  shift 2
  _total=$((_total + 1))
  set +e
  "$@" >/dev/null 2>&1
  local actual="$?"
  set -e
  if [ "$expected" -eq "$actual" ]; then
    _pass=$((_pass + 1))
    printf '  PASS: %s\n' "$desc"
  else
    _fail=$((_fail + 1))
    printf '  FAIL: %s (expected exit %s, got %s)\n' "$desc" "$expected" "$actual"
  fi
}

assert_file_exists() {
  local desc="$1" filepath="$2"
  _total=$((_total + 1))
  if [ -f "$filepath" ]; then
    _pass=$((_pass + 1))
    printf '  PASS: %s\n' "$desc"
  else
    _fail=$((_fail + 1))
    printf '  FAIL: %s — file does not exist: %s\n' "$desc" "$filepath"
  fi
}

assert_file_absent() {
  local desc="$1" filepath="$2"
  _total=$((_total + 1))
  if [ ! -f "$filepath" ]; then
    _pass=$((_pass + 1))
    printf '  PASS: %s\n' "$desc"
  else
    _fail=$((_fail + 1))
    printf '  FAIL: %s — file still exists: %s\n' "$desc" "$filepath"
  fi
}

assert_output_contains() {
  local desc="$1" pattern="$2" output="$3"
  _total=$((_total + 1))
  if printf '%s' "$output" | grep -q "$pattern"; then
    _pass=$((_pass + 1))
    printf '  PASS: %s\n' "$desc"
  else
    _fail=$((_fail + 1))
    printf '  FAIL: %s — pattern %q not found in output:\n%s\n' "$desc" "$pattern" "$output"
  fi
}

# ─── Fixture setup ────────────────────────────────────────────────────────────

_tmp="$(mktemp -d)"

# Create a minimal git repo so the script can run
git -C "$_tmp" init --quiet
git -C "$_tmp" config user.email "test@example.com"
git -C "$_tmp" config user.name "Test"

mkdir -p "$_tmp/docs/reports" "$_tmp/docs/evidence" "$_tmp/scripts"
cp "$GC_SCRIPT" "$_tmp/scripts/gc-artifacts.sh"
chmod +x "$_tmp/scripts/gc-artifacts.sh"

# Create an initial commit so git rm works
touch "$_tmp/docs/reports/.gitkeep"
git -C "$_tmp" add .
git -C "$_tmp" commit --quiet -m 'initial'

# Helper: create a report file with a given date in its name and track it
make_old_report() {
  local name="$1"
  local filepath="$_tmp/docs/reports/$name"
  printf '# %s\n' "$name" > "$filepath"
  git -C "$_tmp" add "$filepath"
  git -C "$_tmp" commit --quiet -m "add $name"
}

make_recent_report() {
  local name="$1"
  local filepath="$_tmp/docs/reports/$name"
  printf '# %s\n' "$name" > "$filepath"
  git -C "$_tmp" add "$filepath"
  git -C "$_tmp" commit --quiet -m "add $name"
}

# ─── Test 1: dry-run deletes nothing ──────────────────────────────────────────

printf '==> gc-artifacts: dry-run mode\n'

# The "recent" fixture must stay inside the --days 30 window on every run
# date, so its date is computed relative to today (BSD date first, GNU
# fallback). A hardcoded date here silently expires and breaks the test.
recent_date="$(date -v-5d +%Y-%m-%d 2>/dev/null || date -d '5 days ago' +%Y-%m-%d)"

make_old_report "self-review-2026-05-01-old-slug.md"
make_recent_report "self-review-${recent_date}-recent-slug.md"

# Dry-run should not delete anything
(cd "$_tmp" && ./scripts/gc-artifacts.sh --days 30)

assert_file_exists "dry-run: old report still present" "$_tmp/docs/reports/self-review-2026-05-01-old-slug.md"
assert_file_exists "dry-run: recent report still present" "$_tmp/docs/reports/self-review-${recent_date}-recent-slug.md"

# ─── Test 2: --apply deletes old reports, keeps recent ────────────────────────

printf '==> gc-artifacts: --apply with date-based reports\n'

(cd "$_tmp" && ./scripts/gc-artifacts.sh --days 30 --apply)

assert_file_absent "--apply: old report deleted" "$_tmp/docs/reports/self-review-2026-05-01-old-slug.md"
assert_file_exists "--apply: recent report kept" "$_tmp/docs/reports/self-review-${recent_date}-recent-slug.md"

# ─── Test 3: README.md is never deleted ───────────────────────────────────────

printf '==> gc-artifacts: README.md is always preserved\n'

# Create a README.md with an old date that would otherwise be pruned
printf '# Reports\n' > "$_tmp/docs/reports/README.md"
git -C "$_tmp" add "$_tmp/docs/reports/README.md"
git -C "$_tmp" commit --quiet -m "add README.md"

# Also create an old-dated report to confirm GC runs
make_old_report "verify-2026-04-01-very-old.md"

(cd "$_tmp" && ./scripts/gc-artifacts.sh --days 30 --apply)

assert_file_exists "README.md always kept" "$_tmp/docs/reports/README.md"
assert_file_absent "old verify report deleted" "$_tmp/docs/reports/verify-2026-04-01-very-old.md"

# ─── Test 4: evidence keep-newest-20 logic ────────────────────────────────────

printf '==> gc-artifacts: evidence keep-newest-20\n'

# Create 25 evidence log files
for i in $(seq -w 1 25); do
  printf 'log %s\n' "$i" > "$_tmp/docs/evidence/verify-2026-07-01-12${i}00.log"
done

(cd "$_tmp" && ./scripts/gc-artifacts.sh --days 30 --apply)

# Count remaining evidence logs
remaining="$(find "$_tmp/docs/evidence" -name "verify-*.log" -type f | wc -l | tr -d ' ')"
assert_eq "evidence: exactly 20 logs remain" "20" "$remaining"

# ─── Test 5: dry-run output lists candidates ──────────────────────────────────

printf '==> gc-artifacts: dry-run output contains candidates\n'

make_old_report "cross-review-triage-2026-05-10-another.md"

dry_out="$(cd "$_tmp" && ./scripts/gc-artifacts.sh --days 30 2>&1)"
assert_output_contains "dry-run lists old report" "cross-review-triage-2026-05-10-another.md" "$dry_out"
assert_output_contains "dry-run mentions summary" "would be deleted" "$dry_out"

# ─── Test 6: exit 0 when nothing to do ───────────────────────────────────────

printf '==> gc-artifacts: exit 0 when nothing to do\n'

# Clean up old files first
(cd "$_tmp" && ./scripts/gc-artifacts.sh --days 30 --apply >/dev/null 2>&1 || true)
# Now run again — nothing should remain
assert_exit "nothing to do exits 0" 0 bash -c "cd '$_tmp' && ./scripts/gc-artifacts.sh --days 30 --apply"

# ─── Test 7: --help exits 0 ───────────────────────────────────────────────────

printf '==> gc-artifacts: --help\n'

assert_exit "--help exits 0" 0 "$GC_SCRIPT" --help

# ─── Summary ──────────────────────────────────────────────────────────────────

printf '\ngc-artifacts tests: %s passed, %s failed, %s total\n' "$_pass" "$_fail" "$_total"
[ "$_fail" -eq 0 ]
