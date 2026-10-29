# Test report: local-config-org

- Date: 2026-10-29(JST 22:03〜23:27)。再実行は 2026-10-30(JST 00:50〜01:23、末尾の節)
- Plan: `docs/plans/active/2026-10-29-local-config-org.md`(AC1〜AC13、Test plan)
- Tester: tester サブエージェント(Claude Opus 5.5)。パイプラインの cycle 1
- Scope: HEAD 64a20f69(merge-base は origin/main 5eba4013)。再実行は HEAD 33b567e7 で、64a20f69 からのコードの変更はテスト 2 本の直しだけ(末尾の節)。振る舞いのテストだけを実行した。静的解析は /verify が済ませている。環境は macOS(Darwin 25.5.0、arm64)、go1.27.3、git 2.50.1。このマシンはほかのセッションと CPU を分け合っており、1 分の load average は 4.85〜82.24 の間で動いた
- Evidence: `docs/evidence/test-2026-10-29-local-config-org.log`(gitignore の対象で、コミットしない)。`run-test.sh` 自身のログは `docs/evidence/verify-2026-10-29-130548.log`(1 回目)と `docs/evidence/verify-2026-10-29-133021.log`(2 回目)、`docs/evidence/verify-2026-10-29-155410.log`(再実行。名前の日付は UTC)。evidence の log には再実行の節(2026-10-30)を足した
- Insight event: `docs/insights/events/2026-10-29-local-config-org.jsonl` に test の行を 1 行足した(`--verdict pass --cycle auto`、cycle 1)。再実行でももう 1 行足した
- 追跡しているファイルは、この報告と insight event のほかは変えていない。隔離の確かめは scratchpad に置いた HEAD の clone で行い、テストのコードの差し替えは `go test -overlay` で scratchpad の写しを当てた

## Test execution

| # | Suite / Command | Tests | Passed | Failed | Skipped | Duration |
| --- | --- | --- | --- | --- | --- | --- |
| 1 | `GOFLAGS=-count=1 ./scripts/run-test.sh`(1 回目) | shell 45 本 + Go 10 パッケージ | shell 45 本(`PASS` の行 4,796)、Go 9 パッケージ | Go 1 パッケージ(`internal/org` の 1 本。当時は既知の不安定なテストで、S6 で直った) | shell の 1 行(前からある git 2.41 の行) | rc 1、1,438 秒(22:05:48〜22:29:46) |
| 2 | `go test -count=1 -run '^TestStopDirectorProcess_AProcessThatIgnoresSIGTERMIsNotEndedInTime$' -v ./internal/org/` | 1 | 1 | 0 | 0 | rc 0、1.0 秒 |
| 3 | `GOFLAGS=-count=1 ./scripts/run-test.sh`(規則どおり 1 回だけ打ち直した) | shell 45 本 + Go 10 パッケージ | すべて | 0 | shell の 1 行(同上) | rc 0、1,229 秒(22:30:21〜22:50:50) |
| 4 | `go test -count=1 -v -cover ./internal/config/ ./internal/gitroot/ ./internal/org/driver/` | トップレベル 159(サブテスト 356) | 159(356) | 0 | 0 | rc 0、8.4 秒 / 4.6 秒 / 2.7 秒 |
| 5 | `go test -count=1 -v ./internal/org/ -run '<AC4・AC8・AC11 のテスト>'` | トップレベル 25(サブテスト 61) | 25(61) | 0 | 0 | rc 0、23.4 秒 |
| 6 | `go test -count=1 -v ./internal/cli/ -run '<AC1〜AC3・AC6〜AC9 のテスト>'` | トップレベル 30(サブテスト 38) | 30(38) | 0 | 0 | rc 0、17.5 秒 |
| 7 | 隔離の確かめ: clone で `go test -count=1` を `config`・`gitroot`・`org/driver`・`org/protocol` | 4 パッケージ | 4 | 0 | - | rc 0、16 秒 |
| 8 | 隔離の確かめ: clone で `go test -count=1 ./internal/org/` | 1 パッケージ | 1 | 0 | - | rc 0、1,034 秒 |
| 9 | 隔離の確かめ: clone で `go test -count=1 ./internal/cli/` | 1 パッケージ | 1 | 0 | - | rc 0、566 秒 |
| 10 | 感度の確かめ 3 つ(下の「AC10: 隔離の確かめ」) | 3 実行 | - | 想定どおり、config で 2 本、cli で 11 本が落ち、agmsg のカナリアに 35 回届いた | - | 25 秒 / 131 秒 / 30 秒 |

