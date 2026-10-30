#!/usr/bin/env sh
set -eu

# plan-visual.sh — helpers for a plan's visual review page (a standalone
# HTML file): open it for the reviewer, screenshot it with headless
# Chrome / Chromium, check its layout, and compute the plan digest recorded
# at approval.
#
# Usage:
#   scripts/plan-visual.sh open <html>
#   scripts/plan-visual.sh shot <html> <png> [--fragment <id>] [--width <px>] [--height <px>] [--scale <n>]
#   scripts/plan-visual.sh lint <html>
#   scripts/plan-visual.sh digest <plan.md>
#   scripts/plan-visual.sh --help
#
# Subcommands:
#   open    Print the absolute path of <html> on stdout, then try to open it
#           with the opener: [plan_visual].opener, else `open` on macOS,
#           else `xdg-open` when it is on PATH. When there is no opener or
#           the opener fails, print a note on stderr and still exit 0, so the
#           caller can hand the printed path to a human.
#   shot    Write a PNG of <html> (optionally scrolled to #<id>) with a
#           headless browser and print the absolute PNG path on stdout.
#           <png> must end in .png and must not be the same file as <html>.
#           Defaults: --width 1200, --height 3000, --scale 1 (all positive
#           integers). The browser's own output is shown only on failure.
#   lint    Check <html> against the page layout rules, reading only its
#           HTML text (no browser): L1 exactly one id="overview", on a
#           <figure class="easy">; L2 1 to 3 easy figures, none inside the
#           fold; L3 no mono class, <code>, path, file name, or snake_case
#           word in an easy figure (its text, class names, and <code> tags;
#           other attribute values such as aria-label are not read; inline
#           tags such as <tspan> and <span> do not split a word, a <tspan>
#           with x or y does); L4 per easy figure at most 5 n-* boxes, an
#           <h2>, and an <svg> with an aria-label; L5 one
#           <section class="before"> outside the fold; L6 one fold
#           (<details class="deep">) holding #detail-overview and every
#           figure that is not easy, and no figure starting inside another
#           figure. Prints nothing when every rule holds; otherwise prints
#           one finding per line on stdout as
#           "<figure id or page>: <rule>: <detail>" and exits 1.
#   digest  Print the first 12 hex characters of the SHA-256 of <plan.md>
#           without the `- Status:`, `- Approved:`, and `- Branch:` lines and
#           without the `## Progress checklist` section (from that exact
#           heading up to the next `## ` heading, or EOF). A checked box at the
#           start of a list item (`- [x]` or `- [X]`, indented or not) is
#           hashed as `- [ ]`, so ticking acceptance criteria or other
#           checklists keeps the digest; an `[x]` anywhere else in a line is
#           hashed as written. Each kept line is hashed followed by a newline,
#           so a plan with or without a final newline gives the same digest.
#
# Settings ([plan_visual] in the git-ignored ralph.local.toml; ralph refuses
# them in ralph.toml, and no environment variable sets them):
#   [plan_visual].opener   command used to open a file (overrides the OS
#                          default); it receives the absolute path as its
#                          only argument. The value "none" disables opening.
#   [plan_visual].browser  absolute path to a Chrome / Chromium executable
#                          (overrides detection). The value "none" means no
#                          browser.
#   An empty value, the default, means the key is not set. open reads only
#   the opener and shot only the browser, once, when it needs it; lint and
#   digest read neither. See plan_visual_setting for how they are read.
#
# Exit codes:
#   0  success (open also exits 0 when nothing could open the file)
#   1  usage error, missing or invalid input, lint findings, failed
#      screenshot, or no SHA-256 tool
#   2  shot only: no Chrome / Chromium browser available

