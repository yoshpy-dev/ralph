#!/usr/bin/env sh
# ralph-dispatch.sh — single settings.json entry per hook event, fanning out
# to layered ".d" directories.
#
# Usage: ralph-dispatch.sh <event>
#
# Runs, in lexicographic order, every executable "*.sh" file found in:
#   1. ./.claude/hooks/<event>.d/           (core, committed)
#   2. ./.ralph/local/hooks/<event>.d/      (downstream local, committed)
#   3. ./.claude/hooks/local/<event>.d/     (downstream local, gitignored)
# Missing directories are skipped silently. Non-executable *.sh files are
# skipped silently (lets a drop-in be disabled with chmod -x without
# removing it).
#
# stdin handling:
#   The hook payload on stdin is buffered once to a temp file, then that
#   file is fed to every script's stdin in turn. Each script sees the full
#   original payload, not a partially-consumed stream.
#
# Per-script semantics (in the order scripts run):
#   a. Non-zero exit: emit that script's stdout verbatim, exit with that
#      exit code, and run no further scripts. First failure wins — this
#      matches the pre-dispatcher behavior where a single blocking hook's
#      non-zero exit stopped Claude Code from proceeding.
#   b. Decision JSON: if a script's stdout is valid JSON containing a
#      decision field (`.hookSpecificOutput.permissionDecision`, top-level
#      `.decision`, or `.continue == false`), emit that JSON verbatim, exit
#      0, and run no further scripts. First decision wins — a later script
#      cannot override an earlier allow/deny/ask.
#   c. Otherwise, accumulate the script's stdout for merging. After all
#      scripts have run: if exactly one script produced output, emit it
#      verbatim (byte-exact passthrough — preserves the pre-dispatcher
#      single-hook behavior). If more than one script produced output,
#      merge: JSON outputs contribute their
#      `.hookSpecificOutput.additionalContext` (and the first non-empty
#      `.hookSpecificOutput.hookEventName` seen), plain-text outputs
#      contribute as-is; emit one JSON object:
#      {"hookSpecificOutput":{"hookEventName":"<event>","additionalContext":"<parts joined with \n\n>"}}
#   d. jq unavailable: no JSON introspection is possible, so (b) is skipped
#      entirely (a script emitting decision JSON is treated as case (c)
#      plain text — the dispatcher cannot special-case it without jq).
#      Single-output passthrough in (c) still applies byte-exact. Multi-
#      output merging in (c) falls back to plain-text concatenation
#      (scripts' raw stdout joined with a blank line) instead of building
#      a JSON envelope. Install jq to restore full JSON-aware merging.
#
# jq availability is detected directly below via `command -v jq`.

set -eu

event="${1:-}"
if [ -z "$event" ]; then
  echo "ralph-dispatch.sh: missing <event> argument" >&2
  exit 2
fi
case "$event" in
  *[!A-Za-z]*)
    echo "ralph-dispatch.sh: invalid <event> argument: $event" >&2
    exit 2
    ;;
esac

have_jq=0
if command -v jq >/dev/null 2>&1; then
  have_jq=1
fi

stdin_buf=""
out_tmp=""
merged_tmp=""
first_output=""
child=""
cleanup() {
  # Guard every removal so a signal that fires before a variable is
  # assigned can't reconstruct a relative path (e.g. an empty out_tmp
  # would otherwise make "$out_tmp.first" evaluate to the bare ".first",
  # a cwd-relative path outside this script's temp namespace). Each
  # branch is an "if", not "[ -n ... ] && rm -f ...", so a false guard
  # still returns 0 — chaining "&&" would make cleanup's own exit status
  # the false test's status (1), and the EXIT trap uses that in place of
  # the real exit code that triggered it.
  if [ -n "$stdin_buf" ]; then rm -f "$stdin_buf"; fi
  if [ -n "$out_tmp" ]; then rm -f "$out_tmp"; fi
  if [ -n "$merged_tmp" ]; then rm -f "$merged_tmp"; fi
  if [ -n "$first_output" ]; then rm -f "$first_output"; fi
}
# kill_child terminates the currently-running hook script's child process (if
# any) before the signal handlers below hand off to cleanup+exit. Without
# this, a TERM/INT/HUP delivered to the dispatcher kills the dispatcher but
# leaves an in-flight hook script running detached, free to keep mutating the
# repository after the dispatcher itself has reported termination.
kill_child() {
  if [ -n "$child" ]; then
    kill -TERM "$child" 2>/dev/null
    wait "$child" 2>/dev/null
  fi
}
# The EXIT trap alone does not run when a fatal signal kills the process
# under POSIX sh; each signal trap below must clean up and then terminate
# explicitly (a bare "trap cleanup INT TERM HUP" would resume the script
# after the trap action instead of exiting it). kill_child runs first so the
# in-flight hook script (if any) is terminated before temp files are removed.
trap cleanup EXIT
trap 'kill_child; cleanup; exit 130' INT
trap 'kill_child; cleanup; exit 143' TERM
trap 'kill_child; cleanup; exit 129' HUP

