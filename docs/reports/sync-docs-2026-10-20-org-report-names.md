# sync-docs report: org-report-names

- Date: 2026-10-21(branch の記録の日付は 2026-10-20 のまま)
- Plan: `docs/plans/active/2026-10-20-org-report-names.md`(承認の digest `b6c0dd650dcc` はこの run の前後で同じ。`./scripts/plan-visual.sh digest` で確かめた)
- Pipeline cycle: 1(`.harness/state/standard-pipeline/cycle-count.json` の値)。base は `8dae179f`、branch は `fix/org-report-names`。run 開始時の HEAD は `1786e2b4`(run 開始時の値で、この report を含む commit の SHA ではない)
- 先行 report: `docs/reports/self-review-2026-10-20-org-report-names.md`(Merge yes。L1〜L3 は直し済み、L4 の残りと L5 は LOW のまま)、`docs/reports/verify-2026-10-20-org-report-names.md`(pass。V1 は計画の checklist の遅れで、先に直してあった)、`docs/reports/test-2026-10-20-org-report-names.md`(pass。G1 は LOW、G2〜G5 は INFO)
- この run が変えたのは、tech-debt の 1 行の追加、計画の Progress checklist の 1 項目、この report、insight event。コード、skill、rule、先行 3 report、計画の本文には触っていない

## Summary

`8dae179f...1786e2b4` の差分で公開の面が変わったのは 3 つある。`scripts/check-sync.sh` が `docs/reports/review-` と `docs/reports/org-manifest-` を `ROOT_ONLY_EXCLUSIONS` に入れたこと、reviewer 座席の雛形(`internal/org/prompts/reviewer.md`)が報告の名前を `docs/reports/review-<YYYY-MM-DD>-{{ORG_ID}}-{{SEAT_ID}}.md` に決めたこと、`internal/org/report.go` が org report の名前の prefix を定数 `orgReportFilePrefix` に出したこと(書き出す名前は変わらない)。名前を手で書く文書のうち、`docs/reports/README.md` の 2 つの写し、tech-debt の F8 と F11 の行、evidence の F8 は差分が直している。

reviewer の報告の名前、org report の名前、check-sync が feature org の gate を落とすという話を、依頼の一覧にあった面と、repo 全体の grep から探した。いまの動きとして旧い動きを書く文書は見つからなかった。

直した点は、test report の G1(LOW)が register に載っていなかったこと。除外の判定を走らせる suite がないという穴で、tech-debt に 1 行足した。計画の Progress checklist は「Test artifact created」にだけ印を付けた。

## Changes made

| File | Change |
|------|--------|
| `docs/tech-debt/README.md`(326 行、末尾に追加) | test report の G1 を 1 行足した。区切りの `|` は隣の行と同じ数で、欄は 5 つ。計画の道は `docs/plans/active/` の形(`/pr` の `archive-plan.sh` が書き換える)。コードの参照は関数と test の名前だけで、行番号は使っていない。G2〜G5 は INFO で、test report に残した |
| `docs/plans/active/2026-10-20-org-report-names.md`(Progress checklist) | 「Test artifact created」に印を付け、test report(`1786e2b4`)の結果と G1・G2〜G5 を 3 行で書いた。本文(Progress checklist より上)は変えていない。digest は前後で `b6c0dd650dcc` |
| `docs/reports/sync-docs-2026-10-20-org-report-names.md` | 新規 |
| `docs/insights/events/2026-10-20-org-report-names.jsonl` | sync_docs の event を 1 行足した(下の「Insight event」) |

## Surfaces checked for drift

