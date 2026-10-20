# Verify report: org-report-names

- Date: 2026-10-20
- Plan: `docs/plans/active/2026-10-20-org-report-names.md`(試行 F8 の修正。Status: Approved、`plan-visual.sh digest` は `b6c0dd650dcc` で `- Approved:` 行と一致)
- Verifier: verifier subagent(Claude Code、pipeline cycle 1)
- Scope: `git diff 8dae179f...HEAD`(HEAD 3b849c35、12 ファイル、+583/-7)。AC1〜AC7、計画の Verify plan、Non-goals、Assumptions。self-review の報告 `docs/reports/self-review-2026-10-20-org-report-names.md`(Merge: yes、修正 3ff6abdc の再確認も yes)
- Evidence: `docs/evidence/verify-2026-10-20-org-report-names.log` と、run-static-verify.sh が書いた `docs/evidence/verify-2026-10-20-141413.log`(どちらも `docs/evidence/*.log` の gitignore でコミットしない生の出力)

## Spec compliance

テストの実行は `/test` の範囲なので、AC2〜AC4 はテストの名前とコードを読み、AC4 はテストの文字列の検査を scratchpad の python に写して HEAD と base の両方に当てた。

| Acceptance criterion | Status | Evidence |
| --- | --- | --- |
| AC1: 未追跡の `review-2026-10-20-x-reviewer.md` と `org-manifest-x-2026-10-20.md` で `check-sync.sh` が rc 0、どちらも `ROOT_ONLY` に出ない。未追跡の `other-x.md` では rc 1 でそのファイルを名指す | 満たす | 何も置かない状態で rc 0。2 つを置くと rc 0、`ROOT_ONLY:      0`、2 つの名前はどの行にも出ない。2 つを消して `other-x.md` を置くと rc 1、`ROOT_ONLY      docs/reports/other-x.md`。消したあとの `git status --porcelain` は 0 行。変更前の `check-sync.sh`(`git show 8dae179f:` を `scripts/check-sync.sh.base` に置いた。自身の除外 `scripts/check-sync.sh` の前方一致に入る)に同じ 2 つを置くと rc 1 で 2 つとも `ROOT_ONLY` になり、rc 0 に変えたのは今回の 2 行だと確かめた。除外の配列を `eval` で読み込んで前方一致を当てると、`docs/reports/templates/review-report.md`・`docs/reports/README.md`・`docs/reports/reviewer-x.md` は除外に入らず、2 つの名前だけが `docs/reports/review-` と `docs/reports/org-manifest-` に入る |
| AC2: 描画した reviewer の prompt が `## レポート契約` の第 1 項目で `docs/reports/review-<YYYY-MM-DD>-<org_id>-<seat_id>.md` を名指し、`TestRenderRolePrompt_Reviewer_ResultCarriesHeadAndBase` が検査する。seat id の違う 2 つは別の名前で、どちらも `docs/reports/review-` で始まる | 満たす(テストの実行は `/test`) | 雛形 `internal/org/prompts/reviewer.md:116` が `` `docs/reports/review-<YYYY-MM-DD>-{{ORG_ID}}-{{SEAT_ID}}.md` `` を書き、`RenderRolePrompt`(`internal/org/prompts.go:87-100`)の `strings.NewReplacer` が `{{ORG_ID}}`・`{{SEAT_ID}}` を埋める。`prompts_test.go:1028-1064` が第 1 項目の名前を抜き出し、`docs/reports/review-<YYYY-MM-DD>-org-a-reviewer.md` と等しいこと、seat `reviewer-2` の名前と違うこと、2 つとも `reviewerReportPrefix` で始まることを検査する。org id と seat id は `identifierPattern` `^[a-z][a-z0-9-]{0,29}$`(`internal/org/identifier.go:38`)を通るので、名前にパスの区切りは入らない。第 2 項目(`leader へは RESULT ...`、119〜123 行)は base と byte 単位で同じ(5 行、sha256 `80354c6b…22bd` が一致) |
| AC3: `report.go` が名前の付いた定数から org report のパスを組み、書く名前は `org-manifest-<org_id>-<date>.md` のまま | 満たす(既存テストの実行は `/test`) | `internal/org/report.go:18` `const orgReportFilePrefix = "org-manifest-"`、64 行 `fmt.Sprintf("%s%s-%s.md", orgReportFilePrefix, p.OrgID, date)`。base の `"org-manifest-%s-%s.md"` と同じ文字列になる。名前を直書きで持つ既存のテストは `TestOrgReport_WritesFileFilteredByOrgAndUsesInjectedClock`(`report_test.go:175`)、`TestOrgReport_DefaultOutDir_WritesUnderDocsReports`(220・224 行)、`TestOrgReport_CLI_WritesFileWithRosterTimelineAndReceipts`(`internal/cli/org_test.go:3032`) |
| AC4: reviewer の雛形、`check-sync.sh` の除外、`docs/reports/README.md`、`/org` skill の `report` 行のどれかが 2 つの名前を書かなくなると落ちるテストが `internal/org` にあり、無いファイルは skip する | 満たす(テストの実行は `/test`) | `TestReportNames_DocsMatchConstants`(`internal/org/report_names_sync_test.go:95`)の subtest 4 つ(98・113・131・150 行)。無いファイルは `readRepoFileOrSkip` が subtest ごとに `t.Skipf` し、`readRepoFile` の「無い」「読める」「ディレクトリ」は `TestReadRepoFile_MissingFileIsNotAnError`(166 行)が押さえる。4 つの検査を python に写して当てた結果: HEAD では 4 つとも PASS、base では雛形・除外・README の 3 つが FAIL で skill の行だけ PASS(skill の行はこの PR で変えていない)。HEAD の写しで skill の `report` 行の `org-manifest-` を `org-report-` に変えると、その検査だけが FAIL になる |
| AC5: `docs/reports/README.md` と template の写しが 2 つの報告を挙げ、同一のまま(`check-sync.sh` rc 0) | 満たす | 2 つの README の 10・11 行(用途)と 20・21 行(Suggested naming)に 2 つが入る。`cmp` で同一。run-static-verify の check-sync は `DRIFTED: 0`・`ROOT_ONLY: 0` |
| AC6: F8 の行が `RESOLVED 2026-10-20 in fix/org-report-names`、F11 の行の Trigger が「the separate PR of the `check-sync.sh` row above」を書かず開いたまま、evidence の F8 が修正のブランチを名指す | 満たす | `docs/tech-debt/README.md:319` が `<!-- RESOLVED 2026-10-20 in fix/org-report-names: … -->`、320 行が Debt 欄の `~~…~~ (RESOLVED 2026-10-20 in fix/org-report-names)` と Trigger 欄の打ち消しと `— RESOLVED:`。321 行(F11)は打ち消しも RESOLVED もなく、Trigger は「The next change to the session-end hook or to where `ralph org report` writes: …」で、旧文「The separate PR of the `check-sync.sh` row above」はファイルのどこにもない。エスケープされていない縦棒で分けると 320・321 行は 5 欄。`docs/evidence/org-trial-2026-10-20.md:771-775` が `fix/org-report-names` を名指す |
| AC7: `./scripts/run-verify.sh` が rc 0、`internal/org` のテストが `GOTOOLCHAIN=go1.25.8` で通る | 静的な半分だけ確かめた(テストは `/test`) | `GOFLAGS=-timeout=30m ./scripts/run-static-verify.sh` が rc 0。`GOTOOLCHAIN=go1.25.8` で `go build ./...` rc 0、`go vet ./internal/org/ ./internal/cli/` rc 0(vet はテストファイルもコンパイルする)。`run-verify.sh` のテスト部分と `go test` は回していない |

