# Test report: org-seat-resume

- Date: 2026-11-07(run-test.sh は 08:38:02 から 08:55:54 JST)
- Plan: `docs/plans/active/2026-11-07-org-seat-resume.md`(`## Test plan`、AC-1〜AC-15)
- Tester: tester サブエージェント(`/test`、pipeline の cycle 1)
- Scope: `git diff 11239c8d...HEAD`(HEAD 39db8f25、72 ファイル)。`./scripts/run-test.sh` は changed scope で打ったが、`internal/org/testdata/resume/*.txt` が言語に分類されないため full に落ちた(`.harness/state/verify-scope` は `scope=full`、`reason=unclassified:internal/org/testdata/resume/claude-continuing-automatically.txt`、`languages=golang`)。production code もテストも worktree では変えていない。変異は `git archive HEAD` の写し(`/tmp/ralph-test-orgseat/mut`、`mut2`)で当てた
- Evidence: `docs/evidence/test-2026-11-07-org-seat-resume.log`(`.gitignore:61` の `docs/evidence/*.log` に当たるのでコミットしない)。run-test.sh の全出力、runner と meta、絞った回の結果行、新しいファイルの coverage、変異の harness・定義・結果、生き残った変異のパッケージ全体での打ち直し、gap の行の overlay の結果を入れた。wrapper 自身のログは `docs/evidence/verify-2026-11-06-233803.log`(名前の時刻は UTC)

## Test execution

| Suite / Command | Tests | Passed | Failed | Skipped | Duration |
| --- | --- | --- | --- | --- | --- |
| `./scripts/run-test.sh`(rc 0) | shell 45 本 + Go 10 パッケージ | shell 45 本 OK(PASS を含む行 5,195)、Go 10 パッケージ ok(`(cached)` なし) | 0 | 2 行 | 1,072 秒。1 分の load は開始時 2.79、Go の段で最大 16.93 |
| └ Go: `internal/org` | 上位 1,580 本 | ok | 0 | — | 556.8 秒 |
| └ Go: `internal/cli` | 上位 1,075 本 | ok | 0 | — | 406.6 秒 |
| └ Go: `internal/config`、`internal/org/driver`、`internal/gitroot`、`internal/insights`、`internal/org/protocol`、`internal/upgrade`、`internal/scaffold`、`cmd/ralph` | 上位 114、102、6、41、15、114、32、2 本 | ok | 0 | — | 11.2、2.9、4.4、3.7、4.0、2.3、1.2、0.8 秒 |
| `go test ./internal/org/ -run 'Resume\|Send\|PaneLock\|ClaudeRetry\|RenderRolePrompt\|Skill' -count=1 -v` | 上位 303 本、subtest 込み 743 | 743 | 0 | 0 | 96.5 秒 |
| `go test ./internal/org/driver/ -count=1 -v` | 上位 102 本、subtest 込み 331 | 331 | 0 | 0 | 1.3 秒 |
| `go test ./internal/config/ -count=1 -v` | 上位 114 本、subtest 込み 751 | 751 | 0 | 0 | 10.6 秒 |
| `go test ./internal/cli/ -run 'Resume\|Org' -count=1 -v` | 上位 549 本、subtest 込み 1,343(+SKIP 1) | 1,343 | 0 | 1 | 363.3 秒 |
| `go test -race ./internal/org/ -run 'Resume\|PaneLock\|Send' -count=1 -v` | 上位 116 本、subtest 込み 367 | 367 | 0 | 0 | 11.0 秒。`DATA RACE` の行は 0 |
| 変異 58 個(下の節) | 58 | 51 個は既存のテストで死んだ | — | — | 1 個 1〜16 秒(2 個は 4 分の時間切れ込み) |

上位の本数は `go test -list '.*' ./...` で数えた。この branch が足したテスト関数は `internal/org` 45、`internal/cli` 12、`internal/org/driver` 6、`internal/config` 3 で、消したものは無い。

起動の仕方: run-test.sh は Bash tool の 600 秒の上限を超えるので、runner(`/tmp/ralph-test-orgseat/runner.sh`。開始・終了・rc を meta に書く)を Bash tool の `run_in_background` で起動した。shell の段のあいだに、driver・config・絞った org の回と、写しでの変異の回を重ねた。変異の R と C の組(下)は run-test.sh の Go の段と重なり、1 分の load は 16.93 まで上がった。それでも run-test.sh は落ちなかった。

