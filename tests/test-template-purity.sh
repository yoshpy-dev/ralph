#!/usr/bin/env bash
# test-template-purity.sh — exercise scripts/check-template-purity.sh
# against the real templates/ tree and synthetic fixtures.
#
# Cases:
#   A. Real templates/ tree passes (AC-11 "green on current tree")
#   B. Fixture with an injected yoshpy-dev reference fails, path in output
#   C. The allowlist arrays are empty (no deferred leaks) and the suppression
#      itself still works against an injected fixture
#   D. Fixture with an unallowlisted occurrence of a leak pattern fails
#   E. Fixture with an injected dated docs/reports/<date>-<slug> path fails
#      via the REGEX_PATTERNS branch (positive detection) — H1 regression
#      guard: before the fix, this pattern's grep -E call errored on
#      unbalanced parentheses and was silently swallowed by "|| true", so
#      the regex branch never actually flagged anything.
#   F. Same as E, for the docs/plans/<date>-<slug> half of the alternation
#   G. Fixture with a file placed at .claude/skills/release/ (innocuous
#      content) fails via the PATH_PATTERNS branch — AR#4 (cycle 2,
#      docs/reports/cross-review-triage-overlay-scaffold-v2-p5.md)
#      regression guard: content scanning alone cannot see a maintainer-only
#      surface shipped with wholly ordinary text.
#   I. Patched-copy fixture proving an allowlist entry suppresses a
#      PATH_PATTERNS hit too (all three dimensions share is_allowlisted)

set -eu

REPO_ROOT="$(cd "$(dirname "$0")/.." && pwd)"
SCRIPT="$REPO_ROOT/scripts/check-template-purity.sh"

if [ ! -x "$SCRIPT" ]; then
  echo "FAIL: $SCRIPT not executable"
  exit 1
fi

pass=0
fail=0

OUT_FILE="$(mktemp)"

run_case() {
  local label="$1"
  local expected_exit="$2"   # 0 (clean) or 1 (leak found)
  local scan_root="$3"

  local actual
  if RALPH_PURITY_SCAN_ROOT="$scan_root" "$SCRIPT" >"$OUT_FILE" 2>&1; then
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
    sed 's/^/      /' "$OUT_FILE"
    fail=$((fail + 1))
  fi
}

run_case_expect_output() {
  # Like run_case, but also asserts the failing path string appears in
  # stdout+stderr.
  local label="$1" scan_root="$2" needle="$3"

  if RALPH_PURITY_SCAN_ROOT="$scan_root" "$SCRIPT" >"$OUT_FILE" 2>&1; then
    echo "  FAIL  $label: expected a leak to be detected, guard passed"
    fail=$((fail + 1))
    return
  fi

  if grep -qF -- "$needle" "$OUT_FILE"; then
    echo "  PASS  $label"
    pass=$((pass + 1))
  else
    echo "  FAIL  $label: expected output to mention '$needle'"
    sed 's/^/      /' "$OUT_FILE"
    fail=$((fail + 1))
  fi
}

cleanup() {
  rm -f "$OUT_FILE"
}
trap cleanup EXIT

# ── A. real templates/ tree passes ──────────────────────────────────────
# Runs with all three scan dimensions active (content fixed/regex + path),
# so this single case also pins that the PATH_PATTERNS dimension introduces
# no false positive against the actual shipped tree (formerly a duplicate
# case H — self-review C3-L6, cycle 3).
# Must use the relative default ("templates"), not an absolute path: the
# script always cd's to its own REPO_ROOT before resolving SCAN_ROOT, and
# Allowlist entries (empty today, but exact-path-scoped whenever one is
# added — see the script's own allowlist-array comment) are written as
# repo-relative paths — an absolute SCAN_ROOT would make grep print
# absolute hit paths that never match an allowlist entry, producing false
# failures the moment a future entry is added.
run_case "A. real templates/ tree (current repo state)" 0 "templates"

