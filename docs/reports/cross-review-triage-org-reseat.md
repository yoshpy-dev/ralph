# Cross-review triage report: org-reseat

- Date: 2026-11-13
- Plan: docs/plans/active/2026-11-12-org-reseat.md
- Base branch: main (14531858, the same commit as origin/main)
- Driver: claude
- Reviewer: codex
- Triager: Claude Code (main context)
- Reviewer status: complete
- Self-review cross-ref: yes
- Cycle: 1/2
- Total reviewer findings: 3
- After triage: ACTION_REQUIRED=3, WORTH_CONSIDERING=0, DISMISSED=0

## Triage context

- Active plan: docs/plans/active/2026-11-12-org-reseat.md(承認 digest 3058f6f66b5b)
- Self-review report: docs/reports/self-review-2026-11-12-org-reseat.md(3 回目、CRITICAL・HIGH・MEDIUM 0、LOW 6)
- Verify report: docs/reports/verify-2026-11-12-org-reseat.md(pass)
- Implementation context summary: `ralph org reseat` は、止める前に後で断られる条件を確かめ、止めて立て直し、途中で失敗したら打ち直しのコマンドを出す。非 leader だけを立て直したときは leader へ送る DECISION の行を出す。仕分けの線引き(人の方針): 今回の変更が持ち込んだ、知らせや案内が働かなくなる向きの穴は直す。3 件ともこれに当たる。

codex の要約(原文): "The failure-recovery paths can produce unusable retries or omit TASK-resend guidance for restarted seats."

## ACTION_REQUIRED

| # | Reviewer finding | Triage rationale | Affected file(s) |
|---|-------------------|------------------|-------------------|
| 1 | [P2] 最初の spawn が `spawn_started` か `spawn_step` で止まった座席は active なので受け付けられるが、`spawned` が無い。reseat が止めて立て直しに失敗すると、active でなくなり、出した打ち直しの `--seat` が P7 の条件(今の命で `spawned` がある)で断る。`reseated` があるので既定の対象にも入らない | 実際に起きる。reseat が出す案内が働かず、座席が止まったまま残る。今回の変更が持ち込んだ穴なので直す。直し方: 今の命で `reseat=<id>` の note を持つ event がある座席も `--seat` で受け付ける | internal/org/reseat.go |
| 2 | [P2] implementer と reviewer だけの reseat で、implementer が立ったあと reviewer の立て直しが失敗すると、DECISION が出ない。打ち直しは reviewer だけを名指しし、その成功の DECISION も reviewer だけになり、implementer の TASK の送り直しを leader に伝える案内が消える | 実際に起きる。立ち直った座席は TASK を持たないので、知らせが消えると座席が遊ぶ。直し方: leader を立て直さない reseat では、途中で失敗しても、立ち直った非 leader の座席について DECISION の行を出す | internal/org/reseat.go、internal/cli/org_reseat.go |
| 3 | [P2] `--config` を明示した reseat が途中で失敗すると、打ち直しのコマンドが `--config` を落とす。既定の設定の承認方式が違うと、打ち直しが結びつきの比較で断られる | 実際に起きる。案内どおりに打っても立て直せない。直し方: 渡された `--config` を打ち直しのコマンドに引き継ぐ(`--state-dir` と同じ扱い) | internal/org/reseat.go、internal/cli/org_reseat.go |

## WORTH_CONSIDERING

| # | Reviewer finding | Triage rationale | Affected file(s) |
|---|-------------------|------------------|-------------------|

## DISMISSED

| # | Reviewer finding | Dismissal reason | Category |
|---|-------------------|------------------|----------|

Categories: false-positive, already-addressed, style-preference, out-of-scope, context-aware-safe
