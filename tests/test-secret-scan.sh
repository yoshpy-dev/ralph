#!/usr/bin/env sh
set -eu

SCRIPT_DIR="$(CDPATH='' cd -- "$(dirname -- "$0")" && pwd)"
REPO_ROOT="$(CDPATH='' cd -- "$SCRIPT_DIR/.." && pwd)"
SCANNER="$REPO_ROOT/scripts/secret-scan.sh"

pass=0
fail=0

ok() {
  pass=$((pass + 1))
  printf '  PASS: %s\n' "$1"
}

not_ok() {
  fail=$((fail + 1))
  printf '  FAIL: %s\n' "$1"
}

expect_exit() {
  name=$1
  want=$2
  shift 2
  set +e
  "$@" >/tmp/ralph-secret-scan-test.out 2>/tmp/ralph-secret-scan-test.err
  got=$?
  set -e
  if [ "$got" -eq "$want" ]; then
    ok "$name"
  else
    not_ok "$name (exit $got, want $want)"
    printf '%s\n' "--- stdout ---"
    cat /tmp/ralph-secret-scan-test.out
    printf '%s\n' "--- stderr ---"
    cat /tmp/ralph-secret-scan-test.err
  fi
}

workdir="$(mktemp -d "${TMPDIR:-/tmp}/ralph-secret-scan.XXXXXX")"
trap 'rm -rf "$workdir" /tmp/ralph-secret-scan-test.out /tmp/ralph-secret-scan-test.err' EXIT HUP INT TERM

cd "$workdir"
git init -q
git config user.email test@example.com
git config user.name "Secret Scan Test"

printf 'hello\n' > clean.txt
git add clean.txt
expect_exit "clean staged file exits 0" 0 "$SCANNER" --staged

value="$(printf 'sk-proj-%s' 'abcdefghijklmnopqrstuvwxyz123456')"
printf 'OPENAI_API_KEY=%s\n' "$value" > secret.env
git add secret.env
expect_exit "staged OpenAI key exits 1" 1 "$SCANNER" --staged

printf '%s\n' "$value" > .gitallowed
expect_exit ".gitallowed suppresses exact false positive" 0 "$SCANNER" --staged

rm .gitallowed
value="$(printf 'ghp_%s' 'abcdefghijklmnopqrstuvwxyzABCDE')"
printf 'deploy token %s\n' "$value" > msg.txt
expect_exit "commit message token exits 1" 1 "$SCANNER" --file msg.txt "commit message"

diff_file="$workdir/synthetic.diff"
value="$(printf 'ABCDEFGHIJKLMNOPQRSTUVWXYZ%s' 'abcdef123456')"
printf 'diff --git a/a b/a\n--- a/a\n+++ b/a\n+aws_secret_access_key = %s\n' "$value" > "$diff_file"
expect_exit "diff added secret exits 1" 1 sh -c "'$SCANNER' --diff 'synthetic diff' < '$diff_file'"

range_dir="$workdir/range"
mkdir "$range_dir"
cd "$range_dir"
git init -q
git config user.email test@example.com
git config user.name "Secret Scan Test"
printf 'clean\n' > README.md
git add README.md
git commit -q -m 'initial commit'
base_branch="$(git branch --show-current)"
git checkout -q -b leak
value="$(printf 'sk_live_%s' 'abcdefghijklmnopqrstuv')"
printf 'stripe_secret=%s\n' "$value" > leak.txt
git add leak.txt
git commit -q -m 'add leaked secret'
expect_exit "range added secret exits 1" 1 "$SCANNER" --range "$base_branch..leak"

printf '\n-- Summary --\n  PASS: %s\n  FAIL: %s\n' "$pass" "$fail"
[ "$fail" -eq 0 ]
