#!/usr/bin/env bash
# test-ralph-orchestrator-parsers.sh — unit tests for ralph-orchestrator.sh parsing functions
# Sources the script (via source guard) to test parse_slices and parse_pr_groups in isolation.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
ORCH_SCRIPT="${PROJECT_ROOT}/scripts/ralph-orchestrator.sh"

PASS=0
FAIL=0

pass() {
  PASS=$((PASS + 1))
  printf '  PASS: %s\n' "$1"
}

fail() {
  FAIL=$((FAIL + 1))
  printf '  FAIL: %s\n' "$1"
  if [ -n "${2:-}" ]; then
    printf '    detail: %s\n' "$2"
  fi
}

assert_eq() {
  _label="$1"
  _expected="$2"
  _actual="$3"
  if [ "$_expected" = "$_actual" ]; then
    pass "$_label"
  else
    fail "$_label" "expected='${_expected}' actual='${_actual}'"
  fi
}

# ═══════════════════════════════════════════════════════════════════
# Sandbox setup
# ═══════════════════════════════════════════════════════════════════

SANDBOX="$(mktemp -d "${TMPDIR:-/tmp}/ralph-orch-parsers.XXXXXX")"
cleanup() { rm -rf "$SANDBOX"; }
trap cleanup EXIT

# Set up a minimal git repo in the sandbox
(
  cd "$SANDBOX"
  git init -q -b main
  git config user.email test@example.com
  git config user.name "Test User"
  printf 'seed\n' > README.md
  git add README.md
  git commit -q -m 'chore: seed'
  mkdir -p .harness/state/orchestrator docs/evidence
)

# Source the orchestrator script from PROJECT_ROOT so BASH_SOURCE[0] resolves
# SCRIPT_DIR to scripts/ and ralph-config.sh / ralph-common.sh are found.
# The source guard prevents main() from running.
# The top-level CLI argument block is also guarded (BASH_SOURCE[0] != 0).
# shellcheck disable=SC1090
source "$ORCH_SCRIPT"

cd "$SANDBOX"

# Helper: field extractor using the US (unit separator, 0x1F) delimiter
# After Slice 3 the records use US instead of pipe; this helper reads field N (1-based)
US=$'\x1f'
get_field() {
  _record="$1"
  _n="$2"
  # Use awk with US as field separator
  printf '%s' "$_record" | awk -F"$US" "{print \$$_n}"
}

printf '==> ralph-orchestrator.sh parser tests\n'

# ═══════════════════════════════════════════════════════════════════
# parse_slices — inline format
# ═══════════════════════════════════════════════════════════════════

printf '\n--- parse_slices: inline format ---\n'

PLAN_DIR="${SANDBOX}/plan-inline"
mkdir -p "$PLAN_DIR"

cat > "${PLAN_DIR}/slice-1-api.md" <<'MD'
# Slice 1: API

- Objective: Implement REST API endpoints
- Dependencies: none
- Affected files: internal/api.go, internal/handler.go
MD

_out="$(parse_slices "$PLAN_DIR")"

_rec1="$(printf '%s\n' "$_out" | head -1)"
_slug="$(get_field "$_rec1" 1)"
_obj="$(get_field "$_rec1" 2)"
_deps="$(get_field "$_rec1" 3)"
_files="$(get_field "$_rec1" 4)"

assert_eq "inline: slug extracted correctly" "1-api" "$_slug"
assert_eq "inline: objective extracted correctly" "Implement REST API endpoints" "$_obj"
assert_eq "inline: deps 'none' becomes empty" "" "$_deps"
assert_eq "inline: files extracted correctly" "internal/api.go, internal/handler.go" "$_files"

# ═══════════════════════════════════════════════════════════════════
# parse_slices — section-header format
# ═══════════════════════════════════════════════════════════════════

printf '\n--- parse_slices: section-header format ---\n'

PLAN_DIR2="${SANDBOX}/plan-section"
mkdir -p "$PLAN_DIR2"

cat > "${PLAN_DIR2}/slice-2-docs.md" <<'MD'
# Slice 2: Docs

## Objective

Update documentation for the new API

## Dependencies

- slice-1-api

## Affected files

- docs/api.md
- README.md
MD

_out2="$(parse_slices "$PLAN_DIR2")"

_rec2="$(printf '%s\n' "$_out2" | head -1)"
_slug2="$(get_field "$_rec2" 1)"
_obj2="$(get_field "$_rec2" 2)"
_deps2="$(get_field "$_rec2" 3)"
_files2="$(get_field "$_rec2" 4)"

assert_eq "section: slug extracted correctly" "2-docs" "$_slug2"
assert_eq "section: objective extracted correctly" "Update documentation for the new API" "$_obj2"
assert_eq "section: deps extracted correctly" "slice-1-api" "$_deps2"
assert_eq "section: files extracted correctly" "docs/api.md, README.md" "$_files2"

