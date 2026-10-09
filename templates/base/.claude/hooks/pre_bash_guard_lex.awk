# pre_bash_guard_lex.awk holds the "Text access" and "Lexing" sections of
# the awk program of pre_bash_guard.sh. The guard passes its three .awk files
# to one awk with -f, in the order pre_bash_guard_lex.awk,
# pre_bash_guard_commands.awk, pre_bash_guard_rules.awk, and awk reads them
# as one program: functions are shared across the files, and BEGIN and END
# are in pre_bash_guard_rules.awk. The shell no longer passes the program
# in single quotes, so a single quote may appear in it.

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
# quotes), WEXP (1 when it has a $ that expands, see lex_dollar), WANSI (1
# when it has ANSI-C or locale quoting) and WP0/WP1 (where the word starts
# and ends in S), indexed
# [ctx, 1..WN[ctx]]; its redirections are RO (operator), RV/RSB (target
# value and subst flag) and RMISS (no target), indexed [ctx, 1..RN[ctx]].
# XS/XE[ctx, 1..XN[ctx]] are the spans of its redirections and
# substitutions, and POUT[ctx] is 1 when it has a >(...). CUR[ctx] is the
# id of that command. HPQ[ctx, 1..HPN[ctx]] are the heredocs opened in ctx
# whose bodies are not read yet (their numbers h, the index of HDL, HQ and
# the other heredoc arrays, set in lex_redir); read_heredocs reads them at
# the next newline of ctx. The pipeline being assembled in ctx (set in
# end_cmd, reset by pipe_close): PLC[ctx, 1..PLN[ctx]] are the ids of its
# commands that end with |, STC[ctx, 1..STN[ctx]] the ids of its commands
# at the top level, staged for pipe_decide, and PLID[ctx] the id of the
# pipeline (CPL[cid] of each command in it).
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
      else add_word(ctx, LW_VAL, LW_RAW, LW_SUBST, p0, P, LW_EXP, LW_ANSI)
    }
    if (P == p0) P++
  }
  end_cmd(ctx, "end")
  RLVL--
}

# lex_word(ctx): read one word at P. Sets LW_VAL (quotes removed), LW_RAW
# (source text), LW_SUBST (1 when $( or a backtick appears outside single
# quotes), LW_QUOTED (1 when any quote or backslash appears), LW_EXP (1
# when a $ that expands appears, outside quotes or inside double quotes:
# LD_EXP of lex_dollar) and LW_ANSI (1 when ANSI-C or locale quoting
# appears: LD_ANSI). When a $ in the word starts a zsh subscript (LD_SUB,
# also inside double quotes), the span from the first such $ to the end of
# the word is noted with xnote, so it is not part of the argument data
# region of a data command. A message word with a subscript is kept out of
# the data regions by LW_EXP instead, which every subscript $ also sets.
# The word ends where it always does (an unquoted blank or operator), so an
# unclosed [ does not reach past it.
function lex_word(ctx,    start, val, buf, c, c2, e, subst, quoted, bsnl, piece, xp, ansi, subp) {
  start = P
  val = ""
  buf = ""
  subst = 0
  quoted = 0
  bsnl = 0
  xp = 0
  ansi = 0
  subp = 0
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
      if (DQ_EXP) xp = 1
      if (DQ_SUB && !subp) subp = DQ_SUB
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
      if (LD_EXP) xp = 1
      if (LD_ANSI) ansi = 1
      if (LD_SUB && !subp) subp = LD_SUB
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
  if (subp) xnote(ctx, subp, P)
  LW_VAL = val buf
  LW_RAW = text(start, P)
  LW_SUBST = subst
  LW_QUOTED = quoted
  LW_BSNL = bsnl
  LW_EXP = xp
  LW_ANSI = ansi
}

# lex_dq(ctx): P is just after an opening double quote. Reads up to the
# closing one (P is left after it) and returns the value; DQ_SUBST is 1
# when a $( or a backtick appeared, DQ_EXP is 1 when a $ that expands
# appeared (LD_EXP), and DQ_SUB is the position of the first $ that starts
# a zsh subscript (LD_SUB), or 0.
function lex_dq(ctx,    val, buf, e, c, c2, subst, piece, xp, subp) {
  val = ""
  buf = ""
  subst = 0
  xp = 0
  subp = 0
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
        if (LD_EXP) xp = 1
        if (LD_SUB && !subp) subp = LD_SUB
      } else {
        piece = piece lex_bq(ctx)
        subst = 1
      }
    }
    buf = buf piece
    if (length(buf) > 512) { val = val buf; buf = "" }
  }
  DQ_SUBST = subst
  DQ_EXP = xp
  DQ_SUB = subp
  return val buf
}

