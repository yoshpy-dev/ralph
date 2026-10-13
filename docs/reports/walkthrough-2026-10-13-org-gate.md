# Walkthrough: org-gate(7 段目の後半、7b)

- Date: 2026-10-13
- Plan: docs/plans/archive/2026-10-13-org-gate.md(PR 作成時に active から移す)
- Branch: feat/org-gate(base 1bddc871。PR 作成の時点で 25 commits、45 files、+25324/-214)
- 読む範囲: 差分の大半(約 2 万行)は `docs/evidence/` のテストの出力で、読むのは結果の表だけでよい(test の報告の各節が要約している)。報告と計画を除くと 29 files、+5419/-214 で、そのうちテストが `internal/org/gate_test.go`(+1491)、`internal/cli/org_gate_test.go`(+832)、`tests/test-run-verify-concurrency.sh`(+498)など。ほかに `/org` skill の 4 面、leader と reviewer の雛形、`run-verify.sh` の 2 面、仕様の注、quality-gates、tech-debt

新しいファイルは `internal/org/gate.go`(gate の確認、base の決め方、記録)、`internal/org/gate_run.go`(スクリプトの実行、出力の buffer、環境)、`internal/cli/org_gate.go`(その CLI)と、それぞれのテスト。次の順に読むと、依存の向きに沿って追える。

## 1. event と見張り(`internal/org/seat.go`、`internal/org/watch.go`、4d5088c9)

- `EventGatePassed`(`gate_passed`)と `EventGateFailed`(`gate_failed`)は org 全体の event で、`SeatID` は空、状態の event ではない(`stateEvents` に入らない)。書くのは `ralph org gate` だけ
- `leaderActivityEventCount` が 2 つを leader の動きに数える。gate を走らせている間は manifest に event が増えないが、終わった時点で数える(AC13)

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

## 4. CLI(`internal/cli/org_gate.go`、`internal/cli/org.go`、296e9d6e、e0adb359)

- `newOrgGateCmd` は `--org-id` を要求し、`--state-dir` と `--config` を他の動詞と同じに受け付ける。`signal.NotifyContext` で SIGINT、SIGTERM、SIGHUP を ctx の取り消しにする
- `printGateResult`: 通れば `GATE: pass`、`HEAD:`、`BASE:`、`LOG:` の行(leader が RESULT のヘッダに写せる形)で終了コード 0。落ちれば `GATE: fail`、`REASON:`、`HEAD:`、`BASE:`、`LOG:` のあと、空行と 2 字下げの詳細(出力の最後か汚れたファイル)で終了コード 1。拒否は理由と次の手を 1 つ言って終了コード 1
- 6b の見張りの関門と 7a の承認の検査はかけない(gate は新しい自律の作業を始めない)

## 5. `run-verify.sh` の受け渡し(`scripts/run-verify.sh`、`templates/base/scripts/run-verify.sh`、`tests/test-run-verify-concurrency.sh`、629cf853)

- 1 回の実行の中で受け渡す終了コード・言語の範囲・docs だけかの印を、`mktemp` の `ralph-verify-status`、`ralph-verify-scope`、`ralph-verify-non-docs` に替えた。EXIT・HUP・INT・TERM の trap で消す。同じ worktree で検証が重なっても、片方の値をもう片方が上書きしない(計画の B10)
- 終了コードのファイルがない、空、0〜255 の整数でないときは失敗にする(前は成功に読めた)
- `.harness/state/verify-scope` は最後に `.$$.tmp` から `mv` で写して残す(読む側のため)
- テストは 37 件。古い `run-verify.sh` に当てると、kill されたブロックと重なった実行で終了コードが 0 になる 3 件が落ちる

## 6. 雛形と文書(2e3e95e3、e0adb359、6716a028、a1590e44)

- `internal/org/prompts/leader.md` の「機能ごとの org」: reviewer が通したあとの順を、fetch と rebase → `ralph org report` と reviewer の報告のコミット → 計画の archive を別のコミット(SHA を控える)→ secret scan → gate をバックグラウンドで走らせて `.harness/state/gate-{{ORG_ID}}.out` から読む → 座席を止める → HEAD が gate の `HEAD:` と同じことを確かめて push → PR → RESULT に `HEAD:` と `BASE:` → worktree を残す → disband にした。archive のあとに implementer へ TASK を送るときは、rebase の途中なら `git rebase --abort`、続けて `git revert --no-edit <archive>`(per-plan の承認の記録は元のパスを指すため。計画の B11)
- `internal/org/approval_enforce_test.go` の `TestOrgSend_ApprovalCheck_PlanArchivedAndRestored` が、archive で断られ、revert で通ることを固定する
- `internal/org/prompts/reviewer.md`: 通す RESULT に `HEAD:` と `BASE:`(既定のブランチは origin/HEAD、なければ main、master)。ralph は読まない
- `TestRenderRolePrompt_Leader_FeatureOrgProcedure` ほか `internal/org/prompts_test.go` が手順を固定する。既存の assertion で書き換えたのは 3 本(手順の番号が変わった 2 本、`--state-dir` を付けるコマンドに `gate` を足した 1 本)
- `/org` skill の 4 面(動詞表の `gate`、機能ごとの org の手順)、`docs/quality/quality-gates.md` と template の写し、仕様 FR-7 の「7 段目の後半(7b)で決めたこと」、AGENTS.md、README
- `docs/tech-debt/README.md` に 11 行を足し、「FR-7 の品質のゲートは雛形の指示で、hook で強制していない」の行を書き換えた

## パイプラインの往復

上限は既定の 2 のまま(cycle 2 で cross-review が指摘 0)。

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

## 実機確認

一時の repo と台帳で、HEAD からビルドした binary を打った(verify の報告、`docs/evidence/verify-2026-10-13-org-gate-probe.txt`)。cycle 1 は拒否と通過と失敗の 9 通り、cycle 2 は manifest のロックを別のプロセスが持つ間の SIGTERM と SIGHUP(HEAD は 0.02 秒で rc 1、manifest は変わらない。修正前の 789c8864 は、ロックが空いたあとに `GATE: pass` と rc 0 で `gate_passed` を書いた)。herdr と本物のモデルの座席は起動していない。
