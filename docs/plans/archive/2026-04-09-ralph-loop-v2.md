# Ralph Loop v2 — 完全自律開発パイプライン

- Status: Draft
- Owner: Claude Code
- Date: 2026-04-09
- Related request: Ralph Loop フロー拡張・強化
- Related issue: N/A
- Branch: feat/ralph-loop-v2

## Objective

Ralph Loop を「実装のみ」から「実装→レビュー→検証→テスト→ドキュメント→PR作成」の完全自律パイプラインに進化させる。ユーザーは `/plan` を手動トリガーするだけで、残りはエージェントが完全自律で実行し、PR作成まで到達する。

## Scope

### Phase 1: フルパイプライン自律ループ（Inner/Outer Loop アーキテクチャ）
- `ralph-pipeline.sh` を新規作成し、Inner Loop（実装→self-review→verify→test）と Outer Loop（sync-docs→codex-review→PR）の二重ループ構造を導入
- Inner Loop 内でテストが通るまで自動で再実装サイクルを回す
- Outer Loop で codex-review トリアージ結果に基づき、ACTION_REQUIRED/WORTH_CONSIDERING があれば Inner Loop に差し戻し
- 全件 DISMISSED になった時点で自動 PR 作成

### Phase 2: コンテキスト戦略の改善
- `claude -p` の制限（hooks 非実行）への対策
- `--append-system-prompt` と `--continue` / `--resume` を活用したセッション継続
- イテレーション間のチェックポイントファイル（構造化 JSON）による状態伝搬
- CLAUDE.md / .claude/rules/ をコンテキスト注入の主要手段として活用（ヘッドレスでも読み込まれる）

### Phase 3: マルチエージェント並列開発（垂直スライス並列）
- 計画の垂直スライス分割: 1スライス = 1ワークツリー = 1フルパイプライン
- オーケストレータスクリプトが複数ワークツリーを生成し、各スライスを並列実行
- 各スライスが独立に Inner Loop → Outer Loop → PR を完走
- Ralph Loop 専用の計画テンプレート（スライス定義を含む）

### Phase 4: CLI 統合
- `ralph` CLI wrapper（Bash ベース）
- サブコマンド: `ralph plan`, `ralph run`, `ralph status`, `ralph abort`

## Non-goals

- Claude Code 以外のエージェントバックエンド対応（ralph-cli 的なマルチバックエンド）
- Web ダッシュボード / Telegram 連携
- 標準フロー（/work）の変更 — /work は対話的フローのまま維持
- 既存スキル（self-review, verify, test, sync-docs, codex-review, pr）のロジック変更 — ループからの呼び出し方法のみ変更
- コスト追跡の組み込み（将来の拡張として記録のみ）

## Assumptions

- `claude -p` は CLAUDE.md と .claude/rules/ を読み込む（hooks は実行されない）
- `claude --continue` / `--resume` でセッション状態を跨いだコンテキスト継続が可能
- `--append-system-prompt` で追加指示を注入可能
- 各スキル（self-review, verify, test 等）は `claude -p` 経由でも CLAUDE.md の指示に従い適切に動作する
- Git worktree は並列で複数作成可能

**注意: 上記の前提は全て実装前に preflight probe で検証する（Codex 指摘 #1 対応）。検証失敗時はパイプライン実行をゲートする。**

## Affected areas

### 新規作成
- `scripts/ralph-pipeline.sh` — フルパイプライン オーケストレータ（Inner/Outer Loop）
- `scripts/ralph-orchestrator.sh` — マルチワークツリー並列オーケストレータ
- `.claude/skills/loop/prompts/pipeline-inner.md` — Inner Loop 用プロンプトテンプレート
- `.claude/skills/loop/prompts/pipeline-review.md` — レビュー/検証/テスト用プロンプト
- `.claude/skills/loop/prompts/pipeline-outer.md` — Outer Loop 用プロンプトテンプレート
- `.harness/state/loop/checkpoint.json` — イテレーション間状態ファイル（ランタイム生成）
- `docs/plans/templates/ralph-loop-plan.md` — Ralph Loop 専用計画テンプレート（スライス定義付き）

### 変更
- `scripts/ralph-loop-init.sh` — パイプラインモード初期化対応
- `.claude/skills/loop/SKILL.md` — フルパイプラインモードの記載追加
- `.claude/skills/plan/SKILL.md` — Ralph Loop 選択時のスライス分割フロー追加
- `.claude/rules/subagent-policy.md` — ループ内パイプライン実行ポリシー追記
- `CLAUDE.md` — Ralph Loop v2 の概要参照追加

