#!/usr/bin/env sh
# ralph-status-helpers.sh — helper functions for ralph status display
# Sourced by scripts/ralph cmd_status(). Not executable standalone.

# ═══════════════════════════════════════════════════════════════════
# Color control
# ═══════════════════════════════════════════════════════════════════

# Detect color support. Sets USE_COLOR=1 if colors should be used.
# Respects: --no-color flag (STATUS_NO_COLOR), NO_COLOR env, non-TTY stdout.
detect_color() {
  USE_COLOR=1
  if [ "${STATUS_NO_COLOR:-0}" -eq 1 ]; then
    USE_COLOR=0
  elif [ -n "${NO_COLOR:-}" ]; then
    USE_COLOR=0
  elif [ ! -t 1 ]; then
    USE_COLOR=0
  fi

  if [ "$USE_COLOR" -eq 1 ]; then
    C_RESET='\033[0m'
    C_BOLD='\033[1m'
    C_DIM='\033[2m'
    C_GREEN='\033[32m'
    C_YELLOW='\033[33m'
    C_RED='\033[31m'
    C_CYAN='\033[36m'
    C_BLUE='\033[34m'
    C_MAGENTA='\033[35m'
  else
    C_RESET=''
    C_BOLD=''
    C_DIM=''
    C_GREEN=''
    C_YELLOW=''
    C_RED=''
    C_CYAN=''
    C_BLUE=''
    C_MAGENTA=''
  fi
}

# ═══════════════════════════════════════════════════════════════════
# Time calculations
# ═══════════════════════════════════════════════════════════════════

# Convert ISO 8601 timestamp to epoch seconds.
# Works on both macOS (BSD date) and Linux (GNU date).
iso_to_epoch() {
  _iso="$1"
  [ -z "$_iso" ] && echo 0 && return

  # Try GNU date first (Linux, or GNU coreutils on macOS)
  _epoch="$(date -d "$_iso" '+%s' 2>/dev/null || true)"
  if [ -n "$_epoch" ]; then
    echo "$_epoch"
    return
  fi

  # Try BSD date (macOS native)
  # Strip trailing 'Z' and convert to format BSD date can parse
  _cleaned="$(echo "$_iso" | sed 's/Z$//' | sed 's/T/ /')"
  _epoch="$(date -j -f '%Y-%m-%d %H:%M:%S' "$_cleaned" '+%s' 2>/dev/null || true)"
  if [ -n "$_epoch" ]; then
    echo "$_epoch"
    return
  fi

  echo 0
}

# Get current epoch seconds.
now_epoch() {
  date '+%s'
}

# Format duration in seconds to human-readable string (e.g., "12m 34s", "1h 5m").
format_duration() {
  _secs="$1"
  [ -z "$_secs" ] && echo "—" && return
  _secs="${_secs%%.*}"  # strip decimals
  [ "$_secs" -le 0 ] 2>/dev/null && echo "—" && return

  _hours=$((_secs / 3600))
  _mins=$(( (_secs % 3600) / 60 ))
  _s=$((_secs % 60))

  if [ "$_hours" -gt 0 ]; then
    printf '%dh %dm' "$_hours" "$_mins"
  elif [ "$_mins" -gt 0 ]; then
    printf '%dm %02ds' "$_mins" "$_s"
  else
    printf '%ds' "$_s"
  fi
}


# ═══════════════════════════════════════════════════════════════════
# Phase display
# ═══════════════════════════════════════════════════════════════════

# Resolve display phase with icon for a slice.
# Args: $1 = status (from .status file), $2 = phase (from checkpoint)
resolve_display_phase() {
  _status="$1"
  _phase="$2"

  case "$_status" in
    complete)
      printf '%b' "${C_GREEN}done${C_RESET}"
      ;;
    failed|stuck|repair_limit|aborted|config_error|gh_unavailable|timeout|max_iterations|max_inner_cycles|max_outer_cycles)
      printf '%b' "${C_RED}${_status}${C_RESET}"
      ;;
    pending)
      printf '%b' "${C_DIM}waiting${C_RESET}"
      ;;
    running)
      refine_inner_phase "$_phase"
      ;;
    *)
      printf '%b' "${C_DIM}${_status}${C_RESET}"
      ;;
  esac
}

