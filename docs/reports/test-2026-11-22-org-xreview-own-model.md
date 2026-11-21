# Test report: org-xreview-own-model

- Date: 2026-11-22(実行は 06:57〜07:31 JST。パイプラインの cycle 1、`cycle-count.json` は `cycle: 1`)
- Plan: `docs/plans/active/2026-11-22-org-xreview-own-model.md`(issue #360)
- Tester: tester サブエージェント(`/test`)
- Scope: HEAD `7c7cdec5`(origin/main `bd079dbb` を取り込み済み、worktree は clean)。`./scripts/run-test.sh` を 1 回だけ、worktree のルートから回した。起動の前に `pgrep -f 'scripts/run-test.sh'` で、ほかの run-test.sh が走っていないことを確かめた。Bash tool の 10 分の上限を超えるので、perl の fork と setsid で切り離して起動し(INT・HUP・QUIT・TERM は既定の扱いに戻し、標準入出力は `/dev/null`。`nohup` は使っていない)、`caffeinate -w` で終わりを待った。範囲は wrapper の既定の changed(`- Requested scope: changed`)で、`scripts/xreview-helpers.sh` が言語に分類されないため `Language scope: full fallback (unclassified:scripts/xreview-helpers.sh)` になった。この repo で見つかる言語は golang だけなので、走る pack は changed のときと同じ。そのあと、依頼の新しいテストの `-v -count=1` の run、変異 25 個と対照 1 個、狭い coverage の run を足した
- Evidence: `docs/evidence/test-2026-11-22-org-xreview-own-model.log`(6,843 行。`.gitignore` の `docs/evidence/*.log` に当たるので commit されない)。run-test.sh の出力、1 分ごとの load、絞った run の `-v` の出力、変異の harness と各変異の落ちたテストと失敗の文、coverage の要約が入っている。run-verify.sh は自分の記録として `docs/evidence/verify-2026-11-21-215713.log` も書いた(同じく追跡外)

## Test execution

| Suite / Command | Tests | Passed | Failed | Skipped | Duration |
| --- | --- | --- | --- | --- | --- |
| `./scripts/run-test.sh` | 下の 2 行の合計 | 全部 | 0 | 2(条件つきの skip、下の行) | 2026-11-22 06:57:12 〜 07:27:48 JST、1,836 s、rc 0。`==> All verifiers passed.` |
| local verifier(`tests/test-*.sh` 46 本) | 46 本、`PASS` で始まる行 5,390 | 46 本(`OK` 46 行) | 0(`FAIL:` の要約行 29 本はどれも `FAIL: 0`) | `test-plan-visual.sh` の browser detection(`/Applications` にブラウザーがある)、`test-secret-scan-branch.sh` の「git 2.41 より古い本物の git」の場合(手元は git 2.49.0) | 約 13 分(06:57:12 〜 07:10 ごろ) |
| `go test ./...`(テストのある 11 パッケージ) | 11 | 11 | 0 | なし。`(cached)` は 0 | 約 17 分(07:10 ごろ 〜 07:27:48)、下の表 |
| `go test -count=1 -v -timeout 30m ./internal/org/ -run '<依頼の新しいテストと周りの 10 の型>'` | 127 関数、`--- PASS` 209 行(subtest を含む) | 全部 | 0 | 0 | 55 s(パッケージ 54.377 s) |

絞った run の `-run` は `Reviewer_Xreview|Reviewer_FanOut|Reviewer_ShowsTheSeatsDriverAndModel|NoDriverOrModelPlaceholderLeft|ReviewerPromptFileNamesTheSeatsDriverAndModel|Leader_Xreview|ReseatRuleSeatCheckItem|Skill_|SayNothingOfTheOtherModels|Leader_FeatureOrgPlanAdvisory`。内訳は `TestDirectorSkill` 44、`TestRenderRolePrompt_Leader` 31、`TestOrgSkill` 31、`TestRenderRolePrompt_Reviewer` 18、ほか 3。

shell の 2 つの skip は、前の周回の codex-seat-network の報告と同じ 2 つで、この変更とは関係がない。`tests/test-xreview-helpers.sh` は `PASS: 53`、`FAIL: 0` だった(AC6、下の Regression checks)。

| パッケージ | 結果 | 時間 |
| --- | --- | --- |
| `cmd/ralph` | ok | 1.604 s |
| `internal/cli` | ok | 996.084 s |
| `internal/config` | ok | 28.179 s |
| `internal/gitroot` | ok | 5.267 s |
| `internal/insights` | ok | 2.276 s |
| `internal/org` | ok | 979.256 s |
| `internal/org/driver` | ok | 4.822 s |
| `internal/org/protocol` | ok | 4.327 s |
| `internal/scaffold` | ok | 3.103 s |
| `internal/toolchain` | ok | 25.345 s |
| `internal/upgrade` | ok | 4.116 s |

golang pack の 1 パッケージの上限は 30 分で、`internal/cli` はその 55%、`internal/org` は 54% で終わった。run のあいだの 1 分 load は 7.73〜33.66(1 分ごとに 31 回測った)。変異の run は 07:05〜07:23 に回し、run-test.sh の shell の段の終わりと Go の段の前半に重なった。落ちたテストはなく、打ち直しもしていない。

run のあと、worktree の `git status --porcelain` は空で、HEAD は `7c7cdec5` のまま。`~/.codex/config.toml` の更新時刻は 2026-11-20 20:41:50 のままで、run のあいだに書かれていない(読まれていないことは macOS では示せない)。本物の herdr server(pid 59855)は run の前後で同じ pid で動いていて、止めても触ってもいない。

### 依頼の新しいテストと AC の対応

| 依頼の項目 / AC | テスト | 結果 |
| --- | --- | --- |
| reviewer の cross-review の手順のコマンドを bash と zsh で走らせる(AC1、AC2) | `TestRenderRolePrompt_Reviewer_XreviewCommandsWriteTheStateAndTheDiff`(bash と zsh)。手順 2 が `diff: written` を出して差分と HEAD で始まる git の状態を書き、手順 5 が `git: same` を出す。codex と claude の stand-in は 1 度も呼ばれず、作業ツリーは clean のまま | PASS |
| git の状態が変わる 4 種類(AC1) | `TestRenderRolePrompt_Reviewer_XreviewCommandOutcomes` の 4 種類 × 2 shell(追跡中のファイルの書き換え、新しい未追跡のファイル、手順 2 の前からある未追跡のファイルの書き換え、新しいコミット)。どれも手順 5 が `git: changed` を出す。同じテストの `a base git diff cannot read` と `a base that reads as an option` も 2 shell で通る | PASS |
| 前の版の TASK の行を読まない(AC2) | `TestRenderRolePrompt_Reviewer_XreviewTaskCarriesItsLinesInTheBody`(3 行だけを挙げ、`IMPLEMENTER_DRIVER:` などを読まないと言う文)、`TestRenderRolePrompt_Reviewer_XreviewReviewsOnTheSeatsOwnModel`(その文と別のモデルを呼ばない文の外に、前の言葉が残らない)、`…CommandsWriteTheStateAndTheDiff` の前の版の本文でのコマンドの比べ | PASS(比べについては Test gaps の 2) |
| R1 の待ちの決まり(self-review の M1、f95a08f9) | `TestRenderRolePrompt_Leader_XreviewBlockedFollowsTheGatesAndStallIsBounded`(手順 12 と cross-review の小節)、`TestRenderRolePrompt_Leader_ReseatRuleSeatCheckItem`(ミッションの 5)、`TestDirectorSkill_XreviewStallIsWaitedOutOnlyWhileTheScreenMoves`、`TestOrgSkill_EndOfOrgStep4WaitsOutAnXreviewStallOnlyWhileTheScreenMoves` | PASS |
| AC3(report の形) | `…XreviewWritesTheReportUncommitted`、`…XreviewReportIsReadByTheTriageCounter`、`…XreviewAnswersBlockedOnlyForActionRequired`、`…XreviewComparesTheGitStateBeforeTheReport` | PASS |
| AC4(雛形の driver と model) | `TestRenderRolePrompt_Reviewer_ShowsTheSeatsDriverAndModel`、`TestRenderRolePrompt_SeatTemplates_NoDriverOrModelPlaceholderLeft`(3 役 × model あり・なし)、`TestOrgSpawn_ReviewerPromptFileNamesTheSeatsDriverAndModel`(fake の herdr で spawn が書いた雛形のファイル) | PASS |
| AC5(leader の小節) | `TestRenderRolePrompt_Leader_XreviewTaskCarriesTheThreeLinesTheReviewerReads`、`…XreviewReadsNoDriverAndNamesNoOtherModel`、`…XreviewHandlesEveryReply`、上限と `Cross-review:` の行を固める残りの `Leader_Xreview` の 16 関数(状態のコマンドを走らせる 8 関数を含む) | PASS |
| AC7(skill) | `TestOrgSkill_RoleModelsSayTheReviewerSeatReviewsOnItsOwnModel`、`TestDirectorSkill_XreviewSaysTheReviewerSeatReviewsOnItsOwnModel`、`TestOrgAndDirectorSkills_SayNothingOfTheOtherModelsCrossReview`(4 写しずつ)。`check-sync.sh` と `check-skill-sync.sh` は local verifier の中で通った | PASS |

### 変異

HEAD の `git archive` を scratch に展開した写しで回し、worktree には触れていない。harness は置き換える文字列の数が決めた数と合うことを確かめてから替え(日本語の折り返しのある文は、文字のあいだの空白を許す一致で数えた)、各変異のあとにファイルを元のバイト列に戻した。最後に、写しを新しく展開した写しと `diff -rq` で比べ、違いがないことを確かめた。雛形と Go のコードの変異は `go test -count=1 -run <絞り込み> ./internal/org/` を写しで打ち、skill の変異は写しで一度だけ build したテストの binary で打った。どの変異も `build failed` は 0 件。

| # | 変異 | 結果 | 落ちたテスト |
| --- | --- | --- | --- |
| M01 | 手順 2 と 5 のコマンドから未追跡のファイルの hash(`git ls-files … \| xargs -0 git hash-object --;`)を外し、共有の定数 `leaderAdvisoryGitState` からも外した(実行の行だけが落とせる形) | 落ちた | `XreviewCommandOutcomes` の `an edit of an untracked file`(bash、zsh) |
| M02 | 同じく `git rev-parse HEAD;` を外した | 落ちた | `XreviewCommandOutcomes` の `a new commit`(2 shell)、`XreviewCommandsWriteTheStateAndTheDiff`(HEAD で始まらない) |
| M03 | 同じく `git diff;`(作業ツリーの差分)を外した | 残った | なし(Test gaps の 1) |
| M03t | M03 を雛形だけに入れ、定数は元のまま | 落ちた | `XreviewWritesTheGitStateAndTheDiff`、`XreviewComparesTheGitStateBeforeTheReport`(コマンドの行の一致) |
| M04 | 同じく `git diff --cached;` を外した | 残った | なし(Test gaps の 1) |
| M05 | 同じく `git status --porcelain=v1 --untracked-files=all;` を外した | 残った | なし(Test gaps の 1) |
| M06 | 手順 2 のコマンドの頭に `codex review > /dev/null 2>&1;` を足した | 落ちた | 行の一致と「codex を打たない」の検査に加え、`XreviewCommandOutcomes` の全 subtest と `…CommandsWriteTheStateAndTheDiff` で `calls = [["codex" "review"]]`(12 件)。stand-in の呼び出しの記録が効いている |
| M07 | 前の版の行を読まない文を消した | 落ちた | `XreviewReviewsOnTheSeatsOwnModel`、`XreviewTaskCarriesItsLinesInTheBody` |
| M08 | 手順 1 の確かめる行に `IMPLEMENTER_DRIVER:` を足した | 落ちた | `XreviewReviewsOnTheSeatsOwnModel`(前の言葉の一覧) |
| M08b | TASK の行の一覧に `IMPLEMENTER_DRIVER:` の項目を足した | 落ちた | `XreviewReviewsOnTheSeatsOwnModel`、`XreviewTaskCarriesItsLinesInTheBody`(行の一覧が 3 行でない) |
| M09 | report の `Reviewer:` の行から `、{{MODEL}}` を外した | 落ちた | `XreviewWritesTheReportUncommitted` |
| M10 | `## Triage context` の `Diff:` の行を前の `Reviewer output: …-out.md` に戻した | 落ちた | `XreviewReviewsOnTheSeatsOwnModel`、`XreviewWritesTheReportUncommitted` |
| M11 | `prompts.go` の `{{MODEL}}` を `ModelOrCLIDefault(vars.Model)` でなく `vars.Model` で置き換えた | 落ちた | `ShowsTheSeatsDriverAndModel` と `ReviewerPromptFileNamesTheSeatsDriverAndModel` の codex の model なしの行、`XreviewWritesTheReportUncommitted` |
| M12 | `spawn.go` の Spawn の呼び出しで `Model: ""` を渡した | 落ちた | `TestOrgSpawn_ReviewerPromptFileNamesTheSeatsDriverAndModel/a_claude_seat_with_a_model` |
| M13 | 同じ所で `Driver: "claude"` を渡した | 落ちた | `TestOrgSpawn_ReviewerPromptFileNamesTheSeatsDriverAndModel/a_codex_seat_with_no_model` |
| M14 | leader のミッションの 5 を前の例外(「cross-review の差分を読んでいるときは、手順 12 と 9 のとおり、画面が動かなくても待ちます」)に戻した | 落ちた | `ReseatRuleSeatCheckItem`、`XreviewBlockedFollowsTheGatesAndStallIsBounded` |
| M14b | ミッションの 5 から「cross-review の差分を読んでいる reviewer は、ほかの座席と同じく扱います」だけを消した | 落ちた | `ReseatRuleSeatCheckItem` |
| M15 | 手順 12 から「10 分たっても画面が動かず … BLOCKED … を上げ、返事が来るまで待つ」を消した | 落ちた | `XreviewBlockedFollowsTheGatesAndStallIsBounded` |
| M16 | cross-review の小節の待ちを前の「手順 12 の gate のときと同じく扱う」に戻した | 落ちた | `XreviewBlockedFollowsTheGatesAndStallIsBounded` |
| M16b | 小節から「gate と違い、10 分たっても … BLOCKED を上げる」だけを消した | 落ちた | `XreviewBlockedFollowsTheGatesAndStallIsBounded` |
| M17 | leader の TASK の例に `IMPLEMENTER_DRIVER: claude` の行を足した | 落ちた | `XreviewTaskCarriesTheThreeLinesTheReviewerReads`、`XreviewReadsNoDriverAndNamesNoOtherModel` |
| M18 | 「本文に書くのはこの 3 行だけで、implementer の driver は … 読まず」の文を消した | 落ちた | 同じ 2 関数 |
| S1 | director の skill から「10 分たっても画面が動かなければ、それ以上は待たずに上のとおり決める(…)」を消した | 落ちた | `TestDirectorSkill_XreviewStallIsWaitedOutOnlyWhileTheScreenMoves` |
| S2 | org の skill の終わり方の 4 から、10 分で escalate する文を消した | 落ちた | `TestOrgSkill_EndOfOrgStep4WaitsOutAnXreviewStallOnlyWhileTheScreenMoves` |
| S3 | org の skill の「役割ごとのモデル」から、同じ系列でも違う系列でも同じ、の文を消した | 落ちた | `TestOrgSkill_RoleModelsSayTheReviewerSeatReviewsOnItsOwnModel` |
| S4 | director の skill に前の例外の言葉(「reviewer が差分を読んでいれば待つ。」)を足しただけの変異(残した文には触れない) | 落ちた | `TestDirectorSkill_XreviewStallIsWaitedOutOnlyWhileTheScreenMoves`(前の言葉が残っていない、の検査) |

S1〜S4 は `.claude/skills/` の写し 1 つだけを替えた。テストは 4 写しを 1 つずつ読むので、どの写しを替えても同じく落ちる。

## Coverage

- Statement: 狭い run(`-run 'RenderRolePrompt|ReviewerPrompt|DryRun'`)の profile で、`RenderRolePrompt` 100%、`ModelOrCLIDefault` 100%。変えた 2 つの呼び出し(`spawn.go` の Spawn の `RenderRolePrompt` の呼び出しを含む 1116〜1122 行の塊と、dryRunSpawn の 2418〜2425 行の塊)はどちらも実行される
- Branch: Go の標準ツールでは測れない
- Function: 上のとおり
- Notes: この変更の中身の多くは雛形(`go:embed` の markdown)と skill と文書で、Go の coverage では測れない。雛形と skill は文を固定するテストと、雛形のコマンドを bash と zsh で走らせるテストで守られ、上の変異でその効きを確かめた

## Failure analysis

| Test | Error | Root cause | Proposed fix |
| --- | --- | --- | --- |
| なし | — | — | — |

## Regression checks

| Previously broken behavior | Status | Evidence |
| --- | --- | --- |
| issue #360: 機能の org の reviewer 座席が座席の中から `codex exec` や `claude -p` を入れ子で呼び、codex の sandbox の中で incomplete になる、20 分待つ | 雛形の上では直った(座席の中で実際に動かしてはいない、Test gaps の 3) | `XreviewReviewsOnTheSeatsOwnModel` の前の言葉の検査、コマンドを走らせるテストの stand-in の呼び出しが 0 件、M06 で呼び出しを足すと落ちる |
| 読んでいるあいだに HEAD や作業ツリーが動くと、見ていない変更に pass が付く(計画助言の MEDIUM 1) | 比べは残っている | `XreviewCommandOutcomes` の 4 種類 × 2 shell が `git: changed`。M01・M02 で比べの部品を外すと落ちる |
| 計画助言の reviewer の行(`- reviewer の codex:` と `- reviewer の claude:`) | 変わらない | `Leader_FeatureOrgPlanAdvisory` で始まる 10 関数が絞った run で PASS |
| 上限の数え方のコマンド(`done:` と `k:`) | 変わらない | `Leader_XreviewStateCommand*` の 8 関数が PASS |
| `XREVIEW_BASE:` の値の確かめ(前の計画の F6〜F8) | 保たれている | `XreviewBaseCheckRunsGitCheckRefFormat`、`…RefusesBeforeItCallsGit`、`…BaseNamesWorkInTheCommands` が bash と zsh で PASS |
| 標準フローの `/cross-review` と `scripts/xreview-helpers.sh`(AC6) | 振る舞いは変わらない | `tests/test-xreview-helpers.sh` が `PASS: 53`、`FAIL: 0`。差分はコメントの 1 か所だけ(verify の報告の V7) |

## Test gaps

1. git の状態の比べの実行の行は 4 種類で、HEAD と未追跡のファイルの中身はそれぞれ 1 つの行が落とせる(M01、M02)。一方、作業ツリーの差分(`git diff`)、index の差分(`git diff --cached`)、`git status` の部品は、雛形と共有の定数 `leaderAdvisoryGitState` から一緒に外すと、どの実行の行も落ちない(M03〜M05)。雛形だけを替えればコマンドの行の一致で落ちる(M03t)ので、今の雛形は守られている。残る穴は、定数と雛形を一緒に変える変更を振る舞いで止める行がないこと。手順 2 の前から書き換えてある追跡中のファイルをもう一度書き換える行(M03)、stage 済みの変更にさらに stage を足す行(M04)、同じ中身の未追跡のファイルの名前を変える行(M05)を足せば、それぞれが落ちる。同じ定数を使う leader の計画助言にも同じことが言える(LOW)
2. `XreviewCommandsWriteTheStateAndTheDiff` の「前の版の leader の TASK から埋めたコマンドが 3 行の TASK から埋めたものと同じ」の比べは、テストの埋め方が `REPORT:` と `XREVIEW_BASE:` の値しか使わないので、組み立てのうえで常に等しく、単独では何も落とせない。前の版の TASK でも動くことは、行を読まないと言う文、3 行だけの一覧、前の言葉の検査が守っている(M07、M08、M08b で落ちる)。前の版の TASK を雛形のとおりに確かめる形のテスト(手順 1 の欠けた行の確かめを、前の版の本文で「欠けていない」と読む)はない(LOW)
3. reviewer 座席が herdr の pane の中で、本当に相手の CLI を呼ばずに自分のモデルで差分を読み、report を書くことは確かめていない。雛形の文とコマンドの確かめまでで、座席の振る舞い(モデルが雛形に従うか)は unit テストの外。依頼どおり本物の herdr server には触れていない
4. 前の版の reviewer 座席に新しい leader が 3 行の TASK を送ると QUESTION になる(計画の Assumptions)ことは、前の版の雛形がこの binary にないので確かめていない。release ノートで reseat を勧める扱い
5. dryRunSpawn の `RenderRolePrompt` に渡す `Driver` と `Model` は、dry run では雛形の長さ(prompt のファイルにするかどうか)にしか使われないので、外しても観測できない。固めるテストはないが、振る舞いの穴ではない

## Verdict

- Pass: `./scripts/run-test.sh` rc 0(1,836 s、shell 46 本、Go 11 パッケージ、`(cached)` なし)、依頼の新しいテスト(bash と zsh でのコマンドの run、git の状態が変わる 4 種類、前の版の TASK の行、R1 の待ちの決まり)を含む絞った run の 127 関数、変異 25 個のうち 22 個と対照 1 個が落ちること
- Fail: 0
- Blocked: 0
- Verdict: pass。残った変異 3 つ(M03〜M05)は、雛形だけの変更なら行の一致で落ちるので、この変更の振る舞いの穴ではなく、Test gaps の 1 に記録した。/sync-docs に進んでよい
