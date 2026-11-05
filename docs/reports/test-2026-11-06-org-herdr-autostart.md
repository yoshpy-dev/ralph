# Test report: org-herdr-autostart

- Date: 2026-11-06(run-test.sh は 03:57:58 から 04:15:48 JST)
- Plan: `docs/plans/active/2026-11-06-org-herdr-autostart.md`(`## Test plan`、AC1〜AC9)。spec は `docs/specs/2026-11-06-org-herdr-autostart-panes.md` の A
- Tester: tester サブエージェント(`/test`、pipeline の cycle 1)
- Scope: `git diff c206fdce...HEAD`(HEAD d50a20c8)。`./scripts/run-test.sh` は changed scope のまま打った。`.harness/state/verify-scope` は `scope=changed`、`reason=changed_languages`、`languages=golang`。Go の全パッケージ(10)と、local verifier(`scripts/verify.local.sh`)が回す `tests/test-*.sh` の 45 本が走った。production code もテストも worktree では変えていない。gap を確かめるテストを 1 ファイル書いたが、worktree の外(`/tmp/oha-gap/`)に置き、`go test -overlay` で打った
- Evidence: `docs/evidence/test-2026-11-06-org-herdr-autostart.log`(`.gitignore:61` の `docs/evidence/*.log` に当たるのでコミットしない)。run-test.sh の全出力の後ろに、load と herdr のプロセスの記録、変異の定義と結果、gap を確かめるテストの本文と結果、coverage を足した。wrapper 自身のログは `docs/evidence/verify-2026-11-05-185758.log`(同じく対象外。名前の時刻は UTC)。実機の確かめ(S5)は `docs/evidence/org-herdr-autostart-2026-11-06.md`

## Test execution

| Suite / Command | Tests | Passed | Failed | Skipped | Duration |
| --- | --- | --- | --- | --- | --- |
| `./scripts/run-test.sh`(changed scope、rc 0) | shell 45 本 + Go 10 パッケージ | shell 45 本 OK(PASS を含む行 5,195)、Go 10 パッケージ ok | 0 | 2 行(下の注) | 1,070 秒。1 分の load は開始時 21.65、終了時 4.77。1 分ごとの記録の最大は開始時の 21.65 |
| └ Go: `internal/org` | — | ok | 0 | 0 | 560.6 秒 |
| └ Go: `internal/cli` | — | ok | 0 | 0 | 388.9 秒 |
| └ Go: `internal/config`、`internal/upgrade`、`internal/gitroot`、`internal/insights` | — | ok | 0 | 0 | 10.7 秒、4.3 秒、4.3 秒、3.5 秒 |
| └ Go: `internal/org/driver`、`internal/scaffold`、`internal/org/protocol`、`cmd/ralph` | — | ok | 0 | 0 | 3.0 秒、2.5 秒、1.8 秒、0.8 秒 |
| 変異の基準の回(変異なし、絞った `-run`) | driver 5 本、org 8 本、cli 30 本(top-level) | すべて PASS | 0 | 0 | 2 秒、2 秒、9 秒 |
| 変異 41 個(`go test -c -overlay`、絞った `-run`。下の「変異」) | 41 | 40 個が死んだ。1 個(C21)は既存のテストで生き残った | — | — | 1 つ 1〜12 秒 |
| gap を確かめるテスト 1 本(overlay)。HEAD で `-count=3`、C21 のもとで 1 回 | 1 本 | HEAD 3/3 PASS。C21 では FAIL(期待どおり) | — | — | 0.76〜0.82 秒 |
| 変更行の coverage の回(driver 全体、org `-run HerdrServerGate`、cli `-run Herdr -coverpkg=./internal/cli/,./internal/org/`) | — | 3 パッケージ ok | 0 | 0 | 1.5 秒、1.4 秒、7.8 秒 |

