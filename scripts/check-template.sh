#!/usr/bin/env sh
set -eu

required_files="
README.md
AGENTS.md
CLAUDE.md
.claude/settings.minimal.example.json
.claude/settings.advanced.example.json
docs/research/approach-comparison.md
docs/roadmap/harness-maturity-model.md
scripts/run-verify.sh
"

for file in $required_files; do
  if [ ! -e "$file" ]; then
    echo "Missing required file: $file"
    exit 1
  fi
done

for script in $(find .claude/hooks packs scripts -type f -name '*.sh'); do
  if [ ! -x "$script" ]; then
    echo "Script is not executable: $script"
    exit 1
  fi
done

echo "Template structure looks good."
