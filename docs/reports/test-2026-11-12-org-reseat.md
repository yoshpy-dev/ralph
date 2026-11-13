# Test report: org-reseat(`ralph org reseat`、#312c の 1 本目)、cycle 2

- Date: 2026-11-13 JST(UTC も 2026-11-13。run-test.sh は 02:00:27Z から 02:55:46Z、時間切れになった org と cli の打ち直しは 02:56:23Z から 03:21:52Z)
- Plan: `docs/plans/active/2026-11-12-org-reseat.md`(`## Test plan`、AC-1〜AC-11、AC-15〜AC-21。AC-12 は実機の記録 `docs/evidence/org-reseat-live-2026-11-12.md` の担当で、ここでは繰り返していない)
- Tester: orchestrator が inline で回した(`/test` の dispatch が失敗したときの fallback)。cycle 2 の tester サブエージェント `test-reseat-c2` は、変異の準備の途中で週の使用上限(`You've hit your weekly limit · resets Nov 14 at 11pm (Asia/Tokyo)`)に当たって止まった。報告もコミットも残っておらず、worktree は clean だった。pipeline の cycle 2(`cycle-count.json` は `{"cycle": 2}`)
- Scope: cycle 1 の報告(69908490)のあとの reseat の直し 3 つと、そのあいだに取り込んだ origin/main(4220ea7c、#308 と #310)。直しは adc3e205(cross-review cycle 1 の ACTION_REQUIRED 3 件)、e9461b79(絶対パスにした `--config` を、reseat が出す reseat・stop・spawn・start の打ち方すべてに付ける)、0468d96f(self-review cycle 2 の L11 と L12)である。run-test.sh は changed scope のまま golang を選んだ(`Language scope: changed (changed_languages)`)。worktree の production code とテストには手を入れていない。変異は `go test -overlay` で当て、scratch は `/private/tmp` と `$TMPDIR` の下に置いた
- Evidence: `docs/evidence/test-2026-11-12-org-reseat.log`(`.gitignore` の `docs/evidence/*.log` に当たるのでコミットしない)。cycle 2 の節に、run-test.sh の全出力(wrapper のログは `docs/evidence/verify-2026-11-13-020027.log`)、org と cli の打ち直し、`-race` の回、変更行の coverage、変異の置き換えと結果を足した

## Test execution

| Suite / Command | Tests | Passed | Failed | Skipped | Duration |
| --- | --- | --- | --- | --- | --- |
| `./scripts/run-test.sh` の shell の段(`tests/test-*.sh`) | 45 本 | 45 | 0 | 2 行(browser の検出、2.41 より古い実 git の場合。どちらも同じ script の中で別の手段が見る) | 約 15 分 |
| `./scripts/run-test.sh` の Go の段(`go test ./...`、`-timeout=40m`) | 10 パッケージ | 8 | 2(org と cli。どちらも 40 分の時間切れ。下の注) | 0 | org 2,402 s、cli 2,402 s |
| 打ち直し `GOFLAGS="-timeout=40m $(go env GOFLAGS)" go test -count=1 ./internal/org/ ./internal/cli/` | 2 パッケージ | 2 | 0 | 0 | org 1,528 s、cli 1,332 s |
| `go test -race -count=3 -v -run 'Reseat\|TestStartFeature_InputRefusalsComeBeforeTheGates' ./internal/org/ ./internal/cli/` | org 36 本、cli 12 本、それぞれ 3 回 | 144 | 0 | 0 | org 10.7 s、cli 165.9 s |
| coverage の回(org `-run 'Reseat\|StartFeature\|SpawnPrecheck\|Stop\|Spawn'`、cli `-run Reseat`) | 2 パッケージ | 2 | 0 | 0 | org 75.3 s、cli 33.2 s |
| 変異(`go test -overlay`、下の節) | 11 個 | 11 個とも死んだ | 生き残り 0 | 0 | 1 個 5〜72 s |

注: 最初の run-test.sh では、org と cli が `panic: test timed out after 40m0s`(約 2,401 s)で落ちた。止まっていた場所は、cli が `TestOrgEscalate_LedgerThatCannotBeResolvedStillReachesTheHuman`、org が `TestGate_BaseResolution` で、どちらも reseat のテストではなく、git を呼んで fixture を作る途中だった。Go の段は 02:15Z〜02:55Z ごろに回り、そのあいだの load average は 7〜16 で、ほかのセッションの go test と重なっていた。cycle 1 の回では org は 777 s で通っている。uptime で負荷を確かめ(load 8〜10、ほかの go test は無し)てから、02:56Z に org と cli だけを同じ 40 分の timeout で打ち直し、両方 ok だった。timeout は延ばしておらず、テストも変えていない。打ち直しのあいだも load は 11〜16 で、03:08Z からは別のセッションの `go test ./...` が並んで回っていた。

## cycle 2 の直しとテスト

cycle 1 のあとに直した 6 件を、どのテストが固定するかで並べる。「変異」の列は、そのテストが落とした cycle 2 の変異の id(下の節)である。

| 直したこと | コミット | テスト(`internal/org/reseat_test.go`、CLI は `internal/cli/org_reseat_test.go`) | 変異 |
| --- | --- | --- | --- |
| 最初の spawn が `spawned` に届かなかった座席の打ち直しが断られた(cross-review の指摘 1、AC-18) | adc3e205 | `TestReseat_RetryStartsASeatWhoseFirstSpawnNeverEnded`。人が止めた同じ座席(note も `spawned` も無い)は今も断られることも見る | N1 |
| leader 以外の座席の reseat が途中で失敗すると DECISION が出なかった(指摘 2、FR-7) | adc3e205 | `TestReseat_DecisionForTheSeatsStartedBeforeAFailure`(implementer が立ち、reviewer が spawn のロックで断られる。DECISION は implementer を名指しし、打ち直しの成功後の DECISION は reviewer だけ)、CLI の `TestOrgReseat_APartialFailurePrintsTheDecisionAndTheConfig` | N3 |
| 打ち直しが `--config` を落とした(指摘 3) | adc3e205 | `TestReseat_DecisionForTheSeatsStartedBeforeAFailure` の打ち直し、`TestReseatRetryCommand` の `--config` の 3 行、CLI の `TestOrgReseat_APartialFailurePrintsTheDecisionAndTheConfig`(相対の `--config` が絶対パスになり、作業ディレクトリの ralph.toml が別の承認方式のとき、`--config` なしの reseat は断られ、出た打ち直しは別のディレクトリから打っても通る) | N4、N4b、N4c、N9、N5 |
| 断りが出す stop・spawn・start・reseat の打ち方にも `--config` を付ける | e9461b79 | `TestPlanReseat_RefusesAnOrgWithoutABinding`、`TestPlanReseat_RefusesACustomRoleSeat`、`TestPlanReseat_NoActiveSeatNamesTheStoppedSeats`、`TestPlanReseat_NoActiveSeatWithoutStoppedOnes` | N4、N4c、N8a、N8b、N8c |
| `--seat` が受け付ける座席の定義を 1 つにする(L11、AC-6) | 0468d96f | `TestPlanReseat_NoActiveSeatNamesOnlySeatsTheSeatFlagTakes`(reseat が触った座席は名指しされ、人が止めた座席は名指しされない。出た `--seat` の打ち方は `PlanReseat` の対象の選び方を通る) | N1、N2 |
| help の「出すコマンドがすべて `--config` を繰り返す」を、reseat・stop・spawn・start に絞る(L12) | 0468d96f | `TestOrgReseatHelp_SaysWhatItDoesAndHowToSwitchAccounts`(固定した文を新しい文に合わせた) | 文なので当てていない |

## 受け入れ条件とテスト

どのテストも run-test.sh の shell の段か、org と cli の打ち直しで PASS し、`Reseat` を名前に含むものと `TestStartFeature_InputRefusalsComeBeforeTheGates` は `-race -count=3` でも PASS した。cycle 1 から変わった行だけ、テストと変異を書き足した。それ以外の行のテストと cycle 1 の変異(M1〜M9、C1)は 69908490 の報告のとおりで、その箇所の production code は cycle 2 で変わっていない。

| AC | テスト | 変異 |
| --- | --- | --- |
| AC-1 3 席を止め、leader → implementer → reviewer の順に立てる | `TestReseat_StopsAndStartsEverySeatInOrder`、`TestPlanReseat_DefaultTargetsAndOrder`、CLI の `TestOrgReseat_StopsAndStartsEverySeat` | — |
| AC-2 leader が新しい workspace の root の pane | `TestReseat_LeaderIsTheFirstPaneOfANewWorkspace` | — |
| AC-3 `--seat implementer --driver codex --model` と DECISION の行 | `TestReseat_OneSeatOnAnotherDriver`、`TestPlanReseat_DriverAndModelForOneSeat`、CLI の `TestOrgReseat_OneSeatPrintsTheDecisionForTheLeader`。途中で失敗したあとの DECISION は cycle 2 で足した `TestReseat_DecisionForTheSeatsStartedBeforeAFailure`、CLI の `TestOrgReseat_APartialFailurePrintsTheDecisionAndTheConfig` | N3 |
| AC-4 止める前の断り | `TestPlanReseat_RefusesWhatTheStartsWouldRefuse`、`TestReseat_ARefusalWritesNothing`、`TestPlanReseat_GatesRunOnceAfterTheChecks` | cycle 1 の M9 は生き残ったまま(Test gaps 1) |
| AC-5 結びつきの無い org の断り | `TestPlanReseat_RefusesAnOrgWithoutABinding`(`--config` を付けた回を cycle 2 で足した)、CLI の `TestOrgReseat_OrgWithoutBindingRefused` | N4、N4c、N8a |
| AC-6 active な座席が無いときの断りと `--seat` の打ち方。打ち直すと立つ | `TestPlanReseat_NoActiveSeatNamesTheStoppedSeats`、`TestPlanReseat_NoActiveSeatWithoutStoppedOnes`、cycle 2 で足した `TestPlanReseat_NoActiveSeatNamesOnlySeatsTheSeatFlagTakes`、`TestReseat_RetryAfterAFailedStart` の leader の subtest | N1、N2、N4、N4c、N8c |
| AC-7 座席の pane から打った reseat の断り | `TestPlanReseat_RefusedFromASeatPane` | — |
| AC-8 pane が無い座席の立て直し | `TestReseat_SeatsWhosePanesAreGone` | — |
| AC-9 `--dry-run` の出力 | `TestPlanReseat_DefaultTargetsAndOrder`、CLI の `TestOrgReseat_DryRunPrintsThePlanAndStartsNothing` | — |
| AC-10 `stop_failed` で何も立てない | `TestReseat_AStopThatFailsStartsNothing` | cycle 1 の C1 は生き残ったまま(Test gaps 3) |
| AC-11 フラグの組み合わせの断り | `TestPlanReseat_RefusesBadFlags`、`TestReseat_AFlagErrorTakesNoLock`、CLI の `TestOrgReseat_FlagRefusals` | — |
| AC-15 `reseat=<id>` の note と `reseated` 1 件 | `TestReseat_RecordsItsIDOnEveryStopAndStart`、`TestReseat_AStopThatFailsStartsNothing`、`TestReseat_RetryAfterAFailedStart`、`TestReseatNoteAndReseatedID` | — |
| AC-16 help が動詞とアカウントの移し方を言う | `TestOrgReseatHelp_SaysWhatItDoesAndHowToSwitchAccounts`。skill の 4 つの写しの文を固定するテストは無い | — |
| AC-17 `reseat.lock` と止める直前の読み直し | `TestReseat_RefusedWhileAnotherReseatHoldsTheLock`、`TestReseat_ASeatChangedBeforeItsStop` | — |
| AC-18 起動の失敗のあと、出た打ち直しで立つ | `TestReseat_RetryAfterAFailedStart`、`TestPlanReseat_SeatAcceptsASeatSpawnedInTheCurrentLife`、`TestReseatRetryCommand`、CLI の `TestOrgReseat_AFailedStartPrintsTheRetry`。cycle 2 で `TestReseat_RetryStartsASeatWhoseFirstSpawnNeverEnded`、`TestReseat_DecisionForTheSeatsStartedBeforeAFailure`、CLI の `TestOrgReseat_APartialFailurePrintsTheDecisionAndTheConfig` を足した | N1、N4、N4b、N4c、N5、N9 |
| AC-19 `--dry-run` は見張りも server も起動せず、何も書かない | `TestPlanReseat_DryRunStartsNoWatchOrServer`、CLI の `TestOrgReseat_DryRunPrintsThePlanAndStartsNothing` | — |
| AC-20 切れた reseat の座席が既定の対象になる。signal と SIGPIPE で終わらない | `TestPlanReseat_FinishesAReseatThatWasCutOff`、CLI の `TestOrgReseat_ARealReseatRunsThroughTheSignalHold`、`TestOrgReseatHoldSignals_KeepsTheProcessAndSaysSoOnce`、`TestOrgReseatHoldSignals_AClosedStderrDoesNotEndIt` | — |
| AC-21 独自の役割の座席の断り | `TestPlanReseat_RefusesACustomRoleSeat`(`--config` を付けた回を cycle 2 で足した) | N4、N8b |

## 回帰

| Previously broken behavior | Status | Evidence |
| --- | --- | --- |
| 最初の spawn が `spawned` に届かなかった座席の打ち直し(cross-review cycle 1 の指摘 1) | Pass | `TestReseat_RetryStartsASeatWhoseFirstSpawnNeverEnded`。N1 の下では FAIL |
| leader 以外の reseat が途中で失敗したときの DECISION(指摘 2) | Pass | `TestReseat_DecisionForTheSeatsStartedBeforeAFailure`、CLI の `TestOrgReseat_APartialFailurePrintsTheDecisionAndTheConfig`。N3(直す前の `finish` の判定)の下では FAIL |
| 打ち直しの `--config`(指摘 3) | Pass | 同上と `TestReseatRetryCommand`。N4b、N9、N5 の下では FAIL |
| active な座席が無いときの断りが、次の回で断られる `--seat` を出した(L11) | Pass | `TestPlanReseat_NoActiveSeatNamesOnlySeatsTheSeatFlagTakes`。N2 の下では FAIL |
| cycle 1 の回帰の項(`StartFeature`、spawn の事前の判定、Note が空の Stop と Spawn の Details、`reseated` を古い binary が数えないこと) | Pass | org と cli の打ち直しの全体。その箇所のコードは cycle 2 で変わっていない |
| 取り込んだ #308(send の画面の fallback、`status --live`)と #310(Codex の hook の信頼の検査) | Pass | org と cli の打ち直しの全体と run-test.sh のほかのパッケージで、それぞれのテストが通った |

## Coverage

- Statement: cycle 1 の報告のあとに `internal/org/reseat.go` と `internal/cli/org_reseat.go` が足した行(`git diff -U0 69908490..HEAD`)のうち、cover profile の block に入る行を、名前で絞った回(org は `Reseat|StartFeature|SpawnPrecheck|Stop|Spawn`、cli は `Reseat`)の profile と突き合わせた。`reseat.go` は 52/52 行、`org_reseat.go` は 4/7 行を通った。通らなかった 3 行(`org_reseat.go:209-211`)は `filepath.Abs` の失敗を返す行で、作業ディレクトリが読めないときにしか通らない
- Branch: Go の標準ツールに branch coverage は無いので測っていない
- Notes: 名前で絞った回なので、パッケージ全体の数字(org 56.5%、cli 8.8%)は意味を持たない

## 変異

cycle 2 の直しが足した判定と文字列に 11 個の変異を当てた。各変異は文字列の置き換え 1 つで、置き換える文字列がファイルに 1 回だけ現れることを確かめてから当てた。build に失敗した変異は無い。org の変異は `-run Reseat ./internal/org/`、N5 は `-run Reseat ./internal/cli/` で回した。11 個とも死んだ。

| id | ファイル | 変異 | 結果 |
| --- | --- | --- | --- |
| N1 | `reseat.go` | `startable` から `\|\| s.touched` を外す | 死んだ。`TestReseat_RetryStartsASeatWhoseFirstSpawnNeverEnded`、`TestPlanReseat_NoActiveSeatNamesOnlySeatsTheSeatFlagTakes` |
| N2 | `reseat.go` | `reseatNoActiveSeatErr` の `if !s.startable() { continue }` を外す | 死んだ。`TestPlanReseat_NoActiveSeatNamesOnlySeatsTheSeatFlagTakes` |
| N3 | `reseat.go` | `finish` の判定を直す前に戻す(立たなかった座席が 1 つでもあれば DECISION を出さない) | 死んだ。`TestReseat_DecisionForTheSeatsStartedBeforeAFailure` |
| N4 | `reseat.go` | `reseatVerbCommand` が `--config` を足さない | 死んだ。6 本(`NoActiveSeatNamesTheStoppedSeats`、`NoActiveSeatWithoutStoppedOnes`、`RefusesACustomRoleSeat`、`RefusesAnOrgWithoutABinding`、`DecisionForTheSeatsStartedBeforeAFailure`、`TestReseatRetryCommand`) |
| N4b | `reseat.go` | `reseatRetryCommand` が `--config` に `""` を渡す | 死んだ。`TestReseat_DecisionForTheSeatsStartedBeforeAFailure`、`TestReseatRetryCommand` |
| N4c | `reseat.go` | `--config` の値を `ShellWord` で包まない | 死んだ。`TestPlanReseat_NoActiveSeatNamesTheStoppedSeats`、`TestPlanReseat_RefusesAnOrgWithoutABinding`、`TestReseatRetryCommand`(空白を含むパスの行) |
| N8a | `reseat.go` | `reseatNoBindingErr` の呼び出しに `""` を渡す | 死んだ。`TestPlanReseat_RefusesAnOrgWithoutABinding` |
| N8b | `reseat.go` | `reseatCustomRoleErr` の呼び出しに `""` を渡す | 死んだ。`TestPlanReseat_RefusesACustomRoleSeat` |
| N8c | `reseat.go` | `reseatNoActiveSeatErr` の呼び出しに `""` を渡す | 死んだ。`TestPlanReseat_NoActiveSeatNamesTheStoppedSeats`、`TestPlanReseat_NoActiveSeatWithoutStoppedOnes` |
| N9 | `reseat.go` | `reseatRun` に `hintConfigPath` を渡さない | 死んだ。`TestReseat_DecisionForTheSeatsStartedBeforeAFailure` |
| N5 | `internal/cli/org_reseat.go` | `--config` を絶対パスにせず `filepath.Clean` だけにする | 死んだ。`TestOrgReseat_APartialFailurePrintsTheDecisionAndTheConfig` |

`reseatLifeSeats` で `touched` を状態の event だけから取る変異は当てていない。`reseat=<id>` の note を持つ座席の event のうち状態の event でないものは止める失敗(`stop_failed`)だけで、その座席は active のまま `startable` が真になるので、結果は変わらない。

## Failure analysis

| Test | Error | Root cause | Proposed fix |
| --- | --- | --- | --- |
| `internal/org` と `internal/cli` の全体(最初の run-test.sh) | `panic: test timed out after 40m0s`(org は `TestGate_BaseResolution`、cli は `TestOrgEscalate_LedgerThatCannotBeResolvedStillReachesTheHuman` の git の fixture の中) | 環境。load average 7〜16 で、ほかのセッションの go test と重なった。止まった 2 本は reseat の変更が触っていないテストで、同じ timeout の打ち直しで両方通った | なし。打ち直しの結果を採った |

run-test.sh の shell の段、org と cli の打ち直し、`-race` の回、coverage の回で落ちたテストは無い。変異の FAIL は、上の節の意図した落ち方である。

## Test gaps

cycle 1 の Test gaps 1〜8 は、cycle 2 の直しでは閉じていない。1〜3 は `docs/tech-debt/README.md` の reseat の Test gaps の行に載っている。

1. 非 leader の reseat の worktree の branch の検査(`reseatSeatsStartErr` の `checkFeatureWorktreeReuse`)を、leader を含まない reseat で通るテストが無い(cycle 1 の M9)。`TestPlanReseat_RefusesWhatTheStartsWouldRefuse` の「the worktree has another branch checked out」は既定の対象(leader を含む)の行である
2. Note が空の spawn の Details を完全一致で固定するテストが無い(cycle 1 の M68)
3. CLI の `stop failed:`・`left running:`・`left as it is:`・`not started:` の行を出すテストが無い(cycle 1 の C1)。cycle 2 の CLI のテストは `left stopped:` を見るが、この 4 つは見ない
4. 検査のあと止める前に org が disband された場合のテストが無い(cycle 1 の probe P4 で確かめた)
5. I/O の失敗の戻りは行の coverage に入らない。cycle 2 では `org_reseat.go:209-211`(`filepath.Abs` の失敗)が増えた
6. `dryRunSpawn` の note(cycle 1 の M4)は届かない行で、固定されていない
7. AC-12 の実機は `docs/evidence/org-reseat-live-2026-11-12.md` の担当で、cycle 2 でも繰り返していない。実機の binary は dd204370 で、cycle 1 の P9・P10、SIGPIPE の直し、cycle 2 の 3 つの直しより前である
8. AC-16 の skill の 4 つの写しの文はテストで固定していない。cycle 2 の `/verify` が見つけた skill と README の drift(V9〜V13)は、`/sync-docs` が直す
9. run-test.sh の 1 回の実行で Go の全パッケージが通った記録は cycle 2 には無い。org と cli は時間切れのあと、2 つだけで打ち直した。ほかの 8 パッケージは最初の回で通っている

## Verdict

- Pass: run-test.sh の shell の段(45 本)と Go の 8 パッケージ、時間切れのあと打ち直した org と cli、`-race -count=3` の回、coverage の回がすべて通った。cycle 2 の直し 6 件はどれもテストが固定し、直しに当てた変異 11 個はすべて死んだ。cycle 1 の Test gaps は残っている
- Fail: 0
- Blocked: なし
