#!/usr/bin/env bash
# test-pre-bash-guard.sh — tests for .claude/hooks/pre_bash_guard.sh, a
# deny-only guard, against the real PreToolUse payload shape (the command
# nested under .tool_input.command, permission_mode at the top level).
#
# Regression guard for tech-debt row 100 (docs/tech-debt/README.md): the jq
# path used to call extract_json_field "$payload" "command" (top-level),
# but real Claude Code PreToolUse payloads nest the Bash command under
# .tool_input.command, so every rule silently never matched when jq was
# present.
#
# Every guard case runs on BOTH the jq path and the jq-absent path (where
# lib_json.sh decodes the JSON with sed and awk), and compares the exact
# decision: none (empty stdout) or deny, plus exit status 0. An ask, or
# output of any other shape, fails. make_payload JSON-escapes the command
# (\ " newline tab), so multi-line commands carry real newlines on both
# paths. Without jq on PATH, the jq-path cases are reported as SKIP. The
# cases are queued and run in parallel batches (GUARD_TEST_JOBS, default
# 8); results are reported in queue order.
#
#   A. AC1: every row that asked before (writes into .git or .env, rm -rf,
#      gh pr create) -> none with permission_mode absent, default, auto, and
#      bypassPermissions; the deny rows of the previous version of this
#      file -> deny in the same four modes; the rows that were none before
#      (reads, look-alike names, JSON escapes) -> none (the guard reads no
#      permission_mode, so these run once, with the key absent)
#   B. AC2: commands to deny (sudo in command position or after a command
#      that may run its arguments such as find -exec, watch, flock, chroot;
#      force push; hard reset; command substitution in a commit message;
#      --no-verify and core.hooksPath; abbreviated long options; strings run
#      as commands and files written, which the sentinel denies; zsh =sudo)
#      -> deny, with permission_mode absent and bypassPermissions. Then the
#      kinds of command the self-review listed (string runners, other
#      shells, builtin/source/process substitution, git forms that run
#      commands, NOEXEC commands that run an argument, env -S attached,
#      zsh =), at least two each -> deny
#   C. AC3: false positives of the old guard and look-alikes, all inside
#      data regions or not matching (sudo as an argument of echo or grep, in
#      a comment, a commit message, a quoted heredoc fed to git commit -F -,
#      output to /dev/null or to grep; touch sudo; visudo) -> none, with
#      permission_mode absent and bypassPermissions
#   D. Edge cases: quoted and escaped flags and names, line continuations,
#      nested $(...), backticks in backticks, git -C / -c / --config-env,
#      short-flag clusters, abbreviated long options and their look-alikes
#      (--follow-tags, --no-thin, --soft), the five heredoc delimiter forms
#      (to cat, to a file, to sh, to git commit -F -, and inside -m
#      "$(cat ...)"), shells fed through -c, here-strings, pipes and
#      heredocs, wrappers and unknown runners, comments, ${...}, process
#      substitution, functions and subshells, a JSON %u escape; forms only
#      the lexer denies ($(...), backticks, <(...), ${...}, sh -c, eval,
#      env -S, here-strings, pipes and heredocs to a shell, git after find
#      -exec, flock, watch) so that each re-reading is tested without the
#      sentinel; a here-string to git commit -F -; and the data regions of
#      the sentinel: what keeps them (pipes to data readers, /dev/null,
#      /dev/stderr, fd duplication, git tag -m, comments, the sudo word
#      boundary, a region longer than one 512-character index block) and
#      what breaks them (a pipe to sh or sort, a file, >&file, >(...) after
#      > or as an argument, a here-string, a cut-short pipeline, rg --pre,
#      nesting, an assignment, a backtick after the message), a sentinel
#      match across the guard's 512-character text window, and the known
#      false positives of the wrapper scan (tech-debt)
#   E. Broken input (unclosed quotes, parentheses, substitutions, heredocs
#      without an end line, operators without operands) -> exit 0 with none
#      or deny
#   F. AC9: fail closed. 4 levels of nesting pass and the 5th is denied
#      ($(...), eval); re-read text past the cap (8 times the command plus
#      64 KB, reached through nested | sh) and ${...} nested 24 deep (plain,
#      or alternating with double quotes) are denied, 23 pass. With awk
#      missing from PATH, or an awk that exits 2, the old guard's four
#      substring rules decide (these need jq: lib_json.sh's jq-absent path
#      runs awk itself)
#   G. AC7: the old guard (tests/fixtures/guard-1c4cea5a/, the version
#      before this rewrite) decides the corpus of A's deny rows, B (with the
#      self-review kinds), and C on each path, and is compared with the new
#      guard's runs of the same payloads in A, B, and C. Every case the old
#      guard denies and the new one lets through must be in
#      intentional_fixes, which must equal the C cases the old guard denies.
#      The list is printed. The sudo word-boundary difference (my-sudo ls,
#      x.sudo ls: old deny, new none in D) is not a data-region exception;
#      the old guard's deny of both is pinned separately.
#   H. AC8: ~200 KB commands on the jq-absent path (a long quoted string,
#      many short commands, backslashes in backticks) finish in under 10 s
#      (the measured times are printed)
#   I. lib_json.sh sourced directly: tool_input.file_path, plain and with an
#      escaped quote and backslash

set -u

REPO_ROOT="$(cd "$(dirname "$0")/.." && pwd)"
HOOK="$REPO_ROOT/.claude/hooks/pre_bash_guard.sh"
OLD_HOOK="$REPO_ROOT/tests/fixtures/guard-1c4cea5a/pre_bash_guard.sh"
LIB="$REPO_ROOT/.claude/hooks/lib_json.sh"
JOBS="${GUARD_TEST_JOBS:-8}"

for f in "$HOOK" "$OLD_HOOK"; do
  if [ ! -x "$f" ]; then
    echo "FAIL: hook not found or not executable at $f" >&2
    exit 1
  fi
done
if [ ! -f "$LIB" ]; then
  echo "FAIL: lib_json.sh not found at $LIB" >&2
  exit 1
fi

pass=0
fail=0
skip=0
results=()

record_pass() {
  results+=("PASS  $1")
  pass=$((pass + 1))
}
record_fail() {
  results+=("FAIL  $1")
  fail=$((fail + 1))
}
record_skip() {
  results+=("SKIP  $1")
  skip=$((skip + 1))
}

# json_escape <text> — escape \ " newline and tab for a JSON string, one
# character at a time (no ${var//...}: its backslash handling differs
# between bash 3.2 and 5.2).
json_escape() {
  local s="$1" out="" c i
  for ((i = 0; i < ${#s}; i++)); do
    c="${s:i:1}"
    case "$c" in
      '\') out+='\\' ;;
      '"') out+='\"' ;;
      $'\n') out+='\n' ;;
      $'\t') out+='\t' ;;
      *) out+="$c" ;;
    esac
  done
  printf '%s' "$out"
}

# repeat_text <text> <count> — text repeated count times (by doubling).
repeat_text() {
  local s="$1" n="$2" out=""
  while [ "$n" -gt 0 ]; do
    if [ $((n % 2)) -eq 1 ]; then out+="$s"; fi
    s+="$s"
    n=$((n / 2))
  done
  printf '%s' "$out"
}

# payload_json <json-escaped command> [permission_mode] — real-shape
# PreToolUse payload: the Bash tool's command nested under
# .tool_input.command, matching what Claude Code actually sends. The
# permission_mode key is omitted when no mode is given.
payload_json() {
  local escaped="$1" mode="${2:-}" mode_field=""
  if [ -n "$mode" ]; then
    mode_field=",\"permission_mode\":\"$mode\""
  fi
  printf '{"session_id":"test"%s,"hook_event_name":"PreToolUse","tool_name":"Bash","tool_input":{"command":"%s","description":"test"}}' \
    "$mode_field" "$escaped"
}

