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
# The awk program below decides in two ways, and either one denies:
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
#      with its name), NAME=value assignments, and the wrappers env (-i,
#      -u NAME, NAME=value), command, exec, nohup, time, nice (-n N), timeout
#      (and its duration), xargs (and its flags), and stdbuf. Names are
#      compared without their directory (/usr/bin/sudo is sudo) and without
#      a leading unquoted = (zsh runs =sudo as sudo). Unless the command is
#      one known not to run its arguments (NOEXEC below: echo, grep, cat,
#      cp, git, ...), its later words count as well: a word that is sudo
#      with another word after it, or a word that is git, is judged as a
#      command there (find -exec sudo, watch sudo, flock /tmp/l git push).
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
#      - --no-verify on git commit, push, merge, rebase, or am; on commit
#        also -n and a short-flag cluster with n before any flag that takes
#        a value (m F C c t; u and S take an attached value), so -nm is
#        denied while -mn (message "n") and -uno are not.
#      - git -c core.hooksPath=... (key in any case; also --config-env)
#        before the subcommand.
#      Long options may be abbreviated, as git allows: an argument that
#      starts with -- and, without its =value, is at least 4 characters and
#      a prefix of --force, --force-with-lease, --hard, --no-verify,
#      --message, or --file counts as that option (--ha is --hard). An
#      abbreviation git rejects as ambiguous is denied too; it fails anyway.
#   5. Fail closed. Nesting deeper than 4 levels, more re-read text than 8
#      times the command plus 64 KB, or $(...) and ${...} inside each other
#      more than 24 levels deep (the command itself counts as one) is denied
#      as too deep. When awk is missing or exits non-zero, the four
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
#      (a) the arguments of a command that only reads data (DATACMD below:
#          echo, printf, cat, grep, ls, ...; rg without --pre, printf
#          without -v), from its first argument to the end of the command,
#          without redirections and substitutions, when the command and
#          every later stage of its pipeline are such commands and send
#          output only to the terminal, the next stage, /dev/null,
#          /dev/stderr, a copy of fd 0, 1 or 2, or a closed fd (>&-) (no
#          file, no fd 3 or above, no &> or &>>, no >(...));
#      (b) the -m or --message value of git commit and git tag when it has
#          no substitution (the recommended heredoc form counts as one);
#      (c) the body of a heredoc whose delimiter is quoted, or whose body
#          has no substitution, fed to a command of (a) or to git commit
#          -F -, under the same conditions as (a);
#      (d) a comment (# where a word starts, to the end of the line).
#      So text the lexer cannot follow (watch 'sudo ls', find -exec sh -c
#      '...', csh -c, source <(...), git rebase -x, a file a script is
#      written to) is still denied, as the previous guard did, while the
#      same words as data (echo 'never use sudo here', a commit message,
#      grep -n 'git push --force' docs.md) pass.
#      The guiding principle: when the guard cannot tell where a read-only
#      command's text ends up, it gives no data region and the previous
#      guard's rules decide. So a group or compound structure at the top
#      level (a subshell (...), a brace group { ...; }, a reserved word such
#      as if, for or case in command position) and an exec with a
#      redirection each drop every data region of the command. A heredoc
#      body joined by a backslash-newline, or read for a delimiter word that
#      has one, is not data (only that body loses its region).
# Not covered: anything only known at run time (variables such as $cmd,
# aliases, functions, git aliases, scripts read from a file, remote commands
# such as ssh host '...'), and shell syntax beyond the above: case
# patterns, arithmetic, arrays, brace expansion ({su,}do), pathname
# expansion (?udo, [s]udo), and the hex and octal escapes of $'...' (only
# \n, \t and \r are decoded). The lexer misses those; the sentinel sees
# only the text as written. Broken input (an unclosed quote or
# parenthesis, a heredoc without its end line) still exits 0, with or
# without a deny.
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
# awk prints the name of the first rule that denies, or nothing. Its exit
# status goes to awk_status (the status of the assignment is that of the
# command substitution, whose last command is awk); a missing or failing
# awk falls back to the previous guard's rules after the program.
awk_status=0
rule="$(printf '%s' "$command" | LC_ALL=C awk '
# ======================================================================
# Text access
# ======================================================================
# S is the text being lexed (the command, or a queued string), N its
# length, P the current position (1-based). BWK awk (macOS) scans the whole
# string on every substr() call, so reading S with substr(S, P, 1) is
# quadratic. at(), find() and skip() read a W-character window of S and
# move it only when the position leaves it; text() copies a span out of
# the window when it fits and out of S otherwise.
function set_text(t) { S = t; N = length(t); P = 1; wb = -W; win = "" }
function win_at(p) { if (p < wb || p >= wb + W) { wb = p; win = substr(S, p, W) } }
function at(p) {
  if (p < 1 || p > N) return ""
  win_at(p)
  return substr(win, p - wb + 1, 1)
}
# text(a, b): S from position a up to, not including, b.
function text(a, b) {
  if (b > N + 1) b = N + 1
  if (b <= a) return ""
  if (a >= wb && b <= wb + W) return substr(win, a - wb + 1, b - a)
  return substr(S, a, b - a)
}
# find(p, c): the position of the first c at or after p, or N + 1.
function find(p, c,    r, k) {
  while (p <= N) {
    win_at(p)
    r = substr(win, p - wb + 1)
    if ((k = index(r, c)) > 0) return p + k - 1
    p = wb + W
  }
  return N + 1
}
# skip(p, re): the position of the first character at or after p that
# matches re (a one-character class), or N + 1.
function skip(p, re,    r) {
  while (p <= N) {
    win_at(p)
    r = substr(win, p - wb + 1)
    if (match(r, re)) return p + RSTART - 1
    p = wb + W
  }
  return N + 1
}
# find_str(p, str): the position of the first str at or after p, or 0. The
# window is moved so that a match across its end is still found.
function find_str(p, str,    L, r, k) {
  L = length(str)
  while (p + L - 1 <= N) {
    win_at(p)
    if (wb + W - p < L) { wb = p; win = substr(S, p, W) }
    r = substr(win, p - wb + 1)
    if ((k = index(r, str)) > 0) return p + k - 1
    if (wb + W > N) return 0
    p = wb + W - L + 1
    wb = p
    win = substr(S, p, W)
  }
  return 0
}

