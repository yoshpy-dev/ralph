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
# Known limits:
#   - The pin-sentence check only covers the implementer pin. implementer is
#     the only seat with a "`model: <x>` pinned in frontmatter" sentence, so
#     every such sentence is compared with agents/implementer.md.
#   - A tier-table row naming a backticked word whose agent file was deleted is
#     ignored rather than failed: a backticked word that is not an agent cannot
#     be told apart from a removed agent. Deleting an agent file on one side
#     only is still caught by the root-vs-template comparison.
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
# Mutation target: any agent but implementer, so (a)-(g) never also trip the pin-sentence check.
TARGET_AGENT="tester"
# Removal target for the cross check: any agent other than TARGET_AGENT, so the cross cases touch two files.
CROSS_FIXTURE_AGENT="doc-maintainer"
# Name of the template-only agent added by self-test (i); must not exist on either side.
EXTRA_AGENT="self-test-extra-agent"

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
#   <model> <backticked token from the "Agents and typical work" column>...
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
# when any FAIL was printed. <tree> is the directory tree to inspect, so the
# self-test can aim them at a throwaway copy.
#
# POSIX sh has no `local`: check_side and its helpers share `_`-prefixed
# globals (_cdir, _routing_rel, _routing, _agents_dir, _assign, _impl_model,
# and loop variables). That is only safe because run_checker runs every
# checker in a `$(...)` subshell, so no state leaks into the caller or into
# the next checker. Always call a checker through run_checker.
# ---------------------------------------------------------------------------

# check_side <tree> <claude_dir>: the tier table, the pin sentence, and the
# agent frontmatter of one side must agree.
check_side() {
  _tree="$1"
  _cdir="$2"
  _routing_rel="$_cdir/rules/ralph/model-routing.md"
  _routing="$_tree/$_routing_rel"
  _agents_dir="$_tree/$_cdir/agents"
  _impl_model=""

  check_side_inputs || return 1

  _side_bad=0
  _rows="$(tier_rows "$_routing")"
  if [ -z "$_rows" ]; then
    printf 'FAIL: %s: no row with a backticked model under "## Tier table"\n' "$_routing_rel"
    _side_bad=1
  fi
  _assign="$(tier_assignments "$_rows")"
  check_agent_models || _side_bad=1
  check_pin_sentence || _side_bad=1
  return "$_side_bad"
}

# check_side_inputs: the side's model-routing.md and agents/ must exist.
check_side_inputs() {
  if [ ! -f "$_routing" ]; then
    printf 'FAIL: %s: file not found\n' "$_routing_rel"
    return 1
  fi
  if [ ! -d "$_agents_dir" ]; then
    printf 'FAIL: %s/agents: directory not found\n' "$_cdir"
    return 1
  fi
}

# tier_assignments <tier_rows output>: print one "<agent> <model>" line per
# (tier row, existing agent file) pair. A name repeated within one row counts
# once; backticked words without an agents/<word>.md are skipped.
tier_assignments() {
  _rows_in="$1"
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
      printf '%s %s\n' "$_name" "$_model"
    done
  done <<EOF
$_rows_in
EOF
}

