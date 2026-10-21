# Walkthrough: org-record-text(org の記録の文字: F12 と F14)

- Date: 2026-10-21
- Plan: docs/plans/active/2026-10-21-org-record-text.md(/pr で archive に移す)
- Branch: fix/org-record-text(base b55d3d73、この walkthrough の前の HEAD 7f124f80。17 commits、16 files、+1007/-27)。PR #246
- 読む範囲: Go の本体は 2 files(`internal/org/verbs.go` +13/-2、`internal/org/report.go` +33/-3)、テストは 4 files(+239/-17)、記録は 3 files(+8/-5)。残りは計画(279 行)、後工程の報告 5 本と triage、insight event 5 行
- 後工程は 1 回(cycle 1)で終え、cross-review の指摘は 0 件

```
何を直すか → 計画 → 切り方(S1) → 表の欄(S2) → テストの整理(S3) → 記録(S4) → self-review の直し → 後工程の報告 → AC の根拠 → 確かめていないこと
```

## 1. この PR が何か

- 段階 9 の試行(#243)で見つかった、記録の文字の所見 2 つを直す
  - F12: `sent` の Details の本文を 200 byte で切ると、多バイト文字が割れて U+FFFD になる。試行の manifest の 4 行と、2 つの org の報告に出ていた
  - F14: `ralph org report` の event timeline が Details の改行をそのまま表の行に書くので、1 件が何行にも割れる。空行を含む本文は GFM の表をそこで終わらせる
- あわせて、PR #244 の self-review の L4 の残り(`prompts_test.go` が別のテストファイルの定数に頼る)と L5(報告名の検査を別のテストに切り出す)を入れる。テストだけの変更で、振る舞いは変えない
- この PR は、試行の所見を直す系列の 2 本目(1 本目は仕様の印の #245)

## 2. 読む順(commit ごと)

1. ee3da6b0 計画
   - Scope、Non-goals、Design decisions、AC1〜AC7 と AC3b を読む。承認の digest は `200072f06b32`。後工程のあいだ本文は変えていない(Progress checklist と AC の checkbox だけが動く)
   - Design decisions: 改行は `<br>` にした。文字の `\n` は、本文が 1 続きの文に見え、`\` と `n` を持つ本文と区別できないので採らなかった。`<br>` は描画したときに本文の行が見え、tech-debt の行が示していた形でもある。どの選択も 1 つの関数とそのテストに閉じていて戻しやすいので、critical fork はなしとした
   - Codex の計画の助言(MEDIUM 1 件)を取り入れ、`\`、バッククォート、`<` も escape することにして、AC3 の事例と AC3b(GFM での 1 回の描画)を足した
2. 19fad46d S1 切り方(F12)
   - `truncateForDetails`(`internal/org/verbs.go:661`)。200 byte を超える入力で、`text[200]` が字の先頭でない間だけ、最大 `utf8.UTFMax - 1` 回戻る。有効な UTF-8 なら 3 回以内に字の先頭に着く
   - `TestTruncateForDetails_CutsAtACharacterBoundary`(`verbs_test.go:1284`、11 事例)と、manifest から読み戻す `TestOrgSend_JapaneseBodyOver200Bytes_DetailsStayValidUTF8`(`verbs_test.go:1350`)
   - 見ること: 不正な列で戻る回数を止める理由(200 から大きく戻らないため)
3. a88e7382 S2 表の欄(F14)
   - `reportCellReplacer` と `escapeReportCell`(`internal/org/report.go:116-146`)。7 組の 1 回の置換で、timeline の行(217 行)だけが通る。roster と receipts の行は変わらない
   - `"\r\n"` を `"\r"` と `"\n"` の前に置く。`strings.Replacer` は同じ位置で引数の順に組を試すので、CRLF が `<br>` 1 つになる
   - `reportCellCases`(`report_test.go:123`、11 事例)を 2 つのテストが使う。`TestBuildOrgReport_DetailsCellKeepsEachEventOnOneRow`(172 行)は行全体を比べ、`TestBuildOrgReport_TimelineHasOneLinePerEventOfFourCells`(191 行)は 11 件で 13 行、各行に escape されていない縦棒が 5 つあることを確かめる
   - 見ること: escape しない文字(`*`、`_`、`[`、`]`、`~`、`&`、`>`)を doc comment が理由つきで挙げているか
4. 3c688ff3 S3 テストの整理
   - `reviewerReportFilePrefix` と `reviewerReportPrefix` を `prompts_test.go:1030-1035` に置いた。`report_names_sync_test.go` は定義をやめ、README の検査で `reviewerReportFilePrefix` を使う
   - 報告名の検査 6 つを `TestRenderRolePrompt_Reviewer_ReportContractNamesTheReport`(1042 行)に移した。verify が、移した範囲が base と byte で同じで、`t.Errorf`/`t.Fatalf` が 17 = 11 + 6 で数が合うことを確かめた
5. f7f2644b S4 記録
   - tech-debt の F12 と F14 の行を RESOLVED にした。F8 の行と同じく、Trigger の欄も打ち消した
   - 根拠の表 `docs/evidence/org-spec-evidence-2026-10-21.md` の F12・F14 の行と、仕様の 141 行の注
   - 計画の active の path を根拠の欄に書いた。計画の参照の検査は、計画が active にあるあいだ archive の path では通らない。`/pr` の `archive-plan.sh` が書き換える
6. 3d5d9a61 計画の Progress、1d3ce15d self-review(1 回目)
7. cab879de self-review の LOW の直し
   - L1: `reportCellReplacer` の doc comment に、escape しない文字とその理由を 1 文足した
   - L2 の前半: 根拠の表の F12・F14 の最後の欄を、F8 の行と同じ `"~~...~~"(RESOLVED の行)` の形にした
   - L3: テストの型 `reportCellCase` を 1 つ置き、匿名の struct を 2 回書くのをやめた
   - I2: 新しい prompts のテストで重複していた先頭のコメントを消した
8. ca1fb310 再レビューの報告、8a52987f I4(doc comment の「hide text」を、renderer が見せるものの言い方に直した)、55338bbc その記録
9. 後工程の記録
   - 6622bce0: verify
   - f8172150: test
   - 696ff5d4: AC の checkbox と Progress(verify の V1)
   - 410b8b7e: sync-docs。F14 の RESOLVED のコメントの「each case」を「among them」に直した(verify の V2)。test の G1 を tech-debt の新しい行にした
   - acebaf3a: cross-review の triage
   - 7f124f80: cross-review の insight event

## 3. AC の根拠

| AC | 根拠 |
|----|------|
| AC1 | `TestTruncateForDetails_CutsAtACharacterBoundary` の 11 事例が計画の事例をすべて含む(199 ASCII のあとの 3 byte の字、200 byte 目で終わる字、197〜200 から始まる 4 byte の字、200 を超える ASCII、ちょうど 200 byte、不正な列)。両方の toolchain で PASS。base では 11 のうち 5 が落ちる |
| AC2 | `TestOrgSend_JapaneseBodyOver200Bytes_DetailsStayValidUTF8` は manifest から読み戻し、`utf8.ValidString`、U+FFFD を含まないこと、`...(truncated)` で終わること、残した文が 200 byte 以下の前方一致であることを確かめる。base では U+FFFD が 1 つ入り、残した文が 202 byte で落ちる |
| AC3 | 11 事例の行全体の一致と、timeline が 13 行・各行 4 欄。base では 11 のうち 9 が落ち、timeline は 24 行になる |
| AC3b | verify が GitHub の GFM renderer(`gh api markdown`、読むだけ)で 1 回描画した。tbody の `<tr>` 11、`<td>` 44、`<br>` 13、`<code>` 0 で、`\`、バッククォート、`<path>`、`\|` が書いたとおりに出る |
| AC4 | `report_test.go` と `verbs_test.go` の diff に消した行はない。`TestBuildOrgReport_EscapesPipesInEventDetails` は新しい置換でも同じ期待で通る |
| AC5 | 定数が `prompts_test.go` にあり、sync のテストは使う側。`go test -run 'TestRenderRolePrompt_Reviewer\|TestReportNames'` は両方の toolchain で PASS |
| AC6 | tech-debt の 2 行は F8 の行と同じ形で、閉じるテスト 5 つを名指し、どれも `func <名前>(` で 1 件ずつ見つかる。仕様の印は `[x]` 20、`[ ]` 12 で base と同じ |
| AC7 | `run-static-verify.sh` rc 0(verify)、`run-test.sh` rc 0(test、1,086 秒)、`GOTOOLCHAIN=go1.25.8 go test -count=1 -timeout 30m ./...` rc 0(test、880 秒) |

## 4. 後工程の報告

- self-review(`docs/reports/self-review-2026-10-21-org-record-text.md`): 再レビューの版で Merge: yes。L1・L3・I2・L2 の前半を cab879de、I4 を 8a52987f で直した。L2 の後半は PR の番号ができてから次の PR で入れる
- verify(`docs/reports/verify-2026-10-21-org-record-text.md`): pass。INFO の V1(AC の checkbox と Progress)は 696ff5d4、V2(RESOLVED のコメントの言い方)は 410b8b7e で直した
- test(`docs/reports/test-2026-10-21-org-record-text.md`): pass
  - mutation は 26 本のうち 25 本が赤。残った V10 は、dry-run の記録で本文を切らない変異で、`internal/org` と `internal/cli` の全体でも生き残った
  - これが G1(LOW)。dry-run も `sent` と同じ関数で切るので、HEAD の振る舞いは正しい
  - 記録にある load に弱いテストは、org と cli の全体を各 3 回回して、どれも落ちなかった
- sync-docs(`docs/reports/sync-docs-2026-10-21-org-record-text.md`): timeline の欄と 200 byte の切り方を書く生きた文書は、記録の 3 か所のほかになかった。G1 を tech-debt の行にした
- cross-review(`docs/reports/cross-review-triage-org-record-text.md`): codex、rc 0、指摘 0 件。codex は、報告を節で切り出すほかのテスト(merge_run、gate、approval)も読み、既存の呼び出し側と合うとした

## 5. 確かめていないこと

- CI
- dry-run の記録で本文を切ることを確かめるテスト(test の G1)。根拠の表の G1(check-sync の suite)とは別もの。tech-debt の行にし、系列 3 本目のテストの安定で 1 本足して閉じる
- PR の番号を、根拠の表の F12・F14 の「直す PR」、仕様と根拠の文書の PR の一覧に足すこと(self-review の L2 の後半)。次の系列 PR で入れる
- 不正な UTF-8 の入力で、切った結果が有効な UTF-8 になること。計画は切り目が 200 から `utf8.UTFMax - 1` byte 以内であることまでにした
- AC3b の描画をテストにすること。verify が 1 回確かめただけ
- Linux での実行
