# Verify report: 雛形の版が CLI より古いことを日々の動詞で知らせ、upgrade の報告に from → to を書く(#313)

- Date: 2026-11-12
- Plan: `docs/plans/active/2026-11-12-scaffold-version-notice.md`(digest `6ac68081cc35`。`./scripts/plan-visual.sh digest` の値と `- Approved:` の行が一致)
- Verifier: verifier subagent(Claude Code)。パイプラインの cycle 1(`cycle-count.json` は `{"cycle": 1}`)
- Scope: `git diff origin/main...HEAD`(HEAD `71c7b31a`、base は merge-base の `9ac72c50`、21 ファイル、+1050/-13)。コードの commit は S1 `b98eb0f6`、S2 `21e129c2`、S3 `6d727afc`、S4 `32850be7`(README と help の言い直し)、`de4eab5f`(`orgScaffoldRoot` の doc comment)。`6ac8fde4` は origin/main の取り込み。`git ls-remote` で origin/main は `9ac72c50` のままなので、base との差は無い。テストは走らせていない(/test の役目)。テストの欄は、テストがあり、その assert が AC の言うことを確かめていることまでを読んで確かめた
- Evidence: `docs/evidence/verify-2026-11-12-scaffold-version-notice.log`(git が無視する)。`run-static-verify.sh` の全出力、`go mod tidy -diff` と `go mod verify`、ldflags で 8.1.0 と dev にした 2 つの binary による `--help` と `ralph status` と `ralph org director attach` の観察。スクリプト自身の写しは `docs/evidence/verify-2026-11-12-094257.log`

## Spec compliance

