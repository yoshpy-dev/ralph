# Verify report: 見張りの ALERT を 1 件にまとめ、leader の stall と予約の中の scope_change を上げない(#307 の前半)

- Date: 2026-11-12
- Plan: `docs/plans/active/2026-11-12-watch-alert-dedupe.md`(digest `e5a18ef9477a`、`./scripts/plan-visual.sh digest` の値と `- Approved:` の行が一致)
- Verifier: verifier subagent(Claude Code)、pipeline cycle 1(`cycle-count.json` は cycle 1)
- Scope: `git diff 028a8f6d...HEAD`(HEAD `f45133f0`、19 ファイル、+3903/-109)。origin/main は `028a8f6d` のまま(`git ls-remote` で確認)なので、base との衝突はない
- Evidence: `docs/evidence/verify-2026-11-12-watch-alert-dedupe.log`(git が無視する。スクリプト自身の写しは `docs/evidence/verify-2026-11-11-223059.log`)

## Spec compliance

計画は Progress checklist に 4 つのずれを記録している。D6 の rebase の扱いと AC15(S4b)、L2 の立てる条件の変更(S6)、L3 の受け入れ(merge-base と ls-tree は候補のある周期だけ打つ)、N5 の受け入れ(doc の言い過ぎを tech-debt に足すのは /sync-docs)。下の判定は、これらで改めた計画に照らしたものである。テストの欄は、テストがあり assert が AC を言っていることまでを読んで確かめた。テストの実行は /test に任せた。

