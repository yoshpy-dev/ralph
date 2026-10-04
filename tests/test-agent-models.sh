#!/usr/bin/env sh
# Regression guard: the subagent model tiers must stay in lock-step across
#   - the "## Tier table" in model-routing.md,
#   - the "`model: <x>` pinned in frontmatter" sentence in model-routing.md,
#   - the `model:` frontmatter of every agents/*.md,
# on BOTH the root side (.claude/) and the scaffold template side
# (templates/base/.claude/), and the two sides' agent models must match.
#
# scripts/check-sync.sh treats model-routing.md as a known diff, so this test
# is the only guard for the template copy of the table.
#
# Expected values are never hard-coded: the table is the declaration, the
# frontmatter is the implementation, and the test fails when they disagree.
# Changing a seat's model means editing the table and the frontmatter together.
#
# The self-test section applies one kind of drift at a time to throwaway
# copies of the real files and asserts the checker fails and names the
# offending file and agent. It never writes to the real tree.

set -eu

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
cd "$PROJECT_ROOT"

ROOT_SIDE=".claude"
TEMPLATE_SIDE="templates/base/.claude"
TARGET_AGENT="tester"
NL='
'

_pass=0
_fail=0
_total=0
_case_n=0

record_pass() {
  _pass=$((_pass + 1))
  _total=$((_total + 1))
  printf '  PASS: %s\n' "$1"
}

record_fail() {
  _fail=$((_fail + 1))
  _total=$((_total + 1))
  printf '  FAIL: %s\n' "$1"
}

TMP_ROOT="$(mktemp -d "${TMPDIR:-/tmp}/test-agent-models.XXXXXX")"
trap 'rm -rf "$TMP_ROOT"' EXIT
trap 'exit 1' HUP INT TERM

# ---------------------------------------------------------------------------
# Parsing helpers
# ---------------------------------------------------------------------------

# frontmatter_model <agent.md>: print the `model:` value found between the
# first two `---` lines (nothing when there is no frontmatter or no model).
frontmatter_model() {
  awk '
    NR == 1 { if ($0 != "---") exit; next }
    $0 == "---" { exit }
    /^model:/ {
      sub(/^model:[ \t]*/, "")
      sub(/[ \t\r]+$/, "")
      print
      exit
    }
  ' "$1"
}

# tier_rows <model-routing.md>: for each row of the "## Tier table" section
# whose Model column holds a backticked token, print one line:
#   <model> <backticked token from the Examples column>...
# Header, separator, and rows without a backticked model (the Orchestrator
# row) print nothing.
tier_rows() {
  awk '
    /^## / {
      if (in_table) exit
      in_table = ($0 ~ /^## Tier table[ \t]*$/)
      next
    }
    in_table && /^\|/ {
      n = split($0, c, "|")
      if (n < 4) next
      if (!match(c[3], /`[^`]+`/)) next
      line = substr(c[3], RSTART + 1, RLENGTH - 2)
      rest = c[4]
      while (match(rest, /`[^`]+`/)) {
        line = line " " substr(rest, RSTART + 1, RLENGTH - 2)
        rest = substr(rest, RSTART + RLENGTH)
      }
      print line
    }
  ' "$1"
}

# ---------------------------------------------------------------------------
# Checkers. They print "PASS: <msg>" / "FAIL: <msg>" lines and return non-zero
# when any FAIL was printed. <root> is the tree to inspect, so the self-test
# can aim them at a throwaway copy.
# ---------------------------------------------------------------------------

# check_side <root> <claude_dir>
check_side() {
  _root="$1"
  _cdir="$2"
  _routing_rel="$_cdir/rules/ralph/model-routing.md"
  _routing="$_root/$_routing_rel"
  _agents_dir="$_root/$_cdir/agents"
  _bad=0
  _impl_model=""
  _assign=""

  if [ ! -f "$_routing" ]; then
    printf 'FAIL: %s: file not found\n' "$_routing_rel"
    return 1
  fi
  if [ ! -d "$_agents_dir" ]; then
    printf 'FAIL: %s/agents: directory not found\n' "$_cdir"
    return 1
  fi

  # One "<agent> <model>" line per (tier row, existing agent) pair.
  _rows="$(tier_rows "$_routing")"
  if [ -z "$_rows" ]; then
    printf 'FAIL: %s: no row with a backticked model under "## Tier table"\n' "$_routing_rel"
    _bad=1
  fi
  while IFS= read -r _line; do
    [ -n "$_line" ] || continue
    set -f
    # shellcheck disable=SC2086
    set -- $_line
    set +f
    _model="$1"
    shift
    _seen=" "
    for _name in "$@"; do
      case "$_name" in
        *[!A-Za-z0-9_-]*) continue ;;
      esac
      [ -f "$_agents_dir/$_name.md" ] || continue
      case "$_seen" in
        *" $_name "*) continue ;;
      esac
      _seen="$_seen$_name "
      _assign="$_assign$_name $_model$NL"
    done
  done <<EOF