起動の仕方: run-test.sh は Bash tool の 600 秒の上限を超えるので、perl の fork と setsid で runner(`/tmp/oha-test/runner.sh`)を切り離して起動した。シグナルは DEFAULT に戻し、標準の入出力は `/dev/null` にした。runner は log、1 分ごとの load、5 秒ごとの `herdr server` のプロセス、開始・終了・rc の meta を書いた。待つあいだは `caffeinate -w` で前景で待ち、meta を読んだ。走っている間に、変異を 1 つずつ順に打った。開始時の load はほかのセッションの Swift のビルドで 21〜31 あり、5 分ほどで 10 を切った。`(cached)` のパッケージはなかった。

Skipped の 2 行は前の report と同じで、どちらもこの diff と関係のない手元の環境による。`tests/test-plan-visual.sh` の browser detection(`/Applications` にブラウザがある)と、`tests/test-secret-scan-branch.sh` の git 2.41 より古い本物の git の case(手元は git 2.49.0。stub の古い git の case は通った)である。shell の 45 本と PASS の行の数(5,195)は、前の 2 本の report と同じである。

## 本物の herdr を起動していないこと(AC8)

- run-test.sh の中で `TestOrgHerdrServerLaunch_TestMainMakesARealLaunchPanic` が通った。TestMain の `orgHerdrServerLaunch` が panic すること、verb に渡るゲートが TestMain の常に通すゲートであることを確かめるテストである。テストが本物の起動を試みれば `internal/cli` の test binary が panic して FAIL になるが、`internal/cli` は ok だった
- 本物のゲート(`orgHerdrServerGateOverride` が nil)を使うテストは `TestOrgHerdrServerGate_EachVerbStartsAStoppedServerAndGoesOn` だけで(`setOrgHerdrServerGateOverride(t, nil)` を grep して 1 件)、そのテストは状態の読み取り・lookup・起動をすべて fake にしている。`internal/org` のテストは `HerdrServerGate` を nil(何もしない)か stub にしている
- runner が 5 秒ごとに `pgrep -f 'herdr server'` を記録した。run-test.sh の 1,070 秒を通じて見えた `herdr server` のプロセスは、ユーザーのサーバー(pid 22408、11/5 18:44:59 開始、`/opt/homebrew/bin/herdr server`)だけだった。ほかに見えた 3 つの pid(75662、69241、70792)は、コマンドの文字列にその語を含む自分の zsh の Bash tool の行で、時刻が自分のコマンドと一致した。2 つ目の `herdr server` は `already running` ですぐ終わるので、5 秒ごとの記録だけでは起動がなかったとは言い切れない。上の panic のほうが主な根拠である
- 変異の回と gap を確かめる回も、起動は fake か TestMain の panic のままで打った。ユーザーの herdr には触れていない

## Coverage

- 変更した行(テストを除く `.go`、`git diff -U0 c206fdce..HEAD` の足された行): 足された行を含む profile のブロックは 127 statement で、123 が打たれていた(96.9%)。profile は driver の全体、org の `-run HerdrServerGate`、cli の `-run Herdr`(`-coverpkg` で cli と org)を合わせたもので、絞った回の値なので下限にあたる
  - `internal/org/driver/herdr.go` 14/14(`ServerStatus` 100%)、`internal/org/herdrgate.go` 3/3、`spawn.go` 3/3、`feature.go` 3/3、`director_start.go` 3/3、`internal/cli/org.go` 17/17、`org_director_headless.go` 8/8
  - `internal/cli/org_herdr_server.go` 72/76。打たれていない 4 つ: 本番の状態の読み取り(`:64`、`driver.ExecRunner` で本物の herdr を打つので、テストでは意図して打たない)、`await` の読み取りのあとで期限が過ぎたときの `break`(`:186`、時刻しだいの分岐)、`announce` の socket が空のときの `not reported`(`:221`)、`os.UserHomeDir` が失敗したときの `dir = ""`(`:276`)
