# scaffold-version-notice

- Status: Approved
- Approved: 2026-11-12 sha256:6ac68081cc35
- Owner: Claude Code
- Date: 2026-11-12
- Related request: 下流の運用 issue #307〜#313 の系列、レーン B の 1 本目。雛形(`.ralph/manifest.toml`)の版が CLI より古いことを、`ralph doctor` 以外の日々の動詞でも知らせる
- Related issue: https://github.com/yoshpy-dev/ralph/issues/313
- Type: feat
- Branch: feat/scaffold-version-notice

## Objective

雛形の版が CLI より古い repo で、`ralph org start`(`--plan` の形も)、`ralph org director attach`、`ralph org director start`、`ralph status` が stderr に `run 'ralph upgrade'` を含む 1 行を出すようにする。あわせて、`ralph upgrade` が書く報告 `docs/reports/upgrade-<version>-<date>.md` に、上げる前と後の版(from → to)を書く。

STOIC では 11/4 から 11/11 まで雛形が 7.0.0、CLI が 8.1.0 のままで、director は 7.0.0 の skill に従って座席の driver と model を書き続けた。ずれを知らせていたのは毎日は流さない `ralph doctor` だけだった。

## Scope

- 版を比べる関数と知らせの 1 行を作る新しいファイル `internal/cli/scaffold_version_notice.go`。版は semver で比べ(`golang.org/x/mod/semver`)、雛形の版が CLI の版より小さいときだけ 1 行を返す
- 4 つの動詞(org start の 2 つの形、director attach、director start、status)から、その 1 行を stderr に出す
- org の動詞は main worktree の manifest を読む: 台帳の main worktree の root(`org.MainWorktreeRoot`)、なければ今いる repo の main worktree(`gitroot.MainWorktree("")`)、なければ toplevel、なければ `.`
- status は scaffold の節と同じく `.` の manifest を読む
- 各動詞の `--help`(Long)に、この 1 行を出すことを 1 文足す
- upgrade の報告: `UpgradeReportData` に `FromVersion` を足し、`Generated:` の下に `Version: <from> → <to>` の行を出す(from が分からないときは出さない)。from は `ralph upgrade` の入口(`runUpgrade`)が読んだ manifest の版で、`runUpgradeV2` に引数で渡す。旧レイアウトの移行(`runMigrateLegacy`)は今の CLI の版で manifest を作り直してから `runUpgradeV2` を呼ぶので、移行前の manifest の版を渡す
- README の `ralph upgrade` の節と、status・org start の説明の該当箇所

## Non-goals

- upgrade の報告に「版ごとに挙動が変わること」の注記を出すこと(ユーザーの決定。/release の運用と合わせて別 issue にし、merge のあとに起票して番号を ralph-d1 に知らせる)
- `ralph doctor` の `checkManifestVersion` の比べ方と文言を変えること(今の文字列一致の warn のまま。雛形のほうが新しいときにも `run 'ralph upgrade'` と言う点は前からの型として Known gaps に書く)
- 雛形のほうが CLI より新しいとき(CLI を上げるべきとき)に 4 つの動詞が知らせること
- `ralph org spawn` など、issue と決定が挙げていない動詞に知らせを足すこと
- 起動を断ること。知らせは 1 行だけで、動詞の動きと終了コードは変えない

## Assumptions

- CLI の版 `cli.Version` は goreleaser の `{{.Version}}` で、`8.1.0` のように先頭の `v` が無い。ldflags の無いビルドは `dev`
- 雛形の版 `meta.version` は `ralph init` と `ralph upgrade` が `cli.Version` をそのまま書いたもの。先頭に `v` が付いていても読めるようにする
- `golang.org/x/mod` は手元のモジュールキャッシュに v0.41.0 があり、`go get` で go.mod と go.sum に足せる
- `internal/cli` のテストは 1 回で 5〜14 分かかる。slice の確かめは `-run` で絞る

