# Test report: org-result-missing

- Date: 2026-11-22(run-test.sh は 12:47:02〜13:13:16 JST。パイプラインの cycle 1)
- Plan: `docs/plans/active/2026-11-22-org-result-missing.md`(issue #361)
- Tester: tester サブエージェント(`/test`)
- Scope: HEAD `46c5040b`(worktree は clean、push 済み。base は origin/main `2a069b50`)。`./scripts/run-test.sh` を `RALPH_VERIFY_SCOPE=full` で 1 回だけ回した。打つ前に `ps -axo pid,command | grep -E 'run-test.sh|run-verify.sh'` を見て、ほかの run がないことを確かめた。Bash tool の 10 分の上限を超えるので、perl の fork と setsid で切り離して起動し(INT・HUP・QUIT・TERM は既定の扱いに戻し、標準入出力は `/dev/null`)、`caffeinate -w` で 168 s ずつ待った。`nohup` は使っていない。そのあと、足したテストの `-v -count=1` の run、変異 13 個、調べのテスト 2 本、coverage の run を、HEAD の `git archive` の写し(セッションの scratchpad の下)で回した。木には何も書いていない
- Evidence: `docs/evidence/test-2026-11-22-org-result-missing.log`(6,140 行。`.gitignore` の `docs/evidence/*.log` に当たるので commit されない)。5,828 行目までが run-test.sh の出力で、そのあとに私が足した節がある。1 分ごとの load、`~/.codex/config.toml` と herdr のプロセスの前後、`-v` の run の結果、変異ごとの diff と落ちたテスト、全体の run、調べのテスト 2 本、coverage の要約。wrapper 自身の log は `docs/evidence/verify-2026-11-22-034702.log`(同じく追跡外)

## Test execution

| Suite / Command | Tests | Passed | Failed | Skipped | Duration |
| --- | --- | --- | --- | --- | --- |
| `RALPH_VERIFY_SCOPE=full ./scripts/run-test.sh` | 下の 2 行の合計 | 全部 | 0 | 2(条件つき。下の行) | 12:47:02〜13:13:16 JST、1,574 s、rc 0。`==> All verifiers passed.` |
| local verifier(`tests/test-*.sh` 46 本) | 46 本、`PASS` で始まる行 5,390 | 46 本 | 0(要約行の `FAIL:` はどれも 0) | `test-plan-visual.sh` の browser detection(`/Applications` にブラウザーがある)、`test-secret-scan-branch.sh` の「git 2.41 より古い本物の git」の場合(手元は git 2.49.0。stub の古い git の行は通る) | 約 13 分(12:47〜13:00 ごろ) |
| `go test ./...`(テストのある 11 パッケージ) | 11 | 11 | 0 | なし。`(cached)` は 0 | 約 13 分(13:00 ごろ〜13:13:16)、下の表 |

| パッケージ | 結果 | 時間 |
| --- | --- | --- |
| `cmd/ralph` | ok | 0.948 s |
| `internal/cli` | ok | 745.214 s |
| `internal/config` | ok | 17.723 s |
| `internal/gitroot` | ok | 2.554 s |
| `internal/insights` | ok | 1.698 s |
| `internal/org` | ok | 772.218 s |
| `internal/org/driver` | ok | 3.670 s |
| `internal/org/protocol` | ok | 2.273 s |
| `internal/scaffold` | ok | 3.869 s |
| `internal/toolchain` | ok | 23.905 s |
| `internal/upgrade` | ok | 3.330 s |

shell の段の `PASS` の行は 5,390 で、org-start-issue の run(`docs/reports/test-2026-11-22-org-start-issue.md`)と同じ数だった。この branch は shell のテストを足していない。`internal/cli` と `internal/org` は golang pack の 1 パッケージの上限 30 分のおよそ 43% で終わった。`(cached)` のパッケージはなかったので、`-count=1` の打ち直しは要らなかった。

1 分 load は run の全体で 5.7〜30.0(1 分ごとに 27 回測った)。高かったのは 12:51〜12:54 の 23〜30 で、私の変異の run(4 個ずつ並べて回した)と coverage の run が shell の段に重なった分。Go の段(13:00〜13:13)は 7.5〜17.9 で、その最初の 1 分に変異 O6 の絞り込みの run(47 s)と調べのテストが重なった。落ちたテストはなく、打ち直しもしていない。

run の前後で、`~/.codex/config.toml` の mtime(1795174910)、大きさ(38,144 バイト)、sha256(`0f076aef…`)は同じだった。書かれていない(読まれたかどうかは、この machine では atime が動かないので測れない)。herdr のサーバー(pid 59855)も前後で同じプロセスで、触れていない。`pkill` と `killall` は使っていない。

### 足したテストの `-v -count=1` の run

run-test.sh には `-v` がないので、足したテストと書き換えたテストを関数の名前の `-run` で回し直した(worktree で、`go test -count=1 -v ./internal/org/ -run 'ResultMissing|SeatSend|OrgSpawn_SeatRoles_AgmsgHomeQuotedIntoPromptFile|SeatsSendMessagesFromAFile|Implementer_ScopeRuleExceptsTheMessageFiles|Reviewer_XreviewTaskIsNotAReviewTask|Leader_ImplementerTaskExampleCarriesTheVerifyLine'`)。rc 0。`=== RUN` は 119(関数 44 と subtest 75)で、119 がすべて PASS、SKIP は 0、FAIL は 0。

- `watch_result_missing_test.go` の 28 関数、`prompts_seat_send_test.go` の 7 関数、`prompts_leader_result_missing_test.go` の 5 関数、合わせて 40 関数。ファイルに宣言された `Test` 関数と PASS の行の名前を突き合わせ、走らなかったものは 0 だった
- 書き換えた既存のテスト 4 関数(`prompts_verification_test.go` の 3 つ、`prompts_xreview_reviewer_test.go` の 1 つ)も PASS
- rule と skill をディスクから読むテスト(`TestAgentMessagingRule_ResultMissingAndTheSeatsSendCheck`、`TestOrgSkill_ResultMissingIsInTheWatchRowAndTheAlertItems`、`TestDirectorSkill_ResultMissingAlertItemReadsTheSeatsScreen`)は、写しがないと skip する形だが、ここでは skip していない

### 計画のテストと AC の対応

| AC | テスト | 結果 |
| --- | --- | --- |
| AC2 | `prompts_seat_send_test.go` の 6 関数(送信のコマンドの形、終了コードと 1 回の送り直し、ファイルの先頭の `TYPE:`、0 のときの履歴の確かめ、結果を送らずに手番を終えない、reviewer のメッセージのファイル)と `TestOrgSpawn_SeatRoles_AgmsgHomeQuotedIntoPromptFile` | PASS |
| AC3 | `TestWatch_ResultMissing_RaisesForAnIdleSeatWithoutTheResultAndClearsOnIt`、`_TheClockStartsAtTheTaskOrAtTheLeadersAnswer`、`_TheAlertSaysWhereTheClockStarted`、`_EscalatesAsAnAlertItemOfTheInbox` | PASS。変異 O2〜O4 で落ちる |
| AC4 | `_IsNotRaisedWhenTheSeatAnsweredOrItCannotBeTold`(18 行)、`_TheSeatGetsNoStallWhileItHolds`、`_HoldsTheSeatsStallWhileItsClockRuns`、`_TheStallIsDecidedAsBeforeOnceTheClockRunsOutWithoutIt`、`_DoesNotHoldTheStallWhenItsClockDoesNotRun`、`_DoesNotHoldTheLeadersStall`、`_ANewTaskClearsItAndIsWatchedOnItsOwn`、`_IsNotRaisedForTheLeader`、`_ASeatStoppedOnAUsageLimitGetsNone`、`_AHistoryReadThatFailsIsTriedAgain` | PASS。変異 S2b・O1・O5 で落ちる |
| AC5 | `prompts_leader_result_missing_test.go` の 5 関数。`check-sync.sh` と `check-skill-sync.sh` は local verifier の中の `scripts/verify.local.sh` で通った | PASS |
| AC6 (a)(b)(c)(e) | `_TheClockStartsAtTheTaskOrAtTheLeadersAnswer` の行 `(a)` 3 つ、`(b)` 4 つ、`(c)`、`(e)` | PASS |
| AC6 (d) | `_ReadsFurtherBackForAResultTheFirstLimitLeftOut`、`_IsNotRaisedWhenTheHistoryDoesNotReachBackToTheTask`(1000 行と 999 行) | PASS。変異 Ra・Rb で落ちる。Rc は残る(Test gaps の 1) |
| AC6 (f)(g)(i) | `_IsNotRaisedWhenTheSeatAnsweredOrItCannotBeTold` の行「the RESULT came in the TASK's own second」「a RESULT the leader answered with a DECISION」「a STOP followed the TASK」 | PASS |
| AC6 (h) | `_WaitsForTheSeatToStayWaitingForStallMinutes` | PASS |
| self-review M1・R2-L2 | `_ANewTaskAlertsWhileTheEarlierTasksItemIsOpen`、`_ANewTaskIsANewAlertWhileTheConditionIsStillActive`、`_TheSameTaskRisingAgainIsARecurrenceOfItsOwnAlert`、`_TheSameTaskIsNotFoldedIntoAnotherTasksItem`、`_TheSameTaskAfterALostStatusFileIsARecurrenceOfItsItem`、`_ANewTaskAfterALostStatusFileIsNotFoldedIntoTheEarlierTasksItem`、`TestResultMissing_AlertOfTask`、`_EachNewAlertStartsTheWatcherOnce` | PASS。変異 M1a〜M1c で落ちる。O6 は残る(Test gaps の 2) |

AC1(S0 の調べの記録)は文書なので、このテストの対象ではない。

## Coverage

- Statement: `internal/org` を `-run 'Watch|ResultMissing|Stall|SeatSend|RenderRolePrompt|Skill|AgentMessaging|Reseat'` で回した run(rc 0、152.7 s)で、`watch_result_missing.go` の関数ごとの値は次のとおり。100%: `resultMissingTaskKey`、`updateIdleSince`、`resultMissingAlertMessage`、`isResultMissingAlert`、`raiseResultMissing`、`resultMissingAlertOfTask`、`waitedFor`、`seatLastTask`、`historyReachesBack`、`resultMissingLineOf`、`parseAgmsgHistory`、`parseAgmsgHistoryLine`。`resultMissing` 97.2%、`seatLinesToLeader` 95.0%、`firstSentAfter` 77.8%、`recordResultMissingRecurrence` 69.0%。`watch.go` の `evaluateSeat` は 100%、`escalateToInbox` は 93.8%、`reseat_progress.go` の `sentDetailsText` は 100%
- Branch: 測っていない(Go の標準の道具にない)
- Function: 上のとおり
- Notes: 通らなかった block は、`resultMissing` の `IdleSince` が RFC 3339 でないとき、`firstSentAfter` の時刻が読めない `sent` を飛ばす `continue`、`seatLinesToLeader` の agmsg に履歴の読みがないとき、`recordResultMissingRecurrence` の 6 か所(manifest を読めない、受信箱がない、受信箱を読めない、追記に失敗した、と、leader が応じた保留中の ALERT を消す `delete` の行)。最後の 1 つだけは失敗の扱いでなく普通の流れなので、Test gaps の 2 に書いた

## 変異

HEAD の `git archive` の写しに、置き換えたファイルを `go test -overlay` で重ねて回した。置き換える文字列はファイルの中に 1 回だけあることを確かめてから替えた。どの変異も `build failed` は 0 件で、落ちたものはテストの assert で落ちた(タイムアウトや panic ではない)。対照(overlay なし)は `go test -count=1 ./internal/org/ -run 'Watch|ResultMissing|Stall'` が rc 0(55 s)。変異は同じ絞り込みで回し、残った 2 つは `internal/org` の全体でも回した。

| # | 変異 | 落ちたテスト |
| --- | --- | --- |
| M1a | 同じ TASK の dedupe が TASK を見ない(`rec.Active && sameTask` → `rec.Active`) | `_ANewTaskIsANewAlertWhileTheConditionIsStillActive` |
| M1b | `escalateToInbox` が result_missing の ALERT も open の件の再発にする(`!isResultMissingAlert(alertID)` を消す) | `_ANewTaskAlertsWhileTheEarlierTasksItemIsOpen`、`_ANewTaskAfterALostStatusFileIsNotFoldedIntoTheEarlierTasksItem` |
| M1c | 別の TASK でも前の ALERT の再発として数える(`sameTask &&` を消す) | `_ANewTaskAlertsWhileTheEarlierTasksItemIsOpen`、`_ANewTaskIsANewAlertWhileTheConditionIsStillActive`、`_TheSameTaskIsNotFoldedIntoAnotherTasksItem` |
| S2b | stall を保留するのが「立っている」ときだけ(`!= resultMissingNo` → `== resultMissingHolds`。時計が回るあいだは保留しない) | `_HoldsTheSeatsStallWhileItsClockRuns`、`_TheStallIsDecidedAsBeforeOnceTheClockRunsOutWithoutIt` |
| Ra | 最後の上限でも遡れない履歴を「読めた、行はない」とする(`return nil, false, true` → `return nil, true, true`) | `_IsNotRaisedWhenTheHistoryDoesNotReachBackToTheTask`、`_TheStallIsDecidedAsBeforeOnceTheClockRunsOutWithoutIt` |
| Rb | 「上限より少ない」を「上限以下」にする(`<` → `<=`) | `_ReadsFurtherBackForAResultTheFirstLimitLeftOut`、`_IsNotRaisedWhenTheHistoryDoesNotReachBackToTheTask`、`_TheStallIsDecidedAsBeforeOnceTheClockRunsOutWithoutIt` |
| Rc | 遡れたとみなす最古の行を「TASK の秒より前」から「TASK の秒以前」にする(`r.At.Before(since)` → `!r.At.After(since)`) | 残る。絞り込みで rc 0、`internal/org` の全体の結果は下の段落。Test gaps の 1 |
| O1 | 座席の今の life を見ない(`start = i + 1` を消す。前の life の TASK も数える) | `_IsNotRaisedWhenTheSeatAnsweredOrItCannotBeTold`、`TestResultMissing_SeatLastTask` |
| O2 | 別の TASK の BLOCKED でも leader の答え待ちにする(`last.TaskID == task.TaskID` を消す) | `_TheClockStartsAtTheTaskOrAtTheLeadersAnswer` |
| O3 | QUESTION と同じ秒の leader の `sent` を答えに数える(`ts.After(at)` → `!ts.Before(at)`) | `_TheClockStartsAtTheTaskOrAtTheLeadersAnswer` |
| O4 | 座席が受けた行も送った行に数える(`From == seat && To == leader` → `From == seat \|\| To == seat`) | `_TheClockStartsAtTheTaskOrAtTheLeadersAnswer` |
| O5 | TASK のあとの STOP を見ない | `_IsNotRaisedWhenTheSeatAnsweredOrItCannotBeTold`、`_DoesNotHoldTheStallWhenItsClockDoesNotRun`、`TestResultMissing_SeatLastTask` |
| O6 | 保留中の ALERT に leader が応じたあとでも、その ALERT の再発に数える(`if !w.leaderActedSinceAlert(...)` → `if true \|\| !w.leaderActedSinceAlert(...)`) | 残る。絞り込みで rc 0、`internal/org` の全体の結果は下の段落。Test gaps の 2 |

13 個のうち 11 個が落ちた。依頼にあった 3 つの振る舞いは、どれも変異で落ちることを確かめた。TASK ごとの新しい ALERT(M1a〜M1c)、stall の保留(S2b)、履歴が遡れないときに上げないこと(Ra、Rb)。

残った Rc と O6 は、run-test.sh が終わったあと(13:13:35〜13:26:38)、`internal/org` の全体(`go test -count=1 -timeout 30m -overlay <json> ./internal/org/`)で 2 つ並べて回した。Rc は rc 0(781 s)、O6 は rc 0(783 s)で、どちらも全体でも残る。どちらの変異の行も `internal/org` の中からしか呼ばれない(`historyReachesBack` と `recordResultMissingRecurrence` を呼ぶのは `watch_result_missing.go` だけ)ので、`internal/cli` では回していない。

### 残った変異を落とす調べのテスト

残った 2 つについて、振る舞いが本当に変わるかを使い捨てのテストで確かめた。どちらも overlay で写しに足しただけで、commit していない。

- Rc: 座席の RESULT が TASK と同じ秒にあり、そのあとに同じ秒の行が 50 行ある(最初の読みの 50 行の最古が TASK の秒にある)。HEAD は最初の読みを「遡れていない」として 200 で読み直し、RESULT を見つけて上げない(PASS)。Rc は最初の読みで遡れたとし、RESULT を見ずに `result_missing` を上げる(FAIL: `history reads [{implementer 50}]`、ALERT 1 件)。実地で TASK と同じ秒に RESULT が来て、さらに 50 行が続くことはまずないので、害は小さい
- O6: t1 の ALERT が保留中のまま、座席が 1 周働いてまた入力待ちになり、t1 が再び立つ周の直前に leader が動く(別の座席への `sent`)。HEAD は保留中の ALERT を消して t1 の 2 つ目の ALERT を leader に送る(PASS)。O6 は応じ済みの保留中の ALERT に再発を数え、新しい ALERT を送らない(FAIL: `ALERTs 1 ... want a second one`)。同じ周の終わりの deadman がその保留中の ALERT を消すので、再び立ったことは leader にも受信箱にも届かない

## Failure analysis

| Test | Error | Root cause | Proposed fix |
| --- | --- | --- | --- |
| なし | — | — | — |

run-test.sh、`-v` の run、対照の run、coverage の run で落ちたテストはない。既知の flake(`flaky_tests.md` の一覧)も今回は出なかった。

## Regression checks

| Previously broken behavior | Status | Evidence |
| --- | --- | --- |
| 座席の stall、liveness、scope_change、usage_limited の ALERT(計画の Regression tests) | 変わっていない。この branch は `watch_test.go` などの既存のテストを書き換えておらず、`internal/org` の全体が rc 0 で通った | run-test.sh の `internal/org` ok(772.218 s) |
| leader の stall(leader は result_missing で保留しない) | 変わっていない | `_DoesNotHoldTheLeadersStall`、`_IsNotRaisedForTheLeader` が PASS |
| 座席が送るメッセージのファイルの決まり(`prompts_verification_test.go` の 3 関数、`prompts_xreview_reviewer_test.go` の 1 関数) | 期待の文に QUESTION と「leader に送るメッセージのファイル」を足しただけで、意図は弱めていない | 4 関数が `-v` で PASS |

## Test gaps

1. 履歴が遡れたとみなす境目(Rc)を固定する行がない。最初の読みの最古の行がちょうど TASK の秒にあるとき、HEAD は読み直すが、その場合の行がないので「TASK の秒以前」に変えても通る。上の調べのテスト(RESULT が TASK と同じ秒、そのあと同じ秒の 50 行、読みは 50 と 200、ALERT なし)を `TestWatch_ResultMissing_ReadsFurtherBackForAResultTheFirstLimitLeftOut` の隣に足せば固定できる。実地で起きる見込みは小さいので、足すかどうかは任せる
2. `recordResultMissingRecurrence` の「leader が応じた保留中の ALERT を消して、新しい ALERT にする」流れ(`watch_result_missing.go:372`)を通るテストがない。同じ TASK が再び立つ周の直前に leader が動いた場合だけ通る。O6(応じ済みでも再発に数える)が残り、そのときは再び立ったことがどこにも届かない。上の調べのテストを `_TheSameTaskRisingAgainIsARecurrenceOfItsOwnAlert` の隣に足せば固定できる
3. `recordResultMissingRecurrence` の失敗の扱い(manifest を読めない、受信箱を読めない、追記に失敗した)を通るテストがない。どれも「新しい ALERT を送る」側に倒すので、知らせが消えることはない。doc comment と rule は、そのとき同じ TASK に 2 つ目の件ができることを書いている(self-review R2-L2 の扱い)
4. `firstSentAfter` の時刻が読めない `sent` を飛ばす分岐を通るテストがない。台帳の `sent` はいつも RFC 3339 で書かれるので、実地で通る見込みは小さい
5. 座席の雛形の新しい手順(送信、終了コード、送り直し、`SEND FAILED:`、履歴の確かめ)は文で固定しているだけで、本物の座席がその手順で送ることは確かめていない。本物の agmsg と herdr を使う実地の run は、この run の範囲の外(本物の herdr のサーバーに触れない、という依頼の条件)
6. `internal/cli` には result_missing のテストがない。見張りの判定は `internal/org` の中で閉じていて、CLI の変更は `org.go` のコメント 2 行だけなので、足す必要はないと見る

## Verdict

- Pass: run-test.sh(full)rc 0、1,574 s。shell 46 本と Go 11 パッケージがすべて通り、`(cached)` は 0、skip は既知の条件つきの 2 つだけ。足したテスト 40 関数と書き換えた 4 関数は `-v -count=1` で 119 の RUN がすべて PASS、SKIP 0。変異 13 個のうち 11 個が落ち、残る 2 つ(Rc、O6)はどちらも狭い境目で、知らせが届かなくなる方向は O6 の 1 つ(Test gaps の 1 と 2)
- Fail: 0
- Blocked: なし。/pr に進んでよい。Test gaps の 1 と 2 は、テストを足すかどうかの判断を orchestrator に任せる
