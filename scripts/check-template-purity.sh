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
# listed in ALLOWLIST for that exact (path, pattern) pair — the allowlist
# exists for pre-existing, tracked leaks that are out of scope to fix in
# the slice that introduced this guard (see each entry's reason) and for
# genuinely intentional occurrences.
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
# "regex|reason" — matched with `grep -E`.
REGEX_PATTERNS=(
  "docs/(reports|plans)/[a-zA-Z_/-]*[0-9]{4}-[0-9]{2}-[0-9]{2}|literal dated meta-repo report/plan filename (as opposed to a <date>-<slug> placeholder) baked into shipped template content"
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

scan_fixed() {
  local pattern="$1" reason="$2" line path
  while IFS= read -r line; do
    [ -z "$line" ] && continue
    path="${line%%:*}"
    if is_allowlisted "$path" "$pattern"; then
      continue
    fi
    hits=$((hits + 1))
    hit_lines+=("$line  [pattern: $pattern — $reason]")
  done < <(grep -rn --binary-files=without-match -F -- "$pattern" "$SCAN_ROOT" 2>/dev/null || true)
}

scan_regex() {
  local pattern="$1" reason="$2" line path
  while IFS= read -r line; do
    [ -z "$line" ] && continue
    path="${line%%:*}"
    if is_allowlisted "$path" "$pattern"; then
      continue
    fi
    hits=$((hits + 1))
    hit_lines+=("$line  [pattern: $pattern — $reason]")
  done < <(grep -rnE --binary-files=without-match -- "$pattern" "$SCAN_ROOT" 2>/dev/null || true)
}

echo "=== Checking $SCAN_ROOT for meta-repo-specific references ==="

for entry in "${FIXED_PATTERNS[@]}"; do
  pattern="${entry%%|*}"
  reason="${entry#*|}"
  scan_fixed "$pattern" "$reason"
done

for entry in "${REGEX_PATTERNS[@]}"; do
  pattern="${entry%%|*}"
  reason="${entry#*|}"
  scan_regex "$pattern" "$reason"
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
