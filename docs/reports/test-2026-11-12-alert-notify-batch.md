# Test report: alert-notify-batch

- Date: 2026-11-12(run-test.sh は 15:23:27 から 15:44:34 JST、追加の回は 16:34 から 17:11 JST)
- Plan: `docs/plans/active/2026-11-12-alert-notify-batch.md`(`## Test plan`、AC1〜AC10、AC1b、AC1c)
- Tester: tester サブエージェント(`/test`、pipeline の cycle 1。`cycle-count.json` は `{"cycle": 1}`)。前の tester の回は `./scripts/run-test.sh` が終わったあと API のネットワークエラーで止まった。この回はその続きで、HEAD が同じ `34becddd` であることを確かめ、run-test.sh は打ち直さずに前の回の出力を使った
- Scope: `git diff 94e61c38...HEAD`(HEAD `34becddd`、37 ファイル、+2098/-93)。コードの commit は S1 `01fcf3fb`、S2 `85889ad9`、S3 `9175cae1`、S4 `5752476a`(文書)、S5 `1f2197dc`(self-review の L1〜L4)。run-test.sh は `templates/base/ralph.toml` が言語に分類されないので full に落ち、golang を選んだ(`Language scope: full fallback (unclassified:templates/base/ralph.toml)`)。worktree の production code とテストには手を入れていない。変異と probe は `go test -overlay` で当て、前後の `git status --porcelain --ignored` は同じだった
- Evidence: `docs/evidence/test-2026-11-12-alert-notify-batch.log`(`.gitignore` の `docs/evidence/*.log` に当たるのでコミットしない)。先頭から run-test.sh の全出力(wrapper のログ `docs/evidence/verify-2026-11-12-062327.log` に、最後の `Evidence saved to:` の 1 行を足したもの)、その後ろにこの回で足した `-race` の回、coverage、変異と probe の定義と結果

## Test execution

| Suite / Command | Tests | Passed | Failed | Skipped | Duration |
| --- | --- | --- | --- | --- | --- |
| `./scripts/run-test.sh`(rc 0、`==> All verifiers passed.`。HEAD `34becddd`、前の tester の回) | shell 45 本 + Go 10 パッケージ | shell 45 本 OK(PASS を含む行 5,195)、Go 10 パッケージ ok | 0(`--- FAIL` の行 0) | 2 行 | 1,267 秒 |
| └ Go: `internal/org` | 上位 1,658 本 | ok | 0 | — | 700.6 秒 |
| └ Go: `internal/cli` | 上位 1,087 本 | ok | 0 | — | 517.1 秒 |
| └ Go: `internal/config`(上位 115 本)、`cmd/ralph`、`internal/gitroot`、`internal/insights`、`internal/org/driver`、`internal/org/protocol`、`internal/scaffold`、`internal/upgrade` | — | ok。`(cached)` は 0 | 0 | — | 12.4、0.7、6.0、1.7、3.0、2.0、4.0、3.2 秒 |
| `-race -count=3 -v`、org: この diff が足したか変えたテストと、名前が同じ道を通るもの(下の正規表現)の 94 本 | 282 回 | 282 | 0 | 0 | 21.5 秒。`DATA RACE` 0 |
| `-race -count=3 -v`、cli: `^(TestOrgWaitInbox\|TestOrgWait_\|TestOrgWaitDirector_\|TestOrgWatch_Help\|TestConfigGet_)` の 55 本 | 165 回 | 165 | 0 | 0 | 30.4 秒。`DATA RACE` 0 |
| `-race -count=3 -v`、`internal/config` 全体 | 上位 115 本 × 3(subtest 込みの PASS 行 345) | 345 | 0 | 0 | 43.4 秒。`DATA RACE` 0 |
| coverage: `internal/config` 全体、cli と org は名前で絞った回(下の節) | config 上位 115 本、cli 絞り込み、org 324 本 | ok | 0 | — | 27.8 秒、15.2 秒、75.1 秒 |
| 変異: 掃き出しの flush を周期の後ろの段へ動かす 4 個(M0〜M3) | 4 | 死んだのは感度の対照の M3 だけ。M1 は org 全体と cli の 230 本でも生き残った(下の節) | — | — | 絞った回は 1 個 86〜93 秒、M1 の org 全体は 1,414.9 秒(load 30〜150 の下)、cli は 120.5 秒 |
| probe: AC1b の director の半分を固定するテスト(overlay で足した 1 本、コミットしない) | 1 本 × HEAD と M0〜M3 | HEAD と M0 で PASS、M1〜M3 で FAIL | — | — | 1 回 1 秒前後 |

