# org-runtime-retire-loop

- Status: Draft
- Owner: Claude Code
- Date: 2026-08-03
- Related request: docs/specs/2026-08-01-org-runtime.md(PR 系列⑤: 旧系撤去+ドキュメント全面改稿、FR-11)
- Related issue: N/A
- Type: refactor
- Branch: refactor/org-runtime-retire-loop

## Objective

org runtime PR⑤(最終段): Ralph Loop 自律実行系(orchestrator / pipeline / loop skill / TUI / スライス状態リーダー)を一括撤去し、org runtime を唯一の自律実行面としてドキュメントを全面改稿する。標準フローの開発パイプライン skill 群(/spec /plan /work /self-review /verify /test /sync-docs /cross-review /pr)は「開発ハーネス」として存続する(ユーザー確定 — spec FR-11 の適用範囲精査の結果。spec 側をこの決定に合わせて改訂)。あわせて PR④ known gaps 2 件と旧 shell CLI 退役 tech-debt をクローズする。

## Scope

### 削除(コード)
- `scripts/`: `ralph-orchestrator.sh`(59k)、`ralph-pipeline.sh`(59k)、`ralph-loop-init.sh`、`ralph-loop.sh`、`ralph-status-helpers.sh`、`build-tui.sh`、`new-ralph-plan.sh`、旧 shell CLI `scripts/ralph`(退役 tech-debt row のクローズ)
- `scripts/ralph-cli-driver.sh`: loop 専用部(`run_agent` / `resolve_phase_model` / `write_model_receipt`)を削除し、`/cross-review` が使う `detect_base_branch` / `pick_reviewer` / `count_triage_findings` を `scripts/xreview-helpers.sh`(新名、grep-able)に移設。cross-review skill の参照を更新
- Go: `cmd/ralph-tui`、`internal/ui`(+panes)、`internal/state`(スライス/checkpoint/orchestrator リーダー)、`internal/action`(retry/abort)、`internal/watcher`(TUI 用 fsnotify)
- `internal/cli`: `run`(orchestrator 起動)/ `retry` / `abort` サブコマンド削除。`status` は org manifest ベースへ全面書き換え(全 org のロースター要約+watch-status。`--org-id` で絞り込み)。`doctor` の loop driver 表示削除
- `internal/config`: `[loop]`・`[pipeline]`(phases 含む)セクションと対応 env(`RALPH_LOOP_*`、フェーズ別 `RALPH_*_MODEL` — `RALPH_STANDARD_MAX_PIPELINE_CYCLES` と `RALPH_CLAUDE_REVIEWER_MODEL` は標準フロー用に存続)。`ralph-config.sh` 両ミラー、`templates/base/ralph.toml`、`defaults_sync_test` 追従
- `tests/`: loop/pipeline/orchestrator/xreview-render 系シェルスイートの削除(cross-review の存続機能に対応するテストは移設)
- `templates/base/` の上記全ミラー、`docs/plans/templates/ralph-loop-*`、`docs/recipes/ralph-loop*`

### 削除・改稿(スキル/ルール/ドキュメント)
- `/loop` skill 削除(4 面ミラー)。`/plan` からフロー選択(標準 vs Ralph Loop)を削除し、常に標準フロー+必要に応じ org runtime(/org)を案内する記述へ
- `.claude/rules/`: `post-implementation-pipeline.md`(Ralph Loop 節・integration pipeline 節を削除、標準フロー正準順序は存続)、`subagent-policy.md`(/loop 節削除)、`model-routing.md`(Ralph Loop per-phase routing 節削除、receipts 記述は org 側へ言及変更)
- AGENTS.md(Primary loop を「標準フロー(開発ハーネス)+ org runtime(自律実行面)」の 2 面構成に改稿、repo map 追従)、CLAUDE.md(loop 記述削除、org 追記)、README(Quick start / Operating loop 改稿)、`docs/quality/definition-of-done.md`(Loop 節削除)
- spec FR-11 の適用範囲注記(開発ハーネス存続の決定)
- tech-debt: 旧 shell CLI row・Ralph Loop deviation row 等のクローズ、PR④ known gaps 2 行のクローズ

### 修正(コード、小)
- **PR④ known gap #5**: デッドマンのプローブ復旧誤クリア — pending record にプローブ可用性を明示保持し、不可用時ベースラインとの比較を activity 扱いしない
- **PR④ known gap #6**: `WatchdogJoined` を join 成功時のみ true に

### AC 検証(spec FR-11)
- `grep -rE "ralph-orchestrator|ralph-pipeline|RALPH_LOOP_DRIVER" --include="*.sh" --include="*.go" --include="*.toml"` が参照ゼロ(docs/plans/archive・docs/specs・docs/reports・docs/insights の履歴文書は除外)

## Non-goals

- 標準フロー開発ハーネス(skill 群・run-verify/run-test・worktree スクリプト・insights)の変更(存続)
- org runtime への新機能追加
- `check-pipeline-sync.sh` は標準フロー正準順序の同期ゲートとして存続(参照リストから loop 系を除去)
- `ralph upgrade` の旧ファイル自動削除ロジック(hash-diff エンジンが TEMPLATE 削除を remove として扱う既存挙動に任せる。動作確認のみ)

## Assumptions

- 削除対象への参照は本 worktree 内の grep で全列挙できる(隠れ参照は AC の grep ゼロ化で検出)
- `ralph insights` は `docs/insights/events/` と receipts を読むのみで pipeline スクリプトに依存しない(要確認、依存があれば読み口だけ維持)
- 下流プロジェクトは `ralph upgrade` で loop 系テンプレートが remove 扱いになる(破壊的変更として PR body に明記)

