# claude 2.1.316 が 5 時間の上限で待つ画面(2026-11-13、#329)

計画 `docs/plans/active/2026-11-13-claude-5h-usage-limit.md`(#329)の S1 の
記録。Claude Code 2.1.316 の座席がアカウントの 5 時間の上限に当たった画面を
実機で読み、この branch の base の `UsageLimitScreen` と `ClassifyResumeScreen`
に通した。画面は fixture
`internal/org/testdata/resume/claude-5h-limit-continuing.txt` にバイト単位で
写した。

結果を先に書く。

- 上限を言う文は 3 か所に出た。送った文のすぐ下の
  `⎿  You've hit your 5-hour limit · resets 3:50pm`、その下の段落
  `⏺ Usage limit reached · continuing automatically at 3:50pm`(小文字の
  `continuing`)、入力欄の下の欄(footer)の
  `⚠ Usage limit reached · limit resets 3:50pm` と
  `Continuing automatically at 3:50pm`(大文字の `C`)。CLI は解除の時刻に
  ひとりでに続きを始める画面で待っていた。
- base(`c610109f`、コードは main の `bbe2807d` と同じ)では、
  `UsageLimitScreen` が `ok=false` を、`ClassifyResumeScreen` が `waiting`
  (理由 `the CLI is retrying or waiting`)を返した。見張りはこの座席を
  usage_limited に入れず、stall の判定のまま進める。
- Enter の 0.3 秒後の読みでは、まだ `✳ Puzzling…` が出ていた。上限の文は
  1.5 秒後の読みで出た。

## 環境と分離

この画面は、#322(branch `fix/send-input-box-marker`)の確かめを回した別の
セッションが読んだもの。ユーザーの herdr ではなく、分離した herdr のサーバーの
pane で読んだ。手順と分離の詳細は、#322 の確かめの記録(branch
`fix/send-input-box-marker` の
`docs/evidence/org-send-marker-probe-2026-11-13.md`)の「環境と分離」と
「Enter のあとの画面」にある。要点は次のとおり。

- macOS 26.5.2、Claude Code 2.1.316、herdr 0.7.5。モデルは Haiku 5.5
  (effort low)。pane の幅は 53 桁。
- herdr は `docs/recipes/isolated-herdr-server.md` のとおり分けた。`env -i` の
  包みのスクリプトで `XDG_CONFIG_HOME`、`XDG_STATE_HOME`、`HERDR_SOCKET_PATH` を
  `/tmp/ml308/` の下に向けた。ユーザーの `herdr server` にはコマンドを 1 つも
  送っていない。
- 座席の cwd は `/private/tmp/ml308/work` の scratch の git repo。
- 送った文は `TYPE: QUESTION`、空行、
  `❯ explain this. This is a probe: do not run any tool or read any file, and reply with the single word: echo.`
  の 126 バイト(改行 2 つ)。ralph は通さず、`herdr pane send-text` で
  bracketed paste として打ち、750 ms 後に Enter を 1 回押した。
- 読みはすべて `herdr pane read <pane> --source recent --lines 60 --format text`
  (ralph の画面の確かめと同じ読み方)。Enter の 0.3 秒後と 1.5 秒後、それに
  herdr が `idle` か `done` に戻ったあとに読んだ。
- 送ったのは 2026-11-13 15:36 JST。15:50 に CLI が続きを始める前に、分離した
  server を止めた。

## 1. Enter の 0.3 秒後

送った文が履歴に移り、その下に `✳ Puzzling…` が出ていた。入力欄は空。読みの
11 行目から最後までを写す(上の 10 行は起動の見出しと Remote Control の案内)。

```text
❯ TYPE: QUESTION

  ❯ explain this. This is a probe: do not run any
  tool or read any file, and reply with the single
  word: echo.

✳ Puzzling…

─────────────────────────────────────────────────────
❯
─────────────────────────────────────────────────────
  🤖 Haiku 5.5 | ⚡ low | 🌿 main
  ⏵⏵ bypass permissions on (shift+tab to cycle)
```

## 2. Enter の 1.5 秒後

読みは 23 行で、最初の 2 行は空行。herdr が `idle` か `done` に戻ったあとの読みは、
この読みと `cmp` でバイト単位まで同じだった。fixture はこの読みをそのまま写した
もの(sha256 `2b9742d9db5dd3c1c10a4c470f5e90c6be3efb33f6166e4988528f81dfa8bd27`)。
下の写しは空行 2 つも含めて全部の行。8 行目の `⎿` のあとは、半角の空白と
U+00A0 が 1 つずつ並ぶ。この写しでは U+00A0 を半角の空白で書いた(fixture には
元のバイトがある)。

