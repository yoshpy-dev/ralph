# Test report: org-narrow-verification

- Date: 2026-10-28(JST 02:33〜03:05)
- Plan: `docs/plans/active/2026-10-27-org-narrow-verification.md`(Test plan、AC3、AC6)
- Tester: tester サブエージェント(Claude Opus 5.5)。パイプラインの 1 回目(`cycle-count.json` の cycle 1)
- Scope: HEAD 065e69b4(base 622cdcfb)。behavioral test だけを実行した(静的解析は /verify の担当)。差分の Go のファイルは `internal/org/prompts_verification_test.go` と `internal/org/prompts_verification_multi_test.go` の 2 つのテストだけで、テストでない `.go` の変更はない(`git diff --name-only 622cdcfb..HEAD -- '*.go' ':!*_test.go'` が 0 行)。振る舞いの変更は go:embed の雛形 3 つ(`internal/org/prompts/{implementer,reviewer,leader}.md`)と、`templates.go` が埋め込む `templates/base/` の skill と文書。環境は macOS(Darwin 25.5.0、arm64)、go1.27.3
- Evidence: `docs/evidence/test-2026-10-27-org-narrow-verification.log`(`run-test.sh` の出力すべて、名指しの 2 つの実行の結果の行、mutant ごとの差分と落ちたテスト、終わったあとの `git status --porcelain`)。gitignore の対象で、コミットしない
- 追跡しているファイルは、この報告と insight event のほかは変えていない。mutant は scratchpad の写しを `go test -overlay` で差し替えた。終わったあと `git status --porcelain` は空だった。`run-verify.sh` が書く `docs/evidence/verify-2026-10-27-173355.log` も gitignore の対象

## Test execution

| # | Suite / Command | Tests | Passed | Failed | Skipped | rc と時間 |
| --- | --- | --- | --- | --- | --- | --- |
| 1 | `./scripts/run-test.sh`(scope changed、pack は golang だけ) | 下の 1a と 1b | すべて | 0 | 1 件(下) | rc 0、1,572 秒(02:33:54〜03:00:06) |
| 1a | 1 の shell の部分(`verify.local.sh` の test モード、`tests/test-*.sh`) | 45 本 | 45 | 0 | 1 件 | 1 に含む |
| 1b | 1 の Go の部分(`go test ./...`、`-timeout=30m`) | テストのある 9 パッケージ | 9 | 0 | - | `internal/org` 455.9 秒、`internal/cli` 393.5 秒、`cmd/ralph` 0.9 秒、ほかの 6 つは 6 秒以下 |
| 2 | `go test ./internal/org/ -run 'TestRenderRolePrompt\|TestRolePrompts\|TestRetiredRole\|OrgSkill\|ReportNames\|SendDefaults\|DirectorSkill' -count=1 -v` | トップレベル 66 | 66 | 0 | 0 | ok、7 秒 |
| 3 | `go test ./internal/org/ -run 'TestStartFeature' -count=1 -v` | トップレベル 27 | 27 | 0 | 0 | ok、26 秒 |
| 4 | 雛形の mutant 7 つ(対照 1 と mutant 6。下の「雛形の mutant」) | 実行ごとにトップレベル 93 | 対照は 93 | mutant はどれも 1〜2 本 | 0 | 1 回 32〜73 秒(c3 は計っていない) |