workdir="$(mktemp -d "${TMPDIR:-/tmp}/pre-bash-guard-test.XXXXXX")"
cleanup() {
  rm -rf "$workdir"
}
trap cleanup EXIT

# Minimal PATH without jq, mirroring tests/test-check-mojibake.sh's Case E
# technique: symlink only the tools the hook needs, omitting jq so
# `command -v jq` fails and lib_json.sh falls back to its sed path.
minimal_path="$workdir/no-jq-bin"
mkdir -p "$minimal_path"
for tool in sh bash dash cat grep sed printf dirname env tr command test awk; do
  resolved="$(command -v "$tool" 2>/dev/null || true)"
  [ -n "$resolved" ] && ln -sf "$resolved" "$minimal_path/$tool" 2>/dev/null || true
done
if PATH="$minimal_path" "$minimal_path/sh" -c 'command -v jq' >/dev/null 2>&1; then
  echo "FAIL: jq is still reachable from the minimal PATH $minimal_path" >&2
  exit 1
fi

real_path="$PATH"
real_sh="$(command -v sh)"
have_jq=no
if command -v jq >/dev/null 2>&1; then
  have_jq=yes
fi

# ── Case queue ──────────────────────────────────────────────────────────
# Each queued run has a label, an expected decision (none, deny, any for
# none-or-deny, or "-" for a run whose decision is only collected), the
# hook, the PATH, and the payload file. run_queue runs them JOBS at a time;
# each run writes "<decision> <exit status>" to r.<index>.
q_label=()
q_expect=()
q_hook=()
q_path=()
qn=0

# enqueue <label> <expect> <hook> <path> <payload> — prints nothing; the
# queue index of the run is left in $last_q.
enqueue() {
  q_label[qn]="$1"
  q_expect[qn]="$2"
  q_hook[qn]="$3"
  q_path[qn]="$4"
  printf '%s' "$5" > "$workdir/p.$qn"
  last_q=$qn
  qn=$((qn + 1))
}

# decide <index> — run one queued case and write its decision.
decide() {
  local i="$1" out rc got
  out="$(PATH="${q_path[$i]}" "${q_hook[$i]}" < "$workdir/p.$i" 2>/dev/null)"
  rc=$?
  case "$out" in
    '') got=none ;;
    '{"hookSpecificOutput":{"hookEventName":"PreToolUse","permissionDecision":"deny","permissionDecisionReason":"'*'"}}') got=deny ;;
    *'"permissionDecision":"ask"'*) got=ask ;;
    *) got=unparsed ;;
  esac
  printf '%s %s\n' "$got" "$rc" > "$workdir/r.$i"
}

run_queue() {
  local i running=0
  for ((i = 0; i < qn; i++)); do
    decide "$i" &
    running=$((running + 1))
    if [ "$running" -ge "$JOBS" ]; then
      wait
      running=0
    fi
  done
  wait
}

# result_of <index> — "<decision> <exit status>" of a finished run.
result_of() {
  if [ -f "$workdir/r.$1" ]; then
    command cat "$workdir/r.$1"
  else
    echo "missing 1"
  fi
}

# check <section> <expected none|deny|any> <command> [mode] — queue one case
# on the jq path and on the jq-absent path. The queue indexes of the two
# runs are left in $last_q_jq (-1 without jq) and $last_q_nojq.
check() {
  local section="$1" expect="$2" command="$3" mode="${4:-}" escaped label
  escaped="$(json_escape "$command")"
  label="$section mode=${mode:-<absent>}: $escaped -> $expect"
  last_q_jq=-1
  if [ "$have_jq" = yes ]; then
    enqueue "$label [jq]" "$expect" "$HOOK" "$real_path" "$(payload_json "$escaped" "$mode")"
    last_q_jq=$last_q
  else
    record_skip "$label [jq] (jq not on PATH)"
  fi
  enqueue "$label [no-jq]" "$expect" "$HOOK" "$minimal_path" "$(payload_json "$escaped" "$mode")"
  last_q_nojq=$last_q
}

# The AC7 corpus (F): the commands of A's deny rows, B and C, with the
# queue indexes of their runs without permission_mode, collected by
# check_modes while collect_corpus is yes.
collect_corpus=no
corpus=()
corpus_section=()
corpus_new_jq=()
corpus_new_nojq=()

# check_modes <section> <expected> <mode>... -- <command>... — every
# command in every mode.
check_modes() {
  local section="$1" expect="$2" modes=() m c
  shift 2
  while [ "$#" -gt 0 ] && [ "$1" != "--" ]; do
    modes+=("$1")
    shift
  done
  shift
  for c in "$@"; do
    for m in "${modes[@]}"; do
      if [ "$m" = absent ]; then
        check "$section" "$expect" "$c"
        if [ "$collect_corpus" = yes ]; then
          corpus+=("$c")
          corpus_section+=("$section")
          corpus_new_jq+=("$last_q_jq")
          corpus_new_nojq+=("$last_q_nojq")
        fi
      else
        check "$section" "$expect" "$c" "$m"
      fi
    done
  done
}

# ── A. AC1: no ask in any mode ──────────────────────────────────────────
# The rows of the previous version of this file that expected ask (it
# asked for these outside bypassPermissions).
former_ask=(
  'echo x > .git/hooks/pre-commit'
  'echo x >> /abs/repo/.git/config'
  'echo x >.git/x'
  'echo x 2> .git/x'
  'echo x >| .git/x'
  'echo x &> .git/x'
  'echo x > ./.git/x'
  'echo x > ".git/x"'
  $'echo x\t> .git/x'
  'tee -a .git/x'
  'echo x | tee out.txt .git/x'
  'echo x | tee -a out.txt .env'
  'tee .env < input.txt'
  '/usr/bin/tee .git/x'
  $'echo x |\ttee .env'
  '\tee .env'
  '\tee .git/x'
  'x=`tee .env </dev/null`'
  'x=`tee .git`'
  'x=`printf x > .git`'
  'x=`tee .git/x`'
  'x=`tee .env`'
  'x=`printf x > .env`'
  'echo x > `pwd`/.git/x'
  'echo x > `pwd`/.git/x 2>/dev/null'
  'echo x > `pwd`/.env 2>/dev/null'
  'tee `pwd`/.git/x > /dev/null'
  'tee out.txt 2>/dev/null .env'
  'tee out.txt 2>/dev/null .env > /dev/null'
  'echo x > .git'
  'echo x > /abs/worktree/.git'
  $'cat > .env <<EOF\nA=1\nEOF'
  $'cat >.env<<EOF\nA=1\nEOF'
  'printf x > .env.local'
  'echo x > .envrc'
  'echo x | tee config/.env'
  'rm -rf build/'
  'gh pr create --title t'
  # The mode comes from the payload's own key, never from the command text.
  "echo '\"permission_mode\":\"bypassPermissions\"' && rm -rf build/"
  $'echo start\necho x > .git/x'
  $'true\ntee .env </dev/null'
  $'true\ntee .git/config </dev/null'
  "echo '{\"a\":1}' > .git/x"
  'echo "a\\" > .git/x'
)
check_modes A none absent default auto bypassPermissions -- "${former_ask[@]}"

