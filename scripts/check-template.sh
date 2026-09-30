#!/usr/bin/env sh
set -eu

status=0

fail() {
  echo "FAIL: $1"
  status=1
}

# All find-driven loops below read from a temp file instead of piping into
# `while` (a pipeline runs the loop body in a subshell, so `fail`'s
# status=1 never reaches this shell) or looping over `$(find ...)` directly
# (word-splits on whitespace, breaking on paths containing spaces).
tmpdir="$(mktemp -d "${TMPDIR:-/tmp}/check-template.XXXXXX")"
cleanup() {
  rm -rf "$tmpdir"
}
trap cleanup EXIT

# --- Required files ---
# README.md, docs/research/approach-comparison.md, and
# docs/roadmap/harness-maturity-model.md are meta-repo-only: scaffolded
# projects never receive them, so they are intentionally not listed here
# (issue #189).
required_files="
AGENTS.md
CLAUDE.md
.claude/settings.json
scripts/run-verify.sh
scripts/run-static-verify.sh
scripts/run-test.sh
scripts/detect-changed-languages.sh
scripts/detect-languages.sh
scripts/archive-plan.sh
scripts/branch-name.sh
scripts/ensure-pr-ready.sh
scripts/ensure-pr-title-prefix.sh
scripts/new-feature-plan.sh
scripts/codex-check.sh
scripts/ralph-config.sh
scripts/ralph-worktree.sh
scripts/xreview-helpers.sh
scripts/secret-scan.sh
scripts/secret-scan-branch.sh
scripts/pre-commit-secret-guard.sh
scripts/commit-msg-guard.sh
scripts/prepare-commit-msg-secret-guard.sh
scripts/pre-merge-commit-secret-guard.sh
scripts/check-template.sh
scripts/check-skill-sync.sh
"

for file in $required_files; do
  if [ ! -e "$file" ]; then
    fail "Missing required file: $file"
  fi
done

# --- Shell scripts must be executable ---
# .claude/hooks/local/ is reserved for user-local (gitignored) hooks; skip.
find .claude/hooks packs scripts -type f -name '*.sh' -not -path '.claude/hooks/local/*' \
  > "$tmpdir/scripts.list" 2>/dev/null || true
while IFS= read -r script; do
  if [ ! -x "$script" ]; then
    fail "Script is not executable: $script"
  fi
done < "$tmpdir/scripts.list"

# --- Every skill directory must have a SKILL.md ---
find .claude/skills -mindepth 1 -maxdepth 1 -type d > "$tmpdir/skills.list" 2>/dev/null || true
while IFS= read -r skill_dir; do
  if [ ! -f "$skill_dir/SKILL.md" ]; then
    fail "Skill missing SKILL.md: $skill_dir"
  fi
done < "$tmpdir/skills.list"

# --- Every agent file must have required frontmatter fields ---
find .claude/agents -type f -name '*.md' > "$tmpdir/agents.list" 2>/dev/null || true
while IFS= read -r agent_file; do
  for field in name description tools; do
    if ! grep -q "^${field}:" "$agent_file"; then
      fail "Agent missing '$field' field: $agent_file"
    fi
  done
done < "$tmpdir/agents.list"

# --- Settings file must reference only existing hook scripts ---
if [ -f .claude/settings.json ]; then
  grep -o '"\./.claude/hooks/[^"]*"' .claude/settings.json 2>/dev/null | tr -d '"' \
    > "$tmpdir/hooks.list" || true
  while IFS= read -r hook_cmd; do
    # Settings commands are "./.claude/hooks/<file> <args...>"; keep only
    # the path (the first word) so an argument is never checked as a path.
    hook_path=${hook_cmd%% *}
    if [ ! -f "$hook_path" ]; then
      fail "Settings file .claude/settings.json references missing hook: $hook_path"
    fi
  done < "$tmpdir/hooks.list"
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
