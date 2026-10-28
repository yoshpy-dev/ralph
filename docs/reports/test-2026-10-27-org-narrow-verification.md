# Test report: org-narrow-verification

- Date: 2026-10-28(JST 08:02〜08:40)
- Plan: `docs/plans/active/2026-10-27-org-narrow-verification.md`(Test plan、AC3、AC6)
- Tester: tester サブエージェント(Claude Opus 5.5)。cycle 2 の回し直し(cap 3)。cross-review 2 回目の ACTION_REQUIRED 2 件(triage `docs/reports/cross-review-triage-org-narrow-verification.md`)を直したあとの実行で、`cycle-count.json` は skill の規則どおり cycle 2 のまま。前回の報告(1637025d)をこの報告で置き換える
- Scope: HEAD 99ed3fde(base 622cdcfb)。behavioral test だけを実行した(静的解析は /verify の担当)。前回の報告のあとのコードと文書の変更は F9 3c64d65d、F10 662d3e47、inline 246c7432、F12 d721ed7d の 4 つで、変えたのは go:embed の雛形 2 つ(`internal/org/prompts/{implementer,leader}.md`)、`/org` skill の 4 つの写し、plan の雛形の 6 つの写し、`internal/org/prompts_verification_test.go`、`tests/test-slice-verification.sh`。テストでない `.go` の変更はない。verify の回し直し(bbb02949)は pass で、F12 はそのあとに入った(skill の文とテストの pin だけ)。環境は macOS(Darwin 25.5.0、arm64)、go1.27.3、`go env GOFLAGS` は空
- Evidence: `docs/evidence/test-2026-10-27-org-narrow-verification.log`(`run-test.sh` の出力すべて、`-count=1` の打ち直し、名指しの実行、コマンドの形の振る舞いの確かめ、mutant ごとの差分と落ちたテストの行、書く前の `git status --porcelain`)。gitignore の対象で、コミットしない
- 追跡しているファイルは、この報告と insight event のほかは変えていない。go:embed の雛形の mutant は scratchpad の写しを `go test -overlay` で差し替えた。ディスクから読む skill と plan の雛形の mutant は、`git archive HEAD` の写しで先に試し、`run-test.sh` が終わってから worktree でも `cp` で退避、変更、実行、戻す、`cmp` の順に行った。`run-verify.sh` が書く `docs/evidence/verify-2026-10-27-230307.log` も gitignore の対象

## Test execution

| # | Suite / Command | Tests | Passed | Failed | Skipped | rc と時間 |
| --- | --- | --- | --- | --- | --- | --- |
| 1 | `RALPH_VERIFY_SCOPE=full ./scripts/run-test.sh`(pack は golang だけが選ばれた) | 下の 1a と 1b | 1 本を除いてすべて | 1(1b の `internal/org`) | 1 件(下) | rc 1、1,371 秒(08:03:07〜08:25:58) |
| 1a | 1 の shell の部分(`verify.local.sh` の test モード、`tests/test-*.sh`) | 45 本 | 45 | 0 | 1 件 | 1 に含む |
| 1b | 1 の Go の部分(`go test ./...`、`-timeout=30m`) | テストのある 9 パッケージ | 8 | 1(`internal/org` の `TestStopDirectorProcess_AProcessThatIgnoresSIGTERMIsNotEndedInTime`) | - | `internal/org` 522.7 秒、`internal/cli` 444.7 秒、`internal/config` 1.1 秒、ほかの 6 つは `(cached)` |
| 1c | `(cached)` の 6 パッケージを `-count=1` で打ち直し(`cmd/ralph`、`internal/insights`、`internal/org/driver`、`internal/org/protocol`、`internal/scaffold`、`internal/upgrade`) | 6 パッケージ | 6 | 0 | - | どれも ok、1.4〜3.4 秒 |
| 1d | `GOFLAGS="-timeout=30m $(go env GOFLAGS)" go test ./internal/org/ -count=1`(1b の失敗のあとの打ち直し) | 1 パッケージ | 1 | 0 | - | rc 0、620.7 秒(08:27:41〜08:38:03) |
| 2a | `GOFLAGS="-timeout=30m $(go env GOFLAGS)" go test ./internal/org/ -run 'TestRenderRolePrompt\|TestRolePrompts\|TestRetiredRole\|OrgSkill\|ReportNames\|SendDefaults\|DirectorSkill' -count=1 -v` | トップレベル 74 | 74 | 0 | 0 | ok、7.2 秒 |
| 2b | `sh tests/test-slice-verification.sh` | 162 | 162 | 0 | 0 | rc 0 |
| 3 | 新しいコマンドの形の振る舞い(scratchpad の使い捨ての module。下の「コマンドの形の振る舞い」) | 11 の実行 | 期待どおり 11 | - | - | 終わったあと module を消した |
| 4 | 雛形と文書の mutant(対照と mutant 4 つ。下の「雛形と文書の mutant」) | 実行ごとにトップレベル 74、または shell の 162 | 対照はすべて通過 | mutant はどれも 1 本以上 | 0 | 1 回 5〜16 秒 |

