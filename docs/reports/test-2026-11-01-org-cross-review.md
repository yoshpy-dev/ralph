# Test report: org-cross-review

- Date: 2026-11-01
- Plan: `docs/plans/active/2026-11-01-org-cross-review.md`(`## Test plan` と AC1〜AC9、Progress の S1〜S5 と F1〜F5)。Refs #269
- Tester: tester サブエージェント(`/test`、pipeline cycle 1。`cycle-count.json` の cycle は 1)
- Scope: `git diff origin/main...HEAD`(base 4fef6b8c、HEAD 15da7eb0)。Go は `internal/org`(雛形 `prompts/reviewer.md` と `prompts/leader.md`、新しいテスト 3 ファイル)、`internal/org/driver`(警告の文)、`internal/cli`(警告のテスト)、`internal/config`(コメント)。shell は `scripts/xreview-helpers.sh`(コメント)。ほかに skill とその写し、文書。テストの関数はこの diff で 38 本増え、消えたものはない。テストの途中(10:10 JST)に afccca76 がブランチに入ったが、変えたのは `docs/evidence/xreview-live-probe-2026-11-01.md` だけで、このファイルを読むテストはない(`grep -rn xreview-live-probe internal/ tests/ scripts/` が 0 件)。結果は afccca76 のコードにもそのまま当てはまる
- Evidence: `docs/evidence/test-2026-11-01-org-cross-review.md`(要点。commit する)、`docs/evidence/test-2026-11-01-org-cross-review.log`(gitignore。全出力と変異の差分)。wrapper 自身の写しは `docs/evidence/verify-2026-11-01-005214.log`(gitignore)

## Test execution

| Suite / Command | Tests | Passed | Failed | Skipped | Duration |
| --- | --- | --- | --- | --- | --- |
| `./scripts/run-test.sh`(changed scope、rc 0) | shell 45 本 + Go 10 パッケージ | shell 45 本 OK(`PASS:` の行 2,590)、Go 10 パッケージ ok(`(cached)` なし) | 0 | 2 行(host の条件: plan-visual の browser 検出、git 2.41 より古い実物の git) | 1,830 秒(09:52:13〜10:22:43 JST)。load は開始 7.12 / 7.06、終了 7.34 / 8.45 |
| └ Go: `internal/org` | — | ok | 0 | 0 | 968.3 秒 |
| └ Go: `internal/cli` | — | ok | 0 | 0 | 728.9 秒 |
| └ Go: `cmd/ralph`、`internal/{config,gitroot,insights,org/driver,org/protocol,scaffold,upgrade}` | — | ok | 0 | 0 | 1.7〜23.0 秒 |
| `go test -race -count=3 ./internal/org/ -run Xreview -v` | 36 の名前 × 3(サブテスト 72 × 3) | トップレベル 108、サブテスト 216 | 0 | 0 | 238.0 秒。DATA RACE 0 |
| `sh tests/test-xreview-helpers.sh` | 53 | 53 | 0 | 0 | 33 秒 |
| `bash tests/test-xreview-helpers.sh`(3.2.57) | 53 | 53 | 0 | 0 | 28 秒 |
| `zsh --emulate sh -f tests/test-xreview-helpers.sh`(5.9) | 53 | 53 | 0 | 0 | 32 秒 |
| `go test -count=1 ./internal/org/driver/ -run PaneEnv -v` | 13 | 13 | 0 | 0 | 1 秒 |
| `go test -count=1 ./internal/cli/ -run 'TestOrgVerbs_HerdrWithoutEnvFlagWarnsOnStderr\|TestOrgRuntimes_HerdrPaneEnvIsSeatPaneEnv' -v` | 2 | 2 | 0 | 0 | 8 秒 |
| 変異 9 個(`go test -overlay`、`-run Xreview`) | 9 | 9 個とも落ちた | — | — | 1 個 79〜148 秒 |

