# Test report: leader-plan-depth

- Date: 2026-10-29(全体の実行は JST 09:01〜09:23、この報告の実行は JST 15:04〜15:40)
- Plan: `docs/plans/active/2026-10-29-leader-plan-depth.md`(Test plan、AC1〜AC8、Progress checklist の slice の注記)
- Tester: tester サブエージェント(Claude Opus 5.5)。pipeline の cycle 1。利用上限で途中で止まった前の tester の続きで、前の tester が済ませた `run-test.sh` の結果を引き継ぎ、残りをこの報告で行った
- Scope: HEAD f35db909(base 9837f744、この branch の変更は `git diff origin/main...HEAD`)。behavioral test だけを実行した(静的解析は /verify の担当)。環境は macOS(Darwin 25.5.0、arm64)、go1.27.3、bash 3.2.57、zsh 5.9、jq 1.7.1-apple、git 2.50.1
- Evidence: `docs/evidence/test-2026-10-29-leader-plan-depth.log`(1〜8 の節: 全体の実行が HEAD のものである根拠、`run-test.sh` の出力、名指しの実行、mutant ごとの差分と落ちたテスト、skill の mutant、隔離の確かめ、待ち切りのテストの時間)。gitignore の対象で、コミットしない
- Insight event: `./scripts/insights-append.sh` は今日の UTC の日付で `2026-10-29-leader-plan-depth.jsonl` を新しく作るので、`--events-dir` で scratchpad に書かせた 1 行(`phase` が `test`、`verdict` が `pass`、`cycle` が 1)を、既存の `docs/insights/events/2026-10-28-leader-plan-depth.jsonl` の末尾に手で足した
- 追跡しているファイルは、この報告と insight event のほかは変えていない。mutant は `go test -overlay` で scratchpad の写しに置き換えるか、scratchpad に置いた `git archive f35db909` の写しで走らせた

## Test execution

| # | Suite / Command | Tests | Passed | Failed | Skipped | Duration |
| --- | --- | --- | --- | --- | --- | --- |
| 1 | `RALPH_VERIFY_SCOPE=full ./scripts/run-test.sh`(前の tester の実行) | shell 45 本 + Go 9 パッケージ | shell 45 本、Go 9 パッケージ | 0 | 0 | rc 0、1,312 秒(09:01:43〜09:23:34 JST) |
| 2 | `go test ./internal/org/ -run 'RenderRolePrompt_Leader\|DirectorSkill\|Skill' -count=1 -v` | トップレベル 58(サブテスト 46) | 58(46) | 0 | 0 | rc 0、22 秒(`ok … 20.213s`) |
| 3 | `go test ./internal/org/driver/ -count=1 -cover` | パッケージ全体 | ok | 0 | - | rc 0、3 秒、coverage 95.3% |
| 4 | `go test ./internal/config/ -count=1 -cover` | パッケージ全体 | ok | 0 | - | rc 0、1 秒、coverage 97.0% |
| 5 | `go test ./internal/cli/ -run 'HerdrWithoutEnvFlag\|OrgVerbs' -count=1 -v` | トップレベル 2(サブテスト 2) | 2(2) | 0 | 0 | rc 0、7 秒 |
| 6 | `leader.md` の mutant 17 個と対照(`go test -overlay`、`-run 'RenderRolePrompt\|Skill'`) | 18 実行 | 対照はトップレベル 85(サブテスト 55)が通り、mutant 17 個がすべて落ちた | 生き残り 0 | 0 | 1 回 24〜84 秒 |
| 7 | `/director` skill の mutant 7 個と対照(`git archive` の写し、同じ絞り込み) | 8 実行 | 対照は 85 が通り(SKIP 0)、mutant 7 個がすべて落ちた | 生き残り 0 | 0 | 1 回 23〜42 秒 |
| 8 | `paneenv.go` の警告文の mutant(`./internal/org/driver/` 全体と cli の 1 本) | 2 実行 | driver で 2 本が落ちた | 生き残り 0 | - | 数秒 |
| 9 | 隔離の確かめ(カナリアの codex と claude、scratch の `HOME` と `TMPDIR`)と、その感度を見る PATH 漏れの mutant | 2 実行 | HEAD は 85 が通り、カナリアの呼び出し 0。漏れの mutant は FAIL 16 行、カナリアの呼び出し 26 | - | 0 | 47 秒と 21 秒 |
| 10 | `go test ./internal/org/ -run '…PlanAdvisoryCodexCommandRuns$' -count=8 -v` | 8 回 × サブテスト 12 | 96 | 0 | 0 | rc 0、123 秒 |