| Acceptance criterion | Status | Evidence |
| --- | --- | --- |
| AC1 open か acked のあいだは件が 1 つで、立つたびに `recurred` が 1 つ増え、show と `--json` に回数と最後の時刻が出る | Met | `internal/org/inbox.go:45`(`recurred`)、`:398-420`(`AppendRecurred`: alert_id と count >= 1 を求め、resolved と ALERT でない件を断る)、`:670-676`(畳み込みは状態を変えない)、`:193-194`(JSON の `recurrences` と `last_recurred_at`)。`internal/cli/org_inbox.go:482-488`(`recurred: <n> times (last <ts>)`)、`:496-502`(event 行の `alert_id=`・`count=`)。見張り側は `internal/org/watch.go:1306-1325`(`raiseOrClear` が立ち上がりで `recordRecurrence` を先に呼ぶ)と `internal/org/watch_recurrence.go:94-123`。テストは `TestWatch_Recurrence_AFlappingLeaderStallIsOneAlertItemUntilItIsResolved`(`watch_test.go:3126`、5 回立って件 1・recurred 4)、`TestInboxRead_RecurredCountsFiringsWithoutChangingTheState`、`TestOrgInbox_ShowsTheRecurrences`、`TestOrgInbox_ShowsOneRecurrenceAsOneTime` |
| AC2 resolved のあとは新しい件 | Met | `openAlertItemForKey`(`watch_recurrence.go:58-66`)は resolved の件を飛ばす。`AppendRecurred` も resolved を `ErrInboxResolved` で断り、呼び手は新しい ALERT に倒れる(`:113-118`)。同じテストの後半(`watch_test.go:3197` 以降、e2 ができる) |
| AC3 続きのあいだは leader に agmsg の ALERT を送らない | Met | `watch.go:1314-1316`: `recordRecurrence` が true なら `sendAlert` と `OnSemanticTrigger` の前に戻る。上のテストが ALERT 1 通と watcher の起動 1 回を数える。既存の 2 本(`TestWatch_Stall_AlertsOnlyNeverStops_RecoversAndRefires`、`TestWatch_EnsureWatchdogJoined_TransientFailure_RetriesUntilSuccess`)の期待の変更は S2 に理由つきで記録されている |
| AC4 deadman の前の再発は pending を増やさず、escalate した件に回数が残る | Met | `watch_recurrence.go:96-98`(pending の `Recurrences++`)、`watch.go:255`(`recurrences,omitempty`)、`:1856-1861`(件の記録の直後に count 付きの `recurred`)。テスト `TestWatch_Recurrence_ASeatsAlertFiringAgainBeforeTheDeadmanCarriesItsCountToTheItem`(`:3225`)。2 つの追記のあいだで止まると回数が落ちる穴は tech-debt の新しい行の (2) にある |
| AC5 watch の立て直しで記録が無くても、同じ鍵の片付いていない件に `recurred` を足す | Met | 立ち上がりの側は `recordRecurrence` が受信箱を読む(`watch_recurrence.go:100-110`)。escalate の側は `watch.go:1845` と `escalateAsRecurrence`(`watch_recurrence.go:135-147`、人には送らない)。テスト `TestWatch_Recurrence_ANewWatchFindsTheOpenItemOfTheKeyInTheInbox`(`:3299`)、`TestWatch_Recurrence_AnEscalationWhoseKeyHasAnItemThatIsNotResolvedIsARecurrenceOfIt`(`:3392`) |
| AC6 片付いていない QUESTION・BLOCKED・RESULT の件があるあいだ、leader の stall は立たず受信箱にも入らない | Met | `watch.go:1257`(`stalled && !leaderIsWaiting`)、`:1155-1188`(`orgHasOpenLeaderItem` は `escalateTypes` = QUESTION・BLOCKED・RESULT の、resolved でない件を数える)。受信箱を読むのは stall しそうな leader のときだけで、周期に 1 回まで。テスト `TestWatch_LeaderStall_IsNotRaisedWhileItsOrgHasAnItemItsLeaderRaised`(`:3762`)と境界の `:3814` |
| AC7 ほかの座席が `working` で文字列が変わるあいだは立たず、変わらなくなれば今までどおり立つ | Met | `watch.go:1051`(leader 以外を先に、leader を最後に評価)、`:1258-1259`、`seatBusy`(`:1197-1203`: 前の周期の返事があり、変わっていて、`herdrAgentStatus` が `working`)。テスト `:3863`、`:3906`、`:3938` |
| AC8 予約の中だけと、自分の新しい計画と報告だけの変更では立たない | Met | `watch.go:1275-1278`(予約のある org は `evaluateReservedScope` へ)、`watch_scope.go:398-449`、`:332-353`(`dropOwnRecords`)、`:124-134`(`ownRecordCandidate`)。実際の git のテスト `TestWatch_ReservedScope_ChangesInsideTheReservationRaiseNothing`(`watch_scope_test.go:390`)と `TestWatch_ReservedScope_TheOrgsOwnNewPlanAndReportsRaiseNothing`(`:428`、origin の 3 つの見つけ方、archive への移動とその commit を含む) |
| AC9 予約の外の変更で立ち、本文に外のパスが出る。外から中への移動を 1 周期で commit しても移動元が出る | Met | `watch_scope.go:70-87`(rename の両側)、`:278-288`(`--no-renames`)、`:430-441`(L2 で改めた条件: 前の周期に無かった外のパスか、外を触る commit)。`TestWatch_ReservedScope_AChangeOutsideRaisesScopeChangeNamingItsPaths`(`:476`、7 つの場合を実際の git で) |
| AC10 予約のない org は今までどおり | Met | `watch.go:1279-1287` は前と同じ `git status --porcelain` の比較(`realGitStatus` は `runGitBounded` に移っただけ)。`TestWatch_ReservedScope_RunWatchAllReadsGitRunAndAnOrgWithoutOneKeepsGitStatus`(`:1110`) |
| AC11 除く規則が役割 prompt の書かせる `docs/` のパスを覆う | Met | `TestOwnRecord_CoversTheDocsPathsTheRolePromptsWrite`(`watch_scope_test.go:1188`): 埋め込みの prompt の `docs/<dir>` をすべて拾い、`docs/plans/`・`docs/reports/` 以外は理由つきの許可の一覧に照らす。両向き(一覧に残った古い項目も落とす) |
| AC15 rebase で入った upstream の commit では立たない | Met | `watch_scope.go:278-288`(`git log ... <prev>..<head> --not <ref>`、`--no-show-signature` は a650ae2e)。`TestWatch_ReservedScope_UpstreamCommitsBroughtInRaiseNothing`(`:595`、rebase と merge)、`TestWatch_ReservedScope_ARebasedOwnCommitOutsideRaisesScopeChange`(`:635`、D6 の「積み直された自分の commit はもう一度知らせる」) |
| AC14 分岐点にあった記録の変更と、分岐点が分からないときは立つ | Met | `dropOwnRecords` は ref が無いか merge-base が読めなければ何も除かない(`watch_scope.go:342-349`)。`TestWatch_ReservedScope_RecordsTheBaseHadAndAnUnknownBaseRaiseScopeChange`(`:529`、8 つの場合) |
| AC12 古い watch-status と `recurred` の無い受信箱を読む | Met | 新しい欄はどれも `omitempty`(`watch.go:255`、`:291-293`)。欄の名前を変えたので、前のビルドの per-entry の基準とは比べない(`:272-284` の doc)。テスト `TestWatch_StatusFile_WrittenBeforeRecurrencesLoadsAndItsPendingAlertTakesARecurrence`(`:3545`)、`TestWatch_ReservedScope_AStatusFileWithoutTheScopeBaselineLoads`(`watch_scope_test.go:884`)、`TestInboxRead_LinesWithoutWokenFoldAsBefore`(`inbox_test.go:859`、recurrences 0) |
| AC13 help、agent-messaging(2 つの写し)、仕様の注記、RC3 の行 | Met(名指しの文書) | `internal/cli/org_watch.go:83-92`(`TestOrgWatch_HelpDescribesAlertRecurrencesLeaderStallAndReservedScope` が文を固定)。`.claude/rules/ralph/agent-messaging.md` と templates/base の写しは `cmp` で同じ。仕様の FR-3(82 行)と FR-6(116 行)に 2026-11-12 の注。tech-debt の RC3 は RESOLVED の注つきで閉じ、既知の穴の行を足した。名指しの外の文書のずれは下の V1〜V5 |

