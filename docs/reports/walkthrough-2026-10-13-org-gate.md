# Walkthrough: org-gate(7 段目の後半、7b)

- Date: 2026-10-13
- Plan: docs/plans/archive/2026-10-13-org-gate.md(f001bd4f で active から移した)
- Branch: feat/org-gate(base 1bddc871。cycle 4 の sync-docs の commit を含めて 38 commits、46 files、+30459/-218。この walkthrough を書いた時点は 25 commits、45 files、+25324/-214 で、増えたのは cycle 3 と cycle 4 の直し 2 commit(8b681dc3、f1b0352e)と各段の記録)
- 読む範囲: 差分の大半(約 2 万 3 千行)は `docs/evidence/` のテストの出力で、読むのは結果の表だけでよい(test の報告の各節が要約している)。報告・evidence・insight を除くと 29 files、+5624/-218 で、そのうちテストが `internal/org/gate_test.go`(+1491)、`internal/cli/org_gate_test.go`(+909)、`tests/test-run-verify-concurrency.sh`(+498)など。ほかに `/org` skill の 4 面、leader と reviewer の雛形、`run-verify.sh` の 2 面、仕様の注、quality-gates、tech-debt

新しいファイルは `internal/org/gate.go`(gate の確認、base の決め方、記録)、`internal/org/gate_run.go`(スクリプトの実行、出力の buffer、環境)、`internal/cli/org_gate.go`(その CLI)と、それぞれのテスト。次の順に読むと、依存の向きに沿って追える。

## 1. event と見張り(`internal/org/seat.go`、`internal/org/watch.go`、4d5088c9)

- `EventGatePassed`(`gate_passed`)と `EventGateFailed`(`gate_failed`)は org 全体の event で、`SeatID` は空、状態の event ではない(`stateEvents` に入らない)。書くのは `ralph org gate` だけ
- `leaderActivityEventCount` が 2 つを leader の動きに数える。gate を走らせている間は manifest に event が増えないが、終わった時点で数える(AC6)

## 2. gate の確認と記録(`internal/org/gate.go`、4d5088c9、e0adb359、b04289c1)

- `Gate(ctx, GateParams{OrgID, Dir, Env})` は `GateResult{Refused, Passed, Reason, Head, Base, BaseRef, Branch, LogPath, Tail, Dirty}` とエラーを返す。`Refused` は打つ前の拒否で、何も記録せず、スクリプトも走らせない。エラーは manifest や git の失敗と中断
- 打つ前の確認の順: `gateOrgLife`(今の命の完全な結びつきと、org が走っていること)→ `gateCheckWorktree`(`git rev-parse --show-toplevel` を symlink を解いて結びつきの worktree と比べる)→ `gateCheckBranch`(detached HEAD と別のブランチを断る)→ `gateRebaseInProgress`(rebase の途中なら `--continue` か `--abort` を言う)→ `gateDirtyLines`(`git status --porcelain --untracked-files=all` が空)→ `resolveGateBase` と `gateCheckAncestor`(base ref の先頭が HEAD の祖先)→ `gateCheckScript`(2 つのスクリプトがあって実行できる)
- base の決め方(`resolveGateBase`): `refs/remotes/origin/HEAD` を完全な ref 名で読み(`--short` はローカルの `origin/<名前>` ブランチと紛れる)、なければ `main`、`master`。base ref は `refs/remotes/origin/<名前>`、なければ `refs/heads/<名前>`。origin のない repo の拒否は fetch を言わない(`gateNoBaseRefusal`)
- 走らせたあと(`gateCheckAfter`): HEAD が動いていれば `head_moved`、作業ツリーが汚れていれば `tree_changed`(汚れたファイルの一覧を `Dirty` に入れる)
- 記録(`recordGate`): manifest のロックを取り、読み直し、`gateOrgLife` で org の命と結びつきが走らせる前と同じかを確かめる。違えば(走らせている間に disband、再起動、別の機能への結びつき)何も記録せず断る。同じなら `gateRecord.details()` の 1 行を足す。Details は `[reason=..] head= base= base_ref= branch= by= herdr_pane= log=<strconv.Quote>` で、`log=` を最後に置く(パスに空白がありうる)。`gateRecordFromDetails` が読み戻しを固定する
- 中断(`gateInterrupted`): ctx が終わっていれば、どの段でも「the scripts were stopped and nothing was recorded」の形のエラーを返す。cross-review cycle 1 #1 の直し(b04289c1)で、`recordGate` が ctx を受け取り、ロックを取る前とロックの中で追記する直前に確かめるようにした。ロックの待ちは `withManifestLockContext`(`internal/org/lockfile.go`)で呼び手の ctx から期限を作るので、中断で待ちが切れる。既存の `withManifestLock` の呼び手は `context.Background()` に委ねるだけで、動きは変わらない

