# Test report: org-seat-resume

- Date: 2026-11-07(run-test.sh は 13:32:53 から 13:54:10 JST)
- Plan: `docs/plans/active/2026-11-07-org-seat-resume.md`(`## Test plan`、AC-1〜AC-15。D8 と仕様の FR-5a は cross-review cycle 2 の直しで改訂し、digest 104ee2d3ca75 で承認し直してある)
- Tester: tester サブエージェント(`/test`、pipeline の cycle 3)
- Cycle: 3。この report は cycle 2 の report(`7f1e823e`)を上書きした。cycle 1 と cycle 2 の結果は、それぞれ `3148561b` と `7f1e823e` の版で読める。cycle 3 では、cross-review cycle 2 の ACTION_REQUIRED 3 件の直し(R5 `323e94d5`、R6 `6993faca`)と計画の改訂(`636ca9cf`)を経た HEAD `0cbce5fb` をテストした
- Scope: `git diff 11239c8d...HEAD`(95 ファイル)。`./scripts/run-test.sh` は changed scope で起動したが、cycle 1、2 と同じく `internal/org/testdata/resume/*.txt` がどの言語にも分類されないため full に落ちた(`Language scope: full fallback (unclassified:internal/org/testdata/resume/claude-continuing-automatically.txt)`)。worktree の production code とテストには手を入れていない。変異は `git archive HEAD` で作った写し(`/tmp/osr-c3/mut`、`mut2`、`mut3`)に当てた
- Evidence: `docs/evidence/test-2026-11-07-org-seat-resume.log`(`.gitignore` の `docs/evidence/*.log` に当たるのでコミットしない。cycle 2 のものを上書きした)。run-test.sh の全出力の後ろに、meta、絞った回の meta と結果行、直しを固定するテストの PASS 行、cached を打ち直した回、coverage、変異の harness・定義・結果・落ちたテスト、生き残った変異のパッケージ全体での打ち直し、gap の probe とその結果を足した。wrapper 自身のログは `docs/evidence/verify-2026-11-07-043253.log`(名前の時刻は UTC)

## Test execution

| Suite / Command | Tests | Passed | Failed | Skipped | Duration |
| --- | --- | --- | --- | --- | --- |
| `./scripts/run-test.sh`(rc 0) | shell 45 本 + Go 10 パッケージ | shell 45 本 OK(PASS を含む行 5,195、cycle 2 と同じ)、Go 10 パッケージ ok | 0 | 2 行 | 1,277 秒。1 分の load は開始時 37.82、終了時 5.75 |
| └ Go: `internal/org` | 上位 1,587 本 | ok | 0 | — | 691.9 秒 |
| └ Go: `internal/cli` | 上位 1,076 本 | ok | 0 | — | 548.5 秒 |
| └ Go: `internal/config`、`internal/insights`、`cmd/ralph` | 上位 114、41、2 本 | ok | 0 | — | 11.9、1.4、0.9 秒 |
| └ Go: `internal/gitroot`、`internal/org/driver`、`internal/org/protocol`、`internal/scaffold`、`internal/upgrade` | 上位 6、102、15、32、114 本 | ok(5 つとも `(cached)`) | 0 | — | — |
| 上の `(cached)` の 5 つを `-count=1` で打ち直した | 同上 | ok | 0 | 0 | 2.5、1.8、1.6、1.0、2.4 秒 |
| `go test ./internal/org/ -run 'Resume\|Send\|PaneLock\|Wake\|Approval\|Approve\|ClaudeRetry\|RenderRolePrompt\|Skill' -count=1 -v` | 上位 379 本、subtest 込み 1,082 | 1,082 | 0 | 0 | 159.3 秒 |
| `go test ./internal/org/driver/ -count=1 -v` | 上位 102 本、subtest 込み 331 | 331 | 0 | 0 | 1.2 秒 |
| `go test ./internal/config/ -count=1 -v` | 上位 114 本、subtest 込み 751 | 751 | 0 | 0 | 14.7 秒 |
| `go test ./internal/cli/ -run 'Resume\|Send\|Wake\|Approve\|Org' -count=1 -v` | 上位 554 本、subtest 込み 1,348(+SKIP 1) | 1,348 | 0 | 1 | 412.2 秒 |
| `go test -race ./internal/org/ -run 'Resume\|PaneLock\|Send\|Wake\|Approval' -count=3 -v` | 上位 172 本 × 3、subtest 込み 1,719 | 1,719 | 0 | 0 | 112.5 秒。`DATA RACE` の行は 0 |
| coverage の回(上の org の filter に `Director` を足し、`-coverprofile`) | — | ok | 0 | — | 191.7 秒 |
| 変異 21 個(下の節) | 21 | 17 個は既存のテストで死んだ | — | — | 1 個 13〜247 秒(2 個は 4 分の時間切れ込み) |

