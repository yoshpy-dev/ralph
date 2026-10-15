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

## Cycle 3

- Date: 2026-10-16(JST 00:44〜01:30)
- Plan: docs/plans/active/2026-10-15-org-director-start.md(`Status: Approved`。`./scripts/plan-visual.sh digest` は 96796452ceae で、`- Approved:` の digest と同じ)
- Tester: tester subagent (Claude Opus 5.5)。パイプラインの 3 回目で、上限を 3 に上げた最後の回。`.harness/state/standard-pipeline/cycle-count.json` は計画どおり 2 のままにした
- Scope: HEAD 3504ee30(`git diff origin/main...HEAD` は 59 ファイル、+13,110/-457)。cycle 2 の報告 6556bce5 からの 10 コミットのうち、Go を変えたのは 2 つ。caf1d025 は D12(codex の director の autonomous と edits を `codex_verified` によらず `ErrDirectorCodexWorkspaceWrite` で断り、`codexWritableRootArgs` を leader だけに戻す)と D13(`--config` の相対の `[org].agmsg_home` を `checkStartAgmsgHome` で断る。判定 `agmsgHomeIsRelative` は見張りの子と共有し、`RALPH_ORG_AGMSG_HOME` があれば警告する)。83287d39 は `CodexModeRunsUnderWorkspaceWrite` の追加、doctor の `codexSeatModesPossible` が role director の edits と autonomous を数えないこと、D13 の拒否を 3 つの場所で確かめるテスト、help と文の直し。残りは計画、記録、文書。behavioral test だけを実行した(静的解析は /verify の担当)。環境は macOS(arm64)、go1.26.0 と go1.25.8(`GOTOOLCHAIN`)、git 2.49.0。実行中の load average は 4〜11(別のセッション)
- Evidence: `docs/evidence/test-2026-10-15-org-director-start.md` の `## Cycle 3`(C3-1 run-test.sh の抜粋、C3-2 Go の結果と数・`-race`・canary・coverage、C3-3 HEAD と直す前の binary の `ralph doctor`、C3-4 mutation と probe のテスト)。生の出力は `docs/evidence/test-2026-10-15-org-director-start.log` の末尾(`===== /test cycle 3` の行から)、wrapper の全ログは `docs/evidence/verify-2026-10-15-154510.log` で、どちらも gitignore の対象なのでコミットしない
- テストもコードも変えていない。mutant と probe のテストは scratchpad に置き、`go test -overlay` で読ませた。worktree に書いたのは、この節、evidence の `## Cycle 3`、insight event の 1 行だけ

### Test execution

| Suite / Command | Tests | Passed | Failed | Skipped | Duration |
| --- | --- | --- | --- | --- | --- |
| `./scripts/run-test.sh`(changed。`templates/base/ralph.toml` の変更で full に広がった。go1.26.0) | shell 41 本、Go 8 パッケージ | shell 41 本すべて OK、Go 8 パッケージすべて ok | 0 | shell 1(cycle 2 と同じ) | 818 秒、rc 0 |
| `GOTOOLCHAIN=go1.25.8 go test -count=1 -v -coverprofile ./internal/...` | cli: top-level 706、subtest 658。org: 804、1,696。org/driver: 48、89。org/protocol: 15、56。config: 37、34。insights: 36、7。scaffold: 32、0。upgrade: 114、13 | 同じ数から skip を引いた数 | 0 | org 1、scaffold 2 | 230 秒(cli 226.1 s、org 100.8 s)、rc 0 |
| `go test -race -count=1 -v ./internal/org/ ./internal/cli/`(go1.26.0) | org 804/1,696、cli 706/658 | 同じ数から skip を引いた数 | 0 | org 1 | 365 秒(org 133.4 s、cli 355.6 s)、rc 0、DATA RACE なし |
| `GOTOOLCHAIN=go1.25.8 go test -race -count=1 -v ./internal/org/ ./internal/cli/` | org 804/1,696、cli 706/658 | 同じ | 0 | org 1 | 323 秒(org 119.0 s、cli 316.0 s)、rc 0、DATA RACE なし |
| `go test -count=1 -v` で残りの 6 パッケージ(go1.26.0) | go1.25.8 と同じ数 | 同じ | 0 | scaffold 2 | 1 パッケージ 1〜3 秒、rc 0 |
| うち cycle 2 のあとで足したテスト | top-level 6(cli)、subtest 23(cli 17、org 6)。名前を変えた org の subtest 2 本は消えた | 4 回の Go の実行のすべてで | 0 | 0 | - |
| mutation(`go test -overlay`、go1.26.0、`-run` で絞った回) | mutant 30 件、対照 4 件 | red 24 件 | 生き残り 6 件 | - | 1 件 39〜73 秒 |
| 生き残りをパッケージ全体で | B2 と C3 は org と cli、D3、D3b、D8、E2 は cli | - | 6 件とも生き残り | - | 264〜315 秒 |
| probe のテスト(overlay だけ、コミットしない) | cli 2 本 | HEAD で通る | 狙った mutant(D3b、D8)のもとでそれぞれ落ちる。もう一方の mutant と、対照の D3、D1 では通る | - | 1 回 2〜4 秒 |
| cycle 2 の mutant の一覧を HEAD で流し直す | 33 件(対照 3 件を含む) | 判定は cycle 2 と同じ(red 27 件) | 生き残りは cycle 2 と同じ K15、K15c、K16 | - | 186 秒 |
| HEAD と直す前の binary の `ralph doctor`(下の「実機の確認」) | 6 通り × 2 | すべて期待どおり | 0 | - | 数秒 |

- shell は 41 本すべてが `OK` で終わり、`FAIL:` の行 25 本はどれも 0 だった。cycle 2 と同じ形で抜き出した 168 行は、行番号と時間を除くと wrapper のログの名前の 1 行だけ違った。PASS で始まる行は 4,283 本で cycle 2 と同じ。Skipped は前からある `tests/test-secret-scan-branch.sh` の 1 件(git が 2.49.0 なので、2.41 より古い git の場合を飛ばす)
- `run-test.sh` の中の Go は cli(317.3 s)、org(128.2 s)、org/driver を流し直し、config、insights、org/protocol、scaffold、upgrade は `(cached)` だった。insights は org を import するので、`(cached)` は同じ org のコードでの前の実行の結果で、影響がない証拠ではない。この 5 つは go1.25.8 と go1.26.0 の両方の `-count=1` で ok を確かめた
- go1.25.8 の名前と結果の行 4,345 本と、go1.26.0 の 2 回(`-race` の org と cli、残り 6 パッケージ)を合わせた 4,345 本は同じだった。2 つの `-race` の回の org と cli の行(3,864 本ずつ)も同じだった。cycle 2 の 4,318 本との差は、29 本の追加と 2 本の削除。消えた 2 本は、caf1d025 が名前を変えた `TestStartDirector_PermissionModeOfRoleDirector` の codex の 2 行(D12 が断る start を通ると期待していた行)
- Skipped は cycle 2 と同じ 3 本(`TestEscalate_NoDesktopNotificationOffDarwinIsSkipped`、scaffold の `TestBaseFS_WithMockFS` と `TestAvailablePacks_WithMockFS`)
- 前に揺れたテスト(`TestRunDoctorOpts_ProbeModelsFalse_NoSubprocess`、`TestRunDoctorFull_StrictFlipsExitCode_DriftedCore`、`TestRunWatcher_TimeoutIndependentOfSmallInterval`、`TestOrgGate_InterruptStopsTheScriptAndRecordsNothing`)は、どの実行でも落ちなかった。`TestOrgGate_InterruptStopsTheScriptAndRecordsNothing` の subtest でいちばん遅かったのは go1.25.8 の `-race` の SIGTERM の 4.64 s で、`gateKillGrace` の 5 s にも 15 s の予算にも届いていない
- AC13: run-test.sh、Go の 4 回、mutation のすべての回、probe のテストは、PATH の先頭に canary の `herdr` と `osascript` を置いて走らせた(呼ばれたら argv を記録して exit 97)。呼ばれた回数は 0 回。最後に手で同じ PATH から打つと 2 つとも記録されたので、PATH の検索は canary に届く。テストのあと、main のチェックアウトにも worktree にも `.harness/state/org` はなく、main のチェックアウトの下に `director.jsonl` は 1 つもない
- `~/.codex/config.toml` は最初の実行の前と最後の実行のあとで、更新時刻(Oct 15 17:51:28 2026)、大きさ(33,673)、sha256(ac1a4821…)が同じだった。本物の codex は起動していない

