# Self-review report: org-record-text

- Date: 2026-10-21
- Plan: `docs/plans/active/2026-10-21-org-record-text.md`(パイプライン cycle 1)
- Reviewer: reviewer サブエージェント(`/self-review`)
- Re-review: 修正 commit `cab879de`(L1・L2・L3・I2 の直し)を `git show cab879de` で読み直した。結果は「## Re-review of cab879de」に書く。下の所見の表は 1d3ce15d 時点の文面のまま、行頭の重大度の欄に直した状態を足した
- Scope: `git diff b55d3d73...HEAD`(HEAD 3d5d9a61、10 ファイル、+569/-24)。S1(`internal/org/verbs.go`、`verbs_test.go`)、S2(`internal/org/report.go`、`report_test.go`)、S3(`prompts_test.go`、`report_names_sync_test.go`)、S4(`docs/tech-debt/README.md`、`docs/evidence/org-spec-evidence-2026-10-21.md`、`docs/specs/2026-10-07-org-multi-org-director.md`)、計画。diff の品質だけを見た。仕様適合、テストの実行、静的解析、文書のずれは `/verify`・`/test`・`/sync-docs` に任せた

## Evidence reviewed

- 4 つの slice の diff を全部読んだ。コードは `git diff b55d3d73...HEAD -- internal/org/`、記録は 3 ファイルの diff と、tech-debt の新しい 2 行(323、326 行)・HTML コメント 2 行(322、325 行)を HEAD で読んだ
- `reportCellReplacer` の並び順(依頼の確認点)。`strings.NewReplacer` は、old が 1 byte でない組(`"\r\n"`)を含むので汎用の実装になり、同じ位置で複数の old が合うときは引数の早いものを取る。scratchpad に最小の probe を書いて 3 通りを打った。`"\r\n"` を先に置く今の並びは `a\r\nb` を `<br>` 1 つにし、`"\r"` を先に置くと `<br><br>` になる。`\` を先頭に置くか最後に置くかでは出力が変わらない(1 回の走査で、足した `\` は読み直されない)。つまり心配は「CRLF の順序」だけが本物で、「`\` を先に」は実害のない並びである。`report.go:116-128` のコメントは CRLF の順序を理由に書き、`\` の順序を理由にはしていないので、コメントと実装は合う
- `truncateForDetails`(`verbs.go:661`)の添字。`len(text) > 200` を通ったあとなので `text[200]` は範囲内。ループは `i < utf8.UTFMax-1`(3 回)で、4 byte 文字が 197 から始まる最悪の場合に cut は 200→199→198→197 と 3 回戻って文字の先頭で止まる。戻りは 197 を下回らないので `text[:cut]` は常に範囲内。不正な UTF-8 では 197 で止まる(テストの最後の事例の期待と一致)。テストの事例を 1 つずつ手で数え直した(199+3 byte、197+3 byte、196+4 byte、197・198・199・200+4 byte、250 ASCII、200 ASCII、197+3 byte の計 200、不正な列)。期待の長さはすべて合う
- Send 経由の試験(`verbs_test.go` の `TestOrgSend_JapaneseBodyOver200Bytes_DetailsStayValidUTF8`)。送る文面は先頭の 25 byte に「日本語の本文を記録する。」(36 byte)の 10 回で、200 byte 目は 175 byte 目 = 文字の 2 番目の byte にあたる(175 = 58×3+1)。試験が `utf8.RuneStart(msg[200])` の偽を `Fatalf` で守るので、前提が崩れれば失敗する。U+FFFD は `utf8.RuneError` で書いていて、diff に U+FFFD の文字そのものはない(`git diff` の中の EF BF BD は 0 件)
- GFM 側の読み。`report.go:129-137` の 7 組を、`|`・`\`・バッククォート・`<` と改行の組合せで頭の中で追った。表の行の分割は `\|` を区切りとみなさず、そのあと `\|` を `|` に戻すので、`a\|b` の `a\\\|b` は「`a\|b` の文字そのまま」に出る(AC3 の `\\\|`)。`<` を `\<` にするので、本文の `<script>` や `<br>` が HTML として働かない。これは前からの `|` だけの escape より安全になる方向の変更
- 計画の数字を試行の manifest で数え直した(`docs/evidence/org-trial-2026-10-20/manifest.jsonl`、`jq` で読むだけ)。Details のある event は 77、`_` を含むのは 71、`_` が語の外にあるのは 0、`*` を含むのは 1、バッククォートと `<` は 0。計画の Non-goals の「77 のうち 71 が `_`、すべて `TASK_ID` のような語の中、`*` は 1」と一致する。`git grep -n '"| %s'` は `report.go` の 3 行(roster、timeline、receipts)だけで、Assumptions の「表を書く Go のコードは `report.go` だけ」と合う。roster と receipts の列は id、名前、model で、本文を含まない
- S3 の分割。移した検査を、元の関数(`git show b55d3d73:internal/org/prompts_test.go`)の `contract` の `GATE: pass` の検査から先と、新しい関数を並べて読んだ。`reportPathRe`、`reportPath` の closure、`first`、`second`、`HasPrefix` の検査まで、本文は一字も変わらない(`t.Errorf`・`t.Fatalf` 6 つが新しい関数に移っただけ)。元の関数に残った部分は `text`・`mission`・`item`・`protocolSection`・`contract` をそのまま使い、未使用の変数は出ない。新しい関数は `renderSeatPrompt` で `text` を作り直すので変数の参照も足りる
- S4 の register。新しい 2 行を `sed 's/\\|//g'` のあと `|` を数えた。323 行と 326 行はどちらも 6 個(5 欄)で、Related 欄を失っていない。隣の 321・324・327・328 行も 6 個。HTML コメント 2 行は `-->` がそれぞれ 1 つで、途中に閉じる文字列はない。形は F8 の行(319・320 行)と同じで、上に `<!-- RESOLVED ... Row preserved for traceability. -->`、Debt 欄を `~~...~~ (RESOLVED ...)`、Trigger 欄を `~~...~~ — RESOLVED: ...`、Related に計画を足す。計画の Progress が書く「Trigger 欄も打ち消した」は、この F8 の行のとおりで、計画の「RESOLVED convention of the rows around them」の内。逸脱と呼ぶ必要はない。コメントが挙げる試験名 4 つ(`TestTruncateForDetails_CutsAtACharacterBoundary`、`TestOrgSend_JapaneseBodyOver200Bytes_DetailsStayValidUTF8`、`TestBuildOrgReport_DetailsCellKeepsEachEventOnOneRow`、`TestBuildOrgReport_TimelineHasOneLinePerEventOfFourCells`)は diff に実在し、挙げた事例(197〜199 の 4 byte、ちょうど 200 byte など)も試験の表と合う。コメントの slice の commit(S1・S2)は `git log` と合う。`docs/plans/active/` のパスは、`scripts/verify.local.sh` の `check_tech_debt_plan_refs` が archive 前の計画の実在を要求し、`scripts/archive-plan.sh` が archive のときに書き換える(同じ名前の境界の規則で、コメントの `)` 直前も、Related 欄の末尾の `` ` `` も拾う)。Progress の 2 つ目の逸脱(active のパスを足した)は、この 2 つの script の組合せから見て妥当
- 差分に、デバッグ出力、TODO、秘密情報、ホームのパス、行末の空白はない。`git diff --check` は出力なし
- 確かめていない: テストと gofmt・vet の結果(範囲外)、`gh api markdown` での描画(計画の AC3b は verifier の仕事)、`docs/evidence/org-trial-2026-10-20/reports/` の内容(計画で書き換えない)、natural-japanese の lint(書き込み時の hook に任せる)

