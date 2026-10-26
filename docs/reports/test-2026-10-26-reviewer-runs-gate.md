# Test report: reviewer-runs-gate

- Date: 2026-10-26(JST 23:50〜翌 00:45)
- Plan: `docs/plans/active/2026-10-26-reviewer-runs-gate.md`(Test plan、AC5、AC8)
- Tester: tester サブエージェント(Claude Opus 5.5)。パイプラインの 1 回目(`cycle-count.json` の cycle 1)
- Scope: HEAD ad1a5c64(base origin/main 633cd7c8)。verify の HEAD c9caf03a のあとのコミットは ad1a5c64 の 1 つで、計画、verify の報告、insight event の 3 ファイルだけを変える。behavioral test だけを実行した(静的解析は /verify の担当)。環境は macOS(Darwin 25.5.0、arm64)、手元の go1.27.3(`GOTOOLCHAIN=local`)と go1.25.8(`GOTOOLCHAIN=go1.25.8`、go.mod と CI の版)
- Evidence: `docs/evidence/test-2026-10-26-reviewer-runs-gate.log`(各実行の要約、coverage、mutant と canary の結果、shell の 44 本の出力すべて)。Go の `-v` の生の出力と coverage のファイルは worktree の `.harness/state/test-rrg/` に、shell の出力は `.harness/state/test-rrg-shell.out` に置いた。どちらも gitignore の対象で、コミットしない
- `./scripts/run-test.sh` は 1 回で通しては打たず、2 つに分けた。shell の部分は `env -u GITHUB_BASE_REF HARNESS_VERIFY_MODE=test RALPH_VERIFY_SCOPE=changed ./scripts/verify.local.sh` を Bash の `run_in_background` で走らせ、`.harness/state/test-rrg-shell.meta` の終わりの行を前面の短いコマンドで待った。Go の部分は、golang の pack が打つ `go test ./...` の代わりに、下の 1〜10 を `-count=1 -timeout 30m` で 1 つずつ前面で打った
- 追跡しているファイルは、この報告のほかは変えていない。mutant は scratchpad の写しを `go test -overlay` で差し替えた。終わったあと `git status --porcelain` は空で、無視されるファイルに増えたのは自分の出力(`.harness/state/test-rrg*` と evidence の log)だけだった。`herdr`・`agmsg`・`ralph-org` の名前のプロセスは残っていない。`go env GOENV` のファイル(`~/Library/Application Support/go/env`)はない。Go の出力に `go: downloading` の行はない

## Test execution

1 分の load average は、実行のあいだ 4.0〜5.5 だった(ほかのセッションの負荷)。

| # | Suite / Command | Tests(トップレベル) | Passed | Failed | Skipped | rc と時間 |
| --- | --- | --- | --- | --- | --- | --- |
| 1 | `./internal/org/ -run '^Test[A-N]'`(go1.27.3、`-coverprofile`) | 390 | 390 | 0 | 0 | ok、86 秒 |
| 2 | `./internal/org/ -run '^Test[^A-N]'`(go1.27.3、`-coverprofile`) | 807 | 807 | 0 | 0 | ok、444 秒 |
| 3 | `./internal/cli/`(go1.27.3、`-coverprofile`) | 826 | 826 | 0 | 0 | ok、430 秒 |
| 4 | 残りのパッケージ(config、insights、org/driver、org/protocol、scaffold、upgrade。go1.27.3、`-cover`) | 6 パッケージ | 6 | 0 | - | ok、6 秒 |
| 5 | `./internal/org/ -run '^Test[A-N]'`(go1.25.8) | 390 | 390 | 0 | 0 | ok、83 秒 |
| 6 | `./internal/org/ -run '^TestR'`(go1.25.8) | 202 | 202 | 0 | 0 | ok、342 秒 |
| 7 | `./internal/org/ -run '^Test[^A-NR]'`(go1.25.8) | 605 | 605 | 0 | 0 | ok、85 秒 |
| 8 | `./internal/cli/ -run '^TestOrg'`(go1.25.8) | 379 | 379 | 0 | 0 | ok、360 秒 |
| 9 | `./internal/cli/ -skip '^TestOrg'`(go1.25.8) | 447 | 447 | 0 | 0 | ok、66 秒 |
| 10 | 残りのパッケージ(4 と同じ 6 つ、go1.25.8) | 6 パッケージ | 6 | 0 | - | ok、5 秒 |
| 11 | shell の部分(`verify.local.sh` の test モード、`tests/test-*.sh`) | 44 本 | 44 | 0 | 1 件(下) | rc 0、656 秒(23:50:51〜00:01:47) |
| 12 | 名指しの確かめ(下の「名指しのテスト」、go1.27.3、`-v`) | 62 | 62 | 0 | 0 | ok、37 秒 |

