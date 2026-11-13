# Test report: watch-usage-limited

- Cycle: 2(HEAD `9f95c51a`)。cycle 1 は HEAD `c191529c` で回した。cycle 2 で書き換えた箇所と足した箇所には「(cycle 2)」と付けた。印のない所見は cycle 1 のもので、cycle 2 の HEAD でもそのまま成り立つ
- Date: 2026-11-13。cycle 2 は、絞り込みの回、変異、coverage を 12:20 から 12:25 JST に、run-test.sh を 12:40:59 から 13:13:07 JST に回した。cycle 1 は run-test.sh を 03:04:43 から 03:30:57 JST に回し、変異、probe、coverage も同じ時間帯に回した
- Plan: `docs/plans/active/2026-11-13-watch-usage-limited.md`(`## Test plan`、AC1〜AC11。issue #312 の受け入れ条件 1)
- Tester: tester サブエージェント(`/test`、pipeline の cycle 2。`cycle-count.json` は `{"cycle": 2}`)
- Scope(cycle 2): cycle 1 の `/test` のあとに入った変更は次の 4 つと、報告のコミットである。worktree の production code とテストには手を入れていない
  - `34e5bd86`: `recordUsageLimitRecurrence`(`internal/org/watch_usage_limit.go`)が、座席の増えた条件を新しい `AppendRecurredIfOpen`(`internal/org/inbox.go`)で足すようにした。inbox のロックの中で件が open かを確かめ、acked の件は `ErrInboxAcked` で断る。テストは `internal/org/inbox_test.go` の 3 本と、新しい `internal/org/watch_usage_limit_race_test.go` の 1 本
  - `193b044a`: `internal/org/watch_recurrence.go` のコメントだけ
  - `86f7c1f8`: sync-docs(skill、README、quality-gates、仕様)
  - `9f95c51a`: origin/main `bb416c48` の取り込み(#317 の `internal/cli` と `internal/upgrade`。`internal/org` は無い)
- Scope(diff): `git diff origin/main...HEAD`(merge-base は `bb416c48`)は 37 ファイル、+3367/-152。run-test.sh は changed scope を求め、そのまま changed で回った(`==> Language scope: changed (changed_languages)`、選ばれた pack は golang だけ)
- 実行の順(cycle 2): orchestrator の指示で、ほかの session の Go の全体の回と重ならないよう 1 本ずつ回した。絞り込みの回と変異を先に済ませ、run-test.sh は、ほかの 2 つの回が終わった(lane D は 03:21Z、lane C は 03:40Z)という連絡を受けてから、03:40:59Z に始めた
- Evidence: `docs/evidence/test-2026-11-13-watch-usage-limited.log`(`.gitignore` の `docs/evidence/*.log` に当たるのでコミットしない)。cycle 1 の節のあとに cycle 2 の節を足した。中身は、絞り込みの回 2 つの出力、変異の harness と各回の出力、診断の overlay、coverage の回と変更行の集計、run-test.sh の全出力、30 秒ごとの load と空きディスクの見本、`(cached)` の 4 パッケージの回し直しである。wrapper 自身のログは、cycle 2 が `docs/evidence/verify-2026-11-13-034059.log`、cycle 1 が `docs/evidence/verify-2026-11-12-180444.log`
- 起動の仕方: cycle 1 と cycle 2 で同じ。run-test.sh は Bash の 600 秒の上限を超えるので、`perl` の fork と `setsid` で切り離し(INT・HUP・QUIT・TERM は DEFAULT、標準入出力は `/dev/null`)、`caffeinate -w` で終わりを待った
- 中断(cycle 1): run-test.sh が rc 0 で終わったあと(03:30:57)、レポートを書く前に API の利用上限で止まり、同じ日の午前に再開した。再開のとき HEAD は `c191529c` のままで、`git status --porcelain` は run の開始時と同じ未追跡の `docs/insights/events/2026-11-12-watch-usage-limited.jsonl` 1 行だけだった。テストする中身が変わっていないので、run-test.sh は回し直していない。cycle 2 に中断はない
- 打ち直し: cycle 1 も cycle 2 もなし。cycle 2 の空きディスクは 77.9 GiB から 76.1 GiB で、`no space left` は run-test.sh の出力にも変異の出力にも 0 行だった(evidence で 1 行だけ一致するのは、harness の grep の行そのものである)。build の失敗も無かった

## Test execution

### Cycle 2(HEAD `9f95c51a`)

| Suite / Command | Tests | Passed | Failed | Skipped | Duration |
| --- | --- | --- | --- | --- | --- |
| `./scripts/run-test.sh`(rc 0、`==> All verifiers passed.`) | shell 45 本 + Go 10 パッケージ | shell 45 本 OK(PASS を含む行 5,195)、Go 10 パッケージ ok | 0(`--- FAIL` の行 0) | 2 行 | 1,928 秒 |
| └ Go: `internal/org` | パッケージ全体 | ok | 0 | — | 1,179.0 秒 |
| └ Go: `internal/cli` | パッケージ全体 | ok | 0 | — | 1,073.3 秒 |
| └ Go: `internal/config`、`internal/insights`、`internal/upgrade`、`cmd/ralph` | パッケージ全体 | ok | 0 | — | 117.2、2.4、2.1、1.3 秒 |
| └ Go: `internal/gitroot`、`internal/org/driver`、`internal/org/protocol`、`internal/scaffold` | パッケージ全体 | ok `(cached)`。run のあとに `-count=1` で回し直して ok | 0 | — | 回し直しで 2.4、1.6、1.6、1.1 秒 |
| org を `-run 'UsageLimit\|InboxAppendRecurred\|Recur' -count=1 -v` | トップレベル 42 本(subtest 込みの RUN 210) | 210 | 0 | 0 | テスト 1.77 秒(build 込み 7.2 秒) |
| 同じ回の `-race` | 同じ 210 | 210 | 0(`DATA RACE` は 0 行) | 0 | テスト 3.20 秒(build 込み 17.3 秒) |
| 変異: M3、M2、M6(下の節) | 3 | M3 と M6 が死に、M2 は生き残った(結果の変わらない変異) | — | — | 1 回 1〜3 秒(build 込み 6〜10 秒) |
| 診断: M3 に、競合のテストの 38 行目を `Errorf` にした写しを重ねた回 | 1 | — | 期待どおり 38 行目と 45 行目で落ちた | — | 1 秒 |
| coverage: org を `-run 'UsageLimit\|Watch\|InboxAppendRecurred\|Recur' -coverprofile` | 名前で絞った回 | ok | 0 | — | 37.4 秒(build 込み 46 秒) |

新しい 4 本(競合のテスト `TestWatchUsageLimit_ASeatAddedWhileTheItemIsAckedBeforeTheAppendGetsANewItem` と、`TestInboxAppendRecurredIfOpen_RecordsOnlyOnAnOpenItem`、`_SharesAppendRecurredsRefusals`、`_ChecksTheStateUnderTheInboxLock`)は、絞り込みの 2 つの回でも run-test.sh でも通った。

Skipped の 2 行は cycle 1 と同じで、browser detection と、git 2.41 より古い本物の git の case である(下の cycle 1 の段落)。run-test.sh で `(cached)` だった 4 パッケージは、cycle 1 の `/test` のあとに Go の変更があった `internal/org`、`internal/cli`、`internal/upgrade` のどれにも依存しない(`go list -deps -test` で 0 件)。それでも、run のあとにこの 4 つだけを `-count=1` で回し直し、通ることを確かめた。

run-test.sh のあいだ、30 秒ごとに 65 回見た 1 分の load は、shell の段(12:53 まで)で 8.0 から 15.1、Go の段で 7.0 から 27.4 だった。org は 1,179 秒、cli は 1,073 秒で、cycle 1(823 秒と 729 秒)より長い。ほかの session の回と重なったためとおもわれるが、未確認です。golang の pack の上限は 1 パッケージ 30 分(`packs/languages/golang/verify.sh` の `-timeout=30m`。`go env GOFLAGS` は空)で、org はその 66% である。`internal/config` も 117 秒で、cycle 1 の 19 秒より長い。落ちたテストは無い。

### Cycle 1(HEAD `c191529c`)

| Suite / Command | Tests | Passed | Failed | Skipped | Duration |
| --- | --- | --- | --- | --- | --- |
| `./scripts/run-test.sh`(rc 0、`==> All verifiers passed.`) | shell 45 本 + Go 10 パッケージ | shell 45 本 OK(PASS を含む行 5,195)、Go 10 パッケージ ok | 0(`--- FAIL` の行 0) | 2 行 | 1,574 秒 |
| └ Go: `internal/org` | パッケージ全体 | ok | 0 | — | 823.0 秒 |
| └ Go: `internal/cli` | パッケージ全体 | ok | 0 | — | 728.9 秒 |
| └ Go: `internal/config`、`internal/gitroot`、`internal/insights`、`internal/org/driver`、`internal/scaffold`、`internal/upgrade`、`internal/org/protocol`、`cmd/ralph` | パッケージ全体 | ok。`(cached)` は 0 | 0 | — | 19.2、4.4、4.1、3.9、2.1、3.0、1.4、0.7 秒 |
| 変異の対照: overlay なしで org を `-run 'UsageLimit' -v` | 新しいテスト 17 本(subtest 込みの RUN 142) | 142 | 0 | 0 | 1 秒 |
| 変異: production code の 5 個(下の節) | 5 | 5 個が死んだ(M2 は cycle 2 の HEAD では生き残る。変異の節) | — | — | 1 個 7〜9 秒 |
| probe: overlay で足したテスト 1 本(Test gaps 1) | 1(subtest 2) | 1 | 0 | 0 | 数秒 |
| coverage: org `-run 'UsageLimit\|Watch' -coverprofile` | 名前で絞った回 | ok | 0 | — | 56 秒 |

Skipped の 2 行は、`tests/test-plan-visual.sh` の browser detection(`/Applications` にブラウザがあると起動してしまうため)と、`tests/test-secret-scan-branch.sh` の git 2.41 より古い本物の git の case(手元は git 2.49.0)である。どちらも手元の環境によるもので、この diff と関係がない。

run-test.sh の Go の段(終わりの 03:30:57 から org の 823 秒を引くと 03:17 ごろから)のあいだ、1 分の load は 5.1 から 13.3 だった。ほかの session の `go test` が同時に回っていた可能性はあるが、落ちたテストは無い。org は 823 秒で、30 分の上限の 46% である。

## 受け入れ条件とテスト

どのテストも、cycle 1 と cycle 2 の run-test.sh で PASS した。「変異」の列は、そのテストが落とした変異の id である(M1〜M5 は cycle 1、M3 の再実行と M6 は cycle 2)。

| AC | テスト(`internal/org/`) | 変異 |
| --- | --- | --- |
| AC1 stall の代わりに `usage_limited`、本文に座席・driver・上限の文 | `TestWatchUsageLimit_SeatsStoppedOnAUsageLimitAreOneItemOfTheOrg`(codex の leader と claude の 2 座席。本文の 3 行に `resets 3:45pm`、`resets 11pm (Asia/Tokyo)`、`try again at 3:05 PM` が入る) | 当てていない |
| AC2 同じ周期の複数座席は 1 件 | 同じテスト(3 座席で件は 1 件、`SEATS: implementer, leader, reviewer`。3 周期目は件も ALERT も増えない) | 当てていない |
| AC3 deadman を待たずに同じ周期で件になり、届け方は見張りの ALERT の件と同じ | 同じテスト(director off で人に送る)、`TestWatchUsageLimit_TheItemIsDeliveredLikeTheWatchsOtherAlertItems` の 4 行(switch off、on で仕事を受ける見張り、on で受けない見張り、受信箱なし。`DeadmanMinutes = 60`) | M4 |
| AC4 条件のあいだに座席が増えたら、open なら recurred、acked か resolve 済みか、append のあいだに resolve か ack されたら新しい件 | `TestWatchUsageLimit_ASeatThatStopsLaterIsARecurrenceOfAnOpenItemAndANewItemOtherwise` の 4 行。(cycle 2)`TestWatchUsageLimit_ASeatAddedWhileTheItemIsAckedBeforeTheAppendGetsANewItem`(読んだあと append の前に e1 が ack される。e1 は acked のまま recurred が付かず、3 座席の e2 と 2 つ目の ALERT ができ、stderr に recurrence の行が出ない)と、`TestInboxAppendRecurredIfOpen_*` の 3 本(open の件だけに足す、`AppendRecurred` と同じ断り、ロックの中で状態を読む) | M3(競合のテストと「e1 acked」の行)、M6(競合のテストだけ)、M1(前提の確かめで落ちた)。(cycle 2)M2 は直しのあと結果の変わらない変異で、落とすテストはない |
| AC10 理由は `usage_limit`、本文は leader 自身の ALERT の文にならない | `assertUsageLimitedBody`(AC1 のテストが使う。`about the leader itself` と `leader_is_anomaly_subject` が無いこと)、`TestWatchUsageLimit_APendingAlertWithSeatsTheWatchDoesNotWriteRecordsNoItem` の 3 行 | M4 |
| AC5 上限の座席がなくなれば条件は下り、再び立てば resolve の有無で recurrence か新しい件 | `TestWatchUsageLimit_TheConditionClearsAndALaterLimitFollowsTheEpisodeRule` の 3 行(open、acked は recurrence、resolved は新しい件。状態ファイルに `usage_limited_seats` が残らない) | 当てていない |
| AC6 読めない画面、通信断、判断できない画面は stall | `TestWatchUsageLimit_ASeatWhoseScreenIsNoUsageLimitStallsAsBefore` の 7 行(読みの失敗、制御文字を含む長いエラー、通信断、知らない画面、codex 座席に claude の上限、driver なし、pane なし)、`_AnOrgWithNoHerdrReadsNoScreenAndReportsNoError` | M5(「a network error」の行) |
| AC7 stall の判定が立たない座席は読まない、読みは driver の呼び出しの上限で区切る | `TestWatchUsageLimit_TheScreenOfASeatThatIsNotStalledIsNotRead`、`_AScreenReadThatDoesNotAnswerIsCutOffAndTheSeatStalls`(50 ms で切れ、stall の ALERT に `context deadline exceeded`)。AC1 のテストも、1 周期目は読まないこと、読みが 60 行で期限つきであることを確かめる | 当てていない |
| AC11 上限の leader は除外の条件があっても入る、上限でない leader は今までどおり除外 | `TestWatchUsageLimit_ALeaderStoppedOnAUsageLimitIsReportedWhileItWouldBeWaiting` の 6 行(ほかの座席が動いている、open の QUESTION、acked の BLOCKED × leader が上限か否か) | M1(6 行すべて) |
| AC8 上限の文の見分けが既存の fixture で期待どおり | `TestUsageLimitScreen_EveryFixture`(testdata/resume の 63 ファイルすべて。上限は 3 つだけ)、`_Edges`、`TestUsageLimitSentences_AreStoppedSentences`、`TestUsageLimitScreen_EachStoppedSentence`、`TestUsageLimitLine` | M5 |
| AC9 skill、agent-messaging.md、仕様、AGENTS.md | 文面はテストで固定していない。写しの一致は run-test.sh の shell の段と Go の skill 系のテストが通った(cycle 2 は `86f7c1f8` の skill の写しを含む)。文の中身は /verify が読んで確かめた | — |

Test plan の項目との突き合わせ:

- Unit(上限の文の見分け、fixture すべて): AC8 の行のとおり、上限 3、通信断、入力待ち、動作中、選択の画面のどれも表にある
- Unit(見張り): Test plan に並んだ場合はすべてテストがある。2 座席で 1 件、3 座席目で recurred が 1 つ(note に 3 座席目)、ack 後と resolve 後は新しい件、append のあいだの resolve で新しい件、理由と本文、上限の leader と除外、上限でない leader の除外、条件が下りる、再び立つ、読めない・通信断で stall、健康な座席は読まない。(cycle 2)append のあいだの ack で新しい件になる場合も、競合のテストが加わった
- Unit(本文の上限): `TestWatchUsageLimit_TheItemBodyStaysWithinTheCap`(6 座席と 20 座席、長い文と制御文字。20 座席では後ろが `+N more` の 1 行になる)
- Integration(`RunWatchAll` の 1 周期): `TestWatchUsageLimit_AStatusFileWrittenBeforeUsageLimitedLoadsAndWorks`(古い形の状態ファイルから 1 周期回し、件が 1 件、書いた状態ファイルに座席が入る)
- Regression: 既存の見張りのテスト(stall、liveness、scope_change、recurrence、deadman、alert_notify)は org のパッケージ全体の回で通った(cycle 1 と cycle 2)
- Edge cases: leader だけが上限(AC11 の上限の行、`SEATS: leader`)、claude と codex の混在(AC1 のテスト)、画面の行の制御文字(`_Edges` の制御文字と不正な UTF-8 の行、`TestUsageLimitLine`、本文の上限のテスト)はテストがある。leader 以外だけが上限の場合は、コミットされたテストが無い(Test gaps 1)

## 変異

### Cycle 2(HEAD `9f95c51a`)

`34e5bd86` の直しについて、verifier(Coverage gaps の最後の項と V8)と self-review の N3 が挙げた 2 つ(M3 と M2 の再実行)と、直し自体の変異 1 つ(M6)を当てた。harness は cycle 1 と同じ形で、置き換えは文字列 1 つにし、置き換える文字列がファイルに 1 回だけ現れることと、置き換えの前と後が違うことを確かめてから当てる(evidence にある)。どれも `go test -overlay` で当て、worktree は書き換えていない(`git status --porcelain --ignored` は当てる前と後で同じ)。どの変異も build は通った。まず競合のテストを名前で回し、次に絞り込みの回全体(`-run 'UsageLimit|InboxAppendRecurred|Recur'`、RUN 210)で回した。

| 変異 | guard | 置き換え | 競合のテストだけの回 | 絞り込みの回全体 |
| --- | --- | --- | --- | --- |
| M3(再実行) | `escalateToInbox` が `usage_limited` を開いた件に畳まない(AC4) | `watch.go:2020` の `ok && !isUsageLimitedAlert(alertID) && w.escalateAsRecurrence(` を `ok && w.escalateAsRecurrence(` に | 死んだ。`watch_usage_limit_race_test.go:38: the seam ran 2 times; want once, between the read that found e1 open and the append` | 死んだ。競合のテストと、`_ASeatThatStopsLater.../e1_acked`(`watch_usage_limit_test.go:511`) |
| M2(再実行) | 読んだ時点で、座席の増えた条件の件が open でなければ記録しない(コメントの言う fast path) | `watch_usage_limit.go:259` の `if !ok \|\| (grown && item.State != InboxStateOpen) {` を `if !ok {` に | 生き残った | 生き残った(210 / 210 が通った) |
| M6 | 直し自体: 座席の増えた条件は、ロックの中で open を確かめてから足す | `watch_usage_limit.go:265` の `recur = w.org.Inbox.AppendRecurredIfOpen` を `recur = w.org.Inbox.AppendRecurred` に | 死んだ。`watch_usage_limit_race_test.go:45: items [...]; want e1 and a new e2, and a second ALERT`(e1 は acked のまま `Recurrences:1` と recurred の出来事を持ち、e2 が無い) | 死んだ。落ちたのは競合のテストだけ |

変異から分かったこと(cycle 2):

- M3 で競合のテストを止めたのは、seam の回数の assert(38 行目)である。M3 では新しい pending が acked の e1 に畳まれ、`escalateAsRecurrence` が seam をもう一度呼ぶので、回数が 2 になる。この assert は `Fatalf` なので、後ろの assert は走っていない。そこで、テストの写しの 38 行目だけを `Errorf` にし、M3 の `watch.go` と一緒に overlay で重ねて回し直した。すると 45 行目(e2 が無く、acked の e1 に recurred が付く)も落ちた。M3 は回数の assert だけでなく件の結果の assert でも落ちるので、verifier の V8 の推定(読んでの推定で、走らせていなかった)は 2 つとも当たっている
- M2 は self-review の N3 のとおり、結果の変わらない変異になった。読んだ時点で acked の件は、`AppendRecurredIfOpen` がロックの中で `ErrInboxAcked` で断り、`recordUsageLimitRecurrence` は stderr を書かずに false を返す。これは早い `return false` と同じ結果である。違うのは seam の呼ばれる回数とロックの 1 回だけで、production の結果には出ない。cycle 1 で M2 を落とした「e1 acked」の行は、今はこの道を通って通る
- M6 を落とすのは競合のテストだけである。「e1 acked」の行では件が読んだ時点で acked なので、読んだ時点の確かめ(M2 の guard)が先に false を返し、`AppendRecurred` まで届かない。直しを watch の側で固めているのは、競合のテスト 1 本である(Test gaps 5)
- inbox の側の `case openOnly && item.State == InboxStateAcked:`(`inbox.go:434`)には変異を当てていない。`TestInboxAppendRecurredIfOpen_*` の 3 本が通ることだけを確かめた

### Cycle 1(HEAD `c191529c`)

orchestrator の指示(ディスクのため変異は 5 個まで)に合わせ、指示の優先順に 1 つずつ当てた。各変異は文字列の置き換え 1 つで、置き換える文字列がファイルに 1 回だけ現れることを確かめてから当てた(harness は evidence にある)。どれも org を `-run 'UsageLimit'` で絞って回し、同じ絞り方の対照は overlay なしで通った。どの変異も build は通り、落ちたのはテストの assert である。

| 変異 | guard | 置き換え | 結果 | 落としたテスト |
| --- | --- | --- | --- | --- |
| M1 | 画面の読みを leader の待ちの除外より前に置く順(D4、AC11) | `watch.go` の `evaluateSeat` で、`if s.SeatID == LeaderIdentity { stalled = stalled && !w.leaderIsWaiting(...) } else if seatBusy(...) {...}` を `if stalled { ... readUsageLimitScreen ... }` の前に移す | 死んだ | `_ALeaderStoppedOnAUsageLimitIsReportedWhileItWouldBeWaiting` の 6 行すべて(`0 reads of the leader's screen ... want one read and no stall`)、`_ASeatThatStopsLater...` の 4 行(reviewer が動いているので leader が読まれず、e1 が leader を含まない) |
| M2(cycle 2 で書き換え) | 増えた座席を recurred にするのは件が open のときだけ(AC4)。`34e5bd86` のあとは、読んだ時点の確かめ(fast path)になった | `watch_usage_limit.go` の `if !ok \|\| (grown && item.State != InboxStateOpen) {` を `if !ok {` に | cycle 1 の HEAD では死んだ。cycle 2 の HEAD では生き残る(結果の変わらない変異。cycle 2 の節) | cycle 1 の HEAD では `_ASeatThatStopsLater.../e1_acked`(e2 ができず、e1 に recurred が付く)。cycle 2 の HEAD では、ロックの中の `AppendRecurredIfOpen` が同じ件を断るので、落ちるテストはない |
| M3 | `escalateToInbox` が `usage_limited` を開いた件に畳まない(AC4) | `watch.go` の `ok && !isUsageLimitedAlert(alertID) && w.escalateAsRecurrence(` を `ok && w.escalateAsRecurrence(` に | 死んだ | `_ASeatThatStopsLater.../e1_acked`(新しい ALERT が acked の e1 の recurrence になり、e2 ができない)。cycle 2 の HEAD では競合のテストも落とす |
| M4 | 理由を Subject より先に条件で決める(D3、AC10) | `watchEscalationReason` の `case isUsageLimitedAlert(alertID):` と `case subject == LeaderIdentity:` の順を入れ替える | 死んだ | `_TheItemIsDelivered.../no_inbox_store`(escalations.jsonl の reason と banner が `leader_is_anomaly_subject`)、`.../switch_on,_a_watch_that_accepts_work`(stderr の head)、`_APendingAlertWithSeats...` の 3 行 |
| M5 | `UsageLimitScreen` が上限の印を求める(D5) | `usage_limit_screen.go` の `if !found \|\| !slices.Contains(table.UsageLimit, s.Text) {` を `if !found \|\| (false && !slices.Contains(...)) {` に | 死んだ | `TestUsageLimitScreen_EveryFixture/claude-enotfound.txt`、`_Edges` の通信断の 3 行、`_EachStoppedSentence`、`_ASeatWhoseScreenIsNoUsageLimitStallsAsBefore/a_network_error` |

変異から分かったこと(cycle 1):

- (cycle 2 で書き換え)cycle 1 では、M2 と M3 を落としたのはどちらも「e1 acked」の 1 行だけだった。resolve 済みの 2 行は `openAlertItemForKey` が件を見つけないので、この 2 つの guard には届かない。AC5 の acked の行は条件が下りたあとの立ち上がり(`grown` が false)なので、これも届かない。cycle 2 の HEAD では、M3 は競合のテストでも落ち、M2 は落ちない(cycle 2 の節)
- M4 で、件の本文の `REASON` は落とす側に回らなかった。`usageLimitedEscalationBody` が `usageLimitReason` を直に書き、`escalateAlert` から渡る理由を使わないためである。渡る理由が効くのは stderr の head と、受信箱なしの道の escalations.jsonl だけで、director off の 2 行(escalations.jsonl の reason は受信箱の送り先の理由になる)はこの変異を見分けない。落としたのは上の 5 行である
- M1 は AC11 のテストのほかに AC4 のテストも落とした。AC4 の前提(reviewer が動いている周期に leader と implementer が上限)が、そのまま「ほかの座席が動いている」除外の場面になっているためである

## Coverage

(cycle 2 で書き換え。数は cycle 2 の HEAD で取り直した)

- Statement(`-run 'UsageLimit|Watch|InboxAppendRecurred|Recur'` で絞った org の回): パッケージ全体では 36.8%(絞った回なので全体の数字としては意味がない)。変更行で数えると、`internal/org` の production code の、origin/main `bb416c48` から足した行のうち文に当たる 253 行のうち 239 行(94.5%)を通った。cycle 1 は 244 行のうち 230 行(94.3%)。直しの 2 コミット(`34e5bd86` と `193b044a`)が足した行は、文に当たる 11 行(`inbox.go` 5、`watch_usage_limit.go` 6)をすべて通った
- 関数: 新しい関数と変えた関数は、次を除いて 100%(`AppendRecurred`、`AppendRecurredIfOpen`、`appendRecurred` も 100%)。`recordUsageLimitRecurrence` 79.2%(cycle 1 は 76.2%)、`withUsageLimitedLines` 75.0%、`cutUsageLimitText` 83.3%、`matchedStopSentence` 80.0%、`escalateToInbox` 93.8%(残りは変更の外の既存の分岐)、`sendAlert` 90.9%(同じく既存の分岐)
- Branch: Go の標準の道具では測っていない
- 通らなかった 14 行は、cycle 1 と同じ分岐である。行番号は cycle 2 の HEAD のもの:
  - `watch_usage_limit.go:247-250`: `recordUsageLimitRecurrence` の、まだ escalate されていない pending の alert に座席を足す分岐(Test gaps 2)
  - `watch_usage_limit.go:256-257`: 受信箱が読めないときに false を返す分岐
  - `watch_usage_limit.go:270-272`: `recur`(`AppendRecurred` か `AppendRecurredIfOpen`)が `ErrInboxResolved` と `ErrInboxAcked` 以外で失敗したときの stderr の 1 行
  - `watch_usage_limit.go:329-330`: `withUsageLimitedLines` で、head だけで余地が無いときに head だけを返す分岐
  - `watch_usage_limit.go:414-415`: `cutUsageLimitText` の n が 1 未満で "" を返す分岐
  - `usage_limit_screen.go:70`: `matchedStopSentence` が文を見つけられないときの戻り。`ClassifyResumeScreen` が stopped と読んだ画面では、どれかの stopped の文が先頭に合うので、おそらく届かない。未確認です
- `ErrInboxAcked` で false を返す道(`watch_usage_limit.go:269` の条件が偽になり、273 行目に進む)は、競合のテストが通した
- Notes: 集計は `git diff -U0 origin/main...HEAD` の足した行と cover profile の block を突き合わせた(evidence に手順と結果)。絞った回なので、パッケージ全体の回ならほかのテストが通る行がある可能性はある。ただし上の 14 行はどれも `usage_limited` の専用の分岐で、既存のテストが届く形ではない

## Failure analysis

| Test | Error | Root cause | Proposed fix |
| --- | --- | --- | --- |
| なし | — | — | — |

## Regression checks

| Previously broken behavior | Status | Evidence |
| --- | --- | --- |
| issue #312: 利用上限で止まった座席に stall の ALERT が上がり続け、上限だと分かるまで約 8 時間かかった | 直っている。上限の画面の座席は stall にならず、同じ周期に `usage_limited` の件 1 件になる | AC1〜AC3 のテスト、M1・M5 |
| (cycle 2)cross-review cycle 1 の ACTION_REQUIRED: 座席が増えたときの「件が open か」の確かめがロックの外にあり、読んだあと append の前に ack された件に recurred が付いた | 直っている。ロックの中で `AppendRecurredIfOpen` が acked の件を断り、増えた座席は新しい件になる | 競合のテスト、`TestInboxAppendRecurredIfOpen_ChecksTheStateUnderTheInboxLock`、M6 |
| #307a の recurrence(件が resolve されていなければ recurred、されていれば新しい件) | 変わらない。`usage_limited` の立ち下がりのあとの再びの立ち上がりもこの規則に乗る(この道は `AppendRecurred` のままで、cycle 2 の直しは触れていない) | `_TheConditionClears...` の 3 行、既存の `TestWatch_Recurrence_*` と `TestInboxAppendRecurred_*` が通った |
| leader の待ちの除外(#307b、`leaderIsWaiting`) | 上限でない leader では変わらない | AC11 の上限でない 3 行、既存の `TestWatch_LeaderStall_*` が通った |
| 見張りの既存の条件(stall、liveness、scope_change、deadman、alert_notify) | 変わらない | org のパッケージ全体が、cycle 1(823 秒)と cycle 2(1,179 秒)で通った |
| (cycle 2)取り込んだ origin/main の #317(upgrade の behavior notes) | この branch と合わせても通る | `internal/cli` と `internal/upgrade` のパッケージ全体が run-test.sh で通った |

## Test gaps

番号は cycle 1 のまま残し、cycle 2 で変わった項に印を付けた。

1. leader 以外だけが上限の場合に、コミットされたテストが無い(/verify の指摘と同じ。cycle 2 でも変わらない)。cycle 1 で、overlay で足した probe(`TestProbeUsageLimit_OnlyANonLeaderSeatIsLimited`、コミットしていない)で 2 つの場合を確かめ、どちらも通った。cycle 2 では probe を回し直していない。
   - leader が動いている(manifest に leader の出来事が毎周期ある)とき: 件は `usage_limited` の 1 件だけで、`SEATS: implementer`、本文に `implementer (claude): ⏺ You've hit your session limit · resets 3:45pm`、`REASON: usage_limit`。stall の ALERT は 0 で、leader の画面は読まれない
   - leader が上限でもなく動いてもいない(入力待ちの画面のまま)とき: 件は 2 件になる。e1 は leader の stall(`REASON: leader_is_anomaly_subject`)、e2 は `usage_limited`(`SEATS: implementer`)。leader の画面は stall が立つ周期ごとに読まれる(3 周期で 2 回)。leader の待ちの除外は、ほかの座席が動いているか leader の上げた件が未解決のときだけで、上限で止まった座席を待つ leader はどちらにも当たらないため、変更の前と同じく leader の stall が上がる。計画の AC11 は除外の規則を変えないと書いているので、テストの失敗ではない。上限の座席を待つ leader を除外に含めるかどうかは、製品の判断として残る
   - 足すなら最小のテストは、probe の「leader が動いている」行である(evidence に全文がある)
2. `recordUsageLimitRecurrence` の pending の分岐(cycle 2 の HEAD で `watch_usage_limit.go:247-250`)にテストが無い。見張りは状態を周期をまたいでメモリに持つ(`watchAllRun.statuses`)ので、上限を立てた周期で manifest の読み直し(`rr2`)が失敗すると、`evaluateCycle` は deadman の前に戻り、送った ALERT の pending が残る。次の周期に座席が増えると、この分岐で pending の座席が差し替わる。めったに無いが届く道である
3. (cycle 2 で書き換え)`recordUsageLimitRecurrence` の、受信箱が読めないとき(256-257)と、`recur`(`AppendRecurred` か `AppendRecurredIfOpen`)が `ErrInboxResolved` と `ErrInboxAcked` 以外で失敗したとき(270-272)の分岐にテストが無い。どちらも新しい ALERT に倒れる作りで、270-272 の stderr の文は固定されていない
4. `withUsageLimitedLines` の head だけを返す分岐(329-330)と `cutUsageLimitText` の n < 1 の分岐(414-415)にテストが無い。座席の id は識別子で長さに上限があるので、head が 2,000 字を超える場面はおそらく無い。未確認です
5. (cycle 2 で書き換え)M3 の guard は、「e1 acked」の行と競合のテストの 2 本で固まっている。直し自体(M6、座席の増えた条件を `AppendRecurredIfOpen` で足すこと)を落とすのは競合のテスト 1 本だけで、inbox の側の `openOnly` の行(`inbox.go:434`)には変異を当てていない。acked の件に、条件が立っているあいだに座席が 2 回続けて増える場合など、2 行目があるとよいことは cycle 1 と変わらない
6. 件の本文の `REASON` は `usageLimitReason` を直に書くので、理由を条件で決める `watchEscalationReason` の順は、本文ではなく stderr の head と受信箱なしの道でしか見えない(M4 の節)。今のテストはその 2 か所で固めている
7. self-review の Known gaps のまま残る 2 点: 画面が読めなかった理由は leader への stall の ALERT(agmsg)にだけ出て、受信箱の件の本文には届かない(N1)。画面が読めなかった周期のあとに上限が読めた座席は「増えた」と数えられ、件が open なら recurred、acked なら新しい件になる(L5)。どちらも計画どおりの挙動で、テストの失敗ではない
8. 上限の画面は fixture だけで確かめている。claude の weekly limit は実機の画面(2026-11-07 の evidence)だが、session limit と codex の 2 つの文は文書と binary の文字列からで、実機の画面では確かめていない
9. AC9 の文面(director と org の skill、agent-messaging.md、仕様、AGENTS.md、`ralph org watch --help`)はテストで固定していない。/verify が読んで確かめた
10. (cycle 2 で足した)M2 は結果の変わらない変異である(self-review N3)。座席の増えた条件の件が acked なら記録しないことを、読んだ時点の確かめとロックの中の `AppendRecurredIfOpen` の 2 か所が決めていて、前の方を消しても結果は変わらない。見分けられる違いは production の結果に無いので、テストを足して閉じる穴ではなく、コードの形の問題として N3 に残る

## Verdict

- Pass: cycle 2(HEAD `9f95c51a`)の run-test.sh(changed scope、shell 45 本、Go 10 パッケージ、1,928 秒、rc 0)、絞り込みの 2 つの回(RUN 210、`-race` を含む)、coverage の回、`(cached)` の 4 パッケージの回し直しが通った。`34e5bd86` の直しに当てた変異のうち、M3 と M6 は死に、M2 は self-review N3 のとおり結果の変わらない変異として生き残った。cycle 1 の M2 の行はこの結果に書き換えた。cycle 1 の所見は、M2 と、Test gaps 3 と 5 の行番号と言い方を除いて、そのまま成り立つ。残る穴は Test gaps の 10 項目で、どれも通るべき挙動の失敗ではない
- Fail: 0
- Blocked: なし

### Verdict(cycle 1)

- Pass: run-test.sh(changed scope、shell 45 本、Go 10 パッケージ、1,574 秒、rc 0)、変異の対照、probe、coverage の回のすべてが通った。指示の優先順に当てた変異 5 個(M1〜M5)はすべて死んだ。Test plan の項目は、「leader 以外だけが上限」の edge case を除いてテストがあり、その場合も probe で期待どおりの件が 1 件できることを確かめた。残る穴は Test gaps の 9 項目で、どれも通るべき挙動の失敗ではない
- Fail: 0
- Blocked: なし
