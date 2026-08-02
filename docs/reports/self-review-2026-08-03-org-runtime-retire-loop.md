# Self-review report: org-runtime-retire-loop

- Date: 2026-08-03
- Plan: `docs/plans/active/2026-08-03-org-runtime-retire-loop.md`
- Reviewer: `reviewer` subagent (self-review, diff quality only)
- Scope: `git diff d074838..1925569` on `refactor/org-runtime-retire-loop` (213 files, +1,812 / −30,497)

## Evidence reviewed

- `git diff --stat d074838..HEAD` and per-file diffs for every non-pure-deletion path.
- Full read of the additions: `scripts/xreview-helpers.sh`, `internal/cli/status.go`,
  `internal/cli/status_test.go`, `tests/test-no-loop-references.sh`,
  `internal/cli/insights_test.go` (new test), `internal/org/watch.go` gap fixes.
- Diffs of the edited-not-deleted surfaces: `internal/config/config.go`,
  `internal/config/defaults_sync_test.go`, `internal/cli/doctor.go`,
  `internal/cli/root.go`, `scripts/ralph-config.sh`, `scripts/verify.local.sh`,
  `scripts/check-*.sh`, `go.mod`/`go.sum`, `README.md`,
  `docs/quality/quality-gates.md`, `docs/tech-debt/README.md`,
  `.claude/skills/cross-review/SKILL.md` (+3 mirrors), `.claude/rules/*`.
- Deletion-completeness sweeps beyond the guard test's own pattern:
  `ralph-cli-driver | ralph-status-helpers | ralph-loop | new-ralph-plan | build-tui |
  ralph-tui | internal/{state,ui,action,watcher} | RALPH_*_MODEL | run_agent |
  resolve_phase_model | write_model_receipt | ralph run | ralph retry | ralph abort`
  over `*.sh *.go *.toml *.md *.yml`, excluding the historical dirs.
- Mirror discipline: `cmp scripts/xreview-helpers.sh templates/base/scripts/xreview-helpers.sh`
  (identical), `git ls-files --stage` (100755 on both sides).
- Consumer checks: `grep` for every function extracted into `xreview-helpers.sh`;
  `grep` for `tests/test-*.sh` runner wiring in `scripts/verify.local.sh`.
- Explicitly NOT run (out of self-review scope): tests, linters, static analysis,
  spec-compliance verification, doc-drift sweeps.

## Findings