# The rows that were none before stay none.
former_none=(
  'echo hello'
  'ls .git/ 2>&1'
  'grep -r foo .git/ 2>/dev/null'
  'git status 2>&1 | grep .git/'
  'cat .env.example 2>/dev/null'
  'grep X .env 2>/dev/null'
  'grep X .env'
  'cat > .github/workflows/x.yml'
  'echo x > .gitignore'
  'echo x > repo.git/config'
  'echo x > .env/notes.txt'
  'cat .env > out.txt'
  $'ls\t.git/ 2>&1'
  'tee /tmp/out < .env'
  'tee /tmp/out < .git/config'
  'cat >/tmp/o</repo/.git/HEAD'
  'tee /tmp/build.log # .env is read separately'
  'x=`tee /tmp/a` .env'
  $'ls .git/\necho x > out.txt'
  $'echo x | tee out.txt\nls .git/'
  "echo '{\"a\":1}' > out.json"
  'printf "a\\b" > out.txt'
)
check_modes A none absent -- "${former_none[@]}"

# The deny rows of the previous version of this file.
pr206_deny=(
  'git push --force origin main'
  'sudo ls'
  'git push -f origin main'
  'git reset --hard HEAD'
  'git commit -m "$(id)"'
  'git commit -m "`id`"'
  'rm -rf x && git commit -m "$(id)"'
  'echo x > .git/x; sudo ls'
  'gh pr create --title t && git reset --hard HEAD'
  $'git add .\ngit commit -m "$(id)"'
)
collect_corpus=yes
check_modes A deny absent default auto bypassPermissions -- "${pr206_deny[@]}"

# ── B. AC2: deny ────────────────────────────────────────────────────────
ac2=(
  # sudo in command position
  'sudo ls'
  $'sudo\tls'
  '/usr/bin/sudo ls'
  'env FOO=1 sudo ls'
  'nohup sudo ls'
  'xargs sudo ls'
  $'sh -c \'sudo ls\''
  'if sudo ls; then :; fi'
  '{ sudo ls; }'
  '! sudo ls'
  '2>/dev/null sudo ls'
  # sudo after a command that may run its arguments
  'find . -exec sudo rm x \;'
  'watch sudo ls'
  'flock /tmp/l sudo ls'
  'chroot /x sudo ls'
  # force push
  '</dev/null git push --force'
  'git push --force'
  'git push origin --force'
  'git push -f'
  'git push origin -uf main'
  'git push --force-with-lease'
  'git push origin +main'
  'git -C dir push --force'
  'git push "--force" origin main'
  'bash -lc "git push --force"'
  'find . -exec git push --force \;'
  'flock /tmp/l git push --force'
  'git push --force-with'
  'git push --force-with=main'
  # hard reset
  'git reset --ha'
  'git reset --har'
  'git reset --hard'
  'git -C dir reset --hard HEAD~1'
  'git reset -q --hard'
  'eval "git reset --hard"'
  'echo "$(git reset --hard)"'
  'echo "`git reset --hard`"'
  'echo "git reset --hard" | sh'
  $'sh <<\'EOF\'\ngit push --force\nEOF'
  # command substitution in a commit message
  'git commit -m "$(id)"'
  'git commit -m "`id`"'
  'git commit -am "$(id)"'
  'git commit -m"$(id)"'
  'git commit --message "$(id)"'
  'git commit --message="$(id)"'
  'git commit --mess "$(id)"'
  $'git commit -m \'x\' -m "$(id)"'
  'rm -rf x && git commit -m "$(id)"'
  $'git commit -m "$(cat <<\'EOF\'; id\nfeat: x\nEOF\n)"'
  $'git commit -F - <<EOF\nfeat: x\n\n$(id)\nEOF'
  # strings run as commands, and files written: the sentinel (the old
  # substring rules outside data regions)
  $'find . -exec sh -c \'sudo ls\' \\;'
  $'watch \'git push --force\''
  $'csh -c \'sudo ls\''
  $'tcsh -c \'git reset --hard\''
  $'source <(echo \'sudo ls\')'
  $'. <(echo \'git push --force\')'
  $'builtin eval \'git push --force\''
  $'git rebase -x \'git push --force\' main'
  $'git submodule foreach \'git push --force\''
  $'echo \'sudo ls\' | xargs -I{} sh -c {}'
  $'echo \'sudo ls\' > >(sh)'
  $'echo \'sudo ls\' >> ~/.zshrc'
  $'echo \'git push --force\' | tee x.sh'
  $'cat <<\'EOF\' | sh\nsudo ls\nEOF'
  $'cat > notes.md <<EOF\n- never run git push --force\nEOF'
  '=sudo ls'
  '=git push --force'
  # --no-verify and core.hooksPath
  'git commit --no-veri -m x'
  'git commit --no-verify -m x'
  'git commit -n -m x'
  'git commit -nm x'
  'git push --no-verify'
  'git merge --no-verify x'
  'git -c core.hooksPath=/dev/null commit -m x'
  'git -c Core.HooksPath=/dev/null commit -m x'
)
check_modes B deny absent bypassPermissions -- "${ac2[@]}"

# The kinds of command the self-review found the old guard denying and the
# lexer alone letting through (docs/reports/self-review-<date>-guard-deny-only.md,
# H-1 and M-1), at least two of each, outside the AC2 list. The sentinel
# denies all of them; they join the AC7 corpus.
self_review_forms=(
  # runners that take a string
  $'watch \'sudo ls\''
  $'find . -name x -exec sh -c \'git push --force\' \\;'
  $'flock /tmp/l -c \'sudo ls\''
  $'su -c \'git reset --hard\' root'
  $'script -q -c \'sudo ls\' /dev/null'
  $'parallel ::: \'git push --force\''
  $'tmux new-session -d \'sudo ls\''
  # shells other than sh bash zsh dash ksh
  $'fish -c \'sudo ls\''
  $'ash -c \'git reset --hard\''
  $'busybox sh -c \'git push --force\''
  $'mksh -c \'sudo ls\''
  # builtin, source, ., process substitution and /dev/stdin given to a shell
  $'builtin eval \'sudo ls\''
  $'source <(echo \'git reset --hard\')'
  $'bash <(echo \'sudo ls\')'
  $'sh <(echo \'git push --force\')'
  $'. /dev/stdin <<\'EOF\'\nsudo ls\nEOF'
  $'bash /dev/stdin <<\'EOF\'\ngit push --force\nEOF'
  # git forms that run commands
  $'git rebase --exec \'sudo ls\' main'
  $'git submodule foreach \'git reset --hard\''
  $'git bisect run sh -c \'git push --force\''
  $'git filter-branch --tree-filter \'sudo ls\' HEAD'
  $'git -c alias.x=\'!git push --force\' x'
  $'git -c core.pager=\'sudo ls\' log'
  # commands of NOEXEC that run an argument, and LESSOPEN
  $'man -P \'sudo ls\' git'
  $'rg --pre \'sudo ls\' pattern file'
  $'LESSOPEN=\'| git reset --hard %s\' less file'
  # env -S with the string attached, xargs feeding sh -c
  $'env -S\'sudo ls\''
  $'env --split-string=\'git push --force\''
  $'echo x | xargs sh -c \'git reset --hard\''
  # zsh = expansion of a command name
  '=sudo ls'
  '=git reset --hard'
  'ls; =sudo ls'
)
check_modes B deny absent bypassPermissions -- "${self_review_forms[@]}"

