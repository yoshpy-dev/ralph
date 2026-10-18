# Verify report: org-director-liveness

- Date: 2026-10-19
- Plan: `docs/plans/active/2026-10-18-org-director-liveness.md`(8b-2a、headless の director の確かめと start の片付け)
- Verifier: verifier subagent(Claude Code、pipeline cycle 1)
- Scope: `git diff 88e37b1d...HEAD`(HEAD 3f5e8409、49 ファイル、+6912/-330)。AC1〜AC9、計画の Verify plan、Design decisions D1〜D9、Non-goals、Progress の逸脱の記録(check-in の starter の読みを `director.lock` の外に出した L1、ファイルを分けた L3)。仕様 `docs/specs/2026-10-07-org-multi-org-director.md` の FR-6、FR-9、FR-10、ユーザーストーリー 3、FR-9 の 8b-2a の注、FR-10 の 8b-1 の注。self-review の報告(再レビューの節を含む)と実機の evidence `docs/evidence/director-liveness-livefire-2026-10-18.md`
- Evidence: `docs/evidence/verify-2026-10-19-org-director-liveness.log` と、run-static-verify.sh が書いた `docs/evidence/verify-2026-10-18-201400.log`(どちらも `docs/evidence/*.log` の gitignore でコミットしない生の出力)

## Spec compliance

テストの実行は `/test` の範囲なので、ここではコードとテストの中身を読み、テストがあることと、そのテストが何を固定しているかを確かめた。

