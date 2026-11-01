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

## Cycle 2

- Date: 2026-11-01
- Tester: tester サブエージェント(`/test`、パイプラインの cycle 2。`cycle-count.json` は 2)
- Scope: `git diff 101e7552..HEAD`(HEAD d4fa10f3)。cross-review cycle 1 の直し S9 9248d73b・S10 e95f19e3 と、self-review cycle 2 の直し S11 80cb8b6b。Go はテストのファイル `internal/org/skill_split_questions_test.go` だけが変わった(断片、helper `directorSplitMarkIssuesCmd` の `--jq`、新しい定数 `directorSplitMarkIssuesOutput`)。テストの関数は増えも減りもしていない。ほかは `/director` skill の写し 4 つ、`docs/tech-debt/README.md` の 1 行、計画、report、insight event
- Evidence: `docs/evidence/test-2026-11-01-director-split-questions-c2.log`(gitignore。`run-test.sh` の全出力と開始・終了の meta、キャッシュなしの打ち直し、変異の結果と harness)。wrapper 自身の写しは `docs/evidence/verify-2026-11-01-000722.log`

### Test execution

| Suite / Command | Tests | Passed | Failed | Skipped | Duration |
| --- | --- | --- | --- | --- | --- |
| `./scripts/run-test.sh`(changed scope、rc 0) | shell 45 本 + Go 10 パッケージ | shell 45 本 OK(`    OK` の行 45)、Go 10 パッケージ ok(うち 8 つは `(cached)`) | 0 | 2 行(cycle 1 の log と同じ。`test-plan-visual.sh` の「browser detection」と、`test-secret-scan-branch.sh` の「real git older than 2.41」) | 1,447 秒(09:07:22〜09:31:29 JST)。load は開始 6.66 / 6.41 / 6.30、終了 4.84 / 5.62 / 6.18、途中の見本 7.76、8.30、5.45 |
| └ Go: `internal/org` | 1,413 本(トップレベル、`-test.list`。cycle 1 の 1,412 に S8 の `TestDirectorSkillSection` が足された) | ok | 0 | 0 | 710.3 秒 |
| └ Go: `internal/config` | — | ok | 0 | 0 | 10.4 秒 |
| └ Go: `cmd/ralph`、`internal/{cli,gitroot,insights,org/driver,org/protocol,scaffold,upgrade}` | — | ok `(cached)` | 0 | 0 | — |
| 上の 8 パッケージをキャッシュなしで打ち直し(`go test -count=1 -timeout=30m`) | 8 パッケージ | 8 ok | 0 | — | 483 秒(09:31:56〜09:39:59 JST)。`internal/cli` 479.6 秒 |
| うち skill の固定(`-test.run Skill`) | 57 本 | 57 | 0 | 0 | 写しの test binary で 2 秒前後 |
| 変異 16 個(`git archive HEAD` の写し、1 つの test binary) | 16 | 15 個が落ちた、1 個が残った(C15、見込みどおり) | — | — | 1 個 2 秒前後 |
| ブランチ数の上限の co-mutation と、それを殺す試し | 4 回 | 対照は落ちた、co-mutation は残った、試しの定数で落ちた | — | — | 再ビルド込みで 1 回 10 秒前後 |

起動の仕方は cycle 1 と同じ。`run-test.sh` は 600 秒を超えるので、Bash tool の `run_in_background` は使わず、perl の fork と setsid で切り離した(INT・HUP・QUIT・TERM を既定に戻し、標準入出力は /dev/null)。runner が log と開始・終了・rc の meta を書き、こちらは前景の `caffeinate -w <pid>` で待った。キャッシュなしの打ち直しも同じ形で切り離した。

`(cached)` が 8 つ出たのは、golang の verifier(`packs/languages/golang/verify.sh`)が `-count=1` を付けず、S9〜S11 がこの 8 つのテストの入力を変えていないため。`cmd/ralph` は `templates/` を埋め込むので、その結果は S11 の後と同じ内容で打たれた回のものになる。Go のキャッシュは test binary と、テストが開いたファイルの大きさと更新時刻が同じときだけ使われるが、それに頼らず `-count=1` で打ち直し、8 つとも ok だった。

Go の 30 分の時間切れはなく、既知の不安定な `TestStopDirectorProcess_AProcessThatIgnoresSIGTERMIsNotEndedInTime` もこの回は通った(`internal/org` ok)。打ち直しはしていない。変異の回は `run-test.sh` と同じ時間に動かしたが、写しの中だけで、worktree の追跡しているファイルには触れていない。

### Coverage

- Statement: 計測の対象外。この cycle の Go はテストのファイルだけで、本体の Go は変わっていない
- skill の文の固定の広さ: S9〜S11 が `.claude/skills/director/SKILL.md` に足した行を文に切り、空白と改行を除いてテストのファイルの文字列と比べた。足した文はすべて、どれかの断片に含まれる。含まれないと出たのは、`-U0` の差分が離れた行をつないだ切れ目と、コード片の行だけで、コード片は `directorSplitMarkIssuesCmd`(`gh issue list` の 2 つ)と step 3 の断片(`followup add` と `followup done` の 2 行、変異 C9 で確かめた)が固定している
- S11 の「`followup add` は gh を読まない」: `FollowupStore.Add`(`internal/org/followup.go:147`)は follow-up のファイルだけを読み、gh を呼ぶ口を持たない。manifest に `pr_merged` のあるブランチで `Add` が通ることは `internal/cli/status_followup_test.go:692` の行が打っている
- `--jq` の出力の形: 手元の jq 1.7.1 で、3 件のうち 2 件が行頭に印の行を持つ JSON に対して `3`、`3<TAB>alice`、`1<TAB>alice` を、空の一覧に対して `0` を出すことを確かめた。gh は gojq で読むので、同じ確かめは S9 が gh 2.102.0 でした。テストは式を打たない(テストの穴の 3)

