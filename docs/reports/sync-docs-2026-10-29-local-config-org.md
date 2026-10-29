# sync-docs report: local-config-org

- Date: 2026-10-30(JST。計画の日付は 2026-10-29)
- Plan: `docs/plans/active/2026-10-29-local-config-org.md`(issue #276 の 1 本目 A1。承認の digest `1609f175543b` はこの run の前後で同じ。`./scripts/plan-visual.sh digest` で確かめた。この run が変えたのは AC1〜AC13 の checkbox と `## Progress checklist` の 1 行だけ)
- Pipeline cycle: 1(`.harness/state/standard-pipeline/cycle-count.json` の値)。base は `origin/main` の `5eba4013`、branch は `feat/local-config-org`。run 開始時の HEAD は `c9e83864`(この report を含む commit の SHA ではない)
- 先行 report: `docs/reports/self-review-2026-10-29-local-config-org.md`(Merge: yes。L1〜L6 は LOW で、L5・L6 は直済み)、`docs/reports/verify-2026-10-29-local-config-org.md`(再実行 2 回とも pass。LOW の V1・V2・V4、V3 は fixed、情報の I1〜I3・I5・I6)、`docs/reports/test-2026-10-29-local-config-org.md`(再実行が pass、Test gaps 5 つ)
- 差分の大きさ(`git diff origin/main...HEAD --shortstat`、この run の前): 74 ファイル、+3076/-975。walkthrough を書くかどうかは `/pr` が決める
- 制約: コードは変えていないので、`go test`・`go build`・`golangci-lint` は打っていない。打ったのは `check-sync.sh`、`check-skill-sync.sh`、`check-template-purity.sh`、`run-static-verify.sh`、`plan-visual.sh digest`、`git diff --check`、tech-debt の欄数と plan 参照の確認、`git grep` で、`secret-scan-branch.sh --strict` は commit の前に打つ

## Summary

この branch は issue #276 の 1 本目(A1)の対応で、個人用の設定ファイル `ralph.local.toml` を足し(`ralph.toml` をキーごとに上書き、git の中では主 worktree の root のものを読む)、`RALPH_ORG_AGMSG_HOME` を読まなくした。gate のスクリプトとテストは `RALPH_LOCAL_CONFIG=off` で個人用のファイルから離した。

文書は S5 がほぼ済ませていて、README、`AGENTS.md`、`templates/base/ralph.toml`、`/org` と `/director` skill(4 つの写し)、recipe 2 本(2 つの写し)は branch の動きと合っていた。この run で直したのは次の 3 種類。

- tech-debt: 依頼の 2a〜2g。Trigger に当たった行 12 行に持ち越しの注、L1 の理由の書き直し、RESOLVED の注 2 行の直し、`isolateGitEnv` の行に写し 2 つの注、177 行の (b) の取り消し、新しい 2 行。
- spec: multi-org spec の FR-6 と FR-9 に 2026-10-30 の箇条(I3)、org-runtime spec の FR-2 の注(`agmsg_home` も `scripts/ralph-config.sh` から消えた)。
- README の 1 文(個人用のファイルの空の `agmsg_home`。self-review の N3)と、計画の AC の checkbox と Progress の 1 行。

## Changes made

| File | Change |
|------|--------|
| `docs/tech-debt/README.md`(`resolvedOrClean` の行、最後から 3 行目) | 依頼 2a(L1)。Why deferred の欄を、S1 が `feature.go` を直したので呼び出しは slice の外だけにはない、という事実に合わせて書き直した。wrapper を残した理由は、名前を付け替えるために `permissions.go`(`:209`、1 行に 2 呼び出し)、`split.go`(`:183`)、`feature_test.go`(5 呼び出し)を触らずに済ませるため。呼び出しは `git grep` で数え直した(本番は `feature.go:197`、`permissions.go:209` の 2、`split.go:183`。`feature_test.go` は 142・176・258・430・1730 行)。Related に self-review の L1 を足した |
| `docs/tech-debt/README.md`(`checkStartAgmsgHome` の D13 の変異 2 つの行) | 依頼 2b(V1)。Trigger の欄の終わりに持ち越しの注。S4b が `checkStartAgmsgHome` の引数(`cmdLaunch`)と env の警告を外し、拒否の文を `relativeAgmsgHomeError` にしたが、拒否の条件(`Changed("config") && configPath != "" && agmsgHomeIsRelative(...)`)と `startDirector` の拒否の順(`enabled`、次に `checkStartAgmsgHome`)は変えていない。(b) は行自身が決定を先に求めるので、2 本のテストは決定を下す小さい変更に回す。Trigger は残した。Related に verify の V1 と計画を足した |
| `docs/tech-debt/README.md`(`doctor_codex_writable_root_test.go` の行) | 依頼 2c(V4)。Trigger の前半に当たった: 3 つの seam のテストで `t.Setenv("RALPH_ORG_AGMSG_HOME", ...)` が `setAgmsgHomeOverride(t, ...)` になった(S4a)。(a) が名指すテストではなく、`runDoctorFull` の対象 directory の決め方も変えていない(`doctor.go` の差分は check 9b の追加だけ)ので、払わず持ち越した |
| `docs/tech-debt/README.md`(org-gate cycle 2 の小さい所見の行) | 依頼 2c(V4)。`gate.go`(`GateParams.Env` の doc)と `gate_run.go`(`gateScriptEnv`、新しい `gateLocalConfigEnv`)、`gate_test.go` と `org_gate_test.go` の 1 行に当たった。`recordGate`、`gateInterrupted`、`acquireFlock`、`stopGateGroup`、`runGateScript`、`lockfile.go`、(a) が名指す 2 つの中断のテストは変えていないので、(a)〜(d) と (f)〜(i) は触れず、持ち越した |
| `docs/tech-debt/README.md`(`wip:` を gate が見ない行) | 依頼 2c(V4)。同じ 2 ファイルに当たった。変えたのはスクリプトの環境で、`ralph org gate` が何を検査するか、`gate_passed` が何を意味するかは変えていない(`wip:` の検査はそれを変える)。leader の雛形の step 12・14 は触れていない。持ち越した |
| `docs/tech-debt/README.md`(`director_start.go` の file size の行) | 依頼 2c(V4)。`DirectorStartParams.AgmsgHome` とその doc を消した(origin/main の 1205 行から 1201 行。行の「1203 lines」は origin/main の時点ですでに 2 行ずれていた)。フィールド 1 つの削除は launch_cmd の半分の移動ではなく、move-only の分割は Scope の外なので、持ち越した |
| `docs/tech-debt/README.md`(recipe が `docs/evidence/` を引く行) | 依頼 2c(V4)。`isolated-herdr-server.md` と `director-launch-cmd.md` を 2 つの写しとも直したが、引用は増やしも言い換えもしていない。数を `git show origin/main:` と HEAD で数え直した(前者は 13 回・6 ファイル、後者は 1 回・1 ファイル。tech-debt の引用は前者に 1 回。どちらも前後で同じ)。Trigger の欄に注、Related に計画を足した。行は開いたまま |
| `docs/tech-debt/README.md`(`/org` skill の command table の行、F-8) | 依頼 2c(V4)。(b) を `~~` で取り消した。`git show 216b2cdd` で確かめた: 2026-10-09 の `feat: add ralph org start --plan --feature and show the binding in status` が、`newOrgStartCmd` の `--scope` の usage の末尾を、backtick を含む形から `, as for ralph org spawn --scope` に変えていて、`216b2cdd` は `origin/main` の祖先、HEAD の `internal/cli/org.go:1021` に backtick はない。(a) は、この branch が command table の `watch`・`start`・`gate`・`director` の 4 行を 4 つの写しとも変えた(`stop` と `disband` は変えていない。`git diff` で table の変わった行はその 4 行だけ)ので、持ち越しの注を足した。`watch`・`start`・`gate`・`regate`・`director` の行はどれも `stop`・`disband` の cell より長い |
| `docs/tech-debt/README.md`(RESOLVED の HTML コメントのうち、`TestStopDirectorProcess_AProcessThatIgnoresSIGTERMIsNotEndedInTime` の行の上) | 依頼 2d(I5)。「up to 10 s like the other marker waits of the package」を、4 つの `waitForFileWithin` の呼び出しに合わせた(10 s は `cmdStartFixture.release`、`startFakeAgent`、`TestStopDirector_LaunchCmdStopsALaunchScriptPausedAfterItsCheckIn`、30 s は `agentMarkerLines`)。行番号でなく名前で引いた |
| `docs/tech-debt/README.md`(`LegacyWorktreeStateDir` の 2 行: 読めない ledger の message と、org-state-dir-common の test gap) | 依頼 2e(I6)。Trigger に当たったと判断して持ち越しの注を足した。判断の理由は下の「Decisions」 |
| `docs/tech-debt/README.md`(org-feature-worktree の test gaps の行、(j)) | 依頼 2e(I1)。`samePath` を本体を変えずに `gitroot.SamePath` へ移し、`LedgerMainWorktreeRoot` を新しい名前に付け替えたので当たった。`TestSamePath`(`internal/gitroot/gitroot_test.go`)に「まだない path がシンボリックリンクの親の下にある」case はないままなので、(j) は開いたまま、持ち越した |
| `docs/tech-debt/README.md`(org-watch-gate の RESOLVED の HTML コメント) | 依頼 2e(I2)。残してよい履歴だが、消えた `orgWatchChildAgmsgHome` と 2 つのテストを今の形で名指すので、末尾に「S4b で消え、`orgWatchChild` が相対の `agmsg_home` を断るようになった」の 1 文を足した(`git grep` で 3 つの名前が `internal` に 0 件なのを確かめた) |
| `docs/tech-debt/README.md`(leader の template のテストなど 5 ファイルの 800 行の行) | 依頼 2e(I6)。leader の template を描画する `TestRenderRolePrompt_AgmsgScriptsIsTheQuotedScriptsDir`(`prompts_leader_inbox_test.go`)を書き換え、`spawn.go` を 1 行、`spawn_test.go` を 5 行増やしたので当たったと判断した。どのファイルも +300 に届かず、`verbs.go`・`verbs_test.go`・`prompts_test.go` は変わっていない。行の行数は origin/main の時点ですでに古い(`spawn.go` 2690、`spawn_test.go` 6025。行は 2680 と 5991)ので、それも書いた |
| `docs/tech-debt/README.md`(`isolateGitEnv` の行) | 依頼 2e(I6)。Trigger には当たっていない(`internal/org` の `isolateGitEnv` は変わっていない)が、この branch が同じ作りで同じ穴(`XDG_CONFIG_HOME` なし)の写しを 2 つ足した(`internal/gitroot/gitroot_test.go` の `isolateGitEnv`、`internal/config/local_test.go` の `isolateLocalGitEnv`)。どちらも `git status` を読まないので今は影響がない。直すときに写しも直せるよう、Trigger の欄に書き、Related に 2 ファイルを足した |
| `docs/tech-debt/README.md`(RC1: 実装と reviewer の座席が ralph の agmsg を使わない行) | 計画外だが sweep で見つけた分。Debt の欄は 2026-10-24 の再確認の記録で、private copy を `RALPH_ORG_AGMSG_HOME` で名指したと書く。この変数はもう読まれないので、Trigger の欄に「今は `[org].agmsg_home`(`ralph.local.toml`)で名指す。所見は変わらない」の注を足した。`{{AGMSG_SCRIPTS}}` の元の `RolePromptVars.AgmsgHome` の doc コメントを直したが role prompt は触れていないので、払わず持ち越した |
| `docs/tech-debt/README.md`(新しい 2 行) | 依頼 2f と 2g。2f: `TestStartDirector_ACleanupThatCannotConfirmTheEndWritesNothingOnceItIsNotCurrent/an_attach_--replace`(20 s の marker 待ち)と `TestWakeDirector_AFailedWakeIsNotTriedAgainForFiveMinutes/no_return_in_time`(`DriverCallTimeout` 3 s)の、負荷に弱い固定の期限。2g: regate のスクリプトに `RALPH_LOCAL_CONFIG=off` が届くことを見るテストがない、`TestOrgSend*` が stub の agmsg のログを見ない(最小の直しは `setupOrgStubPATH` が返す `agmsgLog` に `send.sh` があることを 1 本が確かめる) |
| `docs/specs/2026-10-07-org-multi-org-director.md`(FR-6 の末尾、FR-9 の末尾) | 依頼 2e(I3)。2026-10-30 に決めたことの箇条を 1 つずつ足した。FR-6 のは、見張りの子が相対の `RALPH_ORG_AGMSG_HOME` を絶対パスで受け取る動き(6 段目の後半)がなくなり、`--config` の相対の `agmsg_home` は `watch --all --detach` と `ralph org start` が起動前に断ること。FR-9 のは、起動スクリプトが `RALPH_ORG_AGMSG_HOME` を export しない(8a-2b)こと、start は警告せず、doctor が warn すること。過去の箇条は書き換えず、#265・#267 の前例どおり日付つきの箇条で足した |
| `docs/specs/2026-08-01-org-runtime.md:74`(FR-2) | 2026-10-28 の #273 の注(「`agmsg_home` のほかは照合から外し、2 面にした」)のあとに、2026-10-30 の注を足した。この branch で `agmsg_home` も `scripts/ralph-config.sh` から消え(`git grep -i agmsg scripts/ralph-config.sh` は 0 件)、そこに `[org]` の既定はない |
| `README.md:146` | self-review の N3。個人用のファイルの `[org].agmsg_home = ""` は `ralph.toml` の値ではなく内蔵の既定になる、の 1 文(`TestLoad_LocalAgmsgHome` の `{"", Default().Org.AgmsgHome}` が固定している) |
| `docs/plans/active/2026-10-29-local-config-org.md` | AC1〜AC13 を `[x]` にし、`## Progress checklist` に「Sync-docs report created」の行を足した。本文の節は触れていない。digest は前後で同じ(`1609f175543b`) |
| `docs/reports/sync-docs-2026-10-29-local-config-org.md` | この report |
| `docs/insights/events/2026-10-29-local-config-org.jsonl` | `sync_docs` の行を足した(下の「Insight event」) |

## Surfaces checked for drift

| Surface | 結果 | 根拠 |
|---------|------|------|
| `README.md` の `### ralph.local.toml`(`:142`〜`:146`)、`config get`・status・watch・director・gate・merge-order の段落 | N3 の 1 文を足した | 置き場所、上書きの決まり、git の無視、エラー、`agmsg_home` の形、env を読まなくなったことは branch と合う(verify の AC12 と同じ) |
| `AGENTS.md:89`・`:90`(`internal/config/` と `internal/gitroot/` の行) | 変更なし | `local.go` と `internal/gitroot/` の説明が branch と合う。`.ralph/core/AGENTS.core.md` と `templates/base/AGENTS.md` は `internal/` の地図を持たない(`git grep` で `internal/config`・`internal/gitroot` が 0 件) |
| `docs/architecture/repo-map.md`、`docs/quality/quality-gates.md`(と写し) | 変更なし | `internal/config`・`internal/gitroot`・gate の環境を挙げる行がない(`git grep`) |
| `.claude/rules/ralph/` | 変更なし | `RALPH_ORG_AGMSG_HOME` を含む行がない。`model-routing.md:151` の `defaults_sync_test.go` の列挙は `agmsg_home` を挙げず、この branch が消した shell 側の照合と矛盾しない |
| `/org` と `/director` skill の 4 つの写し | 変更なし | `check-skill-sync.sh`: `14 skill(s) in lock-step`、`check-sync.sh`: IDENTICAL 172、DRIFTED 0。skill の本文と `RALPH_ORG_AGMSG_HOME` の生きた参照は 0 件 |
| recipe `director-launch-cmd.md`・`isolated-herdr-server.md` と `templates/base/docs/recipes/` の写し | 変更なし | `cmp` で一致。`RALPH_ORG_AGMSG_HOME` の参照なし |
| `templates/base/ralph.toml`、`scripts/ralph-config.sh` とその写し、`tests/test-ralph-config.sh` | 変更なし | verify が `cmp`・`sh -n`・shellcheck で確かめた。`defaults_sync_test.go` の `agmsg_home` は config とテンプレートの 2 面だけ |
| `docs/specs/2026-08-01-org-runtime.md:157`(Research findings の「3 面ロックステップで ... env が唯一の普遍チャネル」)、`docs/specs/2026-10-07-org-multi-org-director.md:219`(「3 面の lock-step」)と `:243`(3 つのファイルを並べた表の行) | 直していない(履歴) | 調査の記録と計画時の文で、3 面は #273 からすでに古い。#273 で注が付いたのは `:74` だけ |
| tech-debt の `RALPH_ORG_AGMSG_HOME` を名指すほかの行(`:333` の RESOLVED コメント、`:350`、`:369` の Debt の欄) | 直していない(履歴)。`:369` の Trigger の欄に注 | 2026-10-23〜24 の run の記録で、当時は事実。`:269`・`:270`・`:272`・`:273` は S5 が済ませている |
| 計画の本文の Rollout の「`/ralph.local.toml`」と Assumptions の「`statedir_test.go` に TestMain がある」 | 直していない | 本文を変えると digest が変わる。Progress に読み替えがある(`:198`、verify の I4)。release のノートと PR は先頭の `/` なしで書く |
| `docs/plans/templates/feature-plan.md` の Progress | 変更なし | 「Sync-docs report created」の行はテンプレートにない(前の計画も同じように足した)。テンプレートの直しは別の変更 |
| `docs/tech-debt/README.md` の欄数と plan 参照 | 通った | 欄数: 5 欄の行が 287 から 289 になり(新しい 2 行)、4 欄 1 行と 6 欄 1 行は前のまま。直した行は backtick が偶数、`~~` も偶数。plan 参照: `verify.local.sh` の `check_tech_debt_plan_refs` と同じ pipeline で missing 0 件、`run-static-verify.sh` の同じ check も OK。新しい参照は `docs/plans/active/2026-10-29-local-config-org.md`(`/pr` の `archive-plan.sh` が書き換える) |

## Decisions on the reports' remaining items

| 項目 | 扱い | 理由 |
|------|------|------|
| self-review L1(`resolvedOrClean` の行の理由が事実と違う) | 直した | 行の Why deferred を書き直した(上の表)。wrapper を消す code の変更は、この run の範囲外 |
| self-review L2(相対 `agmsg_home` の判定が `config` と `cli` に 2 つある)、L3(doctor の `Name: "ralph.toml"` と `using defaults`)、L4(`AgmsgHomeOverride` と env の名前が戻らないことを固定するテストがない)、verify V2(`config get` のエラーの頭が `ralph.toml:`) | 残した。PR の Known gaps | code とテストの変更で、`/sync-docs` は文書だけを直す。自己レビューも LOW(ユーザーの規則は CRITICAL と HIGH だけ直す)。orchestrator の指示でもある |
| self-review N3(個人用の `agmsg_home = ""` は既定に戻る) | 直した | README の 1 文 |
| self-review N1・N2・N4・N5 | 残した | Notes で、直しを求めていない。N2(linked worktree の写しは読まれず、知らせる所がない)は README がすでに書いている |
| verify V1 | 持ち越しの注を足した(2b) | Trigger は残した。(b) は決定が先 |
| verify V3 | 直済み(c7095378) | |
| verify V4 | 持ち越しの注を足した(2c)。177 行の (b) は取り消した | 5 行のうち 144・235・364・316・368 は注、177 は (a) の注と (b) の取り消し |
| verify I1(`samePath` の移動を「変更」と見るか) | 見る。198 行の (j) と 170・171 行に注 | Trigger は関数名・ファイル名で書かれていて、この branch は関数を移し、呼び出しを付け替えた。注は無害で、Trigger に当たったのに黙っている行より追跡しやすい。払わない理由は AC11(ledger の決まり方を変えない)と move-only の slice。170 の直し(`SamePath` の check を `os.Stat` の前に移す)は決まり方を変える |
| verify I2(RESOLVED のコメントが消えた名前を名指す) | 1 文を足した | 履歴は残し、今の形を末尾に書いた |
| verify I3(multi-org spec の 2 つの箇条) | 日付つきの箇条を足した | 過去の箇条は書き換えない(計画の決定、#265・#267 の前例) |
| verify I4(Assumptions の TestMain の位置) | 対応済み | e13b217a が Progress に記録している。計画の本文は digest のため直さない |
| verify I5 | 直した | 上の表 |
| verify I6(170・171・362・294 行) | 4 行とも注。294 は Trigger に当たらないが写しの注 | 170・171: 上の I1 と同じ。362: `TestRenderRolePrompt_AgmsgScriptsIsTheQuotedScriptsDir` は行の Related にある `prompts_leader_inbox_test.go` の leader の template のテストなので当たった。294: org の `isolateGitEnv` は変わっていないので当たっていないが、同じ穴の写しが 2 つ増えた |
| test report の Test gaps 1・2(regate の環境、stub の agmsg のログ) | tech-debt に 1 行(2g) | |
| test report の Test gaps 3〜5(gate・regate・detach した watch と director・`ralph upgrade`・本物の herdr と agmsg を打っていない、production の名前の grep を固定するテストがない、通らない分岐) | 残した。PR の Known gaps | 3 つ目は計画の範囲外か、テストの差し替えの制約。4 つ目は L4 と同じ |

## Release notes

(PR 本文の「リリースノート」の材料。`ralph.toml` は seed なので、`ralph upgrade` は既存の project の `ralph.toml` を書き換えない。)

1. ralph は `RALPH_ORG_AGMSG_HOME` を読まなくなった。agmsg の場所は `ralph.toml` の `[org].agmsg_home` で決まり、環境変数で変える方法はない。使っていた人は、値を `ralph.toml` の隣の `ralph.local.toml`(個人用のファイル、git が無視する)の `[org].agmsg_home` に、絶対パスか `~/` で始まる形で書く。環境に `RALPH_ORG_AGMSG_HOME` が残っていれば `ralph doctor` が warn を出す。黙って効かなくなると座席が別の agmsg に向くので、更新のあとに `ralph doctor` を打つ。
2. `ralph.local.toml` は `ralph.toml` と同じ形で、キーごとに上書きする(表はキーごと、配列は置き換え)。`ralph.toml` がなくても読み、`ralph config get` は上書きした値を返す。git の中では、どの worktree やサブディレクトリから打っても、主 worktree の root のものを読む。linked worktree やサブディレクトリに置いた写しは読まれず、何も言われない。読めない、または正しくない `ralph.local.toml` はパスを名指すエラーになる。個人用のファイルの `agmsg_home` は絶対パスか `~/` で始まる形だけを受け付ける。
3. `ralph upgrade` は `.gitignore` の管理ブロックに `ralph.local.toml`(先頭の `/` なし。どの階層でも無視する)を足す。`ralph upgrade` が足すことは、管理ブロックが upgrade の面であることを読んで確かめただけで、打っていない。おそらくそうなる。未確認です。
4. `--config` の `[org].agmsg_home` が相対パスなら、`ralph org watch --all --detach` と `ralph org start` も見張りを起動する前に終了コード 1 で断る(`ralph org director start` は前から断る)。見張りがすでに動いていれば何も起動しないので断らない。
5. `ralph org gate` と `ralph org regate` のスクリプトは、呼んだ側の値にかかわらず `RALPH_LOCAL_CONFIG=off` で走り、開発機の個人用のファイルで検証が変わらない。`RALPH_LOCAL_CONFIG=off` は、ちょうど `off` のときだけ個人用のファイルを読まない内部の受け渡しで、テストの TestMain も同じ値に固定する。
6. `ralph org director start` は `RALPH_ORG_AGMSG_HOME` について警告せず、`launch_cmd` の起動スクリプトも export しない。
7. #276 の残りは、A2(`RALPH_ORG_STATE_DIR`)と B(標準フローの値とスクリプト)。この PR は `Refs #276`。

自動の changelog(`.goreleaser.yml` は `^docs:`・`^test:`・`^chore:` を除く)に載る commit は、次の 4 本と merge commit である。

| commit | 題 |
|--------|----|
| `ab012b7d` | `refactor: move the git worktree helpers into internal/gitroot` |
| `9ed3793d` | `feat: read the personal ralph.local.toml on top of ralph.toml` |
| `08926b11` | `feat: ignore ralph.local.toml and keep the gate and tests off it` |
| `1c530966` | `feat: stop reading RALPH_ORG_AGMSG_HOME and warn about it in ralph doctor` |
| `d5386fb7` | `Merge remote-tracking branch 'origin/main' into feat/local-config-org`(prefix がなく、filter に掛からない) |

`1c530966` は題が `feat:` でも、読んでいた変数を読まなくする破壊的な変更なので、changelog の行だけでは足りない。release のノートの 1 を先に読ませる。

## Notes for reviewers(PR 本文に入れる点)

- Known gaps(コードとテストで、この PR では直していない): self-review の L2(相対 `agmsg_home` の判定が `config.checkLocalAgmsgHome` と `cli.agmsgHomeIsRelative` の 2 か所にある)、L3(`ralph doctor` が個人用のファイルのエラーを `Name: "ralph.toml"` で出し、`using defaults` と言いながら断られた相対の値を次の agmsg の check で使う)、L4(`AgmsgHomeOverride` への代入がテスト以外にないこと、`RALPH_ORG_AGMSG_HOME` の名前が `doctor.go` 以外に戻らないことを固定するテストがない)、verify の V2(`ralph config get` のエラーの頭が個人用のファイルのときも `ralph.toml:` になる)。
- tech-debt に足した行(上の表): 負荷に弱い 2 本のテスト、regate の環境と stub の agmsg のログを見るテストがない 2 つの Test gaps。
- 実行していない確かめ: 実際の `ralph org gate` と `ralph org regate`(gate の環境は `TestGate_ScriptEnvironmentIsFixed` が `Gate` の本物のスクリプトで見ている)、detach した watch と director の実際の起動、`ralph upgrade` が管理ブロックに行を足すこと、本物の herdr と agmsg での通し、git の `safe.directory` で断られる環境と `GIT_DIR` を継承する git hook の中から `ralph` を打つ場合(self-review の N2)。
- 負荷の下で落ちたテスト(1 回の打ち直しで通ったもの)は Progress と test report に記録がある。S6 と V3 の直しで、既知の不安定なテスト 2 本(tech-debt の 2 行)は RESOLVED になった。
- この branch は `RALPH_LOCAL_CONFIG=off` を TestMain で固定するため、`internal/cli`・`internal/org`・`internal/config` のテストは開発機の `ralph.local.toml` と本物の agmsg に触れない。test report の隔離の確かめ(カナリア 3 つ)が、固定を外すと落ちる(または届く)ことを見ている。

## Left open

- tech-debt: 上の持ち越しの注の行(開いたまま)と新しい 2 行。
- L2・L3・L4・V2(code とテスト)。
- 計画の本文の古い記述(Rollout の `/ralph.local.toml`、Assumptions の TestMain の位置)。

## Verification

| Check | Result |
|-------|--------|
| `./scripts/check-sync.sh` | rc 0。`PASS: all files in sync.`(IDENTICAL 172、DRIFTED 0、ROOT_ONLY 0、TEMPLATE_ONLY 10、KNOWN_DIFF 6) |
| `./scripts/check-skill-sync.sh` | rc 0。`[ok] check-skill-sync: 14 skill(s) in lock-step`。skill は変えていない |
| `./scripts/check-template-purity.sh` | rc 0。`PASS: no meta-repo-specific references found in templates.` |
| `./scripts/run-static-verify.sh` | rc 0(20 秒、load average 6 前後)。`All verifiers passed.`。中に `tech-debt README plan references`、check-sync、check-skill-sync を含む。出力は scratchpad に置いた |
| tech-debt の plan 参照 | `verify.local.sh` の `check_tech_debt_plan_refs` と同じ `grep -o -E 'docs/plans/(active\|archive)/[A-Za-z0-9._-]+' \| sed 's/\.$//' \| sort -u` で missing 0 件 |
| tech-debt の欄数 | 5 欄の行が 287 → 289、4 欄 1 行・6 欄 1 行は前のまま。変わった行は 16 行(新しい 2 行を除く)で、`diff` で確かめた。backtick は偶数で、`~~` も偶数(177 行が 0 から 8、198 行と 235 行は前と同じ) |
| `git diff --check` | rc 0、出力なし |
| 変えた 5 ファイルの U+FFFD | 0 件(`grep` でバイト `EF BF BD` を数えた。すべて Edit tool で変えた) |
| `./scripts/plan-visual.sh digest docs/plans/active/2026-10-29-local-config-org.md` | `1609f175543b`(`- Approved:` の行と同じ。AC の checkbox と Progress の編集の前後で同じ) |
| 打っていない | `go test`、`go build`、`golangci-lint`、`run-test.sh`、`run-verify.sh` の既定のモード。この run が変えたのは文書、計画の checkbox、この report、insight event で、コードは変わらない |

## Insight event

`docs/insights/events/2026-10-29-local-config-org.jsonl` の末尾に `sync_docs` の 1 行を足した(`./scripts/insights-append.sh --slug local-config-org --flow standard --phase sync_docs --verdict pass --cycle auto --source skill`、rc 0、ts 2026-10-29T16:48:27Z、cycle 1、findings は 0)。file は 9 行になり、全行が JSON として読める。
