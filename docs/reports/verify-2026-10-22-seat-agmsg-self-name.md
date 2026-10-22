# Verify report: seat-agmsg-self-name

- Date: 2026-10-22
- Plan: `docs/plans/active/2026-10-22-seat-agmsg-self-name.md`(Status: Approved。`./scripts/plan-visual.sh digest` は `dc40c452f092` で、`- Approved:` 行と一致)
- Verifier: verifier サブエージェント(Claude Code、パイプライン cycle 1 / 2)
- Scope: `git diff 1e47c309...HEAD`(HEAD 22b36fda、21 ファイル、+1297/-77)。S1〜S5 の slice、self-review の LOW の直し 4991744e とその記録 22b36fda、self-review の報告と insight event を含む。判定は HEAD の tree に対して行った。self-review の報告 `docs/reports/self-review-2026-10-22-seat-agmsg-self-name.md` は Merge: マージ可(LOW 4)
- Evidence: `docs/evidence/verify-2026-10-22-seat-agmsg-self-name.log` と、run-static-verify.sh が書いた `docs/evidence/verify-2026-10-21-235625.log`。どちらも `docs/evidence/*.log` の gitignore によりコミットしない生の出力で、ホームのパスは `~` に置き換えた

## Spec compliance

テストの実行は `/test` の担当なので、AC1〜AC5 はコードとテストの本文を読んで確かめた。AC6 は evidence の文書を、session の scratchpad に残る S1 の生の出力(`s1/results.jsonl`、`s1/timing.py`、`s1/e-off.txt`、`s1/e-on.txt`、`s1d/d1.txt`、`s1d/d2.txt`)と突き合わせた。

