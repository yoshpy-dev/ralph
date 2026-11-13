# Walkthrough: 利用上限で止まった座席を usage_limited として org ごとに 1 件で知らせる(watch-usage-limited)

- 計画: `docs/plans/active/2026-11-13-watch-usage-limited.md`(PR の作成のあとに archive へ移す)
- issue: #312 の 1 本目(#312a、受け入れ条件 1)。PR は作成前
- 差分: `git diff bb416c48...HEAD`(この walkthrough を除いて 37 ファイル、+3,608 / −153。Go の本体は約 +770 / −56、Go のテストは約 +1,540、残りは skill・文書・計画・報告)
- 経過: pipeline は上限の 2 回走った。cross-review は cycle 1 が ACTION_REQUIRED 1 件(2 節の直し)と WORTH_CONSIDERING 1 件、cycle 2 が 0 件

## 1. 画面を読む場所と順序(読む価値が一番高い)

`internal/org/watch.go` の `evaluateSeat`(:1271)と `evaluateCycle`(:1044)、`internal/org/watch_usage_limit.go` の `readUsageLimitScreen`(:101)。

- stall の判定(herdr の `agent get` の出力と台帳が `stall_minutes` のあいだ変わらない。:1322)が立った座席だけ、pane の最後の 60 行(`resumeReadLines`)を 1 回読む(:1324-1331)。読みは driver の呼び出し 1 回の期限で区切られる。上限の画面なら `cycle.usageLimited` に足して stall を下ろす。
- leader の待ちの除外(`leaderIsWaiting`)はその後ろ(:1332-1333)に置いた。除外を先に当てると、ほかの座席が動いている周期や leader の上げた件が未解決の周期に、上限で止まった leader の画面が読まれない(plan advisory 2、AC11)。
- 読めなかった座席(pane id なし、読みの失敗、期限切れ)と、上限の文でない画面の座席は今までどおり stall(AC6)。読めなかったときは、理由を `usageLimitUnreadNote`(:131)が stall の ALERT の最後に 1 行足す。herdr の無い org は読まない。
- `evaluateCycle` は leader を最後に評価し(:1085-1100)、全座席のあとに `raiseUsageLimited`(:1101)を呼ぶ。`checkDeadman`(:1107)はその後ろなので、件は同じ周期のうちにできる(4 節)。

## 2. org ごとに 1 件にする規則

`internal/org/watch_usage_limit.go` の `raiseUsageLimited`(:181)と `recordUsageLimitRecurrence`(:244)、`internal/org/inbox.go`、`internal/org/watch_recurrence.go`。

- 条件の鍵は (org、leader、`usage_limited`)。watch-status の `UsageLimitedSeats`(watch.go:332)が今の上限の座席の集合を持ち、古い状態ファイルは空として読む。座席が 0 になれば条件を下ろして集合を空にし(AC5)、条件が立っていて新しい座席が無ければ何もしない。
- 集合に新しい座席が入ると立ち上がりとして扱う。まだ escalate されていない pending の ALERT があれば、それに座席を足す(:246-250)。無ければ、resolve されていない件に `recurred` を足す。ただし座席が増えたとき(`grown`)に recurred にするのは件が open のときだけで、acked か resolve 済みなら新しい件と 2 通目の ALERT になる(plan advisory 1、AC4)。条件が下りてから再び立った場合は #307a の規則のまま。
- cross-review cycle 1 の直し(34e5bd86): 読んだ時点の open の確かめ(:259)だけでは、読んだあと append の前に ack された件に recurred が付き、増えた座席が既読の件に埋まる。`AppendRecurredIfOpen`(inbox.go:412)が inbox のロックの中で acked を `ErrInboxAcked`(:103、:434)で断り、:268-274 がそれを新しい件に回す。`beforeAppendRecurred`(watch_recurrence.go:150)の seam が、その窓にテストで ack を差し込む。

## 3. 上限の画面の見分け

`internal/org/usage_limit_screen.go`(96 行)、`internal/org/resume_screens.go`(:183、:205-210、:236)。

- `UsageLimitScreen`(:42)は、`ClassifyResumeScreen` が stopped と読んだ画面のうち、決め手の stopped の文(`matchedStopSentence`、:63)が driver の表の `UsageLimit` にあるものだけを true にする。通信断(`API Error:` など)、Enter 待ち、再試行中、知らない画面、返答や prompt に引用された上限の文は false。
- 印は resume の表の側に持つ(D5)。claude は 4 文、codex は 2 文。印の文がすべて stopped の表にあることは `TestUsageLimitSentences_AreStoppedSentences` が固定している。
- 返す行は、文を含む塊を 1 行にしたもの(`usageLimitLine`、:81)。折り返しと、続く `resets 11pm (Asia/Tokyo)` や `try again at 3:05 PM.` を含み、制御文字を落とし、300 rune で切る。

## 4. 件になるところ

`internal/org/watch.go` の `checkDeadman`(:1797)から `escalateWithoutInbox`(:2170)、`internal/org/watch_usage_limit.go` の本文、`internal/cli/org_watch.go`。

