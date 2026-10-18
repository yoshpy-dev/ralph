# Test report: org-director-wake

- Date: 2026-10-18(JST 18:42〜20:07)
- Plan: docs/plans/active/2026-10-18-org-director-wake.md(8 段目の 8b-1、仕様 FR-10 の push で director を起こす手順)
- Tester: tester subagent (Claude Opus 5.5)。パイプラインの 1 回目
- Scope: HEAD 4331358c(base 9e92d928、20 コミット、55 ファイル、+7,338/-327)。behavioral test だけを実行した(静的解析は /verify の担当)。環境は macOS(arm64、Darwin 25.5.0)、go1.26.0 と go1.25.8(`GOTOOLCHAIN`)、git 2.49.0。実行中の load average は 6〜240 で、大きい値の多くは別のセッションの分
- Evidence: `docs/evidence/test-2026-10-18-org-director-wake.log`(run-test.sh、go1.25.8、`-race` の全出力と、揺れたテストの繰り返し)、`docs/evidence/test-2026-10-18-org-director-wake-mutation.txt`(mutant 83 個の表、パッケージ全体での再実行、probe、同値の判断)。wrapper の全ログは `docs/evidence/verify-2026-10-18-094238.log`。`.log` の 2 つは gitignore の対象なのでコミットしない。`<scratch>` はセッションの scratchpad、`~` は `$HOME`
- テストもコードも変えていない。mutant と probe のテストは scratchpad の `git archive` の写しに置き、`go test -overlay` で読ませた

## Test execution

| Suite / Command | Tests | Passed | Failed | Skipped | Duration |
| --- | --- | --- | --- | --- | --- |
| `./scripts/run-test.sh`(changed を求め、`scripts/ralph-config.sh` が分類されず full に広がった。go1.26.0) | shell 41 本、Go 8 パッケージ | shell 41 本すべて OK、Go 8 パッケージすべて ok | 0 | shell 1(下) | 2,532 秒、rc 0 |
| `GOTOOLCHAIN=go1.25.8 go test ./... -count=1 -timeout 30m -v` | top-level 2,091、subtest を含む PASS の行 5,586 | top-level 2,088 | 0 | 3(下) | 1,082 秒(cli 920.3 s、org 1075.3 s)、rc 0 |
| `go test -race -count=1 -timeout 30m ./internal/org/... ./internal/cli/... ./internal/config/...`(go1.26.0) | 5 パッケージ、top-level 1,909 | top-level 1,908、PASS の行 5,386 | 0 | 1 | 1,016 秒(org 1001.7 s、cli 877.3 s)、DATA RACE 0 |
| うちこの差分で足したテスト | top-level 67 | 上の 3 回のすべてで PASS | 0 | 0 | - |
| 揺れたテストの繰り返し: `-run 'TestWakeDirector_AFailedWakeIsNotTriedAgainForFiveMinutes$' -count=40`(load 15〜240) | 40 | 39 | 1(下の Failure analysis) | 0 | 111 秒、rc 1 |
| mutation(`go test -overlay`。壊したファイルは org 62、cli 14、config 7) | mutant 83 | 赤 77 | 生き残り 6(G5 を含む。下) | - | 合計 4,044 秒(2 本ずつ並べて 18:54〜19:29) |
| 生き残りを org と cli のパッケージ全体で(B1 = G7、B2 = H6 + M11 + O3 + G5) | 2 組 | 赤 0(B2 の 1 回目は regate の揺れで落ち、数えない) | 生き残り 2 組 | - | B1 org 602 s・cli 409 s、B2 org 505 s・cli 326 s |
| probe のテスト 6 本(overlay だけ、コミットしない) | 6 | HEAD ですべて PASS | 0 | 0 | - |

