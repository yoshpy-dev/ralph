# Test report: codex-verified-auto-merge-on

- Date: 2026-10-28(cycle 2 は UTC 17:40〜18:05、cycle 1 は 16:14〜16:45)
- Plan: `docs/plans/active/2026-10-28-codex-verified-auto-merge-on.md`(Test plan、AC5、AC6 のテストの部分)
- Tester: tester サブエージェント(Claude Opus 5.5)。pipeline の cycle 2(cross-review cycle 1 の ACTION_REQUIRED を直した F2 のあとの再実行)
- Scope: HEAD 90f5c80a(merge-base は origin/main の 09dc0260)。behavioral test だけを実行した(静的解析は /verify の担当)。環境は macOS(Darwin 25.5.0、arm64)、go1.27.3、git 2.50.1
- cycle 1(9d78a1c3 の報告、HEAD 473bc790)のあとの変更: F2 aea6b494(`internal/cli/org.go` の `resolveOrgConfig` が読めない `./ralph.toml` をエラーにする、`org_test.go` と `status_test.go` のテスト)、a91f6694(コメントだけ)、aa4ec8d3(origin/main の merge。#282 init-pack-choice の `internal/cli/init.go` と `tests/test-check-template.sh` を含む)、記録の更新。`internal/config` と `templates/base/ralph.toml` は変わっていない(`git diff 9d78a1c3..HEAD -- internal/config templates/base/ralph.toml` が空)
- Evidence: `docs/evidence/test-2026-10-28-codex-verified-auto-merge-on.log`(cycle 2 の `run-test.sh` の出力、名指しの実行の `-v` の出力、mutant ごとの差分と出力。末尾に cycle 1 の evidence をそのまま付けた)。gitignore の対象で、コミットしない
- 追跡しているファイルは、この報告と insight event のほかは変えていない。mutant はすべて scratchpad の写しを `go test -overlay` で差し替えて走らせた。前後の `git status --porcelain --ignored` は、`run-test.sh` が使う `.harness/cache/go-build/` を除いて同じだった

## Test execution(cycle 2)

| # | Suite / Command | Tests | Passed | Failed | Skipped | Duration |
| --- | --- | --- | --- | --- | --- | --- |
| 1 | `./scripts/run-test.sh`(scope は `full fallback (unclassified:templates/base/ralph.toml)`、pack は golang) | shell 45 本 + Go 9 パッケージ | shell 45 本(4,768 件)、Go 9 パッケージ | 0 | shell 1 件(下) | rc 0、956 秒(17:40:28〜17:56:24) |
| 2 | 1 で `(cached)` だった 5 パッケージを `go test -count=1` で | 5 パッケージ | 5 | 0 | 0 | 0.7〜3.3 秒 |
| 3 | `go test ./internal/config/ -count=1 -v` | トップレベル 60(サブテスト 145) | 60(145) | 0 | 0 | ok、0.605 秒 |
| 4 | `go test ./internal/org/ -count=1 -v -run '<下の 7 本>'` | 7(サブテスト 45) | 7(45) | 0 | 0 | ok、1.125 秒 |
| 5 | `go test ./internal/cli/ -count=1 -v -run '<下の 10 本>'` | 10(サブテスト 48) | 10(48) | 0 | 0 | ok、16.135 秒 |
| 6 | mutant 7 個と対照 2 回(下の「F2 の判別」「AC5 の変異」「そのほかの変異」) | 7 | 7 個とも期待どおり落ちた | - | - | 1 回 1〜3 秒 |
| 7 | `go test -cover ./internal/config/ -count=1` | - | ok | 0 | 0 | 0.625 秒 |

1 の Go の行は次のとおり。

```
?   	github.com/yoshpy-dev/ralph	[no test files]
ok  	github.com/yoshpy-dev/ralph/cmd/ralph	0.948s
ok  	github.com/yoshpy-dev/ralph/internal/cli	330.888s
ok  	github.com/yoshpy-dev/ralph/internal/config	1.490s
ok  	github.com/yoshpy-dev/ralph/internal/insights	(cached)
ok  	github.com/yoshpy-dev/ralph/internal/org	425.851s
ok  	github.com/yoshpy-dev/ralph/internal/org/driver	(cached)
ok  	github.com/yoshpy-dev/ralph/internal/org/protocol	(cached)
ok  	github.com/yoshpy-dev/ralph/internal/scaffold	(cached)
ok  	github.com/yoshpy-dev/ralph/internal/upgrade	(cached)

==> All verifiers passed.
```

- `internal/cli`、`internal/org`、`internal/config`、`cmd/ralph` は実際に走った。`(cached)` の 5 パッケージは、cycle 1 から入力が変わらず前回の通過が再利用されたもので、2 で `-count=1` で走らせ直してすべて ok だった(insights 0.691s、org/driver 2.259s、org/protocol 1.135s、scaffold 2.362s、upgrade 3.305s)
- shell の 45 本は `tests/test-*.sh` の数と同じ。要約行(`PASS: n`、`PASS: n / n` が 28 本、`FAIL: 0` が 28 本)は cycle 1 と `diff` で 1 行も違わなかった。4,768 件は、行頭が `PASS` の行 4,796 から要約行 28 を引いた数。#282 が変えた `tests/test-check-template.sh` は失敗のときの文だけが変わり、通ったときの行は同じ
- SKIP は `tests/test-secret-scan-branch.sh` の「the real git older than 2.41 case (git 2.50.1 is 2.41 or later; the stubbed old git covers it)」の 1 件で、手元の git の版による条件つきの skip
- 1 分の load average は、始まりが 58.58、終わりが 6.27。始まりのころはほかのセッションの処理と重なっていた
- 既知の不安定なテスト `TestStopDirectorProcess_AProcessThatIgnoresSIGTERMIsNotEndedInTime` も 30 分の timeout も当たらず、1 回目で通ったので打ち直していない
- 4 の 7 本は `TestPermissionArgsForDriver_Codex_FailClosed`、`TestPermissionArgsForDriver_Codex_VerifiedUnlocksMapping`、`TestOrgSpawn_Codex_ModelObservation_Found`、`TestOrgSpawn_Codex_ModelObservation_NotFoundAfterTimeout`、`TestOrgSpawn_Codex_ModelObservation_ReadError_DistinctReason`、`TestStartDirector_PermissionModeOfRoleDirector`、`TestSpawnPrecheckErr_MatchesSpawn`
- 5 の 10 本は cycle 1 の 9 本(`TestOrgSpawn_CLI_EnvelopeRejection_NoModelWarning`、`TestOrgDirector_StatusShowsAutoMerge`、`TestOrgDirector_AutoMergeFromTheMainWorktreeRalphToml`、`TestOrgMerge_AutoMergeComesFromTheMainWorktreeRalphToml`、`TestStatusCmd_DirectorLineShowsAutoMerge`、`TestStatusCmd_DirectorLinePerState`、`TestStatusCmd_DirectorFallsBackToTheDefaultWhenRalphTomlDoesNotLoad`、`TestStatusCmd_DirectorEnabledFromTheMainWorktree`、`TestRunDoctorOpts_CodexSandboxCheck_HermeticByDefault`)と、F2 の `TestResolveOrgConfig_UnreadableRalphTomlIsAnErrorNotTheDefaults`

## F2 の判別

F2 で、`resolveOrgConfig`(`internal/cli/org.go:613`)は `--config` がないとき `ralph.toml` を `config.Load` に渡すようになった。`config.Load` は `fs.ErrNotExist` だけを「ファイルがない」として既定に倒し、ほかの読み込みのエラーは返す。新しいテストは次の 2 つ。

- `TestResolveOrgConfig_UnreadableRalphTomlIsAnErrorNotTheDefaults`(`internal/cli/org_test.go:1133`): ファイルがなければ既定、false と書いた読めるファイルなら false、権限のないディレクトリの中を指す symlink と mode 000 のファイルはどちらも `fs.ErrPermission`
- `TestStatusCmd_DirectorFallsBackToTheDefaultWhenRalphTomlDoesNotLoad`(`internal/cli/status_test.go:1193`)に足した「cannot be read」の軸。`auto_merge = false` と書いたファイルを権限のないディレクトリの奥に置き、`ralph status` が警告を 1 行出して auto_merge を false と出すことを、`--state-dir` を渡す場合と main の checkout の場合のそれぞれで確かめる(サブテストは「does not parse」と合わせて 4 件)

どちらも root では skip する(`os.Geteuid() == 0`)。今回は root ではないので、5 の `-v` の出力に SKIP はなく、全サブテストが走った。

| ID | 変更 | 落ちたテスト | 通ったまま |
| --- | --- | --- | --- |
| f2old | `org.go` の `resolveOrgConfig` を F2 の前の `os.Stat` の分岐に戻す(`"os"` の import も戻す) | `TestResolveOrgConfig_…/a_symlink_into_a_directory_with_no_permissions`、`TestStatusCmd_DirectorFallsBack…` の「cannot be read」2 件 | `TestResolveOrgConfig_…/a_file_with_no_permissions`、「does not parse」2 件 |
| f2b | `config.go` の `Load` が `fs.ErrPermission` も「ファイルがない」として既定に倒す | `TestResolveOrgConfig_…` の 2 件とも、「cannot be read」2 件 | 「does not parse」2 件 |

f2old の主な出力は次のとおり。

```
org_test.go:1167: resolveOrgConfig("") = codex_verified true, auto_merge true, <nil>; want a permission error, not the defaults
status_test.go:1235: director = {Enabled:true AutoMerge:true State:not_attached …}; want enabled, auto_merge false, not_attached
status_test.go:1238: status [--json] stderr = ""; want exactly one load warning ending "…auto_merge as false (not the default), because ralph.toml could not be read\n"
```

- 変えない対照(`go test ./internal/cli/ -run '<2 本>' -count=1 -v`)は ok(2.076 秒)だった
- 古い分岐を判別するのは symlink のサブテストで、mode 000 のファイルのサブテストは f2old でも通る。mode 000 のファイルは `os.Stat` が通り、そのあとの `config.Load` がエラーを返すので、古いコードでもエラーになるためで、テストのコメントもそう書いている。mode 000 のサブテストは f2b を落とし、`config.Load` が `fs.ErrNotExist` だけを既定に倒す規則を固定している

## AC6: false の動きのテスト

`go test ./internal/config/ ./internal/org/ ./internal/cli/` は 1 の中で通った。D2 で false を明示して残したテストが残っていて通ることを、3〜5 の `-v` の出力で確かめた。

| 動き | テスト | 結果 |
| --- | --- | --- |
| `codex_verified = false` では codex の autonomous と edits を拒む | `internal/org/permissions_test.go` の `TestPermissionArgsForDriver_Codex_FailClosed`(サブテスト 4、「autonomous/edits is rejected fail-closed when codex_verified is false」を含む) | PASS |
| 同じ拒否を CLI の spawn で | `internal/cli/org_test.go` の `TestOrgSpawn_CLI_EnvelopeRejection_NoModelWarning`。fixture の `writeCodexOrgConfig` が `codex_verified = false` を書く | PASS。mutant g で、その 1 行を消すと spawn が通ってこのテストが落ちる(cycle 2 でも確かめた) |
| false では headless の codex の director も guarded に限る | `TestStartDirector_PermissionModeOfRoleDirector` の「codex autonomous before codex_verified」「codex edits before codex_verified」「codex guarded before codex_verified」 | PASS |
| false の codex の leader を spawn の前の確認でも拒む | `TestSpawnPrecheckErr_MatchesSpawn/an_unverified_codex_leader` | PASS |
| `CodexVerified: false` を明示した fixture | `codexGuardedOrg` を使う `TestOrgSpawn_Codex_ModelObservation_*` の 3 本 | PASS |
| `auto_merge = false` の director の行(merge を人に任せる) | `internal/cli/status_test.go` の `TestStatusCmd_DirectorLineShowsAutoMerge` のうち「auto_merge = false」の 6 件(not attached、attached、disabled のそれぞれで orgs false と true) | PASS |
| 同じ行を `ralph org director status` で | `internal/cli/org_director_test.go` の `TestOrgDirector_StatusShowsAutoMerge`(off、attached の off、disabledOff) | PASS |
| main の worktree の `ralph.toml` が優先される | `TestOrgDirector_AutoMergeFromTheMainWorktreeRalphToml`(6 件。「no main ralph.toml, the worktree says false」を含む)、`TestOrgMerge_AutoMergeComesFromTheMainWorktreeRalphToml`(2 件) | PASS |
| false と書いた `./ralph.toml` が読めないとき、既定の true にしない | `TestResolveOrgConfig_UnreadableRalphTomlIsAnErrorNotTheDefaults`(2 件) | PASS。f2old と f2b で落ちる |
| `ralph status` が `./ralph.toml` を読めない(壊れている、読む権限がない)ときは auto_merge を false と出す | `TestStatusCmd_DirectorFallsBackToTheDefaultWhenRalphTomlDoesNotLoad`(4 件、それぞれ文字と `--json`) | PASS。f1 で 4 件とも落ち、f2old と f2b で「cannot be read」の 2 件が落ちる |
| `ralph status` が main の worktree の `ralph.toml` を読めないときも false と出す | `TestStatusCmd_DirectorEnabledFromTheMainWorktree`(文字と `--json`) | PASS。f2 で落ちる |

AC6 の `./scripts/check-sync.sh` と `./scripts/check-skill-sync.sh` は drift の検査で、tester の担当ではないので走らせていない。verify の報告(`docs/reports/verify-2026-10-28-codex-verified-auto-merge-on.md`)が、cycle 2 の HEAD aa4ec8d3 で単独の実行がどちらも rc 0 だったと記録している。aa4ec8d3 から HEAD 90f5c80a までの変更は verify の報告と記録だけ。

## AC5 の変異

### cycle 2 の打ち直し

`internal/config` とテンプレートは cycle 1 から変わっていないので、代表の 2 つを今の HEAD で当て直した。変えない対照(`go test ./internal/config/ -count=1`)は ok(0.620 秒)。

| ID | 変更 | 落ちたテスト | 通ったまま |
| --- | --- | --- | --- |
| a1 | `org.permissions.codex_verified` の `valueDrift` の行を消す | `EachKeyIsComparedOnBothSides/org.permissions.codex_verified`(3 つの変異のどれも `drift = []`)、`DetectsChangedAndMissingKeys` の codex_verified の削除 | `TestDefaultsLockStep` |
| b | `Default()` 側の取り違え。codex_verified を `d.Director.AutoMerge` と、auto_merge を `d.Permissions.CodexVerified` と比べる(どちらも既定 true) | `EachKeyIsComparedOnBothSides` の 2 キーの「Default() changed」だけ | `TestDefaultsLockStep`、`DetectsChangedAndMissingKeys` |

どちらも cycle 1 と同じテストが同じ行で落ちた。

### cycle 1 の結果(HEAD 473bc790)

cycle 1 では次の 6 個を当て、すべて落ちた。`internal/config` とテンプレートは変わっていないので、結果は今の HEAD にもそのまま当てはまる。

| ID | 変更 | `TestOrgTemplateDrift_EachKeyIsComparedOnBothSides` | ほかに落ちたテスト |
| --- | --- | --- | --- |
| a1 | `org.permissions.codex_verified` の `valueDrift` の行を消す | FAIL | `DetectsChangedAndMissingKeys` の codex_verified の削除。`TestDefaultsLockStep` は PASS |
| a2 | `org.director.auto_merge` の行を消す | FAIL | `DetectsChangedAndMissingKeys` の auto_merge の削除。`TestDefaultsLockStep` は PASS |
| b | `Default()` 側の取り違え(codex_verified と auto_merge) | FAIL(2 キーの「Default() changed」) | なし |
| c | テンプレート側の取り違え。codex_verified の行に `p.Director.AutoMerge`、auto_merge の行に `p.Permissions.CodexVerified` | FAIL(2 キーの「template value changed」と「template key removed」) | `DetectsChangedAndMissingKeys` の 2 つの削除。`TestDefaultsLockStep` は PASS |
| d1 | probe の toml のタグの取り違え(同じ表)。`Director.Enabled` を `auto_merge`、`Director.AutoMerge` を `enabled` に(どちらも既定 true) | FAIL(`org.director.enabled` と `org.director.auto_merge` のテンプレート側) | `DetectsChangedAndMissingKeys` の auto_merge の削除。`TestDefaultsLockStep` は PASS |
| d2 | probe の toml のタグの取り違え(表をまたぐ)。`Permissions.CodexVerified` を `auto_merge`、`Director.AutoMerge` を `codex_verified` に | FAIL(414 行の前提「the real template must match Default() first」で止まる) | `TestDefaultsLockStep/org`(2 キーとも missing)、`DetectsChangedAndMissingKeys` の 6 件 |

- b を落としたのは新しいテストだけで、計画の D3 が (i) の変異で捕まえるとした `Default()` 側の取り違えにあたる
- d2 は表をまたいで 2 つのキーが両方とも読めなくなるので、前提の照合で止まり、キーの集合の確認(461 行)には届かない。キーの集合の確認は、下の d3 と d4 で確かめた

## そのほかの変異

cycle 2 で当て直したのは f1、f2、g の 3 つ(`status_test.go` と `org_test.go` が F2 で変わったため)。ほかは cycle 1 の結果で、対象のファイルは cycle 1 から変わっていない。

| ID | ファイル | 変更 | 落ちたテスト | 実行 |
| --- | --- | --- | --- | --- |
| d3 | `defaults_sync_test.go` | probe に `ExtraKey *int toml:"extra_key"` を足す(比較の行も表の行も足さない) | `EachKeyIsComparedOnBothSides` のキーの集合の確認(463 行)。`TestDefaultsLockStep` は PASS | cycle 1 |
| d4 | `defaults_sync_test.go` | 表から `org.approval.mode` の行を消す | `EachKeyIsComparedOnBothSides` のキーの集合の確認(461 行) | cycle 1 |
| p1 | `defaults_sync_test.go` | `org.deadman_minutes` を `d.MaxOrgs` と比べる(どちらも既定 10)。drop-org-shell-defaults の報告の M1 | `EachKeyIsComparedOnBothSides` の `org.max_orgs` と `org.deadman_minutes` だけ | cycle 1 |
| p3 | `defaults_sync_test.go` | `org.watchdog.watcher_enabled` を `d.Director.Enabled` と比べる(どちらも既定 true)。同じ報告の M3 | `EachKeyIsComparedOnBothSides` の `watcher_enabled` と `director.enabled` だけ | cycle 1 |
| p5 | `defaults_sync_test.go` | `org.approval.mode` の `valueDrift` の行を消す。同じ報告の M5 | `EachKeyIsComparedOnBothSides` の `org.approval.mode` だけ | cycle 1 |
| p6 | `defaults_sync_test.go` | `org.director.ack_minutes` を `d.DeadmanMinutes` と比べる(どちらも既定 10) | `EachKeyIsComparedOnBothSides` の `deadman_minutes` と `ack_minutes` だけ | cycle 1 |
| e1 | `internal/config/config.go` | `Default()` の `AutoMerge: true` を `false` に | `TestLoad_CodexVerifiedAndAutoMergeDefaultToTrue`(3 件)、`TestDefault_OrgDirector`、`TestLoad_OrgDirectorMissingSectionBackfillsDefault`、`TestLoad_OrgDirectorRoundTrip`(3 件)、`TestDefaultsLockStep/org`、`DetectsChangedAndMissingKeys`(6 件)、`EachKeyIsComparedOnBothSides`(前提で停止) | cycle 1 |
| e2 | `internal/config/config.go` | `Default()` の `CodexVerified: true` を `false` に | `TestLoad_CodexVerifiedAndAutoMergeDefaultToTrue`(3 件)、`TestDefault_OrgPermissions`、`TestLoad_OrgPermissionsMissingSectionBackfillsDefault`、`TestDefaultsLockStep/org`、`DetectsChangedAndMissingKeys`(6 件)、`EachKeyIsComparedOnBothSides`(前提で停止) | cycle 1 |
| f1 | `internal/cli/status.go` | `statusOrgConfig` の fallback の `fallback.Director.AutoMerge = false` を消す | `TestStatusCmd_DirectorFallsBack…` の 4 件(director の行が `auto_merge = true` の文、`--json` が `AutoMerge:true`、stderr が「auto_merge as true」)。`TestStatusCmd_DirectorEnabledFromTheMainWorktree` は PASS | cycle 2 |
| f2 | `internal/cli/status.go` | `statusDirectorConfig` の fallback の `fallback.AutoMerge = false` を消す | `TestStatusCmd_DirectorEnabledFromTheMainWorktree`。`TestStatusCmd_DirectorFallsBack…` の 4 件は PASS | cycle 2 |
| g | `internal/cli/org_test.go` | `writeCodexOrgConfig` の `codex_verified = false` の行を消す | `TestOrgSpawn_CLI_EnvelopeRejection_NoModelWarning`(spawn が `spawned seat "seat-1"` で通る) | cycle 2 |

- f1 と f2 はそれぞれ別のテストだけを落とすので、2 つの fallback の行はどちらも独立に固定されている
- p1、p3、p5 は drop-org-shell-defaults の報告の Test gaps で生き残った mutant で、いまは新しいテストだけが落とす。同じ報告の M2 は b の片方にあたる

## 端の場合

| Test plan の端の場合 | テスト | 結果 |
| --- | --- | --- |
| `ralph.toml` がない → 2 つとも true | `TestLoad_CodexVerifiedAndAutoMergeDefaultToTrue/no_ralph.toml`、`TestStatusCmd_DirectorLineShowsAutoMerge/not_attached,_no_ralph.toml`(orgs false と true)、`TestResolveOrgConfig_…` の最初の確認(ファイルがなければ既定) | PASS。e1 と e2 で落ちる |
| キーを書かない `ralph.toml` → true | `TestLoad_CodexVerifiedAndAutoMergeDefaultToTrue` の「a ralph.toml without [org]」「both tables without the keys」、`TestLoad_OrgDirectorMissingSectionBackfillsDefault`、`TestLoad_OrgPermissionsMissingSectionBackfillsDefault` | PASS。e1 と e2 で落ちる |
| `codex_verified = false` を明示 → codex は guarded だけ | 上の AC6 の表の codex の 5 行 | PASS。g で落ちる |
| `auto_merge = false` を明示 → director は merge を人に任せる(行が「merges are left to the human ([org.director].auto_merge = false)」) | 上の AC6 の表の auto_merge の 3 行 | PASS |
| テンプレートの 2 行を false に書き換える(AC7) | `TestLoad_TemplateWithBothKeysFalse`(2 つが false で、ほかの値はテンプレートのまま) | PASS |
| `ralph.toml` が壊れている、または読めない(self-review の M1、cross-review の F2) | `TestResolveOrgConfig_…`、`TestStatusCmd_DirectorFallsBack…`、`TestStatusCmd_DirectorEnabledFromTheMainWorktree` | PASS。f1、f2、f2old、f2b で落ちる |

## Coverage

- Statement: `internal/config` 96.9%(`go test -cover`、cycle 1 と同じ)。`internal/cli` と `internal/org` はこの実行では測っていない。cycle 2 で変わった製品コードは `resolveOrgConfig` の分岐の削除(と #282 の merge)で、`resolveOrgConfig` の 3 つの経路(ファイルがない、読める、読めない)は `TestResolveOrgConfig_…` が通す
- Branch: `status.go` の 2 つの fallback の分岐と、`config.Load` の `fs.ErrNotExist` の分岐は、f1、f2、f2b でそれぞれのテストが実行して値を確かめていることを確かめた。`orgTemplateDrift` の 19 キーの配線は、新しいテストで行ごとに 3 つの変異が当たる
- Function: `orgTemplateDrift`、`valueDrift`、`listDrift`、`tomlKeys`、`editTemplateKey`(書き換えと削除、複数行の配列の `model_pool`)を 3 の実行で通した
- Notes: cycle 2 で当てた 7 個(f2old、f2b、a1、b、f1、f2、g)と cycle 1 の 17 個はすべて落ちた。生き残りはない

## Failure analysis

| Test | Error | Root cause | Proposed fix |
| --- | --- | --- | --- |
| なし | - | - | - |

## Regression checks

| Previously broken behavior | Status | Evidence |
| --- | --- | --- |
| false と書いた `./ralph.toml` が読めないと、`resolveOrgConfig` が既定の true を返し、`ralph status` も auto_merge を true と出す(cross-review cycle 1 の ACTION_REQUIRED、F2) | 直っている | `TestResolveOrgConfig_…` と「cannot be read」の 2 件が PASS。f2old(F2 の前のコード)で落ちる |
| #273 の Known gaps (a): 既定値が同じキーどうしの比較の取り違えと、1 キーの比較の削除が照合を素通りする | 直っている | b(cycle 2 でも確認)、p1、p3、p5、p6 を新しいテストだけが落とす |
| #273 の Known gaps (b): 照合するキーの数がコメントにしかない | 直っている | d3(probe にキーを足す)と d4(表の行を消す)をキーの集合の確認が落とす |
| `ralph status` が読めない `ralph.toml` のとき「人に聞かずに merge する」と出す(self-review の M1) | 直っている | `TestStatusCmd_DirectorFallsBack…` と `TestStatusCmd_DirectorEnabledFromTheMainWorktree` が PASS、f1 と f2 で落ちる |
| 古い既定 false に頼っていた codex の拒否のテストが、既定の変更で空振りする | 起きていない | g: false の明示を消すと `TestOrgSpawn_CLI_EnvelopeRejection_NoModelWarning` が落ちる |
| shell の既存のテスト(#282 の merge を含む) | 通る | 1 の shell 45 本、FAIL 0、要約行は cycle 1 と同じ |

## Test gaps

- F2 の 2 つのテストは root では skip する。今回の実行は root ではないので走った。root で走る環境(Docker の既定など)では、読めない `ralph.toml` の確認が抜ける
- `-race` では走らせていない。cycle 2 の製品コードの変更は `resolveOrgConfig` の分岐の削除で、並行の処理には触れていない
- 本物の codex の座席の spawn(`--sandbox workspace-write --ask-for-approval never` で動くか)と、director の本物の merge は走らせていない。どちらも stub と fixture のテストだけで、計画の R1 のとおり実機の確認は範囲の外
- `internal/cli` と `internal/org` の statement coverage は測っていない
- この報告で走らせたのは macOS の go1.27.3 だけで、CI の Linux では走らせていない

## Verdict

- Pass: `./scripts/run-test.sh`(rc 0、shell 45 本 4,768 件、Go 9 パッケージ。`internal/config`、`internal/org`、`internal/cli` を含む。`(cached)` の 5 パッケージも `-count=1` で通った)、false の動きのテストの名指しの実行(`internal/config` 60 本、`internal/org` 7 本、`internal/cli` 10 本)、F2 の判別(f2old と f2b で新しいテストが落ちる)、AC5 の代表の変異 2 個の打ち直し、そのほかの変異 3 個の打ち直し、Test plan の端の場合 6 つ
- Fail: なし
- Blocked: なし
- Verdict: pass
