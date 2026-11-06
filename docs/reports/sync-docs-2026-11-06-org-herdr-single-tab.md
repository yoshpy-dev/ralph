# sync-docs report: org-herdr-single-tab

- Date: 2026-11-06
- Plan: `docs/plans/active/2026-11-06-org-herdr-single-tab.md`(AC1〜AC12、D1〜D8、R1〜R7、S1〜S7)。spec は `docs/specs/2026-11-06-org-herdr-autostart-panes.md` の B
- Branch: `feat/org-herdr-single-tab`。run 開始時の HEAD は `d4fa6ce2`、base は main `920832dd`
- Pipeline cycle: 1
- 先行 report: self-review `97526693`(LOW 9 件の L1〜L9)、verify `feb2e943`(pass。LOW 2 件の V1・V2、INFO 4 件の I1〜I4)、test `d4fa6ce2`(pass)。この run への依頼の中身は verify の I1・I2
- 制約: Go のファイルは変えていない(`paneenv.go` の doc、`spawn.go` の `HerdrClient` の doc、`putSeatPaneInHerdr` の doc は cycle 2 の直しに回す)。plan と spec の箱、plan の Progress checklist は触っていない。plan の digest は `61ba0f08bb4c` のまま

## Summary

verify の I1 の 4 つ(`director-launch-cmd.md`、`director-wake.md`、`git-commit-strategy.md`、tech-debt の 363 行)を、root と `templates/base/` の写しの両方で直した。tech-debt は I2 の 7 行に注を足し、178 行の (c) を RESOLVED にした。自分の sweep でさらに 9 行が当たり、同じ形の注を足した。ファイルの大きさの行は、`herdr.go` と `spawn_layout_test.go` を 1 行にまとめて末尾に足した。

## Changes made

| File | Change |
|------|--------|
| `docs/recipes/director-launch-cmd.md`、`templates/base/docs/recipes/director-launch-cmd.md` | herdr の director の pane の値は `herdr workspace create --env AGMSG_SELF_NAME=off --env RALPH_WIP_CHECKPOINT=off`(新しい workspace のルートの pane)で渡す、確かめるのは `herdr workspace create --help`、警告は `warning: herdr workspace create has no --env flag` で始まる、に直した(263〜269 行)。2 つは `cmp` で同一 |
| `docs/recipes/director-wake.md`、`templates/base/docs/recipes/director-wake.md` | 見張りが director の pane を確かめる条件を、pane の label `director_<token>`(workspace の label `ralph_director_<token>`)に直し、前の版の director は pane の label が無いので tab の label `director_<token>` で確かめる、と足した(139〜143 行)。2 つは `cmp` で同一 |
| `.claude/rules/ralph/git-commit-strategy.md`、`templates/base/.claude/rules/ralph/git-commit-strategy.md` | `--env` の確かめは pane を作るサブコマンドごと(`workspace create`、`pane split`、`tab create`)で、`--env` が無いサブコマンドで作る pane にだけ値が付かず、警告がサブコマンドの名を言う、に直した(57〜60 行)。2 つは `cmp` で同一 |
| `docs/tech-debt/README.md` | 既存 16 行の編集と、末尾の 1 行(435 行)。下の「tech-debt」 |
| `docs/reports/sync-docs-2026-11-06-org-herdr-single-tab.md` | この report |
| `docs/insights/events/2026-11-06-org-herdr-single-tab.jsonl` | `sync_docs` の行(cycle 1、pass)を `insights-append.sh --slug org-herdr-single-tab --cycle auto` で足した。`git status --short` は既存の jsonl を `M` と出しただけで、新しいファイルはできていない |

3 つの文書の直しは、コードの文と突き合わせた。警告の文は `warnNoPaneEnv`(`internal/org/driver/herdr.go`)の `"herdr " + sub + " has no --env flag"`、確かめを持つサブコマンドは `paneEnvProbes` の 3 つ、director の pane の label と workspace の label は `DirectorTabLabel`・`DirectorWorkspaceLabel`、前の版の director の扱いは `confirmPaneTab` である。

## 依頼の項目

| 項 | 結果 |
|----|------|
| 1. `director-launch-cmd.md` | 直した |
| 2. `director-wake.md` | 直した |
| 3. `git-commit-strategy.md` | 直した |
| 4. tech-debt(363 行、発火した Trigger、178 行の (c)、大きさの行) | 直した。下の「tech-debt」 |
| 5. Go を変えない | 守った。`git diff --stat` は md と jsonl だけ |
| 6. plan と spec の箱 | 触っていない |

## tech-debt

