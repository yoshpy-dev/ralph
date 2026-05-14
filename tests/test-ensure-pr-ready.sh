#!/usr/bin/env sh
set -eu

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
ENSURE_READY="${PROJECT_ROOT}/scripts/ensure-pr-ready.sh"

_pass=0
_fail=0
_total=0
_tmp=""

cleanup() {
  [ -n "$_tmp" ] && rm -rf "$_tmp"
}
trap cleanup EXIT HUP INT TERM

assert_eq() {
  _desc="$1"
  _expected="$2"
  _actual="$3"
  _total=$((_total + 1))
  if [ "$_expected" = "$_actual" ]; then
    _pass=$((_pass + 1))
    printf '  PASS: %s\n' "$_desc"
  else
    _fail=$((_fail + 1))
    printf '  FAIL: %s\n    expected: %s\n    actual:   %s\n' "$_desc" "$_expected" "$_actual"
  fi
}

assert_exit() {
  _desc="$1"
  _expected="$2"
  shift 2
  _total=$((_total + 1))
  set +e
  "$@" >/dev/null 2>&1
  _actual="$?"
  set -e
  if [ "$_expected" -eq "$_actual" ]; then
    _pass=$((_pass + 1))
    printf '  PASS: %s\n' "$_desc"
  else
    _fail=$((_fail + 1))
    printf '  FAIL: %s (expected exit %s, got %s)\n' "$_desc" "$_expected" "$_actual"
  fi
}

_tmp="$(mktemp -d)"
mkdir -p "$_tmp/bin"

cat > "$_tmp/bin/gh" <<'GH'
#!/usr/bin/env sh
set -eu

state="${GH_STUB_STATE:?}"
ready_count="${GH_STUB_READY_COUNT:?}"
url="${GH_STUB_URL:-https://github.com/example/repo/pull/123}"

if [ "$1" != "pr" ]; then
  echo "unexpected gh command: $*" >&2
  exit 2
fi

case "$2" in
  view)
    jq_expr=""
    while [ "$#" -gt 0 ]; do
      if [ "$1" = "--jq" ]; then
        jq_expr="$2"
        break
      fi
      shift
    done
    case "$jq_expr" in
      .url) printf '%s\n' "$url" ;;
      .isDraft) cat "$state" ;;
      *) echo "unexpected jq expr: $jq_expr" >&2; exit 2 ;;
    esac
    ;;
  ready)
    count=0
    [ -f "$ready_count" ] && count="$(cat "$ready_count")"
    count=$((count + 1))
    printf '%s\n' "$count" > "$ready_count"
    if [ "${GH_STUB_STICKY_DRAFT:-0}" != "1" ]; then
      printf 'false\n' > "$state"
    fi
    ;;
  *)
    echo "unexpected gh pr command: $2" >&2
    exit 2
    ;;
esac
GH
chmod +x "$_tmp/bin/gh"

export PATH="$_tmp/bin:$PATH"
export GH_STUB_STATE="$_tmp/is-draft"
export GH_STUB_READY_COUNT="$_tmp/ready-count"

printf '==> ensure-pr-ready.sh\n'

printf 'true\n' > "$GH_STUB_STATE"
rm -f "$GH_STUB_READY_COUNT"
assert_exit "draft PR is marked ready" 0 "$ENSURE_READY" "https://github.com/example/repo/pull/123"
assert_eq "draft state flipped" "false" "$(cat "$GH_STUB_STATE")"
assert_eq "gh pr ready called once" "1" "$(cat "$GH_STUB_READY_COUNT")"

printf 'false\n' > "$GH_STUB_STATE"
rm -f "$GH_STUB_READY_COUNT"
assert_exit "ready PR is left unchanged" 0 "$ENSURE_READY" "feat/already-ready"
assert_eq "gh pr ready not called for ready PR" "missing" "$([ -f "$GH_STUB_READY_COUNT" ] && cat "$GH_STUB_READY_COUNT" || printf 'missing')"

printf 'true\n' > "$GH_STUB_STATE"
rm -f "$GH_STUB_READY_COUNT"
export GH_STUB_STICKY_DRAFT=1
assert_exit "still-draft PR fails closed" 1 "$ENSURE_READY" "feat/still-draft"
unset GH_STUB_STICKY_DRAFT
assert_eq "gh pr ready attempted for sticky draft" "1" "$(cat "$GH_STUB_READY_COUNT")"

printf '\nensure-pr-ready tests: %s passed, %s failed, %s total\n' "$_pass" "$_fail" "$_total"
[ "$_fail" -eq 0 ]
