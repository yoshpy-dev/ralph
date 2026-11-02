# Test report: org-cross-review

- Date: 2026-11-02(cycle 2)。cycle 1 は 2026-11-01
- Plan: `docs/plans/active/2026-11-01-org-cross-review.md`(`## Test plan` と AC1〜AC9、Progress の F7 と F8)。Refs #269
- Tester: tester サブエージェント(`/test`、pipeline cycle 2。`cycle-count.json` の cycle は 2)
- 構成: 先頭が cycle 2 の節、次がこのファイルの判定の `## Verdict`(cycle 2 のもの)、最後が `## Cycle 1`(中は run 2、run 1 の順)。cycle 1 の節は中身を変えず、見出しを 1 段ずつ下げ、run 2 の判定の見出しを「Verdict(cycle 1 の run 2)」にし、冒頭の行(Date、Plan、Tester、構成、Evidence)をその節の頭に移した。ファイルの `## Verdict` は cycle 2 の 1 つだけ
- Evidence: `docs/evidence/test-2026-11-01-org-cross-review.md`(要点。commit する。先頭が cycle 2)、`docs/evidence/test-2026-11-01-org-cross-review.log`(gitignore。cycle 2 の分は 8,846 行目の `cycle 2 (HEAD 4424b257, …)` の区切りのあと。全出力、変異ごとの差分と落ちた行、直し方の probe、`git check-ref-format` の probe)。wrapper 自身の写しは `docs/evidence/verify-2026-11-02-003959.log`(gitignore)

## Cycle 2(HEAD 4424b257)

### Scope(cycle 2)

cycle 1 の run 2(HEAD 17fc2b52)のあとにブランチに入ったもの:

