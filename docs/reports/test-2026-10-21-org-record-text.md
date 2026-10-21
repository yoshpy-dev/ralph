# Test report: org-record-text

- Date: 2026-10-21(JST 17:54〜18:50)
- Plan: `docs/plans/active/2026-10-21-org-record-text.md`(試行の所見 F12・F14 と、PR #244 の self-review が残した L4 の残り・L5 の修正)
- Tester: tester subagent(Claude Opus 5.5)。パイプラインの 1 回目。先に走った tester は報告を書く前に止まったので、最初からやり直した
- Scope: HEAD 6622bce0(base b55d3d73 = origin/main)。behavioral test だけを実行した(静的解析は /verify の担当)。環境は macOS 26.5.2(arm64)、`/bin/sh` は bash 3.2.57、git 2.49.0、go1.26.0 と `GOTOOLCHAIN=go1.25.8`。各実行の load average は下の「load と load に弱いテスト」に書いた
- Evidence: `docs/evidence/test-2026-10-21-org-record-text.log`(下の各実行の生の出力、mutation harness の本文、mutant ごとの `-v` の出力。`$HOME` は `~`、session の scratchpad は `<scratchpad>` に伏せた)と、run-test.sh が書いた `docs/evidence/verify-2026-10-21-085418.log`。どちらも `docs/evidence/*.log` の gitignore の対象で、コミットしない
- 追跡しているファイルは変えていない。base の確かめは scratchpad に `git worktree add --detach` した base の worktree で行い、終わったあと `git worktree remove --force` と `git worktree prune` で消した(`git worktree list` は main と task worktree の 2 つ、branch は増えていない)。mutant は `git archive 6622bce0` で scratchpad に出した写しに入れ、各回のあと byte 単位で戻した(写しの全ファイルの sha256 が開始時と一致)。重い実行は 1 つずつ順に打った

## Test execution

| Suite / Command | Tests | Passed | Failed | Skipped | Duration |
| --- | --- | --- | --- | --- | --- |
| 1. `GOFLAGS=-timeout=30m ./scripts/run-test.sh`(changed。golang を選んだ) | shell 41 本、Go 8 パッケージ | shell 41 本すべて OK、Go 8 パッケージすべて ok | 0 | shell 1(下) | 1,086 秒、rc 0 |
| 2. `GOTOOLCHAIN=go1.25.8 go test -count=1 -timeout 30m ./...` | 8 パッケージ | 8 | 0 | 0 | 880 秒、rc 0。cli 676.1 秒、org 874.3 秒、ほかは 1.3〜3.7 秒 |
| 3. 対象のテストを `go test ./internal/org/ -run '<下の正規表現>' -count=1 -v`(go1.26.0) | 17 本と subtest 26 本 | 43 | 0 | 0 | 0.58 秒、rc 0 |
| 4. 3 と同じ(`GOTOOLCHAIN=go1.25.8`) | 17 本と subtest 26 本 | 43 | 0 | 0 | 0.61 秒、rc 0。PASS の一覧は 3 と同じ |
| 5. base の worktree(b55d3d73)に HEAD の `verbs_test.go` と `report_test.go` だけを写し、新しいテストと `TestBuildOrgReport_*` を `-v` で回す | 7 本と subtest 22 本 | 下の表 | 狙いどおり落ちた(rc 1) | 0 | 1.3 秒 |
| 6. mutation harness(HEAD の写しに 1 か所ずつ変更を入れ、対象のテストを `-v` で回す) | 対照 1、mutant 26 | 対照は緑。mutant 25 本が赤 | 1 本が生き残った(V10) | - | 合わせて 164 秒 |
| 7. 生き残った V10 を `go test ./internal/org/ ./internal/cli/ -count=1 -v -timeout 30m` で回す | 2 パッケージ | 2(V10 は生き残る) | 0 | 1(darwin では飛ばす `TestEscalate_NoDesktopNotificationOffDarwinIsSkipped`) | 816 秒。org 813.4 秒、cli 708.1 秒 |
| 8. 3 の範囲のうち新しいテストと `TestBuildOrgReport_*` を `-coverprofile` で回す | 7 本 | 7 | 0 | 0 | 1.3 秒、rc 0 |

- 1 の Go は cli(449.9 秒)、config(1.9 秒)、org(533.7 秒)を走らせ、insights、org/driver、org/protocol、scaffold、upgrade は `(cached)` だった。このうち insights はテストの依存に `internal/org` を含む(`go list -deps -test`)。Go の test cache の鍵には依存のパッケージの中身が入るので、cached の結果は HEAD と同じ中身で前に走った結果である。2 では 8 パッケージすべてを `-count=1` で走らせた
- 1 の shell の Skipped は前からある `tests/test-secret-scan-branch.sh` の 1 件で、git 2.49.0 なので 2.41 より古い本物の git の場合を飛ばしている。shell の出力に、件数が 0 でない FAIL の行はない
- 3 と 4 の正規表現は `^(TestTruncateForDetails_CutsAtACharacterBoundary|TestOrgSend_JapaneseBodyOver200Bytes_DetailsStayValidUTF8|TestBuildOrgReport_.*|TestOrgReport_.*|TestRenderRolePrompt_Reviewer_.*|TestReportNames_DocsMatchConstants|TestOrgSend_Raw_BypassesValidation_RecordsRawInDetails|TestOrgSend_RawAndUnconfirmed_DetailsPrefixOrder)$`。17 本の内訳は、新しいテスト 4 本(`TestTruncateForDetails_CutsAtACharacterBoundary` と subtest 11 本、`TestOrgSend_JapaneseBodyOver200Bytes_DetailsStayValidUTF8`、`TestBuildOrgReport_DetailsCellKeepsEachEventOnOneRow` と subtest 11 本、`TestBuildOrgReport_TimelineHasOneLinePerEventOfFourCells`)、分けた `TestRenderRolePrompt_Reviewer_ReportContractNamesTheReport`、触った `TestRenderRolePrompt_Reviewer_ResultCarriesHeadAndBase` と `TestReportNames_DocsMatchConstants`(subtest 4 本)、前からある `TestBuildOrgReport_*` の 3 本(`RosterTimelineReceiptsAndResiduals`、`EscapesPipesInEventDetails`、`EmptyOrg_NoEventsNote`)、`TestOrgReport_*` の 3 本、reviewer の prompt の残り 2 本、`Send` の Details の prefix のテスト 2 本である

### 5 の結果(base の振る舞いに新しいテストを当てた)

| Test | base での結果 | 落ちた理由 |
| --- | --- | --- |
| `TestTruncateForDetails_CutsAtACharacterBoundary` | 11 事例のうち 5 つが FAIL | 200 byte 目が文字の途中にある 4 事例(199 ASCII + 3 byte 文字、197・198・199 ASCII + 4 byte 文字)で、base は 200 byte で切って文字の前半の byte を残した。不正な UTF-8 の事例は、base が 200 byte(継続 byte 10 個)を残し、期待の 197 byte(7 個)と違う。PASS の 6 事例は、base と HEAD の切り方が同じになる事例(文字が 200 byte 目で終わる、200 byte 目から始まる、ASCII だけ、200 byte 以下) |
| `TestOrgSend_JapaneseBodyOver200Bytes_DetailsStayValidUTF8` | FAIL | manifest から読み戻した Details が「…記録す」の後ろに U+FFFD を 1 つ持ち(`verbs_test.go:1375`)、残した文が 202 byte で本文の前方一致でない(`:1382`)。本文の 199 byte 目から始まる「る」の 1 byte 目だけが残り、JSON の encode がその 1 byte を U+FFFD(3 byte)に変えた。`utf8.ValidString` の検査は、JSON の encode が先に不正な byte を置き換えるので、この経路では発火しない。落としたのは U+FFFD と長さの検査である |
| `TestBuildOrgReport_DetailsCellKeepsEachEventOnOneRow` | 11 事例のうち 9 つが FAIL | base は改行をそのまま書く。header 行と空行の事例は timeline が 3 行でなく 6 行、CRLF も 6 行、フェンスは 5 行になった。単独の CR、`<path>`、既にある `\|` の 3 事例は行数は合うが、行の文字列が期待と違う(CR がそのまま、`<` に `\` がない、`\` が二重にならない)。PASS はふつうの文と空の Details の 2 つ |
| `TestBuildOrgReport_TimelineHasOneLinePerEventOfFourCells` | FAIL | 11 件の event で timeline が 13 行でなく 24 行になった(`report_test.go:203`) |
| `TestBuildOrgReport_RosterTimelineReceiptsAndResiduals`、`..._EscapesPipesInEventDetails`、`..._EmptyOrg_NoEventsNote` | PASS | 前からあるテストで、HEAD でも同じ期待のまま通る(AC4) |

### 6 と 7 の mutant

どの mutant も、`-v` の出力で `--- FAIL` の行と失敗の文言を確かめた。build が通らずに落ちた mutant はない。対照(M00、変更なしで同じ範囲)は rc 0。

| ID | 変えたところ | 結果 |
| --- | --- | --- |
| V1 | `truncateForDetails` が 200 byte で切る(base と同じ) | 赤: 文字の途中の 4 事例、不正な UTF-8 の事例、`Send` のテスト |
| V2 | 戻る上限を `utf8.UTFMax-1` から `utf8.UTFMax` に | 赤: 不正な UTF-8 の事例だけ(196 byte まで戻る) |
| V3 | 戻る上限を `utf8.UTFMax-2` に | 赤: 197 ASCII + 4 byte 文字の事例と不正な UTF-8 の事例 |
| V4 | 戻る上限をなくす | 赤: 不正な UTF-8 の事例(190 byte まで戻る) |
| V5 | 切る位置を後ろへ進める | 赤: V1 と同じ 5 事例と `Send` のテスト(200 byte を超える) |
| V6 | `len(text) <= 200` を `< 200` に | 赤: ちょうど 200 byte の事例で `index out of range [200]` の panic |
| V7 | 戻る条件を反転する | 赤: 切る 9 事例のすべてと `Send` のテスト(200 byte 以下の 2 事例は切らないので緑) |
| V8 | `sent` の記録だけ `p.Text[:200]` で切る | 赤: `Send` のテスト(`:1375` の U+FFFD、`:1382` の 202 byte) |
| V9 | `sent` の記録で本文を切らない | 赤: `Send` のテスト(`:1379` の `...(truncated)`) |
| V10 | dry-run の記録(`verbs.go:348`)で本文を切らない | 生き残った。対象の範囲でも、7 の `internal/org` と `internal/cli` の全体でも緑(下の G1) |
| R1 | `\` の escape を外す | 赤: 行末の `\`、既にある `\|`、timeline のテスト |
| R2 | `\|` の escape を外す | 赤: 前からある `EscapesPipesInEventDetails`、改行の隣の `\|`、既にある `\|`、timeline のテスト |
| R3 | バッククォートの escape を外す | 赤: フェンス、timeline のテスト |
| R4 | `<` の escape を外す | 赤: `<path>`、timeline のテスト |
| R5 | `\r\n` の組を外す | 赤: CRLF(`<br>` が 2 つになる)、timeline のテスト |
| R6 | `\r` の組を外す | 赤: 単独の CR、LF のあとの CR、timeline のテスト |
| R7 | `\n` の組を外す | 赤: 5 事例と timeline のテスト |
| R8 | `\r` の組を `\r\n` の組より前に置く | 赤: CRLF、timeline のテスト |
| R9 | 置換の前に `\|` を escape する(2 度の escape) | 赤: `EscapesPipesInEventDetails`、改行の隣の `\|`、既にある `\|`、timeline のテスト |
| R10 | timeline の欄を base の書き方(`\|` の escape だけ)に戻す | 赤: 9 事例と timeline のテスト(5 の base と同じ 9 事例) |
| R11 | `\n` を `<br/>` と書く | 赤: 5 事例と timeline のテスト |
| P1 | reviewer の雛形の報告名を `reviews-…` に | 赤: `ReportContractNamesTheReport` と `TestReportNames_DocsMatchConstants/reviewer_template`。`ResultCarriesHeadAndBase` は緑 |
| P2 | 雛形の報告名から `-{{SEAT_ID}}` を外す | 赤: P1 と同じ 2 つ。`ResultCarriesHeadAndBase` は緑 |
| P3 | 雛形のレポート契約の第 2 項目の `` `HEAD:` `` を `` `HEAD_SHA:` `` に | 赤: `ResultCarriesHeadAndBase` だけ(`prompts_test.go:1023`)。`ReportContractNamesTheReport` は緑 |
| P4 | テストの定数 `reviewerReportFilePrefix` を `"reviews-"` に | 赤: `ReportContractNamesTheReport`、`reviewer_template`、`check-sync.sh_exclusions`、`reports_README_naming_list` |
| P5 | `docs/reports/README.md` の reviewer の行を `reviews-…` に | 赤: `reports_README_naming_list` だけ |

- P1〜P3 で、S3 で分けた 2 つのテストがそれぞれ自分の検査を持つことを確かめた。報告名の変更は新しいテストだけを、`HEAD:` の文言の変更は残したテストだけを落とす。P4 と P5 で、`prompts_test.go` に移した `reviewerReportFilePrefix` を sync テストの README の検査が使うことを確かめた

### load と load に弱いテスト

| 実行 | 開始時の load average(1・5・15 分) | 終了時 |
| --- | --- | --- |
| 1 | 5.75 / 10.77 / 18.35 | 6.32 / 6.68 / 10.10 |
| 2 | 5.84 / 6.53 / 9.95 | 16.04 / 11.59 / 9.97 |
| 7 | 33.77 / 79.96 / 50.48(ほかの session の負荷を含む) | 5.60 / 18.01 / 32.71 |

`internal/org` の全体を 3 回(1、2、7)、`internal/cli` の全体を 3 回走らせ、記録にある load に弱いテスト(`TestStopDirectorProcess_AProcessThatIgnoresSIGTERMIsNotEndedInTime`、`TestRegate_FetchFailures/the_fetch_runs_past_the_bound`、`TestRunDoctorFull_PrintsTheMergeGuarantee`、`TestCheckScaffoldIntegrity_CorruptManifest_StrictFails`、`TestRunDoctorOpts_CodexSandboxCheck_HermeticByDefault`)は 1 度も落ちなかった。単独での再実行は要らず、`TestRegate_FetchFailures/the_fetch_runs_past_the_bound` の失敗の出力も取れていない(次の PR の原因調べに渡せるものはない)。7 は 5 分平均 80 の負荷の直後に始まったが、両パッケージとも通った。

## Coverage

- Statement: 8 で `truncateForDetails` 100%、`escapeReportCell` 100%、`BuildOrgReport` 98.2%。`BuildOrgReport` で通らなかったのは roster の `[dry-run]` の印の分岐(`report.go:185-187`)だけで、この差分では変えていない(8 の範囲の 7 本は dry-run の seat を作らない)。パッケージ全体の数は今回は測っていない
- Branch: 測っていない。`truncateForDetails` の分岐(200 byte 以下で返す、戻る、戻らない、上限で止まる)は 11 事例がすべて通し、V2〜V7 がそれぞれの境界で赤になった。`reportCellReplacer` の 7 組は、R1〜R8 と R11 がどの組を外しても赤になった
- Function: 同上
- Notes: AC3b の GFM の描画は /verify が `gh api markdown` で 1 回確かめた。/test では gh を呼んでいない

## Failure analysis

| Test | Error | Root cause | Proposed fix |
| --- | --- | --- | --- |
| なし | - | 1〜4、7、8 で落ちたテストはない。5 の FAIL と 6 の mutant 25 本の赤は、確かめのためにわざと起こした失敗である | - |

## Regression checks

| Previously broken behavior | Status | Evidence |
| --- | --- | --- |
| F12: 日本語の本文が 200 byte を超えると、`sent` の Details が文字の途中で切れ、manifest に U+FFFD が入る | 直っている。base は 4 つの文字の途中の事例と `Send` のテストで落ち、HEAD は通る | 3、4、5、6 の V1・V5・V8 |
| F14: Details の改行で timeline の行が割れ、空行で表が終わる | 直っている。base は timeline が 13 行でなく 24 行、HEAD は 1 件 1 行で各行 4 欄 | 3、4、5、6 の R5〜R8・R10・R11 |
| `\|` の escape(前からある `TestBuildOrgReport_EscapesPipesInEventDetails`)が 2 度の escape にならない | pass | 3、4、5、6 の R2・R9 |
| `Send` の Details の prefix(`raw=true `、`submit_unconfirmed=true `)の順(AC4) | pass | 3、4 の `TestOrgSend_Raw_BypassesValidation_RecordsRawInDetails` と `TestOrgSend_RawAndUnconfirmed_DetailsPrefixOrder` |
| S3 で分けた reviewer の検査が消えていない(AC5) | pass。2 つのテストがそれぞれの mutant で落ちる | 3、4、6 の P1〜P5 |
| 全体(shell 41 本、Go 8 パッケージを 2 つの toolchain で、AC7 のテスト部分) | pass | 1、2 |

計画の Edge cases:

- 200 byte をまたぐ各位置の 4 byte 文字、200 byte で終わる文字、ちょうど 200 byte、不正な UTF-8: `TestTruncateForDetails_CutsAtACharacterBoundary` の 11 事例が 3・4 で PASS、5 で base の 5 事例が FAIL
- 空行、`\r\n` と単独の `\r`、改行の隣の `|`、空の Details が `-` のまま: `reportCellCases` の 11 事例が 3・4 で PASS。空の Details は base でも HEAD でも `-`

## Test gaps

- G1(LOW): `Send` の dry-run の記録(`verbs.go:348`)が本文を 200 byte で切ることを確かめるテストがない。V10(dry-run の Details で本文を切らない)は、対象の範囲でも `internal/org` と `internal/cli` の全体でも生き残った。この穴は前からあり、base には本文の切り方を確かめるテストが 1 つもなかった。HEAD の dry-run は `sent` と同じ `truncateForDetails` を呼ぶので振る舞いは正しく、テストがないだけである。埋めるなら、`TestOrgSend_JapaneseBodyOver200Bytes_DetailsStayValidUTF8` と同じ本文を `DryRun: true` で送り、最後の event が `DryRun` で、Details が有効な UTF-8 で U+FFFD を含まず `...(truncated)` で終わることを見る 1 本で足りる
- G2(INFO): `Send` のテストの `utf8.ValidString` の検査は、manifest を通すと発火しない。JSON の encode が不正な byte を U+FFFD に置き換えてから書くので、読み戻した文字列は常に有効な UTF-8 になる(5 の base の結果)。同じテストの U+FFFD と長さの検査が base の振る舞いと V8 を落とすので、穴にはなっていない。`truncateForDetails` の戻り値そのものの `ValidString` は単体テストが見ている
- G3(INFO): AC3b(GitHub の GFM renderer での描画)はテストにない。計画のとおり /verify が 1 回だけ描画して確かめた
- G4(INFO): Linux では走らせていない。CI の verify(`ubuntu-latest`)が run-verify.sh を走らせる

## Verdict

- Verdict: pass
- Pass: 1(run-test.sh。shell 41 本と Go 8 パッケージ、1,086 秒、rc 0)、2(`./...` を go1.25.8 と `-count=1` で、880 秒、rc 0)、3 と 4(対象のテスト 43 件を両方の toolchain で、rc 0)、5(base では新しいテストが F12・F14 の振る舞いで落ち、前からあるテストは通る)、6(対照は緑、mutant 25 本が赤)、7(全体で回しても V10 だけが生き残る)、8(変えた 2 つの関数は 100%)
- Fail: なし
- Blocked: なし
- 残る穴: G1(LOW)。dry-run の記録の切り方を確かめるテストがない。HEAD の振る舞いは正しく、テストがないだけなので、マージを止めるものではない
