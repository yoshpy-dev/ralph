# Cross-review triage report: org-stop-all

- Date: 2026-10-08
- Plan: docs/plans/active/2026-10-07-org-stop-all.md
- Base branch: main (f423f230)
- Driver: claude
- Reviewer: codex
- Triager: Claude Code (main context)
- Reviewer status: complete
- Self-review cross-ref: yes
- Cycle: 1/2
- Total reviewer findings: 3
- After triage: ACTION_REQUIRED=1, WORTH_CONSIDERING=2, DISMISSED=0

## Triage context

- Active plan: docs/plans/active/2026-10-07-org-stop-all.md(AC1〜AC15、S1〜S7、承認 digest 15eb2a79e2f6)
- Self-review report: docs/reports/self-review-2026-10-07-org-stop-all.md(Merge 可、LOW 8 件)
- Verify report: docs/reports/verify-2026-10-07-org-stop-all.md(pass)
- Implementation context summary: stop と disband は、C-c を送って pane を閉じる前に、tab と workspace の label で持ち主を確かめる。コマンドを打った pane と workspace は、台帳への記録と出力をすべて済ませてから最後に閉じる(AC12)。閉じるとコマンドのプロセスも終わるので、記録は閉じる前に書くしかない。leader の雛形(`internal/org/prompts/leader.md` の手順 8)は、`disband` をセッションの最後のコマンドにし、終了コード 1 なら打ち直すよう書いている。Codex の review は read-only の sandbox で動いたため、Go のテストは回せていない(本人の申告)

## ACTION_REQUIRED

| # | Reviewer finding | Triage rationale | Affected file(s) |
|---|-------------------|------------------|-------------------|
| 1 | [P1] 後回しにした自分の pane や workspace の close が失敗すると、台帳は座席を stopped、workspace を closed、org を disbanded と記録したまま残る。leader は動き続けるのに、`disband --all` はその org を飛ばし、`disband --org-id` も閉じる対象を見つけない。`stop --all` も残った座席を飛ばす | 本物の問題と判断した。`openOrgWorkspaces`(`internal/org/spawn.go:1729`)は closed を書いた workspace を外し、`orgsToDisband`(`internal/org/verbs_all.go:307`)は disbanded のあとに記録のない org を外す。このため、打ち直した `disband` は何も閉じずに終了コード 0 で終わる。leader の雛形の手順 8 は「終了コード 1 なら打ち直す」と書いているので、ふつうの締めの経路で、打ち直しが効かないまま leader の workspace が残る。この経路は 2 段目の締めの順(AC12、S6)の中にあり、計画の範囲に入る | internal/org/verbs.go:1040(CloseDeferredSelfWorkspace、Codex の指した :1061-1062 はその中の close)、:998(CloseDeferredSelfPane)、internal/org/spawn.go:1729、internal/org/verbs_all.go:307、internal/cli/org.go:883、internal/org/prompts/leader.md:34-39 |

## WORTH_CONSIDERING

| # | Reviewer finding | Triage rationale | Affected file(s) |
|---|-------------------|------------------|-------------------|
| 2 | [P1] `disband --force` で、座席の pane が tab の label の確認に落ちても、workspace の label が org_id なら workspace を閉じる。その結果、確認に落ちた pane も workspace ごと閉じる | 閉じる workspace は label で org のものと確かめてあり、`--force` を付けないときも、座席が全部止まれば同じ workspace を中の pane ごと閉じる。持ち主の確認の境界は workspace なので、ralph と関係のない workspace を閉じることはない。ただ、sync-docs で足したヘルプの「`--force` は確認に落ちたものを閉じずに記録だけする」は、disband では workspace を閉じる段で中の pane も終わることに触れていない。直すなら文書の 1 文で足りる | internal/org/verbs.go:1560-1567、internal/cli/org.go(disband の Long) |
| 3 | [P2] `stop --force` / `disband --force` が自分の pane や workspace を最後に閉じて失敗すると、終了コードは 1 になる。AC10 の「`--force` は終了コード 0」と合わない | AC10 の文面とは食い違う。ただ、この場合は台帳がすでに閉じたと書いており、終了コード 1 だけが実際の状態とのずれを知らせる(`closeDeferredSelf` の doc comment)。どちらに寄せるかは #1 の直し方で決まる(#1 で打ち直せる状態に戻すなら、`--force` のときも 0 にしてよい) | internal/cli/org.go:883-896 |

## DISMISSED

| # | Reviewer finding | Dismissal reason | Category |
|---|-------------------|------------------|----------|
