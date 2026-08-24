#!/usr/bin/env bash
# test-hook-wiring.sh — regression guard for "moved hooks leave dangling
# references" and for "Codex hooks ship in a form that never fires".
#
# For BOTH the meta-repo root and templates/base/, parses every hook
# command out of .claude/settings.json (jq) and asserts the referenced
# script file exists relative to that surface's root and is executable.
# This is a durable gate against a future hooks reshuffle (e.g. the
# ralph-dispatch.sh / <event>.d migration this test was added alongside)
# silently leaving a settings.json entry pointing at a path that no longer
# exists.
#
# For Codex, .codex/hooks.json is the source of truth (inline
# .codex/config.toml `[[hooks.*]]` entries do not fire — see
# docs/evidence/). This script asserts: hooks.json exists at both surfaces
# and is byte-identical between them; hooks.json is valid JSON; its
# PostToolUse entry routes through ralph-dispatch.sh; neither hooks.json
# nor config.toml references any other hook script directly (guards
# against the legacy direct-call form being reintroduced in either
# representation); and config.toml carries no leftover [hooks]/[[hooks.*]]
# table now that hooks.json is authoritative.

set -eu

REPO_ROOT="$(cd "$(dirname "$0")/.." && pwd)"

if ! command -v jq >/dev/null 2>&1; then
  echo "FAIL: jq is required for this test" >&2
  exit 1
fi

pass=0
fail=0
results=()

record_pass() {
  results+=("PASS  $1")
  pass=$((pass + 1))
}
record_fail() {
  results+=("FAIL  $1")
  fail=$((fail + 1))
}

# strip_command_prefix <command> — hook commands are shell invocations like
# "./.claude/hooks/ralph-dispatch.sh SessionStart"; keep only the leading
# executable token.
strip_command_prefix() {
  printf '%s' "$1" | awk '{print $1}'
}