### Failure analysis

| Test | Error | Root cause | Proposed fix |
| --- | --- | --- | --- |
| なし | — | — | — |

### Regression checks

| Previously broken behavior | Status | Evidence |
| --- | --- | --- |
| cross-review cycle 1 #3: 印の行を探す `gh issue list` が上限で切れても、印の行がないと決める | pass(直っている) | 古い `--jq`(件数を出さない形)に戻す変異 C1 が `TestDirectorSkill_TakeoverRecordsTheMissingFollowupsOfTheFiledIssues` で落ちた |
| cross-review cycle 1 #1: 引き継ぎで待つブランチがすべて merge 済みだと issue が浮かばない | pass | 「記録せず」を「記録し」にする C4、台帳に残す文を消す C5、一部だけ merge 済みの扱いを替える C6 が落ちた |
| cross-review cycle 1 #2: start の断りは最初の重なりだけを出すので、`after:` がほかのブランチを落とす | pass | 「全部数える」を「最初に重なった org だけ」にする C11、「待つブランチの全部に」記録するを「最初のブランチに」にする C12 が落ちた |
| S10 が直した「もう重ならなければ、書き換えずに start を打ち直す」 | pass | S10 の前の文に戻す C10 が落ちた |
| 既存の `TestDirectorSkill_*`、`TestOrgSkill_*` の固定 | pass | `internal/org` ok |
| 既知の不安定なテスト `TestStopDirectorProcess_AProcessThatIgnoresSIGTERMIsNotEndedInTime` | pass(打ち直しなし) | `run-test.sh` の 1 回目で通った |

### Mutation results

写しは `git archive HEAD` で scratch に作り、変異ごとに skill の写し 4 つ(と、co-mutation ではテストのファイル)を別の `git archive HEAD` から戻して `cmp` で確かめた。全部のあとで 2 つの写しを `diff -rq` で比べ、差がない。置き換えは perl の `index`/`substr` で、置き換える文が各ファイルに決めた回数だけあることを確かめてから書いた。断りのないものは skill の写し 4 つすべてを変え、落ちたテストは 4 つの写しの名前をすべて出した。

| # | 変異 | 結果 | 落としたテスト |
| --- | --- | --- | --- |
| C1 | 引き継ぎの `gh issue list --state open` を、件数を出さない古い `--jq` に戻す | 落ちた | `TestDirectorSkill_TakeoverRecordsTheMissingFollowupsOfTheFiledIssues` |
| C2 | 2 か所の「1 行目が `--limit` の値(1000)に届いたら」を「超えたら」に | 落ちた | 同上と `TestDirectorSkill_DocFollowupGatheringDoesNotWaitForAFiledFeature` |
| C3 | 引き継ぎの「印の行のある issue はもうないと決めず、`--limit` を上げて…人に言う」を「もうないと決めてよい」に | 落ちた | `…TakeoverRecordsTheMissingFollowupsOfTheFiledIssues` |
| C4 | 引き継ぎで、すべて merge 済みで follow-up のない issue を「記録せず」→「記録し」 | 落ちた | 同上 |
| C5 | 引き継ぎの「人の答えは「merge を待つ issue」の 3 のとおり台帳に残す」を消す | 落ちた | 同上 |
| C6 | 一部だけ merge 済みのとき「どれも記録しない」に | 落ちた | 同上 |
| C7 | merge を待つ issue の 3: follow-up を「`after:` のブランチの 1 つで」→「ブランチごとに」足す | 落ちた | `TestDirectorSkill_MarksTheFiledIssueDoneOrRewritesItsSplitPlanBeforeStarting` |
| C8 | 同じく「答える前は記録しない」を「聞く前に記録しておく」に | 落ちた | 同上 |
| C9 | 同じくコード片から `ralph org followup done <id> --note <決めたこと>` の行だけを消す | 落ちた | 同上(その行の断片) |
| C10 | 同じく、もう重ならないときの「人に伝えて start してよいかを聞く」を S10 の前の「start を打ち直す」に戻す | 落ちた | 同上 |
| C11 | 同じく「重なる org を全部数える」→「最初に重なった org だけを数える」 | 落ちた | 同上 |
| C12 | 同じく「待つブランチの全部に follow-up を記録する」→「最初のブランチに」 | 落ちた | 同上 |
| C13 | 起票と記録の 4 の「人の答えは…台帳に残す」を消す | 落ちた | `TestDirectorSkill_FilesTheIssueWithAMarkLineAndRecordsAFollowupPerBranch` |
| C14 | merge を待つ issue の 3 で「答える前は記録しない(…)」を段落の頭(「すべて done にする」の直後)へ移す | 落ちた | `…MarksTheFiledIssueDoneOrRewritesItsSplitPlanBeforeStarting`(順の確かめ) |
| C15 | merge を待つ issue の 3 の末尾に「follow-up が 1 件もない issue は台帳に残さない」を足す | **残った**(見込みどおり) | なし |
| C16 | `.agents` の写し 1 つだけで、文書の follow-up の 1 の「起票した機能はもうないと決めず」を「決め」に | 落ちた(その写しの名前だけが出た) | `TestDirectorSkill_DocFollowupGatheringDoesNotWaitForAFiledFeature` |