- shell は 41 本すべてが `OK` で終わり、`FAIL:` の行はどれも 0 だった。Skipped は前からある `tests/test-secret-scan-branch.sh` の 1 件(git が 2.49.0 なので、2.41 より古い git の場合を飛ばす)。この差分は shell のテストを変えていない
- `run-test.sh` の中の Go は cli(428.1 s)、config、org(585.8 s)、org/driver を流し直し、insights、org/protocol、scaffold、upgrade は `(cached)` だった。cache の鍵は toolchain を含むので、この 4 つは同じ入力で go1.26.0 で通った結果になる。`-count=1` では go1.25.8 の回で 8 パッケージすべて、`-race` の回で org、org/driver、org/protocol、cli、config を流し直した
- 3 回(run-test.sh、go1.25.8、`-race`)はどれも `--- FAIL` と `^FAIL` の行が 0 だった。run-test.sh と go1.25.8 の回、run-test.sh と `-race` の回は同じ時間に並べて走らせた
- Skipped の 3 件は `TestEscalate_NoDesktopNotificationOffDarwinIsSkipped`(org。darwin で飛ばす作り)と scaffold の `TestBaseFS_WithMockFS`、`TestAvailablePacks_WithMockFS` で、どれもこの差分と関係ない。`-race` の 1 件は前者
- 差分で足した `func Test` は 67 本(org 41、cli 16、config 10。org は deadline・inbox・prompts・watch の分を含む)で、消したものはない。`defaults_sync_test.go` は既存のテストの期待値に wake を足した変更
- 前に揺れたテスト(`TestRunDoctorOpts_ProbeModelsFalse_NoSubprocess`、`TestRunDoctorFull_StrictFlipsExitCode_DriftedCore`、`TestRunWatcher_TimeoutIndependentOfSmallInterval`、`TestOrgGate_InterruptStopsTheScriptAndRecordsNothing`、`TestRegate_FetchFailures`)は、上の 3 回では落ちなかった。`TestRegate_FetchFailures` は mutation の B2 の 1 回目で落ちた(下)

## Coverage

- Statement: `-run` で絞った回(org は `Wake|Woken|Inbox|Deadline|DirectorAttach|DirectorRead|DirectorRegistration|StartDirector|RunWatchAll|Prompt`、cli は `Wake|Woken|MainWorktreeRalphToml`、config は全体)の `-coverprofile` で、足した関数を見た
  - `internal/config` の wake の 5 関数(`ValidateDirectorWake`、`ValidateDirectorWakeNamed`、`DirectorWakePlaceholders`、`ValidateDirectorWakeCmd`、`ExpandDirectorWakeArg`)は 100%
  - `internal/org/director_wake.go` は 19 関数のうち 14 が 100%、`WakeDirector` 90.0%、`directorToWake` 81.8%、`directorWakeSender` 95.0%、`runDirectorWakeCmd` 92.9%、`typeDirectorWakeLine` 93.3%。`AppendWoken` 96.3%、`foldInbox`・`checkDirectorStartWake`・watch の `wake` は 100%
  - `internal/cli` は `withMainWorktreeDirectorWakeConfig(From)`・`checkOrgDirectorAttachWakeFlags`・`orgDirectorAttachWakeDetails`・`orgDirectorPushDetails` が 100%、`attachWake` 88.9%、`orgWatchDirectorWake` 85.7%
- Branch: 計っていない(Go の標準の道具にない)。通らなかった block は次の 9 つ
  - `director_wake.go:347`(`mem == nil` のときの既定)、`:384`(inbox の store がない)、`:389`(inbox が読めない)、`:529`(`ExpandDirectorWakeArg` のエラー。読み込みで形を確かめたあとは起きない)、`:551`(herdr の client がない)
  - `director_wake.go:492`(herdr の pane id の形が送り先に合わない)。mutant H6 が生き残った箇所で、probe P3 で閉じることを確かめた
  - `inbox.go:332`(`AppendWoken` の中の読み込みエラー)、`cli/org_director_wake.go:59`(`attachWake` の中で main worktree の `ralph.toml` が読めない。attach の外の `TestOrgWatch_AMainWorktreeRalphTomlThatDoesNotLoadIsTheError` は見張りの側で同じ読み込みエラーを通す)、`cli/org_watch.go:178` と `:184`(`filepath.Abs` と `os.Getwd` のエラー)
