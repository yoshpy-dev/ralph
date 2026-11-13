# Test report: gate-tool-versions

- Date: 2026-11-13
- Plan: docs/plans/active/2026-11-12-gate-tool-versions.md(issue #311)
- Cycle: 2(cross-review の cycle 1 の直し 7c0bf678 と、origin/main の 3 回目の取り込み 34936129 のあと)
- Tester: orchestrator の inline 実行。opus のサブエージェントは今日 2 回止まった(cycle 1 の tester は週の利用上限、cross-review の直しの implementer は編集の前に応答が止まった)ので、subagent-policy の Fallback に従ってこのセッションで `/test` を回した
- Scope: `./scripts/run-test.sh`(変更のある言語の範囲。golang と local の verifier、Go は `go test ./...`)。HEAD `34936129`
- Evidence: `docs/evidence/test-2026-11-13-gate-tool-versions-c2.log`(git の追跡外)

## Cycle 2

### Test execution

| Suite / Command | Tests | Passed | Failed | Skipped | Duration |
| --- | --- | --- | --- | --- | --- |
| `GOFLAGS=-timeout=60m ./scripts/run-test.sh` | 下の 2 行の合計 | 全部 | 0 | 2(cycle 1 と同じ、前からある条件つきの skip) | 17:08:17〜17:48:00 JST(負荷 14〜66) |
| local verifier(`tests/*.sh` 45 本) | 45 本すべて | 全部 | 0 | browser detection、git 2.41 より古い本物の git | — |
| `go test ./...`(11 パッケージ) | 全パッケージ ok | 全部 | 0 | — | cli 1452 s、org 1559 s、toolchain 16.2 s、config 57.8 s |

上限を 60 分にしたのは cycle 1 と同じ理由(負荷の高い手元では internal/org と internal/cli が既定の 30 分に近づく)で、今回はどちらも 30 分を下回った。

### cycle 1 からの差

- 直し 7c0bf678 の新しいテスト関数 5 本はすべて通った: `TestBuildScript_SetsTheNineVariablesAgainBeforeTheFirstTool`、`TestProbe_StartupFileCannotTurnTheSafetyVariablesBackOn`(bash の `BASH_ENV` が `COREPACK_ENABLE_AUTO_PIN=1` と `PNPM_CONFIG_MANAGE_PACKAGE_MANAGER_VERSIONS=true` を export しても、偽の node は `0` と未設定を見る)、`TestProbe_AliasOrFunctionInStartupFileIsNotRun`(関数と alias の 2 つの subtest。どちらも unknown で、偽の node は打たれない)、`TestParseProbe_NoExecRecordIsUnknownAndNamesWhatItIs`、`TestDoctorNodeToolchain_NotAFileWarnsWithAHint`
- 対照の実行を持つ 2 本(`TestProbe_ChangesNothingWithShimsThatWouldPinAndInstall`、`TestProbe_PnpmDoesNotFetchThePackageManagerVersion`)は、対照の側でスクリプトの頭の 2 行を外す(`scriptWithoutEnvLines`)ようにした。対照が「変数が無ければ書く」ことを示す意味は変わらず、probe の側の確かめは弱めていない
- 取り込んだ main の変更(#312a の PR #327、#319 の PR #328。verbs.go の Send、send_paste.go、watch の usage_limited ほか)のテストも、この実行で一緒に通った
- 変異の確かめは implementer が `go test -overlay` で行った: スクリプトの頭の行を外すと 2 本、`case $rtc_p in /*` を外すと alias・関数のテストの 2 つの subtest が落ちる(計画の Progress checklist)

### Test gaps(cycle 2 で足したもの)

- 起動ファイルが変数を戻す場面のテストは bash の `BASH_ENV` だけで、zsh の起動ファイルの場面は自動のテストに無い(手元の `/etc/zshenv` が `ZDOTDIR` を上書きするので fixture が作れなかった)。zsh と bash 3.2 の login shell での確かめは verify の cycle 2 が手で行った(verify の report)
- PATH に相対の項目があると、そのファイルの node が「関数か builtin」として unknown になる。直す前は測れていた。テストは無い(verify の V10、tech-debt に残す)

## Verdict

cycle 2 の判定(cycle 1 の判定は下の「Cycle 1 の経過」にある)。


- Pass: yes(`==> All verifiers passed.`、rc 0)
- Fail: なし
- Blocked: なし

## Cycle 1 の経過

HEAD `e0ebff49`。tester サブエージェントが週の利用上限で止まったので inline で回した。evidence は `docs/evidence/test-2026-11-13-gate-tool-versions.log`(2 回目の実行。git の追跡外)。1 回目の時間切れの log はコミットしていない。

### Test execution

| Suite / Command | Tests | Passed | Failed | Skipped | Duration |
| --- | --- | --- | --- | --- | --- |
| 1 回目 `./scripts/run-test.sh`(既定の `-timeout=30m`) | — | — | internal/cli と internal/org がパッケージごと時間切れ | — | 11:00〜11:46(負荷 6〜10) |
| 2 回目 `GOFLAGS=-timeout=60m ./scripts/run-test.sh` | 下の行の合計 | 全部 | 0 | 下の Skipped 参照 | 11:57〜12:40(負荷 8〜13) |
| local verifier(`tests/*.sh` 45 本) | 3074(件数を出す 14 本の合計)+ 件数を出さない 31 本 | 全部 | 0 | 2(下記) | — |
| `go test ./...`(11 パッケージ) | Go のテスト関数 3,000 余り(toolchain 39、org 1773、cli 1214 ほか) | 全パッケージ ok | 0 | — | cli 1675 s、org 1906 s、toolchain 22.7 s、config 47.6 s |

Skipped(どれも前からある条件つきの skip で、この branch の変更とは関係しない):

- `tests/test-plan-visual.sh` の browser detection(/Applications のブラウザが起動するのを避ける)
- `tests/test-secret-scan-branch.sh` の、git 2.41 より古い本物の git の場合(手元は git 2.49.0。古い git は stub で確かめている)

### 1 回目の時間切れの切り分け

1 回目は internal/cli(1800.790 s)と internal/org(1800.698 s)がパッケージ全体の上限の 30 分で止まった。止まった時点で走っていたテストは `TestOrgDirectorAttach_ReadsTheWakeFromTheMainWorktree`(1 分 20 秒)と `TestGate_MaxLoadStartsAnywayAfterTheCap`(16 秒)で、どちらも普通に進んでいて、動かなくなったテストは無かった。

今回の変更のせいで遅くなったのかを確かめるため、branch の元(`0936aaea`)を `git archive` で作業ツリーの外に展開し、同じテストを比べた。

| テスト | branch | 元(0936aaea) |
| --- | --- | --- |
| `go test ./internal/cli/ -run '^TestOrgGate'`(負荷のある中、並列) | 290 s | 200 s |
| `TestOrgGate_InterruptStopsTheScriptAndRecordsNothing` 単独(gate を 4 回打つ) | 3.76 s | 2.67 s |
| `TestOrgGate_OrgDisbandedWhileTheScriptsRanNamesTheLog` 単独 | 1.08 s | 1.32 s |
| `TestOrgGate_LegacyLedgerGuard` 単独 | 0.46 s | 0.72 s |

単独で回すと、今回の変更が足すのは gate 1 回あたり 0.3 秒ほど(本物の probe が `git ls-files` と `/bin/sh` を 1 回ずつ打つ分)だった。手元のツールの版の確かめはどれも 0.02〜0.13 秒で終わる(`node -v`、`python3 --version`、`go version`、`pnpm -v`、`npm -v`、`mise current`)。並列のときの差は、ほかのレーンのテストが同時に走って負荷が 6〜13 あったことによる、とみる(負荷との因果は未確認)。同じ時刻にほかのセッションは `-test.timeout=40m` で `go test ./...` を回していた。

`packs/languages/golang/verify.sh` は呼ぶ側の `GOFLAGS` の `-timeout` を優先するので、2 回目は 60 分で回した。2 回目は internal/org が 1906 s で、30 分を超えている。この負荷では既定の 30 分で通らないことになる。CI の負荷では超えない見込み(未確認)。

### Coverage

- Statement: 測っていない(`run-test.sh` は coverage を出さない)
- Branch: 同上
- Function: 同上
- Notes: 新しいテスト関数は 72(`internal/toolchain` 39、`internal/org/gate_tools_test.go` 11、`internal/cli/doctor_node_test.go` 22)。各 slice の implementer が変異テストを回した: S1 13 個中 12 個(残る 1 個は文字列の固定で補った)、S2 2 個、S3 10 個、S6 3 個、S7 4 個がそれぞれ落ちた(計画の Progress checklist)

### 計画の Test plan との対応

| Test plan の項目 | テスト |
| --- | --- |
| 測る場所の一覧(AC3 の構成、上限、`git ls-files` の失敗) | `TestLayoutFromFiles_FindsPlacesAndTheirPackageManager`、`TestLayoutFromFiles_CapsPlacesAndCountsTheRest`、`TestFindLayout_GitFailureReturnsRootLayoutAndAnError`、`TestFindLayout_LsFilesIsBoundedByLsFilesTimeout` |
| 版のトークン、shim の印 | `TestVersionToken_ReadsEachToolsOutputShape`、`TestShimOf_NamesMiseAndAsdfShimsOnly`、`TestCleanText_ReplacesControlCharactersAndCuts` |
| 環境(D7 の 9 つの変数が呼ぶ側の値を上書きする) | `TestProbeEnv_SetsTheNineVariablesLastOverCallerValues`、`TestDoctorNodeToolchain_ProbeEnvForcesTheNineVariables` |
| probe(全部見つかる、missing、mixed、時間切れ、stdout を握る子、ctx の終わり) | `TestProbe_MeasuresFakeToolsPlaceByPlace`、`TestProbe_TimeoutKeepsWhatWasMeasured`、`TestProbe_ShimChildHoldingTheOutputDoesNotStopTheRest`、`TestProbe_ContextDoneIsAnError`、`TestRun_TimeoutStopsTheWholeGroupAndKeepsTheOutput`、`TestRun_ChildHoldingStdoutAfterTheShellExitsDoesNotHang` |
| 記録の鍵の往復、前の形の読み取り | `TestGateRecordDetails_Tools`、`TestGateToolsSummary_OneValuePerToolInTheFixedOrder` |
| doctor(JS の場所なし、各ファイルの合う・合わない、上のディレクトリ、別名、読めない範囲、shell の失敗・時間切れ、`$SHELL` の選び方、monorepo) | `TestDoctorNodeToolchain_*` の 21 本と `TestNodeErrText_OneShortLine` |
| npm の範囲(AC7 の表) | `TestRangeAllows_NpmRangeForms`、`TestParseRange_RefusesFormsOutsideTheSubset`、`TestRangeAllows_RefusesAVersionThatIsNotFull` |
| 統合: 本物の git の worktree で `# tool` の行と `tool_*` の鍵 | `TestGate_RecordsTheToolVersionsWithTheRealProbe` |
| 統合: 偽の shim(AC10)、前後の作業ツリーの違い | `TestProbe_ChangesNothingWithShimsThatWouldPinAndInstall`、`TestProbe_PnpmDoesNotFetchThePackageManagerVersion`、`TestGate_ProbeThatWritesInTheWorktreeIsReported`、`TestRegate_ProbeThatChangesTheWorktreeSaysSoOnStderr` |
| 統合: CLI の `TOOLS:` の行、doctor の「Node toolchain」の行 | `TestOrgGate_PrintsTheToolsLine`、`internal/cli/org_regate_test.go` の `TOOLS:` の期待、`TestDoctorNodeToolchain_DoctorOutputHasTheLine` |
| Edge cases(壊れた JSON、`.nvmrc` のコメント、2 行出す shim、空白を含む名前) | `TestDoctorNodeToolchain_BrokenPackageJSONFallsBackToTheRoot`、`TestVersionFileSpec_TakesTheFirstLineWithoutComments`、`TestVersionToken_ReadsEachToolsOutputShape`、`TestBuildScript_QuotesPlacesAndListsEachToolOnce` |

### Failure analysis

| Test | Error | Root cause | Proposed fix |
| --- | --- | --- | --- |
| internal/cli、internal/org(1 回目) | `panic: test timed out after 30m0s` | パッケージ全体が負荷の高い中で 30 分を超えた。今回の変更が足すのは gate 1 回あたり 0.3 秒ほど(上の切り分け) | 上限を 60 分にして回し直し、通った。実装は直さない |

### Regression checks

| Previously broken behavior | Status | Evidence |
| --- | --- | --- |
| 既存の gate・regate・merge・doctor のテスト(Details と出力を全文で比べるもの) | pass | 2 回目の `ok github.com/yoshpy-dev/ralph/internal/org` と `internal/cli`。期待値は S2 で、TestMain で固定した probe の結果に合わせて直した |
| #309 の gate.lock、max_load、gate accept | pass | `TestGate_MaxLoad*`、`TestOrgGate_WaitsForGateLockOnTheCommandsStderr`、`TestOrgGateAccept_*` |
| skill と写しの同期 | pass | `tests/test-check-skill-sync.sh`、`tests/test-check-sync.sh`(FAIL: 0) |

### Test gaps

- internal/cli の gate・regate・merge のテストは本物の probe を通る。internal/org は TestMain で probe を固定しているが、cli のテストからは org の固定の口(非公開のパッケージ変数)を触れない。そのため cli のテストは開発者の PATH の node・pnpm・python3・go を、自動インストールと自動 pin を止めた環境で打つ。版の値は期待に入れていない(`TOOLS:` の行は固定の結果で確かめる)ので、テストの結果は手元の版に左右されない。かかる時間は gate 1 回あたり 0.3 秒ほど
- 負荷の高い手元では、internal/org が既定の 30 分を超える(1906 s)。前からの傾向で、今回の変更が足した分は小さい
- 本物の mise・asdf の shim を通る場面(scripts が shim を通して起こす子が別の版を見る、issue の場面)は自動のテストに無い。計画の R2 と Known gaps のとおり
- coverage の数値は取っていない

### Verdict(cycle 1)

- Pass: yes(2 回目。全 verifier pass、`==> All verifiers passed.`)
- Fail: なし(1 回目の時間切れは負荷によるもので、上限を上げた 2 回目で通った)
- Blocked: なし