## Affected areas

上記 Scope の通り(削除 ~20 ファイル+改稿 ~15 ファイル+ミラー)。触らない: `internal/org`(gap fix 2 件のみ)、`internal/scaffold`、`internal/upgrade`、`internal/insights`、標準フロー skills。

## Design decisions

- **撤去範囲は Ralph Loop 系のみ**(ユーザー確定): 標準フロー skill 群は開発ハーネスとして存続。org の reviewer/qa 座席がこれらの検証スクリプト/skill を実行する関係でもある。spec は本決定に合わせ改訂。
- **フロー**: Standard flow (/work)(ユーザー確定)。
- **cross-review の driver 依存は移設**: `pick_reviewer` 等は Loop 非依存の機能として `xreview-helpers.sh` へ(削除でなく移設 — cross-review は存続するため)。
- Critical forks: 上記 2 点を解決済み。

## Acceptance criteria

- [ ] AC-1: 削除対象スクリプト・Go パッケージ・skill・テンプレートが本体/templates 両面から消え、`go build ./...` green。
- [ ] AC-2: `ralph status` が org manifest ベースで動作(org 一覧+ロースター+watch-status。`--org-id` 絞り込み。manifest なし環境では案内表示)。
- [ ] AC-3: `[loop]`/`[pipeline]` 設定と対応 env が 3 面から消え、`defaults_sync_test` green。存続 env(`RALPH_STANDARD_MAX_PIPELINE_CYCLES`/`RALPH_CLAUDE_REVIEWER_MODEL`)は動作不変。
- [ ] AC-4: `/cross-review` が移設ヘルパーで従来通り機能(detect_base_branch / pick_reviewer / count_triage_findings のテスト移設)。
- [ ] AC-5: 参照ゼロ grep(履歴文書除外)が pass し、CI の全 sync ゲート(check-sync / check-skill-sync / check-pipeline-sync)green。
- [ ] AC-6: AGENTS.md / CLAUDE.md / README / rules / definition-of-done / spec が新構成(標準フロー=開発ハーネス、org runtime=自律実行面)で整合。
- [ ] AC-7: PR④ known gaps #5/#6 がテスト付きで修正され、tech-debt 行クローズ(旧 shell CLI row 含む)。
- [ ] AC-8: `go test ./...` / `./scripts/run-verify.sh` green。`ralph init` のスキャフォールドから loop 系が消えることをスモーク確認。

## Implementation outline

1. **Slice 1 — PR④ known gap fixes**(internal/org/watch.go + tests、独立)
2. **Slice 2 — Loop 実行系スクリプト削除**: scripts 群+tests+xreview-helpers 移設+cross-review skill 参照更新+config [loop]/[pipeline] 3 面削除
3. **Slice 3 — Go 面削除+status 書き換え**: cmd/ralph-tui, internal/{ui,state,action,watcher}, cli run/retry/abort 削除、status の org 化、doctor 追従
4. **Slice 4 — skills/rules/テンプレート/ドキュメント改稿**: /loop 削除、/plan 改稿、rules 3 本改稿、AGENTS/CLAUDE/README/DoD/spec、recipes、tech-debt クローズ
5. **Slice 5 — 参照ゼロ sweep+スモーク**: grep AC、run-verify full、`ralph init` スキャフォールド確認、walkthrough 素材

## Verify plan

- Static: run-static-verify.sh full。spec FR-11 と本 plan の Design decisions の整合。参照ゼロ grep。sync ゲート 3 種。
- Doc drift: 全改稿ドキュメントの相互整合(post-implementation-pipeline.md の「Where this order is referenced」リスト更新含む)。

## Test plan

- Unit: gap fix 2 件、status の org 表示、xreview-helpers 移設分。
- Integration: `ralph init` を temp dir へスキャフォールドし loop 系不在+org 系存在を確認。
- Regression: 全 Go テスト+存続シェルスイート。
- Edge: manifest なしでの `ralph status`、非 git cwd。

## Risks and mitigations

| リスク | 影響 | 緩和 |
|---|---|---|
| 隠れ参照の削除漏れ | CI red / 実行時破損 | AC-5 の参照ゼロ grep+sync ゲート+full verify |
| cross-review の機能退行 | 開発ハーネス破損 | ヘルパー移設+テスト移設+本 PR 自身のパイプラインで実地検証される |
| 下流プロジェクトへの破壊的変更 | upgrade 時の混乱 | PR body に BREAKING 明記、upgrade の remove 挙動をスモーク確認 |
| ドキュメント改稿の不整合 | 参照リンク切れ | post-implementation-pipeline.md の参照先リストを索引として全数更新 |

## Rollout or rollback notes

- 削除中心のため revert で完全復元可能。スライス別 green ゲート維持(各スライス単独で build/test green)。
- 下流には BREAKING(loop 系テンプレート消滅)。`ralph upgrade` が remove として提示する。

## Open questions

- `ralph insights` の pipeline receipts 依存の有無(Slice 3 で確認、依存があれば読み口のみ残す)。

## Progress checklist

- [ ] Plan reviewed
- [x] Branch created (refactor/org-runtime-retire-loop)
- [ ] Implementation started
- [ ] Review artifact created
- [ ] Verification artifact created
- [ ] Test artifact created
- [ ] PR created

## Readiness checklist

- [x] 撤去範囲確定(Ralph Loop 系のみ、ユーザー確定)・フロー確定(Standard /work)
- [x] cross-review の driver 依存を事前特定(移設方針決定済み)
- [ ] Codex plan advisory(次ステップ)
