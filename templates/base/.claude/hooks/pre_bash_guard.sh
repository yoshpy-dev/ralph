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
# The awk program below reads the command roughly the way sh does:
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
#      compared without their directory (/usr/bin/sudo is sudo). Unless the
#      command is one known not to run its arguments (NOEXEC below: echo,
#      grep, cat, cp, git, ...), its later words count as well: a word that
#      is sudo with another word after it, or a word that is git, is judged
#      as a command there. This covers find -exec, watch, flock, chroot, and
#      other runners without a list of them.
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
# Not covered: anything only known at run time (variables such as $cmd,
# aliases, functions, git aliases, scripts read from a file, remote commands
# such as ssh host '...'), and shell syntax beyond the above (case
# patterns, arithmetic, arrays). Broken input (an unclosed quote or
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

# ======================================================================
# Lexing
# ======================================================================
# A context is one command list: the command itself, the inside of one
# $(...), or one queued string. CD[ctx] is its depth (0 for the command);
# a context deeper than MAXD is denied as too deep. The words of the
# simple command being assembled are WV (value, quotes removed), WR (source
# text) and WS (1 when the source has $( or a backtick outside single
# quotes), indexed [ctx, 1..WN[ctx]]; its redirections are RO (operator)
# and RV/RSB (target value and subst flag), indexed [ctx, 1..RN[ctx]].
# CUR[ctx] is the id of that command.
function new_ctx(d) {
  if (d > MAXD) deny("too_deep")
  CTX++
  CD[CTX] = d
  WN[CTX] = 0
  RN[CTX] = 0
  PLN[CTX] = 0
  PLID[CTX] = ++PLSER
  CUR[CTX] = ++CIDN
  return CTX
}

# lex_cmds(ctx, closer): lex a command list from P. It stops at the end of
# the text or, when closer is ")", after the ")" that closes the $( or <(
# it was called for. Each simple command goes to end_cmd().
function lex_cmds(ctx, closer,    c, c2, depth, p0) {
  if (++RLVL > RMAX) deny("too_deep")
  depth = 0
  while (P <= N) {
    p0 = P
    c = at(P)
    if (c == " " || c == "\t") P = skip(P, RE_NOBLANK)
    else if (c == BS && at(P + 1) == "\n") P += 2
    else if (c == "\n") { P++; end_cmd(ctx, "nl"); read_heredocs() }
    else if (c == "#") P = find(P, "\n")
    else if (c == ";") {
      P++
      if (at(P) == ";" || at(P) == "&") P++
      end_cmd(ctx, ";")
    } else if (c == "&") {
      c2 = at(P + 1)
      if (c2 == "&") { P += 2; end_cmd(ctx, "&&") }
      else if (c2 == ">") { P++; lex_redir(ctx) }
      else { P++; end_cmd(ctx, "&") }
    } else if (c == "|") {
      c2 = at(P + 1)
      if (c2 == "|") { P += 2; end_cmd(ctx, "||") }
      else { P += (c2 == "&") ? 2 : 1; end_cmd(ctx, "|") }
    } else if (c == "(") { P++; depth++; end_cmd(ctx, "(") }
    else if (c == ")") {
      P++
      if (depth > 0) { depth--; end_cmd(ctx, ")") }
      else if (closer == ")") { end_cmd(ctx, "end"); RLVL--; return }
      else end_cmd(ctx, ")")
    } else if (c == "<" || c == ">") lex_redir(ctx)
    else {
      lex_word(ctx)
      c = at(P)
      # An unquoted number right before < or > is the fd of a redirection.
      if ((c == "<" || c == ">") && LW_RAW ~ /^[0-9]+$/) lex_redir(ctx)
      else add_word(ctx, LW_VAL, LW_RAW, LW_SUBST)
    }
    if (P == p0) P++
  }
  end_cmd(ctx, "end")
  RLVL--
}