- 1 と 3 の Go のパッケージごとの時間(どれも `-count=1` で、`(cached)` はない)

  | パッケージ | `func Test` の数 | 1 回目 | 2 回目 |
  | --- | --- | --- | --- |
  | `cmd/ralph` | 2 | 0.744 秒 | 0.399 秒 |
  | `internal/cli` | 863 | 562.149 秒 | 480.488 秒 |
  | `internal/config` | 80 | 25.706 秒 | 6.569 秒 |
  | `internal/gitroot`(新規) | 6 | 17.166 秒 | 3.868 秒 |
  | `internal/insights` | 41 | 3.610 秒 | 2.550 秒 |
  | `internal/org` | 1,241 | FAIL 693.921 秒 | 580.318 秒 |
  | `internal/org/driver` | 73 | 7.159 秒 | 1.614 秒 |
  | `internal/org/protocol` | 15 | 1.425 秒 | 3.285 秒 |
  | `internal/scaffold` | 32 | 4.030 秒 | 1.448 秒 |
  | `internal/upgrade` | 114 | 4.557 秒 | 3.547 秒 |

  `func Test` の数は `TestMain` を除いた数で、合計 2,467(origin/main は 2,442)。`run-test.sh` の go test は `-v` なしなので、実行した数でなく定義の数を書いた
- `run-test.sh` の範囲は `Language scope: full fallback (unclassified:.gitignore)` で、changed の既定でも shell の全 45 本と Go の全パッケージが走った。lead が名指しした `tests/test-ralph-config.sh` は 19/19、`tests/test-xreview-helpers.sh` は PASS 29・FAIL 0 で、どちらも `run-test.sh` の中で走ったので単独では打っていない
- `run-test.sh` の go test は `GOFLAGS` を受け継ぐ(`packs/languages/golang/verify.sh` の `run_tests` が `go env GOFLAGS` の前に `-timeout=30m` を置く)ので、`GOFLAGS=-count=1` で cache を使わせなかった
- 2 回の `run-test.sh` のあと、worktree の `git status --porcelain` は空だった。親が 1 のプロセスで `sleep`・ralph・herdr・agmsg・codex・claude・shell のものは残っていなかった

### load average(`uptime` の 1 分・5 分・15 分)

| 時点 | load average |
| --- | --- |
| 打つ前(22:03) | 12.33 27.12 40.64 |
| 1 回目の開始(22:05) | 7.36 20.31 36.08 |
| 1 回目の Go の途中(22:28) | 20.26 45.82 50.10 |
| 1 回目の終わり、失敗が分かったとき(22:29) | 16.48 39.03 47.15 |
| 失敗したテストの単独の実行(22:30) | 14.24 37.06 46.26 |
| 2 回目の開始(22:30) | 15.68 36.62 46.00 |
| 2 回目の終わり(22:50) | 4.85 9.57 18.72 |
| 8 の途中(23:04) | 82.24 64.66 46.94 |
| 8 の終わり(23:11) | 15.07 74.27 68.69 |
| 9 の終わり(23:21) | 24.14 24.08 41.33 |

## AC ごとに実行で確かめたもの

テスト名はすべて 4〜6 の `-v` の実行で PASS を見たもの。3 と 8・9 でも同じテストが通っている。

