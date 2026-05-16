#!/usr/bin/env sh
set -eu

# Generate a directory-based Ralph Loop plan with manifest + N slice files.
#
# Usage: ./scripts/new-ralph-plan.sh [--type <type>] <slug> [issue-number] [slice-count]
#
# Creates:
#   docs/plans/active/<date>-<slug>/
#     _manifest.md
#     slice-1-<slug>.md
#     slice-2-<slug>.md
#     ...

usage() {
  echo "Usage: ./scripts/new-ralph-plan.sh [--type <type>] <slug> [issue-number] [slice-count]"
  echo ""
  echo "Arguments:"
  echo "  --type TYPE   Branch type prefix (default: feat)"
  echo "  slug          Short identifier for the plan (e.g., auth-api)"
  echo "  issue-number  GitHub issue number (default: N/A)"
  echo "  slice-count   Number of slice files to generate (default: 2)"
  echo "Allowed types: $(./scripts/branch-name.sh allowed-types)"
  exit 1
}

type="feat"
if [ "${1:-}" = "--type" ]; then
  [ "${2:-}" != "" ] || usage
  type="${2:-}"
  shift 2
fi

if [ "${1:-}" = "" ] || [ "${1:-}" = "-h" ] || [ "${1:-}" = "--help" ]; then
  usage
fi

slug="$1"
issue="${2:-N/A}"
slice_count="${3:-2}"
date_str="$(date '+%Y-%m-%d')"
plan_dir="docs/plans/active/${date_str}-${slug}"
template_dir="docs/plans/templates"

if ! ./scripts/branch-name.sh validate "${type}/placeholder" >/dev/null 2>&1; then
  echo "Error: invalid branch type: ${type}"
  echo "Allowed types: $(./scripts/branch-name.sh allowed-types)"
  exit 1
fi

if [ -d "$plan_dir" ]; then
  echo "Plan directory already exists: ${plan_dir}"
  exit 1
fi

# Validate slice count
if ! echo "$slice_count" | grep -qE '^[0-9]+$' || [ "$slice_count" -lt 1 ]; then
  echo "Error: slice-count must be a positive integer, got: ${slice_count}"
  exit 1
fi

# Check templates exist
if [ ! -f "${template_dir}/ralph-loop-manifest.md" ]; then
  echo "Error: manifest template not found: ${template_dir}/ralph-loop-manifest.md"
  exit 1
fi
if [ ! -f "${template_dir}/ralph-loop-slice.md" ]; then
  echo "Error: slice template not found: ${template_dir}/ralph-loop-slice.md"
  exit 1
fi

# Create plan directory
mkdir -p "$plan_dir"

# Generate manifest
pr_group_slices=""
i=1
while [ "$i" -le "$slice_count" ]; do
  pr_group_slices="${pr_group_slices:+${pr_group_slices}, }\"slice-${i}-${slug}\""
  i=$((i + 1))
done

sed \
  -e "s#__TITLE__#${slug}#g" \
  -e "s#__DATE__#${date_str}#g" \
  -e "s#__REQUEST__#${slug}#g" \
  -e "s#__ISSUE__#${issue}#g" \
  -e "s#__TYPE__#${type}#g" \
  -e "s#__SLUG__#${slug}#g" \
  -e "s#__PR_GROUP_SLICES__#${pr_group_slices}#g" \
  "${template_dir}/ralph-loop-manifest.md" > "${plan_dir}/_manifest.md"

echo "Created ${plan_dir}/_manifest.md"

# Generate slice files
i=1
while [ "$i" -le "$slice_count" ]; do
  slice_file="${plan_dir}/slice-${i}-${slug}.md"
  sed \
    -e "s#__SLICE_NAME__#${slug}-${i}#g" \
    -e "s#__SLICE_NUM__#${i}#g" \
    -e "s#__PARENT_PLAN__#${plan_dir}/_manifest.md#g" \
    "${template_dir}/ralph-loop-slice.md" > "$slice_file"
  echo "Created ${slice_file}"
  i=$((i + 1))
done

echo ""
echo "Ralph Loop plan created: ${plan_dir}/"
echo "  Manifest: ${plan_dir}/_manifest.md"
echo "  Slices:   ${slice_count}"
echo ""
echo "Next steps:"
echo "  1. Fill in the manifest (objective, scope, locklist, dependency graph)"
echo "  2. Record the PR strategy decision: AI recommendation, rationale, and human approval"
echo "  3. Adjust PR grouping (default: grouped) if the generated group is too broad"
echo "  4. Fill in each slice (objective, AC, affected files, dependencies)"
echo "  5. Run: ./scripts/ralph run --plan ${plan_dir}"
