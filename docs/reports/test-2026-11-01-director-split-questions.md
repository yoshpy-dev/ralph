# Test report: director-split-questions

- Date: 2026-11-01
- Plan: `docs/plans/active/2026-11-01-director-split-questions.md`(`## Test plan` と AC8、Progress の S1〜S7)。issue #271
- Tester: tester サブエージェント(`/test`、パイプラインの cycle 1。`cycle-count.json` は 1)
- Scope: `git diff origin/main...HEAD`(merge-base 4fef6b8c、HEAD 65beb971)。Go の変更はテストのファイル 4 つだけ(`internal/org/skill_split_questions_test.go` を新設、`director_skill_sync_test.go`・`skill_doc_followup_test.go`・`skill_followup_test.go` を更新)で、本体の Go は変わっていない。ほかは `/director` と `/org` の skill の写し 8 つ、証跡、計画、report、tech-debt。`./scripts/run-test.sh` は changed scope のまま打ち、Go の pack が選ばれた。test mode は `tests/test-*.sh` を glob で全部打つので、shell の 45 本も走った。この diff でテストの関数は 14 本増え、消えたものはない
- Evidence: `docs/evidence/test-2026-11-01-director-split-questions.log`(gitignore。`run-test.sh` の全出力と開始・終了の meta、変異の出力、変異の harness)。wrapper 自身の写しは `docs/evidence/verify-2026-10-31-221724.log`

## Test execution

| Suite / Command | Tests | Passed | Failed | Skipped | Duration |
| --- | --- | --- | --- | --- | --- |
| `./scripts/run-test.sh`(changed scope、rc 0) | shell 45 本 + Go 10 パッケージ | shell 45 本 OK(`    OK` の行 45)、Go 10 パッケージ ok(`(cached)` なし) | 0 | 1(`plan-visual` の「browser detection」の行。`/Applications/` にブラウザがある手元では打たない) | 1,590 秒(07:17:23〜07:43:53 JST)。load は開始 5.54 / 7.78 / 14.15、終了 5.14 / 7.00 / 9.06、途中の見本 7.74、11.89 |
| └ Go: `internal/org` | 1,412 本(トップレベル、`-test.list`) | ok | 0 | 0 | 743.7 秒 |
| └ Go: `internal/cli` | — | ok | 0 | 0 | 520.6 秒 |
| └ Go: `cmd/ralph`、`internal/{config,gitroot,insights,org/driver,org/protocol,scaffold,upgrade}` | — | ok | 0 | 0 | 1.1〜17.5 秒 |
| └ うち skill の固定(`TestDirectorSkill\|TestOrgSkill`) | 55 本(この diff で 14 本増えた) | 55 | 0 | 0 | 1.7 秒(写しの test binary で別に打った回) |
| 変異 20 個(`git archive HEAD` の写し。skill の文は 1 つの test binary で、Go を変えるものは作り直して) | 20 | 17 個が落ちた、3 個が残った | — | — | 1 個 2〜8 秒 |
| 残った変異を殺す行の試し 4 回(同じ写し) | 4 | 足した行だけの回は通り、3 つの変異はどれも落ちた | — | — | 1 個 6〜8 秒 |

起動の仕方: `run-test.sh` は 600 秒を超えるので、Bash tool の `run_in_background` は使わず、perl の fork と setsid で切り離した(INT・HUP・QUIT・TERM を既定に戻し、標準入出力は /dev/null)。runner が log と開始・終了・rc の meta を書き、こちらは前景の `caffeinate -w <pid>` で待った。shell の SKIP の行は前の回の報告と同じ 1 行だけだった。

Go の 30 分の時間切れはなく、既知の不安定な `TestStopDirectorProcess_AProcessThatIgnoresSIGTERMIsNotEndedInTime` もこの回は通った。打ち直しはしていない。変異の回は `run-test.sh` と同じ時間に動かしたが、写しの中だけで、worktree のファイルには触れていない。

## Coverage

- Statement: 計測の対象外。この diff の Go はテストのファイルだけで、`go test -cover` はテストのファイルを数えない。本体の Go は変わっていない
- Branch: 計測なし
- Function: テストの helper のうち、`directorSkillIntro` には直接の表のテスト(`TestDirectorSkillIntro`)がある。`directorSkillSection`(`director_skill_sync_test.go:223`)にはない(Test gaps の 2)
- Notes:
  - skill の文の固定の広さを、文の単位で数えた。差分で足した行にかかる director の skill の文 151 個のうち、テストの文字列に含まれない部分が 6 字を超える文は 26 個。そのほとんどは数え方の誤り(括弧の中の「。」で文を切った、`+` でつないだ断片の切れ目)で、本当に固定がないのは、「機能ごとの出力先」と「起票と記録」の冒頭の要約の 1 文ずつ、「分割計画の前の質問」の冒頭の「ここには要点と…」の 1 文、`main` から改行だけ変えて移した「start はその節を…」の 1 文。どれも手順や条件ではない
  - `/org` skill の足した 4 文は、すべて固定がある