# ── B. fixture with an injected leak fails, path appears in output ─────
B_DIR="$(mktemp -d)"
mkdir -p "$B_DIR/base/docs"
printf 'This project was scaffolded from yoshpy-dev/ralph.\n' > "$B_DIR/base/docs/README.md"
run_case_expect_output "B. injected yoshpy-dev reference fails, path in output" \
  "$B_DIR" "base/docs/README.md"
rm -rf "$B_DIR"

# ── C. allowlist suppression mechanism works ────────────────────────────
# The allowlist is empty by design (overlay-scaffold-v2-p5 Slice 5 fixed
# every known leak instead of allowlisting around it) — so there is no
# longer a real, persistent leak in templates/ to assert against. Instead:
# (1) assert the invariant that the parallel allowlist arrays are currently
# empty (proves leaks were fixed, not deferred), and (2) prove the
# suppression mechanism still works AND is exact-path scoped by running a
# patched copy of the script, with one allowlist entry injected, against a
# fake repo root containing the allowlisted leak plus a second occurrence
# of the same pattern at a different, non-allowlisted path — the run must
# fail naming only the second path. Allowlist paths are REPO_ROOT-relative
# (see the script's own SCAN_ROOT comment), and REPO_ROOT is derived from
# the script's own path ("$(dirname "$0")/.."), so copying the script into
# a fake root's scripts/ makes that fake root's templates/ the scan target.
if grep -q '^ALLOWLIST_PATHS=()$' "$SCRIPT" && grep -q '^ALLOWLIST_PATTERNS=()$' "$SCRIPT"; then
  echo "  PASS  C1. allowlist arrays are currently empty (no known leaks deferred)"
  pass=$((pass + 1))
else
  echo "  FAIL  C1. expected ALLOWLIST_PATHS=() and ALLOWLIST_PATTERNS=() in $SCRIPT — a leak is being allowlisted instead of fixed"
  fail=$((fail + 1))
fi

C_DIR="$(mktemp -d)"
mkdir -p "$C_DIR/scripts" "$C_DIR/templates/base/docs"
sed -e 's/^ALLOWLIST_PATHS=()$/ALLOWLIST_PATHS=("templates\/base\/docs\/leak.md")/' \
    -e 's/^ALLOWLIST_PATTERNS=()$/ALLOWLIST_PATTERNS=("yoshpy-dev")/' \
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

# Same patched copy, second occurrence at a non-allowlisted path: must fail
# and name only the second path — this is what actually proves the entry is
# scoped to its exact (path, pattern) pair rather than blanket-suppressing
# the pattern (self-review C2-L1, cycle 2).
printf 'Also mentions yoshpy-dev here.\n' > "$C_DIR/templates/base/docs/second.md"
if C3_OUT="$("$C_DIR/scripts/check-template-purity.sh" 2>&1)"; then
  echo "  FAIL  C3. expected the non-allowlisted second occurrence to fail"
  fail=$((fail + 1))
else
  if echo "$C3_OUT" | grep -q 'second.md' && ! echo "$C3_OUT" | grep -q 'leak.md'; then
    echo "  PASS  C3. allowlist entry is exact-path scoped (second path fails, allowlisted path stays suppressed)"
    pass=$((pass + 1))
  else
    echo "  FAIL  C3. expected hit output to name second.md only"
    echo "$C3_OUT" | sed 's/^/      /'
    fail=$((fail + 1))
  fi
fi
rm -rf "$C_DIR"

# ── D. unallowlisted occurrence of a known-leak pattern fails ──────────
# A check-sync.sh reference (one of FIXED_PATTERNS' entries) at a path NOT
# in the allowlist — must still fail. Proves detection is unconditional per
# path (the allowlist is empty today; case C3 above proves an entry, once
# added, is scoped to its exact (path, pattern) pair and does not
# blanket-suppress the pattern everywhere).
D_DIR="$(mktemp -d)"
mkdir -p "$D_DIR/base/docs/other"
printf 'Run ./scripts/check-sync.sh to verify.\n' > "$D_DIR/base/docs/other/NOTES.md"
run_case_expect_output "D. unallowlisted check-sync.sh reference fails" \
  "$D_DIR" "base/docs/other/NOTES.md"
