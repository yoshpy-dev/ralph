# Test report: org-narrow-verification

- Date: 2026-10-28(JST 05:10〜06:30)
- Plan: `docs/plans/active/2026-10-27-org-narrow-verification.md`(Test plan、AC3、AC6)
- Tester: tester サブエージェント(Claude Opus 5.5)。パイプラインの 2 回目(`cycle-count.json` の cycle 2)。1 回目の報告(3afdd03a)をこの報告で置き換える
- Scope: HEAD 7458318d(base 622cdcfb)。behavioral test だけを実行した(静的解析は /verify の担当)。1 回目の報告のあとのコードの変更は ffafc9d2、a7d879d9、854be9db、51cf59e2、28d83b96 の 5 つで、変えたのは go:embed の雛形 3 つ(`internal/org/prompts/{implementer,leader,reviewer}.md`)、`/org` skill の 4 つの写し、雛形のテスト。差分の Go のファイルは `internal/org/prompts_verification_test.go`、`prompts_verification_multi_test.go`、`prompts_verification_predate_test.go` の 3 つのテストだけで(`git diff --name-only 622cdcfb..HEAD -- '*.go'`)、テストでない `.go` の変更はない。環境は macOS(Darwin 25.5.0、arm64)、go1.27.3
- Evidence: `docs/evidence/test-2026-10-27-org-narrow-verification.log`(`run-test.sh` の出力すべて、`-count=1` の打ち直し、名指しの実行、mutant ごとの差分と落ちたテストの行、不安定なテストの単独の実行、終わったあとの `git status --porcelain`)。gitignore の対象で、コミットしない
- 追跡しているファイルは、この報告と insight event のほかは変えていない。mutant は scratchpad の写しを `go test -overlay` で差し替え、base の確かめは `git archive 622cdcfb` の写しで行った。`run-verify.sh` が書く `docs/evidence/verify-2026-10-27-201056.log` も gitignore の対象

## Test execution

| # | Suite / Command | Tests | Passed | Failed | Skipped | rc と時間 |
| --- | --- | --- | --- | --- | --- | --- |
| 1 | `./scripts/run-test.sh`(scope changed、pack は golang だけ) | 下の 1a と 1b | 1 本を除いてすべて | 1(1b の `internal/org`) | 1 件(下) | rc 1、3,302 秒(05:10:54〜06:05:56) |
| 1a | 1 の shell の部分(`verify.local.sh` の test モード、`tests/test-*.sh`) | 45 本 | 45 | 0 | 1 件 | 1 に含む |
| 1b | 1 の Go の部分(`go test ./...`、`-timeout=30m`) | テストのある 9 パッケージ | 8 | 1(`internal/org` の `TestStopDirectorProcess_AProcessThatIgnoresSIGTERMIsNotEndedInTime`) | - | `internal/org` 931.1 秒、`internal/cli` 782.6 秒、`internal/config` 1.6 秒、ほかの 6 つは `(cached)` |
| 1c | `(cached)` の 6 パッケージを `-count=1` で打ち直し(`cmd/ralph`、`internal/insights`、`internal/org/driver`、`internal/org/protocol`、`internal/scaffold`、`internal/upgrade`) | 6 パッケージ | 6 | 0 | - | どれも ok、0.9〜4.6 秒 |
| 1d | `go test ./internal/org/ -count=1 -timeout=30m`(1b の失敗のあとの打ち直し) | 1 パッケージ | 1 | 0 | - | rc 0、943.4 秒(06:08:27〜06:24:12) |
| 2 | `go test ./internal/org/ -run 'TestRenderRolePrompt\|TestRolePrompts\|TestRetiredRole\|OrgSkill\|ReportNames\|SendDefaults\|DirectorSkill' -count=1 -v` | トップレベル 72 | 72 | 0 | 0 | ok、17 秒 |
| 3 | `go test ./internal/org/ -run 'TestStartFeature' -count=1` | トップレベル 27(`-v` の打ち直しで数えた) | 27 | 0 | 0 | ok、81.1 秒 |
| 4 | 雛形の mutant 7 つ(対照 1 と mutant 6。下の「雛形の mutant」) | 実行ごとにトップレベル 99 | 対照は 99 | mutant はどれも 1 本 | 0 | 1 回 69〜176 秒 |
| 5 | `go test ./internal/org/ -run '^TestStopDirectorProcess_AProcessThatIgnoresSIGTERMIsNotEndedInTime$' -count=20 -v` を HEAD と base(622cdcfb の `git archive`)で | 各 20 回 | HEAD 14、base 9 | HEAD 6、base 11 | 0 | どちらも rc 1。1 分の load average は HEAD のとき 18、base のとき 22〜24 |