| Severity | Area | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- |
| HIGH | maintainability | `ralph doctor`'s Check 9 `checkStaleOrchestratorState` survives untouched. It reads `.harness/state/orchestrator/orchestrator.json` — a file only the deleted `scripts/ralph-orchestrator.sh` ever wrote — so the check can never fire again, and its warn text instructs the operator to `run './scripts/ralph abort'`, a script deleted in this same diff. Dead check plus a user-facing instruction pointing at a removed binary. | `internal/cli/doctor.go:88-89` (still wired), `:459-517` (body), `:512` (`"a crashed run may have left it behind — run './scripts/ralph abort' or "`); `internal/cli/doctor_stale_state_test.go` still pins it | Remove `checkStaleOrchestratorState`, its call site, and `doctor_stale_state_test.go` with the rest of the orchestrator surface. If the check is kept intentionally as a cleanup aid for pre-upgrade repos, drop the `./scripts/ralph abort` half of the remediation and say only "delete the file". |
| MEDIUM | maintainability | The new guard test's `PATTERN` (`ralph-orchestrator\|ralph-pipeline\|RALPH_LOOP_DRIVER`) is much narrower than the contract its own header states ("zero live references to the retired Ralph Loop autonomous execution system"). Four live, non-excluded files reference retired surfaces by other names and pass the guard. | `tests/test-no-loop-references.sh:20`; misses `internal/cli/doctor.go` (orchestrator state, HIGH above), `internal/org/watcher.go:116,226-227` (doc comments citing `scripts/ralph-cli-driver.sh` / `_run_agent_claude`), `docs/research/approach-comparison.md:34,99-100`, `.codex/README.md:58` | Widen `PATTERN` to also cover `ralph-cli-driver`, `ralph-loop`, `ralph-status-helpers`, `new-ralph-plan`, `build-tui`, `ralph-tui`, and `RALPH_(FORCE|IMPLEMENT|SELF_REVIEW|PROBE|ESCALATION)_MODEL`, then fix whatever it surfaces. If the narrow pattern is deliberate, narrow the header comment to match so the guard does not over-promise. |
| MEDIUM | maintainability | `count_triage_findings` was carried into `scripts/xreview-helpers.sh` but has **no executable consumer**: its only caller was the deleted `ralph-pipeline.sh`. `/cross-review`'s SKILL.md never invokes it (triage counts are produced inline by the model). `pick_reviewer` is in the same position — SKILL.md Step 2 does reviewer auto-detect in prose and only *mentions* `pick_reviewer` descriptively at line 159. Only `detect_base_branch` is actually invoked (Step 4). Two places assert the opposite. | `scripts/xreview-helpers.sh:9-16` ("`/cross-review` is the only consumer of these three functions"); `.claude/skills/cross-review/SKILL.md:38-42` (prose detect), `:51` (only `detect_base_branch` sourced), `:159`; `docs/tech-debt/README.md:28` ("the only surviving caller is `/cross-review`'s auto-detect (step 2 …)") — step 2 makes no such call | Either make SKILL.md Step 2/5 actually source `xreview-helpers.sh` and call `pick_reviewer` / `count_triage_findings` (the grep-able-contract direction `.claude/rules/architecture.md` favours), or drop `count_triage_findings` + its ~110 lines of tests and correct the two claims. Do not leave a helper whose header misstates its consumer. |
| MEDIUM | maintainability | `.claude/rules/model-routing.md` "Where the values live" (edited in this diff) still claims `scripts/ralph-config.sh` holds `RALPH_MODEL` / `RALPH_EFFORT` and that `ralph.toml` mirrors them. This diff deleted both variables from `ralph-config.sh`, deleted `[pipeline]` from `templates/base/ralph.toml`, deleted `Pipeline` from `config.Default()`, and deleted their lock-step checks from `defaults_sync_test.go`. The final bullet's "tripwire that fails if the three surfaces above drift" is now false for those two names. | `.claude/rules/model-routing.md:95-97` and `templates/base/.claude/rules/model-routing.md:94-95`; `git diff … scripts/ralph-config.sh` (RALPH_MODEL/RALPH_EFFORT removed); `templates/base/ralph.toml` (no `[pipeline]`); `internal/config/defaults_sync_test.go` (`pipeline.model`/`pipeline.effort` checks removed) | Reduce the bullets to the two surviving values (`RALPH_CLAUDE_REVIEWER_MODEL`, `RALPH_STANDARD_MAX_PIPELINE_CYCLES`) and drop the `ralph.toml` mirror bullet, or restore whatever is meant to survive. `RALPH_MODEL` currently exists in no shipped file. |
| MEDIUM | exception-handling | The rewritten `ralph status` silently discards `rr.CorruptLines` on the empty path. A manifest whose every line is corrupt yields `len(groups)==0`, so the operator sees "org runtime state が見つかりません … ralph org spawn で開始してください" and no warning at all — a data-integrity signal reported as "nothing here yet". The pre-existing `ralph org status` prints the corrupt warning regardless of seat count. | `internal/cli/status.go:74-84` (`printStatusEmpty(cmd, stateDir, filterOrgID, jsonOut)` — `rr.CorruptLines` not passed), `:131-150`; compare `internal/cli/org.go:566-569` where the warning is outside the `len==0` branch | Pass `rr.CorruptLines` into `printStatusEmpty` and emit the same `warning: N corrupt manifest line(s) skipped` line (and `corrupt_lines` in the JSON payload). Add a regression case to `status_test.go`. |
| MEDIUM | maintainability | `ralph status --json` emits two different schemas depending on emptiness: the empty payload carries `org_id` and omits `corrupt_lines`; the populated payload carries `corrupt_lines` and omits `org_id`. A machine consumer has to branch on which one it got. | `internal/cli/status.go:134-138` vs `:189-193` | Use one struct for both paths (`state_dir`, `org_id,omitempty`, `orgs`, `corrupt_lines,omitempty`) so `--json` has a single stable shape. |
| MEDIUM | unnecessary-change | `.claude/skills/cross-review/SKILL.md` (and its 3 mirrors) was edited in this diff but still tells the reader `RALPH_CLAUDE_REVIEWER_MODEL` is "exported by `ralph run`" — the `ralph run` subcommand was removed in this same diff (`root.go`, `run.go`, `run_env_test.go` all deleted). | `.claude/skills/cross-review/SKILL.md:55`, `.agents/skills/cross-review/SKILL.md:54`, both `templates/base/` mirrors; `internal/cli/root.go` diff (`newRunCmd()` removed) | Drop the `ralph run` clause; the surviving source is `scripts/ralph-config.sh` (or a plain env export). Regenerate the `.agents` mirror with `scripts/sync-skills.sh` after the edit. |
| MEDIUM | maintainability | Two tech-debt rows this PR invalidated were left open while five others were struck through. (a) "`ralph status --json` outputs plain text, not JSON" is now false — the rewrite emits real JSON via `json.Encoder`. (b) "Phase 6b: Go native pipeline migration pending … parity tests must pass before shell scripts are removed" — the shell scripts were removed outright with no Go migration, so the row's premise no longer exists. | `docs/tech-debt/README.md:18` and `:20` (unchanged in the diff); `internal/cli/status.go:184-197` | Strike through both rows with a `RESOLVED 2026-08-03 in refactor/org-runtime-retire-loop` annotation in the same style used for the five rows this PR did close, or state explicitly why they remain open. |
| MEDIUM | maintainability | Two live docs still advertise removed features as shipped. `docs/research/approach-comparison.md` lists `/loop` skill + `ralph-loop.sh` orchestrator in its present-tense "Borrowed into ralph" column and describes "Autonomous iteration via Ralph Loop" as a current capability. `.codex/README.md` (+ template mirror, a shipped downstream file) documents `RALPH_PERMISSION_MODE`, `ralph.toml [pipeline] permission_mode`, and "toml only honoured via `ralph run`" — all three deleted here. | `docs/research/approach-comparison.md:34,99-100`; `.codex/README.md:58` and `templates/base/.codex/README.md:58` | Update both (the research doc's "Borrowed into ralph" cell should now read `org runtime` / historical). The `.codex/README.md` row should be replaced by the `[org.permissions]` mapping the README table already switched to. |
| LOW | maintainability | `internal/cli/status.go` is the only file in `internal/cli` (and `cmd/`) with Japanese in **user-facing output** rather than comments. The sibling `ralph org status` prints `no seats` in English, and this new message mixes languages within two consecutive lines ("… で開始してください。" then "Run `ralph doctor` to check environment readiness."). | `internal/cli/status.go:144,146` vs `internal/cli/org.go:553`; `grep -lP '[\x{3040}-\x{30ff}]' internal/ cmd/` shows Japanese elsewhere only in comments/prompt templates | Pick one language for CLI output. Given every other command is English, English is the lower-surprise choice; if Japanese is intentional, make the `ralph doctor` hint Japanese too. |
| LOW | readability | Removing "Check 7: Loop driver" left the numbered comment sequence in `runDoctorOpts` reading 6 → 8 → 9. | `internal/cli/doctor.go:82-89` | Renumber, or drop the numbers (they carry no meaning once the list is not contiguous). |
| LOW | readability | `checkDeadman`'s new doc block says the unavailability sentinels are documented "per each field's own doc comment", but `watchPendingAlert.LeadAgentGet` and `.HistoryLeadLines` have no per-field doc comments — the sentinel contract lives on the producers `leadProbeSnapshot` / `historyLeadLineCount`. Sends a reader to the wrong place. | `internal/org/watch.go:1024` vs `:176-183` (bare struct fields), `:900-903` and `:915-919` (where the sentinels are actually documented) | Reword to "per `leadProbeSnapshot` / `historyLeadLineCount`'s doc comments", or add the one-line sentinel note to the two struct fields. |
| LOW | naming | Within the new `xreview-helpers.sh`, `detect_base_branch` and `pick_reviewer` namespace their sourced-into-caller globals (`_dbb_*`, `_pr_*`) but `count_triage_findings` uses bare `_file`, `_category`, `_summary`, `_n`. `_file` is a plausible collision for any script that sources this library. (Verbatim carry-over from `ralph-cli-driver.sh` — extraction was the natural moment to fix it.) | `scripts/xreview-helpers.sh:81-95`; `git show d074838:scripts/ralph-cli-driver.sh` confirms the bodies are byte-identical | Prefix with `_ctf_` for consistency with the two siblings. |
| LOW | typo | README's directory tree lost column alignment for the two lines it touched: `config/` and `insights/` each carry one extra space before `#`. | `README.md` "├── config/                # ralph.toml parser" and "└── insights/              # Insight event aggregation + backfill" vs the aligned `cli/` / `scaffold/` / `upgrade/` / `org/` lines | Re-align to the surrounding column. |
| LOW | maintainability | `scripts/ralph-config.sh`'s new header says `/cross-review` "sources this file (or reads the exported env)" for the pipeline cycle cap, but `RALPH_STANDARD_MAX_PIPELINE_CYCLES` is deliberately absent from the `export` list — only sourcing works for it. | `scripts/ralph-config.sh:6-11` vs `:89` (`export RALPH_CLAUDE_REVIEWER_MODEL` only) | Say "sources this file (the reviewer model is also exported; the cycle cap is not)". |
| LOW | maintainability | The guard test hardcodes this plan's exact filename in `EXCLUDE_REGEX`. `/pr` archives the plan into `docs/plans/archive/` (already excluded), so the clause becomes permanently dead but stays in the regex. | `tests/test-no-loop-references.sh:40` | Drop the plan-specific clause once the plan is archived, or replace it with a `docs/plans/active/` wildcard. |

## Positive notes

- **The two PR④ gap fixes are correct and minimal.** `ensureWatchdogJoined` now gates
  `status.WatchdogJoined = true` on `Join` returning nil, closing the "one bad cycle
  permanently disables ALERT delivery" ratchet; `checkDeadman` now requires a valid
  ALERT-time baseline (`pending.LeadAgentGet != ""` / `pending.HistoryLeadLines >= 0`)
  before letting probe recovery count as lead activity. Both carry doc comments that
  explain the failure mode and cite the originating finding, and both leave the
  unaffected clearing sources intact.
- **The `xreview-helpers.sh` extraction is a true pure move.** Diffing against
  `git show d074838:scripts/ralph-cli-driver.sh` shows all three function bodies carried
  over byte-identical — no behavior smuggled into a "refactor" commit. Root and
  `templates/base/` copies are `cmp`-identical and both `100755`.
- **`status.go` matches the conventions of the code it sits beside.** Column headers,
  `state` suffixing (`(active)` / `[dry-run]`), the `_, _ = fmt.Fprintf` idiom, the
  discarded `source` from `ResolveOrgStateDir`, and the "declare a local read-only
  mirror struct rather than export internal/org's private JSON shape" decision all
  mirror `org.go`, and the mirror-struct choice is explained in a comment.
- **Test wiring is real, not nominal.** Both new shell suites are picked up by
  `scripts/verify.local.sh`'s `for f in tests/test-*.sh` glob (enumeration-drift-proof by
  design), so AC-5's "pin it in CI" holds without touching a hand-maintained list.
