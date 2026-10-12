# Cross-review triage report: org-plan-approval

- Date: 2026-10-13
- Plan: docs/plans/active/2026-10-12-org-plan-approval.md
- Base branch: main(merge-base 225dff27)
- Driver: claude
- Reviewer: codex
- Triager: Claude Code (main context)
- Reviewer status: complete
- Self-review cross-ref: yes
- Cycle: 1/2
- Total reviewer findings: 2
- After triage: ACTION_REQUIRED=2, WORTH_CONSIDERING=0, DISMISSED=0

## Triage context

- Active plan: docs/plans/active/2026-10-12-org-plan-approval.md(承認済み、digest 951cf09bd7bc。変わっていない)
- Self-review report: docs/reports/self-review-2026-10-13-org-plan-approval.md(cycle 1 の 3 回目は 48d7282f で Merge: yes、LOW 3 件の N6〜N8 は 78b3e873 で tech-debt の 218〜220 行にした。N4 は 221 行)
- Verify report: docs/reports/verify-2026-10-13-org-plan-approval.md(9dcdbfab で pass。D1 と D3 は sync-docs の 7e025b45 で直し、D2 は cycle 2 の直しに入れる)
- Test report: docs/reports/test-2026-10-13-org-plan-approval.md(71e42662 で pass。mutation の生き残り 4 件(R03、R04、R05、R11)は cycle 2 の直しで probe のテスト 3 本を足して閉じる)
- Implementation context summary: `codex exec review --base main` を 7e025b45 で実行した(モデルは gpt-6-astra、effort は xhigh、sandbox は read-only。`codex rc=0` で `-o` のファイルは 1,690 バイト)。Codex は read-only の sandbox で Go のビルドディレクトリを作れず、テストは走らせていない。指摘は P2 が 2 件。#2 は self-review の 3 回目の N7(tech-debt の 219 行)と同じ穴で、その行の直し方をそのまま使う。同じ関数の N6(218 行)も同じ直しで閉じられるので、cycle 2 で一緒に直して 218 行と 219 行を消す

## ACTION_REQUIRED

| # | Reviewer finding | Triage rationale | Affected file(s) |
|---|-------------------|------------------|-------------------|
| 1 | [P2] 承認の検査は `AgentWait` で宛先の座席の手が空くのを待つ前に 1 度だけ走る。待つ間に計画が変わると、座席が空いたあと `PaneSendText` まで検査し直さずに TASK を送り、承認されていない版で作業させうる。早い検査は残し、待ったあと `PaneSendText` の前に新しい状態で検査し直し、待つ間に計画を変える回帰テストを足す | 本物で、直す価値がある。待つのは宛先の implementer が作業中のときで、その implementer が同じ worktree の計画を書き換えうる。待ちは `--timeout-ms` まで続くので、検査と送信の間の窓は秒から分になる。直しは manifest の読み直しと `sendApprovalErr` の呼び直しだけで小さい。検査はその時点の判定という設計(tripwire は境界ではない)は変えず、窓を待ちの長さから送る直前までに縮める | `internal/org/verbs.go`(`Send` の `AgentWait` のあと)、`internal/org/approval_enforce_test.go` |
| 2 | [P2] 承認 A、B の順のあと、ファイルを A に戻し、TASK が正しく B を名指すと、A にも承認があるので、`checkTaskPlanApproved` は TASK の `PLAN:` を A に変えるよう案内する。leader は置き換えられた版を実装し、雛形の手順 5(最後に承認された digest を使う)と食い違う。案内する前に承認の順を見て、TASK がすでに最新の承認を名指すなら「承認のあとに変わった」の拒否を保つ | 本物で、直す価値がある。self-review の 3 回目の N7 と同じ穴で、tech-debt の 219 行に直し方まで書いてある(`checkPlanApproved` が見つける path の `latest` を使い、TASK の digest がそれなら TASK 自身の拒否を返す)。台帳の今の命の承認はどれも人が承認した版なので、承認されていない版は動かない。それでも、雛形と拒否の文が別の digest を言い、人が置き換えた版を実装させる。直しは同じ関数の中で小さく、N6(218 行、今の digest が承認済みで digest の外が変わったとき)の直しも同じ分岐で入る | `internal/org/approval.go`(`checkTaskPlanApproved`)、`internal/org/approval_test.go`(`TestCheckTaskPlanApproved_StaleOrMistypedDigest`)、`docs/tech-debt/README.md`(218 行と 219 行) |

## WORTH_CONSIDERING

| # | Reviewer finding | Triage rationale | Affected file(s) |
|---|-------------------|------------------|-------------------|

## DISMISSED

| # | Reviewer finding | Dismissal reason | Category |
|---|-------------------|------------------|----------|

Categories: false-positive, already-addressed, style-preference, out-of-scope, context-aware-safe