## Failure analysis

| Test | Error | Root cause | Proposed fix |
| --- | --- | --- | --- |
| なし | — | — | — |

## Regression checks

| Previously broken behavior | Status | Evidence |
| --- | --- | --- |
| 既存の `TestDirectorSkill_*`、`TestOrgSkill_*`、`skill_doc_followup_test.go` の固定(Test plan の Regression tests) | pass | `internal/org` ok。`directorSkillSection` に替えた `TestDirectorSkill_GathersTheDocFollowupsIntoADocsOnlyFeatureAfterEveryMerge` は、helper を `###` で止まる形に戻す変異 G7 で落ちた(替えた理由のとおり) |
| `director_skill_sync_test.go` の `FORK` の問いの文(`directorSkillApprovalForkSteps`)を push の director に合わせた更新 | pass | `internal/org` ok。古い「AskUserQuestion(Codex では番号つきの選択肢)」を skill のどこに足しても落ちる(変異 M8) |
| 既知の不安定なテスト `TestStopDirectorProcess_AProcessThatIgnoresSIGTERMIsNotEndedInTime` | pass(打ち直しなし) | `run-test.sh` の 1 回目で通った |

## Mutation results

写しは `git archive HEAD` で scratch に作り、変異ごとにファイルを backup から戻した。全部のあとで、新しく取った `git archive HEAD` と `diff -rq` で比べ、差がないこと(rc 0)を確かめた。worktree は `git status --porcelain --ignored` で、無視されたログのほかに変化がない。

| # | 変異 | 結果 | 落としたテスト |
| --- | --- | --- | --- |
| M1 | 分割計画の承認の 4: 「Approve と答えたときだけ」→「答えたら」 | 落ちた | `TestDirectorSkill_AsksApproveOrNeedsChangesBeforeTheSplitPlanDigest` |
| M2 | 人への聞き方: pull だけの director も番号の選択肢で聞く | 落ちた | `TestDirectorSkill_AsksWithNumberedOptionsWhenTheWakePushes` |
| M3 | 出力先の 2: 重なる機能の推奨を「今すぐ着手」に | 落ちた | `TestDirectorSkill_RecommendsFilingAnIssueForAFeatureThatOverlapsARunningOrg` |
| M4 | 起票と記録の 4: 同じ follow-up が open なら「足さない」→「足す」 | 落ちた | `TestDirectorSkill_FilesTheIssueWithAMarkLineAndRecordsAFollowupPerBranch` |
| M5 | 「人の承認の言葉がないまま、この 2 行を書かない」を承認の 4 から 3 へ移す | 落ちた | `…AsksApproveOrNeedsChangesBeforeTheSplitPlanDigest`、`…GathersTheDocFollowupsIntoADocsOnlyFeatureAfterEveryMerge` |
| M6 | `templates/base/.agents` の写しだけで、merge を待つ issue の 3 の「印の行は本文に 1 行だけに保ち」を消す | 落ちた(その写しの名前だけが出た) | `TestDirectorSkill_MarksTheFiledIssueDoneOrRewritesItsSplitPlanBeforeStarting` |
| M7 | merge を待つ issue の 3 に「要らないなら `gh issue close <番号>` で閉じる」を足す | 落ちた | 同上(否定の確かめ) |
| M8 | `## 進捗の報告` に古い「AskUserQuestion(Codex では番号つきの選択肢)」を足す | 落ちた | `TestDirectorSkill_AsksThePlanApprovalForksWithNumberedOptionsWhenTheWakePushes` |
| M9 | 証跡 `docs/evidence/director-ask-herdr-2026-11-01.md` を消す | 落ちた | `TestDirectorSkill_AsksWithNumberedOptionsWhenTheWakePushes` |
| M10 | `/spec` の写し 1 つで「batches of five」→「four」 | 落ちた | `TestDirectorSkill_AsksTheSpecQuestionsBeforeWritingASplitPlan` |
| M11 | `### 人への聞き方` を `### 分割計画の前の質問` のあとへ移す | 落ちた | `TestDirectorSkill_SplitSectionHasTheAskingRuleThenTheFourSteps` |
| M12 | `/org` skill: 印の行を「note に書く」だけにする | 落ちた | `TestOrgSkill_FollowupSubsectionSaysTheDirectorRecordsAFiledFeature` |
| G1 | 印の行の正規表現のブランチの文字に `'` を足す(skill の 4 つの写しとテストの定数を同時に) | **残った** | なし |
| G2 | 同じく `;` を足す | 落ちた | `TestDirectorSkill_UsesOnlyAMarkLineOfTheShapeFromAnIssueTheDirectorFiled`(`after: none;` の行だけが殺した) |
| G3 | 同じく `$()` を足す | **残った** | なし |
| G4 | 同じく空白を足す(G1〜G3 の対照) | 落ちた | 同上(`feat/a b` の行) |
| G5 | 正規表現の末尾の `$` を消す | 落ちた | 同上 |
| G6 | `directorSkillSection` が `## ` で止まらず末尾まで読む | **残った** | なし |
| G7 | `directorSkillSection` が `###` でも止まる | 落ちた | `TestDirectorSkill_GathersTheDocFollowupsIntoADocsOnlyFeatureAfterEveryMerge` |
| G8 | 印の行を探す正規表現から `(?m)` を消す(skill の `gh` のコマンド 2 つ × 4 写しとテストの定数) | 落ちた | `TestDirectorSkill_TakeoverRecordsTheMissingFollowupsOfTheFiledIssues` |

