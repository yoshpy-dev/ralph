# Test report: watch-usage-limit-retry

- Date: 2026-11-20(パイプラインの cycle 1。`cycle-count.json` は `cycle: 1`)
- Plan: `docs/plans/active/2026-11-20-watch-usage-limit-retry.md`(issue #353)
- Tester: tester サブエージェント(`/test`)
- Scope: HEAD `87693818`(base `1d05c9ac`、worktree は clean)。`./scripts/run-test.sh` を 1 回、worktree のルートから回した。Bash tool の 10 分の上限を超えるので、perl の fork と setsid で切り離して起動し(INT・HUP・QUIT・TERM は既定の扱いに戻し、標準入出力は `/dev/null`)、`caffeinate -w` で終わりを待った。範囲は wrapper の既定の changed(`- Requested scope: changed`)で、fixture の `.txt` が言語に分類されないため `Language scope: full fallback (unclassified:internal/org/testdata/resume/claude-weekly-limit-retrying.txt)` になった。この repo で見つかる言語は golang だけなので、走る pack は changed のときと同じ。そのあと、計画の AC のテストを 2 つの絞り込みで回し、verify の報告が勧めた変異 1 個、base のテストのファイルを HEAD のコードに当てる run(AC5)、狭い coverage の run を足した
- Evidence: `docs/evidence/test-2026-11-20-watch-usage-limit-retry.log`(9,346 行。`.gitignore` の `docs/evidence/*.log` に当たるので commit されない)。run-test.sh の出力、AC の 2 つの run の `-v` の出力、変異の diff と出力、base のテストの run の出力、coverage の要約が入っている。run-verify.sh は自分の evidence を `docs/evidence/verify-2026-11-20-083353.log` にも書いた(同じく追跡外)

## Test execution

| Suite / Command | Tests | Passed | Failed | Skipped | Duration |
| --- | --- | --- | --- | --- | --- |
| `./scripts/run-test.sh` | 下の 2 行の合計 | 全部 | 0 | 2(条件つきの skip、下の行) | 2026-11-20 17:33:52 〜 18:04:37 JST、1,845 s、rc 0。開始時の 1 分 load 28.3、終了時 8.3 |
| local verifier(`tests/test-*.sh` 46 本) | 46 本、`PASS` で始まる行 5,390 | 46 本(`OK` 46 行) | 0(`FAIL:` の要約行 29 本はどれも `FAIL: 0`) | `test-plan-visual.sh` の browser detection(`/Applications` にブラウザーがある)、`test-secret-scan-branch.sh` の「git 2.41 より古い本物の git」の場合(手元は git 2.49.0、stub の古い git の行は通る) | 約 18 分(17:33:52 〜 17:51:42)。この段の 1 分 load は 17:47 に 224 まで上がった(ほかのセッションの Xcode のビルド) |
| `go test ./...`(テストのある 11 パッケージ) | 11 | 11 | 0 | なし。`(cached)` は 0 | 約 13 分(17:51:42 〜 18:04:37)、下の表 |
| `go test -count=1 -v ./internal/org/ -run 'UsageLimit\|Resume\|WatchUsage'` | 69 関数、`--- PASS` 424 行(subtest を含む) | 全部 | 0 | 0 | 12 s(パッケージ 10.7 s) |
| `go test -count=1 -v ./internal/cli/ -run 'Resume\|OrgWatch'` | 70 関数、`--- PASS` 148 行 | 全部 | 0 | 0 | 80 s(パッケージ 78.7 s) |

cli の絞り込みは依頼では `-v` なしだったが、関数の数を数えるために `-v` を付けた。絞り込みと `-count=1` は依頼のとおり。

| パッケージ | 結果 | 時間 |
| --- | --- | --- |
| `cmd/ralph` | ok | 0.726 s |
| `internal/cli` | ok | 651.382 s |
| `internal/config` | ok | 17.137 s |
| `internal/gitroot` | ok | 3.477 s |
| `internal/insights` | ok | 2.419 s |
| `internal/org` | ok | 755.975 s |
| `internal/org/driver` | ok | 3.767 s |
| `internal/org/protocol` | ok | 2.360 s |
| `internal/scaffold` | ok | 2.173 s |
| `internal/toolchain` | ok | 20.798 s |
| `internal/upgrade` | ok | 3.244 s |

`internal/org` は golang pack の 1 パッケージの上限(30 分)の 42% で終わった。Go の段の 1 分 load は、17:53 から 18:04 までの 5 回の測りで 8.3〜19.9。私の絞った run(AC の run、変異、base のテスト、coverage)は 17:34〜17:44 の shell の段に重なり、Go の段には何も回していない。負荷に弱いと分かっているテストを含め、落ちたテストはなく、打ち直しもしていない。

### 計画のテストと AC の対応

| AC | テスト | 結果 |
| --- | --- | --- |
| AC1 | `TestUsageLimitScreen_Issue353RetryWaitIsAUsageLimit`、`TestUsageLimitScreen_EveryFixture/claude-weekly-limit-retrying.txt`、`TestClassifyResumeScreen_Fixtures/claude_retrying_until_its_weekly_limit_resets_is_waiting`(org) | PASS。issue の画面を元にした fixture を上限と読み、`⎿ You've hit your weekly limit · resets Nov 21 at 3pm (Asia/Tokyo)` を返す |
| AC2 | `TestWatchUsageLimit_AClaudeSeatRetryingUntilItsWeeklyLimitResetsIsUsageLimited`(org、implementer と leader の 2 subtest) | PASS。stall の ALERT を出さず、`usage_limited` の件 1 つ(`REASON: usage_limit`、`SEATS:`、本文の上限の行)を上げる |
| AC3 | `TestOrgResume_UsageLimitWait_TheReasonNamesTheLimit`(org、14 subtest)、`TestOrgResume_ASeatWaitingForItsUsageLimitToResetShowsTheLimit`(cli、`resume` と `resume --dry-run`) | PASS。14 subtest は、working と idle の再試行の画面、5 時間の上限の footer、pane の読みの失敗、tool の呼び出し中、上限で止まった画面(週と session)の 7 形を、dry run の真と偽で回す。cli は出力の行、pane ごとに読みが 1 回、send と `resumed` がないことを確かめる |
| AC4 | `TestUsageLimitScreen_RetryWaitShapes`(org、26 subtest) | PASS。AC4 の 7 形は verify の報告の対応のとおり、「an API retry status line alone」「a limit only before the last prompt, …」「a limit in the middle of a reply line …」「a limit in the turn above a turn-end status line, with no retry」(止まった読み方の対照は「a limit in the turn that is the stop, with no retry」)「a tool output that prints the limit, then a network retry with no time」(6 番目の形もこの行が覆う)「no prompt on the screen and a retry whose time differs」の行に当たる。codex の行も偽 |
| AC5 | 下の「base のテストを HEAD に当てる」、fixture の 3 つの表、`TestOrgResume_RetryingOrWorking_Waits` | PASS(注つき)。今あるテストの判定は変わらない。base のテストで落ちるのは、AC5 が名指す呼び出しの列の変更と、新しい fixture の行が base の表にないことの 2 種だけ |
| AC6 | 文書と `check-sync.sh`・`check-skill-sync.sh` | /test の範囲の外。verify の報告で 2 つの検査が通っている |

### 変異: 時刻の一致を外す(verify の報告の勧め)

`usageLimitRetryingTurn`(`internal/org/usage_limit_screen.go:172`)の `if strings.Contains(retry.text, "("+limit.reset+")") {` を `if true {` に替え、「手番の中に上限の文があり、入力欄のすぐ上が再試行の行なら上限と読む」形にした。変異の写しと overlay の JSON は `$TMPDIR` の下に置き、`go test -count=1 -vet=off -timeout 4m -overlay <json> -v ./internal/org/ -run 'UsageLimit|Resume|WatchUsage'` で回した。対照の HEAD は同じ絞り込みで rc 0(上の表の行)。

結果は rc 1(`build failed` は 0 件)で、`TestUsageLimitScreen_RetryWaitShapes` の 5 つの subtest が落ちた。

| subtest | 変異での戻り値 | 期待 |
| --- | --- | --- |
| a tool output that prints the limit, then a network retry with no time | `("⎿ You've hit your weekly limit · resets Nov 21 at 3pm", true)` | `("", false)` |
| a retry whose time differs from the limit's reset | `("⎿ You've hit your weekly limit · resets Nov 21 at 3pm (Asia/Tokyo)", true)` | `("", false)` |
| no prompt on the screen and a retry whose time differs | 同上 | `("", false)` |
| of two limits in the turn, the one whose reset the retry shows | 最後の上限(`Nov 28 at 9am`)を返す | 再試行の時刻と一致する `Opus limit`(`Nov 21 at 3pm`)の行 |
| of two limits in one block, the first, to the block's end | 同上 | ブロックの最初の上限からブロックの終わりまで |

計画が一番恐れた取り違え(tool の出力が引用した上限の文のあとの通信断の再試行を上限と読む)と、時刻が合わない画面は、テストで守られている。見張り(`TestWatchUsageLimit_*`)と resume(`TestOrgResume_*`)のテストはこの変異で 1 つも落ちなかった。どちらも `UsageLimitScreen` の答えをそのまま使うので、否定の形は `UsageLimitScreen` の単体の表だけが固めている(Test gaps の 1)。

### base のテストを HEAD に当てる(AC5)

この branch が変えた 6 つのテストのファイル(`internal/cli/org_resume_test.go`、`internal/org/resume_screens_test.go`、`resume_test.go`、`send_screen_test.go`、`usage_limit_screen_test.go`、`watch_usage_limit_test.go`)を base `1d05c9ac` の版に `-overlay` で差し替え、HEAD のコードに当てた。

- org(`-run 'UsageLimit|Resume|WatchUsage|ClassifySend|SendScreen|SendProbe|SendInputHolds|FindResumeInputBox'`): `--- PASS` 668 行、落ちたのは 3 つ
  - `TestOrgResume_RetryingOrWorking_Waits/working`: `herdr calls = ["get org-a_implementer" "read pane-1 60"], want ["get org-a_implementer"]`。AC5 が名指す意図した変更(working の座席の pane を 1 回読む)
  - `TestUsageLimitScreen_EveryFixture` と `TestClassifySendScreen_EveryFixture`: どちらも `fixture testdata/resume/claude-weekly-limit-retrying.txt has no expectation in …Wants` だけで、既存の fixture の subtest はすべて通る。新しい fixture をディレクトリから拾うのに、base の表にその行がないため。判定の変更ではない
- cli(`-run 'Resume'`): rc 0、`--- PASS` 16 行

今ある見張りのテスト(`watch_usage_limit_test.go` の base の 14 関数)と、止まった画面と 5 時間の上限の footer の読み方のテストは、base の版のまま通る。

## Coverage

- Statement: 狭い run の profile で測った(パッケージ全体の profile は取っていない)。`git diff -U0 1d05c9ac HEAD` で足された行に当たる文は、`usage_limit_screen.go` 51/52、`resume.go` 19/19、`internal/cli/org_resume.go` 2/2、`internal/cli/org_watch.go` 1/1。関数では `UsageLimitScreen`、`usageLimitTurnBlocks`、`usageLimitResetsIn`、`usageLimitResetTime`、`resumeLook`、`resumeWorkingReason`、`resumeTargetText` が 100%、`usageLimitRetryingTurn` が 94.1%
- Branch: Go の標準ツールでは測れない
- Function: 上の通り。変更のない関数の数字は狭い run のものなので載せない
- Notes: 実行されない 1 文は `usage_limit_screen.go:157-158`(入力欄が見つからないときの `return "", false`)。doc comment のとおり、`UsageLimitScreen` からはここに来ない(先に `ClassifyResumeScreen` が waiting と読んでいるので、入力欄はある)。直接呼ぶテストはない

## Failure analysis

| Test | Error | Root cause | Proposed fix |
| --- | --- | --- | --- |
| なし | — | — | — |

## Regression checks

| Previously broken behavior | Status | Evidence |
| --- | --- | --- |
| issue #353: 手番の中で再試行して週の上限の解除を待つ claude の座席が stall の件になり、`resume --dry-run` が `herdr reports working` と出す | 直った | AC2 と AC3 のテストが PASS |
| 通信断の再試行(2026-11-07 の evidence の 3 の形)を上限と読まない | 保たれている | 「an API retry status line alone」が PASS。時刻の一致を外す変異で、tool の出力の引用のあとの通信断の再試行の行が落ちる |
| 5 時間の上限の footer の待ち(#329)を上限と読む | 保たれている | `TestUsageLimitScreen_EveryFixture/claude-5h-limit-continuing.txt`、resume の「idle, continuing automatically when the 5-hour limit resets」の 2 行、base の `TestUsageLimitScreen_WaitingForALimitToReset` が PASS |
| herdr が working で上限で止まった画面の座席は、理由が `herdr reports working` のまま(self-review M1) | 保たれている | 「working, the screen stopped on the weekly limit」と「… session limit」の 4 行が PASS |
| working の座席の pane の読みが失敗したら、理由は `herdr reports working` のまま | 保たれている | 「working, the pane read fails」の 2 行が PASS |
| 上限と読めない画面の座席は今までどおり stall になる | 保たれている | base の `TestWatchUsageLimit_ASeatWhoseScreenIsNoUsageLimitStallsAsBefore` と `…ASeatWhoseReplyQuotesTheWaitKeepsItsStall` が PASS |
| codex の読み方 | 変わらない | 「a codex limit and a retry that shows its reset time」が偽、fixture の表の codex の行と base の表がすべて PASS |

## Test gaps

1. 否定の形(通信断の再試行、時刻が合わない画面)は `UsageLimitScreen` の単体の表だけが固める。見張りと resume のテストは肯定の形しか持たないので、時刻の一致を外す変異でも落ちない。両方とも `UsageLimitScreen` の答えをそのまま使うので、呼び出し側で否定の形を足しても新しい分岐には届かない。今のままでよいと考える(LOW)
2. 実機の画面の全体はない。fixture は issue の断片と別の fixture の入力欄と footer の組み立てで、再試行の行の括弧の時刻と上限のリセットの時刻が同じことは 1 つの画面でしか見ていない(計画の Assumptions、register 515 行)。形の違う版では今までどおり stall になる
3. 再試行が解除の時刻より先に尽きたあとの画面と、そのときの resume の出し方は試していない(verify の報告の INFO の 3 つ目)
4. 待ちのあいだ `herdr agent get` の出力が変わらないか(見張りの道が stall の立つ座席でしか走らないこと)は、issue の報告から推しただけで、テストでも実機でも見ていない(verify の V4 の 516 行)
5. verify で直した `ralph org watch --help` の新しい句を固めるテストはない(register 517 行の (d))。help の文の変更で、挙動は変わらない
6. `usageLimitRetryingTurn` の入力欄が見つからないときの return は、どのテストからも実行されない(Coverage の Notes)

## Verdict

- Pass: `./scripts/run-test.sh` rc 0(shell 46 本、Go 11 パッケージ)、AC の 2 つの絞り込み(org 69 関数、cli 70 関数)、時刻の一致を外す変異が 5 つの subtest で落ちること、base のテストを HEAD に当てて落ちるのが意図した変更と新しい fixture の行の不足だけであること
- Fail: 0
- Blocked: 0
- Verdict: pass。/sync-docs に進んでよい
