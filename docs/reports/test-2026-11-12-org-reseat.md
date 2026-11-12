# Test report: org-reseat(`ralph org reseat`、#312c の 1 本目)

- Date: 2026-11-13 JST(UTC では 2026-11-12。run-test.sh は 15:30:16Z から 15:57:47Z、追加の回は 15:35Z から 16:20Z)
- Plan: `docs/plans/active/2026-11-12-org-reseat.md`(`## Test plan`、AC-1〜AC-11、AC-15〜AC-21。AC-12 は実機の記録 `docs/evidence/org-reseat-live-2026-11-12.md` の担当で、ここでは繰り返していない)
- Tester: tester サブエージェント(`/test`、pipeline の cycle 1。`cycle-count.json` は `{"cycle": 1}`)
- Scope: `git diff 0936aaea...af927216`(23 ファイル、+5907/-31)。Go のコードの変更は `internal/org/reseat.go`(新規)、`feature.go`、`verbs.go`、`spawn.go`、`seat.go`、`internal/cli/org_reseat.go`(新規)、`org.go`。run-test.sh は changed scope のままで golang を選んだ(`Language scope: changed (changed_languages)`)。worktree の production code とテストには手を入れていない。変異と probe は `go test -overlay` で当て、scratch は `/private/tmp` の下に置いた
- Evidence: `docs/evidence/test-2026-11-12-org-reseat.log`(`.gitignore` の `docs/evidence/*.log` に当たるのでコミットしない)。run-test.sh の全出力(wrapper のログは `docs/evidence/verify-2026-11-12-153016.log`)、`-race` の回、回帰の回、変更行の coverage、変異の diff と結果、probe のソースと出力
- 起動の仕方: run-test.sh は Bash の 600 秒の上限を超えるので、`perl` の fork と `setsid` で切り離し(INT・HUP・QUIT・TERM は DEFAULT、標準入出力は `/dev/null`)、`caffeinate -w` で終わりを待った。shell の SKIP の行は前の PR の回と同じ 2 行だった

## Test execution

| Suite / Command | Tests | Passed | Failed | Skipped | Duration |
| --- | --- | --- | --- | --- | --- |
| `./scripts/run-test.sh`(rc 0、`==> All verifiers passed.`) | shell 45 本 + Go 10 パッケージ | shell 45 本 OK(PASS を含む行 5,195)、Go 10 パッケージ ok | 0(`--- FAIL` の行 0) | 2 行 | 1,651 秒 |
| └ Go: `internal/org` | 上位 1,757 本 | ok | 0 | — | 777.5 秒 |
| └ Go: `internal/cli` | 上位 1,127 本 | ok | 0 | — | 574.0 秒 |
| └ Go: `internal/config`、`cmd/ralph`、`internal/gitroot`、`internal/insights`、`internal/org/driver`、`internal/org/protocol`、`internal/scaffold`、`internal/upgrade` | — | ok。`(cached)` は 0 | 0 | — | 20.4、0.9、6.8、4.2、3.5、3.2、1.2、2.5 秒 |
| `-race -count=1 -v -run Reseat`(org、cli) | 上位 43 本(org 32、cli 11) | 43 | 0 | 0 | 62 秒。`DATA RACE` 0 |
| `-race -count=3 -v -run 'Reseat\|TestStartFeature_InputRefusalsComeBeforeTheGates'`(org、cli) | 上位 132 回(44 本 × 3、subtest 込みの PASS 行 333) | 333 | 0 | 0 | 65 秒。`DATA RACE` 0 |
| 回帰: `-v -run 'TestStartFeature\|TestSpawnPrecheckErr\|TestOrgStop\|TestStop\|TestSpawn'`(org) | 上位 123 本(`TestStartFeature_*` 34、`TestSpawnPrecheckErr_*` 2、Stop 86、`TestSpawn*` 1) | 123 | 0 | 0 | 24 秒 |
| coverage: org と cli を名前で絞った回(下の Coverage) | org `Reseat\|StartFeature\|SpawnPrecheck\|Stop\|Spawn`、cli `Reseat` | ok | 0 | — | 74 秒、24 秒 |
| 変異: production code の 10 個(下の節) | 10 | 5 個が死んだ | — | — | 絞った回 12〜112 秒、パッケージ全体の回 2 つ |
| probe: overlay で足したテスト 6 本(org 5、cli 1。コミットしない) | 6 | HEAD で 6 本とも PASS | 0 | — | 1 回 1〜3 秒(build を除く) |

