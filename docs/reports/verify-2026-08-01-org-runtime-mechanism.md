# Verify report: org-runtime-mechanism

- Date: 2026-08-01
- Plan: `docs/plans/active/2026-08-01-org-runtime-mechanism.md`
- Spec: `docs/specs/2026-08-01-org-runtime.md` (PR① — FR-1, FR-2, FR-9 mechanism part + doctor part of FR-2/AC-5)
- Verifier: `verifier` subagent (Claude Code, `/verify`)
- Scope: spec compliance (AC-1..AC-10) + static analysis (`./scripts/run-static-verify.sh`) + documentation drift on `git diff 9abaaed...9bfe07e` (branch `docs/spec-org-runtime`). No behavioral tests run (delegated to `/test`).
- Evidence: `docs/evidence/verify-2026-08-01-035137.log`

## Spec compliance

| Acceptance criterion | Status | Evidence |
| --- | --- | --- |
| AC-1 (reject out-of-pool model, receipts `honored=false`) | Verified (static) | `internal/org/envelope.go:30-44` `ValidateSpawn` rejects unknown driver/model/role-model; `internal/org/spawn.go:256-267` `reject()` appends `EventRejected` + `Receipt{Honored: HonoredFalse}`; CLI returns the error so cobra exits non-zero (`internal/cli/org.go` spawn cmd `return err`-style flow). Behavioral proof delegated to `/test`: `TestOrgSpawn_Rejected_OutOfPoolModel` (`internal/org/spawn_test.go:254`). |
| AC-2 (max_seats org-isolated, concurrent 2-namespace test) | Verified (static) | `spawn.go:126` computes `activeSeats := ActiveSeatCount(events, p.OrgID, RosterOptions{})` scoped to `p.OrgID`; `envelope.go:40-42` rejects at `activeSeats >= cfg.MaxSeats`. Delegated to `/test`: `TestOrgSpawn_Rejected_MaxSeatsReached_OrgIsolated` (`spawn_test.go:285`). |
| AC-3 (idempotent respawn, no duplicate pane/agmsg join) | Verified (static) | `spawn.go:148-160`: existing `EventSpawned` → `SpawnOutcomeIdempotent` with no driver calls; existing `EventSpawnStarted`/`EventSpawnStep` (stale in-flight) → `compensateStale` then fresh saga. Delegated to `/test`: `TestOrgSpawn_Idempotent_AlreadySpawned_NoNewDriverCalls`, `TestOrgSpawn_StaleInFlight_CompensatesThenRespawnsFresh`. |
| AC-4 (status works with all LLM/herdr/agmsg stopped; dry-run excluded by default) | Verified (static) | `verbs.go:231-244` `Status()` derives roster purely from `Manifest.Read()` — no Herdr/Agmsg calls. Dry-run excluded by default via `RosterOptions{IncludeDryRun: all}` (`all` defaults false, CLI flag `--all`, `org.go:355`). Corrupt-line tolerance verified in `manifest.go:249-275` (skips and counts, never aborts). Delegated to `/test`: `TestRoster_DryRunExclusionAndInclusion`, `TestManifestStore_Read_SkipsAndCountsCorruptLines`. |
| AC-5 (doctor reports herdr/agmsg absence; `--probe-models` warns invalid model IDs) | Verified (static) | `internal/cli/doctor.go:594-621` `checkHerdrAvailable`/`checkAgmsgAvailable` report `"info"` on absence (never `"fail"`/`"warn"`); `checkOrgModelProbes`/`probeOrgModel` (`doctor.go:639-687`) probe every pool entry, report `"warn"` on failure. Exit-code aggregation only reacts to `"fail"` (`doctor.go:143`), confirming info/warn never affect exit code. Delegated to `/test`: `internal/cli/doctor_org_test.go`. |
| AC-6 (`[org]` defaults agree across config.go / templates/base/ralph.toml / ralph-config.sh; `defaults_sync_test.go` detects drift) | Verified (static + tooling) | `internal/config/defaults_sync_test.go:158-185` adds 7 `check(...)` calls covering `driver_pool`, `model_pool`, `max_seats`, both budget fields, `max_fix_rounds`, `deadman_minutes`. `cmp scripts/ralph-config.sh templates/base/scripts/ralph-config.sh` → identical. Test execution itself delegated to `/test` (`go test ./internal/config/...`). |
| AC-7 (`go test ./...` and `run-verify.sh` green; no diff to existing-flow files) | Partially verified | Static half verified now: `go build ./...` clean, `./scripts/run-static-verify.sh` exit 0 (gofmt/go vet/golangci-lint/staticcheck all clean — see evidence log). `git diff main...HEAD --name-only` confirms **zero** touches to `scripts/ralph-orchestrator.sh`, `scripts/ralph-pipeline.sh`, `internal/ui/`, `internal/state/`, `internal/action/`, `.claude/skills/` — matches the plan's "触らない" list. `go test ./...` itself is behavioral and delegated to `/test`. |
| AC-8 (all verbs support `--dry-run`, no real process start, `dry_run: true` recorded) | Verified with note | `spawn`, `send`, `stop`, `disband` all expose `--dry-run` and route through dry-run-only code paths that skip `Herdr`/`Agmsg` calls entirely (`spawn.go:273-321` `dryRunSpawn`, `verbs.go:57-63` `Send`, `verbs.go:193-194` `Stop`, `verbs.go:267` `Disband`). `wait`/`read`/`status` have **no** `--dry-run` flag — by design, since they are read-only/observational verbs with no external side effect to skip (not a gap against AC-8's intent, but note the AC's literal "全動詞" wording is satisfied only for the 4 mutating verbs). Delegated to `/test`: `TestOrgSpawn_DryRun_NoDriverCalls_EventsFlaggedAndExcludedByDefault`. |
| AC-9 (compat: no `[org]` section loads without warning; doctor exit code unaffected) | Verified (static) | `internal/config/config_test.go:454-491` `TestLoad_OrgMissingSection` backfills full `Default().Org` with no error. Doctor: see AC-5 evidence (info/warn never fail). Test execution delegated to `/test`. |
| AC-10 (saga failure injection → `spawn_failed` + compensation recorded, orphan traceable) | Verified (static) | `spawn.go:354-363` `failStep` records `spawn_failed` with `step=... error=... compensation=...` and preserves `paneID` on the event for traceability even when the saga aborts mid-flight. Injection points covered per self-review: workspace_create, tab_create, agent_start, agmsg_announce. Delegated to `/test`: `TestOrgSpawn_FailureInjection_WorkspaceCreate_NoCompensationAttempted`, `..._TabCreate_...`, `..._AgentStart_CompensatesExistingPane`, `..._AgmsgSend_CompensatesExistingPane`. |