check_settings_json() {
  local label="$1" surface_root="$2"
  local settings="$surface_root/.claude/settings.json"

  if [ ! -f "$settings" ]; then
    record_fail "$label: .claude/settings.json not found at $settings"
    return
  fi

  local commands
  commands="$(jq -r '.hooks // {} | to_entries[] | .value[] | (.hooks // [])[] | select(.type == "command") | .command' "$settings" 2>/dev/null || true)"

  if [ -z "$commands" ]; then
    record_fail "$label: .claude/settings.json has no hook commands (expected at least one)"
    return
  fi

  local count=0
  while IFS= read -r cmd; do
    [ -z "$cmd" ] && continue
    count=$((count + 1))
    local exe rel_path
    exe="$(strip_command_prefix "$cmd")"
    case "$exe" in
      ./*) rel_path="${exe#./}" ;;
      *) rel_path="$exe" ;;
    esac
    local abs_path="$surface_root/$rel_path"
    if [ ! -f "$abs_path" ]; then
      record_fail "$label: settings.json command '$cmd' -> missing file $abs_path"
      continue
    fi
    if [ ! -x "$abs_path" ]; then
      record_fail "$label: settings.json command '$cmd' -> $abs_path exists but is not executable"
      continue
    fi
    record_pass "$label: settings.json command '$cmd' resolves to an executable file"
  done <<EOF
$commands
EOF

  if [ "$count" -eq 0 ]; then
    record_fail "$label: .claude/settings.json parsed zero hook commands"
  fi
}

check_codex_hooks_json() {
  # Dispatcher-existence assertion note: check_settings_json above resolves
  # each Claude-side hook command to a real, executable file. This function
  # does not repeat that check on the Codex side. The shipped command form
  # is `cd "$(git rev-parse --show-toplevel)" && ./.claude/hooks/ralph-dispatch.sh
  # PostToolUse` (C3-M3 fix) — after the leading `cd`, the dispatcher IS
  # invoked by a surface_root-relative path, unlike the earlier absolute
  # `$(git rev-parse --show-toplevel)/...` form this replaced (AR#1). We
  # still do not resolve it here because the command is a compound shell
  # expression (a `cd` plus `&&`), not a bare path — splitting the
  # executable token out of a shell-evaluated string reliably would need a
  # shell parser, not the `awk '{print $1}'` token grab check_settings_json
  # uses. Dispatcher existence is covered there via the Claude-side
  # settings.json entry; the cd-prefix and relative-invocation assertions
  # below instead pin the load-bearing SHAPE of the Codex command string so
  # a revert to the old absolute-path form fails.
  local label="$1" surface_root="$2"
  local hooks_json="$surface_root/.codex/hooks.json"

  if [ ! -f "$hooks_json" ]; then
    record_fail "$label: .codex/hooks.json not found at $hooks_json"
    return
  fi

  if ! jq empty "$hooks_json" >/dev/null 2>&1; then
    record_fail "$label: .codex/hooks.json is not valid JSON"
    return
  fi
  record_pass "$label: .codex/hooks.json is valid JSON"

  local post_tool_use_commands
  post_tool_use_commands="$(jq -r '.hooks.PostToolUse // [] | .[] | (.hooks // [])[] | select(.type == "command") | .command' "$hooks_json" 2>/dev/null || true)"

  if [ -z "$post_tool_use_commands" ]; then
    record_fail "$label: .codex/hooks.json has no PostToolUse command entries"
    return
  fi

  local routed=0 cd_prefix_ok=0 relative_dispatch_ok=0 cmd
  while IFS= read -r cmd; do
    [ -z "$cmd" ] && continue
    case "$cmd" in
      *ralph-dispatch.sh*) routed=1 ;;
    esac
    # C3-M3 regression guard: pin the two load-bearing pieces of the
    # cd-first command form so a revert to the pre-AR#1 absolute-path form
    # (which satisfied *ralph-dispatch.sh* identically) fails here instead
    # of passing silently.
    if printf '%s' "$cmd" | grep -qF 'cd "$(git rev-parse --show-toplevel)"'; then
      cd_prefix_ok=1
    fi
    if printf '%s' "$cmd" | grep -qF './.claude/hooks/ralph-dispatch.sh'; then
      relative_dispatch_ok=1
    fi
  done <<EOF
$post_tool_use_commands
EOF

  if [ "$routed" -eq 1 ]; then
    record_pass "$label: hooks.json PostToolUse routes through ralph-dispatch.sh"
  else
    record_fail "$label: hooks.json PostToolUse does not route through ralph-dispatch.sh"
  fi

  if [ "$cd_prefix_ok" -eq 1 ]; then
    record_pass "$label: hooks.json PostToolUse command cd's to the git root before dispatching (AR#1 regression guard)"
  else
    record_fail "$label: hooks.json PostToolUse command is missing the git-root cd prefix — a revert to the absolute-path form would silently break subdirectory launches (C3-M3)"
  fi

  if [ "$relative_dispatch_ok" -eq 1 ]; then
    record_pass "$label: hooks.json PostToolUse command invokes the dispatcher via a surface_root-relative path"
  else
    record_fail "$label: hooks.json PostToolUse command does not invoke ./.claude/hooks/ralph-dispatch.sh (relative form)"
  fi

  # Multi-event wiring (docs/plans/active/2026-08-24-codex-hooks-multi-event.md):
  # PreToolUse / SessionStart / UserPromptSubmit each must have at least one
  # command entry, routed through ralph-dispatch.sh with the same
  # cd-to-git-root prefix and surface_root-relative dispatcher path pinned
  # above for PostToolUse.
  local event
  for event in PreToolUse SessionStart UserPromptSubmit; do
    local event_commands
    event_commands="$(jq -r --arg event "$event" '.hooks[$event] // [] | .[] | (.hooks // [])[] | select(.type == "command") | .command' "$hooks_json" 2>/dev/null || true)"

    if [ -z "$event_commands" ]; then
      record_fail "$label: .codex/hooks.json has no $event command entries"
      continue
    fi
    record_pass "$label: .codex/hooks.json has at least one $event command entry"

    local event_routed=0 event_cd_prefix_ok=0 event_dispatch_ok=0 event_cmd
    while IFS= read -r event_cmd; do
      [ -z "$event_cmd" ] && continue
      case "$event_cmd" in
        *"ralph-dispatch.sh $event"*) event_routed=1 ;;
      esac
      if printf '%s' "$event_cmd" | grep -qF 'cd "$(git rev-parse --show-toplevel)"'; then
        event_cd_prefix_ok=1
      fi
      if printf '%s' "$event_cmd" | grep -qF './.claude/hooks/ralph-dispatch.sh'; then
        event_dispatch_ok=1
      fi
    done <<EOF
$event_commands
EOF

    if [ "$event_routed" -eq 1 ]; then
      record_pass "$label: hooks.json $event routes through ralph-dispatch.sh $event"
    else
      record_fail "$label: hooks.json $event does not route through ralph-dispatch.sh $event"
    fi

    if [ "$event_cd_prefix_ok" -eq 1 ]; then
      record_pass "$label: hooks.json $event command cd's to the git root before dispatching"
    else
      record_fail "$label: hooks.json $event command is missing the git-root cd prefix"
    fi

    if [ "$event_dispatch_ok" -eq 1 ]; then
      record_pass "$label: hooks.json $event command invokes the dispatcher via a surface_root-relative path"
    else
      record_fail "$label: hooks.json $event command does not invoke ./.claude/hooks/ralph-dispatch.sh (relative form)"
    fi
  done

  # PreToolUse matcher must be exactly "Bash" (Slice 1 live-fire confirmed
  # the real tool name; a looser or missing matcher would fire the guard on
  # non-Bash tool calls it was never validated against).
  local pre_tool_use_matchers
  pre_tool_use_matchers="$(jq -r '.hooks.PreToolUse // [] | .[] | .matcher // ""' "$hooks_json" 2>/dev/null || true)"
  if [ "$pre_tool_use_matchers" = "Bash" ]; then
    record_pass "$label: hooks.json PreToolUse matcher is exactly \"Bash\""
  else
    record_fail "$label: hooks.json PreToolUse matcher is $(printf '%s' "$pre_tool_use_matchers" | tr '\n' ',') , want exactly \"Bash\""
  fi

  # Non-goal guard (plan non-goals, 2026-08-24): SessionEnd / PreCompact
  # auto-WIP-commit the dirty tree and are deliberately NOT wired in this
  # rollout — a premature addition must fail this test.
  local absent_event
  for absent_event in SessionEnd PreCompact; do
    if jq -e --arg event "$absent_event" '.hooks | has($event)' "$hooks_json" >/dev/null 2>&1; then
      record_fail "$label: hooks.json unexpectedly wires $absent_event (deliberately deferred — see plan non-goals)"
    else
      record_pass "$label: hooks.json does not wire $absent_event (deliberately deferred)"
    fi
  done
}

check_codex_hooks_json_byte_identical() {
  local root_file="$REPO_ROOT/.codex/hooks.json"
  local template_file="$REPO_ROOT/templates/base/.codex/hooks.json"

  if [ ! -f "$root_file" ] || [ ! -f "$template_file" ]; then
    record_fail "byte-identity: .codex/hooks.json missing at root or templates/base"
    return
  fi

  if diff -q "$root_file" "$template_file" >/dev/null 2>&1; then
    record_pass "byte-identity: .codex/hooks.json is identical at root and templates/base"
  else
    record_fail "byte-identity: .codex/hooks.json differs between root and templates/base"
  fi
}

# codex_config_toml_has_hooks_table <config_file> — whitespace-tolerant
# detector for a [hooks] / [[hooks.*]] TOML table: strips whitespace from
# each `[`-leading line before matching, so a spaced variant like
# `[[ hooks.PostToolUse ]]` (valid TOML) cannot walk past the guard the way
# an anchored `\[\[?hooks` regex would. Mirrors
# scripts/verify.local.sh's codex_config_has_inline_hooks awk approach.
codex_config_toml_has_hooks_table() {
  awk '
    /^[[:space:]]*\[/ {
      line = $0
      gsub(/[[:space:]]/, "", line)
      if (line ~ /^\[\[?hooks(\.|\]\]?)/) { found = 1 }
    }
    END { exit found ? 0 : 1 }
  ' "$1"
}

# check_codex_config_toml_no_hooks_tables <label> <surface_root> —
# .codex/hooks.json is the Codex hook source of truth now (inline TOML
# [[hooks.*]] entries never fired — see docs/evidence/); a surviving
# [hooks]/[[hooks.*]] table in config.toml is a stale duplicate
# representation that must not be reintroduced.
check_codex_config_toml_no_hooks_tables() {
  local label="$1" surface_root="$2"
  local config="$surface_root/.codex/config.toml"

  [ -f "$config" ] || return

  if codex_config_toml_has_hooks_table "$config"; then
    record_fail "$label: .codex/config.toml still has a [hooks]/[[hooks.*]] table — hooks.json is the source of truth, remove it"
  else
    record_pass "$label: .codex/config.toml has no [hooks]/[[hooks.*]] table"
  fi
}

# test_codex_config_toml_hooks_table_detector — self-test fixture for
# codex_config_toml_has_hooks_table: covers the tight form (`[[hooks.X]]`),
# the spaced form (`[[ hooks.X ]]`), and a clean config with no hooks table,
# so the whitespace-tolerance fix stays pinned by a regression case.
test_codex_config_toml_hooks_table_detector() {
  local tmp_dir tight spaced clean
  tmp_dir="$(mktemp -d)"
  trap 'rm -rf "$tmp_dir"' RETURN

  tight="$tmp_dir/tight.toml"
  spaced="$tmp_dir/spaced.toml"
  clean="$tmp_dir/clean.toml"

  cat > "$tight" <<'EOF'
model = "gpt-5.5"

[[hooks.PostToolUse]]
command = ["./.claude/hooks/check_mojibake.sh"]
EOF

  cat > "$spaced" <<'EOF'
model = "gpt-5.5"

[[ hooks.PostToolUse ]]
command = ["./.claude/hooks/check_mojibake.sh"]
EOF

  cat > "$clean" <<'EOF'
model = "gpt-5.5"

[features]
hooks = true
EOF

  local ok=0

  if codex_config_toml_has_hooks_table "$tight"; then
    :
  else
    echo "FAIL: codex_config_toml_has_hooks_table missed the tight [[hooks.X]] form" >&2
    ok=1
  fi

  if codex_config_toml_has_hooks_table "$spaced"; then
    :
  else
    echo "FAIL: codex_config_toml_has_hooks_table missed the spaced [[ hooks.X ]] form" >&2
    ok=1
  fi

  if codex_config_toml_has_hooks_table "$clean"; then
    echo "FAIL: codex_config_toml_has_hooks_table false-positived on a config with no hooks table" >&2
    ok=1
  fi

  if [ "$ok" -eq 0 ]; then
    record_pass "self-test: codex_config_toml_has_hooks_table detects tight and spaced [[hooks.*]] forms"
  else
    record_fail "self-test: codex_config_toml_has_hooks_table regression (see stderr)"
  fi
}

# check_no_direct_hook_scripts_in_hooks_json <label> <surface_root> — every
# hook script basename (`*.sh`) referenced by an actual "command" value in
# .codex/hooks.json must be ralph-dispatch.sh, not a direct call to another
# hook script. This is the dispatcher-parity regression guard: it fails
# closed the moment a future edit reintroduces the legacy direct-call form.
# Scoped to jq-extracted command STRINGS (not raw file grep) so prose
# elsewhere in the file (e.g. a "description" field) can never be
# misread as a hook command.
check_no_direct_hook_scripts_in_hooks_json() {
  local label="$1" surface_root="$2"
  local hooks_json="$surface_root/.codex/hooks.json"

  [ -f "$hooks_json" ] || return

  local commands
  commands="$(jq -r '.hooks // {} | to_entries[] | .value[] | (.hooks // [])[] | select(.type == "command") | .command' "$hooks_json" 2>/dev/null || true)"

  [ -z "$commands" ] && return

  local cmd basenames name
  while IFS= read -r cmd; do
    [ -z "$cmd" ] && continue
    basenames="$(printf '%s' "$cmd" | grep -oE '[A-Za-z0-9_.-]+\.sh' | sort -u || true)"
    [ -z "$basenames" ] && continue
    while IFS= read -r name; do
      [ -z "$name" ] && continue
      if [ "$name" = "ralph-dispatch.sh" ]; then
        record_pass "$label: hooks.json command references only ralph-dispatch.sh (no direct hook-script call)"
      else
        record_fail "$label: hooks.json command '$cmd' references '$name' directly — route through ralph-dispatch.sh instead"
      fi
    done <<EOF
$basenames
EOF
  done <<EOF
$commands
EOF
}

# check_no_direct_hook_scripts_in_config_toml <label> <surface_root> — same
# guard as above, scoped to .codex/config.toml `command = ...` assignment
# lines only (not arbitrary prose/comments — this file legitimately
# mentions unrelated *.sh scripts, e.g. the git secret-guard hooks, that
# are not Codex PostToolUse commands).
check_no_direct_hook_scripts_in_config_toml() {
  local label="$1" surface_root="$2"
  local config="$surface_root/.codex/config.toml"

  [ -f "$config" ] || return

  local basenames
  basenames="$(grep -E '^[[:space:]]*command[[:space:]]*=' "$config" | grep -oE '[A-Za-z0-9_.-]+\.sh' | sort -u || true)"

  if [ -z "$basenames" ]; then
    record_pass "$label: config.toml has no hook command assignments"
    return
  fi

  local name
  while IFS= read -r name; do
    [ -z "$name" ] && continue
    if [ "$name" = "ralph-dispatch.sh" ]; then
      record_pass "$label: config.toml command references only ralph-dispatch.sh (no direct hook-script call)"
    else
      record_fail "$label: config.toml command references '$name' directly — route through ralph-dispatch.sh instead"
    fi
  done <<EOF
$basenames
EOF
}

check_settings_json "root" "$REPO_ROOT"
check_codex_hooks_json "root" "$REPO_ROOT"
check_codex_config_toml_no_hooks_tables "root" "$REPO_ROOT"
check_no_direct_hook_scripts_in_hooks_json "root" "$REPO_ROOT"
check_no_direct_hook_scripts_in_config_toml "root" "$REPO_ROOT"
check_settings_json "templates/base" "$REPO_ROOT/templates/base"
check_codex_hooks_json "templates/base" "$REPO_ROOT/templates/base"
check_codex_config_toml_no_hooks_tables "templates/base" "$REPO_ROOT/templates/base"
check_no_direct_hook_scripts_in_hooks_json "templates/base" "$REPO_ROOT/templates/base"
check_no_direct_hook_scripts_in_config_toml "templates/base" "$REPO_ROOT/templates/base"
check_codex_hooks_json_byte_identical
test_codex_config_toml_hooks_table_detector

echo
echo "=== test-hook-wiring.sh results ==="
for line in "${results[@]}"; do
  echo "  $line"
done
echo
echo "  PASS: $pass"
echo "  FAIL: $fail"

if [ "$fail" -gt 0 ]; then
  exit 1
fi
exit 0
