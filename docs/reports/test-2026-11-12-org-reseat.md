# Test report: org-reseat(`ralph org reseat`、#312c の 1 本目)、cycle 3

- Date: 2026-11-13 JST(run-test.sh は 05:45:48Z から 06:25:42Z、JST では 14:45:48 から 15:25:42。時間切れになった org と cli の打ち直しは 06:26:17Z から 06:45:08Z、JST では 15:26:17 から 15:45:08)
- Plan: `docs/plans/active/2026-11-12-org-reseat.md`(`## Test plan`、AC-1〜AC-11、AC-15〜AC-21。AC-12 は実機の記録 `docs/evidence/org-reseat-live-2026-11-12.md` の担当で、ここでは繰り返していない)
- Tester: tester サブエージェント(`/test`)。パイプラインの 3 回目で、cycle 2 の cross-review のあと cap を 3 に上げた回である(`cycle-count.json` は `{"cycle": 2, "cap": 3}`。insight event の `--cycle auto` もこの値を読むので、同じ回の self-review と verify の event と同じく `cycle` は 2 になる)。この report は cycle 2 の本文(2d37fa32)を置き換える。cycle 2 の本文は `git show 2d37fa32:docs/reports/test-2026-11-12-org-reseat.md` で読める
- Scope: cycle 2 の報告のあとの reseat の直し 3 つと、そのあとに取り込んだ origin/main(merge 8460bf64、merge base e902705c。PR #327 の watch の usage_limited と PR #325 の upgrade の behavior notes)。直しは def8d2a3(立てた座席が idempotent で返り、その座席が別のコマンドに別の role・driver・model で立てられていたら start_failed にする。`reseatIdempotentMismatchErr`、テスト用のフック `beforeStart`、新しい `internal/org/reseat_start_test.go`)、df8f674e(その座席に `--driver`・`--model` が渡っていなければ `ReseatResult.Changed` にして打ち直しから外す。関数名の変更)、b142944d(`ralph org reseat --help` の文と、その pin)である。run-test.sh は changed scope のまま golang を選んだ(`Language scope: changed (changed_languages)`)。worktree の production code とテストには手を入れていない。変異は `go test -overlay` で当て、scratch は scratchpad の下に置いた
- Evidence: `docs/evidence/test-2026-11-12-org-reseat.log`(`.gitignore` の `docs/evidence/*.log` に当たるのでコミットしない)。cycle 3 の節に、run-test.sh の全出力(wrapper のログは `docs/evidence/verify-2026-11-13-054548.log`)、30 秒ごとの load average と空き容量、org と cli の打ち直しの全出力、`-race` の回、coverage の回と変更行の突き合わせ、変異の置き換えと結果を足した
- 起動の仕方: run-test.sh と打ち直しは Bash ツールの 600 秒の上限を超えるので、perl の fork と setsid で切り離して起動し(INT・HUP・QUIT・TERM は DEFAULT に戻し、標準入出力は `/dev/null`)、開始・終了・終了コードを別のファイルに書かせて、`caffeinate -w` で前景で待った。空き容量は 71〜73 GiB のままで、ENOSPC は出ていない(`grep -ci 'no space left'` は 0)

## Test execution

| Suite / Command | Tests | Passed | Failed | Skipped | Duration |
| --- | --- | --- | --- | --- | --- |
| `./scripts/run-test.sh` の shell の段(`tests/test-*.sh`) | 45 本 | 45 | 0 | 2 行(browser の検出、2.41 より古い実 git の場合。どちらも同じ script の中で別の手段が見る) | 約 10 分(05:45:48Z〜05:55:40Z ごろ) |
| `./scripts/run-test.sh` の Go の段(`go test ./...`、`-timeout=30m`) | 10 パッケージ(テストの無い root を除く) | 8(うち gitroot、org/driver、org/protocol、scaffold は `(cached)`) | 2(org と cli。どちらも 30 分の時間切れ。下の注) | `-v` なしなので数えていない | org 1,802.8 s、cli 1,801.3 s。run-test.sh 全体は 2,394 s |
| 打ち直し `GOFLAGS="-timeout=30m" go test -count=1 -v ./internal/org/ ./internal/cli/` | 2 パッケージ(トップレベルのテストは org 1,823 本、cli 1,207 本) | 2(org 1,822 本、cli 1,206 本) | 0 | 2 本(Linux だけで走る `TestAbsPath_GetwdFailsOnLinuxOnceTheWorkingDirectoryIsRemoved` と `TestOrgRetiredStateDirEnvAbs_GetwdFailsOnLinuxOnceTheWorkingDirectoryIsRemoved`) | org 1,112.5 s、cli 984.4 s(全体 1,131 s) |
| `go test -race -count=1 -v -run 'Reseat' ./internal/org/ ./internal/cli/` | org 40 本、cli 12 本 | 52 | 0 | 0 | org 4.2 s、cli 28.7 s(build を含めて 46 s) |
| coverage の回(`go test -count=1 -run 'Reseat' -coverprofile`、org と cli) | 2 パッケージ | 2 | 0 | 0 | org 2.0 s、cli 48.0 s |
| 変異(`go test -overlay`、下の節) | 6 個 | 6 個とも死んだ | 生き残り 0 | 0 | 1 個 7〜8 s |

注: run-test.sh の Go の段では、org と cli が `panic: test timed out after 30m0s`(1,802.8 s と 1,801.3 s)で落ちた。timeout は golang の verifier(`packs/languages/golang/verify.sh`)の既定の `-timeout=30m` で、この回の環境の `GOFLAGS` は空だった(cycle 2 の回は 40 分だった)。時間切れのときに走っていたのは、cli が `TestOrgLedger_RelativePersonalStateDirFailsUntilStateDirIsPassed`(5 s 目)、org が `TestMerge_RefusedCleanupKeepsTheMerge/the_org_spawned_again_during_gh_pr_merge`(親のテストは 27 s 目)で、どちらも reseat のテストではない。1 本が止まったのではなく、パッケージ全体の時間が 30 分を超えた。ログに `--- FAIL` の行は 0 件である。Go の段のあいだ、1 分の load average は 4.8〜23.4 で、別のセッションの `go test ./internal/org/ -run 'TestGate|TestGateTools|TestRegate'` と、`-run 'RenderRolePrompt|DirectorSkill|RenderDirectorPrompt'` の org のテストが並んで走っていた。run-test.sh が終わった直後に uptime で load を見て(6.34 8.59 8.96。ほかのセッションの org のテストは 1 本だけ残っていた)、06:26:17Z に org と cli だけを同じ 30 分の timeout で打ち直し、両方 ok だった。時間切れのときに走っていた 2 本は、打ち直しでは 0.60 s と 2.62 s で PASS した。timeout は延ばしておらず、テストも変えていない。打ち直しのあいだの load は 6.4〜32.0 で、06:43Z ごろに一時 32 まで上がった。

## cycle 3 の直しとテスト

cycle 2 のあとに直した 3 件を、どのテストが固定するかで並べる。「変異」の列は、そのテストが落とした cycle 3 の変異の id(下の節)である。

| 直したこと | コミット | テスト(`internal/org/reseat_start_test.go`、CLI は `internal/cli/org_reseat_test.go`) | 変異 |
| --- | --- | --- | --- |
| reseat が止めてから立てるまでに別のコマンドが同じ座席を立てると、`Spawn` は idempotent を返し、reseat は頼んだ設定と違う座席を started と数えていた(cross-review cycle 2 の指摘 1)。返った座席の role・driver・model を計画の座席と比べ、違えば start_failed にして両方の設定を名指しし、後の座席を立てない | def8d2a3 | `TestReseat_ASeatAnotherCommandStartedOnOtherSettings`(`--driver codex --model gpt-x` を頼んだ implementer に、別のコマンドが claude opus で立てる。エラーの全文、`reseated` の記録、DECISION が無いこと、別のコマンドの座席が残ること)、`TestReseat_TheLeaderAnotherCommandStartedOnAnotherModel`(leader は `StartFeature` で立てる。別の start が opus で立てる)、`TestReseat_ASeatAnotherCommandStartedIsLeftOutOfTheRetry`(implementer が start_failed、reviewer は止めたまま)、`TestReseatIdempotentMismatchErr`(role・driver・model を 1 つずつ変えた 3 行と、model が両側 `""` の行)。同じ設定で割り込まれた場合は started のまま: `TestReseat_ASeatAnotherCommandStartedOnTheSameSettings` の 2 つの subtest(reviewer が CLI の既定のモデル、implementer が渡した `--driver`・`--model`) | K1、K2、K3、H1 |
| その座席に `--driver`・`--model` が渡っていなければ `Changed` にして打ち直しから外す。渡っていれば `Changed` にせず、打ち直しがその座席を止めて頼んだ設定で立てる | df8f674e | `TestReseat_ASeatAnotherCommandStartedIsLeftOutOfTheRetry`(打ち直しは `--seat reviewer` だけで、走らせると何も止めず、別のコマンドの implementer は sonnet のまま)、`TestReseat_TheLeaderAnotherCommandStartedOnAnotherModel`(対象が leader だけなので打ち直しは出ない)、`TestReseat_ASeatAnotherCommandStartedOnOtherSettings`(`Changed` は空で、打ち直しは `--driver codex --model gpt-x` を持ち、走らせると別のコマンドの implementer を止めて codex で立てる) | X1、X2 |
| help に、打ち直しから外す座席と、立てる時点の検出を足す | b142944d | `TestOrgReseatHelp_SaysWhatItDoesAndHowToSwitchAccounts`(足した 2 つの文を文ごと固定する) | 文なので当てていない |

## 受け入れ条件とテスト

どのテストも run-test.sh の shell の段か、org と cli の打ち直しで PASS し、`Reseat` を名前に含むものは `-race` でも PASS した。cycle 2 から変わった行だけ、テストと変異を書き足した。それ以外の行のテストと cycle 1・cycle 2 の変異(cycle 1 の M1〜M9 と C1、cycle 2 の N1〜N5、N4b、N4c、N8a〜N8c、N9)は 69908490 と 2d37fa32 の報告のとおりで、その箇所の production code は cycle 3 で変わっていない(`git log def8d2a3^..HEAD -- internal/org/reseat.go internal/cli/org_reseat.go` は def8d2a3、df8f674e、b142944d の 3 つだけ)。

| AC | テスト | 変異 |
| --- | --- | --- |
| AC-1 3 席を止め、leader → implementer → reviewer の順に立てる | `TestReseat_StopsAndStartsEverySeatInOrder`、`TestPlanReseat_DefaultTargetsAndOrder`、CLI の `TestOrgReseat_StopsAndStartsEverySeat` | — |
| AC-2 leader が新しい workspace の root の pane | `TestReseat_LeaderIsTheFirstPaneOfANewWorkspace` | — |
| AC-3 `--seat implementer --driver codex --model` と DECISION の行 | `TestReseat_OneSeatOnAnotherDriver`、`TestPlanReseat_DriverAndModelForOneSeat`、CLI の `TestOrgReseat_OneSeatPrintsTheDecisionForTheLeader`、`TestReseat_DecisionForTheSeatsStartedBeforeAFailure`、CLI の `TestOrgReseat_APartialFailurePrintsTheDecisionAndTheConfig`。cycle 3 で、別のコマンドに別の設定で立てられた座席を started と数えないこと(`TestReseat_ASeatAnotherCommandStartedOnOtherSettings`)と、同じ設定なら DECISION に入ること(`TestReseat_ASeatAnotherCommandStartedOnTheSameSettings`)を足した | N3、K1、K2、K3、H1 |
| AC-4 止める前の断り | `TestPlanReseat_RefusesWhatTheStartsWouldRefuse`、`TestReseat_ARefusalWritesNothing`、`TestPlanReseat_GatesRunOnceAfterTheChecks` | cycle 1 の M9 は生き残ったまま(Test gaps 1) |
| AC-5 結びつきの無い org の断り | `TestPlanReseat_RefusesAnOrgWithoutABinding`、CLI の `TestOrgReseat_OrgWithoutBindingRefused` | N4、N4c、N8a |
| AC-6 active な座席が無いときの断りと `--seat` の打ち方。打ち直すと立つ | `TestPlanReseat_NoActiveSeatNamesTheStoppedSeats`、`TestPlanReseat_NoActiveSeatWithoutStoppedOnes`、`TestPlanReseat_NoActiveSeatNamesOnlySeatsTheSeatFlagTakes`、`TestReseat_RetryAfterAFailedStart` の leader の subtest | N1、N2、N4、N4c、N8c |
| AC-7 座席の pane から打った reseat の断り | `TestPlanReseat_RefusedFromASeatPane` | — |
| AC-8 pane が無い座席の立て直し | `TestReseat_SeatsWhosePanesAreGone` | — |
| AC-9 `--dry-run` の出力 | `TestPlanReseat_DefaultTargetsAndOrder`、CLI の `TestOrgReseat_DryRunPrintsThePlanAndStartsNothing` | — |
| AC-10 `stop_failed` で何も立てない | `TestReseat_AStopThatFailsStartsNothing` | cycle 1 の C1 は生き残ったまま(Test gaps 3) |
| AC-11 フラグの組み合わせの断り | `TestPlanReseat_RefusesBadFlags`、`TestReseat_AFlagErrorTakesNoLock`、CLI の `TestOrgReseat_FlagRefusals` | — |
| AC-15 `reseat=<id>` の note と `reseated` 1 件 | `TestReseat_RecordsItsIDOnEveryStopAndStart`、`TestReseat_AStopThatFailsStartsNothing`、`TestReseat_RetryAfterAFailedStart`、`TestReseatNoteAndReseatedID`。cycle 3 の 4 本も、立てる時点で終わった reseat の `reseated`(`implementer:start_failed`、`leader:start_failed`、`implementer:start_failed,reviewer:stopped`)を固定する | — |
| AC-16 help が動詞とアカウントの移し方を言う | `TestOrgReseatHelp_SaysWhatItDoesAndHowToSwitchAccounts`(cycle 3 で 2 文を足した)。skill の 4 つの写しの文を固定するテストは無い | — |
| AC-17 `reseat.lock` と止める直前の読み直し | `TestReseat_RefusedWhileAnotherReseatHoldsTheLock`、`TestReseat_ASeatChangedBeforeItsStop`。止めてから立てるまでに別のコマンドが割り込んだ場合は cycle 3 の 4 本(上の表) | K1、K2、K3、X1、X2、H1 |
| AC-18 起動の失敗のあと、出た打ち直しで立つ | `TestReseat_RetryAfterAFailedStart`、`TestPlanReseat_SeatAcceptsASeatSpawnedInTheCurrentLife`、`TestReseatRetryCommand`、CLI の `TestOrgReseat_AFailedStartPrintsTheRetry`、`TestReseat_RetryStartsASeatWhoseFirstSpawnNeverEnded`、`TestReseat_DecisionForTheSeatsStartedBeforeAFailure`、CLI の `TestOrgReseat_APartialFailurePrintsTheDecisionAndTheConfig`。cycle 3 で、別のコマンドが立てた座席を外した打ち直し(`TestReseat_ASeatAnotherCommandStartedIsLeftOutOfTheRetry`)と、`--driver`・`--model` を持つ打ち直しがその座席を止めて立て直すこと(`TestReseat_ASeatAnotherCommandStartedOnOtherSettings`)を、どちらも打ち直しを走らせて確かめた | N1、N4、N4b、N4c、N5、N9、X1、X2 |
| AC-19 `--dry-run` は見張りも server も起動せず、何も書かない | `TestPlanReseat_DryRunStartsNoWatchOrServer`、CLI の `TestOrgReseat_DryRunPrintsThePlanAndStartsNothing` | — |
| AC-20 切れた reseat の座席が既定の対象になる。signal と SIGPIPE で終わらない | `TestPlanReseat_FinishesAReseatThatWasCutOff`、CLI の `TestOrgReseat_ARealReseatRunsThroughTheSignalHold`、`TestOrgReseatHoldSignals_KeepsTheProcessAndSaysSoOnce`、`TestOrgReseatHoldSignals_AClosedStderrDoesNotEndIt` | — |
| AC-21 独自の役割の座席の断り | `TestPlanReseat_RefusesACustomRoleSeat` | N4、N8b |

## 回帰

| Previously broken behavior | Status | Evidence |
| --- | --- | --- |
| 止めてから立てるまでに別のコマンドが別の設定で立てた座席を、reseat が started と数えた(cross-review cycle 2 の指摘 1) | Pass | `TestReseat_ASeatAnotherCommandStartedOnOtherSettings`、`TestReseat_TheLeaderAnotherCommandStartedOnAnotherModel`、`TestReseat_ASeatAnotherCommandStartedIsLeftOutOfTheRetry`、`TestReseatIdempotentMismatchErr`。K1〜K3 と H1 の下では FAIL |
| `--driver`・`--model` なしの打ち直しが、別のコマンドの座席を止めて ledger の設定(その座席がもう動いている設定)で立て直した | Pass | `TestReseat_ASeatAnotherCommandStartedIsLeftOutOfTheRetry`、`TestReseat_TheLeaderAnotherCommandStartedOnAnotherModel`。X1 の下では FAIL |
| cycle 1・cycle 2 の回帰の項(打ち直しと DECISION と `--config`、`--seat` が受け付ける座席、`StartFeature`、spawn の事前の判定、Note が空の Stop と Spawn の Details、`reseated` を古い binary が数えないこと) | Pass | org と cli の打ち直しの全体(org 1,822 本、cli 1,206 本)。その箇所のコードは cycle 3 で変わっていない |
| 取り込んだ #327(watch の usage_limited)と #325(upgrade の behavior notes) | Pass | #327 のテスト(`internal/org` の `watch_usage_limit_test.go`、`watch_usage_limit_race_test.go`、`usage_limit_screen_test.go`、`inbox_test.go`)は org の打ち直し、#325 のテスト(`internal/upgrade` の `behavior_notes_test.go` と `report_test.go`、`internal/cli` の `upgrade_v2_test.go` と `migrate_test.go`)は run-test.sh の `internal/upgrade`(3.8 s)と cli の打ち直しで通った |

## Coverage

- Statement: def8d2a3、df8f674e、b142944d が足した行を、HEAD の行番号で取った(`git diff -U0 def8d2a3^ HEAD -- internal/org/reseat.go internal/cli/org_reseat.go`。この 2 ファイルをこの範囲で変えたコミットは 3 つだけで、merge 8460bf64 は触っていない)。`-run 'Reseat'` で絞った org と cli の回の cover profile と突き合わせると(`addedcov.py`)、`reseat.go` は cover profile の block に入る足した行 20 行をすべて通った(`beforeStart` の呼び出し 1172〜1174 行、idempotent の枝と `Changed` 1188〜1197 行、`reseatIdempotentMismatchErr` 1226〜1234 行)。`org_reseat.go` の足した行は help の定数の文字列だけで、block に入る行は 0 行
- Branch: Go の標準ツールに branch coverage は無いので測っていない。`!seat.Overridden` の真と偽は、`TestReseat_ASeatAnotherCommandStartedIsLeftOutOfTheRetry`・`TestReseat_TheLeaderAnotherCommandStartedOnAnotherModel`(偽の側、`Changed` を立てる)と `TestReseat_ASeatAnotherCommandStartedOnOtherSettings`(真の側、立てない)が両方通る。X1 と X2 がそれぞれの側で死んだ
- Notes: 名前で絞った回なので、パッケージ全体の数字(org 16.4%、cli 8.8%)は意味を持たない

## 変異

cycle 3 の直しが足した判定とフックに 6 個の変異を当てた。各変異は文字列の置き換え 1 つで、置き換える文字列がファイルに 1 回だけ現れ、置き換えたあとのファイルが元と違うことを確かめてから当てた(cycle 2 の `mutate.pl`)。X1 と X2 は implementer が df8f674e で当てた `m1.go`・`m2.go`、H1 は def8d2a3 の `reseat_mut.go` と同じ置き換えだが、それらは当時の `reseat.go` の写しなので使い回さず、HEAD の `reseat.go`(df8f674e から変わっていない)に当て直した。どれも `go test -count=1 -v -overlay <mutant> -run 'Reseat' ./internal/org/` で回し、build に失敗した変異は無い(`build failed` と `no space left` は 0 件)。6 個とも死んだ。回したあとの `git status --porcelain` は空だった。K1〜K3 は cycle 1 の C1(CLI の変異)と名前が重ならないように付けた id で、scratch と evidence のログでは C1〜C3 と書いてある。

| id | 変異 | 結果(落ちたテスト) | 時間 |
| --- | --- | --- | --- |
| K1 | `reseatIdempotentMismatchErr` の `got.Role == seat.Role` を `true` にする | 死んだ。`TestReseatIdempotentMismatchErr/another_role` | 8 s |
| K2 | 同じく `got.Driver == seat.Driver` を `true` にする | 死んだ。`TestReseatIdempotentMismatchErr/another_driver` | 8 s |
| K3 | 同じく `got.Model == seat.Model` を `true` にする | 死んだ。`TestReseatIdempotentMismatchErr/another_model`、`TestReseat_TheLeaderAnotherCommandStartedOnAnotherModel`、`TestReseat_ASeatAnotherCommandStartedIsLeftOutOfTheRetry` | 7 s |
| X1 | `r.res.Changed = seat.SeatID` を消す | 死んだ。`TestReseat_TheLeaderAnotherCommandStartedOnAnotherModel`、`TestReseat_ASeatAnotherCommandStartedIsLeftOutOfTheRetry` | 7 s |
| X2 | `if !seat.Overridden` を `if true` にする | 死んだ。`TestReseat_ASeatAnotherCommandStartedOnOtherSettings` | 8 s |
| H1 | `beforeStart` の呼び出しを `false && hook != nil` で止める | 死んだ。cycle 3 の 4 本(`OnOtherSettings`、`OnTheSameSettings` の 2 つの subtest、`TheLeaderAnotherCommandStartedOnAnotherModel`、`IsLeftOutOfTheRetry`) | 7 s |

K1 と K2 を落とすのは `TestReseatIdempotentMismatchErr` の表だけである。reseat を通すテストは、どれも model が違う割り込み(`OnOtherSettings` は driver と model の両方)なので、role か driver だけを外した変異では結果が変わらない。比べる場所は `reseatIdempotentMismatchErr` の 1 か所で、`startAll` はその戻りだけを見るので、表のテストで足りると判断した(Test gaps 10)。

## Failure analysis

| Test | Error | Root cause | Proposed fix |
| --- | --- | --- | --- |
| `internal/org` と `internal/cli` の全体(run-test.sh の Go の段) | `panic: test timed out after 30m0s`(org は `TestMerge_RefusedCleanupKeepsTheMerge`、cli は `TestOrgLedger_RelativePersonalStateDirFailsUntilStateDirIsPassed` の途中) | 環境。パッケージ全体の時間が 30 分を超えた。load average 4.8〜23.4 で、ほかのセッションの org のテストと重なった。走っていた 2 本は reseat の変更が触っていないテストで、同じ timeout の打ち直しで両方のパッケージが通った | なし。打ち直しの結果を採った |

run-test.sh の shell の段、org と cli の打ち直し、`-race` の回、coverage の回で落ちたテストは無い。変異の FAIL は、上の節の意図した落ち方である。

## Test gaps

cycle 1 の Test gaps 1〜8 は、cycle 3 の直しでは閉じていない。1〜3 は `docs/tech-debt/README.md` の reseat の Test gaps の行に載っている。

1. 非 leader の reseat の worktree の branch の検査(`reseatSeatsStartErr` の `checkFeatureWorktreeReuse`)を、leader を含まない reseat で通るテストが無い(cycle 1 の M9)。`TestPlanReseat_RefusesWhatTheStartsWouldRefuse` の「the worktree has another branch checked out」は既定の対象(leader を含む)の行である
2. Note が空の spawn の Details を完全一致で固定するテストが無い(cycle 1 の M68)
3. CLI の `stop failed:`・`left running:`・`left as it is:`・`not started:` の行を出すテストが無い(cycle 1 の C1)
4. 検査のあと止める前に org が disband された場合のテストが無い(cycle 1 の probe P4 で確かめた)
5. I/O の失敗の戻りは行の coverage に入らない(`org_reseat.go` の `filepath.Abs` の失敗を返す行など)
6. `dryRunSpawn` の note(cycle 1 の M4)は届かない行で、固定されていない
7. AC-12 の実機は `docs/evidence/org-reseat-live-2026-11-12.md` の担当で、cycle 3 でも繰り返していない。実機の binary は dd204370 で、cycle 1 の P9・P10、SIGPIPE の直し、cycle 2 と cycle 3 の直しより前である
8. AC-16 の skill の 4 つの写しの文はテストで固定していない
9. run-test.sh の 1 回の実行で Go の全パッケージが通った記録は、cycle 2 に続いて cycle 3 にも無い。org と cli は 30 分の時間切れのあと、2 つだけで打ち直した。ほかの 8 パッケージは run-test.sh の回で通った(うち 4 つは `(cached)`)
10. 立てる時点の検出を reseat を通して確かめるテストは、どれも model の違う割り込みである。role だけ、driver だけが違う割り込みは `TestReseatIdempotentMismatchErr` の表だけが見る(K1、K2)
11. 割り込みは `reseatTestHooks.beforeStart` のフックで、reseat と同じ goroutine の中で、立てる直前に 1 回だけ起こしている。別のプロセスの spawn が本当に並んで走る窓は再現しておらず、`-race` の回もこの窓を通っていない
12. 立てる時点で終わった reseat の CLI の出力(`start failed:` の行、エラーの文、打ち直しが空のときに `to start the seats left, run:` を出さないこと)を見る CLI のテストは無い。フックが `internal/org` の中にあり、CLI のテストからは割り込めない。`Retry`、`Changed`、`Err` の値は org のテストが固定し、CLI がそれを出す部分は cycle 2 までのテスト(`TestOrgReseat_AFailedStartPrintsTheRetry` など)が通っている
13. b142944d の help の 2 文は、文ごとの部分一致で固定している。文の中の語を変える変異は当てていない

## Verdict

- Pass: run-test.sh の shell の段(45 本)と Go の 8 パッケージ、時間切れのあと同じ 30 分の timeout で打ち直した org と cli、`-race` の回、coverage の回がすべて通った。cycle 3 の直し 3 件はどれもテストが固定し(b142944d は文の pin)、直しに当てた変異 6 個はすべて死んだ。def8d2a3 と df8f674e が足した行のうち cover profile の block に入る 20 行はすべて通った。cycle 1 の Test gaps は残っている
- Fail: 0
- Blocked: なし
