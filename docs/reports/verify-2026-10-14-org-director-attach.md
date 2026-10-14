# Verify report: org-director-attach

- Date: 2026-10-15
- Plan: docs/plans/active/2026-10-14-org-director-attach.md(承認済み、digest 27414f27525b。仕様 `docs/specs/2026-10-07-org-multi-org-director.md` の FR-5・FR-6・FR-9、8 段目の 8a-1)
- Verifier: verifier subagent (Claude)。パイプライン 1 回目(`.harness/state/standard-pipeline/cycle-count.json` の cycle は 1)
- Scope: `git diff ef575748...HEAD`(HEAD 676e7f98、13 コミット、51 ファイル、+6630/-693)。仕様適合(AC1〜AC12、FR-5・FR-9 の文、受け入れ条件 1 つ、安全の不変条件)、文書のずれ、help と動きの照合、静的解析。テストは走らせていない(/test の受け持ち)。テストはコードを読み、どの AC をどう固定しているかだけを確かめた
- Evidence: `docs/evidence/verify-2026-10-14-org-director-attach.md`(run-static-verify の出力、Go と同期の検査、HEAD の binary の probe、director の動詞の help)。wrapper の全ログは `docs/evidence/verify-2026-10-14-184901.log`(gitignore の対象)

## Deterministic checks run

| Command | Result | Notes |
| --- | --- | --- |
| `./scripts/run-static-verify.sh`(changed scope) | pass(rc 0) | shellcheck、hook の `sh -n`、settings.json の jq、Codex の 3 つの guard、check-sync、check-pipeline-sync、check-skill-sync、check-template-purity、tech-debt の計画の参照、golang(`gofmt: ok`、golangci-lint `0 issues.`)、secret scan(ef575748..676e7f98 clean)。`scripts/ralph-config.sh` の変更で言語の範囲は full に広がった |
| `gofmt -l .` | pass | 出力なし |
| `go vet ./...`(go1.26.0) | pass | 出力なし |
| `GOTOOLCHAIN=go1.25.8 go build ./...` と `go vet ./...` | pass | go.mod の Go 1.25.8 で build と vet が通る |
| `scripts/check-skill-sync.sh` | pass | 14 skill が lock-step(`/director` を含む) |
| `scripts/check-sync.sh` | pass | DRIFTED 0、IDENTICAL 169 |
| `git diff --check ef575748...HEAD` | pass | 出力なし |
| `cmp` で `/director` と `/org` の skill の 4 面 | 同一 | `.claude/`、`.agents/`、`templates/base/` の 2 面。`ralph-workflow.md` の root と template、`scripts/ralph-config.sh` の 2 面も同一 |
| HEAD の binary の probe(一時の台帳、PATH は git だけ) | 期待どおり | 見張りなしの escalate は `inbox_no_watch` と `delivery:"now"`、`ralph org watch --all` が動く間の escalate は `escalations.jsonl` に何も書かず NOTIFIED `deferred`、`--config` の `enabled = false` は attach を断り escalate は `inbox_director_disabled`。A(d1)→ B の `--replace`(d2)→ A の `detach --id d1` は終了コード 1 で、d2 が残る。`--org-id` は断られる |

## Spec compliance

行番号は HEAD 676e7f98 のもの。