| Acceptance criterion | Status | Evidence |
| --- | --- | --- |
| AC1 古い雛形の repo で `ralph org start <task>` と `ralph org start --plan … --feature …` が stderr に `run 'ralph upgrade'` を含む 1 行を 1 回だけ出す | Met | task の形は `internal/cli/org.go:1178`、`--plan` の形は `runOrgStartPlan` の `:1291`。RunE は `:1147-1154` で `--plan` なら `runOrgStartPlan` を return するので、2 つの呼び出しが同じ実行で両方走ることはない。文は `scaffold_version_notice.go:70-72`、改行 1 つで書くのは `:111-115`。テスト `TestScaffoldVersionOrgStart_TaskFormNotesTheScaffoldOnce`(`scaffold_version_notice_cli_test.go:44`)と `TestScaffoldVersionOrgStartPlan_ReadsTheMainCheckoutScaffold`(`:74`)は stderr 全体を 1 行と完全一致で比べるので、「1 回だけ」も確かめている。issue #313 の受け入れ条件 1 に当たる |
| AC2 同じ repo で `ralph status`(`--json` も)が stderr に同じ 1 行を出し、stdout は変わらない | Met | `internal/cli/status.go:222` が `cmd.ErrOrStderr()` に `"."` で出す。テスト `TestScaffoldVersionStatus_NotesOnStderrAndLeavesStdout`(`:183`)は stdout と stderr を別の buffer で取り、dev と 8.1.0 の stdout をバイト単位で比べ、`--json` が JSON として読めることを見る。実機でも同じ結果を得た(Observational checks)。issue #313 の受け入れ条件 2 に当たる |
| AC3 `ralph org director attach` と `ralph org director start` が同じ 1 行を出す | Met | attach は `internal/cli/org_director.go:151`、start は `org_director_headless.go:340`(`startDirector`。呼び手は 1 か所)。テスト `TestScaffoldVersionDirectorAttach_ReadsTheMainWorktreeScaffold`(`:113`、3 つの case)と `TestScaffoldVersionDirectorStart_NotesTheScaffoldOnce`(`:158`)。attach は実機でも確かめた |
| AC4 CLI が `dev` か semver でない、manifest の版が読めない、manifest が無い、雛形が同じか新しい、のときは 4 つの動詞とも何も出さない | Met | 判定は 1 か所: 5 つの呼び出しはすべて `printScaffoldVersionNotice` → `scaffoldVersionNotice`(`scaffold_version_notice.go:56-73`)を通る。読めない manifest は `:57-60`、semver でない版は `scaffoldVersionOlder` の `:28-30`、同じか新しい版は `:31`。テスト: `TestScaffoldVersionOlder_ComparesSemver`(`scaffold_version_notice_test.go:34`、`dev`、空、`not-a-version`、`8.1.0.0`)、`TestScaffoldVersionNotice_SilentCases`(`:70`、manifest 無し、壊れた TOML、同じ版、新しい版、`dev`、空、`[meta]` 無し)、`TestScaffoldVersionStatus_SilentCases`(`scaffold_version_notice_cli_test.go:236`、status で 6 つ)。org の 3 つの動詞の、コマンドを通した黙る case は attach の「main が新しい」だけ(Coverage gaps 1) |
| AC5 linked worktree から org の動詞を流すと main worktree の manifest で決まる。status は今いる directory の manifest で決まる | Met | org の動詞は `orgScaffoldRoot`(`scaffold_version_notice.go:91-102`)で root を決める。順は `org.FeatureRepoRoot`(`internal/org/statedir.go:189-202`)の最初の 2 段と同じ。status は `"."` を渡す(`status.go:222`)。テスト: `TestScaffoldVersionOrgRoot_LedgerPathNamesTheRootWithoutGit`(`:150`、台帳の path から git なしで root を読む)、`TestScaffoldVersionOrgRoot_LinkedWorktreeReadsTheMainWorktree`(`:165`、2 つの向き)、`TestScaffoldVersionOrgRoot_OutsideGitIsTheWorkingDirectory`(`:192`)、CLI の attach の 3 case と start `--plan`(linked worktree の path が stderr に無いことも見る)。status を linked worktree から流すテストは無いが、コードは `"."` 以外を読まない(Coverage gaps 2) |
| AC6 semver で比べる: `7.0.0 < 8.1.0`、`8.1.0 < 8.10.0`、`8.2.0-rc.1 < 8.2.0`、`v8.1.0` と `8.1.0` は同じ | Met | `scaffoldVersionOlder`(`:26-32`)は両方に `v` を補い(`semverWithV`、`:37-43`)、`golang.org/x/mod/semver` の `IsValid` と `Compare` を使う。テスト `TestScaffoldVersionOlder_ComparesSemver` に 4 つの組がそのままある(`:41-47`)。逆向き(`8.2.0` と `8.2.0-rc.1` で false)、build metadata、前後の空白もある。version の短縮形については V5(INFO) |
| AC7 upgrade の報告の `Generated:` の次に `Version: <from> → <to>`。from は上げる前の manifest の版で、旧レイアウトの移行のあとも移行前の版。from が空なら行は無い | Met | `internal/upgrade/report.go:61-63`(`FromVersion` が空なら出さない)。`runUpgradeV2` は引数 `fromVersion` を受ける(`internal/cli/upgrade_v2.go:67`、doc `:60-66`)、`:185` で報告に渡す。no-op の判定は `oldManifest.Meta.Version` のまま(`:174`)。呼び手は 2 つで、`upgrade.go:157` は読んだ manifest の版、`migrate.go:993` は移行前の `oldManifest.Meta.Version` を渡す。`buildMigratedManifest`(`migrate.go:1581-1582`)は `scaffold.NewManifest(version)` で新しい値を作り、`oldManifest` を書き換えない。テスト: `TestRenderUpgradeReport_FromVersion_VersionLineAfterGenerated`(`report_test.go:109`)、`TestRenderUpgradeReport_EmptyFromVersion_NoVersionLine`(`:130`)、`TestRunUpgradeV2_FullPass`(`upgrade_v2_test.go:392`、`1.0.0-test → 1.1.0-test`)、`TestRunUpgradeV2_EmptyManifestVersion_ReportHasNoVersionLine`(`:1135`)、`TestRunMigrateLegacy_HappyPath_Yes`(`migrate_test.go:1292`、`0.9.0-test → 2.0.0-test` があり、`2.0.0-test → 2.0.0-test` が無いこと) |
| AC8 4 つの動詞の `--help` と README が、この 1 行と upgrade の報告の版の行を説明している | Met | help: `orgScaffoldVersionNoticeHelp`(`scaffold_version_notice.go:122-127`)を `org.go:1139`、`org_director.go:137`、`org_director_headless.go:163` が、`statusScaffoldVersionNoticeHelp`(`:133-138`)を `status.go:167` が Long の最後の段落にする。テスト `TestScaffoldVersionHelp_EachVerbDescribesTheNote`(`:272`)。8.1.0 の binary の `--help` で 4 つとも段落が出ることを見た。README: `ralph org` の行(L131)、`ralph status` の行(L132)、`ralph upgrade` の節の報告の説明(L162、`Version: <from> → <to>`、移行のあとの from、版の無い manifest で行が無いこと)。言い方の細部は V4〜V6(INFO) |