- **Dependency hygiene followed the deletion.** `go.mod` correctly demotes
  `bubbles`/`bubbletea`/`lipgloss` to `// indirect` (still reachable via `huh`) and
  `go.sum` drops `fsnotify` outright rather than leaving orphaned direct requires.
- **No secrets, no debug leftovers, no commented-out code** anywhere in the additions;
  no injection-shaped string building in the new shell or Go.

## Tech debt identified

| Debt item | Impact | Why deferred | Trigger to pay down | Related plan/report |
| --- | --- | --- | --- | --- |
| `scripts/xreview-helpers.sh`'s `count_triage_findings` and `pick_reviewer` have no executable caller — `/cross-review`'s SKILL.md does reviewer detection and triage counting inline, so only `detect_base_branch` is actually sourced. The file header and `docs/tech-debt/README.md:28` both state `/cross-review` calls them. | A future editor reading either claim will assume the SKILL.md path is covered by `tests/test-xreview-helpers.sh` when it is not; ~110 lines of tests defend an unreachable helper | Wiring SKILL.md to source the helpers is a behavior change to `/cross-review`, outside the retirement PR's "move, do not modify" contract for these functions | Next edit to `/cross-review` Step 2 or Step 5, or the next time the triage report template's column layout changes (which the awk fallback is already coupled to, per row 29) | This report; `scripts/xreview-helpers.sh:9-16`, `.claude/skills/cross-review/SKILL.md:38-42,159`, `docs/tech-debt/README.md:28-29` |
| `tests/test-no-loop-references.sh`'s `PATTERN` covers 3 tokens while its header claims to guard the whole retired Ralph Loop surface; retired names such as `ralph-cli-driver`, `ralph-loop`, `ralph-tui`, `build-tui`, `new-ralph-plan` and the per-phase `RALPH_*_MODEL` knobs are unguarded | The guard reads as a complete AC-5 ratchet but lets new dangling references to retired surfaces land silently; four already exist at merge time | Widening the pattern surfaces additional cleanup (doctor.go, `internal/org/watcher.go` comments, `docs/research/`, `.codex/README.md`) that is larger than the guard change itself | When any of the four current leftovers is fixed — widen the pattern in the same commit so it cannot regress | This report; `tests/test-no-loop-references.sh:20,40` |

