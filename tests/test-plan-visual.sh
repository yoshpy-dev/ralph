#!/usr/bin/env sh
set -eu

# test-plan-visual.sh — scripts/plan-visual.sh open / shot / digest.
# The opener and the browser are always stubs: no real browser starts.

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
PLAN_VISUAL="${PROJECT_ROOT}/scripts/plan-visual.sh"
SH="$(command -v sh)"

# The caller's own settings must not steer opener / browser resolution.
unset RALPH_PLAN_VISUAL_OPENER RALPH_PLAN_VISUAL_BROWSER

_pass=0
_fail=0
_total=0
_tmp=""

cleanup() {
  if [ -n "$_tmp" ]; then
    rm -rf "$_tmp"
  fi
}
trap cleanup EXIT
trap 'exit 1' HUP INT TERM

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

assert_eq() {
  _desc="$1"
  _expected="$2"
  _actual="$3"
  if [ "$_expected" = "$_actual" ]; then
    pass "$_desc"
  else
    fail "$_desc"
    printf '    expected: %s\n    actual:   %s\n' "$_expected" "$_actual"
  fi
}

assert_ne() {
  _desc="$1"
  _unexpected="$2"
  _actual="$3"
  if [ "$_unexpected" != "$_actual" ]; then
    pass "$_desc"
  else
    fail "$_desc"
    printf '    both values: %s\n' "$_actual"
  fi
}

# assert_has_line <desc> <exact line> <file>
assert_has_line() {
  if [ -f "$3" ] && grep -Fxq -- "$2" "$3"; then
    pass "$1"
  else
    fail "$1"
    printf '    missing line: %s\n' "$2"
    if [ -f "$3" ]; then
      sed 's/^/    | /' "$3"
    else
      printf '    (no file: %s)\n' "$3"
    fi
  fi
}

# assert_contains <desc> <substring> <file>
assert_contains() {
  if [ -f "$3" ] && grep -Fq -- "$2" "$3"; then
    pass "$1"
  else
    fail "$1"
    printf '    missing text: %s\n' "$2"
    if [ -f "$3" ]; then
      sed 's/^/    | /' "$3"
    fi
  fi
}

assert_empty_file() {
  if [ ! -s "$2" ]; then
    pass "$1"
  else
    fail "$1"
    sed 's/^/    | /' "$2"
  fi
}

# assert_same_bytes <desc> <expected file> <actual file>
assert_same_bytes() {
  if cmp -s "$2" "$3"; then
    pass "$1"
  else
    fail "$1"
    printf '    %s differs from %s\n' "$3" "$2"
  fi
}

# run_pv <command...> — run with stdout in $_out, stderr in $_err, exit code
# in $_rc.
run_pv() {
  set +e
  "$@" >"$_out" 2>"$_err"
  _rc=$?
  set -e
}

_tmp="$(mktemp -d)"
_tmp_abs="$(cd "$_tmp" && pwd)"
_out="$_tmp/stdout"
_err="$_tmp/stderr"
_rc=0

PV_STUB_LOG="$_tmp/stub.log"
export PV_STUB_LOG

_space_dir="$_tmp/dir with space"
mkdir -p "$_space_dir" "$_tmp/bin" "$_tmp/out"
_html="$_space_dir/page.html"
printf '<html><body>page</body></html>\n' > "$_html"
_html_abs="$_tmp_abs/dir with space/page.html"
_html_url="file://$_tmp_abs/dir%20with%20space/page.html"
_odd_html="$_space_dir/100% #1?.html"
printf '<html><body>odd</body></html>\n' > "$_odd_html"
_odd_url="file://$_tmp_abs/dir%20with%20space/100%25%20%231%3F.html"
_png="$_tmp/out/shot.png"
_png_abs="$_tmp_abs/out/shot.png"

