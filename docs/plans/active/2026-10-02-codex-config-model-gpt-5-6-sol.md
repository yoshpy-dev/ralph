# codex-config-model-gpt-5-6-sol

- Status: Draft
- Owner: Claude Code
- Date: 2026-10-02
- Related request: メンテナの指示(2026-10-02)。`.codex/config.toml`(と template)の `model = "gpt-5.5"` を変える。置き換え先は `gpt-5.6-sol`(AskUserQuestion で選択。9/25 の退役予告で codex が `gpt-5.5` の移行先として示したモデル)。PR #196 で既定の `[org].model_pool` から `gpt-5.5` を外したのに続く変更
- Related issue: 156
- Type: chore
- Branch: chore/codex-config-model-gpt-5-6-sol

## Objective

project の codex 設定(`.codex/config.toml` と `templates/base/.codex/config.toml`)が使うモデルを、`gpt-5.5` から `gpt-5.6-sol` に変える。

## 調査で確認したこと(2026-10-02、main 47345820)

- `gpt-5.5` は両ファイルの 3 か所にある: トップレベルの `model`(15 行目)、`[profiles.work]`(50 行目)、`[profiles.review]`(55 行目)。root と template は byte 一致で、`scripts/check-sync.sh` が検査する。
- `.codex/agents/*.toml` はどれも `model` を指定していない。Codex が driver のときは、pipeline のエージェントも project の `model` を引き継ぐ。
- ralph の skill(`/plan` の Codex plan advisory、`/cross-review` の codex reviewer)は `-m "${RALPH_CODEX_REVIEWER_MODEL:-gpt-6-astra}"` を明示するので、この設定を使わない(`docs/recipes/codex-setup.md` 85〜100 行)。
- `ralph init` は `.codex/config.toml` を core として扱う(`internal/cli/init.go` の `ownerForScaffoldPath`)。下流には次の `ralph upgrade` で届く。下流で手を加えていれば drift として報告される。
- codex の cache(2026-10-02、モデルごとの `supported_reasoning_levels` と `default_reasoning_level` だけを読んだ):
  - `gpt-5.5`: 既定は `medium`、対応は low〜xhigh。
  - `gpt-5.6-sol`: 既定は `low`、対応は low〜ultra(`max` を含む)。
  - 以前、project の `gpt-5.5` とユーザー設定の `model_reasoning_effort = "max"` を組み合わせて 400 になったことがあった(`max` が `gpt-5.5` に非対応だったため)。`gpt-5.6-sol` ではこの組み合わせが通る。
- 任意の値として `gpt-5.5` を使う fixture: `internal/cli/cli_test.go:43`(mock の FS)、`scripts/verify.local.sh` と `tests/test-hook-wiring.sh` の codex 設定の fixture、`tests/test-ralph-worktree.sh` の #185 の検知の fixture(自分で HEAD を作る)。どれも追跡しているファイルを読まない。
- `docs/specs/2026-05-07-codex-cli-parity.md` の F-6 に「`model = "gpt-5.5"` を既定値とする」とある(当時の決定)。

## Scope

