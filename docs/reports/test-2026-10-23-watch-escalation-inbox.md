# Test report: watch-escalation-inbox

- Date: 2026-10-23(JST 05:42〜06:21、UTC 2026-10-22 20:42〜21:21)
- Plan: `docs/plans/active/2026-10-23-watch-escalation-inbox.md`(試行の所見を直す系列の 6 本目、F6・F13。AC1〜AC10 の挙動と Test plan の Unit・Integration・Regression・Edge cases)
- Tester: tester subagent(Claude Opus 5.5)。パイプラインの 1 回目(`cycle-count.json` の cycle 1)
- Scope: HEAD 47c213f6(base e8e8d652)。verify の報告(7780966a)のあとのコミット 8d1c53ce・000a2352・47c213f6 が変えたのは、Go のコメント 2 か所(`internal/org/watch.go`、`internal/config/config.go`)、`templates/base/ralph.toml` のコメント、新しいテスト `TestOrgWatch_HelpDescribesTheAlertItemEscalation`、文書、計画、insight event で、挙動のコードは変えていない。behavioral test だけを実行した(静的解析は /verify の担当)。環境は macOS(Darwin 25.5.0、arm64、12 CPU)、git 2.49.0、go1.26.0 と go1.25.8(`GOTOOLCHAIN=go1.25.8`、toolchain はキャッシュ済み)
- Evidence: `docs/evidence/test-2026-10-23-watch-escalation-inbox.log`(各実行の出力、時刻と load average、mutant の定義と結果、canary と probe の記録、関数ごとの coverage。ホームは `~`、session の scratchpad は `<scratch>`、`$TMPDIR` は `<tmp>` と書いた)と、run-test.sh が書いた `docs/evidence/verify-2026-10-22-204206.log`。どちらも `docs/evidence/*.log` の gitignore の対象で、コミットしない
- 追跡しているファイルは、この報告のほかは変えていない。mutant と probe は `go test -overlay` で scratchpad のファイルに差し替えて打ち、worktree のファイルには書いていない(`git status --porcelain` は始める前が 0 行、終わったあとはこの報告の 1 行だけ。`.harness/cache/` を除いた `--ignored` の一覧を、1 の実行中(mutant A1〜A24 のあと)に取ったものと比べると、増えたのはこの報告と、この報告の evidence だけ。取った時点の一覧にも、計画の図解、`.harness/state/` の状態ファイル、`docs/evidence/` の log のほかのファイルはなかった)。本物の herdr・agmsg・osascript・gh・network を打つものは実行していない(「本物を打たないことの確かめ」)。insight event は追記していない(末尾の「Insight event」)

## Test execution

| # | Suite / Command | Tests | Passed | Failed | Skipped | rc と時間 |
| --- | --- | --- | --- | --- | --- | --- |
| 1 | `GOFLAGS='-timeout=30m -count=1' ./scripts/run-test.sh` | shell 44 本、Go 8 パッケージ | shell 44 本、Go 8 パッケージ | 0 | shell 1 件(下) | rc 0、851 秒(05:42:05〜05:56:16)。cli 328.3 秒、org 388.0 秒。開始時の load average(1 分)3.6、終了時 8.3 |
| 2 | `GOTOOLCHAIN=go1.25.8 go test -count=1 -timeout 30m ./internal/org/... ./internal/cli/... ./internal/config/...` | 5 パッケージ(org、org/driver、org/protocol、cli、config) | 5 | 0 | 0 | rc 0、391 秒(05:56:32〜06:03:03)。org 386.8 秒、cli 326.0 秒。3・4 と 5 の後半(A25 以降)と重なった |
| 3 | 新しいテストと、同じ関数を見る前からのテスト(下の正規表現)を canary の PATH と偽の HOME で `-count=3 -v`(org)と `-count=1 -v`(cli) | org は RUN 885(1 回あたり 295、subtest を含む)、cli は RUN 148 | 885、148 | 0 | 0(`--- SKIP` は 0 行) | org rc 0、9 秒。cli rc 0、22 秒。canary の呼び出しは 0 回 |
| 4 | 3 の org と同じ集合を `-race -count=1 -v` | RUN 295 | 295 | 0 | 0 | rc 0、16 秒。`DATA RACE` は 0 行 |
| 5 | mutant 54 本(下の表)。それぞれ 3 の集合を canary の PATH で | 54 | - | 51 本が赤 | - | 1 本 7〜11 秒。生き残りは A40・A49・A50 の 3 本 |
| 6 | base(e8e8d652)のテストファイル 8 つを HEAD のコードに重ねる(`-overlay`、canary の PATH) | cli は base の 2 ファイルの 68 本(RUN 157)、org はパッケージ全体の 1,153 本(RUN 3,883) | cli 67 本、org RUN 3,881 | cli 1 本、org 1 本(下) | org 1 件(base の darwin の skip) | cli rc 1、28 秒。org rc 1、412 秒(7 と並べて打った) |
| 7 | org パッケージ全体を canary の PATH で `-coverprofile` | 1 パッケージ | 1 | 0 | 0 | rc 0、412 秒(org 406.4 秒)。statement 94.9% |
| 8 | 通知の題の probe 2 本(`-overlay` で足した使い捨てのテスト。HEAD と A49・A50) | 2 × 3 | HEAD 2、A49 1、A50 1 | A49 1、A50 1 | 0 | HEAD rc 0、mutant rc 1 |

