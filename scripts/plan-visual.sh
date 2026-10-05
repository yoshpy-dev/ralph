#!/usr/bin/env sh
set -eu

# plan-visual.sh — helpers for a plan's visual review page (a standalone
# HTML file): open it for the reviewer, screenshot it with headless
# Chrome / Chromium, and compute the plan digest recorded at approval.
#
# Usage:
#   scripts/plan-visual.sh open <html>
#   scripts/plan-visual.sh shot <html> <png> [--fragment <id>] [--width <px>] [--height <px>] [--scale <n>]
#   scripts/plan-visual.sh digest <plan.md>
#   scripts/plan-visual.sh --help
#
# Subcommands:
#   open    Print the absolute path of <html> on stdout, then try to open it
#           with the opener: RALPH_PLAN_VISUAL_OPENER, else `open` on macOS,
#           else `xdg-open` when it is on PATH. When there is no opener or
#           the opener fails, print a note on stderr and still exit 0, so the
#           caller can hand the printed path to a human.
#   shot    Write a PNG of <html> (optionally scrolled to #<id>) with a
#           headless browser and print the absolute PNG path on stdout.
#           Defaults: --width 1200, --height 3000, --scale 1 (all positive
#           integers). The browser's own output is shown only on failure.
#   digest  Print the first 12 hex characters of the SHA-256 of <plan.md>
#           without the `- Status:`, `- Approved:`, and `- Branch:` lines and
#           without the `## Progress checklist` section (from that exact
#           heading up to the next `## ` heading, or EOF). Each kept line is
#           hashed followed by a newline, so the value matches across
#           sha256sum and shasum.
#
# Environment:
#   RALPH_PLAN_VISUAL_OPENER   command used to open a file (overrides the OS
#                              default); it receives the absolute path as its
#                              only argument. The value "none" disables opening.
#   RALPH_PLAN_VISUAL_BROWSER  path to a Chrome / Chromium executable
#                              (overrides detection). The value "none" means
#                              no browser.
#
# Exit codes:
#   0  success (open also exits 0 when nothing could open the file)
#   1  usage error, missing or invalid input, failed screenshot, or no
#      SHA-256 tool
#   2  shot only: no Chrome / Chromium browser available

usage() {
  cat <<'USAGE'
Usage:
  scripts/plan-visual.sh open <html>
  scripts/plan-visual.sh shot <html> <png> [--fragment <id>] [--width <px>] [--height <px>] [--scale <n>]
  scripts/plan-visual.sh digest <plan.md>
  scripts/plan-visual.sh --help
Environment:
  RALPH_PLAN_VISUAL_OPENER   command used to open a file (overrides OS default). The value "none" disables opening.
  RALPH_PLAN_VISUAL_BROWSER  path to a Chrome/Chromium executable (overrides detection). The value "none" means no browser.
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

resolve_opener() {
  if [ -n "${RALPH_PLAN_VISUAL_OPENER:-}" ]; then
    if [ "$RALPH_PLAN_VISUAL_OPENER" = none ]; then
      return 1
    fi
    printf '%s\n' "$RALPH_PLAN_VISUAL_OPENER"
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

# resolve_browser — print the browser executable, or explain on stderr why
# there is none and return 1.
resolve_browser() {
  if [ -n "${RALPH_PLAN_VISUAL_BROWSER:-}" ]; then
    if [ "$RALPH_PLAN_VISUAL_BROWSER" = none ]; then
      warn "no browser: RALPH_PLAN_VISUAL_BROWSER=none"
      return 1
    fi
    if [ -f "$RALPH_PLAN_VISUAL_BROWSER" ] && [ -x "$RALPH_PLAN_VISUAL_BROWSER" ]; then
      printf '%s\n' "$RALPH_PLAN_VISUAL_BROWSER"
      return 0
    fi
    warn "no browser: RALPH_PLAN_VISUAL_BROWSER is not an executable file: $RALPH_PLAN_VISUAL_BROWSER"
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
  warn "no Chrome / Chromium browser found; set RALPH_PLAN_VISUAL_BROWSER to its executable"
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
  if [ "${RALPH_PLAN_VISUAL_OPENER:-}" = none ]; then
    warn "could not open a browser (RALPH_PLAN_VISUAL_OPENER=none); open the path above manually"
    return 0
  fi
  if ! _opener="$(resolve_opener)"; then
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
  _png_dir="$(parent_dir "$_png")"
  [ -d "$_png_dir" ] || die 1 "output directory not found: $_png_dir"
  [ ! -d "$_png" ] || die 1 "output path is a directory: $_png"
  _html_abs="$(abs_path "$_html")"
  _png_abs="$(abs_path "$_png")"

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

# digest_body <plan.md> — the bytes the approval digest covers.
digest_body() {
  LC_ALL=C awk '
    /^## / { skip = ($0 == "## Progress checklist"); if (skip) next }
    skip { next }
    /^- (Status|Approved|Branch):/ { next }
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
