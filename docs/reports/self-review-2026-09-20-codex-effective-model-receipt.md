# Self-review report: codex-effective-model-receipt

- Date: 2026-09-20
- Plan: `docs/plans/active/2026-09-20-codex-effective-model-receipt.md` (issue #165)
- Reviewer: `reviewer` subagent (Claude Code, opus), standard flow, pipeline cycle 1 of cap 2
- Scope: diff quality only for `git diff main...HEAD` on `feat/codex-effective-model-receipt` (base `642214d`, HEAD `825e925`, working tree clean). No tests, static analysis, spec-compliance, or doc-drift checks were run — those belong to `/verify` and `/test`.

## Verdict

MERGE with fixes. No CRITICAL and no HIGH findings.

The privacy contract the plan set — "取り出すのは model だけ" — holds on every
path I could trace: the observer's single non-nil error carries only an
operation verb and a file path, both production callers discard it with `_`,
and no receipt, manifest event, stderr line, or test log ever touches a decoded
body. The identity contract holds for the cases the plan named (respawn of the
same seat, a stale record touched later, an ambiguous pair), and the
`<org>_<seat>.md` prompt-path convention is unambiguous because
`identifierPattern` (`internal/org/identifier.go:38`) forbids `_` in both
halves.

Four MEDIUM findings are local: three are comments/doc-strings that assert more
than the code does (one of them says the file is not wired in, which its own PR
falsified two commits later), one is a manifest reader that does not filter
dry-run events. Seven LOW findings are naming, an unbounded date-directory
loop, a reason string reused for a cause it did not observe, and doc sentences
that promise an unconditional append.

## Finding counts by severity

| Severity | Count |
|----------|-------|
| CRITICAL | 0 |
| HIGH     | 0 |
| MEDIUM   | 4 |
| LOW      | 7 |

## Evidence reviewed

- `git diff main...HEAD` (23 files, +3211/-40) and `git log --format='%h parent=%p'` for the 9 commits (linear DAG; every report/fix commit is the direct child of the one before it, so no commit in this delta escaped the stated review range).
- Full reads: `internal/org/codex_session.go`, the `Spawn`/`Stop` wiring hunks in `internal/org/spawn.go` and `internal/org/verbs.go`, `internal/cli/org.go`, `internal/cli/doctor_codex_models.go`.
- Targeted reads of the pre-existing code the change now depends on: `o.reject` (`spawn.go:1279`), `dryRunSpawn` (`spawn.go:1299`), `promptFilePath`/`promptFilePointer` (`spawn.go:1239`/`:1272`), `waitBeforeEnter` (`verbs.go:425`), `findSeat` (`verbs.go:56`), `Roster`'s dry-run key (`manifest.go:137`,`:180`), `identifierPattern` (`identifier.go:38`), `Org.now`/`nowTime` (`spawn.go:185`, `watch.go:319`).
- New tests read in full for the timing-sensitive and privacy cases; mirror hunks for the four `/org` SKILL.md faces compared line-for-line (identical).
- No `go build`, `go vet`, `go test`, or sync-gate run — out of scope for this phase.

## Findings

| # | Severity | Area | Finding | Evidence | Recommendation |
|---|----------|------|---------|----------|----------------|
| M1 | MEDIUM | comments / history-in-code | The new file's header says it is not wired in, which is false at HEAD; two more comments describe the wiring in the future tense. The header also pins a `docs/plans/active/` path that `/pr` archives. | `internal/org/codex_session.go:18-21` "This file is the observer only -- nothing here reads the process environment, writes anything, or is wired into Spawn/Stop **yet** (that wiring is a later slice)" — `d2bf6e4` wired it into both. Same tense at `:129-130` ("must never fail the caller (Spawn/Stop, in the slice that wires this in)") and `:244-245` ("e.g. from Stop, in the slice that wires this in"). | Rewrite all three in the present tense describing the code ("Spawn polls this; Stop calls it once"). Drop the `Plan:` path or replace it with the issue number, which survives archival. |
| M2 | MEDIUM | manifest reading | `codexSpawnCorrelation` scans raw events with no `DryRun` filter, so a `--dry-run` spawn of an already-spawned seat becomes "the latest spawn_started" and displaces the real one. Stop then correlates against the dry-run timestamp and silently observes nothing. | `internal/org/verbs.go:702-727` filters only on `OrgID`/`SeatID`/`Event`. `spawn.go:350`'s `if p.DryRun` branch precedes the idempotent check, so a dry-run trail for an already-spawned seat is reachable; `dryRunSpawn` appends `spawn_started` and an `agent_started prompt_file=` step with `DryRun: true` (`spawn.go:1301-1350`). `manifest.go:137` states the package invariant: "a dry-run event for a seat_id never becomes (or clears) a real seat's state". | Skip `ev.DryRun` events in both loops in `codexSpawnCorrelation`. One-line change; add the dry-run-then-stop case to `verbs_test.go`. (`latestSeatEventTS` does not filter either, but it takes a max over all events rather than letting the last one win, so it does not have this displacement.) |
| M3 | MEDIUM | public contract | `SpawnResult.ModelReceipt`'s doc says the zero Receipt means no receipt was appended, and that this is "every outcome except SpawnOutcomeSpawned". Both halves are wrong: `reject()` appends an honored-false receipt, and `dryRunSpawn` returns `SpawnOutcomeSpawned` *with* an appended receipt and a zero `ModelReceipt`. | `internal/org/spawn.go:257-261` vs `reject` (`:1285-1288`) and `dryRunSpawn` (`:1402-1406`, returning `SpawnOutcomeSpawned` at `:1410`). The PR's own `TestOrgSpawn_Codex_DryRun_ModelReceiptUnchanged` (`spawn_test.go`) reads the receipts store instead of `result.ModelReceipt` — the tell. | Reword to what the field actually is: "the model receipt built by the real (non-dry-run) spawn path; zero on every other path, including a dry-run spawn and an envelope rejection, both of which append their own receipt that is not surfaced here." Same sentence applies to `StopResult.ModelReceipt` (`verbs.go:594-598`), which is accurate but would read better in the same terms. |
| M4 | MEDIUM | operator-facing string | `ralph doctor`'s fixed retirement sentence promises unconditionally that ralph records a migration as honored-false, while the feature records `unknown` for every seat it cannot observe (claude seats, inline/empty prompts, a herdr server on a different `CODEX_HOME`, no turn started). | `internal/cli/doctor_codex_models.go:199`: "…; ralph records that as honored=false in the org model receipts." The rule doc hedges correctly (`.claude/rules/ralph/model-routing.md`, "A seat whose record cannot be identified stays `unknown`"); the doctor string does not, and an operator reading it will read `unknown` as "no migration happened". | Hedge the clause to match the rule doc: "…; ralph records that as honored=false **when it can read the seat's codex session record**, and unknown otherwise." |
| L1 | LOW | naming / doc drift | The observe poll reuses `waitBeforeEnter`, whose name and doc describe the send path's Enter keystroke. Neither was updated for the second caller. | `internal/org/spawn.go:1076` calls it; `internal/org/verbs.go:421-424` still reads "Split out of **Send** so the ctx-vs-timer race (the mechanism that lets `--timeout-ms` bound this wait too) is readable". `.claude/rules/ralph/architecture.md`: "Prefer grep-able names over clever names." | Rename to something caller-neutral (`sleepOrCtxDone`) and generalise the doc, or keep the name and add the second caller to it. |
| L2 | LOW | clock mixing / unbounded loop | `codexSessionDateDirs` derives its lower bound from `spawnStarted` (the injectable `o.nowTime()` clock) and its upper bound from wall-clock `time.Now()`, then walks one directory per day with no cap — once per poll iteration. A fixed `Org.Now` makes the loop length proportional to the clock skew. | `internal/org/codex_session.go:246-255`; `spawnStartedAt` comes from `o.nowTime()` (`spawn.go:875`, `watch.go:319`). `Org.Now` is a live seam: `report_test.go:138` pins it to `2026-08-02T12:00:00Z`. No current test both pins the clock and spawns a codex seat, so nothing is flaky today. | Add a `codexObserveMaxDateDirs` cap (the same shape as `codexObserveMaxFiles`) or derive `endDay` from the caller's clock. Cheap insurance against a future fixed-clock test spawning a codex seat. |
| L3 | LOW | error handling | `ObserveCodexEffectiveModel`'s error return is discarded by both production callers, so an unreadable sessions directory is indistinguishable from "no record yet" — and the reason text then asserts a cause that was not observed. | `internal/org/spawn.go:1060` and `internal/org/verbs.go:781` both use `obs, _ :=`. The doc at `codex_session.go:122-125` says the error is "reserved for a specific, content-free signal a caller **might want to log**" — nothing does. `.claude/rules/ralph/golang.md`: "Keep error handling explicit; do not silently swallow errors." | Either drop the error from the signature (the callers' behaviour is unchanged) or have the callers fold it into the unknown receipt's reason as a bare category (e.g. "a candidate record could not be opened"), which stays content-free. Document whichever is chosen at the definition site. |
| L4 | LOW | reason text | The ctx-cancellation path reuses `codexNotFoundReason`, which names two causes neither of which happened; the real cause is the spawn's own `--timeout-ms` expiring mid-poll. | `internal/org/spawn.go:1076-1078` returns `codexUnknownReceipt(base, codexNotFoundReason)` on `waitBeforeEnter` error; the constant at `:1086` reads "no codex session record for this spawn yet (no turn started, or CODEX_HOME differs from the seat's)". The plan's own design decision is "理由の文言は観測した事実だけを書く". | Use a distinct reason for the cancellation arm, e.g. "spawn timed out before a codex session record appeared". |
| L5 | LOW | docs overstate the code | Three shipped surfaces say `ralph org stop` appends the observed receipt, without the "if one is found" condition. Stop appends nothing on not-found, on ambiguous, on a missing prompt file, and on an already-observed receipt. | `.claude/rules/ralph/model-routing.md` + `templates/base/…/model-routing.md`: "looks once more for a seat that has no observed receipt since its spawn **and appends one**". `docs/recipes/codex-seat-permissions.md:120` + template copy: "`ralph org stop` looks once more **and appends the observed one**". `.claude/skills/org/SKILL.md` + 3 mirrors: "stop 時にもう一度だけ探して receipt を 1 件追記する". Actual behaviour: `verbs.go:658-666`, `:826-829`. | Add the condition ("…and appends one if it finds a record"). Six files; the four skill faces must move together (`scripts/sync-skills.sh`). |
| L6 | LOW | stale doc comment | `codexCacheFreshnessClause`'s doc still enumerates two callers' statuses; the clause now also lands on the new `info` Detail. | `internal/cli/doctor_codex_models.go:115-116` "the suffix appended to the slug check's **warn and pass** Detail" vs `:375`, which builds the Detail for both `info` and `pass`, and `:362` for `warn`. `checkCodexModelSlugs`'s own header at `:284` already says "warn/info/pass" — the two comments in one file now disagree. | Change "warn and pass" to "warn, info, and pass". |
| L7 | LOW | identity matching (defence in depth) | The seat/record correlation accepts the bare prompt path as a substring of *any* user message, so any codex session under the same `CODEX_HOME` that started after the spawn and happens to quote that path qualifies. At stop time the window is unbounded (any record from spawn-day−1 to today+1). | `internal/org/codex_session.go:398-418` uses `strings.Contains(part.Text, promptPath)`; `ralph` itself writes a fuller marker, `promptFilePointer` = `"役割指示を読み込んで従ってください: " + path` (`spawn.go:1272-1274`). Impact is bounded: when the seat's own record also qualifies the result is ambiguous and nothing is appended, so the harmful case needs the seat's own record to be absent *and* a third-party session quoting the path. | Match on `promptFilePointer(promptPath)` rather than the bare path. **Do not apply blind**: `docs/evidence/codex-effective-model-receipt-2026-09-20.md` only establishes that the *path* appears in the body, not that the full pointer sentence survives verbatim — re-run the observer against the same five real records before adopting, or leave as-is and record the residual risk. Requiring the *first* user message instead would be wrong: the same evidence (line 19) shows environment/AGENTS.md user messages precede it. |