Skipped の 2 行は前の cycle と同じで、`tests/test-plan-visual.sh` の browser detection と、`tests/test-secret-scan-branch.sh` の git 2.41 より古い本物の git の case である。どちらもこの diff と関係のない手元の環境による。`internal/cli` の SKIP 1 本は `TestOrgRetiredStateDirEnvAbs_GetwdFailsOnLinuxOnceTheWorkingDirectoryIsRemoved`(darwin では `os.Getwd` が失敗しないため)で、これもこの diff と関係ない。

### 本物の herdr に触れていないこと

- `internal/org` の resume のテストは偽物の `fakeResumeHerdr` を使う
- `internal/cli` の resume のテストは `stubResumeHerdr` を通す。これは PATH が見つける `herdr` がテストの temp dir の下にないとテストを落とす。`main_test.go` の TestMain は `orgHerdrServerLaunch` を panic する関数に差し替えている

## Test plan の項目とテストの対応

| Test plan の項目 | テスト(すべて run-test.sh と絞った回で PASS) | 変異で確かめたこと |
| --- | --- | --- |
| 単体: 画面の判定(記録の画面、折り返し、footer、状態行、入力欄の文字、待ち中の文、上限の文、`Press enter to continue`、一覧にない選択画面、空の画面) | `TestClassifyResumeScreen_Fixtures`(fixture 18 個)、`_Edges`(42 行)、`TestResumeInputHolds`(16 行)、`TestResumeScreenSentences_EveryEntryHasASource` | C1〜C8、C11 は死んだ。C9、C10 は生き残った(Test gaps 3、4) |
| 単体: 分から 3 つの変数への換算(0、1、30、境界) | `TestClaudeMaxRetriesForMinutes_FollowsTheMeasuredFormula`、`TestClaudeRetryEnv_EntriesForTheMinutes`、`_ReturnsANewSliceEachCall` | D1〜D4 は死んだ |
| 単体: 設定の負の値 | `TestLoad_OrgAPIWaitMinutes`(-1 と大きな負)、`TestLoad_LocalFileSetsAPIWaitMinutes`(4 行)、`TestDefault_OrgAPIWaitMinutesIs30`、`TestDefaultsLockStep/org`、`TestOrgTemplateDrift_EachKeyIsComparedOnBothSides` | G1、G2 は死んだ |
| 結合: 打つ・飛ばす・待ち中の座席が混ざる org、2 つの org、director、`--org-id` | `TestOrgResume_OrgIDFilter_TwoOrgsAndTheDirector`、`_DirectorsNotInHerdrOrWithoutAPane_Skipped`、`_SeatTargets_GroupsAndFields`、`internal/cli` の `TestOrgResume_PrintsALinePerTargetAndTheSummary`、`_OrgIDLimitsItToThatOrg` | R7、R8、L4 は死んだ。R6 は生き残った(Test gaps 1) |
| 結合: `--dry-run` | `TestOrgResume_DryRun_TypesAndRecordsNothing`、`internal/cli` の `TestOrgResume_DryRunTypesNothingAndSummarizes` | R2、L3 は死んだ |
| 結合: 自分の pane、`agent get` の失敗、送信の失敗のあとも続ける、順番 | `TestOrgResume_CallersOwnPane_Skipped`、`_AgentGetAndPaneReadFailures_Skipped`、`_OneTargetFails_TheNextIsStillResumed`、`_OrderIsSeatsThenLeadersThenDirectors` | R1、R5、R16 は死んだ |
| 結合: spawn の argv の `--env`(claude、codex、0) | `TestOrgSpawn_ClaudeRetryEnvFollowsEachSeatsDriver`(6 行)、`TestOrgSpawn_Layout_TabOfAClaudeSeatGetsTheClaudeRetryEnv`、`TestSeatPaneExtraEnv_ClaudeRetryEnvIsANewSliceEachCall`、`internal/cli` の `TestOrgSpawn_HappyPath_EventSequenceReceiptAndWorkspaceReuse`(期待値に 3 つの変数を足した) | E1、E2 は死んだ |
| 結合: director start と起動スクリプト | `TestDirectorStart_ClaudeRetryEnvFollowsTheDirectorsDriver`、`TestDirectorLaunchScript_ExportsTheClaudeRetryEnvAfterSeatPaneEnv`、`_TheAgentGetsTheClaudeRetryEnv`、`TestStartDirector_LaunchCmdScriptCarriesTheClaudeRetryEnvOfItsDriver`、`internal/cli` の `TestOrgDirectorStart_LaunchesRegistersAndStops`(driver ごとの期待値に変えた) | E3〜E8 は死んだ |
| 結合: CLI の出力と終了コード | `internal/cli` の `TestOrgResume_*` 12 本 | L2 は死んだ。L1、L5(台帳の読み書きの区別)は生き残った(Test gaps 2) |
| 回帰: `Send` の既存のテスト | `TestOrgSend*` 上位 42 本を含む `internal/org` 全体 | S1〜S5(S3 は compile できず S3b に置き換えた)は死んだ |
| 回帰: watch の stall の既存のテスト | `internal/org` 全体 | V1(`resumed` を状態のイベントにする)は死んだ。V2 は生き残った(Test gaps 5) |
| 回帰: 役割 prompt と skill の一致 | `TestRenderRolePrompt*` 上位 124 本、skill のテスト上位 68 本(新しい `TestRenderRolePrompt_*Resume*` 4 本と `TestDirectorSkill_Resume*` 3 本、`TestOrgSkill_ResumeRowAndTheClaudeRetryEnv`) | 変異は当てていない |
| 端: エラー文のあとに入力欄に文字がある、`done`、行数が少ない、2 回の resume、codex の上限の文 | fixture `claude-input-has-text.txt`、`TestOrgResume_StoppedCodexSeat_TypesTheSentence`(`done`)、`_Edges` の「nothing above the input box」「empty screen」、`TestOrgResume_TwoResumesOfOnePane_TheSecondReadsAgainAndTypesNothing`、fixture `codex-usage-limit.txt` | C1、R9 は死んだ |
| 並行(AC-14): ロックを持たれているあいだ打たない、ロックの中の読み直し、Enter の前の確かめ | `TestOrgResume_PaneLockHeld_SkipsAndTypesNothing`、`TestOrgSend_PaneLockHeld_RefusesWithNothingTyped`、`TestOrgResume_ScreenChangedUnderTheLock_TypesNothing`(4 行)、`_TypedTextNotInTheInputBox_NoEnter`(2 行)、`_TwoResumesOfOnePane_...`、`_SendWaitsForAResumeHoldingThePaneLock`、`TestOrgSend_HoldsThePaneLockWhileTypingAndReleasesIt`、`TestOrgSendResume_PaneLockCannotBeOpened`。`-race` で `DATA RACE` 0 | R3、R4、R13、R14、R17〜R20、S1〜S5 は死んだ。R21 は生き残った(Test gaps 1) |
| 実機(AC-10、AC-15) | /test では打っていない。S10 の記録(`docs/evidence/org-seat-resume-live-2026-11-07.md`)を verify が突き合わせた | — |

