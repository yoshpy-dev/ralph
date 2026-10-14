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
