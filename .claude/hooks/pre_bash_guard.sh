#!/usr/bin/env sh
# pre_bash_guard.sh — PreToolUse guard for the Bash tool (Claude Code and
# Codex). It only ever denies: it never asks for confirmation and never
# reads permission_mode, so it behaves the same in every permission mode.
# A command it does not deny gets no output at all.
#
# Real PreToolUse payloads nest the command under .tool_input.command.
# extract_json_field (lib_json.sh) returns the decoded JSON string on both of
# its paths (jq, and the sed + awk fallback when jq is absent), so a newline
# or a tab in the command is a real newline or tab here.
#
# The awk program (three .awk files next to this script; see the comment
# above the awk call below) decides in two ways, and either one denies:
#   1. Lexing. Words keep their value with quotes removed: single quotes,
#      double quotes, and backslashes work as in sh, and quoted and unquoted
#      parts that touch form one word ("--force" and --for""ce are both
#      --force). Unquoted ; && || | & ( ) and newlines separate simple
#      commands. Redirections (with an fd number: 2>/dev/null, 2>&1, &>file)
#      are set aside. Heredoc bodies are data; when the delimiter is not
#      quoted, the $(...) and backticks in the body are expanded, so they
#      are read as commands.
#   2. Re-reading. Text that the shell runs is read again as commands, up to
#      4 levels deep: $(...), <(...) and backticks (outside quotes and inside
#      double quotes), the string given to sh/bash/zsh/dash/ksh with -c (also
#      in a cluster such as -lc) and to env -S, the arguments of eval, the
#      body of a heredoc or here-string fed to a shell that reads stdin, and
#      the arguments and heredoc bodies of the commands piped into such a
#      shell (| sh, | bash -s). Single-quoted text elsewhere is never read as
#      a command.
#   3. Simple-command assembly. The command name is the first word after
#      reserved words (if then else elif do while until ! { }, and function
#      with its name), NAME=value assignments, and the wrappers (WRAPPER in
#      pre_bash_guard_rules.awk) env (-i, -u NAME, NAME=value), command,
#      exec, nohup, time, nice (-n N), timeout (and its duration), xargs (and
#      its flags), and stdbuf. Names are
#      compared without their directory (/usr/bin/sudo is sudo) and without
#      a leading unquoted = (zsh runs =sudo as sudo). Unless the command is
#      one known not to run its arguments (NOEXEC in
#      pre_bash_guard_rules.awk: echo, grep, cat, cp, git, ...), its later
#      words count as well: a word that is sudo with another word after it,
#      or a word that is git, is judged as a command there (find -exec sudo,
#      watch sudo, flock /tmp/l git push).
#   4. Rules (every permission mode):
#      - sudo as the command name, or as a later word as in 3.
#      - git push (after git's -C <dir>, -c <k=v>, --git-dir=... and other
#        global options) with --force, --force-with-lease[=...], a short-flag
#        cluster containing f, or a +refspec.
#      - git reset with --hard.
#      - git commit whose message (-m, --message, --message=, the m of a
#        short-flag cluster, attached -m"...") has $( or a backtick outside
#        single quotes in its source text. The one form allowed is
#        "$(cat <<'EOF' + newline + body + a line that is only EOF + )"
#        (also <<"EOF" and <<\EOF): a quoted delimiter expands nothing.
#        git commit -F - (or --file=-) fed a heredoc whose delimiter is not
#        quoted and whose body has $( or a backtick is denied too.
#      - --no-verify on git commit, push, merge, rebase, am, or pull; on commit
#        also -n and a short-flag cluster with n before any flag that takes
#        a value (m F C c t; u and S take an attached value), so -nm is
#        denied while -mn (message "n") and -uno are not.
#      - git -c core.hooksPath=... (key in any case; also --config-env)
#        before the subcommand.
#      Long options may be abbreviated, as git allows: an argument that
#      starts with -- and, without its =value, has at least one more
#      character and is a prefix of --force, --force-with-lease, --hard,
#      --no-verify, --message, or --file counts as that option (--h is
#      --hard). An abbreviation git rejects as ambiguous is denied too; it
#      fails anyway.
#   5. Fail closed. Nesting deeper than 4 levels, more re-read text than 8
#      times the command plus 64 KB, or $(...) and ${...} inside each other
#      more than 24 levels deep (the command itself counts as one) is denied
#      as too deep. When awk is missing or exits non-zero (as it does when
#      one of the three .awk files is missing or unreadable), the four
#      substring rules of the previous guard decide instead: "sudo "
#      anywhere, git push --force or -f, git reset --hard, and a git commit
#      -m message that starts with a double quote and holds $( or a
#      backtick.
#   6. The sentinel. The four substring rules of the previous guard also
#      run on the whole command: sudo between a word boundary (not a letter,
#      digit, _ . or -) and a space, tab or newline; git push --force and
#      git push -f; git reset --hard; and the first -m " after git commit
#      followed by $( or a backtick. A match denies unless it lies inside a
#      data region, which only the top level of the command has (nothing
#      inside $(...), backticks, or re-read text is data):
#      (a) the arguments of a command that only reads data (DATACMD in
#          pre_bash_guard_rules.awk: echo, printf, cat, grep, ls, ...; not
#          test or [, nor stat, whose zsh -A NAME evaluates the subscript
#          of NAME; rg only without --pre and without a word that has a
#          substitution, a $ that expands, or ANSI-C or locale quoting,
#          since its value could be --pre; printf only without -v and with
#          no %, $ or backtick in any word as written), from its first
#          argument to the end of the command, without redirections,
#          substitutions, ${...} (zsh evaluates the value of ${(e)...}
#          again, which runs a $(...) written there in single quotes or
#          with an escaped $; the text around a ${...} stays data), and a
#          zsh subscript from its $ to the end of its word ($arr[...],
#          $~arr[...], $#x[...] and $@[...] in zsh, and the arithmetic
#          $[...] in bash and zsh, evaluate the text in the brackets, which
#          runs a $(...) written there in single quotes),
#          when the command and every later stage of its pipeline are such
#          commands and send output only to the terminal, the next stage,
#          /dev/null, /dev/stderr, a copy of fd 0, 1 or 2, or a closed fd
#          (>&-) (no file, no fd 3 or above, no &> or &>>, no >(...));
#      (b) the -m or --message value of git commit and git tag when it has
#          no substitution and no $ that expands (one outside single quotes
#          and ANSI-C strings, not escaped, not the $ of a locale string
#          $"..." outside double quotes, and not followed by a space, a
#          tab, a newline, the end, or a closing double quote; the
#          recommended heredoc form counts as one, even with a ${ in its
#          body); a value with such a $ is not denied for that, the other
#          rules decide;
#      (c) the body of a heredoc whose delimiter is quoted, or whose body
#          has no substitution and no ${...}, fed to a command of (a) or to
#          git commit -F -, under the same conditions as (a);
#      (d) a comment (# where a word starts, to the end of the line).
#      So text the lexer cannot follow (watch 'sudo ls', find -exec sh -c
#      '...', csh -c, source <(...), git rebase -x, a file a script is
#      written to) is still denied, as the previous guard did, while the
#      same words as data (echo 'never use sudo here', a commit message,
#      grep -n 'git push --force' docs.md) pass.
#      The guiding principle is an allowlist: a data region exists only when
#      every top-level simple command is one the guard can see is a pure data
#      reader. So the command gets data regions only when every top-level
#      command's first word, with quotes removed, is a bare DATACMD name or
#      git, with no slash in it (/bin/echo and ./echo may be any program),
#      and none of the other NODATA triggers fire. Any other first word (an
#      assignment, a wrapper such as env, command, sh, nice, builtin or exec,
#      a path, a variable or a substitution) or a command of only redirections
#      drops every data region of the whole command, and the previous guard's
#      rules decide. This also covers a compound command at the top level
#      (its first word is a reserved word such as if, for or case, or the {
#      of a brace group { ...; }) and an exec with a redirection, since their
#      first word is neither a data command nor git. The other NODATA
#      triggers are a ( or ) at the top level (a subshell (...)), a heredoc
#      whose delimiter word has a dollar sign or a backtick (the lexer may
#      read it differently from the shell), and a backslash-newline anywhere
#      in the command (bash and dash remove it before reading, even inside
#      double quotes). The backslash-newline rule does not look at quoting,
#      so one that is only text (inside single quotes, at the end of a line
#      of a quoted heredoc body or a comment, after an escaped backslash)
#      also drops them, as the previous guard denied those commands too. A
#      commit message passed in the recommended heredoc form therefore passes
#      only when every command in the same Bash call starts with a data
#      command or git (in practice: run git commit as a command of its own)
#      and no line of the call ends in a backslash.
# Not covered: anything only known at run time (variables such as $cmd,
# aliases, functions, git aliases, scripts read from a file, remote commands
# such as ssh host '...'), and shell syntax beyond the above: case
# patterns, arithmetic, arrays (beyond keeping a zsh subscript out of the
# data regions), brace expansion ({su,}do), pathname
# expansion (?udo, [s]udo), and the hex and octal escapes of $'...' (only
# \n, \t and \r are decoded). The lexer misses those; the sentinel sees
# only the text as written. The guard reads the words of a printf as written
# (item 6(a)), so a printf with a $'...' word, which holds a $, is no data
# command. Shell state set up by an earlier command is invisible the same
# way a variable is: a function named like a data command, or an exec
# redirection done in an earlier command, is not seen when the next
# command is judged. So are shell options set in start-up files (with zsh
# cdablevars, cd looks a non-directory argument up as a variable). Broken
# input (an unclosed quote or parenthesis, a heredoc without its end line)
# still exits 0, with or without a deny.
set -eu

