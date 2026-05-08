#!/usr/bin/env bash
set -euo pipefail

# check-sync.sh — Verify templates/ stays in sync with root files.
#
# Detects three kinds of drift:
#   DRIFTED    — file exists in both locations but content differs
#   ROOT_ONLY  — file exists at root but not in templates (and not excluded)
#   TEMPLATE_ONLY — file exists in templates but not at root (info only)
#
# Exit code: non-zero if any DRIFTED or ROOT_ONLY issues are found.

REPO_ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$REPO_ROOT"

# ─── Exclusion list ───────────────────────────────────────────────────
# Root-only files/directories that intentionally have no template counterpart.
# Prefix match: "cmd/" excludes everything under cmd/.
# Edit this list when adding new root-only files.
ROOT_ONLY_EXCLUSIONS=(
  # Go source and build
  "cmd/"
  "internal/"
  "go.mod"
  "go.sum"
  "templates.go"
  "Makefile"
  # Build and install scripts (repo-specific)
  "scripts/bootstrap.sh"
  "scripts/build-tui.sh"
  "scripts/install.sh"
  "scripts/init-project.sh"
  "scripts/new-language-pack.sh"
  # Sync check (only meaningful in the scaffold repo, not in scaffolded projects)
  "scripts/check-sync.sh"
  # Repo-local static/test runner (scaffolded projects write their own)
  "scripts/verify.local.sh"
  # Hook smoke tests (repo-specific; not part of scaffolded baseline)
  "tests/"
  # CI workflows for this repo only
  ".github/workflows/check-template.yml"
  ".github/workflows/release.yml"
  # Runtime state and local config
  ".claude/agent-memory/"
  ".claude/settings.local.json"
  ".claude/worktrees/"
  # User-local hooks (gitignored; not distributed via template)
  ".claude/hooks/local/"
  # Repo-specific skills (not part of scaffolded baseline)
  ".claude/skills/release/"
  # Repo-specific docs (not template content)
  "docs/architecture/design-principles.md"
  "docs/architecture/repo-map.md"
  "docs/specs/"
  "docs/plans/README.md"
  "docs/plans/active/"
  "docs/plans/archive/"
  "docs/tech-debt/README.md"
  "docs/references/source-notes.md"
  "docs/research/approach-comparison.md"
  "docs/roadmap/harness-maturity-model.md"
  # Evidence and reports are runtime artifacts
  "docs/evidence/"
  "docs/reports/self-review-"
  "docs/reports/verify-"
  "docs/reports/test-"
  "docs/reports/sync-docs-"
  "docs/reports/cross-review-triage-"
  # Pre-rename historical triage reports kept for archival purposes only
  # (the skill rename in 2026-05 stopped emitting this prefix).
  "docs/reports/codex-triage-"
  "docs/reports/walkthrough-"
  # Harness runtime state
  ".harness/"
  # Misc
  "README.md"
  "LICENSE"
  ".gitmodules"
  "packs/"
)

# Files that exist in templates/base but intentionally differ from root.
# These are reported as INFO (KNOWN_DIFF), not DRIFTED.
# Add entries here when root has repo-specific extensions that should
# not be propagated to the template.
KNOWN_DIFFS=(
  # CLAUDE.md: root has repo-specific /plan details; template is generic
  "CLAUDE.md"
  # AGENTS.md: root has repo-specific repo map; template has placeholder
  "AGENTS.md"
  # verify.yml: root adds bootstrap, syntax checks, coverage, pipeline-sync
  ".github/workflows/verify.yml"
)

# ─── Helpers ──────────────────────────────────────────────────────────

is_excluded() {
  local path="$1"
  for pattern in "${ROOT_ONLY_EXCLUSIONS[@]}"; do
    case "$path" in
      "${pattern}"*) return 0 ;;
    esac
  done
  return 1
}

is_known_diff() {
  local path="$1"
  for entry in "${KNOWN_DIFFS[@]+"${KNOWN_DIFFS[@]}"}"; do
    [ "$path" = "$entry" ] && return 0
  done
  return 1
}

# ─── Counters ─────────────────────────────────────────────────────────

identical=0
drifted=0
root_only=0
template_only=0
info_count=0
errors=()

# ─── 1. Check templates/base/ against root ───────────────────────────

