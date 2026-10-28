# Walkthrough: review-switches-config(計画助言と cross-review の実施有無を ralph.toml で決める土台)

- Date: 2026-10-29
- Plan: docs/plans/active/2026-10-29-review-switches-config.md(/pr で archive に移す)
- Issue: #265
- PR: #284
- Branch: feat/review-switches-config(この walkthrough の前の HEAD 2fc350f9。origin/main da9d1b3a から 39 files、+1878/-204。計画、report、insight event を除くと 32 files、+1225/-204)
- 読む範囲: 実行コードは `internal/config/config.go`(2 つの型と既定値と検証)、`internal/cli/config.go`(新しい動詞)、`internal/cli/exitcode.go`、`internal/org/driver/herdr.go` と `paneenv.go`、`internal/org/spawn.go` の 1 行。残りのコードはテスト。設定はテンプレートとルートの `ralph.toml`。文書は README、AGENTS.md、`/org` skill の 4 つの写し、recipe、spec の注、`model-routing.md`
- 後工程は cycle 1 の 1 回。self-review は 2 回(1 回目の M1・L1〜L5 を F1 で直した)。cross-review の指摘は 0。記録だけの修正の例外は使っていない

```
何を足すか → 設定の表(S1) → 読み出しの動詞(S2) → 座席の pane の env(S3) → ルートの ralph.toml と文書(S4) → self-review の直し(F1) → 確かめたことと確かめていないこと
```

## 1. この PR が何か

- これまで: 標準フローの計画助言と cross-review は、相手の CLI があれば必ず走り、止める設定がない。org にもない。座席の pane には `AGMSG_SELF_NAME=off` と `RALPH_WIP_CHECKPOINT=off` だけが渡り、両方の CLI が入った環境では codex の座席でも `pick_reviewer` が driver を claude とみなす。
- この PR から:
  - `[standard]` と `[org.review]` に 4 つの実施有無(既定 off)と org の上限(既定 2)がある。
  - skill と座席は `ralph config get <key>` で値を読める。読み始めるのは #266・#267・#269。
  - 座席の pane に `RALPH_PRIMARY_CLI=<driver>` が渡る。
  - この repo はルートの `ralph.toml` で 4 つを on にする。
- この PR では、どの skill も設定を読まないので、動きは変わらない。
- 戻し方: この PR を revert する。テンプレートの新しい表は seed に残るが、古い binary は知らないキーを無視する。

## 2. 設定の表(S1、eff0340d)

- `internal/config/config.go:35` `StandardConfig`(`[standard]` の `plan_advisory`・`cross_review`)と `:561` `OrgReviewConfig`(`[org.review]` の `plan_advisory`・`cross_review`・`:574` `max_cross_review_cycles`)。既定値は `Default()` の `:586`・`:629` 付近。検証は `:861`(上限は 1 以上)。
- 上限の数え方は、最初の review を含む全部の回数(`RALPH_STANDARD_MAX_PIPELINE_CYCLES` と同じ。F1 で決めて、テンプレート、Go のコメント、skill、spec にそろえた)。
- `templates/base/ralph.toml`: `[standard]` は `[doctor]` と `[org]` の間、`[org.review]` は `[org.approval]` の後。コメントに、読む側の約束(終了コード)を書いた。
- `internal/config/defaults_sync_test.go`: 照合の表のループを `checkEachKeyIsComparedOnBothSides` に切り出し、`:580` `TestStandardTemplateDrift_EachKeyIsComparedOnBothSides` を足した。`[org.review]` の 3 キーは org の probe と表に入った。`shell_and_skills` に上限と `RALPH_STANDARD_MAX_PIPELINE_CYCLES` の照合。
- `internal/config/config_test.go`: 既定値、round trip、`:1906` の上限の境界、未知のキーを含む `ralph.toml` が読めること(古い binary が新しい seed を読める)。

## 3. 読み出しの動詞(S2、8e51e1c9)