- Function: 上のとおり
- Notes: 数字は絞った回のもので、パッケージ全体の値ではない。どの block が通るかを見るために使った

## AC coverage

| AC | テスト | mutation |
| --- | --- | --- |
| AC1 設定 | config の `TestLoad_OrgDirectorWake`、`TestLoad_OrgDirectorWakePushWithoutWakeCmd`、`TestLoad_OrgDirectorWakeCmd`、`TestLoad_OrgDirectorWakeWrongType`、`TestValidateDirectorWake`、`TestValidateDirectorWakeNamed`、`TestValidateDirectorWakeCmd_EmptyIsValid`、`TestDirectorWakePlaceholders`、`TestExpandDirectorWakeArg`、`TestTemplateRalphToml_WakeCmdExampleIsValid`、既定値の 3 面は `TestDefaultsLockStep`(`defaults_sync_test.go`) | K1〜K7 はすべて赤 |
| AC2 attach | org の `TestValidateDirectorWakeTarget`(bell、改行、tab、ESC、DEL、C1、不正な UTF-8、256 文字は通し 257 文字は断る、`-` で始まるもの)、`TestCheckDirectorAttachWake`、`TestDirectorAttach_RecordsTheWakeAndTheWakeTarget`、`_AnEmptyWakeRecordsPull`、`_RefusesABadWakeOrTargetWithNothingWritten`、`TestDirectorRead_AWakeTargetIsReadForAPromotedSessionOnly`。cli の `TestOrgDirectorAttach_RefusesAWakeItCannotSendWithNothingWritten`、`_RecordsTheWakeAndTheWakeTarget`、`_TheReplacedPartShowsTheWakeEscaped`、`TestOrgDirectorStatus_ShowsWhereAPushWakeIsSent`(`director status`、その `--json`、`ralph status` の `director:` の行)、`_EscapesTheWakeAndTheWakeTarget` | A1〜A12 はすべて赤 |
| AC3 start | org の `TestStartDirector_RecordsTheWakeOfTheConfig`、`_ALaunchCmdStartThatPushesPassesWithAWakeCmd`、`_RefusesBeforeWritingOrCallingHerdr` の wake の行、`TestDirectorAttachHeadless_RecordsTheWake`。cli の `TestOrgDirectorStart_RecordsTheWakeAndPushesInHerdrWithNoWakeCmd`、`_ALaunchCmdThatPushesNeedsAWakeCmd` | T1〜T8 はすべて赤(確かめの順を launch_cmd や見張りの門のあとへ移す T4、T5 を含む) |
| AC4 読み方と打つ場所 | cli の `TestOrgDirectorAttach_ReadsTheWakeFromTheMainWorktree`、`TestOrgDirectorStart_ReadsTheWakeFromTheMainWorktree`、`TestWithMainWorktreeDirectorWakeConfig`、`TestOrgWatch_RunsTheWakeCmdOfTheMainWorktreeInItsRoot`(main と feature worktree に同じ相対パスの別のスクリプト)、`TestOrgWatch_AMainWorktreeRalphTomlThatDoesNotLoadIsTheError`、`TestOrgWatchDirectorWake_ReturnsTheDirectoryOfTheRalphTomlItReadFrom`。org の `TestWakeDirector_ARelativeProgramIsResolvedFromTheWakeCmdDir`、`_WithoutAnAbsoluteDirectoryTheWakeCmdFails` | M1〜M14 のうち M11 だけが生き残った(区切りのない名前を PATH で引くこと。probe P2) |
| AC5 起こす手順(wake_cmd) | org の層で端から端まで: `TestRunWatchAll_WakesTheDirectorOnceForAnEscalatedItem`(escalate、`WaitInbox` がその件で返る、次のサイクルで 1 回、その次は 0 回、argv に本文が入らない)。ほかに `TestWakeDirector_RunsTheWakeCmdOnceAndRecordsWoken`、`_OneRunNamesEveryNewItem`(2 件と 21 件以上)、`TestDirectorWakeText`、`TestDirectorWakeItems`、`TestWakeDirector_RunsNothingWithoutAnItemOrAPushRegistration`、`_AHeadlessDirectorIsWokenOnlyWhileRunning`、`_ALaunchCmdDirectorIsWokenAtItsTerminalName`、`TestRunWatchAll_DoesNotWakeAPullDirector` | S1〜S3、C1〜C5、W1〜W5 はすべて赤 |
| AC6 herdr の既定 | `TestWakeDirector_TypesIntoTheHerdrPaneOfAHeadlessDirector`(照合、send-text、Enter の順、照合の失敗、gone、wake_cmd があるときは herdr を打たない) | H1〜H9 のうち H6 だけが生き残った(記録された pane id の形。probe P3) |
| AC7 失敗と期限 | `TestWakeDirector_AFailedWakeIsNotTriedAgainForFiveMinutes`(0 でない終了、時間切れ、4 分 59 秒と 5 分の境目)、`_AFailureLineKeepsTheEndOfTheOutputEscaped`、`_AnUnusableTargetIsSaidOnceAndNothingRuns`、`_NoWayToSendIsSaidOncePerRegistration`、`_AWokenWriteThatFailsHoldsTheRegistrationBack`、`TestRunWatchAll_WokenAndNotWokenItemsBothReachTheHumanAtTheAckDeadline`、`_SweepsTheDeadlinesThenWakesTheDirectorBeforeThePulse`、`_ADirectorFileThatCannotBeReadSkipsOnlyTheWake`、`TestInboxDeadlines_AWokenItemComesDueAsOneNotWoken` | G1〜G8 と O1〜O4 のうち G7、O4(見張りのサイクルをまたぐメモリ。probe P4、P4b)と O3(同値)が生き残り、G5 は揺れでだけ赤になった(同値と判断) |
| AC8 受信箱 | org の `TestInboxRead_WokenMarksTheItemInAnyStateWithoutChangingIt`、`_LinesWithoutWokenFoldAsBefore`、`TestInboxAppendWoken_WritesOneLinePerIDWithOneTS`、`_RefusesAndWritesNothing`、`_OnAMissingInboxCreatesNothing`、`_TornLastLineGetsTheNewlineFirst`、`_WaitsForTheInboxLock`。cli の `TestOrgInbox_ShowsTheWokenEvent` | I1〜I5 はすべて赤 |
| AC9 文書 | テストが固定するのは `TestRenderDirectorPrompt_TellsWhatTheWakeLineIs`、`TestOrgDirectorHelp_DescribesTheWake`、`TestOrgWatch_HelpDescribesTheDirectorWake`、`TestTemplateRalphToml_WakeCmdExampleIsValid` と、run-test.sh の中の shell の suite(`tests/test-check-skill-sync.sh` など)。文書の中身と drift は /verify の担当 | 対象外 |
| AC10 隔離 | 下の「隔離」の節 | - |