起動の仕方: `run-test.sh` は 600 秒を超えるので、Bash tool の `run_in_background` は使わず、perl の fork と setsid で切り離した(INT・HUP・QUIT・TERM を既定に戻し、標準入出力は /dev/null)。runner が log と開始・終了・rc の meta を書き、こちらは前景の `caffeinate -w <pid>` で待った。`-race` の回と変異の回は同じ形で切り離し、`run-test.sh` と並べて打った。その間の load は 6.5〜14.6 で、時間切れも打ち直しもない。`scripts/xreview-helpers.sh` が分類の外なので、scope は changed のまま full fallback になり、`go test ./...` が全パッケージを打った。

スタブで走らせるテストは、どれも bash(3.2.57)と `zsh -f`(5.9)の両方で打つ(`advisoryShells`)。対話の shell では打っていない。本物の claude と codex は呼んでいない(スタブが PATH の先頭にあり、`RALPH_` と `GIT_` の環境変数は外してある)。

### Test plan との対応

| Test plan の項目 | テスト |
| --- | --- |
| reviewer の xreview の節の文(ミッション 1 の除外、report 名、返事の形、skipped と incomplete) | `TestRenderRolePrompt_Reviewer_XreviewTaskIsNotAReviewTask`、`…ReportContractNamesTheReport`、`…AnswersBlockedOnlyForActionRequired`、`…SortsTheRunByItsOutput`、`…WritesTheReportUncommitted` |
| leader の手順 9 の順(commit → fetch → rebase → cross-review → report のコミット → sync-docs) | `TestRenderRolePrompt_Leader_XreviewSitsBetweenTheRebaseAndTheSyncDocs` |
| `### cross-review` の小節(終了コードの分岐、回数の定義、cap のファイル、QUESTION、PR の本文の行) | `…Leader_XreviewReadsTheSwitchWhereItIsUsed`、`…CountsOnlyCompleteReports`、`…ChecksTheCapBeforeTheTask`、`…CapQuestionIsAFork`、`…GoesInThePRBodyAndTheResult` |
| reviewer のコマンドのスタブ: 両方向 | `…Reviewer_XreviewCallsTheOtherModel`(implementer が claude なら codex、codex なら claude。`RALPH_PRIMARY_CLI` を相手の CLI にしても引数で選ぶ) |
| CLI がない、alias だけの zsh | `…Reviewer_XreviewFindsNoCLI` |
| 出力が空、失敗、log の行の 200 字の切り詰めと `~`・`.` への置き換え、watchdog、git の状態の変化、MCP が残る、読めない base、option に見える base | `…Reviewer_XreviewCommandOutcomes` |
| driver が claude と codex 以外、helper がない | `…Reviewer_XreviewPickRefusesABadDriverOrNoHelpers` |
| leader の状態のコマンドのスタブ: switch と cap と cap のファイル | `…Leader_XreviewStateCommandReadsTheSwitchAndTheCap` |
| org をまたいだ report の数え方、このブランチが足したものだけ | `…CountsTheReviewersReports`、`…CountsOnlyThisBranchsReports` |
| base の決め方 | `…FindsStep9sBase` |
| 上げた上限 | `…RaisedCapReachesTheTask` |
| `since:` と rebase、object が消えたとき | `…SeesARebaseAsAChange` |
| HEAD・REPORT・BASE の確かめ | `…ChecksTheReportsHead`、`…ChecksTheValuesItReads`(leader)、`…CommandOutcomes` の option に見える base(reviewer) |
| skill の文 | `internal/org/skill_xreview_test.go` の 7 本 |
| paneenv の警告の文 | `TestHerdr_TabCreate_PaneEnvWarningNamesEachEntrysConsequence`、`TestOrgVerbs_HerdrWithoutEnvFlagWarnsOnStderr` |

`-run Xreview` に掛からない新しいテストは 2 本(`TestDirectorSkill_ResultItemReportsTheCrossReviewLine`、`TestOrgSkill_PrerequisitesSayTheLeaderReadsTheReviewSettings`)で、どちらも文の固定。`run-test.sh` の回で通ったが、`-race -count=3` には入っていない。

### 変異

