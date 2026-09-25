#!/usr/bin/env sh
# secret-scan.sh - shared secret scanner for Ralph git hooks.
#
# CI runs --range with git's default config, so a local --range must read
# the same added lines: scan_range neutralizes every local git setting that
# changes which added lines `git log -p` prints (color, diff.relative,
# textconv, external diff, rename/copy detection, the diff algorithm, the
# big-file threshold, a user-level attributes file, root-commit diffs,
# submodule diffs, replace refs), and checks git's exit status before
# parsing, so a git failure is never read as a clean scan. --staged reads
# each staged blob by its object id, so neither file name quoting nor
# diff.relative can skip a file.
#
# Exit codes:
#   0  scanned, nothing found
#   1  scanned, found something
#   2  usage error
#   3  could not scan (git failed, so the content was not fully read)
set -eu

usage() {
  cat >&2 <<'EOF'
Usage:
  secret-scan.sh --file <path> [label]
  secret-scan.sh --stdin [label]
  secret-scan.sh --staged
  secret-scan.sh --range <rev-range>
  secret-scan.sh --diff [label]
EOF
  exit 2
}

repo_root="$(git rev-parse --show-toplevel 2>/dev/null || pwd)"
allowlist="${RALPH_SECRET_ALLOWLIST:-$repo_root/.gitallowed}"
tmp_dir="${TMPDIR:-/tmp}/ralph-secret-scan.$$"
findings="$tmp_dir/findings"

mkdir -p "$tmp_dir"
trap 'rm -rf "$tmp_dir"' EXIT HUP INT TERM
: > "$findings"

is_allowed() {
  line=$1
  [ -f "$allowlist" ] || return 1

  while IFS= read -r allowed || [ -n "$allowed" ]; do
    case "$allowed" in
      ""|\#*) continue ;;
    esac
    if printf '%s\n' "$line" | grep -Eq "$allowed" 2>/dev/null; then
      return 0
    fi
  done < "$allowlist"

  return 1
}

record_matches() {
  name=$1
  flags=$2
  pattern=$3
  file=$4
  label=$5

  grep "$flags" -n -- "$pattern" "$file" 2>/dev/null | while IFS= read -r match || [ -n "$match" ]; do
    line_no=${match%%:*}
    line=${match#*:}
    if is_allowed "$line"; then
      continue
    fi
    printf '  - %s:%s [%s]\n' "$label" "$line_no" "$name" >> "$findings"
  done
}

scan_file() {
  file=$1
  label=${2:-$file}
  [ -s "$file" ] || return 0

  record_matches "AWS access key id" "-E" "(A3T[A-Z0-9]|AKIA|AGPA|AIDA|AROA|AIPA|ANPA|ANVA|ASIA)[A-Z0-9]{16}" "$file" "$label"
  record_matches "AWS secret assignment" "-iE" "(aws(.{0,20})?)?(secret|access).{0,20}(key|token).{0,20}['\"]?[[:space:]]*[:=][[:space:]]*['\"]?[A-Za-z0-9/+=]{32,}" "$file" "$label"
  bedrock_prefix="$(printf 'bedrock-api-key-%s' 'YmVkcm9jay5hbWF6b25hd3MuY29t')"
  record_matches "Amazon Bedrock API key" "-E" "ABSK[A-Za-z0-9+/]{80,}=*|$bedrock_prefix" "$file" "$label"
  record_matches "GitHub token" "-E" "(ghp|gho|ghs|ghu|ghr)_[A-Za-z0-9_]{30,}|github_pat_[A-Za-z0-9_]{20,}" "$file" "$label"
  record_matches "OpenAI API key" "-E" "sk-(proj|svcacct)?-[A-Za-z0-9_-]{20,}" "$file" "$label"
  record_matches "Slack token" "-E" "xox[abprs]-[A-Za-z0-9-]{20,}" "$file" "$label"
  record_matches "Stripe live secret key" "-E" "sk_live_[A-Za-z0-9]{20,}" "$file" "$label"
  record_matches "private key header" "-E" "BEGIN [A-Z ]*(PRIVATE KEY|RSA PRIVATE|EC PRIVATE|DSA PRIVATE)" "$file" "$label"
  record_matches "generic secret assignment" "-iE" "(api[_-]?key|api[_-]?secret|secret[_-]?key|access[_-]?token|auth[_-]?token|private[_-]?key|client[_-]?secret|password)[[:space:]]*[:=][[:space:]]*['\"]?[^[:space:]'\"]{8,}" "$file" "$label"
}

scan_stdin() {
  label=${1:-stdin}
  file="$tmp_dir/stdin"
  cat > "$file"
  scan_file "$file" "$label"
}

# scan_staged -- scans each staged blob by its object id, so a file name is
# only the finding's label and its quoting cannot decide what gets read:
#   --no-replace-objects       HEAD and each blob as the commit records them
#   diff.relative=false        a subdirectory cwd still lists the whole index
#   core.quotePath=false       non-ASCII names print unquoted
#   --no-renames               a rename or copy lists as a plain addition
#   --diff-filter=d            every change except a deletion, type changes too
# A listed blob that cannot be read exits 3.
scan_staged() {
  staged_list="$tmp_dir/staged-list"
  list_rc=0
  git --no-replace-objects -c diff.relative=false -c core.quotePath=false \
    diff --cached --raw --no-abbrev --no-renames --no-color --diff-filter=d -- \
    > "$staged_list" || list_rc=$?
  if [ "$list_rc" -ne 0 ]; then
    printf 'secret-scan: could not scan the staged changes: git diff --cached exited with %s\n' "$list_rc" >&2
    exit 3
  fi

  tab="$(printf '\t')"
  blob="$tmp_dir/blob"
  # Each line is ":<old mode> <new mode> <old id> <new id> <status>\t<path>".
  # git still C-quotes a path holding a tab, newline, double quote, or
  # backslash, so the line never splits; the path is only a label.
  while IFS="$tab" read -r meta path || [ -n "$meta" ]; do
    [ -n "$meta" ] || continue
    read -r _ new_mode _ new_id _ <<EOF
$meta
EOF
    # A gitlink (submodule) records a commit, not content of this repo.
    case "$new_mode" in
      160000) continue ;;
    esac
    # An all-zero id means no blob is staged (an unmerged path).
    case "$new_id" in
      *[!0]*) ;;
      *) continue ;;
    esac
    if ! git --no-replace-objects cat-file blob "$new_id" > "$blob" 2>/dev/null; then
      printf 'secret-scan: could not scan the staged changes: blob %s for %s cannot be read\n' "$new_id" "$path" >&2
      exit 3
    fi
    scan_file "$blob" "$path"
  done < "$staged_list"
}