- 関数ごと(cli の絞った回): `ensureOrgHerdrServer`・`afterExit`・`wireOrgHerdrServerGate`・`orgHerdrServerEnv`・`orgHerdrServerDropsEnv` は 100%、`await` 94.1%、`orgHerdrServerLaunchRequest` 75.0%、`announce` 66.7%
- パッケージ全体の statement coverage は測り直していない(driver の全体の回だけは 96.3%)
- Branch: 計測なし(Go の道具は statement だけ)

## Failure analysis

| Test | Error | Root cause | Proposed fix |
| --- | --- | --- | --- |
| (なし) | — | — | — |

run-test.sh、基準の回、coverage の回、gap を確かめるテストの HEAD の回のどれにも失敗はなかった。flaky として memory にあるテスト(`TestRunWatcher_TimeoutIndependentOfSmallInterval`、`TestOrgSpawn_HelloPastTimeoutMS_PrintsTheDeadlineHint` など)も落ちなかった。

## Test plan の項目とテストの対応

テストはどれも run-test.sh の中で通った。

| Test plan の項目 | テスト |
| --- | --- |
| driver の状態の読み取り(argv、JSON の 4 通り) | `internal/org/driver/herdr_test.go` の `TestHerdr_ServerStatus_ReadsTheBareJSONObject`、`_UnreadableOutputIsAnError`、`_RunnerErrorIsWrapped`、`_UnknownJSONOptionIsUnsupported`、`_OtherUnknownOptionIsNotUnsupported`(`--jsonl` と `--json-lines` に当たらないこと) |
| CLI の環境の整理(落とす変数と残す変数を名前で固定、AC2) | `TestOrgHerdrServerEnv_DropsThePaneAndClaudeCodeVariablesOnly`(`CLAUDE_CODE` と `CLAUDECODE_X` は残す、PWD の置き換え)、`TestOrgHerdrServerGate_AStoppedServerIsStartedAndAnnounced`(起動の要求に渡る環境と HOME の作業ディレクトリ) |
| 動いている → 起動しない(AC3) | `TestOrgHerdrServerGate_ARunningServerStartsNothing`(`compatible` が true・false・null の 3 通り、読み取りは 5 秒で区切る) |
| 止まっている → 起動して待つ → stderr の 1 行(AC1) | `TestOrgHerdrServerGate_AStoppedServerIsStartedAndAnnounced`(3 つの verb の文)、`TestOrgHerdrServerGate_EachVerbStartsAStoppedServerAndGoesOn`(spawn、start `<task>`、director start が本物のゲートを通って進む) |
| 起動の失敗 → 読み直して running なら進む(AC5) | `TestOrgHerdrServerGate_ALaunchThatFails`(3 通り) |
| 子が先に終わって running でない → ログのパス付きで失敗(AC4、AC5、D4) | `TestOrgHerdrServerGate_AServerThatExitsFirst`(4 通り) |
| 10 秒の待ち切れ → 失敗(AC4) | `TestOrgHerdrServerGate_NoAnswerWithinTheDeadline`(期限を 200ms に縮める。読み取りの context が期限を越えないこと、サーバーを止めないこと) |
| herdr が PATH に無い → 失敗(AC4) | `TestOrgHerdrServerGate_HerdrNotOnPATH`(lookup の失敗と、状態の読み取りの `exec.ErrNotFound`) |
| 読み取りの打ち切り・runner のエラー・壊れた JSON・`running` の無い JSON → 失敗(AC4、D3) | `TestOrgHerdrServerGate_AStatusThatCannotBeReadFails`(本物の `ServerStatus` を通す 4 通り) |
| `unknown option: --json`(標準出力と runner のエラー)→ 警告して通す(D3) | `TestOrgHerdrServerGate_AHerdrWithoutStatusJSONWarnsAndPasses` |
| org の Spawn・StartFeature・StartDirector でゲートを呼ぶ位置と、失敗で何も書かないこと(AC3、AC4) | `internal/org/herdrgate_test.go` の 8 本。Spawn: 失敗で manifest・receipt・herdr・agmsg・spawn lock が変わらない、`--dry-run` は呼ばない、入力の検査と見張りのゲートのあと。StartFeature: 失敗で worktree を作らない。StartDirector: 失敗で書かず herdr を呼ばない、入力の検査のあと、`launch_cmd` の形は呼ばない |
| director start で、herdr の headless の登録が残っているときゲートが `CheckDirector` より先(AC9) | `TestStartDirector_HerdrServerGate_RunsBeforeTheCheckOfAHeadlessHerdrDirector`、`TestStartDirector_HerdrServerGate_AFailureWritesAndCallsNothing/a_current_headless_director_in_herdr` |
| CLI の start・spawn・director start で、ゲートの失敗で verb が失敗し台帳に何も無いこと(AC4) | `TestOrgSpawn_HerdrServerGate`(`--dry-run` は呼ばない subtest を含む)、`TestOrgStart_HerdrServerGate_ARefusalRecordsNothing`、`TestOrgStartPlan_HerdrServerGate_ARefusalMakesNoWorktree`(worktree もブランチもない)、`TestOrgDirectorStart_HerdrServerGate_ARefusalWritesNothing` |
| 3 つの `--help`(AC7) | `TestOrgHerdrServerGate_TheHelpOfEachVerbSaysItStartsTheServer` |
| 本物の herdr を起動しない(AC8) | `TestOrgHerdrServerLaunch_TestMainMakesARealLaunchPanic`(上の節) |
| 既存の org と CLI のテストが TestMain の差し替えの下で通る(Regression) | run-test.sh の `internal/org`(560.6 秒)と `internal/cli`(388.9 秒)が ok |
| ソケットのファイルが残った状態と無い状態、起動したサーバーの環境(`ps -E`) | 実機だけ。S5 の記録 `docs/evidence/org-herdr-autostart-2026-11-06.md`。この /test では打ち直していない(本物の herdr を起動しないため) |

