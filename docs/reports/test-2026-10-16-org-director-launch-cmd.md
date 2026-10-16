# Test report: org-director-launch-cmd

- Date: 2026-10-16(JST 12:21〜13:50)
- Plan: docs/plans/active/2026-10-16-org-director-launch-cmd.md(`Status: Approved`。`./scripts/plan-visual.sh digest` は 2aeecc48d0ba で、`- Approved:` の digest と同じ。仕様 FR-9 の起動 (B) のうち `[org.director].launch_cmd`、8 段目の 8a-2b)
- Tester: tester subagent (Claude Opus 5.5)。パイプラインの 1 回目
- Scope: HEAD e2667a6b(base 3723d900、18 コミット、48 ファイル、+8,331/-515)。/verify が見た 8ad4275d から増えたのは、verify の報告(9c609ad1)、stop の help の文とそれを固定するテストの文字列、recipe と skill の文、`org_legacy_ledger.go` のコメント(0f5fbf89)、`director_stop.go` のコメントと計画(e2667a6b)の 3 つで、挙動は変えていない。behavioral test だけを実行した(静的解析は /verify の担当)。環境は macOS(arm64、Darwin 25.5.0)、go1.26.0 と go1.25.8(`GOTOOLCHAIN`)、git 2.49.0。実行中の load average は 3〜13(別のセッションの分を含む)
- Evidence: `docs/evidence/test-2026-10-16-org-director-launch-cmd.log`(run-test.sh の全出力、2 つの版の Go のパッケージ行、テストのみの AC のテストの時間、`-race`、繰り返し、canary、mutation の結果、Linux の結果、後片付けの確認)。wrapper の全ログは `docs/evidence/verify-2026-10-16-032149.log`。どちらも gitignore の対象なのでコミットしない。`<scratch>` はセッションの scratchpad、`~` は `$HOME`
- テストもコードも変えていない。mutant と probe のテストは scratchpad に置き、`go test -overlay` で読ませた。worktree に残したのは、この報告、insight event の 1 行、上の evidence の log だけ(mutant O23 が書いたファイルは消した。AC10 の節)

## Test execution

| Suite / Command | Tests | Passed | Failed | Skipped | Duration |
| --- | --- | --- | --- | --- | --- |
| `./scripts/run-test.sh`(changed。`templates/base/ralph.toml` が分類されず full に広がった。go1.26.0) | shell 41 本、Go 8 パッケージ | shell 41 本すべて OK、Go 8 パッケージすべて ok | 0 | shell 1(下) | 739 秒、rc 0 |
| `go test ./... -count=1 -v -coverprofile`(go1.26.0) | top-level 1,877、subtest 2,746 | 1,874、2,746 | 0 | 3(下) | 226 秒(cli 224.9 s、org 108.7 s)、rc 0 |
| 同じものを `GOTOOLCHAIN=go1.25.8` で | 同じ | 同じ | 0 | 3 | 236 秒(cli 229.4 s、org 110.1 s)、rc 0 |
| うちこの差分で足したテスト | top-level 73、その下の subtest 143 | 両方の版ですべて | 0 | 0 | - |
| `go test -race -count=1 ./internal/org/... ./internal/cli/...`(go1.26.0) | 4 パッケージ | 4 | 0 | - | 272 秒(org 125.1 s、cli 255.0 s)、DATA RACE なし |
| 繰り返し: org の launch_cmd のテスト 42 本を `-count=5` | 210 | 210 | 0 | 0 | 115 秒 |
| 繰り返し: cli の launch_cmd のテスト 25 本を `-count=3` | 75 | 75 | 0 | 0 | 39 秒 |
| Linux: ubuntu:24.04 の container(procps-ng 4.0.4、kernel 6.8.0、git 2.43.0、uid 1000)で、go1.25.8 で cross build した org の 44 本を 3 回と cli の 25 本を 1 回 | 157 | 157(subtest 109 × 3 と 32) | 0 | 0 | - |
| Linux: 同じ container で lstart を読み直す probe(20 プロセス、120 秒) | 15,560 回 | 最初の値と違う読み 0 回 | - | - | 120 秒 |
| probe のテスト 2 本(overlay だけ、コミットしない)を `-count=3` | 6 | 6 | 0 | 0 | 5.9 秒 |
| mutation(`go test -overlay`。org 58、CLI 17、config 5、prompts 1) | mutant 81 | 赤 78 | 生き残り 3 | - | 約 50 分(org と CLI を並べて) |
| 生き残り 3 個を `-run` なしのパッケージ全体(org と cli)で | 6 回 | 赤 0 | 生き残り 6 | - | org 約 113 秒、cli 約 240 秒 |
| 上の probe のテストを overlay で足した mutant(O33p、O35p) | 2 | 赤 2 | 0 | - | 7 秒ずつ |