### 参照のみ（変更なし）
- `.claude/skills/self-review/SKILL.md`
- `.claude/skills/verify/SKILL.md`
- `.claude/skills/test/SKILL.md`
- `.claude/skills/sync-docs/SKILL.md`
- `.claude/skills/codex-review/SKILL.md`
- `.claude/skills/pr/SKILL.md`

## Acceptance criteria

各 AC にはアーティファクトベースの検証証拠を定義する（Codex 指摘 #2 対応）。

- [ ] AC0: **Preflight probe** — `ralph-pipeline.sh --preflight` が `claude -p` の動作（CLAUDE.md 読み込み、`--continue` 継続、`--append-system-prompt` 注入）を検証し、`docs/evidence/preflight-probe.json` に結果を保存する。probe 失敗時はパイプライン実行を拒否する
  - 証拠: `docs/evidence/preflight-probe.json` に各 capability の pass/fail が記録される
- [ ] AC1: `ralph-pipeline.sh --max-iterations 10` が Inner Loop（実装→self-review→verify→test）を自律反復し、テスト通過時に Outer Loop（sync-docs→codex-review）へ自動遷移する
  - 証拠: `checkpoint.json` の `phase` フィールドが `inner` → `outer` へ遷移するログ。`docs/reports/pipeline-execution-*.json` にフェーズ遷移イベントが記録される
- [ ] AC2: Outer Loop で codex-review トリアージ結果に ACTION_REQUIRED がある場合、Inner Loop に自動差し戻しされる
  - 証拠: `checkpoint.json` の `outer_cycle` がインクリメントされ、`phase` が `outer` → `inner` に戻るログ
- [ ] AC3: 全トリアージ結果が DISMISSED の場合、自動で PR が作成される
  - 証拠: `gh pr view --json url` で PR URL が取得可能。`docs/reports/pipeline-execution-*.json` に `pr_created: true` と PR URL が記録される
- [ ] AC4: `claude -p` 経由でも self-review, verify, test の各スキルが適切に動作する（CLAUDE.md/rules によるコンテキスト注入）
  - 証拠: AC0 の preflight probe が pass。各スキルの実行後に対応するレポートファイル（`docs/reports/self-review-*.md`, `docs/reports/verify-*.md`, `docs/reports/test-*.md`）が生成される
- [ ] AC5: `--continue` によるセッション継続で、前イテレーションのコンテキストが引き継がれる
  - 証拠: AC0 の preflight probe でセッション継続テストが pass。`checkpoint.json` に `session_id` が保存され、次イテレーションで同じ ID が使用される
- [ ] AC6: `checkpoint.json` にイテレーション状態（フェーズ、結果、残課題、failure triage）が構造化保存される
  - 証拠: `checkpoint.json` のスキーマバリデーション（`jq` でパース可能 + 必須フィールド存在確認）
- [ ] AC7: Ralph Loop 専用計画テンプレートでスライス定義（スライス名、受入基準、影響ファイル、共有ファイルロックリスト）が記述できる
  - 証拠: `docs/plans/templates/ralph-loop-plan.md` が存在し、スライス定義セクションとロックリストセクションを含む
- [ ] AC8: `ralph-orchestrator.sh` が複数ワークツリーを生成し、各スライスが独立に並列実行される
  - 証拠: `.claude/worktrees/<slice-slug>` ディレクトリが存在し、各ワークツリーの `checkpoint.json` が独立に記録される
- [ ] AC9: 各スライスが独立に Inner Loop → Outer Loop → PR を完走する
  - 証拠: 各スライスの `docs/reports/pipeline-execution-*.json` に `status: complete` と PR URL が記録される
- [ ] AC10: スタック検出（3回連続変更なし）、最大イテレーション到達、ABORT シグナルで安全に停止する。Inner Loop の修正回数上限（デフォルト 5 回/障害）を超えた場合も安全に停止する
  - 証拠: `checkpoint.json` の `status` フィールドが `stuck`/`max_iterations`/`aborted`/`repair_limit` のいずれかに設定される
- [ ] AC11: 既存の `/work` フロー（標準フロー）に影響がない
  - 証拠: `/work` フローの回帰テスト実行ログ