### cycle 3 の変更とテストの対応

「mutant」の列は、その変更の判定を壊した mutant が落ちたかどうか(id は下の Mutation の表と evidence の C3-4)。

| 変更 | テスト | mutant |
| --- | --- | --- |
| D12: codex の director は、解決した mode が autonomous か edits なら `codex_verified` によらず断る(`ErrDirectorCodexWorkspaceWrite`、何も書かず、herdr を打たない) | `TestStartDirector_PermissionModeOfRoleDirector` の codex の 5 行、`TestStartDirector_RefusesBeforeWritingOrCallingHerdr` の `permission rule` と `permission rule with codex_verified`、`TestOrgDirectorStart_RefusedWithNothingWritten` の codex の 2 行 | A1(外す)、A3(autonomous だけ断る)、A4(claude も断る)、A5(sentinel を包まない)、A6(codex_verified で通す) red |
| D12 の検査は `permissionArgsForDriver` より前(codex_verified が false でも D12 の文で断る) | 上の `before codex_verified` の行が、codex_verified の文が出ないことも確かめる | A2(後ろへ移す)red。codex_verified の文で断るようになり、`errors.Is` が外れる |
| D12: `codexWritableRootArgs` は leader だけ。codex の director の引数に `--add-dir` は入らない | `TestCodexWritableRootArgs` の director の行、`PermissionModeOfRoleDirector` の codex guarded の 3 行(引数を全体で比べる) | B1(director の分岐を戻す)と B3(分岐と `StartDirector` の呼び出しを戻す)red。呼び出しだけを戻す B2 は、関数が director に nil を返すので同じ動き(生き残り、等価) |
| `CodexModeRunsUnderWorkspaceWrite`(autonomous と edits) | 上のテスト、doctor の 2 本、leader の `--add-dir` の行 | C1(guarded も true)と C2(edits を false)red。C3(guarded 以外を true)は読み込んだ設定では等価(生き残り) |
| D13: `--config` の相対の `agmsg_home` を、どこで打っても断る(絶対、`~/`、空は通す) | `TestOrgDirectorStart_ARelativeAgmsgHomeInTheConfigIsRefused`(6 行)、`ARelativeAgmsgHomeIsRefusedWhereverStartRuns`(3 行)、`ARelativeAgmsgHomeWithoutConfigIsNotRefused` | D1(外す)、D2 と D2b(2 つの場所が違うときだけ断る)、D7(model の既定の警告のあとで断る)、D9(`~/` も断る)red。D2 と D2b を落としたのは、同じ場所から解く 2 行(main worktree のルート、main worktree のない台帳)で、linked worktree の行ではない |
| D13 の条件の `Changed("config")` | - | D3(外す)は `--config` の既定値が "" なので等価(生き残り)。`configPath != ""` を外す D3b は `--config ""` のときだけ違い、どのテストも渡さない(生き残り、Test gaps 1) |
| `RALPH_ORG_AGMSG_HOME` があれば stderr に警告を 1 行出して続ける | `TestOrgDirectorStart_WarnsThatTheAgmsgHomeVariableDoesNotReachTheDirector` | D4(外す)、D5(stdout に出す)、D6(`--config` のときだけ出す)red |
| 共通の判定 `agmsgHomeIsRelative`(裸の `~` は相対、`~/` は相対でない) | 上の D13 のテスト、`TestOrgWatchDetach_ARelativeAgmsgHomeIsPassedAbsoluteToTheRelocatedChild` | E1(裸の `~` を相対でないとする)と E3(`~/` を相対とする)red。E2(空を相対とする)は、どちらの呼び出し元も空を渡さないので等価(生き残り) |
| doctor の `codexSeatModesPossible` は role director の edits と autonomous を数えず、guarded の director は数える | `TestCheckCodexAgmsgWritableRoot_DirectorWorkspaceWriteModeIsNotCounted`(4 行)、`TestCodexSeatModesPossible` の 3 行。実機の D1〜D6 | F1(director を数え直す)、F2(guarded の director も数えない)、F3(既定を role director として当てる)red |
| help の D12 と D13 の段落 | `TestOrgDirectorStart_HelpStatesTheCodexAndAgmsgHomeRules` | H1(「wherever it runs」を消す)red |
| enabled の検査を D13 より先にする | なし | D8(順を入れ替える)生き残り(Test gaps 2) |

### 実機の確認(HEAD の binary)

HEAD と、直す前の 0bfbb054(caf1d025 と同じ Go。doctor が role director を数えていたころ)を go1.25.8 で build し、`ralph doctor` を `env -i` で打った(evidence の C3-3)。HOME、CODEX_HOME、TMPDIR は scratch のディレクトリで、PATH は binary のディレクトリ、`/usr/bin`、`/bin` だけ。各回は `git init` したディレクトリに、codex と claude を pool に持ち、`scripts/send.sh` を置いた絶対パスの agmsg_home を名指す `ralph.toml` を置いた。scratchpad は `/private/tmp` の下にあり、そのままでは `/tmp` が暗黙の writable root として agmsg の保存先を覆って全部の回が pass になるので、codex の設定で `/tmp` と `$TMPDIR` を外した。本物の `~/.codex` には触れていない。

1. D1(codex_verified = true、default guarded、`director = "edits"`)と D2(同じで `director = "autonomous"`): HEAD は「Codex sandbox (agmsg writable root)」を pass(`not needed: no role that can use a codex model resolves to edits or autonomous ...`)にした。直す前の binary は同じ設定で warn を出した。83287d39 の直しを binary で確かめた。pass の文が role director の edits の設定では文字どおりに正しくない点は、verify の C3-V3 が挙げている
2. D3(対照。`implementer = "edits"`): 両方の binary で warn。seat の role の edits は数える
3. D4(codex_verified = false、default autonomous、`director = "guarded"`、`sandbox_mode = "workspace-write"`): 両方で warn(理由 (b)、guarded の codex は人の設定を継ぐ)。guarded の director は数える
4. D5(対照。roles なし、default guarded)と D6(対照。roles なし、default autonomous、codex_verified = false、`sandbox_mode = "workspace-write"`): 両方で pass。D4 の warn は director の行から来る
- どの回も `doctor rc=1` で、落ちた検査は claude が PATH にない `Claude Code CLI` の 1 つだけ。scratch の HOME には何も書かれなかった

D12 と D13 の拒否は、verify の cycle 3 が HEAD と直す前の binary で確かめている(verify の evidence の C3-3、Q1〜Q11)ので、この回では binary で打ち直していない。

### Coverage

