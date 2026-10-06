#!/usr/bin/env bash
# insights-append.sh — Append one schema-v1 insight event to the per-task JSONL file.
#
# Interface (named flags):
#   Required: --slug --flow --phase --verdict --source
#   Optional: --run-id --cycle N|auto --critical N --high N --medium N --low N
#             --action-required N --worth-considering N --dismissed N
#             --driver X --requested-model X --effective-model X
#             --honored true|false --events-dir DIR --state-dir DIR
#
# --cycle auto resolves the standard-pipeline cycle with the same rule as
# /cross-review step 1: when <state-dir>/active-plan.json and
# <state-dir>/cycle-count.json both exist, both parse as one JSON object, both
# carry the same non-empty string plan_path (persisted-identity mode), and
# cycle-count.json's cycle is a positive integer, that cycle is used; in every
# other case (fallback mode) the cycle is 1. Missing, unreadable, or broken
# state files never fail the append. <state-dir> defaults to
# <repo root>/.harness/state/standard-pipeline (repo root from
# `git rev-parse --show-toplevel`, else the current directory).
#
# Output: appends one JSON line to <events-dir>/<UTC-date>-<slug>.jsonl
# Exit codes: 0 on success, 1 on validation failure (usage to stderr).
set -euo pipefail

# ─── Defaults ──────────────────────────────────────────────────────────────────

_slug=""
_flow=""
_phase=""
_verdict=""
_source=""
_run_id=""
_cycle=""
_critical=""
_high=""
_medium=""
_low=""
_action_required=""
_worth_considering=""
_dismissed=""
_driver=""
_requested_model=""
_effective_model=""
_honored=""
_events_dir=""
_state_dir=""

# ─── Argument parsing ──────────────────────────────────────────────────────────

usage() {
  cat >&2 <<'USAGE'
Usage: insights-append.sh [OPTIONS]

Append one schema-v1 insight event to docs/insights/events/<date>-<slug>.jsonl.

Required:
  --slug SLUG              Task slug (matches plan basename, e.g. ralph-insights)
  --flow standard|loop     Pipeline flow type
  --phase PHASE            implement|self_review|verify|test|sync_docs|cross_review|pr
  --verdict VERDICT        pass|fail|complete|action_required|n/a
  --source SOURCE          pipeline|skill|backfill

Optional:
  --run-id ID              Per-pipeline-invocation ID (default: omitted)
  --cycle N|auto           1-based outer cycle number (default: 1); auto reads
                           the standard-pipeline cycle from the state dir
                           (active-plan.json + cycle-count.json with matching
                           plan_path), else 1
  --critical N             CRITICAL finding count (default: 0)
  --high N                 HIGH finding count (default: 0)
  --medium N               MEDIUM finding count (default: 0)
  --low N                  LOW finding count (default: 0)
  --action-required N      ACTION_REQUIRED triage count (default: 0)
  --worth-considering N    WORTH_CONSIDERING triage count (default: 0)
  --dismissed N            DISMISSED triage count (default: 0)
  --driver claude|codex    Driver used (default: omitted)
  --requested-model MODEL  Model requested (default: omitted)
  --effective-model MODEL  Model actually used (default: omitted)
  --honored true|false     Whether model request was honored (default: omitted)
  --events-dir DIR         Override events directory (default: docs/insights/events)
  --state-dir DIR          Override the state dir read by --cycle auto
                           (default: <repo root>/.harness/state/standard-pipeline)
USAGE
  exit 1
}

if [ $# -eq 0 ]; then
  usage
fi

while [ $# -gt 0 ]; do
  case "$1" in
    --slug)              shift; _slug="${1:-}"            ;;
    --flow)              shift; _flow="${1:-}"            ;;
    --phase)             shift; _phase="${1:-}"           ;;
    --verdict)           shift; _verdict="${1:-}"         ;;
    --source)            shift; _source="${1:-}"          ;;
    --run-id)            shift; _run_id="${1:-}"          ;;
    --cycle)             shift; _cycle="${1:-}"           ;;
    --critical)          shift; _critical="${1:-}"        ;;
    --high)              shift; _high="${1:-}"            ;;
    --medium)            shift; _medium="${1:-}"          ;;
    --low)               shift; _low="${1:-}"             ;;
    --action-required)   shift; _action_required="${1:-}" ;;
    --worth-considering) shift; _worth_considering="${1:-}" ;;
    --dismissed)         shift; _dismissed="${1:-}"       ;;
    --driver)            shift; _driver="${1:-}"          ;;
    --requested-model)   shift; _requested_model="${1:-}" ;;
    --effective-model)   shift; _effective_model="${1:-}" ;;
    --honored)           shift; _honored="${1:-}"         ;;
    --events-dir)        shift; _events_dir="${1:-}"      ;;
    --state-dir)         shift; _state_dir="${1:-}"       ;;
    -h|--help)           usage                            ;;
    *) printf 'Unknown option: %s\n' "$1" >&2; usage     ;;
  esac
  shift
