# Verify report: org-director-start

- Date: 2026-10-15
- Plan: docs/plans/active/2026-10-15-org-director-start.md(`Status: Approved`。`./scripts/plan-visual.sh digest` は d59311f57d8e で、`- Approved:` の digest と同じ。仕様 `docs/specs/2026-10-07-org-multi-org-director.md` の FR-9 の起動 (B) のうち herdr の部分、8 段目の 8a-2)
- Verifier: verifier subagent (Claude)。パイプライン 1 回目(`.harness/state/standard-pipeline/cycle-count.json` の cycle は 1)
- Scope: `git diff 9cbf7080...HEAD`(HEAD dbd67a58、14 コミット、41 ファイル、+7347/-383)。仕様適合(D1〜D11、AC1〜AC14)、self-review の M1・L1〜L6 の直し(da818b4f)とコメントの名前の直し(dbd67a58)、文書のずれ、静的解析、隔離した herdr server での実機の確認。テストは走らせていない(/test の受け持ち)。テストはコードを読み、どの AC をどう固定しているかを確かめた。行番号は HEAD dbd67a58 のもの
- Evidence: `docs/evidence/verify-2026-10-15-org-director-start.md`(run-static-verify の出力、Go と同期の検査、self-review の直しの grep、実機の確認 L1〜L11 と後片付け)。生の出力は `docs/evidence/verify-2026-10-15-org-director-start.log`、wrapper の全ログは `docs/evidence/verify-2026-10-15-084118.log`(どちらも gitignore の対象)

## Spec compliance

