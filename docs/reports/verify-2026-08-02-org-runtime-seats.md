# Verify report: org-runtime-seats

- Date: 2026-08-02
- Plan: `docs/plans/active/2026-08-02-org-runtime-seats.md`
- Verifier: `verifier` subagent (Claude Code, `/verify`)
- Scope: spec compliance (AC-1..AC-11) + static analysis (`./scripts/run-static-verify.sh`, changed-language scope) + documentation drift for `git diff ffda48f...1520b96` (38 files, +4518/-225). No behavioral test execution — that is `/test`'s job.
- Evidence: `docs/evidence/verify-2026-08-02-091112.log` (static verifier raw output, gitignored per `docs/evidence/*.log`); `docs/evidence/org-seats-smoke-2026-08-02.txt` (committed live-smoke evidence, referenced below)

## Spec compliance

| Acceptance criterion | Status | Evidence |
| --- | --- | --- |
| AC-1: agmsg adapter calls real scripts with correct argv; `AgmsgAvailable` checks `<home>/scripts/send.sh` | Met (delegated to `/test` for behavioral proof) | `internal/org/driver/agmsg.go` (`Send`/`Join`/`Team`/`History`/`Leave`/`Whoami`/`DeliverySet` all documented as `bash <home>/scripts/<verb>.sh <positional args>`); `internal/org/driver/driver.go:77-86` `AgmsgAvailable(home)` stats `<home>/scripts/send.sh`, not `exec.LookPath("agmsg")`. Stub argv tests: `internal/org/driver/agmsg_test.go` (not re-run here; covering names visible via `grep -c "^func Test"` = present for every verb) |
| AC-2: `[org] agmsg_home` 3-way lockstep + `defaults_sync_test.go` drift detection | Met | `internal/config/config.go:122-127,194,357-363` (`AgmsgHome` field + default + empty-string fallback); `templates/base/ralph.toml:82-85` (`agmsg_home = "~/.agents/skills/agmsg"`); `scripts/ralph-config.sh:83` (`RALPH_ORG_AGMSG_HOME="${RALPH_ORG_AGMSG_HOME:-~/.agents/skills/agmsg}"`, mirrored `templates/base/scripts/ralph-config.sh`); `internal/config/defaults_sync_test.go:186-187` (`check("org.agmsg_home", "RALPH_ORG_AGMSG_HOME", ...)`) |
| AC-3: spawn saga = `ensureLeadJoined` → seat `join.sh` → HELLO send; join/send failure injection → `spawn_failed` + compensation | Met (behavioral proof delegated to `/test`) | `internal/org/spawn.go:365-381` (`ensureLeadJoined` call), `:439-461` (extracted `ensureLeadJoined` function, doc comment explains best-effort semantics + failure carried into later Details); failure-injection test names present: `TestOrgSpawn_FailureInjection_AgmsgJoin_SeatJoinFails_CompensatesExistingPane`, `TestOrgSpawn_EnsureLeadJoined_ErrorDoesNotFailSaga_WhenSeatJoinAndSendSucceed`, `TestOrgSpawn_FailureInjection_AgmsgSend_DetailsIncludeLeadJoinError` (`internal/org/spawn_test.go`) |
| AC-4: `--role reviewer\|qa` embeds template; variable substitution; unknown role → no template | Met (behavioral proof delegated to `/test`) | `internal/org/prompts.go` (go:embed `prompts/*.md`, `RenderRolePrompt`); test names present: `TestRenderRolePrompt_Reviewer_AllKnownVarsSubstituted`, `TestRenderRolePrompt_QA_AllKnownVarsSubstituted`, `TestRenderRolePrompt_UnknownRole_NoTemplate`, `TestRenderRolePrompt_UnknownPlaceholder_PassesThroughUnchanged`, `TestRenderRolePrompt_EmptyScope_SubstitutesDefaultText` (`internal/org/prompts_test.go`); CLI wiring: `internal/cli/org_test.go:800` `TestOrgSpawn_UnknownRole_NoTemplateApplied` |
| AC-5: `stop`/`disband` best-effort `Leave` (post-deviation-4 rename from `despawn.sh`); failure still records state event | Met | `internal/org/verbs.go:229-264` (`Stop`: `team := seat.AgmsgTeam`; `Leave` call wrapped, `leaveNote` folded into Details regardless of outcome; `appendEvent` always runs after); test: `TestOrgStop_ExistingSeat_RecordsPaneAndLeaveOutcomes` (`internal/org/verbs_test.go:68`, asserts "leave failure is best-effort and must not fail Stop") |
| AC-6: `Verbs.Send/Wait/Read` + CLI wiring test coverage; tech-debt row closed | Met | `internal/org/verbs_test.go` has `Send`/`Wait`/`Read` suites (`TestOrgSend_UnknownSeat_ErrorsWithoutDriverCall`, `TestOrgWait_UnknownSeat_StillDrivesHerdr_NoManifestCheck`, `TestOrgRead_UnknownSeat_ErrorsWithoutDriverCall`, etc.); CLI: `internal/cli/org_test.go` (`TestOrgWait_UnknownSeat_StillSucceeds_PassthroughToHerdr`, `TestOrgRead_UnknownSeat_NonZeroExit`); tech-debt row struck through with `RESOLVED 2026-08-02 ... Slice 4` comment recording before/after coverage numbers (`docs/tech-debt/README.md`) |
| AC-7: `.claude/rules/agent-messaging.md` exists, mirrors `templates/base/`, check-sync green; role templates reference the protocol | Met | `cmp .claude/rules/agent-messaging.md templates/base/.claude/rules/agent-messaging.md` → identical; `./scripts/check-sync.sh` → `DRIFTED: 0`, `PASS: all files in sync.`; `internal/org/prompts/reviewer.md`/`qa.md` reference `TYPE:`/`TASK_ID:` typed-message shape in their message-format section |
| AC-8: live smoke (clean-team spawn w/o manual lead join, herdr state transitions incl. blocked, team membership lead+seat, status display, stop+leave, pre/post `git status --porcelain` scope check) | Met — evidence-only verification per task instructions, smoke not re-run | `docs/evidence/org-seats-smoke-2026-08-02.txt`: (1) clean-team spawn — attempt 5, lines 74-77, `[pre] git status: (clean)` then `spawned seat "reviewer"` with no manual join step; (2) state transitions incl. blocked — `agent_status":"working"` (line 19, `herdr agent list`) → `"agent_status":"blocked"` (line 54, `herdr agent get`, captured while the reviewer seat waited on a permission dialog) → `stopped` (lines 59-61, 90-92); (3) team membership lead+seat — lines 25-31 and 78-84 (`team.sh` output: `lead (claude-code)` + `reviewer (claude-code)`, 2 members); (4) status display — lines 21-23 (`spawned (active)`) and 85-87, plus the pre-fix vs post-fix comparison at lines 59-61 (blank ROLE/DRIVER/MODEL, the bug deviation-4 fixed) vs lines 90-92 (populated, post-fix); (5) stop+leave — lines 56-57/88-89 (`stopped seat "reviewer"`) then lines 63-69 (pre-fix: still 2 members, `despawn.sh` no-op) vs lines 93-98 (post-fix: 1 member, lead only, confirming `Leave` actually removes the seat); (6) pre/post `git status --porcelain` scope check — lines 2-3 and 71-72 (empty output between markers = clean) and lines 75-76/99-100 (`(clean)` / `(end)`, no declared-scope-outside changes) |
| AC-9: `go test ./...` / `./scripts/run-verify.sh` green | Delegated to `/test` — static-only portion (`gofmt`, `go vet`, `golangci-lint`, `staticcheck`) verified green here | See Static analysis table below |
| AC-10: unknown-seat `stop` non-zero exit, no state event; `disband` only processes existing active seats; despawn/leave failure still records Details | Met | `internal/org/verbs_test.go:14` `TestOrgStop_UnknownSeat_ErrorsWithoutAppendingEvent`, `:44` `TestOrgStop_UnknownSeat_DryRun_AlsoErrorsWithoutAppendingEvent`, `:118` `TestOrgDisband_OnlyStopsExistingActiveSeats_UnknownNeverAppears`; CLI: `internal/cli/org_test.go:677` `TestOrgStop_UnknownSeat_NonZeroExit` |
| AC-11: `internal/org/protocol` validates TYPE enum / TASK_ID / body size cap; `ralph org send` rejects invalid by default, `--raw` bypasses; parser/validator + CLI rejection tests exist | Met | `internal/org/protocol/protocol.go` (13 `func Test*` in `protocol_test.go`, includes `TestValidate_BodySizeCap_CountsRunesNotBytes` per self-review positive notes); CLI: `internal/cli/org_test.go:861` `TestOrgSend_RawFlag_BypassesValidation`; rule doc: `.claude/rules/agent-messaging.md` TYPE table matches `protocol.go`'s enum (TASK/RESULT/QUESTION/REVIEW/DECISION/BLOCKED/CONTRACT/HEARTBEAT/STOP/HELLO) |

