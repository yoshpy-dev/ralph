# org send の長い文の打ち方の確かめ(2026-11-13、#319)

#319 の直し方を決めるために、分離した herdr の上で、長い typed message を
claude と codex の座席に 3 通りの打ち方で打ち、モデルに届いた文をセッションの
記録で元の文と比べた。ralph は通していない。`herdr pane send-text` と
`pane send-keys` を直接呼んだ。

結論は次の 4 点。

- 今の打ち方(1 回の `send-text`)では、claude に 19 行・1117 バイトの文を打つと
  先頭の 1022 バイトが消えた(#319 の再現)。
- 文を bracketed paste の印(`ESC [200~` と `ESC [201~`)で挟んで 1 回で打つと、
  1117 バイトの文も 5340 バイトの文も、claude に 1 字も欠けずに届いた。
  124 バイトの短い文は畳まれず、素の文として届いた。
- codex では、今の打ち方で 5340 バイトの文の頭は消えなかった。ただし 0.75 秒
  後の Enter では送られなかった。bracketed paste では Enter で送られた。
- 打ち込みが途中で切れる場合も確かめた。herdr の client を途中で kill した
  12 回は、どれも文が 1 バイトも届かなかった。claude は終わりの印が来ない
  貼り付けを約 2 秒で閉じ、そのあとの入力を取り込まなかった
  (「打ち込みが途中で切れたときの確かめ」)。

## 環境と分離

- macOS 26.5.2、Claude Code 2.1.316、codex-cli 0.166.0、herdr 0.7.5。
- 座席のモデルは claude が `haiku`(画面の表示は `Haiku 5.5 with low effort`)、
  codex が `gpt-6-luna`(effort low)。
- herdr は `docs/recipes/isolated-herdr-server.md` のとおりに分けた。
  `env -i` の包みのスクリプトで、`XDG_CONFIG_HOME`、`XDG_STATE_HOME`、
  `HERDR_SOCKET_PATH` を `/tmp/ml319/` の下に向け、`CLAUDE_ENV_FILE` も
  そこに置いた。`HOME` は本物のまま、`GIT_CONFIG_GLOBAL` は
  `$HOME/.config/git/config` にした。
- ユーザーの `herdr server`(pid 34407)は、確かめの前とあとで同じ pid で
  動いていた。分離した server(pid 85345)は最後に `herdr server stop` で止めた。
- pane は次の 2 つ。

  ```
  herdr workspace create --cwd /tmp/ml319/work --label probe319 --no-focus \
    --env AGMSG_SELF_NAME=off --env RALPH_WIP_CHECKPOINT=off \
    --env RALPH_PRIMARY_CLI=claude --env CLAUDE_CODE_RETRY_WATCHDOG=1 ...
  herdr agent start claude --kind claude --pane w1:p1 --timeout 60000 -- \
    --model haiku --permission-mode bypassPermissions

  herdr workspace create --cwd /tmp/ml319/work --label probe319cx --no-focus \
    --env AGMSG_SELF_NAME=off --env RALPH_WIP_CHECKPOINT=off \
    --env RALPH_PRIMARY_CLI=codex
  herdr agent start codex --kind codex --pane w2:p1 --timeout 60000 -- \
    --sandbox workspace-write --ask-for-approval never --model gpt-6-luna \
    -c model_reasoning_effort=low
  ```

- codex のフォルダ信頼は `1. Trust and continue` で、フックの確認は
  `3. Continue without trusting (hooks won't run)` で答えた。

## 打った文と打ち方

文は 3 種類。どれも最後の行を `Reply with the single word: <語>.` にし、語を
回ごとに変えて、記録の中で見分けた。

| 文 | 形 | 大きさ |
|---|---|---|
| m1 | `TYPE: TASK`、`TASK_ID:`、`PLAN:` の行と英文の埋め草。#319 の 10 節と同じ形 | 19 行、1117 バイト、改行 18 |
| m2 | 同じ 3 つの見出しの行と、日本語の本文 2000 字(typed protocol の上限) | 44 行、5340 バイト、2146 字、改行 43 |
| s | `TYPE: QUESTION` と 1 行の本文 | 3 行、124 バイト、改行 2 |

打ち方は 3 通り。

- plain: `herdr pane send-text <pane> <文>` を 1 回。今の `ralph org send` と同じ。
- paste: 同じ 1 回の呼び出しで、文の前に `ESC [200~`、後ろに `ESC [201~` を付けた。
- chunk: 文を行の切れ目で 512 バイト以下の塊に分け、塊ごとに `send-text` を
  呼び、あいだを 150 ミリ秒あけた。m1 は 3 塊(444、497、176 バイト)、m2 は
  13 塊。

どの打ち方でも、最後の `send-text` から 0.75 秒待って画面を読み(`ralph org send`
の既定の 750 ミリ秒と同じ)、Enter を押し、0.3 秒後にもう一度読んだ。そのあと
herdr の状態が idle か done になるまで(長くて 60 秒)待った。

届いた文は、claude では `~/.claude/projects/-private-tmp-ml319-work/` の
セッションの記録の user の文、codex では `~/.codex/sessions/2026/11/13/` の
rollout の記録の user の文から取り、元の文とバイト単位で比べた。

## claude の結果

| 回 | 文(語) | 打ち方 | Enter の前の入力欄 | 届いた文 |
|---|---|---|---|---|
| A1 | m1(delta) | plain | `❯ e 09: the quick brown fox jumps over the lazy dog` から始まる末尾の 3 行 | 末尾の 95 バイトだけ。元の文の終わりと一致し、先頭の 1022 バイトはない |
| B1 | m1(tango) | paste | `❯ [Pasted text #2 +18 lines]` | `<pasted_content>` 1 つ。中身は元の文と同じ 1117 バイト |
| B2 | m2(zebra) | paste | `❯ [Pasted text #3 +43 lines]` | `<pasted_content>` 1 つ。中身は元の文と同じ 5340 バイト |
| C1 | m1(hotel) | chunk | 文の終わりの行(入力欄は畳まれず、終わりまで流れていた) | 畳まれない素の文。元の文と同じ 1117 バイト |
| C2 | m2(otter) | chunk | 文の終わりの行(同上) | 畳まれない素の文。元の文と同じ 5340 バイト |
| S1 | s(kappa) | paste | `❯ TYPE: QUESTION` から始まる 3 行(畳まれない) | 素の文。元の文と同じ 124 バイト |

読み取れたこと。

- A1 では、Enter の前から入力欄に末尾しかなかった。頭は送る前に消えている。
- paste では、文の大きさによらず貼り付けが 1 つにまとまった。印の内側を
  herdr が約 1 KB の塊に分けて渡しても、claude は終わりの印までを 1 つの
  貼り付けとして集めた。5340 バイトの B2 でもそうだった。
- 置き換えの `+N lines` の N は、文の改行の数(18 と 43)と同じだった。
  `ralph org send` の画面の道の 3 つ目の確かめ(`sendClaudePastesHold`)は、
  置き換えの N の合計が打った文の改行の数を超えないことを見るので、この形は
  そのまま通る。
- 記録の user の文で、`<pasted_content>` の外にあったのは改行 3 つだけだった。
  `#2`、`#3` の番号は同じセッションの中で増えていく(A1 の消えた塊が `#1`)。
- 124 バイトの S1 は畳まれなかった。短い文は、bracketed paste で打っても今と
  同じ素の文として届く。
- chunk でも届いたが、それは塊が 800 バイトより小さいまま claude に読まれた
  からとみている。claude の読みが遅れて塊がまとめて読まれると、また畳まれる
  大きさになりうる。負荷をかけての確かめはしていない(推測)。

claude の手番は A1、B1、B2、C1、C2、S1 の 6 回使った。

## codex の結果

| 回 | 文(語) | 打ち方 | Enter の前の入力欄 | Enter のあと | 届いた文 |
|---|---|---|---|---|---|
| X1 | m2(otter) | plain | 文の途中の行。0.75 秒後と 1.05 秒後でも、画面はまだ文を描き足していた | 0.75 秒後の Enter では送られず、60 秒後も文は入力欄に残った(herdr の状態は done)。もう一度 Enter を押すと送られた | 元の文と同じ 5340 バイト |
| X2 | m2 の語を変えたもの(walrus、末尾の改行を含めて 5341 バイト) | paste | `› [Pasted Content 2147 chars]` | 0.75 秒後の Enter で送られ、0.3 秒後に `• Working (0s • esc to interrupt)` | 元の文と同じ 5341 バイト |

読み取れたこと。

- codex では、今の打ち方でも文の頭は消えなかった。#319 の 10 節で 1829 バイト
  まで全部届いたのと合う。
- X1 では、0.75 秒後の Enter が送信にならなかった。届いた文に余分な改行は
  なかった。Enter が文の打ち込みの続きとして扱われたのか、捨てられたのかは
  わからない。
- ralph の herdr の道では、Enter のあと `confirmSubmitted` が working か
  blocked を待ち、来なければ `sent` に `submit_unconfirmed=true` を付けて
  知らせる。X1 の形なら、文は送られないまま、その印が付くとみている。ralph を
  通しては試していない(推測)。
- bracketed paste なら Enter で送られた。ただし入力欄は `[Pasted Content N chars]`
  の 1 行に置き換わるので、画面の道の 1 つ目と 2 つ目の確かめ(打った文の頭と
  終わりが入力欄にあること)は通らなくなる。codex を変えるには、画面の規則も
  一緒に変える必要がある。

## この確かめから決めたこと

#319 では、claude の座席に打つ文だけを bracketed paste の印で挟む。

- 1 回の `send-text` のままなので、塊のあいだの時間に頼らない。
- 届いた文は 1117 バイトでも 5340 バイトでも元と同じだった。
- 置き換えの `+N lines` が文の改行の数と同じなので、画面の道の今の確かめが
  そのまま通る。
- 短い文は今と同じ素の文として届く。長い文は今でも 800 バイトを超えた分が
  `<pasted_content>` で届いているので、座席が受け取る形も大きくは変わらない。

chunk は採らない。塊のあいだの時間に頼るうえ、2000 字の文では `send-text` を
13 回呼ぶことになり、pane のロックを持つ時間が 2 秒ほど延びる。

codex の座席の打ち方は変えない。0.75 秒後の Enter で送られない件は、画面の
規則の変更と合わせて別の issue で扱う。

## 打ち込みが途中で切れたときの確かめ

計画の助言で次の所見が出た。開始の印が届いて終わりの印が届く前に打ち込みが
切れると、claude が貼り付けを集め続け、次の send や resume の文を前の文に
取り込むのではないか。ralph は `herdr pane send-text` を
`exec.CommandContext` で呼ぶので、`--timeout-ms` の期限が来ると herdr の
client は SIGKILL で止まる。そこで次の 2 つを確かめた。どちらも Enter は
押しておらず、claude の手番は使っていない。

版は上と同じ。分離した herdr は同じ手順で `/tmp/ml319b/` の下に作り直した
(server の pid 97955)。ユーザーの `herdr server`(pid 34407)は、確かめの前と
あとで同じ pid で動いていた。分離した server は最後に `herdr server stop` で
止めた。

### herdr の client を途中で止める

shell の pane で、標準入力を raw にして、読んだバイトをすべてファイルに書く
受け手(python)を動かした。そこへ 3000 行・300000 バイトの文を
`herdr pane send-text` で 1 回打った。client を止めずに打つと、client は
0.505 秒で戻り、300000 バイトすべてが届いた。

同じ文を打ち、client を 0.01 秒から 0.55 秒のあいだで `kill -9` した。

| 組 | kill までの秒 | 結果 |
|---|---|---|
| 1 | 0.01、0.05、0.1、0.2、0.3 | 5 回とも文は 0 バイト |
| 2 | 0.3、0.4、0.45、0.48、0.5、0.55、0.4、0.45 | 0.48 の回は kill の前に client が終わり(rc 0)、300000 バイトが届いた。残りの 7 回は 0 バイト |

- 1 の組では受け手の待ちを 60 秒にしていたので、次の回のコマンドの行が前の
  回の受け手に入った。0.05 と 0.2 の回は自分の受け手を持たなかったが、その
  文が届いていれば、まだ raw で読んでいた前の回の受け手に入ったはずである。
  前の回の受け手が受け取ったのは、次の回のコマンドの行(57 バイトと
  56 バイト)だけだった。2 の組では受け手の待ちを 8 秒にした。
- すべての回のあと、shell の画面に文の行は流れ込んでいなかった。

kill した 12 回はすべて、文が 1 バイトも届かなかった。途中まで届いた回は
なかった。herdr の中でなぜそうなるのかは見ていない。server が pane に
書いている最中に client が止まる場合を、これで除けたわけではない。

### 終わりの印が来ない貼り付けを claude がどう扱うか

claude の座席(haiku)に、開始の印と 19 行・906 バイトの文だけを打ち、終わりの
印は打たなかった。

- 1 秒後の入力欄は空(`❯` だけ)で、5 秒後には
  `❯ [Pasted text #1 +18 lines]` になっていた。
- そのあと素の短い文 `NEXT-SEND-PLAIN` を打つと、入力欄は
  `❯ [Pasted text #1 +18 lines]NEXT-SEND-PLAIN` になった。貼り付けには
  取り込まれず、置き換えの後ろに打鍵として並んだ。
- 終わりの印だけを打っても、画面は変わらなかった。
- `herdr pane send-keys <pane> ctrl+c` を 1 回押すと、入力欄は空になった。
- 入力欄を空にしてから同じ文をもう一度打ち、0.2 秒おきに読んだ。打ち始めて
  から 1.98 秒の読みでは空で、2.36 秒の読みで `[Pasted text #2 +18 lines]` が
  出た。claude は終わりの印が来ない貼り付けを約 2 秒で閉じる。

読み取れたこと。

- 開始の印のあとで打ち込みが切れても、claude が集め続けるのは約 2 秒で、その
  後の入力は取り込まれない。`ralph org send` は pane のロックを Enter と送信の
  確かめが終わるまで持つので、その 2 秒のあいだに別の send や resume が同じ
  pane に打つことはない。
- 集めているあいだに ralph が Enter を押すと、Enter も貼り付けに入るとみて
  いる(未確認)。そのとき herdr の道では送信の確かめが working を待てずに
  `submit_unconfirmed=true` を付けて知らせ、画面の道では Enter の前の確かめが
  通らずに Enter を押さない。どちらも知らせる向きになる(推測)。
- 入力欄に残った置き換えは、Ctrl+C を 1 回押すと消える。

残ったもの。

- `~/.claude/projects/-private-tmp-ml319b-work/aae85369-b16f-4c61-837a-c6baca19ff3d.jsonl`
  (8 行で、user の文はない)。
- `~/.claude.json` の `projects` の、`/private/tmp/ml319b/work` と、上の確かめの
  `/private/tmp/ml319/work` の 2 つの項目(claude のフォルダ信頼の記録)。
  動いている claude が書くファイルなので、手では消していない。
- `/tmp/ml319b/` はこの節を書いたあとに消した。

## 確かめで残ったもの

- `~/.claude/projects/-private-tmp-ml319-work/`(セッションの記録
  `48afc10d-8eb1-4f36-9b3c-f90c3d889963.jsonl` と、claude が作った `memory/`)。
- `~/.codex/sessions/2026/11/13/rollout-2026-11-13T11-56-53-01a1d2d6-2b4b-7f31-a71a-a93a83ff82af.jsonl`。
- `~/.claude/history.jsonl` の 3 行と `~/.codex/history.jsonl` の 2 行。
- codex がフォルダ信頼で `~/.codex/config.toml` に足した
  `[projects."/private/tmp/ml319/work"]` の 3 行は手で消した。確かめの前に取った
  写しとの diff は空だった。
- `/tmp/ml319/` はこの文書を書いたあとに消した。

## 確かめていないこと

- `ralph org send` を通した送信。直したあとに 1 回確かめる(計画の受け入れ条件)。
- `--raw` で送る、10 KB を超える文。claude がどこまで 1 つの貼り付けとして
  受けるかは見ていない。
- 1 行で 800 バイトを超える文を bracketed paste で打ったときの置き換えの形。
  `+N lines` が付かない形なら、画面の道の 3 つ目の確かめは通らない。typed
  protocol の文は見出しの行と空行を持つので、この形になるのは `--raw` の文だけ。
- 入力欄に前の置き換えが残ったまま、同じ文をもう一度貼り付けたとき。
- codex の Enter が送信にならなくなる文の大きさ。
- 負荷をかけたときの chunk の届き方。
- herdr の server が pane に書いている最中に止まったときの届き方。
- 終わりの印が来ない貼り付けを claude が集めているあいだに Enter を押したとき
  の動き。
