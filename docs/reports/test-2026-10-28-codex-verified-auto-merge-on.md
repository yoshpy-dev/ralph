# Test report: codex-verified-auto-merge-on

- Date: 2026-10-28(UTC 16:14〜16:45)
- Plan: `docs/plans/active/2026-10-28-codex-verified-auto-merge-on.md`(Test plan、AC5、AC6 のテストの部分)
- Tester: tester サブエージェント(Claude Opus 5.5)。pipeline の cycle 1
- Scope: HEAD 473bc790(base ae64da70)。behavioral test だけを実行した(静的解析は /verify の担当)。環境は macOS(Darwin 25.5.0、arm64)、go1.27.3、git 2.50.1
- Evidence: `docs/evidence/test-2026-10-28-codex-verified-auto-merge-on.log`(`run-test.sh` の出力、名指しの実行の `-v` の出力、mutant ごとの差分と出力)。gitignore の対象で、コミットしない
- 追跡しているファイルは、この報告と insight event のほかは変えていない。mutant はすべて scratchpad の写しを `go test -overlay` で差し替えて走らせた。前後の `git status --porcelain --ignored` の差は、`run-test.sh` が作った `.harness/cache/go-build/` の項目と上の evidence の log だけだった

## Test execution

| # | Suite / Command | Tests | Passed | Failed | Skipped | Duration |
| --- | --- | --- | --- | --- | --- | --- |
| 1 | `./scripts/run-test.sh`(scope は `full fallback (unclassified:templates/base/ralph.toml)`、pack は golang) | shell 45 本 + Go 9 パッケージ | shell 45 本(4,768 件)、Go 9 パッケージ | 0 | shell 1 件(下) | rc 0、1,474 秒(16:14:32〜16:39:06) |
| 2 | `go test ./internal/config/ -count=1 -v` | トップレベル 60 | 60 | 0 | 0 | ok、0.569 秒 |
| 3 | `go test ./internal/org/ -count=1 -v -run '<下の 7 本>'` | 7 | 7 | 0 | 0 | ok、1.622 秒 |
| 4 | `go test ./internal/cli/ -count=1 -v -run '<下の 9 本>'` | 9 | 9 | 0 | 0 | ok、20.054 秒 |
| 5 | mutant 17 個と対照 2 回(下の「AC5 の変異」「そのほかの変異」) | 17 | 17 個とも期待どおり落ちた | - | - | 1 回 1〜3 秒 |
| 6 | `go test -cover ./internal/config/ -count=1` | - | ok | 0 | 0 | 1.160 秒 |

1 の Go の行は次のとおり。どのパッケージも `(cached)` ではなく実際に走った。

```
?   	github.com/yoshpy-dev/ralph	[no test files]
ok  	github.com/yoshpy-dev/ralph/cmd/ralph	0.951s
ok  	github.com/yoshpy-dev/ralph/internal/cli	559.499s
ok  	github.com/yoshpy-dev/ralph/internal/config	2.005s
ok  	github.com/yoshpy-dev/ralph/internal/insights	4.654s
ok  	github.com/yoshpy-dev/ralph/internal/org	653.798s
ok  	github.com/yoshpy-dev/ralph/internal/org/driver	4.968s
ok  	github.com/yoshpy-dev/ralph/internal/org/protocol	2.798s
ok  	github.com/yoshpy-dev/ralph/internal/scaffold	3.295s
ok  	github.com/yoshpy-dev/ralph/internal/upgrade	4.734s

==> All verifiers passed.
```