# ======================================================================
# Lexing
# ======================================================================
# A context is one command list: the command itself, the inside of one
# $(...), or one queued string. CD[ctx] is its depth (0 for the command);
# a context deeper than MAXD is denied as too deep. DCTX[ctx] is 1 for the
# top level of the command, the only place with data regions. The words of
# the simple command being assembled are WV (value, quotes removed), WR
# (source text), WS (1 when the source has $( or a backtick outside single
# quotes) and WP0/WP1 (where the word starts and ends in S), indexed
# [ctx, 1..WN[ctx]]; its redirections are RO (operator), RV/RSB (target
# value and subst flag) and RMISS (no target), indexed [ctx, 1..RN[ctx]].
# XS/XE[ctx, 1..XN[ctx]] are the spans of its redirections and
# substitutions, and POUT[ctx] is 1 when it has a >(...). CUR[ctx] is the
# id of that command.
function new_ctx(d) {
  if (d > MAXD) deny("too_deep")
  CTX++
  CD[CTX] = d
  DCTX[CTX] = (MAIN && d == 0)
  WN[CTX] = 0
  RN[CTX] = 0
  XN[CTX] = 0
  POUT[CTX] = 0
  HPN[CTX] = 0
  PLN[CTX] = 0
  STN[CTX] = 0
  PLID[CTX] = ++PLSER
  CUR[CTX] = ++CIDN
  return CTX
}
# xnote(ctx, a, b): S[a, b) is a redirection or a substitution of the
# command being assembled; it is never data.
function xnote(ctx, a, b,    k) {
  if (!DCTX[ctx]) return
  k = ++XN[ctx]
  XS[ctx, k] = a
  XE[ctx, k] = b
}

# lex_cmds(ctx, closer): lex a command list from P. It stops at the end of
# the text or, when closer is ")", after the ")" that closes the $( or <(
# it was called for. Each simple command goes to end_cmd().
function lex_cmds(ctx, closer,    c, c2, depth, p0, e) {
  if (++RLVL > RMAX) deny("too_deep")
  depth = 0
  while (P <= N) {
    p0 = P
    c = at(P)
    if (c == " " || c == "\t") P = skip(P, RE_NOBLANK)
    else if (c == BS && at(P + 1) == "\n") P += 2
    else if (c == "\n") { P++; end_cmd(ctx, "nl"); read_heredocs(ctx) }
    else if (c == "#") {
      # A comment; at the top level it is data.
      e = find(P, "\n")
      if (DCTX[ctx]) add_data(P, e)
      P = e
    } else if (c == ";") {
      P++
      if (at(P) == ";" || at(P) == "&") P++
      end_cmd(ctx, ";")
    } else if (c == "&") {
      c2 = at(P + 1)
      if (c2 == "&") { P += 2; end_cmd(ctx, "&&") }
      else if (c2 == ">") lex_redir(ctx, P)
      else { P++; end_cmd(ctx, "&") }
    } else if (c == "|") {
      c2 = at(P + 1)
      if (c2 == "|") { P += 2; end_cmd(ctx, "||") }
      else { P += (c2 == "&") ? 2 : 1; end_cmd(ctx, "|") }
    } else if (c == "(") { P++; depth++; if (DCTX[ctx]) NODATA = 1; end_cmd(ctx, "(") }
    else if (c == ")") {
      P++
      if (depth > 0) { depth--; end_cmd(ctx, ")") }
      else if (closer == ")") { end_cmd(ctx, "end"); RLVL--; return }
      else { if (DCTX[ctx]) NODATA = 1; end_cmd(ctx, ")") }
    } else if (c == "<" || c == ">") lex_redir(ctx, P)
    else {
      lex_word(ctx)
      c = at(P)
      # An unquoted number right before < or > is the fd of a redirection.
      if ((c == "<" || c == ">") && LW_RAW ~ /^[0-9]+$/) lex_redir(ctx, p0)
      else add_word(ctx, LW_VAL, LW_RAW, LW_SUBST, p0, P)
    }
    if (P == p0) P++
  }
  end_cmd(ctx, "end")
  RLVL--
}

# lex_word(ctx): read one word at P. Sets LW_VAL (quotes removed), LW_RAW
# (source text), LW_SUBST (1 when $( or a backtick appears outside single
# quotes) and LW_QUOTED (1 when any quote or backslash appears).
function lex_word(ctx,    start, val, buf, c, c2, e, subst, quoted, bsnl, piece) {
  start = P
  val = ""
  buf = ""
  subst = 0
  quoted = 0
  bsnl = 0
  while (P <= N) {
    c = at(P)
    if (c == SQ) {
      quoted = 1
      e = find(P + 1, SQ)
      piece = text(P + 1, e)
      P = e + 1
    } else if (c == DQ) {
      quoted = 1
      P++
      piece = lex_dq(ctx)
      if (DQ_SUBST) subst = 1
    } else if (c == BS) {
      quoted = 1
      c2 = at(P + 1)
      if (c2 == "") { piece = BS; P++ }
      else { piece = (c2 == "\n") ? "" : c2; if (c2 == "\n") bsnl = 1; P += 2 }
    } else if (c == "$") {
      c2 = at(P + 1)
      if (c2 == SQ || c2 == DQ) quoted = 1
      piece = lex_dollar(ctx, 0)
      if (LD_SUBST) subst = 1
    } else if (c == BQ) {
      piece = lex_bq(ctx)
      subst = 1
    } else if (index(" \t\n;&|()<>", c)) break
    else {
      e = skip(P, RE_PLAIN)
      piece = text(P, e)
      P = e
    }
    # Pieces gather in buf, which joins val every 512 characters, so a
    # word of many short pieces is not copied once per piece.
    buf = buf piece
    if (length(buf) > 512) { val = val buf; buf = "" }
  }
  LW_VAL = val buf
  LW_RAW = text(start, P)
  LW_SUBST = subst
  LW_QUOTED = quoted
  LW_BSNL = bsnl
}

# lex_dq(ctx): P is just after an opening double quote. Reads up to the
# closing one (P is left after it) and returns the value; DQ_SUBST is 1
# when a $( or a backtick appeared.
function lex_dq(ctx,    val, buf, e, c, c2, subst, piece) {
  val = ""
  buf = ""
  subst = 0
  while (P <= N) {
    e = skip(P, RE_DQ)
    piece = text(P, e)
    P = e
    if (P <= N) {
      c = at(P)
      if (c == DQ) { P++; buf = buf piece; break }
      if (c == BS) {
        c2 = at(P + 1)
        if (c2 == "\n") P += 2
        else if (c2 == "$" || c2 == BQ || c2 == DQ || c2 == BS) { piece = piece c2; P += 2 }
        else { piece = piece BS; P++ }
      } else if (c == "$") {
        piece = piece lex_dollar(ctx, 1)
        if (LD_SUBST) subst = 1
      } else {
        piece = piece lex_bq(ctx)
        subst = 1
      }
    }
    buf = buf piece
    if (length(buf) > 512) { val = val buf; buf = "" }
  }
  DQ_SUBST = subst
  return val buf
}

# lex_dollar(ctx, in_dq): P is at a $. Reads $(...) and ${...}, returning
# their source text, and outside double quotes also the ANSI-C string ($
# and a single quote, returning its value) and $"...". A lone $ is
# returned as itself. LD_SUBST is 1 for $( and for a ${...} that contains
# one or a backtick. (No single quote may appear in this awk program: the
# shell passes it in single quotes.)
function lex_dollar(ctx, in_dq,    s, c, f) {
  s = P
  c = at(P + 1)
  if (c == "(") {
    P += 2
    lex_cmds(new_ctx(CD[ctx] + 1), ")")
    xnote(ctx, s, P)
    LD_SUBST = 1
    return text(s, P)
  }
  if (c == "{") {
    P += 2
    f = lex_brace(ctx, in_dq)
    LD_SUBST = f
    return text(s, P)
  }
  LD_SUBST = 0
  if (!in_dq && c == SQ) { P += 2; return lex_ansi() }
  if (!in_dq && c == DQ) { P++; return "" }
  P++
  return "$"
}