- shell は 41 本すべてが `OK` で終わり、`FAIL:` の行はどれも 0、PASS で始まる行は 4,283 本だった。Skipped は前からある `tests/test-secret-scan-branch.sh` の 1 件(git が 2.49.0 なので、2.41 より古い git の場合を飛ばす)。この差分は shell のテストを変えていない
- `run-test.sh` の中の Go は cli(230.2 s)、config、insights、org(116.2 s)を流し直し、org/driver、org/protocol、scaffold、upgrade は `(cached)` だった。この 4 つも上の 2 回の `-count=1` で ok を確かめた
- go1.25.8 と go1.26.0 で、テストの名前と結果の行(時間を除いて並べた 4,623 行)は同じだった
- Skipped の 3 件は `TestEscalate_NoDesktopNotificationOffDarwinIsSkipped`(org。darwin で飛ばす作り)と scaffold の `TestBaseFS_WithMockFS`、`TestAvailablePacks_WithMockFS`で、どれもこの差分と関係ない
- 差分で足した `func Test` は 73 本で、消したものはない。うち 2 本(`TestDirectorCheckInHelperProcess`、`TestOrgDirectorCheckInHelperProcess`)は check-in の helper process の入口で、環境変数がなければ何もしない
- 前に揺れたテスト(`TestRunDoctorOpts_ProbeModelsFalse_NoSubprocess`、`TestRunDoctorFull_StrictFlipsExitCode_DriftedCore`、`TestRunWatcher_TimeoutIndependentOfSmallInterval`、`TestOrgGate_InterruptStopsTheScriptAndRecordsNothing`)は、どの実行でも落ちなかった
- 繰り返しの対象は、org が `TestReadDirectorProcess_*`、`TestStopDirectorProcess_*`、`TestDirectorLaunchScript_*`、`TestDirectorLaunchArgv_*`、`TestRunDirectorLaunchCmd_*`、launch_cmd の `TestStartDirector_*` と `TestStopDirector_*`、`TestDirectorMarkCmdLaunched_*`、`TestDirectorMarkStoppedIfUnchanged_RefusesACheckInAfterTheRead`、`TestDirectorAttachHeadless_RecordsALaunchCmdAndItsCheckIn`、`TestDirectorLaunchCmdStoppedError_Text`。cli が `TestOrgDirectorLaunched_*`、launch_cmd の `TestOrgDirectorStart_*` と `TestOrgDirectorStop_*`(8a-2 の相対の agmsg home のテスト 3 本も名前で拾われた)、attach と detach と status の launch_cmd のテスト、`TestStatusCmd_DirectorLineShowsLaunchCmdDirectors`、`TestOrgStopAll_NamesTheOpenHeadlessDirectors`。正確な正規表現は evidence にある

## verify が「テストのみ」とした部分

verify の報告(Coverage gaps)が /test に回した部分を、固定するテストの名前に当て、その実行を確かめた。下のテストはどれも go1.26.0、go1.25.8、`-race`、繰り返し(org は 5 回、cli は 3 回)のすべてで PASS だった。Linux の列は ubuntu:24.04 の container での結果。