# Stub opener: records its argument count and arguments, prints noise.
cat > "$_tmp/bin/stub-opener" <<'STUB'
#!/bin/sh
printf '%s\n' "$#" > "$PV_STUB_LOG"
printf '%s\n' "$@" >> "$PV_STUB_LOG"
echo "opener noise on stdout"
STUB
cat > "$_tmp/bin/failing-opener" <<'STUB'
#!/bin/sh
exit 3
STUB
# Stub browser: records its arguments and writes the --screenshot target.
cat > "$_tmp/bin/stub-browser" <<'STUB'
#!/bin/sh
: > "$PV_STUB_LOG"
_target=""
for _arg in "$@"; do
  printf '%s\n' "$_arg" >> "$PV_STUB_LOG"
  case "$_arg" in
    --screenshot=*) _target="${_arg#--screenshot=}" ;;
  esac
done
echo "browser noise on stdout"
echo "browser noise on stderr" >&2
printf 'PNG' > "$_target"
STUB
# Stub browser that exits 0 without writing the PNG.
cat > "$_tmp/bin/silent-browser" <<'STUB'
#!/bin/sh
echo "stub browser failure detail" >&2
exit 0
STUB
chmod +x "$_tmp/bin/stub-opener" "$_tmp/bin/failing-opener" "$_tmp/bin/stub-browser" "$_tmp/bin/silent-browser"
printf '#!/bin/sh\n' > "$_tmp/bin/not-executable"

# PATH-only directories for the OS-default opener resolution.
mkdir -p "$_tmp/path-darwin" "$_tmp/path-linux" "$_tmp/path-bare"
printf '#!/bin/sh\necho Darwin\n' > "$_tmp/path-darwin/uname"
cp "$_tmp/bin/stub-opener" "$_tmp/path-darwin/open"
printf '#!/bin/sh\necho Linux\n' > "$_tmp/path-linux/uname"
cp "$_tmp/bin/stub-opener" "$_tmp/path-linux/xdg-open"
printf '#!/bin/sh\necho Linux\n' > "$_tmp/path-bare/uname"
chmod +x "$_tmp/path-darwin/uname" "$_tmp/path-darwin/open" "$_tmp/path-linux/uname" "$_tmp/path-linux/xdg-open" "$_tmp/path-bare/uname"

printf '==> usage\n'
run_pv "$PLAN_VISUAL"
assert_eq "no args exits 1" 1 "$_rc"
assert_contains "no args prints usage on stderr" "Usage:" "$_err"
assert_empty_file "no args prints nothing on stdout" "$_out"
run_pv "$PLAN_VISUAL" --help
assert_eq "--help exits 0" 0 "$_rc"
assert_contains "--help prints usage on stdout" "Usage:" "$_out"
assert_contains "--help documents RALPH_PLAN_VISUAL_OPENER" "RALPH_PLAN_VISUAL_OPENER" "$_out"
assert_contains "--help documents RALPH_PLAN_VISUAL_BROWSER" "RALPH_PLAN_VISUAL_BROWSER" "$_out"
run_pv "$PLAN_VISUAL" bogus
assert_eq "unknown subcommand exits 1" 1 "$_rc"
assert_contains "unknown subcommand prints usage on stderr" "Usage:" "$_err"
run_pv "$PLAN_VISUAL" open
assert_eq "open without a file exits 1" 1 "$_rc"
run_pv "$PLAN_VISUAL" shot "$_html"
assert_eq "shot without a png path exits 1" 1 "$_rc"
run_pv "$PLAN_VISUAL" digest
assert_eq "digest without a plan exits 1" 1 "$_rc"

printf '==> open\n'
rm -f "$PV_STUB_LOG"
run_pv env RALPH_PLAN_VISUAL_OPENER="$_tmp/bin/stub-opener" "$PLAN_VISUAL" open "$_tmp/missing.html"
assert_eq "missing file exits 1" 1 "$_rc"
assert_contains "missing file is named on stderr" "missing.html" "$_err"
if [ ! -e "$PV_STUB_LOG" ]; then
  pass "missing file does not run the opener"
else
  fail "missing file does not run the opener"
fi

