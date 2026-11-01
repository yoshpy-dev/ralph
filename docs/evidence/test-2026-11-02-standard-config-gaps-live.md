# PR #291 の Known gaps の実物の確かめ

- Date: 2026-11-02(JST 02:07〜02:11)
- Plan: `docs/plans/active/2026-11-02-standard-config-gaps.md`(Scope の「PR #291 の Known gaps の実物の確かめ」、AC5、D4)
- Tester: tester サブエージェント(`/test`、pipeline の cycle 1)
- 生の出力(`.out`・`.err`・`codex.log`)はセッションの scratchpad の `test-beta/` に置き、この文書を書いたあとで消した。コミットしていない。session id、認証の値、token の値は写していない。

## 環境

| 項目 | 値 |
|---|---|
| PATH の ralph | `/opt/homebrew/bin/ralph`、`ralph version` は `ralph 6.0.0 (44c14b97 2026-10-27T01:34:06Z)` |
| build した ralph | 作業ツリーの HEAD 60838e34 から `go build -o <scratchpad>/test-beta/bin/ralph ./cmd/ralph`(rc 0、`ralph config get --help` がある) |
| claude | 2.1.308 (Claude Code) |
| codex | codex-cli 0.163.0。ユーザーの `~/.codex/config.toml` の既定は `model = "gpt-6.1-sol"`、`model_reasoning_effort = "max"` |
| シェル | Bash tool の zsh 5.9。(a) の helper は `sh` でも打った |
| main checkout | d217c575 |

## (a) v6.0.0 での扱い

作業ツリーの root で、PATH の ralph 6.0.0 のまま打った。stdout と stderr は別のファイルに受けた。

```
. ./scripts/ralph-config.sh
ralph_setting reviewer.codex_model
ralph_setting reviewer.claude_effort
ralph_setting_or_fail standard.base_branch
ralph config get standard.cross_review
```

| コマンド | stdout | rc | stderr |
|---|---|---|---|
| `ralph_setting reviewer.codex_model` | `gpt-6-astra` | 0 | 空(0 バイト) |
| `ralph_setting reviewer.claude_effort` | `xhigh` | 0 | 空 |
| `ralph_setting_or_fail standard.base_branch` | 空 | 0 | 空 |
| `ralph config get standard.cross_review` | 空 | 1 | `unknown command "config" for "ralph"` の 1 行 |

- 3 つの helper は既定値(`standard.base_branch` は空が既定)を黙って返した。`sh -c '. ./scripts/ralph-config.sh; …'` で打っても値、rc、stderr は同じだった。
- `ralph config get standard.cross_review` は exit 1 で、stderr に `unknown command "config"` を含む。`.claude/skills/cross-review/SKILL.md` の step 0(32 行)と `.claude/skills/plan/SKILL.md` の step 11.a(76 行)の規則では、この形は「ralph without config get; treated as off」で、review と plan advisory を飛ばして skip を記録する。skill そのものは回していない。

## (b) fable の `--effort`

git の外の scratchpad の `test-beta/b/` から打った(`git rev-parse --show-toplevel` は「not a git repository」で rc 128)。値ごとに 1 回ずつ、次の形で打った。

```
RALPH_WIP_CHECKPOINT=off claude -p --model fable --effort <値> --output-format json 'In one sentence, why is the sky blue?' </dev/null > <値>.out 2> <値>.err
```

`.out` の JSON を `python3 -I` で読んだ。thinking は `usage.output_tokens_details.thinking_tokens`、output は `usage.output_tokens`。`modelUsage` の `thinkingTokens` も同じ値だった。

| `--effort` | rc | thinking | output | `modelUsage` のキー | `is_error` | `.err` | `duration_ms` |
|---|---|---|---|---|---|---|---|
| `low` | 0 | 0 | 69 | `claude-fable-5-1` | false | 空 | 3,054 |
| `medium` | 0 | 0 | 68 | `claude-fable-5-1` | false | 空 | 2,986 |
| `high` | 0 | 0 | 69 | `claude-fable-5-1` | false | 空 | 3,060 |
| `xhigh` | 0 | 0 | 97 | `claude-fable-5-1` | false | 空 | 7,072 |

この質問では、fable はどの値でも thinking を 0 にした。値の違いは output の数(`xhigh` だけ 97 で、答えに「散乱の強さは波長の 4 乗に反比例する」が加わった)と時間にしか出ず、これだけでは `--effort` が fable に届いたかを判断できない。そこで、計画にない追加として、考える手間の要る質問で `low` と `xhigh` を 1 回ずつ打った。

```
RALPH_WIP_CHECKPOINT=off claude -p --model fable --effort <値> --output-format json 'How many positive integers below 1000 are divisible by 7 but not by 11? Reply with the number only.' </dev/null > q2-<値>.out 2> q2-<値>.err
```

| `--effort` | rc | thinking | output | `modelUsage` のキー | `is_error` | `.err` | 答え |
|---|---|---|---|---|---|---|---|
| `low` | 0 | 0 | 31 | `claude-fable-5-1` | false | 空 | 途中の式を書いて `130` |
| `xhigh` | 0 | 78 | 81 | `claude-fable-5-1` | false | 空 | `130` だけ |

- `low` は考えずに式を答えに書き、`xhigh` は thinking に 78 トークン使って数だけを答えた。同じ質問で thinking が 0 と 78 に分かれたので、`-p` の `--effort` は fable にも届いている。
- 6 回とも `.err` は空で、warning は出なかった(正しくない値のときの warning は 2026-10-30 の確かめにある)。

## (c) 設定の値で claude と codex を起こす

scratchpad の `test-beta/c/repo/` に `git init` で一時の repository を作った(ralph の checkout の外。`git rev-parse --show-toplevel` はそのディレクトリ)。その root に次の `ralph.toml` を書いた。main checkout の `ralph.local.toml` には触っていない。

```toml
[reviewer]
claude_model = "haiku"
claude_effort = "low"
codex_model = "gpt-6-astra"
codex_effort = "low"
```

PATH の先頭に build した ralph を置き(`command -v ralph` がそれを指すことを確かめた)、作業ツリーの `scripts/ralph-config.sh` を source して、一時の repository の root で読んだ。

| key | 値 | rc | stderr |
|---|---|---|---|
| `reviewer.claude_model` | `haiku` | 0 | 空 |
| `reviewer.claude_effort` | `low` | 0 | 空 |
| `reviewer.codex_model` | `gpt-6-astra` | 0 | 空 |
| `reviewer.codex_effort` | `low` | 0 | 空 |

対照として、同じ ralph で `ralph.toml` のない scratchpad の `test-beta/` から読むと、`reviewer.claude_model` は `fable`、`reviewer.claude_effort` は `xhigh`(既定値)だった。上の値は一時の `ralph.toml` から来ている。

### claude

`/cross-review` の claude の形で、1 つの Bash 呼び出しの中で読んでから起こした。

```
. <作業ツリー>/scripts/ralph-config.sh; model=$(ralph_setting reviewer.claude_model); effort=$(ralph_setting reviewer.claude_effort); RALPH_WIP_CHECKPOINT=off claude -p --model "$model" --effort "$effort" --permission-mode auto --output-format json 'Reply with the single word ok.' </dev/null > claude.out 2> claude.err
```

- 変数は `model=haiku effort=low` だった。
- rc 0、`.err` は空。`.out` の `modelUsage` のキーは `claude-haiku-5-5` だけで、`is_error` は false、答えは `ok`(output 4、thinking 0)。既定の fable ではなく、`ralph.toml` の haiku が使われた。

### codex

`/cross-review` と同じく `-c` を `exec` の前に置き、`</dev/null` を付けた。skill の watchdog の形を、上限を 1200 秒から 300 秒に縮めて使った。

```
. <作業ツリー>/scripts/ralph-config.sh; model=$(ralph_setting reviewer.codex_model); effort=$(ralph_setting reviewer.codex_effort)
rm -f codex-last.md; command codex -m "$model" -c "model_reasoning_effort=$effort" -c sandbox_mode=read-only exec --ignore-rules -o codex-last.md 'Reply with the single word ok.' </dev/null > codex.log 2>&1 & cpid=$!; sleep 300 & spid=$!; ( while kill -0 "$spid" 2>/dev/null; do sleep 1; done; kill "$cpid" 2>/dev/null ) & wpid=$!; wait "$cpid"; rc=$?; kill "$spid" 2>/dev/null; kill "$wpid" 2>/dev/null; echo "codex rc=$rc"
```

- 変数は `model=gpt-6-astra effort=low` だった。`codex rc=0` で、11 秒で終わった(watchdog は働いていない)。`codex-last.md` は `ok`。
- `codex.log` の頭(session id の行は写していない):

```
OpenAI Codex v0.163.0
model: gpt-6-astra
provider: openai
approval: never
sandbox: read-only
reasoning effort: low
reasoning summaries: none
```

- model と reasoning effort は、渡した `gpt-6-astra` と `low` だった。ユーザーの `~/.codex/config.toml` の既定(`gpt-6.1-sol`、`max`)とも ralph の既定(`gpt-6-astra`、`xhigh`)とも違う `low` が出たので、effort は `ralph.toml` の値が `-c` で届いている。model は ralph の既定と同じ値なので、ralph の既定との区別はつかない(codex の既定 `gpt-6.1-sol` とは区別がつく)。
- ログには、ユーザーの設定の MCP サーバー(Sentry など)の OAuth の更新に失敗した `ERROR` 行が 3 行と、ユーザーの hook の `SessionStart Failed` と `Stop Failed` があった。一時の repository には `.codex` がなく、どちらもユーザーの設定によるもので、答えと rc には関係しない。

### main checkout の `.codex/config.toml`

codex の前後で、main checkout(`/Users/hiroki.yoshioka/MyDev/github.com/yoshpy-dev/ralph`)で確かめた。

| | `git status --short -- .codex/config.toml` | `shasum -a 256` の頭 16 桁 |
|---|---|---|
| codex の前 | 空 | `96919613aeacafd5` |
| codex の後 | 空 | `96919613aeacafd5` |

書き換わっていない。作業ツリーの `git status --porcelain` も前後で空だった。

## 後片付け

この文書を書いたあとで、scratchpad の `test-beta/` の下の一時の repository(`c/`)、build した binary(`bin/`)、(a)(b) の出力(`a/`、`b/`)、読み取りの script を消した。結果は報告 `docs/reports/test-2026-11-02-standard-config-gaps.md` の「AC5」にある。

## 限り

- (b) はどの値も 1 回ずつで、統計ではない。同じ値と同じ質問でも、thinking のトークンは回ごとに変わりうる。1 つ目の質問では 4 つとも thinking が 0 で、効き目の判断は追加の質問の `low` と `xhigh` の 1 組だけに拠る。`medium` と `high` は追加の質問で打っていない。
- (c) の codex の model は ralph の既定と同じ値で、model の値が `ralph.toml` から届くことは effort の結果と helper の読み取りから推している。codex の model を既定と違う値にした実物の起動はしていない。
- skill(`/cross-review`、`/plan`)そのものは回していない。(a) の skip の扱いは skill の文の規則に照らした。