注の形は 2026-11-05 の org-role-models、2026-11-06 の org-herdr-autostart と同じ(何が当たったか、払わなかった理由、「Carried over to the next change that meets this trigger」)。Related には `docs/plans/active/2026-11-06-org-herdr-single-tab.md` を足した(`/pr` の `archive-plan.sh` が `archive/` に書き換える)。行数は base `920832dd` と HEAD `d4fa6ce2` で `wc -l` を取った。

### verify が挙げた 7 行

| 行 | 変更 |
|----|------|
| 178(ownership check) | (c)「`resolveWorkspace` が label を確かめずに再利用する」を RESOLVED にした。Debt・Impact・Trigger の (c) に `~~` を付け、Debt に S4 9df9735c の直し(`WorkspaceGet` で label を読み、herdr に無い、または label が org_id でなければ `org_workspace_closed` を書いて次へ、ほかの get の失敗は `workspace_get` で spawn を失敗にする)と、固定するテスト `TestOrgSpawn_Layout_RecordedWorkspaceNotUsable`、実機の記録の 3 を書いた。直し方は行の案(`confirmOrgWorkspace` を呼ぶ)と違い、`resolveWorkspace` が自分で label を読む。そのことを Debt と Trigger に書いた。(a) は Trigger の末尾に注を足した: `confirmSeatPane` が `confirmPane` に委譲し、`%v` で `%w` の鎖を切る行は `confirmPaneTab` に移り、今も必要で、`TestOrgStop_PaneNotConfirmed_NoCtrlCNoClose` の「tab gone although the pane was found」が固定している。`NotFound()` と `IsNotFound` は変えていない。(d) が名指す最初の `isNotFound` の case は `confirmPane` の最初の case になった |
| 183(test gaps) | (a) と (c) に当たった(`Stop` の doc、`PaneGet` の parser、`fakeWatchHerdr`)。払っていない: 台帳が読めない場合も `stop_failed` の append の失敗も足されず、`PaneGet` の label のテストに JSON でない返事の case は無い。Debt を 3 か所直した。(a)の `org_workspace_created` の append は `placeSeatPane` に移り、`pane_created` の spawn_step の append が加わったこと。(b) の sh の stub は `workspace create` と `pane split` で作った pane については閉じると忘れるようになったこと(`tab create` の pane と `workspace close` は今も記録だけ)。(c) の `fakeWatchHerdr` は `WorkspaceGet` だけ記録した label を返すこと。Trigger の注に (c) の直し方の新しい形(`PaneGet` が `Label` に座席 id を持つ `HerdrPane` を返す)を書いた |
| 276(codex の director の 2 つの前提) | 「`StartDirector` の herdr の呼び出し」に当たった。`TabCreate` を呼ばず、`WorkspaceCreate` のルートの pane に director を置き、`PaneRename` と `TabRename` で label を付ける。`directorAgentArgs` は変えていない。実機の記録は `director start` を打っていないので、2 つの前提はそのまま、(a) は open のまま |
| 319(`director_start.go` の大きさ) | 1260 → 1307 行で当たった。同じ行の他のファイルは `herdr_test.go` が 1693 → 2388(+695)、`director_check_test.go` が 857 → 874、`director_start_cleanup_test.go` が 804 → 808、`director_stop.go` が 769 → 784(800 未満)。move-only の分割は Scope の外 |
| 363(spawn の失敗の道) | Debt の `tab_created` を `pane_created` の spawn_step に、「空の tab」を「label の無い pane(新しい workspace のルート、座席の pane の分割、`tab create` の tab)」に直した。Impact は、`pane_created` の append が失敗した pane は人が閉じるか、`disband` が org の workspace と一緒に閉じる、とした。Trigger の `tab_created` も直し、注を足した: `placeSeatPane` の pane を作る前の失敗は `spawn_failed` で閉じる pane が無く、`pane_rename` の失敗は `closeSpawnPane` で閉じる。払っていない: `pane_created` の spawn_step とその後の記録の append が失敗したときに pane を閉じない道は、行が書くとおり残る |
| 364(`spawn.go` と `spawn_test.go` の大きさ、補償) | `placeSeatPane` の失敗の道と `compensateStale` の変更で当たった。`spawn.go` 2778 → 3059、`spawn_test.go` 6043 → 6406。新しいテストは `spawn_layout_test.go` と `confirm_pane_test.go` に入り、seat-spawn-close の 3 つの節は `spawn_test.go` に残る。分割は Scope の外 |
| 365(5 つのファイルの大きさ) | 行数で当たった。`spawn.go` は 1 列目の 2680 + 300 = 2980 を、`spawn_test.go` は 5991 + 300 = 6291 を超えた。`verbs.go` は 2290 → 2343、`verbs_test.go` は 5854 → 5883 で、+300 に届かない。今後の +300 は 3059 と 6406 から数える、と書いた |