org の `-race` の回の正規表現は `^(TestSweepInboxDeadlines_|TestWaitInbox_|TestWaitInboxForDirector_|TestSendInboxItemToHuman_|TestNotifyInboxItem_|TestParseInboxWaitTypes_|TestDirectorWaitTimeoutDetail$|TestAlertNotifyText_|TestFlushAlertNotifyBatch_|TestWatch_Deadman_|TestWatch_EscalateAlert_|TestRunWatchAll_SendsTheSweepsAlertNotificationBeforeThePulse$|TestRunWatchAll_ALongRunningWatchLeavesASeatsDeadmanEscalationToTheDirectorUntilTheAckDeadline$|TestRunWatchAll_TwoOrgsEscalatingInOneCycleAreEachLeftToTheDirectorUntilTheirAckDeadline$)` である。`alert_notify_test.go` の 12 本、`escalate_test.go` と `director_wait_test.go` で足した 9 本、既存のテストで変えた 3 本(期待を変えた `deadline_test.go` と `watch_all_test.go` の 1 本ずつ、`escalateAlert` の呼び出しに nil を足しただけの `watch_test.go` の 1 本)、変わった helper `runWaitInbox` を使う `TestWaitInbox_` のすべてを含む。対象の本は `git diff -U0` の足した行が入る関数で選んだ。

上位の本数は `go test -list .` で数えた。Skipped の 2 行は前の PR の回と同じで、`tests/test-plan-visual.sh` の browser detection と、`tests/test-secret-scan-branch.sh` の git 2.41 より古い本物の git の case である。どちらも手元の環境によるもので、この diff と関係がない。

追加の回のあいだ、ほかのセッションの Swift のビルドと codex で 1 分の load が 12 から 151 まで上がった。落ちたテストは無く、打ち直しはしていない。

## 受け入れ条件とテスト

どのテストも run-test.sh と上の `-race -count=3` の回で PASS した。

