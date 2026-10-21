# sync-docs report: org-record-text

- Date: 2026-10-21
- Plan: `docs/plans/active/2026-10-21-org-record-text.md`(承認の digest `200072f06b32` はこの run の前後で同じ。`./scripts/plan-visual.sh digest` で確かめた)
- Pipeline cycle: 1(`.harness/state/standard-pipeline/cycle-count.json` の値)。base は `b55d3d73`、branch は `fix/org-record-text`。run 開始時の HEAD は `696ff5d4`(run 開始時の値で、この report を含む commit の SHA ではない)
- 先行 report: `docs/reports/self-review-2026-10-21-org-record-text.md`(Merge yes)、`docs/reports/verify-2026-10-21-org-record-text.md`(pass。INFO V1、V2)、`docs/reports/test-2026-10-21-org-record-text.md`(pass。G1 LOW、G2〜G4 INFO)
- この run が変えたのは、tech-debt の既存の 1 行の 1 語と新しい 1 行、この report、insight event。コード、script、skill、rule、仕様、根拠の表、先行 3 report、計画には触っていない

## Summary

`b55d3d73...696ff5d4` の差分が変えたのは、`truncateForDetails`(`sent` と dry-run の Details を文字の切れ目で切る)と、`ralph org report` の event timeline の details 欄(`reportCellReplacer` の 1 回の置換で escape する)と、それを書く記録(tech-debt の 2 行、根拠の表の 2 行、仕様の 141 行の注)である。この変更を書く文書がほかにあるかを、仕様、README、AGENTS.md、`docs/recipes/`、`docs/reports/README.md`、`.claude/rules/ralph/`、`/org` skill の 4 つの写し、`internal/org/prompts/`、Go のコメントとコマンドの help から探した。timeline が Details をどう書くかを述べる文書と、`sent` の Details の切り方(200 byte)を述べる文書は、記録の 3 か所(tech-debt、根拠の表、仕様の注)のほかになかった。

直した点は 2 つある。verify の V2(INFO)が挙げた、F14 の RESOLVED コメントが事例の一部を全部のように書いていた所。test report の G1(LOW)が見つけた dry-run 側の切り方のテストの穴を、tech-debt に 1 行として記録したこと。

## Changes made

