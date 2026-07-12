# Test Report — ralph-insights

- Date: 2026-07-13
- Plan: docs/plans/active/2026-07-13-ralph-insights.md
- Branch: feat/ralph-insights
- Runner: ./scripts/run-test.sh (changed-language scope)
- Evidence: docs/evidence/test-2026-07-13-ralph-insights.log

## Verdict: PASS

All test suites passed. No failures.

---

## Suite results

### Canonical gate

| Command | Result |
|---------|--------|
| `./scripts/run-test.sh` | PASS — all verifiers passed |

The canonical gate ran in `test` mode (HARNESS_VERIFY_MODE=test), detected `golang` as the changed language, and ran all shell suites + `go test ./...`.

---

### Feature suites (explicit)

| Suite | Tests | Result |
|-------|-------|--------|
| `tests/test-insights-append.sh` | 39 | PASS 39/39 |
| `tests/test-insights-pipeline-events.sh` | 37 | PASS 37/37 |
| `go test ./internal/insights/...` | 28 | PASS 28/28 |
| `go test ./internal/cli/...` | 79 | PASS 79/79 |

**Feature suite total: 183 tests, 0 failures.**

#### test-insights-append.sh coverage (39 tests)
- Case 1: valid append — all 15 required schema-v1 fields present and correct
- Case 2: second append to same slug → 2 lines, correct phases
- Case 3: missing required flag → exit 1, no file created
- Case 4: invalid enum values (flow/phase/verdict/source) → exit 1, no file
- Case 5: counts land correctly in findings and triage objects (7 fields)
- Case 6: `--events-dir` override — custom directory respected
- Case 7: optional fields absent when not provided (driver, requested_model, cycle=null)

#### test-insights-pipeline-events.sh coverage (37 tests)
- Case 1: DRY_RUN=1 `--skip-pr` — 6 events emitted (all phases except pr), all with correct flow/schema/slug/driver/model/run_id
- Case 2: DRY_RUN=1 default — events file present with ≥ 6 events
- Case 3: codex driver — implement event has effective_model=codex-default, honored=false, driver=codex
- Case 4: semantic unit tests — forced outcome paths: self_review CRITICAL→verdict=fail, verify fail→verdict=fail, cross_review action_required→triage counts, test pass→verdict=pass+critical=0

#### Go insights package (28 tests)
- `TestReadEvents_*`: valid file, corrupt lines (2-of-4 skipped with SkippedLines counter), missing dir (graceful), empty dir, multiple files
- `TestReadReceipts_*`: valid file, missing file (graceful)
- `TestAggregate_*`: basic phase grouping, honored rate, escalation detection, no-escalation on single cycle
- `TestAggregateWithReceipts*`: with receipts, absent receipts
- `TestFindings_Total`: arithmetic
- `TestParseReport_*`: self-review with findings, multiple findings, verify pass/fail, test pass/fail, cross-review zero/action-required, unrecognised type, missing file
- `TestDedupeKey`, `TestRunBackfill_*`, `TestAppendBackfillEvent_JSONStructure`

#### Go cli package (79 tests, net-new for insights: 7)
- `TestInsightsCmd_NoData`: exits 0, prints "No insight data yet" naming expected locations
- `TestInsightsCmd_HumanOutput`: Events/Routing sections, phase present, 100% honored rate
- `TestInsightsCmd_JSONOutput`: valid JSON, per-phase fields
- `TestInsightsCmd_EscalationShown`: escalation data surfaced in output
- `TestBackfillCmd_DryRun`: lists derivable events without writing
- `TestBackfillCmd_Apply`: writes events with source=backfill
- `TestBackfillCmd_Idempotent`: second apply run adds 0 duplicates

---

### Regression suites

| Suite | Tests | Result |
|-------|-------|--------|
| `tests/test-model-routing.sh` | 24 | PASS 24/24 |
| `tests/test-ralph-cli-driver.sh` | 103 | PASS 103/103 |
| `tests/test-ralph-config.sh` | 43 | PASS 43/43 |
| `tests/test-ralph-deprecation-notice.sh` | 4 | PASS 4/4 |
| `tests/test-ralph-dry-run-side-effects.sh` | 5 | PASS 5/5 |
| `tests/test-ralph-orchestrator-branch-names.sh` | 3 | PASS 3/3 |
| `tests/test-ralph-orchestrator-parsers.sh` | 17 | PASS 17/17 |
| `tests/test-ralph-orchestrator-pr-strategy.sh` | 24 | PASS 24/24 |
| `tests/test-ralph-pipeline-functions.sh` | 8 | PASS 8/8 |
| `tests/test-ralph-run-options.sh` | 5 | PASS 5/5 |
| `tests/test-ralph-signals.sh` | 3 | PASS 3/3 |
| `tests/test-ralph-slice-skip-pr.sh` | 4 | PASS 4/4 |
| `tests/test-ralph-status.sh` | 51 | PASS 51/51 |
| `tests/test-ralph-worktree.sh` | 29 | PASS 29/29 |

