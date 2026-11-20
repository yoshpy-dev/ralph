# Verify report: org-reseat-restart-herdr

- Date: 2026-11-20(cycle 1。`cycle-count.json` は `cycle: 1`)
- Plan: `docs/plans/active/2026-11-20-org-reseat-restart-herdr.md`。`./scripts/plan-visual.sh digest` の値 `f5d55e0b6ca2` は、計画の `- Approved:` の行と一致する
- Spec: `docs/specs/2026-11-20-org-reseat-restart-herdr.md`(FR-1〜FR-8、FR-5b、NFR-1〜NFR-3、AC1〜AC8)
- Verifier: verifier サブエージェント(`/verify`)
- Scope: `git diff a672b72a...HEAD`(base は `origin/main` の `a672b72a`、HEAD `02402849`、31 ファイル、+4290 −127)。self-review の run 2 のあとの直し `f620a6e8`(L5: `ps eww` を UTF-8 で読む、L7: 不一致の note の文)と、run 2 が読んだ `53c12663`(M1、L1〜L4)も、静的検査と spec の確かめの対象に含めた。最後の pane の確かめと `herdr server stop` のあいだの窓は、計画の Design decisions 3 のとおり意図して残したものなので、指摘にはせず、文書どうしが同じことを書いているかだけを見た
- Evidence: `docs/evidence/verify-2026-11-20-org-reseat-restart-herdr.log`(`.gitignore` の `docs/evidence/*.log` に当たり、commit されない)。静的検査の出力全体、skill の 4 写しの `cmp`、変えた Go ファイルの base と HEAD の行数、register の Trigger の列の走査が入っている。ラッパー自身のログは `docs/evidence/verify-2026-11-20-095415.log`

## Spec compliance

テストは走らせていない(`/test` の担当)。コードを読み、各 AC を固定するテストがあることまでを確かめた。