- 1 は前の tester の実行で、出力を `scratchpad/full/out.log` と evidence の 2 節に写した。HEAD が f35db909 だったことは、reflog の最後が 09:00:16 の f35db909 の commit で、その後に HEAD の移動がないことで確かめた。実行中に書き換わった追跡ファイルは `.agents/skills/` の 23 個(09:14)だけで、これは `tests/test-sync-skills.sh:239` が本物の root で `scripts/sync-skills.sh` を打つためである。中身は変わらず、`git status` は空のままだった(evidence の 1 節)。前の tester の mutant は overlay(json は scratchpad の写しを指す)と `git archive` の写しを使っており、`internal/org/prompts/leader.md`、`.claude/skills/director/SKILL.md`、新しいテストの更新時刻は commit より前(08:09〜08:42)のままである。この根拠から打ち直しはしなかった
- 1 の shell の部分: `tests/test-*.sh` の 45 本がすべて走り、`PASS` で始まる行は 4,796 行で、0 でない `FAIL:` の数を持つ要約行はない。SKIP の行は `tests/test-secret-scan-branch.sh` の「git が 2.41 より古い」の 1 行だけで、前からあるものである
- 1 の Go: `cmd/ralph` 1.274 秒、`internal/cli` 353.331 秒、`internal/config` 1.403 秒、`internal/insights` 2.284 秒、`internal/org` 463.967 秒、`internal/org/driver` 4.241 秒、`internal/org/protocol` 3.301 秒、`internal/scaffold` 4.101 秒、`internal/upgrade` 4.223 秒。どれも `(cached)` ではない
- 2 の 58 本のうち、この branch で足したのは `prompts_leader_plan_test.go` の 17 本(サブテスト 16: `PlanWordingStaysInTheFeatureOrgSection` 2、`CodexCommandRuns` 12、`ClaudeCommandRuns` 2)と `TestDirectorSkill_PinsTheApprovalForkSteps` で、残りは既存の `TestRenderRolePrompt_Leader_*` 27 本と skill のテストである。前の tester の同じ 4 つの実行(09:03)も通っていたが、この報告の数は 2〜5 の打ち直しのものである
- 1 分の load average は、2〜10 の間 9〜23 だった。ほかのセッションの `go test` と codex の実行が並んで走っていた

## Test plan の項目