| AC の部分 | テスト | Linux |
| --- | --- | --- |
| AC3 親の pid と違う `--pid`、ps を読めない pid、gone と zombie の pid の check-in を断る | `TestOrgDirectorLaunched_Refusals`(subtest 10。`a --pid that is not the parent`、`a ps that cannot be read`、`a pid that is gone`、`a zombie` を含む)、store の側の `TestDirectorMarkCmdLaunched_RefusesEveryOtherCheckIn`(subtest 16) | PASS |
| AC4 `detached` だけを書く後始末(SIGTERM を無視する agent、ps を読めない) | `TestStartDirector_ACleanupThatCannotConfirmTheEndLeavesItDetachedOnly`(subtest 2) | PASS |
| AC4 check-in を待つあいだの `attach --replace` | `TestStartDirector_AReplaceWhileWaitingForTheCheckInEndsItsOwnLaunch` | PASS |
| AC4 check-in の直後で止めた起動スクリプトを後始末が止め、agent は起動しない | `TestStartDirector_TheCleanupStopsALaunchScriptPausedAfterItsCheckIn` | PASS |
| AC4 後始末と重なる check-in(読み直して止める) | `TestStartDirector_ACheckInDuringTheCleanupIsReadAgainAndStopped` | PASS |
| AC4 CLI の後始末のエラーが stop と端末を言う | `TestOrgDirectorStart_ALaunchCmdCleanupErrorNamesTheStopAndTheTerminal` | PASS |
| AC5 8a-2 の断り(enabled = false、codex の autonomous と edits、promoted と starting の登録、見張りの関門、読めない `director.jsonl`) | `TestStartDirector_LaunchCmdKeepsTheOtherRefusals`(subtest 7)、agent が PATH にないときは `TestStartDirector_LaunchCmdRefusesWhatCannotRunTheAgent` | PASS |
| AC5 `--config` の相対の agmsg_home | `TestOrgDirectorStart_ARelativeAgmsgHomeIsRefusedWithALaunchCmd` | PASS |
| AC6 gone、zombie、同じ pid で起動時刻の違うプロセスには何も送らない | `TestStopDirector_LaunchCmdSendsNothingToAProcessThatHasEnded`(subtest 3)、unit の `TestStopDirectorProcess_AnEndedProcessIsSentNothing`(subtest 4) | PASS |
| AC6 pid のない `starting` の登録には何も送らず、そのあとの check-in は断られる | `TestStopDirector_ALaunchCmdStillStartingIsRecordedStoppedWithNothingSent` | PASS |
| AC6 exec の前の起動スクリプトにも stop が効く | `TestStopDirector_LaunchCmdStopsALaunchScriptPausedAfterItsCheckIn` | PASS |
| AC7 ps を読めない stop は何も送らない(`--force` でも) | `TestStopDirector_ALaunchCmdWhosePSCannotBeReadIsSentNothing`、unit の `TestStopDirectorProcess_APSThatCannotBeReadSendsNothing` | PASS |
| AC7 stop の読みのあとの check-in | `TestStopDirector_ALaunchCmdCheckInAfterTheReadRecordsNothing`、`TestDirectorMarkStoppedIfUnchanged_RefusesACheckInAfterTheRead` | PASS |
| AC1 と AC8 の herdr の道が 8a-2 のまま | 8a-2 からある herdr の director のテスト 59 本(`TestStartDirector_LaunchesInAWorkspaceOfItsOwn`、`TestStopDirector_ClosesThePaneAndWorkspaceAndRecordsTheStop`、`TestOrgDirectorStart_LaunchesRegistersAndStops` など)。既存のテストで期待値が変わったのは、文の追加(設定の読み込みのエラー、`--config` の help、start の help の agmsg home の文に launch_cmd のことを足したもの)、`OrgDirectorConfig` が slice を持つので `!=` を `reflect.DeepEqual` にした比べ方、herdr の start のテストの `!reg.Headless()` を `!reg.HerdrLaunch()` に絞ったもので、どれも検査を弱めていない | 対象外(macOS だけ) |
| self-review M1 の相対の `RALPH_ORG_AGMSG_HOME` | `TestOrgDirectorStart_LaunchCmdMakesARelativeAgmsgHomeAbsolute` | PASS |
| Linux の ps の前提(verify が CI で見るよう勧めた 2 本) | `TestReadDirectorProcess_AnAliveProcessKeepsItsStartTime`、`TestReadDirectorProcess_RunsPSInCAndUTC` | 3 回とも PASS |