| AC | テスト(ファイル) | 備考 |
| --- | --- | --- |
| AC1 掃き出し 1 回と org 1 つの deadman の段で通知は 1 回、文に件数・最も古い件・その org。各件の `notified` に同じ結果、行と banner は件ごと | `TestSweepInboxDeadlines_SendsTheAlertItemsOfOneSweepInOneNotification`、`TestWatch_Deadman_SendsTheAlertItemsOfOneOrgsStepInOneNotification`(`alert_notify_test.go`)。期待を変えた `TestSweepInboxDeadlines_SendsAnAlertItemWithItsAlertIDAndSeat`(`deadline_test.go`)と `TestRunWatchAll_TwoOrgsEscalatingInOneCycleAreEachLeftToTheDirectorUntilTheirAckDeadline`(`watch_all_test.go`) | 期待を変えた 2 本は、件ごとの通知の列を「2 ALERT items; the oldest is …」の 1 回に置き換えた。数も文も完全一致で比べるので、assert は弱まっていない |
| AC1b 掃き出しのまとめた送りは director の確かめと座席の見回りより前に終わる | 座席の見回りの半分: `TestRunWatchAll_SendsTheSweepsAlertNotificationBeforeThePulse` | director の確かめの半分を固定するテストは無い。flush を `checkDirector` のあとへ動かした M1 は org 全体でも生き残った(Test gaps 1)。overlay で足した probe は HEAD で通り M1 を落とした |
| AC1c `notified` の追記に失敗した件は送りの失敗として記憶され、ack の期限の長さのあいだ送り直されない | `TestSweepInboxDeadlines_AnAlertItemWhoseNotifiedEventIsNotWrittenIsHeldBack`(12 分で held back、21 分で e2 だけを 1 件の文で送り直し、22 分で何も送らない。e1 と e3 は 1 回だけ)、deadman の段は `TestWatch_Deadman_ANotifiedEventNotWrittenAtTheEndOfTheStepNamesInboxNotify` | 複数の周期を、掃き出しを時刻を変えて 3 回呼ぶ形で確かめている |
| AC2 1 件だけなら今までと同じ文 | `TestAlertNotifyText_NamesTheCountTheOldestAndItsOrg`(1 件の 2 形)、AC1c のテストの 21 分の送り直し、`TestSendInboxItemToHuman_AnAlertItemOfABatchNeverFlushedIsSentAtTheAckDeadline` | |
| AC3 `off` では通知なし、`osascript` は `off`、banner と行は出て、期限で送り直されない | `TestSweepInboxDeadlines_AlertNotifyOffSendsNoNotificationForTheAlertItems`(次の掃き出しと、記憶の無い立て直しの掃き出しで 0 件)、`TestWatch_Deadman_AlertNotifyOffSendsNoNotification` | |
| AC4 leader の件、`inbox notify`、予備の道は 1 件ずつで `alert_notify` を読まない | `TestNotifyInboxItem_AlertNotifyOffDoesNotStopTheSendsOutsideTheWatchsSteps`。掃き出しの中の QUESTION は AC1 と AC3 の掃き出しのテストで自分の番に通知される | 周期の初めの再送(`reconcilePendingAlertsWithInbox`、nil の batch)を `off` で通すテストは無い(Test gaps 3) |
| AC5 送りの前に止まった見張りの件は `notified` を持たず、次の見張りの ack の期限で送られる | `TestSendInboxItemToHuman_AnAlertItemOfABatchNeverFlushedIsSentAtTheAckDeadline` | |
| AC6 既定は `"batch"`、ほかの値は検証で断る、テンプレートと既定値の一致 | `TestLoad_OrgWatchdogAlertNotify`、`TestDefault_OrgWatchdog`、`TestLoad_OrgWatchdogRoundTrip`、`TestLoad_OrgWatchdogRejects`、`TestDefaultsLockStep`、`TestOrgTemplateDrift_EachKeyIsComparedOnBothSides`(`internal/config`)、`TestConfigGet_NoRalphTomlPrintsTheDefaults`、`TestConfigGet_PrintsTheValueRalphTomlSets`(`internal/cli/config_test.go`) | |
| AC7 `--types` は ALERT だけのあいだ返らず、指定の種類が届けばその件だけを出す。既定は今と同じ。知らない種類と空の項目は待つ前に断る | `TestOrgWaitInbox_TypesPrintOnlyTheListedItems`、`TestOrgWaitInbox_TypesKeepWaitingPastAnAlertItem`、`TestOrgWait_TypesIsRefusedWithoutInboxOrWithABadValue`(`internal/cli/org_inbox_test.go`)、`TestParseInboxWaitTypes_AcceptsTheInboxTypes`、`TestParseInboxWaitTypes_RefusesAnythingElseAndListsTheAcceptedValues`、`TestWaitInbox_TypesKeepWaitingWhileOnlyAnAlertIsOpen`、`TestWaitInbox_TypesReturnOnlyTheListedItemThatArrives`、`TestWaitInbox_TypesFilterTheOpenItemsInIDOrder`(`escalate_test.go`) | 既定(`--types` なし)は、`runWaitInbox` を通る既存の `TestWaitInbox_` と CLI の `wait --inbox` のテストが今までどおり通ることで確かめた |
| AC8 `--director` と `--types` で、終了コード 4 の判定と印の件数は指定の種類だけで数える | `TestOrgWaitDirector_ABadTypesValueIsRefusedBeforeTheRegistrationCheck`、`TestOrgWaitDirector_TypesPrintAndCountOnlyTheListedItems`、`TestOrgWaitDirector_AnAlertItemLeftOutByTypesDoesNotHoldBackExit4`(`internal/cli/org_wait_director_test.go`)、`TestWaitInboxForDirector_TypesLeaveAnOpenItemOfAnotherTypeOutOfTheWatchEnd`、`TestWaitInboxForDirector_TypesCountOnlyTheListedItemsInTheMark`、`TestWaitInboxForDirector_TypesTimeOutWhileOnlyAnAlertIsOpen`、`TestDirectorWaitTimeoutDetail`(`director_wait_test.go`) | |
| AC9 director skill と org skill の 4 つの写し | テストで固定する文は無い。写しの同期は run-test.sh の Go の回の `director_skill_sync_test.go` ほかが通った | 文の中身は /verify が読んで確かめた(Test gaps 4) |
| AC10 help、仕様の注、tech-debt の行 | `TestOrgWatch_HelpDescribesTheAlertNotifyBatch`(`org_watch_test.go`)、`TestOrgWait_HelpDescribesTypes`(`org_inbox_test.go`) | 仕様と tech-debt は文書で、テストは無い |

