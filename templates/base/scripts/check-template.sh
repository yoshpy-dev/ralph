#!/usr/bin/env sh
set -eu

status=0

fail() {
  echo "FAIL: $1"
  status=1
}

# --- Required files ---
required_files="
README.md
AGENTS.md
CLAUDE.md
.claude/settings.json
docs/research/approach-comparison.md
docs/roadmap/harness-maturity-model.md
scripts/run-verify.sh
scripts/archive-plan.sh
scripts/branch-name.sh
scripts/ensure-pr-ready.sh
scripts/ensure-pr-title-prefix.sh
scripts/secret-scan.sh
scripts/pre-commit-secret-guard.sh
scripts/commit-msg-guard.sh
scripts/prepare-commit-msg-secret-guard.sh
scripts/pre-merge-commit-secret-guard.sh
"

for file in $required_files; do
  if [ ! -e "$file" ]; then
    fail "Missing required file: $file"
  fi
done

# --- Shell scripts must be executable ---
# .claude/hooks/local/ is reserved for user-local (gitignored) hooks; skip.
for script in $(find .claude/hooks packs scripts -type f -name '*.sh' -not -path '.claude/hooks/local/*'); do
  if [ ! -x "$script" ]; then
    fail "Script is not executable: $script"
  fi
done

# --- Every skill directory must have a SKILL.md ---
for skill_dir in $(find .claude/skills -mindepth 1 -maxdepth 1 -type d); do
  if [ ! -f "$skill_dir/SKILL.md" ]; then
    fail "Skill missing SKILL.md: $skill_dir"
  fi
done

# --- Every agent file must have required frontmatter fields ---
for agent_file in $(find .claude/agents -type f -name '*.md'); do
  for field in name description tools; do
    if ! grep -q "^${field}:" "$agent_file"; then
      fail "Agent missing '$field' field: $agent_file"
    fi
  done
done

# --- Settings file must reference only existing hook scripts ---
if [ -f .claude/settings.json ]; then
  grep -o '"\./.claude/hooks/[^"]*"' .claude/settings.json 2>/dev/null | tr -d '"' | while IFS= read -r hook_path; do
    if [ ! -f "$hook_path" ]; then
      fail "Settings file .claude/settings.json references missing hook: $hook_path"
    fi
  done
fi

# --- git secret hook installation check (local only) ---
if [ -d .git ] && [ "${CI:-}" != "true" ]; then
  for hook_spec in \
    "pre-commit:pre-commit-secret-guard" \
    "commit-msg:commit-msg-guard" \
    "prepare-commit-msg:prepare-commit-msg-secret-guard" \
    "pre-merge-commit:pre-merge-commit-secret-guard"
  do
    hook=${hook_spec%%:*}
    marker=${hook_spec#*:}
    hook_path=".git/hooks/$hook"
    if [ ! -f "$hook_path" ]; then
      fail "$hook hook not installed. Run: ralph upgrade"
    elif ! grep -Eq "ralph git hook wrapper|$marker" "$hook_path" 2>/dev/null; then
      fail "$hook hook exists but is not managed by ralph. Run: ralph upgrade to chain it, or keep it intentionally"
    fi
  done
fi

if [ "$status" -eq 0 ]; then
  echo "Template structure looks good."
fi

exit "$status"