ブランチ数の上限(`--limit`)の値は、コマンドの中と、文の「(1000)」の 2 つに分かれている。その組み合わせを次の 4 回で確かめた。

| 試し | 写しに入れたもの | 結果 |
| --- | --- | --- |
| L1(対照) | skill の 2 つのコマンドだけを `--limit 500` に | 落ちた(`…Takeover…`、`…DocFollowupGathering…`。helper の文字列と合わない) |
| L1+L1T | skill の 2 つのコマンドと helper `directorSplitMarkIssuesCmd` を `--limit 500` に。文の「(1000)」は残す | **残った**(57 本すべて通った) |
| R0 | テストに定数 `directorSplitMarkIssuesLimit = "1000"` を置き、helper と 2 つの断片の「(1000)」をその定数から組む(変異なし) | 通った(偽の失敗なし、`gofmt -l` は空) |
| R1 | R0 + 定数を `"500"` + skill の 2 つのコマンドを `--limit 500` に | 落ちた(2 本。断片「の値(500)に届いたら」が見つからない) |

### Test gaps

1. 上限の値の連動がない(L1+L1T が残った)。`directorSplitMarkIssuesCmd` の `--limit 1000` と、断片の「(1000)」は別々の文字列なので、コマンドの上限を skill と helper でそろえて変えても、文の値が古いまま通る。上限を 500 に下げて文が 1000 のままだと、500 件で切れた一覧を director が切れていないと読む。直すなら、定数 1 つから helper と 2 つの断片を組む(R0・R1 で効くことを確かめた)。重さは low
2. 決まりと逆のことを別の場所に書き足す変異(C15)は落ちない。cycle 1 のテストの穴の 3 と同じ、断片の方式に元からある性質で、この cycle で増えたものではない
3. `--jq` の式を打つテストはない。「1 行目は絞り込む前の件数」は文字列としてだけ固定されている。手元の jq 1.7.1 で形を確かめた(Coverage)。gh は gojq で読むので正規表現の流儀が違うが、行頭の `^` の読み方は同じ。cycle 1 のテストの穴の 4 と同じ
4. 静的の検査(AC8 の `check-skill-sync.sh` と `check-sync.sh`)は /verify の担当で、ここでは打っていない(cycle 2 の verify の report で pass)

### Verdict

- Pass: 1,413 本の org を含む Go 10 パッケージと shell 45 本(`run-test.sh` rc 0、1,447 秒)。`(cached)` だった 8 パッケージもキャッシュなしで ok(483 秒)。AC8 の `go test ./internal/org/` の部分を満たす。S9〜S11 の断片への変異は 16 個中 15 個が落ちた
- Fail: 0
- Blocked: 0
- 残った変異 C15 と co-mutation L1+L1T は Test gaps の 2 と 1 に書いた。どちらもテストを強める提案で、skill の文の誤りではないので、/pr を止めない

## Cycle 3

- Date: 2026-11-01
- Tester: tester サブエージェント(`/test`、パイプラインの 3 回目の run)
- Scope: `git diff 6f62affb..HEAD`(HEAD 7f4046a0)。cross-review cycle 2 の直し S12 e775792d と、self-review の 3 回目の直し S13 8a23af28。Go はテストのファイル `internal/org/skill_split_questions_test.go` だけが変わった(定数 `directorSplitDependsLineForm`・`directorSplitDependsLinePattern`、断片、バイト単位の確かめ 2 つ、正規表現の表 12 行)。テストの関数は増えも減りもしていない(`-test.list` で 1,413 本、`Skill` に当たるもの 57 本)。ほかは `/director` skill の写し 4 つ、`docs/tech-debt/README.md` の 2 行、計画、report、insight event
- Evidence: `docs/evidence/test-2026-11-01-director-split-questions-c3.log`(gitignore。`run-test.sh` の全出力と meta、キャッシュなしの打ち直し 2 回、単独の 20 回、変異の結果と harness)。wrapper 自身の写しは `docs/evidence/verify-2026-11-01-015331.log`

### Test execution

| Suite / Command | Tests | Passed | Failed | Skipped | Duration |
| --- | --- | --- | --- | --- | --- |
| `./scripts/run-test.sh`(changed scope、**rc 1**) | shell 45 本 + Go 10 パッケージ | shell 45 本 OK(`    OK` の行 45)、Go 9 パッケージ ok(うち 8 つは `(cached)`) | 1(`internal/org` の `TestRunWatcher_TimeoutIndependentOfSmallInterval`) | 2 行(cycle 1・2 と同じ。`test-plan-visual.sh` の「browser detection」と、`test-secret-scan-branch.sh` の「real git older than 2.41」) | 2,047 秒(10:53:31〜11:27:38 JST)。load は開始 5.41 / 6.57 / 6.81、終了 9.60 / 10.15 / 9.18、途中の見本 9.08、9.47、10.64、9.88 |
| └ Go: `internal/org` | 1,413 本 | 1,412 | 1 | 0 | 1,193.7 秒(cycle 2 は 710.3 秒) |
| └ Go: `internal/config` | — | ok | 0 | 0 | 28.7 秒 |
| └ Go: `cmd/ralph`、`internal/{cli,gitroot,insights,org/driver,org/protocol,scaffold,upgrade}` | — | ok `(cached)` | 0 | 0 | — |
| 上の 8 パッケージをキャッシュなしで打ち直し(`go test -count=1 -timeout=30m`) | 8 パッケージ | 8 ok | 0 | — | 699 秒(11:28:30〜11:40:09 JST)。`internal/cli` 698.0 秒 |
| 落ちたテストだけを単独で(`git archive HEAD` の写しの test binary、`-test.count=20`) | 20 回 | 20 | 0 | — | 1 回 1.58〜3.02 秒(load 7.9〜8.8、上の打ち直しと同じ時間) |
| `internal/org` を 1 回だけキャッシュなしで打ち直し(`go test -count=1 -timeout=30m ./internal/org`) | 1,413 本 | ok | 0 | — | 734 秒(11:40:20〜11:52:34 JST)、パッケージ 732.5 秒。load 7.87 → 7.59 |
| 変異 29 個(`git archive HEAD` の写し。skill の文だけのものは 1 つの test binary、テストの定数も変えるものは作り直して) | 29 | 27 個が落ちた、2 個が残った(D21、E7。どちらも見込みどおり) | — | — | 1 個 2〜10 秒 |

