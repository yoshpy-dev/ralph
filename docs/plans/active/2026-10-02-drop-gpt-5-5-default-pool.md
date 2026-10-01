# drop-gpt-5-5-default-pool

- Status: Draft
- Owner: Claude Code
- Date: 2026-10-02
- Related request: メンテナの指示(2026-10-02)「#156 model_pool については、gpt-5.5 を既定から外してください」。#156 の観測では、2 回目(2026-09-25)に `gpt-5.5` へ退役予告(移行先 `gpt-5.6-sol`、2026-10-14)が付き、3 回目(2026-10-02)には予告が消えて説明が "Legacy coding model." のままだった
- Related issue: 156
- Type: chore
- Branch: chore/drop-gpt-5-5-default-pool

## Objective

既定の `[org].model_pool` から codex の `gpt-5.5` を外す。既定のプールは claude 4 つ(`fable`、`opus`、`sonnet`、`haiku`)と codex 4 つ(`gpt-6-astra`、`gpt-5.6-sol`、`gpt-5.6-terra`、`gpt-5.6-luna`)の 8 エントリになる。

## 調査で確認したこと(2026-10-02、main 7dd6911c)

- 既定の値を持つ 3 面: `internal/config/config.go:150`(`Default()`)、`templates/base/ralph.toml:38`、`scripts/ralph-config.sh:57`(`RALPH_ORG_MODEL_POOL`)。一致は `internal/config/defaults_sync_test.go` が検査する。`templates/base/scripts/ralph-config.sh` は root との byte 一致を `scripts/check-sync.sh` が検査する。
- 手順は `docs/specs/2026-08-01-org-runtime.md` の「運用ノート: 既定 codex スラッグの更新手順」(a)〜(c) に従う。追従させる面:
  - `.claude/skills/org/SKILL.md:88` の「既定の model_pool」表。`scripts/sync-skills.sh` で `.agents/skills/` を作り直し、`templates/base/.claude/skills/` と `templates/base/.agents/skills/` にも同じものを置く(4 面)。
  - 同じ spec の 2026-09-16 改訂注記 (c)(11 行目)。
  - 既定値をハードコードするテスト: `internal/config/config_test.go:131`、`:218`(`!= 9`)、`:743`。
- 既定のプールではなく、任意の値として `gpt-5.5` を使っているテストの fixture は多い(`internal/cli/doctor_org_test.go`、`internal/cli/cli_test.go`、`internal/cli/org_test.go:1019-1022`、`tests/test-ralph-worktree.sh` など)。これらは既定とは関係ないので変えない。
- `.codex/config.toml`(+ template)の `model = "gpt-5.5"`(15、50、55 行目)は codex CLI 自体の既定のモデルで、`[org].model_pool` とは別の設定。今回の指示の対象外にする(Non-goals)。

## Scope

- 3 面から `codex:gpt-5.5` のエントリを外す: `internal/config/config.go`、`templates/base/ralph.toml`、`scripts/ralph-config.sh` と `templates/base/scripts/ralph-config.sh`(byte 一致)。
- `.claude/skills/org/SKILL.md` の「既定の model_pool」表から `gpt-5.5` の行を外す。`scripts/sync-skills.sh` で `.agents/skills/org/SKILL.md` を作り直し、template の 2 面にも同じものを置く。
- 既定値をハードコードするテストを追従させる(`internal/config/config_test.go`)。
- `docs/specs/2026-08-01-org-runtime.md`: 改訂注記 (c) の既定の列挙を直す。運用ノートに 1 行足す: 2026-10-02 にメンテナの判断で `gpt-5.5` を既定から外した。理由は退役予告が付いたことと "Legacy" の表示で、(d) の「2 週間連続の消失」を待たない判断だった。
- 最後に `git grep -n 'gpt-5\.5'` で残りを掃く。履歴(`docs/insights/events/`、`docs/reports/`、`docs/plans/archive/`、`docs/evidence/`)と、既定と関係ない fixture は除く。

## Non-goals

