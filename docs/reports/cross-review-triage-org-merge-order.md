# Cross-review triage report: org-merge-order

- Date: 2026-10-14
- Plan: docs/plans/active/2026-10-14-org-merge-order.md
- Base branch: main(merge-base ac34ccf3)
- Driver: claude
- Reviewer: codex
- Triager: Claude Code (main context)
- Reviewer status: complete
- Self-review cross-ref: yes
- Cycle: 2/2 (cap reached)
- Total reviewer findings: 1
- After triage: ACTION_REQUIRED=1, WORTH_CONSIDERING=0, DISMISSED=0

## Triage context

- Active plan: docs/plans/active/2026-10-14-org-merge-order.md(承認済み、digest c0b1e9230999。変わっていない。cycle 2 の逸脱は `## Progress checklist` の最後の項目)
- 2 周目の理由: 下の「Cycle 1」の ACTION_REQUIRED #1(gh が origin でない repository の PR を読みうる)を 15d4b947 と ee302b06 で直した。`--repo` は渡さない。gh が PR を返したら `git remote get-url origin` を 1 回だけ打ち、PR の url の owner/repo と origin の owner/repo を大文字小文字を無視して比べる。合わなければ PR をすべて捨てて注記を 1 行出し、gh をそれ以上打たない。host は比べず、owner/repo を読めない origin(ローカルのパスなど)では比べない。この 3 つの穴は tech-debt の行と help・skill・README に書いた(f1a05bc7)
- Self-review report: docs/reports/self-review-2026-10-14-org-merge-order.md(cycle 2 は 5d788e2c で Merge: yes、CRITICAL・HIGH・MEDIUM なし、LOW C2-1〜C2-6。C2-3 は sync-docs の f1a05bc7 で直し、残りは tech-debt の最後の行)
- Verify report: docs/reports/verify-2026-10-14-org-merge-order.md(cycle 2 は c8448321 で pass。ずれ D3・D4 は f1a05bc7 で直した)
- Test report: docs/reports/test-2026-10-14-org-merge-order.md(cycle 2 は 312b9df8 で pass。mutation 87 件のうち 72 件が red。本物の gh で、upstream を読む gh の PR を修正前の binary が使い、HEAD が捨てることを確かめた)
- Implementation context summary: `codex exec review --base main` を c82eb031 で実行した(モデルは gpt-6-astra、effort は xhigh、sandbox は read-only。`codex rc=0` で `-o` のファイルは 813 バイト)。Codex はテストを走らせていない。指摘は 1 件(P2)。`lookupMergePR`(`internal/org/mergeorder.go`)は `gh pr list --head <branch> --state all` に `--limit` を渡していない。`gh pr list --help` によると既定の上限は 30 で、`--head` は `<owner>:<branch>` の形を受けない

## ACTION_REQUIRED

| # | Reviewer finding | Triage rationale | Affected file(s) |
|---|-------------------|------------------|-------------------|
| 1 | [P2] fork を外す前に、PR の候補を十分に取る。同じ名前の fork の PR が 30 本以上、本物の PR より新しく並ぶと、`gh pr list` は既定の上限 30 に届き、そのあと `pickMergePR` が cross-repository の PR を外す。本物の PR は結果に入らず、PR なしとして黙って返る。PR が別の base に向いている、または閉じているブランチが `ready` に見えうる。候補をページで辿るか、サーバー側で head の repository で絞ってから、番号の一番大きい PR を選ぶ | 本物の問題で、直す。gh の `--head` 単独の検索は、作成の新しい順に `--limit` まで辿る(consult が gh の source で確かめた)。fork の PR が上限を埋めると本物の PR が落ち、merged・closed・wrong_base が見えなくなる。起きる見込みは低い(slug の名前のブランチに、fork から 30 本以上)ので、2 軸の表では「いま直す価値: 迷う」の WORTH_CONSIDERING に当たる。ただし誤りは黙って `ready` に寄る向きで、merge してよい順を示すという merge-order の目的に当たり、直す手間も小さい。保守の原則で ACTION_REQUIRED に寄せた。直し方は `--limit` を大きく(1000。通常は 1 本しか返らず、gh は 1 ページ 100 件で辿るので費用は変わらない)し、上限に届いて fork でない PR が 1 本もないときは、そのブランチの PR を使わずに注記する。サーバー側で絞る `gh api` は、cycle 1 で `--repo` をやめた理由(host の解決、ssh の別名)が戻るので採らない | internal/org/mergeorder.go(`lookupMergePR`)、internal/org/mergeorder_test.go |

