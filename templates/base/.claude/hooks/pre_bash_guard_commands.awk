# pre_bash_guard_commands.awk holds the "Simple-command assembly" and
# "Data regions" sections of the awk program of pre_bash_guard.sh. The guard
# passes its three .awk files to one awk with -f, in the order
# pre_bash_guard_lex.awk, pre_bash_guard_commands.awk,
# pre_bash_guard_rules.awk, and awk reads them as one program: functions
# are shared across the files, and BEGIN and END are in
# pre_bash_guard_rules.awk. The shell no longer passes the program in
# single quotes.

# ======================================================================
# Simple-command assembly
# ======================================================================
# add_word(ctx, v, r, s, p0, p1, x, a): add a word to the command being
# assembled: value v, source text r, subst flag s, span p0..p1, and the
# marks x (a $ that expands, LW_EXP) and a (ANSI-C or locale quoting,
# LW_ANSI).
function add_word(ctx, v, r, s, p0, p1, x, a,    k) {
  k = ++WN[ctx]
  WV[ctx, k] = v
  WR[ctx, k] = r
  WS[ctx, k] = s
  WP0[ctx, k] = p0
  WP1[ctx, k] = p1
  WEXP[ctx, k] = x
  WANSI[ctx, k] = a
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
  # The allowlist: a data region exists only when every top-level command is
  # known to be a pure data reader. A command whose first word is not a bare
  # data command or git (an assignment, a wrapper such as env or command, a
  # path, a variable, or any other name), or that has only redirections,
  # drops every data region of the whole command.
  if (DCTX[ctx] && !data_first_ok(ctx)) NODATA = 1
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

# data_first_ok(ctx): 1 when the command being assembled may give a data
# region. Its first word, with quotes removed (the value, so "echo", \echo
# and e\cho are echo as the shell sees them), must be a bare DATACMD name or
# git. A word with a slash does not qualify, even when its basename is a
# DATACMD (/bin/echo and ./echo may be any program); a value that still holds
# a substitution or a variable ($(x), $CMD, ${x:-echo}) does not match; and a
# command of only redirections has no first word. This first word is read
# before assignments and wrappers are skipped, so env, command, nice,
# builtin, exec, sh, an assignment, ! and any other name give 0.
function data_first_ok(ctx,    v) {
  if (WN[ctx] < 1) return 0
  v = WV[ctx, 1]
  if (index(v, "/")) return 0
  return (v == "git" || (v in DATACMD))
}

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
# set the command has no data region at all and the sentinel rules decide. It
# is set when a top-level command has a first word that is not a bare data
# command or git, has only redirections, is a group or compound command, is an
# exec with a redirection, has a heredoc delimiter with a dollar sign or a
# backtick, or when a backslash-newline appears anywhere.
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
  # rg --pre runs a program. A word whose value is only known at run time
  # could be --pre too, so a word with a $ that expands ($x, "$x", ${...}),
  # a substitution, or ANSI-C ($SQ...SQ) or locale ($DQ...DQ) quoting, which
  # can spell characters the lexer does not decode (\x2d is -), also makes
  # rg not a data command. The marks come from lex_dollar, so a $ in single
  # quotes (rg SQfoo$SQ) does not count.
  if (ro && J_NM == "rg") for (j = i + 1; j <= n; j++) if (substr(WV[ctx, j], 1, 5) == "--pre" || WEXP[ctx, j] || WANSI[ctx, j] || WS[ctx, j]) ro = 0
  # printf -v NAME (also attached -vNAME) stores into a variable instead of
  # printing (bash and zsh), and in zsh a %n conversion assigns to, and a
  # numeric one such as %d evaluates, an argument as an arithmetic
  # expression, which runs a subscript such as arr[$(cmd)]. So printf only
  # reads data when no word has -v first and no word as written has a %, a $
  # or a backtick (a format from a variable or in $SQ\x25nSQ could be %n).
  if (ro && J_NM == "printf") for (j = i + 1; j <= n; j++) if (substr(WV[ctx, j], 1, 2) == "-v" || index(WR[ctx, j], "%") || index(WR[ctx, j], "$") || index(WR[ctx, j], BQ)) ro = 0
  safe = !POUT[ctx]
  for (k = 1; k <= RN[ctx]; k++) if (!redir_safe(RO[ctx, k], RV[ctx, k], RMISS[ctx, k])) safe = 0
  SRO[cid] = ro
  SSAFE[cid] = safe
  SGCF[cid] = (J_NM == "git" && CCF[cid])
  CIN[cid] = 0
  if (!ro || i >= n) return
  # Sort the excluded spans by start (they come nearly sorted: a
  # redirection is noted after the substitution in its target, a ${...}
  # after the substitutions inside it, and a subscript span at the end of
  # its word, after the substitutions in that word).
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

