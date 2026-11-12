# 手番の途中と入力待ちの座席の画面と herdr の状態(2026-11-12)

計画 `docs/plans/active/2026-11-12-send-screen-fallback.md`(issue #308)の S1 の
記録。分離した herdr の上で claude と codex の座席を 1 つずつ立て、手番の途中
(working)と入力待ち(idle)の画面を読み、そのときの herdr の `agent_status` と
合わせて残した。読んだ画面は `internal/org/testdata/resume/` に足した(最後の
節に一覧)。

結果を先に書く。

- 動作中の claude の画面でも、入力欄は空の `❯` のままだった。動作中かどうかは
  入力欄の上の行(`✶ Deliberating… (6s · ↓ 100 tokens)` の形)でしか分からない。
  `esc to interrupt` はどの画面にも出なかった。
- codex の動作中の行は `• Working (6s • esc to interrupt)` で、入力欄には動作中
  も入力待ちも同じ placeholder(`› Ask Codex to do anything`)が出ていた。
- 動作中に 3 秒あけて 2 回読むと、どちらの CLI でも経過の秒数が変わるので中身が
  違った。入力待ちの画面は 10 秒以上あけても同じだった。
- 手番の途中に文を打って Enter を押すと、どちらの CLI も手番の終わりを待たずに、
  その手番の中で文をモデルに渡した。モデルは新しい文だけに答え、最初の依頼が
  求めた `done` は書かれなかった。計画の Assumptions の 1 つ目(次の入力として
  持ち、今の作業は壊れない)は、この確認では成り立たなかった(5 節)。
- 今の判定(`ClassifyResumeScreen`)は、この確認の codex の画面をすべて
  `input box has text` とする。codex 0.166.0 の footer の 1 行目は
  `Context 7% used` のあとにも項目が続き、`resumeCodexFooterLine` に合わない
  ため(6 節)。
- herdr は、この確認で見た working と idle をすべて端末のタイトルか画面の規則から
  決めていた(`herdr agent explain`)。#308 の症状(手番が終わっても working の
  まま)は再現しなかった。

## 環境と分離

- macOS 26.5.2、Claude Code 2.1.316、codex-cli 0.166.0、herdr 0.7.5。
- 座席のモデルは claude が `haiku`(footer の表示は `Haiku 5.5`)、codex が
  `gpt-6-luna`(`~/.codex/models_cache.json` にあり、`upgrade` の項は null)と
  `-c model_reasoning_effort=low`。
- pane は幅 53 桁、高さ 23 行だった。`--lines 40` で読んでも 23 行が返る。

分離は `docs/recipes/isolated-herdr-server.md` のとおり。

- 専用の場所は `/tmp/rs308`(`mkdir -m 700`)。
- 1 つのコマンドを `env -i` で走らせるラッパー `/tmp/rs308/run` を作り、
  `herdr` はすべてこれを通した。渡した変数は `HOME`(実際のもの)、`USER`、
  `LOGNAME`、`SHELL=/bin/bash`、`LANG`、`TERM=xterm-256color`、`TMPDIR`、
  `PATH`(`/tmp/rs308/bin`、claude、codex、node の実体のディレクトリ、
  `/opt/homebrew/bin`、システムのディレクトリ)、`XDG_CONFIG_HOME` と
  `XDG_STATE_HOME`(`/tmp/rs308` の下)、`HERDR_SOCKET_PATH=/tmp/rs308/h.sock`、
  `CLAUDE_ENV_FILE`。座席は commit も push もしないので、`GIT_CONFIG_GLOBAL` と
  `GH_CONFIG_DIR` は渡していない。ralph、agmsg、ledger は使っていない。
- `run herdr --help` の `Config:` と `Logs:` は `/tmp/rs308/xdg-config/herdr/` を
  指した。
- サーバーは `nohup /tmp/rs308/run herdr server &` で起動した。`lsof -p` で見ると、
  開いているのは `/tmp/rs308` の下のファイル(`herdr-server.out`、
  `xdg-config/herdr/herdr-server.log`)とソケット、`/dev/null` だけで、作業
  ディレクトリは `/private/tmp/rs308` だった。`ps eww` の変数名はラッパーが渡した
  ものだけだった。
