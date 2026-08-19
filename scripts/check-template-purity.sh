#!/usr/bin/env bash
set -euo pipefail

# check-template-purity.sh — FR-10: verify templates/ (the go:embed source
# tree shipped to every `ralph init`/`ralph upgrade` downstream project)
# carries no meta-repo-specific references: this repo's GitHub org/handle,
# the maintainer's personal identifiers, absolute dev-machine paths, or
# citations of meta-repo-only tooling/CI/plan artifacts that do not exist
# in a scaffolded project.
#
# Detection has two independent dimensions, both scanned against every file
# under templates/:
#   - content (FIXED_PATTERNS / REGEX_PATTERNS below): grep across file
#     bytes.
#   - path (PATH_PATTERNS below): the repo-relative path list itself, so a
#     maintainer-only surface that ships with wholly innocuous content (a
#     file merely placed at a path that must never exist in a scaffolded
#     project, e.g. .claude/skills/release/) is still caught — content
#     scanning alone cannot see this (AR#4, cycle 2,
#     docs/reports/cross-review-triage-overlay-scaffold-v2-p5.md).
# A hit in any dimension is reported unless the ALLOWLIST_PATHS/ALLOWLIST_PATTERNS parallel arrays list
# that exact (path, pattern) pair. The allowlist arrays are empty today (every leak
# found when this guard was introduced was fixed, not deferred — see
# the allowlist arrays' own comment below); the mechanism exists for a future
# genuinely intentional occurrence, with a reason recorded at the entry.
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

# ─── Path patterns ──────────────────────────────────────────────────────
# Two parallel arrays (PATH_PATTERNS / PATH_REASONS — not "pattern|reason"
# packed strings, same rationale as REGEX_PATTERNS/REGEX_REASONS above:
# a path pattern is a literal repo-relative substring, matched with `grep
# -F` against the file-path list itself rather than file contents). Every
# entry here is a maintainer-only or meta-repo-only surface that must never
# ship at that PATH regardless of what its content says — this is the
# dimension content scanning above cannot cover (AR#4, cycle 2,
# docs/reports/cross-review-triage-overlay-scaffold-v2-p5.md). Seeded from
# scripts/check-sync.sh's own ROOT_ONLY_EXCLUSIONS list (the authoritative
# "meta-repo-only, not part of the scaffolded baseline" set), narrowed to
# the entries that are plausible to accidentally land under templates/ as
# shipped project content.
PATH_PATTERNS=(
  "/.claude/skills/release/"
  "/.github/workflows/check-template.yml"
  "/.github/workflows/release.yml"
  "/scripts/check-sync.sh"
  "/scripts/check-template-purity.sh"
)
PATH_REASONS=(
  "the /release skill is repo-maintainer-only (see .claude/skills/release/SKILL.md frontmatter: 'Manual trigger only. Repo-specific — not distributed via template.') and must never be scaffolded"
  ".github/workflows/check-template.yml is meta-repo-only (scripts/check-sync.sh ROOT_ONLY_EXCLUSIONS); a scaffolded project must never receive this workflow file"
  ".github/workflows/release.yml is meta-repo-only (scripts/check-sync.sh ROOT_ONLY_EXCLUSIONS); a scaffolded project must never receive this workflow file"
  "scripts/check-sync.sh (templates/root parity checker) is meta-repo-only per CLAUDE.md and scripts/check-sync.sh's own ROOT_ONLY_EXCLUSIONS; a scaffolded project has no templates/ tree to check against, so this script must never ship as project content"
  "scripts/check-template-purity.sh (this script) checks this repo's own nested templates/ tree; a scaffolded project has no such tree, so this script must never ship as project content"
)

# ─── Allowlist ──────────────────────────────────────────────────────────
# Parallel arrays (same shape as REGEX_PATTERNS/REGEX_REASONS above, and
# for the same reason: a "path|pattern|reason" packed string cannot carry a
# regex pattern that itself contains `|` — self-review C2-M2, cycle 2).
# ALLOWLIST_PATHS entries are repo-relative (as printed by grep from
# REPO_ROOT); ALLOWLIST_PATTERNS is the exact FIXED_PATTERNS/REGEX_PATTERNS/
# PATH_PATTERNS entry each applies to (all three dimensions share this one
# mechanism). Empty by design: every pre-existing leak found when
# this guard was introduced (Slice 4 of the overlay-scaffold-v2-p5 plan) was
# fixed in Slice 5 of the same plan rather than allowlisted. Add entries
# only for a genuinely intentional occurrence, with the reason as a trailing
# comment on the ALLOWLIST_PATHS entry.
ALLOWLIST_PATHS=()
ALLOWLIST_PATTERNS=()