### Self-review fix verification (commit `9bfe07e`)

| Finding (severity) | Fixed? | Evidence |
| --- | --- | --- |
| HIGH — herdr agent name not `org_id`-namespaced | Yes | New `herdrAgentName(orgID, seatID)` (`spawn.go:249-251`) used at all 3 call sites: `AgentStart` (`spawn.go:189`), `Send`'s `AgentWait` (`verbs.go:83`), `Wait`'s `AgentWait` (`verbs.go:138`). `WaitParams` gained `OrgID` (`verbs.go:117`), wired from the CLI's `--org-id` (`org.go:239`). New regression tests: `TestOrgSpawn_HerdrAgentNameNamespacedByOrgID`, `TestHerdrAgentName_NamespacesBySeatAndOrg`. |
| MEDIUM — dry-run events could override real seat state in `status --all` | Yes | `manifest.go`: `seatKey` and new `disbandKey` both gained a `DryRun bool` field, splitting the `latest`/`disbandedIdx` maps by the real/dry-run axis so a dry-run `stopped`/`disbanded` event can never become a real seat's latest state or deactivate it. New regression tests: `TestRoster_DryRunDisbandLeavesRealSeatsActive`, `TestRoster_RealDisbandLeavesDryRunSeatsUntouched`, `TestRoster_DryRunStoppedDoesNotDeactivateRealSeatWithSameID`. |
| MEDIUM — 6 stale "PR① is config-only / later slices" comments | Yes | All 6 sites reworded to describe the shipped end-state: `internal/config/config.go:94-96`, `templates/base/ralph.toml:62-63`, `scripts/ralph-config.sh:67-68` (+ template mirror), `internal/org/seat.go:1-9`, `internal/org/envelope.go:10-13`, `internal/org/receipts.go:8-13`. Spot-checked with `grep -n "config-only\|later slices\|Slice 3/4\|no cobra wiring"` — only the (accurate) `seat.go` "no cobra wiring" sentence remains, which correctly still describes package boundaries, not scope. |

