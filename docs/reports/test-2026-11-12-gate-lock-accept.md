# Test report: gate-lock-accept

- Date: 2026-11-12(有効な run-test.sh は 2 回目で、22:10:29 から 22:35:23 JST。1 回目は 21:41:39 から 22:04:45 JST で、共有ディスクの空きが尽きたため無効)
- Plan: `docs/plans/active/2026-11-12-gate-lock-accept.md`(`## Test plan`、AC1〜AC14)
- Tester: tester サブエージェント(`/test`、pipeline の cycle 1。`cycle-count.json` は `{"cycle": 1}`)
- Scope: `git diff origin/main...HEAD`(HEAD `d5ef9a82`、74 ファイル、+6692/-518)。コードの commit は S1 `3e027e09`、S2 `7525c36b`、S3 `6e4c5f48`、S4 `d23ba892`、S5 `4b1eafaa`。run-test.sh は changed scope で呼んだが、`templates/base/ralph.toml` が言語に分類されないので full に落ち(`Language scope: full fallback (unclassified:templates/base/ralph.toml)`)、shell の 45 本と golang を回した。worktree の production code とテストには手を入れていない。変異と probe は `git archive HEAD` の写し(`/private/tmp` の下)で当て、probe のテストは `go test -overlay` で足した
- Evidence: `docs/evidence/test-2026-11-12-gate-lock-accept.log`(`.gitignore` の `docs/evidence/*.log` に当たるのでコミットしない)。2 回目の run-test.sh の全出力、1 回目の要約、変異の定義と結果、probe のソースと出力、changed-line coverage、`-race` の結果を入れた。wrapper のログは `docs/evidence/verify-2026-11-12-131029.log`(2 回目)と `docs/evidence/verify-2026-11-12-124140.log`(1 回目)
- 起動の仕方: run-test.sh は Bash の 600 秒の上限を超えるので、`perl` の fork と `setsid` で切り離し(INT・HUP・QUIT・TERM は DEFAULT、標準入出力は `/dev/null`)、`caffeinate -w` を 175 秒ずつ繰り返して終わりを待った。shell の SKIP の行は前の PR の回と同じ 2 行だった

## Test execution

| Suite / Command | Tests | Passed | Failed | Skipped | Duration |
| --- | --- | --- | --- | --- | --- |
| `./scripts/run-test.sh` 2 回目(rc 0、`==> All verifiers passed.`) | shell 45 本 + Go 10 パッケージ | shell 45 本 OK(PASS を含む行 5,195)、Go 10 パッケージ ok | 0(`--- FAIL` の行 0、`no space left` の行 0) | 2 行 | 1,494 秒 |
| └ Go: `internal/org` | 上位 1,724 本 | ok | 0 | — | 805.2 秒 |
| └ Go: `internal/cli` | 上位 1,116 本 | ok | 0 | — | 605.5 秒 |
| └ Go: `internal/config` | 上位 116 本 | ok | 0 | — | 11.8 秒 |
| └ Go: `cmd/ralph`、`internal/gitroot`、`internal/insights`、`internal/org/driver`、`internal/org/protocol`、`internal/scaffold`、`internal/upgrade` | — | ok `(cached)`。同じ HEAD の 1 回目でディスクが尽きる前に通った結果 | 0 | — | — |
| `./scripts/run-test.sh` 1 回目(rc 1、無効) | 同上 | shell 45 本 OK、Go 8 パッケージ ok | org と cli が FAIL。上位の `--- FAIL` 522 行、`no space left` 1,058 行。原因はすべて ENOSPC(下の Failure analysis) | 2 行 | 1,386 秒 |
| `-race -count=3`、org: `GateLock\|GateAccept\|Load\|TwoGates\|LeaderActivity\|MergeOrder_Accept\|AcceptedBranch\|AcceptanceOf\|AGateRecordAfterTheAcceptance` | 上位 71 本 × 3 | ok | 0 | 0 | 67 秒。`DATA RACE` 0 |
| `-race -count=2`、cli: `OrgGateAccept\|OrgGate_WaitsForGateLock\|OrgGate_PassesMaxLoad\|OrgGate_PrintsTheLoadLine\|OrgMerge_MergesAnAcceptedBranch\|OrgMergeOrder_HelpNamesTheAcceptedClass` | 上位 10 本 × 2 | ok | 0 | 0 | 12 秒。`DATA RACE` 0 |
| coverage: 名前で絞った回(下の Coverage) | org、cli、config | ok | 0 | — | 474.8、207.4、14.1 秒 |
| 変異: production code の 45 個(下の節) | 45 | 39 個が死んだ | — | — | 1 個 1〜66 秒(C3 だけ 3 分の timeout) |
| probe: 生き残った 4 個を落とす行(overlay で足した 4 本、コミットしない) | 4 本 × HEAD と 4 個 | HEAD で 4 本とも PASS、4 個はそれぞれ自分の probe で FAIL | — | — | 1 回 1〜31 秒(build を除く) |