- F7 4d289952(cross-review cycle 1 の ACTION_REQUIRED 2 件)
  - leader の状態のコマンドの `since:` が除くのは `':(exclude,glob)docs/reports/*.md'`、つまり `docs/reports/` の直下の report だけになった(`internal/org/prompts/leader.md:810`、文は `:862-867`)。`docs/reports/templates/` の変更は `changed` になる
  - reviewer の手順 1 は、`XREVIEW_BASE:` を区切りを引用符で囲んだ heredoc で読み、`-` で始まる名前、`..`、`@{`、空白と制御文字、引用符、`$`、`` ` ``、`\` を断ってから `git check-ref-format --branch` に渡すコマンドで確かめる(`internal/org/prompts/reviewer.md:185-208`)
  - テストは `TestRenderRolePrompt_Leader_XreviewStateCommandSeesATemplateChange`(`internal/org/prompts_xreview_leader_test.go:1209`)を足し、reviewer の `…BaseTakesPlusAndAt` を `…Reviewer_XreviewBaseCheckRunsGitCheckRefFormat`(`internal/org/prompts_xreview_reviewer_test.go:1214`。コマンドを bash と `zsh -f` で走らせる表)と `…Reviewer_XreviewBaseNamesWorkInTheCommands`(`:1276`)に分けた。run 2 の helper `reviewerXreviewBaseOK` はなくなった
- F8 7724f6af(self-review cycle 2 の C2-1〜C2-4)
  - reviewer は、base の確かめが `base: invalid` を出したときだけ QUESTION にし、どちらの行も出なければ打ち直す(`reviewer.md:166-168`、`:197-200`)
  - leader は、同じ `<k>` の送り直しを QUESTION のあとと `git_changed` のあとを合わせて 2 回までにし、その上限を TASK_ID の段落に 1 回だけ書く(`leader.md:953-961`)。`git_changed` のあとは、`git status --porcelain` が何も出さないときだけ送り直す(`leader.md:1060-1065`)
  - ほかはコメントと tech-debt の行
- マージ 4424b257(origin/main 31812a3b): main の #271(director の分割の問い。`/director` と `/org` の skill)、#297(standard-config-gaps。`internal/cli` の org start の設定)、org-ledger-gaps を入れた。手で解いた衝突は `docs/tech-debt/README.md` だけで、`/director` と `/org` の skill は、このブランチの cross-review の段落と git が衝突なしに混ぜた。verify cycle 2 の報告(58031854)はこのマージの前

範囲は handoff のとおり。`./scripts/run-test.sh`(changed scope)、`go test -count=1 -race ./internal/org/ -run 'Xreview|Prompt|Leader|Reviewer|Skill|Director|SyncDocs'` を 1 回、`scripts/check-skill-sync.sh`(drift の確かめで普段は verifier の範囲だが、handoff の指示で打った)、F7 と F8 の規則の変異。

### Test execution(cycle 2)

| Suite / Command | Tests | Passed | Failed | Skipped | Duration |
| --- | --- | --- | --- | --- | --- |
| `./scripts/run-test.sh`(changed scope、rc 1) | shell 45 本 + Go 10 パッケージ | shell 45 本 OK(`PASS:` の行 2,604、0 でない `FAIL:` の行 0)、Go 9 パッケージ ok | `internal/org` の 1 本 | 1 行(host の条件: plan-visual の browser 検出) | 2,255 秒(09:39:59〜10:17:34 JST)。load は開始 5.40 / 5.15、終了 5.97 / 6.37 |
| └ Go: `internal/org` | — | 1 本のほかは通った | 1 | 0 | 1,178.1 秒 |
| └ Go: `internal/cli` | — | ok | 0 | 0 | 997.1 秒 |
| └ Go: `cmd/ralph`、`internal/{config,insights,org/driver,scaffold}` | — | ok | 0 | 0 | 1.4〜15.5 秒 |
| └ Go: `internal/{gitroot,org/protocol,upgrade}` | — | `(cached)`。`go test -count=1` で打ち直して ok | 0 | 0 | 打ち直し 1.8〜5.1 秒(合わせて 6 秒) |
| `go test -count=1 -race -timeout=60m ./internal/org/ -run 'Xreview\|Prompt\|Leader\|Reviewer\|Skill\|Director\|SyncDocs' -v` | トップレベル 516、サブテスト 905 | トップレベル 515、サブテスト 905 | 1(`run-test.sh` と同じ 1 本) | 0 | 226 秒(go の表示は 215.0 秒。09:41:42〜09:45:28)。DATA RACE 0 |
| 落ちた 1 本だけ(`-count=3`) | 3 | 0 | 3 | 0 | 2 秒 |
| `scripts/check-skill-sync.sh` | — | rc 0(`14 skill(s) in lock-step`) | 0 | — | 5 秒 |
| 変異 26 個 + 対照 1 個(`go test -overlay`、`-run 'Xreview\|RenderRolePrompt' -v`) | 27 | 対照は通った。変異は 19 個が落ちた | — | — | 1 個 144〜251 秒(09:44:52〜10:26:23。2 個ずつ並べた) |

`-race` の回のトップレベルの内訳は、`TestDirectorSkill_*` 37(落ちた 1 本を含む)、`TestOrgSkill_*` 27、`TestRenderRolePrompt_*` 121、名前に `SyncDocs` を持つもの 17 など。`/org` の skill の写しを読むテストと、このブランチの `skill_xreview_test.go` の 7 本は通った。

起動の仕方: `run-test.sh`、`-race` の回、変異の回は、Bash tool の `run_in_background` を使わず、perl の fork と setsid で切り離した(INT・HUP・QUIT・TERM を既定に戻し、標準入出力は /dev/null)。runner が log と開始・終了・rc の meta を書き、こちらは前景の `caffeinate -w <pid>` で待った。3 つを並べたので、その間の load は 4.9〜10.2。時間切れと打ち直しはない。scope は `Requested scope: changed` のまま、`scripts/xreview-helpers.sh` が分類の外なので `Language scope: full fallback` になり、golang の pack が `go test ./...` で全パッケージを打った。

スタブで走らせるテストは、どれも bash(3.2.57)と `zsh -f`(5.9)で打つ(`advisoryShells`)。本物の claude と codex は呼んでいない。

### Failure analysis(cycle 2)

| Test | Error | Root cause | Proposed fix |
| --- | --- | --- | --- |
| `TestDirectorSkill_AsksThePlanApprovalForksWithNumberedOptionsWhenTheWakePushes`(`internal/org/skill_split_questions_test.go:1127`。main の #271 が足したテスト) | director の SKILL.md の 4 つの写し(`.claude/`、`.agents/`、`templates/base/` の 2 つ)が、どれも「AskUserQuestion(Codex では番号つきの選択肢)」を持つ | マージの意味の衝突。このブランチの 93d58b65 が、「### cross-review の上限」の手順 2(`.claude/skills/director/SKILL.md:1089`。`git blame` も 93d58b65)を旧い言い方で書いた。main の #271 は同じ言い方を 2 か所で「Codex と、wake に push がある director は番号つきの選択肢」に直し、旧い言い方が skill のどこにも残らないことをこのテストで固定した。別の段落なので git は衝突なしで混ぜた | 4 つの写しの該当の文を、#271 の言い方(`:1019-1020`、`:544-545` と同じ「AskUserQuestion(Codex と、wake に push がある director は番号つきの選択肢。「人への聞き方」)」)に直す。同じ文を固定しているこのブランチの `TestDirectorSkill_XreviewCapQuestionGoesToTheHumanAsAFork`(`internal/org/skill_xreview_test.go:152` の断片)も一緒に直す |

旧い言い方の数は、merge-base 4fef6b8c で 2 か所、origin/main で 0、マージ前のブランチ 58031854 で 3 か所(`:801`、`:809`、`:865`。最後がこのブランチの段落)、HEAD では 4 つの写しに 1 か所ずつ。単独で `-count=3` を打つと 3 回とも落ちるので、flaky ではない。

直し方の確かめ: worktree の外の `git archive HEAD` の写しで直し方を試した(worktree のファイルは変えていない)。4 つの写しの文だけを直すと、`go test -count=1 ./internal/org/ -run 'Skill|Director'` は今度は `TestDirectorSkill_XreviewCapQuestionGoesToTheHumanAsAFork` で落ちる(断片が旧い言い方を固定しているため。123 秒)。テストの断片も同じ言い方に直すと、`-run 'Skill|Director|Xreview'` が通り(328.6 秒)、写しの `check-skill-sync.sh` も通った。

### 変異(cycle 2)

`go test -overlay` で go:embed の雛形(E2 ではテストの `prompts_xreview_reviewer_test.go` も)を scratch の写しに差し替えた。写しは perl で文字列を置き換えて作った(置き換えるたびに一致が 1 か所でなければ止まり、置き換えの前後が同じでも止まる)。対照 K0 は雛形を変えずに差し替えたもので、通った。worktree のファイルは変えていない。変異の前後で `git status --porcelain` は空、`git status --porcelain --ignored` は go のビルドキャッシュ(`.harness/cache/`)を除いて同じ。handoff が挙げた 5 つは A1、B1、C1、D1、E1。

| ID | 変異 | 結果 | 落としたテスト(頭の `TestRenderRolePrompt_` は省いた) |
| --- | --- | --- | --- |
| A1 | (a) `since:` が除くのを `docs/reports/` 全体に戻す: `leader.md:810` の `':(exclude,glob)docs/reports/*.md'` → `':(exclude)docs/reports/'` | 落ちた | `Leader_XreviewStateCommandSeesATemplateChange` の bash・zsh(templates/ だけのコミットのあとが since: same、want changed)、`Leader_XreviewChecksTheValuesItReads`(コマンドの文字列の固定) |
| A2 | 同じ箇所の `glob` だけを外す(`':(exclude)docs/reports/*.md'`。glob でない pathspec の `*` は `/` もまたぐので、templates/ も除かれる) | 落ちた | 同じ 2 本(bash・zsh) |
| A3 | 文: `leader.md:865-867` の「`docs/reports/templates/` のような下のディレクトリは…`changed` になる」を消す | 落ちた | `Leader_XreviewCountsOnlyCompleteReports` だけ(文の固定) |
| B1 | (b) `reviewer.md:207` の `git check-ref-format --branch` を外し、case を抜けた値に `base: ok` を出す | 落ちた | `Reviewer_XreviewBaseCheckRunsGitCheckRefFormat` の bash・zsh(`origin/a.lock`、`origin/HEAD`、`origin/a~1`、`origin/a^` が base: ok) |
| B2 | `--branch` → `--allow-onelevel` | 落ちた | 同じテストの bash・zsh(`origin/HEAD` だけ。`--allow-onelevel` は `HEAD` を通す) |
| C1 | (c) case から `*'@{'*` を外す | 落ちた | 同じテストの bash・zsh(`origin/@{-1}` が base: ok。git は `--branch` で `@{-1}` を前のブランチの名前に読み替えて通す) |
| C2 | 引用符の 2 つ(`*\"*` と `*\'*`)を外す | 落ちた | 同じ(`origin/a"b`、`origin/a'b`) |
| C3 | `*\"*` だけを外す | 落ちた | 同じ(`origin/a"b`) |
| C4 | `*\'*` だけを外す | 落ちた | 同じ(`origin/a'b`) |
| C5 | `*\$*` を外す | 落ちた | 同じ(`origin/a$b`、`origin/a$(touch${IFS}pwned)`) |
| C6 | `` *\`* `` を外す | 落ちた | 同じ(``origin/a`b``) |
| C7 | `*\\*` を外す | 残った | なし |
| C8 | `*[[:space:][:cntrl:]]*` を外す | 残った | なし |
| C9 | `*..*` を外す | 残った | なし |
| C10 | `-*` を外す | 残った | なし |
| C11 | `''`(空の名前)を外す | 残った | なし |
| C12 | `case "$b" in origin/*) ;; *) r= ;; esac` を外す(`origin/` で始まらない値もそのまま名前として渡す) | 落ちた | 同じテストの bash・zsh(`feature/x`、`main`) |
| D1 | (d) `leader.md:953-955` の送り直しの上限の文を消す | 落ちた | `Leader_XreviewTaskCarriesTheReviewersLines`(文の固定)、`Leader_XreviewHandlesEveryReply`(上限が 2 つの項目より前にない、「2 回(`-r2` と `-r3`)まで」が 0 回) |
| D2 | 同じ文を TASK_ID の段落から QUESTION の項目(`leader.md:1070`)へ移す(F8 の前の置き場所) | 落ちた | 同じ 2 本(順の固定と、上限が 2 つの項目より前にあるかの確かめ) |
| D3 | 「`git_changed` のあとを合わせて 2 回」→「`git_changed` のあとでそれぞれ 2 回」 | 落ちた | 同じ 2 本(文の固定) |
| D4 | 足すだけ: `git_changed` の項目の「同じ `<k>` を送り直す。」のあとに「この送り直しは送り直しの上限に数えない。」を足す | 残った | なし |
| D5 | F8 の文: `git_changed` の項目の「手順 9 と同じく `git status --porcelain` が何も出さないことを確かめてから、」を消す | 落ちた | `Leader_XreviewHandlesEveryReply` だけ(文の固定) |
| E1 | (e) reviewer の手順 1 の QUESTION の条件を「`base: invalid` を出すか」→「`base: ok` を出さないか」に戻す(雛形だけ) | 落ちた | `Reviewer_XreviewBaseCheckRunsGitCheckRefFormat`、`Reviewer_XreviewTaskCarriesItsLinesInTheBody`(どちらも文の固定) |
| E2 | E1 を雛形とテストの定数 `reviewerXreviewBaseRule` の両方に入れる | 残った | なし |
| E3 | F8 の文: `reviewer.md:197-200` の「コマンドが `base: ok` も `base: invalid` も出さなければ…QUESTION を送らない」を消す | 落ちた | `Reviewer_XreviewBaseCheckRunsGitCheckRefFormat` だけ(文の固定) |
| E4 | heredoc の区切りの引用符を外す: `<<'XREVIEW_BASE_END'` → `<<XREVIEW_BASE_END` | 落ちた | `Reviewer_XreviewBaseCheckRunsGitCheckRefFormat` の bash・zsh(`origin/a$b` が `origin/a` に展開されて base: ok。bash では `$(touch${IFS}pwned)` も展開され、command not found のあと base: ok) |