計画の決定 D1〜D6 との照合: D1・D2 は上の AC1〜AC5、D3・D4 は AC6・AC7、D5 は AC8・AC11・AC14、D6(改めた形)は AC9・AC15 で確かめた。L1(commit の一覧に失敗したら前の HEAD を据え置き、git がその HEAD を持たなければ進める)は `watch_scope.go:414-418`、`:446-448`。L3 は `dropOwnRecords` が候補のないとき git を打たない形(`:339-341`)で、計画の Risks の 120 行と Implementation outline の 4 の「HEAD が変わった周期だけ」とは違うが、Progress に受け入れとして記録されている。

## Static analysis

| Command | Result | Notes |
| --- | --- | --- |
| `./scripts/run-static-verify.sh`(changed-language scope、golang) | PASS(exit 0) | gofmt ok、`go vet ./...` は出力なし、golangci-lint `0 issues.`、staticcheck は出力なし |
| 同じ実行の中の local verifier | PASS | shellcheck と `sh -n`(hooks の 2 つの写し)、guard の awk、`settings.json` の jq、Codex の hook 検査 3 つ |
| `scripts/check-sync.sh` | PASS | DRIFTED 0、ROOT_ONLY 0 |
| `scripts/check-pipeline-sync.sh`、`check-skill-sync.sh`、`check-template-purity.sh` | PASS | skill は 14 本とも同じ |
| tech-debt README の plan 参照の検査 | PASS | |
| branch の secret scan | PASS | `028a8f6d..f45133f0` を origin/main に対して clean |
| `./scripts/plan-visual.sh digest` | `e5a18ef9477a` | `- Approved:` の行と一致 |
| 変更したファイルの U+2028・U+2029・U+FFFD | 1 件 | `docs/reports/self-review-2026-11-12-watch-alert-dedupe.md:33` の U+2028(V7)。Go のファイルには無い(N1 の直しは効いている) |

## Documentation drift

| Doc / contract | In sync? | Notes |
| --- | --- | --- |
| `ralph org watch --help` | Yes(INFO I1) | 3 つの文がテストで固定されている |
| `ralph org inbox show --help` | Yes | event の一覧に `recurred` |
| `ralph org inbox --help` | Yes | ALERT の件を「escalation の件」と書くだけで、間違った文はない |
| `.claude/rules/ralph/agent-messaging.md` と templates/base の写し | Yes | 同じ中身。stall と scope の規則はもともとこの文書の範囲の外 |
| 仕様 `2026-10-07-org-multi-org-director.md` の FR-3・FR-6 の注 | Yes(INFO I2) | |
| `docs/tech-debt/README.md` | 一部 | V6 |
| `/org` skill(4 つの写し) | 一部 | V1 |
| README.md(131 行、278 行) | 一部 | V2 |
| `docs/quality/quality-gates.md:83` と templates/base の `:82` | 一部 | V3 |
| `docs/recipes/worktrees.md:61-64` と templates/base の写し | 一部 | V4(実装者の挙げた候補に無い) |
| AGENTS.md の `internal/org/` の行(91 行) | 一部 | V5 |
| `/director` skill の 967 行 | 計画が #307b に回した | INFO I3 |
| `internal/org/reserve.go:26` のコメント、`/org` skill の 460〜462 行 | Yes | INFO I4 |
| 役割 prompt(`internal/org/prompts/*.md`) | Yes | ALERT の扱いの文(leader.md 517・1038 行、director.md 53 行)は変わった規則に触れず、そのまま正しい |
| 計画の AC の箱と Progress checklist | 遅れ | V8 |
| self-review の報告 | 一部 | V7 |

