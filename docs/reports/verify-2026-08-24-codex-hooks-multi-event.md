# Verify report: codex-hooks-multi-event

- Date: 2026-08-24
- Plan: `docs/plans/active/2026-08-24-codex-hooks-multi-event.md`
- Verifier: `verifier` subagent (Claude Code)
- Scope: spec compliance (AC-1 through AC-8) + static analysis only. Behavioral tests are `/test`'s job and were not run here.
- Diff reviewed: `git diff main...HEAD` — 16 files, +764/−72 (branch `feat/codex-hooks-multi-event`, HEAD `7a98297`). Includes fix commit `c86eafb` (self-review M1/M2/M3 + L1).
- Evidence: `docs/evidence/verify-2026-08-24-codex-hooks-multi-event.log` (copy of the `run-static-verify.sh` run at `2026-08-24T02:38:42Z`)

## Spec compliance

| Acceptance criterion | Status | Evidence |
| --- | --- | --- |
| AC-1 `.codex/hooks.json` (both copies byte-identical) wires PostToolUse + PreToolUse/SessionStart/UserPromptSubmit through the dispatcher; existing PostToolUse command string unchanged (trust hash preserved); no SessionEnd/PreCompact entries | **Pass** | `diff .codex/hooks.json templates/base/.codex/hooks.json` → identical. `git show main:.codex/hooks.json` vs `HEAD:.codex/hooks.json`, PostToolUse entry byte-for-byte identical (`matcher: "Edit\|Write\|MultiEdit\|apply_patch"`, same command string). New `PreToolUse`(matcher `Bash`)/`SessionStart`/`UserPromptSubmit` entries present, each `cd "$(git rev-parse --show-toplevel)" && ./.claude/hooks/ralph-dispatch.sh <event>`. `grep -n "SessionEnd\|PreCompact"` on both files → no match. |
| AC-2 live-fire (trusted meta-repo, bypass) proves dispatcher → third-layer drop-in execution for all shipped events; evidence saved | **Pass** | `docs/evidence/codex-hooks-multi-event-slice1-2026-08-24.md` — all 3 new events captured at the third `.d` layer with distinguishing payload fields (session_id/transcript_path for SessionStart, prompt text for UserPromptSubmit, tool_name=Bash/tool_input.command for PreToolUse), cross-checked against concurrent Claude-side captures via `transcript_path`. PostToolUse wiring pre-existed and is unchanged. |
| AC-3 fresh `ralph init` fixture confirms at least one additional event fires (bypass) | **Pass** | `docs/evidence/codex-hooks-multi-event-fixture-2026-08-24.md` — built `ralph` from the worktree commit, ran `ralph init --yes` into a scratch git repo, confirmed the generated `.codex/hooks.json` has all 4 events, and all 3 new events fired to third-layer probes (`SessionStart`, `UserPromptSubmit`, `PreToolUse` all captured with fixture-root `cwd`). Fixture cleaned up, nothing committed. |
| AC-4 (hard) PreToolUse deny actually blocks the command under Codex, proven via filesystem state; pre_bash_guard extracts the command from the Codex Bash payload | **Pass** | `docs/evidence/codex-hooks-multi-event-slice1-2026-08-24.md` §3 — deny-probe command not executed (marker file absent, only the allowed-run marker present), router log shows `Command blocked by PreToolUse hook`, model responded `BLOCKED`. §2 confirms `tool_name` is literally `Bash` and `tool_input.command` matches `pre_bash_guard.sh`'s existing extraction path (no code change needed, confirmed by the unmodified `pre_bash_guard.sh` in this diff). |
| AC-5 doctor checks dispatcher routing per shipped event, warns on gaps (negative test); existing checks stay green | **Pass** | `internal/cli/doctor.go` — new `codexShippedHookEvents` var (`PostToolUse, PreToolUse, SessionStart, UserPromptSubmit`); `validateCodexHooksJSON` now iterates this set and emits one finding per un-routed event instead of a single PostToolUse-only check. `internal/cli/doctor_hooks_test.go` — `TestValidateCodexHooksJSON_PostToolUseOnly_FlagsMissingShippedEvents` (negative: legacy PostToolUse-only fixture flags the 3 new events, does not flag PostToolUse), `TestValidateCodexHooksJSON_AllFourEventsWired_NoMissingEventFindings` (positive). `TestCodexShippedHookEventsMatchesShippedHooksJSON` (added in fix commit `c86eafb`, resolving self-review M2) binds the Go list to the real `templates/base/.codex/hooks.json` event-key set via `runtime.Caller`-relative path resolution. `./scripts/run-static-verify.sh` shows `golangci-lint`/`gofmt`/`go vet` clean and `check-sync`/`check-template-purity`/`check-pipeline-sync`/`check-skill-sync` all PASS with 0 DRIFTED — existing checks unaffected. |
| AC-6 `tests/test-hook-wiring.sh` covers the shipped event set | **Pass** | `tests/test-hook-wiring.sh:174-246` — per-event loop over `PreToolUse SessionStart UserPromptSubmit` asserts a command entry exists, routes through `ralph-dispatch.sh $event`, and carries the cd-prefix/relative-dispatch form; plus a `PreToolUse` matcher-exactness check (`"Bash"`) and a non-goal guard asserting `SessionEnd`/`PreCompact` are absent. Runs for both `root` and `templates/base` surfaces (`:433-442`), plus the existing byte-identity check (`:443`). |
| AC-7 docs updated (event list, PostToolUseFailure non-support, trust re-approval note) | **Pass** | `.codex/README.md` / twin: event list bullets for all 4 shipped events, explicit "`PostToolUseFailure` — Claude Code-specific, not wired" note, corrected trust-UX paragraph ("any new or edited event requires a fresh interactive session... approvals for command strings that did not change survive" — fixed in `c86eafb` from an over-narrow "this rollout" framing flagged as self-review M3). `.codex/hooks/README.md` / twin: pluralised to "the wired PostToolUse / PreToolUse / SessionStart / UserPromptSubmit groups" (fixed in `c86eafb`, self-review L1). `docs/recipes/codex-setup.md` was left untouched — checked, it contains no event-specific enumeration that would drift (generic "hooks.json dispatcher wiring" language only), so no update was needed there. |
| AC-8 `./scripts/run-verify.sh` exit 0, check-sync/purity/tests all green | **Partial pass (static portion only)** | `./scripts/run-static-verify.sh` (changed-language scope) exited 0: shellcheck/`sh -n` on all hook scripts OK, `jq -e .` on both `settings.json` OK, Codex hook single-source guard / inline-hook-detector / PR-provenance guard OK, `check-sync.sh` → 0 DRIFTED, `check-pipeline-sync.sh` OK, `check-skill-sync.sh` OK, `check-template-purity.sh` PASS (no meta-repo references leaked into templates — confirms the M3 fix), golang verifier (`gofmt`, `go vet` via lint, `golangci-lint` → "0 issues.") OK. **The behavioral-test portion of this AC (`go test ./...`, `tests/test-hook-wiring.sh`, `tests/test-pre-bash-guard.sh`, full `run-verify.sh` including tests) is `/test`'s responsibility and was not executed here** — flagging as unverified, not failed. |

