# codex の座席への bracketed paste の確かめ(2026-11-15、#326)

#326 の直し方を決めるために、分離した herdr の上の codex の座席に、大きさの違う
5 つの文を bracketed paste の印(`ESC [200~` と `ESC [201~`)で挟んで 1 回の
`herdr pane send-text` で打った。ralph org send と同じく 750 ms 待って Enter を
1 回押し、届いた文を codex のセッションの記録で元の文と比べた。ralph は通して
いない。

結果は次の 4 点。

- 5 つとも、750 ms 後の Enter で送られた。届いた文は 5 つとも元の文とバイト単位で
  同じで、codex は文の最後の行が求めた語を返した。#319 の確かめ
  (`docs/evidence/org-send-paste-probe-2026-11-13.md` の X1)では、5340 バイトの
  素の文が 750 ms 後の Enter で送られなかった。
- 651 バイトまでの文は畳まれず、文が入力欄に出た。651 バイトの文では、入力欄は
  文の終わりの数行だけを出し、1 行目の右端に `↑` を付けた。今の画面の規則
  (文の頭、または `↑` 付きの入力欄に文の終わり)で通る形。
- 1698 バイトの文と日本語の 4925 バイトの文は、入力欄が
  `› [Pasted Content <N> chars]` の 1 行になった。
- N は文の字数(Unicode の符号位置の数)だった。日本語の文には絵文字 `😀` を
  1 つ入れてあり、字数は 2016、UTF-16 の単位なら 2017 になる。画面は
  `[Pasted Content 2016 chars]` だった。字数と書記素の数の違いは確かめていない。

## 環境と分離

- macOS 26.5.2、codex-cli 0.166.0(`gpt-6-luna`、effort low)、herdr 0.7.5。
  pane は幅 53 桁。
- herdr は `docs/recipes/isolated-herdr-server.md` のとおりに分けた。`env -i` の
  包みのスクリプトで `XDG_CONFIG_HOME`、`XDG_STATE_HOME`、`HERDR_SOCKET_PATH` を
  `/tmp/ml308/` の下に向けた。`HOME` は本物のまま。codex の binary は
  `/tmp/ml308/bin/codex` から symlink を張った。
- 座席の cwd は `/private/tmp/ml308/work` の scratch の git repo(codex で信頼済み)。
  更新の案内は `2. Skip`、フックの確認は
  `3. Continue without trusting (hooks won't run)` で答えた。返事の前後には、
  ユーザーの codex のフックの `• Hook failed` の行が出た。
- ユーザーの `herdr server`(pid 34407)には、コマンドを 1 つも送っていない。
  分離した server は最後に `herdr server stop` で止め、`/tmp/ml308` を消した。
- claude の座席は使っていない。codex の手番は 5 回使った。

## 打った文と結果

どれも「道具を使わず、ファイルも読まずに、1 語で答えよ」と頼む文。

| 名前 | 中身 | バイト | 字数 | 改行 | Enter の前の入力欄 | 750 ms 後の Enter | 返事 |
|---|---|---|---|---|---|---|---|
| p0 | `TYPE: QUESTION`、空行、1 行 | 49 | 49 | 2 | 文がそのまま出た | 送られた | `echo` |
| p1 | 同じ形で 1 行が長い | 105 | 105 | 2 | 文がそのまま出た(本文は 2 行に折り返し) | 送られた(0.3 秒後に `• Working`) | `alpha` |
| p2 | `TYPE: TASK`、`TASK_ID:`、空行、10 行 | 651 | 651 | 12 | 文の終わりの数行と、1 行目の右端に `↑` | 送られた | `bravo` |
| p3 | 同じ形で 26 行 | 1698 | 1698 | 28 | `› [Pasted Content 1698 chars]` | 送られた | `charlie` |
| p4 | `TYPE: DECISION`、空行、日本語 2000 字の本文(絵文字 1 つ) | 4925 | 2016 | 32 | `› [Pasted Content 2016 chars]` | 送られた | `delta` |

Enter から 0.3 秒後には、どれも入力欄が空の `› Ask Codex to do anything` に戻って
いた(p0 は 0.3 秒後に送った文が履歴に出ていた)。herdr の状態は、手番の終わりに
`idle` になった。

届いた文は、codex のセッションの記録
(`~/.codex/sessions/2026/11/15/rollout-2026-11-15T20-38-58-….jsonl`)の user の
文と比べた。5 つとも、元の文とバイト単位で同じだった。

## この確かめから決めたこと

- codex の座席にも、claude と同じく bracketed paste で打つ。750 ms の Enter の前の
  待ちを長くする案は、どれだけ延ばせばよいかを測っていない。
- 画面の道の Enter の前の確かめに、codex の置き換えの規則を足す。入力欄の文が
  `[Pasted Content <N> chars]` の 1 つだけで、N が打った文の字数と等しいときに
  通す。それ以外は今までどおり Enter を押さない。
- 置き換えの画面(p3、p4 の Enter の前)と、Enter のあとの画面、畳まれなかった
  画面(p1、p2)を testdata の fixture にする。画面の写しは scratchpad の
  `probe326/out/` にある。

## 確かめていないこと

- 畳まれ始める大きさ(651 バイトと 1698 バイトのあいだ)。どちらの形も、上の
  規則と今の規則のどちらかで通る。
- 字数と書記素の数が違う文(結合文字を含む文)での N。
- 2000 字の本文の上限を超える `--raw` の文、10 KB を超える文。
- 前の置き換えが入力欄に残っているときの貼り付け。
- codex-cli 0.166.0 のほかの版。
- ralph を通した送信(実装のあとに、実機の確かめとして行う)。
