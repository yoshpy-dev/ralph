#!/usr/bin/env sh
# shellcheck disable=SC1090  # the whole point is sourcing $CONFIG dynamically
# test-codex-exec-invocation.sh — regression guard for the codex-exec-stdin-and-model
# fix (issue #184).
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
# Issue #197 adds a read-only sandbox to the same lines. The project's
# .codex/config.toml sets sandbox_mode = "danger-full-access" and
# `codex exec` never asks for approval, so a reviewer that inherits it can
# run whatever an instruction in the reviewed diff asks for. Every
# invocation line must therefore also select a sandbox, and every sandbox
# it selects must be read-only (`--sandbox read-only` for /plan's `exec`,
# `-c sandbox_mode=read-only` for /cross-review's `exec review`, which has
# no --sandbox flag). It must carry `--ignore-rules` (an execpolicy
# `.rules` allow rule runs a matching command outside the sandbox), and it
# must not name any option that widens or bypasses the sandbox or
# approvals (see is_widening_word).
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

# expected_invocation_count <skill>
# /plan carries exactly one codex-exec invocation line (step 11.c).
# /cross-review carries exactly two: the Step 4 fenced-block line and the
# "CLI execution modes" table row. A line split across a backslash
# continuation, or a removed invocation, changes this count.
expected_invocation_count() {
  case "$1" in
    plan) echo 1 ;;
    cross-review) echo 2 ;;
    *) echo 0 ;;
  esac
}

# is_widening_word <word>
# Succeeds when <word> (quotes already removed) is an option that widens or
# bypasses the sandbox or approvals. The list comes from `codex --help`,
# `codex exec --help` and `codex exec review --help` on codex-cli 0.154.0,
# plus options observed outside those helps (R-numbers are the probes in
# docs/reports/self-review-2026-10-03-cross-review-codex-read-only.md):
#   --dangerously-bypass-approvals-and-sandbox  no sandbox, no approvals
#                       (all three helps)
#   --yolo              hidden alias, in no help; 0.154.0 accepts it at all
#                       three levels and it beats a read-only selection (R9)
#   --approve-for-me    approvals go to automatic review in the
#                       workspace-write sandbox (root and exec helps; at the
#                       root it beats a read-only selection, R10)
#   --full-auto         older workspace-write shortcut, in no help; 0.154.0
#                       rejects it, listed so a codex that accepts it again
#                       cannot slip through
#   --add-dir           extra writable directories (root and exec helps)
#   -a / --ask-for-approval  approval policy (root help)
#   -p / --profile      layers a config file this test cannot see (root and
#                       exec helps)
#   --dangerously-bypass-hook-trust  runs hooks without persisted trust
#                       (all three helps)
# and the `-c` keys that set the same things: approval_policy,
# default_permissions (R4, R5), sandbox_permissions, sandbox_workspace_write.
is_widening_word() {
  case "$1" in
    --dangerously-bypass-approvals-and-sandbox|--yolo|--approve-for-me|--full-auto) return 0 ;;
    --add-dir|--add-dir=*|-a|--ask-for-approval|--ask-for-approval=*) return 0 ;;
    -p|--profile|--profile=*|--dangerously-bypass-hook-trust) return 0 ;;
    *approval_policy=*|*default_permissions=*|*sandbox_permissions=*|*sandbox_workspace_write*) return 0 ;;
  esac
  return 1
}

# check_sandbox_tokens <label> <content>
# Splits the codex command on the line (from `command codex ` up to its
# `</dev/null`, so the claude column of the cross-review table row is not
# read) into words, and checks that (i) it selects a sandbox and every
# selection (`--sandbox <v>`, `--sandbox=<v>`, `-s <v>`, `-s<v>`, or
# `sandbox_mode=<v>`) has v = read-only, and (ii) no word is an option that
# is_widening_word lists. Globbing is already off (the caller runs `set -f`
# around its line loop); IFS is set to blanks here and restored afterwards.
check_sandbox_tokens() {
  _sb_label="$1"
  _sb_cmd="${2#*command codex }"
  _sb_cmd="${_sb_cmd%%</dev/null*}"
  _sb_count=0
  _sb_bad=""
  _sb_widen=""
  _sb_prev=""
  _sb_ifs=$IFS
  IFS=' 	'
  for _sb_w in $_sb_cmd; do
    case "$_sb_w" in
      *\"*|*\'*) _sb_w="$(printf '%s' "$_sb_w" | tr -d "\"'")" ;;
    esac
    _sb_sel=0
    _sb_v=""
    case "$_sb_prev" in
      --sandbox|-s) _sb_sel=1; _sb_v="$_sb_w" ;;
    esac
    case "$_sb_w" in
      --sandbox=*) _sb_sel=1; _sb_v="${_sb_w#--sandbox=}" ;;
      -s?*) _sb_sel=1; _sb_v="${_sb_w#-s}" ;;
      *sandbox_mode=*) _sb_sel=1; _sb_v="${_sb_w#*sandbox_mode=}" ;;
    esac
    if [ "$_sb_sel" = 1 ]; then
      _sb_count=$((_sb_count + 1))
      [ "$_sb_v" = read-only ] || _sb_bad="$_sb_bad ${_sb_v:-<empty>}"
    fi
    if is_widening_word "$_sb_w"; then
      _sb_widen="$_sb_widen $_sb_w"
    fi
    _sb_prev="$_sb_w"
  done
  IFS=$_sb_ifs

  if [ "$_sb_count" -eq 0 ]; then
    fail "$_sb_label: selects no sandbox (want --sandbox read-only, or -c sandbox_mode=read-only for exec review) -- $2"
  elif [ -n "$_sb_bad" ]; then
    fail "$_sb_label: selects a sandbox other than read-only:$_sb_bad -- $2"
  else
    pass "$_sb_label: selects only the read-only sandbox ($_sb_count selection(s))"
  fi

  if [ -n "$_sb_widen" ]; then
    fail "$_sb_label: names an option that widens or bypasses the sandbox or approvals:$_sb_widen -- $2"
  else
    pass "$_sb_label: names no option that widens or bypasses the sandbox or approvals"
  fi
}

