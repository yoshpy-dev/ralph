# Self-review report: org-result-missing

## Run 2 (cycle 1、Run 1 の直し)

- Date: 2026-11-22
- Plan: `docs/plans/active/2026-11-22-org-result-missing.md`(issue #361)
- Reviewer: reviewer サブエージェント(`/self-review`)。Run 1 の指摘を直したコミットだけを見る周で、`cycle-count.json` は cycle 1 のまま。Run 1 は下に、見出しを 1 段下げて残す。今の判定はこの節の Recommendation の行
- Scope: `git diff 98392dd5..f57cb92d`(コミット `f57cb92d` の 1 つ、10 ファイル、+789 / -118)。M1 の直し(`raiseResultMissing`、`recordResultMissingRecurrence`、座席の記録の `ResultMissingTask` と `ResultMissingAlert`、`escalateToInbox` の `!isResultMissingAlert(alertID)`)、L1 の ALERT の本文、L2 と L3 の doc comment、L4 の `ResultOutOfReach`、L5 の空行の読み飛ばしと雛形の 1 行目の決まり、rule の段落(2 写し)、台帳 563 行目の (b)
- 走らせたもの: worktree で `go test -count=1 ./internal/org/ -run 'ResultMissing|Watch|Escalat|Recur|SeatSend'` を 1 回(`ok`、37.7 s)。`f57cb92d` を `git archive` で scratchpad に写し、調べのテストを 2 本足して走らせた(下の「M1 の確認」と R2-L2 の根拠)。worktree には何も書いていない。全体のテスト、静的解析、`run-test.sh` は走らせていない

Run 1 の ID と取り違えないよう、この run の ID には `R2-` を付ける。CRITICAL、HIGH、MEDIUM はなく、LOW が 4 件。

### Evidence reviewed

- `f57cb92d` の diff の全部と、変わった関数の今の全文: `watch_result_missing.go` の `resultMissing`、`resultMissingAlertMessage`、`raiseResultMissing`、`recordResultMissingRecurrence`、`seatLinesToLeader`、`historyReachesBack`、`resultMissingLineOf`、定数と型の doc comment
- `watch.go`: `watchSeatSnapshot`、`evaluateSeat` とその doc comment、`raiseOrClear`、`sendAlert`、`checkDeadman`、`leaderActedSinceAlert`、`escalateAlert`、`escalateToInbox`、`watchAlertID`、`alertIDConditionType`、`escalatedTimestamp`、`leaderHistoryFromField`
- `watch_recurrence.go` の全文(`pendingAlertForKey`、`openAlertItemForKey`、`reconcilePendingAlertsWithInbox`、`recordRecurrence`、`foldIntoPendingAlert`、`escalateAsRecurrence`)と、`openAlertItemForKey` と `pendingAlertForKey` の呼び手(grep。result_missing の鍵で呼ぶのは `escalateToInbox` の 1 か所だけで、その結果は除外で捨てられる)
- `protocol.Parse`(1 行目が `TYPE:` でなければ失敗、ヘッダは最初の空行か `KEY: value` でない行まで)
- 雛形: `implementer.md` と `reviewer.md` の差分と RESULT の例(どれも `TASK_ID:` が 2 行目)、`leader.md` の `result_missing` の項
- 手元の agmsg 1.5.1 の `scripts/history.sh`(表示は `echo "  $status [$ts] $from → $to: $body"`、行を飛ばすのは `[ -n "$ts$from$to$body" ] || continue`。読んだだけ)
- テスト: 足された M1 の 5 本、L1 の 1 本、L4 と L5 の行、雛形と rule の断片
- rule の 2 写しの `cmp`(同じ)、台帳 563 行目の差分、467 行目と 88 行目の trigger

### Run 1 の 6 件の行き先

| Run 1 | 状態 | 根拠 |
| --- | --- | --- |
| M1 | 直った | 下の「M1 の確認」 |
| L1 | 直った | 本文は「has sent the leader no RESULT or REVIEW for TASK %s since that TASK was sent at ...」と言い、QUESTION か BLOCKED のあとは、その行と、そのあと座席に送られた最初のメッセージの時刻を書く。座席が BLOCKED を送っていないとは言わない(`TestWatch_ResultMissing_TheAlertSaysWhereTheClockStarted`) |
| L2 | 直った | `leaderHistoryFromField` と `parseAgmsgHistoryLine` の doc comment が互いを名指し、parser を分けたままにする理由を書いた。ファイルは移していない(勧めの「少なくとも」の側) |
| L3 | 直った | `historyEscapes` の doc がバックスラッシュを戻さないことを、`historyReachesBack` の doc が history.sh の飛ばす行を書いた。手元の history.sh の行と合う |
| L4 | 直った。doc comment の 1 か所が追いついていない(R2-L1) | `ResultOutOfReach`。調べ: TASK のあとに座席の行が 1000 ある座席を 7 周回すと、履歴の読みは 1 周目の 3 回(50、200、1000)だけ |
| L5 | 直った。直しの説明は TYPE について言えることより広い(R2-L4) | 見張りは本文の先頭の空行を飛ばし、雛形はファイルの 1 行目を `TYPE:` にさせ、手順 2 は `leader: ` の直後を確かめる |

### M1 の確認(頼まれた観点)

- 別の TASK に入らないこと: 件や保留中の ALERT に畳む道は 3 つある。(1) `raiseResultMissing` が `recordResultMissingRecurrence` を呼ぶのは `snap.ResultMissingTask` が今の TASK の鍵と同じときだけで、畳む先は `snap.ResultMissingAlert` の alert_id の保留中の ALERT か、`AlertID` がそれと同じ件だけ。2 つの欄は新しい ALERT を送るときに一緒に書かれるので、食い違わない。(2) `escalateToInbox` の鍵での畳み込みは `!isResultMissingAlert` で外れた。(3) `alertItemWithRecurredID` と `reconcilePendingAlertsWithInbox` は alert_id の一致でしか拾わず、`recordResultMissingRecurrence` が `recurred` に書く alert_id は保留中の ALERT の id ではない。`recordRecurrence` と `foldIntoPendingAlert` は result_missing からは呼ばれなくなった。新しい TASK が前の TASK の件や保留中の ALERT に入る道は見つからなかった
- 同じ TASK は今も畳まれる: 保留中なら `Recurrences` が増え、件が open か acked なら `recurred` になり、件が resolve 済みか leader が動いたあとなら新しい ALERT になる。`TestWatch_ResultMissing_TheSameTaskRisingAgainIsARecurrenceOfItsOwnAlert` が 3 つとも固定している
- `checkDeadman`: leader が動いて消された保留中の ALERT を `snap.ResultMissingAlert` が指していても、`recordResultMissingRecurrence` は保留中の ALERT も件も見つけずに false を返し、新しい ALERT が出る。黙る道にはならない
- `reconcilePendingAlertsWithInbox`: 受信箱にすでにある保留中の ALERT は、座席の評価の前に escalate されて消える。同じ TASK の次の立ち上がりは件の側(`AlertID` の一致)で畳まれる
- 見張りの再起動(replay): 件を書いたあと状態ファイルを保存する前に止まった見張りは、次の起動で保留中の ALERT を `escalateToInbox` の最初の分岐(同じ alert_id の件)で拾うので、2 つ目の件はできない。除外がかかるのは鍵での畳み込みの分岐だけ。`pruneEscalated` は監査の記録を刈るだけで、この流れに関わらない
- 古い状態ファイル(新しい欄がない): `result_missing` はこの branch で入る条件なので、前の release の見張りが書いたファイルには `result_missing` の記録も件もない。欄が消えるのは、新しい binary の書いたファイルを古い binary が読み書きしたあと(下げてから上げ直す)と、ファイルを失ったとき。そのときの動きは R2-L2
- 状態ファイルを失ったとき: 調べのテスト(`m1Fixture` で t1 の件 e2 を open にしたあと `f.status` を空にし、座席を入力待ちのまま t0+40m まで回す)で、ALERT は 2 回、件は e2 と e3 の 2 つ(どちらも open、recurrences 0)になった。同じ TASK の件が 2 つできる。黙るより騒がしい側に倒れるので受け入れられるが、rule は何も書いていない(R2-L2)
- L4 の記憶は新しい TASK で外れる: `ResultOutOfReach` は TASK の鍵(`sent` の時刻と TASK_ID)と比べるので、次の TASK では一致しない。欄は消されずに次の上書きを待つが、古い値はどの判定にも効かない。stall への効き方は rule に書かれた(「neither known to be done nor out of the history's reach」)が、`evaluateSeat` の doc comment には書かれていない(R2-L1)
- 雛形の手順 2 と history.sh: history.sh は `  <●|○> [<at>] <from> → <to>: <body>` を出し、本文の改行を `\n` と書くので、`leader: ` の直後は本文の 1 行目になる。本文が `TYPE:` で始まれば「`leader: ` の直後が `TYPE: <TYPE>`」に合う。先頭が空行のファイルを送ると、座席の確かめは通らず(`SEND FAILED:` を書いて止まる)、見張りは RESULT を数える。座席の側が厳しい向きなので、見張りが届いた RESULT について `result_missing` を上げる道にはならない。leader には RESULT が届いていて、座席の画面に誤った `SEND FAILED:` が残るだけ

### Findings (Run 2)

| ID | Severity | Area | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- | --- |
| R2-L1 | LOW | maintainability | L4 の直しで、履歴が遡れなかった TASK は `resultMissing` の頭で `resultMissingNo` になる。座席がそのあと動いてまた入力待ちになっても、時計のあいだ stall を抑えない。`evaluateSeat` の doc comment は「resultMissingClockRuns: a last TASK with no STOP after it and not known to be done, ...」と「once the clock has run out and result_missing does not hold (...), its stall is decided as before」のままで、遡れなかった TASK(終わったとは分かっていない)の stall も抑えるように読める。`resultMissingNo` の doc も、遡れないときを時計が切れたあとの値としてだけ書いている。rule と `resultMissingClockRuns` の doc は直っている | `internal/org/watch.go:1300-1308`、`internal/org/watch_result_missing.go:36-41` と `:194`、`.claude/rules/ralph/agent-messaging.md` の「neither known to be done nor out of the history's reach」 | `evaluateSeat` の括弧に「and not out of the history's reach (ResultOutOfReach)」を足し、「the history cannot tell」の後ろに、遡れなかった TASK はあとの周でも stall を抑えないことを 1 文足す。`resultMissingNo` の doc にも同じ 1 文 |
| R2-L2 | LOW | maintainability | 同じ TASK に新しい ALERT と 2 つ目の件ができる道は、状態ファイルを失ったときのほかにもある。`escalateToInbox` は result_missing の ALERT を鍵の open の件に畳まなくなったので、同じ TASK の 2 つ目の保留中の ALERT は必ず別の件になる。2 つ目の保留中の ALERT は `recordResultMissingRecurrence` が false を返すとできる: 座席の記録がない(ファイルを失った、古い binary が欄を落とした)、保留中の ALERT があるのに台帳が読めない、受信箱が読めない、`AppendRecurred` が resolve 以外の理由で失敗した。ほかの条件では、これらの失敗のあとの escalate は鍵の件に畳まれる(watch-alert-dedupe の D2)。result_missing では件が増える。`escalateToInbox` の doc の括弧は「after the watch lost its status file」だけを挙げる。rule の「A rise for the same `TASK` follows the rules above, with that `ALERT` and its own item only.」は例外を書かず、台帳にも行がない | 調べ: 上の「状態ファイルを失ったとき」(ALERT 2 回、件 e2 と e3)。`internal/org/watch_result_missing.go:340-383`(false を返す道)、`internal/org/watch.go:2052-2058` と `:2096`、`.claude/rules/ralph/agent-messaging.md:306` | 少なくとも、rule に 1 文(見張りが座席の記録を持たないときと、読み書きに失敗したときは、同じ TASK でも新しい ALERT と件になる)を足し、`escalateToInbox` の括弧に失敗の道を足す。安い守りもある。alert_id に入った時刻(`escalatedTimestamp`)が今の TASK の `sent` の時刻以後なら、その ALERT は今の TASK について上がったもの(そのときの座席の最後の TASK は今も最後の TASK)なので、`raiseResultMissing` は `snap` に一致がないとき、鍵の保留中の ALERT か open の件のうち TASK の時刻以後に上がったものを、同じ TASK の episode として使える。これで状態ファイルを失ったときの重複は消える。escalate のときの失敗の道まで消すには、保留中の ALERT に TASK の鍵を持たせる必要があり、台帳 563 行目の (a) と同じところに触る |
| R2-L3 | LOW | maintainability | `recordResultMissingRecurrence` は、立ち上がりの `recurred` を書く 3 つ目の関数になった(`recordRecurrence`、`recordUsageLimitRecurrence` と並ぶ)。`recordRecurrence` の受信箱の分岐(`watchAlertID(key, now)` を `recurred` の alert_id にする、`beforeAppendRecurred`、`ErrInboxResolved` の扱い、stderr の文)と、`foldIntoPendingAlert` の leader の確かめを写している。新しい関数の doc は `recordRecurrence` を名指すが、逆向きの指しがない。`recordRecurrence` の doc と `watch_recurrence.go` の冒頭は「raiseOrClear and escalateToInbox (watch.go) call these」のまま。台帳 467 行目の (7) は、この `recurred` の alert_id を、再起動した見張りがもう一度作れる値に変える予定で、その行のファイルの一覧に `watch_result_missing.go` はない。(7) を払う変更は `recordRecurrence` だけを直し、result_missing の写しを前の形のまま残しうる | `internal/org/watch_result_missing.go:321-383`(`:374` の `watchAlertID(conditionKey(orgID, seatID, condResultMissing), now)`)、`internal/org/watch_recurrence.go:13-17` と `:152-199`、`docs/tech-debt/README.md:467`(trigger の (7)) | `recordRecurrence` の doc に「result_missing の立ち上がりは recordResultMissingRecurrence(watch_result_missing.go)が同じ形で書く」と 1 文足し、ファイル冒頭の呼び手に `raiseResultMissing` を足す。467 行目の (7) に `watch_result_missing.go` を足すのは /sync-docs に渡す(下の一覧) |
| R2-L4 | LOW | readability | L5 の直しの説明が、TYPE について言えることを送信全体に広げている。`resultMissingLineOf` の doc は「so a send a seat counts as made is one the watch counts too」と言い、`TestRenderRolePrompt_SeatSend_ChecksTheHistoryAfterExitZero` の doc は「a line that passes this check also passes the watch's」と言う。座席の確かめは TYPE を行の頭で見るが、TASK_ID は今も「その行に `TASK_ID: <TASK_ID>` がある」(行のどこでもよい)で、見張りは `protocol.Parse` のヘッダの範囲でしか TASK_ID を読まない。TASK_ID を空行のあとの本文に置いたファイルや、`TYPE: RESULT`、`SUMMARY: C:\new`、`TASK_ID: t1` の順のファイル(history.sh はバックスラッシュをそのまま出し、`historyEscapes` が `\n` を改行に戻すので、見張りには `ew` の行でヘッダが終わる。この直しで足した `historyEscapes` の doc のとおり)は、座席の確かめを通り、見張りには数えられない。雛形の例はどれも `TASK_ID:` を 2 行目に置くので、起きるのは例から外れた順のときだけ | `internal/org/watch_result_missing.go:520-527`、`internal/org/prompts_seat_send_test.go:156-157`、`internal/org/prompts/implementer.md:147-149`、`internal/org/prompts/reviewer.md:456-458` | 2 つの doc を「TYPE を読む位置は座席と見張りでそろう」に狭める。そろえ切るなら、雛形に「2 行目を `TASK_ID:` の行にする」と書き、手順 2 も「`TYPE: <TYPE>` の次が `TASK_ID: <TASK_ID>`」を確かめる形にして、`prompts_seat_send_test.go` の断片を合わせる |

### Positive notes (Run 2)

- M1 の直しは、同じ TASK の畳み込みを `snap` の alert_id 1 つに絞り、鍵での検索(`pendingAlertForKey`、`openAlertItemForKey`)を result_missing から外した。別の TASK に入る道を、条件の分岐ではなく参照の仕方で閉じている
- 失敗の道はどれも新しい ALERT の側に倒れる(台帳が読めない、受信箱が読めない、append の失敗)。台帳が読めないときと append の失敗は stderr に 1 行が出る
- 新しい 3 つの欄は `omitempty` で、欄のないファイルは「記録なし」として読まれる。誤って黙る側には倒れない
- `resultMissingFinding` で TASK を判定と一緒に返すので、`raiseResultMissing` は ALERT の本文と同じ TASK を記録する。TASK を 2 回読んで食い違う余地がない
- `seatLinesToLeader` の `outOfReach` は、最後の limit でも遡れないときだけ true で、読みの失敗と parse の失敗は覚えない。`TestWatch_ResultMissing_AHistoryReadThatFailsIsTriedAgain` が固定している
- M1 のテストは、前の TASK の件が open のまま、条件が active のまま(見張りが周を飛ばしたとき)、同じ TASK の 3 つの状態、別の TASK の件に入らないこと、watcher の起動回数を、それぞれ別の関数で固定している
- 台帳 563 行目の (b) は、`raiseResultMissing` が trigger を満たして払わなかったことを、行の慣例どおり日付つきで書いた

### 範囲外で /sync-docs に渡すもの

implementer が挙げた 4 件を確かめた。どれも今の文が `f57cb92d` の動きと合わない。

- 台帳 467 行目: trigger の「The next change to the `scope_change`, `stall`, or recurrence code of `internal/org/watch.go`, ...」を `escalateToInbox` の除外が満たした。行の慣例の「hit this trigger ... but did not pay it」の注がない。(7) のファイルの一覧に `watch_result_missing.go` を足す(R2-L3)
- 台帳 88 行目: この branch の持ち越しの注は `watch.go` を 2,300 行、`watchSeatSnapshot` の欄を `IdleSince` と `ResultSeen` とし、「43b7777e, the last commit of the branch that changed it」と書く。今は `f57cb92d` で 2,322 行、欄は 3 つ増えた
- `README.md:284` の「an ALERT that fires again while the same seat's and condition's item is not resolved is a `recurred` count on that item, not a new ALERT or item」は、別の TASK の result_missing では成り立たない
- `leader.md` の `result_missing` の項(「その TASK の RESULT を送っていない」)は REVIEW を言わず、ALERT の本文(「no RESULT or REVIEW」)と言い方が違う

足すもの 2 件。

- 台帳 563 行目の (a) の Impact は「finds the TASK on the seat's screen or in the ledger」と、座席ごとに件が 1 つの書き方をしている。M1 の直しのあと、1 つの座席に TASK_ID のない open の件がいくつも並びうる。画面に出るのは最後の TASK だけなので、前の件の TASK は `ALERT_TS` の前に座席へ送られた最後の TASK として台帳で探すことになる。そのことを Impact に足す
- 仕様 `docs/specs/2026-08-01-org-runtime.md` の (c)(「入力待ちの座席に終わったと分からない TASK があるあいだ」)と、`/org` skill の stall の項は、遡れなかった TASK を抑えの外に置いていない(R2-L1 の文書の側)

### Tech debt identified (Run 2)

| Debt item | Impact | Why deferred | Trigger to pay down | Related plan/report |
| --- | --- | --- | --- | --- |
| 追加なし。R2-L1 から R2-L4 は、直すか残すかの判断を実装の側に返す。R2-L2 を残すなら、台帳 563 行目に (c)(同じ TASK の 2 つ目の件ができる道と、上の安い守り)を足す。R2-L3 を残すなら、467 行目の (7) に `watch_result_missing.go` を足す | | | | この report |

_(行は足していない。追記は不要。)_

### Recommendation (Run 2)

- Merge: yes(CRITICAL 0、HIGH 0、MEDIUM 0、LOW 4)。Run 1 の M1 は直っている。別の TASK は前の TASK の件にも保留中の ALERT にも入らず、同じ TASK の再発は今もその ALERT の episode に畳まれる。残る 4 件は doc comment と rule の文、指しの 1 文の直しで、R2-L2 だけは安い守りを足す選択がある
- Follow-ups:
  - R2-L1、R2-L3、R2-L4 は doc comment の直し(R2-L4 の後半の案を取るなら雛形とテストの断片も)。判定の論理は変わらない
  - R2-L2 は、rule に 1 文足すだけなら文書の直し。安い守りを足すなら `raiseResultMissing` に 10 行ほどと、`m1Fixture` で状態ファイルを空にするテスト 1 本(上の調べの形でよい)
  - Go か雛形に触れるので、どれも記録だけの修正には当たらない。直したら `/self-review` から回し直す
  - 上の「範囲外で /sync-docs に渡すもの」の 6 件は `/sync-docs` で直す

---

## Run 1 (cycle 1、履歴)

この下は Run 1(HEAD `22d2f0cf`、report のコミット `98392dd5`)の report を、見出しの階層を 1 つ下げた以外は変えずに残したもの。

- Date: 2026-11-22
- Plan: `docs/plans/active/2026-11-22-org-result-missing.md`(issue #361)
- Reviewer: reviewer サブエージェント(`/self-review`)。cycle 1
- Scope: `git diff origin/main...HEAD`(origin/main = 2a069b50、HEAD = 22d2f0cf、main の取り込みは b0c56556)。30 ファイル、+2732 / -98。中心は `internal/org/watch_result_missing.go`(新規)、`internal/org/watch.go` の `evaluateSeat` と `watchSeatSnapshot`、`internal/org/reseat_progress.go` の `sentDetailsText`、座席の雛形 3 つ、rule、skill、仕様、テスト
- 走らせたもの: HEAD を `git archive` で scratchpad に写し、そこに調べ用のテストを 1 本足して `go test -count=1 ./internal/org/ -run TestProbe_ResultMissing_SecondTaskWhileFirstItemOpen -v` を 2 回(下の M1 の根拠)。worktree には何も書いていない。全体のテスト、静的解析、`run-test.sh` は走らせていない

### Evidence reviewed

- 計画の全文(Progress の S0、S1、計画の直し、S2、S2b、S3、main の取り込み)
- `watch_result_missing.go` の全文と、`watch.go` の `evaluateSeat`、`raiseOrClear`、`sendAlert`、`checkDeadman`、`leaderActedSinceAlert`、`escalateAlert`、`leaderHistoryFromField`、`historyLeaderLineCount`
- `watch_recurrence.go` の `recordRecurrence`、`foldIntoPendingAlert`、`reconcilePendingAlertsWithInbox`
- `verbs.go` の `Send` の終わり(`sent` の TS は Enter と確かめのあと、確かめは既定 3 秒まで)、`callWithTimeout`(既定 10 秒)
- `driver/agmsg.go` の `History`(`AGMSG_SELF_NAME=off` が history.sh にも付く)と `driver.go` の Runner(stderr は出力に混ざらない)
- 手元の agmsg の `scripts/history.sh` と `send.sh`(repo の外。読んだだけ)
- `protocol.Parse`(1 行目が `TYPE:` でなければ失敗)
- テスト: `watch_result_missing_test.go` の全関数、`prompts_seat_send_test.go`、`prompts_leader_result_missing_test.go` の構成、`prompts_verification_test.go` と `prompts_xreview_reviewer_test.go` の差分
- 雛形: `implementer.md:115-156`、`reviewer.md:424-465` と冒頭の書く場所の段落、`leader.md:1664-1689`
- rule と skill: `agent-messaging.md` の 2 つの新しい段落、`/director` と `/org` の skill の差分、`docs/tech-debt/README.md` の追記

### Findings

| ID | Severity | Area | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- | --- |
| M1 | MEDIUM | maintainability | 同じ座席の前の TASK の `result_missing` の受信箱の件が open か acked のあいだに、次の TASK の `result_missing` が立つと、その件の `recurred` になる。note は空で、leader には ALERT が届かない。新しい TASK_ID はどこにも残らない。条件の鍵は (org, 座席, 条件) だけで TASK を含まず、立ち上がりで Active になるので、件が resolve されたあとも、座席が入力待ちのままなら新しい TASK の ALERT は出ない。leader は届かない RESULT を待ち続けるので、#361 と同じ形が残る。rule の「Its deadman escalation and its re-firings follow the rules above」は既存の決まりに従うとだけ言い、別の TASK の立ち上がりが前の件に吸われることは書いていない。台帳の (a) の行は件に TASK_ID がないことだけを扱う | `internal/org/watch.go:1387`(`raiseOrClear(..., condResultMissing, ..., true, "")`、recurNote が空)、`internal/org/watch_recurrence.go:172-197`(`openAlertItemForKey` は鍵で引く)と `:267`(escalation の側も note は空)、`.claude/rules/ralph/agent-messaging.md:301`。調べ: t1 を上げて deadman で件 e2 にしたあと、t2 を送り、座席を 1 周 working にしてから入力待ちにした。t0+30m で ALERT は t1 の 1 件のまま、e2 に `recurrences=1`、stderr は `condition=result_missing fired again; recorded as a recurrence on inbox item e2`。e2 を resolve してさらに 30 周回しても ALERT は 1 件、条件は active のまま | 少なくとも、`evaluateSeat` が `raiseOrClear` に渡す recurNote に新しい TASK_ID を入れる(たとえば `TASK_ID <id>`)。件の `recurred` に TASK が残り、director が別の TASK だと分かる。できれば、条件を最後に上げた TASK(`resultMissingTaskKey`)を座席の記録に持ち、TASK が変わった立ち上がりは前の件の再発にせず新しい ALERT として leader に送る。台帳の (a)(件に TASK_ID を持たせる)と同じところを触るので、まとめて直すとよい。直さずに進めるなら、rule と `/director` の skill にこの穴を 1 文書き、台帳に行を足す |
| L1 | LOW | readability | ALERT の本文は「without sending the leader a RESULT or BLOCKED for it」と言う。しかし座席がその TASK の BLOCKED を送り、leader が答え、そのあと座席が入力待ちのまま `stall_minutes` を過ぎた道でも条件は立つ。この道では座席は BLOCKED を送っているので、本文は事実と違う。QUESTION の道でも、待ちは TASK からではなく leader の答えから数えているのに、本文は「received TASK %s and has been waiting」としか言わない。leader の雛形(「その TASK の RESULT を送っていない」)と rule(RESULT か REVIEW)はそれぞれ別の言い方をしている | `internal/org/watch_result_missing.go:181-192`(BLOCKED と QUESTION のあと `start` を leader の答えに移す)、`:198-203`(本文)、`internal/org/prompts/leader.md:1664-1667` | 本文を「has sent the leader no RESULT for it since the TASK (or since the leader's last answer to its QUESTION or BLOCKED)」のように、数え始めを含む形にする。`assertResultMissingAlert` の `head` も合わせて直す |
| L2 | LOW | maintainability | history.sh の 1 行を読む関数が 2 つになった。新しい `parseAgmsgHistoryLine` は最初の `[` と次の `] ` で切り、既存の `leaderHistoryFromField` は最初の `] ` で切る。どちらも doc comment で history.sh の形を別々に説明していて、互いを名指していない。`agmsgHistoryRecord`、`parseAgmsgHistory`、`historyEscapes`、`agmsgNoHistory` は agmsg の履歴一般の読み手なのに、条件のファイルに置かれている。history.sh の形が変わったとき、片方だけ直る余地がある | `internal/org/watch_result_missing.go:61-67`、`:336-397`、`internal/org/watch.go:1770-1804` | `leaderHistoryFromField` を `parseAgmsgHistoryLine` の `From` で書き直すか、少なくとも 2 つの doc comment で互いを名指す。履歴の読み手を `agmsg_history.go` のような小さなファイルに移すと、grep で見つけやすい |
| L3 | LOW | null-safety | 履歴の読み方の doc comment が、history.sh の保証より強いことを言う。(a) history.sh は本文の改行とタブを `\n` と `\t` にするが、バックスラッシュはそのまま出す(`replace(replace(body, char(10), '\n'), char(9), '\t')`)。`historyEscapes` は本文に書かれた文字どおりの `\n` も改行に戻すので、`SUMMARY:` のようなヘッダの値に `\n` という文字列があると、そこで行が割れ、それより後のヘッダは本文になる。1 行目の TYPE と 2 行目の TASK_ID には効かない。(b) history.sh は、どれかの欄が NULL の行(SQLite の `\|\|` が全体を NULL にする)を `[ -n "$ts$from$to$body" ] \|\| continue` で飛ばす。そういう記録があると、出力の行数が limit より少ないのに古い行が残っていて、`historyReachesBack` の「limit より少なければ全部読めた」が成り立たない。agmsg がそういう記録を書くかは確かめていない | `internal/org/watch_result_missing.go:65-67`(「which are undone」)、`:310-324`、`:359-369`。手元の agmsg の `scripts/history.sh` の rows の SQL と表示のループ | (a) と (b) を doc comment に書く(元に戻す処理は完全ではない、limit より少ないことは history.sh が飛ばす行がない前提)。(b) を気にするなら、limit より少ない場合も、最も古い行が `since` より前か、行が 1 つもないときだけ遡れたとする案がある。ただし短い履歴で「わからない」が増えるので、doc に書くだけでもよい |
| L4 | LOW | maintainability | 時計が切れたあとは、座席が入力待ちでいるかぎり、毎周(既定 30 秒)履歴を読み直す。答え待ち、条件が立っている、わからない、のどれでも読む。遡れない「わからない」は 50、200、1000 の 3 回を毎周読み、各回に 10 秒の期限がある。TASK のあとの行は増えるだけなので、1000 で遡れなかった TASK は、そのあとも遡れない。計画の S3 の注は「`result_missing` は座席ごとに履歴を最大 3 回読む(1 回 10 秒まで)」を範囲の外の発見として挙げているが、台帳にも rule にもない | `internal/org/watch_result_missing.go:163-169`、`:279-308`、`internal/org/watch_result_missing_test.go:604-633`(1 周だけを見ていて、次の周の読み直しは固定していない)、計画の Progress の S3 | 1000 でも遡れなかった TASK を `resultMissingTaskKey` で座席の記録に覚え、その TASK では読まない(`ResultSeen` と同じ形)。直さないなら台帳に行を足す(毎周の読みの数、期限、どれで止まるか) |
| L5 | LOW | maintainability | 座席の確かめと見張りの判定が、同じ契約を違う緩さで読む。雛形の手順 2 は「`{{SEAT_ID}} → leader:` の最後の行に `TYPE: <TYPE>` と `TASK_ID: <TASK_ID>` がある」ことを確かめる(行のどこかにあればよい)。見張りは `protocol.Parse` で読み、1 行目が `TYPE:` でなければ型のない行として数える。先頭に空行、BOM、コードの囲みがあるファイルを送ると、座席の確かめは通り、見張りは RESULT を数えず、`stall_minutes` 後に届いている RESULT について `result_missing` を上げる。`$(cat <ファイル>)` は先頭の空行を残す | `internal/org/prompts/implementer.md:141-151`、`internal/org/prompts/reviewer.md:450-460`、`internal/org/watch_result_missing.go:326-334`、`internal/org/protocol/protocol.go:100-106` | 手順 2 を「その行の `leader: ` の直後が `TYPE: <TYPE>` で始まり、`TASK_ID: <TASK_ID>` がある」と書き、ファイルの 1 行目を `TYPE:` の行にすることも雛形で言う。`prompts_seat_send_test.go` で固定する |

### Positive notes

- 台帳から読む TASK_ID は、切れたもの、空のもの、制御文字を含むものを外している(`seatLastTask`)。ALERT のヘッダに改行を差し込む道はない。ALERT は `sendAlert` の `ValidateText` も通る
- 履歴の行は、最初の `[`、次の `] `、次の ` → `、次の `:` で切る。どの区切りも本文より前にあるので、本文に `] ` や ` → ` があっても from と to はずれない(`TestResultMissing_ParseAgmsgHistoryLine` の 6 つ目の行が固定している)
- 読めない行、読めない履歴、遡れない履歴、切れた TASK_ID は、どれも上げない側に倒れている。誤った ALERT で leader を止めない計画の方針と合う
- 見張りが座席を名指して history.sh を打っても、`driver.Agmsg.run` が `AGMSG_SELF_NAME=off` を付けるので、agmsg の自分の名付けは走らない
- 時刻は台帳も agmsg も秒で比べている。同じ秒の RESULT は数え、leader の答えは秒で後のものだけを数える。どちらもテストで固定されている
- 見張りの状態ファイルの新しい 2 つの欄は `omitempty` で、古いファイルは `IdleSince` なしで読み込まれ、その周に今の時刻が入る。数え始めが遅れるだけで、誤って上げる側には倒れない
- テストは偽の時計で回し、sleep がない。偽の history.sh は件数の上限、時刻順、`\n` と `\t` の書き方、出力の trim まで本物に合わせてある
- 座席の送り直しは、終了コードが 0 でないときの 1 回だけで、履歴の確かめに失敗したときは送り直さない。送信と確かめが輪になる道はない。leader の頼みのあとも、2 回目の `SEND FAILED:` で BLOCKED に移る

### Tech debt identified

| Debt item | Impact | Why deferred | Trigger to pay down | Related plan/report |
| --- | --- | --- | --- | --- |
| 追加なし。M1 と L1 から L5 は、直すか残すかの判断を実装の側に返す。残すものがあれば、M1 と L4 は台帳に行が要る(M1 は台帳の「Known gaps of fix/org-result-missing」の行の (c) として足すのがよい)。L2、L3、L5 は doc comment と雛形の文の直しで済む | | | | この report、`docs/plans/active/2026-11-22-org-result-missing.md`(Progress の S3) |

_(行は足していない。追記は不要。)_

### Recommendation

- Merge: yes(CRITICAL 0、HIGH 0、MEDIUM 1、LOW 5)。M1 は、届かない RESULT を見張りが黙って見逃す道を 1 つ残す。条件は「前の TASK の件が open か acked」に限られるので止める理由にはしないが、直すことを勧める。直さずに進めるなら、PR の本文に known gap として書き、台帳に行を足す
- Follow-ups:
  - M1 の最小の直しは `evaluateSeat` の recurNote 1 か所と、それを固定するテスト 1 本(上の調べのテストの形でよい)。TASK が変わったら新しい ALERT にする直しは、台帳の (a) と同じ `watchPendingAlert` と件の本文に触れる
  - L1 と L5 は雛形、本文、テストの文の直しで、判定の論理は変わらない
  - L4 の「遡れない TASK を覚える」は `ResultSeen` と同じ形の欄を 1 つ足す直し
  - どれも Go か雛形に触れるので、記録だけの修正には当たらない。直したら `/self-review` から回し直す