### 自分の sweep で足した 9 行

全 open 行の Trigger(取り消し線の外)を python で引き、この diff が変えたファイル名、関数名、skill、recipe と突き合わせた。当たりは 1 行ずつ、Trigger が名指す関数の hunk を読んで決めた。コメントだけの変更も当たりとした(org-role-models が 179・361 行に、`Stop` の comment の編集で注を足した前例に合わせた)。

| 行 | 当たったもの | 払わなかった理由 |
|----|--------------|------------------|
| 179(`Stop` ほかの code shape) | `Stop` の doc(S3)と、F-7「driver API を次に触るとき」(`PaneSplit` ほかの追加。新しい fake も `driver.NewHerdrError` を使う) | `Stop` の最後の `result.Err = err` も 5 つの自分の pane の確かめも、`NewHerdrError` の export も変えていない |
| 180(`/org` の command table) | `spawn` 2,802 → 2,998、`stop` 1,060 → 1,107、`disband` 1,217 → 1,219、`director` 6,347 → 6,377 文字(`.claude/skills/org/SKILL.md` の表の行全体で数えた) | (a) が名指す `stop` と `disband` のセルを今回は変えたが、短くしていない。表の組み直しは Scope の外 |
| 191(`Spawn` の doc) | `Spawn` の doc(配置のロックと pane の記録)と本文(`placeSeatPane` の 1 呼び出し)。`spawn.go` 2778 → 3059 | `idempotentRespawn` と `orgWideLimitsHelp` は変えていない。(b)〜(e) は同じ |
| 238(gate と lockfile の small findings) | `lockfile.go` の `layoutLockName` と `heldFileLock` の doc の 1 句(S4) | (c) と (g) が名指す doc には触れていない。既存のコードは変えていない |
| 303(`newTestWatchRun` の MkdirTemp) | `watch_test.go` の `fakeWatchHerdr`(3005 → 3051 行) | `newTestWatchRun` は変えていない |
| 361(`Stop` が spawn ロックを取らない) | `Stop` の doc と、spawn が 2 つ目のロック(配置のロック)を取るようになったこと | `Stop` はどちらのロックも取らない。plan の R3 が、stop と spawn の競りを受け入れている |
| 371(recipe 5 つの evidence の引用) | `isolated-herdr-server.md`(S6)と、この sync-docs の `director-launch-cmd.md`、`director-wake.md`。どれも 2 つの写し | `docs/evidence/` と tech-debt の引用は足しても直してもいない。数は `isolated-herdr-server.md` が 13 件・6 ファイル、`director-launch-cmd.md` が 1 件・1 ファイル、`director-wake.md` が 3 件・2 ファイルのまま(`grep -o` で数えた) |
| 395(`resolvedOrClean` の呼び出し側) | `feature_test.go`(S4。fake の `paneID` → `nextPaneID`、`tabCreateErr` → `workspaceCreateErr`) | `resolvedOrClean` の呼び出しは 1 つも変えていない。org-role-models が同じファイルで同じ判断をした |
| 396(固定の期限のテスト 2 つ) | `director_wake_test.go`(S5。pane の label で確かめる呼び出しの列、1630 → 1635 行) | 名指された `TestWakeDirector_AFailedWakeIsNotTriedAgainForFiveMinutes` と `director_cmd_test.go` の 20 秒のテストは変えていない |

### 退けたもの

| 行 | 退けた理由 |
|----|------------|
| 137(`fakeHerdr` の遅延差し込み) | 足したのは `closeRemoves` と pane の模擬で、遅延ではない。`splitEnteredHerdr` はテストの包みで、`fakeHerdr` の hook ではない(verify も同じ) |
| 184・186・193・194(`CloseDeferredSelfPane` ほか) | 名指された関数のコードは変えていない。変わったのは `confirmSeatPane` の中身で、呼び出しの形は同じ |
| 200 の (d)(`HerdrClient` の comment) | 型自身の comment(77〜87 行、91 桁の行は 91 行目)は変えていない。変わったのは interface の中のメソッドの doc |
| 225(approval の helper) | `approval_enforce_test.go` の変更は、テスト本体の fake の field 名(`paneID` → `nextPaneID`)で、approval の helper には触れていない |
| 268(director stop の確かめ) | director stop は今も `confirmSeatPane` と `confirmOrgWorkspace` を使う。自前の確かめには替えていない |
| 277・278(director stop の error) | `DirectorCleanupError` の型の doc の 1 句を替えたが、Trigger が名指す `Error`、`directorRecordStopError`、`restoreLocked`、`recordDirectorStop`、`workspace close` の呼び出しは変えていない |
| 262・379・380(`/director` skill) | 直した節は「立てる」の手順と stop の 3 で、受信箱の節にも、送って確かめる手順にも、wait の終了コードにも触れていない |
| 275(`startDirector` の拒否の順) | CLI の `startDirector` は出力の 1 行を替えただけ |
| 288・381・390(b)・362 | 冪等な枝の `idempotentRespawn`、`reject`、`StartFeature`、`manifest.go` と `status.go` は変えていない |
| 318・431・432(herdr のサーバーのゲート) | `HerdrServerGate` を呼ぶ場所も、`checkDirectorStart` の検査の順も変えていない |
| 201 の (h)(`Spawn` が pane を始める方法) | 取り消し線の中(moot) |
| 177・195・219・244 | 「report が出たとき」や、director の段階への言及で、この diff が触れるコードを名指さない |