| Acceptance criterion | Status | Evidence |
| --- | --- | --- |
| AC1: 座席の pane だけのサーバーで `--restart-herdr` が、座席を止め、`herdr server stop`、running でなくなったあとゲートで起動、座席を立て、`reseated` を記録する(呼ぶ順をテストで固定) | Met | `Reseat` は `r.stopAll() && r.restartHerdr()` のあとに `startAll`(`internal/org/reseat.go:1389`)。`restartHerdr`(`:1495`)は 2 回目の pane の一覧、`ServerStop`、`waitHerdrStopped`(`:1548`、10 秒、100 ms ごと)、`checkHerdrServerGate` の順で、一覧と stop のあいだに呼び出しはない(`r.attempted = true` の代入だけ)。順は `TestReseat_RestartHerdr_StopsTheServerBetweenTheStopsAndTheStarts`(`internal/org/reseat_restart_test.go:218`、stop の直前の呼び出しが `pane_list_all` であることも見る)と CLI の `TestOrgReseat_RestartHerdrStopsTheServerBetweenTheStopsAndTheStarts`(`internal/cli/org_reseat_restart_test.go:106`)が固定する |
| AC1b: `--seat` で一部だけなら何もせずに断る | Met | `reseatRestartHerdrErr`(`reseat.go:649`)が `PlanReseat` の 9 番で、active で targets に入らない座席を名指して断る。`TestPlanReseat_RestartHerdr_RefusesASubsetOfTheActiveSeats`(`:255`)と CLI の `TestOrgReseat_RestartHerdrRefusals`(`:188`) |
| AC2: ほかの pane があれば何も止めずに断り、id・label・cwd と、記録だけの pane の見分け方を出す(終了コード 1) | Met | `reseatFirstPanesErr`(`reseat.go:730`)、`reseatOtherPanesText`(`:746`)、`reseatOtherPanesHint`(`:769`、`herdr pane process-info` と `herdr pane close`)。CLI の `TestOrgReseat_RestartHerdrRefusesAnotherPane`(`:156`)が文を全部固定し、stdout が空、close も stop もない、manifest が変わらないことを見る。終了コードは error を返すことで 1 になる(実行での確かめは `/test`) |
| AC3: `HERDR_PANE_ID` があれば断る | Met | `reseatRestartHerdrErr` の最初の枝。`TestPlanReseat_RestartHerdr_RefusedInAHerdrPane`(`:339`)と CLI の refusals の表 |
| AC4: 座席を止めたあとにほかの pane ができていれば、サーバーを止めずに終わり、打ち直しを出す | Met(注 V1) | `restartHerdr` の 1 番。`TestReseat_RestartHerdr_APaneMadeWhileTheSeatsStopKeepsTheServer`(`:458`)が、server stop なし、打ち直しに `--restart-herdr` と `--claude-config-dir` があることを見る。同じテストが `reseated` の記録も固定する(`:487`)。仕様 FR-3 の括弧書きはこれと逆のことを書く(V1) |
| AC4b: サーバーを起動し直したあと leader だけが立てば、打ち直しは `--restart-herdr` を持たず `--claude-config-dir` を持ち、打つと残りだけを立てる | Met | `finish`(`reseat.go:1669`)が `restartHerdr = RestartsHerdr && !HerdrStopped` で打ち直しを作る。`TestReseat_RestartHerdr_TheRetryAfterTheServerStartedLeavesOutRestartHerdr`(`:496`)は打ち直しを実際に打ち、status も pane の一覧も読まず、何も止めずに 2 座席を立てることを見る。CLI 側は `TestOrgReseat_RestartHerdrRetryAfterTheServerStarted`(`:457`) |
| AC5: `--claude-config-dir /x` でゲートの起動するサーバーの環境が `/x`、`''` ならなし、存在しないディレクトリは断る、打ち直しにも写る | Met | `orgHerdrServerEnvOverride.apply`(`internal/cli/org_herdr_server_env.go:53`)、`checkOrgClaudeConfigDir`(`:74`、`runOrgReseat` の最初で読む、`internal/cli/org_reseat.go:279`)、`reseatRetryHerdrFlags`(`reseat.go:1867`)。テストは `TestOrgHerdrServerEnvOverride_ApplySetsReplacesOrRemovesClaudeConfigDir`、`TestOrgHerdrServerGate_StartsTheServerWithTheClaudeConfigDirOverride`、`TestOrgReseat_ClaudeConfigDirIsCheckedFirst`、`TestOrgReseat_ClaudeConfigDirReachesTheHerdrServerGate`(起動したサーバーの env の `CLAUDE_CONFIG_DIR` の項目を数える)、`TestReseat_RestartHerdr_AGateThatFailsAfterTheStop`(`''` を写した打ち直しを打つ) |
| AC6: サーバーの環境が読めれば `CLAUDE_CONFIG_DIR` を出し、違えば知らせ、読めなければ理由を出して断らない。pid が 2 つ以上なら herdr に絞る | Met | `readOrgHerdrServerClaudeConfigDir`(`org_herdr_server_env.go:170`)、`orgHerdrServerPID`(`:231`)、`printOrgReseatHerdrServerNotes`(`org_reseat.go:538`)。読みの表テスト 12 本(`org_herdr_server_env_test.go:374`〜`:727`)と、stderr 全体を固定する `TestOrgReseat_HerdrServerLineAndNotes`(`org_reseat_restart_test.go:356`) |
| AC7: `--restart-herdr` なしの今の reseat のテストが変わらずに通る | Met(注) | 既存の reseat のテストで変わったのは `reseat_test.go:1999` の 1 行だけで、`53c12663`(self-review L1)が消した `reseatRetryCommand` の呼び出しを `reseatRetryCommandWithHerdr(…, reseatRetryHerdr{}, …)` に替えた。期待の文字列は変わっていない。CLI の TestMain はサーバーの環境の読みを nil にし(`internal/cli/main_test.go`)、stderr 全体を固定する既存のテストには新しい行が出ない。これは計画の S2 のずれ (1) に記録してある。`TestReseat_WithoutRestartHerdrTheServerIsLeftAlone`(`reseat_restart_test.go:708`)が、flag なしではサーバーに触れないことを見る |
| AC8: 文書(help、director と org の skill の 4 写し、README、reseat の仕様、AGENTS.md)が新しい flag と確かめを書く。`check-sync.sh` と `check-skill-sync.sh` が通る | Met(注 V2、V3) | help(`orgReseatHelp` の 3 段落と 2 つの flag の文)、director skill の「座席を立て直す」3 と「してはいけないこと」、org skill の「座席の立て直し」、README の 304 行、`docs/specs/2026-11-12-org-reseat.md` の D6 の追記、AGENTS.md の `reseat.go` の句がある。skill の 4 写しは `cmp` で同じ。`check-sync.sh`(DRIFTED 0)と `check-skill-sync.sh`(14 skill)は静的検査の中で通った。`/org` skill のコマンド表の `reseat` の行(V2)と、herdr のサーバーのゲートを説明する文(V3)は直っていない |

