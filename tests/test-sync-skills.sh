#!/usr/bin/env bash
# tests/test-sync-skills.sh — exercise scripts/sync-skills.sh against
# synthetic fixture skill trees.
#
# Covers:
#   A. clean run: generator creates a mirror passing check-skill-sync.sh
#   B. orphan removal: deleting a source skill removes the mirror dir
#   C. Claude-only frontmatter key (allowed-tools) is dropped from mirror
#   D. Claude-only frontmatter key (disable-model-invocation) is dropped;
#      agents/openai.yaml is created instead
#   E. auxiliary files (template.md, prompts/) are copied verbatim
#   F. real skill tree: generator on the current tree is idempotent
#      (second run produces no diff)
#   G. removing disable-model-invocation and re-syncing deletes agents/openai.yaml
#      and removes the empty agents/ dir

set -eu

REPO_ROOT="$(cd "$(dirname "$0")/.." && pwd)"
SYNC_SCRIPT="$REPO_ROOT/scripts/sync-skills.sh"
CHECK_SCRIPT="$REPO_ROOT/scripts/check-skill-sync.sh"

if [ ! -x "$SYNC_SCRIPT" ]; then
  echo "FAIL: $SYNC_SCRIPT not executable"
  exit 1
fi
if [ ! -x "$CHECK_SCRIPT" ]; then
  echo "FAIL: $CHECK_SCRIPT not executable"
  exit 1
fi

pass=0
fail=0