### 大きさの行(新しい 1 行)

`docs/tech-debt/README.md` の 435 行(末尾)。`internal/org/driver/herdr.go`(732 → 963 行)と `internal/org/spawn_layout_test.go`(新規、812 行)を 1 行にした。慣例は、1 つの PR で 800 行を超えたファイルを、その PR の名前で 1 行にまとめる形(404・416・420 行)である。`herdr_test.go`(800 超えのまま 1693 → 2388)は 319 行の対象なので、新しい行には名前だけ書いた。移す先は案: `herdr.go` の `--env` の確かめを `herdr_env.go` に、pane の操作を `herdr_pane.go` に。`spawn_layout_test.go` の配置のロックのテストを `spawn_layout_lock_test.go` に。

## 確かめたこと

- `./scripts/check-sync.sh`: DRIFTED 0、ROOT_ONLY 0、TEMPLATE_ONLY 10、KNOWN_DIFF 6、PASS
- `./scripts/check-skill-sync.sh`: 14 skill が一致
- `cmp`: recipe 2 つとルールの root と template の写しが、それぞれ同一(編集の前も後も)
- tech-debt の plan の参照(`verify.local.sh` の `check_tech_debt_plan_refs` と同じ grep を手で打った): 存在しない参照は 0 件。足した参照は `docs/plans/active/2026-11-06-org-herdr-single-tab.md` で、存在する
- tech-debt の編集を `git show HEAD:` と突き合わせた(python)。変わった行は 178・179・180・183・191・238・276・303・319・361・363・364・365・371・395・396 の 16 行と、末尾の 1 行だけ。どれも 5 セル、バッククォートと `~~` はセルごとに偶数、新しい `file:line` の引用は 0。178・183・363 以外は、古いセルの全文が新しいセルに含まれる(末尾への追記だけ)
- 行数は `wc -l` で取った(`spawn.go`、`spawn_test.go`、`verbs.go`、`verbs_test.go`、`director_start.go`、`herdr.go`、`herdr_test.go`、`spawn_layout_test.go`、`watch_test.go`、`director_wake_test.go`、`director_stop.go`、`director_check_test.go`、`director_start_cleanup_test.go`)。skill の表の行の長さは python で数えた
- `go test` は打っていない(Go のファイルを変えていない)

## 残っていること(orchestrator へ)

- plan の AC1〜AC12、spec の FR-B1〜B8・AC-B1〜B9・非機能要件の箱、plan の Progress の Review・Verification・Test の箱は `[ ]` のまま。spec の箱を付けるときは、verify の V1(非機能要件 2 を、2 席目からの spawn は字面では満たさない)と AC5 の L2 を項目の下に書く
- self-review の L4(`placeSeatPane` が長い、`spawn.go` が 3059 行)は 364・365 行の注が受ける。L5(最初の座席で `WorkspaceCreate` から spawn_step までの窓)と L7(`DirectorTabLabel` ほかの名前が pane の label を指す)は register に行が無い。self-review は L4・L5・L7 を 1 行にまとめる案を出したが、cycle 2 の直しで L5 や L7 を直すかどうかが決まる前なので、この run では足していない。直さずに `/pr` へ進むなら、足す
- コメントだけの直しは cycle 2 に回してある: `paneenv.go` の 2 つの doc(V2)、`spawn.go` の `HerdrClient.WorkspaceCreate` の doc(L1)、`putSeatPaneInHerdr` の doc(I4)。L2(座席 id が org_id と同じ座席)と L3(`--env` の無い herdr で agmsg が pane の label を書き換える)は、直すなら Go の変更になる
- 178 行の (c) は RESOLVED にしたが、行の本文に残る (a)・(d) は open のまま
