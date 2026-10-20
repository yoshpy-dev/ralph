# Walkthrough: org-report-names(9 段目の試行の F8 の修正)

- Date: 2026-10-21(ファイル名と branch の記録の日付は 2026-10-20)
- Plan: docs/plans/active/2026-10-20-org-report-names.md(/pr で archive に移す)
- Branch: fix/org-report-names(base 8dae179f、この walkthrough の前の HEAD 459fe801。13 commits、16 files、+896/-7。PR はこれから作る)
- 読む範囲: コードと README の 7 files が +256/-4 で、うち 192 行は新しい test の 1 file。tech-debt と evidence の 2 files は +9/-3。残りは計画(245 行)、後工程の報告 5 本、insight event 5 行で、+631/0
- 後工程は 1 回(cycle 1)で終えた。cross-review の指摘は 0 件

```
何を直すか → 計画 → check-sync の除外(S1) → 名前・雛形・同期 test(S2) → 記録(S3) → self-review の直し → 後工程の報告 → AC の根拠 → 確かめていないこと
```

## 1. この PR が何か

- 試行(`docs/evidence/org-trial-2026-10-20.md` の F8)で、機能の org の gate が落ちた。gate は `run-static-verify.sh` を走らせ、その中の `scripts/check-sync.sh` が、reviewer 座席の報告(試行では `review-<date>-<org_id>.md`)と `ralph org report` の org report(`org-manifest-<org_id>-<date>.md`)を `ROOT_ONLY` と数える。doctor-escape の 1 回目の gate が `static_failed` になった
- 直し方は consult-plan-f8 の案 B。除外を 2 つ足し、reviewer 座席の報告の名前を雛形で `docs/reports/review-<YYYY-MM-DD>-<org_id>-<seat_id>.md` に決め、名前を手で書く場所を test で結ぶ。案 C(org report の出力先を worktree の外へ移す)は Go の挙動と `/org` skill の締めの手順を変えるので退けた。案 D(`docs/reports/` 直下をすべて除外)は、新しい最上位の file の本物の `ROOT_ONLY` を隠すので退けた
- 名前に seat id を入れたのは Codex の計画助言(MEDIUM)。leader の雛形は reviewer 座席を複数置けるので、入れないと同じ org の 2 座席が同じ file に書く。`self-review-` を避けたのは、`internal/insights/backfill.go` が `self-review-` で始まる file を `/self-review` の報告として読むため
- F11(session-end の WIP hook が機能の worktree に commit する)は直さない。tech-debt の行は開いたままで、Trigger の文だけ書き直した

## 2. 読む順(commit ごと)

1. b7ce3d23 計画
   - Scope、Non-goals、Design decisions、AC1〜AC7 を読む。承認の digest は `b6c0dd650dcc`。後工程のあいだ本文は変えていない(Progress checklist だけが動く)
   - 見ること: Non-goals の「leader の雛形と `/org` skill の締めの手順は変えない」が、後の diff でも守られているか
2. 9349914b S1 `scripts/check-sync.sh`(+5)と `docs/reports/README.md` の 2 つの写し(各 +4)
   - `ROOT_ONLY_EXCLUSIONS` に `docs/reports/review-` と `docs/reports/org-manifest-` が入る。`is_excluded` は前方一致なので、末尾のハイフンまで含めて一致したときだけ効く
   - 見ること: `docs/reports/templates/review-report.md` が覆われず、今も比べられること(間に `templates/` が入る。test report の T1・T2)。`reviewer-x.md`、`review.md`、`org-manifest.md` が今も落ちること(T3)
