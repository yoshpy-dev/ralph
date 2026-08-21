#!/usr/bin/env sh
set -eu

HOOK_DIR="$(cd "$(dirname "$0")" && pwd)"
. "$HOOK_DIR/lib_json.sh"

payload="$(cat | tr '\n' ' ')"
# Claude Code PostToolUse payloads nest the target path under tool_input.
file_path="$(extract_json_field "$payload" "tool_input.file_path")"

# ralph-dispatch.sh (and the .codex/hooks.json entry that invokes it) cd to
# the git root before running this script, so $PWD here is the repo root
# in production and test fixtures alike (tests scope cwd per case to
# sandbox .harness/state writes). Used below to turn a cwd-resolved
# apply_patch path back into a readable root-relative log entry.
dispatch_root="$PWD"

mkdir -p .harness/state
: > .harness/state/needs-verify

# Reset consecutive failure counter on successful tool use
printf '0\n' > .harness/state/tool_failures.count

# Collect every path this tool call touched. Claude Code's Edit/Write/
# MultiEdit payloads carry a single tool_input.file_path. Codex's
# apply_patch payloads carry the patch body in tool_input.command instead
# and have no file_path -- derive the touched paths from the
# "*** Add/Update/Delete File:" and "*** Move to:" envelope lines in that
# patch body.
edited_paths=""
session_cwd=""
if [ -n "$file_path" ]; then
  edited_paths="$file_path"
elif command -v jq >/dev/null 2>&1; then
  # jq is required for this branch: the sed fallback in lib_json.sh only
  # matches a single-line quoted value and cannot reliably decode a
  # JSON-escaped multi-line patch body (an embedded quote in the diff
  # truncates the match early). Without jq, an apply_patch payload falls
  # through with no derived paths -- the same no-op this hook already had
  # for Codex edits before apply_patch support was added (now observable:
  # see the jq-missing marker below).
  #
  # apply_patch envelope paths are relative to the Codex SESSION's cwd
  # (the payload's top-level "cwd" field), not to this hook's own cwd (the
  # git root, since dispatch_root above). Captured here so the classify/log
  # loop below can resolve a relative envelope path before it is written.
  session_cwd="$(printf '%s' "$payload" | jq -r '.cwd // empty' 2>/dev/null || true)"
  patch_text="$(printf '%s' "$payload" | jq -r '.tool_input.command // empty' 2>/dev/null || true)"
  if [ -n "$patch_text" ]; then
    edited_paths="$(printf '%s\n' "$patch_text" | sed -n \
      -e 's/^\*\*\* Add File: \(.*\)$/\1/p' \
      -e 's/^\*\*\* Update File: \(.*\)$/\1/p' \
      -e 's/^\*\*\* Delete File: \(.*\)$/\1/p' \
      -e 's/^\*\*\* Move to: \(.*\)$/\1/p')"
  fi
else
  # jq missing and no tool_input.file_path: mirror check_mojibake.sh's
  # fail-safe-but-observable convention (this hook only classifies/logs,
  # so the failure mode is a silent no-op rather than a crash either way,
  # but a distinct marker + stderr note makes the reduced function visible
  # to the operator instead of a source comment nobody reads at runtime).
  # .harness/state already exists (mkdir -p above runs unconditionally).
  : > .harness/state/post-edit-verify-jq-missing 2>/dev/null || true
  printf 'post_edit_verify.sh: jq not found; skipping apply_patch path derivation (install jq to enable).\n' >&2
fi

# Log every derived path, then classify the whole set into one message.
# A code-class path anywhere in the set wins over a doc-class path, which
# wins over the skip list -- this mirrors the single-path precedence the
# case statement below encoded back when only one path could ever be
# derived per call.
class="skip"
if [ -n "$edited_paths" ]; then
  while IFS= read -r p; do
    [ -z "$p" ] && continue
    # Resolve a relative apply_patch envelope path against the session
    # cwd captured above; an absolute path (the Claude Code case, or an
    # apply_patch payload whose "cwd" was omitted) passes through
    # unchanged, and an empty session_cwd leaves it relative (fallback:
    # logged as given, matching pre-fix behavior).
    case "$p" in
      /*) resolved="$p" ;;
      *)
        if [ -n "$session_cwd" ]; then
          resolved="$session_cwd/$p"
        else
          resolved="$p"
        fi
        ;;
    esac
    # Prefer a root-relative form in the log for readability: strip
    # dispatch_root when the resolved path is under it, keep the absolute
    # form otherwise.
    logged="$resolved"
    case "$resolved" in
      "$dispatch_root"/*) logged="${resolved#"$dispatch_root"/}" ;;
    esac
    printf '%s\n' "$logged" >> .harness/state/edited-files.log
    case "$logged" in
      # Instruction and documentation files. Both a nested form (leading
      # "/") and a bare root-relative form are matched -- Codex apply_patch
      # envelope paths and the root-relative log form above never carry a
      # leading "/", so the nested-only patterns previously missed them
      # (C3-M2).
      *"/AGENTS.md"|*"AGENTS.md"|*"/CLAUDE.md"|*"CLAUDE.md"|*"/docs/"*|docs/*|*"/.claude/rules/"*|.claude/rules/*)
        if [ "$class" != "code" ]; then class="doc"; fi
        ;;
      # Known non-code files: skip verify reminder
      *.md|*.txt|*.json|*.yaml|*.yml|*.toml|*.ini|*.cfg|*.conf|*.lock|*.csv)
        ;;
      # Everything else is treated as code
      *)
        class="code"
        ;;
    esac
  done <<EOF
$edited_paths
EOF
fi

msg=""
case "$class" in
  doc)
    msg="Instruction or documentation files changed. Keep plans, docs, and implementation aligned, and record evidence for behavior changes."
    ;;
  code)
    msg="Code file edited. Run ./scripts/run-verify.sh before claiming done. Save evidence to docs/evidence/."
    ;;
esac

if [ -n "$msg" ]; then
  escaped="$(printf '%s' "$msg" | sed 's/"/\\\"/g')"
  printf '{"hookSpecificOutput":{"hookEventName":"PostToolUse","additionalContext":"%s"}}\n' "$escaped"
fi

exit 0