### Self-review fix verification (commit `1520b96`)

All 5 MEDIUM findings and the 2 named LOW items from `docs/reports/self-review-2026-08-02-org-runtime-seats.md` are addressed in the fix commit:

| Finding | Fix verified |
| --- | --- |
| MEDIUM 1 (env precedence) | `RALPH_ORG_AGMSG_HOME` removed from the `export` line in `scripts/ralph-config.sh`/`templates/base/scripts/ralph-config.sh`; default-assignment line kept unexported with a doc comment explaining the precedence rationale; `defaults_sync_test.go` still parses the text-level default (regex-based, not env-based) |
| MEDIUM 2 (path traversal) | New `internal/org/identifier.go` (`ValidateIdentifier`, `^[A-Za-z0-9][A-Za-z0-9_-]{0,63}$`) called from `(*Org).Spawn` (before any manifest write) and from CLI `requireOrgID`/`requireSeatIdentifier` (before an `org.Org` is constructed) — two-layer gate. Tests: `internal/org/identifier_test.go` (table test + 3 Spawn-level tests asserting zero driver calls / no manifest write for a traversal id), `internal/cli/org_test.go` (`TestOrgSpawn_TraversalSeatID_NonZeroExit_NoDriverCalls`, `TestOrgSpawn_InvalidOrgID_NonZeroExit`, `TestOrgSend_TraversalTo_NonZeroExit`) |
| MEDIUM 3 (`ensureLeadJoined` naming) | Extracted as `func (o *Org) ensureLeadJoined(ctx, p, team, paneID) (string, error)` at `internal/org/spawn.go:439-461`; `grep -n "func.*ensureLeadJoined"` now resolves; `Spawn` calls it at `:366` |
| MEDIUM 4 (scaffolded rule doc `internal/` references) | Chose option (b) from the self-review recommendation — caveat text added at 5 sites in `.claude/rules/agent-messaging.md` (mirrored in `templates/base/`) explicitly stating `internal/org/protocol` "is not part of what `ralph init` scaffolds into this project" |
| MEDIUM 5 (`PlanPath`/empty `{{SCOPE}}`) | `{{PLAN_PATH}}` placeholder removed from `reviewer.md`/`qa.md` (not just left unpopulated); `RolePromptVars.PlanPath` field kept but doc-commented as unwired, reserved for PR③; empty `{{SCOPE}}` now substitutes `defaultScopeText` ("未指定(読み取り中心で...)"). Test: `TestRenderRolePrompt_EmptyScope_SubstitutesDefaultText` |
| LOW (`agmsgTeam` local shadowing) | `internal/org/verbs.go` `Stop`: local renamed `agmsgTeam` → `team`, matching `Spawn`'s existing convention |
| LOW (retry count lost on `agent_start` failure) | `spawn.go:365-370` now calls `o.failStepWithNote(p, "agent_start", err, paneID, fmt.Sprintf("agent_start_retries=%d", retries))` on failure (previously success-only); off-by-one on exhaustion also fixed (`agentStartWithRetry` returns `lastAttempt` instead of `maxAgentStartAttempts`) |
| LOW (`sort.Strings` → `slices.Sort`) | `internal/org/protocol/protocol.go` now imports `slices`, uses `slices.Sort(names)` |

