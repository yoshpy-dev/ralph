# Cross-review triage report: org-merge-order

- Date: 2026-10-14
- Plan: docs/plans/active/2026-10-14-org-merge-order.md
- Base branch: main(merge-base ac34ccf3)
- Driver: claude
- Reviewer: codex
- Triager: Claude Code (main context)
- Reviewer status: complete
- Self-review cross-ref: yes
- Cycle: 1/2
- Total reviewer findings: 1
- After triage: ACTION_REQUIRED=1, WORTH_CONSIDERING=0, DISMISSED=0

## Triage context

- Active plan: docs/plans/active/2026-10-14-org-merge-order.md(承認済み、digest c0b1e9230999。変わっていない)
- Self-review report: docs/reports/self-review-2026-10-14-org-merge-order.md(346304fc、CRITICAL・HIGH・MEDIUM なし、LOW 8、Merge: yes)
- Verify report: docs/reports/verify-2026-10-14-org-merge-order.md(a8882d6d、pass。ずれ D1・D2 は sync-docs の 93af1f87 で直した)
- Test report: docs/reports/test-2026-10-14-org-merge-order.md(e20f2616、pass。mutation 66 件のうち 58 件が red、テストの穴 M05・C12・M03・C06/C07・M09)
- Implementation context summary: `codex exec review --base main` を 93af1f87 で実行した(モデルは gpt-6-astra、effort は xhigh、sandbox は read-only。`codex rc=0` で `-o` のファイルは 1055 バイト)。Codex はテストを走らせていない。指摘は 1 件(P2)。merge-order は origin を `git ls-remote --symref origin` で読み、PR は `gh pr list --head <branch>` を打った場所の repo で引く(`internal/org/mergeorder.go` の `lookupMergePRs`)。gh がどの repository を引くかは確かめていない

## ACTION_REQUIRED

| # | Reviewer finding | Triage rationale | Affected file(s) |
|---|-------------------|------------------|-------------------|
| 1 | [P2] `gh pr list` に repository を渡していないので、`gh repo set-default` で origin でない repository を選んでいると、gh はその repository を引き、`listOriginBranches` は origin を読む。同じ名前のブランチの PR で、origin のブランチが merged / closed として隠れたり、関係のない PR が ready の行に付いたりする。`isCrossRepository` は PR の head と base の repository を比べるだけで、origin と比べてはいない。origin の repository を割り出して `--repo` で渡し、gh の既定が origin でない場合を回帰テストにする | 本物の問題で、直す。gh は remote が複数あるとき、`gh repo set-default` の既定(なければ remote の名前の優先順)で引く repository を決める(`gh repo set-default --help`)。fork から作業する repo では upstream を引き、origin の証拠と別の repository の PR の state を組み合わせて分類する。C5 で fork の PR を外したのと同じ種類の取り違えになる。別の repository の PR を読むと、merged / closed と誤って origin の ready のブランチを隠し、その PR の統合先で wrong_base と誤り、関係のない PR の URL を ready の行に出す。直し方は計画の Scope(PR の状態は任意で、取れなければ PR の欄を空にする)の範囲で選ぶ | internal/org/mergeorder.go(`lookupMergePRs`、`pickMergePR`)、internal/org/mergeorder_test.go、internal/cli/org_merge_order_test.go |

## WORTH_CONSIDERING

なし。

## DISMISSED

なし。
