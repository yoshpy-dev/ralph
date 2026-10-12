# Verify report: org-watch-gate

- Date: 2026-10-12
- Plan: docs/plans/active/2026-10-12-org-watch-gate.md(承認済み。`scripts/plan-visual.sh digest` の値 de5ca6b7e6f0 は `- Approved:` 行と一致する)
- Verifier: verifier subagent (Claude)。パイプライン 1 回目(cycle 1、上限 2)
- Scope: 仕様適合、静的解析、文書の整合。対象は `git diff dc41c4f5...HEAD`(HEAD 9823d97d、36 ファイル、+3,727/-350)。self-review を通していない bda15444 は全文を読んだ。テストは走らせていない(/test の担当)。ブランチのビルド(`go build ./cmd/ralph`)で、repo の外の一時の台帳を使った実機確認をした。herdr の座席は立てていない(spawn と send は `--dry-run`、start は未知の model で envelope に断らせた)
- Evidence: `docs/evidence/verify-2026-10-12-org-watch-gate.log`(gitignore の対象で、コミットしない)

## Spec compliance

コードとテストを読み合わせて判定した。テストが通るかどうかは /test で確かめる。

| Acceptance criterion | Status | Evidence |
| --- | --- | --- |
| AC1 見張りがないときの spawn の拒否 | 満たす | 検査は `checkSpawnInput` の直後で、manifest の読みと dry-run の枝(`internal/org/spawn.go:493`)より前にある(`:470`)。拒否は `SpawnOutcomeRejected` を返すだけで `reject()` を通らないので、`rejected` も receipt も書かない。CLI の配線は `wireOrgWatchGate`(`internal/cli/org.go:266`、spawn は `:554`)。状態ごとの次の手は `WatchGateError.Error`(`internal/org/watchgate.go:43-68`)。テストは `TestOrgSpawn_WatchGate_RefusesAWatchThatAcceptsNoWork`(`internal/cli/org_watch_gate_test.go:154`。6 状態それぞれで、実行と `--dry-run` の両方を見る。`checkNothingStarted` が manifest・receipt・herdr・agmsg を確かめる)と `TestOrgSpawn_WatchGate_ARefusalRecordsAndCallsNothing`(`internal/org/watchgate_positions_test.go:37`)。running で通ることは `TestOrgSpawn_WatchGate_ARunningWatchPasses`(cli `:181`)。実機では、見張りのない台帳で `spawn --dry-run` が rc=1 になり、manifest は作られなかった |
| AC2 TASK の send だけを断る | 満たす | `Send` は最初に `protocol.Parse` を呼び(`internal/org/verbs.go:293`)、TYPE が `TASK` のときだけ検査する(`:303-307`)。検査は dry-run の枝(`:314`)より前にある。raw のメッセージにも同じ `Parse` を使い、`Parse` が key と値を `TrimSpace` する(`internal/org/protocol/protocol.go:140`、`:144`)。テストは `TestOrgSend_WatchGate_RefusesOnlyATask`(cli `:289`)と `TestOrgSend_WatchGate_RefusesATaskOnly`(`watchgate_positions_test.go:161`)。実機では、見張りなしで型付きの TASK と raw の `  TYPE:  TASK` が rc=1 になり、raw の `type: task` と QUESTION は rc=0 で送れた。manifest に残ったのは、通った 2 件の dry-run の記録だけ |
| AC3 start が見張りを起動する | 満たす | `wireOrgStartWatchGate`(`internal/cli/org.go:290-318`)が `ensureOrgWatch`(`internal/cli/org_watch.go:427`)を呼んでから本物の検査を呼ぶ。起動したときは stderr に 1 行を出す(`org.go:313`)。配線は start の 2 つの形にある(`:730`、`:827`)。起動した子は `WatchAcceptsWork` になるまで待つ(`org_watch.go:473-474`)。テストは `TestOrgStart_LaunchesTheWatchWhenNoneRuns`(`internal/cli/org_start_watch_test.go:152`)。実機では、kill -9 のあとの `start <task>` が `ralph org start: started the cross-org watch (pid 48334); log: …/watch.log` を出し、start が終わったあとも `ralph status` は `org watch: running (pid 48334)` だった。2 回目の start は何も起動しなかった。leader は envelope で断らせたので、「印が running になってから leader が立つ」順は実機では見ていない(テストが見る) |
| AC4 stale の拒否と、running にならない子の回収 | 満たす | stale の間は最大 `orgWatchEnsureDeadline`(10 秒)待ち、そのあとは `e.refuse` を返す(`org_watch.go:427-446`)。start の `refuse` は `WatchGateError` を返し、`StartFeature` は検査を worktree の前で呼ぶ(`internal/org/feature.go:198`、worktree は `:210` 以降)。起動した子が期限までに running にならなければ、子を止めてロックの解放を待ってから返す(`org_watch.go:498-507`)。テストは `TestOrgStart_WatchGate_RefusesAStaleWatch`(cli `org_watch_gate_test.go:396`)、`TestOrgStartPlan_WatchGate_RefusesAStaleWatchBeforeTheWorktree`(`:421`)、`TestOrgStart_ALaunchedWatchThatNeverRuns_IsStoppedAndTheStartRefused`(`org_start_watch_test.go:213`)。実機では、ロックを持ったまま 1 時間前の印を置いた台帳で、`--detach` が 10 秒待ってから stale の文で rc=1 になった(start で同じ台帳は試していない) |
| AC5 子が先に終わっても、ほかの見張りが running なら成功 | 満たす | `launchAndAwaitOrgWatch` は、どの pid の見張りでも `WatchAcceptsWork` なら成功とし、`launched` を pid で判定する(`org_watch.go:473-474`)。`--detach` は `launched` で `started` と `already running` を出し分ける(`:528`、`:531`)。テストは `TestOrgWatchDetach_AWatchStartedByAnotherProcessIsNotTakenForTheChild`(`internal/cli/org_watch_test.go:442`。終了コードの期待だけを 0 に変え、pid の照合は残した)、`TestOrgWatchDetach_AChildThatExitsBeforeTheOtherWatchsFirstMark_IsAlreadyRunning`(`:467`)、`TestOrgStart_AChildThatExitsWhileAnotherWatchRuns_StartGoesOn`(`org_start_watch_test.go:256`)。並行の `--detach` を実機で重ねて打つことはしていない |
| AC6 子の作業ディレクトリ、`--config`、環境 | 満たす | `orgWatchChild`(`org_watch.go:312-333`)は `org.LedgerMainWorktreeRoot` で main worktree のルートを求め(`:319`)、`orgWatchChildDroppedEnv` の 4 つ(`:300`)を外し、作業ディレクトリを変えたときは `PWD` も合わせる(`:331`)。`--config` は `filepath.Abs` で絶対パスにする(`orgWatchChildArgs`、`:277`)。`Env` を明示すると os/exec が `PWD` を直さない、というコメントの前提は Go の `Cmd.environ` の動き(`Env` が nil のときだけ `PWD` を足す)と合う。テストは `TestOrgStart_TheWatchRunsInTheMainWorktreeWithoutTheHerdrPaneEnv`(`org_start_watch_test.go:458`)、`TestOrgWatchDetach_TheChildRunsInTheMainWorktreeWithoutTheHerdrPaneEnv`(`org_watch_test.go:670`)、`TestOrgWatchDetach_StartDetachedProcess_RunsInDirWithEnv`(`:748`) |
| AC7 印の `deadman_minutes` | 満たす | 見張りは設定の値をそのまま印に書く(`internal/org/watch.go:603`、`:618`)。`ProbeWatch` は印に値があればそれを使い(`internal/org/watchalive.go:215-217`)、なければ呼んだ側の値を使う。テストは `TestRunWatchAll_TheMarkCarriesTheWatchsDeadmanMinutesAndOnce`(`internal/org/watch_all_test.go:861`)、`TestProbeWatch_JudgesTheWindowByTheDeadmanMinutesOfTheMark`(`internal/org/watchalive_test.go:267`)、`TestStatusCmd_OrgWatchJudgesTheMarkByTheWatchsOwnDeadmanMinutes`(`internal/cli/status_test.go:862`)、`TestStatusCmd_OrgWatchOfAWatchWithAnotherConfigReadsByItsMark`(`:899`)。実機の印にも `"deadman_minutes": 10` があった |
| AC8 見張りなしで使える動詞 | 満たす(一部は構造だけで確認) | `wireOrgWatchGate` と `wireOrgStartWatchGate` を呼ぶのは spawn・`start <task>`・`start --plan`・send の 4 か所だけである(`org.go:554`、`:730`、`:827`、`:901`。grep で確認)。`TestOrgVerbsThatStartNoWork_WorkWithNoWatch`(`org_watch_gate_test.go:497`)が本物の検査のもとで `ralph status`・org status・read・report・escalate・inbox・`wait --inbox`・stop・disband・`stop --all`・`disband --all` を通す。座席の `wait` と、`inbox show/ack/resolve/notify` はこのテストに入っていない。どれも検査を配線しないことは grep で確かめた |
| AC9 2 つ目の見張りは banner を出さない | 満たす | banner は `WatchAllParams.OnLocked` から出す(`org_watch.go:170`、`:218`)。`RunWatchAll` はロックを取ってから呼ぶ(`watch.go:539-541`)。テストは `TestOrgWatch_ASecondWatchIsRefusedWhileAnotherHoldsTheLock`(`org_watch_test.go:191`)と `TestRunWatchAll_CallsOnLockedOnceItHoldsTheLock`(`watch_all_test.go:736`)。実機では、動いている見張りの台帳で 2 つ目の `watch --all --once` が rc=1 になり、stdout は 0 バイトだった |
| AC10 既存のテストを書き換えない | 満たす(記録の漏れは下の「計画からのずれ」7) | CLI 側は TestMain が `orgWatchGateOverride` を「いつも通す」にし(`internal/cli/main_test.go`)、org 側は `Org.WatchGate` の既定が nil である(`spawn.go` の `Org` の欄)。既存のテストで行が消えたのは `internal/cli/org_watch_test.go` と `internal/org/watch_all_test.go` だけ(`git diff` の `-` 行で確認)。期待が変わったのは、Progress に書いた 4 本(後始末の間の印、`--once` の probe、AC5 の終了コード、stale の待ちを 300ms にしたもの)に、bda15444 の文言の変更(`nothing was started` から `so no new watch runs`)が加わる。ほかは起動の差し替え口の型(`orgWatchLaunchRequest`)に合わせた書き換えと、`orgWatchDetachDeadline` の改名で、期待は変えていない |
| AC11 leader の雛形 | 満たす | `internal/org/prompts/leader.md:236-248` の `## 運用規律` の項目。`TestRenderRolePrompt_Leader_WatchRefusalEscalatesWithoutRetry`(`internal/org/prompts_test.go:705`)が語句を固定し、`WatchGateError` の 5 通りの文が雛形の見分け方の語(`ralph org watch --all` と `no new work`)を含むことも確かめる。send の拒否は `taskNotSentError` が検査の文をそのまま後ろに付けるので、同じ語を含む(実機の文でも確認) |
| AC12 文書の整合と同期の検査 | 一部(文書の残り 5 件を /sync-docs で直す) | `scripts/check-sync.sh`(DRIFTED 0)と `scripts/check-skill-sync.sh`(13 skill)は rc 0。skill の 4 面と agent-messaging の 2 面は `cmp` で一致する。文書の残りは下の「Documentation drift」の D1〜D5 |
| AC13 止まりかけの見張り | 満たす | `drainBeforeStop` が後始末の前に stopping を書き、間隔ごとに書き直す(`watch.go:781-806`)。`stopAndRelease` が stopped を書いてからロックを放す(`:814`)。`ProbeWatch` は stopping を返し(`watchalive.go:222-224`)、`WatchAcceptsWork` は通さない(`watchgate.go:107-109`)。`--detach` と start は `ensureOrgWatch` の最初の判定で断る(`org_watch.go:440-441`)。テストは `TestRunWatchAll_CallsBeforeStopWhileItStillHoldsTheLockAndTheMarkSaysStopping`(`watch_all_test.go:1222`)、`TestProbeWatch_AStoppingMarkIsStoppingOnlyWhileHeldAndFresh`(`watchalive_test.go:224`)、`gateRefusals` の stopping の行(spawn)、`TestOrgStart_AStoppingOrOnceWatch_IsRefusedAndNothingLaunched`(`org_start_watch_test.go:345`)、`TestOrgWatchDetach_AStoppingOrOnceWatch_StartsNothing`(`org_watch_test.go:603`)、`TestStatusCmd_OrgWatchShowsStoppingAndOnce`(`status_test.go:777`)。spawn と start のテストは、ロックを持ったプロセスに stopping の印を置いた台帳で見る。本物の見張りが後始末をしている間に spawn を打つテストはない(印の書き手と検査を別々に固定している) |
| AC14 `--once` の見張り | 満たす | 印の `Once` は `p.Cycles > 0`(`watch.go:619`)。`WatchAcceptsWork` が通さず、`ralph status` は `running (--once)` を出す(`internal/cli/status.go:172-175`)。`--json` には `once` が足された(`:434`)。テストは AC13 と同じ 4 本の `--once` の行と `TestRunWatchAll_TheMarkCarriesTheWatchsDeadmanMinutesAndOnce`。実機では、`--once` の見張りの最後の印に `"once": true` があった。動いている間の status の行は、1 サイクルがすぐ終わったので見られなかった |
| AC15 最初の印の前の stale を待つ | 満たす | spawn と TASK の検査は `NewWatchGate` が stale のときだけ settle(2 秒)まで判定し直す(`watchgate.go:123-141`)。start と `--detach` は `ensureOrgWatch` が 10 秒待つ(`org_watch.go:428-436`)。テストは `TestOrgStart_WaitsForAWatchThatWritesItsFirstMarkLate`(`org_start_watch_test.go:387`)、`TestOrgWatchDetach_AWatchThatWritesItsFirstMarkLate_IsAlreadyRunning`(`org_watch_test.go:640`)、`TestOrgSpawn_WatchGate_AStaleWatchThatTurnsRunningWithinTheSettlePasses`(cli `org_watch_gate_test.go:212`)、`TestNewWatchGate_AStaleWatchThatWritesItsMarkWithinTheSettle`(`internal/org/watchgate_test.go:156`)。実機では、本当に返らない見張りで spawn は 2 秒、`--detach` は 10 秒待ってから断った |

