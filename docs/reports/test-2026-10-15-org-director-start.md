# Test report: org-director-start

- Date: 2026-10-15(JST 18:23〜19:25)
- Plan: docs/plans/active/2026-10-15-org-director-start.md(`Status: Approved`。`./scripts/plan-visual.sh digest` は 671c60416856 で、`- Approved:` の digest と同じ)
- Tester: tester subagent (Claude Opus 5.5)。パイプラインの 1 回目
- Scope: HEAD b6c86188(base 9cbf7080、17 コミット、43 ファイル、+8,189/-383)。/verify が見た dbd67a58 から増えたのは、verify の報告(ec70c739)、古い印のテスト(ca98c248)、計画の label の長さの直し(b6c86188)の 3 つ。behavioral test だけを実行した(静的解析は /verify の担当)。環境は macOS(arm64)、go1.26.0 と go1.25.8(`GOTOOLCHAIN`)、git 2.49.0。実行中の load average は 8〜12(別のセッション)
- Evidence: `docs/evidence/test-2026-10-15-org-director-start.md`(1. run-test.sh の抜粋、2. Go の結果と数・`-race`・canary・coverage、3. stub の herdr に対する HEAD の binary、4. 隔離した本物の herdr server に対する HEAD の binary、5. mutation と probe のテスト、6. 今の main と合わせた tree)。生の出力は `docs/evidence/test-2026-10-15-org-director-start.log`、wrapper の全ログは `docs/evidence/verify-2026-10-15-092359.log`(どちらも gitignore の対象なのでコミットしない)
- テストもコードも変えていない。mutant と probe のテストは scratchpad に置き、`go test -overlay` で読ませた。worktree に書いたのは、この報告と上の evidence だけ

## Test execution

| Suite / Command | Tests | Passed | Failed | Skipped | Duration |
| --- | --- | --- | --- | --- | --- |
| `./scripts/run-test.sh`(changed。`templates/base/ralph.toml` の変更で full に広がった。go1.26.0) | shell 41 本、Go 8 パッケージ | shell 41 本すべて OK、Go 8 パッケージすべて ok | 0 | shell 1(下) | 819 秒、rc 0 |
| `GOTOOLCHAIN=go1.25.8 go test -count=1 -v -coverprofile ./internal/...` | cli: top-level 699、subtest 639。org: 800、1,682。org/driver: 48、89。org/protocol: 15、56。config: 37、34。insights: 36、7。scaffold: 32、0。upgrade: 114、13 | 同じ数から skip を引いた数 | 0 | org 1、scaffold 2 | 255 秒(cli 250.0 s、org 114.9 s)、rc 0 |
| `go test -count=1 -v ./internal/...`(go1.26.0) | go1.25.8 と同じ | go1.25.8 と同じ | 0 | org 1、scaffold 2 | 220 秒(cli 219.2 s、org 83.3 s)、rc 0 |
| うちこの差分で足したテスト 67 本 | top-level 67、その下の subtest 187 | 両方の版ですべて | 0 | 0 | - |
| `go test -race -count=1 ./internal/org/ ./internal/cli/`(パッケージ全体、go1.26.0) | 2 パッケージ | 2 | 0 | - | 445 秒(org 121.8 s、cli 308.0 s)、DATA RACE なし |
| mutation(`go test -overlay`、go1.26.0、`-run` で絞った回) | mutant 86 件と対照 2 件 | red 78 件 | 生き残り 8 件 | - | 1 件 2〜56 秒 |
| 生き残り 8 件をパッケージ全体で(org と cli の両方) | 15 回 | A12、A13、A14 が org の全体で red | D4、E1、E12、E13、P2 は生き残り | - | 1 回 90〜262 秒 |
| probe のテスト(overlay だけ、コミットしない) | org 3 本(subtest 2)、cli 1 本(subtest 2) | HEAD ですべて通る | 狙った mutant のもとでそれぞれ落ちる。対照の D1 と P3 では通る | - | 1 回 2 秒前後 |
| 今の main(f21df66c)と合わせた tree で `go test -count=1 ./internal/org/... ./internal/cli/` | 4 パッケージ | 4 | 0 | - | 222 秒、rc 0 |
| HEAD の binary と stub の herdr(下の「実機の確認」の 1〜5) | 5 段 | すべて期待どおり | 0 | - | 測っていない |
| HEAD の binary と隔離した本物の herdr(下の 6〜8) | 3 段 | すべて期待どおり | 0 | - | 測っていない |

- shell は 41 本すべてが `OK` で終わり、`FAIL:` の行 25 本はどれも 0 だった。PASS で始まる行は 4,143 本。Skipped は前からある `tests/test-secret-scan-branch.sh` の 1 件(git が 2.49.0 なので、2.41 より古い git の場合を飛ばす)。この差分は shell のテストを変えていない
- `run-test.sh` の中の Go は cli(221.5 s)、org(87.5 s)、org/driver を流し直し、config、insights、org/protocol、scaffold、upgrade は `(cached)` だった。この 5 つは上の 2 回の `-count=1` の実行で ok を確かめた
- go1.25.8 と go1.26.0 で、テストの名前と結果の行(時間を除いて並べた 4,301 行)は同じだった
- Skipped は `TestEscalate_NoDesktopNotificationOffDarwinIsSkipped`(org。darwin で飛ばす作り)と scaffold の `TestBaseFS_WithMockFS`、`TestAvailablePacks_WithMockFS`。どれもこの差分と関係ない
- 差分で足した `func Test` は 67 本で、消したものはない
- 前に揺れたテスト(`TestRunDoctorOpts_ProbeModelsFalse_NoSubprocess`、`TestRunDoctorFull_StrictFlipsExitCode_DriftedCore`、`TestRunWatcher_TimeoutIndependentOfSmallInterval`、`TestOrgGate_InterruptStopsTheScriptAndRecordsNothing`)は、どの実行でも落ちなかった。`TestOrgGate_InterruptStopsTheScriptAndRecordsNothing` は go1.25.8 で 4.33 s、go1.26.0 で 2.48 s だった
- AC13: Go の 2 回の実行、`-race`、mutant と対照の 103 回(絞った回 88、全体の回 15)、probe のテスト、main と合わせた tree の回は、PATH の先頭に canary の `herdr` と `osascript` を置いて走らせた(呼ばれたら argv を記録して exit 97)。呼ばれた回数は 0 回。手で同じ PATH から打つと 2 つとも記録されたので、PATH の検索は canary に届く。テストのあと、main のチェックアウトにも worktree にも `.harness/state/org` はなく、main のチェックアウトの下に `director.jsonl` は 1 つもない

## 実機の確認(HEAD の binary)

HEAD を go1.25.8 で build した binary を使った。どの回も `--state-dir` と `--config` は scratch のもので、`--config` は `[org.permissions].default = "autonomous"` と `[org.permissions.roles].director = "guarded"` を持つ。repo の `.harness/state/org/` には書いていない。evidence の 3 と 4 に script と出力がある。