3. 7a45cb14 S2 名前、雛形、test(4 files、+238/-4)
   - `internal/org/prompts/reviewer.md`: `## レポート契約` の第 1 項目だけを変えた。第 2 項目(`leader へは RESULT ...`)は base と byte 単位で同じで、`TestRenderRolePrompt_Reviewer_ResultCarriesHeadAndBase` が読む文を壊さない
   - `internal/org/report.go`: 定数 `orgReportFilePrefix` を足し、`Sprintf` を `"%s%s-%s.md"` にした。書き出す名前は変わらない
   - `internal/org/prompts_test.go`: 上の test の末尾に検査を足した。seat `reviewer` と `reviewer-2` の第 1 項目から `docs/reports/...md` を抜き、期待の名前と合うこと、2 つが違うこと、どちらも除外の prefix で始まることを見る
   - `internal/org/report_names_sync_test.go`: `TestReportNames_DocsMatchConstants` の subtest が 4 つ(雛形、`check-sync.sh` の除外、README の `Suggested naming:`、skill の `report` 行)。無い file は subtest ごとに skip する
   - 見ること: 抜き出す範囲が狭いか。除外は `ROOT_ONLY_EXCLUSIONS=(` から `)` の間の行を完全一致で見るので、comment や配列の外に同じ文字列があっても通らない。README は箇条書きの 1 行、skill は `` | `report` | `` で始まる行だけを見る
4. 5b7fe3c0 S3 記録
   - `docs/tech-debt/README.md` の F8 の行(319・320 行)を RESOLVED にした。F11 の行(321 行)は開いたまま、Trigger を書き直した。`docs/evidence/org-trial-2026-10-20.md` の F8 に修正の branch 名を足した
   - 見ること: 2 行とも欄が 5 つのままか(未エスケープの `|` で数える)。F8 の行の RESOLVED の文が、名前ごとに結ぶ場所を test の subtest どおりに挙げているか
5. 87ae8481 計画の Progress、a462abd9 self-review(Merge: yes、LOW 5、INFO 2)
6. 3ff6abdc self-review の修正(4 files、+20/-15)
   - L1: test の冒頭と定数の doc と register の文が、test の守る範囲より広く書かれていた点を、守る 4 か所と守らない場所に分けて書き直した
   - L2: F11 の行の Why 欄を、閉じた F8 の行を生きた依存として指さない文にした
   - L3: Go の comment が archive で壊れる計画の path を引いていたのを、slug 引きにした
   - L4 の一部: `reviewerReportPrefix` を `defaultReportDir + "/review-"` で組む形にした。3b849c35 の再確認で Merge: yes
7. 後工程の記録: dc98070b verify、c6824707 計画の Progress(verify の V1)、1786e2b4 test、015922f5 sync-docs(G1 の行を tech-debt の 326 行に足した)、459fe801 cross-review の triage。この 5 つは報告と記録で、コードは動かない

## 3. AC の根拠

| AC | 根拠 |
|----|------|
| AC1 | 手で走らせた結果。未追跡の `review-` と `org-manifest-` の 2 つを置くと rc 0 で `ROOT_ONLY: 0`、`other-x.md` を置くと rc 1 でその file を名指す。base の script に同じ 2 つを置くと rc 1 で、rc を変えたのはこの 2 行だと分かる(verify と test の 2 か所)。suite には入っていない(下の G1) |
| AC2 | `TestRenderRolePrompt_Reviewer_ResultCarriesHeadAndBase` の新しい検査。mutant の M01〜M03、M22(名前の綴り、seat id、節の外、順の入れ替え)で赤になる。描画の probe で `docs/reports/review-<YYYY-MM-DD>-org-a-reviewer.md` と `...-org-a-reviewer-2.md` を確かめた |
| AC3 | `report.go` の `orgReportFilePrefix` と `Sprintf`。名前を直書きで持つ既存の `TestOrgReport_*` が通り、M14(prefix を変える)で赤になる |
| AC4 | `TestReportNames_DocsMatchConstants` と `TestReadRepoFile_MissingFileIsNotAnError`。mutant 22 本のうち 18 本が狙った検査で赤、4 本(M16〜M19、対象 file を消す)は該当 subtest だけ skip で rc 0 |
| AC5 | 2 つの README が `cmp` で同一。`check-sync.sh` が rc 0(`DRIFTED: 0`、`ROOT_ONLY: 0`) |
| AC6 | tech-debt の 319〜321 行と evidence の F8 を verify が読んだ。未エスケープの `|` を数える awk が挙げる崩れた行は 22、32、278 行の 3 つだけで、base と同じ。319〜321 行は入らない |
| AC7 | `run-static-verify.sh` rc 0(verify、HEAD 3b849c35)。`run-test.sh` rc 0(shell 41 本と Go 8 パッケージ、test、HEAD c6824707)。`internal/org` を go1.26.0 と go1.25.8 で通した。`run-verify.sh` を 1 本のコマンドとしては走らせていない |