# lex_brace(ctx, in_dq): P is just after ${. Skips to the matching } and
# returns 1 when a $( or a backtick inside it was lexed.
function lex_brace(ctx, in_dq,    c, subst) {
  if (++RLVL > RMAX) deny("too_deep")
  subst = 0
  while (P <= N) {
    P = skip(P, in_dq ? RE_BRACE_DQ : RE_BRACE)
    if (P > N) break
    c = at(P)
    if (c == "}") { P++; break }
    if (c == BS) P += 2
    else if (c == "$") { lex_dollar(ctx, in_dq); if (LD_SUBST) subst = 1 }
    else if (c == BQ) { lex_bq(ctx); subst = 1 }
    else if (c == DQ) { P++; lex_dq(ctx); if (DQ_SUBST) subst = 1 }
    else P = find(P + 1, SQ) + 1
  }
  RLVL--
  return subst
}

# lex_ansi(): P is just after $ and the opening single quote. Returns the
# value of the ANSI-C string (\n, \t and \r decoded, other escapes give the
# character after the backslash) and leaves P after the closing quote.
function lex_ansi(    v, c, e) {
  v = ""
  while (P <= N) {
    e = skip(P, RE_ANSI)
    v = v text(P, e)
    P = e
    if (P > N) break
    if (at(P) == SQ) { P++; break }
    c = at(P + 1)
    if (c == "n") c = "\n"
    else if (c == "t") c = "\t"
    else if (c == "r") c = "\r"
    v = v c
    P += 2
  }
  return v
}

# lex_bq(ctx): P is at a backtick. The text up to the closing backtick,
# with \\ \` and \$ unescaped as the shell does, is queued to be read as
# commands one level deeper. Returns the source text, backticks included.
# The unescaped text gathers in buf like a word in lex_word, so many
# backslashes cost linear time.
function lex_bq(ctx,    s, e, p, c, val, buf, piece) {
  s = P
  p = P + 1
  val = ""
  buf = ""
  while (1) {
    e = skip(p, RE_BQ)
    piece = text(p, e)
    if (e > N || at(e) == BQ) { buf = buf piece; break }
    c = at(e + 1)
    if (c == BS || c == BQ || c == "$") piece = piece c
    else piece = piece BS c
    p = e + 2
    buf = buf piece
    if (length(buf) > 512) { val = val buf; buf = "" }
  }
  queue("c", val buf, CD[ctx] + 1)
  P = (e > N) ? N + 1 : e + 1
  xnote(ctx, s, P)
  return text(s, P)
}

# lex_redir(ctx, rs): P is at < or > (an fd number before it is already
# read; rs is where the redirection starts). Reads the operator and its
# target word. The target of << and <<- is a heredoc delimiter; the body is
# read at the next newline. <(...) and >(...) run commands, so they are
# lexed like $(...) and kept as a word.
function lex_redir(ctx, rs,    c, c2, op, s, k) {
  c = at(P)
  c2 = at(P + 1)
  if ((c == "<" || c == ">") && c2 == "(") {
    s = P
    P += 2
    lex_cmds(new_ctx(CD[ctx] + 1), ")")
    add_word(ctx, text(s, P), text(s, P), 1, s, P)
    xnote(ctx, s, P)
    if (c == ">") POUT[ctx] = 1
    return
  }
  # &> and &>> send both stdout and stderr to a file (never a safe
  # duplication), so they keep their own operator for redir_safe.
  if (c == "&") {
    if (at(P + 2) == ">") { op = "&>>"; P += 3 }
    else { op = "&>"; P += 2 }
  } else if (c == "<") {
    if (c2 == "<") {
      if (at(P + 2) == "<") { op = "<<<"; P += 3 }
      else if (at(P + 2) == "-") { op = "<<-"; P += 3 }
      else { op = "<<"; P += 2 }
    } else if (c2 == "&" || c2 == ">") { op = c c2; P += 2 }
    else { op = "<"; P++ }
  } else if (c2 == ">" || c2 == "&" || c2 == "|") { op = c c2; P += 2 }
  else { op = ">"; P++ }
  P = skip(P, RE_NOBLANK)
  c = at(P)
  k = ++RN[ctx]
  RO[ctx, k] = op
  RMISS[ctx, k] = (c == "" || index("\n;&|()<>", c) > 0)
  RV[ctx, k] = ""
  RSB[ctx, k] = 0
  if (RMISS[ctx, k]) { xnote(ctx, rs, P); return }
  lex_word(ctx)
  RV[ctx, k] = LW_VAL
  RSB[ctx, k] = LW_SUBST
  xnote(ctx, rs, P)
  if (op == "<<" || op == "<<-") {
    HN++
    HDL[HN] = LW_VAL
    # A backslash-newline in the delimiter word is removed before tokenizing
    # (cat <<EO\<newline>F is the unquoted delimiter EOF), so it does not
    # quote the delimiter; such a body gets no data region (HBSNL).
    HQ[HN] = LW_QUOTED && !LW_BSNL
    HBSNL[HN] = LW_BSNL
    HT[HN] = (op == "<<-")
    HO[HN] = CUR[ctx]
    HX[HN] = ctx
    HPQ[ctx, ++HPN[ctx]] = HN
  }
}

# read_heredocs(ctx): P is just after a newline in ctx. Reads the bodies of
# the heredocs opened on the line that ended, in order. Pending heredocs are
# kept per context so a newline inside a $(...), a backtick or a queued
# string does not start the body of a heredoc the enclosing command
# declared: the body begins only at a newline of the declaring context.
function read_heredocs(ctx,    k) {
  for (k = 1; k <= HPN[ctx]; k++) read_body(HPQ[ctx, k])
  HPN[ctx] = 0
}