| Acceptance criterion | Status | Evidence |
| --- | --- | --- |
| AC1: ralph が打つ agmsg の 7 つの呼び出しが `AGMSG_SELF_NAME=off` を含む環境で打たれ、親の環境の `on` より off が勝つ | 満たす。実行は `/test` | `internal/org/driver/agmsg.go:48-55` の `run` は、Runner が `EnvRunner` を実装していれば `RunEnv(ctx, []string{AgmsgSelfNameOff}, "bash", …)` で、していなければ `Run` で打つ。Send・Join・TeamMembers・History・Leave・Whoami・DeliverySet はどれも `a.run` を通り、`Agmsg` に `R.Run` を直に呼ぶ道はない。`ExecRunner.RunEnv`(`driver.go:58`)は `append(os.Environ(), env...)` を `cmd.Env` にする。os/exec は同じ key の最後の値を使うので、親の `on` より off が勝つ。`TestAgmsg_EnvRunner_EveryCallRunsWithSelfNameOff`(`agmsg_test.go:255`)は 7 つのメソッドで env がちょうど `[AGMSG_SELF_NAME=off]` で、`Run` が 0 回であることを見る。`TestExecRunner_RunEnv_AddsToTheParentEnvironment`(`driver_test.go:175`)は `t.Setenv("AGMSG_SELF_NAME", "on")` の下で本物の `sh` が `off|kept` を出すことを見る。`internal/` の非テストの Go で agmsg の script を打つのは `Agmsg.run` だけで(script 名で grep)、`AgmsgAvailable` は `send.sh` を stat するだけ |
| AC2: 本番の組み立ての agmsg の Runner が環境を足せることをテストが固める | 満たす。実行は `/test` | `internal/cli/org.go:357` の Agmsg は、349 行の `driver.ExecRunner{}` を使う。`internal/cli` で `org.Org` を組み立てるのは `newOrgRuntimeAt`(`org.go:350`)、`newOrgEscalateFallbackRuntime`(`org_inbox.go:348`)、`newOrgDirectorHerdrRuntime`(`org_director_headless.go:788`)の 3 つで、Agmsg を持つのは 1 つ目だけ。`TestNewOrgRuntimeAt_AgmsgRunnerAddsSelfNameOff`(`internal/cli/org_runtime_env_test.go:25`)は Runner が `driver.EnvRunner` であることと、親が `on` のとき代わりの `team.sh` が `off` を読むことを固める |
| AC3: `--env` のある herdr で、座席と headless の director の pane の `herdr tab create` に `--env AGMSG_SELF_NAME=off` が付き、確かめは同じ driver の値で 1 回 | 満たす。実行は `/test` | `herdr.go:241-248` の `TabCreate` は、`PaneEnv` があって `paneEnvSupported` が真のとき、`--label` のあとに `--env KEY=VALUE` を足す。`NewHerdr`(61 行)は `*paneEnvProbe` を持ち、値のコピーは同じ確かめを共有する。確かめは mutex(283 行から)の中で 1 回だけ打たれる。本番は `org.go:356` と `org_director_headless.go:788` が `NewHerdr(…, driver.AgmsgSelfNameOff)` を使う。座席の pane は `spawn.go:904`、headless の director の pane は `director_start.go:1123` の `TabCreate` で作られる。テスト: `TestHerdr_TabCreate_PaneEnvAddsEnvWhenHelpListsIt`(`herdr_test.go:317`、値とそのコピーで help は 1 回)、`TestHerdr_TabCreate_PaneEnvConcurrentCallsCheckOnce`(541 行、8 goroutine で 1 回)、`TestOrgRuntimes_HerdrPaneEnvIsSelfNameOff`(cli、2 つの組み立ての呼び出しが help、tab create、tab create の順)、`TestOrgSpawn_HappyPath_EventSequenceReceiptAndWorkspaceReuse`(`org_test.go:360`、どの tab create も `--env AGMSG_SELF_NAME=off` で終わる)、`TestOrgDirectorStart_LaunchesRegistersAndStops`(`org_director_headless_test.go:208` の期待行) |
| AC4: `--env` のない herdr で `--env` なしに pane を作り、spawn と start は成功し、stderr に何が起きうるかを書いた警告が 1 行出る | 満たす(driver の水準)。実行は `/test` | `TestHerdr_TabCreate_PaneEnvWithoutEnvFlagWarnsOnce`(`herdr_test.go:340`)は、help に `--env` がない、help が失敗する、help が `--env` を挙げて 0 以外で終わる、help が時間切れになるの 4 通りで、`--env` のない argv で 2 回作り、help は 1 回、警告はちょうど 1 行であることを見る。`TabCreate` は警告のあとも pane の id を返すので、spawn と director start は次の手順へ進む。警告の文は、座席の中の agmsg が座席の herdr agent を改名しうること、そのあと ralph が届かないかもしれないこと(`agent_not_found`)、`--env` を挙げる herdr を使うことを書く(`herdr.go:321-323`)。`--envelope`・`--environment`・`--env-file`・`--no-env` を `--env` と見なさないことは `TestHerdrHelpEnvFlag`(582 行)が見る。CLI の水準で `--env` のない herdr に spawn を打つテストはなく、`Warn` が nil のとき `os.Stderr` に出ることもテストは見ていない(Coverage gaps) |
| AC5: launch_cmd の起動スクリプトが check-in と exec の前に `AGMSG_SELF_NAME=off` を export する | 満たす。実行は `/test` | `director_cmd.go:323` が、`RALPH_ORG_AGMSG_HOME` の有無によらず `AGMSG_SELF_NAME='off'` と `export AGMSG_SELF_NAME` を check-in の行(324 行)の前に書き、exec はそのあと。`TestDirectorLaunchScript_TextQuotesEveryValue`(`director_cmd_test.go:442`)は全文と、agmsg home がないときも check-in の直前に来ることを固める。`TestDirectorLaunchScript_ChecksInThenExecsTheAgent`(548 行)は端末の環境に `AGMSG_SELF_NAME=on` があるとき check-in と agent の両方が `off` を読むことを、`TestStartDirector_LaunchCmdOpensTheTerminalAndChecksIn`(1186 行)は start を通しても `off` になることを見る |
| AC6: 分離した herdr での計測と確かめ (a)〜(f) を evidence に書く | 満たす | `docs/evidence/agmsg-self-name-2026-10-22.md` に (a)〜(f) と「触れていないもの」がある。(c) の表の 8 つの値(中央値と最小〜最大)は、`s1/results.jsonl` の 8 行(n は 20 か 24、fails はどれも 0)と一致した。本文の差、約 65 ms(346 − 281)、約 120 ms(657 − 533 = 124)、約 1 s(1278 − 314)、約 2 s(2648 − 698)、最大 1.2 s(1219)も合う。L4 の直しで足した「どの列も send.sh 1 本ごとの時間で、20 本か 24 本の中央値」は、`timing.py` が 1 本ごとの時間を集めて全体の中央値を取る実装と合う。(d) と (e) の受信箱の読み取りは `s1d/d1.txt`・`d2.txt`・`s1/e-off.txt`・`e-on.txt` と合う。(e) の `herdr agent get` の結果は scratchpad に生の出力が見当たらず、evidence の文だけが根拠になる。(b) の対照(on)では pane の名前は変わらず、`agent rename` が `agent_not_found` で失敗したと書いてある。名前が変わることは (e) の対照が示している |
| AC7: tech-debt、根拠の表、仕様の注と Status、V4 を合わせる | 満たす | 下の「記録の確かめ」 |
| AC8: `GOFLAGS=-timeout=30m ./scripts/run-verify.sh` が rc 0、go1.25.8 のテスト | 静的な部分だけ確かめた。テストは `/test` | HEAD で `GOFLAGS=-timeout=30m ./scripts/run-static-verify.sh` が rc 0、`GOTOOLCHAIN=go1.25.8` の `go build ./...` と `go vet ./internal/org/... ./internal/cli/...` が rc 0。計画が書く 08d091f5 の run-verify の rc 0 は、scratchpad の `run-verify-4a.log`(Timestamp 2026-10-21T23:16:41Z、mode all、scope full、secret scan の行が `1e47c309..08d091f5` を名指し、最後が `All verifiers passed.`)と合う。そのあとの 4991744e は Go の 3 ファイル(定数の分割と doc)を変えたので、HEAD のテストは `/test` で打ち直す |

