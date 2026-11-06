# Test report: org-herdr-single-tab

この report には 2 回分の結果がある。上の「Run 2(cycle 2)」と最初の `## Verdict` が pipeline の cycle 2 の結果である。下の「Run 1(cycle 1)」以下は、cycle 1 の report(d4fa6ce2)の本文を 1 バイトも変えずに残したもので、Go のコメントと `docs/tech-debt/README.md` が引く M1〜M3、P5、「10 of 10」、Test gaps の番号はそちらを指す。

## Run 2(cycle 2)

- Date: 2026-11-06(run-test.sh は 13:03:32 から 13:27:09 JST、`-race` の回は 13:27:16 から 13:38:17 JST)
- Plan: `docs/plans/active/2026-11-06-org-herdr-single-tab.md`(`## Test plan`、AC1〜AC12)
- Tester: tester サブエージェント(`/test`、pipeline の cycle 2。上限の 2 回目)
- Scope: `git diff 920832dd...HEAD`(HEAD 609e95ff)。cycle 1 の report(d4fa6ce2)のあとに入ったコードの変更は dce22057 と f07959d6 である。dce22057 は cross-review の ACTION_REQUIRED 2 件と LOW の直しで、座席 id が org_id と同じときはタブのラベルで受け入れない、director の start を `WorkspaceCreate` → `RecordWorkspace` → `PaneRename` → `RecordPane` → `TabRename` の順にする、`TabCreate` の応答にタブの id が無ければ失敗にする、`--env` の無い herdr の警告文を足す、の 4 つを入れた。あわせて、テスト 3 本(`TestConfirmSeatPane_SeatIDIsTheOrgID_StopDisbandAndOwnPaneClose`、`TestDirector_AStartThatDiedBeforeItsPaneRenameIsClosedByItsWorkspace`、`TestOrgSpawn_Layout_LockHeldUntilThePaneIsLabelled`)と、`TestOrgSpawn_Layout_FailuresBeforeThePane` の P5 の行を足した。f07959d6 は stop と disband の help の文と、テストのコメントだけである。`./scripts/run-test.sh` は changed scope のまま打った(`.harness/state/verify-scope` は `scope=changed`、`reason=changed_languages`、`languages=golang`)。production code もテストも worktree では変えていない。直す前のコードの写しと変異は `/tmp/rt-single-tab-c2/` に置き、変異は `go test -overlay` で差し替えた
- Evidence: `docs/evidence/test-2026-11-06-org-herdr-single-tab-cycle2.log`(`.gitignore:61` の `docs/evidence/*.log` に当たるのでコミットしない)。cycle 1 の log(`docs/evidence/test-2026-11-06-org-herdr-single-tab.log`)は Run 1 の本文が引くので、上書きせずに残した。cycle 2 の log には、run-test.sh の全出力の後ろに、runner、load と herdr のプロセスの記録、キャッシュされたパッケージの打ち直し、直す前のコードでの結果、変異の定義と結果、繰り返しの回、`-race` の回、変更行の coverage を足した。wrapper 自身のログは `docs/evidence/verify-2026-11-06-040332.log`(名前の時刻は UTC)

### Test execution

| Suite / Command | Tests | Passed | Failed | Skipped | Duration |
| --- | --- | --- | --- | --- | --- |
| `./scripts/run-test.sh`(changed scope、rc 0) | shell 45 本 + Go 10 パッケージ | shell 45 本 OK(PASS を含む行 5,195)、Go 10 パッケージ ok | 0 | 2 行(Run 1 と同じ) | 1,417 秒。1 分の load は開始時 8.35、終了時 8.40。記録の最大は shell の段の 13:13:17 の 126.66、Go の段では 13:20:53 の 41.79 |
| └ Go: `internal/org` | — | ok | 0 | 0 | 680.8 秒 |
| └ Go: `internal/cli` | — | ok | 0 | 0 | 512.3 秒 |
| └ Go: `internal/config`、`internal/org/driver`、`internal/insights`、`cmd/ralph` | — | ok | 0 | 0 | 11.5 秒、2.2 秒、1.4 秒、0.7 秒 |
| └ Go: `internal/gitroot`、`internal/org/protocol`、`internal/scaffold`、`internal/upgrade` | — | `(cached)` の ok。4 つとも `internal/org`・`internal/org/driver`・`internal/cli` に依存しない(`go list -deps`)。`-count=1` で打ち直して ok | 0 | 0 | 打ち直しは 2.0 秒、0.8 秒、1.2 秒、1.9 秒 |
| `go test -race ./internal/org/... ./internal/cli/... -count=1 -timeout 30m` | 4 パッケージ | ok。`DATA RACE` の行は 0 | 0 | 0 | 661 秒(org 646.0 秒、cli 466.4 秒、driver 2.4 秒、protocol 2.6 秒)。load は開始時 8.11、終了時 13.92 |
| HEAD で `go test ./internal/org -run '^TestOrgSpawn_Layout_' -count=20` | 14 本 | 280/280 | 0 | 0 | 36 秒(load 23〜29) |
| HEAD で同じものを `-race -count=10` | 14 本 | 140/140 | 0 | 0 | 20 秒 |
| HEAD で confirm・director・補償のテスト 12 本を `-count=20` | 12 本 | 240/240 | 0 | 0 | 6 秒 |
| HEAD で driver の `TestHerdr_*` を `-count=20` | 54 本 | 1,080/1,080 | 0 | 0 | 2 秒 |
| 直す前のコードの写しで cycle 2 のテスト(下の節) | org 11 本、driver の `TestHerdr_*` | — | 意図どおりの赤(下の節) | — | — |
| 変異 17 個(下の節) | 17 | 17 個とも死んだ | — | — | 1 個 24〜56 秒 |
| 新しいロックのテストを M1〜M3 で `-count=10` ずつ | 1 本 | — | 10/10 ずつ赤(期待どおり) | — | 1 秒ずつ |

