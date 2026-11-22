# sync-docs report: issue-depends-wait

- Date: 2026-11-22
- Plan: `docs/plans/active/2026-11-22-issue-depends-wait.md`(承認の digest `7b44cea2d6bc`。この run の前後とも同じ)。issue #364
- Branch: `feat/issue-depends-wait`。run 開始時の HEAD は `63ad5152`(origin/main は `2a069b50`、取り込み済み)
- Pipeline run: 1 回目(`cycle-count.json` は `cycle` 1)
- 先行 report(引いた SHA は各 report を含む commit。report の中の `HEAD:` の行の値ではない): self-review `c1ff3341`(Merge: yes、MEDIUM 1、LOW 6。M1・L1・L2・L3・L5 は `03ced266` で直った。L4・L6 は台帳へ)、verify `95d755b2`(pass、HEAD 03ced266、LOW 2・INFO 4)、test `63ad5152`(pass、`run-test.sh` rc 0、変異 10 個がすべて落ちた)
- 制約: `./scripts/run-test.sh` は回していない(依頼の指示)。skill は触っていないので `sync-skills.sh` は要らなかった
- 差分の大きさ(`/pr` が walkthrough を決める材料。この report の commit の前の数): `git diff origin/main...HEAD` は 16 ファイル、+1,882/-92。Go は 2 ファイルだけで、どちらもテスト(`skill_issue_depends_test.go` +607、`skill_split_questions_test.go` +18/-12)。テスト以外の Go は 0。walkthrough を書くかどうかは `/pr` が決める

## Summary

仕様 `2026-10-07-org-multi-org-director.md` の 2026-11-22 の項(173 行目)を、`03ced266` が決めた skill に合わせた(verify V1・V2)。台帳 `docs/tech-debt/README.md` に新しい行を 2 つ足した(561・562 行目)。台帳の Trigger は全行を照合し、この PR が満たす行は 0 だった(verify の結論と同じ)。`followup add --help` と `/org` のコマンド表の一般の規則は直さなかった(V5、理由は下)。README と AGENTS.md は起票の待ちを書いていないので直していない。計画は Progress の行と AC1〜AC7 の印だけを足し、本文は変えていない。

## Changes made

| File | Change | Commit |
|------|--------|--------|
| `docs/specs/2026-10-07-org-multi-org-director.md` | 173 行目の 2026-11-22 の項の 2 か所。(1) 「`feature` だけが合うときは人に聞く。…同じ数え方にした」を、聞く場所と回数(起票の待つブランチを決める前に 1 回)、選択肢(「新しい merge を待つ」が推奨、「この merge と数える」)、答えが `after:` に残ること、follow-up のあるブランチは引き継ぎでも着手を聞く前の判定でも聞き直さないこと、引き継ぎで聞くのは follow-up のないブランチ(起票の記録が途中で止まった場合)の 1 回だけで `merged` なのに `feature` だけが合うものに限ること、merge 済みと数える確かめは起票と同じ (a) か (b) であること、に書き直した。(2) 「この場合だけは、今までどおり director が印の行を書き換える」を、書き換える 2 つの場合((1) 頼る先を start するときに slug か Type を変えたとき、(2) 引き継ぎで「この merge と数える」と答えたとき。どちらも「merge を待つ issue」の 3 の手順)に直し、続く `/org` skill の文の「この例外」を「頼る先のブランチの例外(前の別の分割計画の merge は数えず、記録して待つ)」に言い換えた | この report の commit |
| `docs/tech-debt/README.md` | 新しい行 2 つ(561・562 行目、末尾に追記)。561: 「merge を待つ issue」の 3 の小さな所見(self-review L4 の (a)(b)(c)、verify V3 の (d))。562: この PR のテストの穴(self-review L6、test の Test gaps 2)。既存の行は変えていない | この report の commit |
| `docs/plans/active/2026-11-22-issue-depends-wait.md` | Progress の `Test artifact created` の次に `Sync-docs artifact created` の行(`[x]`)。AC1〜AC7 に `[x]`(verify と test がすべて Met とした)。本文の文は変えていない。digest は前後とも `7b44cea2d6bc` | この report の commit |
| `docs/reports/sync-docs-2026-11-22-issue-depends-wait.md` | この report | この report の commit |
| `docs/insights/events/2026-11-22-issue-depends-wait.jsonl` | `sync_docs` の行(cycle 1、pass)。3 行から 4 行(`self_review`・`verify`・`test` は揃っていた) | この report の commit |

Go のコードにも skill にも触れていない。パイプラインの再実行になる変更はない。

## 依頼の項目

### 1. 仕様の 2026-11-22 の項(V1・V2)