- 1b の `go test ./...` に `(cached)` の行はない。雛形を埋め込む `internal/org`、`cmd/ralph`、`internal/cli` も、この実行でテストを走らせた。そのため `-count=1` の打ち直しはしていない
- 1a の 45 本は `tests/test-*.sh` の数と同じで、どの本の要約にも 0 でない失敗の数はない。Skipped は前からある `tests/test-secret-scan-branch.sh` の 1 件(git 2.49.0 なので「2.41 より古い本物の git」の事例を飛ばし、stub の古い git がその分岐を見る)
- 2 の 66 本の内訳は、`TestRenderRolePrompt*` 49、`TestDirectorSkill*` 7、`TestOrgSkill*` 3、`TestRetiredRoles` 3、`TestRetiredRoleConfigKeys`・`TestRolePrompts`・`TestReportNames*`・`TestSendDefaults*` が 1 ずつ。このブランチで足した 18 本(2 つの新しいファイルのトップレベルのテストすべて)は、どれもこの中で通った。`internal/org/prompts_test.go` は base の版と同じ内容(`cmp` で一致)で、その中のテストも通った
- 3 の 27 本は、leader の雛形を読む機能の org の start のテスト
- 1 の実行のあいだ、ほかのセッションの iOS と Swift のビルドで 1 分の load average が 4.8〜191 まで動いた(02:44〜02:46 に 64〜191)。2〜4 は 1 の shell の部分のあいだに走らせた。時間に頼るテストが落ちやすい条件だったが、既知の不安定なテストを含め、落ちたものはない

## 雛形の mutant

`internal/org/prompts/` の雛形の写しを scratchpad に置き、1 か所だけ変えて `go test -overlay` で差し替えた(go:embed のファイルも差し替わる)。変える所は、写しの中に 1 回だけ現れる字面で決めた(2 回以上や 0 回なら止まる)。対象は 2 と 3 を合わせた `-run 'TestRenderRolePrompt|TestRolePrompts|TestRetiredRole|OrgSkill|ReportNames|SendDefaults|DirectorSkill|TestStartFeature'` のトップレベル 93 本で、どの実行でも 93 本が走り、build の失敗はない。

| mutant | 変更 | rc | 落ちたテスト(場所) |
| --- | --- | --- | --- |
| 対照 | 変えない `leader.md` の写し | 0 | なし(93 本が通過) |
| (a) | `reviewer.md` のミッションの 1 の既定を静的検査だけにする(「なければ `./scripts/run-static-verify.sh` と `./scripts/run-test.sh` を実行する」から `./scripts/run-test.sh` を外す) | 1 | `TestRenderRolePrompt_Reviewer_ReviewTaskRunsExactlyTheNamedCommands`(`prompts_verification_test.go:245`)、`TestRenderRolePrompt_Leader_MissionPutsTheSliceCommandOnAVerifyLine`(`:458`、reviewer の既定が両方であることの突き合わせ) |
| (b) | `leader.md` の機能ごとの org の手順 1 から「`RALPH_VERIFY_SCOPE=changed` は絞り込みに使わない(…)」の文を消す | 1 | `TestRenderRolePrompt_Leader_FeatureOrgPlanCarriesTheSliceCommands`(`:527`、消した文を名指す) |
| (c1) | `implementer.md` のミッションに前の既定を戻す(「`VERIFY:` の行のコマンド(指示がなければ `./scripts/run-verify.sh`)を実行する」) | 1 | `TestRenderRolePrompt_Implementer_MissionRunsTheSliceCommand`(`:57` と `:61`)、`TestRenderRolePrompt_Leader_MissionPutsTheSliceCommandOnAVerifyLine`(`:452`) |
| (c2) | 同じ項の「TASK に `VERIFY:` の行がなければ、…いちばん狭い検査を選ぶ」を「…なければ `./scripts/run-verify.sh` を打つ」にする(前の字面を使わずに既定を戻す) | 1 | `TestRenderRolePrompt_Implementer_MissionRunsTheSliceCommand`(`:57`) |
| (c3) | 同じ項の「打ったコマンドは RESULT に書く」のあとに「(指示がなければ `./scripts/run-verify.sh`)」だけを足す(ほかの文は崩さない) | 1 | `TestRenderRolePrompt_Implementer_MissionRunsTheSliceCommand`(`:61` だけ) |
| (d) | `leader.md` の implementer への TASK の例で、`VERIFY:` の行を本文からヘッダ(`TASK_ID: t-2` の次の行)に移す | 1 | `TestRenderRolePrompt_Leader_ImplementerTaskExampleCarriesTheVerifyLine`(`:699` がヘッダの `VERIFY:` を、`:703` が本文の行がないことを報告) |

