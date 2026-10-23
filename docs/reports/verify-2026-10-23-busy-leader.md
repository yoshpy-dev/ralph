# Verify report: busy-leader

- Date: 2026-10-23
- Plan: `docs/plans/active/2026-10-23-busy-leader.md`(パイプライン cycle 1。`./scripts/plan-visual.sh digest` は承認の digest と同じ `f046c0eac2ff` を返した)
- Verifier: verifier サブエージェント(`/verify`)
- Scope: `git diff a58f7c64...HEAD`(HEAD 429d7546、27 ファイル、+1907/-207)。S6 の 1183b08e、S7 の 36725559、self-review の report と insight event を含む。テストは実行していない(`/test` の範囲)
- Evidence: `docs/evidence/verify-2026-10-23-021854.log`(`run-static-verify.sh` が自分で書いたログ。git が無視する場所)。依頼どおりコミットするのはこの report と insight event だけなので、`verify-<date>-<slug>.log` は作っていない

## Spec compliance

| Acceptance criterion | Status | Evidence |
| --- | --- | --- |
| AC1 leader の雛形の待ち方 | 満たす(コードと雛形で確認。テストの実行は `/test`) | `## agmsg の受信箱の運用`(`internal/org/prompts/leader.md:406`)の前置きが、`ralph org send` は手番が終わって `idle` か `done` になるのを待つので手番の途中には届かないと書く。Monitor の項(`:415`)は 1 行のコマンド(`:421`。`{{AGMSG_SCRIPTS}}` と `{{TEAM}}`、空行を落とす grep、`sleep 10`、1500 秒の期限、`kill -0 "$p"`、終わりの 1 行)、`timeout_ms` 1800000、`description`、張り直し、見張りを重ねないことを書く。`history.sh … leader 50` の読み直し、`run_in_background`、Codex の項(`:443`)の「区切っても届かない」と「返事だけを待つときは手番を終える」もある。「能動的に確認」「適切な間隔」「適当な間隔」は repo 全体で 0 件(テストの禁止語の一覧を除く)。使命の 4(`:26`)、手順 3・7、手順 12(`:227`)、件を上げたあと、運用規律の 2 項がこの節を指す。`ralph org wait` が出るのは節の中と使命の 4 だけ。テストは `TestRenderRolePrompt_Leader_AgmsgInboxWaitsByEndingTheTurn` と `TestRenderRolePrompt_Leader_WaitsPointAtTheAgmsgInboxSection`(`internal/org/prompts_leader_inbox_test.go:74`、`:140`)、bash と zsh でコマンドを走らせる `TestRenderRolePrompt_Leader_MonitorCommandRuns`(`:258`)と `TestRenderRolePrompt_Leader_MonitorCommandEndsWhenItsParentIsGone`(`:322`)。既存の `TestRenderRolePrompt_Leader_NamesWhichInbox`(`internal/org/prompts_test.go:1346`)と `TestRenderRolePrompt_Leader_FeatureOrgGate`(`:765`)は新しい文に合わせてある |
| AC2 `{{AGMSG_SCRIPTS}}` | 満たす(コードで確認) | `agmsgScriptsWord`(`internal/org/prompts.go:79`)は、空なら `driver.ResolveAgmsgHome("")` を使い、`scripts` を足して絶対パスにし、`shellQuote` で 1 語にする。置き換えは雛形が置き換え文字を含むときだけ値を作る(`:117-121`、S6 の直し)。`Org.AgmsgHome`(`internal/org/spawn.go:189`)を `Spawn` と `dryRunSpawn` が渡す(`:1014`、`:2270`)。CLI は `driver.ResolveAgmsgHome(orgCfg.AgmsgHome)` を 1 回だけ解き、`driver.Agmsg.Home` と `AgmsgHome` の両方に入れる(`internal/cli/org.go:353`、`:366`)。テストは `TestRenderRolePrompt_AgmsgScriptsIsTheQuotedScriptsDir`(`prompts_leader_inbox_test.go:197`、空白と単一引用符、末尾の `/`、相対パス、空のときの env と既定、`{{` が残らない)、`TestOrgSpawn_LeaderRole_AgmsgHomeQuotedIntoPromptFile`(`internal/org/spawn_test.go:3315`)、`TestNewOrgRuntimeAt_AgmsgHomeIsTheAgmsgDriversHome`(`internal/cli/org_runtime_env_test.go:66`、env が `[org].agmsg_home` に勝つ) |
| AC3 `Send` の時間切れのエラー | 満たす(コードで確認) | `Send` は idle/done の待ちが失敗したとき、`ctx.Err()` が `context.DeadlineExceeded` なら `sendWaitDeadlineHint(timeoutMS)` を `%w` の後ろに足す(`internal/org/verbs.go:394-402`)。書き出し `org: send: wait for seat %q idle/done:` は変わらない。hint(`:517`)は、手番が期限の ms のうちに終わらなかったこと、何も打っていないこと、前面で待つ座席には手番が終わるまで届かないこと、あとで打つか `--timeout-ms` を長くして打ち直すことを言う。テストは `TestOrgSend_IdleDoneWaitTimesOut_SaysWhyAndHowToRetry`(`internal/org/verbs_test.go:1249`。driver が ctx の期限を包む形と herdr 自身の `--timeout` の形の 2 通り、Progress、PaneID、呼び出し、event の数)。期限でない失敗の文は `TestOrgSend_IdleDoneWaitFails_ErrorsBeforeAnyTypingOrEvent`(`:1181`)が丸ごと固める |
| AC4 skill の送り方 | 満たす(文書とテストで確認) | `/director` skill の「答える順」(`.claude/skills/director/SKILL.md:688`)の 2 が、送る前の待ち、Claude Code の `--timeout-ms 1500000` と `run_in_background`、Codex の `--timeout-ms 300000`、同じ leader への send は 1 本ずつ、時間切れの文で resolve しないこと、終了コード 0 のあとの `ralph org read` の確かめ(`Press up to edit queued messages`、Enter は 1 回だけ)を書く。「計画の承認」(`:754`)の 3 と、止めない約束の項(`:966`)も合う。`/org` skill の `send` の行(`.claude/skills/org/SKILL.md:187`)、「受信箱」節の答え方(`:337`)、typed protocol 節の leader の受信箱の段落(`:1319`)も Scope どおり。4 写しは `cmp` で director も org もバイト一致。テストは `TestDirectorSkill_PinsTheAnswerOrderSendAndPaneCheck`(`internal/org/director_skill_sync_test.go:750`。skill の時間切れの句が `sendWaitDeadlineHint` に入っていることも見る)と `TestOrgSkill_SendRowSaysItWaitsForIdleBeforeTyping`(`internal/org/org_skill_send_sync_test.go:102`)で、どちらも 4 写しを読む |
| AC5 実機の確かめ | 満たす(記録で確認。実機は打ち直していない) | `docs/evidence/busy-leader-2026-10-23.md` に (a)〜(f) と disband のあとの `ps` がある。分離した herdr server(`env -i`、XDG の 2 変数と `HERDR_SOCKET_PATH`)、本物の claude haiku、PATH の osascript の shim(呼び出しは 1 回)。追加行に `/Users/` の home、U+FFFD、20 桁以上の 16 進、UUID はない。pane・session の id は写していない(記録には pid だけがある)。記録は 3d1946d6 の binary で打ったもので、そのあとの 02e27384(`description` の文)は実機で打ち直していない。仕様の FR-5 の注、根拠の表、tech-debt の注はこのことを書いている |
| AC6 記録 | 満たす(LOW 1 件) | 仕様の印は `[x]` 29、`[ ]` 3(base は 28 と 4)で、動いたのは FR-5 だけ。残る `[ ]` は FR-3、FR-10、ツール非依存の 3 つ。FR-5 の印(2026-10-23)の注は Codex の leader の欠けをツール非依存に結び、ツール非依存にも 2026-10-23 の注がある。Status の節は #252 と `fix/busy-leader` を書き、PR の数は 35。根拠の表の「項目」の表は `[x]` 28、`[ ]` 3 で、仕様の印と食い違わない(FR-12 は表に入らない)。PR の番号は、仕様の Status の一覧と根拠の表の「段ごとの PR」のどちらも重複なしで 35 個。`fix/watch-escalation-inbox` は `docs/specs`・`docs/evidence`・`docs/tech-debt` で 0 件(base は 3・7・7 件)。#252 が merge 済みであることは、base の a58f7c64 が「Merge pull request #252」であることで確かめた(gh は打っていない)。#252 の計画と 4 つの report のファイルはある。tech-debt の 3 行(F7 の行、F7(a) の行、ファイルの大きさの行)はコードに合う(下の文書のずれの表)。根拠の表の 1 か所だけ、動かしたテストのファイル名が古い(Findings の LOW-2) |
| AC7 run-verify と go1.25.8 | 一部だけ確認 | `docs/evidence/verify-2026-10-23-012904.log` は Mode all、scope full で「All verifiers passed」。始まりは 01:29:04Z(10:29:04 JST)で、d4922368 のコミット(10:28:55 JST)の直後にあたる。ログの最終更新は 10:43:51 JST で、887 秒は報告と合う。秘密の走査の行が 81f30f43 を挙げるのは、走っているあいだに self-review の report のコミットが入ったため。そのあとの S6・S7 で `internal/org/prompts.go`(置き換えの値を作る条件)と 3 つのテストのファイルが変わったので、その run-verify は HEAD のコードを見ていない。HEAD では静的解析(下)だけを回した。テストの実行(go1.26 と go1.25.8)は `/test` の範囲 |

