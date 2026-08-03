# Cross-review triage — org-debt-batch

- Date: 2026-08-03
- Driver: claude  Reviewer: codex
- Base: main (merge-base fde0e84), HEAD: 5147f2a
- Cycle: 1/2
- After triage: ACTION_REQUIRED 1 / WORTH_CONSIDERING 0 / DISMISSED 0

## ACTION_REQUIRED

### AR-1 (Codex P3) tech-debt RESOLVED コメントの doctor 過大記載

- 対象: `docs/tech-debt/README.md:79`(watchdog 4 LOW row の RESOLVED コメント)
- 指摘: コメントが「`doctor` prints the org state check line when applicable and otherwise notes it as not applicable」と記載しているが、本バッチは `internal/cli/doctor.go` を変更していない(doctor に state-dir/ResolveOrgStateDir 出力は存在しない)。トラッカーとしてこの row を読む将来の編集者に、doctor 側も実装済みと誤認させる。
- トリアージ: 真正(row の記載と実装の不一致)。Slice 4 の判断自体は正当 — doctor には org state-dir 行が存在せず、handoff の指示どおり「対象外」と報告された。誤っているのは RESOLVED コメントの文言のみ。修正は 1 行の文言修正で安価。severity は低い(P3、doc-only)が、tech-debt row は「正直な台帳」という本リポジトリの規約上、過大記載は放置しない。
- 修正方針: RESOLVED コメントの doctor 言及を「doctor は org state-dir 診断行を持たないため対象外(変更なし)」へ書き換える。

## WORTH_CONSIDERING

なし。

## DISMISSED

なし。

## 判断

Cycle 1/2(cap 未到達)。ユーザーの常任指示に基づき **Fix を選択**。修正後、フルパイプライン(/self-review → /verify → /test → /sync-docs → /cross-review)を cycle 2 として再実行する(修正は doc 1 行のため、各フェーズは差分フォーカスで実行)。

---

# Cycle 2(再レビュー)

- Date: 2026-08-03
- Driver: claude  Reviewer: codex
- HEAD: dd9e7e2(AR-1 fix 53145ea + cycle-2 self-review doc fixes を含む)
- Cycle: 2/2(cap 到達)
- After triage: ACTION_REQUIRED 0 / WORTH_CONSIDERING 1 / DISMISSED 0

## WORTH_CONSIDERING(cap 到達につき Known gap として記録)

### WC-1 (Codex P2) 成功遮断後の resume では total-budget 条件が完全無音

- 対象: `internal/org/watch.go` `evaluateTotalBudget`(Cutoff early-return)
- 指摘: 先行遮断が**成功**していた場合(`Cutoff=true`)、`if rec != nil && rec.Cutoff { return }` が ALERT 経路より先に効くため、resume 後の over-budget 新座席には ALERT も Stop も発生しない。Slice 4 の re-alert fix が効くのは先行遮断が失敗していた場合(`Cutoff=false`)のみ。
- トリアージ: 事実として正確(実コードで確認)。ただし Cutoff ラチェットのセマンティクス変更は本プランが Design decisions / 逸脱ノートで明示的にスコープ外とし、専用の設計判断を要する事項として tech-debt 登録済み(Slice 5 の新規 row)。cap 到達(2/2)でもあるため、**修正はせず**: (1) tech-debt row の事実誤認(「re-alert は継続する」)を本コミットで訂正、(2) PR body の Known gaps に記録。Real issue = Yes / Worth fixing now = No(設計パス要 + cap)→ WORTH_CONSIDERING。
- フォローアップ: tech-debt row のトリガー(evaluateTotalBudget を次に触るとき、または実運用で resume 後の再遮断が必要になったとき)で Cutoff の clear or resume-epoch スコープ化を設計する。
