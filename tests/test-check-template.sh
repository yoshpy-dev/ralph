#!/usr/bin/env bash
# tests/test-check-template.sh — regression coverage for scripts/check-template.sh's
# required_files list, its settings hook-reference check, its find-driven
# loops, and a fresh `ralph init` scaffold.
#
# Without this test, an entry can be silently dropped from required_files
# and no automated check notices: #169's /test mutation 6f removed
# scripts/xreview-helpers.sh from the list and nothing failed. This test
# closes that gap, plus the fail-open regressions in the hook-reference
# check and the three find-driven loops, and the required_files drift that
# broke every fresh `ralph init` scaffold's own CI, all fixed for #189,
# with these cases:
#   A. golden: the required_files block in scripts/check-template.sh must
#      match a hardcoded golden list, in order.
#   B. fixture: a synthetic project with every golden file present passes
#      check-template.sh; removing any one golden file (one at a time) makes
#      check-template.sh fail with "Missing required file: <entry>".
#   C. root sanity: every golden entry exists at this repo's own root.
#   D. root sanity: `CI=true sh scripts/check-template.sh` from the repo root
#      exits 0 with no FAIL line.
#   E. settings hook-reference check: an existing hook referenced with an
#      argument passes; a missing hook fails with the path only (no
#      argument) and exit 1; an unreadable settings.json is reported
#      instead of silently skipping the check (skipped when running as
#      root); a dispatcher missing across several events produces exactly
#      one FAIL line.
#   F. the three find-driven loops (executable check, SKILL.md check, agent
#      frontmatter check) each propagate a failure to the exit code, a
#      script path containing a space is still checked correctly, an
#      unreadable subtree is reported instead of silently skipping the
#      files inside it (skipped when running as root), the user-local
#      .claude/hooks/local/ tree is pruned rather than merely filtered
#      (so an unreadable path inside it never reaches find's own exit
#      code, skipped when running as root) while scripts elsewhere under
#      .claude/hooks/ are still checked, a `.claude/hooks/local/` without
#      its own execute bit does not leak into the printed results (pins
#      the find expression's `-print` placement), and an unreadable
#      subtree under `.claude/agents/` is reported the same way as the
#      scripts/ case, and an unreadable `.claude/skills/` itself (where the
#      skills listing find fails, unlike an unreadable child directory,
#      which the SKILL.md check reports) is reported instead of silently
#      passing (the last three skipped when running as root).
#   G. fresh scaffold: `go run ./cmd/ralph init --yes <tmp>` (when `go` is
#      available) passes check-template.sh with no FAIL line.
#   H. a project missing a whole search root (`packs/`, `.claude/agents/`,
#      or `.claude/skills/`) passes with no FAIL line; a missing root is
#      skipped, not reported.
#   I. signal cleanup (dash only, skipped when `dash` is unavailable): a
#      TERM, INT, or HUP sent to check-template.sh while it is parked in a
#      stub `find` makes it exit 143, 130, or 129 and leaves no
#      check-template.* directory in its TMPDIR. dash skips the EXIT trap
#      when a signal kills it, so only the script's own signal traps
#      remove the temp directory there.
#
# Spec: issue #183, issue #189. The Go-side counterpart lives in
# internal/scaffold/embed_test.go (TestTemplateBaseScriptsMatchCheckTemplateRequiredFiles).

set -u

REPO_ROOT="$(cd "$(dirname "$0")/.." && pwd)"
CHECK_TEMPLATE="$REPO_ROOT/scripts/check-template.sh"

if [ ! -f "$CHECK_TEMPLATE" ]; then
  echo "FAIL: $CHECK_TEMPLATE not found"
  exit 1
fi

pass_count=0
fail_count=0
skip_count=0

pass() {
  pass_count=$((pass_count + 1))
  printf '  PASS: %s\n' "$1"
}

fail() {
  fail_count=$((fail_count + 1))
  printf '  FAIL: %s\n' "$1"
}

# skip <description> — prints a SKIP line and counts it separately from
# pass/fail (same convention as tests/test-secret-scan-branch.sh's inline
# SKIP lines). Used for cases whose fixture only proves anything on a
# non-root user (chmod 000 does not block root's own read access) and for
# case G when `go` is unavailable.
skip() {
  skip_count=$((skip_count + 1))
  printf '  SKIP: %s\n' "$1"
}