- 1b の `(cached)` のうち、雛形を埋め込む `cmd/ralph` と、`internal/org` を import する `internal/insights`(`go list -deps`)は、同じ入力で前に走った結果を使っていた。1c で `-count=1` で打ち直し、どれも通った。`internal/org` と `internal/cli` は 1b でテストを走らせた
- 1a の 45 本は `tests/test-*.sh` の数と同じで、どの本の要約にも 0 でない失敗の数はない。Skipped は前からある `tests/test-secret-scan-branch.sh` の 1 件(git 2.49.0 なので「2.41 より古い本物の git」の事例を飛ばし、stub の古い git がその分岐を見る)
- 2 の 72 本の内訳は、`TestRenderRolePrompt*` 53、`TestDirectorSkill*` 7、`TestOrgSkill*` 5、`TestRetiredRoles` 3、`TestRetiredRoleConfigKeys`・`TestRolePrompts`・`TestReportNames*`・`TestSendDefaults*` が 1 ずつ。1 回目の 66 本から増えた 6 本は `prompts_verification_predate_test.go` のトップレベルのテストすべて。このブランチの 3 つのテストのファイルのトップレベルのテスト 24 本(16、2、6)は、どれもこの中で通った。`internal/org/prompts_test.go` は base から変わっていない
- 3 の 27 本は、leader の雛形を読む機能の org の start のテスト
- 1 の実行のあいだ、ほかのセッションの iOS Simulator と Xcode のビルドで、1 分の load average が 8.6〜204 まで動いた(05:27 に 136〜204)。mutant(4)は 1 の shell の部分のあいだ(05:14〜05:27)に、2 と 3 は 05:29〜05:31 に走らせた。1d のあいだも、5 分の load average は 06:21 に 75 まで上がった

## 雛形の mutant

`internal/org/prompts/` の雛形の写しを scratchpad に置き、1 か所だけ変えて `go test -overlay` で差し替えた(go:embed のファイルも差し替わる)。変える所は、写しの中に 1 回だけ現れる字面で決めた(2 回以上や 0 回なら止まる)。対象は 2 と 3 を合わせた `-run 'TestRenderRolePrompt|TestRolePrompts|TestRetiredRole|OrgSkill|ReportNames|SendDefaults|DirectorSkill|TestStartFeature'` のトップレベル 99 本で、どの実行でも 99 本が走り、build の失敗はない。落ちたテストの行は、どれも `internal/org/prompts_verification_predate_test.go` のもの。

