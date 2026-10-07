#!/usr/bin/env sh
# tests/test-archive-plan.sh — scripts/archive-plan.sh rewrites the plan's
# docs/plans/active/ links in docs/tech-debt/README.md before it moves the
# plan, and scripts/verify.local.sh fails when a plan link in that README
# does not exist at its path.
#
# archive-plan.sh (a copy runs inside each fixture; it uses paths relative to
# the current directory):
#   1. file plan: two links rewritten (one inside backticks, one ending a
#      sentence with "."), a similar name and a regex look-alike untouched,
#      "Updated 2", README mode kept, no temp file left
#   2. directory plan: <slug>/plan.md and <slug> at the end of a line
#      rewritten, <slug>-bar.md (same line, before the real link) and
#      <slug>.md untouched
#   3. no README (docs/tech-debt/ with only .gitkeep, and no docs/tech-debt/
#      at all): plan moved, exit 0, nothing created, no "Updated" line
#   4. the move fails (docs/plans/archive not writable): non-zero exit, README
#      already rewritten, plan still active; after restoring the permission a
#      re-run moves the plan and rewrites 0 (README unchanged). SKIP as root.
#   5. destination already exists: exit 1, README untouched
# verify.local.sh (HARNESS_VERIFY_MODE=static in a minimal fixture: the
# checks that need other repo files are skipped, and the Codex provenance
# check gets the two real AGENTS.override.md files):
#   6. every link exists (active file, archived file with a sentence-ending
#      ".", archived directory plan) -> exit 0, check OK
#   7. an archived plan still linked as active, and an archive link to a plan
#      that does not exist -> non-zero exit, check FAIL, both listed
#   8. no README -> exit 0, check OK
#   9. archive-plan.sh on a linked plan keeps verify.local.sh green
set -eu

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
ARCHIVE_PLAN="${PROJECT_ROOT}/scripts/archive-plan.sh"
VERIFY_LOCAL="${PROJECT_ROOT}/scripts/verify.local.sh"

TMP_ROOT="$(mktemp -d)"
# chmod first: case 4 makes a fixture directory read-only.
trap 'chmod -R u+w "$TMP_ROOT" 2>/dev/null; rm -r -f "$TMP_ROOT"' EXIT
trap 'exit 1' HUP INT TERM

_pass=0
_fail=0
_total=0

pass() {
  _total=$((_total + 1))
  _pass=$((_pass + 1))
  printf '  PASS: %s\n' "$1"
}

fail() {
  _total=$((_total + 1))
  _fail=$((_fail + 1))
  printf '  FAIL: %s\n' "$1"
}

# make_archive_fixture NAME — a fixture with scripts/archive-plan.sh and
# empty docs/plans/active and docs/tech-debt. Prints the fixture directory.
make_archive_fixture() {
  _dir="${TMP_ROOT}/$1"
  mkdir -p "${_dir}/scripts" "${_dir}/docs/plans/active" "${_dir}/docs/tech-debt"
  cp "$ARCHIVE_PLAN" "${_dir}/scripts/archive-plan.sh"
  printf '%s\n' "$_dir"
}

# run_archive DIR ARG LABEL — run DIR's archive-plan.sh from DIR with ARG.
# Sets _rc to its exit code and _out to the file holding its output.
run_archive() {
  _out="${TMP_ROOT}/$3.out"
  _rc=0
  (cd "$1" && sh scripts/archive-plan.sh "$2") > "$_out" 2>&1 || _rc=$?
}

# output_has TEXT — the last run's output contains TEXT.
output_has() {
  grep -F -- "$1" "$_out" >/dev/null 2>&1
}

show_output() {
  sed 's/^/      | /' "$_out"
}

# same_file DESC ACTUAL EXPECTED — ACTUAL equals EXPECTED byte for byte.
same_file() {
  if cmp -s "$2" "$3"; then
    pass "$1"
  else
    fail "$1"
    diff "$3" "$2" | sed 's/^/      | /' || true
  fi
}