計画のほかの項目:

- Assumptions: `scripts/check-sync.sh` は自身の除外リスト(34 行)にあり、`templates/` に写しも参照もない。`docs/reports/review-` の前方一致は `docs/reports/templates/review-report.md` を覆わない(上の `eval` の確かめ)。template 側の `review-report.md` は root と同一
- Non-goals: diff に leader の雛形・`/org` skill・`internal/insights/backfill.go`・`self-review-` の前方一致の変更はない。`ralph org report` の出力先も変えていない(`defaultReportDir` はそのまま)
- self-review の L1〜L3 の直し(3ff6abdc)は HEAD に入っている。`report_names_sync_test.go:70-94` の冒頭は検査する 4 か所と検査しない場所を分けて書き、register の 319 行は名前ごとに結ぶ場所を書く

## Static analysis

| Command | Result | Notes |
| --- | --- | --- |
| `GOFLAGS=-timeout=30m ./scripts/run-static-verify.sh`(HEAD 3b849c35) | rc 0、`All verifiers passed.` | `scripts/check-sync.sh` が言語の分類に入らないため full に落ちた(`unclassified:scripts/check-sync.sh`)。shellcheck、hook の `sh -n`、guard の awk の構文、settings.json の jq、Codex hook の 3 つの guard、check-sync(IDENTICAL 171、DRIFTED 0、ROOT_ONLY 0)、check-pipeline-sync、check-skill-sync(14 skill)、check-template-purity、tech-debt README の計画の参照が OK。golang は `gofmt: ok`、go vet は無出力、golangci-lint `0 issues.`、staticcheck は無出力(`command -v staticcheck` あり)。branch の secret scan は `8dae179f..3b849c35` で clean |
| `GOTOOLCHAIN=go1.25.8 go build ./...`、`go vet ./internal/org/ ./internal/cli/` | どちらも rc 0 | go.mod は `go 1.25.8`、手元は go1.26.0 |
| `staticcheck ./internal/org/` | rc 0 | |
| `shellcheck scripts/check-sync.sh`(0.11.0、既定の severity)、`bash -n` | どちらも rc 0 | base の script も shellcheck rc 0 |
| `git diff --check 8dae179f...HEAD` | rc 0 | |

## Documentation drift

