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
- org skill の「既定の model_pool」節(4 面)に、移行と復旧の段落を足す(Codex plan advisory の HIGH)。
  - 移行(バイナリを更新する前): `model_pool` を書かずに `[org.roles]` で `gpt-5.5` を指定している project は、`[org].model_pool` に `gpt-5.5` を含めて明示するか、role から外す。
  - 復旧(更新した後に `ralph org` の verb が `[org.roles].<role> references model "gpt-5.5" not present in [org].model_pool` で止まった場合): `ralph.toml` に同じ修正を入れるか、直したコピーを `--config` で渡す。state dir は設定ファイルの場所で変わらない(`--state-dir`、環境変数、git の toplevel で決まる)ので、`ralph org status` / `stop` は同じ座席を扱える。
- `internal/cli` に復旧の経路の回帰テストを足す。`model_pool` を書かずに `[org.roles]` で `gpt-5.5` を指定した `ralph.toml` で、`ralph org status` が上のエラーで失敗する。同じ state dir に座席がある状態で、直した設定を `--config` で渡すと、`status` がその座席を表示する。
- 最後に `git grep -n 'gpt-5\.5'` で残りを掃く。履歴(`docs/insights/events/`、`docs/reports/`、`docs/plans/archive/`、`docs/evidence/`)と、既定と関係ない fixture は除く。

## Non-goals