- Statement(go1.25.8、各パッケージ自身のテストの profile): `internal/org` 94.6%、`internal/cli` 87.7%、`internal/config` 93.3%、`internal/org/driver` 93.1%、`internal/org/protocol` 100.0%、`internal/insights` 86.1%、`internal/scaffold` 75.7%、`internal/upgrade` 91.2%(cycle 2 は cli 87.8%、ほかは同じ)
- 変わった関数はすべて通る: D12 の検査(`director_start.go` 541〜548)、`CodexModeRunsUnderWorkspaceWrite`(`permissions.go` 85〜87)、`checkStartAgmsgHome`(`org_director_headless.go` 253〜265)、`agmsgHomeIsRelative`(`org_watch.go` 395〜397)、`codexSeatModesPossible`(`doctor_codex_writable_root.go` 344〜368)
- `internal/org/director_start.go` は 216/232 文(93.1%。cycle 2 は 216/231)。cycle 3 で新しく通らなくなったのは、`checkDirectorStart` の中の `permissionArgsForDriver` の error の return(549〜552)。cycle 2 までは codex_verified の前の codex の director がここを通ったが、今は D12 が先に断る。今ここに届くのは autonomous、edits、guarded のどれでもない mode だけで、`config.Load` はそういう mode を断る。テストを足しても届かない分岐とみている(コードを読んだ推測で、未確認です)。残りの通らない文は cycle 2 の Coverage の項と同じ(行番号がずれただけ)
- `internal/cli/org_director_headless.go` は 102/105 文(97.1%。cycle 2 は 95/98)。通らない 3 文は cycle 2 と同じ(設定と model の解決の失敗、`os.Getwd` の失敗)
- `internal/org/permissions.go` は 29/32 文(90.6%)、`internal/cli/doctor_codex_writable_root.go` は 271/276 文(98.2%)。どちらも通らない文は差分の外
- Branch: Go の標準ツールには branch coverage がないので測っていない。代わりに mutation で分岐ごとの判定を確かめた(下)

### Mutation

- 対象は cycle 3 の 2 つの直し(caf1d025、83287d39)。ファイルを overlay で差し替え、org と cli とも `-run` で流した。`-run` は cycle 2 のものに、変わったコードのテスト(permission、doctor の 2 つの検査、agmsg_home)を足した。生き残りはパッケージ全体でも流した。手を加えないファイルを overlay で読ませた対照の X0(`director_start.go`)、X0p(`permissions.go`)、X0c(`org_director_headless.go`)、X0d(`doctor_codex_writable_root.go`)は通った。どの差し替えも元の文字列がちょうど 1 か所にあり、すべての mutant が build できた
- handoff が名指した 9 つの mutant の結果は次のとおり。D12 を外す(A1)、D12 を `permissionArgsForDriver` の後ろへ移す(A2)、`codexWritableRootArgs` に director の分岐を戻す(B1)、D13 を外す(D1)、D13 を 2 つの場所が違うときだけにする(D2、パスを解決して比べる D2b も)、env の警告を外す(D4)、doctor が role director を数え直す(F1)、`CodexModeRunsUnderWorkspaceWrite` が guarded にも true を返す(C1)は red。`Changed("config")` の条件を外す(D3)は生き残り、等価
- 30 件のうち 24 件が red、6 件(B2、C3、D3、D3b、D8、E2)が生き残った。そのうち 4 件(B2、C3、D3、E2)は等価で、2 件(D3b、D8)は閉じられる穴

| ID | 変異 | 結果 | 落としたテスト |
| --- | --- | --- | --- |
| A1 | D12 を外す | red | `TestStartDirector_PermissionModeOfRoleDirector`、`TestStartDirector_RefusesBeforeWritingOrCallingHerdr`、`TestOrgDirectorStart_RefusedWithNothingWritten` |
| A2 | D12 を `permissionArgsForDriver` の後ろへ移す | red | 同上(`before codex_verified` の行が codex_verified の文で断られる) |
| A3 | D12 が autonomous だけを断る | red | 同上 |
| A4 | D12 が driver を見ない(claude も断る) | red | director の start を通すテスト 41 本 |
| A5 | D12 が `ErrDirectorCodexWorkspaceWrite` を `%w` で包まない | red | A1 と同じ 3 本 |
| A6 | D12 が codex_verified のあるときは通す | red | 同上(`with codex_verified` の行) |
| B1 | `codexWritableRootArgs` に director の分岐を戻す | red | `TestCodexWritableRootArgs`(director の 2 行) |
| B2 | `StartDirector` が `codexWritableRootArgs` をまた呼ぶ(分岐は戻さない) | 生き残り(全体でも)、等価 | - |
| B3 | B1 と B2 を合わせる(cycle 2 の配線を戻し、D12 は残す) | red | `TestCodexWritableRootArgs` |
| C1 | `CodexModeRunsUnderWorkspaceWrite` が guarded にも true | red | `PermissionModeOfRoleDirector` の codex guarded の 3 行、`CodexWritableRootArgs` の leader guarded、doctor のテストなど 13 本 |
| C2 | 同じく edits に false | red | `PermissionModeOfRoleDirector`、`CodexWritableRootArgs`、doctor の 2 本など 6 本 |
| C3 | 同じく guarded 以外に true | 生き残り(全体でも)、読み込んだ設定では等価 | - |
| D1 | D13 を外す | red | `ARelativeAgmsgHomeInTheConfigIsRefused`、`ARelativeAgmsgHomeIsRefusedWhereverStartRuns` |
| D2 | D13 を director と見張りの場所が違うときだけにする | red | 同上(同じ場所から解く行) |
| D2b | D2 で、パスを symlink まで解いて比べる | red | 同上 |
| D3 | D13 の `Changed("config")` を外す | 生き残り(全体でも)、等価 | - |
| D3b | D13 の `configPath != ""` を外す | 生き残り(全体でも) | -(probe PD3b が落とす) |
| D4 | `RALPH_ORG_AGMSG_HOME` の警告を外す | red | `WarnsThatTheAgmsgHomeVariableDoesNotReachTheDirector` |
| D5 | 警告を stdout に出す | red | 同上 |
| D6 | 警告を `--config` のときだけ出す | red | 同上 |
| D7 | D13 を model の既定の解決のあとにする | red | `ARelativeAgmsgHomeInTheConfigIsRefused`(stderr が空であること) |
| D8 | D13 を enabled の検査より前にする | 生き残り(全体でも) | -(probe PD8 が落とす) |
| D9 | D13 が共通の判定でなく「`/` で始まらない」で断る | red | `ARelativeAgmsgHomeInTheConfigIsRefused` の `~/` と空の行、`LaunchesRegistersAndStops` など 5 本 |
| E1 | `agmsgHomeIsRelative` が裸の `~` を相対でないとする | red | `ARelativeAgmsgHomeInTheConfigIsRefused` の `a bare ~` |
| E2 | 同じく空を相対とする | 生き残り(全体でも)、等価 | - |
| E3 | 同じく `~/` を相対とする | red | D9 と同じ 5 本と `TestOrgWatchDetach_ARelativeAgmsgHomeIsPassedAbsoluteToTheRelocatedChild` |
| F1 | doctor が role director の edits と autonomous を数え直す | red | `DirectorWorkspaceWriteModeIsNotCounted`、`TestCodexSeatModesPossible` |
| F2 | doctor が role director を guarded も含めて数えない | red | 同上(guarded の director の行) |
| F3 | doctor が既定の mode を role director として当てる | red | doctor のテスト 30 本 |
| H1 | help から「wherever it runs」を消す | red | `TestOrgDirectorStart_HelpStatesTheCodexAndAgmsgHomeRules` |

