# Test report: org-seat-resume

- Date: 2026-11-07(run-test.sh は 11:16:28 から 11:36:50 JST)
- Plan: `docs/plans/active/2026-11-07-org-seat-resume.md`(`## Test plan`、AC-1〜AC-15。D8 は cross-review cycle 1 の直しで改訂済み)
- Tester: tester サブエージェント(`/test`、pipeline の cycle 2)
- Cycle: 2。この report は cycle 1 の report(`3148561b`)を上書きした。cycle 1 の結果はその commit で読める。cycle 2 は、cross-review cycle 1 の ACTION_REQUIRED 4 件の直し(R1 `0aa8e274`、R2 `306ee58f`)と、cycle 2 の self-review を受けた直し(R3 `3714f6d5`、R4 `6ee7c1dd`)のあとの HEAD `8e91fa6f` を打った
- Scope: `git diff 11239c8d...HEAD`(91 ファイル)。`./scripts/run-test.sh` は changed scope で打ったが、cycle 1 と同じく `internal/org/testdata/resume/*.txt` が言語に分類されないため full に落ちた(`Language scope: full fallback (unclassified:internal/org/testdata/resume/claude-continuing-automatically.txt)`)。production code もテストも worktree では変えていない。変異は `git archive HEAD` の写し(`/tmp/ralph-test-orgseat-c2/mut`、`mut2`、`mut3`)で当てた
- Evidence: `docs/evidence/test-2026-11-07-org-seat-resume.log`(`.gitignore` の `docs/evidence/*.log` に当たるのでコミットしない。cycle 1 のものを上書きした)。run-test.sh の全出力と meta、絞った回の meta と上位のテストの結果行、coverage、変異の harness・定義・結果、生き残った変異のパッケージ全体での打ち直し、gap の行の probe とその結果を入れた。wrapper 自身のログは `docs/evidence/verify-2026-11-07-021628.log`(名前の時刻は UTC)

## Test execution

| Suite / Command | Tests | Passed | Failed | Skipped | Duration |
| --- | --- | --- | --- | --- | --- |
| `./scripts/run-test.sh`(rc 0) | shell 45 本 + Go 10 パッケージ | shell 45 本 OK(PASS を含む行 5,195、cycle 1 と同じ)、Go 10 パッケージ ok | 0 | 2 行 | 1,222 秒。1 分の load は開始時 3.38、終了時 6.61 |
| └ Go: `internal/org` | 上位 1,583 本 | ok | 0 | — | 675.9 秒 |
| └ Go: `internal/cli` | 上位 1,076 本 | ok | 0 | — | 531.2 秒 |
| └ Go: `internal/config`、`internal/insights`、`cmd/ralph` | 上位 114、41、2 本 | ok | 0 | — | 13.2、1.5、0.8 秒 |
| └ Go: `internal/gitroot`、`internal/org/driver`、`internal/org/protocol`、`internal/scaffold`、`internal/upgrade` | 上位 6、102、15、32、114 本 | ok(5 つとも `(cached)`) | 0 | — | — |
| 上の `(cached)` の 5 つを `-count=1` で打ち直した | 同上 | ok | 0 | 0 | 2.6、1.2、0.6、1.6、1.4 秒 |
| `go test ./internal/org/ -run 'Resume\|Send\|PaneLock\|Wake\|ClaudeRetry\|RenderRolePrompt\|Skill' -count=1 -v` | 上位 345 本、subtest 込み 890 | 890 | 0 | 0 | 124.2 秒 |
| `go test ./internal/org/driver/ -count=1 -v` | 上位 102 本、subtest 込み 331 | 331 | 0 | 0 | 1.2 秒 |
| `go test ./internal/config/ -count=1 -v` | 上位 114 本、subtest 込み 751 | 751 | 0 | 0 | 9.4 秒 |
| `go test ./internal/cli/ -run 'Resume\|Send\|Org' -count=1 -v` | 上位 552 本、subtest 込み 1,345(+SKIP 1) | 1,345 | 0 | 1 | 424.1 秒 |
| `go test -race ./internal/org/ -run 'Resume\|PaneLock\|Send\|Wake' -count=3 -v` | 上位 160 本 × 3、subtest 込み 1,548 | 1,548 | 0 | 0 | 84.8 秒。`DATA RACE` の行は 0 |
| 変異 37 個(下の節) | 37 | 30 個は既存のテストで死んだ | — | — | 1 個 11〜74 秒(2 個は 4 分の時間切れ込み) |

