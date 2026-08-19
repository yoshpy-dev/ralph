#!/usr/bin/env bash
# test-template-purity.sh — exercise scripts/check-template-purity.sh
# against the real templates/ tree and synthetic fixtures.
#
# Cases:
#   A. Real templates/ tree passes (AC-11 "green on current tree")
#   B. Fixture with an injected yoshpy-dev reference fails, path in output
#   C. ALLOWLIST is empty (no deferred leaks) and the suppression mechanism
#      itself still works against an injected fixture
#   D. Fixture with an unallowlisted occurrence of a leak pattern fails

set -eu

REPO_ROOT="$(cd "$(dirname "$0")/.." && pwd)"
SCRIPT="$REPO_ROOT/scripts/check-template-purity.sh"

if [ ! -x "$SCRIPT" ]; then
  echo "FAIL: $SCRIPT not executable"
  exit 1
fi

pass=0
fail=0

run_case() {
  local label="$1"
  local expected_exit="$2"   # 0 (clean) or 1 (leak found)
  local scan_root="$3"

  local actual
  if RALPH_PURITY_SCAN_ROOT="$scan_root" "$SCRIPT" >/tmp/test-template-purity-out.$$ 2>&1; then
    actual=0
  else
    actual=1
  fi

  if [ "$actual" -eq "$expected_exit" ]; then
    echo "  PASS  $label (exit $actual)"
    pass=$((pass + 1))
  else
    echo "  FAIL  $label: expected exit $expected_exit, got $actual"
    echo "    output:"
    sed 's/^/      /' /tmp/test-template-purity-out.$$
    fail=$((fail + 1))
  fi
}

run_case_expect_output() {
  # Like run_case, but also asserts the failing path string appears in
  # stdout+stderr.
  local label="$1" scan_root="$2" needle="$3"

  if RALPH_PURITY_SCAN_ROOT="$scan_root" "$SCRIPT" >/tmp/test-template-purity-out.$$ 2>&1; then
    echo "  FAIL  $label: expected a leak to be detected, guard passed"
    fail=$((fail + 1))
    return
  fi

  if grep -qF -- "$needle" /tmp/test-template-purity-out.$$; then
    echo "  PASS  $label"
    pass=$((pass + 1))
  else
    echo "  FAIL  $label: expected output to mention '$needle'"
    sed 's/^/      /' /tmp/test-template-purity-out.$$
    fail=$((fail + 1))
  fi
}

cleanup() {
  rm -f /tmp/test-template-purity-out.$$
}
trap cleanup EXIT

# ── A. real templates/ tree passes ──────────────────────────────────────
# Must use the relative default ("templates"), not an absolute path: the
# script always cd's to its own REPO_ROOT before resolving SCAN_ROOT, and
# ALLOWLIST entries are written as repo-relative paths (e.g.
# "templates/base/scripts/xreview-helpers.sh") — an absolute SCAN_ROOT
# would make grep print absolute hit paths that never match the allowlist,
# producing false failures on the known, allowlisted leaks.
run_case "A. real templates/ tree (current repo state)" 0 "templates"

# ── B. fixture with an injected leak fails, path appears in output ─────
B_DIR="$(mktemp -d)"
mkdir -p "$B_DIR/base/docs"
printf 'This project was scaffolded from yoshpy-dev/ralph.\n' > "$B_DIR/base/docs/README.md"
run_case_expect_output "B. injected yoshpy-dev reference fails, path in output" \
  "$B_DIR" "base/docs/README.md"
rm -rf "$B_DIR"

# ── C. allowlist suppression mechanism works ────────────────────────────
# ALLOWLIST is empty by design (overlay-scaffold-v2-p5 Slice 5 fixed every
# known leak instead of allowlisting around it) — so there is no longer a
# real, persistent leak in templates/ to assert against. Instead: (1) assert
# the invariant that ALLOWLIST is currently empty (proves leaks were fixed,
# not deferred), and (2) prove the suppression mechanism itself still works
# by running a patched copy of the script, with one ALLOWLIST entry
# injected, against a fake repo root containing a matching fixture leak.
# ALLOWLIST paths are REPO_ROOT-relative (see the script's own SCAN_ROOT
# comment), and REPO_ROOT is derived from the script's own path
# ("$(dirname "$0")/.."), so copying the script into a fake root's
# scripts/ makes that fake root's templates/ the scan target.
if grep -q '^ALLOWLIST=()$' "$SCRIPT"; then
  echo "  PASS  C1. ALLOWLIST is currently empty (no known leaks deferred)"
  pass=$((pass + 1))
else
  echo "  FAIL  C1. expected ALLOWLIST=() in $SCRIPT — a leak is being allowlisted instead of fixed"
  fail=$((fail + 1))
fi

C_DIR="$(mktemp -d)"
mkdir -p "$C_DIR/scripts" "$C_DIR/templates/base/docs"
sed 's/^ALLOWLIST=()$/ALLOWLIST=("templates\/base\/docs\/leak.md|yoshpy-dev|test fixture: injected suppression")/' \
  "$SCRIPT" > "$C_DIR/scripts/check-template-purity.sh"
chmod +x "$C_DIR/scripts/check-template-purity.sh"
printf 'This project was scaffolded from yoshpy-dev/ralph.\n' > "$C_DIR/templates/base/docs/leak.md"

if C_OUT="$("$C_DIR/scripts/check-template-purity.sh" 2>&1)"; then
  echo "  PASS  C2. allowlisted occurrence passes with the leak still present"
  pass=$((pass + 1))
else
  echo "  FAIL  C2. expected allowlisted occurrence to pass"
  echo "$C_OUT" | sed 's/^/      /'
  fail=$((fail + 1))
fi
rm -rf "$C_DIR"

# ── D. unallowlisted occurrence of a known-leak pattern fails ──────────
# Same pattern as the real allowlisted leak (check-sync.sh), but at a path
# NOT in ALLOWLIST — must still fail, proving the allowlist is exact-path
# scoped and does not blanket-suppress the pattern everywhere.
D_DIR="$(mktemp -d)"
mkdir -p "$D_DIR/base/docs/other"
printf 'Run ./scripts/check-sync.sh to verify.\n' > "$D_DIR/base/docs/other/NOTES.md"
run_case_expect_output "D. unallowlisted check-sync.sh reference fails" \
  "$D_DIR" "base/docs/other/NOTES.md"
rm -rf "$D_DIR"

echo ""
echo "  PASS: $pass"
echo "  FAIL: $fail"
exit "$fail"