| Acceptance criterion | Status | Evidence |
| --- | --- | --- |
| AC1 starter | 満たす | `StartDirector` は herdr でも launch_cmd でも、`AttachHeadless` の前に `directorStarter` で自分の pid と起動時刻を読む(`internal/org/director_start.go:900`、`:952`)。読めない、alive でない、起動時刻が空のときは両方とも 0 と空で、start は進む。`AttachHeadless` は片方だけの組と負の pid を、読み書きの前に断る(`:158` から)。fold は headless の `attached` にだけ、正の pid と空でない起動時刻の組で読む(`internal/org/director.go:741`)。テストは `TestDirectorAttachHeadless_RecordsTheStarterOnTheAttachedOfEitherLaunch`、`TestDirectorRead_FoldsTheStarterOfAHeadlessAttachedOnly`、`TestStartDirector_RecordsItsOwnProcessAsTheStarter`、`TestStartDirector_LaunchCmdRecordsItsOwnProcessAsTheStarter`、`TestStartDirector_ARefusedStartReadsNoProcess` |
| AC2 check-in | 満たす | `MarkCmdLaunched` はロックの前に登録を読んで同じ検査を通し、starter を `directorCheckInStarterEnded`(`director_start.go:484`、gone・zombie・起動時刻の違いを ended と読む `directorProcessEnded`)で読む(`:444`)。ロックの下では検査をやり直し、読んだときと同じ starter が残っていれば `ErrDirectorCheckInRefused` で断り、何も書かない(`:459`)。ps の失敗、知らない state、starter なしは受け付ける。CLI は `orgDirectorCheckInReadProcess` を `ReadStarter` に渡し(`internal/cli/org_director_headless.go:774`)、断りは error として返るので exit 1 になる。D2 の「ロックの下で読む」からの逸脱は Progress(計画 173 行)と仕様の 8b-2a の注に理由とともに書いてある。テストは `TestDirectorMarkCmdLaunched_RefusesACheckInOnceTheStartThatRecordedItHasEnded`、`..._ReadsTheStarterWithTheDirectorLockFree`、`..._UsesTheStarterItReadOnlyWhileTheRegistrationStillRecordsIt`、`..._ReadsTheStarterOnlyAfterItsOtherChecks`、`TestStartDirector_ACheckInRefusedForItsStarterRunsNoAgent` |
| AC3 確かめ | 満たす | `CheckDirector`(`internal/org/director_check.go:266`)と `checkDirectorStarter`(`:326`)、`checkDirectorAgent`(`:369`)、`checkDirectorProcess`(`:400`)。D3 の表の行はすべて how の定数があり(`:51`〜`:114`)、`TestCheckDirector_StartingRows`、`..._HerdrRunningRows`(label の不一致は `pane_unconfirmed`、`agent get` の失敗は `agent_get_failed`)、`..._CmdRunningRows`、`..._NoAgentGet`、`..._ChangedOnBothTriesIsUnknown` が偽の herdr と偽の ps で state と how を固定する。昇格したセッションと stopped は `..._NothingToCheck`。確かめの途中で `launched` が増える場合は `TestCheckDirector_LaunchedAfterTheReadIsNotReadAsStartEnded` が herdr と launch_cmd の両方を `directorCheckAfterRead` の差し込みで固定する。結果は版 A の登録と `directorVersion` とともに返る |
| AC4 見張り | 満たす | `cycle()` は期限 → `checkDirector` → `wake(check)` → パルス → 印の順(`internal/org/watch.go:669`〜`:680`)。`checkDirector`(`:693`)は `Checked` のときだけ `director-alive.json` を書き(`:702`)、失敗は 1 行で nil を返し、起こす手順は確かめなしで進む。確かめの herdr 呼び出しは `agent get` と `pane/tab/workspace get` だけで、`director.lock` は読み直しにだけ取る(`readDirectorVersion`)。テストは `TestRunWatchAll_ChecksTheDirectorAfterTheDeadlinesAndBeforeTheWakeAndThePulse`、`..._TheDirectorCheckWritesOnlyItsMark`(director.jsonl が変わらない、herdr は get だけ)、`..._WritesNoDirectorMarkWithoutACurrentHeadlessRegistration`、`..._ADirectorCheckThatFailsIsOneLineAndTheWakeAndThePulseGoOn`、`..._ADirectorMarkThatCannotBeWrittenIsOneLineAndTheCycleGoesOn` |
| AC5 起こす手順 | 満たす | `WakeDirector` の手順 3(`internal/org/director_wake.go:367`)が、同じサイクルの確かめが同じ id と `attached_at` の登録を ended と読んだときに `DirectorWakeSkippedEnded` で何も送らず、`warnOnce` で登録ごとに 1 回だけ stderr に書く。`woken` は送ったあとにしか書かれないので書かれない。unknown と alive、別の登録の確かめ、nil は今までどおり送る(`directorCheckFoundEnded`、`:428`)。テストは `TestWakeDirector_ARegistrationTheCheckFoundEndedIsNotWoken`、`..._AnUnknownCheckWakesAsBefore`、`..._AnEndedCheckOfAnotherRegistrationWakesAsBefore` |
| AC6 start の片付け | 満たす | `checkDirectorStart` は見張りの検査の前に `checkCurrentDirector` を呼ぶ(`director_start.go:1036`)。headless なら `CheckDirector` をその場で打ち、alive と unknown は `DirectorAttachedError` に `Check` を付けて断り、昇格したセッションは確かめずに断る(`internal/org/director_start_cleanup.go:31`)。ended は `cleanUpEndedDirector`(`:90`)が版 A の登録だけに `closeHerdrDirector`(label の照合のあとで C-c と close)か `stopDirectorProcess` を打ち、`MarkStoppedAtVersion`(`director_start.go:591`、ロックの下で最初に版を比べる `:619`)で記録してから `Notice` に 1 行を出す。記録まで届かなければ `DirectorCleanupError` で断り、StopCommand が `stop --id` を言う。テストは `TestStartDirector_CleansUpAnEndedHerdrDirectorAndStarts`、`..._CleansUpAnEndedLaunchCmdDirectorAndStarts`、`..._CleansUpARegistrationWhoseStartDied`、`..._RefusesADirectorItsCheckFindsAliveOrCannotTell`、`..._ACleanupThatCannotFinishRefusesStart`、2 つの start の重なりは `..._AStartThatFinishedDuringTheCheckIsNotCleanedUp`(herdr と launch_cmd、signal も close も d2 もない)、版 A だけに打つことは `..._TheCleanupActsOnTheRegistrationTheCheckRead` と `..._ARegistrationThatChangesBeforeTheRecordIsNotRecordedStopped` |
| AC7 attach の断りと表示 | 満たす | attach の断りは `attachHeadlessChecks`(`internal/cli/org_director_check.go:69`)が、印が今の登録のもの(id と `attached_at`)なら印の結果と時刻を、`starting` で starter があれば `CheckDirectorStarter` の結果を足し、どちらかが ended なら start の片付けを案内する。herdr は打たず、何も書かない。表示は `readOrgDirectorCheck`(`:128`)と `orgDirectorWatchCheckClause`(`:153`)で、`ralph status` と `ralph org director status` の両方が `orgDirectorLine` を通る。`--json` は `check`(4 キー、`omitempty`)。テストは `TestDirectorLine_ShowsTheWatchsCheckOfAHeadlessDirector`(印なし、別の id、別の `attached_at`、alive、ended、unknown、制御文字、壊れた印、両方の `--json`)、`TestDirectorLine_APromotedSessionIsNotChecked`(8a-1 の文と byte で一致)、`TestOrgDirectorAttach_RefusalAddsTheWatchsMark`、`..._RefusalAddsTheStartersCheck`、`..._APromotedSessionsRefusalIsUnchanged` |
| AC8 文書 | 満たす(計画の文に情報 4 件) | skill の 4 面は director と org の両方で cmp が一致し、director skill に「確かめを読む」の節がある(`.claude/skills/director/SKILL.md:306`)。`check-skill-sync.sh` と `check-sync.sh` は通る。help(attach、start、launched、director status、status、watch の Short と Long)、README、仕様の注 2 つ、tech-debt(209・247・249・254 の書き換え、255 の行は解決の注と取り消し線と「Still open」に、codex の daemon と別の herdr server の行、ファイルの大きさの行)、AGENTS.md が新しい動きを書いている。計画の文に V1〜V4 がある(下の Findings) |
| AC9 テストの隔離と実機 | 満たす | 新しいテストの herdr は偽物か `setupOrgStubPATH` の stub で、`stubHerdrAgentNotFound` は PATH の herdr がこのテストの一時 dir の下になければ `t.Fatalf` する(`internal/cli/org_director_cleanup_test.go:35`〜`:41`)。osascript、tmux、Orca、gh、network を打つ `exec` はない。本物の ps はテスト自身の pid と、テストが起こして待った子(`deadChildPID`、`startTestProcess` の sleep)の読みにだけ使う。signal はテストが起こした sleep にだけ届く(偽の ps の表で alive かつ起動時刻が一致する pid は、テストが起こしたプロセスだけ)。台帳はすべて `t.TempDir()`。実機は D9 の (a) と (b) を分離した herdr server と隔離した台帳で行い、evidence は `$HOME` を `~`、token と pane と pid を伏せている(16 桁の 16 進は人の herdr ファイルの sha256 の頭で、過去の evidence と同じ扱い) |

