# Self-review report: org-report-names

- Date: 2026-10-20
- Plan: `docs/plans/active/2026-10-20-org-report-names.md`(F8 の修正、パイプライン cycle 1)
- Reviewer: reviewer サブエージェント(`/self-review`)
- Scope: `git diff 8dae179f...HEAD`(HEAD 87ae8481、10 ファイル、+492/-7)。コミットは b7ce3d23(計画)、9349914b(S1)、7a45cb14(S2)、5b7fe3c0(S3)、87ae8481(計画の Progress)。diff の質だけを見た。spec 準拠、テストの実行、静的解析、ドキュメントの drift は `/verify` と `/test` と `/sync-docs` の担当

## Evidence reviewed

- 全文を読んだ: 計画(233 行)、新規の `internal/org/report_names_sync_test.go`(188 行)、`git diff 8dae179f...HEAD` の残り 8 ファイルの差分
- 周辺を読んだ: `scripts/check-sync.sh` の除外リストと `is_excluded`(121〜129 行、前方一致)、`internal/org/prompts/reviewer.md` の RESULT・BLOCKED の例とレポート契約、`internal/org/prompts.go` の変数の置換(`{{ORG_ID}}`・`{{SEAT_ID}}`)、`.claude/skills/org/SKILL.md` の `report` 行(180)・締めの手順(828〜860)・チェックリスト(1336)、`internal/org/prompts/leader.md` の手順 9〜10、`prompts_test.go` の `renderSeatPrompt`・`testRolePromptVars`・`markdownSection`・`markdownItem`、同じ形の先例 `send_defaults_sync_test.go`・`director_skill_sync_test.go`・`split_test.go:838`、`internal/insights/backfill.go:70,183`(`self-review-` の前方一致)、`scripts/archive-plan.sh` の書き換えの範囲
- tech-debt の変えた 2 行(319〜321 行)を、unescaped の `|` で分けて欄ごとに読んだ。どちらも 5 欄。F8 の行の形は Debt と Trigger を `~~` で打ち消し、Debt に `(RESOLVED 2026-10-20 in fix/org-report-names)` を付け、上に `<!-- RESOLVED ... -->` の行を置く形で、register の多数派と同じ
- `git diff --check 8dae179f...HEAD`: 問題なし。追加行の grep(`/Users/`、`TODO`・`FIXME`・`XXX`、`fmt.Print`・`println`): 0 件。秘密情報の形: なし
- `cmp docs/reports/README.md templates/base/docs/reports/README.md`: 同一
- `org-manifest-` を記録(evidence・plans・reports・tech-debt)以外で grep し、手で書かれた場所を洗い出した(L1)。`review-` で始まる既存の `docs/reports/` 直下のファイルはなし
- 確かめていない: テスト・gofmt・vet・lint の実行(依頼の範囲外。新しいテストファイルの import は目で見て、`errors`・`io/fs`・`os`・`path/filepath`・`runtime`・`strings`・`testing` がすべて使われていること、`prompts_test.go` が `regexp` を既に import していることだけ確かめた)、`check-sync.sh` を未追跡のレポートで動かした結果(AC1 は `/verify` の担当)、図解ページ `.harness/state/plan-visual/org-report-names.html`

## Findings

<!-- Area recommended values: naming, readability, unnecessary-change, typo,
     null-safety, debug-code, secrets, exception-handling, security, maintainability -->

