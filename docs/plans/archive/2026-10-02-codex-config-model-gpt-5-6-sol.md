# codex-config-model-gpt-5-6-sol

- Status: PR created (#198), awaiting CI and merge
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
- (self-review の M-1、L-1、L-2、L-4 で追加)両ファイルのコメントを codex の実際の挙動に合わせて書き直す: トップレベルの `model` の上のコメント(Codex 側の pipeline のエージェントもこのモデルを引き継ぐ、effort はユーザー設定、なければモデルの既定)と、profile の節のコメント(codex は project の `[profiles.*]` を捨てる、`/cross-review` は `--profile` を使わない)。表と値は残す。spec の 2 か所(`2026-05-07-codex-cli-parity.md:180`、`2026-08-01-org-runtime.md` の (e))に変更の注記を足す。
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
- `docs/specs/2026-05-07-codex-cli-parity.md`、`docs/specs/2026-08-01-org-runtime.md`(Slice B で追加)

## Design decisions

- 置き換え先は `gpt-5.6-sol`(メンテナの選択)。退役予告で codex が `gpt-5.5` の移行先として示したモデルなので、系統が近い。
- 3 か所とも同じモデルにする(profile ごとに分けない)。
- Critical forks: None(置き換え先はメンテナが選んだ)

## Acceptance criteria

- [x] AC-1: `.codex/config.toml` と `templates/base/.codex/config.toml` の `model` が 3 か所とも `gpt-5.6-sol` で、両ファイルが byte 一致。`./scripts/check-sync.sh` が green。
- [x] AC-2: 両ファイルが TOML として読める(`codex` の設定の読み込みか TOML の parser で確認)。TOML として読んだ中身を main と比べて、違うのは 3 つのモデル値だけ(コメントは Slice B と L-4 の修正で意図して書き直した。値、キー、表、並びは変えていない)。
- [x] AC-3: `git grep -n 'gpt-5\.5'` の結果に、project の codex の既定として書いた箇所が残っていない。残るのは履歴、spec の記録、任意の値として使う fixture だけ。分類を verify report に書く。
- [x] AC-4: `gpt-5.6-sol` への実際の要求が成功する(Codex plan advisory の MEDIUM)。いつもの codex の認証を使い、scratch のディレクトリから `command codex -m gpt-5.6-sol exec --sandbox read-only -o <file> '<小さな指示>' </dev/null` を、effort を指定しない場合(モデルの既定の `low`)と `-c model_reasoning_effort=max` の場合の 2 回送り、どちらも rc 0 で `-o` のファイルが空でない。認証がなくて送れなければ合格にせず、未解決のゲートとして PR に載せる。
- [x] AC-4b: この worktree で、既定、`--profile work`、`--profile review` の 3 通りの codex が `gpt-5.6-sol` を選ぶ(`-m` を付けずに実行し、選ばれたモデルを codex の出力か実行の記録で確かめる)。project の設定を codex に信頼させる必要があり、trust を得るために `~/.codex` の設定を書き換えることはしない。信頼されていなくて確かめられなければ、その状況を report に書き、PR に「未確認」として載せる。
- [x] AC-5: `RALPH_VERIFY_SCOPE=full ./scripts/run-verify.sh` が green。`internal/scaffold` と `internal/cli` のテスト(template の埋め込み、init、upgrade)が green。

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
- 2026-10-02 self-review(cycle 1、f7cb8c0c): CRITICAL 0 / HIGH 0 / MEDIUM 1 / LOW 3、merge 可。M-1: `.codex/config.toml` のコメントが、codex が project の `[profiles.*]` を捨てること(0.154.0 と 0.159.2 で確認)と合わない。「`codex --profile <name>` で切り替える」「`/cross-review` が使う」と書いてあるが、cross-review は `--profile` を渡さず、`--profile review` でも `danger-full-access` のまま。この差分より前からある。L-1: spec に project の codex のモデルを `gpt-5.5` と書いた文が 2 か所(`2026-08-01-org-runtime.md:23`、`2026-05-07-codex-cli-parity.md:180`)。L-2: `.codex/config.toml:12-14` の「no multi-agent」は古く(Codex 側の pipeline も `.codex/agents/` を使い、モデルを引き継ぐ)、既定の effort が `low` になることにも触れていない。L-3: `chore:` のコミットは changelog から除かれるので、effort の件は #186 にも残す。全件を in-cycle で扱う(orchestrator 判断): M-1 と L-2 は Slice B でコメントを codex の実際の挙動に合わせて書き直す(表と値は残す。表を消すかは挙動に関わる別の判断なので tech-debt に記録)、L-1 も Slice B で注記を足す、L-3 はマージ後に #186 にコメントする
- 2026-10-02 work: Slice B は implementer に委譲(dadc1b40、4 ファイル、+32 / -14、push 済み)。M-1: profile の節のコメントを「project の `[profiles.*]` は codex が捨てる(0.154.0 と 0.159.2 で警告を確認)、ユーザーレベルの設定に写したときだけ効く、この project で `--profile review` を付けても read-only にならない、`/cross-review` は `--profile` を使わず `-m` と `-c` を明示する」に書き直し、表と値は残した。L-2: 先頭のコメントを「Codex 側の pipeline のエージェント(`.codex/agents/` はモデルを指定しない)もこのモデルを引き継ぐ、effort は設定していないのでユーザー設定、なければモデルの既定(`gpt-5.6-sol` は low)」に書き直した。L-1: spec の 2 か所に変更の注記を足した。確認: 2 ファイルは byte 一致、TOML として読んだ中身は HEAD と同じ(orchestrator も確認)、差分はコメント行だけ、check-sync green、`tests/test-ralph-worktree.sh` 143 / 0、worktree での codex の実行は `model: gpt-5.6-sol`、main のチェックアウトは clean
- 2026-10-02 self-review addendum(cycle 1、Slice B、dbd3002b): CRITICAL 0 / HIGH 0 / MEDIUM 1 / LOW 1、merge 可。M-1、L-1、L-2 は意図どおり直っていて、新しいコメントの主張(警告の文と版、`.codex/agents/` がモデルを指定しないこと、`/cross-review` が `--profile` を使わないこと、`--profile review` が read-only にならないこと)は証拠と合う。M-2(この差分の外、前からある、security に関わる): `/cross-review` の `codex exec review` は `--sandbox` を渡さないので、trust 済みの project ではトップレベルの `sandbox_mode = "danger-full-access"` と `approval: never` で動く(reviewer の probe で確認)。レビューする差分の中の指示に codex が従えば、承認なしでコマンドを実行できる。`/plan` の advisory は `--sandbox read-only` を渡している。この PR の範囲外なので、tech-debt に記録し、follow-up の issue を起票する(`-c sandbox_mode=read-only` を足す案。read-only でも `-o` が書かれることを認証のある環境で確かめる必要がある)。L-4(任意): 「low for gpt-5.6-sol」は OpenAI 側の値で変わりうる
- 2026-10-02 work(inline、軽微な変更の例外): L-4 のコメントに「as of 2026-10-02」を足した(2 ファイル、各 1 行)。byte 一致、TOML として読んだ中身は直前と同じ、check-sync green、`tests/test-ralph-worktree.sh` 143 / 0、`RALPH_VERIFY_SCOPE=full ./scripts/run-verify.sh` green
- 2026-10-02 verify(cycle 1、57c24384): PASS、LOW 2。`RALPH_VERIFY_SCOPE=full ./scripts/run-static-verify.sh` green、既定の `run-static-verify.sh` は `unclassified:.codex/config.toml` の full。AC-1、AC-3、AC-4、AC-4b は pass(AC-4 と AC-4b は実装者の log を読んで確認。AC-4b の `--profile` の 2 通りはトップレベルの `model` の効果)、AC-2 は pass(TOML として読んだ中身を main と比べて違うのは 3 つのモデル値だけ)、AC-5 は静的な半分を確認。出荷する面に `gpt-5.5` は 0 件。新しいコメントの主張は 7 つのうち 6 つを確認、「エージェントがこのモデルを引き継ぐ」は `.codex/agents/` にモデルの指定がないことまで(実際の引き継ぎは未観測)。D-1 / D-2(LOW): plan のチェックボックスと、AC-2、Scope、Affected areas が Slice B を含んでいなかった(orchestrator が直した)。D-3: tech-debt に M-1(project の profile が捨てられる)と M-2(#197)の行がない(sync-docs)。D-4: `docs/specs/2026-05-07-codex-cli-parity.md:77` のユーザーストーリーが project の profile で切り替えられる前提になっている(sync-docs で注記を検討)
- 2026-10-02 test(cycle 1、a695e85a): PASS、LOW 1。既定の `run-test.sh` は `unclassified:.codex/config.toml` の full。shell 32 スイート、Go 8 パッケージ green(scaffold、cli、upgrade は 834 PASS)。fresh scaffold の `.codex/config.toml` は template と byte 一致で 3 つの値が新しく、その scaffold の `check-template.sh` と `ralph doctor --strict` も通る。upgrade: main のビルドで作った project にこの branch の `ralph upgrade` をかけると、手を加えていない `.codex/config.toml` は新しい内容に置き換わり(2 回目は何もしない)、手を加えたものは上書きされず drift として報告される(rc 3)。#185 の検知は新しい内容を HEAD にしても 16 / 16。worktree の codex は `model: gpt-5.6-sol`、main は clean。G-1(LOW、以前から): 配布する `.codex/config.toml` の中身を読むテストがなく、TOML を壊しても `ralph doctor` 以外では捕まらない。今回このファイルのコメントを大きく書き直したので、Slice C で塞ぐ(orchestrator 判断)
- 2026-10-02 work: Slice C は implementer に委譲(21291931、テストだけの変更、+54、push 済み)。`TestTemplateBaseCodexTomlFilesParse` を足した: template の `.codex/config.toml` と `.codex/agents/*.toml` を TOML として読み(agents が 0 件なら失敗)、config のトップレベルの `model` が空でない文字列であることを確かめる(モデル名は固定しない)。root の `.codex/` は check-sync の byte 一致に任せる。red: config の構文エラー、トップレベルの `model` の削除、agents の構文エラー、`model` を表の中だけに置く、`model = ""` で落ちる。範囲外の気づき: `TestTemplateBaseCodexAssetsExist` の一覧に `implementer.toml` がない(別件)。orchestrator も新しいテストを確認
- 2026-10-02 sync-docs(cycle 1、8005657f): ずれは 1 件(verify の D-4): `docs/specs/2026-05-07-codex-cli-parity.md:77` のユーザーストーリーに、codex が project の profile を捨てることの注記を足した(本文は書き換えていない)。ほかの文書(recipes、`.codex/README.md`、`.codex/AGENTS.override.md`、model-routing、README、AGENTS)にずれはない。tech-debt に 3 行: project の `[profiles.*]` が捨てられる件(表を消すか例として残すかは未決)、`/cross-review` の codex reviewer が `danger-full-access` で動く件(#197)、`TestTemplateBaseCodexAssetsExist` の一覧に `implementer.toml` がない件。check-sync、check-skill-sync、purity green
- 2026-10-03 cross-review(cycle 1、HEAD 8005657f、watchdog の 1 行、`codex rc=0`、`-o` 214 バイト): 指摘 0 件(Case C)。差分は plan・報告・insight を除いて 6 ファイル、+97 / -22 で 500 行未満なので、walkthrough は作らない。PR に進む

## Progress checklist

- [x] Plan reviewed
- [x] Branch created
- [x] Implementation started
- [x] Review artifact created
- [x] Verification artifact created
- [x] Test artifact created
- [x] PR created (#198)

## Readiness checklist

- [x] 変更する 3 か所と、影響を受ける経路(project の codex、Codex 側の pipeline のエージェント、`ralph upgrade`)を特定した
- [x] critical fork なし(置き換え先はメンテナが選んだ)
- [x] Codex plan advisory(MEDIUM 1、対応案で plan を更新)
