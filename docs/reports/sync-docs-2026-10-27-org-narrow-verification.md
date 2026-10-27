# sync-docs report: org-narrow-verification

- Date: 2026-10-28
- Plan: `docs/plans/active/2026-10-27-org-narrow-verification.md`(承認の digest `97abe345a188` はこの run の前後で同じ。`./scripts/plan-visual.sh digest` で確かめた。この run が変えたのは `## Progress checklist` の 1 項目と、`## Acceptance criteria` の AC1〜AC6 の `[ ]` → `[x]` だけ)
- Pipeline cycle: 1(`.harness/state/standard-pipeline/cycle-count.json` の値。上限は 2)。base は `622cdcfb`、branch は `perf/org-narrow-verification`。run 開始時の HEAD は `3afdd03a`(この report を含む commit の SHA ではない)
- 先行 report: `docs/reports/self-review-2026-10-27-org-narrow-verification.md`(2 回目、LOW 2、Merge yes。2 件とも F3 `8cda482e` で直した)、`docs/reports/verify-2026-10-27-org-narrow-verification.md`(`584f8103`、pass。LOW 1 と既知 1 を sync-docs に渡した)、`docs/reports/test-2026-10-27-org-narrow-verification.md`(`3afdd03a`、pass)
- 差分の大きさ(`git diff 622cdcfb...HEAD --shortstat`、この run の前): 20 ファイル、+1766/-100。walkthrough を書くかどうかは `/pr` が決める

## Summary

verify が渡した 2 件を直した。1 つは `docs/specs/2026-10-07-org-multi-org-director.md` の FR-7 で、「テストは review と gate の 2 回走る」と書く 2026-10-26 と 7b の記述を残したまま、日付つきの「2026-10-27 に決めたこと」と「印(2026-10-27)」を足した。もう 1 つは `docs/tech-debt/README.md` の「org runtime FR-7 quality gate is prompt-instructed」の行で、reviewer が `run-static-verify.sh` と `run-test.sh` を必ず打つと読める文を直し、この計画が trigger に当たったが払わず繰り越した注記を 2026-10-26 の注記のあとに足した。

そのほかの面(README、AGENTS.md、`.claude/rules/ralph/`、`docs/quality/`、`docs/recipes/`、`docs/architecture/`、`templates/base/` の写し、skill、CLI の help)を洗い直し、ほかに直す文は見つからなかった。`agent-messaging.md` は `VERIFY:` と `CHECKS:` を書かないままにした(下の表)。

## Changes made

| File | Change |
|------|--------|
| `docs/specs/2026-10-07-org-multi-org-director.md` | FR-7 の項目の下、「印(2026-10-26)」のすぐ後に 2 行を足した。(1) 「2026-10-27 に決めたこと」: 機能の org の reviewer のレビューの TASK はコマンドとして `run-static-verify.sh` だけを `CHECKS:` の行で名指しする。reviewer はその行だけから取って `CHECKS:` で返し、行が欠けたら leader が送り直す(2 回続けて通らなければ BLOCKED を escalate)。フルの test は gate の TASK で 1 回。`CHECKS:` の行がない TASK では既定の両方(計画 `2026-10-27-org-narrow-verification` の D6)。`ralph org gate`/`regate` は変えていない。implementer は `VERIFY:` の行かいちばん狭い検査と機械的な確かめ。`docs/specs/2026-08-01-org-runtime.md` の「2026-10-27 改訂」を指す。(2) 「印(2026-10-27)」: 雛形の文を固定するテスト 5 本を挙げ、実機の org では動かしていないこと、FR-7 は 2026-10-26 の印ですでに `[ ]` なので `[ ]` のままで、この変更を含む binary での次の実行の記録で付け直すことを書いた。2026-10-26 の項目と 7b の注は書き換えていない |
| `docs/tech-debt/README.md`(「org runtime FR-7 quality gate is prompt-instructed, not hook-enforced.」の行) | (1) 1 列目の「The reviewer seat re-runs `run-static-verify.sh` / `run-test.sh`」を「runs the commands its review TASK names (`run-static-verify.sh` only in a feature org, both ... when the TASK names none)」に直した。(2) Trigger 列の 2026-10-26 の注のあとに、2026-10-27 の繰り越しの注を足した(trigger に当たったが、Non-goals が `ralph org gate`/`regate` を変えず、Scope に seat が打ったものを確かめる hook や verb がないので払わない)。(3) Related 列に `docs/plans/active/2026-10-27-org-narrow-verification.md (Non-goals; the carry-over note)` を足した。`|` は 6 個のまま(5 セル)、`~~` なし。`archive-plan.sh` が `/pr` で `archive/` に書き換える |
| `docs/plans/active/2026-10-27-org-narrow-verification.md` | `## Progress checklist` の「Test artifact created」を `[x]` にして 1 行の結果(test `3afdd03a`: pass、`run-test.sh` rc 0・1,572 秒、shell 45 本と Go 9 パッケージ、名指しの 66 本と `TestStartFeature` 27 本、変異 a〜d、実機は未確認)を足した。`## Acceptance criteria` の AC1〜AC6 を `[ ]` → `[x]` にした。それ以外の節は触れていない(digest は同じ) |
| `docs/insights/events/2026-10-27-org-narrow-verification.jsonl` | `sync_docs` の event を 1 行追記した(cycle 1、verdict pass)。既存のファイルへの追記で、新しいファイルはできていない |
| `docs/reports/sync-docs-2026-10-27-org-narrow-verification.md` | この report |