### Design decisions F1〜F10 とコード

| 決定 | コードとの一致 | 根拠 |
| --- | --- | --- |
| F1 拒否は記録しない | 一致 | spawn・StartFeature・Send の 3 か所とも、検査の拒否はエラーを返すだけで、`reject()` も `appendEvent` も通らない(`spawn.go:470-472`、`feature.go:198-200`、`verbs.go:303-307`) |
| F2 org 層の差し込み口、CLI の 3 コマンドだけが設定する | 一致 | `Org.WatchGate`(nil で検査しない、`watchgate.go:144-149`)。`newOrgRuntimeAt` には入れず、spawn・start・send の RunE だけが設定する。StartFeature は自分で呼び、そのあと Spawn がもう一度呼ぶ |
| F3 `--dry-run` も断る | 一致 | 3 か所とも dry-run の枝より前にある(`spawn.go:493`、`verbs.go:314`)。実機でも `--dry-run` が断られた |
| F4 宛先を問わず TASK を断る | 一致 | `Send` は `p.To` を見ずに `msg.Type` だけで決める(`verbs.go:303`) |
| F5 raw は `Parse` の 1 行目で読む | 一致 | `Parse` が `ErrMissingType` を返すのは 1 行目が `TYPE:` でないときだけである(`protocol.go:103-106`)。小文字の `type: task` は key が `TYPE` と一致しないので TASK ではない(実機で rc=0 を確認) |
| F6 子の作業ディレクトリ、`--config`、環境 | 一致 | AC6 の行を参照。start と `--detach` は同じ `orgWatchChild` を使う |
| F7 受け入れ条件の読み替え | 一致 | kill された見張りは `not_running` になり、start は見張りを起動して進む(実機で確認)。断るのは 10 秒たっても stale のときだけである。仕様の注(`docs/specs/2026-10-07-org-multi-org-director.md:64`)にこの読み替えが書いてある。PR の本文と最後の報告に書くのは /pr の役目で、まだ確かめられない |
| F8 tech-debt 201・204 (e)・205・206 を直す | 一致 | 201 は AC7、204 (e) は AC9、205 は AC6、206 は AC5 の行を参照。tech-debt の行は閉じてある(`docs/tech-debt/README.md:201-202`、`:205`、`:206-209`) |
| F9 stopping と `--once` の印 | 一致 | AC13 と AC14 の行を参照 |
| F10 stale の待ち(spawn と TASK は 2 秒、start と `--detach` は 10 秒) | 一致 | AC15 の行を参照 |