| Acceptance criterion | Status | Evidence |
| --- | --- | --- |
| AC1 `[org.director].enabled` は bool で既定 true。4 面が同じ値で lock-step、`[org.director]` なしは true | Met | `internal/config/config.go:236`(`Enabled: true`)、`templates/base/ralph.toml:125`、`scripts/ralph-config.sh:97`(template と同一)、`internal/config/defaults_sync_test.go:173` が 3 面を比べる。テスト: `TestDefault_OrgDirector`(`config_test.go:796`)、`TestLoad_OrgDirectorMissingSectionBackfillsDefault`(812)、`TestLoad_OrgDirectorRoundTrip`(841、`enabled` のない `[org.director]` も true) |
| AC2 attach / `--replace` / detach `--id` / `--force` / 拒否の各場合、`id` の割り当て、driver、`enabled = false`、`--org-id` | Met | store: `Attach`(`internal/org/director.go:196`、attached のままなら 207-209 で何も書かずに断る、`--replace` は 216-229 で 1 回の write)、`Detach`(252、`id` が違えば 280-282 で `ErrDirectorIDMismatch`)、`nextID`(436、`directorRawIDPattern` 71 で壊れた行の `d<N>` も数える)。CLI: driver の検査(`internal/cli/org_director.go:97-103`)、`enabled = false` の拒否(108-112)、`--org-id` の拒否(258-260)。テスト: `TestOrgDirector_AttachReplaceAndDetachByID`(`org_director_test.go:103`。A→B→A の順を含む)、`DetachNeedsExactlyOneOfIDAndForce`(238)、`DetachForce`(268)、`AttachRefusesABadDriverBeforeReadingAnything`(294)、`RefusesOrgID`(321)、`EnabledFromTheMainWorktreeRalphToml`(368)、`TestDirectorAttach_NeverReusesTheIDOfADamagedLine`(`director_test.go:486`)、`TestDirectorDetach_AReplacedDirectorCannotEndItsSuccessor`(366)、`TestDirectorAttach_ConcurrentAttachesWithoutReplace_OnlyOneSucceeds`(657)。`ralph status` の attached と `id` は `TestStatusCmd_DirectorLinePerState`(`status_test.go:669`)と binary の probe |
| AC3 enabled で見張りが受け付けているとき記録だけ(`escalations.jsonl` も `notified` も書かない、終了コード 0)、ack の期限で `inbox_ack_deadline`、director が attached でも同じ | Met | `escalateDelivery`(`internal/org/escalate.go:310`)が `WatchAcceptsWork` のときだけ `InboxDeliveryDeadline` を返し、`Escalate`(173)は 189-193 で banner だけ書いて戻る。CLI は `wireOrgEscalateWatch`(`internal/cli/org_inbox.go:100`)。テスト: `TestEscalate_DeliveryFollowsTheDirectorSwitchAndTheWatch`(`escalate_test.go:512`、9 つの場合を attached あり・なしの両方で回し、`director.jsonl` を書かないことも見る)、`TestSweepInboxDeadlines_ADeferredItemIsSentAtTheAckDeadline`(`deadline_test.go:555`、偽の時計で 10 分ちょうどまでは送らず 10 分 1 秒で `inbox_ack_deadline`)、CLI の `TestOrgEscalate_WatchAcceptsWork_RecordsTheItemOnly`(`org_inbox_test.go:505`)と本物の probe の `TestOrgEscalate_ProbesTheLedgersWatch`(584) |
| AC4 見張りが受け付けていない(not_running・stale・stopping・`--once`)ときは `inbox_no_watch`、`enabled = false` は `inbox_director_disabled`、通知の失敗の扱いは同じ、`inbox notify` は `inbox_manual_notify` | Met | `escalate.go:310-318` の 2 つの理由、`NotifyInboxItem` の `InboxReasonManualNotify`(247)。テスト: 上の `DeliveryFollowsTheDirectorSwitchAndTheWatch` の 4 状態と mark のない running、`TestEscalate_SendAtOnceFailureIsHandledAlikeForEitherReason`(668)、`TestNotifyInboxItem_SendsADeferredItemNow`(717)、CLI の `TestOrgEscalate_ProbesTheLedgersWatch`(once・stale・no watch)、`TestOrgDirector_AttachWarnsWhileNoWatchAcceptsWork`(`org_director_test.go:540`、stopping を含む) |
| AC5 `enabled` は main worktree の規則で読み、feature worktree の `ralph.toml` は escalate と attach を変えない | Met | `withMainWorktreeDirectorEnabled`(`internal/cli/org.go:201`)と `loadMainWorktreeOrgConfig`(219)。escalate は `org_inbox.go:94`、attach と status は `org_director.go:271`、`ralph status` は `statusDirectorEnabled`(`status.go:237`)。テスト: `TestOrgEscalate_DirectorEnabledFromTheMainWorktreeRalphToml`(`org_inbox_test.go:627`、8 つの場合。`--state-dir` と `RALPH_ORG_STATE_DIR` で共通の台帳を指す場合を含む)、`TestOrgDirector_EnabledFromTheMainWorktreeRalphToml`(`org_director_test.go:368`)、`TestStatusCmd_DirectorEnabledFromTheMainWorktree`(`status_test.go:820`) |
| AC6 `ralph status` の `director:` の行(org があってもなくても)と `--json` の `director`。attached・not attached・disabled・読めないときを言い分ける | Met | `status.go:425`(空の roster)と 634(org のある表)、`statusJSON.Director`(494)、`orgDirectorLine`(`org_director.go:405`)。テスト: `TestStatusCmd_DirectorLinePerState`(669、7 つの状態 × org の有無で、`org watch:` の直後にあることと JSON の state を見る)。binary の probe でも `org watch:` の直後に出た |
| AC7 `/director` skill の 4 面が同じ中身、2 つの同期の検査が通る、`ralph-workflow.md` の 2 面に `/director` | Met | `cmp` で 4 面同一、`check-skill-sync.sh` と `check-sync.sh` は pass、`.claude/rules/ralph/ralph-workflow.md:10` と 28-30 に `/director`(template と同一) |
| AC8 help・leader の雛形・`/org` skill・README・quality-gates の 2 面・仕様の注・AGENTS.md・tech-debt が 8a-1 を説明し、「director を登録できない」が残らない | Met(AGENTS.md の 1 語のずれは下の Documentation drift の V1) | 「Until a director can be registered」「今後入る director」などの残りは、消えたことを確かめるテストの文字列(`org_inbox_test.go:780`、`prompts_test.go:1128`)と、仕様の 5 段目・6 段目の注(8a-1 の注を指す但し書き付きの履歴)だけ。leader の雛形の受信箱の段落、README の 4 段落、quality-gates の 2 面の 2 行、仕様 FR-9 の 8a-1 の注、AGENTS.md の `director.go`、tech-debt の 3 行の追加と既存の行の書き換えを読んだ。help は `TestOrgEscalateAndInboxHelp_DescribeTheDirectorSwitch`、`TestOrgDirectorHelp_DescribesTheRegistration`、`TestStatusCmd_HelpDescribesTheDirectorLine` が文言を固定する |
| AC9 テストは repo の台帳に `director.jsonl` を書かず、本物の osascript と herdr を打たない | Met(コードを読んだ範囲) | `internal/cli/main_test.go:85` が `orgDesktopNotifyOverride` を何もしない stub に、`orgWatchLaunch` を panic に固める。`orgEscalateWatchOverride`(`org.go:429`)は nil のままで、各テストは自分の一時の台帳を probe する。director の CLI テストはどれも `--state-dir <TempDir>` か `newLegacyLedgerRepo`(一時の repo へ chdir)を使う。org 層の escalate と approve の fixture は `DesktopNotify` を stub にするか、notify を打たない(approve は通知しない)。main の checkout と worktree のどちらにも `.harness/state/org/` はない。実行時の確認は /test |
| AC10 approve は記録だけ、件は open か acked のまま、send と resolve の打ち方をこの順に出す、記録済みの件は断る、send の前に止まった件は期限で人に届く | Met | `checkNotApprovedYet`(`internal/org/approve.go:306`、171 でロックの中から呼ぶ)、`ApprovalResolveCommand`(427)、`checkResolvedMeanwhile`(442、受信箱に書かない)、`printApprovedPlans`(`internal/cli/org_approve.go:119`、send の次に 130 で resolve)。テスト: `TestOrgApprove_RecordsAndLeavesTheItemAndThePlan`(`approve_test.go:50`、inbox のファイルが byte で同じ)、`RefusesAnItemAlreadyApproved`(476、エラー全体の等号)、`AnApproveThatStopsBeforeTheSendReachesAHumanAtADeadline`(531、open は ack の期限、acked は resolve の期限、偽の時計)、CLI の `RecordsAndPrintsTheSendThenTheResolve`(`org_approve_test.go:162`)、`ASecondApproveOfTheItemIsRefused`(479)、`TheDecisionAndResolveCommandsRunInAPOSIXShell`(228) |
| AC11 記録だけの件は `delivery: deadline`、NOTIFIED `deferred`、show は「director か期限が受け取る」。すぐ送った件と古い行は `now`。期限の結果は `delivery` で変わらない | Met | 書く側は `InboxStore.Escalate`(`internal/org/inbox.go:214`、2 値以外を断る)、畳み込みは 471(`deadline` 以外を `now` に寄せる)、`InboxDeadlineDue`(`deadline.go:58`)は `Delivery` を読まない。表示は `inboxNotifiedColumn`(`org_inbox.go:402`)、`inboxItemDeferred`(412)、show の 441。テスト: `TestInboxEscalate_WritesAndFoldsTheDelivery`(`inbox_test.go:192`)、`TestInboxDeadlineDue`(`deadline_test.go:116`、164-173 で 3 つの delivery を同じ期待値で回す)、CLI の `TestOrgEscalate_WatchAcceptsWork_RecordsTheItemOnly`(list・show・`--json`)。binary の probe でも `deferred` と show の行が出た |
| AC12 `/director` と `/org` の skill は DECISION を送り終了コード 0 を確かめてから resolve する順を書き、approve の出力の send と resolve をこの順に打つと書く。resolve を send より先に書いた手順がない | Met | `.claude/skills/director/SKILL.md` の「答える順」(124-154、send の 2 のあとに resolve の 3)、「計画の承認」の 3(168-171)と 5(174-176)、「してはいけないこと」の 225。`.claude/skills/org/SKILL.md` の受信箱の節(315-320、321-324)、「機能の計画の承認」の 4(763-771)と 5(772-776)、approve の表の行(183)。resolve だけの手順は、返す DECISION がない RESULT(`director` 117-118、`org` 390-392)で、どちらもそう書いてある。leader の雛形には resolve の手順がない |
| FR-9 起動 (A)「対話セッションを `/director` で昇格させ、`ralph org director attach` で共通の台帳に登録する」 | Met(8a-1 の範囲) | AC2・AC7。登録するのは driver と起こし方(`wake` は `["pull"]`)。push の送り先は 8b、`pane_id` は 8a-2 が書く(計画の Non-goals と仕様の 8a-1 の注のとおり) |
| FR-9「有効でも director が登録されていなければ、受信箱の件は FR-6 の期限で人に届く」 | Met | AC3。登録の有無で動きが変わらないことは `DeliveryFollowsTheDirectorSwitchAndTheWatch` の attached あり・なしで固定 |
| FR-5「director が無効のとき、escalate は受信箱に記録したうえで、人への経路にすぐ送る」 | Met | AC4 の `inbox_director_disabled`。binary の probe の 3 |
| 受け入れ条件「director が登録されていない、when leader が escalate する、then ack の期限が切れた時点で人への経路に届く」 | Met | `escalateDeferred`(`deadline_test.go`、director を attach しない fixture)から `ADeferredItemIsSentAtTheAckDeadline` が escalate から期限の送信までを 1 本で通す |
| 安全の不変条件(全 LLM を止めても期限切れの件が人に届く) | Met(既知の穴は記録済み) | escalate の時点で見張りが仕事を受け付けていなければすぐ送る(AC4)。見張りがあれば ack の期限で送る(AC3)。escalate のあとで見張りが止まると、記録だけの件は見張りが動き出すまで届かない。この穴は計画の Non-goals、README、`/org` skill、tech-debt の 209 行目(「delivery `deadline` ... until a watch starts again」)に書いてある |

## Documentation drift

