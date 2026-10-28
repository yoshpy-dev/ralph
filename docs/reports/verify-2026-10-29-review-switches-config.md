# Verify report: review-switches-config

- Date: 2026-10-29
- Plan: `docs/plans/active/2026-10-29-review-switches-config.md`(issue #265、AC1〜AC6、D1〜D6、Progress の S1〜S4・M1 の Deviation・L3 の決定・F1)
- Verifier: verifier サブエージェント(`/verify`、パイプラインの cycle 1)
- Scope: `git diff da9d1b3a...HEAD`(HEAD d017007c、18 コミット、34 ファイル、+1459/-203)。self-review の run 2(Merge: OK、MEDIUM 0、L6 は残す判断、N1 は 94ed9717 で直した)のあとの全体。
- Evidence: `docs/evidence/verify-2026-10-29-review-switches-config.log`(gitignore の対象でコミットしない。`run-static-verify.sh` の全出力、3 つの検査の個別実行、`go run` と build した binary の probe、base の binary の probe、tomllib の照合)。wrapper 自身の保存先は `docs/evidence/verify-2026-10-28-202001.log`(UTC の日付)。
- Insight event: `docs/insights/events/2026-10-28-review-switches-config.jsonl` に 1 行(phase verify、cycle 1、verdict pass、low 1)。file 名は UTC の日付で、self-review の 2 行と同じ file。

## Spec compliance

| Acceptance criterion | Status | Evidence |
| --- | --- | --- |
| AC1: 5 つのキーと既定値(false、false、false、false、2)、`max_cross_review_cycles` が 0 以下で `Load` がエラー | 満たす(テストの実行は /test) | 型は `internal/config/config.go:35`(`StandardConfig`)と `:561`(`OrgReviewConfig`)、`Config` と `OrgConfig` への組み込みは `:18` と `:115`。既定値は `:585` と `:628`。検証は `:860`(`< 1` でエラー)。テストは `internal/config/config_test.go:1838`(既定値)、`:1860`(Load の 5 通り)、`:1906`(0 と -1 を拒み 1 を受ける)、`:1942`(知らないキーを無視)。build した binary で `max_cross_review_cycles = 0` は exit 3、`[org.review].max_cross_review_cycles must be >= 1, got 0` を確かめた。 |
| AC2: テンプレートに 5 キー、lock-step と照合の表が新しいキーも見る、`RALPH_STANDARD_MAX_PIPELINE_CYCLES` との照合 | 満たす(テストの実行は /test) | テンプレートは `templates/base/ralph.toml:28-30` と `:255-258`。probe は `internal/config/defaults_sync_test.go:147`(`[org.review]`)と `:195`(`standardTemplateProbe`)、`TestDefaultsLockStep` の `standard` の行は `:252`。照合の表は `:531`(`org.review` の 3 行を追加)と `:580`(`[standard]` の 2 行)で、どちらも `checkEachKeyIsComparedOnBothSides`(`:466`)がキーの削除ごとに 1 行の drift を求める(`:518`)。シェルとの照合は `:297`。tomllib で同じ照合を移して打ち、テンプレートの 5 キーが既定と等しく、シェルの既定が `2` で、各キーを消すと `<missing>` になることを確かめた(evidence 3f)。 |
| AC3: ルートの `ralph.toml` で 4 つが true、`check-sync.sh` が通る | 満たす | `ralph.toml:9-15`。tomllib で読むと、ルートのキーは 4 つの実施有無だけで、すべて true、テンプレートにないキーはない(evidence 3f)。`./scripts/check-sync.sh` は rc 0、`KNOWN_DIFF ralph.toml`、`DRIFTED: 0`。Scope の「テンプレートの写し」は Progress の Deviation(M1)で「4 つだけ」に変わった。AC3 の文はどちらの形でも成り立つ。 |
| AC4: `ralph config get` の値、既定、終了コード 2 と 3、D6 の回帰、help の約束 | 満たす | 動詞は `internal/cli/config.go:25` と `:46`、root への登録は `internal/cli/root.go:37`。終了コードは `internal/cli/exitcode.go:31` と `:33`。help の約束は `internal/cli/config.go:57-66`。build した binary での実測は下の「Observational checks」。テストは `internal/cli/config_test.go` の 9 本(既定、値あり、`--config`、exit 2 の 15 通り、exit 3 の 5 通り、D6 の回帰、cobra の誤り、help、`ExitCode`)。 |
| AC5: claude と codex の座席(leader を含む)に `RALPH_PRIMARY_CLI=<driver>`、`--env` のない herdr では env なしで warn | 満たす(テストの実行は /test。実物の herdr では見ていない) | `internal/org/spawn.go:998` が `driver.PrimaryCLIEntry(p.Driver)` を `TabCreate` の `extraEnv` で渡す。driver は先に `permissionArgsForDriver` が `claude` と `codex` に絞る(`internal/org/spawn.go:680` と `:845`)。leader は `internal/org/feature.go:246` で同じ `Spawn` を通る。director は `internal/org/director_start.go:1125` で `extraEnv` なし。driver 側は `internal/org/driver/herdr.go:249-251` と `paneenv.go:41`、`:50`、`:65`。テストは `internal/org/spawn_test.go:3764`(claude と codex の worker と leader の 4 通り)、`director_start_test.go:263`(director は `extraEnv` なし)、`driver/herdr_test.go:578`、`:595`、`:665`(順序と warning の文)、`internal/cli/org_test.go:431`、`internal/cli/org_runtime_env_test.go:191` と `:202`(stub の herdr での argv と warning)。 |
| AC6: `go test` 4 パッケージ、`check-sync.sh`、`check-skill-sync.sh`、同期の検査 | 一部(静的な部分のみ) | `check-sync.sh` と `check-skill-sync.sh` はどちらも rc 0。テストの diff には `+func Test` の行が 19 本あり(うち `TestOrgTemplateDrift_EachKeyIsComparedOnBothSides` は既存のテストを共通の関数に切り出して書き直したもの)、ほかに既存のテスト 5 本(`TestDefaultsLockStep`、`TestOrgSpawn_HappyPath_EventSequenceReceiptAndWorkspaceReuse`、`TestOrgVerbs_HerdrWithoutEnvFlagWarnsOnStderr`、`TestStartDirector_LaunchesInAWorkspaceOfItsOwn`、`TestHerdr_TabCreate_PaneEnvWarningNamesEachEntrysConsequence`)が変わった。`run-static-verify.sh` の `go vet ./...` は、テストのファイルも含めて compile を通した。`go test` の実行は /test の担当で、ここでは打っていない。 |

issue #265 の受け入れ条件との対応:

| issue の受け入れ条件 | Status | Evidence |
| --- | --- | --- |
| 4 つの実施有無と org の上限があり、既定値がテストで固定 | 満たす | AC1 と同じ。 |
| この repo の `ralph.toml` で 4 つが on | 満たす | AC3 と同じ。`go run ./cmd/ralph config get` が 4 つとも `true`。 |
| 設定を読む動詞があり、未設定と古い binary では off を返すか off として扱える | 満たす | 未設定は exit 0 で `false`(ファイルなし、表だけでキーなしの両方を実測)。古い binary(da9d1b3a を build)は exit 1、stderr が `unknown command "config" for "ralph"` で、help の約束(1 と `unknown command "config"` だけを off)に当たる。 |
| 座席の pane に `RALPH_PRIMARY_CLI=<driver>`(claude と codex) | 満たす(テスト。実物の herdr は未確認) | AC5 と同じ。 |
| 既定値の同期の検査が通る | 静的に確認 | 照合のコードと照合先の値(シェルの `2`、テンプレートの 5 キー)は確かめた。テストの実行は /test。 |

記録された外れ: issue の「古い binary では off として扱うことを skill に書く」は、この PR では skill に書かず、動詞の help(`internal/cli/config.go:57-66`)とテンプレートのコメント(`templates/base/ralph.toml:19-27`)に書いた。計画の D4 がこの外れと理由(#266 が skill を設定を読む形に変える変更で書く)を記録している。PR の本文にも書くと D4 にあるが、PR はまだないので未確認。

## Static analysis

| Command | Result | Notes |
| --- | --- | --- |
| `./scripts/run-static-verify.sh` | pass(rc 0) | 言語の scope は `full fallback (unclassified:ralph.toml)`。ルートの `ralph.toml` が言語に分類されないため全体の検査に落ち、golang の検査が走った。gofmt は `ok`、go vet は出力なし、golangci-lint 2.15.0 は `0 issues.`、staticcheck 2026.2.2 は出力なし(`command -v staticcheck` で入っていることを確かめた)。shellcheck、`sh -n`、settings.json の jq、Codex の hook の検査、check-sync、check-pipeline-sync、check-skill-sync、check-template-purity、tech-debt の参照も通った。branch の secret scan は `da9d1b3a..d017007c` で clean。 |
| `./scripts/check-sync.sh`(単独) | pass(rc 0) | IDENTICAL 172、DRIFTED 0、ROOT_ONLY 0、TEMPLATE_ONLY 10、KNOWN_DIFF 6(`ralph.toml` を含む)。 |
| `./scripts/check-skill-sync.sh`(単独) | pass(rc 0) | `14 skill(s) in lock-step`。 |
| `./scripts/check-template-purity.sh`(単独) | pass(rc 0) | `no meta-repo-specific references found in templates`。 |
| `git diff --check da9d1b3a...HEAD` | pass(rc 0) | 空白の誤りなし。 |
| `bash -n scripts/check-sync.sh` | pass(rc 0) | `KNOWN_DIFFS` に足した行の構文。 |

## Documentation drift

| Doc / contract | In sync? | Notes |
| --- | --- | --- |
| pane の環境を列挙する文書(座席には `RALPH_PRIMARY_CLI`、director にはなし) | yes | 座席: `/org` skill の 4 つの写し(`.claude/skills/org/SKILL.md:27`、ほか 3 つは `cmp` で一致)、README の director の段落(座席の pane は `RALPH_PRIMARY_CLI=<driver>` も持ち、director は持たない、と書く)、`docs/recipes/isolated-herdr-server.md:241`(テンプレートの写しと `cmp` で一致)、spec の FR-7 の注(`docs/specs/2026-10-07-org-multi-org-director.md:129`)。director: `/director` skill(`:230`、`:288`、`:295`)と `docs/recipes/director-launch-cmd.md`(`:133`、`:254-267`)は 2 つの値だけで、director の pane の話なので正しい。`git-commit-strategy.md:53` は `RALPH_WIP_CHECKPOINT` だけの話。 |
| README の command 表と AGENTS.md の地図 | yes | `README.md:128` に `ralph config get <key>` の行。`AGENTS.md` の `internal/cli/` の列挙に `config`。build した binary の `ralph --help` にも `config  Read ralph.toml settings` が出る。 |
| 終了コードの約束(テンプレート、help、README、`/org` skill) | yes(矛盾なし。下の V2 と V3) | help(`internal/cli/config.go:57-66`)と README の行は、0 は値、1 は `unknown command "config"` のときだけ off、2 と 3 は止まって報告、で一致する。テンプレートの `[standard]` のコメント(`:19-27`)も同じ向きで、`[org.review]` のコメント(`:251-253`)はそれを参照する。`/org` skill は終了コードを書かない(V3)。 |
| 上限の数え方 | yes | 「最初の 1 回を含む全部の回数、2 は最初の review と直したあとの review 1 回」が、テンプレート(`:246-250`)、`config.go` の `OrgReviewConfig` の doc、`/org` skill(`:49`)、spec の注に入り、`post-implementation-pipeline.md` の「2 total runs」と同じ数え方。help と README は上限の単位を書かない(キーの例として名前を出すだけ)。 |
| skill がもう設定を読むと書いていないか | yes | テンプレートは「The skills start reading these switches in a later release.」(`:27`)と「They start reading them in a later release.」(`:253`)、`/org` skill は「座席が読み始めるのは後の版から。」(`:51`)、spec の注は「この版では設定を読む座席はまだない」。`config.go` の doc は「they start to in issue #266」。`/plan` と `/cross-review` の skill は変わっていない(diff の skill は `/org` の 4 つの写しだけ)。 |
| check-sync の `KNOWN_DIFFS` の注とルートの実物 | yes | `scripts/check-sync.sh:112-116` は「root holds only the four review switches of [standard] and [org.review], turned on … leaves every other key to the built-in default」。ルートは 4 キーだけで、その値はすべて true(tomllib)。ルートの先頭の注(`ralph.toml:1-7`)が名指しする 3 つの読み手(`ralph config get`、`ralph doctor`、`ralph org` の verb)は、どれも `config.Load` で読む(`internal/cli/config.go:73`、`doctor.go:93`、`org.go:331` と `:621`)。 |
| 計画の Scope、R2、AC のチェック | 遅れあり(情報。V4) | Scope の「テンプレートの写しに 4 つを true」と R2 の「テンプレートの写し」は古いままで、Progress の Deviation(M1)が訂正を持つ。AC1〜AC6 のチェックは `[ ]` のまま。 |

## Observational checks

`go run ./cmd/ralph config get`(worktree のルートで打った。evidence 3a):

| 打ったもの | stdout | rc |
| --- | --- | --- |
| `standard.plan_advisory` | `true` | 0 |
| `standard.cross_review` | `true` | 0 |
| `org.review.plan_advisory` | `true` | 0 |
| `org.review.cross_review` | `true` | 0 |
| `org.review.max_cross_review_cycles` | `2` | 0 |
| `org.max_seats` | `5`(`Default()` の値、`internal/config/config.go:602`) | 0 |
| `--config templates/base/ralph.toml` で 4 つの実施有無 | どれも `false` | 0 |
| `--config templates/base/ralph.toml` で `org.review.max_cross_review_cycles` | `2` | 0 |

build した binary(`go build -o <scratchpad>/ralph ./cmd/ralph`、HEAD d017007c。`go run` は子の終了コードを 1 として返すので、終了コードはこちらで見た。evidence 3b):

| 場合 | rc | stdout / stderr |
| --- | --- | --- |
| ルートで `standard.nope`、`nope` | 2 | stdout 空、`not a key of one value in ralph.toml: no such key` |
| ルートで `org.review`(表)、`org.driver_pool`(配列) | 2 | `it is a table`、`it is an array` |
| ルートで `standard.plan_advisory.x` | 2 | `standard.plan_advisory is a boolean` |
| `[standard] cross_review = true` と `[org] max_seats = 0` の同居で `standard.cross_review` | 3 | stdout 空、`[org].max_seats must be >= 1, got 0` |
| 同じ file で知らないキー `standard.nope` | 3 | Load がキーの照合より先なので 3 |
| parse できない `ralph.toml` | 3 | `toml: expected character ]` |
| `max_cross_review_cycles = 0` | 3 | `[org.review].max_cross_review_cycles must be >= 1, got 0` |
| `ralph.toml` がディレクトリ | 3 | `read ralph.toml: is a directory` |
| `ralph.toml` のない空のディレクトリ | 0 | `false` と `2`、stderr は空 |
| `[standard]` と `[org.review]` の表だけでキーなし | 0 | `false` と `2` |
| 存在しない `--config` のパス | 0 | `false`(D3 と `config.Load` の規則どおり) |
| `ralph config gte x` | 1 | `unknown command "gte" for "ralph config"`(`unknown command "config"` を含まない) |
| `ralph config get`(キーなし) | 1 | `accepts 1 arg(s), received 0` |
| `ralph config`(引数なし) | 0 | group の help |
| `ralph config get --help` | 0 | D6 の約束の 4 行(0、1、2、3)がある(evidence 3c) |

古い binary の側(evidence 3d と 3e):

- da9d1b3a を `git archive` で scratchpad に出して build した binary は、`ralph config get standard.plan_advisory` で rc 1、stdout は空、stderr は `unknown command "config" for "ralph"`。`--config` を `get` の後ろに置いても `config` の後ろに置いても同じ。help の 1 の行が読む文字列を含む。
- 同じ da9d1b3a の `config.Load` で、新しいテンプレートとルートの `ralph.toml` を読むと、どちらも `err=<nil>`。計画の Rollback の「古い binary は新しい節を知らないキーとして無視する」が成り立つ。

## Coverage gaps

- テストは打っていない(`go test` と `run-test.sh` は /test の担当)。AC1、AC2、AC5、AC6 の「テストが通る」は /test の結果で決まる。
- 変異は打ち直していない。Progress の S1(テンプレートのキーの削除、`[standard].cross_review` の削除、既定の 2 → 3、検証の `< 1` → `< 0`)、S2(`-overlay` の変異 4 つ)、S3(座席の項を消すと 4 件落ちる)の結果は implementer の記録で、こちらでは見ていない。
- 実物の herdr の pane の環境は見ていない。AC5 の根拠は fake と stub の herdr のテストと、`spawn.go:998` の配線の読みだけ。
- PR の本文(release のノート、D4 の外れ)はまだないので未確認。
- V1(LOW): `ralph` が PATH にないときの扱いが、計画と動詞の help で食い違う。計画の Assumptions(`docs/plans/active/2026-10-29-review-switches-config.md:53`)は「ないときは古い binary と同じく off として扱う」と書く。help(`internal/cli/config.go:57-66`)は「exit 1 で `unknown command "config"` のときだけ off、ほかの exit 1 は止まる」と書き、PATH にないとき(シェルの exit 127)には触れない。テンプレートのコメントと README も触れない。この PR には設定を読むものがないので今は害がない。#266 の計画で、127 を off に倒すか止まるかを決め、help かテンプレートのコメントに 1 行足すとよい。
- V2(情報): テンプレートの `[standard]` のコメント(`templates/base/ralph.toml:22-26`)は、help より約束が狭い。exit 2 を「an unknown key」とだけ書き(help は表と配列も含む)、「off として扱ったことを記録する」と「ほかの exit 1 は止まる」を書かない。向きは同じで矛盾はない。4 つの実施有無は表でも配列でもないので、読み手が迷う場面は今はない。
- V3(情報): `/org` skill は終了コードの約束を書かない。D4(skill への組み込みは #266、#267、#269)と合っていて、ほかの文書とも食い違わない。
- V4(情報): 計画の AC1〜AC6 のチェックが `[ ]` のまま。Scope と R2 の「テンプレートの写し」は Progress の Deviation(M1)で訂正されている。
- V5(情報): `ralph.toml` は seed(`internal/cli/init.go:474`)なので、`ralph upgrade` は既存の project の `ralph.toml` に `[standard]` と `[org.review]` を足さない。既定の off は表がなくても効くので動きは変わらないが、#266 のあとで on にしたい project は自分で表を書く。release のノートの「`[standard]` の 2 つを true にする」の項に、表ごと書き足すことを添えると迷いにくい(`[org.approval]` の release のときと同じ書き方)。

所見の数: LOW 1(V1)、情報 4(V2〜V5)。CRITICAL、HIGH、MEDIUM はない。

最も安く確信を上げる追加の確認: /test で `go test ./internal/config/ ./internal/cli/ ./internal/org/ ./internal/org/driver/` を通すこと。とくに `TestDefaultsLockStep`、`TestStandardTemplateDrift_EachKeyIsComparedOnBothSides`、`TestOrgSpawn_PaneGetsRalphPrimaryCLIOfItsDriver`、`TestOrgVerbs_HerdrWithoutEnvFlagWarnsOnStderr` が AC2 と AC5 を直接固定する。

## Verdict

pass

- Verified: AC1、AC3、AC4 は静的な読みと build した binary の実測で確かめた。AC2 はテンプレート、シェル、照合のコードを読み、tomllib に移した照合で確かめた。AC5 は配線とテストの存在を確かめた。静的検査はすべて rc 0。文書の pane の環境、command 表、地図、終了コードの約束、上限の数え方は食い違いがない。古い binary が `unknown command "config"` で exit 1 になることと、新しい seed を古い `config.Load` が読めることも実測した。
- Partially verified: AC6(`check-sync.sh` と `check-skill-sync.sh` は通った。`go test` は /test)。AC5 の実物の herdr での pane の環境。
- Not verified: テストの実行、変異の打ち直し、PR の本文の release のノートと D4 の外れの記載。