- B2: HEAD の `codexWritableRootArgs` は role director に nil を返すので、呼んでも引数は変わらない。分岐も戻す B3 は red
- C3: `config.Load` は `[org.permissions].default` と `[org.permissions.roles]` の値に autonomous、edits、guarded だけを許し(`internal/config/config.go` 405〜411 行)、`ResolvePermissionMode` はそのどれかを返す。読み込んだ設定では「guarded 以外」と「autonomous か edits」は同じになる
- D3: `--config` の既定値は ""(`internal/cli/org.go` 52 行)なので、`Changed("config")` が false なら `configPath` も ""。org-watch-gate の cycle 3 の A8a と同じ形
- E2: `config.Load` は空の `agmsg_home` を既定値で埋め(`config.go` 392〜394 行)、見張りの子の `orgWatchChildAgmsgHome` は、変数が空で `--config` がないとき判定の前に返る。どちらの呼び出し元も空を渡さない
- D3b: `--config ""`(flag を空で渡す)のときだけ違う。HEAD はそのとき cwd の `./ralph.toml` を読み、相対の `agmsg_home` でも断らない。director も見張りも `--config` なしと同じに解くので、HEAD の動きは D13 の理由と合う。この形を固定するテストはない(Test gaps 1)
- D8: `enabled = false` と相対の `agmsg_home` を両方持つ `--config` を渡すテストがなく、2 つの拒否の順は固定されていない。計画も順を決めていない(Test gaps 2)
- probe のテスト 2 本(evidence の C3-4.4)は、HEAD で通り、狙った mutant のもとで落ち、もう一方の mutant と、対照の D3 と D1 のもとでは通る
- cycle 2 の mutant の一覧 33 件を HEAD で流し直すと、判定は cycle 2 と同じだった(evidence の C3-4.5)。caf1d025 は `TestOrgDirectorStart_LaunchesRegistersAndStops` の codex の行を guarded に書き直したが、その行で落ちていた L4(start が出す stop の打ち方に `--config` を足す)は今も red。テストの書き直しで cycle 2 の mutant を落とす力は減っていない

### Failure analysis

| Test | Error | Root cause | Proposed fix |
| --- | --- | --- | --- |
| なし | - | - | - |

落ちたテストはない。

### Regression checks

| Previously broken behavior | Status | Evidence |
| --- | --- | --- |
| cross-review cycle 2 の指摘 1: codex の director が autonomous か edits で立ち、workspace-write のため `ralph org start --plan` を通せない | 直っている。start が `codex_verified` によらず断る | A1〜A6 red。verify の binary の Q1〜Q3(直す前の binary は Q8 で立てた) |
| cross-review cycle 2 の指摘 2: `--config` の相対の `agmsg_home` を director と見張りが別の場所から解く | 直っている。start がどこで打っても断る | D1、D2、D2b、D7、D9 red。verify の binary の Q4、Q5 |
| self-review cycle 3 の C3-L4: doctor が、start の断る codex の director の mode を workspace-write の role として数える | 直っている | F1〜F3 red。実機の D1、D2(直す前の binary は warn) |
| cycle 1 と cycle 2 のテストと既存のテスト | すべて PASS。名前の並びは cycle 2 から 29 行増え、名前を変えた 2 行が消えた | evidence の C3-2.1 |
| cycle 2 の mutant の判定 | 同じ(red 27 件、生き残り K15、K15c、K16) | evidence の C3-4.5 |
| 前に揺れた doctor、watcher、gate のテスト | 落ちていない | evidence の C3-2.1 |

### Test gaps

1. D3b: `--config ""`(flag を空で渡す)の start を固定するテストがない。HEAD は cwd の `./ralph.toml` の相対の `agmsg_home` を断らず、`configPath != ""` を外す mutant でも全テストが通る。影響は小さいとみている(`--config ""` を打つ人は少ないという推測で、未確認です)。一番小さい足し方は probe の `TestProbe_C3_AnEmptyConfigFlagIsNotRefusedForARelativeAgmsgHome`(evidence の C3-4.4)。`directorStartLedger` の cwd に相対の `agmsg_home` の `ralph.toml` を置き、`--config ""` で start が d1 を launched まで書くことを確かめる
2. D8: `enabled = false` と相対の `agmsg_home` を両方持つ `--config` で、どちらの拒否が出るかを固定するテストがない。計画は順を決めていないので、今の順を固定するかどうかは判断による。固定するなら probe の `TestProbe_C3_EnabledFalseIsRefusedBeforeARelativeAgmsgHome`
3. `checkDirectorStart` の `permissionArgsForDriver` の error の return(549〜552)は、D12 のあとは読み込んだ設定では届かない。テストで届かせるには、`config.Load` を通らない mode を `o.Config` に直接入れるしかない
4. guarded の codex の director が、人の codex の設定(`sandbox_mode = "danger-full-access"` など)の下で `ralph org start --plan` の `git worktree add` を通せるかは、本物の codex で見ていない(verify の cycle 3 の Coverage gaps と同じ。D12 の案内はこれを前提にする)
5. `RALPH_ORG_AGMSG_HOME` が director の pane に届かないことは、本物の herdr server の pane の環境で見ていない。cycle 1 の実機の確認(evidence の 4.2)で pane に渡る変数は `HERDR_` で始まるものだけを抜き出したので、この変数の有無は記録にない
6. cycle 2 の Test gaps 1〜2(K15、K16)は、cycle 3 でもテストが足されておらず、そのまま残る(C3-4.5 で K15、K15c、K16 が生き残り)。3〜6 も今回の変更の外で、そのまま残る

### Verdict

- Pass: yes。`./scripts/run-test.sh`(changed、full に広がった)は rc 0(shell 41 本すべて OK、Go 8 パッケージすべて ok)。go1.25.8 の `go test -count=1 ./internal/...` は rc 0(cli 706/658、org 804/1,696 と skip 1、ほか 6 パッケージ)。org と cli の `-race` は go1.26.0 と go1.25.8 のどちらでも rc 0 で DATA RACE もなく、go1.26.0 の残り 6 パッケージも rc 0。テストの名前は両方の版で同じ。cycle 2 のあとで足したテスト 6 本(subtest 23)はどれも PASS。テストは本物の herdr と osascript を打たず(canary は 0 回)、repo の台帳に書いていない。`~/.codex/config.toml` は変わっていない
- Fail: なし
- Blocked: なし
- Verified: D12(codex の director の autonomous と edits を codex_verified によらず、`permissionArgsForDriver` より前に断る、`codexWritableRootArgs` は leader だけ)、D13(`--config` の相対の `agmsg_home` を、2 つの場所が同じでも断る、`RALPH_ORG_AGMSG_HOME` の警告)、doctor が role director の edits と autonomous を数えないことを、テストと mutant 30 件(red 24 件、生き残りのうち等価 4 件)で確かめた。handoff が名指した 9 つの mutant のうち、等価の `Changed("config")` を除く 8 つは red。HEAD の binary の `ralph doctor` で、director の edits と autonomous が pass になり、直す前の binary では warn になることを見た
- Not verified: Test gaps の 1〜6。どれも PR を止めるものではない

## Cycle 4

- Date: 2026-10-16(JST 04:28〜05:30)
- Plan: docs/plans/active/2026-10-15-org-director-start.md(`Status: Approved`。`./scripts/plan-visual.sh digest` は 58be042da6fd で、`- Approved:` の digest と同じ)
- Tester: tester subagent (Claude Opus 5.5)。パイプラインの 4 回目。`.harness/state/standard-pipeline/cycle-count.json` は 2 のままで、この回も書き換えていない
- Scope: 905fc0a7 で始め、この回のテストを足した 10efa82c で終えた(10efa82c で `git diff origin/main...HEAD` は 60 ファイル、+15,580/-476)。cycle 3 の報告 b0adfc63 からのコミットのうち、Go を変えたのは 3 つ。b888d637 は D14(lock の下で、stop が読んだときの workspace と pane の id と記録を比べる `MarkStoppedIfUnchanged`。違えば `ErrDirectorChangedDuringStop` を返して何も書かない)と D15(`/director` skill の description を YAML として読める形にし、`internal/config/skill_frontmatter_test.go` を足す)。5097b67e は self-review cycle 4 の LOW(文書の一覧、拒否の注記を CLI の stderr に 1 回だけ出す、frontmatter のテストの名前と doc)。44b80391 は S12(記録のときに登録がすでに stopped なら stop を成功にする `DirectorStopResult.AlreadyStopped`、`org: director: ` を 1 回にする `directorRecordStopError`、`MarkStoppedIfUnchanged` と `MarkForceStopped` の writer の行)。残りは計画と記録。behavioral test だけを実行した(静的解析は /verify の担当)。環境は macOS(arm64)、go1.26.0 と go1.25.8(`GOTOOLCHAIN`)、git 2.49.0、YAML の確認に Ruby 3.4.10(Psych)。実行中の load average は 4〜14(別のセッション)
- Evidence: `docs/evidence/test-2026-10-15-org-director-start.md` の `## Cycle 4`(C4-1 run-test.sh の抜粋、C4-2 Go の結果と数・`-race`・新しいテストの PASS の行・canary・coverage、C4-3 YAML の確認と frontmatter のテストの感度、C4-4 mutation、C4-5 足したテスト)。生の出力は `docs/evidence/test-2026-10-15-org-director-start.log` の末尾(`===== /test cycle 4` の行から)、wrapper の全ログは `docs/evidence/verify-2026-10-15-192839.log` と `docs/evidence/verify-2026-10-15-195306.log` で、どれも gitignore の対象なのでコミットしない
- コードは変えていない。テストを 2 本足した(10efa82c、下の「この回で足したテスト」)。mutant と probe のテストは scratchpad に置き、`go test -overlay` で読ませた。ほかに worktree に書いたのは、この節、evidence の `## Cycle 4`、insight event の 1 行だけ