### 仕様 FR-6 との対応

| 仕様の項目 | 判定 | 根拠 |
| --- | --- | --- |
| 印が古いとき、start・spawn・実装の TASK の送信を断る | 満たす | AC1、AC2、AC4。「実装の TASK」を TYPE が TASK のすべての送信と読むことは、仕様の注(`:64`)に理由とともに書いてある(F4) |
| start は見張りが動いていなければ起動する | 満たす | AC3 |
| 横断の stop は見張りなしで使える | 満たす | AC8 |
| 雛形と現セッション昇格の手順の書き換え | 満たす | AC11。昇格の手順は `.claude/skills/org/SKILL.md:681-683` が `ralph org watch --all --detach` を先に打つと書く |
| 受け入れ条件 107 行(kill のあとの start) | 読み替えで満たす | F7 の行。kill -9 のあとの start が見張りを起動して進むことを実機で確かめた。ロックを持ったまま印の古い台帳では、`--detach` が 10 秒待ってから断ることを実機で確かめた(start は同じ経路の `ensureOrgWatch` を使う) |
| 「6 段目の後半で決めたこと」の注 | コードと一致 | 注の各文(2 秒と 10 秒の待ち、stopping と `once`、印の `deadman_minutes`、raw の読み方、入力の検査で断られる start は見張りを起動しないこと、それより後の検査で断られた start は見張りを残すこと)は、上の各行のコードと合う。後の検査で断られた start が見張りを残すことは、実機(未知の model で断られた start のあとも見張りが running)で確かめた |