# The golden required_files list: the 3 non-script entries in their current
# order, followed by the 22 scripts/ entries, ordered here to match
# internal/scaffold/embed_test.go's requiredTemplateScripts as a
# readability convention only (the Go test compares both as sets, so
# reordering either list does not by itself fail anything). Update this
# list in the same commit as any change to required_files in
# scripts/check-template.sh, templates/base/scripts/check-template.sh, and
# requiredTemplateScripts in internal/scaffold/embed_test.go.
GOLDEN_ENTRIES=(
  "AGENTS.md"
  "CLAUDE.md"
  ".claude/settings.json"
  "scripts/run-verify.sh"
  "scripts/run-static-verify.sh"
  "scripts/run-test.sh"
  "scripts/detect-changed-languages.sh"
  "scripts/detect-languages.sh"
  "scripts/archive-plan.sh"
  "scripts/branch-name.sh"
  "scripts/ensure-pr-ready.sh"
  "scripts/ensure-pr-title-prefix.sh"
  "scripts/new-feature-plan.sh"
  "scripts/codex-check.sh"
  "scripts/ralph-config.sh"
  "scripts/ralph-worktree.sh"
  "scripts/xreview-helpers.sh"
  "scripts/secret-scan.sh"
  "scripts/secret-scan-branch.sh"
  "scripts/pre-commit-secret-guard.sh"
  "scripts/commit-msg-guard.sh"
  "scripts/prepare-commit-msg-secret-guard.sh"
  "scripts/pre-merge-commit-secret-guard.sh"
  "scripts/check-template.sh"
  "scripts/check-skill-sync.sh"
)

# extract_required_files <file> — print the entries of a check-template.sh's
# required_files="..." block, one per line, in order, with blank lines
# dropped.
extract_required_files() {
  local file="$1"
  sed -n '/^required_files="$/,/^"$/p' "$file" | sed '1d;$d' | sed '/^[[:space:]]*$/d'
}