「赤になること」は下の mutation で確かめた。

## AC10: テストの隔離

- canary: PATH の先頭に `herdr`、`osascript`、`tmux`、`wezterm`、`orca` の canary(呼ばれたら argv を記録して exit 97)を置いて、`-race` の回、繰り返しの回、probe の回、mutation のすべての回を走らせた。繰り返しの回には `claude` と `codex` の canary も足した。記録は 0 行だった。同じ PATH から手で `herdr --version` を打つと記録され exit 97 になったので、PATH の検索は canary に届く。CLI のテストは `setupOrgStubPATH` で PATH を自分の stub に差し替えるので、この canary が見張るのは PATH を引き継ぐテストだけになる。claude と codex を起動スクリプトから動かさない点は、CLI の `TestMain` が `orgDirectorAgentLookPath` と `orgDirectorRalphExecutable` を失敗に固定していることで守られている(コードで確認)
- signal: テストが signal を送るのは、`startTestProcess` と `startDirectorTestProcess` で起こした自分の子と、`killCheckedInProcesses` と `killDirectorLaunchedProcesses` が ps の起動時刻を check-in の記録と比べたプロセスだけだった(コードで確認)。pid 4242 や 99 を記録する行は、stop を通らないか、ps を必ず失敗させる reader(`refusingProcessReader`)と組になっている
- 残ったプロセス: 各回のあとに `ps -axo pid,ppid,lstart,command` を `sleep [0-9]`、`director_`、`CheckInHelper`、`org.test`、`cli.test`、`/launch/`、`nohup` で絞って見た。テストが起こしたものは 0 だった。見えたのは別のプロセスの `sleep 2448`(`/usr/local/bin/malware_monitor_update` の子、テストの前から)と、`/usr/local/bin/malware_monitor` の子の `sleep 0.5` の繰り返しだけ。mutation のあと(13:42)も同じだった。Linux の container の中でも、3 回の org と 1 回の cli のあとに 0 だった
- 台帳と worktree: テストのあと、main のチェックアウトにも worktree にも `.harness/state/org` はない。`git status --porcelain` は、この報告と insight event と evidence を書く前は空だった。gitignore の対象の一覧も、テストの前と同じだった(`.harness/cache/` を除く)。mutation のあとには O23 の mutant が書いた 5 つのファイルが `internal/org/` にあった(下の Mutation の節)。消したあとは、`git status --porcelain` がこの報告だけ、gitignore の対象の一覧もテストの前と同じに戻った
- 手順の誤り 1 件: go1.26.0 の全体の 1 回目を、`run_in_background` の中の末尾の `&` で起こしてしまった。job が親を失って動き続けたので、15 秒ほどで自分の `go test` を止めた。中の `cli.test` と `org.test` は SIGPIPE で終わり、`t.Cleanup` は走っていない。直後に上の絞り方で見て、残ったプロセスは 0 だった。この回の結果は使わず、上の表の go1.26.0 の行は起こし直した回のもの

## Linux