| Acceptance criterion | Status | Evidence |
| --- | --- | --- |
| AC1 start は `attached`(launch・token・model・permission_mode)を先に書き、label `ralph_director_<token>` の workspace と `director_<token>` の tab を作るたびに記録し、agent start を permission の引数・`--model`・雛形を指す 1 文の順で打ち、`launched` を書く。manifest と model receipts には書かない。claude も同じ形 | Met | `StartDirector`(`internal/org/director_start.go:458`)が `AttachHeadless`(101)、`createWorkspaceAndTab`(575、`recordHerdrID` 187)、`writePrompt`(603)、`startAgent`(617)、`MarkLaunched`(239)の順に呼ぶ。stdout の 3 行は `startDirector`(`internal/cli/org_director_headless.go:161`)。テスト: `TestStartDirector_LaunchesInAWorkspaceOfItsOwn`(`director_start_test.go:176`、引数の順を含む)、`TestOrgDirectorStart_LaunchesRegistersAndStops`(`org_director_headless_test.go:139`)。実機 L1・L2(codex `gpt-6-astra`)と L5・L6(claude `haiku`): イベントは attached → workspace_created → pane_created → launched、herdr の label と agent 名は記録と同じ、台帳に manifest も receipts もない。claude の pane には `claude --model haiku '<pointer>'` が打たれた |
| AC2 attached の登録(`starting` を含む)、`enabled = false`、`--org-id`、違う driver、`[org.roles]` で許されない model、`HERDR_PANE_ID` では start が断り、herdr を打たず何も書かない。headless なら `director stop --id`、そうでなければ `detach --id` を案内する。`starting` には attach も stop を案内する | Met | `checkDirectorStart`(`director_start.go:508`)が enabled、`HERDR_PANE_ID`(517)、driver、model、permission(523-524)、登録の読み(531)、見張り(538)の順に確かめる。案内の文は `directorStartAttachedErr`(161)と CLI の `startAttachedError`(`org_director_headless.go:230`)。テスト: `TestStartDirector_RefusesBeforeWritingOrCallingHerdr`(`director_start_test.go:331`)、`ARegistrationAfterTheFirstLookIsRefusedUnderTheLock`(438)、CLI の `TestOrgDirectorStart_RefusedWithNothingWritten`(235)、`AnAttachedDirectorIsRefusedBeforeTheWatchGate`(324)、`TestOrgDirectorAttach_RefusedByAHeadlessDirectorNamesStop`(925)。実機 L7: d2 が動く間の start と attach、`HERDR_PANE_ID` のある start が断られ、`director.jsonl` の md5 は前後で同じ |
| AC3 workspace・tab・雛形・agent start のどこかで失敗すると、作った分を閉じて `stopped` と `detached` を書き終了コード 1。閉じられなければ `detached` だけを書き、stop と herdr の打ち方を出す | Met | `abort`(`director_start.go:645`)、`DirectorCleanupError`(`internal/org/director_stop.go:231`)。テスト: `AFailedStepClosesWhatItMadeAndRecordsTheStop`(`director_start_test.go:460`)、`ARecordThatFailsStillClosesWhatHerdrMade`(525)、`ACleanupThatCannotCloseLeavesItDetachedOnly`(568)、CLI の `TestOrgDirectorStart_AFailedAgentStartIsCleanedUp`(561)。実機 L11: `--timeout-ms 200` の agent start が `agent_pane_busy` で失敗し、start は w2 を閉じ、d3 に detached と stopped を書いて終了コード 1。閉じられない分岐はテストのコードだけで確かめた |
| AC4 `attached` と `launched` の間の `attach --replace` で、start は自分の分を閉じ `stopped` を書き `launched` を書かない。雛形の最初の手順は登録の `id` の確かめ | Met | `replacedError`(`director_start.go:630`、632 で読み直す)と `MarkLaunched` の `stillCurrent`。テスト: `TestStartDirector_AReplaceOrDetachWhileTheAgentStartsEndsItsOwnLaunch`(640)、雛形の文は `TestRenderDirectorPrompt_PinsTheHeadlessDirectorsProcedure`(`prompts_test.go:1479`)。実機 L3: codex の director は最初に `ralph org director status --state-dir '<台帳>'` を打ち、確かめられなかったので以降を打たずに止まった(下の Observational checks) |
| AC5 2 つの台帳が 1 つの herdr を使い id が振り直されたとき、A の stop は B の pane と workspace に C-c も close も打たず、何も書かない。token は起動ごとに違う | Met | `confirmDirectorResources`(`director_stop.go:307`)が `confirmSeatPane`(313)と `confirmOrgWorkspace`(321)を `directorLabelWords` で呼ぶ。テスト: `TestDirector_TwoLedgersOnOneHerdrServerNeverCloseEachOther`(`director_stop_test.go:271`)、`ARenumberedWorkspaceIDIsTheLabelMismatch`(556)、CLI の `TestOrgDirectorStop_RenumberedHerdrIDsAreRefusedThenForced`(665)。実機 L9: 手で書いた台帳 B の d1(token `aaaaaaaaaaaa`、w1・w1:p2)の stop は label の違いで断られ、B の md5 は同じで、herdr の d2 は残った。3 回の start の token は 9da3599cba71・82130355444e・c7805b15e99b |
| AC6 permission は `[org.permissions.roles].director`、なければ default。claude は autonomous で bypassPermissions、edits で acceptEdits、guarded で引数なし。codex は `codex_verified = false` で guarded のほかを断る。`--config` がなく台帳が main worktree のものなら main worktree の `ralph.toml` を読む | Met | `checkDirectorStart` の `ResolvePermissionMode(o.Config, DirectorRole)`(523)と `permissionArgsForDriver`(524)、`codexWritableRootArgs`(`internal/org/permissions.go:177`、178 で director を含む)、`withMainWorktreeDirectorStartConfig`(`internal/cli/org.go:230`)。テスト: `TestStartDirector_PermissionModeOfRoleDirector`(`director_start_test.go:261`)、`TestOrgDirectorStart_ReadsTheMainWorktreeRalphToml`(`org_director_headless_test.go:390`)。実機: `default = "autonomous"` の設定で `roles.director = "guarded"` が勝ち、`codex_verified = false` の codex は guarded で通り、claude の pane には permission の引数がなかった |
| AC7 見張りがなければ start が見張りを起動してから director を立てる。止まりかけ・印が古い・`--once` では断り、herdr を打たず何も書かない | Met(印が古い場合は共有の配線から推した) | `wireOrgStartWatchGate`(`internal/cli/org.go:396`)を verb `director start` で使う。テスト `TestOrgDirectorStart_WatchGate`(`org_director_headless_test.go:487`)は止まりかけ、`--once`、見張りなしで起動、起動の失敗の 4 つ。印が古い場合は同じ `ensureOrgWatch` を通る `ralph org start` のテスト(`org_start_watch_test.go:427`)にしかない。実機 L1: 見張りのない台帳で start が `started the cross-org watch (pid 44058)` を出し、L5 はその見張りを使った |
| AC8 label が合えば C-c、pane と workspace の close、`stopped`。今の登録なら `detached` も同じ write。detach 後と `starting` の登録にも効き、herdr が知らない分は閉じ済みとする | Met | `StopDirector`(`director_stop.go:114`)、`closeConfirmedDirectorResources`(337)、`recordDirectorStop`(153)、`markStopped`(`director_start.go:312`、detached を先に書く)。テスト: `TestStopDirector_ClosesThePaneAndWorkspaceAndRecordsTheStop`(`director_stop_test.go:23`)、`WhatHerdrNoLongerKnowsCountsAsClosed`(67)、`TestDirectorMarkStopped_TheCurrentRegistrationGetsDetachedAndStoppedInOneWrite`(`director_test.go:1581`)、CLI の `TestOrgDirectorStop_ADetachedHeadlessDirector`(622)、`AStartingRegistrationAndABrokenRalphToml`(891)。実機 L4・L10: `ctrl_c=ok, pane=closed, workspace=closed`、workspace と agent の一覧は空、`ralph status` は not attached |
| AC9 `--id` なし、閉じていない登録なし、`stopped` 済み、pane か workspace の中からの stop は断り何も書かない。`--force` なしでは label の違いと close の失敗も断る。昇格したセッションには `detach --id` を案内する | Met | `openHeadlessRegistration`(`director_stop.go:185`)、pane の拒否(128)と workspace の拒否(133、label を確かめたときだけ)、`notHeadlessError`(`org_director_headless.go:301`)。テスト: `TestStopDirector_Refusals`(`director_stop_test.go:200`)、`ALabelNotThisLaunchsSendsNothing`(100)、`ACloseThatFailsRecordsNothing`(158)、`RefusesInsideTheDirectorsWorkspace`(475)、CLI の `TestOrgDirectorStop_Refusals`(811)、`RefusedInsideTheDirectorsPaneOrWorkspace`(737)、`ACloseThatFailsNamesForce`(774)。実機 L7 と L9 で 6 つの拒否を見た |
| AC10 `director status` と `ralph status` の `director:` の行に headless の状態・pane・workspace・model、`--json` にも。`ralph org stop --all` は閉じていない headless の登録の止め方を 1 行出し、終了コードを変えない | Met(`starting` は実機で見ていない) | `orgDirectorLine`(`internal/cli/org_director.go:494`)、`orgDirectorOpenHeadlessNote`(550)、`printStopAllDirectorNote`(`internal/cli/org.go:1451`)。テスト: `TestStatusCmd_DirectorLineShowsHeadlessDirectors`(`status_test.go:821`)、`DirectorStopCommandRepeatsTheStateDir`(896)、`TestOrgDirectorStatus_ShowsHeadlessDirectors`(`org_director_headless_test.go:1042`)、`TestOrgStopAll_NamesTheOpenHeadlessDirectors`(`org_stop_all_test.go:910`)、`TestOrgStopAll_ReportsOpenHeadlessDirectorsAndLeavesThem`(`verbs_all_test.go:149`)。実機 L2・L6 で `headless running` と pane・workspace・model、`--json` の欄。L8 の `stop --all --dry-run` は note を 1 行出し終了コード 0、L10 のあとは note なし |
| AC11 `ralph org spawn --role director` は断られ何も書かない。`RenderDirectorPrompt` は `{{...}}` を残さず、雛形は `/director` skill・`--state-dir`・登録の確かめ・stop の頼み方・detach を打たないことを書く | Met | `DirectorRoleInputErr`(`internal/org/spawn.go:1430`、1117 と CLI の `org.go:692` で呼ぶ)、`RenderDirectorPrompt`(`internal/org/prompts.go:129`)。テスト: `TestOrgSpawn_DirectorRole_RejectedBeforeAnyManifestWrite`(`spawn_test.go:3316`)、`TestDirectorRoleInputErr_IsExactAndCaseSensitive`(3381)、CLI の `TestOrgSpawn_RoleDirectorIsRefusedBeforeTheModelFallback`(1107)、`prompts_test.go` の 1452・1479・1529・1560。実機 L7: spawn `--role director` は終了コード 1 で、manifest は書かれなかった。L3 の雛形の写しで `--state-dir` は引用符付きだった |
| AC12 `/director` skill の 4 面が同じ、2 つの同期の検査が通る。help・README・仕様 FR-9 の注・AGENTS.md・`templates/base/ralph.toml`・tech-debt が 8a-2 を説明し、「まだない」の残りがない | Met(AC の外の 2 面のずれは V1・V2) | 4 面の md5 は 6509f6c71f7e855db1c4b83f56794051 で同じ、`check-skill-sync.sh` と `check-sync.sh`(DRIFTED 0)は pass。`/director` skill の「まだないもの」は launch_cmd・自動 merge・push だけ。仕様の 8a-2 の注(`docs/specs/2026-10-07-org-multi-org-director.md:85`)は launch_cmd を 8a-2b に送ったと書く。残りの grep で当たった README:258 の「left to a later stage」は launch_cmd の Non-goal で、計画と合う |
| AC13 テストは本物の herdr と osascript を打たず repo の台帳に書かない。実機は人の herdr server に触れない隔離した server で行う | Met(テストの側はコードを読んだ範囲) | テストは偽の herdr のログと `t.TempDir` の台帳を使う(`directorStartLedger` など)。実行時の確認は /test。実機は `env -i` で XDG の 2 変数と `HERDR_SOCKET_PATH` を `/tmp/rdst` に分けた server で行い、`lsof` でその log だけを開いていることを確かめた。人の `~/.config/herdr/session.json` の sha256 と server log の更新時刻は前後で同じ。main と worktree のどちらにも `.harness/state/org` はない |
| AC14 `--force` は label が違う・確かめられない・閉じられないとき、打っていない C-c と close を打たずに `force` 付きの `stopped` を書き、label の探し方を出す。label が合えば閉じる。pane と workspace の中からの拒否は変わらない | Met | `MarkForceStopped`(`director_start.go:307`)。テスト: `ForceRecordsAStopWhoseLabelsAreAnotherLaunchs`(`director_stop_test.go:338`)、`ForceWithLabelsThatMatchClosesAsWithoutIt`(382)、`ForceAfterACloseThatFailsRecordsWhatWasLeftOpen`(403)、`ForceWhenAGetTimesOutSendsNothing`(446)、`AForcedStopThatCannotRecordSaysToRetryWithForce`(600)、`TestDirectorMarkForceStopped_OnlyTheStoppedLineCarriesForce`(`director_test.go:1658`)。実機 L9: 台帳 B に detached と `"force":true` の stopped が書かれ、d2 は残り、台帳 A の md5 は同じ。L7: `HERDR_WORKSPACE_ID=w1` の `--force` も断られた |
| FR-3「director は座席ではないので数えない」 | Met | AC1 の manifest に書かないこと、AC11 の spawn の拒否。実機の台帳にも manifest はない |
| D3 登録を先に書き、`director.lock` の下で herdr を打たない | Met | `AttachHeadless` は `withLock`(`director_start.go:127`)の中で attached を書いて戻り、`StartDirector` はそのあとロックの外で `createWorkspaceAndTab`(474)と `startAgent`(483)を呼ぶ。ここはコードを読んで確かめた。attach との同時の登録は `TestDirectorAttachAndAttachHeadless_ConcurrentlyOnlyOneRegisters`(`director_test.go:1957`)が固定する |