# read_body(h): the body of heredoc h starts at P and runs up to the first
# line equal to the delimiter (after leading tabs for <<-), or to the end
# of the text. P is left after the delimiter line. At the top level the
# span of the body is kept (HB0/HB1) for the data regions.
function read_body(h,    d, dl, b0, ls, le, t, body, joined, bs, q, cmp, p2) {
  d = HDL[h]
  dl = length(d)
  b0 = P
  joined = 0
  while (1) {
    if (P > N) { body = text(b0, N + 1); ls = N + 1; break }
    ls = P
    le = find(P, "\n")
    # For an unquoted delimiter, a line ending in an odd number of
    # backslashes is joined with the next line before the terminator
    # comparison (bash, zsh). A body with such a join gets no data region.
    if (!HQ[h] && le <= N) {
      bs = 0
      q = le - 1
      while (q >= ls && at(q) == BS) { bs++; q-- }
      if (bs % 2 == 1) {
        joined = 1
        cmp = text(ls, le - 1)
        p2 = le + 1
        while (1) {
          le = find(p2, "\n")
          bs = 0
          q = le - 1
          while (q >= p2 && at(q) == BS) { bs++; q-- }
          if (le <= N && bs % 2 == 1) { cmp = cmp text(p2, le - 1); p2 = le + 1 }
          else { cmp = cmp text(p2, le); break }
        }
        t = HT[h] ? lstrip_tabs(cmp) : cmp
        if (t == d) { body = text(b0, ls); P = le + 1; break }
        P = le + 1
        continue
      }
    }
    t = HT[h] ? skip(ls, RE_NOTAB) : ls
    if (le - t == dl && text(t, le) == d) { body = text(b0, ls); P = le + 1; break }
    P = le + 1
  }
  if (DCTX[HX[h]] && !joined && !HBSNL[h]) { HDZ[h] = 1; HB0[h] = b0; HB1[h] = ls }
  heredoc_done(h, body)
}
# lstrip_tabs(s): s without its leading tabs (for <<- delimiter matching).
function lstrip_tabs(s,    i) {
  i = 1
  while (substr(s, i, 1) == "\t") i++
  return substr(s, i)
}

# heredoc_done(h, body): what the body of heredoc h means for its command.
function heredoc_done(h, body,    cid, ctx, d, k, q0) {
  ctx = HX[h]
  cid = HO[h]
  d = CD[ctx]
  # Fed to a shell that reads stdin, directly or through a pipe: commands.
  if (CSH[cid] || CPSH[cid]) queue("c", body, d + 1)
  # Its command is in a pipeline that is still open: keep the body in case
  # a later command of that pipeline is such a shell.
  else if (CPL[cid] == PLID[ctx] && PLN[ctx] > 0) { k = ++CBN[cid]; CB[cid, k] = body }
  if (!HQ[h]) {
    if (CCF[cid] && (index(body, "$(") || index(body, BQ))) deny("commit_message")
    q0 = QN
    if (index(body, "$") || index(body, BQ)) queue("h", body, d)
    if (QN > q0) QH[QN] = h
  }
}

# lex_hd(ctx): a heredoc body whose delimiter is not quoted. It is data,
# except that $(...), ${...} and backticks in it are expanded; finding one
# marks heredoc CUR_H as having a substitution (HSUB).
function lex_hd(ctx,    c) {
  while (P <= N) {
    P = skip(P, RE_HD)
    if (P > N) break
    c = at(P)
    if (c == BS) P += 2
    else if (c == "$") { lex_dollar(ctx, 1); if (LD_SUBST) HSUB[CUR_H] = 1 }
    else { lex_bq(ctx); HSUB[CUR_H] = 1 }
  }
}

# queue(kind, t, d): read t later at depth d, as commands ("c") or as an
# unquoted heredoc body ("h"). Text deeper than MAXD, or queued text past
# QMAX characters in total, is denied as too deep rather than skipped.
function queue(kind, t, d) {
  if (t == "") return
  if (d > MAXD) deny("too_deep")
  QBYTES += length(t)
  if (QBYTES > QMAX) deny("too_deep")
  QN++
  QK[QN] = kind
  QT[QN] = t
  QD[QN] = d
}

# ======================================================================
# Simple-command assembly
# ======================================================================
function add_word(ctx, v, r, s, p0, p1,    k) {
  k = ++WN[ctx]
  WV[ctx, k] = v
  WR[ctx, k] = r
  WS[ctx, k] = s
  WP0[ctx, k] = p0
  WP1[ctx, k] = p1
}
# wv(ctx, i): the value of word i of the current command, empty past its
# end.
function wv(ctx, i) { return (i >= 1 && i <= WN[ctx]) ? WV[ctx, i] : "" }
# base(v): a command name without its directory. A long word is no command
# name this guard knows, and walking it would be slow.
function base(v,    k) {
  if (length(v) > 256) return ""
  while ((k = index(v, "/")) > 0) v = substr(v, k + 1)
  return v
}
# cname(ctx, i): the command name of word i: base() of its value, after
# dropping a leading = that zsh expands to the path of the command (=sudo
# runs sudo). A quoted or escaped = is not expanded, and then the source
# does not start with =.
function cname(ctx, i,    v) {
  v = WV[ctx, i]
  if (substr(WR[ctx, i], 1, 1) == "=") v = substr(v, 2)
  return base(v)
}
function join_words(ctx, i,    out, buf, n) {
  out = ""
  buf = ""
  n = WN[ctx]
  for (; i <= n; i++) {
    buf = buf WV[ctx, i] (i < n ? " " : "")
    if (length(buf) > 512) { out = out buf; buf = "" }
  }
  return out buf
}

# end_cmd(ctx, sep): the simple command being assembled in ctx ends at the
# separator sep ("nl" for a newline, "end" at the end of the text). A
# command followed by | joins the open pipeline of ctx; any other separator
# after a command closes it. An empty command (as after | and a newline)
# changes nothing, except that ; & ( ) also close the pipeline. At the top
# level each command is a stage (STC) whose data regions are decided when
# its pipeline ends; a pipeline closed by an empty command has no data.
function end_cmd(ctx, sep,    cid) {
  if (WN[ctx] == 0 && RN[ctx] == 0) {
    if (sep != "nl" && sep != "|") pipe_close(ctx)
    return
  }
  # A reserved word in command position or a brace group at the top level is
  # a compound command the lexer cannot follow; it drops every data region.
  if (DCTX[ctx] && WN[ctx] > 0 && (WR[ctx, 1] in RESW)) NODATA = 1
  cid = CUR[ctx]
  judge(ctx, cid, sep)
  # exec with a redirection changes the current shell fds, so what a later
  # command writes may be run; drop every data region.
  if (DCTX[ctx] && EXEC_SEEN && RN[ctx] > 0) NODATA = 1
  if (DCTX[ctx]) { stage_note(ctx, cid); STC[ctx, ++STN[ctx]] = cid }
  if (sep == "|") { PLC[ctx, ++PLN[ctx]] = cid; CPL[cid] = PLID[ctx] }
  else {
    if (DCTX[ctx]) pipe_decide(ctx)
    pipe_close(ctx)
  }
  WN[ctx] = 0
  RN[ctx] = 0
  XN[ctx] = 0
  POUT[ctx] = 0
  CUR[ctx] = ++CIDN
}
function pipe_close(ctx) { PLN[ctx] = 0; STN[ctx] = 0; PLID[ctx] = ++PLSER }

