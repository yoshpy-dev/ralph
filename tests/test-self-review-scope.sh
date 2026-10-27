#!/usr/bin/env sh
# Regression guard for the self-review phase boundary.

set -eu

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
cd "$PROJECT_ROOT"

_pass=0
_fail=0
_total=0

record_pass() {
  _pass=$((_pass + 1))
  _total=$((_total + 1))
  printf '  PASS: %s\n' "$1"
}

record_fail() {
  _fail=$((_fail + 1))
  _total=$((_total + 1))
  printf '  FAIL: %s\n' "$1"
}

# Case handling differs between the helpers below (and from the helpers of the
# same names in other tests), so each comment says which one it uses.

# assert_contains: ignores case, line by line.
assert_contains() {
  _desc="$1"
  _needle="$2"
  _file="$3"
  if grep -Fiq -- "$_needle" "$_file"; then
    record_pass "$_desc"
  else
    record_fail "$_desc (missing: $_needle in $_file)"
  fi
}

# assert_not_contains: case-sensitive, line by line (the script-name checks).
assert_not_contains() {
  _desc="$1"
  _needle="$2"
  _file="$3"
  if grep -Fq -- "$_needle" "$_file"; then
    record_fail "$_desc (unexpected: $_needle in $_file)"
  else
    record_pass "$_desc"
  fi
}

# assert_not_contains_any_case: ignores case, line by line, so a removed item
# stays out however it is capitalized.
assert_not_contains_any_case() {
  _desc="$1"
  _needle="$2"
  _file="$3"
  if grep -Fiq -- "$_needle" "$_file"; then
    record_fail "$_desc (unexpected in any case: $_needle in $_file)"
  else
    record_pass "$_desc"
  fi
}

# join_lines: stdin on one line, line breaks and runs of spaces as single
# spaces.
join_lines() {
  tr '\n' ' ' | tr -s ' '
}

# assert_joined_contains_exact: case-sensitive, on the file with its line
# breaks folded into spaces, so a re-wrapped sentence still matches.
assert_joined_contains_exact() {
  _desc="$1"
  _needle="$2"
  _file="$3"
  if [ ! -f "$_file" ]; then
    record_fail "$_desc (no such file: $_file)"
  elif join_lines < "$_file" | grep -Fq -- "$_needle"; then
    record_pass "$_desc"
  else
    record_fail "$_desc (missing: $_needle in $_file)"
  fi
}

# review_areas FILE: the lines that list the review areas: the description
# line, and the "Focus on diff quality only:" or "Evaluate the diff for:"
# line with the "- " or "N. " items right below it.
review_areas() {
  awk '
    index($0, "description") == 1 { print; next }
    !inside && (index($0, "Focus on diff quality only:") == 1 || index($0, "Evaluate the diff for:") == 1) { inside = 1; print; next }
    inside && ($0 ~ /^- / || $0 ~ /^[0-9]+\. /) { print; next }
    inside { exit }
  ' "$1"
}

# area_values FILE: the report template's "Area recommended values" comment,
# from its first line through the line that closes it.
area_values() {
  awk '
    index($0, "Area recommended values") { inside = 1 }
    inside { print }
    inside && index($0, "-->") { exit }
  ' "$1"
}

# assert_part_contains_exact DESC PART FILE NEEDLE: case-sensitive; NEEDLE is
# in what the function PART prints for FILE.
assert_part_contains_exact() {
  _desc="$1"
  _part="$2"
  _file="$3"
  _needle="$4"
  if "$_part" "$_file" | grep -Fq -- "$_needle"; then
    record_pass "$_desc"
  else
    record_fail "$_desc (missing: $_needle in the $_part of $_file)"
  fi
}

# assert_part_lacks_any_case DESC PART FILE NEEDLE: ignores case; NEEDLE is
# not in what the function PART prints for FILE. An empty part fails, so a
# renamed marker line cannot pass by matching nothing.
assert_part_lacks_any_case() {
  _desc="$1"
  _part="$2"
  _file="$3"
  _needle="$4"
  _text="$("$_part" "$_file")"
  if [ -z "$_text" ]; then
    record_fail "$_desc (no $_part found in $_file)"
  elif printf '%s\n' "$_text" | grep -Fiq -- "$_needle"; then
    record_fail "$_desc (unexpected in any case: $_needle in the $_part of $_file)"
  else
    record_pass "$_desc"
  fi
}

# The five mechanical items, worded as in the canonical list
# (.claude/rules/ralph/model-routing.md, the implementer's report contract).
mechanical_items="unnecessary changes, typos and copy-paste errors, debug leftovers, secrets, and files outside the slice"

# Words that mark one of the four items /self-review leaves to the
# implementing side, if they show up among its review areas.
removed_area_words="unnecessary typo debug secret"

self_review_files="
.agents/skills/self-review/SKILL.md
.claude/skills/self-review/SKILL.md
templates/base/.agents/skills/self-review/SKILL.md
templates/base/.claude/skills/self-review/SKILL.md
.claude/agents/reviewer.md
templates/base/.claude/agents/reviewer.md
.codex/agents/reviewer.toml
templates/base/.codex/agents/reviewer.toml
"