### Test execution

| Suite / Command | Tests | Passed | Failed | Skipped | Duration |
| --- | --- | --- | --- | --- | --- |
| `./scripts/run-test.sh`(changed。`templates/base/ralph.toml` の変更で full に広がった。go1.26.0。10efa82c) | shell 41 本、Go 8 パッケージ | shell 41 本すべて OK、Go 8 パッケージすべて ok | 0 | shell 1(前の回と同じ) | 511 秒、rc 0 |
| 同じコマンドを 905fc0a7 で(1 回目。下の理由で HEAD の結果に数えない) | 同じ | 同じ | 0 | 同じ | 665 秒、rc 0 |
| `GOTOOLCHAIN=go1.25.8 go test ./... -count=1 -v -coverprofile`(10efa82c) | cli: top-level 708、subtest 662。org: 812、1,712。org/driver: 48、89。org/protocol: 15、56。config: 39、53。insights: 36、7。scaffold: 32、0。upgrade: 114、13 | 同じ数から skip を引いた数 | 0 | org 1、scaffold 2 | 218 秒(cli 217.1 s、org 81.9 s)、rc 0 |
| 同じコマンドを 905fc0a7 で | org だけ 810、1,710。ほかは上と同じ | 同じ | 0 | 同じ | 234 秒、rc 0 |
| `go test -race -count=1 -v ./internal/org/ ./internal/cli/`(go1.26.0、10efa82c) | org 812/1,712、cli 708/662 | 同じ数から skip を引いた数 | 0 | org 1 | 232 秒(org 86.9 s、cli 224.7 s)、rc 0、DATA RACE なし |
| `GOTOOLCHAIN=go1.25.8 go test -race -count=1 -v ./internal/org/ ./internal/cli/`(10efa82c) | 同じ | 同じ | 0 | org 1 | 225 秒(org 85.2 s、cli 218.4 s)、rc 0、DATA RACE なし |
| `go test -count=1 -v` で残りの 6 パッケージ(go1.26.0、10efa82c) | go1.25.8 と同じ数 | 同じ | 0 | scaffold 2 | 1 パッケージ 1〜3 秒、rc 0 |
| うち cycle 3 のあとで足したテスト | top-level 12(cli 2、config 2、org 8)、subtest 39(cli 4、config 19、org 16)。この回で足した 2 本(subtest 2)を含む | Go のすべての実行で | 0 | 0 | - |
| Ruby の `YAML.safe_load` で、追跡している SKILL.md をすべて読む | 54 本 | 905fc0a7 と 10efa82c のどちらでも 54 本 ok | 直す前の `/director` の 4 本は 4 本とも `Psych::SyntaxError` | - | 1 秒未満 |
| frontmatter のテストを、`git archive` の写しの skill を 1 本ずつ壊して流す(Y0〜Y6) | 7 回 | 対照 2 回と、値を引用符で囲んだ Y4 は PASS | 壊した Y1、Y2、Y3、Y5 は狙いどおり FAIL | - | 1 回 1 秒前後 |
| mutation(`go test -overlay`、go1.26.0、`-run` で絞った回) | mutant 42 件、対照 6 件 | red 39 件 | 生き残り 3 件(F1、F3、F7) | - | 1 件 1〜51 秒 |
| 生き残りをパッケージ全体で | F1、F3、F7 は config。この回のテストを外した P9h、P11h、H7h は org と cli | - | 6 件とも生き残り | - | F は 1〜2 秒、ほかは 307〜312 秒 |

- shell は 41 本すべてが `OK` で終わり、`FAIL:` の行 25 本はどれも 0 だった。cycle 3 と同じ形で抜き出した 168 行は、行番号と時間を除くと、Go の行の `(cached)` と wrapper のログの名前だけが違った。PASS で始まる行は 4,283 本で cycle 3 と同じ。Skipped は前からある `tests/test-secret-scan-branch.sh` の 1 件(git が 2.49.0 なので、2.41 より古い git の場合を飛ばす)
- 1 回目の `run-test.sh` は 905fc0a7 で走らせたが、Go の部分が走っている間(04:38:24)に 1 本目のテストを worktree に書いた。org の binary はその前に build されていたとみている(org は Go の部分の始めから 81.966 s かかっている。推測で、未確認です)。そのため 1 回目は HEAD の結果に数えず、10efa82c で打ち直した 2 回目を gate にした
- 2 回目の Go は config と org を流し直し、ほかの 6 パッケージは `(cached)` だった。cli の `(cached)` は 1 回目の go1.26.0 の結果(212.1 s)で、2 回の間に変わったのは `internal/org` の `_test.go` だけで、cli の build には入らない。insights は org を import するが、org のテストでないコードは変わっていない。8 パッケージとも、go1.25.8 と go1.26.0 の両方の `-count=1` で ok を確かめた
- go1.25.8 の名前と結果の行 4,396 本と、go1.26.0 の 2 回(`-race` の org と cli、残り 6 パッケージ)を合わせた 4,396 本は同じだった(両方を `LC_ALL=C sort`)。2 つの `-race` の回の org と cli の行(3,894 本ずつ)も同じ。cycle 3 の 4,345 本との差は 51 本の追加だけで、消えた行はない
- Skipped は cycle 3 と同じ 3 本(`TestEscalate_NoDesktopNotificationOffDarwinIsSkipped`、scaffold の `TestBaseFS_WithMockFS` と `TestAvailablePacks_WithMockFS`)
- 前に揺れたテスト(`TestRunDoctorOpts_ProbeModelsFalse_NoSubprocess`、`TestRunDoctorFull_StrictFlipsExitCode_DriftedCore`、`TestRunWatcher_TimeoutIndependentOfSmallInterval`、`TestOrgGate_InterruptStopsTheScriptAndRecordsNothing`)は、どの実行でも落ちなかった。`TestOrgGate_InterruptStopsTheScriptAndRecordsNothing` の subtest でいちばん遅かったのは 905fc0a7 の go1.25.8 の SIGHUP の 2.49 s で、`gateKillGrace` の 5 s にも 15 s の予算にも届いていない
- AC13: run-test.sh の 2 回、Go のすべての実行、mutation のすべての回、probe のテストは、PATH の先頭に canary の `herdr` と `osascript` を置いて走らせた(呼ばれたら argv を記録して exit 97)。呼ばれた回数は 0 回。最後に手で同じ PATH から打つと 2 つとも記録されたので、PATH の検索は canary に届く。テストのあと、main のチェックアウトにも worktree にも `.harness/state/org` はなく、main のチェックアウトの下に `director.jsonl` は 1 つもない。`~/.codex/config.toml` は更新時刻(Oct 15 17:51:28 2026)、大きさ(33,673)、sha256(ac1a4821…)が cycle 3 と同じだった