| Test plan の項目 | テスト | 結果 |
| --- | --- | --- |
| 手順 1 の前に AGENTS.md・`.claude/rules/`・計画が触るコードを読む | `TestRenderRolePrompt_Leader_FeatureOrgPlanReadsTheRulesAndCodeFirst` | PASS |
| 分かれ目の 3 条件、機能の内側に限る、2〜4 の案と推奨を `## Design decisions` に書く、split は推奨を採って記録する | `…FeatureOrgPlanListsTheForksInsideTheFeature` | PASS |
| per-plan の QUESTION の `FORK 1:` と `(推奨)`、数で打ち切らない、2 往復目の `(決定済み)` | `…FeatureOrgQuestionListsEveryFork`(打ち切りの言い回しを見る正規表現の表 6 行を含む)、`…FeatureOrgQuestionExampleCarriesForks` | PASS |
| 推奨と違う答えは直しの依頼 | `…FeatureOrgNonRecommendedForkIsAChangeRequest` | PASS |
| `ralph config get org.review.plan_advisory` と終了コードの扱い、`false` なら呼ばない | `…FeatureOrgPlanAdvisoryReadsTheSwitchFirst`(`internal/cli/config.go` の help の約束も照合する) | PASS |
| `pick_reviewer`、codex と claude の打ち方、呼ぶ前後の git の比べ、終わったとみなす条件、飛ばして記録する | `…PicksTheOtherCLI`(bash と zsh で 5 行ずつ)、`…PromptIsThePlanSkills`、`…RunsInOneShell`、`…CodexTurnsMCPOff`、`…ClaudeReadsOnly`、`…Outcomes`、`…QuestionExampleCarriesTheAdvisory`、`…CodexCommandRuns`(bash と zsh で 6 つずつ: すべて止まる、1 つ止まらない、`enabled` の欄がない、`plugins` を知らない古い codex、書き換える助言者、止まらない助言者)、`…ClaudeCommandRuns`(bash と zsh) | PASS |
| 機能ごとでない org の節にこれらが現れない | `…PlanWordingStaysInTheFeatureOrgSection`(サブテスト 2)、`…PlanAdvisoryIsASubsectionAfterTheSteps` | PASS |
| director の skill の「計画の承認」 | `TestDirectorSkill_PinsTheApprovalForkSteps`(4 つの写しをすべて読む) | PASS |
| Edge: 分かれ目がない、4 つ以上、MCP の server を止められない、`RALPH_PRIMARY_CLI` がない pane | 分かれ目なしと 4 つ以上は上の文の pin。止められないときは `CodexCommandRuns` の 3 つの case。変数のない pane は `PicksTheOtherCLI` の 2 行 | PASS |
| Manual check(AC8: 実機の codex で MCP を止める) | この報告では打っていない。実機の codex は呼ばない方針で、計画の Progress checklist に implementer の結果(server 20、止めたあとの enabled 0)がある | 対象外 |

## mutant

### `internal/org/prompts/leader.md`

`leader.md` は go:embed なので、scratchpad の写しを perl で置き換えて `-overlay` で差し替えた(置き換える字面が 0 回なら止め、何回目を置き換えるかを指定する)。対照(置き換えなし)は rc 0 だった。build の失敗は 0 件。

| id | 変更 | 落としたテスト |
| --- | --- | --- |
| a1 | 名前の一覧の jq を `select(.enabled)` に戻す | `CodexTurnsMCPOff`、`CodexCommandRuns/{bash,zsh}/servers_with_no_enabled_field` |
| a2 | 後確認の数の jq を `select(.enabled)` に戻す | `CodexTurnsMCPOff`、`CodexCommandRuns/{bash,zsh}/servers_with_no_enabled_field` |
| b1 | 名前の一覧から `--disable plugins` を外す | `CodexTurnsMCPOff`、`CodexCommandRuns` の 10 個(plugin の server の上書きで stand-in が設定を読めず `mcp: not off` になる) |
| b2 | 引数の並び(`set --`)から `--disable plugins` を外す | `CodexTurnsMCPOff`、`CodexCommandRuns` の 12 個すべて |
| b3 | b1 と b2 の両方 | `CodexTurnsMCPOff`、`CodexCommandRuns` の 10 個 |
| c1 | 後確認を `n=0` に置き換える | `CodexTurnsMCPOff`、`CodexCommandRuns` の 8 個(止まらない server が残っても exec を打つ) |
| c2 | `if [ "$n" = 0 ]` を `if :` にする | `CodexTurnsMCPOff`、`CodexCommandRuns` の 6 個 |
| d1 | codex の塊の `cmp` を外し、いつも `git: same` を出す | `RunsInOneShell`、`Outcomes`、`CodexCommandRuns/{bash,zsh}/a_reviewer_that_edits` |
| d2 | claude の塊の `cmp` を外す | `RunsInOneShell`、`Outcomes`(文の pin だけ。下の Test gaps) |
| d3 | codex の塊で前の状態を前の状態と比べる | `RunsInOneShell`、`CodexCommandRuns/{bash,zsh}/a_reviewer_that_edits` |
| d4 | 4 つの git の状態から未追跡のファイルの hash を外す | `RunsInOneShell`(文の pin だけ。下の Test gaps) |
| e1 | 手順 3 の `FORK` の形の区切りを `/` から `\|` にする | `QuestionListsEveryFork`、`TestDirectorSkill_PinsTheApprovalForkSteps` |
| e2 | 例の `(推奨)` を案 b に移す | `QuestionExampleCarriesForks` |
| e3 | `(決定済み)` の形の `→` を `=` にする | `QuestionListsEveryFork`、`TestDirectorSkill_PinsTheApprovalForkSteps` |
| e4 | 例の `FORK 2:` を `FORK 3:` にする | `QuestionExampleCarriesForks` |
| e5 | 例の `FORK 1:` の区切りを `\|` にする | `QuestionExampleCarriesForks` |
| g | `pick_reviewer "${RALPH_PRIMARY_CLI:-<あなたの CLI>}"` を `pick_reviewer "$RALPH_PRIMARY_CLI"` に戻す | `PicksTheOtherCLI` |

