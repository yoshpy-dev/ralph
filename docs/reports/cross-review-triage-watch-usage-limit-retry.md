# Cross-review triage report: watch-usage-limit-retry

- Date: 2026-11-20
- Plan: docs/plans/active/2026-11-20-watch-usage-limit-retry.md
- Base branch: main (26b9925c; merge base 26b9925c)
- Driver: claude
- Reviewer: codex
- Triager: Claude Code (main context)
- Reviewer status: complete
- Self-review cross-ref: yes
- Cycle: 2/2 (cap reached)
- Total reviewer findings: 0
- After triage: ACTION_REQUIRED=0, WORTH_CONSIDERING=0, DISMISSED=0

## Triage context

- Active plan: docs/plans/active/2026-11-20-watch-usage-limit-retry.md(承認 70b063e49ad8。cycle 1 の直しは Scope の 3 からのずれとして Progress に記録した)
- Self-review report: docs/reports/self-review-2026-11-20-watch-usage-limit-retry.md(cycle 2、Merge: Yes、LOW L7〜L11。L7・L10・L11 は a5ec5464 でコメントと help を直し、L7 の register の部分・L8・L9 は /sync-docs が tech-debt に書いた)
- Verify report: docs/reports/verify-2026-11-20-watch-usage-limit-retry.md(cycle 2、pass、LOW V7〜V10 は /sync-docs で扱った)
- Test report: docs/reports/test-2026-11-20-watch-usage-limit-retry.md(cycle 2、pass。`./scripts/run-test.sh` は rc 1 で、落ちたのはこの branch が触っていない `TestGate_TwoGatesRunTheirScriptsOneAtATime`(gate.lock の待ちの競合)1 件だけ。`internal/org` の打ち直しは ok、単独の `-count=20` も 20/20。状態行の始まりの条件を外す変異で、引用した返答の 3 行が落ちる)
- Implementation context summary: codex(`-c sandbox_mode=read-only exec review --ignore-rules --base main`、HEAD 8c50d3cf、origin/main 26b9925c を取り込んだあと)は指摘なし。返答の全文は「No actionable regressions were found in usage-limit detection, resume reporting, or the shared status-symbol handling. Tests and live-pane behavior were not independently verified in this read-only review.」。テストは /test の cycle 2 が回した。本物の pane での確かめがないことは計画の Known gaps(fixture は issue の断片から組み立てた)のとおり。レビューの前後で worktree の git の状態は変わっていない(`cmp` で一致)

## ACTION_REQUIRED

| # | Reviewer finding | Triage rationale | Affected file(s) |
|---|-------------------|------------------|-------------------|

## WORTH_CONSIDERING

| # | Reviewer finding | Triage rationale | Affected file(s) |
|---|-------------------|------------------|-------------------|

## DISMISSED

| # | Reviewer finding | Dismissal reason | Category |
|---|-------------------|------------------|----------|

Categories: false-positive, already-addressed, style-preference, out-of-scope, context-aware-safe

## Cycle 1(履歴)

cycle 1 の triage をそのまま残す。ACTION_REQUIRED #1(引用した返答を上限の待ちと読む)は 51fe7ddf で直した。

- Date: 2026-11-20
- Plan: docs/plans/active/2026-11-20-watch-usage-limit-retry.md
- Base branch: main (a672b72a; merge base a672b72a)
- Driver: claude
- Reviewer: codex
- Triager: Claude Code (main context)
- Reviewer status: complete
- Self-review cross-ref: yes
- Cycle: 1/2
- Total reviewer findings: 1
- After triage: ACTION_REQUIRED=1, WORTH_CONSIDERING=0, DISMISSED=0

### Triage context

- Active plan: docs/plans/active/2026-11-20-watch-usage-limit-retry.md(承認 70b063e49ad8)
- Self-review report: docs/reports/self-review-2026-11-20-watch-usage-limit-retry.md(Merge: yes)
- Verify report: docs/reports/verify-2026-11-20-watch-usage-limit-retry.md(pass、LOW 6 件は 29b5d5d6 と /sync-docs で扱った)
- Test report: docs/reports/test-2026-11-20-watch-usage-limit-retry.md(pass。時刻の一致を外す変異で単体の表だけが落ちる)
- Implementation context summary: codex(`-c sandbox_mode=read-only exec review --ignore-rules --base main`、HEAD 944a7820)の指摘は 1 件(P2)。Go のテストは read-only の sandbox で build のディレクトリを作れず走らなかった(テストは /test が `./scripts/run-test.sh` で通した)。レビューの前後で worktree の git の状態は変わっていない(`cmp` で一致)。3 つ目の読み方は、入力欄の上の最後の block が `Retrying in` と `attempt` を含むかを部分文字列で見るだけで、その block が claude の状態行かどうかを見ていない。`ClassifyResumeScreen` は待ちの文を block のどこにあっても数える(resume が何も打たない向きなので安全、`resume_screens.go` の画面の型の 4 (b))。そのため、手番の時間の行を出さない設定の座席では、issue #353 の 2 行を引用した返答が最後の block になると、待ちと読まれ、3 つ目の読み方も通る

### ACTION_REQUIRED

| # | Reviewer finding | Triage rationale | Affected file(s) |
|---|-------------------|------------------|-------------------|
| 1 | [P2] 最後の block が本物の再試行の状態行かを確かめずに上限と読む。手番の時間の表示を切った claude で、issue #353 の上限の行と再試行の行を引用して終えた返答が入力欄の上の最後の block になると、見張りは座席の stall を `usage_limited` に替え、resume は解除待ちの理由を出す。状態行の形を確かめ、引用した返答の回帰テストを足すべき | 本当の問題: 3 つ目の読み方は待ちの文の緩い読み(どこにあっても数える)をそのまま受け継ぐが、こちらは resume の何も打たない判断ではなく、見張りの件の種類(stall か、人へすぐ届く `usage_limited` か)と resume の理由の文を変える。この repo の座席は issue #353 自体を扱うことがあり、引用は起こりうる。直す価値: 最後の block の先頭が claude の状態行の記号(`send_screen.go` の `sendClaudeBusyLine` と同じ `*·✢✳✶✻✽`)と空白か行末で始まることを足すだけで、issue の画面(`✻` の 1 行で始まる)と evidence 2026-11-07 の `✻ Can't reach the… · Retrying in 4s · attempt 13/300` はどちらも通る。引用した返答(`⏺` で始まる)と、返答の後ろの段落(記号なし)の回帰テストを足す | internal/org/usage_limit_screen.go(`usageLimitRetryingTurn`)、internal/org/usage_limit_screen_test.go |

### WORTH_CONSIDERING

| # | Reviewer finding | Triage rationale | Affected file(s) |
|---|-------------------|------------------|-------------------|

### DISMISSED

| # | Reviewer finding | Dismissal reason | Category |
|---|-------------------|------------------|----------|

Categories: false-positive, already-addressed, style-preference, out-of-scope, context-aware-safe
