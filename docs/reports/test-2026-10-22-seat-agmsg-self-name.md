# Test report: seat-agmsg-self-name

- Date: 2026-10-22(JST 09:14〜09:58、UTC 00:14〜00:58)
- Plan: `docs/plans/active/2026-10-22-seat-agmsg-self-name.md`(試行の所見を直す系列の 4a。AC1〜AC5 の挙動、境界の場合、回帰)
- Tester: tester subagent(Claude Opus 5.5)。パイプラインの 1 回目
- Scope: HEAD 73fd8c78(base 1e47c309 = origin/main)。behavioral test だけを実行した(静的解析は /verify の担当)。環境は macOS(Darwin 25.5.0、arm64、12 CPU)、go1.26.0 と go1.25.8(`GOTOOLCHAIN=go1.25.8 GOPROXY=off`。toolchain はキャッシュ済みで、network は使っていない)
- Evidence: `docs/evidence/test-2026-10-22-seat-agmsg-self-name.log`(各実行の出力、時刻と load average、mutant の定義と HEAD との差分と結果、AC4 の probe、回帰と canary の出力。ホームは `~`、session の scratchpad は `<scratch>` と書いた)と、run-test.sh が書いた `docs/evidence/verify-2026-10-22-001451.log`。どちらも `docs/evidence/*.log` の gitignore の対象で、コミットしない
- 追跡しているファイルは、この報告と insight event のほかは変えていない。mutant と probe は `go test -overlay` で scratchpad のファイルに差し替えて打ち、worktree のファイルには書いていない。worktree の `git status --porcelain --ignored` の開始時と終了時の差は、`.harness/cache/go-build/` の build cache と run-test.sh の evidence の log だけだった。`go env -w` は打っておらず、利用者の go env のファイル(`~/Library/Application Support/go/env`)は存在しないまま。本物の herdr・agmsg・osascript・Orca・tmux・gh や network を打つものは実行していない。新しいテストが本物の herdr・osascript・tmux・gh に届かないことは 12 の canary で確かめた

## Test execution

| # | Suite / Command | Tests | Passed | Failed | Skipped | Duration |
| --- | --- | --- | --- | --- | --- | --- |
| 1 | `GOFLAGS=-timeout=30m ./scripts/run-test.sh` | shell 43 本、Go 8 パッケージ | shell 43 本すべて OK、Go 8 パッケージすべて ok | 0 | shell 1 件(下) | 876 秒、rc 0。cli 318.6 秒、org 391.3 秒、driver 1.8 秒、config 2.0 秒、insights 1.2 秒。protocol・scaffold・upgrade は `(cached)` |
| 2 | `go test -count=1 -timeout 30m ./internal/org/... ./internal/cli/...`(go1.26.0) | 4 パッケージ | 4 | 0 | 0 | 326 秒、rc 0。org 325.2 秒、cli 246.6 秒、driver 1.4 秒、protocol 1.1 秒 |
| 3 | 2 と同じ(`GOTOOLCHAIN=go1.25.8 GOPROXY=off`) | 4 パッケージ | 4 | 0 | 0 | 305 秒、rc 0。org 300.6 秒、cli 230.5 秒、driver 1.8 秒、protocol 1.9 秒 |
| 4 | 2 に `-coverprofile` を足して(go1.26.0) | 4 パッケージ | 4 | 0 | 0 | 333 秒、rc 0 |
| 5 | 新しいテストと、期待を変えたテストを `-count=1 -v -run '^(…)$'` で(driver 14 本、org 3 本、cli 4 本) | 21(subtest 36) | 21(36) | 0 | 0(`--- SKIP` は 0 行) | 1〜6 秒、rc 0 |
| 6 | `TestHerdr_TabCreate_PaneEnvConcurrentCallsCheckOnce` を `-count=3 -v`、続けて `-count=200` | 3、200 | 3、200 | 0 | 0 | 1 秒、5 秒。1 回 0.02 秒 |
| 7 | `-race`: PaneEnv と RunEnv の 11 本を `-count=3 -v`、driver 全体を `-count=1`、確かめの 3 本を `-count=50` | 33、driver 全体、150 | すべて | 0 | 0 | 5 秒、2 秒、5 秒。`WARNING: DATA RACE` は 0 行 |
| 8 | 確かめの 3 本と RunEnv の 2 本を `-race -count=50`、CPU を回す perl 24 本の下で | 5 本 × 50 回 | すべて | 0 | 0 | 12 秒、rc 0。load average は開始 13.2、終了 17.4 |
| 9 | mutant 40 本(下の表) | 40 | - | 35 本が赤 | - | 生き残り 5 本(1 本は等価、1 本は 10 の probe で赤) |
| 10 | AC4 を CLI で確かめる使い捨ての probe(overlay で足した test file。コミットしない) | 4 subtest | 4 | 0 | 0 | 5 秒、rc 0 |
| 11 | base 1e47c309 の driver の test file 3 つ(`agmsg_test.go`・`driver_test.go`・`herdr_test.go`)を HEAD のコードで | 51 | 51 | 0 | 0 | rc 0。HEAD の test file では 65 本(51 本 + 新しい 14 本) |
| 12 | 新しいテストを canary の PATH で(`herdr`・`osascript`・`tmux`・`gh` を、呼ばれたら記録して exit 97 する script にした dir を PATH の先頭に置く) | cli 4 本、driver 全体、org の `TestDirectorLaunchScript_*` と `TestStartDirector_LaunchCmd*` | すべて ok | 0 | 0 | canary の呼び出しは 0 回 |

