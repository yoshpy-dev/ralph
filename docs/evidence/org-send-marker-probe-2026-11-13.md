# 入力欄の印で始まる行を含む文の画面の確かめ(2026-11-13、#322)

#322 の直し方を決めるために、分離した herdr の上の claude と codex の座席で、
入力欄の印(claude は `❯`、codex は `›`)で始まる行を含む文を入力欄に打ち、
画面を読んだ。この確かめでは Enter を押さず、読んだあと Ctrl+C を 1 回押して
入力欄を空にしたので、手番は使っていない。そのあと計画助言の指摘を受けて、
Enter で送ったあとの画面も取った(「Enter のあとの画面」の節、claude と codex で
手番 1 回ずつ)。ralph は通していない。`herdr pane send-text`、`pane send-keys`、
`pane read` を直接呼んだ。

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

## Enter のあとの画面(計画助言の指摘を受けて足した確かめ)

計画助言(codex)の指摘を受けて、Enter で送ったあとの画面も取った。送れた画面で
Enter のあとの確かめ(`SendScreenSubmitted`)が「送れた」とみるかを、テストで
固定するため。同じ手順で分離した herdr を立て直し、claude と codex の座席に
1 回ずつ、印で始まる行を含む文を送った。打ち方は ralph と同じにした(claude は
bracketed paste、codex は素の文、打って 750 ms 後に Enter を 1 回)。Enter から
0.3 秒後と 1.5 秒後、herdr が `idle`/`done` に戻ったあとに読んだ。

| 座席 | 文 | バイト | 改行 |
|---|---|---|---|
| claude | `TYPE: QUESTION`、空行、`❯ explain this. This is a probe: … reply with the single word: echo.` | 126 | 2 |
| codex | `TYPE: QUESTION`、空行、`› explain this`、`This is a probe: … Reply with the single word: foxtrot.` | 124 | 3 |

claude では、Enter の前の入力欄に、折り返した `❯` の行が 2 字下げで出た。

```text
─────────────────────────────────────────────────────
❯ TYPE: QUESTION
  ❯ explain this. This is a probe: do not run any
  tool or read any file, and reply with the single
  word: echo.
─────────────────────────────────────────────────────
```

Enter から 0.3 秒後には、送った文が履歴に移った。履歴でも `❯` の行は 2 字下げで、
その下に `✳ Puzzling…` が出て、入力欄は空になった。

```text
❯ TYPE: QUESTION
  ❯ explain this. This is a probe: do not run any
  tool or read any file, and reply with the single
  word: echo.
✳ Puzzling…
─────────────────────────────────────────────────────
❯
─────────────────────────────────────────────────────
```

1.5 秒後には、送った文の下に
`⎿  You've hit your 5-hour limit · resets 3:50pm (Asia/Tokyo)` が出て、手番が
`⏺ Usage limit reached · continuing automatically at 3:50pm · esc to cancel`
で止まっていた。座席が使う claude のアカウントが、5 時間の上限(15:50 に解ける)に
当たっていた。文はモデルに渡る前に止まったので、返事の語は確かめていない。
15:50 に Claude Code が自分で続きを始める前に、分離した server を止めた。

codex では、Enter から 0.3 秒後に送った文が履歴に移り、入力欄は空の
`› Ask Codex to do anything` に戻った。1.5 秒後は
`• Working (0s • esc to interrupt)`、手番の終わりには `• foxtrot` が出た。履歴でも
文の中の `›` の行は 2 字下げだった。

```text
› TYPE: QUESTION
  › explain this
  This is a probe: do not run any tool or read any
  file. Reply with the single word: foxtrot.
› Ask Codex to do anything
  GPT-6-Luna · low · Context 0% used · main · No chan…
```

どちらでも、下から最初の印の行は、空になった入力欄の頭の行(0 桁目)だった。
文の中の印の行は、その上の履歴にある。codex の返事の前後には、ユーザーの codex の
フックの `• Hook failed` の行が出た(フックの確認で信頼しないを選んだため)。

この確かめで使った手番は、claude と codex で 1 回ずつ。claude の 1 回は上限で
止まった。

## この確かめから決めたこと

- 送る側の Enter の前の確かめ(`sendTypedInputBox`)だけで、入力欄の頭を
  選び直す。最後の印の行が字下げされていて、その中身が打った文の一部なら、
  その上の 0 桁目の印の行を入力欄の頭とする。決め方の詳細は計画
  `docs/plans/active/2026-11-13-send-input-box-marker.md` の Design decisions。
- resume の画面の型(`findResumeInputBox`、`ClassifyResumeScreen`)と、Enter の
  あとの確かめ(`SendScreenSubmitted`)の入力欄の探し方は変えない。文が入力欄に
  残っていれば、どちらも今までどおり入力欄を見つけられず、「送れていない」側に
  倒れる。
- 上の画面のうち codex の 4 つと claude の two-nl、one-nl、no-nl、short、
  Enter のあとの節の claude の Enter の前の画面と、両方の Enter から 0.3 秒後の
  画面を、testdata の fixture にする。

## 確かめで残ったもの

- `~/.claude.json` と `~/.codex/config.toml` の信頼の項(前からあった
  `/private/tmp/ml308/work`)。
- `~/.claude/projects/-private-tmp-ml308-work/` のセッションの記録 2 つ。
  Enter を押さなかった回の `b32b4039-….jsonl` には user の文がない。Enter の
  あとの節の回の `f6567ebf-….jsonl` には、送った文がある。
- `~/.codex/sessions/2026/11/13/` の、Enter のあとの節の回の記録(15:35 に
  始まったもの)。Enter を押さなかった回は記録を作らなかった。
- `~/.claude/history.jsonl` と `~/.codex/history.jsonl` の、送った文の行。

## 確かめていないこと

- claude の座席が上限に当たらずに返事をしたあとの画面。上限で止まったため、
  返事の語(`echo`)は確かめていない。
- 画面の道(herdr が手番の終わった座席を `working` と言い続ける場合)を ralph で
  通しで走らせること。#308 の Known gaps (a) と同じく、その状態を作れない。
- Claude Code 2.1.316 と codex-cli 0.166.0 のほかの版。
- codex に bracketed paste で打ったとき(#326 で扱う)。