verify V1 が指した「報告のたびに聞くとも読める」文は、起票で 1 回だけ聞く形に書いた。V2 が指した 2 つ目の書き換え(引き継ぎで「この merge と数える」と答えると、そのブランチを `gh issue edit` で `after:` から除く)は、書き換える場合の一覧に入れた。skill のどの文に当たるかは次のとおり(HEAD の `.claude/skills/director/SKILL.md`)。

| 仕様の文 | skill の根拠 |
|------|------|
| 起票で待つブランチを決める前に 1 回聞く。選択肢は「新しい merge を待つ」(推奨・先頭)と「この merge と数える」 | 「起票と記録」の 1 の「頼る先の merge の確かめ」の後ろの段落 |
| 答えは `after:` に残る。follow-up のあるブランチは聞き直さない | 同じ段落の「答えは印の行の `after:` に残る」、「merge を待つ issue」の 2 の「follow-up のあるそのブランチは、`feature` だけが合っても聞かない」 |
| 引き継ぎで聞くのは follow-up のないブランチだけ | 「merge を待つ issue」の引き継ぎの「ただし、follow-up のないそのブランチが…聞くのはこの場合だけ」 |
| 書き換えは 2 つの場合 | 同じ節の 3 の、slug か Type を変えたときの移す手順と、引き継ぎの「この merge と数える」の書き換え(「3 の書き換えの手順で…`after:` からそのブランチだけを除き」) |

計画の Assumptions(例外は 1 つ)と Scope の 4 つ目の項目(引き継ぎと 2 でも聞く)は、計画の本文なので直していない(本文を直すと digest が動く)。ずれは verify V4 が Progress に書いており、PR の本文の Known gaps にも入れてほしい(下)。

### 2. 台帳の行

| 行 | 内容 | 出どころ |
|------|------|------|
| 561 | 「merge を待つ issue」の 3 の 4 点。(a) 移す手順が、人が答え済みの issue も選び、新しいブランチの merge で同じ人に聞き直す。(b) start が worktree の記録で断ったときの案内が、slug を変える案を先に置き、その費用(待つ側の全 issue を書き換える)を言わない。(c) 元の機能の Type を本文の `- Type:` の行から読むが、行の数と値(`splitFeatureTypes`)を確かめない。(d) `/org` skill の文が「起票のときに 1 回だけ」と言い、`/director` skill は引き継ぎでも 1 回聞く | self-review L4、verify V3 |
| 562 | この PR のテストの穴 2 つ。(1) `TestDirectorSkill_DependencyBranchIsRecordedBeforeItExistsAndWaits` の後半が、既存の 2 つのテスト(`TestMergeOrderFollowup_StateComesFromTheAfterBranchsPRs` の「no PR is waiting」の行、`TestOrgMergeOrder_FollowupSectionFollowsTheNotes` の `feat/none` の行)と同じことを固定し、前半の `.git` の確かめは `t.TempDir()` の直下しか見ない(CLI の経路も通らない)。(2) `directorDependsAskingSentences` が「`feature`だけが合」と「聞く」を両方含む文だけを数えるので、別の言い方の問いを足しても落ちない(変異 D3 は同じ言い方なので落ちる) | self-review L6、test の Test gaps 2 |

どちらも、Debt の 1 列目に原文の ID(L4、V3、L6、Test gaps 2)を書き、Trigger に直し方を、Related に計画を `docs/plans/active/...` の path で足した(`/pr` の archive が書き換える)。テストの場所は `file:line` ではなくテストの名前で引いた(L6 の `mergeorder_followup_test.go:197` と `internal/cli/org_merge_order_followup_test.go:156` は、それぞれ `TestMergeOrderFollowup_StateComesFromTheAfterBranchsPRs` と `TestOrgMergeOrder_FollowupSectionFollowsTheNotes` の中)。

self-review の「Tech debt identified」のもう 1 行(L1: 確かめの文が 5 か所に分かれる)は、`03ced266` が規則を 1 か所に置いたので(`TestDirectorSkill_DependencyMergeQuestionIsAskedOnceAtFiling` が規則の文を 1 回だけ数える)、行にしていない。

### 3. 台帳の Trigger(照合)

依頼は「verify は満たす行なしと言った。確かめて、あれば注を足す」だった。台帳の全行(取り消し線の文を除いた Trigger のセル)を、この diff が触れたもの(director skill と org skill の節の名前、2 つのテストファイル、`起票と記録`、`merge を待つ issue`、`after:`、印の行、`follow-up`、`merge-order`、`Depends on`)で 2 回探した。1 回目は名前の一致で 71 行、2 回目は話題の一致で 44 行が候補になり、約 60 行の Trigger の頭を読んだ。**満たす行は 0**(注は足していない)。