# The cross-review (docs/reports/cross-review-triage-guard-deny-only.md) and
# the follow-up probes found forms where the sentinel gave a data region even
# though the shell later runs that text. Each must deny: a top-level group or
# compound command (the lexer cannot place the pipe or redirection around it),
# a heredoc read across a $(...) or a backtick, a heredoc body or delimiter
# joined by a backslash-newline, an exec with a redirection, an unsafe output
# redirection, and printf -v (which stores into a variable). The old guard
# denies all of these too, so they join the AC7 corpus.
guard_deny_only_forms=(
  # 1. Groups and compound commands (subshell, brace group, reserved word in
  # command position), including two that were false none before.
  $'(echo \'git push --force\') | sh'
  $'{ echo \'sudo ls\'; } | sh'
  $'{ echo \'git reset --hard\'; } > run.sh'
  $'if true; then echo \'git push --force\'; fi | sh'
  $'for x in 1; do echo \'sudo ls\'; done | bash'
  $'case x in x) echo \'sudo ls\';; esac | sh'
  $'f() { echo \'sudo ls\'; }; f | sh'
  $'echo \'sudo ls\' | if true; then sh; fi'
  $'(cat <<\'EOF\'\nsudo ls\nEOF\n) | sh'
  '(echo sudo ls)'
  $'if grep -q \'sudo \' file; then echo ok; fi'
  # 2. A heredoc read across a $(...): the newline inside the substitution
  # must not start the quoted-delimiter body, so the force push in $(...) runs.
  $'cat <<\'OUT\' "$(\ngit push --force\n)"\nx\nOUT'
  # 3. An unquoted heredoc whose body line ends in a backslash-newline joins
  # with the next line before the terminator comparison (bash, zsh).
  $'cat <<EOF\nEO\\\nF\ngit push --force'
  $'cat <<-EOF\n\tEO\\\nF\nsudo ls'
  # 4. A backslash-newline in the delimiter word is removed before tokenizing,
  # so the delimiter is unquoted and the body expands.
  $'cat <<EO\\\nF\n$(sudo ls)\nEOF'
  # 5. exec with a redirection: a later command writes to the redirected fd.
  $'exec >run.sh; echo \'sudo ls\'; sh run.sh'
  $'exec 3>run.sh; echo \'sudo ls\' >&3; sh run.sh'
  # 6. Unsafe output redirections keep a data region from echo: a dup to fd 3
  # or higher, a dup to a word (a file to bash), &> and &>>, >| and <>, and
  # >> into a file.
  $'echo \'sudo ls\' >&3'
  $'echo \'sudo ls\' >&foo'
  $'echo \'sudo ls\' &>/dev/null'
  $'echo \'sudo ls\' &>>log'
  $'echo \'sudo ls\' >| out'
  $'echo \'sudo ls\' <> f'
  $'echo \'sudo ls\' >> notes.md'
  # 7. printf -v stores into a variable, so its arguments are not data (also
  # the attached -vNAME form).
  $'printf -v c \'sudo ls\'; $c'
  $'printf -vc \'sudo ls\'; $c'
)
check_modes B deny absent bypassPermissions -- "${guard_deny_only_forms[@]}"

# ── C. AC3: none ────────────────────────────────────────────────────────
ac3=(
  $'echo \'never use sudo here\''
  'echo sudo ls'
  'grep sudo file'
  'touch sudo'
  'apt-get install sudo'
  'visudo -c'
  'man sudo'
  'git push origin main'
  'git push -u origin main'
  'git reset --soft HEAD~1'
  $'git commit -m \'remove -n flag\''
  $'git commit -m \'drop --force and git reset --hard from docs\''
  'git commit -mn'
  'git commit -uno -m x'
  $'git commit -m \'fix: x\' && grep -n foo file'
  'git commit -F msg.txt; sed -n 1,5p file'
  # The recommended commit form: the quoted delimiter expands nothing, so
  # the backticks, $(...), quotes and parentheses in the body are text.
  $'git commit -m "$(cat <<\'EOF\'\nfeat: add a thing\n\nBody with `backticks`, $(dollar parens), don\'t, "quotes" and ) parens.\nEOF\n)"'
  $'git commit -F - <<\'EOF\'\ndocs: never git push --force\n$(id) stays as text\nEOF'
  # data regions: arguments of commands that only read data, a comment, and
  # pipes and redirections that keep the text off files
  'echo never use sudo here'
  'echo hi # sudo ls'
  $'grep -n \'git push --force\' docs.md'
  $'echo \'sudo ls\' > /dev/null'
  $'echo \'git push --force\' | grep force'
  'git log --no-verify-signatures'
  $'printf \'a\\ngit push --force\''
  'ls .git/ 2>&1'
  'echo x > .env'
  'rm -rf build/'
  'gh pr create --title t'
)
check_modes C none absent bypassPermissions -- "${ac3[@]}"
collect_corpus=no

