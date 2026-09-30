# sync-docs report: check-template-scaffold-green

- Date: 2026-10-01
- Plan: `docs/plans/active/2026-09-30-check-template-scaffold-green.md` (issue #189)
- Prior reports: `docs/reports/self-review-2026-09-30-check-template-scaffold-green.md`
  (Pass, recommend merge, MEDIUM 1 + LOW 8, all fixed in Slice B `8fd5113`),
  `docs/reports/verify-2026-10-01-check-template-scaffold-green.md` (Pass),
  `docs/reports/test-2026-10-01-check-template-scaffold-green.md` (Pass)

## Summary

`scripts/check-template.sh` (and its byte-identical `templates/base/scripts/check-template.sh`
copy) now trims `required_files` to the 25 entries a `ralph init` scaffold
actually receives, reads the settings hook-reference check and the three
`find`-driven structure checks (executable bit, `SKILL.md`, agent
frontmatter) through temp files instead of a `| while` pipeline subshell so
`fail()`'s `status=1` reaches the exit code, and reports unreadable inputs
as `FAIL` instead of silently passing. No CLI flag, required-file *set* for
any other tool, or CI workflow step name changed — only this one script's
internal checks and its `required_files` list.

This pass checked every documentation surface the task handoff named for
drift and found none. No doc edits were needed beyond adding this pass's
own Deviation-notes bullet to the plan.

## Documentation drift checked (no changes needed)

- **`docs/quality/quality-gates.md`** (root, line 49-50, and
  `templates/base/docs/quality/quality-gates.md`, line 49) — both mentions
  of `check-template.sh` are generic ("required-file / structure check",
  a CI-workflow pointer); neither states a required-file count, describes
  the old fail-open subshell bug, or implies the scaffold's PR CI step was
  failing. The root/template wording difference (separate
  `check-template.yml` job vs. a step inside `verify.yml`) pre-dates this
  fix and is a `check-sync.sh` `KNOWN_DIFF` documented at
  `docs/tech-debt/README.md:117` — not something this pass should
  reunify.
- **`docs/quality/definition-of-done.md`** — no mention of
  `check-template.sh` or `check_template`.
- **`docs/architecture/repo-map.md`** — the one mention (`scripts/`
  inventory line) lists `check-template.sh` alongside other drift-check
  scripts with no description of its internals; no template counterpart
  file exists to check.
- **`README.md`**, **`AGENTS.md`** (root and template copies) — no mention
  of `check-template.sh` / `check_template`.
- **`docs/recipes/`** — no file mentions `check-template.sh` /
  `check_template`.
- **`scripts/bootstrap.sh`** (read-only per task; meta-repo only, not
  templated) — lines 51-59 print a generic
  `[ok] Template structure check passed.` / `[warn] Template structure
  check found issues (see above).` pair with no reference to specific
  checks or the `required_files` count. No drift, no edit made (script
  edits are out of scope for this pass regardless).
- **`docs/tech-debt/README.md`** — `grep -n '189\|check-template'` finds
  no `#189` row (the plan's Slice A already removed it) and no stray
  reference to the old fail-open behavior; the one `check-template`-adjacent
  hit (line 117) is the pre-existing `quality-gates.md` `KNOWN_DIFF` note
  above, unrelated to #189.
- **`docs/evidence/codex-config-rewrite-2026-09-30.md`** and the archived
  `#183` plan — left untouched, as instructed; these are historical
  records of a different issue, not live documentation of
  `check-template.sh`'s current behavior.

## Files changed in this pass

- **`docs/plans/active/2026-09-30-check-template-scaffold-green.md`** —
  added one Deviation-notes bullet for this `/sync-docs` pass, dated
  2026-10-01, matching the per-step-bullet style already used for
  work/self-review/verify/test in the same section. No AC checkbox or
  Progress-checklist box touched (the template has no "documentation
  artifact" slot, and none was missing here).
- **`docs/insights/events/2026-09-30-check-template-scaffold-green.jsonl`** —
  appended via `./scripts/insights-append.sh` (sync_docs, cycle 1,
  verdict pass).

## Gates

```
$ ./scripts/check-skill-sync.sh        # [ok] check-skill-sync: 13 skill(s) in lock-step
$ ./scripts/check-sync.sh              # PASS: all files in sync. IDENTICAL: 159, DRIFTED: 0, ROOT_ONLY: 0, TEMPLATE_ONLY: 11, KNOWN_DIFF: 5
$ ./scripts/check-template-purity.sh   # PASS: no meta-repo-specific references found in templates.
```

No skill body or `templates/base` mirror was touched this pass (the diff
under review only ever touched `scripts/check-template.sh` and its
byte-identical template copy, plus tests/docs already verified), so
`sync-skills.sh` was not re-run; the three gates above were re-run
directly against the current worktree state rather than trusting the
verify report's word.

## Walkthrough

`git diff main...HEAD --stat` (excluding `docs/plans/`, `docs/reports/`,
`docs/insights/`): 6 files, 427 insertions(+), 35 deletions(-) — 462
changed lines, under the 500-line rule-of-thumb. Not writing a walkthrough
report in this pass; per the task handoff, `/pr` decides whether one is
warranted.

```
docs/tech-debt/README.md                 |   1 -
internal/scaffold/embed_test.go          |   2 +-
scripts/check-template.sh                |  81 +++++++--
scripts/verify.local.sh                  |   2 +-
templates/base/scripts/check-template.sh |  81 +++++++--
tests/test-check-template.sh             | 295 +++++++++++++++++++++++++++++--
6 files changed, 427 insertions(+), 35 deletions(-)
```

## Known gaps

- Did not re-run `./scripts/run-verify.sh`, `./scripts/run-test.sh`,
  `bash tests/test-check-template.sh`, or `go test ./internal/scaffold/...`;
  that is `/verify`'s and `/test`'s completed job (both cited above, Pass).
- Did not re-audit the pre-existing `quality-gates.md` root/template
  divergence itself (the `KNOWN_DIFF` reason, not this fix, governs it);
  flagged only as context above, not as a finding.
