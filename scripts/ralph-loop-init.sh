#!/usr/bin/env sh
set -eu

# Initialize a Ralph Loop session.
# Generates PROMPT.md and state files from a prompt template.

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

# Create fresh state directory
mkdir -p "$LOOP_DIR"

# Resolve plan path
plan_path=""
if [ -n "$plan_slug" ]; then
  candidate="docs/plans/active/${plan_slug}.md"
  if [ -f "$candidate" ]; then
    plan_path="$candidate"
  else
    echo "Warning: plan file not found at ${candidate}, continuing without plan reference"
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
echo "  2. Run: ./scripts/ralph-loop.sh"
echo "  3. Optional: ./scripts/ralph-loop.sh --verify --max-iterations 10"