## Answers to the ten points raised in the hand-off

1. **Privacy — is there any path from record content to an output?** No path found.
   The only non-nil error is built at `codex_session.go:304` as
   `fmt.Errorf("org: open codex session record %s: %w", path, openErr)` — an
   operation verb, the rollout file's path, and the `os.PathError`. Line content
   is never in it, and both production callers discard it (`spawn.go:1060`,
   `verbs.go:781`), so it reaches no receipt, no manifest event, and no stderr.
   Decoded bodies exist only as locals inside `codexResponseItemMentionsPrompt`
   and are never returned. `CodexModelObservation` carries two fields, both
   model/status. No new test calls `t.Logf` (0 matches in the diff), and
   `TestObserveCodexEffectiveModel_PermissionDeniedDoesNotLeakContent`
   (`codex_session_test.go:546`) pins the contract with a sentinel *and* skips
   under uid 0, so root does not silently defeat the chmod. The residual issue
   is not a leak but L3: nobody can tell that the open failed at all.
2. **Identity — can another session's model be attributed to this spawn?** Three
   sub-cases:
   - *Respawn while the old process still writes*: excluded correctly, by
     `session_meta.timestamp` rather than ModTime — `codex_session.go:345-348`
     bails as soon as the meta line disqualifies, and
     `TestOrgSpawn_Codex_OldSessionRecordNotPickedOnRespawn` proves it through
     `Spawn` by chmod-ing the stale file's ModTime into the future.
   - *Two seats of different orgs sharing a state dir*: not reachable.
     `promptFilePath` is `<state-dir>/prompts/<org>_<seat>.md` (`spawn.go:1244`)
     and `identifierPattern` is `^[a-z][a-z0-9-]{0,29}$` (`identifier.go:38`),
     so `_` cannot occur inside either half and the split is unambiguous.
   - *A user message that merely quotes the path*: reachable, see L7. Worst case
     is a wrong `reported_effective_model` on one receipt, not a leak or a
     failed spawn, and it needs the seat's own record to be missing. Smallest
     fix is the pointer-sentence match in L7, gated on re-running the five real
     records.