- 1 の run-test.sh は `==> Language scope: full fallback (unclassified:templates/base/ralph.toml)`、pack は golang。`templates/base/ralph.toml` を変えたので changed の範囲が全体に落ちた。GOFLAGS に `-count=1` を足したので、Go の 8 パッケージはどれも `(cached)` ではない。shell の 44 本はどれも最後の行が `OK` だった。Skipped は前からある `tests/test-secret-scan-branch.sh` の 1 件で、git 2.49.0 なので「2.41 より古い本物の git」の事例を飛ばしている
- 3・4・5 の org の集合は `-run 'TestWatch_Escalation|TestWatch_Deadman|TestRunWatchAll_(AOnceRun|ALongRunningWatch|ACycleWithAStopRequest|TwoOrgsEscalating|WakesTheDirectorOnceForAnEscalatedItem)|TestInbox|TestEscalat|TestSendInboxItemToHuman|TestNotifiableType|TestAlertIDSeat|TestSweepInboxDeadlines|TestNotifyInboxItem|TestRenderDirectorPrompt_LeavesTheStop'`。cli の集合は `TestOrgWatch|TestOrgInbox|TestOrgEscalate` で、新しい `TestOrgWatch_HelpDescribesTheAlertItemEscalation` と、期待を直した `TestOrgEscalateAndInboxHelp_DescribeTheDirectorSwitch` を含む
- AC10 の Go のテストの部分は 1 と 2 が通った。静的な部分(`run-static-verify.sh`)は verify の報告にある。`run-verify.sh` を 1 本のコマンドとしては打っていない

## AC とテストの対応

