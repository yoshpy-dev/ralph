# Sync-docs report: codex-exec-stdin-and-model

- Date: 2026-09-30
- Plan: `docs/plans/active/2026-09-29-codex-exec-stdin-and-model.md` (issue #184)
- Agent: `doc-maintainer` subagent (cycle 1)
- Branch: `fix/codex-exec-stdin-and-model`, HEAD `b045ccb`

## Checked

- `README.md`: grepped for `codex` (37 hits) and `codex exec` specifically
  (lines 249, 285). Both are name-only mentions (`calls \`codex exec
  review\``, a hook-trust reminder) that describe *that* Codex is invoked,
  not the invocation form (flags, stdin, model/effort, background/watchdog
  contract). Neither sentence claims anything this fix changed, so neither
  is affected.
- `.claude/rules/ralph/post-implementation-pipeline.md` and its template
  copy (`templates/base/.claude/rules/ralph/post-implementation-pipeline.md`,
  confirmed present): the `/cross-review` row (line 24, both files) reads
  `inline; calls \`codex exec review\`` — name-only, unaffected.
- `.claude/rules/ralph/ralph-workflow.md` and its template copy (confirmed
  present): read lines 20-35 in full context. "Codex advisory is optional.
  If the `codex` binary is available, ... If unavailable, the step is
  silently skipped" describes the *binary-missing* case, which this fix does
  not touch. Cross-checked against the skill bodies themselves
  (`.claude/skills/cross-review/SKILL.md:7,49-50`,
  `.claude/skills/plan/SKILL.md:67`): the "binary unavailable → silently
  skip to /pr / to completion" path is worded identically to before and is
  a distinct code path from the new "binary available but the call times
  out or returns rc≠0 or an empty `-o` file → incomplete" path this fix
  added. No contradiction; no edit needed.
- `.claude/rules/ralph/subagent-policy.md` (cross-review triage section,
  lines 74-92): mentions of `Codex advisory response` and
  `.codex/agents/` custom agents are about delegation/dispatch, not the
  `codex exec` invocation shape. No drift.
- `docs/quality/definition-of-done.md`: only one Codex-related line (line
  20, "both agent surfaces were exercised"), generic and unaffected.
  `docs/quality/quality-gates.md`: zero `codex` hits.
- `docs/insights/README.md`: already documents `verdict` as `pass`, `fail`,
  `complete`, `action_required`, or **`n/a`** (line 28) — the value
  `/cross-review`'s new incomplete path calls via
  `insights-append.sh --verdict n/a`. Confirmed via
  `git diff main...HEAD -- docs/insights/README.md`: this file is untouched
  by the branch, so `n/a` was already a valid, documented value before this
  plan started. No schema change needed or made.
- `docs/tech-debt/README.md`: confirmed the one row this plan's Slice B
  added (line 142, the `parseCrossReviewAllCycles`/`crossReviewVerdict`
  backfill gap) via `git diff main...HEAD`. Column count: 6 pipes / 5 cells,
  matching the table's header row (also 6/5). Wording accurately describes
  the residual gap (backfill can't read the triage report's `Reviewer
  status:` header, so an incomplete run ingested via backfill — as opposed
  to a live `/cross-review` run, which appends its own `n/a` event directly
  — reads as `pass`). No other row in the file references `codex exec`,
  stdin, or model/effort in a way this fix touches.
- `.codex/README.md` and its template copy: the one `codex exec review`
  mention (line 63) is the same Known-differences table row as README.md's,
  name-only. The other four `codex exec` mentions (lines 155-171) are about
  the separate hook-trust-approval mechanism (non-interactive `codex exec`
  silently skipping unapproved hooks) — unrelated to stdin/model/effort.
  No drift.
- Repo-wide sweep (`grep -rn "codex exec" --include="*.md" .`) for any
  invocation example still missing `</dev/null`: all hits outside
  `docs/reports/`, `docs/plans/`, and this plan's own updated files
  (`docs/recipes/codex-setup.md` ×2 copies, already updated in Slice A/B)
  are either (a) name-only mentions already covered above, or (b) dated,
  frozen historical records (`docs/evidence/codex-hooks-*.md`,
  `docs/specs/2026-05-07-codex-cli-parity.md`,
  `docs/specs/2026-08-01-org-runtime.md`, three `docs/tech-debt/README.md`
  rows about unrelated resolved/open issues — hook-trust semantics, a
  retired Loop driver, a Codex model-arg limitation) describing what was
  true when written, not living guidance about the current call shape.
  None claim the current invocation form.
- Cross-checked against the verify report's own equivalent sweep
  (`docs/reports/verify-2026-09-30-codex-exec-stdin-and-model.md`, AC-5 row
  and Verdict section): same conclusion, no doc drift in README.md,
  `post-implementation-pipeline.md`, `docs/quality/`, or AGENTS.md.
- Gates: `./scripts/check-skill-sync.sh` (13 skills in lock-step),
  `./scripts/check-sync.sh` (159 IDENTICAL / 0 DRIFTED / 0 ROOT_ONLY /
  11 TEMPLATE_ONLY / 5 KNOWN_DIFF, PASS), `./scripts/check-template-purity.sh`
  (PASS, no meta-repo-specific references in templates).

## Stale

None found.

## Changed

None. No edits made in this cycle — every doc location the task named was
already in sync with the shipped behavior at HEAD `b045ccb`.

## Left

None outstanding from this cycle's scope. The verify report's own
documentation-accuracy gap (the plan's 2026-09-29 self-review deviation
note overclaiming "全件を in-cycle で修正" when one LOW — cross-skill
duplication of the `</dev/null`/`command` rationale and the completion-
contract paragraph between `/plan` and `/cross-review` — is still open) is
outside this task's assignment, which named specific files to check for
drift and did not include the plan file or skill bodies themselves. Not
acted on here.

## Verdict

No drift found. No files changed in this cycle.
