#!/usr/bin/env sh
# shellcheck disable=SC1090  # the whole point is sourcing $CONFIG dynamically
# test-codex-exec-invocation.sh — regression guard for the codex-exec-stdin-and-model
# plan (issue #184, docs/plans/active/2026-09-29-codex-exec-stdin-and-model.md).
#
# `codex exec` calls launched from an agent's Bash tool used to hang waiting
# for stdin ("Reading additional input from stdin...", #153) and could 400
# when a shell alias injected a conflicting `-m` (#162). The fix pins every
# codex invocation line inside the /plan (Codex plan advisory) and
# /cross-review (reviewer = codex) skill bodies to a fixed shape:
#
#   command codex -m "${RALPH_CODEX_REVIEWER_MODEL:-<default>}" \
#     -c "model_reasoning_effort=${RALPH_CODEX_REASONING_EFFORT:-<default>}" \
#     exec ... -o <file> </dev/null
#
# This test checks that shape holds across all four skill-body faces
# (.claude/skills, .agents/skills, templates/base/.claude/skills,
# templates/base/.agents/skills) and that the documented fallback values
# match scripts/ralph-config.sh's actual defaults, so a drift in either
# place fails loudly instead of silently reintroducing the stdin hang or
# the alias/model conflict.
#
# It also checks (AC-8) that /cross-review's four faces document the
# "reviewer incomplete" path for a codex background call that does not
# finish cleanly (non-zero exit, timeout, or a missing/empty -o file).

set -eu

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
cd "$PROJECT_ROOT"

CONFIG="$PROJECT_ROOT/scripts/ralph-config.sh"

_pass=0
_fail=0
_total=0

pass() {
  _pass=$((_pass + 1))
  _total=$((_total + 1))
  printf '  PASS: %s\n' "$1"
}

fail() {
  _fail=$((_fail + 1))
  _total=$((_total + 1))
  printf '  FAIL: %s\n' "$1"
}

# ═══════════════════════════════════════════════════════════════════
# Expected fallback values, read from the shell config itself so a
# default change there (mutation iv in the plan's red evidence) fails
# this test rather than requiring a second hardcoded copy here.
# ═══════════════════════════════════════════════════════════════════

EXPECT_MODEL="$(unset RALPH_CODEX_REVIEWER_MODEL; . "$CONFIG"; echo "$RALPH_CODEX_REVIEWER_MODEL")"
EXPECT_EFFORT="$(unset RALPH_CODEX_REASONING_EFFORT; . "$CONFIG"; echo "$RALPH_CODEX_REASONING_EFFORT")"

FACES=".claude/skills .agents/skills templates/base/.claude/skills templates/base/.agents/skills"
SKILLS="plan cross-review"

# check_invocation_line <face> <skill> <lineno> <content>
# Verifies one codex-exec invocation line carries all four required tokens.
check_invocation_line() {
  face="$1"; skill="$2"; lineno="$3"; content="$4"
  label="$face/$skill/SKILL.md:$lineno"

  case "$content" in
    *'</dev/null'*) pass "$label: has </dev/null" ;;
    *) fail "$label: missing </dev/null -- $content" ;;
  esac

  case "$content" in
    *'-m "${RALPH_CODEX_REVIEWER_MODEL:-'*) pass "$label: has -m \"\${RALPH_CODEX_REVIEWER_MODEL:-...}\"" ;;
    *) fail "$label: missing -m \"\${RALPH_CODEX_REVIEWER_MODEL:-...}\" -- $content" ;;
  esac

  case "$content" in
    *'model_reasoning_effort=${RALPH_CODEX_REASONING_EFFORT:-'*) pass "$label: has model_reasoning_effort=\${RALPH_CODEX_REASONING_EFFORT:-...}" ;;
    *) fail "$label: missing model_reasoning_effort=\${RALPH_CODEX_REASONING_EFFORT:-...} -- $content" ;;
  esac

  case "$content" in
    *' -o '*|*'--output-last-message'*) pass "$label: has -o / --output-last-message" ;;
    *) fail "$label: missing -o / --output-last-message -- $content" ;;
  esac
}