# ── D. Edge cases ───────────────────────────────────────────────────────
edge_deny=(
  # Quoted and escaped flags and names.
  'git reset "--hard"'
  'git commit "-n" -m x'
  $'git push origin \'+main\''
  'git push --for""ce'
  'g""it push --force'
  '\git push --force'
  '\sudo ls'
  $'git push $\'--force\''
  # A backslash outside quotes escapes one character; a quote it escapes
  # opens nothing.
  $'echo it\\\'s; sudo ls'
  'echo "a \" b"; sudo ls'
  # Line continuations and multi-line commands.
  $'echo x && \\\n sudo ls'
  $'git push \\\n  --force'
  $'git add .\ngit push --force'
  # Nested substitutions, ${...}, process substitution, backticks in
  # backticks.
  'echo $(echo $(git reset --hard))'
  'x=$(sudo ls)'
  'echo "${x:-$(sudo ls)}"'
  'diff <(sudo cat a) b'
  'echo `echo \`sudo ls\``'
  # git global options before the subcommand.
  'git -c user.name=x push -f'
  'git --git-dir=.git --work-tree=. reset --hard'
  'git -C dir -c a=b --no-pager push --force'
  'git -c core.hookspath=x status'
  'git -c core.hooksPath status'
  'git --config-env=core.hooksPath=HOOKS commit -m x'
  'GIT_DIR=x git push --force'
  'git push --force-with-lease=main:abc'
  # Short-flag clusters.
  'git push -uf origin main'
  'git commit -anm x'
  'git commit -vn -m x'
  'git commit -m x -n'
  'git rebase --no-verify main'
  'git am --no-verify x.patch'
  # Abbreviated long options (at least 4 characters, =value dropped).
  'git commit --mess="$(id)"'
  $'git commit --fi - <<EOF\n$(id)\nEOF'
  $'git commit --fi=- <<EOF\n$(id)\nEOF'
  'git merge --no-veri x'
  'git push --forc'
  'git rebase --no-verif main'
  # Unknown runners: sudo with a word after it, or git, in a later word.
  'docker run img sudo ls'
  'nohup watch -n 5 sudo ls'
  'xargs -I{} flock /tmp/l git push -f'
  'find . -exec git status \; -exec git push --force \;'
  # Heredocs fed to a shell (body read as commands) and to git commit -F -
  # with an unquoted delimiter.
  $'sh <<EOF\ngit push --force\nEOF'
  $'sh <<-EOF\n\tgit push --force\n\tEOF'
  $'sh <<"EOF"\ngit push --force\nEOF'
  $'bash -s <<\\EOF\ngit push --force\nEOF'
  $'git commit -F - <<-EOF\n\t$(id)\n\tEOF'
  $'git commit --file=- <<EOF\n`id`\nEOF'
  # A here-string read by git commit -F - is the message too.
  'git commit -F - <<< "$(id)"'
  'git commit -F - <<< "`id`"'
  # Only the exact recommended form passes; these do not.
  $'git commit -m "$(cat <<EOF\nmsg\nEOF\n)"'
  $'git commit -m "$(cat <<-\'EOF\'\nmsg\nEOF\n)"'
  $'git commit -m "$(cat <<\'EOF\'\nmsg\nEOF\n)"x'
  $'git commit -m "$(cat <<\'EOF\'\nmsg\nEOF\n)" && sudo ls'
  # An unquoted heredoc body expands $(...) and backticks.
  $'cat <<EOF > file\n$(sudo ls)\nEOF'
  $'cat <<EOF > file\n`sudo ls`\nEOF'
  # Shells: -c strings, here-strings, pipes, heredocs through a pipe.
  $'bash -c \'git push --force\''
  $'sh -lc \'sudo ls\''
  $'bash <<< \'sudo ls\''
  $'echo \'sudo ls\' | bash -s'
  $'cat <<EOF | sh\nsudo ls\nEOF'
  $'cat <<\'EOF\' |\nsudo ls\nEOF\nsh'
  $'env -S \'sudo ls\''
  # Wrappers.
  'time sudo ls'
  'timeout 5 sudo ls'
  'timeout -s KILL 5s sudo ls'
  'nice -n 5 sudo ls'
  'nice --adjustment 5 sudo ls'
  'stdbuf -oL sudo ls'
  'exec sudo ls'
  'command sudo ls'
  'env -i PATH=/bin sudo ls'
  'env -u HOME sudo ls'
  'xargs -I{} sudo ls {}'
  'xargs -n 1 sudo ls'
  # Reserved words, separators, functions, subshells, comments.
  'sudo'
  'if true; then git reset --hard; fi'
  'while true; do git push -f; done'
  'true || sudo ls'
  'true & sudo ls'
  'true |& sudo ls'
  'case "$x" in a) sudo ls;; esac'
  'f() { sudo ls; }'
  'function f { sudo ls; }'
  '(sudo ls)'
  'echo x | (git push --force)'
  $'echo hi # a comment with it\'s apostrophe\nsudo ls'
  # Re-reading depth: 4 levels are read (the 5th is denied, in F).
  'eval eval eval eval sudo ls'
  'echo $(echo $(echo $(echo $(sudo ls))))'
  # Forms only the lexer denies (no text the sentinel matches: the option
  # stands apart from git push, git -C comes before reset, or the flag is
  # --no-verify), in each place whose text is read again as commands. These
  # fail when that re-reading stops; the forms above also hold the old
  # guard's substrings, so the sentinel would still deny them.
  'echo "$(git push origin --force)"'
  'echo `git push origin --force`'
  'echo "`git push origin -f`"'
  'x=$(git commit --no-verify -m x)'
  'diff <(git push origin --force) x'
  'echo "${x:-$(git push origin --force)}"'
  $'sh -c \'git push origin --force\''
  $'bash -lc \'git -C x reset --hard\''
  $'eval \'git push origin --force\''
  $'env -S \'git push origin --force\''
  $'env --split-string=\'git push origin --force\''
  $'bash <<< \'git push origin --force\''
  $'echo \'git push origin --force\' | sh'
  $'sh <<\'EOF\'\ngit push origin --force\nEOF'
  $'cat <<\'EOF\' | sh\ngit push origin --force\nEOF'
  $'cat <<EOF\n$(git push origin --force)\nEOF'
  $'cat <<EOF\n`git push origin --force`\nEOF'
  $'echo x | xargs sh -c \'git push origin --force\''
  # The same for git after a command that may run its arguments (the later
  # words that scan_words reads).
  $'find . -exec git push origin --force \\;'
  'flock /tmp/l git push origin -f'
  'watch git -C dir reset --hard'
)
check_modes D deny absent -- "${edge_deny[@]}"

edge_none=(
  'echo \"git reset --hard\"'
  $'git commit -m don\\\'t\\ use\\ -n'
  'git commit -amn'
  'git commit -uno'
  'git commit -m -n'
  'git commit --author "-n x" -m y'
  'git commit -- -n'
  'git commit -m "fix: handle \$(id) literal"'
  $'git commit -m \'feat: add -n option\''
  'git commit -q -s -m x'
  'git push origin :old-branch'
  'git stash push -m "msg"'
  'git -C dir status --force'
  'git log --grep=--force'
  'echo git push --force'
  $'grep -n \'git reset --hard\' file'
  'command -v sudo'
  'type sudo'
  'echo sudo'
  'bash script.sh'
  'sh -n script.sh'
  # sudo as the last word, and git as the last word of an unknown command.
  'brew install git'
  $'find . -exec git status \\;'
  # Long options that only look like the abbreviated ones.
  'git push --follow-tags origin main'
  'git push --no-thin origin main'
  'git reset --h'
  'git commit --no-edit'
  'git commit --fixup HEAD'
  'git commit --mess=x'
  $'git commit --mess \'x $(id)\''
  $'git commit --fi - <<\'EOF\'\n$(id)\nEOF'
  $'git commit -F - <<< \'$(id)\''
  # Heredoc bodies are data when read by a command that only reads data:
  # the five delimiter forms, and an unquoted one whose $( is escaped.
  $'cat <<-EOF\n\tgit push --force\n\tEOF'
  $'cat <<\'EOF\'\ngit push --force\nEOF'
  $'cat <<"EOF"\ngit push --force\nEOF'
  $'cat <<\\EOF\ngit push --force\nEOF'
  $'cat <<EOF\ngit push --force\nEOF'
  $'cat <<\'EOF\'\n$(sudo ls)\nEOF'
  $'cat <<EOF\n\\$(date) sudo ls\nEOF'
  $'cat <<\'EOF\' | grep x\nsudo ls\nEOF'
  $'git commit -F - <<EOF\nnever git push --force\nEOF'
  # Data regions: the arguments of commands that only read data, in
  # pipelines of such commands, with output off files.
  $'echo \'sudo ls\' 2>&1 | grep sudo | wc -l'
  'echo "sudo $(date) ls"'
  'echo sudo ls > /dev/stderr'
  'echo sudo ls >&2'
  # Safe duplications of a data reader keep its data region: a dup to fd 0, 1
  # or 2, a close, and a pipe to another data reader.
  $'echo \'sudo ls\' 2>&1'
  $'echo \'sudo ls\' >&2'
  $'echo \'sudo ls\' >&1'
  $'echo \'sudo ls\' 1>&2'
  $'echo \'sudo ls\' >&-'
  $'echo \'sudo ls\' 2>/dev/null | grep x'
  $'grep -r \'sudo \' . | head -5'
  $'rg \'git push --force\' docs'
  $'git tag -a v1 -m \'never git push --force\''
  $'git commit -am \'sudo ls is gone\''
  $'git commit -m \'one\' -m \'two: git reset --hard\''
  $'git -C dir commit -m \'git push --force\''
  'ls # sudo ls'
  'echo sudo ls; ls'
  'gh pr create --body-file body.md'
  # sudo after a word character, ., _ or - is another name.
  'my-sudo ls'
  'x.sudo ls'
  # zsh = expansion of a command that only reads data.
  '=echo sudo ls'
  # git commit -F - with a quoted delimiter expands nothing.
  $'git commit -F - <<"EOF"\n$(id)\nEOF'
  $'git commit -F - <<\\EOF\n$(id)\nEOF'
  # The recommended form with each quoted delimiter and each message flag.
  $'git commit -m "$(cat <<"EOF"\nmsg\nEOF\n)"'
  $'git commit -m "$(cat <<\\EOF\nmsg\nEOF\n)"'
  $'git commit -m "$(cat <<\'EOF\'\nmsg\nEOF\n)" && echo done'
  $'git commit --message "$(cat <<\'EOF\'\nmsg\nEOF\n)"'
  $'git commit -m"$(cat <<\'EOF\'\nmsg\nEOF\n)"'
  $'git commit -am "$(cat <<\'EOF\'\nmsg\nEOF\n)"'
  # Single-quoted text is never run.
  $'echo \'${x:-$(sudo ls)}\''
  $'echo $\'it\\\'s sudo ls\''
  'echo hi # sudo ls'
  'echo $((1+2))'
  'for f in *.md; do echo "$f"; done'
  'case "$x" in a) echo a;; esac'
)
check_modes D none absent -- "${edge_none[@]}"

