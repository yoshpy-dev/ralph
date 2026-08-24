# Walkthrough: codex-hooks-multi-event(2026-08-24)

- Branch: feat/codex-hooks-multi-event(base: main)
- Diff 規模: 23 files, +1617 / -122
- Plan: docs/plans/archive/2026-08-24-codex-hooks-multi-event.md(/pr でアーカイブ)

## 変更の骨子

配布 `.codex/hooks.json` を PostToolUse 単独から 4 イベント
(PostToolUse + PreToolUse[matcher `Bash`] + SessionStart + UserPromptSubmit)へ
拡張し、doctor / CI ゲート / ドキュメントを追随させた。SessionEnd / PreCompact は
自動 WIP コミット副作用のため意図的に配線しない(tech-debt 登録済み)。

## 読み順ガイド

1. **`.codex/hooks.json` + `templates/base/.codex/hooks.json`**(byte-identical)—
   3 イベント追加。既存 PostToolUse エントリの command 文字列は 1 文字も不変
   (Codex trust は per-command-hash のため、既存承認を維持)。
2. **`docs/evidence/codex-hooks-multi-event-slice1-2026-08-24.md`** — 出荷判断の
   実測根拠。3 イベントの発火+第 3 層到達、実ツール名 `Bash`、
   **deny 実効性**(`hook: PreToolUse Blocked`、コマンド不実行をファイルシステム
   状態で確認 — ハード AC-4)。
3. **`docs/evidence/codex-hooks-multi-event-fixture-2026-08-24.md`** — fresh
   `ralph init` fixture でも 3 イベント発火(AC-3)。
4. **`internal/cli/doctor.go`** — `validateCodexHooksJSON` を
   `codexShippedHookEvents` 集合検査へ拡張。ルーティング判定は
   `ralph-dispatch.sh <当該イベント名>`(引用符許容・語境界付き regexp
   `dispatchEventArgRes`)の照合を要求(cycle-2 cross-review AR#1 対応)。
   イベント集合と実配布 hooks.json の同期は
   `TestCodexShippedHookEventsMatchesShippedHooksJSON` が固定。
5. **`tests/test-hook-wiring.sh`**(66→68 checks)— 4 イベントを単一ループで
   event 引数+境界照合(doctor と同一意味論、引用符も同等に許容)。
   SessionEnd / PreCompact の**不在**も機械的にゲート。
6. **`internal/cli/doctor_hooks_test.go` / `cli_test.go`** — per-event 欠落
   warn の negative、誤イベント引数(PreToolUse→PostToolUse)検出、
   prefix 衝突(PostToolUseExtra)拒否、引用符許容の各テスト。
7. **`.codex/README.md` / `.codex/hooks/README.md`(+twins)** — 4 イベント記述、
   PostToolUseFailure 非対応(Claude 固有)、trust 再承認 UX、
   SessionStart の冪等な housekeeping 副作用の正確な記述。
8. **`docs/tech-debt/README.md`** — 2 行追加: SessionEnd/PreCompact 分離
   ロールアウト、Codex `ask` 意味論未実証。+1 行: insight cycle スタンプ機構。
9. **`docs/insights/events/2026-08-24-codex-hooks-multi-event.jsonl`** —
   3 cycle 分のパイプライン記録(cycle-2 sync_docs の誤スタンプは訂正済み)。

## パイプライン履歴

- cycle 1: self-review M1〜M3+L1 修正、cross-review AR#1(doctor イベント引数照合)→ Fix
- cycle 2: AR#1 修正+self-review C2 群修正。cross-review AR#1(insight cycle 誤スタンプ)→ cap 到達、操作者承認で cap 2→3
- cycle 3: 1 行訂正+引用符非対称修正。cross-review **所見ゼロ**

## Known gaps

- Codex `ask` decision の実効性は未実証(deny のみ実証)— tech-debt 記録済み
- live-fire 証跡は codex-cli 0.147.0 に固定(決定的テストでは再実行されない)