## 変異

`mutate.pl` が 1 個ずつ、元の文字列がファイルに 1 回だけ出ることを確かめてから置き換え、`go test ./<pkg>/ -run <filter> -count=1 -timeout 4m` を打ち、写しの元のファイルに戻した。最後に写しと元の写しを `diff -rq` で比べ、違いは無かった。変異を当てているあいだ、worktree の `git status --porcelain` は空だった。

| 組 | 変異の数 | 結果 |
| --- | --- | --- |
| `resume.go`(R1〜R21): 自分の pane、dry run、ロックの中の読み直し、順番、leader の見分け、`--org-id`、`idle`/`done`、待ち中、一覧にない画面、Enter の前の確かめ、Enter だけの道、director の記録、ロックの名前と外す位置、ロックの待ち | 21 | 19 個が死んだ。R6、R21 が生き残った |
| `verbs.go` の `Send` と `typeAndSubmit`(S1、S2、S3b、S4、S5): ロックの名前、外す位置 2 か所、待ちを 0 に、Enter の前の確かめを飛ばす | 5(S3 は `deadline` が使われず compile できなかったので、同じ意味の S3b に置き換えた) | 5 個とも死んだ |
| `resume_screens.go`(C1〜C11): 入力欄の文字、選択画面、状態行、stop の文を探す範囲、prompt の区切り、prompt のまとまりを飛ばす、codex の placeholder、罫線、Enter だけと待ち中の順、先頭 12 文字、空行での区切り | 11 | 9 個が死んだ。C9、C10 が生き残った |
| 環境の変数(E1〜E8、D1〜D4、G1、G2): codex にも足す、seat から外す、director の herdr と launch script で外す・driver を claude に固定する・順を変える、0 の扱い、式、ミリ秒、下限、負の値、既定値 | 14 | 14 個とも死んだ |
| イベント(V1、V2): `resumed` を状態のイベントにする、deadman の leader の活動に数える | 2 | V1 は死んだ。V2 が生き残った |
| CLI(L1〜L5): dry run を書き換える側にする、終了コード 1 をやめる、`DryRun` を落とす、`--org-id` を落とす、本番を読むだけの側にする | 5 | L2〜L4 は死んだ。L1、L5 が生き残った |