- ユーザーの herdr サーバーには何も送っていない。始める前と片付けたあとの
  `pgrep -fl 'herdr server'` の両方に、同じ pid で残っていた。
- 座席の cwd は `/tmp/rs308/work` の scratch の git repo(`main` に base の
  コミット、repo の `user.name` と `user.email`、空のディレクトリを指す
  `core.hooksPath`)。
- pane は座席ごとに `herdr workspace create --cwd /tmp/rs308/work` で作り、ralph が
  座席の pane に付ける `--env` を同じように付けた(`AGMSG_SELF_NAME=off`、
  `RALPH_WIP_CHECKPOINT=off`、`RALPH_PRIMARY_CLI=<driver>`。claude にはさらに
  `[org].api_wait_minutes` の既定 30 分の 3 つ、`CLAUDE_CODE_RETRY_WATCHDOG=1`、
  `CLAUDE_CODE_RETRY_WATCHDOG_MAX_WAIT_MS=1800000`、`CLAUDE_CODE_MAX_RETRIES=55`)。
  claude の pane の shell では `type claude codex` が実体のパスを出し、`alias` は
  何も出さなかった。
- 座席は ralph と同じ引数で起動した。
  - `herdr agent start rs-claude --kind claude --pane <pane> -- --model haiku --permission-mode bypassPermissions`
  - `herdr agent start rs-codex --kind codex --pane <pane> -- --sandbox workspace-write --ask-for-approval never --model gpt-6-luna -c model_reasoning_effort=low`
- 文は ralph と同じ手順で打った。`herdr pane send-text <pane> <文>` のあと
  750 ms 待ち(`DefaultSendEnterDelayMS`)、`herdr pane send-keys <pane> Enter`。
- 画面は `herdr pane read <pane> --source recent --lines 40 --format text`、状態は
  `herdr agent get <name>` の `agent_status` で読んだ。各場面のあいだ、別の
  プロセスで約 2 秒ごとに `agent get` を読み続けた。以下の時間は Enter を押した
  時刻からの秒数。

## 1. 最初の画面

### claude

起動するとフォルダ信頼の画面が出て、herdr は `blocked` と報告した。2.1.316 の
画面には番号がなく、カーソルは `No, exit` にあった。Down と Enter で
`Yes, I trust this folder` を選んだ。bypass permissions の同意の画面は出なかった
(このマシンでは前に同意している)。

```text
 Security guide

 ❯ No, exit
   Yes, I trust this folder

 Enter to confirm · Esc to cancel
```

信頼したあとの入力待ちの画面では、空の入力欄に placeholder の文が出た。herdr は
`idle` と報告した。

```text
                                 ◐ medium · /effort
─────────────────────────────────────────────────────
❯ Try "how do I log an error?"
─────────────────────────────────────────────────────
  🤖 Haiku 5.5 | ⚡ medium | 🌿 main
  ⏵⏵ bypass permissions on (shift+tab to cycle) · …
```

### codex

3 つの画面が順に出た。

1. 更新の案内(`Update available · 0.166.0 → 0.167.2`、`1. Update now` /
   `2. Skip` / `3. Skip until next version`)。herdr は `blocked`。Down と Enter で
   `2. Skip` を選んだ。
2. フォルダの信頼(`Folder access`、`Trust this folder?`、`1. Trust and continue`
   / `2. Quit`)。herdr は `done`。Enter で `1. Trust and continue` を選んだ。
3. フックの確認(`Hooks need review`、`9 hooks are new or changed.`、3 択)。
   herdr は `done`。Down 2 回と Enter で
   `3. Continue without trusting (hooks won't run)` を選んだ。
   `2. Trust all and continue` は、ユーザーの 9 個のフックをユーザー自身の codex
   でも sandbox の外で動かすことになるので選ばなかった。2026-11-07 の確認
   (`docs/evidence/org-seat-resume-probe-2026-11-07.md` の 4 節)も Esc で
   飛ばしている。このため、この確認の codex では `~/.codex/hooks.json` の
   フック(herdr の SessionStart のフックを含む)は動いていない。

