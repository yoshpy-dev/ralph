#!/usr/bin/env sh
set -eu

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
ENSURE_TITLE="${PROJECT_ROOT}/scripts/ensure-pr-title-prefix.sh"

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

title_file="${GH_STUB_TITLE:?}"
head_file="${GH_STUB_HEAD:?}"
edit_count="${GH_STUB_EDIT_COUNT:?}"

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
      .title) cat "$title_file" ;;
      .headRefName) cat "$head_file" ;;
      *) echo "unexpected jq expr: $jq_expr" >&2; exit 2 ;;
    esac
    ;;
  edit)
    new_title=""
    while [ "$#" -gt 0 ]; do
      if [ "$1" = "--title" ]; then
        new_title="$2"
        break
      fi
      shift
    done
    [ -n "$new_title" ] || { echo "missing --title" >&2; exit 2; }
    count=0
    [ -f "$edit_count" ] && count="$(cat "$edit_count")"
    count=$((count + 1))
    printf '%s\n' "$count" > "$edit_count"
    if [ "${GH_STUB_STICKY_TITLE:-0}" != "1" ]; then
      printf '%s\n' "$new_title" > "$title_file"
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
export GH_STUB_TITLE="$_tmp/title"
export GH_STUB_HEAD="$_tmp/head"
export GH_STUB_EDIT_COUNT="$_tmp/edit-count"

printf '==> ensure-pr-title-prefix.sh\n'

printf 'feat/title-prefix-policy\n' > "$GH_STUB_HEAD"
printf 'feat: タイトルprefixを追加\n' > "$GH_STUB_TITLE"
rm -f "$GH_STUB_EDIT_COUNT"
assert_exit "matching prefix is left unchanged" 0 "$ENSURE_TITLE" "https://github.com/example/repo/pull/123"
assert_eq "matching prefix does not edit" "missing" "$([ -f "$GH_STUB_EDIT_COUNT" ] && cat "$GH_STUB_EDIT_COUNT" || printf 'missing')"

printf 'docs/title-prefix-policy\n' > "$GH_STUB_HEAD"
printf 'READMEを更新\n' > "$GH_STUB_TITLE"
rm -f "$GH_STUB_EDIT_COUNT"
assert_exit "missing prefix is added" 0 "$ENSURE_TITLE" "docs/title-prefix-policy"
assert_eq "missing prefix title" "docs: READMEを更新" "$(cat "$GH_STUB_TITLE")"
assert_eq "missing prefix edit count" "1" "$(cat "$GH_STUB_EDIT_COUNT")"

printf 'fix/74/title-prefix-policy\n' > "$GH_STUB_HEAD"
printf 'feat: 間違ったprefix\n' > "$GH_STUB_TITLE"
rm -f "$GH_STUB_EDIT_COUNT"
assert_exit "wrong prefix is replaced" 0 "$ENSURE_TITLE" "fix/74/title-prefix-policy"
assert_eq "wrong prefix title" "fix: 間違ったprefix" "$(cat "$GH_STUB_TITLE")"
assert_eq "wrong prefix edit count" "1" "$(cat "$GH_STUB_EDIT_COUNT")"

printf 'fix/title-prefix-spacing\n' > "$GH_STUB_HEAD"
printf 'fix:スペースなし\n' > "$GH_STUB_TITLE"
rm -f "$GH_STUB_EDIT_COUNT"
assert_exit "matching prefix without space is normalized" 0 "$ENSURE_TITLE" "fix/title-prefix-spacing"
assert_eq "normalized title" "fix: スペースなし" "$(cat "$GH_STUB_TITLE")"

printf 'codex-title-prefix-policy\n' > "$GH_STUB_HEAD"
printf 'タイトル\n' > "$GH_STUB_TITLE"
rm -f "$GH_STUB_EDIT_COUNT"
assert_exit "invalid branch fails closed" 1 "$ENSURE_TITLE" "codex-title-prefix-policy"

printf 'chore/sticky-title\n' > "$GH_STUB_HEAD"
printf 'タイトル\n' > "$GH_STUB_TITLE"
rm -f "$GH_STUB_EDIT_COUNT"
export GH_STUB_STICKY_TITLE=1
assert_exit "sticky title fails closed" 1 "$ENSURE_TITLE" "chore/sticky-title"
unset GH_STUB_STICKY_TITLE
assert_eq "sticky title edit attempted" "1" "$(cat "$GH_STUB_EDIT_COUNT")"

printf '\nensure-pr-title-prefix tests: %s passed, %s failed, %s total\n' "$_pass" "$_fail" "$_total"
[ "$_fail" -eq 0 ]