### 記録の確かめ(AC7)

- tech-debt の F3 の行: 317 行の `<!-- RESOLVED 2026-10-22 in fix/seat-agmsg-self-name (S2 b0a607d9, S3 f16423da, S4 a34612fd): … -->` と、318 行の打ち消した行がある。コメントの 3 つの SHA は slice の commit と合う。AC6 の (e) が通ったので、RESOLVED にする条件を満たす。コメントは「`--env` のない herdr では、座席が自分で打つ agmsg がまだ改名しうる」と範囲も書く
- tech-debt の ALERT の行(334 行): 計測の数字、時間切れが再現しなかったこと、load の違い、ralph が off を付けたことを足し、行は閉じていない。AC6 の (f) の判断と合う
- 根拠の表: F3 と D-F3 は「いいえ(今は)」になり、「直す PR」は「座席の環境 4a(fix/seat-agmsg-self-name で直した)」。ALERT の行は「はい」のままで、計測の数字を足した。FR-3 は `[ ]` のまま、FR-5 は F7 が残るので `[ ]` のまま、FR-9 は `[x]`
- 印の数: 仕様は 32 項目で `[x]` 22、`[ ]` 10(base 1e47c309 は 21 と 11)で、変わったのは FR-9 だけ。根拠の表は 31 項目で 21 と 10(base は 20 と 11)。1 項目ずつ比べた違いは、表の前文が入れない FR-12 だけで、印の食い違いは 0。`[ ]` の 10 項目はどれも「印(2026-10-21)」の注を持つ。表の数の行(「`[x]` が 21、`[ ]` が 10」と `[ ]` の一覧)も表と合う
- FR-9 を `[x]` にした理由: 計画の Progress(128 行)に、consult-plan-seatenv の判定 (i)、F8・F10 の前例、印の数の変化(22/10 と 21/10)が書いてある。仕様の FR-9 の注は「印(2026-10-22)」に置き換わり、AC3・AC4 のテスト名、evidence の (e)、「ほかに食い違う所見はない」を書く。根拠の表で FR-9 を挙げる所見は D-F3 と folder trust で、どちらも「いいえ」
- 仕様の Status: 「30 本」「10 項目」「試行の所見 F9・F10・G1 の直し: #247」がある。食い違う所見の一覧は F4、F7、F13、「leader への ALERT」になり、F3 と D-F3 が消えた
- PR の一覧: 仕様の Status と根拠の表の「段ごとの PR」に出る番号は、どちらも同じ 30 個(`cmp` で一致)。「NFR 入れ方」の 27 本(30 本から #205・#241・#242 を除く。9 段の 24 本と #244・#246・#247)と「#246 と #247 を除く 25 本」も数が合う。#247 の計画 `docs/plans/archive/2026-10-21-test-stability.md` と 4 つの report は HEAD にある。#247 が merge 済みであることは base 1e47c309 の commit「Merge pull request #247」で確かめた。`gh` は打っていない(ほかの 29 本が `MERGED` であることは、前の PR の verify の報告が `gh pr view` で確かめている)
- V4: 「fix/test-stability」は仕様で 1 個から 0 個、根拠の表で 4 個(F9・F10・G1 の行と FR-7 の行)から 0 個になった。仕様の FR-7 の注は「#247 は」に変わった。tech-debt の RESOLVED の行に残る 10 個は、tech-debt を branch 名で書く書き方(#246 の `fix/org-record-text` の 4 個も同じ)に沿っており、AC7 の範囲の外
- 引用: 足した行が tech-debt の行を名指す引用は、`~~` を外すとどれも 1 行だけに当たる(前の PR と同じ書き方)。足した行が挙げるテストの関数名 36 個は、どれも `func <名前>(` が、文の中で隣に書いたファイルにある

## Static analysis

| Command | Result | Notes |
| --- | --- | --- |
| `GOFLAGS=-timeout=30m ./scripts/run-static-verify.sh`(HEAD 22b36fda) | rc 0、`All verifiers passed.` | changed scope で、選ばれた pack は golang。shellcheck、hook の `sh -n`、guard の awk の構文、settings.json の jq、Codex の 3 つの guard、check-sync(IDENTICAL 171、DRIFTED 0、ROOT_ONLY 0)、check-pipeline-sync、check-skill-sync(14 skill)、check-template-purity、tech-debt の計画の参照、`gofmt: ok`、go vet、golangci-lint `0 issues.`、staticcheck(出力なし)、secret scan(`1e47c309..22b36fda`、clean) |
| `GOTOOLCHAIN=go1.25.8 GOPROXY=off go build ./...` | rc 0 | go.mod は `go 1.25.8`、手元の go は go1.26.0。toolchain はキャッシュにあり、network は使っていない |
| `GOTOOLCHAIN=go1.25.8 GOPROXY=off go vet ./internal/org/... ./internal/cli/...` | rc 0 | vet はテストファイルもコンパイルする |
| `git diff --check 1e47c309...HEAD` | rc 0 | |
| 足した行の走査(ホームのパス、U+FFFD、16 桁以上の 16 進、token の形、pane の id、UUID) | 漏れなし | 当たったのは、self-review の報告が grep の語を説明する文、`secret` の語、テストの fixture の `w1:p2`・`w3:p2`(base の `realTabCreateEnvelope` と同じ形)だけ。evidence の文書は pane の id、session の id、agmsg が付けた名前の値を写していない |
| shell の script | 変更なし | diff に `.sh` はない |

## Documentation drift

| Doc / contract | In sync? | Notes |
| --- | --- | --- |
| `docs/recipes/director-launch-cmd.md` と `templates/base/` の写し(`cmp` で同一) | ずれ(V1、`/sync-docs`) | 131 行の手順 2 は export するものとして `RALPH_DIRECTOR_TOKEN` と `RALPH_ORG_AGMSG_HOME` だけを挙げる。Environment の節(244〜257 行)にも `AGMSG_SELF_NAME` がなく、255 行は「these two variables」と書くが、今は 3 つ |
| `/director` skill の 4 つの写し(`.claude`、`.agents`、`templates/base` の 2 つ。同一) | ずれ(V1、`/sync-docs`) | 222〜224 行の起動スクリプトの説明は、`RALPH_DIRECTOR_TOKEN`(と `RALPH_ORG_AGMSG_HOME`)の export だけを書く |
| herdr の director の pane の環境を書く文(`/director` skill 276 行、recipe 252〜253 行) | ほぼ合う(V2) | 「herdr の pane の環境は herdr server のもの」は `RALPH_ORG_AGMSG_HOME` について今も正しい。ただ、今は pane に `AGMSG_SELF_NAME=off` が足される |
| `--env` のない herdr で出る新しい警告の行 | 書かれていない(V3、`/sync-docs`) | `/org` skill、`/director` skill、org runtime の仕様、README のどれも、この警告を書かない。director start が出す `RALPH_ORG_AGMSG_HOME` の警告は、skill と recipe に書いてある |
| `README.md` | 任意 | 258 行は「`RALPH_ORG_AGMSG_HOME` reaches this director through the launch script」と書く。`AGMSG_SELF_NAME` を足すかは `/sync-docs` が決める。agmsg の環境や座席の pane の作り方を書いた文は、ほかにない |
| `docs/specs/2026-08-01-org-runtime.md` | 合う | pane を作る flag や agmsg の環境を書いた文はない。Dependencies の herdr の行は版を書かない(4a の Non-goals は herdr の最低の版を決めないとした) |
| `.claude/rules/ralph/agent-messaging.md` | 合う | message の型と形の規約だけで、環境を書かない |
| `/org` skill(4 つの写し) | 警告のほかは合う | spawn と start の行は herdr の pane の flag を書かない |
| `CLAUDE_ENV_FILE` の回り道(D-F3)を手順として書く文書 | ない | 記録(evidence、reports、plans)の外で `CLAUDE_ENV_FILE` が出るのは、仕様の FR-9 の注(D-F3 で何が起きたか)と tech-debt の 2 行(F2 の PATH の話と、打ち消した F3 の行)だけで、どれも手順ではない。`AGMSG_SELF_NAME` は README、recipes、rules、skills、templates、prompts、org runtime の仕様のどこにも出ない |
| `AgmsgSelfNameOff` の doc コメント | 合う | 計画 132 行が `/sync-docs` に渡すとした「起動スクリプトも export する」は、4991744e で doc に入った(`agmsg.go:33-39`)。計画のその項目は済んでいる |
| 計画の Progress と Assumptions | 一部遅れている(V5) | 下の Findings |

## Observational checks

- 計画の digest を計算し直して `dc40c452f092` を得た。`- Approved:` 行と一致するので、承認のあとに変わったのは digest の範囲の外(Progress、AC の checkbox)だけである
- `herdr tab create --help` の出力: session の scratchpad の `hr/out.txt` は、この session の別の agent が 08:32 に打った出力で、この verifier は herdr を打っていない。herdr 0.7.5 の help が stdout に出て `--env <KEY=VALUE>` の行があり、stderr は 0 byte、時間は 0.01 s だった。`execRun` は stdout だけを返すので、確かめは本物の 0.7.5 で `--env` を見つけるはずである。おそらく合うが、ralph の binary から本物の herdr に打った確かめはないので未確認
- `gh`、herdr、agmsg、osascript、Orca、tmux、network は使っていない

## Findings

| ID | Severity | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- |
| V1 | LOW | 起動スクリプトの export を書く文書が `AGMSG_SELF_NAME=off` を挙げない。recipe は「these two variables」と数も書いている | `docs/recipes/director-launch-cmd.md:131`、`:244-257`(と `templates/base/` の写し)、`.claude/skills/director/SKILL.md:222-224`(と 3 つの写し) | `/sync-docs` で、手順 2 と Environment の節と skill の説明に `AGMSG_SELF_NAME=off`(agmsg home の有無によらず export する)を足し、「two」を直す。4 つと 2 つの写しを同じに保つ |
| V2 | INFO | 「herdr の director の pane の環境は herdr server のもの」は、今は `--env AGMSG_SELF_NAME=off` が足される点を言わない | `.claude/skills/director/SKILL.md:276`、`docs/recipes/director-launch-cmd.md:252-253` | 文は `RALPH_ORG_AGMSG_HOME` の話として正しいので、直さなくてもよい。直すなら `/sync-docs` で「ralph が足す `AGMSG_SELF_NAME=off` のほかは」と添える |
| V3 | LOW | `--env` のない herdr で出る警告の行(利用者に見える stderr)を、どの文書も書かない | `internal/org/driver/herdr.go:321-323`。`/org`・`/director` skill、org runtime の仕様、README に該当の文がない | `/sync-docs` で、`/org` skill の spawn と start の説明と `/director` skill の herdr の director の項に、警告が出る条件、その herdr では座席の中の agmsg が名前を変えうること、0.7.5 は `--env` を持つことを 1 文で足す |
| V4 | INFO | 新しい警告は `ralph: warning:` で始まり、`os.Stderr` に直に書く(CLI の組み立ては `Warn` を設定しない)。`internal/cli` のほかの警告は `warning: <verb>: …` の形で `cmd.ErrOrStderr()` に書くので、形と出口が違う。AC4 は stderr の 1 行を求めるだけなので、AC は満たす | `herdr.go:311-323`、`org.go:356`、`org_director_headless.go:410` ほか(`"warning: ` は `internal/cli` の非テストの Go に 18 か所) | 直すなら、CLI の組み立てで `Warn` に `cmd.ErrOrStderr()` を渡すか、接頭辞をほかに合わせる。挙動の変更なので、この PR で直すかは orchestrator が決める |
| V5 | INFO | 計画の Progress checklist の「Review artifact created」が `[ ]` のまま(self-review の報告は 995f3b0f でコミット済み)。Assumptions の `internal/cli/org_director_headless.go:787` は、コメントが 1 行増えて今は 788 行 | 計画の 133 行、39 行 | `/sync-docs` か `/pr` でほかの項目と一緒に直す。行番号は計画を書いた時点のもので、直さなくてもよい |

CRITICAL、HIGH、MEDIUM はない。

## Coverage gaps

- HEAD の Go のテスト(go1.26.0 と `GOTOOLCHAIN=go1.25.8`)は `/test` に残す。計画が書く run-verify の rc 0 は 08d091f5 のもので、そのあと 4991744e が Go の 3 ファイルを変えた
- AC4 は driver の水準でしか固まっていない。`--env` を挙げない help を返す herdr の stub で `ralph org spawn` や `ralph org director start` を打ち、終了コード 0 と stderr の 1 行を見る CLI のテストはない。`Warn` が nil のとき `os.Stderr` に書くことも、どのテストも見ていない。足すなら、`internal/cli/org_test.go` の `herdrStub` を `--env` のない help に替えた 1 件が最も小さい(警告は `os.Stderr` に出るので、テストは `os.Stderr` を差し替えて読む必要がある。V4 を直すならそちらで足りる)
- 新しい binary の `ralph org spawn` を本物の herdr に打った実機の確かめはない。S1 は手で作った pane と claude の座席で、codex の座席の環境も実機では見ていない。0.7.5 より古い herdr が `tab create --help` を無視して tab を作るかどうかも未確認(計画の Progress が同じことを書いている)
- AC6 の (e) の `herdr agent get` の生の出力は scratchpad に見当たらず、evidence の文だけが根拠になる
- ALERT の 10 秒の時間切れの原因は分かっていない。負荷をかけた計測はしていない(計画と evidence のとおり)
- insight event: `./scripts/insights-append.sh --slug seat-agmsg-self-name --flow standard --phase verify --verdict pass --cycle auto --source skill` で 1 行を足し(cycle 1)、この報告と同じ commit に入れた。append は UTC の 2026-10-22T00:09Z に走ったので、script はその日付の新しいファイル `docs/insights/events/2026-10-22-seat-agmsg-self-name.jsonl` に書いた。self-review の行は `2026-10-21-seat-agmsg-self-name.jsonl` にあり、この task の event は 2 つのファイルに分かれる。`docs/insights/README.md` の「Events for each task are stored in a single file」と、append の時点の UTC の日付でファイルを選ぶ script の動きが、日付をまたぐ task で食い違う。この PR が作った食い違いではない

## Verdict

- Verdict: pass(CRITICAL、HIGH、MEDIUM はない。LOW 2 件(V1、V3)と INFO 3 件(V2、V4、V5)は、V4 を除いて `/sync-docs` で直せる文書のずれで、V4 は挙動の形の違い)
- Verified: AC1〜AC5 のコードとテストの本文の対応(7 つの agmsg の呼び出し、本番の 3 つの組み立て、座席と director の pane の `TabCreate`、確かめの共有と mutex、4 通りの警告、起動スクリプトの export の位置)、AC6 の evidence と S1 の生の出力の 8 つの数字、AC7 の記録(tech-debt の 2 行、根拠の表の F3・D-F3・ALERT と FR-3・FR-5・FR-9、仕様の印 22 と 10、表の 21 と 10、1 項目ずつの一致、Status、30 本と 27 本と 25 本、#247 の計画と 4 つの report、`fix/test-stability` の置き換え、引用とテスト名 36 個)、静的解析(run-static-verify の rc 0、go1.25.8 の build と vet、`git diff --check`、足した行の走査)、文書のずれ、計画の digest
- Partially verified: AC4(driver の水準。CLI の水準と `os.Stderr` の出口は未確認)、AC8(静的な部分だけ)
- Not verified: Go のテストの実行(`/test`)、本物の herdr での `ralph org spawn` と headless の director、codex の座席、0.7.5 より古い herdr の挙動、ALERT の時間切れの原因
