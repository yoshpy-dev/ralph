# Cross-review triage report: org-ssh-alias-host

- Date: 2026-11-20
- Plan: docs/plans/active/2026-11-19-org-ssh-alias-host.md
- Base branch: main (32274500; merge base e6d5d0ed, after the merge of origin/main into the branch at 0a71a930)
- Driver: claude
- Reviewer: codex
- Triager: Claude Code (main context)
- Reviewer status: complete
- Self-review cross-ref: yes
- Cycle: 1/2
- Total reviewer findings: 1
- After triage: ACTION_REQUIRED=1, WORTH_CONSIDERING=0, DISMISSED=0

## Triage context

- Active plan: docs/plans/active/2026-11-19-org-ssh-alias-host.md(承認 f4a9803ca80e)
- Self-review report: docs/reports/self-review-2026-11-19-org-ssh-alias-host.md(2 回目、Merge: yes。L8・L10 は 3c369ff4 で直し、L4・L6・L7・L9 は残した)
- Verify report: docs/reports/verify-2026-11-19-org-ssh-alias-host.md(2 回目で pass)
- Implementation context summary: codex(`-c sandbox_mode=read-only exec review --ignore-rules --base main`)の指摘は 1 件。codex は read-only の sandbox でビルドのディレクトリを作れず、テストは打てなかったと書いている。テストの根拠は /test の報告(run-test.sh が rc 0)になる。#340 は `ssh -G` を `realMergeDetachedExec`(自分のプロセスグループ)で打つ。計画助言の指摘(期限で止めても `Match exec` の子が残る)を受けた形で、期限と ctx の取り消しではグループごと止まる。ralph org merge・cleanup・regate の CLI は `signal.NotifyContext` の ctx で呼ぶので、Ctrl-C でも ctx が終わり、グループごと止まる。doctor だけが `checkMergeGuarantee(context.Background(), ...)`(`internal/cli/doctor.go:238`)で呼んでいた

## ACTION_REQUIRED

| # | Reviewer finding | Triage rationale | Affected file(s) |
|---|-------------------|------------------|-------------------|
| 1 | [P2] doctor の Merge guarantee は `context.Background()` で検査を呼ぶ。`ssh -G` が `Match exec` の命令で待っている間に Ctrl-C(SIGINT)や SIGTERM で doctor を止めると、ctx は終わらず、自分のプロセスグループで動く ssh とその子は端末の割り込みも期限の片付けも受けずに残る。doctor から signal を受ける ctx を渡し(Node の検査と同じ形)、CLI の段で割り込みのテストを足す | 本物の穴で、この PR が持ち込んだ。#340 より前は doctor の git と gh が doctor と同じプロセスグループで動き、Ctrl-C で一緒に止まっていた。ssh を自分のグループに移したので、ctx を終わらせる道のない doctor だけが残る。merge・cleanup・regate は `signal.NotifyContext` を使っており当たらない。直し方は既にある形(`doctorNodeSignalContext`、`internal/cli/doctor_node.go:45-53`)をそのまま使える | `internal/cli/doctor.go`(238 行)、`internal/cli/doctor_merge_guarantee.go`、そのテスト |

## WORTH_CONSIDERING

| # | Reviewer finding | Triage rationale | Affected file(s) |
|---|-------------------|------------------|-------------------|

## DISMISSED

| # | Reviewer finding | Dismissal reason | Category |
|---|-------------------|------------------|----------|

Categories: false-positive, already-addressed, style-preference, out-of-scope, context-aware-safe