`-race` の回は 2 つの signal のテスト(`TestOrgReseatHoldSignals_KeepsTheProcessAndSaysSoOnce`、`TestOrgReseatHoldSignals_AClosedStderrDoesNotEndIt`)の子の process も `-race` の binary で回る。上位の本数は `go test -list .` で数えた。Skipped の 2 行は `tests/test-plan-visual.sh` の browser detection と、`tests/test-secret-scan-branch.sh` の git 2.41 より古い本物の git の case で、どちらも手元の環境による。

run-test.sh のあいだ、1 分の load は 7.7 から 23.0 だった。shell の段のあいだに、絞った `go test`(変異 7 個、probe、`-race -count=1`、回帰の回)を並べて回した。Go の段とは重ねていない。org は 777 秒で、30 分の上限の 43% だった。落ちたテストは無く、打ち直しはしていない。

## 受け入れ条件とテスト

どのテストも run-test.sh の回で PASS し、`Reseat` を名前に含むものと `TestStartFeature_InputRefusalsComeBeforeTheGates` は `-race -count=3` でも PASS した。「変異」の列は、そのテストが落とした変異の id(下の節)である。

| AC | テスト(`internal/org/reseat_test.go`、CLI は `internal/cli/org_reseat_test.go`) | 変異 |
| --- | --- | --- |
| AC-1 3 席を止め、leader → implementer → reviewer の順に立てる。`scope_reserved` が増えない。implementer の spawn と TASK が承認の検査を通る | `TestReseat_StopsAndStartsEverySeatInOrder`、`TestPlanReseat_DefaultTargetsAndOrder`、CLI の `TestOrgReseat_StopsAndStartsEverySeat` | — |
| AC-2 leader が新しい workspace の root の pane | `TestReseat_LeaderIsTheFirstPaneOfANewWorkspace` | — |
| AC-3 `--seat implementer --driver codex --model` と DECISION の行 | `TestReseat_OneSeatOnAnotherDriver`、`TestPlanReseat_DriverAndModelForOneSeat`、CLI の `TestOrgReseat_OneSeatPrintsTheDecisionForTheLeader` | — |
| AC-4 止める前の断り(計画の変更、自分の節か前文、計画のファイル、承認方式、見張り、herdr server、worktree の記録、regate、implementer の計画) | `TestPlanReseat_RefusesWhatTheStartsWouldRefuse`(13 行)、`TestReseat_ARefusalWritesNothing`、`TestPlanReseat_GatesRunOnceAfterTheChecks` | M9 は生き残った(Test gaps 1) |
| AC-5 結びつきの無い org の断り | `TestPlanReseat_RefusesAnOrgWithoutABinding`(5 行)、CLI の `TestOrgReseat_OrgWithoutBindingRefused` | — |
| AC-6 active な座席が無いときの断りと `--seat` の打ち方。打ち直すと立つ | `TestPlanReseat_NoActiveSeatNamesTheStoppedSeats`、`TestPlanReseat_NoActiveSeatWithoutStoppedOnes`。全座席が止まった状態から `--seat` で立つ実行は `TestReseat_RetryAfterAFailedStart` の leader の subtest | — |
| AC-7 座席の pane から打った reseat の断り | `TestPlanReseat_RefusedFromASeatPane` | — |
| AC-8 pane が無い座席の立て直し | `TestReseat_SeatsWhosePanesAreGone` | M3 |
| AC-9 `--dry-run` の出力 | `TestPlanReseat_DefaultTargetsAndOrder`、CLI の `TestOrgReseat_DryRunPrintsThePlanAndStartsNothing` | — |
| AC-10 `stop_failed` で何も立てない | `TestReseat_AStopThatFailsStartsNothing`。CLI の出力は probe で確かめた(Test gaps 3) | M3。C1 は生き残った |
| AC-11 フラグの組み合わせの断り | `TestPlanReseat_RefusesBadFlags`(7 行)、`TestReseat_AFlagErrorTakesNoLock`、CLI の `TestOrgReseat_FlagRefusals`(6 行) | — |
| AC-15 `reseat=<id>` の note と `reseated` 1 件。途中の失敗でも座席ごとの結果 | `TestReseat_RecordsItsIDOnEveryStopAndStart`、`TestReseat_AStopThatFailsStartsNothing`、`TestReseat_RetryAfterAFailedStart`、`TestReseatNoteAndReseatedID` | M2、M3、M5、M7 |
| AC-16 help が動詞とアカウントの移し方を言う | `TestOrgReseatHelp_SaysWhatItDoesAndHowToSwitchAccounts`。skill の 4 つの写しの文を固定するテストは無い(/verify が読んで確かめた) | — |
| AC-17 `reseat.lock` と止める直前の読み直し | `TestReseat_RefusedWhileAnotherReseatHoldsTheLock`(3 行)、`TestReseat_ASeatChangedBeforeItsStop`(2 つの subtest、test hook で順序を決める) | — |
| AC-18 起動の失敗のあと、出た打ち直しで立つ | `TestReseat_RetryAfterAFailedStart`(leader の失敗、非 leader の失敗)、`TestPlanReseat_SeatAcceptsASeatSpawnedInTheCurrentLife`(stopped、spawn_failed、rejected)、`TestReseatRetryCommand`、CLI の `TestOrgReseat_AFailedStartPrintsTheRetry` | — |
| AC-19 `--dry-run` は見張りも server も起動せず、何も書かない | `TestPlanReseat_DryRunStartsNoWatchOrServer`(5 行)、CLI の `TestOrgReseat_DryRunPrintsThePlanAndStartsNothing`(4 行) | — |
| AC-20 切れた reseat の座席が既定の対象になる。signal と SIGPIPE で終わらない | `TestPlanReseat_FinishesAReseatThatWasCutOff`(4 つの subtest)、CLI の `TestOrgReseat_ARealReseatRunsThroughTheSignalHold`、`TestOrgReseatHoldSignals_KeepsTheProcessAndSaysSoOnce`、`TestOrgReseatHoldSignals_AClosedStderrDoesNotEndIt` | M3 |
| AC-21 独自の役割の座席の断り | `TestPlanReseat_RefusesACustomRoleSeat` | — |

