# Test report: busy-leader

- Date: 2026-10-23(JST 11:33〜12:15、UTC 02:33〜03:15)
- Plan: `docs/plans/active/2026-10-23-busy-leader.md`(試行の所見を直す系列の 7 本目、F7。AC1〜AC7 の挙動と Test plan の Unit・Integration・Regression・Edge cases)
- Tester: tester サブエージェント(Claude Opus 5.5)。パイプラインの 1 回目(cycle 1 / 2)
- Scope: HEAD 1d088f87(base a58f7c64)。verify の報告(6478fe25、静的解析は 429d7546 で実行)のあとのコードの変更は、新しいテストのファイル `internal/cli/org_send_help_test.go` だけ。behavioral test だけを実行した(静的解析は /verify の担当)。環境は macOS(Darwin 25.5.0、arm64、12 CPU)、git 2.49.0、go1.26.0 と go1.25.8(`GOTOOLCHAIN=go1.25.8`)
- Evidence: run-test.sh が書いた `docs/evidence/verify-2026-10-23-023356.log`(`docs/evidence/*.log` は gitignore の対象で、コミットしない)。ほかの実行の出力、mutant の差し替えファイルと結果は session の scratchpad(以下 `<scratch>`)に置き、worktree には書いていない
- 追跡しているファイルは、この報告と insight event のほかは変えていない。mutant は `go test -overlay` で scratchpad のファイルに差し替えて打った。`git status --porcelain --ignored`(`.harness/cache/` を除く)は mutant の前後で同じだった。本物の herdr・agmsg・osascript・端末・tmux・gh・network を打つものは実行していない

## Test execution

| # | Suite / Command | Tests | Passed | Failed | Skipped | rc と時間 |
| --- | --- | --- | --- | --- | --- | --- |
| 1 | `GOFLAGS=-timeout=30m ./scripts/run-test.sh` | shell 44 本、Go 8 パッケージ | 44 本、8 パッケージ | 0 | shell 1 件(下) | rc 0、844 秒(11:33:56〜11:48:00) |
| 2 | `GOTOOLCHAIN=go1.25.8 go test ./internal/org/ ./internal/cli/ -count=1 -timeout 30m` | 2 パッケージ | 2 | 0 | 0 | rc 0、375 秒(11:48:38〜11:54:53)。org 371.0 秒、cli 301.7 秒。load average(1 分)は始まり 3.14、終わり 4.10 |
| 3a | `go test ./internal/org/ -run 'TestRenderRolePrompt_Leader\|TestRenderRolePrompt_AgmsgScripts\|TestOrgSend_IdleDoneWait\|TestDirectorSkill\|TestOrgSkill' -count=1 -v -timeout 30m` | `--- PASS` 66 行(subtest を含む) | 66 | 0 | 0 | rc 0、6 秒 |
| 3b | `go test ./internal/cli/ -run 'TestNewOrgRuntimeAt_AgmsgHome\|TestOrgSend_Help' -count=1 -v -timeout 30m` | 4 行 | 4 | 0 | 0 | rc 0、1 秒 |
| 3c | Monitor のコマンドの 2 本と `TestOrgSpawn_LeaderRole_AgmsgHomeQuotedIntoPromptFile` を `-count=5 -v` | 45 行 | 45 | 0 | 0 | rc 0、17 秒 |
| 3d | Monitor のコマンドの 2 本を `-count=20 -v`。org パッケージ全体の `-coverprofile` の実行(7)と並べて打った | 120 行 | 120 | 0 | 0 | rc 0、80 秒。load average は始まり 4.58、終わり 6.80 |
| 3e | Monitor のコマンドの 2 本を `GOTOOLCHAIN=go1.25.8` で `-count=5 -v` | 30 行 | 30 | 0 | 0 | rc 0、21 秒 |
| 4 | Regression と Edge の 6 本(下の表)を `-count=1 -v` | 8 行 | 8 | 0 | 0 | rc 0 |
| 5 | 3a・3b の集合と spawn の 1 本を canary の PATH と空の HOME で | org 70 行、cli 32 行 | 102 | 0 | 0 | rc 0。canary の呼び出しは 0 回 |
| 6 | mutant 28 本(下の表) | 28 | - | 26 本が赤 | - | 1 本 3〜67 秒。生き残りは S2・V4 |
| 7 | `go test ./internal/org/ -count=1 -timeout 30m -coverprofile=<scratch>/org-cover.out` | 1 パッケージ | 1 | 0 | 0 | rc 0、432 秒。statement 95.0% |