stdin_buf="$(mktemp "${TMPDIR:-/tmp}/ralph-dispatch-stdin.XXXXXX")"
cat > "$stdin_buf"

out_tmp="$(mktemp "${TMPDIR:-/tmp}/ralph-dispatch-out.XXXXXX")"
merged_tmp="$(mktemp "${TMPDIR:-/tmp}/ralph-dispatch-merged.XXXXXX")"

: > "$merged_tmp"
output_count=0

# Directories are searched in this fixed order; scripts run in
# lexicographic order within each directory.
dirs="./.claude/hooks/${event}.d ./.ralph/local/hooks/${event}.d ./.claude/hooks/local/${event}.d"

for d in $dirs; do
  [ -d "$d" ] || continue
  for script in "$d"/*.sh; do
    [ -e "$script" ] || continue
    [ -x "$script" ] || continue

    : > "$out_tmp"
    set +e
    "$script" < "$stdin_buf" > "$out_tmp" &
    child=$!
    wait "$child"
    rc=$?
    set -e
    child=""

    if [ "$rc" -ne 0 ]; then
      cat "$out_tmp"
      exit "$rc"
    fi

    if [ ! -s "$out_tmp" ]; then
      continue
    fi

    if [ "$have_jq" -eq 1 ]; then
      decision="$(jq -r '(.hookSpecificOutput.permissionDecision // .decision // empty), (if .continue == false then "stop" else empty end)' "$out_tmp" 2>/dev/null | sed -n '1p')"
      if [ -n "$decision" ]; then
        cat "$out_tmp"
        exit 0
      fi
    fi

    output_count=$((output_count + 1))
    if [ "$output_count" -eq 1 ]; then
      first_output="$out_tmp.first"
      cp "$out_tmp" "$first_output"
      chmod 600 "$first_output"
    fi

    if [ "$have_jq" -eq 1 ]; then
      ctx="$(jq -r '.hookSpecificOutput.additionalContext // empty' "$out_tmp" 2>/dev/null || true)"
      if [ -n "$ctx" ]; then
        printf '%s\n' "$ctx" >> "$merged_tmp"
      else
        cat "$out_tmp" >> "$merged_tmp"
        printf '\n' >> "$merged_tmp"
      fi
    else
      cat "$out_tmp" >> "$merged_tmp"
      printf '\n' >> "$merged_tmp"
    fi
    printf '\n' >> "$merged_tmp"
  done
done

if [ "$output_count" -eq 0 ]; then
  exit 0
fi

if [ "$output_count" -eq 1 ]; then
  cat "$first_output"
  rm -f "$first_output"
  exit 0
fi
rm -f "$first_output"

if [ "$have_jq" -eq 1 ]; then
  # Collapse consecutive blank-line-separated parts, then join with a
  # blank line for readability inside the JSON string, and JSON-escape
  # the result via jq itself (handles quotes, backslashes, newlines).
  merged_context="$(awk 'BEGIN{first=1} {if ($0=="" ) {blank=1; next} if (!first && blank) {print ""} print; first=0; blank=0}' "$merged_tmp")"
  jq -n --arg event "$event" --arg ctx "$merged_context" \
    '{hookSpecificOutput:{hookEventName:$event, additionalContext:$ctx}}'
else
  awk 'BEGIN{first=1} {if ($0=="" ) {blank=1; next} if (!first && blank) {print ""} print; first=0; blank=0}' "$merged_tmp"
fi

exit 0