- ubuntu:24.04(GitHub の ubuntu-latest と同じ 24.04、procps-ng 4.0.4)の container を Docker Desktop の Linux VM(kernel 6.8.0、arm64)で動かした。テストの binary は macOS で `GOOS=linux GOARCH=arm64 GOTOOLCHAIN=go1.25.8 go test -c` で作り、git だけを apt で入れ、uid 1000 で走らせた
- org の launch_cmd のテスト 44 本(繰り返しの 42 本と、fold の `TestDirectorRead_FoldsLaunchCmdRegistrations`、`TestDirectorRegistration_HeadlessHerdrAndCmdLaunches`)を 3 回、cli の 25 本を 1 回流し、すべて PASS だった。本物の ps と SIGTERM を使う start、check-in、stop(`TestStartDirector_LaunchCmdOpensTheTerminalAndChecksIn`、`TestStopDirector_LaunchCmdEndsTheCheckedInProcessAndRecordsTheStop`)、zombie と gone の読み(`TestReadDirectorProcess_GoneAndZombie`)、`LC_ALL=C TZ=UTC` の lstart の形(`TestReadDirectorProcess_AnAliveProcessKeepsItsStartTime`)が procps でも通る
- 読み直しの probe: 0.137 秒ずつずらして起こした 20 個の sleep の lstart を、120 秒のあいだ約 0.1 秒ごとに `LC_ALL=C TZ=UTC ps -o lstart= -p <pid>` で読み、15,560 回の読みで最初の値と違うものは 0 回だった。20 個は 2 秒あまりに広がっているので、秒の境目をまたいだプロセスを含む
- 確かめていないこと: 時計の step(NTP の補正など)のあとの読み。self-review が挙げた心配で、procps は lstart を `/proc/stat` の btime と起動からの tick から組み立てる。おそらく wall clock が step すると btime が動き、同じプロセスの lstart が 1 秒以上ずれる。その場合、stop は生きた director を「別のプロセスが pid を使っている」と読み、何も送らずに `stopped` を書く。未確認です。Docker の VM の時計を動かすのは他の container にも響くので試していない

## Mutation

mutant は 81 個。scratchpad の写しの中で対象の文字列が 1 回だけ現れることを確かめてから置き換え、`go test -overlay` で読ませた。PATH の先頭には AC10 の canary を置いた。org の mutant は `-run Director`、CLI は `-run 'Director|Status|StopAll'`、config はパッケージ全体、prompts は `-run 'Director|Prompt'` で流した。S8(check-in の start time の trim を外す)は `strings` が使われなくなって build できなかったので、`strings.TrimSpace("") + c.StartTime` に直した S8b で数えた。