## Static analysis

| Command | Result | Notes |
| --- | --- | --- |
| `./scripts/run-static-verify.sh`(scope changed) | rc 0、約 27 秒 | shellcheck、hook の `sh -n`、guard の awk の構文、`jq -e`(settings.json 2 つ)、Codex の hook の 3 検査、`check-sync.sh`(IDENTICAL 171、DRIFTED 0、KNOWN_DIFF 5、TEMPLATE_ONLY 11)、`check-pipeline-sync.sh`、`check-skill-sync.sh`(14 skill)、`check-template-purity.sh`、tech-debt の計画の参照、Go の pack(変わった言語は golang だけ。gofmt ok、golangci-lint は 0 issues、go vet と staticcheck は出力なし)、`secret-scan-branch.sh` は `a58f7c64..429d7546` で clean。`check-skill-sync.sh` と `check-sync.sh` はこの中で走ったので、別には打っていない |
| `GOTOOLCHAIN=go1.25.8 go vet ./internal/org/... ./internal/cli/...` | rc 0 | go.mod は `go 1.25.8`、手元は go1.26.0 |
| `GOTOOLCHAIN=go1.25.8 go build ./...` | rc 0 | 作業ツリーに残ったのは、git が無視する verify のログ 2 つだけ(`git status --porcelain --ignored`) |
| `git diff a58f7c64...HEAD --check` | rc 0 | 空白の誤りなし |