| AC | テスト | 確かめたこと |
| --- | --- | --- |
| AC1 | `TestWatch_Escalation_ASeatsAlertAndAnAlertAboutTheLeaderEachBecomeOneAlertItem`(`internal/org/watch_test.go:950`、2 件)、`TestInboxEscalate_WritesAndFoldsTheAlertID`(`internal/org/inbox_test.go:265`)、`TestInboxRead_AlertIDComesOnlyFromTheEscalatedEvent`(`inbox_test.go:316`) | 座席の ALERT(deadman_timeout、6 分後のサイクル)と leader の ALERT(その場で)が、それぞれ TYPE ALERT の件 e1 を 1 つだけ書く。本文は `protocol.Parse` と `protocol.Validate` を通り、見出しは `ORG_ID`・`SEAT`・`CONDITION`・`REASON`・`ALERT_ID`・`ALERT_TS` のちょうど 6 つ(`reflect.DeepEqual`)、説明は 1 行。escalated の event と件の両方が alert_id を持つ。inbox.jsonl の行をバイトで比べ、alert_id のない件には欄が出ない。欄のない古い行は "" で畳まれ、壊れた行に数えない。あとの event の alert_id は件の値を変えない。mutant A29・A30・A31・A36 が赤 |
| AC2 | `TestWatch_Escalation_LeavesTheAlertItemToTheDirectorWhenThisWatchAcceptsWork`(`watch_test.go:1097`)、`TestRunWatchAll_ALongRunningWatchLeavesItsEscalationToTheDirectorUntilADeadline`(`internal/org/watch_all_test.go:2259`、2 件)、`TestSweepInboxDeadlines_SendsAnAlertItemWithItsAlertIDAndSeat`(`internal/org/deadline_test.go:617`) | delivery deadline の件は escalations.jsonl を作らず、通知もしない。stderr はちょうど 1 行で、`WATCHDOG ESCALATION`、件の ID、director に任せたこと、ack と resolve の期限、`--state-dir` つきの `inbox show` を文字列全体で比べる。長く動く見張りでは、open の件は 11 分後のサイクルで `inbox_ack_deadline`、ack した件は ack の期限では送られず、61 分後のサイクルで `inbox_resolve_deadline` で届く。その行は alert_id と subject `leader` を持つ。mutant A3・A12・A41・A48 が赤 |
| AC3 | `TestWatch_Escalation_SendsTheAlertItemAtOnceUnlessThisWatchCanLeaveItToTheDirector`(`watch_test.go:1034`、4 件)、`TestRunWatchAll_AOnceRunSendsItsEscalationAtOnceWithTheDirectorOn`(`watch_all_test.go:2232`)、`TestRunWatchAll_ACycleWithAStopRequestSendsItsEscalationAtOnce`(`:2341`、2 件)、`TestSendInboxItemToHuman_AnAlertItemCarriesItsAlertIDAndSeat`(`internal/org/escalate_test.go:1028`、4 件)、`TestWatch_Escalation_ASendThatIsNotCompletedNamesInboxNotifyAndLeavesTheItemToTheAckDeadline`(`watch_test.go:1129`)、`TestAlertIDSeat`(`escalate_test.go:1152`、15 行) | director が無効なら `inbox_director_disabled`(受け付けの問い合わせは 0 回)、有効で答えがないか受け付けない見張りなら `inbox_no_watch`。`--once` と止める頼み(起動時にある、サイクルの途中で書かれる)は `inbox_no_watch` でその場で送る。escalations.jsonl の行は ts・org_id・inbox_id・alert_id・subject・reason のちょうど 6 欄。通知が失敗すると notified の event が `failed: exit status 1: Not authorized to send Apple events`。人への道を終えられないと `ralph org inbox notify e1` を示す 1 行が出て、件は open で未通知のまま残り、ack の期限の sweep で送られる。mutant A1・A2・A4・A5・A6・A17・A18・A19・A26・A38・A44・A45・A46 が赤 |
| AC4 | `TestWatch_Escalation_AnAlertAlreadyInTheInboxGetsNoSecondItem`(`watch_test.go:1168`、9 件)、`TestRunWatchAll_AOnceRunSendsTheUnnotifiedAlertItemOfAPendingAlertUnderItsOwnID`(`watch_all_test.go:2392`、2 件) | 下の「重複した件を送る条件」 |
| AC5 | `TestWatch_Escalation_WithoutTheInboxWritesTheBannerThenNotifiesThenWritesOneLine`(`watch_test.go:1272`、5 件)、`TestWatch_Escalation_WithoutTheInboxALineThatCannotBeWrittenStillNamesTheNotificationsResult`(`:1356`、3 件)、`TestWatch_Escalation_AnAlertWithValuesTheWatchDoesNotWriteRecordsNoItem`(`:1407`、4 件)、`TestWatch_Deadman_EscalateFuncGetsItsOwnDeadlineEvenWhenTheLoopIsCancelled`(`:784`、2 件) | 受信箱がない、読めない(inbox.jsonl を dir にする)、書けない(inbox.lock を dir にする)の 3 件で、stderr の banner(原因を含む 1 行)、通知、escalations.jsonl の 1 行の順になる。順は、通知の stub の中で banner がもう出ていて行がまだないことで見る。行は ts・org_id・alert_id・subject・reason・osascript のちょうど 6 欄で、osascript は `ok`、`failed: <err>`、`skipped`。行が書けないと stderr の 2 行目が `notification=<結果>` を持つ。loop の context が切れていても、2 つの道の通知はどちらも 10 秒以内の自分の期限で呼ばれる。値の検査は下の「値の検査で今の道に落ちる形」。mutant A13〜A16・A24・A43 が赤 |
| AC6 | `TestInboxHumanTextFor_AnAlertItemGetsTheWatchText`(`escalate_test.go:1099`、9 件)、`TestNotifiableType_TakesTheInboxItemTypes`(`:931`)、`TestEscalate_RefusesTypeAlertAndWritesNothing`(`:956`)、上の `TestSendInboxItemToHuman_...` と `TestSweepInboxDeadlines_...` | ALERT の件の banner は `WATCHDOG ESCALATION` で始まり alert を逃がして出す。返り値の題は `ralph org watch`、通知の文は `org org-a raised e1 (ALERT)` で本文を含まない。期限の文も `(ALERT)`。小文字の `alert` は ALERT の件にならず `<invalid>`。`escalateTypes` は 3 つのまま。`ralph org escalate` は TYPE ALERT を断り、inbox.jsonl・inbox.lock・escalations.jsonl を作らず、banner・通知・見張りへの問い合わせもない。mutant A20〜A23・A27・A28・A42 が赤。題が osascript に届くことは打たれていない(穴 2) |
| AC7 | 前からある leader の escalate のテスト(`TestEscalate_*`、`TestNotifyInboxItem_*`、`TestSweepInboxDeadlines_*`、`TestInboxDeadlines_*`)、`TestInboxDelivery`(`escalate_test.go:982`、6 件) | 下の「leader の escalate の既存テスト」。mutant A25(leader の escalate に見張りを渡さない)は前からあるテスト 7 本で赤 |
| AC8 | `TestRenderDirectorPrompt_LeavesTheStopForAnAlertItemToTheHuman`(`internal/org/prompts_test.go:1987`、2 件)、`TestOrgWatch_HelpDescribesTheAlertItemEscalation`(`internal/cli/org_watch_test.go:1879`)、`TestOrgEscalateAndInboxHelp_DescribeTheDirectorSwitch`(`internal/cli/org_inbox_test.go:753`) | herdr と launch_cmd の両方の director の雛形が、ミッションの節で ALERT の件の 1 行を言い、バッククォートつきの TYPE ALERT は雛形の中で 1 回だけ。watch と inbox と inbox notify の help が ALERT の件を言う。mutant P1・P2・C1・C2 が赤。skill と規則の写しは /verify の範囲 |
| AC9 | なし(記録) | /verify の範囲 |
| AC10 | 1、2 | 上の表 |

## 計画の Test plan の項目

