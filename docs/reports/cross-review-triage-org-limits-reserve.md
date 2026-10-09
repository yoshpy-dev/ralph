# Cross-review triage report: org-limits-reserve

- Date: 2026-10-09
- Plan: docs/plans/active/2026-10-08-org-limits-reserve.md
- Base branch: main (merge-base 51855166。main は 0931f791 に進んでいるが、diff は merge-base から取る。`git merge-tree` でぶつからない)
- Driver: claude
- Reviewer: codex
- Triager: Claude Code (main context)
- Reviewer status: complete
- Self-review cross-ref: yes
- Cycle: 3/3 (cap reached。ユーザーが上限を 3 に上げた回。cross-review の skill の決まりで `cycle-count.json` は 2 のまま)
- Total reviewer findings: 2
- After triage: ACTION_REQUIRED=0, WORTH_CONSIDERING=2, DISMISSED=0

## Triage context

- Active plan: docs/plans/active/2026-10-08-org-limits-reserve.md(AC1〜AC15、S1〜S4、承認 digest 1a165903b5df)
- Self-review report: docs/reports/self-review-2026-10-08-org-limits-reserve.md(cycle 3、Merge 可、LOW の C3-1〜C3-4)
- Verify report: docs/reports/verify-2026-10-08-org-limits-reserve.md(cycle 3、pass、V3-1〜V3-4)
- Implementation context summary: cycle 2 の WORTH_CONSIDERING は a94c914f で直した。自分の pane か workspace の close が失敗したときの補償は、台帳のロックの下で読み直してから判断し、新しい記録があれば戻さない。この回の review は HEAD 736ce19d に対して read-only の sandbox で動いた。Go のテストは回していない(本人の申告)

## ACTION_REQUIRED

| # | Reviewer finding | Triage rationale | Affected file(s) |
|---|-------------------|------------------|-------------------|

## WORTH_CONSIDERING

| # | Reviewer finding | Triage rationale | Affected file(s) |
|---|-------------------|------------------|-------------------|
| 1 | [P2] 自分の workspace の close を待つ間に別の `disband` が打たれると、何も止めずに `disbanded` だけが足される。そのあと元の close が失敗すると、workspace と leader は戻るが、予約は「最後の disbanded の前の予約」が nil になるので戻らない。org は走っている扱いなのに予約を持たず、ほかの org が同じ範囲を予約できる。`TestOrgCloseDeferredSelfWorkspace_NewerRecordsWhileClosing_RulesDiffer` がこの状態を示している | 本物の問題と判断した。verify の V3-1、self-review の C3-4 と同じで、/test が `RulesDiffer` のテストで今の振る舞いを固定した。plan の進捗の節と `docs/tech-debt/README.md` の 179 行目に記録済み。起きるのは、締めの disband の close が失敗し、その数十秒の間に同じ org にもう一度 disband が打たれたときだけ。打ち直しの disband で解ける。3 つの補償の「新しい記録」の判定をそろえる設計の判断が要る(tech-debt の 179 行目の (d)) | internal/org/verbs.go(reserveAgain、compensateUnderLock) |
| 2 | [P2] 呼んだ pane が org の記録した workspace の外にあるとき、`Disband` は workspace ではなく pane の close を後回しにする。その close が `--force` なしで失敗すると、補償は `disbanded` のあとに座席を戻すが、`disbanded` が解いた予約は戻さない。org は予約を失ったまま走り、ほかの org がその範囲を予約できる | 本物の問題と判断した。plan の進捗の (a) と、1 回目の self-review の F-10 と同じ指摘で、`docs/tech-debt/README.md` の 172 行目の (b) に記録済み。起きるのは、leader の pane が org の workspace の外にある場合だけ(headless の leader は org の workspace の中で立つので、ふつうは当たらない)。ふつうの `stop` と disband から来た pane の補償を分ける必要があり、直すには設計の判断が要る | internal/org/verbs.go(CloseDeferredSelfPane の補償) |

## DISMISSED

| # | Reviewer finding | Dismissal reason | Category |
|---|-------------------|------------------|----------|

## 付録 B: cycle 2(2026-10-09、HEAD 27fefc47 に対する review)

cycle 2 の triage の原文。件数の行だけ、件数を読むスクリプトが拾わない書き方に変え、見出しを 1 段下げた。cycle 2 の WORTH_CONSIDERING は a94c914f で直した。

- Cycle 2 counts: ACTION_REQUIRED=0, WORTH_CONSIDERING=1, DISMISSED=0(Total reviewer findings: 1)

### Triage context

- Active plan: docs/plans/active/2026-10-08-org-limits-reserve.md(AC1〜AC15、S1〜S4、承認 digest 1a165903b5df)
- Self-review report: docs/reports/self-review-2026-10-08-org-limits-reserve.md(cycle 2、Merge 可、LOW の C2-1〜C2-7)
- Verify report: docs/reports/verify-2026-10-08-org-limits-reserve.md(cycle 2、pass)
- Implementation context summary: cycle 1 の ACTION_REQUIRED は 975df92b と afcbc6c2 で直した。動いていない leader への予約の前に、`validateMaxOrgs` で `max_orgs` を判定する。この回の review は HEAD 27fefc47 に対して read-only の sandbox で動いた。Go のテストは回していない(本人の申告)