3. **Lexicographic TS comparison in `hasObservedCodexReceiptSince`.** Safe today.
   Every writer goes through `o.now()` (`spawn.go:185-191`), which is
   `nowFn().UTC().Format(time.RFC3339)` — fixed width, always `Z`, no fractional
   part, no offset — and the compared value comes from the same formatter via
   `checkCapacityAndStart` (`spawn.go:884`). Byte order equals chronological
   order for that set. `latestSeatEventTS` (`watch.go:334`) already relies on the
   same property, and the new comment cites it. I would not change it: parsing
   would add an error path (what does an unparsable receipt TS mean?) for no
   behavioural gain. Non-finding.
4. **Spawn latency and locking.** The poll holds no lock. Both `withManifestLock`
   closures have returned by then (`spawn.go:557`, `:625`), and the surrounding
   appends take the flock per call. ctx cancellation does end it: each wait goes
   through `waitBeforeEnter(ctx, …)` (`spawn.go:1076`), which selects on
   `ctx.Done()`, and each wait is clamped to the remaining budget, so it cannot
   overshoot the deadline. The `spawned` event is appended *before* the poll and
   the receipt *after*, so a ctx expiry never strands the side effect — the
   unknown receipt is still appended (only its reason text is wrong, L4). 8 s
   against a measured 2.3–3.2 s is a reasonable default: it is ~2.5x the slowest
   observed case, it only applies to real codex seats with a prompt file, and it
   returns as soon as the record appears. Non-finding.
5. **Wall clock vs injectable clock.** Two uses of `time.Now()`: the poll
   deadline (`spawn.go:1059`) and the date-directory upper bound
   (`codex_session.go:248`). The deadline is right to use the real clock — it
   measures elapsed real time, and a fixed clock would make it never expire. The
   date bound is the one worth fixing (L2). No midnight hazard: the window is
   already padded one day on each side and the endpoint is "today+1", so a file
   created just across a boundary is still covered. No current flakiness — no
   test sets `Org.Now` and spawns a codex seat.
6. **Stop ordering.** The observation cannot delay or alter the stop. It runs
   after both driver calls and before the `stopped` append, it never waits (one
   call, no retry — `verbs.go:781`), a receipts-append failure leaves the token
   at `none` rather than failing the stop (`verbs.go:657-663`), and every error
   path returns `observed=false`. The two extra whole-file reads (manifest via
   `observeStopModelReceipt`, on top of `findSeat`'s own read at `verbs.go:57`;
   plus the receipts file) are acceptable for a per-stop operation on
   append-only JSONL. Worth knowing rather than fixing. Non-finding beyond M2.
7. **Doctor wording.** The clause itself (`; retiring: a -> b on YYYY-MM-DD`) is
   accurate and the info-vs-warn split is right: nothing is missing, so warn
   would overstate, and `pass` would hide it. The fixed sentence is the part
   that claims more than ralph does — see M4. One more detail in its favour: the
   `Seven deterministic outcomes` header at `:267` does match seven bullets at
   HEAD, so the "additive fix to a closed list" trap was avoided here.
