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

RALPH_MODEL="${RALPH_MODEL:-opus}"
RALPH_EFFORT="${RALPH_EFFORT:-high}"
RALPH_PERMISSION_MODE="${RALPH_PERMISSION_MODE:-bypassPermissions}"
RALPH_MAX_ITERATIONS="${RALPH_MAX_ITERATIONS:-20}"
RALPH_MAX_INNER_CYCLES="${RALPH_MAX_INNER_CYCLES:-10}"
RALPH_MAX_OUTER_CYCLES="${RALPH_MAX_OUTER_CYCLES:-2}"
RALPH_MAX_REPAIR_ATTEMPTS="${RALPH_MAX_REPAIR_ATTEMPTS:-5}"
RALPH_MAX_PARALLEL="${RALPH_MAX_PARALLEL:-4}"
RALPH_SLICE_TIMEOUT="${RALPH_SLICE_TIMEOUT:-1800}"
RALPH_STANDARD_MAX_PIPELINE_CYCLES="${RALPH_STANDARD_MAX_PIPELINE_CYCLES:-2}"

# Ralph Loop driver settings (Phase 2 / issue #44).
# RALPH_LOOP_DRIVER selects which driver ralph-pipeline.sh invokes per slice
# (claude|codex). When driver=codex, ralph-cli-driver.sh assembles
# `codex exec -s <sandbox> -c approval_policy=<policy> --output-last-message ...`.
# RALPH_CLAUDE_REVIEWER_MODEL is used when cross-review inverts the reviewer
# (driver=codex → claude -p as the adversarial reviewer).
RALPH_LOOP_DRIVER="${RALPH_LOOP_DRIVER:-claude}"
RALPH_CODEX_SANDBOX="${RALPH_CODEX_SANDBOX:-workspace-write}"
RALPH_CODEX_APPROVAL_POLICY="${RALPH_CODEX_APPROVAL_POLICY:-on-failure}"
RALPH_CLAUDE_REVIEWER_MODEL="${RALPH_CLAUDE_REVIEWER_MODEL:-opus}"

# ═══════════════════════════════════════════════════════════════════
# Per-phase model routing (plan-big-execute-small)
#
# Routing rationale: .claude/rules/model-routing.md
# Precedence: RALPH_FORCE_MODEL > RALPH_<PHASE>_MODEL > phase default.
# RALPH_MODEL remains the fallback for any unrouted agent turn.
# ═══════════════════════════════════════════════════════════════════

# Single-knob override: when non-empty, all phase models resolve to this value.
# Use for rollback (RALPH_FORCE_MODEL=opus) or "run everything on X".
RALPH_FORCE_MODEL="${RALPH_FORCE_MODEL:-}"

# Per-phase defaults (implement/upgrade seat is sonnet; judgment seat is opus).
RALPH_IMPLEMENT_MODEL="${RALPH_IMPLEMENT_MODEL:-sonnet}"
RALPH_SELF_REVIEW_MODEL="${RALPH_SELF_REVIEW_MODEL:-opus}"
RALPH_VERIFY_MODEL="${RALPH_VERIFY_MODEL:-sonnet}"
RALPH_TEST_MODEL="${RALPH_TEST_MODEL:-sonnet}"
RALPH_SYNC_DOCS_MODEL="${RALPH_SYNC_DOCS_MODEL:-sonnet}"
RALPH_PR_MODEL="${RALPH_PR_MODEL:-sonnet}"
RALPH_PROBE_MODEL="${RALPH_PROBE_MODEL:-haiku}"

# Escalation: when the Outer Loop enters a fix-and-revalidate cycle (cycle >= 2),
# the implement phase runs on this model instead of RALPH_IMPLEMENT_MODEL.
RALPH_ESCALATION_MODEL="${RALPH_ESCALATION_MODEL:-opus}"

# ═══════════════════════════════════════════════════════════════════
# [org] envelope defaults — mirror internal/config/config.go OrgConfig
# and templates/base/ralph.toml [org]. Must change in lock-step (see
# .claude/rules/model-routing.md and defaults_sync_test.go). Except for
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

# Export so values reach grandchild processes (e.g. ralph-pipeline.sh
# spawned from ralph-orchestrator.sh, or codex/claude invoked via xargs).
# Without `export`, shell-local defaults set above would be invisible to
# children that did not source this file directly.
export RALPH_LOOP_DRIVER RALPH_CODEX_SANDBOX RALPH_CODEX_APPROVAL_POLICY RALPH_CLAUDE_REVIEWER_MODEL
export RALPH_FORCE_MODEL RALPH_IMPLEMENT_MODEL RALPH_SELF_REVIEW_MODEL
export RALPH_VERIFY_MODEL RALPH_TEST_MODEL RALPH_SYNC_DOCS_MODEL RALPH_PR_MODEL
export RALPH_PROBE_MODEL RALPH_ESCALATION_MODEL
export RALPH_ORG_DRIVER_POOL RALPH_ORG_MODEL_POOL RALPH_ORG_MAX_SEATS
export RALPH_ORG_SEAT_BUDGET_MINUTES RALPH_ORG_TOTAL_BUDGET_MINUTES
export RALPH_ORG_MAX_FIX_ROUNDS RALPH_ORG_DEADMAN_MINUTES RALPH_ORG_AGMSG_HOME

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