### self-review の直し(da818b4f、dbd67a58)

| Finding | Status | Evidence |
| --- | --- | --- |
| M1 tech-debt の行が 8a-2 の動きと合わない | 直った | 245 行目は 8a-2 が `pane_id` を埋めないと書き、171・189・192・212・234〜241 に 8a-2 の文が入った。変えた行の列はすべて 5 |
| L1 見張りの検査より先に登録を読む | 直った | `checkDirectorStart` の `Store.Read`(531)が `checkWatchGate`(538)より前。`AnAttachedDirectorIsRefusedBeforeTheWatchGate` が固定する |
| L2 detach と `attach --replace` のあとに閉じていない headless の登録を知らせる | 直った | `printHeadlessStillOpen`(`org_director.go:360`、139 と 206 で呼ぶ) |
| L3 `ralph status` の stop の打ち方に `--state-dir` | 直った | `status.go` の 2 か所が `orgDirectorLine(director, orgInboxHintStateDir(cmd, stateDir))` を呼ぶ |
| L4 director のエラーに seat と org_id が出る | 直った | `confirmSeatPane` と `confirmOrgWorkspace` が `labelWords` を取り、座席の 4 か所(`verbs.go:1138`、1239、1325、2135)は `seatLabelWords`、director の 2 か所は `directorLabelWords` |
| L5 包むだけの関数 | 直った | `ModelAllowedForRole`(`internal/org/envelope.go:141`)と `DirectorRoleInputErr` を直接呼ぶ。`directorRoleSpawnErr` と小文字の `modelAllowedForRole` は grep で 0 件 |
| L6 長いコメント行 | 直った | 指摘のファイルに 82 桁を超えるコメント行はない。`spawn.go` の 53・93・1640・2193 行は blame で 9cbf7080 より前のコミット |
| dbd67a58 改名前の名前が残ったコメント | 直った | 3 ファイルの 4 行(`doctor_codex_writable_root.go` の 1 行、そのテストの 2 行、`prompts.go` の 1 行)が `ModelAllowedForRole` と `DirectorRoleInputErr` になった |

## Static analysis

| Command | Result | Notes |
| --- | --- | --- |
| `./scripts/run-static-verify.sh`(changed scope) | pass(rc 0) | shellcheck、hook の `sh -n`、guard の awk、settings.json の jq、Codex の 3 つの guard、check-sync(IDENTICAL 169、DRIFTED 0)、check-pipeline-sync、check-skill-sync(14 skill)、check-template-purity、tech-debt の計画の参照、golang(`gofmt: ok`、golangci-lint `0 issues.`、staticcheck は出力なし)、secret scan(9cbf7080..dbd67a58 clean)。`templates/base/ralph.toml` の変更で言語の範囲は full に広がった |
| `gofmt -l`、`go vet ./...`(go1.26.0) | pass | 出力なし |
| `GOTOOLCHAIN=go1.25.8 go build ./...` と `go vet ./...` | pass | go.mod は `go 1.25.8`。go1.25.8 の `gofmt -l ./cmd ./internal` も出力なし |
| `git diff --check 9cbf7080...HEAD` | pass | 出力なし |
| `/director` skill の 4 面の md5 | 同一 | 6509f6c71f7e855db1c4b83f56794051 |

