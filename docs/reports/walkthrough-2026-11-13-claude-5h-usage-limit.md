# Walkthrough: claude の 5 時間の上限と、解除を待つ画面を usage_limited と見分ける(claude-5h-usage-limit)

- 計画: `docs/plans/archive/2026-11-13-claude-5h-usage-limit.md`(PR の作成のあとに archive へ移す)
- issue: #329。claude 2.1.316 の座席が 5 時間の上限で解除を待つ画面を、見張りが usage_limited と見分けず stall にしていた
- 差分: `git diff bbe2807d...HEAD`(28 ファイル、+1,359 / −105。報告・evidence・insights・計画を除くと約 +524 / −105。Go の本体は約 +142、Go のテストは約 +280、fixture は 23 行、残りは文書)
- `ralph org resume` の判定は変えない。待つ画面は今までどおり waiting で、resume は何も打たない

## 1. 待ちの道: `UsageLimitScreen` は footer だけを読む(読む価値が一番高い)

`internal/org/usage_limit_screen.go`。

- `UsageLimitScreen`(`:65-86`)に `ResumeScreenWaiting` の枝(`:78-83`)が増えた。stopped の枝は変えていない。
- `usageLimitWaitingFooter`(`:95-112`)は `findResumeInputBox` で入力欄を見つけ、`box.footerBlocks`(`resume_screens.go:553`)が返す入力欄の下のブロックだけを、`UsageLimitWaiting` の文を含むかで調べる。入力欄より上のブロックは読まない。`ClassifyResumeScreen` の footer の切り出しも同じ `footerBlocks` に替えた(`resume_screens.go:358`)ので、2 つの判定は同じ footer を読む。
- footer に限る理由: resume の waiting の判定は待ちの文を、入力欄のすぐ上のブロックと候補のブロック(返事の最後の段落、ツールの出力)でも数える。返事が文を引用しただけで waiting と読むので、そこを上限扱いにすると stall の知らせが止まり、件を resolve しても画面が変わらないあいだ stall に戻らない(codex の計画助言 MEDIUM、計画 D2)。footer は CLI が描く欄で、返事は出ない。
- 返す行は、待ちの文のある footer のブロックを `usageLimitLine` で 1 行にしたもの(D4)。
- `:98-103` の「入力欄が見つからない」の枝は、`ClassifyResumeScreen` が先に入力欄を見つけたあとにしか通らない。通れば false で、stall が残る向き。

## 2. 文の表

`internal/org/resume_screens.go`。

- `UsageLimitWaiting` の欄(`:184-192`)。claude は `Continuing automatically at` と `Continuing shortly`(`:246`)で、どちらも同じ表の `Waiting` の文(`:236-237`。出典は docs の wait-for-a-usage-limit-to-reset)。`Retrying in` と `Waiting for API response` は API の再試行の待ちなので印を付けず、stall のまま。codex は空。
- `You've hit your 5-hour limit` を `Stopped`(`:215`)と `UsageLimit`(`:222`)に足した。出典は `seen: Claude Code 2.1.316`。この文が効くのは、`⎿` の行が候補のブロックの頭に来て待ちの行が出ない画面だけ。実機の画面そのものを決めるのは 1 の footer の道(D5)。

## 3. 人と leader に届く文

- `internal/org/watch_usage_limit.go`: 件の本文の理由 `usageLimitedWhy`(`:47`)と leader への ALERT(`:308-310`)は「hit a usage limit (stopped, or waiting for it to reset)」。`recurred` の note は「hit a usage limit: 」で始まり(`:297`)、stderr の 2 行(`:280`、`:285`)は「seats … hit a usage limit」。待つ座席も `SEATS` に並ぶので、「stopped」のままでは偽になる(self-review L2、N1)。
- `internal/cli/org_watch.go:102-110`: `ralph org watch --help` の `usage_limited` の段落。
- 確かめること: 件の本文、`recurred` の note、`:285` の stderr の行は `watch_usage_limit_test.go` が文字列で固めている。`:280` の行(recur が失敗したとき)と `--help` の段落を固めるテストはない(test 報告 Test gaps 6、7)。

## 4. テストと実機の fixture