起動の仕方は cycle 1・2 と同じ。`run-test.sh` と 2 つの打ち直しは 600 秒を超えるので、Bash tool の `run_in_background` は使わず、perl の fork と setsid で切り離した(INT・HUP・QUIT・TERM を既定に戻し、標準入出力は /dev/null)。runner が log と開始・終了・rc の meta を書き、こちらは前景の `caffeinate -w <pid>` で待った。打ち直しは 3 つとも順に打ち、重ねていない。

打ち直しの扱い: 落ちたテストは、指示で打ち直してよいとされた 2 つ(30 分の時間切れと `TestStopDirectorProcess_AProcessThatIgnoresSIGTERMIsNotEndedInTime`)のどちらでもない。そこで orchestrator に、`internal/org` をキャッシュなしで 1 回だけ打ち直す案を送り、異論がなければその案で進めると伝えた。返事はなく、案のとおり 1 回だけ打ち直した。`run-test.sh` 全体は打ち直していない。shell の 45 本と `internal/config` は 1 回目で通っており、cached の 8 パッケージと `internal/org` はキャッシュなしで打ち直したので、Go の 10 パッケージはすべて、この HEAD でキャッシュを使わずに通った。

`(cached)` が 8 つ出たのは cycle 2 と同じ理由で、golang の verifier が `-count=1` を付けず、S12・S13 がこの 8 つのテストの入力を変えていないため。

### Failure analysis

| Test | Error | Root cause | Proposed fix |
| --- | --- | --- | --- |
| `TestRunWatcher_TimeoutIndependentOfSmallInterval`(`internal/org/watcher_test.go:311`) | `org: watcher: invoke claude: claude: timed out: context deadline exceeded`(5.00 秒) | 負荷による時間切れで、この diff とは関係ない。stub の claude は `sleep 1` のあと JSON を返し、テストは `watcherInvokeTimeout` を 5 秒にして待つ。単独で 20 回打つと 1.58〜3.02 秒かかり、`sleep 1` に 0.6〜2 秒が足されている。`internal/org` が cycle 2 の 1.7 倍の 1,193.7 秒かかった回で、5 秒を超えた。S12・S13 は Go の本体も `watcher*.go` も変えていない(`git log` で `watcher.go`・`watcher_test.go` の最後の変更は 33f6383b)。同じ HEAD の `internal/org` の打ち直しと、単独の 20 回はすべて通った | この PR では直さない。`docs/tech-debt/README.md` にこのテストの行はない(`TimeoutIndependentOfSmallInterval` で 0 件)。tester の記録では、このテストは前にも並列の負荷の下で落ち、2026-08-03 と 2026-09-18 に単独では通ることを確かめている。行を足すか、`watcherInvokeTimeout` を 5 秒より長くする(テストが確かめたいのは「interval の 1 秒に縛られない」ことなので、15 秒でも意図は変わらない)のどちらかを orchestrator に任せる |

### Coverage

- Statement: 計測の対象外。この run の Go はテストのファイルだけで、本体の Go は変わっていない
- skill の文の固定の広さ: S12・S13 が `.claude/skills/director/SKILL.md` に足した行を文に切り、空白と改行を除いてテストのファイルの文字列と比べた(cycle 2 と同じ `pinned.pl`)。切り出した 26 個(差分の切れ目の切れ端を含む)のうち 24 個はどれかの断片にそのまま含まれる。含まれないと出た 2 つは、`頼られる issue: #<番号>` を定数 `directorSplitDependsLineForm` からつないで組む断片(変異 D1・D8 で固定を確かめた)と、正規表現のコード片の行が次の文とつながった切れ目(コード片は D4〜D7、続く文は前からある断片)で、固定の漏れではない
- 写しごとの確かめ: 1 つの写しだけを変える D3(`.agents`)と D5(`templates/base/.claude`)は、その写しの名前だけを出して落ちた。4 つの写しをそれぞれ読んでいる

### Regression checks

