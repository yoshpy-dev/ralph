# Test report: org-codex-slug-update-procedure

- Date: 2026-09-18
- Plan: `docs/plans/active/2026-09-18-org-codex-slug-update-procedure.md` (issue #156)
- Tester: `tester` subagent (Claude Code, standard flow)
- Diff scope: `git diff main...HEAD` on `docs/org-codex-slug-update-procedure` (HEAD `9576574`, base `main`@`93cf3bb`), 9 files, +335/-0. Docs-only: `.claude/skills/org/SKILL.md` + 3 mirrors, `docs/specs/2026-08-01-org-runtime.md`, plus plan/self-review/verify/insight-event bookkeeping. No Go source, no scripts, no `ralph.toml` changed.

## Overall verdict: Pass

Full shell suite green, both requested Go regressions green, and both edge-case heading-structure inspections confirmed by inspection. No behavioral change was made by this diff (docs-only), so this pass is a regression check plus the two inspections the plan's Test plan calls out.

## Suites run

### `./scripts/run-test.sh` (`RALPH_VERIFY_SCOPE` default `changed`, `HARNESS_VERIFY_MODE=test`)

Language-scope resolution logged at the end of the run: `Language scope: changed (docs_only)` / `Language packs selected: none` — correctly resolved from the diff (no Go/TS/Python/Rust/Dart files touched), so no per-language test gate ran, only the full shell suite (which is scope-independent). Exit code 0.

28 shell suites executed under `tests/`, all passing, 0 failures anywhere in the log:

| Suite | Result |
| --- | --- |
| `test-agent-phase-boundaries.sh` | 44/44 |
| `test-branch-name.sh` | 26 passed, 0 failed, 26 total |
| `test-check-mojibake.sh` | 15/15 (`PASS: 15`) |
| `test-check-skill-sync.sh` | 13/13 |
| `test-detect-changed-languages.sh` | 23/23 |
| `test-detect-languages-terraform.sh` | 8/8 |
| `test-ensure-pr-ready.sh` | pass (embedded assertions, no failures) |
| `test-ensure-pr-title-prefix.sh` | 13/13 |
| `test-gc-artifacts.sh` | 11/11 (7 scenario blocks, all pass) |
| `test-hook-wiring.sh` | 68/68 |
| `test-insights-append.sh` | 7 cases, all pass |
| `test-language-pack-monorepo-roots.sh` | 29/29 |
| `test-no-loop-references.sh` | pass (1 guard assertion) |
| `test-post-edit-verify.sh` | pass |
| `test-pre-bash-guard.sh` | pass |
| `test-ralph-config.sh` | 24/24 |
| `test-ralph-dispatch.sh` | pass |
| `test-ralph-worktree.sh` | 26/26 (state-paths/ensure-resume/collision/cleanup sub-summary) + gc scenarios, all pass |
| `test-run-verify-scope.sh` | 12/12 |
| `test-secret-scan.sh` | 6/6 |
| `test-self-review-scope.sh` | 64/64 |
| `test-sync-skills.sh` | 22/22 |
| `test-template-purity.sh` | 10/10 |
| `test-terraform-gitignore.sh` | 47/47 |
| `test-terraform-pack-verify.sh` | 36/36 |
| `test-terraform-rule-frontmatter.sh` | 11/11 |
| `test-verify-mode-split.sh` | 59/59 |
| `test-xreview-helpers.sh` | 29/29 |

`grep -n "FAIL" /tmp/ralph-run-test-156.log` matched only `FAIL: 0` summary lines — no suite reported a nonzero failure count. Counts and suite roster match the known-stable baseline in tester memory (`tests/test-xreview-helpers.sh` 26→29 already reflected the earlier `detect_base_branch` gate-proof addition; no new suites added or removed by this docs-only diff).

### `go test ./internal/config/... -count=1 -v`

All tests pass, including `TestDefaultsLockStep` — the test the plan names as the regression tying the spec's description of the 3-face lock-step to `defaults_sync_test.go`. Confirmed by direct source read (`internal/config/defaults_sync_test.go:95` opens `scripts/ralph-config.sh`, `:102` opens `templates/base/ralph.toml`, both compared against `config.Default()`), matching the spec text's claim exactly.

### `go test ./internal/... -count=1`

All 8 packages pass:

| Package | Result |
| --- | --- |
| `internal/cli` | ok (38.029s) |
| `internal/config` | ok (0.273s) |
| `internal/insights` | ok (1.310s) |
| `internal/org` | ok (10.691s) |
| `internal/org/driver` | ok (2.374s) |
| `internal/org/protocol` | ok (0.949s) |
| `internal/scaffold` | ok (2.651s) |
| `internal/upgrade` | ok (2.419s) |

Package roster and pass count match the known-stable 8-package baseline. Neither of the two watchlisted transient flakes (`TestRunDoctorOpts_ProbeModelsFalse_NoSubprocess` in `internal/cli`, `TestRunWatcher_TimeoutIndependentOfSmallInterval` in `internal/org`) failed in this run.

## Edge-case inspections (plan's Test plan, no source edits)

- **SKILL.md heading structure**: `sed -n '42,79p' .claude/skills/org/SKILL.md | grep -n "^#"` shows only the section's own `### 既定の model_pool` heading at line 42 and the following `## 動詞リファレンス` at line 79 — no `## ` heading was introduced by the new paragraph in between. Confirmed intact.
- **Spec section placement**: `docs/specs/2026-08-01-org-runtime.md` has `## Summary` at line 3, the new `### 運用ノート: 既定 codex スラッグの更新手順(2026-09-18、issue #156)` at line 15, and `## Background and problem` at line 24 — the new section is a `###` nested under `## Summary`, placed before `## Background and problem`, matching the plan's Scope item 2 placement instruction ("「2026-09-16 改訂」節の直後"). Confirmed intact.

## Known gaps / out of scope for this pass

- No static analysis was run (verifier's job, already reported Pass in `docs/reports/verify-2026-09-18-org-codex-slug-update-procedure.md`).
- No new behavioral tests were added — correct for a docs-only diff; the plan's own Test plan states "変更なし(文書のみ)" and scopes the check to regression only.
- AC-3 (issue comment) and AC-5 (`Refs #156` PR body) remain out of scope for `/test`, same as for `/verify` — both are produced at `/pr` time per the plan's Implementation outline.