`go test -overlay` で go:embed の雛形を scratch の写しに差し替えた。写しは perl で文字列を 1 か所だけ置き換えて作り、worktree のファイルは変えていない。変異の前後の `git status --porcelain --ignored` は、go のビルドキャッシュと wrapper の log 1 本のほかに違いがない。

| ID | 変異 | 落としたテスト |
| --- | --- | --- |
| M1a | reviewer が implementer と同じ系列を選ぶ(引数を落とす): reviewer.md:196 `pick_reviewer "<IMPLEMENTER_DRIVER>"` → `pick_reviewer` | `…Reviewer_XreviewCallsTheOtherModel` の 4 行(implementer の CLI を呼んだ)、`…FindsNoCLI` の 4 行、`…PicksByTheImplementersDriver` |
| M1b | 同じ(逆にする): `r=<IMPLEMENTER_DRIVER>` | `…CallsTheOtherModel` の 4 行、`…FindsNoCLI` の 10 行すべて、`…PicksByTheImplementersDriver` |
| M2a | switch が off でも TASK を送る(コマンド): leader.md:799 `0:false)` で `switch: on` を出す | `…StateCommandReadsTheSwitchAndTheCap` の off の行、`…ReadsTheSwitchWhereItIsUsed` |
| M2b | 同じ(文): leader.md:818 「TASK を送らず」→「TASK を送り」 | `…ReadsTheSwitchWhereItIsUsed` だけ |
| M3 | cap のファイルを読まない: leader.md:802 `f=$(head -n 1 "$p-cap" …)` → `f=` | `…ReadsTheSwitchAndTheCap`(cap 3、want 5)、`…RaisedCapReachesTheTask` |
| M4a | ブランチが足していない report を数える(書き換えたものも): leader.md:804 `--diff-filter=A` を外す | `…CountsOnlyThisBranchsReports`(done 2、want 1) |
| M4b | 同じ(追跡している report をすべて): leader.md:808 `"$p-reports.txt"` → `"$p-tracked.txt"` | `…CountsOnlyThisBranchsReports`(done 2、want 0)、`…FindsStep9sBase` |
| M5 | HEAD の確かめを外す: leader.md:810 16 進の `grep -qE` を外す | `…ChecksTheReportsHead`(sha1・sha256 で `HEAD`、省略した SHA、大文字の SHA が since: same)、`…ChecksTheValuesItReads` |
| M6 | reviewer の claude の `git diff` の `--end-of-options` を外す: reviewer.md:285 | `…CommandOutcomes` の option に見える base(`--output=` のファイルが書かれた)、`…RunsTheOtherModelReadOnly` |

9 個とも落ちた。各変異の落ちた行と時間は evidence の要点にある。M2b のような文だけの変異は、文を固定するテストだけが落とす。leader は文を読むモデルなので、これは予想どおり。残りの 8 個は、雛形のコマンドをスタブで走らせる行が落とす。

## Coverage

- Statement: `internal/org/driver` はパッケージ全体で 95.3%。この diff で中身を変えた Go の関数は `paneEnvConsequence`(`internal/org/driver/paneenv.go:64`)だけで、100%。`spawn.go` と `config.go` はコメントだけ
- Branch: 計測なし(Go の道具は statement だけ)
- Function: 上のとおり
- Notes:
  - 変更の中心は Markdown の雛形で、statement の計測には出ない。雛形のコマンド(reviewer の手順 2 と 3、leader の状態のコマンド)はスタブで走らせる行が分岐を打ち、変異 8 個がそれを確かめた。モデルが読んで判断する文(reviewer の手順 1 と 4〜7、leader の送る・送らない・QUESTION の分け方)は文の固定だけ
  - `internal/org` と `internal/cli` のパッケージ全体の数字は、この回は取っていない

## Failure analysis

| Test | Error | Root cause | Proposed fix |
| --- | --- | --- | --- |
| なし | — | — | — |

## Regression checks