| ID | Severity | Where | Finding |
| --- | --- | --- | --- |
| V1 | LOW | `AGENTS.md:90`(repo map の `internal/org/` の項) | この PR は同じ行に `director.go` を足したが、「`ralph org escalate` with its interim human notification (`escalate.go`)」が残った。「interim」は、director を登録できないあいだ escalate が全件をすぐ送っていた 5 段目の形を指す。8a-1 では既定で記録だけになり、すぐ送るのは `enabled = false` か見張りがないときの常設の経路になったので、この語は今の動きと合わない。「`ralph org escalate`, which leaves an item to the director and the ack deadline or sends it to the human at once (`escalate.go`)」のような言い方に直す。/sync-docs で直せる |
| V2 | INFO | `docs/tech-debt/README.md:202` | 解決済みの取り消し線の行が、57c551ff で `statusOrgConfig` に置き換えられた `statusDeadmanMinutes` の名前を引く。解決済みの履歴の行なので直さなくてよい |
| V3 | INFO | 計画の Progress checklist | 「Review artifact created」が未チェックのまま(self-review の report は bf615006 にある)。オーケストレーターが更新する欄で、失敗とはしない |
| V4 | INFO | `ralph org director status --help` | 「when a director is attached, the registration's fields」とあるが、`enabled = false` で登録が残っている状態(state は `disabled`)でも欄を出す(`TestOrgDirector_StatusDisabledNamesALeftRegistration`)。登録自体は attached のままなので、誤りとまでは言えない |

ほかに確かめたこと。

- approve が件を resolve すると書いた箇所の残り: `/org` skill の 4 面、README、仕様、quality-gates の 2 面、`org_approve.go`、`approve.go`、`approval.go` を英語と日本語の両方で grep した。当たったのはどれも否定形(「resolves no inbox item」「件を resolve しない」)か、仕様の 7 段目の前半の注(8a-1 で変えたと但し書きのある履歴)だけ
- `inbox_no_director` の残り: 古い記録を読む互換の説明(`escalate.go:48`)、互換のテストの入力、過去の evidence、仕様の 5 段目の注(3 つの理由に分けたと但し書きあり)だけ
- 57c551ff で消えた名前(`resolveApproved`、`ApprovedPlan.Resolved`、`statusDeadmanMinutes`): 現行のコードと文書に残りはない。残りは過去の evidence、計画の記述、V2 の解決済みの行
- tech-debt の Trigger に「stage 8a's director」と書いた 8 行(234〜241)は、どれも 8a-1 の判断を書いている。追加の 3 行を含め、変えた行の列の数は 5 で見出しと合う(列の数が違う 10 行は ef575748 の時点からある)
- `/director` skill が使う動詞と flag を実物で照らした: `wait --inbox --timeout-ms 0` は無期限、時間切れは終了コード 1、`org read --seat`、`org report --out`、`org watch:` の `running (--once)`、機能の worktree の `.claude/worktrees/org-<org_id>`(`internal/org/feature.go:54`、140)
- help と動き: `ralph org director` とその 3 つの動詞、`org escalate`、`org inbox`、`org inbox notify`、`org approve`、`status` の help を HEAD の binary で出し、上の分岐と照らした。`org --config` の help の「escalate, director attach, and director status read [org.director].enabled ... from the main worktree's ralph.toml」は、detach が設定を読まないこと(`org_director.go:174` の `loadConfig` false)と合う。`approve --help` の「the ack and resolve deadlines ... can still bring it to a human」は「can」で言い、1 度送った resolve の期限のあとは効かない例外を tech-debt の行が持つ

## Observational checks

- HEAD の binary の probe(evidence の 3)。PATH を git だけにしたので osascript は `executable file not found` になり、desktop 通知は出ていない。一時の台帳で foreground の `ralph org watch --all` を立て、`--stop` で止めた(`stopped ralph org watch --all (pid ...)`)
- 見張りがないときの attach の警告の理由(`none runs`)、2 つ目の attach の拒否の文言(detach と `--replace` の打ち方が 1 回ずつ)、`--replace` の出力、入れ替えられた `id` の detach の拒否の文言は、テストの期待値と同じ
- 記録だけの件の banner は分の値を出さず、`[org.director].ack_minutes` と書く(self-review L3 の直し)

## Coverage gaps

- テストの実行結果(/test の受け持ち)。AC9 はコードを読んだだけで、テストが repo の台帳に書かないことは実行して確かめていない
- 連結した worktree から `--state-dir` を渡す `ralph status` の `git worktree list --porcelain` の分岐は、binary でもテストでも通していない(self-review の Coverage gaps と同じ)
- 同じ件への approve の 2 process の同時実行(manifest のロックの中の `checkNotApprovedYet` で片方が断られる設計)は、直列のテストだけを読んだ
- ack の期限で人に届くところは、偽の時計の Go のテストに任せた。binary では 10 分待っていない(計画の Evidence to capture のとおり)
- `ralph init` で `/director` skill が scaffold に入ることは、`templates.go` の `//go:embed all:templates` と check-sync で確かめた。init を実際に打ってはいない

## Verdict

- Verdict: pass(V1 は AGENTS.md の 1 語のずれで、/sync-docs で直せる。merge を止めるものではない)
- Verified: AC1〜AC12 と FR-5・FR-9 の文、受け入れ条件 1 つがコードとテストのコードに対応している(テストは読んだだけ)。静的解析はすべて通った(run-static-verify、gofmt、go vet、go1.25.8 の build と vet、skill と template の同期、`git diff --check`)。HEAD の binary で escalate の 3 つの届き方、NOTIFIED `deferred`、show の行、attach・`--replace`・入れ替えられた `id` の detach の拒否、`enabled = false` の拒否、`--org-id` の拒否、`ralph status` と `director status` の行を確かめた。help は動きと合う
- Partially verified: AC8(V1)。AC9(コードを読んだ範囲。実行は /test)
- Not verified: テストの実行、連結した worktree からの `git worktree list` の分岐、approve の 2 process の同時実行、binary での ack の期限の送信、`ralph init` の実行
- Insight event: オーケストレーターが前の計画と同じく別のコミットで記録する前提で、この commit には jsonl を入れていない。`--events-dir` を一時の場所にして作った正規の 1 行は次のとおり: `{"schema":1,"ts":"2026-10-14T19:02:53Z","slug":"org-director-attach","flow":"standard","phase":"verify","cycle":1,"verdict":"pass","findings":{"critical":0,"high":0,"medium":0,"low":1},"triage":{"action_required":0,"worth_considering":0,"dismissed":0},"source":"skill"}`

## Cycle 2

- Date: 2026-10-15
- Verifier: verifier subagent (Claude)。パイプラインの 2 回目(`.harness/state/standard-pipeline/cycle-count.json` の cycle は 2 で、既定の上限 2 の最後)。cross-review cycle 1 の ACTION_REQUIRED 1 件(`docs/reports/cross-review-triage-org-director-attach.md`)の直し 634b019a と、self-review cycle 2(71dac14e、CRITICAL 0・HIGH 0・MEDIUM 0・LOW 5、Merge: yes)の LOW の直し f4c1a394 のあとに走らせた
- Scope: `git diff 7cd73066..HEAD`(HEAD 35a453aa、10 コミット、16 ファイル、+2451/-28)。Go の変更は `internal/org/director.go`、`internal/org/director_test.go`、`internal/cli/org_director.go`、`internal/cli/org_director_test.go` の 4 ファイルだけ。残りは self-review・test・sync-docs・cross-review の記録、sync-docs 624f603b の文書(AGENTS.md、仕様の 8a-1 の注、quality-gates の 2 面、計画の Progress)、f4c1a394 の tech-debt の 1 行。AC の確かめ直しに要るところは `git diff ef575748...HEAD` も読んだ。テストは走らせていない(/test の担当)。この節の行番号は HEAD 35a453aa のもの。上の cycle 1 の節の行番号は 676e7f98 のままで、たとえば `Attach` は 196 行から 216 行に、`Detach` は 252 行から 272 行に動いた
- Plan: `Status: Approved`。`./scripts/plan-visual.sh digest` は 27414f27525b で、`- Approved:` の digest と同じ
- Evidence: `docs/evidence/verify-2026-10-14-org-director-attach.md` の `## Cycle 2 (HEAD 35a453aa)`。C2-1 は run-static-verify の出力、C2-2 は Go と同期の検査、C2-3 は HEAD の binary で拒否と失敗のたびに director.jsonl を直前の写しと `cmp` した probe、C2-4 は RLIMIT_FSIZE で本物の途中までの書き込みを作ろうとした試み、C2-5 は `director status --help`。wrapper の全ログは `docs/evidence/verify-2026-10-14-204359.log`(gitignore の対象)
- Insight event: 指示どおり記録していない(オーケストレーターが別のコミットで記録する)。値は cycle 2、verdict pass、findings はすべて 0(下の情報 4 件は数えない)