- [ ] AC12: `./scripts/run-verify.sh` が全変更に対してパスする
  - 証拠: `docs/evidence/verify-*.log` に exit code 0 が記録される
- [ ] AC13: **Hook parity** — `ralph-pipeline.sh` のオーケストレータが hooks の安全チェック（シークレット漏洩検出、禁止コマンド検出、未コミット変更警告）を同等に実施する
  - 証拠: `docs/evidence/hook-parity-checklist.json` に各チェックの実施結果が記録される
- [ ] AC14: **Failure triage** — Inner Loop でテスト失敗時に「仮説→修正→期待証拠」の failure triage が `checkpoint.json` に記録される。同一障害の修正回数が上限（デフォルト 5）に達した場合、人間にエスカレーションされる
  - 証拠: `checkpoint.json` の `failure_triage` 配列に各 triage エントリが構造化保存される
- [ ] AC15: **Cleanup on abort** — `ralph abort` がワークツリー、状態ファイルのアーカイブ・削除を実行し、監査ログを `docs/evidence/abort-audit-*.json` に記録する
  - 証拠: `docs/evidence/abort-audit-*.json` にクリーンアップ対象と結果が記録される

## Implementation outline

### Phase 1: Inner/Outer Loop アーキテクチャ（コア）

```
                    ┌─────────────────────────────────────────┐
                    │            ralph-pipeline.sh            │
                    │                                         │
 ┌──────────┐      │  ┌───────── Inner Loop ─────────┐       │
 │ PROMPT +  │──────│→ │ 実装 → self-review → verify  │       │
 │ Plan      │      │  │    → test                    │       │
 └──────────┘      │  │         │                     │       │
                    │  │    fail ↓ pass                │       │
                    │  │    再実装 ←──┘    ↓           │       │
                    │  └──────────────────────────────┘       │
                    │                    ↓ tests pass          │
                    │  ┌───────── Outer Loop ─────────┐       │
                    │  │ sync-docs → codex-review      │       │
                    │  │         │                     │       │
                    │  │  ACTION_REQUIRED ↓ DISMISSED  │       │
                    │  │   → Inner Loop    → PR 作成   │       │
                    │  └──────────────────────────────┘       │
                    └─────────────────────────────────────────┘
```

0. **Preflight capability probe**（Codex 指摘 #1 対応）
   - `ralph-pipeline.sh --preflight` サブコマンドを実装
   - 最小限の `claude -p` 呼び出しで以下を検証:
     a. CLAUDE.md が読み込まれるか（マーカー文字列の応答確認）
     b. `--continue` でセッション継続が機能するか
     c. `--append-system-prompt` で追加指示が注入されるか
   - 結果を `docs/evidence/preflight-probe.json` に保存
   - いずれかが fail の場合、パイプライン実行を拒否しエラーメッセージを出力
   - `ralph-pipeline.sh` の通常実行時も初回に自動で probe を実行

1. **`scripts/ralph-pipeline.sh`** を新規作成
   - Inner Loop: `claude -p` でプロンプト実行 → self-review → verify → test を順次実行
   - 各フェーズは別の `claude -p` 呼び出し（コンテキスト分離、ralphex パターンに準拠）
   - **テスト失敗時の failure triage**（Codex 指摘 #3 対応）:
     a. 失敗情報を `checkpoint.json` の `failure_triage` 配列に構造化記録:
        ```json
        {
          "failure_id": "F001",
          "test_name": "test_auth_flow",
          "hypothesis": "トークンリフレッシュのタイムアウトが短すぎる",
          "planned_fix": "タイムアウト値を5秒→30秒に変更",
          "expected_evidence": "test_auth_flow がパスする",
          "attempt": 1,
          "max_attempts": 5
        }
        ```
     b. 同一障害（同じ test_name）の修正試行回数を追跡
     c. 上限（デフォルト 5 回、`--max-repair-attempts N` で設定可能）に達した場合:
        - `checkpoint.json` の `status` を `repair_limit` に設定
        - パイプラインを安全停止
        - ユーザーへのエスカレーションメッセージを出力
     d. 修正用プロンプトに前回の仮説と失敗理由を注入して再実装
   - テスト通過時: Outer Loop へ遷移
   - Outer Loop: sync-docs → codex-review を実行
   - codex-review トリアージ結果に基づき、必要なら Inner Loop に差し戻し（最大 3 回）
   - 全件 DISMISSED → PR 作成（`claude -p` で `/pr` スキル相当を実行）
   - **パイプライン実行レポート**: 各フェーズ遷移イベントを `docs/reports/pipeline-execution-<timestamp>.json` に記録

