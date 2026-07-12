# Walkthrough: standard-flow-orchestrator

- Date: 2026-07-11
- Plan: docs/plans/archive/2026-07-11-standard-flow-orchestrator.md
- Branch: feat/standard-flow-orchestrator (base: main @ 45e9060)
- Diff size: 26 files, +1237 / -50(うち約半分はレポート/プラン成果物)

## 読む順序(推奨)

1. `.claude/agents/implementer.md` — 新設の実装ワーカー(`model: sonnet`)。
   構造化ハンドオフの必須フィールド、ベースライン規律(scope 外 dirt は
   許容・scope 重複のみ STOP)、ステージング allowlist、レポート契約
2. `.claude/skills/work/SKILL.md` — step 6(委譲 + ディスパッチ前の記帳
   コミット)と step 7(two-mode Validation Gate)
3. `.claude/rules/model-routing.md` — "Standard flow delegation (/work)"
   セクション(ハンドオフ表・inline 例外・エスカレーション方針)
4. `.claude/rules/subagent-policy.md` — implementer 委譲ポリシー
5. `.claude/skills/cross-review/SKILL.md` — `--model opus` ハードコード解消
6. `.codex/agents/implementer.toml` / `.codex/README.md` — Codex 側パリティ
7. `.claude/rules/git-commit-strategy.md` — 委譲スライスのコミット所有権

## 変更の核

- **オーケストレーター規律**: メインセッション(セッションモデル =
  Fable 5 等)は分解・ハンドオフ作成・レポート裁定・最終レビューに専念し、
  実装スライスは `implementer`(sonnet)へ。品質はモデルティアではなく
  ハンドオフの精度(受け入れ基準・検証コマンド・レポート契約)と、変更
  なしのポスト実装パイプライン(reviewer=opus / verify / test /
  cross-review)で担保する
- **two-mode Validation Gate**: 委譲スライスは implementer が検証+コミット
  を所有し、オーケストレーターはレポート裁定(検証証拠、コミット境界、
  `git rev-parse HEAD` == 報告 SHA)。インラインスライスは従来の
  verify→stage→commit ゲート
- **cross-review モデルノブ**: claude レビュアーパスが
  `${RALPH_CLAUDE_REVIEWER_MODEL:-opus}` を参照(env 経由、フォールバック
  opus — 挙動不変)
- **4コピー同期**: skills は root/.claude ↔ root/.agents ↔ templates/base
  の両系統、agents/README は root ↔ templates/base。全同期ゲート通過

## クロスレビュー履歴(3サイクル)

- Cycle 1: ACTION_REQUIRED 2件(記帳 dirt でディスパッチ停止/二重コミット)
  → 修正(a910547, 1de7a20)
- Cycle 2: WORTH_CONSIDERING 1件(delegated gate の "clean" 文言矛盾)→
  cap を 3 に一時引き上げて修正(435ce16)、フルパイプライン再実行
- Cycle 3: WORTH_CONSIDERING 2件(README の "(sonnet)" 過大表記/
  `git log -1` の SHA 検証弱)→ cap-reached オプション2を選択: 文言修正
  適用(99df6db)+決定論ゲート再実行、4周目のフルパイプラインは実施せず
  (逸脱としてトリアージレポートに記録)

## 既知のギャップ

- `Task(subagent_type="implementer")` の実行時ディスパッチはセッション
  レジストリの制約(セッション開始時にプロジェクトルートから読込)により
  本ブランチ内では検証不能 — merge 後の新セッションで確認(Claude/Codex
  両方)。テストレポートに記録済み
- テストフィクスチャ `cli-stubs/codex` の stdin ハング(本 PR のスコープ外、
  docs/tech-debt/README.md に記録)

## エビデンス

- Self-review: docs/reports/self-review-2026-07-11-standard-flow-orchestrator.md(MERGE ×3 cycles)
- Verify: docs/reports/verify-2026-07-11-standard-flow-orchestrator.md(PASS ×3)
- Test: docs/reports/test-2026-07-11-standard-flow-orchestrator.md(710/710 + Go ok ×3)
- Cross-review: docs/reports/cross-review-triage-standard-flow-orchestrator.md