- 1 の run-test.sh は `==> Language scope: changed (changed_languages)`、pack は golang。Go の ok の行は `internal/cli` 294.412s、`internal/config` 2.183s、`internal/insights` 2.168s、`internal/org` 366.234s。`internal/org/driver`、`internal/org/protocol`、`internal/scaffold`、`internal/upgrade` は `(cached)` だった(この差分はこの 4 つも、その依存も変えていない。GOFLAGS は依頼どおり `-timeout=30m` だけで `-count=1` を付けていない)。`github.com/yoshpy-dev/ralph` と `cmd/ralph` はテストのファイルがない。shell の 44 本はどれも最後の行が `OK` で、Skipped は前からある `tests/test-secret-scan-branch.sh` の 1 件(git 2.49.0 なので「2.41 より古い本物の git」の事例を飛ばす)
- 3a・3b・3c・3e の `--- SKIP` は 0 行。zsh の subtest も飛ばされずに走った(`exec.LookPath("zsh")` が見つけた)
- 3a で PASS したテスト: `TestRenderRolePrompt_Leader_AgmsgInboxWaitsByEndingTheTurn`、`TestRenderRolePrompt_Leader_WaitsPointAtTheAgmsgInboxSection`、`TestRenderRolePrompt_AgmsgScriptsIsTheQuotedScriptsDir`(6 件)、`TestRenderRolePrompt_Leader_MonitorCommandRuns`(bash、zsh)、`TestRenderRolePrompt_Leader_MonitorCommandEndsWhenItsParentIsGone`(bash、zsh)、前からある `TestRenderRolePrompt_Leader_*` 14 本(subtest 4 件)、`TestOrgSend_IdleDoneWaitFails_ErrorsBeforeAnyTypingOrEvent`、`TestOrgSend_IdleDoneWaitTimesOut_SaysWhyAndHowToRetry`(2 件)、`TestDirectorSkill_*` 5 本と `TestDirectorSkillWaitBoundProblems`(21 件)と `TestDirectorSkillNumberedStep`、`TestOrgSkill_SendRowSaysItWaitsForIdleBeforeTyping`
- 3b で PASS したテスト: `TestNewOrgRuntimeAt_AgmsgHomeIsTheAgmsgDriversHome`(`RALPH_ORG_AGMSG_HOME`、`[org].agmsg_home`)、`TestOrgSend_HelpSaysItWaitsForTheTurnToEnd`
- 1〜7 のあと、`ps -axo pid,command | grep 'ralph-org-a leader' | grep -v grep` は何も出さなかった(rc 1)。mutant L1・L4 でループが 30 秒の期限まで残ったときも、テストがプロセスグループごと止めたので、そのあとの `ps` にも残っていない

### Monitor のコマンドのテストの安定

| テスト | 3c(go1.26、5 回) | 3d(go1.26、20 回、パッケージ全体と並走) | 3e(go1.25.8、5 回) |
| --- | --- | --- | --- |
| `MonitorCommandRuns/bash` | 5/5、0.78〜0.82 秒 | 20/20、0.63〜1.47 秒(平均 0.96) | 5/5、0.64〜1.38 秒 |
| `MonitorCommandRuns/zsh` | 5/5、0.73〜0.80 秒 | 20/20、0.63〜1.58 秒(平均 0.95) | 5/5、0.73〜1.39 秒 |
| `MonitorCommandEndsWhenItsParentIsGone/bash` | 5/5、0.80〜0.87 秒 | 20/20、0.71〜1.59 秒(平均 1.04) | 5/5、0.80〜1.08 秒 |
| `MonitorCommandEndsWhenItsParentIsGone/zsh` | 5/5、0.81〜0.94 秒 | 20/20、0.72〜1.48 秒(平均 0.96) | 5/5、0.84〜1.29 秒 |