## Findings

<!-- Area recommended values: naming, readability, unnecessary-change, typo,
     null-safety, debug-code, secrets, exception-handling, security, maintainability -->

| Severity | Area | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- |
| LOW (L1)(cab879de で直した) | maintainability(コメントが挙動より広い) | `reportCellReplacer` の doc は「a GFM renderer shows the text as written」と言うが、`*`・`_`・`[`・`]`・`~`・`&`・`>` は escape しないので、本文の `[x](http://...)` はリンクに、`*a*` は強調に、`&lt;` は `<` に描画される。escape しない文字の一覧とその理由は、計画の Non-goals と Open questions、register の HTML コメント(325 行)にあり、コードのそばにはない。計画は archive に移り、HTML コメントは描画では見えないので、`report.go` を読む人が「そのまま出る」と受け取りうる | `internal/org/report.go:116-128`(「shows the text as written」)。escape しない文字は `docs/plans/active/2026-10-21-org-record-text.md` の Non-goals と Open questions、`docs/tech-debt/README.md:325` | doc の末尾に 1 文足す。例: 「`*`, `_`, `[`, `]`, `~`, `&`, and `>` are left as they are: they style or link a span but never move a row or a line end, and escaping every one would fill the raw report with backslashes.」。挙動は変わらない |
| LOW (L2)(前半は cab879de で直した。PR の番号を足す後半は次の系列の PR へ) | readability(兄弟の行との書き方の不一致) | 所見の表で、直した行 F8 は「直す PR」欄が `#244 で直した`、「tech-debt の行」欄が `"~~...~~"(RESOLVED の行)` と、register の閉じ方まで書く。F12・F14 は「直す PR」欄に `記録の文字(fix/org-record-text で直した)` と書くだけで、「tech-debt の行」欄は閉じる前のまま(register の 323・326 行はもう打ち消してある)。表の冒頭(16 行)は「系列の PR は、直した所見に結びつく項目の印とこの表を同じ PR で更新する」と言うので、この表は系列の進み具合を読む場所になる。F8 だけ閉じた形で、F12・F14 は半分だけ閉じた形に見える | `docs/evidence/org-spec-evidence-2026-10-21.md:27`(F8)と `:31`・`:33`(F12・F14)。計画の S4 は「F12 and F14 rows name this branch as the fix」と書くだけで、最後の欄には触れていない | F12・F14 の最後の欄を F8 と同じ `"~~...~~"(RESOLVED の行)` にする。PR の番号は出来てから(`/pr` のあと)で、そのときに「直す PR」欄を `#<番号> で直した` にし、仕様 18 行の「試行の所見 F8 の直し: #244」と 97 行の表にも 1 行ずつ足す必要がある。次の PR への引き継ぎとして、PR の本文に 1 文書いておくとよい |
| LOW (L3)(cab879de で直した) | readability(同じ匿名 struct を 2 回書く) | `reportCellCases` は戻り値の型と本体の複合リテラルで、同じ 3 欄の匿名 struct を 2 回書く。使う側は 2 つの試験で `range` するだけ | `internal/org/report_test.go:117-127` | `type reportCellCase struct { name, details, wantCell string }` を 1 つ置き、`[]reportCellCase` を返す。事例を足すときに 2 か所を揃える必要がなくなる |
| INFO (I1) | maintainability(依存の向き) | S3 は prompts のテストが sync テストのファイルに頼る向きを消したが、逆向きに替わっただけである。`report_names_sync_test.go` は `reviewerReportPrefix` を 2 か所(102・119 行)、`reviewerReportFilePrefix` を 1 か所(135 行)で使い、定義は 1 か所しか使わない `prompts_test.go`(1078 行)にある。`prompts_test.go` を動かすか分けると、sync テストがコンパイルできない。前回の L4 の推奨(定数を `prompts_test.go` に置く)と計画の AC5 はこの形を求めているので、設計どおり | `internal/org/prompts_test.go:1030,1035`、`internal/org/report_names_sync_test.go:102,119,135` | 直さなくてよい。移すなら、定数は「名前の契約」を 3 回使う sync テスト側に置いた方が自然だったが、前回の L4 が prompts 側を勧めた経緯があるので、いまの形で足りる |
| INFO (I2)(cab879de で直した) | readability(コメントの重複) | `TestRenderRolePrompt_Reviewer_ReportContractNamesTheReport` の doc(1038-1041 行)と、本体の最初の行のコメント(1045-1047 行)が、同じ内容(contract の最初の項目がファイルを名指す、2 座席は別のファイル、どちらも除外の prefix で始まる)を 2 回書く。後者は分割前の関数の中にあった説明をそのまま運んだもの | `internal/org/prompts_test.go:1038-1041,1045-1047` | 本体側のコメントを消すか、doc を 1 文に縮める。どちらでもよい |
| INFO (I3)(直さずに残した) | readability(`truncateForDetails` の添字) | ループの添字 `i` は数えるだけで、境界は `utf8.UTFMax-1` の 1 か所に書いてある。`cut > sendDetailsMaxLen-(utf8.UTFMax-1)` を条件に入れれば、添字なしで同じ 3 回に収まる。また doc の「byte `sendDetailsMaxLen`」は 0 始まりの添字で、試験の doc(「zero-based byte n」)にだけ 0 始まりと書いてある | `internal/org/verbs.go:653-670`、`internal/org/verbs_test.go` の `TestTruncateForDetails_CutsAtACharacterBoundary` の doc | 直さなくてよい。直すなら、`verbs.go` の doc に「(zero-based)」と 1 語足すだけで足りる |
| INFO (I4)(cab879de の再読で見つけた) | maintainability(L1 の文が言い過ぎる、小さい) | L1 で足した文の末尾「they cannot split the row or hide text」のうち「hide text」は言い過ぎである。同じ文が「a link」を挙げており、`[x](http://example.invalid)` は `x` だけを見せて `(http://example.invalid)` を見せない(強調の `*` も描画では見えない)。行を割らない、行末を動かさない、という点は正しい。冒頭の「a GFM renderer shows the text as written」も、足した文が挙げる 7 文字には当てはまらないので、同じ doc の 2 文が少し食い違う | `internal/org/report.go:118,130` | 直さなくてよい。Go を編集する機会が別にあるなら、末尾を「they cannot split the row or end the table」にする。この 1 語のために Go を変えて pipeline を回し直すほどではない |