handoff が名指したテストと D14 のテストは、`-run` で名前を指定し、`-v` で走らせた(905fc0a7、go1.26.0)。どれも skip されていない。主な PASS の行は次のとおり。全体の行は evidence の C4-2.3 にある。10efa82c の go1.25.8 の全体の実行でも、どれも PASS だった。

```text
--- PASS: TestStopDirector_SucceedsWhenTheStopWasRecordedFirst (0.01s)
    --- PASS: TestStopDirector_SucceedsWhenTheStopWasRecordedFirst/another_stop_recorded_it_first (0.00s)
    --- PASS: TestStopDirector_SucceedsWhenTheStopWasRecordedFirst/start's_cleanup_recorded_it (0.00s)
    --- PASS: TestStopDirector_SucceedsWhenTheStopWasRecordedFirst/start's_cleanup_recorded_it,_with_force (0.00s)
    --- PASS: TestStopDirector_SucceedsWhenTheStopWasRecordedFirst/start's_cleanup_recorded_it_while_a_forced_stop's_workspace_close_failed (0.00s)
--- PASS: TestStopDirector_ARecordThatFailsSaysItsPrefixOnce (0.00s)
--- PASS: TestStartDirector_ACleanupWhoseRecordFailsSaysThePrefixOnce (0.00s)
--- PASS: TestOrgDirectorStop_AStopRecordedFirstNamesNoStopToRun (0.00s)
    --- PASS: TestOrgDirectorStop_AStopRecordedFirstNamesNoStopToRun/the_workspace_closed (0.00s)
    --- PASS: TestOrgDirectorStop_AStopRecordedFirstNamesNoStopToRun/a_forced_stop_whose_workspace_close_failed (0.00s)
--- PASS: TestDirectorHeadlessWriters_ACutOffWriteIsRestored (0.01s)
    --- PASS: TestDirectorHeadlessWriters_ACutOffWriteIsRestored/MarkStoppedIfUnchanged_of_the_current_registration (0.00s)
    --- PASS: TestDirectorHeadlessWriters_ACutOffWriteIsRestored/MarkForceStopped_of_the_current_registration (0.00s)
--- PASS: TestSkillFrontmatter_UnquotedValuesHaveNoColonSpaceOrSpaceHash (0.01s)
--- PASS: TestDirectorMarkStoppedIfUnchanged_RefusesAnIDRecordedAfterTheRead (0.01s)
--- PASS: TestStopDirector_RefusesWhenAStartRecordsMoreWhileItRuns (0.01s)
--- PASS: TestStopDirector_AForcedStopRecordsWhatStartRecordedMeanwhile (0.00s)
--- PASS: TestOrgDirectorStop_AStartThatRecordedMeanwhileNamesTheStopAgain (0.00s)
```

`TestSkillFrontmatter_UnquotedValuesHaveNoColonSpaceOrSpaceHash` は `checked the frontmatter of 54 SKILL.md files under .claude/skills, .agents/skills, templates/base/.claude/skills, templates/base/.agents/skills` を log に出す。4 つの根のどれも読んでいる。D14 のテストは、`_test.go` で `ErrDirectorChangedDuringStop` を grep して見つけた(`internal/org/director_test.go`、`internal/org/director_stop_test.go` の helper `wantChangedDuringStop`、`internal/cli/org_director_headless_test.go`)。

### cycle 4 の変更とテストの対応

「mutant」の列は、その変更の判定を壊した mutant が落ちたかどうか(id は下の Mutation の表と evidence の C4-4)。

| 変更 | テスト | mutant |
| --- | --- | --- |
| D14: 強制でない stop の記録(`MarkStoppedIfUnchanged`)は、lock の下で workspace と pane の id を stop が読んだときと比べ、違えば `ErrDirectorChangedDuringStop` を返して何も書かない | `TestDirectorMarkStoppedIfUnchanged_RefusesAnIDRecordedAfterTheRead`(5 行)、`TestStopDirector_RefusesWhenAStartRecordsMoreWhileItRuns`(3 行) | G1(外す)、G2(workspace だけ比べる)、G3(pane だけ)、G4(state も比べる)、G5(読んだ値を渡さない)、G6(`MarkStopped` で書く)、G9(sentinel を包まない)、G10(2 つの読みを逆に書く)、G11(no workspace と no pane の語を入れ替える)red |
| D14: 強制の stop(`MarkForceStopped`)と start の後始末(abort の `MarkStopped`)は比べない | `TestStopDirector_AForcedStopRecordsWhatStartRecordedMeanwhile`、`TestStartDirector_ARecordThatFailsStillClosesWhatHerdrMade` | G7(強制でも比べる)、G8(abort が `r.reg` と比べる)red |
| D14 の拒否の文(「this stop closed only what it had read」と、打ち直す stop) | `RefusesWhenAStartRecordsMoreWhileItRuns`、`TestOrgDirectorStop_AStartThatRecordedMeanwhileNamesTheStopAgain`(2 行) | P7、P8 red |
| AC8 の最後の文(S12): 記録のときに登録がすでに stopped(`ErrDirectorStopped`)なら stop を成功にし、`AlreadyStopped` を立てる。CLI は打つ stop を示さない | `TestStopDirector_SucceedsWhenTheStopWasRecordedFirst`(4 行)、`TestOrgDirectorStop_AStopRecordedFirstNamesNoStopToRun`(2 行) | H1〜H6、C1、C2 red |
| 同じ文の start の側: start の後始末の `MarkStopped` が `ErrDirectorStopped` を受けたら記録済みとする | この回で足した `TestStartDirector_AStopRecordedWhileTheAgentStartsCountsAsTheCleanupsRecord` | H7 red。905fc0a7 のテストだけでは、org と cli の全体でも生き残った(H7org、H7h) |
| `directorRecordStopError`: `org: director: ` と `stop <id>: ` を 1 回にし、`Unwrap` で中の error を返す | `TestStopDirector_ARecordThatFailsSaysItsPrefixOnce`、`TestStartDirector_ACleanupWhoseRecordFailsSaysThePrefixOnce`、`TestStopDirector_AForcedStopThatCannotRecordSaysToRetryWithForce` など | P1〜P6 red |
| 書き戻せない記録(`ErrDirectorWriteNotRestored`)のときの `DirectorCleanupError.Error` の文 | この回で足した `TestStopDirector_ARecordThatCannotBeCutBackSaysTheFileMayHoldPartOfIt`(2 行) | P9、P11 red。905fc0a7 のテストだけでは全体でも生き残った(P9h、P11h) |
| writer の行に `MarkStoppedIfUnchanged` と `MarkForceStopped` を足す | `TestDirectorHeadlessWriters_ACutOffWriteIsRestored` の 2 行 | この回は mutant を作っていない(行が走って PASS することは上の PASS の行で見た) |
| `ralph org director stop` の help の D14 と AC8 の文 | `AStartThatRecordedMeanwhileNamesTheStopAgain`、`AStopRecordedFirstNamesNoStopToRun` | C3、C4 red |
| D15: `/director` の description を YAML として読める形にし、plain な値の `: `、`:` で終わる値、` #` をテストで見つける | `TestSkillFrontmatter_UnquotedValuesHaveNoColonSpaceOrSpaceHash`(54 本)、`TestPlainFrontmatterFindings`(19 行)。Psych で 54 本 ok | F2、F4、F5、F6、F8、F9、F10 red。F1、F3、F7 は生き残り(下)。壊した写しの Y1、Y2、Y3、Y5 は FAIL |

### Coverage

