# Test report: org-report-names

- Date: 2026-10-20(JST 23:24〜23:59。evidence の取りまとめは 10-21 の 0 時)
- Plan: `docs/plans/active/2026-10-20-org-report-names.md`(試行 F8 の修正。reviewer 座席の報告と org report を `check-sync.sh` の除外に入れ、名前をテストで結ぶ)
- Tester: tester subagent(Claude Opus 5.5)。パイプラインの 1 回目
- Scope: HEAD c6824707(base 8dae179f)。behavioral test だけを実行した(静的解析は /verify の担当)。環境は macOS 26.5.2(arm64)、`/bin/sh` は bash 3.2.57、git 2.49.0、go1.26.0 と `GOTOOLCHAIN=go1.25.8`。load average(1 分)は開始時 5.5、`internal/org` の 1 本目の開始時 25(ほかの session の負荷)、それ以外は 5〜10
- Evidence: `docs/evidence/test-2026-10-20-org-report-names.log`(下の各実行の生の出力と mutation harness の本文。`$HOME` は `~`、session の scratchpad は `<scratchpad>` に伏せた)と、run-test.sh が書いた `docs/evidence/verify-2026-10-20-142637.log`。どちらも `docs/evidence/*.log` の gitignore の対象で、コミットしない
- 追跡しているファイルは変えていない。worktree に置いたのは未追跡の 3 つの報告(E1、E2)だけで、消したあとの `git status --porcelain --ignored docs/reports` は置く前と同じ 0 行だった。文書と Go を変える確かめは、`git archive` で scratchpad に出した HEAD と base の写しで打った。各実行は順に 1 つずつ打ち、重ねていない

## Test execution

| Suite / Command | Tests | Passed | Failed | Skipped | Duration |
| --- | --- | --- | --- | --- | --- |
| 1. `GOFLAGS=-timeout=30m ./scripts/run-test.sh`(changed。`scripts/check-sync.sh` が言語に分類されず full に落ち、golang を選んだ) | shell 41 本、Go 8 パッケージ | shell 41 本すべて OK、Go 8 パッケージすべて ok | 0 | shell 1(下) | 716 秒、rc 0 |
| 2. `go test ./internal/org/... -count=1 -timeout 30m`(go1.26.0) | 3 パッケージ | 3 | 0 | 0 | 493 秒、rc 0。org 492.0 秒、driver 2.1 秒、protocol 1.9 秒 |
| 3. `GOTOOLCHAIN=go1.25.8 go test ./internal/org/... -count=1 -timeout 30m` | 3 パッケージ | 3 | 0 | 0 | 436 秒、rc 0。org 435.3 秒、driver 1.6 秒、protocol 1.5 秒 |
| 4. `go test ./internal/org/ -run 'TestRenderRolePrompt_Reviewer\|TestReportNames\|TestReadRepoFile\|TestOrgReport\|TestBuildOrgReport' -count=1 -v -timeout 30m` | 11 本と subtest 7 本 | 18 | 0 | 0 | 0.6 秒、rc 0 |
| 5. worktree の `./scripts/check-sync.sh`。E0 何も置かない、E1 未追跡の `review-2026-10-20-x-reviewer.md` と `org-manifest-x-2026-10-20.md`、E2 さらに `other-x.md` | 3 回 | E0・E1 は rc 0、E2 は狙いどおり rc 1 | 0 | - | 各 1 秒未満 |
| 6. scratchpad の写しの `check-sync.sh`(base と HEAD に E1 の 2 つ、T0〜T5) | 8 回 | 全回が下の期待どおり | 0 | - | 各 1 秒未満 |
| 7. mutation harness(HEAD の写しに 1 か所ずつ変更を入れ、4 の範囲を `-v` で回す) | 対照 1、mutant 22、描画の probe 1 | 対照は緑。mutant 18 本が赤、4 本が狙いの subtest だけ skip | - | - | 合わせて 235 秒 |
| 8. `go test ./internal/org/ -run 'TestOrgReport\|TestBuildOrgReport' -count=1 -coverprofile` | 6 本 | 6 | 0 | 0 | 0.9 秒、rc 0 |