$_rows
EOF

  _agent_count=0
  for _f in "$_agents_dir"/*.md; do
    [ -f "$_f" ] || continue
    _agent_count=$((_agent_count + 1))
    _name="$(basename "$_f" .md)"
    _path="$_cdir/agents/$_name.md"
    _fm="$(frontmatter_model "$_f")"
    if [ -z "$_fm" ]; then
      printf 'FAIL: %s (agent %s): frontmatter has no model: value (omitting it means inherit, which model-routing.md forbids)\n' "$_path" "$_name"
      _bad=1
      continue
    fi
    if [ "$_name" = "implementer" ]; then
      _impl_model="$_fm"
    fi
    _hits="$(printf '%s\n' "$_assign" | awk -v n="$_name" '$1 == n { c++ } END { print c + 0 }')"
    if [ "$_hits" -eq 0 ]; then
      printf 'FAIL: %s (agent %s): not listed in any "## Tier table" row of %s\n' "$_path" "$_name" "$_routing_rel"
      _bad=1
    elif [ "$_hits" -gt 1 ]; then
      printf 'FAIL: %s (agent %s): listed in %s "## Tier table" rows of %s (must be exactly one)\n' "$_path" "$_name" "$_hits" "$_routing_rel"
      _bad=1
    else
      _want="$(printf '%s\n' "$_assign" | awk -v n="$_name" '$1 == n { print $2 }')"
      if [ "$_fm" = "$_want" ]; then
        printf 'PASS: %s (agent %s): model %s matches the tier table\n' "$_path" "$_name" "$_fm"
      else
        printf 'FAIL: %s (agent %s): frontmatter model %s but %s tier table says %s\n' "$_path" "$_name" "$_fm" "$_routing_rel" "$_want"
        _bad=1
      fi
    fi
  done
  if [ "$_agent_count" -eq 0 ]; then
    printf 'FAIL: %s/agents: no agent files found\n' "$_cdir"
    _bad=1
  fi

  # The pin sentence is matched on newline-collapsed text so a re-wrap of the
  # paragraph does not hide it.
  _pins="$(tr '\n' ' ' < "$_routing" | grep -oE '`model: [^`]+` +pinned in +frontmatter' | sed -e 's/^`model: //' -e 's/`.*$//')"
  if [ -z "$_pins" ]; then
    printf 'FAIL: %s: sentence "model: <x> pinned in frontmatter" not found\n' "$_routing_rel"
    _bad=1
  elif [ -z "$_impl_model" ]; then
    printf 'FAIL: %s: pin sentence cannot be checked because %s/agents/implementer.md has no frontmatter model\n' "$_routing_rel" "$_cdir"
    _bad=1
  else
    while IFS= read -r _pin; do
      [ -n "$_pin" ] || continue
      if [ "$_pin" = "$_impl_model" ]; then
        printf 'PASS: %s: pin sentence model %s matches %s/agents/implementer.md\n' "$_routing_rel" "$_pin" "$_cdir"
      else
        printf 'FAIL: %s: pin sentence says model %s but %s/agents/implementer.md frontmatter model is %s\n' "$_routing_rel" "$_pin" "$_cdir" "$_impl_model"
        _bad=1
      fi
    done <<EOF
$_pins
EOF
  fi

  return "$_bad"
}

# check_cross <root>: root and template must carry the same agent files with
# the same frontmatter model.
check_cross() {
  _root="$1"
  _ra="$_root/$ROOT_SIDE/agents"
  _ta="$_root/$TEMPLATE_SIDE/agents"
  _bad=0

  for _f in "$_ra"/*.md; do
    [ -f "$_f" ] || continue
    _name="$(basename "$_f" .md)"
    if [ ! -f "$_ta/$_name.md" ]; then
      printf 'FAIL: %s/agents/%s.md (agent %s): missing from %s/agents\n' "$ROOT_SIDE" "$_name" "$_name" "$TEMPLATE_SIDE"
      _bad=1
      continue
    fi
    _rm="$(frontmatter_model "$_f")"
    _tm="$(frontmatter_model "$_ta/$_name.md")"
    if [ "$_rm" = "$_tm" ]; then
      printf 'PASS: agent %s: root and template frontmatter model both %s\n' "$_name" "${_rm:-<none>}"
    else
      printf 'FAIL: %s/agents/%s.md vs %s/agents/%s.md (agent %s): frontmatter model %s (root) != %s (template)\n' "$ROOT_SIDE" "$_name" "$TEMPLATE_SIDE" "$_name" "$_name" "${_rm:-<none>}" "${_tm:-<none>}"
      _bad=1
    fi
  done
  for _f in "$_ta"/*.md; do
    [ -f "$_f" ] || continue
    _name="$(basename "$_f" .md)"
    if [ ! -f "$_ra/$_name.md" ]; then
      printf 'FAIL: %s/agents/%s.md (agent %s): missing from %s/agents\n' "$TEMPLATE_SIDE" "$_name" "$_name" "$ROOT_SIDE"
      _bad=1
    fi
  done

  return "$_bad"
}

# run_checker <function> [args...]: run a checker in a subshell; sets _out/_rc.
run_checker() {
  if _out="$("$@")"; then
    _rc=0
  else
    _rc=$?
  fi
}

# emit_results <label> <rc> <output>: turn checker output into counted results.
emit_results() {
  _label="$1"
  _erc="$2"
  _eout="$3"
  _saw_fail=0
  while IFS= read -r _line; do
    case "$_line" in
      "PASS: "*) record_pass "${_line#PASS: }" ;;
      "FAIL: "*)
        record_fail "${_line#FAIL: }"
        _saw_fail=1
        ;;
      "") ;;
      *) record_fail "$_label: unexpected checker output: $_line" ;;
    esac
  done <<EOF
$_eout
EOF
  if [ "$_erc" -ne 0 ] && [ "$_saw_fail" -eq 0 ]; then
    record_fail "$_label: checker exited with status $_erc without reporting a failure"
  fi
}

# ---------------------------------------------------------------------------
# Real tree
# ---------------------------------------------------------------------------

printf '== Tier table, pin sentence, and agent frontmatter (real tree) ==\n'
for _side in "$ROOT_SIDE" "$TEMPLATE_SIDE"; do
  run_checker check_side "$PROJECT_ROOT" "$_side"
  emit_results "$_side" "$_rc" "$_out"
done
run_checker check_cross "$PROJECT_ROOT"
emit_results "root vs template" "$_rc" "$_out"

# ---------------------------------------------------------------------------
# Self-test: mutate throwaway copies, never the real tree
# ---------------------------------------------------------------------------

# new_fixture: copy the four inputs of each side into a fresh temp tree; sets FIXTURE.
new_fixture() {
  _case_n=$((_case_n + 1))
  FIXTURE="$TMP_ROOT/case-$_case_n"
  for _s in "$ROOT_SIDE" "$TEMPLATE_SIDE"; do
    mkdir -p "$FIXTURE/$_s/rules/ralph"
    cp -R "$PROJECT_ROOT/$_s/agents" "$FIXTURE/$_s/agents"
    cp "$PROJECT_ROOT/$_s/rules/ralph/model-routing.md" "$FIXTURE/$_s/rules/ralph/model-routing.md"
  done
}

other_model() {
  case "$1" in
    opus) printf 'sonnet' ;;
    *) printf 'opus' ;;
  esac
}

# Mutators: <file> [args]. Each rewrites the file through a temp file.

mut_set_fm_model() {
  awk -v m="$2" '
    NR == 1 { infm = 1; print; next }
    infm && $0 == "---" { infm = 0 }
    infm && /^model:/ && !done { print "model: " m; done = 1; next }
    { print }
  ' "$1" > "$1.tmp" && mv "$1.tmp" "$1"
}

mut_drop_fm_model() {
  awk '
    NR == 1 { infm = 1; print; next }
    infm && $0 == "---" { infm = 0 }
    infm && /^model:/ && !done { done = 1; next }
    { print }
  ' "$1" > "$1.tmp" && mv "$1.tmp" "$1"
}

# mut_tier_model <file> <agent> <model>: change the Model of the row naming <agent>.
mut_tier_model() {
  awk -v a="$2" -v m="$3" '
    /^## / { in_table = ($0 ~ /^## Tier table[ \t]*$/) }
    in_table && /^\|/ && index($0, "`" a "`") && !done {
      n = split($0, c, "|")
      sub(/`[^`]+`/, "`" m "`", c[3])
      line = c[1]
      for (i = 2; i <= n; i++) line = line "|" c[i]
      print line
      done = 1
      next
    }
    { print }
  ' "$1" > "$1.tmp" && mv "$1.tmp" "$1"
}

# mut_tier_drop_agent <file> <agent>: remove <agent> from the tier table rows.
mut_tier_drop_agent() {
  awk -v a="$2" '
    /^## / { in_table = ($0 ~ /^## Tier table[ \t]*$/) }
    in_table && /^\|/ { gsub("`" a "`", "") }
    { print }
  ' "$1" > "$1.tmp" && mv "$1.tmp" "$1"
}

# mut_tier_dup_agent <file> <agent>: also list <agent> in a row that lacks it.
mut_tier_dup_agent() {
  awk -v a="$2" '
    /^## / { in_table = ($0 ~ /^## Tier table[ \t]*$/) }
    in_table && /^\|/ && !done && !index($0, "`" a "`") {
      n = split($0, c, "|")
      if (n >= 4 && match(c[3], /`[^`]+`/)) {
        sub(/ *$/, ", `" a "` ", c[4])
        line = c[1]
        for (i = 2; i <= n; i++) line = line "|" c[i]
        print line
        done = 1
        next
      }
    }
    { print }
  ' "$1" > "$1.tmp" && mv "$1.tmp" "$1"
}

# mut_tier_add_unknown <file>: add backticked non-agent words to the first row.
mut_tier_add_unknown() {
  awk '
    /^## / { in_table = ($0 ~ /^## Tier table[ \t]*$/) }
    in_table && /^\|/ && !done {
      n = split($0, c, "|")
      if (n >= 4 && match(c[3], /`[^`]+`/)) {
        sub(/ *$/, ", `not-an-agent` and `design` ", c[4])
        line = c[1]
        for (i = 2; i <= n; i++) line = line "|" c[i]
        print line
        done = 1
        next
      }
    }
    { print }
  ' "$1" > "$1.tmp" && mv "$1.tmp" "$1"
}

# mut_pin <file> <model>: rewrite the pin sentence.
mut_pin() {
  sed 's/`model: [^`]*` pinned in frontmatter/`model: '"$2"'` pinned in frontmatter/' "$1" > "$1.tmp" && mv "$1.tmp" "$1"
}

mut_remove_file() {
  rm -f "$1"
}

# mutate <relative-path> <mutator> [args...]: apply a mutator to FIXTURE/<path>;
# fails (and records a FAIL) when the mutation left the file identical to the
# real one, so a stale mutation cannot masquerade as a detection.
mutate() {
  _mrel="$1"
  _mfn="$2"
  shift 2
  "$_mfn" "$FIXTURE/$_mrel" "$@"
  if cmp -s "$FIXTURE/$_mrel" "$PROJECT_ROOT/$_mrel" 2> /dev/null; then
    record_fail "self-test setup: $_mfn left $_mrel unchanged (mutation did not apply)"
    return 1
  fi
}

# assert_detected <description> <rc> <output> <needle>...
assert_detected() {
  _desc="$1"
  _arc="$2"
  _aout="$3"
  shift 3
  if [ "$_arc" -eq 0 ]; then
    record_fail "$_desc: checker passed but should have failed"
    return 0
  fi
  case "$_aout" in
    *"FAIL: "*) ;;
    *)
      record_fail "$_desc: checker exited $_arc without printing a FAIL line"
      return 0
      ;;
  esac
  for _needle in "$@"; do
    if ! printf '%s\n' "$_aout" | grep -Fq -- "$_needle"; then
      record_fail "$_desc: failure output does not name '$_needle'"
      return 0
    fi
  done
  record_pass "$_desc: detected (output names $*)"
}

# assert_clean <description> <rc> <output>
assert_clean() {
  if [ "$2" -eq 0 ]; then
    record_pass "$1"
  else
    record_fail "$1: checker failed unexpectedly: $3"
  fi
}

self_test_side() {
  _side="$1"
  _agent_rel="$_side/agents/$TARGET_AGENT.md"
  _routing_rel="$_side/rules/ralph/model-routing.md"
  _cur="$(frontmatter_model "$PROJECT_ROOT/$_agent_rel")"
  _impl_cur="$(frontmatter_model "$PROJECT_ROOT/$_side/agents/implementer.md")"

  new_fixture
  run_checker check_side "$FIXTURE" "$_side"
  assert_clean "[$_side] baseline copy passes" "$_rc" "$_out"

  new_fixture
  if mutate "$_agent_rel" mut_set_fm_model "$(other_model "$_cur")"; then
    run_checker check_side "$FIXTURE" "$_side"
    assert_detected "[$_side] (a) agent model changed" "$_rc" "$_out" "$_agent_rel" "$TARGET_AGENT"
  fi

  new_fixture
  if mutate "$_routing_rel" mut_tier_model "$TARGET_AGENT" "$(other_model "$_cur")"; then
    run_checker check_side "$FIXTURE" "$_side"
    assert_detected "[$_side] (b) tier row model changed" "$_rc" "$_out" "$_agent_rel" "$TARGET_AGENT"
  fi

  new_fixture
  if mutate "$_routing_rel" mut_tier_drop_agent "$TARGET_AGENT"; then
    run_checker check_side "$FIXTURE" "$_side"
    assert_detected "[$_side] (c) agent name removed from tier table" "$_rc" "$_out" "$_agent_rel" "$TARGET_AGENT"
  fi

  new_fixture
  if mutate "$_routing_rel" mut_pin "$(other_model "$_impl_cur")"; then
    run_checker check_side "$FIXTURE" "$_side"
    assert_detected "[$_side] (d) pin sentence reverted" "$_rc" "$_out" "$_routing_rel" "implementer"
  fi

  new_fixture
  if mutate "$_agent_rel" mut_drop_fm_model; then
    run_checker check_side "$FIXTURE" "$_side"
    assert_detected "[$_side] (e) agent frontmatter lacks a model line" "$_rc" "$_out" "$_agent_rel" "$TARGET_AGENT"
  fi

  new_fixture
  if mutate "$_routing_rel" mut_tier_dup_agent "$TARGET_AGENT"; then
    run_checker check_side "$FIXTURE" "$_side"
    assert_detected "[$_side] (f) agent listed in two tier rows" "$_rc" "$_out" "$_agent_rel" "$TARGET_AGENT"
  fi

  new_fixture
  if mutate "$_routing_rel" mut_tier_add_unknown; then
    run_checker check_side "$FIXTURE" "$_side"
    assert_clean "[$_side] (g) backticked words that are not agent files are ignored" "$_rc" "$_out"
  fi
}

self_test_cross() {
  new_fixture
  run_checker check_cross "$FIXTURE"
  assert_clean "baseline copy passes root-vs-template comparison" "$_rc" "$_out"

  new_fixture
  _tcur="$(frontmatter_model "$PROJECT_ROOT/$TEMPLATE_SIDE/agents/$TARGET_AGENT.md")"
  if mutate "$TEMPLATE_SIDE/agents/$TARGET_AGENT.md" mut_set_fm_model "$(other_model "$_tcur")"; then
    run_checker check_cross "$FIXTURE"
    assert_detected "template-only agent model change (root vs template)" "$_rc" "$_out" "$ROOT_SIDE/agents/$TARGET_AGENT.md" "$TEMPLATE_SIDE/agents/$TARGET_AGENT.md" "$TARGET_AGENT"
  fi

  new_fixture
  if mutate "$TEMPLATE_SIDE/agents/doc-maintainer.md" mut_remove_file; then
    run_checker check_cross "$FIXTURE"
    assert_detected "agent file missing from template (root vs template)" "$_rc" "$_out" "$TEMPLATE_SIDE/agents" "doc-maintainer"
  fi
}

printf '\n== Self-test: mutations on throwaway copies ==\n'
if [ "$_fail" -gt 0 ]; then
  # The mutations are measured against a consistent baseline copy of the real
  # tree; with the real tree already failing, their results would only add noise.
  printf '  SKIP: real tree has %d failure(s) above; fix them to run the mutation self-test\n' "$_fail"
else
  self_test_side "$ROOT_SIDE"
  self_test_side "$TEMPLATE_SIDE"
  self_test_cross
fi

printf '\n-- Summary --\n'
printf '  PASS: %d / %d\n' "$_pass" "$_total"
printf '  FAIL: %d\n' "$_fail"

if [ "$_fail" -gt 0 ]; then
  exit 1
fi