- Statement(go1.25.8、10efa82c、各パッケージ自身のテストの profile): `internal/org` 94.7%、`internal/cli` 87.8%、`internal/config` 93.3%、`internal/org/driver` 93.1%、`internal/org/protocol` 100.0%、`internal/insights` 86.1%、`internal/scaffold` 75.7%、`internal/upgrade` 91.2%(cycle 3 は org 94.6%、cli 87.7%、ほかは同じ)
- 変わった関数はすべて通る: `markStopped` の D14 の比較(`director_start.go` 354〜357)、`directorRecordedIDs`(380〜389)、`recordDirectorStop`(`director_stop.go` 198〜230)、`directorRecordStopError`(247〜253)、`DirectorCleanupError.Error` の変わった case(335〜336)、`printOrgDirectorStop` の AlreadyStopped の分岐(`org_director_headless.go` 400〜402)
- `internal/org/director_stop.go` は 146/149 文(98.0%)。905fc0a7 では 145/149 で、`DirectorCleanupError.Error` の書き戻せない場合(319〜320)が通らなかった。この回で足した 1 本目のテストがそこを通す。残る 334 は新しい switch の空の `case e.StopCommand == "":` で、`newDirectorCleanupError` は StopCommand を必ず入れるので、手で作った error しか届かない(switch の前と同じ)
- `internal/org/director_start.go` は 226/241 文(93.8%)。新しく通らないのは、`markStopped` の lock の下の読みの失敗(346〜348)と、abort の detach の失敗(734〜737)で、どちらも S1b から。残りは cycle 3 と同じ
- `internal/org/director.go` は 193/194 文(99.5%。通らないのは `json.Marshal` の失敗の 481〜483)。`internal/cli/org_director_headless.go` は 105/108 文(97.2%)で、通らない 3 文は cycle 3 と同じ(設定と model の解決の失敗、`os.Getwd` の失敗)
- Branch: Go の標準ツールには branch coverage がないので測っていない。代わりに mutation で分岐ごとの判定を確かめた(下)

### Mutation

- 対象は D14、AC8 の最後の文、接頭辞の直し、frontmatter のテストの helper。ファイルを overlay で差し替え、`-run` で流した(org、cli は cycle 3 と同じ `-run`、config は `SkillFrontmatter|PlainFrontmatter`)。生き残りはパッケージ全体でも流した。手を加えないファイルを overlay で読ませた対照の X0(`director_start.go`)、X0s(`director_stop.go`)、X0c(`org_director_headless.go`)、X0f(`skill_frontmatter_test.go`)と、足したテストだけを外した X0t、X0t2 は通った。どの差し替えも元の文字列がちょうど 1 か所にあり、すべての mutant が build できた(F4 と F6 は 1 回目の形で変数と import が使われずに build できなかったので、使う形に直した)
- batch は 04:40〜04:49 に 905fc0a7 に対して流した。1 本目のテストはその時点で worktree にあり、2 本目(04:50:07)はまだなかった。そのため P1〜P4 と P6 の「落としたテスト」に 1 本目のテストも並ぶが、どれも 905fc0a7 からあるテストでも落ちている。H7 は 2 本目を書いたあとに流した。「h」の付く回は、足したテストを外して流し直したもの
- 42 件のうち 39 件が red、3 件(F1、F3、F7)が生き残った。905fc0a7 のテストだけでは、さらに P9、P11、H7 が両パッケージの全体でも生き残った。この 3 件は、足した 2 本のテストが閉じた穴

| ID | 変異 | 結果 | 落としたテスト |
| --- | --- | --- | --- |
| G1 | D14 の比較を外す | red | `TestDirectorMarkStoppedIfUnchanged_RefusesAnIDRecordedAfterTheRead`、`TestStopDirector_RefusesWhenAStartRecordsMoreWhileItRuns` |
| G2 | D14 が workspace だけ比べる | red | 同上 |
| G3 | D14 が pane だけ比べる | red | `RefusesAnIDRecordedAfterTheRead` |
| G4 | D14 が state も比べる(launched で断る) | red | 同上 |
| G5 | `MarkStoppedIfUnchanged` が読んだ値を渡さない(比べない) | red | G1 と同じ 2 本 |
| G6 | stop の強制でない記録が `MarkStopped` を呼ぶ | red | `RefusesWhenAStartRecordsMoreWhileItRuns` |
| G7 | 強制の記録も id を比べる | red | `TestStopDirector_AForcedStopRecordsWhatStartRecordedMeanwhile` |
| G8 | start の後始末(abort)が `r.reg` と比べて記録する | red | `TestStartDirector_ARecordThatFailsStillClosesWhatHerdrMade` |
| G9 | D14 の拒否が `ErrDirectorChangedDuringStop` を `%w` で包まない | red | G1 と同じ 2 本 |
| G10 | D14 の拒否が 2 つの読みを逆に書く | red | 同上 |
| G11 | `directorRecordedIDs` の no workspace と no pane の語を入れ替える | red | 同上 |
| H1 | AlreadyStopped の分岐を外す(stopped の拒否がまた cleanup error になる) | red | `TestStopDirector_SucceedsWhenTheStopWasRecordedFirst` |
| H2 | AlreadyStopped を強制でないときだけにする | red | 同上 |
| H3 | AlreadyStopped を強制のときだけにする | red | 同上 |
| H4 | AlreadyStopped を立てない(nil の error と素の結果を返す) | red | 同上 |
| H5 | 強制の stop で AlreadyStopped と一緒に Forced も立てる | red | 同上 |
| H6 | AlreadyStopped と一緒に WasCurrent も立てる | red | 同上 |
| H7 | abort が、先に記録された stop(`ErrDirectorStopped`)を記録の失敗とする | red(足したテストで) | `TestStartDirector_AStopRecordedWhileTheAgentStartsCountsAsTheCleanupsRecord`。H7org と H7h は生き残り |
| P1 | `directorRecordStopError` が何も削らない | red | `TestStopDirector_ARecordThatFailsSaysItsPrefixOnce`、`TestStartDirector_ACleanupWhoseRecordFailsSaysThePrefixOnce`、`AForcedStopThatCannotRecordSaysToRetryWithForce`、`RefusesWhenAStartRecordsMoreWhileItRuns` ほか |
| P2 | 同じく `org: director: ` だけを削る | red | 同上 |
| P3 | `directorRecordStopError.Unwrap` が nil を返す | red | 同上 |
| P4 | start の後始末を `fmt.Errorf("record %s stopped: %w")` に戻す | red | `ACleanupWhoseRecordFailsSaysThePrefixOnce` ほか |
| P5 | stop の強制の分岐を同じ形に戻す | red | `AForcedStopThatCannotRecordSaysToRetryWithForce` |
| P6 | stop の一般の分岐を同じ形に戻す | red | `ARecordThatFailsSaysItsPrefixOnce` ほか |
| P7 | D14 の分岐から「; this stop closed only what it had read」を外す | red | `RefusesWhenAStartRecordsMoreWhileItRuns` |
| P8 | `DirectorCleanupError.Error` から D14 の case を外す(「record the stop with」と言う) | red | `AStartThatRecordedMeanwhileNamesTheStopAgain`、`RefusesWhenAStartRecordsMoreWhileItRuns` |
| P9 | `DirectorCleanupError.Error` から書き戻せない場合の case を外す(「nothing was recorded」と言う) | red(足したテストで) | `TestStopDirector_ARecordThatCannotBeCutBackSaysTheFileMayHoldPartOfIt`。P9h は生き残り |
| P11 | 書き戻せない場合より先に `e.Err` を見る(start の後始末が「nothing more was recorded」と言う) | red(足したテストで) | 同上。P11h は生き残り |
| C1 | `printOrgDirectorStop` が AlreadyStopped を見ない | red | `TestOrgDirectorStop_AStopRecordedFirstNamesNoStopToRun` |
| C2 | 同じく AlreadyStopped の 2 行目を出さない | red | 同上 |
| C3 | stop の help から D14 の文を外す | red | `AStartThatRecordedMeanwhileNamesTheStopAgain` |
| C4 | stop の help から AC8 の文を外す | red | `AStopRecordedFirstNamesNoStopToRun` |
| F1 | frontmatter の検査がいつも skip する | 生き残り(全体でも) | - |
| F2 | 二重引用符の値も検査する | red | `TestPlainFrontmatterFindings` |
| F3 | `>` か `\|` で始まる値も検査する | 生き残り(全体でも) | - |
| F4 | CR を削らない | red | `TestPlainFrontmatterFindings` |
| F5 | `:` で終わる値の検査を外す | red | 同上 |
| F6 | 桁を byte で数える | red | 同上 |
| F7 | 1 行に 2 つの形があるとき、一覧の後ろの形を選ぶ | 生き残り(全体でも) | - |
| F8 | `: ` の検査を外す | red | `TestPlainFrontmatterFindings` |
| F9 | ` #` の桁を空白に置く | red | 同上 |
| F10 | 字下げした行も検査する | red | 同上 |

