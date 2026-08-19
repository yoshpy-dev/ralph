#!/usr/bin/env bash
set -euo pipefail

# check-template-purity.sh — FR-10: verify templates/ (the go:embed source
# tree shipped to every `ralph init`/`ralph upgrade` downstream project)
# carries no meta-repo-specific references: this repo's GitHub org/handle,
# the maintainer's personal identifiers, absolute dev-machine paths, or
# citations of meta-repo-only tooling/CI/plan artifacts that do not exist
# in a scaffolded project.
#
# Detection is a declarative pattern list (fixed-string + regex), scanned
# against every file under templates/. A hit is reported unless it is
# listed in ALLOWLIST for that exact (path, pattern) pair. ALLOWLIST is
# empty today (every leak found when this guard was introduced was fixed,
# not deferred — see ALLOWLIST's own comment below); the mechanism exists
# for a future genuinely intentional occurrence, with a reason recorded at
# the entry.
#
# Exit code: 0 if clean (after allowlisting), 1 if any unallowlisted hit
# is found.
#
# Env:
#   RALPH_PURITY_SCAN_ROOT — directory to scan, relative to cwd
#                             (default: templates). Tests override this to
#                             point at a fixture's own templates/ tree.

REPO_ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$REPO_ROOT"

SCAN_ROOT="${RALPH_PURITY_SCAN_ROOT:-templates}"

if [ ! -d "$SCAN_ROOT" ]; then
  echo "FAIL: scan root not found: $SCAN_ROOT" >&2
  exit 1
fi

# ─── Fixed-string patterns ─────────────────────────────────────────────
# "pattern|reason" — matched with `grep -F` (literal substring).
FIXED_PATTERNS=(
  "yoshpy-dev|this repo's GitHub org/user handle must never ship to downstream projects"
  "github.com/yoshpy-dev|full repo URL form of the org/user handle above"
  "hiroki|maintainer personal name/username fragment"
  "/Users/|hardcoded developer-machine absolute path"
  "skills/release|the /release skill is repo-maintainer-only (see .claude/skills/release/SKILL.md frontmatter: 'Manual trigger only. Repo-specific — not distributed via template.') and must never be scaffolded"
  "check-sync.sh|scripts/check-sync.sh (templates/root parity checker) is meta-repo-only per CLAUDE.md and scripts/check-sync.sh's own ROOT_ONLY_EXCLUSIONS; it is not shipped under templates/base/scripts/, so any reference to it in template content is a broken instruction for downstream users"
  "check-template.yml|.github/workflows/check-template.yml is meta-repo-only (scripts/check-sync.sh ROOT_ONLY_EXCLUSIONS); not shipped under templates/base/.github/workflows/"
  "release.yml|.github/workflows/release.yml is meta-repo-only (scripts/check-sync.sh ROOT_ONLY_EXCLUSIONS); not shipped under templates/base/.github/workflows/"
  "overlay-scaffold|meta-repo development phase/plan-slug family name; template content must not cite meta-repo build history"
  "org-runtime-retire-loop|meta-repo plan-slug name; template content must not cite meta-repo build history"
)

# ─── Regex patterns ─────────────────────────────────────────────────────
# Two parallel arrays (REGEX_PATTERNS / REGEX_REASONS), matched with
# `grep -E` — NOT single "pattern|reason" strings like FIXED_PATTERNS
# above. A regex pattern is free to contain a literal `|` (alternation),
# so splitting a combined string on the first `|` truncates any regex that
# uses alternation before its own reason field even begins, silently
# corrupting the pattern instead of erroring. The pattern below is exactly
# such a case: its `|` is alternation, not a delimiter — this was the
# tech-debt row 106 lesson ("a fixed regression testing gap") recurring in
# a new shape, where the split-on-`|` idiom is only safe for a delimiter
# character that cannot itself appear in the payload.
REGEX_PATTERNS=(
  "docs/(reports|plans)/[a-zA-Z_/-]*[0-9]{4}-[0-9]{2}-[0-9]{2}"
)
REGEX_REASONS=(
  "literal dated meta-repo report/plan filename (as opposed to a <date>-<slug> placeholder) baked into shipped template content"
)