残った 3 つは、写しで次の行を足すと落ちた。足した行だけの回(変異なし)は 56 本すべて通った。

| 試し | 写しに足したもの | 結果 |
| --- | --- | --- |
| R0 | 下の 4 行とテスト 1 本(変異なし) | 通った(偽の失敗なし) |
| R1 | 表に `after: feat/x'y`、`after: feat/x$y`、`after: feat/x(y)`、`after: feat/x;y` を、どれも false で足す + G1 | 落ちた(`feat/x'y` の行) |
| R2 | 同じ行 + G3 | 落ちた(`feat/x$y` と `feat/x(y)` の行) |
| R3 | `directorSkillSection` の表のテスト(`## a` の節が `### b` を含み `## c` の前で止まる、ない見出しは空)+ G6 | 落ちた |

## Test gaps

1. 印の行の正規表現の表で、shell に届く文字を拒む行は、どれも空白か別の欠けを一緒に持っている(`x';touch pwned;echo '`、`x;touch pwned`、`feat/x$(touch pwned)`)。空白だけで拒まれるので、ブランチの文字に `'` や `$()` を足しても表は通る(G1、G3)。skill は値を単一引用符で囲むので、`'` を許すと引用が切れる。self-review M1 の直しの要の性質なのに、テストがそれを固定していない。直すなら、欠けを 1 つだけ持つ行を表に足す(R1・R2 で効くことを確かめた 4 行)。重さは low〜medium
2. `directorSkillSection` に直接のテストがない(G6 が残った)。今の呼び出しは 1 か所で、そこで読む断片は新しいテストが手順の範囲で固定し直しているので、実害は小さい。`directorSkillIntro` と同じ形の表のテストを足せば殺せる(R3)。重さは low
3. 固定は「この文がある」の確かめなので、決まりと逆のことを別の場所に書き足す変異は、否定の確かめがある 2 か所(merge を待つ issue の 3 の `gh issue close`、skill 全体の古い Codex だけの文)のほかでは落ちない。断片の方式に元からある性質で、この diff で増えたものではない
4. skill のコマンドを本物で打つことはしていない。`gh` の `--jq` の `test("(?m)^ralph-split: ")` は gojq が Go の regexp で読む前提で、テストは Go の regexp で代わりに確かめている。director が分割計画から start までを通すことは計画の Non-goals
5. 静的の検査(AC8 の `check-skill-sync.sh` と `check-sync.sh`)は /verify の担当で、ここでは打っていない(verify の report で pass)

## Verdict

- Pass: 1,412 本の org を含む Go 10 パッケージと shell 45 本(`run-test.sh` rc 0、1,590 秒)。AC8 の `go test ./internal/org/` の部分を満たす。変異は 20 個中 17 個が落ちた
- Fail: 0
- Blocked: 0
- 残った変異 3 つは Test gaps の 1 と 2 に書いた。どちらもテストを強める提案で、今の skill の文の誤りではないので、/pr を止めない

## Re-run(S8 のあと)

- S8 8bc6ae84(テストの穴 1 と 2 を埋めた。`internal/org/skill_split_questions_test.go` の当たってはいけない行の表に shell に届く文字を 1 つだけ持つ行を 14 行、`internal/org/director_skill_sync_test.go` に `TestDirectorSkillSection` の 6 行の表)。変えたのはテストの 2 ファイルだけで、skill の文とコードは変えていない
- 打ったもの(HEAD 8bc6ae84、orchestrator): `test -z "$(gofmt -l internal/org)"`、`go vet ./internal/org/`(ok)、`golangci-lint run ./internal/org/`(0 issues)、`go test ./internal/org/ -run 'Skill' -count=1`(ok、`-v` の PASS の行 78)
- 変異(S8 の implementer、`git archive HEAD` の写し): G1・G3・G6 とその派生がすべて足したテストで落ちた(残った変異は 0)。HEAD の前のテストでは G1・G3・G6 が残ることも同じ写しで確かめた
- `run-test.sh` は打ち直していない。S8 はテストを足しただけで、ほかのテストの対象を変えないため。Verdict は上のまま