- 1 の scope は changed で、`.harness/state/verify-scope` は `reason=changed_languages`、`languages=golang`。shell の 43 本は local verifier が test mode で走らせる全 suite で、どの suite も最後の行が `OK` だった。Skipped は前からある `tests/test-secret-scan-branch.sh` の 1 件で、git 2.49.0 なので「2.41 より古い本物の git」の事例を飛ばしている
- 1 は計画どおり `-count=1` を付けていないが、Go のテストは 4991744e で変わった driver・org・cli とそれに依存する config・insights が実際に走った。`(cached)` の protocol・scaffold・upgrade は、`go list -test -deps` で driver・org・cli に依存しないことを確かめた。2〜4 は `-count=1` なので、org と cli は全部で 4 回(1〜4)走った
- 1 の load average(1 分)は開始時 14.7、pre-bash-guard の suite の間に 66.7、終了時 10.6。2 は開始 7.4、3 は開始 3.7、4 は開始 3.8 だった
- 5 の 21 本は AC の節の表に挙げたテストで、どれも実行され、飛ばされていない
- 12 の感度: canary の PATH で、stub を置かずに `NewHerdr(ExecRunner{}, AgmsgSelfNameOff).TabCreate` を打つ使い捨てのテストを overlay で足すと、canary は `herdr tab create --help` と `herdr tab create --workspace w --cwd /tmp --label l` の 2 回を記録した(help が exit 97 なので `--env` は付かない)。新しいテストが canary を 1 回も呼ばなかったのは、どの呼び出しも各テストの stub に届いたためである

## AC とテストの対応