| Previously broken behavior | Status | Evidence |
| --- | --- | --- |
| cross-review cycle 2 #1: 頼られる issue がまだ open でも「着手する」を推奨する | pass(直っている) | `OPEN` を `CLOSED` にする D12、「全部書く」を「1 つ書く」にする D11 が `…AsksWhetherToStartAFiledIssueOnceEveryBranchItWaitsForIsMerged` で落ちた |
| cross-review cycle 2 #2: 引き継ぎが前の待ちのブランチの done の件を数える | pass | 「今の `after:` のブランチの件だけ」を「すべて」にする D13、「数えない」を「数える」にする D14 が `…TakeoverRecordsTheMissingFollowupsOfTheFiledIssues` で落ちた |
| cross-review cycle 2 #3: done を打つ途中で止まった issue を引き継ぎが聞き直す、着手の org を確かめない | pass | D15(聞き直す)、D18(`ralph status` の文を消す)が引き継ぎのテストで、D19(step 3 の「途中で止まったとき」を消す)が `…MarksTheFiledIssueDoneOrRewritesItsSplitPlanBeforeStarting` で落ちた |
| self-review C3-1: `- Depends on:` の slug(`oauth2` など)から番号を拾う | pass | D10(「からも番号を拾う」)、D9(CR を取る文を消す)が落ちた。正規表現の形は E1〜E5・E8 が表の行で落ちた |
| self-review C3-3: step 3 が前の待ちの follow-up を「ある」と数える | pass | D20(「今の `after:` のブランチの」を消す)が落ちた |
| self-review C3-4: `'` を含む `DONE_NOTE` をそのまま写す | pass | D16(二重引用符)、D17(「そのまま写し」)が落ちた |
| 既存の `TestDirectorSkill_*`、`TestOrgSkill_*` の固定 | pass | `internal/org` の打ち直しで ok、`Skill` の 57 本は写しの test binary でも通った |

### Mutation results

写しは `git archive HEAD` で scratch に 2 つ作り、変異ごとに skill の写し 4 つとテストのファイルを、変えていない方から戻して `cmp` で確かめた。全部のあとで 2 つの写しを `diff -rq` で比べ、差がない。置き換えは cycle 2 の `apply.pl`(perl の `index`/`substr`、決めた回数だけあることを確かめてから書く)に、置き換えの前後が同じ変異を断る行を足して使った。D5 の 1 回目は、変異の入力ファイルを書いたときに行末の空白が落ち、前後が同じで何も変えずに通った。入力を直して打ち直した結果を下に書いた。断りのないものは skill の写し 4 つすべてを変えた。

| # | 変異 | 結果 | 落としたテスト(行) |
| --- | --- | --- | --- |
| D1 | 起票と記録の 2 の `頼られる issue: #<番号>` から `#` を消す | 落ちた | `…FilesTheIssueWithAMarkLineAndRecordsAFollowupPerBranch`(断片 :417 とバイトの確かめ :478) |
| D2 | 同じ形の `: #` を `:#` に(空白だけ) | 落ちた | 同上(:478 だけ。断片は空白を無視するので、足したバイトの確かめだけが落とす) |
| D3 | D2 を `.agents` の写し 1 つだけに | 落ちた | 同上(その写しの名前だけ) |
| D4 | 正規表現のコード片の行の `: #` を `:#` に | 落ちた | `…AsksWhetherToStartAFiledIssueOnceEveryBranchItWaitsForIsMerged`(行の確かめ :772 だけ) |
| D5 | 同じ行の末尾に空白を 1 つ、`templates/base/.claude` の写しだけに | 落ちた | 同上(:772、その写しの名前だけ) |
| D6 | 同じ行の頭に「正規表現 」を足す | 落ちた | 同上(:772 だけ。断片は含まれるので通る) |
| D7 | 同じ行の `$` を消す | 落ちた | 同上(断片 :723 と :772) |
| D8 | 「2 つ以上なら 1 つ 1 行」を「1 行にまとめる」に | 落ちた | `…FilesTheIssue…`(:417) |
| D9 | 「(行末の CR は取ってから確かめる)」を消す | 落ちた | `…AsksWhetherToStart…`(:723) |
| D10 | `- Depends on:` の行「からは番号を拾わない」を「からも番号を拾う」に | 落ちた | 同上 |
| D11 | 「open の頼られる issue を全部書く」を「1 つ書く」に | 落ちた | 同上 |
| D12 | `gh issue view` の結果の `OPEN` を `CLOSED` に | 落ちた | 同上 |
| D13 | 引き継ぎで数える件を「`AFTER_BRANCH` の列によらず、その issue の件すべて」に | 落ちた | `…TakeoverRecordsTheMissingFollowupsOfTheFiledIssues`(:637) |
| D14 | 前の待ちの done の件を「数えない」を「数える」に | 落ちた | 同上 |
| D15 | 「(着手は聞き直さない)」を「(着手を聞き直す)」に | 落ちた | 同上 |
| D16 | `--note '<DONE_NOTE>'` を二重引用符に | 落ちた | 同上 |
| D17 | `'` を含む `DONE_NOTE` を「写さず」を「そのまま写し」に | 落ちた | 同上 |
| D18 | 「答えが「着手する」なら…`ralph status` で確かめ…聞く。」を消す | 落ちた | 同上 |
| D19 | merge を待つ issue の 3 の「途中で止まったときは、引き継ぎが…done にする。」を消す | 落ちた | `…MarksTheFiledIssueDone…`(:841) |
| D20 | 同じく「今の `after:` のブランチの follow-up が 1 件もないとき」の「今の `after:` のブランチの」を消す | 落ちた | 同上 |
| D21 | 「からは番号を拾わない。」の直後に「その行がなければ `- Depends on:` の数字を番号として使ってよい。」を足す | **残った**(見込みどおり) | なし |
| E1 | 正規表現の `$` を消す(skill とテストの定数の両方) | 落ちた | `…AsksWhetherToStart…`(表 :799 の `#12;x`、`#12 #13`、`#12\r` の 3 行) |
| E2 | `^` を消す(両方) | 落ちた | 同上(行頭の空白の行) |
| E3 | `[0-9]+` を `[0-9]*` に(両方) | 落ちた | 同上(`#` だけの行) |
| E4 | `$` を `\r?$` に(両方) | 落ちた | 同上(CR の行) |
| E5 | `[0-9]` を `[0-9０-９]` に(両方) | 落ちた | 同上(全角の数字の行) |
| E6 | 書く形を `頼られる issue #<番号>`(コロンなし)に(skill と `directorSplitDependsLineForm` の両方、正規表現はそのまま) | 落ちた | 同上(:778、書く形を正規表現が読めない) |
| E7 | `: #` を `:\s#` に(両方) | **残った**(見込みどおり) | なし |
| E8 | `#` を `#?` に(両方) | 落ちた | 同上(`#` のない行) |