| 区分 | 項目 | テスト |
| --- | --- | --- |
| Unit | inbox_test.go の `alert_id` の書き込みと読み戻し、欄のない古い行 | `TestInboxEscalate_WritesAndFoldsTheAlertID`、`TestInboxRead_AlertIDComesOnlyFromTheEscalatedEvent` |
| Unit | 受信箱の件の TYPE の一覧と `notifiableType`(ALERT が通る、escalate は ALERT を断る) | `TestNotifiableType_TakesTheInboxItemTypes`、`TestEscalate_RefusesTypeAlertAndWritesNothing` |
| Unit | 届け方の判定の関数(Enabled と受け付けの 4 通り) | `TestInboxDelivery` の 6 行(無効 × 受け付ける・受け付けない・nil、有効 × nil・受け付けない・受け付ける)。呼び出しの回数も見る |
| Unit | ALERT の件の人への道(escalations.jsonl の行、通知の題と文、`failed:` の結果) | `TestSendInboxItemToHuman_AnAlertItemCarriesItsAlertIDAndSeat`、`TestInboxHumanTextFor_AnAlertItemGetsTheWatchText`。題は返り値の構造体でだけ見る(穴 2) |
| Integration | `evaluateCycle` で、受信箱のある Org の deadman と leader の異常が件になる(delivery の 2 通り) | now は `TestWatch_Escalation_ASeatsAlert...` が deadman と leader の両方で打つ。deadline は `TestWatch_Escalation_LeavesTheAlertItemToTheDirector...` が leader の ALERT でだけ打つ(穴 1) |
| Integration | 重複(届け済み・ack 済み・resolve 済みは送らない、未通知で open の件は今の見張りが任せられないなら同じ ID で送り、任せられるなら件の delivery が `now` でも送らずに 1 行) | `TestWatch_Escalation_AnAlertAlreadyInTheInboxGetsNoSecondItem` の 9 件 |
| Integration | 受信箱に書けないときの今の道と `osascript`(`ok`、`failed:`、`skipped`) | `TestWatch_Escalation_WithoutTheInbox...` の 2 本、8 件 |
| Integration | `RunWatchAll` の `--once`(Cycles 1)は `now` | `TestRunWatchAll_AOnceRunSendsItsEscalationAtOnceWithTheDirectorOn` |
| Integration | 長く動く見張りは `deadline` で、ack の期限を過ぎたサイクルで届き、ack したまま resolve されない件は resolve の期限を過ぎたサイクルで届く | `TestRunWatchAll_ALongRunningWatchLeavesItsEscalationToTheDirectorUntilADeadline` の 2 件 |
| Integration | 止める頼みが出たサイクルは `now` | `TestRunWatchAll_ACycleWithAStopRequestSendsItsEscalationAtOnce` の 2 件(起動時にある頼み、サイクルの途中で書かれた頼み) |
| Integration | 件を書いたあと通知の前に落ちた形から `--once` で起動すると、同じ ID で届き件は 1 つのまま | `TestRunWatchAll_AOnceRunSendsTheUnnotifiedAlertItemOfAPendingAlertUnderItsOwnID` の 2 件(件の delivery が now と deadline)。保存された status ファイルから alert が pending を外れることも見る |
| Regression | 受信箱のない Org の今の deadman のテスト(行の数、banner、1 回だけ)が通る | `testWatchOrg` は Inbox を持たないので、前からある `TestWatch_Deadman_*`(`NoActivity_EscalatesOnceAfterTimeout`、`LeaderIsAnomalySubject_EscalatesImmediately` ほか)は今の道を通る。1・2・3 で通り、mutant A13(今の道に落とさない)では前からある 6 本と書き直した `EscalateFuncGetsItsOwnDeadline...` が赤になった |
| Regression | leader の escalate のテストが期待を変えずに通る | 下の「leader の escalate の既存テスト」 |
| Regression | `TestRunWatchAll_WakesTheDirectorOnceForAnEscalatedItem` が通る | 1・2・3 で通った。mutant A25 でも赤になる |
| Edge | alert_id の座席の部分が `leader`(理由 `leader_is_anomaly_subject`) | `TestAlertIDSeat` の `org-a/leader/liveness@1`、`TestSendInboxItemToHuman_...` の 2 件目、`TestSweepInboxDeadlines_...` の e2 |
| Edge | 受信箱が読めるが書けない | `TestWatch_Escalation_WithoutTheInboxWritesTheBannerThenNotifiesThenWritesOneLine` の「an inbox that cannot be written」(inbox.lock を dir にする。読めることを確かめたうえで件が 0 であることも見る) |
| Edge | DesktopNotify が nil で darwin でない(`skipped`) | 今の道は「no notification off darwin」の 2 件(`setDesktopNotifyGOOS(t, "linux")`)。受信箱の道は leader の件の `TestEscalate_NoDesktopNotificationOffDarwinIsSkipped`(同じ `notifyBounded`)。mutant A24(継ぎ目を外す)は 3 本で赤 |
| Edge | 同じサイクルで 2 つの org が escalate する | `TestRunWatchAll_TwoOrgsEscalatingInOneCycleGetOneAlertItemEach`(director は無効、delivery now だけ) |

## 頼まれた観点ごとの確かめ

### 2 つの道で通知の失敗が `failed: <err>` になる

- 受信箱の道: `TestWatch_Escalation_SendsTheAlertItemAtOnce...` の「the desktop notification fails」が、見張りの `evaluateCycle` から送った件の notified の event の `osascript` を `failed: exit status 1: Not authorized to send Apple events` と比べる。`TestSendInboxItemToHuman_...` の 2 件目も同じ値を見る
- 受信箱に書けないときの今の道: `TestWatch_Escalation_WithoutTheInboxWritesTheBannerThenNotifiesThenWritesOneLine` の「the notification fails」が escalations.jsonl の行の `osascript` を同じ値と比べる。行が書けないときは `..._ALineThatCannotBeWrittenStillNamesTheNotificationsResult` が stderr の 2 行目の末尾 `; notification=failed: ...` を見る
- mutant A14(行から `osascript` を落とす)、A16(stderr から `notification=` を落とす)、A43(結果を常に `ok` にする)が赤