### cycle 2 の直しの確かめ

| 変更 | Status | Evidence |
| --- | --- | --- |
| cross-review #1: write か close が失敗したら、同じロックの中で追記前の大きさに切り戻す | 満たす(コードとテストのコードで確認。本物の途中までの書き込みは binary で再現できなかった。C2-4) | `appendLocked`(`internal/org/director.go:359`)は write の失敗(`:379-381`)と close の失敗(`:383-384`)で `restoreLocked(len(raw), …)`(`:393`)を呼ぶ。`raw` は `Attach`(`:216`)と `Detach`(`:272`)が同じ `withLock` の中で `readRaw`(ファイル全体を読む)したもので、`appendLocked` を呼ぶのはこの 2 か所(`:247`、`:307`)だけ。open と marshal の失敗は何も書かないので戻さない。テスト(読んだだけ): `TestDirectorAttach_AReplaceCutOffPartwayIsRestored`(`director_test.go:308`。detached の行を書き終えて attached の途中で ENOSPC、元の bytes に戻り、次の replace は d2 を得る)、`TestDirectorStore_PlainAttachAndDetachCutOffPartwayAreRestored`(`:393`。ファイルがなかった attach は空のファイルに戻る。detach も同じ)、`TestDirectorAttach_ACloseFailureAfterAFullWriteIsRestored`(`:453`。seam が先に閉じ、本物の `Close` を失敗させる)。seam の `write` と `truncate`(`:166-167`)は unexported で、本番では nil のまま `(*os.File).Write` と `os.Truncate` になる |
| 戻せたときだけ「nothing was recorded」と言い、戻せなかったときは言わない | 満たす | `directorAppendFailed`(`director.go:412`)の `switch`。`ErrDirectorWriteNotRestored` を包むエラーには `org: director: <verb>: ` だけを足し、`appendLocked` の文で終える(`:414-415`)。そうでなく今の登録があれば「nothing was recorded, so <登録> is still the current registration」(`:416-417`)、登録がなければ「nothing was recorded」(`:418-419`)。`restoreLocked` は truncate が失敗したときだけ、元のエラー・sentinel・truncate のエラーの 3 つを `%w, and %w (%w), so it may hold part of the write` で包む(`:398-399`)。テスト: `TestDirectorAttach_AReplaceThatCannotBeCutBackSaysSo`(`:355`。文全体を等号で比べ、3 つの `errors.Is`、truncate の引数が path と `len(before)` であること、ファイルが途中までの書き込みのままで Read が登録なしと壊れた 1 行を返すことを見る)。戻せた側の 3 本は `ErrDirectorWriteNotRestored` を包まないことも見る |
| 戻せなかったときの確かめ方(self-review C2-L1) | 満たす(helper のテストだけ) | store の文はコマンドを持たない。CLI の `notRestoredError`(`internal/cli/org_director.go:314`)が sentinel のときだけ `rt.command("ralph org director status")` を足すので、`--state-dir` を渡した呼び出しには `--state-dir` が付く。attach(`:121`)と detach(`:183`)の RunE がこれを通る。テスト `TestOrgDirector_NotRestoredErrorNamesTheStatusCommand`(`org_director_test.go:243`)は `--state-dir` のありとなし、ほかのエラー(`ErrDirectorAttached` を含む)がそのまま返ることを見る。RunE の 2 か所がこの helper を通ることにはテストがない(self-review の「Fixes after the cycle-2 review」が known gap と書いている)。2 か所とも通ることはコードを読んで確かめた |
| `org: director:` を 1 回にする(C2-L2) | 満たす(binary でも確認) | `appendLocked` の 4 つのエラー(`:367`、`:373`、`:381`、`:384`)は接頭辞を持たず、`directorAppendFailed` が 1 回だけ足す。C2-3 の読み取り専用の 3 行でも `org: director:` は 1 回。path が 2 回出る別の重なりは下の C2-V2 |
| コメントの指す先(C2-L3) | 満たす | 「cross-review finding」「finding 1」「mutant D5」は Go の 4 ファイルにない(grep 0 件)。コメントは起きることを書く形になった(`director.go:66-73`、`director_test.go:848-853`) |
| doc の言い回し(C2-L4) | 満たす | `ErrDirectorWriteNotRestored`(`:66-73`)は「a failed open or marshal writes nothing, and a failed write or close is cut back」、`Attach`(`:198-215`)と `Detach`(`:260-271`)はどちらも「ends one of two ways」と書く。中身がコードと合うことは下の「文書のずれ(cycle 2)」 |
| tech-debt の 1 行(C2-L5) | 満たす | `docs/tech-debt/README.md:248`、列は 5。引数の数(`printStatusEmpty` 10、`printStatusJSONAllOrgs` 11、`printStatusTableAllOrgs` 10。`internal/cli/status.go:401`、`:586`、`:622`)は HEAD と合い、self-review C2-L5 の「10〜12」より正確。Related の計画の path は active のもので、`/pr` の archive が書き換える |
| test 報告の生き残った mutant(ほかの id の detached が今の登録を終わらせる) | 満たす(テストのコードで確認) | `TestDirectorRead_ADetachedForAnotherIDLeavesTheCurrentRegistration`(`director_test.go:854`)。attached d1 と detached d2 の 2 行だけのファイルで、current が d1、IgnoredEvents が 1。畳み込みの id の検査を外すと current が nil になり赤くなる(読んで判断した。実行は /test) |
| cycle 1 の verify の V1・V4(sync-docs 624f603b) | 直った | V1: AGENTS.md に「interim」はなく(`grep -c interim AGENTS.md` は 0)、escalate の項は「with `[org.director].enabled` and a cross-org watch accepting work it only records the item, for the director and the ack deadline; otherwise it also sends it to the human at once」になった。`escalateDelivery`(`internal/org/escalate.go:305` の doc)の分岐と合う。V4: `director status --help` に「With enabled false the state is disabled whether or not a registration is recorded; one that is still recorded is named and its fields are shown」が入った(`org_director.go:212-213`、C2-5)。`readOrgDirectorReport` は enabled が false なら登録によらず `disabled` にし、`printOrgDirectorReport` は登録があれば欄を出すので合う。V3(計画の checkbox)も 624f603b で `[x]` になった |

### AC の確かめ直し(HEAD 35a453aa)

cycle 2 の Go の差分は、director の store の書き込みが失敗したときの経路、CLI の attach と detach のエラーの返し方、`director status` の help の 1 文だけだった。escalate、inbox、deadline、approve、config、`ralph status` の Go と skill の 4 面は変わっていない(`git diff --stat 7cd73066..HEAD -- '*.go'` は上の 4 ファイル)。

