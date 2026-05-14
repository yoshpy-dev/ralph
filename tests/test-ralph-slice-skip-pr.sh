#!/usr/bin/env sh
set -eu

# Ensure Ralph Loop slice pipelines never enter their own PR creation phase.
# The orchestrator creates the unified PR after merge/integration checks.

pass=0
fail=0

check_slice_invocation() {
  file="$1"
  if awk '
    /ralph-pipeline\.sh" \\/ { in_call = 1; seen_skip = 0; lines = 0 }
    in_call {
      lines++
      if ($0 ~ /--skip-pr/) {
        seen_skip = 1
      }
      if ($0 ~ /2>&1/ || lines > 8) {
        if (seen_skip) {
          found = 1
        }
        in_call = 0
      }
    }
    END { exit found ? 0 : 1 }
  ' "$file"; then
    printf '  PASS: %s run_slice passes --skip-pr\n' "$file"
    pass=$((pass + 1))
  else
    printf '  FAIL: %s run_slice does not pass --skip-pr\n' "$file"
    fail=$((fail + 1))
  fi
}

check_retry_invocation() {
  file="$1"
  if grep -Fq '"${SCRIPT_DIR}/ralph-pipeline.sh" --resume --skip-pr' "$file"; then
    printf '  PASS: %s retry passes --skip-pr\n' "$file"
    pass=$((pass + 1))
  else
    printf '  FAIL: %s retry does not pass --skip-pr\n' "$file"
    fail=$((fail + 1))
  fi
}

echo "==> Ralph Loop slice skip-PR tests"
check_slice_invocation scripts/ralph-orchestrator.sh
check_slice_invocation templates/base/scripts/ralph-orchestrator.sh
check_retry_invocation scripts/ralph
check_retry_invocation templates/base/scripts/ralph

printf '\nRalph Loop slice skip-PR tests: %d passed, %d failed\n' "$pass" "$fail"
[ "$fail" -eq 0 ]