| AC | テスト | 確かめたこと |
| --- | --- | --- |
| AC1 | `TestAgmsg_EnvRunner_EveryCallRunsWithSelfNameOff`(`internal/org/driver/agmsg_test.go:255`)、`TestAgmsg_RunOnlyRunner_FallsBackToRun`(282)、`TestAgmsg_DeliverySet_InvalidModeRejectedBeforeRunEnv`(301)、`TestExecRunner_RunEnv_AddsToTheParentEnvironment`(`internal/org/driver/driver_test.go:175`)、`TestExecRunner_RunEnv_TimeoutHonored`(226) | Send・Join・TeamMembers・History・Leave・Whoami・DeliverySet の 7 つが RunEnv を env `[AGMSG_SELF_NAME=off]` で 1 回ずつ打ち、Run を打たない。RunEnv を持たない Runner では 7 つとも同じ argv で Run に戻る。`t.Setenv("AGMSG_SELF_NAME", "on")` の下で、本物の `sh` の子は RunEnv では `off|kept`、Run では `on|kept` を出す。mutant D1〜D5 が赤 |
| AC2 | `TestNewOrgRuntimeAt_AgmsgRunnerAddsSelfNameOff`(`internal/cli/org_runtime_env_test.go:25`) | `newOrgRuntimeAt` の Agmsg の Runner が `driver.EnvRunner` で、親の環境が on のとき、代わりの `team.sh`(本物の bash の子)が `off` を読む。mutant C4(Runner を RunEnv のない型で包む)と D1・D2 が赤 |
| AC3 | `TestHerdr_TabCreate_PaneEnvAddsEnvWhenHelpListsIt`(`internal/org/driver/herdr_test.go:317`)、`TestHerdr_TabCreate_PaneEnvConcurrentCallsCheckOnce`(541)、`TestHerdr_TabCreate_PaneEnvKeepsOrder`(447)、`TestNewHerdr_CopiesPaneEnv`(462)、`TestOrgRuntimes_HerdrPaneEnvIsSelfNameOff`(`internal/cli/org_runtime_env_test.go:78`)、`TestOrgSpawn_HappyPath_EventSequenceReceiptAndWorkspaceReuse`(`internal/cli/org_test.go:360`)、`TestOrgDirectorStart_LaunchesRegistersAndStops`(`internal/cli/org_director_headless_test.go:158`) | help が `--env` を挙げる herdr で、tab create の argv が `--label` のあとに `--env AGMSG_SELF_NAME=off` で終わる。help は NewHerdr の値とそのコピーで 1 回、8 つの goroutine で同時に打っても 1 回。CLI の 2 つの組み立て(座席の `newOrgRuntimeAt`、headless の director の `newOrgDirectorHerdrRuntime`)の PaneEnv がちょうど `[AGMSG_SELF_NAME=off]` で、stub の herdr への呼び出しが help、tab create、tab create の順。spawn の 2 つの座席と director start の pane の tab create に `--env` が付く。mutant D6〜D8・D14・D16〜D19・C1〜C3・C5 が赤 |
| AC4 | `TestHerdr_TabCreate_PaneEnvWithoutEnvFlagWarnsOnce`(340、4 subtest)、`TestHerdr_TabCreate_NoPaneEnvRunsNoProbe`(418、3 subtest)、`TestHerdr_TabCreate_PaneEnvLiteralChecksEveryCall`(474)、10 の probe | help に `--env` がない、help が失敗する、help は `--env` を挙げるが 0 以外で終わる、help が時間切れになる、の 4 通りで、`--env` なしの tab create が 2 回成功し、help は 1 回、警告はちょうど 1 行。PaneEnv が空(literal、`NewHerdr(r)`、空の slice)なら help を打たず、argv は今までと同じ。mutant D10・D12・D13・D15・D21 が赤 |
| AC5 | `TestDirectorLaunchScript_TextQuotesEveryValue`(`internal/org/director_cmd_test.go:442`)、`TestDirectorLaunchScript_ChecksInThenExecsTheAgent`(548)、`TestStartDirector_LaunchCmdOpensTheTerminalAndChecksIn`(1186) | 起動スクリプトの全文で、`AGMSG_SELF_NAME='off'` と `export AGMSG_SELF_NAME` が check-in の行の直前にあり、agmsg home がなくても書かれる。端末の環境が `AGMSG_SELF_NAME=on` のとき、check-in の fake ralph と exec された fake agent の両方が `off` を読む。start を通しても agent は `off`。mutant O1〜O5 が赤 |
| 境界 | `TestHerdrHelpEnvFlag`(582)、`TestHerdr_TabCreate_PaneEnvCanceledCallerIsNotCached`(506) | `--envelope`・`--environment`・`--env-file`・`--no-env` を `--env` と見なさない。`-e, --env`、`[--env KEY]`、`--env=…`、行の頭の `--env` は見なす。呼び手の ctx が切れていて失敗した確かめは記録されず、警告も出ず、次の呼び出しでもう一度確かめて `--env` を付ける。mutant D23・D24・D26・D27・D29 と D9・D11 が赤 |

## Mutant

