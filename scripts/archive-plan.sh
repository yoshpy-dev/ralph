#!/usr/bin/env sh
set -eu

usage() {
  echo "Usage: ./scripts/archive-plan.sh <plan-file-or-directory-or-slug>"
  echo ""
  echo "Move an active plan (file or directory) to the archive."
  echo "Accepts a full path, a filename/dirname, or a slug (matched against docs/plans/active/)."
  exit 1
}

if [ "${1:-}" = "" ]; then
  usage
fi

arg="$1"
src=""

# Resolution order:
# 1. Exact file or directory path
# 2. Name under docs/plans/active/
# 3. Fuzzy match (file or directory)
if [ -f "$arg" ] || [ -d "$arg" ]; then
  src="$arg"
elif [ -f "docs/plans/active/$arg" ]; then
  src="docs/plans/active/$arg"
elif [ -d "docs/plans/active/$arg" ]; then
  src="docs/plans/active/$arg"
else
  # Fuzzy match: try files first, then directories
  match="$(find docs/plans/active -maxdepth 1 -name "*${arg}*" 2>/dev/null | head -n 1)"
  if [ -n "$match" ]; then
    src="$match"
  fi
fi

if [ -z "$src" ]; then
  echo "No matching active plan found for: $arg"
  echo ""
  echo "Active plans:"
  find docs/plans/active -maxdepth 1 \( -type f -name '*.md' -o -type d ! -name active \) 2>/dev/null || echo "  (none)"
  exit 1
fi

mkdir -p docs/plans/archive
name="$(basename "$src")"
dest="docs/plans/archive/$name"

if [ -e "$dest" ]; then
  echo "Archive already contains $name. Aborting to avoid overwrite."
  exit 1
fi

# Rewrite the plan's links in docs/tech-debt/README.md BEFORE moving the plan.
# If the move then fails, re-running this script just moves the plan: the
# README has no docs/plans/active/<name> link left, so it rewrites 0.
#
# A link is the literal text docs/plans/active/<name> followed by the end of
# the line, by a character that is not a name character ([A-Za-z0-9._-]), or
# by one "." that is itself followed by one of those (a sentence-ending
# period). Archiving <date>-foo therefore leaves
# docs/plans/active/<date>-foo-bar.md and docs/plans/active/<date>-foo.md
# alone. <name> is compared as a plain string (index/substr), never as a
# regex, so the "." in a name matches only a ".".
readme="docs/tech-debt/README.md"
tmp=""
trap 'if [ -n "$tmp" ]; then rm -f "$tmp"; fi' EXIT
trap 'exit 1' HUP INT TERM
if [ -f "$readme" ]; then
  tmp="$(mktemp "$(dirname "$readme")/.README.md.XXXXXX")"
  # Copy first so the README keeps its file mode when tmp replaces it.
  cp -p "$readme" "$tmp"
  if ! count="$(ARCHIVE_PLAN_NAME="$name" ARCHIVE_PLAN_OUT="$tmp" LC_ALL=C awk '
    function is_name_char(c) {
      return c != "" && index(name_chars, c) > 0
    }
    BEGIN {
      name_chars = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789._-"
      old = "docs/plans/active/" ENVIRON["ARCHIVE_PLAN_NAME"]
      new = "docs/plans/archive/" ENVIRON["ARCHIVE_PLAN_NAME"]
      out = ENVIRON["ARCHIVE_PLAN_OUT"]
      n = 0
    }
    {
      rest = $0
      res = ""
      while ((i = index(rest, old)) > 0) {
        after = i + length(old)
        c1 = substr(rest, after, 1)
        c2 = substr(rest, after + 1, 1)
        if (!is_name_char(c1) || (c1 == "." && !is_name_char(c2))) {
          res = res substr(rest, 1, i - 1) new
          n++
        } else {
          res = res substr(rest, 1, after - 1)
        }
        rest = substr(rest, after)
      }
      print res rest > out
    }
    END {
      if (NR > 0 && close(out) != 0) exit 1
      print n
    }
  ' "$readme")"; then
    echo "Failed to rewrite plan links in $readme. The plan was not moved." >&2
    exit 1
  fi
  if [ "$count" -gt 0 ]; then
    if ! mv "$tmp" "$readme"; then
      echo "Failed to replace $readme. The plan was not moved." >&2
      exit 1
    fi
    tmp=""
    echo "Updated $count reference(s) in $readme"
  else
    rm -f "$tmp"
    tmp=""
  fi
fi

if ! mv "$src" "$dest"; then
  echo "Failed to move $src to $dest. Fix the cause and re-run this script." >&2
  exit 1
fi
echo "Archived: $src -> $dest"