# Refine display phase for running slices based on checkpoint phase.
# Args: $1 = phase from checkpoint
refine_inner_phase() {
  _phase="$1"
  case "$_phase" in
    inner)
      printf '%b' "${C_CYAN}implement${C_RESET}"
      ;;
    self-review|self_review)
      printf '%b' "${C_YELLOW}review${C_RESET}"
      ;;
    verify)
      printf '%b' "${C_YELLOW}verify${C_RESET}"
      ;;
    test)
      printf '%b' "${C_YELLOW}test${C_RESET}"
      ;;
    outer)
      printf '%b' "${C_BLUE}outer${C_RESET}"
      ;;
    preflight)
      printf '%b' "${C_DIM}preflight${C_RESET}"
      ;;
    *)
      printf '%b' "${C_CYAN}${_phase}${C_RESET}"
      ;;
  esac
}

# ═══════════════════════════════════════════════════════════════════
# Status icon helpers
# ═══════════════════════════════════════════════════════════════════

# Return a status icon for a slice status.
status_icon() {
  _s="$1"
  case "$_s" in
    complete)  printf '%b' "${C_GREEN}+${C_RESET}" ;;
    running)   printf '%b' "${C_CYAN}*${C_RESET}" ;;
    pending)   printf '%b' "${C_DIM}-${C_RESET}" ;;
    failed|stuck|repair_limit|aborted|config_error|gh_unavailable|timeout|max_*) printf '%b' "${C_RED}!${C_RESET}" ;;
    *)         printf '%b' "${C_DIM}?${C_RESET}" ;;
  esac
}

# ═══════════════════════════════════════════════════════════════════
# Table drawing
# ═══════════════════════════════════════════════════════════════════