計画の Test plan の Edge cases は次のとおりである。leader だけの org は `TestPlanReseat_LeaderOnly`、同じ座席を 2 回名指しは `TestPlanReseat_RefusesBadFlags`、`--seat` に leader と非 leader の両方は `TestPlanReseat_LeaderNotATarget`、分割計画のファイルが消えた org は `TestPlanReseat_RefusesWhatTheStartsWouldRefuse` の「the split plan is gone」。残りの 2 つ(reviewer がまだいない org、承認方式が split の org)は、`Reseat` まで回すテストが無いので probe で回した。どちらも HEAD で PASS した(下の節の P1、P2)。

## 回帰

| Previously broken behavior | Status | Evidence |
| --- | --- | --- |
| `StartFeature` の動き(検査を `startFeatureInput` と `checkLedger` に切り出した) | Pass | 回帰の回の `TestStartFeature_*` 34 本、`TestStartFeature_InputRefusalsComeBeforeTheGates`(入力の断りが gate より前)。run-test.sh の org 全体 |
| spawn の事前の判定が Spawn と一致すること | Pass | `TestSpawnPrecheckErr_MatchesSpawn`、`TestSpawnPrecheckErr_InFlightSeatRefusal_MatchesSpawn` |
| Note が空の Stop の Details が前と同じこと | Pass | 回帰の回の Stop の 86 本。M1(空の note で Details を変える)は、3 本の Stop のテストが Details の完全一致で落とした |
| Note が空の Spawn の Details が前と同じこと | Pass(固定は弱い) | spawn のテスト(`TestOrgSpawn_*` ほか)は run-test.sh の org 全体で通った。今の Details は前と同じ(`noteDetails` は空の note で入力をそのまま返し、M1 がそれを固定する)。ただし spawn の call site だけを変える M68 は org と cli の全体で生き残った(Test gaps 2) |
| `reseated` を古い binary の判定が数えないこと | Pass | `reseated` は `stateEvents` に無い(`internal/org/seat.go`)。run-test.sh の org 全体の roster、watch、disband のテストが変わらず通った |