# check_agent_models: every agents/*.md must pass check_one_agent, and the
# side must have at least one agent file.
check_agent_models() {
  _models_bad=0
  _agent_count=0
  for _f in "$_agents_dir"/*.md; do
    [ -f "$_f" ] || continue
    _agent_count=$((_agent_count + 1))
    check_one_agent "$_f" || _models_bad=1
  done
  if [ "$_agent_count" -eq 0 ]; then
    printf 'FAIL: %s/agents: no agent files found\n' "$_cdir"
    _models_bad=1
  fi
  return "$_models_bad"
}

# check_one_agent <agent.md>: the agent's frontmatter model must equal the
# model of the one tier row that names it. Records implementer's model in
# _impl_model for check_pin_sentence.
check_one_agent() {
  _name="$(basename "$1" .md)"
  _path="$_cdir/agents/$_name.md"
  _fm="$(frontmatter_model "$1")"
  if [ -z "$_fm" ]; then
    printf 'FAIL: %s (agent %s): frontmatter has no model: value (omitting it means inherit, which model-routing.md forbids)\n' "$_path" "$_name"
    return 1
  fi
  if [ "$_name" = "implementer" ]; then
    _impl_model="$_fm"
  fi
  _hits="$(printf '%s\n' "$_assign" | awk -v n="$_name" '$1 == n { c++ } END { print c + 0 }')"
  if [ "$_hits" -eq 0 ]; then
    printf 'FAIL: %s (agent %s): not listed in any "## Tier table" row of %s\n' "$_path" "$_name" "$_routing_rel"
    return 1
  fi
  if [ "$_hits" -gt 1 ]; then
    printf 'FAIL: %s (agent %s): listed in %s "## Tier table" rows of %s (must be exactly one)\n' "$_path" "$_name" "$_hits" "$_routing_rel"
    return 1
  fi
  _want="$(printf '%s\n' "$_assign" | awk -v n="$_name" '$1 == n { print $2 }')"
  if [ "$_fm" != "$_want" ]; then
    printf 'FAIL: %s (agent %s): frontmatter model %s but %s tier table says %s\n' "$_path" "$_name" "$_fm" "$_routing_rel" "$_want"
    return 1
  fi
  printf 'PASS: %s (agent %s): model %s matches the tier table\n' "$_path" "$_name" "$_fm"
}

# check_pin_sentence: every "`model: <x>` pinned in frontmatter" sentence must
# name the frontmatter model of agents/implementer.md (set by check_one_agent).
# The sentence is matched on newline-collapsed text so a re-wrap of the
# paragraph does not hide it.
check_pin_sentence() {
  _pins="$(tr '\n' ' ' < "$_routing" | grep -oE '`model: [^`]+` +pinned in +frontmatter' | sed -e 's/^`model: //' -e 's/`.*$//')"
  if [ -z "$_pins" ]; then
    printf 'FAIL: %s: sentence "model: <x> pinned in frontmatter" not found\n' "$_routing_rel"
    return 1
  fi
  if [ -z "$_impl_model" ]; then
    printf 'FAIL: %s: pin sentence cannot be checked because %s/agents/implementer.md has no frontmatter model\n' "$_routing_rel" "$_cdir"
    return 1
  fi
  _pin_bad=0
  while IFS= read -r _pin; do
    [ -n "$_pin" ] || continue
    if [ "$_pin" = "$_impl_model" ]; then
      printf 'PASS: %s: pin sentence model %s matches %s/agents/implementer.md\n' "$_routing_rel" "$_pin" "$_cdir"
    else
      printf 'FAIL: %s: pin sentence says model %s but %s/agents/implementer.md frontmatter model is %s\n' "$_routing_rel" "$_pin" "$_cdir" "$_impl_model"
      _pin_bad=1
    fi
  done <<EOF
$_pins
EOF
  return "$_pin_bad"
}

# check_cross <tree>: root and template must carry the same agent files with
# the same frontmatter model.
check_cross() {
  _tree="$1"
  _ra="$_tree/$ROOT_SIDE/agents"
  _ta="$_tree/$TEMPLATE_SIDE/agents"
  _cross_bad=0

  for _d in "$ROOT_SIDE" "$TEMPLATE_SIDE"; do
    if [ ! -d "$_tree/$_d/agents" ]; then
      printf 'FAIL: %s/agents: directory not found, cannot compare root and template agents\n' "$_d"
      _cross_bad=1
    fi
  done
  [ "$_cross_bad" -eq 0 ] || return 1

  for _f in "$_ra"/*.md; do
    [ -f "$_f" ] || continue
    _name="$(basename "$_f" .md)"
    if [ ! -f "$_ta/$_name.md" ]; then
      printf 'FAIL: %s/agents/%s.md (agent %s): missing from %s/agents\n' "$ROOT_SIDE" "$_name" "$_name" "$TEMPLATE_SIDE"
      _cross_bad=1
      continue
    fi
    _rm="$(frontmatter_model "$_f")"
    _tm="$(frontmatter_model "$_ta/$_name.md")"
    # Both sides lacking model: prints nothing: check_side already fails each side.
    if [ "$_rm" != "$_tm" ]; then
      printf 'FAIL: %s/agents/%s.md vs %s/agents/%s.md (agent %s): frontmatter model %s (root) != %s (template)\n' "$ROOT_SIDE" "$_name" "$TEMPLATE_SIDE" "$_name" "$_name" "${_rm:-<none>}" "${_tm:-<none>}"
      _cross_bad=1
    elif [ -n "$_rm" ]; then
      printf 'PASS: agent %s: root and template frontmatter model both %s\n' "$_name" "$_rm"
    fi
  done
  for _f in "$_ta"/*.md; do
    [ -f "$_f" ] || continue
    _name="$(basename "$_f" .md)"
    if [ ! -f "$_ra/$_name.md" ]; then
      printf 'FAIL: %s/agents/%s.md (agent %s): missing from %s/agents\n' "$TEMPLATE_SIDE" "$_name" "$_name" "$ROOT_SIDE"
      _cross_bad=1
    fi
  done

  return "$_cross_bad"
}

# run_checker <function> [args...]: run a checker in a subshell (the isolation
# the checkers' shared globals rely on); sets _out/_rc.
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

# require_fixture_agent <claude_dir> <name>: record an explicit setup FAIL when
# a self-test fixture agent is missing from the real tree (renamed or removed),
# instead of letting a mutation miss without explanation.
require_fixture_agent() {
  if [ -f "$PROJECT_ROOT/$1/agents/$2.md" ]; then
    return 0
  fi
  record_fail "self-test fixture agent $2 not found under $1/agents"
  return 1
}

# add_fixture_agent <claude_dir> <name>: add agents/<name>.md to FIXTURE only
# (a copy of TARGET_AGENT); fails the setup when <name> exists in the real tree.
add_fixture_agent() {
  for _as in "$ROOT_SIDE" "$TEMPLATE_SIDE"; do
    if [ -e "$PROJECT_ROOT/$_as/agents/$2.md" ]; then
      record_fail "self-test setup: $_as/agents/$2.md exists in the real tree; rename EXTRA_AGENT"
      return 1
    fi
  done
  cp "$FIXTURE/$1/agents/$TARGET_AGENT.md" "$FIXTURE/$1/agents/$2.md"
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

# mut_drop_pin <file>: delete the pin sentence.
mut_drop_pin() {
  sed 's/`model: [^`]*` pinned in frontmatter//' "$1" > "$1.tmp" && mv "$1.tmp" "$1"
}

# mut_remove <path>: delete a fixture file or directory.
mut_remove() {
  rm -rf "$1"
}

# mutate <relative-path> <mutator> [args...]: apply a mutator to FIXTURE/<path>.
# Records a setup FAIL and returns 1 when <path> is not in the fixture, or when
# the mutation left it identical to the real one, so a stale mutation cannot
# masquerade as a detection.
mutate() {
  _mrel="$1"
  _mfn="$2"
  shift 2
  if [ ! -e "$FIXTURE/$_mrel" ]; then
    record_fail "self-test setup: $_mrel not found in the fixture copy"
    return 1
  fi
  "$_mfn" "$FIXTURE/$_mrel" "$@"
  # A removal is a change by definition; anything still present must differ.
  [ -e "$FIXTURE/$_mrel" ] || return 0
  if [ -d "$FIXTURE/$_mrel" ] || cmp -s "$FIXTURE/$_mrel" "$PROJECT_ROOT/$_mrel" 2> /dev/null; then
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

# self_test_guards: the fixture guard must name a missing fixture agent.
self_test_guards() {
  _guard_out="$(require_fixture_agent "$ROOT_SIDE" "no-such-agent" || :)"
  case "$_guard_out" in
    *"FAIL: self-test fixture agent no-such-agent not found under $ROOT_SIDE/agents"*)
      record_pass "fixture guard names a missing fixture agent and its directory"
      ;;
    *) record_fail "fixture guard did not report a missing fixture agent: $_guard_out" ;;
  esac
}

# self_test_side_drift <claude_dir>: table, frontmatter, and pin drift on one side.
self_test_side_drift() {
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
  if mutate "$_routing_rel" mut_drop_pin; then
    run_checker check_side "$FIXTURE" "$_side"
    assert_detected "[$_side] (h) pin sentence removed" "$_rc" "$_out" "FAIL: $_routing_rel: sentence \"model: <x> pinned in frontmatter\" not found"
  fi
}

# self_test_side_edges <claude_dir>: missing values, duplicates, ignored words,
# and missing inputs on one side.
self_test_side_edges() {
  _side="$1"
  _agent_rel="$_side/agents/$TARGET_AGENT.md"
  _routing_rel="$_side/rules/ralph/model-routing.md"

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

  new_fixture
  if mutate "$_side/agents" mut_remove; then
    run_checker check_side "$FIXTURE" "$_side"
    assert_detected "[$_side] (j) agents directory missing" "$_rc" "$_out" "FAIL: $_side/agents: directory not found"
  fi

  new_fixture
  if mutate "$_routing_rel" mut_remove; then
    run_checker check_side "$FIXTURE" "$_side"
    assert_detected "[$_side] (j) model-routing.md missing" "$_rc" "$_out" "FAIL: $_routing_rel: file not found"
  fi
}

self_test_side() {
  require_fixture_agent "$1" "$TARGET_AGENT" || return 0
  self_test_side_drift "$1"
  self_test_side_edges "$1"
}

# self_test_cross_models: frontmatter model drift between the two sides.
self_test_cross_models() {
  new_fixture
  run_checker check_cross "$FIXTURE"
  assert_clean "baseline copy passes root-vs-template comparison" "$_rc" "$_out"

  new_fixture
  _tcur="$(frontmatter_model "$PROJECT_ROOT/$TEMPLATE_SIDE/agents/$TARGET_AGENT.md")"
  if mutate "$TEMPLATE_SIDE/agents/$TARGET_AGENT.md" mut_set_fm_model "$(other_model "$_tcur")"; then
    run_checker check_cross "$FIXTURE"
    assert_detected "template-only agent model change (root vs template)" "$_rc" "$_out" "$ROOT_SIDE/agents/$TARGET_AGENT.md" "$TEMPLATE_SIDE/agents/$TARGET_AGENT.md" "$TARGET_AGENT"
  fi

  # check_side fails an agent with no model: on each side; the comparison must
  # not count "both <none>" as a match.
  new_fixture
  if mutate "$ROOT_SIDE/agents/$TARGET_AGENT.md" mut_drop_fm_model &&
    mutate "$TEMPLATE_SIDE/agents/$TARGET_AGENT.md" mut_drop_fm_model; then
    run_checker check_cross "$FIXTURE"
    case "$_out" in
      *"PASS: agent $TARGET_AGENT:"*)
        record_fail "agent with no model: on both sides: comparison printed a PASS line"
        ;;
      *) record_pass "agent with no model: on both sides gets no PASS line in the root-vs-template comparison" ;;
    esac
  fi
}

# self_test_cross_files: agent files or the agents directory present on one side only.
self_test_cross_files() {
  new_fixture
  if mutate "$TEMPLATE_SIDE/agents/$CROSS_FIXTURE_AGENT.md" mut_remove; then
    run_checker check_cross "$FIXTURE"
    assert_detected "agent file missing from template (root vs template)" "$_rc" "$_out" "$TEMPLATE_SIDE/agents" "$CROSS_FIXTURE_AGENT"
  fi

  new_fixture
  if add_fixture_agent "$TEMPLATE_SIDE" "$EXTRA_AGENT"; then
    run_checker check_cross "$FIXTURE"
    assert_detected "(i) agent file only in template (root vs template)" "$_rc" "$_out" "$TEMPLATE_SIDE/agents/$EXTRA_AGENT.md" "missing from $ROOT_SIDE/agents"
  fi

  for _xs in "$ROOT_SIDE" "$TEMPLATE_SIDE"; do
    new_fixture
    if mutate "$_xs/agents" mut_remove; then
      run_checker check_cross "$FIXTURE"
      assert_detected "(j) $_xs/agents missing (root vs template)" "$_rc" "$_out" "FAIL: $_xs/agents: directory not found"
    fi
  done
}

self_test_cross() {
  require_fixture_agent "$ROOT_SIDE" "$TARGET_AGENT" || return 0
  require_fixture_agent "$TEMPLATE_SIDE" "$TARGET_AGENT" || return 0
  require_fixture_agent "$TEMPLATE_SIDE" "$CROSS_FIXTURE_AGENT" || return 0
  self_test_cross_models
  self_test_cross_files
}

printf '\n== Self-test: mutations on throwaway copies ==\n'
if [ "$_fail" -gt 0 ]; then
  # The mutations are measured against a consistent baseline copy of the real
  # tree; with the real tree already failing, their results would only add noise.
  printf '  SKIP: real tree has %d failure(s) above; fix them to run the mutation self-test\n' "$_fail"
else
  self_test_guards
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
