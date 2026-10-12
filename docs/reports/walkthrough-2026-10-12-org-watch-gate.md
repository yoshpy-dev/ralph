# Walkthrough: org-watch-gate(6 段目の後半)

- Date: 2026-10-12
- Plan: docs/plans/archive/2026-10-12-org-watch-gate.md(PR 作成時に active から移す)
- Branch: feat/org-watch-gate(base dc41c4f5。PR 作成の時点で 40 commits、41 files、+7359/-361。うち `internal/` は 21 files、+4036/-272 で、そのうちテストが 11 files、+3177/-61)
- PR: #223

新しいファイルは `internal/org/watchgate.go`(検査とそのエラー)だけで、ほかは今のファイルに足した。次の順に読むと、依存の向きに沿って追える。

## 1. 生存の印と判定(`internal/org/watchalive.go`、`internal/org/watch.go`、S1 20cf6fdf)

- 印 `WatchAlive` に `deadman_minutes`(ポインタ、omitempty)と `once` を足した。見張りは自分の設定の値をそのまま書き、`--once` の見張りは `once: true` を書く
- 印の状態に `stopping` を足した。見張りは後始末(`drainBeforeStop`)に入るときに `stopping` を書き、後始末の間の書き直しも `stopping`、最後に `stopped` を書いてからロックを放す
- `ProbeWatch` は、ロックが持たれていて印が `stopping` で窓の内なら `stopping` を返す(前は `stale` に読まれた)。窓は、印に `deadman_minutes` があればその値で、なければ呼んだ側の値で決める(tech-debt 201 行)。前半の版の印も読める
- `org.DefaultWatchLockWait` を外に出した(CLI が子の終わりを待つ時間の根拠に使う)
- `WatchAllParams.OnLocked` はロックを取ったあとに呼ばれる。CLI はここで起動の banner を出す(tech-debt 204 行の (e))

## 2. 検査(`internal/org/watchgate.go`、`spawn.go`、`feature.go`、`verbs.go`、S1 20cf6fdf)

- `WatchAcceptsWork` は、`running` で `--once` でない見張りだけを通す
- `NewWatchGate(stateDir, hintStateDir, now, deadman, settle)` は `func() error` を返す。`stale` のときは `settle`(`DefaultWatchGateSettle` 2 秒)まで判定し直す。エラーは `*WatchGateError` で、`errors.Is(err, ErrNoRunningWatch)` に合い、状態ごとの次の手(起動、`--stop`、kill の前の確かめ、終わるまで待つ)を言う。`--state-dir` を明示したときだけ、案内のコマンドに付ける(`CommandWithStateDir`)
- `Org.WatchGate` が nil なら検査しない。org 層の既存のテストはこの前提のまま通る
- 位置: `Spawn` は `checkSpawnInput` の直後(dry-run の枝と `withManifestLock` の前)、`StartFeature` は `worktrees.Ensure` の前、`Send` は TYPE が `TASK` のときだけ(`--raw` も `protocol.Parse` で 1 行目を読む)。断ったときは manifest にも receipt にも書かない(F1)。`Send` のエラーは `taskNotSentError` で包み、`org: send: ` の接頭辞は org 層で 1 回だけ付ける

## 3. CLI の配線(`internal/cli/org.go`、`internal/cli/main_test.go`、S2 831bc342)

- `wireOrgWatchGate` を spawn・`start <task>`・`start --plan`・send に入れた。`newOrgRuntimeAt` には入れないので、stop・disband・escalate・inbox・watch・status は見張りなしで動く
- テストの差し替え口: `orgWatchGateOverride`(TestMain は「いつも通す」)と `orgWatchGateSettleOverride`(TestMain は 50ms)。CLI の既存のテストは書き換えていない
- spawn と `start <task>` は自分で台帳を 1 回解決してから `newOrgSpawnRuntimeAt` を呼ぶ。呼び手のなくなった `newOrgSpawnRuntime` は消した

## 4. start の起動と `--detach`(`internal/cli/org_watch.go`、S3 36b34bee、fdfb5b23、3fb0fd1b、8d8ddbd2)