# lex_dollar(ctx, in_dq): P is at a $. Reads $(...) and ${...}, returning
# their source text, and outside double quotes also the ANSI-C string ($
# and a single quote, returning its value) and $"...". A lone $ is
# returned as itself. LD_SUBST is 1 for $( and for a ${...} that contains
# one or a backtick. Both $(...) and the whole ${...} are noted with xnote,
# so neither is part of the argument data region (a) of a data command: zsh
# re-evaluates the value of ${(e)...}, which runs a $(...) that the lexer
# reads as quoted text (${(e):-SQ$(cmd)SQ}, also with the $ escaped by a
# backslash). LD_SUBST stays 0 for such a ${...}, and only its own span is
# excluded, so the text around it keeps its data region (echo "${HOME}"
# SQsudo lsSQ).
# Three more marks describe the $ just read. This function is called only
# for a $ outside quotes or inside double quotes (from lex_word, lex_dq and
# lex_brace) and for a $ in an unquoted heredoc body (lex_hd), so they
# describe a $ the shell reads: a $ inside single quotes, escaped by a
# backslash, or inside the value of an ANSI-C string never comes here.
#   LD_ANSI is 1 for $SQ...SQ and $DQ...DQ outside double quotes (ANSI-C
#   and locale quoting, which can spell characters the lexer does not
#   decode, such as \x2d for -).
#   LD_EXP is 1 for any other $ that expands: every $ except one followed
#   by the end of the text, a space, a tab, a newline, or (inside double
#   quotes) the closing double quote. It covers $(...), ${...}, $name, $1,
#   $@ and the zsh forms with flags such as $~x, and is set on some $ that
#   only stand for themselves too ($; for one), which only ever drops a
#   data region. msg_check and the rg check of stage_note read it per word.
#   LD_SUB is the position of the $ when the $ starts a zsh subscript, else
#   0: zero or more of the flag characters ~ = ^ + and # (the length of
#   $#x), then either one special parameter character (@ * ? ! $ -) or
#   zero or more name characters, and then [ ($arr[, $~arr[, $#x[, $@[,
#   $*[, and the old arithmetic $[ of bash and zsh; zsh also subscripts $#,
#   $? and the other special parameters, but not $1, which it reads as a
#   glob, so marking it only drops a data region). zsh evaluates the
#   subscript, which runs a $(...) written there in single quotes
#   ($arr[SQ$(cmd)SQ]), while the lexer reads that as quoted text; a
#   subscript opened inside double quotes runs on past the closing quote
#   ("$arr["SQ$(cmd)SQ"]"), so lex_dq reports its $ too (DQ_SUB). lex_word
#   keeps the span from this $ to the end of its word out of the argument
#   data region; the subscript is not skipped here, so lex_word still lexes
#   a $(...) or a backtick inside it.
# All marks are set after the nested lex_cmds or lex_brace call returns, as
# LD_SUBST is, so a $ read inside them does not overwrite the marks of
# this one.
function lex_dollar(ctx, in_dq,    s, c, f, q, v) {
  s = P
  c = at(P + 1)
  if (c == "(") {
    P += 2
    lex_cmds(new_ctx(CD[ctx] + 1), ")")
    xnote(ctx, s, P)
    LD_SUBST = 1
    LD_EXP = 1
    LD_ANSI = 0
    LD_SUB = 0
    return text(s, P)
  }
  if (c == "{") {
    P += 2
    f = lex_brace(ctx, in_dq)
    xnote(ctx, s, P)
    LD_SUBST = f
    LD_EXP = 1
    LD_ANSI = 0
    LD_SUB = 0
    return text(s, P)
  }
  if (!in_dq && (c == SQ || c == DQ)) {
    if (c == SQ) { P += 2; v = lex_ansi() }
    else { P++; v = "" }
    LD_SUBST = 0
    LD_EXP = 0
    LD_ANSI = 1
    LD_SUB = 0
    return v
  }
  LD_SUBST = 0
  LD_ANSI = 0
  LD_EXP = !(c == "" || c == " " || c == "\t" || c == "\n" || (in_dq && c == DQ))
  q = skip(P + 1, RE_NOTFLAG)
  c = at(q)
  if (c != "" && index(SPECIAL_PARAMS, c)) q++
  else q = skip(q, RE_NOTNAME)
  LD_SUB = (at(q) == "[") ? s : 0
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
    add_word(ctx, text(s, P), text(s, P), 1, s, P, 0, 0)
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
    # A delimiter word with a dollar sign or a backtick drops every data
    # region of the command. The shells read most such words as written, as
    # this lexer does: bash, zsh and dash take ${x} and $x literally, and
    # bash and zsh a word in backticks (dash reports a syntax error). Two
    # forms are read differently: an ANSI-C quote with an escape other than
    # \n \t \r, which the lexer reads as the character after the backslash
    # ($SQ\x45SQ is E to bash and zsh, x45 here), and $"...", which the lexer
    # and bash read as the quoted text and zsh and dash as a $ before it.
    # The shell can then end the body on a line before the one the lexer
    # finds, and the lines between are commands. The rule covers every $
    # and backtick, more than those two forms.
    if (index(LW_RAW, "$") || index(LW_RAW, BQ)) NODATA = 1
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
# marks heredoc CUR_H as having a substitution (HSUB). Any ${...} counts,
# with or without a $( inside it the lexer can see: zsh re-evaluates the
# value of ${(e):-\$(cmd)}, so the body is no data region. An escaped \${
# is text.
function lex_hd(ctx,    c) {
  while (P <= N) {
    P = skip(P, RE_HD)
    if (P > N) break
    c = at(P)
    if (c == BS) P += 2
    else if (c == "$") { if (at(P + 1) == "{") HSUB[CUR_H] = 1; lex_dollar(ctx, 1); if (LD_SUBST) HSUB[CUR_H] = 1 }
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

