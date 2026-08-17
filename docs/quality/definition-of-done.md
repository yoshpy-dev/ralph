# Definition of done

A task is done only when all applicable items are satisfied.

## For non-trivial code changes (standard /work flow)

- [ ] Active plan exists or was explicitly deemed unnecessary
- [ ] Repo writes happened inside a clean-base task worktree, not the default checkout
- [ ] Acceptance criteria were addressed
- [ ] Each implementation slice is individually committed (see `.claude/rules/ralph/git-commit-strategy.md`)
- [ ] Self-review artifact exists in `docs/reports/` (diff quality only; no test/static/spec/doc-drift execution)
- [ ] Verification was run and recorded in `docs/reports/` (spec compliance + static analysis via `./scripts/run-static-verify.sh`; changed-language scope by default; no tests)
- [ ] Test artifact exists in `docs/reports/` (behavioral tests via `./scripts/run-test.sh`; changed-language scope by default; no static analysis)
- [ ] Docs and contracts were updated if behavior changed (`/sync-docs`)
- [ ] Remaining gaps are explicit
- [ ] PR created via `/pr` skill (includes plan archival, hand-off, and task worktree/local branch cleanup)
- [ ] PR title starts with the branch type prefix (`feat/...` -> `feat: ...`) and `./scripts/ensure-pr-title-prefix.sh` passed
- [ ] CI verify passes on the PR
- [ ] Skill mirror sync is green (CI-enforced via `./scripts/check-skill-sync.sh` in verify.yml; regenerate the mirror with `./scripts/sync-skills.sh` if needed)
- [ ] If the change touches `.claude/`, `.codex/`, `.agents/skills/`, or shared rules, both agent surfaces were exercised (or the gap is recorded explicitly)

### Post-implementation pipeline order

The full pipeline must run in this order — no steps may be skipped:

```
/self-review → /verify → /test → /sync-docs → /cross-review → /pr
```

If `/cross-review` finds ACTION_REQUIRED issues and the user chooses to fix them, the **full pipeline** re-runs from `/self-review` through `/cross-review` again. `/sync-docs` must not be skipped in the re-run.

The pipeline is capped at **2 total runs by default** (initial + 1 re-run), controlled by `RALPH_STANDARD_MAX_PIPELINE_CYCLES` (default `2`). See `.claude/rules/ralph/post-implementation-pipeline.md` for cap semantics and state files.

Phase boundaries are part of the definition of done. A passing pipeline that
runs tests during `/verify`, static analysis during `/test`, or broad
verification work during `/self-review` is not valid.

## For org runtime tasks (`ralph org`)

Autonomous multi-seat execution is a separate surface from the standard
`/work` flow above; see `docs/specs/2026-08-01-org-runtime.md` and
`.claude/rules/ralph/agent-messaging.md` for its own definition of done (roster
status, manifest events, watchdog alerts).

## For risky or broad changes

Add:
- [ ] Walkthrough included in PR or `docs/reports/`
- [ ] Rollback note or recovery path
- [ ] Known follow-ups or tech debt recorded

## For docs-only changes

- [ ] Source of truth is aligned
- [ ] No commands or workflows became stale
- [ ] Any changed process still matches scripts and rules

- Evidence files committed under `docs/evidence/` must redact the local home directory (`$HOME` → `~`) before commit; session UUIDs and other machine-local identifiers should be trimmed when they carry no evidentiary value.