_(Both rows appended to `docs/tech-debt/README.md`.)_

## Recommendation

- **Merge: conditional.** No CRITICAL findings; the deletion itself is coherent, the
  additions are well-commented, and the two watchdog gap fixes are sound.
  - Fix before merge: **HIGH #1** (`checkStaleOrchestratorState` — dead check whose
    user-facing remediation names a script this PR deletes). It is a small, contained
    removal and it is the one leftover that actively misinforms an operator.
  - Strongly recommended in the same pass (all one-line doc/text edits with no risk):
    the stale `ralph run` clause in the 4 cross-review SKILL.md copies, the
    `RALPH_MODEL`/`RALPH_EFFORT` bullets in `model-routing.md` ×2, and the two
    unclosed tech-debt rows.
- **Follow-ups:** widen the guard test's pattern and fix what it surfaces
  (`internal/org/watcher.go` comments, `docs/research/approach-comparison.md`,
  `.codex/README.md` ×2); pass `CorruptLines` through `printStatusEmpty` and unify the
  `--json` schema; decide the fate of `count_triage_findings`; settle the CLI output
  language for `ralph status`.
- **Known gaps in this review:** scope was diff quality only — spec compliance
  (AC-1…AC-8), test adequacy for the new `status` surface, and repo-wide documentation
  drift were deliberately not assessed here and belong to `/verify` and `/test`. In
  particular, whether the `ralph upgrade` remove-path actually retires the deleted
  templates downstream (plan risk row 3) was not exercised.

