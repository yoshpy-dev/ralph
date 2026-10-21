# Walkthrough: org-spec-complete(機能ごとの org と director の仕様の完了表示)

- Date: 2026-10-21
- Plan: docs/plans/active/2026-10-21-org-spec-complete.md(/pr で archive に移す)
- Branch: docs/org-spec-complete(base 765bcc46、この walkthrough の前の HEAD 3fa5d6b5。17 commits、10 files、+673/-34)。PR #245
- 読む範囲: 仕様の 1 file が +76/-33、根拠の表(新規)が +109、tech-debt が +2/-1。残りは計画(134 行)、後工程の報告 5 本、insight event 5 行
- 変更は docs だけで、コード、script、skill は変えていない。後工程は 1 回(cycle 1)で終え、cross-review の指摘は 0 件

```
何を直すか → 計画 → 根拠の表(S1) → 仕様の印と Status(S2) → self-review の直し → 再レビューの直し → 後工程の報告 → AC の根拠 → 確かめていないこと
```

## 1. この PR が何か

- 仕様 `docs/specs/2026-10-07-org-multi-org-director.md` は 9 段がすべて merge したあとも、Rollout に「1. 共通の台帳(FR-1、進行中)」が残り、FR などの checkbox に印がなかった。この PR は、その完了表示を今の状態に合わせる
- 印の規則: 実装があり、テストで確かめていて、外のツールに関わる項目は実機の記録があり、試行の所見と食い違わない項目だけを `[x]` にする。食い違う項目は `[ ]` のまま、所見と tech-debt の行を「印(2026-10-21)」の注で指す。読み替えた項目と、いくつかの段を合わせて満たした項目は、`[x]` に注を付ける
- Status の節は、段が終わったことと項目の印を分けて書く。試行の所見を直すあとの系列の PR が、直した所見に結びつく印、Status の数と所見の一覧、根拠の表を同じ PR で更新すると決めた
- この PR は、試行の所見を直す系列の 1 本目。系列の順は consult(plan-trialfixes)で決め、根拠の表の「試行の所見」の節の「直す PR」の列に書いた

## 2. 読む順(commit ごと)

1. 85ec1339 計画
   - Scope、Non-goals、Design decisions、AC1〜AC8 と AC6b を読む。承認の digest は `98781cfa6662`。後工程のあいだ本文は変えていない(Progress checklist と AC の checkbox だけが動く)
   - 見ること: AC7(印・注・Status・Rollout の PR 番号の外の行は変えない)が、後の diff でも守られているか
2. 1a5bc2ee S1 根拠の表 `docs/evidence/org-spec-evidence-2026-10-21.md`(新規)
   - 「試行の所見」の表が先にある。各所見について、関係する項目、食い違うかどうか、直す PR を書く
   - 次に 31 項目の表。各行に、テストのパスと関数名、証拠のファイル、仕様の注のどれか 1 つ以上と、判定がある
   - 末尾に段ごとの PR と、確かめていないこと
   - この slice は implementer が run-verify のあとコミットせずに止まったので、表の行数、テスト名、tech-debt の引用を確かめてから手でコミットした(計画の Progress に記録)
   - 見ること: 「食い違う」を「はい」とした所見と、その所見に結びつく項目の判定
3. bc9fe9e0 S2 仕様
   - 先頭の `## Status` の節、Rollout の各段の PR 番号、31 項目の印と「印(2026-10-21)」の注
   - 「checkbox は付けない」と書いていた文を、今の印に合う文に直した
   - FR-10 の 8b-1 の注にあった「121 行」「120 行」の参照は、受け入れ条件の文言で指す形にした
   - 見ること: `[ ]` の項目の注が、所見と tech-debt の行を名指しているか。Status の「12 項目」と印の数が合うか
4. 264814a4 計画の Progress
5. 7bf4bbb9 self-review(1 回目)
   - M1: FR-3 の扱いの不一致。FR-3 は `[x]` だったが、試行で見張りの leader への ALERT が 10 回時間切れで送れなかったことと食い違う
6. 9733eb31 self-review の直し
   - FR-3 を `[ ]` にした
   - 根拠の表に「leader への ALERT」の所見の行を足した
   - tech-debt に、ALERT が送れなかった行を足した
7. c56b70a4 計画の逸脱の記録(FR-3 を `[ ]` にしたこと、新しい行)、370d30ff 再レビューの報告
8. 43bf5c6e 再レビューの N1・N4・N5 の直し
   - N1: Status の数と所見の一覧に「2026-10-21 の時点で」を付け、更新の担当を系列の PR に決めた
   - N4: ALERT の原因の候補から F7 を外した。ALERT は agmsg の送信を直に呼び、F7 で待ち切れた `ralph org send` の待ち(座席の手が空くのを待つ)を通らない。そのため候補は F3(agmsg 1.5.1 に関わる座席の環境)に絞った
   - N5: 根拠の表の F4 の行の「関係する項目」に、受け入れ条件 3 を足した(仕様の受け入れ条件 3 の注と表の項目の行は F4 を指していたが、所見の行から受け入れ条件 3 へたどれなかった)
   - N2(計画の本文の古い言い回し)は、承認の digest を保つために意図して残した
