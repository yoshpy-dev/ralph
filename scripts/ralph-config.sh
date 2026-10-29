#!/usr/bin/env sh
# ralph-config.sh — shared configuration for the standard-flow development
# harness's /plan and /cross-review reviewer calls.
#
# The Ralph Loop autonomous execution system (its batch orchestrator/pipeline
# scripts, the per-phase CLI driver, and the legacy shell CLI) was removed;
# this file's [loop]/[pipeline] defaults (RALPH_LOOP_*, per-phase
# RALPH_*_MODEL, RALPH_MAX_* iteration caps) went with it. The surviving
# standard-flow vars below have two live consumers:
# /cross-review (.claude/skills/cross-review/SKILL.md), which sources this
# file for the standard-flow pipeline cycle cap (RALPH_STANDARD_MAX_PIPELINE_CYCLES
# is deliberately not exported below -- only sourcing this file picks it up),
# the claude-as-reviewer model fallback (RALPH_CLAUDE_REVIEWER_MODEL), and the
# codex-as-reviewer model/effort fallbacks (RALPH_CODEX_REVIEWER_MODEL,
# RALPH_CODEX_REASONING_EFFORT); and /plan (.claude/skills/plan/SKILL.md),
# whose other-model plan advisory reads the same three reviewer variables. All
# three reviewer model/effort vars are exported, so setting them directly in
# the environment also works without sourcing this file.
#
# Priority: environment variable > default value
#
# Usage:
#   . "$(dirname "$0")/ralph-config.sh"

# ═══════════════════════════════════════════════════════════════════
# Defaults (override via environment variables)
# ═══════════════════════════════════════════════════════════════════

RALPH_STANDARD_MAX_PIPELINE_CYCLES="${RALPH_STANDARD_MAX_PIPELINE_CYCLES:-2}"

# RALPH_CLAUDE_REVIEWER_MODEL is the model used by `claude -p` when it plays
# adversarial reviewer (driver = codex, reviewer = claude) in /plan's plan
# advisory and /cross-review.
RALPH_CLAUDE_REVIEWER_MODEL="${RALPH_CLAUDE_REVIEWER_MODEL:-opus}"

# RALPH_CODEX_REVIEWER_MODEL and RALPH_CODEX_REASONING_EFFORT are the model
# and reasoning effort passed as `-m` / `-c model_reasoning_effort=` to
# `codex exec` when codex is the reviewer (driver = claude) in /plan's plan
# advisory and /cross-review. Explicit so the call does not depend on
# .codex/config.toml's model or a shell alias.
RALPH_CODEX_REVIEWER_MODEL="${RALPH_CODEX_REVIEWER_MODEL:-gpt-6-astra}"
RALPH_CODEX_REASONING_EFFORT="${RALPH_CODEX_REASONING_EFFORT:-xhigh}"

# Export so values reach any grandchild processes.
export RALPH_CLAUDE_REVIEWER_MODEL RALPH_CODEX_REVIEWER_MODEL RALPH_CODEX_REASONING_EFFORT

# ═══════════════════════════════════════════════════════════════════
# Validation helpers
# ═══════════════════════════════════════════════════════════════════

# validate_numeric <name> <value>
# Exits with error if value is not a positive integer.
validate_numeric() {
  _vn_name="$1"
  _vn_value="$2"
  case "$_vn_value" in
    ''|*[!0-9]*)
      printf 'Error: %s must be a positive integer, got: %s\n' "$_vn_name" "$_vn_value" >&2
      exit 1
      ;;
  esac
  if [ "$_vn_value" -le 0 ] 2>/dev/null; then
    printf 'Error: %s must be greater than 0, got: %s\n' "$_vn_name" "$_vn_value" >&2
    exit 1
  fi
}

# validate_all_numeric — validate all numeric config values
validate_all_numeric() {
  validate_numeric "RALPH_STANDARD_MAX_PIPELINE_CYCLES" "$RALPH_STANDARD_MAX_PIPELINE_CYCLES"
}