### 所見

- V1(LOW): `/org` skill(`.claude/skills/org/SKILL.md` と `.agents/skills/`、`templates/base/` の 2 つ、計 4 つの写し)に今回の挙動が入っていない。どれも今の文は間違いではないが、足りない。
  - (a) 367 行の `watch` の行。パルス層を「stall/生存/スコープ変更の ALERT を leader へ送る」とだけ書き、続きのあいだの再発が件の `recurred` になって leader に送られないこと、待っている leader の stall を立てないこと、予約のある org の scope_change が予約の外だけで立つことを書かない。
  - (b) 370 行の `inbox` の行。`--json` の件の欄として `delivery`・`woken`・`woken_at`・`alert_id` を挙げるが、`recurrences` と `last_recurred_at` を挙げない。`show` の履歴の `recurred` も書かない。
  - (c) 608〜657 行の「見張りの ALERT の件」。「受信箱に TYPE `ALERT` の件を 1 つ記録する」と「同じ alert_id の件がもう受信箱にあれば新しい件は書かない」だけで、D2(同じ org・座席・条件の片付いていない件があれば `recurred` を足し、人には送らず、件の期限はそのまま)を書かない。
  - 計画の Rollout は「役割 prompt と skill は変えないので、下流の `ralph upgrade` は要らない」と書く。/sync-docs が skill を直すなら、挙動ではなく文書の更新に `ralph upgrade` が要ることを Rollout の文に足すか、直さない判断を記録する。