| mutant | 変更 | rc | 落ちたテスト(場所と、見つからないと報告した字面) |
| --- | --- | --- | --- |
| 対照 | 変えない `implementer.md` と `leader.md` の写し | 0 | なし(99 本が通過) |
| (a) | `implementer.md` の機械的な確かめの項から、cross-review の直しの文「前からあった変更は、stage 済みのものも含めて、書き換えも削除も unstage もしない。」を消す | 1 | `TestRenderRolePrompt_Implementer_SelfCheckLeavesChangesThatPredateTheSlice`(`:103`、消した文) |
| (b1) | 同じ項から「、パスを付けない `git reset` は打たない(前からあった変更の stage まで外れる)」を消し、前の文を「…`git restore --staged -- <このスライスのパス>...` を打つ。」で終える | 1 | 同じテスト(`:103`、「…を打ち」で終わる restore の文) |
| (b2) | 同じ所を「を打ち、パスを付けない `git reset` を打ってもよい。」にする(restore の文は崩さない) | 1 | 同じテスト(`:103`、「パスを付けない `git reset` は打たない(…)」) |
| (c1) | `leader.md` のミッションの 3 で、複数のスライスの `RESUME:` の規則から「各行をそのスライスの名前で始める(例: `RESUME: S1: <パス>...`)」を消し、「1 行ずつ書く。」で終える | 1 | `TestRenderRolePrompt_Leader_ResendsAStoppedSliceWithAResumeLine`(`:220`、複数のスライスの `RESUME:` の文) |
| (c2) | 同じ規則の例だけを `RESUME: <パス>...` にする(スライスの名前で始める、という文は残す) | 1 | 同じテスト(`:220`、同じ文) |
| (d) | `implementer.md` から `RESUME:` の引き継ぎの文「ただし TASK の本文に `RESUME:` の行があれば、…このスライスの変更として引き継ぐ(…機械的な確かめを行う)。」を消す | 1 | `TestRenderRolePrompt_Implementer_TakesOverThePathsOfAResumeLine`(`:264`、消した文) |

- (b) はリードの指定どおり規則を消した b1 のほかに b2 を試した。`SelfCheckLeavesChangesThatPredateTheSlice` は字面を順に探し、最初に見つからない字面で止まる。b1 は文の終わりを「を打つ」に変えたので、その前の restore の字面(「…を打ち」)で落ちた。規則の字面だけで落ちるかを見るため、restore の文を崩さない b2 を足し、`git reset` の字面で落ちることを確かめた
- (c) はリードの指定どおり規則を消した c1 のほかに、例の `S1: ` だけを消す c2 を試した。どちらも同じ字面で落ちる
- 名指しの 4 つ(a、b、c、d)は、どれも 1 本のテストが落とした。生き残った mutant はない
- `/org` skill の写しは変えていない。skill の側を固めるテスト(`TestOrgSkill_ImplementerBulletLeavesChangesThatPredateTheSlice`、`TestOrgSkill_ImplementerBulletAndLeaderCycleCarryTheResumeLine`)はルートの写しをディスクから読むので、overlay では差し替わらない

## Coverage

- Statement: 計っていない。テストでない Go のコードに差分がないので、この差分で変わる statement の coverage はない
- Branch: 計っていない(Go の標準の道具にない)
- Function: 同上
- Notes: 雛形の文は coverage に出ない。代わりに上の mutant で、名指しの文を変えるとテストが落ちることを確かめた。1 回目の mutant(reviewer の既定、leader の `RALPH_VERIFY_SCOPE=changed` の文、implementer の既定、TASK の例の `VERIFY:` の行)は打ち直していない。その文を固めるテストは 2 で通っている

## Failure analysis

| Test | Error | Root cause | Proposed fix |
| --- | --- | --- | --- |
| `TestStopDirectorProcess_AProcessThatIgnoresSIGTERMIsNotEndedInTime`(`internal/org/director_cmd_test.go:310`) | `stopDirectorProcess = {outcome:ended signalled:true note:process=ended after SIGTERM err:<nil>}, want not ended within 300ms` | 前からある不安定なテストで、このブランチの変更が原因ではない。テストは `sh -c "trap '' TERM; exec sleep 30"` を起こして 100 ms 待つだけなので、load が高いと trap より先に SIGTERM が届き、`sh` が終わる。`docs/tech-debt/README.md:285` の行が同じ原因を書いている。このブランチは `director_cmd*.go` も `director_stop*.go` も変えていない。単独の 20 回(5)では HEAD で 6 回、base でも 11 回落ちたので、base でも同じように落ちる。1d の `internal/org` の全体の打ち直しは通った | このブランチでは直さない(計画の scope の外)。tech-debt の行の契機(「次の失敗」)が 3 回目で満たされたので、行の案(`ps` が `sleep` を実行していると示すまで待ってから SIGTERM を送る)で直す別の変更を勧める |

## Regression checks