### 重複した件を送る条件(open・未通知・今の見張りの判定が now)

`escalateRecordedItem` の条件 `item.State != InboxStateOpen || item.Notified || watchDelivery != InboxDeliveryNow` の 3 つの項それぞれに、その項だけで送らない形の行がある。acked(受け付けない見張り)と resolved(director 無効)が state、already notified(director 無効)が notified、「open, to be sent at once, this watch accepts work」が今の見張りの判定。送る行は 4 つで、理由は今の見張りの判定から来る(`inbox_no_watch` 2 件、`inbox_director_disabled` 2 件)。件の delivery が now と deadline の両方に、送る行と送らない行がある。mutant A7・A8・A9(項を 1 つずつ外す)、A10(件を書いたときの delivery で決める)、A11(alert_id で探さない)、A39(理由を固定する)、A47(送る前の 1 行を消す)がすべて赤

### `--once`、止める頼み、長く動く見張りの 3 通り

`acceptsWork` は `!w.mark.Once && !w.stopRequested()`。mutant A1(Once を見ない)は `--once` の 2 本、A2(止める頼みを見ない)は止める頼みの 1 本、A3(配線しない)は長く動く見張りの 1 本で赤。A4(起動時に 1 回だけ決める)は「a stop request written during the cycle」の行で赤になり、判定がサイクルのたびに読まれることも固まっている

### ack の期限と resolve の期限

`TestRunWatchAll_ALongRunningWatch...` が `RunWatchAll` の中で両方を打ち、`TestSweepInboxDeadlines_SendsAnAlertItem...` が sweep の単位で、open の件の ack の期限、その件の resolve の期限、ack した件の resolve の期限だけを打つ。`TestWatch_Escalation_ASendThatIsNotCompleted...` は、送り終えなかった件が ack の期限で送られることを見る

### 値の検査で今の道に落ちる形

`TestWatch_Escalation_AnAlertWithValuesTheWatchDoesNotWriteRecordsNoItem` の 4 件(subject の改行、alert_id の条件の改行、別の形の alert_id、ts の改行)。どれも件を書かず、banner は 1 行で値を逃がして出し、原因の文を含み、escalations.jsonl の行は値をそのまま持つ。mutant A32〜A35(検査を 1 つずつ外す)と A37(banner で逃がさない)が赤

### leader の escalate の既存テスト

- テストファイルの差分で消えた行は 4 か所だけ: `TestEscalate_NoDesktopNotificationOffDarwinIsSkipped` の darwin の skip(GOOS の継ぎ目と偽の osascript に置き換え、期待は `skipped` のまま)、`TestEscalationRecord_WatchAndInboxLinesKeepTheirOwnID` の doc コメント、`TestWatch_Deadman_EscalateFuncGetsItsOwnDeadline...` の本体(2 つの道の subtest に書き直し)、cli の inbox の help の期待の 1 行。leader の escalate の期待は変わっていない
- base のテストファイル 8 つを HEAD のコードに重ねて打った(6)。cli で落ちたのは `TestOrgEscalateAndInboxHelp_DescribeTheDirectorSwitch` の 1 本だけで、理由は「deferred for one escalate left to the director」の文を help が「or the cross-org watch」に直したこと(self-review F2 の直し)。org は base の 1,153 本のうち落ちたのが `TestWatch_Deadman_EscalateFuncGetsItsOwnDeadlineEvenWhenTheLoopIsCancelled` の 1 本だけで(`EscalateFunc called 0 times; want 1`)、理由は `newWatchAllFixture` が受信箱を持つので、leader の escalation が `WatchAllParams.Escalate` でなく `Org.DesktopNotify` で通知されるようになったこと。HEAD はこのテストを 2 つの道の subtest に書き直している。base の `TestEscalate_*`・`TestNotifyInboxItem_*`・`TestSweepInboxDeadlines_*`・`TestInboxDeadlines_*`・`TestInbox*` と、ほかの `TestRunWatchAll_*`・`TestWatch_Deadman_*` は、base の期待のまま HEAD のコードで通った。skip の 1 件は base の `TestEscalate_NoDesktopNotificationOffDarwinIsSkipped` が darwin で飛ばすもので、HEAD では継ぎ目を使って darwin でも打つ

## 本物を打たないことの確かめ

- fixture と継ぎ目: `newWatchInboxFixture`(`watch_test.go`)は `Org.DesktopNotify` と `watchRun.escalateFn` を記録するだけの stub にし、herdr は `fakeWatchHerdr`。`newWatchAllFixture`(`watch_all_test.go:123`)は `fakeWatchHerdr`・`fakeWatchAgmsg`・stub の `DesktopNotify`、`f.params` は stub の `Escalate` と `GitStatus` を渡す。`escalateFn` か `DesktopNotify` を nil にする 3 か所(`escalate_test.go:330`、`watch_test.go:1317`、`:1380`)は先に `setDesktopNotifyGOOS(t, "linux")` を呼ぶので `notifyBounded` が osascript を呼ばない。`escalate_test.go:322` はさらに PATH を失敗する偽の osascript だけにする。cli は `main_test.go:117` が `orgDesktopNotifyOverride` を何もしない stub に固定する。gh と network を打つ新しい呼び出しは差分にない
- canary: `herdr`・`agmsg`・`osascript`・`open`・`tmux`・`gh`・`orca`・`claude`・`codex`・`curl`・`wget`・`nc`・`ssh` を、呼ばれたら引数を記録して exit 97 する script にした dir を PATH の先頭に置き、`HOME` を空の scratch の dir にした(`GOCACHE`・`GOMODCACHE`・`GOPATH` は元の値、`GOPROXY=off`、`GOFLAGS` は空)。3、5、6、7 はこの環境で打ち、canary の記録は 0 行だった
- canary の感度: mutant A24(`desktopNotifyGOOS` の代わりに `runtime.GOOS` を読む)では、canary の osascript が `display notification "ralph org watch: escalation for org org-a (leader_is_anomaly_subject)" with title "ralph org watch"` を 2 回記録し、テストは `failed: exit status 97` で赤になった。継ぎ目が外れれば canary が気づく

