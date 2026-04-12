# Self-review report: spec-skill

- Date: 2026-04-12
- Plan: docs/plans/active/2026-04-12-spec-skill.md
- Reviewer: reviewer agent (claude-sonnet-4-6)
- Scope: diff quality only — naming, readability, unnecessary changes, typos, null safety, debug code, secrets, exception handling, security, maintainability

## Evidence reviewed

- `git diff main...HEAD` for all 5 changed files
- `.claude/skills/spec/SKILL.md` (new, 98 lines)
- `.claude/skills/spec/template.md` (new, 70 lines)
- `CLAUDE.md` (2-line addition)
- `AGENTS.md` (1-line addition)
- `docs/plans/active/2026-04-12-spec-skill.md` (new, 117 lines)
- Compared against `.claude/skills/plan/SKILL.md` as the closest existing peer skill
- Inspected `docs/tech-debt/README.md` for pre-existing debt patterns

## Findings

| Severity | Area | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- |
| MEDIUM | security | `gh issue create --body "$(cat docs/specs/<date>-<slug>.md)"` passes an untrusted file path into command substitution inside double quotes. A spec file containing backticks or `$(...)` in its content would be shell-expanded before being passed to `gh`. This violates the project's safe-quoting rule. | `SKILL.md` line 59: `gh issue create --title "<spec title>" --body "$(cat docs/specs/<date>-<slug>.md)"` | Use `--body-file docs/specs/<date>-<slug>.md` (gh CLI flag) instead of command substitution. See `.claude/rules/git-commit-strategy.md` "Safe Quoting" section. |
| MEDIUM | maintainability | The plan file at `docs/plans/active/2026-04-12-spec-skill.md` is included in the diff as a new file, but its Progress checklist shows all steps unchecked (`[ ] Plan reviewed`, `[ ] Branch created`, etc.). The plan was committed in an inaccurate state — the branch exists and implementation is complete. | Plan lines 111–117 | Update the checklist before (or immediately after) this review: mark "Plan reviewed", "Branch created", and "Implementation started" as done. |
| LOW | naming | `template.md` uses `__TITLE__`, `__DATE__`, `__ISSUE__` as placeholder tokens. The peer skill `.claude/skills/plan/template.md` uses the same `__TITLE__` convention, so this is locally consistent. However, `__ISSUE__` is ambiguous — it could mean a GitHub issue number, a URL, or `N/A`. | `template.md` line 4: `- Related issue: __ISSUE__` | Rename to `__ISSUE_URL_OR_NA__` or add a comment: `<!-- GitHub issue URL or N/A -->`. |
| LOW | readability | In `SKILL.md` Step 8, the fallback for option 1 (issue-only) states "still save the file as fallback before attempting issue creation". This introduces a subtle ordering dependency: the file must be saved first, then the issue creation attempted. The instruction is clear in prose but would benefit from a numbered sub-list to match the formatting of sibling option blocks. | `SKILL.md` lines 66–69 | Convert the option-1 fallback to a numbered sub-list (a, b, c) consistent with option-1's main success path (a, b). |
| LOW | readability | Step 7's `AskUserQuestion` option text is written entirely in Japanese while the rest of SKILL.md is in English. The peer `plan/SKILL.md` uses Japanese only for the question/options (line 24–28), which is an established pattern. This is acceptable but worth noting as it may confuse tooling that parses SKILL.md programmatically. | `SKILL.md` lines 53–60 | No change required if the Japanese/English mixing is intentional for user-facing text; document the convention explicitly if not already. |
| LOW | null-safety | In Step 8, the `gh issue create` command embeds `docs/specs/<date>-<slug>.md` by path, but there is no guard ensuring the file exists before the `gh` call. If Step 5 (write file) failed silently, the `gh` command would receive an empty body. | `SKILL.md` lines 57–70 | Add an explicit existence check: `if [[ ! -f "docs/specs/<date>-<slug>.md" ]]; then abort with warning`. |

## Positive notes

- The frontmatter is complete and matches the acceptance criteria: `name`, `description`, `disable-model-invocation: true`, and all required `allowed-tools` are present.
- Role separation table between `/spec` and `/plan` is clear and mirrored in both SKILL.md and the plan — reduces confusion for readers new to the workflow.
- The anti-bottleneck section is included and references the existing `anti-bottleneck` skill correctly, consistent with `plan/SKILL.md`.
- Fallback behavior for `gh` failures is explicitly defined with a user-facing warning message in Japanese, which matches the project's user-communication style.
- `CLAUDE.md` and `AGENTS.md` changes are minimal and targeted — no unrelated whitespace or formatting modifications.
- `template.md` section structure (Overview, Background, Requirements, Constraints, User stories, Dependencies, Research findings, Open questions, References) matches the acceptance criteria list exactly.
- No secrets, credentials, debug code, or commented-out blocks found anywhere in the diff.

## Tech debt identified

| Debt item | Impact | Why deferred | Trigger to pay down | Related plan/report |
| --- | --- | --- | --- | --- |
| `gh issue create --body "$(cat ...)"` shell quoting risk in SKILL.md. Spec files with backticks or `$()` in body text would expand at shell invocation time. | MEDIUM: potential secret expansion in issue body if spec contains shell-like text | Skill files are prose templates, not enforced code — fixing requires `--body-file` flag or alternative | When `spec` skill is exercised in production or when commit-msg-guard catches an instance | docs/reports/self-review-2026-04-12-spec-skill.md |

_(Appended to `docs/tech-debt/README.md`.)_

## Recommendation

- Merge: **YES with follow-ups** — no CRITICAL findings. One MEDIUM finding (shell quoting) should be fixed before the skill is used in production, but it does not block merging the skill definition itself.
- Follow-ups:
  1. Replace `--body "$(cat ...)"` with `--body-file <path>` in `SKILL.md` Step 8 (gh CLI supports this flag natively).
  2. Update the plan progress checklist to reflect actual branch/implementation state.
  3. Consider renaming `__ISSUE__` placeholder to `__ISSUE_URL_OR_NA__` for clarity.
  4. Add a file-existence guard before the `gh issue create` invocation.