## 変異

どの変異も production のファイルの写しを `go test -c -overlay` で差し替え、worktree のファイルは変えていない。perl で 1 か所だけ置き換え、置き換える文字列が 1 か所でなければ止めた。打つ前と後で `git status --porcelain --ignored`(`.harness/cache/` を除く)が同じであることも確かめた。絞った `-run` は、cli が `Herdr`、org が `HerdrServerGate`、driver が `ServerStatus`。

| 変異 | 変えたところ | 殺したテスト |
| --- | --- | --- |
| C1b | 動いているサーバーを通す case を外す | `TestOrgHerdrServerGate_ARunningServerStartsNothing` |
| C2 | 古い herdr の警告の case を外す | `TestOrgHerdrServerGate_AHerdrWithoutStatusJSONWarnsAndPasses` |
| C3 | 読み取りの `exec.ErrNotFound` の case を外す | `TestOrgHerdrServerGate_HerdrNotOnPATH/the_status_read_finds_no_herdr` |
| C4 | 読めない状態を通す | `TestOrgHerdrServerGate_AStatusThatCannotBeReadFails`(4 通りとも) |
| C5 | 起動の失敗のあと running でも失敗にする | `TestOrgHerdrServerGate_ALaunchThatFails/another_server_answers` |
| C6 | 起動の失敗を通す | `TestOrgHerdrServerGate_ALaunchThatFails/no_server` |
| C7 | 子が終わっていても 1 行を出す | `TestOrgHerdrServerGate_AServerThatExitsFirst/another_server_answers_on_the_first_read` |
| C8 | 1 行を出さない | `AStoppedServerIsStartedAndAnnounced`、`EachVerbStartsAStoppedServerAndGoesOn` |
| C9 | 待ちの間の読み取りを期限で区切らない | `TestOrgHerdrServerGate_NoAnswerWithinTheDeadline` |
| C10 | 子が終わったら読み直さずに通す | `AServerThatExitsFirst` の 2 つ |
| C11 | 子が終わって running でないときに通す | `AServerThatExitsFirst/no_server` |
| C12 | 待つ期限を 0 にする | 起動と待ちのテスト 4 本 |
| C13、C14、C16b | 環境から `CLAUDE_CODE_*`、`CLAUDECODE`、herdr の pane の変数を落とさない | `TestOrgHerdrServerEnv_DropsThePaneAndClaudeCodeVariablesOnly`、`AStoppedServerIsStartedAndAnnounced` |
| C15 | PWD を HOME に置き換えない | 同上 |
| C17 | 作業ディレクトリを HOME にしない | `AStoppedServerIsStartedAndAnnounced` |
| C18 | ログのファイル名を `herdr-server.log` にする | 起動と待ちのテスト 3 本 |
| C19 | 差し替えのゲートを verb につながない | CLI の拒否のテスト 4 本 |
| C20、C22、C23、C24 | start --plan、start `<task>`、spawn、director start のつなぎを外す | それぞれの CLI の拒否のテスト(C22〜C24 は `EachVerbStartsAStoppedServerAndGoesOn` も) |
| C25、C26 | director start、start `<task>` の verb の文を変える | `EachVerbStartsAStoppedServerAndGoesOn` |
| C21 | start --plan(`runOrgStartPlan`)の verb を `"start"` から `"spawn"` にする | **既存のテストでは死なない**。gap を確かめるテストが殺した(下の Test gaps の 1) |
| O1 | Spawn の `--dry-run` でもゲートを呼ぶ | `TestOrgSpawn_HerdrServerGate_ADryRunDoesNotAskIt` |
| O2、O6 | Spawn、StartFeature のゲートの失敗を `rejected` にする | Spawn の 2 本、StartFeature の 1 本 |
| O3、O5、O8 | Spawn、StartFeature、StartDirector のゲートを外す | それぞれの失敗のテスト(O3 は StartFeature のテストも) |
| O4 | Spawn のゲートを見張りのゲートの前に移す | Spawn の 3 本と StartFeature の 1 本 |
| O7 | `launch_cmd` の形でもゲートを呼ぶ | `TestStartDirector_HerdrServerGate_ALaunchCmdStartDoesNotAskIt` |
| O9 | `checkHerdrServerGate` がゲートの失敗を捨てる | Spawn の 2 本、StartFeature と StartDirector の失敗のテスト各 1 本 |
| O10 | director start のゲートを残った登録の確認(`checkCurrentDirector`)のあとに移す(AC9) | `RunsBeforeTheCheckOfAHeadlessHerdrDirector`、`AFailureWritesAndCallsNothing/a_current_headless_director_in_herdr` |
| D1 | 古い herdr の判定から語の境界を外す | `TestHerdr_ServerStatus_OtherUnknownOptionIsNotUnsupported` |
| D2 | runner のエラーの文を見ない | `TestHerdr_ServerStatus_UnknownJSONOptionIsUnsupported/only_in_the_Runner_error's_text` |
| D3 | `running` が無い JSON を通す | `TestHerdr_ServerStatus_UnreadableOutputIsAnError/JSON_without_running` |
| D4 | `compatible` を捨てる | `TestHerdr_ServerStatus_ReadsTheBareJSONObject` |
| D5 | argv から `server` を落とす | driver の 5 本すべて |

