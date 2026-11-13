# Test report: watch-usage-limited

- Date: 2026-11-13(run-test.sh は 03:04:43 から 03:30:57 JST。変異、probe、coverage は同じ時間帯に回した)
- Plan: `docs/plans/active/2026-11-13-watch-usage-limited.md`(`## Test plan`、AC1〜AC11。issue #312 の受け入れ条件 1)
- Tester: tester サブエージェント(`/test`、pipeline の cycle 1。`cycle-count.json` は `{"cycle": 1}`)
- Scope: `git diff origin/main...HEAD`(origin/main `4220ea7c`、HEAD `c191529c`、24 ファイル、+2605/-106)。run-test.sh は changed scope を求め、そのまま changed で回った(`==> Language scope: changed (changed_languages)`、選ばれた pack は golang だけ)。worktree の production code とテストには手を入れていない。変異と probe は `go test -overlay` で当て、worktree のファイルは書き換えていない
- Evidence: `docs/evidence/test-2026-11-13-watch-usage-limited.log`(`.gitignore` の `docs/evidence/*.log` に当たるのでコミットしない)。run-test.sh の全出力のあとに、30 秒ごとの load と空きディスクの見本、変異の harness と定義と結果、対照の回、probe のテストと結果、coverage の回と変更行の集計を足した。wrapper 自身のログは `docs/evidence/verify-2026-11-12-180444.log`
- 起動の仕方: run-test.sh は Bash の 600 秒の上限を超えるので、`perl` の fork と `setsid` で切り離し(INT・HUP・QUIT・TERM は DEFAULT、標準入出力は `/dev/null`)、`caffeinate -w` で終わりを待った
- 中断: run-test.sh が rc 0 で終わったあと(03:30:57)、レポートを書く前に API の利用上限で止まり、同じ日の午前に再開した。再開のとき HEAD は `c191529c` のままで、`git status --porcelain` は run の開始時と同じ未追跡の `docs/insights/events/2026-11-12-watch-usage-limited.jsonl` 1 行だけだった。テストする中身が変わっていないので、run-test.sh は回し直していない
- 打ち直し: なし。空きディスクは 85 GiB から 82 GiB で、`no space left` は run-test.sh の出力にも変異の出力にも 0 行だった。build の失敗も無かった

## Test execution

| Suite / Command | Tests | Passed | Failed | Skipped | Duration |
| --- | --- | --- | --- | --- | --- |
| `./scripts/run-test.sh`(rc 0、`==> All verifiers passed.`) | shell 45 本 + Go 10 パッケージ | shell 45 本 OK(PASS を含む行 5,195)、Go 10 パッケージ ok | 0(`--- FAIL` の行 0) | 2 行 | 1,574 秒 |
| └ Go: `internal/org` | パッケージ全体 | ok | 0 | — | 823.0 秒 |
| └ Go: `internal/cli` | パッケージ全体 | ok | 0 | — | 728.9 秒 |
| └ Go: `internal/config`、`internal/gitroot`、`internal/insights`、`internal/org/driver`、`internal/scaffold`、`internal/upgrade`、`internal/org/protocol`、`cmd/ralph` | パッケージ全体 | ok。`(cached)` は 0 | 0 | — | 19.2、4.4、4.1、3.9、2.1、3.0、1.4、0.7 秒 |
| 変異の対照: overlay なしで org を `-run 'UsageLimit' -v` | 新しいテスト 17 本(subtest 込みの RUN 142) | 142 | 0 | 0 | 1 秒 |
| 変異: production code の 5 個(下の節) | 5 | 5 個が死んだ | — | — | 1 個 7〜9 秒 |
| probe: overlay で足したテスト 1 本(Test gaps 1) | 1(subtest 2) | 1 | 0 | 0 | 数秒 |
| coverage: org `-run 'UsageLimit\|Watch' -coverprofile` | 名前で絞った回 | ok | 0 | — | 56 秒 |

Skipped の 2 行は、`tests/test-plan-visual.sh` の browser detection(`/Applications` にブラウザがあると起動してしまうため)と、`tests/test-secret-scan-branch.sh` の git 2.41 より古い本物の git の case(手元は git 2.49.0)である。どちらも手元の環境によるもので、この diff と関係がない。

run-test.sh の Go の段(終わりの 03:30:57 から org の 823 秒を引くと 03:17 ごろから)のあいだ、1 分の load は 5.1 から 13.3 だった。ほかの session の `go test` が同時に回っていた可能性はあるが、落ちたテストは無い。org は 823 秒で、30 分の上限の 46% である。

## 受け入れ条件とテスト

どのテストも run-test.sh で PASS した。「変異」の列は、そのテストが落とした変異の id である。

| AC | テスト(`internal/org/`) | 変異 |
| --- | --- | --- |
| AC1 stall の代わりに `usage_limited`、本文に座席・driver・上限の文 | `TestWatchUsageLimit_SeatsStoppedOnAUsageLimitAreOneItemOfTheOrg`(codex の leader と claude の 2 座席。本文の 3 行に `resets 3:45pm`、`resets 11pm (Asia/Tokyo)`、`try again at 3:05 PM` が入る) | 当てていない |
| AC2 同じ周期の複数座席は 1 件 | 同じテスト(3 座席で件は 1 件、`SEATS: implementer, leader, reviewer`。3 周期目は件も ALERT も増えない) | 当てていない |
| AC3 deadman を待たずに同じ周期で件になり、届け方は見張りの ALERT の件と同じ | 同じテスト(director off で人に送る)、`TestWatchUsageLimit_TheItemIsDeliveredLikeTheWatchsOtherAlertItems` の 4 行(switch off、on で仕事を受ける見張り、on で受けない見張り、受信箱なし。`DeadmanMinutes = 60`) | M4 |
| AC4 条件のあいだに座席が増えたら、open なら recurred、acked か resolve 済みか、append のあいだに resolve されたら新しい件 | `TestWatchUsageLimit_ASeatThatStopsLaterIsARecurrenceOfAnOpenItemAndANewItemOtherwise` の 4 行 | M2、M3(どちらも「e1 acked」の行)、M1(前提の確かめで落ちた) |
| AC10 理由は `usage_limit`、本文は leader 自身の ALERT の文にならない | `assertUsageLimitedBody`(AC1 のテストが使う。`about the leader itself` と `leader_is_anomaly_subject` が無いこと)、`TestWatchUsageLimit_APendingAlertWithSeatsTheWatchDoesNotWriteRecordsNoItem` の 3 行 | M4 |
| AC5 上限の座席がなくなれば条件は下り、再び立てば resolve の有無で recurrence か新しい件 | `TestWatchUsageLimit_TheConditionClearsAndALaterLimitFollowsTheEpisodeRule` の 3 行(open、acked は recurrence、resolved は新しい件。状態ファイルに `usage_limited_seats` が残らない) | 当てていない |
| AC6 読めない画面、通信断、判断できない画面は stall | `TestWatchUsageLimit_ASeatWhoseScreenIsNoUsageLimitStallsAsBefore` の 7 行(読みの失敗、制御文字を含む長いエラー、通信断、知らない画面、codex 座席に claude の上限、driver なし、pane なし)、`_AnOrgWithNoHerdrReadsNoScreenAndReportsNoError` | M5(「a network error」の行) |
| AC7 stall の判定が立たない座席は読まない、読みは driver の呼び出しの上限で区切る | `TestWatchUsageLimit_TheScreenOfASeatThatIsNotStalledIsNotRead`、`_AScreenReadThatDoesNotAnswerIsCutOffAndTheSeatStalls`(50 ms で切れ、stall の ALERT に `context deadline exceeded`)。AC1 のテストも、1 周期目は読まないこと、読みが 60 行で期限つきであることを確かめる | 当てていない |
| AC11 上限の leader は除外の条件があっても入る、上限でない leader は今までどおり除外 | `TestWatchUsageLimit_ALeaderStoppedOnAUsageLimitIsReportedWhileItWouldBeWaiting` の 6 行(ほかの座席が動いている、open の QUESTION、acked の BLOCKED × leader が上限か否か) | M1(6 行すべて) |
| AC8 上限の文の見分けが既存の fixture で期待どおり | `TestUsageLimitScreen_EveryFixture`(testdata/resume の 63 ファイルすべて。上限は 3 つだけ)、`_Edges`、`TestUsageLimitSentences_AreStoppedSentences`、`TestUsageLimitScreen_EachStoppedSentence`、`TestUsageLimitLine` | M5 |
| AC9 skill、agent-messaging.md、仕様、AGENTS.md | 文面はテストで固定していない。写しの一致は run-test.sh の shell の段と Go の skill 系のテストが通った。文の中身は /verify が読んで確かめた | — |

Test plan の項目との突き合わせ:

- Unit(上限の文の見分け、fixture すべて): AC8 の行のとおり、上限 3、通信断、入力待ち、動作中、選択の画面のどれも表にある
- Unit(見張り): Test plan に並んだ場合はすべてテストがある。2 座席で 1 件、3 座席目で recurred が 1 つ(note に 3 座席目)、ack 後と resolve 後は新しい件、append のあいだの resolve で新しい件、理由と本文、上限の leader と除外、上限でない leader の除外、条件が下りる、再び立つ、読めない・通信断で stall、健康な座席は読まない
- Unit(本文の上限): `TestWatchUsageLimit_TheItemBodyStaysWithinTheCap`(6 座席と 20 座席、長い文と制御文字。20 座席では後ろが `+N more` の 1 行になる)
- Integration(`RunWatchAll` の 1 周期): `TestWatchUsageLimit_AStatusFileWrittenBeforeUsageLimitedLoadsAndWorks`(古い形の状態ファイルから 1 周期回し、件が 1 件、書いた状態ファイルに座席が入る)
- Regression: 既存の見張りのテスト(stall、liveness、scope_change、recurrence、deadman、alert_notify)は org のパッケージ全体の回で通った
- Edge cases: leader だけが上限(AC11 の上限の行、`SEATS: leader`)、claude と codex の混在(AC1 のテスト)、画面の行の制御文字(`_Edges` の制御文字と不正な UTF-8 の行、`TestUsageLimitLine`、本文の上限のテスト)はテストがある。leader 以外だけが上限の場合は、コミットされたテストが無い(Test gaps 1)

## 変異

orchestrator の指示(ディスクのため変異は 5 個まで)に合わせ、指示の優先順に 1 つずつ当てた。各変異は文字列の置き換え 1 つで、置き換える文字列がファイルに 1 回だけ現れることを確かめてから当てた(harness は evidence にある)。どれも org を `-run 'UsageLimit'` で絞って回し、同じ絞り方の対照は overlay なしで通った。どの変異も build は通り、落ちたのはテストの assert である。

| 変異 | guard | 置き換え | 結果 | 落としたテスト |
| --- | --- | --- | --- | --- |
| M1 | 画面の読みを leader の待ちの除外より前に置く順(D4、AC11) | `watch.go` の `evaluateSeat` で、`if s.SeatID == LeaderIdentity { stalled = stalled && !w.leaderIsWaiting(...) } else if seatBusy(...) {...}` を `if stalled { ... readUsageLimitScreen ... }` の前に移す | 死んだ | `_ALeaderStoppedOnAUsageLimitIsReportedWhileItWouldBeWaiting` の 6 行すべて(`0 reads of the leader's screen ... want one read and no stall`)、`_ASeatThatStopsLater...` の 4 行(reviewer が動いているので leader が読まれず、e1 が leader を含まない) |
| M2 | 増えた座席を recurred にするのは件が open のときだけ(AC4) | `watch_usage_limit.go` の `if !ok \|\| (grown && item.State != InboxStateOpen) {` を `if !ok {` に | 死んだ | `_ASeatThatStopsLater.../e1_acked`(e2 ができず、e1 に recurred が付く) |
| M3 | `escalateToInbox` が `usage_limited` を開いた件に畳まない(AC4) | `watch.go` の `ok && !isUsageLimitedAlert(alertID) && w.escalateAsRecurrence(` を `ok && w.escalateAsRecurrence(` に | 死んだ | `_ASeatThatStopsLater.../e1_acked`(新しい ALERT が acked の e1 の recurrence になり、e2 ができない) |
| M4 | 理由を Subject より先に条件で決める(D3、AC10) | `watchEscalationReason` の `case isUsageLimitedAlert(alertID):` と `case subject == LeaderIdentity:` の順を入れ替える | 死んだ | `_TheItemIsDelivered.../no_inbox_store`(escalations.jsonl の reason と banner が `leader_is_anomaly_subject`)、`.../switch_on,_a_watch_that_accepts_work`(stderr の head)、`_APendingAlertWithSeats...` の 3 行 |
| M5 | `UsageLimitScreen` が上限の印を求める(D5) | `usage_limit_screen.go` の `if !found \|\| !slices.Contains(table.UsageLimit, s.Text) {` を `if !found \|\| (false && !slices.Contains(...)) {` に | 死んだ | `TestUsageLimitScreen_EveryFixture/claude-enotfound.txt`、`_Edges` の通信断の 3 行、`_EachStoppedSentence`、`_ASeatWhoseScreenIsNoUsageLimitStallsAsBefore/a_network_error` |

変異から分かったこと:

- M2 と M3 を落としたのは、どちらも「e1 acked」の 1 行だけである。resolve 済みの 2 行は `openAlertItemForKey` が件を見つけないので、この 2 つの guard には届かない。AC5 の acked の行は条件が下りたあとの立ち上がり(`grown` が false)なので、これも届かない。1 行で固まっていることは確かめたが、その行を消すと 2 つの guard はどちらも守られなくなる
- M4 で、件の本文の `REASON` は落とす側に回らなかった。`usageLimitedEscalationBody` が `usageLimitReason` を直に書き、`escalateAlert` から渡る理由を使わないためである。渡る理由が効くのは stderr の head と、受信箱なしの道の escalations.jsonl だけで、director off の 2 行(escalations.jsonl の reason は受信箱の送り先の理由になる)はこの変異を見分けない。落としたのは上の 5 行である
- M1 は AC11 のテストのほかに AC4 のテストも落とした。AC4 の前提(reviewer が動いている周期に leader と implementer が上限)が、そのまま「ほかの座席が動いている」除外の場面になっているためである

## Coverage

- Statement(`-run 'UsageLimit|Watch'` で絞った org の回): パッケージ全体では 36.6%(絞った回なので全体の数字としては意味がない)。変更行で数えると、`internal/org` の production code に足した行のうち文に当たる 244 行のうち 230 行(94.3%)を通った
- 関数: 新しい関数と変えた関数は、次の 5 つを除いて 100%。`recordUsageLimitRecurrence` 76.2%、`withUsageLimitedLines` 75.0%、`cutUsageLimitText` 83.3%、`matchedStopSentence` 80.0%、`escalateToInbox` 93.8%(残りは変更の外の既存の分岐)、`sendAlert` 90.9%(同じく既存の分岐)
- Branch: Go の標準の道具では測っていない
- 通らなかった 14 行:
  - `watch_usage_limit.go:242-245`: `recordUsageLimitRecurrence` の、まだ escalate されていない pending の alert に座席を足す分岐(Test gaps 2)
  - `watch_usage_limit.go:251-252`: 受信箱が読めないときに false を返す分岐
  - `watch_usage_limit.go:261-263`: `AppendRecurred` が `ErrInboxResolved` 以外で失敗したときの stderr の 1 行と false
  - `watch_usage_limit.go:320-321`: `withUsageLimitedLines` で、head だけで余地が無いときに head だけを返す分岐
  - `watch_usage_limit.go:405-406`: `cutUsageLimitText` の n が 1 未満で "" を返す分岐
  - `usage_limit_screen.go:70`: `matchedStopSentence` が文を見つけられないときの戻り。`ClassifyResumeScreen` が stopped と読んだ画面では、どれかの stopped の文が先頭に合うので、おそらく届かない。未確認です
- Notes: 集計は `git diff -U0 origin/main...HEAD` の足した行と cover profile の block を突き合わせた(evidence に手順と結果)。絞った回なので、パッケージ全体の回ならほかのテストが通る行がある可能性はある。ただし上の 14 行はどれも `usage_limited` の専用の分岐で、既存のテストが届く形ではない

## Failure analysis

| Test | Error | Root cause | Proposed fix |
| --- | --- | --- | --- |
| なし | — | — | — |

## Regression checks

| Previously broken behavior | Status | Evidence |
| --- | --- | --- |
| issue #312: 利用上限で止まった座席に stall の ALERT が上がり続け、上限だと分かるまで約 8 時間かかった | 直っている。上限の画面の座席は stall にならず、同じ周期に `usage_limited` の件 1 件になる | AC1〜AC3 のテスト、M1・M5 |
| #307a の recurrence(件が resolve されていなければ recurred、されていれば新しい件) | 変わらない。`usage_limited` の立ち下がりのあとの再びの立ち上がりもこの規則に乗る | `_TheConditionClears...` の 3 行、既存の `TestWatch_Recurrence_*` が org の全体の回で通った |
| leader の待ちの除外(#307b、`leaderIsWaiting`) | 上限でない leader では変わらない | AC11 の上限でない 3 行、既存の `TestWatch_LeaderStall_*` が通った |
| 見張りの既存の条件(stall、liveness、scope_change、deadman、alert_notify) | 変わらない | org のパッケージ全体(823 秒)が通った |

## Test gaps

1. leader 以外だけが上限の場合に、コミットされたテストが無い(/verify の指摘と同じ)。overlay で足した probe(`TestProbeUsageLimit_OnlyANonLeaderSeatIsLimited`、コミットしていない)で 2 つの場合を確かめ、どちらも通った。
   - leader が動いている(manifest に leader の出来事が毎周期ある)とき: 件は `usage_limited` の 1 件だけで、`SEATS: implementer`、本文に `implementer (claude): ⏺ You've hit your session limit · resets 3:45pm`、`REASON: usage_limit`。stall の ALERT は 0 で、leader の画面は読まれない
   - leader が上限でもなく動いてもいない(入力待ちの画面のまま)とき: 件は 2 件になる。e1 は leader の stall(`REASON: leader_is_anomaly_subject`)、e2 は `usage_limited`(`SEATS: implementer`)。leader の画面は stall が立つ周期ごとに読まれる(3 周期で 2 回)。leader の待ちの除外は、ほかの座席が動いているか leader の上げた件が未解決のときだけで、上限で止まった座席を待つ leader はどちらにも当たらないため、変更の前と同じく leader の stall が上がる。計画の AC11 は除外の規則を変えないと書いているので、テストの失敗ではない。上限の座席を待つ leader を除外に含めるかどうかは、製品の判断として残る
   - 足すなら最小のテストは、probe の「leader が動いている」行である(evidence に全文がある)
2. `recordUsageLimitRecurrence` の pending の分岐(`watch_usage_limit.go:242-245`)にテストが無い。見張りは状態を周期をまたいでメモリに持つ(`watchAllRun.statuses`)ので、上限を立てた周期で manifest の読み直し(`rr2`)が失敗すると、`evaluateCycle` は deadman の前に戻り、送った ALERT の pending が残る。次の周期に座席が増えると、この分岐で pending の座席が差し替わる。めったに無いが届く道である
3. `recordUsageLimitRecurrence` の、受信箱が読めないときと `AppendRecurred` がほかの理由で失敗したときの分岐(251-252、261-263)にテストが無い。どちらも新しい ALERT に倒れる作りで、stderr の文は固定されていない
4. `withUsageLimitedLines` の head だけを返す分岐と `cutUsageLimitText` の n < 1 の分岐にテストが無い。座席の id は識別子で長さに上限があるので、head が 2,000 字を超える場面はおそらく無い。未確認です
5. M2 と M3 の guard は「e1 acked」の 1 行だけで固まっている(変異の節)。acked の件に、条件が立っているあいだに座席が 2 回続けて増える場合など、2 行目があるとよい
6. 件の本文の `REASON` は `usageLimitReason` を直に書くので、理由を条件で決める `watchEscalationReason` の順は、本文ではなく stderr の head と受信箱なしの道でしか見えない(M4 の節)。今のテストはその 2 か所で固めている
7. self-review の Known gaps のまま残る 2 点: 画面が読めなかった理由は leader への stall の ALERT(agmsg)にだけ出て、受信箱の件の本文には届かない(N1)。画面が読めなかった周期のあとに上限が読めた座席は「増えた」と数えられ、件が open なら recurred、acked なら新しい件になる(L5)。どちらも計画どおりの挙動で、テストの失敗ではない
8. 上限の画面は fixture だけで確かめている。claude の weekly limit は実機の画面(2026-11-07 の evidence)だが、session limit と codex の 2 つの文は文書と binary の文字列からで、実機の画面では確かめていない
9. AC9 の文面(director と org の skill、agent-messaging.md、仕様、AGENTS.md、`ralph org watch --help`)はテストで固定していない。/verify が読んで確かめた

## Verdict

- Pass: run-test.sh(changed scope、shell 45 本、Go 10 パッケージ、1,574 秒、rc 0)、変異の対照、probe、coverage の回のすべてが通った。指示の優先順に当てた変異 5 個(M1〜M5)はすべて死んだ。Test plan の項目は、「leader 以外だけが上限」の edge case を除いてテストがあり、その場合も probe で期待どおりの件が 1 件できることを確かめた。残る穴は Test gaps の 9 項目で、どれも通るべき挙動の失敗ではない
- Fail: 0
- Blocked: なし