handoff の 5 つはすべて落ち、26 個では 19 個が落ちた。コマンドの変異(A1、A2、B1、B2、C1〜C6、C12、E4)はスタブで走らせる行が落とし、文だけの変異(A3、D1〜D3、D5、E1、E3)は文を固定するテストだけが落とす。leader と reviewer は文を読むモデルなので、cycle 1 と同じく予想どおり。

残った 7 個:

- C7〜C11 は、git と同じ規則なので結果が変わらない変異。手元の git 2.49.0 で `git check-ref-format --branch` に名前だけを渡すと、`a\b`、`a b`、タブと `\x01` を持つ名前、`a..b`、`-x` と `-`、空の名前をどれも断る(evidence の probe の表)。case が断らなくても git が断るので、表の行は `base: invalid` のまま。case のこの 5 つは、git が変わったときの備えとして効く
- D4 は足すだけの文。断片を順に固定するテストは、間に足した文を見ない
- E2 は雛形とテストの定数を一緒に変えたもの。手順 1 の QUESTION の条件が「`base: ok` を出さないか」に戻っても、同じ手順の「どちらの行も出なければ打ち直し、QUESTION を送らない」の文は残り、2 つの文が食い違っていても落ちるテストはない

### Coverage(cycle 2)

- この回も Go の関数の中身の変更はない(F8 の `internal/config/config.go` はコメントだけ。マージで入った main のコードは、main の各 PR の範囲)。statement の計測はしていない
- 雛形の新しいコマンド(reviewer の手順 1 の base の確かめ、leader の `since:` の pathspec)はスタブで走らせる行が分岐を打ち、コマンドの変異 11 個がそれを確かめた。F8 の規則(送り直しの上限、`git_changed` のあとの確かめ、打ち直し)は文だけで、文の固定が確かめる