計画の決定との照合: D1〜D6 はコードと一致した。D1 の「semver の package は他に何も引かない」は `go list -deps` で確かめた(標準ライブラリの外は `golang.org/x/mod/semver` だけ)。D4 の文は `scaffoldVersionNoticeWant`(`scaffold_version_notice_test.go:26-29`)が固定し、実機の出力も同じ形だった。D5 の位置は 5 つの呼び出しとも `printLegacyModelPoolNotice` の隣か、status では scaffold の節を読んだ直後。D6 は AC7 の行のとおり。計画の本文(digest の内側)は変わっていない。

issue #313 との照合: 受け入れ条件 2 つは AC1 と AC2 に入っている。「やること」の 2 つ目(upgrade の報告の先頭に挙動が変わることを出す)は、計画の Non-goals でユーザーの決定として別 issue に回している。

## Static analysis

| Command | Result | Notes |
| --- | --- | --- |
| `./scripts/run-static-verify.sh` | Pass(exit 0、66 秒) | changed scope で golang が選ばれた。shellcheck と `sh -n`(hooks)、awk の parse、`jq -e`(settings)、Codex の hook の 3 つの guard、`check-sync.sh`(DRIFTED 0)、`check-pipeline-sync.sh`、`check-skill-sync.sh`(14 skill)、`check-template-purity.sh`、tech-debt の計画の参照、golang(gofmt ok、`go vet ./...`、golangci-lint `0 issues.`、staticcheck は出力なし)、branch の secret scan(`9ac72c50..71c7b31a` clean) |
| `go mod tidy -diff` | Pass(rc 0、差なし) | `-diff` はファイルを書かない。go.mod と go.sum は tidy のまま |
| `go mod verify` | Pass | `all modules verified`。`golang.org/x/mod v0.41.0` は直接依存の block にあり、go.sum は h1 と go.mod の 2 行 |
| `./scripts/plan-visual.sh digest` | Pass | `6ac68081cc35`、`- Approved:` の行と一致 |

実行中に負荷平均が 40 前後あったが、時間切れは起きなかった。

## Documentation drift

| Doc / contract | In sync? | Notes |
| --- | --- | --- |
| 4 つの動詞の `--help` | Yes | 段落はある。言い方の細部は V4〜V6(INFO) |
| `ralph upgrade --help` | Yes | 報告の中身(見出し)を説明していないので、足す所は無い。計画の Scope も README だけを挙げる |
| README(`ralph org`、`ralph status`、`ralph upgrade` の節) | Yes | 3 か所。V4〜V6(INFO) |
| `docs/specs/2026-08-17-overlay-scaffold-v2.md` | Yes | 報告の path だけを書き、見出しの中身は書いていない |
| `docs/specs/2026-04-16-ralph-cli-tool.md` の依存の表 | 対象外 | 2026-04 の記録で、`charmbracelet/x/term` など今の直接依存もすでに載っていない。生きた契約ではないので、`golang.org/x/mod` を足す必要は無い |
| `AGENTS.md` の repo map | Yes | 新しいファイルは `internal/cli/` の中で、map はそこのファイルを個別に挙げない |
| `.claude/rules/ralph/`、skill(`.claude`、`.agents`、`templates/base`)、`internal/org/prompts`、recipes | Yes | 版のずれや `meta.version` を書いた所は無い(grep 0 件)。4 つの動詞の stderr を `2>&1` で読む所も無い |
| tech-debt の register | No(LOW) | V1〜V3。この PR が引いた Trigger に、払わなかったことの記録が無い |
| 計画 | Yes | AC のチェックボックスと Progress の遅れは V7(INFO) |