## Documentation drift

| Doc / contract | In sync? | Notes |
| --- | --- | --- |
| V1 `.claude/skills/org/SKILL.md`(4 面) | No(LOW) | 58 行目の台帳の動詞の一覧に `director start` と `director stop` がない。186 行目の `director` の行は attach・detach・status と `attached`・`detached` のイベントだけを書き、director を「人が `/director` で昇格させた対話セッション」と言う。312 行目も同じ言い方。計画の Progress は /sync-docs に渡すと書いている |
| V2 `.claude/rules/ralph/ralph-workflow.md`(root と template、同一) | No(LOW) | 28〜30 行目の `/director` の項は昇格と `director attach` だけを書く。headless の director(`ralph org director start`)に触れない。skill の説明として誤りではないが、director の起こし方がもう 1 つあることが分からない |
| V3 `templates/base/ralph.toml:105-107` | 条件付き(LOW) | 「A guarded claude director stops at the permission prompt in its herdr pane」は、人の Claude Code の設定が `permissions.defaultMode` を `bypassPermissions` にしていないときだけ成り立つ。guarded は引数を渡さないので、claude は人の `~/.claude/settings.json` に従う。この環境ではその設定だったので、実機では `CLAUDE_CONFIG_DIR` を空の一時の場所に向けて避けた。codex も同じで、guarded の codex の director は `~/.codex/config.toml` の `approval_policy = "never"` と `danger-full-access` で動いた(L3)。96〜99 行目の reviewer の注と同じ種類の前提なので、「claude's own permission settings decide」のような書き方か tech-debt の 1 行にする |
| V4 計画の Evidence to capture(141 行目)と self-review の 26・62 行目 | No(INFO) | label を「22 字」と書くが、`ralph_director_<token>` は 15 + 12 で 27 字。herdr は 27 字の label をそのまま返した(L2、L5)。tab と agent 名の `director_<token>` は 21 字 |
| V5 `docs/tech-debt/README.md:189` | 古くなった(INFO) | 「the version skew can reach the director too ... (a judgment, not measured)」は今回測れた。codex の director が pane で打った `ralph` は Homebrew の 5.1.0 で、`org director` がなく、状態の確かめは `ralph org` の help を返した(L3)。dev build で確かめたときに起きる形で、同じ版を入れた利用者には起きない。行の書き換えは /sync-docs の判断に任せる |
| V6 計画の Acceptance criteria と Progress checklist | 遅れ(INFO) | AC の checkbox と「Review artifact created」が未チェック。オーケストレーターが更新する欄なので失敗とはしない |
| help、README:258、仕様 FR-9 の 8a-2 の注、AGENTS.md の repo map、tech-debt の新しい行 | Yes | help の動詞と flag(`--driver`、`--model`、`--timeout-ms`、`--id`、`--force`)は実機の出力と合う。README の段落は start・stop・`--force`・permission・`stop --all` の note・launch_cmd の Non-goal を書き、実機の動きと合う |

## Observational checks

- 実機の確認は、計画の Evidence to capture のとおり隔離した herdr server(herdr 0.7.5)で行った。codex-cli 0.160.1 と Claude Code 2.1.291 の両方で start と stop が通った。詳細は evidence の 4
- codex の director(guarded)は 2 つのコマンドだけを打った。雛形のファイルの `cat` と `ralph org director status --state-dir '<台帳>'` で、後者は Homebrew の古い `ralph` が `ralph org` の help を返した。director は「登録 ID が `d1` か確認できませんでした。役割指示に従い、以降の操作を進めず停止しています。」と書いて止まった。雛形の最初の手順が確かめられないときに止まることを、偶然ながら実機で見た。約 50 秒後に stop した
- claude の director は空の `CLAUDE_CONFIG_DIR` のため初回のテーマ選択の画面で止まり、雛形を読んでいない。herdr はこの画面を `interactive_ready` と数えたので、start は `launched` を書いた。claude の director が雛形どおりに動くかは確かめていない
- herdr server を立て直すと id は w1 から振り直された。d2 は止めた d1 と同じ `w1`・`w1:p2` を得た。AC5 の前提(id の振り直し)は実機で起きる
- 拒否の文はどれも打ち方を `--state-dir '<台帳>'` 付きで出した
- 後片付け: start が起動した見張り(pid 44058)は `ralph org watch --stop` で止めた。隔離した herdr server(1 台目 pid 35480、2 台目 pid 77399)は `herdr server stop` で止め、`/tmp/rdst` を消した。前後の `pgrep -fl 'herdr server'` はどちらも空で、人の herdr server は動いていなかった。scratch と `director_<token>` の名前に当たるプロセスは残っていない。osascript は PATH にないので打たれていない
- 一時の場所の外に残ったもの: codex のセッション記録 1 件(`~/.codex/sessions/2026/10/15/`)と、codex の起動時刻に更新時刻が変わった `~/.codex/config.toml`。後者に scratch の path はない。何が変わったかは前の写しを取っていないので分からない(おそらく `[tui.model_availability_nux]` の数。未確認)。どちらも戻していない

## Coverage gaps

- テストの実行結果(/test の受け持ち)。AC1〜AC14 のテストはコードを読んだだけ
- AC7 の印が古い見張りでの director start の拒否は、`ralph org start` のテストと共有の配線から推した。director start のテストにも実機にもない。いちばん安い追加は `TestOrgDirectorStart_WatchGate` の表に「stale with an old mark」の 1 行(`org_watch_gate_test.go:77` の印を使う)を足すこと
- AC3 の閉じられない分岐と、AC14 の close の失敗と get の時間切れは、偽の herdr のテストだけ
- AC10 の `starting` の状態は実機で作っていない(start は数秒で `launched` まで進む)
- claude の director が雛形どおりに動くことは見ていない(テーマ選択で止まった)。codex の director が新しい `ralph` で動いた場合の 2 手目以降も見ていない(pane の PATH が古い `ralph` を見つけた)
- macOS 以外と、herdr 0.7.5 以外の版

## Verdict