C1 と C16 の最初の形は、使われない変数と import が残って compile に失敗したので、compile できる形(C1b、C16b)で打ち直した。C21 は、変わった verb が出力に出るのが本物のゲート(差し替えが nil のときの closure)を通るときだけで、それを通すテストは start `<task>`・spawn・director start の `EachVerbStartsAStoppedServerAndGoesOn` だけである(grep で確かめた)。そのため `internal/cli` の全体を打っても死なない。全体の回は打っていない。

## Regression checks

| Previously broken behavior | Status | Evidence |
| --- | --- | --- |
| 事案: herdr のサーバーが止まっていると `ralph org start --plan` が worktree とブランチを作ってから `workspace_create` で失敗し、両方が残った | OK | `TestStartFeature_HerdrServerGate_AFailureMakesNoWorktree`(org)と `TestOrgStartPlan_HerdrServerGate_ARefusalMakesNoWorktree`(CLI)が通った。StartFeature のゲートを外す O5 と start --plan のつなぎを外す C20 は、どちらもこれらで死んだ |
| 既存の org と CLI のテスト(TestMain の差し替えの下) | OK | run-test.sh の `internal/org` と `internal/cli` が ok。spawn・start・director start の既存のテストは、TestMain の常に通すゲートで herdr のサーバーなしに通った |
| shell のテストへの影響 | OK | 45 本すべて OK、PASS の行は前の report と同じ 5,195 |