# no_temp_left DESC DIR — archive-plan.sh left no .README.md.* in DIR.
no_temp_left() {
  _left="$(find "$2" -name '.README.md.*' 2>/dev/null)"
  if [ -z "$_left" ]; then
    pass "$1"
  else
    fail "$1: ${_left}"
  fi
}

# ─── archive-plan.sh ────────────────────────────────────────────────────

printf '==> archive-plan.sh rewrites tech-debt README links before moving the plan\n'

# Case 1: file plan.
_fx="$(make_archive_fixture file-plan)"
printf 'plan\n' > "${_fx}/docs/plans/active/2026-10-07-foo.md"
printf 'other plan\n' > "${_fx}/docs/plans/active/2026-10-07-foo-bar.md"
cat > "${_fx}/docs/tech-debt/README.md" <<'EOF'
# Tech debt

| Item | Plan |
|------|------|
| row one | `docs/plans/active/2026-10-07-foo.md` and docs/plans/active/2026-10-07-foo-bar.md |
| look-alike | docs/plans/active/2026-10-07-fooXmd is a different name |

See docs/plans/active/2026-10-07-foo.md.
EOF
cat > "${TMP_ROOT}/file-plan.expected" <<'EOF'
# Tech debt

| Item | Plan |
|------|------|
| row one | `docs/plans/archive/2026-10-07-foo.md` and docs/plans/active/2026-10-07-foo-bar.md |
| look-alike | docs/plans/active/2026-10-07-fooXmd is a different name |

See docs/plans/archive/2026-10-07-foo.md.
EOF
chmod 644 "${_fx}/docs/tech-debt/README.md"
run_archive "$_fx" docs/plans/active/2026-10-07-foo.md file-plan
if [ "$_rc" -eq 0 ] && [ -f "${_fx}/docs/plans/archive/2026-10-07-foo.md" ] \
  && [ ! -e "${_fx}/docs/plans/active/2026-10-07-foo.md" ]; then
  pass "file plan: exit 0 and the plan is in docs/plans/archive"
else
  fail "file plan: exit ${_rc}, plan not moved"
  show_output
fi
same_file "file plan: both links rewritten, foo-bar.md and fooXmd untouched" \
  "${_fx}/docs/tech-debt/README.md" "${TMP_ROOT}/file-plan.expected"
if output_has "Updated 2 reference(s) in docs/tech-debt/README.md" \
  && output_has "Archived: docs/plans/active/2026-10-07-foo.md -> docs/plans/archive/2026-10-07-foo.md"; then
  pass "file plan: prints Updated 2 and the Archived line"
else
  fail "file plan: output lacks the Updated 2 or Archived line"
  show_output
fi
_perm="$(ls -l "${_fx}/docs/tech-debt/README.md" | cut -c1-10)"
if [ "$_perm" = "-rw-r--r--" ]; then
  pass "file plan: README keeps mode 644"
else
  fail "file plan: README mode is ${_perm}, expected -rw-r--r--"
fi
no_temp_left "file plan: no temp file left in docs/tech-debt" "${_fx}/docs/tech-debt"

# Case 2: directory plan whose name is a prefix of another plan's name.
_fx="$(make_archive_fixture dir-plan)"
mkdir -p "${_fx}/docs/plans/active/2026-10-07-foo"
printf 'plan\n' > "${_fx}/docs/plans/active/2026-10-07-foo/plan.md"
cat > "${_fx}/docs/tech-debt/README.md" <<'EOF'
- docs/plans/active/2026-10-07-foo-bar.md, then docs/plans/active/2026-10-07-foo/plan.md
- docs/plans/active/2026-10-07-foo.md is a file plan with a longer name
- the directory itself: docs/plans/active/2026-10-07-foo
EOF
cat > "${TMP_ROOT}/dir-plan.expected" <<'EOF'
- docs/plans/active/2026-10-07-foo-bar.md, then docs/plans/archive/2026-10-07-foo/plan.md
- docs/plans/active/2026-10-07-foo.md is a file plan with a longer name
- the directory itself: docs/plans/archive/2026-10-07-foo
EOF
run_archive "$_fx" 2026-10-07-foo dir-plan
if [ "$_rc" -eq 0 ] && [ -f "${_fx}/docs/plans/archive/2026-10-07-foo/plan.md" ] \
  && output_has "Updated 2 reference(s) in docs/tech-debt/README.md"; then
  pass "directory plan: exit 0, moved, Updated 2"