E1〜E8 は skill とテストを一緒に変えたので、skill の文の固定(:723、:772)は通り、落としたのは正規表現の表と、書く形と正規表現のつながりの確かめだけだった。この 2 つが、定数を書き換えても意図を守っている。

### Test gaps

1. 決まりと逆のことを固定した文の後ろに書き足す変異(D21)は落ちない。cycle 1 のテストの穴の 3、cycle 2 の C15 と同じ、断片の方式に元からある性質で、この run で増えたものではない
2. 正規表現の表に、`:` と `#` のあいだの空白がタブなど別の空白の行がない(E7 が残った)。`\s` に替えても表の 12 行は同じ答えになる。足すなら `{"頼られる issue:\t#12", false}` の 1 行で E7 が落ちる(試していない)。重さは low。director が読む正規表現の文は D4 の行の確かめで固定されているので、skill の文が `\s` に替わることは別に止まる
3. 正規表現の使い方(行ごとに CR を取って当てる、番号を `gh issue view` に渡す)は文としてだけ固定されている。director はモデルで、文を読んで動くので、打つテストはない。cycle 2 のテストの穴の 3 と同じ種類
4. `TestRunWatcher_TimeoutIndependentOfSmallInterval` の時間切れに tech-debt の行がない(Failure analysis)
5. 静的の検査(AC8 の `check-skill-sync.sh` と `check-sync.sh`)は /verify の担当で、ここでは打っていない(3 回目の verify の report で pass)

### Verdict

- Pass: Go 10 パッケージ(1,413 本の org を含む)と shell 45 本。`run-test.sh` は rc 1(2,047 秒)で、落ちたのは負荷による時間切れの `TestRunWatcher_TimeoutIndependentOfSmallInterval` 1 本だけ。`internal/org` をキャッシュなしで 1 回だけ打ち直して ok(734 秒)、単独でも 20/20。`(cached)` だった 8 パッケージもキャッシュなしで ok(699 秒)。AC8 の `go test ./internal/org/` の部分を満たす。S12・S13 への変異は 29 個中 27 個が落ちた
- Fail: 0
- Blocked: 0
- 1 回目の rc 1 は、この diff が触れていないテストの時間切れで、同じ HEAD の打ち直しで通ったので Fail に数えない。tech-debt の行を足すかどうかは Test gaps の 4 として orchestrator に任せる。残った変異 D21 と E7 は Test gaps の 1 と 2 に書いた。どちらもテストを強める提案で、skill の文の誤りではないので、/pr を止めない

## Cycle 4

- Date: 2026-11-01
- Tester: tester サブエージェント(`/test`、パイプラインの 4 回目の run)
- Scope: `git diff 3edfa501..HEAD`(HEAD b05d6af6)。cross-review cycle 3 の直し S14 09c9f34d(起票の前に重なる org を数え直す、答えの記録が add と done のあいだで止まった件を拾う)と S15 ac5a0578(記録の直前に merge 済みのブランチを待ち終えたものに数える)。Go はテストのファイル `internal/org/skill_split_questions_test.go` だけが変わった(断片の追加と差し替え、step 2 から古い規則が消えたことの確かめ 1 つ、`t.Errorf` の文)。テストの関数は増えも減りもしていない(`-test.list` で 1,413 本、`Skill` に当たるもの 57 本)。ほかは `/director` skill の写し 4 つ(4 つとも同じバイト)、計画、self-review と verify の report、insight event
- Evidence: `docs/evidence/test-2026-11-01-director-split-questions-c4.log`(gitignore。`run-test.sh` の全出力と meta、キャッシュなしの打ち直し、変異の結果と harness)。wrapper 自身の写しは `docs/evidence/verify-2026-11-01-035810.log`

### Test execution

| Suite / Command | Tests | Passed | Failed | Skipped | Duration |
| --- | --- | --- | --- | --- | --- |
| `./scripts/run-test.sh`(changed scope、rc 0) | shell 45 本 + Go 10 パッケージ | shell 45 本 OK(`    OK` の行 45)、Go 10 パッケージ ok(うち 8 つは `(cached)`) | 0 | 2 行(前の run と同じ。`test-plan-visual.sh` の「browser detection」と、`test-secret-scan-branch.sh` の「real git older than 2.41」) | 1,625 秒(12:58:10〜13:25:15 JST)。load は開始 6.66 / 7.07 / 6.60、終了 6.32 / 6.58 / 6.85、途中の見本 8.40、6.07 |
| └ Go: `internal/org` | 1,413 本 | ok | 0 | 0 | 862.5 秒(cycle 3 は 1,193.7 秒) |
| └ Go: `internal/config` | — | ok | 0 | 0 | 19.8 秒 |
| └ Go: `cmd/ralph`、`internal/{cli,gitroot,insights,org/driver,org/protocol,scaffold,upgrade}` | — | ok `(cached)` | 0 | 0 | — |
| 上の 8 パッケージをキャッシュなしで打ち直し(`go test -count=1 -timeout=30m`) | 8 パッケージ | 8 ok | 0 | — | 503 秒(13:25:31〜13:33:54 JST)。`internal/cli` 501.8 秒 |
| 変異 20 個(`git archive HEAD` の写し、1 つの test binary で `-test.run Skill`) | 20 | 17 個が落ちた、3 個が残った(G16、G20、G21。どれも見込みどおり) | — | — | 1 個 1 秒未満(G1〜G19 の 18 個で 7.9 秒) |

