# Sync-docs report: local-branch-secret-scan

- Date: 2026-09-24
- Plan: `docs/plans/active/2026-09-20-local-branch-secret-scan.md` (issue #169)
- Agent: `doc-maintainer` subagent (Claude Code), pipeline cycle 1
- Scope: documentation sync only. Reference reports: `docs/reports/self-review-2026-09-21-local-branch-secret-scan.md` (12 findings, fixed in `3afac6e`), `docs/reports/verify-2026-09-22-local-branch-secret-scan.md` (pass), `docs/reports/test-2026-09-24-local-branch-secret-scan.md` (pass, two gaps named).
- Precondition confirmed: HEAD `9554a58` (docs: add test report for local-branch-secret-scan), `git status --porcelain` empty at start.

## Surfaces checked, no change needed

Every surface below was read and compared against the code at HEAD; none was stale.

- `README.md` — the "Deterministic hooks" table row's "commit-msg secret scan" names `commit-msg-guard.sh`, a per-commit git hook, a different mechanism from this branch's history-range scan; the `scripts/` tree summary already ends in "etc." and does not enumerate scripts elsewhere either.
- `templates/base/README.md` — does not exist. `README.md` documents the ralph meta-repo itself and is not part of what `ralph init` scaffolds (confirmed: no `templates/base/README.md`, and no scaffold-manifest reference to shipping one).
- `docs/quality/definition-of-done.md` — no mention of secret scanning at all (neither the pre-existing CI scan nor the new local one); this is a generic per-phase checklist that was already this abstract before this branch, not a regression.
- `.claude/rules/ralph/git-commit-strategy.md` — the "Enforcement" sentence under Safe Quoting names `pre_bash_guard.sh` and `commit-msg-guard.sh`, both per-commit guards for a different concern (safe quoting / leaked secrets in the commit message itself), not the branch-history range scan this plan adds. Still accurate and complete for what it describes.
- `.claude/rules/ralph/ralph-workflow.md` — "Run `./scripts/run-verify.sh` ... before claiming success" is generic by design; the branch scan is now included transparently as one of `run-verify.sh`'s steps, which is the whole point of wiring it there instead of updating every caller.
- `.claude/skills/work/SKILL.md` (steps referencing `run-verify.sh`, Completion gate) — same generic-reference reasoning; no enumeration of what `run-verify.sh` checks exists there to go stale.
- `docs/recipes/` — only `adding-a-language-pack.md` matches `secret`, and that hit is its own unrelated "secret-bearing files" `.gitignore`-block note, not in scope for this diff.
- `.ralph/core/AGENTS.core.md` — the scaffolded-`AGENTS.md` managed-block source. It has no `scripts/` enumeration or repo-map section at all (56 lines, Mission/Primary loop/Source of truth/Verification/Hard rules only); the detailed `scripts/` bullet with the new `secret-scan-branch.sh` mention lives in the meta-repo's own `AGENTS.md` "Repo map" section (a meta-repo addition outside the managed block), which was already updated per the plan's Deviation notes. Nothing to add to the managed-block source.
- `docs/quality/quality-gates.md` + template copy — both already carry the branch-scan bullet, identical in wording (root omits `(issue #169)`, matching the self-review LOW fix already applied), confirmed via `diff`.
- `.claude/skills/pr/SKILL.md` + 3 mirrors — Step 3 (strict scan before push), Step 8 (re-scan before the archive-commit push), and the Completion gate all read correctly against `scripts/secret-scan-branch.sh`'s actual exit-code contract at HEAD. `cmp` confirms byte-identity across all four faces.
- `AGENTS.md` `scripts/` bullet — already names `secret-scan-branch.sh` ("branch-history secret scan `secret-scan-branch.sh`").

## `.gitallowed` guidance (task item 2)

No standalone doc explains how to write an allowlist regex; the documentation is intentionally the tool itself, and it is now internally consistent:

- `scripts/secret-scan.sh`'s on-finding guidance text (added in this plan) tells the reader to write the key name as a bracket expression (e.g. `api_ke[y]`) so the allowlist line does not match a scanner pattern, and explains why (a branch-history scan also reads the commit that adds the allowlist line).
- `.gitallowed`'s own header comments already document the same bracket-expression convention with worked examples, and were written before this plan.

Both agree. No new doc was created — consistent with the plan's own framing that the scanner's finding-time output is the documentation.

## `docs/tech-debt/README.md` (task item 3)

Added one new row for (a): `scripts/check-template.sh`'s `required_files` list has no regression test — confirmed via the test report's mutation 6f (dropping `scripts/xreview-helpers.sh` from the list caused zero test failures anywhere in the suite, and `check-template.sh` itself still reported success). Framed as pre-existing and not introduced by this plan, per the test report's own judgment.

(b) — the symlinked-`.gitallowed` cosmetic notice quirk — was **not** added as a register row, per the handoff's own default: it is cosmetic, not security-relevant (scan results stay correct), and not tied to a named acceptance criterion. It is already recorded in the test report's "Edge cases explored" and "Test gaps" sections, which is sufficient traceability for something this narrow (nobody has a legitimate reason to symlink `.gitallowed`).

## Plan readability (task item 4)

Scope row 1 (`docs/plans/active/2026-09-20-local-branch-secret-scan.md`) now opens with a "現在の契約(要約)" (current contract, summarized) sentence stating the final exit-code rules in one place, before the "Codex advisory" / "self-review cycle 1" revision history that follows it. The revision history was left intact — nothing was deleted, and no AC text was touched. Progress checklist: ticked "Verification artifact created" and "Test artifact created" (both now exist and pass). Added one Deviation notes bullet each for the `/verify` pass, the `/test` pass, and this `/sync-docs` pass, so the plan's own history stays complete without requiring a reader to cross-reference the reports directory to know what happened after the self-review cycle.

## Insight events (task item 5)

Three per-day event files exist for this slug (`docs/insights/events/2026-09-{20,22,24}-local-branch-secret-scan.jsonl`) because `scripts/insights-append.sh` names its output file `<events-dir>/<UTC-date-of-the-append-call>-<slug>.jsonl` — the date is computed fresh on every invocation (`_date="$(date -u '+%Y-%m-%d')"`), not pinned to the task's start date. `docs/insights/README.md`'s "File layout" section documents exactly this naming pattern (`<UTC-date>-<slug>.jsonl`), and its "Why per-task files?" rationale (avoiding conflicting concurrent appends across branches) holds regardless of whether one task produces one file or several, since no two branches ever touch the same file either way. Left as three files; no merge and no doc change.

## Gate results

| Gate | Result |
| --- | --- |
| `./scripts/check-skill-sync.sh` | PASS — 13 skills in lock-step |
| `./scripts/check-sync.sh` | PASS — IDENTICAL 159 / DRIFTED 0 / ROOT_ONLY 0 / TEMPLATE_ONLY 11 / KNOWN_DIFF 5 (unchanged from the verify report; `docs/tech-debt/README.md` has no template counterpart, `.gitkeep` only) |
| `./scripts/check-template-purity.sh` | PASS — no meta-repo-specific references in templates |
| `./scripts/check-template.sh` | PASS (exit 0) — prints the same 7 pre-existing `FAIL: ... references missing hook` lines the verify report already reproduced against `main`'s own copy of the script; not introduced or affected by this change |
| `go test ./internal/org/ -run TestSendDefaults -count=1` | PASS |
| `./scripts/secret-scan.sh --staged` (before commit) | clean, exit 0 |
| `./scripts/secret-scan-branch.sh --strict` (after commit) | see commit section below |

No skill, script, `.gitallowed`, or `.github/workflows` file was touched — only `docs/plans/active/2026-09-20-local-branch-secret-scan.md` and `docs/tech-debt/README.md` — so `sync-skills.sh` and the template-copy step were not needed.

## Commit

`docs: sync docs for local-branch-secret-scan` — files: `docs/plans/active/2026-09-20-local-branch-secret-scan.md`, `docs/tech-debt/README.md`, this report.