## Affected areas

- `internal/cli/scaffold_version_notice.go`(新規)と `internal/cli/scaffold_version_notice_test.go`(新規)
- `internal/cli/org.go`(start の 2 つの形、Long)
- `internal/cli/org_director.go`(attach、Long)
- `internal/cli/org_director_headless.go`(start、Long)
- `internal/cli/status.go`(runStatus、Long)
- `internal/upgrade/report.go`、`internal/upgrade/report_test.go`
- `internal/cli/upgrade_v2.go`、`internal/cli/upgrade.go`、`internal/cli/migrate.go`、`internal/cli/upgrade_v2_test.go`、`internal/cli/migrate_test.go`
- `go.mod`、`go.sum`
- `README.md`

## Visual review

- Page: `.harness/state/plan-visual/scaffold-version-notice.html`(やさしい図 2 枚、詳しい図 2 枚)
- Lint: `./scripts/plan-visual.sh lint` exit 0
- Self-check: full、overview、easy-when、detail-overview、detail-decide の 5 枚を撮って読んだ。detail-decide で矢印とラベルが箱に重なっていたので配置を直し、撮り直して重なりが無いことを確かめた

## Design decisions

Critical forks: None。ユーザーが決めた点(対象の 4 動詞、semver、dev では出さない、org は main worktree・status は `.`、upgrade の報告は from→to だけ)の残りを、次のとおり自律で選んだ。

- D1(自律で選んだ): 版の比べ方は `golang.org/x/mod/semver` を使う。prerelease の順序(`8.2.0-rc.1 < 8.2.0`)まで正しく比べる実装を手で書くより、Go の toolchain が使う実装に任せるほうが誤りが少ない。依存は go.mod に 1 行増えるだけで、semver の package は他に何も引かない。どちらの版にも先頭に `v` を足してから `semver.IsValid` で確かめ、どちらかが読めなければ(CLI の `dev` を含む)何も出さない
- D2(自律で選んだ): 知らせは「雛形の版 < CLI の版」のときだけ出す。manifest が無い、読めない、雛形のほうが新しいか同じ、のときは何も出さない。manifest が読めないことは status の scaffold の節と doctor がすでに知らせるので、ここで重ねない
- D3(自律で選んだ): org の動詞が読む root は、台帳の main worktree の root → 今いる repo の main worktree → toplevel → `.` の順に決める。`org.FeatureRepoRoot` と同じ順で、`--state-dir` で台帳を外に置いたときも今いる repo の雛形を見る。linked worktree の写しは古いことがあるので読まない
- D4(自律で選んだ): 文言は 1 行で、どの動詞から出たか、どの root の雛形か、その版と CLI の版、`run 'ralph upgrade' in <root>` を含める。例: `note: org start: the scaffold in /repo (.ralph/manifest.toml) is ralph 7.0.0, older than this CLI (8.1.0), so its skills and rules may not match this CLI; run 'ralph upgrade' in /repo`。stdout(status の `--json` を含む)には何も足さない
- D5(自律で選んだ): 出す位置は、各動詞が `printLegacyModelPoolNotice` を出す所の隣(status は scaffold の節を読んだ直後)。動詞が後で断られても、知らせは先に出る
- D6(自律で選んだ): upgrade の報告の行は `Version: 7.0.0 → 8.1.0` の形で `Generated:` の次に置く。`FromVersion` が空なら行を出さない(既存の呼び出しと golden の出力は変わらない)。stdout の `Upgrade complete: <to>` は変えない。from は `runUpgradeV2` の引数 `fromVersion` で受け、`oldManifest.Meta.Version` からは取らない。旧レイアウトの経路では `oldManifest` が移行後に今の版で作り直したものになり、`8.1.0 → 8.1.0` と誤って書くため(codex の計画助言の指摘 1)。no-op の判定(`oldManifest.Meta.Version == Version`)は今のまま

## Acceptance criteria