- `internal/org/testdata/resume/claude-5h-limit-continuing.txt`(23 行): 実機(Claude Code 2.1.316、herdr 0.7.5、幅 53 桁)の画面をバイト単位で写した。上限の文は 8 行目の `⎿`、11〜12 行目の小文字の `continuing automatically at`、19〜21 行目の footer に出る。
- `usage_limit_screen_test.go`: fixture の表(`:32`)、`TestUsageLimitScreen_FiveHourLimitUnderThePromptIsAStop`(`:371`)、`TestUsageLimitScreen_WaitingForALimitToReset`(`:394`。footer の `Continuing shortly` が肯定、footer の `Retrying in` と `Waiting for API response`・返事の引用・ツールの出力・待ちが終わったあとの小文字の行が否定)、`TestUsageLimitWaitingSentences_AreWaitingSentences`(`:501`)。`EveryFixture` の「上限の画面は必ず stopped」の断定は、stopped か footer の待ちか、に直した(`usageLimitFooterLine`、`:159`)。
- `watch_usage_limit_test.go:700`: 実機の画面の座席が、stall の判定が立つ 2 周期目に usage_limited に入り、stall が出ない。`:729`: 返事が待ちの文を引用した座席は、4 周期まで stall のまま。
- `resume_screens_test.go:99` と `send_screen_test.go:97`: 新しい fixture の行。resume は waiting、send は busy。#322(`fix/send-input-box-marker`)も同じ表に行を足すので、後から merge する側が両方の行を残す(計画 Risks)。
- test 報告の変異 7 個(a、a2、b1、b2、c、d、d2)はすべて死んだ。a2(footer に入力欄より上のブロックを足す)は否定の側のテストだけが落とす。プロンプトが待ちの文を引用する画面の専用の行は無い(Test gaps 3)。

## 5. 文書と記録

- `docs/specs/2026-08-01-org-runtime.md:87-92`: 2026-11-13 改訂の (a)〜(c)。`docs/specs/2026-11-07-org-seat-resume.md:58,60`: 止まった文の一覧に `5-hour limit`、待ち中の文の行に usage_limited の注。
- `.claude/rules/ralph/agent-messaging.md` と `templates/base/` の写し(同じ内容): `usage_limited` の段落。`AGENTS.md:91`: `internal/org/` の行に 1 句。
- `/director` skill の 2 文(`.claude/skills/director/SKILL.md:992-994`、`:1038-1042`。写しは `.agents/skills/` と `templates/base/` の 3 つで、4 つとも同じ)。計画の Non-goal「director の skill の `usage_limited` の扱いを変えること」からのずれ。解除を待つ座席も件に入る(D1)ので、「`usage_limit` は座席が上限で止まったこと」と「`note=` に出た座席も上限で止まった」が偽になった(verify V1)。
- `docs/tech-debt/README.md`: `:452` に 5 時間の上限を実機で見たことを足した。新しい行は `:495`(待ちの文が入力欄の上にだけ出る版)、`:496`(`herdr agent get` の出力)、`:497`(test の穴)。
- `docs/evidence/claude-5h-limit-screen-2026-11-13.md`: 実機の画面、直す前の判定(`ok=false`、waiting)、この branch の判定、確かめていないこと。

## 見どころ

- footer だけか、どのブロックでも数えるか。`footerBlocks` を 2 つの判定が共有するので、footer の切り出しを変えると resume の判定にも効く。
- 作った fixture `claude-continuing-automatically.txt`(待ちの文が入力欄の上のブロックにだけある)は、上限の画面のまま `{}`(`usage_limit_screen_test.go:61`)。返事の引用と区別できないためで、2.1.316 の実機は footer に出す。入力欄の上にだけ出す版の claude は stall のままになる(tech-debt `:495`)。
- 未確認: 待つあいだ `herdr agent get` の出力が変わらないか。見張りは stall の判定が立つ座席の画面しか読まない(`watch.go:1325`)ので、出力が変わり続けるなら待ちの道は実機で動かず、#329 の前と同じ黙り方になる。テストの fake herdr は出力が一定で、これを確かめられない(tech-debt `:496`、test 報告 Test gaps 1)。
- Known gap L4: 返す行は footer のブロックの全部なので、状態の行(`🤖 Haiku 5.5 | ⚡ low | 🌿 main`、権限の行)も、件の本文、leader への ALERT、stderr に 300 文字まで入る(`usageLimitLineMaxRunes`、`usage_limit_screen.go:33`)。`usageLimitLine` が制御文字を落として長さを切る。計画 D4 で受け入れた。状態の行がユーザーの設定した `statusLine` の出力かは未確認(self-review L4)。
