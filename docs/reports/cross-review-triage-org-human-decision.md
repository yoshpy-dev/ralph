# Cross-review triage report: org-human-decision

- Date: 2026-11-22
- Plan: docs/plans/active/2026-11-21-org-human-decision.md
- Base branch: main (10e9322d; merge base 10e9322d)
- Driver: claude
- Reviewer: codex
- Triager: Claude Code (main context)
- Reviewer status: complete
- Self-review cross-ref: yes
- Cycle: 2/2 (cap reached)
- Total reviewer findings: 2
- After triage: ACTION_REQUIRED=0, WORTH_CONSIDERING=2, DISMISSED=0

## Triage context

- Active plan: docs/plans/active/2026-11-21-org-human-decision.md(承認 fe3344aa0cac)
- Self-review report: docs/reports/self-review-2026-11-21-org-human-decision.md(先頭は Run 3。Run 2 の指摘は d78f379a、Run 3 の指摘は a9b160eb で直した)
- Verify report: docs/reports/verify-2026-11-21-org-human-decision.md(Run 2 が pass。V8 は 6dc7c22f と a9b160eb、V7・V9・V10 は 2 周目の /sync-docs で直した)
- Test report: docs/reports/test-2026-11-21-org-human-decision.md(Run 2 が pass、`./scripts/run-test.sh` rc 0、HEAD b337156e)
- Implementation context summary: codex(`-c sandbox_mode=read-only exec review --ignore-rules --base main`、HEAD f83e7a94、origin/main 10e9322d を 2211a00b で取り込んだあと)の指摘は 2 件(P2 が 2 件)。Go のテストは read-only の sandbox で走らなかった(テストは /test が回した)。レビューの前後で worktree の git の状態は変わっていない(`cmp` で一致)。1 周目の ACTION_REQUIRED(手の Enter の回復は記録を残さない)は、skill と rule で人の決定に手の Enter を使わせず、CLI の note も手で送らないと言う形で閉じた。2 件とも、`Send` が `sent` を台帳に書くのが Enter のあと、pane のロックを外したあとであること(`internal/org/verbs.go` の 674 行の unlock と 722 行の `appendEvent`)と、`readReseatProgress` が台帳の並びで org の命と決定の順を決めることによる。どちらも 2 つの操作が数ミリ秒の窓で重なるときにだけ起きる。director の skill は人の決定を 1 本ずつ send が終わるのを待って送り、disband は leader が PR のあとか人が打つ

## ACTION_REQUIRED

| # | Reviewer finding | Triage rationale | Affected file(s) |
|---|-------------------|------------------|-------------------|

## WORTH_CONSIDERING

| # | Reviewer finding | Triage rationale | Affected file(s) |
|---|-------------------|------------------|-------------------|
| 1 | [P2] 人の決定の記録を pane のロックの中で書く。`Send` は 674 行でロックを外してから `sent` を書くので、同じ leader への人の決定の send が 2 本重なると、先に押した側が記録で遅れ、台帳の順が届いた順と逆になりうる。`readReseatProgress` は台帳の順を届いた順として立て直した leader に渡すので、古い決定が新しい決定を上書きしうる | 本当の問題か: 起きうる(同じ leader への 2 本の send が、片方の Enter とその記録の間に、もう片方の Enter と記録が入る形で重なるとき)。直す価値: 迷う。director の skill は人の決定を 1 本ずつ、send が終わってから次を送り、人の決定を送る主体は director だけなので、ふつうの流れでは 2 本は重ならない。ロックの中で書くと、すべての send の記録の位置が変わり、記録の失敗の note(EnterPressed)の扱いにも触れる。上限の回なので直さず、PR の Known gaps と tech-debt の行にする | internal/org/verbs.go(`Send` の unlock と `appendEvent`)、internal/org/reseat_progress.go |
| 2 | [P2] 立て直しで渡す決定を、決定を送った org の命に結びつける。Enter のあと `sent` を書く前に org が disband されると、その `sent` は `disbanded` のあとに並び、`readReseatProgress` は前の leader の決定を次の命のものとして読む。Disband は send の pane のロックと調整しない。命を記録して確かめるべき(agent-messaging.md の「今の命の決定」の約束) | 本当の問題か: 起きうる(send の Enter と記録の間の数ミリ秒に disband が重なるとき。org_id を別の機能で使い回すと、次の命の leader に前の決定が渡る)。直す価値: 迷う。disband はふつう leader が PR のあとか人が打ち、人の決定の send と重なる窓は狭い。直すには `sent` に命の印(その命の始まりの事件)を持たせ、読み手で確かめる形になり、台帳の形が変わる。上限の回なので直さず、PR の Known gaps と tech-debt の行にする | internal/org/verbs.go(`Send`、`Disband`)、internal/org/reseat_progress.go、.claude/rules/ralph/agent-messaging.md |

