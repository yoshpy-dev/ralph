#!/usr/bin/env sh
set -eu

ROOT_DIR="$(cd "$(dirname "$0")/.." && pwd)"

PASS=0
FAIL=0

pass() {
  PASS=$((PASS + 1))
  printf '  PASS: %s\n' "$1"
}

fail() {
  FAIL=$((FAIL + 1))
  printf '  FAIL: %s\n' "$1"
}

assert_contains() {
  _name="$1"
  _needle="$2"
  _file="$3"
  if grep -qF "$_needle" "$_file"; then
    pass "$_name"
  else
    fail "$_name"
    printf '    expected to find: %s\n' "$_needle"
  fi
}

assert_eq() {
  _name="$1"
  _want="$2"
  _got="$3"
  if [ "$_want" = "$_got" ]; then
    pass "$_name"
  else
    fail "$_name"
    printf '    want: %s\n' "$_want"
    printf '     got: %s\n' "$_got"
  fi
}

assert_not_exists() {
  _name="$1"
  _path="$2"
  if [ ! -e "$_path" ]; then
    pass "$_name"
  else
    fail "$_name"
    printf '    unexpected path exists: %s\n' "$_path"
  fi
}

printf '==> ralph orchestrator dry-run side-effect tests\n'

tmp="$(mktemp -d)"
old_pwd="$(pwd)"
trap 'cd "$old_pwd"; rm -rf "$tmp"' EXIT INT TERM

cd "$tmp"
git init -q
git config user.email "test@example.com"
git config user.name "Test User"
printf 'initial\n' > README.md
git add README.md
git commit -q -m "initial"

plan_dir="docs/plans/active/2026-05-14-dry-run-side-effects"
mkdir -p "$plan_dir"
cat > "$plan_dir/_manifest.md" <<'MANIFEST'
# Dry Run Side Effects

- Type: fix
- Related issue: #83

## Shared-file locklist
- scripts/ralph-orchestrator.sh
MANIFEST
cat > "$plan_dir/slice-1-check.md" <<'SLICE'
# Slice 1

- Objective: Check dry-run behavior
- Dependencies: none
- Affected files: scripts/ralph-orchestrator.sh
SLICE

before="$(git for-each-ref --format='%(refname:short)' refs/heads | sort)"
"$ROOT_DIR/scripts/ralph-orchestrator.sh" --plan "$plan_dir" --dry-run > dry-run.log
after="$(git for-each-ref --format='%(refname:short)' refs/heads | sort)"

assert_eq "dry-run leaves branches unchanged" "$before" "$after"
assert_not_exists "dry-run does not write orchestrator state" ".harness/state/orchestrator/orchestrator.json"
assert_not_exists "dry-run does not create evidence directory" "docs/evidence"
assert_contains "dry-run parses plan" "[DRY RUN] Plan parsed successfully" dry-run.log
assert_contains "dry-run reports integration branch" "[DRY RUN] Integration branch: fix/83/dry-run-side-effects" dry-run.log

printf '\nralph orchestrator dry-run tests: %d passed, %d failed\n' "$PASS" "$FAIL"

if [ "$FAIL" -ne 0 ]; then
  exit 1
fi