- `ensureOrgWatch` を start と `--detach` が共有する。`running` ならそのまま、`stale` は最大 10 秒(`orgWatchEnsureDeadline`)`running` を待つ、`stopping` と `--once` は断る、`not_running` は子を起動する
- `launchAndAwaitOrgWatch` は子を起動し、どの pid の見張りでも作業を受け付ければ成功とする(tech-debt 206 行)。ほかの見張りが先にロックを取ったときは、自分の子が断られて終わるまで最大 `orgWatchChildExitWait`(`DefaultWatchLockWait` + 1 秒)待ってから返る(fdfb5b23、/test cycle 1 で見つけた後退の直し)。期限を過ぎて子が生きていれば止め、止めたあとにもう一度判定する(3fb0fd1b、self-review cycle 2 の C2-1)
- `wireOrgStartWatchGate` は、ensure を走らせてから `NewWatchGate` を返す。org 層の検査の位置で起動するので、入力の検査で断られる start は見張りを起動しない
- 子の起動の要求 `orgWatchLaunchRequest{Args, LogPath, Dir, Env}`(`orgWatchChild`): Dir は台帳の main worktree のルート(なければ打った場所)、Env は `HERDR_PANE_ID`・`HERDR_TAB_ID`・`HERDR_WORKSPACE_ID`・`HERDR_ENV` を外して `PWD` を Dir に、`--config` は絶対パス(tech-debt 205 行)
- Dir を変えるときは、相対の agmsg home を打った場所から絶対パスにし、子の env のただ 1 つの `RALPH_ORG_AGMSG_HOME` として渡す(`orgWatchChildAgmsgHome`、8d8ddbd2、cross-review cycle 2 #1)。値は env が空でなければ env、空か未設定なら明示した `--config` の `agmsg_home`。絶対パス、`~/`、`--config` なし(子は main のルートの `ralph.toml` を読む)は今のまま

## 5. `ralph status`(`internal/cli/status.go`、S1 20cf6fdf)

- `org watch:` の行に `stopping` と `running (--once)` を足し、`--json` の `org_watch` に `once` を足した。判定は印の `deadman_minutes` を使う

## 6. 雛形と文書(S4 ed66d634、sync-docs 2fada512・073e9119)

- `internal/org/prompts/leader.md` の `## 運用規律`: 見張りの不在で断られたら、打ち直さず、TYPE を変えたり `--raw` にしたりして送り直さず、見張りを自分で起動せず、BLOCKED を escalate して返事を待つ。`TestRenderRolePrompt_Leader_WatchRefusalEscalatesWithoutRetry` で固定した
- `/org` skill の 4 面、README、仕様 FR-6 の「6 段目の後半で決めたこと」(F7 の読み替え)、agent-messaging と quality-gates(root と template)、AGENTS.md
- `docs/tech-debt/README.md`: 201・205・206 行と 204 行の (e) を閉じ、見張りが途中で落ちたときの穴、別の端末の `--stop` をロックを待つ側が追い越す穴、`CommandWithStateDir` と `InboxCommand` の写し(self-review C3-2)を新しい行にした

## パイプラインの往復

上限は既定の 2 から 3 に上げた(cycle-count.json は skill の決まりで 2 のまま)。

| 回 | 見つかったもの | 直した commit |
|---|---|---|
| self-review 1 回目 | LOW 6 件(L1〜L6) | dbeb45ca(L1〜L4)、7d2e4bbb(CLI の send の二重の接頭辞)、L5・L6 は sync-docs の 2fada512 |
| self-review 再実行 | LOW 3 件(N1〜N3) | bda15444 |
| verify cycle 1 | 文書のずれ D1〜D7 | sync-docs の 2fada512 |
| test cycle 1 | 後退 1 件(同時の `--detach` 8 本の直後の `--stop` が 30 秒で時間切れ)、生き残った mutant 6 件 | fdfb5b23(後退の直しと probe 5 本の取り込み) |
| self-review cycle 2 | LOW 4 件(C2-1〜C2-4) | 3fb0fd1b(C2-1〜C2-3)、C2-4 は sync-docs の 2fada512 |
| verify cycle 2 | 文書のずれ D1〜D10 | sync-docs の 2fada512 |
| test cycle 2 | 生き残った mutant 1 件(K8) | 47635690(テストだけ) |
| cross-review cycle 2 | ACTION_REQUIRED 1 件(相対の agmsg home が子の作業ディレクトリの移動で別の場所を指す) | 8d8ddbd2 |
| self-review cycle 3 | LOW 2 件(C3-1・C3-2) | C3-1 は triage の付録で指し先を保ち、C3-2 は sync-docs の 073e9119 で tech-debt に |
| verify cycle 3 | 文書のずれ D11〜D15 | sync-docs の 073e9119 |
| test cycle 3 | なし(生き残りの 1 件は等価) | なし |
| cross-review cycle 3 | 指摘 0 件 | なし |