## Coverage

- Statement: この diff が Go のコード(テストを除く)に足した行のうち、cover profile の block に入る 805 行の 739 行(91.8%)を通った。`git diff -U0 0936aaea...HEAD` の足した行と、上の絞った回の profile を突き合わせた。`internal/org/reseat.go` 526/572、`internal/cli/org_reseat.go` 168/188、`internal/org/feature.go` 36/36、`spawn.go` 3/3、`verbs.go` 5/5、`internal/cli/org.go` 1/1。名前で絞った回なので、パッケージ全体の数字(org 58.2%、cli 9.5%)は意味を持たない
- Branch: Go の標準ツールに branch coverage は無いので測っていない
- 通らなかった行(evidence に行番号の一覧がある):
  - `reseat.go` の 46 行のうち 33 行は I/O の失敗の戻り(manifest の読み直し、`reseat.lock` の marker の読み書き、乱数の読み取り、`reseated` の追記、`RenderRolePrompt` と worktree の Lookup の失敗)、4 行は不正な org_id の断りである。残りの 9 行は、並べ替えの比較の片側(192)、手で書いた台帳の役割の読み(205)、非 leader の reseat の worktree の branch の検査(429-430、Test gaps 1)、非 leader の座席の `checkSpawnInput` の断り(447-448)、Err の無い起動の失敗(1102-1103)、検査と止めるあいだの disband(1148、Test gaps 4)
  - `org_reseat.go` の 20 行のうち 8 行は台帳と設定の読みの失敗、全 signal が無視された起動、herdr が PATH に無い dry run の読み。残りの 12 行は、dry run では呼ばれない gate の closure(311-315)と、止める失敗と途中の座席の出力の行(393、398-399、440、453、455、457、Test gaps 3)
- Notes: M68 が変えた `spawn.go` の 2 行は通った行だが、変えても org と cli の全体が通った。行の coverage では見えない穴で、変異で見つけた。M9 が外した 428-430 は通らなかった行である。どちらも下の節に書く

## 変異と probe

production code に 10 個の変異を当てた。各変異は文字列の置き換え 1 つで(M68 は 2 つ)、置き換える文字列がファイルに 1 回だけ現れることを確かめてから当てた。build に失敗した変異は無い。diff と結果は evidence にある。