## Positive notes

- S1 の戻りの上限(3 byte)が、ループの条件、doc、試験の下限(`sendDetailsMaxLen - (utf8.UTFMax - 1)`)の 3 か所で同じ式に揃う。不正な UTF-8 で止まることも、試験の最後の事例が 197 byte の位置で押さえる。前方ではなく後方へ切る選び方は、manifest の行が 200 byte を超えて伸びない
- S2 の `strings.NewReplacer` を 1 回の走査にした選び方は、`\` を先に置くかの議論を不要にし、`TestBuildOrgReport_EscapesPipesInEventDetails` の既存の期待(`\|`)も変えない。事例が 11 件(CRLF、単独の CR、LF のあとの CR、`\` が行末、フェンス、`<path>`、既に `\|` を含む文、空)と広く、各事例を行全体の文字列で比べる(`strings.Contains` で部分一致にしない)。タイムラインの行数と欄の数も別の試験が数える
- S3 の分割は、`t.Errorf`・`t.Fatalf` を 1 つも落とさず、関数名が中身を言うようになった(前回の L5)。doc の plan 参照を `docs/plans/active/` のパスから slug にしたので、archive でパスが切れない(前回の L3 と同じ直し方)
- S4 の記録は、F8 の行の閉じ方(コメント、打ち消し、`(RESOLVED ...)`、Trigger の打ち消し、Related に計画)を 5 欄のまま写している。コメントが挙げる試験名と事例が diff と合う
- 差分に、デバッグ出力、TODO、秘密情報、ホームのパス、U+FFFD の文字そのものはない

## Re-review of cab879de

読んだもの: `git show cab879de`(4 ファイル、+13/-16)、`internal/org/report.go:116-131`、`internal/org/report_test.go:114-125,172-195`、`internal/org/prompts_test.go:1036-1046`、`docs/evidence/org-spec-evidence-2026-10-21.md:27-34`、register の 320・323・326 行。`git diff --check cab879de^ cab879de` は出力なし。diff に U+FFFD の文字そのものはない(F12 の行にある「U+FFFD」は既存の文面)。

| 所見 | 状態 | 確かめたこと |
| --- | --- | --- |
| L1 | 直した(cab879de) | `report.go:128-130` に 1 文足した。挙げる 7 文字(`*`、`_`、`[`、`]`、`~`、`&`、`>`)は、`strings.NewReplacer` の 7 組(バックスラッシュ、パイプ、バッククォート、`<`、CRLF・CR・LF)のどれとも重ならず、計画の Non-goals の一覧と同じ。理由の半分(行や行末を動かさない)は挙動のとおり。「全部 escape すると生の report が `\` だらけになる」という理由は書いていないが、足りている。挙動は変わらない(組は同じ)。ただし末尾の「hide text」が言い過ぎで、I4 に書いた |
| L2 の前半 | 直した(cab879de) | F12(31 行)と F14(33 行)の最後の欄が、F8(27 行)と同じ `"~~...~~"(RESOLVED の行)` の形になった。`~~` の中の引用は直す前の引用と同じ文字で、register の 323・326 行の Debt 欄の打ち消しの先頭(`~~` で始まる)と合う。27〜34 行の 8 行を、escape したパイプを除いて区切りのパイプを数えると、すべて 7 個(6 欄)で、F8 の行と同じ。この表を読む Go・shell のテストや script はなく(`org-spec-evidence` を挙げるのは docs だけ)、形の変更で落ちるものはない |
| L2 の後半 | 残した(依頼どおり) | PR の番号を、所見の表の「直す PR」欄(F12・F14 は今も `記録の文字(fix/org-record-text で直した)`)、仕様 18 行、根拠の文書の PR 一覧に足す作業は、PR が出来てからで、系列の次の PR が、PR #245 が F8 に `#244 で直した` と書いたのと同じ形で行う。この PR の範囲では出来ない |
| L3 | 直した(cab879de) | `type reportCellCase struct`(`report_test.go:116`)を 1 つ置き、`reportCellCases` が `[]reportCellCase` を返す(123・124 行)。匿名 struct を 2 回書く箇所は消えた。使う側の 2 つの試験(174・192 行)は `c.name`・`c.details`・`c.wantCell` を読むだけで、型名を書かない。package に同じ名前はない(`git grep` は `report_test.go` だけ)。欄の揃え(`name` の後ろの 5 空白)は gofmt の形に見える(gofmt は回していない) |
| I2 | 直した(cab879de) | 本体の先頭のコメント 3 行を消した。残った doc(`prompts_test.go:1038-1041`)に、消した 3 行の 3 点(最初の項目がファイルを名指す、2 座席は別のファイル、どちらも `reviewerReportPrefix` で始まる)が全部あり、消した行だけにある情報はない。`text := ...` のあとの空行は残り、`reportPathRe` の定義へつながる |
| I3 | 残した(依頼どおり) | 直さなくてよい所見のまま |

