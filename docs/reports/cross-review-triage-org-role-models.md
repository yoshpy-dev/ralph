# Cross-review triage report: org-role-models

- Date: 2026-11-05
- Plan: docs/plans/active/2026-11-05-org-role-models.md
- Base branch: main
- Driver: claude
- Reviewer: codex
- Triager: Claude Code (main context)
- Reviewer status: complete
- Self-review cross-ref: yes
- Cycle: 1/2
- Total reviewer findings: 1
- After triage: ACTION_REQUIRED=1, WORTH_CONSIDERING=0, DISMISSED=0

## Triage context

- Active plan: docs/plans/active/2026-11-05-org-role-models.md
- Self-review report: docs/reports/self-review-2026-11-05-org-role-models.md(Run 2、Merge: yes。この指摘は扱っていない)
- Verify report: docs/reports/verify-2026-11-05-org-role-models.md(pass。この指摘は扱っていない)
- Implementation context summary: codex(gpt-6-astra、xhigh)を `-c sandbox_mode=read-only exec review --ignore-rules --base main` で動かした。rc=0 で、`-o` のファイルは空でなく、前後の git の状態は同じだった。codex は read-only のサンドボックスで Go の一時ディレクトリを作れず、テストは打てなかったと書いている。`ResolveRoleModel`(`internal/org/role_models.go`)は S2 で足し、S3 から spawn・start・start --plan・director start の CLI が `org.Spawn` の前に呼ぶ。計画の D4 と D9 は、解決の関数が `--model` の文字と、解決した driver が `driver_pool` にあるかを確かめると決めていた。

## ACTION_REQUIRED

| # | Reviewer finding | Triage rationale | Affected file(s) |
|---|-------------------|------------------|-------------------|
| 1 | [P2] `driver_pool` の検査が冪等の打ち直しより前に走る。codex で立てた座席の `driver_pool` をあとで `["claude"]` に狭め、同じ spawn を `--driver codex --model ...` で打ち直すと、`ResolveRoleModel` が断る。CLI は `Org.Spawn` の前にこの関数を呼ぶので、ロックの下の冪等の return を通らない。leader の start の打ち直しも同じ。新しい起動には検査を残し、既にある座席の判断のあとに回すべき | 実際の後退。`internal/org/spawn.go:800-812` は「既に立っている座席の打ち直しは envelope の検査より前に既存の座席を返す。冪等の no-op は検査で失敗してはならない」(AC-3)と決めていて、`ValidateSpawnEnvelope` はその return のあとで driver_pool と model の文字を見る。この PR の前は、CLI は `--driver` をそのまま渡し、`--model` を省いたときだけプールを引いた。`--model` の文字の検査も同じく return の前に移っている。spawn と start は `ResolveRoleModel` で driver とモデルを決めるだけにし、`driver_pool` と文字の検査は `Spawn`(`ValidateSpawnEnvelope`)に任せる。director start は座席ではなく冪等の打ち直しもないので、自分の検査(`checkDirectorStart`)で `driver_pool` も見る。役割のエントリを名指す文は保つ | `internal/org/role_models.go`、`internal/org/envelope.go`、`internal/org/director_start.go`、`internal/cli/org.go`、`internal/cli/org_director_headless.go` とテスト |

## WORTH_CONSIDERING

| # | Reviewer finding | Triage rationale | Affected file(s) |
|---|-------------------|------------------|-------------------|

## DISMISSED

| # | Reviewer finding | Dismissal reason | Category |
|---|-------------------|------------------|----------|

Categories: false-positive, already-addressed, style-preference, out-of-scope, context-aware-safe