# cmd_pos(ctx): the index of the command-name word, or 0 when there is none
# (or the command runs nothing, as command -v).
function cmd_pos(ctx,    i, n, r, nm) {
  n = WN[ctx]
  i = 1
  EXEC_SEEN = 0
  while (i <= n) {
    r = WR[ctx, i]
    if (r == "if" || r == "then" || r == "else" || r == "elif" || r == "do" || r == "while" || r == "until" || r == "!" || r == "{" || r == "}") { i++; continue }
    if (r == "function") { i += 2; continue }
    if (r ~ /^[A-Za-z_][A-Za-z0-9_]*[+]?=/) { i++; continue }
    nm = cname(ctx, i)
    if (nm == "env") i = skip_env(ctx, i + 1)
    else if (nm == "command") i = skip_command(ctx, i + 1)
    else if (nm == "exec") { EXEC_SEEN = 1; i = skip_opts(ctx, i + 1, "a", "") }
    else if (nm == "nohup") i = skip_opts(ctx, i + 1, "", "")
    else if (nm == "time") i = skip_opts(ctx, i + 1, "fo", " --format --output ")
    else if (nm == "nice") i = skip_opts(ctx, i + 1, "n", " --adjustment ")
    else if (nm == "stdbuf") i = skip_opts(ctx, i + 1, "ioe", " --input --output --error ")
    else if (nm == "timeout") i = skip_opts(ctx, i + 1, "sk", " --signal --kill-after ") + 1
    else if (nm == "xargs") i = skip_xargs(ctx, i + 1)
    else return i
  }
  return 0
}
# takes_next(a, req, opt): for a short-flag cluster a, 1 when its value is
# the next word: the first flag in req (a flag that needs a value) ends the
# cluster, and takes the next word when nothing follows it; a flag in opt
# takes only an attached value.
function takes_next(a, req, opt,    k, n, f) {
  n = length(a)
  if (n > 64) return 0
  for (k = 2; k <= n; k++) {
    f = substr(a, k, 1)
    if (index(req, f)) return k == n
    if (index(opt, f)) return 0
  }
  return 0
}
# skip_opts(ctx, i, req, longreq): skip the options of a wrapper from word
# i and return the index of the first other word. The short flags in req
# and the long options listed in longreq (space-separated, with spaces at
# both ends) take a value; a long option written with = is one word.
function skip_opts(ctx, i, req, longreq,    n, a) {
  n = WN[ctx]
  while (i <= n) {
    a = WV[ctx, i]
    if (a == "--") return i + 1
    if (substr(a, 1, 2) == "--") {
      i += index(longreq, " " a " ") ? 2 : 1
      continue
    }
    if (a ~ /^-[A-Za-z]/) { i += 1 + takes_next(a, req, ""); continue }
    if (a ~ /^-[0-9]/) { i++; continue }
    return i
  }
  return i
}
function skip_env(ctx, i,    n, a) {
  n = WN[ctx]
  while (i <= n) {
    a = WV[ctx, i]
    if (a == "--") return i + 1
    if (a == "-u" || a == "--unset" || a == "-C" || a == "--chdir") { i += 2; continue }
    if (a == "-S" || a == "--split-string") { queue("c", wv(ctx, i + 1), CD[ctx] + 1); i += 2; continue }
    if (substr(a, 1, 15) == "--split-string=") { queue("c", substr(a, 16), CD[ctx] + 1); i++; continue }
    if (substr(a, 1, 1) == "-") { i++; continue }
    if (a ~ /^[A-Za-z_][A-Za-z0-9_]*=/) { i++; continue }
    return i
  }
  return i
}
function skip_command(ctx, i,    n, a) {
  n = WN[ctx]
  while (i <= n) {
    a = WV[ctx, i]
    if (a == "--") return i + 1
    if (a ~ /^-[pvV]+$/) {
      if (a ~ /[vV]/) return n + 1
      i++
      continue
    }
    return i
  }
  return i
}
function skip_xargs(ctx, i,    n, a) {
  n = WN[ctx]
  while (i <= n) {
    a = WV[ctx, i]
    if (a == "--") return i + 1
    if (substr(a, 1, 2) == "--") {
      if (a == "--arg-file" || a == "--delimiter" || a == "--max-args" || a == "--max-procs" || a == "--max-chars" || a == "--process-slot-var") i += 2
      else i++
      continue
    }
    if (a ~ /^-./) { i += 1 + takes_next(a, "ILnPsEdaJRS", "ile"); continue }
    return i
  }
  return i
}

# ======================================================================
# Data regions
# ======================================================================
# Spans DS/DE[1..DN] of the command (at the top level only) that are data:
# a sentinel match inside one is ignored. BKI/BKN index them by 512-character
# block, so a lookup reads only the spans that touch the block of the match.
function add_data(s, e,    k, b, b1) {
  if (e <= s) return
  k = ++DN
  DS[k] = s
  DE[k] = e
  b1 = int((e - 2) / BKW)
  for (b = int((s - 1) / BKW); b <= b1; b++) {
    if (!(b in BKN)) BKN[b] = 0
    BKI[b, ++BKN[b]] = k
  }
}
# in_data(a, b): 1 when S[a, b) lies inside one data span. When NODATA is
# set (a top-level group or compound command, or an exec with a redirection),
# the command has no data region at all and the sentinel rules decide.
function in_data(a, b,    bk, j, k) {
  if (NODATA) return 0
  bk = int((a - 1) / BKW)
  if (!(bk in BKN)) return 0
  for (j = 1; j <= BKN[bk]; j++) {
    k = BKI[bk, j]
    if (DS[k] <= a && b <= DE[k]) return 1
  }
  return 0
}
# redir_safe(op, v, miss): 1 when a redirection keeps the output of its
# command off files. Any input redirection (<, <<, <<-, <<<, <&) is safe. An
# output fd duplication (>&M) is safe only when M is 0, 1, 2 or - (a close):
# >&3 and higher, a non-literal fd, and >&word (a file to bash) are unsafe.
# Output to /dev/null or /dev/stderr is safe; any other file is not. >|, <>,
# &> and &>> never qualify.
function redir_safe(op, v, miss) {
  if (op == "<" || op == "<<" || op == "<<-" || op == "<<<" || op == "<&") return 1
  if (miss) return 0
  if (op == ">&") return v == "0" || v == "1" || v == "2" || v == "-"
  if (op == ">" || op == ">>") return v == "/dev/null" || v == "/dev/stderr"
  return 0
}
# stage_note(ctx, cid): after judge() of a top-level command: whether it
# only reads data (SRO), whether its output stays off files (SSAFE), whether
# it is git commit -F - (SGCF), and the spans of its arguments without its
# redirections and substitutions (CIS/CIE[cid, 1..CIN[cid]]), which become
# data when its pipeline qualifies (pipe_decide).
function stage_note(ctx, cid,    i, n, j, k, ro, safe, rs, re, cur, xs, xe, m, ts, te) {
  i = J_I
  n = WN[ctx]
  ro = (i >= 1 && (J_NM in DATACMD))
  if (ro && J_NM == "rg") for (j = i + 1; j <= n; j++) if (substr(WV[ctx, j], 1, 5) == "--pre") ro = 0
  # printf -v NAME (also attached -vNAME) stores into a variable instead of
  # printing, so its arguments are not read-only data.
  if (ro && J_NM == "printf") for (j = i + 1; j <= n; j++) if (substr(WV[ctx, j], 1, 2) == "-v") ro = 0
  safe = !POUT[ctx]
  for (k = 1; k <= RN[ctx]; k++) if (!redir_safe(RO[ctx, k], RV[ctx, k], RMISS[ctx, k])) safe = 0
  SRO[cid] = ro
  SSAFE[cid] = safe
  SGCF[cid] = (J_NM == "git" && CCF[cid])
  CIN[cid] = 0
  if (!ro || i >= n) return
  # Sort the excluded spans by start (they come nearly sorted: a
  # redirection is noted after the substitution in its target).
  for (k = 2; k <= XN[ctx]; k++) {
    ts = XS[ctx, k]
    te = XE[ctx, k]
    for (j = k - 1; j >= 1 && XS[ctx, j] > ts; j--) { XS[ctx, j + 1] = XS[ctx, j]; XE[ctx, j + 1] = XE[ctx, j] }
    XS[ctx, j + 1] = ts
    XE[ctx, j + 1] = te
  }
  rs = WP0[ctx, i + 1]
  re = WP1[ctx, n]
  cur = rs
  for (k = 1; k <= XN[ctx]; k++) {
    xs = XS[ctx, k]
    xe = XE[ctx, k]
    if (xe <= cur || xs >= re) continue
    if (xs > cur) { m = ++CIN[cid]; CIS[cid, m] = cur; CIE[cid, m] = xs }
    cur = xe
  }
  if (cur < re) { m = ++CIN[cid]; CIS[cid, m] = cur; CIE[cid, m] = re }
}
# pipe_decide(ctx): the top-level pipeline of ctx has ended. From its last
# stage back: a stage whose output stays off files and whose later stages
# all only read data with output off files gets its argument spans as data
# (if it only reads data itself), and CDC (the heredoc bodies it reads may
# be data) when it only reads data or is git commit -F -.
function pipe_decide(ctx,    k, cid, later, ok, j) {
  later = 1
  for (k = STN[ctx]; k >= 1; k--) {
    cid = STC[ctx, k]
    ok = SSAFE[cid] && later
    if (ok && SRO[cid]) for (j = 1; j <= CIN[cid]; j++) add_data(CIS[cid, j], CIE[cid, j])
    CDC[cid] = ok && (SRO[cid] || SGCF[cid])
    later = later && SRO[cid] && SSAFE[cid]
  }
}

