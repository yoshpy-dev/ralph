# Sync-docs report: doctor-agmsg-store-in-project

- Date: 2026-09-27
- Plan: `docs/plans/active/2026-09-26-doctor-agmsg-store-in-project.md` (issue #170)
- Agent: `doc-maintainer` subagent (cycle 1)
- Branch: `fix/doctor-agmsg-store-in-project`, base HEAD `9d22ac7` (base `main` at `88e7429`)

## Checked

- `docs/recipes/codex-seat-permissions.md` and its `templates/base/` copy — the
  in-project-narrowing sentence and the `.git`/`.agents`/`.codex` exception,
  against `codexWritableRootDetail`'s final wording
  (`internal/cli/doctor_codex_writable_root.go:823-829,835-840`).
- `.claude/skills/org/SKILL.md`, `.agents/skills/org/SKILL.md`, and both
  `templates/base/` mirrors — the equivalent Japanese sentence, four-way
  `diff`.
- `docs/tech-debt/README.md` — the existing `checkCodexAgmsgWritableRoot` row
  (#170 sentence already present from Slice B/C).
- `README.md` and `docs/specs/2026-08-01-org-runtime.md` — grepped for any
  description of this doctor check; none found.
- `docs/quality/definition-of-done.md` and `docs/quality/quality-gates.md` —
  grepped for `doctor`; no hits, nothing to update.
- `docs/plans/active/2026-09-26-doctor-agmsg-store-in-project.md` — Progress
  checklist and Deviation notes against the actual commit history and the
  test report's own findings.

## Stale

- None of the six doc copies (recipe x2, skill x4) had drifted from the final
  code wording. All four `codexRootCoverage`/`containingProject` claims —
  "inside this project", the `.git`/`.agents`/`.codex` exception, "contains
  it can already write it", and "does not contain it ... (a task worktree,
  for example)" — matched the code exactly.
- `docs/tech-debt/README.md`'s existing #170 row was accurate but did not yet
  record the two coverage gaps the tester's report named. Not a wording
  drift, but a missing entry (see Changed).
- The plan's Progress checklist had `Verification artifact created` and
  `Test artifact created` still unchecked even though both reports
  (`verify-2026-09-27-...md` at `ebb112d`, `test-2026-09-27-...md` at
  `9d22ac7`) already exist.
- The plan's `再開の手順` section had no marker distinguishing it from live
  guidance now that the pause it describes is over.

## Changed

- `docs/tech-debt/README.md` — appended one new row (existing five-column
  format) for the two test-report gaps: (a)
  `TestCheckCodexAgmsgWritableRoot_StoreEqualsProjectDir_Warn` only asserts
  the "inside this project" substring, not which half of
  "contains it can already write it" / "does not contain it ... cannot send
  RESULT" attaches to which clause, so a swap of those two halves in only
  that one format string would stay green; (b) `ralph doctor` run from a
  project subdirectory with no `ralph.toml` of its own never reaches this
  check's warn branch (`runDoctorFull(".")` does no upward search), which
  predates and is unrelated to #170.
- `docs/plans/active/2026-09-26-doctor-agmsg-store-in-project.md`:
  - Added a Deviation notes bullet for test cycle 1 (`9d22ac7`, PASS; 5
    mutations, 4 fully discriminating and one narrow gap; live comparison
    against the main binary byte-identical outside the project; the paused
    run's draft report was not reused).
  - Added a Deviation notes bullet for this sync-docs pass (drift check
    result, the new tech-debt row).
  - Ticked `Verification artifact created` and `Test artifact created` in
    the Progress checklist (both artifacts already exist; AC checkboxes were
    left untouched).
  - Added one line at the top of `再開の手順` noting the run resumed on
    2026-09-27 and the section is historical.

## Left

- The plan's AC checkboxes (`AC-1`..`AC-8`) — out of this step's scope per
  the task assignment; the verify report already maps each to evidence.
- The `checkCodexAgmsgWritableRoot` tech-debt row's own text — accurate as
  written for #170's shipped behavior; the new row is additive, not a
  rewrite of the existing one.
- No spec, README, or `/org`-runtime spec changes were needed; the grep in
  Checked found nothing describing this check outside the six doc copies
  and the tech-debt row.

## Diff size (internal/)

```
$ git diff main...HEAD --stat -- internal/
 internal/cli/doctor.go                             |  10 +-
 internal/cli/doctor_codex_writable_root.go         |  66 +++-
 internal/cli/doctor_codex_writable_root_test.go    | 417 ++++++++++++++++++---
 internal/cli/doctor_codex_writable_root_unix_test.go |   4 +-
 4 files changed, 436 insertions(+), 61 deletions(-)
```

## Gates

- `./scripts/check-skill-sync.sh` — PASS
- `./scripts/check-sync.sh` — PASS
- `./scripts/check-template-purity.sh` — PASS
- `./scripts/secret-scan.sh --staged` — clean (run after staging)
- `./scripts/secret-scan-branch.sh --strict` — clean

## Verdict

Documentation is in sync with the shipped code. One tech-debt row added for
two test-report gaps; plan checklist and deviation notes brought current.
No behavior, contract, or wording changes were needed in the six doc copies
the plan named.