usage() {
  cat <<'USAGE'
Usage:
  scripts/plan-visual.sh open <html>
  scripts/plan-visual.sh shot <html> <png> [--fragment <id>] [--width <px>] [--height <px>] [--scale <n>]
  scripts/plan-visual.sh lint <html>
  scripts/plan-visual.sh digest <plan.md>
  scripts/plan-visual.sh --help
Lint:
  Checks the page layout rules L1-L6 in the HTML text (no browser).
  Prints one finding per line on stdout ("<figure id or page>: <rule>: <detail>") and exits 1 when any rule fails.
Digest:
  Skips the "- Status:", "- Approved:", and "- Branch:" lines and the "## Progress checklist" section.
  Reads a "- [x]" or "- [X]" box at the start of a list item as "- [ ]", so ticking checkboxes keeps the digest.
Settings (ralph.local.toml, read with "ralph config get" in this script's repository):
  [plan_visual].opener   command used to open a file (overrides OS default). The value "none" disables opening.
  [plan_visual].browser  absolute path to a Chrome/Chromium executable (overrides detection). The value "none" means no browser.
USAGE
}

warn() {
  printf 'plan-visual: %s\n' "$*" >&2
}

die() {
  _code="$1"
  shift
  warn "$@"
  exit "$_code"
}

usage_error() {
  usage >&2
  exit 1
}

_log=""
cleanup() {
  if [ -n "$_log" ]; then
    rm -f "$_log"
  fi
}
trap cleanup EXIT
trap 'exit 1' HUP INT TERM

# parent_dir <path> — the directory part of <path> ("." when there is none).
parent_dir() {
  case "$1" in
    */*)
      _parent="${1%/*}"
      printf '%s\n' "${_parent:-/}"
      ;;
    *)
      printf '.\n'
      ;;
  esac
}

# abs_path <path> — absolute form of <path>; its parent directory must exist.
abs_path() {
  _dir="$(parent_dir "$1")"
  _base="${1##*/}"
  _dir="$(CDPATH='' cd -- "$_dir" && pwd)" || return 1
  case "$_dir" in
    /) printf '/%s\n' "$_base" ;;
    *) printf '%s/%s\n' "$_dir" "$_base" ;;
  esac
}

# url_encode_path <abs path> — percent-encode the characters that would
# otherwise end or corrupt the path part of a file:// URL.
url_encode_path() {
  printf '%s\n' "$1" | sed -e 's/%/%25/g' -e 's/ /%20/g' -e 's/#/%23/g' -e 's/?/%3F/g'
}

# plan_visual_setting <opener|browser> — print the value of
# [plan_visual].<key> with ralph_setting of ralph-config.sh, which sits next
# to this script. `ralph config get` runs in the root of the repository this
# script is in (the parent of its directory), not in the caller's working
# directory, so the value is the one of the project the script belongs to
# wherever it is run from (/plan runs it from that root). ralph_setting
# prints the empty default silently when ralph is not on PATH or has no
# `config get`, and after one line on stderr when ralph fails otherwise
# (such as an invalid ralph.local.toml), so an unreadable setting means the
# default.
plan_visual_setting() {
  _scripts_dir="$(parent_dir "$0")"
  _scripts_dir="$(CDPATH='' cd -- "$_scripts_dir" && pwd)"
  # shellcheck source=ralph-config.sh
  . "$_scripts_dir/ralph-config.sh"
  (CDPATH='' cd -- "$_scripts_dir/.." && ralph_setting "plan_visual.$1")
}

# resolve_opener <setting> — print the opener command: <setting>, the value
# of [plan_visual].opener, when it is not empty, else the OS default; return
# 1 when there is none. cmd_open handles the value none before calling this.
resolve_opener() {
  if [ -n "$1" ]; then
    printf '%s\n' "$1"
    return 0
  fi
  if [ "$(uname -s 2>/dev/null || true)" = Darwin ]; then
    printf 'open\n'
    return 0
  fi
  if command -v xdg-open >/dev/null 2>&1; then
    printf 'xdg-open\n'
    return 0
  fi
  return 1
}