上位の本数は `go test -list '.*' ./...` で数えた。cycle 2 の report を書いた時点の HEAD(`8e91fa6f`)から増えたのは `internal/org` の 4 本(`TestOrgSend_ApprovalCheck_AgainUnderThePaneLock`、`TestOrgSend_ApprovalCheck_PlanChangedWhileSendWaitsForThePaneLock`、`TestWakeDirector_AWakeCmdOfADirectorInHerdrRunsUnderItsPaneLock`、`TestWakeDirector_AWakeCmdOfARegistrationWithNoHerdrPaneTakesNoPaneLock`)だけで、消えたテストは無い。絞った cli の回の上位の本数が cycle 2 の 552 本から 554 本になったのは、filter に `Wake|Approve` を足したためである。

`(cached)` の 5 つは、どれも `go list -deps` に `internal/org` を含まない(driver は `internal/org` の下のパッケージで、R5、R6 は driver に触れていない)。念のため `-count=1` で打ち直し、すべて ok だった。

起動の仕方: run-test.sh は Bash tool の 600 秒の上限を超えるので、runner(`/tmp/osr-c3/runner.sh`。開始・終了・rc を meta に書く)を perl の fork と setsid で切り離して起動した(INT、HUP、QUIT、TERM を既定に戻し、標準の入出力を `/dev/null` にした)。絞った 5 つの回、変異の harness 2 本、生き残った変異のパッケージ全体の回も同じ方法で切り離し、run-test.sh の shell の段と Go の段に重ねた。開始時の load 37.82 はほかの session の負荷で、その後は下がった。変異を 1 個打ち終えるたびに読んだ 1 分の load は、最大 19.38 だった。それでも run-test.sh と絞った回は落ちず、打ち直しが要ったテストも無かった。

Skipped の 2 行は cycle 1、2 と同じで、`tests/test-plan-visual.sh` の browser detection と、`tests/test-secret-scan-branch.sh` の git 2.41 より古い本物の git の case である。どちらもこの diff と関係のない手元の環境による。`internal/cli` の SKIP 1 本は `TestOrgRetiredStateDirEnvAbs_GetwdFailsOnLinuxOnceTheWorkingDirectoryIsRemoved`(darwin では `os.Getwd` が失敗しないため)で、これもこの diff と関係ない。

### 本物の herdr に触れていないこと

- R5 の新しいテストは偽物の herdr を使う。`underLockEditHerdr` と `planEditingHerdr` は `fakeHerdr` を包み、wake のテストの `checkHerdr` も偽物である。wake_cmd は一時ディレクトリの shell script で、herdr を呼ばない(テストは herdr の呼び出しが 0 回であることも見る)
- `internal/cli` の resume のテストは `stubResumeHerdr` を通す。`main_test.go` の TestMain は `orgHerdrServerLaunch` を panic する関数に差し替えている

## cross-review cycle 2 の直し(R5、R6)とそれを固定するテスト

どのテストも run-test.sh と絞った回で PASS した。変異の名前は下の「変異」の節の表のもの。