# ═══════════════════════════════════════════════════════════════════
# parse_slices — objective containing pipe character (Slice 3 regression)
# ═══════════════════════════════════════════════════════════════════

printf '\n--- parse_slices: pipe character in objective (regression) ---\n'

PLAN_DIR3="${SANDBOX}/plan-pipe"
mkdir -p "$PLAN_DIR3"

cat > "${PLAN_DIR3}/slice-3-pipe.md" <<'MD'
# Slice 3: Pipe

- Objective: Handle requests | responses in pipeline
- Dependencies: none
- Affected files: internal/pipe.go
MD

_out3="$(parse_slices "$PLAN_DIR3")"

_rec3="$(printf '%s\n' "$_out3" | head -1)"
_slug3="$(get_field "$_rec3" 1)"
_obj3="$(get_field "$_rec3" 2)"
_files3="$(get_field "$_rec3" 4)"

assert_eq "pipe-in-obj: slug extracted correctly" "3-pipe" "$_slug3"
assert_eq "pipe-in-obj: objective with pipe character survives intact" "Handle requests | responses in pipeline" "$_obj3"
assert_eq "pipe-in-obj: files extracted correctly despite pipe in objective" "internal/pipe.go" "$_files3"

# ═══════════════════════════════════════════════════════════════════
# parse_slices — no-match returns non-zero
# ═══════════════════════════════════════════════════════════════════

printf '\n--- parse_slices: empty plan dir ---\n'

PLAN_DIR4="${SANDBOX}/plan-empty"
mkdir -p "$PLAN_DIR4"

if parse_slices "$PLAN_DIR4" >/dev/null 2>&1; then
  fail "empty plan dir exits non-zero"
else
  pass "empty plan dir exits non-zero"
fi

# ═══════════════════════════════════════════════════════════════════
# parse_pr_groups — valid groups
# ═══════════════════════════════════════════════════════════════════

printf '\n--- parse_pr_groups: valid groups ---\n'

PLAN_DIR5="${SANDBOX}/plan-groups"
mkdir -p "$PLAN_DIR5"

cat > "${PLAN_DIR5}/_manifest.md" <<'MD'
# Manifest

## PR grouping

```toml
pr_strategy = "grouped"

[[pr_groups]]
name = "core"
slices = ["slice-1-api", "slice-2-auth"]

[[pr_groups]]
name = "docs"
slices = ["slice-3-docs"]
depends = ["core"]
```
MD

_groups="$(parse_pr_groups "$PLAN_DIR5")"

_lines="$(printf '%s\n' "$_groups" | grep -c '.'  || true)"
if [ "$_lines" -ge 2 ]; then
  pass "parse_pr_groups returns at least 2 group lines"
else
  fail "parse_pr_groups returns at least 2 group lines" "got ${_lines} lines"
fi

# Check core group name is present in output
if printf '%s\n' "$_groups" | grep -q 'core'; then
  pass "parse_pr_groups: core group name present"
else
  fail "parse_pr_groups: core group name present"
fi

# Check docs group name is present in output
if printf '%s\n' "$_groups" | grep -q 'docs'; then
  pass "parse_pr_groups: docs group name present"
else
  fail "parse_pr_groups: docs group name present"
fi

# ═══════════════════════════════════════════════════════════════════
# parse_pr_groups — missing PR groups section falls back gracefully
# ═══════════════════════════════════════════════════════════════════

printf '\n--- parse_pr_groups: missing section fallback ---\n'

PLAN_DIR6="${SANDBOX}/plan-no-groups"
mkdir -p "$PLAN_DIR6"

cat > "${PLAN_DIR6}/_manifest.md" <<'MD'
# Manifest

- Type: feat
- Related issue: #42
MD

_groups6="$(parse_pr_groups "$PLAN_DIR6")"
if [ -z "$_groups6" ]; then
  pass "parse_pr_groups: empty output when no [[pr_groups]] section"
else
  fail "parse_pr_groups: empty output when no [[pr_groups]] section" "got: ${_groups6}"
fi

# ═══════════════════════════════════════════════════════════════════
# parse_pr_groups — missing _manifest.md returns zero exit
# ═══════════════════════════════════════════════════════════════════

printf '\n--- parse_pr_groups: missing manifest ---\n'

PLAN_DIR7="${SANDBOX}/plan-no-manifest"
mkdir -p "$PLAN_DIR7"

_groups7="$(parse_pr_groups "$PLAN_DIR7")"
if [ -z "$_groups7" ]; then
  pass "parse_pr_groups: empty output when _manifest.md absent"
else
  fail "parse_pr_groups: empty output when _manifest.md absent" "got: ${_groups7}"
fi

printf '\n==> ralph-orchestrator.sh parser tests: %d passed, %d failed\n' "$PASS" "$FAIL"
[ "$FAIL" -eq 0 ]