フックの確認に答えて 6 秒後の `agent get` は `working` だった。画面はすでに
入力待ちだった。その画面を保存したあと(数秒後)から 40 秒間、2 秒ごとに読んだ
状態はすべて `done` で、40 秒あけた 2 回の画面は同じだった。

```text
› Ask Codex to do anything

  GPT-6-Luna · low · Context 0% used · main · No chan…
  ? for shortcuts                           ⚠ 11 · f2
```

## 2. 短い返事のあとの入力待ち

### claude(`say hi`)

| Enter からの秒数 | herdr | 画面 |
|---|---|---|
| +0.6、+2.7 | working | +1 で `✳ Elucidating…`、+3 で `✳ Pollinating… (3s · thinking)` |
| +4.8 以降 | idle | 返事、`✻ Cooked for 3s · done 6:09 PM`、空の `❯` |

```text
❯ say hi

⏺ こんにちは。何かお手伝いできることはありますか？

✻ Cooked for 3s · done 6:09 PM







─────────────────────────────────────────────────────
❯
─────────────────────────────────────────────────────
  🤖 Haiku 5.5 | ⚡ medium | 🧠 ░░░░░░░░░░ 7% | 🌿…
  ⏵⏵ bypass permissions on (shift+tab to cycle)
```

返事が日本語なのは、座席の claude が実際の `~/.claude` の設定を読むため。+10 と
+20 の画面は同じだった。

### codex(`say hi`)

| Enter からの秒数 | herdr | 画面 |
|---|---|---|
| +0.3 | done | `› say hi` が履歴に移り、動作中の行はまだない |
| +1.1、+2.7、+3.1 | working | `• Working (0s • esc to interrupt)`、`(2s` |
| +4.9 以降 | done | 返事、`Worked for 4s • 18:17`、placeholder の入力欄 |

```text
• Hi!

• Hook failed
  └ hook exited with code 1

• Hook failed
  └ plugin contents changed; hook approval required

  Worked for 4s • 18:17


› Ask Codex to do anything

  GPT-6-Luna · low · Context 7% used · main · No chan…
  ? for shortcuts                           ⚠ 11 · f2
```

`Hook failed` の行は手番ごとに出た。どのフックかは調べていない。+15 と +30 の
画面は同じだった。

## 3. 手番の途中(シェルの命令を実行中)

claude には `perl -e 'select(undef,undef,undef,30)'` を、codex には
`sleep 30` を前面で走らせてから `done` と答えるよう頼んだ。claude に perl を
使ったのは、Claude Code の Bash ツールが単独の `sleep N` を断った記録がある
ため(`docs/evidence/busy-leader-2026-10-23.md`。2.1.316 では試していない)。

### claude

herdr は +0.5 から +36.3 まで `working`、+38.4 から `idle`。+2、+6、+9、+15、
+25 の画面では、入力欄はずっと空の `❯` だった。

```text
⏺ Waiting 30 seconds in the foreground · 3s
  ⎿  $ perl -e 'select(undef,undef,undef,30)'

✶ Deliberating… (6s · ↓ 100 tokens)
  ⎿  Tip: Use ctrl+v to paste images from your
     clipboard

─────────────────────────────────────────────────────
❯
─────────────────────────────────────────────────────
```

- 動作中の行は、記号、動詞、`…` の順で始まる。記号は読むたびに変わり、この
  確認では `✳`、`✢`、`✶`、`✻`、`✽`、`·` を見た。括弧の中は経過の秒数と
  トークン数で、+2 では
  `✢ Deliberating… (running PreToolUse hooks… 5/10 · 2s` と
  ` · ↓ 14 tokens)` の 2 行に折り返した。
- 手番の終わりの行(`✻ Cooked for 36s · done 6:11 PM`)も `✻` で始まる。違い
  は、動詞のあとが `…` か `for <時間> · done <時刻>` か。