- lead が名指しした (a)〜(e) と (g) はすべて落ちた。(a)〜(c) は文の pin(`CodexTurnsMCPOff`)と、stand-in を相手に雛形のコマンドを実際に打つ `CodexCommandRuns` の両方が落とした
- a1 の実行では、`CodexCommandRuns/{bash,zsh}/a_reviewer_that_hangs` も落ちた。出力は正しく(`codex rc=143` と `git: same`)、23.7 秒と 29.9 秒かかって 20 秒の上限を超えただけである。a1 の変更はこの case の経路(`enabled` の欄のある server だけ)に効かない。この case の時間は下の Failure analysis に書いた
- g の pin を落としたのは文の照合で、`PicksTheOtherCLI` が実際に打つのはテストの定数 `leaderAdvisoryPickCommand` である。雛形の文と定数は、その照合でつながっている

### `/director` skill(lead の (f))

`director_skill_sync_test.go` は skill を `runtime.Caller` から求めた repository の root から実行時に読むので、overlay ではなく scratchpad に `git archive f35db909` の写しを 8 つ作り、写しの中で `go test ./internal/org/ -run 'RenderRolePrompt|Skill' -count=1 -v` を打った。対照の写しは 85 本が通り、SKIP は 0 だった(写しの skill を読んでいることは、mutant が落ちることでも確かめた)。

| id | 変更(4 つの写しすべて。f3 だけ 1 つ) | 落としたテスト |
| --- | --- | --- |
| f1 | 「計画の承認」の手順 2(`FORK` の行を聞く手順)を消す。番号は振り直さない | `PinsTheApprovalForkSteps` |
| f2 | 手順 2 を消し、3〜7 を 2〜6 に振り直す | `PinsTheApprovalForkSteps`、`PinsTheAnswerOrderSendAndPaneCheck` |
| f3 | f2 を `templates/base/.agents/skills/director/SKILL.md` だけに入れる | `PinsTheApprovalForkSteps`、`PinsTheAnswerOrderSendAndPaneCheck` |
| f4 | 「承認と答えていても approve を打たず」を「承認と答えていれば approve を打ち」にする | `PinsTheApprovalForkSteps` |
| f5 | 「分かれ目が 3 つまでなら」を「4 つまでなら」にする | `PinsTheApprovalForkSteps` |
| f6 | 手順 1 の、人が会話にいないときは notify で送るだけにする文を消す | `PinsTheApprovalForkSteps` |
| f7 | 手順 1 の `ADVISORY:` の要約を伝える文を消す | `PinsTheApprovalForkSteps` |

### `internal/org/driver/paneenv.go`