else
  fail "directory plan: exit ${_rc}, or not moved, or no Updated 2"
  show_output
fi
same_file "directory plan: <slug>/plan.md and <slug> rewritten, <slug>-bar.md and <slug>.md untouched" \
  "${_fx}/docs/tech-debt/README.md" "${TMP_ROOT}/dir-plan.expected"

# Case 3: no README (downstream ships docs/tech-debt/ with only .gitkeep).
_fx="$(make_archive_fixture no-readme)"
: > "${_fx}/docs/tech-debt/.gitkeep"
printf 'plan\n' > "${_fx}/docs/plans/active/2026-10-07-foo.md"
run_archive "$_fx" 2026-10-07-foo.md no-readme
_listing="$(ls -A "${_fx}/docs/tech-debt")"
if [ "$_rc" -eq 0 ] && [ -f "${_fx}/docs/plans/archive/2026-10-07-foo.md" ] \
  && [ "$_listing" = ".gitkeep" ] && ! output_has "Updated"; then
  pass "no README: exit 0, plan moved, docs/tech-debt still holds only .gitkeep"
else
  fail "no README: exit ${_rc}, docs/tech-debt holds [${_listing}]"
  show_output
fi

_fx="$(make_archive_fixture no-tech-debt-dir)"
rmdir "${_fx}/docs/tech-debt"
printf 'plan\n' > "${_fx}/docs/plans/active/2026-10-07-foo.md"
run_archive "$_fx" 2026-10-07-foo.md no-tech-debt-dir
if [ "$_rc" -eq 0 ] && [ -f "${_fx}/docs/plans/archive/2026-10-07-foo.md" ] \
  && [ ! -e "${_fx}/docs/tech-debt" ]; then
  pass "no docs/tech-debt: exit 0, plan moved, docs/tech-debt not created"
else
  fail "no docs/tech-debt: exit ${_rc}, or not moved, or docs/tech-debt created"
  show_output
fi

# Case 4: the move fails after the rewrite; a re-run finishes the job.
if [ "$(id -u)" = "0" ]; then
  printf '  SKIP: retry after a failed move (running as root; a read-only directory does not stop root)\n'