- V2(LOW): README.md の 131 行(コマンドの表の `watch --all`)と 278 行(見張りの段落)は、ALERT の件を escalation ごとの件として書き、続きの再発、leader の stall、予約の外だけの scope-change を書かない。
- V3(LOW): `docs/quality/quality-gates.md:83` と `templates/base/docs/quality/quality-gates.md:82` の Watchdog pulse layer の行も、escalation を「`ALERT` item of the inbox」とだけ書く。再発が既存の件の `recurred` になることを足す。
- V4(LOW、実装者の挙げた候補の外): `docs/recipes/worktrees.md:61-64` と templates/base の写しは「The pulse-layer watchdog compares a seat's live `git status --porcelain` against its recorded worktree to detect out-of-scope changes」と書く。予約のある org では、`git status --porcelain=v1 -z --untracked-files=all` と前の周期からの commit のうち予約の外のパスだけを見る(`watch_scope.go:398-449`)。「recorded worktree と比べる」も前からの言い方で、比べる相手は前の周期の記録である。
- V5(LOW): AGENTS.md の `internal/org/` の行は、見張りの escalation を「are `ALERT` items of the same inbox」と書き、同じ鍵の片付いていない件があれば `recurred` になることと、新しいファイル `watch_recurrence.go`(続きの扱い)と `watch_scope.go`(予約のある org の scope_change)を挙げない。map なので、ファイルの名前を足せば grep で辿れる。
- V6(LOW): tech-debt の register。Trigger の欄は python で 4 つ目のセルを切り出し、この diff が変えた関数名とファイル名で照合した(行の grep ではない)。
  - (a) 今回足した 462 行(「Four gaps that fix/watch-alert-dedupe leaves」)の Trigger は「the `scope_change`, `stall`, or recurrence code of `internal/org/watch.go` or `internal/org/watch_scope.go`」と書くが、続きの扱いのコードは L4 で `internal/org/watch_recurrence.go` に移った(`recordRecurrence`、`escalateAsRecurrence`、`openAlertItemForKey`、`pendingAlertForKey`)。Trigger と最後のセルにこのファイルを足す。
  - (b) N5(`evaluateReservedScope` の doc は、予約の外の commit がいつも回数になるように読めるが、ALERT を立てた直後の周期の commit は `raiseOrClear` の Active で抑えられる)はまだ入っていない。計画どおり /sync-docs が足すと「Four gaps」の数も変わる。
  - (c) この diff が Trigger に当たり、払わず、引き継ぎの注も無い行。357 行(`watch.go` と `watch_test.go` の大きさ。Trigger「The next change to the watch's escalation or to its tests」。`escalateToInbox` を変え、`watch_test.go` は 3051 行から 3975 行、`watch.go` は 1865 行から 2046 行になった。escalation のコードは `watch_escalate.go` に移していない)。303 行(Trigger「The next change to `watch_test.go`」。`newTestWatchRun` は `watch_test.go:378` で今も `os.MkdirTemp`)。211 行の (a)(Trigger「the next edit of ... `internal/org/inbox_test.go`」。`TestPrintableInboxText` は今も `escalate_test.go:1188`)。353 行(Trigger「The next change to `sendAlert`」。S2(`e9c49d0c`)は `sendAlert` の alert_id を作る 1 行を `watchAlertID` に替えた。境界のものだが、続きの扱いで leader への ALERT の数そのものが減ることも、この行の Impact に関わる)。それぞれ払うか、引き継ぎの注を足す。
  - (d) コメントだけで当たった境界のもの: 355 行(Trigger「the next change to `InboxDeadlineDue`」。doc に Recurrences を読まない 1 文を足した)と 258 行(「the next change to the inbox deadlines (`deadline.go`)」、同じ doc)。179 行には、コメントだけの変更を当たったとみなして注を足した前例がある。どちらにするかは /sync-docs が決める。
  - (e) 210 行(止める頼みが周期の終わりまで待つ)は「Each herdr, agmsg, and `git status` call of the pulse gets its own `DriverCallTimeout`」と書く。予約のある org の座席は、1 周期に git を最大 9 回打つ(status、HEAD の rev-parse、symbolic-ref と origin/main・origin/master の rev-parse、log か diff、前の HEAD の rev-parse、merge-base、ls-tree。`watch_scope.go:403-420`、`:250-262`、`:305-316`)。どれも 10 秒で切られるので、git が止まると 1 座席で待ちが延びうる。Trigger(打ち切り方の変更)には当たらないが、Item の呼び出しの数え方に足す。
- V7(LOW、記録): `docs/reports/self-review-2026-11-12-watch-alert-dedupe.md:33` の N1 の行は、`watch_scope.go:138` の中身を引くところに U+2028 の文字そのものが入り、表示では行が切れる。コード側(`watch_scope.go:138`)は 6 文字の escape で正しい。報告の行を 6 文字の escape に直す。
- V8(LOW): 計画の AC の箱は 15 個とも `[ ]` のまま、Progress checklist にこの verify の行が無い。/sync-docs に回すもの(V1〜V8 と N5)を ID で並べる。

### INFO

- I1: help の「while another seat of the org is working」は、`agent get` の返事が前の周期から変わったことも条件にする規則(`seatBusy`)の要約である。仕様の FR-6 の注と tech-debt の RC3 の注は条件を全部書いている。
- I2: 仕様の FR-3 の注の「予約の外のパスが変わったときだけ立ち」は、L2 のあとの規則(前の周期に無かった外のパスか、外を触る commit。外のパスが減ったときと stage しただけでは立たない)の要約である。`evaluateReservedScope` の doc と計画の Progress の S6 が細部を書く。
- I3: `/director` skill の 967 行の ALERT の件の項は、件に回数が付くことを書かない。計画の Non-goals が director skill の変更を #307b に回している。#307b で直す。
- I4: 実装者が挙げた `internal/org/reserve.go:26`(「the watchdog's scope_change alert reports such writes」)と `/org` skill の 460〜462 行(「範囲の外への変更は scope_change ALERT で知らせる」)は、今の挙動と合う。今回の変更でむしろ文どおりになった。自分の新しい計画と報告を除くことを足すかは任意。
- I5: 計画の Affected areas は `watch_all_test.go` と `reserve.go` を挙げるが、どちらも変わっていない。`reserve.go` は「置くなら」の条件つきで、`RunWatchAll` を通す検査は `watch_scope_test.go:1110` にある。Test plan の「watch_all_test.go の流れで件数を照らす」が満たされたかは /test が見る。

## Observational checks

- `watch.go`、`watch_recurrence.go`、`watch_scope.go`、`inbox.go`、`org_inbox.go` の diff を読み、AC ごとに道とテストの名前と assert を突き合わせた(上の表)。
- doc コメントの主張を 1 つ確かめた。`orgHasOpenLeaderItem` と `recordRecurrence` の「受信箱が読めないことは周期の初めの期限の掃除が stderr に書く」は、`watchAllRun.cycle`(`watch.go:747-750`)が `SweepInboxDeadlines` の失敗を `watchdog: inbox deadlines skipped this cycle:` で書くので正しい。
- tech-debt の RESOLVED の注と新しい行が名指す関数(`leaderIsWaiting`、`seatBusy`、`raiseOrClear`、`escalateToInbox` は `watch.go`、`evaluateReservedScope`、`scopeCommittedPaths` は `watch_scope.go`)の場所を grep で確かめた。合っている(V6 (a) のファイルの漏れを除く)。
- self-review の直しの主張を確かめた: M1(`LastRecurredTS` はコードに無い。読み飛ばすテストは `TestWatch_StatusFile_APendingAlertWithLastRecurredTSLoadsAndDropsIt`)、L1、L2、L4、L5(`ScopeOutsidePathsSeen` は bool)、N1、N4(`1 time`)。
- origin/main は `028a8f6d` から動いていない(`git ls-remote origin refs/heads/main`)。

## Coverage gaps

- Go のテストは走らせていない(/test)。表のテストの欄は、テストがあり assert が AC を言っていることまで。Verify plan の「再発を模したテストの件数(修正前は立つたびに 1 件、修正後は 1 件と recurred の数)」は、修正後の数(5 回立って件 1、recurred 4、ALERT 1 通)を `watch_test.go:3126` が assert することまでを読んだ。修正前の数(leader の stall は立つたびに件 1 つ)は旧コードを読んで推した数で、実行で数えていない。
- 実機の走行は無い。機能の org を新しい binary で走らせ、受信箱の件を数え直すことは RC3 の注のとおり残っている。4 時間で 95 件の再発の仕組み(herdr の文字列の揺れ)は推測のまま(計画の Assumptions)。
- herdr の版による `agent_status` の読み方の違いは、`herdrAgentStatus` を共有することで扱う形を読んだだけ。
- 決定的な検査が無いもの: skill の `--json` の欄の一覧が `org.InboxItem` の JSON の欄に追いつかないこと(V1 (b))。最小の検査は、`InboxItem` の `json:"..."` の名前のうち `id`・`state` などの基本の欄を除いたものが、`.claude/skills/org/SKILL.md` の `inbox` の行にすべて出ることを見る Go のテスト 1 本である。

## Verdict

pass

- Verified: AC1〜AC15 は、コードの道とそれを assert するテストがあることを確かめた(上の表)。計画の決定 D1〜D6 と、Progress に記録されたずれ(D6 の rebase、AC15、L1、L2、L3、N5)は実装と合う。静的解析はすべて通った(gofmt、go vet、golangci-lint、staticcheck、shellcheck、同期の検査 4 つ、template purity、tech-debt の plan 参照、secret scan)。計画の digest は承認の行と同じ。base との衝突はない
- Partially verified: AC13 は名指しの文書(help、agent-messaging の 2 つの写し、仕様の注、RC3)で満たす。それ以外の文書のずれは V1〜V5
- Not verified: Go のテストの結果と `run-test.sh`(/test)、実機での件の数え直し
- 所見: LOW 8 件(V1〜V8)、INFO 5 件(I1〜I5)。どれも merge を止めない。V4 と V6 (a)・(c)・(e)、V7 はこの verify で新しく見つけた。V1〜V3 と V5 は実装者が挙げた候補を確かめて広げたもの
- /sync-docs に回すもの: V1(`/org` skill の 4 つの写しの 3 か所と、計画の Rollout の文)、V2(README の 2 か所)、V3(quality-gates の 2 つの写し)、V4(worktrees の recipe の 2 つの写し)、V5(AGENTS.md)、V6(register の 462・357・303・211・353 行、355・258 行の判断、210 行の Item)、V7(self-review の報告の 33 行)、V8(計画の Progress)、それに計画どおりの N5