| AC | 実行で確かめたもの | 結果 |
| --- | --- | --- |
| AC1 | `TestLoad_LocalFileOverridesKeyByKey`、`TestLoad_LocalFileReplacesArrays`、`TestLoad_LocalFileWithoutRalphToml`、`TestLoad_LocalFileThroughASymlink`、`TestLoad_LocalPoolKeys`(3 通り)。`ralph config get` が重ねた値を返すことは cli の `TestConfigGet_ReadsTheMainWorktreesLocalFileFromAnyWorktree` | PASS |
| AC2 | `TestLocalPath_InGitIsTheMainWorktreeRoot`、`_BareRepositoryLinkedWorktreeIsItsToplevel`、`_OutsideGitIsTheConfigFilesDirectory`、`_RelativeConfigPath`、`TestLoad_LinkedWorktreeReadsTheMainWorktreesLocalFile`、`TestConfigGet_ReadsTheMainWorktreesLocalFileFromAnyWorktree`(主 worktree の root、linked worktree の root、サブディレクトリから打つ) | PASS |
| AC3 | `TestLoad_LocalFileErrorsNameItsPath`(ディレクトリ、読めない、TOML が壊れている、検査に落ちる、の 4 つを `ralph.toml` のあり・なしで 8 通り)、`TestLoad_RalphTomlValidationErrorDoesNotNameAnAbsentLocalFile`、`TestConfigGet_LocalFileThatCannotBeReadOrIsInvalidExits3`(4 通り、相対の `agmsg_home` を含む) | PASS |
| AC4 | Load 側: `TestLoad_LocalConfigEnv`(`off` はファイルも git も読まない、空・`OFF`・`on`・未設定は読む)。gate 側: `TestGate_ScriptEnvironmentIsFixed` は `Gate` を本物のスクリプトの実行で走らせ、スクリプトが書き出した環境を読む。呼んだ側の `RALPH_LOCAL_CONFIG=on` は static と test の両方のスクリプトにちょうど 1 つの `RALPH_LOCAL_CONFIG=off` として届いた。`TestGateScriptEnv` は受け継ぐ 4 つの形(`=on`、`=`、`=off`、`=` なし)を見る | PASS(実際の `ralph org gate` は打っていない) |
| AC5 | 隔離の確かめの準備で、clone の `git check-ignore -v ralph.local.toml` が `.gitignore:25` に当たった。2 つの管理ブロックが同じことは静的な検査なので /verify の担当 | 確かめた範囲で満たす |
| AC6 | `TestResolveAgmsgHome_Precedence`(7 通り。env を設定しても cfgValue と既定が使われる行を含む)、`TestNewOrgRuntimeAt_AgmsgHomeIsTheAgmsgDriversHome`、`TestOrgWatchDetach_…`の「`RALPH_ORG_AGMSG_HOME` が絶対パスでも相対の `--config` は断る」行、`TestOrgDirectorStart_TheAgmsgHomeVariableGetsNoWarning`。「production でこの名前を読むのは doctor だけ」は grep の検査なので /verify の担当 | PASS |
| AC7 | Load 側: `TestLoad_LocalAgmsgHome`(`agmsg`、`./agmsg`、`../agmsg`、`~`、`~user/agmsg` を断り、絶対パス、`~/agmsg`、空を通し、`ralph.toml` の相対は今までどおり読む)。watch 側: `TestOrgWatchDetach_ARelativeAgmsgHomeInConfigIsRefusedBeforeTheLaunch`(10 通り)、`TestOrgStart_ARelativeAgmsgHomeInConfigIsRefusedAndNothingLaunched`(`<task>` と `--plan`)、`TestOrgStart_AnAbsoluteOrHomeAgmsgHomeInConfigLaunchesTheWatch`(2 通り)。director start の側の `TestOrgDirectorStart_ARelativeAgmsgHome…` 4 本も通った | PASS(detach した watch の実際の起動はしていない。テストでは `orgWatchLaunch` が差し替えられている) |
| AC8 | `TestOrgDirectorStart_LaunchCmdPassesTheDirectorNoAgmsgHomeVariable`、`TestOrgDirectorStart_TheAgmsgHomeVariableGetsNoWarning`、`TestDirectorLaunchScript_*` 4 本、`TestStartDirector_LaunchCmd*` 4 本 | PASS |
| AC9 | `TestCheckLegacyAgmsgHomeEnv`、`TestRunDoctorOpts_LegacyAgmsgHomeEnv`(設定あり・未設定・空) | PASS |
| AC10 | 下の「AC10: 隔離の確かめ」 | 満たす |
| AC11 | `statedir_test.go` の 12 本(`TestResolveOrgStateDir_*` 9 本、`TestMainWorktreeRoot`、`TestLedgerMainWorktreeRoot`、`TestLegacyWorktreeStateDir`)、`internal/gitroot` の 6 本(サブテスト 27、移した `TestParseMainWorktreeRecord` を含む)。`internal/org` 全体も 3 と 8 で通った | PASS |
| AC12 | 文書の記述なので /verify の担当。実行するテストはない | 対象外 |
| AC13 | Go のテストと `run-test.sh` は 3 で rc 0。check-skill-sync、check-sync、`run-verify.sh` の static の部分は /verify の `run-static-verify.sh` が通している。`run-verify.sh` の既定(all)をそのまま 1 回打つことはしていない | test の部分は満たす |