- `.codex/config.toml`(+ template)の `model = "gpt-5.5"` の変更。これは codex CLI の既定のモデルで、model_pool とは別の判断になる。
- `gpt-5.5` を任意の値として使うテストの fixture の書き換え。
- 下流の `ralph.toml`(seed-once)の書き換え。`ralph upgrade` の advisory diff で見えるようになり、直すのは下流の運用者(spec の運用ノート (c))。
- 他のスラッグの追加や入れ替え。
- リリース(#186)。release notes に載せる 1 行(移行と復旧の要点)は、PR の本文に申し送りとして書く。
- teardown 系の verb(`stop`、`status`、`disband`)が設定の検証で止まらないようにする変更。範囲が広がるので、今回は文書とテストで扱う。

## Assumptions

- `gpt-5.5` を外しても、codex の既定(プール先頭へのフォールバック先)は `gpt-6-astra` のまま変わらない。
- `[org.roles]` の既定は空(`Roles: map[string][]string{}`)なので、既定の role から `gpt-5.5` を参照している箇所はない。

## Affected areas

- `internal/config/config.go`、`internal/config/config_test.go`
- `templates/base/ralph.toml`
- `scripts/ralph-config.sh`、`templates/base/scripts/ralph-config.sh`
- `.claude/skills/org/SKILL.md`、`.agents/skills/org/SKILL.md`、`templates/base/.claude/skills/org/SKILL.md`、`templates/base/.agents/skills/org/SKILL.md`
- `docs/specs/2026-08-01-org-runtime.md`
- `internal/cli/` の org のテスト(復旧の経路)

## Design decisions

- `gpt-5.5` は外すだけで、後継は足さない。移行先の `gpt-5.6-sol` はすでに既定のプールにある。
- 消失の判定基準 (d) は、cache からスラッグが消えた場合の基準。今回は、退役予告と "Legacy" の表示を理由にしたメンテナの判断で外す。spec にはその旨を記録する。
- Critical forks: None

## Acceptance criteria

- [ ] AC-1: `config.Default()` の `Org.ModelPool` が、claude 4 つ + codex 4 つの 8 エントリで、`gpt-5.5` を含まない(テストあり)。
- [ ] AC-2: `templates/base/ralph.toml` と `scripts/ralph-config.sh`(+ template)の既定も同じ 8 エントリで、`defaults_sync_test.go` が green、root と template の `ralph-config.sh` が byte 一致。
- [ ] AC-3: org skill の「既定の model_pool」表(4 面)に `gpt-5.5` の行がなく、`./scripts/check-skill-sync.sh` と `./scripts/check-sync.sh` が green。
- [ ] AC-4: `git grep -n 'gpt-5\.5'` の結果に、既定のプールとしての記述が残っていない(残るのは履歴、spec の運用ノートの記録、既定と関係ない fixture、`.codex/config.toml` だけ)。残った箇所の分類を verify report に書く。
- [ ] AC-5: 作業 branch の HEAD のビルドで、`model_pool` を書かない(既定のプールを使う)scratch の project と、残す 4 つの codex スラッグだけを入れた固定の cache(`CODEX_HOME` を scratch に向ける)を使い、`ralph doctor` の「Org codex model slugs」が pass で `4 codex model_pool slug(s)` を出す。利用者の実際の cache での確認は補足として report に書くだけで、合否には使わない(Codex plan advisory の MEDIUM)。
- [ ] AC-7: org skill の「既定の model_pool」節(4 面)に移行と復旧の段落があり、`internal/cli` の回帰テストで、`model_pool` を省略して `[org.roles]` で `gpt-5.5` を指定した設定では `ralph org status` がモデル名を挙げて失敗し、直した設定を `--config` で渡すと同じ state dir の座席を表示することを確かめる。
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

- 下流で `[org.roles]` に `gpt-5.5` を書き、`model_pool` を省略している project は、バイナリを更新すると設定の検証エラーになる。`ralph org` の verb はどれも設定を読むので、`stop` / `status` / `disband` も止まり、動いている座席を止められないまま課金が続きうる(Codex plan advisory の HIGH)。上流のコミットを revert しても、更新済みのバイナリは戻らない。対策: org skill に更新前の移行と更新後の復旧を書き、復旧の経路をテストで確かめ、release notes に 1 行載せるよう申し送る。`ralph init` が作る `ralph.toml` は `model_pool` を明示していて `gpt-5.5` も含むので、そのまま使っている project は影響を受けない。
- 既定のプールを前提にした、別の場所のテストが落ちる: `go test ./...` で拾う。

## Rollout or rollback notes

既定値と文書の変更だけ。問題があれば 1 コミットを revert する。

## Open questions

- `.codex/config.toml` の `model = "gpt-5.5"` を変えるかどうか(今回は対象外。メンテナに確認する)。

## Deviation notes

- 2026-10-02 plan: Codex plan advisory(gpt-6-astra、xhigh、watchdog の 1 行、`codex rc=0`、`-o` 2231 バイト)は HIGH 1 / MEDIUM 1。HIGH: `model_pool` を省略して `[org.roles]` で `gpt-5.5` を指定した設定は、更新後に `stop` / `status` / `disband` まで止める(`internal/cli/org.go:115` の `resolveOrgConfig` → `internal/config/config.go:298` の検証。コードで確認)。MEDIUM: AC-5 が利用者の環境に左右される。ユーザー決定: 対応案で plan を更新。org skill に移行と復旧の段落、`internal/cli` に復旧の経路の回帰テスト(AC-7)、release notes への申し送りを足し、AC-5 を作業 branch のビルドと固定の cache で確かめる形にした。teardown 系の verb の検証を緩める変更は入れない
- 2026-10-02 work: Slice A は implementer(sonnet)に委譲(22df1937、11 ファイル、+177 / -14、push 済み)。3 面と template の `ralph-config.sh` から `gpt-5.5` を外し、既定は 8 エントリになった。`config_test.go` の 3 か所を追従させ、`TestDefault_Org` に `gpt-5.5` がないことの確認を足した。新しいテストは 2 つ: `TestLoad_OmittedModelPoolWithRolesNamingDroppedDefault_Errors` と `TestOrgStatus_ConfigFlagRecoversAfterDefaultPoolDropsRoleModel`(壊れた設定では status がモデル名を挙げて失敗し、直したコピーを `--config` で渡すと同じ state dir の座席を表示する。直し方 2 通りのサブテスト)。org skill(4 面)に移行と復旧の段落を足した(state dir の決まり方 `--state-dir` → `RALPH_ORG_STATE_DIR` → git の toplevel はコードで確認)。spec は改訂注記 (c) を直し、運用ノートに (e) を足した。逸脱 1 件: (d) の記録項目「既定 5 スラッグの有無」を「既定の codex スラッグの有無」に直した(個数が古くなるため)。`git grep` の残りは、履歴、移行と復旧の文、spec の記録、任意の値として使う fixture、`.codex/config.toml` の系統(Non-goal)だけ。red: template の `ralph.toml` にだけ戻すと `TestDefaultsLockStep`、`--config` を無視させると AC-7 のテスト、`Default()` に戻すと 5 つのテストが落ちる。AC-5 の probe(作業 branch のビルド、`model_pool` なしの scratch の project、4 スラッグだけの固定の cache): `pass — 4 codex model_pool slug(s) present`。`go test ./...` と `RALPH_VERIFY_SCOPE=full ./scripts/run-verify.sh` green。orchestrator も差分を読み、config と AC-7 のテストを確認

## Progress checklist

- [x] Plan reviewed
- [x] Branch created
- [x] Implementation started
- [ ] Review artifact created
- [ ] Verification artifact created
- [ ] Test artifact created
- [ ] PR created

## Readiness checklist

- [x] 既定値の 3 面と追従させる面を特定した
- [x] critical fork なし
- [x] Codex plan advisory(HIGH 1 / MEDIUM 1、対応案で plan を更新)