| Severity | Area | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- |
| LOW (L1) | maintainability(コメントと記録の主張) | 新しいコメントと register の閉じ方が、test の守る範囲より広く書かれている。(a) test の冒頭は 2 つの名前が「every hand-written place」で同じ綴りだと言うが、守るのは 4 か所(reviewer の雛形、`check-sync.sh` の除外、`docs/reports/README.md`、`/org` skill の `report` 行)で、`org-manifest-` を手で書いた場所はほかにもある。(b) `report.go` の定数の doc は「check-sync.sh, docs/reports/README.md and the /org skill spell it by hand」と 3 か所を挙げるが、skill の締めのチェックリストと help も同じ名前を書く。(c) register の RESOLVED は「ties both names to the template, the exclusions, `docs/reports/README.md`, and the `/org` skill's `report` row」と書くが、雛形が書くのは reviewer の名前だけ、skill の `report` 行が書くのは org report の名前だけで、名前ごとに結ぶ場所は違う | (a) `internal/org/report_names_sync_test.go:70-77`(75 行に「every hand-written place」)。(b) `internal/org/report.go:14-16`。手で書かれていて test に結ばれていない場所: `.claude/skills/org/SKILL.md:1336`(`docs/reports/org-manifest-*.md`)、`internal/cli/org.go:1931`(help)、`internal/cli/org_test.go:3032`(前方一致)、`internal/org/report.go:35,115`(同じファイルの doc)。計画の Progress は cli の help・`org_test.go`・skill の 2 か所を範囲外と書くが、コメントと register にはその限定がない。(c) `docs/tech-debt/README.md:319` | (a) は「every」を「the places listed below」に、(b) は「spell it by hand」の主語を「the places the sync test checks」にして、(c) は名前ごとに結ぶ場所を書く(reviewer の名前: 雛形・除外・README、org report の名前: `report.go`・除外・README・skill の `report` 行)。範囲外の場所は計画の Non-goals どおりで、直すのは文だけ |
| LOW (L2) | maintainability(記録の食い違い) | F11 の行の Why 欄が、Trigger 欄と食い違う。Trigger は今回「The `check-sync.sh` row above was fixed in fix/org-report-names without moving the org report, so it no longer covers this row」と書き直したが、同じ行の Why 欄は「which sessions it should skip, or where `ralph org report` writes, is a design choice tied to the `check-sync.sh` row above (F8)」のまま。F8 の行は閉じたので、「tied to」は閉じた行を指して読める | `docs/tech-debt/README.md:321`(Why 欄と Trigger 欄。Impact 欄の「the decision of the `check-sync.sh` row above」は試行の記録なので、そのままでよい) | Why 欄の「tied to the `check-sync.sh` row above (F8)」を、「F8's fix (fix/org-report-names) left it open」の形にするか削る |
| LOW (L3) | maintainability(参照の壊れやすさ) | 新しい Go のコメント 2 か所が、計画を `docs/plans/active/2026-10-20-org-report-names.md` で引く。`/pr` の `scripts/archive-plan.sh` が書き換えるのは `docs/tech-debt/README.md` の参照だけなので、archive 後はこの 2 か所が存在しないパスを指す。register の参照は書き換わるので問題ない。`internal/org/` には同じ形の先例が多く(`approval.go:21` ほか)、この PR だけの癖ではない | `internal/org/prompts_test.go:970`、`internal/org/report_names_sync_test.go:71`。書き換えの範囲は `scripts/archive-plan.sh:53-60` と awk の `old`・`new` | 「plan 2026-10-20-org-report-names, AC2」のように slug で引く(パスと違って archive で壊れない)。直さないなら先例どおりで、害は archive 後のコメントの参照切れだけ |
| LOW (L4) | naming(テスト間の結びつき、定数の組み立て) | `reviewerReportPrefix` は test ファイル `report_names_sync_test.go` で定義され、`prompts_test.go` の新しい検査もこれを使う。sync test のファイルを移したり消したりすると `prompts_test.go` がコンパイルできない。また同じファイルの中で、org report 側の prefix は `defaultReportDir + "/" + orgReportFilePrefix`(92 行)と本番の定数から組むのに、reviewer 側は `"docs/reports/review-"` と書き、README の検査は `strings.TrimPrefix(reviewerReportPrefix, "docs/reports/")`(135 行)でディレクトリを手で書き直す。`reviewerReportPrefix` のディレクトリを変えると、この `TrimPrefix` は何も削らず、README に期待する行がディレクトリ付きの名前になって、失敗の理由が読み取りにくい | `internal/org/report_names_sync_test.go:16,92,135`、`internal/org/prompts_test.go:1061-1062` | `const reviewerReportFilePrefix = "review-"` を足して、`reviewerReportPrefix = defaultReportDir + "/" + reviewerReportFilePrefix` とし、README の検査は `reviewerReportFilePrefix` をそのまま使う(文字列定数なので const のまま組める)。`prompts_test.go` が test 間の定義に頼る点は、この定数を `prompts_test.go` に置けば消える |
| LOW (L5) | readability(テストの長さと名前) | `TestRenderRolePrompt_Reviewer_ResultCarriesHeadAndBase` が 56 行から 94 行になり(`git show 8dae179f:internal/org/prompts_test.go` で数えた)、名前(RESULT が HEAD と BASE を持つ)がもう中身の半分(レポート名と seat id の違い)を言わない。doc には 3 行足したが、失敗の一覧に出るのは関数名だけ。計画の Scope は「one more check in the reviewer test」と書いているので、設計どおりではある | `internal/org/prompts_test.go:972-1065`(新しい検査は 1027〜1064 行)。`testing.md` の「test names specific enough that failures explain intent」 | 新しい検査を `TestRenderRolePrompt_Reviewer_ReportContractNamesTheReport` に切り出す(`reportPath` の closure はそのまま移せる)。計画の AC2 は「`TestRenderRolePrompt_Reviewer_ResultCarriesHeadAndBase` checks it」と書くので、切り出すなら AC2 の名前と計画の Progress も合わせる。直さないなら、いまの形でよい |
| INFO (I1) | maintainability(重複) | `reportNamesRepoRoot` は同じ本体の repo root のヘルパーの 4 つ目の写し(`splitTestRepoRoot` `split_test.go:838`、`sendDefaultsRepoRoot` `send_defaults_sync_test.go:50`、`directorSkillRepoRoot` `director_skill_sync_test.go:35`)。計画が「as `send_defaults_sync_test.go` does」と指示し、先例も 1 ファイル 1 本の写しなので、この PR の欠陥ではない | `internal/org/report_names_sync_test.go:21-28` | 直さなくてよい。まとめるなら 4 本を `internal/org` の test の共有ヘルパー 1 本にする別の掃除 |
| INFO (I2) | readability(コメントの言語) | `scripts/check-sync.sh` の英語のコメントに、日本語の `レポート契約` が 1 つだけ入る。スクリプトの既存の非 ASCII は罫線とダッシュだけ。節の見出しそのものなので grep はできる | `scripts/check-sync.sh:78` | 直さなくてよい。直すなら「the report contract section of reviewer.md」 |