## Documentation drift

| Doc / contract | In sync? | Notes |
| --- | --- | --- |
| `ralph org send --help`(`internal/cli/org.go:1131-1138`) | はい | 送る前の idle/done の待ち、手番の途中には届かないこと、時間切れでは何も打たないこと、submit の確かめが working と blocked しか見ないこと(`confirmSubmitted`、`verbs.go:640`)が、コードと skill に合う。この段落を固めるテストはない(Coverage gaps) |
| `Send` の doc と `sendWaitDeadlineHint`(`internal/org/verbs.go:318-323`、`:513-519`) | はい | doc は「期限で切れたときだけ hint、ほかの失敗には足さない」と書き、コードは `ctx.Err()` で見分ける。既定の 30000 ms(`verbs.go:35`)は、雛形の「既定で 30 秒まで」、`/director` の「既定 30 秒」、`/org` の「既定 30000」、`--timeout-ms` の flag の既定と同じ |
| 雛形 `leader.md` と `/director`・`/org` の skill | はい | 3 つとも、Claude Code の leader は Monitor で手番を終えて待つ、Codex の leader には座席を待つあいだ届かない、director は長い send を裏で 1 本ずつ打ち、pane を読んでから resolve する、の 3 点で同じことを書く。skill の「`--timeout-ms` を長くして打ち直す」の句は、hint の文と同じ |
| tech-debt の F7 の行と RESOLVED の注(`docs/tech-debt/README.md:326-327`) | はい | Codex の leader に限る文で、`Send` が idle/done を待つこと、区切った待ちでも届かないこと、雛形が Codex の leader に書くことがコードと雛形に合う。注のコミット(S1〜S4)はどれもある。挙げたテスト 5 つは、どれも挙げたファイルに 1 つだけある。セルは 5 つ |
| tech-debt の F7(a) の行(`:345`) | はい | `sendInboxItemToHuman(orgID, id, typ, alertID, reason string, banner io.Writer)`(`internal/org/escalate.go:435`)と `inboxHumanTextFor`(`:472`)は文字列 5 つが同じ順。`sendAlertItemToHuman`(`internal/org/watch.go:1730`)は 4 つ。`sendInboxItemToHuman` のテストの外の呼び出しは `Escalate`(`escalate.go:214`)、`NotifyInboxItem`(`:267`)、`SweepInboxDeadlines`(`internal/org/deadline.go:213`)、`sendAlertItemToHuman`(`watch.go:1731`)の 4 か所。`inboxHumanTextFor` の呼び出しは 1 か所。`escalate_test.go` はそれぞれを 1 回ずつ呼ぶ |
| tech-debt のファイルの大きさの行(`:352`) | はい | `wc -l` は HEAD で verbs.go 2277、verbs_test.go 5854、prompts_test.go 2165、spawn.go 2674、spawn_test.go 5904、base で 2255、5762、2163、2667、5859 で、行の数字と一致する。`director_skill_sync_test.go` は 795 行で 800 を超えていない |
| 記録が挙げるテストの名前と置き場所 | ほぼはい(LOW 1 件) | 仕様、根拠の表、tech-debt、skill、雛形の追加行にあるテストの名前 26 個は、どれも `func <名前>(` で 1 件だけ見つかる。名前の前(日本語の記録)か後ろ(tech-debt)に書いたファイル名も、移したテストを含めて置き場所と合う。合わないのは、テストの名前を書かずにファイル名だけを挙げる 1 か所(Findings の LOW-2) |
| README と quality-gates の答える順(計画の範囲の外) | いいえ(LOW) | Findings の LOW-1 |
| 実機の記録の 1 文(`docs/evidence/busy-leader-2026-10-23.md:181`) | INFO | Findings の INFO-1 |
| 計画の範囲 | はい | 変わった 27 ファイルは、計画の Affected areas、Progress に書いた S4(`ralph org send --help`、`/org` の `approve` の行と「機能の計画の承認」の 4・5)と S6(`prompts_leader_inbox_test.go`)、パイプラインの記録(self-review の report と insight event)のどれかに入る |