起動の仕方は前の run と同じ。`run-test.sh` と打ち直しは 600 秒を超えうるので、Bash tool の `run_in_background` は使わず、perl の fork と setsid で切り離した(INT・HUP・QUIT・TERM を既定に戻し、標準入出力は /dev/null)。runner が log と開始・終了・rc の meta を書き、こちらは前景の `caffeinate -w <pid>` で待った。打ち直しは `run-test.sh` が終わってから打ち、重ねていない。変異は `run-test.sh` の走っているあいだに写しの上で打った(`Skill` の 57 本だけで、1 個 1 秒未満)。

`run-test.sh` は rc 0 で、30 分の時間切れも既知の不安定なテスト(`TestStopDirectorProcess_AProcessThatIgnoresSIGTERMIsNotEndedInTime`、`TestRunWatcher_TimeoutIndependentOfSmallInterval`)の失敗も起きなかったので、`internal/org` は打ち直していない。`(cached)` の 8 パッケージはキャッシュなしで打ち直して通ったので、Go の 10 パッケージはすべて、この HEAD でキャッシュを使わずに通った。8 つが `(cached)` になったのは前の run と同じ理由で、golang の verifier が `-count=1` を付けず、S14・S15 がこの 8 つのテストの入力を変えていないため。

### Failure analysis

落ちたテストはない。cycle 3 の Test gaps の 4(`TestRunWatcher_TimeoutIndependentOfSmallInterval` に tech-debt の行がない)は、573f5265 で `docs/tech-debt/README.md` に行が足され、閉じている。

### Coverage

- Statement: 計測の対象外。この run の Go はテストのファイルだけで、本体の Go は変わっていない
- skill の文の固定の広さ: S14・S15 が `.claude/skills/director/SKILL.md` に足した 31 行を文に切り、空白と改行を除いてテストのファイルの文字列と比べた(cycle 2 からの `pinned.pl`)。18 個のうち 15 個はどれかの断片にそのまま含まれる。含まれないと出た 3 つは差分の切れ目の切れ端で、固定の漏れではない。「1. 待つブランチを決める。」は断片が番号を除いた「待つブランチを決める」を持つ。残り 2 つは、変えていない行をはさんで別の文の頭とつながったもので、それぞれの文の全体は断片にある(G5、G11 で確かめた)
- 写しごとの確かめ: 1 つの写しだけを変える G15(`templates/base/.agents`)は、その写しの名前だけを出して落ちた。消えたことの確かめも 4 つの写しをそれぞれ読んでいる

### Regression checks

| Previously broken behavior | Status | Evidence |
| --- | --- | --- |
| cross-review cycle 3 #1: start の断りのあと、最初に重なった org だけで起票し、ほかの待つブランチを落とす | pass(直っている) | G1(出力先の 3 で数え直さない)が `…RecommendsFilingAnIssueForAFeatureThatOverlapsARunningOrg`(:356)で、G3(`ralph status` を `ralph org status` に)、G4(数え直しをエラーの org だけに)、G5(数え直しの文を `feature:` の行を読む文の後ろへ)が `…FilesTheIssueWithAMarkLineAndRecordsAFollowupPerBranch`(:424)で落ちた |
| cross-review cycle 3 #2: add のあと done の前に止まると、人が承認した着手が黙って失われる | pass | 引き継ぎの G6・G7・G8 が `…TakeoverRecordsTheMissingFollowupsOfTheFiledIssues`(:653)で、「merge を待つ issue」の 3 の G9 が `…MarksTheFiledIssueDoneOrRewritesItsSplitPlanBeforeStarting`(:882)で、2 の G10〜G13・G17〜G19 が `…AsksWhetherToStartAFiledIssueOnceEveryBranchItWaitsForIsMerged`(:749)で落ちた |
| 古い規則「`waiting` のブランチが残っていれば聞かない」が step 2 に戻る | pass | 置き換えの G13 は断片(:749)と消えたことの確かめ(:809)の両方で、足すだけの G14 は :809 だけで落ちた。G14 が落ちたので、消えたことの確かめは空振りしていない |
| 既存の `TestDirectorSkill_*`、`TestOrgSkill_*` の固定 | pass | `run-test.sh` の `internal/org` で ok、`Skill` の 57 本は写しの test binary でも通った |

### Mutation results

写しは `git archive HEAD` で scratch に 2 つ作り、変異ごとに skill の写し 4 つとテストのファイルを、変えていない方から戻して `cmp` で確かめた。全部のあとで 2 つの写しを `diff -rq` で比べ、差がない。置き換えは cycle 3 の `apply.pl`(決めた回数だけあることを確かめてから書き、前後が同じ変異は断る)をそのまま使った。テストのファイルを変える変異はないので、test binary は 1 つ。断りのないものは skill の写し 4 つすべてを変えた。G2 は欠番(G1 と同じ文を消すだけで、G1 より得るものがないので作らなかった)。