| Previously broken behavior | Status | Evidence |
| --- | --- | --- |
| cross-review の 1 回目 #1(秘密情報の片付けが前からあった変更を消しうる)の直し | `TestRenderRolePrompt_Implementer_SelfCheckLeavesChangesThatPredateTheSlice` と `TestOrgSkill_ImplementerBulletLeavesChangesThatPredateTheSlice` が通過。(a)、(b1)、(b2) の mutant を前者が落とす | 2、4 |
| 「前からあった変更」の 1 つの名前(古い呼び方が戻らないこと) | `TestRenderRolePrompt_Implementer_RecordsPredatingChangesAtTheStartOfEachTask` が通過 | 2 |
| `RESUME:` の行による止まったスライスの再開(leader、implementer、skill) | predate のファイルの 3 本が通過。(c1)、(c2)、(d) の mutant を落とす | 2、4 |
| gate の TASK の節を変えないこと(計画の Test plan) | `TestRenderRolePrompt_Leader_FeatureOrgAsksTheReviewerForTheGate`、`TestRenderRolePrompt_Leader_DoesNotRunTheGateItself`、`TestRenderRolePrompt_Reviewer_GateTaskRunsTheTaskCommand` が通過 | 2 |
| 機能の org の start が leader の雛形を読むこと | `TestStartFeature*` の 27 本が通過 | 3 |
| `ralph org gate`・`regate`・spawn の既存のテスト(計画の Regression tests) | 1b では上の不安定な 1 本だけが落ちた。1d の `internal/org` の全体の打ち直し(キャッシュなし)は ok | 1b、1d |
| `/org` skill の 4 つの写しと scaffold の同期 | `tests/test-check-skill-sync.sh`、`tests/test-check-sync.sh` が通過。`internal/cli`、`cmd/ralph`、`internal/scaffold` も ok | 1a、1b、1c |

## Test gaps

1. 実機での確かめはない。leader が `VERIFY:`、`CHECKS:`、`RESUME:` の行を書くこと、implementer が `RESUME:` の行のパスを引き継ぎ、前からあった変更に触らずに BLOCKED を返すこと、reviewer が `CHECKS:` の行のコマンドだけを打つことは、本物の herdr、agmsg、codex、claude が要る。計画の Test plan の Integration tests は「なし」
2. 雛形のテストは文があるかと順を見るだけで、モデルがその文に従うかは見ない
3. 新しい binary と古い binary の座席が混ざる場合(古い leader が `RESUME:` の行を書かず、新しい implementer が前の TASK の変更を見て止まる、など)は、雛形の文として確かめただけで、版を混ぜた org は立てていない
4. mutant はリードが名指しした 4 つ(b と c は 2 つの形)だけを試し、`/org` skill の写しは変えていない。skill の側のテストが skill の文の変更で落ちるかは、この報告では確かめていない
5. `TestStopDirectorProcess_AProcessThatIgnoresSIGTERMIsNotEndedInTime` は load が高いと base でも落ちる。この不安定さが残るあいだ、`run-test.sh` の 1 回の実行が rc 1 になることがある
6. statement の coverage は計っていない(テストでない Go に差分がないため)

## Verdict

- Pass: pass。HEAD 7458318d で `./scripts/run-test.sh` は rc 1(3,302 秒)だったが、落ちたのは `internal/org` の `TestStopDirectorProcess_AProcessThatIgnoresSIGTERMIsNotEndedInTime` の 1 本だけだった。このテストは tech-debt に載っている前からの不安定なテストで、base でも 20 回中 11 回落ちる。`internal/org` の全体をキャッシュなしで打ち直すと ok だった(943.4 秒)。shell の 45 本と、ほかの Go の 8 パッケージは通り、`(cached)` の 6 パッケージも `-count=1` で通った。名指しの実行は、雛形の組がトップレベル 72 本、`TestStartFeature` が 27 本で、すべて通った。雛形の mutant は名指しの 4 つ(b と c は 2 つの形)がすべて落ち、対照は通った
- Fail: なし(`run-test.sh` の rc 1 は上の不安定なテストによるもので、このブランチの変更による失敗はない)
- Blocked: なし
