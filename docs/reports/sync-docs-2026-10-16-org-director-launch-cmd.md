# sync-docs report: org-director-launch-cmd

- Date: 2026-10-16(JST)
- Plan: `docs/plans/active/2026-10-16-org-director-launch-cmd.md`(承認済み。`./scripts/plan-visual.sh digest` は `2aeecc48d0ba` で、`- Approved:` の行と同じ。この run で計画は触っていない)
- Pipeline cycle: 1。base は `3723d900`、branch は `feat/org-director-launch-cmd`。run 開始時の HEAD は `69484db4`(run 開始時の値で、この report を含む commit の SHA ではない)
- 先行 report: `docs/reports/self-review-2026-10-16-org-director-launch-cmd.md`(MEDIUM 1、LOW 8。全件直った)、`docs/reports/verify-2026-10-16-org-director-launch-cmd.md`(pass。V1 と I1 は 0f5fbf89 で直った)、`docs/reports/test-2026-10-16-org-director-launch-cmd.md`(pass。Test gaps のうち 4 は 6b9b27d7 で閉じた)
- コードとテストは変えていない。変えたのは Markdown 3 つだけ。skill の 4 面と recipe の 2 面は変える必要がなく、触っていない

## Summary

S4(a6dda336)と、そのあとの verify と test の直しで、launch_cmd の文書はほぼ揃っていた。この run で足したのは、依頼された tech-debt の 2 行、AGENTS.md の repo map に隠れた動詞を 1 句、repo-map の recipe の一覧に新しい recipe を足した 3 点。ほかの面は、コードとヘルプの出力に当てて、ずれを見つけなかった。stale な Go のコメントとヘルプも、新しいものは見つからなかった。

## Changes made

| File | Change |
|------|--------|
| `docs/tech-debt/README.md`(`ps -o lstart=` の精度の行の次) | Linux で wall clock が step したあと、`ps -o lstart=` が同じプロセスに別の起動時刻を返しうる、という行を足した。stop と start の後始末(`abortCmd`)が生きた director を別のプロセスと読み、何も送らずに `stopped` を記録する。安全な側(別のプロセスには送らない)だが、動いている director が `stopped` と記録される。未確認と書いた(test の Linux の container は 120 秒で 15,560 回読んで変化 0 回、時計は動かしていない)。直し方の候補は、許容幅を付けて比べる、または Linux で `/proc/<pid>/stat` の 22 番目の欄(起動からの clock tick)を読む |
| `docs/tech-debt/README.md`(表の末尾) | `abortCmd` の 2 つ目の `ErrDirectorStopped` の枝(`MarkStoppedIfUnchanged` が、stop に先に記録されていたと返す枝)に、テストがない、という行を足した。1 つ目の枝(最初の読み)は `TestStartDirector_AStopWhileWaitingForTheCheckInIsTheCleanupsRecord` が固定している。2 つ目は、同じ行の `err == nil ||` が通るので文のカバレッジに出ない。herdr 側の `abort` の同じ枝は `TestStartDirector_AStopRecordedWhileTheAgentStartsCountsAsTheCleanupsRecord` が固定している。下の「依頼 2 の確かめ」を参照 |
| `AGENTS.md`(`internal/org/` の行) | `director_cmd.go` の句に「起動スクリプトが隠れた動詞 `ralph org director launched` で check-in する」を足した。依頼で名指しされた項目。`ralph org --help` には出ない動詞で、`internal/org/` の行から grep で見つけられるようにした |
| `docs/architecture/repo-map.md`(`docs/recipes/` の行) | 一覧に `launch_cmd` の recipe(headless の director の端末)を足した。一覧に前からなかった `codex-seat-permissions` も足した(この PR の外だが、同じ行で見つけた欠けで、1 語) |

2 行の tech-debt はどちらも 5 列で、計画を `docs/plans/active/` のパスで指している(`/pr` の `archive-plan.sh` が書き換える)。Go のコードは `file:line` でなく関数名とテスト名で指した。

## 依頼 2 の確かめ