2. **プロンプトテンプレート** を Phase 別に作成
   - `pipeline-inner.md`: 実装フェーズ用（受入基準、前回失敗情報、残課題を注入）
   - `pipeline-review.md`: self-review/verify/test 実行指示
   - `pipeline-outer.md`: sync-docs/codex-review/PR 作成指示

3. **`checkpoint.json`** の設計（Codex 指摘 #2, #3 対応で拡張）
   ```json
   {
     "schema_version": 1,
     "iteration": 3,
     "phase": "inner",
     "status": "running",
     "inner_cycle": 2,
     "outer_cycle": 1,
     "last_test_result": "fail",
     "test_failures": ["test_auth_flow", "test_token_refresh"],
     "failure_triage": [
       {
         "failure_id": "F001",
         "test_name": "test_auth_flow",
         "hypothesis": "トークンリフレッシュのタイムアウトが短すぎる",
         "planned_fix": "タイムアウト値を変更",
         "expected_evidence": "test_auth_flow がパスする",
         "attempt": 2,
         "max_attempts": 5,
         "resolved": false
       }
     ],
     "review_findings": [],
     "codex_triage": { "action_required": 0, "worth_considering": 1, "dismissed": 3 },
     "acceptance_criteria_met": ["AC1", "AC3"],
     "acceptance_criteria_remaining": ["AC2"],
     "session_id": "abc123",
     "phase_transitions": [
       { "from": "preflight", "to": "inner", "timestamp": "2026-04-09T10:00:00Z" },
       { "from": "inner", "to": "outer", "timestamp": "2026-04-09T10:15:00Z" },
       { "from": "outer", "to": "inner", "timestamp": "2026-04-09T10:20:00Z", "reason": "codex ACTION_REQUIRED" }
     ]
   }
   ```

### Phase 2: コンテキスト戦略

4. **セッション継続戦略**
   - 各 Inner Loop イテレーションは `claude -p --continue` で前回セッションを継続
   - セッション ID を `checkpoint.json` に保存
   - `--append-system-prompt` で動的コンテキスト注入:
     - 前回イテレーションの結果サマリー
     - 残っている受入基準
     - テスト失敗情報（失敗時）
     - codex-review トリアージ結果（差し戻し時）
   - フェーズ遷移（Inner → Outer）時はセッションをリセット（コンテキスト汚染防止）

5. **hooks 非実行への対策 + Hook parity checklist**（Codex 指摘 #6 対応）
   - `claude -p` では hooks が実行されないが、CLAUDE.md と .claude/rules/ は読み込まれる
   - **Hook parity checklist**: オーケストレータが以下のチェックを各イテレーション後に実行:
     | Hook | Parity check in orchestrator |
     |------|------------------------------|
     | `pre_bash_guard.sh` — シークレット漏洩検出 | `checkpoint.json` + コミットメッセージを `commit-msg-guard.sh` で検査 |
     | `pre_bash_guard.sh` — 禁止コマンド検出 | `--allowedTools` で Bash ツールの範囲を制限 |
     | `post_edit_verify.sh` — 編集後の検証フラグ | Inner Loop の verify フェーズが同等の役割を果たす |
     | `session_end_summary.sh` — 未コミット変更チェック | 各イテレーション後に `git status --porcelain` を実行 |
     | `precompact_checkpoint.sh` — コンパクション前チェックポイント | `checkpoint.json` が同等の役割を果たす |
   - チェック結果を `docs/evidence/hook-parity-checklist.json` に記録
   - いずれかのチェックが fail した場合、イテレーションを停止しログに記録
   - `--allowedTools` で許可ツールを明示的に制限

### Phase 3: マルチエージェント並列開発

6. **Ralph Loop 専用計画テンプレート + Shared-file locklist**（Codex 指摘 #4 対応）
   ```markdown
   ## Vertical slices

   ### Shared-file locklist
   Files that must not be modified by parallel slices simultaneously.
   If a slice needs to modify a locked file, it must run sequentially after
   all other slices touching that file have completed.
   - `CLAUDE.md`
   - `.claude/rules/subagent-policy.md`
   - `scripts/ralph-pipeline.sh`
   - (auto-detected from affected files overlap)

   ### Slice 1: <name>
   - Acceptance criteria: [...]
   - Affected files: [...]
   - Dependencies: none | [slice N]

   ### Slice 2: <name>
   - Acceptance criteria: [...]
   - Affected files: [...]
   - Dependencies: [slice 1]
   ```