### 計画からのずれ(Progress checklist の記録)の判断

1. S1: `ralph status` の stopping と `(--once)` の表示を S2 から S1 に移した。スライスの境目だけの話で、問題はない
2. S1: StartFeature の検査を `checkSpawnInput` の直後(`Manifest.Read` と `spawnPrecheckErr` の前)に置いた。F2 の「worktree の前」を満たし、Spawn と同じ順になる。max_orgs と見張りなしが重なれば見張りのエラーが先に出るが、どちらも記録を書かない拒否なので害はない。妥当
3. S1: 検査が打ち直しの冪等の戻りより前にあるので、立っている座席の打ち直しも見張りなしでは断られる。F2 の位置からの帰結で、何も書かない拒否なので状態は壊れない。妥当
4. S1 と S3: 期待を変えた 6a のテスト(後始末の間の印、`--once` の probe、AC5 の終了コード、stale の待ちの短縮)は、AC10 が許す範囲に入る
5. S2: spawn と `start <task>` が台帳を 1 回だけ解決するようにし、`newOrgSpawnRuntime` を消した。挙動は変わらない。ただし tech-debt の 183 行と 205 行の文が古くなった(D1、D2)
6. S3: エンベロープ・max_orgs・予約の検査と打ち直しの戻りで断られる start でも、起動した見張りは残る。見張りは止めるまで動くという前半の決定と合い、仕様の注と skill に書いてある。妥当。`PWD` の書き換えも正しい(AC6)
7. 記録にない小さなずれが 2 つある。どちらも妥当で、直す必要はない。(a) `ensureOrgWatch` は stale を待つ間にロックを持つ見張りが終わって `not_running` になると、そのまま起動に進む(`org_watch.go:428-446`。関数の doc には書いてある)。計画と図 2 は「stale なら起動せず待ち、running にならなければ断る」としか言わない。持ち主が消えたら起動するのは not_running の規則と同じ扱いで、合理的である。skill の文が少しずれる(D6、任意)。この分岐を固定するテストはない(Coverage gaps)。(b) AC10 が並べた期待の変更のほかに、bda15444 で 6a のテスト `TestOrgWatchDetach_Stale_StartsNothingAndSaysHowToStopIt` の期待の文言が変わった。Progress の N2 の記録で理由は分かる
8. 7d2e4bbb(CLI の send が `org: send:` を重ねていたのを直した): この PR より前からある包みの直しで、計画の範囲の外にある。変わるのは二重だった `org: send: org: send: …` が 1 つになることだけで、文書もテストもその二重の文を前提にしていない(grep で確認)。Progress に記録がある。妥当