- 1b の `(cached)` のうち、雛形を埋め込む `cmd/ralph` と、`internal/org` を import する `internal/insights`(`go list -deps`)は、同じ入力で前に走った結果を使っていた。1c で `-count=1` で打ち直し、どれも通った。`internal/org` と `internal/cli` は 1b でテストを走らせた
- 1a の 45 本は `tests/test-*.sh` の数と同じで、どの本の要約にも 0 でない失敗の数はない。Skipped は前からある `tests/test-secret-scan-branch.sh` の 1 件(git 2.49.0 なので「2.41 より古い本物の git」の事例を飛ばし、stub の古い git がその分岐を見る)
- 2a の 74 本の内訳は、`TestRenderRolePrompt*` 54、`TestDirectorSkill*` 7、`TestOrgSkill*` 6、`TestRetiredRoles` 3、`TestRetiredRoleConfigKeys`・`TestRolePrompts`・`TestReportNames*`・`TestSendDefaults*` が 1 ずつ。前回の 72 本から増えた 2 本は、F9 で足した `TestRenderRolePrompt_GoSliceCheckKeepsThePackTimeoutAndEntersANestedModule` と `TestOrgSkill_ImplementerBulletNamesTheGoSliceCheck`(F12 で手順 3 の照合を足した)
- 2b の 162 本には、F10 で足した plan の雛形の写しごとの pin(timeout と入れ子の module の 2 本ずつ、6 つの写しで 12 本)が入る
- 1d の 620.7 秒は、`go test` の既定の timeout の 10 分を超えている。cross-review の 1 件目が指摘したとおり、`-timeout=30m` のない狭めた `go test ./internal/org/` なら、この負荷では打ち切られたはず。打ち切りそのものは試していない
- 1 分の load average は、始まり(08:02)が 13.1、`run-test.sh` の終わり(08:25)が 12.5、全部の終わり(08:38)が 8.0。08:06 には、ほかのセッションのビルドと m1・m2 の実行が重なって 53.9 まで上がった。m1・m2、2a、2b、archive の写しでの m3・m4 は 1 の shell の部分のあいだ(08:05〜08:08)に、worktree での m3・m4 は 1 が終わったあと(08:26〜08:27)に走らせた

## コマンドの形の振る舞い

cross-review の 2 件(狭めた Go の検査に timeout がない、入れ子の module の root に入らない)の直しが、コマンドとして意図どおりに動くかを、scratchpad の使い捨ての module で確かめた。root の module(`example.com/root`、`pkg/`)と、root の下に自分の `go.mod` を持つ `service/`(`example.com/service`、`service/pkg/`)を作り、`pkg` のテストは `t.Deadline()` の残り時間を分に丸めて出す。root に `go.work` は置かない(b5 を除く)。終わったあと module と scratch の `GOENV` のファイルを消し、ユーザーの `go env` のファイルに `GOFLAGS` がないことを確かめた。

