# Test report: director-notice-handling

- Cycle: 3(HEAD `ad590839`。cycle 3 の verify のコミット `6d4b91c5` と計画の記録 `059ff4c1` のあとに、origin/main `7433ce84`(#331)を取り込んだ merge)。cycle 2 は HEAD `1d7eda3d`、cycle 1 は HEAD `fcd368b0` で回した。cycle 3 の結果は `## Cycle 3` の節にまとめ、`## Verdict` を cycle 3 の判定にした。cycle 2 と cycle 1 の判定は、その下の `### Verdict(cycle 2)` と `### Verdict(cycle 1)` に置いた。`## Test execution` から `## Test gaps` までの節は cycle 2 のときのままで、各節に cycle 2 の小節を先に、cycle 1 の小節をそのあとに置いてある。cycle 1 と cycle 2 の文は、cycle 2 の判定に見出しを足したほかは変えていない。下の Date から Evidence までの行のうち、「(cycle 3)」と「(cycle 2)」の印のないものは cycle 1 のもの
- Date(cycle 3): 2026-11-15。run-test.sh は 20:39:15 から 21:16:22 JST(2,227 秒)。変異 (a)〜(d) と追加の g3 は 20:41 から 20:53(run-test.sh の shell の段)に流した。`(cached)` の回し直し、名前を絞った 2 回、追加の g4 は、run-test.sh が終わったあとの 21:17 から 21:22 に流した
- Tester(cycle 3): tester サブエージェント(`/test`、パイプラインの cycle 3。main のチェックアウトの `.harness/state/standard-pipeline/cycle-count.json` は `"cycle":2,"cap":3` のままなので、insight event には `--cycle 3` を明示した)
- Scope(cycle 3): cycle 2 の `/test`(`7a70e773`)のあとに入ったのは、origin/main の取り込み 3 つ(`aeab4b80` が `bbe2807d` の #328、`1f713222` が `c53bf39c` の #330、`ad590839` が `7433ce84` の #331)、cycle 2 の sync-docs と triage(`9e0182f9`、`d00a2bfa`)、cycle 3 の直し(`356adb05`、`5d7e90af`、`ef90e11a`、`d365fa04`)、self-review の 2 回(`869d92ae`、`2abe22c2`)、verify(`6d4b91c5`)、計画の記録(`059ff4c1`)である。この branch が cycle 3 で変えたテストは `internal/org/prompts_notice_test.go` だけで(`1d7eda3d` から +127/-16。leader の本文の 3 文の断片、新しいテスト `TestOrgSkill_InboxResolvesTheHookQuestionAtOnce`、コメント)、`internal/cli/director_notice_skill_test.go` と、固定した断片を持つ `prompts_test.go`、`prompts_leader_plan_test.go`、`prompts_xreview_leader_test.go` は `1d7eda3d` から変わっていない(`git diff --stat` が空)。`git diff 7433ce84 HEAD` は 23 ファイル、+2404/-20 で、Go はテスト 2 ファイルだけである。木には main の側の変更も入っている。#330 は `internal/org` の画面の読み(`usage_limit_screen.go`、`resume_screens.go`、`watch_usage_limit.go`)と `internal/cli/org_watch.go` を変えた。#331 は `internal/org/send_screen.go` と、leader.md(+12 行)、director.md(+6 行)、director の skill(+12 行)、`/org` skill(send の表の 1 行)を変えた。run-test.sh は changed scope のまま golang だけを選んだ(`Language scope: changed (changed_languages)`、`Language packs selected: golang`)
- Evidence(cycle 3): `docs/evidence/test-2026-11-13-director-notice-handling-c3.log`(`.gitignore` の `docs/evidence/*.log` に当たるのでコミットしない)。run-test.sh の全出力、runner と meta、約 30 秒ごとの load、`(cached)` だった 5 パッケージの回し直し、名前を絞った 2 回の `-v`、変異 4 つと追加の g3、g4(とその対照)の当てた文と出力を入れた。wrapper 自身のログは `docs/evidence/verify-2026-11-15-113915.log`(名前の日付と時刻は UTC。5,616 行で、上のログの先頭 5,616 行と同じ)
- Date(cycle 2): 2026-11-13。run-test.sh は 15:57:00 から 16:33:04 JST(2,164 秒)。変異は 15:58 から 16:16 と 16:36 から 16:41、名前を絞った 2 回は 16:34 から 16:36 に流した
- Tester(cycle 2): tester サブエージェント(`/test`、パイプラインの cycle 2。main のチェックアウトの `.harness/state/standard-pipeline/cycle-count.json` が `"cycle":2`)
- Scope(cycle 2): cycle 1 の `/test`(`4618677e`)のあとに入ったのは、origin/main `e902705c`(#327)の取り込み `f65575ad`、sync-docs `f78e2b1b`、cycle 2 の 3 つのコミット(`8ae5ae1c`、`7bcef73e`、`8e8d26b5`)と記録のコミットである。cycle 2 のコミットが変えたのは director の skill の 4 写し(各 +24/-3)、`internal/org/prompts/leader.md`(+19/-9)、`internal/org/prompts_notice_test.go`(+107/-10、新しいテスト `TestRenderRolePrompt_Leader_HookSeatDecisionIsNotAnotherAnswer`)で、本番の Go のコードは変わっていない。`internal/cli/director_notice_skill_test.go` は cycle 2 で変わっていない。固定した断片を持つ `prompts_test.go`、`prompts_leader_plan_test.go`、`prompts_xreview_leader_test.go` は merge-base `e902705c` から変わっていない(`git diff --stat` が空)。`git diff origin/main...HEAD` は 18 ファイル、+1738/-16。run-test.sh は changed scope のまま golang だけを選んだ(`Language scope: changed (changed_languages)`、`Language packs selected: golang`)。origin/main はその後 `bbe2807d`(#328)まで 20 コミット進んでおり、この run はそれを取り込んでいない HEAD の木を回した
- Evidence(cycle 2): `docs/evidence/test-2026-11-13-director-notice-handling-c2.log`(`.gitignore` の `docs/evidence/*.log` に当たるのでコミットしない)。run-test.sh の全出力、runner と meta、30 秒ごとの load、`(cached)` だった 5 パッケージの回し直し、名前を絞った 2 回の `-v`、変異 9 つ(a、b、c、d、e1、e2、f、g と、その対照 g0)の適用した文と落ちた行を入れた。wrapper 自身のログは `docs/evidence/verify-2026-11-13-065700.log`(先頭 5,614 行は上のログと同じ)
- Date: 2026-11-13(run-test.sh は 13:35:26 から 14:07:36 JST)
- Plan: `docs/plans/active/2026-11-13-director-notice-handling.md`(`## Test plan`、AC6 と AC7、issue #318)
- Tester: tester サブエージェント(`/test`、パイプラインの cycle 1。`.harness/state/standard-pipeline/cycle-count.json` が cycle 1)
- Scope: `git diff main...HEAD`(HEAD `fcd368b0`、base `bb416c48`)。本番の Go のコードは変わっていない。テストは新しい 2 ファイルに 4 本(`internal/org/prompts_notice_test.go` に 3 本、`internal/cli/director_notice_skill_test.go` に 1 本)。既存のテストファイルに削除行は無い。`./scripts/run-test.sh` は changed scope のまま golang だけを選んだ(`Language scope: changed (changed_languages)`、`Language packs selected: golang`)。変異は repo の外の写し(`git archive fcd368b0` を `$TMPDIR` に展開したもの)で流し、worktree の `git status --porcelain --ignored` は前後で同じだった
- Evidence: `docs/evidence/test-2026-11-13-director-notice-handling.log`(`.gitignore` の `docs/evidence/*.log` に当たるのでコミットしない)。run-test.sh の全出力の後ろに、runner と meta、名前を絞った 2 回の `-v` の結果、変異 7 つの出力を足した。wrapper 自身のログは `docs/evidence/verify-2026-11-13-043527.log`(名前の日付と時刻は UTC。先頭 5,600 行は上のログと同じ)

## Test execution

### Cycle 2(HEAD `1d7eda3d`)

| Suite / Command | Tests | Passed | Failed | Skipped | Duration |
| --- | --- | --- | --- | --- | --- |
| `./scripts/run-test.sh`(rc 0、`==> All verifiers passed.`) | shell 45 本 + Go 10 パッケージ | shell 45 本すべて OK(PASS を含む行 5,195)、Go 10 パッケージ ok | 0(`--- FAIL` の行 0、`panic:` の行 0) | 2 行 | 2,164 秒 |
| └ Go: `internal/org` | パッケージ全体 | ok | 0 | — | 1,093.4 秒 |
| └ Go: `internal/cli` | パッケージ全体 | ok | 0 | — | 964.2 秒 |
| └ Go: `internal/config`、`internal/insights`、`cmd/ralph` | パッケージ全体 | ok | 0 | — | 56.9、3.3、1.4 秒 |
| └ Go: `internal/gitroot`、`internal/org/driver`、`internal/org/protocol`、`internal/scaffold`、`internal/upgrade` | パッケージ全体 | ok `(cached)`。run のあとに `-count=1` で回し直して ok | 0 | — | 回し直しで 2.6、1.5、1.8、1.3、2.7 秒 |
| `go test -count=1 -v -run '^(<下の 4 ファイルの 86 本の名前>\|TestDirectorSkill_.*\|TestRenderDirectorPrompt_.*\|TestRenderRolePrompt_Leader_.*)$' ./internal/org/` | 上位 146 本、subtest を含めて 251 | 251 | 0 | 0 | 99.7 秒 |
| `go test -count=1 -v -run '^TestDirectorNoticeSkill_' ./internal/cli/` | 上位 1 本、subtest 4(skill の 4 写し) | 5 | 0 | 0 | 0.7 秒 |
| 変異 9 つ(下の「変異の確かめ」) | 9 | handoff の (a)〜(e) に当てた 6 つと追加の f はすべて落ちた。隙間を示す g は残り、その対照 g0 は落ちた | — | — | skill の変異は各 1 秒未満、leader.md の変異は build と実行で各 2〜4 分 |

名前を絞った org の回の 86 本は、`prompts_notice_test.go` の 4 本、`prompts_test.go` の 43 本、`prompts_leader_plan_test.go` の 18 本、`prompts_xreview_leader_test.go` の 21 本の `func Test` の名前すべてである。86 本とも走って PASS したことを、`-v` の `--- PASS` の行と名前の一覧の突き合わせで確かめた(走らなかった名前は 0)。handoff が名指したテストは次のとおりで、すべて PASS だった。

| テスト | ファイル | 結果 |
| --- | --- | --- |
| `TestDirectorSkill_SaysWhatToDoWithTheHumanNotes` | `internal/org/prompts_notice_test.go` | PASS |
| `TestRenderDirectorPrompt_PointsAtTheHumanNotesSubsection` | 同 | PASS |
| `TestRenderRolePrompt_Leader_RaisesTheCodexHookNoteWithoutStopping` | 同 | PASS |
| `TestRenderRolePrompt_Leader_HookSeatDecisionIsNotAnotherAnswer`(cycle 2 で新規) | 同 | PASS |
| `TestRenderRolePrompt_Leader_FeatureOrgPlanApproval`(手順 4 の承認の読み方の断片、`:660`、`:694`、`:697`) | `internal/org/prompts_test.go`(変更なし) | PASS。ファイルのほかの 42 本も PASS |
| `TestRenderRolePrompt_Leader_FeatureOrgNonRecommendedForkIsAChangeRequest`(手順 4 の断片、`:300`、`:302`) | `internal/org/prompts_leader_plan_test.go`(変更なし) | PASS。ファイルのほかの 17 本も PASS |
| `TestRenderRolePrompt_Leader_XreviewCapQuestionIsAFork`(cap の読み手の 2 文、`:306`、`:307`) | `internal/org/prompts_xreview_leader_test.go`(変更なし) | PASS。ファイルのほかの 20 本も PASS |
| `TestDirectorNoticeSkill_QuotesTheFragmentsOfTheRealLines`(4 写しの subtest) | `internal/cli/director_notice_skill_test.go` | PASS(subtest 4 つとも) |

run-test.sh の Go の段で、`internal/org` は 30 分の上限(golang pack の `-timeout`)の 61% を使った。時間切れ(`panic: test timed out after 30m0s`)は出ず、handoff の 60 分での回し直しは要らなかった。`internal/org` と `internal/cli` の時間は cycle 1(1,067.4 秒と 943.4 秒)より 2% ほど長いだけである。5 パッケージが `(cached)` だった。どれも `internal/org` を import せず(`go list -deps` で確かめた)、この branch が変えたファイルを embed もしない。それでも run のあとに 5 つを `-count=1` で回し直し、すべて ok だった。

1 分の load は開始時 12.04、shell の段のあいだの最大が 44.89(16:05。ほかのセッションの作業と、下の変異 (c) の build と実行が重なった)、Go の段(16:14 ごろから 16:33)のあいだは 6.83〜26.88、終了時 13.02 だった。30 秒ごとの値は evidence にある。空き容量は開始時 71 GiB、終了後 65 GiB だった。

変異は run-test.sh の shell の段(15:58〜16:13)に流した。ただし e1、e2、f のテストの実行は 16:16 まで続き、Go の段の始めの 2 分ほどと重なった。重なったのは作ってあったテストの binary の実行だけで、build は 16:12 に終わっていた。g と g0、名前を絞った 2 回、5 パッケージの回し直しは、run-test.sh が終わったあと(16:34〜16:41)に流した。

起動は cycle 1 と同じである。run-test.sh は Bash tool の 600 秒の上限を超えるので、runner(start、end、rc を meta に書き、30 秒ごとに load を記録する)を perl の fork と setsid で切り離して起動し(INT、HUP、QUIT、TERM は DEFAULT に戻した)、`caffeinate -w <pid>` を perl の `alarm 175` で区切った前景の待ちで終わりを待った。この run のあいだ、手元でほかに全体の run は走っていなかった(開始の直前の 15:56 に見えたのは、ほかのセッションの名前を絞った `internal/org` の回だけだった。16:26 に見たときに走っていたテストの binary は、この run の `org.test` と `cli.test` だけだった)。

Skipped の 2 行は cycle 1 と同じ(`tests/test-plan-visual.sh` の browser detection と、`tests/test-secret-scan-branch.sh` の git 2.41 より古い本物の git の case)で、この diff と関係が無い。

### Cycle 1(HEAD `fcd368b0`)

| Suite / Command | Tests | Passed | Failed | Skipped | Duration |
| --- | --- | --- | --- | --- | --- |
| `./scripts/run-test.sh`(rc 0) | shell 45 本 + Go 10 パッケージ | shell 45 本すべて OK(PASS を含む行 5,195)、Go 10 パッケージ ok | 0 | 2 行 | 1,930 秒 |
| └ Go: `internal/org` | — | ok | 0 | — | 1,067.4 秒 |
| └ Go: `internal/cli` | — | ok | 0 | — | 943.4 秒 |
| └ Go: `internal/config`、`internal/upgrade`、`internal/scaffold`、`internal/org/driver` | — | ok | 0 | — | 18.0、9.0、8.4、8.0 秒 |
| └ Go: `internal/org/protocol`、`internal/gitroot`、`internal/insights`、`cmd/ralph` | — | ok | 0 | — | 7.6、6.8、3.7、1.7 秒 |
| `go test -count=1 -v -run '^(TestDirectorSkill_\|TestRenderDirectorPrompt_\|TestRenderRolePrompt_Leader_)' ./internal/org/` | 上位 126 本(`TestDirectorSkill_` 37、`TestRenderDirectorPrompt_` 9、`TestRenderRolePrompt_Leader_` 80)、subtest を含めて 207 | 207 | 0 | 0 | 108.0 秒 |
| `go test -count=1 -v -run '^TestDirectorNoticeSkill_' ./internal/cli/` | 上位 1 本、subtest 4(skill の 4 写し) | 5 | 0 | 0 | 0.5 秒 |
| 変異 7 つ(下の「変異の確かめ」) | 7 | 期待どおり 5 つが落ち、隙間を示す 2 つは残った | — | — | 各 1〜10 秒、(b) だけ 2 分弱 |

run-test.sh の Go の段では、10 パッケージとも `(cached)` にならずに走った。`internal/org` は 30 分の上限の 59% を使った。1 分の load は開始時 10.46、走っているあいだ 7.86〜17.33、終了時 5.44 で、ほかのセッションも同じマシンを使っていた。変異の回は shell の段のあいだ(13:36〜13:44)に流し、Go の段(13:47 から 13:50 のあいだに始まり 14:07 まで)には何も重ねていない。名前を絞った 2 回は run-test.sh が終わったあとに流した。空き容量は開始時も終了後も 75 GiB だった。`internal/org` と `internal/cli` の時間は、同じ日の朝の upgrade-behavior-notes の回(722.6 秒と 551.5 秒)より長い。load が高めだったことによるものとみられ、この diff はテストを 4 本足しただけで、4 本とも 0.01 秒以内に終わる。

run-test.sh は Bash tool の 600 秒の上限を超えるので、runner(start、end、rc を meta に書く)を perl の fork と setsid で切り離して起動し(INT、HUP、QUIT、TERM は DEFAULT に戻した)、`caffeinate -w <pid>` を perl の `alarm 175` で区切った前景の待ちで終わりを待った。

Skipped の 2 行は前の PR の /test と同じで、`tests/test-plan-visual.sh` の browser detection(`/Applications` に browser がある手元の環境による)と、`tests/test-secret-scan-branch.sh` の git 2.41 より古い本物の git の case(手元は git 2.49.0)である。どちらもこの diff と関係が無い。

## 受け入れ条件とテスト

### Cycle 2(HEAD `1d7eda3d`)

| AC | テスト | 結果 |
| --- | --- | --- |
| AC2 と AC6 skill の小節の hook の項(cycle 2 で書き換え: 人の答えを待たない、DECISION を送らず決まった note ですぐ resolve する、あとの人の答えは leader の行が `(active)` のときだけ `HOOK_SEAT: <座席 id>` の DECISION で送る) | `TestDirectorSkill_SaysWhatToDoWithTheHumanNotes`(小節の断片 12 個を cycle 2 で足した) | PASS。変異 (a) と (b) で 4 写しとも落ちる |
| AC6 skill の「件の処理」の手順 3 の hook の QUESTION の項(cycle 2 で新規) | 同じテストの行 `the hook QUESTION bullet of step 3 of ### 件の処理` | PASS。変異 f で 4 写しとも落ちる |
| AC5 と AC6 leader の prompt の hook の項(cycle 2 で書き換え: `HOOK_SEAT:` の DECISION はこの QUESTION だけへの答え、ほかの QUESTION や BLOCKED の答えは待ち続ける、この QUESTION の答えは待たずに進む) | `TestRenderRolePrompt_Leader_RaisesTheCodexHookNoteWithoutStopping`(断片 2 つを 6 つに置き換えた) | PASS。変異 e1 と e2 で落ちる |
| leader の手順 4 と cross-review の上限の読み手の除外(計画からのずれとして cycle 2 で足した文) | `TestRenderRolePrompt_Leader_HookSeatDecisionIsNotAnotherAnswer`(新規) | PASS。変異 (c) と (d) で落ちる |
| AC4 と AC6 headless の director の prompt | `TestRenderDirectorPrompt_PointsAtTheHumanNotesSubsection` | PASS(director.md は cycle 2 で変わっていない) |
| AC7 | `TestDirectorNoticeSkill_QuotesTheFragmentsOfTheRealLines`(4 写しの subtest) | PASS(テストも本番の 2 つの行も cycle 2 で変わっていない) |
| Regression: 固定した断片の 3 ファイル | `prompts_test.go` 43 本、`prompts_leader_plan_test.go` 18 本、`prompts_xreview_leader_test.go` 21 本 | すべて PASS。3 ファイルとも変わっていない |

### Cycle 1(HEAD `fcd368b0`)

| AC | テスト | 結果 |
| --- | --- | --- |
| AC6 skill の 4 写しの小節と禁止の項 | `TestDirectorSkill_SaysWhatToDoWithTheHumanNotes` | PASS。変異 (a) と (c) で落ちる |
| AC6 headless の director の prompt(herdr と launch_cmd) | `TestRenderDirectorPrompt_PointsAtTheHumanNotesSubsection`(`/herdr`、`/cmd`) | PASS |
| AC6 leader の prompt | `TestRenderRolePrompt_Leader_RaisesTheCodexHookNoteWithoutStopping` | PASS。変異 (b) で落ちる |
| AC7 2 つの行の決まった部分を skill の小節が引く | `TestDirectorNoticeSkill_QuotesTheFragmentsOfTheRealLines`(4 写しの subtest) | PASS。skill の側の変異 (a) と、本番の文面の側の変異 p1、p2 で落ちる |
| Regression: 既存の `TestDirectorSkill_*`、`TestRenderDirectorPrompt_*`、`TestRenderRolePrompt_Leader_*` | 上の名前を絞った回(新しい 3 本を除いて上位 123 本) | すべて PASS。既存のテストは変わっていない |

## 変異の確かめ

### Cycle 2(HEAD `1d7eda3d`)

`git archive 1d7eda3d` を `$TMPDIR/t318c2-mut/tree` に展開し、その写しだけに当てた。skill は実行時に `runtime.Caller` から求めた repo の根で読まれるので、写しの中で `org.test` と `cli.test` を 1 度だけ build し、skill の 4 写しをその場で書き換えて走らせ、元に戻して 4 写しとも `cmp` で一致を確かめた(どの変異のあとも一致した)。leader.md は `go:embed` なので、変異ごとに `go test -c -overlay` で binary を作り直した(写しの leader.md そのものは書き換えていない)。文の削除は perl の小さな script で行った。始まりの目印がファイルにちょうど 1 度あることを確かめ、その後ろで最初に現れる終わりの目印までを消し(800 バイトまで)、消した文を evidence に残した。worktree の `git status --porcelain` は前後とも空で、`--ignored` の差は `.harness/cache/go-build/` の build cache と wrapper のログ `docs/evidence/verify-2026-11-13-065700.log` だけだった。skill の変異では `^TestDirectorSkill_`(37 本)と `^TestDirectorNoticeSkill_` を、leader.md の変異では `^TestRenderRolePrompt_`(126 本)を走らせた。

| 変異 | 当てた所 | 結果 | 落ちた行 |
| --- | --- | --- | --- |
| (a) 決まった note ですぐ resolve する文を消す | 4 写しの小節の 1513〜1516 行、`` 行と leader の問い(座席をこのまま使うか止めるか)を人に伝えたら、DECISION は送らず、すぐに `ralph org inbox resolve <id> --note '人に伝えた(hook の承認の知らせ)'` で件を resolve する。 `` | skill のテストが落ちる(37 本のうちこの 1 本だけ)。AC7 のテストは通る | `internal/org/prompts_notice_test.go:170` が 4 回(写しごとに 1 回): `` <写し>: the subsection ### 人に向けた知らせの行 does not contain "行と leader の問い(座席をこのまま使うか止めるか)を人に伝えたら、DECISION は送らず、すぐに `ralph org inbox resolve <id> --note '人に伝えた(hook の承認の知らせ)'` で件を resolve する" after the fragments before it -- see this test's doc comment for what to update `` |
| (b) `HOOK_SEAT:` のヘッダの文を消す | 4 写しの小節の 1524〜1526 行、`` DECISION には、`TYPE: DECISION` の次の行に、QUESTION が名指した座席を `HOOK_SEAT: <座席 id>` のヘッダで書き(`HOOK_SEAT: reviewer` など)、本文は `SUMMARY: hook の QUESTION への答え。<人の判断>` で始める。 `` | skill のテストが落ちる(この 1 本だけ)。AC7 のテストは通る | `prompts_notice_test.go:170` が 4 回(写しごと): `<写し>: the subsection ### 人に向けた知らせの行 does not contain "DECISION には、…で始める" after the fragments before it ...`(引いた断片は消した文の句点の前まで) |
| (c) leader.md の手順 4 の除外を消す | `internal/org/prompts/leader.md` の 251〜253 行、`` `HOOK_SEAT:` のヘッダの付いた DECISION は、承認でも直しの依頼でもない(「運用規律」の節の codex の hook の項)ので、計画を直さず、承認か直しの依頼の DECISION を待ち続ける。 `` | `TestRenderRolePrompt_Leader_HookSeatDecisionIsNotAnotherAnswer` だけが落ちる(126 本のうち 1 本) | `prompts_notice_test.go:426`: `` expected the 機能ごとの org item naming "4. (per-plan)" to say [...] in this order, but "4. (per-plan)`HOOK_SEAT:` のヘッダの付いた DECISION は、承認でも直しの依頼でもない(「運用規律」の節の codex の hook の項)ので、計画を直さず、承認か直しの依頼の DECISION を待ち続ける" is missing or out of order `` |
| (d) leader.md の cap の読み手の除外を消す | 同 937〜939 行、`` `HOOK_SEAT:` のヘッダの付いた DECISION はこの答えではない(「運用規律」の節の codex の hook の項)ので、それを受けても上限の QUESTION を上げ直さず、この答えを待ち続ける。 `` | 同じ 1 本だけが落ちる | `prompts_notice_test.go:431`: `` expected the cross-review subsection of the 機能ごとの org section to say [...] in this order, but "`HOOK_SEAT:` のヘッダの付いた DECISION はこの答えではない(「運用規律」の節の codex の hook の項)ので、それを受けても上限の QUESTION を上げ直さず、この答えを待ち続ける" is missing or out of order `` |
| (e1) hook の項の `や BLOCKED` を消す | 同 1362 行、`QUESTION や BLOCKED の答えではないので` を `QUESTION の答えではないので` に(8e8d26b5 で足した語) | `TestRenderRolePrompt_Leader_RaisesTheCodexHookNoteWithoutStopping` だけが落ちる | `prompts_notice_test.go:366`: `expected the 運用規律 bullet on the codex hook trust note to say [...] in this order, but "計画の承認(「機能ごとの org」の節の手順 4)、cross-review の上限の QUESTION、ほかに答えを待っている QUESTION や BLOCKED の答えではないので、それらの答えは待ち続けてください" is missing or out of order` |
| (e2) hook の項の `QUESTION や BLOCKED` を消す | 同じ所を `の答えではないので` に | 同じ 1 本だけが落ちる | e1 と同じ行と同じ断片 |
| f(追加)「件の処理」の手順 3 の項を消す | 4 写しの 964〜967 行の項全体(`- codex の hook の承認の知らせを載せた QUESTION(本文に` から `人の答えを待たずに resolve する。` まで) | skill のテストが落ちる(この 1 本だけ) | `prompts_notice_test.go:170` が 8 回(写しごとに断片 2 つ): `<写し>: the hook QUESTION bullet of step 3 of ### 件の処理 does not contain "- codex の hook の承認の知らせを載せた QUESTION(…)"` と `... does not contain "下の「進捗の報告」の「人に向けた知らせの行」のとおり、人の答えを待たずに resolve する"` |
| g(追加)ヘッダの名前を leader の側だけで変える | leader.md の `HOOK_SEAT` を 4 行とも `HOOK_SEAT_ID` にし、`prompts_notice_test.go` の 290 行目より後ろ(leader の側。8 行で、うち 4 行は断片、4 行はコメント)も同じに変える。skill の 4 写しと skill の側の文字列(132 行)はそのまま | 残る。`^(TestDirectorSkill_\|TestRenderRolePrompt_)` の上位 163 本がすべて PASS | —(下の G5) |
| g0(g の対照)leader.md だけを変える | g と同じ leader.md、テストはそのまま | 2 本が落ちる | `TestRenderRolePrompt_Leader_RaisesTheCodexHookNoteWithoutStopping` と `TestRenderRolePrompt_Leader_HookSeatDecisionIsNotAnotherAnswer` |

handoff の 5 つ (a)〜(e) はどれも落ちた。(e) は、テストが `QUESTION や BLOCKED` を含む文を丸ごと固定している(`prompts_notice_test.go:332-333`、`leaderHookNotePhrases`)ので、語を消しても変えても落ちる。隙間ではない。(c) と (d) では新しいテストだけが落ち、変わっていない 3 ファイルの手順 4 と cap の読み手のテストは通った。除外の文を固定しているのは新しいテストだけで、役割の分け方どおりである。f は cycle 2 で足された「件の処理」の項に当てた追加の変異で、4 写しとも落ちた。g は G5 の隙間を示すために流した。

### Cycle 1(HEAD `fcd368b0`)

どの変異も `$TMPDIR` の写しで当て、当てる前に元を残し、当てたあとに戻して `cmp` で一致を確かめた。skill は実行時に `runtime.Caller` から求めた repo の根で読まれるので、写しの中でビルドしたテストの binary を写しの中で走らせた。leader.md は `go:embed` なので、(b) は写しの leader.md を変えてから binary を作り直した。

| 変異 | 当てた所 | 結果 | 落ちた行 |
| --- | --- | --- | --- |
| (a) 2 つの決まった部分を消す | `templates/base/.agents/skills/director/SKILL.md` の小節の 1452 行の `run 'ralph upgrade' in ` と 1483 行の `; run 'ralph doctor' for ` | AC7 のテストが落ちる(その写しの subtest だけ FAIL、ほかの 3 写しは PASS)。skill のテストも落ちる | `internal/cli/director_notice_skill_test.go:100` が 2 回: `templates/base/.agents/skills/director/SKILL.md: the "### 人に向けた知らせの行" subsection does not quote "run 'ralph upgrade' in ", the fragment of the line scaffoldVersionNotice prints; ...` と `... does not quote "; run 'ralph doctor' for ", the fragment of the line codexHookTrustNotice prints; ...`。skill のテストは `internal/org/prompts_notice_test.go:129`(決まった文の 2 つ)と `:166`(CLI の断片の 2 つ) |
| (b) leader.md の hook の項を消す | `internal/org/prompts/leader.md` の 1335〜1359 行(25 行、運用規律の項) | leader のテストが落ちる。`-run 'TestRenderRolePrompt_'` の上位 125 本のうち落ちたのはこの 1 本だけ | `internal/org/prompts_notice_test.go:306`: ``運用規律 section has no top-level bullet starting "- `ralph org spawn` が stderr に、":`` |
| (c) 「してはいけないこと」の `ralph upgrade` の項を消す | `.claude/skills/director/SKILL.md` の 2039〜2041 行 | skill のテストが落ちる。AC7 のテストは通る(小節だけを読むので、禁止の項は見ない) | `internal/org/prompts_notice_test.go:129` が 3 回: ``.claude/skills/director/SKILL.md: the ralph upgrade bullet of ## してはいけないこと does not contain "`ralph upgrade` を打たない。雛形の版の `note:` の行が名指していても、人に頼まれても打たない" after the fragments before it ...``、同じ形で `codex の設定を書き換えず、codex を起動して hook を承認しない` と `(「進捗の報告」の「人に向けた知らせの行」)` |
| p1 本番の版の行の文面を変える | `scaffold_version_notice.go:61` の `run 'ralph upgrade' in %s` を `at %s` に(`-overlay`) | AC7 のテストが落ちる | `director_notice_skill_test.go:64`(行が断片を含まない)と `:74`(行が断片と絶対パスで終わらない) |
| p2 本番の hook の行の文面を変える | `codex_hook_trust_notice.go:80` の `; run 'ralph doctor' for %s` を `to see %s` に(`-overlay`) | AC7 のテストが落ちる | `director_notice_skill_test.go:64`(行が断片を含まない) |
| p3 hook の行の断片の外を変える | 同じ行の ` and how to %s` を ` and how %s` に | AC7 のテストは通る(残る)。既存の `CodexHookTrust` のテスト 8 本が落ちる | 下の G1 |
| p4 版の行の断片の外を変える | `scaffold_version_notice.go:60` の `older than this CLI (%s)` を `behind this CLI (%s)` に | AC7 のテストは通る(残る)。既存の `ScaffoldVersion` のテスト 10 本が落ちる | 下の G1 |

(a)〜(c) は handoff の 3 つで、どれも名指したテストが落ちた。p1 と p2 は AC7 の「文面が変わったら落ちる」を本番の側から確かめたもので、verify が読んで判断した所を実行で確かめた。p3 と p4 は G1 の隙間を示すために流した。

## Coverage

### Cycle 2(HEAD `1d7eda3d`)

- Statement: 計測していない。本番の Go のコードは cycle 2 でも変わっていないため、Go の coverage の数字は変わらない
- Branch: 計測していない
- Function: 該当なし
- Notes: cycle 2 が足した文のうち、テストが読むのは、skill の小節の hook の項の 12 断片、「件の処理」の手順 3 の項の 2 断片、leader の hook の項の 6 断片(cycle 1 の 2 断片を置き換えた)、手順 4 と cap の読み手の除外(前後の断片と順に読む 3 断片ずつ)である。skill の 3 か所 (a)、(b)、f は 4 写しすべてに当て、4 写しとも落ちた

### Cycle 1(HEAD `fcd368b0`)

- Statement: 計測していない。本番の Go のコードが変わっていないため、Go の coverage の数字は変わらない
- Branch: 計測していない
- Function: 該当なし
- Notes: この diff が足したのは skill と prompt の文と、その文を読むテストである。テストが文をどこまで覆うかは、変異の表で確かめた。小節の決まった文(27 断片)、昇格の 2、出力先の 3、禁止の項、director の prompt の 6 断片、leader の prompt の 23 断片と止める規則の例外の一句をテストが読む。(a) は生成された写しの 1 つ、(c) は元の写しの 1 つに当て、どちらも当てた写しだけが落ちた。4 写しすべてに断片の変異を当てる確かめは S3 の implementer が済ませている(計画の Progress)

## Failure analysis

### Cycle 2(HEAD `1d7eda3d`)

| Test | Error | Root cause | Proposed fix |
| --- | --- | --- | --- |
| — | 失敗なし | — | — |

### Cycle 1(HEAD `fcd368b0`)

| Test | Error | Root cause | Proposed fix |
| --- | --- | --- | --- |
| — | 失敗なし | — | — |

## Regression checks

### Cycle 2(HEAD `1d7eda3d`)

| Previously broken behavior | Status | Evidence |
| --- | --- | --- |
| 固定した断片を持つ既存のテスト(`prompts_test.go` 43 本、`prompts_leader_plan_test.go` 18 本、`prompts_xreview_leader_test.go` 21 本。merge-base `e902705c` から変わっていない) | 通る | 名前を絞った org の回。手順 4 の承認の読み方の断片(`prompts_test.go:694`、`prompts_leader_plan_test.go:300`)と、cap の読み手の続いた 2 文(`prompts_xreview_leader_test.go:306-307`、順に読む)は、cycle 2 で足した除外の文が前や間に入っても、元の順のまま見つかる |
| cycle 1 のテスト 4 本(`prompts_notice_test.go` の 3 本と `director_notice_skill_test.go`) | 通る | 名前を絞った 2 回 |
| `TestDirectorSkill_*` 37 本、`TestRenderDirectorPrompt_*` 9 本、`TestRenderRolePrompt_Leader_*` 81 本(新しい 1 本を含む) | 通る | 名前を絞った org の回 |
| 取り込んだ origin/main `e902705c`(#327 watch-usage-limited)と合わせた `internal/org` と `internal/cli` の全体 | 通る | run-test.sh の Go の段(`(cached)` なし) |

### Cycle 1(HEAD `fcd368b0`)

| Previously broken behavior | Status | Evidence |
| --- | --- | --- |
| director の skill の既存の確かめ(`TestDirectorSkill_*` の既存 36 本) | 通る | 名前を絞った org の回 |
| director の prompt の既存の確かめ(`TestRenderDirectorPrompt_*` の既存 8 本。ralph org の命令に設定のフラグを付ける `TestRenderDirectorPrompt_GivesEveryRalphOrgCommandTheConfig` を含む) | 通る | 名前を絞った org の回。新しい項は `ralph org start` を名指さない書き方にしてあり、この確かめと食い違わない |
| leader の prompt の既存の確かめ(`TestRenderRolePrompt_Leader_*` の既存 79 本) | 通る | 名前を絞った org の回 |
| 2 つの知らせの文面(#313、#310) | 通る | run-test.sh の `internal/cli`(`(cached)` なし)。本番の Go は変わっていない |
| `internal/org` と `internal/cli` の全体 | 通る | run-test.sh の Go の段 |

## Test gaps

### Cycle 2(HEAD `1d7eda3d`)

- G1〜G4 は cycle 1 のまま成り立つ。AC7 のテストも本番の 2 つの行も cycle 2 で変わっていない。G3 は cycle 2 の文にも当てはまる。director が hook の QUESTION を本当にすぐ resolve するか、承認待ちや cap の待ちの leader が `HOOK_SEAT:` の DECISION を読み分けるかは、単体テストでは確かめられない(verify の cycle 2 と同じ判断)
- G5(LOW): ヘッダの名前 `HOOK_SEAT:` を、テストは skill の側(`prompts_notice_test.go:132-133`)と leader の側(`:329`、`:331`、`:402`、`:408`)で別々の文字列として持ち、2 つが同じであることはどのテストも確かめていない。変異 g(leader.md と leader の側の文字列だけを `HOOK_SEAT_ID` に変える)はすべてのテストを通り、そのとき skill の 4 写しは director に `HOOK_SEAT:` を送らせ、leader は `HOOK_SEAT_ID:` を探す。対照 g0(leader.md だけを変える)は 2 本が落ちるので、片側の文面だけを変えれば捕まる。捕まらないのは、片側の文面とその側のテストを一緒に直したときである。閉じるなら、`hookSeatHeader = "HOOK_SEAT:"` のような定数を 1 つ置いて両方の断片をそれで組み立てるか、skill の小節と leader.md の hook の項から見つけたヘッダが同じことを確かめる 1 本を足す
- G6(INFO): skill の小節が示す DECISION の形(`TYPE: DECISION`、`HOOK_SEAT: reviewer`、空行、`SUMMARY: hook の QUESTION への答え。…`)を、`protocol.Parse` と `Validate` に通すテストは無い。読んだ限りでは、`splitHeaderLine`(`internal/org/protocol/protocol.go:171-182`)は空白を含まないキーをすべて受けるので `HOOK_SEAT` も `Fields` に入り、`ralph org send` は通すはずである。実行では確かめていない
- G7(INFO): leader.md で `HOOK_SEAT:` の除外を名指して書いた読み手は、手順 4 と cap の読み手の 2 つだけで、テストもその 2 つを固定している。ほかの DECISION の待ち(cap の答えが c のときの「次の DECISION を待つ」、leader.md 961 行など)は、hook の項の「ほかに答えを待っている QUESTION や BLOCKED の答えではない」という一般の文に頼っている。この一般の文は e1 と e2 で落ちることを確かめた。ただ、あとから DECISION の読み手が足されても、除外を書くことをテストは求めない

### Cycle 1(HEAD `fcd368b0`)

- G1(LOW): AC7 のテストが本番の行と結び付けるのは 2 つの断片だけで、小節の行の形のほかの部分は結び付けていない。小節は版の行を `note: <動詞>: the scaffold in <dir> (.ralph/manifest.toml) is ralph <雛形の版>, older than this CLI (<CLI の版>), ...`(1451 行)、hook の行の終わりを `...; run 'ralph doctor' for <the key(s)> and how to <直し方>`(1483 行)と書く。本番の `older than this CLI` を変える変異 p4 と、` and how to ` を変える変異 p3 は、AC7 のテストでも skill のテストでも落ちなかった。どちらも既存の `ScaffoldVersion`(10 本)と `CodexHookTrust`(8 本)のテストでは落ちるので、文面の変更が気づかれずに入ることはない。ただ、それらのテストの失敗文は skill に触れないので、直す人が skill の 4 写しを直し忘れうる。計画は 2 つの断片だけを結び付けると決めており(Test plan の Edge cases)、計画どおりの範囲である。閉じるなら、AC7 のテストで行の形の英語の部分(`older than this CLI (`、` and how to `)も断片に足す
- G2(LOW、self-review の L3 で受け入れ済み): AC7 のテストは hook の行が断片を含むことを `Contains` で確かめるだけで、断片で終わることまでは見ない。小節の終わりを切る補助は `#`、`##`、`###` の見出しで止まり、internal/org の補助と規則が違う。今は差が出ない
- G3(INFO): director と leader が実際の会話でこの文に従うかは、単体テストでは確かめられない(verify と同じ判断)
- G4(INFO): 変異 (c) では AC7 のテストが通った。AC7 のテストは小節だけを読み、禁止の項は skill のテストが受け持つので、役割の分け方どおりである

## Cycle 3(HEAD `ad590839`)

### Test execution(cycle 3)

| Suite / Command | Tests | Passed | Failed | Skipped | Duration |
| --- | --- | --- | --- | --- | --- |
| `./scripts/run-test.sh`(rc 0、`==> All verifiers passed.`) | shell 45 本 + Go 10 パッケージ | shell 45 本すべて OK(PASS を含む行 5,195)、Go 10 パッケージ ok | 0(`--- FAIL` の行 0、`panic:` の行 0) | 2 行 | 2,227 秒 |
| └ Go: `internal/org` | パッケージ全体 | ok | 0 | — | 1,102.6 秒 |
| └ Go: `internal/cli` | パッケージ全体 | ok | 0 | — | 945.2 秒 |
| └ Go: `internal/config`、`internal/insights`、`cmd/ralph` | パッケージ全体 | ok | 0 | — | 30.2、3.2、1.4 秒 |
| └ Go: `internal/gitroot`、`internal/org/driver`、`internal/org/protocol`、`internal/scaffold`、`internal/upgrade` | パッケージ全体 | ok `(cached)`。run のあとに `-count=1` で回し直して ok | 0 | — | 回し直しで 2.9、2.0、1.8、2.4、3.8 秒 |
| `go test -count=1 -v -run '^(<下の 6 ファイルの 109 本の名前>\|TestDirectorSkill_.*\|TestRenderDirectorPrompt_.*\|TestRenderRolePrompt_Leader_.*\|TestOrgSkill_.*)$' ./internal/org/` | 上位 200 本、subtest を含めて 450 | 450 | 0 | 0 | 152.5 秒 |
| `go test -count=1 -v -run '^TestDirectorNoticeSkill_' ./internal/cli/` | 上位 1 本、subtest 4(skill の 4 写し) | 5 | 0 | 0 | 0.9 秒 |
| 変異 4 つと追加の 2 つ(下の「変異の確かめ(cycle 3)」) | 6 と対照 2 | handoff の (a)〜(d) はすべて落ちた。隙間を示す g3 と g4 は残り、その対照 g3-0 と g4-0 は落ちた | — | — | skill の変異は各 1〜3 秒、leader.md の変異は build 21 秒と実行 2 分強 |

名前を絞った org の回の 109 本は、`prompts_notice_test.go` の 5 本、`prompts_test.go` の 43 本、`prompts_leader_plan_test.go` の 18 本、`prompts_xreview_leader_test.go` の 21 本、#330 の `usage_limit_screen_test.go` の 8 本と `watch_usage_limit_test.go` の 14 本の `func Test` の名前すべてである。109 本とも `--- PASS` の行があることを、名前の一覧との突き合わせで確かめた(走らなかった名前は 0)。前の cycle と同じ範囲の確かめとして、`TestDirectorSkill_` 37 本、`TestRenderDirectorPrompt_` 10 本、`TestRenderRolePrompt_Leader_` 82 本、`TestOrgSkill_` 30 本も同じ回で走らせ、すべて PASS だった(名前の重なりを除いて上位 200 本)。cycle 2 の 9 本と 81 本から 1 本ずつ増えたのは、#331 が足した `TestRenderDirectorPrompt_SendTextTypedNotSubmittedReadsTheLeadersScreen` と `TestRenderRolePrompt_Leader_SendTextTypedNotSubmittedEscalates` である。handoff が名指したテストの結果は次のとおり。

| テスト | ファイル | 結果 |
| --- | --- | --- |
| `TestOrgSkill_InboxResolvesTheHookQuestionAtOnce`(cycle 3 で新規) | `internal/org/prompts_notice_test.go` | PASS |
| `TestRenderRolePrompt_Leader_HookSeatDecisionIsNotAnotherAnswer` | 同 | PASS |
| `TestDirectorSkill_SaysWhatToDoWithTheHumanNotes`、`TestRenderDirectorPrompt_PointsAtTheHumanNotesSubsection`(`/herdr`、`/cmd`)、`TestRenderRolePrompt_Leader_RaisesTheCodexHookNoteWithoutStopping` | 同 | PASS |
| 固定した断片の 3 ファイル(手順 4 の承認の読み方、cap の読み手など) | `prompts_test.go` 43 本、`prompts_leader_plan_test.go` 18 本、`prompts_xreview_leader_test.go` 21 本(どれも `1d7eda3d` から変更なし) | 82 本すべて PASS |
| `TestDirectorNoticeSkill_QuotesTheFragmentsOfTheRealLines`(4 写しの subtest) | `internal/cli/director_notice_skill_test.go` | PASS(subtest 4 つとも) |
| #330 の `TestUsageLimitScreen_*` 5 本と `TestUsageLimitSentences_AreStoppedSentences`、`TestUsageLimitWaitingSentences_AreWaitingSentences`、`TestUsageLimitLine` | `internal/org/usage_limit_screen_test.go` | 8 本すべて PASS |
| #330 の見張りの `TestWatchUsageLimit_*` | `internal/org/watch_usage_limit_test.go` | 14 本すべて PASS |

run-test.sh の Go の段で、`internal/org` は 30 分の上限の 61% を使った。runner の環境に `GOFLAGS` は無く、`go env GOFLAGS` も空だったので、上限は golang pack の `-timeout=30m`(`packs/languages/golang/verify.sh:115`)である。時間切れ(`panic: test timed out after 30m0s`)は出ず、handoff の 60 分での回し直しは要らなかった。この branch が触っていないテストの失敗も無く、パッケージの回し直しも要らなかった。`internal/org` と `internal/cli` の時間は cycle 2(1,093.4 秒と 964.2 秒)とほぼ同じで、org は 1% 長く、cli は 2% 短い。`(cached)` の 5 パッケージはどれも `internal/org` を import しない(`go list -deps` で確かめた)。それでも run のあとに `-count=1` で回し直し、すべて ok だった。

1 分の load は開始時 8.85、shell の段(20:39 から 20:57 ごろ)のあいだが 7.68〜139.03、Go の段(20:57 ごろから 21:16)のあいだが 7.12〜36.12、終了時 14.66 だった(約 30 秒ごとに 72 回)。100 を超えたのは 20:42:54 から 20:45:22 で、最大の 139.03 は 20:45:22 の値である。20:43 に CPU の上位を見ると、ほかのセッションの Xcode の Swift の build(`swift-frontend`、`SWBBuildService`)が並んでいた。同じころ、変異 (c) と (d) の binary の build(20:42〜20:43)と、`nice -n 19` での実行(20:44〜20:52)も重なっている。どちらがどれだけ load を上げたかは分けていない。空き容量は開始時 49 GiB、終了時 47 GiB だった。

起動は cycle 1 と cycle 2 と同じである。runner(start、end、rc を meta に書き、約 30 秒ごとに load を記録する)を perl の fork と setsid で切り離して起動し(INT、HUP、QUIT、TERM は DEFAULT に戻した)、`caffeinate -w <pid>` を perl の `alarm 170` で区切った前景の待ちで終わりを待った。開始の直前に `pgrep` で見たとき、手元でほかの `go test` も run-test.sh も走っていなかった。

Skipped の 2 行は前の cycle と同じ(`tests/test-plan-visual.sh` の browser detection と、`tests/test-secret-scan-branch.sh` の git 2.41 より古い本物の git の case)で、この diff と関係が無い。

### 受け入れ条件とテスト(cycle 3)

| AC | テスト | 結果 |
| --- | --- | --- |
| AC2 と AC6 `/org` skill の `## 受信箱` の hook の QUESTION の項(cycle 3 で新規: 決まった note ですぐ resolve する、理由は見張りの扱い、あとの答えは leader の行が `(active)` のときだけ、QUESTION が名指した座席を `HOOK_SEAT:` に書いた DECISION で送る) | `TestOrgSkill_InboxResolvesTheHookQuestionAtOnce`(新規) | PASS。変異 (a)(1 写しから項を消す)と (b)(4 写しから「QUESTION が名指した座席」を消す)で落ちる |
| AC5 と AC6 leader の hook の項の本文の 3 文(cycle 3 で書き換え: 動機、決まった note、`HOOK_SEAT:` の答え)と「director か人が」の文 | `TestRenderRolePrompt_Leader_RaisesTheCodexHookNoteWithoutStopping`(断片を cycle 3 で書き換えた) | PASS。変異 (c)(動機の文を消す)で落ちる |
| leader の手順 4 と cap の読み手の除外 | `TestRenderRolePrompt_Leader_HookSeatDecisionIsNotAnotherAnswer` | PASS。変異 (d)(手順 4 の除外を消す)で落ちる |
| AC1〜AC3 と AC6 director の skill の小節と禁止の項 | `TestDirectorSkill_SaysWhatToDoWithTheHumanNotes` | PASS(小節は cycle 3 で変わっていない。#330 と #331 の取り込みで skill のほかの所が変わった) |
| AC4 と AC6 headless の director の prompt | `TestRenderDirectorPrompt_PointsAtTheHumanNotesSubsection`(`/herdr`、`/cmd`) | PASS(#331 が director.md に 6 行を足したあとも通る) |
| AC7 | `TestDirectorNoticeSkill_QuotesTheFragmentsOfTheRealLines`(4 写しの subtest) | PASS(テストも、本番の 2 つの行を作る `scaffold_version_notice.go` と `codex_hook_trust_notice.go` も、`fcd368b0` から変わっていない) |
| Regression: 固定した断片の 3 ファイル | `prompts_test.go` 43 本、`prompts_leader_plan_test.go` 18 本、`prompts_xreview_leader_test.go` 21 本 | すべて PASS。#331 が leader.md に 12 行を足したあとも通る |
| Regression: 取り込んだ #330 の画面の読みと見張り | `usage_limit_screen_test.go` 8 本、`watch_usage_limit_test.go` 14 本 | すべて PASS |

### 変異の確かめ(cycle 3)

`git archive ad590839` を `$TMPDIR/t318c3/tree` に展開し、その写しだけに当てた。skill は実行時に `runtime.Caller` から求めた repo の根で読まれるので、写しの中で `org.test` を 1 度だけ build し、skill の写しをその場で書き換えて走らせ、元に戻して 4 写しとも `cmp` で一致を確かめた(どの変異のあとも一致した)。leader.md は `go:embed` なので、変異ごとに `go test -c -overlay` で binary を作り直した(写しの leader.md そのものは書き換えていない)。文の削除と置き換えは perl の小さな script で行い、消す文がファイルにちょうど 1 度あることを確かめてから置き換えた。skill の変異では、`/org` skill を読むテストファイル 12 本の上位 88 本を走らせた(`orgSkillCopies` か `checkOrgSkillPlaces` を使う 7 ファイルと、`skills/org/SKILL.md` を名指す 5 ファイル。HEAD の写しでは 88 本すべて PASS)。leader.md の変異では `^TestRenderRolePrompt_Leader_`(82 本。HEAD の写しではすべて PASS)を走らせた。worktree の `git status --porcelain` は前後とも空で、`--ignored` の差は `.harness/cache/go-build/` の build cache だけだった。

| 変異 | 当てた所 | 結果 | 落ちた行 |
| --- | --- | --- | --- |
| (a) `/org` skill の hook の QUESTION の項を 1 写しから消す | `.claude/skills/org/SKILL.md` の 530〜546 行(17 行、1,559 バイト。`- codex の hook の承認の知らせを載せた QUESTION(本文に` から `と同じ。` まで)。ほかの 3 写しはそのまま | `TestOrgSkill_InboxResolvesTheHookQuestionAtOnce` だけが落ちる(88 本のうち 1 本)。落ちた行はすべて消した写しのもので、ほかの 3 写しの行は無い | 15 行。`prompts_notice_test.go:523` が 13 回(項の断片 12 個と、節の並びの断片 `- codex の hook の承認の知らせを載せた QUESTION`)で、1 つ目は `` .claude/skills/org/SKILL.md: the bullet of ## 受信箱 on the hook QUESTION does not contain "- codex の hook の承認の知らせを載せた QUESTION(本文に `; run 'ralph doctor' for ` を含む `note:` の行がある件。leader が `ralph org spawn` の行を上げたもの)は、上の項の順(送ってから resolve)の例外" after the fragments before it -- see this test's doc comment for what to update ``。`:541` が `.claude/skills/org/SKILL.md: the section ## 受信箱 has 0 bullets starting "- codex の hook の承認の知らせを載せた QUESTION", want 1 -- ...`。`:544` が `.claude/skills/org/SKILL.md: the bullet of ## 受信箱 on the hook QUESTION does not quote "; run 'ralph doctor' for ", the end of the codex hook trust note, on one line as it is -- ...` |
| (b) 「QUESTION が名指した座席」の語を消す | 4 写しの 540〜541 行の `QUESTION が` と改行と `名指した座席`(33 バイト)。行は `` `TYPE: DECISION` の次の行に、(spawn の行を出した座席で、`leader` ではない)を `` になる | 同じ 1 本だけが落ちる(88 本のうち 1 本) | `prompts_notice_test.go:523` が 4 回(写しごとに 1 回): `` <写し>: the bullet of ## 受信箱 on the hook QUESTION does not contain "QUESTION が名指した座席(spawn の行を出した座席で、`leader` ではない)を `HOOK_SEAT: <座席 id>` のヘッダで書いた DECISION で答えを送り" after the fragments before it -- see this test's doc comment for what to update `` |
| (c) leader の本文の動機の文を消す | `internal/org/prompts/leader.md` の 1370〜1371 行の `この件が resolve されないあいだ、見張りは leader が答えを待っているとみなし、leader が止まっても知らせない。`(153 バイト。本文は「leader はこの問いの」から始まる 2 文になる) | `TestRenderRolePrompt_Leader_RaisesTheCodexHookNoteWithoutStopping` だけが落ちる(82 本のうち 1 本) | `prompts_notice_test.go:384`: `` expected the 運用規律 bullet on the codex hook trust note to say [...] in this order, but "「この件が resolve されないあいだ、見張りは leader が答えを待っているとみなし、leader が止まっても知らせない。leader はこの問いの答えを待たずに作業を続けているので、この件は答える前でも `--note '人に伝えた(hook の承認の知らせ)'` を付けてすぐ resolve してよい。答えは `HOOK_SEAT: <座席 id>` のヘッダを付けた DECISION で送ってほしい(ほかの待ちの答えと見分けるため)。」" is missing or out of order `` |
| (d) leader.md の手順 4 の `HOOK_SEAT:` の除外を消す | 同 251〜253 行の `` `HOOK_SEAT:` のヘッダの付いた DECISION は、承認でも直しの依頼でもない(「運用規律」の節の codex の hook の項)ので、計画を直さず、承認か直しの依頼の DECISION を待ち続ける。 ``(239 バイト) | `TestRenderRolePrompt_Leader_HookSeatDecisionIsNotAnotherAnswer` だけが落ちる(82 本のうち 1 本) | `prompts_notice_test.go:445`: `` expected the 機能ごとの org item naming "4. (per-plan)" to say [...] in this order, but "4. (per-plan)`HOOK_SEAT:` のヘッダの付いた DECISION は、承認でも直しの依頼でもない(「運用規律」の節の codex の hook の項)ので、計画を直さず、承認か直しの依頼の DECISION を待ち続ける" is missing or out of order `` |
| g3(追加)`/org` の側だけヘッダの名前を変える | 4 写しの 542 行の `` `HOOK_SEAT: <座席 id>` `` を `` `HOOK_SEAT_ID: <座席 id>` `` にし、`orgSkillHookQuestionRules` の同じ断片(`prompts_notice_test.go:487`)も同じに変える(テストは `-overlay`)。director の skill、leader.md、ほかの断片はそのまま | 残る。`/org` skill を読む 88 本と `TestDirectorSkill_` 37 本(重なりを除いて上位 97 本)がすべて PASS | —(下の G5) |
| g3-0(g3 の対照)skill だけを変える | g3 と同じ 4 写し、テストはそのまま | 1 本が落ちる | `TestOrgSkill_InboxResolvesTheHookQuestionAtOnce`。`prompts_notice_test.go:523` が 4 回(写しごとに、`HOOK_SEAT:` を含む断片) |
| g4(追加)`/org` の側だけ決まった note を変える | 4 写しの 535 行の `--note '人に伝えた(hook の承認の知らせ)'` を `--note '人に伝えた(hook の知らせ)'` にし、`orgSkillHookQuestionRules` の同じ断片(`:480-481`)も同じに変える | 残る。上位 97 本がすべて PASS | —(下の G5) |
| g4-0(g4 の対照)skill だけを変える | g4 と同じ 4 写し、テストはそのまま | 1 本が落ちる | 同じテスト。`:523` が 4 回(写しごとに、note を含む断片) |

handoff の 4 つ (a)〜(d) はどれも落ち、落ちたのはどれも名指したテストの 1 本だけだった。(a) は 1 写しだけに当て、その写しの行だけが出た。テストは写しごとに確かめるので、`sync-skills.sh` を流し忘れた写しも捕まる(写しのずれは static の check-skill-sync も捕まえるが、/test では流していない)。(b) が落ちたのは、「QUESTION が名指した座席」から `HOOK_SEAT:` の句までを 1 つの断片で持つため(`prompts_notice_test.go:486-487`)である。(c) が落ちたのは、本文の 3 文を 1 つの断片で持つため(`:339-342`)で、3 文のどこを変えても落ちる。動機の文だけを固める断片は無いが、隙間ではない。(d) は cycle 2 の (c) と同じ所で、HEAD でも新しいテストだけが落ち、固定した断片の 3 ファイルのテストは通った。g3 と g4 は、verify の V3-5 が挙げた 3 つ目の面を実行で確かめるために流した。

### Coverage(cycle 3)

- Statement: 計測していない。この branch は本番の Go を変えていない(`git diff 7433ce84 HEAD` の Go はテスト 2 ファイルだけ)。木に入った #330 と #331 の本番の Go の coverage はそれぞれの PR の /test が見たもので、ここでは run-test.sh のパッケージ全体の ok までを確かめた
- Branch: 計測していない
- Function: 該当なし
- Notes: cycle 3 が足した文のうち、テストが読むのは、`/org` skill の hook の QUESTION の項の 12 断片と並びの 3 断片(4 写し)、leader の本文の 3 文(1 断片)と「director か人が」の文である。`/org` の項の変異は (a) を 1 写し、(b) を 4 写しに当て、当てた写しだけが落ちた

### Failure analysis(cycle 3)

| Test | Error | Root cause | Proposed fix |
| --- | --- | --- | --- |
| — | 失敗なし | — | — |

### Regression checks(cycle 3)

| Previously broken behavior | Status | Evidence |
| --- | --- | --- |
| 固定した断片を持つ既存のテスト(`prompts_test.go` 43 本、`prompts_leader_plan_test.go` 18 本、`prompts_xreview_leader_test.go` 21 本。`1d7eda3d` から変わっていない) | 通る | 名前を絞った org の回。#331 が leader.md に 12 行を足したあとも、手順 4 と cap の読み手の断片は元の順のまま見つかる |
| cycle 1 と cycle 2 のテスト(`prompts_notice_test.go` の 4 本と `director_notice_skill_test.go`) | 通る | 名前を絞った 2 回 |
| `TestDirectorSkill_*` 37 本、`TestRenderDirectorPrompt_*` 10 本、`TestRenderRolePrompt_Leader_*` 82 本、`TestOrgSkill_*` 30 本(新しい 1 本を含む) | 通る | 名前を絞った org の回 |
| 取り込んだ #330(`TestUsageLimit*` 8 本、`TestWatchUsageLimit_*` 14 本)と #331 を合わせた `internal/org` と `internal/cli` の全体 | 通る | 名前を絞った org の回と、run-test.sh の Go の段(2 パッケージとも `(cached)` なし) |

### Test gaps(cycle 3)

- G1、G2、G6、G7 は cycle 2 のまま成り立つ。AC7 のテストも、本番の 2 つの行を作る関数も変わっていない。G6 は `/org` skill の項にも当てはまり、項が人に送らせる `HOOK_SEAT:` の付いた DECISION を `protocol.Parse` と `Validate` に通すテストは無い
- G3(INFO)は `/org` skill の道にも広がった。cross-review の所見 3 の閉じ方は、人か session が `/org` skill の項か leader の本文に従って件をすぐ resolve することに掛かっており、見張りの `orgHasOpenLeaderItem` は変わっていない。どちらにも従わない人の件は、resolve されるまで leader の stall の知らせを止める。人や session が実際に従うかは単体テストでは確かめられない(verify の cycle 3 の Coverage gaps と同じ判断)
- G5(LOW)の面が 3 つになった。`HOOK_SEAT:` の字と決まった note `人に伝えた(hook の承認の知らせ)` を、テストは director の skill の側(`directorNoticeRules`)、leader の側(`leaderHookNotePhrases`、`leaderHookSeatApprovalSkip`、`leaderHookSeatCapSkip`)、`/org` skill の側(`orgSkillHookQuestionRules`)の 3 組の別々の文字列として持ち、3 組が同じであることはどのテストも確かめていない。cycle 2 の g は leader の側で、cycle 3 の g3(ヘッダ)と g4(note)は `/org` skill の側で、片側の文面とその側の文字列を一緒に変えるとすべてのテストを通ることを示した。g3 のとき、`/org` skill は人に `HOOK_SEAT_ID:` を送らせ、leader は `HOOK_SEAT:` を探す。対照 g3-0 と g4-0 は 1 本が落ちるので、片側の文面だけの変更は捕まる。verify の V3-5 と self-review の C3-7 と同じ穴で、tech-debt の G5 の行(cycle 3 の sync-docs で持ち越しの注を足す予定)に入る。閉じるなら、ヘッダと note を定数に 1 つずつ置いて 3 組の断片をそれで組み立てるか、3 つの文書から見つけたヘッダと note が同じことを確かめる 1 本を足す

## Verdict

- Pass: cycle 3(HEAD `ad590839`)の run-test.sh が rc 0(2,227 秒、20:39:15〜21:16:22 JST、1 分の load 7.12〜139.03。Go の段は 7.12〜36.12)。shell 45 本すべて OK(PASS を含む行 5,195)、SKIP 2 行は手元の環境による。Go 10 パッケージすべて ok: `internal/org` 1,102.6 秒(30 分の上限の 61%)、`internal/cli` 945.2 秒、`internal/config` 30.2 秒、`internal/insights` 3.2 秒、`cmd/ralph` 1.4 秒。`(cached)` の 5 パッケージ(`internal/gitroot`、`internal/org/driver`、`internal/org/protocol`、`internal/scaffold`、`internal/upgrade`)は `-count=1` の回し直しで ok。名前を絞った回は `internal/org` が PASS 450・SKIP 0(上位 200 本。handoff が名指したテスト、固定した断片の 3 ファイルの 82 本、#330 の 22 本を含む)、`internal/cli` が PASS 5・SKIP 0。handoff の変異 (a)〜(d) はすべて落ちた
- Fail: 0
- Blocked: 0
- Verdict: pass。G5 の面が 3 つになった(`/org` skill の側でもヘッダと note を別の文字列として持つ。片側の文面だけの変更は捕まるので LOW のまま)。G1、G2 と合わせて /pr の Known gaps に書くことを提案する

### Verdict(cycle 2)

- Pass: cycle 2(HEAD `1d7eda3d`)の run-test.sh が rc 0(2,164 秒、15:57:00〜16:33:04 JST、1 分の load 6.83〜44.89)。shell 45 本すべて OK(PASS を含む行 5,195)、SKIP 2 行は手元の環境による。Go 10 パッケージすべて ok: `internal/org` 1,093.4 秒、`internal/cli` 964.2 秒、`internal/config` 56.9 秒、`internal/insights` 3.3 秒、`cmd/ralph` 1.4 秒。`(cached)` の 5 パッケージ(`internal/gitroot`、`internal/org/driver`、`internal/org/protocol`、`internal/scaffold`、`internal/upgrade`)は `-count=1` の回し直しで ok。名前を絞った回は `internal/org` が PASS 251・SKIP 0(上位 146 本。handoff が名指したテストと、固定した断片の 3 ファイルの 82 本を含む)、`internal/cli` が PASS 5・SKIP 0。handoff の変異 (a)〜(e)(e は 2 通り)と追加の f はすべて落ちた
- Fail: 0
- Blocked: 0
- Verdict: pass。新しい G5(ヘッダの名前を skill と leader の側で別の文字列として持つ。片側の文面だけの変更は捕まるので LOW)を、cycle 1 の G1、G2 と合わせて /pr の Known gaps に書くことを提案する

### Verdict(cycle 1)

- Pass: run-test.sh rc 0(1,930 秒、13:35:26〜14:07:36 JST、1 分の load 5.44〜17.33)。shell 45 本すべて OK(PASS を含む行 5,195)、SKIP 2 行は手元の環境による。Go 10 パッケージすべて ok で `(cached)` なし: `internal/org` 1,067.4 秒、`internal/cli` 943.4 秒、`internal/config` 18.0 秒、`internal/upgrade` 9.0 秒、`internal/scaffold` 8.4 秒、`internal/org/driver` 8.0 秒、`internal/org/protocol` 7.6 秒、`internal/gitroot` 6.8 秒、`internal/insights` 3.7 秒、`cmd/ralph` 1.7 秒。名前を絞った回は `internal/org` が PASS 207・SKIP 0(上位 126 本)、`internal/cli` が PASS 5・SKIP 0。変異は handoff の (a)、(b)、(c) と本番の側の p1、p2 の 5 つがすべて落ちた
- Fail: 0
- Blocked: 0
- Verdict: pass。G1(行の形の断片の外を AC7 のテストが結び付けない。既存の文面のテストが本番の変更を捕まえるので LOW)と G2(受け入れ済みの L3)は、/pr の Known gaps に書くことを提案する