HOOK_DIR="$(cd "$(dirname "$0")" && pwd)"
. "$HOOK_DIR/lib_json.sh"

payload="$(cat | tr '\n' ' ')"
command="$(extract_json_field "$payload" "tool_input.command")"

emit_deny() {
  escaped="$(printf '%s' "$1" | sed 's/"/\\\"/g')"
  printf '{"hookSpecificOutput":{"hookEventName":"PreToolUse","permissionDecision":"deny","permissionDecisionReason":"%s"}}\n' "$escaped"
  exit 0
}

[ -n "$command" ] || exit 0

# The command goes to awk on stdin (awk -v would process its backslashes).
# awk reads the program from the three .awk files next to this script, with
# -f in the order lex, commands, rules, as one program whose functions and
# globals are shared across the files: pre_bash_guard_lex.awk (text access
# and lexing), pre_bash_guard_commands.awk (simple-command assembly and data
# regions) and pre_bash_guard_rules.awk (rule judgement, the sentinel, BEGIN
# with the command lists NOEXEC, WRAPPER and DATACMD, and END). This script,
# lib_json.sh and the three .awk files must be installed together. awk
# prints the name of the first rule that denies, or nothing. Its exit
# status goes to awk_status (the status of the assignment is that of the
# command substitution, whose last command is awk). A missing or failing
# awk, or a missing or unreadable .awk file (awk then exits non-zero),
# falls back to the previous guard's rules below.
awk_status=0
rule="$(printf '%s' "$command" | LC_ALL=C awk -f "$HOOK_DIR/pre_bash_guard_lex.awk" -f "$HOOK_DIR/pre_bash_guard_commands.awk" -f "$HOOK_DIR/pre_bash_guard_rules.awk" 2>/dev/null)" || awk_status=$?