依頼は「まだ covered でなければ足す」だった。`beforeMarkStopped`(`internal/org/director.go` のテスト用の継ぎ目)を使うのは、check-in を重ねるテスト(`TestStartDirector_ACheckInDuringTheCleanupIsReadAgainAndStopped`)と stop 側のテストだけで、stop を重ねて `abortCmd` の `MarkStoppedIfUnchanged` に `ErrDirectorStopped` を返させるテストはない。実際に確かめた。

- scratchpad の overlay で、`abortCmd` の記録の枝から `|| errors.Is(err, ErrDirectorStopped)` を外した。`go test -overlay=<scratch> ./internal/org -run Director -count=1` は ok(32 秒)で、mutant は生き残った
- 対照として、1 つ目の枝(`case errors.Is(err, ErrDirectorStopped):`)を `case false && ...` にした overlay では、`TestStartDirector_AStopWhileWaitingForTheCheckInIsTheCleanupsRecord` が落ちた。overlay は効いている
- どちらも tracked file には触れていない(`git status` は私の 3 ファイルの変更だけ)

枝が壊れたときの影響は、start の終了時のエラーが `*DirectorLaunchCmdStoppedError` ではなく `*DirectorCleanupError` になり、もう要らない stop を勧める文になること。`director.jsonl` は同じになる。

## Surfaces checked for drift

| 面 | 結果 | メモ |
|----|------|------|
| `README.md`(`ralph org` の行、`ralph status` の行、director の段落) | 合っている | シェルなしの argv、4 つの置き換え、check-in、SIGTERM のみ、`RALPH_DIRECTOR_TOKEN` の断り、`RALPH_ORG_AGMSG_HOME` が届くこと、`launch cmd` と `terminal_name` の表示。start のヘルプの出力行と同じ形 |
| `AGENTS.md` | 直した | 上の変更。managed block は `check-sync.sh` が同期を見ていて PASS |
| `docs/architecture/repo-map.md` | 直した | skill の行は launch_cmd を言っていた。recipe の行を直した |
| `docs/specs/2026-10-07-org-multi-org-director.md`(FR-9 の 8a-2b の注、86 行目あたり) | 合っている | D1〜D11 の要約。注が挙げるテスト 4 本(`TestStartDirector_LaunchCmdOpensTheTerminalAndChecksIn`、`TestStopDirector_LaunchCmdEndsTheCheckedInProcessAndRecordsTheStop`、`TestOrgDirectorStart_LaunchCmdOpensTheTerminal`、`TestOrgDirectorStop_LaunchCmdEndsTheProcessAndLeavesTheTerminal`)は存在する。8a-2 の注の「8a-2b に送った」は当時の記録で、次の注がその続き。FR-9 の「Given director start ... herdr の専用 workspace」の受け入れ条件の checkbox は未チェックのまま(8a-3 が残るので FR-9 は閉じない) |
| `docs/specs/2026-08-01-org-runtime.md` | 変更不要 | director の動詞を並べていない。後継の仕様への 1 行の案内だけ |
| `docs/recipes/director-launch-cmd.md` と template の写し | 合っている | `cmp` で同じ。置き換えの規則(`\{[^{}]*\}`)は `internal/config/config.go` の `directorLaunchPlaceholderPattern` に、stdin が空であることは `runDirectorLaunchCmd` に、10 秒の打ち切りは `startCmd` の `callWithTimeout`(`DriverCallTimeout` の既定)に合う |
| `templates/base/ralph.toml` のコメント | 合っている | 4 つの置き換え、`{script}` か `{script_sh}` が要ること、戻るコマンドにすること、端末は人が閉じること。例は `TestTemplateRalphToml_LaunchCmdExampleIsValid` が読む |
| `/director` skill の 4 面と `/org` skill の 4 面 | 合っている | 4 面とも同じ md5(director は `e23aceaa…`、org は `37238361…`)。「まだないもの」から launch_cmd は外れている。書き換える動詞の一覧に `director launched` がある。skill は編集していないので `sync-skills.sh` は走らせていない |
| `.claude/rules/ralph/ralph-workflow.md` と template の写し | 変更不要 | 「`ralph org director start` ... `director stop`」は herdr に限らない書き方 |
| `.claude/rules/ralph/agent-messaging.md` | 変更不要 | message の形の規則で、launch_cmd に触れない |
| `docs/quality/quality-gates.md` と template の写し | 変更不要 | 見張りの関門の行の `director start` は launch_cmd の道にも当たる(AC5)。`director` の動詞のうち `start` 以外は見張りなしで動く、という文は `launched` にも当てはまる |
| `docs/tech-debt/README.md` | 直した | 上の 2 行。S4 の 5 行は、列の数、`plans/active/` の参照、13 件・12 ファイルの数(`grep -rn 'plans/active/2026-10-15-org-director-start' internal` は 13 件、12 ファイル)を確かめた |
| コマンドのヘルプ(`director start`、`stop`、`status`、`launched`、`director`、`attach`、`detach`、`ralph org stop`、`ralph status`、`ralph org`) | 合っている | HEAD の binary を scratchpad に build して読んだ。`launched` は `ralph org director --help` の一覧に出ず、`ralph org director launched --help` では読める(隠れた動詞)。`--timeout-ms` の説明は launch_cmd の道を含む。`--force` の説明も同じ |
| 断りの文(`startAttachedError`、`attachedError`) | 合っている | launch cmd の登録なら「director の端末とは別の端末から stop」、herdr なら「herdr の外から」と分けている |
| Go のコメントとヘルプの stale | 新しいものなし | S2a と S3 の途中の言い方(`until S3` など)、`TODO`、未来形の `8a-2b` は残っていない。8a-2 の計画を `plans/active/` で指す 13 件のコメントは tech-debt の既存の行にある |
| 絶対パスの `$HOME` | なし | 変更した文書と差分に `/Users/` も `hiroki.yoshioka` もない |
| 計画の Progress | 変更不要 | S1〜S4、実機の確認、3 つの report まで記録済み。AC1〜AC10 は付いている。sync-docs の行は計画の様式にない。`PR created` は /pr が付ける |