仕様の確かめ:

- ユーザーストーリー 3 と FR-6: サイクルの最初は今までどおり `SweepInboxDeadlines` で、確かめはそのあとに来る。`director_check.go` は受信箱を読まない(`PrintableInboxText` のほかに inbox の参照がないことを grep で確かめた)。期限の送りは変わらない
- FR-9: start と stop の今の道は残り、start の最初の確かめだけが増えた。`StopDirector` から `closeHerdrDirector` を取り出した変更は、`recordDirectorStop` の呼び出しも含めて、前の本体と同じ順に打つ
- FR-10: 起こし口は ended の登録に送らない(AC5)
- FR-9 の 8b-2a の注は D1〜D8、L1 の逸脱、非ゴール、実機の evidence を書いている。FR-10 の 8b-1 の注は 8b-2 を 2 本に分けたことを書いている

## Static analysis

| Command | Result | Notes |
| --- | --- | --- |
| `./scripts/run-static-verify.sh`(changed scope) | pass(rc 0) | 選ばれた言語は golang。shellcheck、hook の `sh -n`、check-sync、check-pipeline-sync、check-skill-sync、template purity、tech-debt の plan 参照、gofmt ok、golangci-lint 0 issues、secret scan(88e37b1d..3f5e8409、clean) |
| `go vet ./...`(go1.26.0) | pass(rc 0) | |
| `GOTOOLCHAIN=go1.25.8 go build ./... && go vet ./...` | pass(rc 0) | go.mod の Go 1.25.8 |
| `./scripts/check-skill-sync.sh` | pass | 14 skill が lock-step |
| `./scripts/check-sync.sh` | pass | DRIFTED 0、KNOWN_DIFF 5、TEMPLATE_ONLY 11 |
| `git diff --check 88e37b1d...HEAD` | pass(出力なし) | |