## Findings

- LOW-1: 計画の範囲の外の 4 か所が、DECISION の send が終了コード 0 で終われば resolve してよいと書いたままになっている。この PR の `/director` skill(「答える順」の 2 と 3)は、終了コード 0 だけでは送れたとせず、`ralph org read` で DECISION が入力欄に残っていないことを確かめてから resolve するよう変えた(計画の D12)。
  - `README.md:254`「Answer an item by sending the leader a `DECISION` with `ralph org send`, and resolve it only once that send exited 0」
  - `README.md:262`「then the `ralph org inbox resolve` to run once that send exited 0」
  - `docs/quality/quality-gates.md:81` と `templates/base/docs/quality/quality-gates.md:80`「to run once that send exited 0」(template の写しは `ralph init` で利用者の repo に入る)

  勧める直し(`/sync-docs` で、2 つの quality-gates は同じ文に): 「exited 0」を「exited 0 and `ralph org read` shows the `DECISION` was submitted, not left in the leader's input box (the `/director` skill's 答える順)」のように直す。README:254 には、send が leader の手番が終わるのを待つので、座席を待つ leader には長い `--timeout-ms` で送ることも 1 句で足すとよい。
- LOW-2: `docs/evidence/org-spec-evidence-2026-10-21.md:117` の「02e27384 のあとの雛形では、実機を打ち直していない(`internal/org/prompts_test.go` のテストだけ)」のファイル名が古い。`description` の文を固めるのは `TestRenderRolePrompt_Leader_AgmsgInboxWaitsByEndingTheTurn` で、S6 の 1183b08e でこのテストは `internal/org/prompts_leader_inbox_test.go:98` に移った。self-review の F8 の掃き出しは、`prompts_test.go` のすぐ後ろに移したテストの名前が続く形を探したので、名前のないこの 1 か所は拾えなかった。勧める直し: 「(`internal/org/prompts_leader_inbox_test.go` の `TestRenderRolePrompt_Leader_AgmsgInboxWaitsByEndingTheTurn` だけ)」に書き換える。
- INFO-1: `docs/evidence/busy-leader-2026-10-23.md:181` の「雛形は `description` に触れていない」は、記録の 17 行目が書く 3d1946d6 の雛形では正しいが、HEAD の雛形(02e27384 のあと)では正しくない。仕様の FR-5 の注、根拠の表の 117 行目、tech-debt の RESOLVED の注はこの差を書いているので、読み違えるのはこの記録だけを読んだときに限られる。直すなら「(このあと 02e27384 で雛形に書いた。実機では打ち直していない)」を 1 句足す。
- INFO-2: 古い順序を書く文がほかに 2 つあるが、どちらも残してよい。仕様 `docs/specs/2026-10-07-org-multi-org-director.md:126` は 8a-1 で決めたことの記録で、同じ仕様の FR-5 の 2026-10-23 の注が新しい順序を書く。`ralph org approve` の出力(`internal/org/approve.go:314`、`internal/cli/org_approve.go:130`)の「once that send succeeded」は、届いたことまで含めて読めるので、skill と食い違わない。