| 対象 | mutant | 壊したもの | 赤にしたテスト(代表) |
| --- | --- | --- | --- |
| ps の読み(`director_cmd.go`) | O1〜O4、O34 | `LC_ALL=C TZ=UTC` を外す、zombie の判定、「exit 1 で出力なし」を gone と読む条件、起動時刻の比較、lstart の形の検査 | `TestReadDirectorProcess_RunsPSInCAndUTC`、`_GoneAndZombie`、`_Errors`、`TestStopDirectorProcess_AnEndedProcessIsSentNothing` |
| SIGTERM と待ち | O5〜O9 | ps を読めないのを ended に、SIGKILL を送る、待たずに ended、SIGTERM のあとの読み失敗と時間切れを ended に | `TestStopDirectorProcess_*` の 4 本、`TestStartDirector_ACleanupThatCannotConfirmTheEndLeavesItDetachedOnly`、`TestStopDirector_ALaunchCmdProcessThatOutlivesItsSIGTERM` |
| 起動スクリプト | O10〜O20 | 相対の agmsg home を通す、check-in の `\|\| exit 1`、cd の quote と `\|\| exit 1`、agmsg home の export を消す・空でも書く、token の export、ファイルと `launch/` の 0700、`{script_sh}` の quote、`{name}` | `TestDirectorLaunchScript_*`(`a cwd that is gone`、`the check-in is refused` の subtest を含む)、`TestDirectorLaunchArgv_ReplacesThePlaceholders` |
| launch_cmd の実行 | O21〜O24 | `ErrWaitDelay` の扱い、shell を通す、`cmd.Dir`、`WaitDelay` | `TestRunDirectorLaunchCmd_*` |
| check-in を待つ、後始末 | O26〜O32 | 待つ期限、2 回目の読み、`detachCmdOnly` へ進む条件、Detach を消す、`MarkStoppedIfUnchanged` を `MarkStopped` に、agent の lookPath | `TestStartDirector_*`(`ACheckInDuringTheCleanupIsReadAgainAndStopped`、`ACleanupThatCannotConfirmTheEndLeavesItDetachedOnly` など) |
| `director_start.go` | S1〜S12 | herdr の中の断りを launch_cmd にも/herdr でも外す、check-in の断り 6 つ、check-in の pid と start time の比べ直し、事前の検査、launch の値の検査 | `TestDirectorMarkCmdLaunched_RefusesEveryOtherCheckIn`、`TestStartDirector_RefusesBeforeWritingOrCallingHerdr`、`TestStartDirector_LaunchCmdKeepsTheOtherRefusals` |
| fold(`director.go`) | D1〜D4 | launched の pid の有無の検査 2 つ、`Headless()` が launch cmd を含む、start time を捨てる | `TestDirectorRead_FoldsLaunchCmdRegistrations`、`TestStartDirector_LaunchCmdOpensTheTerminalAndChecksIn` |
| `director_stop.go` | T1〜T7 | launch cmd の分岐、自分の token の断り、`--force` なしの断り、`MarkStoppedIfUnchanged`、`MarkForceStopped`、`Forced`、check-in の重なりのエラー | `TestStopDirector_*` |
| config | F1〜F5 | 先頭の要素の検査、未知の placeholder、`{script}` か `{script_sh}` の必須、`{script_sh}` の扱い、検査の呼び出し | `TestLoad_OrgDirectorLaunchCmd`、`TestTemplateRalphToml_LaunchCmdExampleIsValid` |
| CLI | C1〜C17 | check-in の親の pid、生きているか、ps の失敗、hidden、相対の agmsg home(C5〜C7)、status と stop の pid と名前、端末の名前、main worktree の launch_cmd、エラーの文 | `TestOrgDirectorLaunched_Refusals`、`TestOrgDirectorStart_LaunchCmdMakesARelativeAgmsgHomeAbsolute`、`TestOrgDirectorStop_LaunchCmdErrorsAndAlreadyStopped` など |
| prompts | P1 | launch ごとの行を全部残す | `TestRenderDirectorPrompt_LaunchCmdTalksInTheTerminal`、`_NoLaunchIsTheHerdrPrompt` |

生き残った 3 個は、`-run` なしの org と cli のパッケージ全体でも生き残った。

| mutant | 変えたところ | 分類 |
| --- | --- | --- |
| O25 | `waitForCheckIn` の「今の登録が `stopped`」の条件を外す(`director_cmd.go:585`) | 同値。ralph が書く行では起きない。今の登録を止める書き込み(`markStopped`)は `detached` と `stopped` を 1 回で足すので、`stopped` の登録が今のもののまま残ることはない。手で足した行でだけ届く守りで、8a-2 の herdr の待ち(`director_start.go:272`)も同じ条件を持つ |
| O33 | `abortCmd` の最初の読みで `ErrDirectorStopped` を見る枝を外す(632 行) | テストの穴(LOW)。check-in を待つあいだに人が `ralph org director stop --id d1` を打つと、mutant の start は d1 がもう止まっているのを「記録に失敗した」と読み、`DirectorCleanupError`(`...; nothing more was recorded; record the stop with ralph org director stop --id d1`)を返す。もう要らない stop を勧めることになる。probe A(`TestProbe_AStopDuringTheCheckInWaitCountsAsTheCleanupsRecord`。HEAD で 3 回 PASS)を足すと赤になる(O33p) |
| O35 | `detachCmdOnly` で、Detach が `ErrDirectorIDMismatch` か `ErrDirectorNotAttached` のときに `Detached: true` を返す(664 行) | テストの穴(LOW、文だけ)。mutant のエラーは「nothing more was recorded」の代わりに「d1 was recorded detached only, so it is no longer the current registration but stays open」と言う。台帳の状態の説明としては合っている(`attach --replace` が d1 の `detached` を書いた)が、後始末が書いたように読める。probe B(`TestProbe_AnUnconfirmedCleanupAfterAReplaceLeavesItOpen`。HEAD で 3 回 PASS)を足すと赤になる(O35p) |