### Test gaps(cycle 2)

1. 「### 人への聞き方」の冒頭の場面の列挙(`.claude/skills/director/SKILL.md:478-479`)に、cross-review の上限の `FORK` の問いが入っていない。#271 のテストは旧い言い方を禁じるだけで、AskUserQuestion で聞く場面がこの列挙に入っているかは確かめない。上の直しのときに一緒に見てほしい
2. C7〜C11: base の確かめの case のうち、`\`、空白と制御文字、`..`、頭の `-`、空の名前の 5 つは、git も断るので表の行では区別できない。固定するなら、PATH の先頭に常に 0 で終わる `git` のスタブを置き、case だけで断る行を足す(case が唯一の門になる)
3. E2: 手順 1 の QUESTION の条件(`reviewerXreviewBaseRule`)と打ち直しの文の食い違いを、どのテストも見ない。固定するなら、#271 のテストと同じく、旧い言い方「`base: ok` を出さないか」が手順 1 にないことを確かめる行を足す
4. D4: 文を足すだけの変異は残る。断片を順に固定するテストの性質で、cycle 1 と同じ種類
5. F8 の leader の 2 つの規則(送り直しの上限、`git_changed` のあとの `git status --porcelain` の確かめ)は文だけで、D1〜D3 と D5 は文の固定だけが落とす。cycle 1 の run 2 の gap 4 と同じ種類

cycle 1 の gap の扱い: run 1 の gap 2 のうち `XREVIEW_BASE:` の形は、F7 でコマンドになり、表の行が走らせる(B1、B2、C1〜C6、C12、E4 が落ちた)。`REPORT:` と `REVIEWER_*` の確かめは文のまま。run 2 の gap 1 と 2 は、F7 で `reviewerXreviewBaseOK` と `origin/a@{1}` だけに頼る表がなくなったので当たらない(表には `origin/@{-1}` と、断る文字を 1 つだけ持つ行がある)。run 1 の gap 1、3、4、5 と run 2 の gap 3、4 は残る。

## Verdict

- Verdict(cycle 2、HEAD 4424b257): fail
- Fail: 1(`TestDirectorSkill_AsksThePlanApprovalForksWithNumberedOptionsWhenTheWakePushes`。`run-test.sh` の `internal/org` と `-race` の回の両方で落ちた。マージ 4424b257 の意味の衝突で、Failure analysis(cycle 2)のとおり)
- Pass: ほかはすべて通った。shell 45 本、Go の 9 パッケージ(`(cached)` の 3 つは打ち直して ok)、`-race` の 515 本(DATA RACE 0)、`check-skill-sync.sh`。変異は handoff の 5 つがすべて落ち、26 個では 19 個が落ちた
- Blocked: 0
- 残った変異: C7〜C11(git と同じ規則)、D4、E2(Test gaps(cycle 2)の 2〜4)
- テストが落ちたので `/pr` に進まない。直すのは director の SKILL.md の 4 つの写しと `internal/org/skill_xreview_test.go:152` の断片

## Cycle 1

- Date: 2026-11-01(run 1 と run 2 の実行日)
- Plan: `docs/plans/active/2026-11-01-org-cross-review.md`(`## Test plan` と AC1〜AC9、Progress の S1〜S5 と F1〜F6)。Refs #269
- Tester: tester サブエージェント(`/test`、pipeline cycle 1。`cycle-count.json` の cycle は 1)。run 2 は F6 のあとの打ち直しで、cycle は 1 のまま
- 構成: 先頭が run 2 の節、次がこのファイルの判定の `## Verdict`(run 2 のもの)、最後が run 1 の節。run 1 の節は中身を変えず、見出しを 1 段下げ、判定の見出しを「Verdict(run 1)」にし、冒頭の Scope と Evidence の行をその節に移した。ファイルの `## Verdict` は run 2 の 1 つだけ
- Evidence: `docs/evidence/test-2026-11-01-org-cross-review.md`(要点。commit する。先頭が run 2、続いて run 1)、`docs/evidence/test-2026-11-01-org-cross-review.log`(gitignore。run 2 の分は 7,187 行目の `run 2 (HEAD 17fc2b52, after F6)` の区切りのあと。全出力、変異ごとの差分と落ちた行)