- 動作中の行と入力欄のあいだに `⎿  Tip: …` の行が入ることがある(空行で区切ら
  れない)。
- ツールの行の前の `⏺` は +25 の画面で消えていた(点滅とみられる)。
- +6 と +9 の 2 回の読みは、`(6s` と `(9s`、ツールの経過(`· 3s` と `· 6s`)、
  `(5s)` と `(ctrl+b to run in background)` の行の有無で違った。

### codex

herdr は +0.9 から +34.6 まで `working`、+36.8 から `done`。命令そのものは画面に
出ず、動作中の行の後ろに実行中の端末の数が付いた(pane の幅で切れる)。入力欄は
ずっと placeholder だった。

```text
• Working (6s • esc to interrupt) · 1 background term…


› Ask Codex to do anything

  GPT-6-Luna · low · Context 7% used · main · No chan…
  ? for shortcuts                           ⚠ 11 · f2
```

+6 と +9 の 2 回の読みは `(6s` と `(8s` だけが違った。終わったあとは
`• done` と `Worked for 34s • 18:19` が出た。

herdr の `revision` は、claude では読むたびに増えたが、codex では手番のあいだも
変わらなかった(2 のまま)。`state_change_seq` はどちらも状態が変わるたびに増えた。

## 4. 考え中

claude は手番の初めに `✳ Elucidating…`(括弧なし)、3 秒後に
`✳ Pollinating… (3s · thinking)` を出した(2 節の表)。codex は effort が low の
この確認では、考え中にも `• Working (Ns • esc to interrupt)` だけを出した。

## 5. 手番の途中に打った文(計画の Assumptions の確かめ)

3 節と同じ依頼を送り、8 秒後に `Reply with the single word: queued` を打ち、
750 ms 待って Enter を押した。

### claude

打った直後(Enter の前)、文は入力欄に入っていた。動作中の行はその上にあった。

```text
✳ Generating… (9s · ↓ 95 tokens)
  ⎿  Tip: Run /ultrareview for a cloud-based
     multi-agent review that finds and verifies bugs
     in your branch — 3 free reviews left

─────────────────────────────────────────────────────
❯ Reply with the single word: queued
─────────────────────────────────────────────────────
```

Enter のあと(+10、+13)、文は動作中の行の上に移り、入力欄には別の文が出た。

```text
❯ Reply with the single word: queued
  ctrl+enter to send now

✶ Generating… (10s · ↓ 100 tokens)
  ⎿  Tip: Run /ultrareview for a cloud-based
     multi-agent review that finds and verifies bugs
     in your branch — 3 free reviews left

─────────────────────────────────────────────────────
❯ Press up to edit queued messages
─────────────────────────────────────────────────────
```

座席のセッションの記録(`~/.claude/projects/-private-tmp-rs308-work/` の
jsonl。種類と時刻だけを jq で読んだ)では、文は +9.4 に `queue-operation` の
`enqueue` で積まれ、ツールの結果が出た直後の +38.6 に `remove` で取り出され、
`queued_command` として同じ手番の次の要求に付いた。モデルの返事は `queued` だけ
で、最初の依頼の `done` は書かれなかった。手番の終わりの行は
`✻ Brewed for 42s · done 6:13 PM` の 1 つで、herdr は +2.1 から +41.9 まで
`working`、+44.0 から `idle` だった。新しい手番は始まっていない。

### codex

打った直後(Enter の前)、文は入力欄に入り、footer の 2 行目が
`tab to queue message` に変わった。

```text
• Working (8s • esc to interrupt) · 1 background term…


› Reply with the single word: queued

  GPT-6-Luna · low · Context 7% used · main · No chan…
  tab to queue message
```

Enter で文はすぐ手番に入った(codex の steer)。セッションの記録
(`~/.codex/sessions/2026/11/12/` の rollout の jsonl。種類と時刻と短い抜き出し
だけを読んだ)では、`sleep 30` を待っていたツールの呼び出し(`yield_time_ms` は
30000)が Enter と同時の +9.1 に `Wall time 5.9 seconds` で返り、命令は動いた
まま残った。そのすぐあとに打った文が同じ手番に入り、+11.1 にモデルが `queued` と
答え、+11.5 に手番が終わった。`done` は書かれなかった。

