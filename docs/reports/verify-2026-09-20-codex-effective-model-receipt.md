# Verify report: codex-effective-model-receipt

- Date: 2026-09-20
- Plan: `docs/plans/active/2026-09-20-codex-effective-model-receipt.md` (issue #165)
- Verifier: `verifier` subagent (Claude Code), standard flow, pipeline cycle 1 of cap 2
- Scope: spec compliance (AC-1..AC-11) + static analysis + documentation drift for
  `git diff main...HEAD` on `feat/codex-effective-model-receipt` (base `642214d`,
  HEAD `ccd3d2f`, working tree clean at start and end). No tests run — that is
  `/test`'s job.

## Verdict

**PASS.** No AC is not-met. AC-11's test-execution half and `Closes #165` half are
out of scope here (owned by `/test` and `/pr` respectively). All requested static
checks are green. No documentation drift found beyond one pre-existing, already-approved
`KNOWN_DIFF` (`model-routing.md`, unrelated to this PR).

## AC table

| AC | Status | Evidence |
|----|--------|----------|
| AC-1 | Met | `ObserveCodexEffectiveModel`/`scanRolloutRecord` (`internal/org/codex_session.go:137-181,309-372`) return `matched=false` (never a guessed model) on: no pointer-sentence match, a `session_meta` timestamp before cutoff (`:356-360`, AC-2b's own exclusion), a non-regular file (Lstat + `IsRegular()` check, `:215-218`), a malformed JSON line (`json.Unmarshal` error swallowed, `:337`), and a line over `codexObserveMaxLineBytes` (length-gated before decode, `:333`). Tests: `TestObserveCodexEffectiveModel_PromptPathAbsent`, `_MalformedLineSkipped`, `_OversizedLineSkipped`, `_SymlinkIgnored`, `_FIFODoesNotHang` (`internal/org/codex_session_test.go`). No panic path found by read (every decode failure degrades to a zero-value ignore, not a fatal). |
| AC-2 | Met | `codexFoundReceipt` (`internal/org/spawn.go` — see diff hunk) sets `Honored=HonoredTrue`/`Reason="codex session record reports the commanded model"` on match, `Honored=HonoredFalse` + a `Reason` naming both the commanded and effective model on mismatch. Test: `TestOrgSpawn_Codex_ModelObservation_Found` (both branches; `internal/org/spawn_test.go:2160`) plus the real-record fixture reproduction in `docs/evidence/codex-effective-model-receipt-2026-09-20.md` Run E1 (`gpt-5.5` → `gpt-5.6-sol`). |
| AC-2b | Met | `codexSessionMetaQualifies` (`codex_session.go:378-389`) disqualifies any record whose `session_meta.timestamp` is before `cutoff` (spawn start, truncated to whole seconds), and `scanRolloutRecord` returns immediately once that disqualification is known (`:356-360`) — no later line in the file can override it. Two qualifying records both after cutoff return `CodexObservationAmbiguous` (`ObserveCodexEffectiveModel`, `matchCount>=2` branch, `:167-179`), never a guess. Tests: `TestObserveCodexEffectiveModel_OldSessionUpdatedLaterNotPicked`, `_TwoQualifyingRecordsAmbiguous` (`codex_session_test.go`); `TestOrgSpawn_Codex_OldSessionRecordNotPickedOnRespawn`, `TestOrgSpawn_Codex_ModelObservation_Ambiguous` (`spawn_test.go`) exercise it through `Spawn` itself. |
| AC-2c | Met | `observeCodexSpawnReceipt` (`spawn.go`) returns the "no role-prompt file" unknown receipt immediately when `promptPath == ""`, before ever calling `ObserveCodexEffectiveModel` — no wait. Test: `TestOrgSpawn_Codex_InlinePromptSkipsObservation_NoWaiting` asserts elapsed `< 2s` against an 8s-class configured timeout (`spawn_test.go:2422`). |
| AC-3 | Met | `observeCodexSpawnReceipt`'s poll loop returns `codexUnknownReceipt(base, codexNotFoundReason)` once `o.codexModelObserveTimeout()` elapses with nothing found, and `Spawn` still appends `spawned` before this step runs and always appends the receipt regardless of outcome (`spawn.go`, ~line 833 area in the diff). `ObserveCodexEffectiveModel` itself treats a missing `sessionsDir` (`os.Stat` failure) as `CodexObservationNotFound, nil` — not an error (`codex_session.go:141-145`) — so a missing/unreadable directory never fails the caller. Tests: `TestOrgSpawn_Codex_ModelObservation_NotFoundAfterTimeout`, `_ReadError_DistinctReason` (`spawn_test.go`). |
| AC-4 | Met | In `Spawn`, the receipt branch is `if p.Driver == "codex" { ... } else { receipt.Honored = HonoredUnknown; receipt.Reason = "interactive session; effective model not yet observable" }` — byte-identical to the pre-PR text for any non-codex driver. `dryRunSpawn` always writes `Honored: HonoredUnknown, Reason: "dry-run"` regardless of `p.Driver` (unchanged branch). Test: `TestOrgSpawn_Claude_ModelReceiptTextUnchanged` (`spawn_test.go:2453`); dry-run regression: `TestOrgSpawn_Codex_DryRun_ModelReceiptUnchanged` (`spawn_test.go:2474`). |
| AC-5 | Met | `observeStopModelReceipt` (`verbs.go`) only calls `ObserveCodexEffectiveModel` when `seat.Driver == "codex"`, not dry-run, `codexSpawnCorrelation` resolves a spawn_started + non-empty `promptPath`, and `hasObservedCodexReceiptSince` finds no receipt since that spawn with a non-empty `ReportedEffectiveModel` — which is true whether zero receipts exist (interrupted mid-poll) or only a rejection/dry-run receipt exists (neither carries a reported model), matching the plan's AC-5 wording. `codexSpawnCorrelation` filters `ev.DryRun` in both its spawn_started and spawn_step scans (self-review M2 fix), so a `--dry-run` respawn of an already-spawned seat cannot displace the real correlation. Only a `CodexObservationFound` result appends a receipt (`if obs.Status != CodexObservationFound { return Receipt{}, false }`). Tests: `TestOrgStop_Codex_AppendsWhenSpawnReceiptWasUnknown`, `_HonoredFalse_ObservedAtStop`, `_DoesNotAppendWhenAlreadyObserved`, `_AppendsWhenNoReceiptAtAllForThisSpawn`, `_AppendsWhenOnlyRejectionReceiptExistsSince`, `_DryRunRespawnDoesNotDisplaceRealSpawnCorrelation`, `_NothingAppended_NotFound`, `_DryRun_NoObservationAttempted`, `_NoPromptFileSeat_NothingAppended`, `_ReceiptsAppendFailureLeavesStopSuccessful` (`verbs_test.go:1273-1685`) — all ten named states in the plan's Test plan edge-case list are covered. |
| AC-6 | Met | `printCodexModelMismatchWarning` (`internal/cli/org.go`) gates on `r.Honored != org.HonoredFalse \|\| r.ReportedEffectiveModel == ""` returning early, so only a genuine observed mismatch (not an envelope rejection, which is `honored=false` with no reported model) prints. Called from `printSpawnResult` (used by both `spawn` and `start`, `org.go:315,425`) and directly in `stop` (`org.go:685`). It writes to `cmd.ErrOrStderr()` only and never touches the command's returned error / exit path. Tests: `TestOrgSpawn_CLI_CodexModelMismatch_PrintsWarning`, `_CodexModelMatch_NoWarning`, `TestOrgStop_CLI_CodexModelMismatch_PrintsWarning`, `_CodexModelMatch_NoWarning` (`internal/cli/org_test.go:1135-1257`); self-review's dedicated negative test for the rejection case, `TestOrgSpawn_CLI_EnvelopeRejection_NoModelWarning`, exists per the self-review report (positive notes) — confirmed present at `org_test.go` by grep. |
| AC-7 | Met | `codexRetirementClause` (`internal/cli/doctor_codex_models.go`) reports `retiring:`/`retired:` + `codexRetirementSentence` only for pool slugs present in the cache with a `parseCodexModelUpgrade`-usable `upgrade`; `checkCodexModelSlugs` prioritizes `warn` (missing slugs) over the new `info` branch (`if len(missing) > 0 { ... return r }` runs first), and falls back to `pass` when `retirement == ""`. `parseCodexModelUpgrade` returns `ok=false` (silently ignored, not a check failure) for JSON `null`, an empty/absent field, an unparsable object, or an empty `model` — and a missing/unparsable `retirement_at` still returns `ok=true, hasDate=false` so the slug is named without a date rather than dropped. Tests: `TestCodexRetirementClause`, `TestCheckCodexModelSlugs_Retiring_InfoWithClause`, `_Retired_InfoWithClause`, `_MissingWins_RetiringClauseStillAppended`, `_WrongTypedUpgrade_SlugStillPresentDocStillDecodes` (`internal/cli/doctor_org_test.go:877-1110`). |
| AC-8 | Met | `internal/org/spawn_test.go`'s `testOrg(t)` sets `CodexSessionsDir: filepath.Join(dir, "codex-sessions")` (a fresh `t.TempDir()` subpath that does not exist) and both observe durations to `time.Millisecond` (`spawn_test.go:246-260`) for every `Org` the package's tests construct. `internal/cli/main_test.go`'s `TestMain` pins the package-level `orgCodexSessionsDirOverride` to a nonexistent path under `os.TempDir()` and both duration overrides to tens of milliseconds, applied to every `org.Org` `newOrgRuntimeAt` builds. Neither seam reads `CODEX_HOME`/`os.UserHomeDir()`; `TMPDIR=/tmp` only relocates `os.TempDir()`'s own base, which both seams already route through — not independently re-verified by re-running with `TMPDIR=/tmp` in this pass (would require `go test`, out of `/verify`'s scope; noted as unverified below). |
| AC-9 | Met | `docs/evidence/codex-effective-model-receipt-2026-09-20.md` records five real-record runs (model + status + elapsed only) with two negative-control cases; its own "何を確認したか"/"記録には会話の本文が入っている…" framing states the redaction intent, and a full read of the file (`Read` tool) found no line content, no user-message text, and no full file paths beyond the documented `<state-dir>/prompts/<org>_<seat>.md` shape. Home paths are redacted to `~`. |
| AC-10 | Met | See Documentation drift section below — all six named surfaces plus the two skill mirrors were updated and are internally consistent with the code at HEAD; `check-skill-sync.sh` / `check-sync.sh` / `check-template-purity.sh` all pass (see Static analysis). |
| AC-11 | Not applicable to `/verify` | Test execution is `/test`'s scope; `Closes #165` is `/pr`'s scope. Not evaluated here. |

## Static analysis results

| Command | Result |
|---|---|
| `HARNESS_VERIFY_MODE=static ./scripts/run-static-verify.sh` | PASS (exit 0). Includes `check-sync.sh` (DRIFTED: 0), `check-pipeline-sync.sh` (ok), `check-skill-sync.sh` (13 skills in lock-step), `check-template-purity.sh` (PASS), and the golang pack verifier (`gofmt: ok`, `0 issues.` from golangci-lint) scoped to changed languages. |
| `gofmt -l internal/org internal/cli` | Clean (no output). |
| `go vet ./internal/org/... ./internal/cli/...` | Clean (no output). |
| `./scripts/check-skill-sync.sh` | PASS, run standalone. |
| `./scripts/check-sync.sh` | PASS, run standalone (`DRIFTED: 0`, `KNOWN_DIFF: 5` — see below). |
| `./scripts/check-template-purity.sh` | PASS, run standalone. |

Full command transcripts: `docs/evidence/verify-2026-09-20-codex-effective-model-receipt.log` (gitignored per `docs/evidence/*.log`, not committed — consistent with prior verify cycles in this repo).

## Documentation drift

None found in the surfaces the plan names as Scope item 9. Each of the following
was read at HEAD and cross-checked against the corresponding code:

- `.claude/rules/ralph/model-routing.md` ("Org runtime model receipts", new
  paragraph after the existing one): states the 8 s window, `$CODEX_HOME/sessions/`
  else `~/.codex/sessions/`, `turn_context.model` as `reported_effective_model`,
  the `honored=true|false` split, the "record matched by the pointer sentence
  and a session start not before the spawn" rule, all four `unknown` causes
  (no turn, inline/empty prompt, differing `CODEX_HOME`, plus implicitly
  ambiguous/timeout via "cannot be identified"), the stop-appends-only-when-found
  rule, and "claude seats are not observed and stay unknown" — all match the
  code read above. `templates/base/.claude/rules/ralph/model-routing.md` carries
  the same paragraph minus the `internal/org/codex_session.go` path reference,
  consistent with the pre-existing, already-approved `KNOWN_DIFF` entry for this
  file in `scripts/check-sync.sh` (meta-only internal-path references are
  intentionally omitted from the template copy — same pattern already used for
  the untouched `internal/org/receipts.go` reference one paragraph above).
- `docs/specs/2026-08-01-org-runtime.md` FR-9: the added clause "codex 座席は
  codex の session 記録の `turn_context.model` を観測する。claude 座席は未観測で
  `unknown`。#165" is accurate and consistent with AC-2/AC-4.
- `docs/recipes/codex-seat-permissions.md` (+ template copy, byte-identical
  diffs): the added sentences about the retirement dialog, `reported_effective_model`,
  the stderr warning, `ralph doctor`'s retiring-slug list, and "stop 時に
  もう一度だけ探し、見つかれば追記する" match the code.
- `.claude/skills/org/SKILL.md` + 3 mirrors (`.agents/skills/org/SKILL.md`,
  `templates/base/.claude/skills/org/SKILL.md`, `templates/base/.agents/skills/org/SKILL.md`):
  all four hunks are byte-identical (diffed pairwise); content matches the
  code (8 s window, `honored=false` + stderr warning, `ralph doctor`'s
  retiring-slug display, the three `unknown` causes, "claude 座席は観測しない").
- `docs/evidence/codex-effective-model-receipt-2026-09-20.md`: the doctor
  Detail line quoted in "`ralph doctor` の表示" ("info — 5 codex model_pool
  slug(s) present in ~/.codex/models_cache.json; retiring: gpt-5.5 ->
  gpt-5.6-sol on 2026-10-14. codex may run the replacement instead of the
  commanded model; when ralph can identify the seat's codex session record,
  it records the mismatch as honored=false in the org model receipts.")
  matches `codexRetirementClause`'s construction and `codexRetirementSentence`'s
  current text (`internal/cli/doctor_codex_models.go`) word-for-word.
- `docs/evidence/codex-seat-permissions-2026-09-18.md` P5: the plan's Scope
  table calls for "1 行追記"; confirmed present (checked via the file's diff
  in `git diff main...HEAD` — not reproduced here per the pointers-not-dumps
  convention).

Surfaces checked for staleness but found **not** to need updating (no false or
overstated claim found):

- `README.md`, `AGENTS.md` (repo-map lines for `internal/org/` and
  `internal/insights/`) — neither names the doctor slug check or model
  receipts at a level of detail this PR would invalidate.
- `docs/insights/README.md` — its `honored`/`driver`/`effective_model` field
  rows (e.g. "`false` for Codex driver (known gap)") describe a *different*,
  historical mechanism: the retired `ralph-pipeline.sh`'s `source:pipeline`
  events for `/work` subagent model routing, not org-runtime spawn receipts.
  Its own "Org runtime receipts(参照)" section (bottom of the file, unchanged
  by this PR) already correctly points to `model-routing.md`'s "Org runtime
  model receipts" section as the schema's source of truth, and that section
  is exactly what this PR updated.
- `.claude/rules/ralph/agent-messaging.md` — no mention of model receipts;
  out of this feature's scope (message protocol, not spawn/stop receipts).
- `internal/org/prompts/lead.md` — grepped for `honored`/`model.receipt`/
  `reported_effective_model`; no hits. Lead is not told about receipts in
  its role prompt today; this predates the PR and the plan's Scope item 9
  does not name this file, so not a drift introduced here.
- `docs/tech-debt/README.md` — scanned rows mentioning `codex model slug`;
  none describes the pre-PR receipt-observation gap or would be invalidated
  by this PR's fix (the one related closed row, about `doctor`'s slug-check
  doc comment, was already resolved in an earlier PR and is unaffected).

One pre-existing item, not introduced by this PR: `scripts/check-sync.sh`
carries `model-routing.md` as a `KNOWN_DIFF` (root documents the Go-layer
default `internal/config` reference, which does not exist in scaffolded
projects) — reported as INFO, not DRIFTED, by `check-sync.sh` itself; this
PR's new paragraph follows the same established omission pattern rather than
introducing a new inconsistency.

## Non-goals held

- No pane-content reading: grepped the diff for `PaneRead`/`pane.*read` inside
  the new observation code — none; `ObserveCodexEffectiveModel` only opens
  files under `sessionsDir`.
- Receipts and manifest schemas unchanged: `git diff main...HEAD -- internal/org/receipts.go internal/org/manifest.go` is empty.
- No new `ralph.toml` key: `git diff main...HEAD -- ralph.toml templates/base/ralph.toml internal/config/` is empty.
- Claude seats' receipt text unchanged: see AC-4.
- `ralph insights` aggregation untouched: `git diff main...HEAD -- internal/insights/` is empty.

## Privacy property (static reading only)

Confirmed by reading `internal/org/codex_session.go` in full: the only
non-nil error `ObserveCodexEffectiveModel`/`scanRolloutRecord` can return is
built at `codex_session.go:315` as `fmt.Errorf("org: open codex session
record %s: %w", path, openErr)` — an operation description, the candidate
file's own path, and the wrapped `os.PathError`; no line content, no decoded
field, no session ID beyond the filename ever appears in it. `CodexModelObservation`
carries exactly two fields (`Model string`, `Status CodexObservationStatus`);
nothing else crosses the function boundary. Both production call sites
discard the error entirely: `spawn.go`'s `observeCodexSpawnReceipt` only
reads it to choose between two fixed reason-string constants
(`codexReadErrorReason` vs `codexNotFoundReason`, neither containing the
error's own text), and `verbs.go:781`'s `obs, _ := ObserveCodexEffectiveModel(...)`
drops it outright. `git diff main...HEAD -- internal/org internal/cli | grep
't\.Logf\|log\.Print'` on added lines returned nothing.

## What remains unverified

- Whether the code compiles/vets/tests cleanly under `go test` — not run, by
  design; `/test`'s scope.
- AC-8's `TMPDIR=/tmp` clause was checked by code read (both test seams route
  through `os.TempDir()`/an explicit `t.TempDir()`, neither reads `CODEX_HOME`
  or `HOME` directly), not by an actual `TMPDIR=/tmp go test` run — that run
  belongs to `/test`.
- AC-11's `Closes #165` PR-body requirement — `/pr`'s scope, not evaluated.
- End-to-end behavior against a live codex seat — the evidence file itself
  and the self-review both already list this as unverified; unchanged since
  self-review.
- The self-review's L7 residual note (pointer-sentence match verified against
  the same five real records, not against a fresh codex 0.154.0 run) —
  confirmed the code now matches on `promptFilePointer(promptPath)` rather
  than the bare path (`codex_session.go:416-436`), consistent with the
  self-review's stated fix and the evidence file's "26b03ce では座席の特定を…"
  note; not independently re-run against new real records in this pass.

## Insight event

Appended via `./scripts/insights-append.sh --slug codex-effective-model-receipt --flow standard --phase verify --verdict pass --source skill --cycle 1`.

## Cycle 2 (2026-09-20)

- Verifier: `verifier` subagent (Claude Code), standard flow, pipeline cycle 2 of cap 2 (the final cycle).
- Scope: spec compliance (AC-1..AC-11) + static analysis + documentation drift for
  `git diff main...HEAD` on `feat/codex-effective-model-receipt`, re-derived against
  HEAD `a4f6db2` (working tree clean at start and end). The cycle-1 section above
  describes HEAD `ccd3d2f` and names `hasObservedCodexReceiptSince`, which no
  longer exists at HEAD (renamed to `hasObservedCodexReceiptAfter` — see AC-5
  below) — every row in this section is re-derived from scratch against `a4f6db2`,
  per self-review cycle-2 finding C2-6, not carried forward. No tests run — that
  is `/test`'s job.
- What changed since the cycle-1 verify commit (`3031afc`): two test-only commits
  (`771e446`, `a7180ca`); a sync-docs commit (`27b6759`, one skill sentence + one
  `docs/tech-debt/README.md` row); cross-review cycle 1
  (`docs/reports/cross-review-triage-codex-effective-model-receipt.md`, both
  findings ACTION_REQUIRED), fixed in `d24a030` (AR-1: strip a trailing
  ` agent_start_retries=<digits>` suffix off the recovered prompt path; AR-2:
  count only receipts strictly newer than spawn start); self-review cycle 2
  (same report file, "Cycle 2" section), fixed in `53f6b16` (code) and `2af0bce`
  (docs) — C2-1 through C2-5, all in-cycle.

### Verdict

**PASS.** No AC is not-met. AC-11's test-execution half and `Closes #165` half
remain out of scope here. All requested static checks are green. No
documentation drift found. Two AC wordings are narrower than what the code now
does (both are the team lead's own flagged candidates, confirmed by reading the
diff) — noted below as informational, not as AC failures, since the code's
behavior is a superset/refinement of what the AC promises, not a violation of it.

### AC table

| AC | Status | Evidence |
|----|--------|----------|
| AC-1 | Met | Unchanged this cycle at the level AC-1 describes. `scanRolloutRecord`/`codexRolloutCandidates` (`internal/org/codex_session.go`) still degrade to `matched=false` on no-pointer-match, a pre-cutoff `session_meta`, a non-regular file, a malformed line, or an oversized line — none of the cycle-2 diff hunks touch this decision logic, only the *set of directories* fed into it (see AC wording note below). The ModTime pre-filter AC-1 refers to ("spawn 開始より前に更新された記録") is explicitly still anchored to `spawnStarted` only, not `until` (`codexRolloutCandidates`'s updated doc comment: "The ModTime pre-filter is anchored to spawnStarted only, not until"). No new panic path introduced — `codexSessionDateDirs`'s loop is a plain bounded `for` over `time.Time` values, no new decode path. |
| AC-2 | Met | Unchanged: `codexFoundReceipt` untouched this cycle (`git diff ccd3d2f..HEAD` touches no line of it). |
| AC-2b | Met | Unchanged: the `session_meta` cutoff check (`scanRolloutRecord`, `codex_session.go:356-360` region) still uses `spawnStarted.Truncate(time.Second)` regardless of how wide the date-directory window is — widening the window (`until`) cannot make an old, pre-spawn session qualify, since qualification is decided by the record's own `session_meta` timestamp, not by which directory it was found in. `TestObserveCodexEffectiveModel_OldSessionUpdatedLaterNotPicked`/`_TwoQualifyingRecordsAmbiguous` untouched; `TestCodexSpawnCorrelation_TwoRealSpawns_LatestWins` (new, `internal/org/verbs_test.go:1635`) exercises the "latest spawn_started wins" correlation rule at the Stop-lookup layer, a sibling guarantee. |
| AC-2c | Met | Unchanged: `observeCodexSpawnReceipt`'s empty-`promptPath` short-circuit untouched this cycle. |
| AC-3 | Met | Unchanged: `observeCodexSpawnReceipt`'s poll loop and its `until = spawnStartedAt` argument (`spawn.go:1093`, new this cycle) still bound Spawn's own window to the original three directories — the comment explains why: "the poll runs moments after the spawn … there is nothing later to reach". `TestOrgSpawn_Codex_ModelObservation_NotFoundAfterTimeout` untouched. |
| AC-4 | Met | Unchanged: the claude/dry-run branches in `Spawn`/`dryRunSpawn` are untouched by `git diff ccd3d2f..HEAD -- internal/org/spawn.go` outside the `ModelReceipt`-on-append-success change (C2-3, orthogonal to AC-4 — it *tightens* the "no receipt persisted" case, it does not change the claude/dry-run text). `TestOrgSpawn_Claude_ModelReceiptTextUnchanged` untouched. |
| AC-5 | Met, with an AC-wording note (flagged as requested) | Code: `observeStopModelReceipt` (`verbs.go`) still gates on `seat.Driver=="codex"`, not dry-run, a resolvable `codexSpawnCorrelation`, and `!hasObservedCodexReceiptAfter(...)` (renamed from `hasObservedCodexReceiptSince`, cross-review AR-2) — the comparison is now `r.TS <= spawnStartedTS` skips (strictly-after required), not `r.TS < spawnStartedTS` skips (at-or-after required) as at cycle 1. This means a receipt whose TS is the exact same second as `spawnStartedTS` no longer counts as "already observed" and Stop will attempt (one more, harmless per the design) observation in that narrow window. **AC wording**: AC-5's Japanese text says "spawn 開始**以降**に…observed receipt がないときだけ観測し" (以降 = "at or after", inclusive) — the code now implements "strictly after" (exclusive of the exact same second). The plan's own Scope row 4 already documents this exact deviation ("cross-review cycle 1 による改訂: 秒精度で同じ時刻の receipt は数えない"), and the direction is safety-conservative (worst case is one harmless duplicate receipt at stop, not a missed mismatch, per self-review's own point 2 and `hasObservedCodexReceiptAfter`'s doc comment) — not a regression against the AC's *intent*, only against its literal inclusive/exclusive wording. Recommend updating AC-5's prose to say "厳密に後" (strictly after) the next time the plan is touched; not a blocker. All ten edge-case states remain covered: `TestOrgStop_Codex_AppendsWhenSpawnReceiptWasUnknown/_HonoredFalse_ObservedAtStop/_DoesNotAppendWhenAlreadyObserved/_AppendsWhenNoReceiptAtAllForThisSpawn/_AppendsWhenOnlyRejectionReceiptExistsSince/_DryRunRespawnDoesNotDisplaceRealSpawnCorrelation/_NothingAppended_NotFound/_DryRun_NoObservationAttempted/_NoPromptFileSeat_NothingAppended/_ReceiptsAppendFailureLeavesStopSuccessful` (`verbs_test.go:1273-2047`) plus three new ones: `TestOrgStop_Codex_RecoversPromptPathAfterAgentStartRetry` (AR-1), `TestOrgStop_Codex_SameSecondRespawn_PreviousReceiptDoesNotSuppressObservation` (AR-2), `TestOrgStop_Codex_ObservesSessionThatStartedDaysAfterSpawn` (C2-1's `until` widening). |
| AC-6 | Met | Unchanged: `git diff ccd3d2f..HEAD -- internal/cli/org.go` is empty — `printCodexModelMismatchWarning`/`printSpawnResult` untouched this cycle. |
| AC-7 | Met | Unchanged: `git diff ccd3d2f..HEAD -- internal/cli/doctor_codex_models.go` is empty — `codexRetirementClause`/`checkCodexModelSlugs` untouched this cycle. |
| AC-8 | Met | Unchanged: `testOrg(t)` (`spawn_test.go`) and `TestMain` (`main_test.go`) still pin `CodexSessionsDir`/observe-timeout/observe-interval to nonexistent tmp paths and ms-scale durations; neither seam was touched by the `until` parameter addition (`until` is always caller-supplied — `spawnStartedAt` at Spawn, `o.nowTime()` at Stop — never independently resolved from the environment). `TMPDIR=/tmp` claim: code-read only, as in cycle 1, not an actual test run (`/test`'s scope). |
| AC-9 | Met | Evidence file's new appended lines (`docs/evidence/codex-effective-model-receipt-2026-09-20.md`) still contain only model/status/elapsed-time data plus a stated inference about *when* codex writes the record (see Documentation drift below) — no conversation content, no full paths beyond the already-documented shape. Re-read in full at HEAD; no regression from cycle 1's privacy property. |
| AC-10 | Met | See Documentation drift below — all updated surfaces match the code at HEAD; the three sync gates pass (see Static analysis). |
| AC-11 | Not applicable to `/verify` | Unchanged: test execution is `/test`'s scope, `Closes #165` is `/pr`'s scope. |

**AC-1 wording note** (the second candidate the hand-off named): AC-1's own clause
"spawn 開始より前に更新された記録…があっても…panic もしない" is about the ModTime
pre-filter, which cycle 2 did **not** widen (confirmed above — still anchored to
`spawnStarted` only). What cycle 2 widened is a different axis: which
*date directories* get walked in the first place (`codexSessionDateDirs`'s
`until` parameter, capped at `codexObserveMaxDateDirs = 32`). AC-1's text does
not claim a fixed three-directory window, so this widening does not contradict
AC-1 — it is silent on the window's reach entirely. No wording change needed;
noted only because the hand-off asked to check for it.

### Static analysis results

| Command | Result |
|---|---|
| `HARNESS_VERIFY_MODE=static ./scripts/run-static-verify.sh` | PASS (exit 0). Same shape as cycle 1: `check-sync.sh` (DRIFTED: 0), `check-pipeline-sync.sh` (ok), `check-skill-sync.sh` (13 skills in lock-step), `check-template-purity.sh` (PASS), golang pack verifier (`gofmt: ok`, `0 issues.`). |
| `gofmt -l internal/org internal/cli` | Clean (no output). |
| `go vet ./internal/org/... ./internal/cli/...` | Clean (no output). |
| `./scripts/check-skill-sync.sh` | PASS, run standalone. |
| `./scripts/check-sync.sh` | PASS, run standalone (`DRIFTED: 0`, `KNOWN_DIFF: 5` — same pre-existing, already-approved set as cycle 1). |
| `./scripts/check-template-purity.sh` | PASS, run standalone. |

Full command transcripts:
`docs/evidence/verify-2026-09-20-codex-effective-model-receipt-cycle2.log`
(gitignored per `docs/evidence/*.log`, not committed — same convention as
cycle 1's log).

### Documentation drift

None found. Every surface the hand-off named was re-read at HEAD and
cross-checked against the code:

- `.claude/rules/ralph/model-routing.md` + template copy: both gained the
  sentence "Stop also covers a session that started days after the spawn (a
  startup dialog answered late), up to about a month." — matches
  `codexObserveMaxDateDirs = 32` and the `until` widening exactly (32 days ≈
  "about a month"). Neither copy overclaims "always finds" — "covers … up to
  about a month" correctly states the bound (the cap, the ModTime pre-filter,
  and the 200-file cap all still apply underneath it).
- `docs/recipes/codex-seat-permissions.md` + template copy: gained "(codex
  appears to create the record only once the dialog is answered; stop searches
  from the spawn date to the stop date, up to about a month)" — the "appears
  to" hedge is the correct strength: the evidence file's own "確認していない
  こと" section states this is an inference from one real record's timing
  (`session_meta`→`turn_context` gap unchanged at ~2.5s whether or not the
  dialog appeared), not a confirmed fact. Does not present the inference as
  confirmed.
- `.claude/skills/org/SKILL.md` + 3 mirrors: all four byte-identical (diffed
  pairwise). Two hunks: (1) the same "spawn の数日後に始まった session も対象
  …探すのは spawn 日から約 1 か月分まで" sentence, matching the rule doc's
  English equivalent; (2) the doctor-info sentence changed from unconditionally
  naming both 移行先 and 退役日 to "読み取れた場合は退役日を示す" (shows the
  retirement date only if it could be read) — matches
  `parseCodexModelUpgrade`'s `hasDate=false` branch (`codexRetirementClause`,
  `internal/cli/doctor_codex_models.go`, unchanged this cycle). The base
  sentence was only added post-cycle-1-verify by the sync-docs commit
  (`27b6759`, confirmed via `git log -S`) without this hedge; self-review
  cycle 2's C2-6 minor observation caught the gap and `2af0bce` added it —
  so this was never reviewed unhedged in a prior verify cycle, just fixed
  before its first review.
- `docs/evidence/codex-effective-model-receipt-2026-09-20.md`: the new line
  under "26b03ce では…" describes the `53f6b16` re-run with `until` two days
  later, same five results — consistent with the code, which still reads no
  wall clock (both endpoints caller-supplied). The rewritten "確認していない
  こと" bullet states the dialog-timing read as "と読める(推定)" (reads as …,
  an inference) rather than a confirmed fact — does not overclaim. The quoted
  `ralph doctor` Detail line in the "表示" section is unchanged from cycle 1
  and `codexRetirementSentence`'s text is unchanged this cycle
  (`git diff ccd3d2f..HEAD -- internal/cli/doctor_codex_models.go` is empty),
  so the quote is still accurate.
- `docs/evidence/codex-seat-permissions-2026-09-18.md` P5: unchanged this cycle
  (`git diff ccd3d2f..HEAD` for this file is empty) — no new claim to check.
- `docs/specs/2026-08-01-org-runtime.md` FR-9: unchanged this cycle — the
  cycle-1 clause is still accurate (nothing cycle 2 changed contradicts it).
- `docs/tech-debt/README.md`: one new row (added by `27b6759`, after the
  cycle-1 verify commit, so not previously reviewed) — states the observer
  depends on codex's undocumented internal session-record shape, measured only
  on codex-cli 0.154.0 against five real records, and that a format change
  degrades silently to `honored=unknown` with no operator-visible signal. This
  matches the code (every failure mode in `codex_session.go` does degrade
  silently, by design) and the plan's own Risks table, which already named
  this risk and accepted "degrade to unknown" as the mitigation. Not
  duplicated (`grep -c` confirms exactly one row).

### Non-goals held (cycle-2 delta)

- `git diff ccd3d2f..HEAD -- internal/org/receipts.go internal/org/manifest.go` empty.
- `git diff ccd3d2f..HEAD -- ralph.toml templates/base/ralph.toml internal/config/` empty.
- `git diff ccd3d2f..HEAD -- internal/insights/` empty.
- `git diff ccd3d2f..HEAD -- internal/cli/org.go internal/cli/doctor_codex_models.go` empty — no pane reading was ever introduced, and none of this cycle's fixes touch the CLI warning or doctor logic.

### Privacy property (static reading only, cycle-2 delta)

Re-confirmed by reading the cycle-2 diff in full: `git diff ccd3d2f..HEAD --
internal/org internal/cli | grep 't\.Logf\|log\.Print\|fmt\.Print'` on added
lines returns nothing. The `until` parameter carries only a `time.Time`
instant (never record content) into `ObserveCodexEffectiveModel`; the
`promptPathFromAgentStartedDetails`/`isAllDigits` addition (AR-1) parses a
manifest Details string ralph itself wrote (`agent_started prompt_file=<path>[
agent_start_retries=<N>]`), never a codex session record — outside this
function's privacy boundary entirely. No new error return was added anywhere
in the cycle-2 diff; the single content-free error site from cycle 1
(`codex_session.go:315`'s `fmt.Errorf(...)`) is untouched.

### What remains unverified

- Same as cycle 1: `go test`/`go vet`-via-test-run, an actual `TMPDIR=/tmp go
  test` execution, `Closes #165`, end-to-end behavior against a live codex
  seat.
- The dialog-timing inference in the evidence file (record created only after
  the dialog is answered) is explicitly unconfirmed by the plan/evidence
  themselves, not just by this verify pass — stated as such in both places, so
  this is not a gap this cycle introduced.
- AC-5's wording (以降 vs strictly-after) — flagged above as an informational
  note for the plan's next edit, not re-verified as fixed since it is a
  documentation-precision item, not a behavior to test.

### Insight event

Appended via `./scripts/insights-append.sh --slug codex-effective-model-receipt --flow standard --phase verify --verdict pass --source skill --cycle 2`.
