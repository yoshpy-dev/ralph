# Verify report: org-send-enter-timing

- Date: 2026-09-19
- Plan: `docs/plans/active/2026-09-19-org-send-enter-timing.md` (issue #163)
- Verifier: `verifier` subagent (Claude Code), standard flow, pipeline cycle 1 of cap 2
- Scope: spec compliance (AC-1..AC-8) + static analysis for `git diff main...HEAD` on `fix/org-send-enter-timing`, base `f006200`, HEAD `9d99f39`, working tree clean at start and end. No test execution (`/test`'s job) and no diff-quality review (`/self-review`'s job, already MERGE per `docs/reports/self-review-2026-09-19-org-send-enter-timing.md`).

## Acceptance criteria

| AC | Status | Evidence |
|----|--------|----------|
| AC-1: fixed fake-herdr call order (idle/done wait → send-text → wait → Enter → working/blocked wait) | **Met** | `internal/org/verbs.go:251` (idle/done `AgentWait`) → `:284` (`PaneSendText`) → `:288` (`waitBeforeEnter`) → `:296` (`PaneSendKeys("Enter")`) → `:304` (`confirmSubmitted` → `AgentWait`). Pinned by `TestOrgSend_Confirmed_ExactCallOrderAndUntilStates` (`internal/org/verbs_test.go:450-485`), which asserts `wantOrder := []string{"agent_wait", "pane_send_text", "pane_send_keys", "agent_wait"}` against `h.calls` and separately checks the first `AgentWait`'s `until` is `["idle","done"]` and the second is `["working","blocked"]`. |
| AC-2: unconfirmed submit → `submit_unconfirmed=true` on `sent`, `Send` still succeeds, Enter sent exactly once even when unconfirmed | **Met** | `internal/org/verbs.go:304-309` prepends `submit_unconfirmed=true ` to Details only when `!submitConfirmed`, and the call still reaches the success return (`:330`). `TestOrgSend_UnconfirmedSubmit_NeverResendsEnter` (`verbs_test.go:496-526`) scripts the confirm `AgentWait` to fail, asserts `result.Err == nil`, `SubmitConfirmed == false`, `len(h.sendKeysKeys) == 1` with `h.sendKeysKeys[0] == ["Enter"]`, and `Details` starts with `submit_unconfirmed=true `. `TestOrgSend_RawAndUnconfirmed_DetailsPrefixOrder` (`:533-554`) pins the `raw=true submit_unconfirmed=true ` ordering. |
| AC-3: `--enter-delay-ms` usable, negative rejected; unconfirmed-submit stderr note includes `pane_id`, exit 0 | **Met** | Flag: `internal/cli/org.go:347,377-378,443-444`. Rejection: `:363-365` (`if enterDelayMS < 0 { return fmt.Errorf(...) }`), tested by `TestOrgSend_NegativeEnterDelayMS_NonZeroExit` (`internal/cli/org_test.go:1198-1213`, asserts non-zero exit, error mentions the flag, and zero herdr calls). Unconfirmed-submit warning: `internal/cli/org.go:424-432`, format args `to, *orgID, to, result.PaneID` — includes the pane id and stays on stderr with the command returning `nil` (exit 0). `TestOrgSend_UnconfirmedSubmit_WarnsOnStderr_ExitZero` (`internal/cli/org_test.go:1221-1252`, read in full) asserts exit 0, `"could not confirm"`, the seat name, the pane id (`pane-stub-1`), and `"does not resend Enter"` all present in output; `TestOrgSend_DryRun_NeverWarnsAboutUnconfirmedSubmit` at `:1259` confirms the warning is suppressed for `--dry-run`. |
| AC-4: existing paths unchanged (protocol-validation rejection, dry-run, unknown seat, send-text/Enter errors) | **Met** | `internal/org/verbs.go:199-217` (validation-before-anything, dry-run path) and `:219-228` (unknown seat / no pane-id) are untouched in shape from `main`'s pre-fix logic per the self-review's diff read; regression tests `TestOrgSend_Malformed_RejectedNoManifestEventNoDriverCall`, `TestOrgSend_DryRun_Malformed_AlsoRejectedBeforeManifestEvent`, `TestOrgSend_UnknownSeat_ErrorsWithoutDriverCall`, `TestOrgSend_SeatWithoutPaneID_Errors` (`verbs_test.go:192-392`) remain present and unmodified. New send-text/Enter-failure paths are additionally covered by `TestOrgSend_PaneSendKeysFails_ReportsTypedButNotSubmitted` (`verbs_test.go:708-736`) and its CLI counterpart `TestOrgSend_PaneSendKeysFails_NotesTypedButNotSubmittedOnStderr` (`org_test.go:1280`). |
| AC-5: live evidence for codex + claude idle seats, pre-fix repro, `sent` Details, blocked-dialog state, cleanup | **Met** | `docs/evidence/org-send-enter-timing-2026-09-19.md`: results table (codex pre-fix 2/5 auto-submit vs 5/5 post-fix; claude 5/5 both), the blocked-dialog section (`Press enter to confirm`, "Yes, proceed" preselected — the no-resend justification), and a "後始末" section recording seat stop/disband, deleted duplicated `auth.json`, stopped dedicated herdr server, and confirming the user's own dotfiles/config are unchanged (one residual: a `~/.claude.json` scratch-dir trust entry, disclosed rather than hidden). |
| AC-6: recipe (root/template) and 4 skill surfaces describe the new behavior with no manual-Enter-only wording left, 3 sync gates pass | **Met** | `docs/recipes/codex-seat-permissions.md:140-146` and its template copy are byte-identical (`diff` empty) and describe: wait → Enter once → confirm via herdr → manual Enter only if text is still in the composer → ralph never resends. `.claude/skills/org/SKILL.md:105` and its 3 mirrors (`.agents/skills/org/SKILL.md`, `templates/base/.claude/...`, `templates/base/.agents/...`) are byte-identical and state the same contract. `./scripts/check-skill-sync.sh`, `./scripts/check-sync.sh`, `./scripts/check-template-purity.sh` all PASS (see Deterministic checks table). |
| AC-7: `gofmt`/`go vet`/golangci-lint clean; `go test`/`run-verify.sh`/range secret scan green | **Partially verified** | `gofmt`, `go vet`, golangci-lint (`0 issues.`) and staticcheck (silent success) all clean via `./scripts/run-static-verify.sh` and a manually scoped re-run (see Deterministic checks table) — this is the static half. The `go test` / `run-verify.sh` / range-secret-scan half is **out of scope for `/verify`** by design (that is `/test`'s job); the plan's own Deviation notes record these as green from the implementer's runs (`-count=20`, `TMPDIR=/tmp`, `-race`, range secret scan exit 0 for the final Slice E commit `bcc1eb2`/`f7ea44d`), which `/verify` did not independently re-run and is not the authority for. |
| AC-8: PR body `Closes #163` | **Not yet applicable** | No PR exists yet at this pipeline stage (`/pr` runs after `/sync-docs` and `/cross-review`). Nothing to verify here now. |

## Deterministic checks run

| Command | Result | Notes |
| --- | --- | --- |
| `./scripts/run-static-verify.sh` (default changed-language scope) | PASS | Config-validity checks, Codex hook/PR-provenance guards, `check-sync.sh` (0 DRIFTED), `check-pipeline-sync.sh`, `check-skill-sync.sh` (13 skills in lock-step), `check-template-purity.sh`, then the golang verifier: `gofmt: ok`, `go vet` silent (pass), golangci-lint `0 issues.`, staticcheck silent (pass, binary present at `~/go/bin/staticcheck`). Evidence: `docs/evidence/verify-2026-09-19-142546.log`. |
| `gofmt -l internal/org internal/cli` | PASS | No output — both changed packages are formatted. |
| `go vet ./internal/org/... ./internal/cli/...` | PASS | No output. |
| `./scripts/check-skill-sync.sh` | PASS | Already included in the `run-static-verify.sh` run above; re-confirmed as part of it. |
| `./scripts/check-sync.sh` | PASS | `DRIFTED: 0`; all differences from `templates/base/` are pre-existing `KNOWN_DIFF`/`TEMPLATE_ONLY` entries unrelated to this change. |
| `./scripts/check-template-purity.sh` | PASS | No meta-repo-specific references found in templates. |

## Documentation drift

None found. Checked:

- `internal/org/verbs.go` doc comments (`Send`, `SendResult.TextTyped`/`EnterPressed`/`SubmitConfirmed`, `confirmSubmitted`, `capMSToContext`, `defaultSendEnterDelay`/`defaultSendSubmitConfirmTimeout`) against the code they describe — all consistent, including the two self-review-revalidation fixes (M1's herdr-contract correction, NEW-1's `EnterPressed` split).
- `internal/cli/org.go`'s `send` `Long` description and the two post-error stderr notes (`EnterPressed` branch vs `TextTyped` branch) against `SendResult`'s field contract — consistent; the `EnterPressed` branch correctly says "do not send the message again" rather than suggesting a retry.
- `--timeout-ms` help text (`internal/cli/org.go:439-440`, "overall herdr timeout in milliseconds for one send (idle wait + pre-Enter wait + submit confirmation)") against the ctx's actual span (`verbs.go:234` `context.WithTimeout` wraps the idle/done wait, the budget check, `PaneSendText`, `waitBeforeEnter`, `PaneSendKeys`, and `confirmSubmitted`) — consistent; this is the self-review M3 fix landing correctly.
- `.claude/skills/org/SKILL.md`'s `send` row and its 3 mirrors, `docs/recipes/codex-seat-permissions.md` and its template copy — consistent with code and byte-identical across mirrors/copies (see AC-6 evidence above).
- `docs/evidence/codex-seat-permissions-2026-09-18.md`'s P2 follow-up — one added line (`:44`) pointing at the new evidence file, present and accurate.
- `docs/specs/2026-08-01-org-runtime.md`, `README.md`, `.claude/rules/ralph/agent-messaging.md` — grepped for `org send`/`Enter`; no mentions of Enter timing, submit confirmation, or manual-Enter guidance in any of the three, so nothing there needed updating and nothing there is now stale.
- `internal/org/send_defaults_sync_test.go` — exists, and its adjacency check (`lineContainsBoth`, requiring the default's marker and `--enter-delay-ms` on the *same line*) matches the self-review revalidation LOW-4 fix, not the weaker whole-file `strings.Contains` the cycle-1 version had.
- `docs/tech-debt/README.md` — no diff on this branch; consistent with the self-review's own conclusion that all findings were fixed in-cycle rather than deferred, so no row was owed.

## Non-goals sweep

- **No automatic Enter resend**: `grep -rn 'PaneSendKeys(' internal/org internal/cli` (non-test) shows exactly one Enter call site (`verbs.go:296`, inside `Send`) plus two unrelated `C-c` sites (`Stop`, `spawn.go:1324` and `verbs.go:556`). `grep -n resend` only turns up comments explaining the deliberate no-resend design, never code that does it. `Send` is called from exactly one CLI call site (`internal/cli/org.go:376`), with no retry wrapper around it.
- **No pane-content scraping in `Send`**: the only `PaneRead` call in `internal/org/verbs.go` is inside `Read` (line 510), not `Send`.
- **No new `ralph.toml` key**: `git diff main...HEAD -- ralph.toml templates/base/ralph.toml internal/config/` is empty.

## Coverage gaps / what remains unverified

- **Behavioral test execution** (`go test ./internal/org/... ./internal/cli/...`, `-race`, `-count=20`, `TMPDIR=/tmp`) was intentionally not run here — that is `/test`'s responsibility. The plan's Deviation notes record these as green from the implementer's own runs on the final commits, but `/verify` has not independently reproduced them.
- **Range secret scan** (`./scripts/secret-scan.sh --range "$(git merge-base HEAD origin/main)..HEAD"`) was not re-run by `/verify`; the plan records it as exit 0 from the implementer's runs. Given this repo's history of secret-scan CI failures on unrelated branches (fixture wording, allowlist lines), a fresh run before `/pr` is still worth doing even though nothing in this diff resembles a credential-shaped string (spot-checked: no `key=value`-style assignments anywhere in the diff).
- AC-8 (`Closes #163` in the PR body) cannot be checked until `/pr` runs; not a gap in this cycle, just not yet applicable.

## Verdict

**Pass.** AC-1 through AC-6 are fully met with direct code + test evidence. AC-7 is met for its static-analysis half (the half `/verify` owns); its test-execution half is deferred to `/test` as designed, not a failure here. AC-8 is not yet applicable. No documentation drift found across the code doc comments, CLI help/flag text, the `/org` skill (4 mirrors), the recipe (2 copies), or the three cross-referenced docs that mention `ralph org send`. All three requested sync gates and the scoped `gofmt`/`go vet` checks pass clean.

## Cycle 2 (2026-09-19)

- Scope: re-verify against HEAD `bf4c8c8` (the cycle-1 section above is scoped to `9d99f39` and describes the superseded `SendResult.TextTyped`/`EnterPressed` two-bool contract — do not treat it as current for anything past AC wording). Diff since the cycle-1 verify commit: `git diff 354e580..HEAD`, 18 files, +1156/−180. `git status --porcelain` empty at start and end.
- What changed: cross-review cycle 1 found two ACTION_REQUIRED items (`docs/reports/cross-review-triage-org-send-enter-timing.md`) — AR-1 replaced `TextTyped`/`EnterPressed` with `SendResult.Progress` (`SendProgress`, five states: nothing-sent, text-unacknowledged, text-typed, enter-unacknowledged, enter-pressed), because a ctx deadline can kill a herdr call mid-flight and Send genuinely cannot tell whether the pane call landed; AR-2 added `orgReadCommandHint`, which appends `--state-dir <resolved-absolute-path>` to the printed `ralph org read` recovery command only when `--state-dir` was explicitly passed. Cycle-2 self-review then found and fixed six more items (C2-1..C2-6): a CLI timing test with an accidental (not designed) margin, one added sentence to the exit-0 unconfirmed-submit warning, two test renames, a doc-comment reorder, and a recipe/skill wording qualification.

### Acceptance criteria (re-verified against bf4c8c8)

| AC | Status | Evidence |
|----|--------|----------|
| AC-1: fixed fake-herdr call order | **Met, unchanged** | `TestOrgSend_Confirmed_ExactCallOrderAndUntilStates` (`internal/org/verbs_test.go:448-483`) is untouched by the AR-1/self-review-cycle-2 diff and still asserts `wantOrder := []string{"agent_wait", "pane_send_text", "pane_send_keys", "agent_wait"}` plus the `["idle","done"]` / `["working","blocked"]` `until` values. Code path unchanged: `verbs.go:305` (idle/done wait) → `:338` (`PaneSendText`) → `:351` (`waitBeforeEnter`) → `:362` (`PaneSendKeys("Enter")`) → `:373` (`confirmSubmitted`). |
| AC-2: unconfirmed → `submit_unconfirmed=true`, `Send` succeeds, exactly one Enter | **Met, unchanged** | `TestOrgSend_UnconfirmedSubmit_NeverResendsEnter` (`verbs_test.go:494-524`) unchanged: `result.Err == nil`, `SubmitConfirmed == false`, `len(h.sendKeysKeys) == 1` with `["Enter"]`, `Details` starts with `submit_unconfirmed=true `. Code: `verbs.go:373-378`. |
| AC-3: `--enter-delay-ms` usable/negative-rejected; unconfirmed-submit stderr note has `pane_id`, exit 0 | **Met, strengthened** | Flag/rejection unchanged (`internal/cli/org.go:400,416-417,520-521`; `TestOrgSend_NegativeEnterDelayMS_NonZeroExit`). The warning itself changed (C2-2): `org.go:499-509` now adds "If the pane shows anything else (the seat is working, or it shows a dialog), do not press Enter." `TestOrgSend_UnconfirmedSubmit_WarnsOnStderr_ExitZero` (`org_test.go:1238-1272`, read in full) asserts exit 0, `"could not confirm"`, seat name, `pane-stub-1`, `"does not resend Enter"`, and the new `"do not press Enter"` clause — all five present. The AC's wording ("pane_id を含む") still matches: the pane id is still in the message, just alongside more text. |
| AC-4: existing paths unchanged (protocol-validation rejection, dry-run, unknown seat, send-text/Enter errors) | **Met, one path enriched not broken** | Validation/dry-run/unknown-seat/no-pane-id (`verbs.go:254-282`) are byte-for-byte unchanged from cycle-1 HEAD; their four regression tests (`TestOrgSend_Malformed_RejectedNoManifestEventNoDriverCall`, `TestOrgSend_DryRun_Malformed_AlsoRejectedBeforeManifestEvent`, `TestOrgSend_UnknownSeat_ErrorsWithoutDriverCall`, `TestOrgSend_SeatWithoutPaneID_Errors`) are present and unmodified (`git diff` shows only additions in this test file, no deleted `func Test`). The send-text/Enter error paths still error the same way at the top level (non-nil `Err`, no `sent` event appended) — what changed is that they now report *more* information (`Progress` + `PaneID`) than the old two-bool return, which is the AR-1 fix, not a regression. New tests: `TestOrgSend_PaneSendTextFails_ReportsTextUnacknowledged` (`verbs_test.go:784-818`), `TestOrgSend_PaneSendKeysFails_ReportsEnterUnacknowledged` (`:743-771`), and their CLI counterparts `TestOrgSend_PaneSendTextFails_NotesTextUnacknowledgedOnStderr` / `TestOrgSend_PaneSendKeysFails_NotesEnterUnacknowledgedOnStderr` (`org_test.go:1302,1356`). |
| AC-5: live evidence | **Met, unchanged** | `git diff 354e580..HEAD -- docs/evidence/org-send-enter-timing-2026-09-19.md` is empty — the cycle-1 timing measurements are unaffected by AR-1/AR-2 (both are error-reporting/message-wording changes, not timing changes), so no re-measurement was needed or done. |
| AC-6: recipe + 4 skill surfaces describe current behavior, 3 sync gates pass | **Met, updated correctly** | Recipe (`docs/recipes/codex-seat-permissions.md:140-152` + template) and skill (`.claude/skills/org/SKILL.md:105` + 3 mirrors) now describe the AR-1 "may or may not have reached the pane" uncertainty, the AR-2 `--state-dir`-carrying hint, and the C2-5 qualification (no note on `SendProgressNothingSent` failures). All 6 files re-diffed pairwise: `diff .claude/skills/org/SKILL.md .agents/skills/org/SKILL.md`, `diff .claude/skills/org/SKILL.md templates/base/.claude/skills/org/SKILL.md`, `diff .claude/skills/org/SKILL.md templates/base/.agents/skills/org/SKILL.md`, `diff docs/recipes/codex-seat-permissions.md templates/base/docs/recipes/codex-seat-permissions.md` — all empty (byte-identical). `./scripts/check-skill-sync.sh` / `check-sync.sh` / `check-template-purity.sh` all PASS (see Deterministic checks below). |
| AC-7: static half | **Met** | See Deterministic checks below — clean at `bf4c8c8`. Test-execution half remains `/test`'s job; not re-run here. |
| AC-8: PR body `Closes #163` | **Not yet applicable** | No PR exists yet (this is pipeline cycle 2 of 2, still pre-`/pr`). |

**AC wording vs. the old two-bool contract**: none of AC-1 through AC-8's text names `TextTyped` or `EnterPressed` directly — they describe behavior ("Enter は 1 回しか送られない", "pane_id を含む", "既存の経路が不変") at a level of abstraction the AR-1 `SendProgress` refactor preserves. No AC reads stale.

### Static analysis (re-run at bf4c8c8)

| Command | Result | Notes |
| --- | --- | --- |
| `./scripts/run-static-verify.sh` (changed-language scope) | PASS | Same battery as cycle 1: config-validity checks, Codex hook/PR-provenance guards, `check-sync.sh` (`DRIFTED: 0`), `check-pipeline-sync.sh`, `check-skill-sync.sh` (13 skills in lock-step), `check-template-purity.sh`, then golang: `gofmt: ok`, `go vet` silent, golangci-lint `0 issues.`, staticcheck silent. Evidence: `docs/evidence/verify-2026-09-19-160021.log`. |
| `gofmt -l internal/org internal/cli` | PASS | No output. |
| `go vet ./internal/org/... ./internal/cli/...` | PASS | No output. |
| `./scripts/check-skill-sync.sh` | PASS | Included in the run above. |
| `./scripts/check-sync.sh` | PASS | `DRIFTED: 0`. |
| `./scripts/check-template-purity.sh` | PASS | No meta-repo-specific references in templates. |

### Documentation drift (re-checked against bf4c8c8)

One finding, not present in cycle 1 (it was introduced by a commit in this cycle's own range):

- **`docs/tech-debt/README.md:133` cites a test function name that no longer exists.** The row (added at `6a3a9d5`, during cycle-1 sync-docs, before the AR-1/cycle-2-self-review commits) reads "...the read-only-manifest trick used by `TestOrgSend_AppendEventFailsAfterEnter_ReportsSubmittedButUnrecorded` fails a write...". That test was renamed to `TestOrgSend_AppendEventFailsAfterEnter_ReportsEnterPressedButUnrecorded` by the C2-3 self-review fix (`646fb4b`) — confirmed by `grep -rn ReportsSubmittedButUnrecorded .`, which now only matches this tech-debt row and the two dated report files (`docs/reports/test-2026-09-19-org-send-enter-timing.md`, itself a HEAD-pinned cycle-1 artifact the cross-review report already flagged as due for its own cycle-2 `/test` re-verification, not a live doc). The self-review's own C2-3 recommendation named the test-report reference for a possible fix but did not catch this tech-debt-register reference. This is cosmetic (the row's substance — the coverage gap itself — is still accurate and still open) but is exactly the kind of dangling-identifier drift `/verify` exists to catch. **Recommendation**: one-word edit, `s/ReportsSubmittedButUnrecorded/ReportsEnterPressedButUnrecorded/` in `docs/tech-debt/README.md:133`, foldable into `/sync-docs`'s cycle-2 pass rather than a blocker.

Everything re-checked from cycle 1 and still sound:

- `internal/cli/org.go`'s `send` `Long` description, `orgReadCommandHint`'s doc comment (including the C2-6 "same shell, run from the same directory" qualification), and the `--timeout-ms` help text are all consistent with the code they describe.
- `docs/evidence/codex-seat-permissions-2026-09-18.md`'s P2 pointer line is unchanged and still accurate.
- `docs/specs/2026-08-01-org-runtime.md`, `README.md`, `.claude/rules/ralph/agent-messaging.md`: `git diff 354e580..HEAD` touches none of the three, and cycle 1 already confirmed none of them mention `ralph org send`'s Enter-timing behavior — still true.
- `internal/org/send_defaults_sync_test.go` untouched by this cycle's diff, still enforces the line-adjacency check.
- No leftover bare `TextTyped`/`EnterPressed` identifiers anywhere in `*.go` or current `*.md` docs (excluding the dated `docs/reports/` artifacts, which cite them accurately for the commits they describe): `grep -rn 'TextTyped\|EnterPressed'` over the whole tree turns up only `SendProgressTextTyped`/`SendProgressEnterPressed` enum-constant names in code/tests, plus the dated reports and the plan's own revision history (the plan intentionally keeps its superseded Scope-row-1 sentence followed by the AR-1 revision paragraph as a revision record, per the cross-review triage's own answer to this exact question — not drift).

### Non-goals sweep (re-checked at bf4c8c8)

- **No automatic Enter resend**: `grep -rn 'PaneSendKeys(' internal/org internal/cli` (non-test) — exactly one Enter call site (`verbs.go:362`, inside `Send`) plus the two pre-existing `C-c` sites in `Stop` (`spawn.go:1324`, `verbs.go:624`). No retry loop around any pane call (`grep -n 'for.*PaneSendText\|for.*PaneSendKeys' internal/org/verbs.go` — no hits). `Send` still has exactly one CLI call site (`internal/cli/org.go:438`).
- **No pane-content scraping in `Send`**: the only `PaneRead` call in `internal/org/verbs.go` remains inside `Read`, not `Send`.
- **No new `ralph.toml` key**: `git diff main...HEAD -- ralph.toml templates/base/ralph.toml internal/config/` is empty.

### Coverage gaps / what remains unverified

- Same as cycle 1: behavioral test execution and the range secret scan remain `/test`'s and the pre-push-check's responsibility, not independently re-run here.
- `docs/reports/test-2026-09-19-org-send-enter-timing.md` and `docs/reports/sync-docs-2026-09-19-org-send-enter-timing.md` are cycle-1-pinned artifacts per the cross-review triage's own "Known gaps" section; cycle 2's own `/test` and `/sync-docs` runs (still pending in this pipeline cycle) are the ones responsible for producing current versions, not this `/verify` pass.
- AC-8 remains not yet applicable (pre-`/pr`).

### Verdict (cycle 2)

**Pass.** AC-1 through AC-6 remain met (three strengthened by the AR-1/AR-2/self-review-cycle-2 fixes, three unchanged), AC-7's static half is clean, AC-8 is not yet applicable. No AC reads stale against the `SendProgress` refactor. One minor, non-blocking documentation-drift finding: a dangling pre-rename test-function-name citation in `docs/tech-debt/README.md:133`, worth a one-word fix in `/sync-docs` but not a merge blocker on its own.
