#!/usr/bin/env bash
#
# check-skill-sync.sh — verify Claude (.claude/skills/) and Codex
# (.agents/skills/) carry the same skill bodies and matching trigger metadata.
#
# Spec: AC-3 of docs/specs/2026-05-07-codex-cli-parity.md
#
# Six checks per skill name:
#   1. Inventory parity     : skill directories on both sides match
#   2. SKILL.md body parity : frontmatter stripped, leading/trailing whitespace
#                             normalized, trailing whitespace dropped per line
#   3. name parity          : frontmatter `name:` matches the skill directory
#                             AND matches the other side
#   4. description parity   : frontmatter `description:` (single- or
#                             multi-line YAML folded scalar) matches between
#                             sides — Codex uses this for implicit invocation,
#                             so any drift quietly breaks discoverability
#   5. policy parity        : Claude `disable-model-invocation: true` ⇔
#                             Codex `policy.allow_implicit_invocation: false`
#                             (in agents/openai.yaml). Default on both sides
#                             is "implicit invocation allowed".
#   6. prompts/ parity      : every file under each skill's prompts/ directory
#                             must exist on both sides and be byte-identical;
#                             missing-in-mirror and extra-in-mirror both fail.
#
# NOTE on frontmatter parity scope: `allowed-tools` and
# `disable-model-invocation` are Claude Code-specific frontmatter fields and
# are intentionally NOT mirrored to the Codex side. Codex equivalents live in
# agents/openai.yaml policy metadata. This script checks body parity and the
# cross-vendor policy abstraction (check 5) — not raw frontmatter equality.
#
# Exit 0 = parity / exit 1 = drift; problem details printed to stderr.

set -eu

CLAUDE_ROOT="${CLAUDE_ROOT:-.claude/skills}"
CODEX_ROOT="${CODEX_ROOT:-.agents/skills}"

if [ ! -d "$CLAUDE_ROOT" ]; then
  echo "FAIL: claude root not found: $CLAUDE_ROOT" >&2
  exit 1
fi
if [ ! -d "$CODEX_ROOT" ]; then
  echo "FAIL: codex root not found: $CODEX_ROOT" >&2
  exit 1
fi

# ── helpers ──────────────────────────────────────────────────────────────────
list_skills() {
  # Print skill directory names (one per line) under the given root that
  # contain a SKILL.md. Sort for deterministic comparison.
  local root="$1"
  find "$root" -mindepth 2 -maxdepth 2 -name SKILL.md -type f \
    | sed -e "s|^$root/||" -e 's|/SKILL\.md$||' \
    | LC_ALL=C sort
}

# Strip the leading YAML frontmatter (--- fenced block) and emit only the body.
strip_frontmatter() {
  awk '
    BEGIN { in_fm = 0; past_fm = 0 }
    NR == 1 && /^---$/ { in_fm = 1; next }
    in_fm && /^---$/ { in_fm = 0; past_fm = 1; next }
    in_fm { next }
    { print }
  ' "$1"
}

normalize_body() {
  # Trim trailing whitespace from each line and collapse trailing blank lines
  # so cosmetic edits do not produce false positives.
  sed -e 's/[[:space:]]*$//' "$1" | awk '
    { lines[NR] = $0 }
    END {
      end = NR
      while (end > 0 && lines[end] == "") end--
      for (i = 1; i <= end; i++) print lines[i]
    }
  '
}

# Extract a top-level scalar field from the YAML frontmatter. Supports both
# single-line `key: value` and folded multi-line `key: >` blocks. Returns the
# value with internal whitespace collapsed for comparison.
extract_fm_field() {
  local file="$1"
  local field="$2"
  awk -v field="$field" '
    BEGIN { in_fm = 0; capture = 0; folded = 0; out = "" }
    NR == 1 && /^---$/ { in_fm = 1; next }
    in_fm && /^---$/ { in_fm = 0; exit }
    !in_fm { next }
    {
      if (capture) {
        # Folded scalar: indented continuation lines become part of the value.
        if ($0 ~ /^[[:space:]]/) {
          line = $0
          sub(/^[[:space:]]+/, "", line)
          out = (out == "") ? line : out " " line
          next
        } else {
          # End of folded block — emit and reset.
          print out
          capture = 0
          folded = 0
          out = ""
          # fall through so the new line can re-trigger a match
        }
      }
      if ($0 ~ "^" field ":[[:space:]]*$") {
        # Bare key — likely block scalar follows on next lines.
        capture = 1
        folded = 0
        out = ""
        next
      }
      if ($0 ~ "^" field ":[[:space:]]*>[+-]?[[:space:]]*$") {
        capture = 1
        folded = 1
        out = ""
        next
      }
      if ($0 ~ "^" field ":") {
        # Inline scalar — `key: value`. Print and stop.
        line = $0
        sub("^" field ":[[:space:]]*", "", line)
        # Strip surrounding quotes if present.
        gsub(/^"|"$/, "", line)
        gsub(/^'\''|'\''$/, "", line)
        print line
        exit
      }
    }
    END {
      if (capture && out != "") print out
    }
  ' "$file"
}

