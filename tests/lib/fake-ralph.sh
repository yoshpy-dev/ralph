# shellcheck shell=sh
# fake-ralph.sh — a stand-in `ralph` for the shell tests of scripts that read
# settings with scripts/ralph-config.sh (`ralph config get <key>`). Source it
# and put the stand-in's directory first on the PATH of the script under
# test:
#
#   . "$PROJECT_ROOT/tests/lib/fake-ralph.sh"
#   fake_ralph_install "$tmp/bin"
#   fake_ralph_value "$tmp/bin" verify.base HEAD~1
#   fake_ralph_fail "$tmp/bin" standard.base_branch 3 'config get: ralph.toml: ...'
#   PATH="$tmp/bin:$PATH" ./scripts/detect-changed-languages.sh
#   fake_ralph_calls "$tmp/bin"    # what it was asked, from where
#
# The stand-in answers `ralph config get <key>` from the answers set for the
# key: an exit code, and what it prints on stdout and stderr. A key with no
# answer exits 2 with the line ralph writes for a key it does not know. Any
# other call exits 99 with "fake ralph: no answer for: <arguments>" on
# stderr, so a test sees a call it did not expect. After fake_ralph_old,
# every call fails as a ralph older than `config get` does: exit 1 with the
# line it writes, `unknown command "<first argument>" for "ralph"`. To have
# no ralph at all (exit 127 in the script), leave the directory off PATH.
#
# Each call appends a line `<working directory>|<arguments>` to a log that
# fake_ralph_calls prints, the working directory with symbolic links
# resolved (pwd -P). Answers, the log, and the old mode live in
# <bindir>/ralph.fake/, next to the stand-in, which finds them through its
# own path; no environment variable is read, so a script under test that
# starts from a cleared environment still gets them. The functions use
# variables that start with _fr_, since POSIX sh has no local variables.

# fake_ralph_install <bindir>
# Writes the stand-in <bindir>/ralph, making <bindir> if needed, with no
# answer, an empty log, and the old mode off.
fake_ralph_install() {
  mkdir -p "$1/ralph.fake"
  cat > "$1/ralph" <<'EOF'
#!/bin/sh
# A stand-in ralph written by fake_ralph_install (tests/lib/fake-ralph.sh).
state="${0%/*}/ralph.fake"
printf '%s|%s\n' "$(pwd -P)" "$*" >> "$state/calls"
if [ -e "$state/old" ]; then
  printf 'unknown command "%s" for "ralph"\n' "${1-}" >&2
  exit 1
fi
if [ "$#" = 3 ] && [ "$1" = config ] && [ "$2" = get ]; then
  case "$3" in
    '' | . | .. | */*) answer='' ;;
    *) answer="$state/keys/$3" ;;
  esac
  if [ -n "$answer" ] && [ -e "$answer.rc" ]; then
    if [ -e "$answer.out" ]; then cat "$answer.out"; fi
    if [ -e "$answer.err" ]; then cat "$answer.err" >&2; fi
    exit "$(cat "$answer.rc")"
  fi
  printf 'config get: %s: not a key of one value in ralph.toml: no such key\n' "$3" >&2
  exit 2
fi
printf 'fake ralph: no answer for: %s\n' "$*" >&2
exit 99
EOF
  chmod +x "$1/ralph"
  fake_ralph_reset "$1"
}

# fake_ralph_reset <bindir>
# Removes every answer, empties the log, and turns the old mode off.
fake_ralph_reset() {
  rm -rf "$1/ralph.fake/keys" "$1/ralph.fake/old"
  mkdir "$1/ralph.fake/keys"
  : > "$1/ralph.fake/calls"
}

# _fake_ralph_answer <bindir> <key> <exit>
# Replaces the answer for <key> with exit code <exit> and no output.
_fake_ralph_answer() {
  _fr_key="$1/ralph.fake/keys/$2"
  rm -f "$_fr_key.out" "$_fr_key.err"
  printf '%s\n' "$3" > "$_fr_key.rc"
}

# fake_ralph_value <bindir> <key> <value>
# Answers `config get <key>` as ralph does with a value: exit 0, the value
# and a newline on stdout (only the newline for ""), and nothing on stderr.
fake_ralph_value() {
  _fake_ralph_answer "$1" "$2" 0
  printf '%s\n' "$3" > "$_fr_key.out"
}

# fake_ralph_fail <bindir> <key> <exit> [<stderr>]
# Answers `config get <key>` with exit code <exit>, nothing on stdout, and
# <stderr> and a newline on stderr when it is given (it may hold several
# lines).
fake_ralph_fail() {
  _fake_ralph_answer "$1" "$2" "$3"
  if [ "$#" -ge 4 ]; then
    printf '%s\n' "$4" > "$_fr_key.err"
  fi
}

# fake_ralph_old <bindir>
# Makes every call fail as a ralph without `config get` does, until
# fake_ralph_reset.
fake_ralph_old() {
  : > "$1/ralph.fake/old"
}

# fake_ralph_calls <bindir>
# Prints the log, one `<working directory>|<arguments>` line per call since
# the last fake_ralph_reset.
fake_ralph_calls() {
  cat "$1/ralph.fake/calls"
}

# fake_ralph_clear_calls <bindir>
# Empties the log and keeps the answers.
fake_ralph_clear_calls() {
  : > "$1/ralph.fake/calls"
}