HEAD の 1 ファイルを、文字列 1 か所(1 回だけ現れることを確かめた)を置き換えた写しに `-overlay` で差し替えて打った。driver の mutant は driver のパッケージ全体を、org の mutant は `^(TestDirectorLaunchScript_|TestStartDirector_LaunchCmd|TestWriteDirectorLaunchScript)` を、cli の mutant は AC の表の cli の 4 本を打った。差分と出力は evidence にある。

| ID | 変えたところ | 結果(赤にしたテスト) |
| --- | --- | --- |
| D1 | `Agmsg.run` が RunEnv を使わない | 赤(EveryCall の 7 subtest。cli で打つと AC2 のテストも赤) |
| D2 | RunEnv に env を渡さない(nil) | 赤(EveryCall の 7 subtest、cli の AC2) |
| D3 | 親の環境を env のあとに足す(親の on が勝つ) | 赤(RunEnv の 2 subtest) |
| D4 | env だけを渡し、親の環境を落とす | 赤(RunEnv の 2 subtest。`kept` が消える) |
| D5 | `execRun` が env を `cmd.Env` に入れない | 赤(RunEnv の 2 subtest) |
| D6 | 各 `--env` に PaneEnv の先頭の値を使う | 赤(KeepsOrder) |
| D7 | PaneEnv が空でも確かめる | 赤(NoPaneEnvRunsNoProbe の 3 subtest、既存の `TestHerdr_TabCreate` ほか 2 本) |
| D8 | 確かめの結果を記録しない | 赤(AddsEnv、WithoutEnvFlag の 4 subtest、Concurrent) |
| D9 | ctx が切れたときの失敗も記録する | 赤(CanceledCaller) |
| D10 | ctx ではなく probeCtx を見る(どの失敗も記録しない) | 赤(WithoutEnvFlag の 3 subtest) |
| D11 | 確かめを呼び手の ctx から切り離す | 赤(CanceledCaller) |
| D12 | 確かめに独自の時間切れを持たせない | 赤(WithoutEnvFlag の「help times out」、2.00 秒) |
| D13 | help が 0 以外で終わっても `--env` があれば付ける | 赤(WithoutEnvFlag の「lists --env but exits non-zero」) |
| D14 | 付けられるときも警告する | 赤(AddsEnv、LiteralChecksEveryCall、CanceledCaller) |
| D15 | 警告しない | 赤(WithoutEnvFlag の 4 subtest、LiteralChecksEveryCall) |
| D16 | 共有の確かめを使わない | 赤(AddsEnv、WithoutEnvFlag の 4 subtest、Concurrent) |
| D17 | NewHerdr が確かめを持たせない | 赤(D16 と同じ。cli で打つと `TestOrgRuntimes_HerdrPaneEnvIsSelfNameOff` の 2 subtest も赤) |
| D18 | NewHerdr が呼び手の slice をそのまま持つ | 赤(CopiesPaneEnv) |
| D19 | 確かめの mutex を外す | 赤(Concurrent。help が 8 回) |
| D20 | Warn が nil のとき os.Stderr ではなく io.Discard に書く | **生き残り**。10 の probe を足すと赤(下の G1) |
| D21 | help の失敗の文を 1 行にまとめない | 赤(WithoutEnvFlag の「help fails」) |
| D22 | 警告の中で PaneEnv の要素を `" --env "` ではなく空白で区切る | **生き残り**。どのテストも 1 要素で警告を見る(G3) |
| D23 | 正規表現の前の文字の集合から `[` を外す | 赤(HelpEnvFlag) |
| D24 | 正規表現から `^` を外す | 赤(HelpEnvFlag) |
| D25 | 正規表現の後ろから `$` を外す | **生き残り**(G2) |
| D26 | 正規表現の後ろの集合から `=` を外す | 赤(HelpEnvFlag) |
| D27 | 正規表現の後ろの集合に `-` を足す | 赤(HelpEnvFlag の `--env-file`) |
| D28 | 正規表現の前の部分 `(^|[\s,\[(])` をまるごと外す | **生き残り**(G2) |
| D29 | 後ろの部分を `\b` にする | 赤(HelpEnvFlag の `--env-file`) |
| D30 | `(?m)` を外す | 生き残り。等価: 行の境目の `\n` は前後どちらの集合の `\s` にも入るので、`^` と `$` が行の境目で合うかどうかは結果を変えない |
| O1 | 起動スクリプトから AGMSG_SELF_NAME の 2 行を消す | 赤(TextQuotes、ChecksIn の 2 subtest、LaunchCmdOpens の 2 subtest) |
| O2 | agmsg home があるときだけ書く | 赤(TextQuotes、ChecksIn の「no agmsg home」、LaunchCmdOpens の claude) |
| O3 | check-in の行のあとに書く | 赤(TextQuotes、ChecksIn の 2 subtest。LaunchCmdOpens は agent だけを見るので緑) |
| O4 | 値を `'0'` にする | 赤(TextQuotes、ChecksIn の 2 subtest、LaunchCmdOpens の 2 subtest) |
| O5 | `export` の行を書かず、代入だけにする | 赤(TextQuotes だけ。下の注) |
| C1 | `newOrgRuntimeAt` の Herdr を `NewHerdr(runner)`(PaneEnv なし)に | 赤(`TestOrgRuntimes_HerdrPaneEnvIsSelfNameOff/newOrgRuntimeAt`、spawn の happy path) |
| C2 | `newOrgRuntimeAt` の Herdr を literal の `Herdr{R: runner}` に | 赤(C1 と同じ) |
| C3 | `newOrgDirectorHerdrRuntime` の Herdr を literal に | 赤(`TestOrgDirectorStart_LaunchesRegistersAndStops` の 2 subtest、`TestOrgRuntimes_…/newOrgDirectorHerdrRuntime`) |
| C4 | `newOrgRuntimeAt` の Agmsg の Runner を `struct{ driver.Runner }` で包む(RunEnv が見えない) | 赤(`TestNewOrgRuntimeAt_AgmsgRunnerAddsSelfNameOff`) |
| C5 | `newOrgDirectorHerdrRuntime` を `NewHerdr(driver.ExecRunner{})` に | 赤(C3 と同じ) |

