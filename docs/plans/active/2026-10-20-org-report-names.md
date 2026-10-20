# org-report-names

- Status: Approved
- Approved: 2026-10-20 sha256:b6c0dd650dcc
- Owner: Claude Code
- Date: 2026-10-20
- Related request: the F8 finding of the stage 9 trial (`docs/tech-debt/README.md`, the `scripts/check-sync.sh` row; `docs/evidence/org-trial-2026-10-20.md`, Findings, F8)
- Related issue: N/A
- Type: fix
- Branch: fix/org-report-names

## Objective

Let a feature org in this meta-repo pass its gate with the `/org` skill's
closing steps as written. Today `scripts/check-sync.sh` reports the
reviewer seat's report and the org report (`ralph org report`) as
`ROOT_ONLY`, so the gate's static step fails (stage 9 trial, F8:
doctor-escape's first gate ended `static_failed`).

## Scope

- `scripts/check-sync.sh`: add `docs/reports/review-` and
  `docs/reports/org-manifest-` to `ROOT_ONLY_EXCLUSIONS`, next to the
  other runtime report prefixes
- `internal/org/prompts/reviewer.md`: name the reviewer seat's report
  `docs/reports/review-<YYYY-MM-DD>-{{ORG_ID}}-{{SEAT_ID}}.md` in the first item of
  `## レポート契約`. The second item (`leader へは RESULT ...`) stays as it
  is, because `TestRenderRolePrompt_Reviewer_ResultCarriesHeadAndBase`
  reads it
- `internal/org/report.go`: move the org report's file name prefix
  `org-manifest-` into a named constant used by the `Sprintf` that builds
  the path; no change to the name it writes
- A sync test in `internal/org` that ties the two names to every place
  that spells them by hand: the reviewer template, the two exclusions in
  `scripts/check-sync.sh`, the naming list in `docs/reports/README.md`,
  and the `report` row of the `/org` skill. It finds the repo root with
  `runtime.Caller`, as `internal/org/send_defaults_sync_test.go` does, and
  skips when a file is absent
- `internal/org/prompts_test.go`: one more check in the reviewer test, for
  the report name in the first item of `## レポート契約`
- `docs/reports/README.md` and `templates/base/docs/reports/README.md`
  (kept identical by `scripts/check-sync.sh`): add the reviewer seat's
  report and the org report to the lists
- Records: close the F8 row in `docs/tech-debt/README.md`; reword the
  Trigger of the session-end WIP hook row (F11), which names "the separate
  PR of the `check-sync.sh` row above", and keep that row open; name the
  fix branch in the evidence's F8

## Non-goals

- F11 (the session-end WIP hook commits in a seat's worktree). Its row
  stays open; this PR only rewords its Trigger
- Moving `ralph org report`'s output outside the worktree (option C of
  the consult). It changes Go behavior and the closing steps of the
  `/org` skill and the leader template
- Excluding every top-level file in `docs/reports/` (option D). It would
  hide a real `ROOT_ONLY` for a new top-level file such as a README
- Renaming reports that already exist, or the `self-review-` prefix:
  `internal/insights/backfill.go` reads `self-review-` files as
  `/self-review` reports, so the reviewer seat keeps a prefix of its own
- Changing the leader template or the `/org` skill's closing steps. They
  already commit "the reviewer's report under `docs/reports/`" and the org
  report

## Assumptions

- `scripts/check-sync.sh` runs only in this meta-repo (it is in its own
  exclusion list and is not shipped), so the new exclusions change nothing
  downstream
- The reviewer template is embedded in the `ralph` binary and reaches
  downstream projects; there it only fixes a file name, which nothing
  downstream checks
- The prefix match `docs/reports/review-` does not cover
  `docs/reports/templates/review-report.md`, because `templates/` sits
  between them
- Section 3 of `check-sync.sh` lists files with `find`, so an untracked
  report counts the same as a committed one

