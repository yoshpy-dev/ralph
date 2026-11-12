# Self-review report: scaffold-version-notice

- Date: 2026-11-12
- Plan: `docs/plans/active/2026-11-12-scaffold-version-notice.md`(issue #313)
- Reviewer: reviewer サブエージェント(`/self-review`)。パイプライン 1 回目(`cycle-count.json` は cycle 1)
- Scope: `git diff origin/main...HEAD`(HEAD `1c5e4596`、base は `origin/main`)。19 ファイル、978 行追加、13 行削除(計画ファイルを含む)。見たのは diff の質(名前、読みやすさ、null 安全、エラーの扱い、安全性、保守性)だけ。仕様への適合、テストの網羅、文書の食い違いは見ていない(`/verify`、`/test`、`/sync-docs` の担当)

## Evidence reviewed

読んだもの。

- `internal/cli/scaffold_version_notice.go`(138 行、全部)
- 4 つの動詞の差分の全部: `internal/cli/org.go`(`newOrgStartCmd` の RunE と `runOrgStartPlan`)、`org_director.go`(attach)、`org_director_headless.go`(`startDirector`)、`status.go`(`runStatus`)。各呼び出しの前後も読んだ
- upgrade の報告の差分の全部: `internal/upgrade/report.go`、`internal/cli/upgrade_v2.go`、`upgrade.go`、`migrate.go`(`runMigrateLegacy` の chain と `buildMigratedManifest` の頭)
- 比較先として `internal/org/statedir.go`(`ResolveOrgStateDir`、`defaultOrgStateDirBase`、`MainWorktreeRoot`、`LedgerMainWorktreeRoot`、`FeatureRepoRoot`)、`internal/scaffold/manifest.go` の `ReadManifest` と `Meta.Version`、兄弟の知らせ `printLegacyModelPoolNotice` と `note:` の接頭辞を持つ既存の出力(`org_legacy_ledger.go`、`org_approve.go`)
- テストの差分の全部: `scaffold_version_notice_test.go`(232 行)、`scaffold_version_notice_cli_test.go`(301 行)、`migrate_test.go`、`status_scaffold_test.go`、`upgrade_v2_test.go`、`report_test.go`。`newLegacyLedgerRepo` と `isolateFeatureGitEnv` の定義も読んだ(前者は linked worktree へ chdir する)
- `go.mod`、`go.sum`、README の 3 つの追記(`ralph org` の行、`ralph status` の行、`ralph upgrade` の節)。`docs/tech-debt/README.md` は差分の両方向で突き合わせた(この diff は register に触れない。関係する既存の行は `runUpgradeIO` の引数の増加 1 件だけ)
- 追加行に不可視 Unicode(U+2028、U+200B 系、U+202A から U+202E、U+FEFF)と U+FFFD は無い(コードとテストの追加行で 0 行)

指摘なしで確かめたこと(次の run で繰り返さなくてよい)。

- 知らせの文に入る値: 版は `semver.IsValid` を通ったものだけが入る(制御文字は通らない)。root は `filepath.Abs` を通り、verb は呼び手が渡す定数。ユーザーの入力がそのまま文に入る道は無い
- stdout に触れない: 4 つの呼び出しはすべて `cmd.ErrOrStderr()` を渡し、`printScaffoldVersionNotice` は writer 1 つにしか書かない。`ralph status` の出力を `2>&1` で JSON として読む本番コードは repo の `.claude/skills`、`scripts`、`internal/org/prompts` に無い(`ralph status` に当たる行を grep して見た)
- 二重に出ない: `org start` の 2 つの形は `RunE` の `orgStartUsesPlan` の分岐で排他(`runOrgStartPlan` が return する)。`startDirector` の呼び手は `org_director_headless.go:177` の 1 か所、`runStatus` の呼び手は `status.go:173` の 1 か所
- from の版の取り方: `runUpgradeV2` の呼び手 2 つ(`upgrade.go:157`、`migrate.go:993`)はどちらも移行前の manifest の版を渡す。`buildMigratedManifest` は `scaffold.NewManifest(version)` で新しい値を作り、`oldManifest` を書き換えないので、`migrate.go:993` の `oldManifest.Meta.Version` は移行前の版のまま。no-op の判定(`oldManifest.Meta.Version == Version`)は元のまま。`FromVersion` が空なら `Version:` の行を出さない
- エラーの扱い: `scaffold.ReadManifest` の失敗を全部「何も出さない」にするのは計画の D2(status の scaffold の節と doctor が別に知らせる)で、知らせは動詞の終了コードを変えない。門(gate)ではなく情報なので、壊れた manifest で開いたまま(fail open)になる点は意図どおり
- 版の比べ方: `semverWithV` で `v` を補い、両方が `semver.IsValid` を通らなければ `false`。`dev`、空、`8.1.0.0`、大文字の `V` は黙る側に倒れる
- 接頭辞 `note: ` は `org_legacy_ledger.go:151,169`、`org_approve.go:100,134` と同じ型
- go.mod の `golang.org/x/mod v0.41.0` は直接依存のブロックにあり、go.sum は 2 行(h1 と go.mod)

読んでいない・実行していないもの。テスト、`go build`、`go vet`、linter、`go mod tidy` は走らせていない(このレビューの対象外。`/verify` と `/test` が見る)。

## Findings

<!-- Area recommended values: naming, readability, null-safety,
     exception-handling, security, maintainability -->

| Severity | Area | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- |
| LOW | maintainability | L1: `orgScaffoldRoot` の最初の 2 段は `org.FeatureRepoRoot` の最初の 2 段の写しで、「同じ順」であることを doc comment の約束だけで守っている。片方の順が変わると、org の動詞が名指す root と `start --plan` が worktree を作る root が黙ってずれる | `internal/cli/scaffold_version_notice.go:91-102`(`org.MainWorktreeRoot`、`gitroot.MainWorktree("")`、`gitroot.Toplevel`)と `internal/org/statedir.go:189-202`(`FeatureRepoRoot` の同じ 2 段のあとに Toplevel でエラーの文を作る)。doc comment の「It tries them in the order org.FeatureRepoRoot does」(同 86-90 行)が唯一のつなぎ | 先頭を `if root, err := org.FeatureRepoRoot(stateDir, source); err == nil { return root }` にして、そのあとに Toplevel と `"."` だけを残す。`FeatureRepoRoot` が失敗する道(git が main worktree を名指さない)だけ Toplevel を 2 回呼ぶが、その道は稀で、結果は変わらない |
| LOW | readability | L2: help の段落と README、`runOrgStartPlan`・`startDirector` の doc comment は「main worktree」だけを言うが、コードは main worktree を名指せないとき toplevel、git の外では `.` を読む。テストはこの `.` の道をそのまま固定している | `scaffold_version_notice.go:122-127`(`orgScaffoldVersionNoticeHelp`)。コードは同 91-102 行。固定しているテストは `TestScaffoldVersionOrgStart_TaskFormNotesTheScaffoldOnce` と `TestScaffoldVersionDirectorStart_NotesTheScaffoldOnce`(どちらも git の外で、`.` の manifest を読ませる)。README の `ralph org` の行の追記も同じ言い方 | help と README に 1 句足す(「git の外や main worktree の無い repository では、その toplevel か今の directory の」)。help の定数は 1 か所なので、足す所は `orgScaffoldVersionNoticeHelp` と README の 1 文、doc comment 2 つ |
| LOW | maintainability | L3: テストの comment が `docs/plans/active/<slug>.md` を path で引いている。`/pr` が計画を `docs/plans/archive/` へ移すと、この path は古くなる。この diff の他の comment は `plan 2026-11-12-scaffold-version-notice` と slug だけで引いている | `internal/cli/scaffold_version_notice_cli_test.go:14`(`docs/plans/active/2026-11-12-scaffold-version-notice.md`)。slug だけの引き方は `scaffold_version_notice.go:18,47,78,107`、`upgrade_v2.go`、`migrate.go` ほか | その 1 行を `plan 2026-11-12-scaffold-version-notice` の形にそろえる(周辺の既存コードにも `plans/active` を引く所が `internal/cli` の本番コードに 25 あるが、この diff の外) |
| LOW | readability | L4(体裁の 3 点): (a) `startDirector` の doc comment の言い直しで、`// switched-off director,` が 1 語だけの行になり、次の行と折り返しが合っていない。(b) `status_scaffold_test.go` の新しい comment が 1 行 161 桁(周りは 80 桁前後で折っている)。(c) `TestScaffoldVersionOrgRoot_OutsideGitIsTheWorkingDirectory` が、同じ package の `scaffoldVersionOutsideGit` と同じ 2 行(`isolateFeatureGitEnv`、`GIT_CEILING_DIRECTORIES`)を繰り返している | (a) `internal/cli/org_director_headless.go:307-309`。(b) `internal/cli/status_scaffold_test.go:173`。(c) `scaffold_version_notice_test.go:193-198` と `scaffold_version_notice_cli_test.go:34-38`(helper は cli_test 側の定義で、同じ package から呼べる) | (a) 「refuses a switched-off director, a relative [org].agmsg_home …」を 1 つの段落に折り直す。(b) 80 桁前後で折る。(c) helper を呼ぶ。どれも動作に影響しない |

重大度の数: CRITICAL 0、HIGH 0、MEDIUM 0、LOW 4。

## Positive notes

- 判定(`scaffoldVersionOlder`)、文の組み立て(`scaffoldVersionNotice`)、root の選び方(`orgScaffoldRoot`)、書き込み(`printScaffoldVersionNotice`)が 4 つの小さな関数に分かれ、`cliVersion` を引数で受けるので、単体テストが package 変数に触れずに済む。package 変数 `Version` を読むのは `printScaffoldVersionNotice` の 1 行だけ
- `fromVersion` を引数にして、移行の経路で `oldManifest` が今の版で作り直されることによる `8.1.0 → 8.1.0` の誤りを避けている。`runUpgradeV2` の doc と `migrate.go` の呼び出しの前の comment が、なぜ `oldManifest.Meta.Version` から取らないかを書いている。`migrate_test.go` が「from が今の版ではない」ことを否定の形でも確かめる
- 新しいテストは `Version` を `t.Cleanup` で戻し、`t.Parallel` を使わない。status のテストは `dev` の実行と `8.1.0` の実行の stdout をバイト単位で比べ、「stdout は変わらない」を直接確かめている。`TestScaffoldVersionOrgRoot_LedgerPathNamesTheRootWithoutGit` は PATH から git を外して、先頭の段が git を呼ばないことを固定している(後ろの段なら `.` が返るので差が出る)
- 既存の `status_scaffold_test.go` のテストが、直前のテストの `Version` の残りで壊れた原因を、そのテストの中で `Version` を manifest にそろえて直した(計画の Risks に書いたとおり)

観察として残すもの(指摘にはしない)。

- 知らせの文は `run 'ralph upgrade' in <dir>` に dir をそのまま入れる。空白を含む path だと読みづらいが、これは command ではなく「in <dir>」と言う文なので、`ShellWord`(`internal/org/shellword.go`。merge と cleanup が出す command 用)で包む必要は無いと見た
- `runUpgradeV2` の引数は 9 個になった。`fromVersion` は表示や IO の knob ではなく版の値なので、register の「`runUpgradeIO` positional-parameter creep」の行の trigger(別の display/IO knob が入るとき)には当たらない。`upgrade.go:157` の `oldManifest, oldManifest.Meta.Version` の並びは、呼び出しの所に comment が無く、理由は関数側の doc にある。次に引数を足すときに、`runUpgradeV2` の引数をまとめるか決めるとよい
- `status` は `buildScaffoldStatus(".")` で manifest を読んだあと、`printScaffoldVersionNotice` がもう一度 `.ralph/manifest.toml` を読む。計画は「scaffold の節と同じく `.` を読む」と決めていて、競合の恐れも無いので、直さない

## Tech debt identified

| Debt item | Impact | Why deferred | Trigger to pay down | Related plan/report |
| --- | --- | --- | --- | --- |
| (なし) | - | - | - | - |

_(行は足していないので `docs/tech-debt/` への追記もしない。)_ 計画が範囲の外としたもの(director の skill と headless の prompt は、この知らせを見たときの動きを決めていない。`ralph doctor` の `checkManifestVersion` は文字列一致の warn のままで、雛形のほうが新しいときも `run 'ralph upgrade'` と言う)は、計画が PR の Known gaps に書くと決めている。register に行を足すかは `/pr` の側で決める。この diff は register のどの行も閉じていない。

## Recommendation

- Merge: yes(CRITICAL 0、HIGH 0、MEDIUM 0、LOW 4)
- Follow-ups: L1 から L4 はどれも直さなくても動作は変わらない。直すなら 1 つの小さな commit で足りる(L1 は `orgScaffoldRoot` の先頭の置き換え、L2 は help の定数と README の 1 文、L3 は comment の 1 行、L4 は体裁)。直した場合は、パイプラインの規則どおり `/self-review` から回し直す。直さず進める場合は、L1 と L2 を PR の Known gaps か follow-up に書く