### bda15444 の確認

- N1(テストのコメントの空白): コメントだけ
- N2(`--detach` の拒否の文): 3 通りの文とも「so nothing was started」を「so no new watch runs」に変え、関数の doc に「起動した子がロックを断られて終わったあとも同じ文」と足した(`org_watch.go:535-551`)。文は子を起動した場合にも正しい。古い文言はコードにも文書にも残っていない(`git grep 'nothing was started'` はこの PR の文書とコードで 0 件)。期待を変えたテストは 3 本(`org_watch_test.go` の stale、子が終わったときの 3 状態、stopping と `--once`)
- N3(dry-run の追記のエラー): `verbs.go:314-322` が追記の失敗を `org: send: ` で包むようになった。`Send` のすべての戻り口を読み、どのエラーも `org: send: ` で始まることを確かめた(`:300`、`:305`(`taskNotSentError`)、`:319`、`:326`、`:329`、`:332`、`:357`、`:383`、`:396`、`:407`、`:418`、`:444`)。CLI は `result.Err` をそのまま返す(`org.go:953-954`)ので、二重にも欠けにもならない。`rt.Send(` を呼ぶのは send のコマンドだけである(`org.go:902`)。`%w` で包むので `errors.Is` の鎖も保たれる。新しい org 層のテスト `TestOrgSend_DryRun_AppendFails_ErrorStartsWithTheSendPrefix`(`verbs_test.go:965`)が接頭辞が 1 回であることと EISDIR を固定する
- 新しい問題は見つからなかった。気づいたのは名前だけである。CLI のテスト `TestOrgSend_AnErrorWithoutTheSendPrefixGetsItOnce`(`org_watch_gate_test.go:349`)は、CLI が接頭辞を足さなくなったあとも「GetsItOnce」の名前のまま残っている。コメントは直してあり、期待も正しいので、直さなくてよい