Not asked to fix in this pass (self-review MEDIUM/LOW follow-ups, confirmed still open by design — not blocking this PR): `org stop` on an unknown seat still creates a phantom roster entry; `--timeout-ms` still covers the whole saga, not per-step; `reject`/`failStep`/`compensateStale` still discard `appendEvent`/`Receipts.Append` errors with `_ =` (in tension with `.claude/rules/golang.md`'s "do not silently swallow errors" — logged as tech debt, not a regression introduced by this PR); dead `ManifestRelPath`/`ReceiptsRelPath`/`NewManifestStore`/`NewReceiptStore`; rune-unsafe truncation; `send --dry-run` skipping seat-lookup validation; `max_seats` read-then-append race (already tracked in `docs/tech-debt/README.md`).

## Static analysis

| Command | Result | Notes |
| --- | --- | --- |
| `./scripts/run-static-verify.sh` (`HARNESS_VERIFY_MODE=static`, changed-language scope, fell back to full due to an unclassified shell diff) | Pass (exit 0) | gofmt clean, `go vet ./...` clean, `golangci-lint run ./...` → 0 issues, `staticcheck ./...` clean (no output = clean), settings.json JSON validity, Codex hook guards, `check-sync.sh`, `check-pipeline-sync.sh`, `check-skill-sync.sh` all OK. Full output: `docs/evidence/verify-2026-08-01-035137.log`. |
| `go build ./...` | Pass | Compiles cleanly (supplementary spot check beyond the wrapper). |
| `./scripts/check-sync.sh` | Pass | `DRIFTED: 0`, `ROOT_ONLY: 0` — root/templates parity holds, including the new `AGENTS.md` repo-map line and `ralph-config.sh` `[org]` block. |
| `./scripts/check-skill-sync.sh` | Pass | 13 skills in lock-step (this PR touches no skill bodies). |
| `cmp scripts/ralph-config.sh templates/base/scripts/ralph-config.sh` | Identical | Manual spot-check beyond the wrapper's own `check-sync.sh` run. |
| `cmp AGENTS.md templates/base/AGENTS.md` | Identical | Both carry the new `internal/org/` repo-map line verbatim. |

## Documentation drift

| Doc / contract | In sync? | Notes |
| --- | --- | --- |
| `AGENTS.md` repo map (`internal/org/` entry) | Yes | Added at line 61 in both root and `templates/base/` mirrors, byte-identical, references the spec path. |
| `docs/specs/2026-08-01-org-runtime.md` FR-9 (tri-state receipts) | Yes | Spec already specifies `honored: true|false|unknown` tri-state and "true only with driver-observed confirmation" — the plan's note that this "needs to be added to the spec" was already satisfied when the spec was authored; no drift, no action needed. |
| `docs/tech-debt/README.md` — herdr-namespace row | **Drifted (stale)** | The row added in commit `d3e36a7` (before the fix commit) describes the **pre-fix** state verbatim: "`internal/org/spawn.go` registers the herdr agent under the bare `seat_id`", "`org wait` requires `--org-id` and then discards it (`WaitParams` has no `OrgID` field)". Both claims are now false — commit `9bfe07e` fixed exactly this (see fix-verification table above). This row should be marked resolved (or removed) in the same way prior resolved rows in this file use a `~~strikethrough~~` + `RESOLVED` HTML-comment convention (see the "Cross-review base detection" row for the precedent). The other two tech-debt rows added alongside it (`max_seats` race, unverified agmsg CLI flag shape) remain accurate and unresolved — no drift there. |
| Plan `docs/plans/active/2026-08-01-org-runtime-mechanism.md` Progress checklist | Stale but expected | "Review artifact created" / "Verification artifact created" / "Test artifact created" / "PR created" are still unchecked — this is normal at this point in the pipeline (this report is the verification artifact being produced); not a defect. |
| Plan AC-1..AC-10 checkboxes / spec FR-1/FR-2/FR-9 checkboxes | Stale but expected | All still `[ ]` in the source files even though this report finds them met for PR①'s scope. Per project convention, plan/spec checkboxes lag implementation and are not a `/verify` blocker — flagging for `/sync-docs` or plan owner to tick, not treating as a failure. |
| `docs/quality/definition-of-done.md`, `README.md`, `.claude/rules/*` | No changes expected, none found | Plan's Non-goals correctly exclude the pipeline/skill/rules rewrite (that's PR⑤); confirmed no incidental edits landed in `.claude/rules/` or `docs/quality/`. |

