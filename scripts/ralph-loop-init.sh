#!/usr/bin/env sh
set -eu

# Initialize a Ralph Loop session.
# Generates PROMPT.md and state files from a prompt template.
#
# Usage: ./ralph-loop-init.sh <task-type> <objective> [plan-slug]

LOOP_DIR=".harness/state/loop"
ARCHIVE_DIR=".harness/state/loop-archive"
TEMPLATE_DIR=".claude/skills/loop/prompts"

VALID_TYPES="general refactor test-coverage bugfix docs migration"

usage() {
  echo "Usage: $0 <task-type> <objective> [plan-slug]"
  echo ""
  echo "Task types: ${VALID_TYPES}"
  echo ""
  echo "Examples:"
  echo "  $0 general 'Implement user auth'"
  echo "  $0 refactor 'Extract shared utils' extract-utils"
  echo "  $0 bugfix 'Fix login timeout' login-timeout"
  exit 1
}

if [ $# -lt 2 ]; then
  usage
fi

task_type="$1"
objective="$2"
plan_slug="${3:-}"

# Validate task type
valid=0
for t in $VALID_TYPES; do
  if [ "$t" = "$task_type" ]; then
    valid=1
    break
  fi
done

if [ "$valid" -eq 0 ]; then
  echo "Error: invalid task type '${task_type}'"
  echo "Valid types: ${VALID_TYPES}"
  exit 1
fi

# Select template
template_file="${TEMPLATE_DIR}/${task_type}.md"

if [ ! -f "$template_file" ]; then
  echo "Error: template not found: ${template_file}"
  exit 1
fi

# Archive previous loop state if it exists
if [ -d "$LOOP_DIR" ] && [ -f "${LOOP_DIR}/task.json" ]; then
  archive_ts="$(date -u '+%Y%m%d-%H%M%S')"
  archive_dest="${ARCHIVE_DIR}/${archive_ts}"
  mkdir -p "$archive_dest"
  cp -r "${LOOP_DIR}/." "$archive_dest/"
  echo "Archived previous loop state to ${archive_dest}"
  rm -rf "$LOOP_DIR"
fi

# Create fresh state directories
mkdir -p "$LOOP_DIR"

# Resolve plan path — supports 3 forms:
#   1. Full path to an existing file (e.g., docs/plans/active/2026-04-10-foo/slice-1-bar.md)
#   2. Directory path (e.g., docs/plans/active/2026-04-10-foo) → uses _manifest.md
#   3. Slug (e.g., 2026-04-10-foo) → tries docs/plans/active/<slug>.md, then directory
plan_path=""
if [ -n "$plan_slug" ]; then
  if [ -f "$plan_slug" ]; then
    # Form 1: Full path to existing file
    plan_path="$plan_slug"
  elif [ -d "$plan_slug" ]; then
    # Form 2a: Direct directory path
    _manifest="${plan_slug}/_manifest.md"
    if [ -f "$_manifest" ]; then
      plan_path="$_manifest"
    else
      plan_path="$plan_slug"
      echo "Warning: directory plan has no _manifest.md: ${plan_slug}"
    fi
  elif [ -d "docs/plans/active/${plan_slug}" ]; then
    # Form 2b: Slug resolves to a directory
    _manifest="docs/plans/active/${plan_slug}/_manifest.md"
    if [ -f "$_manifest" ]; then
      plan_path="$_manifest"
    else
      plan_path="docs/plans/active/${plan_slug}"
      echo "Warning: directory plan has no _manifest.md: docs/plans/active/${plan_slug}"
    fi
  elif [ -f "docs/plans/active/${plan_slug}.md" ]; then
    # Form 3: Slug resolves to a single file
    plan_path="docs/plans/active/${plan_slug}.md"
  else
    echo "Warning: plan not found for '${plan_slug}', continuing without plan reference"
  fi
fi

# Generate PROMPT.md from template
sed \
  -e "s|__OBJECTIVE__|${objective}|g" \
  -e "s|__PLAN_PATH__|${plan_path}|g" \
  -e "s|__TASK_TYPE__|${task_type}|g" \
  "$template_file" > "${LOOP_DIR}/PROMPT.md"

echo "Generated ${LOOP_DIR}/PROMPT.md"

# Create task.json
created_ts="$(date -u '+%Y-%m-%dT%H:%M:%SZ')"

cat > "${LOOP_DIR}/task.json" <<EOF
{
  "objective": "${objective}",
  "task_type": "${task_type}",
  "plan": "${plan_path}",
  "created": "${created_ts}",
  "status": "pending"
}
EOF

echo "Created ${LOOP_DIR}/task.json"

# Initialize progress log
cat > "${LOOP_DIR}/progress.log" <<EOF
# Progress log
# Task: ${objective}
# Type: ${task_type}
# Created: ${created_ts}

EOF

echo "Created ${LOOP_DIR}/progress.log"

# Initialize stuck counter
echo "0" > "${LOOP_DIR}/stuck.count"

# Initialize status
echo "pending" > "${LOOP_DIR}/status"

echo ""
echo "Ralph Loop initialized."
echo "  Type:      ${task_type}"
echo "  Objective: ${objective}"
echo "  Plan:      ${plan_path:-none}"
echo ""
echo "Next steps:"
echo "  1. Review ${LOOP_DIR}/PROMPT.md"
echo "  2. Run: ./scripts/ralph run --plan <plan-directory> --unified-pr"
