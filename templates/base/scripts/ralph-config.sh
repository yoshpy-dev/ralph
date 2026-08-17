#!/usr/bin/env sh
# ralph-config.sh — shared configuration for the standard-flow development
# harness's cross-review gate and the [org] envelope lock-step surfaces.
#
# The Ralph Loop autonomous execution system (its batch orchestrator/pipeline
# scripts, the per-phase CLI driver, and the legacy shell CLI) was removed;
# this file's [loop]/[pipeline] defaults (RALPH_LOOP_*, per-phase
# RALPH_*_MODEL, RALPH_MAX_* iteration caps) went with it. The two survivors
# below still have a live consumer:
# /cross-review (.claude/skills/cross-review/SKILL.md), which sources this
# file for the standard-flow pipeline cycle cap (RALPH_STANDARD_MAX_PIPELINE_CYCLES
# is deliberately not exported below -- only sourcing this file picks it up)
# and reads the claude-as-reviewer model fallback either way (RALPH_CLAUDE_REVIEWER_MODEL
# is exported, so setting it directly in the environment also works).
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

# ═══════════════════════════════════════════════════════════════════
# [org] envelope defaults — mirror internal/config/config.go OrgConfig
# and templates/base/ralph.toml [org]. Must change in lock-step (see
# .claude/rules/ralph/model-routing.md and defaults_sync_test.go). Except for
# RALPH_ORG_AGMSG_HOME below (which the Go binary's
# driver.ResolveAgmsgHome DOES read directly, as the runtime override for
# [org].agmsg_home), no `ralph org` verb reads these shell vars — the Go
# config (internal/config) is the runtime source; the rest exist only so
# the three surfaces agree.
# ═══════════════════════════════════════════════════════════════════

RALPH_ORG_DRIVER_POOL="${RALPH_ORG_DRIVER_POOL:-claude,codex}"
RALPH_ORG_MODEL_POOL="${RALPH_ORG_MODEL_POOL:-claude:opus,claude:sonnet,claude:haiku}"
RALPH_ORG_MAX_SEATS="${RALPH_ORG_MAX_SEATS:-5}"
RALPH_ORG_SEAT_BUDGET_MINUTES="${RALPH_ORG_SEAT_BUDGET_MINUTES:-30}"
RALPH_ORG_TOTAL_BUDGET_MINUTES="${RALPH_ORG_TOTAL_BUDGET_MINUTES:-120}"
RALPH_ORG_MAX_FIX_ROUNDS="${RALPH_ORG_MAX_FIX_ROUNDS:-2}"
RALPH_ORG_DEADMAN_MINUTES="${RALPH_ORG_DEADMAN_MINUTES:-10}"
RALPH_ORG_AGMSG_HOME="${RALPH_ORG_AGMSG_HOME:-~/.agents/skills/agmsg}"
# RALPH_ORG_PERMISSION_DEFAULT mirrors [org.permissions].default. Kept
# unexported for the same reason as RALPH_ORG_AGMSG_HOME above: nothing in
# this file's export block should risk shadowing a value the Go config
# (internal/config) is the sole runtime source for. This var exists only so
# the three lock-step surfaces agree on the default string; no `ralph org`
# verb reads it from the environment.
RALPH_ORG_PERMISSION_DEFAULT="${RALPH_ORG_PERMISSION_DEFAULT:-autonomous}"
# RALPH_ORG_PERMISSIONS_CODEX_VERIFIED mirrors
# [org.permissions].codex_verified (PR④ AC-8): false keeps codex seats
# fail-closed to guarded until an operator has live-verified their installed
# codex CLI's interactive sandbox/approval flags. Kept unexported for the
# same reason as RALPH_ORG_PERMISSION_DEFAULT above: nothing in this file's
# export block should risk shadowing a value the Go config
# (internal/config) is the sole runtime source for. This var exists only so
# the three lock-step surfaces agree on the default value; no `ralph org`
# verb reads it from the environment.
RALPH_ORG_PERMISSIONS_CODEX_VERIFIED="${RALPH_ORG_PERMISSIONS_CODEX_VERIFIED:-false}"
# RALPH_ORG_WATCHDOG_* mirror [org.watchdog] (interval_seconds/stall_minutes/
# watcher_enabled/watcher_model). Kept unexported for the same reason as
# RALPH_ORG_AGMSG_HOME above: nothing in this file's export block should risk
# shadowing a value the Go config (internal/config) is the sole runtime
# source for. These vars exist only so the three lock-step surfaces agree on
# the default values; no `ralph org` verb reads them from the environment.
RALPH_ORG_WATCHDOG_INTERVAL_SECONDS="${RALPH_ORG_WATCHDOG_INTERVAL_SECONDS:-30}"
RALPH_ORG_WATCHDOG_STALL_MINUTES="${RALPH_ORG_WATCHDOG_STALL_MINUTES:-15}"
RALPH_ORG_WATCHDOG_WATCHER_ENABLED="${RALPH_ORG_WATCHDOG_WATCHER_ENABLED:-true}"
RALPH_ORG_WATCHDOG_WATCHER_MODEL="${RALPH_ORG_WATCHDOG_WATCHER_MODEL:-haiku}"

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
export RALPH_CLAUDE_REVIEWER_MODEL
export RALPH_ORG_DRIVER_POOL RALPH_ORG_MODEL_POOL RALPH_ORG_MAX_SEATS
export RALPH_ORG_SEAT_BUDGET_MINUTES RALPH_ORG_TOTAL_BUDGET_MINUTES
export RALPH_ORG_MAX_FIX_ROUNDS RALPH_ORG_DEADMAN_MINUTES

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
