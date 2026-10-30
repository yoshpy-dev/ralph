# `claude -p --effort` の確かめ

- Date: 2026-10-30(JST 16:36〜16:37)
- Plan: `docs/plans/active/2026-10-30-standard-flow-config.md`(Assumptions、D4、AC4)
- Issue: #276 の受け入れ条件「`claude -p --effort` が効くかを確かめた結果が、PR か report に残っている」
- 計画の Progress の「Plan approved」の下にある記録を写した。数字と warning の文は、生の出力(セッションの scratchpad の `effort-probe/` にある `<値>.out` と `<値>.err`)で確かめ直した。生の出力はコミットしていない。session id などの値は写していない。

## 環境

| 項目 | 値 |
|---|---|
| claude | `claude --version` は 2.1.308 (Claude Code)。`--model haiku`(出力の `modelUsage` のキーは `claude-haiku-5-5`) |
| 場所 | scratchpad の `effort-probe/`。git の外で、`git rev-parse` は「not a git repository」を返す |
| 環境変数 | `RALPH_WIP_CHECKPOINT=off` |

## 打ったコマンド

1 回目(質問 B、`reply ok`):

```
RALPH_WIP_CHECKPOINT=off claude -p --model haiku --effort bogus --output-format json 'reply ok' </dev/null > bogus.out 2> bogus.err
RALPH_WIP_CHECKPOINT=off claude -p --model haiku --effort xhigh --output-format json 'reply ok' </dev/null > xhigh.out 2> xhigh.err
```

2 回目(質問 A、`In one sentence, why is the sky blue?`):

```
for e in low max; do RALPH_WIP_CHECKPOINT=off claude -p --model haiku --effort $e --output-format json 'In one sentence, why is the sky blue?' </dev/null > $e.out 2> $e.err; done
```

- どの値も 1 回ずつ打った。macOS には `timeout` がないので付けていない。
- thinking のトークンは、`.out` の JSON の `usage.output_tokens_details.thinking_tokens` を `python3 -I` で読んだ。

## 結果

### 正しくない値は warning を出し、既定の effort で続く

`--effort bogus` は rc 0 で、`.out` の `is_error` は false だった。stderr の全文は次の 1 行。

```
Warning: Unknown --effort value 'bogus' — ignoring it and using the default effort. Valid values: low, medium, high, xhigh, max.
```

ほかの 3 つの `.err` は空で、`is_error` はどれも false だった。

### thinking のトークン

値は `.out` の `usage.output_tokens_details.thinking_tokens` と `usage.output_tokens`。

| `--effort` | 質問 | thinking のトークン | output のトークン | 答え |
|---|---|---|---|---|
| `low` | A | 0 | 77 | 1 文の説明 |
| `max` | A | 1066 | 1154 | 1 文の説明 |
| `bogus`(既定の effort) | B | 58 | 65 | 「了解しました。」 |
| `xhigh` | B | 236 | 240 | 「ok」 |

## 判断

- `-p` でも `--effort` は効く。同じ質問 A で、thinking のトークンは `low` で 0、`max` で 1066 だった。質問 B では、既定の 58 に対して `xhigh` が 236 だった。
- 正しくない値は、claude が warning を出して無視し、既定の effort で続ける。
- そのため計画の D4 のとおり、claude の reviewer の呼び出し(`/plan` の計画助言、`/cross-review`、feature org の leader の計画助言)に `--effort <[reviewer].claude_effort>` を渡す。warning で黙って既定に戻らないよう、Load は 5 つの値(low・medium・high・xhigh・max)の外を先に断る。

## 限り

- どの値も 1 回ずつ試しただけで、統計ではない。同じ値と同じ質問でも、thinking のトークンは回ごとに変わりうる。
- 質問 A と質問 B は別なので、A の数(0・1066)と B の数(58・236)は比べられない。
- 試したモデルは haiku だけで、reviewer の既定の fable では試していない。`medium` と `high` も試していない。