## Affected areas

- `scripts/check-sync.sh`
- `internal/org/prompts/reviewer.md`, `internal/org/prompts_test.go`
- `internal/org/report.go`, a new `internal/org/report_names_sync_test.go`
- `docs/reports/README.md`, `templates/base/docs/reports/README.md`
- `docs/tech-debt/README.md`, `docs/evidence/org-trial-2026-10-20.md`

## Visual review

- Page: `.harness/state/plan-visual/org-report-names.html` (one easy
  figure `overview`; in the fold `detail-overview`, the file map, and
  `detail-names`, which names pass and which still fail)
- Lint: `./scripts/plan-visual.sh lint` exit 0 (again after the Codex
  advisory's seat-id change)
- Self-check: shot the full page, `overview`, `detail-overview`, and
  `detail-names`, and read each PNG; no overlap, clipping, or line through
  a label. `detail-names` was shot and read again after the seat-id change

## Design decisions

- The fix (consult-plan-f8, 2026-10-20): option B, the two exclusions plus
  a fixed name for the reviewer seat's report, tied together by a test.
  Option A alone leaves the reviewer's name free, so a reviewer that picks
  another name fails the gate again. Option C changes Go and the closing
  steps. Option D hides real `ROOT_ONLY` files
- The name is `review-<YYYY-MM-DD>-<org_id>-<seat_id>.md`: the shape
  doctor-escape's reviewer chose in the trial (`review-<date>-<org_id>`)
  plus the seat id, because the leader template allows more than one
  reviewer seat (`internal/org/prompts/leader.md`, item 2 of the mission)
  and two seats of one org would otherwise write the same file (Codex plan
  advisory, MEDIUM). `self-review-` would be read by the insights backfill
  as a `/self-review` report
- Critical forks: None

## Acceptance criteria

- [x] AC1: with untracked `docs/reports/review-2026-10-20-x-reviewer.md` and
  `docs/reports/org-manifest-x-2026-10-20.md` in the worktree,
  `./scripts/check-sync.sh` exits 0 and prints neither as `ROOT_ONLY`;
  with an untracked `docs/reports/other-x.md` it still exits 1 and names
  that file (the exclusion stays narrow)
- [x] AC2: the rendered reviewer prompt names its report
  `docs/reports/review-<YYYY-MM-DD>-<org_id>-<seat_id>.md`, with the org
  id and the seat id filled in, in the first item of `## レポート契約`, and
  `TestRenderRolePrompt_Reviewer_ResultCarriesHeadAndBase` checks it; two
  prompts rendered for one org with different seat ids name different
  files, and both names start with the excluded `docs/reports/review-`
- [x] AC3: `report.go` builds the org report's path from a named
  constant, and the file it writes is still
  `org-manifest-<org_id>-<date>.md` (the existing report tests pass)
- [x] AC4: a test in `internal/org` fails if the reviewer template, the
  `check-sync.sh` exclusions, `docs/reports/README.md`, or the `/org`
  skill's `report` row stops spelling the two names; it skips a file
  that is absent
- [x] AC5: `docs/reports/README.md` and its template copy list both
  reports and stay identical (`./scripts/check-sync.sh` exits 0)
- [x] AC6: the F8 tech-debt row is resolved (`RESOLVED 2026-10-20 in
  fix/org-report-names`), the F11 row's Trigger no longer names "the
  separate PR of the `check-sync.sh` row above" and the row stays open,
  and the evidence's F8 names the fix branch
- [x] AC7: `./scripts/run-verify.sh` exits 0, and the `internal/org` tests
  pass with `GOTOOLCHAIN=go1.25.8`

## Implementation outline

1. Slice 1 (check-sync and the docs README): add the two exclusions with
   a comment; check AC1 with untracked files and remove them; update both
   README copies
2. Slice 2 (names in Go and the template): the constant in `report.go`,
   the name in `reviewer.md`, the new check in `prompts_test.go`, the sync
   test
3. Slice 3 (records): the tech-debt rows and the evidence's F8. They name
   the branch `fix/org-report-names`, as the register's
   `<!-- RESOLVED <date> in <branch>: ... -->` comments do, so no PR number
   is needed before `/pr`

## Verify plan

- Static analysis checks: `./scripts/run-static-verify.sh` (gofmt,
  go vet, golangci-lint for the Go change; check-sync, skill sync,
  template purity)
- Spec compliance criteria to confirm: AC1 to AC7; the reviewer template
  change keeps the `## レポート契約` second item byte-for-byte
- Documentation drift to check: the `/org` skill's `report` row and the
  leader template still describe the same closing steps; `docs/reports/README.md`
  copies identical; no other doc lists report prefixes without the new two
- Evidence to capture: the AC1 runs of `check-sync.sh` (rc and the
  `ROOT_ONLY` lines), the test names that cover AC2 to AC4

## Test plan

- Unit tests: the new check in
  `TestRenderRolePrompt_Reviewer_ResultCarriesHeadAndBase`; the new sync
  test; the existing `report.go` tests for the file name
- Integration tests: AC1's `check-sync.sh` runs with untracked files
- Regression tests: `go test ./internal/org/...` with the default
  toolchain and with `GOTOOLCHAIN=go1.25.8`, `-timeout 30m`;
  `./scripts/run-test.sh`
- Edge cases: an unrelated top-level report name still fails
  `check-sync.sh`; `docs/reports/templates/review-report.md` is still
  compared; two reviewer seats of one org (seat ids `reviewer` and
  `reviewer-2`) get different report names; the sync test skips cleanly
  when a doc file is missing (as in a scaffolded project)
- Evidence to capture: test output and `check-sync.sh` output in the
  test report

## Risks and mitigations

- A seat already running with the old template names its report freely.
  Mitigation: the exclusion of `review-` covers the name the trial's
  reviewer chose; other names still fail loudly in the gate
- The sync test reads markdown by phrase and could pass on an unrelated
  mention. Mitigation: it checks the reviewer name inside the
  `## レポート契約` section and the org report name on the `/org` skill's
  `report` row, not anywhere in the file
- The exclusion could hide a real root-only file whose name starts with
  `review-` or `org-manifest-`. Mitigation: only reports use those
  prefixes, and `docs/reports/templates/` is not matched

## Rollout or rollback notes

- No migration. Rollback is a revert of the PR; the gate then fails again
  as in the trial

## Open questions

- None

## Progress checklist

- [x] Plan reviewed (consult-plan-f8: option B; Codex plan advisory: one
      MEDIUM, the seat id in the report name, applied)
- [x] Plan approved (the user's standing instruction approves plans
      automatically)
- [x] Branch created
- [x] Implementation started
  - S1 9349914b: the two exclusions in `scripts/check-sync.sh`, both
    `docs/reports/README.md` copies. AC1: rc=0 with the two untracked
    reports, rc=1 naming `docs/reports/other-x.md`
  - S2 7a45cb14: `orgReportFilePrefix`, the reviewer template's name,
    the new check in `prompts_test.go`, `report_names_sync_test.go`
    (`TestReportNames_DocsMatchConstants`, `TestReadRepoFile_MissingFileIsNotAnError`);
    the drift checks failed as expected on the README and on check-sync
  - S3 5b7fe3c0: the F8 row resolved, the F11 Trigger reworded, the
    evidence's F8 names the branch
  - Out of scope, left as they are: the `ralph org report` help text
    (`internal/cli/org.go`), the prefix check in `internal/cli/org_test.go`,
    and two other mentions of the org report in the `/org` skill (the
    `inbox resolve` example, the closing checklist); none of them is a
    report name a gate reads
- [ ] Review artifact created
- [ ] Verification artifact created
- [ ] Test artifact created
- [ ] PR created
