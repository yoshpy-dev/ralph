#!/usr/bin/env bash
# test-hook-wiring.sh — regression guard for "moved hooks leave dangling
# references" (overlay-scaffold-v2 Phase 2 plan, Codex advisory #2).
#
# For BOTH the meta-repo root and templates/base/, parses every hook
# command out of .claude/settings.json (jq) and .codex/config.toml (grep
# the TOML command strings) and asserts the referenced script file exists
# relative to that surface's root and is executable. This is a durable
# gate against a future hooks reshuffle (e.g. the ralph-dispatch.sh /
# <event>.d migration this test was added alongside) silently leaving a
# settings.json or config.toml entry pointing at a path that no longer
# exists.

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

check_codex_config_toml() {
  local label="$1" surface_root="$2"
  local config="$surface_root/.codex/config.toml"

  if [ ! -f "$config" ]; then
    record_fail "$label: .codex/config.toml not found at $config"
    return
  fi

  # Restrict to `command = [ "..." , "..." ]` (possibly multi-element) TOML
  # array entries: first select lines that assign the `command` key, then
  # pull out each double-quoted `./`-prefixed string from those lines only.
  local commands
  commands="$(grep -E '^[[:space:]]*command[[:space:]]*=' "$config" | grep -oE '"\./[^"]+"' | tr -d '"' || true)"

  if [ -z "$commands" ]; then
    record_fail "$label: .codex/config.toml has no [[hooks.*]] command entries (expected at least one)"
    return
  fi

  local count=0
  while IFS= read -r exe; do
    [ -z "$exe" ] && continue
    count=$((count + 1))
    local rel_path="${exe#./}"
    local abs_path="$surface_root/$rel_path"
    if [ ! -f "$abs_path" ]; then
      record_fail "$label: config.toml command '$exe' -> missing file $abs_path"
      continue
    fi
    if [ ! -x "$abs_path" ]; then
      record_fail "$label: config.toml command '$exe' -> $abs_path exists but is not executable"
      continue
    fi
    record_pass "$label: config.toml command '$exe' resolves to an executable file"
  done <<EOF
$commands
EOF

  if [ "$count" -eq 0 ]; then
    record_fail "$label: .codex/config.toml parsed zero hook commands"
  fi
}

check_settings_json "root" "$REPO_ROOT"
check_codex_config_toml "root" "$REPO_ROOT"
check_settings_json "templates/base" "$REPO_ROOT/templates/base"
check_codex_config_toml "templates/base" "$REPO_ROOT/templates/base"

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
