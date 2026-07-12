#!/usr/bin/env bash
set -euo pipefail

# Ralph Orchestrator — multi-worktree parallel pipeline execution
#
# Reads a Ralph Loop plan (with vertical slice definitions), creates a Git
# worktree for each independent slice, and runs ralph-pipeline.sh in each
# worktree concurrently. Slices with dependencies wait for prerequisites.
#
# Requires: git, jq, ralph-pipeline.sh

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
. "${SCRIPT_DIR}/ralph-config.sh"
. "${SCRIPT_DIR}/ralph-common.sh"

WORKTREE_BASE=".claude/worktrees"
ORCH_STATE=".harness/state/orchestrator"
EVIDENCE_DIR="docs/evidence"
PLAN_FILE=""
MAX_PARALLEL="$RALPH_MAX_PARALLEL"
MAX_ITERATIONS="$RALPH_MAX_ITERATIONS"
DRY_RUN=0
PREFLIGHT_ONLY=0
RESUME=0
UNIFIED_PR=0
PR_STRATEGY_OVERRIDE=""
PR_STRATEGY=""

usage() {
  cat <<'USAGE'
Usage: ralph-orchestrator.sh --plan <plan-directory> [OPTIONS]

Multi-worktree parallel pipeline orchestrator for Ralph Loop.

Options:
  --plan <directory>     Path to a plan directory with _manifest.md + slice-*.md files (required)
  --max-parallel N       Max concurrent worktree pipelines (default: 4)
  --max-iterations N     Per-slice iteration cap passed to ralph-pipeline.sh (default: 20)
  --preflight            Parse the plan and run the per-slice pipeline preflight once
  --resume               Resume existing per-slice pipeline checkpoints
  --pr-strategy <mode>   PR strategy: grouped (default), stacked, or unified
  --unified-pr           Compatibility alias for --pr-strategy unified
  --dry-run              Parse plan and show what would run without executing
  -h, --help             Show this help
USAGE
  exit 1
}

while [ $# -gt 0 ]; do
  case "$1" in
    --plan)            shift; PLAN_FILE="${1:?requires a file path}" ;;
    --max-parallel)    shift; MAX_PARALLEL="${1:?requires a number}"; validate_numeric "--max-parallel" "$MAX_PARALLEL" ;;
    --max-iterations)  shift; MAX_ITERATIONS="${1:?requires a number}"; validate_numeric "--max-iterations" "$MAX_ITERATIONS" ;;
    --preflight)       PREFLIGHT_ONLY=1 ;;
    --resume)          RESUME=1 ;;
    --pr-strategy)
      shift
      PR_STRATEGY_OVERRIDE="${1:?requires a strategy}"
      case "$PR_STRATEGY_OVERRIDE" in
        grouped|stacked|unified) ;;
        *) echo "Error: --pr-strategy must be one of grouped, stacked, unified"; exit 1 ;;
      esac
      ;;
    --unified-pr)      UNIFIED_PR=1; PR_STRATEGY_OVERRIDE="unified" ;;
    --dry-run)         DRY_RUN=1 ;;
    -h|--help)         usage ;;
    *)                 echo "Unknown option: $1"; usage ;;
  esac
  shift
done

validate_all_numeric
validate_loop_driver

if [ -z "$PLAN_FILE" ]; then
  echo "Error: --plan <directory> is required"
  usage
fi
if [ ! -d "$PLAN_FILE" ]; then
  echo "Error: --plan must be a directory-based plan (with _manifest.md + slice-*.md files)"
  echo "  Got: ${PLAN_FILE}"
  echo "  Create one with: ./scripts/new-ralph-plan.sh --type <type> <slug> [issue] [slice-count]"
  exit 1
fi

# ═══════════════════════════════════════════════════════════════════
# Utility functions
# ═══════════════════════════════════════════════════════════════════

# ts, ts_file, log, log_error are provided by ralph-common.sh (sourced above).
log_warn() { printf '[%s] WARNING: %s\n' "$(ts)" "$*" >&2; }

# ═══════════════════════════════════════════════════════════════════
# Signal handling and cleanup
# ═══════════════════════════════════════════════════════════════════

_INTERRUPTED=0

# Signal handler — sets flag and exits to trigger EXIT trap
_on_signal() {
  _INTERRUPTED=1
  exit 1
}

# Kill active child processes and update orchestrator status.
# Uses .pid files as the authoritative source of running slice processes.
# (The in-memory _CHILD_PIDS list is NOT used here because completed PIDs
# are never removed from it, risking PID reuse kills on long-running sessions.)
cleanup_on_exit() {
  # Kill PIDs recorded in state files (.pid files are deleted when slices complete)
  for _pf in "${ORCH_STATE}"/slice-*.pid; do
    [ -f "$_pf" ] || continue
    _spid="$(cat "$_pf" 2>/dev/null || true)"
    [ -z "$_spid" ] && continue
    if kill -0 "$_spid" 2>/dev/null; then
      kill "$_spid" 2>/dev/null || true
      log "Killed slice process: $_spid"
    fi
    rm -f "$_pf"
  done
  # Update orchestrator.json status ONLY on genuine signal interrupts.
  # Normal non-zero exits (e.g., partial failures) preserve their own status.
  if [ "$_INTERRUPTED" -eq 1 ] && [ -f "${ORCH_STATE}/orchestrator.json" ]; then
    if command -v jq >/dev/null 2>&1; then
      jq --arg s "interrupted" '.status = $s | .ended = "'"$(ts)"'"' \
        "${ORCH_STATE}/orchestrator.json" > "${ORCH_STATE}/orchestrator.tmp.$$.json" 2>/dev/null \
        && mv "${ORCH_STATE}/orchestrator.tmp.$$.json" "${ORCH_STATE}/orchestrator.json" 2>/dev/null || true
    fi
  fi
}

trap _on_signal INT TERM
trap cleanup_on_exit EXIT

# ═══════════════════════════════════════════════════════════════════
# Plan parsing — extract slices from markdown
# ═══════════════════════════════════════════════════════════════════

# Parse slice definitions from a directory-based plan.
# Input: path to a plan directory containing _manifest.md + slice-*.md files
# Output: one line per slice: slug|objective|dependencies|affected_files|plan_file_path
#
# Each slice file supports two field formats:
#   1. Inline fields: "- Objective: ...", "- Dependencies: ...", "- Affected files: ..."
#   2. Section headers: "## Objective", "## Dependencies", "## Affected files"
parse_slices() {
  _plan_dir="$1"
  _found=0

  for _slice_file in "$_plan_dir"/slice-*.md; do
    [ -f "$_slice_file" ] || continue
    _found=1

    # Extract slug from filename: slice-1-auth-api.md -> 1-auth-api
    # Keeps the number prefix to guarantee uniqueness across slices
    _basename="$(basename "$_slice_file" .md)"
    _slug="$(echo "$_basename" | sed 's/^slice-//')"

    _objective=""
    _deps=""
    _files=""
    _section=""

    while IFS= read -r line; do
      case "$line" in
        # --- Inline format ---
        "- Objective: "*)
          _objective="$(echo "$line" | sed 's/^- Objective: *//')"
          _section=""
          ;;
        "- Dependencies: "*)
          _raw_deps="$(echo "$line" | sed 's/^- Dependencies: *//')"
          case "$_raw_deps" in
            none|None|"") _deps="" ;;
            *) _deps="$_raw_deps" ;;
          esac
          _section=""
          ;;
        "- Affected files: "*)
          _files="$(echo "$line" | sed 's/^- Affected files: *//' | tr -d '[]')"
          _section=""
          ;;
        # --- Section header format ---
        "## Objective"*)   _section="objective" ;;
        "## Dependencies"*)  _section="deps" ;;
        "## Affected files"*) _section="files" ;;
        "## "*)            _section="" ;;
        # --- Section body ---
        *)
          if [ -n "$_section" ] && [ -n "$line" ]; then
            case "$_section" in
              objective)
                if [ -z "$_objective" ]; then
                  _objective="$line"
                fi
                ;;
              deps)
                _raw_dep="$(echo "$line" | sed 's/^- *//' | tr -d '`')"
                case "$_raw_dep" in
                  none|None) ;;
                  *)
                    if [ -n "$_raw_dep" ]; then
                      _deps="${_deps:+${_deps}, }${_raw_dep}"
                    fi
                    ;;
                esac
                ;;
              files)
                _raw_file="$(echo "$line" | sed 's/^- *//' | tr -d '`')"
                if [ -n "$_raw_file" ]; then
                  _files="${_files:+${_files}, }${_raw_file}"
                fi
                ;;
            esac
          fi
          ;;
      esac
    done < "$_slice_file"

    printf '%s|%s|%s|%s|%s\n' "$_slug" "$_objective" "$_deps" "$_files" "$_slice_file"
  done

  if [ "$_found" -eq 0 ]; then
    return 1
  fi
}