## Mutant

54 本。A は `internal/org` の Go、P は `internal/org/prompts/director.md`(go:embed)、C は `internal/cli` の help。`-overlay` で 1 本ずつ差し替え、3 の集合(C は `TestOrgWatch_Help|TestOrgEscalateAndInboxHelp`)を打った。A25 と A45 は最初の形が未使用の変数で build に落ちたので、`_ = acceptsWork` と `&& false` の形に直して打ち直した。

| ID | 変更 | 結果 | 赤にしたテスト(主なもの) |
| --- | --- | --- | --- |
| A1 | `acceptsWork` が Once を見ない | 赤 | `TestRunWatchAll_AOnceRun...` の 2 本 |
| A2 | `acceptsWork` が止める頼みを見ない | 赤 | `TestRunWatchAll_ACycleWithAStopRequest...` |
| A3 | `w.run.acceptsWork` を配線しない | 赤 | `TestRunWatchAll_ALongRunningWatch...` |
| A4 | 受け付けを起動時に 1 回だけ決める | 赤 | `TestRunWatchAll_ACycleWithAStopRequest...` |
| A5 | `inboxDelivery` が director の switch を見ない | 赤 | `TestInboxDelivery` ほか 7 本 |
| A6 | nil の `acceptsWork` を受け付けとみなす | 赤 | `TestInboxDelivery`、`TestEscalate_DeliveryFollowsTheDirectorSwitchAndTheWatch` ほか |
| A7〜A9 | 重複の条件から state、notified、今の判定を 1 つずつ外す | 3 本とも赤 | `TestWatch_Escalation_AnAlertAlreadyInTheInboxGetsNoSecondItem` |
| A10 | 重複を件の delivery で決める | 赤 | 同上と `TestRunWatchAll_AOnceRunSendsTheUnnotifiedAlertItem...` |
| A11 | alert_id で探さない | 赤 | 同上 |
| A12 | deadline の件もその場で送る | 赤 | `TestWatch_Escalation_LeavesTheAlertItem...`、`TestRunWatchAll_ALongRunningWatch...` |
| A13 | 受信箱に書けないとき今の道に落とさない | 赤 | 前からある `TestWatch_Deadman_*` 6 本、書き直した 1 本、新しい 3 本 |
| A14 | 今の道の行から `osascript` を落とす | 赤 | `TestWatch_Escalation_WithoutTheInboxWrites...`、`..._AnAlertWithValues...` |
| A15 | 今の道の banner を通知のあとに書く | 赤 | `TestWatch_Escalation_WithoutTheInboxWrites...` |
| A16 | 行が書けないときの stderr から `notification=` を落とす | 赤 | `..._ALineThatCannotBeWritten...` |
| A17・A18・A19 | ALERT の行の subject を leader のまま、alert_id を落とす、別の形の alert_id でも leader | 3 本とも赤 | `TestSendInboxItemToHuman_...` ほか |
| A20・A21・A22 | ALERT の文の分岐を消す、題を escalate のものにする、`notifiableType` が `escalateTypes` を読む | 3 本とも赤 | `TestInboxHumanTextFor_...` ほか |
| A23 | `escalateTypes` に ALERT を足す | 赤 | `TestEscalate_RefusesTypeAlertAndWritesNothing`、`TestNotifiableType_...` |
| A24 | `notifyBounded` が `runtime.GOOS` を読む | 赤 | 3 本(canary の感度の確かめを兼ねる) |
| A25 | leader の escalate に見張りを渡さない(AC7) | 赤 | 前からある 7 本 |
| A26 | 見張りが director の switch を見ない | 赤 | 3 本 |
| A27・A28 | 期限の送信、`inbox notify` が alert_id を渡さない | 2 本とも赤 | `TestSweepInboxDeadlines_SendsAnAlertItem...`、`TestSendInboxItemToHuman_...` |
| A29・A30・A31 | fold が alert_id を落とす、書かない、あとの event で上書きする | 3 本とも赤 | inbox の 2 本ほか |
| A32〜A35 | 値の検査を 1 つずつ外す(subject、alert_id の制御文字、alert_id の形、ts) | 4 本とも赤 | `TestWatch_Escalation_AnAlertWithValues...` |
| A36 | 本文の ALERT_ID と ALERT_TS を入れ替える | 赤 | `TestWatch_Escalation_ASeatsAlert...` |
| A37 | 見張りの行の頭で alert と subject を逃がさない | 赤 | `..._AnAlertWithValues...` |
| A38 | 送り終えなかったときの `inbox notify` の 1 行を出さない | 赤 | `..._ASendThatIsNotCompleted...` |
| A39 | 重複した件を常に `inbox_no_watch` で送る | 赤 | `..._AnAlertAlreadyInTheInbox...` |
| A40 | 本文の protocol の検査を飛ばす | 生き残り | なし(下) |
| A41 | deadline の 1 行から resolve の期限を落とす | 赤 | `..._LeavesTheAlertItem...` |
| A42 | ALERT の banner で alert_id を逃がさない | 赤 | `TestInboxHumanTextFor_...` |
| A43 | 今の道の通知の結果を常に `ok` にする | 赤 | 今の道の 2 本 |
| A44・A45・A46 | `alertIDSeat` が 3 つより多い区切り、数字でない時刻、空の条件を通す | 3 本とも赤 | `TestAlertIDSeat` |
| A47 | 重複した件を送る前の 1 行を消す | 赤 | `..._AnAlertAlreadyInTheInbox...` |
| A48 | deadline の 1 行の `inbox show` から `--state-dir` を落とす | 赤 | `..._LeavesTheAlertItem...` |
| A49 | `realEscalate` の osascript の題を `ralph org escalate` にする | 生き残り | なし(穴 2) |
| A50 | 受信箱の osascript の道が文の題を使わない | 生き残り | なし(穴 2) |
| P1・P2 | 雛形の「止めない」の文を消す、バッククォートつきの TYPE ALERT を 2 回にする | 2 本とも赤 | `TestRenderDirectorPrompt_LeavesTheStop...` |
| C1・C2 | watch の help から `--once` を消す、inbox の help から TYPE ALERT を消す | 2 本とも赤 | `TestOrgWatch_HelpDescribesTheAlertItemEscalation`、`TestOrgEscalateAndInboxHelp_...` |