`paneEnvConsequence` の警告文から「, and so may a feature org leader's plan advisory」を外すと、`TestHerdr_TabCreate_PaneEnvWarningNamesEachEntrysConsequence` と `TestHerdr_TabCreate_ExtraEnvWithoutEnvFlagWarnsWithItsClause` が落ちた。`internal/cli` の `TestOrgVerbs_HerdrWithoutEnvFlagWarnsOnStderr` は文の前半だけを pin するので通る(設計どおり)。

## 隔離の確かめ

- 本物の codex と claude を呼ばないこと: ログを書いて exit 0 で終わるだけのカナリアの `codex` と `claude` を、受け継ぐ `PATH` の先頭に置いて、`-run 'RenderRolePrompt|Skill'` を走らせた。85 本が通り、カナリアの呼び出しは 0 だった。感度の確かめとして、`fx.run` の `PATH` から stand-in の `fx.bin` を外す overlay を当てると、FAIL が 16 行出て、カナリアに 26 回届いた(`codex --disable plugins mcp list --json` 24 回、`claude -p …` 2 回)。漏れればこのカナリアで分かる
- `~/.codex` に触れないこと: 同じ 2 回を `HOME` を空の scratch にし、`CODEX_HOME` を外して走らせた(Go の cache と module の場所は元の値を渡した)。走ったあとの scratch の `HOME` にあったのは、go のツールの telemetry の数え(`Library/Application Support/go/telemetry/`)だけで、`.codex` も `.gitconfig` もなかった。テストの source に `HOME`、`UserHomeDir`、`CODEX_HOME` の参照はない。本物の `~/.codex` は、ほかのセッションと Codex のアプリが書き続けているので、更新時刻の比べは根拠にしていない
- 一時ファイル: `TMPDIR` を空の scratch にした実行のあと、中身は 0 個だった(`t.TempDir` が片付ける)。worktree の `git status --porcelain --ignored` は、実行の前と後で同じだった
- 残るプロセス: 実行のあと、親が 1 の `sleep` は 0 個だった。`pgrep -f 'sleep 1200'` で見つかる `sleep 1200` と `sleep 30` はあったが、親をたどると別のセッションのもの(別の Claude Code のセッションの watchdog と、別の project の CI の loop)で、このテストのものではない

## Coverage

- Statement: `internal/org/driver` 95.3%、`internal/config` 97.0%(この branch がこの 2 つで変えたのは doc comment と警告文の文字列だけ)。`internal/org` は、変わったのが go:embed の雛形の文とテストで、パッケージ全体の実行に 8 分かかるため測っていない
- Branch: 測っていない
- Function: 測っていない
- Notes: 雛形と skill の文は statement coverage に現れないので、上の mutant(雛形 17、skill 7、警告文 1)で代えた。生き残りは 0 個

## Failure analysis

HEAD の実行で落ちたテストはない。mutant の実行で 1 回、変更と関係のない落ち方があったので記録する。

| Test | Error | Root cause | Proposed fix |
| --- | --- | --- | --- |
| `TestRenderRolePrompt_Leader_FeatureOrgPlanAdvisoryCodexCommandRuns/{bash,zsh}/a_reviewer_that_hangs`(mutant a1 の実行だけ。HEAD ではない) | `output = "codex rc=143\ngit: same\n" after 23.659180333s; want a non-zero codex rc from the watchdog, soon, and git: same`(zsh は 29.867 秒) | 負荷による時間切れ。テストは `fx.run` 全体(git の状態の取得 2 回、stand-in の codex 3 回、jq 2 回、見張り)に 20 秒の上限を置いている。見張りそのものは 1〜2 秒で codex を止めているが、並んで走る 12 個のサブテストとほかのセッションの負荷(load 約 15)で、全体が 20 秒を超えた。この case の実行は、この報告の記録の中で 64 回(codex の経路を変える mutant b1〜b3 と PATH 漏れの実行を除く)あり、落ちたのはこの 2 回だけだった。サブテストの時間は 6.0〜33.5 秒、中央値 13.5 秒で、単独に近い形(2 つのサブテストだけ)でも 8.8〜11.5 秒かかった。HEAD で `-count=8` を打つと 16 回すべて通った(10) | この branch の新しいテストなので、PR の前か直後に直すことを勧める。stand-in の codex は止められなければ 30 秒後に rc 0 で終わり、その場合はテストが `codex rc=0` の確かめで落ちるので、時間の上限がなくても見張りの欠けは見つかる。直し方は、時間の上限を外すか、stand-in の hang を長く(たとえば 300 秒)して上限を `fx.run` の 60 秒の期限に近い値(たとえば 50 秒)に上げるかのどちらか。直さないなら、`docs/tech-debt/` に負荷に弱いテストとして行を足す |

