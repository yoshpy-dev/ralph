#!/usr/bin/env sh
set -eu

msg="Prompt reminder: for risky or multi-file work, refresh the plan first. Prefer verification, evidence, or a specialized subagent before asking the user for obvious next steps."
escaped="$(printf '%s' "$msg" | sed 's/"/\\\"/g')"
printf '{"hookSpecificOutput":{"hookEventName":"UserPromptSubmit","additionalContext":"%s"}}\n' "$escaped"