## Observational checks

- `git diff main...HEAD --name-only` reviewed in full; confirmed zero touches to the plan's explicit "触らない" (do-not-touch) list: `scripts/ralph-orchestrator.sh`, `scripts/ralph-pipeline.sh`, `internal/ui/`, `internal/state/`, `internal/action/`, `.claude/skills/`.
- `internal/cli/root.go` diff is exactly the `+1` line self-review reported (`newOrgCmd()` registration) — no other wiring changes.
- Manually walked the spawn saga (`internal/org/spawn.go`) end to end against FR-9's required manifest fields (`org_id`, `seat_id`, `worktree`) — all present as required/omitempty JSON tags on `ManifestEvent` (`manifest.go:19-30`).
- Confirmed `HonoredTrue` is never assigned in production code paths (only `HonoredUnknown` in the real and dry-run spawn paths, and `HonoredFalse` in `reject`) — consistent with the plan's Non-goal that effective-model observation is deferred to PR②.

## Coverage gaps

- Everything marked "delegated to `/test`" above (behavioral proof of AC-1, AC-2, AC-3, AC-4 dry-run exclusion, AC-5 doctor checks, AC-6 test execution, AC-7 `go test ./...`, AC-8 dry-run, AC-9 compat test, AC-10 failure injection) is unknown at this point in the pipeline — this report only confirms the code paths and tests *exist* and read as internally consistent; it does not execute `go test ./...`.
- Two-org concurrency for the herdr-namespace fix is proven only by a synthetic unit test (`TestOrgSpawn_HerdrAgentNameNamespacedByOrgID`) with a fake `HerdrClient` — no live herdr binary exercised (expected, per plan Assumptions: CI has no herdr/agmsg).
- `agmsgArgs` CLI flag shape (`--team`/`--as`) remains an unverified assumption against the real agmsg CLI (tracked in tech debt, not a gap introduced by this review).

## Verdict

**Pass**, with one documentation-drift item to correct (stale tech-debt row) before/alongside `/sync-docs`.