# resolve_browser — print the browser executable: [plan_visual].browser when
# it is set, else the first Chrome / Chromium found; or explain on stderr why
# there is none and return 1.
resolve_browser() {
  _browser_setting="$(plan_visual_setting browser)"
  if [ -n "$_browser_setting" ]; then
    if [ "$_browser_setting" = none ]; then
      warn "no browser: [plan_visual].browser is none"
      return 1
    fi
    if [ -f "$_browser_setting" ] && [ -x "$_browser_setting" ]; then
      printf '%s\n' "$_browser_setting"
      return 0
    fi
    warn "no browser: [plan_visual].browser is not an executable file: $_browser_setting"
    return 1
  fi
  for _candidate in \
    "/Applications/Google Chrome.app/Contents/MacOS/Google Chrome" \
    "/Applications/Chromium.app/Contents/MacOS/Chromium"
  do
    if [ -x "$_candidate" ]; then
      printf '%s\n' "$_candidate"
      return 0
    fi
  done
  for _name in google-chrome google-chrome-stable chromium chromium-browser; do
    if _found="$(command -v "$_name" 2>/dev/null)"; then
      printf '%s\n' "$_found"
      return 0
    fi
  done
  warn "no Chrome / Chromium browser found; set [plan_visual].browser in ralph.local.toml to its executable"
  return 1
}

# require_positive_int <option> <value>
require_positive_int() {
  case "$2" in
    ''|0*|*[!0-9]*) die 1 "$1 must be a positive integer: $2" ;;
  esac
}

cmd_open() {
  [ "$#" -eq 1 ] || usage_error
  [ -f "$1" ] || die 1 "file not found: $1"
  _html_abs="$(abs_path "$1")"
  printf '%s\n' "$_html_abs"
  _opener_setting="$(plan_visual_setting opener)"
  if [ "$_opener_setting" = none ]; then
    warn "could not open a browser ([plan_visual].opener is none); open the path above manually"
    return 0
  fi
  if ! _opener="$(resolve_opener "$_opener_setting")"; then
    warn "could not open a browser (no opener found); open the path above manually"
    return 0
  fi
  # The opener's stdout goes to stderr so that stdout carries only the path.
  if ! "$_opener" "$_html_abs" </dev/null >&2; then
    warn "could not open a browser ($_opener failed); open the path above manually"
  fi
  return 0
}

cmd_shot() {
  [ "$#" -ge 2 ] || usage_error
  _html="$1"
  _png="$2"
  shift 2
  [ -n "$_png" ] || usage_error
  _fragment=""
  _width=1200
  _height=3000
  _scale=1
  while [ "$#" -gt 0 ]; do
    case "$1" in
      --fragment|--width|--height|--scale)
        [ "$#" -ge 2 ] || die 1 "$1 needs a value"
        case "$1" in
          --fragment) _fragment="${2#\#}" ;;
          --width) _width="$2" ;;
          --height) _height="$2" ;;
          --scale) _scale="$2" ;;
        esac
        shift 2
        ;;
      *)
        warn "unknown option: $1"
        usage_error
        ;;
    esac
  done
  require_positive_int --width "$_width"
  require_positive_int --height "$_height"
  require_positive_int --scale "$_scale"

  [ -f "$_html" ] || die 1 "file not found: $_html"
  case "$_png" in
    *.[Pp][Nn][Gg]) ;;
    *) die 1 "output path must end in .png: $_png" ;;
  esac
  _png_dir="$(parent_dir "$_png")"
  [ -d "$_png_dir" ] || die 1 "output directory not found: $_png_dir"
  [ ! -d "$_png" ] || die 1 "output path is a directory: $_png"
  _html_abs="$(abs_path "$_html")"
  _png_abs="$(abs_path "$_png")"
  # Writing the PNG over the input would destroy the page.
  [ "$_png_abs" != "$_html_abs" ] || die 1 "output path is the same file as the input html: $_png"

  _browser="$(resolve_browser)" || exit 2

  _url="file://$(url_encode_path "$_html_abs")"
  if [ -n "$_fragment" ]; then
    _url="${_url}#${_fragment}"
  fi

  _log="$(mktemp "${TMPDIR:-/tmp}/plan-visual.XXXXXX")"
  # A PNG left over from an earlier run must not count as this run's output.
  rm -f "$_png_abs"
  set +e
  "$_browser" --headless=new --disable-gpu --hide-scrollbars \
    --force-device-scale-factor="$_scale" \
    --window-size="$_width,$_height" \
    --screenshot="$_png_abs" \
    "$_url" </dev/null >"$_log" 2>&1
  _rc=$?
  set -e

  if [ ! -s "$_png_abs" ]; then
    warn "screenshot failed: $_png_abs was not written (browser exit $_rc); browser output (last 20 lines):"
    tail -n 20 "$_log" >&2
    exit 1
  fi
  printf '%s\n' "$_png_abs"
}