- 1 の Go は cli(225.2 秒)、config(0.8 秒)、org(292.2 秒)を走らせ、insights、org/driver、org/protocol、scaffold、upgrade は `(cached)` だった。org/driver と org/protocol は 2 と 3 で `-count=1` でも通した。Go のテストの cache の鍵には依存のパッケージの中身が入るので、cached の結果は HEAD と同じ中身で前に走った結果である。shell の Skipped は前からある `tests/test-secret-scan-branch.sh` の 1 件で、git 2.49.0 なので 2.41 より古い本物の git の場合を飛ばしている
- 1 から 3 の `internal/org` 全体の 3 回で、記録にある load に弱いテスト(`TestRegate_FetchFailures/the_fetch_runs_past_the_bound`、`TestStopDirectorProcess_AProcessThatIgnoresSIGTERMIsNotEndedInTime`、`TestWakeDirector_AFailedWakeIsNotTriedAgainForFiveMinutes/no_return_in_time`)は 1 度も落ちなかった。2 は load 25 で始まり、1 の org(292 秒)より長い 492 秒かかったが、すべて通った。単独での再実行は要らなかった
- 4 の 11 本は、reviewer の prompt の 3 本(`TestRenderRolePrompt_Reviewer_AllKnownVarsSubstituted`、`..._MissionRunsGateFirstAndBlocksWithoutReviewing`、`..._ResultCarriesHeadAndBase`)、`TestReportNames_DocsMatchConstants`(subtest 4 本)、`TestReadRepoFile_MissingFileIsNotAnError`(subtest 3 本)、`TestBuildOrgReport_*` の 3 本、`TestOrgReport_*` の 3 本である
- 5 の E1 は `ROOT_ONLY:      0` で、2 つの名前はどの行にも出ない。E2 は `ROOT_ONLY      docs/reports/other-x.md` の 1 件だけで rc 1。E0 と E1 の数(IDENTICAL 171、DRIFTED 0、TEMPLATE_ONLY 11、KNOWN_DIFF 5)は同じ
- 6 の内訳:
  - base(8dae179f)の写しに E1 の 2 つを置くと rc 1 で、`docs/reports/org-manifest-x-2026-10-20.md` と `docs/reports/review-2026-10-20-x-reviewer.md` の 2 件が `ROOT_ONLY`。HEAD の写しでは同じ 2 つで rc 0。rc を 0 に変えたのはこの差分の 2 行である
  - T0 と T4 は HEAD の写しの対照で、どちらも rc 0
  - T1 は root の `docs/reports/templates/review-report.md` に 1 行を足した。rc 1、`DRIFTED        docs/reports/templates/review-report.md`
  - T2 は template 側の `templates/base/docs/reports/templates/review-report.md` を消した。rc 1、`ROOT_ONLY      docs/reports/templates/review-report.md`。前方一致 `docs/reports/review-` は `templates/` の下のこのファイルを覆わない
  - T3 は名前の近い 4 つ(`reviewer-x.md`、`review.md`、`templates/review-x.md`、`org-manifest.md`)を置いた。rc 1 で 4 件とも `ROOT_ONLY`。除外は `review-` と `org-manifest-` の後ろのハイフンまで含めて一致したときだけ効く
  - T5 は試行で reviewer が実際に書いた名前 `review-2026-10-20-doctor-escape.md`(seat id なし)を置いた。rc 0。古い雛形のまま走る座席の報告も gate を通る(計画のリスク欄の 1 つ目)
- 7 の mutant と結果は下の表のとおり。どの mutant も、赤になったのは狙った検査で、ほかのテストは緑のままだった。build が通らずに落ちた mutant はない(mutant ごとの `-v` の出力で `--- FAIL` の行と失敗の文言を確かめた)