+10 の画面では文が動作中の行の上の履歴に移り、+13 の画面は次のとおり。

```text
• queued

• Hook failed
  └ hook exited with code 1

• Hook failed
  └ plugin contents changed; hook approval required
  └ Tip: Press f3 to search this conversation.

  Worked for 11s • 18:19

  1 background terminal running · /ps to view · /stop


› Ask Codex to do anything
```

herdr は +2.4 から +10.9 まで `working`、+13.0 から `done`。+45 では
`1 background terminal running` の行が消えていた(裏で `sleep` が終わった)。
新しい手番は始まらなかった。

### 分かったこと

- どちらの CLI も、手番の途中に打たれた文を受け付け、手番の終わりを待たずに
  その手番の中でモデルに渡した。claude は走っているツールの結果のあとに渡し、
  codex は Enter で走っているツールの待ちを切り上げて渡した(命令は裏の端末で
  続いた)。
- どちらでも、モデルは新しい文だけに答え、最初の依頼の終わりの指示は果たさ
  れなかった。それぞれ 1 回ずつの結果。
- 計画の Assumptions の「次の入力として持つ」「今の作業は壊れない」は、この
  確認では成り立たなかった。手番の途中に打つと、その手番の作業が変わる。
- codex の `tab`(queue)と claude の `ctrl+enter` は試していない。

## 6. 判定に効く画面の形

### 送った直後

- claude: Enter から 0.27 秒後には、打った文が履歴に移り
  (`❯ Reply with the single word: second`)、その下に `✽ Bunning…`、入力欄は
  空の `❯` だった。
  1.1 秒後は `· Bunning…`(記号が変わる)。herdr は 0.27 秒後に `working`、
  2 秒ごとの読みで +3.3 から `idle`。
- codex: 0.27 秒後には、文が履歴に移り、`• Working (0s • esc to interrupt)`、
  入力欄は placeholder だった。2 節の `say hi` では、0.3 秒後に文は履歴に移って
  いたが、動作中の行はまだなく、herdr も `done` のままだった。
- どちらも、最初の読みの時点で入力欄から打った文は消えていた。

### 入力欄

- claude は入力欄に placeholder を出すことがある。新しいセッションの
  `❯ Try "how do I log an error?"` と、文が積まれているあいだの
  `❯ Press up to edit queued messages`。plain text では打った文と見分けられない。
  返事のあとの入力待ちでは空の `❯` だった。
- claude の新しいセッションでは、上の罫線のすぐ上に右寄せの
  `◐ medium · /effort` の行があった。
- codex の入力欄は、動作中も入力待ちも `› Ask Codex to do anything` だった。
- codex の footer の 1 行目は
  `GPT-6-Luna · low · Context 0% used · main · No chan…` で、`Context N% used`
  のあとにブランチと変更の要約が続き、pane の幅で切れる。今の
  `resumeCodexFooterLine` は状態の行が `Context N% used` か `left` で終わることを
  求めるので、この行は footer にならず、入力欄に文があるとみなされる。2 行目は
  `? for shortcuts                           ⚠ 11 · f2`、手番の途中に打つと
  `tab to queue message`。
- 今の `ClassifyResumeScreen` をこの確認の画面すべてに当てた(使い捨ての
  テストを `go test -overlay` で走らせ、commit していない)。claude の入力待ちと
  動作中の画面は `no known stop sentence above the input box`、placeholder の
  2 つと入力欄に文がある画面は `input box has text`、番号のない信頼の画面は
  `no input box`、codex の画面は選択の 2 つ(`choice dialog`)を除いてすべて
  `input box has text` だった。

### 動作中の行の文