stub の herdr(evidence の 3)。PATH には ralph、CLI のテストと同じ状態つきの stub の `herdr`(agent start で眠らせるつまみを 1 つ足したもの)、記録するだけの `osascript` を置き、`env -i` で打った(USER もない)。

1. K0: `ralph org watch --all --detach` で見張りを立てた。`ralph status` は `org watch: running` と `director: not attached` を出した
2. K1(`starting` の表示と agent start の時間切れ): agent start を 6 秒眠らせ、`--timeout-ms 2500` で start を裏で打った。`pane_created d1` が書かれた時点で、`ralph status` は `director: attached d1 (driver=claude, headless starting, model=haiku, pane pane-stub-1, workspace ws-stub-1, ...)` を出した。`director status --json` の `state` は `starting`、`permission_mode` は `guarded`、`user` は空文字だった。USER がないので、status の行に `by` は付かない。start は約 3 秒で exit 1 になり、`herdr: timed out: context deadline exceeded; start closed what it had made in herdr, and d1 is recorded stopped` と言った。herdr には label の確認 4 回のあと C-c、pane close、workspace close が打たれ、`director.jsonl` は attached、workspace_created、pane_created、detached、stopped の順になった。verify が「Partially verified」とした AC10 の `starting` を binary で通した
3. K2(落ちた start): agent start の最中に start を SIGKILL すると、d2 は ws と pane が記録された `starting` のまま残った。codex の start も attach も exit 1 で断られ、どちらも `stop it with ralph org director stop --id d2 --state-dir '<run>/ledger'` を出した。2 つの拒否で herdr は 0 回だった。`stop --id d2` は C-c と 2 つの close を打ち、detached と stopped を書いた
4. K3(後始末で閉じられない): `ORG_STUB_FAIL=agent:start` と pane close の失敗を重ねると、start は exit 1 で、閉じるための herdr の打ち方と `--state-dir` つきの stop を出し、detached だけを書いた。`ralph status` は「d3 is still open but not attached: stop it with ...」の案内を出した。close が失敗したままの stop は何も書かずに断り、`--force --state-dir` の打ち方を出した。close が通るようになったあとの stop は 2 つとも閉じて stopped を書いた
5. K4(自分の pane と workspace の中から): `HERDR_PANE_ID=pane-stub-4` の stop は herdr を打たずに断った。`HERDR_WORKSPACE_ID=ws-stub-4` の stop は `--force` があってもなくても、label の確認 4 回のあと断った。`HERDR_WORKSPACE_ID=ws-stub-9`(別の workspace)の stop は普通に閉じた。後片付けでは `watch --stop` のあと、run の dir を名に持つ process は 0、`osascript` の記録は 0 行だった

隔離した本物の herdr(evidence の 4)。herdr 0.7.5 の server を `env -i` で立て、`XDG_CONFIG_HOME`、`XDG_STATE_HOME`、`HERDR_SOCKET_PATH` を `/tmp/rdt9/` の下に置いた。`lsof` で見ると、server が開いていたのは `/tmp/rdt9` と scratch の下のファイルだけだった。座席の agent は claude 2.1.291(空の `CLAUDE_CONFIG_DIR`、model haiku、guarded)。

6. R1: start は `started d1 (... permission=guarded, pane=w1:p2, workspace=w1)` を出した。workspace w1 の label は `ralph_director_61ea4337317f`、tab の label は `director_61ea4337317f` だった。w1 には herdr が workspace を作るときに付ける tab「1」(pane w1:p1)も入っている
7. R2: w1:p1 に `ralph org director stop --id d1` を打ち込むと、`HERDR_WORKSPACE_ID is its workspace "w1", labelled ralph_director_61ea4337317f, ...` で断られた。`--force` でも同じだった。pane の環境変数は `HERDR_WORKSPACE_ID=w1`、`HERDR_PANE_ID=w1:p1` で、`director.jsonl` は md5 で変わっていない。herdr が pane に `HERDR_WORKSPACE_ID` を渡すこと、自分の workspace の中からの拒否が本物の herdr でも効くことを確かめた
8. R3: 別の workspace(w2、label `probe_other`)の pane から打つと、`stopped d1 (ctrl_c=ok, pane=closed, workspace=closed)` になった。あとの workspace の一覧は w2 だけ、agent の一覧は空で、`director.jsonl` は attached、workspace_created、pane_created、launched、detached、stopped の順だった

- 後片付け: 同じ `env -i` の環境で `herdr workspace close w2`、`ralph org watch --stop`、`herdr server stop` がどれも rc 0。そのあと herdr server の process はなく、`/tmp/rdt9` や scratch の `real` を名に持つ process もなく、`/tmp/rdt9` は消した。人の herdr の状態は読むだけにした。`~/.config/herdr/session.json` の sha256 の先頭(004173699fb3c932)と `herdr-server.log` の mtime(Oct 1 14:35:45)は前後で同じで、人の herdr server は前も後も動いていなかった

## Coverage

- Statement(go1.25.8、各パッケージ自身のテストの profile): `internal/org` 94.6%、`internal/cli` 87.7%、`internal/config` 93.3%、`internal/org/driver` 93.1%、`internal/org/protocol` 100.0%、`internal/insights` 86.1%、`internal/scaffold` 75.7%、`internal/upgrade` 91.2%(8a-1 の 2 回目は org 94.6%、cli 87.3%)
- 新しいファイル: `internal/org/director_start.go` 212/228 文(93.0%)、`internal/org/director_stop.go` 132/138 文(95.7%)、`internal/cli/org_director_headless.go` 94/98 文(95.9%)。変えたファイル: `director.go` 99.5%、`org_director.go` 98.3%、`spawn.go` 92.0%、`prompts.go` 97.4%、`verbs_all.go` 99.1%、`permissions.go` 90.3%、`org.go` 95.8%、`status.go` 96.2%
  - `director_start.go` で通らない 16 文は、token の `rand.Read` の失敗、ロックの中の `readRaw` と `nextID` の失敗(5 か所)、`AttachHeadless` が拒否以外の error を返したとき(470、mutant E1)、最初の `Store.Read` の失敗(532、mutant D4)、`absPath` の失敗 2 つ、`RenderDirectorPrompt` の失敗、`replacedError` と `abort` の error の分岐(`MarkStopped` や `Detach` が失敗したとき)
  - `director_stop.go` で通らない 6 文は、`readRaw` の失敗、`DirectorCleanupError.Error` の 2 つの分岐、C-c の送信の失敗(372)、`directorShellWord` の quote の分岐
  - `org_director_headless.go` で通らない 4 文は、設定と model の解決の失敗、`os.Getwd` の失敗、`startAttachedError` の `cur == nil`(233、mutant P2)