| Doc / contract | In sync? | Notes |
| --- | --- | --- |
| `/org` skill の締めの手順(`.claude/skills/org/SKILL.md:825-860`)と leader の雛形の手順 9〜10(`internal/org/prompts/leader.md:170-200`) | 合う | どちらもこの PR で変わっていない。reviewer のレポートを「reviewer が RESULT や BLOCKED に書いた、`docs/reports/` の下のファイル」として手順 1(雛形は 9)でコミットし、report は手順 2(雛形は 10)で report だけをコミットする。名前を書かない一般の言い方なので、reviewer の名前が決まっても食い違わない。`report` 行(180 行)の名前は 4 つの写し(`.claude`・`.agents`・`templates/base` の 2 つ)で同じ |
| `docs/reports/README.md` と `templates/base/docs/reports/README.md` | 合う | `cmp` で同一。この文を持つファイルはこの 2 つだけ |
| `docs/reports/` の報告の prefix を並べるほかの文書 | 合う | prefix を並べるのは 2 つの README と `check-sync.sh` の除外だけ。`.claude/skills/pr/` の `walkthrough-` と `subagent-policy.md:28` の `self-review-*` は 1 つの報告を名指すだけで、一覧ではない。`scripts/gc-artifacts.sh` は `docs/reports/*.md` を名前の日付で扱い、prefix を見ない。`docs/specs/`・`docs/quality/`・`docs/recipes/` に reviewer の報告の名前や ROOT_ONLY の除外を書く場所はない。旧文「self-review / cross-review 系のレポート命名規約」はどこにも残っていない |
| `docs/tech-debt/README.md` の欄の数 | 合う | 指定の awk の出力は `22: 7`・`32: 5`・`278: 10` の 3 行だけで、base と同じ |
| `docs/tech-debt/README.md` の行どうしの参照 | 合う | F11 の行(321 行)の Impact 欄「the decision of the `check-sync.sh` row above」は試行の記録で、閉じた行を指してよい。Why 欄と Trigger 欄は F8 の行が org report を動かさずに閉じたことを書く。160 行の「separate PR」は別の件 |
| 計画の Progress checklist | 遅れている(V1) | 下の Findings |

## Observational checks

- AC1 の 3 回の run と、変更前の script での run は、生ログの `## AC1` の 2 節にある。どの run のあとも作ったファイルは消し、最後の `git status --porcelain` は 0 行
- reviewer が試行で選んだ名前 `review-2026-10-20-doctor-escape.md`(seat id なし、evidence 757 行)も `docs/reports/review-` の前方一致に入るので、古い雛形のまま走る座席の報告も gate を通る。計画のリスク欄の 1 つ目のとおり

## Findings

| ID | Severity | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- |
| V1 | INFO | 計画の Progress checklist の「Review artifact created」が `[ ]` のまま。self-review の報告は a462abd9 と 3b849c35 でコミット済み。Progress の S1〜S3 の注にも、self-review の修正 3ff6abdc(L1〜L3 と L4 の一部)の記録がない | 計画 230 行、216〜229 行 | `/sync-docs` で checkbox と 3ff6abdc の 1 行を足す。AC の可否は変わらない |

## Coverage gaps

- AC2〜AC4 のテストと AC7 のテスト部分(`go test ./internal/org/...` を既定と `GOTOOLCHAIN=go1.25.8` で、`run-verify.sh` のテスト部分)は回していない(`/test` の担当)。AC4 はテストの文字列の検査を python に写しただけで、Go のテストそのものが落ちるかは確かめていない
- 描画した prompt の文面は `RenderRolePrompt` のコードを読んで確かめただけで、描画はしていない
- self-review の L4 の残り(`prompts_test.go` が別のテストファイルの `reviewerReportPrefix` に頼る点)と L5(テストの長さと名前)は LOW のまま残っている。扱いは cross-review の triage か最終回の register に任せる
- 決定的な verifier は足りている。prefix の結びつきは `TestReportNames_DocsMatchConstants` が、除外の効き目は AC1 の run が受け持つ。AC1 の run を自動化する check(未追跡の `review-` と `org-manifest-` を置いて `check-sync.sh` を走らせる)は足せるが、除外の文字列はすでにテストが固定しているので、今回は提案に留める

## Verdict

- Verdict: pass(CRITICAL・HIGH・MEDIUM・LOW はない。INFO 1 件(V1)は計画の Progress の遅れで、`/sync-docs` で直せる)
- Verified: AC1(未追跡の 3 つの名前での `check-sync.sh` の rc と出力、変更前の script との比べ、前方一致の範囲)、AC5、AC6、AC2・AC3 のコードと雛形の第 2 項目の byte 一致、AC4 のテストの文字列の検査が HEAD で通り base で落ちること、静的解析(run-static-verify rc 0、go1.25.8 の build と vet、staticcheck、shellcheck、`git diff --check`)、文書のずれ(締めの手順、README の 2 つの写し、prefix の一覧、register の欄の数と参照)、計画の digest
- Partially verified: AC7(静的な半分だけ)、AC2〜AC4(テストの名前とコードを読み、AC4 の検査は写しで当てた)
- Not verified: Go のテストの実行、描画した reviewer の prompt の実物