| CLI | 動作中 | 手番の終わり |
|---|---|---|
| claude | `✶ Deliberating… (6s · ↓ 100 tokens)`、`✳ Pollinating… (3s · thinking)`、`✳ Elucidating…`、`✽ Bunning…`(記号は `✳ ✢ ✶ ✻ ✽ ·` が入れ替わる) | `✻ Cooked for 3s · done 6:09 PM` |
| codex | `• Working (6s • esc to interrupt)`、後ろに `· 1 background terminal running`(幅で切れる) | `Worked for 4s • 18:17`(記号なし、2 字下げ)、ときに `1 background terminal running · /ps to view · /stop` |

### 2 回の読み

動作中は、3 秒あけた 2 回の読みがどちらの CLI でも違った(秒数、記号、ツールの
経過)。入力待ちの画面は、claude で +10 と +20、codex で +15 と +30、codex の
起動直後で 0 秒と 40 秒が、それぞれ同じだった。

## 7. herdr の状態の決め方

`herdr agent explain <name>` は、状態を決めた規則と根拠を出す。最後に両方の
座席へ 3 節と同じ形の依頼(待ちを 15 秒にしたもの)を同時に送り、+1、+6、+30 で
読んだ。

| 座席 | 時点 | state | rule | evidence |
|---|---|---|---|---|
| claude | 手番の途中(+1、+6) | working | `osc_title_working` | `◑ Perl sleep command test`、`◐ Perl sleep command test` |
| claude | 終わったあと(+30) | idle | `live_prompt_box` | `❯` |
| codex | 手番の途中(+1、+6) | working | `osc_title_working` | `⠙ Say hi \| work`、`⠋ Say hi \| work` |
| codex | 終わったあと(+30) | idle | `osc_title_idle` | `Say hi \| work` |

evidence は端末のタイトル(OSC)か画面の行。規則は、サーバーが起動時に取って
きた manifest(`$XDG_STATE_HOME/herdr/agent-detection/remote/claude.toml` と
`codex.toml`、どちらも版 2026.10.11.1、`status.toml` に取得の記録)にあり、
`explain` の出力では `manifest: remote:...` と表示される。herdr の binary を
変えずに規則が変わりうる(推測。実行中に取り直すかは確かめていない)。

- codex: タイトルが点字の回転記号を含むと working(`osc_title_working`、優先度
  1050)、含まなければ idle(`osc_title_idle`、優先度 100)。画面の規則
  `screen_working_fallback`(優先度 500)は、今の入力欄の上に
  `… (Ns • … to interrupt)` の行があれば working とする。この規則には、codex が
  文を積むときの文(`Queued follow-up inputs`、
  `Messages to be submitted after next tool call`、
  `Messages to be submitted at end of turn`)と
  `Reconnect failed — check the endpoint, then relaunch` が書かれている。この
  確認の画面には出ていない。
- claude: タイトルが点字か `◐◑◒◓` で始まると working。画面の下の 12 行に、
  `* · ✢ ✳ ✶ ✻ ✽` のどれか、語、`…` の順の行(後ろが ` (Ns` か行末)があっても
  working(`live_turn_working`)。`⏸` か `⏵` で始まり `esc to interrupt` を含む
  行も working とする規則があるが、2.1.316 のこの幅の画面には出なかった。入力欄
  の `❯` の行があれば idle(`live_prompt_box`)。
- 手番が終わってもタイトルに回転記号が残れば、herdr は working と言い続ける
  ことになる。#308 の症状に至る道の 1 つと考えられるが、推測で、確かめていない。

## 8. herdr の連携の状態

ラッパーを通した `herdr integration status` の出力。ホームのパスを `~` に置き
換えた以外はそのまま。

```text
pi: not installed (~/.pi/agent/extensions/herdr-agent-state.ts)
omp: not installed (~/.omp/agent/extensions/herdr-omp-agent-state.ts)
claude: current (v7) (~/.claude/hooks/herdr-agent-state.sh)
codex: not installed (~/.codex/herdr-agent-state.sh)
copilot: not installed (~/.copilot/hooks/herdr-agent-state.sh)
devin: not installed (/tmp/rs308/xdg-config/devin/herdr-agent-state.sh)
droid: not installed (~/.factory/hooks/herdr-agent-state.sh)
kimi: not installed (~/.kimi-code/hooks/herdr-agent-state.sh)
opencode: not installed (~/.config/opencode/plugins/herdr-agent-state.js)
kilo: not installed (~/.config/kilo/plugin/herdr-agent-state.js)
hermes: not installed (~/.hermes/plugins/herdr-agent-state/__init__.py)
qodercli: not installed (~/.qoder/hooks/herdr-agent-state.sh)
cursor: not installed (~/.cursor/herdr-agent-state.sh)
mastracode: not installed (~/.mastracode/hooks/herdr-agent-state.sh)
```