### Run 2(HEAD 17fc2b52、F6 のあと)

#### Scope(run 2)

- run 1(HEAD 15da7eb0)のあとのコードの変更は F6 17fc2b52 だけ。変えたのは `internal/org/prompts/leader.md`、`internal/org/prompts/reviewer.md`、テストの `internal/org/prompts_xreview_leader_test.go` と `internal/org/prompts_xreview_reviewer_test.go`。afccca76 と 5b6c3d42 は文書だけ
- F6 の中身:
  - leader の状態のコマンドが、`- Reviewer status: complete` の行を行末に空白があっても数える(`leader.md:808` の `grep -qE '^- Reviewer status: complete[[:space:]]*$'`。文は `:848`)
  - leader が `IMPLEMENTER_DRIVER:` を spawn のときの記憶でなく台帳から読む(行の説明は `leader.md:954-955`、段落とコマンドは `:968-991`)。コマンドは `ralph org status --json` の `seats` のうち `role` が `implementer` の座席の `driver` を jq で読み、座席がない、driver が claude でも codex でもない、driver が 2 種類ある、`ralph org status` か jq が失敗した、のどれかなら `driver: error (<理由>)` を出す
  - 同じ `<k>` の送り直しは `-r3` まで。`-r3` の TASK への QUESTION は BLOCKED にする(`leader.md:1055-1059`。文だけ)
  - reviewer の手順 1 が `XREVIEW_BASE:` のブランチ名の `+` と `@` を受け、`origin/` の直後の `-`、`..`、`{`(したがって `@{`)を断る(`reviewer.md:165-171`。文だけ)
  - reviewer が所見のパスを worktree の root からの相対パスで report に書く(`reviewer.md:364-368`。文だけ)
- 足したテストは 4 本: `TestRenderRolePrompt_Reviewer_XreviewBaseTakesPlusAndAt`(`internal/org/prompts_xreview_reviewer_test.go:1206`)、`TestRenderRolePrompt_Leader_XreviewStateCommandCountsACompleteLineWithTrailingSpace`(`internal/org/prompts_xreview_leader_test.go:1264`)、`…Leader_XreviewDriverCommandReadsTheRecord`(`:1339`)、`…Leader_XreviewDriverCommandReadsTheCLIsFields`(`:1438`)。既存の 5 本(reviewer 2 本、leader 3 本)は固定する文を直した。消えたテストはない
- 範囲は handoff のとおり、`internal/org` パッケージ全体、`-race` の `Xreview`、F6 の変異。`./scripts/run-test.sh` と `internal/cli` は打っていない。F6 が変えたのは go:embed の雛形 2 つと `internal/org` のテストだけで、`internal/cli`・`cmd/`・`tests/`・`scripts/` には F6 が変えた文を固定するテストがない(`grep -rln 'IMPLEMENTER_DRIVER\|Reviewer status: complete\|XREVIEW_BASE\|-r3'` に当たるのは環境変数 `RALPH_XREVIEW_BASE` の行と helper だけ)。ほかのパッケージの結果は run 1(15da7eb0)の `run-test.sh` のもの

#### Test execution(run 2)

| Suite / Command | Tests | Passed | Failed | Skipped | Duration |
| --- | --- | --- | --- | --- | --- |
| `GOFLAGS="-timeout=30m $(go env GOFLAGS)" go test -count=1 ./internal/org/` | パッケージ全体(`-v` なし) | ok | 0 | — | 1,096.4 秒(11:00:03〜11:18:20 JST)。load は開始 6.29 / 7.17、終了 10.67 / 9.26 |
| `GOFLAGS="-timeout=30m $(go env GOFLAGS)" go test -race -count=3 ./internal/org/ -run Xreview -v` | 40 の名前 × 3(サブテスト 84 × 3) | トップレベル 120、サブテスト 252 | 0 | 0 | 488.8 秒(11:03:41〜11:12:01)。DATA RACE 0。load 6.32 → 8.39 |
| 変異 17 個(`go test -overlay`、`-run Xreview -v`) | 17 | 14 個が落ちた | — | — | 1 個 112〜168 秒(11:12:12〜11:32:21。2 個ずつ並べた) |

名前は run 1 の 36 から 40 に、サブテストは 72 から 84 に増えた。増えた 12 は `…BaseTakesPlusAndAt` の 8(bash・zsh × base 2 つ × implementer の driver 2 つ)と、新しい leader の 2 本の bash・zsh。