else
  _fx="$(make_archive_fixture retry)"
  mkdir -p "${_fx}/docs/plans/archive"
  printf 'plan\n' > "${_fx}/docs/plans/active/2026-10-07-foo.md"
  printf 'See docs/plans/active/2026-10-07-foo.md for details\n' > "${_fx}/docs/tech-debt/README.md"
  printf 'See docs/plans/archive/2026-10-07-foo.md for details\n' > "${TMP_ROOT}/retry.expected"
  chmod 555 "${_fx}/docs/plans/archive"
  run_archive "$_fx" 2026-10-07-foo.md retry-first
  chmod 755 "${_fx}/docs/plans/archive"
  if [ "$_rc" -ne 0 ] && [ -f "${_fx}/docs/plans/active/2026-10-07-foo.md" ] \
    && [ ! -e "${_fx}/docs/plans/archive/2026-10-07-foo.md" ] \
    && output_has "Updated 1 reference(s) in docs/tech-debt/README.md" \
    && output_has "Failed to move"; then
    pass "failed move: exit ${_rc}, plan still in docs/plans/active, Updated 1 printed"
  else
    fail "failed move: exit ${_rc}, or plan moved, or messages missing"
    show_output
  fi
  same_file "failed move: README already rewritten" \
    "${_fx}/docs/tech-debt/README.md" "${TMP_ROOT}/retry.expected"
  no_temp_left "failed move: no temp file left in docs/tech-debt" "${_fx}/docs/tech-debt"

  run_archive "$_fx" 2026-10-07-foo.md retry-second
  if [ "$_rc" -eq 0 ] && [ -f "${_fx}/docs/plans/archive/2026-10-07-foo.md" ] \
    && [ ! -e "${_fx}/docs/plans/active/2026-10-07-foo.md" ] \
    && ! output_has "Updated" && output_has "Archived:"; then
    pass "re-run: exit 0, plan moved, 0 references updated"
  else
    fail "re-run: exit ${_rc}, or plan not moved, or an Updated line"
    show_output
  fi
  same_file "re-run: README unchanged by the second run" \
    "${_fx}/docs/tech-debt/README.md" "${TMP_ROOT}/retry.expected"
  no_temp_left "re-run: no temp file left in docs/tech-debt" "${_fx}/docs/tech-debt"
fi

# Case 5: the destination exists, so nothing is rewritten or moved.
_fx="$(make_archive_fixture dest-exists)"
mkdir -p "${_fx}/docs/plans/archive"
printf 'plan\n' > "${_fx}/docs/plans/active/2026-10-07-foo.md"
printf 'old copy\n' > "${_fx}/docs/plans/archive/2026-10-07-foo.md"
printf 'See docs/plans/active/2026-10-07-foo.md\n' > "${_fx}/docs/tech-debt/README.md"
cp "${_fx}/docs/tech-debt/README.md" "${TMP_ROOT}/dest-exists.expected"
run_archive "$_fx" 2026-10-07-foo.md dest-exists
if [ "$_rc" -eq 1 ] && output_has "Archive already contains 2026-10-07-foo.md. Aborting to avoid overwrite." \
  && [ -f "${_fx}/docs/plans/active/2026-10-07-foo.md" ] && ! output_has "Updated"; then
  pass "destination exists: exit 1, plan left in docs/plans/active"
else
  fail "destination exists: exit ${_rc}, or wrong message, or plan moved"
  show_output
fi
same_file "destination exists: README untouched" \
  "${_fx}/docs/tech-debt/README.md" "${TMP_ROOT}/dest-exists.expected"

# ─── verify.local.sh ────────────────────────────────────────────────────

printf '\n==> verify.local.sh checks the plan links in docs/tech-debt/README.md\n'

# make_verify_fixture NAME — a fixture that verify.local.sh's static mode
# passes in: its own copy, the two AGENTS.override.md files the Codex
# provenance check reads, and empty plan and tech-debt directories.
make_verify_fixture() {
  _dir="${TMP_ROOT}/$1"
  mkdir -p "${_dir}/scripts" "${_dir}/.codex" "${_dir}/templates/base/.codex" \
    "${_dir}/docs/plans/active" "${_dir}/docs/plans/archive" "${_dir}/docs/tech-debt"
  cp "$VERIFY_LOCAL" "${_dir}/scripts/verify.local.sh"
  cp "${PROJECT_ROOT}/.codex/AGENTS.override.md" "${_dir}/.codex/AGENTS.override.md"
  cp "${PROJECT_ROOT}/templates/base/.codex/AGENTS.override.md" \
    "${_dir}/templates/base/.codex/AGENTS.override.md"
  printf '%s\n' "$_dir"
}

# run_verify DIR LABEL — run DIR's verify.local.sh in static mode.
run_verify() {
  _out="${TMP_ROOT}/$2.out"
  _rc=0
  HARNESS_VERIFY_MODE=static sh "$1/scripts/verify.local.sh" > "$_out" 2>&1 || _rc=$?
}