- [ ] AC1: `.ralph/manifest.toml` の `meta.version` が CLI より古い repo で `ralph org start <task>` と `ralph org start --plan … --feature …` を流すと、stderr に `run 'ralph upgrade'` を含む 1 行が 1 回だけ出る
- [ ] AC2: 同じ repo で `ralph status`(`--json` も)を流すと、stderr に同じ 1 行が出て、stdout は変わらない
- [ ] AC3: 同じ repo で `ralph org director attach` と `ralph org director start` を流すと、stderr に同じ 1 行が出る
- [ ] AC4: CLI の版が `dev` か semver として読めないとき、manifest の版が読めないとき、manifest が無いとき、雛形の版が CLI と同じか新しいときは、4 つの動詞とも何も出さない
- [ ] AC5: linked worktree から org の動詞を流すと、main worktree の manifest の版で決まる(worktree の写しが古く main が新しいなら出さない、main が古いなら main の root を名指しして出す)。status は今いる directory の manifest で決まる
- [ ] AC6: 版は semver で比べる: `7.0.0 < 8.1.0`、`8.1.0 < 8.10.0`、`8.2.0-rc.1 < 8.2.0`、`v8.1.0` と `8.1.0` は同じ
- [ ] AC7: `ralph upgrade` が書く報告の `Generated:` の次に `Version: <from> → <to>` の行がある。from は上げる前の manifest の版で、旧レイアウトから移行して続けて上げたときも移行前の版になる。from が空なら行は無い
- [ ] AC8: 4 つの動詞の `--help` と README が、この 1 行と upgrade の報告の版の行を説明している

## Implementation outline

1. 版の比べ方と知らせの 1 行: `golang.org/x/mod` を go.mod に足し、`internal/cli/scaffold_version_notice.go` に `scaffoldVersionOlder(manifestVersion, cliVersion string) bool`、`scaffoldVersionNotice(verb, root, cliVersion string) string`(root の manifest を読む)、`orgScaffoldRoot(stateDir, source string) string`、`printScaffoldVersionNotice(w io.Writer, verb, root string)` を作り、単体テストを書く(AC4 の判定、AC6) — verify: `test -z "$(gofmt -l internal/cli)" && go vet ./internal/cli/ && GOFLAGS="-timeout=30m $(go env GOFLAGS)" go test ./internal/cli/ -run 'ScaffoldVersion'`
2. 4 つの動詞につなぐ: org start の 2 つの形、director attach、director start、status から呼び、各 `--help` に 1 文足す。CLI のテストで AC1〜AC5 を確かめる(`cli.Version` をテストの中で `8.1.0` にして戻す) — verify: `test -z "$(gofmt -l internal/cli)" && go vet ./internal/cli/ && GOFLAGS="-timeout=30m $(go env GOFLAGS)" go test ./internal/cli/ -run 'ScaffoldVersion|TestStatus|TestOrgDirector_Attach|TestOrgDirectorStart_RoleModels|TestOrgStart_RoleModels|TestOrgStartPlan'`
3. upgrade の報告に from → to: `UpgradeReportData.FromVersion` と `Version:` の行、`runUpgradeV2` に引数 `fromVersion` を足し、`runUpgrade`(`upgrade.go`)は読んだ manifest の版を、`runMigrateLegacy`(`migrate.go`)は移行前の manifest の版を渡す(AC7) — verify: `test -z "$(gofmt -l internal/upgrade internal/cli)" && go vet ./internal/upgrade/ ./internal/cli/ && GOFLAGS="-timeout=30m $(go env GOFLAGS)" go test ./internal/upgrade/ && GOFLAGS="-timeout=30m $(go env GOFLAGS)" go test ./internal/cli/ -run 'Upgrade|Migrat'`
4. README: `ralph upgrade` の節の報告の説明に版の行を、status と org start・director の説明に知らせの 1 行を足す(AC8) — verify: `./scripts/check-sync.sh && git diff --check`