# Pad a cell value to a given width, accounting for ANSI escape sequences.
# Args: $1 = value (may contain ANSI), $2 = target width
padded_cell() {
  _val="$1"
  _width="$2"

  # Strip ANSI codes to measure visible length (POSIX-compatible)
  _esc="$(printf '\033')"
  _visible="$(printf '%b' "$_val" | sed "s/${_esc}\[[0-9;]*m//g" 2>/dev/null || printf '%b' "$_val")"
  _vis_len="${#_visible}"
  _pad_needed=$((_width - _vis_len))

  printf '%b' "$_val"
  if [ "$_pad_needed" -gt 0 ]; then
    printf "%${_pad_needed}s" ""
  fi
}

# Column widths
COL_SLICE=20
COL_STATUS=10
COL_PHASE=14
COL_CYCLE=5
COL_TIME=8
COL_DETAIL=14
COL_TOTAL=$((COL_SLICE + COL_STATUS + COL_PHASE + COL_CYCLE + COL_TIME + COL_DETAIL + 17))

print_table_header() {
  _line="$(printf '%*s' "$COL_TOTAL" '' | tr ' ' '-')"
  printf '%b\n' "${C_BOLD}"
  printf '| %-*s | %-*s | %-*s | %-*s | %-*s | %-*s |\n' \
    "$COL_SLICE" "Slice" \
    "$COL_STATUS" "Status" \
    "$COL_PHASE" "Phase" \
    "$COL_CYCLE" "Cyc" \
    "$COL_TIME" "Time" \
    "$COL_DETAIL" "Detail"
  printf '%b' "${C_RESET}"
  printf '|-%s-|-%s-|-%s-|-%s-|-%s-|-%s-|\n' \
    "$(printf '%*s' "$COL_SLICE" '' | tr ' ' '-')" \
    "$(printf '%*s' "$COL_STATUS" '' | tr ' ' '-')" \
    "$(printf '%*s' "$COL_PHASE" '' | tr ' ' '-')" \
    "$(printf '%*s' "$COL_CYCLE" '' | tr ' ' '-')" \
    "$(printf '%*s' "$COL_TIME" '' | tr ' ' '-')" \
    "$(printf '%*s' "$COL_DETAIL" '' | tr ' ' '-')"
}

# Print a table row.
# Args: $1=slice, $2=status, $3=phase(ANSI), $4=cycle, $5=time, $6=detail(ANSI)
print_table_row() {
  _slice="$1"
  _status="$2"
  _phase="$3"
  _cycle="$4"
  _time="$5"
  _detail="$6"

  # Truncate slice name if too long
  if [ "${#_slice}" -gt "$COL_SLICE" ]; then
    _slice="$(echo "$_slice" | cut -c1-$((COL_SLICE - 2))).."
  fi

  printf '| %-*s | ' "$COL_SLICE" "$_slice"
  padded_cell "$(status_icon "$_status") $_status" "$COL_STATUS"
  printf ' | '
  padded_cell "$_phase" "$COL_PHASE"
  printf ' | %*s | %-*s | ' "$COL_CYCLE" "$_cycle" "$COL_TIME" "$_time"
  padded_cell "$_detail" "$COL_DETAIL"
  printf ' |\n'
}

print_table_footer() {
  printf '|-%s-|-%s-|-%s-|-%s-|-%s-|-%s-|\n' \
    "$(printf '%*s' "$COL_SLICE" '' | tr ' ' '-')" \
    "$(printf '%*s' "$COL_STATUS" '' | tr ' ' '-')" \
    "$(printf '%*s' "$COL_PHASE" '' | tr ' ' '-')" \
    "$(printf '%*s' "$COL_CYCLE" '' | tr ' ' '-')" \
    "$(printf '%*s' "$COL_TIME" '' | tr ' ' '-')" \
    "$(printf '%*s' "$COL_DETAIL" '' | tr ' ' '-')"
}

# ═══════════════════════════════════════════════════════════════════
# Progress bar
# ═══════════════════════════════════════════════════════════════════

# Render a progress bar.
# Args: $1 = completed, $2 = total, $3 = bar_width (default 20)
render_progress_bar() {
  _done="$1"
  _total="$2"
  _bar_width="${3:-20}"

  if [ "$_total" -eq 0 ]; then
    printf 'No slices'
    return
  fi

  _pct=$(( (_done * 100) / _total ))
  _filled=$(( (_done * _bar_width) / _total ))
  _empty=$((_bar_width - _filled))

  printf '%b' "${C_GREEN}"
  _i=0
  while [ "$_i" -lt "$_filled" ]; do
    printf '#'
    _i=$((_i + 1))
  done
  printf '%b' "${C_DIM}"
  _i=0
  while [ "$_i" -lt "$_empty" ]; do
    printf '.'
    _i=$((_i + 1))
  done
  printf '%b' "${C_RESET}"

  printf ' %d%% (%d/%d)' "$_pct" "$_done" "$_total"
}

# ═══════════════════════════════════════════════════════════════════
# ETA estimation
# ═══════════════════════════════════════════════════════════════════

# Estimate remaining time based on completed slice durations.
# Args: $1 = completed_count, $2 = remaining_count, $3 = total_elapsed_for_completed (seconds)
estimate_eta() {
  _comp="$1"
  _remaining="$2"
  _total_elapsed="$3"

  if [ "$_comp" -eq 0 ] || [ "$_remaining" -eq 0 ]; then
    echo "—"
    return
  fi

  _avg=$((_total_elapsed / _comp))
  _eta=$((_avg * _remaining))
  printf '~%s' "$(format_duration "$_eta")"
}

# ═══════════════════════════════════════════════════════════════════
# Renderers
# ═══════════════════════════════════════════════════════════════════

# Render rich table output.
# Args: $1 = ORCH_STATE directory
_render_table() {
  _orch_dir="$1"
  _wt_base="$2"
  _now="$(now_epoch)"

  printf '%b\n' "${C_BOLD}=== Ralph Pipeline Status ===${C_RESET}"
  echo ""

  # Read orchestrator state
  _orch_file="${_orch_dir}/orchestrator.json"
  if [ ! -f "$_orch_file" ]; then
    echo "No active orchestrator state found."
    echo "Run 'ralph run --plan <dir> --unified-pr' to start."
    return
  fi

  _orch_plan="$(jq -r '.plan // "unknown"' "$_orch_file" 2>/dev/null || echo "?")"
  _orch_status="$(jq -r '.status // "unknown"' "$_orch_file" 2>/dev/null || echo "?")"
  _orch_started="$(jq -r '.started // ""' "$_orch_file" 2>/dev/null || echo "")"

  _elapsed_str="—"
  if [ -n "$_orch_started" ]; then
    _start_epoch="$(iso_to_epoch "$_orch_started")"
    if [ "$_start_epoch" -gt 0 ]; then
      _elapsed=$((_now - _start_epoch))
      _elapsed_str="$(format_duration "$_elapsed")"
    fi
  fi

  printf 'Plan:    %s\n' "$_orch_plan"
  _status_color="$C_YELLOW"
  case "$_orch_status" in
    running)  _status_color="$C_CYAN" ;;
    complete) _status_color="$C_GREEN" ;;
  esac
  printf 'Status:  %b%-8s%b  Elapsed: %s\n' \
    "$_status_color" "$_orch_status" "$C_RESET" "$_elapsed_str"
  echo ""

  # Collect slice data
  _completed=0
  _total=0
  _completed_elapsed=0

  print_table_header

  for sf in "${_orch_dir}"/slice-*.status; do
    [ -f "$sf" ] || continue
    _name="$(basename "$sf" | sed 's/^slice-//;s/\.status$//')"
    _ss="$(cat "$sf")"
    _total=$((_total + 1))

    # Read checkpoint from worktree
    _ckpt="${_wt_base}/${_name}/.harness/state/pipeline/checkpoint.json"
    _phase="unknown"
    _cycle="—"
    _time_str="—"
    _detail=""
    _start_epoch=0

    if [ -f "$_ckpt" ]; then
      _phase="$(jq -r '.phase // "unknown"' "$_ckpt" 2>/dev/null || echo "unknown")"
      _cycle="$(jq -r '.inner_cycle // 0' "$_ckpt" 2>/dev/null || echo "0")"
      _test_result="$(jq -r '.last_test_result // ""' "$_ckpt" 2>/dev/null || echo "")"
      _pr_url="$(jq -r '.pr_url // ""' "$_ckpt" 2>/dev/null || echo "")"
      _started_ts="$(jq -r '(.phase_transitions[0].timestamp) // ""' "$_ckpt" 2>/dev/null || echo "")"

      [ "$_cycle" = "0" ] && _cycle="—"

      if [ -n "$_started_ts" ]; then
        _start_epoch="$(iso_to_epoch "$_started_ts")"
        if [ "$_start_epoch" -gt 0 ]; then
          _dur=$((_now - _start_epoch))
          _time_str="$(format_duration "$_dur")"
          if [ "$_ss" = "complete" ]; then
            _completed_elapsed=$((_completed_elapsed + _dur))
          fi
        fi
      fi

      # Build detail column
      if [ -n "$_pr_url" ] && [ "$_pr_url" != "null" ]; then
        _pr_num="$(printf '%s' "$_pr_url" | sed 's/.*\/\([0-9][0-9]*\)$/\1/' 2>/dev/null || echo "")"
        if [ -n "$_pr_num" ]; then
          _detail="PR #${_pr_num}"
        fi
      elif [ "$_ss" = "failed" ] || [ "$_ss" = "stuck" ] || [ "$_ss" = "repair_limit" ] || [ "$_ss" = "timeout" ]; then
        _detail="$_ss"
      elif [ -n "$_test_result" ] && [ "$_test_result" = "fail" ]; then
        _detail="test fail"
      fi
    else
      # No checkpoint — use status file only
      case "$_ss" in
        pending) _phase="waiting" ;;
        *) _phase="$_ss" ;;
      esac
    fi

    if [ "$_ss" = "complete" ]; then
      _completed=$((_completed + 1))
    fi

    _phase_display="$(resolve_display_phase "$_ss" "$_phase")"
    print_table_row "$_name" "$_ss" "$_phase_display" "$_cycle" "$_time_str" "$_detail"
  done

  print_table_footer
  echo ""

  # Progress bar
  _remaining=$((_total - _completed))
  printf 'Progress: '
  render_progress_bar "$_completed" "$_total"

  # ETA
  _eta="$(estimate_eta "$_completed" "$_remaining" "$_completed_elapsed")"
  if [ "$_eta" != "—" ]; then
    printf '  ETA: %s' "$_eta"
  fi
  echo ""

  # Worktrees
  _wt_count=0
  _wt_count="$(git worktree list 2>/dev/null | grep -c "${_wt_base}" 2>/dev/null)" || _wt_count=0
  if [ "$_wt_count" -gt 0 ]; then
    echo ""
    printf '%bActive worktrees: %d%b\n' "$C_DIM" "$_wt_count" "$C_RESET"
  fi
}