O11(check-in の `|| exit 1` を消す)では、check-in が断られたあとも起動スクリプトが agent の代わりの `sleep 600` を exec し、テストが待ち続けた。親が自分の mutant の `org.test` である `sleep 600` だけを 3 回 SIGTERM で終わらせた(ほかのプロセスには送っていない)。O11 はそれでも赤だった。O23(`cmd.Dir` を消す)は偽の launch_cmd を package の directory で動かし、`internal/org/` に `args`、`pwd`、`stdin`、`child.pid`、`out` を書いた。中身と時刻(13:10:51、O23 の実行中)を見てから 5 つとも消した。HEAD のテストの回のあとは `git status` が空だったので、HEAD のテストはそこに書かない

## Coverage

- Statement(go1.26.0、各パッケージ自身のテストの profile): `internal/org` 94.7%、`internal/cli` 88.0%、`internal/config` 96.2%、`internal/org/driver` 93.1%、`internal/org/protocol` 100.0%、`internal/insights` 86.1%、`internal/scaffold` 75.7%、`internal/upgrade` 91.2%。go1.25.8 でも同じ値だった(8a-2 の 4 回目は org 94.7%、cli 87.8%、config 93.3%)
- 新しいファイル `internal/org/director_cmd.go` は 216/238 文(90.8%)。変えたファイルは `director_start.go` 94.5%、`director_stop.go` 98.4%、`director.go` 99.5%、`prompts.go` 98.4%、`internal/cli/org_director_headless.go` 97.7%、`org_director.go` 98.5%、`status.go` 96.2%、`org.go` 95.8%、`internal/config/config.go` 96.2%
- `director_cmd.go` で通らない 22 文は次のとおり
  - SIGTERM の送信が失敗する枝(ESRCH とそれ以外、211〜216 行)
  - 起動スクリプトの書き込みの失敗(mkdir、chmod、CreateTemp、rename。341〜365 行)
  - 検証のあとの置き換えの失敗(391 行)、agent と ralph のパスの `absPath` の失敗(487、498 行)
  - check-in を読む間隔の既定値(513 行。テストはどれも 10 ms を設定する)
  - `startCmd` の prompt、スクリプト、argv の失敗(537〜550 行)、待つあいだの `director.jsonl` の読みの失敗(581 行)
  - `abortCmd` の、stop が先に `stopped` を書いていた枝(632 行)と読みの失敗(634 行)、`MarkStoppedIfUnchanged` が別の理由で失敗する枝(645〜649 行)
  - `detachCmdOnly` の、登録がもう今のものでない枝(664 行)と Detach がほかの理由で失敗する枝(667 行)
  - 632 行と 664 行は、手で打つ stop や attach --replace と重なれば起きる道なので、下の probe で確かめた

## Failure analysis

| Test | Error | Root cause | Proposed fix |
| --- | --- | --- | --- |
| なし | - | - | - |

## Regression checks

| Previously broken behavior | Status | Evidence |
| --- | --- | --- |
| launch_cmd が空なら herdr の道が 8a-2 のまま(AC1) | 通る | 8a-2 からある herdr の director のテスト 59 本が両方の版と `-race` で PASS。mutant S2(herdr の道の `HERDR_PANE_ID` の断りを外す)は赤 |
| promoted session の登録の扱い | 通る | `TestStartDirector_LaunchCmdKeepsTheOtherRefusals/a_promoted_session_is_attached`、8a-1 の attach と detach のテストが PASS |
| self-review M1(相対の `RALPH_ORG_AGMSG_HOME`) | 直っている | `TestOrgDirectorStart_LaunchCmdMakesARelativeAgmsgHomeAbsolute` が PASS。mutant C5(絶対にしない)と C6(main worktree のルートを基準にする)はどちらも赤 |
| 前に揺れたテスト 4 本 | 落ちなかった | 上の Test execution の注 |