起動の仕方: パッケージ全体の回は 600 秒を超えるので、Bash tool の `run_in_background` は使わず、perl の fork と setsid で切り離した(INT・HUP・QUIT・TERM を既定に戻し、標準入出力は /dev/null)。runner が log と開始・終了・rc の meta を書き、こちらは前景の `caffeinate -w <pid>` で待った。`-race` の回は前景で、パッケージ全体の回と並べて打った。変異の回も同じ形で切り離し(`xargs -P 2`)、パッケージ全体の回の終わりと重ねた。R1f は R1c の結果を見て足し、そのあと前景で打った。その間の load は 6.3〜13.6 で、時間切れも打ち直しもない。

スタブで走らせるテストは、どれも bash(3.2.57)と `zsh -f`(5.9)で打つ(`advisoryShells`)。対話の shell では打っていない。`…DriverCommandReadsTheRecord` は jq がないと SKIP するが、手元には jq があり、SKIP は 0。

#### 変異(run 2)

`go test -overlay` で go:embed の雛形(と、R1c〜R1f ではテストの `prompts_xreview_reviewer_test.go`)を scratch の写しに差し替えた。写しは perl で文字列を 1 か所だけ置き換えて作った(一致が 1 か所でなければ止まる)。worktree のファイルは変えていない。変異の前後で `git status --porcelain` は空、`git status --porcelain --ignored` は go のビルドキャッシュ(`.harness/cache/`)を除いて同じ。handoff が挙げた 4 つは L1a・L1b、R1a・R1b、L3a、L2a。

| ID | 変異 | 結果 | 落としたテスト(頭の `TestRenderRolePrompt_` は省いた) |
| --- | --- | --- | --- |
| L1a | driver を記憶から書く(文): `leader.md:954-955` の行の説明を F6 の前の「手順 5 で implementer を spawn したときの `--driver` の値」に戻す | 落ちた | `Leader_XreviewTaskCarriesTheReviewersLines` だけ(文の固定) |
| L1b | driver を記憶から書く(コマンド): `leader.md:978` の `ralph org status` を `:` で打たない形にし、覚えている claude の JSON(`{"seats":[{"role":"implementer","driver":"claude"}]}`)を status.json に書く。テストが文字列で探す `command ralph org status --org-id … --json` の並びと `<台帳>` の 1 か所は残る | 落ちた | `Leader_XreviewDriverCommandReadsTheRecord` の bash・zsh。14 行のうち 11 行(codex、止まった implementer、error の各行)と、status.err に stderr がない |
| L2a | 行末の空白を許さない: `leader.md:808` の `grep -qE '^- Reviewer status: complete[[:space:]]*$'` を F6 の前の `grep -qxF -e '- Reviewer status: complete'` に戻す | 落ちた | `Leader_XreviewStateCommandCountsACompleteLineWithTrailingSpace` の bash・zsh(done 1、want 3。last は c1) |
| L2b | 空白は許しタブは許さない: `[[:space:]]*$` → ` *$` | 落ちた | 同じ(done 2、want 3。last は c2) |
| L2c | 行末の錨を外す: `[[:space:]]*$'` → `[[:space:]]*'` | 落ちた | 同じ(done 4、want 3。`completed` も数える) |
| L2d | 文: `leader.md:848` の「(行の終わりの空白は問わない)」を消す | 落ちた | `Leader_XreviewCountsOnlyCompleteReports` だけ |
| L3a | 送り直しの上限を消す: `leader.md:1055-1059` の「同じ `<k>` で送り直すのは 2 回」から「止まる」までを消す | 落ちた | `Leader_XreviewHandlesEveryReply` だけ |
| L3b | 上限を 1 つ緩める: 「2 回(`-r2` と `-r3`)まで。`-r3` の TASK」→「3 回(`-r2`、`-r3`、`-r4`)まで。`-r4` の TASK」 | 落ちた | 同じ |
| L4a | jq の `unique` を外す(`leader.md:979`) | 落ちた | `Leader_XreviewDriverCommandReadsTheRecord` の two implementers on claude(bash・zsh) |
| L4b | `select(.role == "implementer")` → `select(.seat_id == "implementer")` | 落ちた | `…DriverCommandReadsTheRecord` の an implementer with no role yet と implementers on two drivers、`…DriverCommandReadsTheCLIsFields` |
| L4c | `2> "$p-status.err"` → `2> /dev/null`(`leader.md:978`) | 落ちた | `…DriverCommandReadsTheRecord` の status.err の確かめ(bash・zsh) |
| R1a | `@{` を断らない(雛形だけ): `reviewer.md:168` の許す文字に `` `{` `` を足す | 落ちた | `Reviewer_XreviewBaseTakesPlusAndAt`、`Reviewer_XreviewTaskCarriesItsLinesInTheBody`(どちらも文の固定) |
| R1b | 同じ(雛形だけ): `reviewer.md:169-171` の「(`{` を許さないので `@{` も入らない。…revision の式になる)」を消す | 落ちた | 同じ 2 本 |
| R1c | R1a を雛形とテストの定数 `reviewerXreviewBaseRule` の両方に入れる | 残った | なし。表の行 `origin/a@{1}` は `}` でも断られる(下の probe) |
| R1d | 雛形と定数の両方から「`..` を含むか」を消す | 残った | なし。`reviewerXreviewBaseOK` は `..` の規則を文から読まず、コードに持つ |
| R1e | 雛形と定数の両方から「`origin/` のすぐあとが `-` か」を消す | 残った | なし。`-` の規則も同じくコードに持つ |
| R1f | `{` と `}` の両方を雛形と定数に足す | 落ちた | `Reviewer_XreviewBaseTakesPlusAndAt`(`origin/a@{1}` = true、want false) |