echo "=== Checking templates/base/ <-> root ==="
while IFS= read -r tpl_file; do
  rel="${tpl_file#templates/base/}"

  if [ ! -f "$rel" ]; then
    template_only=$((template_only + 1))
    echo "  TEMPLATE_ONLY  $rel"
    continue
  fi

  if diff -q "$tpl_file" "$rel" >/dev/null 2>&1; then
    identical=$((identical + 1))
  else
    if is_known_diff "$rel"; then
      info_count=$((info_count + 1))
      echo "  KNOWN_DIFF     $rel"
    else
      drifted=$((drifted + 1))
      echo "  DRIFTED        $rel"
      errors+=("DRIFTED: $rel (templates/base/$rel vs $rel)")
    fi
  fi
done < <(find templates/base -type f | sort)

# ─── 2. Check templates/packs/ against packs/languages/ ──────────────

echo ""
echo "=== Checking templates/packs/ <-> packs/languages/ ==="
while IFS= read -r tpl_file; do
  rel="${tpl_file#templates/packs/}"
  root_file="packs/languages/$rel"

  if [ ! -f "$root_file" ]; then
    template_only=$((template_only + 1))
    echo "  TEMPLATE_ONLY  packs/languages/$rel"
    continue
  fi

  if diff -q "$tpl_file" "$root_file" >/dev/null 2>&1; then
    identical=$((identical + 1))
  else
    drifted=$((drifted + 1))
    echo "  DRIFTED        packs/languages/$rel"
    errors+=("DRIFTED: packs/languages/$rel (templates/packs/$rel vs packs/languages/$rel)")
  fi
done < <(find templates/packs -type f | sort)

# ─── 3. Detect root-only files missing from templates/base/ ──────────

echo ""
echo "=== Checking for root-only files not in templates/base/ ==="

# Directories to scan at root for potential template candidates
SCAN_DIRS=(
  ".claude"
  ".github/workflows"
  "docs"
  "scripts"
)

# Also check top-level files that templates/base/ contains
SCAN_FILES=(
  "AGENTS.md"
  "CLAUDE.md"
  ".gitignore"
)

for scan_dir in "${SCAN_DIRS[@]}"; do
  [ -d "$scan_dir" ] || continue
  while IFS= read -r root_file; do
    if is_excluded "$root_file"; then
      continue
    fi
    tpl_file="templates/base/$root_file"
    if [ ! -f "$tpl_file" ]; then
      root_only=$((root_only + 1))
      echo "  ROOT_ONLY      $root_file"
      errors+=("ROOT_ONLY: $root_file (exists at root but missing in templates/base/)")
    fi
  done < <(find "$scan_dir" -type f | sort)
done

for root_file in "${SCAN_FILES[@]}"; do
  [ -f "$root_file" ] || continue
  if is_excluded "$root_file"; then
    continue
  fi
  tpl_file="templates/base/$root_file"
  if [ ! -f "$tpl_file" ]; then
    root_only=$((root_only + 1))
    echo "  ROOT_ONLY      $root_file"
    errors+=("ROOT_ONLY: $root_file (exists at root but missing in templates/base/)")
  fi
done

# ─── 4. Detect packs/languages/ files missing from templates/packs/ ──

echo ""
echo "=== Checking for packs/languages/ files not in templates/packs/ ==="
if [ -d "packs/languages" ]; then
  while IFS= read -r root_file; do
    rel="${root_file#packs/languages/}"
    tpl_file="templates/packs/$rel"
    if [ ! -f "$tpl_file" ]; then
      root_only=$((root_only + 1))
      echo "  ROOT_ONLY      $root_file"
      errors+=("ROOT_ONLY: $root_file (exists at root but missing in templates/packs/)")
    fi
  done < <(find packs/languages -type f | sort)
fi

# ─── Summary ──────────────────────────────────────────────────────────

echo ""
echo "=== Sync Summary ==="
echo "  IDENTICAL:      $identical"
echo "  DRIFTED:        $drifted"
echo "  ROOT_ONLY:      $root_only"
echo "  TEMPLATE_ONLY:  $template_only"
echo "  KNOWN_DIFF:     $info_count"

if [ ${#errors[@]} -gt 0 ]; then
  echo ""
  echo "=== Issues Found ==="
  for err in "${errors[@]}"; do
    echo "  - $err"
  done
  echo ""
  echo "FAIL: $((drifted + root_only)) sync issue(s) found."
  echo ""
  echo "To fix:"
  echo "  DRIFTED    — copy the root version to templates/ (or vice versa)"
  echo "  ROOT_ONLY  — add to templates/ or add to ROOT_ONLY_EXCLUSIONS in this script"
  exit 1
fi

echo ""
echo "PASS: all files in sync."