## Positive notes

- 新しい除外は 2 つの前方一致 `docs/reports/review-` と `docs/reports/org-manifest-` だけで、`docs/reports/templates/review-report.md` は間に `templates/` が入るので覆われない(`is_excluded` は `"${pattern}"*` の前方一致、`check-sync.sh:121-129`)。`docs/reports/` 直下に `review-` で始まる既存のファイルはない。除外が広すぎて本物の ROOT_ONLY を隠す心配は、読んだ範囲ではない
- reviewer の雛形の変更は、レポート契約の第 1 項目だけ。第 2 項目(`leader へは RESULT ...`)は 1 バイトも変わらず、`TestRenderRolePrompt_Reviewer_ResultCarriesHeadAndBase` が読む文を壊さない。`{{ORG_ID}}` と `{{SEAT_ID}}` は `prompts.go:88-89` の置換に載る既存の変数で、新しい変数は足していない
- `self-review-` を避ける理由が雛形に 1 文で書かれ、その主張は `internal/insights/backfill.go:70` の `strings.HasPrefix(base, "self-review-")` と合う
- seat id を名前に入れた判断(Codex の計画助言 MEDIUM の反映)が、新しい検査で効く。同じ org の `reviewer` と `reviewer-2` が別のファイルを名指し、2 つとも除外の prefix で始まることを確かめる
- `report.go` の定数への置き換えは、`Sprintf` の書式を `"%s%s-%s.md"` にしただけで、書き出す名前は変わらない。既存の `report_test.go:175,220,224` と `internal/cli/org_test.go:3032` が名前を直書きで持つので、名前を変えると落ちる(差分の外)
- sync test の組み立てが先例より良い。`send_defaults_sync_test.go` は最初の面が無いと `t.Skipf` で関数全体を止めるが、今回は面ごとの subtest で skip するので、1 つ無くても残りは検査される(`readRepoFileOrSkip`)。`readRepoFile` は「無い」と「読めない」を分け、`TestReadRepoFile_MissingFileIsNotAnError` が 3 つ(読める、無い、ディレクトリ)を押さえる
- sync test の抜き出しは範囲を絞る。除外は `ROOT_ONLY_EXCLUSIONS=(` から `)` の間の行だけを `hasLine` で完全一致させるので、コメントに書かれた同じ文字列では通らない。README は `Suggested naming:` の箇条書きだけ、skill は `| \`report\` |` の行だけを見る。計画のリスク欄の「by phrase」の心配に対する対処が、コードに入っている
- register の閉じ方は、F8 の行が 5 欄で、Debt と Trigger の打ち消し、`(RESOLVED ...)`、上の `<!-- RESOLVED ... -->`、Related への計画の追加まで多数派の形に合う。閉じた中身(2 つの除外、雛形の名前、`orgReportFilePrefix`)は diff と合う。F11 を開けたまま Trigger だけ書き直す扱いも、計画の Non-goals どおり
- 差分に、デバッグ出力、TODO、秘密情報、ホームのパスはない。`git diff --check` も通る