| # | 条件 | コマンド | 結果 |
| --- | --- | --- | --- |
| a0 | 対照(timeout を付けない元の形) | `test -z "$(gofmt -l pkg)" && go vet ./pkg/ && go test ./pkg/ -v -count=1` | rc 0、残り 10m0s |
| a1 | 新しい形、`GOFLAGS` は空 | `test -z "$(gofmt -l pkg)" && go vet ./pkg/ && GOFLAGS="-timeout=30m $(go env GOFLAGS)" go test ./pkg/ -v` | rc 0、残り 30m0s |
| a2 | 同じ、環境に `GOFLAGS=-timeout=45m` | 同じ | rc 0、残り 45m0s |
| a3 | 同じ、scratch の `GOENV` に `go env -w GOFLAGS=-timeout=50m`(環境の `GOFLAGS` はなし) | 同じ | rc 0、残り 50m0s |
| b1 | 入れ子の module を root から | `test -z "$(gofmt -l service/pkg)" && go vet ./service/pkg/ && GOFLAGS=… go test ./service/pkg/ -v` | rc 1、`main module (example.com/root) does not contain package example.com/root/service/pkg`(`go vet` で止まる) |
| b1' | 同じ、`go test` だけ | `GOFLAGS=… go test ./service/pkg/ -v` | rc 1、`FAIL ./service/pkg [setup failed]` |
| b2 | 入れ子の module を包んで | `(cd service && test -z "$(gofmt -l pkg)" && go vet ./pkg/ && GOFLAGS="-timeout=30m $(go env GOFLAGS)" go test ./pkg/ -v)` | rc 0、残り 30m0s |
| b3 | b2 に、整形していないファイルを `service/pkg` に足して | b2 と同じ | rc 1(`gofmt -l` の `test -z` で止まる) |
| b4 | root に `go.mod` がなく `service/go.mod` だけ、root から | b1 と同じ | rc 1、`go.mod file not found in current directory or any parent directory` |
| b4' | 同じ、包んで | b2 と同じ | rc 0、残り 30m0s(`(cached)`) |
| b5 | root に `go.work`(`use . ./service`)を置き、root から | b1 と同じ | rc 0、残り 30m0s(`(cached)`) |

- a1〜a3 で、`-timeout=30m` が既定になり、環境の `GOFLAGS` か `go env -w` で決めた timeout があとに並んで勝つことを確かめた(雛形の文「`GOFLAGS` の先頭に置くので、…そちらがあとに並んで効く」のとおり)。a の 4 つは timeout が違うので test cache は使われず、どれも実際に走った
- b1、b1'、b4 で、root に `go.work` がなければ root から入れ子の module に入れないこと、b2 と b4' で `(cd <module> && …)` で包めば通ることを確かめた。b5 は雛形の括弧書き「root に `go.work` がなければ」の条件の裏側で、`go.work` があれば root からも通る
- b4' と b5 の `(cached)` は、同じフラグで走った b2 の結果の再生で、残り時間の表示も b2 のもの。フラグが同じなので結論は変わらない

## 雛形と文書の mutant

変える所は、写しの中に 1 回だけ現れる字面で決めた(2 回以上や 0 回なら止める perl の置き換え。evidence にスクリプトを残した)。Go の実行は 2a と同じ `-run` の組で、どの実行でもトップレベル 74 本が走り、build の失敗はない。