- 40 本のうち 35 本が赤、等価が 1 本(D30)、生き残りが 4 本(D20・D22・D25・D28)。driver の生き残りは driver のパッケージ全体で打った。org と cli のテストで help を答える stub は 2 つ(`internal/cli/org_test.go` の `herdrStub` と `internal/cli/org_runtime_env_test.go` の stub)で、どちらも `      --env <KEY=VALUE>` を返すので、これらの mutant を org と cli で打っても結果は変わらない(org の HerdrClient は fake で、`driver.Herdr` を通らない)
- O5 の注: 挙動を見る 2 本は、どちらも端末の環境に `AGMSG_SELF_NAME=on` を置いて打つ。sh では、環境から受け取った変数への代入は子にも渡るので、`export` がなくても子は `off` を読む。端末の環境に AGMSG_SELF_NAME がないときだけ違いが出る(子は値を受け取らない)。この違いは全文を比べる TextQuotes が固めている
- 10 の probe は、D20 のほか D13 と C1 とも組み合わせて打った。D20 と C1 では probe が赤になり(警告の行が 0)、D13 では緑だった。D13 は help が `--env` を挙げて失敗する場合の mutant で、probe の help はどちらの mode も `--env` を挙げないためである

## AC4 の CLI の probe(10)

`internal/cli` に overlay で足した使い捨ての test file で、`setupOrgStubPATH` の stub の前に、`tab create --help` だけに答える herdr を置いた。mode `noenv` は `--env` のない help(`--envelope` はある)を exit 0 で、mode `fail` は stderr に 1 行を出して exit 2 で返し、ほかの呼び出しは stub に渡す。os.Stderr は pipe に差し替えて読んだ。

| verb | mode | 結果 |
| --- | --- | --- |
| `ralph org spawn` を 2 回(座席 2 つ、同じ org) | noenv、fail | 2 回とも err nil で `spawned seat` を出した。stub の log の tab create は 2 回とも `--env` なし。os.Stderr の警告は 2 行で、spawn 1 回(runtime 1 つ)につき 1 行。cobra の stderr は空 |
| `ralph org director start --driver claude --model opus` | noenv、fail | err nil で `started d1 ` を出した。herdr の呼び出しに `--env` はない。os.Stderr の警告は 1 行。cobra の stderr は空 |

