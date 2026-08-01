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