- Verdict: pass(V1〜V3 は LOW の文書のずれで /sync-docs で直せる。merge を止めるものではない)
- Verified: AC1〜AC14 と D3、FR-3 の文がコードとテストのコードに対応している。self-review の M1・L1〜L6 と dbd67a58 の直し。静的解析(run-static-verify、gofmt、go vet、go1.25.8 の build・vet・gofmt、`git diff --check`、skill の 4 面)。隔離した herdr server での実機: codex と claude の start、label と agent 名、`director.jsonl` のイベント、`ralph status` と `director status` の行、stop で pane と workspace が閉じること、L7 の 9 つの拒否、id の振り直しのあとの label の違いでの拒否と `--force`、agent start の失敗の後始末、`stop --all` の note
- Partially verified: AC7(印が古い場合)、AC10(`starting`)、AC13(テストの側はコードを読んだ範囲)
- Not verified: テストの実行、claude の director が雛形を読んで動くこと、新しい `ralph` を PATH に持つ codex の director の 2 手目以降、`~/.codex/config.toml` の変更の中身

## Cycle 2

- Date: 2026-10-15
- Verifier: verifier subagent (Claude)。パイプラインの 2 回目(`.harness/state/standard-pipeline/cycle-count.json` の cycle は 2 で、既定の上限 2 の最後)。cross-review cycle 1 の ACTION_REQUIRED 1 件(`docs/reports/cross-review-triage-org-director-start.md`)の直し 3243c499、/test の穴 D4・E1・E12/E13・P2 のテスト 4ef33195、self-review cycle 2(391b8b80、CRITICAL 0・HIGH 0・MEDIUM 0・LOW 5、Merge: yes)の C2-L1〜C2-L4 の直し a539a765 のあとに走らせた
- Scope: `git diff origin/main...HEAD`(merge base f21df66c、HEAD a539a765)のうち、cycle 2 の差分 `git diff 54cab8a2..HEAD`(15 ファイル、+526/-55。Go は `internal/org/prompts.go`・`prompts/director.md`・`director_start.go`・`internal/cli/org_director_headless.go` とそのテスト)を中心に読んだ。cycle 1 の verify(ec70c739)から 54cab8a2 までの ca98c248(AC7 の印が古い見張りのテスト)、b6c86188(計画の字数)、1cdddbbb(origin/main の merge)、e80ccafe(sync-docs)、e683ee47(AC の tick)も、cycle 1 の指摘が閉じたかを見る範囲で読んだ。テストは走らせていない(/test の担当)。この節の行番号は HEAD a539a765 のもの。上の cycle 1 の節の行番号は dbd67a58 のまま
- Plan: `Status: Approved`。`./scripts/plan-visual.sh digest` は f65d7c59746c で、`- Approved:` の digest と同じ。Scope の `--config` の 1 文(46 行)と AC11 は cycle 2 のために足された
- Evidence: `docs/evidence/verify-2026-10-15-org-director-start.md` の `## Cycle 2 (HEAD a539a765)`。C2-1 は run-static-verify の出力、C2-2 は Go と同期の検査、C2-3 は HEAD の binary と偽の herdr で `--config` ありとなしの start を 1 回ずつ打った probe(雛形のファイル 2 つの全文)、C2-4 は cross-review の場面を HEAD と直す前の 54cab8a2 の binary で並べた probe、C2-5 は `ralph status` の `director:` の行、C2-6 は help、C2-7 は後片付け。wrapper の全ログは `docs/evidence/verify-2026-10-15-122004.log`、生の出力は `docs/evidence/verify-2026-10-15-org-director-start.log` の末尾(どちらも gitignore の対象)
- Insight event: `./scripts/insights-append.sh --slug org-director-start --flow standard --phase verify --verdict pass --cycle auto --source skill` で 1 行足し、この報告と同じコミットに入れた。findings はすべて 0(下の情報 5 件は数えない)

### cycle 2 の直しの確かめ