# ref_check_result — the OK/FAIL line of the tech-debt check in the last run.
ref_check_result() {
  awk '
    $0 == "==> tech-debt README plan references" { inblock = 1; next }
    inblock && /^    (OK|FAIL)$/ { sub(/^    /, ""); print; exit }
    inblock && /^==> / { exit }
  ' "$_out"
}

# expect_verify_ok DESC — the last run exited 0 and the check printed OK.
expect_verify_ok() {
  _result="$(ref_check_result)"
  if [ "$_rc" -eq 0 ] && [ "$_result" = "OK" ]; then
    pass "$1 (exit 0, check OK)"
  else
    fail "$1: exit ${_rc}, check result '${_result}'"
    show_output
  fi
}

# Case 6: every link exists.
_fx="$(make_verify_fixture verify-ok)"
printf 'plan\n' > "${_fx}/docs/plans/active/2026-10-07-live.md"
printf 'plan\n' > "${_fx}/docs/plans/archive/2026-10-01-done.md"
mkdir -p "${_fx}/docs/plans/archive/2026-10-02-dir"
printf 'plan\n' > "${_fx}/docs/plans/archive/2026-10-02-dir/plan.md"
cat > "${_fx}/docs/tech-debt/README.md" <<'EOF'
- active file plan: docs/plans/active/2026-10-07-live.md
- archived file plan, end of a sentence: docs/plans/archive/2026-10-01-done.md.
- archived directory plan: docs/plans/archive/2026-10-02-dir/plan.md
- a placeholder is not a link: docs/plans/active/<date>-<slug>.md
EOF
run_verify "$_fx" verify-ok
expect_verify_ok "all links exist"

# Case 7: a plan archived by hand, and a link to a plan that never existed.
_fx="$(make_verify_fixture verify-missing)"
printf 'plan\n' > "${_fx}/docs/plans/archive/2026-10-01-done.md"
cat > "${_fx}/docs/tech-debt/README.md" <<'EOF'
- moved by hand: docs/plans/active/2026-10-01-done.md
- never existed: docs/plans/archive/2026-09-30-gone.md
- fine: docs/plans/archive/2026-10-01-done.md
EOF
run_verify "$_fx" verify-missing
_result="$(ref_check_result)"
if [ "$_rc" -ne 0 ] && [ "$_result" = "FAIL" ] \
  && output_has "  - docs/plans/active/2026-10-01-done.md" \
  && output_has "  - docs/plans/archive/2026-09-30-gone.md" \
  && ! output_has "  - docs/plans/archive/2026-10-01-done.md"; then
  pass "missing links: exit ${_rc}, check FAIL, both missing links listed, the existing one not"
else
  fail "missing links: exit ${_rc}, check result '${_result}', or wrong list"
  show_output
fi

# Case 8: no README.
_fx="$(make_verify_fixture verify-no-readme)"
run_verify "$_fx" verify-no-readme
expect_verify_ok "no README"

# Case 9: archive-plan.sh keeps the links valid.
_fx="$(make_verify_fixture verify-after-archive)"
cp "$ARCHIVE_PLAN" "${_fx}/scripts/archive-plan.sh"
printf 'plan\n' > "${_fx}/docs/plans/active/2026-10-07-live.md"
printf 'See docs/plans/active/2026-10-07-live.md.\n' > "${_fx}/docs/tech-debt/README.md"
run_verify "$_fx" verify-before-archive
expect_verify_ok "before archiving: the active link exists"
run_archive "$_fx" 2026-10-07-live.md verify-archive
if [ "$_rc" -ne 0 ]; then
  fail "archive-plan.sh in the verify fixture: exit ${_rc}"
  show_output
fi
run_verify "$_fx" verify-after-archive
expect_verify_ok "after archive-plan.sh: the rewritten link exists"

printf '\narchive-plan tests: %s passed, %s failed, %s total\n' "$_pass" "$_fail" "$_total"
[ "$_fail" -eq 0 ]