| ID | 変えたところ | 結果 |
| --- | --- | --- |
| M00 | なし(対照) | rc 0 |
| M01 | 雛形の名前を `reviews-…` に | 赤: prompt の reviewer の検査、`reviewer_template` |
| M02 | 雛形の名前から `-{{SEAT_ID}}` を外す | 赤: 同上。prompt の検査は `prompts_test.go:1047`(期待の名前と違う)と `:1058`(seat `reviewer` と `reviewer-2` が同じ名前)の 2 つ |
| M03 | 雛形の名前を `## レポート契約` の外(末尾の新しい節)へ移す | 赤: 同上。節の外にある名前では通らない |
| M22 | 雛形の `{{ORG_ID}}` と `{{SEAT_ID}}` の順を入れ替える | 赤: 同上(`prompts_test.go:1047`) |
| M04 | `check-sync.sh` の配列から `"docs/reports/review-"` を消す | 赤: `check-sync.sh_exclusions` |
| M05 | 同じく `"docs/reports/org-manifest-"` を消す | 赤: 同上 |
| M06 | `"docs/reports/review-"` を `"docs/reports/review"` に | 赤: 同上 |
| M07 | `"docs/reports/review-"` を comment にする | 赤: 同上 |
| M08 | `"docs/reports/org-manifest-"` を配列の外(ファイルの末尾)へ移す | 赤: 同上 |
| M09 | README の reviewer の行から `-<seat_id>` を外す | 赤: `reports_README_naming_list` |
| M10 | README の org report の行を `org-report-…` に | 赤: 同上 |
| M11 | README の org report の行を `Use this directory for:` の一覧へ移す | 赤: 同上 |
| M12 | `/org` skill の `report` 行の名前を `org-report-…` に | 赤: `org_skill_report_row` |
| M13 | `/org` skill の `report` 行を消す | 赤: 同上(行がない) |
| M14 | `orgReportFilePrefix` を `"org-report-"` に | 赤: 文書の 3 つの subtest と `TestOrgReport_WritesFileFilteredByOrgAndUsesInjectedClock`、`TestOrgReport_DefaultOutDir_WritesUnderDocsReports` |
| M15 | `reviewerReportPrefix` を `…/reviews-` に | 赤: prompt の reviewer の検査、`reviewer_template`、`check-sync.sh_exclusions`、`reports_README_naming_list`(skill の行は reviewer の名前を持たないので緑のまま) |
| M16 | `scripts/check-sync.sh` を消す | rc 0。`check-sync.sh_exclusions` だけ skip |
| M17 | `docs/reports/README.md` を消す | rc 0。`reports_README_naming_list` だけ skip |
| M18 | `.claude/skills/org/SKILL.md` を消す | rc 0。`org_skill_report_row` だけ skip |
| M19 | 上の 3 つをすべて消す | rc 0。3 つの subtest が skip、`reviewer_template` は埋め込みの雛形を読むので走って通る |
| M20 | `readRepoFile` の「無いファイルは ok=false で error なし」の分岐を消す | 赤: `TestReadRepoFile_MissingFileIsNotAnError/a_missing_file_is_not_ok_and_not_an_error` |
| M21 | `readRepoFile` がどの読み込みの error も ok=false で返す | 赤: `TestReadRepoFile_MissingFileIsNotAnError/a_directory_is_an_error` |
| R01 | 写しに使い捨てのテストを足し、seat `reviewer` と `reviewer-2`(org `org-a`)の prompt を描画して第 1 項目を出す | `docs/reports/review-<YYYY-MM-DD>-org-a-reviewer.md` と `docs/reports/review-<YYYY-MM-DD>-org-a-reviewer-2.md`。どちらも `docs/reports/review-` で始まる |

## Coverage

- Statement: 8 の `report.go` の `Report` は 78.9%。この差分で変わった 64 行(`orgReportFilePrefix` を使う `Sprintf`)は実行回数 1 で通った。通らないのは `MkdirAll` と `WriteFile` の error の分岐で、この差分では変えていない。パッケージ全体の数は今回は測っていない
- Branch: 測っていない。新しいテストの helper の分岐は、`readRepoFile` の 3 つ(読める、無い、ほかの error)を `TestReadRepoFile_MissingFileIsNotAnError` が、`readRepoFileOrSkip` の skip を M16〜M19 が、`report` 行が見つからない分岐を M13 が通した。`Fatalf` で止まるほかの「見つからない」の分岐(`## レポート契約` の節、`ROOT_ONLY_EXCLUSIONS=(` の行と配列の終わりの `)`、`Suggested naming:` の行)は通していない。M08 と M11 は行を動かしただけで、これらの目印は残っている
- Function: 同上
- Notes: shell の `check-sync.sh` には instrumented な coverage の道具がない。変わったのは配列の 2 行で、5 と 6 で除外に入る名前と入らない名前の両方を通した

## Failure analysis

| Test | Error | Root cause | Proposed fix |
| --- | --- | --- | --- |
| なし | - | 1〜4 と 8 で落ちたテストはない。5 の E2、6 の base・T1・T2・T3 の rc 1 と、7 の mutant 18 本の赤は、確かめのためにわざと起こした失敗である | - |

## Regression checks

