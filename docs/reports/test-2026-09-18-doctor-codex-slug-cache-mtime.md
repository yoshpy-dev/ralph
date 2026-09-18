# Test report: doctor-codex-slug-cache-mtime

- Date: 2026-09-18
- Plan: `docs/plans/active/2026-09-18-doctor-codex-slug-cache-mtime.md` (issue #159)
- Tester: `tester` subagent (Claude Code, standard flow)
- Branch: `feat/doctor-codex-slug-cache-mtime`, HEAD `5589d5f` (base `main` @ `26ca767`)
- Scope: changed-language scope (golang) via `./scripts/run-test.sh`, plus fresh
  `go test ./internal/... -count=1`, targeted `-run`/`-v` invocations, coverage,
  and a `-count=3` repeat of the new tests. No static analysis run here (that
  is `/verify`'s job; see `docs/reports/verify-2026-09-18-doctor-codex-slug-cache-mtime.md`,
  PASS, which explicitly deferred `go test` / `./scripts/run-test.sh` to this
  report).
- Evidence: `docs/evidence/test-2026-09-18-doctor-codex-slug-cache-mtime.log`
  (gitignored, full `./scripts/run-test.sh` output)

## Overall verdict: Pass

All behavioral tests pass. The 6 new test functions (13 assertions/subtests:
3 single-case tests, 1 five-row table, 1 five-case table, 1 seam-driven test)
pass individually with verbose output, and alongside the 8 pre-existing
`TestCheckCodexModelSlugs_*` tests as a full regression set. Full internal
package regression (`go test ./internal/... -count=1`, fresh, no cache) is
8/8 packages green. `./scripts/run-test.sh` exits 0 across 28 shell test
files plus the same 8 Go packages. A `-count=3` repeat of the 6 new tests
shows no flakiness.

## Test execution

### `./scripts/run-test.sh` (changed-language scope = golang; shell suite runs unscoped as a regression net)

| Suite | Result |
| --- | --- |
| 28 shell test files under `tests/*.sh` (`test-agent-phase-boundaries.sh`, `test-branch-name.sh`, `test-check-mojibake.sh`, `test-check-skill-sync.sh`, `test-detect-changed-languages.sh`, `test-detect-languages-terraform.sh`, `test-ensure-pr-ready.sh`, `test-ensure-pr-title-prefix.sh`, `test-gc-artifacts.sh`, `test-hook-wiring.sh`, `test-insights-append.sh`, `test-language-pack-monorepo-roots.sh`, `test-no-loop-references.sh`, `test-post-edit-verify.sh`, `test-pre-bash-guard.sh`, `test-ralph-config.sh`, `test-ralph-dispatch.sh`, `test-ralph-worktree.sh`, `test-run-verify-scope.sh`, `test-secret-scan.sh`, `test-self-review-scope.sh`, `test-sync-skills.sh`, `test-template-purity.sh`, `test-terraform-gitignore.sh`, `test-terraform-pack-verify.sh`, `test-terraform-rule-frontmatter.sh`, `test-verify-mode-split.sh`, `test-xreview-helpers.sh`) | All PASS. Zero `FAIL: <n>` (n≥1) lines anywhere in the log; every suite that prints an explicit summary reports `FAIL: 0`. None of these suites touch `internal/cli`'s doctor check — unrelated regression net, unaffected by this PR's Go-only + docs diff. |
| `golang` language verifier (`go test ./...` for the 8 test-bearing packages, cached at this point in the wrapper) | `ok` for `internal/cli`, `internal/config`, `internal/insights`, `internal/org`, `internal/org/driver`, `internal/org/protocol`, `internal/scaffold`, `internal/upgrade`; `[no test files]` for the two `main`-package roots (`github.com/yoshpy-dev/ralph`, `cmd/ralph`) |
| Wrapper exit code | 0 (`==> All verifiers passed.`) |

### Fresh `go test ./internal/... -count=1` (no cache)

```
ok  	github.com/yoshpy-dev/ralph/internal/cli	35.228s
ok  	github.com/yoshpy-dev/ralph/internal/config	1.121s
ok  	github.com/yoshpy-dev/ralph/internal/insights	0.765s
ok  	github.com/yoshpy-dev/ralph/internal/org	9.601s
ok  	github.com/yoshpy-dev/ralph/internal/org/driver	1.973s
ok  	github.com/yoshpy-dev/ralph/internal/org/protocol	1.178s
ok  	github.com/yoshpy-dev/ralph/internal/scaffold	2.229s
ok  	github.com/yoshpy-dev/ralph/internal/upgrade	2.981s
```

8/8 packages ok, 0 failures. Re-run separately without `tee` to confirm the
exit code directly (per this repo's tee-masks-exit convention): `EXIT=0`.

### Targeted tests: all 14 `TestCheckCodexModelSlugs_*` / `TestFormatCacheAge` / `TestCodexCacheFreshnessClause` functions, `-run ... -count=1 -v`

| Test | Result |
| --- | --- |
| `TestCheckCodexModelSlugs_AllPresent_Pass` (pre-existing) | PASS |
| `TestCheckCodexModelSlugs_SomeMissing_WarnNamesEach` (pre-existing) | PASS |
| `TestCheckCodexModelSlugs_CacheMissing_InfoNamesPath` (pre-existing) | PASS |
| `TestCheckCodexModelSlugs_NoCodexEntries_Info` (pre-existing) | PASS |
| `TestCheckCodexModelSlugs_UnparsableCache_InfoNotWarn` (pre-existing) | PASS |
| `TestCheckCodexModelSlugs_CodexHomeUsedLiterally_TrailingSpace` (pre-existing) | PASS |
| `TestCheckCodexModelSlugs_WhitespaceOnlyCodexHome_UsedLiterally` (pre-existing) | PASS |
| `TestCheckCodexModelSlugs_HomeUnresolvable_Info` (pre-existing) | PASS |
| `TestCheckCodexModelSlugs_DetailCarriesExactCacheMtime` (new, AC-1/AC-2) | PASS |
| `TestCheckCodexModelSlugs_StaleCache_WarnCarriesExactMtimeAndStaleNote` (new, AC-1/AC-2) | PASS |
| `TestCheckCodexModelSlugs_StaleCache_PassAlsoCarriesStaleNote` (new, AC-2) | PASS |
| `TestCodexCacheFreshnessClause` (new, AC-1/AC-2/AC-4; 5 subtests: fresh/exactly-24h/24h+1s/changed-wins/zero-mtime) | PASS (parent + all 5 subtests) |
| `TestFormatCacheAge` (new, AC-3; 5 subtests: negative/sub-minute/90m/47h/49h) | PASS (parent + all 5 subtests) |
| `TestCheckCodexModelSlugs_CacheChangedWhileReading_FreshnessUnknown` (new, AC-4) | PASS |

Raw run (`go test ./internal/cli/ -run '<all 14 names, alternated>' -count=1 -v`):

```
=== RUN   TestCheckCodexModelSlugs_AllPresent_Pass
--- PASS: TestCheckCodexModelSlugs_AllPresent_Pass (0.00s)
=== RUN   TestCheckCodexModelSlugs_SomeMissing_WarnNamesEach
--- PASS: TestCheckCodexModelSlugs_SomeMissing_WarnNamesEach (0.00s)
=== RUN   TestCheckCodexModelSlugs_CacheMissing_InfoNamesPath
--- PASS: TestCheckCodexModelSlugs_CacheMissing_InfoNamesPath (0.00s)
=== RUN   TestCheckCodexModelSlugs_NoCodexEntries_Info
--- PASS: TestCheckCodexModelSlugs_NoCodexEntries_Info (0.00s)
=== RUN   TestCheckCodexModelSlugs_UnparsableCache_InfoNotWarn
--- PASS: TestCheckCodexModelSlugs_UnparsableCache_InfoNotWarn (0.00s)
=== RUN   TestCheckCodexModelSlugs_CodexHomeUsedLiterally_TrailingSpace
--- PASS: TestCheckCodexModelSlugs_CodexHomeUsedLiterally_TrailingSpace (0.00s)
=== RUN   TestCheckCodexModelSlugs_WhitespaceOnlyCodexHome_UsedLiterally
--- PASS: TestCheckCodexModelSlugs_WhitespaceOnlyCodexHome_UsedLiterally (0.00s)
=== RUN   TestCheckCodexModelSlugs_HomeUnresolvable_Info
--- PASS: TestCheckCodexModelSlugs_HomeUnresolvable_Info (0.00s)
=== RUN   TestCheckCodexModelSlugs_DetailCarriesExactCacheMtime
--- PASS: TestCheckCodexModelSlugs_DetailCarriesExactCacheMtime (0.00s)
=== RUN   TestCheckCodexModelSlugs_StaleCache_WarnCarriesExactMtimeAndStaleNote
--- PASS: TestCheckCodexModelSlugs_StaleCache_WarnCarriesExactMtimeAndStaleNote (0.00s)
=== RUN   TestCheckCodexModelSlugs_StaleCache_PassAlsoCarriesStaleNote
--- PASS: TestCheckCodexModelSlugs_StaleCache_PassAlsoCarriesStaleNote (0.00s)
=== RUN   TestCodexCacheFreshnessClause
=== RUN   TestCodexCacheFreshnessClause/fresh_cache_renders_mtime_and_age,_no_stale_note
=== RUN   TestCodexCacheFreshnessClause/exactly_24h_old_is_not_stale
=== RUN   TestCodexCacheFreshnessClause/24h_plus_one_second_is_stale
=== RUN   TestCodexCacheFreshnessClause/changed_wins_over_an_old_mtime
=== RUN   TestCodexCacheFreshnessClause/zero_mtime_(Stat_failed)_yields_an_empty_clause
--- PASS: TestCodexCacheFreshnessClause (0.00s)
=== RUN   TestFormatCacheAge
=== RUN   TestFormatCacheAge/negative_clamps_to_0m
=== RUN   TestFormatCacheAge/sub-minute_rounds_to_0m
=== RUN   TestFormatCacheAge/90_minutes_is_1h
=== RUN   TestFormatCacheAge/47_hours_is_1d
=== RUN   TestFormatCacheAge/49_hours_is_2d
--- PASS: TestFormatCacheAge (0.00s)
=== RUN   TestCheckCodexModelSlugs_CacheChangedWhileReading_FreshnessUnknown
--- PASS: TestCheckCodexModelSlugs_CacheChangedWhileReading_FreshnessUnknown (0.00s)
PASS
ok  	github.com/yoshpy-dev/ralph/internal/cli	0.313s
```

### Flakiness check: 6 new tests, `-count=3 -v`

All 18 runs (6 tests × 3 repeats) PASS, identical output each repeat, no
flakiness observed from the real-clock + `os.Chtimes` approach:

```
--- PASS: TestCheckCodexModelSlugs_DetailCarriesExactCacheMtime (0.00s)
--- PASS: TestCheckCodexModelSlugs_StaleCache_WarnCarriesExactMtimeAndStaleNote (0.00s)
--- PASS: TestCheckCodexModelSlugs_StaleCache_PassAlsoCarriesStaleNote (0.00s)
--- PASS: TestCodexCacheFreshnessClause (0.00s)
--- PASS: TestFormatCacheAge (0.00s)
--- PASS: TestCheckCodexModelSlugs_CacheChangedWhileReading_FreshnessUnknown (0.00s)
[x3, identical]
ok  	github.com/yoshpy-dev/ralph/internal/cli	0.313s
```

This is expected given the plan's own Assumptions: the tests that assert
exact mtime strings second-align via `Truncate(time.Second)` before calling
`os.Chtimes`, and `TestCodexCacheFreshnessClause` injects a fixed `now`
rather than reading the wall clock, so there is no timing race to flake on.

## Coverage

| Package | Coverage | Note |
| --- | --- | --- |
| `internal/cli` | 81.1% | +0.1pp vs. the 81.0% baseline recorded in the org-stop-reason-removal test report (2026-09-18, same day, prior plan). The new tests exercise previously-uncovered lines in `readCodexModelsCache` (the Stat-before/read/Stat-after branch and the seam call) and `codexCacheFreshnessClause`, so a small rise is expected. |

`go test ./internal/cli/... -cover -count=1` was run per-package rather than
a whole-repo profile, since the plan's test plan scopes coverage interest to
`internal/cli` (the only package this diff touches).

## Discrimination checks (reasoned from source, no source edits made)

The handoff asked me to trace three guard-discrimination points against the
actual code rather than by patching and reverting `internal/cli/doctor_codex_models.go`. All three traces are closed, deterministic calculations (no
live clock reads or probes in the discriminating branch), so tracing is
reliable here without an empirical mutation run.

**(a) Exact-RFC3339 assertions pin the file mtime, not `time.Now()`.**
`readCodexModelsCache` (`internal/cli/doctor_codex_models.go:66-89`) returns
`mtime = after.ModTime()` — the second `f.Stat()` result taken after
`io.ReadAll`, i.e. the file's actual on-disk mtime, which `os.Chtimes` set to
a known, second-aligned value in the tests. `codexCacheFreshnessClause`
(`:113-124`) renders `mtime.UTC().Format(time.RFC3339)` from that value, not
from a `time.Now()` call. `TestCheckCodexModelSlugs_DetailCarriesExactCacheMtime`
asserts the Detail contains `"cache written " + known.UTC().Format(time.RFC3339)`
as an exact substring, where `known = time.Now().Add(-5*time.Minute).Truncate(time.Second)`.
Confirmed: an implementation that rendered `time.Now()` at check time instead
of the stored mtime would produce a different (current) second in that
substring and fail this exact-match assertion — the test is discriminating.

**(b) The 24h-boundary pair pins strict `>`, not `>=`.** `codexCacheFreshnessClause`
appends the stale suffix `iff age > codexCacheStaleAfter` (`:122`, strict
greater-than). `TestCodexCacheFreshnessClause`'s "exactly 24h old is not
stale" case (`now.Add(-codexCacheStaleAfter)`) asserts `cache may be stale`
is **absent** (`wantLacks`); the adjacent "24h plus one second is stale" case
(`now.Add(-codexCacheStaleAfter - time.Second)`) asserts it is **present**
(`wantHas`). Confirmed: if the implementation used `>=` instead of `>`, the
exactly-24h case would incorrectly gain the stale suffix and its `wantLacks`
assertion would fail — the pair is discriminating for the boundary operator.

**(c) The changed-while-reading test would fail on a re-read or a missed
`freshness unknown` marker.** `readCodexModelsCache` captures `data` via
`io.ReadAll` *before* the `codexCacheBetweenReadAndStat` seam fires and the
second `f.Stat()` runs (`:73-84`); it never re-reads the file contents after
detecting the mismatch, it only re-derives `changed` from the two `Stat`
results and returns the already-read `data` unconditionally with a zero
`mtime`. `codexCacheFreshnessClause` checks `changed` first, before the
zero-mtime check (`:114-116`), so a changed cache always yields `"(cache
changed while reading; freshness unknown — re-run)"` regardless of the mtime
value. `TestCheckCodexModelSlugs_CacheChangedWhileReading_FreshnessUnknown`
sets the seam to rewrite the cache (dropping the missing slug) mid-read and
asserts: Status stays `warn` naming the originally-missing slug (not `pass`,
which is what a re-read of the rewritten cache would produce), Detail
contains `freshness unknown`, and Detail contains neither `cache may be
stale` nor `cache written`. Confirmed: (i) a re-read implementation would
flip the verdict to `pass` and fail the Status/slug-name assertions; (ii) an
implementation that omitted the `freshness unknown` clause, or that let the
zero-mtime/stale checks run ahead of the `changed` check, would fail the
`freshness unknown` presence assertion or the `cache written`/`cache may be
stale` absence assertions respectively — the test discriminates on both
axes independently. Confirmed as-run PASS (see targeted run above), and
stable across 3 repeats (see Flakiness check above).

## Test plan coverage vs. plan section

| Plan item | Status |
| --- | --- |
| Unit tests: `internal/cli`'s `TestCheckCodexModelSlugs_*` (8 pre-existing + 6 new) and `TestFormatCacheAge` | All 14 functions (23 assertions/subtests counting the two tables and the 5-subtest `TestCodexCacheFreshnessClause`) PASS individually and together. |
| Integration tests: none (plan states the CLI path is covered by existing doctor tests) | N/A, matches plan — no new integration surface. |
| Regression tests: `go test ./internal/... -count=1` | PASS, fresh, 8/8 packages. |
| Edge case (1): negative age (future mtime) clamps to `0m` | Confirmed — `TestFormatCacheAge`'s `"negative clamps to 0m"` row (`-time.Minute` → `"0m"`) PASS. |
| Edge case (2): Windows `os.Chtimes` skip — not applicable, CI targets darwin/linux | N/A per plan; not exercised (no Windows CI in this repo). |
| Edge case (3): empty-models-JSON cache (all slugs missing) + stale, folded into case (b) | Confirmed — `TestCheckCodexModelSlugs_StaleCache_WarnCarriesExactMtimeAndStaleNote` uses a cache with only `gpt-5.5` against a pool requesting `gpt-5.5` + `gpt-9000-retired`, i.e. a missing-slug + stale combination; PASS. A fully-empty-`models` cache specifically isn't separately tested, but the plan itself folds this edge case into test (b) rather than requiring a distinct fixture. |

## Coverage gaps

- **`f.Stat()` failure path is untested by design** (plan's own recorded
  Known gap, AC-4). `readCodexModelsCache` has two `if statErr != nil`
  branches (`:74-77`, `:81-83`) that return a zero mtime with
  `changed=false`, which `codexCacheFreshnessClause` renders as an empty
  clause. No test forces a real `Stat` failure (would need an
  fs-abstraction/fault-injection seam the plan's Non-goals deliberately did
  not add). Confirmed present as a code branch by reading `:74-83`; not
  reachable from any of the 14 tests above. This matches both the plan text
  and the verify report's AC-4 note — not a new finding.
- **No test exercises a `Stat`-fails-then-succeeds partial scenario** (e.g.
  first `Stat` fails but a hypothetical retry path exists) — not applicable,
  the implementation has no retry logic, so there's nothing to under-test
  here beyond the single-failure branch above.
- **Doc-only changes** (`.claude/skills/org/SKILL.md` + 3 mirrors,
  `docs/specs/2026-08-01-org-runtime.md`) have no executable test — expected;
  already grep/`cmp`-verified by `/verify` (AC-5/AC-6).

## Failure analysis

None. No test failed in any run (targeted, full package, `-count=3` repeat,
or full wrapper).

## Regression checks

| Previously passing behavior | Status | Evidence |
| --- | --- | --- |
| 8 pre-existing `TestCheckCodexModelSlugs_*` tests | PASS, no regressions | Targeted `-v` run above; all Detail-prefix assertions from before this PR are unaffected by the new suffix-only Detail changes. |
| `go test ./internal/... -count=1` (8 packages) | PASS, fresh | Full run above. |
| `./scripts/run-test.sh` (28 shell suites + golang verifier) | PASS, exit 0 | Wrapper log, `docs/evidence/test-2026-09-18-doctor-codex-slug-cache-mtime.log` |
| Known flaky tests (`TestRunDoctorOpts_ProbeModelsFalse_NoSubprocess`, `TestRunWatcher_TimeoutIndependentOfSmallInterval`, per prior tester-agent memory) | Did not flake this run | Both included in the fresh `go test ./internal/...` pass (`internal/cli` 35.228s ok, `internal/org` 9.601s ok) |

## Verdict

- Pass: YES — 28 shell test files + 8 Go packages green (both via the
  wrapper and a fresh uncached full run), all 14 targeted
  `TestCheckCodexModelSlugs_*`/`TestFormatCacheAge`/`TestCodexCacheFreshnessClause`
  tests individually confirmed PASS with verbose output, 6 new tests stable
  across a `-count=3` repeat, `internal/cli` coverage 81.1% (+0.1pp, expected
  from new coverage, not a regression), all three requested discrimination
  checks confirmed true by tracing the shipped source.
- Fail: NO
- Blocked: NO — safe to proceed to `/pr` (AC-7's `go test`/`run-test.sh`
  half is now satisfied; the `Closes #159` half is `/pr`'s job as the plan
  states).