## AC10: 隔離の確かめ

lead の指示は、テストが本物の agmsg と開発機の `ralph.local.toml` に触れないことを実行で確かめることだった。main のチェックアウトに `ralph.local.toml` を置くことはできないので、HEAD 64a20f69 を scratchpad に clone し、次の 3 つのカナリアを置いて 7〜9 を走らせた。

- clone の root(D2 で主 worktree の root になる場所)に、TOML として読めない `ralph.local.toml`(`[org` と 1 行)。読めば `config.Load` がそのパスを名指すエラーを返す。clone では `.gitignore:25` が無視する
- `HOME` を scratch のディレクトリにし、既定の agmsg の場所 `$HOME/.agents/skills/agmsg/scripts/` に、本物と同じ名前の 36 本のスクリプトを置いた。どれも呼ばれた引数を 1 つのログに書いて exit 0 で終わる
- `RALPH_ORG_AGMSG_HOME` を、同じ形でログを書く 2 つ目の agmsg に向けた。`RALPH_LOCAL_CONFIG` は外した。`GOCACHE`・`GOMODCACHE`・`GOPATH` は元の値を渡し、git の global の設定は scratch の `HOME` に写した

結果:

- 7〜9 の 6 パッケージはすべて通り、カナリアの呼び出しは 0 件だった
- cli と org の TestMain が固定する存在しない agmsg の場所(`$TMPDIR/ralph-cli-test-no-such-home`、`$TMPDIR/ralph-org-test-no-such-home`)は、すべての実行のあとも作られていなかった

感度の確かめ(カナリアが漏れを捕まえられること):

| 差し替え(`-overlay`) | 実行 | 結果 |
| --- | --- | --- |
| config の TestMain の `RALPH_LOCAL_CONFIG` を `off` から空にする | clone で `./internal/config/` 全体 | rc 1。`TestLoad_TemplateRalphToml` と `TestLoad_TemplateWithBothKeysFalse` が `Load(templates/base/ralph.toml): <clone>/ralph.local.toml: toml: expected character ]` で落ちた |
| cli の TestMain の `RALPH_LOCAL_CONFIG` を `off` から空にする | clone で `./internal/cli/ -run 'TestConfigGet\|TestStatusCmd\|TestOrgMergeOrder'` | rc 1。11 本が落ちた。カナリアのパスを名指す行が 18 行あり、`ralph status` が `load ralph.toml: <clone>/ralph.local.toml: toml: expected character ]` を出している。git の呼び出しを数える 2 本(`TestStatusCmd_MergeGuaranteeRunsNoGH`、`TestOrgMergeOrder_NoLedger…`)も落ちた |
| cli の TestMain から `driver.AgmsgHomeOverride` の固定を外し、`setAgmsgHomeOverride` を何もしない関数にする | clone で `./internal/cli/ -run 'TestOrgSend'` | rc 0 のまま、既定の場所のカナリアに 35 回届いた(`join.sh` 24 回、`send.sh` 11 回)。`RALPH_ORG_AGMSG_HOME` の側は 0 回 |

- 3 つ目は、2 つの固定がなければ org の verb のテストが既定の agmsg(開発機では本物)を呼ぶこと、そしてカナリアがそれを捕まえることを示す。ただし、そのとき `TestOrgSend*` 自体は落ちない(Test gaps の 2 つ目)
- `RALPH_ORG_AGMSG_HOME` の側の 0 回は、参考にとどまる。cli の TestMain はこの変数を外し、org のテストでは差し替え口が先に返るので、この変数が読まれるかどうかをこの実行は区別できない。AC6 の根拠は上の表のテストと /verify の probe

## Coverage