8. **Test quality.** Fixtures match the real-record shape documented in the
   evidence file (`session_meta` first, `turn_context` before the prompt-path
   user message, millisecond RFC3339). Nothing looked flaky:
   - `TestOrgSpawn_Codex_ModelObservation_FoundOnLaterPoll` writes at 30 ms into
     a 2 s window at 10 ms intervals, and the goroutine correctly avoids
     `t.Fatalf`, reporting through a channel the test drains.
   - `TestOrgSpawn_Codex_InlinePromptSkipsObservation_NoWaiting` asserts elapsed
     `< 2 s` for a path that does zero I/O — a very wide margin.
   - The CLI fixture's `now + 1 s` is not brittle in the shape I worried about:
     the cutoff is `spawn_started`, which `checkCapacityAndStart` appends inside
     Phase 1 before any stub-process round trip, so the gap between the fixture
     write and the cutoff is a few milliseconds, not the whole saga. The stop
     tests are looser still (cutoff is the earlier spawn's timestamp).
   - Doctor tests use ±10 years, so no date-boundary flake.
9. **Comment volume and history-in-comments.** Volume is high but mostly
   load-bearing (the constants' doc comments carry the real-record measurements
   that justify them, which is the right place for them). The history problem is
   real in exactly three places — M1. `AC-N` citations are an established
   convention in this package (`spawn.go` already carries AC-2b/AC-3/AC-8 from
   earlier PRs), so I am not flagging those; the plan-path pointer in M1 is
   different because `/pr` archives the file.
10. **Docs that say more than the code.** L5 (stop "appends one") on six files,
    and M4 (doctor's honored-false promise). Everything else checked out: the
    spec FR-9 addition, the rule paragraph's `unknown` enumeration, and the
    evidence file — which is unusually careful, listing only key names, line
    numbers, times and models, redacting `$HOME` to `~`, and carrying a
    "確認していないこと" section that correctly declines to claim the
    retirement-dialog case was verified.

## Positive notes

- The privacy design survives contact with the code, not just the plan: the
  error string is content-free, the sentinel test proves it, and the test skips
  under root instead of passing vacuously.
- `AC-2b` is enforced at the right layer — the record's own `session_meta`
  timestamp decides, and the ModTime check is explicitly documented as a
  read-volume pre-filter that "never decides which record belongs to which seat"
  (`codex_session.go:48-54`). The respawn test proves the distinction by giving
  the stale file a future ModTime.
- Ambiguity is a first-class outcome rather than a coin flip
  (`CodexObservationAmbiguous`), and the poll stops immediately on it
  (`spawn.go:1064-1065`) instead of burning the rest of the budget.
- The envelope-rejection false positive was anticipated and has its own negative
  test (`TestOrgSpawn_CLI_EnvelopeRejection_NoModelWarning`): the CLI gate is
  `Honored == false && ReportedEffectiveModel != ""`, not `Honored == false`.
- `codexPromptFileDetailsPrefix` replaced two independent `fmt.Sprintf` literals
  with one shared constant across the write side (`spawn.go:722`, `:1349`) and
  the new read side (`verbs.go:723`) — the drift this PR could have introduced
  was designed out.
- The `json.RawMessage` decision for `upgrade` is correct and well argued: one
  odd entry cannot hide every slug, and there is a test for the wrong-typed case.
- `testOrg` (`spawn_test.go`) and `TestMain` (`internal/cli/main_test.go`) pin
  the sessions directory package-wide, so the AC-8 "never read a developer's
  real `~/.codex`" property holds for tests that were never touched by this PR,
  not only for the new ones.
- All four `/org` SKILL.md faces received byte-identical hunks (verified by
  diffing the added lines of each mirror against `.claude/skills/org/SKILL.md`).
- No debug statements, no commented-out code, no TODO/FIXME markers, and no
  credential-shaped strings in the added lines.

## Coverage gaps (things this review did not establish)

- Whether the code compiles, vets, formats, or passes its tests — not run, by
  design. `/verify` and `/test` own that.
- Whether the four skill mirrors and the two template copies pass
  `check-skill-sync.sh` / `check-sync.sh` / `check-template-purity.sh`.
- Whether the new doc sentences satisfy spec FR-9 — `/verify`'s scope.
- L7's recommended tightening is unverified against real records; I read the
  evidence file rather than any real session record, per the hand-off constraint.
- The end-to-end behaviour against a live codex seat, which the evidence file
  itself lists as unverified.

## Tech debt identified

None recommended for deferral. All eleven findings are in-cycle fixes: M1/M3/M4/L1/L4/L6
are single-sentence edits, M2 and L2 are one-line code changes, L5 is a
mechanical sweep of six files, and L3/L7 are small decisions that should be
made now rather than tracked. If any LOW is consciously left unfixed at the
cap, it should be batched into one `docs/tech-debt/README.md` row before `/pr`,
since the plan is archived at that point and nothing else would carry it.

## Recommendation

- Merge: **yes, after the MEDIUM fixes.** No CRITICAL or HIGH findings, so this
  does not block the pipeline; M1–M4 are cheap enough that deferring them costs
  more than fixing them.
- Follow-ups:
  1. Fix M1 (three comments + the plan pointer), M2 (dry-run filter + one test),
     M3 and M4 (two doc-string rewrites).
  2. Decide L3 and L7 explicitly — both are "keep it or tighten it", and both
     deserve a sentence in the plan's Deviation notes either way.
  3. Sweep L5 across the two rule copies, the two recipe copies, and the four
     skill faces together.

## Cycle 2 (2026-09-20)

- Reviewer: `reviewer` subagent (Claude Code, opus), standard flow, pipeline cycle 2 of cap 2 — the final cycle, so every finding below is either fixed in this cycle or a deferral.
- Scope: `git diff d73dbdd..HEAD` (the cycle-1 fix commits, the two cross-review AR fixes, and the pipeline artifacts), with `git diff main...HEAD` as context. HEAD `79d7c16`, working tree clean. Diff quality only; no tests, static analysis, spec-compliance, or doc-drift checks were run.
- DAG checked with `git log --format='%h parent=%p'`: linear, 10 commits, no commit in the delta falls outside a review's stated range.

### Verdict

MERGE. No CRITICAL and no HIGH. One MEDIUM (a coverage narrowing the cycle-1 L2 fix introduced, whose cheapest honest fix is two sentences of documentation) and five LOW.

All eleven cycle-1 findings are genuinely fixed — not one is half-fixed, and none was deferred. Two were over-delivered: L2 removed the wall-clock read from the observer entirely and backed the new bound with real file-metadata evidence, and L7 landed with a negative test, a positive test, a wrapped-text test, and a re-run against the five real records. The two cross-review AR fixes are both anchored by integration tests that go red on the pre-fix code, and the implementer's unprompted repair of `TestOrgStop_Codex_DoesNotAppendWhenAlreadyObserved` fixed a test that genuinely could not distinguish "correctly skipped" from "re-observed but ambiguous".

### Finding counts by severity

| Severity | Count |
|----------|-------|
| CRITICAL | 0 |
| HIGH     | 0 |
| MEDIUM   | 1 |
| LOW      | 5 |

### Cycle-1 findings: disposition

| Cycle-1 ID | Status | Evidence at HEAD |
|---|---|---|
| M1 (history in comments) | Fixed | `grep -c "that wires\|later slice\|docs/plans" internal/org/codex_session.go` → 0. All three sites rewritten in the present tense; the header now points at issue #165 and the evidence file, both of which survive plan archival. |
| M2 (dry-run filter) | Fixed, both loops | `verbs.go:713` and `:737` each lead with `ev.DryRun ||`; the doc comment states the `manifest.go` invariant it restores. Regression test `TestOrgStop_Codex_DryRunRespawnDoesNotDisplaceRealSpawnCorrelation` uses a monotonic fake clock and was confirmed red on the old code by the cycle-1 `/test` mutation run. |
| M3 (`ModelReceipt` doc) | Fixed, and the behaviour was aligned rather than the doc weakened | `reject()` (`spawn.go:1330-1341`) and `dryRunSpawn` (`spawn.go:1454-1465`) now both set the field; the doc enumerates the three paths that set it and the three that leave it zero. Residual: C2-3 below. |
| M4 (doctor sentence) | Fixed | `doctor_codex_models.go:207` now hedges with "when ralph can identify the seat's codex session record". The doctor tests reference the constant rather than a copy of its text, so the change required no test churn and no change-detector was created. The evidence file's captured output was re-taken. |
| L1 (`waitBeforeEnter`) | Fixed | Renamed `waitOrCtxDone`; its doc now names both callers and both delay sources. Zero occurrences of the old name remain in any `.go` file; the only hits are in prior-PR reports, which correctly describe the state they reviewed. |
| L2 (date window) | Fixed, over-delivered | `codexSessionDateDirs` returns exactly three directories and no longer reads the wall clock — so the observer as a whole now reads no clock at all. Backed by real file-metadata evidence (1,013 records, 57 modified on a later day, none moved) and two tests covering both directions. Introduced C2-1 below. |
| L3 (discarded observer error) | Fixed | `codexReadErrorReason` (`spawn.go:1105`); `lastErr` is threaded through the poll and consulted at the timeout branch. Still content-free — the constant names a category, never the error text or a path. |
| L4 (ctx reason) | Fixed | `codexCutShortReason` (`spawn.go:1106`), returned only from the `waitOrCtxDone` error arm. |
| L5 (docs "appends one") | Fixed, all eight surfaces | Both rule copies, both recipe copies, and all four `/org` SKILL.md faces now say "when it finds the record" / "記録が見つかれば". The two rule copies wrap the sentence differently because the template copy deliberately omits the `internal/org/` source path — an intended difference, not drift. |
| L6 (freshness doc) | Fixed | `doctor_codex_models.go:115-116` now reads "warn, info, and pass". |
| L7 (bare-path match) | Fixed, with the caveat I asked for honoured | `codexResponseItemMentionsPrompt` matches `promptFilePointer(promptPath)`. Crucially the fixture sweep was complete: of nineteen `userMessageLine` call sites in `codex_session_test.go`, every one that is meant to qualify now passes the pointer sentence, and only the two deliberate negatives (`:145` unrelated text, `:198` bare-path quote) do not — so no pre-existing "not-found" test became vacuous behind the tightened matcher. The observer was re-run against the five real records after the change. |

No cycle-1 finding reached `docs/tech-debt/README.md`, and none needed to. The one row this PR did add (`docs/tech-debt/README.md:134`, codex record-shape fragility) came from `/sync-docs` and is about a risk the plan already named, not about a deferred finding.

### Findings

| # | Severity | Area | Finding | Evidence | Recommendation |
|---|----------|------|---------|----------|----------------|
| C2-1 | MEDIUM | observation coverage / doc accuracy | Narrowing the walk to three directories around **spawnStarted's** date means Stop's second-chance observation can only ever find a session that *started* within a day of the spawn. The old window (`spawnStarted−1d … today+1d`) covered a session that started arbitrarily later. The one scenario where that gap is real is the one the shipped recipe promises works: a seat parked on the model-retirement dialog, answered more than a day later — and whether codex writes the rollout file at launch or only when the first turn begins is recorded as **unverified** in the plan's own Open questions. | `codex_session.go:246-255` keys the window solely to `spawnStarted`; both callers pass the spawn's own start time (`spawn.go:1080`, `verbs.go:857`), so a seat spawned Monday and stopped Friday only ever reads Sun/Mon/Tue. `TestObserveCodexEffectiveModel_DirectoryThreeDaysAfterSpawnDateNotWalked` pins the narrowing as the contract. `docs/recipes/codex-seat-permissions.md:119-123` (and the template copy) says "While the dialog is open no turn has started, so the spawn receipt stays `unknown`; `ralph org stop` looks once more and, when it finds the seat's session record, appends the observed one." Plan Open questions: "codex の退役ダイアログが出ている間に session 記録が作られるかどうかは未確認". | Cheapest and safest at the final cycle: **document the bound** rather than change the walk. One sentence in `codexSessionDateDirs`'s doc ("a session that starts more than a day after the spawn is out of range — the second-chance observation does not cover it") and one clause in both recipe copies. If a code fix is preferred instead, keep the function clock-free by passing the observation instant as a second parameter (`Stop` already has `o.nowTime()`) and walking `spawnStarted−1d … observedAt+1d` with a day cap — that restores the old reach *and* keeps the unbounded-loop fix that motivated L2. Do not reintroduce a bare `time.Now()` read inside the observer. |
| C2-2 | LOW | latent re-break of AR-1 | The Details string is now parsed by stripping one trailing suffix, but nothing states that the suffix must stay last. Appending a second space-separated field after it — the obvious next edit at that line — makes `digits` non-numeric, so the parser returns the retry suffix as part of the path and AR-1 silently returns, in exactly the same shape. | `spawn.go:744` appends the suffix with no ordering note; `codexAgentStartRetriesDetailsSuffixKey`'s doc (`spawn.go:1245-1255`) says "trailing" but never states the invariant. `promptPathFromAgentStartedDetails` (`verbs.go:760-773`) requires the digits to run to end-of-string. | One sentence at the append site and in the constant's doc: nothing may be appended to `agentStartedDetails` after this suffix; a new field goes *before* `prompt_file=` or the parser must be extended. (A structurally stronger option, if anyone touches the format later: emit the retry count *before* `prompt_file=`, so the path is always "everything to end of string" and further fields cost nothing. That changes the persisted Details layout, which is why I am not recommending it in this cycle.) |
| C2-3 | LOW | contract honesty, diverging from its own sibling | `reject()` still discards the receipts-append error and reports `ModelReceipt` regardless, so on an append failure the field names a receipt that was never persisted — contradicting the field's own newly written doc ("the zero Receipt on every path that appends no receipt at all"). `Stop`, in this same PR, deliberately guards exactly this case and says why. | `spawn.go:1338-1341`: `_ = o.Receipts.Append(receipt)` then `ModelReceipt: receipt`. Versus `verbs.go:658-666`: `if appendErr := o.Receipts.Append(receipt); appendErr == nil { modelReceipt = receipt … }` with the comment "nothing was actually persisted, so Stop must not claim otherwise". `dryRunSpawn` is also honest (it returns `SpawnOutcomeFailed` on an append error). Harmless today: the CLI gate needs a non-empty `ReportedEffectiveModel`, which a rejection never carries. | Capture the error and leave `ModelReceipt` zero when it is non-nil, matching Stop — three lines, and it makes the field's doc true on every path. Softening the doc instead would be the weaker choice, since two of the three appenders already have the stronger property. |
| C2-4 | LOW | cross-package literal duplication + a dangling ID | The CLI fixture hardcodes a byte-for-byte copy of `promptFilePointer`'s Japanese sentence because that function is unexported, with only a comment holding the two in sync — and that comment cites `F9`, an identifier that appears exactly once in the whole repository. | `internal/cli/org_test.go:1103` (the literal) and `:1098` ("The observer (F9/L7 fix) …"); `grep -rn "F9"` over the repo returns that single line. `promptFilePointer` is unexported (`spawn.go:1286`), while the same package already exports `ObserveCodexEffectiveModel` and `CodexSessionsDir` for cross-package use. Drift is not silent — the CLI tests would fail — but the failure message would be "expected the model-mismatch warning in output", which does not name the cause. | Export it as `PromptFilePointer` and have the CLI fixture call it; the exported-observer precedent in the same file makes that the consistent choice. Drop `F9` (or replace it with "cycle-1 self-review L7"). |
| C2-5 | LOW | ID pointers into an in-place-extended report | Two test comments cite bare `L7`/`L2` into `docs/reports/self-review-2026-09-20-<slug>.md` — the file this very commit adds a second cycle to, with its own `C2-n` numbering. The sibling convention in the same package cites the cycle *and* the report path, precisely because a bare ID into a per-slug report has been ambiguous before. | `internal/org/codex_session_test.go:183` ("the self-review L7 fix") and `:590` ("the self-review L2 fix"). Compare `internal/org/verbs_test.go:721` ("self-review revalidation LOW-3 fix (docs/reports/self-review-2026-09-19-org-send-enter-timing.md, …)") and `:857` ("renamed in cycle-2 (C2-3, …)"). | Add the cycle and the path: "cycle-1 self-review L7 (docs/reports/self-review-2026-09-20-codex-effective-model-receipt.md)". Two comment edits. Worth doing before `/cross-review` runs, after which the numbering space grows again. |
| C2-6 | LOW | same-PR reports stale against HEAD | Three pipeline reports in this PR name `hasObservedCodexReceiptSince`, which no longer exists at HEAD, and one of them describes the old inclusive semantics as the verified behaviour. Separately, the cycle-1 test report's self-declared "most substantive gap I found in this whole pass" was closed by `771e446` — the direct child of the commit that wrote it — so the merged PR would otherwise ship a report asserting an open gap that is covered. | `docs/reports/verify-2026-09-20-<slug>.md:28` (AC-5 row, old name and "no receipt since that spawn"); `docs/reports/test-2026-09-20-<slug>.md:52, 67, 91, 104`; `docs/reports/sync-docs-2026-09-20-<slug>.md:18`. The closing test is `TestHasObservedCodexReceiptAfter_OnlyThisSpawnsObservedReceiptCounts`'s case "observed receipt from an earlier spawn of the same seat" (`verbs_test.go:1601`), which is exactly the branch the test report called uncovered. `git log --format='%h parent=%p'` confirms `771e446`'s parent is `a7180ca`, the test-report commit. | No code change. This cycle's `/verify`, `/test`, and `/sync-docs` passes rewrite these three files — they must re-derive those rows against HEAD rather than carrying them forward, and the tester should not re-file the closed gap. Flagging it here so it is not re-discovered as new. |

### Answers to the seven points raised in the hand-off

1. **`promptPathFromAgentStartedDetails` anchoring.** `LastIndex` plus an all-digits tail is the right rule for the format as it stands, and the "path that itself ends in ` agent_start_retries=7`" case is **not reachable**: `promptFilePath` builds every path as `filepath.Join(stateDir, "prompts", fmt.Sprintf("%s_%s.md", orgID, seatID))` (`spawn.go:1253`), so a real prompt path always ends in `.md`, and `.md` fails `isAllDigits`. A state dir containing the literal text mid-path is handled — `LastIndex` plus the digits test — and the table test covers both halves of that case explicitly. The writer surface is genuinely narrow: `agentStartedDetails` is assembled at exactly two lines (`spawn.go:730` prefix+path, `spawn.go:744` optional suffix) and consumed at one (`:748`); no other `EventSpawnStep` Details in the file starts with the prefix (`tab_created`, `agmsg_lead_joined …`, `agmsg_announced`, `agmsg_joined`). The exposure is future growth, not today's code — that is C2-2. On **recomputing from the state dir instead**: I would not. `o.promptFilePath(orgID, seatID)` is deterministic and available at Stop, but reading it back from the manifest is the repo's own stated preference for exactly this class of value — `spawn.go:813-819` persists `herdr_agent_name` rather than deriving it, citing the tech-debt row "derived at every call site … instead of being persisted", so that "a future change to the naming convention now orphans no existing seat". The prompt path has the same property. A defensible hybrid, if C2-2's comment is felt to be too thin a guard, is to use Details only as the boolean "this spawn wrote a prompt file" and `promptFilePath` for the value — but that reintroduces the derive-at-call-site pattern the register already flags, so I am not recommending it.
2. **The strict tie rule.** I could not construct a realistic sequence where it misleads, and the doc comment's justification is accurate. For the current spawn's own observed receipt to share the second of its own `spawn_started`, the entire saga (workspace create, tab create, agent start, two agmsg round trips, the `spawned` append) *and* a successful first poll would have to complete inside one wall-clock second — and the poll can only succeed if a qualifying record with a `turn_context` already exists, which the plan's real-record measurement puts 2.3–3.2 s after session start. The comment says "cannot", which is stronger than five records on one codex version support, but it immediately qualifies itself ("If it ever did land in the same second, the cost … is one duplicate observed receipt at stop"), so it does not overstate. The insights consequence of that duplicate is one extra `true`/`false` row for that seat, which the plan's Non-goals already blesses ("1 座席の receipt が複数件でも、true / false / unknown をそのまま数える"). I also checked the inverse direction: a second `Stop` cannot duplicate, because the first Stop's receipt lands strictly after `spawn_started`. And on test flakiness — this rule makes "spawn finds a fixture, then stop" timing-dependent under a real clock, which is a genuine hazard; `TestOrgStop_Codex_DoesNotAppendWhenAlreadyObserved` is the only test with that shape and it is precisely the one given a monotonic fake clock. I enumerated every `TestOrgStop_Codex_*` and confirmed the other eleven write their fixture *after* `Spawn`, so the spawn receipt carries no model and cannot suppress. No flake introduced.
3. **Three-directory window and local dates.** `truncateToLocalDay` is applied to the right value at both callers. At Spawn, `spawnStartedAt` comes from `o.nowTime()`, already in the local zone. At Stop it comes from `time.Parse(time.RFC3339, …)` of a `…Z` string, so it carries `time.UTC` — and `truncateToLocalDay` calls `t.Local()` *before* zeroing the time of day (`codex_session.go:259-262`), which is the conversion that matters. DST is safe: the only thing extracted is the `2006/01/02` component, and `AddDate` re-resolves from the already-normalised instant's own components, so even a zone whose local midnight does not exist on a transition day yields the right date string. A timezone change between spawn and stop shifts the computed local date by at most ~26 h, which the ±1 day pad absorbs. The real exposure in this area is not the arithmetic — it is the *reach* of the window, C2-1.
4. **Pointer-sentence match and normalisation.** The sentence does contain characters with canonical decompositions (`で`, `だ`), so NFC/NFD is a fair question — but nothing in the path normalises: the Go literal's bytes go to argv, through herdr's pane write, into codex, and out through serde into JSON, and none of those layers is a filesystem name (which is where macOS normalisation would bite). The five real records confirm the round trip on the normal path. Two residual risks worth stating rather than fixing: (a) the needle grew from the path (~60 bytes) to the sentence plus path (~85), so a line wrap landing *inside* the sentence would now produce a silent `unknown` where the old matcher survived — `TestObserveCodexEffectiveModel_PointerSentenceInsideLongerTextMatches` covers text *around* the sentence, not a break *within* it, and the five records show no intra-sentence wrapping on 0.154.0; (b) upgrading the ralph binary between a seat's spawn and its stop changes the needle while the record still carries the old sentence. Both degrade to `unknown`, which is the designed failure direction, and both are inside the blast radius of the tech-debt row `/sync-docs` added. If cheap hardening is wanted later, matching on the sentence's leading fragment **and** the path separately would recover (a) without giving back L7's tightening.
5. **`ModelReceipt` on reject.** "The receipt this call appended" is **not** true on `reject`'s append-failure path — that is C2-3. It is true on every other path I traced: `dryRunSpawn` returns `SpawnOutcomeFailed` when its append fails, the real spawn path does the same, and `Stop` assigns only when the append succeeded. The doc's other claims check out: there really is a `SpawnOutcomeRejected` return that appends nothing (the identifier and combined-length validations at `spawn.go:320-337` return before any manifest write), so "reject() is never reached for those" is accurate.
6. **Leftovers.** Stale names: none in Go — `waitBeforeEnter` and `hasObservedCodexReceiptSince` are both gone from every `.go` file, and the renamed table test had its name, its doc comment, *and* the affected case's label updated, not just its assertions. The remaining hits are C2-6 (three same-PR reports) and the plan's cycle-1 log line, which reads as a dated chronological entry and is corrected by the Slice G entry below it — I would leave the plan alone. History in comments: the production files are clean; the test-side citations follow the package's existing convention and only need the cycle and path (C2-5). Plan paths in code: zero. On the rule/recipe/skill wording under the strict rule — "no observed receipt since its spawn" is now off by one second from the code's "strictly after", but "since" at that level of prose means "belonging to this spawn rather than the previous one", which is exactly what the code implements; spelling out the second-boundary tie in an operator-facing rule would add noise, so I am explicitly not filing it.
7. **Test quality of the three new/repaired tests.** All three are sound and all three discriminate. `TestOrgStop_Codex_RecoversPromptPathAfterAgentStartRetry` drives a real retry through `fakeHerdr.agentStartErrs`, asserts the persisted Details actually carries the suffix before relying on it (a setup invariant, not an assumption), and would fail pre-fix because the unstripped path never matches the pointer sentence. `TestOrgStop_Codex_SameSecondRespawn_…` seeds spawn A's receipt at a fixed past second, pins `o.Now` to that same instant for spawn B, and asserts three receipts; pre-fix, A's receipt is `>=` B's `spawn_started`, suppression fires, and the `HonoredTrue` assertion fails. Its one subtlety is handled and commented: the fixture is written relative to the *fake* clock's date so it lands in the window B derives, while its real ModTime is far later, which the pre-filter only ever excludes in the other direction. The repair to `TestOrgStop_Codex_DoesNotAppendWhenAlreadyObserved` is the most valuable change in the delta — dropping the second, contradicting fixture is what turns "exactly one receipt" into a real proof, since with two fixtures a wrongly re-observing Stop would have gone ambiguous and appended nothing, producing the identical pass. `TestPromptPathFromAgentStartedDetails`'s eight cases include both the mid-string-suffix variants, which is the case the recommendation could most easily have skipped.

### Minor observations (no fix requested, recorded so they are not re-found)

- The `/org` skill's new doctor sentence says the info outcome shows 移行先 **and** 退役日; a slug whose `upgrade.model` parses but whose `retirement_at` does not is listed without a date (`codexRetirementClause`, `doctor_codex_models.go:244-247`). A one-word softening if anyone is editing that line anyway; not worth a round on its own.
- `codexRetirementSentence` is consumed by the doctor tests as a constant rather than a copied literal, so no change-detector was created by M4 — the right shape, and worth noting because the opposite is the usual outcome.
- The two `model-routing.md` copies now wrap the edited paragraph differently. That follows from the template copy deliberately omitting the `internal/org/` source path and is not drift.

### Positive notes

- Every one of the eleven cycle-1 findings was fixed in-cycle; nothing was half-swept. The L7 fixture sweep in particular converted eighteen of nineteen `userMessageLine` call sites and left exactly the two intended negatives, so no previously-discriminating "not-found" assertion became vacuous behind the tightened matcher — the failure mode I most expected to find.
- L2's fix is backed by measurement rather than reasoning: 1,013 real records checked on metadata only, with the count of late-modified files and the maximum lag both stated, and the evidence file records the method so it can be repeated on a future codex version.
- Both AR fixes introduced a shared constant across the write and read sides rather than a second literal, matching what `codexPromptFileDetailsPrefix` already did — the drift this PR could most easily have created is now designed out at three seams.
- The implementer repaired a test it was not asked to touch, and explained in the test's own doc comment why the previous shape could not fail. That is the reverse of the usual direction.
- Privacy is unchanged by this cycle: the observer still reads no clock, no environment, and nothing but a model out of the records; the new `codexReadErrorReason` names a category and never the error text or a path; the sentinel test still guards the one error string.

### Tech debt identified

None deferred. All six cycle-2 findings are in-cycle fixes: C2-1's recommended form is two sentences of documentation, C2-2 and C2-5 are comment edits, C2-3 is three lines, C2-4 is an export plus one call, and C2-6 is handled by this cycle's own `/verify`, `/test`, and `/sync-docs` passes. If any of them is consciously left unfixed at the cap, batch them into a single `docs/tech-debt/README.md` row before `/pr` — the plan is archived there and nothing else would carry them.

### Recommendation

- Merge: **yes.** No CRITICAL or HIGH. C2-1 is the only finding I would not ship silently, and its recommended fix is documentation, not code.
- Follow-ups:
  1. C2-1: document the window's reach in `codexSessionDateDirs` and both recipe copies (or widen it with an `observedAt` parameter plus a day cap, keeping the observer clock-free).
  2. C2-2, C2-3, C2-4, C2-5: four small, independent edits.
  3. C2-6: this cycle's `/verify`, `/test`, and `/sync-docs` must re-derive their rows against HEAD instead of carrying the old symbol name and the closed gap forward.