生き残った 7 個は、`-run` で絞った回のあと、パッケージ全体で打ち直した。`internal/org` は R6、R21、C9、C10、V2 を同時に当てて `go test ./internal/org/ -count=1` が ok(569.8 秒)、`internal/cli` は L1 と L5 を同時に当てて `go test ./internal/cli/ -count=1` が ok(395.4 秒)だった。1 個ずつ `-run` で絞った回でも生き残っていて、同時に当てた組はそれぞれ別の枝を変えるので、1 個ずつパッケージ全体で当てても通るとみている。1 個ずつのパッケージ全体の回は打っていない。

L1 と L5 には、テストの表に 1 行ずつ足す案を overlay で確かめた(コミットしていない)。`TestOrgLegacyLedger_MutatingVerbsRefusedWhileLegacySeatsActive` に `{"resume", ... "resume"}`、`TestOrgLegacyLedger_ReadOnlyVerbsNoteAndContinue` に `{"org_resume_dry_run", []string{"org", "resume", "--dry-run"}, false, stateDirHint}` を足すと、HEAD では両方通り、L1 は `.../org_resume_dry_run`、L5 は `.../resume` で赤になった。

C3 と C11 は 4 分の時間切れでも終わった。判定の変異で 1 回目の resume が打たなくなると、`TestOrgResume_TwoResumesOfOnePane_TheSecondReadsAgainAndTypesNothing` が `<-typing`(`pane_lock_test.go:373`)で止まり、`-timeout 4m` の panic まで待った。ほかのテストの assert が先に赤を出しているので、変異は死んでいる(Test gaps 6)。

## Coverage

- Statement: 新しい `internal/org` の 3 ファイル(`resume.go`、`resume_screens.go`、`pane_lock.go`)は、絞った org の回で 316 statement 中 313(99.1%)。`driver/paneenv.go` の関数は 5 つとも 100%(driver パッケージ全体は 97.2%)。`internal/cli/org_resume.go` は `TestOrgResume` の回に `-coverpkg=./internal/cli/,./internal/org/` を付けて 67 statement 中 65(97.0%)。`internal/config` パッケージ全体は 96.6%
- Branch: 測っていない(Go の標準の道具は statement だけ)
- Function: 新しい関数で 100% に届かないのは、`resumeTarget.label` 66.7%、`resumeOne` 98.3%、`resumeSubmit` 95.8%、`newOrgResumeCmd` 93.3%、`resumeTargetText` 75.0% の 5 つ
- Notes: 打たれていない 5 statement は次のとおり
  - `resume.go:180` の `label` の director の枝。director の対象が失敗したときの `Errs` の名前
  - `resume.go:391` の `submit_unconfirmed=true`。org の絞った回では打たれないが、`internal/cli` の `TestOrgResume_UnconfirmedSubmitNamesTheReadCommand` が manifest の Details まで固定している
  - `resume.go:520` の「the pause before Enter ended」。Enter の前の 750ms のあいだに期限が来た場合
  - `org_resume.go:74` の `newOrgRuntime` の失敗の return
  - `org_resume.go:201` の `resumeTargetText` の最後の `return t.Reason`。Line も Reason もない対象

## Failure analysis

| Test | Error | Root cause | Proposed fix |
| --- | --- | --- | --- |
| なし | — | run-test.sh、絞った 5 つの回、`-race` の回で、落ちたテストも、打ち直しが要ったテストも無かった | — |

## Regression checks

| Previously broken behavior | Status | Evidence |
| --- | --- | --- |
| `Send` の打ち方(`typeAndSubmit` に切り出した、計画の D2) | 変わっていない | `TestOrgSend*` 上位 42 本を含む `internal/org` 全体が ok。S1〜S5 は死んだ |
| S10 で見つけた、prompt の直下の上限の文(S11、`8e9504f6`) | 直っている | fixture `claude-weekly-limit-under-prompt.txt` の行が通る。prompt の区切りを外す C5 で、その行が赤になる |
| 既存の spawn と director の `--env` の期待値 | 強めた方向にだけ変わった | 既存のテスト 6 ファイルの変更を読んだ。3 つの変数を期待値に足したか、codex の director の期待値を分けた変更で、assert を緩めたものは無い。`testOrgConfig` の `api_wait_minutes` は 0 なので、既存の org のテストの argv は変わらない |
| watch の stall と deadman | 変わっていない | `internal/org` 全体が ok。`resumed` は状態のイベントではない(V1 で赤) |