## Edge cases

| 計画の edge case | テスト | 結果 |
| --- | --- | --- |
| 起こしているあいだに件が ack される | `TestWakeDirector_AnItemAckedDuringTheWakeIsStillRecordedAsWoken` | PASS(3 回とも) |
| 同じサイクルで期限の送りと起こすことが重なる | `TestRunWatchAll_SweepsTheDeadlinesThenWakesTheDirectorBeforeThePulse`(順)、`TestRunWatchAll_WokenAndNotWokenItemsBothReachTheHumanAtTheAckDeadline`、`TestInboxDeadlines_AWokenItemComesDueAsOneNotWoken` | PASS。順を入れ替える R1、R2 は赤 |
| 送り先に制御文字を書いた手の行 | `TestWakeDirector_AnUnusableTargetIsSaidOnceAndNothingRuns`(`t\x1b[2J`、`-x`、空)、`TestValidateDirectorWakeTarget` | PASS。herdr の既定の道で手で書き換えた pane id は固定されていない(H6、probe P3) |
| wake_cmd が戻らない(時間切れ) | `TestWakeDirector_AFailedWakeIsNotTriedAgainForFiveMinutes/no_return_in_time` | 3 回の全体の実行では PASS。負荷の下で落ちることがある(Failure analysis) |
| 登録が入れ替わった直後 | `TestWakeDirector_AReplacedRegistrationIsNotHeldBackAndDoesNotWakeAgain` | PASS。W3 は赤 |
| headless の `starting` のあいだの件 | `TestWakeDirector_AHeadlessDirectorIsWokenOnlyWhileRunning` | PASS |