終了コードは 0。1 行が `<target>: <状態> (<パス>)` の形で、状態は
`current (v7)` か `not installed` だった。devin のパスだけが `XDG_CONFIG_HOME`
に従った。

このマシンの連携のファイルについて確かめたこと。

- `~/.codex/herdr-agent-state.sh` はない。`~/.codex/hooks/herdr-agent-state.sh`
  があり、`~/.codex/hooks.json` の SessionStart がこれを
  `bash ... session` で呼ぶ。
- `~/.codex/hooks/herdr-agent-state.sh` と `~/.claude/hooks/herdr-agent-state.sh`
  はバイトまで同じで、どちらも `HERDR_INTEGRATION_ID=claude`、
  `HERDR_INTEGRATION_VERSION=7` と書かれ、`agent` を `claude` として報告する。
  herdr 0.7.5 の binary には codex 用の別のスクリプト
  (`HERDR_INTEGRATION_ID=codex`)が入っている。
- このスクリプトは SessionStart で `pane.report_agent_session` を 1 回送るだけで、
  working や idle は送らない(スクリプトを読んで確かめた)。`HERDR_ENV=1`、
  `HERDR_SOCKET_PATH`、`HERDR_PANE_ID` のどれかが欠けると何もせずに終わり、
  `python3` を呼ぶ。claude の pane の shell では `command -v python3` が
  `/usr/local/bin/python3` を出した。
- `~/.claude/settings.json` で herdr のスクリプトを呼ぶのは SessionStart の 1 つ
  だけ。

## 9. #308 の症状

再現しなかった。claude と codex のどの場面でも、手番の終わりの行が出てから
2 秒ごとの読みの 1 回か 2 回のうちに、herdr は `idle` か `done` になった。
食い違いは次の 2 つだけ。

- codex の起動直後、フックの確認に答えて 6 秒後の `agent get` は `working` で、
  画面は入力待ちだった。数秒後に始めた 2 秒ごとの読みは、最初から `done` だった。
- codex のフォルダ信頼とフックの確認の選択画面のあいだ、herdr は `done` と報告
  した(更新の案内は `blocked`)。2026-11-07 の確認でも、herdr はフックの確認を
  `idle` と報告している。

## 確かめで残ったもの

専用の場所の外、ユーザーの持ち物に次のものが残った。手で書き換えたものはない。

- `~/.claude.json` の `projects` の `/private/tmp/rs308/work`(claude の
  フォルダ信頼)。
- `~/.codex/config.toml` の末尾に足された 3 行
  (`[projects."/private/tmp/rs308/work"]`、`trust_level = "trusted"`、空行)。
  始める前に取った写しとの diff はこの 3 行だけで、`~/.codex/hooks.json` は
  変わっていない。
- claude のセッションの記録 `~/.claude/projects/-private-tmp-rs308-work/` と、
  codex のセッションの記録 `~/.codex/sessions/2026/11/12/` の rollout の 1 つ。
- `~/.claude/history.jsonl` と `~/.codex/history.jsonl` の行、codex の
  キャッシュとログ(`models_cache.json`、`logs_2.sqlite` など)、座席の claude が
  動かしたユーザーのプラグインとフックが書くもの。

片付けは recipe の順に行った。2 つの workspace を閉じ(`herdr agent list` が空)、
`pgrep -fl /tmp/rs308` に何もなく、`lsof +D /tmp/rs308` が専用のサーバーだけで
あることを見てから、`run herdr server stop` で専用のサーバーを止めた。ソケットが
消え、`lsof +D` が何も出さないことを見て、`/tmp/rs308` を消した。ユーザーの
サーバーは最後まで動いていた。