雛形だけを変えた 13 個はすべて落ちた。雛形とテストの定数を一緒に変えた 4 個(R1c〜R1f)では R1f だけが落ちた。コマンドの変異(L1b、L2a〜L2c、L4a〜L4c)はスタブで走らせる行が落とし、文だけの変異(L1a、L2d、L3a、L3b、R1a、R1b)は文を固定するテストだけが落とす。F6 の送り直しの上限、`XREVIEW_BASE:` の規則、相対パスの規則は、モデルが読む文なので、これは予想どおり。

残った 3 個は、雛形とテストの定数を一緒に変えたもの(規則を変える人が両方を直す場合)。R1c の理由は、overlay で足した使い捨てのテストで確かめた: R1c の定数から `reviewerXreviewBaseOK` が読む許す文字は `._/+@{-` になり、`origin/a@{1}` は `}` で false を返す。

#### Test gaps(run 2 で見つけたもの)

1. `reviewerXreviewBaseOK`(`internal/org/prompts_xreview_reviewer_test.go:57-74`)が規則の文から読むのは許す文字の一覧だけで、`origin/` の直後の `-` と `..` はコード(`:65`)に持つ。表の行 `origin/a..b` と `origin/-x`(`:1217-1218`)は helper を確かめるだけで、規則の文とはつながらない(R1d、R1e が残った)。雛形だけの変更は文の固定(`:205-207`、`:1207`)が落とす。直すなら、この 2 つの規則も定数の文に句があるときだけ当てる
2. 表の行 `origin/a@{1}`(`:1216`)は断る文字を `{` と `}` の 2 つ持つので、`{` だけを許す規則(R1c)でも false になる。規則が名指しする `{` の除外は、`origin/a{b` のように 1 字だけ外れた行がないと固定されない。`}` のない `@{` は git の revision の式にならないので、実害は小さい
3. driver のコマンドを走らせる行は、手で作った JSON(`leaderXreviewStatusJSON`、`internal/org/prompts_xreview_leader_test.go:1311`)とスタブの ralph(`:1291`)だけを使う。本物の `ralph org status --json` が止まった implementer の `role` と `driver` を出すことは、コードを読んで確かめた(`internal/org/verbs.go:1183-1186` が `stopped` に座席の Role と Driver を書き、`Roster`(`internal/org/manifest.go:206-210`)が最後の状態のイベントの値を出す)が、この diff のテストでは確かめていない。`…ReadsTheCLIsFields`(`:1438`)は `internal/cli/org.go` の struct tag を文字列で探すだけで、`TestOrgStatus_JSON`(`internal/cli/org_test.go:987`)は `seat_id` しか見ない。`ralph org status` の `--org-id`・`--state-dir`・`--json` の組み合わせも、スタブが引数の形を確かめるだけ
4. 送り直しの上限(`internal/org/prompts/leader.md:1055-1059`)と、所見のパスを相対で書く規則(`internal/org/prompts/reviewer.md:364-368`)は文だけで、L3a・L3b は文の固定だけが落とす。leader が report をコミットする前に、表に絶対パスがないかを確かめるコマンドはない(`leader.md:1018-1027` の確かめは REPORT のパスの形と番号だけ)。run 1 の gap 1 と同じ種類

run 1 の gap 1〜5 は残る。gap 2 の「`XREVIEW_BASE:` の形は手順 1 の文だけ」は F6 のあとも同じ(F6 は文の規則を変えただけで、コマンドの側は変わらない)。

#### Verdict(cycle 1 の run 2)

- Verdict(cycle 1 の run 2、HEAD 17fc2b52): pass
- Pass: `go test -count=1 ./internal/org/`(ok、1,096.4 秒)、`go test -race -count=3 ./internal/org/ -run Xreview`(トップレベル 120、サブテスト 252、DATA RACE 0)。handoff の 4 つの変異(L1a・L1b、R1a・R1b、L3a、L2a)はすべて落ちた。変異 17 個では 14 個が落ちた
- Fail: 0
- Blocked: 0
- 残った変異: R1c・R1d・R1e(雛形とテストの定数を一緒に変えたもの。Test gaps(run 2)の 1 と 2)。計画の判定は変えない

### Run 1(HEAD 15da7eb0)

- Scope(run 1): `git diff origin/main...HEAD`(base 4fef6b8c、HEAD 15da7eb0)。Go は `internal/org`(雛形 `prompts/reviewer.md` と `prompts/leader.md`、新しいテスト 3 ファイル)、`internal/org/driver`(警告の文)、`internal/cli`(警告のテスト)、`internal/config`(コメント)。shell は `scripts/xreview-helpers.sh`(コメント)。ほかに skill とその写し、文書。テストの関数はこの diff で 38 本増え、消えたものはない。テストの途中(10:10 JST)に afccca76 がブランチに入ったが、変えたのは `docs/evidence/xreview-live-probe-2026-11-01.md` だけで、このファイルを読むテストはない(`grep -rn xreview-live-probe internal/ tests/ scripts/` が 0 件)。結果は afccca76 のコードにもそのまま当てはまる
- Evidence(run 1): `docs/evidence/test-2026-11-01-org-cross-review.md`(要点。commit する)、`docs/evidence/test-2026-11-01-org-cross-review.log`(gitignore。全出力と変異の差分)。wrapper 自身の写しは `docs/evidence/verify-2026-11-01-005214.log`(gitignore)