## 隔離(AC10)

- canary: PATH の先頭に `herdr`、`osascript`、`tmux`、`orca`、`gh`、`open` の canary(呼ばれたら 1 行を記録して exit 97)を置き、run-test.sh、go1.25.8、`-race`、繰り返し、coverage、mutation、probe のすべての回を走らせた。記録は 0 行だった
- canary の感度: 同じ形の stub を sh から 6 本とも打つと 6 行が記録された。Go の側では、別の canary の dir を PATH の先頭に置き、`herdr` がそこに解かれることを確かめてから本物の driver(`driver.Herdr{R: driver.ExecRunner{}}`)で `WakeDirector` を呼ぶ probe を overlay で走らせ、1 行が記録され、起こす手順は失敗した。PATH を引き継ぐテストが herdr を打てば canary に届く。CLI のテストの多くは PATH を自分の stub に差し替えるので、そこはこの canary の外になる
- signal: 足したテストは自分では signal を送らない。時間切れの行の stand-in(`exec sleep 30`)は、テストが起こしたプロセスで、`runDirectorCmd` の `exec.CommandContext` が ctx の期限で kill する(コードで確認)
- 台帳: main のチェックアウトの `.harness/state/org` はテストの前も後もない。`git status --porcelain --ignored`(`.harness/cache/` を除く)をテストの前と比べた。この /test が増やしたのは、この報告、insight event の 1 行、evidence の 3 つ(mutation の表、test の log、run-test.sh の `verify-2026-10-18-094238.log`)、計画の 1 行だけだった。ほかに増えていた verify の報告への追記と `verify-2026-10-18-094658.log`(Mode: static)は、同じ時間に走っていた verifier のもの
- `ORCA_TERMINAL_HANDLE` の値、herdr の pane id、送り先の値、token は evidence と報告に写していない
- `$TMPDIR` には、run-test.sh を始めてから `watch-status-*` の dir が 174 個増えた(全部で 1,927 個)。前から知られている `newTestWatchRun`(`internal/org/watch_test.go`)の後始末のない `os.MkdirTemp` によるもので、この差分の外。別のセッションの分も混ざりうるので消していない
- 手順の誤り 1 件: 生き残りの再実行を、最初は Bash の末尾の `&` で起こした(子が SIGINT を無視し、signal を使うテストの前提が崩れる)。自分の pid だけを止め、`run_in_background` で起こし直した。止めた回の結果は使っていない

## Mutation

mutant は 83 個。表と方法は `docs/evidence/test-2026-10-18-org-director-wake-mutation.txt` にある。写しの中で対象の文字列が 1 回だけ現れることを確かめてから置き換え、`go test -overlay` で読ませた。HEAD の写しでの基準の回(org と cli と config)はすべて ok だった。