- Statement: `internal/config` 96.6%、`internal/gitroot` 92.9%、`internal/org/driver` 95.3%(4 の `-cover`)
- Branch: Go の標準のツールにはないので測っていない
- Function: 新しいコードの関数ごと(`go tool cover -func`): `LocalPath` 100%、`readLocalConfig` 100%、`checkLocalAgmsgHome` 85.7%、`Load` 90.3%、`validateLoaded` 98.3%、`orgPoolKeysPresent` 75.0%。gitroot は `MainWorktree` 85.7%、ほかの 5 つ(`parseMainWorktreeRecord`、`Toplevel`、`SamePath`、`ResolvedOrClean`、`command`)は 100%
- Notes: 通らないブロックは次のとおり。
  - `config.go:701-702`: `ralph.toml` が読めるのに `ErrNotExist` 以外のエラーになる場合
  - `config.go:717-718`・`:731-732` と `local.go:98-99`: 一度 `Unmarshal` に通った同じ文書の 2 回目の parse のエラー。実際には起きない防御の分岐
  - `config.go:668-669`: `orgPoolKeysPresent` の parse のエラー。同じ理由
  - `gitroot.go:42-43`・`:54-55`・`:61-62`: `rev-parse` の出力が 2 行でない、`worktree list` の失敗、主 worktree の root が stat できない。gitroot の自前のテストはどれも通らない。org の statedir のテストを `-coverpkg=./internal/gitroot/` で測ると `:61-62` は通り、`:42-43` と `:54-55` はどちらのテストからも通らない

## Failure analysis

| Test | Error | Root cause | Proposed fix |
| --- | --- | --- | --- |
| `TestStopDirectorProcess_AProcessThatIgnoresSIGTERMIsNotEndedInTime`(`internal/org`、1 の 1 回目だけ) | `director_cmd_test.go:310: stopDirectorProcess = {outcome:ended signalled:true note:process=ended after SIGTERM err:<nil>}, want not ended within 300ms` | 64a20f69 の時点では既知の不安定なテストで、tech-debt に行があった(いまは `docs/tech-debt/README.md:287` の取り消し線の行と、その上の RESOLVED の注)。当時のテストは `sh -c "trap '' TERM; exec sleep 30"` を起こして固定の 100 ms だけ待っていたので、trap を置く前に SIGTERM が届くと sh が終わった。失敗したときの load average は 16.48 39.03 47.15 だった。単独の実行(2)は 0.48 秒で通り、打ち直した 3 も通った。64a20f69 の差分は `stopDirectorProcess`(`internal/org/director_cmd.go:199`)もこのテストも変えていなかった(`director_cmd.go` の差分は 249 行以降)ので、この PR のコードの不具合ではない | この報告のあと、この PR の S6(321203dc、140ed1d2)が直した。固定の 100 ms をやめ、sh が trap の直後に作る marker(`trap '' TERM; : > <ready>; exec sleep 30`)を 10 s まで待ってから SIGTERM を送る(`director_cmd_test.go:307`)。tech-debt の行は RESOLVED になった。33b567e7 では `-count=20` で 20/20 通った(末尾の節) |

## Regression checks

| Previously broken behavior | Status | Evidence |
| --- | --- | --- |
| org の ledger の決まり方(`internal/gitroot` に移す前と同じ) | 通る | `statedir_test.go` の 12 本(5)、`internal/org` 全体(3、8) |
| Load が個人用のファイルのために打つ git が、git の呼び出しを数えるテストを壊さない(S2 で起きたずれ) | 通る | `TestStatusCmd_MergeGuaranteeRunsNoGH`、`TestOrgMergeOrder_NoLedgerPrintsTheNoBranchesNoteWithoutRunningGit`(6)。感度の確かめでは TestMain の `off` を外すと 2 本とも落ちた |
| `RALPH_ORG_AGMSG_HOME` で隔離していたテストが、env をやめたあとも本物の agmsg に向かない | 通る | 隔離の確かめでカナリアの呼び出し 0 件 |
| `scripts/ralph-config.sh` の既定の行を消しても、ほかの既定と検査は保たれる | 通る | `tests/test-ralph-config.sh` 19/19、`internal/config` の `defaults_sync_test.go`(3) |
| cross-review の helper | 通る | `tests/test-xreview-helpers.sh` PASS 29・FAIL 0(3) |
| 既存の全テスト | 通る | 3: shell 45 本と Go 10 パッケージ、rc 0 |

## Test gaps