# ─── Allowlist ──────────────────────────────────────────────────────────
# "path|pattern|reason" — path is repo-relative (as printed by grep from
# REPO_ROOT), pattern is the exact FIXED_PATTERNS/REGEX_PATTERNS entry it
# applies to. Empty by design: every pre-existing leak found when this guard
# was introduced (Slice 4 of the overlay-scaffold-v2-p5 plan) was fixed in
# Slice 5 of the same plan rather than allowlisted. Add an entry here only
# for a genuinely intentional occurrence, with a reason.
ALLOWLIST=()

is_allowlisted() {
  local path="$1" pattern="$2" entry entry_path entry_pattern
  for entry in "${ALLOWLIST[@]+"${ALLOWLIST[@]}"}"; do
    entry_path="${entry%%|*}"
    entry_pattern="${entry#*|}"
    entry_pattern="${entry_pattern%%|*}"
    if [ "$path" = "$entry_path" ] && [ "$pattern" = "$entry_pattern" ]; then
      return 0
    fi
  done
  return 1
}

hits=0
hit_lines=()

# grep_scan_or_fail <-F|-E> <pattern> — runs grep in the given mode against
# $SCAN_ROOT, capturing matched lines into the global _scan_output. Returns
# 0 when there were matches (caller should process _scan_output), 1 when
# there were no matches (nothing to do, same as a plain grep miss), and
# exits the WHOLE script on any grep-level error (exit >1: bad regex,
# unreadable file, etc.) instead of the silently swallowed "|| true" this
# replaces — the exact omission that let H1's dead regex branch
# (parentheses-unbalanced `grep -E` failure) go unnoticed for so long.
# Deliberately NOT invoked via process substitution (`< <(...)`) by its
# callers below: a subshell's `exit` only terminates the subshell, not the
# script, which would silently defeat this function's entire purpose.
grep_scan_or_fail() {
  local mode="$1" pattern="$2" status
  set +e
  _scan_output="$(grep -rn --binary-files=without-match "$mode" -- "$pattern" "$SCAN_ROOT" 2>&1)"
  status=$?
  set -e
  case "$status" in
    0) return 0 ;;
    1) return 1 ;;
    *)
      echo "" >&2
      echo "FAIL: grep $mode failed (exit $status) while scanning pattern: $pattern" >&2
      echo "$_scan_output" >&2
      exit 1
      ;;
  esac
}

scan_fixed() {
  local pattern="$1" reason="$2" line path
  if ! grep_scan_or_fail -F "$pattern"; then
    return
  fi
  while IFS= read -r line; do
    [ -z "$line" ] && continue
    path="${line%%:*}"
    if is_allowlisted "$path" "$pattern"; then
      continue
    fi
    hits=$((hits + 1))
    hit_lines+=("$line  [pattern: $pattern — $reason]")
  done <<< "$_scan_output"
}

scan_regex() {
  local pattern="$1" reason="$2" line path
  if ! grep_scan_or_fail -E "$pattern"; then
    return
  fi
  while IFS= read -r line; do
    [ -z "$line" ] && continue
    path="${line%%:*}"
    if is_allowlisted "$path" "$pattern"; then
      continue
    fi
    hits=$((hits + 1))
    hit_lines+=("$line  [pattern: $pattern — $reason]")
  done <<< "$_scan_output"
}

echo "=== Checking $SCAN_ROOT for meta-repo-specific references ==="

for entry in "${FIXED_PATTERNS[@]}"; do
  pattern="${entry%%|*}"
  reason="${entry#*|}"
  scan_fixed "$pattern" "$reason"
done

for i in "${!REGEX_PATTERNS[@]}"; do
  scan_regex "${REGEX_PATTERNS[$i]}" "${REGEX_REASONS[$i]}"
done

if [ "$hits" -gt 0 ]; then
  echo ""
  echo "=== Leaks found ==="
  for hl in "${hit_lines[@]}"; do
    echo "  $hl"
  done
  echo ""
  echo "FAIL: $hits meta-repo-specific reference(s) found in $SCAN_ROOT."
  echo ""
  echo "To fix:"
  echo "  - Remove/generalize the reference in the template file, or"
  echo "  - If genuinely intentional, add a path+pattern pair to ALLOWLIST"
  echo "    in this script with a reason."
  exit 1
fi

echo ""
echo "PASS: no meta-repo-specific references found in $SCAN_ROOT."