AC4 の「spawn と start は今までどおり成功し、stderr に警告が 1 行出る」は、CLI の水準でも成り立つ。警告は cobra の stderr(`cmd.ErrOrStderr()`)ではなく os.Stderr に直に出る。verify の報告の V4 が書いたとおりである。

## Coverage

- Statement: org 94.7%、driver 94.6%、protocol 100.0%、cli 88.4%(4 の go1.26.0、パッケージごと)。4 パッケージの合計は 92.2%
- Branch: Go の標準のツールにはないので測っていない
- Function(変えた関数): `Agmsg.run` 100%、`ExecRunner.Run` 100%、`ExecRunner.RunEnv` 100%、`execRun` 94.1%、`NewHerdr` 100%、`Herdr.TabCreate` 83.3%、`paneEnvSupported` 100%、`warnNoPaneEnv` 85.7%、`newOrgRuntimeAt` 100%、`newOrgDirectorHerdrRuntime` 100%、`directorLaunchScript.text` 100%
- Notes: 変えた関数の中で通らない block は 5 つ。`warnNoPaneEnv` の `w = os.Stderr`(herdr.go:313-315、D20 が生き残った行)だけが今回の変更の行で、残りの `TabCreate` の Run の失敗と結果の解析の失敗(herdr.go:249-251、264-266、267-269)と `execRun` の stderr のない失敗(driver.go:95)は、base にもある行である

## Failure analysis

| Test | Error | Root cause | Proposed fix |
| --- | --- | --- | --- |
| なし | - | 1〜12 のどの実行でも失敗はなかった | - |

load に弱いことが分かっているテスト(`TestStopDirectorProcess_AProcessThatIgnoresSIGTERMIsNotEndedInTime`、`TestWakeDirector_AFailedWakeIsNotTriedAgainForFiveMinutes/no_return_in_time` ほか)も、org と cli の 4 回の実行で 1 度も落ちなかった。

## Regression checks

| Previously broken behavior | Status | Evidence |
| --- | --- | --- |
| driver の今までのテスト(argv の assert を含む)が HEAD のコードで変わらず通る | 通る | 11: base 1e47c309 の driver の test file 3 つ(51 本)を HEAD のコードで打って rc 0 |
| 今までの spawn・director・driver のテストが argv を変えずに通る | 通る | test file から消えた行は、director start の tab create の期待(`--env AGMSG_SELF_NAME=off` を足した。AC3 の変更)と、起動スクリプトの全文と fake の記録(AGMSG_SELF_NAME を足した。AC5 の変更)だけ。ほかのテストは手を入れずに 1〜4 で通った |
| literal の `Herdr{R: …}`(テストの fake の書き方)の tab create は今までと同じ argv で、help を打たない | 通る | `TestHerdr_TabCreate_NoPaneEnvRunsNoProbe/literal`、既存の `TestHerdr_TabCreate`。D7 で既存の 3 本も赤になる |
| 所見 F3(agmsg 1.5.1 が座席の herdr の agent の名前を変える) | ralph の側の 3 経路(ralph が打つ agmsg、座席と director の pane、launch_cmd の起動スクリプト)を AC1〜AC5 のテストが固める | 上の AC の表と mutant。本物の herdr と agmsg での確かめは S1 の evidence(`docs/evidence/agmsg-self-name-2026-10-22.md` の (e))で、今回は打っていない |

## Test gaps