| Previously broken behavior | Status | Evidence |
| --- | --- | --- |
| F8: 機能の org の gate の静的の段で、`check-sync.sh` が reviewer 座席の報告と org report を `ROOT_ONLY` にして落ちる | 直っている。base は rc 1 で 2 件とも `ROOT_ONLY`、HEAD は rc 0 | 5 の E1、6 の base と HEAD |
| 除外が広がりすぎない(ほかの最上位の報告名は `ROOT_ONLY` のまま) | pass | 5 の E2、6 の T3 |
| `docs/reports/templates/review-report.md` は今までどおり比べられる | pass。書き換えると DRIFTED、template 側を消すと ROOT_ONLY | 6 の T1、T2 |
| org report のファイル名は `org-manifest-<org_id>-<date>.md` のまま(AC3) | pass。名前を直書きで持つ `TestOrgReport_*` が通り、M14 で赤になる | 4、7 の M14 |
| reviewer の `## レポート契約` の第 2 項目(RESULT の `HEAD:`・`BASE:`)の検査 | pass | 4 の `TestRenderRolePrompt_Reviewer_ResultCarriesHeadAndBase` |
| 全体(shell 41 本、Go 8 パッケージ、`internal/org` を 2 つの toolchain で) | pass | 1、2、3 |

計画の Edge cases:

- ほかの最上位の報告名が `check-sync.sh` で落ちる: 5 の E2(rc 1、`docs/reports/other-x.md` を名指す)
- `docs/reports/templates/review-report.md` が比べられる: 6 の T1 と T2
- 1 つの org の 2 つの reviewer 座席(`reviewer` と `reviewer-2`)が別の名前になる: 4 で `TestRenderRolePrompt_Reviewer_ResultCarriesHeadAndBase` が走って PASS。R01 の描画で 2 つの名前を確かめ、M02 と M22 で検査が赤になる
- 文書のファイルが無いときに同期のテストが skip する: 4 で `TestReadRepoFile_MissingFileIsNotAnError` の 3 本が PASS。M16〜M19 で、消したファイルの subtest だけが skip になり rc 0

## Test gaps

- G1(LOW): AC1 の振る舞い(未追跡の `review-` と `org-manifest-` の報告を置いて `check-sync.sh` が rc 0)を確かめるテストが suite にない。`tests/` に `check-sync.sh` の除外の判定を走らせる suite はなく、5 と 6 は suite の外の手の実行である。除外の文字列は `TestReportNames_DocsMatchConstants` が固定する(M04〜M08 が赤)ので、除外の行を消す変更は止まる。止まらないのは、`is_excluded` の前方一致の書き方を変える変更である。verify の Coverage gaps の提案と同じく、`check-sync.sh` の写しに未追跡の 2 つと `other-x.md` を置いて rc を見る小さな shell の suite で埋められる
- G2(INFO): 機能の org の gate そのもの(`ralph org gate` が機能の worktree で `run-static-verify.sh` を走らせる)は通していない。`run-static-verify.sh` は /verify の担当で、gate の端から端までの実行は機能の org を立てる必要がある。今回は gate の静的の段が呼ぶ `check-sync.sh` だけを、同じ未追跡の名前で走らせた
- G3(INFO): 座席の LLM が雛形の名前のとおりにファイルを書くかは、テストで確かめられない。名前が外れたときは gate が `ROOT_ONLY` で落ちる(5 の E2 と同じ形)。T5 のとおり、seat id のない試行の名前も前方一致に入る
- G4(INFO): `templates/base/docs/reports/README.md` と `/org` skill の写し(`.agents`、`templates/base` の 2 つ)は、同期のテストが直接は読まない。`check-sync.sh` と `check-skill-sync.sh` が同一に保つ(/verify の静的の段)
- G5(INFO): Linux では走らせていない。CI の verify(`ubuntu-latest`)が run-verify.sh を走らせる

## Verdict

- Verdict: pass
- Pass: 1(run-test.sh。shell 41 本と Go 8 パッケージ、716 秒、rc 0)、2(`internal/org/...`、go1.26.0、rc 0)、3(`internal/org/...`、go1.25.8、rc 0)、4(対象のテスト 18 件、rc 0)、5(E0・E1 rc 0、E2 は狙いどおり rc 1)、6(base は 2 件とも ROOT_ONLY、HEAD は rc 0、T1〜T3 は狙いどおり落ち、T0・T4・T5 は rc 0)、7(対照は緑、mutant 18 本が赤、4 本が skip、R01 の 2 つの名前が別)、8(`Report` の変わった行を通った)
- Fail: なし
- Blocked: なし
- 残る穴: G1(LOW)。除外の判定の振る舞いを確かめる suite がない。HEAD の除外は正しく、テストがないだけなので、マージを止めるものではない