rm -f "$PV_STUB_LOG"
run_pv env RALPH_PLAN_VISUAL_OPENER="$_tmp/bin/stub-opener" "$PLAN_VISUAL" open "$_html"
assert_eq "stub opener exits 0" 0 "$_rc"
assert_eq "stdout is only the absolute path (opener stdout kept off it)" "$_html_abs" "$(cat "$_out")"
assert_eq "opener gets exactly one argument" 1 "$(sed -n 1p "$PV_STUB_LOG")"
assert_eq "opener gets the absolute path" "$_html_abs" "$(sed -n 2p "$PV_STUB_LOG")"

rm -f "$PV_STUB_LOG"
_saved_pwd="$(pwd)"
cd "$_space_dir"
run_pv env RALPH_PLAN_VISUAL_OPENER="$_tmp/bin/stub-opener" "$PLAN_VISUAL" open page.html
cd "$_saved_pwd"
assert_eq "relative path exits 0" 0 "$_rc"
assert_eq "relative path is printed as absolute" "$_html_abs" "$(cat "$_out")"
assert_eq "relative path reaches the opener as absolute" "$_html_abs" "$(sed -n 2p "$PV_STUB_LOG")"

run_pv env RALPH_PLAN_VISUAL_OPENER=none "$PLAN_VISUAL" open "$_html"
assert_eq "RALPH_PLAN_VISUAL_OPENER=none exits 0" 0 "$_rc"
assert_eq "RALPH_PLAN_VISUAL_OPENER=none prints the path" "$_html_abs" "$(cat "$_out")"
assert_contains "RALPH_PLAN_VISUAL_OPENER=none notes it on stderr" "open the path above manually" "$_err"

run_pv env RALPH_PLAN_VISUAL_OPENER="$_tmp/bin/failing-opener" "$PLAN_VISUAL" open "$_html"
assert_eq "failing opener still exits 0" 0 "$_rc"
assert_eq "failing opener still prints the path" "$_html_abs" "$(cat "$_out")"
assert_contains "failing opener notes it on stderr" "open the path above manually" "$_err"

rm -f "$PV_STUB_LOG"
run_pv env PATH="$_tmp/path-darwin" "$SH" "$PLAN_VISUAL" open "$_html"
assert_eq "Darwin default opener exits 0" 0 "$_rc"
assert_eq "Darwin default opener is open" "$_html_abs" "$(sed -n 2p "$PV_STUB_LOG" 2>/dev/null || true)"

rm -f "$PV_STUB_LOG"
run_pv env PATH="$_tmp/path-linux" "$SH" "$PLAN_VISUAL" open "$_html"
assert_eq "non-Darwin default opener exits 0" 0 "$_rc"
assert_eq "non-Darwin default opener is xdg-open" "$_html_abs" "$(sed -n 2p "$PV_STUB_LOG" 2>/dev/null || true)"

run_pv env PATH="$_tmp/path-bare" "$SH" "$PLAN_VISUAL" open "$_html"
assert_eq "no opener on PATH exits 0" 0 "$_rc"
assert_eq "no opener on PATH prints the path" "$_html_abs" "$(cat "$_out")"
assert_contains "no opener on PATH notes it on stderr" "open the path above manually" "$_err"

printf '==> shot\n'
run_pv env RALPH_PLAN_VISUAL_BROWSER="$_tmp/bin/stub-browser" "$PLAN_VISUAL" shot "$_tmp/missing.html" "$_png"
assert_eq "missing html exits 1" 1 "$_rc"

run_pv env RALPH_PLAN_VISUAL_BROWSER=none "$PLAN_VISUAL" shot "$_html" "$_png"
assert_eq "RALPH_PLAN_VISUAL_BROWSER=none exits 2" 2 "$_rc"
assert_contains "RALPH_PLAN_VISUAL_BROWSER=none says so on stderr" "no browser" "$_err"

run_pv env RALPH_PLAN_VISUAL_BROWSER="$_tmp/bin/not-executable" "$PLAN_VISUAL" shot "$_html" "$_png"
assert_eq "non-executable browser path exits 2" 2 "$_rc"
assert_contains "non-executable browser path is named on stderr" "not-executable" "$_err"

run_pv env RALPH_PLAN_VISUAL_BROWSER="$_tmp/bin" "$PLAN_VISUAL" shot "$_html" "$_png"
assert_eq "directory as browser path exits 2" 2 "$_rc"