| File | Change |
|------|--------|
| `docs/tech-debt/README.md`(325 行、F14 の RESOLVED コメント) | 「compares the whole row of each case (header lines and ...」を「(among them header lines and ...」にした。テストの事例は 11 あり、括弧の中の 7 つは一部だと分かる。verify の V2 の提案どおり。HTML コメントなので描画は変わらない |
| `docs/tech-debt/README.md`(330 行、新しい行) | test report の G1(LOW)を記録した。`Send` の dry-run の `sent`(`p.DryRun` の枝)が本文を切ることを確かめるテストがない、という穴。欄は 5 つで、329 行までの行と同じ。引く名前は関数名とテスト名(`Send`、`truncateForDetails`、`TestTruncateForDetails_CutsAtACharacterBoundary`、`TestOrgSend_JapaneseBodyOver200Bytes_DetailsStayValidUTF8`)で、行番号は使っていない。計画は `docs/plans/active/` の形で書いた(`/pr` の `archive-plan.sh` が書き換える。`scripts/verify.local.sh` の `check_tech_debt_plan_refs` が読む形)。trigger は「dry-run の枝か `truncateForDetails` を次に変えるとき」で、足すテストは G1 が示した 1 本(同じ日本語の本文を `DryRun: true` で送る) |
| `docs/reports/sync-docs-2026-10-21-org-record-text.md` | 新規 |
| `docs/insights/events/2026-10-21-org-record-text.jsonl` | sync_docs の event を 1 行足した(下の「Insight event」) |

## Surfaces checked for drift

| Surface | 結果 | 根拠 |
|---------|------|------|
| `docs/specs/2026-08-01-org-runtime.md`(FR-9 の 73 行、159 行) | 変更なし | 「完了時に編成履歴を `docs/reports/` へ成果物化」と「正本は manifest / git / reports」を言うだけで、report の欄の書き方も Details の切り方も書かない |
| `docs/reports/README.md` と `templates/base/docs/reports/README.md` | 変更なし | 名前の一覧だけ。`cmp` で同一 |
| `/org` skill(`.claude/skills/org/SKILL.md` と 3 つの写し) | 変更なし | 180 行の `report` の行は出力先の名前だけ。817 行の「承認の記録(`USER`、`HERDR_PANE_ID` の有無、cwd)を event timeline で追える」は、cwd に `\`、バッククォート、`<`、縦棒があっても escape されたまま読めるので今も正しい。919 行の `gate_failed` の Details の先頭の `reason=` は、置換の対象の文字を含まない。写しは `check-skill-sync.sh` で lock-step |
| `internal/org/prompts/leader.md`(39、192、391 行) | 変更なし | `ralph org report` を打つ手順と置き場所で、report の中身の書き方は書かない |
| `internal/cli/org.go` の `newOrgReportCmd`(1922〜1933 行)の Short と Long | 変更なし | 「roster、event timeline、model receipts」と並べるだけで、Details の escape は述べない。コードは今回の範囲 |
| `internal/org/report.go` の `BuildOrgReport` の doc(150〜160 行) | 変更なし | 節の並びと「空の org でも report を書く」だけを述べる。新しい `reportCellReplacer` と `escapeReportCell` の doc(116〜145 行)は、置換の 7 組(`\`、`\|`、バッククォート、`<`、`\r\n`、`\r`、`\n`)と、置かない 7 文字の理由が実装と合う |
| `internal/org/verbs.go` の `truncateForDetails` の doc(653〜660 行) | 合っている | 「最大 `sendDetailsMaxLen` byte、文字の途中なら戻る、戻りは最大 `utf8.UTFMax-1` byte」は 661〜670 行の実装と同じ。不正な UTF-8 で 197 byte で止まる所も合う |
| `README.md`(242 行の例、266 行)、`AGENTS.md`(90 行) | 変更なし | 242 行は動詞の並び、90 行は「org report generation」の一語。266 行の Details は `merge-order` と `regate` の話で別 |
| `.claude/rules/ralph/agent-messaging.md`(93 行) | 変更なし | 「`sent` の manifest event は `raw=true` を記録する」は、Details の先頭の印のままで、切るのは印の後ろの本文だけ(`TestOrgSend_Raw_BypassesValidation_RecordsRawInDetails` と `TestOrgSend_RawAndUnconfirmed_DetailsPrefixOrder` が通る。test report の 3、4)。本文の 2,000 文字の上限は `Validate` のもので、Details の 200 byte とは別 |
| `docs/recipes/codex-seat-permissions.md` と写し(164、329 行) | 変更なし | 164 行は `spawned` の Details の `permission_mode=`、329 行は別の話の「Details」。どちらも timeline の書き方と `sent` の切り方に依らない |
| 「200 byte」「200 バイト」「truncat」「event timeline」「org-manifest-」「BuildOrgReport」を書く文書 | 記録の 3 か所のほかになし | `git grep` の結果は、tech-debt の 2 行、根拠の表、仕様の注、この計画、日付つきの過去の report と archive、無関係な 3 つ(`/audit-harness` の「silently truncated」、`check-template-purity.sh` の regex の話、plan の図の template の `<b>Details</b>`)。skill と rule と recipe に、この変更が古くする文はない |
| tech-debt の 322〜327 行(F12 の RESOLVED コメントと行、F13 の行、F14 の RESOLVED コメントと行) | 合っている | コメントは `RESOLVED 2026-10-21 in fix/org-record-text (S1 19fad46d)` と `(S2 a88e7382)` の形で、F8 の 319〜320 行と同じ。行は欄が 5 つで、第 1 欄と第 4 欄を `~~` で打ち消し、第 5 欄に `docs/plans/active/2026-10-21-org-record-text.md` を足している。コメントの内容は実装と合う(切りは最大 `utf8.UTFMax - 1` byte 戻る、置換は 7 組で `\r\n` と単独の `\r` は 1 つずつ `<br>`、置かない 7 文字は計画の Open questions)。2 つのコメントが挙げる 5 つのテスト(F12 は 2 つ、F14 は 3 つ)は、`func <名前>(` で 1 件ずつある |
| tech-debt の 213、214、217 行(`ralph org report` の event timeline を言う行) | 変更なし | 213 と 214 は「timeline に承認の記録が残る」で、今も正しい。217 行の (a) は PR #241 の RESOLVED の履歴(「`\|` を escape する」)で、今は `\|` に加えて `\`、バッククォート、`<` を escape し、改行を `<br>` にするが、書かれた時点の記録として正しい |
| 根拠の表 `docs/evidence/org-spec-evidence-2026-10-21.md` の F12(31 行)と F14(33 行) | 合っている | 縦棒は 7 本(6 欄)で F8 の行と同じ形。引用は `~~` で打ち消し、第 1 欄に含む行を 1 行に決められる(F12 は 323 行だけ。F14 の引用「`ralph org report`'s event timeline」を第 1 欄に含む行は 214、217、326 行の 3 行だが、先頭が一致するのは 326 行の 1 行で、表が使う「先頭の数語で指す」規則なら決まる)。引用 18 個のどれも、新しい 330 行の第 1 欄には当たらない |
| 根拠の表の「直す PR」欄と 16 行の「番号はまだない」 | 変更なし(意図) | F12 と F14 の PR の番号は、系列の次の PR が足す(#245 が F8 でそうした)。仕様の Status の PR 一覧(「試行の所見 F8 の直し: #244」の次)と根拠の表の PR 一覧も同じ。計画の Progress の S4 の記録に書いてある |
| 根拠の表の 60 行(NFR 記録の置き場所) | 変更なし | 「F12 と F14 は記録の見た目の所見で、食い違わない」は、直したあとも成り立つ。直したことは F12 と F14 の行が持つ |
| 仕様 `docs/specs/2026-10-07-org-multi-org-director.md` の 141 行 | 合っている | 追加の 1 文「`sent` の Details は字の切れ目で切り、`ralph org report` の event timeline は改行を `<br>` にして 1 件を 1 行に収める」は実装と同じ。印の数は `[x]` 20、`[ ]` 12 で base と同じ。Status の 20 行目あたりの食い違う所見の一覧(F3、F4、F7、F10、F13、D-F3、「leader への ALERT」)に F12 と F14 は入っていない |
| 計画 | 合っている | digest は `200072f06b32` で承認の行と一致。AC1〜AC7 と AC3b、Progress checklist は `696ff5d4` で印が付き、未了は「PR created」だけ。verify の V1(checkbox の遅れ)は解消済み。この run は計画に触っていない |

## Plan drift notes

- 計画の Affected areas に、新しい tech-debt の行(330 行)はない。S4 の記録は F12 と F14 の 2 行だけを予定していた。G1 の行は test report が見つけた穴の記録で、計画の AC は変わらない
- Progress の S4 の記録(F12 と F14 の行が trigger の欄も打ち消す、Related 欄に計画の active の path を足す、Go のコメントは「pipe」を語で書く、仕様 141 行は 。の後ろに 1 文足す)は、差分と合う

## Left open

- G1(LOW)の穴のテスト: tech-debt の 330 行に記録した。この PR ではテストを足さない(テストの追加は計画の AC にない)。依頼の範囲(この差分が起こす drift)より広い編集なので、lead が不要と見るなら 330 行の 1 行を消すだけで戻せる。ほかの変更とは独立
- G2(INFO)の `utf8.ValidString` の検査: `Send` のテストでは manifest の JSON の encode が不正な byte を先に U+FFFD にするので発火しない。F12 の RESOLVED コメントは、テストが見る項目(有効な UTF-8、U+FFFD を含まない、`...(truncated)` で終わる)を並べるだけで、「どの検査が効くか」は書かない。直さない。test report が記録している
- 試行のスナップショット `docs/evidence/org-trial-2026-10-20/`(manifest と 2 つの org の report): U+FFFD と崩れた表が残る。計画の Non-goals で書き換えない。tech-debt の RESOLVED コメントがその旨を書く
- F12 と F14 の PR の番号: 系列の次の PR が、根拠の表の 2 行、仕様の PR 一覧、根拠の表の PR 一覧に足す(上の表のとおり意図)
- release のノート: ralph は CHANGELOG を持たない。新しい binary が書く report の表の見た目と `sent` の Details の切り方が変わるが、計画の Rollout が「次の release に入り、移行はなく、既存の manifest と report は書き換えない」と書いている。release はユーザーが切る

## Verification

| 確かめたこと | 結果 |
|--------------|------|
| `./scripts/plan-visual.sh digest docs/plans/active/2026-10-21-org-record-text.md` | `200072f06b32`(承認の digest と同じ) |
| `./scripts/check-sync.sh`(編集のあと) | rc 0。`PASS: all files in sync.`(IDENTICAL 171、DRIFTED 0、ROOT_ONLY 0) |
| `./scripts/check-skill-sync.sh` | rc 0。`14 skill(s) in lock-step`。skill は編集していない |
| `git diff --check` | rc 0。diff に EF BF BD の byte 列は 0 件 |
| tech-debt の計画の path(`check_tech_debt_plan_refs` と同じ grep) | 欠けた path は 0 件 |
| tech-debt の 323、326、330 行 | 欄はどれも 5 つ。322、325 行のコメントは `<!-- RESOLVED` で始まる。引いた test 名 2 つは `internal/org/verbs_test.go` に 1 件ずつ、引いた report と計画は存在する |
| 根拠の表の引用 18 個と新しい 330 行 | 330 行の第 1 欄に含まれる引用は 0 件 |
| 上の「Surfaces checked」の `git grep`、`sed`、行の数え方 | 結果は表のとおり。scratchpad の python で打ち、仕様、根拠の表、tech-debt の既存の行は読み取りだけ |

`go test`、`run-static-verify.sh`、`run-verify.sh` は走らせていない。この run が変えたのは markdown 2 file(tech-debt の 1 file とこの report)と insight event で、コードも script も skill も変えていない。`scripts/verify.local.sh` の既定の実行は全 test を走らせるので回さず、計画の path の存在は同じ grep で確かめた。

## Verdict

- Verdict: pass(sync-docs に stop condition はない)
- Not verified: Linux での実行(test report の G4 と同じ)。GFM の描画(verify の AC3b が 1 回確かめた範囲のみ)。日付つきの履歴の記録の中身

## Insight event

`./scripts/insights-append.sh --slug org-record-text --flow standard --phase sync_docs --verdict pass --source skill --cycle 1` で `docs/insights/events/2026-10-21-org-record-text.jsonl` に 1 行足した。cycle は `cycle-count.json` の値と同じ 1 を明示した。
