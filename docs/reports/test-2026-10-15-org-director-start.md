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
