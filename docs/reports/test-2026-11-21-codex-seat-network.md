# Test report: codex-seat-network

- Date: 2026-11-21(実行は 2026-11-22 01:34〜02:21 JST。パイプラインの cycle 1、`cycle-count.json` は `cycle: 1`)
- Plan: `docs/plans/active/2026-11-21-codex-seat-network.md`(issue #362)
- Tester: tester サブエージェント(`/test`)
- Scope: HEAD `50a47846`(base `09d27450`、worktree は clean)。`./scripts/run-test.sh` を 1 回だけ、worktree のルートから回した。Bash tool の 10 分の上限を超えるので、perl の fork と setsid で切り離して起動し(INT・HUP・QUIT・TERM は既定の扱いに戻し、標準入出力は `/dev/null`)、`caffeinate -w` で終わりを待った。範囲は wrapper の既定の changed(`- Requested scope: changed`)で、`templates/base/ralph.toml` が言語に分類されないため `Language scope: full fallback (unclassified:templates/base/ralph.toml)` になった。この repo で見つかる言語は golang だけなので、走る pack は changed のときと同じ。そのあと、依頼の AC の 3 つの絞り込み、依頼の変異 3 つと追加の変異 3 つ、`ralph config get` の binary での確かめ、狭い coverage の run を足した
- Evidence: `docs/evidence/test-2026-11-21-codex-seat-network.log`(10,050 行。`.gitignore` の `docs/evidence/*.log` に当たるので commit されない)。run-test.sh の出力、1 分ごとの load、AC の 3 つの run の `-v` の出力、変異 6 つの diff と落ちたテストと失敗の文、`ralph config get` の出力、coverage の要約が入っている。run-verify.sh は自分の記録として `docs/evidence/verify-2026-11-21-163431.log` も書いた(同じく追跡外)

## Test execution

| Suite / Command | Tests | Passed | Failed | Skipped | Duration |
| --- | --- | --- | --- | --- | --- |
| `./scripts/run-test.sh` | 下の 2 行の合計 | 全部 | 0 | 2(条件つきの skip、下の行) | 2026-11-22 01:34:30 〜 02:20:24 JST、2,754 s、rc 0。`==> All verifiers passed.` |
| local verifier(`tests/test-*.sh` 46 本) | 46 本、`PASS` で始まる行 5,390 | 46 本(`OK` 46 行) | 0(`FAIL:` の要約行 29 本はどれも `FAIL: 0`) | `test-plan-visual.sh` の browser detection(`/Applications` にブラウザーがある)、`test-secret-scan-branch.sh` の「git 2.41 より古い本物の git」の場合(手元は git 2.49.0、stub の古い git の行は通る) | 約 24 分(01:34:30 〜 01:58 ごろ) |
| `go test ./...`(テストのある 11 パッケージ) | 11 | 11 | 0 | なし。`(cached)` は 0 | 約 22 分(01:58 ごろ 〜 02:20:24)、下の表 |
| `go test -count=1 -v ./internal/org/ -run 'Permission\|Spawn\|CodexWritable\|Envelope'` | 232 関数、`--- PASS` 626 行(subtest を含む) | 全部 | 0 | 0 | 27 s(パッケージ 25.571 s) |
| `go test -count=1 -v ./internal/config/` | 119 関数、`--- PASS` 783 行 | 全部 | 0 | 0 | 39 s(パッケージ 37.493 s) |
| `go test -count=1 -v ./internal/cli/ -run 'Doctor\|ConfigGet'` | 81 関数、`--- PASS` 239 行 | 全部 | 0 | 0 | 64 s(パッケージ 62.465 s) |

3 つの絞り込みには、関数の数で止まらないよう `-timeout 30m` を足した。そのほかは依頼のとおり。

| パッケージ | 結果 | 時間 |
| --- | --- | --- |
| `cmd/ralph` | ok | 1.341 s |
| `internal/cli` | ok | 1,252.346 s |
| `internal/config` | ok | 41.211 s |
| `internal/gitroot` | ok | 13.367 s |
| `internal/insights` | ok | 3.537 s |
| `internal/org` | ok | 1,201.770 s |
| `internal/org/driver` | ok | 11.117 s |
| `internal/org/protocol` | ok | 4.859 s |
| `internal/scaffold` | ok | 4.642 s |
| `internal/toolchain` | ok | 38.195 s |
| `internal/upgrade` | ok | 6.779 s |

golang pack の 1 パッケージの上限は 30 分で、`internal/cli` はその 70%、`internal/org` は 67% で終わった。run のあいだの 1 分 load は 10.4〜32.7(1 分ごとに 46 回測った)。私の絞った run(AC の 3 つ、変異 M1〜M5、coverage、binary のビルド)はどれも 01:34:56〜01:42:02 の shell の段に重ね、Go の段には何も回していない。M6 は run-test.sh が終わったあと(02:21)に回した。落ちたテストはなく、打ち直しもしていない。

### 計画のテストと AC の対応

| AC | テスト | 結果 |
| --- | --- | --- |
| AC1 | `TestPermissionArgsForDriver_CodexNetworkAccess`(driver 2 × mode 3 × 鍵 2 × codex_verified 2 の 24 subtest)、`TestPermissionArgsForDriver_Codex_VerifiedUnlocksMapping`、`TestOrgSpawn_Codex_NetworkAccessInStartArgs`(implementer の AgentStart の argv 全体、8 subtest)、leader と機能の org の reviewer は `TestOrgSpawn_Codex_WorkspaceWrite_AddDirWhenStateDirOutsideCwd` と `…FeatureOrgReviewerGetsAddDir` | PASS。既定で autonomous と edits の codex の座席に `-c sandbox_workspace_write.network_access=true` が入り、guarded の codex と claude の座席は変わらない |
| AC2 | 上の 2 つの表の `codex_network_access=false` の行 | PASS。`=false` が明示して渡る |
| AC3 | `TestDefault_OrgPermissions`、`TestLoad_OrgPermissionsCodexNetworkAccess`、`…CodexNetworkAccessWrongType`、`TestLoad_TemplateWithCodexNetworkAccessFalse`、`TestDefaultsLockStep/org`、`TestOrgTemplateDrift_*`(config)。`ralph config get` は下の binary の確かめ | PASS。`ralph config get` を固める Go のテストの行はない(Test gaps の 1) |
| AC4 | `TestDoctorCheckCodexSeatNetwork_Matrix`(driver_pool 2 × codex_verified 2 × mode 3 × 鍵 2 の 24 subtest、役はすべて claude)、`…_DefaultConfigAllClaudeRolesReports`、`…_DirectorOnlyWorkspaceWriteIsNotCounted`、`TestRunDoctorFull_CodexSeatNetworkLine`(既定の info、鍵 false の warn、`--strict` でも warn) | PASS |
| AC5、AC7 | 文書と `check-sync.sh`・`check-skill-sync.sh` | local verifier の中で通った(`scripts/verify.local.sh` の check-sync と check-skill-sync)。文書の中身は /test の範囲の外で、verify の報告が確かめている |
| AC6 | 既存の `--add-dir` の 2 つのテスト、`VerifiedUnlocksMapping`、`TestPermissionArgsForDriver_CodexReturnsANewSlice` | PASS。並びは「権限の引数 → `-c` → `--add-dir` → `--model` → prompt」 |

### `ralph config get` の確かめ(AC3)

HEAD から build した binary を、git の外の scratch のディレクトリで打った。

| ralph.toml | `ralph config get org.permissions.codex_network_access` |
| --- | --- |
| なし | `true`、exit 0 |
| `codex_network_access = true` | `true`、exit 0 |
| `codex_network_access = false` | `false`、exit 0 |
| `codex_network_access = "yes"` | 出力なし、exit 3(`cannot decode TOML string into struct field config.OrgPermissionsConfig.CodexNetworkAccess of type bool`) |
| `templates/base/ralph.toml` の写し | `true`、exit 0 |

### 変異

変異の写しと overlay の JSON は scratch(`$TMPDIR` の下の自分のディレクトリ)に置き、`go test -count=1 -v -timeout 30m -overlay <json>` で回した。置き換える文字列はファイルの中に 1 回だけあることを確かめてから替えた。どの変異も `build failed` は 0 件で、テストが落ちて rc 1 になった。M1〜M3 は依頼の 3 つ、M4〜M6 は私が足した。

| # | 変異 | 回したテスト | 落ちたテスト |
| --- | --- | --- | --- |
| M1 | edits の `-c` を外す: `permissions.go` の `return slices.Concat(codexEditsArgs, codexNetworkAccessArgs(cfg)), nil` を `return slices.Clone(codexEditsArgs), nil` に替えた(新しい slice を返すテストに当たらないよう `Clone` にした) | org の AC の絞り込み | 5 関数、11 subtest。`VerifiedUnlocksMapping/edits`、`CodexNetworkAccess` の edits の 2 行、`NetworkAccessInStartArgs` の edits の 2 行、`AddDirWhenStateDirOutsideCwd` の edits の 3 行、`FeatureOrgReviewerGetsAddDir` の edits の 3 行 |
| M2 | doctor の項目が `codex_verified` を見ない: `doctor_codex_network.go` の `if !orgCfg.Permissions.CodexVerified \|\| !workspaceWriteRole {` を `if !workspaceWriteRole {` に替えた | cli の AC の絞り込み | `TestDoctorCheckCodexSeatNetwork_Matrix` の 4 subtest(driver_pool に codex があり、codex_verified が false で、workspace-write の役がある行)。どれも `want (pass, "not needed: [org.permissions].codex_verified is false, …")` に対して info か warn を返した |
| M3 | `Default()` の `CodexNetworkAccess: true` を `false` に替えた | config の全体、cli の AC の絞り込み | config は 5 関数、12 subtest: `TestDefault_OrgPermissions`、`TestLoad_OrgPermissionsCodexNetworkAccess` の 3 行(ralph.toml なし、`[org.permissions]` なし、鍵なし)、`TestDefaultsLockStep/org`、`TestOrgTemplateDrift_DetectsChangedAndMissingKeys` の 8 行、`TestOrgTemplateDrift_EachKeyIsComparedOnBothSides`。cli は 2 関数: `TestDoctorCheckCodexSeatNetwork_DefaultConfigAllClaudeRolesReports`、`TestRunDoctorFull_CodexSeatNetworkLine/default_config_prints_the_info` |
| M4 | doctor の結果に項目を足さない: `doctor.go` の `results = append(results, checkCodexSeatNetwork(cfg.Org))` を `_ = checkCodexSeatNetwork` に替えた | cli の `-run 'Doctor'` | `TestRunDoctorFull_CodexSeatNetworkLine` の 2 subtest(info と warn) |
| M5 | 鍵を無視して常に `=true` を渡す: `codexNetworkAccessArgs` の条件を `cfg.Permissions.CodexNetworkAccess \|\| true` に替えた(`-vet=off`) | org の AC の絞り込み | 2 関数、4 subtest。`CodexNetworkAccess` と `NetworkAccessInStartArgs` の、codex_verified が true で鍵が false の autonomous と edits の行 |
| M6 | autonomous の `-c` を外す(M1 の対): `return slices.Concat(codexAutonomousArgs, …)` を `return slices.Clone(codexAutonomousArgs), nil` に替えた | org の AC の絞り込み | 5 関数、11 subtest。M1 と同じ 5 関数の autonomous の行 |

M3 で `TestLoad_TemplateWithCodexNetworkAccessFalse` が落ちないのは、ralph.toml に `false` を書いた読みのテストで、既定値に依らないため。意図どおりである。

## Coverage

- Statement: 狭い run の profile で測った(org は `-run 'Permission|Spawn_Codex|CodexWritable'`、cli は `-run 'CodexSeatNetwork'`、config はパッケージ全体)。`codexNetworkAccessArgs` 100%、`checkCodexSeatNetwork` 100%、`permissionArgsForDriver` 87.5%、`internal/config` 全体 96.7%
- Branch: Go の標準ツールでは測れない
- Function: 上の通り
- Notes: `permissionArgsForDriver` の実行されない 2 文は、claude と codex の `default:` の `unknown permission mode`(`permissions.go:173`、`:193`)で、この branch は触っていない。狭い run に当たるテストがないだけで、変更した行(`:181`、`:186`)はどちらも実行される。`doctor.go` の配線の 1 行は `TestRunDoctorFull_CodexSeatNetworkLine` が実行する

## Failure analysis

| Test | Error | Root cause | Proposed fix |
| --- | --- | --- | --- |
| なし | — | — | — |

## Regression checks

| Previously broken behavior | Status | Evidence |
| --- | --- | --- |
| issue #362: workspace-write の codex の座席が外へ通信できず、tflint の初期化などが落ちる | 引数の上では直った(座席の TUI の中では未確認、Test gaps の 2) | AC1 のテストが PASS。M1・M6 で autonomous と edits のどちらの `-c` を外しても落ちる |
| `codex_verified = false` で autonomous と edits の codex の座席を断る(fail-closed) | 保たれている | `TestPermissionArgsForDriver_Codex_FailClosed` と、`CodexNetworkAccess` の表の codex_verified=false の行が PASS。断るときは `-c` も返さない |
| guarded の codex の座席と claude の座席の引数 | 変わらない | `CodexNetworkAccess` の表の claude の 12 行と guarded の codex の 4 行、`NetworkAccessInStartArgs` の guarded の 4 行が PASS |
| 既存の `--add-dir` の並び(leader と機能の org の reviewer) | 保たれている | 2 つのテストが `-c` を足した argv で PASS |
| パッケージの引数の組を返した先で書き換えても次の座席に響かない | 保たれている | `TestPermissionArgsForDriver_CodexReturnsANewSlice` が PASS |
| doctor の既存の agmsg writable root の検査 | 変わらない | cli の `Doctor` の絞り込みと、run-test.sh の `internal/cli` 全体が PASS |

## Test gaps

1. `ralph config get org.permissions.codex_network_access` を固める Go のテストの行はない。`TestConfigGet_NoRalphTomlPrintsTheDefaults` の表にこの鍵がなく、上の binary の確かめだけが既定の `true` を見ている。`config get` は鍵の表を持たず、`Config` の struct を `reflect` でたどって読む(`internal/cli/config.go:148`)ので、`Default()` と読みのテスト(M3 で落ちる)が実質を守る。表に 1 行足せば AC3 の文面どおりに固まる(LOW)
2. 座席として herdr の pane に立てた codex の TUI の中で、`-c` の上書きが効いて tflint の初期化が通ることは確かめていない。`docs/evidence/codex-seat-network-2026-11-21.md` は `codex sandbox` と `codex exec` の 2 通りで確かめ、TUI は「確かめていないこと」に挙げている。/test は依頼どおり本物の herdr server と `~/.codex/config.toml` に触れていない
3. 設定ファイルに `network_access` の鍵がないマシンで、codex の既定が `false` であることは確かめていない(同じ evidence の「確かめていないこと」)。ralph は両方の値を明示して渡すので、ralph の動きはこの既定に依らない
4. spawn は `[org.permissions]` を呼んだ側の `ralph.toml` から読むので、機能の worktree の写しで鍵を変えられる。計画の S1 の範囲外の気づきで、tech-debt に記録済み。挙動のテストはない

## Verdict

- Pass: `./scripts/run-test.sh` rc 0(shell 46 本、Go 11 パッケージ、`(cached)` なし)、AC の 3 つの絞り込み(org 232 関数、config 119 関数、cli 81 関数)、依頼の変異 3 つ(M1 edits の `-c`、M2 doctor の `codex_verified`、M3 `Default()` の false)と追加の 3 つ(M4 doctor の配線、M5 鍵の無視、M6 autonomous の `-c`)がどれも落ちること、`ralph config get` の 5 通り
- Fail: 0
- Blocked: 0
- Verdict: pass。/sync-docs に進んでよい
