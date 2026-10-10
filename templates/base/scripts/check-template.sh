#!/usr/bin/env sh
set -eu

status=0

fail() {
  echo "FAIL: $1"
  status=1
}

# The find-driven loops below (scripts, skills, agents) read from a temp
# file instead of looping over `$(find ...)` directly, which word-splits
# on whitespace and breaks on paths containing spaces. The settings
# hook-reference check below reads from a temp file instead of piping into
# `while`, because a pipeline runs the loop body in a subshell, so `fail`'s
# status=1 set there would never reach this shell.
tmpdir="$(mktemp -d "${TMPDIR:-/tmp}/check-template.XXXXXX")"
cleanup() {
  rm -rf "$tmpdir"
}
trap cleanup EXIT
trap 'exit 130' INT
trap 'exit 143' TERM
trap 'exit 129' HUP

# --- Required files ---
# Files a `ralph init` scaffold does not receive are deliberately not
# listed here (for example, the ralph repository's own README and its
# docs/research/ and docs/roadmap/ notes).
# The five .claude/hooks/ entries are listed because pre_bash_guard.sh needs
# all of them: without one of its three .awk files, awk exits non-zero and
# the guard falls back to the previous guard's four substring rules without
# any message.
required_files="
AGENTS.md
CLAUDE.md
.claude/settings.json
.claude/hooks/pre_bash_guard.sh
.claude/hooks/lib_json.sh
.claude/hooks/pre_bash_guard_lex.awk
.claude/hooks/pre_bash_guard_commands.awk
.claude/hooks/pre_bash_guard_rules.awk
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
scripts/plan-visual.sh
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
# .claude/hooks/local/ is reserved for user-local (gitignored) hooks; its
# whole subtree is pruned (not just filtered out of the results), so an
# unreadable directory anywhere inside it never reaches find's own
# traversal and can't turn into the FAIL below.
# Only existing search roots are passed to find; a missing root is
# skipped without a message. An unreadable directory inside a root that
# does exist makes find exit non-zero, which is reported below instead
# of silenced.
set --
for root in .claude/hooks packs scripts; do
  [ -d "$root" ] && set -- "$@" "$root"
done
if [ "$#" -gt 0 ]; then
  rc=0
  find "$@" -path '.claude/hooks/local' -prune -o -type f -name '*.sh' -print > "$tmpdir/scripts.list" || rc=$?
  if [ "$rc" -ne 0 ]; then
    fail "Could not list scripts under $*: find exited with $rc"
  fi
else
  : > "$tmpdir/scripts.list"
fi
while IFS= read -r script; do
  if [ ! -x "$script" ]; then
    fail "Script is not executable: $script"
  fi
done < "$tmpdir/scripts.list"

# --- Every skill directory must have a SKILL.md ---
if [ -d .claude/skills ]; then
  rc=0
  find .claude/skills -mindepth 1 -maxdepth 1 -type d > "$tmpdir/skills.list" || rc=$?
  if [ "$rc" -ne 0 ]; then
    fail "Could not list skill directories under .claude/skills: find exited with $rc"
  fi
else
  : > "$tmpdir/skills.list"
fi
while IFS= read -r skill_dir; do
  if [ ! -f "$skill_dir/SKILL.md" ]; then
    fail "Skill missing SKILL.md: $skill_dir"
  fi
done < "$tmpdir/skills.list"

# --- Every agent file must have required frontmatter fields ---
if [ -d .claude/agents ]; then
  rc=0
  find .claude/agents -type f -name '*.md' > "$tmpdir/agents.list" || rc=$?
  if [ "$rc" -ne 0 ]; then
    fail "Could not list agent files under .claude/agents: find exited with $rc"
  fi
else
  : > "$tmpdir/agents.list"
fi
while IFS= read -r agent_file; do
  for field in name description tools; do
    if ! grep -q "^${field}:" "$agent_file"; then
      fail "Agent missing '$field' field: $agent_file"
    fi
  done
done < "$tmpdir/agents.list"

# --- Settings file must reference only existing hook scripts ---
if [ -f .claude/settings.json ]; then
  rc=0
  grep -o '"\./.claude/hooks/[^"]*"' .claude/settings.json > "$tmpdir/hooks.raw" || rc=$?
  if [ "$rc" -gt 1 ]; then
    fail "Could not read hook commands from .claude/settings.json: grep exited with $rc"
  fi
  # Settings commands are "./.claude/hooks/<file> <args...>"; keep only the
  # path (the first field) so an argument is never checked as a path, and
  # de-duplicate so a dispatcher referenced by several events produces
  # exactly one FAIL line when it is missing.
  tr -d '"' < "$tmpdir/hooks.raw" | cut -d ' ' -f 1 | sort -u > "$tmpdir/hooks.list"
  while IFS= read -r hook_path; do
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