- 実際の `ralph org gate` と `ralph org regate` は打っていない。gate の環境は `TestGate_ScriptEnvironmentIsFixed` が `Gate` の本物のスクリプトの実行で見ている。一方、regate(`internal/org/regate.go:659`)は同じ `gateScriptEnv` を通るだけで、regate のスクリプトに `RALPH_LOCAL_CONFIG=off` が届くことを見るテストはない
- agmsg の隔離は TestMain の固定と `setupOrgStubPATH` の差し替えだけが守っている。感度の確かめの 3 つ目では、2 つを外すと `TestOrgSend*` が既定の agmsg を 35 回呼び、それでも全部通った。テストが stub の agmsg のログに呼び出しが届いたことを見ていないためとみられる(未確認)。足すなら、`setupOrgStubPATH` を使うテストのうち 1 本で、返る `agmsgLog` に `send.sh` の呼び出しがあることを確かめる行が最小
- detach した watch と director の実際の起動、`ralph upgrade` が `.gitignore` の管理ブロックに行を足すこと、本物の herdr と agmsg での通しは打っていない(テストでは `orgWatchLaunch` が差し替えられている。後の 2 つは計画の範囲外)
- 「production のコードで `RALPH_ORG_AGMSG_HOME` を読むのは doctor だけ」と「`AgmsgHomeOverride` に代入するのはテストだけ」を固定するテストはない(self-review L4、verify の Coverage gaps と同じ)
- Coverage の Notes の通らない分岐。防御の分岐が大半で、テストを足す必要は薄い

## Verdict

ブランチの今の HEAD 33b567e7 の判定は pass。根拠は末尾の節の再実行で、1 回目の rc 0 で、打ち直しはしていない。64a20f69 では、1 回目が当時の既知の不安定なテスト 1 本で rc 1 になり、lead の規則どおり 1 回だけ打ち直した 2 回目が rc 0 だった。そのテストは S6 で直った。

- Pass: 3 つの実行(33b567e7)。R1 の `run-test.sh` で shell 45 本と Go 10 パッケージが rc 0、R2・R3 で直した 2 本を含む 2 つの系統を `-count=20`、すべて通った。64a20f69 では 3(`run-test.sh` の打ち直しが rc 0)、2・4〜6 の名指しの実行(SKIP 0)、7〜9 の隔離の確かめ(カナリアの呼び出し 0 件)、10 の感度の確かめ 3 つ(どれも想定どおり)
- Fail: 0(64a20f69 の 1 回目の `TestStopDirectorProcess_AProcessThatIgnoresSIGTERMIsNotEndedInTime` は、単独と打ち直しで通り、S6 で直った。33b567e7 で落ちたテストはない)
- Blocked: 0
- Verdict: pass

## Re-run after S6 and the V3 fix (33b567e7)

2026-10-30 に HEAD 33b567e7 で打ち直した。148d542e(この報告の最初の版)のあとのコードの変更は、テストの 2 か所だけである。

- S6 321203dc と 140ed1d2(`internal/org/director_cmd_test.go`): `TestStopDirectorProcess_AProcessThatIgnoresSIGTERMIsNotEndedInTime` は固定の 100 ms の待ちをやめ、sh が trap の直後に作る marker を 10 s まで待つ
- c7095378(`internal/cli/org_watch_test.go`、verify の V3): `TestOrgWatchDetach_StartDetachedProcess_NewSessionDevNullStdinAppendedLog` の子のスクリプトは、1 行目で `trap '' TERM` を張ってから log を書く
- tech-debt の 2 つの行(`docs/tech-debt/README.md:287` と `:367`)は取り消し線になり、RESOLVED の注が付いた。ほかのコミットは計画、self-review、verify の報告と insight の行だけ

### 打つ前の待ち

lead の指示どおり、1 分と 5 分の load average がともに 30 を切るまで、前景で `sleep 60` を繰り返して `sysctl -n vm.loadavg` を読んだ。00:50:34 に 36.85 45.34 43.64 から始め、00:53:54 に 12.73 28.01 36.49 になった。待った時間は 3 分 20 秒。

### 実行

