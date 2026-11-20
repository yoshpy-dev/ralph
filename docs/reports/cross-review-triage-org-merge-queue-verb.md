# Cross-review triage report: org-merge-queue-verb

- Date: 2026-11-20
- Plan: docs/plans/active/2026-11-20-org-merge-queue-verb.md
- Base branch: main (d919479a; merge base d919479a)
- Driver: claude
- Reviewer: codex
- Triager: Claude Code (main context)
- Reviewer status: complete
- Self-review cross-ref: yes
- Cycle: 2/2 (cap reached)
- Total reviewer findings: 0
- After triage: ACTION_REQUIRED=0, WORTH_CONSIDERING=0, DISMISSED=0

## Triage context

- Active plan: docs/plans/active/2026-11-20-org-merge-queue-verb.md(承認 d584e5cd388b。cycle 1 の直しで D7 と Scope の文を変え、digest を取り直した)
- Self-review report: docs/reports/self-review-2026-11-20-org-merge-queue-verb.md(run 2、Merge: yes、LOW 4。L12・L13 は ad178695 で直し、L14 は /sync-docs が tech-debt に書いた)
- Verify report: docs/reports/verify-2026-11-20-org-merge-queue-verb.md(run 2、pass)
- Test report: docs/reports/test-2026-11-20-org-merge-queue-verb.md(run 2、pass。gate の境目を戻す変異で新しい 2 つの subtest が落ちる)
- Implementation context summary: codex(`-c sandbox_mode=read-only exec review --ignore-rules --base main`、HEAD e3f0bd11)は指摘なし。返答の全文は「No actionable regressions were identified within the patch's scope. Diff checks passed, but Go tests could not start because the read-only sandbox blocked temporary-directory creation; live GitHub queue behavior remains unverified.」。テストが走らなかったのは read-only の sandbox のためで、テストは /test の run 2 が `./scripts/run-test.sh` で通した(rc 0)。GitHub の queue での実走がないことは、計画の Non-goals とユーザーの決定(fake で実装し、実地は release のあと下流で確かめる)どおりで、tech-debt に行がある。レビューの前後で worktree の git の状態は変わっていない(`cmp` で一致)

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

cycle 1 の triage をそのまま残す。ACTION_REQUIRED #1(D7 の二重記録の防止が最新の gate の記録より後だけを探していた)は 2c9a7fe6 で直した。

- Date: 2026-11-20
- Plan: docs/plans/active/2026-11-20-org-merge-queue-verb.md
- Base branch: main (d919479a; merge base d919479a)
- Driver: claude
- Reviewer: codex
- Triager: Claude Code (main context)
- Reviewer status: complete
- Self-review cross-ref: yes
- Cycle: 1/2 (the first run)
- Total reviewer findings: 1
- Cycle 1 after triage: ACTION_REQUIRED=1, WORTH_CONSIDERING=0, DISMISSED=0

### Triage context

- Active plan: docs/plans/active/2026-11-20-org-merge-queue-verb.md(承認 aef9c99a6ea5)
- Self-review report: docs/reports/self-review-2026-11-20-org-merge-queue-verb.md(2 回目、Merge: yes、LOW 5)
- Verify report: docs/reports/verify-2026-11-20-org-merge-queue-verb.md(pass)
- Implementation context summary: codex(`-c sandbox_mode=read-only exec review --ignore-rules --base main`)の指摘は 1 件で、テストと GitHub の実際の挙動は確かめていないと書いている。D7(計画助言の指摘を受けた二重記録の防止)は、manifest の lock の中で「その branch の最新の gate の記録より後」にある同じ branch・PR・head の `pr_merged` を探す(`recordQueueMerge`、`internal/org/merge_queue_wait.go`)。待つ間は `regate.lock` を離すので、2 つの merge が同じ PR を待つ間に regate が gate の記録を足すと、その境目が後ろへ動き、最初の merge の `pr_merged` が境目の前に入って見えなくなる。GitHub で PR は 1 度しか merge されないので、同じ branch・PR・head の `pr_merged` は ledger のどこにあっても同じ merge の記録になる

### ACTION_REQUIRED

| # | Reviewer finding | Triage rationale | Affected file(s) |
|---|-------------------|------------------|-------------------|
| 1 | [P2] 2 つの merge が同じ PR を待つ間に local の regate が gate の記録を足すと、D7 の二重記録の防止が「最新の gate の記録より後」で探すので、最初の merge の `pr_merged` を見落とし、2 つ目が同じ merge をあとの時刻で書く。`FollowupRecordedMerge` が、本当の merge のあとに作られた follow-up を ready と読みうる。branch・PR・head で ledger 全体から探すか、元の gate の境目を持ち続ける | 本物の穴で、この PR が持ち込んだ(D7 の境目の取り方)。起きるには、2 つの待ちと、その間の regate が要るのでまれだが、記録の正しさに関わり、直しは小さい。PR は 1 度しか merge されないので、境目をなくして ledger 全体で同じ branch・PR・head を探すのが単純で正しい。queue のない道の `recordMerge` は変えない(NFR-1) | `internal/org/merge_queue_wait.go`(`recordQueueMerge`)、そのテスト |

### WORTH_CONSIDERING

| # | Reviewer finding | Triage rationale | Affected file(s) |
|---|-------------------|------------------|-------------------|

### DISMISSED

| # | Reviewer finding | Dismissal reason | Category |
|---|-------------------|------------------|----------|

Categories: false-positive, already-addressed, style-preference, out-of-scope, context-aware-safe