- go1.25.8 では、リードの分け方より細かく分けた。go1.27.3 の 2 が 444 秒かかり、そのうち 365 秒が `TestRegate_*` などの `TestR` で始まるテストだった。3 も 430 秒かかった。版ごとの実行が Bash の 600 秒の上限を越えないよう、org を 5・6・7 に、cli を 8・9 に分けた。org の 6 と 7 の合計 807 本は 2 と同じ数、cli の 8 と 9 の合計 826 本は 3 と同じ数で、取りこぼしはない
- 2 つの版の実行は並べず、1〜4 を終えてから 5〜10 を打った。11 は 1 より前に走らせ、11 のあいだは 12 と mutant の m0〜m5、c1〜c4(1 回が数秒)だけを打った。m6 と canary の組は 10 のあとに打った
- 既知の不安定なテスト `TestStopDirectorProcess_AProcessThatIgnoresSIGTERMIsNotEndedInTime` は、go1.27.3(0.50 秒)と go1.25.8(0.48 秒)のどちらでも通った。落ちていないので、単独の打ち直しはしていない
- 11 の 44 本は `tests/test-*.sh` の数と同じで、どの本の要約にも 0 でない失敗の数はない。Skipped は前からある `tests/test-secret-scan-branch.sh` の 1 件(git 2.49.0 なので「2.41 より古い本物の git」の事例を飛ばし、stub の古い git がその分岐を見る)

### 名指しのテスト(12)

リードが挙げたテストはすべて通った。

- `internal/org/prompts_test.go`: `TestRenderRolePrompt_Leader_FeatureOrgAsksTheReviewerForTheGate`、`TestRenderRolePrompt_Leader_DoesNotRunTheGateItself`、`TestRenderRolePrompt_Leader_FeatureOrgGate`、`TestRenderRolePrompt_Reviewer_GateTaskRunsTheTaskCommand`。reviewer の例の RESULT と BLOCKED を `protocol.Parse`・`Validate`・`HeaderValues` に通す検査は、最後のテストの中にある(`prompts_test.go:1640` から)
- `internal/org/permissions_test.go`: `TestCodexWritableRootArgs`(25 行)と `TestCodexWritableRootArgs_FeatureOrgReviewerGetsAddDir`
- `internal/org/spawn_test.go`: `TestOrgSpawn_Codex_WorkspaceWrite_AddDirWhenStateDirOutsideCwd` と `TestOrgSpawn_Codex_WorkspaceWrite_FeatureOrgReviewerGetsAddDir`
- regate の断りの文: `TestRegate_RefusalOfARunningOrgNamesTheBranchItRuns`(`regate_test.go:585`。文を字面どおり固定している)と `TestRegate_RefusedBeforeTheFetch`
- `leaderActivityEventCount` の既存のテスト: `TestLeaderActivityEventCount_GateEvents`(`gate_test.go`)、`TestLeaderActivityEventCount_StopFailed`、`TestPlanApprovedEvent_IsNotASeat`、`TestPRMerged_IsNotASeatAndDoesNotRun`、`TestWatch_Deadman_*`(19 本)
- 同じ組で、`TestRenderRolePrompt_*` のほかのテストと `TestGateWording_RefusalsThatAskForACommit` も通った

## AC5 の mutant

### 雛形の文(`go test -overlay`、追跡しているファイルは変えていない)

`internal/org/prompts/leader.md` と `reviewer.md` の写しを scratchpad に置き、新しい文を 1 つずつ消して差し替えた(go:embed のファイルも差し替わる)。消すところは、写しの中に 1 回だけ現れる始めと終わりの字面で決めた。対象は雛形を読む 5 つのテストファイル(`prompts_test.go`、`prompts_leader_inbox_test.go`、`director_skill_sync_test.go`、`org_skill_send_sync_test.go`、`report_names_sync_test.go`)のトップレベルの 60 本で、どの実行でも 60 本が走った。verify が `python3 -I` の写しで確かめた 4 つ(`gate-<n>` の文、leader が打たない文、停滞の ALERT の文、手順 14 の spawn し直しの文)とは別の文を選んだ。

