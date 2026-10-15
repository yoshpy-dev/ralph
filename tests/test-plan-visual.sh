#!/usr/bin/env sh
set -eu

# test-plan-visual.sh — scripts/plan-visual.sh open / shot / lint / digest.
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

# assert_only_rule <desc> <rule> <file> — the file has at least one line and
# every line reports <rule> (": L<n>: ").
assert_only_rule() {
  if [ -s "$3" ] && ! grep -Fvq -- ": $2: " "$3"; then
    pass "$1"
  else
    fail "$1"
    printf '    expected only %s lines, got:\n' "$2"
    sed 's/^/    | /' "$3"
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
assert_contains "--help documents that ticked boxes keep the digest" '"- [x]" or "- [X]" box' "$_out"
# /plan tells an old helper apart by this exact line.
assert_contains "--help lists the lint subcommand" "  scripts/plan-visual.sh lint <html>" "$_out"
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

printf '==> lint\n'
# Parts of a page in the plain-figure layout that passes lint; each case below
# swaps, moves, or drops one part. Its figures in order: overview, easy-2,
# detail-overview, detail-2, and one without an id (figure#5).
_pg_head='<!doctype html>
<html lang="ja">
<head>
<meta charset="utf-8">
<title>Plan visual review: sample</title>
<style>
  body:has(figure:target) > figure:not(:target) { display: none; }
  .mono { font-family: ui-monospace, monospace; }
</style>
<!-- <figure id="overview"> inside a comment is not a tag -->
</head>
<body>
<main>
  <div class="chrome"><h1>見出し</h1><div class="meta"><code>docs/plans/active/sample.md</code></div></div>'
_pg_overview='  <figure class="easy" id="overview">
    <h2>承認する人は、上の図だけで決められる</h2>
    <svg viewBox="0 0 1080 330" role="img" aria-label="今は読み解いてから決める。これからは図を読んで決める。">
      <rect class="n-old"/><rect class="n-old"/><rect class="n-new"/><rect class="box n-new"/><rect class="n-fail"/>
      <text x="20" y="92" class="t-row now">今</text>
    </svg>
    <div class="foot"><p class="words">人が打つのは /director と ralph org stop --all。1/3 が終わったら見る。</p><span class="made nomono">S1・S2 で作る</span></div>
  </figure>'
_pg_easy2='  <figure class="easy" id="easy-2">
    <h2>決まりは機械で確かめる</h2>
    <svg viewBox="0 0 1080 180" role="img" aria-label="書く、確かめる、見せる">
      <style>.t { font: 13px/1.4 sans-serif; }</style>
      <rect class="n-old"/><rect class="n-new"/>
    </svg>
  </figure>'
_pg_before='  <section class="before chrome">
    <h2>承認の前に</h2>
    <ul><li>3 回に分けて作ります</li></ul>
  </section>'
_pg_fold_open='  <details class="deep">
    <summary>くわしい図</summary>
    <div class="chrome"><code>scripts/plan-visual.sh</code> と <span class="mono">workspace_created</span></div>'
_pg_details='    <figure id="detail-overview">
      <h3>詳細 1</h3>
      <svg viewBox="0 0 1080 420" role="img" aria-label="変わるファイル"><text class="mono">internal/org/x.go</text></svg>
    </figure>
    <figure id="detail-2"><h3>詳細 2</h3><details><summary>入れ子</summary>中身</details></figure>
    <figure><h3>詳細 3</h3></figure>
    <p class="out chrome">この図にないもの: なし</p>'
_pg_fold_close='  </details>'

# write_page <file> <part>... — the head, the parts, then the closing tags.
write_page() {
  _page="$1"
  shift
  {
    printf '%s\n' "$_pg_head"
    printf '%s\n' "$@"
    printf '%s\n' '</main>' '</body>' '</html>'
  } > "$_page"
}

# lint_case <name> <part>... — write $_tmp/lint-<name>.html with write_page
# and lint it.
lint_case() {
  _case_page="$_tmp/lint-$1.html"
  shift
  write_page "$_case_page" "$@"
  run_pv "$PLAN_VISUAL" lint "$_case_page"
}

lint_case good "$_pg_overview" "$_pg_easy2" "$_pg_before" "$_pg_fold_open" "$_pg_details" "$_pg_fold_close"
_good="$_case_page"
assert_eq "good page exits 0 (/director, --all, 1/3, class nomono, and a style body in an easy figure; paths, <code> and mono in the fold)" 0 "$_rc"
assert_empty_file "good page prints no finding" "$_out"
assert_empty_file "good page prints nothing on stderr" "$_err"

_ov_id_first="$(printf '%s\n' "$_pg_overview" | sed 's/<figure class="easy" id="overview">/<figure id="overview" class="easy">/')"
lint_case id-first "$_ov_id_first" "$_pg_easy2" "$_pg_before" "$_pg_fold_open" "$_pg_details" "$_pg_fold_close"
assert_contains "id-first fixture really has id before class" '<figure id="overview" class="easy">' "$_case_page"
assert_eq "id before class in the overview tag exits 0" 0 "$_rc"
assert_empty_file "id before class prints no finding" "$_out"

sed "s/\"/'/g" "$_good" > "$_tmp/lint-single.html"
assert_contains "single-quote fixture really uses single quotes" "<figure class='easy' id='overview'>" "$_tmp/lint-single.html"
run_pv "$PLAN_VISUAL" lint "$_tmp/lint-single.html"
assert_eq "single-quoted attributes exit 0" 0 "$_rc"
assert_empty_file "single-quoted attributes print no finding" "$_out"

_ov_split='  <figure
      class="easy"
      id="overview">
    <h2>承認する人は、上の図だけで決められる</h2>
    <svg viewBox="0 0 1080 330"
         role="img"
         aria-label="今 -> これから">
      <rect class="n-new"/>
    </svg>
  </figure>'
lint_case split "$_ov_split" "$_pg_before" "$_pg_fold_open" "$_pg_details" "$_pg_fold_close"
assert_eq "start tags split over lines exit 0" 0 "$_rc"
assert_empty_file "start tags split over lines print no finding" "$_out"

_ov_gt='  <figure data-note="a > b" class="easy" id="overview">
    <h2>見出し</h2>
    <svg viewBox="0 0 10 10" aria-label="図"><rect class="n-new"/></svg>
  </figure>'
lint_case quoted-gt "$_ov_gt" "$_pg_before" "$_pg_fold_open" "$_pg_details" "$_pg_fold_close"
assert_eq "a > inside a quoted value does not end the tag" 0 "$_rc"
assert_empty_file "a > inside a quoted value prints no finding" "$_out"

# awk would take a bare a=b.html operand for an assignment and read stdin.
cp "$_good" "$_tmp/a=b.html"
_saved_pwd="$(pwd)"
cd "$_tmp"
run_pv "$PLAN_VISUAL" lint a=b.html </dev/null
cd "$_saved_pwd"
assert_eq "a relative file name with = is read as the file" 0 "$_rc"
assert_empty_file "a relative file name with = prints no finding" "$_out"

# L1
_ov_renamed="$(printf '%s\n' "$_pg_overview" | sed 's/id="overview"/id="easy-1"/')"
lint_case l1-missing "$_ov_renamed" "$_pg_easy2" "$_pg_before" "$_pg_fold_open" "$_pg_details" "$_pg_fold_close"
assert_eq "no id=overview exits 1" 1 "$_rc"
assert_has_line "no id=overview is reported (the one in a comment does not count)" 'page: L1: no element has id="overview"' "$_out"
assert_only_rule "no id=overview breaks only L1" L1 "$_out"

_before_ov="$(printf '%s\n' "$_pg_before" | sed 's/<section class="before chrome">/<section class="before chrome" id="overview">/')"
lint_case l1-section "$_ov_renamed" "$_pg_easy2" "$_before_ov" "$_pg_fold_open" "$_pg_details" "$_pg_fold_close"
assert_eq "id=overview on a section exits 1" 1 "$_rc"
assert_has_line "id=overview on a section is reported" 'overview: L1: id="overview" is on <section>, not on <figure class="easy">' "$_out"
assert_only_rule "id=overview on a section breaks only L1" L1 "$_out"

_easy2_ov="$(printf '%s\n' "$_pg_easy2" | sed 's/id="easy-2"/id="overview"/')"
lint_case l1-two "$_pg_overview" "$_easy2_ov" "$_pg_before" "$_pg_fold_open" "$_pg_details" "$_pg_fold_close"
assert_eq "two id=overview exit 1" 1 "$_rc"
assert_has_line "two id=overview are reported" 'page: L1: 2 elements have id="overview", expected 1' "$_out"
assert_only_rule "two id=overview break only L1" L1 "$_out"

# The page layout before this rule: overview is not easy, no fold.
_old_page='  <figure id="overview">
    <h2>全体図</h2>
    <svg viewBox="0 0 1080 420" aria-label="全体"><text class="mono">internal/org/x.go</text></svg>
  </figure>
  <figure id="detail-1"><h3>詳細 1</h3></figure>'
lint_case old "$_old_page"
assert_eq "old-format page exits 1" 1 "$_rc"
assert_has_line "old-format page is reported under L1" 'overview: L1: the id="overview" figure lacks class easy' "$_out"

# L2
_easy3="$(printf '%s\n' "$_pg_easy2" | sed 's/id="easy-2"/id="easy-3"/')"
_easy4="$(printf '%s\n' "$_pg_easy2" | sed 's/id="easy-2"/id="easy-4"/')"
lint_case l2-four "$_pg_overview" "$_pg_easy2" "$_easy3" "$_easy4" "$_pg_before" "$_pg_fold_open" "$_pg_details" "$_pg_fold_close"
assert_eq "four easy figures exit 1" 1 "$_rc"
assert_has_line "four easy figures are reported" 'page: L2: 4 easy figures (<figure class="easy">), expected 1 to 3' "$_out"
assert_only_rule "four easy figures break only L2" L2 "$_out"

lint_case l2-fold "$_pg_overview" "$_pg_before" "$_pg_fold_open" "$_easy3" "$_pg_details" "$_pg_fold_close"
assert_eq "an easy figure inside the fold exits 1" 1 "$_rc"
assert_has_line "an easy figure inside the fold is reported" 'easy-3: L2: easy figure inside the fold' "$_out"
assert_only_rule "an easy figure inside the fold breaks only L2" L2 "$_out"

# L3
_easy2_l3='  <figure class="easy" id="easy-2">
    <h2>決まりは機械で確かめる</h2>
    <svg viewBox="0 0 1080 180" role="img" aria-label="書く">
      <rect class="n-old"/><text class="label mono">x</text>
    </svg>
    <p>internal/org を直す。S1/S2 で作る。plan-visual/&lt;slug&gt;.html に書く。README.md も。workspace_created を出す。<code>x</code></p>
  </figure>'
lint_case l3 "$_pg_overview" "$_easy2_l3" "$_pg_before" "$_pg_fold_open" "$_pg_details" "$_pg_fold_close"
assert_eq "code-like text in an easy figure exits 1" 1 "$_rc"
assert_has_line "class mono is reported" 'easy-2: L3: class mono on <text>' "$_out"
assert_has_line "<code> is reported" 'easy-2: L3: <code> element' "$_out"
assert_has_line "a path is reported with the word" 'easy-2: L3: path-like text: internal/org' "$_out"
assert_has_line "S1/S2 is reported as path-like" 'easy-2: L3: path-like text: S1/S2' "$_out"
assert_has_line "&lt;slug&gt;.html is reported by its extension" 'easy-2: L3: file extension: plan-visual/&lt;slug&gt;.html' "$_out"
assert_has_line "README.md is reported by its extension" 'easy-2: L3: file extension: README.md' "$_out"
assert_has_line "a snake_case word is reported" 'easy-2: L3: snake_case identifier: workspace_created' "$_out"
assert_eq "code-like text gives exactly these 7 findings" 7 "$(wc -l < "$_out" | tr -d ' ')"
assert_only_rule "code-like text breaks only L3" L3 "$_out"

# L4
_easy2_l4='  <figure class="easy" id="easy-2">
    <h3>見出しが h3</h3>
    <svg viewBox="0 0 1080 180" role="img" aria-label="">
      <rect class="n-old"/><rect class="n-old"/><rect class="n-new"/><rect class="n-new"/><rect class="n-fail"/><g class="group n-new"></g>
    </svg>
  </figure>'
lint_case l4 "$_pg_overview" "$_easy2_l4" "$_pg_before" "$_pg_fold_open" "$_pg_details" "$_pg_fold_close"
assert_eq "an easy figure with 6 boxes, no h2, and an empty aria-label exits 1" 1 "$_rc"
assert_has_line "6 boxes are reported" 'easy-2: L4: 6 boxes (class n-*), at most 5' "$_out"
assert_has_line "a missing h2 is reported" 'easy-2: L4: no <h2> heading' "$_out"
assert_has_line "an empty aria-label is reported" 'easy-2: L4: no <svg> with a non-empty aria-label' "$_out"
assert_only_rule "6 boxes, no h2, and an empty aria-label break only L4" L4 "$_out"

_easy2_noaria="$(printf '%s\n' "$_pg_easy2" | sed 's/ aria-label="[^"]*"//')"
lint_case l4-noaria "$_pg_overview" "$_easy2_noaria" "$_pg_before" "$_pg_fold_open" "$_pg_details" "$_pg_fold_close"
assert_eq "an svg without aria-label exits 1" 1 "$_rc"
assert_has_line "an svg without aria-label is reported" 'easy-2: L4: no <svg> with a non-empty aria-label' "$_out"
assert_only_rule "an svg without aria-label breaks only L4" L4 "$_out"

_easy2_blank="$(printf '%s\n' "$_pg_easy2" | sed 's/ aria-label="[^"]*"/ aria-label="  "/')"
lint_case l4-blank "$_pg_overview" "$_easy2_blank" "$_pg_before" "$_pg_fold_open" "$_pg_details" "$_pg_fold_close"
assert_eq "an aria-label of spaces only exits 1" 1 "$_rc"
assert_only_rule "an aria-label of spaces only breaks only L4" L4 "$_out"

# L5
lint_case l5-missing "$_pg_overview" "$_pg_easy2" "$_pg_fold_open" "$_pg_details" "$_pg_fold_close"
assert_eq "no before section exits 1" 1 "$_rc"
assert_has_line "no before section is reported" 'page: L5: no <section class="before">' "$_out"
assert_only_rule "no before section breaks only L5" L5 "$_out"

lint_case l5-fold "$_pg_overview" "$_pg_easy2" "$_pg_fold_open" "$_pg_before" "$_pg_details" "$_pg_fold_close"
assert_eq "a before section inside the fold exits 1" 1 "$_rc"
assert_has_line "a before section inside the fold is reported" 'page: L5: <section class="before"> is inside the fold' "$_out"
assert_only_rule "a before section inside the fold breaks only L5" L5 "$_out"

# L6
lint_case l6-nofold "$_pg_overview" "$_pg_easy2" "$_pg_before" "$_pg_details"
assert_eq "no fold exits 1" 1 "$_rc"
assert_has_line "no fold is reported" 'page: L6: no fold (<details class="deep">)' "$_out"
assert_has_line "no detail-overview inside the fold is reported" 'page: L6: no <figure id="detail-overview"> inside the fold' "$_out"
assert_has_line "a detail figure outside the fold is reported" 'detail-overview: L6: figure without class easy is outside the fold' "$_out"
assert_has_line "a figure without an id is labelled by its order" 'figure#5: L6: figure without class easy is outside the fold' "$_out"
assert_only_rule "no fold breaks only L6" L6 "$_out"

lint_case l6-detail-after "$_pg_overview" "$_pg_easy2" "$_pg_before" "$_pg_fold_open" "$_pg_fold_close" "$_pg_details"
assert_eq "detail figures after the fold exit 1" 1 "$_rc"
assert_has_line "detail figures after the fold are reported" 'page: L6: no <figure id="detail-overview"> inside the fold' "$_out"
assert_only_rule "detail figures after the fold break only L6" L6 "$_out"

lint_case l6-two "$_pg_overview" "$_pg_easy2" "$_pg_before" "$_pg_fold_open" "$_pg_details" "$_pg_fold_close" '  <details class="deep"><summary>もう 1 つ</summary></details>'
assert_eq "two folds exit 1" 1 "$_rc"
assert_has_line "two folds are reported" 'page: L6: 2 folds (<details class="deep">), expected 1' "$_out"
assert_only_rule "two folds break only L6" L6 "$_out"

lint_case l6-open-fold "$_pg_overview" "$_pg_easy2" "$_pg_before" "$_pg_fold_open" "$_pg_details"
assert_eq "a fold without </details> exits 1" 1 "$_rc"
assert_has_line "a fold without </details> is reported" 'page: L6: a fold has no matching </details>' "$_out"
assert_only_rule "a fold without </details> breaks only L6" L6 "$_out"

_details_open="$(printf '%s\n' "$_pg_details" | sed '/<figure><h3>/s|</figure>||')"
lint_case l6-open-figure "$_pg_overview" "$_pg_easy2" "$_pg_before" "$_pg_fold_open" "$_details_open" "$_pg_fold_close"
assert_eq "a figure without </figure> exits 1" 1 "$_rc"
assert_has_line "a figure without </figure> is reported" 'figure#5: L6: no </figure> after this figure' "$_out"
assert_only_rule "a figure without </figure> breaks only L6" L6 "$_out"

: > "$_tmp/lint-empty.html"
run_pv "$PLAN_VISUAL" lint "$_tmp/lint-empty.html"
assert_eq "empty file exits 1" 1 "$_rc"
assert_has_line "empty file is reported under L1" 'page: L1: no element has id="overview"' "$_out"

# Cut off inside a quoted value: no closing quote, >, </figure>, or </html>.
printf '%s\n' "$_pg_head" '  <figure class="easy" id="overview"><h2>途中</h2><svg aria-label="終わらない' > "$_tmp/lint-cut.html"
run_pv "$PLAN_VISUAL" lint "$_tmp/lint-cut.html"
assert_eq "a page cut off inside a tag exits 1" 1 "$_rc"
assert_has_line "a page cut off inside a tag reports the open figure" 'overview: L6: no </figure> after this figure' "$_out"

run_pv "$PLAN_VISUAL" lint
assert_eq "lint without a file exits 1" 1 "$_rc"
assert_contains "lint without a file prints usage on stderr" "Usage:" "$_err"
assert_empty_file "lint without a file prints nothing on stdout" "$_out"
run_pv "$PLAN_VISUAL" lint "$_good" "$_good"
assert_eq "lint with two files exits 1" 1 "$_rc"
assert_contains "lint with two files prints usage on stderr" "Usage:" "$_err"
run_pv "$PLAN_VISUAL" lint "$_tmp/missing.html"
assert_eq "lint of a missing file exits 1" 1 "$_rc"
assert_contains "lint of a missing file names it on stderr" "missing.html" "$_err"
assert_empty_file "lint of a missing file prints no finding" "$_out"
run_pv "$PLAN_VISUAL" lint "$_tmp"
assert_eq "lint of a directory exits 1" 1 "$_rc"

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

# write_ac_plan <file> <line>... — a plan whose acceptance criteria are the
# given lines.
write_ac_plan() {
  _file="$1"
  shift
  {
    printf '%s\n' '# sample-plan' '' '- Status: Draft' '' '## Acceptance criteria' ''
    printf '%s\n' "$@"
    printf '%s\n' '' '## Progress checklist' '' '- [ ] PR created'
  } > "$_file"
}

_tab="$(printf '\t')"
_ac_open="$_tmp/ac-open.md"
write_ac_plan "$_ac_open" '- [ ] AC1: first' '- [ ] AC2: second' '  - [ ] AC2a: nested' "${_tab}- [ ] AC2b: tab-nested"
_ac_open_digest="$("$PLAN_VISUAL" digest "$_ac_open")"
write_ac_plan "$_v" '- [x] AC1: first' '- [X] AC2: second' '  - [x] AC2a: nested' "${_tab}- [X] AC2b: tab-nested"
assert_eq "ticked - [x] / - [X] / indented boxes: same digest as unticked" "$_ac_open_digest" "$("$PLAN_VISUAL" digest "$_v")"

write_ac_plan "$_v" '- [x] AC1: first' '- [ ] AC2: second' '  - [ ] AC2a: nested' "${_tab}- [ ] AC2b: tab-nested"
assert_eq "one ticked box: same digest as unticked" "$_ac_open_digest" "$("$PLAN_VISUAL" digest "$_v")"

write_ac_plan "$_v" '- [x] AC1: first!' '- [ ] AC2: second' '  - [ ] AC2a: nested' "${_tab}- [ ] AC2b: tab-nested"
assert_ne "text after a ticked box still counts" "$_ac_open_digest" "$("$PLAN_VISUAL" digest "$_v")"

write_ac_plan "$_v" '  - [x] AC1: first' '- [ ] AC2: second' '  - [ ] AC2a: nested' "${_tab}- [ ] AC2b: tab-nested"
assert_ne "indentation before a ticked box still counts" "$_ac_open_digest" "$("$PLAN_VISUAL" digest "$_v")"

write_ac_plan "$_v" '- [ ] AC1: first' 'Mid-line text [ ] stays as written.' '- item with [ ] inside'
_mid_open="$("$PLAN_VISUAL" digest "$_v")"
write_ac_plan "$_v" '- [ ] AC1: first' 'Mid-line text [x] stays as written.' '- item with [ ] inside'
assert_ne "an [x] in the middle of a line still counts" "$_mid_open" "$("$PLAN_VISUAL" digest "$_v")"
write_ac_plan "$_v" '- [ ] AC1: first' 'Mid-line text [ ] stays as written.' '- item with [x] inside'
assert_ne "an [x] later in a list item still counts" "$_mid_open" "$("$PLAN_VISUAL" digest "$_v")"

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