計画の Test plan の Edge cases: 掃き出しの途中で resolve された件は `TestSweepInboxDeadlines_AnAlertItemResolvedBeforeTheEndOfTheSweepIsStillSent`、`notified` の追記の失敗は AC1c の 2 本、識別子でない org の値は `TestAlertNotifyText_NamesTheCountTheOldestAndItsOrg`、空の batch と flush 後の空は `TestFlushAlertNotifyBatch_SendsNothingForAnEmptyBatchAndEmptiesTheBatch`。送りの失敗の `failed: <err>` は AC1 の掃き出しのテストが `ok` と並べて見る。

## Coverage

- Statement: この diff が変えた Go のコード(テストを除く)の、文を持つ行 140 行のうち 140 行を通った(`git diff -U0 94e61c38...HEAD` の足した行と cover profile の block を突き合わせた。`internal/org` の 6 ファイル 96/96、`internal/cli` の 3 ファイル 40/40、`internal/config` 4/4)。org と cli は名前で絞った回なので、パッケージ全体の数字(org 53.4%、cli 10.4%)は意味を持たない。config は全体で 96.7%
- Branch: Go の標準ツールに branch coverage は無いので測っていない
- Function: 新しい関数と変えた関数はすべて通った。`alert_notify.go` の 4 つ、`SweepInboxDeadlines`、`sendInboxItemToHuman`、`openInboxItems`、`ParseInboxWaitTypes`、`inboxWaitTypesEntryText`、`directorWaitTimeoutDetail`、`checkDeadman`、`flushDeadmanAlertNotifications`、`escalateAlert`、`escalateRecordedItem`、`sendAlertItemToHuman`、`alertSendNotCompleted` は 100%。`WaitInbox` 95.7%、`escalateToInbox` 93.8%、`cycle` 88.9%、`WaitInboxForDirector` 87.5% の残りは、この diff が変えていない行で、絞った回では通らなかった。store や state dir が無いときの guard、既定の poll 間隔、protocol の検証の失敗、再発の記録の失敗、`cycle()` が `sweep.Failed` を 1 件ずつ書く行(`watch.go:759`)である。最後の行は、掃き出しのまとめた送りで追記に失敗した件(AC1c)が `RunWatchAll` の周期の中で通る行だが、AC1c のテストは掃き出しを直接呼ぶので通らない
- Notes: 行はすべて通るが、行を通ることは順を固定することではない。AC1b の director の確かめの半分は、行の coverage では見えない穴で、変異で見つけた(Test gaps 1)。`flushAlertNotifyBatch` の `lineWritten` が false の件を飛ばす行も通っている

## 変異と probe

flush の位置だけを動かす。`deadline.go` の `SweepInboxDeadlines` を、flush の関数を受け取る `sweepInboxDeadlinesMut` に包み(`SweepInboxDeadlines` 自身は今までどおり `o.flushAlertNotifyBatch` を渡すので、掃き出しを直接呼ぶテストの挙動は変わらない)、`watch.go` の `cycle()` だけが batch を預かる関数を渡して、預かった batch を下の位置で flush する。その flush は、追記に失敗した件を今と同じ鍵で記憶し、今と同じ `watchdog: inbox item … not sent` の行を書く(D2b を保つ)。定義は `/private/tmp` の scratch の `mk.py` で作り、evidence に写した。

| 変異 | flush の位置 | 結果 |
| --- | --- | --- |
| M0(対照) | 掃き出しの直後、`checkDirector` の前 | 絞った回(`-run 'RunWatchAll\|Sweep\|Director\|Alert\|Deadman\|Deadline'`、468 本)で 0 FAIL。harness が挙動を変えないことの確かめ |
| M1 | `checkDirector` のあと、`wake` の前 | 絞った回で 0 FAIL。org 全体(上位 1,658 本)でも ok で 0 FAIL(1,414.9 秒)。cli の `-run 'Watch\|Director'`(230 本)も ok。生き残り |
| M2 | `wake` のあと、`pulse` の前 | 絞った回で 0 FAIL(情報として当てた) |
| M3(感度の対照) | `pulse` のあと、印の書き込みの前 | 絞った回で `TestRunWatchAll_SendsTheSweepsAlertNotificationBeforeThePulse` だけが落ちた(`at the first herdr call: notifications [], unnotified items [e1 e2]; want ["2 ALERT items; the oldest is e1 of org org-a"] and none`) |