| Surface | 結果 | 根拠 |
|---------|------|------|
| `AGENTS.md` の repo map(97 行 `docs/reports/`、`scripts/` の行) | 変更なし | `docs/reports/` の行は org report も reviewer 座席の報告も前から挙げていない。この差分が新しい種類の報告を足したのではなく(org report の名前は同じ、reviewer の報告は名前が決まっただけ)、地図が間違ったわけではない。短い地図のままにする |
| `docs/architecture/repo-map.md`(33 行 `docs/reports/`、62 行 `scripts/`) | 変更なし | 上と同じ。check-sync の記述は「KNOWN_DIFFS が 0」の話で、除外の一覧は持たない |
| `.claude/rules/ralph/agent-messaging.md` | 変更なし | EVIDENCE はポインタという原則だけで、報告の名前を書かない。`subagent-policy.md` の `self-review-*.md` は pipeline の reviewer subagent の報告で、org の reviewer 座席ではない |
| `/org` skill(`.claude/skills/org/SKILL.md` と 3 つの写し) | 変更なし | `report` の行(180 行)は `org-manifest-<org_id>-<date>.md` で、定数と合う(同期テストが読む行)。「org の終わり方」の 1(834 行)、leader の締めの 7(1187 行)、締めの checklist(1317・1337 行)は、reviewer の報告を「RESULT や BLOCKED に書いた `docs/reports/` の下のファイル」と呼ぶだけで名前を持たない。新しい名前でも文が合う |
| `/org` skill の RESULT の例(1211 行 `EVIDENCE: docs/reports/self-review-foo.md`、4 つの写し) | 触らない | 下の「Left open」 |
| `internal/org/prompts/leader.md`(39・170〜173・192〜194・391 行) | 変更なし | reviewer の報告を「RESULT や BLOCKED に書いた、`docs/reports/` の下のファイル」と呼び、コミットを頼むだけ。名前を持たない |
| `docs/specs/2026-08-01-org-runtime.md`、`docs/specs/2026-10-07-org-multi-org-director.md` | 変更なし | reviewer の報告の名前も、check-sync が feature org を落とすことも書かない。check-sync は parity の gate としてだけ出る(前者 38・82 行、後者 108 行)。後者の 9 段目の試行の段落は F1〜F14 を証拠と tech-debt に記録したと書くだけで、F8 を開いたままとは書かない |
| `README.md`(264 行の `ralph org gate` の段落)、`docs/recipes/` | 変更なし | README は「the leader commits the reviewer's reports」と「commits the org report」と書き、名前を持たない。recipes に reviewer の報告、org report、check-sync の除外を書く文はない |
| `docs/reports/README.md` と `templates/base/docs/reports/README.md` | 合っている | 2 つの写しは `diff` で一致。「Use this directory for」と「Suggested naming」の 2 か所に reviewer 座席の報告と org report が入り、名前は reviewer の雛形と定数の綴りと合う |
| `internal/cli/org.go` の `ralph org report` の help(1931 行)と `internal/cli/org_test.go` の前方一致 | 合っている | どちらも `org-manifest-<org_id>-<date>.md` のまま。名前を変えていない。計画が範囲外と書いた箇所 |
| `docs/tech-debt/README.md` の F8 の行(319・320 行)と F11 の行(321 行) | 合っている | self-review と verify が欄の数、取り消し線、RESOLVED の comment を確かめてある。F11 の Why 欄と Trigger 欄が同じ事実を 2 回書く点(self-review の N2)は重複で、食い違いではない |
| `docs/evidence/org-trial-2026-10-20.md` の F8 | 合っている | 修正の branch が `fix/org-report-names` で、除外、報告の名前、テストの 3 つの内容が差分と合う |
| `docs/quality/quality-gates.md`、`docs/quality/definition-of-done.md` | 変更なし | check-sync の行(58 行)は templates と root の一致で、除外を持たない。definition-of-done は pipeline の順を持つだけで、報告の名前を持たない |
| 計画 | 合っている | Progress checklist に S1〜S3、self-review、verify、test の記録がある。残る未了は「PR created」だけ |

## Plan drift notes