for file in $self_review_files; do
  assert_contains "$file declares diff quality only" "diff quality only" "$file"
  assert_contains "$file forbids running tests" "Do NOT run tests" "$file"
  assert_contains "$file forbids static analysis" "static analysis" "$file"
  assert_contains "$file forbids broad unrelated scope" "broad unrelated" "$file"
  assert_contains "$file forbids repo audits" "repo audits" "$file"

  assert_not_contains "$file must not call run-test.sh" "run-test.sh" "$file"
  assert_not_contains "$file must not call run-static-verify.sh" "run-static-verify.sh" "$file"
  assert_not_contains "$file must not call run-verify.sh" "run-verify.sh" "$file"
done

# /self-review checks six areas; the four mechanical ones moved to the
# implementing side (the implementer's mechanical self-check before commit).
self_review_skill_files="
.agents/skills/self-review/SKILL.md
.claude/skills/self-review/SKILL.md
templates/base/.agents/skills/self-review/SKILL.md
templates/base/.claude/skills/self-review/SKILL.md
"

for file in $self_review_skill_files; do
  assert_not_contains_any_case "$file drops the unnecessary-changes area" "**Unnecessary changes**" "$file"
  assert_not_contains_any_case "$file drops the typos area" "**Typos and copy-paste errors**" "$file"
  assert_not_contains_any_case "$file drops the debug-code area" "**Debug code**" "$file"
  assert_not_contains_any_case "$file drops the secrets area" "**Secrets and credentials**" "$file"

  # The bold headings, not the bare words: assert_contains ignores case, and
  # the frontmatter description also says "null safety, exception handling".
  assert_contains "$file keeps the null safety area" "**Null safety and defensive checks**" "$file"
  assert_contains "$file keeps the exception handling area" "**Exception handling**" "$file"

  # The numbered list itself, so the review-area checks below read it.
  assert_part_contains_exact "$file review areas include null safety" \
    review_areas "$file" "3. **Null safety and defensive checks**"
  assert_part_contains_exact "$file review areas include exception handling" \
    review_areas "$file" "4. **Exception handling**"
done

# The reviewer definitions' focus list. Pinned by phrase, not by bare word:
# the sentence about the implementing side names the mechanical items too.
reviewer_md_files="
.claude/agents/reviewer.md
templates/base/.claude/agents/reviewer.md
"

for file in $reviewer_md_files; do
  assert_part_contains_exact "$file focus line lists the six areas" review_areas "$file" \
    "Focus on diff quality only: naming, readability, null safety, exception handling, security, and maintainability."
done

reviewer_toml_files="
.codex/agents/reviewer.toml
templates/base/.codex/agents/reviewer.toml
"

for file in $reviewer_toml_files; do
  assert_part_contains_exact "$file focus list has the null safety item" review_areas "$file" \
    "- null safety: null/empty mistakes and missing defensive checks visible in the diff"
  assert_part_contains_exact "$file focus list has the exception handling item" review_areas "$file" \
    "- exception handling: swallowed errors and missing error paths visible in the diff"
done

# None of the four mechanical items is listed as a review area: not in the
# description, the focus list, or the numbered list.
for file in $self_review_skill_files $reviewer_md_files $reviewer_toml_files; do
  for word in $removed_area_words; do
    assert_part_lacks_any_case "$file review areas leave out \"$word\"" review_areas "$file" "$word"
  done
  # The sentence that hands the mechanical items to the implementing side
  # uses the canonical wording.
  assert_joined_contains_exact "$file names the mechanical items in the canonical words" \
    "$mechanical_items" "$file"
done

# The report template's Area values carry the six areas only.
self_review_template_files="
.claude/skills/self-review/template.md
.agents/skills/self-review/template.md
templates/base/.claude/skills/self-review/template.md
templates/base/.agents/skills/self-review/template.md
"

for file in $self_review_template_files; do
  for word in $removed_area_words; do
    assert_part_lacks_any_case "$file Area values leave out \"$word\"" area_values "$file" "$word"
  done
  assert_part_contains_exact "$file Area values keep null-safety" area_values "$file" "null-safety"
  assert_part_contains_exact "$file Area values keep exception-handling" area_values "$file" "exception-handling"
done

implementer_files="
.claude/agents/implementer.md
templates/base/.claude/agents/implementer.md
.codex/agents/implementer.toml
templates/base/.codex/agents/implementer.toml
"

for file in $implementer_files; do
  assert_contains "$file runs the mechanical self-check" "Mechanical self-check before commit" "$file"
  assert_joined_contains_exact "$file lists the five items in the canonical words" \
    "check five items: $mechanical_items (the handoff's files in scope)." "$file"
  assert_joined_contains_exact "$file reports an item it cannot fix" \
    "When an item cannot be fixed within scope, leave it and report it as found and not fixed, with the reason." "$file"
  assert_joined_contains_exact "$file stops before committing a secret it cannot remove" \
    "if a secret in the staged diff cannot be removed within scope, do not commit; unstage it and stop, and report it." "$file"
  assert_joined_contains_exact "$file report contract gives a result per item" \
    "mechanical self-check result: for each of the five items, one of clean, fixed (what you fixed), or found and not fixed (what you found, and why it could not be fixed within scope)" "$file"
  assert_joined_contains_exact "$file says why the checks matter without history" \
    "These are the checks \`/self-review\` leaves to the implementing side, so do not skip them." "$file"
  assert_not_contains_any_case "$file has no change-history wording" "no longer" "$file"
  assert_not_contains_any_case "$file does not point at items by position" "first four" "$file"
done

printf '\n-- Summary --\n'
printf '  PASS: %d / %d\n' "$_pass" "$_total"
printf '  FAIL: %d\n' "$_fail"

if [ "$_fail" -gt 0 ]; then
  exit 1
fi
