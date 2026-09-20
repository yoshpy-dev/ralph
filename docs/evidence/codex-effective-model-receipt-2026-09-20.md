# codex 座席の実効モデルの観測: 実記録での確認(#165)

- Date: 2026-09-20
- Plan: `docs/plans/active/2026-09-20-codex-effective-model-receipt.md`
- 対象: `internal/org/codex_session.go` の `ObserveCodexEffectiveModel`(commit 78692f4 で実行。self-review の修正後の 26b03ce で再実行して同じ結果)
- codex CLI: 0.154.0(記録を作った時点の版)

## 何を確認したか

codex の session 記録から座席の実効モデルを読む観測関数が、fixture だけでなく実際の記録でも期待どおりに動くこと。新しい座席は起動していない。#155(2026-09-18)と #163(2026-09-19)の実機確認で、偽 HOME の `.codex/sessions/` に残っていた記録 5 件を読み取り専用で使った。

記録には会話の本文が入っているので、この確認では model と status だけを出力した。以下に載せているのも、記録の構造(キー名と行番号)、時刻、model だけである。

## 記録の形(実測)

- 場所: `<CODEX_HOME>/sessions/YYYY/MM/DD/rollout-<ローカル時刻>-<uuid>.jsonl`。1 行 1 JSON で、最上位は `timestamp` / `type` / `payload` / `ordinal`
- 1 行目は `session_meta`。`payload.timestamp` が session の開始時刻(UTC、ミリ秒)。`payload.originator` は `codex-tui`
- `turn_context` の `payload.model` が実効モデル(`payload.effort` もある)
- 座席の最初のプロンプトは `response_item`(`payload.type = message`、`payload.role = user`、`payload.content[].text`)。ralph がプロンプトをファイルで渡した座席では、本文に役割指示ファイルの絶対パス(`<state-dir>/prompts/<org>_<seat>.md`)が入る。環境や AGENTS.md の user メッセージが先にあり、それらにはパスは入らない

| 記録(ローカル時刻) | `turn_context` の行 | 役割指示のメッセージの行 | session 開始から `turn_context` まで | 最長の行 | サイズ |
|---|---|---|---|---|---|
| 09-18 15:53:56 | 5 | 6 | 2.5 秒 | 44,488 B | 304 KB |
| 09-18 16:12:15 | 5 | 6 | 3.2 秒 | 44,488 B | 222 KB |
| 09-18 16:13:57 | 7 | 8 | 3.0 秒 | 44,500 B | 197 KB |
| 09-18 16:15:14 | 5 | 6 | 2.3 秒 | 44,489 B | 413 KB |
| 09-19 22:02:22 | 7 | 8 | 2.9 秒 | 28,129 B | 304 KB |

行番号は 0 始まり。役割指示のメッセージは `turn_context` の 0.1〜0.4 秒後に書かれていた。spawn 時の待ちの上限 8 秒と、1 ファイルあたりの読み取り上限 4 MiB・1 行 256 KiB は、この実測に対して余裕がある。

## 観測の結果

コミットしない一時テスト(package `org` の中から `ObserveCodexEffectiveModel` を呼び、`status` と `model` だけを `t.Logf` する)を、sessions ディレクトリと役割指示ファイルのパスを環境変数で渡して実行した。

| ケース | 指定したモデル | spawn 開始時刻として渡した値 | 結果 | 所要 |
|---|---|---|---|---|
| Run E1(real-copy config、#155) | `gpt-5.5` | 07:13:00Z(session 開始 07:13:57Z の前) | `found`、`gpt-5.6-sol` | 11 ms |
| Run E2(minimal config、#155) | `gpt-5.5` | 07:15:00Z | `found`、`gpt-5.5` | 4 ms |
| #163 の codex 座席 | `gpt-6-astra` | 13:02:00Z | `found`、`gpt-6-astra` | 1 ms |
| Run E1 の役割指示ファイル、spawn 開始を session 開始より後に | — | 07:14:30Z | `not-found` | 6 ms |
| 存在しない役割指示ファイルのパス | — | 07:13:00Z | `not-found` | 8 ms |

- Run E1 は #165 のきっかけになった実例で、`--model gpt-5.5` で起動した座席が `gpt-5.6-sol` で動いていた。receipts では `honored=false`、`reported_effective_model=gpt-5.6-sol` になる
- 4 行目は、同じ org・seat で再 spawn したときに古い session を採らないことの確認(session の開始時刻が spawn 開始より前の記録は対象外)
- エラーはどのケースでも nil
- 26b03ce では座席の特定を 2 点変えた: user メッセージは役割指示ファイルのパスだけでなく、ralph が渡す指示文全体(`役割指示を読み込んで従ってください: <パス>`)を含むこと。探す日付ディレクトリは spawn 日の前日・当日・翌日の 3 つだけ。上の 5 ケースは変更後も同じ結果だった(実記録の user メッセージは指示文全体を含んでいる)

## 記録は開始日のディレクトリに残る(メタデータだけで確認)

探すディレクトリを spawn 日の前後 1 日に限ってよい根拠。この環境の実ホームの `~/.codex/sessions/` について、ファイルのパスと更新時刻だけを調べた(内容は読んでいない): 記録 1,013 件のうち 57 件は、ディレクトリの日付より後の日に更新されていた(最大 17 日後)。どれも開始日のディレクトリに残っており、ファイル名の日付はすべてディレクトリの日付と一致した。長く動いた座席を何日も後に stop しても、記録は spawn 日のディレクトリにある。

## `ralph doctor` の表示(この環境の実 cache)

`models_cache.json` の `gpt-5.5` には `upgrade`(移行先 `gpt-5.6-sol`、`retirement_at = 2026-10-14T19:00:00Z`)が入っている。commit 26b03ce のバイナリでの表示:

```
ℹ Org codex model slugs: info — 5 codex model_pool slug(s) present in ~/.codex/models_cache.json; retiring: gpt-5.5 -> gpt-5.6-sol on 2026-10-14. codex may run the replacement instead of the commanded model; when ralph can identify the seat's codex session record, it records the mismatch as honored=false in the org model receipts. (cache written 2026-09-20T01:39:04Z, 0m ago)
```

(パスのホーム部分は `~` に置き換えた。)

## 確認していないこと

- 新しい座席を起動しての end-to-end(spawn 中の待ちで receipt が書かれること、stop 時の追記、CLI の警告)。これらは fixture と herdr stub のテストで確認している
- 退役ダイアログの表示中に session 記録が作られるかどうか。作られない、または `turn_context` がない場合、spawn 時は `unknown` になり、stop 時の観測で拾う設計
- codex CLI 0.154.0 以外の版の記録の形