## Static analysis

| Command | Result | Notes |
| --- | --- | --- |
| `./scripts/run-static-verify.sh`(changed scope) | rc 0 | golang の verifier は `gofmt: ok`、`go vet` は出力なし、`golangci-lint` は `0 issues.`、staticcheck は出力なし。shellcheck、hook の `sh -n`、settings の `jq -e`、Codex の guard、`check-pipeline-sync.sh`、`check-template-purity.sh`、tech-debt の plan 参照の検査、branch の secret scan(dc41c4f5..9823d97d、clean)も通った |
| `./scripts/check-sync.sh` | rc 0 | IDENTICAL 167、DRIFTED 0、KNOWN_DIFF 5(`docs/quality/quality-gates.md` を含む。この PR は両面に同じ行を足した) |
| `./scripts/check-skill-sync.sh` | rc 0 | 13 skill(s) in lock-step |
| `cmp` で mirror の一致 | 一致 | `/org` skill の 4 面、`agent-messaging.md` の 2 面 |
| `./scripts/plan-visual.sh digest <plan>` | de5ca6b7e6f0 | `- Approved:` 行と一致 |
| `go build ./cmd/ralph`(一時の場所へ) | 成功 | `ralph org start/spawn/send/watch --help` と `ralph status --help` の文を読むためと、実機確認のため |

## Documentation drift