| 変更 | Status | Evidence |
| --- | --- | --- |
| cross-review #1: `--config` を絶対パスにして雛形に入れ、director が打つ `ralph org` のコマンドに付ける(3243c499) | 満たす(binary でも確認) | CLI の `startDirector`(`internal/cli/org_director_headless.go:168`)が `ConfigPath: configPath` を `StartDirector` に渡す(`:199`)。`checkDirectorStart`(`internal/org/director_start.go:516`)が state dir と cwd のあとで `absPath` にし(`:555-559`)、`writePrompt` が `RenderDirectorPrompt` に渡す(`:617`)。`RenderDirectorPrompt`(`internal/org/prompts.go:155`)は絶対でない値を断り(`:165`)、`{{ORG_FLAGS}}` を `--state-dir <台帳>`、設定があればそのあとに `--config <path>`(どちらも `shellQuote` の 1 語)で描く(`:172-186`)。雛形で `{{ORG_FLAGS}}` を使うのは 13・30・33 行。テスト(読んだだけ): `TestRenderDirectorPrompt_GivesEveryRalphOrgCommandTheConfig`(`prompts_test.go:1583`。空白と単引用符を含むパス、ありとなしの 2 通り、`ralph org` の言及の数とコードの中の数の一致)、`TestRenderDirectorPrompt_RefusesBadVarsAndLeftoverPlaceholders` の 2 行(`:1640`。相対のパス、`{{` を含むパス)、`TestStartDirector_CarriesTheConfigIntoThePrompt`(`director_start_test.go:288`。相対のパスが呼んだ側の場所で絶対になる)、CLI の `TestOrgDirectorStart_ReadsTheMainWorktreeRalphToml` の `--config` の行(`org_director_headless_test.go:447`。linked worktree で打ち、director は main のルートで動く)。binary: C2-3 の R1 で相対の `cfg/ralph.toml` が雛形に `'$PROBE/work/cfg/ralph.toml'` として入り、雛形の最初のコマンドをそのまま打つと `attached d1`。C2-4 で、`./ralph.toml` が `enabled = false`、`--config` の設定が true のとき、54cab8a2 の director の最初のコマンドは `director: disabled` を返し(雛形の手順では何もせず止まる)、HEAD は `attached d1`・`enabled: true` を返した |
| 人に頼む stop の行は `--state-dir` だけ(C2-L1) | 満たす(binary でも確認) | 雛形の 51 行は `--state-dir {{STATE_DIR}}` だけを持ち、33〜36 行は「あなたが打つ `ralph org` のコマンド」に `{{ORG_FLAGS}}`、`ralph status` と stop には `--state-dir` だけと書く。help(`org_director_headless.go:59-62`)、`startDirector` の doc(`:155-167`)、`DirectorPromptVars`・`RenderDirectorPrompt` の doc(`prompts.go:111-154`)も同じ区別を書く。`director stop` は ralph.toml を読まない(`newOrgDirectorRuntime` を `loadConfig` false で呼ぶ、`org_director_headless.go:141`)。テストは prompt・org・CLI の 3 層で stop の行の末尾が `--state-dir` だけであることを見る。binary: R1 と R2 の 51 行はどちらも `ralph org director stop --id dN --state-dir '$PROBE/state'` で、そのまま打って `stopped dN (ctrl_c=ok, pane=closed, workspace=closed)`。start が stdout に出す stop の行とも同じ形 |
| `--config` がなければ `--config` を書かない(AC11 の後半) | 満たす(binary でも確認) | 設定の行は `directorPromptNoConfigText`(`prompts.go:136`)で、`--config` の字を含まない。R2 の雛形で `grep -c -- --config` は 0。R1 との差は id、設定の行、`--config` を持つ 3 行だけ(C2-3 の diff) |
| `{{CONFIG}}` を `{{CONFIG_LINE}}` に改名(C2-L2) | 満たす | 雛形(5 行)と `prompts.go`(`:129`、`:145`、`:184`)だけが使い、`{{CONFIG}}` は `internal/`、skill の 4 面、README に残っていない。計画の Progress の 188 行は 3243c499 の時点の名前 `{{CONFIG}}` を書いている(下の C2-V1) |
| skill の「始める」の 1 と 3(C2-L3) | 満たす | `.claude/skills/director/SKILL.md:149-153` は最初のコマンドを `--config` ありとなしの 2 形で書き、`:159-164` は目印を「`- 設定:` の行が `(start の --config)` で終わっていれば」と名指す。R1 の雛形の 5 行は `'<絶対パス>'(start の --config)` で終わり、R2 の 5 行はその目印を持たないので、skill の文で見分けられる。4 面の md5 は dbe32a18b8b46c47b44657c4e2443cfe で同じ |
| テストのコメントの報告番号(C2-L4) | 満たす | branch の Go の追加行に「finding N」「cross-review cycle」「/test gap」「mutant X1」「gap N」はない(C2-2 の grep が exit 1) |
| /test の穴 D4・E1・E12/E13・P2 のテスト(4ef33195) | 満たす(テストのコードで確認) | D4: `TestStartDirector_RefusesBeforeWritingOrCallingHerdr` の「director.jsonl unreadable」の行(`director_start_test.go:454`。`director.jsonl` を directory にし、gate を呼ばずに EISDIR で断る)。E1: `TestStartDirector_AFailedWriteOfAttachedCallsNoHerdr`(`:655`。最初の write の ENOSPC で herdr の呼び出しが 0、`prompts/` もない)。E12/E13: `TestStartDirector_AFailedCreatesOutputIsNotAnID`(`:614`。失敗した create が出力も返す偽の herdr、`failedCreateOutput`、`:22-50`)。P2: `TestOrgDirectorStart_AnAttachedRefusalWithNoCurrentNamesStatus`(`org_director_headless_test.go:377`。`startAttachedError` の `cur == nil` の文を `--state-dir` ありとなしで等号で比べる)。赤くなるかは /test の担当 |
| tech-debt の行を A12〜A14 の注に縮めた(4ef33195) | 満たす | `docs/tech-debt/README.md:256` は 5 列。挙げた seat のテスト `TestOrgStopDisband_UnansweredGet_LeavesItAlone`(`internal/org/verbs_test.go:2436`)と `TestOrgStop_PaneNotConfirmed_NoCtrlCNoClose`(`:2305`)は HEAD にあり、test の evidence の 949〜954 行(A12〜A14 の `-orgfull` が red、`-clifull` が SURVIVED)と合う |
| cycle 1 の V1〜V6 | 直った | V1: `/org` skill の動詞の一覧(58〜59 行)に `director start`・`director stop`、`director` の行(187 行)に headless の director。V2: `ralph-workflow.md:31-32` に `ralph org director start` と `director stop`(root と template は check-sync で同じ)。V3: `templates/base/ralph.toml:105-108` は guarded の director が agent 自身の承認の設定に従うと書く。V4: b6c86188 で計画の字数を 27 に。V5: e80ccafe で tech-debt の版の違いの行に測った結果。V6: e683ee47 で AC と Progress の checkbox |
| AC7 の印が古い見張り(cycle 1 の一部未確認) | 満たす(テストのコードで確認) | ca98c248 が `TestOrgDirectorStart_WatchGate` に「stale with an old mark」の行を足した(`org_director_headless_test.go:556`)。cycle 1 の Coverage gaps で勧めた形 |

### AC の確かめ直し(HEAD a539a765)

cycle 2 の Go の差分は、雛形の描き方(`--config` の運び方、設定の行、stop の行)、`DirectorStartParams.ConfigPath` の絶対化、CLI の受け渡しと help、テストだけだった。登録・停止・表示・permission・見張りの関門の本体のコードは変わっていない(`git diff --stat 54cab8a2..HEAD -- '*.go'` は上の 4 ファイルとそのテスト)。