既存の順のテスト `TestRunWatchAll_ChecksTheDirectorAfterTheDeadlinesAndBeforeTheWakeAndThePulse` と `TestRunWatchAll_SweepsTheDeadlinesThenWakesTheDirectorBeforeThePulse` は、director の確かめや wake の時点の `escalations.jsonl` を写して比べる。M1 と M2 でも行は掃き出しの中で書かれ、件は `escalateInboxAt` が作る QUESTION なので、どちらも落ちない。

probe(`internal/org/zz_probe_ac1b_director_test.go` を overlay で足した 1 本。コミットしない): `headlessInHerdr` で headless の director を登録し、ack の期限を過ぎた ALERT の件を 2 つ置き、director の確かめの herdr の `agent_get` の時点で通知の列と `Notified` を読む。HEAD と M0 で PASS(HEAD では `-race -count=5` でも PASS)、M1・M2・M3 で `at the director check: notifications [], unnotified items [e1 e2]; want ["2 ALERT items; the oldest is e1 of org org-a"] and none` で落ちた。全文は evidence にある。

## Failure analysis

| Test | Error | Root cause | Proposed fix |
| --- | --- | --- | --- |
| なし | — | — | — |

run-test.sh、`-race` の回、coverage の回で落ちたテストは無い。変異の FAIL は上の節の意図した落ち方である。

## Regression checks

| Previously broken behavior | Status | Evidence |
| --- | --- | --- |
| 掃き出しの ALERT の件の送りの記録(`escalations.jsonl` の行、alert_id、座席)と ack・resolve の期限 | Pass | `deadline_test.go` の 15 本(期待を変えた 1 本は通知の数と文だけ)、run-test.sh の org |
| 見張りの deadman の escalate と、受信箱への記録・再発(#307a、PR #314) | Pass | `watch_test.go`、`watch_all_test.go`、`watch_recurrence` の各テスト。`escalateAlert` の呼び出しを変えた `TestWatch_EscalateAlert_PrunesAt101stEscalation` は nil の batch を渡すだけ |
| `wait --inbox` と `wait --inbox --director` の `--types` なしの挙動 | Pass | 既存の `TestWaitInbox_`、`TestWaitInboxForDirector_`、`TestOrgWaitInbox_`、`TestOrgWaitDirector_` |
| `ralph config get` の既存のキー | Pass | `TestConfigGet_*`(行を足しただけ) |

## Test gaps

1. AC1b の director の確かめの半分を固定するテストが無い。flush を `checkDirector` のあとへ動かした M1 は org 全体と cli の 230 本で生き残り、`wake` のあとへ動かした M2 も絞った回で生き残った。いちばん小さい足し方は、上の probe をそのまま `alert_notify_test.go` に足すこと(`headlessInHerdr` と `recordAlertItemAt` と `alertItems` を使う 40 行ほど)。HEAD で通り、M1・M2・M3 を落とすことを確かめた。/verify の Coverage gaps 1 と同じ穴である
2. 計画の Test plan の「掃き出しで 3 件、deadman の段で 2 件を送る周期で、送りが 2 回」は、1 つの `RunWatchAll` の周期では確かめていない。掃き出しのテストと、`checkDeadman` を直接 2 回呼ぶ deadman のテストに分かれている。1 周期で通す形の変異(たとえば batch を周期で共有する形)は、今の構造では当てにくいので当てていない
3. 周期の初めの再送(`reconcilePendingAlertsWithInbox`)が `alert_notify = "off"` でも通知することを固定するテストは無い。nil の batch を渡す 1 か所(`watch_recurrence.go`)で、AC4 のテストは leader の件、`inbox notify`、予備の道の 3 つを見る
4. AC9 の skill の文(処理の順、`recurred` の読み方、`--types` と `alert_notify`)を固定するテストは無い。文書の範囲として /verify が読んで確かめた

## Verdict

- Pass: run-test.sh(shell 45 本、Go 10 パッケージ)、`-race -count=3` の 3 つの回、coverage の回のすべてが通った。AC1〜AC10 と AC1b、AC1c の挙動はテストが固定し(AC9 は文書、AC1b の director の半分は Test gaps 1)、落ちたテストは無い
- Fail: 0
- Blocked: なし
