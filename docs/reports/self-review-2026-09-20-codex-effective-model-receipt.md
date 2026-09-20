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