上位の本数は `go test -list .` で数えた。Skipped の 2 行は前の PR の回と同じで、`tests/test-plan-visual.sh` の browser detection と、`tests/test-secret-scan-branch.sh` の git 2.41 より古い本物の git の case である。どちらも手元の環境によるもので、この diff と関係がない。

2 回目のあいだ、1 分の load は 7.6 から 15.3 だった(`uptime` の標本)。org は 805 秒で、30 分の上限の 45% だった。2 回目で落ちたテストは無い。

## Failure analysis

| Test | Error | Root cause | Proposed fix |
| --- | --- | --- | --- |
| 1 回目の org と cli の 522 本(最初は cli の `TestOrgResume_WarnsWhenTheDirectorsCannotBeRead`、22:03 ごろ) | `TempDir: mkdir ...: no space left on device`、`fatal: unable to write new index file`、`remote unpack failed`、`couldn't write ... .git/refs/heads/wt.lock`、`testing: can't write .../testlog.txt` | 環境の失敗。共有の Data volume の空きが 22:03 ごろに 117〜220 MB まで減った(`~/Library/Caches/go-build` が 58 GB)。`no space left` を出力に含まない 6 本も、git の書き込みの失敗か herdr stub の `No space left on device` で、同じ原因だった。コードの失敗ではない | 別のセッションが go-build のキャッシュを消して空きが約 40 GB に戻ったあと、22:10 に run-test.sh を最初から回し直し、rc 0 で通った。キャッシュは自分では消していない |

2 回目は何も落ちなかったので、負荷による flaky の打ち直しはしていない。変異の回のうち 22:01〜22:03 の C8・C9・C10・D1・D2・E1 は、同じ ENOSPC で build に失敗した(`compile: writing output: ... no space left on device`)。死んだとは数えず、空きが戻ってから回し直した。C5 のログにも `no space left` が 1 行あった(`TestWatch_ReservedScope_AStatusFileWithoutTheScopeBaselineLoads` の prompt file の書き込み)ので、C5 も空きのある状態で回し直し、`max_load` のテスト 2 本だけで落ちることを確かめた。

## 受け入れ条件とテスト

どのテストも 2 回目の run-test.sh で PASS した。「変異」の列は、そのテストが落とした変異の id(下の節)である。