---

# Cycle 2 — fix-and-revalidate re-review

- Date: 2026-08-03
- Scope: `git diff 61006d5..fea2ee6` (the delta since the cycle-1 report), i.e.
  the findings-fix commit `746b35b`, the insight/verify/test report commits
  (`a046b41`, `ccd03c3`, `e0f4446`, `f41bef5`), the sync-docs commit `a868fca`,
  the cross-review triage report `7c74dc8`, and the cross-review fix `fea2ee6`.
  31 files, +793 / −325. The ~30k-line deletion reviewed in cycle 1 was **not**
  re-reviewed from scratch.

## Cycle-1 findings: disposition

| Cycle-1 finding | Status | Evidence |
| --- | --- | --- |
| HIGH — dead `checkStaleOrchestratorState` naming a deleted script | **Closed** | `746b35b` removes the body, the call site, and `doctor_stale_state_test.go`; Check numbering renumbered 6→7→8→9→10→11 including the `runDoctorOpts` doc comment. No orphaned imports (`errors`/`io/fs`/`time`/`encoding/json` all still used at `doctor.go:59,158,221,268,300,344,417,584`). |
| MEDIUM — guard `PATTERN` narrower than its header | **Partially closed** — see C2-5 | `PATTERN` widened to 11 tokens and all four named leftovers fixed, but the header's "per-phase `RALPH_*_MODEL` knobs" still over-claims by 4 knobs. |
| MEDIUM — `count_triage_findings`/`pick_reviewer` have no executable consumer | **Closed** | SKILL.md Step 2.b now does `. scripts/xreview-helpers.sh; REVIEWER=$(pick_reviewer "$DRIVER")`; new Step 6 count-verification block calls `count_triage_findings` for all three categories. The grep-able-contract direction `.claude/rules/architecture.md` favours, not the deletion direction. |
| MEDIUM — `model-routing.md` names `RALPH_MODEL`/`RALPH_EFFORT`/`ralph.toml` mirror | **Closed** | Reduced to the two surviving values + the `defaults_sync_test.go` bullet, in both root and `templates/base/` copies. |
| MEDIUM — `printStatusEmpty` discards `rr.CorruptLines` | **Closed** | `corruptLines` threaded through; warning emitted on the empty path; regression test `TestStatusCmd_EmptyRosterFromFullyCorruptManifestStillWarns`. |
| MEDIUM — two `--json` schemas | **Closed** | Single `statusJSON` struct used by both paths (`status.go:199-211`); `TestStatusCmd_JSONSchemaIsIdenticalEmptyVsPopulated` added (but see C2-8 on its comment). |
| MEDIUM — stale `ralph run` clause in 4 SKILL.md copies | **Closed** | All four mirrors updated; `md5` confirms `.claude`↔`templates/base/.claude` identical and `.agents`↔`templates/base/.agents` identical. |
| MEDIUM — two tech-debt rows left open | **Closed** | Both struck through with `RESOLVED 2026-08-03` annotations in the established style, each with an HTML-comment rationale. |
| MEDIUM — `approach-comparison.md` / `.codex/README.md` advertise removed features | **Closed** (differently than recommended) | `.codex/README.md` row replaced with the `[org.permissions]` mapping (+ mirror `cmp`-identical). The research doc got a historical-document banner instead of a per-cell rewrite — a legitimate alternative for a dated research note, but see C2-10. |
| LOW — Japanese in `ralph status` user-facing output | **Closed** | Both messages converted to English; the test assertion and its failure message were updated with it. |
| LOW — doctor check numbering 6→8→9 | **Closed** | Renumbered contiguously. |
| LOW — `checkDeadman` doc points at the wrong place for sentinels | **Closed** | Reworded to name `leadProbeSnapshot`/`historyLeadLineCount` as "the producers of these values". |
| LOW — `count_triage_findings` bare `_file`/`_category`/`_n` globals | **Not addressed** | Still bare at `scripts/xreview-helpers.sh:86-95`, while the two siblings use `_dbb_*`/`_pr_*`. Carried forward as a nit, not re-raised as a numbered finding. |
| LOW — README directory-tree column drift | **Not addressed** | Still misaligned. Carried forward as a nit. |
| LOW — `ralph-config.sh` header over-claims the export list | **Closed** | Header now states explicitly that the cycle cap is not exported and only sourcing picks it up. |
| LOW — plan-specific clause hardcoded in `EXCLUDE_REGEX` | **Closed** | Replaced with a `docs/plans/active/` wildcard. |