どれも 1 回も落ちていない。最も遅い 1.59 秒は、テスト自身の期限 30 秒の 5.3% にあたる。親がいなくなったテストは、親の sh が目印のファイルを待つ形(S7 の 36725559)なので、負荷で sh の終わりが遅れても、ループが `$PPID` を読む前に sh が終わることはない。

## AC とテストの対応

| AC | テスト | 確かめたこと |
| --- | --- | --- |
| AC1 | `TestRenderRolePrompt_Leader_AgmsgInboxWaitsByEndingTheTurn`(`internal/org/prompts_leader_inbox_test.go:74`)、`TestRenderRolePrompt_Leader_WaitsPointAtTheAgmsgInboxSection`(`:140`)、`TestRenderRolePrompt_Leader_MonitorCommandRuns`(`:258`、2 件)、`TestRenderRolePrompt_Leader_MonitorCommandEndsWhenItsParentIsGone`(`:322`、2 件)、期待を新しい文に合わせた `TestRenderRolePrompt_Leader_FeatureOrgPlanApproval`・`_FeatureOrgGate`・`_NamesWhichInbox`(`internal/org/prompts_test.go`) | Monitor のコマンドを文字列全体で比べ、節の前置きと各項目の文、古い文(「能動的に確認」「適当な間隔」ほか)がないこと、ほかの待ちが節を指すこと、`ralph org wait` が節と使命の 4 にしか出ないことを見る。描いたコマンドを bash と zsh で走らせ、空行と空白だけの行を落とし、stderr も出し、10 秒の sleep を 1 回、期限で終わって終わりの 1 行を最後に出すこと、親の sh が終わると期限の前に終わることを見る。mutant L1〜L5 と P4 が赤 |
| AC2 | `TestRenderRolePrompt_AgmsgScriptsIsTheQuotedScriptsDir`(`prompts_leader_inbox_test.go:197`、6 件)、`TestOrgSpawn_LeaderRole_AgmsgHomeQuotedIntoPromptFile`(`internal/org/spawn_test.go:3315`、2 件)、`TestNewOrgRuntimeAt_AgmsgHomeIsTheAgmsgDriversHome`(`internal/cli/org_runtime_env_test.go:66`、2 件) | 単一引用符で囲む(空白と `'` を含む置き場所)、末尾の `/`、相対の置き場所を絶対にする、空のとき `RALPH_ORG_AGMSG_HOME`、それもなければ `$HOME/.agents/skills/agmsg`。描いた雛形の inbox.sh と history.sh のコマンドすべてに入り、`{{` が残らない。spawn が書く role prompt のファイルに入る。CLI の `Org.AgmsgHome` と agmsg の driver の Home が同じ値(env が優先、なければ ralph.toml)。mutant P1〜P5、S1、C1、C2 が赤 |
| AC3 | `TestOrgSend_IdleDoneWaitTimesOut_SaysWhyAndHowToRetry`(`internal/org/verbs_test.go:1249`、2 件)、`TestOrgSend_IdleDoneWaitFails_ErrorsBeforeAnyTypingOrEvent`(`:1181`)、`TestOrgSend_HelpSaysItWaitsForTheTurnToEnd`(`internal/cli/org_send_help_test.go:15`) | ctx の期限で切れた待ちは、driver が ctx の期限を包んだ形と herdr 自身の `--timeout` の形の両方で、書き出しと原因のあとに文が付く(文字列全体で比べる)。前者は `errors.Is(err, context.DeadlineExceeded)` のまま。Progress は `SendProgressNothingSent`、PaneID は空、AgentWait のほかの呼び出しはなく、event は増えない。期限の前に失敗した待ちの文は前と同じ。help の段落の 4 文。mutant V1・V2・V3・V5・K8・C3 が赤、V4 が生き残り(穴 1) |
| AC4 | `TestDirectorSkill_PinsTheAnswerOrderSendAndPaneCheck`(`internal/org/director_skill_sync_test.go:750`)、`TestOrgSkill_SendRowSaysItWaitsForIdleBeforeTyping`(`internal/org/org_skill_send_sync_test.go:102`) | 4 写しそれぞれの「答える順」と「計画の承認」の 3、`/org` の `send` の行と「受信箱」節の答え方の項目を、断片の順で見る。skill の名指す文が `sendWaitDeadlineHint` の文に含まれること。mutant K1〜K8 が赤(K1〜K7 は 4 写しのうち 1 つだけを変えた) |
| AC5 | なし(実機の記録) | 依頼の規則で本物の herdr・agmsg・claude を打たないので、/test では打ち直していない。記録は `docs/evidence/busy-leader-2026-10-23.md` |
| AC6 | なし(記録) | /verify の範囲 |
| AC7 | 1、2 | 下の「AC7」 |