| AC | テスト(ファイル) | 変異 |
| --- | --- | --- |
| AC1 2 本の gate は重ならない。待つ間の stderr と log の見出し | `TestGate_TwoGatesRunTheirScriptsOneAtATime`(`gate_lock_gate_test.go`、実際の git の worktree で 2 本の Gate を goroutine で同時に回す)、`TestAcquireGateLock_WaitsForTheHolderAndSaysWhoItIs`・`_SaysSoWhenTheHolderChanges`・`_RemindsWhileTheSameHolderHoldsIt`・`_SaysAMarkerIsNotWrittenYet`、`TestGateLockHolder`(`gate_lock_test.go`)、見出しの `# waited` は `TestGate_WaitsForGateLockAndGatesTheHeadItFindsAfterTheWait`、cli の `TestOrgGate_WaitsForGateLockOnTheCommandsStderr` | G1(gate.lock の待ちが塞がらない)で `TwoGates` が 3 回とも「the two gates' scripts overlapped」で落ちた |
| AC2 待ちを ctx で切ると何も記録せず log も作らない | `TestGate_InterruptedWhileWaitingForGateLockRecordsNothing`、`TestAcquireGateLock_ACancelledWaitHoldsAndWritesNothing` | 当てていない(S1 の実装者が 4 個当てて全部落ちたと報告) |
| AC3 待ったあとに org と worktree を読み直す | `TestGate_WaitsForGateLockAndGatesTheHeadItFindsAfterTheWait`(待つ間に HEAD を動かす)、`TestGate_DisbandedWhileWaitingForGateLockRunsNothing` | 同上 |
| AC4 local の regate は regate.lock の前に gate.lock を取る。ci の regate は取らない | `TestRegate_WaitsForGateLockWithRegateLockFree`、`TestRegate_MarksGateLockWithTheBranchAndThenItsOrg`、`TestRegate_ARefusalOrAnInterruptReleasesGateLock`、`TestRegate_ACIRegateTakesNoGateLock`(`regate_test.go`) | R1(gate.lock を待つ間 regate.lock を持つ)で `WaitsForGateLockWithRegateLockFree` が「regate.lock held = true ... while the regate waits for gate.lock」で落ちた |
| AC5 負荷の読み取り、log の見出しと結び、記録の鍵、`unknown`、前の形の Details | `TestParseSysctlLoadavg`、`TestParseProcLoadavg`、`TestReadGateLoad_ReadsThisMachine`、`TestFormatGateLoad`、`TestGateLoadSampler_KeepsTheHighestAndStops`、`TestGateLoadSampler_EndsWithCtx`、`TestGateRecordDetails_Load`、`TestGate_RecordsTheLoadAroundTheScripts`、`TestGate_RecordsAnUnknownLoad`(`gate_load_test.go`)、cli の `TestOrgGate_PrintsTheLoadLine` | C4、C5 が死んだ。C8(終わりの値を最大に入れない)は生き残った(Test gaps 3) |
| AC6 `max_load` の待ち、30 分の上限、ctx、config | `TestWaitGateMaxLoad`(表)、`TestWaitGateMaxLoad_StartsAnywayAfterTheCap`、`TestWaitGateMaxLoad_CtxEndsTheWait`、`TestGate_MaxLoadStartsAnywayAfterTheCap`、`TestGate_MaxLoadNotAboveDoesNotWait`、`TestGate_MaxLoadWaitInterrupted`、`TestRegate_MaxLoadWaitsBeforeRegateLock`、`TestRegateCI_NoLoadAndNoMaxLoadWait`、config の `TestLoad_OrgGateMaxLoad`、cli の `TestOrgGate_PassesMaxLoadFromTheConfig` | C3(上限で抜けない。3 分の timeout で落ちた)、C4、C5、C10、D2 が死んだ。C1、C2、D1 は生き残り、C9 は効き目の上で同じだった(Test gaps 1、2、4、5) |
| AC7 `gate accept` の記録と断り、同時の 2 本、打ち直された失敗 | `TestGateAccept_RecordsTheLatestFailure`、`_RefusalsRecordNothing`、`_ReasonOf500CharactersIsRecorded`、`_ALogOfAFailureANewerOneReplacedIsRefused`、`_TwoAtOnceRecordOne`、`_SeatPaneTripwire`、`_ManifestThatCannotBeReadIsAnError`、`TestGateAcceptanceDetails_RoundTrip`、`TestGateAcceptanceFromDetails_Malformed`(`gate_accept_test.go`)、`TestApproveTripwire_TextIsApprovesSentence`、cli の `TestOrgGateAccept_*` 4 本 | A12〜A22、A17、A18、A20 が死んだ。A18(manifest lock を外す)では、`TwoAtOnceRecordOne` が `-count=5` の 5 回とも、2 本とも記録した結果になって落ちた |
| AC14 gate.lock は scripts に継がれる | `TestGate_AScriptKeepsTheLockAfterItsRalphIsKilled`(`gate_lock_test.go`。テストの binary を子の process として起こし、`TestGateLockHelperProcess` が gate.lock を取って script を走らせ、子だけを SIGKILL する)、`TestGateLock_ReleaseFreesTheLockAChildStillHasOpen` | E1(`cmd.ExtraFiles` に渡さない)で「gate.lock was free while the script still ran after its ralph was killed」で落ちた |
| AC8 merge-order の `accepted`、JSON、後ろの記録で消える、regate の merge commit | `TestMergeOrder_AcceptedFailureIsListedRightAfterReady`、`_AcceptedFailureIsClassifiedAsAPassWouldBe`、`_AcceptanceOfRegatesMergeCommitAppliesOnlyToThatHead`、`_AGateRecordAfterTheAcceptanceVoidsIt`、`_AnAcceptanceOfAnotherRecordDoesNotApply`、`_SeveralBranchesNoteCountsTheAcceptedOnes`(`mergeorder_test.go`)、`TestGateAccept_MergeOrderListsTheBranchAsAccepted`、cli の `TestOrgGateAccept_MergeOrderListsTheBranchAsAccepted`、`TestOrgMergeOrder_HelpNamesTheAcceptedClass` | A1〜A8(`accepts` の比較を 1 つずつ外す)、B1〜B10 が死んだ。A11 は同値(Test gaps 5) |
| AC9 merge は `accepted` を受け付け、`pr_merged` に `gate=accepted` | `TestMerge_MergesAnAcceptedBranchAndRecordsGateAccepted`、`TestCheckMerge_PassesForAnAcceptedBranch`、cli の `TestOrgMerge_MergesAnAcceptedBranch` | B12、B13 が死んだ |
| AC10 `leaderActivityEventCount` は `gate_accepted` を数えない | `TestLeaderActivityEventCount_GateEvents` の「gate_accepted in the watched org does not count」の行 | 当てていない |
| AC11 `max_load` の既定値とテンプレート | `TestDefault_OrgGate`、`TestLoad_OrgGateMaxLoad`、`TestOrgTemplateDrift_EachKeyIsComparedOnBothSides`、`TestDefaultsLockStep`(`internal/config`) | 当てていない |
| AC12 skill と reviewer の役割 prompt | `TestDirectorSkill_PinsTheAcceptedGateFailureMerge`、`TestOrgSkill_PinsTheGateLockAndTheAcceptedMerge`、`TestRenderRolePrompt_Reviewer_GateTaskRunsTheTaskCommand`。4 つの写しの一致は /verify が確かめた | 当てていない |
| AC13 help | `TestOrgGateHelp`、`TestOrgGateAccept_Help`、`TestOrgRegate_Help`、`TestOrgMergeOrder_HelpNamesTheAcceptedClass`。AGENTS.md、仕様、tech-debt は /verify | 当てていない |

