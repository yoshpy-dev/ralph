#!/usr/bin/env sh
# secret-scan-branch.sh - scan this branch's committed history for secrets,
# the same way CI does: .github/workflows/verify.yml scans
# merge-base(HEAD, origin/<base>)..HEAD with `git log -p` (scripts/secret-scan.sh
# --range). Editing away a leaked line in a later commit does not clear a
# finding here or in CI -- the range scan still reads the commit that
# introduced it.
#
# The base ref used for merge-base is always the fully qualified form
# (refs/remotes/origin/<base> or refs/heads/<base>) that was actually
# checked to exist, never the short name: git resolves a short name
# through refs/<name>, refs/tags/<name>, refs/heads/<name>, then
# refs/remotes/<name>, so a tag or a differently-scoped ref sharing the
# base's short name could otherwise be picked instead and silently change
# the range. The allowlist is always .gitallowed as committed at HEAD; if
# that entry is a symlink, its target is resolved one level inside the
# committed tree (the same content a checkout of HEAD would show CI's
# scanner). Anything that cannot be resolved that way scans with an empty
# allowlist rather than skipping the scan, since fewer exceptions can only
# add findings, never hide one.
#
# Usage: secret-scan-branch.sh [--strict]
#
# Exit codes:
#   0  scanned and found nothing; in default mode, also nothing to scan or
#      could not scan (see --strict below)
#   1  scanned and found something
#   2  usage error
#   3  (--strict only) could not scan, or nothing to scan
#
# Default mode never fails on an unscannable state (no base ref, no
# merge-base, ...) or on a genuinely empty range: it is meant for
# run-verify.sh, an early-warning gate that must not block on environment
# gaps such as a repo without a fetched base branch. --strict is for a
# caller that needs a real answer before an irreversible step (the /pr
# skill, right before push), where exit 0 must mean exactly "scanned,
# clean": both "could not scan" and "nothing to scan" (HEAD is the base
# branch, or the base and HEAD have no commits between them -- either one
# reachable by pointing the base at HEAD's own branch, or by fast-forwarding
# a local base ref onto HEAD while origin/<base> is absent) exit 3, so a
# caller cannot mistake "did not look" for "looked and it was clean".
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
# POSIX sh resumes the script after a signal trap's action runs (a signal
# trap is not a special early exit): without an explicit exit here, a
# HUP/INT/TERM delivered mid-script would run its trap action and then fall
# through to whatever command was next, instead of stopping. Exit with the
# conventional 128+signum code so the EXIT trap (cleanup) still fires.
trap cleanup EXIT
trap 'exit 129' HUP
trap 'exit 130' INT
trap 'exit 143' TERM

report() {
  printf 'secret-scan-branch: %s\n' "$1" >&2
}

# report_and_exit <prefix> <reason> -- shared exit logic for both
# "cannot scan" (could not determine what to scan) and "nothing to scan"
# (determined there is genuinely nothing to look at): default mode never
# fails on either (this scan is an early-warning gate, not a hard
# requirement). --strict treats both as "not a clean scan" (exit 3), so a
# caller that needs a real answer never confuses "did not look" with
# "looked and it was clean" -- exit 0 under --strict means exactly
# "scanned, clean".
report_and_exit() {
  report "$1, $2"
  if [ "$strict" -eq 1 ]; then
    exit 3
  fi
  exit 0
}

# cannot_scan <reason> -- see report_and_exit.
cannot_scan() {
  report_and_exit "cannot scan" "$1"
}