# lint_page <html> — print the findings of the layout rules L1-L6, one per
# line in rule order, from the HTML text alone; the awk exits 1 when there is
# any. The page is read on stdin so that a name like a=b.html is not taken
# for an awk assignment.
lint_page() {
  LC_ALL=C awk '
    # Quotes are written as escapes because this program sits inside a
    # single-quoted shell string.
    BEGIN {
      SQ = "\047"; DQ = "\""
      QUOTE_RE = "[\"\047]"
      ATTR_NAME_RE = "^[^ \t\r\n/>=\"\047]+"
      doc = ""
      NFIG = 0; NEASY = 0; NOPEN = 0; NPEND = 0
      OV_COUNT = 0; NBEFORE = 0; NFOLD = 0; FOLD_DEPTH = 0; DETAIL_IN_FOLD = 0
      n = split("tspan a b i em strong span small sub sup", INLINE_NAMES, " ")
      for (i = 1; i <= n; i++) INLINE[INLINE_NAMES[i]] = 1
    }

    { doc = doc $0 "\n" }

    # add(rule, label, detail) — record a finding once; label "page" for a
    # finding about the whole page, else the figure label.
    function add(rule, label, detail,    line) {
      line = label ": L" rule ": " detail
      if (line in SEEN) return
      SEEN[line] = 1
      if (label == "page") PAGE[rule, ++NPAGE[rule]] = line
      else FIGS[rule, ++NFIGS[rule]] = line
    }

    function class_has(cls, tok,    w, n, i) {
      n = split(cls, w, " ")
      for (i = 1; i <= n; i++) if (w[i] == tok) return 1
      return 0
    }

    function class_has_prefix(cls, pre,    w, n, i) {
      n = split(cls, w, " ")
      for (i = 1; i <= n; i++) if (index(w[i], pre) == 1) return 1
      return 0
    }

    # parse_attrs(t) — read the id, class, and aria-label of a start tag
    # (t is the text after the element name) into A_ID, A_CLASS, A_ARIA, and
    # set HAS_XY when it has an x or y attribute. Values may be
    # double-quoted, single-quoted, or bare; the first of a repeated
    # attribute wins.
    function parse_attrs(t,    name, v, q, k) {
      A_ID = ""; A_CLASS = ""; A_ARIA = ""; HAS_ID = 0; HAS_CLASS = 0; HAS_ARIA = 0
      HAS_XY = 0
      while (match(t, /[^ \t\r\n\/]/)) {
        t = substr(t, RSTART)
        if (!match(t, ATTR_NAME_RE)) { t = substr(t, 2); continue }
        name = substr(t, 1, RLENGTH)
        t = substr(t, RLENGTH + 1)
        v = ""
        if (match(t, /^[ \t\r\n]*=[ \t\r\n]*/)) {
          t = substr(t, RLENGTH + 1)
          q = substr(t, 1, 1)
          if (q == DQ || q == SQ) {
            k = index(substr(t, 2), q)
            if (k == 0) { v = substr(t, 2); t = "" }
            else { v = substr(t, 2, k - 1); t = substr(t, k + 2) }
          } else {
            match(t, /^[^ \t\r\n>]*/)
            v = substr(t, 1, RLENGTH)
            t = substr(t, RLENGTH + 1)
          }
        }
        if (name == "id" && !HAS_ID) { A_ID = v; HAS_ID = 1 }
        else if (name == "class" && !HAS_CLASS) { A_CLASS = v; HAS_CLASS = 1 }
        else if (name == "aria-label" && !HAS_ARIA) { A_ARIA = v; HAS_ARIA = 1 }
        else if (name == "x" || name == "y") HAS_XY = 1
      }
    }

    # tag_end(s) — position of the ">" that ends the start tag at the head
    # of s, skipping any ">" inside a quoted value; 0 when there is none. An
    # unclosed quote falls back to the first ">".
    function tag_end(s,    e, k, j, q, n) {
      e = index(s, ">")
      if (e == 0) return 0
      k = 1
      while (match(substr(s, k + 1, e - k), QUOTE_RE)) {
        k += RSTART
        q = substr(s, k, 1)
        j = index(substr(s, k + 1, e - k), q)
        if (j == 0) j = index(substr(s, k + 1), q)
        if (j == 0) return index(s, ">")
        k += j
        if (k > e) {
          n = index(substr(s, k + 1), ">")
          if (n == 0) return 0
          e = k + n
        }
      }
      return e
    }

    # Every tag that splits a word (see tag_splits), comment, and skipped
    # style or script body is a space in the text of the easy figures that
    # are open.
    function tag_space(    k) {
      for (k = 1; k <= NOPEN; k++) TEXT[OPEN[k]] = TEXT[OPEN[k]] " "
    }

    # tag_splits(name, start) — whether a start tag (start set, after
    # parse_attrs) or an end tag splits the words on its two sides. An inline
    # tag does not: internal/<tspan>org</tspan> reads as internal/org. A tspan
    # start tag with an x or y attribute starts a new line, so it does.
    function tag_splits(name, start) {
      if (!(name in INLINE)) return 1
      return (start && name == "tspan" && HAS_XY)
    }

    function text(t,    k) {
      for (k = 1; k <= NOPEN; k++) TEXT[OPEN[k]] = TEXT[OPEN[k]] t
    }

    # close_figures() — check and close every figure still open.
    function close_figures(    k) {
      for (k = 1; k <= NOPEN; k++) check_easy(OPEN[k])
      NOPEN = 0; NPEND = 0
    }

    function start_tag(name, attrs,    infold, f, k, deep) {
      parse_attrs(attrs)
      if (tag_splits(name, 1)) tag_space()
      infold = (FOLD_DEPTH > 0)
      if (HAS_ID && A_ID == "overview") {
        OV_COUNT++
        if (OV_COUNT == 1) { OV_NAME = name; OV_EASY = class_has(A_CLASS, "easy") }
      }
      if (name == "figure") {
        f = ++NFIG
        LABEL[f] = (A_ID != "") ? A_ID : ("figure#" f)
        # A figure that starts while another is open means a missing
        # </figure>: report it and close the open ones here, so that their
        # text and boxes do not run on into this figure.
        if (NPEND > 0) {
          add(6, LABEL[f], "figure starts inside figure " LABEL[PEND[NPEND]] " (missing </figure>?)")
          close_figures()
        }
        PEND[++NPEND] = f
        if (A_ID == "detail-overview" && infold) DETAIL_IN_FOLD = 1
        if (class_has(A_CLASS, "easy")) {
          NEASY++
          OPEN[++NOPEN] = f
          TEXT[f] = ""; BOXES[f] = 0; H2[f] = 0; SVG[f] = 0
          if (infold) add(2, LABEL[f], "easy figure inside the fold")
        } else if (!infold) {
          add(6, LABEL[f], "figure without class easy is outside the fold")
        }
      }
      if (name == "section" && class_has(A_CLASS, "before")) {
        NBEFORE++
        if (infold) add(5, "page", "<section class=\"before\"> is inside the fold")
      }
      if (name == "details") {
        deep = class_has(A_CLASS, "deep")
        if (deep) NFOLD++
        if (deep || FOLD_DEPTH > 0) FOLD_DEPTH++
      }
      for (k = 1; k <= NOPEN; k++) {
        f = OPEN[k]
        if (class_has(A_CLASS, "mono")) add(3, LABEL[f], "class mono on <" name ">")
        if (name == "code") add(3, LABEL[f], "<code> element")
        if (class_has_prefix(A_CLASS, "n-")) BOXES[f]++
        if (name == "h2") H2[f] = 1
        if (name == "svg" && A_ARIA ~ /[^ \t\r\n]/) SVG[f] = 1
      }
    }

    # A </figure> ends every figure still open (figures do not nest: see
    # start_tag).
    function end_tag(name) {
      if (tag_splits(name, 0)) tag_space()
      if (name == "figure") close_figures()
      if (name == "details" && FOLD_DEPTH > 0) FOLD_DEPTH--
    }

    # widen(s, a, b, cls, right) — s[a..b] grown left (and right when right
    # is set) over characters matching cls; sets W_END to the new end. It
    # must not call match: its callers read RSTART and RLENGTH after it.
    function widen(s, a, b, cls, right,    n) {
      while (a > 1 && substr(s, a - 1, 1) ~ cls) a--
      if (right) {
        n = length(s)
        while (b < n && substr(s, b + 1, 1) ~ cls) b++
      }
      W_END = b
      return substr(s, a, b - a + 1)
    }

    # scan_text(label, s) — L3 on the text of an easy figure. s ends with a
    # space, so a file extension at the end of the text is still followed by
    # a character.
    function scan_text(label, s,    t, tok) {
      t = s
      while (match(t, /[A-Za-z_.][A-Za-z0-9_.-]*\/[A-Za-z0-9_.-]/)) {
        tok = widen(t, RSTART, RSTART + RLENGTH - 1, "[A-Za-z0-9_./-]", 1)
        t = substr(t, W_END + 1)
        sub(/\.+$/, "", tok)
        add(3, label, "path-like text: " tok)
      }
      t = s
      # The ";" in [A-Za-z0-9_;-] lets &gt;.html (from &lt;slug&gt;.html) be caught.
      while (match(t, /[A-Za-z0-9_;-]\.(go|sh|md|json|jsonl|toml|html|yaml|yml|py|ts|js|awk)[^A-Za-z0-9]/)) {
        tok = widen(t, RSTART, RSTART + RLENGTH - 2, "[A-Za-z0-9_.;&/-]", 0)
        t = substr(t, RSTART + RLENGTH - 1)
        add(3, label, "file extension: " tok)
      }
      t = s
      while (match(t, /[a-z][a-z0-9]*_[a-z0-9_]*[a-z0-9]/)) {
        tok = widen(t, RSTART, RSTART + RLENGTH - 1, "[A-Za-z0-9_]", 1)
        t = substr(t, W_END + 1)
        add(3, label, "snake_case identifier: " tok)
      }
    }

    function check_easy(f) {
      scan_text(LABEL[f], TEXT[f] " ")
      if (BOXES[f] > 5) add(4, LABEL[f], BOXES[f] " boxes (class n-*), at most 5")
      if (!H2[f]) add(4, LABEL[f], "no <h2> heading")
      if (!SVG[f]) add(4, LABEL[f], "no <svg> with a non-empty aria-label")
    }

    function finish(    k, r, n) {
      for (k = 1; k <= NOPEN; k++) check_easy(OPEN[k])
      for (k = 1; k <= NPEND; k++) add(6, LABEL[PEND[k]], "no </figure> after this figure")
      if (FOLD_DEPTH > 0) add(6, "page", "a fold has no matching </details>")
      if (OV_COUNT == 0) add(1, "page", "no element has id=\"overview\"")
      else if (OV_COUNT > 1) add(1, "page", OV_COUNT " elements have id=\"overview\", expected 1")
      else if (OV_NAME != "figure") add(1, "overview", "id=\"overview\" is on <" OV_NAME ">, not on <figure class=\"easy\">")
      else if (!OV_EASY) add(1, "overview", "the id=\"overview\" figure lacks class easy")
      if (NEASY < 1 || NEASY > 3) add(2, "page", NEASY " easy figures (<figure class=\"easy\">), expected 1 to 3")
      if (NBEFORE == 0) add(5, "page", "no <section class=\"before\">")
      else if (NBEFORE > 1) add(5, "page", NBEFORE " <section class=\"before\"> elements, expected 1")
      if (NFOLD == 0) add(6, "page", "no fold (<details class=\"deep\">)")
      else if (NFOLD > 1) add(6, "page", NFOLD " folds (<details class=\"deep\">), expected 1")
      if (!DETAIL_IN_FOLD) add(6, "page", "no <figure id=\"detail-overview\"> inside the fold")
      n = 0
      for (r = 1; r <= 6; r++) {
        for (k = 1; k <= NPAGE[r] + 0; k++) { print PAGE[r, k]; n++ }
        for (k = 1; k <= NFIGS[r] + 0; k++) { print FIGS[r, k]; n++ }
      }
      return n
    }

    # Cut the document into text, comments, declarations, end tags, and
    # start tags. A "<" that starts none of them is text.
    END {
      rest = doc
      while (rest != "") {
        i = index(rest, "<")
        if (i == 0) { text(rest); break }
        if (i > 1) { text(substr(rest, 1, i - 1)); rest = substr(rest, i) }
        c = substr(rest, 2, 1)
        if (substr(rest, 1, 4) == "<!--") {
          tag_space()
          j = index(substr(rest, 5), "-->")
          if (j == 0) break
          rest = substr(rest, j + 7)
        } else if (c ~ /[A-Za-z]/) {
          e = tag_end(rest)
          if (e == 0) { body = substr(rest, 2); rest = "" }
          else { body = substr(rest, 2, e - 2); rest = substr(rest, e + 1) }
          match(body, /^[A-Za-z][A-Za-z0-9-]*/)
          name = substr(body, 1, RLENGTH)
          start_tag(name, substr(body, RLENGTH + 1))
          # The body of a style or script element is not page text.
          if (name == "style" || name == "script") {
            j = index(rest, "</" name)
            if (j == 0) j = length(rest) + 1
            if (j > 1) { tag_space(); rest = substr(rest, j) }
          }
        } else if (c == "/" && substr(rest, 3, 1) ~ /[A-Za-z]/) {
          e = index(rest, ">")
          if (e == 0) { body = substr(rest, 3); rest = "" }
          else { body = substr(rest, 3, e - 3); rest = substr(rest, e + 1) }
          match(body, /^[A-Za-z][A-Za-z0-9-]*/)
          end_tag(substr(body, 1, RLENGTH))
        } else if (c == "!" || c == "?") {
          tag_space()
          e = index(rest, ">")
          if (e == 0) break
          rest = substr(rest, e + 1)
        } else {
          text("<")
          rest = substr(rest, 2)
        }
      }
      exit (finish() > 0)
    }
  ' < "$1"
}