## Tech debt identified

| Debt item | Impact | Why deferred | Trigger to pay down | Related plan/report |
| --- | --- | --- | --- | --- |
| なし | — | cycle 1 で、L1〜L3 は文の直し(test の冒頭、定数の doc、register の 2 行)、L4・L5 は test の数十行の整理。cap の最後の回で残るときだけ、1 行にまとめて足す | — | — |

_(If any rows were added above, also append them to `docs/tech-debt/`.)_

## Recommendation

- Merge: yes(CRITICAL・HIGH・MEDIUM なし。LOW 5 件は、コメントと register の閉じ方が test の守る範囲より広い点(L1)、F11 の行の Why 欄が Trigger と食い違う点(L2)、Go のコメントが archive で壊れる plan のパスを引く点(L3)、test 定数の置き場所と組み立て(L4)、reviewer のテストの長さと名前(L5)。INFO 2 件は直さなくてよい)
- Follow-ups:
  - L1 は test の冒頭、`report.go` の定数の doc、register の 1 行の文面で、挙動は変わらない。L2 は register の 1 欄。L3 は 2 つのコメント。L4 は 2 つの test ファイルの定数 1 つ。L5 は test の切り出し(計画の AC2 の名前に触れる)。L1〜L3 を直すなら 1 commit で足りる。register と Go の編集を含むので、`post-implementation-pipeline.md` の「記録だけの修正」の外にあたり、直したあとは `/self-review` から回し直す。いまは `/verify` の前なので、直すかどうかは lead の判断に任せる。直さずに進めるなら、cycle 2 の最終回に、残った LOW を 1 行にまとめて register に足す
  - I1・I2 は直さなくてよい
- Known gaps:
  - テスト・静的解析・`check-sync.sh` の実行は範囲外なので、AC1(未追跡のレポートで rc 0、`other-x.md` で rc 1)・AC3(既存の report のテスト)・AC7(`run-verify.sh`、`GOTOOLCHAIN=go1.25.8`)の結果は、この報告では確かめていない
  - 新しい test ファイルは gofmt・vet を通していない(目で見たのは import の使用と、doc コメントの箇条書きの字下げだけ)
  - F11 の行の Impact と Trigger の中身(この修正のあと、leader が skill の手順 2 で org report をコミットすると、session-end の WIP hook が拾うものが残るのはどの場合か)は、diff の質ではなく設計の話なので見ていない。L2 は欄どうしの食い違いだけを指す
  - 計画の「Visual review」(図解ページと PNG の確認)は読んでいない

## Revalidation (3ff6abdc)

対象は修正コミット 3ff6abdc(`git show 3ff6abdc`、4 ファイル、+20/-15)。上の Findings と Merge 行、既存の重大度は変えていない。diff の質だけを見た。テストと静的解析は回していない。

