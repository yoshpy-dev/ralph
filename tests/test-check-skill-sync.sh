#!/usr/bin/env bash
# tests/test-check-skill-sync.sh — exercise scripts/check-skill-sync.sh against
# synthetic fixtures. Verifies the five drift modes (inventory, body, name,
# description, policy) each fail closed, and that a clean fixture passes.
#
# Spec: AC-3 of docs/specs/2026-05-07-codex-cli-parity.md.

set -eu

REPO_ROOT="$(cd "$(dirname "$0")/.." && pwd)"
SCRIPT="$REPO_ROOT/scripts/check-skill-sync.sh"

if [ ! -x "$SCRIPT" ]; then
  echo "FAIL: $SCRIPT not executable"
  exit 1
fi

pass=0
fail=0

run_case() {
  local label="$1"
  local expected_exit="$2"   # 0 (parity) or 1 (drift)
  local fixture="$3"

  if (
    cd "$fixture" &&
    CLAUDE_ROOT=".claude/skills" CODEX_ROOT=".agents/skills" \
      "$SCRIPT" >/dev/null 2>&1
  ); then
    actual=0
  else
    actual=1
  fi

  if [ "$actual" -eq "$expected_exit" ]; then
    echo "  PASS  $label (exit $actual)"
    pass=$((pass + 1))
  else
    echo "  FAIL  $label: expected exit $expected_exit, got $actual"
    fail=$((fail + 1))
  fi
}

mk_skill_pair() {
  # mk_skill_pair <fixture-dir> <name> <description> <body> <claude-fm-extra> <codex-policy-yaml>
  local fixture="$1"
  local name="$2"
  local desc="$3"
  local body="$4"
  local claude_extra="$5"
  local codex_policy="$6"

  mkdir -p "$fixture/.claude/skills/$name" "$fixture/.agents/skills/$name"
  {
    printf -- '---\n'
    printf 'name: %s\n' "$name"
    printf 'description: %s\n' "$desc"
    if [ -n "$claude_extra" ]; then printf '%s\n' "$claude_extra"; fi
    printf -- '---\n'
    printf '%s\n' "$body"
  } > "$fixture/.claude/skills/$name/SKILL.md"
  {
    printf -- '---\n'
    printf 'name: %s\n' "$name"
    printf 'description: %s\n' "$desc"
    printf -- '---\n'
    printf '%s\n' "$body"
  } > "$fixture/.agents/skills/$name/SKILL.md"
  if [ -n "$codex_policy" ]; then
    mkdir -p "$fixture/.agents/skills/$name/agents"
    printf '%s\n' "$codex_policy" > "$fixture/.agents/skills/$name/agents/openai.yaml"
  fi
}

# ── A. clean fixture passes ──────────────────────────────────────────────────
A_DIR="$(mktemp -d)"
trap 'rm -rf "$A_DIR"' EXIT
mk_skill_pair "$A_DIR" "alpha" "Alpha description." "Body line 1\nBody line 2" "" ""
run_case "A. clean fixture (parity)" 0 "$A_DIR"

# ── B. inventory drift (skill only on Claude side) ──────────────────────────
B_DIR="$(mktemp -d)"
mk_skill_pair "$B_DIR" "alpha" "Alpha description." "Body" "" ""
mkdir -p "$B_DIR/.claude/skills/orphan"
printf -- '---\nname: orphan\ndescription: Orphan.\n---\nBody\n' \
  > "$B_DIR/.claude/skills/orphan/SKILL.md"
run_case "B. inventory drift (claude-only skill)" 1 "$B_DIR"
rm -rf "$B_DIR"

# ── C. body drift ────────────────────────────────────────────────────────────
C_DIR="$(mktemp -d)"
mk_skill_pair "$C_DIR" "alpha" "Alpha description." "Original body" "" ""
printf -- '---\nname: alpha\ndescription: Alpha description.\n---\nDifferent body\n' \
  > "$C_DIR/.agents/skills/alpha/SKILL.md"
run_case "C. body drift" 1 "$C_DIR"
rm -rf "$C_DIR"

# ── D. description drift ─────────────────────────────────────────────────────
D_DIR="$(mktemp -d)"
mk_skill_pair "$D_DIR" "alpha" "Alpha description." "Body" "" ""
printf -- '---\nname: alpha\ndescription: Different description.\n---\nBody\n' \
  > "$D_DIR/.agents/skills/alpha/SKILL.md"
run_case "D. description drift" 1 "$D_DIR"
rm -rf "$D_DIR"

# ── E. policy drift (Claude forbids implicit, Codex allows) ─────────────────
E_DIR="$(mktemp -d)"
mk_skill_pair "$E_DIR" "alpha" "Alpha description." "Body" "disable-model-invocation: true" ""
run_case "E. policy drift (claude forbid / codex allow)" 1 "$E_DIR"
rm -rf "$E_DIR"

# ── F. policy parity (both sides forbid implicit invocation) ────────────────
F_DIR="$(mktemp -d)"
mk_skill_pair "$F_DIR" "alpha" "Alpha description." "Body" "disable-model-invocation: true" \
  "policy:
  allow_implicit_invocation: false"
run_case "F. policy parity (both forbid)" 0 "$F_DIR"
rm -rf "$F_DIR"

echo ""
echo "  PASS: $pass"
echo "  FAIL: $fail"
exit "$fail"
