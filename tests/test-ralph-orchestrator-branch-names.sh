#!/usr/bin/env sh
set -eu

# Regression coverage for Ralph Loop slice branch naming. The orchestrator
# must create, report, and merge the same per-slice branch name.

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
ORCHESTRATOR="${PROJECT_ROOT}/scripts/ralph-orchestrator.sh"

pass=0
fail=0

check_contains() {
  label="$1"
  needle="$2"
  file="$3"
  if grep -Fq "$needle" "$file"; then
    printf '  PASS: %s\n' "$label"
    pass=$((pass + 1))
  else
    printf '  FAIL: %s\n' "$label"
    printf '    missing: %s\n' "$needle"
    fail=$((fail + 1))
  fi
}

check_not_contains() {
  label="$1"
  needle="$2"
  file="$3"
  if grep -Fq "$needle" "$file"; then
    printf '  FAIL: %s\n' "$label"
    printf '    unexpected: %s\n' "$needle"
    fail=$((fail + 1))
  else
    printf '  PASS: %s\n' "$label"
    pass=$((pass + 1))
  fi
}

work_dir="$(mktemp -d "${TMPDIR:-/tmp}/ralph-orch-branches.XXXXXX")"
trap 'rm -rf "$work_dir"' EXIT HUP INT TERM

cd "$work_dir"
git init -q -b main
git config user.email test@example.com
git config user.name "Test User"
printf 'seed\n' > README.md
git add README.md
git commit -q -m 'chore: seed'

plan_dir="docs/plans/active/2026-05-14-branch-sync"
mkdir -p "$plan_dir"
cat > "${plan_dir}/_manifest.md" <<'MD'
# Branch sync

- Type: fix
- Related issue: N/A

### Shared-file locklist
- AGENTS.md
MD

cat > "${plan_dir}/slice-1-api.md" <<'MD'
# Slice 1

- Objective: API slice
- Dependencies: none
- Affected files: internal/api.go
MD

"$ORCHESTRATOR" --plan "$plan_dir" --dry-run > output.log 2>&1

echo "==> Ralph orchestrator branch-name tests"
check_contains "dry-run reports generated slice branch" "branch fix/branch-sync-1-api" output.log
check_not_contains "dry-run no longer reports obsolete slice/ branch" "branch slice/2026-05-14-branch-sync/1-api" output.log

if grep -Fq '_slice_branch="slice/${PLAN_SLUG}/${s}"' "$ORCHESTRATOR"; then
  printf '  FAIL: integration merge still hard-codes obsolete slice/ branch\n'
  fail=$((fail + 1))
else
  printf '  PASS: integration merge no longer hard-codes obsolete slice/ branch\n'
  pass=$((pass + 1))
fi

printf '\nRalph orchestrator branch-name tests: %d passed, %d failed\n' "$pass" "$fail"
[ "$fail" -eq 0 ]