## Findings

CRITICAL、HIGH、MEDIUM はない。どれも計画の文で、コードと出力は変わらない。

| ID | Severity | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- |
| V1 | LOW | 計画の Progress の S5 の行は、tech-debt の 255 行の「残り(no_starter などの unknown)を新しい行にした」と書くが、新しい行はない。残りは同じ行(今の 256 行)の Debt セルに `Still open:` として書かれている | 計画 170 行。`git show 5e76f820 -- docs/tech-debt/README.md` で足された行は、解決の注、取り消し線の行、codex の daemon の行、別の herdr server の行だけ。256 行の Debt セルに `Still open:` がある | `/sync-docs` で「残りを同じ行の Debt セルに Still open として書いた」に直す |
| V2 | INFO | AC1〜AC9 のチェックボックスが `[ ]` のまま | 計画 90〜98 行 | `/sync-docs` か `/pr` で、この報告の判定に合わせて付ける |
| V3 | INFO | D2 の本文は今も「ロックの下で登録の starter を読み」と書く。逸脱は Progress と仕様の注にあるが、D2 だけを読むと今の実装と違う | 計画 67 行と 173 行。`director_start.go:444` の読みはロックの外 | D2 の末尾に「self-review L1 でロックの外に移した(Progress)」と足すかを `/sync-docs` が決める |
| V4 | INFO | 計画が名指す tech-debt の行番号は base のもので、255 行の前に解決の注の行が 1 行入ったので、それより下は 1 行ずれた(255 → 256、270・271 → 271・272)。tech-debt の 256 行の `Still open` が挙げる unknown の how 7 つには、同じく start を断らせる `no_agent_get` と `changed` が入っていない | 計画 33 行、113 行、142 行。`git show 88e37b1d:docs/tech-debt/README.md` の 270・271 行と今の 271・272 行が同じ行。`changed` はもう一度 start を打てば消え、`no_agent_get` は `AgentGet` を持たない herdr の client でだけ起きる | 行番号は計画を archive する前に直すか、そのままにするかを決める。how の列挙は直さなくても誤りにはならない |

## Documentation drift