- G1(LOW): `Warn` が nil のときに os.Stderr へ書くことと、AC4 の CLI の水準の挙動(spawn と director start が成功して os.Stderr に 1 行)を固めるテストはない。D20 が生き残った。10 の probe が D20 を赤にするので、spawn の 1 回分(mode `noenv`)を `internal/cli` の常設のテストにすれば足りる。os.Stderr を差し替えるので、そのテストは `t.Parallel()` にしない。verify の V4 を直して警告を `cmd.ErrOrStderr()` に出すなら、os.Stderr の差し替えは要らなくなる
- G2(LOW): `TestHerdrHelpEnvFlag` の行に、help の文字列が `--env` で終わる場合(`ExecRunner` は出力の前後の空白を落とすので、最後の行が `--env` だけの help がこれに当たる)と、`--env` の前に区切りでない文字が来る場合(`x--env`)がない。D25 と D28 が生き残った。`--no-env` の行は `--env` という部分文字列を含まないので、前の部分をどう変えても赤にならない。足すなら `"Options:\n      --env": true` と `"      --keep--env": false` の 2 行で、この 2 行は HEAD で緑、D25 と D28 でそれぞれ赤になることを overlay の使い捨てのテストで確かめた
- G3(INFO): 警告の中の PaneEnv の要素の区切り(D22)は、2 要素の PaneEnv で警告を見るテストがないので固まっていない。本番は 1 要素だけを渡す
- G4(INFO): `ralph org start --plan` の leader の pane は、spawn と同じ `newOrgRuntimeAt` と同じ `TabCreate` の呼び出し(`internal/org/spawn.go:904`)を通るが、start の CLI のテストで tab create の argv の `--env` を見るものはない。組み立ては `TestOrgRuntimes_HerdrPaneEnvIsSelfNameOff` が固めている
- G5(INFO、挙動の観察): headless の director の pane でも、警告の文は「an agmsg run inside a seat may rename the seat's herdr agent」と座席の言葉で書かれる(10 の probe の director start の出力)。AC4 は 1 行の警告を求めるだけなので、AC は満たす
- 確かめていないこと: 本物の herdr と agmsg での `ralph org spawn` と headless の director、0.7.5 より古い herdr の挙動、help が `--env` を挙げるのに tab create が `--env` を断る herdr(spawn は失敗する。D3 の設計どおりで、テストはない)、codex の座席、ALERT の時間切れの原因

## Verdict

- Pass: run-test.sh(shell 43 本、Go 8 パッケージ)、org・cli・driver・protocol の `-count=1` を go1.26.0 と go1.25.8 で、新しい 21 本の `-v`(SKIP 0)、確かめの同時実行のテストの `-count=3`・`-count=200`・`-race`(負荷の下を含む)、回帰の 51 本。mutant は 40 本のうち 35 本が赤で、生き残りは等価の 1 本と、G1〜G3 の穴に当たる 4 本
- Fail: なし
- Blocked: なし

## テストのあとの直し(報告のあと)

- G1・G2 と、警告の文の G5・verify の V4 は 9c88b759 で直した。警告の行は `warning: ` で始め(CLI のほかの警告と同じ形)、「an agmsg run inside such a pane (a seat's or a headless director's) may rename the pane's herdr agent」と座席と headless の director の両方を指す文にした。出先は今までどおり Warn(nil なら os.Stderr)で、`cmd.ErrOrStderr()` には変えていない
- G1: driver に `TestHerdr_TabCreate_PaneEnvNilWarnWritesToStderr`(Warn が nil のとき os.Stderr に 1 行)を、CLI に `TestOrgVerbs_HerdrWithoutEnvFlagWarnsOnStderr`(`internal/cli/org_runtime_env_test.go`。spawn と director start の 2 つの subtest)を足した。CLI の herdr の stub は `ORG_STUB_TAB_CREATE_HELP=noenv` のとき `--env` のない help を返し、既定の help は変わらない。G2: `TestHerdrHelpEnvFlag` に `"Options:\n      --env": true` と `"      --keep--env": false` の 2 行を足した
- 確かめたこと(implementer の報告): `gofmt -l` は空、`go vet` は rc 0、`go test -count=1 -timeout 30m` は driver・org・cli で rc 0、`GOTOOLCHAIN=go1.25.8` でも同じ 3 つで rc 0。新しいテストは `-v` で PASS(SKIP 0)。mutant は D20 で driver と CLI の新しいテストが赤、D25 と D28 で `TestHerdrHelpEnvFlag` が赤、stub が mode を無視する mutant で CLI のテストが赤になり、手を加えない tree ではどれも緑
- G3(警告の区切り、INFO)と G4(start --plan の CLI テストが `--env` を見ない、INFO)は直さない。G3 は本番の PaneEnv が 1 つだけで、G4 は組み立て側のテストが固めている
- この直しは、続く `/cross-review` の差分に入る