新しい所見は INFO 1 件(I4、上の表)。cab879de は新しい挙動も、デバッグ出力も、秘密情報も足していない。

## Tech debt identified

| Debt item | Impact | Why deferred | Trigger to pay down | Related plan/report |
| --- | --- | --- | --- | --- |
| なし | — | cycle 1 で、L1・L3・I2 と L2 の前半は cab879de で直した。残る L2 の後半は PR の番号が要る作業で、系列の次の PR への引き継ぎ(deferral ではない)。I1・I3・I4 は INFO | — | — |

_(If any rows were added above, also append them to `docs/tech-debt/`.)_

## Recommendation

- Merge: yes(cab879de の再読後。CRITICAL・HIGH・MEDIUM・この PR の中で直せる LOW はなし。L1・L3・I2 と L2 の前半は cab879de で直した。残りは、PR の番号が出来るまで足せない L2 の後半(系列の次の PR への引き継ぎ)と、INFO の I1・I3・I4)
- Follow-ups:
  - cab879de は Go のコメントと test の型の編集を含むので、`post-implementation-pipeline.md` の「記録だけの修正」の外にあたる。`/verify` と `/test` は cab879de より前の tree の結果を使わず、cab879de 以降の tree で走らせる。I4 を直すなら、Go を編集する別の機会に 1 文として入れる。単独では直さない
  - PR の番号が出来たら、所見の表の F12・F14 の「直す PR」欄、仕様の「試行の所見 F8 の直し: #244」の並び(18 行)と、根拠の文書の PR 一覧の表(97 行)に、この PR を足す(L2 の後半。この PR の範囲では出来ない)。PR の本文に 1 文書いておくとよい
  - 計画の Progress の逸脱のうち、「Trigger 欄も打ち消した」は F8 の行の形なので逸脱ではなく、「active のパスを足した」は `check_tech_debt_plan_refs` と `archive-plan.sh` から見て妥当。「Go のコメントは pipe を語で書く」「仕様 141 行は 。の後に 1 文」も妥当
