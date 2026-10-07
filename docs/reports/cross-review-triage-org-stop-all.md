# Cross-review triage report: org-stop-all

- Date: 2026-10-08
- Plan: docs/plans/active/2026-10-07-org-stop-all.md
- Base branch: main (f423f230)
- Driver: claude
- Reviewer: codex
- Triager: Claude Code (main context)
- Reviewer status: complete
- Self-review cross-ref: yes
- Cycle: 2/2 (cap reached)
- Total reviewer findings: 3
- After triage: ACTION_REQUIRED=1, WORTH_CONSIDERING=2, DISMISSED=0

## Triage context

- Active plan: docs/plans/active/2026-10-07-org-stop-all.md(AC1〜AC16、S1〜S8、承認 digest 2f2cfde39e9f)
- Self-review report: docs/reports/self-review-2026-10-07-org-stop-all.md(cycle 2、Merge 可、LOW の C2-1〜C2-6)
- Verify report: docs/reports/verify-2026-10-07-org-stop-all.md(cycle 2、pass)
- Implementation context summary: cycle 1 の ACTION_REQUIRED(自分の pane や workspace の最後の close が失敗すると打ち直しが効かない)を、S8 で直した。close が失敗したら、補償の `spawned` と `org_workspace_created` を書き足して台帳を「動いている」に戻す。そのあと sync-docs が help・skill・leader の雛形の文を直した(3f52d676)。この回の review は HEAD de3461ae に対して、read-only の sandbox で動いた。テストは回していない(本人の申告)。3 件とも、指摘の中身は tech-debt の行にすでに入っている(167 行と 169 行)

## ACTION_REQUIRED

| # | Reviewer finding | Triage rationale | Affected file(s) |
|---|-------------------|------------------|-------------------|
| 1 | [P2] 閉じられなかった stop は `stop_failed` だけを書き、watchdog の `leaderActivityEventCount` はこれを数えない。herdr が応答しないときに leader が座席を止めようとしても、活動として数えられず、`checkDeadman` が leader の無活動として人に上げる | この PR で入った後退と判断した。main の `Stop` は、C-c が失敗しても `stopped`(`pane=failed: …`)を書いていた(`git show f423f230:internal/org/verbs.go` の :658 と :713)。`stopped` は `internal/org/watch.go` の `leaderActivityEventCount` が数える。この PR は失敗時に `stop_failed` を書くように変えたので、herdr が応答しない間の leader の stop は数えられなくなった。直すには `EventStopFailed` を数える種類に足し、テストを 1 本足す。tech-debt の 167 行に記録済みだが、まだ直していない | internal/org/watch.go(leaderActivityEventCount)、internal/org/verbs.go(Stop の stop_failed の分岐) |

## WORTH_CONSIDERING

| # | Reviewer finding | Triage rationale | Affected file(s) |
|---|-------------------|------------------|-------------------|
| 2 | [P2] 補償は、driver を呼ぶ前に読んだ台帳の写しで判断する。後回しの close を待つ間に、ほかのプロセスが同じ座席を spawn し直すと、古い pane の `spawned` を新しい座席の記録のあとに足してしまい、Roster は新しい pane を見失う。manifest の lock の下で読み直してから、条件つきで書き足すべき | self-review の C2-1 と同じ指摘で、tech-debt の 169 行の (a) に記録済み。起きるのは、herdr が応答しない 10〜40 秒の間に、止めている最中の座席を別のプロセスが spawn し直したときだけで、2 段目の使い方(leader が自分の org を閉じる)では起きにくい。ただ、director が座席を立て直す 8 段目では起きうる。直すには、条件つきで追記する API が manifest の層に要る | internal/org/verbs.go(reactivateSeat、CloseDeferredSelfPane、CloseDeferredSelfWorkspace)、internal/org/manifest.go |
| 3 | [P3] workspace 側の補償は、自分の座席を `HERDR_PANE_ID` と一致する過去の記録から探す。herdr が pane の id を振り直したあとでは、この disband が止めていない昔の座席を active に戻しうる。その座席は tab の label の確認に落ちるので、そのあとの disband は workspace を閉じない。Disband が後回しにした座席を、そのまま持ち回すべき | self-review の C2-2 と同じ指摘で、tech-debt の 169 行の (b) に記録済み。起きるのは、herdr のセッションの保存ファイルが失われて id が振り直され、しかも座席でない pane から disband を打ち、その pane の id が同じ org の昔の座席と一致したときだけ。そのあとの disband が止まる側に倒れるので、関係のないものを閉じることはない(`--force` で片付けられる) | internal/org/verbs.go(CloseDeferredSelfWorkspace の自分の座席の探し方) |