| mutant | 変更 | 差し替え方 | rc | 落ちたテスト(場所) |
| --- | --- | --- | --- | --- |
| 対照(Go) | 変えない `implementer.md` の写し | `-overlay` | 0 | なし(74 本が通過) |
| m1 | `implementer.md:39` の既定のコマンドから `GOFLAGS="-timeout=30m $(go env GOFLAGS)" ` を消す(RESULT の例の行と理由の文は残す) | `-overlay` | 1 | `TestRenderRolePrompt_Implementer_MissionRunsTheSliceCommand`(`prompts_verification_test.go:75`)、`TestRenderRolePrompt_Leader_MissionPutsTheSliceCommandOnAVerifyLine`(`:470`、leader の文が implementer の雛形の選ぶものと同じかを見る行)、`TestRenderRolePrompt_GoSliceCheckKeepsThePackTimeoutAndEntersANestedModule/implementer`(`:505`) |
| m2 | `leader.md:40-44` のミッションの 3 から、入れ子の module の文「パッケージがリポジトリの root の module ではなく、…入らない)。」を消す | `-overlay` | 1 | `TestRenderRolePrompt_GoSliceCheckKeepsThePackTimeoutAndEntersANestedModule/leader`(`:509`) |
| m3 | `.claude/skills/org/SKILL.md:1458-1460` の「Leader 運用 2 経路」の手順 3 から、F12 の句「。Go なら「役割」の節の implementer の項の形で、…そのまま使う」を消す | archive の写し、そのあと worktree で `cp` | 1(どちらでも) | `TestOrgSkill_ImplementerBulletNamesTheGoSliceCheck`(`:567`) |
| m4 | `docs/plans/templates/feature-plan.md:39` の入れ子の module の例から `(cd service && ` と閉じの `)` を消す | archive の写し、そのあと worktree で `cp` | 1(どちらでも) | `tests/test-slice-verification.sh:124` の「docs/plans/templates/feature-plan.md runs a nested module's check inside its root」(161/162) |
| 対照(戻したあと) | worktree の m3・m4 を戻し、`cmp` と `git diff --quiet` で元と同じことを確かめてから | なし | 0 | なし(Go 74 本、shell 162 本が通過) |

- archive の写しの対照(Go 74 本、shell 162 本が通過)も、mutant の前に走らせた
- 名指しの 4 つ(m1〜m4)は、どれも 1 本以上のテストが落とした。生き残った mutant はない。m1 は 3 本が落ち、そのうち 2 本は F9 より前からあるテスト(定数 `implementerGoSliceCheck` を F9 で新しい形にしたので、前からのテストも新しい形を求める)
- m3 のテストはルートの写しだけを読む。`.agents/` と `templates/base/` の 3 つの写しの文の同期は、1a の `tests/test-check-skill-sync.sh` と `tests/test-check-sync.sh` が見る(通過)。m3 をその 2 本で落ちるかは試していない
- 戻したあとの `git status --porcelain` は空だった

## Coverage

- Statement: 計っていない。テストでない Go のコードに差分がないので、この差分で変わる statement の coverage はない
- Branch: 計っていない(Go の標準の道具にない)
- Function: 同上
- Notes: 雛形と文書の文は coverage に出ない。代わりに上の mutant で、名指しの文を変えるとテストが落ちることを確かめた。前回までの mutant(a、b1、b2、c1、c2、d と 1 回目のもの)は打ち直していない。その文を固めるテストは 2a で通っている

## Failure analysis

| Test | Error | Root cause | Proposed fix |
| --- | --- | --- | --- |
| `TestStopDirectorProcess_AProcessThatIgnoresSIGTERMIsNotEndedInTime`(`internal/org/director_cmd_test.go:310`) | `stopDirectorProcess = {outcome:ended signalled:true note:process=ended after SIGTERM err:<nil>}, want not ended within 300ms` | 前からある不安定なテストで、このブランチの変更が原因ではない。テストは `sh -c "trap '' TERM; exec sleep 30"` を起こして 100 ms 待つだけなので、load が高いと trap より先に SIGTERM が届き、`sh` が終わる。`docs/tech-debt/README.md:285` の行が同じ原因を書いている。このブランチは `director_cmd*.go` も `director_stop*.go` も変えていない。前回の報告では、単独の 20 回で HEAD 6 回、base 11 回落ちた。1d の `internal/org` の全体の打ち直しは通った | このブランチでは直さない(計画の scope の外)。前回の報告と同じく、tech-debt の行の案(`ps` が `sleep` を実行していると示すまで待ってから SIGTERM を送る)で直す別の変更を勧める |

## Regression checks