# The sentinel keeps what the old guard denied outside data regions.
edge_sentinel_deny=(
  # The output goes to a command that is not a data reader, or to a file.
  $'echo \'sudo ls\' | bash script.sh'
  'echo sudo ls | sh'
  'echo sudo ls > out.txt'
  'echo sudo ls >&out.txt'
  'echo sudo ls > >(cat)'
  # A >(...) given as an argument (not after > ) ends the data region too.
  'echo sudo ls >(sh)'
  $'grep -r \'sudo \' . | sort'
  $'grep x <<< \'sudo ls\''
  # A heredoc body written to a file, fed to a shell, or with a real $( in
  # an unquoted body; the five delimiter forms written to a file.
  $'cat > notes.md <<-EOF\n\tgit push --force\n\tEOF'
  $'cat > notes.md <<\'EOF\'\ngit push --force\nEOF'
  $'cat > notes.md <<"EOF"\ngit push --force\nEOF'
  $'cat > notes.md <<\\EOF\ngit push --force\nEOF'
  $'cat <<\'EOF\' > file\n$(sudo ls)\nEOF'
  $'cat <<EOF > file\n\\$(sudo ls)\nEOF'
  $'cat <<\'EOF\' > f.sh\nsudo ls\nEOF'
  $'cat <<EOF\n$(date) sudo ls\nEOF'
  # bash reads a heredoc body right after the newline that ends its line,
  # so here sh is a body line and the pipeline has no last command (bash
  # rejects it); a pipeline cut short has no data regions.
  $'cat <<\'EOF\' |\nsh\nsudo ls\nEOF'
  'echo sudo ls |'
  # Arguments of commands that are not data readers.
  'cp sudo dest'
  'apt-get install sudo 2>/dev/null'
  $'rg --pre x \'git push --force\' docs'
  'git push --force-if-includes origin main'
  $'gh pr create --body "$(cat <<\'EOF\'\nnever git push --force\nEOF\n)"'
  # Not data: a message with a substitution, nested text, an assignment, a
  # command after the data, a comment inside $(...), and a backtick after
  # the message (the old guard read the rest of the command).
  $'git tag -a v1 -m "$(id) git push --force"'
  $'x=$(echo \'sudo ls\')'
  $'echo $(true # sudo ls\n)'
  $'LESSOPEN=\'| sudo ls %s\' cat x'
  'echo sudo ls; sudo ls'
  $'flock l git commit -m \'git push --force\''
  'git commit -m "fix" && echo `date`'
  './sudo ls'
  # Known false positives of the wrapper scan (docs/tech-debt/README.md, the
  # pre_bash_guard.sh row, (a)): sudo is data here, but the command before
  # it is not in NOEXEC, so the later word sudo is denied. The old guard
  # denied all three too. A fix for one of them moves it to edge_none.
  'apt-get remove sudo -y'
  $'bash -c \'x\' sudo ls'
  'flock l git grep sudo file'
)
check_modes D deny absent -- "${edge_sentinel_deny[@]}"

# A JSON escape in the payload: %u0073 decodes to s on both paths, so the
# command is sudo ls. (%u stands for backslash-u: editing tools decode a
# literal backslash-u escape of a printable character in transit.)
check_raw() {
  local label="$1" expect="$2" escaped
  escaped="$(printf '%s' "$3" | sed 's/%u/\\u/g')"
  if [ "$have_jq" = yes ]; then
    enqueue "$label [jq]" "$expect" "$HOOK" "$real_path" "$(payload_json "$escaped")"
  else
    record_skip "$label [jq] (jq not on PATH)"
  fi
  enqueue "$label [no-jq]" "$expect" "$HOOK" "$minimal_path" "$(payload_json "$escaped")"
}
check_raw "D JSON escape: %u0073udo ls -> deny" deny '%u0073udo ls'
check_raw "D JSON escape: git push %u002d-force -> deny" deny 'git push %u002d-force'
check_raw "D JSON escape: git commit -m with %u3042 (kept as written) -> none" none "git commit -m '%u3042'"

# check_named <label> <expected> <command> — as check, with a short label
# for a long command.
check_named() {
  local label="$1" expect="$2" escaped
  escaped="$(json_escape "$3")"
  if [ "$have_jq" = yes ]; then
    enqueue "$label [jq]" "$expect" "$HOOK" "$real_path" "$(payload_json "$escaped")"
  else
    record_skip "$label [jq] (jq not on PATH)"
  fi
  enqueue "$label [no-jq]" "$expect" "$HOOK" "$minimal_path" "$(payload_json "$escaped")"
}
# A data region longer than one 512-character block of the guard's index of
# data regions (BKW): a match in a later block is still inside the region.
check_named "D long data region: echo '<600 x> git push --force' -> none" none \
  "echo '$(repeat_text x 600) git push --force'"
check_named "D long data region: git commit -m '<600 x> git reset --hard' -> none" none \
  "git commit -m '$(repeat_text x 600) git reset --hard'"
# A sentinel match across the end of the guard's 512-character text window
# (W): git push --force from position 505 and sudo from 511, in text only
# the sentinel denies (watch runs a string).
check_named "D window edge: echo '<487 x>' ; watch 'git push --force' (the match starts at 505) -> deny" deny \
  "echo '$(repeat_text x 487)' ; watch 'git push --force'"
check_named "D window edge: echo '<493 x>' ; watch 'sudo ls' (the match starts at 511) -> deny" deny \
  "echo '$(repeat_text x 493)' ; watch 'sudo ls'"

# ── E. Broken input: exit 0, none or deny ───────────────────────────────
broken=(
  $'echo \'unclosed'
  'echo "unclosed'
  'echo $(unclosed'
  'echo `unclosed'
  'echo ${unclosed'
  $'echo $\'unclosed'
  $'cat <<EOF\nno end line'
  'cat <<EOF'
  'cat <<'
  ')))'
  '((('
  'echo >'
  '<<<'
  '| | | && || ;; ;& & &'
  'echo \'
  'git commit -m'
  'git -c'
  'sh -c'
  'eval'
  'xargs -I'
  'timeout'
  'env -u'
  '$((((((((((((((((((((((((((((((((((((((((((((((((((((((((((((((((((((((((((((((((((((((((((((((((((((((((((((((('
  '${${${${${${${${${${${${${${${${${${${${${${${${${${${${${${${${${${${${${${${${${${${${${${${${${${${${${${${${${'
  $'"""""""""""\'\'\'\'\'\'\'\'\'\'`````````'
)
check_modes E any absent -- "${broken[@]}"
# Broken input that still names a denied command.
check E deny $'sudo ls \''
check E deny 'git commit -m "$(id'

