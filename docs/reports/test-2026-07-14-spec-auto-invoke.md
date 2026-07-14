# Test report: spec-auto-invoke

- Date: 2026-07-14
- Plan: docs/plans/active/2026-07-14-spec-auto-invoke.md
- Tester: tester subagent (behavioral tests via `./scripts/run-test.sh`)
- Scope: changed-language scope (shell + golang detected; full language scope applied due to unclassified `.agents/skills/spec/agents/openai.yaml` deletion)
- Evidence: `docs/evidence/test-2026-07-14-spec-auto-invoke.log`

## Test execution

| Suite / Command | Tests | Passed | Failed | Skipped |
| --- | --- | --- | --- | --- |
| `tests/test-agent-phase-boundaries.sh` | 44 | 44 | 0 | 0 |
| `tests/test-branch-name.sh` | 29 | 29 | 0 | 0 |
| `tests/test-check-mojibake.sh` | 11 | 11 | 0 | 0 |
| `tests/test-check-skill-sync.sh` | 13 | 13 | 0 | 0 |
| `tests/test-detect-changed-languages.sh` | 23 | 23 | 0 | 0 |
| `tests/test-detect-languages-terraform.sh` | 8 | 8 | 0 | 0 |
| `tests/test-ensure-pr-ready.sh` | 7 | 7 | 0 | 0 |
| `tests/test-ensure-pr-title-prefix.sh` | 13 | 13 | 0 | 0 |
| `tests/test-gc-artifacts.sh` | 11 | 11 | 0 | 0 |
| `tests/test-insights-append.sh` | 39 | 39 | 0 | 0 |
| `tests/test-insights-pipeline-events.sh` | 37 | 37 | 0 | 0 |
| `tests/test-language-pack-monorepo-roots.sh` | 6 | 6 | 0 | 0 |
| `tests/test-model-routing.sh` | 24 | 24 | 0 | 0 |
| `tests/test-ralph-cleanup-no-remote.sh` | (inline pass) | — | 0 | 0 |
| `tests/test-ralph-cli-driver.sh` | 103 | 103 | 0 | 0 |
| `tests/test-ralph-config.sh` | 43 | 43 | 0 | 0 |
| `tests/test-ralph-deprecation-notice.sh` | (inline pass) | — | 0 | 0 |
| `tests/test-ralph-dry-run-side-effects.sh` | (inline pass) | — | 0 | 0 |
| `tests/test-ralph-orchestrator-branch-names.sh` | (inline pass) | — | 0 | 0 |
| `tests/test-ralph-orchestrator-parsers.sh` | (inline pass) | — | 0 | 0 |
| `tests/test-ralph-orchestrator-pr-strategy.sh` | (inline pass) | — | 0 | 0 |
| `tests/test-ralph-pipeline-functions.sh` | (inline pass) | — | 0 | 0 |
| `tests/test-ralph-run-options.sh` | (inline pass) | — | 0 | 0 |
| `tests/test-ralph-signals.sh` | 12 | 12 | 0 | 0 |
| `tests/test-ralph-slice-skip-pr.sh` | (inline pass) | — | 0 | 0 |
| `tests/test-ralph-status.sh` | 51 | 51 | 0 | 0 |
| `tests/test-ralph-worktree.sh` | 29 | 29 | 0 | 0 |
| `tests/test-run-verify-scope.sh` | 47 | 47 | 0 | 0 |
| `tests/test-secret-scan.sh` | (inline pass) | — | 0 | 0 |
| `tests/test-self-review-scope.sh` | 96 | 96 | 0 | 0 |
| `tests/test-sync-skills.sh` | 22 | 22 | 0 | 0 |
| `tests/test-terraform-gitignore.sh` | 47 | 47 | 0 | 0 |
| `tests/test-terraform-pack-verify.sh` | 36 | 36 | 0 | 0 |
| `tests/test-terraform-rule-frontmatter.sh` | 11 | 11 | 0 | 0 |
| `tests/test-verify-mode-split.sh` | 59 | 59 | 0 | 0 |
| `tests/test-xreview-gate-regression.sh` | 21 | 21 | 0 | 0 |
| `tests/test-xreview-prompt-render.sh` | 54 | 54 | 0 | 0 |
| `go test ./...` (10 packages) | all | PASS | 0 | 0 |

Exit code: **0 (all pass)**

### Focus suites (plan-mandated)

**`tests/test-sync-skills.sh` — Case G (新規)**

Plan AC7 で要求された「flag 削除 → 再同期で openai.yaml が削除される」回帰ケース。4アサーションすべてパス:

| Assertion | Result |
|-----------|--------|
| G. openai.yaml present after initial sync with disable-model-invocation: true | PASS |
| G. openai.yaml removed after re-sync without disable-model-invocation | PASS |
| G. agents/ dir removed when empty after openai.yaml deletion | PASS |
| G. check-skill-sync.sh passes after flag removal and re-sync | PASS |

**`tests/test-check-skill-sync.sh` — 全 13 ケース**

既存の A–M ケースがすべてパス。新規実装による spec openai.yaml 削除後も check-skill-sync.sh は実ツリーで通過(Case K: real skill tree passes prompts/ parity)。

**`go test ./internal/scaffold/...`**

`internal/scaffold` パッケージ(embed_test.go を含む)が `ok` で完了。`.agents/skills/spec/agents/openai.yaml` 削除後も template-parity assertions に影響なし。

## Coverage

- Statement: Go パッケージ — `go test` pass (no regression)
- Branch: Shell テスト — カバレッジ計測ツールなし; テストケース網羅で代替
- Function: N/A (shell scripts)
- Notes: `internal/scaffold/embed_test.go` の `.agents/skills/` 存在チェックは `openai.yaml` ファイル単位ではなく gitkeep レベルの検証のため、削除による影響なし

## Failure analysis

| Test | Error | Root cause | Proposed fix |
| --- | --- | --- | --- |
| (なし) | — | — | — |

## Regression checks

| Previously broken behavior | Status | Evidence |
| --- | --- | --- |
| `disable-model-invocation` を削除後に再同期しても `openai.yaml` が残存する | FIXED & tested | test-sync-skills.sh Case G: 4/4 PASS |
| `check-skill-sync.sh` が実ツリーで spec openai.yaml 削除後に失敗する | NOT REGRESSED | test-check-skill-sync.sh 13/13 PASS |
| `embed_test.go` が `.agents/skills/spec/agents/openai.yaml` 削除で失敗する | NOT REGRESSED | `go test ./internal/scaffold/...` PASS |

## Test gaps

- **自動起動しきい値の実効性**: description の記述がモデルの実際の自動起動判断にどう反映されるかは runtime/behavioral 特性であり、シェルテストでは検証不能。これはプランの Non-goals 範囲外。
- **`templates/base` 側 openai.yaml 削除の独立確認**: `test-sync-skills.sh` はルート側 sync のみをテスト。templates/base 側は `check-sync.sh` (verify スコープ) で確認済みだが、test スイートに専用ケースなし。

## Verdict

- Pass: YES — 全シェルテストスイートおよび Go テスト (10 パッケージ) がパス。Exit code 0。
- Fail: 0 件
- Blocked: なし

**PASS** — PRへの進行を承認する。