この記録と testdata には、token、API key、アカウントのメールアドレス、org の id、
pane の id、セッションの id を写していない。ホームのパスは `~` と書いた。

## 確かめていないこと

- #308 の症状の再現と、その原因。
- codex のフックを信頼した状態(herdr の SessionStart のフックが codex でも
  動く状態)。
- 幅の広い pane。footer の項目と切れる位置は幅 53 桁でのもの。
- claude の `esc to interrupt` の行、codex の文を積む表示
  (`Queued follow-up inputs` など)、codex の `tab` と claude の `ctrl+enter`。
- 手番の途中に打ったときの動きの再現性(各 1 回)と、ツールを走らせていない
  考え中に打った場合。
- 承認の画面や AskUserQuestion の画面(bypass と `never` なので出ない)。
- 別のモデルや effort での表示。
- herdr が manifest を取り直す時機。

## 足した testdata

すべて `internal/org/testdata/resume/` にある。
`herdr pane read --source recent --lines 40 --format text` の出力をそのまま
1 ファイルずつ置いた(23 行)。
`claude-folder-trust-unnumbered.txt` だけは、先頭の 19 行(pane の shell の
出力で、ホスト名とホームのパスを含む)を落とした。「今の判定」は今の
`ClassifyResumeScreen` の結果。

| ファイル | 場面 | herdr | 今の判定 |
|---|---|---|---|
| `claude-idle-fresh-placeholder.txt` | 信頼のあとの入力待ち(placeholder) | idle | unknown: input box has text |
| `claude-idle-after-reply.txt` | 2 節の +10 | idle | unknown: no known stop sentence |
| `claude-working-thinking-start.txt` | 2 節の +1(`✳ Elucidating…`) | working | unknown: no known stop sentence |
| `claude-working-thinking.txt` | 2 節の +3(`(3s · thinking)`) | working | unknown: no known stop sentence |
| `claude-working-tool-hooks.txt` | 3 節の +2(折り返した動作中の行) | working | unknown: no known stop sentence |
| `claude-working-tool.txt` | 3 節の +6 | working | unknown: no known stop sentence |
| `claude-working-tool-3s-later.txt` | 3 節の +9 | working | unknown: no known stop sentence |
| `claude-working-input-typed.txt` | 5 節、打った直後の Enter の前 | working(打つ直前) | unknown: input box has text |
| `claude-queued-while-working.txt` | 5 節の +10 | working | unknown: input box has text |
| `claude-submitted-just-now.txt` | 送って 0.27 秒後 | working | unknown: no known stop sentence |
| `claude-submitted-1s.txt` | 送って 1.1 秒後 | working | unknown: no known stop sentence |
| `claude-folder-trust-unnumbered.txt` | 番号のない信頼の画面 | blocked | unknown: no input box |
| `codex-idle-fresh-update-notice.txt` | フックの確認のあとの入力待ち | working(9 節) | unknown: input box has text |
| `codex-idle-after-reply.txt` | 2 節の +15 | done | unknown: input box has text |
| `codex-working-thinking.txt` | 2 節の +3 | working | unknown: input box has text |
| `codex-working-tool.txt` | 3 節の +6 | working | unknown: input box has text |
| `codex-working-tool-3s-later.txt` | 3 節の +9 | working | unknown: input box has text |
| `codex-working-input-typed.txt` | 5 節、打った直後の Enter の前 | working(打つ直前) | unknown: input box has text |
| `codex-queued-steer-while-working.txt` | 5 節の +10(steer した文が履歴に) | working | unknown: input box has text |
| `codex-idle-background-terminal.txt` | 5 節の +13(裏の端末が残る) | done | unknown: input box has text |
| `codex-submitted-just-now.txt` | 送って 0.27 秒後 | working | unknown: input box has text |
| `codex-update-available.txt` | 更新の案内の選択画面 | blocked | unknown: choice dialog |
| `codex-folder-trust.txt` | フォルダ信頼の選択画面 | done | unknown: choice dialog |