done

# ─── Validation helpers ────────────────────────────────────────────────────────

_err() { printf 'insights-append.sh: %s\n' "$*" >&2; exit 1; }

require_field() {
  _rf_name="$1"
  _rf_val="$2"
  if [ -z "$_rf_val" ]; then
    _err "Missing required flag: --${_rf_name}"
  fi
}

validate_enum() {
  _ve_name="$1"
  _ve_val="$2"
  shift 2
  for _ve_allowed; do
    [ "$_ve_val" = "$_ve_allowed" ] && return 0
  done
  _err "Invalid value for --${_ve_name}: '${_ve_val}'. Allowed: $*"
}

validate_nonneg_int() {
  _vni_name="$1"
  _vni_val="$2"
  case "$_vni_val" in
    ''|*[!0-9]*) _err "Invalid non-negative integer for --${_vni_name}: '${_vni_val}'" ;;
  esac
}

# ─── --cycle auto resolution ──────────────────────────────────────────────────

# Print the plan_path of a state file, or nothing when the file is missing,
# unreadable, not exactly one JSON object, or has no non-empty string plan_path.
state_plan_path() {
  [ -f "$1" ] || return 0
  jq -rs 'if length == 1 and (.[0] | type) == "object"
             and (.[0].plan_path | type) == "string"
          then .[0].plan_path else empty end' "$1" 2>/dev/null || true
}

# Print the cycle of cycle-count.json when it is a positive integer, else nothing.
state_cycle() {
  [ -f "$1" ] || return 0
  jq -rs 'if length == 1 and (.[0] | type) == "object"
             and (.[0].cycle | type) == "number"
             and .[0].cycle >= 1 and .[0].cycle == (.[0].cycle | floor)
          then .[0].cycle | floor | tostring else empty end' "$1" 2>/dev/null || true
}

# Print the cycle for --cycle auto (see the header comment for the rule).
resolve_auto_cycle() {
  _rac_dir="$_state_dir"
  if [ -z "$_rac_dir" ]; then
    _rac_root="$(git rev-parse --show-toplevel 2>/dev/null)" || _rac_root="$(pwd)"
    _rac_dir="${_rac_root}/.harness/state/standard-pipeline"
  fi
  _rac_active="$(state_plan_path "${_rac_dir}/active-plan.json")"
  _rac_counted="$(state_plan_path "${_rac_dir}/cycle-count.json")"
  _rac_cycle="$(state_cycle "${_rac_dir}/cycle-count.json")"
  if [ -z "$_rac_active" ] || [ "$_rac_active" != "$_rac_counted" ]; then
    _rac_cycle=""
  fi
  case "$_rac_cycle" in
    ''|*[!0-9]*) printf '1' ;;
    *)           printf '%s' "$_rac_cycle" ;;
  esac
}

# ─── Required field checks ────────────────────────────────────────────────────

require_field "slug"    "$_slug"
require_field "flow"    "$_flow"
require_field "phase"   "$_phase"
require_field "verdict" "$_verdict"
require_field "source"  "$_source"

# ─── Enum validation ──────────────────────────────────────────────────────────

validate_enum "flow"    "$_flow"    standard loop
validate_enum "phase"   "$_phase"   implement self_review verify test sync_docs cross_review pr
validate_enum "verdict" "$_verdict" pass fail complete action_required n/a
validate_enum "source"  "$_source"  pipeline skill backfill

if [ -n "$_driver" ]; then
  validate_enum "driver" "$_driver" claude codex
fi