- shell の 45 本は `tests/test-*.sh` の数と同じ。要約行の `FAIL:` は 28 本ともすべて 0。4,768 件は、行頭が `PASS` の行 4,796 から要約行 `PASS: n` の 28 を引いた数で、drop-org-shell-defaults の報告と同じ
- SKIP は `tests/test-secret-scan-branch.sh` の「the real git older than 2.41 case (git 2.50.1 is 2.41 or later; the stubbed old git covers it)」の 1 件で、手元の git の版による条件つきの skip
- 1 分の load average は、始まりが 9.36、終わりが 5.33。同じ時間に別の worktree(`init-pack-choice`)の `run-verify.sh` が走っていた
- 既知の不安定なテスト `TestStopDirectorProcess_AProcessThatIgnoresSIGTERMIsNotEndedInTime` も、`internal/org` と `internal/cli` の 30 分の timeout も当たらず、1 回目で通ったので打ち直していない
- 3 の 7 本は `TestPermissionArgsForDriver_Codex_FailClosed`、`TestPermissionArgsForDriver_Codex_VerifiedUnlocksMapping`、`TestOrgSpawn_Codex_ModelObservation_Found`、`TestOrgSpawn_Codex_ModelObservation_NotFoundAfterTimeout`、`TestOrgSpawn_Codex_ModelObservation_ReadError_DistinctReason`、`TestStartDirector_PermissionModeOfRoleDirector`、`TestSpawnPrecheckErr_MatchesSpawn`
- 4 の 9 本は `TestOrgSpawn_CLI_EnvelopeRejection_NoModelWarning`、`TestOrgDirector_StatusShowsAutoMerge`、`TestOrgDirector_AutoMergeFromTheMainWorktreeRalphToml`、`TestOrgMerge_AutoMergeComesFromTheMainWorktreeRalphToml`、`TestStatusCmd_DirectorLineShowsAutoMerge`、`TestStatusCmd_DirectorLinePerState`、`TestStatusCmd_DirectorFallsBackToTheDefaultWhenRalphTomlDoesNotLoad`、`TestStatusCmd_DirectorEnabledFromTheMainWorktree`、`TestRunDoctorOpts_CodexSandboxCheck_HermeticByDefault`

## AC6: false の動きのテスト

`go test ./internal/config/ ./internal/org/ ./internal/cli/` は 1 の中で通った。D2 で false を明示して残したテストが残っていて通ることを、2〜4 の `-v` の出力で確かめた。

| 動き | テスト | 結果 |
| --- | --- | --- |
| `codex_verified = false` では codex の autonomous と edits を拒む | `internal/org/permissions_test.go` の `TestPermissionArgsForDriver_Codex_FailClosed`(サブテスト 4、「autonomous/edits is rejected fail-closed when codex_verified is false」を含む) | PASS |
| 同じ拒否を CLI の spawn で | `internal/cli/org_test.go` の `TestOrgSpawn_CLI_EnvelopeRejection_NoModelWarning`。fixture の `writeCodexOrgConfig` が `codex_verified = false` を書く | PASS。下の mutant g で、その 1 行を消すと spawn が通ってこのテストが落ちることを確かめた |
| false では headless の codex の director も guarded に限る | `TestStartDirector_PermissionModeOfRoleDirector` の「codex autonomous before codex_verified」「codex edits before codex_verified」「codex guarded before codex_verified」 | PASS |
| false の codex の leader を spawn の前の確認でも拒む | `TestSpawnPrecheckErr_MatchesSpawn/an_unverified_codex_leader` | PASS |
| `CodexVerified: false` を明示した fixture | `codexGuardedOrg` を使う `TestOrgSpawn_Codex_ModelObservation_*` の 3 本 | PASS |
| `auto_merge = false` の director の行(merge を人に任せる) | `internal/cli/status_test.go` の `TestStatusCmd_DirectorLineShowsAutoMerge` のうち「auto_merge = false」の 6 件(not attached、attached、disabled のそれぞれで orgs false と true) | PASS |
| 同じ行を `ralph org director status` で | `internal/cli/org_director_test.go` の `TestOrgDirector_StatusShowsAutoMerge`(off、attached の off、disabledOff) | PASS |
| main の worktree の `ralph.toml` が優先される | `TestOrgDirector_AutoMergeFromTheMainWorktreeRalphToml`(6 件。「no main ralph.toml, the worktree says false」を含む)、`TestOrgMerge_AutoMergeComesFromTheMainWorktreeRalphToml`(2 件) | PASS |
| `ralph status` が `./ralph.toml` を読めないときは auto_merge を false と出す | `TestStatusCmd_DirectorFallsBackToTheDefaultWhenRalphTomlDoesNotLoad`(2 件、それぞれ文字と `--json`) | PASS。mutant f1 で落ちる |
| `ralph status` が main の worktree の `ralph.toml` を読めないときも false と出す | `TestStatusCmd_DirectorEnabledFromTheMainWorktree`(文字と `--json`) | PASS。mutant f2 で落ちる |