| 直し | 固定するテスト | 変異で確かめたこと |
| --- | --- | --- |
| #1(R5): `Send` は TASK の計画の承認を、ペインロックを持ったまま、ロックの中の `idle`/`done` の待ち直しのあと、打つ直前にもう一度確かめる | `TestOrgSend_ApprovalCheck_AgainUnderThePaneLock`(10 行。ロックの中の待ちのあいだに、計画の変更、Progress の行の追加、改訂の承認、変更なし、チェックの印だけ、の 5 つを typed と raw で起こす。見るのは、その待ちの時点でロックが持たれていたこと、断りの文、herdr の呼び出しが 2 回の待ちだけであること、manifest と receipts が待ちのあとのままであること、ロックが外れていること、変更なしと印だけでは送られること)、`TestOrgSend_ApprovalCheck_PlanChangedWhileSendWaitsForThePaneLock`(4 行。ほかの holder がロックを持つあいだに `Send` がロックを待ち、そのあいだに計画を変えると、外れたあとロックの中の待ちを経て断る。変えなければ送る) | A1(ロックの中の確かめを消す)、A3(ロックを取った直後、ロックの中の待ちの前に移す)、A4(条件を `!isTask` に)は死んだ。A3 は、2 本目のテストでも herdr の呼び出しが 1 回の待ちだけになって赤になった。A2(ロックの前の 2 回目の確かめを消す)は生き残った(Test gaps 1) |
| #2(R6): 止まった文は、エラーの印、空白、文の順で始まるまとまりだけに一致させる。codex の印から `•` を外した | fixture `claude-reply-error-paragraph.txt`、`claude-tool-output-error-paragraph.txt`(`TestClassifyResumeScreen_Fixtures`)、`TestClassifyResumeScreen_Edges` の「a stop sentence with no glyph before it is not a stop」(cycle 2 までは stopped を期待していた行で、R6 で期待を unknown に変えた)、「a stop glyph with no white space after it is not a stop」、「a codex item glyph before a usage limit is not a stop」、「a codex error glyph before a usage limit is stopped」、既存の「a codex warning glyph before a stop sentence is dropped」 | G1〜G8 は死んだ。G9(印のあとの空白を半角の空白だけにする)は生き残った(Test gaps 4) |
| #3(R5): herdr の headless の director に設定された `wake_cmd` も、同じペインロックを持って走らせる | `TestWakeDirector_AWakeCmdOfADirectorInHerdrRunsUnderItsPaneLock`(ロックを持たれているあいだは wake_cmd を走らせずに失敗し、banner は 1 行、`woken` は無く、直後の wake は held back。ロックが外れたあとは、wake_cmd が走っているあいだずっとロックを持ち、`{target}` に pane を渡し、終わると外す。herdr は呼ばない)、`TestWakeDirector_AWakeCmdOfARegistrationWithNoHerdrPaneTakesNoPaneLock`(3 行。昇格した session、launch_cmd が開いた端末の headless、pane の記録がない herdr の headless は、ロックのファイルを作らずに wake_cmd を 1 回走らせる) | W1、W2、W4、W5、W7、W8 は死んだ。W3(条件から `HerdrLaunch()` を外す)と W6(ロックを待つ長さを 0 にする)は生き残った(Test gaps 2、3) |

cycle 2 までの直しを固定するテスト(`TestOrgSend_AResumeStartsATurnWhileSendWaitsForThePaneLock_TypesNothing`、`TestOrgSend_IdleDoneWaitUnderThePaneLockFails_TypesNothing`、`TestWakeDirector_TakesThePaneLockOfTheDirectorAroundItsTyping`、`TestOrgSend_ApprovalCheck_AgainAfterTheWait`、`TestResumeInputHolds`、`TestOrgSendResume_PaneLockCannotBeOpened`、footer の `_Edges` の行)も、絞った org の回と run-test.sh で PASS した。

## 変異