| Acceptance criterion | Status | Evidence |
| --- | --- | --- |
| AC1 | 満たす(cycle 1 のまま) | 設定の 4 面は差分の外 |
| AC2 | 満たす(binary でも確認) | 拒否: HEAD の binary(C2-3)で、登録がないときの detach、`--driver gpt`、`--org-id`、`--config` の `enabled = false` の attach は、どれも終了コード 1 で director.jsonl を作らない。d1 を attach したあとの 2 つ目の attach、`--id` のない detach、`--id` と `--force` の併用、`x1`、違う `id`(`d7`)、`--replace` で入れ替えられた `d1` の detach は、どれも終了コード 1 で、director.jsonl は直前の写しと `cmp` で同じ。失敗: 読み取り専用にした director.jsonl への `--replace`、`detach --id d1`、`detach --force` は open で失敗して「nothing was recorded, so d1 … is still the current registration」と出し、ファイルは `cmp` で同じ。途中まで書けた失敗が元の bytes に戻ることは、テストの seam で読んで確かめた(上の表の 1 行目)。元の bytes に戻らない場合は 2 つあり、どちらも doc に書いてある。ファイルがなかった attach の失敗は空のファイルを残す(`appendLocked` の doc。空は登録なしと読まれ、次の attach は d1 を得る)。切り戻し自体が失敗したときはファイルが途中までの書き込みを持ちうるが、エラーはそう言い、「nothing was recorded」と言わない。A(d1)→ B の `--replace`(d2)→ A の `detach --id d1` の順では A が断られ、d2 が残る(C2-3) |
| AC3〜AC5 | 満たす(cycle 1 のまま) | `escalate.go`、`inbox.go`、`deadline.go`、`org_inbox.go`、`org.go` は差分の外。attach の見張りの警告(C2-3 の stderr)は cycle 1 と同じ文 |
| AC6 | 満たす | `orgDirectorLine` と `status.go` は差分の外。C2-3 の `director status` は attached d2 とその欄を出す |
| AC7 | 満たす | skill の 4 面と `ralph-workflow.md` の 2 面は差分の外。`check-skill-sync.sh`(14 skill)と `check-sync.sh`(DRIFTED 0)が通る |
| AC8 | 満たす | cycle 1 の V1 が直った(上の表の最後の行)。cycle 2 で足された文書が書き込みの失敗について誤ったことを言っていないことは下の「文書のずれ(cycle 2)」 |
| AC9 | 満たす(コードを読んだ範囲) | 新しい store のテストは `newTestDirectorStore`(`director_test.go:23`、`t.TempDir()`)を使い、CLI のテストはファイルに触れない `orgDirectorRuntime` の値(`org_director_test.go:253`)だけを使う。osascript と herdr を打つ経路はない |
| AC10〜AC12 | 満たす(cycle 1 のまま) | `approve.go`、`org_approve.go`、skill の 4 面は差分の外 |

FR-5・FR-9 の文、受け入れ条件、安全の不変条件は、escalate と期限の経路が変わっていないので cycle 1 の判定のまま。

### 静的解析(cycle 2)

| Command | Result | Notes |
| --- | --- | --- |
| `./scripts/run-static-verify.sh`(changed scope) | pass(rc 0) | local verifier の全項目が OK。check-sync PASS、check-pipeline-sync ok、`check-skill-sync: 14 skill(s) in lock-step`、check-template-purity PASS、tech-debt の計画の参照 OK、`gofmt: ok`、golangci-lint `0 issues.`、`secret-scan-branch: scanned ef575748..35a453aa against origin/main: clean`。branch の差分に `scripts/ralph-config.sh` があるので、言語の範囲は cycle 1 と同じく full に広がった |
| `gofmt -l .` | pass | 出力なし |
| `go vet ./...`(go1.26.0) | pass | 出力なし |
| `GOTOOLCHAIN=go1.25.8 go build ./...` と `go vet ./...` | pass | 出力なし。`%w` を 3 つ持つ `fmt.Errorf`(`director.go:399`)は Go 1.20 から使える |
| `./scripts/check-skill-sync.sh` | pass | 14 skill が lock-step |
| `./scripts/check-sync.sh` | pass | IDENTICAL 169、DRIFTED 0、KNOWN_DIFF 5 |
| `git diff --check ef575748...HEAD` と `git diff --check 7cd73066..HEAD` | pass | 出力なし |
| cycle 2 で変わったファイルの U+FFFD、追加行の TODO・FIXME・`fmt.Print`・`/Users/` | pass | 0 件 |

### 文書のずれ(cycle 2)

| Doc / contract | In sync? | Notes |
| --- | --- | --- |
| `ErrDirectorWriteNotRestored`、`Attach`、`Detach`、`appendLocked`、`restoreLocked`、`directorAppendFailed` の doc(`director.go:66-73`、`:198-215`、`:260-271`、`:343-358`、`:389-392`、`:404-411`) | yes | 「戻せたら nothing was recorded、戻せなければ sentinel を包んでそう言わない」「open と marshal の失敗は何も書かない」「作ったファイルは消さずに空に戻す」「appendLocked のエラーは `org: director: ` で始まらず、呼ぶのは Attach と Detach だけ」「ロックなしの Read だけが失敗と切り戻しの間に途中の行を見うる」は、どれも上のコードと合う |
| `notRestoredError` の doc(`org_director.go:307-313`) | yes | sentinel のときだけ status の打ち方を足し、ほかは素通しする。`attachedError` と同じ作り |
| README の director の段落(`README.md:258`)、AGENTS.md の `director.go` の項、`/director` skill、`/org` skill の `director` の行(`.claude/skills/org/SKILL.md:186`)、仕様の 8a-1 の注(`docs/specs/2026-10-07-org-multi-org-director.md:84`)、quality-gates の 2 面 | yes | 書き込みの失敗について言う文はない。「何も書かずに断られる」「writes nothing」は `enabled = false` と違う `id` の拒否についての文で、拒否は書く前に起きるので切り戻しと関係しない。仕様の「1 つのロックの下で書く」、AGENTS.md の「under `director.lock`」、attach の help の「in one write, under one lock」は今も正しい。切り戻しが失敗しても何も残らないと言う文書はない |
| `docs/tech-debt/README.md:248` | yes(情報 C2-V3) | 引数の数と Trigger は HEAD と合う |
| 計画の Progress checklist | 一部遅れ(情報 C2-V1) | 下の表 |

| ID | Severity | Where | Finding |
| --- | --- | --- | --- |
| C2-V1 | INFO | 計画の `## Progress checklist` | 最後の行は sync-docs cycle 1(624f603b)で、cross-review cycle 1 の ACTION_REQUIRED #1、その直し 634b019a、self-review cycle 2 の直し f4c1a394 の行がない。self-review cycle 2 の Follow-ups 4 が /sync-docs で足すよう勧めている。オーケストレーターが更新する欄なので失敗とはしない |
| C2-V2 | INFO | `director.go:373`、`:381`、`:384` | `os.OpenFile`、`(*os.File).Write`、`Close` のエラーは path を含む `*PathError` なので、`open %s: %w` と包むと path が 2 回出る。C2-3 の binary では `open $SCRATCH/…/director.jsonl: open $SCRATCH/…/director.jsonl: permission denied` になった。本物の write の失敗は `append to <path>: write <path>: no space left on device` になるはずで、seam が素の `syscall.ENOSPC` を返すテストの期待値(`append to <path>: no space left on device`)とは形が違う(おそらく。本物の ENOSPC は未確認)。cycle 1 の 8b2effab からあり、受信箱の `inbox.go:400` も同じ書き方なので、cycle 2 が持ち込んだものではない。動きは変わらない |
| C2-V3 | INFO | `docs/tech-debt/README.md:248` の Why deferred | 「the last pipeline run (cycle 2 of the default 2)」は HEAD では正しいが、上限を上げて 3 回目を回すと偽になる。今のずれではない |
| C2-V4 | INFO | `/director` skill の「昇格」の 5 | 「`ralph org director status` に出る `id` が覚えている `id` と違えば入れ替えられた」と書くが、登録がない(`not attached`)場合は名指ししていない。人の `detach --force` と、cycle 2 で起こりうると doc に書かれた「切り戻せなかった replace」(detached の行が残り、attached が壊れた行になる)がこの状態を作る。「違う」に含めて読めるので誤りではない。8a-2 か次に skill に触れるときに 1 句足すとよい |