| 行(最初の語) | Trigger | 当てない理由 |
|------|------|------|
| 267 `Three test gaps of the org-director-attach PR` | `/director` skill の `受信箱の処理` の節の次の変更 | 触った節は `### 人への聞き方`・`### 起票と記録`(どちらも `## 分割計画と機能ごとの org`)と `### merge を待つ issue`(`## 進捗の報告`)。`## 受信箱の処理`(992 行目から)の中ではない |
| 274 (b) | `director_skill_sync_test.go` に次のテストを足す、`始める` の 3 | そのファイルは触っていない。新しいテストは新しい `skill_issue_depends_test.go` |
| 395 `Four test files went past the 800-line guide` | 4 つのファイル(`org_merge_test.go`、`merge_run_test.go`、`director_skill_sync_test.go`、`cleanup_test.go`)の次の変更 | どれも触っていない。`skill_split_questions_test.go` は 4 つに入っていない |
| 426 `internal/org/skill_split_questions_test.go is over 800 lines` | このファイルに次のテストを足す、節の読み手を変える | 6 つの既存のテストの fragment を直しただけ(+18/-12)。テストも読み手(`directorSkillIntro` ほか)も足していない。行は「count は書かない」と断っているので、古い `wc -l` もない |
| 427 `The --limit value of the two gh issue list commands` | `--limit` の次の変更、そのファイルに次のテストを足す | `--limit 1000` は変えていない(diff の hunk の見出しに文脈として出るだけ)。テストを足していない |
| 470 `~~The /director skill ... scaffold version note~~` | `/director` skill の次の変更 | 1 つ目のセルが取り消し線で、2026-11-13 に docs/director-notice-handling が閉じた行(最後の注は 11-15)。閉じたあとの director skill の変更に注は足されておらず、#365 の sync-docs も閉じた行として当てていない |
| 477、509、512、513、385、386 | gate accept、知らせの行の小節、hook QUESTION の項目、send の確かめの手順、wait の終了コード | その節も項目も触っていない |
| 182 `Two documentation findings of org-stop-all` (a) | `/org` skill のコマンド表の次の変更 | 表は触っていない。足した 1 文は表の下の `followup` の小節(`PR がもう merge されたブランチは follow-up にしない` の項目)にある。F-8 の `stop`・`disband` のセルの長さも変わらない |
| 382 `RC4` | `/org` skill の分割計画の例、`leader.md` の手順 | どちらも触っていない |
| 416、255 | 予約外の文書の follow-up の小節、origin の ssh の確かめ | その小節も確かめも触っていない |
| 547、555 (b)、557 (2) | `feature:` の行、`共有のパスの直し方` の手順 1、`--issue` の実地 | `feature:` の行も手順も変えていない(skill は `issue:` の行を読む側で、行を変えていない)。実地は報告待ち |
| 252、259、311、423、424、247〜249、241 | `mergeorder*.go`、`FollowupStore`、`gate.go` の次の変更 | Go の本体は触っていない |
| 558 `The tech-debt rows whose Trigger a branch meets are found by hand` | `/self-review`・`/verify`・`/sync-docs` が行を見落とす | 見落としはない(verify と今回の照合が同じ結論) |

### 4. `followup add --help` と `/org` の表(V5)

直さなかった。理由は 3 つ。(1) help の文は「記録より前の merge は数えない。そのため、PR がもう merge されたブランチは follow-up にせず、issue をすぐ聞く」で、「PR がもう merge された」は follow-up が待つその PR を指す。頼る先の同じ名前の前の PR は待つ PR ではなく、待つのは次の PR なので、help の文と食い違わない。(2) 例外は `/org` skill の同じ小節に 2 つとも書いてある(答えの記録と、頼る先のブランチ)。答えの記録の例外も、この PR の前から表と help にはなかった。(3) help を変えると `internal/cli` の Go の変更になり、pin されたテストと台帳の Trigger を引く。得るものは小さい。director が読む手順は `/director` skill にある。

### 5. README と AGENTS.md

直していない。README の follow-up の記述(134・294 行目)は CLI の振る舞いで、この PR は変えていない。AGENTS.md の `internal/org/` の行は follow-up の記録と merge-order の読みを述べ、skill の手順もテストのファイルも挙げない。`docs/recipes/`、`docs/quality/`、`.claude/rules/ralph/` に `ralph-split` や `merge を待つ issue` を引く所はない(`git grep` で `docs/evidence/` の履歴だけが当たる)。

### 6. 検査

