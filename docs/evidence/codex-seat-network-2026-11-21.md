# codex の座席の sandbox と外への通信の確かめ(2026-11-21)

issue #362 の計画 `docs/plans/archive/2026-11-21-codex-seat-network.md` のための確かめ。codex-cli 0.166.0、macOS 26.5.2。`codex sandbox` は座席と同じ seatbelt の sandbox でコマンドを走らせる。

## このマシンの設定

`~/.codex/config.toml` は `sandbox_mode = "danger-full-access"` で、`[sandbox_workspace_write]` に `network_access = true` がある(下流の報告が回避に使ったのと同じ設定)。

## 結果

`/tmp` で `/usr/bin/curl -sS -o /dev/null -w '%{http_code}' --max-time 15 https://example.com` を走らせた。

| codex sandbox の上書き | 結果 |
|---|---|
| `-c sandbox_mode=workspace-write` だけ(設定ファイルの `network_access = true` が効く) | 200 |
| `-c sandbox_mode=workspace-write -c sandbox_workspace_write.network_access=true` | 200 |
| `-c sandbox_mode=workspace-write -c sandbox_workspace_write.network_access=false` | `curl: (6) Could not resolve host: example.com`、000 |

`-c sandbox_workspace_write.network_access=<bool>` は設定ファイルの値を上書きする。`false` で通信は止まり、`true` で通る。codex の既定(設定ファイルに鍵がないとき)は `false`(issue #362 の本文、codex の文書)。

## 確かめていないこと

- 座席(`codex --sandbox workspace-write --ask-for-approval never -c sandbox_workspace_write.network_access=true`)として立てた TUI の中で、tflint の初期化が通ること
- 設定ファイルに鍵がないマシンでの既定の値(このマシンは `true` を書いてあるため、`false` の上書きで代わりに確かめた)