# build_fixture <dir> — populate a minimal project tree that passes
# check-template.sh: every golden entry exists, .claude/settings.json is
# "{}", every scripts/*.sh is an executable one-line stub, and the
# directories check-template.sh scans with `find` exist (even if empty),
# so every search root is present for the caller to mutate (e.g. making a
# subtree unreadable) without also having to create the root itself.
build_fixture() {
  local dir="$1" entry
  for entry in "${GOLDEN_ENTRIES[@]}"; do
    mkdir -p "$dir/$(dirname "$entry")"
    case "$entry" in
      .claude/settings.json)
        printf '{}\n' > "$dir/$entry"
        ;;
      scripts/*.sh)
        printf '#!/bin/sh\n' > "$dir/$entry"
        chmod +x "$dir/$entry"
        ;;
      *)
        printf 'fixture placeholder\n' > "$dir/$entry"
        ;;
    esac
  done
  mkdir -p "$dir/.claude/hooks" "$dir/.claude/skills" "$dir/.claude/agents" "$dir/packs"
}

# --- A. golden list comparison ----------------------------------------------
run_case_a() {
  local actual expected

  actual="$(extract_required_files "$CHECK_TEMPLATE")"

  if [ -z "$actual" ]; then
    fail "A. required_files block is non-empty (extraction format of $CHECK_TEMPLATE may have changed)"
    return
  fi
  pass "A. required_files block is non-empty"

  expected="$(printf '%s\n' "${GOLDEN_ENTRIES[@]}")"
  if [ "$actual" = "$expected" ]; then
    pass "A. required_files matches the golden list (${#GOLDEN_ENTRIES[@]} entries, in order)"
  else
    fail "A. required_files differs from the golden list"
    echo "    --- diff (< script: $CHECK_TEMPLATE, > golden) ---"
    diff <(printf '%s\n' "$actual") <(printf '%s\n' "$expected") | sed 's/^/    /'
  fi
}

# --- B. fixture pass + per-entry removal detection --------------------------
run_case_b() {
  local fixture entry saved output rc

  fixture="$(mktemp -d "${TMPDIR:-/tmp}/check-template-fixture.XXXXXX")" || {
    fail "B. mktemp -d failed; skipping the fixture cases"
    return
  }
  # Safety net for an interrupted run or a runner timeout mid-loop; the
  # explicit rm -rf at the end of this function also runs on the normal
  # path, after which this trap is cleared.
  trap 'rm -rf "$fixture"' EXIT

  build_fixture "$fixture"

  output="$(cd "$fixture" && CI=true sh "$CHECK_TEMPLATE" 2>&1)"
  rc=$?
  if [ "$rc" -eq 0 ] && printf '%s\n' "$output" | grep -qF 'Template structure looks good.'; then
    pass "B. clean fixture passes check-template.sh"
  else
    fail "B. clean fixture did not pass check-template.sh (exit $rc)"
    printf '%s\n' "$output" | sed 's/^/    /'
  fi

  for entry in "${GOLDEN_ENTRIES[@]}"; do
    saved="$fixture/${entry}.bak"
    mv "$fixture/$entry" "$saved"
    output="$(cd "$fixture" && CI=true sh "$CHECK_TEMPLATE" 2>&1)"
    rc=$?
    if [ "$rc" -ne 0 ] && printf '%s\n' "$output" | grep -qF "Missing required file: $entry"; then
      pass "B. removing $entry is detected"
    else
      fail "B. removing $entry was NOT detected (exit $rc)"
      printf '%s\n' "$output" | sed 's/^/    /'
    fi
    mv "$saved" "$fixture/$entry"
  done

  rm -rf "$fixture"
  trap - EXIT
}

# --- C. root sanity: this repo satisfies its own golden list ----------------
run_case_c() {
  local entry missing=0
  for entry in "${GOLDEN_ENTRIES[@]}"; do
    if [ ! -e "$REPO_ROOT/$entry" ]; then
      fail "C. $entry missing at repo root"
      missing=$((missing + 1))
    fi
  done
  if [ "$missing" -eq 0 ]; then
    pass "C. all ${#GOLDEN_ENTRIES[@]} golden entries exist at the repo root"
  fi
}

# --- D. root sanity: CI=true run from the repo root has no FAIL line -------
run_case_d() {
  local output rc

  output="$(cd "$REPO_ROOT" && CI=true sh "$CHECK_TEMPLATE" 2>&1)"
  rc=$?
  if [ "$rc" -eq 0 ] \
     && printf '%s\n' "$output" | grep -qF 'Template structure looks good.' \
     && ! printf '%s\n' "$output" | grep -q '^FAIL:'; then
    pass "D. CI=true sh check-template.sh from the repo root exits 0 with no FAIL line"
  else
    fail "D. CI=true sh check-template.sh from the repo root did NOT pass cleanly (exit $rc)"
    printf '%s\n' "$output" | sed 's/^/    /'
  fi
}

# --- E. settings hook-reference check: existing (with arg) vs missing ------
run_case_e() {
  local fixture output rc fail_lines

  fixture="$(mktemp -d "${TMPDIR:-/tmp}/check-template-hooks.XXXXXX")" || {
    fail "E. mktemp -d failed; skipping the hook-reference cases"
    return
  }
  trap 'rm -rf "$fixture"' EXIT

  build_fixture "$fixture"
  printf '#!/bin/sh\necho ok\n' > "$fixture/.claude/hooks/ok.sh"
  chmod +x "$fixture/.claude/hooks/ok.sh"

  cat > "$fixture/.claude/settings.json" <<'JSON'
{
  "hooks": {
    "SessionStart": [
      {"hooks": [{"type": "command", "command": "./.claude/hooks/ok.sh SessionStart"}]}
    ]
  }
}
JSON
  output="$(cd "$fixture" && CI=true sh "$CHECK_TEMPLATE" 2>&1)"
  rc=$?
  if [ "$rc" -eq 0 ] && ! printf '%s\n' "$output" | grep -q '^FAIL:'; then
    pass "E. an existing hook referenced with an argument passes with no FAIL"
  else
    fail "E. an existing hook referenced with an argument did NOT pass (exit $rc)"
    printf '%s\n' "$output" | sed 's/^/    /'
  fi

  cat > "$fixture/.claude/settings.json" <<'JSON'
{
  "hooks": {
    "SessionStart": [
      {"hooks": [{"type": "command", "command": "./.claude/hooks/missing.sh SessionStart"}]}
    ]
  }
}
JSON
  output="$(cd "$fixture" && CI=true sh "$CHECK_TEMPLATE" 2>&1)"
  rc=$?
  if [ "$rc" -ne 0 ] \
     && printf '%s\n' "$output" | grep -qF 'FAIL: Settings file .claude/settings.json references missing hook: ./.claude/hooks/missing.sh' \
     && ! printf '%s\n' "$output" | grep -qF './.claude/hooks/missing.sh SessionStart'; then
    pass "E. a missing hook fails with the path only (no argument) and exit 1"
  else
    fail "E. a missing hook did NOT fail as expected (exit $rc)"
    printf '%s\n' "$output" | sed 's/^/    /'
  fi

  # A dispatcher missing across several events must produce exactly one
  # FAIL line for it (the paths are de-duplicated before being checked).
  cat > "$fixture/.claude/settings.json" <<'JSON'
{
  "hooks": {
    "SessionStart": [
      {"hooks": [{"type": "command", "command": "./.claude/hooks/missing-dispatch.sh SessionStart"}]}
    ],
    "PreToolUse": [
      {"hooks": [{"type": "command", "command": "./.claude/hooks/missing-dispatch.sh PreToolUse"}]}
    ],
    "PostToolUse": [
      {"hooks": [{"type": "command", "command": "./.claude/hooks/missing-dispatch.sh PostToolUse"}]}
    ]
  }
}
JSON
  output="$(cd "$fixture" && CI=true sh "$CHECK_TEMPLATE" 2>&1)"
  rc=$?
  fail_lines="$(printf '%s\n' "$output" | grep -cF 'FAIL: Settings file .claude/settings.json references missing hook: ./.claude/hooks/missing-dispatch.sh')"
  if [ "$rc" -ne 0 ] && [ "$fail_lines" -eq 1 ]; then
    pass "E. a dispatcher missing across several events produces exactly one FAIL line"
  else
    fail "E. a dispatcher missing across several events produced $fail_lines FAIL line(s) instead of exactly one (exit $rc)"
    printf '%s\n' "$output" | sed 's/^/    /'
  fi

  # An unreadable settings.json must be reported instead of silently
  # skipping the hook check. chmod 000 does not block root's own read
  # access, so this only proves anything as a non-root user.
  if [ "$(id -u)" -eq 0 ]; then
    skip "E. unreadable settings.json check skipped (running as root)"
  else
    printf '{}\n' > "$fixture/.claude/settings.json"
    chmod 000 "$fixture/.claude/settings.json"
    output="$(cd "$fixture" && CI=true sh "$CHECK_TEMPLATE" 2>&1)"
    rc=$?
    chmod 644 "$fixture/.claude/settings.json"
    if [ "$rc" -ne 0 ] && printf '%s\n' "$output" | grep -qF 'FAIL: Could not read hook commands from .claude/settings.json'; then
      pass "E. an unreadable settings.json is reported instead of silently skipped"
    else
      fail "E. an unreadable settings.json was NOT reported (exit $rc)"
      printf '%s\n' "$output" | sed 's/^/    /'
    fi
  fi

  rm -rf "$fixture"
  trap - EXIT
}

# --- F. the three find-driven loops propagate failures ---------------------
run_case_f() {
  local fixture output rc

  fixture="$(mktemp -d "${TMPDIR:-/tmp}/check-template-loops.XXXXXX")" || {
    fail "F. mktemp -d failed; skipping the loop-propagation cases"
    return
  }
  trap 'rm -rf "$fixture"' EXIT

  build_fixture "$fixture"

  # F1: a non-executable scripts/*.sh must fail the executable check.
  printf '#!/bin/sh\n' > "$fixture/scripts/extra-nonexec.sh"
  chmod -x "$fixture/scripts/extra-nonexec.sh"
  output="$(cd "$fixture" && CI=true sh "$CHECK_TEMPLATE" 2>&1)"
  rc=$?
  if [ "$rc" -ne 0 ] && printf '%s\n' "$output" | grep -qF 'FAIL: Script is not executable: scripts/extra-nonexec.sh'; then
    pass "F. a non-executable scripts/*.sh is detected and fails"
  else
    fail "F. a non-executable scripts/*.sh was NOT detected (exit $rc)"
    printf '%s\n' "$output" | sed 's/^/    /'
  fi
  rm -f "$fixture/scripts/extra-nonexec.sh"

  # F2: a skill dir without SKILL.md must fail.
  mkdir -p "$fixture/.claude/skills/broken-skill"
  output="$(cd "$fixture" && CI=true sh "$CHECK_TEMPLATE" 2>&1)"
  rc=$?
  if [ "$rc" -ne 0 ] && printf '%s\n' "$output" | grep -qF 'FAIL: Skill missing SKILL.md: .claude/skills/broken-skill'; then
    pass "F. a skill dir without SKILL.md is detected and fails"
  else
    fail "F. a skill dir without SKILL.md was NOT detected (exit $rc)"
    printf '%s\n' "$output" | sed 's/^/    /'
  fi
  rm -rf "$fixture/.claude/skills/broken-skill"

  # F3: an agent file without a 'tools:' field must fail.
  printf 'name: broken\ndescription: broken\n' > "$fixture/.claude/agents/broken-agent.md"
  output="$(cd "$fixture" && CI=true sh "$CHECK_TEMPLATE" 2>&1)"
  rc=$?
  if [ "$rc" -ne 0 ] && printf '%s\n' "$output" | grep -qF "FAIL: Agent missing 'tools' field: .claude/agents/broken-agent.md"; then
    pass "F. an agent file without 'tools:' is detected and fails"
  else
    fail "F. an agent file without 'tools:' was NOT detected (exit $rc)"
    printf '%s\n' "$output" | sed 's/^/    /'
  fi
  rm -f "$fixture/.claude/agents/broken-agent.md"

  # F4: a script path containing a space that IS executable must not fail.
  mkdir -p "$fixture/scripts/with space"
  printf '#!/bin/sh\n' > "$fixture/scripts/with space/exec.sh"
  chmod +x "$fixture/scripts/with space/exec.sh"
  output="$(cd "$fixture" && CI=true sh "$CHECK_TEMPLATE" 2>&1)"
  rc=$?
  if [ "$rc" -eq 0 ] && ! printf '%s\n' "$output" | grep -q '^FAIL:'; then
    pass "F. an executable script path containing a space is not flagged"
  else
    fail "F. an executable script path containing a space was incorrectly flagged (exit $rc)"
    printf '%s\n' "$output" | sed 's/^/    /'
  fi
  rm -rf "$fixture/scripts/with space"

  # F5: an unreadable subtree under scripts/ must be reported (via find's
  # own non-zero exit) instead of being silently skipped. chmod 000 does
  # not block root's own read access, so this only proves anything as a
  # non-root user.
  if [ "$(id -u)" -eq 0 ]; then
    skip "F. unreadable subtree check skipped (running as root)"
  else
    mkdir -p "$fixture/scripts/locked"
    printf '#!/bin/sh\n' > "$fixture/scripts/locked/inner.sh"
    chmod 000 "$fixture/scripts/locked"
    output="$(cd "$fixture" && CI=true sh "$CHECK_TEMPLATE" 2>&1)"
    rc=$?
    chmod 755 "$fixture/scripts/locked"
    rm -rf "$fixture/scripts/locked"
    if [ "$rc" -ne 0 ] && printf '%s\n' "$output" | grep -qF 'FAIL: Could not list scripts'; then
      pass "F. an unreadable subtree under scripts/ is reported instead of silently skipped"
    else
      fail "F. an unreadable subtree under scripts/ was NOT reported (exit $rc)"
      printf '%s\n' "$output" | sed 's/^/    /'
    fi
  fi

  # F6: a non-executable .sh directly under .claude/hooks/ (outside
  # local/) must still fail the executable check -- pruning local/ must
  # not swallow the rest of .claude/hooks/ along with it.
  printf '#!/bin/sh\n' > "$fixture/.claude/hooks/core-nonexec.sh"
  chmod -x "$fixture/.claude/hooks/core-nonexec.sh"
  output="$(cd "$fixture" && CI=true sh "$CHECK_TEMPLATE" 2>&1)"
  rc=$?
  rm -f "$fixture/.claude/hooks/core-nonexec.sh"
  if [ "$rc" -ne 0 ] && printf '%s\n' "$output" | grep -qF 'FAIL: Script is not executable: .claude/hooks/core-nonexec.sh'; then
    pass "F. a non-executable .sh under .claude/hooks/ (outside local/) is detected and fails"
  else
    fail "F. a non-executable .sh under .claude/hooks/ (outside local/) was NOT detected (exit $rc)"
    printf '%s\n' "$output" | sed 's/^/    /'
  fi

  # F7: an unreadable directory under .claude/hooks/local/ must not cause
  # a FAIL -- the whole local/ tree is pruned before find ever descends
  # into it, so a permission error inside it never surfaces. chmod 000
  # does not block root's own read access, so this only proves anything
  # as a non-root user.
  if [ "$(id -u)" -eq 0 ]; then
    skip "F. unreadable .claude/hooks/local/ subtree check skipped (running as root)"
  else
    printf '#!/bin/sh\necho ok\n' > "$fixture/.claude/hooks/normal-hook.sh"
    chmod +x "$fixture/.claude/hooks/normal-hook.sh"
    mkdir -p "$fixture/.claude/hooks/local/disabled"
    chmod 000 "$fixture/.claude/hooks/local/disabled"
    output="$(cd "$fixture" && CI=true sh "$CHECK_TEMPLATE" 2>&1)"
    rc=$?
    chmod 755 "$fixture/.claude/hooks/local/disabled"
    rm -rf "$fixture/.claude/hooks/local" "$fixture/.claude/hooks/normal-hook.sh"
    if [ "$rc" -eq 0 ] && ! printf '%s\n' "$output" | grep -q '^FAIL:'; then
      pass "F. an unreadable directory under .claude/hooks/local/ does not cause a FAIL"
    else
      fail "F. an unreadable directory under .claude/hooks/local/ incorrectly caused a FAIL (exit $rc)"
      printf '%s\n' "$output" | sed 's/^/    /'
    fi
  fi

  # F8: a non-executable .sh under .claude/hooks/local/ must not cause a
  # FAIL -- the local/ tree stays excluded from the executable check, as
  # before (now via pruning instead of post-hoc filtering).
  mkdir -p "$fixture/.claude/hooks/local"
  printf '#!/bin/sh\n' > "$fixture/.claude/hooks/local/disabled.sh"
  chmod -x "$fixture/.claude/hooks/local/disabled.sh"
  output="$(cd "$fixture" && CI=true sh "$CHECK_TEMPLATE" 2>&1)"
  rc=$?
  rm -rf "$fixture/.claude/hooks/local"
  if [ "$rc" -eq 0 ] && ! printf '%s\n' "$output" | grep -q '^FAIL:'; then
    pass "F. a non-executable .sh under .claude/hooks/local/ does not cause a FAIL"
  else
    fail "F. a non-executable .sh under .claude/hooks/local/ incorrectly caused a FAIL (exit $rc)"
    printf '%s\n' "$output" | sed 's/^/    /'
  fi

  # F8 extension: .claude/hooks/local itself at mode 644 (no execute bit
  # on the directory), containing a non-executable .sh, must still not
  # cause a FAIL. Without an explicit -print on the find expression's
  # right-hand branch, find's "no action other than -prune" default-print
  # rule would print the pruned local/ directory entry itself (not its
  # contents) alongside the matched scripts; that directory entry would
  # then fail "[ -x ]" because it has no execute bit, producing a false
  # FAIL. The explicit -print scopes printing to the right-hand branch
  # only, so the pruned entry itself is never printed. chmod 644 removing
  # every execute bit does not block root's own traversal the same way,
  # so this only proves anything as a non-root user.
  if [ "$(id -u)" -eq 0 ]; then
    skip "F. .claude/hooks/local at mode 644 check skipped (running as root)"
  else
    mkdir -p "$fixture/.claude/hooks/local"
    printf '#!/bin/sh\n' > "$fixture/.claude/hooks/local/disabled.sh"
    chmod -x "$fixture/.claude/hooks/local/disabled.sh"
    chmod 644 "$fixture/.claude/hooks/local"
    output="$(cd "$fixture" && CI=true sh "$CHECK_TEMPLATE" 2>&1)"
    rc=$?
    chmod 755 "$fixture/.claude/hooks/local"
    rm -rf "$fixture/.claude/hooks/local"
    if [ "$rc" -eq 0 ] && ! printf '%s\n' "$output" | grep -q '^FAIL:'; then
      pass "F. .claude/hooks/local at mode 644 with a non-executable .sh inside does not cause a FAIL"
    else
      fail "F. .claude/hooks/local at mode 644 with a non-executable .sh inside incorrectly caused a FAIL (exit $rc)"
      printf '%s\n' "$output" | sed 's/^/    /'
    fi
  fi

  # F9: an unreadable subtree under .claude/agents/ must be reported (via
  # find's own non-zero exit) instead of being silently skipped, the same
  # way as F5's scripts/ case. chmod 000 does not block root's own read
  # access, so this only proves anything as a non-root user.
  if [ "$(id -u)" -eq 0 ]; then
    skip "F. unreadable .claude/agents/ subtree check skipped (running as root)"
  else
    mkdir -p "$fixture/.claude/agents/locked"
    chmod 000 "$fixture/.claude/agents/locked"
    output="$(cd "$fixture" && CI=true sh "$CHECK_TEMPLATE" 2>&1)"
    rc=$?
    chmod 755 "$fixture/.claude/agents/locked"
    rm -rf "$fixture/.claude/agents/locked"
    if [ "$rc" -ne 0 ] && printf '%s\n' "$output" | grep -qF 'FAIL: Could not list agent files'; then
      pass "F. an unreadable subtree under .claude/agents/ is reported instead of silently skipped"
    else
      fail "F. an unreadable subtree under .claude/agents/ was NOT reported (exit $rc)"
      printf '%s\n' "$output" | sed 's/^/    /'
    fi
  fi

  # F10: .claude/skills itself unreadable. The skills find then fails
  # (unlike an unreadable child directory, which the SKILL.md loop reports
  # on its own), and that failure must be reported. Without the find
  # exit-status check the script prints nothing and exits 0, leaving every
  # skill unchecked. chmod 000 does not block root's own read access, so
  # this only proves anything as a non-root user.
  if [ "$(id -u)" -eq 0 ]; then
    skip "F. unreadable .claude/skills/ check skipped (running as root)"
  else
    chmod 000 "$fixture/.claude/skills"
    output="$(cd "$fixture" && CI=true sh "$CHECK_TEMPLATE" 2>&1)"
    rc=$?
    chmod 755 "$fixture/.claude/skills"
    if [ "$rc" -eq 1 ] && printf '%s\n' "$output" | grep -qF 'Could not list skill directories under .claude/skills'; then
      pass "F. an unreadable .claude/skills/ is reported instead of silently skipped"
    else
      fail "F. an unreadable .claude/skills/ was NOT reported (exit $rc)"
      printf '%s\n' "$output" | sed 's/^/    /'
    fi
  fi

  rm -rf "$fixture"
  trap - EXIT
}

# --- G. fresh scaffold: go run ./cmd/ralph init --yes <tmp> -----------------
run_case_g() {
  local fresh init_output output rc

  if ! command -v go >/dev/null 2>&1; then
    skip "G. fresh scaffold check skipped (go is unavailable)"
    return
  fi

  fresh="$(mktemp -d "${TMPDIR:-/tmp}/check-template-fresh.XXXXXX")" || {
    fail "G. mktemp -d failed; skipping the fresh scaffold case"
    return
  }
  trap 'rm -rf "$fresh"' EXIT

  if ! init_output="$(cd "$REPO_ROOT" && go run ./cmd/ralph init --yes "$fresh" 2>&1)"; then
    fail "G. go run ./cmd/ralph init --yes $fresh failed"
    printf '%s\n' "$init_output" | sed 's/^/    /'
    rm -rf "$fresh"
    trap - EXIT
    return
  fi

  output="$(cd "$fresh" && CI=true sh scripts/check-template.sh 2>&1)"
  rc=$?
  if [ "$rc" -eq 0 ] && ! printf '%s\n' "$output" | grep -q '^FAIL:'; then
    pass "G. a fresh ralph init scaffold passes check-template.sh with no FAIL line"
  else
    fail "G. a fresh ralph init scaffold did NOT pass check-template.sh (exit $rc)"
    printf '%s\n' "$output" | sed 's/^/    /'
  fi

  rm -rf "$fresh"
  trap - EXIT
}

# --- H. missing search roots are skipped, not reported -----------------
run_case_h() {
  local fixture output rc

  fixture="$(mktemp -d "${TMPDIR:-/tmp}/check-template-missing-roots.XXXXXX")" || {
    fail "H. mktemp -d failed; skipping the missing-root cases"
    return
  }
  trap 'rm -rf "$fixture"' EXIT

  # H1: a project without packs/ and without .claude/agents/ must still
  # pass. The scripts find only receives roots that exist ([ -d ]), so a
  # missing packs/ is simply left out of the search instead of making
  # find error on a nonexistent path; the agents loop is skipped entirely
  # (not run against a missing .claude/agents) the same way.
  build_fixture "$fixture"
  rm -rf "$fixture/packs" "$fixture/.claude/agents"
  output="$(cd "$fixture" && CI=true sh "$CHECK_TEMPLATE" 2>&1)"
  rc=$?
  if [ "$rc" -eq 0 ] && ! printf '%s\n' "$output" | grep -q '^FAIL:'; then
    pass "H. a project without packs/ and without .claude/agents/ passes with no FAIL"
  else
    fail "H. a project without packs/ and without .claude/agents/ did NOT pass (exit $rc)"
    printf '%s\n' "$output" | sed 's/^/    /'
  fi

  # Rebuild a clean fixture for H2 so it isolates only the missing
  # .claude/skills/ condition.
  rm -rf "$fixture"
  mkdir -p "$fixture"
  build_fixture "$fixture"
  rm -rf "$fixture/.claude/skills"
  output="$(cd "$fixture" && CI=true sh "$CHECK_TEMPLATE" 2>&1)"
  rc=$?
  if [ "$rc" -eq 0 ] && ! printf '%s\n' "$output" | grep -q '^FAIL:'; then
    pass "H. a project without .claude/skills/ passes with no FAIL"
  else
    fail "H. a project without .claude/skills/ did NOT pass (exit $rc)"
    printf '%s\n' "$output" | sed 's/^/    /'
  fi

  rm -rf "$fixture"
  trap - EXIT
}

# --- I. signal cleanup under dash ------------------------------------------
# run_signal_case <signal> <expected-exit-status> <work-dir> — start
# check-template.sh under dash in the background from <work-dir>/fixture
# with a private TMPDIR and the stub `find` (<work-dir>/bin) first on PATH,
# wait until the stub reports it is running (so the traps are installed
# and the script is parked inside find), send <signal> to dash, then check
# the exit status and that the private TMPDIR holds no check-template.*
# directory. dash runs a trapped signal's handler only after the foreground
# stub returns, so the handler's `exit` (and the EXIT trap's cleanup) runs
# once the stub's sleep ends, bounded at about a second per signal. Without
# the INT/TERM/HUP traps the signal kills dash outright, skipping the EXIT
# trap and leaving the temp directory behind.
run_signal_case() {
  local sig="$1" want_rc="$2" work="$3"
  local private_tmp="$work/tmp-$sig" ready="$work/ready-$sig" log="$work/out-$sig.log"
  local pid rc polls leftover entry

  mkdir -p "$private_tmp"
  # Job control gives the background job its own process group and keeps it
  # from starting with SIGINT ignored (a plain `&` job ignores SIGINT and
  # SIGQUIT), so INT reaches dash's trap instead of being dropped.
  set -m
  (cd "$work/fixture" && PATH="$work/bin:$PATH" TMPDIR="$private_tmp" STUB_READY="$ready" CI=true exec dash "$CHECK_TEMPLATE") >"$log" 2>&1 </dev/null &
  pid=$!
  set +m

  polls=0
  while [ ! -e "$ready" ] && [ "$polls" -lt 100 ]; do
    sleep 0.05
    polls=$((polls + 1))
  done
  if [ -e "$ready" ]; then
    kill -s "$sig" "$pid"
  else
    kill -s KILL "$pid" 2>/dev/null
  fi
  wait "$pid" 2>/dev/null
  rc=$?
  # Reap what the signal can leave running: with the traps missing, dash
  # dies at once and the stub's sleep outlives it as an orphan in the same
  # process group.
  kill -s KILL -- "-$pid" 2>/dev/null

  if [ ! -e "$ready" ]; then
    fail "I. dash: $sig case never started (the stub find did not run within 5s)"
    sed 's/^/    /' "$log"
    return
  fi
  if grep -qF 'Template structure looks good.' "$log"; then
    skip "I. dash: $sig cleanup check skipped (the script ran to completion; $sig is ignored in this environment)"
    return
  fi

  leftover=0
  for entry in "$private_tmp"/check-template.*; do
    [ -e "$entry" ] && leftover=$((leftover + 1))
  done
  if [ "$rc" -eq "$want_rc" ]; then
    pass "I. dash: $sig makes check-template.sh exit $want_rc"
  else
    fail "I. dash: $sig made check-template.sh exit $rc (want $want_rc)"
    sed 's/^/    /' "$log"
  fi
  if [ "$leftover" -eq 0 ]; then
    pass "I. dash: $sig leaves no check-template.* temp directory"
  else
    fail "I. dash: $sig left $leftover check-template.* temp director(ies) behind"
  fi
}

run_case_i() {
  local work real_find

  if ! command -v dash >/dev/null 2>&1; then
    skip "I. signal cleanup check skipped (dash is unavailable)"
    return
  fi

  work="$(mktemp -d "${TMPDIR:-/tmp}/check-template-signals.XXXXXX")" || {
    fail "I. mktemp -d failed; skipping the signal cleanup cases"
    return
  }
  trap 'rm -rf "$work"' EXIT

  build_fixture "$work/fixture"
  # The stub marks that it is running, sleeps long enough to comfortably
  # outlast the delay before the signal is sent, then runs the real find.
  mkdir -p "$work/bin"
  real_find="$(command -v find)"
  printf '#!/bin/sh\n: > "$STUB_READY"\nsleep 1\nexec "%s" "$@"\n' "$real_find" > "$work/bin/find"
  chmod +x "$work/bin/find"

  run_signal_case TERM 143 "$work"
  run_signal_case INT 130 "$work"
  run_signal_case HUP 129 "$work"

  rm -rf "$work"
  trap - EXIT
}

run_case_a
run_case_b
run_case_c
run_case_d
run_case_e
run_case_f
run_case_g
run_case_h
run_case_i

printf '\ntest-check-template: %s passed, %s failed, %s skipped\n' "$pass_count" "$fail_count" "$skip_count"
[ "$fail_count" -eq 0 ]