- `./scripts/check-sync.sh`: `PASS: all files in sync.`(KNOWN_DIFF 6)
- `./scripts/check-skill-sync.sh`: 14 skill そろっている
- `go test -count=1 ./internal/org/ -run 'Skill|DirectorSkill|DocsMatch|Followup'`: ok
- `go test -count=1 ./internal/cli/ -run 'Followup|Help|Skill|Docs'`: ok
- 台帳の plan 参照(`verify.local.sh` の `check_tech_debt_plan_refs` と同じ検索): 存在しない参照は 0 件
- 台帳: 新しい 2 行は、5 セル(`(?<!\\) \| ` で割る)、セルごとのバッククォートが偶数、`~~` なし、`file:line` なし、U+FFFD なし。`git show HEAD:` と比べて、変わった既存の行は、最後の空行(新しい行 1 つに置き換わる)だけ
- 計画の digest: `7b44cea2d6bc`(前後とも)。`git diff --numstat` は +9/-7(AC の 7 行の印と Progress の 2 行)

## handoff と実際の違い

- 依頼は L4・L6・Test gaps 2 の 3 点だった。verify V3(INFO、org skill の「起票のときに 1 回だけ」)は、同じ小節の文なので 561 の (d) に入れた。落としてよい
- 依頼は仕様の「この場合だけ」を「2 つの場合」にすることだった。続く文の「この例外」(`/org` skill の規則に足した例外を指す)が、書き換えの 2 つの場合と読めるようになったので、「頼る先のブランチの例外」に言い換えた
- Trigger を満たす行は 0 で、注は足していない。台帳の既存の行は変えていない

## PR の本文の Known gaps に入れるもの

1. director が skill の手順どおりに動く run がない(test の Test gaps 1)。起票で 1 回だけ聞くこと、まだないブランチの follow-up を記録して `waiting` で待つこと、頼る先の PR の merge で待つ側が `ready` になり着手を聞くことは、skill の文とそれを固定するテスト、fixture の `FollowupAdd` と merge-order の `waiting` までで、実機の director では追っていない
2. 計画の本文と、仕上がりの skill が 2 点で違う(計画の本文は digest で守られるので直していない)。(a) Scope の 4 つ目の項目は「引き継ぎと 2 でも `feature` だけが合えば人に聞く」だが、`03ced266` で起票の 1 回だけになり、引き継ぎと 2 が聞くのは follow-up のないブランチだけ(verify V4。仕様の 2026-11-22 の項に今の形を書いた)。(b) Assumptions は印の行を director が書き換える例外を 1 つ(slug か Type を変える場合)と書くが、引き継ぎの「この merge と数える」の書き換えが 2 つ目(verify V2。仕様に 2 つとも書いた)。受け入れ条件 3(印の行を手で書き換えなくてよい)の例外が 2 つあることを PR の本文に書く
3. 「merge を待つ issue」の 3 の移す手順は、人が答え済みの issue も選び、slug を変える案を費用の説明なしに先に出し、`- Type:` の行の数と値を確かめない(self-review L4、台帳 561 の (a)(b)(c))
4. テストの穴: `TestDirectorSkill_DependencyBranchIsRecordedBeforeItExistsAndWaits` の後半が既存のテストと重なる、問いの禁止の検査が別の言い方を拾わない(self-review L6、test の Test gaps 2、台帳 562)。fragment の照合は固定した文のあいだに足した矛盾する文を拾わない(Test gaps 3、skill の文を固定するテストに共通)。仕様の日付つきの項を固定するテストはなく、V1・V2 は `03ced266` のあとも残った(Test gaps 4。今回は文書を直したが、固定はない)
5. `followup add --help` と `/org` のコマンド表は「merge 済みのブランチは follow-up にしない」を一般の規則のまま言う。例外は `/org` の小節にある(verify V5、上の理由で直していない)
6. (b) の `binding.issue` が実際の `ralph org start --plan --issue` で記録されることは、#365 の単体テストが固定している。実機の確かめはない(verify の Coverage gaps)

## Insight event

`./scripts/insights-append.sh --slug issue-depends-wait --flow standard --phase sync_docs --verdict pass --cycle auto --source skill` で 1 行(cycle 1、pass)を足した。rc 0。`git status --short` は既存の jsonl の `M` だけで、迷子のファイルはない(`--slug` に日付を付けていない)。UTC の日付が 11-22 なので `self_review`・`verify`・`test` の行と同じファイル。

## 実行結果

- 仕様の Edit は 3 回(2 か所と、1 か所の言い直し)。Edit は一意でない文を断るので、どれも置き換える前の文は 1 回だけだった
- `git status --short`: 変更は 4 ファイルの `M`(jsonl、計画、仕様、台帳)とこの report の `??` だけ。skill と Go は変わっていない
- この report の commit は `docs: sync docs for issue-depends-wait` の 1 つ。push はしていない