AC6 の `./scripts/check-sync.sh` と `./scripts/check-skill-sync.sh` は drift の検査で、tester の担当ではないので走らせていない。verify の報告(`docs/reports/verify-2026-10-28-codex-verified-auto-merge-on.md`)が、HEAD 7ca29c41 で単独の実行がどちらも rc 0 だったと記録している。7ca29c41 から HEAD までの変更は、計画、verify の報告、insight event の 3 つだけ。

## AC5 の変異

`internal/config/defaults_sync_test.go` の写しを変え、`go test -overlay <json> ./internal/config/ -count=1` を実行した。置き換えは、写しの中に 1 回だけ現れる字面を perl で変えた(0 回か 2 回以上なら止める)。変えない対照は ok(0.581 秒)だった。「ほかに落ちたテスト」の列は、新しいテスト以外で同じ mutant を落としたものを書く。

| ID | 変更 | `TestOrgTemplateDrift_EachKeyIsComparedOnBothSides` | ほかに落ちたテスト |
| --- | --- | --- | --- |
| a1 | `org.permissions.codex_verified` の `valueDrift` の行を消す | FAIL(`org.permissions.codex_verified`、3 つの変異のどれも `drift = []`) | `DetectsChangedAndMissingKeys` の codex_verified の削除。`TestDefaultsLockStep` は PASS |
| a2 | `org.director.auto_merge` の行を消す | FAIL(`org.director.auto_merge`、同じく 3 つとも) | `DetectsChangedAndMissingKeys` の auto_merge の削除。`TestDefaultsLockStep` は PASS |
| b | `Default()` 側の取り違え。codex_verified を `d.Director.AutoMerge` と、auto_merge を `d.Permissions.CodexVerified` と比べる(どちらも既定 true) | FAIL(2 キーの「Default() changed」) | なし |
| c | テンプレート側の取り違え。codex_verified の行に `p.Director.AutoMerge`、auto_merge の行に `p.Permissions.CodexVerified` | FAIL(2 キーの「template value changed」と「template key removed」) | `DetectsChangedAndMissingKeys` の 2 つの削除。`TestDefaultsLockStep` は PASS |
| d1 | probe の toml のタグの取り違え(同じ表)。`Director.Enabled` を `auto_merge`、`Director.AutoMerge` を `enabled` に(どちらも既定 true) | FAIL(`org.director.enabled` と `org.director.auto_merge` のテンプレート側の 2 つ) | `DetectsChangedAndMissingKeys` の auto_merge の削除。`TestDefaultsLockStep` は PASS |
| d2 | probe の toml のタグの取り違え(表をまたぐ)。`Permissions.CodexVerified` を `auto_merge`、`Director.AutoMerge` を `codex_verified` に | FAIL(414 行の前提「the real template must match Default() first」で止まる) | `TestDefaultsLockStep/org`(2 キーとも missing)、`DetectsChangedAndMissingKeys` の 6 件 |

b、c、d1 の主な出力は次のとおり。

```
b:  defaults_sync_test.go:482: Default() changed: drift = ["org.director.auto_merge: template = true, Default() = false"], want one line starting with "org.permissions.codex_verified: template = "
c:  defaults_sync_test.go:488: template value changed: drift = ["org.director.auto_merge: template = false, Default() = true"], want one line starting with "org.permissions.codex_verified: template = "
d1: defaults_sync_test.go:488: template value changed: drift = ["org.director.auto_merge: template = false, Default() = true"], want one line starting with "org.director.enabled: template = "
```

- b を落としたのは新しいテストだけだった。計画の D3 が (i) の変異で捕まえるとした `Default()` 側の取り違えにあたる
- a1、a2、c、d1 は、`TestDefaultsLockStep` が通るので、実テンプレートの照合だけでは見つからない。新しいテストはどれも落とした
- d2 は表をまたいで 2 つのキーが両方とも読めなくなるので、前提の照合で止まり、キーの集合の確認(461 行)には届かない。キーの集合の確認そのものは、下の d3 と d4 で確かめた

## そのほかの変異