rm -f "$PV_STUB_LOG" "$_png"
run_pv env RALPH_PLAN_VISUAL_BROWSER="$_tmp/bin/stub-browser" "$PLAN_VISUAL" shot "$_html" "$_png" --fragment overview --width 1150 --height 470 --scale 2
assert_eq "stub browser exits 0" 0 "$_rc"
assert_eq "stdout is only the absolute png path" "$_png_abs" "$(cat "$_out")"
if [ -s "$_png" ]; then
  pass "png is written"
else
  fail "png is written"
fi
assert_empty_file "browser noise is hidden on success" "$_err"
assert_has_line "browser gets --headless=new" "--headless=new" "$PV_STUB_LOG"
assert_has_line "browser gets --window-size=W,H" "--window-size=1150,470" "$PV_STUB_LOG"
assert_has_line "browser gets --force-device-scale-factor=N" "--force-device-scale-factor=2" "$PV_STUB_LOG"
assert_has_line "browser gets the absolute --screenshot target" "--screenshot=$_png_abs" "$PV_STUB_LOG"
assert_has_line "file URL encodes the space and carries #fragment" "$_html_url#overview" "$PV_STUB_LOG"

rm -f "$PV_STUB_LOG" "$_png"
run_pv env RALPH_PLAN_VISUAL_BROWSER="$_tmp/bin/stub-browser" "$PLAN_VISUAL" shot "$_html" "$_png"
assert_eq "defaults exit 0" 0 "$_rc"
assert_has_line "default window size is 1200,3000" "--window-size=1200,3000" "$PV_STUB_LOG"
assert_has_line "default scale is 1" "--force-device-scale-factor=1" "$PV_STUB_LOG"
assert_has_line "no --fragment means no #" "$_html_url" "$PV_STUB_LOG"

rm -f "$PV_STUB_LOG" "$_png"
run_pv env RALPH_PLAN_VISUAL_BROWSER="$_tmp/bin/stub-browser" "$PLAN_VISUAL" shot "$_html" "$_png" --fragment '#overview'
assert_has_line "--fragment with a leading # does not double it" "$_html_url#overview" "$PV_STUB_LOG"

rm -f "$PV_STUB_LOG" "$_png"
run_pv env RALPH_PLAN_VISUAL_BROWSER="$_tmp/bin/stub-browser" "$PLAN_VISUAL" shot "$_odd_html" "$_png"
assert_eq "html name with % # ? exits 0" 0 "$_rc"
assert_has_line "file URL encodes % first, then space, # and ?" "$_odd_url" "$PV_STUB_LOG"

printf 'stale\n' > "$_png"
run_pv env RALPH_PLAN_VISUAL_BROWSER="$_tmp/bin/silent-browser" "$PLAN_VISUAL" shot "$_html" "$_png"
assert_eq "browser that writes nothing exits 1 (stale png ignored)" 1 "$_rc"
assert_empty_file "browser that writes nothing prints no path" "$_out"
assert_contains "browser output is shown on failure" "stub browser failure detail" "$_err"

run_pv env RALPH_PLAN_VISUAL_BROWSER="$_tmp/bin/stub-browser" "$PLAN_VISUAL" shot "$_html" "$_tmp/no-such-dir/shot.png"
assert_eq "missing output directory exits 1" 1 "$_rc"