## Cycle-2 findings

| ID | Severity | Area | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- | --- |
| C2-1 | MEDIUM | correctness | `fea2ee6`'s `IncludeDryRun: true` flip makes `ralph status`'s **aggregate** counts include dry-run seats. The per-row `[dry-run]` marker is present, but `active %d/%d` in the table header and `active_count`/`total_count` in `--json` carry no such split. The in-repo precedent does exactly the opposite: `internal/org/report.go` lists the roster with `IncludeDryRun: true` (`:136`) and computes "active seats" with `ActiveSeatCount(events, orgID, RosterOptions{})` (`:201`). Capacity enforcement agrees (`internal/org/spawn.go:333,788` — both `RosterOptions{}`). So after one `ralph org spawn --dry-run`, `ralph status` reports `active 3/3` while `max_seats` accounting and `ralph org report` both see 2, with nothing in the aggregate saying why. | `internal/cli/status.go:73` (flip), `:261-267` (table aggregate), `:226-237` (`buildStatusOrgJSON` aggregate); vs `internal/org/report.go:136` and `:201` | Keep the roster listing dry-run-inclusive, but compute the aggregate from real seats only (or emit `dry_run_count` alongside `active_count`) so the summary number matches the number that actually gates spawning. Assert it in `status_test.go`. |
| C2-2 | MEDIUM | maintainability | The flip's rationale was documented at the new call site but not at the option's definition, so the contract comment is now false. `RosterOptions.IncludeDryRun` still says the dry-run audit trail "stays visible only via `status --all` (AC-8, dry-run audit separation)", and `disbandKey` still says "the moment `IncludeDryRun` is true (`status --all`)". The top-level `ralph status` has no `--all` flag and now sets it unconditionally. | `internal/org/manifest.go:110-118` and `:135-140` (both unchanged in this PR) vs `internal/cli/status.go:63-73` | Update the doc at the definition to name both consumers (`ralph org status --all`, and the top-level `ralph status` unconditionally), since that comment is what a future reader of `RosterOptions` will trust. |
| C2-3 | MEDIUM | maintainability | The AR-1 fix removed the last caller of `org.NewManifestStore(root)` but left the exported root-relative constructor — and `org.ManifestRelPath` — in the API with zero callers repo-wide (only three doc comments now mention them). That constructor *is* the AR-1 footgun: the next person who reaches for the obvious-looking `NewManifestStore(stateDir)` re-creates the double-join. `org.NewReceiptStore(root)`/`org.ReceiptsRelPath` are in the same zero-caller position. | `grep -rn "NewManifestStore(" --include="*.go" .` → only `internal/org/manifest.go:56` (definition) plus comment mentions at `internal/cli/org.go:137` and `internal/cli/status_test.go:99`; same for `NewReceiptStore` (`internal/org/receipts.go:46`) and `ReceiptsRelPath` (`:37`) | Delete both root-relative constructors and their `*RelPath` constants, or move `orgManifestPath`/`orgReceiptsPath` into `internal/org` so there is exactly one grep-able derivation. Tech-debt row added. |
| C2-4 | MEDIUM | maintainability | Tech-debt half-closure. The two rows cycle 1 appended were both invalidated by `746b35b` and left open and unannotated, in the same file where that commit struck through two other rows. Row 84 said the helpers have no executable caller — SKILL.md Step 2.b and Step 6 now call both. Row 85 enumerated four live leftovers — all four were fixed and the pattern widened. Two further rows cited `scripts/xreview-helpers.sh:61-68` / `:80-`; after `fea2ee6` added 6 doc lines and the `tr` line, `pick_reviewer` is at `:66` and `count_triage_findings` at `:86`. | `docs/tech-debt/README.md:84,85` before this review; `:30,31` line citations; `.claude/skills/cross-review/SKILL.md:44,105` | **Fixed in this review**: row 84 struck through with a RESOLVED annotation, row 85 rewritten down to its actual residual (C2-5), the two line-number citations refreshed. |
| C2-5 | MEDIUM | maintainability | The widened guard still under-claims relative to its own header — the same class of gap cycle 1 raised, half-fixed. The header now advertises "the per-phase `RALPH_*_MODEL` knobs"; `PATTERN` covers `FORCE\|IMPLEMENT\|SELF_REVIEW\|PROBE\|ESCALATION` — 5 of the 8 knobs the retired `model-routing.md` table defined. `RALPH_VERIFY_MODEL`, `RALPH_TEST_MODEL`, `RALPH_SYNC_DOCS_MODEL`, `RALPH_PR_MODEL` are unguarded. The fix copied cycle 1's suggested token list verbatim instead of the actual knob set. Verified all four have zero live hits today, so widening is free. Separately, `EXCLUDE_REGEX` now exempts the whole of `scripts/xreview-helpers.sh` (+ mirror) rather than just its past-tense provenance comment. | `tests/test-no-loop-references.sh:1-9` (header), `:29` (`PATTERN`), `:54` (`EXCLUDE_REGEX`); `grep -rEl 'RALPH_(VERIFY\|TEST\|SYNC_DOCS\|PR)_MODEL'` over `*.sh *.go *.toml *.md` minus historical dirs → no matches | Complete the alternation to all 9 tokens (`FORCE\|IMPLEMENT\|SELF_REVIEW\|VERIFY\|TEST\|SYNC_DOCS\|PR\|PROBE\|ESCALATION`) and narrow the `xreview-helpers.sh` exclusion, or narrow the header to what the pattern actually covers. Tech-debt row rewritten to this residual. |
| C2-6 | MEDIUM | maintainability | Retirement residual the widened guard structurally cannot catch: `ralph insights` still defaults `--receipts` to `.harness/state/pipeline/model-receipts.jsonl`, whose only writer was `write_model_receipt` in the deleted `scripts/ralph-cli-driver.sh`. The org runtime writes to `.harness/state/org/model-receipts.jsonl` with a different schema. Both the flag help and the reader's package doc present the retired path as a live source; the receipt-diagnostics section is now permanently empty on a fresh repo. The guard only matches script/symbol names, never state-dir path strings. | `internal/cli/insights.go:25,39,46`; `internal/insights/receipts.go:9`; vs `internal/org/receipts.go:37` (`ReceiptsRelPath`) | Repoint the default at the org runtime's receipts (needs a schema decision) or mark the flag historical-only in its help text. Tech-debt row added. Also worth adding state-dir path fragments to the guard's pattern vocabulary. |
| C2-7 | LOW | readability | The `IncludeDryRun` rationale argues backwards: "…already render a `[dry-run]` marker per seat, **which would otherwise be unreachable dead code**" uses marker-deadness to justify a user-visible semantics change. The actual justification (a summary command with no `--all` flag should show the whole manifest) is buried mid-sentence, and the trailing three lines are provenance about how the change was discovered rather than why it is correct. | `internal/cli/status.go:63-72` | Lead with the semantics reason, keep the marker note as a supporting detail, and move the discovery narrative to the commit message. |
| C2-8 | LOW | comment accuracy | `TestStatusCmd_JSONSchemaIsIdenticalEmptyVsPopulated`'s closing comment claims "Both payloads decode into the same schema with no divergent fields (verified above by sharing the `payload` struct for both `Unmarshal` calls)". `json.Unmarshal` silently ignores unknown fields, so sharing a struct verifies nothing about divergence — an extra field on either path would still pass. The underlying fix (one `statusJSON`) is correct; only the claim is wrong. | `internal/cli/status_test.go` (`TestStatusCmd_JSONSchemaIsIdenticalEmptyVsPopulated`, closing block) | Either drop the claim, or make it real by unmarshalling both into `map[string]any` and comparing key sets. |
| C2-9 | LOW | readability | The `claudeEnvelope` comment fix removed a line's worth of text without rewrapping, leaving a ~110-char line in a file wrapped at ~72. | `internal/org/watcher.go:116` (`// ({"result": "...", "session_id": "..."}). Model is populated only if the installed claude version happens`) | Rewrap to the surrounding width. |
| C2-10 | LOW | consistency | The new historical banner on `docs/research/approach-comparison.md` is written in Japanese inside an otherwise entirely English document — in the same commit that converted `ralph status`'s empty-state message *from* Japanese to English for exactly this consistency reason (and updated the test's failure message to say "in English (matching the rest of the CLI's output)"). | `docs/research/approach-comparison.md:3-8` vs `internal/cli/status.go:155-159` and `internal/cli/status_test.go`'s updated assertion message | Pick one language per artifact class. English matches the document it sits on. |
| C2-11 | LOW | naming | Asymmetric extraction inside the one file the AR-1 fix touched: the manifest path got a named, documented `orgManifestPath` helper while the receipts path two lines above still uses a bare `filepath.Join(resolvedStateDir, "model-receipts.jsonl")` literal that duplicates `org.ReceiptsRelPath`'s basename. The identical double-join trap is one new receipts *reader* away, with no helper and no warning comment to prevent it. | `internal/cli/org.go:122-123` (receipts literal) vs `:128-142` (`orgManifestPath` + its 12-line rationale) | Add `orgReceiptsPath` beside `orgManifestPath` (or handle both in the `internal/org` move proposed in C2-3). |
| C2-12 | LOW | test hygiene | Three small issues in the new AR-1 regression test. (a) `t.Setenv("PATH", "")` is a blunt instrument — it works today only because `--state-dir` is explicit and `--dry-run` execs nothing; if any code on that path later shells out, the failure surfaces as an opaque `exec: "git": executable file not found in $PATH` rather than as the behavior under test. (b) The comment "Deliberately does not touch `orgManifestPath`'s fixture helper directly (unlike `seedTwoOrgManifest`)" is confusing — `orgManifestPath` is not a fixture helper, and the test seeds no fixture at all; the intended point is "writes via the real spawn path". (c) `strings.Contains(out, "lead")` also matches the ROLE column, so it cannot distinguish seat from role. | `internal/cli/status_test.go` (`TestStatusCmd_SeesSeatWrittenByRealOrgSpawn`) | Drop or narrow the `PATH` override (a comment already explains dry-run needs no binaries), reword the comment to "writes through the real spawn path rather than a hand-seeded fixture", and assert on the rendered row (e.g. `"lead\tlead\tclaude\topus"`) instead of a bare substring. |

