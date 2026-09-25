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

# ---------------------------------------------------------------------------
# Hermetic section: CI scans a range with git's default config, so local git
# config and repo state must not change which added lines --range reads, and
# a git failure must not read as a clean scan. From here on every repo runs
# under an isolated HOME and global config (the cases above run under the
# caller's own config). Fixture values are assembled at runtime from split
# pieces: this file's own history is read by the same range scan.
# ---------------------------------------------------------------------------
hermetic_home="$workdir/.home"
mkdir -p "$hermetic_home"
HOME="$hermetic_home"
GIT_CONFIG_GLOBAL="$hermetic_home/.gitconfig"
GIT_CONFIG_SYSTEM=/dev/null
GIT_TERMINAL_PROMPT=0
export HOME GIT_CONFIG_GLOBAL GIT_CONFIG_SYSTEM GIT_TERMINAL_PROMPT
unset RALPH_SECRET_ALLOWLIST
git config --global user.email test@example.com
git config --global user.name "Secret Scan Test"
git config --global commit.gpgsign false

token="$(printf 'sk_live_%s' 'abcdefghijklmnopqrstuv')"

expect_stderr_contains() {
  name=$1
  needle=$2
  if grep -F -- "$needle" /tmp/ralph-secret-scan-test.err >/dev/null 2>&1; then
    ok "$name"
  else
    not_ok "$name (missing in stderr: $needle)"
    cat /tmp/ralph-secret-scan-test.err
  fi
}

expect_stderr_not_contains() {
  name=$1
  needle=$2
  if grep -F -- "$needle" /tmp/ralph-secret-scan-test.err >/dev/null 2>&1; then
    not_ok "$name (unexpectedly in stderr: $needle)"
    cat /tmp/ralph-secret-scan-test.err
  else
    ok "$name"
  fi
}

# new_repo <dir> -- a repo whose main branch has one clean commit.
new_repo() {
  mkdir -p "$1"
  (
    cd "$1"
    git init -q
    git checkout -q -B main
    printf 'clean\n' > README.md
    git add README.md
    git commit -q -m init
  )
}

# One branch adding a token at the repo root, scanned under one local
# setting at a time.
repo="$workdir/cfg-leak"
new_repo "$repo"
cd "$repo"
mkdir sub
printf 'placeholder\n' > sub/placeholder.txt
git add sub/placeholder.txt
git commit -q -m 'add sub'
git checkout -q -b feature
printf 'deploy token %s\n' "$token" > leak.txt
git add leak.txt
git commit -q -m 'add leaked token'

git config color.ui always
expect_exit "AC-1: range scan with color.ui=always finds the token" 1 "$SCANNER" --range main..feature
git config --unset color.ui
git config color.diff always
expect_exit "AC-1: range scan with color.diff=always finds the token" 1 "$SCANNER" --range main..feature
git config --unset color.diff

git config diff.relative true
cd sub
expect_exit "AC-2: range scan with diff.relative=true from a subdirectory finds a token outside it" 1 "$SCANNER" --range main..feature
cd ..
git config --unset diff.relative

git config core.bigFileThreshold 1
expect_exit "AC-16: range scan with core.bigFileThreshold=1 still reads the text file" 1 "$SCANNER" --range main..feature
git config --unset core.bigFileThreshold

printf '*.txt -diff\n' > "$workdir/user-attributes"
git config core.attributesFile "$workdir/user-attributes"
expect_exit "range scan ignores a user-level attributes file marking the file -diff" 1 "$SCANNER" --range main..feature
git config --unset core.attributesFile

git config diff.orderFile "$workdir/does-not-exist.order"
expect_exit "AC-15: git log failing before any output (missing diff.orderFile) exits 3" 3 "$SCANNER" --range main..feature
expect_stderr_contains "AC-15: missing diff.orderFile names the unscanned range" "could not scan main..feature"
expect_stderr_not_contains "AC-15: missing diff.orderFile is not reported as findings" "BLOCKED"
git config --unset diff.orderFile

expect_exit "AC-15: a range that does not resolve exits 3" 3 "$SCANNER" --range does-not-exist..feature
expect_stderr_contains "AC-15: unresolved range names the unscanned range" "could not scan does-not-exist..feature"

# A local textconv for a diff driver named in the committed .gitattributes
# would rewrite the token before the scan reads it.
repo="$workdir/cfg-textconv"
new_repo "$repo"
cd "$repo"
printf '*.txt diff=scrub\n' > .gitattributes
git add .gitattributes
git commit -q -m 'add diff driver attribute'
git checkout -q -b feature
printf 'deploy token %s\n' "$token" > leak.txt
git add leak.txt
git commit -q -m 'add leaked token'
git config diff.scrub.textconv cksum
expect_exit "AC-3: range scan ignores a local textconv for a committed diff driver" 1 "$SCANNER" --range main..feature