cmd_lint() {
  [ "$#" -eq 1 ] || usage_error
  if [ ! -f "$1" ] || [ ! -r "$1" ]; then
    die 1 "html file not found or not readable: $1"
  fi
  set +e
  lint_page "$1"
  _rc=$?
  set -e
  case "$_rc" in
    0|1) exit "$_rc" ;;
    *) die 1 "lint failed: awk exited $_rc" ;;
  esac
}

# digest_body <plan.md> — the bytes the approval digest covers.
digest_body() {
  LC_ALL=C awk '
    /^## / { skip = ($0 == "## Progress checklist"); if (skip) next }
    skip { next }
    /^- (Status|Approved|Branch):/ { next }
    /^[[:space:]]*- \[[xX]\]/ {
      match($0, /^[[:space:]]*- \[/)
      $0 = substr($0, 1, RLENGTH) " " substr($0, RLENGTH + 2)
    }
    { print }
  ' "$1"
}

cmd_digest() {
  [ "$#" -eq 1 ] || usage_error
  if [ ! -f "$1" ] || [ ! -r "$1" ]; then
    die 1 "plan file not found or not readable: $1"
  fi
  if command -v sha256sum >/dev/null 2>&1; then
    _sum="$(digest_body "$1" | sha256sum)"
  elif command -v shasum >/dev/null 2>&1; then
    _sum="$(digest_body "$1" | shasum -a 256)"
  else
    die 1 "neither sha256sum nor shasum is available"
  fi
  _digest="$(printf '%.12s' "$_sum")"
  case "$_digest" in
    ????????????) ;;
    *) die 1 "unexpected SHA-256 output: $_sum" ;;
  esac
  case "$_digest" in
    *[!0-9a-f]*) die 1 "unexpected SHA-256 output: $_sum" ;;
  esac
  printf '%s\n' "$_digest"
}

case "${1:-}" in
  open)
    shift
    cmd_open "$@"
    ;;
  shot)
    shift
    cmd_shot "$@"
    ;;
  lint)
    shift
    cmd_lint "$@"
    ;;
  digest)
    shift
    cmd_digest "$@"
    ;;
  -h|--help|help)
    usage
    ;;
  *)
    usage_error
    ;;
esac
