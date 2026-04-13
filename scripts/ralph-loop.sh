#!/usr/bin/env sh
set -eu

# Ralph Loop orchestrator
# Runs `cat PROMPT.md | claude -p` in a loop with safety rails.
# State lives in .harness/state/loop/

LOOP_DIR=".harness/state/loop"
MAX_ITERATIONS=20
VERIFY=0
DRY_RUN=0

usage() {
  echo "Usage: $0 [--max-iterations N] [--verify] [--dry-run]"
  echo ""
  echo "Runs the Ralph Loop from ${LOOP_DIR}/PROMPT.md."
  echo ""
  echo "Options:"
  echo "  --max-iterations N  Maximum iterations (default: 20)"
  echo "  --verify            Run ./scripts/run-verify.sh after each iteration"
  echo "  --dry-run           Print what would run without executing claude"
  exit 1
}

while [ $# -gt 0 ]; do
  case "$1" in
    --max-iterations)
      shift
      MAX_ITERATIONS="${1:?--max-iterations requires a number}"
      ;;
    --verify)
      VERIFY=1
      ;;
    --dry-run)
      DRY_RUN=1
      ;;
    -h|--help)
      usage
      ;;
    *)
      echo "Unknown option: $1"
      usage
      ;;
  esac
  shift
done

# --- Pre-flight checks ---

if [ ! -f "${LOOP_DIR}/PROMPT.md" ]; then
  echo "Error: ${LOOP_DIR}/PROMPT.md not found."
  echo "Run ./scripts/ralph-loop-init.sh first."
  exit 1
fi

if [ "$DRY_RUN" -eq 0 ] && ! command -v claude >/dev/null 2>&1; then
  echo "Error: claude CLI not found in PATH."
  exit 1
fi

mkdir -p "${LOOP_DIR}"

# Initialize or read stuck counter
stuck_count=0
uncommitted_count=0
if [ -f "${LOOP_DIR}/stuck.count" ]; then
  stuck_count="$(cat "${LOOP_DIR}/stuck.count")"
fi

echo "running" > "${LOOP_DIR}/status"

iteration=0
start_ts="$(date -u '+%Y-%m-%dT%H:%M:%SZ')"

echo "=== Ralph Loop started ==="
echo "- Max iterations: ${MAX_ITERATIONS}"
echo "- Verify after each: ${VERIFY}"
echo "- Dry run: ${DRY_RUN}"
echo "- Start: ${start_ts}"
echo ""

while [ "$iteration" -lt "$MAX_ITERATIONS" ]; do
  iteration=$((iteration + 1))
  iter_padded="$(printf '%03d' "$iteration")"
  log_file="${LOOP_DIR}/iteration-${iter_padded}.log"
  iter_ts="$(date -u '+%Y-%m-%dT%H:%M:%SZ')"

  echo "--- Iteration ${iteration}/${MAX_ITERATIONS} [${iter_ts}] ---"

  # Capture git state before iteration
  diff_before=""
  if command -v git >/dev/null 2>&1; then
    diff_before="$(git diff HEAD 2>/dev/null || true)"
  fi

  if [ "$DRY_RUN" -eq 1 ]; then
    echo "[dry-run] Would run: cat ${LOOP_DIR}/PROMPT.md | claude -p"
    echo "[dry-run] Output would be saved to: ${log_file}"
    echo "[dry-run] iteration ${iteration} complete" > "$log_file"
  else
    cat "${LOOP_DIR}/PROMPT.md" | claude -p --model opus --effort max 2>&1 | tee "$log_file"
  fi

  # Check for completion signal
  if grep -q '<promise>COMPLETE</promise>' "$log_file" 2>/dev/null; then
    echo ""
    echo "=== Loop complete: agent signalled COMPLETE ==="
    echo "complete" > "${LOOP_DIR}/status"
    echo "0" > "${LOOP_DIR}/stuck.count"
    break
  fi

  # Check for abort signal
  if grep -q '<promise>ABORT</promise>' "$log_file" 2>/dev/null; then
    echo ""
    echo "=== Loop aborted: agent signalled ABORT ==="
    echo "aborted" > "${LOOP_DIR}/status"
    echo "0" > "${LOOP_DIR}/stuck.count"
    break
  fi

  # Stuck detection: compare git diff before and after
  if command -v git >/dev/null 2>&1; then
    diff_after="$(git diff HEAD 2>/dev/null || true)"
    if [ "$diff_before" = "$diff_after" ]; then
      stuck_count=$((stuck_count + 1))
      echo "Warning: no file changes detected (stuck count: ${stuck_count}/3)"
    else
      stuck_count=0
    fi
    printf '%s' "$stuck_count" > "${LOOP_DIR}/stuck.count"

    if [ "$stuck_count" -ge 3 ]; then
      echo ""
      echo "=== Loop stopped: stuck detected (3 consecutive iterations with no changes) ==="
      echo "stuck" > "${LOOP_DIR}/status"
      break
    fi
  fi

  # Check for uncommitted changes after each iteration
  if command -v git >/dev/null 2>&1; then
    uncommitted="$(git status --porcelain 2>/dev/null || true)"
    if [ -n "$uncommitted" ]; then
      uncommitted_count=$((uncommitted_count + 1))
      echo "> [orchestrator] Warning: uncommitted changes detected after iteration ${iteration} (total: ${uncommitted_count})"
      if [ -f "${LOOP_DIR}/progress.log" ]; then
        printf '\n> [orchestrator] Warning: uncommitted changes after iteration %d\n' "$iteration" >> "${LOOP_DIR}/progress.log"
      fi
    fi
  fi

  # Optional verification after each iteration
  if [ "$VERIFY" -eq 1 ] && [ "$DRY_RUN" -eq 0 ]; then
    echo "--- Running verification ---"
    if ! ./scripts/run-verify.sh; then
      echo "Warning: verification failed after iteration ${iteration}"
    fi
  fi
done

# Check if we hit max iterations
if [ "$iteration" -ge "$MAX_ITERATIONS" ]; then
  current_status="$(cat "${LOOP_DIR}/status" 2>/dev/null || echo "running")"
  if [ "$current_status" = "running" ]; then
    echo ""
    echo "=== Loop stopped: max iterations (${MAX_ITERATIONS}) reached ==="
    echo "max_iterations" > "${LOOP_DIR}/status"
  fi
fi

end_ts="$(date -u '+%Y-%m-%dT%H:%M:%SZ')"
final_status="$(cat "${LOOP_DIR}/status")"

echo ""
echo "=== Ralph Loop summary ==="
echo "- Iterations run: ${iteration}"
echo "- Final status: ${final_status}"
echo "- Started: ${start_ts}"
echo "- Ended: ${end_ts}"
echo "- Uncommitted warnings: ${uncommitted_count}"
echo "- Logs: ${LOOP_DIR}/iteration-*.log"
if [ "$uncommitted_count" -gt 0 ]; then
  echo ""
  echo "Warning: ${uncommitted_count} iteration(s) left uncommitted changes."
  echo "Review with: git status && git diff"
fi