| # | Suite / Command | Tests | Passed | Failed | Skipped | Duration |
| --- | --- | --- | --- | --- | --- | --- |
| R1 | `GOFLAGS=-count=1 ./scripts/run-test.sh` | shell 45 本 + Go 10 パッケージ | shell 45 本(`PASS` の行 4,796)、Go 10 パッケージ | 0 | shell の 1 行(前からある git 2.41 の行) | rc 0、1,602 秒(00:54:10〜01:20:52) |
| R2 | `go test ./internal/org/ -run '^TestStopDirectorProcess_' -count=20 -timeout=10m -v` | トップレベル 4 × 20 | 80 | 0 | 0 | rc 0、58.0 秒 |
| R3 | `go test ./internal/cli/ -run '^TestOrgWatchDetach_StartDetachedProcess_' -count=20 -timeout=10m -v` | トップレベル 2 × 20 | 40 | 0 | 0 | rc 0、13.3 秒 |

- R1 の Go(どれも `-count=1` で、`(cached)` はない): `cmd/ralph` 0.759 秒、`internal/cli` 774.684 秒、`internal/config` 13.380 秒、`internal/gitroot` 4.778 秒、`internal/insights` 1.973 秒、`internal/org` 937.513 秒、`internal/org/driver` 5.852 秒、`internal/org/protocol` 3.017 秒、`internal/scaffold` 4.923 秒、`internal/upgrade` 5.374 秒。`tests/test-ralph-config.sh` は 19/19、`tests/test-xreview-helpers.sh` は PASS 29・FAIL 0
- R1 は 1 回目で rc 0 だったので、打ち直していない。30 分の timeout に届いたパッケージはない。implementer が S6 の検証で負荷 126〜428 のときに落ちたと報告した 2 本(`TestStartDirector_ACleanupThatCannotConfirmTheEndWritesNothingOnceItIsNotCurrent/an_attach_--replace` と `TestWakeDirector_AFailedWakeIsNotTriedAgainForFiveMinutes/no_return_in_time`)は、R1 では落ちなかった(`internal/org` が ok)。単独の実行はしていない
- R2 の 4 本は `TestStopDirectorProcess_` の系統のすべてで、直した `AProcessThatIgnoresSIGTERMIsNotEndedInTime` は 20/20 通った。1 回の時間は 0.43〜2.01 秒、中央値 0.74 秒で、marker の待ちの上限 10 s に届いた回はない
- R3 の 2 本は `TestOrgWatchDetach_StartDetachedProcess_` の系統のすべてで、直した `NewSessionDevNullStdinAppendedLog` は 20/20 通った
- R1 は 600 秒を超えるので、`run_in_background` を使わず、perl の `fork` と `setsid` で切り離して起こし、前景の `caffeinate -w <pid>`(1 回 570 秒まで)で終わりを待った。切り離す前に INT・HUP・QUIT・TERM を既定の扱いに戻した。shell の SKIP の行は前と同じ 1 行で、signal を使う検査が無視された signal のために SKIP になることはなかった
- R1 のあと、worktree の `git status --porcelain` は空だった。親が 1 のプロセスで `sleep`・ralph・herdr・agmsg・`go test` のものは残っていなかった

### load average(`sysctl -n vm.loadavg` と `uptime`)

| 時点 | load average |
| --- | --- |
| 待ちの開始(00:50:34) | 36.85 45.34 43.64 |
| 待ちの終わり(00:53:54) | 12.73 28.01 36.49 |
| R1 の開始(00:54:10) | 11.64 27.04 35.99 |
| R1 の途中(01:03:57) | 4.98 8.57 20.84 |
| R1 の途中(01:13:36) | 11.43 9.90 15.28 |
| R1 の終わり(01:20:52) | 6.99 8.17 12.16 |
| R2 の開始と終わり(01:21:29、01:22:28) | 5.83 7.74 11.84、14.67 9.52 12.20 |
| R3 の開始と終わり(01:22:45、01:23:00) | 12.44 9.28 12.07、11.20 9.16 11.98 |

load は低かったが、`internal/cli` と `internal/org` は 64a20f69 の 2 回目(480 秒と 580 秒)より長くかかった。間の load を 10 分ごとにしか読んでいないので、理由は確かめていない。

### 打ち直さなかったもの

AC ごとの名指しの実行(4〜6)、隔離の確かめ(7〜9)、感度の確かめ(10)は打ち直していない。64a20f69 からのコードの変更は上の 2 つのテストの直しだけで、production のコードと、それらの実行が見ていたテストは変わっていない。R1 では、それらのテストを含む全パッケージが `-count=1` で通った。Test gaps の 5 つはどれもこの変更で変わらず、残っている。