## WORTH_CONSIDERING

なし。

## DISMISSED

なし。

## Cycle 1

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
- Cycle 1 counts: ACTION_REQUIRED=1, WORTH_CONSIDERING=0, DISMISSED=0

### Triage context(cycle 1)

- Active plan: docs/plans/active/2026-10-14-org-merge-order.md(承認済み、digest c0b1e9230999。変わっていない)
- Self-review report: docs/reports/self-review-2026-10-14-org-merge-order.md(346304fc、CRITICAL・HIGH・MEDIUM なし、LOW 8、Merge: yes)
- Verify report: docs/reports/verify-2026-10-14-org-merge-order.md(a8882d6d、pass。ずれ D1・D2 は sync-docs の 93af1f87 で直した)
- Test report: docs/reports/test-2026-10-14-org-merge-order.md(e20f2616、pass。mutation 66 件のうち 58 件が red、テストの穴 M05・C12・M03・C06/C07・M09)
- Implementation context summary: `codex exec review --base main` を 93af1f87 で実行した(モデルは gpt-6-astra、effort は xhigh、sandbox は read-only。`codex rc=0` で `-o` のファイルは 1055 バイト)。Codex はテストを走らせていない。指摘は 1 件(P2)。merge-order は origin を `git ls-remote --symref origin` で読み、PR は `gh pr list --head <branch>` を打った場所の repo で引く(`internal/org/mergeorder.go` の `lookupMergePRs`)。gh がどの repository を引くかは確かめていない

### ACTION_REQUIRED(cycle 1)

| # | Reviewer finding | Triage rationale | Affected file(s) |
|---|-------------------|------------------|-------------------|
| 1 | [P2] `gh pr list` に repository を渡していないので、`gh repo set-default` で origin でない repository を選んでいると、gh はその repository を引き、`listOriginBranches` は origin を読む。同じ名前のブランチの PR で、origin のブランチが merged / closed として隠れたり、関係のない PR が ready の行に付いたりする。`isCrossRepository` は PR の head と base の repository を比べるだけで、origin と比べてはいない。origin の repository を割り出して `--repo` で渡し、gh の既定が origin でない場合を回帰テストにする | 本物の問題で、直す。gh は remote が複数あるとき、`gh repo set-default` の既定(なければ remote の名前の優先順)で引く repository を決める(`gh repo set-default --help`)。fork から作業する repo では upstream を引き、origin の証拠と別の repository の PR の state を組み合わせて分類する。C5 で fork の PR を外したのと同じ種類の取り違えになる。別の repository の PR を読むと、merged / closed と誤って origin の ready のブランチを隠し、その PR の統合先で wrong_base と誤り、関係のない PR の URL を ready の行に出す。直し方は計画の Scope(PR の状態は任意で、取れなければ PR の欄を空にする)の範囲で選ぶ。選んだ直し方(`--repo` は渡さず、返った PR の url を origin と比べる)は consult の判定による。`--repo` の host は GH_HOST、認証済みの host、github.com の順に決まり、ssh の別名を解かないため | internal/org/mergeorder.go(`lookupMergePRs`、`pickMergePR`)、internal/org/mergeorder_test.go、internal/cli/org_merge_order_test.go |

### WORTH_CONSIDERING(cycle 1)

なし。

### DISMISSED(cycle 1)

なし。
