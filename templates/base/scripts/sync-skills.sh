#!/usr/bin/env bash
#
# sync-skills.sh — regenerate .agents/skills/ from .claude/skills/.
#
# Transforms each Claude skill into its Codex-side mirror:
#   - SKILL.md: copies body verbatim; drops Claude-only frontmatter keys
#     (allowed-tools, disable-model-invocation); keeps all other keys
#     (name, description, user-invocable, etc.).
#   - agents/openai.yaml: created when the source has
#     "disable-model-invocation: true" (maps to Codex's
#     policy.allow_implicit_invocation: false). Removed when no longer needed.
#   - All other files (template.md, prompts/, etc.): copied byte-for-byte.
#   - Orphan mirror dirs (skill removed from source) are deleted.
#
# Run this after editing any .claude/skills/ file. Check parity with:
#   ./scripts/check-skill-sync.sh
#
# Usage:
#   ./scripts/sync-skills.sh [--help]
#
# Exit 0 = success (mirror regenerated or already in sync).
# Exit 1 = fatal error.

set -euo pipefail

CLAUDE_ROOT="${CLAUDE_ROOT:-.claude/skills}"
CODEX_ROOT="${CODEX_ROOT:-.agents/skills}"

usage() {
  cat <<'USAGE'
Usage: ./scripts/sync-skills.sh [--help]

Regenerates .agents/skills/ from .claude/skills/ by transforming SKILL.md
frontmatter (dropping Claude-only keys) and copying auxiliary files verbatim.

Options:
  --help    Show this message

Environment:
  CLAUDE_ROOT   Source skill root (default: .claude/skills)
  CODEX_ROOT    Mirror skill root (default: .agents/skills)
USAGE
}

log() {
  printf '[sync-skills] %s\n' "$*"
}

# ── argument parsing ──────────────────────────────────────────────────────────

for arg in "$@"; do
  case "$arg" in
    --help|-h)
      usage
      exit 0
      ;;
    *)
      echo "sync-skills.sh: unknown argument: $arg" >&2
      usage >&2
      exit 1
      ;;
  esac
done

# ── sanity checks ─────────────────────────────────────────────────────────────

if [ ! -d "$CLAUDE_ROOT" ]; then
  echo "sync-skills.sh: source root not found: $CLAUDE_ROOT" >&2
  exit 1
fi

# ── helpers ───────────────────────────────────────────────────────────────────

# List skill names (directory names that contain a SKILL.md) under a root.
list_skills() {
  local root="$1"
  find "$root" -mindepth 2 -maxdepth 2 -name SKILL.md -type f \
    | sed -e "s|^$root/||" -e 's|/SKILL\.md$||' \
    | LC_ALL=C sort
}

# Transform SKILL.md frontmatter: drop Claude-only keys, keep the rest.
# Claude-only keys: allowed-tools, disable-model-invocation.
# Output goes to stdout.
transform_skill_md() {
  local src="$1"
  awk '
    BEGIN { in_fm = 0; past_fm = 0; skip_line = 0 }
    NR == 1 && /^---$/ { in_fm = 1; print; next }
    in_fm && /^---$/ {
      in_fm = 0; past_fm = 1
      print
      next
    }
    in_fm {
      # Drop Claude-only top-level keys (and their continuation lines).
      if (/^allowed-tools:/ || /^disable-model-invocation:/) {
        skip_line = 1
        next
      }
      # Continuation line: starts with whitespace while a key was being skipped.
      # (These keys use inline scalar values so no true continuation, but guard
      # defensively for future multi-line forms.)
      if (skip_line && /^[[:space:]]/) {
        next
      }
      # A new key (or the closing ---) resets skip_line.
      skip_line = 0
      print
      next
    }
    { print }
  ' "$src"
}

# Determine whether the source SKILL.md has disable-model-invocation: true.
has_disable_invocation() {
  local src="$1"
  grep -q '^disable-model-invocation:[[:space:]]*true' "$src" 2>/dev/null
}

# ── main sync loop ────────────────────────────────────────────────────────────

# Ensure the mirror root exists.
mkdir -p "$CODEX_ROOT"

SOURCE_SKILLS="$(list_skills "$CLAUDE_ROOT")"

# 1. For each source skill: write the mirror SKILL.md and copy auxiliary files.
while IFS= read -r skill; do
  [ -z "$skill" ] && continue

  src_dir="$CLAUDE_ROOT/$skill"
  dst_dir="$CODEX_ROOT/$skill"
  src_md="$src_dir/SKILL.md"
  dst_md="$dst_dir/SKILL.md"

  mkdir -p "$dst_dir"

  # Transform and write SKILL.md.
  transform_skill_md "$src_md" > "$dst_md"

  # agents/openai.yaml: create when source forbids implicit invocation;
  # remove when source no longer has the flag.
  agents_dir="$dst_dir/agents"
  openai_yaml="$agents_dir/openai.yaml"
  if has_disable_invocation "$src_md"; then
    mkdir -p "$agents_dir"
    printf 'policy:\n  allow_implicit_invocation: false\n' > "$openai_yaml"
  else
    # Remove if it exists and the flag was dropped.
    if [ -f "$openai_yaml" ]; then
      rm -f "$openai_yaml"
      # Remove agents/ dir if now empty.
      rmdir "$agents_dir" 2>/dev/null || true
    fi
  fi

  # Copy every other file/directory verbatim (template.md, prompts/, etc.).
  # We use find to enumerate regular files in the source dir (excluding SKILL.md)
  # and copy them, preserving relative paths.
  find "$src_dir" -mindepth 1 -type f ! -name 'SKILL.md' | while IFS= read -r src_file; do
    rel="${src_file#"$src_dir/"}"
    dst_file="$dst_dir/$rel"
    mkdir -p "$(dirname "$dst_file")"
    cp "$src_file" "$dst_file"
  done

  # Remove mirror-side files that no longer exist on the source side
  # (e.g. a prompts/ file was deleted).  Walk the mirror dir for non-SKILL.md
  # files and prune those absent from the source.
  find "$dst_dir" -mindepth 1 -type f ! -name 'SKILL.md' | while IFS= read -r dst_file; do
    rel="${dst_file#"$dst_dir/"}"
    # Skip the agents/openai.yaml — managed above.
    if [ "$rel" = "agents/openai.yaml" ]; then
      continue
    fi
    src_file="$src_dir/$rel"
    if [ ! -f "$src_file" ]; then
      rm -f "$dst_file"
    fi
  done

  # Prune empty directories left behind in the mirror (except agents/ which is
  # managed explicitly).
  find "$dst_dir" -mindepth 1 -type d ! -name 'agents' | LC_ALL=C sort -r | while IFS= read -r d; do
    rmdir "$d" 2>/dev/null || true
  done

done <<<"$SOURCE_SKILLS"

# 2. Remove mirror dirs for skills that no longer exist on the source side.
if [ -d "$CODEX_ROOT" ]; then
  MIRROR_SKILLS="$(list_skills "$CODEX_ROOT")"
  while IFS= read -r skill; do
    [ -z "$skill" ] && continue
    if ! echo "$SOURCE_SKILLS" | grep -qx "$skill"; then
      log "removing orphan mirror: $CODEX_ROOT/$skill"
      rm -rf "${CODEX_ROOT:?}/$skill"
    fi
  done <<<"$MIRROR_SKILLS"
fi

log "done: $(echo "$SOURCE_SKILLS" | grep -c .) skill(s) mirrored to $CODEX_ROOT"