- 赤 77 個。生き残りは 6 個で、内訳は穴 4 個(G7、O4、H6、M11)、同値 2 個(O3、G5)
- G7(見張りが呼ぶたびに新しい DirectorWakeMemory を渡す)と O4(サイクルごとにメモリを空にする): `WakeDirector` を直接呼ぶテストはメモリを 1 つ使い回すが、`RunWatchAll` が何サイクルもまたいで同じメモリを持つことを確かめるテストがない。壊れると、失敗する wake_cmd が 5 分を待たずに毎サイクル打たれ、失敗の 1 行と「起こせない」の 1 行も毎サイクル出る。probe P4(3 サイクルで wake_cmd は 1 回、失敗の行は 1 本)と P4b(送る手段なしの行は 1 本)がどちらも赤にした
- H6(herdr の pane id の形の確かめを消す): wake_cmd が空で herdr の headless の director のとき、記録された pane id が送り先の形に合わない(手で書き換えた `pane_created`)場合のテストがない。壊れると、`-x` のような pane id がそのまま `herdr pane get` の引数に入る。probe P3(pane `-x` と ESC を含む値。2 回呼んで 1 行だけ、herdr は 0 回)が赤にした
- M11(区切りのない実行ファイルの名前も設定のディレクトリから解く): `./wake.sh` のような相対のパスはテストがあるが、`wakeprobe` のような名前を PATH で引く場合のテストがない。probe P2 が赤にした
- O3(一度だけの鍵から理由を外す): ralph が書く台帳では、同じ登録の id のまま理由が変わることがない(launch、wake、送り先、pane は attach と start で決まり、変わるときは id も変わる)。手で `director.jsonl` を書き換えたときだけ 2 つ目の理由が出なくなる。同値と判断し、穴としては数えない
- G5(送れたあとの recordSuccess を消す): 送るのは holdsBack が false のとき、つまり最後の失敗から 5 分以上たったときだけなので、残った失敗の時刻はまた失敗したときに上書きされるまで効かない。同値と判断した。赤になった 1 回は下の揺れだけによる
- probe は狙った mutant だけを赤にし、ほかの生き残り(O3、G5 を含む)は緑のままだった(evidence の kill matrix)。P1(`ralph status --json` に wake と wake_target が載る)は mutant でなく verify の I3 を閉じる probe で、HEAD で PASS
- パッケージ全体での再実行: B1(G7)は org と cli の全体で緑のままだった。B2(H6 + M11 + O3 + G5)の 1 回目は、org のパッケージ全体を 2 本並べて走らせていたときに `TestRegate_FetchFailures/the_fetch_runs_past_the_bound`(regate_test.go:771、exit 128 の `Could not read from remote repository`)だけで落ちた。この 4 個が触らない regate の fetch の時間切れのテストで、org-auto-merge の /test で記録した揺れと同じ形だった。並べる本数を減らした 2 回目は org と cli の全体で緑のままだった。O4 は G7 と同じ種類の変更(見張りのメモリが呼び出しをまたいで残らない)で、P4 と P4b が両方を赤にするので、全体では流し直していない。パッケージ全体で流しても、生き残りを赤にするテストはなかった

## Failure analysis

| Test | Error | Root cause | Proposed fix |
| --- | --- | --- | --- |
| `TestWakeDirector_AFailedWakeIsNotTriedAgainForFiveMinutes/no_return_in_time`(internal/org/director_wake_test.go:952。HEAD で揺れる。3 回の全体の実行では PASS) | `woken [] after 0 runs, want none after one` | この行は `DriverCallTimeout` を 1 秒にする。stand-in の sh は印のファイルに 1 行を書いてから `exec sleep 30` するが、負荷の下では sh の起動と書き込みが 1 秒に間に合わず、記録の前に kill される。HEAD の `-count=40` で 1 回(load 15〜240)、時間と関係のない mutant A3、C2、G5、G6 の回でも 1 回ずつ落ちた | この行の timeout を 3 秒ほどに上げるか、時間切れの行では `len(runs) <= 1` を受け入れる。テストを直すのはこのタスクの範囲外なので、報告だけにした |
| `TestRegate_FetchFailures/the_fetch_runs_past_the_bound`(internal/org/regate_test.go:771。この差分の外) | 時間切れの文の代わりに exit 128 の文 | org-auto-merge の /test で記録した揺れと同じ形(そのときは org の全体を 3 本、今回は 2 本並べていた)。process の起動と I/O の取り合いで、時間切れの前に fetch が失敗するとみている。原因は未確認です | 前と同じく、落ちたら単独で流し直す |