### AC7

テストの側は HEAD で満たした。`GOFLAGS=-timeout=30m ./scripts/run-test.sh`(run-verify.sh の test の mode)が rc 0、`GOTOOLCHAIN=go1.25.8` で変えた 2 パッケージが rc 0。

ただし AC7 の文どおりの `GOFLAGS=-timeout=30m ./scripts/run-verify.sh` を、HEAD で 1 本のコマンドとしては打っていない。静的解析の側は verify の報告が 429d7546 で `run-static-verify.sh` を rc 0 で回していて、そのあとのコードの変更は `internal/cli/org_send_help_test.go`(1d088f87、32 行)だけである。このファイルは 1・2 でコンパイルされて走り、`go test` が自分で回す vet の一部も通ったが、gofmt、golangci-lint、staticcheck はこのファイルに対してまだ誰も回していない。AC7 の印を付ける前に、HEAD で `./scripts/run-static-verify.sh`(verify の報告では 27 秒ほど)か run-verify.sh 全体を 1 回打つことを勧める。

## 計画の Test plan の項目

| 区分 | 項目 | テスト |
| --- | --- | --- |
| Unit | 節の文(Monitor のコマンドの期限、`kill -0 "$p"`、終わりの 1 行、`timeout_ms`、手番を終える、`history.sh`、`run_in_background`、Codex の「届かない」) | `TestRenderRolePrompt_Leader_AgmsgInboxWaitsByEndingTheTurn` |
| Unit | 置き換え文字が描いた値になる(単一引用符、空のときの既定)、古い文がない | `TestRenderRolePrompt_AgmsgScriptsIsTheQuotedScriptsDir`、同上 |
| Unit | ctx の期限で切れた待ちのエラーの文、期限でない失敗の文が変わらない | `TestOrgSend_IdleDoneWaitTimesOut_SaysWhyAndHowToRetry`、`TestOrgSend_IdleDoneWaitFails_ErrorsBeforeAnyTypingOrEvent` |
| Integration | `Org` に置き場所を渡した leader の spawn が、role prompt のファイルにその `scripts` を書く | `TestOrgSpawn_LeaderRole_AgmsgHomeQuotedIntoPromptFile` |
| Integration | skill の文のテストが 4 写しを読む | `TestDirectorSkill_PinsTheAnswerOrderSendAndPaneCheck`、`TestOrgSkill_SendRowSaysItWaitsForIdleBeforeTyping`。1 写しだけを変えた mutant K1〜K7 が、変えた写しの名前を出して赤になった |
| Regression | `TestRenderRolePrompt_Leader_NamesWhichInbox` が新しい文で通る | 3a・4 で PASS。期待の文を新しい前置きに直している(下の「前からあるテストの期待の変更」) |
| Regression | `TestOrgSend_IdleDoneWaitFails_ErrorsBeforeAnyTypingOrEvent` が通る | 3a で PASS。文字列全体の比較が足された |
| Regression | 「`{{` が残らない」テストが leader で通る | `TestRenderRolePrompt_Leader_AllKnownVarsSubstituted`、`_EmptyTaskAndEnvelope_NoLeftoverPlaceholders` が 4 で PASS |
| Edge | 置き場所に単一引用符や空白を含む | `TestRenderRolePrompt_AgmsgScriptsIsTheQuotedScriptsDir` の 2 件目(`'/opt/a b/it'\''s/scripts'`)、`MonitorCommandRuns` は `agmsg it's here` の下の inbox.sh を bash と zsh で実際に走らせる、spawn の 1 件目(`/opt/agmsg home`) |
| Edge | 置き場所が空 | `TestRenderRolePrompt_AgmsgScriptsIsTheQuotedScriptsDir` の 5・6 件目、spawn の 2 件目 |
| Edge | `--timeout-ms` が小さく待ちの前に予算が尽きる(今の予算の検査の文は変えない) | 前からある `TestOrgSend_BudgetTooSmallForEnterDelay_NothingTyped`、`TestOrgSend_CtxExpiresDuringEnterDelay_AfterBudgetCheckPasses` が 4 で PASS |
| Edge | 待ちが ctx の期限でなく herdr の失敗で終わる | `TestOrgSend_IdleDoneWaitFails_ErrorsBeforeAnyTypingOrEvent`(文に hint が付かない)。mutant V1 が赤 |
| Evidence | `go test -count=1 -timeout 30m ./internal/org/... ./internal/cli/...` を go1.26 と go1.25.8 で | go1.26 は 1(`./internal/org/...` の driver と protocol は cached)と 7(org を `-count=1`)、go1.25.8 は 2(`./internal/org/` と `./internal/cli/` の 2 つ。依頼のコマンド) |