# lex_word(ctx): read one word at P. Sets LW_VAL (quotes removed), LW_RAW
# (source text), LW_SUBST (1 when $( or a backtick appears outside single
# quotes) and LW_QUOTED (1 when any quote or backslash appears).
function lex_word(ctx,    start, val, buf, c, c2, e, subst, quoted, piece) {
  start = P
  val = ""
  buf = ""
  subst = 0
  quoted = 0
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
      else { piece = (c2 == "\n") ? "" : c2; P += 2 }
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
function lex_bq(ctx,    s, e) {
  s = P
  e = P + 1
  while (1) {
    e = skip(e, RE_BQ)
    if (e > N || at(e) == BQ) break
    e += 2
  }
  queue("c", unbq(text(s + 1, e)), CD[ctx] + 1)
  P = (e > N) ? N + 1 : e + 1
  return text(s, P)
}
function unbq(t,    out, k, c) {
  out = ""
  while ((k = index(t, BS)) > 0) {
    c = substr(t, k + 1, 1)
    if (c == BS || c == BQ || c == "$") out = out substr(t, 1, k - 1) c
    else out = out substr(t, 1, k + 1)
    t = substr(t, k + 2)
  }
  return out t
}

# lex_redir(ctx): P is at < or > (an fd number before it is already read).
# Reads the operator and its target word. The target of << and <<- is a
# heredoc delimiter; the body is read at the next newline. <(...) and
# >(...) run commands, so they are lexed like $(...) and kept as a word.
function lex_redir(ctx,    c, c2, op, s, k) {
  c = at(P)
  c2 = at(P + 1)
  if (c2 == "(") {
    s = P
    P += 2
    lex_cmds(new_ctx(CD[ctx] + 1), ")")
    add_word(ctx, text(s, P), text(s, P), 1)
    return
  }
  if (c == "<") {
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
  if (c == "" || index("\n;&|()<>", c)) return
  lex_word(ctx)
  k = ++RN[ctx]
  RO[ctx, k] = op
  RV[ctx, k] = LW_VAL
  RSB[ctx, k] = LW_SUBST
  if (op == "<<" || op == "<<-") {
    HN++
    HDL[HN] = LW_VAL
    HQ[HN] = LW_QUOTED
    HT[HN] = (op == "<<-")
    HO[HN] = CUR[ctx]
    HX[HN] = ctx
  }
}

# read_heredocs(): P is just after a newline. Reads the bodies of the
# heredocs opened on the line that ended, in order.
function read_heredocs() {
  while (HR < HN) {
    HR++
    read_body(HR)
  }
}

# read_body(h): the body of heredoc h starts at P and runs up to the first
# line equal to the delimiter (after leading tabs for <<-), or to the end
# of the text. P is left after the delimiter line.
function read_body(h,    d, dl, b0, ls, le, t, body) {
  d = HDL[h]
  dl = length(d)
  b0 = P
  while (1) {
    if (P > N) { body = text(b0, N + 1); break }
    ls = P
    le = find(P, "\n")
    t = HT[h] ? skip(ls, RE_NOTAB) : ls
    if (le - t == dl && text(t, le) == d) { body = text(b0, ls); P = le + 1; break }
    P = le + 1
  }
  heredoc_done(h, body)
}

# heredoc_done(h, body): what the body of heredoc h means for its command.
function heredoc_done(h, body,    cid, ctx, d, k) {
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
    if (index(body, "$") || index(body, BQ)) queue("h", body, d)
  }
}

# lex_hd(ctx): a heredoc body whose delimiter is not quoted. It is data,
# except that $(...), ${...} and backticks in it are expanded.
function lex_hd(ctx,    c) {
  while (P <= N) {
    P = skip(P, RE_HD)
    if (P > N) break
    c = at(P)
    if (c == BS) P += 2
    else if (c == "$") lex_dollar(ctx, 1)
    else lex_bq(ctx)
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
function add_word(ctx, v, r, s,    k) {
  k = ++WN[ctx]
  WV[ctx, k] = v
  WR[ctx, k] = r
  WS[ctx, k] = s
}
# wv/wr/ws(ctx, i): word i of the current command, empty past its end.
function wv(ctx, i) { return (i >= 1 && i <= WN[ctx]) ? WV[ctx, i] : "" }
function wr(ctx, i) { return (i >= 1 && i <= WN[ctx]) ? WR[ctx, i] : "" }
function ws(ctx, i) { return (i >= 1 && i <= WN[ctx]) ? WS[ctx, i] : 0 }
# base(v): a command name without its directory. A long word is no command
# name this guard knows, and walking it would be slow.
function base(v,    k) {
  if (length(v) > 256) return ""
  while ((k = index(v, "/")) > 0) v = substr(v, k + 1)
  return v
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
# changes nothing, except that ; & ( ) also close the pipeline.
function end_cmd(ctx, sep,    cid) {
  if (WN[ctx] == 0 && RN[ctx] == 0) {
    if (sep != "nl" && sep != "|") pipe_close(ctx)
    return
  }
  cid = CUR[ctx]
  judge(ctx, cid, sep)
  if (sep == "|") { PLC[ctx, ++PLN[ctx]] = cid; CPL[cid] = PLID[ctx] }
  else pipe_close(ctx)
  WN[ctx] = 0
  RN[ctx] = 0
  CUR[ctx] = ++CIDN
}
function pipe_close(ctx) { PLN[ctx] = 0; PLID[ctx] = ++PLSER }

# cmd_pos(ctx): the index of the command-name word, or 0 when there is none
# (or the command runs nothing, as command -v).
function cmd_pos(ctx,    i, n, r, nm) {
  n = WN[ctx]
  i = 1
  while (i <= n) {
    r = WR[ctx, i]
    if (r == "if" || r == "then" || r == "else" || r == "elif" || r == "do" || r == "while" || r == "until" || r == "!" || r == "{" || r == "}") { i++; continue }
    if (r == "function") { i += 2; continue }
    if (r ~ /^[A-Za-z_][A-Za-z0-9_]*[+]?=/) { i++; continue }
    nm = base(WV[ctx, i])
    if (nm == "env") i = skip_env(ctx, i + 1)
    else if (nm == "command") i = skip_command(ctx, i + 1)
    else if (nm == "exec") i = skip_opts(ctx, i + 1, "a", "")
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
# Rule judgement
# ======================================================================
# deny(rule): print the rule name and stop.
function deny(rule) { print rule; exit 0 }

# judge(ctx, cid, sep): apply the rules to the simple command cid of ctx.
function judge(ctx, cid, sep,    i, nm) {
  i = cmd_pos(ctx)
  if (i < 1 || i > WN[ctx]) return
  nm = base(WV[ctx, i])
  if (sep == "|") keep_args(ctx, cid, i + 1)
  if (nm == "sudo") deny("sudo")
  if (nm == "sh" || nm == "bash" || nm == "zsh" || nm == "dash" || nm == "ksh") shell_rules(ctx, cid, i + 1)
  else if (nm == "eval") queue("c", join_words(ctx, i + 1), CD[ctx] + 1)
  else if (nm == "git") git_rules(ctx, cid, i + 1, WN[ctx])
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
    b = base(WV[ctx, i])
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
    if (a == "-m") { msg_rule(wr(ctx, i + 1), ws(ctx, i + 1)); i += 2; continue }
    # --message and --file, also abbreviated, with the value attached
    # after = or in the next word.
    if (opt_is(a, "--message")) {
      k = index(a, "=")
      if (k) { msg_attached(ctx, i, substr(a, 1, k)); i++ }
      else { msg_rule(wr(ctx, i + 1), ws(ctx, i + 1)); i += 2 }
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
            if (f == "m") msg_rule(wr(ctx, i + 1), ws(ctx, i + 1))
            i++
          } else {
            v = substr(a, k + 1)
            if (f == "m") msg_attached(ctx, i, substr(a, 1, k))
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
# msg_rule(raw, subst): a message given as its own word.
function msg_rule(raw, subst) {
  if (subst && !safe_heredoc_msg(raw)) deny("commit_message")
}
# msg_attached(ctx, i, pre): a message attached to word i after the flag
# text pre (-m, -am, --message=).
function msg_attached(ctx, i, pre,    r) {
  if (!WS[ctx, i]) return
  r = WR[ctx, i]
  if (substr(r, 1, length(pre)) == pre && safe_heredoc_msg(substr(r, length(pre) + 1))) return
  deny("commit_message")
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
# Main
# ======================================================================
BEGIN {
  RS = "\001"
  FS = "\001"
  W = 512
  MAXD = 4
  # Nested lex_cmds and lex_brace calls. mawk stops with "eval stack size"
  # near 49 levels of ${...} that alternate with double quotes (99 without
  # the quotes), so the limit stays well below that.
  RMAX = 24
  # Commands known not to run their arguments; any other command has its
  # later words checked by scan_words. git has rules of its own.
  nx = split("echo printf man info whatis apropos which type grep egrep fgrep zgrep rg ag cat less more head tail wc sort cut jq ls test [ cd true false cp mv rm mkdir touch ln chmod stat file diff git", noexec_list, " ")
  for (; nx > 0; nx--) NOEXEC[noexec_list[nx]] = 1
  SQ = sprintf("%c", 39)
  DQ = "\""
  BS = "\\"
  BQ = "`"
  RE_NOBLANK = "[^ \t]"
  RE_NOTAB = "[^\t]"
  RE_PLAIN = "[ \t\n;&|()<>" SQ DQ "\\\\$" BQ "]"
  RE_DQ = "[" DQ "\\\\$" BQ "]"
  RE_HD = "[\\\\$" BQ "]"
  RE_BQ = "[\\\\" BQ "]"
  RE_ANSI = "[\\\\" SQ "]"
  RE_BRACE = "[}\\\\$" BQ DQ SQ "]"
  RE_BRACE_DQ = "[}\\\\$" BQ DQ "]"
  CTX = CIDN = PLSER = QN = QBYTES = HN = HR = RLVL = 0
}
{ IN = (NR == 1) ? $0 : IN "\001" $0 }
END {
  set_text(IN)
  # Re-read text may grow past the command (pipes feed a shell each
  # argument and their join), but not by more than this.
  QMAX = 8 * N + 65536
  lex_cmds(new_ctx(0), "")
  # Queued text: substitutions, -c strings, eval, what is fed to a shell.
  # A job may queue more; QN is read again on every pass.
  for (qi = 1; qi <= QN; qi++) {
    HR = HN
    RLVL = 0
    set_text(QT[qi])
    if (QK[qi] == "c") lex_cmds(new_ctx(QD[qi]), "")
    else lex_hd(new_ctx(QD[qi]))
  }
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