# Determine the implicit-invocation policy on each side.
# Claude side: "disable-model-invocation: true" → forbid implicit (return: forbid).
# Codex side: agents/openai.yaml `policy.allow_implicit_invocation: false` → forbid implicit.
# Default for both sides is "allow".
claude_policy() {
  local skill="$1"
  if grep -q '^disable-model-invocation:[[:space:]]*true' "$CLAUDE_ROOT/$skill/SKILL.md" 2>/dev/null; then
    echo "forbid"
  else
    echo "allow"
  fi
}

codex_policy() {
  local skill="$1"
  local yaml="$CODEX_ROOT/$skill/agents/openai.yaml"
  if [ -f "$yaml" ] && grep -q 'allow_implicit_invocation:[[:space:]]*false' "$yaml" 2>/dev/null; then
    echo "forbid"
  else
    echo "allow"
  fi
}

# ── checks ───────────────────────────────────────────────────────────────────
status=0
fail() { echo "FAIL: $*" >&2; status=1; }

CLAUDE_LIST="$(list_skills "$CLAUDE_ROOT")"
CODEX_LIST="$(list_skills "$CODEX_ROOT")"

# 1. Inventory parity.
ONLY_CLAUDE="$(LC_ALL=C comm -23 <(echo "$CLAUDE_LIST") <(echo "$CODEX_LIST"))"
ONLY_CODEX="$(LC_ALL=C comm -13 <(echo "$CLAUDE_LIST") <(echo "$CODEX_LIST"))"
if [ -n "$ONLY_CLAUDE" ]; then
  while IFS= read -r s; do
    [ -n "$s" ] && fail "skill '$s' exists in $CLAUDE_ROOT but not in $CODEX_ROOT"
  done <<<"$ONLY_CLAUDE"
fi
if [ -n "$ONLY_CODEX" ]; then
  while IFS= read -r s; do
    [ -n "$s" ] && fail "skill '$s' exists in $CODEX_ROOT but not in $CLAUDE_ROOT"
  done <<<"$ONLY_CODEX"
fi

# Per-skill checks for the intersection.
INTERSECT="$(LC_ALL=C comm -12 <(echo "$CLAUDE_LIST") <(echo "$CODEX_LIST"))"