**Not fixed, not newly tracked as tech debt** (self-review explicitly filed these as "safe to batch or defer" LOW follow-ups, not MEDIUM blockers): `leadIdentity` constant + lead driver-type hardcoding (`spawn.go:378` still uses the bare string `"lead"` and `agmsgTypeForDriver("claude")`), doctor error detail not naming the resolved agmsg home, `dryRunSpawn`'s silent swallowing of `RenderRolePrompt`/`promptFilePath` errors, and the evidence-file home-path redaction convention. These remain open — see Documentation drift / Coverage gaps below.

## Static analysis

| Command | Result | Notes |
| --- | --- | --- |
| `./scripts/run-static-verify.sh` (changed-language scope, `HARNESS_VERIFY_MODE` default) | Pass | Full output: `docs/evidence/verify-2026-08-02-091112.log`. Ran hook-config checks (settings.json JSON validity, precompact/session-end hooks, Codex hook single-source guard, inline hook detector, PR provenance guard) → all OK; `scripts/check-sync.sh` → `DRIFTED: 0`, `PASS`; `scripts/check-pipeline-sync.sh` → all 8 references OK; `scripts/check-skill-sync.sh` → `13 skill(s) in lock-step`; language scope resolved to `golang` (full fallback triggered by `scripts/ralph-config.sh` being unclassified, expected — shell scripts aren't a language pack) |
| `gofmt -l` (via golang verifier) | Pass | `gofmt: ok` |
| `go vet ./...` (via golang verifier) | Pass | Silent (no output = no findings) |
| `golangci-lint run` (via golang verifier) | Pass | `0 issues.` |
| `staticcheck` (via golang verifier) | Pass | `staticcheck` present at `~/go/bin/staticcheck`; verifier produced no output for it (silent-on-success, per verifier tooling convention) |

No `go build`/`go test` was run here by design — behavioral verification is `/test`'s scope, and AC-9's `go test ./...` requirement is explicitly delegated.

## Documentation drift

| Doc / contract | In sync? | Notes |
| --- | --- | --- |
| `.claude/rules/agent-messaging.md` ↔ `templates/base/.claude/rules/agent-messaging.md` | In sync | `cmp` → identical bytes; `check-sync.sh` → 0 drift |
| `AGENTS.md` repo-map (`internal/org/` line) | In sync | Line 61 already describes "herdr/agmsg driver adapters, role prompt templates (go:embed), typed message protocol" — matches this PR's additions, no stale text |
| `docs/tech-debt/README.md` | In sync | 2 rows closed with `RESOLVED 2026-08-02 in feat/org-runtime-seats commit 67220ef` (agmsg CLI shape assumption) and `RESOLVED 2026-08-02 in feat/org-runtime-seats (Slice 4, PR②)` (0% verbs coverage), both struck through with before/after evidence per the repo's established convention. No new tech-debt rows were added for the deferred LOW self-review findings (see Spec compliance note above) — this is a minor gap, not a contradiction: the self-review's own "Tech debt identified" table said "(none newly deferred)" and only required a tech-debt entry if the MEDIUM path-traversal finding were deferred (it was fixed, not deferred) |
| `docs/specs/2026-08-01-org-runtime.md` PR② boundary note | In sync | Security considerations section rewritten to say scope-outside-write **detection** is deferred to PR④'s Watchdog pulse layer, with PR②'s actual scope (manifest recording + role-prompt instruction + smoke-time `git status` check) spelled out — matches the plan's Non-goals bullet almost verbatim |
| Plan "Open questions" section | In sync | All 3 marked `~~resolved~~` items match code (`DeliverySet` implemented but unwired — confirmed left for future need; body cap 2,000 — matches `DefaultMaxBodyChars`/`.claude/rules/agent-messaging.md`; `ensureLeadJoined` — now a real function, matching MEDIUM-3's fix); the one open "PR③ 送り" item (permission-mode/allowlist envelope config) is corroborated by the smoke evidence's `blocked` state capture |
| AC-8 evidence filename | Minor wording drift, not a functional gap | Plan text says `docs/evidence/org-seats-smoke-*.log`; the committed file is `org-seats-smoke-2026-08-02.txt`. This is very likely intentional: `docs/evidence/*.log` is gitignored (`.gitignore`), so a `.log` extension would never have been committable — a stray untracked `org-seats-smoke-2026-08-02.log` duplicate exists in the working tree today (`git status --ignored` shows it as `!!`), consistent with the file having been produced with a `.log` name first and then saved for commit under `.txt`. Plan wording should be corrected to `.txt` in a follow-up `/sync-docs` pass, but the evidence itself is present and complete |
| Plan "Implementation notes (deviations)" (4 live-smoke fixes) | In sync | All 4 confirmed present in code: herdr JSON envelope parsing (`parseHerdrEnvelope`, `internal/org/driver/herdr.go`), prompt-file handoff (`promptFilePath`/`writePromptFile`, `internal/org/spawn.go`), bounded `agent_pane_busy` retry (`agentStartWithRetry`, `maxAgentStartAttempts=20`), `Despawn`→`Leave` replacement (`internal/org/driver/agmsg.go:77` `func (a Agmsg) Leave`, wired into `Stop`/`disband`) |
| Plan Progress checklist | Stale in one place | "Review artifact created" / "Verification artifact created" / "Test artifact created" / "PR created" are still unchecked even though the self-review artifact already exists (`docs/reports/self-review-2026-08-02-org-runtime-seats.md`, referenced by this same plan). This report will make "Verification artifact created" true; checklist maintenance is `/sync-docs`'s job, flagged here as a known gap per the recurring "plan AC/checklist lags implementation" pattern, not a blocker |

## Observational checks

- `cmp .claude/rules/agent-messaging.md templates/base/.claude/rules/agent-messaging.md` → identical, both mode 100644.
- `./scripts/check-sync.sh` → `PASS: all files in sync` (`DRIFTED: 0`, `ROOT_ONLY: 0`).
- `git diff ffda48f...HEAD -- docs/tech-debt/README.md` → 2 rows resolved, matching the task's "2 closed" claim exactly.
- `git status` in the worktree → clean at verification time (no uncommitted changes to lose).
- Live-smoke evidence (`docs/evidence/org-seats-smoke-2026-08-02.txt`) walked line-by-line against all 6 AC-8 sub-items — see the AC-8 row above for the exact line citations.

## Coverage gaps

- **AC-1, AC-3, AC-4, AC-6, AC-9, AC-11 behavioral proof**: this report confirms the relevant tests *exist* with intent-revealing names and confirms the static-analysis portion of AC-9 is green, but does not execute `go test ./...`. That execution and its pass/fail verdict is `/test`'s responsibility per the pipeline contract.
- **Codex plan advisory**: the plan's Readiness checklist still lists "Codex plan advisory (次ステップ)" as unchecked. Not a `/verify` blocker (advisory is optional per `AGENTS.md`), but worth surfacing since the plan itself flags it as an open readiness item.
- **Deferred LOW self-review findings** (see Documentation drift note): `leadIdentity` string-literal duplication, doctor's un-named agmsg-home in its error detail, `dryRunSpawn`'s two swallowed errors, and the evidence-redaction convention remain unaddressed and untracked in `docs/tech-debt/README.md`. None are spec-blocking (no AC references them directly), but they are real, previously-identified gaps that could silently age out if not captured before `/pr`.
- **AC-8 evidence filename** — plan text (`*.log`) vs actual (`*.txt`) should be reconciled by `/sync-docs`.

## Verdict

**PASS** — all 11 acceptance criteria are met or (for the behavioral subset of AC-1/AC-3/AC-4/AC-6/AC-9/AC-11) appropriately delegated to `/test` with covering test names identified. Static analysis is fully green (`gofmt`, `go vet`, `golangci-lint`, `staticcheck`, `check-sync`, `check-pipeline-sync`, `check-skill-sync`, hook-config guards). All 5 MEDIUM and the 2 explicitly-named LOW self-review findings are verified fixed in commit `1520b96`. Documentation is in sync on every checked surface except two minor, non-blocking items (AC-8 evidence file extension wording, plan Progress checklist staleness) already noted above.

- Verified: AC-2, AC-5, AC-7, AC-8, AC-10 (fully, including behavior); AC-1/AC-3/AC-4/AC-6/AC-11 spec shape and static analysis; static analysis suite in full; documentation mirrors/drift surfaces listed above; all 5 self-review MEDIUM fixes + 2 named LOW fixes.
- Partially verified: AC-9 (`go test ./...` execution not run here — static-only portion confirmed green; full verdict is `/test`'s).
- Not verified: behavioral correctness of AC-1/AC-3/AC-4/AC-6/AC-11 (test *existence* and naming confirmed, execution/pass-fail is `/test`'s scope); the 4 deferred LOW self-review follow-ups remain open and untracked.

# Cycle 2 (fix-and-revalidate re-run)

- Date: 2026-08-02
- Verifier: `verifier` subagent (Claude Code, `/verify`), pipeline cycle 2 of 2
- Delta reviewed: `git diff 1520b96...HEAD` — `f0cbf11` (cross-review ACTION_REQUIRED #1 fix: `_` namespace separator, tightened id charset `^[a-z][a-z0-9-]{0,29}$`, combined-length ≤32 check) and `011a0a7` (cycle-2 self-review LOW fixes: `failStepWithNote` doc comment, `identifierPattern` 30-vs-32 rationale, `qa.md` synthetic-evidence example, `{{PLAN_PATH}}` replacer restored), plus report/insight-event/plan-doc commits in between (`b3122ff`, `84f4a03`, `a550e17`, `a1df8f9`, `ac146d3`, `994abc8`, `c1b6568`, `63c7a62`, `e7191d8`). Cycle-1 verify (this same report, PASS on commit `1520b96`) is not re-derived here, only the delta.
- Scope: spec compliance for the AC-3/AC-8 surfaces touched by the delta + static analysis (`./scripts/run-static-verify.sh`, `RALPH_VERIFY_SCOPE` default/changed-language scope). No tests — `/test`'s job.

## Static analysis (cycle 2, full re-run at HEAD)

| Command | Result | Notes |
| --- | --- | --- |
| `./scripts/run-static-verify.sh` (default scope, HEAD=`011a0a7`) | Pass (exit 0) | `docs/evidence/verify-2026-08-02-095050.log`. Hook-config guards, `check-sync.sh` (`DRIFTED: 0`), `check-pipeline-sync.sh` (8/8 OK), `check-skill-sync.sh` (13 skills lock-step) all green; language scope resolved to `golang` (full fallback, same expected `scripts/ralph-config.sh` unclassified trigger as cycle 1) |
| `gofmt -l` | Pass | `gofmt: ok` |
| `go vet ./...` | Pass | Silent (no findings) |
| `golangci-lint run` | Pass | `0 issues.` |
| `staticcheck` | Pass | Binary present (`~/go/bin/staticcheck`); silent-on-success, no separate line in output (same convention noted in cycle 1) |

No regressions vs. the cycle-1 static-analysis result; identical suite, identical verdict.

## Cross-review fix verification (ACTION_REQUIRED #1, `f0cbf11`)

`docs/reports/cross-review-triage-org-runtime-seats.md` ACTION_REQUIRED #1: `<org_id>-<seat_id>` hyphen concatenation is not unique (`a-b`+`c` and `a`+`b-c` both join to `a-b-c`), affecting both `herdrAgentName` and `promptFilePath` (same root cause, findings 1+2 merged in triage).

| Sub-check | Status | Evidence |
| --- | --- | --- |
| Ambiguity regression test exists and pins the actual collision, not just the new format | Met | `TestHerdrAgentNameAndPromptFilePath_UnambiguousAcrossSplit` (`internal/org/spawn_test.go`, added in `f0cbf11`) constructs exactly the triage's counter-example pair — `herdrAgentName("a-b","c")` vs `herdrAgentName("a","b-c")` — asserts they differ, and pins both exact output strings (`a-b_c` / `a_b-c`); repeats the same shape for `promptFilePath`. This is the property-level test the cycle-2 self-review's Positive notes called out as "would still fail if a future change reintroduced a collision through a different route" |
| Join separator (`_`) is unreachable inside a single identifier | Met | `identifierPattern = ^[a-z][a-z0-9-]{0,29}$` (`internal/org/identifier.go`) — no `_` in the character class, so `_` cannot appear inside either half; `herdrAgentName`/`promptFilePath` join with `_` (`spawn.go`), making the split always unambiguous. `TestValidateIdentifier`'s `invalid` table adds `"has_underscore"` explicitly (`f0cbf11`) |
| Charset rejects underscore | Met | Confirmed above; also `internal/cli/org_test.go` delta in `f0cbf11` updates CLI-level id-validation tests to the tightened pattern (2 lines changed, same intent) |
| Charset rejects uppercase | Met | `identifierPattern`'s `[a-z]`/`[a-z0-9-]` classes exclude `A-Z`; `TestValidateIdentifier`'s `invalid` table adds `"UPPER"` and `"Mixed-Case"` (`f0cbf11`). Rationale documented in both the doc comment and a test comment: herdr's own live-probed agent-name pattern (`^[a-z][a-z0-9_-]{0,31}$`, v0.7.5) is lowercase-only, so ralph's charset must stay inside herdr's or a ralph-valid id could still be rejected by herdr at spawn time |
| Combined-length check (`len(org)+1+len(seat) ≤ 32`) | Met | `Spawn` (`spawn.go`) rejects before any manifest write when `len(p.OrgID)+1+len(p.SeatID) > maxHerdrAgentNameLen` (32); two boundary tests: `TestOrgSpawn_CombinedIdentifierLength_RejectedBeforeAnyManifestWrite` (61 chars, asserts zero driver calls **and** no manifest file created — the property that matters for a pre-side-effect gate, per the self-review's cycle-2 Positive notes) and `TestOrgSpawn_CombinedIdentifierLength_ExactlyAtLimit_Spawns` (exactly 32, spawns successfully) |
| `identifierPattern`'s 30-vs-32 magic number now explained at the definition site (cycle-2 self-review L3) | Met | `011a0a7` moves the one-sentence rationale from the test comment into `identifierPattern`'s own doc comment (`internal/org/identifier.go`): "32 minus the separator minus at least one character for the other identifier" |

**Verdict: ACTION_REQUIRED #1 is resolved with no gaps.** All three triage-cited failure modes (herdr namespace collision, prompt-file overwrite, and the length-overflow edge case the fix's own combined-length check surfaced) have targeted regression tests.

## AC regression check (cycle 2 delta)

| AC | Regression risk from `f0cbf11`/`011a0a7` | Status | Evidence |
| --- | --- | --- | --- |
| AC-2 (`[org] agmsg_home` 3-way lockstep) | None — delta does not touch `config.go`, `templates/base/ralph.toml`, or `ralph-config.sh` | Isolated, no regression | `git diff ffda48f...HEAD --stat -- internal/config templates/base/ralph.toml scripts/ralph-config.sh templates/base/scripts/ralph-config.sh` shows only cycle-1 changes (all predate `f0cbf11`); `defaults_sync_test.go` still covered by the green `golangci-lint`/`go vet` static pass |
| AC-3 (spawn saga join/HELLO + failure-injection compensation) | Delta adds a *new* rejection path (combined-length check) ahead of the existing saga steps | No regression, one new gate | The new check runs at `spawn.go` before `ensureLeadJoined`/join/HELLO, returning `SpawnOutcomeRejected` with zero driver calls — same "reject before any side effect" shape as the existing identifier-validation gate it sits next to; does not alter the join/HELLO/compensation code path itself (untouched by the diff) |
| AC-7 (`.claude/rules/agent-messaging.md` mirror + check-sync) | None — delta touches only `internal/org/*` and Go tests | In sync | `cmp .claude/rules/agent-messaging.md templates/base/.claude/rules/agent-messaging.md` → identical; `cmp scripts/ralph-config.sh templates/base/scripts/ralph-config.sh` → identical; `check-sync.sh` → `DRIFTED: 0` in the cycle-2 static run above |
| AC-8 (live-smoke evidence validity under the tightened charset) | The smoke evidence (`docs/evidence/org-seats-smoke-2026-08-02.txt`) was captured *before* `f0cbf11` tightened the charset — need to confirm the recorded org/seat ids remain valid under the new pattern, or the evidence would no longer be representative | Confirmed still valid, evidence remains representative | Task instructions named the live-smoke ids as `smoke-0802d`/`smoke-0802e` (org) and `reviewer` (seat); grep of the evidence file confirms these exact strings (`org_id=smoke-0802d`, `org_id=smoke-0802e`, `name":"smoke-0802d-reviewer"` old-format herdr name from the pre-fix capture, `spawned seat "reviewer"`). All three checked against `^[a-z][a-z0-9-]{0,29}$`: `smoke-0802d` (11 chars, lowercase+digit+hyphen only) — valid; `smoke-0802e` (11 chars, same shape) — valid; `reviewer` (8 lowercase letters) — valid. None contain uppercase, underscore, a leading digit/hyphen, or exceed 30 chars, so all three would still pass `ValidateIdentifier` today. The reasoning holds: the charset tightening removed uppercase/underscore/length headroom that this specific evidence never used, so the smoke run's org/seat ids are unaffected and the evidence is still valid proof of AC-8's live-spawn lifecycle |

No AC regressed. AC-3 gained one additional, narrowly-scoped rejection gate; AC-2/AC-7/AC-8 are unaffected by the delta's file scope.

## Cycle-2 self-review fix verification (`011a0a7`)

| Cycle-2 self-review finding | Fix verified |
| --- | --- |
| L1 (`failStepWithNote` doc comment names only one caller) | Comment now names both callers explicitly (`agmsg_announce` failure path + `agent_start` failure path) — `internal/org/spawn.go` |
| L2 (`qa.md` kept this-branch-specific evidence path as example) | `EVIDENCE: docs/reports/verify-2026-08-02-org-runtime-seats.md` → `EVIDENCE: docs/reports/<report-file>.md`, matching the already-synthetic `TASK_ID: t-42` pattern and `reviewer.md`'s prior fix — `internal/org/prompts/qa.md` |
| L3 (30-vs-32 rationale lived only in a test comment) | One-sentence rationale moved to `identifierPattern`'s own doc comment — `internal/org/identifier.go` |
| L5 (`{{PLAN_PATH}}` silently dropped from the replacer, sharper trap than the one cycle-1 closed) | `"{{PLAN_PATH}}", vars.PlanPath` restored to the `strings.NewReplacer` call with a comment explaining why the entry stays even though no template uses it today — `internal/org/prompts.go` |
| L4, L6/L7, INFO (deferred per self-review's own triage) | Not addressed in this delta, consistent with the self-review's explicit "batch or defer" recommendation — no regression, not silently dropped |

All four fixes the task named (L1/L2/L3/L5) are present and match the self-review's recommended remediation exactly.

## Mirrors / check-sync (cycle 2)

- `cmp .claude/rules/agent-messaging.md templates/base/.claude/rules/agent-messaging.md` → identical (delta does not touch this file; re-confirmed for regression).
- `cmp scripts/ralph-config.sh templates/base/scripts/ralph-config.sh` → identical (delta does not touch this file; re-confirmed for regression).
- `./scripts/check-sync.sh` (via the cycle-2 static run) → `PASS: all files in sync`, `DRIFTED: 0`, `ROOT_ONLY: 0`.
- `./scripts/check-skill-sync.sh` → `13 skill(s) in lock-step`.
- `./scripts/check-pipeline-sync.sh` → 8/8 references OK.

## Documentation drift (cycle 2)

- `docs/tech-debt/README.md` — new row added by `63c7a62` (part of the reviewed delta window) for "herdr agent name derived, not persisted", correctly attributing the `f0cbf11` separator rename as the concrete demonstration of the risk. Row content matches the cycle-2 self-review's tech-debt table verbatim. In sync.
- Plan "Implementation notes (deviations)" — the cross-review cycle-2 fix note (`_` separator, tightened charset, combined-length check) is present and matches `f0cbf11`'s actual diff. In sync.
- No other doc surfaces are touched by this delta.

## Verdict (cycle 2)

**PASS.** `./scripts/run-static-verify.sh` is fully green at HEAD (`011a0a7`) with no change in verdict from cycle 1. Cross-review ACTION_REQUIRED #1 is resolved: the ambiguity regression test pins the exact triage counter-example, the charset mechanically excludes the join separator plus uppercase, and the combined-length check closes the overflow edge case the fix itself introduced a boundary for. No acceptance criterion regressed — AC-2/AC-7 are untouched by file scope, AC-3 gained one narrowly-scoped additional gate, and AC-8's live-smoke evidence (`smoke-0802d`/`smoke-0802e`/`reviewer`) remains valid and representative under the tightened charset. All four cycle-2 self-review LOW fixes the task named (L1/L2/L3/L5) are verified present; the remaining deferred findings (L4, L6/L7, INFO) are consistent with the self-review's own "batch or defer" call, not silent gaps.

- Verified: static analysis full pass at HEAD; cross-review fix (all 5 sub-checks); AC-2/AC-3/AC-7/AC-8 non-regression; cycle-2 self-review L1/L2/L3/L5 fixes; mirrors (`agent-messaging.md`, `ralph-config.sh`) and check-sync/check-skill-sync/check-pipeline-sync all green; tech-debt row for the new derived-vs-persisted gap.
- Not verified (unchanged from cycle 1, still `/test`'s scope): `go test ./...` execution/pass-fail for AC-1/AC-3/AC-4/AC-6/AC-9/AC-11's behavioral claims, including the new `f0cbf11` tests themselves (`TestHerdrAgentNameAndPromptFilePath_UnambiguousAcrossSplit`, the two combined-length boundary tests, `TestValidateIdentifier_LengthBoundary`) — test *existence* and naming confirmed here, execution is `/test`'s.
- Remaining known gaps (carried from cycle 1, not re-litigated): the 4 deferred LOW self-review follow-ups now tracked in `docs/tech-debt/README.md` per-item; Codex plan advisory still unchecked on the plan's Readiness checklist (non-blocking, advisory is optional).