| Previously broken behavior | Status | Evidence |
| --- | --- | --- |
| 既存の雛形のテスト(手順の 1 行目、sync-docs の順、計画助言の小節の位置、gate の停滞の文。この diff で `prompts_test.go`、`prompts_leader_syncdocs_test.go`、`prompts_leader_plan_test.go`、`skill_doc_followup_test.go` を直した) | pass | `run-test.sh` の `internal/org` ok |
| 負荷で bash のジョブの通知が出力の行の間に入って落ちた 2 か所(S1c で行ごとの確かめに直した) | pass | `-race -count=3` の 3 回と `run-test.sh` の回で `…CommandOutcomes` と `…CallsTheOtherModel` が通った(load 6.5〜14.6) |
| `pick_reviewer` の大文字小文字と既定値(`tests/test-xreview-helpers.sh`) | pass | sh・bash・`zsh --emulate sh -f` で 53/53 |
| paneenv の警告に「座席の cross-review」の文が戻らないこと | pass | driver の 13 本、cli の 2 本 |

## Test gaps

1. leader が返事の `REPORT:` のパスを確かめてからコミットする手順(`internal/org/prompts/leader.md:996` からの段落。形と、TASK_ID の `c<k>` と同じ番号か)は文だけで、走らせるコマンドがない。固定しているのは `TestRenderRolePrompt_Leader_XreviewChecksTheValuesItReads` の文(`internal/org/prompts_xreview_leader_test.go:1172-1183`)だけ。M2b と同じく、文を変えても落とすのは文の固定だけになる
2. reviewer の手順 1 の値の確かめ(`internal/org/prompts/reviewer.md:165-183`。`XREVIEW_BASE:` の形、`REPORT:` の形、`REVIEWER_*` の文字)も文だけ。コマンドの側で効くのは、手順 2 の一重引用符の `case`(`reviewer.md:196`)と、claude の `git diff` の `--end-of-options`(`reviewer.md:285`)だけ。codex のコマンド(`reviewer.md:254`)は `--base "<XREVIEW_BASE>"` と `-m "<REVIEWER_CODEX のモデル>"` を二重引用符で埋め、コマンドの中では確かめない。スタブの行が codex のコマンドに渡すのは形の正しい値だけ(`internal/org/prompts_xreview_reviewer_test.go:707-716`)。手順 2 の `case` の行(`:977-979`)にも一重引用符を含む値の行はない。範囲の外なので、それ以上は探っていない
3. スタブで走らせるのは bash 3.2.57 と `zsh -f` 5.9 だけ(`advisoryShells`、`internal/org/prompts_leader_plan_test.go:1049-1064`)。dash や POSIX の sh では、leader の状態のコマンドも reviewer のコマンドも打っていない。座席はどちらも自分の CLI の shell(bash か zsh)で打つので、影響は小さい
4. leader が状態の値をもとに分ける部分(TASK を送るか、上限の QUESTION の 1 つ目か 2 つ目か、cap のファイルに書く数)はモデルの判断で、文の固定(`…ChecksTheCapBeforeTheTask`、`internal/org/prompts_xreview_leader_test.go:246-259`)だけ。`…RaisedCapReachesTheTask` は規則を手で当てた数を cap のファイルに書き、状態のコマンドがそれを読むことだけを確かめる(docstring のとおり)
5. 座席の中(権限、sandbox、network)での実行は /test の範囲の外で、計画の known gap(AC9、Risks)のまま。本物の CLI での実行は `docs/evidence/xreview-live-probe-2026-11-01.md` の 2 回(2 回目は afccca76 で 62d62be0 の版)にあり、この回では打っていない

## Verdict

- Verdict(cycle 1、HEAD 15da7eb0。afccca76 は evidence の文書だけ): pass
- Pass: `./scripts/run-test.sh`(rc 0、shell 45 本、Go 10 パッケージ)、`-race -count=3` の `Xreview`(DATA RACE 0)、`test-xreview-helpers.sh` の sh・bash・`zsh --emulate sh -f`、paneenv の警告のテスト、変異 9 個はすべて落ちた
- Fail: 0
- Blocked: 0