- Function: `director_stop.go` と `org_director_headless.go` の関数は `directorShellWord`(66.7%)のほか、どれも 80% 以上。`director_start.go` で低いのは `newDirectorToken` と `abort`(75.0%)、`replacedError`(83.3%)、`writePrompt`(85.7%)、`checkDirectorStart`(87.5%)
- Branch: Go の標準ツールには branch coverage がないので測っていない。代わりに mutation で分岐ごとの判定を確かめた(下)

## Test plan coverage map

計画の Test plan の項目ごとに、それを確かめるテストを挙げる。「mutant」の列は、その項目の判定を壊した mutant が落ちたかどうか(id は evidence の 5.2)。

| Test plan の項目 | テスト | mutant |
| --- | --- | --- |
| director store: 新しい field とイベント、`stopped` の書き方、閉じていない headless の登録の畳み込み、ほかの `id` を指すイベント、古い行、壊れた行 | `TestDirectorRead_FoldsHeadlessRegistrations`、`_AStage8a1FileReadsAsBefore`、`TestDirectorAttachHeadless_*`(6 本)、`TestDirectorRecordHerdrIDs_Refusals`、`TestDirectorMarkLaunched_*`、`TestDirectorMarkStopped_*`(4 本)、`TestDirectorMarkForceStopped_*`、`TestDirectorHeadlessWriters_ACutOffWriteIsRestored`、同時の attach の 1 本 | D5、D6、E7 red |
| StartDirector: イベントの順、引数、label と agent 名の token、雛形のファイル、各段の失敗の後始末、`launched` の前の `--replace`(AC4)、落ちたあとの `starting`、manifest に書かないこと | `TestStartDirector_*`(10 本)、`TestDirector_AStartThatDiedAfterItsAttachedLeavesAStartingRegistration`。実機の K1〜K3 | D1〜D3、C9、E2〜E11、P18、Z2 red。D4、E1、E12、E13 は生き残り(下) |
| StopDirector: 成功、not found、label の違い、close の失敗、pane の中から、`stopped` 済み、detach 後、`starting`、2 つの台帳と id の振り直し(AC5)、`--force`(AC14)、workspace の中から | `TestStopDirector_*`(12 本)、`TestDirector_TwoLedgersOnOneHerdrServerNeverCloseEachOther`。実機の K2〜K4、R2、R3 | A1〜A14、B1〜B5、B7、B8、B11、C1〜C7、C10 red(A12〜A14 は seat のテストで) |
| token: 12 桁の hex、agent 名の規則、2 回の start で違う | `TestDirectorAttachHeadless_TokensAndTheNamesBuiltFromThem`、`TestDirector_TwoLedgersOnOneHerdrServerNeverCloseEachOther` | - |
| 雛形と spawn の拒否 | `TestRenderDirectorPrompt_*`(3 本)、`TestRenderRolePrompt_DirectorIsNotASeatTemplate`、`TestDirectorRoleInputErr_IsExactAndCaseSensitive`、`TestOrgSpawn_DirectorRole_RejectedBeforeAnyManifestWrite` | P19〜P23、G1〜G3、G5 red |
| permission: role director の解決、codex の `--add-dir` | `TestStartDirector_PermissionModeOfRoleDirector`、`TestCodexWritableRootArgs`。実機の K4(guarded で引数なし) | Z2 red |
| Integration: CLI の `start`・`stop`・`status`、`ralph status` の行と JSON、`stop --all` の案内 | `TestOrgDirectorStart_*`(6 本)、`TestOrgDirectorStop_*`(6 本)、`TestOrgDirectorStatus_ShowsHeadlessDirectors`、attach・replace・detach の 3 本、`TestStatusCmd_Director*`(2 本)、`TestOrgStopAll_*`(3 本)、`TestOrgSpawn_RoleDirectorIsRefusedBeforeTheModelFallback` | B6、B9、C8、D7、D8、P1、P3〜P17、G3c、G4、Z1 red。P2 は生き残り |
| Regression: go1.26 と go1.25.8、`-race`、8a-1 のテスト | 上の 2 回の全体の実行(名前も同じ)と `-race` | - |
| Edge: `USER` がない | Go のテストはない(8a-1 の status のテストは昇格したセッションだけ)。実機の K1 | - |
| Edge: 台帳が main worktree のものでない(cwd と codex の `--add-dir`) | `TestOrgDirectorStart_ReadsTheMainWorktreeRalphToml`、`TestCodexWritableRootArgs` | Z2 red |
| Edge: 雛形のファイルが書けない、agent start の時間切れ、id の書き足しの失敗 | `TestStartDirector_AFailedStepClosesWhatItMadeAndRecordsTheStop`(prompt write)、`TestStartDirector_BoundsTheAgentStart`、`TestStartDirector_ARecordThatFailsStillClosesWhatHerdrMade`。実機の K1 | E3、E9 red。attached の書き込みの失敗は E1 で生き残り |

## Mutation

- 依頼にあった 7 つの観点はどれも mutant で確かめ、どれも red になった。C-c と close の前の label の確認(A1〜A11、B1、B2、B11)、`--force` が何も送らないこと(B1、B3〜B5、B7)、自分の pane と workspace の拒否(C1〜C10)、見張りの前の attached の拒否(D1〜D3)、登録を先に書く順(E2〜E5)、打ち方の `--state-dir`(P1、P3〜P23)、spawn の role director の拒否(G1〜G5)
- 86 件のうち 78 件が `-run` で絞った回で red だった。生き残った 8 件をパッケージ全体で流し直すと、A12、A13、A14 は org の全体で red になった(`TestOrgStopDisband_UnansweredGet_LeavesItAlone`、`TestOrgStop_PaneNotConfirmed_NoCtrlCNoClose`)。この 3 つは seat と共有する `verbs.go` の `confirmSeatPane`・`confirmOrgWorkspace` の変異で、director のテストには当たらないが seat のテストが止める。最後まで生き残ったのは D4、E1、E12、E13、P2 の 5 件で、81 件が red になった
- 生き残り 5 件には probe のテストを書いた(evidence の 5.4)。どれも HEAD で通り、狙った mutant のもとで落ち、狙っていない mutant(D1、P3)では通る

