# Verify report: issue-depends-wait

- Date: 2026-11-22
- Plan: `docs/plans/active/2026-11-22-issue-depends-wait.md`(issue #364。承認の digest `7b44cea2d6bc` は今の本文から計算した値と一致)
- Verifier: verifier サブエージェント(`/verify`)。パイプライン 1 回目(`cycle-count.json` は cycle 1)
- Scope: origin/main `2a069b50` から HEAD `03ced266` までの `git diff origin/main...HEAD`(14 ファイル)。self-review の対象(`2407bb35` まで)に、そのあとの直し `03ced266`(M1・L1・L2・L3・L5)を加えた範囲。テストは実行していない(`/test` の担当)
- Evidence: `docs/evidence/verify-2026-11-22-issue-depends-wait.log`(git の管理外。静的検証の全出力、4 写しの md5、digest、コードの名前の grep、既存テストの行、仕様の該当文)

## Spec compliance

行番号は `.claude/skills/director/SKILL.md` と `.claude/skills/org/SKILL.md` の HEAD のもの。

| Acceptance criterion | Status | Evidence |
| --- | --- | --- |
| AC1: 「起票と記録」の 1 が、重なる org のブランチに加えて頼る先ごとに、走っている org のブランチか分割計画の `<Type>/<slug>` を待つと言う(merge 済みは待たない)。テストで固定 | Met | director 704-781。走っている org は `feature:` の行と `issue:` の行が (a) か (b) に当たる org だけで、slug だけ合う org は走っていない扱い。走っていなければ `<Type>/<slug>`(`- Type:` がなければ `feat/<slug>`)。待たないのは確かめで数えたときと、人が「この merge と数える」と答えたときだけ。ブランチ名はコードの `feature.go:449`(`Type + "/" + Slug`)、既定の Type は `split.go:99`・`:718` の `feat` と合う。固定は `TestDirectorSkill_FilingWaitsForTheBranchesOfTheFeaturesItDependsOn`(1 の fragment、問いの行のバイト一致、`defaultSplitFeatureType`、`MergeClassMerged`) |
| AC2: 同じ節の 4 と引き継ぎが、まだないブランチの follow-up を記録し `waiting` のまま待つと言う | Met | director 809-838(4)、2516-2528(引き継ぎ)。コードでは `validateFollowupBranch`(`followup.go:239`)が形だけを見て、CLI の `followup add`(`internal/cli/org_followup.go:84-105`)は検査のあと `store.Add` を呼ぶだけで git も gh も読まない。PR のないブランチは `mergeorder_followup.go:315` で `waiting`、DETAIL は `<branch> has no PR yet: ...`。既存テスト `mergeorder_followup_test.go:197` と `internal/cli/org_merge_order_followup_test.go:156` が固定済み |
| AC3: 「merge を待つ issue」の 3 が、Type と slug を変えないこと、変えたら待つ issue を移すことを言う | Met | director 2658-2674。固定は `TestDirectorSkill_RewriteKeepsTheTypeAndSlugOrMovesTheWaitingIssues`。移す issue の選び方が広い点は self-review の L4 で、台帳へ回す |
| AC4: #364 の例で、待つ側の `after:` に頼る先の `<Type>/<slug>` が入り、頼る先の merge で follow-up が `ready` になる流れを追える | Met | director 843-888。印の行は issue ごとの見出しの下に枠 1 つずつ。`locks` を同じ slug と Type と `--issue` で立て、その PR の merge で `api` の follow-up が `ready` になる流れと、前の別の分割計画の同名ブランチがあるときの問いと答えごとの結果を書いている。`ready` への移り方はコードの「記録より後に merge された PR があれば ready」(`mergeorder_followup.go:306-307`)と合う。固定は `TestDirectorSkill_IssueDependsExampleFollowsAFiledDependency`(印の行の正規表現、`ValidateFollowupAdd`、`#` のあとの番号が #364 だけ) |
| AC5: `check-sync.sh` と `check-skill-sync.sh` が通り、director skill の 4 写しが同じ | Met | 静的検証の中で check-sync は PASS(DRIFTED 0)、check-skill-sync は「14 skill(s) in lock-step」。md5 は director の 4 写しが `7bd283f9ccb33ec533e7746482b5a2d1`、org の 4 写しが `f6fdd1b5bed6a4c24f42687a293dc2fa` で、それぞれ同じ |
| AC6: 前の別の分割計画の同名ブランチが merge 済みでも、`binding` の (a) か (b) に当たるときだけ merge 済みと数え、`feature` だけ合うときは人に聞く。(b) も数える。テストで固定し、JSON の `binding` のキーは既存のテストで確かめる | Met | director 731-775 に確かめの規則を 1 回だけ書き、4・引き継ぎ・2・org skill は名前で呼ぶ(規則の文が skill 全体で 1 回だけ出ることを `TestDirectorSkill_DependencyMergeQuestionIsAskedOnceAtFiling` が数える)。`feature` だけ合うときは起票の 1 で 1 回聞く(選択肢 2 つ、推奨が先頭、「人への聞き方」の列挙に入った)。引き継ぎと 2 は follow-up のあるブランチでは聞かない(V3・V4 参照)。JSON のキーは既存の `mergeorder_test.go:3233`(`feature`・`org_id`・`split`・`ts`)、`internal/cli/org_start_issue_test.go:391-404` と `mergeorder_issue_test.go:61`(`issue`)が固定し、新しい `checkMergeOrderJSONNames` も構造体のタグと突き合わせる。コード側のタグは `mergeorder.go:238`・`:253`・`:256`・`:279`・`:326`・`:327`・`:340` |
| AC7: 「起票と記録」の 2 が本文に `- Type:` を書くと言い、「merge を待つ issue」の 3 が本文の `- Type:` から読む | Met | director 782-796(2、分割計画になければ `- Type: feat`)、2658-2661(3、本文の行、なければ分割計画の機能の値、それもなければ `feat`)。どちらも fragment で固定 |

## self-review の直し(`03ced266`)の確かめ

| ID | 直ったか | 確かめたこと |
| --- | --- | --- |
| M1 | 直った(推奨から 1 点ずれる) | 問いは起票の 1 の「待つブランチを決める前」に 1 回(director 752-775)。選択肢は「新しい merge を待つ」(推奨・先頭)と「この merge と数える」で、問いの文は 1 行の枠。「人への聞き方」の列挙に入った(521-527)。2 は follow-up のあるブランチでは聞かない(2579-2582)。例の最後は答えごとの結果を書く(879-888)。推奨との差は、follow-up のないブランチ(起票の記録が途中で止まった場合)だけ引き継ぎで 1 回聞く例外を足したこと(2482-2496)。答えは `after:` に残るので同じブランチで 2 回は聞かない |
| L1 | 直った | 「頼る先の merge の確かめ」の名前で規則を 1 か所に置き、ほかは名前で呼ぶ。(b) の番号の読み方(待つ側の `頼られる issue:` の行、起票の途中は 3 で出た番号)、起票していない機能は (a) だけ、slug だけ合う走っている org の扱い、頼る先でない同じ分割計画の機能に (a) を使うことを書いた |
| L2 | 直った | 番号を `<locks の番号>` の形にし、印の行を見出しつきの枠 2 つに分けた。テストが `#` のあとの番号を #364 だけに限る |
| L3 | 直った | org skill 2436-2442 が (a)・(b) と「起票のときに 1 回だけ人に聞く」を言う(言い回しの差は V3) |
| L5 | 直った | 引き継ぎの知らせは「まだ人に伝えていないものだけを 1 回」で、頼る先の issue の番号と、頼る先が `after: none` なら人が着手を決めるまで進まないことを言う(2519-2528)。起票の 3 も番号と同じ注意を言う(802-808) |
| L4・L6 | 予定どおり未修正 | 3 の移す手順(2664-2674)とテストの後半(`skill_issue_depends_test.go:330-352`)は self-review の時点のまま。`/sync-docs` で台帳に入れる |

## Static analysis

| Command | Result | Notes |
| --- | --- | --- |
| `./scripts/run-static-verify.sh` | pass(rc=0) | changed scope で golang を選んだ。`./scripts/detect-languages.sh`(full scope の選び方)も golang だけを返すので、full scope でも同じ pack になる。hook の shellcheck と `sh -n`、settings.json の jq、Codex の hook の 3 つの guard、check-sync、check-pipeline-sync、check-skill-sync、check-template-purity、tech-debt の plan 参照、gofmt(ok)、golangci-lint(0 issues)、branch の secret scan(`2a069b50..03ced266` clean)がすべて通った |
| `./scripts/plan-visual.sh digest <plan>` | `7b44cea2d6bc` | 計画の `- Approved:` の行と一致。計画の本文は承認のあと変わっていない |
| `md5`(skill の 4 写し) | 一致 | director、org とも 4 写しが同じ値(AC5 の行) |

## Documentation drift

| Doc / contract | In sync? | Notes |
| --- | --- | --- |
| director skill(4 写し) | Yes | skill が引くコマンド名、JSON のキー(`entries`・`branch`・`class`・`binding.split`・`binding.feature`・`binding.issue`)、class `merged`、STATE `waiting`、DETAIL `has no PR yet`、`issue: #<番号>` と `feature:` の行(`internal/cli/org.go:2322`・`:2324`)、`followup list` の列名 `AFTER_BRANCH`(`org_followup.go:267`)、既定の Type `feat` は、すべてコードと合う |
| org skill(4 写し) | Yes(INFO V3) | 「merge を待つ issue」の例外の文は director と同じ (a)・(b) を言う |
| `docs/specs/2026-10-07-org-multi-org-director.md` 173 行(2026-11-22 の項目) | No(V1・V2) | `03ced266` より前に書いた文のまま |
| `ralph org followup add --help`、org skill のコマンド表の `followup add` の行 | Yes(INFO V5) | どちらも「merge 済みのブランチは follow-up にしない」という一般の規則で、例外は org skill の節に書いてある。この PR の前から、答えの記録の例外も表と help にはなかった |
| `README.md`、`AGENTS.md` | Yes | どちらも起票の手順と `after:` を書いていない。この PR は Go のコードを変えず、AGENTS.md はテストのファイルを挙げない |
| `internal/org/prompts/director.md` | Yes | 起票と follow-up の文はない |
| `docs/tech-debt/README.md` | Yes | Trigger の列を python で読んで調べ、この diff で満たす行はなかった。行 426(`skill_split_questions_test.go` に次のテストを足すか、節の読み手を変える)と行 427(同じファイルに次のテストを足す)は、この PR が fragment とコメントだけを直し、新しいテストを新しいファイルに置いたので満たさない。行 395 のファイルは触っていない。行 480(leader のタスクの `Depends on`)は leader のタスクを変えていないので当たらない。行 470 は解決済み。self-review の L4・L6 は `/sync-docs` で足す |
| 計画の Progress checklist | 補った(V4) | `03ced266` の直しが計画の Scope の文からずれたことの記録がなかったので、verify の行に短く書いた |

## Findings

| ID | Severity | Finding | Evidence | Hand-off |
| --- | --- | --- | --- | --- |
| V1 | LOW | 仕様の 2026-11-22 の項目は「`feature` だけが合うときは人に聞く。引き継ぎと着手を聞く前の判定(「merge を待つ issue」)も同じ数え方にした」と書き、`03ced266` の決めごとを持たない。今の skill は、起票の 1 で 1 回だけ 2 つの選択肢(「新しい merge を待つ」が推奨)で聞き、答えは `after:` に残り、follow-up のあるブランチは引き継ぎと 2 で聞かない。聞くのは follow-up のないブランチ(起票の記録が途中で止まった場合)だけで、引き継ぎで 1 回。今の文は、報告のたびに聞くとも読める | 仕様 173 行(evidence に該当の 2 文)、director 752-775・2482-2496・2579-2582 | `/sync-docs`: 173 行の項目に、問いを聞く場所と回数、選択肢、答えの残り方、引き継ぎの例外を足す |
| V2 | LOW | 同じ項目の「この場合だけは、今までどおり director が印の行を書き換える」(slug か Type を変えたとき)と、計画の Assumptions が挙げる例外 1 つが、`03ced266` で増えた 2 つ目の書き換えを含まない。引き継ぎで「この merge と数える」と答えると、そのブランチを `gh issue edit` で `after:` から除く | 仕様 173 行、director 2489-2496、計画 41 行 | `/sync-docs`: 仕様の「この場合だけ」を 2 つの場合に直す。`/pr`: Known gaps に 2 つ目の書き換えの経路を書く |
| V3 | INFO | 問いを聞く場所の書き方が 3 か所で少し違う。org skill は「director が起票のときに 1 回だけ人に聞く」、director の 1 は「例外は 1 つで、引き継ぎが follow-up のないブランチを記録するときに聞くことがある」、2 は「引き継ぎのとおり、記録の前に 1 回聞く」と自分の中でも聞き、「人への聞き方」は引き継ぎだけを挙げる。動作は同じ条件(follow-up のないブランチ、ブランチごとに 1 回)で、食い違いはない | org 2441-2442、director 523-524・770-773・2584-2587 | 任意。直すなら org skill の文に「記録が途中で止まったときは引き継ぎでも 1 回」を足す |
| V4 | INFO | 計画の Progress に `03ced266` の行がなく、M1 の直しが Scope の 4 つ目の項目(引き継ぎと 2 でも `feature` だけ合えば人に聞く)からずれたことが記録されていなかった。計画の本文は digest で守られているので本文は直さない | 計画 24 行・112-115 行 | verify の Progress の行に一文で書いた。`/pr` の本文にも書く |
| V5 | INFO | `ralph org followup add --help` の「a branch whose PR is already merged needs no follow-up」と、org skill のコマンド表の「PR がもう merge されたブランチは follow-up にせず」は一般の規則で、頼る先の同名ブランチの例外を書かない。例外は org skill の節にあり、計画の Progress も help は直さなくてよいと記録している | `internal/cli/org_followup.go:74-75`、org 399 | 任意 |
| V6 | INFO | 計画の受け入れ条件の印(AC1〜AC7)がすべて `[ ]` のまま。この report では 7 つとも Met | 計画 63-69 行 | `/pr` の前に付ける(印を付けても digest は変わらない) |

## Observational checks

- 振る舞いの観察はしていない。この PR は skill の文とテストだけを変え、Go の本体のコードは変えていない(`git diff origin/main...HEAD` の Go のファイルは `internal/org/skill_issue_depends_test.go` と `internal/org/skill_split_questions_test.go` だけ)
- skill の該当の節の文は evidence の代わりに上の行番号で示した

## Coverage gaps

- テストは走らせていない。新しい 7 本と直した fragment が通るかは `/test` で確かめる
- director が skill の手順のとおりに 1 回だけ聞くか、`after:` が実際の起票で頼る先のブランチを持つかは、実機の director で確かめていない(skill は LLM が読む手順で、決定的な検査はテストの fragment だけ)
- (b) の `binding.issue` が実際の `ralph org start --plan --issue` で記録されることは #365 の単体テストが固定している。実機の確かめはない
- 決定的な検査で足りないもの: 仕様の日付つきの項目を固定するテストはなく、V1・V2 はそのため `03ced266` のあとも残った。skill の側は、規則の文が 1 回だけ出ること、step 4 が聞かないこと、引き継ぎと 2 の問いが follow-up のないブランチだけに付くことをテストが数えているので、足すなら org skill の文が「起票のとき」以外の聞く場所を言わないことの扱い(V3)を決めてから、その fragment を 1 つ足すのが最小

## Verdict

- Result: pass(LOW 2、INFO 4。どれも merge を止めない)
- Verified: AC1〜AC7 を skill の文、コードの名前と挙動、既存のテストと照らした。静的検証は rc=0、skill の 4 写しは director と org の両方で同じ、計画の digest は一致、self-review の M1・L1・L2・L3・L5 の直しが入ったこと
- Partially verified: AC4 の流れ(skill の文と、`mergeorder_followup.go` の ready の判定までは確かめた。実機の起票と merge は通していない)
- Not verified: テストの実行(`/test` の担当)、実機の director が問いを 1 回だけ聞くこと