## DISMISSED

| # | Reviewer finding | Dismissal reason | Category |
|---|-------------------|------------------|----------|

## 付録: cycle 1(2026-10-08、HEAD 2c97407c に対する review)

cycle 1 の triage の原文。件数の行だけ、件数を読むスクリプトが拾わない書き方に変え、見出しを 1 段下げた。cycle 1 の ACTION_REQUIRED は S8(53807a82)で直した。

- Cycle 1 counts: ACTION_REQUIRED=1, WORTH_CONSIDERING=2, DISMISSED=0(Total reviewer findings: 3)

### Triage context

- Active plan: docs/plans/active/2026-10-07-org-stop-all.md(AC1〜AC15、S1〜S7、承認 digest 15eb2a79e2f6)
- Self-review report: docs/reports/self-review-2026-10-07-org-stop-all.md(Merge 可、LOW 8 件)
- Verify report: docs/reports/verify-2026-10-07-org-stop-all.md(pass)
- Implementation context summary: stop と disband は、C-c を送って pane を閉じる前に、tab と workspace の label で持ち主を確かめる。コマンドを打った pane と workspace は、台帳への記録と出力をすべて済ませてから最後に閉じる(AC12)。閉じるとコマンドのプロセスも終わるので、記録は閉じる前に書くしかない。leader の雛形(`internal/org/prompts/leader.md` の手順 8)は、`disband` をセッションの最後のコマンドにし、終了コード 1 なら打ち直すよう書いている。Codex の review は read-only の sandbox で動いたため、Go のテストは回せていない(本人の申告)

### ACTION_REQUIRED

| # | Reviewer finding | Triage rationale | Affected file(s) |
|---|-------------------|------------------|-------------------|
| 1 | [P1] 後回しにした自分の pane や workspace の close が失敗すると、台帳は座席を stopped、workspace を closed、org を disbanded と記録したまま残る。leader は動き続けるのに、`disband --all` はその org を飛ばし、`disband --org-id` も閉じる対象を見つけない。`stop --all` も残った座席を飛ばす | 本物の問題と判断した。`openOrgWorkspaces`(`internal/org/spawn.go:1729`)は closed を書いた workspace を外し、`orgsToDisband`(`internal/org/verbs_all.go:307`)は disbanded のあとに記録のない org を外す。このため、打ち直した `disband` は何も閉じずに終了コード 0 で終わる。leader の雛形の手順 8 は「終了コード 1 なら打ち直す」と書いているので、ふつうの締めの経路で、打ち直しが効かないまま leader の workspace が残る。この経路は 2 段目の締めの順(AC12、S6)の中にあり、計画の範囲に入る | internal/org/verbs.go:1040(CloseDeferredSelfWorkspace、Codex の指した :1061-1062 はその中の close)、:998(CloseDeferredSelfPane)、internal/org/spawn.go:1729、internal/org/verbs_all.go:307、internal/cli/org.go:883、internal/org/prompts/leader.md:34-39 |

### WORTH_CONSIDERING

| # | Reviewer finding | Triage rationale | Affected file(s) |
|---|-------------------|------------------|-------------------|
| 2 | [P1] `disband --force` で、座席の pane が tab の label の確認に落ちても、workspace の label が org_id なら workspace を閉じる。その結果、確認に落ちた pane も workspace ごと閉じる | 閉じる workspace は label で org のものと確かめてあり、`--force` を付けないときも、座席が全部止まれば同じ workspace を中の pane ごと閉じる。持ち主の確認の境界は workspace なので、ralph と関係のない workspace を閉じることはない。ただ、sync-docs で足したヘルプの「`--force` は確認に落ちたものを閉じずに記録だけする」は、disband では workspace を閉じる段で中の pane も終わることに触れていない。直すなら文書の 1 文で足りる | internal/org/verbs.go:1560-1567、internal/cli/org.go(disband の Long) |
| 3 | [P2] `stop --force` / `disband --force` が自分の pane や workspace を最後に閉じて失敗すると、終了コードは 1 になる。AC10 の「`--force` は終了コード 0」と合わない | AC10 の文面とは食い違う。ただ、この場合は台帳がすでに閉じたと書いており、終了コード 1 だけが実際の状態とのずれを知らせる(`closeDeferredSelf` の doc comment)。どちらに寄せるかは #1 の直し方で決まる(#1 で打ち直せる状態に戻すなら、`--force` のときも 0 にしてよい) | internal/cli/org.go:883-896 |

### DISMISSED

| # | Reviewer finding | Dismissal reason | Category |
|---|-------------------|------------------|----------|