| ID | 変異 | 結果 | probe |
| --- | --- | --- | --- |
| D4 | `checkDirectorStart` の最初の `Store.Read` の失敗を無視する(見張りの検査が先に走る) | org と cli の全体でも生き残り | PD4: `director.jsonl` をディレクトリにして start を打つ。HEAD では見張りの検査が呼ばれず、D4 では 1 回呼ばれる |
| E1 | `AttachHeadless` の失敗(attached の書き込みの失敗)で start を止めない | 同上 | PE1: 1 回目の write を ENOSPC にする。HEAD は herdr を打たずに ENOSPC を返す。E1 は workspace を作ってから `"" is not a registration id` で後始末にも失敗する |
| E12 | workspace の作成の失敗で、herdr の出力を workspace の id として残す | 同上 | PE12: 失敗した作成が出力も返す偽の herdr。HEAD の herdr の呼び出しは `workspace_create` だけで、E12 は `workspace_get` を足す |
| E13 | tab の作成の失敗で、herdr の出力を pane の id として残す | 同上 | PE13: 同じ偽で tab の作成を失敗させる。E13 は `pane_get` を足す |
| P2 | `startAttachedError` の `cur == nil` の案内で `--state-dir` を付けない | cli の全体でも生き残り | PP2: `orgDirectorRuntime{hintStateDir: ...}` の helper を直接呼ぶ。HEAD は `ralph org director status --state-dir '/work/the ledger'` を出す |

- E12 と E13 が生き残るのは、テストの `fakeHerdr` が作成の失敗で `""` を返すからだ。本物の `driver.Herdr.WorkspaceCreate` と `TabCreate`(`internal/org/driver/herdr.go` 123〜126、153〜156)は、コマンドが失敗すると stdout を error と一緒に返す。HEAD の `createWorkspaceAndTab` はその値を捨てるが、捨てる行を消してもテストは気づかない

## Failure analysis

| Test | Error | Root cause | Proposed fix |
| --- | --- | --- | --- |
| なし | - | - | - |

落ちたテストはない。

## Regression checks