## Left open

- Linux で wall clock が step したあとの lstart。新しい tech-debt 行に書いた。未確認です。CI の ubuntu と実機で NTP の補正が入ったときだけ起きうる
- `abortCmd` の 2 つ目の `ErrDirectorStopped` の枝のテスト。新しい tech-debt 行に書いた。mutant は生き残る(上の確かめ)
- recipe の Orca と WezTerm の例は実機で動かしていない。tech-debt の既存の行にある
- HEAD の binary での本物の claude と codex。live run は 88bb29fd の binary で、そのあとのコードの変更は 98131af5 だけ(test と verify の probe が見ている)
- tech-debt の新しい行は計画を active のパスで指している。`/pr` の `archive-plan.sh` が archive に書き換える

## Verification

| Command | Result |
|---------|--------|
| `./scripts/check-skill-sync.sh` | `[ok] check-skill-sync: 14 skill(s) in lock-step` |
| `./scripts/check-sync.sh` | PASS(IDENTICAL 170、DRIFTED 0、ROOT_ONLY 0、KNOWN_DIFF 5) |
| `cmp docs/recipes/director-launch-cmd.md templates/base/docs/recipes/director-launch-cmd.md` | 同じ |
| `md5` で director と org の skill の 4 面 | それぞれ 4 面とも同じ |
| `docs/tech-debt/README.md` の `docs/plans/(active\|archive)/...` の参照 67 件が存在するか | 欠けなし(`verify.local.sh` の `check_tech_debt_plan_refs` と同じ規則を手で当てた) |
| 新しい 2 行の列の数 | 5 列 |
| `./scripts/plan-visual.sh digest <plan>` | `2aeecc48d0ba`(`- Approved:` の行と同じ) |
| `git diff --check` | 出力なし |
| HEAD の binary のヘルプ | 上の表のとおり |
| scratchpad の overlay の mutant 2 つ | 上の「依頼 2 の確かめ」 |

`./scripts/verify.local.sh` は、tech-debt の参照の確認のつもりで誤って全体を起動し、約 2 分半でテストの子プロセスごと止めた。終わっていないので、その結果は使っていない。止めたあと、tracked file への変更は私の 3 ファイルだけであることを `git status` で確かめた。

## Verdict

- Verdict: pass(sync-docs に stop condition はない)
- Not verified: Linux の wall clock の step、Orca と WezTerm の例、HEAD の binary での本物の claude と codex
