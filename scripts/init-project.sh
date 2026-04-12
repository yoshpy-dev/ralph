#!/usr/bin/env sh
set -eu

# ============================================================================
# init-project.sh — Initialize a new project from the harness template
#
# Usage:
#   ./scripts/init-project.sh
#
# What it does:
#   1. Removes template-development artifacts (reports, evidence, plans, etc.)
#   2. Runs bootstrap (directories, permissions, git hooks)
#   3. Validates language pack configuration
#   4. Prints next steps including language pack implementation reminder
# ============================================================================

echo "=== Harness project initialization ==="
echo

# --- 1. Clean template artifacts ---
echo "--- Cleaning template artifacts ---"

# Reports: remove generated reports, keep templates/ and README.md
find docs/reports -maxdepth 1 -type f \
  ! -name README.md \
  -delete 2>/dev/null || true

# Evidence: remove generated logs and JSON, keep README.md
find docs/evidence -maxdepth 1 -type f \
  ! -name README.md \
  -delete 2>/dev/null || true

# Archived plans: remove all except .gitkeep
find docs/plans/archive -maxdepth 1 -type f \
  ! -name .gitkeep \
  -delete 2>/dev/null || true

# Active plans: remove template plans, keep .gitkeep
find docs/plans/active -maxdepth 1 -type f \
  ! -name .gitkeep \
  -delete 2>/dev/null || true

# Runtime state: clean local state (already gitignored)
if [ -d .harness/state ]; then
  rm -rf .harness/state/*
  touch .harness/state/.gitkeep
fi

# Logs: clean local logs (already gitignored)
if [ -d .harness/logs ]; then
  rm -rf .harness/logs/*
fi

# Agent memory: clean learned patterns from template development
if [ -d .claude/agent-memory ]; then
  find .claude/agent-memory -name 'MEMORY.md' -exec sh -c '
    dir="$(dirname "$1")"
    agent="$(basename "$dir")"
    printf "# %s Agent Memory\n" "$agent" > "$1"
  ' _ {} \;
fi

# Tech debt: reset entries table, keep header and column structure
if [ -f docs/tech-debt/README.md ]; then
  cat > docs/tech-debt/README.md <<'TECHDEBT'
# Tech debt

Record debt that should not disappear into chat history.

Recommended fields:
- debt item
- impact
- why it was deferred
- trigger for paying it down
- related plan or report

## Entries

| Debt item | Impact | Why deferred | Trigger to pay down | Related plan/report |
| --- | --- | --- | --- | --- |
TECHDEBT
fi

echo "[ok] Template artifacts cleaned."

# --- 2. Run bootstrap ---
echo
echo "--- Running bootstrap ---"
./scripts/bootstrap.sh

# --- 3. Language pack validation ---
echo
echo "--- Language packs ---"

# Detect which languages are present in the project
detected=""
if [ -x scripts/detect-languages.sh ]; then
  detected="$(./scripts/detect-languages.sh 2>/dev/null || true)"
fi

# List available packs
available=""
for pack_dir in packs/languages/*/; do
  pack="$(basename "$pack_dir")"
  case "$pack" in
    _template) continue ;;
  esac
  available="$available $pack"
done

if [ -n "$detected" ]; then
  echo "Detected languages: $detected"
  echo
  for lang in $detected; do
    verifier="packs/languages/$lang/verify.sh"
    if [ -x "$verifier" ]; then
      echo "  [ok] $lang — pack found at $verifier"
    else
      echo "  [!!] $lang — no pack found. Create one:"
      echo "       ./scripts/new-language-pack.sh $lang"
    fi
  done
else
  echo "No languages auto-detected in the project root."
  echo "This is expected for a fresh project."
fi

echo
echo "Available packs:$available"
echo
echo "To add a new pack:     ./scripts/new-language-pack.sh <name>"
echo "To customize a pack:   Edit packs/languages/<name>/verify.sh"
echo
echo "IMPORTANT: Each language pack's verify.sh must be implemented"
echo "before './scripts/run-verify.sh' will pass for code changes."

# --- 4. Summary ---
echo
echo "=== Initialization complete ==="
echo
echo "Next steps:"
echo "  1. Edit AGENTS.md  — update mission and repo map for your project"
echo "  2. Edit CLAUDE.md  — customize for your workflow"
echo "  3. Implement packs/languages/<name>/verify.sh for your stack"
echo "  4. Create your first plan:"
echo "       ./scripts/new-feature-plan.sh <slug>"
echo "  5. Confirm setup:"
echo "       ./scripts/run-verify.sh"
echo