9. 後工程の記録
   - 82ba75b5: 計画の Progress
   - 641036ad: verify
   - f16e90f8: AC の checkbox と self-review の insight event
   - 84ef6f59: test
   - 85256177: verify の報告のテスト名の数を 67 から 66 に直した
   - 9015064a: sync-docs。tech-debt の doctor のテストの行に、この PR の test で負荷のせいで落ちた 3 本目のテストを足した。計画の Affected areas の外の編集になる
   - a773f0b7: cross-review の triage
   - 3fa5d6b5: cross-review の insight event

## 3. AC の根拠

| AC | 根拠 |
|----|------|
| AC1 | 仕様 3〜24 行の Status。PR 28 本を段ごとに並べ、verify と test がそれぞれ `gh pr view` で MERGED を確かめた(読むだけ)。12 項目が `[ ]` で残ることと、その理由の置き場所(根拠の表と各項目の注)、release は人が切ることを書く |
| AC2 | `grep -c 進行中` は HEAD で 0(base は 1)。Rollout の 9 行すべてに PR 番号があり、Status の番号と同じ |
| AC3 | verify が 31 項目の印を根拠の表の判定と 1 つずつ比べ、31 件一致した。`[x]` は 20(表の 19 と、前から `[x]` の FR-12)、`[ ]` は 12 |
| AC4 | `checkbox` を含む行は HEAD で 2 行(base は 6 行)。残る 2 行は、FR-7 の承認の digest が計画の checkbox を飛ばす話と、「FR-12 の checkbox は付けた」で、どちらも今の印と合う |
| AC5 | 「121 行」「120 行」は HEAD で 0 件(base は各 2 件)。FR-10 の 8b-1 の注は、受け入れ条件を文言で指す |
| AC6 | 根拠の表の「項目」は 31 行。表のテスト名 66 個(103 か所)は、どれも `func <名前>(` で 1 件ずつ見つかり、組になるパスにある(verify と test)。test は 66 本と subtest 430 本を `-v` で回し、すべて PASS、SKIP 0 |
| AC6b | 所見の表は 18 行(F1〜F14、D-F3、folder trust、G1、「leader への ALERT」)。「食い違う」が「はい」の 7 行は Status の所見の一覧と同じで、結びつく項目は仕様で `[ ]` のまま、所見と tech-debt の行を指す |
| AC7 | verify が、Status の節と注を除き、印を戻してから base と比べた。残る差は checkbox の文の直しと PR 番号だけで、要件と設計の文は変わっていない |
| AC8 | `run-static-verify.sh` rc 0(verify)。`run-test.sh` rc 0(shell 41 本。差分が docs だけなので Go は走らない)。Go は別に go1.25.8 で `go test ./...` が 8 パッケージとも ok(test) |

## 4. 後工程の報告

- self-review(`docs/reports/self-review-2026-10-21-org-spec-complete.md`): 再レビューの版で Merge: yes。1 回目の M1 と L1〜L5、再レビューの N1・N4・N5 を直した。N2 は意図して残した
- verify(`docs/reports/verify-2026-10-21-org-spec-complete.md`): pass。INFO の V1(FR-4 と受け入れ条件 5 を F4 のせいで `[ ]` にしたことが計画に書かれていない)は計画の Progress に記録した
- test(`docs/reports/test-2026-10-21-org-spec-complete.md`): pass
  - go1.26.0 の `go test ./...` で `TestRunDoctorOpts_CodexSandboxCheck_HermeticByDefault` が 1 回落ちた。load average 100〜220 の下で、`probeBinary` の 5 秒を stub が超えた
  - 単独では 5 回とも通った。この PR は Go に触れていない
- sync-docs(`docs/reports/sync-docs-2026-10-21-org-spec-complete.md`): この diff が起こす drift はなかった。tech-debt の doctor のテストの行を広げた
- cross-review(`docs/reports/cross-review-triage-org-spec-complete.md`): codex、rc 0、指摘 0 件。codex は自分で、印の一致、テスト名、JSONL、変更が docs だけであることを確かめた

## 5. 確かめていないこと

- CI
- codex の director の実機(受け入れ条件 15)。仕様の注で残している
- 見張りの leader への ALERT が送れなかった原因。tech-debt の行と根拠の表に、座席の環境の系列 PR で、先に agmsg の送信の時間を測ると書いた
- tech-debt の doctor のテストの行の書き出しは「Two doctor tests」のまま。計画の Progress と test の報告がこの書き出しで行を指すので、テストの安定の系列 PR で行を書き直すときに直す