- Verified: AC-1, AC-2, AC-3, AC-4, AC-5, AC-6 (config wiring), AC-8 (with the wait/read/status dry-run-flag note), AC-9, AC-10 — all at the static/code-reading level; all three self-review fixes (herdr namespace, dry-run roster isolation, stale comments) confirmed present in commit `9bfe07e`. Static analysis (`run-static-verify.sh`, `check-sync.sh`, `check-skill-sync.sh`) all green.
- Partially verified: AC-7 (static half green; `go test ./...` execution itself not run here — that is `/test`'s scope).
- Not verified (by design, `/verify` does not run tests): behavioral correctness of every test listed under "delegated to `/test`" above.
- Drift found: `docs/tech-debt/README.md`'s herdr-namespace row is now stale (describes a bug that commit `9bfe07e` already fixed) and should be marked resolved.
- Smallest next check that would most increase confidence: run `go test ./internal/org/... ./internal/cli/... ./internal/config/...` (this is `/test`'s job) — that closes the AC-7/AC-1..AC-10 behavioral gap directly.

---

## Cycle 2 (fix-and-revalidate re-run)

- Date: 2026-08-02
- Verifier: `verifier` subagent (Claude Code, `/verify`), pipeline cycle 2 of 2 (`RALPH_STANDARD_MAX_PIPELINE_CYCLES` default)
- Scope: the delta since the cycle-1 reviewed state (`git diff 9bfe07e...HEAD`). Cycle-1 content above is left intact. Code delta is exactly two fix commits — `4dcfc03` (fix: return idempotent spawn before envelope validation) and `e6a162c` (fix: reject stateless envelope violations before stale compensation) — the four intervening commits (`4820cd5`, `365f1ff`, `2e0364c`, `a7f36db`, `9d02dbc`, `6980dd7`, `b6fd875`) are report/insight/docs artifacts only, confirmed by `git diff --stat 9bfe07e...HEAD -- internal/ scripts/ templates/` (7 files: `internal/org/{driver/driver,driver/herdr,driver/probe,envelope,manifest,spawn,spawn_test}.go`).
- Evidence: `docs/evidence/verify-2026-08-02-053222.log`

### AC-3 re-verified against the new spawn ordering

`e6a162c` reorders `(*Org).Spawn` (`internal/org/spawn.go:110-207`) to, in order: (1) idempotent early return, (2) stateless envelope validation (`ValidateSpawnEnvelope` — driver/model pool membership, role restriction; a pure function of `cfg`+`req`), (3) stale-in-flight compensation, (4) capacity validation (`ValidateSpawnCapacity`, depends on the recomputed `activeSeats`), then the saga side effects. `ValidateSpawn` is preserved unchanged as `ValidateSpawnEnvelope` + `ValidateSpawnCapacity` composed (`internal/org/envelope.go:30-72`) and is still the sole path used by the dry-run branch (`spawn.go:133-137`, unchanged).

- **AC-3 (idempotent respawn) still holds**: the idempotent check (`spawn.go:172-181`, "an already-spawned seat returns the existing seat with no validation attempted at all") runs first, before either envelope or capacity checks — unchanged from cycle 1. The stale-in-flight branch (`spawn.go:194-203`) now runs after the stateless envelope check but still before the capacity check, matching the plan's stated intent that compensation must happen before `activeSeats` is recomputed. Confirmed no new driver calls are made on the idempotent path (unchanged code) and confirmed by reading `TestOrgSpawn_Idempotent_AlreadySpawned_NoNewDriverCalls` / `TestOrgSpawn_StaleInFlight_CompensatesThenRespawnsFresh` (unmodified in this delta — `git diff` shows only additions to `spawn_test.go`, no edits to the existing tests).
- **New ordering guarantee added by this fix**: a stateless-envelope-invalid request (bad driver/model/role) against a seat with a *stale* in-flight record is now rejected with **zero driver calls** — no `C-c` compensation, no `spawn_failed` write — instead of the pre-fix behavior where `4dcfc03` had moved the *entire* `ValidateSpawn` (including the stateless checks) after compensation, so an envelope-invalid respawn attempt would trigger a destructive compensation side effect on an already-doomed request. New regression test `TestOrgSpawn_StaleInFlight_StatelessEnvelopeViolation_RejectedBeforeCompensation` (`spawn_test.go`, added in `e6a162c`) asserts exactly this: `SpawnOutcomeRejected`, zero herdr/agmsg calls, zero `PaneSendKeys` calls, and the event sequence is `[spawn_started (seeded), rejected]` — no `spawn_failed` interposed. Ran this test directly as a targeted compile+behavioral sanity spot check (not a substitute for `/test`'s full run): `go test ./internal/org/... -run TestOrgSpawn_StaleInFlight_StatelessEnvelopeViolation_RejectedBeforeCompensation -v` → PASS.
- **`reject()`'s doc comment is now accurate on this path**: cycle-2 self-review's MEDIUM finding was that `reject()`'s claim "no external side effect was ever attempted" was false when an envelope-invalid request hit a stale seat (because the entire `ValidateSpawn` — capacity included — ran after compensation). With the split, only `ValidateSpawnCapacity` runs after compensation; `ValidateSpawnEnvelope` runs before any side effect, so a stateless rejection is once again a guaranteed no-op. Verified by code reading (`spawn.go:181-191`, comment naming this exact invariant) plus the new regression test above.

### Other ACs re-checked for regressions from the two fix commits

| AC | Still holds? | Evidence |
| --- | --- | --- |
| AC-1 (reject → non-zero exit, manifest + receipt with `honored: false`) | Yes, unaffected | `reject()` (`spawn.go:303-314`) is unchanged by either fix commit — both the envelope-check call site and the capacity-check call site route through the same unmodified `o.reject(p, err)`. `git diff 9bfe07e...HEAD -- internal/org/spawn.go` shows `reject`'s body untouched. |
| AC-2 (`max_seats` org-isolated) | Yes, unaffected | `ValidateSpawnCapacity` (`envelope.go:63-72`) is a verbatim extraction of the pre-fix `max_seats` check from `ValidateSpawn` — same comparison (`activeSeats >= cfg.MaxSeats`), same error message, still called with `activeSeats := ActiveSeatCount(events, p.OrgID, RosterOptions{})` scoped to `p.OrgID` (`spawn.go:205-207`, unchanged). |
| AC-10 (saga failure injection → `spawn_failed` + compensation recorded) | Yes, unaffected | `failStep`/`compensateStale` bodies are untouched by this delta (only a one-word comment edit in `compensateStale`'s call site, `spawn.go:197`: "envelope validation below" → "capacity check below"). The saga side-effect sequence (workspace → tab → agent → agmsg, each wrapped in `failStep` on error) starts only after both `ValidateSpawnEnvelope` and `ValidateSpawnCapacity` pass — same as cycle-1's verified saga entry point, just reached via two checks instead of one. |
| AC-8 (`--dry-run`, no real process start) | Yes, unaffected | Dry-run branch (`spawn.go:133-137`) still calls the composed `ValidateSpawn(o.Config, req, activeSeats)` unchanged, then `o.dryRunSpawn(p)` — neither touched by this delta. |
| AC-9 (compat: missing `[org]` section, doctor exit code) | Yes, unaffected | Neither fix commit touches `internal/config/` or `internal/cli/doctor.go`; `git diff --stat 9bfe07e...HEAD` confirms zero changes outside `internal/org/`. |

### Cycle-2 self-review MEDIUM fix verification

| Finding (severity, from self-review Cycle 2 addendum) | Fixed? | Evidence |
| --- | --- | --- |
| MEDIUM — `4dcfc03` moved *all* of `ValidateSpawn` (not just the capacity check) after stale-in-flight compensation, so an envelope-invalid respawn against a stale seat triggered destructive compensation before rejection | Yes | `e6a162c` splits `ValidateSpawn` into `ValidateSpawnEnvelope` (stateless, run before compensation) and `ValidateSpawnCapacity` (stated-dependent, run after) — see AC-3 section above. New regression test asserts zero driver calls on this path. |
| MEDIUM — 5 remaining stale phase-reference comments (`manifest.go:16`, `driver/driver.go:3`, `driver/herdr.go:9,12`, `driver/probe.go:16`) still described shipped functionality as deferred to a future slice | Yes, all 5 fixed | Diff-verified each site rewords to the shipped end-state (e.g. `driver.go`: "the `ralph org` verbs, Slice 4" → "the `ralph org` verbs in internal/org, spawn.go and verbs.go"; `herdr.go`: "Seat termination strategy (send-keys based) is a Slice 4 concern" → names the actual caller `(*Org).Stop` in `verbs.go`). Repo-wide grep `grep -rn -E "PR.?①|later slice|Slice [0-9]|config-only" internal/ scripts/ralph-config.sh templates/base/` now returns only the two accurate PR①/PR④ budget-field notes in `config.go:116,120` and `templates/base/ralph.toml:79` that both the cycle-1 and cycle-2 self-review explicitly identified as correct-as-is (nothing reads those two fields beyond `Load()` validation). No remaining stale references. |

### Static analysis (full scope re-run)

| Command | Result | Notes |
| --- | --- | --- |
| `RALPH_VERIFY_SCOPE=full ./scripts/run-static-verify.sh` | Pass (exit 0) | gofmt clean, `go vet ./...` clean (silent success — binary present, no "skipping" line emitted), `golangci-lint run ./...` → `0 issues.`, `staticcheck ./...` clean (silent success — `command -v staticcheck` confirms the binary is present so the script's own `command -v` guard did not skip it), `jq -e` on both `settings.json` mirrors, Codex hook guards, `check-sync.sh` (`DRIFTED: 0`, `ROOT_ONLY: 0`), `check-pipeline-sync.sh`, `check-skill-sync.sh` (13 skills in lock-step) all OK. Full output: `docs/evidence/verify-2026-08-02-053222.log`. |
| `go build ./...` | Pass | Compiles cleanly, exit 0. |
| `git diff main...HEAD --name-only` vs. plan's "触らない" list | Pass | Still zero touches to `scripts/ralph-orchestrator.sh`, `scripts/ralph-pipeline.sh`, `internal/ui/`, `internal/state/`, `internal/action/`, `.claude/skills/` — unchanged from cycle 1; the cycle-2 delta only adds files under `internal/org/`. |

### Documentation drift (cycle-2 delta)

| Doc / contract | In sync? | Notes |
| --- | --- | --- |
| `docs/tech-debt/README.md` herdr-namespace row | Yes, already resolved | The stale-row drift flagged in the cycle-1 verdict was fixed in commit `4820cd5` (`docs: sync-docs for org-runtime-mechanism`, predates this cycle-2 delta) — the row now carries a `RESOLVED 2026-08-01 in commit 9bfe07e` HTML comment and `~~strikethrough~~` per the repo's established convention. No action needed here. |
| Plan `docs/plans/active/2026-08-01-org-runtime-mechanism.md` — "Implementation notes (deviations from initial outline)" | **Drifted (minor, non-blocking)** | The section does not mention either cycle-2 fix commit (`4dcfc03`, `e6a162c`) or the resulting 4-step spawn ordering (idempotent → stateless envelope → compensation → capacity). Per `.claude/rules/planning.md` ("record meaningful deviations from the original plan instead of silently drifting"), this ordering refinement is exactly the kind of deviation the section exists to capture. Not a `/verify` blocker (this is `/sync-docs` or plan-owner territory), but flagging so it isn't missed before archive. |
| Plan AC-1..AC-10 / Progress checklist checkboxes | Stale but expected (same as cycle 1) | Still unticked; per established project convention this lags implementation and is not treated as a `/verify` failure. |
| `docs/specs/2026-08-01-org-runtime.md` | No changes expected, none found | Neither fix commit touches spawn's external contract (CLI flags, manifest field names, exit codes) — only internal validation ordering — so no spec update is implied. |

### Verdict — Cycle 2

**Pass.**

- Verified: AC-3 re-confirmed against the new 4-step spawn ordering (idempotent → stateless envelope → stale compensation → capacity), with the specific regression the reorder was designed to prevent (destructive compensation on an envelope-invalid stale respawn) covered by a new, passing regression test. AC-1, AC-2, AC-8, AC-9, AC-10 re-checked for collateral regressions from the two fix commits — none found; all four go through code paths this delta did not touch or extracted verbatim. Both cycle-2 self-review MEDIUMs (validation-ordering split, 5 remaining stale comments) confirmed fixed by diff reading and repo-wide grep. Static analysis at full scope (`run-static-verify.sh`, `check-sync.sh`, `check-skill-sync.sh`, `go build ./...`) all green.
- Partially verified: same AC-7 caveat as cycle 1 — full `go test ./...` execution remains `/test`'s scope; this report ran exactly one targeted existing test (`TestOrgSpawn_StaleInFlight_StatelessEnvelopeViolation_RejectedBeforeCompensation`) as a spot check, not as a substitute for the tester's full behavioral pass.
- Not verified (by design): behavioral correctness of the full suite, including the three paired tests self-review cited for `4dcfc03` (`..._AtMaxSeats_RespawnSucceedsInsteadOfRejected`, `..._NewSeatStillRejected`, and the companion stale-seat case).
- Drift found: one minor, non-blocking item — the plan's "Implementation notes" section doesn't yet record the cycle-2 ordering fix as a deviation. Suggest `/sync-docs` (or the plan owner, before archive) add one line.
- Smallest next check that would most increase confidence: run `go test ./internal/org/... ./internal/cli/... ./internal/config/...` (full suite, `/test`'s job) to close the same AC-1..AC-10 behavioral gap noted in cycle 1, now including the two new regression tests from this delta.
