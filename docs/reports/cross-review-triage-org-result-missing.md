# Cross-review triage report: org-result-missing

- Date: 2026-11-22
- Plan: docs/plans/active/2026-11-22-org-result-missing.md
- Base branch: main (2a069b50; merge base 2a069b50)
- Driver: claude
- Reviewer: codex
- Triager: Claude Code (main context)
- Reviewer status: complete
- Self-review cross-ref: yes
- Cycle: 1/2
- Total reviewer findings: 1
- After triage: ACTION_REQUIRED=1, WORTH_CONSIDERING=0, DISMISSED=0

## Triage context

- Active plan: docs/plans/active/2026-11-22-org-result-missing.md(承認 698af1d189e8)
- Self-review report: docs/reports/self-review-2026-11-22-org-result-missing.md(Run 1 の M1 と L1〜L5 は f57cb92d、Run 2 の LOW 4 件は b0a9b28a で直した)
- Verify report: docs/reports/verify-2026-11-22-org-result-missing.md(pass)
- Test report: docs/reports/test-2026-11-22-org-result-missing.md(pass、`./scripts/run-test.sh` rc 0、HEAD 6a7b39ab。Test gaps 1・2・4 はテストだけの e976f232 で閉じた)
- Implementation context summary: codex(`-c sandbox_mode=read-only exec review --ignore-rules --base main`、HEAD 31d69757)の指摘は 1 件(P2)。`Send` は台帳の `sent` を、Enter と送信の確かめ(`typeAndSubmit`、確かめの待ちを含む)が終わってから `o.now()` で書く(`internal/org/verbs.go` の 716-725 行付近)。見張りは TASK の時刻としてこの `TS` を使い、それより前の座席の行を数えない。座席が確かめの待ちのあいだに RESULT か QUESTION を送ると、その行は TASK の時刻より前に見えて捨てられ、誤った `result_missing` が上がる。Go のテストは read-only の sandbox で走らなかった。レビューの前後で worktree の git の状態は変わっていない(`cmp` で一致)

## ACTION_REQUIRED

| # | Reviewer finding | Triage rationale | Affected file(s) |
|---|-------------------|------------------|-------------------|
| 1 | [P2] 返事を TASK の実際の送信の時刻と突き合わせる。`Send` は `EventSent.TS` を `typeAndSubmit` のあと(確かめの待ちを含む)に書く(`verbs.go:722-725`)。短い手番の座席がその間に RESULT か QUESTION を送ると、`task.At` より 1 秒以上前になり、`watch_result_missing.go:212-216` の時刻の絞り込みで捨てられ、誤った `result_missing` が上がる(座席が答えを待っているときも)。送る前に取った時刻で突き合わせ、確かめの待ちのあいだに来た返事をテストで固める | 本当の問題: 台帳の `sent` の時刻は Enter より遅いので、すぐ返した座席の行を落とす。誤った ALERT の向き(見逃しではない)だが、leader を誤って動かす。直す価値: 台帳の形を変えずに、返事の照合を「TASK の時刻から、Enter から記録までの最大の幅を引いた時刻」以後にし、QUESTION のあとの leader の答えの比べも同じ幅で上げない側に寄せる(または送る前の時刻を Details に足す)だけで済む。テストで固定する | internal/org/watch_result_missing.go、internal/org/watch_result_missing_test.go(必要なら internal/org/verbs.go) |

## WORTH_CONSIDERING

| # | Reviewer finding | Triage rationale | Affected file(s) |
|---|-------------------|------------------|-------------------|

## DISMISSED

| # | Reviewer finding | Dismissal reason | Category |
|---|-------------------|------------------|----------|

Categories: false-positive, already-addressed, style-preference, out-of-scope, context-aware-safe
