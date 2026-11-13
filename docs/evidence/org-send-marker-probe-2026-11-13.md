# 入力欄の印で始まる行を含む文の画面の確かめ(2026-11-13、#322)

#322 の直し方を決めるために、分離した herdr の上の claude と codex の座席で、
入力欄の印(claude は `❯`、codex は `›`)で始まる行を含む文を入力欄に打ち、
画面を読んだ。Enter は 1 回も押していない。読んだあと Ctrl+C を 1 回押して
入力欄を空にしたので、claude と codex の手番は 1 回も使っていない。ralph は
通していない。`herdr pane send-text`、`pane send-keys`、`pane read` を直接
呼んだ。

分かったことは次の 4 点。

- どちらの CLI も、入力欄の 1 行目は印が 0 桁目にあり、文の 2 行目からは
  2 字下げで描く。文の中の印で始まる行も 2 字下げになる(`  › explain this`、
  `  ❯ explain this`)。
- codex(今の打ち方は素の文)では、文の中の `›` の行が入力欄に出る。下から
  最初の印の行を入力欄の頭とみなす今の確かめ(`findResumeInputBox`)は、その
  行を頭と取り違える(#322 の再現)。
- claude(#319 から bracketed paste で打つ)では、改行が 3 つ以上の文は、
  67 バイト・4 行の短い文でも `[Pasted text #N +M lines]` の 1 行に畳まれた。
  印の行は入力欄に出ない。改行が 2 つ以下の文は畳まれず、文の中の `❯` の行が
  2 字下げで出た。typed protocol の文は `TYPE:` の行と空行のあとに本文が続く
  ので、本文が 1 行の文(改行 2 つ)で #322 が起きる。
- claude の入力欄の頭の印は `❯` のあとに NBSP(U+00A0)が続いた
  (`❯ TYPE: QUESTION`)。文の中の行は 2 字下げの `❯` と普通の空白だった。

## 環境と分離

- macOS 26.5.2、Claude Code 2.1.316、codex-cli 0.166.0、herdr 0.7.5。
- 座席のモデルは claude が `haiku`、codex が `gpt-6-luna`(effort low)。pane は
  幅 53 桁、高さ 23 行。
- herdr は `docs/recipes/isolated-herdr-server.md` のとおりに分けた。`env -i` の
  包みのスクリプトで、`XDG_CONFIG_HOME`、`XDG_STATE_HOME`、`HERDR_SOCKET_PATH` を
  `/tmp/ml308/` の下に向け、`CLAUDE_ENV_FILE` もそこに置いた。`HOME` は本物の
  まま、`GIT_CONFIG_GLOBAL` は `$HOME/.config/git/config` にした。codex の
  binary は包みの `PATH` に入っていなかったので、`/tmp/ml308/bin/codex` から
  codex-cli 0.166.0 の binary へ symlink を張った。
- 座席の cwd は `/private/tmp/ml308/work` の scratch の git repo。claude
  (`~/.claude.json`)と codex(`~/.codex/config.toml`)の両方で信頼済みの場所で、
  フォルダ信頼の画面は出なかった。codex の更新の案内は `2. Skip`、フックの確認は
  `3. Continue without trusting (hooks won't run)` で答えた。
- pane の起動は `docs/evidence/org-send-paste-probe-2026-11-13.md` と同じ
  (`herdr workspace create ... --env AGMSG_SELF_NAME=off --env RALPH_WIP_CHECKPOINT=off --env RALPH_PRIMARY_CLI=<driver>`
  と `herdr agent start <driver>`)。
- ユーザーの `herdr server`(pid 34407)には、コマンドを 1 つも送っていない。
  分離した server は最後に `herdr server stop` で止め、`/tmp/ml308` を消した。

## 打った文と打ち方

claude には、ralph org send が #319 から claude の座席に打つのと同じく、文を
`ESC [200~` と `ESC [201~` で挟んで 1 回の `herdr pane send-text` で打った。
codex には素の文を 1 回で打った。1.5 秒待って
`herdr pane read <pane> --source recent --lines 60 --format text`(ralph の画面の
確かめと同じ読み方)で読み、`herdr pane send-keys <pane> ctrl+c` を 1 回押して、
1.5 秒後にもう一度読んだ。

| 名前 | 中身 | バイト | 改行 |
|---|---|---|---|
| short | `TYPE: QUESTION`、空行、印で始まる行、`Reply with the single word: alpha.` | 67 | 3 |
| last | short と同じで、印で始まる行が最後 | 67 | 3 |
| medium | `TYPE: TASK`、`TASK_ID:`、空行、10 行のうち 5 行目と 9 行目が印で始まる、最後の 1 行 | 619 | 13 |
| numbered | `TYPE: TASK`、`TASK_ID:`、空行、`1. read the plan`、印 + `2. explain this step`、最後の 1 行 | 107 | 5 |
| two-nl(claude だけ) | `TYPE: QUESTION`、空行、`❯ explain this` | 32 | 2 |
| one-nl(claude だけ) | `TYPE: QUESTION`、`❯ explain this` | 31 | 1 |
| no-nl(claude だけ) | `❯ explain this` | 16 | 0 |

印は、claude に打つ文では `❯`、codex に打つ文では `›` にした。

## codex の結果

4 つとも、文の中の `›` の行が 2 字下げで入力欄に出た。入力欄の頭の `›` は
0 桁目だった。

short:

```text
› TYPE: QUESTION

  › explain this
  Reply with the single word: alpha.

  GPT-6-Luna · low · Context 0% used · main · No chan…
                                            ⚠ 11 · f2
```

medium(入力欄は文の 6 行目から下を出し、1 行目の右端に `↑` が付いた):

```text
› line 06: the quick brown fox jumps over the lazy   ↑
  dog
  line 07: the quick brown fox jumps over the lazy
  dog
  line 08: the quick brown fox jumps over the lazy
  dog
  › quoted line 09: the quick brown fox jumps over
  the lazy dog
  line 10: the quick brown fox jumps over the lazy
  dog
  Reply with the single word: charlie.

  GPT-6-Luna · low · Context 0% used · main · No chan…
                                            ⚠ 11 · f2
```

numbered では `  › 2. explain this step` が出た。番号付きの行なので、今の
`findResumeInputBox` はこれを選択の画面の cursor の行とみなす。last では
`  › explain this` が入力欄の最後の行に出た。

Ctrl+C のあとは、4 つとも入力欄が空の画面(`› Ask Codex to do anything`)に
戻った。herdr の状態はどれも `done` だった。

## claude の結果

short、last、medium、numbered(改行 3 つ以上)は、入力欄が placeholder の
1 行になった。

```text
─────────────────────────────────────────────────────
❯ [Pasted text #1 +3 lines]
─────────────────────────────────────────────────────
  🤖 Haiku 5.5 | ⚡ low | 🌿 main
  paste again to expand
```

M は文の改行の数だった(short と last は 3、medium は 13、numbered は 5)。
`#N` は貼り付けるたびに 1 ずつ増えた。

改行が 2 つ以下の文は畳まれなかった。two-nl:

```text
                          ctrl+g to edit in VS Code
─────────────────────────────────────────────────────
❯ TYPE: QUESTION

  ❯ explain this
─────────────────────────────────────────────────────
  🤖 Haiku 5.5 | ⚡ low | 🌿 main
  ⏵⏵ bypass permissions on (shift+tab to cycle)
```

one-nl は `❯ TYPE: QUESTION` と `  ❯ explain this` の 2 行、no-nl は
`❯ ❯ explain this` の 1 行だった。two-nl と one-nl では、下から最初の `❯` の行が
文の中の行で、その真上は罫線ではない。今の `findResumeInputBox` は入力欄を
見つけられない(#322 の再現)。no-nl では、入力欄の頭の `❯` の行が最後の印の
行になるので、今の確かめで通る。

入力欄の頭の行は `❯` のあとに NBSP(U+00A0)が続いた。文の中の行は、2 字下げの
`❯` と普通の空白(U+0020)だった。Ctrl+C のあとは、どれも空の入力欄(`❯`)に
戻った。herdr の状態はどれも `idle` だった。

## この確かめから決めたこと

- 送る側の Enter の前の確かめ(`sendTypedInputBox`)だけで、入力欄の頭を
  選び直す。最後の印の行が字下げされていて、その中身が打った文の一部なら、
  その上の 0 桁目の印の行を入力欄の頭とする。決め方の詳細は計画
  `docs/plans/active/2026-11-13-send-input-box-marker.md` の Design decisions。
- resume の画面の型(`findResumeInputBox`、`ClassifyResumeScreen`)と、Enter の
  あとの確かめ(`SendScreenSubmitted`)の入力欄の探し方は変えない。文が入力欄に
  残っていれば、どちらも今までどおり入力欄を見つけられず、「送れていない」側に
  倒れる。
- 上の画面のうち codex の 4 つと claude の two-nl、one-nl、no-nl、short を、
  testdata の fixture にする。

## 確かめで残ったもの

- `~/.claude.json` と `~/.codex/config.toml` の信頼の項(前からあった
  `/private/tmp/ml308/work`)。
- `~/.claude/projects/-private-tmp-ml308-work/` のセッションの記録 1 つ
  (`b32b4039-….jsonl`、user の文は 0)。codex は、15:15 より後に
  `~/.codex/sessions/2026/11/13/` に記録を作らず、`~/.codex/history.jsonl` にも
  この場所の行はなかった。

## 確かめていないこと

- 文を Enter で送ったあとの画面(手番を使うので打っていない)。送った文の中の
  印の行が履歴でどう描かれるかは、Enter のあとの確かめに関わるが、今の型は
  入力欄が最後の印の行になるので変わらないとみる。未確認。
- 画面の道(herdr が手番の終わった座席を `working` と言い続ける場合)を ralph で
  通しで走らせること。#308 の Known gaps (a) と同じく、その状態を作れない。
- Claude Code 2.1.316 と codex-cli 0.166.0 のほかの版。
- codex に bracketed paste で打ったとき(#326 で扱う)。