| id | ファイル | 変異 | 回した範囲 | 結果 |
| --- | --- | --- | --- | --- |
| M1 | `verbs.go` | `noteDetails` で note が空のとき `details + " x"` を返す | `-run 'Stop\|Spawn'` | 死んだ。`TestOrgStop_CallOrder_CtrlCThenCloseThenLeaveThenStopped`、`TestOrgStop_DryRun_ExistingSeat_RecordsDryRunDetails`、`TestOrgStop_PaneGoneAtCheck_RecordsAlreadyClosed` |
| M2 | `spawn.go` | `spawned` の Details に note を足さない | `-run Reseat` | 死んだ。`TestReseat_RecordsItsIDOnEveryStopAndStart` |
| M3 | `verbs.go` | Stop の Details に note を足さない | `-run Reseat` | 死んだ。上位 4 本(`FinishesAReseatThatWasCutOff`、`RecordsItsIDOnEveryStopAndStart`、`SeatsWhosePanesAreGone`、`AStopThatFailsStartsNothing`) |
| M4 | `spawn.go` | `dryRunSpawn` の `spawn_started` に note を足さない | `-run Reseat` | 生き残った。note と DryRun を一緒に渡す呼び手は無い(reseat の dry run は Spawn を呼ばない)ので、届かない行である |
| M5 | `feature.go` | `StartFeature` が leader の spawn に Note を渡さない | `-run Reseat` | 死んだ。`TestReseat_RecordsItsIDOnEveryStopAndStart` |
| M6 | `spawn.go` | `spawned` の Details を `join + " " + p.Note` にする(note が空のときだけ末尾に空白が付く) | `-run 'Spawn\|Report\|Stop\|Status\|Roster'` | 生き残った |
| M7 | `spawn.go` | `spawn_started` の Details を `" " + p.Note` にする | 同上 | 死んだ。`TestReseat_RecordsItsIDOnEveryStopAndStart`(note がある側で) |
| M68 | `spawn.go` | M6 に加え、note が空のときだけ `spawn_started` の Details を `"x"` にする | org と cli の全体 | 生き残った(org 854.5 秒、cli 643.4 秒) |
| M9 | `reseat.go` | 非 leader の reseat の `checkFeatureWorktreeReuse` を外す(428-430) | `-run 'Reseat\|StartFeature'`、org と cli の全体 | 生き残った(org 817.2 秒、cli 688.3 秒)。probe P3 で死んだ |
| C1 | `org_reseat.go` | 止める失敗の行の `stop failed` を `stopped` にする | cli `-run Reseat`(`reseat` を打つテストは `org_reseat_test.go` だけ) | 生き残った。probe P5 で死んだ |

M68 と M9 の全体の回は 2 つを並べて回した(1 分の load 8〜10)。

probe(overlay で足した。コミットしない。ソースは evidence にある):

| id | probe | HEAD | 確かめたこと |
| --- | --- | --- | --- |
| P1 | `TestGapReseat_SplitApprovalOrg` | PASS | 承認方式が split の org の 3 席の reseat。結びつきは split のまま、`scope_reserved` は増えず、implementer の spawn の検査が通る。設定を per-plan に変えると止める前に断る |
| P2 | `TestGapReseat_NoReviewerYet` | PASS | leader と implementer だけの org。implementer → leader の順に止め、leader → implementer の順に立てる |
| P2b | `TestGapReseat_LeaderAloneWhileOthersRun` | PASS | 他の座席が動いているときの `--seat leader`。leader は org の workspace に戻り(workspace を作らない)、他の座席には event が無く、DECISION は出ない |
| P3 | `TestGapReseat_OtherSeatRefusedWhenTheWorktreeHasAnotherBranch` | PASS | worktree に別の branch がある org の `--seat reviewer` を止める前に断る。M9 の下では FAIL |
| P4 | `TestGapReseat_OrgDisbandedBeforeAStop` | PASS | 検査のあと最初の停止の前に disband されたとき、何も止めず、`reseat=` の付いた event も `reseated` も書かない。error は「latest state event is now no state event since the org's last disband」。出る打ち直し(`--seat leader --seat implementer`)は、結びつきが無いので断られる |
| P5 | `TestGapOrgReseat_AStopThatFailsPrintsTheSeatsLeft`(cli) | PASS | implementer の pane を閉じられないとき、CLI は `stopped: reviewer`、`stop failed: implementer`、`left running: leader (not stopped)`、`left stopped: reviewer`、打ち直しの行を出して exit 1。C1 の下では FAIL |

