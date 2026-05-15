#!/usr/bin/env sh
set -eu

# Regression coverage for wrapper/orchestrator option parity.

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
RALPH="${PROJECT_ROOT}/scripts/ralph"

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

work_dir="$(mktemp -d "${TMPDIR:-/tmp}/ralph-run-options.XXXXXX")"
trap 'rm -rf "$work_dir"' EXIT HUP INT TERM

cd "$work_dir"
git init -q -b main
git config user.email test@example.com
git config user.name "Test User"
printf 'seed\n' > README.md
git add README.md
git commit -q -m 'chore: seed'

plan_dir="docs/plans/active/2026-05-14-run-options"
mkdir -p "$plan_dir"
cat > "${plan_dir}/_manifest.md" <<'MD'
# Run options

- Type: fix
- Related issue: N/A
MD
cat > "${plan_dir}/slice-1-options.md" <<'MD'
# Slice 1

- Objective: Option parsing
- Dependencies: none
- Affected files: scripts/ralph
MD

echo "==> ralph run option tests"
"$RALPH" run --plan "$plan_dir" --preflight --dry-run > preflight.log 2>&1
check_contains "preflight flag is accepted" "Preflight only: 1" preflight.log
check_contains "preflight dry-run does not invoke pipeline" "[DRY RUN] Preflight parsed 1 slice(s)" preflight.log

"$RALPH" run --plan "$plan_dir" --resume --dry-run > resume.log 2>&1
check_contains "resume flag is accepted" "Resume: 1" resume.log
check_contains "resume dry-run parses plan" "[DRY RUN] Plan parsed successfully" resume.log

"$RALPH" run --plan "$plan_dir" --pr-strategy stacked --dry-run > strategy.log 2>&1
check_contains "pr-strategy flag is accepted" "[DRY RUN] PR strategy: stacked" strategy.log

printf '\nralph run option tests: %d passed, %d failed\n' "$pass" "$fail"
[ "$fail" -eq 0 ]