# check_fallback_values <face> <skill> <file>
# Extracts every ${RALPH_CODEX_REVIEWER_MODEL:-X} / ${RALPH_CODEX_REASONING_EFFORT:-Y}
# occurrence in the file and confirms X == EXPECT_MODEL, Y == EXPECT_EFFORT.
check_fallback_values() {
  face="$1"; skill="$2"; file="$3"
  label="$face/$skill/SKILL.md"

  model_matches="$(grep -oE '\$\{RALPH_CODEX_REVIEWER_MODEL:-[^}]*\}' "$file" || true)"
  if [ -z "$model_matches" ]; then
    fail "$label: no \${RALPH_CODEX_REVIEWER_MODEL:-...} fallback found"
  else
    old_ifs=$IFS
    IFS='
'
    for m in $model_matches; do
      fb="${m#*:-}"
      fb="${fb%\}}"
      if [ "$fb" = "$EXPECT_MODEL" ]; then
        pass "$label: RALPH_CODEX_REVIEWER_MODEL fallback matches ralph-config.sh default ($EXPECT_MODEL)"
      else
        fail "$label: RALPH_CODEX_REVIEWER_MODEL fallback = '$fb', want '$EXPECT_MODEL' (scripts/ralph-config.sh default)"
      fi
    done
    IFS=$old_ifs
  fi

  effort_matches="$(grep -oE '\$\{RALPH_CODEX_REASONING_EFFORT:-[^}]*\}' "$file" || true)"
  if [ -z "$effort_matches" ]; then
    fail "$label: no \${RALPH_CODEX_REASONING_EFFORT:-...} fallback found"
  else
    old_ifs=$IFS
    IFS='
'
    for m in $effort_matches; do
      fb="${m#*:-}"
      fb="${fb%\}}"
      if [ "$fb" = "$EXPECT_EFFORT" ]; then
        pass "$label: RALPH_CODEX_REASONING_EFFORT fallback matches ralph-config.sh default ($EXPECT_EFFORT)"
      else
        fail "$label: RALPH_CODEX_REASONING_EFFORT fallback = '$fb', want '$EXPECT_EFFORT' (scripts/ralph-config.sh default)"
      fi
    done
    IFS=$old_ifs
  fi
}

# check_face_skill <face> <skill>
check_face_skill() {
  face="$1"; skill="$2"
  file="$PROJECT_ROOT/$face/$skill/SKILL.md"
  label="$face/$skill/SKILL.md"

  if [ ! -f "$file" ]; then
    fail "$label: file missing"
    return
  fi

  # A codex-exec invocation line: contains the lowercase `codex` binary name
  # and a following ` exec` token. Lowercase `codex` (vs. prose "Codex") is
  # deliberate -- it is what distinguishes a real invocation from a prose
  # mention, so this file's own prose is worded to avoid a false match
  # (no lowercase `codex` token sitting on the same line as the substring
  # " exec", e.g. inside a word like "execution").
  invocation_lines="$(grep -n 'codex' "$file" | grep ' exec' || true)"

  if [ -z "$invocation_lines" ]; then
    fail "$label: no codex exec invocation line found"
  else
    pass "$label: has at least one codex exec invocation line"
    old_ifs=$IFS
    IFS='
'
    for line in $invocation_lines; do
      lineno="${line%%:*}"
      content="${line#*:}"
      check_invocation_line "$face" "$skill" "$lineno" "$content"
    done
    IFS=$old_ifs
  fi

  check_fallback_values "$face" "$skill" "$file"
}

# check_reviewer_incomplete <face>
# AC-8: /cross-review's four faces must document the "reviewer incomplete"
# path for a codex background call that does not finish cleanly.
check_reviewer_incomplete() {
  face="$1"
  file="$PROJECT_ROOT/$face/cross-review/SKILL.md"
  label="$face/cross-review/SKILL.md"

  if [ ! -f "$file" ]; then
    fail "$label: file missing"
    return
  fi

  if grep -q 'Reviewer status: incomplete' "$file"; then
    pass "$label: documents 'Reviewer status: incomplete'"
  else
    fail "$label: missing 'Reviewer status: incomplete' (reviewer-incomplete path, AC-8)"
  fi
}

main() {
  echo "=== codex exec invocation tests ==="
  echo "  EXPECT_MODEL=$EXPECT_MODEL EXPECT_EFFORT=$EXPECT_EFFORT (from scripts/ralph-config.sh)"

  for face in $FACES; do
    for skill in $SKILLS; do
      echo ""
      echo "--- $face/$skill ---"
      check_face_skill "$face" "$skill"
    done
  done

  echo ""
  echo "=== reviewer-incomplete path (AC-8) ==="
  for face in $FACES; do
    check_reviewer_incomplete "$face"
  done

  echo ""
  echo "========================================="
  printf 'Results: %d/%d passed' "$_pass" "$_total"
  if [ "$_fail" -gt 0 ]; then
    printf ', %d FAILED' "$_fail"
  fi
  echo ""
  echo "========================================="

  [ "$_fail" -eq 0 ]
}

main