## Regression checks

| Previously broken behavior | Status | Evidence |
| --- | --- | --- |
| 既存の `TestRenderRolePrompt_Leader_*` 27 本(`FeatureOrgPlanApproval`、`FeatureOrgProcedure` を含む) | 通る | 1、2 |
| 既存の `director_skill_sync_test.go` のテスト(`PinsTheAnswerOrderSendAndPaneCheck` など 7 本)と `/org` skill のテスト | 通る | 1、2 |
| `tests/test-check-skill-sync.sh`、`tests/test-sync-skills.sh` | 通る | 1 |
| `--env` のない herdr の警告(`TestOrgVerbs_HerdrWithoutEnvFlagWarnsOnStderr` と driver の警告のテスト) | 通る | 1、3、5、警告文の mutant |
| 機能ごとでない org の節に、計画の読み込み・分かれ目・計画助言の文が漏れない | 通る | `PlanWordingStaysInTheFeatureOrgSection` |

## Test gaps

- `a_reviewer_that_hangs` の 20 秒の上限は負荷に弱い(Failure analysis)。CI の runner は手元より遅いことがあるので、直すか tech-debt に記録するかを決める必要がある
- claude の塊の `git: changed` の経路は、実際に打つテストがない。`ClaudeCommandRuns` には書き換える助言者の case がなく、mutant d2(claude の塊の `cmp` を外す)を落としたのは文の pin(`RunsInOneShell` が両方の塊の最後の 2 行を同じ文字列に固定する)だけである。固定された行は codex の側で実際に打って確かめているので、危険は小さい
- 未追跡のファイルと index の変化は、実際に打つテストで起こしていない。stand-in の助言者は追跡しているファイルに書くだけで、mutant d4(未追跡のファイルの hash を外す)を落としたのは `RunsInOneShell` の行の完全一致だけである
- stand-in の codex は codex-cli 0.163.0 の振る舞い(plugin の server の上書きで設定を読めない、知らない機能名で exit 1)をまねている。本物の codex と claude の flag は、計画の Non-goals のとおりテストでは打っていない。AC8 の実機の確かめは implementer の記録(計画の Progress checklist)にあり、この報告では打ち直していない
- 走らせたのは macOS の bash 3.2.57 と zsh 5.9 だけである。CI(ubuntu-latest)では bash 5 になり、zsh がなければ zsh の行は飛ばされる。runner に zsh が入っているかは未確認
- leader と director を実機で通す確かめはしていない(計画の Non-goals)
- この branch とは別の既存の性質: `tests/test-sync-skills.sh:239` が本物の root で `scripts/sync-skills.sh` を打つので、`run-test.sh` のたびに `.agents/skills/` の更新時刻が変わる(中身は変わらない)

## Verdict

- Pass: 1(`run-test.sh` rc 0、shell 45 本、Go 9 パッケージ。前の tester の実行で、HEAD f35db909 のものと確かめた)、2〜5 の名指しの実行(rc 0)、mutant 25 個(雛形 17、skill 7、警告文 1)がすべて落ちた、隔離の確かめ(カナリアの呼び出し 0、感度の確かめで 26)、10 の `-count=8`
- Fail: 0
- Blocked: 0
- Verdict: pass(`a_reviewer_that_hangs` の時間の上限が負荷に弱いことを Test gaps に残した)