| Doc / contract | In sync? | Notes |
| --- | --- | --- |
| 「registration does not tell whether its session is still running」 | 合う | 残りは昇格したセッションを指す help(`internal/cli/status.go:76`、`internal/cli/org_director.go:292`)、確かめない登録の文(`org_director.go:627`、昇格したセッションと stopped のまま今の登録)、テスト、過去の evidence だけ。headless の director の行にこの文が出ないことは `TestDirectorLine_ShowsTheWatchsCheckOfAHeadlessDirector` が固定する |
| tech-debt 247 行 | 合う | 昇格したセッションの分に絞り、8b-2a が headless の director を確かめることと、8b-2b に残す分を書き分けた |
| tech-debt 255 行(今の 256 行) | 合う(V1 は計画の文) | 解決の注(S1〜S4 のコミット)、取り消し線、`Still open:` の unknown の分、Trigger の取り消し線と残りのきっかけ。セルは 5 つ |
| tech-debt 209・249・254 行 | 合う | 209 と 249 のきっかけは 8b-2b に移り、254 は start の片付けが `start_ended` で閉じるものがないときの label の案内を書く |
| tech-debt 303〜305 行(新規) | 合う | codex の daemon、別の herdr server、`director_start.go` 1203 行と 800 行を超えるテスト 2 つ(self-review R1)。どれも 5 セル |
| `ralph org director start` の help | 合う | 確かめ、ended の片付け、1 行の通知、片付けが終わらないときの exit 1 を書く(`org_director_headless.go:137` から) |
| attach、launched、director status、status、watch の help | 合う | attach は断りに足す 2 つの確かめと「片付けない」、launched は starter が終わったときの断り、status と director status は `checked by the watch at` と `not checked yet` と `check`、watch はサイクルの順 |
| README、AGENTS.md | 合う | README は status の行、watch の行、director の段落、start の片付け。AGENTS.md は `director_check.go` と `director_start_cleanup.go` |
| 仕様の注 | 合う | FR-9 の 8b-2a の注と FR-10 の 8b-1 の注 |
| `/director` と `/org` の skill の 4 面、recipe の 2 面 | 合う | cmp がすべて一致。how の一覧はコードの定数と一致する |

## Observational checks

- 80dd2982(L3、ファイルを分けたコミット)は、移動元 3 ファイルで消えたコメント以外の行と、新しい 2 ファイルに足された行を sort して比べ、違いは `package` と `import` の行だけだった
- 実機の binary は 0b9b7ed3(S4)。それより後の Go の変更のうち、`director_check.go`、`watch.go`、`driver/herdr.go` はコメントだけで、`director_start.go` の動きの変更は launch_cmd の check-in(実機の範囲の外)だけ、残りは移動だった。evidence はこの SHA を書いている
- `git status` は検証の前もあとも clean(gitignore の evidence の log だけが増えた)

## Coverage gaps

- テストは走らせていない(`/test` の範囲)。AC の「満たす」は、実装とテストの中身を読んで照らした結果
- 実機は herdr の codex director だけで、launch_cmd の director(`process_ended`、`pid_reused`、断られる遅い check-in)、push の起こし口の飛ばし(D5)、2 つの start の重なり、unknown の行は、evidence も書くとおりテストだけが見ている
- `ps -o stat=,lstart=` を procps(Linux)で読むことは手元で確かめていない(計画の Assumptions)
- テストは本物の ps を自分の pid と自分の子にだけ打つ。AC9 は ps を禁じていないが、ps のない環境ではそのテストが落ちる
- insight event は、このコミットに入れない指示のため `docs/insights/events/` に追記していない。`--events-dir` で一時の dir に作った行は次のとおり: `{"schema":1,"ts":"2026-10-18T20:20:33Z","slug":"org-director-liveness","flow":"standard","phase":"verify","cycle":1,"verdict":"pass","findings":{"critical":0,"high":0,"medium":0,"low":0},"triage":{"action_required":0,"worth_considering":0,"dismissed":0},"source":"skill"}`

## Verdict

- Verdict: pass(CRITICAL、HIGH、MEDIUM はない。LOW 1 件(V1)と情報 3 件(V2〜V4)はどれも計画の文で、`/sync-docs` で直せる)
- Verified: AC1〜AC9 のコードとテストの突き合わせ。D1〜D8 と実装の一致(D2 の逸脱は記録あり)。静的解析(run-static-verify、go1.26.0 の vet、go1.25.8 の build と vet、check-skill-sync、check-sync、diff --check)。skill と recipe の写しの一致。文書のずれの 3 項目(残った文、tech-debt の書き分け、start の help)。80dd2982 が移動だけであること。実機の binary より後の変更の範囲
- Partially verified: AC9 の実機(herdr の codex director の (a) と (b) だけ。launch_cmd と push は実機なし)
- Not verified: テストの実行と `-race`、Linux の ps、Claude Code の director、launch_cmd の director の実機