# Parse shared-file locklist from the plan directory (_manifest.md)
parse_locklist() {
  _plan_dir="$1"
  _manifest="${_plan_dir}/_manifest.md"
  if [ -f "$_manifest" ]; then
    parse_locklist_from_file "$_manifest"
  fi
}

parse_locklist_from_file() {
  _file="$1"
  _in_locklist=0

  while IFS= read -r line; do
    case "$line" in
      "### Shared-file locklist"*|"## Shared-file locklist"*)
        _in_locklist=1
        continue
        ;;
    esac

    if [ "$_in_locklist" -eq 1 ]; then
      case "$line" in
        "### "*|"## "*)
          # Next section
          break
          ;;
        "- "*)
          # Extract file path (remove leading "- " and surrounding backticks)
          echo "$line" | sed 's/^- *//' | tr -d '`'
          ;;
      esac
    fi
  done < "$_file"
}

# Parse PR strategy from _manifest.md. Supports either:
#   pr_strategy = "grouped"
#   - PR strategy: grouped
parse_pr_strategy() {
  _plan_dir="$1"
  _manifest="${_plan_dir}/_manifest.md"
  [ -f "$_manifest" ] || return 0

  awk '
    {
      line = $0
      sub(/[[:space:]]*#.*/, "", line)
      sub(/^[[:space:]]*[-*]?[[:space:]]*/, "", line)
      key = tolower(line)
      if (key ~ /^pr[_ -]?strategy[[:space:]]*[:=]/) {
        sub(/^[^:=]*[:=][[:space:]]*/, "", line)
        gsub(/["'\''`]/, "", line)
        sub(/[[:space:]].*$/, "", line)
        print tolower(line)
        exit
      }
    }
  ' "$_manifest"
}

# Parse a scalar field from the optional [pr_strategy_decision] manifest block.
parse_pr_strategy_decision_field() {
  _plan_dir="$1"
  _field="$2"
  _manifest="${_plan_dir}/_manifest.md"
  [ -f "$_manifest" ] || return 0

  awk -v want="$_field" '
    function trim(s) {
      sub(/^[[:space:]]+/, "", s)
      sub(/[[:space:]]+$/, "", s)
      return s
    }
    function clean_value(v) {
      sub(/[[:space:]]*#.*/, "", v)
      v = trim(v)
      gsub(/^["`]|["`]$/, "", v)
      return v
    }
    /^[[:space:]]*\[pr_strategy_decision\][[:space:]]*$/ {
      in_decision = 1
      next
    }
    /^[[:space:]]*\[/ {
      if (in_decision) {
        exit
      }
    }
    in_decision {
      line = $0
      sub(/[[:space:]]*#.*/, "", line)
      key = line
      sub(/[[:space:]]*=.*/, "", key)
      key = trim(key)
      if (key == want) {
        sub(/^[^=]*=[[:space:]]*/, "", line)
        print clean_value(line)
        exit
      }
    }
  ' "$_manifest"
}

count_pr_strategy_group_rationales() {
  _plan_dir="$1"
  _manifest="${_plan_dir}/_manifest.md"
  [ -f "$_manifest" ] || { printf '0\n'; return 0; }
  grep -c '^[[:space:]]*\[\[pr_strategy_decision\.group_rationale\]\]' "$_manifest" 2>/dev/null || true
}

has_stacked_dependency_rationale() {
  _plan_dir="$1"
  _manifest="${_plan_dir}/_manifest.md"
  [ -f "$_manifest" ] || return 1

  awk '
    function trim(s) {
      sub(/^[[:space:]]+/, "", s)
      sub(/[[:space:]]+$/, "", s)
      return s
    }
    function value(line) {
      sub(/[[:space:]]*#.*/, "", line)
      sub(/^[^=]*=[[:space:]]*/, "", line)
      line = trim(line)
      gsub(/^["`]|["`]$/, "", line)
      return line
    }
    function list_value(line) {
      line = value(line)
      gsub(/[\[\]"`]/, "", line)
      gsub(/[[:space:]]/, "", line)
      return line
    }
    function flush_group() {
      if (in_group && reason != "" && depends_on != "") {
        found = 1
      }
    }
    /^[[:space:]]*\[\[pr_strategy_decision\.group_rationale\]\][[:space:]]*$/ {
      flush_group()
      in_group = 1
      reason = ""
      depends_on = ""
      next
    }
    /^[[:space:]]*\[/ {
      if (in_group) {
        flush_group()
        in_group = 0
      }
      next
    }
    in_group {
      line = $0
      sub(/[[:space:]]*#.*/, "", line)
      key = line
      sub(/[[:space:]]*=.*/, "", key)
      key = trim(key)
      if (key == "reason") {
        reason = value(line)
      } else if (key == "depends_on" || key == "depends") {
        depends_on = list_value(line)
      }
    }
    END {
      flush_group()
      exit(found ? 0 : 1)
    }
  ' "$_manifest"
}

pr_strategy_decision_json() {
  _plan_dir="$1"
  _effective="$2"
  _recorded="$3"
  _override="$4"
  _override_mismatch="$5"

  _selected="$(parse_pr_strategy_decision_field "$_plan_dir" selected || true)"
  _recommended_by="$(parse_pr_strategy_decision_field "$_plan_dir" recommended_by || true)"
  _human_approved="$(parse_pr_strategy_decision_field "$_plan_dir" human_approved || true)"
  _approval_note="$(parse_pr_strategy_decision_field "$_plan_dir" approval_note || true)"
  _rationale="$(parse_pr_strategy_decision_field "$_plan_dir" rationale || true)"
  _group_rationale_count="$(count_pr_strategy_group_rationales "$_plan_dir")"
  _stacked_dependency_rationale=false
  if has_stacked_dependency_rationale "$_plan_dir"; then
    _stacked_dependency_rationale=true
  fi

  jq -n \
    --arg effective "$_effective" \
    --arg selected "$_selected" \
    --arg recorded "$_recorded" \
    --arg recommended_by "$_recommended_by" \
    --arg human_approved "$_human_approved" \
    --arg approval_note "$_approval_note" \
    --arg rationale "$_rationale" \
    --arg override "$_override" \
    --argjson override_mismatch "$_override_mismatch" \
    --argjson group_rationale_count "$_group_rationale_count" \
    --argjson stacked_dependency_rationale "$_stacked_dependency_rationale" \
    '{
      effective: $effective,
      selected: (if $selected == "" then null else $selected end),
      recorded_strategy: (if $recorded == "" then null else $recorded end),
      recommended_by: (if $recommended_by == "" then null else $recommended_by end),
      human_approved: (
        if $human_approved == "true" then true
        elif $human_approved == "false" then false
        elif $human_approved == "" then null
        else $human_approved end
      ),
      approval_note: (if $approval_note == "" then null else $approval_note end),
      rationale: (if $rationale == "" then null else $rationale end),
      override: (if $override == "" then null else $override end),
      override_mismatch: $override_mismatch,
      group_rationale_count: $group_rationale_count,
      stacked_dependency_rationale: $stacked_dependency_rationale
    }'
}

normalize_list_value() {
  _raw="$1"
  printf '%s' "$_raw" |
    sed -E 's/^[^:=]*[:=][[:space:]]*//' |
    tr -d '[]"` ' |
    sed -E 's/[[:space:]]+//g;s/,+/,/g;s/^,//;s/,$//'
}

sanitize_branch_component() {
  printf '%s' "$1" |
    tr '[:upper:]' '[:lower:]' |
    sed -E 's/[^a-z0-9._-]+/-/g;s/-+/-/g;s/^-//;s/-$//'
}

# Parse explicit PR groups from _manifest.md.
# Output: name|slice_csv|depends_csv
parse_pr_groups() {
  _plan_dir="$1"
  _manifest="${_plan_dir}/_manifest.md"
  [ -f "$_manifest" ] || return 0

  _name=""
  _slices=""
  _depends=""
  _in_group=0
  while IFS= read -r line; do
    _clean="$(printf '%s' "$line" | sed -E 's/[[:space:]]*#.*$//;s/^[[:space:]]*//;s/[[:space:]]*$//')"
    [ -n "$_clean" ] || continue

    case "$_clean" in
      "[[pr_groups]]")
        if [ "$_in_group" -eq 1 ] && { [ -n "$_name" ] || [ -n "$_slices" ]; }; then
          printf '%s|%s|%s\n' "$_name" "$_slices" "$_depends"
        fi
        _in_group=1
        _name=""
        _slices=""
        _depends=""
        continue
        ;;
      "["*)
        if [ "$_in_group" -eq 1 ] && { [ -n "$_name" ] || [ -n "$_slices" ]; }; then
          printf '%s|%s|%s\n' "$_name" "$_slices" "$_depends"
        fi
        _in_group=0
        _name=""
        _slices=""
        _depends=""
        continue
        ;;
    esac

    [ "$_in_group" -eq 1 ] || continue

    case "$_clean" in
      name[[:space:]]*=*|"- name:"*)
        _name="$(printf '%s' "$_clean" | sed -E 's/^[^:=]*[:=][[:space:]]*//;s/["'\''`]//g;s/^[[:space:]]*//;s/[[:space:]]*$//')"
        _name="$(sanitize_branch_component "$_name")"
        ;;
      slices[[:space:]]*=*|"- slices:"*)
        _slices="$(normalize_list_value "$_clean")"
        ;;
      depends[[:space:]]*=*|dependencies[[:space:]]*=*|"- depends:"*|"- dependencies:"*)
        _depends="$(normalize_list_value "$_clean")"
        ;;
    esac
  done < "$_manifest"

  if [ "$_in_group" -eq 1 ] && { [ -n "$_name" ] || [ -n "$_slices" ]; }; then
    printf '%s|%s|%s\n' "$_name" "$_slices" "$_depends"
  fi
}

all_slice_csv() {
  _slices_file="$1"
  _csv=""
  while IFS='|' read -r s _o _d _f _p; do
    [ -n "$s" ] || continue
    _csv="${_csv:+${_csv},}${s}"
  done < "$_slices_file"
  printf '%s\n' "$_csv"
}

default_pr_groups() {
  _slices_file="$1"
  _all="$(all_slice_csv "$_slices_file")"
  [ -n "$_all" ] || return 0
  printf 'all|%s|\n' "$_all"
}

resolve_slice_ref() {
  _ref="$1"
  _slices_file="$2"
  _ref="$(printf '%s' "$_ref" | sed -E 's/^slice[-_ ]*//;s/^[[:space:]]*//;s/[[:space:]]*$//')"
  [ -n "$_ref" ] || return 1

  _match=""
  _matches=0
  while IFS='|' read -r s _o _d _f _p; do
    [ -n "$s" ] || continue
    case "$s" in
      "$_ref")
        printf '%s\n' "$s"
        return 0
        ;;
      "$_ref"-*)
        _match="$s"
        _matches=$((_matches + 1))
        ;;
    esac
  done < "$_slices_file"

  if [ "$_matches" -eq 1 ]; then
    printf '%s\n' "$_match"
    return 0
  fi

  return 1
}

normalize_pr_groups() {
  _groups_raw_file="$1"
  _slices_file="$2"
  _groups_out_file="$3"

  : > "$_groups_out_file"
  while IFS='|' read -r name slices depends; do
    [ -n "$name" ] || name="group"
    _name="$(sanitize_branch_component "$name")"
    if [ -z "$_name" ]; then
      log_error "Invalid PR group name: ${name}"
      return 1
    fi
    if [ -z "$slices" ]; then
      log_error "PR group '${_name}' has no slices"
      return 1
    fi

    _resolved=""
    printf '%s\n' "$slices" | tr ',' '\n' | while IFS= read -r ref; do
      [ -n "$ref" ] || continue
      if _slice="$(resolve_slice_ref "$ref" "$_slices_file")"; then
        printf '%s\n' "$_slice"
      else
        printf 'ERROR:%s\n' "$ref"
      fi
    done > "${_groups_out_file}.$$.slices"

    if grep -q '^ERROR:' "${_groups_out_file}.$$.slices"; then
      # head -1 may cause SIGPIPE to sed/grep if multiple ERROR lines exist; || true suppresses SIGPIPE propagation under pipefail
      _bad="$(grep '^ERROR:' "${_groups_out_file}.$$.slices" | sed 's/^ERROR://' | head -1 || true)"
      rm -f "${_groups_out_file}.$$.slices"
      log_error "PR group '${_name}' references unknown slice: ${_bad}"
      return 1
    fi

    _resolved="$(paste -sd, "${_groups_out_file}.$$.slices" 2>/dev/null || true)"
    rm -f "${_groups_out_file}.$$.slices"
    if [ -z "$_resolved" ]; then
      log_error "PR group '${_name}' resolved to no slices"
      return 1
    fi
    printf '%s|%s|%s\n' "$_name" "$_resolved" "$depends" >> "$_groups_out_file"
  done < "$_groups_raw_file"
}

pr_groups_json() {
  _groups_file="$1"
  _json="[]"
  while IFS='|' read -r name slices depends; do
    [ -n "$name" ] || continue
    _json="$(printf '%s' "$_json" | jq \
      --arg name "$name" \
      --arg slices "$slices" \
      --arg depends "$depends" \
      '. += [{name:$name,slices:($slices | split(",") | map(select(. != ""))),depends:($depends | split(",") | map(select(. != "")))}]')"
  done < "$_groups_file"
  printf '%s\n' "$_json"
}

# Auto-detect shared files: files that appear in more than one slice
detect_shared_files() {
  _slices_data="$1"
  _all_files=""

  echo "$_slices_data" | while IFS='|' read -r _s _o _d files _p; do
    echo "$files" | tr ',' '\n' | while IFS= read -r f; do
      _f="$(echo "$f" | tr -d ' ')"
      if [ -n "$_f" ]; then
        echo "$_f"
      fi
    done
  done | sort | uniq -d
}

# Check if a slice has dependencies on locked files that another running slice owns
check_locklist_conflict() {
  _slice_files="$1"
  _locklist="$2"
  _running_slices_files="$3"

  echo "$_slice_files" | tr ',' '\n' | while IFS= read -r f; do
    _f="$(echo "$f" | tr -d ' ')"
    if [ -z "$_f" ]; then continue; fi
    # Check if this file is in the locklist
    if echo "$_locklist" | grep -qF "$_f"; then
      # Check if any running slice also touches this file
      if echo "$_running_slices_files" | grep -qF "$_f"; then
        echo "$_f"
        return 0
      fi
    fi
  done
}

# ═══════════════════════════════════════════════════════════════════
# Integration branch management
# ═══════════════════════════════════════════════════════════════════

INTEGRATION_BRANCH=""
PLAN_SLUG=""

# Extract slug from plan path for branch naming
extract_plan_slug() {
  _path="$1"
  if [ -d "$_path" ]; then
    basename "$_path"
  else
    basename "$_path" .md
  fi
}

# Create the integration branch used for full merged verification.
create_integration_branch() {
  _plan_file="$1"
  _base="$2"
  INTEGRATION_BRANCH="$("${SCRIPT_DIR}/branch-name.sh" from-plan "$_plan_file")" || return 1

  if git rev-parse --verify "$INTEGRATION_BRANCH" >/dev/null 2>&1; then
    log "Integration branch already exists: ${INTEGRATION_BRANCH}"
    return 0
  fi

  git branch "$INTEGRATION_BRANCH" "$_base" 2>/dev/null || {
    log_error "Failed to create integration branch: ${INTEGRATION_BRANCH}"
    return 1
  }
  log "Created integration branch: ${INTEGRATION_BRANCH} (from ${_base})"
}

slice_branch_name() {
  _slug="$1"
  printf '%s-%s\n' "$INTEGRATION_BRANCH" "$_slug"
}

group_branch_name() {
  _group="$1"
  _safe_group="$(sanitize_branch_component "$_group")"
  printf '%s-%s\n' "$INTEGRATION_BRANCH" "$_safe_group"
}

# ═══════════════════════════════════════════════════════════════════
# Worktree management
# ═══════════════════════════════════════════════════════════════════

create_worktree() {
  _slug="$1"
  # Always use integration branch as base
  _base_branch="${INTEGRATION_BRANCH:-$(git rev-parse --abbrev-ref HEAD)}"
  _wt_path="${WORKTREE_BASE}/${_slug}"
  _wt_branch="$(slice_branch_name "$_slug")"
  if ! "${SCRIPT_DIR}/branch-name.sh" validate "$_wt_branch" >/dev/null 2>&1; then
    log_error "Invalid generated slice branch name: ${_wt_branch}"
    return 1
  fi

  if [ -d "$_wt_path" ]; then
    log "Worktree already exists: ${_wt_path}"
    return 0
  fi

  mkdir -p "$WORKTREE_BASE"
  git worktree add -b "$_wt_branch" "$_wt_path" "$_base_branch" 2>/dev/null || {
    # Branch might already exist
    git worktree add "$_wt_path" "$_wt_branch" 2>/dev/null || {
      log_error "Failed to create worktree for slice: ${_slug}"
      return 1
    }
  }
  log "Created worktree: ${_wt_path} (branch: ${_wt_branch}, base: ${_base_branch})"
}

remove_worktree() {
  _slug="$1"
  _wt_path="${WORKTREE_BASE}/${_slug}"

  if [ -d "$_wt_path" ]; then
    git worktree remove "$_wt_path" --force 2>/dev/null || true
    log "Removed worktree: ${_wt_path}"
  fi
}

# ═══════════════════════════════════════════════════════════════════
# Slice execution
# ═══════════════════════════════════════════════════════════════════

# Run ralph-pipeline.sh in a worktree for a single slice
run_slice() {
  _slug="$1"
  _objective="$2"
  _slice_plan="${3:-}"
  _wt_path="${WORKTREE_BASE}/${_slug}"
  _log_file="${ORCH_STATE}/slice-${_slug}.log"

  log "Starting slice: ${_slug} — ${_objective}"

  if [ "$DRY_RUN" -eq 1 ]; then
    log "[DRY RUN] Would run ralph-pipeline.sh in ${_wt_path}"
    log "[DRY RUN] Slice plan: ${_slice_plan:-none}"
    echo "complete" > "${ORCH_STATE}/slice-${_slug}.status"
    return 0
  fi

  # Copy slice plan into the worktree so the agent can read it via relative path
  _wt_plan_path=""
  if [ -n "$_slice_plan" ] && [ -f "$_slice_plan" ]; then
    _wt_plan_dir="${_wt_path}/$(dirname "$_slice_plan")"
    mkdir -p "$_wt_plan_dir"
    cp "$_slice_plan" "${_wt_path}/${_slice_plan}"
    _wt_plan_path="$_slice_plan"
  fi

  # Initialize pipeline state in the worktree
  (
    cd "$_wt_path"
    "${SCRIPT_DIR}/ralph-loop-init.sh" general "$_objective" "$_wt_plan_path" 2>&1 || true
    _resume_arg=""
    if [ "$RESUME" -eq 1 ]; then
      _resume_arg="--resume"
    fi
    # shellcheck disable=SC2086  # optional single flag
    "${SCRIPT_DIR}/ralph-pipeline.sh" \
      $_resume_arg \
      --skip-pr \
      --max-iterations "$MAX_ITERATIONS" \
      2>&1
  ) > "$_log_file" 2>&1 &

  _pid=$!
  echo "$_pid" > "${ORCH_STATE}/slice-${_slug}.pid"
  echo "running" > "${ORCH_STATE}/slice-${_slug}.status"
  date +%s > "${ORCH_STATE}/slice-${_slug}.started"
  log "Slice ${_slug} started (PID: ${_pid})"
}

# Check if a slice has completed
check_slice_status() {
  _slug="$1"
  _status_file="${ORCH_STATE}/slice-${_slug}.status"
  _pid_file="${ORCH_STATE}/slice-${_slug}.pid"

  if [ ! -f "$_status_file" ]; then
    echo "pending"
    return
  fi

  _status="$(cat "$_status_file" | tr -d '[:space:]')"
  if [ "$_status" != "running" ]; then
    echo "$_status"
    return
  fi

  # Check if PID is still running
  if [ -f "$_pid_file" ]; then
    _pid="$(cat "$_pid_file" | tr -d '[:space:]')"
    if kill -0 "$_pid" 2>/dev/null; then
      echo "running"
    else
      # Process ended — clean up stale PID file and check exit code
      rm -f "$_pid_file"
      _wt_path="${WORKTREE_BASE}/${_slug}"
      _ckpt="${_wt_path}/.harness/state/pipeline/checkpoint.json"
      if [ -f "$_ckpt" ]; then
        _ckpt_status="$(jq -r '.status // "unknown"' "$_ckpt" 2>/dev/null || echo "unknown")"
        echo "$_ckpt_status" > "$_status_file"
        echo "$_ckpt_status"
      else
        echo "failed" > "$_status_file"
        echo "failed"
      fi
    fi
  else
    echo "unknown"
  fi
}

# Wait for a specific slice to complete
wait_for_slice() {
  _slug="$1"
  _pid_file="${ORCH_STATE}/slice-${_slug}.pid"

  if [ ! -f "$_pid_file" ]; then
    return 0
  fi

  _pid="$(cat "$_pid_file")"
  if kill -0 "$_pid" 2>/dev/null; then
    log "Waiting for slice ${_slug} (PID: ${_pid})..."
    wait "$_pid" 2>/dev/null || true
  fi
}

# Sequential merge of completed slice branches into the integration branch.
# Merges slices in file-order (which matches dependency order from parse_slices).
# Aborts on first conflict.
integration_merge() {
  _int_branch="$1"
  _slices_file="$2"
  _conflicts=0
  _merged=0

  log "Running sequential merge into ${_int_branch}..."

  # Save current branch to return to later
  _orig_branch="$(git rev-parse --abbrev-ref HEAD)"

  git checkout "$_int_branch" 2>/dev/null || {
    log_error "Failed to checkout integration branch: ${_int_branch}"
    return 1
  }

  # Merge each completed slice in order
  while IFS='|' read -r s _o _d _f _p; do
    _status_file="${ORCH_STATE}/slice-${s}.status"
    [ -f "$_status_file" ] || continue
    _status="$(cat "$_status_file")"
    if [ "$_status" != "complete" ]; then
      log "Skipping slice ${s} (status: ${_status})"
      continue
    fi

    _slice_branch="$(slice_branch_name "$s")"

    log "Merging ${_slice_branch} into ${_int_branch}..."
    # Intentionally unquoted heredoc: branch names are safe and need expansion
    _merge_msg="chore: merge ${_slice_branch} into ${_int_branch}"
    if ! git merge --no-ff "$_slice_branch" -m "$_merge_msg" 2>/dev/null; then
      log_error "CONFLICT merging ${_slice_branch} into ${_int_branch}"
      git merge --abort 2>/dev/null || true
      _conflicts=$((_conflicts + 1))
      # Return to original branch before reporting error
      git checkout "$_orig_branch" 2>/dev/null || true
      log_error "Sequential merge aborted at slice ${s}. ${_merged} slice(s) merged before conflict."
      return 1
    fi
    _merged=$((_merged + 1))
    log "Merged ${_slice_branch} (${_merged} total)"
  done < "$_slices_file"

  # Return to original branch
  git checkout "$_orig_branch" 2>/dev/null || true

  log "Sequential merge complete: ${_merged} slice(s) merged into ${_int_branch}"
  return 0
}

merge_group_branch() {
  _group_name="$1"
  _group_slices="$2"
  _branch="$3"
  _base_branch="$4"
  _orig_branch="$(git rev-parse --abbrev-ref HEAD)"

  if git rev-parse --verify "$_branch" >/dev/null 2>&1; then
    git branch -D "$_branch" >/dev/null 2>&1 || {
      log_error "Failed to reset existing group branch: ${_branch}"
      return 1
    }
  fi

  git branch "$_branch" "$_base_branch" 2>/dev/null || {
    log_error "Failed to create group branch: ${_branch}"
    return 1
  }
  git checkout "$_branch" >/dev/null 2>&1 || {
    log_error "Failed to checkout group branch: ${_branch}"
    return 1
  }

  log "Building PR group '${_group_name}' on ${_branch}..."
  _group_slices_tmp="${ORCH_STATE}/.group-${_group_name}-slices.$$"
  printf '%s\n' "$_group_slices" | tr ',' '\n' > "$_group_slices_tmp"
  _rc=0
  while IFS= read -r s; do
    [ -n "$s" ] || continue
    _slice_branch="$(slice_branch_name "$s")"
    log "Merging ${_slice_branch} into ${_branch}..."
    _merge_msg="chore: merge ${_slice_branch} into ${_branch}"
    git merge --no-ff "$_slice_branch" -m "$_merge_msg" >/dev/null 2>&1 || {
      log_error "CONFLICT merging ${_slice_branch} into group branch ${_branch}"
      git merge --abort >/dev/null 2>&1 || true
      _rc=1
      break
    }
  done < "$_group_slices_tmp"
  rm -f "$_group_slices_tmp"

  git checkout "$_orig_branch" >/dev/null 2>&1 || true
  if [ "$_rc" -ne 0 ]; then
    return 1
  fi
  log "Group branch ready: ${_branch}"
}

create_pr_groups() {
  _strategy="$1"
  _groups_file="$2"
  _base_branch="$3"
  _plan_slug="$4"
  _integration_sha="$5"
  _pr_urls_file="${ORCH_STATE}/.pr-urls.jsonl"
  : > "$_pr_urls_file"

  if ! command -v gh >/dev/null 2>&1; then
    log_error "gh CLI not found — cannot create grouped PRs. Install gh and retry."
    return 1
  fi

  _previous_base="$_base_branch"
  while IFS='|' read -r name slices depends; do
    [ -n "$name" ] || continue
    _branch="$(group_branch_name "$name")"
    _pr_base="$_base_branch"
    if [ "$_strategy" = "stacked" ]; then
      _pr_base="$_previous_base"
    fi

    merge_group_branch "$name" "$slices" "$_branch" "$_pr_base" || return 1

    _pr_type="$(printf '%s' "$_branch" | cut -d/ -f1)"
    log "Creating ${_strategy} PR for group '${name}': ${_branch} → ${_pr_base}..."
    git push -u origin "$_branch" >/dev/null 2>&1 || {
      log_error "Failed to push group branch: ${_branch}"
      return 1
    }

    _pr_body="$(cat <<PR_BODY
## Summary

Ralph Loop ${_strategy} PR group: ${name}

- Plan: ${_plan_slug}
- Strategy: ${_strategy}
- Group branch: ${_branch}
- Integration branch: ${INTEGRATION_BRANCH}
- Integration verification SHA: ${_integration_sha}
- Slices: ${slices}
- Group dependencies: ${depends:-none}

## Related PRs

Other group PRs are recorded in \`${ORCH_STATE}/.pr-urls.jsonl\` as they are created.

## Test plan

- [x] Slice pipelines passed before grouping
- [x] Full integration merge passed
- [x] Full integration pipeline passed without producing unsubmitted fix commits
- [ ] CI checks pass on this PR

Generated by Ralph Orchestrator
PR_BODY
)"

    _pr_url="$(gh pr create \
      --base "$_pr_base" \
      --head "$_branch" \
      --title "${_pr_type}: ${_plan_slug} (${name})" \
      --body "$_pr_body" 2>/dev/null)" || {
      log_error "Failed to create PR for group '${name}'"
      return 1
    }

    if ! "${SCRIPT_DIR}/ensure-pr-title-prefix.sh" "$_pr_url" >/dev/null 2>&1; then
      log_error "Grouped PR exists but could not be verified with branch type title prefix: ${_pr_url}"
      return 1
    fi
    if ! "${SCRIPT_DIR}/ensure-pr-ready.sh" "$_pr_url" >/dev/null 2>&1; then
      log_error "Grouped PR exists but could not be verified as ready-for-review: ${_pr_url}"
      return 1
    fi

    jq -n --arg name "$name" --arg branch "$_branch" --arg base "$_pr_base" --arg url "$_pr_url" \
      '{name:$name,branch:$branch,base:$base,url:$url}' >> "$_pr_urls_file"
    log "Grouped PR created: ${_pr_url}"

    if [ "$_strategy" = "stacked" ]; then
      _previous_base="$_branch"
    fi
  done < "$_groups_file"

  return 0
}

pr_urls_json() {
  _file="${1:-${ORCH_STATE}/.pr-urls.jsonl}"
  _json="[]"
  [ -f "$_file" ] || { printf '%s\n' "$_json"; return; }
  while IFS= read -r line; do
    [ -n "$line" ] || continue
    _json="$(printf '%s' "$_json" | jq --argjson item "$line" '. += [$item]')"
  done < "$_file"
  printf '%s\n' "$_json"
}

cleanup_success_artifacts() {
  _strategy="$1"
  _slices_file="$2"
  [ "$_strategy" != "unified" ] || return 0

  log "Cleaning temporary Ralph Loop branches/worktrees after successful ${_strategy} PR creation..."
  _orig_branch="$(git rev-parse --abbrev-ref HEAD)"

  while IFS='|' read -r s _o _d _f _p; do
    [ -n "$s" ] || continue
    remove_worktree "$s"
    _slice_branch="$(slice_branch_name "$s")"
    if git rev-parse --verify "$_slice_branch" >/dev/null 2>&1; then
      git branch -D "$_slice_branch" >/dev/null 2>&1 || {
        log_error "Failed to delete temporary slice branch: ${_slice_branch}"
        return 1
      }
      log "Deleted temporary slice branch: ${_slice_branch}"
    fi
  done < "$_slices_file"

  case "$_orig_branch" in
    "$INTEGRATION_BRANCH"|"$INTEGRATION_BRANCH"-*)
      git checkout "$_base_branch" >/dev/null 2>&1 || true
      ;;
  esac

  if git rev-parse --verify "$INTEGRATION_BRANCH" >/dev/null 2>&1; then
    git branch -D "$INTEGRATION_BRANCH" >/dev/null 2>&1 || {
      log_error "Failed to delete temporary integration branch: ${INTEGRATION_BRANCH}"
      return 1
    }
    log "Deleted temporary integration branch: ${INTEGRATION_BRANCH}"
  fi

  jq --arg cleanup "success" \
    '.cleanup_status = $cleanup' \
    "${ORCH_STATE}/orchestrator.json" > "${ORCH_STATE}/orchestrator.tmp.$$.json" \
    && mv "${ORCH_STATE}/orchestrator.tmp.$$.json" "${ORCH_STATE}/orchestrator.json"
}

print_cleanup_instructions() {
  _reason="$1"
  log_error "Temporary Ralph Loop branches retained for diagnosis (${_reason})."
  log_error "After inspection, run: ./scripts/ralph cleanup --plan ${PLAN_FILE}"
}

# Create a unified PR from the integration branch to the base branch
create_unified_pr() {
  _int_branch="$1"
  _base_branch="$2"
  _plan_slug="$3"
  _total_slices="$4"
  _completed="$5"

  log "Creating unified PR: ${_int_branch} → ${_base_branch}..."

  if ! command -v gh >/dev/null 2>&1; then
    log_error "gh CLI not found — cannot create PR. Install gh and retry."
    return 1
  fi
  if ! "${SCRIPT_DIR}/branch-name.sh" validate "$_int_branch" >/dev/null 2>&1; then
    log_error "Invalid unified PR branch name: ${_int_branch}"
    return 1
  fi
  _pr_type="$(printf '%s' "$_int_branch" | cut -d/ -f1)"

  # Push integration branch
  git push -u origin "$_int_branch" 2>/dev/null || {
    log_error "Failed to push integration branch"
    return 1
  }

  # Create PR (heredoc intentionally unquoted for variable expansion)
  _pr_body="$(cat <<PR_BODY
## Summary

Unified PR for Ralph Loop parallel slices: ${_plan_slug}

- Total slices: ${_total_slices}
- Completed: ${_completed}
- Integration branch: ${_int_branch}

## Slice branches merged

$(for sf in "${ORCH_STATE}"/slice-*.status; do
  [ -f "$sf" ] || continue
  _name="$(basename "$sf" | sed 's/^slice-//;s/\.status$//')"
  _ss="$(cat "$sf")"
  printf '%s %s: %s\n' "-" "$_name" "$_ss"
done)

## Test plan

- [x] All slice pipelines passed (self-review, verify, test)
- [x] Integration pipeline passed (self-review, verify, test, sync-docs, cross-review)
- [x] All self-review findings fixed (--fix-all)
- [x] Integration merge passed without conflicts
- [ ] CI checks pass on this PR

Generated by Ralph Orchestrator
PR_BODY
)"

  _pr_url="$(gh pr create \
    --base "$_base_branch" \
    --head "$_int_branch" \
    --title "${_pr_type}: ${_plan_slug}" \
    --body "$_pr_body" 2>/dev/null)" || {
    log_error "Failed to create unified PR"
    return 1
  }

  if ! "${SCRIPT_DIR}/ensure-pr-title-prefix.sh" "$_pr_url" >/dev/null 2>&1; then
    log_error "Unified PR exists but could not be verified with branch type title prefix: ${_pr_url}"
    return 1
  fi

  if ! "${SCRIPT_DIR}/ensure-pr-ready.sh" "$_pr_url" >/dev/null 2>&1; then
    log_error "Unified PR exists but could not be verified as ready-for-review: ${_pr_url}"
    return 1
  fi

  log "Unified PR created: ${_pr_url}"
  echo "$_pr_url"
}

# ═══════════════════════════════════════════════════════════════════
# Integration pipeline — post-merge quality gate
# ═══════════════════════════════════════════════════════════════════

# Run ralph-pipeline.sh on the integration branch with --skip-pr --fix-all
# to catch cross-module issues before grouped/stacked/unified PR creation.
run_integration_pipeline() {
  _int_branch="$1"
  _base_branch="$2"
  _plan_slug="$3"

  log "═══ Integration Pipeline ═══"

  # 1. Set up .harness/state/loop/
  mkdir -p .harness/state/loop

  cat > .harness/state/loop/PROMPT.md <<INT_PROMPT
# Integration Review & Fix

All slices merged into ${_int_branch}. Base: ${_base_branch}
Full diff: git diff ${_base_branch}...HEAD
Plan: docs/plans/active/${_plan_slug}/_manifest.md

## Objective

- Review the integration diff for cross-module issues
- Fix ALL self-review findings (CRITICAL/HIGH/MEDIUM/LOW)
- Fix ALL codex findings (ACTION_REQUIRED + WORTH_CONSIDERING)
- Run tests and fix regressions
- Signal COMPLETE only when self-review is clean and tests pass

## Constraints

- Fix only, no new features
- Commit format: fix: <description>
INT_PROMPT

  cat > .harness/state/loop/task.json <<INT_TASK
{
  "type": "integration",
  "slug": "${_plan_slug}",
  "branch": "${_int_branch}",
  "base": "${_base_branch}",
  "started": "$(ts)"
}
INT_TASK

  : > .harness/state/loop/progress.log

  # 2. Copy & adapt pipeline prompts for integration context
  mkdir -p .harness/state/pipeline
  for f in .claude/skills/loop/prompts/pipeline-*.md; do
    [ -f "$f" ] || continue
    _name="$(basename "$f")"
    sed "s|Then run \`git diff\`|Then run \`git diff ${_base_branch}...HEAD\`|g" \
      "$f" > ".harness/state/pipeline/${_name}"
  done

  # 3. Run pipeline with --skip-pr --fix-all
  "${SCRIPT_DIR}/ralph-pipeline.sh" \
    --skip-pr --fix-all \
    --max-iterations 10 \
    --max-inner-cycles 5 \
    --max-outer-cycles 2

  # 4. Check terminal status
  _int_status="$(jq -r '.status // "unknown"' .harness/state/pipeline/checkpoint.json 2>/dev/null || echo 'unknown')"
  if [ "$_int_status" = "complete" ]; then
    log "═══ Integration Pipeline PASSED ═══"
    return 0
  else
    log_error "Integration Pipeline: ${_int_status}"
    return 1
  fi
}

# ═══════════════════════════════════════════════════════════════════
# Main
# ═══════════════════════════════════════════════════════════════════

main() {
  log "═══ Ralph Orchestrator ═══"
  log "Plan: ${PLAN_FILE}"
  log "Loop driver: ${RALPH_LOOP_DRIVER}"
  if [ "$RALPH_LOOP_DRIVER" = "codex" ]; then
    log "  codex sandbox: ${RALPH_CODEX_SANDBOX}"
    log "  codex approval policy: ${RALPH_CODEX_APPROVAL_POLICY}"
    log "  cross-review reviewer: claude/${RALPH_CLAUDE_REVIEWER_MODEL}"
  fi
  log "Max parallel: ${MAX_PARALLEL}"
  log "Max iterations per slice: ${MAX_ITERATIONS}"
  log "Preflight only: ${PREFLIGHT_ONLY}"
  log "Resume: ${RESUME}"
  log "Dry run: ${DRY_RUN}"
  log ""

  # --- Extract plan slug and parse plan ---
  PLAN_SLUG="$(extract_plan_slug "$PLAN_FILE")"
  _base_branch="$(git rev-parse --abbrev-ref HEAD)"
  # Export the launch branch as RALPH_XREVIEW_BASE so per-slice pipelines and the
  # integration pipeline diff against the true merge target (the branch the Loop
  # was started from), not the repo default. detect_base_branch() in
  # ralph-cli-driver.sh checks this variable first. An operator-supplied
  # RALPH_XREVIEW_BASE takes precedence and is preserved as-is.
  RALPH_XREVIEW_BASE="${RALPH_XREVIEW_BASE:-$_base_branch}"
  export RALPH_XREVIEW_BASE
  INTEGRATION_BRANCH="$("${SCRIPT_DIR}/branch-name.sh" from-plan "$PLAN_FILE")" || return 1
  slices_data="$(parse_slices "$PLAN_FILE")"
  locklist="$(parse_locklist "$PLAN_FILE")"
  _manifest_strategy="$(parse_pr_strategy "$PLAN_FILE" || true)"
  _decision_strategy="$(parse_pr_strategy_decision_field "$PLAN_FILE" selected || true)"
  _recorded_strategy="${_decision_strategy:-$_manifest_strategy}"
  _override_mismatch=false

  if [ -n "$_manifest_strategy" ] && [ -n "$_decision_strategy" ] && [ "$_manifest_strategy" != "$_decision_strategy" ]; then
    log_warn "Manifest pr_strategy '${_manifest_strategy}' differs from pr_strategy_decision.selected '${_decision_strategy}'; using decision selected value."
  fi

  PR_STRATEGY="${PR_STRATEGY_OVERRIDE:-${_recorded_strategy:-grouped}}"
  case "$PR_STRATEGY" in
    grouped|stacked|unified) ;;
    *)
      log_error "Invalid PR strategy '${PR_STRATEGY}'. Expected grouped, stacked, or unified."
      return 1
      ;;
  esac
  if [ -n "$PR_STRATEGY_OVERRIDE" ] && [ -n "$_recorded_strategy" ] && [ "$PR_STRATEGY_OVERRIDE" != "$_recorded_strategy" ]; then
    _override_mismatch=true
    log_warn "Runtime --pr-strategy '${PR_STRATEGY_OVERRIDE}' overrides manifest PR strategy '${_recorded_strategy}'."
  fi
  if [ "$PR_STRATEGY" = "stacked" ] && ! has_stacked_dependency_rationale "$PLAN_FILE"; then
    log_warn "Stacked PR strategy selected without dependency rationale. Add [[pr_strategy_decision.group_rationale]] with depends_on and reason before plan approval."
  fi
  _decision_human_approved="$(parse_pr_strategy_decision_field "$PLAN_FILE" human_approved || true)"
  _decision_json="$(pr_strategy_decision_json "$PLAN_FILE" "$PR_STRATEGY" "${_recorded_strategy:-}" "$PR_STRATEGY_OVERRIDE" "$_override_mismatch")"
  if [ "$PR_STRATEGY" = "unified" ]; then
    UNIFIED_PR=1
  else
    UNIFIED_PR=0
  fi
  log "PR strategy: ${PR_STRATEGY}"
  log "PR strategy decision: selected=${_decision_strategy:-not-recorded}, human approved=${_decision_human_approved:-not-recorded}"

  # Auto-detect additional shared files
  auto_shared="$(detect_shared_files "$slices_data")"
  if [ -n "$auto_shared" ]; then
    locklist="$(printf '%s\n%s' "$locklist" "$auto_shared" | sort -u)"
    log "Auto-detected shared files added to locklist:"
    echo "$auto_shared" | while IFS= read -r f; do
      log "  - $f"
    done
  fi

  _slice_count="$(echo "$slices_data" | grep -c '|' || echo 0)"
  log "Found ${_slice_count} slice(s)"

  if [ "$_slice_count" -eq 0 ]; then
    log_error "No slices found in plan directory. Ensure directory contains slice-*.md files."
    exit 1
  fi

  _tmp_slices_file="${TMPDIR:-/tmp}/ralph-orch-slices.$$"
  _tmp_groups_raw_file="${TMPDIR:-/tmp}/ralph-orch-pr-groups-raw.$$"
  _tmp_groups_file="${TMPDIR:-/tmp}/ralph-orch-pr-groups.$$"
  _slices_file="$_tmp_slices_file"
  _groups_file="$_tmp_groups_file"
  echo "$slices_data" > "$_slices_file"

  _groups_raw_file="$_tmp_groups_raw_file"
  parse_pr_groups "$PLAN_FILE" > "$_groups_raw_file"
  if [ ! -s "$_groups_raw_file" ]; then
    default_pr_groups "$_slices_file" > "$_groups_raw_file"
  fi
  normalize_pr_groups "$_groups_raw_file" "$_slices_file" "$_groups_file" || return 1

  log ""
  log "Slices:"
  echo "$slices_data" | while IFS='|' read -r s o d f p; do
    log "  ${s}: ${o} (deps: ${d:-none}, plan: ${p:-none})"
  done
  log ""

  log "PR groups:"
  while IFS='|' read -r name slices depends; do
    log "  ${name}: ${slices} (depends: ${depends:-none})"
  done < "$_groups_file"
  log ""

  if [ -n "$locklist" ]; then
    log "Shared-file locklist:"
    echo "$locklist" | while IFS= read -r f; do
      log "  - $f"
    done
    log ""
  fi

  if [ "$PREFLIGHT_ONLY" -eq 1 ]; then
    if [ "$DRY_RUN" -eq 1 ]; then
      log "[DRY RUN] Preflight parsed ${_slice_count} slice(s). Would run: ${SCRIPT_DIR}/ralph-pipeline.sh --preflight"
      return 0
    fi
    mkdir -p "$ORCH_STATE" "$EVIDENCE_DIR"
    log "Running per-slice pipeline preflight probe once..."
    "${SCRIPT_DIR}/ralph-pipeline.sh" --preflight
    return $?
  fi

  if [ "$DRY_RUN" -eq 1 ]; then
    log "[DRY RUN] Plan parsed successfully. Would create ${_slice_count} worktree(s)."
    log "[DRY RUN] Integration branch: ${INTEGRATION_BRANCH}"
    log "[DRY RUN] PR strategy: ${PR_STRATEGY}"
    log "[DRY RUN] PR strategy decision: selected=${_decision_strategy:-not-recorded}, human approved=${_decision_human_approved:-not-recorded}"
    echo "$slices_data" | while IFS='|' read -r s o d f p; do
      log "[DRY RUN] Slice ${s}: worktree at ${WORKTREE_BASE}/${s}, branch $(slice_branch_name "$s"), plan: ${p:-none}"
    done
    while IFS='|' read -r name slices depends; do
      log "[DRY RUN] PR group ${name}: branch $(group_branch_name "$name"), slices ${slices}, depends: ${depends:-none}"
    done < "$_groups_file"
    rm -f "$_tmp_slices_file" "$_tmp_groups_raw_file" "$_tmp_groups_file"
    return 0
  fi

  # Always create an integration branch for sequential merge
  create_integration_branch "$PLAN_FILE" "$_base_branch"
  log "Integration branch: ${INTEGRATION_BRANCH}"
  log ""

  mkdir -p "$ORCH_STATE" "$EVIDENCE_DIR"
  _slices_file="${ORCH_STATE}/.slices.dat"
  _groups_raw_file="${ORCH_STATE}/.pr-groups.raw"
  _groups_file="${ORCH_STATE}/.pr-groups.dat"
  cp "$_tmp_slices_file" "$_slices_file"
  cp "$_tmp_groups_raw_file" "$_groups_raw_file"
  cp "$_tmp_groups_file" "$_groups_file"
  rm -f "$_tmp_slices_file" "$_tmp_groups_raw_file" "$_tmp_groups_file"

  # Save orchestrator state
  _started="$(ts)"
  _groups_json="$(pr_groups_json "$_groups_file")"
  cat > "${ORCH_STATE}/orchestrator.json" <<ORCH_JSON
{
  "schema_version": 1,
  "plan": "${PLAN_FILE}",
  "started": "${_started}",
  "max_parallel": ${MAX_PARALLEL},
  "max_iterations": ${MAX_ITERATIONS},
  "pr_strategy": "${PR_STRATEGY}",
  "pr_strategy_decision": ${_decision_json},
  "pr_groups": ${_groups_json},
  "unified_pr": $([ "$UNIFIED_PR" -eq 1 ] && echo true || echo false),
  "integration_branch": "${INTEGRATION_BRANCH}",
  "cleanup_status": "pending",
  "status": "running"
}
ORCH_JSON

  # --- Create worktrees ---
  while IFS='|' read -r s o d f p; do
    create_worktree "$s"
  done < "$_slices_file"

  # --- Execute slices respecting dependencies ---
  _running=0
  _completed=0
  _failed=0
  _total="$_slice_count"

  # Track running files for locklist
  : > "${ORCH_STATE}/.running_files"

  while [ "$((_completed + _failed))" -lt "$_total" ]; do
    # Try to start eligible slices
    while IFS='|' read -r s o d f p; do
      _s_status="$(check_slice_status "$s")"

      # Skip if already started or done (includes all terminal pipeline statuses)
      case "$_s_status" in
        running|complete|failed|stuck|repair_limit|aborted|config_error|gh_unavailable|timeout|max_iterations|max_inner_cycles|max_outer_cycles) continue ;;
      esac

      # Check dependency satisfaction (avoid pipe-subshell by using temp file)
      _deps_met=1
      if [ -n "$d" ]; then
        _deps_tmp="${ORCH_STATE}/.deps_check.$$.tmp"
        echo "$d" | tr ',' '\n' > "$_deps_tmp"
        while IFS= read -r dep; do
          _dep_slug="$(echo "$dep" | tr -d ' []' | tr '[:upper:]' '[:lower:]' | sed 's/^slice[- ]*//')"
          [ -z "$_dep_slug" ] && continue
          # Resolve short dep slug to full slug (e.g., "1" -> "1-ralph-tui")
          # The dep field may use short names like "slice-1" while actual slugs
          # include a suffix like "1-ralph-tui". Match by prefix.
          _resolved_slug="$_dep_slug"
          _match_count=0
          while IFS='|' read -r _rs _ro _rd _rf _rp; do
            case "$_rs" in
              "${_dep_slug}"-*|"${_dep_slug}") _resolved_slug="$_rs"; _match_count=$((_match_count + 1)) ;;
            esac
          done < "$_slices_file"
          if [ "$_match_count" -gt 1 ]; then
            log "Warning: ambiguous dependency '${_dep_slug}' matched ${_match_count} slices, using '${_resolved_slug}'"
          fi
          if [ "$_resolved_slug" = "$_dep_slug" ] && [ "$_match_count" -eq 0 ]; then
            log "Warning: dependency '${_dep_slug}' did not match any known slice"
          fi
          _dep_status="$(check_slice_status "$_resolved_slug")"
          if [ "$_dep_status" != "complete" ]; then
            _deps_met=0
            break
          fi
        done < "$_deps_tmp"
        rm -f "$_deps_tmp"
      fi

      if [ "$_deps_met" -eq 0 ]; then
        continue
      fi

      # Check locklist conflicts
      _running_files="$(cat "${ORCH_STATE}/.running_files" 2>/dev/null || true)"
      _conflict="$(check_locklist_conflict "$f" "$locklist" "$_running_files")"
      if [ -n "$_conflict" ]; then
        log "Slice ${s} deferred: locklist conflict on ${_conflict}"
        continue
      fi

      # Check parallel capacity
      _current_running=0
      _current_running="$(grep -c 'running' "${ORCH_STATE}"/slice-*.status 2>/dev/null)" || _current_running=0
      if [ "$_current_running" -ge "$MAX_PARALLEL" ]; then
        continue
      fi

      # Start the slice
      echo "$f" | tr ',' '\n' >> "${ORCH_STATE}/.running_files"
      run_slice "$s" "$o" "$p"
    done < "$_slices_file"

    # Update status counts, check timeouts, and rebuild running_files
    _completed=0
    _failed=0
    _running=0
    : > "${ORCH_STATE}/.running_files"
    _now_epoch="$(date +%s)"
    while IFS='|' read -r _rf_s _rf_o _rf_d _rf_f _rf_p; do
      _rf_status="$(check_slice_status "$_rf_s")"
      case "$_rf_status" in
        complete)                        _completed=$((_completed + 1)) ;;
        failed|stuck|repair_limit|aborted|config_error|gh_unavailable|timeout|max_iterations|max_inner_cycles|max_outer_cycles) _failed=$((_failed + 1)) ;;
        running)
          # Check for timeout
          _started_file="${ORCH_STATE}/slice-${_rf_s}.started"
          if [ -f "$_started_file" ]; then
            _start_epoch="$(cat "$_started_file" | tr -d '[:space:]')"
            _elapsed=$((_now_epoch - _start_epoch))
            if [ "$_elapsed" -ge "$RALPH_SLICE_TIMEOUT" ]; then
              log_error "Slice ${_rf_s} timed out after ${_elapsed}s (limit: ${RALPH_SLICE_TIMEOUT}s)"
              _pid_file="${ORCH_STATE}/slice-${_rf_s}.pid"
              if [ -f "$_pid_file" ]; then
                _spid="$(cat "$_pid_file" | tr -d '[:space:]')"
                kill "$_spid" 2>/dev/null || true
                # Keep .pid file so cleanup_on_exit can re-kill if process lingers
              fi
              echo "timeout" > "${ORCH_STATE}/slice-${_rf_s}.status"
              _failed=$((_failed + 1))
              continue
            fi
          fi
          _running=$((_running + 1))
          # Re-add only currently running slice files to locklist
          echo "$_rf_f" | tr ',' '\n' >> "${ORCH_STATE}/.running_files"
          ;;
      esac
    done < "$_slices_file"

    if [ "$((_completed + _failed))" -ge "$_total" ]; then
      break
    fi

    # Wait a bit before checking again
    sleep 5
  done

  log ""
  log "═══ Orchestrator Results ═══"
  log "Completed: ${_completed}/${_total}"
  log "Failed: ${_failed}/${_total}"

  # --- Integration merge ---
  _merge_status="skipped"
  _pr_url=""
  _pr_urls_json="[]"
  _cleanup_status="pending"
  _integration_sha=""

  if [ "$_completed" -gt 0 ] && [ "$_failed" -eq 0 ]; then
    # Sequential merge into integration branch
    if integration_merge "$INTEGRATION_BRANCH" "$_slices_file"; then
      _merge_status="clean"
      log "Sequential merge to ${INTEGRATION_BRANCH} passed."

      # Run integration pipeline on the merged branch
      _orig_branch="$(git rev-parse --abbrev-ref HEAD)"
      git checkout "$INTEGRATION_BRANCH" 2>/dev/null || true
      _integration_pre_pipeline_sha="$(git rev-parse HEAD)"

      if run_integration_pipeline "$INTEGRATION_BRANCH" "$_base_branch" "$PLAN_SLUG"; then
        _merge_status="pipeline_passed"
        _integration_sha="$(git rev-parse HEAD)"
        if [ "$PR_STRATEGY" != "unified" ] && [ "$_integration_sha" != "$_integration_pre_pipeline_sha" ]; then
          _merge_status="pipeline_fixed_unsubmitted"
          log_error "Integration pipeline produced fix commits on ${INTEGRATION_BRANCH}; grouped/stacked PRs would not contain those fixes."
          log_error "Apply the integration fixes back to the owning PR group branch or create an integration-fixes group, then rerun."
          print_cleanup_instructions "$_merge_status"
        elif [ "$PR_STRATEGY" = "unified" ]; then
          _pr_url="$(create_unified_pr "$INTEGRATION_BRANCH" "$_base_branch" "$PLAN_SLUG" "$_total" "$_completed")" || {
            log_error "Unified PR creation failed."
            print_cleanup_instructions "pr_creation_failed"
            _pr_url=""
          }
        else
          if create_pr_groups "$PR_STRATEGY" "$_groups_file" "$_base_branch" "$PLAN_SLUG" "$_integration_sha"; then
            _pr_urls_json="$(pr_urls_json "${ORCH_STATE}/.pr-urls.jsonl")"
            _pr_url="$(printf '%s' "$_pr_urls_json" | jq -r '.[0].url // ""')"
            if cleanup_success_artifacts "$PR_STRATEGY" "$_slices_file"; then
              _cleanup_status="success"
            else
              _cleanup_status="failed"
              print_cleanup_instructions "cleanup_failed"
            fi
          else
            _merge_status="pr_creation_failed"
            _cleanup_status="retained_for_diagnosis"
            print_cleanup_instructions "$_merge_status"
          fi
        fi
      else
        _merge_status="pipeline_failed"
        log_error "Integration pipeline failed. PR not created."
        _cleanup_status="retained_for_diagnosis"
        print_cleanup_instructions "$_merge_status"
      fi

      git checkout "$_orig_branch" 2>/dev/null || true
    else
      _merge_status="conflict"
      log_error "Sequential merge failed. Manual resolution needed on ${INTEGRATION_BRANCH}."
      _cleanup_status="retained_for_diagnosis"
      print_cleanup_instructions "$_merge_status"
    fi
  fi

  # --- Generate execution report ---
  _report_file="${EVIDENCE_DIR}/orchestrator-$(ts_file).json"

  cat > "$_report_file" <<REPORT_JSON
{
  "plan": "${PLAN_FILE}",
  "plan_slug": "${PLAN_SLUG}",
  "started": "${_started}",
  "ended": "$(ts)",
  "total_slices": ${_total},
  "completed": ${_completed},
  "failed": ${_failed},
  "merge_status": "${_merge_status}",
  "pr_strategy": "${PR_STRATEGY}",
  "pr_strategy_decision": ${_decision_json},
  "pr_groups": $(pr_groups_json "$_groups_file"),
  "unified_pr": $([ "$UNIFIED_PR" -eq 1 ] && echo true || echo false),
  "integration_branch": "${INTEGRATION_BRANCH}",
  "integration_sha": "${_integration_sha}",
  "cleanup_status": "${_cleanup_status}",
  "pr_url": "${_pr_url}",
  "pr_urls": ${_pr_urls_json}
}
REPORT_JSON

  log "Report: ${_report_file}"

  # Update orchestrator status
  _final_status="complete"
  if [ "$_failed" -gt 0 ]; then
    _final_status="partial"
  else
    case "$_merge_status" in
      pipeline_passed) _final_status="complete" ;;
      *) _final_status="partial" ;;
    esac
  fi

  jq --arg s "$_final_status" \
    --arg pr "${_pr_url}" \
    --arg merge_status "${_merge_status}" \
    --arg cleanup_status "${_cleanup_status}" \
    --arg integration_sha "${_integration_sha}" \
    --argjson pr_urls "${_pr_urls_json}" \
    '.status = $s | .ended = "'"$(ts)"'" | .pr_url = $pr | .pr_urls = $pr_urls | .merge_status = $merge_status | .cleanup_status = $cleanup_status | .integration_sha = $integration_sha' \
    "${ORCH_STATE}/orchestrator.json" > "${ORCH_STATE}/orchestrator.tmp.$$.json" \
    && mv "${ORCH_STATE}/orchestrator.tmp.$$.json" "${ORCH_STATE}/orchestrator.json"

  if [ "$_failed" -gt 0 ]; then
    log_error "Some slices failed. Check individual slice logs in ${ORCH_STATE}/"
    return 1
  fi

  if [ "$_final_status" != "complete" ]; then
    log_error "Ralph Orchestrator did not complete cleanly (merge_status=${_merge_status})."
    return 1
  fi

  return 0
}

main