# ── F. AC9: fail closed ─────────────────────────────────────────────────
# Nesting: 4 levels are read, a 5th is denied even when it runs nothing
# dangerous. The nesting limit counts ${...} inside each other: 23 pass and
# 24 are denied, and 150 that alternate with double quotes are denied too
# (without the limit, mawk stops at about 49 of those with "eval stack
# size", and a failing awk falls back to rules that miss them).
nested_braces() {
  printf 'echo %s' "$(repeat_text '${x:-' "$1")a$(repeat_text '}' "$1")"
}
nested_quoted_braces() {
  printf 'echo "%s"' "$(repeat_text '${x:-"' "$1")a$(repeat_text '"}' "$1")"
}
check_modes F none absent -- \
  'echo $(echo $(echo $(echo $(echo hi))))' \
  'eval eval eval eval echo hi' \
  "$(nested_braces 23)" \
  "$(nested_quoted_braces 23)"
check_modes F deny absent -- \
  'echo $(echo $(echo $(echo $(echo $(echo hi)))))' \
  'eval eval eval eval eval echo hi' \
  'echo $(echo $(echo $(echo $(echo $(sudo ls)))))' \
  'eval eval eval eval eval sudo ls' \
  "$(nested_braces 24)" \
  "$(nested_quoted_braces 24)" \
  "$(nested_quoted_braces 150)"

# Re-read text past the cap (8 times the command plus 64 KB). wrap C is
# echo 'C' "" | sh, which has the guard read C twice (the argument and the
# arguments joined), so four wraps of an 8,000-character command make it
# read about 30 times that, within 4 levels: three wraps pass, four are
# denied. The JSON is escaped with sed (no newline or tab in these).
wrap_for_sh() {
  printf "echo '%s' \"\" | sh" "$(printf '%s' "$1" | sed "s/'/'\\\\''/g")"
}
check_sed_escaped() {
  local label="$1" expect="$2" escaped
  escaped="$(printf '%s' "$3" | sed -e 's/\\/\\\\/g' -e 's/"/\\"/g')"
  if [ "$have_jq" = yes ]; then
    enqueue "$label [jq]" "$expect" "$HOOK" "$real_path" "$(payload_json "$escaped")"
  else
    record_skip "$label [jq] (jq not on PATH)"
  fi
  enqueue "$label [no-jq]" "$expect" "$HOOK" "$minimal_path" "$(payload_json "$escaped")"
}
wrapped="echo $(repeat_text x 8000)"
for k in 1 2 3 4; do
  wrapped="$(wrap_for_sh "$wrapped")"
  if [ "$k" -lt 4 ]; then expect=none; else expect=deny; fi
  check_sed_escaped "F re-read cap: $k nested | sh around an 8,000-character echo (${#wrapped} characters) -> $expect" "$expect" "$wrapped"
done

# Without awk the old guard's four substring rules decide. lib_json.sh
# needs jq here (its jq-absent path runs awk too), so these run with jq on a
# PATH that has no awk, and with an awk that only exits 2.
if [ "$have_jq" = yes ]; then
  noawk_path="$workdir/no-awk-bin"
  fakeawk_path="$workdir/fake-awk-bin"
  mkdir -p "$noawk_path" "$fakeawk_path"
  for tool in sh bash dash cat grep sed printf dirname env tr test jq; do
    resolved="$(command -v "$tool" 2>/dev/null || true)"
    if [ -n "$resolved" ]; then
      ln -sf "$resolved" "$noawk_path/$tool" 2>/dev/null || true
      ln -sf "$resolved" "$fakeawk_path/$tool" 2>/dev/null || true
    fi
  done
  printf '#!/bin/sh\nexit 2\n' > "$fakeawk_path/awk"
  chmod +x "$fakeawk_path/awk"
  if PATH="$noawk_path" "$noawk_path/sh" -c 'command -v awk' >/dev/null 2>&1; then
    record_fail "F. awk is still reachable from the no-awk PATH $noawk_path"
  else
    for c in 'sudo ls' 'git push --force' 'git reset --hard' 'git commit -m "$(id)"' \
      $'echo \'never use sudo here\''; do
      escaped="$(json_escape "$c")"
      enqueue "F. no awk on PATH: $escaped -> deny" deny "$HOOK" "$noawk_path" "$(payload_json "$escaped")"
    done
    enqueue "F. no awk on PATH: ls -> none" none "$HOOK" "$noawk_path" "$(payload_json ls)"
  fi
  for c in 'sudo ls' 'git reset --hard' 'git commit -m "$(id)"'; do
    escaped="$(json_escape "$c")"
    enqueue "F. awk exits 2: $escaped -> deny" deny "$HOOK" "$fakeawk_path" "$(payload_json "$escaped")"
  done
  enqueue "F. awk exits 2: ls -> none" none "$HOOK" "$fakeawk_path" "$(payload_json ls)"
else
  record_skip "F. the awk fallback cases (jq not on PATH: lib_json.sh needs awk or jq)"
fi

# ── G. AC7: the old guard's denies, kept or listed ──────────────────────
# The cases of C that the old guard denies and the new one lets through:
# dangerous text that is not run (single quotes, an argument, a heredoc body
# read as data, the quoted-delimiter heredoc of the recommended commit
# form).
intentional_fixes=(
  $'echo \'never use sudo here\''
  'echo sudo ls'
  'grep sudo file'
  'visudo -c'
  $'git commit -m \'drop --force and git reset --hard from docs\''
  $'git commit -m "$(cat <<\'EOF\'\nfeat: add a thing\n\nBody with `backticks`, $(dollar parens), don\'t, "quotes" and ) parens.\nEOF\n)"'
  $'git commit -F - <<\'EOF\'\ndocs: never git push --force\n$(id) stays as text\nEOF'
  'echo never use sudo here'
  'echo hi # sudo ls'
  $'grep -n \'git push --force\' docs.md'
  $'echo \'sudo ls\' > /dev/null'
  $'echo \'git push --force\' | grep force'
  $'printf \'a\\ngit push --force\''
)
# The new guard's decisions are those of the corpus runs in A, B and C;
# the old guard runs the same payloads (no permission_mode) here.
ac7_paths=(no-jq)
if [ "$have_jq" = yes ]; then
  ac7_paths=(jq no-jq)
else
  record_skip "G. AC7 comparison on the jq path (jq not on PATH)"
fi
corpus_old_jq=()
corpus_old_nojq=()
for ((k = 0; k < ${#corpus[@]}; k++)); do
  escaped="$(json_escape "${corpus[$k]}")"
  for p in "${ac7_paths[@]}"; do
    if [ "$p" = jq ]; then
      enqueue "G. AC7 old [jq]: $escaped" - "$OLD_HOOK" "$real_path" "$(payload_json "$escaped")"
      corpus_old_jq[k]=$last_q
    else
      enqueue "G. AC7 old [no-jq]: $escaped" - "$OLD_HOOK" "$minimal_path" "$(payload_json "$escaped")"
      corpus_old_nojq[k]=$last_q
    fi
  done
done
# The sudo word boundary. The old guard matched "sudo " inside a longer
# name, so it denied my-sudo ls and x.sudo ls; the sentinel's boundary (not
# a letter, digit, _ . or -, the one that lets visudo -c through) does not
# match them, and the new guard lets them through (D, edge_none). This is
# not a data-region exception, so it is outside intentional_fixes and the
# corpus; the old guard's side of the difference is pinned here.
for c in 'my-sudo ls' 'x.sudo ls'; do
  escaped="$(json_escape "$c")"
  for p in "${ac7_paths[@]}"; do
    if [ "$p" = jq ]; then
      enqueue "G. AC7 word boundary: the old guard denies $escaped, the new one does not (D) [jq]" deny "$OLD_HOOK" "$real_path" "$(payload_json "$escaped")"
    else
      enqueue "G. AC7 word boundary: the old guard denies $escaped, the new one does not (D) [no-jq]" deny "$OLD_HOOK" "$minimal_path" "$(payload_json "$escaped")"
    fi
  done
done

# ── Run the queue and check the expected decisions ─────────────────────
run_queue
for ((i = 0; i < qn; i++)); do
  read -r got rc <<< "$(result_of "$i")"
  expect="${q_expect[$i]}"
  ok=no
  if [ "$rc" = 0 ]; then
    case "$expect" in
      none|deny) [ "$got" = "$expect" ] && ok=yes ;;
      any) case "$got" in none|deny) ok=yes ;; esac ;;
      -) case "$got" in none|deny|ask) ok=yes ;; esac ;;
    esac
  fi
  if [ "$expect" = - ]; then
    # Collected for F; only a non-zero exit or unparsed output fails here.
    [ "$ok" = yes ] || record_fail "${q_label[$i]} (got $got, exit $rc)"
    continue
  fi
  if [ "$ok" = yes ]; then
    record_pass "${q_label[$i]}"
  else
    record_fail "${q_label[$i]} (got $got, exit $rc)"
  fi
