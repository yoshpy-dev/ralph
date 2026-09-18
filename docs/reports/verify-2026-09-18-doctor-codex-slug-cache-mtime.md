# Verify report: doctor-codex-slug-cache-mtime

- Date: 2026-09-18
- Plan: `docs/plans/active/2026-09-18-doctor-codex-slug-cache-mtime.md` (issue #159)
- Verifier: `verifier` subagent (Claude Code)
- Scope: spec compliance (AC-1..AC-6; AC-7 split, see below) + static analysis for
  `git diff 26ca767...HEAD` on `feat/doctor-codex-slug-cache-mtime` (HEAD `0c00a9d`,
  10 commits, worktree clean). No tests were run (that is `/test`'s job).

## Overall verdict: PASS

All acceptance criteria the plan assigns to `/verify` are satisfied by evidence
gathered directly against the current tree (code read, greps, `cmp`, sync scripts,
`./scripts/run-static-verify.sh`). No spec/plan/implementation drift found beyond
one already-recorded, already-fixed self-review follow-up (confirmed closed here).

## Acceptance criteria

| AC | Verdict | Evidence |
| --- | --- | --- |
| AC-1 | PASS | `internal/cli/doctor_codex_models.go:113-124` (`codexCacheFreshnessClause`) renders `fmt.Sprintf(" (cache written %s, %s ago)", mtime.UTC().Format(time.RFC3339), formatCacheAge(age))` from the `mtime` returned by `readCodexModelsCache`, not `time.Now()`. `TestCheckCodexModelSlugs_DetailCarriesExactCacheMtime` (`internal/cli/doctor_org_test.go:608-636`) sets a known, second-aligned mtime via `os.Chtimes` and asserts the Detail contains `"cache written " + known.UTC().Format(time.RFC3339)` and `"5m ago"` as an exact substring — an implementation rendering `time.Now()` would not produce this string. `TestCheckCodexModelSlugs_StaleCache_WarnCarriesExactMtimeAndStaleNote` (`:641-673`) repeats the exact-mtime assertion on the warn path. |
| AC-2 | PASS | `codexCacheFreshnessClause` appends `"; cache may be stale — launch codex once to refresh, then re-run"` iff `age > codexCacheStaleAfter` (`:122-123`, `codexCacheStaleAfter = 24 * time.Hour` at `:23`), on both the warn (`:238-241`) and pass (`:243-244`) Detail — the `+freshness` suffix is applied identically to both branches. `TestCheckCodexModelSlugs_StaleCache_WarnCarriesExactMtimeAndStaleNote` (48h-old cache) asserts the stale note on warn; `TestCheckCodexModelSlugs_StaleCache_PassAlsoCarriesStaleNote` (`:678-699`) asserts it on pass; `TestCheckCodexModelSlugs_DetailCarriesExactCacheMtime` (5-minute-old cache) asserts the note's *absence*. All three pass conditions from the plan's tests (a)/(b)/(c) are present. |
| AC-3 | PASS | `TestFormatCacheAge` (`internal/cli/doctor_org_test.go:748-767`) is a 5-row table: `-1m→0m`, `30s→0m`, `90m→1h`, `47h→1d`, `49h→2d` — matches the plan's table exactly, including the negative-clamp edge case from the Test plan. `formatCacheAge` (`:96-111`) implements the Nm/Nh/Nd switch with a `d < 0 → d = 0` clamp and a `hoursPerDay` constant scoped inside the function (self-review L5 fix). Status-unchanged claim verified structurally: `git diff 26ca767...HEAD -- internal/cli/doctor_org_test.go` contains zero `-` lines (diff is purely additive), so all 8 pre-existing tests and their `r.Status` assertions are byte-unmodified; combined with the info-branch-literal check under "Invariant checks" below, Status values for every outcome are unchanged. |
| AC-4 | PASS (with the plan's own recorded Known gap, not a verify blocker) | `readCodexModelsCache` (`:66-89`) opens the file once, Stats before `io.ReadAll`, Stats again after, and returns `changed=true` with a zero mtime when `!before.ModTime().Equal(after.ModTime()) \|\| before.Size() != after.Size()` (`:86-88`). `codexCacheFreshnessClause` maps `changed=true` to `" (cache changed while reading; freshness unknown — re-run)"` and that branch is checked first, before the zero-mtime check, so it wins over any mtime value (`:114-116`). `TestCheckCodexModelSlugs_CacheChangedWhileReading_FreshnessUnknown` (`:779-815`) drives this via the `codexCacheBetweenReadAndStat` seam, rewriting the cache mid-read, and asserts the verdict (`warn`, naming `gpt-9000-retired`) comes from the bytes read *before* the rewrite while the Detail carries `freshness unknown` and neither `cache may be stale` nor `cache written`. Seam placement matches the doc comment exactly (self-review L4 fix): the seam call (`:78-80`) sits after the first Stat's error return (`:75-77`) and before the second Stat (`:81`), so it never fires on the first-Stat-failure path. `TestCodexCacheFreshnessClause` (`:706-743`) additionally pins the clause builder's `changed`-wins-over-mtime precedence and the zero-mtime-yields-empty-string case with an injected `now`, closing the L2 follow-up. The plan's own Known gap — an actual `f.Stat()` failure is not exercised by any test, only reasoned about by code read — remains true and is not something `/verify` can close by static analysis; it is explicitly out of `/test`'s and `/verify`'s reach without a fault-injectable `fs` abstraction, which the plan's Non-goals and Design decisions deliberately did not introduce. |
| AC-5 | PASS | `grep -c '鮮度確認もしない' .claude/skills/org/SKILL.md` = 0. `.claude/skills/org/SKILL.md:61-64` contains `stale 注記` (`"...cache の書き込み時刻を UTC で Detail に出し、24 時間より古ければ stale 注記が付く)ので..."`). `cmp` confirms `.claude/skills/org/SKILL.md`, `.agents/skills/org/SKILL.md`, `templates/base/.claude/skills/org/SKILL.md`, `templates/base/.agents/skills/org/SKILL.md` are byte-identical. `./scripts/check-skill-sync.sh` → `[ok] check-skill-sync: 13 skill(s) in lock-step` (exit 0). `./scripts/check-sync.sh` → `DRIFTED: 0`, `PASS: all files in sync.` (exit 0). |
| AC-6 | PASS | `grep -c '鮮度確認もしない' docs/specs/2026-08-01-org-runtime.md` = 0. Section (d) (`docs/specs/2026-08-01-org-runtime.md:22`) contains both `` `cache written <UTC>` `` and `stale 注記`, and explicitly cites `issue #159`. The observation procedure was also simplified as the plan's Verify plan asked ("`stat` を任意に"): the rewritten step reads the Detail's `cache written` (UTC, +9h JST note) or optionally `stat`, and adds that a just-refreshed cache reads `<age> ago` = `0m`, removing the need to do timezone arithmetic — this closes self-review finding M1. No other mention of `checkCodexModelSlugs` exists in the spec (`grep -n 'checkCodexModelSlugs'` returns only the unchanged intro at line 17 and the rewritten (d) at line 22). |
| AC-7 | SPLIT — not fully verifier's scope | `go test ./internal/cli/... -count=1` and `./scripts/run-test.sh` (or `./scripts/run-verify.sh` for the full gate) are `/test`'s responsibility per the pipeline contract and were **not run here**, per the assignment. `./scripts/run-static-verify.sh` (this report's own scope) passed — see below. The PR-body "Closes #159" half is unverifiable until `/pr` runs; deferred as instructed. |

## Static analysis

`RALPH_VERIFY_SCOPE=changed ./scripts/run-static-verify.sh` → **exit 0**, evidence
saved to `docs/evidence/verify-2026-09-18-053206.log` (gitignored, per repo convention).
Notable checks inside the run, all green:

- `scripts/check-sync.sh` — `DRIFTED: 0`
- `scripts/check-skill-sync.sh` — `13 skill(s) in lock-step`
- `scripts/check-pipeline-sync.sh` — all referencing docs in sync
- `scripts/check-template-purity.sh` — no meta-repo-specific references in templates
- Go verifier (`packs/languages/golang/verify.sh`), scoped to this branch's changed
  language: `gofmt: ok`, `go vet` silent (pass), `golangci-lint` → `0 issues.`,
  `staticcheck` silent (pass; binary present at `~/go/bin/staticcheck`, confirmed
  via `command -v staticcheck`)

Cross-checked independently, outside the wrapper, for the two touched Go files:
`gofmt -l internal/cli/doctor_codex_models.go internal/cli/doctor_org_test.go` →
no output (clean), `go vet ./internal/cli/...` → exit 0, silent.

## Invariant checks (requested by the task brief)

1. **`checkCodexModelSlugs` Status values unchanged.** Confirmed via `git diff
   26ca767...HEAD -- internal/cli/doctor_codex_models.go`: all four `info`
   branches (empty pool `:186-189`, cache-path resolution error `:192-196`,
   cache-missing/unreadable `:203-211`, invalid-JSON `:215-219`) are present in
   the diff only as unchanged context lines — no `Status =` or `Detail =`
   literal in those branches is touched. Only the `warn` (`:238-241`) and
   `pass` (`:243-244`) branches gained a `+ freshness` suffix appended
   *outside* the existing `fmt.Sprintf(...)` call, so the pre-existing format
   strings for those two Statuses are byte-identical to base as well; only the
   trailing bytes are new.
2. **`readCodexModelsCache` single-handle read + before/after Stat.** Confirmed
   by reading `internal/cli/doctor_codex_models.go:66-89`: one `os.Open`, Stat
   before `io.ReadAll`, Stat after, `changed=true`+zero mtime on ModTime/Size
   mismatch, zero mtime + `changed=false` when either Stat fails, and the seam
   `codexCacheBetweenReadAndStat` fires only between the successful first Stat
   and the second Stat (guarded by the first Stat's early-return check ahead
   of it) — matches the task brief's stated invariant exactly.
3. **`codexCacheFreshnessClause` precedence.** Confirmed: `changed` branch
   checked first → `"freshness unknown"`; then zero-mtime → `""`; else the
   `cache written ... ago` clause with the stale suffix iff `age >
   codexCacheStaleAfter` (strict `>`, pinned by `TestCodexCacheFreshnessClause`'s
   "exactly 24h old is not stale" / "24h plus one second is stale" pair).
4. **`formatCacheAge` clamping and granularity.** Confirmed via code and the
   5-row table test.
5. **Test coverage for AC-1..AC-4 with second-aligned `os.Chtimes` mtimes and
   exact RFC3339 substring assertions.** Confirmed — see AC-1/AC-2/AC-4 rows
   above.
6. **Skill (4 mirrors) and spec (d) prose accuracy, including "UTC" and `<age>
   ago`.** Confirmed — see AC-5/AC-6 rows above.
7. **`os.ReadFile` no longer appears in the file.** Confirmed:
   `grep -n 'os.ReadFile' internal/cli/doctor_codex_models.go` → no match.

## Documentation drift check

- Plan's Deviation notes accurately describe the implementation: the "clock
  注入は入れない" reversal is real (`codexCacheFreshnessClause` takes `now
  time.Time`, and `TestCodexCacheFreshnessClause` uses an injected `now` —
  `checkCodexModelSlugs` itself still calls `time.Now()` at its one call site,
  `:227`, matching the plan's note that the check-level tests stay on real
  time + `os.Chtimes`).
- Self-review's four "New issues introduced by the fix" cosmetic follow-ups
  (orphaned `TestFormatCacheAge` doc comment, a 111-column comment line, an
  83-column skill line across 4 mirrors, a displaced `stat` qualifier in spec
  (d)) are **all confirmed fixed** in `5143e3b` (`git show 5143e3b --stat` +
  diff read): the doc comment sits against `TestFormatCacheAge` again, the
  comment line is re-wrapped, the skill line is re-wrapped in all 4 mirrors
  (`cmp` still confirms byte-identity post-fix), and the `stat` qualifier moved
  back beside the confirmation clause. This closes the self-review's own
  "record as debt if not swept" contingency — no batched tech-debt row was
  needed because the sweep happened before this verify pass.
- `docs/tech-debt/README.md`: not touched by this diff (`git diff
  26ca767...HEAD --name-only` does not list it), consistent with the plan's
  Non-goal ("該当行なし") and the self-review's own cross-check.
- No stray references to the retired phrase "鮮度確認もしない" remain in either
  file the plan targeted for update. A repo-wide grep does surface the phrase
  in `docs/plans/archive/2026-09-18-org-codex-slug-update-procedure.md`,
  `docs/reports/self-review-2026-09-18-org-codex-slug-update-procedure.md`,
  `docs/reports/walkthrough-2026-09-18-org-codex-slug-update-procedure.md`,
  and this plan's own "Related request" narrative (`docs/plans/active/2026-09-18-doctor-codex-slug-cache-mtime.md:6`,
  quoting the old wording as historical context) — all are historical
  snapshots or the current plan's own framing, not live documentation the
  plan claims to update, so this is not drift.

## Known gaps (carried from self-review, not closed by this verify pass)

- AC-4's `f.Stat()` failure path is untested by any unit test (reasoned about
  by code read only, as the plan itself records). Closing it would need a
  fault-injectable filesystem seam, which both the plan's Design decisions and
  Non-goals deliberately declined to add for this issue's scope.
- No test suite, `go test`, or binary execution occurred in this verify pass —
  by design, per the pipeline contract (`/test` owns behavioral verification).
- Whether the 24h staleness threshold is operationally correct is a product
  judgment, explicitly out of scope for both self-review and verify.

## Evidence log

- `./scripts/run-static-verify.sh` full output (this session) — exit 0,
  `docs/evidence/verify-2026-09-18-053206.log`
- `gofmt -l` / `go vet ./internal/cli/...` direct run — both clean
- `./scripts/check-skill-sync.sh` — `[ok] check-skill-sync: 13 skill(s) in lock-step`
- `./scripts/check-sync.sh` — `DRIFTED: 0`, `PASS: all files in sync.`
- `cmp` of the 4 skill mirrors — byte-identical
- `git diff 26ca767...HEAD -- internal/cli/doctor_org_test.go` — 0 removed lines (purely additive)
- `git diff 26ca767...HEAD -- internal/cli/doctor_codex_models.go` — full hunk read
- `git show 5143e3b` — full diff read, confirms all 4 cosmetic follow-ups closed
- `python3 -m json.tool` on `docs/insights/events/2026-09-18-doctor-codex-slug-cache-mtime.jsonl` — valid JSON