## Test gaps

1. start --plan の stderr の 1 行と失敗の文の verb(`ralph org start`)を固定するテストがない。C21(`runOrgStartPlan` の verb を `"spawn"` にする)は既存のテストで生き残り、そのもとでは start --plan が herdr を起動したときの 1 行が「ralph org spawn: started the herdr server ...」になり、失敗の文は「run ralph org spawn again」と言う。worktree の外に書いた `TestOrgStartPlan_HerdrServerGate_StartsAStoppedServerOnceInStartsWords`(`/tmp/oha-gap/org_herdr_server_plan_gap_test.go`。本文は evidence の log にもある)が、HEAD で 3/3 通り、C21 を殺した。このテストは、計画の D1 が書く「StartFeature の中の Spawn もゲートを呼ぶが、動いていれば読み取り 1 回で終わる」も固定する(起動 1 回、読み取り 3 回)。今は CLI でこれを見るテストがない。足すかは orchestrator が決める
2. `org_herdr_server.go` の 3 つの分岐を打つテストがない。socket が空のときの `not reported`(`:221`)、`os.UserHomeDir` の失敗(`:276`)、待ちの中で期限が読み取りの直後に過ぎたときの `break`(`:186`)である。どれも文の中身か時刻しだいの分岐で、危険は小さい
3. 本番の起動(`startDetachedProcess` に渡る `herdr server`、環境、作業ディレクトリ)と本番の状態の読み取り(`:64`)は、AC8 のとおりテストでは打たない。実機の確かめ(S5)の記録だけが見ている。ソケットのファイルが残った状態も同じ
4. `TestOrgHerdrServerGate_EachVerbStartsAStoppedServerAndGoesOn` は start `<task>`・spawn・director start だけで、start --plan を含まない(gap 1 と同じ穴)
5. パッケージ全体の statement coverage は測り直していない(Coverage の節)
6. `-race` は打っていない。計画の Test plan は求めていない。ゲートは goroutine を足さず、`await` は子の `exited` のチャネルを読むだけである
7. 5 秒ごとの `herdr server` の記録は、すぐ終わる 2 つ目のサーバーを見逃しうる(AC8 の節)。テストが本物の起動をしないことの根拠は、TestMain の panic と、それを確かめるテストのほうである

## Verdict

- Pass: `./scripts/run-test.sh`(rc 0。shell 45 本、Go 10 パッケージ、`(cached)` なし)。計画の Test plan の項目には、実機だけの項目を除き、どれも run-test.sh の中で通ったテストがある(上の対応表)。本物の `herdr server` を起動しようとしたテストはなかった(AC8 の節)。変更行の coverage は絞った回で 127 statement 中 123(96.9%)。変異 41 個のうち 40 個は既存のテストが殺し、残る C21 は gap を確かめるテストが殺した
- Fail: 0
- Blocked: なし。C21 を殺すテスト(gap 1)を足すかは orchestrator が決める
