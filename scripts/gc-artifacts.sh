#!/usr/bin/env bash
set -euo pipefail

# gc-artifacts.sh — Garbage-collect stale pipeline artifacts.
#
# Targets:
#   1. docs/reports/*.md  older than N days (by filename date, falling back to
#      mtime). Never deletes README.md or dotfiles. Uses `git rm` for tracked
#      files.
#   2. docs/evidence/verify-*.log — keeps the newest 20, removes the rest via
#      plain `rm` (these are local/gitignored).
#
# Default mode is dry-run: lists candidates without deleting.
# Pass --apply to perform deletion.

usage() {
  cat <<'USAGE'
Usage: ./scripts/gc-artifacts.sh [--apply] [--days N] [--help]

Options:
  --apply     Actually delete files (default: dry-run, print candidates only)
  --days N    Age threshold in days (default: 30)
  --help      Show this message
USAGE
}

log() {
  printf '[gc-artifacts] %s\n' "$*"
}

# ─── Argument parsing ─────────────────────────────────────────────────────────

APPLY=0
DAYS=30

while [ $# -gt 0 ]; do
  case "$1" in
    --apply)
      APPLY=1
      shift
      ;;
    --days)
      shift
      if [ "${1:-}" = "" ]; then
        echo "gc-artifacts.sh: --days requires a value" >&2
        exit 1
      fi
      DAYS="$1"
      shift
      ;;
    --help|-h)
      usage
      exit 0
      ;;
    *)
      echo "gc-artifacts.sh: unknown option: $1" >&2
      usage >&2
      exit 1
      ;;
  esac
done

# ─── Resolve repo root ────────────────────────────────────────────────────────

REPO_ROOT="$(git rev-parse --show-toplevel)"
cd "$REPO_ROOT"

# ─── Helpers ──────────────────────────────────────────────────────────────────

# extract_filename_date <filename>
# Try to extract a YYYY-MM-DD from the basename. Returns empty string on failure.
extract_filename_date() {
  local base
  base="$(basename "$1")"
  printf '%s' "$base" | grep -oE '[0-9]{4}-[0-9]{2}-[0-9]{2}' | head -n 1 || true
}

# date_is_older_than_days <date_string> <days>
# Returns 0 (true) if the date is older than <days> days ago.
date_is_older_than_days() {
  local date_str="$1"
  local days="$2"
  # macOS stat/date compatibility: use date -j -f
  local cutoff
  if date --version >/dev/null 2>&1; then
    # GNU date
    cutoff="$(date -d "$days days ago" '+%Y-%m-%d')"
  else
    # BSD date (macOS)
    cutoff="$(date -v"-${days}d" '+%Y-%m-%d')"
  fi
  [ "$date_str" '<' "$cutoff" ] || [ "$date_str" = "$cutoff" ]
}

# mtime_is_older_than_days <filepath> <days>
# Returns 0 (true) if the file mtime is older than <days> days ago.
mtime_is_older_than_days() {
  local filepath="$1"
  local days="$2"
  local cutoff_epoch now_epoch mtime_epoch
  now_epoch="$(date '+%s')"
  cutoff_epoch=$((now_epoch - days * 86400))
  if stat --version >/dev/null 2>&1; then
    # GNU stat
    mtime_epoch="$(stat -c '%Y' "$filepath")"
  else
    # BSD stat (macOS)
    mtime_epoch="$(stat -f '%m' "$filepath")"
  fi
  [ "$mtime_epoch" -le "$cutoff_epoch" ]
}

# is_git_tracked <filepath>
# Returns 0 if the file is tracked by git.
is_git_tracked() {
  git ls-files --error-unmatch "$1" >/dev/null 2>&1
}

# delete_file <filepath>
# Deletes a tracked file with `git rm`, untracked file with `rm`.
delete_file() {
  local filepath="$1"
  if is_git_tracked "$filepath"; then
    git rm --quiet "$filepath"
  else
    rm -f "$filepath"
  fi
}

# ─── Target 1: docs/reports/*.md ──────────────────────────────────────────────

reports_deleted=0
reports_candidates=()

if [ -d "docs/reports" ]; then
  while IFS= read -r f; do
    base="$(basename "$f")"
    # Never touch README.md or dotfiles
    case "$base" in
      README.md|.*)
        continue
        ;;
    esac

    older=0
    fname_date="$(extract_filename_date "$f")"
    if [ -n "$fname_date" ]; then
      if date_is_older_than_days "$fname_date" "$DAYS"; then
        older=1
      fi
    else
      # Fall back to mtime
      if mtime_is_older_than_days "$f" "$DAYS"; then
        older=1
      fi
    fi

    if [ "$older" -eq 1 ]; then
      reports_candidates+=("$f")
    fi
  done < <(find docs/reports -maxdepth 1 -name "*.md" -type f | sort)
fi

# ─── Target 2: docs/evidence/verify-*.log ─────────────────────────────────────

evidence_deleted=0
evidence_candidates=()

if [ -d "docs/evidence" ]; then
  # Collect all verify-*.log files, sorted newest-first
  # Use a counter instead of mapfile (bash 3 compatible)
  ev_count=0
  while IFS= read -r ev_file; do
    ev_count=$((ev_count + 1))
    if [ "$ev_count" -gt 20 ]; then
      evidence_candidates+=("$ev_file")
    fi
  done < <(find docs/evidence -maxdepth 1 -name "verify-*.log" -type f | sort -r)
fi

# ─── Output / execution ───────────────────────────────────────────────────────

# Bash 3.2: empty array[@] triggers unbound variable under -u; use set +u guard
set +u
n_reports="${#reports_candidates[@]}"
n_evidence="${#evidence_candidates[@]}"
set -u
n_reports="${n_reports:-0}"
n_evidence="${n_evidence:-0}"
total_candidates=$(( n_reports + n_evidence ))

if [ "$total_candidates" -eq 0 ]; then
  log "Nothing to do (reports older than ${DAYS}d: 0, evidence beyond newest 20: 0)"
  exit 0
fi

if [ "$APPLY" -eq 0 ]; then
  log "Dry-run mode (pass --apply to delete)"
  log ""
  if [ "$n_reports" -gt 0 ]; then
    log "docs/reports/ candidates (older than ${DAYS} days):"
    set +u
    for f in "${reports_candidates[@]}"; do
      printf '  %s\n' "$f"
    done
    set -u
  fi
  if [ "$n_evidence" -gt 0 ]; then
    log "docs/evidence/ candidates (beyond newest 20 verify-*.log):"
    set +u
    for f in "${evidence_candidates[@]}"; do
      printf '  %s\n' "$f"
    done
    set -u
  fi
  log ""
  log "Summary: ${n_reports} report(s) + ${n_evidence} evidence log(s) would be deleted"
else
  log "Applying deletions..."
  set +u
  for f in "${reports_candidates[@]}"; do
    delete_file "$f"
    printf '  deleted: %s\n' "$f"
    reports_deleted=$((reports_deleted + 1))
  done
  for f in "${evidence_candidates[@]}"; do
    rm -f "$f"
    printf '  deleted: %s\n' "$f"
    evidence_deleted=$((evidence_deleted + 1))
  done
  set -u
  total_deleted=$(( reports_deleted + evidence_deleted ))
  log "Summary: ${reports_deleted} report(s) + ${evidence_deleted} evidence log(s) deleted (${total_deleted} total)"
fi