- 生き残りの 3 本は org パッケージ全体(`-run '.*'`、canary の PATH、3 本を同時に)でも打ち直し、3 本とも生き残った(rc 0、`--- FAIL` 0 件、A40 496 秒、A49 494 秒、A50 493 秒)。canary の記録は 0 件のままだった
- A40 は不変条件で同値と判断した。`protocol.Validate` が落ちるのは、1 行目が TYPE でない、TYPE が表にない、TASK_ID が要る TYPE、本文(最初の空行より後)が 2,000 字を超える、の 4 つだけである。`watchEscalationBody` の 1 行目は `TYPE: ALERT` で、ALERT は表にあり TASK_ID は要らない。見出しの鍵は固定なので `splitHeaderLine` はどの値でも見出しとして読み、値に改行などの制御文字がないことは先に `checkWatchEscalationValues` が確かめるので、本文は固定の 1 行(100 字ほど)のままになる。この検査は防御のための写しで、テストで赤にする入力はない
- A49・A50 は打たれていない(穴 2)。PATH に引数を記録する偽の osascript だけを置き、`setDesktopNotifyGOOS(t, "darwin")` にした使い捨てのテスト 2 本を `-overlay` で足すと、HEAD では 2 本とも通り、A49 では今の道の 1 本、A50 では受信箱の道の 1 本が `with title "ralph org escalate"` で落ちた(8)。既存の `TestEscalate_NoDesktopNotificationOffDarwinIsSkipped` と同じ道具で塞げる

## Coverage

- Statement: `internal/org` 94.9%(7、canary の PATH でパッケージ全体)。前回の記録(`docs/reports/test-2026-10-22-split-preamble.md`)は 94.8%
- Branch: 計っていない(Go の標準の道具にない)
- Function: 差分で足した関数と変えた関数(`go tool cover -func`)。100% は `inboxDelivery`、`escalateDelivery`、`sendInboxItemToHuman`、`inboxHumanTextFor`、`notifiableType`、`alertIDSeat`、`acceptsWork`、`escalateAlert`、`checkWatchEscalationValues`、`escalateRecordedItem`、`sendAlertItemToHuman`、`escalationHead`、`watchEscalationBody`、`escalateWithoutInbox`、`notifyWithoutInbox`、`SweepInboxDeadlines`、inbox の fold の `apply`。100% でないものは `realEscalate` 0.0%、`inboxDesktopNotify` 50.0%、`notifyBounded` 88.9%、`escalateToInbox` 95.8%、inbox の `Escalate` 95.7%、escalate の `Escalate` 95.2%、`NotifyInboxItem` 86.7%、`newWatchAllRun` 78.6%、`alertIDConditionType` 75.0%
- Notes: `realEscalate`、`inboxDesktopNotify` の osascript の closure、`notifyBounded` の `notify = onDarwin` の行は、darwin で通知の手段が nil のときだけ通る道で、テストはどれも継ぎ目で避けるので打たれない(穴 2 と同じ所)。`escalateToInbox` の残りは `protocol.ValidateText` が落ちたときの return で、mutant A40 と同じく到達しない。残りの関数の未到達の行は、この差分が変えていない行(inbox と escalate の `Escalate` と `NotifyInboxItem` の誤りの return、`newWatchAllRun` の既定値の分岐、`alertIDConditionType` の形の外れ)。未到達の block を `git diff -U0` の足した行の範囲と突き合わせて確かめた