is_allowlisted() {
  # Length-based loop, not "${!arr[@]}" index expansion: under bash 3.2's
  # set -u the guarded index expansion yields one spurious empty iteration
  # on an empty array, while ${#arr[@]} is a plain 0.
  local path="$1" pattern="$2" i=0 n="${#ALLOWLIST_PATHS[@]}"
  while [ "$i" -lt "$n" ]; do
    if [ "$path" = "${ALLOWLIST_PATHS[$i]}" ] && [ "$pattern" = "${ALLOWLIST_PATTERNS[$i]}" ]; then
      return 0
    fi
    i=$((i + 1))
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
  # stderr deliberately NOT merged into the captured hit stream (grep
  # warnings would otherwise be parsed as hits — self-review C2-L2, cycle
  # 2); it flows to the script's own stderr, and exit >1 still hard-fails.
  _scan_output="$(grep -rn --binary-files=without-match "$mode" -- "$pattern" "$SCAN_ROOT")"
  status=$?
  set -e
  case "$status" in
    0) return 0 ;;
    1) return 1 ;;
    *)
      echo "" >&2
      # grep's own diagnostics already went to this script's stderr
      # (stderr is deliberately not captured — self-review C2-L2/C3-L3).
      echo "FAIL: grep $mode failed (exit $status) while scanning pattern: $pattern" >&2
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

# scan_path <pattern> <reason> — the path-dimension counterpart of
# scan_fixed/scan_regex above: matches pattern (a literal substring, via
# grep -F) against ALL_PATHS (every file under $SCAN_ROOT, one per line, as
# printed by `find "$SCAN_ROOT" -type f` -- same "$SCAN_ROOT/..." repo-
# relative form the content scanners above report, so ALLOWLIST_PATHS stays
# one shape for both dimensions) instead of against file contents. Reuses
# grep_scan_or_fail's fail-hard-on-real-error contract (H1's lesson: a
# swallowed grep error must never look like "no hits").
scan_path() {
  local pattern="$1" reason="$2" line status
  if [ -z "$ALL_PATHS" ]; then
    return
  fi
  set +e
  _scan_output="$(printf '%s' "$ALL_PATHS" | grep -F -- "$pattern")"
  status=$?
  set -e
  case "$status" in
    0) : ;;
    1) return ;;
    *)
      echo "" >&2
      # diagnostics already on stderr (see grep_scan_or_fail's note).
      echo "FAIL: path pattern scan failed (exit $status) while scanning pattern: $pattern" >&2
      exit 1
      ;;
  esac
  while IFS= read -r line; do
    [ -z "$line" ] && continue
    if is_allowlisted "$line" "$pattern"; then
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

# ALL_PATHS is computed once, up front, for every PATH_PATTERNS entry to
# scan against -- `find` (not `grep -rl`) because a path-only leak has no
# content signature to search for; some entries above (a maintainer-only
# skill shipped with innocuous content) exist exactly to catch what content
# scanning cannot see. 2>/dev/null so a permission-denied subdirectory
# degrades to "fewer paths found" rather than aborting the whole guard --
# unlike grep_scan_or_fail's content scan, a partial file list here is a
# safe-direction degradation (fewer hits possible, never more), whereas a
# swallowed grep content-scan error could hide an actual leak.
ALL_PATHS="$(find "$SCAN_ROOT" -type f 2>/dev/null)"

for i in "${!PATH_PATTERNS[@]}"; do
  scan_path "${PATH_PATTERNS[$i]}" "${PATH_REASONS[$i]}"
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
  echo "  - Remove/generalize the reference in the template file (content patterns), or"
  echo "  - Delete/relocate the offending file so its path no longer matches (path patterns), or"
  echo "  - If genuinely intentional, add matching ALLOWLIST_PATHS /"
  echo "    ALLOWLIST_PATTERNS entries in this script with a reason comment."
  exit 1
fi

echo ""
echo "PASS: no meta-repo-specific references found in $SCAN_ROOT."
