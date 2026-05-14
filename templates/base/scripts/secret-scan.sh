#!/usr/bin/env sh
# secret-scan.sh - shared secret scanner for Ralph git hooks.
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

scan_staged() {
  staged_paths="$tmp_dir/staged-paths"
  git diff --cached --name-only --diff-filter=ACMR -- > "$staged_paths"

  if [ ! -s "$staged_paths" ]; then
    return 0
  fi

  while IFS= read -r path || [ -n "$path" ]; do
    [ -n "$path" ] || continue
    blob="$tmp_dir/blob"
    if git show ":$path" > "$blob" 2>/dev/null; then
      scan_file "$blob" "$path"
    fi
  done < "$staged_paths"
}

scan_diff_stream() {
  label=${1:-diff}
  diff_file="$tmp_dir/diff"
  added_file="$tmp_dir/diff-added"
  cat > "$diff_file"
  awk '
    /^\+\+\+ / { next }
    /^\+/ { sub(/^\+/, ""); print }
  ' "$diff_file" > "$added_file"
  scan_file "$added_file" "$label"
}

scan_range() {
  range=$1
  git log --format='commit %H' --no-ext-diff -p "$range" -- | scan_diff_stream "$range"
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
  exit 1
fi

exit 0