### 所見

LOW(3 件。どれも /sync-docs への記録の依頼で、コードは変えない。前の PR の書き方「<日付>: <branch> hit this trigger with ..., but did not pay it: ... Carried over to the next change that meets this trigger」に合わせるとよい)

- V1(LOW)`docs/tech-debt/README.md` 88 行「6 batchable LOW findings from org-debt-batch」: Trigger は「Next time the corresponding file is touched for any other reason」。項目 (4) の `internal/cli/status.go` をこの PR が変えた(966 → 971 行、`runStatus` の知らせの呼び出しと Long の段落)。`printStateDirLine` は 1 か所のまま、`state_dir_source` の `omitempty` も残っている。2026-11-02 と 2026-11-05 の PR は同じ引き方を記録している
- V2(LOW)同 201 行「Test gaps left by org-feature-worktree」: (e) と (f) の Trigger「the next change to `runOrgStartPlan`, `StartFeature`, or `scriptFeatureWorktrees`」を引いた(`runOrgStartPlan` に呼び出し 1 行と doc の 4 行、`org.go:1265-1268` と `:1291`)。(e) の `org.ResolveRoleModel` の失敗の枝は今も `runOrgStartPlan` を通して走らない。2026-11-06 の PR は同じ形の変更(呼び出し 1 行と doc)を記録している
- V3(LOW)同 261 行「Two test gaps of the org-director-attach PR」: H4 と H5 の Trigger「the next change ... to the `attach` or `detach` `RunE` of `org_director.go` for H4 and H5: then add the tests with that change」を引いた(attach の RunE に `org_director.go:151` の 1 行)。H4 のテスト(store が `ErrDirectorWriteNotRestored` を返す形)は足していない。払うには store の seam が要るので、この PR の Scope の外として記録するのが妥当

引いていないと判断した行: 27 行(`runUpgradeIO` の引数。変わったのは `runUpgradeV2` で、`fromVersion` は表示や IO の knob ではない)、99 行(挙げる関数 `WriteUpgradeReport`、`writeFileV2`、`writeMigrationFile` などは変わっていない)、115 行(`relocationOutcome` と `buildMigratedManifest` は変わっていない)、148 行と 190 行の (d) と 201 行の (j)(`MainWorktreeRoot`、`LedgerMainWorktreeRoot`、`withMainWorktreeOrgLimits` は変わっていない。`org.MainWorktreeRoot` を `internal/cli` から呼ぶのはこれが初めてだが、呼び手を数えて書いた行は無い)、260 行と 417 行(status の header に行を足す、または section を足すとき。知らせは stderr の 1 行で、stdout の節は変わらない)、275 行(`startDirector` の断る順。知らせは断りではなく、順は変わらない)、319 行(`director_start.go` は変わっていない)。self-review は「関係する既存の行は `runUpgradeIO` の 1 件だけ」としていたが、V1〜V3 は見ていなかった。

INFO

- V4: help の段落(`scaffold_version_notice.go:122-127`)と README の `ralph org` の行は root を「main worktree」とだけ言う。コードは main worktree が名指せないとき toplevel、git の外では `.` を読む(`:98-101`)。self-review の L2 と同じ所見で、PR の Known gaps か follow-up に書く扱いが計画と self-review で決まっている
- V5: `golang.org/x/mod/semver` は `vMAJOR` と `vMAJOR.MINOR` を `vMAJOR.0.0` と `vMAJOR.MINOR.0` の短縮形として valid にする(package の doc)。このため manifest の `8.1` は `8.1.0` として比べられ、help と README の「a version that is not semver」の言い方よりコードのほうが受け入れる範囲が広い。ralph は `init` と `upgrade` で 3 つの数の版を書くので、実際に差が出ることはまず無い
- V6: help と README は知らせが出ない場合として「dev build、semver でない版、同じか新しい雛形、manifest が無い」を挙げ、manifest が読めない(壊れた TOML)場合を挙げていない。コードは黙る(D2)。列挙は「これで全部」とは言っていないので drift ではない
- V7: 計画の AC1〜AC8 のチェックボックスはすべて `[ ]`、Progress の「Review artifact created」も `[ ]` のまま(self-review の報告は `71c7b31a` にある)。/pr までに付ける記録の遅れ
- V8: `scaffold_version_notice_cli_test.go:14` は計画を `docs/plans/active/...` の path で引く。/pr が計画を archive に移すと古くなる(self-review の L3)
- V9: director の skill と headless の prompt は、この知らせを見たときに人に伝えるか自分で `ralph upgrade` を流すかを決めていない。計画の Progress が Known gaps として残すと書いている

