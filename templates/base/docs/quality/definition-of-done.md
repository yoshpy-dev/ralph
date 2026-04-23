# Definition of done

A task is done only when all applicable items are satisfied.

## For non-trivial code changes (standard /work flow)

- [ ] Active plan exists or was explicitly deemed unnecessary
- [ ] Acceptance criteria were addressed
- [ ] Each implementation slice is individually committed (see `.claude/rules/git-commit-strategy.md`)
- [ ] Self-review artifact exists in `docs/reports/` (diff quality)
- [ ] Verification was run and recorded in `docs/reports/` (spec compliance + static analysis)
- [ ] Test artifact exists in `docs/reports/` (behavioral tests pass)
- [ ] Docs and contracts were updated if behavior changed (`/sync-docs`)
- [ ] Remaining gaps are explicit
- [ ] PR created via `/pr` skill (includes plan archival and hand-off)
- [ ] CI verify passes on the PR

### Post-implementation pipeline order

The full pipeline must run in this order — no steps may be skipped:

```
/self-review → /verify → /test → /sync-docs → /codex-review → /pr
```

If `/codex-review` finds ACTION_REQUIRED issues and the user chooses to fix them, the **full pipeline** re-runs from `/self-review` through `/codex-review` again. `/sync-docs` must not be skipped in the re-run.

The pipeline is capped at **2 total runs by default** (initial + 1 re-run). Standard flow uses `RALPH_STANDARD_MAX_PIPELINE_CYCLES` (default `2`), Ralph Loop uses `RALPH_MAX_OUTER_CYCLES` (default `2`). See `.claude/rules/post-implementation-pipeline.md` for cap semantics and state files.

## For Ralph Loop (/loop)

- [ ] Directory-based plan exists under `docs/plans/active/<date>-<slug>/`
- [ ] `_manifest.md` has shared-file locklist and dependency graph
- [ ] Each `slice-*.md` has self-contained AC, affected files, and verify/test plan
- [ ] All slice pipelines completed (`ralph status` shows all slices `complete`)
- [ ] Sequential merge to integration branch passed without conflicts
- [ ] Integration pipeline passed on merged branch (`--skip-pr --fix-all`)
- [ ] Unified PR created from `integration/<slug>` to base branch
- [ ] Plan directory archived from `docs/plans/active/` to `docs/plans/archive/`

Ralph Loop handles the full lifecycle autonomously per slice (implement → self-review → verify → test → sync-docs → codex-review), then merges slices into the integration branch, runs an integration pipeline (`--skip-pr --fix-all`) to catch cross-module issues, and creates a unified PR.

**Pipeline report output:** Each pipeline agent (self-review, verify, test) writes reports to both `.harness/state/pipeline/` (for orchestrator consumption) and `docs/reports/` (for PR pre-checks and human review). This dual-write ensures pipeline artifacts are available for the same quality checks as the standard flow.

## For risky or broad changes

Add:
- [ ] Walkthrough included in PR or `docs/reports/`
- [ ] Rollback note or recovery path
- [ ] Known follow-ups or tech debt recorded

## For docs-only changes

- [ ] Source of truth is aligned
- [ ] No commands or workflows became stale
- [ ] Any changed process still matches scripts and rules
