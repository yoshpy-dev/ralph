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
.claude/settings.minimal.example.json
.claude/settings.advanced.example.json
docs/research/approach-comparison.md
docs/roadmap/harness-maturity-model.md
scripts/run-verify.sh
scripts/archive-plan.sh
"

for file in $required_files; do
  if [ ! -e "$file" ]; then
    fail "Missing required file: $file"
  fi
done

# --- Shell scripts must be executable ---
for script in $(find .claude/hooks packs scripts -type f -name '*.sh'); do
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

# --- Settings example files must reference only existing hook scripts ---
for settings_file in .claude/settings.minimal.example.json .claude/settings.advanced.example.json; do
  if [ ! -f "$settings_file" ]; then
    continue
  fi
  # Extract hook script paths from "command" fields
  grep -o '"\./.claude/hooks/[^"]*"' "$settings_file" 2>/dev/null | tr -d '"' | while IFS= read -r hook_path; do
    if [ ! -f "$hook_path" ]; then
      fail "Settings file $settings_file references missing hook: $hook_path"
    fi
  done
done

if [ "$status" -eq 0 ]; then
  echo "Template structure looks good."
fi

exit "$status"