依頼で名指しされた 5 本は、どれも 2 回目で通り、変異でも感度を確かめた。AC14 の実 process の SIGKILL は E1、同時の accept は A18、打ち直しのあとの log の不一致は A12 と B1、2 本の gate が重ならないことは G1、regate が gate.lock を待つ間に regate.lock が空いていることは R1 で落ちた。

計画の Test plan の Edge cases のうち、空の印と壊れた印は `TestGateLockHolder` と `TestAcquireGateLock_SaysAMarkerIsNotWrittenYet`、`--reason` の改行と制御文字は `TestGateAccept_RefusalsRecordNothing` と `TestGateAcceptanceFromDetails_Malformed`、`max_load` が設定され負荷が読めない場合は `TestWaitGateMaxLoad` の「unreadable」の行と `TestGate_RecordsAnUnknownLoad` にある。

## Coverage

- Statement: この diff が足した Go のコード(テストを除く)の行のうち、cover profile の block に入る 851 行の 795 行(93.4%)を通った(`git diff -U0 origin/main...HEAD -- internal` の足した行と、上の絞った回の profile を突き合わせた)。`mergeorder.go` 48/48、`merge.go` 9/9、`merge_run.go` 7/7、`approve.go` 5/5、`gate_accept.go` 145/151、`gate_load.go` 141/156、`gate_lock.go` 69/80、`lockfile.go` 16/21、`gate.go` 30/32、`gate_run.go` 16/22、`gate_record.go` 24/26、`regate.go` 20/24、cli の `org_gate.go` 75/75、`org_gate_accept.go` 89/94、`org_merge.go` 21/21、`org_merge_order.go` 30/30、`org_regate.go` 26/26、config 4/4。名前で絞った回なので、パッケージ全体の数字(org 70.6%、cli 33.8%、config 96.7%)は意味を持たない
- Branch: Go の標準ツールに branch coverage は無いので測っていない
- Function: 測っていない(行の突き合わせで足りた)
- Notes: 通らなかった 56 行の内訳。I/O の失敗(gate.lock の印の書き込み、`gate_accepted` の追記、manifest lock、gate log の作成、manifest の読み取り)が多い。`gate_load.go` の 85〜91 行は linux と他の OS の分岐で、darwin の手元では通らない。目立つのは 3 か所で、`gate_load.go:276`(終わりの値を最大に入れる。C8 と同じ穴)、`gate_accept.go:326-327`(知らない reason の失敗を断る行。表に行が無い)、`gate_load.go:358` と `:383`(読み取りの直後に ctx が切れている場合)である