# ======================================================================
# Rule judgement
# ======================================================================
# deny(rule): print the rule name and stop.
function deny(rule) { print rule; exit 0 }

# judge(ctx, cid, sep): apply the rules to the simple command cid of ctx.
# J_I and J_NM keep the index and name of its command word for stage_note.
function judge(ctx, cid, sep,    i, nm) {
  J_I = 0
  J_NM = ""
  i = cmd_pos(ctx)
  if (i < 1 || i > WN[ctx]) return
  nm = cname(ctx, i)
  J_I = i
  J_NM = nm
  if (sep == "|") keep_args(ctx, cid, i + 1)
  if (nm == "sudo") deny("sudo")
  if (nm == "sh" || nm == "bash" || nm == "zsh" || nm == "dash" || nm == "ksh") shell_rules(ctx, cid, i + 1)
  else if (nm == "eval") queue("c", join_words(ctx, i + 1), CD[ctx] + 1)
  else if (nm == "git") {
    # Only the git that is the command word gives data regions (messages).
    DATA_OK = DCTX[ctx]
    git_rules(ctx, cid, i + 1, WN[ctx])
    DATA_OK = 0
  }
  if (!(nm in NOEXEC)) scan_words(ctx, cid, i + 1)
}

# scan_words(ctx, cid, i): the command before word i is not in NOEXEC, so
# it may run its arguments (find -exec, watch, flock, chroot, ...). A later
# word that is sudo with another word after it is denied, and a later word
# that is git gets the git rules on the words after it, up to the next git
# word (so each word is read by one git command: linear in the words).
function scan_words(ctx, cid, i,    n, b, g) {
  n = WN[ctx]
  g = 0
  for (; i <= n; i++) {
    b = cname(ctx, i)
    if (b == "sudo" && i < n) deny("sudo")
    if (b == "git") {
      if (g) git_rules(ctx, cid, g + 1, i - 1)
      g = i
    }
  }
  if (g) git_rules(ctx, cid, g + 1, n)
}

# keep_args(ctx, cid, i): a command followed by | keeps its arguments (one
# by one and joined by spaces) in case the pipeline ends in a shell.
function keep_args(ctx, cid, i,    n, k) {
  n = WN[ctx]
  k = 0
  for (; i <= n; i++) CA[cid, ++k] = WV[ctx, i]
  CAN[cid] = k
  if (k > 1) CAJ[cid] = join_words(ctx, n - k + 1)
}

# shell_rules(ctx, cid, i): sh, bash, zsh, dash or ksh with arguments from
# word i. With -c (alone or in a cluster) the command string is read as
# commands. With no script operand (or with -s) the shell reads commands
# from stdin: its heredocs and here-strings, and what the earlier commands
# of its pipeline print, are read as commands.
function shell_rules(ctx, cid, i,    n, a, k, f, cflag, sflag, d, m, j) {
  n = WN[ctx]
  cflag = 0
  sflag = 0
  while (i <= n) {
    a = WV[ctx, i]
    if (a == "--" || a == "-") { i++; break }
    if (a == "--rcfile" || a == "--init-file") { i += 2; continue }
    if (substr(a, 1, 2) == "--") { i++; continue }
    if (a ~ /^[-+][A-Za-z]+$/ && length(a) <= 64) {
      for (k = 2; k <= length(a); k++) {
        f = substr(a, k, 1)
        if (substr(a, 1, 1) == "-" && f == "c") cflag = 1
        if (substr(a, 1, 1) == "-" && f == "s") sflag = 1
        if (f == "o" || f == "O") i++
      }
      i++
      continue
    }
    break
  }
  d = CD[ctx] + 1
  if (cflag) { if (i <= n) queue("c", WV[ctx, i], d); return }
  if (!sflag && i <= n) return
  CSH[cid] = 1
  for (k = 1; k <= RN[ctx]; k++) if (RO[ctx, k] == "<<<") queue("c", RV[ctx, k], d)
  for (k = 1; k <= PLN[ctx]; k++) {
    m = PLC[ctx, k]
    CPSH[m] = 1
    for (j = 1; j <= CAN[m]; j++) queue("c", CA[m, j], d)
    if (CAN[m] > 1) queue("c", CAJ[m], d)
    for (j = 1; j <= CBN[m]; j++) queue("c", CB[m, j], d)
  }
}