| Previously broken behavior | Status | Evidence |
| --- | --- | --- |
| cross-review 2 回目の 1 件目(狭めた `go test` に timeout がなく、既定の 10 分で打ち切られる) | 雛形と skill の新しい形を `TestRenderRolePrompt_GoSliceCheckKeepsThePackTimeoutAndEntersANestedModule` ほかが固め、m1 を落とす。コマンドとして a1〜a3 で残り 30m、45m、50m になる | 2a、3、4 |
| cross-review 2 回目の 2 件目(入れ子の `go.mod` の module を root から打つと落ちる) | 雛形の文を同じテストが、plan の雛形の例を `tests/test-slice-verification.sh` が固め、m2 と m4 を落とす。コマンドとして b1 は落ち、b2 は通る | 2a、2b、3、4 |
| verify の回し直しの LOW 1(現セッションから昇格した leader が読む skill の手順 3 に Go の形がない)の直し F12 | `TestOrgSkill_ImplementerBulletNamesTheGoSliceCheck` が通過し、m3 を落とす | 2a、4 |
| cross-review 1 回目の直し(前からあった変更に触らない、`RESUME:` の行による再開) | predate のファイルと verification のファイルのテストが 2a で通過 | 2a |
| gate の TASK の節を変えないこと(計画の Test plan) | `TestRenderRolePrompt_Leader_FeatureOrgAsksTheReviewerForTheGate`、`TestRenderRolePrompt_Leader_DoesNotRunTheGateItself`、`TestRenderRolePrompt_Reviewer_GateTaskRunsTheTaskCommand` が通過 | 2a |
| `ralph org gate`・`regate`・spawn・機能の org の start の既存のテスト(計画の Regression tests) | 1b では上の不安定な 1 本だけが落ちた。1d の `internal/org` の全体の打ち直し(キャッシュなし)は ok | 1b、1d |
| `/org` skill と plan の雛形の写しと scaffold の同期 | `tests/test-check-skill-sync.sh`、`tests/test-check-sync.sh` が通過。`internal/cli`、`cmd/ralph`、`internal/scaffold` も ok | 1a、1b、1c |

## Test gaps

1. 実機での確かめはない。leader が新しい形の `VERIFY:` の行を書くこと、implementer が入れ子の module のパッケージで `(cd <module> && …)` を選ぶことは、本物の herdr、agmsg、codex、claude が要る。計画の Test plan の Integration tests は「なし」
2. 雛形のテストは文があるかと順を見るだけで、モデルがその文に従うかは見ない
3. コマンドの形は使い捨ての module で確かめたが、この repo の `internal/org` を新しい形で打って 10 分を超えたときに打ち切られないこと(1d の 620.7 秒)は、`go test` を直接打った結果で、implementer の座席が打った結果ではない。timeout のない形で打ち切られることも試していない
4. mutant はリードが名指しした 4 つだけを試した。plan の雛形の理由の文(timeout を先頭に置く理由)は shell の pin が固定しない(self-review 3 回目の R3-2。計画では、この回の sync-docs で tech-debt の R2-5 の行に足す。この報告の時点の `docs/tech-debt/README.md` にはまだない)
5. `TestStopDirectorProcess_AProcessThatIgnoresSIGTERMIsNotEndedInTime` は load が高いと base でも落ちる。この不安定さが残るあいだ、`run-test.sh` の 1 回の実行が rc 1 になることがある
6. statement の coverage は計っていない(テストでない Go に差分がないため)

## Verdict

- Pass: pass。HEAD 99ed3fde で `RALPH_VERIFY_SCOPE=full ./scripts/run-test.sh` は rc 1(1,371 秒)だったが、落ちたのは `internal/org` の `TestStopDirectorProcess_AProcessThatIgnoresSIGTERMIsNotEndedInTime` の 1 本だけだった。このテストは tech-debt に載っている前からの不安定なテストで、`internal/org` の全体をキャッシュなしで打ち直すと ok だった(620.7 秒)。shell の 45 本と、ほかの Go の 8 パッケージは通り、`(cached)` の 6 パッケージも `-count=1` で通った。名指しの実行は、雛形の組がトップレベル 74 本、`tests/test-slice-verification.sh` が 162 本で、すべて通った。新しいコマンドの形は、使い捨ての module で timeout の既定と上書き、入れ子の module の包みのどれも期待どおりに動いた。mutant は名指しの 4 つがすべて落ち、対照は通った
- Fail: なし(`run-test.sh` の rc 1 は上の不安定なテストによるもので、このブランチの変更による失敗はない)
- Blocked: なし