- L1: resolved。(a) `internal/org/report_names_sync_test.go:70-82` の冒頭が「every hand-written place」をやめ、検査する 4 か所と、検査しない場所(`ralph org report` の help、`internal/cli/org_test.go` の前方一致、skill の `inbox resolve` の例と締めのチェックリスト)を分けて書く。挙げた 3 か所の実在は `git grep org-manifest-` で確かめた(`internal/cli/org.go:1931`、`org_test.go:3032`、`.claude/skills/org/SKILL.md:184,1336`)。(b) `internal/org/report.go:14-17` の主語が「the /org skill's `report` row」になり、test の検査と合う。(c) `docs/tech-debt/README.md:319` の RESOLVED が、reviewer の名前(雛形・除外・README)と org report の名前(除外・README・skill の `report` 行)を分けて書き、test の subtest 4 つと合う
- L2: resolved。`docs/tech-debt/README.md:321` の Why 欄が「a design choice that was tied to the `check-sync.sh` row above (F8), which fix/org-report-names closed without moving the org report」になり、閉じた行を生きた依存として指さない。この行は 5 欄のまま
- L3: resolved。`internal/org/prompts_test.go:970` と `report_names_sync_test.go:71` が `plan 2026-10-20-org-report-names` の slug 引きになった。このブランチの差分(docs/plans・docs/reports・docs/tech-debt を除く `*.go` `*.sh` `*.md`)に `plans/active/2026-10-20-org-report-names` を引く場所は残っていない。register 側のパスは `archive-plan.sh` が書き換える。reflow した 2 つのコメントの行幅は 78 文字以内で、過長行はない
- L4: 一部 resolved。ディレクトリの組み立ては直った。`reviewerReportPrefix = defaultReportDir + "/review-"`(`report_names_sync_test.go:16`)になり、README の検査は `strings.TrimPrefix(reviewerReportPrefix, defaultReportDir+"/")`(139 行)で、`defaultReportDir` を変えても両方が追随する。残り: `prompts_test.go:1061-1062` が、別の test ファイルで定義された `reviewerReportPrefix` に依存する点は、そのまま(推奨は定数を `prompts_test.go` に置くか `reviewerReportFilePrefix` を足す形だった)。LOW のまま。コンパイルが落ちるのは sync test のファイルを消す・移すときだけで、害は小さい
- L5: not addressed(依頼どおり)。計画が AC2 の検査をこの test に足すと書くので、現状でよい。LOW のまま
- I1・I2: not addressed(依頼どおり、直さなくてよい項目)

新しい指摘:

- INFO (N1) maintainability(コメントの列挙): `report_names_sync_test.go:80-82` の「Other prose that names the org report is not checked: ...」は閉じた列挙に読めるが、`internal/org/report.go:36` と `:116` の doc コメントも org report の名前を手で書く(同じファイルの定数のすぐ近くなので害は小さい)。同じ冒頭の「the ones a feature org's gate and the naming list rely on」は、reviewer の雛形と skill の `report` 行には当てはまらず、関係節が列挙より狭い。直すなら関係節を削り、「not checked」の列挙に「and the doc comments in report.go」を足すか「for example」に変える。直さなくてもよい
- INFO (N2) readability(記録の重複): F11 の行(`docs/tech-debt/README.md:321`)の Why 欄と Trigger 欄の最後の文が、同じ事実(F8 の行は org report を動かさずに閉じた)を 2 回書く。食い違いではなく重複。直さなくてよい

確かめたこと: `awk '/^\| /{...}' docs/tech-debt/README.md` の出力は `22: 7`、`32: 5`、`278: 10` の 3 行だけ(今回の修正で欄の数が崩れた行はない)。`git diff 3ff6abdc^ 3ff6abdc --check` は rc 0。確かめていないこと: テスト・gofmt・vet の実行、修正後の `check-sync.sh` の実行。

残りの LOW(L4 の残り、L5)は、この回が最終回なら 1 行にまとめて register に足す。最終回でなければ次の回で扱う。

Revalidation merge: yes
