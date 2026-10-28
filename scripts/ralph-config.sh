#!/usr/bin/env sh
# ralph-config.sh — shared configuration for the standard-flow development
# harness's /plan and /cross-review codex calls, plus one [org] default kept
# for a lock-step test (see the [org] block below).
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
# whose Codex plan advisory reads the same two codex-side variables. All
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
# adversarial reviewer in /cross-review's reviewer-inversion path (driver =
# codex, reviewer = claude).
RALPH_CLAUDE_REVIEWER_MODEL="${RALPH_CLAUDE_REVIEWER_MODEL:-opus}"

# RALPH_CODEX_REVIEWER_MODEL and RALPH_CODEX_REASONING_EFFORT are the model
# and reasoning effort passed as `-m` / `-c model_reasoning_effort=` to
# `codex exec` by /plan (Codex plan advisory) and /cross-review (reviewer =
# codex). Explicit so the call does not depend on .codex/config.toml's model
# or a shell alias.
RALPH_CODEX_REVIEWER_MODEL="${RALPH_CODEX_REVIEWER_MODEL:-gpt-6-astra}"
RALPH_CODEX_REASONING_EFFORT="${RALPH_CODEX_REASONING_EFFORT:-xhigh}"

# ═══════════════════════════════════════════════════════════════════
# [org] default — RALPH_ORG_AGMSG_HOME only. The line stays so
# internal/config/defaults_sync_test.go can compare its default with
# config.Default() and templates/base/ralph.toml; ralph issue #276 decides
# what happens to it. The other [org] defaults live only in
# internal/config/config.go and templates/base/ralph.toml: no `ralph org`
# verb ever read shell copies of them, so this file no longer carries any.
# Unlike those, RALPH_ORG_AGMSG_HOME is read from the environment by the Go
# binary (driver.ResolveAgmsgHome), which is why it is not exported below.
# ═══════════════════════════════════════════════════════════════════

RALPH_ORG_AGMSG_HOME="${RALPH_ORG_AGMSG_HOME:-~/.agents/skills/agmsg}"

# Export so values reach any grandchild processes.
#
# RALPH_ORG_AGMSG_HOME is deliberately NOT exported here. The Go binary's
# driver.ResolveAgmsgHome treats env as the highest-precedence override of
# `[org].agmsg_home` (env > toml > default). If this default assignment
# were exported unconditionally, every `ralph org` process launched from a
# shell that merely sourced this file would see the env var "set" and
# silently ignore a user-configured `[org].agmsg_home` in ralph.toml — env
# should only win when the *user* actually set it, not because a pipeline
# script sourced defaults. The default-assignment line above is kept
# unexported so `defaults_sync_test.go` (which parses this file's text via
# regex) still sees the same default value.
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
