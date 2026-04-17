#!/usr/bin/env sh
set -eu

status=0

fail() {
  echo "FAIL: $1"
  status=1
}

# --- Required files ---
required_files="
AGENTS.md
CLAUDE.md
.claude/settings.json
scripts/run-verify.sh
scripts/commit-msg-guard.sh
"

for file in $required_files; do
  if [ ! -e "$file" ]; then
    fail "Missing required file: $file"
  fi
done

# --- Shell scripts must be executable ---
for dir in .claude/hooks scripts; do
  if [ -d "$dir" ]; then
    find "$dir" -type f -name '*.sh' | while IFS= read -r script; do
      if [ ! -x "$script" ]; then
        fail "Script is not executable: $script"
      fi
    done
  fi
done
if [ -d packs ]; then
  find packs -type f -name '*.sh' | while IFS= read -r script; do
    if [ ! -x "$script" ]; then
      fail "Script is not executable: $script"
    fi
  done
fi

# --- Every skill directory must have a SKILL.md ---
if [ -d .claude/skills ]; then
  find .claude/skills -mindepth 1 -maxdepth 1 -type d | while IFS= read -r skill_dir; do
    if [ ! -f "$skill_dir/SKILL.md" ]; then
      fail "Skill missing SKILL.md: $skill_dir"
    fi
  done
fi

# --- Every agent file must have required frontmatter fields ---
if [ -d .claude/agents ]; then
  find .claude/agents -type f -name '*.md' | while IFS= read -r agent_file; do
    for field in name description tools; do
      if ! grep -q "^${field}:" "$agent_file"; then
        fail "Agent missing '$field' field: $agent_file"
      fi
    done
  done
fi

# --- Settings file must reference only existing hook scripts ---
if [ -f .claude/settings.json ]; then
  grep -o '"\./.claude/hooks/[^"]*"' .claude/settings.json 2>/dev/null | tr -d '"' | while IFS= read -r hook_path; do
    if [ ! -f "$hook_path" ]; then
      fail "Settings references missing hook: $hook_path"
    fi
  done
fi

if [ "$status" -eq 0 ]; then
  echo "Template structure check passed."
fi

exit "$status"