# check_invocation_line <face> <skill> <lineno> <content>
# Verifies one codex-exec invocation line carries the five required tokens
# from #184 and `--ignore-rules` from #197, then hands the sandbox checks
# to check_sandbox_tokens.
check_invocation_line() {
  face="$1"; skill="$2"; lineno="$3"; content="$4"
  label="$face/$skill/SKILL.md:$lineno"

  case "$content" in
    *'command codex '*) pass "$label: has command codex " ;;
    *) fail "$label: missing 'command codex ' -- $content" ;;
  esac

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

  case "$content" in
    *'--ignore-rules'*) pass "$label: has --ignore-rules" ;;
    *) fail "$label: missing --ignore-rules -- $content" ;;
  esac

  check_sandbox_tokens "$label" "$content"
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
    set -f
    for m in $model_matches; do
      fb="${m#*:-}"
      fb="${fb%\}}"
      if [ "$fb" = "$EXPECT_MODEL" ]; then
        pass "$label: RALPH_CODEX_REVIEWER_MODEL fallback matches ralph-config.sh default ($EXPECT_MODEL)"
      else
        fail "$label: RALPH_CODEX_REVIEWER_MODEL fallback = '$fb', want '$EXPECT_MODEL' (scripts/ralph-config.sh default)"
      fi
    done
    set +f
    IFS=$old_ifs
  fi

  effort_matches="$(grep -oE '\$\{RALPH_CODEX_REASONING_EFFORT:-[^}]*\}' "$file" || true)"
  if [ -z "$effort_matches" ]; then
    fail "$label: no \${RALPH_CODEX_REASONING_EFFORT:-...} fallback found"
  else
    old_ifs=$IFS
    IFS='
'
    set -f
    for m in $effort_matches; do
      fb="${m#*:-}"
      fb="${fb%\}}"
      if [ "$fb" = "$EXPECT_EFFORT" ]; then
        pass "$label: RALPH_CODEX_REASONING_EFFORT fallback matches ralph-config.sh default ($EXPECT_EFFORT)"
      else
        fail "$label: RALPH_CODEX_REASONING_EFFORT fallback = '$fb', want '$EXPECT_EFFORT' (scripts/ralph-config.sh default)"
      fi
    done
    set +f
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
  # and the token " exec " (a bare "exec" preceded and followed by a space).
  # The trailing space is deliberate: it excludes a word like "execution"
  # that only shares the substring "exec", the exact false match this
  # skill body's own prose used to trip before the wording was fixed
  # (see docs/reports/self-review-2026-09-29-codex-exec-stdin-and-model.md,
  # LOW "test (detector wording)"). Every real invocation line also starts
  # with `command codex `, checked separately in check_invocation_line.
  invocation_lines="$(grep -n 'codex' "$file" | grep ' exec ' || true)"

  actual_count=0
  if [ -n "$invocation_lines" ]; then
    actual_count="$(printf '%s\n' "$invocation_lines" | grep -c .)"
  fi
  expected_count="$(expected_invocation_count "$skill")"
  if [ "$actual_count" = "$expected_count" ]; then
    pass "$label: has exactly $expected_count codex exec invocation line(s)"
  else
    fail "$label: expected $expected_count codex exec invocation line(s), found $actual_count -- a line split (e.g. a backslash continuation) or a removed invocation changes this count"
  fi

  if [ -z "$invocation_lines" ]; then
    fail "$label: no codex exec invocation line found"
  else
    old_ifs=$IFS
    IFS='
'
    set -f
    for line in $invocation_lines; do
      lineno="${line%%:*}"
      content="${line#*:}"
      check_invocation_line "$face" "$skill" "$lineno" "$content"
    done
    set +f
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