run_case() {
  local label="$1"
  local expected_exit="$2"  # 0 = pass, 1 = fail
  shift 2
  if "$@" >/dev/null 2>&1; then
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

# Build a minimal git repo fixture with one source skill.
mk_fixture() {
  local dir="$1"
  git -C "$dir" init -q
  git -C "$dir" config user.email "test@example.com"
  git -C "$dir" config user.name "Test"
  mkdir -p "$dir/.claude/skills/alpha" "$dir/.agents/skills"
  cat > "$dir/.claude/skills/alpha/SKILL.md" <<'EOF'
---
name: alpha
description: Alpha skill.
---
Alpha body line 1.
Alpha body line 2.
EOF
}

# ── A. clean run: mirror is created and check-skill-sync.sh passes ───────────
A_DIR="$(mktemp -d)"
trap 'rm -rf "$A_DIR"' EXIT
mk_fixture "$A_DIR"

run_case "A. generator runs without error" 0 \
  bash -c "cd '$A_DIR' && CLAUDE_ROOT=.claude/skills CODEX_ROOT=.agents/skills '$SYNC_SCRIPT'"
run_case "A. generated mirror passes check-skill-sync.sh" 0 \
  bash -c "cd '$A_DIR' && CLAUDE_ROOT=.claude/skills CODEX_ROOT=.agents/skills '$CHECK_SCRIPT'"

# ── B. orphan removal: deleting source skill removes mirror dir ──────────────
B_DIR="$(mktemp -d)"
trap 'rm -rf "$A_DIR" "$B_DIR"' EXIT
mk_fixture "$B_DIR"

# Create the mirror for the initial skill.
bash -c "cd '$B_DIR' && CLAUDE_ROOT=.claude/skills CODEX_ROOT=.agents/skills '$SYNC_SCRIPT'" >/dev/null 2>&1
# Now add a second source skill and re-sync.
mkdir -p "$B_DIR/.claude/skills/beta"
cat > "$B_DIR/.claude/skills/beta/SKILL.md" <<'EOF'
---
name: beta
description: Beta skill.
---
Beta body.
EOF
bash -c "cd '$B_DIR' && CLAUDE_ROOT=.claude/skills CODEX_ROOT=.agents/skills '$SYNC_SCRIPT'" >/dev/null 2>&1
# Confirm both mirror dirs exist.
run_case "B. both mirror dirs present before removal" 0 \
  test -d "$B_DIR/.agents/skills/beta"
# Now remove the beta source skill.
rm -rf "$B_DIR/.claude/skills/beta"
bash -c "cd '$B_DIR' && CLAUDE_ROOT=.claude/skills CODEX_ROOT=.agents/skills '$SYNC_SCRIPT'" >/dev/null 2>&1
run_case "B. orphan mirror dir removed after source deleted" 1 \
  test -d "$B_DIR/.agents/skills/beta"

# ── C. allowed-tools is dropped from mirror SKILL.md frontmatter ────────────
C_DIR="$(mktemp -d)"
trap 'rm -rf "$A_DIR" "$B_DIR" "$C_DIR"' EXIT
git -C "$C_DIR" init -q
git -C "$C_DIR" config user.email "test@example.com"
git -C "$C_DIR" config user.name "Test"
mkdir -p "$C_DIR/.claude/skills/gamma" "$C_DIR/.agents/skills"
cat > "$C_DIR/.claude/skills/gamma/SKILL.md" <<'EOF'
---
name: gamma
description: Gamma skill.
allowed-tools: Read, Write, Bash
---
Gamma body.
EOF

bash -c "cd '$C_DIR' && CLAUDE_ROOT=.claude/skills CODEX_ROOT=.agents/skills '$SYNC_SCRIPT'" >/dev/null 2>&1

run_case "C. allowed-tools absent from mirror frontmatter" 1 \
  grep -q '^allowed-tools:' "$C_DIR/.agents/skills/gamma/SKILL.md"
run_case "C. name still present in mirror frontmatter" 0 \
  grep -q '^name: gamma' "$C_DIR/.agents/skills/gamma/SKILL.md"
run_case "C. description still present in mirror frontmatter" 0 \
  grep -q '^description: Gamma skill.' "$C_DIR/.agents/skills/gamma/SKILL.md"
run_case "C. check-skill-sync.sh passes after transform" 0 \
  bash -c "cd '$C_DIR' && CLAUDE_ROOT=.claude/skills CODEX_ROOT=.agents/skills '$CHECK_SCRIPT'"

# ── D. disable-model-invocation dropped; agents/openai.yaml created ──────────
D_DIR="$(mktemp -d)"
trap 'rm -rf "$A_DIR" "$B_DIR" "$C_DIR" "$D_DIR"' EXIT
git -C "$D_DIR" init -q
git -C "$D_DIR" config user.email "test@example.com"
git -C "$D_DIR" config user.name "Test"
mkdir -p "$D_DIR/.claude/skills/delta" "$D_DIR/.agents/skills"
cat > "$D_DIR/.claude/skills/delta/SKILL.md" <<'EOF'
---
name: delta
description: Delta skill. Manual trigger only.
disable-model-invocation: true
allowed-tools: Bash, Read
---
Delta body.
EOF

bash -c "cd '$D_DIR' && CLAUDE_ROOT=.claude/skills CODEX_ROOT=.agents/skills '$SYNC_SCRIPT'" >/dev/null 2>&1

run_case "D. disable-model-invocation absent from mirror frontmatter" 1 \
  grep -q '^disable-model-invocation:' "$D_DIR/.agents/skills/delta/SKILL.md"
run_case "D. agents/openai.yaml created" 0 \
  test -f "$D_DIR/.agents/skills/delta/agents/openai.yaml"
run_case "D. agents/openai.yaml has allow_implicit_invocation: false" 0 \
  grep -q 'allow_implicit_invocation: false' "$D_DIR/.agents/skills/delta/agents/openai.yaml"
run_case "D. check-skill-sync.sh passes (policy parity)" 0 \
  bash -c "cd '$D_DIR' && CLAUDE_ROOT=.claude/skills CODEX_ROOT=.agents/skills '$CHECK_SCRIPT'"

# ── E. auxiliary files (template.md, prompts/) are copied verbatim ───────────
E_DIR="$(mktemp -d)"
trap 'rm -rf "$A_DIR" "$B_DIR" "$C_DIR" "$D_DIR" "$E_DIR"' EXIT
git -C "$E_DIR" init -q
git -C "$E_DIR" config user.email "test@example.com"
git -C "$E_DIR" config user.name "Test"
mkdir -p "$E_DIR/.claude/skills/epsilon/prompts" "$E_DIR/.agents/skills"
cat > "$E_DIR/.claude/skills/epsilon/SKILL.md" <<'EOF'
---
name: epsilon
description: Epsilon skill.
---
Epsilon body.
EOF
printf 'Template content.\n' > "$E_DIR/.claude/skills/epsilon/template.md"
printf 'Adversarial prompt.\n' > "$E_DIR/.claude/skills/epsilon/prompts/adversarial.md"

bash -c "cd '$E_DIR' && CLAUDE_ROOT=.claude/skills CODEX_ROOT=.agents/skills '$SYNC_SCRIPT'" >/dev/null 2>&1

run_case "E. template.md copied to mirror" 0 \
  test -f "$E_DIR/.agents/skills/epsilon/template.md"
run_case "E. template.md byte-identical" 0 \
  cmp -s "$E_DIR/.claude/skills/epsilon/template.md" "$E_DIR/.agents/skills/epsilon/template.md"
run_case "E. prompts/adversarial.md copied to mirror" 0 \
  test -f "$E_DIR/.agents/skills/epsilon/prompts/adversarial.md"
run_case "E. prompts/adversarial.md byte-identical" 0 \
  cmp -s "$E_DIR/.claude/skills/epsilon/prompts/adversarial.md" \
      "$E_DIR/.agents/skills/epsilon/prompts/adversarial.md"
run_case "E. check-skill-sync.sh passes (prompts/ parity)" 0 \
  bash -c "cd '$E_DIR' && CLAUDE_ROOT=.claude/skills CODEX_ROOT=.agents/skills '$CHECK_SCRIPT'"

# ── G. removing disable-model-invocation and re-syncing deletes openai.yaml ───
# Verifies that the openai.yaml cleanup branch in sync-skills.sh fires when the
# flag is removed from a skill that previously had it.
G_DIR="$(mktemp -d)"
trap 'rm -rf "$A_DIR" "$B_DIR" "$C_DIR" "$D_DIR" "$E_DIR" "$G_DIR"' EXIT
git -C "$G_DIR" init -q
git -C "$G_DIR" config user.email "test@example.com"
git -C "$G_DIR" config user.name "Test"
mkdir -p "$G_DIR/.claude/skills/gamma_dm" "$G_DIR/.agents/skills"
cat > "$G_DIR/.claude/skills/gamma_dm/SKILL.md" <<'EOF'
---
name: gamma_dm
description: Gamma-dm skill. Manual trigger only.
disable-model-invocation: true
---
Gamma-dm body.
EOF

# First sync: disable-model-invocation: true → openai.yaml should be created.
bash -c "cd '$G_DIR' && CLAUDE_ROOT=.claude/skills CODEX_ROOT=.agents/skills '$SYNC_SCRIPT'" >/dev/null 2>&1
run_case "G. openai.yaml present after initial sync with disable-model-invocation: true" 0 \
  test -f "$G_DIR/.agents/skills/gamma_dm/agents/openai.yaml"

# Rewrite source SKILL.md without the flag.
cat > "$G_DIR/.claude/skills/gamma_dm/SKILL.md" <<'EOF'
---
name: gamma_dm
description: Gamma-dm skill. Auto-invoked when relevant.
---
Gamma-dm body.
EOF

# Second sync: flag removed → openai.yaml must be deleted and agents/ dir gone.
bash -c "cd '$G_DIR' && CLAUDE_ROOT=.claude/skills CODEX_ROOT=.agents/skills '$SYNC_SCRIPT'" >/dev/null 2>&1
run_case "G. openai.yaml removed after re-sync without disable-model-invocation" 1 \
  test -f "$G_DIR/.agents/skills/gamma_dm/agents/openai.yaml"
run_case "G. agents/ dir removed when empty after openai.yaml deletion" 1 \
  test -d "$G_DIR/.agents/skills/gamma_dm/agents"
run_case "G. check-skill-sync.sh passes after flag removal and re-sync" 0 \
  bash -c "cd '$G_DIR' && CLAUDE_ROOT=.claude/skills CODEX_ROOT=.agents/skills '$CHECK_SCRIPT'"

# ── F. real skill tree: generator is idempotent ───────────────────────────────
# Run the generator on the actual repo twice; diff the two snapshots.
FIRST_SNAP="$(mktemp -d)"
SECOND_SNAP="$(mktemp -d)"
trap 'rm -rf "$A_DIR" "$B_DIR" "$C_DIR" "$D_DIR" "$E_DIR" "$G_DIR" "$FIRST_SNAP" "$SECOND_SNAP"' EXIT

bash -c "cd '$REPO_ROOT' && './scripts/sync-skills.sh'" >/dev/null 2>&1
cp -a "$REPO_ROOT/.agents/skills/." "$FIRST_SNAP/"
bash -c "cd '$REPO_ROOT' && './scripts/sync-skills.sh'" >/dev/null 2>&1
cp -a "$REPO_ROOT/.agents/skills/." "$SECOND_SNAP/"

run_case "F. generator is idempotent on real skill tree" 0 \
  diff -r "$FIRST_SNAP" "$SECOND_SNAP"

echo ""
echo "  PASS: $pass"
echo "  FAIL: $fail"
exit "$fail"