上位の本数は `go test -list '.*' ./...` で数えた。cycle 1 の report の HEAD(`39db8f25`)からの増分は `internal/org` の 3 本(`TestOrgSend_AResumeStartsATurnWhileSendWaitsForThePaneLock_TypesNothing`、`TestOrgSend_IdleDoneWaitUnderThePaneLockFails_TypesNothing`、`TestWakeDirector_TakesThePaneLockOfTheDirectorAroundItsTyping`)と、`internal/cli` の 1 本(`TestOrgSend_HelpSaysItTakesThePaneLock`、`a42b9845`)で、消したものは無い。

`(cached)` の 5 つは、どれも `go list -deps` に `internal/org` を含まず、cycle 1 のあと変わったファイルに依らない。念のため `-count=1` で打ち直し、すべて ok だった。

起動の仕方: run-test.sh は Bash tool の 600 秒の上限を超えるので、runner(`/tmp/ralph-test-orgseat-c2/runner.sh`。開始・終了・rc を meta に書く)を perl の fork と setsid で切り離して起動した(INT、HUP、QUIT、TERM を既定に戻し、標準の入出力を `/dev/null` にした)。shell の段のあいだに driver・config・絞った org・`-race` の回を前景で打ち、cli の回と変異の harness を同じ方法で切り離して重ねた。変異の回の前半は run-test.sh の Go の段と重なり、変異の回を通した 1 分の load は最大 13.85 だった。それでも run-test.sh は落ちなかった。

Skipped の 2 行は cycle 1 と同じで、`tests/test-plan-visual.sh` の browser detection と、`tests/test-secret-scan-branch.sh` の git 2.41 より古い本物の git の case である。どちらもこの diff と関係のない手元の環境による。`internal/cli` の SKIP 1 本は `TestOrgRetiredStateDirEnvAbs_GetwdFailsOnLinuxOnceTheWorkingDirectoryIsRemoved`(darwin では `os.Getwd` が失敗しないため)で、これもこの diff と関係ない。

### 本物の herdr に触れていないこと

- `internal/org` の resume、send、wake のテストは偽物の herdr(`fakeResumeHerdr`、`fakeHerdr`、`checkHerdr`、`lockProbeHerdr`)を使う
- `internal/cli` の resume のテストは `stubResumeHerdr` を通す。`main_test.go` の TestMain は `orgHerdrServerLaunch` を panic する関数に差し替えている

## cross-review cycle 1 の直し(R1〜R4)とそれを固定するテスト

どのテストも run-test.sh と絞った回で PASS した。変異の名前は下の「変異」の節の表のもの。

