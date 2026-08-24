# Slice 1 実測証跡: Codex hooks 多イベント発火・deny 実効性(2026-08-24)

対象プラン: `docs/plans/active/2026-08-24-codex-hooks-multi-event.md`
実測環境: メタリポ main checkout(trusted)、codex-cli 0.147.0、model gpt-5.5、
`codex exec --skip-git-repo-check --dangerously-bypass-hook-trust`

## 実験セットアップ

- `.codex/hooks.json` を一時的に実験形へ差し替え(ファイルコピーでバックアップ、
  終了後に復元済み)。全エントリ command は配布形
  `cd "$(git rev-parse --show-toplevel)" && ./.claude/hooks/ralph-dispatch.sh <event>`:
  - `PostToolUse`: matcher `Edit|Write|MultiEdit|apply_patch`(既存)
  - `PreToolUse`: matcher `Bash`
  - `SessionStart`: matcher 省略
  - `UserPromptSubmit`: matcher 省略
- 第 3 層 `.claude/hooks/local/<event>.d/99-probe.sh` に stdin 捕捉プローブ、
  `PreToolUse.d/50-deny-probe.sh` に「payload に DENYPROBE を含むとき deny の
  decision JSON を出す」プローブを設置(終了後に撤去済み)。

## 結果 1: 3 イベントすべて発火し dispatcher 第 3 層まで到達

許可コマンド(`touch .harness/state/me-s1/allowed-marker.txt`)の実行依頼で:

- `SessionStart` 捕捉: `source: "startup"`、keys = session_id / transcript_path /
  cwd / hook_event_name / model / permission_mode / source
- `UserPromptSubmit` 捕捉: `prompt` にユーザプロンプト全文、keys = session_id /
  turn_id / transcript_path / cwd / hook_event_name / model / permission_mode / prompt
- `PreToolUse` 捕捉: `tool_name: "Bash"`、`tool_input: {"command": "touch ..."}`、
  `cwd` あり
- マーカーファイル作成を確認(コマンドは実行された)

transcript_path が codex セッションを指すことで、同一マシンで並行する
Claude Code セッション由来の発火と判別した(Claude 由来の捕捉は
tool_input に description/timeout を含む・transcript_path が `.claude` 配下)。

## 結果 2: PreToolUse の実ツール名は `Bash`

Codex がシェルコマンドを実行するときの PreToolUse payload `tool_name` は
逐語的に `Bash`。`tool_input` はネスト形 `{"command": "..."}` で、
`pre_bash_guard.sh` の `tool_input.command` 抽出とそのまま互換。
matcher `Bash` は拡張不要。

## 結果 3: deny は Codex で実行ブロックとして機能する(ハード AC-4)

deny 対象コマンド(`touch .harness/state/me-s1/DENYPROBE-run4.txt`)の実行依頼で:

```
hook: PreToolUse
ERROR codex_core::tools::router: error=Command blocked by PreToolUse hook: slice-1 deny probe. Command: touch .harness/state/me-s1/DENYPROBE-run4.txt
hook: PreToolUse Blocked
codex
BLOCKED
```

- マーカーファイルは作成されず(ファイルシステム確認: `allowed-marker.txt` のみ存在)
- codex ルーターがコマンドをブロックし、deny 理由がモデルに提示された
- モデルは指示どおり `BLOCKED` と応答し、リトライしなかった

→ **PreToolUse は出荷する**(deny 実効性の実証済み)。

副次確認: 同じ deny プローブは Claude Code 側でも機能した(実験中、
DENYPROBE を含む当セッション自身の Bash コマンドが同一 decision JSON で
ブロックされた)。dispatcher の decision 経路は両 CLI で実効。

## 付随観測

- dispatcher は「first decision wins — run no further scripts」の設計どおり、
  50-deny-probe の decision 後に 99-probe を実行しない(deny 実行時に
  PreToolUse の捕捉ファイルが生成されないことで観測)。
- codex 実行ログの `hook: SessionStart Failed` / `hook: Stop ...` は codex
  プラグイン側 hooks(`~/.codex/plugins/cache/openai-codex/codex/1.0.6/hooks/hooks.json`)
  由来。当方のエントリはすべて Completed で、プローブ捕捉も成功している。
- 実験の教訓: deny トークンを含むコマンドは実験者自身のセッションの
  PreToolUse でもブロックされるため、codex へのプロンプトはトークンを
  文字列連結で組み立てたファイル経由で渡す必要があった。

## 確定事項(プランへ反映)

| 未確定点 | 確定値 |
|---|---|
| Bash 系実ツール名 | `Bash`(matcher `Bash` のままで一致) |
| deny 実効性 | ブロックされる → PreToolUse を出荷イベントに含める |
| 出荷イベント集合 | PostToolUse(既存)+ PreToolUse / SessionStart / UserPromptSubmit |