- `internal/cli/config.go:25` `newConfigCmd`(`config` グループ。引数のない形と知らない subcommand は help と exit 1 で、`unknown command "config"` と区別できる)、`:46` `newConfigGetCmd`、`:92` `configValue`(toml のタグをたどる)。
- 終了コード: `:18` `ErrConfigKey` → 2(知らないキー、表、配列、値の下のキー)、`:23` `ErrConfigLoad` → 3(読めない、parse できない、検証で落ちる)。`internal/cli/exitcode.go:31-33` で対応づけた。3 は `ralph upgrade` の drift の 3 と同じ数で、動詞ごとに意味が違う(計画の D6)。
- Load をキーの照合より先に呼ぶので、壊れた `ralph.toml` では知らないキーでも 3。この順序はテストで固定していない(PR の Known gaps)。
- テスト: `internal/cli/config_test.go:33`〜`:240`。`:180` `TestConfigGet_InvalidOrgSettingDoesNotTurnCrossReviewOff` は、Codex の計画助言(MEDIUM)の回帰テストで、正しくない `[org]` と true の `standard.cross_review` が同じ file にあるとき 3 で終わる。

## 4. 座席の pane の env(S3、33d859bb)

- `internal/org/driver/herdr.go:249` `TabCreate` に可変長の `extraEnv`。`--env` は固定の 2 つのあとに extraEnv。
- `internal/org/driver/paneenv.go:41` `PrimaryCLIEnv`、`:50` `PrimaryCLIEntry`。`--env` のない herdr の warn に `RALPH_PRIMARY_CLI` の節。
- `internal/org/spawn.go:998`: spawn が座席の driver から作って渡す。headless の director の `TabCreate`(`internal/org/director_start.go`)は extraEnv なし。
- テスト: `internal/org/spawn_test.go:3764` `TestOrgSpawn_PaneGetsRalphPrimaryCLIOfItsDriver`(claude と codex の worker と leader)、director の pane に入らないこと、driver の `--env` の順と warn、`internal/cli` の 2 つのテスト(argv と warn)。

## 5. ルートの ralph.toml と文書(S4、98e19052)と self-review の直し(F1)

- ルートの `ralph.toml`: F1(10a05dfe)で、4 つの実施有無と先頭の注だけにした。全文の写しだと、次に `Default()` を変えたときにルートが古い値のまま残り、lock-step のテストも check-sync(`KNOWN_DIFFS`)も気づかない。`config.Load` が既定から始めるので、ほかの値は既定になる。
- `scripts/check-sync.sh` の `KNOWN_DIFFS` に `ralph.toml`。
- 文書: README の command 表と pane の env、AGENTS.md の `config`、`/org` skill の前提の節、`docs/recipes/isolated-herdr-server.md`、spec の FR-7 の「2026-10-29 に決めたこと」、`model-routing.md` の照合の列挙(sync-docs)。
- F1 の言い回しの直し: 座席が `RALPH_PRIMARY_CLI` を要る理由は「座席の cross-review が `pick_reviewer` でもう一方の CLI を選ぶため」。今の `/plan` の計画助言は codex を直接呼ぶので、計画助言には触れない。値がないと `pick_reviewer` は driver を claude とみなす。

## 6. 確かめたことと確かめていないこと

- 確かめた:
  - /verify: `run-static-verify.sh` rc 0(golangci-lint 0 件)。ルートで 4 つが true、上限 2、ほかは既定。build した binary で 0・2・3。base から build した古い binary は `unknown command "config" for "ralph"` で rc 1、新しいテンプレートも読める。
  - /test: `run-test.sh` rc 0(Go 9 パッケージ、shell 45 本)。`-overlay` の変異 19 個中 18 個が落ちた。
  - cross-review: 指摘 0。
- 確かめていない:
  - 変異 k(終了コード 3 と 2 の順序)。
  - PATH に `ralph` がないとき(exit 127)の扱い。#266 で決める。
  - ルートの `ralph.toml` のキーを読む常設のテスト。#266 で足す。
  - 本物の herdr での pane の env と、Linux(PR の CI で見る)。