# Render JSON output.
# Args: $1 = ORCH_STATE directory
_render_json() {
  _orch_dir="$1"
  _wt_base="$2"
  _now="$(now_epoch)"

  _orch_file="${_orch_dir}/orchestrator.json"
  if [ ! -f "$_orch_file" ]; then
    printf '{"error":"no_active_orchestrator","message":"No active orchestrator state found."}\n'
    return
  fi

  _orch_plan="$(jq -r '.plan // "unknown"' "$_orch_file" 2>/dev/null || echo "unknown")"
  _orch_status="$(jq -r '.status // "unknown"' "$_orch_file" 2>/dev/null || echo "unknown")"
  _orch_started="$(jq -r '.started // ""' "$_orch_file" 2>/dev/null || echo "")"

  _elapsed=0
  if [ -n "$_orch_started" ]; then
    _start_epoch="$(iso_to_epoch "$_orch_started")"
    if [ "$_start_epoch" -gt 0 ]; then
      _elapsed=$((_now - _start_epoch))
    fi
  fi

  # Build slices array using jq for safe JSON construction
  _slices_json="[]"
  _completed=0
  _total=0

  for sf in "${_orch_dir}"/slice-*.status; do
    [ -f "$sf" ] || continue
    _name="$(basename "$sf" | sed 's/^slice-//;s/\.status$//')"
    _ss="$(cat "$sf")"
    _total=$((_total + 1))

    _phase="unknown"
    _cycle=0
    _slice_elapsed=0
    _test_result=""
    _pr_url=""

    _ckpt="${_wt_base}/${_name}/.harness/state/pipeline/checkpoint.json"
    if [ -f "$_ckpt" ]; then
      _phase="$(jq -r '.phase // "unknown"' "$_ckpt" 2>/dev/null || echo "unknown")"
      _cycle="$(jq -r '.inner_cycle // 0' "$_ckpt" 2>/dev/null || echo "0")"
      _test_result="$(jq -r '.last_test_result // ""' "$_ckpt" 2>/dev/null || echo "")"
      _pr_url="$(jq -r '.pr_url // ""' "$_ckpt" 2>/dev/null || echo "")"
      _started_ts="$(jq -r '(.phase_transitions[0].timestamp) // ""' "$_ckpt" 2>/dev/null || echo "")"

      if [ -n "$_started_ts" ]; then
        _start_epoch="$(iso_to_epoch "$_started_ts")"
        if [ "$_start_epoch" -gt 0 ]; then
          _slice_elapsed=$((_now - _start_epoch))
        fi
      fi
    fi

    [ "$_ss" = "complete" ] && _completed=$((_completed + 1))

    _slices_json="$(printf '%s' "$_slices_json" | jq \
      --arg name "$_name" \
      --arg status "$_ss" \
      --arg phase "$_phase" \
      --argjson cycle "$_cycle" \
      --argjson elapsed "$_slice_elapsed" \
      --arg test_result "$_test_result" \
      --arg pr_url "$_pr_url" \
      '. += [{name:$name,status:$status,phase:$phase,cycle:$cycle,elapsed_seconds:$elapsed,test_result:$test_result,pr_url:$pr_url}]')"
  done

  _pct=0
  [ "$_total" -gt 0 ] && _pct=$(( (_completed * 100) / _total ))

  jq -n \
    --arg plan "$_orch_plan" \
    --arg status "$_orch_status" \
    --argjson elapsed "$_elapsed" \
    --argjson slices "$_slices_json" \
    --argjson completed "$_completed" \
    --argjson total "$_total" \
    --argjson pct "$_pct" \
    '{plan:$plan,status:$status,elapsed_seconds:$elapsed,slices:$slices,progress:{completed:$completed,total:$total,percent:$pct}}'
}