`mutate.pl` が 1 個ずつ、元の文字列がファイルに 1 回だけ出ることと元と新が違うことを確かめてから置き換え、`go test ./internal/org/ -run <filter> -count=1 -timeout 4m` を打ち、写しの元のファイルに戻した。A3 は 2 か所を同時に置き換えた(ロックの中の確かめを止め、ロックを取った直後に確かめを足す)。最後に写しと元の写しを `diff -rq` で比べ、違いは無かった。変異を当てているあいだ、worktree の `git status --porcelain` は空だった。filter は `Send|Resume|PaneLock|Wake|Approval`(A)、それに `Director` を足したもの(W)、`Classify|Resume|Screen`(G)。

| 組 | 変異の数 | 結果 |
| --- | --- | --- |
| `verbs.go` の `Send` の承認の確かめ(A1〜A4): ロックの中の確かめを消す、ロックの前の 2 回目の確かめを消す、ロックの中の確かめをロックの中の待ちの前に移す、条件を `!isTask` にする | 4 | A1、A3、A4 は死んだ。A2 が生き残った |
| `director_wake.go` の wake_cmd のロック(W1〜W8): ロックの枝を通らなくする、条件から pane の有無を外す、条件から `HerdrLaunch()` を外す、ロックを走らせる前に外す、外さない、待ちを 0 にする、wake_cmd の失敗の文を typed の wake の文にする、持たれているときの枝を反転する | 8 | W1、W2、W4、W5、W7、W8 は死んだ。W3、W6 が生き残った |
| `resume_screens.go` の止まった文と印(G1〜G9): `!glyph` の確かめを消す、`glyph` を常に true にする、codex に `•` を戻す、codex から `■` を・`⚠` を消す、claude から `⎿` を・`⏺` を消す、`CutPrefix` を `block.text` でなく `head`(空)に当てる、印のあとの trim を半角の空白だけにする | 9 | G1〜G8 は死んだ。G9 が生き残った |

G1 が死んだのは「a stop glyph with no white space after it is not a stop」の 1 行だけだった。`head` は空で始まり、印で始まらないまとまりでは空のままなので、印のないまとまりは `!glyph` が無くても一致しない。`!glyph` の確かめが効くのは、印のあとに空白が無いまとまりだけである。

G7 と G8 は 4 分の時間切れで終わった。cycle 1、2 と同じく、resume が打たなくなると `TestOrgResume_TwoResumesOfOnePane_TheSecondReadsAgainAndTypesNothing` が `<-typing` で止まる。どちらもほかのテストの assert が先に 10 行の FAIL を出しているので、変異は死んでいる(`docs/tech-debt/README.md` の「Seven mutants」の行の (g) と同じ)。

### 生き残った 4 個

`-run` で絞った回で生き残った A2、W3、W6、G9 は、4 個を同時に当てて、パッケージ全体で打ち直した。`go test ./internal/org/ -count=1 -timeout 30m` は ok(587.7 秒)、`go test ./internal/cli/ -count=1 -timeout 30m` も ok(416.0 秒)だった。4 個は別々のファイルか別々の関数を変えるので、cycle 2 の F2 と F3 のように 1 つの式を二重に緩めることはない。

gap の probe: A2 と W3 には、HEAD で通り、その変異だけで赤になるテストを 1 本ずつ書き、写しに throwaway の `zz_gap_test.go` を置いて確かめた(コミットしていない。全文は evidence にある)。HEAD では 2 本とも PASS し、A2 では A2 の行だけが(herdr の呼び出しが `["agent_wait" "agent_wait"]`)、W3 では W3 の行だけが(`pane-w9_p9.lock` ができた)FAIL した。W6 は cycle 2 の W2 と同じ穴なので、cycle 2 の probe の形をそのまま使える。G9 は INFO なので probe を書いていない。

## Coverage