rm -rf "$D_DIR"

# ── E. dated docs/reports/<date>-<slug> path fires the regex branch ────
# H1 regression guard: the REGEX_PATTERNS entry alternates reports|plans,
# so it must independently detect a hit under each branch of the
# alternation. Before the fix, the "pattern|reason" combined-string split
# truncated this entry at its first `|` (the alternation's own `|`, not a
# delimiter), producing an unbalanced-parentheses regex that grep -E
# errored on — an error `check-template-purity.sh` used to silently
# swallow with `|| true`, so this whole branch never actually fired.
E_DIR="$(mktemp -d)"
mkdir -p "$E_DIR/base/docs"
printf 'See docs/reports/2026-08-19-example.md for background.\n' > "$E_DIR/base/docs/NOTES.md"
run_case_expect_output "E. dated docs/reports/<date>-<slug> path fails (regex branch)" \
  "$E_DIR" "base/docs/NOTES.md"
rm -rf "$E_DIR"

# ── F. dated docs/plans/<date>-<slug> path fires the regex branch ──────
# Same as E, for the other half of the reports|plans alternation.
F_DIR="$(mktemp -d)"
mkdir -p "$F_DIR/base/docs"
printf 'See docs/plans/active/2026-01-02-example.md for background.\n' > "$F_DIR/base/docs/NOTES.md"
run_case_expect_output "F. dated docs/plans/<date>-<slug> path fails (regex branch)" \
  "$F_DIR" "base/docs/NOTES.md"
rm -rf "$F_DIR"

# ── G. maintainer-only path with innocuous content fails (path branch) ──
# AR#4 regression guard: a file living at .claude/skills/release/ must fail
# regardless of what it says — content scanning (cases B/D above) has
# nothing to find here, only the path itself is the leak.
G_DIR="$(mktemp -d)"
mkdir -p "$G_DIR/base/.claude/skills/release"
printf '# Some skill\n\nNothing meta-repo-specific in this text at all.\n' \
  > "$G_DIR/base/.claude/skills/release/SKILL.md"
run_case_expect_output "G. .claude/skills/release/ path fails with innocuous content (path branch)" \
  "$G_DIR" "base/.claude/skills/release/SKILL.md"
rm -rf "$G_DIR"

# ── I. allowlist suppresses a PATH_PATTERNS hit too ─────────────────────
# The three scan dimensions share is_allowlisted; case C proved it for a
# content pattern, this proves it for a path pattern (self-review C3-L2,
# cycle 3). Same patched-copy technique as case C.
I_DIR="$(mktemp -d)"
mkdir -p "$I_DIR/scripts" "$I_DIR/templates/base/.claude/skills/release"
sed -e 's/^ALLOWLIST_PATHS=()$/ALLOWLIST_PATHS=("templates\/base\/.claude\/skills\/release\/SKILL.md")/' \
    -e 's/^ALLOWLIST_PATTERNS=()$/ALLOWLIST_PATTERNS=("\/.claude\/skills\/release\/")/' \
  "$SCRIPT" > "$I_DIR/scripts/check-template-purity.sh"
chmod +x "$I_DIR/scripts/check-template-purity.sh"
printf '# Some skill\n' > "$I_DIR/templates/base/.claude/skills/release/SKILL.md"
if I_OUT="$("$I_DIR/scripts/check-template-purity.sh" 2>&1)"; then
  echo "  PASS  I. allowlisted path-pattern hit passes"
  pass=$((pass + 1))
else
  echo "  FAIL  I. expected allowlisted path-pattern hit to pass"
  echo "$I_OUT" | sed 's/^/      /'
  fail=$((fail + 1))
fi
rm -rf "$I_DIR"


echo ""
echo "  PASS: $pass"
echo "  FAIL: $fail"
exit "$fail"