| ID | ファイル | 変更 | 落ちたテスト |
| --- | --- | --- | --- |
| d3 | `defaults_sync_test.go` | probe に `ExtraKey *int toml:"extra_key"` を足す(比較の行も表の行も足さない) | `EachKeyIsComparedOnBothSides` のキーの集合の確認(463 行の `row keys = … want the toml keys of orgTemplateProbe`)。`TestDefaultsLockStep` は PASS |
| d4 | `defaults_sync_test.go` | 表から `org.approval.mode` の行を消す | `EachKeyIsComparedOnBothSides` のキーの集合の確認(461 行) |
| p1 | `defaults_sync_test.go` | `org.deadman_minutes` を `d.MaxOrgs` と比べる(どちらも既定 10)。drop-org-shell-defaults の報告の M1 | `EachKeyIsComparedOnBothSides` の `org.max_orgs` と `org.deadman_minutes` だけ |
| p3 | `defaults_sync_test.go` | `org.watchdog.watcher_enabled` を `d.Director.Enabled` と比べる(どちらも既定 true)。同じ報告の M3 | `EachKeyIsComparedOnBothSides` の `watcher_enabled` と `director.enabled` だけ |
| p5 | `defaults_sync_test.go` | `org.approval.mode` の `valueDrift` の行を消す。同じ報告の M5 | `EachKeyIsComparedOnBothSides` の `org.approval.mode` だけ |
| p6 | `defaults_sync_test.go` | `org.director.ack_minutes` を `d.DeadmanMinutes` と比べる(どちらも既定 10) | `EachKeyIsComparedOnBothSides` の `deadman_minutes` と `ack_minutes` だけ |
| e1 | `internal/config/config.go` | `Default()` の `AutoMerge: true` を `false` に | `TestLoad_CodexVerifiedAndAutoMergeDefaultToTrue`(3 件とも `director.auto_merge = false, want true`)、`TestDefault_OrgDirector`、`TestLoad_OrgDirectorMissingSectionBackfillsDefault`、`TestLoad_OrgDirectorRoundTrip`(3 件)、`TestDefaultsLockStep/org`、`DetectsChangedAndMissingKeys`(6 件)、`EachKeyIsComparedOnBothSides`(前提で停止) |
| e2 | `internal/config/config.go` | `Default()` の `CodexVerified: true` を `false` に | `TestLoad_CodexVerifiedAndAutoMergeDefaultToTrue`(3 件)、`TestDefault_OrgPermissions`、`TestLoad_OrgPermissionsMissingSectionBackfillsDefault`、`TestDefaultsLockStep/org`、`DetectsChangedAndMissingKeys`(6 件)、`EachKeyIsComparedOnBothSides`(前提で停止) |
| f1 | `internal/cli/status.go` | `statusOrgConfig` の fallback の `fallback.Director.AutoMerge = false` を消す | `TestStatusCmd_DirectorFallsBackToTheDefaultWhenRalphTomlDoesNotLoad` の 2 件(director の行が `auto_merge = true` の文、`--json` が `AutoMerge:true`、stderr が「auto_merge as true」)。`TestStatusCmd_DirectorEnabledFromTheMainWorktree` は PASS |
| f2 | `internal/cli/status.go` | `statusDirectorConfig` の fallback の `fallback.AutoMerge = false` を消す | `TestStatusCmd_DirectorEnabledFromTheMainWorktree`(文字と `--json` の行、stderr)。`TestStatusCmd_DirectorFallsBack…` は PASS |
| g | `internal/cli/org_test.go` | `writeCodexOrgConfig` の `codex_verified = false` の行を消す | `TestOrgSpawn_CLI_EnvelopeRejection_NoModelWarning`(`org_test.go:1457`、spawn が `spawned seat "seat-1"` で通る) |

- f1 と f2 は `go test ./internal/cli/ -run '<3 本>'` で走らせ、変えない対照は ok(1.915 秒)だった。f1 と f2 はそれぞれ別のテストだけを落とすので、2 つの fallback の行はどちらも独立に固定されている
- p1、p3、p5 は drop-org-shell-defaults の報告の Test gaps で生き残った mutant で、いまは新しいテストだけが落とす。同じ報告の M2 は b の片方にあたる
- e1 と e2 は、AC1 のテストと `TestDefaultsLockStep` が既定の値の変更を捕まえることを示す

## 端の場合

