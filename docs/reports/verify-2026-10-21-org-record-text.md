# Verify report: org-record-text

- Date: 2026-10-21
- Plan: `docs/plans/active/2026-10-21-org-record-text.md`(試行の所見 F12・F14 と、PR #244 の self-review が残した L4 の残り・L5 の修正。Status: Approved、`plan-visual.sh digest` は `200072f06b32` で `- Approved:` 行と一致)
- Verifier: verifier subagent(Claude Code、pipeline cycle 1)
- Scope: `git diff b55d3d73...HEAD`(HEAD 55338bbc、12 ファイル、+655/-27)。AC1〜AC7 と AC3b、計画の Verify plan、Non-goals、Assumptions。判定は HEAD の tree に対して行った。self-review の修正 cab879de(Go のコメント、テストの型 `reportCellCase`、evidence の表)と 8a52987f(`reportCellReplacer` の doc)は slice のあとに入っており、どちらも HEAD に含まれる。self-review の報告 `docs/reports/self-review-2026-10-21-org-record-text.md` は Merge: yes
- Evidence: `docs/evidence/verify-2026-10-21-org-record-text.log` と、run-static-verify.sh が書いた `docs/evidence/verify-2026-10-21-075413.log`(どちらも `docs/evidence/*.log` の gitignore でコミットしない生の出力。ホームのパスは `~` に置き換えた)。AC3b の見本と描画結果の HTML は scratchpad に置いた

## Spec compliance

テストの実行は `/test` の担当なので、AC1〜AC5 はテストのコードと実装を読んで確かめた。AC3b だけは計画のとおり GitHub の GFM renderer で 1 回描画した。

| Acceptance criterion | Status | Evidence |
| --- | --- | --- |
| AC1: `truncateForDetails` は有効な UTF-8 の入力に有効な UTF-8 を返し、接尾辞の前に残すのは `sendDetailsMaxLen` byte まで。200 byte 以下の入力はそのまま。指定の 6 種の事例 | 満たす(テストの実行は `/test`) | `internal/org/verbs.go:661-670`。`len(text) > 200` のあとで `cut := 200` から、`text[cut]` が文字の先頭でない間だけ最大 `utf8.UTFMax-1`(3)回戻る。有効な UTF-8 では 1 文字の続きの byte は多くても 3 つなので、3 回以内に文字の先頭に着き、`text[:cut]` は文字の切れ目で終わる。不正な列では 197 で止まる。`TestTruncateForDetails_CutsAtACharacterBoundary`(`verbs_test.go:1284`)の 11 事例は、計画の事例をすべて含む(199 ASCII + 3 byte 文字で 199、197 + 3 byte で 200 byte 目に終わる文字を残す、4 byte 文字が 197・198・199・200 から始まる 4 通り、250 ASCII は 200 で切る、ちょうど 200 byte はそのまま、190 ASCII + 継続 byte 20 個は 197 で止まる)。不正な列の期待値(`ascii(190) + "\x80"×7`)はループを手で 3 回追った結果と一致する |
| AC2: 200 byte を超える日本語の本文の `Send` が、有効な UTF-8 で U+FFFD を含まず `...(truncated)` で終わる Details の `sent` を記録する | 満たす(テストの実行は `/test`) | `TestOrgSend_JapaneseBodyOver200Bytes_DetailsStayValidUTF8`(`verbs_test.go:1350`)。manifest から読み戻す(`mustReadEvents` は `o.Manifest.Read()`)。200 byte 目が文字の途中にあることを `utf8.RuneStart(msg[200])` の偽で前提として確かめ、`utf8.ValidString`、`utf8.RuneError` を含まないこと、接尾辞、残した文が本文の前方一致で 200 byte 以下であることを検査する。U+FFFD は `utf8.RuneError` で書き、diff に EF BF BD の byte 列は 0 件 |
| AC3: header 行と空行は 1 行 4 欄で改行ごとに `<br>`、`\r\n` と単独の `\r` は 1 つずつ、改行の隣の `\|`、行末の `\` は `\\<br>`、バッククォートは 1 つずつ escape、`<path>` は `\<path>`、既にある `\|` は `\\\|`。timeline は 1 件 1 行と見出し 2 行。行全体で比べる | 満たす(テストの実行は `/test`) | `internal/org/report.go:132-146` の `reportCellReplacer` は `\`、`\|`、バッククォート、`<`、`\r\n`、`\r`、`\n` の 7 組の 1 回の置換で、217 行の timeline の行だけが `escapeReportCell` を通る。roster(196 行)と receipts(233 行)の行は変わっていない。`reportCellCases`(`report_test.go:123`)の 11 事例が AC3 の事例をすべて含み、`TestBuildOrgReport_DetailsCellKeepsEachEventOnOneRow`(172 行)は timeline が 3 行であることと行全体の一致を、`TestBuildOrgReport_TimelineHasOneLinePerEventOfFourCells`(191 行)は 11 件で 13 行、各行で escape されていない縦棒が 5 つあることを検査する。`strings.Contains` の部分一致は使っていない。置換の組と事例の関数を awk で scratchpad の Go のプログラムに写し、11 事例すべてで `escapeReportCell` の結果がテストの `wantCell` と一致した(mismatches=0) |
| AC3b: AC3 の事例の report を GFM で 1 回描画し、1 件 1 つの `<tr>`、改行ごとの `<br>`、`<code>` なし、各行 4 欄、`\`・バッククォート・`<path>`・`\|` が書いたとおりに出る | 満たす | 下の「Observational checks」。`gh api markdown -f mode=gfm -F text=@sample.md` は rc 0。table 1 つ、tbody の `<tr>` 11(事例 11 件)、`<td>` 44(11 行すべて 4 欄)、`<br>` は見本の 13 と同じ 13、`<code` 0 |
| AC4: 既存の report と send のテストが変更なしで通る(`TestBuildOrgReport_EscapesPipesInEventDetails` と `Send` の Details の prefix のテストを含む) | 変更なしは確かめた。通るかは `/test` | `report_test.go` と `verbs_test.go` の diff に消した行はない(足した行と import だけ)。`TestBuildOrgReport_EscapesPipesInEventDetails`(96 行)の期待 `cmd=a\|b scope=x` は新しい置換でも同じ文字列になる。timeline の行を文字列で比べるほかのテスト `merge_run_test.go:600-605` の `pr_merged` の行は、`/`・`=`・16 進だけで escape の対象の文字を含まない。`approval_test.go:1198` と `gate_test.go:1511` は roster の節だけを読み、roster は `permissionModeFromDetails(s.Details)` で生の Details を読む |
| AC5: 2 つの定数が `prompts_test.go` にあり、sync テストは `reviewerReportPrefix` を定義せず README の検査は `reviewerReportFilePrefix` を使う。報告名の検査は新しいテストに移り、検査の集合は同じ | 満たす(`go test -run` は `/test`) | `prompts_test.go:1030` `const reviewerReportFilePrefix = "review-"`、1035 行 `const reviewerReportPrefix = defaultReportDir + "/" + reviewerReportFilePrefix`(base の `defaultReportDir + "/review-"` と同じ値)。`report_names_sync_test.go` に定義はなく、135 行が `reviewerReportFilePrefix` を使う。base の `TestRenderRolePrompt_Reviewer_ResultCarriesHeadAndBase` の `t.Errorf`/`t.Fatalf` は 17 個で、HEAD の同名の関数(970 行)に 11 個、`TestRenderRolePrompt_Reviewer_ReportContractNamesTheReport`(1042 行)に 6 個。base の関数の `reportPathRe := ` から末尾までと新しい関数の同じ範囲は `cmp` で byte 同一。残った前半も、消したコメント 3 行のほかは base と同じ。2 つの doc は plan を slug で呼ぶ |
| AC6: tech-debt の 2 行が周りの行の形で閉じ、閉じるテストを名指す。evidence の表の F12・F14 がこのブランチを名指す。仕様の NFR 記録の置き場所の注が直したことを書き、印と Status の数は変わらない | 満たす | `docs/tech-debt/README.md:322` が `<!-- RESOLVED 2026-10-21 in fix/org-record-text (S1 19fad46d): … Row preserved for traceability. -->`、325 行が同じ形の `(S2 a88e7382)`(`-->` はそれぞれ 1 つ)。323・326 行は Debt 欄の打ち消しと `(RESOLVED …)`、Trigger 欄の打ち消しと `— RESOLVED:`、Related 欄に計画で、F8 の 319・320 行と同じ形。`\|` を除いて縦棒を数えると 320〜328 行の表の行はすべて 6 本(5 欄)。コメントが挙げるテスト 5 つは `func <name>(` で 1 件ずつ見つかる。evidence の 31・33 行は「記録の文字(fix/org-record-text で直した)」と打ち消した引用で、7 本(6 欄)は F8 の行と同じ。仕様 141 行は「F12 と F14 は fix/org-record-text で直した(…)」を足した 1 行の変更で、`^- \[x\]` は 20、`^- \[ \]` は 12 で base と同じ。印の行の差分は 0 行 |
| AC7: `./scripts/run-verify.sh` が rc 0(`GOFLAGS=-timeout=30m`)、`GOTOOLCHAIN=go1.25.8 go test -timeout 30m ./internal/org/` が通る | 静的な半分だけ確かめた(テストは `/test`) | `GOFLAGS=-timeout=30m ./scripts/run-static-verify.sh` が rc 0。`GOTOOLCHAIN=go1.25.8` で `go build ./...` と `go vet ./internal/org/` が rc 0(vet はテストファイルもコンパイルする)。`run-verify.sh` の既定(`HARNESS_VERIFY_MODE=all`)はテストを含むので回していない |

計画のほかの項目:

- Assumptions: `truncateForDetails` の呼び出しは `verbs.go:348`(dry-run の記録)と 469 行(`sent` の記録)の 2 つ。`git grep -n '"| %s' -- '*.go'` は `report.go` の 196・217・233 行だけで、表を書く Go のコードはここだけ。roster は生の Details を読む(上の AC4)
- Non-goals: 200 byte の上限、`...(truncated)`、Details の中身、manifest の形、ほかの表、report の名前と置き場所、`docs/evidence/org-trial-2026-10-20/reports/` は diff で変わっていない。`*`・`_`・`[`・`]`・`~`・`&`・`>` は escape しない(置換の 7 組に含まれず、`report.go` の doc と F14 のコメントに理由がある)
- Verify plan の「S3 の diff は移したテストのコードと doc のコメントだけ」: diff の消した行は古い定数とその doc、README の検査の `TrimPrefix` の 1 行、古い doc の 7 行で、足した行は 2 つの定数と新しい関数とその doc

## Static analysis

| Command | Result | Notes |
| --- | --- | --- |
| `GOFLAGS=-timeout=30m ./scripts/run-static-verify.sh`(HEAD 55338bbc) | rc 0、`All verifiers passed.` | changed scope で golang を選んだ。shellcheck、hook の `sh -n`、guard の awk の構文、settings.json の jq、Codex hook の 3 つの guard、check-sync(IDENTICAL 171、DRIFTED 0、ROOT_ONLY 0)、check-pipeline-sync、check-skill-sync(14 skill)、check-template-purity、tech-debt README の計画の参照が OK。golang は `gofmt: ok`、go vet は無出力、golangci-lint `0 issues.`、staticcheck は無出力(どちらも PATH にある)。branch の secret scan は `b55d3d73..55338bbc` で clean |
| `GOTOOLCHAIN=go1.25.8 go build ./...`、`go vet ./internal/org/` | どちらも rc 0 | go.mod は `go 1.25.8`、手元は go1.26.0 |
| `gofmt -l internal/org/` | 出力なし | |
| `git diff --check b55d3d73...HEAD` | rc 0 | diff に EF BF BD の byte 列は 0 件、ホームのパスも 0 件 |

## Documentation drift

| Doc / contract | In sync? | Notes |
| --- | --- | --- |
| report の details の欄と `sent` の本文の切り方を書く文書 | 合う | `git grep -n 'truncateForDetails\|<br>' -- docs .claude internal/org/prompts` で出るのは、tech-debt の 2 行、試行の evidence、仕様 141 行の注、この計画、過去の報告(その時点の記録)と、`.claude/skills/plan/diagrams.md:235` の SVG の `<br>`(別の話)だけ。`docs/specs`・`.claude/skills`・`.claude/rules`・`.agents`・`templates/base`・`README.md`・`AGENTS.md`・`internal/org/prompts`・`docs/recipes` に「Event timeline」「200 byte」「details cell」を書く場所はない |
| `docs/reports/README.md` と `templates/base/docs/reports/README.md` | 合う | `cmp` で同一。この PR では変わっていない |
| 仕様の印と Status | 合う | 印の数は base と同じ(上の AC6)。Status の 20 行目が挙げる食い違う所見(F3、F4、F7、F10、F13、D-F3、「leader への ALERT」)に F12・F14 は入っておらず、変える必要はない。evidence の「項目」表の 60 行「F12 と F14 は記録の見た目の所見で、食い違わない」は今も正しい |
| 計画の AC の checkbox と Progress checklist | 遅れている(V1) | 下の Findings |

## Observational checks

AC3b の見本は、`internal/org/report.go` の `reportCellReplacer` の宣言と `report_test.go` の `reportCellCase`・`reportCellCases` を awk で抜き出して scratchpad の Go のプログラムに写し、`BuildOrgReport` の timeline の節と同じ形(`## Event timeline`、見出し、区切り、`"| %s | %s | %s | %s |\n"` の行、空行、`## Model receipts`)で書いて作った(写しで作り、手で置換を当ててはいない)。11 事例は header 行と空行、CRLF、単独の CR、LF のあとの CR、改行の隣の `|`、行末の `\`、フェンス、`<path>`、既にある `\|`、ふつうの文、空の Details。`gh api markdown -f mode=gfm -F text=@sample.md` は 1 回だけ打ち、rc 0 で 1,357 byte の HTML を返した。

見本の行(抜粋):

````text
| 2026-08-02T10:00:00Z | s1 | sent | TYPE: TASK<br>TASK_ID: t-1<br><br>body |
| 2026-08-02T10:00:04Z | s1 | sent | a\|<br>\|b |
| 2026-08-02T10:00:05Z | s1 | sent | go test \\<br>  ./internal/org/ |
| 2026-08-02T10:00:06Z | s1 | sent | \`\`\`sh<br>ralph org report<br>\`\`\` |
| 2026-08-02T10:00:07Z | s1 | sent | write \<path> then stop |
| 2026-08-02T10:00:08Z | s1 | sent | a\\\|b |
````

描画された details の欄(同じ行の抜粋):

````text
<td>TYPE: TASK<br>TASK_ID: t-1<br><br>body</td>
<td>a|<br>|b</td>
<td>go test \<br>  ./internal/org/</td>
<td>```sh<br>ralph org report<br>```</td>
<td>write &lt;path&gt; then stop</td>
<td>a\|b</td>
````

- `<tr>`: thead に 1、tbody に 11(事例 11 件)。table は 1 つで、空行で表が切れていない
- 各行の `<td>` は 4 つ(11 行すべて)。`<th>` も 4 つ
- `<br>` は 13 で、見本の Details の改行の数(3、3、1、2、1、1、2)の合計と同じ。CRLF の行も LF の行と同じ `<br>` 3 つになった
- `<code` は 0。フェンスのバッククォートは文字のまま出た
- 行末の `\` は `\` 1 つで、そのあとの `<br>` は要素のまま。`<path>` は `&lt;path&gt;` で、ブラウザでは `<path>` と見える。既にある `\|` は `a\|b` と書いたとおりに出た
- 比べるための描画(変更前の `|` だけの escape)はしていない。計画が許すのは 1 回の描画だけなので、変更前に表が崩れることは試行の report(tech-debt の F14 の行)の記録に頼った

## Findings

| ID | Severity | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- |
| V1 | INFO | 計画の AC1〜AC7・AC3b の checkbox がすべて `[ ]` のままで、Progress checklist の「Review artifact created」も `[ ]`。self-review の報告は 1d3ce15d からコミット済み。Progress の注に、self-review の修正 cab879de と 8a52987f の記録がない | 計画 159〜203 行、270〜275 行 | `/sync-docs` で checkbox と 2 つの修正の 1 行を足す。AC の可否は変わらない |
| V2 | INFO | F14 の RESOLVED のコメントは `TestBuildOrgReport_DetailsCellKeepsEachEventOnOneRow` の事例を括弧で 7 つ挙げるが、テストの事例は 11 ある(LF のあとの CR、既にある `\|`、ふつうの文、空の Details が挙がっていない)。「each case (…)」は全部を挙げたように読める。誤りではなく、HTML コメントの中なので描画では見えない | `docs/tech-debt/README.md:325`、`internal/org/report_test.go:123-136` | 直さなくてよい。直すなら括弧の前に「such as」を足すか、4 つを足す |

## Coverage gaps

- 手順の逸脱: 静的な確かめのつもりで `GOTOOLCHAIN=go1.25.8 go test -count=1 -run XXX_NONE ./internal/org/` を 1 回打った。テストの binary をコンパイルしただけで、どのテストも走っていない(`[no tests to run]`)。/verify はテストを打たない決まりなので、この結果は証拠に使っていない。テストファイルのコンパイルは `go vet` が受け持っている。repo のファイルには何も書いていない
- AC1〜AC5 のテストと AC7 のテスト部分(`go test ./internal/org/` を既定と `GOTOOLCHAIN=go1.25.8` で、`run-verify.sh` のテスト部分)は回していない(`/test` の担当)。AC1 と AC3 は実装とテストの期待を読み、AC3 は置換の組と事例を写したプログラムで期待と一致することを確かめただけで、Go のテストそのものが通るかは確かめていない
- AC3b の見本は写しから作った。`BuildOrgReport` を直接呼んではいない(写しの抜き出しは awk で、手で書き直していない)
- self-review の L2 の後半(evidence の表の「直す PR」欄、仕様 18 行、根拠の文書の PR 一覧に PR の番号を足す)は PR の番号が出来てからの作業で、この PR の範囲にない
- 決定的な verifier は足りている。GFM の描画は自動の検査がなく、計画どおりこの 1 回の描画が証拠になる。行と欄の数はテストが受け持つ

## Verdict

- Verdict: pass(CRITICAL・HIGH・MEDIUM・LOW はない。INFO 2 件(V1、V2)は記録の文面で、V1 は `/sync-docs` で直せる)
- Verified: AC3b(GFM での 1 回の描画: `<tr>` 11、各行 4 欄、`<br>` 13、`<code>` 0、`\`・バッククォート・`<path>`・`\|` が書いたとおり)、AC6(tech-debt の 2 行の形と欄の数、名指すテストの実在、evidence の 2 行、仕様 141 行、印の数 20 と 12)、AC5 の定数の場所と検査の移動(`t.Errorf`/`t.Fatalf` 17 = 11 + 6、移した範囲の byte 一致)、AC1〜AC3 の実装と事例の対応、置換の写しがテストの期待 11 件と一致すること、静的解析(run-static-verify rc 0、go1.25.8 の build と vet、`gofmt -l`、`git diff --check`)、文書のずれ(report の欄と切り方を書く文書、README の 2 つの写し、仕様の Status)、計画の digest
- Partially verified: AC7(静的な半分だけ)、AC1〜AC5(テストのコードと実装を読み、AC3 は写しで期待を当てた。テストの実行は `/test`)
- Not verified: Go のテストの実行、`BuildOrgReport` を直接呼んだ出力の描画、変更前の escape での描画(比べる描画)