| 直し | 固定するテスト | 変異で確かめたこと |
| --- | --- | --- |
| #1(R1): `Send` はロックを取ったあと残りの期限で `idle`/`done` をもう一度待ち、そうならなければ何も打たない | `TestOrgSend_AResumeStartsATurnWhileSendWaitsForThePaneLock_TypesNothing`(resume がロックを持って打つあいだに send がロックを待ち、resume の Enter で座席が `working` になると send は何も打たずに断り、座席が `idle` に戻ったあとの send は通る)、`TestOrgSend_IdleDoneWaitUnderThePaneLockFails_TypesNothing`(2 行。待ちの失敗と期限切れの文、2 回目の待ちの状態 `idle`/`done`、2 回目の待ちに渡す残りの期限、何も記録しないこと、ロックを外すこと)、3 回目の `AgentWait` を数えるように変えた既存の `TestOrgSend_Confirmed_ExactCallOrderAndUntilStates` ほか 8 本 | S6〜S14 の 9 個とも死んだ(2 回目の待ちを消す・結果を捨てる・状態を変える・期限を `timeoutMS` か 0 にする・ロックの前に移す、ほか) |
| #2(R2): 入力欄のすべての行が空のときだけ空とする | fixture `claude-input-draft-after-empty-line.txt`、`codex-input-draft-after-empty-line.txt`(`TestClassifyResumeScreen_Fixtures`)、`TestClassifyResumeScreen_Edges` の「codex draft after a blank line under an empty › line has text」「codex placeholder on the line under an empty › line is text」「codex footer right under the › line counts as input」「codex empty › line with nothing below it is empty」 | C12、C13、C15、C16、C17 は死んだ。C14 は生き残った(Test gaps 4) |
| #3(R1): director の push の wake も同じペインロックを取る | `TestWakeDirector_TakesThePaneLockOfTheDirectorAroundItsTyping`(ロックを持たれていると何も打たずに失敗し、外れたあとの wake は send-text と send-keys のどちらでもロックを持ち、終わると外す) | W1、W3、W4 は死んだ。W2(待つ長さを 0 にする)は生き残った(Test gaps 1) |
| #4(R2): 止まった文は、まとまりの先頭の UI の印を 1 つ落とした先頭に来るときだけ一致とする | fixture `claude-reply-quotes-error.txt`、`_Edges` の 7 行(tool の結果の API エラー、claude と codex の返事が上限の文を引く、印のない stop の文、`⚠`、印は 1 つだけ落とす、ほかの driver の印は落とさない) | C18〜C20、C22〜C24、C26、C27 は死んだ。C25(codex の `•`)は生き残った(Test gaps 3)。C21 は等価(Test gaps 5) |
| R3: codex の footer は既知の形の行だけからなる並びに限る。`Send` のロックの中の待ちの期限切れに期限のヒントを付ける | fixture `codex-input-draft-no-footer.txt`、`_Edges` の 5 行(状態行だけ、hint の行だけ、`% left` の 2 つの形、下書きと footer が続いた並び、`press ? for shortcuts`)、`TestResumeInputHolds` の「codex draft below the › line with no footer under it」、`TestOrgSend_IdleDoneWaitUnderThePaneLockFails_TypesNothing/the_deadline_runs_out_in_the_wait` | F4、F7、F8、F9、S11〜S13 は死んだ |
| R4: footer の 3 つの形はそれぞれ行全体に一致させる | `_Edges` の「codex draft that contains the context used figure is input」「codex draft that ends in the context left figure is input」 | F1、F5 は死んだ。F2、F3、F6 は 1 つずつでは生き残った(Test gaps 2)。F2 と F3 を同時に当てると「contains the context used figure」の行で死ぬ |

verify の report の Coverage gaps にある「`resumeCodexFooterLine` の正規表現を Go で走らせていない」は、この /test で、footer の形を見る `_Edges` の 8 行と codex の fixture、下の gap の probe の 3 行を Go で走らせて埋めた。

## 変異

`mutate.pl` が 1 個ずつ、元の文字列がファイルに 1 回だけ出ることと元と新が違うことを確かめてから置き換え、`go test ./internal/org/ -run <filter> -count=1 -timeout 4m` を打ち、写しの元のファイルに戻した。S14 は 2 か所を同時に置き換えた(ロックの前に待ちを足し、ロックの中の待ちを消す)。最後に写しと元の写しを `diff -rq` で比べ、違いは無かった。変異を当てているあいだ、worktree の `git status --porcelain` は空だった。filter は `Send|Resume|PaneLock|Wake`(S)、`Wake|Director|PaneLock`(W)、`Resume`(C、F)。

| 組 | 変異の数 | 結果 |
| --- | --- | --- |
| `verbs.go` の `Send` のロックの中の待ち(S6〜S14): 待ちを消す、結果を捨てる、状態に `working` を足す、期限を `timeoutMS` に・0 に、期限切れの枝を `false` に・`true` に、ヒントを消す、ロックの前に移す | 9 | 9 個とも死んだ |
| `director_wake.go` の wake のロック(W1〜W4): 別の pane のロックを取る、待ちを 0 に、すぐ外す、持たれているときの文の枝を `false` に | 4 | W1、W3、W4 は死んだ。W2 が生き残った |
| `resume_screens.go` の入力欄と止まった文(C12〜C27): 入力欄のほかの行を見ない(全部と codex だけ)、placeholder の例外を claude にも、placeholder を認めない、footer が `›` の直下でも footer とする、末尾の空行を飛ばさない、先頭一致を部分一致に、印のあとの空白を落とさない、印を 2 つ以上落とす、prompt のまとまりの除外を消す、印の表から 1 つずつ消す(5 個)、codex に `⏺` を足す | 16 | 13 個が死んだ。C14、C21、C25 が生き残った |
| `resume_screens.go` の codex の footer の形(F1〜F9): 3 つの形の先頭と末尾の錨を 1 つずつ外す、状態行のほかの欄を求めない、`left` を外す、`· ? for shortcuts` の続きを外す、形の確かめを飛ばす、trim の前の行に当てる | 8 | 5 個が死んだ。F2、F3、F6 が生き残った |

