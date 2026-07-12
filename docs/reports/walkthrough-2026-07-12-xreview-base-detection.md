# Walkthrough: xreview-base-detection

- Date: 2026-07-12
- Plan: docs/plans/archive/2026-07-12-xreview-base-detection.md
- Branch: fix/xreview-base-detection (base: main @ 84c1b6e)
- Diff size: 21 files, +732 / -11(うち約6割はプラン/レポート成果物)

## 読む順序(推奨)

1. `scripts/ralph-cli-driver.sh` — 新ヘルパー `detect_base_branch`。解決順:
   `RALPH_XREVIEW_BASE`(明示指定)> `git symbolic-ref
   refs/remotes/origin/HEAD`(リポジトリのデフォルトブランチ)>
   main/master フォールバック(既存 `default_branch()` と同セマンティクス)
2. `scripts/ralph-pipeline.sh` — cross-review ゲートのベース検出を
   `HEAD@{upstream}`(追跡ref)からヘルパー呼び出しへ。push 済みブランチで
   diff が空になり **cross-review がサイレントスキップされるバグ**の修正
3. `scripts/ralph-orchestrator.sh` — Loop 起動ブランチ(`_base_branch`)を
   `RALPH_XREVIEW_BASE` として export(Codex plan advisory HIGH:
   `develop`/`release/*` 起動の Loop ではリポジトリデフォルトではなく
   起動ブランチが正しいマージターゲット)。cycle 1 cross-review 指摘で
   「オペレーター指定値を保持する」preserve-form に修正
4. `tests/test-ralph-cli-driver.sh` Test 14(11 アサーション)— ゲートの
   エンドツーエンド証明: 同一フィクスチャで旧検出は空 diff(= スキップ)、
   新ヘルパーは非空 diff(= レビュー発火)。ほか明示指定優先・非 main
   デフォルト・`origin/release/1.0` エッジ・main/master フォールバック・
   worktree 共有 refs
5. cross-review SKILL ×4、レシピ env 表(`RALPH_XREVIEW_BASE` 行)、
   AGENTS.md 関数リスト、tech-debt RESOLVED 行

## 設計判断の要点

- ゲートの `git diff base...HEAD --quiet` は「diff 失敗 = 変更あり」として
  レビューを実行する fail-open-to-review — 安全側の既存挙動を維持
- Codex plan advisory 4件・cross-review 指摘1件(計5件)を全採用。経緯は
  plan の advisory セクションとトリアージレポートに記録

## エビデンス

- Self-review: docs/reports/self-review-2026-07-12-xreview-base-detection.md(MERGE ×2 cycles)
- Verify: docs/reports/verify-2026-07-12-xreview-base-detection.md(PASS ×2)
- Test: docs/reports/test-2026-07-12-xreview-base-detection.md(103/103 driver + xreview 21/54 + orchestrator + フル回帰 0 失敗)
- Cross-review: docs/reports/cross-review-triage-xreview-base-detection.md(cycle 1: 1件→修正、cycle 2: 指摘なし)