- F1: `.claude/skills` のない repo(vendored の repo)で skip するのは設計どおりで、skip しても何も落ちない。走ったことは log の行(54 本)で分かる
- F3: `>` や `|` のあとの同じ行に `: ` や ` #` がある値(`a: > # note` は正しい YAML)は、行にも SKILL.md にもない。違いが出るのはそこだけで、向きは誤検出のほう
- F7: 「the first of two shapes on a line」の行(`x #y: z`)は、行の中で先に来る形が一覧では後ろにあるので、どちらでも同じ形を選ぶ。違うのは失敗の文に出る規則の名前だけで、行はどちらでも落ちる(`x: y #z` の行なら見分けられる)
- P9h、P11h: 905fc0a7 には、書き戻せない記録の stop や start の後始末を試すテストがなかった
- H7h: AC4 の行は start を attach --replace か detach --force で終わらせ、どちらも登録を開いたままにするので、start の `MarkStopped` は成功する。先に stop が記録する行はなかった

### この回で足したテスト(10efa82c)

- `TestStopDirector_ARecordThatCannotBeCutBackSaysTheFileMayHoldPartOfIt`(`internal/org/director_stop_test.go`、subtest 2 本)。動いている d1 の stop と、agent の start が失敗したあとの start の後始末で、記録の書き込みを途中で止め(`directorPartialWrite`)、切り戻しも失敗させる。error は `*DirectorCleanupError` で、`ErrDirectorWriteNotRestored`、ENOSPC、truncate の error を包み、接頭辞は 1 回、`; director.jsonl may hold part of the write` と言い、何も(それ以上)記録していないとは言わず、stop の打ち方を示して herdr の打ち方は示さない。ファイルには detached が丸ごとと stopped の一部(壊れた行 1 本)が残り、current はなく d1 は開いたまま。そのあと示された stop を、書き込みを直し、herdr が pane も workspace も知らない状態で打つと、stopped を記録して開いた登録は残らない。P9 と P11 を落とす
- `TestStartDirector_AStopRecordedWhileTheAgentStartsCountsAsTheCleanupsRecord`(`internal/org/director_start_test.go`)。fake の agent の start の中で stop を走らせ、ws-1 と pane-1 を閉じ、detached と stopped を記録させる。start の `MarkLaunched` は current の登録を見つけられず、後始末は herdr がもう作ったものを知らないと分かり、`MarkStopped` は `ErrDirectorStopped` を受ける。start は `ErrDirectorReplacedDuringStart` を返し、文は `; start closed what it had made in herdr, and d1 is recorded stopped` で終わる。`*DirectorCleanupError` ではない。agent の start のあとに C-c も close も送らず、記録は attached、workspace_created、pane_created、detached、stopped。H7 を落とす
- 2 本とも、overlay で外した対照(X0t、X0t2)で通ることと、狙った mutant だけで落ちることを確かめた。Go のすべての実行で PASS

### Failure analysis

| Test | Error | Root cause | Proposed fix |
| --- | --- | --- | --- |
| なし | - | - | - |

落ちたテストはない。

### Regression checks

| Previously broken behavior | Status | Evidence |
| --- | --- | --- |
| cross-review cycle 3 の指摘(D14): stop が読んだあとで start が記録した pane や workspace を、stop は閉じないまま stopped と記録する | 直っている。lock の下で比べ、違えば何も書かずに断る | G1〜G11 red |
| cross-review cycle 3 の指摘(D15): `/director` の description が YAML として読めない | 直っている | Psych で 54 本 ok、直す前の 4 本は SyntaxError。壊した写しの Y1、Y2 は FAIL |
| verify cycle 4 の指摘(S12): start の後始末が先に stopped を記録した d1 の stop が、成功にならず cleanup error で終わる | 直っている | H1〜H6、C1、C2 red |
| 記録の失敗の文で `org: director: ` が 2 回出る | 直っている | P1〜P6 red |
| cycle 1〜3 のテストと既存のテスト | すべて PASS。名前の並びは cycle 3 から 51 行増え、消えた行はない | evidence の C4-2.2 |
| 前に揺れた doctor、watcher、gate のテスト | 落ちていない | evidence の C4-2.1 |

### Test gaps

1. F1: frontmatter のテストは `.claude/skills` がないと skip し、skip しても何も落ちない。vendored の repo のための設計で、走ったかどうかは log の行で分かる
2. F3: `>` か `|` のあとの同じ行に `: ` や ` #` を持つ値の行がない。違いが出るのは誤検出の向きだけ
3. F7: 1 行に 2 つの形があるとき、失敗の文がどちらの規則を名指すかを固定していない。`x: y #z` の行を足せば固定できる
4. D14 と AC8 の競合は、fake の herdr を使う org のテストで見ただけで、build した binary を CLI から打つ形では走らせていない。CLI のテストが使う stub の herdr は、close の途中で台帳に書けない
5. 本物の herdr で、start の agent の start の途中に stop が pane と workspace を閉じたときの順と結果は見ていない
6. 書き戻せない記録の文は「may hold part of the write」を 2 回言う(store の文と `DirectorCleanupError.Error` の注記)。S1b からある形で、`directorRecordStopError` は接頭辞の重なりだけを削った。足したテストは文を部分ごとに確かめ、重なった句は固定していない(evidence の C4-4.5)
7. writer の行の `MarkStoppedIfUnchanged` と `MarkForceStopped` には、この回は mutant を作っていない
8. cycle 3 の Test gaps 1〜6 は、cycle 4 の変更の外で、そのまま残る

### Verdict

- Pass: yes。`./scripts/run-test.sh`(changed、full に広がった)は 10efa82c で rc 0(shell 41 本すべて OK、Go 8 パッケージすべて ok)。go1.25.8 の `go test ./... -count=1` は 10efa82c で rc 0(cli 708/662、org 812/1,712 と skip 1、config 39/53、insights 36/7、org/driver 48/89、org/protocol 15/56、scaffold 32/0 と skip 2、upgrade 114/13。fail はどれも 0)。org と cli の `-race` は go1.26.0 と go1.25.8 のどちらでも rc 0 で DATA RACE もなく、go1.26.0 の残り 6 パッケージも rc 0。テストの名前は両方の版で同じ。cycle 3 のあとで足したテスト 12 本(subtest 39)はどれも PASS。テストは本物の herdr と osascript を打たず(canary は 0 回)、repo の台帳に書いていない。`~/.codex/config.toml` は変わっていない
- Fail: なし
- Blocked: なし
- Verified: D14(強制でない stop は lock の下で workspace と pane の id を比べ、違えば何も書かずに断る。強制の stop と start の後始末は比べない)、AC8 の最後の文(stop の側の AlreadyStopped と、start の後始末の側の記録済みの扱い)、接頭辞を 1 回にする直し、D15(`/director` の description、54 本の SKILL.md を Psych で読めること、frontmatter のテストの 4 つの根と規則)を、テストと mutant 42 件(red 39 件)で確かめた。足した 2 本のテストで、905fc0a7 のテストでは生き残った P9、P11、H7 を落とした
- Not verified: Test gaps の 1〜8。どれも PR を止めるものではない