7. **`scripts/ralph-orchestrator.sh`** を新規作成
   - 計画ファイルからスライス定義を読み取り
   - **Shared-file locklist の自動検出**（Codex 指摘 #4 対応）:
     - 各スライスの affected files を比較し、重複ファイルを自動で locklist に追加
     - locklist に含まれるファイルを変更するスライスは順次実行を強制
   - 依存関係のないスライスを並列実行:
     - 各スライスごとに `git worktree add .claude/worktrees/<slice-slug>`
     - 各ワークツリーで `ralph-pipeline.sh` を独立実行
   - 依存関係のあるスライスは前提スライス完了後に開始
   - **統合マージチェック**: 全スライス完了後、PR 作成前に:
     a. 各ワークツリーのブランチを試験的にマージ（`git merge --no-commit --no-ff`）
     b. コンフリクト検出時はユーザーにエスカレーション
     c. コンフリクトなしの場合のみ PR 作成に進行
   - マージ戦略: スライスごとの個別 PR（デフォルト）or 統合 PR（`--unified-pr` フラグ）

8. **`/plan` スキルの拡張**
   - Ralph Loop 選択時に追加ステップ: スライス分割の提案・確認
   - スライス間の依存関係グラフを計画ファイルに記録
   - 並列度の推奨（ワークツリー数 = 独立スライス数）

### Phase 4: CLI 統合

9. **`ralph` CLI wrapper**（Phase 1-3 の上位レイヤー）
   - `ralph plan` → 対話的計画
   - `ralph run [--slices N] [--max-iterations N]` → パイプライン実行
   - `ralph status` → 全スライスの進捗表示
   - `ralph abort [--slice N]` → 安全停止 + **明示的クリーンアップ**（Codex 指摘 #5 対応）:
     a. 実行中のパイプラインプロセスを安全に停止
     b. `.harness/state/loop/` の状態ファイルを `.harness/state/loop-archive/<timestamp>/` にアーカイブ
     c. 孤立ワークツリーを検出・削除（`git worktree list` + `git worktree remove`）
     d. 監査ログを `docs/evidence/abort-audit-<timestamp>.json` に記録:
        ```json
        {
          "timestamp": "2026-04-09T10:30:00Z",
          "reason": "user_abort",
          "archived_state": ".harness/state/loop-archive/20260409-103000/",
          "worktrees_removed": [".claude/worktrees/slice-1"],
          "checkpoint_at_abort": { ... }
        }
        ```
   - 初期は Bash スクリプト群のラッパー

## Verify plan

- Static analysis checks:
  - `shellcheck` で全新規シェルスクリプトを検証
  - 既存スクリプトとの互換性確認（ralph-loop.sh の後方互換）
- Spec compliance criteria to confirm:
  - AC1-AC12 の各基準を個別に検証
  - 既存の `/work` フローが影響を受けないことを確認
- Documentation drift to check:
  - CLAUDE.md, AGENTS.md, subagent-policy.md が新フローを正確に反映
  - loop/SKILL.md が新機能を正確に記述
- Evidence to capture:
  - `ralph-pipeline.sh --dry-run` の出力ログ
  - 実際のパイプライン実行ログ（テストプロジェクトでの E2E 実行）
  - checkpoint.json のサンプル出力

## Test plan

- Unit tests:
  - `ralph-pipeline.sh --dry-run` が正しいフェーズ遷移を出力
  - `checkpoint.json` の読み書きが正しい
  - スタック検出ロジックのテスト
- Integration tests:
  - Inner Loop: テスト失敗→修正→テスト通過のサイクル確認
  - Outer Loop: codex-review 結果に基づく差し戻しと再実行確認
  - セッション継続: `--continue` でのコンテキスト引き継ぎ確認
- Regression tests:
  - `ralph-loop.sh`（既存）が後方互換で動作
  - `/work` フローが影響を受けない
  - 既存の hooks が標準フローで正常に動作
- Edge cases:
  - Codex CLI 利用不可時のスキップ動作
  - 全テストが初回から通過する場合（Inner Loop 1回で Outer Loop 遷移）
  - 全 codex-review 指摘が初回から DISMISSED の場合（即 PR 作成）
  - ワークツリー作成失敗時
  - 最大イテレーション到達時の安全停止と状態保存