## Surfaces checked for drift

verify が洗った面(`run-test.sh` と `run-verify.sh` の grep、README の gate の段落、`/org` skill の gate の節)を、この run で別の語(`VERIFY:`、`CHECKS:`、reviewer が実行する、implementer の検証)でも洗い直した。root と `templates/base/` の両方を見た。

| Surface | 結果 | 根拠 |
|---------|------|------|
| `.claude/rules/ralph/agent-messaging.md` | 変更なし | この rule が書くのは `ralph org send` が検証するもの(TYPE の表、`TASK_ID`、ヘッダの形、2,000 字の上限、TASK の `PLAN:` ヘッダ)で、`VERIFY:` と `CHECKS:` は TASK と RESULT の本文の行である。計画の D4 のとおり ralph は本文を読まない(`PLAN:` だけがヘッダとして検査される)。`GATE:`・`HEAD:`・`BASE:`・`EVIDENCE:` といった本文の行も、この rule には並んでいない。役割ごとの本文の約束は役割の雛形(`leader.md` の typed protocol の節)と `/org` skill にあり、そこに入っている。本文をファイルで渡す決まり(`.harness/state/`)は、shell の展開を避ける座席の運用で、protocol の契約ではない。2,000 字の上限は `VERIFY:` の行を入れても収まる(計画の Assumptions) |
| `.claude/rules/ralph/git-commit-strategy.md`、`subagent-policy.md`、`model-routing.md`、`post-implementation-pipeline.md`(`templates/base/` の写しを含む) | 変更なし | 「Org Runtime Commits」の「implement, verify, commit」は、implementer がスライスのコマンドを打つ形と合う。標準フローの implementer と `/implement` の話は #261 で済んでいて、この変更は触れない。`run-test.sh` で当たる `post-implementation-pipeline.md:95` は `/test` の行 |
| `README.md`(124、252、264 行目) | 変更なし | 124 行目と 264 行目は `ralph org gate` が 2 つのスクリプトを走らせる説明で、gate の動作は変わっていない。264 行目の「After the reviewer passes its review」はレビューの中身を言わない。252 行目は座席の構成だけ |
| `AGENTS.md`(90 行目)、`.ralph/core/AGENTS.core.md`、`templates/base/AGENTS.md` | 変更なし | `run-test.sh` で当たるのは `ralph org gate` の説明だけ。reviewer と implementer の検証には触れない。`.ralph/core/AGENTS.core.md` は当たる文がない |
| `docs/quality/quality-gates.md`(82 行目)、`templates/base/docs/quality/quality-gates.md`(81 行目)、`definition-of-done.md` | 変更なし | 82 行目と 81 行目は S5 が直した文で、最終の雛形(コマンドは leader の TASK が名指す、既定は両方、機能ごとの org は静的検査だけ、フルの test は gate で 1 回)と合う。2 つの差は KNOWN_DIFF の「(see docs/tech-debt)」だけ。`definition-of-done.md` の `run-test.sh` は `/test` の行 |
| `docs/recipes/codex-seat-permissions.md`(220〜223 行目)と `templates/base/` の写し | 変更なし | S5 が直した文(「the commands the TASK names, or both ... when it names none」)で、最終の雛形と合う。recipe の TASK の先頭の「sandbox probe, not a review」の段落は、reviewer の最初の動作が変わっていないので今も効く。2 つの写しは `check-sync.sh` で一致 |
| `docs/architecture/repo-map.md` | 変更なし | `run-test.sh` はスクリプトの一覧の中の名前だけ。skill、hook、script、動詞の追加と削除はない |
| `templates/base/ralph.toml:99` の「the reviewer re-runs run-static-verify.sh / run-test.sh, so a guarded claude reviewer stops at the permission prompt」 | 変更なし(正確) | 機能の org の reviewer も、レビューで `run-static-verify.sh` を、gate の TASK で `ralph org gate`(中で `run-test.sh`)を打つので、guarded の claude の reviewer が許可待ちで止まるという注意は今も正しい。この repo の root に `ralph.toml` はなく、同じ文は `templates/base/` にだけある。文は今も正しいので、scaffold 先に配る設定の注釈は触れていない |
| `/org` skill の 4 つの写し | 変更なし(一致) | `.claude/skills/org/SKILL.md` を触っていないので `sync-skills.sh` は打っていない。`check-skill-sync.sh` が 14 skill の一致を確かめ、`TestOrgSkill_ResultExampleFollowsTheReviewerTemplate` を含む `OrgSkill\|ReportNames\|SendDefaults\|DirectorSkill` のテストが通った |
| 他の skill、`internal/org/prompts/`、CLI の help、scripts | 変更なし | reviewer と `run-test.sh` が同じ行に出る文を、`.claude/skills/`、`.agents/skills/`、`.codex/`、`internal/org/prompts/`、`internal/cli/`、`scripts/` で探した(テストと `/org` skill を除く)。当たるのは `leader.md:244` の新しい文(`run-test.sh` も並んでいれば reviewer は既定を打っただけ)だけ |
| `docs/specs/2026-08-01-org-runtime.md` の「2026-10-27 改訂」(a)〜(d)と FR-7 の印 | 変更なし | 最終の雛形と矛盾しない。(b) の「レビューの TASK が指示したコマンドだけ」は、F3 で決めた「指示は TASK の本文の `CHECKS:` の行だけ」と同じ意味で、(b) は `CHECKS:` で返すことも書いている。入力の行の名前は director の仕様の新しい項目に書いた |
| `docs/tech-debt/README.md` のほかの行(224〜226、233 行目) | 変更なし(正確) | 224 と 225 行目は S5 が解決済みにした「tests twice」の行で、gate が静的検査をもう 1 回打つこと、指示のないレビューは両方を打つこと(D6)も書いてある。233 行目は「(and, outside a feature org, its `run-test.sh`)」に直っている。226 行目の `run-static-verify.sh` と `run-test.sh` は gate を含むスクリプト一般の話。360 行目の「ファイルが長い」の行の `prompts_test.go` 2670 行は、この PR が触れていないので今も正しい。新しいテストは 799 行と 90 行の別ファイルにあり、+300 行の trigger にも当たらない。`reviewer` と `run-test.sh` が同じ行に出る行はほかにない |
| tech-debt の計画参照 | OK | `docs/plans/active/` と `archive/` の参照をすべて辿り、存在しない参照は 0 件。`scripts/verify.local.sh` の `check_tech_debt_plan_refs` と同じ検査 |

