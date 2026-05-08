#!/usr/bin/env sh
set -eu

# Pipeline order sync checker
# Canonical: /self-review -> /verify -> /test -> /sync-docs -> /cross-review -> /pr
# Source of truth: .claude/rules/post-implementation-pipeline.md

CANONICAL=".claude/rules/post-implementation-pipeline.md"
status=0

fail() { echo "FAIL: $1"; status=1; }
ok()   { echo "[ok] $1"; }

# 1. Verify canonical source exists and contains the expected order
if [ ! -f "$CANONICAL" ]; then
  fail "Canonical source missing: $CANONICAL"
  exit 1
fi

if ! grep -q 'self-review.*verify.*test.*sync-docs.*cross-review' "$CANONICAL"; then
  fail "Canonical order not found in $CANONICAL"
  exit 1
fi
ok "Canonical source valid"

# 2. Check each referenced file
REFS="
.claude/skills/work/SKILL.md
.claude/skills/loop/SKILL.md
.claude/skills/cross-review/SKILL.md
.claude/rules/subagent-policy.md
CLAUDE.md
docs/quality/definition-of-done.md
README.md
AGENTS.md
"

for ref in $REFS; do
  [ -f "$ref" ] || { fail "Referenced file missing: $ref"; continue; }

  # Check that the file mentions all 5 pipeline steps
  missing=""
  for step in self-review verify test sync-docs cross-review; do
    if ! grep -qi "$step" "$ref" 2>/dev/null; then
      missing="${missing:+${missing}, }${step}"
    fi
  done

  if [ -n "$missing" ]; then
    fail "${ref}: missing pipeline step reference(s): ${missing}"
  else
    ok "${ref}: all pipeline steps referenced"
  fi
done

exit "$status"