- Known gaps:
  - cab879de の再読でも、テスト、gofmt、vet、`check-sync.sh` は回していない。`report_test.go` の欄の揃えは目で読んだだけである
  - テスト、gofmt、vet、`check-sync.sh` は回していない。AC1〜AC7 の結果は、この報告では確かめていない
  - `gh api markdown` での描画(AC3b)は見ていない。`\\\|` が「`\|` の文字そのまま」に出ることは、GFM の表の分割と escape の規則から読んだだけで、描画では確かめていない
  - 計画の「Visual review」(図解ページと PNG)は読んでいない
  - `docs/evidence/org-trial-2026-10-20/` の本文と、`org-spec-evidence` の「項目」表の NFR 記録の置き場所の注(60 行、「F12 と F14 は…食い違わない」)は、文書のずれの範囲なので見ていない(読んだ限り、この文は今も正しい)

## I4 の直し(orchestrator の追記)

I4 は、Go のコメントだけを直す 8a52987f で直した。`reportCellReplacer` の doc の冒頭を「GFM の表の 1 行に収まり、pipe、backslash、backtick、`<` は書いたとおりに出る」の意味の文にし、末尾を「they cannot split the row or end the table」にした。直したのはコメントだけで、`strings.NewReplacer` の組は変わらない。`gofmt -l` は出力なし、`go vet ./internal/org/` は通った。`/verify` と `/test` は 8a52987f 以降の tree で走らせる。