## Decisions on the reports' remaining items

| 出どころ | 項目 | 扱い |
|----------|------|------|
| verify | 【LOW】director の仕様 FR-7 の「テストは review と gate の 2 回」が古い | 直した。history は書き換えず、日付つきの項目と印を足した(Changes made) |
| verify | 【既知】tech-debt の FR-7 の gate の行の reviewer の文 | 直した。文を直し、繰り越しの注を足した |
| verify | 【情報】`templates/base/ralph.toml:99` | 変更なし(上の表) |
| verify | 【情報】Progress に `[ ] Review artifact created` が重複して残る | 解消済み。verify の記録のコミット `065e69b4` が消していて、HEAD の Progress に同名の行は 1 つだけ(`[x]`) |
| verify、test | AC6 と S1 が名指す `internal/org/prompts_test.go` と、テストの実際の置き場(`prompts_verification_test.go`、`prompts_verification_multi_test.go`)の違い | 計画の AC の文は digest の範囲なので触れない。Progress の F2 と S1 の記録がすでに説明している。AC6 の `[x]` は、足したテスト 18 本と同期の検査が通ったことによる |
| test | Test gaps 1〜5(実機で動かしていない、雛形のテストは文の有無だけ、新旧の binary が混ざる org は立てていない、変異は名指しの分だけ、statement coverage なし) | tech-debt の行は足していない。コードのある負債ではなく、前の org の雛形の変更(2026-10-26 の reviewer-runs-gate など)と同じ種類の確認の穴である。実機の確認は FR-7 の `[ ]` の印(director の仕様)が持ち、次の実行で付け直す |
| self-review | LOW 2(複数スライスの RESULT の形、散文の指示を受け入れる文) | F3 `8cda482e` で直し、verify が HEAD で解消を確かめている。Tech debt の欄は空 |