- `usage_limited` の ALERT は subject が leader なので、`checkDeadman` の「subject が leader」の分岐(:1812-1824)に乗り、deadman を待たずに `escalateAlert` に進む(D3)。届け方(`inboxDelivery`: director か人)は変えていない。
- 理由は条件を先に見て決める(`watchEscalationReason`、:1941)。subject が leader でも `usage_limit` で、leader 自身の ALERT の文にならない(AC10)。`escalateToInbox` は `usage_limited` を `escalateAsRecurrence` に回さない(:2020)。recurrence は `raiseUsageLimited` が決めるため。
- 本文は `usageLimitedEscalationBody`(watch_usage_limit.go:312)。`SEAT: leader`、`REASON: usage_limit`、`SEATS:`、座席ごとの `<座席> (<driver>): <上限の文>` の行が並ぶ。2,000 字に収めるため、`fitUsageLimitedSeats`(:345)が行を切り、入らない座席を `+N more` にする。
- 状態ファイルの座席と driver は `checkUsageLimitedSeats`(:457)が識別子として確かめる(`checkWatchEscalationValues`、watch.go:2060 から)。inbox に載せられない道は banner に `seats=` を足す(:2171)。`escalate.go:578-586` はコメントだけ。`org_watch.go` は `--help` の文(:72、:102-110)。

## 5. テスト(`internal/org/`)

- `usage_limit_screen_test.go`(402 行): `TestUsageLimitScreen_EveryFixture`(:142)が testdata/resume の 63 ファイルすべてを見る(上限は 3 つだけ)。`_Edges`(:188)に通信断と制御文字の行。
- `watch_usage_limit_test.go`(936 行): 偽の herdr と pane の画面で 1 周期ずつ回す。`_ASeatThatStopsLaterIsARecurrenceOfAnOpenItemAndANewItemOtherwise`(:451)が AC4 の 4 行の表、`_ALeaderStoppedOnAUsageLimitIsReportedWhileItWouldBeWaiting`(:751)が除外の 6 行。
- `watch_usage_limit_race_test.go`(63 行)と `inbox_test.go`(+138 行): 競合のテストは、watch が e1 を open と読んだあと append の前に ack する。e1 は acked のまま recurred が無く、3 座席の e2 と 2 通目の ALERT ができ、stderr に recurrence の行が出ない。`TestInboxAppendRecurredIfOpen_*` の 3 本がロックの中の確かめを固定する。
- 変異(test report): cycle 1 の M1〜M5 は死んだ。cycle 2 の HEAD では M2 だけ生き残る(結果の変わらない変異、N3)。M6(`AppendRecurredIfOpen` を `AppendRecurred` に戻す)を落とすのは競合のテスト 1 本だけ。run-test.sh は cycle 2 で rc 0。
- コミットしたテストが無いのは「leader 以外だけが上限」の場合(Test gaps 1。overlay の probe では期待どおり)。

## 6. 文書と記録

- skill: director と org の各 4 つの写し。director は `SEAT` ではなく `SEATS:` の座席の画面を読み、リセットまで待つか立て直すかを人と決める(`.claude/skills/director/SKILL.md:1028-1047`)。org は見張りの条件の一覧に足した(`.claude/skills/org/SKILL.md:647-654`)。
- `.claude/rules/ralph/agent-messaging.md:140-155`(`templates/base` の写しも)、`docs/specs/2026-08-01-org-runtime.md:79-86` の 2026-11-13 改訂 (a)〜(d)、`docs/specs/2026-10-07-org-multi-org-director.md:119`、AGENTS.md:91、README.md:133 と :283、`docs/quality/quality-gates.md:83`(`templates/base` の写しも)。
- `docs/tech-debt/README.md`: 新しい行が 7(L5、N1、テストの穴、2 つのテストファイルの大きさ、recurred の note、N3 と N4)。既存の 11 行には、この branch が Trigger に当たったという 2026-11-13 の持ち越しの note を足した(1 行は、`usage_limited` も resume の表を読むという一文)。

## 見どころ

- 順序(1 節): `evaluateSeat` の :1324-1333 の並びが逆だと、上限で止まった leader が除外に隠れる。AC11 の 6 行と M1 が固定している。上限でも動いてもいない leader は除外に当たらず、stall の件と `usage_limited` の件の 2 件になる(Test gaps 1)。計画は除外の規則を変えないと書いているが、この挙動でよいかは判断が要る。
- acked と open(2 節): 増えた座席を acked の件の recurred にしない規則(plan advisory 1)と、open の確かめをロックの中に移した直しを見る。:259 の読んだ時点の確かめは速い経路で、消しても結果は変わらない(N3)。
- 上限の文の根拠(3 節): 実機で見たのは claude の weekly limit だけで、session limit と codex の 2 文は文書と binary の文字列から取った(Test gaps 8)。上限でない画面を上限と読むと stall を隠すので、表の文の裏取りは見る価値がある。
- PR 本文の Known gaps に書く 4 点。L5: 集合にいた座席の読みが 1 回失敗すると stall になり、次に読めた周期で「新しい座席」と数えられる。N1: 読めなかった理由は leader への agmsg にだけ出て、件の本文に届かない。recurred の note は 500 rune(`InboxNoteMaxRunes`)で、`max_seats` の既定 5 を大きく超える org では `+N more` の id が切れる(id は stderr と `usage_limited_seats` に残る)。N3 と N4: N4 は tech-debt の行と競合テストのコメントが triage の番号を指すこと(中身は文に書いてある)。