# awk is missing or failed: decide with the substring rules of the previous
# guard, so the four classic denies still hold. They match text anywhere in
# the command, quoted or not, so this path also denies some harmless
# commands (echo "never use sudo here").
if [ "$awk_status" -ne 0 ]; then
  case "$command" in
    *"sudo "*) rule=sudo ;;
    *"git push --force"*|*"git push -f"*) rule=force_push ;;
    *"git reset --hard"*) rule=hard_reset ;;
    *"git commit"*"-m "*)
      case "${command#*-m }" in
        '"'*'`'*|'"'*'$('*) rule=commit_message ;;
      esac
      ;;
  esac
fi

case "$rule" in
  sudo)
    emit_deny "Avoid sudo inside the harness. Use project-local commands or escalate to a human only if truly necessary."
    ;;
  force_push)
    emit_deny "Force push is blocked by the scaffold."
    ;;
  hard_reset)
    emit_deny "Hard reset is blocked by the scaffold."
    ;;
  commit_message)
    emit_deny "コミットメッセージのダブルクォート内にバッククォートまたは \$() を検出しました。シェルのコマンド置換として解釈され、環境変数やシークレットが漏洩する恐れがあります。代わりにシングルクォートまたは HEREDOC (<<'EOF') を使用してください。HEREDOC の形は git commit を単独のコマンドで打ったときだけ通ります (同じ呼び出しに make や ./scripts/ などのコマンドを並べると止まります)。"
    ;;
  no_verify)
    emit_deny "--no-verify (and -n on git commit) skips the git hooks and is blocked by the scaffold. Fix what the hook reports instead."
    ;;
  hooks_path)
    emit_deny "git -c core.hooksPath=... replaces the repository's git hooks and is blocked by the scaffold."
    ;;
  too_deep)
    emit_deny "The command is nested too deeply for the guard to check (more than 4 levels of \$(...), backticks, sh -c, eval, or input fed to a shell, or too much text to re-read), so it is blocked. Split it into simpler commands."
    ;;
esac

exit 0