#### Test execution

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

##### Test plan との対応

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

##### 変異

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

#### Coverage

- Statement: `internal/org/driver` はパッケージ全体で 95.3%。この diff で中身を変えた Go の関数は `paneEnvConsequence`(`internal/org/driver/paneenv.go:64`)だけで、100%。`spawn.go` と `config.go` はコメントだけ
- Branch: 計測なし(Go の道具は statement だけ)
- Function: 上のとおり
- Notes:
  - 変更の中心は Markdown の雛形で、statement の計測には出ない。雛形のコマンド(reviewer の手順 2 と 3、leader の状態のコマンド)はスタブで走らせる行が分岐を打ち、変異 8 個がそれを確かめた。モデルが読んで判断する文(reviewer の手順 1 と 4〜7、leader の送る・送らない・QUESTION の分け方)は文の固定だけ
  - `internal/org` と `internal/cli` のパッケージ全体の数字は、この回は取っていない

#### Failure analysis

| Test | Error | Root cause | Proposed fix |
| --- | --- | --- | --- |
| なし | — | — | — |

#### Regression checks

| Previously broken behavior | Status | Evidence |
| --- | --- | --- |
| 既存の雛形のテスト(手順の 1 行目、sync-docs の順、計画助言の小節の位置、gate の停滞の文。この diff で `prompts_test.go`、`prompts_leader_syncdocs_test.go`、`prompts_leader_plan_test.go`、`skill_doc_followup_test.go` を直した) | pass | `run-test.sh` の `internal/org` ok |
| 負荷で bash のジョブの通知が出力の行の間に入って落ちた 2 か所(S1c で行ごとの確かめに直した) | pass | `-race -count=3` の 3 回と `run-test.sh` の回で `…CommandOutcomes` と `…CallsTheOtherModel` が通った(load 6.5〜14.6) |
| `pick_reviewer` の大文字小文字と既定値(`tests/test-xreview-helpers.sh`) | pass | sh・bash・`zsh --emulate sh -f` で 53/53 |
| paneenv の警告に「座席の cross-review」の文が戻らないこと | pass | driver の 13 本、cli の 2 本 |

#### Test gaps

1. leader が返事の `REPORT:` のパスを確かめてからコミットする手順(`internal/org/prompts/leader.md:996` からの段落。形と、TASK_ID の `c<k>` と同じ番号か)は文だけで、走らせるコマンドがない。固定しているのは `TestRenderRolePrompt_Leader_XreviewChecksTheValuesItReads` の文(`internal/org/prompts_xreview_leader_test.go:1172-1183`)だけ。M2b と同じく、文を変えても落とすのは文の固定だけになる
2. reviewer の手順 1 の値の確かめ(`internal/org/prompts/reviewer.md:165-183`。`XREVIEW_BASE:` の形、`REPORT:` の形、`REVIEWER_*` の文字)も文だけ。コマンドの側で効くのは、手順 2 の一重引用符の `case`(`reviewer.md:196`)と、claude の `git diff` の `--end-of-options`(`reviewer.md:285`)だけ。codex のコマンド(`reviewer.md:254`)は `--base "<XREVIEW_BASE>"` と `-m "<REVIEWER_CODEX のモデル>"` を二重引用符で埋め、コマンドの中では確かめない。スタブの行が codex のコマンドに渡すのは形の正しい値だけ(`internal/org/prompts_xreview_reviewer_test.go:707-716`)。手順 2 の `case` の行(`:977-979`)にも一重引用符を含む値の行はない。範囲の外なので、それ以上は探っていない
3. スタブで走らせるのは bash 3.2.57 と `zsh -f` 5.9 だけ(`advisoryShells`、`internal/org/prompts_leader_plan_test.go:1049-1064`)。dash や POSIX の sh では、leader の状態のコマンドも reviewer のコマンドも打っていない。座席はどちらも自分の CLI の shell(bash か zsh)で打つので、影響は小さい
4. leader が状態の値をもとに分ける部分(TASK を送るか、上限の QUESTION の 1 つ目か 2 つ目か、cap のファイルに書く数)はモデルの判断で、文の固定(`…ChecksTheCapBeforeTheTask`、`internal/org/prompts_xreview_leader_test.go:246-259`)だけ。`…RaisedCapReachesTheTask` は規則を手で当てた数を cap のファイルに書き、状態のコマンドがそれを読むことだけを確かめる(docstring のとおり)
5. 座席の中(権限、sandbox、network)での実行は /test の範囲の外で、計画の known gap(AC9、Risks)のまま。本物の CLI での実行は `docs/evidence/xreview-live-probe-2026-11-01.md` の 2 回(2 回目は afccca76 で 62d62be0 の版)にあり、この回では打っていない

#### Verdict(run 1)

- Verdict(cycle 1、HEAD 15da7eb0。afccca76 は evidence の文書だけ): pass
- Pass: `./scripts/run-test.sh`(rc 0、shell 45 本、Go 10 パッケージ)、`-race -count=3` の `Xreview`(DATA RACE 0)、`test-xreview-helpers.sh` の sh・bash・`zsh --emulate sh -f`、paneenv の警告のテスト、変異 9 個はすべて落ちた
- Fail: 0
- Blocked: 0