## Verify plan

- Static analysis checks: `./scripts/run-static-verify.sh`(gofmt、go vet、golangci-lint、staticcheck、skill と template の同期)
- Spec compliance criteria to confirm: AC1〜AC8 を、テスト名と help の文で 1 つずつ突き合わせる。issue #313 の受け入れ条件 2 つ(org start と status)が AC1 と AC2 に入っていること
- Documentation drift to check: README の `ralph upgrade` の節、`ralph status` と `ralph org start` の説明、AGENTS.md の repo map(新しいファイルは internal/cli の中なので行は要らない見込み)、`.claude/rules/ralph/` に版のずれを書いた所が無いか
- Evidence to capture: `run-static-verify.sh` の結果、`go mod tidy` で go.mod と go.sum が変わらないこと

## Test plan

- Unit tests: `scaffoldVersionOlder` の表(AC6 の組、`dev`、空、`v` 付き、build metadata `+abc`、壊れた版)。`scaffoldVersionNotice` の manifest 無し、壊れた TOML、同じ版、新しい版、古い版、CLI が `dev`
- Integration tests: 一時 directory に `.ralph/manifest.toml`(7.0.0)を置き、`cli.Version = "8.1.0"` で `ralph status`、`ralph status --json`、`ralph org start <task>`(stub の herdr)、`ralph org start --plan`(既存の plan のテストの道具で)、`ralph org director attach`、`ralph org director start` を流し、stderr に 1 行が 1 回だけあることを確かめる。status は `runStatusCmd`(stdout と stderr を 1 つの buffer に混ぜる)を使わず、`status_test.go` の broken ralph.toml のテストのように stdout と stderr を別の buffer で取り、stdout に知らせが無いこと、`--json` の stdout が JSON として読めることを確かめる。`newLegacyLedgerRepo` で main と linked worktree の manifest の版を変えて AC5 を確かめる。upgrade の報告は `RenderUpgradeReport` の単体テストと、v2 の `ralph upgrade` の既存テストに `Version: <前の版> → <今の版>` の行の確かめを足す。旧レイアウトから移行して続けて上げるテスト(`migrate_test.go`)で、報告の from が移行前の manifest の版であり、今の版ではないことを確かめる
- Regression tests: 既存の org start・director・status のテストは stderr を完全一致で見ている。`internal/cli` のテストは package 変数 `Version` を `0.1.0-test` や `1.1.0-test` に書き換えて戻さない(`cli_test.go:74` ほか 20 箇所超)ので、テストの `Version` は `dev` とは限らない。既存のテストが知らせを出さずに済むのは、manifest を置くテストが `executeInit` で同じ `Version` を書くか、manifest を置かないからで、手で古い版の manifest を書くテストがあれば知らせが出て落ちる。`go test ./internal/cli/ ./internal/upgrade/` の全体を /test で流し、落ちたらそのテストが手で書いた manifest の版と直前の `Version` を見て直す
- Edge cases: CLI が `dev`、manifest の版が空、`8.1.0` と `v8.1.0`、prerelease の CLI(`8.2.0-rc.1`)と雛形 `8.1.0`(出す)、雛形 `8.2.0` と CLI `8.2.0-rc.1`(出さない)、`--state-dir` で台帳を repo の外に置いたとき
- Evidence to capture: `go test` の結果と所要時間、AC ごとのテスト名

## Risks and mitigations