## 変異と probe

`git archive HEAD` の写しに 1 個ずつ文字列の置き換えで当て(置き換える文字列はファイルに 1 回だけあることを確かめた)、`go test -c` で作った binary を名前で絞って回し、元に戻して byte で比べた。最後に写しの `internal/` が worktree と同じことを `diff -rq` で確かめた。

| id | 変えたところ | 結果 |
| --- | --- | --- |
| A1〜A8 | `GateAcceptance.accepts` の比較(log、ts、org、head、base、base_ref、event、reason)を 1 つずつ `true` に | 8 個とも `TestMergeOrder_AnAcceptanceOfAnotherRecordDoesNotApply` で死んだ |
| A11 | `accepts` の `a.Branch == r.Branch` を `true` に | 生き残った。同値: 呼び出しは `byBranch[a.Branch]` で枝を引き、読める記録の branch はその枝と同じ。読めない記録は head が空なので A4 の比較で外れる |
| A12、A13、A19、A21 | `gateAcceptFailure` の断り(log の不一致、受け入れ済み、head_moved/tree_changed、log の無い記録)を外す | 死んだ(`_ALogOfAFailureANewerOneReplacedIsRefused`、`_RefusalsRecordNothing`、`_TwoAtOnceRecordOne`) |
| A14、A15、A16、A22 | `gateAcceptanceFromDetails` の検査(note の後ろの余り、重複した鍵、log が `"` で始まること、herdr_pane の値)を外す | 4 個とも `TestGateAcceptanceFromDetails_Malformed` で死んだ |
| A17、A20 | note を trim しない、`HerdrPane` を逆に | 死んだ |
| A18 | `GateAccept` の manifest lock を外す | `TwoAtOnceRecordOne` が 5 回とも落ちた |
| B1、B2、B3 | 新しい gate 記録(読める、読めない)と後ろの壊れた行で accept を消さない | 死んだ。B2 と B3 は JSON の `gate.accepted` でしか見えない形だが、`_AGateRecordAfterTheAcceptanceVoidsIt` の行が落とした |
| B4〜B10 | accepts を見ない、注記の数え方 2 つ、regate の detail、分類の順、JSON の `accepted`、`a.OrgID` を入れない | 死んだ |
| B12、B13 | merge が `accepted` を断る、`pr_merged` に `gate=accepted` を書かない | 死んだ |
| C1 | `waitGateMaxLoad` の待ちの中の `<=` を `<` に(待ったあと上限と同じ値) | 生き残った。probe で死んだ |
| C2 | 最後の sleep を上限の残りに切り詰めない | 生き残った。probe で死んだ |
| C3 | 30 分の上限で抜けない | 死んだ(`_StartsAnywayAfterTheCap` が 3 分の timeout) |
| C4、C5 | 記録の `max_load` を 0 に、待った時間を残さない | 死んだ |
| C8 | `finish` で終わりの値を最大に入れない | 生き残った。probe で死んだ |
| C9 | Gate が `waitGateMaxLoad` のエラーを無視する | 生き残った。効き目の上で同じ: 次の `gateCheck` が切れた ctx で同じ「interrupted ... before the scripts ran」を返す |
| C10 | regate が `max_load` を渡さない | 死んだ |
| D1 | cli の regate が config の `max_load` を渡さない | 生き残った。probe で死んだ |
| D2 | cli の gate が config の `max_load` を渡さない | `TestOrgGate_PassesMaxLoadFromTheConfig` で死んだ |
| E1 | gate.lock を scripts に `ExtraFiles` で渡さない | `TestGate_AScriptKeepsTheLockAfterItsRalphIsKilled` で死んだ |
| G1 | `waitHeldFileLock` が塞がらない(いつも取れたことにする) | `TestGate_TwoGatesRunTheirScriptsOneAtATime` で死んだ |
| R1 | regate が gate.lock を待つ間 regate.lock を持つ | `TestRegate_WaitsForGateLockWithRegateLockFree` で死んだ |