どれも挙動は変えない。LOW-1 は README と quality-gates(template の写しを含む)を変えるので、記録だけの修正の例外には当たらない。`/sync-docs` の範囲で直せる。

## Observational checks

- 実機は打ち直していない。AC5 は記録の内容を読んで確かめた。記録の生の出力は記録を書いた時点で消してあるので、数字(9.7 秒、8 秒、10.23 秒の間隔など)は記録どおりとして扱った
- 新しいテストが本物の herdr、agmsg、osascript に届かないことを読んで確かめた。Monitor のコマンドを走らせる 2 つのテストは、`t.TempDir()` の下の偽の `inbox.sh` と、PATH の先頭に置いた偽の `date` と `sleep` を使う。親がいなくなる形は `/bin/sh` を自分のプロセスグループで動かし、30 秒を過ぎたらグループごと止める

## Coverage gaps

- `ralph org send --help` に足した段落(`internal/cli/org.go:1131-1138`)を固めるテストがない。`internal/cli` には send の help を見るテストがまだない。いちばん小さい足し方は、`newOrgSendCmd` の `Long` が「Before typing, send waits up to --timeout-ms」と「`ralph org read` shows whether the message arrived」を含むことを見るテストを 1 つ足すこと
- AC7 のうち、S6・S7 のあとの HEAD での run-verify と、go1.26・go1.25.8 での `go test` は、この report では見ていない(`/test` の範囲)
- Codex の leader、`guarded` の leader、25 分の期限で終わったあとの張り直しは、計画と記録のとおり確かめていない

## Verdict

- Verdict: pass
- Verified: AC1〜AC6 を、コード、雛形、skill の 4 写し、記録で確かめた。静的解析(`run-static-verify.sh`、go1.25.8 の go vet と build、`git diff --check`)はすべて rc 0。仕様の印(29 / 3)と根拠の表(28 / 3)、PR の一覧(35 本)は数が合い、動いた印は FR-5 だけ
- Partially verified: AC7 は、d4922368 と同じコードでの run-verify が rc 0(887 秒)であることまで。そのあとの S6・S7 は静的解析だけを HEAD で回した
- Not verified: テストの実行(go1.26 と go1.25.8)、実機の打ち直し、gh での PR の状態の読み直し
- Findings: LOW 2 件(README と quality-gates の古い答える順、根拠の表の古いファイル名 1 か所)と INFO 2 件。どれも挙動は変えず、`/sync-docs` で直せる