- Statement: 新しい `internal/org` の 3 ファイルは、coverage の回で 337 statement 中 334(99.1%。`pane_lock.go` 10/10、`resume.go` 164/167、`resume_screens.go` 160/160)。R5、R6 が足したり変えたりした production の行(`git diff -U0 8e91fa6f..HEAD` の行と profile の block を突き合わせた)にかかる block は 28 statement 中 27。打たれていない 1 つは `director_wake.go:611`(`lockDirectorWakePane` で、wake がロックのファイルを開けないときの文)で、cycle 2 の 608 行と同じ文が R5 で共通の関数に移ったものである
- Branch: 測っていない(Go の標準の道具は statement だけ)
- Function: R5、R6 が触った関数は、`runDirectorWakeCmdUnderPaneLock`、`directorWakeSender`、`findResumeStop` が 100%、`lockDirectorWakePane` 83.3%(上の 611 行)、`typeDirectorWakeLine` 95.0%、`Send` 97.5%、`checkSendApproval` 75.0%。`checkSendApproval` の打たれていない枝は manifest を読めないときのもので、org-plan-approval からある
- Notes: `resume.go` の打たれていない 3 statement は cycle 1、2 と同じ(director の対象が失敗したときの名前、`submit_unconfirmed=true` の detail、Enter の前の待ちの期限切れ)

## Failure analysis

| Test | Error | Root cause | Proposed fix |
| --- | --- | --- | --- |
| なし | — | run-test.sh、絞った 5 つの回、`-race` の回、cached を打ち直した回、coverage の回で、落ちたテストも、打ち直しが要ったテストも無かった。flaky と分類したものも無い | — |

## Regression checks

| Previously broken behavior | Status | Evidence |
| --- | --- | --- |
| cross-review cycle 2 の ACTION_REQUIRED 3 件 | 直っている | 上の「直しとそれを固定するテスト」の表。どの行にも PASS するテストがあり、主な変異は死んだ |
| cross-review cycle 1 の ACTION_REQUIRED 4 件と R3、R4 | 直ったまま | それを固定するテスト(上の表の下に挙げたもの)が run-test.sh と絞った回で PASS した。cycle 2 の変異で確かめた枝は R5、R6 で変わっていない(R5 の `typeDirectorWakeLine` の変更はロックの取り方を `lockDirectorWakePane` に寄せただけで、W8 は `TestWakeDirector_TakesThePaneLockOfTheDirectorAroundItsTyping` でも赤になった) |
| 既存のテストの変更 | 意図した期待の変更が 1 行だけ | `git diff 8e91fa6f..HEAD` のテストの変更は、新しい 2 ファイル、fixture 2 つ、`resume_screens_test.go` の 5 行の追加と 1 行の変更である。変更した 1 行は `_Edges` の codex の「印のない stop の文」で、期待を stopped から unknown にした。これは R6 が意図した挙動の変更で、計画の D8 の改訂と仕様の FR-5a に書かれている。resume が打たない側に倒れる変更で、assert を緩めたのではない。`verbs_test.go`、`approval_enforce_test.go`、`director_wake_test.go` は cycle 2 から変わっていない |
| S10 で見つけた、prompt の直下の上限の文(S11、`8e9504f6`) | 直ったまま | fixture `claude-weekly-limit-under-prompt.txt` の行が通る。`⎿` を印の表から消す G6 で、その行が赤になる |
| watch の stall と deadman、spawn と director の `--env` | 変わっていない | R5、R6 はこれらのファイルに触れていない。`internal/org` と `internal/cli` 全体が ok |

## Test gaps