# Copy detection would show a copied file as "copy from/copy to" with no
# added lines; rename detection stays on, as in CI, so a pure rename adds
# nothing to scan under any diff.renames setting.
repo="$workdir/cfg-renames"
new_repo "$repo"
cd "$repo"
printf 'deploy token %s\nline two\nline three\n' "$token" > base.txt
git add base.txt
git commit -q -m 'add base file'
git checkout -q -b copy
cp base.txt copy.txt
printf 'line four\n' >> base.txt
git add base.txt copy.txt
git commit -q -m 'copy base file'
git config diff.renames copies
expect_exit "AC-4: range scan with diff.renames=copies finds the token in a copied file" 1 "$SCANNER" --range main..copy
git config --unset diff.renames
git checkout -q -b rename main
git mv base.txt renamed.txt
git commit -q -m 'rename base file'
expect_exit "AC-4: pure rename under the default config adds nothing to scan" 0 "$SCANNER" --range main..rename
for renames in copies false; do
  git config diff.renames "$renames"
  expect_exit "AC-4: pure rename with diff.renames=$renames matches the default result" 0 "$SCANNER" --range main..rename
  git config --unset diff.renames
done

# The token line moves from the end of the file to its start past a run of
# repeated lines. The default (myers) diff reports it as added; histogram
# and patience keep it as a unique common line and report the repeated
# lines as added instead.
repo="$workdir/cfg-algorithm"
new_repo "$repo"
cd "$repo"
printf 'anchor\nsame\nsame\nsame\nsame\ndeploy token %s\n' "$token" > moved.txt
git add moved.txt
git commit -q -m 'add moved file'
git checkout -q -b feature
printf 'deploy token %s\nsame\nsame\nsame\nsame\nanchor\n' "$token" > moved.txt
git add moved.txt
git commit -q -m 'move the token line'
expect_exit "AC-5: moved token line under the default diff algorithm exits 1" 1 "$SCANNER" --range main..feature
for algorithm in histogram patience; do
  git config diff.algorithm "$algorithm"
  expect_exit "AC-5: moved token line with diff.algorithm=$algorithm matches the default result" 1 "$SCANNER" --range main..feature
  git config --unset diff.algorithm
done

# The newest commit is clean and the older one's added blob is missing:
# git log prints the newest commit, then fails.
repo="$workdir/cfg-midway"
new_repo "$repo"
cd "$repo"
git checkout -q -b feature
printf 'deploy token %s\n' "$token" > old.txt
git add old.txt
git commit -q -m 'older commit'
printf 'still clean\n' > new.txt
git add new.txt
git commit -q -m 'newest commit'
blob="$(git rev-parse feature~1:old.txt)"
rm -f ".git/objects/$(printf '%s' "$blob" | cut -c1-2)/$(printf '%s' "$blob" | cut -c3-)"
expect_exit "AC-15: git log failing partway through the range exits 3" 3 "$SCANNER" --range main..feature
expect_stderr_contains "AC-15: partway failure names the unscanned range" "could not scan main..feature"

# log.showRoot=false hides the diff of a root commit, here an unrelated
# history merged into the branch.
repo="$workdir/cfg-root"
new_repo "$repo"
cd "$repo"
git checkout -q --orphan other
git rm -q -r -f .
printf 'deploy token %s\n' "$token" > root.txt
git add root.txt
git commit -q -m 'unrelated root commit'
git checkout -q -b feature main
git merge -q --allow-unrelated-histories -m 'merge unrelated history' other
git config log.showRoot false
expect_exit "range scan with log.showRoot=false still reads a root commit's diff" 1 "$SCANNER" --range main..feature

# A local replace ref substitutes a clean commit for the leaking one; CI's
# clone has no refs/replace/*.
repo="$workdir/cfg-replace"
new_repo "$repo"
cd "$repo"
git checkout -q -b feature
printf 'deploy token %s\n' "$token" > leak.txt
git add leak.txt
git commit -q -m 'add leaked token'
stand_in="$(git commit-tree -p main -m 'clean stand-in' "$(git rev-parse 'main^{tree}')")"
git replace feature "$stand_in"
expect_exit "range scan ignores a local replace ref for the leaking commit" 1 "$SCANNER" --range main..feature

# diff.submodule=diff would inline the submodule's own history, which CI
# never reads.
sub_src="$workdir/cfg-submodule-src"
new_repo "$sub_src"
repo="$workdir/cfg-submodule"
new_repo "$repo"
cd "$repo"
git -c protocol.file.allow=always submodule --quiet add "$sub_src" sub
git commit -q -m 'add submodule'
git checkout -q -b feature
(
  cd sub
  printf 'deploy token %s\n' "$token" > leak.txt
  git add leak.txt
  git commit -q -m 'add leaked token inside the submodule'
)
git add sub
git commit -q -m 'bump submodule'
expect_exit "submodule bump under the default config reads no submodule content" 0 "$SCANNER" --range main..feature
git config diff.submodule diff
expect_exit "submodule bump with diff.submodule=diff matches the default result" 0 "$SCANNER" --range main..feature

colored_diff="$workdir/colored.diff"
printf '\033[1mdiff --git a/a b/a\033[m\n\033[1m--- a/a\033[m\n\033[1m+++ b/a\033[m\n\033[32m+\033[m\033[32mdeploy token %s\033[m\n' "$token" > "$colored_diff"
expect_exit "AC-10: --diff finds the token in a colored added line" 1 sh -c "'$SCANNER' --diff 'colored diff' < '$colored_diff'"

printf '\n-- Summary --\n  PASS: %s\n  FAIL: %s\n' "$pass" "$fail"
[ "$fail" -eq 0 ]