| # | 変異 | 結果 | 落としたテスト(行) |
| --- | --- | --- | --- |
| G1 | 「機能ごとの出力先」の 3:「重なる org を全部数え直し、説明に全部書く」を「数え直さず、エラーの org だけを説明に書く」に | 落ちた | `…RecommendsFilingAnIssueForAFeatureThatOverlapsARunningOrg`(:356) |
| G3 | 「起票と記録」の 1:「先に `ralph status` を打ち直して」の `ralph status` を `ralph org status` に | 落ちた | `…FilesTheIssueWithAMarkLineAndRecordsAFollowupPerBranch`(:424) |
| G4 | 同じ 1 の「重なる org を全部数え直す」を「start のエラーの org だけを使う」に | 落ちた | 同上 |
| G5 | 同じ 1 の数え直しの文を、`ralph org status --org-id` の文の後ろへ動かす(語は変えない) | 落ちた | 同上(断片の順の確かめ) |
| G6 | 引き継ぎ:「(答えを聞く前に done にしない)」を「done にする」に | 落ちた | `…TakeoverRecordsTheMissingFollowupsOfTheFiledIssues`(:653) |
| G7 | 引き継ぎ:「merge 済みのブランチも待ち終えたものに数える」を「merge 済みのブランチは待ち終えたものに数えない」に | 落ちた | 同上 |
| G8 | 引き継ぎ:「今の `after:` のブランチの follow-up がすべて open」から「今の」を消す | 落ちた | 同上 |
| G9 | 「merge を待つ issue」の 3:「2 の問いでもう一度聞いてから done にする」を「聞かずにすぐ done にする」に | 落ちた | `…MarksTheFiledIssueDoneOrRewritesItsSplitPlanBeforeStarting`(:882) |
| G10 | 同じく 2:「`ready` の件がなくても」を「`ready` の件があるときだけ」に | 落ちた | `…AsksWhetherToStartAFiledIssueOnceEveryBranchItWaitsForIsMerged`(:749) |
| G11 | 2:「STATE は `waiting` でも」を「STATE が `ready` で」に | 落ちた | 同上 |
| G12 | 2:「すべて `ready` か記録の直前に merge 済みの」から「か記録の直前に merge 済み」を消す(S15 の前の文に戻す) | 落ちた | 同上 |
| G13 | 2:「待ち終えていないブランチが残っていれば聞かない」を S15 の前の「`waiting` のブランチが残っていれば聞かない」に戻す | 落ちた | 同上(:749 と :809) |
| G14 | 2: 新しい文はそのままに、直後に古い文「`waiting` のブランチが残っていれば聞かない。」を足す | 落ちた | 同上(:809 だけ) |
| G15 | G14 を `templates/base/.agents` の写し 1 つだけに | 落ちた | 同上(:809、その写しの名前だけ) |
| G16 | 引き継ぎ:「(答えを聞く前に done にしない)。」の直後に「急ぐときは答えの前に done にしてよい。」を足す | **残った**(見込みどおり) | なし |
| G17 | 2:「merge されていないことを確かめてから記録するので」を「merge を確かめずに記録するので」に | 落ちた | 同上(:749) |
| G18 | 2:「待ち終えたブランチに数える」を「待ち終えていないブランチに数える」に | 落ちた | 同上 |
| G19 | 2:「`ralph org merge-order --all` で `merged` なのに」の `ralph org merge-order --all` を `ralph status` に | 落ちた | 同上 |
| G20 | 古い文「`waiting` のブランチが残っていれば聞かない。」を、step 2 でなく 3 の末尾(「もう一度聞いてから done にする。」の後ろ)に足す | **残った**(見込みどおり) | なし |
| G21 | 2 の末尾に、古い文の言い換え「`waiting` の件が残っていれば聞かない。」を足す | **残った**(見込みどおり) | なし |

G14 と G15 は、新しい断片がすべて残ったまま古い文だけが入る変異で、消えたことの確かめ(:809)だけが落とした。G20 と G21 は、その確かめが読む範囲を測るために足した。確かめは step 2 だけを読み、古い文と同じ語のときだけ落とす。

### Test gaps

1. 決まりと逆のことを固定した文の後ろに書き足す変異(G16)は落ちない。cycle 1 のテストの穴の 3、cycle 2 の C15、cycle 3 の D21 と同じで、断片の方式に元からある性質。この run で増えたものではない
2. 古い規則が消えたことの確かめは「merge を待つ issue」の step 2 だけを読み、古い文と同じ語だけを探す。step 3 に書かれた古い文(G20)と、言い換え(G21)は通る。重さは low。step 3 は答えのあとの記録の手順で、そこに「聞かない」の規則が戻る道筋は考えにくい。広げるなら、読む範囲を `### merge を待つ issue` の節全体にする(G20 が落ちる。試していない)。言い換えは断片の方式では止められない
3. director の振る舞い(重なる org の数え直し、add と done のあいだの中断からの拾い直し)は文としてだけ固定されている。director はモデルで、文を読んで動くので、打つテストはない。cycle 2・3 のテストの穴の 3 と同じ種類
4. 静的の検査(AC8 の `check-skill-sync.sh` と `check-sync.sh`)は /verify の担当で、ここでは打っていない(4 回目の verify の report で pass)

### Verdict

- Pass: Go 10 パッケージ(1,413 本の org を含む)と shell 45 本。`run-test.sh` は rc 0(1,625 秒)。`(cached)` だった 8 パッケージもキャッシュなしで ok(503 秒)。打ち直しの条件(30 分の時間切れ、既知の不安定なテスト)は起きず、`internal/org` は打ち直していない。AC8 の `go test ./internal/org/` の部分を満たす。S14・S15 への変異は 20 個中 17 個が落ちた
- Fail: 0
- Blocked: 0
- 残った変異 G16・G20・G21 は Test gaps の 1 と 2 に書いた。どれも見込みどおりで、テストを強める提案にとどまり、skill の文の誤りではないので、/pr を止めない