- 計画の Non-goals は「leader の雛形と `/org` skill の締めの手順を変えない」と「F11 の行は開いたままにする」を挙げる。差分もこの run も、どちらにも触れていない(F11 の行は Why 欄と Trigger 欄の言い換えだけが差分にあり、開いたまま)
- G1 の tech-debt の行は計画の Scope にない。test report の LOW の穴を register に載せる規約(先例: 2026-10-20 の secret-scan-test-tmp)に合わせたもので、計画の本文は digest を保つために変えていない

## Left open

- `/org` skill の RESULT の例(1211 行)は reviewer 風の座席の完了報告で、EVIDENCE に `docs/reports/self-review-foo.md` を挙げる。reviewer の雛形は今回「`self-review-` で始まる名前は使わない」と書くので、例と雛形は食い違って見える。ただし例は typed protocol の形を示すもので、座席は雛形を読み、skill の例は読まない。gate が落ちる形でもない(`self-review-` は前から除外に入っている)。直すなら 4 つの写しに同じ編集が要り(`sync-skills.sh` と `check-skill-sync.sh`)、計画の Non-goals(skill の締めの手順は変えない)に隣り合うので、drift と数えず、この run では触らなかった。次の cycle で直すなら `docs/reports/review-2026-10-20-org-a-reviewer.md` の形が合う
- `AGENTS.md` と `docs/architecture/repo-map.md` の `docs/reports/` の行は、org report と reviewer 座席の報告を挙げない。org report は 2026-08 からあり、この差分の前から挙げていない行で、間違ってはいない。足すなら 2 行の編集で済む
- G1 は register に載せた。止めるには `tests/` に、`check-sync.sh` を走らせて除外の効きを見る小さな shell の suite を足す(未追跡の `review-` と `org-manifest-` で rc 0、`other-x.md` で rc 1)
- self-review の L4 の残り(`prompts_test.go` が別のテストファイルの `reviewerReportPrefix` に頼る点)と L5(reviewer のテストの長さと名前)は、self-review の「Tech debt identified」が「cap の最後の回で残るときだけ足す」と書くので、register に載せていない。cycle 2 で残るなら、そのときの sync-docs が 1 行にまとめる
- G2〜G5(機能の org の gate の端から端、座席の LLM が名前のとおり書くか、skill と README の写しの直接の読み、Linux)は INFO で、test report に残した

## Verification

| 確かめたこと | 結果 |
|--------------|------|
| `./scripts/check-sync.sh`(register と計画の編集のあと) | rc 0。`PASS: all files in sync.`(IDENTICAL 171、DRIFTED 0、ROOT_ONLY 0、TEMPLATE_ONLY 11、KNOWN_DIFF 5) |
| `./scripts/check-skill-sync.sh` | rc 0。`14 skill(s) in lock-step`。skill は編集していないので `sync-skills.sh` は走らせていない |
| `./scripts/plan-visual.sh digest docs/plans/active/2026-10-20-org-report-names.md` | `b6c0dd650dcc`(承認の digest と同じ) |
| tech-debt の新しい行 | 欄は 5 つで 325・324 行と同じ。引いた 5 つのパス(計画、test report、verify report、`scripts/check-sync.sh`、`internal/org/report_names_sync_test.go`)と `TestReportNames_DocsMatchConstants` は存在する。計画のパスは `scripts/verify.local.sh` が読む `docs/plans/(active|archive)/…` の形 |
| `docs/reports/README.md` の 2 つの写し | `diff` で一致 |

`go test`、`run-static-verify.sh`、`run-verify.sh` は走らせていない。この run が変えたのは markdown 2 file と insight event と、この report で、コードも skill も変えていない。

## Verdict

- Verdict: pass(sync-docs に stop condition はない)
- Not verified: 上の Left open の G2〜G5(test report の記録のまま)。`scripts/verify.local.sh` の既定の実行(全 test を走らせるので、この run では回さず、計画のパスの存在を手で確かめた)

## Insight event

`./scripts/insights-append.sh --slug org-report-names --flow standard --phase sync_docs --verdict pass --cycle auto --source skill` で `docs/insights/events/2026-10-20-org-report-names.jsonl` に 1 行足した。
