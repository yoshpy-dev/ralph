# AC-3 fixture live-fire 証跡: fresh `ralph init` での多イベント発火(2026-08-24)

対象プラン: `docs/plans/active/2026-08-24-codex-hooks-multi-event.md`
対象コミット: c8dfdf2(4 イベント配線)を含む worktree からビルドした `ralph` バイナリ

## 手順

1. worktree で `go build -o <tmp>/ralph ./cmd/ralph`
2. `<tmp>/proj` に `git init` → `ralph init --yes .` → 初期コミット
   - 生成された `.codex/hooks.json` のイベント集合を確認:
     `['PostToolUse', 'PreToolUse', 'SessionStart', 'UserPromptSubmit']`
     (embed テンプレートに 4 イベント形が反映されている)
3. 第 3 層 `.claude/hooks/local/{PreToolUse,SessionStart,UserPromptSubmit}.d/99-probe.sh`
   に stdin 捕捉プローブを設置
4. fixture git root で
   `codex exec --skip-git-repo-check --dangerously-bypass-hook-trust`
   により `touch .harness/state/fixture-marker-2.txt` を実行依頼

## 結果

3 イベントすべてで dispatcher → 第 3 層プローブの実行を確認
(捕捉 payload の transcript_path は codex セッション、cwd は fixture git root):

| イベント | 捕捉内容 |
|---|---|
| `SessionStart` | 発火・捕捉(cwd = fixture root) |
| `UserPromptSubmit` | `prompt` にプロンプト全文 |
| `PreToolUse` | `tool_name: "Bash"`、`tool_input.command: "touch .harness/state/fixture-marker-2.txt"` |

- マーカーファイル作成を確認(許可コマンドは実行される)
- codex 実行ログ: `hook: PreToolUse Completed` / `hook: SessionStart Completed` /
  `hook: UserPromptSubmit Completed`(`SessionStart Failed` 1 件は codex
  プラグイン側 hooks 由来で、当方エントリではない — Slice 1 証跡と同じ観測)
- 未信頼 fixture のため bypass フラグを使用(対話 codex セッションでは
  per-command trust 承認が必要 — README に記載済み)

fixture ディレクトリ(セッション scratchpad 配下)は証跡取得後に削除し、
fixture 生成物は本リポジトリへ一切コミットしていない。

meta-repo での同型実測(deny 実効性含む)は
`docs/evidence/codex-hooks-multi-event-slice1-2026-08-24.md` を参照。