45 個のうち 39 個が死んだ。生き残った 6 個のうち A11 と C9 は上のとおり同値か、効き目の上で同じである。残りの 4 個(C1、C2、C8、D1)は、overlay で足した probe(`TestProbe_WaitGateMaxLoad_EqualAfterAWaitStarts`、`TestProbe_WaitGateMaxLoad_CapCutsALongPoll`、`TestProbe_GateRunLoad_TheEndCountsTowardTheHighest`、cli の `TestProbe_OrgRegate_PassesMaxLoadFromTheConfig`)が HEAD で通り、それぞれの変異で落ちた。probe のソースは evidence のログにある。

## Regression checks

| Previously broken behavior | Status | Evidence |
| --- | --- | --- |
| 既存の gate・regate・merge-order・merge・approve・watch のテスト | 2 回目で org と cli のパッケージ全体が ok | `docs/evidence/test-2026-11-12-gate-lock-accept.log` の先頭 |
| `TestRegate_Interrupted` と `TestRegate_AnInterruptBeforeTheRecordUndoesTheMerge` の閾値を 2 から 3 に変えた | 弱めていない。gate.lock の印で時計の読み取りが 1 回増え、切るのは今も event の TS の読み取り(3 回目)で、`reads != 3` で回数も固定した | `git diff origin/main...HEAD -- internal/org/regate_test.go` |
| Details と出力を全文で比べる既存テストの更新(S2 で 21 本、S3 で固定文字列 5 か所) | 2 回目で通った。計画の Progress には、固定した負荷の鍵に合わせて直し、CLI の e2e は値の形を確かめてから置き換えて比べる、とある。私は 1 本ずつは読んでいない | 計画の Progress の S2 と S3 |

## Test gaps

1. `waitGateMaxLoad` の待ちの中で、上限と同じ値になった場合の行が無い(C1)。最初の読み取りの「equal」の行はあるが、待ったあとの比較は `<` にしても落ちない。`TestWaitGateMaxLoad` に「above, then equal」(6 → 4、上限 4)の行を足せば落ちる
2. 最後の sleep を上限の残りに切り詰めることを確かめる行が無い(C2)。切り詰めないと、待ちは上限を最大で 1 回の poll(30 秒)ぶん超える。poll を上限より長くした probe(poll 1 時間、上限 50 ms)で落ちる
3. `finish` が終わりの値を最大に入れることを確かめる行が無い(C8、`gate_load.go:276` は coverage でも通らない)。始めが 3 で終わりが 7 の場合と、始めが読めず終わりが 5 の場合で落ちる。issue の受け入れ条件 2 の「最大」に関わるので、足すことを勧める
4. cli の `ralph org regate` が config の `max_load` を渡すことを確かめるテストが無い(D1)。gate には `TestOrgGate_PassesMaxLoadFromTheConfig` がある。regate の fixture で `--config` に `max_load = 100000` を書き、`gate_passed` の Details に `max_load=100000` が入ることを見る probe で落ちる
5. 同値か、効き目の上で同じ変異: A11 と C9(上の表)。テストは要らない
6. `gateAcceptFailure` の「知らない reason の失敗」を断る行(`gate_accept.go:326-327`)は、今の binary が書く reason では届かない。前の版や後の版が書いた記録を読む場合に備えた行で、テストの行は無い
7. 負荷の読み取りの linux の分岐(`/proc/loadavg` の読み取り)は解析(`TestParseProcLoadavg`)だけで、ファイルを読む行は darwin の手元では通らない。CI(`.github/workflows/verify.yml`)は `ubuntu-latest` で回るので、そこで通るはずだが、CI の結果はまだ見ていない
8. 変異を当てていない AC: AC2、AC3(S1 の実装者が 4 個当てたと報告)、AC10、AC11、AC12、AC13

1〜4 はどれも LOW で、PR を止めるものではない。probe の 4 本は commit していない。足すかどうかは lead の判断に任せる。

## Verdict

- Pass: 45 本の shell テストと Go 10 パッケージ(2 回目の `./scripts/run-test.sh`、rc 0、`==> All verifiers passed.`)。`-race` の 81 本も通り、`DATA RACE` は 0
- Fail: 0(1 回目の 522 本はディスクの空きが尽きたことによる環境の失敗で、空きが戻ってからの 2 回目では 0 だった。コードによる失敗は無い)
- Blocked: なし