| Doc / contract | In sync? | Notes |
| --- | --- | --- |
| D1 `docs/tech-debt/README.md:183`(self-review L5 (a)) | いいえ | F-7 の文が、この PR で消えた `newOrgSpawnRuntime` を挙げている(「`newOrgSpawnRuntime` (`internal/cli/org.go`) copies the first two steps of `newOrgRuntime`」)。写しは今、spawn・`start <task>`・`start --plan` の 3 か所の `org.ResolveOrgStateDir` の呼び出し(`internal/cli/org.go:549`、`:725`、`:822`。self-review の 550・726・823 から 1 行ずつずれている)と `newOrgSpawnRuntimeAt` の guard にある。文をこの形に書き直す。Trigger はまだ起きていないので、行は開いたままでよい |
| D2 `docs/tech-debt/README.md:205`(L5 (b)) | いいえ | (b) が `newOrgSpawnRuntimeAt` の呼び手を「(spawn and `start --plan`)」と書く。今は `start <task>` も呼ぶので、呼び手に足す |
| D3 `docs/tech-debt/README.md:202`(L5 (c)) | いいえ | 閉じた行の Impact の欄に「still open: a cycle that takes longer than the deadman of `./ralph.toml` but not the watch's own」が残る。印が見張り自身の deadman を持つようになったので、この残りも閉じている。取り消し線を引くか、RESOLVED の注に「この残りも閉じた」と足す |
| D4 `README.md:256`(L6) | いいえ | 「Either way the watch runs detached in the ledger's main worktree root, so without `--config` it reads that root's `ralph.toml`.」に、台帳に main worktree がないとき(repo の外を指す `--state-dir`)は打った場所で動く、という条件がない。「in the ledger's main worktree root (when the ledger has one)」のように足す。help(`internal/cli/org.go:693-694`、`internal/cli/org_watch.go:63`)と skill はこの条件を書いている |
| D5 `README.md:125`(新しく見つけた) | いいえ | `ralph status` の行が `org watch:` の行を「running, stale, or not running」と書いている。この PR で `stopping` と `running (--once)` が増え、`--json` には `once` が足された(`internal/cli/status.go:172-178`、`:434`)。「running (with `(--once)` for a `--once` watch), stopping, stale, or not running」のように直す。同じ README の 256 行は stopping を書いている |
| D6 `.claude/skills/org/SKILL.md:176`、`:177`、`:585`(任意) | ほぼ合う | 「ロックを持つ見張りが stale なら、起動せず最大 10 秒 running になるのを待ち、ならなければ断る」は、待つ間に持ち主が終わった(`not_running` になった)ときに起動へ進む動き(`org_watch.go:428-446`)を書いていない。直すなら 4 面とも同じに直し、`scripts/sync-skills.sh` で揃える。直さなくても誤った操作には導かない |
| D7 `internal/cli/org_watch.go:558-559`(任意、コードのコメント) | ほぼ合う | `runOrgWatchStop` の doc の「running and stale are handled alike」に stopping がない。`--stop` は `not_running` 以外をすべて同じに扱う(`:563-566`)。「running, stopping, and stale」にする |
| `/org` skill の 4 面 | 合う | `spawn`・`send`・`watch`・`start` の行(`:168`、`:169`、`:176`、`:177`)、`start` が拒否するもの(`:584-588`)、(A) 現セッション昇格(`:681-683`)、(B) headless(`:688-689`)、長時間運用の項(`:780-814`)が新しい動きと合う。4 面はバイト一致 |
| `README.md:235`、`:246` | 合う | 例の先頭に `ralph org watch --all --detach` を足し、spawn と TASK が断られることと start が起動することを書いた |
| `AGENTS.md:90` | 合う | repo map に `watchgate.go` を足した。「refuses spawn, start, and TASK sends」の start は、起動できないときに断る、の意味で読める |
| 仕様の注 `docs/specs/2026-10-07-org-multi-org-director.md:64` | 合う | F4 と F7 の読み替えを含め、上の「仕様 FR-6 との対応」のとおりコードと合う。前半の注(`:63`)の 3 状態の説明には「後半で stopping に変えた」の括弧書きが付いている。仕様のチェックボックスは FR-1〜FR-12 のどれにも付いていないので、FR-6 が未チェックなのは揃っている |
| `.claude/rules/ralph/agent-messaging.md:96-100`(と template) | 合う | TASK の拒否と raw の読み方。2 面はバイト一致 |
| `docs/quality/quality-gates.md:80`(と template) | 合う | 「Cross-org watch gate」の行。両面に同じ行がある |
| tech-debt の 201・204 (e)・205・206 と新しい行 | 合う(D1〜D3 を除く) | 201 は `:201-202`、205 は `:206-207`、206 は `:208-209` で閉じ、204 (e) は `:205` の中で取り消し線と RESOLVED の注が付いた。見張りが途中で落ちたときの穴は `:210` に新しい行がある。`:210` は計画を `docs/plans/active/` のパスで指すが、/pr の `scripts/archive-plan.sh` が `docs/tech-debt/README.md` のこのリンクを書き換えるので、ずれではない |
| `ralph org start/spawn/send/watch --help`、`ralph status --help` | 合う | ブランチのビルドで読んだ。start は起動と 10 秒の待ちと拒否の 3 状態、spawn と send は `--dry-run` も断ること、watch は `--detach` の作業ディレクトリ・環境・終了コード、status は stopping と `(--once)` を書く。`TestOrgStartSpawnSendHelp_SayWhatTheWatchGateDoes`(`org_start_watch_test.go:499`)が固定する |
| leader の雛形 `internal/org/prompts/leader.md:236-248` | 合う | AC11 の行 |

D1〜D5 は /sync-docs で直す。D6 と D7 は任意。

## Observational checks

ブランチのビルドで、repo の外の一時の台帳(`--state-dir`)を使って確かめた。出力は evidence の log にある。