## 前からあるテストの期待の変更

`internal/org/prompts_test.go` の差分で消えた期待は 5 つで、どれも雛形の文の書き換えに合わせたもの。

- `_FeatureOrgPlanApproval` の手順 3: 「上げたら返事を待つ」を「上げたら、「agmsg の受信箱の運用」の節のとおり手番を終えて返事を待つ」に
- `_FeatureOrgGate` の手順 12: 「バックグラウンドで走らせ」「終わるのを待ってから」「前景で打つなら」の 3 つを、Claude Code の `run_in_background` と終わりの知らせ、Codex の前面と timeout の 2 つの文に。Codex の leader が前面で打つときの timeout の文は残っていて、弱めていない
- `_NamesWhichInbox`: 「能動的に確認してください」の文を、新しい前置きの文に

`internal/org/director_skill_sync_test.go` で消えた行は、`directorSkillWaitSubsection` の本体を、見出しを引数に取る `directorSkillSubsection` に移したこと(と doc コメント)によるもので、期待の文は変わっていない。

## Mutant

28 本。P は `internal/org/prompts.go`、S は `internal/org/spawn.go`、V と K8 は `internal/org/verbs.go`、L は `internal/org/prompts/leader.md`(go:embed。`-overlay` で差し替わる)、C は `internal/cli/org.go`、K1〜K7 は skill の写し。Go と雛形の mutant は 1 本ずつ `-overlay` で差し替え、下の集合を打った。skill の写しはテストが実行時に読むので `-overlay` では差し替わらない。そこで、`directorSkillRepoRoot` が環境変数で別の root を返すようにしたテストのファイルを `-overlay` で入れ、8 つの写しを置いた scratchpad の root で 1 写しだけを変えて打った。変えない root(対照)では 2 本とも PASS、SKIP は 0 だった。