## Static analysis

| Command | Result | Notes |
| --- | --- | --- |
| `./scripts/run-static-verify.sh` (changed-language scope, default) | Exit 0 | Full log at `docs/evidence/verify-2026-08-24-codex-hooks-multi-event.log`. Language scope resolved to `golang` (fallback triggered by `unclassified:.codex/hooks.json`, expected — JSON isn't a language-pack extension). |
| `diff .codex/hooks.json templates/base/.codex/hooks.json` | Identical | Manual spot-check independent of `check-sync.sh`. |
| `diff .codex/README.md templates/base/.codex/README.md` | Identical | Manual spot-check. |
| `diff .codex/hooks/README.md templates/base/.codex/hooks/README.md` | Identical | Manual spot-check. |
| `git diff main...HEAD -- .codex/hooks.json` PostToolUse entry | No change | Confirms trust-hash preservation claim independently of the plan's own assertion. |

## Documentation drift

| Doc / contract | In sync? | Notes |
| --- | --- | --- |
| `.codex/README.md` + twin | Yes | Event list, `PostToolUseFailure` non-support, trust re-approval note, `SessionStart`/`UserPromptSubmit` side-effect description all corrected in `c86eafb` per self-review M1/M3. |
| `.codex/hooks/README.md` + twin | Yes | Pluralised per self-review L1, fixed in `c86eafb`. |
| `docs/tech-debt/README.md` | Yes | New row for M4 (Codex `ask`-decision semantics unverified) added in `c86eafb`; pre-existing `SessionEnd`/`PreCompact` deferral row untouched and still accurate (plan's Non-goals still match). |
| `docs/recipes/codex-setup.md` | Yes (no change needed) | Contains no per-event enumeration; generic language does not drift. |
| Plan progress checklist | Minor drift (non-blocking) | `docs/plans/active/2026-08-24-codex-hooks-multi-event.md` line 129 still shows `- [ ] Post-implementation pipeline` — expected at this point mid-pipeline, will be ticked by `/pr` per repo convention; not a defect. |
| `internal/cli/doctor.go` comment referencing the plan path | Consistent | Comment cites `docs/plans/active/2026-08-24-codex-hooks-multi-event.md` (self-review L4, not fixed — plan will move to `docs/plans/archive/` at `/pr`, matching the repo's dominant convention per self-review's own note that this is "convention-consistent, not a rule break"). Flagging as a known, accepted minor gap, not a defect. |

## Observational checks

- `git status` / `git diff` on the worktree: clean, nothing uncommitted — all evidence reflects committed state (`HEAD=7a98297`).
- Self-review (`docs/reports/self-review-2026-08-24-codex-hooks-multi-event.md`) findings cross-checked against `c86eafb`: M1 (session_start_context.sh side-effect claim), M2 (Go/JSON event-set sync test), M3 (meta-repo rollout narrative leaking into template README), L1 (hooks/README.md pluralisation) all fixed and verified in the diff above. M2's new test (`TestCodexShippedHookEventsMatchesShippedHooksJSON`) was read in full — it correctly resolves the path via `runtime.Caller` (cwd-independent) and fails closed (both length-mismatch and element-mismatch branches call `t.Fatalf`).
- M4 (Codex `ask`-decision semantics unverified) and self-review L2 (malformed matcher failure-message string, `tests/test-hook-wiring.sh:233`)/L3 (PostToolUse loop duplication)/L4 (plan-path citation) were left as-is — all four were explicitly scored optional/deferrable by the self-review's own recommendation, and M4 has a tech-debt row. This matches the self-review's stated "merge after M1/M3" bar, not a stricter one.
- `codexShippedHookEvents`'s doc comment cites the active plan path (not yet archived) — consistent with existing convention elsewhere in the repo (`config.go:31` pattern), not treated as a defect.

## Coverage gaps

- **Behavioral tests not run** (by design — `/test`'s scope): `go test ./internal/cli/...` (would exercise the four new/modified doctor tests), `./tests/test-hook-wiring.sh` (would exercise the new per-event assertions end-to-end against the real shipped files), full `./scripts/run-verify.sh`. AC-8's test-suite clause is therefore unverified here, not failed.
- **AC-4's `ask`-path gap is a known, accepted gap**, not a verify-time miss: the plan's hard condition (AC-4) was scoped to `deny` only, and the self-review already filed M4 + a tech-debt row for the `ask` path. No further action needed from `/verify`.
- Live-fire evidence (AC-2/AC-3/AC-4) is Codex-CLI-version-pinned (`codex-cli 0.147.0`) and environment-observational, not something `/verify`'s static tooling re-executes — treated as verified-by-evidence-doc per the plan's own "live-fire" verify plan, consistent with how prior plans in this family (`2026-08-20-codex-hooks-json-wiring`) were verified.

## Verdict

**Overall: Pass** (all 8 ACs pass or partial-pass; no fail, no blocking gap).

- Verified: AC-1, AC-2, AC-3, AC-4, AC-5, AC-6, AC-7 (full pass, with direct evidence-doc and diff cross-checks beyond restating the plan's own claims). Static-analysis portion of AC-8 (`run-static-verify.sh` exit 0, check-sync 0 DRIFTED, check-template-purity PASS, check-pipeline-sync/check-skill-sync PASS, golang lint/vet/fmt clean). Self-review M1/M2/M3/L1 fix commit (`c86eafb`) independently confirmed correct and complete.
- Partially verified: AC-8 — static gates confirmed green; the behavioral-test clause (`go test`, `tests/test-hook-wiring.sh`, `tests/test-pre-bash-guard.sh`, full `run-verify.sh`) is out of `/verify`'s scope and remains for `/test` to execute and report.
- Not verified: Runtime Codex semantics beyond what the evidence docs captured (e.g. `ask`-decision handling under Codex — tracked as tech-debt, not a gap in this verification's scope, since AC-4 only requires the `deny` proof).