done

# sorted_lines <item>... — the items, one per line, sorted.
sorted_lines() {
  if [ "$#" -gt 0 ]; then
    printf '%s\n' "$@" | LC_ALL=C sort
  fi
}

expected_fixes="$(for c in "${intentional_fixes[@]}"; do json_escape "$c"; echo; done | LC_ALL=C sort)"
old_denied_ac3=()
for p in "${ac7_paths[@]}"; do
  regressions=()
  for ((k = 0; k < ${#corpus[@]}; k++)); do
    if [ "$p" = jq ]; then
      oi=${corpus_old_jq[$k]}
      ni=${corpus_new_jq[$k]}
    else
      oi=${corpus_old_nojq[$k]}
      ni=${corpus_new_nojq[$k]}
    fi
    read -r old_got _ <<< "$(result_of "$oi")"
    read -r new_got _ <<< "$(result_of "$ni")"
    escaped="$(json_escape "${corpus[$k]}")"
    if [ "$old_got" = deny ] && [ "$new_got" = none ]; then
      regressions+=("$escaped")
    fi
    if [ "$old_got" = deny ] && [ "${corpus_section[$k]}" = C ]; then
      old_denied_ac3+=("$escaped")
    fi
  done
  got_fixes="$(sorted_lines "${regressions[@]+"${regressions[@]}"}")"
  if [ "$got_fixes" = "$expected_fixes" ]; then
    record_pass "G. AC7 [$p]: every old deny that is none now is one of the ${#intentional_fixes[@]} intentional fixes"
  else
    record_fail "G. AC7 [$p]: old deny -> new none differs from intentional_fixes; got: $(printf '%s' "$got_fixes" | tr '\n' '|')"
  fi
done
got_old_ac3="$(sorted_lines "${old_denied_ac3[@]+"${old_denied_ac3[@]}"}" | uniq)"
if [ "$got_old_ac3" = "$expected_fixes" ]; then
  record_pass "G. AC7: intentional_fixes equals the C (AC3) cases the old guard denies"
else
  record_fail "G. AC7: the C cases the old guard denies differ from intentional_fixes; got: $(printf '%s' "$got_old_ac3" | tr '\n' '|')"
fi

# ── H. AC8: a ~200 KB command ───────────────────────────────────────────
# A 200,000-character single-quoted string, then a commit message with a
# command substitution (the payload of the 28 s measurement of the old
# guard); 200 KB of short commands with a force push at the end; and
# 200,000 backslashes inside backticks (the text the backticks unescape,
# which took 2.6 s at this size before the unescaping was made linear).
# The JSON is written directly (each unit of 8 backslashes in the JSON is 4
# in the command) and fed on stdin.
long_cases=(
  "echo '$(repeat_text x 200000)' && git commit -m \\\"\$(id)\\\""
  "$(repeat_text 'echo a b c d; ' 13400)git push --force"
  'echo `echo '"$(repeat_text '\\\\\\\\' 50000)"'`; git push --force'
)
long_names=("a 200,000-character single-quoted string" "13,400 short commands" "200,000 backslashes in backticks")
for ((k = 0; k < ${#long_cases[@]}; k++)); do
  payload_json "${long_cases[$k]}" > "$workdir/long.json"
  TIMEFORMAT=%R
  { time out="$(PATH="$minimal_path" "$HOOK" < "$workdir/long.json" 2>/dev/null)"; } 2> "$workdir/long.time"
  rc=$?
  secs="$(command cat "$workdir/long.time")"
  bytes="$(wc -c < "$workdir/long.json" | tr -d ' ')"
  case "$out" in *'"permissionDecision":"deny"'*) got=deny ;; '') got=none ;; *) got=unparsed ;; esac
  if [ "$rc" -eq 0 ] && [ "$got" = deny ] && awk -v s="$secs" 'BEGIN { exit !(s + 0 < 10) }'; then
    record_pass "H. AC8 [no-jq]: ${long_names[$k]} (${bytes}-byte payload) -> deny in ${secs} s (limit 10 s)"
  else
    record_fail "H. AC8 [no-jq]: ${long_names[$k]} (${bytes}-byte payload): got $got, exit $rc, ${secs} s (limit 10 s)"
  fi
done

# ── I. lib_json.sh sourced directly ─────────────────────────────────────
# lib_case <label> <use_path> <payload> <field> <expected>
lib_case() {
  local label="$1" use_path="$2" payload="$3" field="$4" expect="$5"
  local got rc
  got="$(PATH="$use_path" "$real_sh" -c '. "$1"; extract_json_field "$2" "$3"' sh "$LIB" "$payload" "$field" 2>/dev/null)"
  rc=$?
  if [ "$rc" -eq 0 ] && [ "$got" = "$expect" ]; then
    record_pass "$label"
  else
    record_fail "$label (expected [$expect], got [$got], exit $rc)"
  fi
}

lib_both() {
  local label="$1" payload="$2" field="$3" expect="$4"
  if [ "$have_jq" = yes ]; then
    lib_case "$label [jq]" "$real_path" "$payload" "$field" "$expect"
  else
    record_skip "$label [jq] (jq not on PATH)"
  fi
  lib_case "$label [no-jq]" "$minimal_path" "$payload" "$field" "$expect"
}

lib_both "I. lib_json.sh: tool_input.file_path" \
  '{"session_id":"s","tool_name":"Write","tool_input":{"file_path":"/repo/docs/a b.md","content":"x"},"tool_response":{"filePath":"/repo/docs/a b.md"}}' \
  "tool_input.file_path" "/repo/docs/a b.md"
lib_both "I. lib_json.sh: tool_input.file_path with an escaped quote and backslash" \
  '{"tool_input":{"file_path":"/repo/we\"ird\\name.md","content":"say \"hi\""}}' \
  "tool_input.file_path" '/repo/we"ird\name.md'

echo ""
echo "=== test-pre-bash-guard.sh results ==="
for line in "${results[@]}"; do
  echo "  $line"
done
echo ""
echo "  AC7 intentional false-positive fixes (old guard deny -> none):"
for c in "${intentional_fixes[@]}"; do
  echo "    $(json_escape "$c")"
done
echo ""
echo "  PASS: $pass"
echo "  FAIL: $fail"
echo "  SKIP: $skip"

if [ "$fail" -gt 0 ]; then
  exit 1
fi
exit 0