| mutant | 消した文 | rc | 落ちたテスト(場所) |
| --- | --- | --- | --- |
| m0(対照) | 変えない写し(leader.md と reviewer.md のそれぞれ) | 0 | なし |
| m1 | leader.md 手順 12 の「この org では implementer と reviewer を手順 13 まで止めない(…)」の文 | 1 | `TestRenderRolePrompt_Leader_FeatureOrgAsksTheReviewerForTheGate`(`prompts_test.go:1123`、消した文を missing と名指す) |
| m2 | reviewer.md の「gate の TASK で台帳に書いてよいのは…打たずに QUESTION で leader に聞いてください」の段落 | 1 | `TestRenderRolePrompt_Reviewer_GateTaskRunsTheTaskCommand`(`prompts_test.go:1579`) |
| m3 | leader.md 手順 12 の「同じ理由(…)で gate の TASK が 2 回続けて通らなければ…BLOCKED を上げる」の文 | 1 | `TestRenderRolePrompt_Leader_FeatureOrgAsksTheReviewerForTheGate`(`prompts_test.go:1123`、消した文を名指す) |
| m4 | reviewer.md の「gate の TASK では、TASK のコマンドを打つほかに worktree のファイルを書き換えず、コミットもしない(…)」の項 | 1 | `TestRenderRolePrompt_Reviewer_GateTaskRunsTheTaskCommand`(`prompts_test.go:1596`) |
| m5 | leader.md 手順 12 の「送る前に `ralph org status` を打ち、reviewer が動いていなければ…spawn し直してから送る」の文 | 1 | `TestRenderRolePrompt_Leader_FeatureOrgAsksTheReviewerForTheGate`(`prompts_test.go:1123`、消した文を名指す) |
| m6 | reviewer.md の gate が通ったときの RESULT の例から `GATE: pass` のヘッダの行 | 1 | `TestRenderRolePrompt_Reviewer_GateTaskRunsTheTaskCommand`(`prompts_test.go:1651`、「one GATE: header ("pass"), got []」) |

m1、m3、m5 は同じ検査(手順 12 の文を順に探す表、`prompts_test.go:1123`)で落ちる。この検査は最初に見つからない文で止まり(`break`)、3 つともその文を消した文として名指した。

### spawn の引数(S1、`go test -overlay`)

`internal/org/permissions.go` と `internal/org/spawn.go` の写しの 1 か所を変えた。対象は `-run '^(TestCodexWritableRootArgs|TestOrgSpawn_Codex_)'`(トップレベル 17 本)。

| mutant | 変更 | 落ちたテスト(場所) |
| --- | --- | --- |
| c1 | `(featureOrg && role == reviewerRole)` を `(role == reviewerRole)` にする(結びつきのない org の reviewer にも付く) | `TestCodexWritableRootArgs` の結びつきのない org の reviewer の 2 行(`permissions_test.go:247`)、`TestOrgSpawn_Codex_WorkspaceWrite_FeatureOrgReviewerGetsAddDir` の同じ 2 つ(`spawn_test.go:3902`) |
| c2 | 同じ所を `(featureOrg && false)` にする(機能の org の reviewer に付かない) | `TestCodexWritableRootArgs_FeatureOrgReviewerGetsAddDir` の台帳が cwd の外の 2 行(`permissions_test.go:302`)、spawn の argv のテストの機能の org の reviewer の 2 つ(`spawn_test.go:3902`) |
| c3 | `spawn.go` の `ActiveFeature(events, p.OrgID) != nil` を `== nil` にする | spawn の argv のテストの 4 つ(`spawn_test.go:3902`) |
| c4 | 同じ所に `&& false` を足す(spawn が機能の org と見ない) | spawn の argv のテストの機能の org の reviewer の 2 つ(`spawn_test.go:3902`) |

c3 と c4 は表のテストでは落ちず、argv のテストだけが落とす。spawn が `featureOrg` を渡すところは argv のテストだけが見ている、ということで、穴ではない。

## 外の道具を打たないこと(canary)

`herdr`、`agmsg`、`osascript`、`gh`、`tmux`、`orca`、`Orca`、`codex`、`claude` の名前で、呼ばれると log に 1 行書いて exit 97 で終わる script を作り、PATH の先頭に置いた。

- 感度の確かめ: `-overlay` で足した使い捨てのテスト(`gh --version` を打つだけ、追跡しているパスにはないファイル)を同じ PATH で走らせると、log に 1 行が残った
- org の名指しの組(上の 12 に `TestPermissionArgsForDriver_*`、`TestDirectorSkill*`、`TestOrgSkill*` を足したトップレベル 87 本): すべて PASS、log は 0 行
- cli の `-run '^TestOrg(Gate|Regate)'`(24 本): すべて PASS、log は 0 行