## Plan drift notes

- 計画本文の「Scope」と「Affected areas」は、S1〜S5 のとおりに実行した。F1〜F3 で足した分(`CHECKS:` の行、送り直しの上限、複数スライスの形、`.harness/state/` のメッセージのファイル、テストの別ファイル化)は Progress に記録がある。Design decisions と Risks は、digest の範囲なので触れていない。
- Progress の「Remaining(sync-docs に渡す)」に書かれた tech-debt の繰り越しの注は、この run で足した。verify の LOW(director の仕様)は計画の Progress にはなく、verify の report の渡しとして処理した。
- 計画の Acceptance criteria は AC1〜AC6 がすべて `[x]`。残る `[ ]` は Progress の「PR created」だけで、`/pr` が付ける。

## `/pr` と release に渡すこと

- 実機の org での確かめはない。leader が `VERIFY:` と `CHECKS:` の行を書くこと、implementer がその行のコマンドと 5 項目の確かめを RESULT に書くこと、reviewer が `CHECKS:` の行のコマンドだけを打って返すこと、新旧の binary の座席が混ざったときの動きは、雛形の文のテストでだけ確かめた(test の Test gaps 1〜3)。FR-7 は `[ ]` のままで、次の実行の記録で付け直す。
- 差分は 20 ファイル、+1766/-100(`git diff 622cdcfb...HEAD --shortstat`、この run の前)。walkthrough を書くかどうかは `/pr` が決める。
- release ノートに次を手で足す。org の役割雛形は go:embed なので、新しい binary で spawn した座席から効き、古い binary で立てた org の座席は今の雛形のまま動く。機能の org の reviewer は、レビューで `run-test.sh` を打たない(leader が `CHECKS: ./scripts/run-static-verify.sh` を書く)。フルの test は PR の前の gate で 1 回走る。レビューの `GATE: pass` は静的検査が通ったことだけを指す。TASK に `CHECKS:` の行がないレビューでは、reviewer は今までどおり両方を打つ。implementer の既定のスライスの検証は、`run-verify.sh` ではなく、触ったところを覆ういちばん狭い検査になり、leader は TASK の本文の `VERIFY:` の行でスライスごとのコマンドを渡す。`ralph org gate` と `ralph org regate` は変わらない。
- コミットの型は `perf:` が 6 件、`docs:` が 12 件(この report のコミットを除く)。`.goreleaser.yml` の changelog は `^docs:`、`^test:`、`^chore:` を除くので、`perf:` の 6 件だけが自動の changelog に載る。`docs:` の 12 件(report、計画、文書、skill の同期)は載らないが、skill の 4 つの写しの変更(`docs: align the org skill ...`)は release ノートの上の文に含めた。
- tech-debt の行の Related 列は `docs/plans/active/` を指す。`/pr` の `archive-plan.sh` が `archive/` に書き換える。

## Verification

| Check | Result |
|-------|--------|
| `./scripts/check-sync.sh` | rc 0。IDENTICAL 172、DRIFTED 0、ROOT_ONLY 0、TEMPLATE_ONLY 11、KNOWN_DIFF 5。`PASS: all files in sync.` |
| `./scripts/check-skill-sync.sh` | rc 0。`[ok] check-skill-sync: 14 skill(s) in lock-step`(skill は触れていないので `sync-skills.sh` は打っていない) |
| `go test ./internal/org/ -run 'OrgSkill\|ReportNames\|SendDefaults\|DirectorSkill' -count=1` | rc 0。`ok  github.com/yoshpy-dev/ralph/internal/org  0.644s` |
| `./scripts/plan-visual.sh digest docs/plans/active/2026-10-27-org-narrow-verification.md` | `97abe345a188`(`- Approved:` 行の値と同じ。AC と Progress の編集の前後で同じ) |
| tech-debt の計画参照 | 存在しない参照 0 件 |
| tech-debt の編集した行 | `\|` 6 個(5 セル)、`~~` 0 個、U+FFFD なし |
| テスト | 上のほかは打っていない(`/test` の範囲。この run が変えたのは文書、計画の Progress と AC の印、report、insight event だけで、コードとスクリプトには触れていない) |
| insight event | `./scripts/insights-append.sh --slug org-narrow-verification --flow standard --phase sync_docs --verdict pass --cycle auto --source skill` を実行した。`docs/insights/events/2026-10-27-org-narrow-verification.jsonl` の既存のファイルへの追記(`ts` は 2026-10-27T18:12:40Z、cycle 1) |