## Failure analysis

| Test | Error | Root cause | Proposed fix |
| --- | --- | --- | --- |
| なし | — | — | — |

run-test.sh、`-race` の 2 回、回帰の回、coverage の回で落ちたテストは無い。変異と probe の FAIL は、上の節の意図した落ち方である。

## Test gaps

1. 非 leader の reseat の worktree の branch の検査(`reseat.go:428-430`)を通るテストが無い。外しても(M9)org と cli の全体が通る。外すと、worktree に別の branch がある org の `--seat reviewer` は断られず、reviewer がその worktree で立つ(FR-4 の取りこぼし)。HEAD は正しく断る(P3)。いちばん小さい足し方は、`TestPlanReseat_RefusesWhatTheStartsWouldRefuse` に「the worktree has another branch checked out, one other seat」(`seats: []string{"reviewer"}`)の 1 行を足すことである
2. Note が空の spawn の Details(`spawned` と `spawn_started`)を完全一致で固定するテストが無い。計画の Risks は「note が空なら何も足さない。既存のテストで確かめる」と書くが、既存の spawn のテストは `permission_mode=` を Contains で見るだけで、M68 が org と cli の全体で生き残った。今のコードは正しく(`noteDetails` を M1 が固定する)、壊れても末尾の空白か余分な token だけである。足すなら、spawn のテスト 1 本で `spawned` の Details が `scope=<v> permission_mode=<mode>` と等しく、`spawn_started` の Details が空であることを見る
3. CLI の止める失敗と途中の座席の行(`stop failed:`、`left running:`、`left as it is:`、`not started:`)を出すテストが無い。org 層の結果は `TestReseat_AStopThatFailsStartsNothing` と `TestReseat_ASeatChangedBeforeItsStop` が固定している。CLI の出力は P5 で正しいことを確かめた。足すなら P5 の形(`ORG_STUB_CLOSE_FAIL_IDS` に implementer の pane)がそのまま使える
4. 検査のあと止める前に org が disband された場合(`reseatSeatNow` の not found)のテストが無い。P4 で、何も止めず何も書かないことを確かめた。打ち直しの案内は disband 後の org を指すが、打てば結びつきが無いと断られるので害は無い
5. I/O の失敗の戻り(manifest の読み直し、`reseat.lock` の marker、乱数、`reseated` の追記)は行の coverage に入らない。どれも error を包んで返すだけの行である
6. `dryRunSpawn` の note(M4)は届かない行で、固定されていない
7. AC-12 の実機は `docs/evidence/org-reseat-live-2026-11-12.md` の担当で、この回では繰り返していない。実機の binary は dd204370 で、P9・P10 と SIGPIPE の直し(93395521、e5b2cb85)より前である(/verify の I1)。signal の扱いは、子の process に本物の signal を送るテストで確かめた
8. AC-16 の skill の 4 つの写しの文はテストで固定していない。/verify が読んで確かめ、drift(V1)を `/sync-docs` に回している

## Verdict

- Pass: run-test.sh(shell 45 本、Go 10 パッケージ)、`-race` の 2 回、回帰の回、coverage の回がすべて通った。AC-1〜AC-11 と AC-15〜AC-21 の挙動はテストが固定し、計画の Edge cases の残り 2 つも probe で通った。変異 10 個のうち 5 個が死んだ。生き残った 5 個は、届かない行(M4)、note が空の spawn の Details の形(M6、M68)、CLI の 1 語(C1)、非 leader の reseat の worktree の branch の検査(M9)である。断るか通すかを変えるのは M9 だけで、HEAD は正しく断る(P3)。足し方は Test gaps 1 に書いた
- Fail: 0
- Blocked: なし