起動の仕方: run-test.sh は Bash tool の 600 秒の上限を超えるので、runner(`/tmp/rt-single-tab-c2/runner.sh`)を Bash tool の `run_in_background` で起動した。runner は log と開始・終了・rc・load の meta を書いた。待つあいだは runner の pid を `caffeinate -w` で前景で待ち、meta を読んだ。`-race` の回も同じ形で、run-test.sh が終わってから打った。shell の段の途中(13:05〜13:15)に自分の変異の回を重ねたので、1 分の load が 126 まで上がった。Go の段(13:15 から)に重ねたのは、P6 の probe(36 秒)と絞った coverage の回(org 20 秒)などの短い回だけである。

Skipped の 2 行は Run 1 と同じで、`tests/test-plan-visual.sh` の browser detection と、`tests/test-secret-scan-branch.sh` の git 2.41 より古い本物の git の case である。どちらもこの diff と関係のない手元の環境による。

### 本物の herdr に触れていないこと

- 5 秒ごとの `pgrep -fl '^[^ ]*herdr server'` の記録(13:03:33〜13:27:09、274 回)に出たのは、ユーザーのサーバー(pid 22408、`/opt/homebrew/bin/herdr server`)だけだった。今回は run-test.sh の開始から記録した
- cycle 2 で足したテストは org のテストで、`fakeHerdr` か director の片付けの fixture を使う。dce22057 に `ExecRunner` の行は無い。`internal/cli` の変更は help の文だけで、Run 1 の節の 2 点(偽の `herdr` を PATH の先頭に置くこと、`orgHerdrServerLaunch` の panic)は変わっていない

### cycle 2 の直しと、直す前のコードでの赤

直す前のコードの写しは、`git archive HEAD` を `/tmp/rt-single-tab-c2/prefix/` に展開し、production の 5 ファイル(`verbs.go`、`director_start.go`、`driver/herdr.go`、`driver/paneenv.go`、`spawn.go`)を dce22057^ の版に、`internal/cli/org.go` を dce22057 の版に戻して作った。テストは HEAD のままで、`go vet` は通った。