- `.codex/config.toml` と `templates/base/.codex/config.toml` の 3 か所を `gpt-5.6-sol` に変える(byte 一致を保つ)。
- `docs/specs/2026-05-07-codex-cli-parity.md` の F-6 に、2026-10-02 に `gpt-5.6-sol` へ変えたことを短く書き足す(#156)。
- 最後に `git grep -n 'gpt-5\.5'` で、project の codex の既定として書いた箇所が残っていないか確かめる。

## Non-goals

- `model_reasoning_effort` を project の設定で固定すること。固定すると、ユーザー設定で選んだ effort を上書きしてしまう(リスクに記録する)。
- `gpt-5.5` を任意の値として使うテストの fixture の書き換え。
- `RALPH_CODEX_REVIEWER_MODEL`(`gpt-6-astra`)の変更。
- `.codex/config.toml` のほかの設定の変更。

## Assumptions

- `gpt-5.6-sol` は、利用者の codex で選べる(2026-10-02 の cache に存在し、`upgrade` は null)。

## Affected areas

- `.codex/config.toml`、`templates/base/.codex/config.toml`
- `docs/specs/2026-05-07-codex-cli-parity.md`

## Design decisions

- 置き換え先は `gpt-5.6-sol`(メンテナの選択)。退役予告で codex が `gpt-5.5` の移行先として示したモデルなので、系統が近い。
- 3 か所とも同じモデルにする(profile ごとに分けない)。
- Critical forks: None(置き換え先はメンテナが選んだ)

## Acceptance criteria

- [ ] AC-1: `.codex/config.toml` と `templates/base/.codex/config.toml` の `model` が 3 か所とも `gpt-5.6-sol` で、両ファイルが byte 一致。`./scripts/check-sync.sh` が green。
- [ ] AC-2: 両ファイルが TOML として読める(`codex` の設定の読み込みか TOML の parser で確認)。値のほかに、コメント、空行、並びを変えていない(`git diff` で 3 行の置換だけ)。
- [ ] AC-3: `git grep -n 'gpt-5\.5'` の結果に、project の codex の既定として書いた箇所が残っていない。残るのは履歴、spec の記録、任意の値として使う fixture だけ。分類を verify report に書く。
- [ ] AC-4: `gpt-5.6-sol` への実際の要求が成功する(Codex plan advisory の MEDIUM)。いつもの codex の認証を使い、scratch のディレクトリから `command codex -m gpt-5.6-sol exec --sandbox read-only -o <file> '<小さな指示>' </dev/null` を、effort を指定しない場合(モデルの既定の `low`)と `-c model_reasoning_effort=max` の場合の 2 回送り、どちらも rc 0 で `-o` のファイルが空でない。認証がなくて送れなければ合格にせず、未解決のゲートとして PR に載せる。
- [ ] AC-4b: この worktree で、既定、`--profile work`、`--profile review` の 3 通りの codex が `gpt-5.6-sol` を選ぶ(`-m` を付けずに実行し、選ばれたモデルを codex の出力か実行の記録で確かめる)。project の設定を codex に信頼させる必要があり、trust を得るために `~/.codex` の設定を書き換えることはしない。信頼されていなくて確かめられなければ、その状況を report に書き、PR に「未確認」として載せる。
- [ ] AC-5: `RALPH_VERIFY_SCOPE=full ./scripts/run-verify.sh` が green。`internal/scaffold` と `internal/cli` のテスト(template の埋め込み、init、upgrade)が green。

## Implementation outline

1. Slice A(implementer、sonnet): Scope を 1 コミットで。
2. pipeline: self-review → verify → test → sync-docs → cross-review → PR(`Refs #156`)。

## Verify plan

- Static analysis checks: `./scripts/check-sync.sh`、`RALPH_VERIFY_SCOPE=full ./scripts/run-static-verify.sh`、TOML の parse。
- Spec compliance criteria to confirm: AC-1〜AC-5。
- Documentation drift to check: `docs/recipes/codex-setup.md`、`.codex/README.md`(+ template)、`.claude/rules/ralph/model-routing.md`、README で project の codex モデルに触れている箇所。
- Evidence to capture: `git grep` の分類、AC-4 の確認結果。

## Test plan

- Unit tests: `go test ./internal/scaffold/... ./internal/cli/... -count=1`。
- Integration tests: `RALPH_VERIFY_SCOPE=full ./scripts/run-test.sh`。AC-4 と AC-4b の実際の codex の実行(`~/.codex` は書き換えない。codex がふだんどおり書く session の記録と cache は除く)。fresh scaffold(`go run ./cmd/ralph init --yes <tmp>`)の `.codex/config.toml` が `gpt-5.6-sol` になること。
- Regression tests: `tests/test-ralph-worktree.sh`(#185 の書き換えの検知は HEAD の内容と比べるだけなので、モデルの値に依らないことを確かめる)。
- Edge cases: 書き換えの検知(#185)の形の差分が、新しい値でも従来どおり見分けられること。
- Evidence to capture: test report。

## Risks and mitigations

- `gpt-5.6-sol` の既定の reasoning level は `low`(`gpt-5.5` は `medium`)。project の設定は effort を指定していないので、ユーザー設定でも effort を指定していない環境では、project の codex と Codex 側の pipeline のエージェントが `low` で動く。effort を project で固定するとユーザー設定を上書きするので、今回は固定せず、PR と release の説明で触れる。
- 下流で `.codex/config.toml` に手を加えている project は、`ralph upgrade` で drift として報告される(core の扱い)。従来どおりの挙動で、新しいことではない。
- メンテナのチェックアウトの `.codex/config.toml` は、原因不明の書き換え(#185)が起きることがある。PR の差分で変わるのは、値の 3 行だけにする。

## Rollout or rollback notes

設定の値と文書だけ。問題があれば 1 コミットを revert する。

## Open questions

なし。

## Deviation notes

- 2026-10-02 plan: Codex plan advisory(gpt-6-astra、xhigh、watchdog の 1 行、`codex rc=0`、`-o` 1279 バイト)は MEDIUM 1: AC-4 は実際に動かす確認を省けてしまい、モデルが選ばれることを見るだけでは要求の成功が分からない。以前 400 になった effort `max` の組み合わせも対象外だった。ユーザー決定: 対応案で plan を更新。AC-4 を「`gpt-5.6-sol` に effort 未指定と `max` の 2 通りで要求が成功する」に、AC-4b を「worktree で既定、work、review の 3 通りが `gpt-5.6-sol` を選ぶ(trust のために `~/.codex` は書き換えない)」に分けた
- 2026-10-02 work: Slice A は implementer(sonnet)に委譲(0c45e78d、3 ファイル、+7 / -7、push 済み)。両ファイルの 3 行を `gpt-5.6-sol` に置き換え(byte 一致、tomllib で 3 つの値を確認)、spec の F-6 に注記を足した。AC-5: `go test ./internal/scaffold/... ./internal/cli/...`、`tests/test-ralph-worktree.sh` 143 / 0、`RALPH_VERIFY_SCOPE=full ./scripts/run-verify.sh` green、fresh scaffold も新しい値。AC-4: scratch から `command codex -m gpt-5.6-sol exec` で、effort 未指定(header は `reasoning effort: low`)と `max` の 2 通りとも rc 0、`-o` は `ok`、400 なし。AC-4b: この worktree で `-m` なしの既定、`--profile work`、`--profile review` の 3 通りとも header が `model: gpt-5.6-sol`(project の設定は trust 済みで読まれている)。対照として scratch の repo の外で `-m` なしにすると、利用者の既定のモデルになった。重要な観測: codex 0.154.0 は project の `.codex/config.toml` の `[profiles.*]` を読まずに捨てる(`Ignored unsupported project-local config keys ...: profiles`。orchestrator もログで確認)。`--profile` で `gpt-5.6-sol` になったのはトップレベルの `model` の効果で、profile の値は効いていない。この変更より前からある挙動なので、tech-debt に記録し PR に書く(sync-docs で profile が効くと書いた文書がないかも確かめる)。codex の実行の前後で、worktree の 2 つの設定の内容は同じで、main のチェックアウトは clean のまま(`.codex/config.toml` の mtime も変わらず)。`~/.codex` の設定と trust は変えていない。orchestrator も差分と cmp を確認

## Progress checklist

- [x] Plan reviewed
- [x] Branch created
- [x] Implementation started
- [ ] Review artifact created
- [ ] Verification artifact created
- [ ] Test artifact created
- [ ] PR created

## Readiness checklist

- [x] 変更する 3 か所と、影響を受ける経路(project の codex、Codex 側の pipeline のエージェント、`ralph upgrade`)を特定した
- [x] critical fork なし(置き換え先はメンテナが選んだ)
- [x] Codex plan advisory(MEDIUM 1、対応案で plan を更新)