## Regression checks

| Previously broken behavior | Status | Evidence |
| --- | --- | --- |
| `wake = ["pull"]`(既定)のとき、見張りは wake_cmd も herdr の send-text も打たない | PASS | `TestRunWatchAll_DoesNotWakeAPullDirector`、`TestWakeDirector_RunsNothingWithoutAnItemOrAPushRegistration` |
| 8a の attach、start、stop、escalate、期限のテスト | PASS | 3 回の全体の実行で既存のテストに FAIL なし |
| `woken` のない今までの受信箱の行の読み方 | PASS | `TestInboxRead_LinesWithoutWokenFoldAsBefore` |

## Test gaps

1. 見張りのサイクルをまたぐ DirectorWakeMemory(G7、O4)。probe P4 と P4b の形のテストを `watch_all_test.go` に足せば閉じる。壊れたときに毎サイクル打ち直すので、穴の中では影響が最も大きい(LOW〜MEDIUM)
2. herdr の既定の道で、記録された pane id が送り先の形に合わない場合(H6)。probe P3 の形で閉じる(LOW。pane id は ralph が herdr から受けて書くので、手で書き換えたときだけ起きる)
3. 区切りのない wake_cmd の名前を PATH で引く場合(M11)。probe P2 の形で閉じる(LOW)
4. 時間切れの行の揺れ(Failure analysis の 1 行目)。CI の負荷で落ちれば偽の赤になる
5. `ralph status --json` の wake と wake_target(verify の I3)。probe P1 の形で閉じる(LOW)
6. CLI の binary を通した端から端まで(`ralph org escalate`、`ralph org wait --inbox`、`ralph org watch --all` の順)のテストはない。org の層の `TestRunWatchAll_WakesTheDirectorOnceForAnEscalatedItem` と、CLI の `TestOrgWatch_RunsTheWakeCmdOfTheMainWorktreeInItsRoot` に分かれて固定されている
7. `watch --all --detach` の子が打つ場所を、端から端までは確かめていない。子は main worktree のルートで `--state-dir` 付きで起き、同じ `orgWatchDirectorWake` を通る。`LedgerMainWorktreeRoot` は flag で渡された台帳も、main worktree の既定の台帳と同じパスなら main worktree のものと読む(コードで確認)
8. 本物の herdr、Orca、tmux での起こし方は、テストでは打たない作り。実機は計画の S5(`docs/evidence/director-wake-livefire-2026-10-18.md`)にあり、この /test では走らせていない
9. 通らなかった block(Coverage の節)の多くは、読み込みエラーなどの守りの分岐

## Verdict

- Verdict: pass
- Pass: yes。`./scripts/run-test.sh`(changed、full に広がった)は rc 0(shell 41 本すべて OK、Go 8 パッケージすべて ok)。`GOTOOLCHAIN=go1.25.8 go test ./... -count=1` は rc 0、`-race`(go1.26.0、org・cli・config)も rc 0 で DATA RACE はなかった。足した 67 本のテストは 3 回のすべてで PASS。時間切れの行の揺れは HEAD の繰り返しで 40 回に 1 回出たが、3 回の全体の実行では出ていないので、判定を fail にはしない(Failure analysis と Test gaps の 4)
- Fail: なし
- Blocked: なし
- Verified: AC1〜AC8 をテストの名前に当て、mutant 83 個のうち 77 個が赤。生き残り 6 個は同値 2 個と LOW〜MEDIUM の穴 4 個で、穴はどれも probe で閉じることを確かめた。AC10 の隔離: canary は 0 回(感度は確認済み)、台帳は作られず、worktree にはこの報告と insight event と evidence と計画の 1 行だけが増えた
- Unverified: 本物の herdr、Orca、tmux での起こし方(テストの範囲外)、CI の負荷の下での時間切れの行の揺れの頻度