C19 と C23 は 4 分の時間切れでも終わった。cycle 1 の C3、C11 と同じく、1 回目の resume が打たなくなると `TestOrgResume_TwoResumesOfOnePane_TheSecondReadsAgainAndTypesNothing` が `<-typing` で止まる。どちらもほかのテストの assert が先に 10 行の FAIL を出しているので、変異は死んでいる(`docs/tech-debt/README.md` の 455 行の (g) と同じ)。

### 生き残った 7 個

`-run` で絞った回で生き残った W2、C14、C21、C25、F2、F3、F6 は、パッケージ全体(`go test ./internal/org/ -count=1 -timeout 30m`)で打ち直した。

- 7 個を同時に当てた回は、`TestClassifyResumeScreen_Edges/codex_draft_that_contains_the_context_used_figure_is_input` の 1 行で FAIL した(595.3 秒)。F2 と F3 を同時に当てると、状態行の形が `^.*Context \d+% (?:used|left)` になって末尾も「ほかの欄」も求めなくなり、その行の下書きが footer と読まれる。1 つずつなら、F2 は ` · ` を、F3 は `$` を残すので、その行は赤にならない
- そこで F3 を外した 6 個(`mut2`)と F3 だけ(`mut3`)に分けて打ち直し、どちらも ok だった(747.8 秒、747.7 秒)。6 個の組はそれぞれ別の枝を変えるので、1 個ずつパッケージ全体で当てても通るとみている。1 個ずつのパッケージ全体の回は打っていない

gap の行の probe: 生き残った 6 個(C21 を除く)それぞれに、HEAD で通り、その変異だけで赤になるテストの行を書き、写しに throwaway の `zz_gap_test.go` を置いて確かめた(コミットしていない。全文は evidence にある)。HEAD では 6 行とも PASS し、変異ごとに当てると、W2、C14、C25、F2、F3、F6 はそれぞれ自分の行だけが FAIL した。C21 は等価なので行を書いていない。

## Coverage

- Statement: 新しい `internal/org` の 3 ファイル(`resume.go`、`resume_screens.go`、`pane_lock.go`)は、絞った org の回で 334 statement 中 331(99.1%。164/167、157/157、10/10)。R1〜R4 が足したり変えたりした production の行にかかる block は 43 statement 中 42 で、打たれていない 1 つは `director_wake.go:608`(wake がロックのファイルを開けないときの文)。`internal/cli/org_resume.go` は cycle 1 の 65/67(97.0%)から help の文字列しか変わっていないので測り直していない
- Branch: 測っていない(Go の標準の道具は statement だけ)
- Function: R1〜R4 が触った関数で 100% に届かないのは `typeDirectorWakeLine` 90.9%(上の 608 行と、R1 の前からある枝 1 つ)と `Send` 97.4%。`capMSToContext` 80.0% は R3 が doc comment を変えただけで、打たれていない枝は R1 の前からある
- Notes: `resume.go` の打たれていない 3 statement は cycle 1 と同じ(director の対象が失敗したときの名前、`submit_unconfirmed=true` の detail、Enter の前の待ちの期限切れ)。行番号だけがずれた(185、398、527 行)

## Failure analysis

| Test | Error | Root cause | Proposed fix |
| --- | --- | --- | --- |
| なし | — | run-test.sh、絞った 5 つの回、`-race` の回、cached を打ち直した回で、落ちたテストも、打ち直しが要ったテストも無かった | — |

## Regression checks

| Previously broken behavior | Status | Evidence |
| --- | --- | --- |
| cross-review cycle 1 の ACTION_REQUIRED 4 件 | 直っている | 上の「直しとそれを固定するテスト」の表。どの行にも PASS するテストがあり、主な変異は死んだ |
| `Send` の既存のテスト | 強めた方向にだけ変わった | `verbs_test.go` の変更 9 か所を読んだ。どれも `AgentWait` を 2 回から 3 回に数え直したか、2 回目の待ちの状態を新しく見るもので、`len(...) != 2` を `slices.Equal` にした行もある。assert を緩めたものは無い。`internal/org` 全体が ok |
| S10 で見つけた、prompt の直下の上限の文(S11、`8e9504f6`) | 直っている | fixture `claude-weekly-limit-under-prompt.txt` の行が通る。`⎿` を印の表から消す C22 で、その行が赤になる |
| watch の stall と deadman、spawn と director の `--env` | 変わっていない | R1〜R4 はこれらのファイルに触れていない。`internal/org` と `internal/cli` 全体が ok |