## Observational checks

ldflags で `main.Version=8.1.0` にした binary と、ldflags なし(`dev`)の binary を scratchpad に作り、scratchpad の中だけで流した(`GIT_CEILING_DIRECTORIES` で上の repo を見ない)。

- `--help`: `org start`、`org director attach`、`org director start`、`status` の 4 つとも、最後の段落に知らせの説明が出る
- `ralph status`(table と `--json`)、manifest 7.0.0: 8.1.0 の binary は stderr に 1 行(`note: status: the scaffold in <dir> (.ralph/manifest.toml) is ralph 7.0.0, older than this CLI (8.1.0), ...; run 'ralph upgrade' in <dir>`)。dev の binary は stderr が空。stdout は table も `--json` も 2 つの binary でバイト単位で同じ(`cmp`)で、`--json` は JSON として読める
- `ralph status`、8.1.0 の binary: manifest が `8.1.0`、`v8.1.0`、`9.0.0`、`""`、壊れた TOML、manifest 無しで 0 行、`8.0.9` で 1 行
- `ralph org director attach --driver claude --state-dir <repo の外>` を linked worktree から: main 7.0.0、linked 8.1.0 で main の root を名指す 1 行、linked worktree の path は出ない。登録はされる(rc 0)。main 8.1.0、linked 7.0.0 では 0 行
- `ralph org start` の 2 つの形と `director start` は実機で流していない(herdr と座席を起こすため)。CLI のテストが stub の herdr で確かめている

## Coverage gaps

1. AC4 をコマンドを通して確かめたのは status の 6 case と attach の 1 case。org start の 2 つの形と director start の黙る case は、同じ `printScaffoldVersionNotice` を通ることをコードで確かめた(5 つの呼び出しで判定の関数は 1 つ)
2. status を linked worktree から流し、今いる worktree の manifest で決まることを見るテストは無い。コードは `"."` だけを読む(`status.go:222`)
3. main worktree の無い repository(bare の linked worktree)で toplevel の段に落ちる道は、CLI でも単体でもテストが無い。V4 の言い方の差もこの道に当たる
4. `internal/cli` の既存のテストのうち、手で古い版の manifest を書いて stderr を完全一致で見るものが、直前のテストの残した `Version` で落ちるかは、package 全体を流さないと分からない(計画の Risks)。S2 で 1 本(`TestStatusCmd_ScaffoldSection_LegacyManifest`)が見つかって直っている。確信を最も上げる次の確かめは、/test が `go test ./internal/cli/ ./internal/upgrade/` の全体を流すこと
5. テスト、mutant は走らせていない(/test の役目)

## Verdict

- Verified: AC1〜AC8(コード、テストの assert、help、README の照合。AC2、AC3、AC4、AC5、AC8 の一部は実機の binary でも確かめた)。`run-static-verify.sh` exit 0、`go mod tidy -diff` 差なし、`go mod verify` 通過、計画の digest 一致
- Partially verified: AC4 と AC5 のうち、org start と director start の黙る case と status の linked worktree の case は、コードの構造で確かめた(Coverage gaps 1、2)
- Not verified: テストの実行結果(/test)、bare の linked worktree の道、org start と director start の実機での出力
- Findings: LOW 3(V1〜V3、tech-debt の register に Trigger を引いた記録が無い。/sync-docs で記録する)、INFO 6(V4〜V9)
- Verdict: pass