```text


❯ TYPE: QUESTION

  ❯ explain this. This is a probe: do not run any
  tool or read any file, and reply with the single
  word: echo.
  ⎿  You've hit your 5-hour limit · resets 3:50pm
     (Asia/Tokyo)

⏺ Usage limit reached · continuing
  automatically at 3:50pm · esc to cancel

✻ Brewed for 0s · done 3:36 PM

─────────────────────────────────────────────────────
❯
─────────────────────────────────────────────────────
  ⚠ Usage limit reached · limit resets 3:50pm
    Continuing automatically at 3:50pm · Esc to
    cancel
  🤖 Haiku 5.5 | ⚡ low | 🌿 main
  ⏵⏵ bypass permissions on (shift+tab to cycle)
```

上限を言う文は 3 か所にある。

- 送った文のすぐ下の `⎿` の行。`You've hit your 5-hour limit · resets 3:50pm`
  のあとに、時間帯 `(Asia/Tokyo)` が折り返して続く。
- `⏺` で始まる段落。`continuing automatically at` は小文字。
- 入力欄の下の欄の `⚠` で始まる行。`Continuing automatically at 3:50pm` は
  大文字の `C` で始まり、状態の行 `🤖 Haiku 5.5 | ⚡ low | 🌿 main` と
  `⏵⏵ bypass permissions on …` が空行なしで続く。

文はモデルに渡る前に上限で止まったので、返事(`echo`)は出ていない。

## 3. base での判定

追跡ファイルは変えずに、`go test -overlay` で一時のテストを足し、fixture を
この branch の base の関数に通した。base は `c610109f` で、親は main の
`bbe2807d`、変えたのは計画のファイルだけ。

```text
UsageLimitScreen ok=false line=""
ClassifyResumeScreen={Kind:waiting Line:⚠ Usage limit reached · limit resets 3:50pm Continuing automatically at 3:50pm · Esc to cancel 🤖 Haiku 5.5 | ⚡ low | 🌿 main ⏵⏵ bypass permissions on (shift+tab to cycle) Reason:the CLI is retrying or waiting}
```

`ClassifyResumeScreen` が waiting と読む道筋は次のとおり。入力欄のすぐ上の
ブロックは手番の終わりの行なので 1 つ飛ばし、その上の
`⏺ Usage limit reached · continuing automatically at …` が候補になる。待ち中の
文は大文字と小文字を区別するので、候補の小文字の `continuing automatically at`
には当たらない。footer のブロックの `Continuing automatically at` が待ち中の文に
当たり、止まった文を見る前に waiting が決まる。`⎿` の行は候補より上にあるので
読まれない。base の `UsageLimitScreen` は stopped の画面しか上限と見なかったので
`ok=false` になった。#329 に書いた main(`e902705c` 以降)での結果と同じ。

## 4. この branch での判定

S1 の変更のあと、`UsageLimitScreen("claude", …)` は `ok=true` を返す。行は
footer のブロックを 1 行にしたもので、解除の時刻が入る。

```text
⚠ Usage limit reached · limit resets 3:50pm Continuing automatically at 3:50pm · Esc to cancel 🤖 Haiku 5.5 | ⚡ low | 🌿 main ⏵⏵ bypass permissions on (shift+tab to cycle)
```

`ClassifyResumeScreen` は base と同じく waiting のままで、`ralph org resume` は
この画面に何も打たない。`ClassifySendScreen` は busy(理由
`the CLI is retrying or waiting`)を返す。これらは
`TestUsageLimitScreen_EveryFixture`、`TestClassifyResumeScreen_Fixtures`、
`TestClassifySendScreen_EveryFixture`、
`TestWatchUsageLimit_AClaudeSeatWaitingForItsLimitToResetIsUsageLimited` が
固めている。

## 確かめていないこと

- 解除の時刻(15:50)のあとの画面。分離した server をその前に止めた。
- `Continuing shortly` の画面と、ほかの幅、版、モデルの画面。
- 待ちのあいだ `herdr agent get` の出力が変わらないか。見張りの stall の判定は
  これで決まる(計画の Assumptions)。