- `.codex/config.toml`(+ template)の `model = "gpt-5.5"` の変更。これは codex CLI の既定のモデルで、model_pool とは別の判断になる。
- `gpt-5.5` を任意の値として使うテストの fixture の書き換え。
- 下流の `ralph.toml`(seed-once)の書き換え。`ralph upgrade` の advisory diff で見えるようになり、直すのは下流の運用者(spec の運用ノート (c))。
- 他のスラッグの追加や入れ替え。
- リリース(#186)。

## Assumptions

- `gpt-5.5` を外しても、codex の既定(プール先頭へのフォールバック先)は `gpt-6-astra` のまま変わらない。
- `[org.roles]` の既定は空(`Roles: map[string][]string{}`)なので、既定の role から `gpt-5.5` を参照している箇所はない。

## Affected areas

- `internal/config/config.go`、`internal/config/config_test.go`
- `templates/base/ralph.toml`
- `scripts/ralph-config.sh`、`templates/base/scripts/ralph-config.sh`
- `.claude/skills/org/SKILL.md`、`.agents/skills/org/SKILL.md`、`templates/base/.claude/skills/org/SKILL.md`、`templates/base/.agents/skills/org/SKILL.md`
- `docs/specs/2026-08-01-org-runtime.md`

## Design decisions

- `gpt-5.5` は外すだけで、後継は足さない。移行先の `gpt-5.6-sol` はすでに既定のプールにある。
- 消失の判定基準 (d) は、cache からスラッグが消えた場合の基準。今回は、退役予告と "Legacy" の表示を理由にしたメンテナの判断で外す。spec にはその旨を記録する。
- Critical forks: None

## Acceptance criteria

- [ ] AC-1: `config.Default()` の `Org.ModelPool` が、claude 4 つ + codex 4 つの 8 エントリで、`gpt-5.5` を含まない(テストあり)。
- [ ] AC-2: `templates/base/ralph.toml` と `scripts/ralph-config.sh`(+ template)の既定も同じ 8 エントリで、`defaults_sync_test.go` が green、root と template の `ralph-config.sh` が byte 一致。
- [ ] AC-3: org skill の「既定の model_pool」表(4 面)に `gpt-5.5` の行がなく、`./scripts/check-skill-sync.sh` と `./scripts/check-sync.sh` が green。
- [ ] AC-4: `git grep -n 'gpt-5\.5'` の結果に、既定のプールとしての記述が残っていない(残るのは履歴、spec の運用ノートの記録、既定と関係ない fixture、`.codex/config.toml` だけ)。残った箇所の分類を verify report に書く。
- [ ] AC-5: 既定の設定で `ralph doctor` の「Org codex model slugs」が pass で、`4 codex model_pool slug(s)` を出す(main のビルドで確認)。
- [ ] AC-6: `RALPH_VERIFY_SCOPE=full ./scripts/run-verify.sh` と `go test ./... -count=1` が green。

## Implementation outline

1. Slice A(implementer、sonnet): Scope を 1 コミットで。
2. pipeline: self-review → verify → test → sync-docs → cross-review → PR(`Refs #156`。#156 は観測と手順の issue なので、閉じるかどうかはメンテナの判断に残す)。

## Verify plan

- Static analysis checks: `RALPH_VERIFY_SCOPE=full ./scripts/run-static-verify.sh`、`./scripts/check-sync.sh`、`./scripts/check-skill-sync.sh`、`go vet ./...`。
- Spec compliance criteria to confirm: AC-1〜AC-6 を該当行と実行結果で確認。
- Documentation drift to check: spec の改訂注記と運用ノート、org skill の表とプール先頭の記述、`templates/base/ralph.toml` のコメント、README / recipes に既定のプールの列挙がないこと。
- Evidence to capture: `docs/evidence/verify-*.log`、`git grep` の残りの分類、doctor の出力。

## Test plan

- Unit tests: `go test ./internal/config/... ./internal/org/... ./internal/cli/... -count=1`。
- Integration tests: `RALPH_VERIFY_SCOPE=full ./scripts/run-test.sh`、main のビルドでの `ralph doctor`。
- Regression tests: `defaults_sync_test.go`(3 面の一致)。
- Edge cases: `model_pool` を書かず `driver_pool = ["codex"]` だけの `ralph.toml` で、絞った既定が codex 4 つになる。`[org.roles]` で `gpt-5.5` を指定した `ralph.toml`(既定のプールのまま)が、プールにないモデルとして検証エラーになる(既存の挙動。メッセージを確認する)。
- Evidence to capture: test report。

## Risks and mitigations

- 下流で `[org.roles]` に `gpt-5.5` を書き、`model_pool` を省略している project は、バイナリを更新すると検証エラーになる。エラーは「プールにないモデル」を名指しするので、直し方は分かる。release notes(#186)に 1 行書くよう申し送る。
- 既定のプールを前提にした、別の場所のテストが落ちる: `go test ./...` で拾う。

## Rollout or rollback notes

既定値と文書の変更だけ。問題があれば 1 コミットを revert する。

## Open questions

- `.codex/config.toml` の `model = "gpt-5.5"` を変えるかどうか(今回は対象外。メンテナに確認する)。

## Deviation notes

## Progress checklist

- [x] Plan reviewed
- [x] Branch created
- [ ] Implementation started
- [ ] Review artifact created
- [ ] Verification artifact created
- [ ] Test artifact created
- [ ] PR created

## Readiness checklist

- [x] 既定値の 3 面と追従させる面を特定した
- [x] critical fork なし
- [ ] Codex plan advisory
