# review-switches-config

- Status: Approved
- Approved: 2026-10-29 sha256:cc42f13b59d0
- Owner: Claude Code
- Date: 2026-10-29
- Related request: 計画助言と cross-review の実施有無を ralph.toml で設定する(標準フローと org)
- Related issue: #265
- Type: feat
- Branch: feat/review-switches-config

## Objective

`ralph.toml` に、計画助言と cross-review の実施有無の設定 4 つ(標準フローの計画助言、標準フローの cross-review、org の leader の計画助言、org の reviewer の cross-review)と、org の cross-review の直しの回数の上限を足す。skill がその値を読む動詞 `ralph config get <key>` を足す。`ralph org spawn` が座席の driver を `RALPH_PRIMARY_CLI=<driver>` として pane に渡す。この repo のルートに `ralph.toml` を置き、4 つを on にする。

設定を読んで実際に飛ばすのは後続の issue の担当で、この PR は土台だけを作る(標準フローの skill は #266、leader は #267、reviewer は #269)。

## Scope

- `internal/config/config.go`
  - 新しい節 `[standard]` に `plan_advisory`(bool、既定 false)と `cross_review`(bool、既定 false)を足す。#276 が同じ節に標準フローのほかの設定を足す。
  - 新しい表 `[org.review]` に `plan_advisory`(bool、既定 false、leader の計画助言)、`cross_review`(bool、既定 false、reviewer の cross-review)、`max_cross_review_cycles`(int、既定 2、`RALPH_STANDARD_MAX_PIPELINE_CYCLES` の既定に合わせる)を足す。
  - 検証: `max_cross_review_cycles` は 1 以上。
- `templates/base/ralph.toml`: 2 つの節を、既定値(off、off、off、off、2)と説明のコメントつきで書く。
- `internal/config/defaults_sync_test.go`: `[org.review]` の 3 キーを `orgTemplateProbe` と照合の表(`TestOrgTemplateDrift_EachKeyIsComparedOnBothSides`)に足す。`[standard]` の 2 キーも、テンプレートと `Default()` の値とキーの有無を照合する(同じ pointer の probe の形)。`shell_and_skills` のサブテストに、`Default()` の `[org.review].max_cross_review_cycles` と `scripts/ralph-config.sh` の `RALPH_STANDARD_MAX_PIPELINE_CYCLES` の既定が等しいことの照合を足す(issue の受け入れ条件 5、consult の指摘)。
- 動詞 `ralph config get <key>`(`internal/cli/config.go`、root への登録)
  - `<key>` は toml の点つきの名前(例: `standard.plan_advisory`、`org.review.max_cross_review_cycles`)。
  - `--config` があればそのファイル、なければ `./ralph.toml` を `config.Load` で読む(ないときは既定、読めないときはエラー。#274 の F2 と同じ規則)。
  - 値を 1 行で stdout に出す(bool は `true`/`false`、int は 10 進、文字列はそのまま)。終了コードを分ける(D6): 表や知らないキーは 2、`ralph.toml` を読めない・parse できない・検証で落ちるときは 3。この動詞は 1 を使わない(cobra の引数の誤りを除く)。
  - help に、skill が使う約束を書く: 0 なら値を使う。終了コード 1 で stderr が `unknown command "config"` を含むとき(動詞のない古い binary)だけ off として扱い、そのことを記録する。2 と 3 は止まって人に報告する(黙って off にしない)。
- spawn の pane の環境
  - `ralph org spawn`(leader を含むすべての座席)が、その座席の driver を `RALPH_PRIMARY_CLI=<claude|codex>` として herdr の `tab create --env` で渡す。
  - herdr に `--env` がないときの扱いは `RALPH_WIP_CHECKPOINT` と同じ(env なしで作り、stderr に warn。`paneEnvConsequence` に 1 項足す)。
- この repo のルートの `ralph.toml`: `templates/base/ralph.toml` の写しに、4 つの実施有無を true にしたもの。`scripts/check-sync.sh` の `KNOWN_DIFFS` に足す。#264 も同じファイルを作るので、先に入った側が作り、後の側は merge で両方の変更を残す。
- 文書
  - `README.md` の command 表に `ralph config get`。`AGENTS.md` の地図の `internal/cli/` の subcommand の列挙に `config`。
  - `/org` skill(4 つの写し)と、pane の環境を列挙している文書(`README.md`、`docs/specs/2026-10-07-org-multi-org-director.md`、`docs/recipes/isolated-herdr-server.md` など、実装のときに `git grep` で洗い出す)に `RALPH_PRIMARY_CLI` を足す。
  - 設定の説明は `templates/base/ralph.toml` のコメントに書く。
- PR の本文の release のノート: 2 つの節と動詞を足したこと。この版では設定を読む skill はまだなく、標準フローの動きは変わらない。#266 が入ると、既定の off で計画助言と cross-review が飛ばされる。#266 の skill は新しい binary の `ralph config get` を使うので、PATH の `ralph` が古いと(この repo で release 済みの v6.0.0 を使っている場合も)設定を on にしても off として扱われる。新しい binary(`go install ./cmd/ralph` か次の release)が要る。

## Non-goals

- `/plan` と `/cross-review` の skill が設定を読んで飛ばすこと(#266)。leader の計画助言(#267)と reviewer の cross-review(#269)。
- 標準フローのほかの設定を `ralph.toml` に移すことと、環境変数での上書きをやめること(#276)。
- `ralph config set` などの書き込み。
- headless の director の pane に `RALPH_PRIMARY_CLI` を渡すこと(director は座席ではなく、計画助言も cross-review もしない)。
- `scripts/ralph-config.sh` の変更。

## Assumptions

- 4 つの既定 off、この repo では on、org の上限の既定 2 は、2026-10-27 にユーザーが決めた(issue #265)。蒸し返さない。
- `pick_reviewer`(`scripts/xreview-helpers.sh`)は `RALPH_PRIMARY_CLI` を driver として読む。pane に渡れば、座席の中の判別が正しくなる。
- skill は Bash から `ralph config get` を打てる(座席も標準フローも `ralph` が PATH にある前提。ないときは古い binary と同じく off として扱う)。

## Affected areas

- `internal/config/config.go`、`internal/config/config_test.go`、`internal/config/defaults_sync_test.go`
- `templates/base/ralph.toml`、ルートの `ralph.toml`(新規)、`scripts/check-sync.sh`
- `internal/cli/config.go`(新規)、`internal/cli/root.go`、`internal/cli/exitcode.go`、`cmd/ralph/main.go`(コメント)、`internal/cli` のテスト
- `internal/org/spawn.go`、`internal/org/driver/herdr.go`、`internal/org/driver/paneenv.go` とそのテスト
- `README.md`、`AGENTS.md`、`/org` skill の 4 つの写し、pane の環境を書いた文書

## Visual review

None (設定のキー、読み出しの動詞、pane の環境変数を 1 つ足す変更で、図にする仕組みの変化がない)

## Design decisions

- D1 設定の置き場所と名前(consult-plan-reviewswitch が同意): `[standard]` に `plan_advisory`・`cross_review`、`[org.review]` に `plan_advisory`・`cross_review`・`max_cross_review_cycles`。`[standard]` は #276 が標準フローのほかの設定を足す節にする。org の 3 つは表 `[org.review]` にまとめ、`[org]` の直下を増やさない。上限の名前は、#276 が `[standard]` に足す `max_pipeline_cycles` と語順をそろえて `max_cross_review_cycles` にした(consult の指摘。seed に残ってからは変えにくい)。go-toml/v2 の `Unmarshal` は未知のキーを無視し、internal に `DisallowUnknownFields` の使用はないので、新しい seed を古い binary が読んでも落ちない(consult が確認)。
- D2 動詞は汎用の `ralph config get <点つきの名前>` にする。`Config` を `config.Load` で読み、toml のタグをたどって値を出す。#276 が足すキーも、動詞を変えずに読める。表やリストは今は扱わず、終了コード 2 にする(必要になった版で足す)。
- D3 動詞の読み込みの規則は `resolveOrgConfig` と同じにする(`--config`、なければ `./ralph.toml`、ないときは既定、読めないときはエラー)。読めないときに既定を返すと、off を書いた file が既定で読まれる。既定が off なので害は小さいが、#274 の F2 と規則をそろえる。
- D4 skill の側の約束(D6)は、動詞の help と `ralph.toml` のコメントに書き、skill への組み込みは #266 で行う(この PR では skill を変えない)。issue #265 の「古い binary では off として扱うことを skill に書く」から外れる。#266 は「`/plan` と `/cross-review` が #265 の設定を読み、off なら飛ばす」担当で、skill を設定を読む形に変えるのと同じ変更で書くほうが、読まない skill に約束だけを書くより食い違いが出ない。同じセッションが #266 を続けて担当する。PR の本文にもこの外れを書く。
- D6 終了コードを分ける(Codex の計画助言、MEDIUM): 古い binary(終了コード 1、`unknown command "config"`)だけを off として扱い、知らないキー(2)と、読めない・正しくない `ralph.toml`(3)は止まって報告する。「0 以外は off」にすると、標準フローと関係のない設定の誤り(例: `[org].max_seats = 0`)で、人が on にした review が黙って外れる。
- D5 `RALPH_PRIMARY_CLI` は座席ごとの値なので、herdr の `PaneEnv`(全 pane に共通の固定の列)とは別に、spawn が座席の driver から作って `TabCreate` に渡す。`--env` がない herdr の扱いは既存の 2 つと同じにする。
- Critical forks: D1 と D2 は public な契約(`ralph.toml` のキーと動詞)で、後で名前を変えると下流の seed ファイルに残る。ユーザーの指示どおり consult に聞き、consult-plan-reviewswitch の「進めてよい」と上限の名前の指摘に従って選んだ(ユーザーには聞かない)。

## Acceptance criteria

- [ ] AC1: `ralph.toml` に `[standard].plan_advisory`、`[standard].cross_review`、`[org.review].plan_advisory`、`[org.review].cross_review`、`[org.review].max_cross_review_cycles` があり、既定値(false、false、false、false、2)が `internal/config` のテストで固定されている。`max_cross_review_cycles` が 0 以下なら `Load` がエラーを返す。
- [ ] AC2: `templates/base/ralph.toml` が 5 つのキーを既定値で書き、`TestDefaultsLockStep` と照合の表のテストが新しいキーも見る(テンプレートからキーを消すと落ちる)。`Default()` の `max_cross_review_cycles` と `RALPH_STANDARD_MAX_PIPELINE_CYCLES` の既定がずれると `TestDefaultsLockStep` が落ちる。
- [ ] AC3: この repo のルートの `ralph.toml` で 4 つが true になっている。`./scripts/check-sync.sh` が通る(`KNOWN_DIFFS` に `ralph.toml`)。
- [ ] AC4: `ralph config get <key>` が値を出す。キーのない `ralph.toml` と `ralph.toml` のない場合は既定(false、2)を出す。知らないキーと表は終了コード 2、読めない・parse できない・検証で落ちる `ralph.toml` は終了コード 3。`standard.cross_review = true` と正しくない `[org]` の設定(例: `max_seats = 0`)が同じ `ralph.toml` にあるとき、`ralph config get standard.cross_review` は 0 で `false` を出すことも 1 で終わることもなく、3 で終わる(回帰テスト)。help に D6 の約束(1 と `unknown command "config"` だけを off、2 と 3 は止まる)がある。
- [ ] AC5: `ralph org spawn` が、claude の座席には `RALPH_PRIMARY_CLI=claude`、codex の座席には `RALPH_PRIMARY_CLI=codex` を `tab create --env` で渡す(leader を含む)。`--env` のない herdr では env なしで作り、stderr に warn する。`internal/org` のテストで確かめる。
- [ ] AC6: `go test ./internal/config/ ./internal/cli/ ./internal/org/ ./internal/org/driver/`、`./scripts/check-sync.sh`、`./scripts/check-skill-sync.sh` が通る。`internal/config/defaults_sync_test.go` の同期の検査が通る。

## Implementation outline

1. S1 設定と雛形: `internal/config/config.go` に `[standard]` と `[org.review]` の型、既定値、検証を足す。`templates/base/ralph.toml` に 2 つの節を書く。`internal/config` のテスト(既定値、round trip、上限の検証)と、`defaults_sync_test.go` の probe・照合の表・`[standard]` の照合を足す。 — verify: `test -z "$(gofmt -l internal/config)" && go vet ./internal/config/ && GOFLAGS="-timeout=30m $(go env GOFLAGS)" go test ./internal/config/`
2. S2 読み出しの動詞: `internal/cli/config.go` に `ralph config get`、root に登録、終了コード 2 と 3 を `cli.ExitCode` に足す(`internal/cli/exitcode.go`、`cmd/ralph/main.go` のコメント)、テスト(既定、値あり、知らないキー、表、読めない file、正しくない `[org]` と true の `standard.cross_review` の同居、`--config`)。 — verify: `test -z "$(gofmt -l internal/cli)" && go vet ./internal/cli/ && GOFLAGS="-timeout=30m $(go env GOFLAGS)" go test ./internal/cli/ -run 'TestConfig'`
3. S3 pane の `RALPH_PRIMARY_CLI`: `internal/org/driver` に座席ごとの env を `TabCreate` に足す道と `paneEnvConsequence` の 1 項、`internal/org/spawn.go` が座席の driver から env を作って渡す。テスト(claude と codex の座席、leader、`--env` のない herdr)。 — verify: `test -z "$(gofmt -l internal/org)" && go vet ./internal/org/ ./internal/org/driver/ && GOFLAGS="-timeout=30m $(go env GOFLAGS)" go test ./internal/org/driver/ && GOFLAGS="-timeout=30m $(go env GOFLAGS)" go test ./internal/org/ -run 'Spawn|PaneEnv|TabCreate'`
4. S4 この repo の `ralph.toml` と文書: ルートの `ralph.toml`(テンプレートの写しに 4 つを true)、`scripts/check-sync.sh` の `KNOWN_DIFFS`、`README.md`、`AGENTS.md`、`/org` skill(`.claude/skills/` と `templates/base/.claude/skills/` を直し `./scripts/sync-skills.sh`)、pane の環境を書いた文書。 — verify: `./scripts/check-sync.sh && ./scripts/check-skill-sync.sh && ./scripts/check-template-purity.sh && go run ./cmd/ralph config get standard.plan_advisory | grep -qx true`

## Verify plan

- Static analysis checks: `./scripts/run-static-verify.sh`(gofmt、go vet、golangci-lint、staticcheck、shellcheck、check-sync、check-skill-sync、check-template-purity)。
- Spec compliance criteria to confirm: AC1〜AC6。issue #265 の受け入れ条件 5 つ。
- Documentation drift to check: pane の環境を列挙した文書が `RALPH_PRIMARY_CLI` を含むこと。`README.md` と `AGENTS.md` の command の列挙。`templates/base/ralph.toml` のコメントが動詞と off の約束を説明していること。
- Evidence to capture: `run-static-verify.sh` の出力、`ralph config get` の手での実行の出力。

## Test plan

- Unit tests: 新しい既定値、`max_cross_review_cycles` の検証(0、負)、round trip。`ralph config get` の各場合。spawn の env(claude、codex、leader)。
- Integration tests: `ralph config get` をルートの `ralph.toml` に対して打つ(S4 の検証)。
- Regression tests: 照合の表のテストと `TestDefaultsLockStep` に新しいキーが入ること。既存の pane の env(`AGMSG_SELF_NAME=off`、`RALPH_WIP_CHECKPOINT=off`)が変わらないこと。
- Edge cases: キーのない `ralph.toml`、`ralph.toml` なし、読めない `ralph.toml`、知らないキー、表の名前(`org.review`)、`--env` のない herdr。
- Evidence to capture: `./scripts/run-test.sh` の出力。

## Risks and mitigations

- R1: 設定と動詞の名前は public な契約で、後から変えると下流の `ralph.toml` に古い名前が残る。軽減: D1 と D2 を consult に聞いて決め、#266・#267・#269・#276 の issue の文と合うかを確かめる。
- R2: ルートの `ralph.toml` を足すと、この repo の `ralph org` と `ralph doctor` がそれを読む。テンプレートの写しなので値は既定と同じで、違うのは 4 つの実施有無だけ。軽減: 4 つはまだ誰も読まない(#266 以降)。
- R3: #264 も同じルートの `ralph.toml` を作る。軽減: 先に入った側が作り、後の側は merge で両方の変更を残す(ユーザーの決まり)。
- R4: 座席ごとの env を足すため、`TabCreate` の呼び方が変わる。headless の director の pane は変えない。軽減: S3 のテストで、座席と director の pane の env をそれぞれ確かめる。
- R5: `internal/org` と `internal/cli` のテストは時間がかかり、ほかのセッションと CPU を分け合う。軽減: スライスの検証は狭く、全体は /test で 1 回。30 分の timeout か既知の不安定なテストで落ちたら 1 回だけ打ち直して記録する。

## Rollout or rollback notes

- release のノートの項(PR の本文): `[standard]` と `[org.review]` の 5 つの設定と `ralph config get` を足した。この版ではまだどの skill も設定を読まないので、動きは変わらない。#266 が入ると、標準フローの計画助言と cross-review は既定の off で飛ばされるので、今の動きを続けたい project は `[standard]` の 2 つを true にする。#266 の skill は新しい binary が要る(古い binary では off として扱う)。`ralph org spawn` は座席の pane に `RALPH_PRIMARY_CLI` を渡す。
- 戻すときはこの PR を revert する。テンプレートの新しい節は seed なので、`ralph init` で作った project に残っても、古い binary は知らないキーとして無視する(go-toml/v2 の `Unmarshal` は未知のキーでエラーにしない。consult が確認、S1 のテストでも確かめる)。

## Open questions

- なし

## Progress checklist

- [x] Plan reviewed — consult-plan-reviewswitch: 進めてよい(D1〜D5 に同意)。指摘 3 つを反映: 上限の名前を `max_cross_review_cycles` に、`max_cross_review_cycles` と `RALPH_STANDARD_MAX_PIPELINE_CYCLES` の照合を AC2 と Scope に、新しい binary が要ることを release のノートに、D4 の issue からの外れを明記。Codex plan advisory: MEDIUM 1 件を反映(「0 以外は off」は古い binary と設定の誤りを区別しない → D6 で終了コードを分け、AC4 に回帰テスト)。
- [x] Plan approved — 自動承認(ユーザーの指示)。digest は Approved の行。Critical forks は D1・D2 で、consult の判定に従って選んだ
- [x] Branch created (feat/review-switches-config, worktree .claude/worktrees/review-switches-config)
- [ ] Implementation started
- [ ] Review artifact created
- [ ] Verification artifact created
- [ ] Test artifact created
- [ ] PR created
