# pre_bash_guard_rules.awk holds the "Rule judgement", "The sentinel" and
# "Main" sections of the awk program of pre_bash_guard.sh; Main has BEGIN
# (with the command lists NOEXEC and DATACMD), the action that
# collects the input, and END. How awk reads it together with the other two
# .awk files is in the comment above the awk call in pre_bash_guard.sh.

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
  else if (sc == "merge" || sc == "rebase" || sc == "am" || sc == "pull") no_verify_rules(ctx, i + 1, n)
}
# hooks_key(kv): 1 when the config key of kv (name or name=value) is
# core.hooksPath; git config keys are case-insensitive.
function hooks_key(kv,    k) {
  k = index(kv, "=")
  if (k) kv = substr(kv, 1, k - 1)
  return tolower(kv) == "core.hookspath"
}
# opt_is(a, full): 1 when the argument a names the long option full. git
# accepts any unique prefix of a long option, even one letter (git reset
# --h is a hard reset), so a counts when it starts with --, has at least one
# more character before any =value, and is a prefix of full (--h is --hard,
# --force-with=x is --force-with-lease). A prefix that is ambiguous for git
# makes git stop with an error, so matching it too is harmless.
function opt_is(a, full,    k) {
  if (substr(a, 1, 2) != "--") return 0
  k = index(a, "=")
  if (k) a = substr(a, 1, k - 1)
  k = length(a)
  return k >= 3 && k <= length(full) && substr(full, 1, k) == a
}
# push_rules(ctx, i, n): git push. A value-taking option is consumed with its
# value first, so a value that looks like a flag (--push-option=--force) or a
# refspec is not read as one: the long options of push_val_opt (value after =
# or in the next word) and the short -o (--push-option; its value is the rest
# of the cluster, or the next word). Then --force, --force-with-lease, a short
# cluster whose f comes before any o, and a +refspec deny.
function push_rules(ctx, i, n,    a, fo, ff) {
  for (; i <= n; i++) {
    a = WV[ctx, i]
    if (push_val_opt(a)) { if (!index(a, "=")) i++; continue }
    if (opt_is(a, "--no-verify")) deny("no_verify")
    if (opt_is(a, "--force") || opt_is(a, "--force-with-lease")) deny("force_push")
    if (substr(a, 1, 1) == "+") deny("force_push")
    if (a ~ /^-[A-Za-z0-9]+$/) {
      # Read the cluster left to right by position: f (force) before the
      # first o denies; at o the rest of the cluster is its value, and when
      # nothing follows o the next word is the value.
      fo = index(a, "o")
      ff = index(a, "f")
      if (ff > 1 && (fo == 0 || ff < fo)) deny("force_push")
      if (fo > 1 && fo == length(a)) i++
    }
  }
}
# push_val_opt(a): 1 when a is one of the value-taking long options of git
# push, possibly abbreviated as opt_is allows (git push -h on this machine).
function push_val_opt(a) {
  return opt_is(a, "--repo") || opt_is(a, "--receive-pack") || opt_is(a, "--exec") || opt_is(a, "--recurse-submodules") || opt_is(a, "--push-option")
}
# reset_rules(ctx, i, n): git reset with --hard, but everything after -- is a
# pathspec (a file named --hard is not a flag). --pathspec-from-file takes its
# file in the next word unless it has =, so that word (even --) is skipped
# first (git reset --pathspec-from-file -- --hard is a hard reset).
function reset_rules(ctx, i, n,    a) {
  for (; i <= n; i++) {
    a = WV[ctx, i]
    if (opt_is(a, "--pathspec-from-file")) { if (!index(a, "=")) i++; continue }
    if (a == "--") return
    if (opt_is(a, "--hard")) deny("hard_reset")
  }
}
# no_verify_rules(ctx, i, n): git merge, rebase, am and pull. The scan does not
# stop at --, since a value-taking option can take -- as its value (git
# merge -m -- --no-verify still skips the hooks); a later word that only
# looks like the flag (a message --no-verify) is denied as well.
function no_verify_rules(ctx, i, n) {
  for (; i <= n; i++) if (opt_is(WV[ctx, i], "--no-verify")) deny("no_verify")
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
    # Other value-taking long options, also abbreviated: the value is after
    # = or in the next word (even when that word is --).
    if (opt_is(a, "--author") || opt_is(a, "--date") || opt_is(a, "--fixup") || opt_is(a, "--squash") || opt_is(a, "--template") || opt_is(a, "--cleanup") || opt_is(a, "--trailer") || opt_is(a, "--reuse-message") || opt_is(a, "--reedit-message") || opt_is(a, "--pathspec-from-file")) { i += (index(a, "=") ? 1 : 2); continue }
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
# git is the command word. A word with a $ that expands (WEXP, set by
# lex_dollar for the whole word, so also when quotes split the flag, as in
# --message''=${...}, where msg_attached gives an empty raw) is no data
# either: zsh re-evaluates the value of ${(e)...}, and a zsh subscript
# ($arr['$(cmd)']) runs a $(...) written in single quotes. It is not
# denied (git commit -m "${msg}" stays allowed); the sentinel decides. A $
# inside single quotes, escaped by a backslash, or inside an ANSI-C string
# expands nothing, so 'mention ${HOME}' stays data. The recommended
# heredoc form expands nothing either, so it stays data with a ${ in its
# body.
function msg_check(ctx, j, raw, commit) {
  if (WS[ctx, j] && !safe_heredoc_msg(raw)) {
    if (commit) deny("commit_message")
    return
  }
  if (WEXP[ctx, j] && !safe_heredoc_msg(raw)) return
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
  nx = split("echo printf man info whatis apropos which type grep egrep fgrep zgrep rg ag cat less more head tail wc sort cut tr jq ls test [ cd true false cp mv rm mkdir touch ln chmod stat file diff git", noexec_list, " ")
  for (; nx > 0; nx--) NOEXEC[noexec_list[nx]] = 1
  # Wrappers: commands that run the command named after their options (and,
  # for timeout, after its duration). cmd_pos steps past a word whose name is
  # here, each wrapper with its own branch, and takes any other name as the
  # command name. A name here needs its branch in cmd_pos and a branch needs
  # its name here: without either, the wrapper is taken as the command name
  # and the command after it is not read (a D row of
  # tests/test-pre-bash-guard.sh pins each wrapper, and its F section reads
  # this list).
  nx = split("env command exec nohup time nice stdbuf timeout xargs", wrapper_list, " ")
  for (; nx > 0; nx--) WRAPPER[wrapper_list[nx]] = 1
  # Commands that only read data (and print to stdout): their arguments are
  # data regions for the sentinel. sed, awk, man, less, more, sort, tee and
  # jq can run commands or write files, so they are not here; nor are test
  # and [, whose -v in zsh and bash 5 evaluates a subscript such as
  # arr[$(cmd)], nor stat, whose -A NAME in zsh (the zsh/stat module) does
  # the same with the subscript of NAME.
  # No reserved word (if, for, case, {, ...) and no name in WRAPPER may be
  # listed here. For a command that starts with a reserved word, and for an
  # exec with a redirection, the allowlist in end_cmd is what drops the data
  # regions, because their first word is not in this list. Some forms meet
  # the ( or ) rule of lex_cmds: a case clause meets the ) rule as well, by
  # the ) after its pattern, and a subshell meets only the ( rule, by its
  # opening (. A brace group, and an if, for, while, until or select with no
  # ( or ) at the top level, meet only the allowlist.
  # tests/test-pre-bash-guard.sh checks this.
  nx = split("echo printf cat head tail wc cut tr grep egrep fgrep zgrep rg ls diff cd true false which type", datacmd_list, " ")
  for (; nx > 0; nx--) DATACMD[datacmd_list[nx]] = 1
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
  # After a $: the first character that is not a zsh flag character (or the
  # # of $#x), the one-character special parameters, and the first
  # character that is not a name character (lex_dollar, subscripts).
  RE_NOTFLAG = "[^~=^+#]"
  SPECIAL_PARAMS = "@*?!$-"
  RE_NOTNAME = "[^A-Za-z0-9_]"
  CTX = CIDN = PLSER = QN = QBYTES = HN = RLVL = DN = 0
  MAIN = DATA_OK = CUR_H = J_I = NODATA = 0
  J_NM = ""
}
{ IN = (NR == 1) ? $0 : IN "\001" $0 }
END {
  set_text(IN)
  # The shell drops a backslash-newline before it reads the command, also
  # inside double quotes ("$\ newline (cmd)" is a substitution in bash and
  # dash), while the lexer sees the two characters. A command that has one
  # gets no data region, so the sentinel decides as the previous guard did.
  if (index(IN, BS "\n")) NODATA = 1
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