# A bad <png> must be rejected before the browser could write over <html>.
_html_orig="$_tmp/page.html.orig"
cp "$_html" "$_html_orig"
_html_as_png="$_space_dir/page-as.png"
cp "$_html" "$_html_as_png"
rm -f "$PV_STUB_LOG"
run_pv env RALPH_PLAN_VISUAL_BROWSER="$_tmp/bin/stub-browser" "$PLAN_VISUAL" shot "$_html" ""
assert_eq "empty png path exits 1" 1 "$_rc"
assert_contains "empty png path prints usage on stderr" "Usage:" "$_err"
assert_same_bytes "empty png path leaves the html untouched" "$_html_orig" "$_html"
run_pv env RALPH_PLAN_VISUAL_BROWSER="$_tmp/bin/stub-browser" "$PLAN_VISUAL" shot "$_html" "$_html"
assert_eq "png path not ending in .png (the html itself) exits 1" 1 "$_rc"
assert_contains "png path not ending in .png is named on stderr" "must end in .png" "$_err"
assert_same_bytes "png path not ending in .png leaves the html untouched" "$_html_orig" "$_html"
run_pv env RALPH_PLAN_VISUAL_BROWSER="$_tmp/bin/stub-browser" "$PLAN_VISUAL" shot "$_html_as_png" "$_tmp/dir with space/../dir with space/page-as.png"
assert_eq "png path resolving to the html exits 1" 1 "$_rc"
assert_contains "png path resolving to the html says so on stderr" "same file as the input html" "$_err"
assert_same_bytes "png path resolving to the html leaves the html untouched" "$_html_orig" "$_html_as_png"
if [ ! -e "$PV_STUB_LOG" ]; then
  pass "rejected png paths do not start the browser"
else
  fail "rejected png paths do not start the browser"
fi

rm -f "$PV_STUB_LOG"
run_pv env RALPH_PLAN_VISUAL_BROWSER="$_tmp/bin/stub-browser" "$PLAN_VISUAL" shot "$_html" "$_tmp/out/upper.PNG"
assert_eq "upper-case .PNG is accepted" 0 "$_rc"
assert_has_line "upper-case .PNG reaches the browser" "--screenshot=$_tmp_abs/out/upper.PNG" "$PV_STUB_LOG"

rm -f "$PV_STUB_LOG"
run_pv env RALPH_PLAN_VISUAL_BROWSER="$_tmp/bin/stub-browser" "$PLAN_VISUAL" shot "$_html" "$_png" --width abc
assert_eq "non-numeric width exits 1" 1 "$_rc"
if [ ! -e "$PV_STUB_LOG" ]; then
  pass "invalid width does not start the browser"
else
  fail "invalid width does not start the browser"
fi
run_pv env RALPH_PLAN_VISUAL_BROWSER="$_tmp/bin/stub-browser" "$PLAN_VISUAL" shot "$_html" "$_png" --width 0
assert_eq "zero width exits 1" 1 "$_rc"
run_pv env RALPH_PLAN_VISUAL_BROWSER="$_tmp/bin/stub-browser" "$PLAN_VISUAL" shot "$_html" "$_png" --height -5
assert_eq "negative height exits 1" 1 "$_rc"
run_pv env RALPH_PLAN_VISUAL_BROWSER="$_tmp/bin/stub-browser" "$PLAN_VISUAL" shot "$_html" "$_png" --scale 1.5
assert_eq "fractional scale exits 1" 1 "$_rc"
run_pv env RALPH_PLAN_VISUAL_BROWSER="$_tmp/bin/stub-browser" "$PLAN_VISUAL" shot "$_html" "$_png" --width
assert_eq "option without a value exits 1" 1 "$_rc"
run_pv env RALPH_PLAN_VISUAL_BROWSER="$_tmp/bin/stub-browser" "$PLAN_VISUAL" shot "$_html" "$_png" --bogus 1
assert_eq "unknown option exits 1" 1 "$_rc"

printf '==> digest\n'
# write_plan <file> <status> <approved> <branch> <owner> <objective> <checklist item> <notes>
write_plan() {
  printf '%s\n' \
    '# sample-plan' \
    '' \
    "- Status: $2" \
    "- Approved: $3" \
    "- Owner: $5" \
    "- Branch: $4" \
    '' \
    '## Objective' \
    '' \
    "$6" \
    '' \
    '## Progress checklist' \
    '' \
    "$7" \
    '- [ ] PR created' \
    '' \
    '## Notes' \
    '' \
    "$8" > "$1"
}

_base="$_tmp/base.md"
write_plan "$_base" Draft N/A TBD someone 'Do the thing.' '- [ ] Plan reviewed' 'Kept after the checklist.'
_base_digest="$("$PLAN_VISUAL" digest "$_base")"
# Recorded with an independent Python implementation of the same rule.
assert_eq "known answer for the base fixture" "1174b9b1faae" "$_base_digest"
case "$_base_digest" in
  [0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f]) pass "digest is 12 lowercase hex characters" ;;
  *) fail "digest is 12 lowercase hex characters (got: $_base_digest)" ;;