| ID | 変更 | 打った集合 | 結果 | 赤にしたテスト(主なもの) |
| --- | --- | --- | --- | --- |
| P1 | `agmsgScriptsWord` が引用しない | `TestRenderRolePrompt\|TestOrgSpawn_LeaderRole` | 赤 | 引用の 6 件、Monitor の 2 件、spawn の 2 件 |
| P2 | 空の置き場所を既定にしない | 同上 | 赤 | 空の 2 件、spawn の 2 件目 |
| P3 | 相対の置き場所を絶対にしない | 同上 | 赤 | 相対の 1 件 |
| P4 | 置き換えの値を作る条件を逆にする(leader で空になる) | 同上 | 赤 | 引用の 6 件、Monitor の 4 件、spawn の 2 件 |
| P5 | `'` を逃がさずに単一引用符で囲む | 同上 | 赤 | `'` を含む 1 件、`MonitorCommandRuns` の 2 件 |
| S1 | `Spawn` が `AgmsgHome` を渡さない | `TestOrgSpawn_LeaderRole\|DryRun` | 赤 | spawn の 1 件目 |
| S2 | `dryRunSpawn` が `AgmsgHome` を渡さない | 同上 | 生き残り | なし(同値、下) |
| V1 | hint をいつも付ける | `TestOrgSend` | 赤 | `IdleDoneWaitFails_...` |
| V2 | hint を付けない | 同上 | 赤 | `IdleDoneWaitTimesOut_...` の 2 件 |
| V3 | 期限を ctx でなく err で見分ける | 同上 | 赤 | herdr 自身の `--timeout` の 1 件 |
| V4 | hint の ms を既定を入れる前の `p.TimeoutMS` にする | 同上 | 生き残り | なし(穴 1) |
| V5 | 原因を `%w` でなく `%v` で包む | 同上 | 赤 | ctx の期限を包む 1 件(`errors.Is`) |
| K8 | hint の文の「did not end within」を「did not finish within」にする | `TestOrgSend\|TestDirectorSkill_PinsTheAnswerOrder` | 赤 | `IdleDoneWaitTimesOut_...` の 2 件、`TestDirectorSkill_PinsTheAnswerOrderSendAndPaneCheck` |
| L1 | Monitor のコマンドから `kill -0 "$p" 2>/dev/null &&` を消す | `MonitorCommandEndsWhenItsParentIsGone\|MonitorCommandRuns` だけ | 赤 | 親がいなくなったテストの bash と zsh(30 秒の期限で止まる)。`MonitorCommandRuns` は通る |
| L2 | `sleep 10` を `sleep 1` にする | `TestRenderRolePrompt` | 赤 | 文字列の比較のテスト、`MonitorCommandRuns` の 2 件(sleep の記録) |
| L3 | grep で空白だけの行を落とさない(`'^$'`) | 同上 | 赤 | 同上(出力の比較) |
| L4 | `p=$PPID` を `p=$$` にする | L1 と同じ | 赤 | 親がいなくなったテストの bash と zsh(30 秒の期限で止まる) |
| L5 | inbox.sh の `2>&1` を消す | `TestRenderRolePrompt` | 赤 | 文字列の比較のテスト、`MonitorCommandRuns` の 2 件(stderr の行) |
| C1 | CLI が `Org.AgmsgHome` を入れない | `TestNewOrgRuntimeAt\|TestOrgSend_Help` | 赤 | 配線の 2 件 |
| C2 | CLI が `ResolveAgmsgHome("")` を入れる(ralph.toml を読まない) | 同上 | 赤 | `[org].agmsg_home` の 1 件 |
| C3 | help から「(a Claude Code leader woken by its Monitor, say)」を消す | 同上 | 赤 | `TestOrgSend_HelpSaysItWaitsForTheTurnToEnd` |
| K1 | templates/base/.agents の director: 答える順の `--timeout-ms 1500000` を 600000 に | skill の 2 本 | 赤 | 答える順のテスト |
| K2 | .agents の director: 「終了コード 0 でも、それだけでは送れたとしない。」を消す | 同上 | 赤 | 同上 |
| K3 | templates/base/.claude の director: 「同じ leader への send は 1 本ずつ…」を消す | 同上 | 赤 | 同上 |
| K4 | .claude の director: 計画の承認の 3 の「確かめてから、」を消す | 同上 | 赤 | 同上 |
| K5 | templates/base/.agents の org: `send` の行の「そのまま送り直せる(…)」を消す | 同上 | 赤 | `TestOrgSkill_SendRowSaysItWaitsForIdleBeforeTyping` |
| K6 | .claude の org: 受信箱の答え方から「(待ちの時間切れも含む)」を消す | 同上 | 赤 | 同上 |
| K7 | .agents の director: Claude Code と Codex の 2 つの項目の順を入れ替える | 同上 | 赤 | 答える順のテスト(順の検査) |

- L1 と L4 は、文字列の比較のテストを外した集合で打ったので、親の確かめを実際に走らせるテストだけで赤になることが分かる
- S2 は同値と判断した。dry run は role prompt のファイルを書かず、描いた文は `needsPromptFile`(改行を含むか、長いか)の判定にしか使わない。leader の雛形は置き場所の値によらず改行を含むので、結果は変わらない
- K5 の「そのまま送り直せる」は同じ行の後ろの方(「pane に何も送る前の失敗…そのまま送り直せる」)にもう 1 回出るので、消しても断片そのものは見つかる。赤になったのは、次の断片「座席が自分で始めた手番」がその後ろにないためである。順の検査が効いている例で、穴ではない

## Coverage

