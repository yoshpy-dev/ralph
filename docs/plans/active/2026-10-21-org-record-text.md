# org-record-text

- Status: Approved
- Approved: 2026-10-21 sha256:200072f06b32
- Owner: Claude Code
- Date: 2026-10-21
- Related request: the F12 and F14 findings of the stage 9 trial (`docs/evidence/org-trial-2026-10-20.md`, Findings; `docs/tech-debt/README.md`, the `truncateForDetails` row and the `ralph org report`'s event timeline row), and the LOW findings L4 (its rest) and L5 that PR #244's self-review left open (`docs/reports/self-review-2026-10-20-org-report-names.md`); the user's request of 2026-10-21 to fix every trial finding that can be fixed. Second PR of the trial-fix series (`docs/evidence/org-spec-evidence-2026-10-21.md`, 試行の所見)
- Related issue: N/A
- Type: fix
- Branch: fix/org-record-text

## Objective

Keep the org records readable when a message body is Japanese or has
several lines:

- F12: `truncateForDetails` (`internal/org/verbs.go`) cuts a `sent`
  event's body at 200 bytes, which can split a multi-byte character. JSON
  encoding then writes one U+FFFD per byte of the split character into the
  manifest, and `ralph org report` copies it into the org report. In the
  trial four `sent` lines ended that way.
- F14: `BuildOrgReport` (`internal/org/report.go`) writes each event's
  Details into one Markdown table row as it is. A `sent` event's Details
  hold the message body, whose header lines and blank line split the row,
  so every `sent` row of the trial's two org reports spans several lines
  and a blank line ends the table in a GFM renderer.

Also close the two test-only LOW findings PR #244 left in `internal/org`
(L4's rest, L5), since this PR touches the same package's tests.

## Scope

- S1 (F12) `internal/org/verbs.go`: `truncateForDetails` keeps the cut at
  or below `sendDetailsMaxLen` bytes and moves it back to the start of the
  character it would split, so the kept prefix is valid UTF-8 whenever the
  input is. The `...(truncated)` suffix and the 200-byte limit stay. A text
  of at most 200 bytes stays as it is. The step back is bounded
  (`utf8.UTFMax - 1` bytes), so an input that is not valid UTF-8 is still
  cut near 200 bytes
- S1 tests in `internal/org/verbs_test.go`: a unit test of
  `truncateForDetails` (table of cases below) and one test through `Send`
  that a Japanese body over 200 bytes leaves a `sent` event whose Details
  are valid UTF-8 with no U+FFFD
- S2 (F14) `internal/org/report.go`: write the event timeline's details
  cell so a row stays on one line and a GFM renderer shows the text as
  written. In one pass (one `strings.NewReplacer`, so nothing is escaped
  twice): put a backslash before `\`, `|` (the existing escape), `` ` ``,
  and `<`, and write each line break as `<br>`, `\r\n` before a lone `\r`
  and `\n`. The backslash before `\` keeps a shell continuation (`\` at a
  line end) from escaping the `<br>` after it; the one before `` ` ``
  keeps a fenced snippet, once on one line, from becoming a code span that
  shows `<br>` as text; the one before `<` keeps a placeholder such as
  `<path>` from being read as an HTML tag. Other cells and other tables do
  not change (their values are ids, names, and timestamps)
- S2 tests in `internal/org/report_test.go`: Details with header lines and
  a blank line, with `\r\n`, with a lone `\r`, with a `|` and a line break
  together, with a `\` at a line end, with a fenced snippet, and with a
  `<path>` placeholder; each event stays one row of four cells and the
  timeline has one line per event
- S3 (L4's rest, L5) tests only:
  - Move `reviewerReportPrefix` from `internal/org/report_names_sync_test.go`
    into `internal/org/prompts_test.go`, built from a new
    `reviewerReportFilePrefix = "review-"` as
    `defaultReportDir + "/" + reviewerReportFilePrefix`, so the reviewer
    template's test no longer needs the sync test's file. The sync test's
    README check uses `reviewerReportFilePrefix` instead of trimming the
    directory off `reviewerReportPrefix`
  - Move the report-name check out of
    `TestRenderRolePrompt_Reviewer_ResultCarriesHeadAndBase` into a new
    `TestRenderRolePrompt_Reviewer_ReportContractNamesTheReport`, with the
    same checks. Both doc comments name plans by slug, not by a
    `docs/plans/active/` path
- S4 records:
  - `docs/tech-debt/README.md`: resolve the `truncateForDetails` row and
    the `ralph org report`'s event timeline row with the RESOLVED
    convention of the rows around them (a `<!-- RESOLVED ... -->` line
    above the row and the row struck through with a `(RESOLVED ...)` tail)
  - `docs/evidence/org-spec-evidence-2026-10-21.md`: in the 試行の所見
    table, the F12 and F14 rows name this branch as the fix
  - `docs/specs/2026-10-07-org-multi-org-director.md`: the 印(2026-10-21)
    note of NFR 記録の置き場所, which says F12 and F14 do not contradict
    the item, adds that this branch fixed them. No mark changes: neither
    finding contradicts an item, and the Status section lists only the
    findings that do

## Non-goals

- Escaping the other characters GFM reads inline (`*`, `_`, `[`, `]`,
  `~`, `&`, `>`). They change how a span is styled or linked, not where a
  row or a line ends, and escaping every one would fill the raw report
  with backslashes. Of the trial's 77 events with Details, 71 hold `_`,
  all inside words such as `TASK_ID` (GFM does not emphasize there), and
  one holds `*` (counted in the trial manifest). Recorded under Open
  questions
- Changing the 200-byte limit, the `...(truncated)` suffix, what a `sent`
  event's Details hold, or the manifest format
- Changing the report's other tables, or how `ralph org report` names or
  places its file
- Other trial findings (F3 to F11, F13, D-F3, folder trust, G1, the leader
  ALERT). They are later PRs of the series
- Rewriting the existing org reports under `docs/evidence/org-trial-2026-10-20/reports/`:
  they are the trial's snapshot

## Assumptions

- `truncateForDetails` has two callers (`Send`'s dry-run record and its
  `sent` record, `internal/org/verbs.go`), and both want the same cut
- `BuildOrgReport` is the only Markdown table writer in the Go code
  (`git grep -n '"| %s'` finds only `internal/org/report.go`)
- GFM renders `<br>` inside a table cell as a line break, so a rendered
  report still shows the body's lines; the raw file shows `<br>`
- `permissionModeFromDetails` reads the raw Details for the roster, not the
  escaped cell, so S2 does not change the roster
- The test files of `internal/org` compile as one package, so moving a
  test constant between them changes nothing outside the tests

## Affected areas

- `internal/org/verbs.go` (S1)
- `internal/org/verbs_test.go` (S1)
- `internal/org/report.go` (S2)
- `internal/org/report_test.go` (S2)
- `internal/org/prompts_test.go`, `internal/org/report_names_sync_test.go` (S3)
- `docs/tech-debt/README.md`, `docs/evidence/org-spec-evidence-2026-10-21.md`,
  `docs/specs/2026-10-07-org-multi-org-director.md` (S4)

## Visual review

- Page: `.harness/state/plan-visual/org-record-text.html` (one easy figure
  `overview`; in the fold `detail-overview` and `detail-cell`)
- Lint: `./scripts/plan-visual.sh lint` exit 0
- Self-check: shot the full page, `overview`, `detail-overview`, and
  `detail-cell` and read each PNG; fixed one label (the example of a split
  character showed the character itself instead of its JSON escape)

## Design decisions

- Line breaks in a report cell become `<br>` (S2). Alternatives: a literal
  `\n` text, which keeps the raw file and the rendered view on one line
  but shows the body as one run of text and cannot be told from a body
  that holds a backslash and an `n`; or a code span, which a table cell
  cannot hold across lines either. `<br>` is what the tech-debt row
  suggested and keeps the body's lines visible when rendered. The choice
  is one `strings.NewReplacer` and its tests, so it is cheap to change
- The cell escapes `\`, `|`, `` ` ``, and `<` with a backslash, not with
  HTML entities (Codex plan advisory, MEDIUM: without the first and the
  third, a shell continuation and a fenced snippet lose text or show
  `<br>` literally once rendered). A backslash escape is the GFM way for
  any ASCII punctuation and reads better in the raw file than `&#92;` or
  `&#96;`. The table parser takes `\|` as a pipe even inside a code span,
  and with every backtick escaped no code span forms
- The cut in S1 moves back, not forward, so the kept prefix never grows
  past 200 bytes and the manifest line never grows
- Critical forks: None. Each choice above is local to one function and its
  tests and is cheap to reverse

## Acceptance criteria

- [ ] AC1: `truncateForDetails` returns a valid UTF-8 string for every
  valid UTF-8 input, keeps at most `sendDetailsMaxLen` bytes before the
  suffix, and returns an input of at most 200 bytes unchanged. Cases: 199
  ASCII bytes then a 3-byte character (kept prefix is the 199 bytes); a
  3-byte character ending exactly at byte 200 (kept); a 4-byte character
  starting at bytes 197, 198, 199, and 200; all-ASCII over 200 bytes
  (cut at 200, as today); exactly 200 bytes (unchanged); an input that is
  not valid UTF-8 (the cut stays within `utf8.UTFMax - 1` bytes of 200)
- [ ] AC2: a `Send` with a Japanese body longer than 200 bytes records a
  `sent` event whose Details are valid UTF-8, hold no U+FFFD, and end in
  `...(truncated)`
- [ ] AC3: `BuildOrgReport` writes an event whose Details hold header
  lines and a blank line as one row of four cells, with `<br>` for each
  line break; `\r\n` and a lone `\r` give one `<br>` each; a `|` next to a
  line break is still escaped as `\|`; a `\` at a line end is written
  `\\<br>`; each backtick of a fenced snippet is written ``\` ``; `<path>`
  is written `\<path>`; a `\|` already in the text is written `\\\|`. The
  event timeline has exactly one line per event
  plus its two header lines. The cell of exact expected strings is
  compared whole, not by `strings.Contains` on a part
- [ ] AC3b: a report with the AC3 cases, rendered once with GitHub's GFM
  renderer (`gh api markdown -f mode=gfm -F text=@<file>`, read-only, by
  the verifier as evidence, not in a test), has one `<tr>` per event, a
  `<br>` element for each line break, no `<code>` element, four cells in
  each row, and shows the `\`, the backticks, `<path>`, and `\|` as
  written
- [ ] AC4: the existing report and send tests pass unchanged, including
  `TestBuildOrgReport_EscapesPipesInEventDetails` and the Details prefix
  tests of `Send`
- [ ] AC5: `reviewerReportPrefix` and `reviewerReportFilePrefix` are
  defined in `internal/org/prompts_test.go`; `report_names_sync_test.go`
  no longer defines `reviewerReportPrefix` and its README check uses
  `reviewerReportFilePrefix`; `TestRenderRolePrompt_Reviewer_ReportContractNamesTheReport`
  holds the report-name checks and `TestRenderRolePrompt_Reviewer_ResultCarriesHeadAndBase`
  no longer does. `go test ./internal/org -run 'TestRenderRolePrompt_Reviewer|TestReportNames'`
  passes, and the set of checks is the same as before (each moved
  `t.Errorf`/`t.Fatalf` is in the new test)
- [ ] AC6: the two tech-debt rows are resolved with the convention of the
  rows around them, each naming the tests that close it; the evidence
  table's F12 and F14 rows name this branch; the spec's NFR 記録の置き場所
  note says this branch fixed F12 and F14, and no mark or Status count
  changes
- [ ] AC7: `./scripts/run-verify.sh` exits 0 (with `GOFLAGS=-timeout=30m`),
  and `GOTOOLCHAIN=go1.25.8 go test -timeout 30m ./internal/org/` passes

## Implementation outline

1. S1: `truncateForDetails` and its tests (one commit)
2. S2: the details cell of `BuildOrgReport` and its tests (one commit)
3. S3: the test constant move and the test split (one commit)
4. S4: the records (one commit)

## Verify plan

- Static analysis checks: `./scripts/run-static-verify.sh` (gofmt, go vet,
  check-sync, check-skill-sync); `git diff` of S3 shows only moved test
  code and doc comments
- Spec compliance criteria to confirm: AC1 to AC7; the spec's marks and
  Status counts are the same as on main (20 `[x]`, 12 `[ ]`)
- Documentation drift to check: no doc describes the report's details cell
  or the cut of a `sent` body other than the two tech-debt rows and the
  trial evidence (`git grep -n 'truncateForDetails\|<br>' -- docs .claude
  internal/org/prompts`); the two `docs/reports/README.md` copies stay
  identical
- Evidence to capture: run-static-verify output, the grep results

## Test plan

- Unit tests: the `truncateForDetails` table (AC1); the report cases (AC3)
- Integration tests: the `Send` test through the stub driver (AC2); a
  report built from events read back from a manifest is not needed, since
  `BuildOrgReport` takes the events as they are read
- Regression tests: the whole `internal/org` package with go1.26 and
  go1.25.8 (AC4, AC7); `./scripts/run-test.sh`
- Edge cases: a 4-byte character at each offset across byte 200; a
  character ending exactly at 200; exactly 200 bytes; invalid UTF-8; a
  blank line; `\r\n` and a lone `\r`; `|` next to a line break; an event
  with empty Details still prints `-`
- Evidence to capture: `go test -v` output of the new tests, the package
  run on both toolchains

## Risks and mitigations

- A reader or tool that parses the org report's raw text and expects real
  newlines in the details cell. Mitigation: none known in the repo (the
  `/org` skill and leader template only commit the report); the report is
  for humans
- The S3 split loses a check. Mitigation: AC5 compares the moved
  `t.Errorf`/`t.Fatalf` lines; the self-review reads the diff of the two
  tests side by side
- Invalid UTF-8 already in a body. Mitigation: the bounded step back keeps
  the cut near 200 bytes, and JSON encoding still replaces bad bytes as
  today

## Rollout or rollback notes

- Ships in the next `ralph` release with no migration. Existing manifests
  and reports are not rewritten. Revert the PR to roll back

## Open questions

- Should the report also escape `*`, `_`, `[`, `]`, `~`, `&`, and `>` so a
  body renders with no styling or links? Not needed for F14; left until a
  report shows the problem

## Progress checklist

- [x] Plan reviewed (Codex plan advisory: one MEDIUM, applied: the details cell also escapes `\`, `` ` ``, and `<`, with AC3 cases and AC3b; consult-plan-recordtext: proceed)
- [x] Plan approved (auto-approved per the user's standing instruction of 2026-10-11)
- [x] Branch created
- [x] Implementation started
  - S1 (F12) 19fad46d: `truncateForDetails` cuts at a character boundary, with its tests
  - S2 (F14) a88e7382: `escapeReportCell` in `BuildOrgReport`, with its tests
  - S3 3c688ff3: `reviewerReportFilePrefix` and `reviewerReportPrefix` move to `prompts_test.go`, and `TestRenderRolePrompt_Reviewer_ReportContractNamesTheReport` takes the 6 report-name checks (tests only)
  - S4 f7f2644b: records. Deviations: the F12 and F14 tech-debt rows strike the trigger cell too, as the F8 row does; their evidence cells add this plan's active path, because the plan-reference check fails on the archive path while the plan is in flight (`/pr`'s `archive-plan.sh` rewrites it); the new Go comments write "pipe" as a word; spec line 141 gets one sentence after its 。
  - Out of scope, left as is: the doc comment of `TestReportNames_DocsMatchConstants` (`internal/org/report_names_sync_test.go`) still names a plan by its date-slug form
- [ ] Review artifact created
- [ ] Verification artifact created
- [ ] Test artifact created
- [ ] PR created