仕様の要件の確かめ:

| 要件 | Status | Evidence |
| --- | --- | --- |
| FR-1(`PlanReseat` のあと、`reseat.lock` の中で、止めたあと立てる前にサーバーを止めて起動し直す。起動はゲート) | Met | 順のずれ(確かめを 9、ゲートを 10)は FR-1 の 2026-11-20 の注に書いてあり、`PlanReseat` の doc(`reseat.go:425-434`)と合う。起動は `ensureOrgHerdrServerWithEnv`(`ensureOrgHerdrServer` に env を足した形) |
| FR-2(4 つの断り、TUI の 1 行) | Met | HERDR_PANE_ID、ほかの pane(id・label・cwd、記録だけの pane の案内)、一覧が読めない、サーバーが動いていない(案内つき)。TUI の行は `orgReseatTUINote`(`org_reseat.go:522`、stderr)。本当の reseat では 1 回目の pane の確かめが 10 番のゲートのあとに来るので、そこで断ると見張りは起動済みのことがある。`PlanReseat` の doc がそう書き(self-review L2 の直し)、help と README の「nothing stopped」は正しい |
| FR-3(止める直前の 2 回目の確かめ) | Met(注 V1) | 括弧書きの「`reseated` は記録しない」とコードが違う |
| FR-4(10 秒待ち、過ぎたら「止まったかもしれない」) | Met | `waitHerdrStopped` の文 `the server may still stop`。`TestReseat_RestartHerdr_AServerThatDoesNotStop`(`:539`) |
| FR-5、FR-5b | Met | AC5 の行 |
| FR-6 | Met | AC6 の行。dry run でも `printOrgReseatHerdrServerNotes` を `rt.Reseat` の前に呼ぶ |
| FR-7(lsof、ps eww、pid の絞り、次の `KEY=` まで、Linux は /proc) | Met | `orgPSEnvKey` は ` [^ =]+=`(M1 の直し)。`ps eww` は `LC_ALL=en_US.UTF-8`、lsof と `ps -o comm=` は `LC_ALL=C`(`orgHerdrServerEnvCommandEnv`、`f620a6e8`)。Linux の経路は fake の `readFile` でだけ確かめてある |
| FR-8 | Met | AC8 の行 |
| NFR-1 | Met | AC7 の行。flag なしの違いは stderr の 1〜2 行だけ |
| NFR-2 | Met | 一覧が読めない、ほかの pane がある、のどちらでも `ServerStop` を呼ばない |
| NFR-3 | Met | `PaneListAll` と `ServerStop` は `callWithTimeout`(driverCallTimeout)、status は `readOrgHerdrServerStatusWithin(orgHerdrStatusTimeout)`、lsof と ps は `orgHerdrServerEnvCallTimeout`(5 秒)と `WaitDelay` 1 秒 |

## Static analysis

| Command | Result | Notes |
| --- | --- | --- |
| `./scripts/run-static-verify.sh`(changed scope) | Pass(rc=0) | shellcheck と `sh -n`(hook と verify のスクリプト)、guard の awk、`jq -e`、Codex の hook の guard 3 つ、`check-sync.sh`(IDENTICAL 172、DRIFTED 0、ROOT_ONLY 0、KNOWN_DIFF 6)、`check-pipeline-sync.sh`、`check-skill-sync.sh`(14 skill)、`check-template-purity.sh`、tech-debt の計画の参照、Go の verifier(言語の範囲は changed で golang だけ。`gofmt` は ok、`go vet ./...` は出力なし、golangci-lint は `0 issues.`、staticcheck は出力なし)、branch の secret scan(`a672b72a..02402849` は clean)。`All verifiers passed.` |
| `./scripts/plan-visual.sh digest docs/plans/active/2026-11-20-org-reseat-restart-herdr.md` | `f5d55e0b6ca2` | 計画の `- Approved:` と一致 |
| `cmp .claude/skills/{director,org}/SKILL.md` と 3 つの写し | 6 組とも同じ | `.agents/skills/`、`templates/base/.claude/skills/`、`templates/base/.agents/skills/` |