## Test gaps

1. W2: wake がロックを待つ長さ(`driverCallTimeout`)が固定されていない。待ちを 0 にしても `internal/org` 全体が通る。`TestWakeDirector_TakesThePaneLockOfTheDirectorAroundItsTyping` は 50ms の期限でロックを持ったままにするので、待つか待たないかを区別しない。ロックを 300ms だけ持ち、`DriverCallTimeout` 5 秒の wake が待ってから打つことを見る 1 本で死ぬ(probe で確かめた)。LOW
2. F2、F3、F6: R4 の「行全体に一致させる」のうち、状態行の形の末尾の `$`(F2)、状態行の形が求める「ほかの欄」(F3)、`% context left` の形の末尾の `$`(F6)は、1 つずつは固定されていない。R4 の行「contains the context used figure」は F2 と F3 を同時に外したときだけ赤になる。`›` の下の空行のあとの下書きが `note · Context 5% used before you go on`、`please keep Context 5% used`、`50% context left is plenty, go on` のような行だと、変異の下では footer と読まれて入力欄が空になり、resume が下書きの後ろに文を足しうる。この 3 つの下書きを `_Edges` に 1 行ずつ足せば、それぞれ自分の変異だけで死ぬ(probe で確かめた)。LOW
3. C25: codex の印の表の `•` が固定されていない。消しても `•` で始まる上限の文が「不明」になるだけで、打たない側に倒れる。`• You've hit your usage limit. ...` を stopped と見る 1 行で死ぬ。INFO
4. C14: placeholder の例外が codex に限ることが固定されていない。claude の入力欄が codex の placeholder の文(`Ask Codex to do anything`)そのものを持つと、変異の下では空と読まれる。その画面を「入力欄に文字がある」と見る 1 行で死ぬ。INFO
5. C21: `findResumeStop` の `block.prompt` の除外は、今の表では等価である。prompt のまとまりの text は印(`❯`、`›`)で始まり、印はどの driver の StopGlyphs にもなく、止まった文もそれで始まらない。守りの重ねとして残る。テストでは殺せない。INFO
6. `director_wake.go:608`(wake がロックのファイルを開けないときの文)は打たれていない。`Send` と resume には `TestOrgSendResume_PaneLockCannotBeOpened` があるが、wake には無い。LOW
7. cycle 1 の gap(`docs/tech-debt/README.md` の 455 行の (a)〜(g): R6、R21、L1、L5、C9、C10、V2、`<-typing` の期限のない待ち)はそのまま残っている。この cycle の C19、C23 も (g) の待ちで 4 分止まった
8. 実機の道は cycle 1 と同じ(Enter だけ、codex の座席、headless の director、leader とほかの座席の順、応答が返らない通信断)。R1〜R4 のあとの binary と、push の wake とロックの取り合いは、偽物の herdr のテストでしか確かめていない。AC-10 の「作業を最後まで進める」は週の上限のため実機で確かめていない。/test はテストだけを打ち、実機には触れていない

## Verdict

- Pass: 7(7 回の実行がすべて通った。`./scripts/run-test.sh` は rc 0 で、shell 45 本と Go 10 パッケージが通った。`(cached)` だった 5 パッケージは `-count=1` で打ち直して ok。絞った 5 つの回(org、driver、config、cli、`-race` の org を 3 回)も 0 failed で、`DATA RACE` は 0。cross-review cycle 1 の ACTION_REQUIRED 4 件と R3、R4 には、どれも PASS するテストがあり、名前は上の表にある。変異 37 個のうち 30 個は既存のテストが殺し、生き残った 7 個は LOW か INFO の穴か等価(Test gaps 1〜5)で、6 個はテストの行を 1 つずつ足せば死ぬことを probe で確かめた。新しい org の 3 ファイルの coverage は 99.1%。本物の herdr には触れていない)
- Fail: 0
- Blocked: 0。Test gaps 1、2、6 のテストを足すかと、tech-debt の行に足すかは orchestrator が決める
