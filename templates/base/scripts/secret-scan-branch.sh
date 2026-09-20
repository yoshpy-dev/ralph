#!/usr/bin/env sh
# secret-scan-branch.sh - scan this branch's committed history for secrets,
# the same way CI does: .github/workflows/verify.yml scans
# merge-base(HEAD, origin/<base>)..HEAD with `git log -p` (scripts/secret-scan.sh
# --range). Editing away a leaked line in a later commit does not clear a
# finding here or in CI -- the range scan still reads the commit that
# introduced it.
#
# Usage: secret-scan-branch.sh [--strict]
#
# Exit codes:
#   0  nothing to scan, or scanned and found nothing
#   1  scanned and found something
#   2  usage error
#   3  (--strict only) could not determine what to scan
#
# Default mode never fails on an unscannable state (no base ref, no
# merge-base, ...): it is meant for run-verify.sh, an early-warning gate
# that must not block on environment gaps such as a repo without a fetched
# base branch. --strict is for a caller that needs a real answer before an
# irreversible step (the /pr skill, right before push): "could not check"
# (exit 3) is kept apart from "checked, clean" (exit 0).
set -eu

usage() {
  cat >&2 <<'EOF'
Usage: secret-scan-branch.sh [--strict]
EOF
  exit 2
}

strict=0
case $# in
  0) ;;
  1)
    case "$1" in
      --strict) strict=1 ;;
      *) usage ;;
    esac
    ;;
  *) usage ;;
esac

script_dir="$(CDPATH='' cd -- "$(dirname -- "$0")" && pwd)"
scanner="$script_dir/secret-scan.sh"

tmp_allowlist=""
cleanup() {
  [ -n "$tmp_allowlist" ] && rm -f "$tmp_allowlist"
  return 0
}
trap cleanup EXIT HUP INT TERM

report() {
  printf 'secret-scan-branch: %s\n' "$1" >&2
}

# cannot_scan <reason> -- default mode: exit 0 with the reason (this scan is
# an early-warning gate, not a hard requirement). --strict: exit 3, so a
# caller that needs a real answer can tell "could not check" apart from a
# clean scan.
cannot_scan() {
  report "cannot scan, $1"
  if [ "$strict" -eq 1 ]; then
    exit 3
  fi
  exit 0
}

# nothing_to_scan <reason> -- always exit 0: there is genuinely nothing to
# check (not "could not check"), so --strict does not change this.
nothing_to_scan() {
  report "nothing to scan, $1"
  exit 0
}

if ! repo_root="$(git rev-parse --show-toplevel 2>/dev/null)"; then
  cannot_scan "not inside a git work tree"
fi

if [ -n "${GITHUB_BASE_REF:-}" ]; then
  base_name="$GITHUB_BASE_REF"
else
  helpers="$script_dir/xreview-helpers.sh"
  if [ ! -r "$helpers" ]; then
    cannot_scan "xreview-helpers.sh not found next to this script"
  fi
  # shellcheck source=xreview-helpers.sh
  . "$helpers"
  base_name="$(detect_base_branch)"
fi

if ! current_branch="$(git symbolic-ref --quiet --short HEAD 2>/dev/null)"; then
  current_branch=""
fi
if [ "$current_branch" = "$base_name" ]; then
  nothing_to_scan "HEAD is the base branch ($base_name)"
fi

if git rev-parse --verify --quiet "refs/remotes/origin/$base_name" >/dev/null 2>&1; then
  base_ref="origin/$base_name"
elif git rev-parse --verify --quiet "refs/heads/$base_name" >/dev/null 2>&1; then
  base_ref="$base_name"
else
  cannot_scan "no base ref for '$base_name' (checked refs/remotes/origin/$base_name and refs/heads/$base_name)"
fi

if ! merge_base="$(git merge-base HEAD "$base_ref" 2>/dev/null)"; then
  cannot_scan "no merge-base between HEAD and $base_ref (unrelated histories or a shallow clone?)"
fi

commit_count="$(git rev-list --count "$merge_base..HEAD")"
if [ "$commit_count" -eq 0 ]; then
  nothing_to_scan "no commits between $base_ref and HEAD"
fi

if [ ! -x "$scanner" ]; then
  cannot_scan "scanner not found or not executable at $scanner"
fi

base_short="$(git rev-parse --short "$merge_base")"
head_short="$(git rev-parse --short HEAD)"

# Always scan with the .gitallowed COMMITTED at HEAD, regardless of
# uncommitted worktree edits or a caller-set RALPH_SECRET_ALLOWLIST: CI
# reads only the committed file, so this must match what CI will see.
tmp_allowlist="$(mktemp "${TMPDIR:-/tmp}/ralph-secret-scan-branch-allowlist.XXXXXX")"
if git cat-file -e "HEAD:.gitallowed" 2>/dev/null; then
  git show "HEAD:.gitallowed" > "$tmp_allowlist"
else
  : > "$tmp_allowlist"
fi

worktree_allowlist="$repo_root/.gitallowed"
worktree_differs=0
if [ -f "$worktree_allowlist" ]; then
  diff -q "$worktree_allowlist" "$tmp_allowlist" >/dev/null 2>&1 || worktree_differs=1
elif [ -s "$tmp_allowlist" ]; then
  worktree_differs=1
fi

if [ -n "${RALPH_SECRET_ALLOWLIST:-}" ] || [ "$worktree_differs" -eq 1 ]; then
  report "ignoring uncommitted .gitallowed edits and/or RALPH_SECRET_ALLOWLIST; CI reads only the .gitallowed committed at HEAD"
fi

if RALPH_SECRET_ALLOWLIST="$tmp_allowlist" "$scanner" --range "$merge_base..HEAD"; then
  report "scanned ${base_short}..${head_short} against ${base_ref}: clean"
  exit 0
else
  scan_rc=$?
  report "scanned ${base_short}..${head_short} against ${base_ref}: findings"
  exit "$scan_rc"
fi