| 直し | テスト | 直す前のコードでの結果 |
| --- | --- | --- |
| ACTION_REQUIRED 1(座席 id が org_id と同じとき、名前の無いペインをタブのラベルで受け入れない) | `TestConfirmSeatPane_TheRule` の新しい 4 行、`TestConfirmSeatPane_SeatIDIsTheOrgID_StopDisbandAndOwnPaneClose` | TheRule の 2 行(「no label in the org's tab」と、stale で別のタブにある行)が赤で、err が nil だった。新しいテストの「an unlabelled pane in the org's tab」は stop・disband・自分のペインを閉じるの 3 つとも赤で、stop は C-c と close を送って `stopped` を記録し、disband はワークスペースまで閉じた。cross-review の所見どおりの閉じすぎである。残りの行は通った |
| ACTION_REQUIRED 2(director の start がペインにラベルを付けてから記録する) | `TestStartDirector_LabelsTheRootPaneThenRecordsIt`、`_APaneRenameThatFailsClosesTheWorkspaceItMade`(2 subtest)、`_AFailedStepClosesWhatItMadeAndRecordsTheStop/pane_rename`、`_ARecordThatFailsStillClosesWhatHerdrMade/pane_created` | 4 本とも赤。前の順(`write pane_created` のあとに `pane_rename`)、trail に残る `pane_created d1`、記録に失敗したときにペインを閉じずワークスペースだけを閉じる呼び出しが出た |
| 同じ(rename の前に死んだ start の片付け) | `TestDirector_AStartThatDiedBeforeItsPaneRenameIsClosedByItsWorkspace` | 通った。このテストは、新しい順で死んだときに残る状態(ワークスペースだけを記録)を台帳に置き、stop と start の片付けがそれを閉じることを固定する。その状態になる順そのものは上の 4 本が固定し、変異 DS1・DS3 もそれらで死ぬ |
| self-review L6(`TabCreate` の応答にタブの id が無ければ失敗) | `TestHerdr_TabCreate_TabIDFallsBackToResultTab` の「no tab id at all」「both tab ids empty」 | 2 行とも赤(`TabCreate = {PaneID:w3:p2 TabID:}, <nil>`) |
| self-review L3(`--env` の無い herdr の警告文) | `TestHerdr_TabCreate_PaneEnvWithoutEnvFlagWarnsOnce`、`_PaneEnvWarningNamesEachEntrysConsequence`、`_PaneEnvNilWarnWritesToStderr`、`_ExtraEnvWithoutEnvFlagWarnsWithItsClause`、`TestHerdr_EnvCheck_WithoutEnvFlagWarnsOncePerSubcommand` | 5 本とも赤。テストが文を書き写して固定しているので、前の文では食い違う |
| Run 1 の Test gaps 1(ロックをペインのラベルまで持つ) | `TestOrgSpawn_Layout_LockHeldUntilThePaneIsLabelled` | 通った(ロックのコードは変わっていない)。赤になることは変異 M1〜M3 で確かめた(下) |
| Run 1 の Test gaps 2(P5) | `TestOrgSpawn_Layout_FailuresBeforeThePane` の「workspace create names a root pane and no tab」 | 通った(判定のコードは cycle 1 からある)。赤になることは変異 P5 で確かめた(下) |
| stop と disband の help(f07959d6) | なし | —(Test gaps の 2) |

### 変異

どの変異も、HEAD の production のファイルの写しを `go test -overlay` で差し替えた。M1〜M3 と P5 は Run 1 と同じ変更を HEAD の `spawn.go` に当て直した。V1〜V7 は Run 1 の変更を HEAD の `verbs.go` に当て直し、`confirmPaneTab` の引数が変わった V1・V5・V7 は同じ意味の置き換えを書き直した。org の `-run` と `-skip` は Run 1 と同じで、1 個ずつ `-timeout 4m` を付けた。compile に失敗した変異はなかった。打つ前と後で `git status --porcelain --ignored`(`.harness/cache/` を除く)の違いは、run-test.sh の wrapper が書いたログ 1 つだけだった。

| 変異 | 変えたところ | 殺したテスト |
| --- | --- | --- |
| M1 | ロックの名前に座席 id を足す | `LockHeldUntilThePaneIsLabelled`、`ConcurrentSpawnsShareOneTab`、`LockHeldPastTheDeadline` |
| M2 | ロックを `spawn_step` の前に放す | `LockHeldUntilThePaneIsLabelled`(Run 1 では既存のテストが 5/50 しか赤にしなかった) |
| M3 | ロックを `spawn_step` のあと、`PaneRename` の前に放す | `LockHeldUntilThePaneIsLabelled`(Run 1 では `internal/org` の全体でも生き残った) |
| P5 | `workspace create` の返事にタブが無いことを見ない | `FailuresBeforeThePane`(Run 1 では全体でも生き残った) |
| C1 | `confirmPaneTab` の `label != workspaceLabel` を外す(直す前の判定) | `TheRule`、`SeatIDIsTheOrgID_StopDisbandAndOwnPaneClose` |
| C2 | 「which is also the workspace's label」の文を足さない | 同じ 2 本 |
| C3 | `confirmPaneTab` に workspaceLabel の代わりに座席 id を渡す(前の版の座席を受け入れなくなる) | `TheRule`、`StopDisbandAndOwnPaneClose`、`TestCompensateStale_ClosesOnlyThePaneItsSpawnMade`、`TestStopDirector_ConfirmsThePaneByItsLabel`、`TestOrgSpawn_StaleInFlight_*` など 16 本 |
| C4 | workspaceLabel に空の文字列を渡す | `TheRule`、`SeatIDIsTheOrgID_StopDisbandAndOwnPaneClose` |
| V1〜V7 | Run 1 の V1〜V7(ラベルとワークスペースの確かめを外す、ゆるめる) | Run 1 で殺したテストがどれも赤になった。V1 と V5 では `SeatIDIsTheOrgID_StopDisbandAndOwnPaneClose` も赤になった |
| DS1 | director の start で、ラベルを付ける前に `reg.PaneID` を入れる(rename に失敗したときの片付けがペインを確かめにいく) | `APaneRenameThatFailsClosesTheWorkspaceItMade`、`AFailedStepClosesWhatItMadeAndRecordsTheStop` |
| DS3 | `RecordPane` が成功したあとで `reg.PaneID` を入れる(記録に失敗するとペインを閉じない) | `ARecordThatFailsStillClosesWhatHerdrMade` |

新しいロックのテストだけを M1・M2・M3 のそれぞれで `-count=10` 打つと、10/10 ずつ赤になった。M3 のもとでは、reviewer の spawn が implementer のペインにラベルが付く前に `workspace_get pane_list pane_split pane_rename agent_start` まで進んだ。HEAD では同じテストが 20/20 通った(上の繰り返しの回)。

ほかに、ペインのラベルを org_id にする P6(Run 1 と同じ)を、新しいロックのテストだけに `-timeout 30s` で打った。テストは赤になったが、`<-h.entered`(`spawn_layout_test.go:557`)で止まり、30 秒の時間切れの panic で落ちた(Test gaps の 1)。

### Coverage

- cycle 2 の直しの変更行(`git diff -U0 e8bfac61..HEAD` のうち、テストを除く `.go` の足された行を含む profile のブロック): 12 statement のうち 12。`director_start.go` 5/5、`verbs.go` 4/4、`driver/herdr.go` 2/2、`driver/paneenv.go` 1/1。`internal/cli/org.go` の変更は help の文字列だけで、statement は無い。profile は spawn・stop・disband・director・confirm に絞った org の回(56.0%。パッケージ全体の値ではない)と、driver の全体(97.1%)から取った
- パッケージ全体の coverage は測り直していない。Run 1 の値(`internal/org` 95.0%、driver 97.1%)のあとの production の変更は、上の 12 statement とコメントと help の文だけである(d4fa6ce2 から e8bfac61 までに `.go` の変更は無い)
- Branch: 計測なし(Go の道具は statement だけ)

### Failure analysis

| Test | Error | Root cause | Proposed fix |
| --- | --- | --- | --- |
| (なし) | — | — | — |

run-test.sh、`-race` の回、HEAD での繰り返しの回のどれにも失敗はなかった。赤になったのは、直す前のコードの写しと変異の回だけで、どれも意図したものである。flaky として memory にあるテスト(`TestRunWatcher_TimeoutIndependentOfSmallInterval`、`TestOrgSpawn_HelloPastTimeoutMS_PrintsTheDeadlineHint` など)も落ちなかった。

### Test plan の項目とテストの対応(cycle 2 で変わった行)

ほかの行は Run 1 の表のとおりで、どのテストも run-test.sh の中で通った。

| Test plan の項目 | テスト |
| --- | --- |
| D4 の規則の表(AC5、AC6) | `TestConfirmSeatPane_TheRule` に、座席 id が org_id と同じ 4 行を足した(16 行と、herdr がペインを知らない subtest)。`TestConfirmSeatPane_SeatIDIsTheOrgID_StopDisbandAndOwnPaneClose`(2 つの状態 × stop・disband・自分のペインを閉じる) |
| AC8(director のルートペイン) | `TestStartDirector_LabelsTheRootPaneThenRecordsIt`(`_RecordsTheRootPaneThenLabelsIt` から名前と期待の順を変えた)、`TestDirector_AStartThatDiedBeforeItsPaneRenameIsClosedByItsWorkspace` |
| AC9(同時の spawn、ロック) | `TestOrgSpawn_Layout_LockHeldUntilThePaneIsLabelled` を足した |
| ペインを作る前の失敗 | `TestOrgSpawn_Layout_FailuresBeforeThePane` が 5 行になった(ルートペインはあるがタブが無い返事) |
| driver の返事(`tab create` のタブの id) | `TestHerdr_TabCreate_TabIDFallsBackToResultTab` の 2 行 |
| AC11(実機) | 実機の記録は cycle 2 の前に取ったもので、cycle 2 の変更は実機で見ていない(Test gaps の 3) |

### Regression checks

| Previously broken behavior | Status | Evidence |
| --- | --- | --- |
| cross-review の ACTION_REQUIRED 1: 座席 id が org_id と同じ座席の記録した id が、org のタブの中の名前の無いペインを指すと、stop が C-c と close を送った | OK | 直す前のコードで `SeatIDIsTheOrgID_StopDisbandAndOwnPaneClose` の 3 subtest が赤(stop は `stopped` を記録し、disband はワークスペースまで閉じた)。HEAD では 20/20 |
| ACTION_REQUIRED 2: director の start がペインを記録してからラベルを付けたので、その間に死ぬと stop も次の start の片付けも断った | OK | 直す前のコードで 4 本が赤、HEAD で 20/20。死んだあとの片付けは `TestDirector_AStartThatDiedBeforeItsPaneRenameIsClosedByItsWorkspace` |
| Run 1 の Test gaps 1 と 2(M3 と P5 が生き残った) | OK | M2・M3・P5 が cycle 2 で足したテストで死んだ |
| 前の版の座席の受け入れ(AC6) | OK | `TheRule` と `StopDisbandAndOwnPaneClose` が通り、受け入れを壊す C3 で赤になる |
| 既存の org と CLI のテスト | OK | run-test.sh と `-race` の回が ok |
| shell のテストへの影響 | OK | 45 本すべて OK、PASS の行は Run 1 と同じ 5,195 |

### Test gaps

1. 新しい `TestOrgSpawn_Layout_LockHeldUntilThePaneIsLabelled` も、Run 1 の Test gaps 3 と同じく、`<-h.entered`(`spawn_layout_test.go:557`)を期限なしで待つ。implementer の spawn が座席 id のラベルを付けにいかない回帰(P6)では、このテストが `go test` の時間切れまで止まり、`-timeout 30s` では 30 秒で panic した。run-test.sh の `-timeout=30m` ではパッケージ全体が 30 分止まる。`docs/tech-debt/README.md` の 100 ms の待ちの行が挙げているのは `ConcurrentSpawnsShareOneTab` の待ちだけである。triage は期限を足さないと決めたので、その行にこのテストも足すかは orchestrator が決める
2. stop と disband の help に足した「unless that tab label is also the org_id」(f07959d6)を固定するテストが無い(`internal/cli` のテストを grep して 0 件)。director の help の同じ形の文は `org_director_test.go:1066` が固定している。文だけの変更で、危険は小さい
3. 実機の確かめ(AC11、`docs/evidence/org-herdr-single-tab-2026-11-06.md`)は cycle 2 の前に取ったものである。director の start の新しい順と、座席 id が org_id と同じときの確かめは、偽物でしか見ていない
4. 座席 id が org_id と同じ前の版の座席(自分のタブの中にある名前の無いペイン)は、cycle 2 から止められない。herdr から見ると「org のタブの中の名前の無いペイン」と同じ状態なので区別できず、テストは断る側(`TheRule` の「no label in the org's tab」)を固定している。振る舞いの変化は tech-debt の行に書いてある
5. Run 1 の Test gaps のうち 5(「two first spawns」はロックが無くても通る)、6(打たれていない 11 statement)、8(`internal/cli` の coverage)は変わっていない。パッケージ全体の coverage は測り直していない

## Verdict

- Pass: `./scripts/run-test.sh`(rc 0。shell 45 本、Go 10 パッケージ。`(cached)` の 4 パッケージは `-count=1` で打ち直して ok)と、`go test -race ./internal/org/... ./internal/cli/...`(ok、`DATA RACE` は 0)。cycle 2 で足したテストと変えたテストは HEAD で 20 回ずつ通り、配置のテスト 14 本は 20 回と `-race` の 10 回で一度も落ちなかった。ACTION_REQUIRED 1・2、L6、L3 のテストは直す前のコードで赤になる。Run 1 で生き残った M3 と P5 は、cycle 2 で足したテストで死ぬ(新しいロックのテストは M1〜M3 を 10/10 ずつ赤にした)。変異 17 個はすべて死んだ。本物の herdr には触れていない。cycle 2 の変更行の coverage は 12 statement 中 12
- Fail: 0
- Blocked: なし。Test gaps の 1(新しいテストの待ちに期限が無いこと)を tech-debt の行に足すかは orchestrator が決める

## Run 1(cycle 1)

- Date: 2026-11-06(run-test.sh は 10:39:05 から 11:07:43 JST)
- Plan: `docs/plans/active/2026-11-06-org-herdr-single-tab.md`(`## Test plan`、AC1〜AC12)。spec は `docs/specs/2026-11-06-org-herdr-autostart-panes.md` の B
- Tester: tester サブエージェント(`/test`、pipeline の cycle 1)
- Scope: `git diff 920832dd...HEAD`(HEAD feb2e943)。`./scripts/run-test.sh` は changed scope のまま打った。`.harness/state/verify-scope` は `scope=changed`、`reason=changed_languages`、`languages=golang`。Go の全パッケージ(10)と、local verifier(`scripts/verify.local.sh`)が回す `tests/test-*.sh` の 45 本が走った。worktree のファイルは変えていない。変異と gap を確かめるテストは `/tmp/rt-single-tab/` に置き、`go test -overlay` で打った
- Evidence: `docs/evidence/test-2026-11-06-org-herdr-single-tab.log`(`.gitignore:61` の `docs/evidence/*.log` に当たるのでコミットしない)。run-test.sh の全出力の後ろに、load と herdr のプロセスの記録、変異の定義と結果、gap を確かめるテストの本文と結果、L2 の probe、coverage を足した。wrapper 自身のログは `docs/evidence/verify-2026-11-06-013905.log`(同じく対象外。名前の時刻は UTC)。実機の確かめ(AC11)は `docs/evidence/org-herdr-single-tab-2026-11-06.md`

## Test execution

| Suite / Command | Tests | Passed | Failed | Skipped | Duration |
| --- | --- | --- | --- | --- | --- |
| `./scripts/run-test.sh`(changed scope、rc 0) | shell 45 本 + Go 10 パッケージ | shell 45 本 OK(PASS を含む行 5,195)、Go 10 パッケージ ok | 0 | 2 行(下の注) | 1,718 秒。1 分の load は開始時 4.27、終了時 27.99、記録の最大は 10:54:39 の 155.43 |
| └ Go: `internal/org` | — | ok | 0 | 0 | 1,100.7 秒 |
| └ Go: `internal/cli` | — | ok | 0 | 0 | 823.6 秒 |
| └ Go: `internal/config`、`internal/upgrade`、`internal/scaffold`、`internal/gitroot` | — | ok | 0 | 0 | 16.6 秒、4.3 秒、4.1 秒、3.8 秒 |
| └ Go: `internal/org/driver`、`internal/insights`、`internal/org/protocol`、`cmd/ralph` | — | ok | 0 | 0 | 3.6 秒、2.6 秒、2.5 秒、0.8 秒 |
| 配置のロックの並行のテスト、HEAD で `-count=20`(load 5 前後) | 1 本(subtest 2) | 20/20 | 0 | 0 | 3.8 秒 |
| 同じテストと `LockHeldPastTheDeadline`、HEAD で `-count=50`(load 15〜44、org 全体の coverage の回と並行) | 2 本 | 50/50 ずつ | 0 | 0 | 1 回 0.32 秒以下 |
| `go test -race ./internal/org -run '^TestOrgSpawn_Layout_' -count=5` | Layout の 13 本 | ok | 0 | 0 | 5.5 秒 |
| 配置のロックの変異 3 個(M1〜M3、下の節) | — | — | — | — | — |
| ほかの変異 32 個(下の「変異」) | 32 | 31 個が死んだ。P5 は org の全体でも生き残った | — | — | 1 個 1.3〜53 秒(P7 は止まって 604 秒)、全体の回は 694 秒 |
| gap を確かめるテスト 1 本(overlay)。HEAD と M1〜M3 で `-count=10` ずつ | 1 本 | HEAD 10/10 PASS。M1〜M3 では 10/10 FAIL(期待どおり) | — | — | 1 回 1 秒強(1 秒の観測を含む) |
| coverage の回(HEAD の `git archive` の写し、`./internal/org/...`、`-timeout 30m`) | 3 パッケージ | ok | 0 | 0 | 762 秒(org 761.9 秒) |

起動の仕方: run-test.sh は Bash tool の 600 秒の上限を超えるので、runner(`/tmp/rt-single-tab/runner.sh`)を Bash tool の `run_in_background` で起動した。runner は log と開始・終了・rc・load の meta を書いた。待つあいだは runner の pid を `caffeinate -w` で前景で待ち、meta を読んだ。`(cached)` のパッケージはなかった。Go の段の途中で、ほかのセッションの Xcode のビルドと自分の変異と coverage の回が重なり、1 分の load は 10:54 前後と 11:05 前後に 110〜155 まで上がった。それでも `internal/org` は 30 分の上限の 61% で終わった。

Skipped の 2 行は前の report と同じで、どちらもこの diff と関係のない手元の環境による。`tests/test-plan-visual.sh` の browser detection(`/Applications` にブラウザがある)と、`tests/test-secret-scan-branch.sh` の git 2.41 より古い本物の git の case(手元は git 2.49.0。stub の古い git の case は通った)である。shell の 45 本と PASS の行の数(5,195)も前の report と同じである。

## 本物の herdr に触れていないこと

- 新しいテストのうち本物の `driver.ExecRunner` を使うのは `internal/cli/org_herdr_stub_test.go` だけで、`setupOrgStubPATH` が一時ディレクトリの偽の `herdr` を PATH の先頭に置く。偽の `herdr` は shell の script で、本物の herdr を呼ぶ行はない(`exec` を grep して 0 件)。org と driver のテストは `fakeHerdr` か stub の runner を使う
- `internal/cli` の TestMain は `orgHerdrServerLaunch` を panic する関数に差し替えている(`internal/cli/main_test.go:193`)。テストが本物のサーバーを起動しようとすれば test binary が panic するが、`internal/cli` は ok だった
- 5 秒ごとの `pgrep -fl '^[^ ]*herdr server'` の記録(10:52:39〜11:16:30、275 回)に出たのは、ユーザーのサーバー(pid 22408、`/opt/homebrew/bin/herdr server`)だけだった。記録は run-test.sh の開始から 13 分遅れて始めたので、shell の段の前半は見ていない。上の 2 点のほうが主な根拠である

## 配置のロック(AC9、D3、self-review の L8)

`TestOrgSpawn_Layout_ConcurrentSpawnsShareOneTab` の「after the leader」は、implementer の spawn を `PaneSplit` の中で止め、reviewer の `spawn_started` を待ってから 100 ms 眠り、止めた spawn を放す。ロックがあれば reviewer はいつ着いてもロックで待つので、HEAD では眠りの長さによらず通る。HEAD では 20/20、load 15〜44 の 50/50、`-race` の 5 回が通った。

眠りの長さが効くのは、ロックが無い版をこのテストが赤にできるかのほうである。次の 3 つの変異を `spawn.go` の写しで打った。

| 変異 | 変えたところ | 既存のテストの結果 |
| --- | --- | --- |
| M1 | ロックの名前に座席 id を足し、同じ org の座席どうしで排他しない(ロックを外したのと同じ) | 「after the leader」が load 5 前後で 20/20、load 14〜40 で 50/50 赤(2 席とも leader のペインを右に分けた)。「two first spawns」は 0/20 と 1/50 しか赤にならない |
| M2 | ロックを `spawn_step`(pane_created)を書く前に放す | 「after the leader」が 5/50 だけ赤 |
| M3 | ロックを `spawn_step` のあと、`PaneRename` の前に放す | 0/20。spawn まわりに絞った回でも、`internal/org` の全体(693.9 秒)でも生き残った |

ロックを丸ごと外す変異(M1)は、既存のテストで確実に赤になる。L8 の心配(遅い環境で偽の成功が出る)は、load 40 でも起きなかった。一方、D3 が書く「ペインにラベルを付けるまで持つ」の部分は、既存のテストではほぼ固定されていない(M2 は 1 割、M3 は 0)。M3 のもとでは、implementer の `pane rename` が失敗してそのペインを閉じる間に、reviewer が同じペインを分けうる。

worktree の外に書いた `TestGap_LayoutLockHeldUntilThePaneIsLabelled`(`/tmp/rt-single-tab/mut/zz_gap_layout_lock_test.go`。本文は evidence の log にもある)は、implementer の spawn を `PaneRename` の中で止め、その間に reviewer の spawn が配置の呼び出し(`workspace_get`、`pane_list`、`pane_split`、`tab_create`、`workspace_create`)を 1 秒のあいだ 1 つもしないことを確かめる。HEAD で 10/10 通り、M1・M2・M3 をそれぞれ 10/10 赤にした。正しいコードでは呼び出しが起きえないので、赤になる側だけが時刻に左右される作りである。足すかは orchestrator が決める(Test gaps の 1)。

## 変異

どの変異も production のファイルの写しを `go test -overlay` で差し替え、worktree のファイルは変えていない。perl で 1 か所だけ置き換え、置き換える文字列が 1 か所でなければ止めた。org の変異は spawn・stop・disband・director・confirm に絞った `-run`(遅い regate・cleanup・gate・wake のテストを `-skip`)で、driver の変異は driver の全体で打った。compile に失敗した変異はなかった。打つ前と後で `git status --porcelain --ignored`(`.harness/cache/` を除く)の違いは、run-test.sh の wrapper が書いたログ 1 つだけだった。

| 変異 | 変えたところ | 殺したテスト(代表) |
| --- | --- | --- |
| V1 | ラベルの無いペインでタブの確かめをしない | `TestConfirmSeatPane_TheRule` など 17 本 |
| V2 | ペインのラベルが座席 id と違っても受け入れる | `TheRule`、`StopDisbandAndOwnPaneClose` など 10 本 |
| V3、V4 | ワークスペースの確かめの失敗、ワークスペースが無いことを無視する | V3 は `TheRule` など 7 本、V4 は `TestOrgStop_PaneNotConfirmed_NoCtrlCNoClose` と `TestOrgSpawn_StaleSeat_PaneNotClosed_RefusesTheSpawn` |
| V5 | タブのラベルがどうでも受け入れる | `TheRule` など 10 本 |
| V6 | herdr がタブを知らないときに受け入れる | `TestOrgSpawn_StaleSeat_PaneNotClosed_RefusesTheSpawn` と `TestOrgStop_PaneNotConfirmed_NoCtrlCNoClose` |
| V7 | 補償で、記録したタブと違うタブのラベルの無いペインも受け入れる(AC7) | `TheRule`、`TestCompensateStale_ClosesOnlyThePaneItsSpawnMade` |
| S1〜S3 | 起点の候補から「herdr に無いペイン」「自分」「止まった座席」を外さない | S1 は `TestOrgSpawn_Layout_SplitParent` など 9 本、S2 と S3 は `SplitParent` だけ |
| S4〜S7 | 候補 1 つで下に分ける、leader を選ぶ、古い spawn を選ぶ、最初の候補を選ぶ | S4 は `LeaderImplementerReviewer` など 5 本、S5 は 4 本、S6 と S7 は `ReviewerBeforeImplementer` と `SplitParent` |
| R1 | ラベルが org_id でないワークスペースを使う(AC4) | `TestOrgSpawn_Layout_RecordedWorkspaceNotUsable` |
| P1、P2 | `tab rename` の失敗で spawn を失敗にする、タブのラベルを座席 id にする | `TabRenameFailsSpawnGoesOn`、`LeaderImplementerReviewer` |
| P3 | 分ける向きを逆にする | `LeaderImplementerReviewer` など 3 本 |
| P4 | `spawn_step` にタブを記録しない(D5) | `LeaderImplementerReviewer` など 4 本 |
| P5 | `workspace create` の返事にタブが無いことを見ない(ルートペインだけ見る) | **生き残った**。絞った回でも `internal/org` の全体(693.9 秒)でも死なない(Test gaps の 2) |
| P6、P7 | ペインのラベルを org_id にする、ワークスペースのラベルを座席 id にする | P6 は 82 本、P7 は 13 本。P7 では並行のテストが 10 分の上限まで止まった(Test gaps の 3) |
| D1〜D5 | director のラベル済みの印、補償でペインを残す、`tab rename` を打たない、ペインのラベルを取り違える、返事にタブが無いことを見ない | `TestStartDirector_*` の 1〜19 本ずつ(D5 は `AWorkspaceReplyWithNoRootPaneOrTabFails`) |
| DR1〜DR5 | `pane split` の `--env` の確かめを `tab create` と共有する、別のサブコマンドのヘルプを読む、返事の `tab_id` を見ない、`--ratio` を変える、`label` の JSON の名前を変える | `TestHerdr_EnvCheck_EachSubcommandChecksItsOwnHelp`、`TestHerdr_PaneSplit_*`、`TestHerdr_PaneGet_ReadsTheLabel` ほか |

## Coverage

- パッケージ全体(HEAD の写し、`-count=1 -timeout 30m`): `internal/org` 95.0%、`internal/org/driver` 97.1%、`internal/org/protocol` 100.0%。最初の回は `-timeout` を付け忘れて Go の既定の 10 分で止まった(10:45〜10:55、1 分の load が 155 まで上がった間。そのとき走っていたのは `TestRegateCI_GateOnlyRecordsNothingWhileTheChecksAreNotDone`)。自分の回の設定の誤りで、テストの失敗ではない
- 変更した行(テストを除く `.go`、`git diff -U0 920832dd..HEAD` の足された行を含む profile のブロック): 268 statement のうち 257(95.9%)。`director_start.go` 25/25、`director_stop.go` 3/3、`director_wake.go` 1/1、`lockfile.go` 1/1、`verbs.go` 27/27、`spawn.go` 112/117、`driver/herdr.go` 88/94
- 打たれていない 11 statement: driver の JSON の `parse result` の失敗 5 つ(`herdr.go:325`、`:367`、`:427`、`:471`、`:827`。外側の封筒は読めて中身の形が違う場合)、`envProbe` の知らないサブコマンドの戻り(`:112`、呼ぶ側は 3 つの定数だけを渡す)、`placeSeatPane` のロックの中の manifest の読み取りの失敗(`spawn.go:2544`)、`org_workspace_created` と `spawn_step` の追記の失敗(`:2565`、`:2601`)、`resolveWorkspace` の `org_workspace_closed` の追記の失敗(`:2641`)
- `internal/cli` の変更は `--help` の文と director start の出力の 1 行で、どちらもテストが文字列で固定している(`org_director_headless_test.go:191`、`org_director_test.go:1036`)。cli の coverage は測っていない
- Branch: 計測なし(Go の道具は statement だけ)

## Failure analysis

| Test | Error | Root cause | Proposed fix |
| --- | --- | --- | --- |
| (なし) | — | — | — |

run-test.sh、HEAD での繰り返しの回、`-race` の回、coverage の回(`-timeout 30m` の 2 回目)、gap を確かめるテストの HEAD の回のどれにも失敗はなかった。flaky として memory にあるテスト(`TestRunWatcher_TimeoutIndependentOfSmallInterval`、`TestOrgSpawn_HelloPastTimeoutMS_PrintsTheDeadlineHint` など)も、load 155 の山のなかで落ちなかった。

## Test plan の項目とテストの対応

テストはどれも run-test.sh の中で通った。

| Test plan の項目 | テスト |
| --- | --- |
| driver の新しいメソッドの argv と返事(`pane split`、`workspace create` の `root_pane` と `tab`、`pane get` の `label`、`pane list`) | `internal/org/driver/herdr_test.go` の `TestHerdr_PaneSplit_ArgvAndReply`、`_RejectsOtherDirections`、`_UnreadableReplyIsAnError`、`TestHerdr_WorkspaceCreate_PartialEnvelopeKeepsTheWorkspace`、`TestHerdr_PaneRename_TabRename`、`TestHerdr_PaneList`、`TestHerdr_PaneGet_ReadsTheLabel` ほか |
| サブコマンドごとの `--env` の確かめ(AC10) | `TestHerdr_EnvCheck_WithoutEnvFlagWarnsOncePerSubcommand`、`_EachSubcommandChecksItsOwnHelp`、`_LiteralChecksEveryCallOfEachSubcommand`、`TestHerdr_WorkspaceCreate_*Env*`、`TestHerdr_PaneSplit_PaneEnvGoesBeforeNoFocus`。CLI の偽物は `TestOrgHerdrStub_HelpsListEnvUnlessNoenv` |
| D4 の規則の表(受け入れる 3 つと断る場合、AC5、AC6) | `TestConfirmSeatPane_TheRule`(12 行と、herdr がペインを知らない subtest)、`TestConfirmSeatPane_StopDisbandAndOwnPaneClose`(5 つの状態 × stop・disband・自分のペインを閉じる) |
| D1 の起点の選び方(候補 0〜3、leader だけ、herdr に無いペイン) | `TestOrgSpawn_Layout_SplitParent` |
| AC1、AC2 の呼び出しの順と台帳 | `TestOrgSpawn_Layout_LeaderImplementerReviewer`、`_ReviewerBeforeImplementer`。CLI は `TestOrgSpawn_HappyPath_EventSequenceReceiptAndWorkspaceReuse` |
| AC3(implementer だけを閉じる) | `TestOrgSpawn_Layout_StopImplementerOfThree` |
| AC4(使えないワークスペースの記録と作り直し) | `TestOrgSpawn_Layout_RecordedWorkspaceNotUsable` |
| AC7(名前を付ける前に止まった spawn の補償、4 つの場合と別のタブ) | `TestOrgSpawn_Layout_InterruptedAfterPaneCreated`、`TestCompensateStale_PaneCreatedDetails`、`TestCompensateStale_ClosesOnlyThePaneItsSpawnMade` |
| AC8(director のルートペイン、stop、前の版の director) | `internal/org/director_start_root_pane_test.go` の 4 本、`TestStopDirector_ConfirmsThePaneByItsLabel`、書き換えた `director_start_test.go` と `director_stop_test.go` |
| AC9(同時の spawn、ロック) | `TestOrgSpawn_Layout_ConcurrentSpawnsShareOneTab`、`_LockHeldPastTheDeadline`(上の節。「ラベルまで持つ」は固定が弱い) |
| ルートペインの `pane rename` の失敗 | `TestOrgSpawn_Layout_RootPaneRenameFails` |
| `tab rename` の失敗(spawn は続く) | `TestOrgSpawn_Layout_TabRenameFailsSpawnGoesOn` |
| 起点のペインが分ける直前に閉じた(`pane_not_found`)、ペインを作る前の失敗 | `TestOrgSpawn_Layout_FailuresBeforeThePane`(4 行) |
| 前の版のワークスペースで座席がすべて止まっている(`tab create`) | `TestOrgSpawn_Layout_NoLiveSeatPaneMakesATab` |
| `--dry-run` は herdr を呼ばない | `TestOrgSpawn_Layout_DryRunAsksHerdrNothing` |
| テストの偽物が herdr 0.7.5 の返事をすること | `TestSpawnFakeHerdr_ModelsPanesOfOneTab`、`internal/cli/org_herdr_stub_test.go` の 6 本 |
| 既存の org と CLI のテストが、書き換えた期待で通る(Regression) | run-test.sh の `internal/org`(1,100.7 秒)と `internal/cli`(823.6 秒)が ok |
| AC11(実機の配置、停止、全席の停止のあとの spawn) | 実機だけ。`docs/evidence/org-herdr-single-tab-2026-11-06.md`。この /test では打ち直していない(本物の herdr に触れないため) |
| AC12(文書と 2 つの sync) | /verify の担当。/test では見ていない |

## Regression checks

| Previously broken behavior | Status | Evidence |
| --- | --- | --- |
| 事案: org のワークスペースが座席ごとのタブに分かれていた(「1、leader、implementer、reviewer」のタブ) | OK | `LeaderImplementerReviewer` と `ReviewerBeforeImplementer` が `tab create` の 0 回を確かめて通った。分ける向きを逆にする P3 と、タブのラベルを変える P2 はこれらで死んだ |
| 既存の org と CLI のテスト(タブを前提にした期待を書き換えたもの) | OK | run-test.sh の `internal/org` と `internal/cli` が ok |
| shell のテストへの影響 | OK | 45 本すべて OK、PASS の行は前の report と同じ 5,195 |

## Test gaps

1. 配置のロックを「ペインにラベルを付けるまで」持つこと(D3)が、既存のテストではほぼ固定されていない。ロックを `spawn_step` の前に放す M2 は 5/50、`PaneRename` の前に放す M3 は 0/20 で、M3 は `internal/org` の全体でも生き残った。worktree の外に書いた `TestGap_LayoutLockHeldUntilThePaneIsLabelled` は HEAD で 10/10 通り、M1〜M3 を 10/10 ずつ殺した。足すかは orchestrator が決める
2. `placeSeatPane` の「`workspace create` の返事にタブが無い」の判定を固定するテストがない。P5(`ws.TabID == ""` を見ない)は `internal/org` の全体でも生き残った。`noRootPaneHerdr` はルートペインとタブを両方落とすので、ルートペインだけがある返事の行が無い。director の側の同じ判定(D5)は `TestStartDirector_AWorkspaceReplyWithNoRootPaneOrTabFails` が殺した。driver はタブの無い返事をそのまま返す(`TestHerdr_WorkspaceCreate_PartialEnvelopeKeepsTheWorkspace`)ので、`FailuresBeforeThePane` に 1 行足せば固定できる
3. 並行のテスト(「after the leader」)は、1 つ目の spawn が `PaneSplit` に届かない回帰で赤にならず、`<-h.entered` で止まる。P7 では Go の既定の 10 分まで止まり、run-test.sh の 30 分の上限ではパッケージ全体がそこまで止まる。`select` に `time.After` を足して早く落とすほうが、失敗の場所がすぐわかる
4. self-review の L2(座席 id が org_id と同じ座席)を確かめるテストがない。overlay の probe(本文は evidence の log)で、`confirmSeatPane("org-a", "org-a", ...)` がラベルの無い、タブのラベルが `org-a` のペインを受け入れる(err が nil)ことを確かめた。同じペインで座席 id が `seat-1` なら断る。self-review が書いたとおりの振る舞いで、L2 を直すときの行は `TestConfirmSeatPane_TheRule` に足せる
5. 「two first spawns」は、ロックが無くても通る(M1 のもとで 0/20、1/50)。偽物が速く、2 つの spawn がほとんど重ならないためである。新しい org で 1 つのワークスペースだけを作ることは、gap 1 のテストでも固定されない
6. 変更した行のうち 11 statement が打たれていない(Coverage の節)。ロックの中の追記の失敗と、driver の中身の形が違う返事で、どれも失敗を返すだけの行である
7. 実機の確かめ(AC11)は、`docs/evidence/org-herdr-single-tab-2026-11-06.md` の記録だけが見ている。同時の spawn、前の版の org への spawn、`--env` の無い古い herdr は、偽物でしか確かめていない(verify の report の Partially verified と同じ)
8. `internal/cli` の coverage は測っていない。変更は文字列で、テストが固定している

## Verdict

- Pass: `./scripts/run-test.sh`(rc 0。shell 45 本、Go 10 パッケージ、`(cached)` なし)。計画の Test plan の項目には、実機だけの AC11 と /verify の AC12 を除き、どれも run-test.sh の中で通ったテストがある(上の対応表)。配置のロックの並行のテストは HEAD で 20/20、load 15〜44 で 50/50 通り、ロックを外す変異(M1)を 20/20 と 50/50 で赤にした。本物の herdr には触れていない。変更行の coverage は 268 statement 中 257(95.9%)。ほかの変異 32 個のうち 31 個は既存のテストが殺した
- Fail: 0
- Blocked: なし。gap 1(ロックを持つ範囲)と gap 2(P5)のテストを足すか、gap 3 の待ちに期限を付けるかは orchestrator が決める