### 観察

- C2-3 は PATH を git だけにして打ったので、osascript、herdr、agmsg は動いていない。台帳と cwd は scratch。cycle 1 の probe と同じく見張りなしの attach の警告が出る
- C2-4: `ulimit -f 1` の下で `attach --replace` を 2 通り打った。cwd を 150 文字伸ばして detached までが上限に収まる形にしたものと、director.jsonl が初めから上限を超えているもの。どちらも追記がそのまま通り、ファイルは上限を超えた(435 から 1270 bytes、1898 から 2435 bytes)。この macOS ではこの追記を上限で止められず、途中までの書き込みの切り戻しは Go のテストの seam でしか確かめていない。止まらない理由はおそらく `O_APPEND` の書き込みに上限の検査が効いていないこと(未確認)
- ralph のプロセスが write と切り戻しの間で死ぬと(SIGKILL、電源断)、途中までの書き込みが残り、エラーも出ない。doc の「A write that fails ends one of two ways」はエラーを返す失敗についての文で、この場合を否定していない。cycle 1 の前からある性質で、読み手は `ralph org director status`(登録なし、または壊れた行の警告)で気づける

### Coverage gaps(cycle 2)

- テストの実行(/test の担当)。新しい store のテスト 5 本と CLI の helper のテストを、go1.26 と `GOTOOLCHAIN=go1.25.8` の両方で走らせるのは /test
- 本物の ENOSPC で途中まで書けた追記を binary で再現していない(C2-4)。truncate の失敗も binary では起こしていない
- attach と detach の RunE が `notRestoredError` を通ることのテストがない。2 か所のどちらかを `return err` に戻しても赤くならない(読んで判断した。mutant は走らせていない)
- 末尾に改行のない raw への追記が失敗した場合(先頭の `\n` ごと切り戻す)のテストがない(self-review cycle 2 の Coverage gaps と同じ)
- 足すと確信がいちばん増える確認: 小さなディスクイメージ(`hdiutil` の 1 MB の HFS+ など)を一杯にし、その上の台帳で `attach --replace` を打って、ENOSPC で途中まで書けた追記が元の bytes に戻ることを binary で見る。CLI の 2 か所の配線は、store の seam を CLI のテストから差せるようにすれば 1 本で固定できる

### 判定(cycle 2)

- Verdict: pass
- Verified: cross-review #1 の直し(write と close の失敗の切り戻し、戻せたときと戻せなかったときの文の分け方)と self-review cycle 2 の C2-L1〜C2-L5 の直しを、コードとテストの名前で辿った。AC2 は HEAD 35a453aa の binary で、拒否 10 通りと open の失敗 3 通りのどれもが director.jsonl を変えないこと(`cmp`)、入れ替えられた `id` の detach が断られることを確かめた。ほかの AC は経路が変わっていないことを diff で確かめた。cycle 1 の V1・V3・V4 は 624f603b で直った。文書は、切り戻しが失敗しても何も残らないとは言っていない。静的解析はすべて通った(run-static-verify、gofmt、go vet、go1.25.8 の build と vet、skill と template の同期、`git diff --check`)。計画の digest は承認時と同じ
- Partially verified: 途中まで書けた追記の切り戻しと、切り戻しの失敗(テストのコードを読んだだけ。binary では再現できなかった)。CLI の `notRestoredError` の配線(helper のテストと、コードを読んだ範囲)
- Not verified: テストの実行(/test)、本物の ENOSPC、RunE の配線の mutant
- Findings: CRITICAL 0、HIGH 0、MEDIUM 0、LOW 0。情報は 4 つ(C2-V1〜C2-V4)

## Cycle 3

- Date: 2026-10-15
- Verifier: verifier subagent (Claude)。パイプラインの 3 回目。cross-review cycle 2 の ACTION_REQUIRED 1 件(受信箱の wait の失敗を時間切れと見分けずに打ち直す)を直すために上限 `RALPH_STANDARD_MAX_PIPELINE_CYCLES` を 3 に上げた回で、`.harness/state/standard-pipeline/cycle-count.json` は 2 のまま(意図どおり。触っていない)。直し c8a9227b と、self-review cycle 3(a16b5c21、CRITICAL 0・HIGH 0・MEDIUM 0・LOW 3、Merge: yes)の LOW の直し 8ac59595 のあとに走らせた
- Scope: `git diff a094c93b..HEAD`(HEAD d2eaa665、5 コミット、9 ファイル、+144/-20)。Go の変更は `internal/org/escalate.go` のコメント 5 行と、テスト 2 ファイルのコメント 1 行ずつで、動きは変わらない。残りは `/director` skill の 4 面の受信箱の節(1 hunk)と、self-review の記録と insight の 1 行。AC の確かめ直しには `git diff ef575748...HEAD` も読んだ。テストは走らせていない(/test の担当)。この節の行番号は HEAD d2eaa665 のもの。self-review cycle 3 が引く `escalate.go:295` の時間切れの文は、8ac59595 がコメントを足したので 300 行に動いた
- Plan: `Status: Approved`。`./scripts/plan-visual.sh digest` は 27414f27525b で、`- Approved:` の digest と同じ
- Evidence: `docs/evidence/verify-2026-10-14-org-director-attach.md` の `## Cycle 3 (HEAD d2eaa665)`。C3-1 は run-static-verify の出力、C3-2 は Go と同期の検査、C3-3 は HEAD の binary で `ralph org wait --inbox` の時間切れ 2 通り、件のある 1 通り、時間切れでない失敗 10 回(8 通りの失敗。受信箱がディレクトリの場合は `--timeout-ms` を変えて 3 回)の終了コードと stderr、C3-4 は読めない受信箱での `ralph org watch --all --once` 1 回、C3-5 は終了コードを決める行の grep。wrapper の全ログは `docs/evidence/verify-2026-10-14-222813.log`(gitignore の対象)
- Insight event: 指示どおり記録していない。値は cycle 3、verdict pass、findings は LOW 1(下の C3-V1)。情報 3 件は数えない

### cycle 3 の直しの確かめ