| Acceptance criterion | Status | Evidence |
| --- | --- | --- |
| AC1 | 満たす(binary でも確認) | C2-3 の R1・R2 の `director.jsonl` は attached(launch・token・model・permission_mode)→ workspace_created → pane_created → launched の順。偽の herdr の記録では label が `ralph_director_<token>` と `director_<token>`、agent start は `--` のあとに permission の引数(guarded はなし、edits は `--permission-mode acceptEdits`)、`--model haiku`、雛形を指す 1 文の順。4 つの台帳に `manifest.jsonl` も `model-receipts.jsonl` もない |
| AC2 | 満たす(テストのコードで確認) | 拒否の経路は変わっていない。D4 と P2 のテストが、読めない `director.jsonl` と `cur == nil` の案内の 2 つの穴を埋めた。binary での拒否は cycle 1 の L7 のまま |
| AC3 | 満たす(テストのコードで確認) | E1 と E12/E13 のテストが、attached の write の失敗と、失敗した create の出力を id として使わないことを固定した。後始末の本体は変わっていない |
| AC4 | 満たす(binary でも確認) | 雛形の最初の手順は `ralph org director status {{ORG_FLAGS}}` になり、`--state-dir <台帳>` を含む(`--config` があればそのあとに続く)。違う `id` と `not attached` で何もしない文は 16〜19 行のまま。R1・R2 でそのコマンドを書かれたまま打つと、自分の `dN` が今の登録として出た。`launched` の前の `--replace` の経路は変わっていない |
| AC5 | 満たす(cycle 1 のまま) | `director_stop.go` と `confirmSeatPane`・`confirmOrgWorkspace` は差分の外。R1・R2・R3・R4 の 5 回の start の token はどれも違う |
| AC6 | 満たす(binary でも確認) | R1 は `--config cfg/ralph.toml` の `roles.director = "guarded"` を、R2 は `./ralph.toml` の `"edits"` を読み、`permission=` と agent start の引数がそれぞれ合った。main worktree の `ralph.toml` を読む規則は差分の外で、CLI のテストは `--config` の行を足しただけ |
| AC7 | 満たす(テストのコードで確認) | 印が古い場合は ca98c248 のテストで固定された。R1・R3・R4 では見張りがなく、start が起動した(`started the cross-org watch`)。R2 は R1 の見張りを使った |
| AC8 | 満たす(binary でも確認) | 雛形の stop の行をそのまま打つと、偽の herdr の label の確かめ(pane get、tab get、workspace get)、C-c、pane と workspace の close のあと `detached` と `stopped` が書かれ、`the director is no longer attached`(R1・R2) |
| AC9 | 満たす(cycle 1 のまま) | stop の拒否の経路は差分の外 |
| AC10 | 満たす(`starting` は実機で見ていない) | `orgDirectorLine` と `status.go` は差分の外。R1・R2 の `director status` は `headless running`、pane、workspace、model を出した |
| AC11 | 満たす(binary でも確認) | spawn の `--role director` の拒否は差分の外。`RenderDirectorPrompt` は `{{` の残りを断る検査を保ち、R1・R2 の雛形にも残っていない。`/director` skill、`--state-dir`、最初の登録の確かめ、stop の頼み方、detach を打たないことの文はどちらの雛形にもある。足された文: `--config` を渡した start の雛形は director が打つ `ralph org` のコマンドに `--config '<絶対パス>'` を付け(R1 の 13・30・33 行)、渡さない start の雛形は `--config` を書かない(R2)。人に頼む stop の行は `--config` を持たない。AC の文の「`ralph org` のコマンド」は、Scope の 46 行の「director が打つ `ralph org` のコマンド」の短い言い方として読んだ(下の C2-V2) |
| AC12 | 満たす | skill の 4 面は同じ(md5 dbe32a18b8b46c47b44657c4e2443cfe)、`check-skill-sync.sh`(14 skill)と `check-sync.sh`(DRIFTED 0)が通る。「not available yet」「start はまだ」「headless の director はまだ」は、ないことを確かめるテストと計画の文にだけある。`--config` の文の文書ごとの扱いは下の「文書のずれ(cycle 2)」 |
| AC13 | 満たす(テストの側はコードを読んだ範囲) | 新しいテストは `newDirStartFixture` の偽の herdr と `t.TempDir` の台帳を使う。binary の probe は `env -i` と、ralph・git・偽の herdr だけの PATH で打ち、本物の herdr と osascript は見つからない。repo と worktree の `.harness/state/org` はない |
| AC14 | 満たす(cycle 1 のまま) | `--force` の経路は差分の外 |

FR-3、FR-9 の起動 (B) の herdr の部分、受け入れ条件「director start」の判定は cycle 1 のまま。cycle 2 の直しで、`--config` で渡した設定を director も読むようになった。

### 静的解析(cycle 2)

| Command | Result | Notes |
| --- | --- | --- |
| `./scripts/run-static-verify.sh`(changed scope) | pass(rc 0) | local verifier の全項目が OK(shellcheck、hook の `sh -n`、guard の awk、settings.json の jq、Codex の 3 つの guard、check-sync、check-pipeline-sync、check-skill-sync、check-template-purity、tech-debt の計画の参照)、`gofmt: ok`、golangci-lint `0 issues.`、`secret-scan-branch: scanned f21df66c..a539a765 against origin/main: clean`。`templates/base/ralph.toml` の変更で言語の範囲は full に広がった |
| `gofmt -l ./cmd ./internal`、`go vet ./...`(go1.26.0) | pass | 出力なし |
| `GOTOOLCHAIN=go1.25.8 go build ./...` と `go vet ./...` | pass | go.mod は `go 1.25.8`。go1.25.8 の GOROOT の `gofmt -l ./cmd ./internal` も出力なし |
| `./scripts/check-skill-sync.sh`、`./scripts/check-sync.sh` | pass | 14 skill が lock-step。IDENTICAL 169、DRIFTED 0、KNOWN_DIFF 5 |
| `git diff --check origin/main...HEAD` と `git diff --check 54cab8a2..HEAD` | pass | 出力なし |
| 計画の digest | 一致 | f65d7c59746c |

### 文書のずれ(cycle 2)