- 見張りなし: `spawn --dry-run`、型付きの TASK と raw の `  TYPE:  TASK` の `send --dry-run` は rc=1。エラーは `--state-dir` を付けた `ralph org watch --all --detach` と、新しい org なら `ralph org start` を案内した。raw の `type: task` と QUESTION は rc=0。manifest には通った 2 件の記録だけが残った
- `watch --all --detach` は `started … (pid N)`、2 回目は `already running (pid N)` で、どちらも rc=0。印は `deadman_minutes: 10` を持つ。そのあとの spawn と TASK の `--dry-run` は通った
- 動いている見張りの台帳で、2 つ目の `watch --all --once` は rc=1、stdout は 0 バイト、stderr は相手の pid を言った
- `--stop` は `stopped ralph org watch --all (pid N)` で、印は stopped
- `--detach` で起動した見張りを kill -9 すると、`ralph status` は `org watch: not running (a running mark is left from pid N, …)` になり、spawn は not_running の文で断られた。続く `start <task>` は新しい見張りを起動して stderr に pid と watch.log を出し、そのあと envelope(未知の model)で断られた。見張りは start のあとも running のまま残り、2 回目の start は何も起動しなかった
- python の flock でロックを持ち、1 時間前の印を置いた台帳では、spawn が 2 秒、`--detach` が 10 秒待ってから stale の文で rc=1 になった
- `--once` の見張りの最後の印には `once: true` があった
- 終わったあと、一時の ralph のプロセスは残っていない(`pgrep` で確認)

気づいたこと(コードの文言、LOW。直すかは任せる): `--detach` の拒否の文は、`--state-dir` を明示しても、案内するコマンド(stale の `ralph org watch --stop`、stopping と `--once` の `run ralph org watch --all --detach again`)に `--state-dir` を付けない(`org_watch.go:541-551`)。検査の文(`WatchGateError`)は付ける。stale の文は 6a からこの形で、stopping と `--once` の文はこの PR で同じ形に足された。実機の stale の拒否でも、spawn の文には付き、`--detach` の文には付かなかった

## Coverage gaps

- テストは走らせていない(/test の担当)。上の判定は、テストがあり、読んだ範囲で期待が AC と合う、という意味である
- `ensureOrgWatch` が stale を待つ間に持ち主が終わって起動へ進む分岐(`org_watch.go:428-446`)を固定するテストはない。この分岐を「断る」に変える mutant は、今のテストでは落ちない見込みである(読んだだけで、走らせてはいない)
- 本物の見張りが後始末(stopping)をしている間に spawn・TASK・start を打つテストはない。印の書き手(`watch_all_test.go:1222`)と、stopping の印に対する検査(cli のテスト)を別々に固定している
- 実機で見ていないもの: 後始末の間と `--once` が動いている間の拒否(窓が短く再現できなかった)、start が stale の台帳で断ること(`--detach` で同じ経路を見た)、`start --plan` の起動、並行の `--detach` と start、main worktree の中の台帳での子の作業ディレクトリ、Linux での動き
- 前の版の見張り(印に `once` も `deadman_minutes` もない)と台帳を共有したときの動きは見ていない。計画の Risks と Rollout のとおり、release のノートで扱う
- 計画の Verify plan の「実機確認を `docs/evidence/` に」は、上の確認を evidence の log(gitignore の対象)に残しただけで、コミットする記録は作っていない
- 計画の受け入れ条件のチェックボックス(AC1〜AC15)はまだ付いていない。前の PR と同じく、検証のあとに付ける段取りと読んだ

## Verdict

pass

- Verified: AC1〜AC11 と AC13〜AC15 は、コードとテストの読み合わせで満たす。F1〜F10 はコードと一致し、仕様 FR-6 の残りの 4 項目と受け入れ条件 107 行(F7 の読み替え)を満たす。Progress に記録されたずれと、記録のない小さなずれ 2 つはどれも妥当である。bda15444 は文言とエラーの包み方だけで、新しい問題はない。静的解析(changed scope)、`check-sync.sh`、`check-skill-sync.sh` は rc 0。見張りなしの拒否、`--detach` の起動と `already running`、banner を出さない 2 つ目の見張り、kill のあとの start の起動、stale の待ちと拒否、印の `deadman_minutes` と `once` は実機でも確かめた
- Partially verified: AC12(D1〜D5 の文書が古い。/sync-docs で直す。D6 と D7 は任意)、AC8(座席の `wait` と `inbox show/ack/resolve/notify` は配線がないことを grep で確かめただけ)
- Not verified: テストの実行、後始末の間と `--once` の間の拒否の実機での確認、start の stale の拒否と `start --plan` の起動の実機での確認、並行の起動、Linux