## 3. スクリプトの実行(`internal/org/gate_run.go`、e0adb359)

- `runGateScripts` は `./scripts/run-static-verify.sh`、通れば `./scripts/run-test.sh` を worktree のルートで順に走らせ、出力を `createGateLog` が作る `<state-dir>/gates/<org_id>-<UTC>.log` に書く。worktree を汚さない
- `runGateScript` はスクリプトを自分のプロセスグループ(`Setpgid`)で走らせる。ctx が終われば `stopGateGroup` がグループに SIGTERM を送り、`gateKillGrace`(5 秒)待って SIGKILL。普通に終わったあとに残ったグループは `stopGateStrays` が止めて log に書く
- `gateScriptEnv` は打った側の環境から名前が `RALPH_VERIFY_` か `HARNESS_VERIFY_` で始まる変数を除き、`RALPH_VERIFY_SCOPE=full` を足す(計画の B9: 打った側の `RALPH_VERIFY_BASE=HEAD` などで検証を省けないように)
- `gateOutput` は出力を log に流しながら最後の 20 行(`gateTailLines`)を持つ。汚れたファイルの一覧も 20 行まで(`GateDirtyListLines`)

## 4. CLI(`internal/cli/org_gate.go`、`internal/cli/org.go`、296e9d6e、e0adb359、8b681dc3)

- `newOrgGateCmd` は `--org-id` を要求し、`--state-dir` と `--config` を他の動詞と同じに受け付ける。`gateSignalContext` で SIGINT、SIGTERM、SIGHUP を ctx の取り消しにする
- `gateSignalContext`(8b681dc3)は最初の信号で ctx を「`<信号> signal received`」を cause にして取り消し、中断のエラーが信号の名前を言うようにする。`signal.NotifyContext` は信号を cause に付けるのが Go 1.26 からで、go.mod は 1.25.8(release の build も同じ)なので使わない。PR #225 の CI で、中断のテストの信号の 3 subtest が「context canceled」で落ちたのがきっかけ。後の信号も stop が呼ばれるまで捕まえるので、2 つ目の Ctrl-C で ralph が終わることはない。`signal.NotifyContext` は `org_watch.go` の 1 か所に残る(cause を読まない)
- `printGateResult`: 通れば `GATE: pass`、`HEAD:`、`BASE:`、`LOG:` の行(leader が RESULT のヘッダに写せる形)で終了コード 0。落ちれば `GATE: fail`、`REASON:`、`HEAD:`、`BASE:`、`LOG:` のあと、空行と 2 字下げの詳細(出力の最後か汚れたファイル)で終了コード 1。拒否は理由と次の手を 1 つ言って終了コード 1
- 6b の見張りの関門と 7a の承認の検査はかけない(gate は新しい自律の作業を始めない)

## 5. `run-verify.sh` の受け渡し(`scripts/run-verify.sh`、`templates/base/scripts/run-verify.sh`、`tests/test-run-verify-concurrency.sh`、629cf853)

- 1 回の実行の中で受け渡す終了コード・言語の範囲・docs だけかの印を、`mktemp` の `ralph-verify-status`、`ralph-verify-scope`、`ralph-verify-non-docs` に替えた。EXIT・HUP・INT・TERM の trap で消す。同じ worktree で検証が重なっても、片方の値をもう片方が上書きしない(計画の B10)
- 終了コードのファイルがない、空、0〜255 の整数でないときは失敗にする(前は成功に読めた)
- `.harness/state/verify-scope` は最後に `.$$.tmp` から `mv` で写して残す(読む側のため)
- テストは 37 件。古い `run-verify.sh` に当てると、kill されたブロックと重なった実行で終了コードが 0 になる 3 件が落ちる

## 6. 雛形と文書(2e3e95e3、e0adb359、6716a028、a1590e44、f1b0352e)