canary を置いたのはこの 2 つの組だけで、1〜10 の全体の実行には置いていない。

## Coverage

- Statement: go1.27.3 の実行で計った。`internal/org` は 1 と 2 の profile を合わせて 95.0%(mode は set で、2 つのファイルの block の並びは同じ 5,754 行。block ごとに大きい方を取った)。`internal/cli` 88.4%、`internal/config` 96.9%、`internal/insights` 86.4%、`internal/org/driver` 95.2%、`internal/org/protocol` 100.0%、`internal/scaffold` 75.7%、`internal/upgrade` 91.2%
- 差分の関数: `codexWritableRootArgs`(`permissions.go:198`)100.0%、`leaderActivityEventCount`(`watch.go:1386`)100.0%、`Spawn`(`spawn.go:585`)92.5%。差分のうちコメントでない行は、`permissions.go:198-200`(引数と条件)、`spawn.go:1050-1051`、`regate.go:481-483`(断りの文)、`approval.go` の定数 `reviewerRole` だけで、profile ではこれらを含む block がすべて 1 回以上通っている。`watch.go`、`gate.go`、`regate_merge.go` の差分と `permissions.go` のほかの塊はコメントだけ。`internal/cli/org_gate.go` と `org_regate.go` の差分は help の文で、`TestOrgGateHelp` が固定している
- Branch: 計っていない(Go の標準の道具にない)
- Function: 上の 3 つ
- Notes: 雛形の文は coverage に出ない。上の m1〜m6 と verify の 4 つで、文を消すとテストが落ちることを確かめた

## Failure analysis

| Test | Error | Root cause | Proposed fix |
| --- | --- | --- | --- |
| なし | - | - | - |

## Regression checks

| Previously broken behavior | Status | Evidence |
| --- | --- | --- |
| 機能の org の leader が gate を自分で打つ文(origin/main の leader.md) | 戻すと落ちる。`TestRenderRolePrompt_Leader_DoesNotRunTheGateItself` が通り、verify の写しの probe では origin/main の leader.md が 8 件で落ちた | 12、verify の報告の Observational checks |
| `leaderActivityEventCount` の数え方(D6 で変えないと決めた) | 変わらず通る。関数の coverage は 100.0% | 12 の `TestLeaderActivityEventCount_GateEvents` ほか、1〜10 |
| 既知の不安定 `TestStopDirectorProcess_AProcessThatIgnoresSIGTERMIsNotEndedInTime`(`docs/tech-debt/README.md:283`) | 2 つの版のどちらでも通った | 1 と 5(evidence の 1 節) |

## Test gaps

1. 実機での確かめはない。reviewer が herdr の上で gate の TASK を受けて `ralph org gate` を打つこと、codex の reviewer が `--add-dir` で台帳に書けること、leader が gate の結果を雛形どおりに扱うことは、本物の herdr、agmsg、codex、claude が要る。計画の Non-goals、R6、D8 のとおりで、FR-7 は `[ ]` のまま
2. 雛形のテストは文があるかと順を見るだけで、モデルがその文に従うかは見ない
3. canary の PATH を置いたのは名指しの組(org 87 本、cli 24 本)だけで、パッケージ全体には置いていない
4. `./scripts/run-test.sh` は通しでは打っていない。shell の部分は `verify.local.sh` を直接打ち、Go の部分は golang の pack の `go test` の代わりに版ごとに分けて打った。wrapper の言語の選び方(scope が changed のときに golang の pack を選ぶこと)はこの実行では通っていない。verify の報告では `run-static-verify.sh` が同じ選び方で golang の pack を選んでいる
5. AC8 の `./scripts/run-verify.sh` の静的な半分は、verify が c9caf03a で確かめた。ad1a5c64 での静的解析は打っていない(/verify の担当で、c9caf03a からの差分は文書の 3 ファイル)

## Verdict

- Pass: pass。HEAD ad1a5c64 で、Go のテストは go1.27.3(1〜4)と go1.25.8(5〜10)の両方で、テストのある 8 パッケージがすべて ok だった(`internal/org` 1,197 本、`internal/cli` 826 本、落ちたものなし)。shell の 44 本も rc 0(656 秒)。AC5 は、雛形の文の mutant 6 つと spawn の引数の mutant 4 つがすべて落ち、対照の 2 つは通った。名指しのテストと canary の組は、外の道具を 1 回も呼ばずに通った
- Fail: なし
- Blocked: なし