## Cycle-2 positive notes

- **The AR-1 fix is the right shape, not just the right patch.** `orgManifestPath`
  makes the write path (`newOrgRuntimeAt`) and the read path (`runStatus`) share one
  derivation, and the regression test deliberately drives a *real* `ralph org spawn
  --dry-run` followed by a *real* `ralph status` against the same `--state-dir` rather
  than hand-seeding a fixture — so a future refactor that moves only one call site off
  the helper still fails. That is the failure mode a fixture-based test would miss.
- **The `pick_reviewer` case fix is portable and fully mirrored.** `printf '%s' | tr`
  avoids the `echo` portability trap, the doc comment now states the case-insensitivity
  contract (and it genuinely matches SKILL.md Step 2.a's "Accepts `claude` or `codex`
  (case-insensitive)"), and three test cases cover the arg path, the mixed-case path,
  and the `RALPH_PRIMARY_CLI` env path. `cmp` clean against `templates/base/`, both
  `100755`.
- **Mirror discipline held across the whole cycle-2 delta.** `cmp`/`md5` identical for
  `scripts/xreview-helpers.sh`, `scripts/ralph-config.sh`, `.codex/README.md`,
  `docs/insights/README.md`, `.claude/skills/cross-review/SKILL.md` ↔
  `templates/base/.claude/...`, and `.agents/...` ↔ `templates/base/.agents/...`.
- **The `count_triage_findings` wiring chose the harder, better direction.** Rather than
  deleting the unreferenced helper and its ~110 lines of tests, Step 6 now re-derives the
  counts from the written report and cross-checks them against the canonical
  `- After triage: …` line — turning a dead helper into a drift detector for the triage
  template. Consistent with `.claude/rules/architecture.md`'s grep-able-contract rule.
- **`docs/insights/README.md` was reworded, not gutted.** Historical `flow: loop` /
  `source: pipeline` values are documented as read-compat rather than deleted, the
  example block now shows a live event *and* a historical one side by side, and the
  "why per-task files" rationale was rewritten from Loop-slice framing to worktree
  framing without losing the argument.
- **No secrets, no debug output, no commented-out code, no leftover TODOs** anywhere in
  the +793 lines. No new error paths are swallowed; `printStatusEmpty`'s change moves in
  the opposite direction by surfacing a previously discarded signal.

## Cycle-2 recommendation

- **Merge: proceed.** No CRITICAL, no HIGH. Cycle 1's one HIGH is fully closed, and 8 of
  its 9 MEDIUMs are closed outright (the 9th, guard breadth, is closed enough to pass its
  own contract and downgraded to the residual in C2-5).
- **Worth fixing before merge (small, contained):**
  - **C2-1** — the aggregate `active N/M` / `active_count` now disagrees with the number
    that gates spawning. It is the only cycle-2 finding that changes what an operator
    reads off the screen, and `internal/org/report.go:136,201` already shows the intended
    split two files away.
  - **C2-2** — one comment at `internal/org/manifest.go:110-118`; leaving it says the
    opposite of what the code now does.
- **Follow-ups (tracked in `docs/tech-debt/README.md`):** C2-3 (dead root-relative
  constructors), C2-5 (4 unguarded model knobs + whole-file guard exclusion), C2-6
  (`ralph insights --receipts` default has no writer). C2-7…C2-12 are one-line polish
  items with no behavioral risk.
- **Register hygiene performed during this review:** cycle-1 row 84 struck through as
  RESOLVED, row 85 rewritten to its true residual, two stale line-number citations
  refreshed, and three new rows added (C2-3, C2-6, and the C2-5 residual).
- **Known gaps in this review:** delta-scoped by instruction — the ~30k-line deletion
  reviewed in cycle 1 was not re-read, and spec compliance, test adequacy, and
  documentation drift remain `/verify` and `/test` territory. No tests, linters, or
  static analysis were run; `tests/test-no-loop-references.sh`'s outcome was established
  by replicating its `grep` manually, not by executing it.