# git_rules(ctx, cid, i, n): git with arguments in words i..n. Global
# options come first; the first other word is the subcommand. The rules
# below read the same range.
function git_rules(ctx, cid, i, n,    a, sc) {
  while (i <= n) {
    a = WV[ctx, i]
    if (a == "-c" || a == "--config-env") {
      if (hooks_key(wv(ctx, i + 1))) deny("hooks_path")
      i += 2
      continue
    }
    if (substr(a, 1, 13) == "--config-env=") {
      if (hooks_key(substr(a, 14))) deny("hooks_path")
      i++
      continue
    }
    if (a == "-C" || a == "--git-dir" || a == "--work-tree" || a == "--namespace" || a == "--super-prefix" || a == "--attr-source") { i += 2; continue }
    if (substr(a, 1, 1) == "-") { i++; continue }
    break
  }
  if (i > n) return
  sc = WV[ctx, i]
  if (sc == "push") push_rules(ctx, i + 1, n)
  else if (sc == "reset") reset_rules(ctx, i + 1, n)
  else if (sc == "commit") commit_rules(ctx, cid, i + 1, n)
  else if (sc == "tag") tag_rules(ctx, i + 1, n)
  else if (sc == "merge" || sc == "rebase" || sc == "am") no_verify_rules(ctx, i + 1, n)
}
# hooks_key(kv): 1 when the config key of kv (name or name=value) is
# core.hooksPath; git config keys are case-insensitive.
function hooks_key(kv,    k) {
  k = index(kv, "=")
  if (k) kv = substr(kv, 1, k - 1)
  return tolower(kv) == "core.hookspath"
}
# opt_is(a, full): 1 when the argument a names the long option full. git
# accepts any unique prefix of a long option, so a counts when it starts
# with -- and, without its =value, is at least 4 characters long and a
# prefix of full (--ha is --hard, --force-with=x is --force-with-lease).
function opt_is(a, full,    k) {
  if (substr(a, 1, 2) != "--") return 0
  k = index(a, "=")
  if (k) a = substr(a, 1, k - 1)
  k = length(a)
  return k >= 4 && k <= length(full) && substr(full, 1, k) == a
}
function push_rules(ctx, i, n,    a) {
  for (; i <= n; i++) {
    a = WV[ctx, i]
    if (opt_is(a, "--no-verify")) deny("no_verify")
    if (opt_is(a, "--force") || opt_is(a, "--force-with-lease")) deny("force_push")
    if (a ~ /^-[A-Za-z0-9]+$/ && index(a, "f")) deny("force_push")
    if (substr(a, 1, 1) == "+") deny("force_push")
  }
}
function reset_rules(ctx, i, n) {
  for (; i <= n; i++) if (opt_is(WV[ctx, i], "--hard")) deny("hard_reset")
}
function no_verify_rules(ctx, i, n) {
  for (; i <= n; i++) {
    if (WV[ctx, i] == "--") return
    if (opt_is(WV[ctx, i], "--no-verify")) deny("no_verify")
  }
}

# commit_rules(ctx, cid, i, n): git commit with arguments in words i..n.
function commit_rules(ctx, cid, i, n,    a, la, k, f, v) {
  while (i <= n) {
    a = WV[ctx, i]
    if (a == "--") break
    if (opt_is(a, "--no-verify")) deny("no_verify")
    if (a == "-m") { msg_word(ctx, i + 1, n, 1); i += 2; continue }
    # --message and --file, also abbreviated, with the value attached
    # after = or in the next word.
    if (opt_is(a, "--message")) {
      k = index(a, "=")
      if (k) { msg_attached(ctx, i, substr(a, 1, k), 1); i++ }
      else { msg_word(ctx, i + 1, n, 1); i += 2 }
      continue
    }
    if (opt_is(a, "--file")) {
      k = index(a, "=")
      if (k) { if (substr(a, k + 1) == "-") CCF[cid] = 1; i++ }
      else { if (wv(ctx, i + 1) == "-") CCF[cid] = 1; i += 2 }
      continue
    }
    if (a == "--author" || a == "--date" || a == "--fixup" || a == "--squash" || a == "--template" || a == "--cleanup" || a == "--trailer" || a == "--reuse-message" || a == "--reedit-message" || a == "--pathspec-from-file") { i += 2; continue }
    if (a ~ /^-[^-]/) {
      # A short-flag cluster, read left to right. n is --no-verify. m F C c
      # t take a value (the rest of the cluster, or the next word), and u
      # and S an attached one; either ends the cluster.
      la = length(a)
      for (k = 2; k <= la && k <= 64; k++) {
        f = substr(a, k, 1)
        if (f == "n") deny("no_verify")
        if (index("mFCct", f)) {
          if (k == la) {
            v = wv(ctx, i + 1)
            if (f == "m") msg_word(ctx, i + 1, n, 1)
            i++
          } else {
            v = substr(a, k + 1)
            if (f == "m") msg_attached(ctx, i, substr(a, 1, k), 1)
          }
          if (f == "F" && v == "-") CCF[cid] = 1
          break
        }
        if (f == "u" || f == "S") break
      }
      i++
      continue
    }
    i++
  }
  # A here-string read by -F - is a message too.
  if (CCF[cid]) for (k = 1; k <= RN[ctx]; k++) if (RO[ctx, k] == "<<<" && RSB[ctx, k]) deny("commit_message")
}
# tag_rules(ctx, i, n): git tag; only its -m / --message value matters, as
# a data region when it has no substitution.
function tag_rules(ctx, i, n,    a, la, k, f) {
  while (i <= n) {
    a = WV[ctx, i]
    if (a == "--") break
    if (a == "-m") { msg_word(ctx, i + 1, n, 0); i += 2; continue }
    if (opt_is(a, "--message")) {
      k = index(a, "=")
      if (k) { msg_attached(ctx, i, substr(a, 1, k), 0); i++ }
      else { msg_word(ctx, i + 1, n, 0); i += 2 }
      continue
    }
    if (a ~ /^-[^-]/) {
      # m F u take a value (the rest of the cluster, or the next word).
      la = length(a)
      for (k = 2; k <= la && k <= 64; k++) {
        f = substr(a, k, 1)
        if (index("mFu", f)) {
          if (k < la) { if (f == "m") msg_attached(ctx, i, substr(a, 1, k), 0) }
          else { if (f == "m") msg_word(ctx, i + 1, n, 0); i++ }
          break
        }
      }
      i++
      continue
    }
    i++
  }
}
# msg_word(ctx, j, n, commit): word j (when j <= n) is a -m message given as
# its own word. msg_attached(ctx, i, pre, commit): the message is attached
# to word i after the flag text pre (-m, -am, --message=).
function msg_word(ctx, j, n, commit) {
  if (j <= n) msg_check(ctx, j, WR[ctx, j], commit)
}
function msg_attached(ctx, i, pre, commit,    r) {
  r = WR[ctx, i]
  msg_check(ctx, i, (substr(r, 1, length(pre)) == pre) ? substr(r, length(pre) + 1) : "", commit)
}
# msg_check(ctx, j, raw, commit): raw is the source text of a message in
# word j. A command substitution in it (other than the recommended heredoc
# form) is denied for git commit; a message without one is data when its
# git is the command word.
function msg_check(ctx, j, raw, commit) {
  if (WS[ctx, j] && !safe_heredoc_msg(raw)) {
    if (commit) deny("commit_message")
    return
  }
  if (DATA_OK) add_data(WP0[ctx, j], WP1[ctx, j])
}
# safe_heredoc_msg(r): 1 when the source text r is exactly a double quote,
# $(cat <<, the delimiter D in single quotes (or <<"D", or <<\D), a
# newline, the body, a line that is only D, and ) plus a double quote. The
# first line equal to D ends the body.
function safe_heredoc_msg(r,    q, d, rest, k) {
  if (substr(r, 1, 9) != DQ "$(cat <<") return 0
  rest = substr(r, 10)
  q = substr(rest, 1, 1)
  if (q == BS) {
    if (!match(rest, /^.[A-Za-z0-9_]+/)) return 0
    d = substr(rest, 2, RLENGTH - 1)
    rest = substr(rest, RLENGTH + 1)
  } else if (q == SQ || q == DQ) {
    if (!match(rest, /^.[A-Za-z0-9_]+/)) return 0
    d = substr(rest, 2, RLENGTH - 1)
    if (substr(rest, RLENGTH + 1, 1) != q) return 0
    rest = substr(rest, RLENGTH + 2)
  } else return 0
  if (substr(rest, 1, 1) != "\n") return 0
  rest = substr(rest, 2)
  if (substr(rest, 1, length(d) + 1) == d "\n") k = 1
  else if ((k = index(rest, "\n" d "\n")) > 0) k++
  else return 0
  return substr(rest, k + length(d) + 1) == ")" DQ
}