### ACTION_REQUIRED

| # | Reviewer finding | Triage rationale | Affected file(s) |
|---|-------------------|------------------|-------------------|

### WORTH_CONSIDERING

| # | Reviewer finding | Triage rationale | Affected file(s) |
|---|-------------------|------------------|-------------------|
| 1 | [P2] 自分の workspace の close が失敗したときの補償は、予約を書き戻す前に今の予約を確かめ直さない。`closeSelfWorkspace` が herdr を待っている間に同じ org が新しい予約で立て直されると、`reserveAgain` は herdr を呼ぶ前の台帳の写しで判断するので、新しい予約のあとに古いパスを書き足す。`ActiveReservation` は最新の記録を使うので、立て直した org の予約が黙って古いものに置き換わり、ほかの org がその org の実際の範囲を予約できるようになる。台帳を読み直し、共通のロックの下で確かめてから書き足すべき | 本物の問題と判断した。1 回目の self-review の F-5 と同じ指摘で、`docs/tech-debt/README.md` の 172 行目の (a) にすでに記録してある。起きるのは、締めの disband で自分の workspace の close が失敗し、herdr の呼び出しの期限(10 秒が最大 3 回)の間に同じ org_id が新しい予約で立て直されたときだけ。2 段目の補償にもある「呼ぶ前の写しで判断する」問題(tech-debt の 170 行目、C2-1)と同じ形で、直すにはロックの下で読み直して条件つきで書き足す手順が要る。パイプラインの上限に達しているので、直すかどうかはユーザーが決める | internal/org/verbs.go(reserveAgain と、それを呼ぶ CloseDeferredSelfWorkspace) |

### DISMISSED

| # | Reviewer finding | Dismissal reason | Category |
|---|-------------------|------------------|----------|

## 付録 A: cycle 1(2026-10-08、HEAD 196205f8 に対する review)

cycle 1 の triage の原文。件数の行だけ、件数を読むスクリプトが拾わない書き方に変え、見出しを 1 段下げた。cycle 1 の ACTION_REQUIRED は 975df92b と afcbc6c2 で直した。

- Cycle 1 counts: ACTION_REQUIRED=1, WORTH_CONSIDERING=0, DISMISSED=0(Total reviewer findings: 1)

### Triage context

- Active plan: docs/plans/active/2026-10-08-org-limits-reserve.md(AC1〜AC15、S1〜S4、承認 digest 1a165903b5df)
- Self-review report: docs/reports/self-review-2026-10-08-org-limits-reserve.md(Merge 可。MEDIUM の F-1 と LOW の F-3 は 8dd19634 で直した)
- Verify report: docs/reports/verify-2026-10-08-org-limits-reserve.md(pass)
- Implementation context summary: 座席を立てるとき、台帳のロックの中で `max_seats` → `max_orgs` → `max_total_seats` → 予約の順に判定する。すでに立っている leader への spawn に `--reserve` を渡すと、`idempotentRespawn` が、既存の座席を返す前に予約を判定する(AC14)。review は HEAD 196205f8 に対して read-only の sandbox で動いた。テストは回していない(本人の申告)

### ACTION_REQUIRED

| # | Reviewer finding | Triage rationale | Affected file(s) |
|---|-------------------|------------------|-------------------|
| 1 | [P2] すでに立っている leader への予約の経路は、座席が動いているかを確かめない。古い台帳で leader の `spawned` のあとに、`stopped` なしで `disbanded` が来ていると、Roster はその座席を `spawned` のまま Active でないと示す。この座席に `--reserve` で打ち直すと、上限を見ずに `scope_reserved` を書く。`max_orgs = 1` でほかの org が走っていても、2 つ目の org が走っている扱いになる | 本物の問題と判断した。`Spawn` の 2 か所(`internal/org/spawn.go` の Phase 1 と Phase 2)は、最後の状態が `spawned` なら Active を見ずに `idempotentRespawn` を呼ぶ。`idempotentRespawn` は予約を書くだけで、上限は判定しない。予約は「走っている org」の条件の 1 つなので、走っていなかった org が枠を取る。今の `Disband` はかならず先に座席を止める(`stopped` か `stop_failed`、`--force` でも `stopped`)ので、起きるのは古い ralph の台帳だけ。ただ、この PR が入れる上限をすり抜ける穴で、計画の範囲(AC1、AC14)の中にあり、直すのは数行で済む | internal/org/spawn.go(idempotentRespawn と、その 2 つの呼び出し元) |

### WORTH_CONSIDERING

| # | Reviewer finding | Triage rationale | Affected file(s) |
|---|-------------------|------------------|-------------------|

### DISMISSED

| # | Reviewer finding | Dismissal reason | Category |
|---|-------------------|------------------|----------|