if [ -n "$_honored" ]; then
  validate_enum "honored" "$_honored" true false
fi

# ─── Numeric validation ───────────────────────────────────────────────────────

_critical="${_critical:-0}"
_high="${_high:-0}"
_medium="${_medium:-0}"
_low="${_low:-0}"
_action_required="${_action_required:-0}"
_worth_considering="${_worth_considering:-0}"
_dismissed="${_dismissed:-0}"

validate_nonneg_int "critical"          "$_critical"
validate_nonneg_int "high"              "$_high"
validate_nonneg_int "medium"            "$_medium"
validate_nonneg_int "low"               "$_low"
validate_nonneg_int "action-required"   "$_action_required"
validate_nonneg_int "worth-considering" "$_worth_considering"
validate_nonneg_int "dismissed"         "$_dismissed"

# Default cycle to 1 when omitted (source:pipeline events are always cycle >= 1).
# source:skill events written by post-implementation skills pass --cycle auto so
# a cycle-2 run is not recorded as cycle 1.
_cycle="${_cycle:-1}"
if [ "$_cycle" = "auto" ]; then
  _cycle="$(resolve_auto_cycle)"
fi
validate_nonneg_int "cycle" "$_cycle"

# ─── Destination ─────────────────────────────────────────────────────────────

_events_dir="${_events_dir:-docs/insights/events}"
mkdir -p "$_events_dir"

_date="$(date -u '+%Y-%m-%d')"
_outfile="${_events_dir}/${_date}-${_slug}.jsonl"

# ─── Build JSON with jq -n (never string interpolation) ──────────────────────

_ts="$(date -u '+%Y-%m-%dT%H:%M:%SZ')"

# Build the base object with all required fields.
# shellcheck disable=SC2016  # $ts etc. are jq variables, expanded by jq
_jq_filter='{
  schema: 1,
  ts: $ts,
  slug: $slug,
  flow: $flow,
  phase: $phase,
  cycle: ($cycle | tonumber),
  verdict: $verdict,
  findings: {
    critical: ($critical | tonumber),
    high: ($high | tonumber),
    medium: ($medium | tonumber),
    low: ($low | tonumber)
  },
  triage: {
    action_required: ($action_required | tonumber),
    worth_considering: ($worth_considering | tonumber),
    dismissed: ($dismissed | tonumber)
  },
  source: $source
}'

# We build the line in two passes:
# Pass 1: required fields + always-present optional-numeric fields.
# Pass 2: conditionally add optional string fields (only when non-empty).

_json_line="$(jq -cn \
  --arg ts               "$_ts" \
  --arg slug             "$_slug" \
  --arg flow             "$_flow" \
  --arg phase            "$_phase" \
  --arg cycle            "$_cycle" \
  --arg verdict          "$_verdict" \
  --arg critical         "$_critical" \
  --arg high             "$_high" \
  --arg medium           "$_medium" \
  --arg low              "$_low" \
  --arg action_required  "$_action_required" \
  --arg worth_considering "$_worth_considering" \
  --arg dismissed        "$_dismissed" \
  --arg source           "$_source" \
  "$_jq_filter")"

# Add run_id when provided.
if [ -n "$_run_id" ]; then
  _json_line="$(printf '%s\n' "$_json_line" | jq -c --arg v "$_run_id" '. + {run_id: $v}')"
fi

# Add routing fields when provided.
if [ -n "$_driver" ]; then
  _json_line="$(printf '%s\n' "$_json_line" | jq -c --arg v "$_driver" '. + {driver: $v}')"
fi
if [ -n "$_requested_model" ]; then
  _json_line="$(printf '%s\n' "$_json_line" | jq -c --arg v "$_requested_model" '. + {requested_model: $v}')"
fi
if [ -n "$_effective_model" ]; then
  _json_line="$(printf '%s\n' "$_json_line" | jq -c --arg v "$_effective_model" '. + {effective_model: $v}')"
fi
if [ -n "$_honored" ]; then
  _honored_bool="$([ "$_honored" = "true" ] && printf 'true' || printf 'false')"
  _json_line="$(printf '%s\n' "$_json_line" | jq -c --argjson v "$_honored_bool" '. + {honored: $v}')"
fi

# Append to JSONL file.
printf '%s\n' "$_json_line" >> "$_outfile"
