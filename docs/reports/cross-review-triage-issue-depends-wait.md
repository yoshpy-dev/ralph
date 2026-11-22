# Cross-review triage report: issue-depends-wait

- Date: 2026-11-22
- Plan: docs/plans/active/2026-11-22-issue-depends-wait.md
- Base branch: main (2a069b50; merge base 2a069b50)
- Driver: claude
- Reviewer: codex
- Triager: Claude Code (main context)
- Reviewer status: complete
- Self-review cross-ref: yes
- Cycle: 1/2
- Total reviewer findings: 2
- After triage: ACTION_REQUIRED=2, WORTH_CONSIDERING=0, DISMISSED=0

## Triage context

- Active plan: docs/plans/active/2026-11-22-issue-depends-wait.md(承認 7b44cea2d6bc)
- Self-review report: docs/reports/self-review-2026-11-22-issue-depends-wait.md(Merge: yes。M1・L1・L2・L3・L5 は 03ced266 で直し、L4・L6 は /sync-docs で台帳に入れた)
- Verify report: docs/reports/verify-2026-11-22-issue-depends-wait.md(pass)
- Test report: docs/reports/test-2026-11-22-issue-depends-wait.md(pass、`./scripts/run-test.sh` rc 0、HEAD 95d755b2)
- Implementation context summary: codex(`-c sandbox_mode=read-only exec review --ignore-rules --base main`、HEAD f6d1cd35)の指摘は 2 件(どちらも P2)。どちらも、同じ名前のブランチ(`<Type>/<slug>`)の別の PR を、頼る先の機能の merge と取り違える道。merge-order の JSON は結びつきの時刻(`binding.ts`)を出すが、PR の merge の時刻は出さないので(`MergeOrderPR` は `number`・`state`・`url`・`head_ref_oid`・`base_ref_name`)、skill は `gh pr view <番号> --json mergedAt` で merge の時刻を読み、結びつきの時刻と比べる必要がある。レビューの前後で worktree の git の状態は変わっていない(`cmp` で一致)

## ACTION_REQUIRED

| # | Reviewer finding | Triage rationale | Affected file(s) |
|---|-------------------|------------------|-------------------|
| 1 | [P2] merge された PR が、一致した結びつきのものかを確かめる。昔の `feat/locks` の PR が merge 済みで、今の `locks` の org が立ったあと PR を開く前に disband されると、結びつきは今の分割計画の (a) に当たり、merge-order は昔の PR を `merged` と出す(`collectMergeTargets` は最新の結びつき、`pickMergePR` と `classifyMergeTarget` は昔の merge 済みの PR を別々に選ぶ)。走っている org を外す決まりは disband した試みを守らない | 本当の問題: 「頼る先の merge の確かめ」は結びつきと `merged` の組み合わせだけを見ており、両者が別の試みのものでも通る。直す価値: skill の確かめに「その PR の `mergedAt`(`gh pr view`)が結びつきの `binding.ts` より後」を足すだけで済む。テストで固定する | `.claude/skills/director/SKILL.md`(「起票と記録」の 1 の「頼る先の merge の確かめ」)と写し、`internal/org/skill_issue_depends_test.go` |
| 2 | [P2] `ready` の follow-up にも頼る先の確かめを掛ける。別の分割計画の `locks` の org が走っているあいだに、新しい分割計画のまだ立っていない `locks` に頼る `api` を起票すると、どちらも `feat/locks`。別の分割計画の PR が merge されると、`followupMergeState` は記録より後の merge なので `ready` を返し、`api` が頼る先のないまま着手できると出る | 本当の問題: 記録より後の merge だけを見る follow-up の `ready` は、同じ名前のブランチの別の機能の merge でも立つ。この節は記録より前の merge にだけ確かめを掛けている。直す価値: 「merge を待つ issue」の 2(着手を聞く前の判定)で、頼る先のブランチの `ready` にも同じ確かめを掛け、当たらなければ着手を聞かずに follow-up を記録し直して待つ、と書くだけで済む | `.claude/skills/director/SKILL.md`(「merge を待つ issue」の 2)と写し、`internal/org/skill_issue_depends_test.go` |

## WORTH_CONSIDERING

| # | Reviewer finding | Triage rationale | Affected file(s) |
|---|-------------------|------------------|-------------------|

## DISMISSED

| # | Reviewer finding | Dismissal reason | Category |
|---|-------------------|------------------|----------|

Categories: false-positive, already-addressed, style-preference, out-of-scope, context-aware-safe