## Documentation drift

| Doc / contract | In sync? | Notes |
| --- | --- | --- |
| V1: 仕様 FR-3 の括弧書き(`docs/specs/2026-11-20-org-reseat-restart-herdr.md:39`) | No(LOW、`/sync-docs` へ) | FR-3 は、2 回目の確かめで断ったときを「今の reseat の失敗の扱い(座席の記録は stopped、`reseated` は記録しない)」と書く。コードは座席を止めたあとなので `r.attempted` が立ち、`finish`(`reseat.go:1670`)が `reseated` を記録する。AC4 のテストもそれを固定する(`reseat_restart_test.go:487`)。コードは前の仕様の FR-8(`docs/specs/2026-11-12-org-reseat.md:94`、「途中で失敗したときも書く」)と README の「a reseat that tried to stop or start a seat ends with one `reseated` event」に従っていて、FR-3 の「今の reseat の失敗の扱い」とも合う。食い違うのは括弧の中だけなので、FR-1 と同じ形の日付つきの注を FR-3 に足す。計画の Progress の S2 のずれの一覧にもこの点はない |
| V2: `/org` skill のコマンド表の `reseat` の行(`.claude/skills/org/SKILL.md:385`、4 写し) | No(LOW、`/sync-docs` へ) | 行は `--seat`、`--driver`、`--model`、`--dry-run`、`--timeout-ms` を並べ、「本当の reseat は … herdr のサーバーを確かめ、動いていなければ起動してから止める」と書くが、`--restart-herdr` と `--claude-config-dir` がない。節の「座席の立て直し」(`:1843-1891`)には書いてある。直すと register の「Two documentation findings of org-stop-all」(a)(コマンド表の次の編集)が当たるので、その行にも注が要る |
| V3: herdr のサーバーのゲートを説明する文 | No(LOW、`/sync-docs` へ) | `docs/quality/quality-gates.md:99` と `templates/base/docs/quality/quality-gates.md:98` の「Herdr server gate」の行は、サーバーの環境を「pane の変数と Claude Code のセッションの変数を除いたもの」とだけ書き、失敗の列で「no seat stopped for `reseat`」と書く。`--restart-herdr` でサーバーを止めたあとのゲートが失敗すると、座席は止まったまま残る(`TestReseat_RestartHerdr_AGateThatFailsAfterTheStop`)。`/org` skill の「前提」(`.claude/skills/org/SKILL.md:27-31`)の環境の文にも `--claude-config-dir` の例外がない(`orgHerdrServerHelp` の文は `f620a6e8` の前の S1 で足してある)。template の quality-gates を直すと、register の 118 行(template の quality-gates に触れるたびに当たる)にも注が要る |
| V4: `TestHerdr_ServerStop` の doc(`internal/org/driver/herdr_test.go:2457`) | No(LOW) | 「since the reply of a stop was not captured live」は、`47a2235d` が実機の応答(標準出力なし)を `docs/evidence/org-reseat-restart-herdr-live-2026-11-20.md` に記録し、`ServerStop` の doc(`internal/org/driver/herdr.go:996-1005`)をその記録に替えたあとも残っている。テストの `""` の場合が実機の形だと書き直す |
| V5: README の `ralph org reseat` の段落(`README.md:304`) | No(LOW) | 「refuses, with nothing stopped, when … the server holds any pane but the org's active seats (…; the panes are listed again right before the stop)」は、2 回目の一覧で断ったときも何も止めていないと読める。実際は座席が止まったまま、サーバーは動いたままで終わる。help(「a pane that appeared while it stopped the seats ends the reseat there, with the seats stopped and the server running」)と skill は正しい |
| V6: org runtime の 2 つの仕様 | No(LOW) | `docs/specs/2026-08-01-org-runtime.md` の 2026-11-12 改訂 (d)(`:84`)は、アカウントを移す手順を人がサーバーを止める道だけで書く。この PR の分の改訂(2026-11-20)がない。2026-11-15 改訂が (d) に印を付けた形が前例。`docs/specs/2026-10-07-org-multi-org-director.md` には、director が人に頼まれたときに herdr のサーバーを止める reseat を打ってよい、という director の権限の広がりの日付つきの注がない(#318 などの前例。ただし #312c の reseat 自体もこの仕様に注を足していない) |
| V7: register(`docs/tech-debt/README.md`)の Trigger | No(LOW、`/sync-docs` へ) | 下の「register の Trigger の走査」。当たった行に、この branch の注が 1 つもない |
| AGENTS.md の `internal/org/` の行(`reseat.go` の句) | Yes | 「`--restart-herdr` stops the herdr server between stopping and starting the seats once `herdr pane list` shows no pane but the org's seats, and `--claude-config-dir` sets the started server's `CLAUDE_CONFIG_DIR`」はコードと合う |
| `ralph org reseat --help` と flag の文 | Yes | 3 つの段落の各文を `Reseat`、`restartHerdr`、`finish`、`printOrgReseatHerdrServerNotes` に当てた。偽の文はなかった |
| `/director` skill の「座席を立て直す」3 と「してはいけないこと」 | Yes | 打ってよいのは人が頼んだときだけ、dir は人に聞く、断りの 4 つ、2 回目の確かめ、10 秒、`stopped: herdr server` の有無で打ち直しの flag が変わること、窓、人が止める道。どれもコードと合う。step 1 の「人が頼んだときだけ」と例外の文も矛盾しない |
| `docs/specs/2026-11-12-org-reseat.md` の D6 の追記 | Yes | (1) と (3) を満たし (2) の keychain が残る、は計画の Non-goals と合う |
| I1: 窓の書き方 | INFO | register の新しい行(539 行)、両 skill、計画は、窓が残ること、消えうるもの、herdr 0.7.5 に条件つきの停止がないことを同じように書く。違いは 2 点。計画の Design decisions 3(`:59`)と Risks(`:101`)は窓を「1 秒未満」と書くが、register は「its length was not measured」と書き、長さを測った記録はない。register と skill は窓で生まれうる座席を「別の repository の ralph」の座席に限って書くが、`reseat.lock` を取るのは reseat だけなので、同じ repository の別の shell で人が打つ `start` や `spawn` も当たる。help と README は窓に触れない(計画は skill、Known gaps、register に書くと決めている) |
| I2: 計画のパスを指す新しいコメント | INFO | `internal/org/reseat_restart_test.go:13` と `internal/cli/org_reseat_restart_test.go:21` が `docs/plans/active/2026-11-20-org-reseat-restart-herdr.md` を指す。`/pr` が計画を archive へ移しても Go のコメントは書き換わらない。register の 518 行 (e) と同じ種類 |
| I3: 計画の文の古い関数名 | INFO | Scope(`:22`)は打ち直しの関数を `reseatRetryCommand` と書き、Progress の S2 のずれ (5) は「既存の関数 … は変えずに足した(`reseatRetryCommandWithHerdr`)」と書く。`53c12663` が `reseatRetryCommand` を消したので、どちらも今の形ではない。テストの名前 `TestReseatRetryCommand`(`reseat_test.go:1962`)も古い名前のまま |
| I4: チェックボックス | INFO | 計画の AC1〜AC8 と、仕様の FR、NFR、AC はすべて `[ ]`。前の仕様では実装の PR が印を付けた(`docs/specs/2026-11-12-org-reseat.md`) |
| I5: L6 の書き方 | INFO | 計画は L6 を「直さず、doc に書いた」と書く。書いてあるのは `orgPSEnvValue` の doc の「A word of the command with "=" in it counts as an entry too」だけで、`readOrgHerdrServerClaudeConfigDir` の doc(`org_herdr_server_env.go:162-165`)は「An environment with no entry at all cannot be read … which must not pass as unset」を条件なしで書く |

## register の Trigger の走査

`docs/tech-debt/README.md` の表の各行を ` | ` で 5 つに分け、4 つ目(Trigger)を、変えたファイルの名前と diff の関数の名前で照らし、当たった行を読んだ(走査の出力は evidence の log)。この branch が register に足したのは窓の新しい行(539 行)だけで、既存の行への注はない。

当たって払っていない行(V7、どれも LOW):

- 518 行(「Fourteen small findings of docs/spec-org-reseat」): (a)「the next change to `orgReseatHerdrServerProbe` or `ensureOrgHerdrServer`」に、`ensureOrgHerdrServer` を `ensureOrgHerdrServerWithEnv` の包みにした変更が当たる。(a) は status を読む関数を 1 つにまとめることを求めるが、この branch は status を読んで動いているかを返す 3 つ目の関数 `orgReseatRestartHerdrServerProbe`(`org_reseat.go:500`)と、`printOrgReseatHerdrServerNotes` の中の 4 つ目の読み(`:563`)を足した。(d) `PlanReseat` の変更(9 番の確かめ)、(f) と (j) と (m) の `orgReseatHelp` の変更、(l) の `printOrgReseatResult` の変更、(n) の `finish` と `ReseatResult` の doc の変更も当たる。(b)「the refusal text of `PlanReseat`」は、新しい断りの文を足したことが当たるかどうか判断が要る。どれも払っていない(help は `ralph org status --org-id <id>` を案内せず、L24 の「`""` also when the org was disbanded」も足していない)
- 519 行(`reseat.go` の大きさ): Trigger「The next change to `reseat.go`, `reseat_test.go`, …」が当たる。`reseat.go` は 1,574 → 1,933 行、`reseat_test.go` は 2,004 行のまま 1 行を変えた(L1)。`internal/cli/org_reseat.go` は 554 → 793 行で、800 行の目安まで 7 行。新しいファイル(`reseat_restart_test.go` 762、`org_reseat_restart_test.go` 580、`org_herdr_server_env.go` 384、`org_herdr_server_env_test.go` 749)はどれも目安の下。行数はどれも HEAD `02402849` の `wc -l`
- 520 行(「Test gaps of docs/spec-org-reseat」): (c)「`printOrgReseatResult` or the stop and start order of `Reseat`」、(h)「the reseat subsection of the `/org` skill, or … 「座席を立て直す」 of the `/director` skill」、(i) と (j)「`runOrgReseat`」、(m)「`printOrgReseatResult`」、(n)「`orgReseatHelp`」が当たる。(h) の skill の文と help を比べるテストは足していない。(n) は新しい文を `TestOrgReseatHelp_SaysHowToRestartTheHerdrServer` が文単位で固定し、前の文は 1 つの部分文字列のまま
- 441 行(`internal/org/driver/herdr.go` の大きさ): Trigger「the pane operations of `herdr.go`」に `PaneListAll` と `herdrPaneListResult` への切り出しが当たる。971 → 1,009 行。`herdr_test.go` は 2,397 → 2,543 行(この行と 324 行の 1 列目が数える)
- 181 行(org-stop-all の F-7): 「F-7 when the driver API is next touched」に `PaneListAll`、`ServerStop`、`HerdrPane.Cwd` が当たる。`driver.NewHerdrError` は今もテストのために export されている(新しい fake のメソッドはこれを呼ばない)
- 88 行 (2): 「Next time the corresponding file is touched」に `spawn_test.go`(+40 行、fake の `PaneListAll`、`ServerStop`、`paneCwds`)と `watch_test.go`(+9 行)が当たる。`filepath.Join(dir, "receipts.jsonl")` は `spawn_test.go:881` と `watch_test.go:367` に残る(2026-11-07 の前例が同じ当たり方を記録した)
- 308 行: 「The next change to `watch_test.go`」に +9 行が当たる。`newTestWatchRun` はまだ `os.MkdirTemp` を使う
- 440 行 (a): 「the next change to FR-A3 or `orgHerdrServerDropsEnv`」に、ゲートが起動するサーバーの環境の変更(`orgHerdrServerLaunchRequest` の `env.apply`)と `orgHerdrServerDropsEnv` の doc の変更が当たる。`CODEX_` の名前はそのまま残る
- 判断が要るもの: 185 行 (a)「the herdr `get` parsers」(`herdrPaneJSON` に `cwd` を足し、`PaneList` の読みを `herdrPaneListResult` に移した。新しい `TestHerdr_PaneListAll` には壊れた envelope の case がある)と、323 行「the next change to how ralph reaches the herdr server」(サーバーを止める呼び出しと、ソケットに lsof を当てる読みを足した)

`reseatRetryCommand` の名前: 518 行と 520 行の Trigger の列に、2026-11-13 の注の中で出てくる(「def8d2a3 and df8f674e changed `startAll`, `reseatRetryCommand`, …」)。どちらもそのときの変更の記録で、今の関数を指す文ではないので、そのままでよい。519 行の Trigger(`PlanReseat` を `reseat_plan.go` へ移す)は上のとおり当たっている。

当たらなかった行: 182 行(コマンド表は編集していない。V2 を直すと当たる)、521 行(`Disband` と `takeReseatLock` は変わっていない)、139 行、193 行、202 行 (d)(`HerdrClient` の型の doc ではなく、interface の中のメソッドの doc を足した)、267 行(編集した節は「受信箱の処理」の下にない)、362 行、366 行、369 行と 370 行(`spawn.go` 3,122 行は 3,059 + 300 の下、`spawn_test.go` 6,448 行は 6,406 + 300 の下)、403 行、438 行、509 行、524 行、530 行、532 行。

## Observational checks

- 実機の記録 `docs/evidence/org-reseat-restart-herdr-live-2026-11-20.md` を読んだ。分離した herdr 0.7.5 のサーバーで、`pane list`(workspace なし)の cwd、`lsof -t` が pid 1 つ、`ps eww` で `CLAUDE_CONFIG_DIR` が読めること、`herdr server stop` が標準出力なしで終わりサーバーが止まることを確かめてある。ドライバの `ServerStop` と `PaneListAll` が読む形と合う
- この verify では herdr も ralph の binary も動かしていない

## Coverage gaps

- 座席のある org での `ralph org reseat --restart-herdr` の通しは、実機で打っていない(evidence の「確かめていないこと」)。分離したサーバーで、shell の pane があるときに ralph が断ることと、止めたあとゲートがサーバーを起動し直すことも、ralph を通しては確かめていない(計画の Test plan の「実機の確かめ」のうち、部品だけが記録にある)
- 新しいアカウントの資格で座席が動くことは、2 つのアカウントを持つ人の手でしか確かめられない(仕様の Open questions)
- Linux の `/proc/<pid>/environ` の経路は fake の `readFile` だけ
- I6(INFO): `restartHerdr` は、`waitHerdrStopped` が止まったのを見てからゲートを呼ぶまでのあいだに、ほかのプロセス(別の repository の ralph の start など)がサーバーを起動すると、ゲートはそのサーバーで通り、`HerdrRestarted` が立って stdout に `started: herdr server` が出る。座席はそのサーバーの環境で立つ。ゲート自身の「started the herdr server」の stderr の行はそのとき出ないが、reseat はサーバーを起動し直したあとの `CLAUDE_CONFIG_DIR` を読まない。いちばん小さい確かめは、`started: herdr server` のあとに `herdr server:` の行をもう 1 度出すことで、人が切り替わったかを目で見られる
- I7(INFO): `internal/cli/org_reseat.go` は 793 行で、次の変更で 800 行の目安を越えそう。519 行は 1 列目でこのファイルを名指す
- 決定的な検査がない点: skill の reseat の文と help を比べるテストがない(520 行 (h) の求めるもの)。V2 のようなコマンド表の取り残しは、今のテストでは落ちない

## Verdict

- Verdict: pass
- Verified: 静的検査(`run-static-verify.sh` が rc=0、gofmt、go vet、golangci-lint、staticcheck、check-sync、check-skill-sync、check-template-purity、secret scan)、計画の承認の digest、skill の 4 写しの一致、AC1〜AC8 と FR-1〜FR-8、FR-5b、NFR-1〜NFR-3 をコードと、それを固定するテストの存在で確かめた
- Partially verified: AC4 と FR-3(V1: 仕様の括弧書きがコードと違う)、AC8(V2、V3: コマンド表とゲートの説明の文が古い)
- Not verified: テストの実行(`/test` の担当)、座席のある org での実機の通し、2 つのアカウントでの切り替え、Linux の `/proc` の経路
- Findings: LOW 7 件(V1〜V7)、INFO 7 件(I1〜I7)。CRITICAL、HIGH、MEDIUM はない。どれも文書と register の直しで、コードの直しはない

## Insight event

`./scripts/insights-append.sh --slug org-reseat-restart-herdr --flow standard --phase verify --verdict pass --cycle auto --source skill` で 1 行を足し、この report と同じ commit に入れた(cycle は `cycle-count.json` から 1)。skill の命令は件数を渡さないので、行の `findings` はすべて 0 になる(この report の LOW 7 件は数えていない。前の verify の行と同じ形)。