| Previously broken behavior | Status | Evidence |
| --- | --- | --- |
| 8a-1 の attach・detach・status | 保たれている。headless の登録があるときだけ stop の案内に変わる | 8a-1 のテストが両方の版で PASS。P9、P10 red。実機の K2 の attach の拒否 |
| seat の stop の label の確認(`confirmSeatPane`・`confirmOrgWorkspace` を director と共有) | 保たれている | A5〜A14 が red(A12〜A14 は seat のテストで) |
| 古い印の見張りで start を断る(verify の AC7 の穴、ca98c248) | 保たれている | `TestOrgDirectorStart_WatchGate`、D7 red |
| 前に揺れた doctor、watcher、gate のテスト | 落ちていない | evidence の 2.1 |
| 今の main(PR #230)との合わせ | Go は通る。`docs/tech-debt/README.md` が衝突する | evidence の 6。`git merge-tree` で衝突は 1 ファイル、合わせた tree で org と cli が rc 0 |

## Test gaps

1. D4: 最初に `director.jsonl` を読めないときの拒否を固定するテストがない。D4 のもとでは見張りの検査が先に走る(probe の PD4 で確認)。CLI の見張りの検査は見張りがなければ立ち上げるので、おそらく見張りを 1 つ立ててから断ることになり、影響はそれだけとみている(コードを読んだ推測で、未確認です)。一番小さい足し方は、`TestStartDirector_RefusesBeforeWritingOrCallingHerdr` に「director.jsonl unreadable」の行(setup で `os.Mkdir(f.store.Path(), 0o755)`、`wantGate` は false)を足すこと。probe の PD4 と同じ形
2. E1: attached の書き込みの失敗を固定するテストがない。`TestStartDirector_ARecordThatFailsStillClosesWhatHerdrMade` の表は 2 回目と 3 回目の write だけを失敗させる。E1 のもとでは herdr の workspace が作られ、後始末も記録できずに残る。足し方は、1 回目の write を ENOSPC にして、ENOSPC が返ること、herdr の呼び出しがないこと、ファイルが空か無いことを確かめるテスト(probe の PE1)。今の表の行は ws-1 が閉じられることを全行で求めるので、表に足すなら行ごとの期待値を足す必要がある
3. E12、E13: 偽の herdr が本物の driver と違い、作成の失敗で出力を返さない。足し方は、失敗した作成が出力も返す包みの偽(probe の `junkOutputHerdr`)で、herdr の呼び出しが `workspace_create` だけ(tab の場合は `workspace_create`、`tab_create`、`workspace_get`、`workspace_close`)であることを確かめるテスト。`fakeHerdr` 自体に「失敗でも出力を返す」つまみを足せば、seat の spawn の同じ経路にも使える
4. P2: `startAttachedError` の `cur == nil`(AttachHeadless に断られたあと読み直すと今の登録がない、または読めない)を通るテストがない。CLI の実行で再現するには競合が要る。足し方は、`TestOrgDirector_NotRestoredErrorNamesTheStatusCommand` と同じく helper を直接呼ぶ表のテスト(probe の PP2、`--state-dir` のありとなし)
5. A12〜A14 は director のテストには当たらない。seat のテストが共有の関数を止めているので、今は穴ではない。director の stop が `confirmSeatPane`・`confirmOrgWorkspace` を使わなくなったときに穴になる
6. Go のテストで通らない文のうち、C-c の送信の失敗の note(`director_stop.go` 372)、`abort` の中で `MarkStopped` や `Detach` が失敗したとき(`director_start.go` 652、659〜662)、CLI の設定・model・`os.Getwd` の失敗にはテストがない(Coverage の項)
7. `USER` がない headless の登録は、Go のテストがなく、実機の K1 だけで見た
8. 本物の herdr で見たのは claude の director だけ。codex の director は verify が隔離した server で見ている。claude の director が雛形を読んで手順どおりに動くか(最初の `director status` で自分の `id` を確かめる、など)は、どちらも見ていない
9. `ralph init` で `/director` skill と `director.md` が scaffold に入ることは打っていない。macOS 以外と、herdr 0.7.5 以外の版も見ていない
10. 今の main と合わせた tree では shell のテストを流していない。`docs/tech-debt/README.md` の衝突は /pr の前の rebase で解く必要がある

## Verdict

- Pass: yes。`./scripts/run-test.sh`(changed、full に広がった)は rc 0(shell 41 本すべて OK、Go 8 パッケージすべて ok)。`go test -count=1 ./internal/...` は go1.25.8 と go1.26.0 の両方で rc 0(cli 699/639、org 800/1,682 と skip 1、ほか 6 パッケージ。名前は両方の版で同じ)。足した 67 本のテストはどれも PASS。`-race` もパッケージ全体で通った。テストは本物の herdr と osascript を打たず(canary は 0 回)、repo の台帳に書いていない
- Fail: なし
- Blocked: なし
- Verified: 計画の Test plan の Unit、Integration、Regression、Edge case の各項目(上の対応表)。mutant 86 件のうち 81 件が red で、依頼の 7 つの観点はすべて red。HEAD の binary で、`starting` の表示(verify が Partially verified とした AC10)、agent start の時間切れと閉じられないときの後始末、落ちた start の登録の stop、自分の pane と workspace の中からの拒否を stub の herdr で見た。本物の herdr では、`HERDR_WORKSPACE_ID` が pane に渡ることと、自分の workspace の中からの拒否、別の workspace からの stop を見た
- Not verified: Test gaps の 1〜10。生き残り 5 件はどれも、テストを足せば止まることを probe で確かめた。どれも PR を止めるものではない

## Cycle 2

- Date: 2026-10-15(JST 21:38〜22:20)
- Plan: docs/plans/active/2026-10-15-org-director-start.md(`Status: Approved`。`./scripts/plan-visual.sh digest` は f65d7c59746c で、`- Approved:` の digest と同じ)
- Tester: tester subagent (Claude Opus 5.5)。パイプラインの 2 回目(`.harness/state/standard-pipeline/cycle-count.json` は 2)
- Scope: HEAD 5dce71de(origin/main f21df66c を 1cdddbbb で合わせた。`git diff origin/main...HEAD` は 56 ファイル、+10,973/-426)。cycle 1 の b6c86188 から Go が変わったのは 3 コミット、7 ファイル(`internal/org/director_start.go`、`prompts.go`、`prompts/director.md` とそれぞれのテスト、`internal/cli/org_director_headless.go` とそのテスト)。3243c499 は start の `--config` を絶対パスにして雛形の `ralph org` のコマンドに渡す直し、a539a765 は self-review cycle 2 の C2-L1〜C2-L4 の直し(雛形の stop の行は `--state-dir` だけ、`{{CONFIG}}` を `{{CONFIG_LINE}}` に改名、skill の 2 か所、テストのコメント)、4ef33195 は cycle 1 の Test gaps 1〜4(D4、E1、E12/E13、P2)のテスト。残りは記録と文書と、merge で入った main の PR #230。behavioral test だけを実行した(静的解析は /verify の担当)。環境は macOS(arm64)、go1.26.0 と go1.25.8(`GOTOOLCHAIN`)、git 2.49.0。実行中の load average は 4〜17(別のセッション)。go test の 10 分の時間切れには当たらなかった
- Evidence: `docs/evidence/test-2026-10-15-org-director-start.md` の `## Cycle 2`(C2-1 run-test.sh の抜粋、C2-2 Go の結果と数・`-race`・canary・coverage、C2-3 stub の herdr に対する HEAD の binary、C2-4 mutation と probe のテスト)。生の出力は `docs/evidence/test-2026-10-15-org-director-start.log` の末尾(`===== /test cycle 2` の行から)、wrapper の全ログは `docs/evidence/verify-2026-10-15-123924.log` で、どちらも gitignore の対象なのでコミットしない
- テストもコードも変えていない。mutant と probe のテストは scratchpad に置き、`go test -overlay` で読ませた。go:embed の雛形 `prompts/director.md` も overlay で差し替わる。worktree に書いたのは、この節、evidence の `## Cycle 2`、insight event の 1 行だけ

### Test execution

| Suite / Command | Tests | Passed | Failed | Skipped | Duration |
| --- | --- | --- | --- | --- | --- |
| `./scripts/run-test.sh`(changed。`templates/base/ralph.toml` の変更で full に広がった。go1.26.0) | shell 41 本、Go 8 パッケージ | shell 41 本すべて OK、Go 8 パッケージすべて ok | 0 | shell 1(下) | 701 秒、rc 0 |
| `GOTOOLCHAIN=go1.25.8 go test -count=1 -v -coverprofile ./internal/...` | cli: top-level 700、subtest 641。org: 804、1,692。org/driver: 48、89。org/protocol: 15、56。config: 37、34。insights: 36、7。scaffold: 32、0。upgrade: 114、13 | 同じ数から skip を引いた数 | 0 | org 1、scaffold 2 | 250 秒(cli 245.6 s、org 95.9 s)、rc 0 |
| `go test -race -count=1 -v ./internal/org/ ./internal/cli/`(go1.26.0) | org 804/1,692、cli 700/641 | 同じ数から skip を引いた数 | 0 | org 1 | 249 秒(org 102.0 s、cli 241.8 s)、rc 0、DATA RACE なし |
| `go test -count=1 -v` で残りの 6 パッケージ(go1.26.0) | go1.25.8 と同じ数 | 同じ | 0 | scaffold 2 | 1 パッケージ 1〜3 秒、rc 0 |
| うち cycle 1 のあとで足したテスト | top-level 5(org 4、cli 1)、subtest 11 | 両方の版ですべて | 0 | 0 | - |
| mutation(`go test -overlay`、go1.26.0、`-run` で絞った回) | mutant 26 件、別のパッケージでの流し直し 3 件、対照 3 件 | red 24 件。流し直しはどれも red | 生き残り 2 件(K15、K16) | - | 1 件 3〜71 秒 |
| 生き残りをパッケージ全体で | K15 は org と cli、K16 は org | - | どれも生き残り | - | 115 秒、266 秒、80 秒 |
| probe のテスト(overlay だけ、コミットしない) | org 2 本 | HEAD で通る | 狙った mutant(K15、K16)のもとでそれぞれ落ちる。K15 の probe は K7 と K16 で、K16 の probe は K15 で通る | - | 1 回 1〜3 秒 |
| HEAD の binary と stub の herdr(下の「実機の確認」) | 4 段 | すべて期待どおり | 0 | - | 数秒 |

- shell は 41 本すべてが `OK` で終わり、`FAIL:` の行 25 本はどれも 0 だった。cycle 1 と同じ形で抜き出した 168 行は、行番号と時間を除くと 2 行だけ違った。`plan-visual tests: 103 passed` が `243 passed` になった行(merge で入った PR #230 のテスト)と、wrapper のログの名前の行。PASS で始まる行は 4,283 本で、cycle 1 の 4,143 本との差 140 はこの plan-visual の分。Skipped は前からある `tests/test-secret-scan-branch.sh` の 1 件(git が 2.49.0 なので、2.41 より古い git の場合を飛ばす)
- `run-test.sh` の中の Go は cli(213.7 s)、org(82.7 s)、org/driver を流し直し、config、insights、org/protocol、scaffold、upgrade は `(cached)` だった。この 5 つは go1.25.8 と go1.26.0 の両方の `-count=1` で ok を確かめた
- go1.25.8 の名前と結果の行 4,318 本と、go1.26.0 の 2 回(`-race` の org と cli、残り 6 パッケージ)を合わせた 4,318 本は同じだった。cycle 1 の 4,301 本との差は 17 本の追加で、消えた行はない。17 本のうち 16 本はこの cycle で足したテスト、1 本は merge で入った `TestPlanDigest_MatchesScript/archive/2026-10-15-plan-visual-easy.md`(PR #230 が archive した計画)
- Skipped は cycle 1 と同じ 3 本(`TestEscalate_NoDesktopNotificationOffDarwinIsSkipped`、scaffold の `TestBaseFS_WithMockFS` と `TestAvailablePacks_WithMockFS`)
- 前に揺れたテスト(`TestRunDoctorOpts_ProbeModelsFalse_NoSubprocess`、`TestRunDoctorFull_StrictFlipsExitCode_DriftedCore`、`TestRunWatcher_TimeoutIndependentOfSmallInterval`、`TestOrgGate_InterruptStopsTheScriptAndRecordsNothing`)は、どの実行でも落ちなかった。`TestOrgGate_InterruptStopsTheScriptAndRecordsNothing` は go1.25.8 で 2.51 s、`-race` で 2.76 s だった
- AC13: run-test.sh、Go の 3 回、mutation のすべての回、probe のテストは、PATH の先頭に canary の `herdr` と `osascript` を置いて走らせた(呼ばれたら argv を記録して exit 97。cycle 1 の canary と違い、herdr の `--version` も本物に渡さない)。呼ばれた回数は 0 回。最後に手で同じ PATH から打つと 2 つとも記録されたので、PATH の検索は canary に届く。テストのあと、main のチェックアウトにも worktree にも `.harness/state/org` はなく、main のチェックアウトの下に `director.jsonl` は 1 つもない

### cycle 2 の変更とテストの対応

「mutant」の列は、その変更の判定を壊した mutant が落ちたかどうか(id は下の Mutation の表と evidence の C2-4)。

| 変更 | テスト | mutant |
| --- | --- | --- |
| start が `--config` を受けたとき、雛形の中で director が打つ `ralph org` のコマンド(最初の手順の status、要約のあとの status、指示の文)の `--state-dir` のあとに `--config '<絶対パス>'` を付ける(`{{ORG_FLAGS}}`) | `TestRenderDirectorPrompt_GivesEveryRalphOrgCommandTheConfig`、`TestStartDirector_CarriesTheConfigIntoThePrompt`、`TestOrgDirectorStart_ReadsTheMainWorktreeRalphToml`。実機の W2、W3 | K1、L1、K5、K6、K8〜K10 red。K1 は cli のテストでも red(K1c) |
| `--config` を、director の cwd ではなく start を打った場所に対して絶対パスにする | `TestStartDirector_CarriesTheConfigIntoThePrompt` の相対パスの行、`TestOrgDirectorStart_ReadsTheMainWorktreeRalphToml` の `--config names the worktree's ralph.toml`。実機の W2、W2b | K2、K2b、K2c、L3 red。K2c は cli のテストでも red(K2cc) |
| `RenderDirectorPrompt` は絶対パスでない ConfigPath を断る | `TestRenderDirectorPrompt_RefusesBadVarsAndLeftoverPlaceholders` に足した 2 行 | K4、K2b red |
| `--config` がなければ、雛形のどこにも `--config` はなく、`- 設定:` の行は main worktree の ralph.toml を名指す | 上の 3 本の `without --config`、`no config path`、`no --config` の行。実機の W1 | K12、K13 red |
| 雛形の見出しの `- 設定:` の行(`{{CONFIG_LINE}}`。パスのあとに `(start の --config)`) | `TestRenderDirectorPrompt_GivesEveryRalphOrgCommandTheConfig` の見出しの確認 | K7、K14 red。skill の 4 つの写しがこの印を名指すことを固定するテストはない(K16 生き残り) |
| 雛形の stop の行と `ralph status` の指示は `--state-dir` だけ(C2-L1) | stop の行は上の 3 本、`ralph status` の文は prompts のテスト | K3、K11 red。K3 は cli のテストでも red(K3c)。start が出す stop の打ち方に `--config` を足す L4 も red |
| `--config` のパスを絶対パスにできないときは start を断る | なし | K15 生き残り(org と cli の全体でも) |
| cycle 1 の Test gaps 1〜4 のテスト(4ef33195) | `TestStartDirector_RefusesBeforeWritingOrCallingHerdr` の `director.jsonl unreadable`、`TestStartDirector_AFailedWriteOfAttachedCallsNoHerdr`、`TestStartDirector_AFailedCreatesOutputIsNotAnID`(2 行)、`TestOrgDirectorStart_AnAttachedRefusalWithNoCurrentNamesStatus`(2 行) | cycle 1 の生き残り D4、E1、E12、E13、P2 はどれも red |

### 実機の確認(HEAD の binary)

HEAD を go1.25.8 で build した binary を、`env -i` で PATH に ralph、cycle 1 と同じ状態つきの stub の `herdr`、記録するだけの `osascript`、`/usr/bin`、`/bin` だけを置いて打った(evidence の C2-3)。台帳は scratch の `--state-dir`、start の cwd は `ralph.toml` のない scratch のディレクトリ。`--config` のファイルは空白と single quote を含む `<run>/cfg dir/it's ralph.toml` で、`[org.permissions].default = "autonomous"` と `[org.permissions.roles].director = "guarded"` を持つ。雛形の行は、start の cwd とは別の `<run>/elsewhere/sub` から `sh -c` でそのまま打った。repo の `.harness/state/org/` には書いていない。

1. W0: `ralph org watch --all --detach` で見張りを立てた
2. W1(`--config` なし): start は `started d1 (... permission=autonomous ...)` と `stop it with: ralph org director stop --id d1 --state-dir '<run>/ledger'` を出した。雛形のファイルの `- 設定:` の行は `main worktree の ralph.toml(台帳が main worktree のものでなければ、cwd の ralph.toml)` で、ファイルに `--config` は 1 つもない。最初の手順の status の行(13 行目)を打つと rc 0 で d1 を出し、stop の行(51 行目)を打つと d1 を閉じて stopped を書いた
3. W2(相対パスの `--config '../cfg dir/it'\''s ralph.toml'`): start は `permission=guarded` で立ち、stdout の stop の打ち方は `--state-dir` だけだった。雛形の `- 設定:` の行は `'<run>/cfg dir/it'\''s ralph.toml'(start の --config)`。13 行目と 30 行目の status、33 行目の指示の文は `--state-dir '<run>/ledger' --config '<run>/cfg dir/it'\''s ralph.toml'` で、34 行目の `ralph status` と 51 行目の stop は `--state-dir` だけ。`--config` は 4 回(見出し 1、status 2、指示 1)出る。13 行目を打つと rc 0 で d2 を出した。`--config` のファイルの `[org.director].enabled` を false に書き換えてから同じ行を打つと `director: disabled ([org.director].enabled = false; ...)` と `enabled: false` が出て、start の cwd で `--config` なしに打った status は `enabled: true` を出した。雛形の行が、start の読んだファイルを読むことを確かめた。ファイルは元の bytes に戻し、stop の行で d2 を閉じた
4. W2b(対照): start に渡したのと同じ相対パスを `<run>/elsewhere/sub` から打つと、そこでは存在しない `<run>/elsewhere/cfg dir/it's ralph.toml` を指す。それでも status は rc 0 で `enabled: true` を出した。`config.Load` は存在しないファイルを既定値として読む(`internal/config/config.go` 304 行目の `fs.ErrNotExist`)ので、相対パスのまま雛形に入っていれば、director の `ralph org` のコマンドはエラーを出さずに既定値を読む。絶対パスにする直しは、この取り違えを防ぐ。存在しないファイルを既定値として読む動き自体は cycle 2 の前からある
5. W3(絶対パスの `--config`): 雛形は W2 と同じだった(`d<N>` をならすと差はない)
- 後片付け: `watch --stop` のあと、run の dir を名に持つ process は 0、`osascript` の記録は 0 行。stub の herdr の記録は 3 回の start と 3 回の stop の 30 行だけだった

### Coverage

- Statement(go1.25.8、各パッケージ自身のテストの profile): `internal/org` 94.6%、`internal/cli` 87.8%、`internal/config` 93.3%、`internal/org/driver` 93.1%、`internal/org/protocol` 100.0%、`internal/insights` 86.1%、`internal/scaffold` 75.7%、`internal/upgrade` 91.2%(cycle 1 は org 94.6%、cli 87.7%)
- `internal/org/director_start.go` は 216/231 文(93.5%。cycle 1 は 212/228)。cycle 1 で通らなかった最初の `Store.Read` の失敗(D4)と attached の書き込みの失敗(E1)は、新しいテストが通す。cycle 2 で増えた 3 文のうち、`--config` のパスを絶対パスにできないときの return(556〜558、mutant K15)だけが通らない。残りの 14 文は cycle 1 の Coverage の項と同じ
- `internal/cli/org_director_headless.go` は 95/98 文(96.9%。cycle 1 は 94/98)。`startAttachedError` の `cur == nil`(P2)を新しいテストが通す。残りの 3 文は設定と model の解決の失敗、`os.Getwd` の失敗
- `internal/org/prompts.go` は 45/46 文(97.8%)。`RenderDirectorPrompt` の cycle 2 の分岐(絶対パスの確認、`--config` のありとなし)はすべて通り、通らないのは埋め込みの雛形を読めないとき(169〜171)だけ
- Branch: Go の標準ツールには branch coverage がないので測っていない。代わりに mutation で分岐ごとの判定を確かめた(下)

### Mutation

- 対象は cycle 2 の 2 つの直し(3243c499、a539a765)と cycle 1 の生き残り 5 件。ファイルを overlay で差し替え、org と cli とも cycle 1 と同じ `-run` で流した。生き残りはパッケージ全体でも流した。手を加えないファイルを overlay で読ませた対照の X0(`director_start.go`)、X0t(`prompts/director.md`)、X0c(`org_director_headless.go`)は通った。どの差し替えも元の文字列がちょうど 1 か所にあり、飛ばした mutant はない
- handoff が名指した 4 つはどれも red だった。config を prompt の vars から落とす(K1)、絶対パスにしない(K2。`RenderDirectorPrompt` の拒否と合わせて外した K2b、director の cwd に対して絶対パスにする K2c と L3 も)、stop の行に `{{ORG_FLAGS}}`(K3)、`RenderDirectorPrompt` が相対パスを受ける(K4)。cycle 1 の生き残り D4、E1、E12、E13、P2 は、4ef33195 のテストを狙った行に戻すとどれも red になった
- 26 件のうち 24 件が red、2 件(K15、K16)が生き残った。K1、K2c、K3 を cli のテストで流し直した 3 回(K1c、K2cc、K3c)もどれも red で、雛形の stop の行は prompts、org の start、CLI の 3 層のテストが止める

| ID | 変異 | 結果 | 落としたテスト |
| --- | --- | --- | --- |
| K1 | `writePrompt` が ConfigPath を prompt の vars に渡さない | red | `TestStartDirector_CarriesTheConfigIntoThePrompt`(cli では `TestOrgDirectorStart_ReadsTheMainWorktreeRalphToml`) |
| L1 | CLI が `--config` を `StartDirector` に渡さない | red | `TestOrgDirectorStart_ReadsTheMainWorktreeRalphToml` |
| K2 | `checkDirectorStart` が ConfigPath を絶対パスにしない(雛形が相対パスを断る) | red | `TestStartDirector_CarriesTheConfigIntoThePrompt` |
| K2b | K2 に加えて `RenderDirectorPrompt` も相対パスを受ける | red | 同上、`TestRenderDirectorPrompt_RefusesBadVarsAndLeftoverPlaceholders` |
| K2c | 相対パスを director の Cwd に対して絶対パスにする | red | `TestStartDirector_CarriesTheConfigIntoThePrompt`(cli では `ReadsTheMainWorktreeRalphToml`) |
| L3 | CLI が相対パスを director の cwd(main worktree のルート)とつなぐ | red | `TestOrgDirectorStart_ReadsTheMainWorktreeRalphToml` |
| K3 | 雛形の stop の行が `{{ORG_FLAGS}}` を描く | red | `TestRenderDirectorPrompt_GivesEveryRalphOrgCommandTheConfig`、`TestStartDirector_CarriesTheConfigIntoThePrompt`(cli では `ReadsTheMainWorktreeRalphToml`) |
| K4 | `RenderDirectorPrompt` が相対パスを受ける | red | `TestRenderDirectorPrompt_RefusesBadVarsAndLeftoverPlaceholders` |
| K5 | config のパスを shell の 1 語に quote しない | red | `GivesEveryRalphOrgCommandTheConfig`、`CarriesTheConfigIntoThePrompt` |
| K6 | `{{ORG_FLAGS}}` で `--config` を `--state-dir` の前に置く | red | 同上 |
| K7 | ConfigPath があっても見出しが `--config` なしの文のまま | red | `GivesEveryRalphOrgCommandTheConfig` |
| K8 | 要約のあとの status が `--state-dir` だけ | red | 同上 |
| K9 | 最初の手順の status が `--state-dir` だけ | red | `GivesEveryRalphOrgCommandTheConfig`、`CarriesTheConfigIntoThePrompt` |
| K10 | 指示の文が `--state-dir` だけを言う | red | `GivesEveryRalphOrgCommandTheConfig` |
| K11 | `ralph status` に `{{ORG_FLAGS}}` を付けよと言う | red | 同上 |
| K12 | 空の ConfigPath も絶対パスにする(cwd になる) | red | `CarriesTheConfigIntoThePrompt` |
| K13 | 空の ConfigPath でも `--config` の分岐に入る | red | `GivesEveryRalphOrgCommandTheConfig`、`PinsTheHeadlessDirectorsProcedure`、`CarriesTheConfigIntoThePrompt` |
| K14 | 雛形から `- 設定:` の行を消す | red | `GivesEveryRalphOrgCommandTheConfig` |
| L4 | start が出す stop の打ち方に、start の `--config` を足す | red | `TestOrgDirectorStart_LaunchesRegistersAndStops`(codex の行) |
| D4 | 最初の `Store.Read` の失敗を無視する(cycle 1 の生き残り) | red | `TestStartDirector_RefusesBeforeWritingOrCallingHerdr` の `director.jsonl unreadable` |
| E1 | attached の書き込みの失敗で start を止めない(同) | red | `TestStartDirector_AFailedWriteOfAttachedCallsNoHerdr` |
| E12 | workspace の作成の失敗で、herdr の出力を workspace の id として残す(同) | red | `TestStartDirector_AFailedCreatesOutputIsNotAnID/workspace_create` |
| E13 | tab の作成の失敗で、herdr の出力を pane の id として残す(同) | red | `TestStartDirector_AFailedCreatesOutputIsNotAnID/tab_create` |
| P2 | `startAttachedError` の `cur == nil` の案内で `--state-dir` を付けない(同) | red | `TestOrgDirectorStart_AnAttachedRefusalWithNoCurrentNamesStatus/with_--state-dir` |
| K15 | `--config` のパスを絶対パスにできなくても start を止めない | org と cli の全体でも生き残り | - |
| K16 | `- 設定:` の行の印を `prompts.go` と `prompts_test.go` の両方で変える(skill は古い印のまま) | org の全体でも生き残り | - |

- K15: 本物の `filepath.Abs` が失敗するのは cwd を読めないときだけで、どのテストも `absPath` を ConfigPath のときに失敗させない。K15 のもとでは ConfigPath が "" になり、start は `--config` のない雛形で成功する(probe で `err = <nil>` を確かめた)
- K16: 印を名指すのは `prompts.go`、`prompts_test.go`、skill の 4 つの写しだけで、skill を読む Go のテストは `TestDirectorSkill_NamesTheInboxWaitTimeoutPhrase`(受信箱の文だけを見る)しかない。shell のテストと script にも印を読むものはない。今の 4 つの写しと `prompts.go` の印は同じ文字列(ASCII の括弧)であることを確かめた

### Failure analysis

| Test | Error | Root cause | Proposed fix |
| --- | --- | --- | --- |
| なし | - | - | - |

落ちたテストはない。

### Regression checks

| Previously broken behavior | Status | Evidence |
| --- | --- | --- |
| cycle 1 の 67 本と既存のテスト | すべて PASS。名前の並びは cycle 1 から 17 行増えただけで、消えた行はない | evidence の C2-2.1 |
| `--config` のない start の雛形(cycle 1 と同じく `--state-dir` だけ) | 保たれている。雛形に `--config` は 1 つもない | K12、K13 red。実機の W1 |
| 雛形の stop の行と start が出す stop の打ち方は `--state-dir` だけ(stop は ralph.toml を読まない) | 保たれている | K3、K3c、L4 red。実機の W1〜W3 |
| cycle 1 の生き残り D4、E1、E12、E13、P2 | 4ef33195 のテストがどれも止める | 上の Mutation の表 |
| 8a-1 の attach・detach・status | 保たれている | 8a-1 のテストが両方の版で PASS |
| 前に揺れた doctor、watcher、gate のテスト | 落ちていない | evidence の C2-2.1 |

### Test gaps

1. K15: `--config` のパスを絶対パスにできないときの拒否(`director_start.go` 556〜558)を固定するテストがない。起きるのは cwd を読めないときだけなので、影響は小さいとみている(推測で、未確認です)。一番小さい足し方は probe の `TestProbe_C2_AConfigPathThatCannotBeMadeAbsoluteIsRefused`(evidence の C2-4.4)。`absPath` を ConfigPath のときだけ失敗させ(`spawn_test.go` 2242 行目と同じ差し替え方)、error が `org: director: start: resolve the --config path: ` で始まること、herdr の呼び出しがないこと、`director.jsonl` と `prompts` ができないことを確かめる。HEAD で通り、K15 で落ち、K7 と K16 では通る
2. K16: skill の 4 つの写しの「始める」の 3 が名指す印 `(start の --config)` と、`RenderDirectorPrompt` が `- 設定:` の行に書く印を突き合わせるテストがない。足し方は probe の `TestProbe_C2_DirectorSkillNamesTheConfigLineMarker`。ConfigPath つきで雛形を描き、`- 設定:` の行のパスのあとの文字列を取り出し、4 つの SKILL.md がそれを backtick で囲んで含むことを確かめる(8a-1 の `TestDirectorSkill_NamesTheInboxWaitTimeoutPhrase` と同じ形)。HEAD で通り、K16 で 4 つとも落ち、K15 では通る
3. 本物の herdr と agent で、director が雛形の `--config` つきのコマンドを打つかは見ていない。HEAD の binary で見たのは、雛形の行を `sh -c` で打つと start の読んだファイルを読むところまで
4. 台帳が main worktree のもので、director が start と別の場所(main worktree のルート)で動く場合の相対パスの `--config` は、Go のテスト(`--config names the worktree's ralph.toml`。L3 と K2cc が red)だけで見た。HEAD の binary では台帳が scratch のもので、director は start と同じ場所で動いた
5. 存在しないファイルを指す `--config` で start を打つ場合は見ていない。`config.Load` が既定値として読むので、start も雛形の行も既定値を読み、食い違いは出ないとみている(コードを読んだ推測で、未確認です)
6. cycle 1 の Test gaps のうち、1〜4(D4、E1、E12/E13、P2)は閉じた。5〜10 は今回の変更の外で、そのまま残る

### Verdict

- Pass: yes。`./scripts/run-test.sh`(changed、full に広がった)は rc 0(shell 41 本すべて OK、Go 8 パッケージすべて ok)。go1.25.8 の `go test -count=1 ./internal/...` は rc 0(cli 700/641、org 804/1,692 と skip 1、ほか 6 パッケージ)。go1.26.0 では org と cli を `-race` で、残り 6 パッケージをそのまま流し、どれも rc 0 で DATA RACE もなかった。テストの名前は両方の版で同じ。cycle 1 のあとで足したテスト 5 本(subtest 11)はどれも PASS。テストは本物の herdr と osascript を打たず(canary は 0 回)、repo の台帳に書いていない
- Fail: なし
- Blocked: なし
- Verified: start の `--config` が、start を打った場所に対する絶対パスになって、雛形の中で director が打つ `ralph org` のコマンドに 1 語で付くこと、stop の行と `ralph status` と start が出す stop の打ち方には付かないこと、`--config` がなければ雛形に `--config` が出ないことを、テストと mutant 26 件(red 24 件)で確かめた。handoff が名指した 4 つの mutant と cycle 1 の生き残り 5 件はどれも red。HEAD の binary では、描かれた status の行を別の場所から打つと start の読んだファイルを読むこと(中身を変えて確かめた)、描かれた stop の行で director を止められることを stub の herdr で見た
- Not verified: Test gaps の 1〜6。どれも PR を止めるものではない