while IFS= read -r skill; do
  [ -z "$skill" ] && continue
  cl_md="$CLAUDE_ROOT/$skill/SKILL.md"
  cx_md="$CODEX_ROOT/$skill/SKILL.md"

  # 2. Body parity.
  cl_body="$(mktemp)"
  cx_body="$(mktemp)"
  trap 'rm -f "$cl_body" "$cx_body"' EXIT
  strip_frontmatter "$cl_md" > "$cl_body.raw"
  strip_frontmatter "$cx_md" > "$cx_body.raw"
  normalize_body "$cl_body.raw" > "$cl_body"
  normalize_body "$cx_body.raw" > "$cx_body"
  if ! diff -q "$cl_body" "$cx_body" >/dev/null 2>&1; then
    fail "skill '$skill': SKILL.md body differs (frontmatter excluded). diff:"
    diff "$cl_body" "$cx_body" >&2 || true
  fi
  rm -f "$cl_body" "$cx_body" "$cl_body.raw" "$cx_body.raw"

  # 3. name parity (and matches the directory).
  cl_name="$(extract_fm_field "$cl_md" name | tr -d '[:space:]')"
  cx_name="$(extract_fm_field "$cx_md" name | tr -d '[:space:]')"
  if [ "$cl_name" != "$skill" ]; then
    fail "skill '$skill': claude frontmatter name='$cl_name' does not match directory"
  fi
  if [ "$cx_name" != "$skill" ]; then
    fail "skill '$skill': codex frontmatter name='$cx_name' does not match directory"
  fi
  if [ "$cl_name" != "$cx_name" ]; then
    fail "skill '$skill': name drift claude='$cl_name' codex='$cx_name'"
  fi

  # 4. description parity.
  cl_desc="$(extract_fm_field "$cl_md" description | tr -s '[:space:]' ' ' | sed 's/^ //;s/ $//')"
  cx_desc="$(extract_fm_field "$cx_md" description | tr -s '[:space:]' ' ' | sed 's/^ //;s/ $//')"
  if [ "$cl_desc" != "$cx_desc" ]; then
    fail "skill '$skill': description drift"
    echo "  claude: $cl_desc" >&2
    echo "  codex:  $cx_desc" >&2
  fi

  # 5. policy parity.
  cl_pol="$(claude_policy "$skill")"
  cx_pol="$(codex_policy "$skill")"
  if [ "$cl_pol" != "$cx_pol" ]; then
    fail "skill '$skill': implicit-invocation policy drift claude=$cl_pol codex=$cx_pol"
  fi

  # 6. prompts/ parity (byte-identical; missing-in-mirror and extra-in-mirror both fail).
  #    Recursively compares all files under prompts/ (including nested subdirectories)
  #    using relative paths so e.g. prompts/sub/foo.md participates in both checks.
  cl_prompts="$CLAUDE_ROOT/$skill/prompts"
  cx_prompts="$CODEX_ROOT/$skill/prompts"
  cl_has_prompts=0; cx_has_prompts=0
  [ -d "$cl_prompts" ] && cl_has_prompts=1
  [ -d "$cx_prompts" ] && cx_has_prompts=1
  if [ "$cl_has_prompts" -ne "$cx_has_prompts" ]; then
    if [ "$cl_has_prompts" -eq 1 ]; then
      fail "skill '$skill': prompts/ exists in $CLAUDE_ROOT but not in $CODEX_ROOT"
    else
      fail "skill '$skill': prompts/ exists in $CODEX_ROOT but not in $CLAUDE_ROOT"
    fi
  elif [ "$cl_has_prompts" -eq 1 ]; then
    # Both sides have a prompts/ directory — compare file lists and content recursively.
    cl_files="$(find "$cl_prompts" -type f | sed "s|^$cl_prompts/||" | LC_ALL=C sort)"
    cx_files="$(find "$cx_prompts" -type f | sed "s|^$cx_prompts/||" | LC_ALL=C sort)"
    only_cl="$(LC_ALL=C comm -23 <(echo "$cl_files") <(echo "$cx_files"))"
    only_cx="$(LC_ALL=C comm -13 <(echo "$cl_files") <(echo "$cx_files"))"
    if [ -n "$only_cl" ]; then
      while IFS= read -r f; do
        [ -n "$f" ] && fail "skill '$skill': prompts/$f exists in $CLAUDE_ROOT but not in $CODEX_ROOT"
      done <<<"$only_cl"
    fi
    if [ -n "$only_cx" ]; then
      while IFS= read -r f; do
        [ -n "$f" ] && fail "skill '$skill': prompts/$f exists in $CODEX_ROOT but not in $CLAUDE_ROOT"
      done <<<"$only_cx"
    fi
    # Byte-identical check for files present on both sides.
    common_files="$(LC_ALL=C comm -12 <(echo "$cl_files") <(echo "$cx_files"))"
    while IFS= read -r f; do
      [ -z "$f" ] && continue
      if ! cmp -s "$cl_prompts/$f" "$cx_prompts/$f"; then
        fail "skill '$skill': prompts/$f content differs between $CLAUDE_ROOT and $CODEX_ROOT"
      fi
    done <<<"$common_files"
  fi
done <<<"$INTERSECT"

if [ "$status" -eq 0 ]; then
  count="$(echo "$CLAUDE_LIST" | grep -c .)"
  echo "[ok] check-skill-sync: $count skill(s) in lock-step"
fi

exit "$status"