## Failure analysis

| Test | Error | Root cause | Proposed fix |
| --- | --- | --- | --- |
| なし(1〜4 は落ちたテストがない) | - | - | - |

6 で base のテストが HEAD のコードで落ちるのは、この差分が意図して変えた期待の 2 本だけである(cli の help の文 1 本と、org の通知の継ぎ目 1 本。上の「leader の escalate の既存テスト」)。

## Regression checks

| Previously broken behavior | Status | Evidence |
| --- | --- | --- |
| F6: 見張りの escalation 30 件が director を通らずに人へ直接届いた | 直った(director が有効で長く動く見張りなら件は deadline で記録だけされ、通知も行もない) | `TestWatch_Escalation_LeavesTheAlertItemToTheDirector...`、`TestRunWatchAll_ALongRunningWatch...`。mutant A3・A12 が赤 |
| F13: 見張りの escalation の道が通知の結果を捨てる | 直った(受信箱の道は notified の event、今の道は行の `osascript` か stderr の `notification=`) | 上の「2 つの道で通知の失敗が `failed: <err>` になる」。mutant A14・A16・A43 が赤 |
| 受信箱のない Org の deadman の escalation(行の数、banner、1 回だけ) | 変わらない | 前からある `TestWatch_Deadman_*` が 1〜3 で通る |
| leader の escalate の届け方 | 変わらない | 前からある escalate・notify・期限のテストが 1〜3 で通る。mutant A25 が前からある 7 本で赤 |
| 見張りの通知が止める頼みや SIGINT で切られない(self-review L2 of org-cross-watch) | 2 つの道とも保たれる | `TestWatch_Deadman_EscalateFuncGetsItsOwnDeadline...` の 2 件 |
| director の wake は escalate した件に 1 回だけ | 変わらない | `TestRunWatchAll_WakesTheDirectorOnceForAnEscalatedItem` が 1〜3 で通る |

## Test gaps

1. 座席の ALERT(`deadman_timeout`)が delivery `deadline` で記録される形を、`evaluateCycle` と `RunWatchAll` のどちらも打っていない。director が有効なときの届け方を見張りから打つテスト(deadline になる `TestWatch_Escalation_LeavesTheAlertItemToTheDirector...` と `TestRunWatchAll_ALongRunningWatch...`、now になる止める頼みと `--once` の 3 本)はどれも leader の ALERT を使う。座席の ALERT の deadline の件の期限は、sweep の単位(`TestSweepInboxDeadlines_SendsAnAlertItem...`、件を直接書く)でだけ打たれる。届け方の判定は理由を読まないので、壊れる見込みは低い。Test plan の「deadman と leader の異常が件になること(delivery の 2 通り)」の deadman × deadline の組み合わせが欠けている
2. ALERT の件と今の道のデスクトップ通知の題が `ralph org watch` であること(AC6)は、`inboxHumanTextFor` の返り値の `title` でだけ固まっていて、osascript に渡る題は打たれていない。mutant A49(`realEscalate` の題)と A50(`inboxDesktopNotify` が `text.title` を使わない)が生き残る。テストの stub は通知の文だけを受け取るためである。偽の osascript と `setDesktopNotifyGOOS(t, "darwin")` の使い捨てのテストで両方とも赤にできることを確かめた(Mutant の節)
3. 同じサイクルで 2 つの org が escalate する形は、director が無効(delivery `now`)でだけ打たれる。2 つの件が deadline で記録され、それぞれの期限で届く形はない。alert_id は org を含むので重複の判定は交わらず、壊れる見込みは低い

計画の Test plan の外で気づいた点(穴としては数えない):

- cobra の `ralph org watch --all --once` を通して ALERT の件ができるテストは cli にない。CLI が見張りと同じ Org の `Inbox` と `EscalationsPath` を渡すことは、期限の送信を見る既存の cli テスト(`internal/cli/org_watch_test.go:252` 付近、`--once` の期限の送信が `EscalationsPathIn(stateDir)` に行を書く)で間接に固まっている。`WatchAllParams.StatusDir` の doc が呼び出し側に求める「`Org.EscalationsPath` を `EscalationsPathIn(StatusDir)` にする」ことを、ALERT の件で直接見るテストはない
- push で登録した director の wake は、サイクルの中で pulse より前に走るので、deadline で記録された ALERT の件の wake は次のサイクルになる。ALERT の件の wake を見るテストはない(`WakeDirector` は TYPE を読まない)

## Verdict

- Pass: pass。`GOFLAGS='-timeout=30m -count=1' ./scripts/run-test.sh` が rc 0(851 秒、shell 44 本と Go 8 パッケージ)、`GOTOOLCHAIN=go1.25.8 go test -count=1 -timeout 30m ./internal/org/... ./internal/cli/... ./internal/config/...` が rc 0(391 秒)。Test plan の項目はすべて対応するテストがあり、欠けは穴 1〜3 の組み合わせと題の確かめだけ。mutant は 54 本中 51 本が赤で、生き残りは同値の 1 本と穴 2 の 2 本
- Fail: なし
- Blocked: なし

## Insight event

依頼がこの報告のほかを変えないとしたので、追記していない。追記するなら:

```
./scripts/insights-append.sh --slug watch-escalation-inbox --flow standard --phase test \
  --verdict pass --cycle auto --source skill || true
```