1. A2: `Send` のロックの前の 2 回目の承認の確かめが固定されていない。消しても、断りはロックの中の確かめで同じ文になるので、`internal/org` と `internal/cli` 全体が通る。違うのは、変異の下では計画が変わった TASK もロックを待ち、ロックの中でもう 1 回待ってから断ることである。ほかの send か resume がロックを `--timeout-ms` いっぱいまで持っていれば、承認の断りではなく、ロックの待ちの時間切れの文で断る。`Send` の doc の「the check after the first wait refuses before Send waits for the lock」はこの振る舞いを書いている。最初の待ちのあいだに計画を変え、herdr の呼び出しが最初の待ちの 1 回だけで、ロックのファイルができないことを見る 1 本で死ぬ(probe で確かめた)。LOW
2. W3: wake_cmd がロックを取る条件のうち `reg.HerdrLaunch()` が固定されていない。今の `Attach` は昇格した session の attached に `pane_id` を書かないので、テストの昇格した session には PaneID が無い。古い binary が書いたか手で直した `director.jsonl` で、`pane_id` を持つ昇格した session があると、変異の下ではその wake_cmd がその pane のロックを取る。`pane_lock.go` の doc は「ほかの登録の wake_cmd はロックを取らない」と書いている。害は、ロックの取り合いで wake が失敗しうることだけである。`pane_id` を持つ attached を書いた昇格した session の 1 本で死ぬ(probe で確かめた)。INFO
3. W6: wake がロックを待つ長さ(`lockDirectorWakePane` の `driverCallTimeout`)が固定されていない。cycle 2 の W2(`docs/tech-debt/README.md` の 460 行、「Five test gaps from the cycle 2 /test」の (a))と同じ穴で、R5 で typed の wake と wake_cmd が `lockDirectorWakePane` を共有したため、待ちを 0 にすると両方の道で待たなくなる。どちらかの道で、ロックを 300ms だけ持ち、`DriverCallTimeout` 5 秒の wake が待ってから進むことを見る 1 本で、両方の道が固定される(cycle 2 の probe と同じ形)。LOW
4. G9: 印のあとの空白の判定(`unicode.IsSpace`)を半角の空白だけにしても通る。印のあとにタブや NBSP が来る画面は記録に無いので、今の画面では等価とみている。INFO
5. `director_wake.go:611`(wake がロックのファイルを開けないときの文)は打たれていない。cycle 2 の Test gaps 6 と tech-debt の 460 行の (e) と同じ行で、R5 で `typeDirectorWakeLine` から `lockDirectorWakePane` に移ったので、wake_cmd の道からも通る。LOW
6. tech-debt の記録との食い違い: 460 行の (c)「C25: the codex `•` in `StopGlyphs` is not pinned」は、R6 で `•` を印の表から外し、`_Edges` の「a codex item glyph before a usage limit is not a stop」がそれを固定した(`•` を戻す G3 が死んだ)ので、もう当たらない。(a) の W2 は wake_cmd の道にも広がり(上の 3)、(e) の行は `lockDirectorWakePane` に移った(上の 5)。直すかどうかは sync-docs が決める
7. cycle 1、2 の gap(`docs/tech-debt/README.md` の「Seven mutants」の行の (a)〜(g)、460 行の (b) F2・F3・F6 と (d) C14)はそのまま残っている。この cycle の G7、G8 も (g) の待ちで 4 分止まった
8. 実機の道は cycle 1、2 と同じで、R5、R6 のあとの binary は実機で動かしていない。ロックの中の承認の確かめ、wake_cmd とロックの取り合い、印の判定は、偽物の herdr と記録の画面のテストでしか確かめていない。/test はテストだけを打ち、実機には触れていない

## Verdict

- Pass: 8(8 回の実行がすべて通った。`./scripts/run-test.sh` は rc 0 で、shell 45 本と Go 10 パッケージが通った。`(cached)` だった 5 パッケージは `-count=1` で打ち直して ok。絞った 5 つの回(org、driver、config、cli、`-race` の org を 3 回)と coverage の回も 0 failed で、`DATA RACE` は 0。cross-review cycle 2 の ACTION_REQUIRED 3 件には、どれも PASS するテストがあり、名前は上の表にある。変異 21 個のうち 17 個は既存のテストが殺し、生き残った 4 個は `internal/org` と `internal/cli` の全体でも生き残った。どれも LOW か INFO の穴(Test gaps 1〜4)で、A2 と W3 はテストを 1 本ずつ足せば死ぬことを probe で確かめた。新しい org の 3 ファイルの coverage は 99.1%、R5、R6 の変えた行は 28 statement 中 27。本物の herdr には触れていない)
- Fail: 0
- Blocked: 0。Test gaps 1、3、5 のテストを足すか、tech-debt の行に足すかと、Test gaps 6 の食い違いの扱いは orchestrator が決める