| 変更 | Status | Evidence |
| --- | --- | --- |
| cross-review cycle 2 #1: Codex の項は、終了コード 1 で stderr が `no open inbox item arrived within` を含むときだけ打ち直す(c8a9227b) | 満たす(binary でも確認) | skill の 107〜110 行。照合する文は `WaitInbox` の時間切れのエラー `org: wait --inbox: no open inbox item arrived within %d ms`(`internal/org/escalate.go:300`)の部分文字列。root が `SilenceErrors`(`internal/cli/root.go:23`)なので cobra は出さず、`cmd/ralph/main.go:21` の `Fprintln(os.Stderr, err)` が 1 行だけ出し、`:31` で終了コード 1 になる。C3-3 の P1(空の台帳、300 ms)と P8(acked の件だけ)は、どちらも rc 1、stdout 0 行、stderr 1 行で、stderr が skill の文を含む。P1 は 339〜357 ms で返った(新しく作った binary の最初の 1 回だけ 1654 ms) |
| 時間切れでない終了コード 1 はどれも skill の文を含まず、打ち直されない | 満たす(binary で 10 回) | 終了コード 1 の元は 4 つ。(a) cobra の flag の解析(P6 `invalid argument "abc" for "--timeout-ms" flag`)。(b) `--inbox` と `--org-id`・`--seat`・`--until` の併用の拒否(`internal/cli/org.go:1174-1178`、P5a〜P5c)。(c) `newOrgRuntime`(`org.go:1179`)。旧台帳の guard は `orgLedgerMutating` のときだけエラーを返し(`internal/cli/org_legacy_ledger.go:48`、`:61`)、read-only の `wait` には stderr に `note:` を書いて nil を返すので、残るのは `resolveOrgConfig` の失敗(`org.go:246` の `org: load config: %w`)。`--config` の壊れた TOML(P4a)と cwd の壊れた `ralph.toml`(P4b)で確かめた。(d) `WaitInbox` の read の失敗(`escalate.go:283`)。inbox.jsonl がディレクトリ(P2a〜P2c)と mode 000(P3)で確かめた。`escalate.go:263` の `no inbox store is configured` は、`newOrgRuntimeAt` が必ず `wireOrgInbox` を呼ぶ(`org.go:270`)ので CLI からは届かない。10 回とも rc 1、stdout 0 行、stderr 1 行で、どれも skill の文を含まない。`--timeout-ms 300000` を付けても 70 ms 未満で返った(P2b、P3、P4a、P4b)ので、skill の「すぐ返り」と合う |
| 終了コード 3 は `wait --inbox` から届かない | 満たす | `main.go:28-29` が 3 で終わるのは `errors.Is(err, cli.ErrUpgradeDriftRemaining)` のときだけ。この sentinel を包むのは `internal/cli/upgrade_v2.go:226` と `:302` だけで、`migrate.go:989` は upgrade の経路でそれをそのまま返す(C3-5)。上の (a)〜(d) のエラーは cobra、`internal/cli/org.go`、`internal/config`、`internal/org` で作られる。`internal/config` と `internal/org` は `internal/cli` に import される側なので、cli の sentinel を作れない。Go の panic の終了コード 2 と signal での終わりは下の C3-V3 |
| Claude Code の項: `--timeout-ms 0 2>&1`、件の行を出さずに返ったら失敗で打ち直さない(8ac59595、self-review C3-L2) | 満たす(binary でも確認) | skill の 101〜106 行。`WaitInbox` は `timeout > 0` のときだけ期限を作る(`escalate.go:270`)ので、`--timeout-ms 0` の run は時間切れの文を出さない。`WaitInbox` は `(open, nil)`(`:292`)か `(nil, err)`(`:263`、`:283`、`:300`)を返し、`runOrgWaitInbox` は件があるときだけ件の行を書く(`org.go:1183-1187`)ので、件の行が出るのは終了コード 0 のときだけ。P7(open の件 1 つ)は rc 0 で、タブで区切った `e1`、`o1`、`QUESTION`、`probe` の 1 行、P2c(読めない受信箱)は rc 1、件の行なしで 29 ms で返った。`2>&1` の副作用は C3-V2 |
| Go のコメント(self-review C3-L1) | 満たす | `escalate.go:295-299` のコメントは、skill が stderr のこの文で時間切れと失敗を見分けること、一致したときだけ打ち直すこと、文を変えたら skill の受信箱の節とその 3 つの写しも直すことを書く。写しは `.agents/skills/director/`、`templates/base/.claude/skills/director/`、`templates/base/.agents/skills/director/` の 3 つで、数が合う。文を固定するテスト 2 本(`internal/org/escalate_test.go:1216`、`internal/cli/org_inbox_test.go:1222`)の期待値の上に、skill を指す 1 行がある。期待値 `no open inbox item arrived within 300 ms` は skill の文を含む。skill の文を読む機械の検査はない(Coverage gaps) |
| 失敗のときに止まる先(self-review C3-L3) | 満たす(最後の 1 文を除く) | skill の 112〜116 行。打ち直さない、受信箱を待つのをやめる、エラーの文を人に伝える、人の指示を待つ、detach しない(登録は残る)。昇格の 5(68〜74 行、ack と resolve と send と start をしない、detach も打たない)と同じ扱いで、終わりの節(215〜226 行、detach は終わるときに打つ)とも食い違わない。最後の文「見張り(`ralph org watch --all`)が動いていれば、件は期限で人に届く」は、`ralph.toml` の失敗では正しいが、受信箱が読めない失敗では正しくない(C3-V1) |
| skill の 4 面 | 満たす | md5 は 4 面とも `98aa4417031ee20463a94d3594a91b48`。`check-skill-sync.sh`(14 skill)と `check-sync.sh`(DRIFTED 0)が通る。cycle 3 の skill の差分は 1 hunk(`@@ -98,12 +98,22 @@`)だけ |

### AC の確かめ直し(HEAD d2eaa665)

cycle 3 の変更は、skill の受信箱の節と Go のコメントだけだった。Go の動き、AC8 の一覧の文書、計画は変わっていない(`git diff 9548a8cd..HEAD --stat` を README、AGENTS.md、`docs/specs`、`docs/quality`、`templates/base/docs`、`internal/org/prompts`、`/org` skill、`.claude/rules` に絞ると空)。

| Acceptance criterion | Status | Evidence |
| --- | --- | --- |
| AC1〜AC6、AC9〜AC11 | 満たす(cycle 2 のまま) | 8ac59595 の Go の差分はコメント 7 行だけで、動く行はない |
| AC7 | 満たす | 4 面の md5 が同じ。`check-skill-sync.sh` と `check-sync.sh` が通る。`ralph-workflow.md` の 2 面の 10 行目と 28 行目に `/director` がある |
| AC8 | 満たす | AC8 の一覧の文書は cycle 3 で変わっていない。`wait --inbox` を書くほかの場所(`/org` skill の 173 行と 313 行、README の 124 行と 258 行、仕様の 52 行・87 行・206 行、`wait` と `escalate` の help)は、どれも director に終了コード 1 のたびに打ち直せとは書いていない。`/org` skill の 173 行と `wait` の help の「届かなければ終了コード 1」は時間切れの説明で、正しい |
| AC12 | 満たす | cycle 3 の skill の差分は受信箱の節の 1 hunk で、「答える順」(136〜166 行)と「計画の承認」(168〜192 行)は変わっていない。新しい段落は resolve に触れない。`/org` skill は変わっていない |

### 静的解析(cycle 3)

| Command | Result | Notes |
| --- | --- | --- |
| `./scripts/run-static-verify.sh`(changed scope) | pass(rc 0) | local verifier の全項目が OK。check-sync PASS、check-pipeline-sync ok、`check-skill-sync: 14 skill(s) in lock-step`、check-template-purity PASS、tech-debt の計画の参照 OK、`gofmt: ok`、golangci-lint `0 issues.`、`secret-scan-branch: scanned ef575748..d2eaa665 against origin/main: clean`。branch の差分に `scripts/ralph-config.sh` があるので、言語の範囲は cycle 1・2 と同じく full に広がった |
| `gofmt -l internal` | pass | 出力なし |
| `go vet ./...`(go1.26.0) | pass | 出力なし |
| `GOTOOLCHAIN=go1.25.8 go build ./...` と `go vet ./...` | pass | 出力なし |
| `./scripts/check-skill-sync.sh` | pass | 14 skill が lock-step |
| `./scripts/check-sync.sh` | pass | IDENTICAL 169、DRIFTED 0、KNOWN_DIFF 5 |
| `git diff --check ef575748...HEAD` と `git diff --check 9548a8cd..HEAD` | pass | 出力なし |
| branch の 58 ファイルの U+FFFD、cycle 3 の追加行の TODO・FIXME・`fmt.Print`・`/Users/` | pass | 0 件。U+FFFD の grep は、U+FFFD を 1 つ入れた scratch のファイルで 1 件を返すことを先に確かめた |

### 文書のずれ(cycle 3)

| Doc / contract | In sync? | Notes |
| --- | --- | --- |
| `/director` skill の受信箱の節(4 面、97〜119 行) | 一部(C3-V1) | 打ち直しの条件、Claude Code の項、止まる先は上の表のとおりコードと binary の動きに合う。最後の 1 文だけが、受信箱が読めない失敗のときに合わない |
| `WaitInbox` のコメント(`escalate.go:295-299`)とテスト 2 本のコメント | yes | 上の表の「Go のコメント」の行 |
| README、AGENTS.md、仕様 FR-9 の 8a-1 の注、quality-gates の 2 面、`/org` skill の 4 面、leader の雛形 | yes | cycle 3 で変わっておらず、director の受信箱の wait の打ち直しについて書いた文はない。仕様の 87 行の「Claude Code は Monitor で待ち、codex は timeout 付きの wait を繰り返す」は今の skill と合う。leader の雛形は `wait --inbox` に触れない |
| 計画の Progress checklist | 一部遅れ(C3-V4) | 下の表 |