| Doc / contract | In sync? | Notes |
| --- | --- | --- |
| `ralph org director start --help`(`org_director_headless.go:59-62`) | yes | 「With --config, the ralph org commands the director's prompt has it run carry --config with that file's absolute path ... (the prompt's stop command, for the human, carries none: stop reads no ralph.toml)」は R1・R2 の雛形と合う(C2-6) |
| `/director` skill の 4 面(「始める」の 1 と 3) | yes | 最初のコマンドの 2 形、`(start の --config)` の目印、`ralph status` には `--config` がないことは、R1・R2 の雛形と `status.go:105-107` の flag と合う |
| 雛形 `director.md` と `prompts.go`・`director_start.go`・`startDirector` の doc | yes | `{{ORG_FLAGS}}` と `{{CONFIG_LINE}}` の中身、stop の行の扱い、絶対化の場所の説明はコードと合う |
| README の director の段落(258 行)、仕様の 8a-2 の注(`docs/specs/2026-10-07-org-multi-org-director.md:85`) | yes(情報 C2-V3) | どちらも `--config` が雛形に入ることを書かないが、誤ったことも書かない |
| 計画の Progress checklist | 一部遅れ(情報 C2-V1) | 下の表 |

| ID | Severity | Where | Finding |
| --- | --- | --- | --- |
| C2-V1 | INFO | 計画の `## Progress checklist`(188 行) | cycle 2 の行は 3243c499 を「`{{ORG_FLAGS}}` と `{{CONFIG}}` で」と書くが、a539a765 で `{{CONFIG_LINE}}` に改名された。a539a765(self-review cycle 2 の C2-L1〜C2-L4 の直し)の行もない。3243c499 の時点の記録としては正しい。オーケストレーターが更新する欄なので失敗とはしない |
| C2-V2 | INFO | 計画の AC11 と Scope の 46 行 | AC11 は「`ralph org` のコマンドに `--config <絶対パス>` を付け」と書き、Scope は「director が打つ `ralph org` のコマンド」と書く。雛形の stop の行は人が打つ `ralph org` のコマンドで、`--config` を持たない(stop は ralph.toml を読まないので、C2-L1 で外した)。実装は Scope の言い方どおりで、AC の文は同じ意味の短い言い方と読んだ。ずれではない |
| C2-V3 | INFO | README:258、仕様の 8a-2 の注(85 行)、`/org` skill の `director` の行(187 行) | `--config` で渡した start の設定を director の雛形に入れることを書かない。help と `/director` skill が書いており、3 つの文はどれも誤りを言わない。次にこの段落に触れるときに 1 句足すとよい |
| C2-V4 | INFO | `/director` skill の「始める」の 3 | `ralph status` には `--config` がないので、`--config` の設定と repo の `ralph.toml` で `[org.director].enabled` が違うと、`ralph status --state-dir` の `director:` の行は `disabled` を出し、director が自分で打つ `director status --config` は `attached`・`enabled: true` を出す(C2-5)。leader の `ralph org escalate` も `--config` なしで repo の `ralph.toml` を読むので、届き方については `ralph status` の行のほうが合っている。`--config` がコマンドごとの指定であることから来る形で、AC11 の外。skill はこの食い違いを書かない |
| C2-V5 | INFO | `internal/org/prompts_test.go:1583` | テストの名前 `TestRenderDirectorPrompt_GivesEveryRalphOrgCommandTheConfig` の「Every」は、C2-L1 で stop の行が `--config` を持たなくなったあとでは言い過ぎになった。doc comment は「every ralph org command the template has the director run」と範囲を書いており、テストの中身は stop の行に `--state-dir` だけを求める |

### 観察

- C2-3〜C2-5 の probe は、HEAD の binary を ralph・git・偽の herdr だけの PATH と `env -i` で打った。偽の herdr は herdr の server に触れず、呼び出しを記録し、label をファイルに持つ。本物の herdr、claude、codex、osascript は動いていない。台帳と cwd は scratch で、cwd は git の repo の外なので、director は start を打った場所で動く(雛形の `- cwd:` の行)
- 雛形の最初のコマンドと stop の行は、雛形のファイルから行をそのまま取り、pane のシェルと同じく `/bin/sh -c` で打った。director が雛形を読んで打つコマンドの形は binary で確かめたが、agent が雛形を読んでそのとおりに打つかは見ていない
- 直す前の binary は `git archive 54cab8a2` から scratch に作った。C2-4 の違いは雛形の 2 行(設定の行と `--config`)と最初のコマンドの答えだけで、start の出力と permission は同じ
- start が起動した見張り 4 つは `ralph org watch --stop` で止め、あとの `pgrep` は空だった。偽の herdr の workspace と pane は stop がすべて閉じた

### Coverage gaps(cycle 2)

- テストの実行(/test の担当)。cycle 2 で足されたテスト(AC11 の 3 層、D4、E1、E12/E13、P2)を go1.26 と `GOTOOLCHAIN=go1.25.8` で走らせるのは /test
- 本物の herdr と agent の上で、claude か codex の director が `--config` 付きの雛形を読み、最初のコマンドをそのまま打つところは見ていない(cycle 1 と同じく、claude はテーマ選択で止まり、codex は古い `ralph` を見つけた)
- `--config` の設定と repo の `ralph.toml` が食い違うときの `ralph status` の行(C2-V4)を、skill か help に書くかは決まっていない
- cwd が symlink 経由で `--config` に `..` があるときの、start が読むファイルと雛形のパスの一致(self-review cycle 2 の Coverage gaps と同じ)
- 足すと確信がいちばん増える確認: cycle 1 の隔離した herdr server で、`CLAUDE_CONFIG_DIR` を初回の画面を済ませた一時の場所にし、PATH の先頭に HEAD の `ralph` を置いて、`--config` 付きの claude の director が最初のコマンドを打って `attached` を読むところを 1 回見る

### Verdict(cycle 2)

- Verdict: pass(cycle 2、HEAD a539a765。情報 5 件はどれも merge を止めない)
- Verified: cross-review #1 の直し(`--config` の絶対化、雛形への受け渡し、director が打つ `ralph org` のコマンドへの付け方、`--config` がないときに書かないこと)を、コードとテストの名前で辿り、HEAD の binary と偽の herdr で `--config` ありとなしの start を打って雛形のファイル 2 つを読み、最初のコマンドと stop の行を書かれたまま打って確かめた。直す前の 54cab8a2 の binary で cross-review の場面が再現し、HEAD で起きないことを確かめた。self-review cycle 2 の C2-L1〜C2-L4、/test の穴 4 つのテスト、tech-debt の行、cycle 1 の V1〜V6、AC7 の印が古い場合のテストを辿った。AC1・AC4・AC6・AC8・AC10・AC11 は binary でも見た。静的解析はすべて通った(run-static-verify、gofmt、go vet、go1.25.8 の build・vet・gofmt、skill と template の同期、`git diff --check`)。計画の digest は承認時と同じ
- Partially verified: AC10(`starting` は実機で見ていない)、AC13(テストの側はコードを読んだ範囲)
- Not verified: テストの実行、本物の agent が `--config` 付きの雛形のとおりに打つこと、symlink 経由の cwd での `--config` の `..`
- Findings: CRITICAL 0、HIGH 0、MEDIUM 0、LOW 0。情報は 5 つ(C2-V1〜C2-V5)