## 4. 後工程の報告

- self-review(`docs/reports/self-review-2026-10-20-org-report-names.md`): Merge: yes。LOW 5(L1〜L5)、INFO 2。再確認で INFO 2(N1、N2)を足した。N1 は `report_names_sync_test.go` の冒頭の列挙が閉じて読めること、N2 は F11 の行の同じ事実の重複で、どちらも直していない
- verify(`docs/reports/verify-2026-10-20-org-report-names.md`): pass。INFO の V1(計画の checklist の遅れ)は c6824707 で直した
- test(`docs/reports/test-2026-10-20-org-report-names.md`): pass。G1 が LOW、G2〜G5 が INFO
- sync-docs(`docs/reports/sync-docs-2026-10-20-org-report-names.md`): pass。G1 を tech-debt に 1 行足した
- cross-review(`docs/reports/cross-review-triage-org-report-names.md`): codex の指摘は 0 件で cycle 1/2。codex の sandbox が build directory を作れず、Go の test は走らせられなかった。tester が c6824707 で走らせた結果に頼り、その後の 015922f5 は tech-debt、計画の Progress、この task の報告と insight event しか変えていない
- insight event(`docs/insights/events/2026-10-20-org-report-names.jsonl`): self_review から cross_review まで 5 行、どれも cycle 1。記録だけの修正の例外は使っていない(triage の commit は 459fe801 の 1 つだけ)

## 5. 確かめていないこと

- gate を端から端まで: 機能の worktree で `ralph org gate` を走らせてはいない(test の G2)。gate の静的の段が呼ぶ `check-sync.sh` を、同じ未追跡の名前で走らせただけ
- LLM の reviewer 座席が雛形の名前で書くか(G3): test では確かめられない。外れた名前は gate が `ROOT_ONLY` で落とす(AC1 の `other-x.md` と同じ形)。雛形が古いまま走る座席のうち、試行の名前(seat id なし)は前方一致に入って通る(T5)
- `check-sync.sh` の除外を走らせる suite がない(G1、LOW): 除外の文字列は test が固定するので、行を消す変更は止まる。`is_excluded` の前方一致の書き方を変える変更と、除外を最上位の全 report に広げる変更は、どの suite にも当たらない。tech-debt の 326 行に載せた
- Linux(G5): 手元は macOS の bash 3.2 だけ。CI の verify(`ubuntu-latest`)が `run-verify.sh` を走らせる
- `/org` skill の RESULT の例が `EVIDENCE: docs/reports/self-review-foo.md` のまま(`.claude/skills/org/SKILL.md:1211`。`.agents` と `templates/base` の 3 つの写しも同じ)。雛形は今回「`self-review-` で始まる名前は使わない」と書くので、例と雛形は食い違って見える。例は typed protocol の形を示すもので座席が読むのは雛形、`self-review-` はもともと除外に入っていて gate は落ちないので、sync-docs は drift と数えず触らなかった。直すなら 4 つの写しに同じ編集(`sync-skills.sh` と `check-skill-sync.sh` を通す)で、形は `docs/reports/review-2026-10-20-org-a-reviewer.md`
- self-review の L4 の残りと L5: `prompts_test.go` の新しい検査が、別の test file の `reviewerReportPrefix` に頼る(その file を消すか移すと compile が落ちる)。`TestRenderRolePrompt_Reviewer_ResultCarriesHeadAndBase` は 56 行から 94 行になり、名前が中身の半分を言わない。どちらも register に載せていない(self-review が、載せるのは cap の最後の回で残るときだけと書いた)
- 描画した reviewer の prompt 全文を人が読んではいない。第 1 項目の名前を test と probe で確かめただけ
- `AGENTS.md` と `docs/architecture/repo-map.md` の `docs/reports/` の行は、org report と reviewer 座席の報告を挙げない。この差分の前からで、間違いではない。足すなら 2 行
- F11 は直していない。leader が skill の手順 2 で org report を commit したあと、session-end の WIP hook が worktree に残りを拾うかは、この PR の前と変わらない