## Test gaps

1. `resume.go` の 2 か所が固定されていない。どちらも LOW
   - R6: leader を座席 id で見分ける節(`t.id == LeaderIdentity`)。`TestOrgResume_SeatTargets_GroupsAndFields` のコメントは「by role or seat id」と書くが、id が `leader` の座席は役割も `leader` なので、節を消しても通る。役割が leader でない(か空の)id `leader` の座席を 1 行足せば固定できる
   - R21: `LockWait` が 0 のときの既定の 30 秒(`defaultResumePaneLockWait`)。CLI は `LockWait` を渡さないので、本物の `ralph org resume` はいつもこの 30 秒で待つ。テストはどれも `LockWait` を渡すか、ロックを取り合わない。`LockWait` なしの resume が、短く持たれたロックを待って打つことを見るテストで固定できる
2. resume の台帳の読み書きの区別(L1、L5)。`ralph org resume` は書き換える動詞として、古い worktree の台帳に動いている座席があると断り、`--dry-run` は読むだけの動詞として注記して続ける(`org_legacy_ledger.go` の列挙と verify の report のとおり)。どちらもテストが無く、2 つを入れ替えても `internal/cli` 全体が通った。上の「変異」の節の 2 行を足せば両方死ぬ。LOW
3. C9: 画面の判定で、Enter だけの文と待ち中の文を探す順を入れ替えても通る。両方の文が見える画面の fixture が無い。どちらを先にしても打たない側か Enter だけの側に倒れるので、危険は小さい。INFO
4. C10: `ResumeInputHolds` が先頭 12 文字だけを比べること。fixture の入力欄はどれも文を最後まで持つので、全文を比べても通る。入力欄に文の先頭だけが見える(残りが切れた)画面を 1 行足せば固定できる。LOW
5. V2: `resumed` を watch の deadman の leader の活動に数えないこと。`EventResumed` の doc comment と org runtime の仕様の改訂がそう書いているが、数えるように変えても `internal/org` 全体が通った。`leaderActivityEventCount` はこの branch で変わっておらず、もともと数えない。`resumed` の行を入れても数が変わらないことを見る 1 行で固定できる。LOW
6. `TestOrgResume_TwoResumesOfOnePane_TheSecondReadsAgainAndTypesNothing`(`pane_lock_test.go:373`)と `TestOrgResume_SendWaitsForAResumeHoldingThePaneLock`(同 408 行)は `<-typing` を期限なしで待つ。1 回目の resume が打たなくなる回帰(C3、C11)では、テストがパッケージの時間切れまで止まる。run-test.sh の `-timeout=30m` では `internal/org` 全体が 30 分止まる。前の cycle の `docs/tech-debt/README.md` の行(`TestOrgSpawn_Layout_ConcurrentSpawnsShareOneTab` の `<-h.entered`)と同じ形で、その triage は遅い CI での誤りを避けるため期限を足さないと決めている。この 2 本をその行に足すかは orchestrator が決める
7. 打たれていない枝(Coverage の Notes): director の失敗の名前、Enter の前の待ちの期限切れ、`newOrgRuntime` の失敗。どれも文を作るか、既存の道に合流するだけの枝である
8. 実機の道(verify の report の Coverage gaps と同じ): Enter だけ、codex の座席、headless の director、leader とほかの座席の順、応答が返らない通信断、S11 のあとの binary。AC-10 の「作業を最後まで進める」は週の上限のため実機で確かめていない。/test はテストだけを打ち、実機には触れていない

## Verdict

- Pass: 6(6 回の実行がすべて通った。`./scripts/run-test.sh` は rc 0 で、shell 45 本と Go 10 パッケージが通り、`(cached)` は無い。絞った 5 つの回(org、driver、config、cli、`-race` の org)も 0 failed で、`DATA RACE` は 0。計画の Test plan の項目には、実機の AC-10 と AC-15 を除き、どれも通ったテストがある(上の対応表)。変異 58 個のうち 51 個は既存のテストが殺し、生き残った 7 個はどれも LOW か INFO の穴(Test gaps 1〜5)で、うち L1 と L5 はテストの表に 1 行ずつ足せば死ぬことを overlay で確かめた。新しいファイルの coverage は org の 3 ファイルで 99.1%、CLI の 1 ファイルで 97.0%。本物の herdr には触れていない)
- Fail: 0
- Blocked: 0。Test gaps 1、2、4、5 のテストを足すかと、6 を tech-debt の行に足すかは orchestrator が決める