# scan_diff_stream [label] -- scans the added lines of a unified diff on
# stdin. ANSI color sequences are removed first: in a colored diff an added
# line starts with an escape sequence, not with "+".
scan_diff_stream() {
  label=${1:-diff}
  diff_file="$tmp_dir/diff"
  added_file="$tmp_dir/diff-added"
  cat > "$diff_file"
  awk '
    { gsub(/\033\[[0-9;:]*m/, "") }
    /^\+\+\+ / { next }
    /^\+/ { sub(/^\+/, ""); print }
  ' "$diff_file" > "$added_file"
  scan_file "$added_file" "$label"
}

# scan_range <rev-range> -- reads the range the way a fresh clone with git's
# default config (CI) does. Each option below neutralizes one local setting
# that changes which added lines `git log -p` prints:
#   --no-replace-objects             refs/replace/* substitutions
#   diff.relative=false              a subdirectory cwd
#   diff.renames=true                renames detected, copies not
#   core.bigFileThreshold=512m       a small threshold turns text into binary
#   core.attributesFile=/dev/null    a user-level attributes file (-diff)
#   log.showRoot=true                a root commit's own diff
#   --no-color                       color.ui / color.diff
#   --no-textconv, --no-ext-diff     diff drivers from the local config
#   --diff-algorithm=default         diff.algorithm
#   --submodule=short                diff.submodule
#   --no-show-signature              log.showSignature
# The output goes to a file, not a pipe, so git's exit status is checked
# before parsing: a git log that fails before or partway through its output
# exits 3 rather than scanning what it printed.
scan_range() {
  range=$1
  log_file="$tmp_dir/range-log"
  log_rc=0
  git --no-replace-objects \
    -c diff.relative=false \
    -c diff.renames=true \
    -c core.bigFileThreshold=512m \
    -c core.attributesFile=/dev/null \
    -c log.showRoot=true \
    log --format='commit %H' --no-color --no-textconv --no-ext-diff \
    --diff-algorithm=default --submodule=short --no-show-signature \
    -p "$range" -- > "$log_file" || log_rc=$?
  if [ "$log_rc" -ne 0 ]; then
    printf 'secret-scan: could not scan %s: git log exited with %s, so the range was not (fully) scanned\n' "$range" "$log_rc" >&2
    exit 3
  fi
  scan_diff_stream "$range" < "$log_file"
}

case "${1:-}" in
  --file)
    [ "$#" -ge 2 ] || usage
    scan_file "$2" "${3:-$2}"
    ;;
  --stdin)
    scan_stdin "${2:-stdin}"
    ;;
  --staged)
    scan_staged
    ;;
  --range)
    [ "$#" -eq 2 ] || usage
    scan_range "$2"
    ;;
  --diff)
    scan_diff_stream "${2:-diff}"
    ;;
  *)
    usage
    ;;
esac

if [ -s "$findings" ]; then
  printf '\n=== ralph secret scan: BLOCKED ===\n' >&2
  printf 'Potential secrets were found:\n' >&2
  sort -u "$findings" >&2
  printf '\nIf this is a false positive, add a narrow regex to .gitallowed.\n' >&2
  printf 'Write the key name as a bracket expression (e.g. api_ke[y]) so the\n' >&2
  printf 'allowlist line itself does not match a scanner pattern -- a branch-\n' >&2
  printf 'history scan also reads the commit that adds the allowlist line.\n' >&2
  exit 1
fi

exit 0