## Test gaps

1. Linux で時計が step したあとの lstart(上の Linux の節)。CI の ubuntu で NTP の補正が入ると、stop が生きた director を別のプロセスと読み、何も送らずに `stopped` を書くおそれがある。未確認です。container の中で 120 秒読み直したかぎりでは、ずれは 0 回だった
2. 本物の claude と codex を HEAD では動かしていない。live run(`docs/evidence/director-launch-cmd-livefire-2026-10-16.md`)は 88bb29fd の binary で、macOS と tmux だけ。そのあとのコードの変更は 98131af5(相対の agmsg home を絶対にする、self-review の LOW)で、テスト(`TestOrgDirectorStart_LaunchCmdMakesARelativeAgmsgHomeAbsolute`、mutant C5〜C7)だけが見ている
3. recipe(`docs/recipes/director-launch-cmd.md`)の Orca と WezTerm の launch_cmd の例は、本物の端末で試していない(テストは偽の launch_cmd を使い、端末のアプリを呼ばない)。recipe 自身も、この 2 つは `--help` から書いて動かしていないと記している
4. O33 と O35 の穴(上の Mutation)。どちらも stop や `attach --replace` が start の後始末と重なったときの戻り値と文で、記録の中身は HEAD でも mutant でも同じ。probe A と B をテストに足せば閉じる。PR を止めるものではない
5. `director_cmd.go` の通らない 22 文(上の Coverage)。起動スクリプトの書き込みの失敗、SIGTERM の送信の失敗、`absPath` の失敗など、OS の失敗を起こさないと届かない枝で、mutation の対象にもしていない

## Verdict

- Verdict: pass
- Pass: yes。`./scripts/run-test.sh`(changed、full に広がった)は rc 0(shell 41 本すべて OK、Go 8 パッケージすべて ok)。`go test ./... -count=1` は go1.26.0 と go1.25.8 の両方で rc 0 で、名前と結果の行は同じ。足した 73 本のテストはどれも PASS。`-race` も org と cli のパッケージ全体で通り、DATA RACE はなかった。繰り返し(org 210 回、cli 75 回)と Linux の container の 157 回でも落ちなかった
- Fail: なし
- Blocked: なし
- Verified: verify が「テストのみ」とした AC3〜AC8 の部分(上の対応表)と self-review M1。mutant 81 個のうち 78 個が赤で、生き残った 3 個は同値 1 個と LOW の穴 2 個(どちらも probe で閉じることを確かめた)。AC10 の隔離: canary は 0 回、残ったプロセスは 0、台帳は作られず、worktree には報告と insight event と evidence だけが残った(O23 の mutant が書いた 5 つのファイルは消した)
- Not verified: Test gaps の 1〜5。どれも PR を止めるものではない

## 追記(orchestrator): Test gaps の 4 を閉じた

O33 と O35 の穴は、この報告のあとで 6b9b27d7 がテスト 2 本を足して閉じた(テストだけで、本番のコードは変えていない)。

- `TestStartDirector_AStopWhileWaitingForTheCheckInIsTheCleanupsRecord`: O33 の overlay の mutant で赤、HEAD で 5 回とも緑
- `TestStartDirector_ACleanupThatCannotConfirmTheEndWritesNothingOnceItIsNotCurrent`: `attach --replace` と `detach --force` の 2 ケースの表。O35 の mutant では 2 ケースとも赤になる。片方の枝だけを壊した mutant では、その枝のケースだけが赤になる。probe のポーリングの速さに頼らないよう、launch_cmd は replace か detach を記録したあとで終わる

6b9b27d7 のあとも、`go test ./internal/org/... -count=1` は go1.26 と go1.25.8 の両方で ok だった。`-race -count=2` も 2 本で ok、テストが起こしたプロセスも残っていない。この報告の実行(e2667a6b)からの変更は org のテストのファイル 1 つだけなので、cli と残りのパッケージの結果はこの報告のままでよい。