## DISMISSED

| # | Reviewer finding | Dismissal reason | Category |
|---|-------------------|------------------|----------|

Categories: false-positive, already-addressed, style-preference, out-of-scope, context-aware-safe

## Cycle 1

- Date: 2026-11-22
- Base branch: main (9cd63d60; merge base 9cd63d60)
- Driver: claude
- Reviewer: codex
- Reviewer status: complete
- Cycle: 1/2
- Total reviewer findings: 1
- After triage (cycle 1): ACTION_REQUIRED=1, WORTH_CONSIDERING=0, DISMISSED=0

### Triage context

- Active plan: docs/plans/active/2026-11-21-org-human-decision.md(承認 fe3344aa0cac)
- Self-review report: docs/reports/self-review-2026-11-21-org-human-decision.md(Merge: yes、M1・M2・L1・L2・L4 は fc09a923 で直した)
- Verify report: docs/reports/verify-2026-11-21-org-human-decision.md(pass、V2 は b97a336c、残りは /sync-docs で直した)
- Test report: docs/reports/test-2026-11-21-org-human-decision.md(pass、`./scripts/run-test.sh` rc 0)
- Implementation context summary: codex(`-c sandbox_mode=read-only exec review --ignore-rules --base main`、HEAD 7e8ac5b1、origin/main 9cd63d60 を取り込んだあと)の指摘は 1 件(P2)。Go のテストは read-only の sandbox で走らなかった(テストは /test が回した)。レビューの前後で worktree の git の状態は変わっていない(`cmp` で一致)。`Send` は、Enter を押す前の確かめが通らずに `(text typed but not submitted)` で終わる道では、`sent` を記録せずに返る(`internal/org/verbs.go` の `SendProgressTextTyped`)。director skill の「答える順」は、そのとき DECISION の全文が leader の入力欄に残っていれば Enter を手で 1 回押してよいとし、`### 人の決定を送る` は「答える順」の 2 のとおりに送ると書くので、この回復を受け継ぐ

### ACTION_REQUIRED

| # | Reviewer finding | Triage rationale | Affected file(s) |
|---|-------------------|------------------|-------------------|
| 1 | [P2] 手で Enter を押して回復した人の決定は記録されない。send が `(text typed but not submitted)` で終わると `sent` を記録せずに返り、手の Enter も記録を作らないので、leader は決定を受け取っても、立て直したあと `readReseatProgress` の「人の決定」に載らない(agent-messaging.md の引き継ぎの約束に反する)。打ち直さずに届いたことを記録する道を足すべき | 本当の問題: この PR の目的の 1 つは、人の決定を立て直したあとの leader に引き継ぐことで、手の Enter の回復はその記録を残さない。直す価値: 人の決定にかぎって手の Enter の回復を使わず、人に leader の入力欄を空けてもらってから `ralph org send` で送り直すと skill に書けば、記録は send の exit 0 のときに必ず残る(コードは変えない)。agent-messaging.md にも、人の決定は send が exit 0 で終わったときだけ記録されると書く。打ち直さずに記録する道(コードの追加)は、入力欄に残ったまま押されない決定を記録してしまうおそれがあるので採らない | .claude/skills/director/SKILL.md(`### 人の決定を送る`、「答える順」の 2)と 3 写し、.claude/rules/ralph/agent-messaging.md と雛形の写し |

### WORTH_CONSIDERING

| # | Reviewer finding | Triage rationale | Affected file(s) |
|---|-------------------|------------------|-------------------|

### DISMISSED

| # | Reviewer finding | Dismissal reason | Category |
|---|-------------------|------------------|----------|