- (c) はリードの指定の字面を戻した c1 のほかに 2 つ試した。c1 は字面を `VERIFY:` の行の文の途中に入れたので、肯定の検査(`:57`、「…`VERIFY:` の行のコマンドを実行する」)も崩れた。否定の検査(`:61`、「指示がなければ `./scripts/run-verify.sh`」がないこと)だけで落ちるかを見るため、肯定の文をどれも崩さない c3 を足し、`:61` だけで落ちることを確かめた。c2 は、前の字面を使わずに既定を `run-verify.sh` に戻しても、肯定の検査が落とすことを見る
- (a) は計画の Edge cases の「leader が指示を書き忘れたとき、reviewer の既定が両方のままであること」(D6)を固める検査が働くことも示す
- 名指しの 4 つ(a、b、c、d)は、どれも少なくとも 1 本のテストが落とした。生き残った mutant はない

## Coverage

- Statement: 計っていない。テストでない Go のコードに差分がないので、この差分で変わる statement の coverage はない
- Branch: 計っていない(Go の標準の道具にない)
- Function: 同上
- Notes: 雛形の文は coverage に出ない。代わりに上の mutant で、名指しの文を変えるとテストが落ちることを確かめた。計画の進捗の記録では、implementer がスライスごとに 6、14、13、11、6 の mutant を試している。この報告ではそれらを打ち直していない

## Failure analysis

| Test | Error | Root cause | Proposed fix |
| --- | --- | --- | --- |
| なし | - | - | - |

## Regression checks

| Previously broken behavior | Status | Evidence |
| --- | --- | --- |
| gate の TASK の節を変えないこと(計画の Test plan) | 変わらず通る。`TestRenderRolePrompt_Leader_FeatureOrgAsksTheReviewerForTheGate`、`TestRenderRolePrompt_Leader_DoesNotRunTheGateItself`、`TestRenderRolePrompt_Reviewer_GateTaskRunsTheTaskCommand` が通過 | 2(evidence の 2 節) |
| reviewer の RESULT の例と `/org` skill の突き合わせ(S2 で例のヘッダを変えない理由) | `TestOrgSkill_ResultExampleFollowsTheReviewerTemplate` が通過 | 2 |
| 機能の org の start が leader の雛形を読むこと | `TestStartFeature*` の 27 本が通過 | 3 |
| `ralph org gate`・`regate`・spawn の既存のテスト(計画の Regression tests) | `internal/org` の全体が ok(455.9 秒、キャッシュなし) | 1b |
| `/org` skill の 4 つの写しと scaffold の同期 | `tests/test-check-skill-sync.sh`、`tests/test-check-sync.sh` が通過。`internal/cli`、`cmd/ralph`、`internal/scaffold` も ok | 1a、1b |

## Test gaps

1. 実機での確かめはない。leader が `VERIFY:` と `CHECKS:` の行を書くこと、implementer がその行のコマンドを打って 5 項目の確かめを RESULT に書くこと、reviewer が `CHECKS:` の行のコマンドだけを打つことは、本物の herdr、agmsg、codex、claude が要る。計画の Test plan の Integration tests は「なし」
2. 雛形のテストは文があるかと順を見るだけで、モデルがその文に従うかは見ない
3. 新しい binary と古い binary の座席が混ざる場合(D6 の「古い leader と新しい reviewer」など)は、雛形の文として確かめただけで、実際に版を混ぜた org は立てていない
4. mutant はリードが名指しした 4 つ(c は 3 つの形)だけを試した。スライスごとの mutant は implementer の記録にあり、打ち直していない
5. statement の coverage は計っていない(テストでない Go に差分がないため)

## Verdict

- Pass: pass。HEAD 065e69b4 で `./scripts/run-test.sh` が rc 0(1,572 秒)。shell の 45 本と、テストのある Go の 9 パッケージがすべて通り、`(cached)` のパッケージはなかった。名指しの実行は、雛形の組がトップレベル 66 本、`TestStartFeature` が 27 本で、すべて通った。雛形の mutant は名指しの 4 つ(c は 3 つの形)がすべて落ち、対照は通った
- Fail: なし
- Blocked: なし
