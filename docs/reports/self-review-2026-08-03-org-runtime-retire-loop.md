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