| ID | Severity | Where | Finding |
| --- | --- | --- | --- |
| C3-V1 | LOW | `.claude/skills/director/SKILL.md:115-116` と 3 つの写し | 「見張り(`ralph org watch --all`)が動いていれば、件は期限で人に届く」は、同じ段落の 112 行が挙げる 2 つの失敗のうち、受信箱(`inbox.jsonl`)が読めない失敗では成り立たない。見張りの期限の処理も同じファイルを同じ `InboxStore.Read` で読み(`internal/org/deadline.go:177`)、読めなければ `Err` を返す。見張りは `watchdog: inbox deadlines skipped this cycle: …` を stderr(`--detach` なら `watch.log`)に 1 行書くだけで、件を送らない(`internal/org/watch.go:627-629`)。C3-4 で mode 000 の受信箱に `watch --all --once` を 1 回回すと、終了コード 0 でこの 1 行が出て、`escalations.jsonl` は作られなかった。受信箱が読めるようになるまで、open と acked の件はどちらの期限でも人に届かない。新しい escalate は記録できず、その場で人に知らせる(`escalate.go:184-186` の `alertUnrecordedEscalation`)ので、新しい件は期限を待たずに届く。`ralph.toml` の失敗では文は正しい。見張りは起動時に組み立てた設定で動き(`internal/cli/org_watch.go:90`)、周回ごとに読み直さない。8ac59595 の commit message も同じ言い方をしている。director がする手順(打ち直さない、人に伝える、detach しない)は変わらないが、この文をもとに「件は期限で届く」と人に伝えると、受信箱が読めない場合に人の判断を誤らせる。直し方の例: 最後の文を「`ralph.toml` の失敗なら、見張りが動いていれば件は期限で人に届く。受信箱が読めない失敗では見張りも期限を見られないので、そのことも人に伝える」にする。skill の 4 面を直すので、記録だけの例外には入らない |
| C3-V2 | INFO | `.claude/skills/director/SKILL.md:101-106` | `2>&1` で、件の行とエラーの文のほかに、stderr の `note:` の行も同じ出力に入る。旧台帳の guard は read-only の動詞では断らずに `note:` を 1 行書く(`org_legacy_ledger.go:53-55`、`:73-75`。古い台帳を持つ linked worktree で打ったとき)ので、Claude Code の形では終了コード 0 の出力にも件でない行が混ざりうる。件の行の中身(ID、org、TYPE、本文の 1 行目)は `/org` skill の `wait` の行(173 行)にあり、`note:` で始まる行と取り違えにくい。Codex の形は stderr を「含む」で見るので影響しない |
| C3-V3 | INFO | `.claude/skills/director/SKILL.md:101-116` | 終了コード 0 と 1 のほかの終わり方(Go の runtime の panic の 2、signal での終わり)を skill は名指ししない。Codex の項の打ち直しの条件は終了コード 1 と時間切れの文の両方なので、どちらの終わり方でも打ち直されず、止まる側に倒れる。止まってからの扱いは終了コード 1 の段落として書かれているが、そのまま当てはめて読める。self-review cycle 3 の Coverage gaps(外からの kill)と同じ穴で、cycle 3 が持ち込んだものではない |
| C3-V4 | INFO | 計画の `## Progress checklist` | 最後の行は /sync-docs cycle 2 で、cross-review cycle 2 の ACTION_REQUIRED #1、その直し c8a9227b、self-review cycle 3 とその直し 8ac59595 の行がない。この cycle の /sync-docs で足すとオーケストレーターから聞いている。オーケストレーターが更新する欄なので失敗とはしない |

### 観察

- C3-3 と C3-4 は `env -i` で HOME を scratch にし、PATH を git だけにして打った。osascript、herdr、agmsg は動いておらず、台帳と cwd はどれも scratch。C3-4 の受信箱は読めないので、見張りは期限の処理の手前で止まり、人への経路には何も送っていない
- P1 の最初の 1 回だけ 1654 ms かかった。新しく作った binary を初めて実行したときの macOS の検査の分とみられる(おそらく。未確認)。同じ case をあと 4 回打つと 339〜357 ms だった
- 受信箱が読めないときのエラーは path を 2 回出す(`read <path>: read <path>: is a directory`)。cycle 2 の C2-V2 と同じ包み方で、`inbox.go` に前からある。設定の失敗のエラー(`org: load config: toml: expected character ]`)はファイル名を出さないので、director がこの文をそのまま人に伝えても、cwd の `ralph.toml` と `--config` のどちらが壊れているかは文からは分からない。どちらも cycle 3 より前からある形で、skill の指示(エラーの文を人に伝える)とは食い違わない

### Coverage gaps(cycle 3)

- テストの実行(/test の担当)。8ac59595 はテストにコメントを足しただけなので、期待値は変わっていない
- skill の文と Go のエラーの文が一致することを機械で見る検査はない。self-review C3-L1 の推奨のうち、テストを足す部分は採られず、コメントだけが入った。文を変える人がコメントを読まなければ、skill はずれる
- skill の文を、実際の Claude Code(Monitor、`run_in_background`)と Codex の session で動かしていない。Monitor が `2>&1` の出力をどう渡すか、Codex の shell tool が 300000 ms の wait を許すかは未確認(self-review cycle 3 と同じ)
- C3-4 は `--once` の 1 回だけ。長く動く見張りが周回ごとに同じ read をすることはコードで確かめた(`watch.go:627-629`、`deadline.go:177`)が、binary で何周も回してはいない
- 足すと確信がいちばん増える確認: `/director` skill の受信箱の節から backtick の中の時間切れの文を読み、`WaitInbox` の時間切れのエラーに含まれることを見るテストを 1 本足す(C3-L1 の残り)。C3-V1 は skill の 1 文を直せば閉じる

### 判定(cycle 3)

- Verdict: pass(C3-V1 は skill の 1 文の言い過ぎで、director のする手順は変わらない。merge を止めるものではない)
- Verified: cross-review cycle 2 #1 の直し(Codex は時間切れの文があるときだけ打ち直す、それ以外の終了コード 1 は打ち直さずに人に伝える)と、self-review cycle 3 の C3-L1〜C3-L3 の直しを、コードと HEAD d2eaa665 の binary で辿った。binary では、時間切れ 2 通りの stderr が skill の文を含むこと、時間切れでない失敗 10 回(flag の解析、flag の併用 3 つ、`--config` と cwd の壊れた TOML、ディレクトリの受信箱を 3 つの `--timeout-ms` で、mode 000 の受信箱)が終了コード 1 で skill の文を含まずにすぐ返ること、`--timeout-ms 0` の失敗が件の行なしで返ること、件があれば終了コード 0 で件の行が出ることを確かめた。終了コード 3 が `wait --inbox` から届かないことはコードで確かめた。AC7・AC8・AC12 を確かめ直し、ほかの AC は経路が変わっていないことを diff で確かめた。静的解析はすべて通った(run-static-verify、gofmt、go vet、go1.25.8 の build と vet、skill と template の同期、`git diff --check`、U+FFFD)。計画の digest は承認時と同じ
- Partially verified: 受信箱が読めないときの見張りの動き(`--once` の 1 回と、コードを読んだ範囲)
- Not verified: テストの実行(/test)、実際の Claude Code と Codex の session での skill の文の動き、Monitor の出力の渡し方、Codex の shell tool の時間の上限
- Findings: CRITICAL 0、HIGH 0、MEDIUM 0、LOW 1(C3-V1)。情報は 3 つ(C3-V2〜C3-V4)