esac

_v="$_tmp/variant.md"
write_plan "$_v" Approved '2026-10-05 sha256:0123456789ab' feat/sample-plan someone 'Do the thing.' '- [x] Plan reviewed' 'Kept after the checklist.'
assert_eq "only Status/Approved/Branch and checklist body differ: same digest" "$_base_digest" "$("$PLAN_VISUAL" digest "$_v")"

grep -v -e '^- Status:' -e '^- Approved:' -e '^- Branch:' "$_base" > "$_v"
assert_eq "Status/Approved/Branch lines removed: same digest" "$_base_digest" "$("$PLAN_VISUAL" digest "$_v")"

printf '%s' "$(cat "$_base")" > "$_v"
assert_ne "fixture without a final newline really lacks it" "0a" "$(tail -c 1 "$_v" | od -An -tx1 | tr -d ' \n')"
assert_eq "no final newline: same digest" "$_base_digest" "$("$PLAN_VISUAL" digest "$_v")"

write_plan "$_v" Draft N/A TBD someone 'Do the thing!' '- [ ] Plan reviewed' 'Kept after the checklist.'
assert_ne "one character in the body differs: different digest" "$_base_digest" "$("$PLAN_VISUAL" digest "$_v")"

write_plan "$_v" Draft N/A TBD someone-else 'Do the thing.' '- [ ] Plan reviewed' 'Kept after the checklist.'
assert_ne "another header line differs: different digest" "$_base_digest" "$("$PLAN_VISUAL" digest "$_v")"

write_plan "$_v" Draft N/A TBD someone 'Do the thing.' '- [ ] Plan reviewed' 'Kept after the checklist!'
assert_ne "a line in a section after Progress checklist differs: different digest" "$_base_digest" "$("$PLAN_VISUAL" digest "$_v")"

write_plan "$_v" Draft N/A TBD someone '  - Status: nested, not a header' '- [ ] Plan reviewed' 'Kept after the checklist.'
_nested_a="$("$PLAN_VISUAL" digest "$_v")"
write_plan "$_v" Draft N/A TBD someone '  - Status: nested, changed' '- [ ] Plan reviewed' 'Kept after the checklist.'
assert_ne "an indented - Status: line is not excluded" "$_nested_a" "$("$PLAN_VISUAL" digest "$_v")"

run_pv "$PLAN_VISUAL" digest "$_tmp/missing.md"
assert_eq "missing plan exits 1" 1 "$_rc"
assert_empty_file "missing plan prints no digest" "$_out"

# Each SHA-256 tool alone on PATH must give the same value.
_awk="$(command -v awk)"
for _tool in sha256sum shasum; do
  if ! _tool_path="$(command -v "$_tool" 2>/dev/null)"; then
    printf '  SKIP: %s is not installed here\n' "$_tool"
    continue
  fi
  mkdir -p "$_tmp/path-$_tool"
  ln -s "$_awk" "$_tmp/path-$_tool/awk"
  ln -s "$_tool_path" "$_tmp/path-$_tool/$_tool"
  run_pv env PATH="$_tmp/path-$_tool" "$SH" "$PLAN_VISUAL" digest "$_base"
  assert_eq "$_tool alone gives the known answer" "1174b9b1faae" "$(cat "$_out")"
done

mkdir -p "$_tmp/path-nohash"
ln -s "$_awk" "$_tmp/path-nohash/awk"
run_pv env PATH="$_tmp/path-nohash" "$SH" "$PLAN_VISUAL" digest "$_base"
assert_eq "no SHA-256 tool exits 1" 1 "$_rc"
assert_empty_file "no SHA-256 tool prints no digest" "$_out"

printf '\nplan-visual tests: %s passed, %s failed, %s total\n' "$_pass" "$_fail" "$_total"
[ "$_fail" -eq 0 ]
