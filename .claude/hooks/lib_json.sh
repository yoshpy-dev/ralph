#!/usr/bin/env sh
# Shared JSON field extraction for hooks.
# Source this file, then call: extract_json_field "$payload" "field_name"
#
# The field argument accepts a dotted path (e.g. "tool_input.file_path")
# to reach nested values. Top-level keys work without a dot.
#
# Uses jq when available. Without jq, sed reads a string value up to its
# closing quote, skipping JSON escapes (so \" inside the value does not end
# it), and an awk program decodes the escapes the way jq does: \" \\ \/ \n
# \t \r \b \f, and \u0000 through \u007F (hex digits in either case) as that
# ASCII character. A real newline in the value (\n in the JSON) and a
# backslash followed by n (\\n in the JSON) therefore come back different,
# as they do with jq. \u0080 and above, surrogate pairs included, stay as
# the six characters written: the guard matches only ASCII, and turning a
# code point into UTF-8 differs between awk dialects. Claude Code sends
# non-ASCII text as raw UTF-8, which passes through unchanged. An unknown
# escape or a lone trailing backslash also stays as written. Both paths
# print the value plus one newline (as jq -r does), so callers using $(...)
# see the same string. The awk program runs under LC_ALL=C and uses POSIX
# awk only, so macOS awk, mawk, and gawk agree.
#
# The fallback expects the payload on one line (hooks pipe it through
# `tr '\n' ' '`) and only reads string values. It matches the leaf key name
# anywhere in the payload (the last occurrence wins), which works for the
# common case of unique key names, but users with ambiguous payloads should
# install jq.

extract_json_field() {
  _payload="$1"
  _field="$2"
  if command -v jq >/dev/null 2>&1; then
    printf '%s' "$_payload" | jq -r ".${_field} // empty" 2>/dev/null
  else
    _leaf="${_field##*.}"
    # Neither jq nor awk: decode only \" and \\ with sed, as before the awk
    # decoder existed. \n and \t stay as two characters, but callers such as
    # the Bash guard still get the command instead of nothing.
    if ! command -v awk >/dev/null 2>&1; then
      printf '%s\n' "$_payload" \
        | sed -n -E "s/.*\"${_leaf}\"[[:space:]]*:[[:space:]]*\"(([^\"\\\\]|\\\\.)*)\".*/\\1/p" \
        | sed -E 's/\\(["\\])/\1/g'
      return 0
    fi
    # The trailing newline makes sed end its output line even for an empty
    # value (GNU and BSD sed omit it when the input lacks one), so awk sees
    # an empty record and prints a newline, as jq -r does for "".
    printf '%s\n' "$_payload" \
      | sed -n -E "s/.*\"${_leaf}\"[[:space:]]*:[[:space:]]*\"(([^\"\\\\]|\\\\.)*)\".*/\\1/p" \
      | LC_ALL=C awk '
        # Each input line is one raw (still escaped) JSON string value.
        # Decode it one character at a time and print it plus a newline.
        #
        # BWK awk (macOS) copies the whole string on every substr() call,
        # so substr(v, i, 1) over a 200 KB value is quadratic. at() reads
        # from a W-character window of v and slides the window when the
        # walk leaves it; emit() buffers the output and flushes it every
        # W characters.
        function at(p) {
          if (p < wb || p >= wb + W) { wb = p; win = substr(v, p, W) }
          return substr(win, p - wb + 1, 1)
        }
        function emit(s) {
          buf = buf s
          if (++nbuf >= W) flush()
        }
        function flush() {
          printf "%s", buf
          buf = ""
          nbuf = 0
        }
        # hex4(p): the four hex digits at p..p+3 as a number, or -1.
        function hex4(p,    k, d, val) {
          if (p + 3 > n) return -1
          val = 0
          for (k = 0; k < 4; k++) {
            d = index("0123456789abcdef", tolower(at(p + k))) - 1
            if (d < 0) return -1
            val = val * 16 + d
          }
          return val
        }
        BEGIN {
          W = 512
          for (k = 1; k < 128; k++) CHR[k] = sprintf("%c", k)
        }
        {
          v = $0
          n = length(v)
          wb = -W
          buf = ""
          nbuf = 0
          i = 1
          while (i <= n) {
            c = at(i)
            # A plain character, or a lone backslash at the end (as written).
            if (c != "\\" || i == n) { emit(c); i++; continue }
            # An escape. An unknown one, and \u0080 and above, keep the
            # backslash and the next character; the rest of the sequence
            # follows as plain characters, so it stays as written.
            e = at(i + 1)
            if (e == "\"" || e == "\\" || e == "/") emit(e)
            else if (e == "n") emit("\n")
            else if (e == "t") emit("\t")
            else if (e == "r") emit("\r")
            else if (e == "b") emit("\b")
            else if (e == "f") emit("\f")
            else if (e == "u" && (u = hex4(i + 2)) >= 0 && u < 128) {
              # A NUL byte cannot live in a BWK awk string; print it directly.
              if (u == 0) { flush(); printf "%c", 0 } else emit(CHR[u])
              i += 6
              continue
            } else emit("\\" e)
            i += 2
          }
          flush()
          printf "\n"
        }'
  fi
}