| Test plan の端の場合 | テスト | 結果 |
| --- | --- | --- |
| `ralph.toml` がない → 2 つとも true | `TestLoad_CodexVerifiedAndAutoMergeDefaultToTrue/no_ralph.toml`、`TestStatusCmd_DirectorLineShowsAutoMerge/not_attached,_no_ralph.toml`(orgs false と true) | PASS。e1 と e2 で落ちる |
| キーを書かない `ralph.toml` → true | `TestLoad_CodexVerifiedAndAutoMergeDefaultToTrue` の「a ralph.toml without [org]」「both tables without the keys」、`TestLoad_OrgDirectorMissingSectionBackfillsDefault`、`TestLoad_OrgPermissionsMissingSectionBackfillsDefault` | PASS。e1 と e2 で落ちる |
| `codex_verified = false` を明示 → codex は guarded だけ | 上の AC6 の表の codex の 5 行 | PASS。g で落ちる |
| `auto_merge = false` を明示 → director は merge を人に任せる(行が「merges are left to the human ([org.director].auto_merge = false)」) | 上の AC6 の表の auto_merge の 3 行 | PASS |
| テンプレートの 2 行を false に書き換える(AC7) | `TestLoad_TemplateWithBothKeysFalse`(2 つが false で、ほかの値はテンプレートのまま) | PASS |
| `ralph.toml` が読めない(self-review の M1) | `TestStatusCmd_DirectorFallsBack…`、`TestStatusCmd_DirectorEnabledFromTheMainWorktree` | PASS。f1 と f2 で落ちる |

## Coverage

- Statement: `internal/config` 96.9%(`go test -cover`)。`internal/cli` と `internal/org` はこの実行では測っていない。2 つのパッケージの製品コードの変更は、`internal/cli/status.go` の fallback の 2 行と文言、ほかはコメントだけ
- Branch: `status.go` の 2 つの fallback の分岐は、f1 と f2 でそれぞれのテストが実行して値を確かめていることを確かめた。`orgTemplateDrift` の 19 キーの配線は、新しいテストで行ごとに 3 つの変異が当たる
- Function: `orgTemplateDrift`、`valueDrift`、`listDrift`、`tomlKeys`、`editTemplateKey`(書き換えと削除、複数行の配列の `model_pool`)を 2 の実行で通した
- Notes: AC5 で頼まれた 4 種の変異(6 個)と、そのほかの 11 個はすべて落ちた。生き残りはない

## Failure analysis

| Test | Error | Root cause | Proposed fix |
| --- | --- | --- | --- |
| なし | - | - | - |

## Regression checks

| Previously broken behavior | Status | Evidence |
| --- | --- | --- |
| #273 の Known gaps (a): 既定値が同じキーどうしの比較の取り違えと、1 キーの比較の削除が照合を素通りする | 直っている | b、p1、p3、p5、p6 を新しいテストだけが落とす |
| #273 の Known gaps (b): 照合するキーの数がコメントにしかない | 直っている | d3(probe にキーを足す)と d4(表の行を消す)をキーの集合の確認が落とす |
| `ralph status` が読めない `ralph.toml` のとき「人に聞かずに merge する」と出す(self-review の M1) | 直っている | `TestStatusCmd_DirectorFallsBack…` と `TestStatusCmd_DirectorEnabledFromTheMainWorktree` が PASS、f1 と f2 で落ちる |
| 古い既定 false に頼っていた codex の拒否のテストが、既定の変更で空振りする | 起きていない | g: false の明示を消すと `TestOrgSpawn_CLI_EnvelopeRejection_NoModelWarning` が落ちる。つまりこのテストは明示した false で拒否を確かめている |
| shell の既存のテスト | 通る | 1 の shell 45 本、FAIL 0 |

## Test gaps

- `-race` では走らせていない。この branch の製品コードの変更は値の既定と status の fallback の 2 行で、並行の処理には触れていない
- 本物の codex の座席の spawn(`--sandbox workspace-write --ask-for-approval never` で動くか)と、director の本物の merge は走らせていない。どちらも stub と fixture のテストだけで、計画の R1 のとおり実機の確認は範囲の外
- `internal/cli` と `internal/org` の statement coverage は測っていない
- この報告で走らせたのは macOS の go1.27.3 だけで、CI の Linux では走らせていない

## Verdict

- Pass: `./scripts/run-test.sh`(rc 0、shell 45 本 4,768 件、Go 9 パッケージ。`internal/config`、`internal/org`、`internal/cli` を含む)、false の動きのテストの名指しの実行(`internal/config` 60 本、`internal/org` 7 本、`internal/cli` 9 本)、AC5 の変異 6 個(a1、a2、b、c、d1、d2)とそのほかの 11 個がすべて落ちること、Test plan の端の場合 6 つ
- Fail: なし
- Blocked: なし
- Verdict: pass