- Statement: `internal/org` 95.0%(7)。前回の記録(`docs/reports/test-2026-10-23-watch-escalation-inbox.md`)は 94.9%
- Branch: 計っていない(Go の標準の道具にない)
- Function: `agmsgScriptsWord` 100%、`RenderRolePrompt` 100%、`sendWaitDeadlineHint` 100%、`Send` 97.0%、`Spawn` 92.4%、`dryRunSpawn` 93.7%
- Notes: この差分が足した行と変えた行の block はすべて 1 回以上通った(`prompts.go:79〜136`、`verbs.go:394〜402`、`spawn.go` の 2 か所の `RenderRolePrompt` の呼び出し)。`Spawn` と `dryRunSpawn` の未到達の block(`spawn.go:1016`、`:2272` の `RenderRolePrompt` の誤りの return、`:2276` の dry run の `--prompt` つきの分岐)は、この差分が変えていない行である。`internal/cli` の coverage は計っていない。変えた `newOrgRuntimeAt` の行は 3b のテストが値を直接見る

## Failure analysis

| Test | Error | Root cause | Proposed fix |
| --- | --- | --- | --- |
| なし(1〜5、7 で落ちたテストはない) | - | - | - |

flake の打ち直しは要らなかった。

## Regression checks

| Previously broken behavior | Status | Evidence |
| --- | --- | --- |
| F7: agmsg を前面で polling する leader に、director の `ralph org send` が待ちで 4 回切れた | 雛形とエラーの文の側は直った(手番を終えて待つ文、待ちが切れたときの理由と打ち直し方)。届くかどうかの挙動は実機の記録(AC5)にあり、/test では打っていない | AC1・AC3 のテスト。mutant L1〜L5、V1〜V3、V5 が赤 |
| 待ちが ctx の期限でなく herdr の失敗で終わったときの文 | 変わらない | `TestOrgSend_IdleDoneWaitFails_ErrorsBeforeAnyTypingOrEvent` が文字列全体で比べる。mutant V1 が赤 |
| `--timeout-ms` の予算の検査 | 変わらない | 前からある予算のテスト 2 本が PASS |
| leader の雛形に置き換え文字が残らない | 変わらない | 前からある 2 本と、新しいテストの `{{` の検査 |
| ほかの役割の雛形が `AgmsgHome` の影響を受けない | 変わらない | 1・2 で `TestRenderRolePrompt_*` 全体が PASS。値は `{{AGMSG_SCRIPTS}}` を含む雛形でだけ作る |

## Test gaps

1. 待ちが切れたときの hint の ms が、既定を入れたあとの値であることを、`--timeout-ms` が 0 以下の send で打っていない。mutant V4(`p.TimeoutMS` を使う)が生き残る。CLI の `--timeout-ms` の既定は 30000 なので、`--timeout-ms 0` か負の値を明示したときだけ届く道で、文が「0 ms」になるだけで動きは変わらない。テストにするには既定の 30 秒を待つか、既定の値の継ぎ目が要る
2. `agmsgScriptsWord` の、置き場所を絶対にできないとき(`filepath.Abs` が失敗する、cwd が消えたときなど)にそのまま残す道は、打っていない。block としては分かれておらず、coverage の数には出ない
3. 本物の Monitor ツール、本物の herdr、本物の agmsg の `inbox.sh` で雛形のコマンドが動くことは、テストでは見ていない。コマンドは stand-in の inbox.sh と date と sleep で走らせている。本物で動くことは実機の記録(AC5)にあり、依頼の規則で /test では打ち直していない
4. Codex の leader に `ralph org send` が届かないことは、雛形の文として固めているだけで、挙動のテストはない(計画の Non-goals と tech-debt の F7 の行)

## Verdict

- Pass: pass。`GOFLAGS=-timeout=30m ./scripts/run-test.sh` が HEAD 1d088f87 で rc 0(844 秒、shell 44 本と Go 8 パッケージ)、`GOTOOLCHAIN=go1.25.8 go test ./internal/org/ ./internal/cli/ -count=1 -timeout 30m` が rc 0(375 秒)。Monitor のコマンドの 2 本は、go1.26 の 5 回と 20 回(負荷のもと)、go1.25.8 の 5 回で一度も落ちていない。mutant は 28 本中 26 本が赤で、生き残りは同値の S2 と穴 1 の V4。AC7 のうち、静的解析を含めた run-verify.sh を HEAD で 1 本としては打っていない(「AC7」の節)
- Fail: なし
- Blocked: なし