# nothing_to_scan <reason> -- see report_and_exit.
nothing_to_scan() {
  report_and_exit "nothing to scan" "$1"
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
  base_ref_full="refs/remotes/origin/$base_name"
  base_ref="origin/$base_name"
elif git rev-parse --verify --quiet "refs/heads/$base_name" >/dev/null 2>&1; then
  base_ref_full="refs/heads/$base_name"
  base_ref="$base_name"
else
  cannot_scan "no base ref for '$base_name' (checked refs/remotes/origin/$base_name and refs/heads/$base_name)"
fi

# merge-base is computed from base_ref_full (the fully qualified ref just
# checked above), never from the short base_ref: a bare short name is
# resolved by git's own disambiguation order, which checks refs/tags/<name>
# and refs/heads/<name> before refs/remotes/<name> -- a tag or a
# differently-scoped ref sharing the base's short name would otherwise be
# picked over the ref that was actually validated. base_ref (short form)
# is used only in the printed report lines below.
if ! merge_base="$(git merge-base HEAD "$base_ref_full" 2>/dev/null)"; then
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

# allowlist_target_is_safe <target> -- true if <target> (the stored
# content of a .gitallowed symlink, i.e. its link target path) is safe to
# resolve one level inside the committed tree: not absolute, and with no
# ".." path component. .gitallowed always sits at the repo root, so a
# relative target is relative to the root.
allowlist_target_is_safe() {
  case "$1" in
    /* | "") return 1 ;;
  esac
  case "/$1/" in
    */../*) return 1 ;;
  esac
  return 0
}

# newline / tab -- single-character values used to parse `git ls-tree`
# output below (one entry per line, fields "<mode> <type> <sha>\t<path>").
newline='
'
tab="$(printf '\t')"

# allowlist_ls_tree_mode <path> -- prints the tree-entry MODE for <path>
# at HEAD, but only when `git ls-tree --full-tree` returns EXACTLY ONE
# entry whose own recorded path is <path> itself; prints nothing (and
# fails) otherwise. --full-tree makes the pathspec root-relative
# regardless of the script's own cwd: a plain `git ls-tree HEAD --
# <path>` is cwd-relative and silently finds nothing when this script
# runs from a subdirectory. Requiring a single entry whose path matches
# exactly guards against a directory pathspec (bare, or with a trailing
# slash): `git ls-tree` lists a directory's CHILDREN instead of failing,
# so without this check a symlink target naming a directory could pass
# the mode check via its first child's mode while what gets read next
# (`git show HEAD:<path>`) is a tree listing, not a file.
allowlist_ls_tree_mode() {
  entry="$(git ls-tree --full-tree HEAD -- "$1" 2>/dev/null)"
  [ -n "$entry" ] || return 1
  case "$entry" in
    *"$newline"*) return 1 ;;
  esac
  [ "${entry#*"$tab"}" = "$1" ] || return 1
  printf '%s\n' "${entry%% *}"
}

# Always scan with the .gitallowed COMMITTED at HEAD, regardless of
# uncommitted worktree edits or a caller-set RALPH_SECRET_ALLOWLIST: CI
# reads only the committed file, so this must match what CI will see.
#
# The tree entry's mode decides how it is read: a regular file (100644 /
# 100755) is read directly with `git show`. A symlink (120000) stores its
# link target as the blob's content -- `git show HEAD:.gitallowed` on a
# symlink prints that target PATH STRING, not file content, so it is
# resolved one level inside the committed tree instead of used as-is.
# Anything else (no entry, a directory, a submodule, or a symlink target
# that cannot be resolved to a readable file) falls back to an empty
# allowlist: fewer exceptions can only add findings, never hide one.
tmp_allowlist="$(mktemp "${TMPDIR:-/tmp}/ralph-secret-scan-branch-allowlist.XXXXXX")"
allowlist_mode="$(allowlist_ls_tree_mode .gitallowed)" || allowlist_mode=""
case "$allowlist_mode" in
  100644|100755)
    git show "HEAD:.gitallowed" > "$tmp_allowlist"
    ;;
  120000)
    allowlist_link_target="$(git show "HEAD:.gitallowed" 2>/dev/null)"
    allowlist_resolved=0
    if allowlist_target_is_safe "$allowlist_link_target"; then
      while [ "${allowlist_link_target#./}" != "$allowlist_link_target" ]; do
        allowlist_link_target="${allowlist_link_target#./}"
      done
      allowlist_target_mode="$(allowlist_ls_tree_mode "$allowlist_link_target")" || allowlist_target_mode=""
      case "$allowlist_target_mode" in
        100644|100755)
          if git show "HEAD:$allowlist_link_target" > "$tmp_allowlist" 2>/dev/null; then
            allowlist_resolved=1
          fi
          ;;
      esac
    fi
    if [ "$allowlist_resolved" -eq 0 ]; then
      : > "$tmp_allowlist"
      report "committed .gitallowed could not be read as a file; scanning without allowlist exceptions"
    fi
    ;;
  "")
    : > "$tmp_allowlist"
    ;;
  *)
    : > "$tmp_allowlist"
    report "committed .gitallowed could not be read as a file; scanning without allowlist exceptions"
    ;;
esac

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
  # Only exit 1 is "the scanner ran and found something" (secret-scan.sh's
  # own contract). Any other non-zero exit (a usage error, a signal) is a
  # scanner failure, not a finding -- fail closed (propagate the exit code
  # so the caller still blocks) but say so accurately, instead of sending
  # the operator looking for a leak that does not exist.
  if [ "$scan_rc" -eq 1 ]; then
    report "scanned ${base_short}..${head_short} against ${base_ref}: findings"
  else
    report "scanner failed with exit ${scan_rc} on ${merge_base}..HEAD"
  fi
  exit "$scan_rc"
fi