**Regression total: 323 tests, 0 failures.**

Pipeline hook calls (`insights-append.sh`) introduced in `ralph-pipeline.sh` did not alter phase exit codes or semantics — confirmed by test-ralph-pipeline-functions.sh (8/8), test-model-routing.sh (24/24), test-ralph-dry-run-side-effects.sh (5/5), and test-ralph-cli-driver.sh (103/103) all passing unchanged.

---

## Edge case spot-checks (plan Test plan)

| Edge case | Coverage status |
|-----------|----------------|
| Empty events dir | COVERED — `TestReadEvents_EmptyDir` (Go) |
| Receipts without events | COVERED — `TestInsightsCmd_HumanOutput` uses events only, `TestAggregateWithReceipts_Absent` (Go) |
| Corrupt JSONL line skipping | COVERED — `TestReadEvents_CorruptLines`: 2-of-4 lines skipped, SkippedLines=2; testdata/corrupt_lines.jsonl has 2 valid + `this is not json at all` + `{broken json` |
| Slug with unusual characters | PARTIAL GAP — test-insights-append.sh Case 1 uses slug="my-task" (hyphen only); no test for slugs with spaces, slashes, or percent-encoded chars. The appender writes a filename `<date>-<slug>.jsonl` directly, so unusual chars could produce invalid paths. Documented as gap below. |
| Missing schema field tolerance | COVERED — `TestReadEvents_CorruptLines` skips lines that fail json.Unmarshal; valid events with all fields are accepted. However, there is no explicit test for a valid-JSON line that is missing one optional schema field (e.g. no `cycle` key). The reader uses `omitempty`-compatible structs so unknown/absent fields degrade to zero values — behavior is correct but not directly asserted. Documented as gap below. |

---

## Coverage gaps

**G1 — Slug with unusual characters (LOW)**
No test exercises `insights-append.sh --slug 'task name with spaces'` or `--slug 'feat/my-task'` (contains slash). The appender constructs the output path as `<events-dir>/<UTC-date>-<slug>.jsonl` without sanitization. A slug with `/` would create unexpected subdirectories. Risk is low because slugs are generated from plan filenames (already sanitized) but the contract is not enforced.

**G2 — Schema-field-absent tolerance (LOW)**
No test verifies that a valid-JSON event line missing an optional field (e.g. no `cycle` key, no `driver` key) is read gracefully with zero-valued defaults. The Go struct uses `json:",omitempty"` so behavior is correct, but there is no regression guard. Adding one fixture line with missing fields to `testdata/corrupt_lines.jsonl` (or a new `testdata/sparse_fields.jsonl`) would close this gap.

**G3 — `ralph insights backfill` against real fixture reports (LOW)**
The backfill parser Go tests (`TestRunBackfill_*`) use inline string fixtures. There are no tests that run `ralph insights backfill` against the actual `docs/reports/` directory in the worktree. This means the regex patterns are exercised but the CLI end-to-end path (`ralph insights backfill --apply`) against real reports is only tested by `TestBackfillCmd_Apply` using synthetic fixtures. If future report formats diverge, the parser would silently emit zero events rather than erroring.

None of the gaps above block the PR. All are non-critical coverage improvements.

---

## AC compliance

| AC | Status | Evidence |
|----|--------|---------|
| AC1: insights-append.sh schema-v1 + exit ≠ 0 on missing args | PASS | test-insights-append.sh 39/39 |
| AC2: DRY_RUN=1 emits one event per executed phase; forced-outcome tests verify verdict/findings/triage | PASS | test-insights-pipeline-events.sh Cases 1–4 |
| AC3: `ralph insights` table + `--json` valid; Go unit tests with fixtures | PASS | TestInsightsCmd_HumanOutput, TestInsightsCmd_JSONOutput |
| AC4: zero data sources → exit 0 with explicit note naming locations | PASS | TestInsightsCmd_NoData |
| AC5: backfill dry-run lists events; `--apply` writes; second run adds 0 duplicates; multi-cycle yields distinct events | PASS | TestBackfillCmd_DryRun, TestBackfillCmd_Apply, TestBackfillCmd_Idempotent, TestRunBackfill_MultiCycleDistinctEvents |
| AC6: skill bodies instruct event append; mirror regenerated; check-skill-sync.sh + check-sync.sh + check-template.sh pass | PASS (by /verify) | verify-2026-07-13-ralph-insights.md |
| AC7: docs/insights/README.md schema + AGENTS.md/model-routing.md pointers | PASS (by /verify) | verify-2026-07-13-ralph-insights.md |
| AC8: `go test ./... -count=1` passes | PASS | run-test.sh canonical gate |

---

## Known gaps carried forward

- G1: slug unusual character path sanitization — LOW, no PR block
- G2: sparse-field event tolerance fixture — LOW, no PR block
- G3: backfill CLI against real report directory — LOW, no PR block