- 新しいテストが `cli.Version` を書き換えたまま残すと、ほかのテストの stderr が変わる → 新しいテストは直前の値を `t.Cleanup` で戻し、`t.Parallel` を使わない。`scaffoldVersionNotice` は CLI の版を引数で受け、単体テストは package 変数に触れない
- 既存のテストは `Version` を戻さないので、テストの `Version` は直前のテストが入れた `0.1.0-test` などになりうる。stderr を完全一致で見るテストが、手で古い版の manifest を書いた directory で 4 つの動詞を流すと落ちる → manifest を置く既存のテストの多くは `executeInit` で同じ版を書くので出ない。/test で package 全体を流して確かめ、落ちたテストは manifest の版か `Version` をそのテストの中で決めて直す
- org の動詞が `gitroot.MainWorktree("")` を呼ぶので、chdir しないテストは ralph の repo の main worktree を読む → この repo の root には `.ralph/manifest.toml` が無い(`.ralph/` は `core/` と `local/` だけ)ので知らせは出ない。CI も同じ checkout
- main worktree を探す git の呼び出しが増え、org の動詞が遅くなる → git を 1〜2 回呼ぶだけ。台帳の root が分かるときは git を呼ばない
- 依存を足すことで CI の `go mod` の確かめが落ちる → `go mod tidy` を流して差が無いことを確かめる
- ほかのレーンが `internal/cli/org.go` と README を同時に変える → PR を作る前と merge の前に origin/main を merge する

## Rollout or rollback notes

- 動詞の動きは変えず、stderr に 1 行を足すだけ。戻すときは PR を revert する。upgrade の報告の行は新しく書く報告にだけ出る
- 次の release ノートに「`ralph org start`・`director attach`・`director start`・`ralph status` が、雛形が CLI より古いと stderr に 1 行出す」「upgrade の報告に from → to の版が出る」を入れる

## Open questions

- なし(ユーザーの決定で埋まっている)

## Progress checklist

- [x] Plan reviewed — plan advisory: ran with codex. 1 件 [MEDIUM]「旧レイアウトからの移行では `runUpgradeV2` に渡る manifest が今の版で作り直されたもので、報告の from が `8.1.0 → 8.1.0` になる」。`internal/cli/migrate.go:954`(`buildMigratedManifest(Version, …)`)と `:988` で事実を確かめ、計画を直した(Scope、Affected areas、D6、AC7、slice 3、Test plan)。consult(consult-plan-scaffoldver)の指摘 2 件(テストの `Version` は戻されない、`runStatusCmd` は stdout と stderr を混ぜる)も Risks と Test plan に入れた
- [x] Plan approved — 自動承認(ユーザーの指示「計画は自動承認」、2026-11-12)
- [x] Branch created
- [x] Implementation started
  - S1 b98eb0f6(implementer): 版の比較と知らせの部品、golang.org/x/mod v0.41.0。確認コマンド通過、`go mod tidy` で差なし
  - origin/main(#315、9ac72c50)を 6ac8fde4 で取り込み
  - S2 21e129c2(implementer、1 回止まって再開): 4 つの動詞と help。最初の実行で `TestStatusCmd_ScaffoldSection_LegacyManifest` が落ちた(直前のテストが残した `Version = "1.0.0-test"` と手書きの `0.9.0-test` の manifest で知らせが出て、`runStatusCmd` が stderr を `--json` の stdout に混ぜた)。Risks に書いたとおり、そのテストの中で `Version` を manifest と同じ版に決めて直した(`status_scaffold_test.go` をスコープに足した)
  - S3 6d727afc(implementer): upgrade の報告の `Version: <from> → <to>`。`runUpgradeV2(…, fromVersion, …)`、旧レイアウトの移行は移行前の版を渡す
  - S4 の範囲に、S2 で入れた help の文の言い直し(`internal/cli/scaffold_version_notice.go` の `orgScaffoldVersionNoticeHelp`、「the ledger's main worktree when the ledger is one」が分かりにくい)を足す。確認コマンドに `go test ./internal/cli/ -run ScaffoldVersion` を足す
  - 範囲の外として残すもの(Known gaps): director の skill と headless の prompt は、この知らせを見たときに人に伝えるか自分で `ralph upgrade` を流すかを決めていない
- [ ] Review artifact created
- [ ] Verification artifact created
- [ ] Test artifact created
- [ ] PR created