- Evidence to capture:
  - 各テストの実行ログ
  - パイプライン全体の実行時間とイテレーション数

## Risks and mitigations

| Risk | Impact | Likelihood | Mitigation |
|------|--------|------------|------------|
| `claude -p` でスキル指示が適切に動作しない | High | Medium | **Preflight probe で事前検証**（AC0）。fail 時はパイプライン実行を拒否。CLAUDE.md/rules での明示的指示 + `--append-system-prompt` でのフェーズ別コンテキスト注入 |
| コスト爆発（Outer Loop 差し戻し無限ループ化） | High | Medium | Outer Loop 最大差し戻し回数を制限（デフォルト 3 回）。超過時は人間にエスカレーション |
| Inner Loop の修復振動（テスト失敗→修正→別の失敗→...） | High | Medium | **Failure triage で仮説ベースの修復戦略**（AC14）。同一障害の修正回数上限（デフォルト 5 回）。超過時は `repair_limit` ステータスで安全停止 |
| セッション継続のコンテキスト劣化 | Medium | Medium | フェーズ遷移時にセッションリセット。チェックポイントで状態を明示的に伝搬 |
| 並列ワークツリーのマージコンフリクト | High | High | **Shared-file locklist で重複検出**（AC7）。重複ファイルを持つスライスは順次実行を強制。PR 作成前に統合マージチェック。コンフリクト時は人間にエスカレーション |
| hooks 非実行による安全性低下 | Medium | High | **Hook parity checklist**（AC13）でオーケストレータが同等チェックを実施。結果を `docs/evidence/hook-parity-checklist.json` に記録 |
| 失敗パイプライン実行の状態汚染 | Medium | Medium | **`ralph abort` による明示的クリーンアップ**（AC15）。状態アーカイブ + ワークツリー削除 + 監査ログ |

## Rollout or rollback notes

### 段階的ロールアウト
1. Phase 1（Inner/Outer Loop）を先行実装・検証
2. Phase 2（コンテキスト戦略）を Phase 1 と並行して検証
3. Phase 3（マルチエージェント）は Phase 1+2 が安定してから
4. Phase 4（CLI）は Phase 1-3 のラッパーとして最後に

### ロールバック
- 既存の `ralph-loop.sh` はそのまま維持（後方互換）
- 新スクリプトは別ファイル（`ralph-pipeline.sh`, `ralph-orchestrator.sh`）で追加
- `/loop` スキルは新旧両方のモードに対応
- **失敗時のクリーンアップ**（Codex 指摘 #5 対応）:
  - `ralph abort` で状態ファイルをアーカイブし、ワークツリーを削除
  - 監査ログを `docs/evidence/abort-audit-*.json` に記録
  - 孤立ワークツリーの自動検出・削除（`ralph status --check-orphans`）
  - 部分的に完了したパイプラインの状態は `checkpoint.json` に保存され、`ralph run --resume` で再開可能

## Open questions

1. 並列スライスの最大数の推奨値 — API レート制限とローカルリソースに依存
2. codex-review の差し戻し上限（デフォルト 3 回）は適切か — 実運用で調整
3. Phase 4 の CLI は Bash で十分か — 初期は Bash、規模拡大時に移行検討

## Progress checklist

- [x] Plan reviewed
- [x] Branch created (feat/ralph-loop-v2)
- [x] Phase 1: Inner/Outer Loop 実装 (ralph-pipeline.sh, prompt templates, ralph-loop-init.sh --pipeline)
- [x] Phase 2: コンテキスト戦略実装 (checkpoint.json, session continuation, hook parity — embedded in ralph-pipeline.sh)
- [x] Phase 3: マルチエージェント並列実装 (ralph-orchestrator.sh, ralph-loop-plan.md template)
- [x] Phase 4: CLI ラッパー実装 (scripts/ralph with plan/run/status/abort)
- [x] ドキュメント更新 (loop/SKILL.md, plan/SKILL.md, subagent-policy.md, CLAUDE.md, AGENTS.md)
- [x] Review artifact created (docs/reports/self-review-2026-04-10-ralph-loop-v2.md)
- [x] Verification artifact created (docs/reports/verify-2026-04-10-ralph-loop-v2.md)
- [x] Test artifact created (docs/reports/test-2026-04-10-ralph-loop-v2.md)
- [ ] PR created