# ======================================================================
# The sentinel
# ======================================================================
# sentinel(): the four substring rules of the previous guard on the whole
# command (S is the command again). A match inside a data region is
# ignored; any other match denies with the rule of the previous guard.
function sentinel(    p, a, q, t, b) {
  p = 1
  while ((a = find_str(p, "sudo")) > 0) {
    if ((a == 1 || !index(NONB, at(a - 1))) && at(a + 4) != "" && index(SPACES, at(a + 4)) && !in_data(a, a + 4)) deny("sudo")
    p = a + 1
  }
  sentinel_str("git push --force", "force_push")
  sentinel_str("git push -f", "force_push")
  sentinel_str("git reset --hard", "hard_reset")
  # The previous guard read the message after the first "-m " of the
  # command, once git commit came before some "-m ": a double quote, then
  # $( or a backtick anywhere after it. The match runs from that quote to
  # the first $( or backtick.
  a = find_str(1, "git commit")
  if (a > 0 && find_str(a + 10, "-m ") > 0) {
    q = find_str(1, "-m ") + 3
    if (at(q) == DQ) {
      t = find_str(q + 1, "$(")
      b = find(q + 1, BQ)
      if (b <= N && (t == 0 || b < t)) t = b
      if (t > 0 && !in_data(q, t + (at(t) == BQ ? 1 : 2))) deny("commit_message")
    }
  }
}
function sentinel_str(str, rule,    p, a, L) {
  L = length(str)
  p = 1
  while ((a = find_str(p, str)) > 0) {
    if (!in_data(a, a + L)) deny(rule)
    p = a + 1
  }
}

# ======================================================================
# Main
# ======================================================================
BEGIN {
  RS = "\001"
  FS = "\001"
  W = 512
  BKW = 512
  MAXD = 4
  # Nested lex_cmds and lex_brace calls. mawk stops with "eval stack size"
  # near 49 levels of ${...} that alternate with double quotes (99 without
  # the quotes), so the limit stays well below that.
  RMAX = 24
  # Commands known not to run their arguments; any other command has its
  # later words checked by scan_words. git has rules of its own.
  nx = split("echo printf man info whatis apropos which type grep egrep fgrep zgrep rg ag cat less more head tail wc sort cut jq ls test [ cd true false cp mv rm mkdir touch ln chmod stat file diff git", noexec_list, " ")
  for (; nx > 0; nx--) NOEXEC[noexec_list[nx]] = 1
  # Commands that only read data (and print to stdout): their arguments are
  # data regions for the sentinel. sed, awk, man, less, more, sort, tee and
  # jq can run commands or write files, so they are not here.
  nx = split("echo printf cat head tail wc cut tr grep egrep fgrep zgrep rg ls stat diff test [ cd true false which type", datacmd_list, " ")
  for (; nx > 0; nx--) DATACMD[datacmd_list[nx]] = 1
  # Reserved words in command position, and the brace-group words, that make
  # a top-level command a compound one with no data region.
  nx = split("if then elif else fi for while until do done case esac select function { }", resw_list, " ")
  for (; nx > 0; nx--) RESW[resw_list[nx]] = 1
  SQ = sprintf("%c", 39)
  DQ = "\""
  BS = "\\"
  BQ = "`"
  # Characters that make sudo part of a longer name (visudo, my-sudo).
  NONB = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789_.-"
  SPACES = " \t\n\r" sprintf("%c%c", 11, 12)
  RE_NOBLANK = "[^ \t]"
  RE_NOTAB = "[^\t]"
  RE_PLAIN = "[ \t\n;&|()<>" SQ DQ "\\\\$" BQ "]"
  RE_DQ = "[" DQ "\\\\$" BQ "]"
  RE_HD = "[\\\\$" BQ "]"
  RE_BQ = "[\\\\" BQ "]"
  RE_ANSI = "[\\\\" SQ "]"
  RE_BRACE = "[}\\\\$" BQ DQ SQ "]"
  RE_BRACE_DQ = "[}\\\\$" BQ DQ "]"
  CTX = CIDN = PLSER = QN = QBYTES = HN = RLVL = DN = 0
  MAIN = DATA_OK = CUR_H = J_I = NODATA = EXEC_SEEN = 0
  J_NM = ""
}
{ IN = (NR == 1) ? $0 : IN "\001" $0 }
END {
  set_text(IN)
  # Re-read text may grow past the command (pipes feed a shell each
  # argument and their join), but not by more than this.
  QMAX = 8 * N + 65536
  MAIN = 1
  lex_cmds(new_ctx(0), "")
  MAIN = 0
  # Queued text: substitutions, -c strings, eval, what is fed to a shell.
  # A job may queue more; QN is read again on every pass.
  for (qi = 1; qi <= QN; qi++) {
    RLVL = 0
    CUR_H = (QK[qi] == "h") ? QH[qi] : 0
    set_text(QT[qi])
    if (QK[qi] == "c") lex_cmds(new_ctx(QD[qi]), "")
    else lex_hd(new_ctx(QD[qi]))
  }
  # Heredoc bodies of the top level that are data (c). Whether an unquoted
  # body has a substitution is known once its queued read has run.
  for (hi = 1; hi <= HN; hi++) if (HDZ[hi] && CDC[HO[hi]] && (HQ[hi] || !HSUB[hi])) add_data(HB0[hi], HB1[hi])
  set_text(IN)
  sentinel()
}
' 2>/dev/null)" || awk_status=$?

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
    emit_deny "コミットメッセージのダブルクォート内にバッククォートまたは \$() を検出しました。シェルのコマンド置換として解釈され、環境変数やシークレットが漏洩する恐れがあります。代わりにシングルクォートまたは HEREDOC (<<'EOF') を使用してください。"
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