- `internal/org/prompts/leader.md` の「機能ごとの org」: reviewer が通したあとの順を、reviewer のレポートのコミット(`git status --porcelain` が空かを確かめる。残りは implementer に TASK でコミットを頼むか、BLOCKED)→ fetch と rebase → `ralph org report` を打って report だけをコミット → 計画の archive を別のコミット(SHA を控える)→ secret scan → gate をバックグラウンドで走らせて `.harness/state/gate-{{ORG_ID}}.out` から読む → 座席を止める → HEAD が gate の `HEAD:` と同じことを確かめて push → PR → RESULT に `HEAD:` と `BASE:` → worktree を残す → disband にした。archive のあとに implementer へ TASK を送るときは、rebase の途中なら `git rebase --abort`、続けて `git revert --no-edit <archive>`(per-plan の承認の記録は元のパスを指すため。計画の B11)
- reviewer のレポートのコミットを rebase の前に置いたのは f1b0352e(cross-review cycle 3 の #1)。gate をやり直す通過では、reviewer がコミット済みのレポートを書き換えるので、既定のブランチが進んでいると `git rebase` が `cannot rebase: You have unstaged changes` で断られる。rebase は始まっていないので、雛形の衝突の手当て(`git rebase --abort`)も効かない(verify cycle 4 と test cycle 4 が git だけで再現し、先にコミットすれば rebase が通ることを確かめた)。計画の Scope と AC8 は fetch と rebase を先に書いていて、計画は archive 済みで書き換えないので、この順は計画からの意図したずれ。`/org` skill の 4 面(「org の終わり方」の手順 1・2、要約の文、順を書いた行、チェックリスト)も同じ順にした
- `internal/org/approval_enforce_test.go` の `TestOrgSend_ApprovalCheck_PlanArchivedAndRestored` が、archive で断られ、revert で通ることを固定する
- `internal/org/prompts/reviewer.md`: 通す RESULT に `HEAD:` と `BASE:`(既定のブランチは origin/HEAD、なければ main、master)。ralph は読まない
- `TestRenderRolePrompt_Leader_FeatureOrgProcedure` ほか `internal/org/prompts_test.go` が手順を固定する。既存の assertion で書き換えたのは 3 本(手順の番号が変わった 2 本、`--state-dir` を付けるコマンドに `gate` を足した 1 本。f1b0352e は、この 3 本のうち節の冒頭の文(予約の外に書く手順)の 1 本を、「9 の reviewer のレポート、10 の report と計画の移動、12 の計画の戻し」に書き直した)。f1b0352e は手順 9 の中の順序と手順 10 の否定の assertion も足した。skill の締めの順を読むテストはない(test cycle 4 の生き残り E)
- `/org` skill の 4 面(動詞表の `gate`、機能ごとの org の手順、「org の終わり方」の順)、`docs/quality/quality-gates.md` と template の写し、仕様 FR-7 の「7 段目の後半(7b)で決めたこと」、AGENTS.md、README
- `docs/tech-debt/README.md` に 11 行を足し、「FR-7 の品質のゲートは雛形の指示で、hook で強制していない」の行を書き換えた

## パイプラインの往復

上限は既定の 2 で、cycle 2 の cross-review は指摘 0 だった。そのあと PR #225 の CI が落ちたので、上限を 3 に上げて cycle 3 を回した。cycle 3 の cross-review が ACTION_REQUIRED 1 件を出したので、上限を 4 に上げて cycle 4 を回した(`cycle-count.json` は 2 のまま)。cycle 4 の cross-review はこのあとで、この表に結果はない。

| 回 | 見つかったもの | 直した commit |
|---|---|---|
| self-review cycle 1 | MEDIUM 3(M1〜M3)、LOW 9 | e0adb359(`gate_run.go` へ分ける、SIGHUP、残ったグループを止める、rebase の途中と origin のない repo の拒否、雛形の revert とバックグラウンド)、aeba07cb(package doc)。SIGKILL で残るスクリプトは tech-debt の行 |
| verify cycle 1 | 文書のずれ D1〜D4 | 6716a028(D1〜D3)、D4 は計画の Notes |
| test cycle 1 | なし(mutation 15 件すべて red) | - |
| cross-review cycle 1 | ACTION_REQUIRED 1 件(スクリプトのあと、記録の前の中断が記録されうる) | b04289c1 |
| self-review cycle 2 | LOW C2-1〜C2-6 | a1590e44(tech-debt の 1 行) |
| verify cycle 2 | 文書のずれ C2-D1・C2-D2 | a1590e44 |
| test cycle 2 | mutation の C2h(5 秒のロック timeout の文に届くテストがない)、CLI の中断のテストが時々 5.7 秒 | a1590e44(tech-debt の 1 行の (h)・(i)) |
| cross-review cycle 2 | なし | - |
| PR #225 の CI(run 38401704870、38401983445) | `TestOrgGate_InterruptStopsTheScriptAndRecordsNothing` の SIGINT・SIGTERM・SIGHUP の 3 subtest が失敗。中断のエラーが「context canceled」で、信号の名前がなかった(`signal.NotifyContext` が cause を付けるのは Go 1.26 から) | 8b681dc3(`gateSignalContext`) |
| self-review cycle 3 | LOW C3-1(`gateSignalContext` の doc に標準の関数へ戻す合図がない)、C3-2(テスト名が後の信号の半分しか言わない) | 直さない。tech-debt の行の (j)・(k) |
| verify cycle 3 | 文書のずれ C3-D1〜C3-D4(この walkthrough 2 件、tech-debt 2 件) | cycle 3 の sync-docs |
| test cycle 3 | なし(go1.25.8 と go1.26.0 で pass)。mutation の生き残り M_d2・M_j。M_a は go1.26.0 でだけ生き残る | 直さない。tech-debt の行の (l)〜(n) |
| cross-review cycle 3 | ACTION_REQUIRED 1 件(gate をやり直すとき、reviewer がコミット済みのレポートを書き換え、既定のブランチが進んでいると `git rebase` が `cannot rebase: You have unstaged changes` で断られる。雛形の `git rebase --abort` も効かない) | f1b0352e(reviewer のレポートのコミットを手順 9 の先頭に動かし、rebase の前に作業ツリーを空にする。skill の 4 面と `prompts_test.go` も) |
| self-review cycle 4 | LOW C4-1〜C4-3(手順 9 の最初のコミットが空振りする場合と add の範囲、skill の 2 か所の折り返し、テストの doc comment が AC8 を上書きしたと言わない) | 直さない。tech-debt の行の (o)〜(q) |
| verify cycle 4 | 文書のずれ C4-D1〜C4-D3(この walkthrough、README の gate の段落の順、tech-debt に C4-1〜C4-3 がない) | cycle 4 の sync-docs |
| test cycle 4 | なし(go1.25.8 と go1.26.0 で pass)。mutation は雛形 13 件が red で、skill の 4 面を旧い順に戻す E が生き残った(締めの順を読むテストがない) | 直さない。tech-debt の行の (r)・(s) |

## 実機確認

一時の repo と台帳で、HEAD からビルドした binary を打った(verify の報告、`docs/evidence/verify-2026-10-13-org-gate-probe.txt`)。cycle 1 は拒否と通過と失敗の 9 通り、cycle 2 は manifest のロックを別のプロセスが持つ間の SIGTERM と SIGHUP(HEAD は 0.02 秒で rc 1、manifest は変わらない。修正前の 789c8864 は、ロックが空いたあとに `GATE: pass` と rc 0 で `gate_passed` を書いた)、cycle 3 は go1.25.8 で build した HEAD の binary への 7 通りの入力(ロックの待ちと走行中のスクリプトへの SIGINT・SIGTERM・SIGHUP、中断なしの通過、2 つ目の SIGINT)。中断はどれも rc 1 で信号の名前を出し、manifest は変わらず、2 つ目の SIGINT でも ralph は終わらない。修正前の f001bd4f の binary は「context canceled」を出した。cycle 4 は ralph の binary を使わず、git 2.49.0 だけで手順 9 の順を打った(verify の P1〜P3、test の R1〜R5、`docs/evidence/verify-2026-10-13-org-gate-probe.txt` と `docs/evidence/test-2026-10-13-org-gate-mutation.txt` の「Cycle 4」)。gate をやり直す通過で旧い順だと `git rebase` が rc 1 で断られ、始まっていない rebase への `git rebase --abort` は rc 128。先にレポートをコミットすれば porcelain は空になり、rebase は rc 0。herdr と本物のモデルの座席は起動していない。
