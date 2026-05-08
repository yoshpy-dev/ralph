#!/usr/bin/env sh
# ralph-config.sh — shared configuration for all Ralph pipeline scripts
#
# Source this file at the top of ralph-pipeline.sh, ralph-orchestrator.sh,
# ralph-loop.sh, and scripts/ralph to get consistent defaults.
#
# Priority: CLI argument > environment variable > default value
#
# Usage:
#   . "$(dirname "$0")/ralph-config.sh"
#   # or from scripts/ralph:
#   . "${SCRIPT_DIR}/ralph-config.sh"

# ═══════════════════════════════════════════════════════════════════
# Defaults (override via environment variables)
# ═══════════════════════════════════════════════════════════════════

RALPH_MODEL="${RALPH_MODEL:-claude-opus-4-7}"
RALPH_EFFORT="${RALPH_EFFORT:-xhigh}"
RALPH_PERMISSION_MODE="${RALPH_PERMISSION_MODE:-bypassPermissions}"
RALPH_MAX_ITERATIONS="${RALPH_MAX_ITERATIONS:-20}"
RALPH_MAX_INNER_CYCLES="${RALPH_MAX_INNER_CYCLES:-10}"
RALPH_MAX_OUTER_CYCLES="${RALPH_MAX_OUTER_CYCLES:-2}"
RALPH_MAX_REPAIR_ATTEMPTS="${RALPH_MAX_REPAIR_ATTEMPTS:-5}"
RALPH_MAX_PARALLEL="${RALPH_MAX_PARALLEL:-4}"
RALPH_SLICE_TIMEOUT="${RALPH_SLICE_TIMEOUT:-1800}"
RALPH_STANDARD_MAX_PIPELINE_CYCLES="${RALPH_STANDARD_MAX_PIPELINE_CYCLES:-2}"

# Ralph Loop driver settings (Phase 2 / issue #44).
# RALPH_LOOP_DRIVER selects which CLI ralph-pipeline.sh invokes per slice
# (claude|codex). When driver=codex, ralph-cli-driver.sh assembles
# `codex exec -s <sandbox> -c approval_policy=<policy> --output-last-message ...`.
# RALPH_CLAUDE_REVIEWER_MODEL is used when cross-review inverts the reviewer
# (driver=codex → claude -p as the adversarial reviewer).
RALPH_LOOP_DRIVER="${RALPH_LOOP_DRIVER:-claude}"
RALPH_CODEX_SANDBOX="${RALPH_CODEX_SANDBOX:-workspace-write}"
RALPH_CODEX_APPROVAL_POLICY="${RALPH_CODEX_APPROVAL_POLICY:-on-failure}"
RALPH_CLAUDE_REVIEWER_MODEL="${RALPH_CLAUDE_REVIEWER_MODEL:-claude-opus-4-7}"

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
  validate_numeric "RALPH_MAX_ITERATIONS" "$RALPH_MAX_ITERATIONS"
  validate_numeric "RALPH_MAX_INNER_CYCLES" "$RALPH_MAX_INNER_CYCLES"
  validate_numeric "RALPH_MAX_OUTER_CYCLES" "$RALPH_MAX_OUTER_CYCLES"
  validate_numeric "RALPH_MAX_REPAIR_ATTEMPTS" "$RALPH_MAX_REPAIR_ATTEMPTS"
  validate_numeric "RALPH_MAX_PARALLEL" "$RALPH_MAX_PARALLEL"
  validate_numeric "RALPH_SLICE_TIMEOUT" "$RALPH_SLICE_TIMEOUT"
  validate_numeric "RALPH_STANDARD_MAX_PIPELINE_CYCLES" "$RALPH_STANDARD_MAX_PIPELINE_CYCLES"
}

# validate_loop_driver — verify RALPH_LOOP_DRIVER is one of the supported
# values. Called early so a typo fails before we attempt to invoke the wrong
# CLI binary. Mirrors the Go-side allowlist in internal/config/config.go.
validate_loop_driver() {
  case "$RALPH_LOOP_DRIVER" in
    claude|codex) ;;
    *)
      printf 'Error: RALPH_LOOP_DRIVER must be "claude" or "codex", got: %s\n' "$RALPH_LOOP_DRIVER" >&2
      exit 1
      ;;
  esac
  case "$RALPH_CODEX_SANDBOX" in
    read-only|workspace-write|danger-full-access) ;;
    *)
      printf 'Error: RALPH_CODEX_SANDBOX must be one of read-only|workspace-write|danger-full-access, got: %s\n' "$RALPH_CODEX_SANDBOX" >&2
      exit 1
      ;;
  esac
  case "$RALPH_CODEX_APPROVAL_POLICY" in
    untrusted|on-failure|on-request|never) ;;
    *)
      printf 'Error: RALPH_CODEX_APPROVAL_POLICY must be one of untrusted|on-failure|on-request|never, got: %s\n' "$RALPH_CODEX_APPROVAL_POLICY" >&2
      exit 1
      ;;
  esac
}
