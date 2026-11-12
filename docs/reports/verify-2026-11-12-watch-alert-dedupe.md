# Verify report: 見張りの ALERT を 1 件にまとめ、leader の stall と予約の中の scope_change を上げない(#307 の前半)

- Date: 2026-11-12
- Plan: `docs/plans/active/2026-11-12-watch-alert-dedupe.md`(digest `e5a18ef9477a`、`./scripts/plan-visual.sh digest` の値と `- Approved:` の行が一致)
- Verifier: verifier subagent(Claude Code)、pipeline cycle 2(`cycle-count.json` は cycle 2)。cycle 1 の報告(`b8058948`)をこの報告で上書きした。cycle 1 の判定と所見は末尾の「Cycle 1 の記録」に残し、tech-debt の行が指す V6 もそこにある
- Scope: branch 全体は `git diff 028a8f6d...HEAD`(HEAD `3bb1828b`、34 ファイル、+4956/-166)。cycle 2 で新しく読んだのは、コードの 3 commit(S8 `907afde3` は cross-review の 4 件、S9 `5393a316` は cycle 2 の self-review の C2-1〜C2-4、`3bb1828b` はテストのコメントだけで C2b-1)と、記録の commit(`18bb7424`、`d3a63398`、`f853d973`、`7ac1e5c1`、`d3a34921`)。`internal/` の差分は `2bebc1ac..3bb1828b` で 5 ファイル、+507/-74。cycle 1 の verify のあとの /test(`3506f99a`)と /sync-docs(`45e84b3e`、`984761f8`、`0c727338`、`2bebc1ac`)はそれぞれの報告が扱うので、ここでは文書の照合にだけ使った。origin/main は `028a8f6d` のまま(`git ls-remote`)なので、base との衝突はない
- Evidence: `docs/evidence/verify-2026-11-12-watch-alert-dedupe.log`(git が無視する。cycle 1 の内容のあとに cycle 2 の静的解析と probe を足した。足す前の内容がバイトで先頭に残ることを `cmp` で確かめた)。スクリプト自身の写しは `docs/evidence/verify-2026-11-12-003232.log`

## Spec compliance

計画は Progress checklist に cycle 1 のずれ(D6 の rebase と AC15、L1〜L3、N5)を記録し、S8 の行(160 行)に D6 の細部も記録している。D6 の本文の `git log` には `--cc` と origin/HEAD の確かめが書かれていないが、本文は変えず Progress に書く、という扱いである。下の判定は、これらで改めた計画に照らした。テストの欄は、テストがあり assert が AC を言っていることまでを読んで確かめた。テストは走らせていない(/test の役目)。

S8 が触れた AC1〜AC5、AC9、AC15 は、新しい道とテストを読み直した。それ以外の AC は、cycle 2 の hunk がその道を変えていないことを `git diff -U0 2bebc1ac HEAD` で確かめ、行番号を今の HEAD に合わせた。

| Acceptance criterion | Status | Evidence |
| --- | --- | --- |
| AC1 open か acked のあいだは件が 1 つで、立つたびに `recurred` が 1 つ増え、show と `--json` に回数と最後の時刻が出る | Met | 受信箱の側は cycle 2 で変わっていない(`internal/org/inbox.go:45`、`:398-420` の `AppendRecurred`、`:670` の畳み込み、`internal/cli/org_inbox.go:482-502`)。見張りの側は `internal/org/watch.go:1307-1322`(`raiseOrClear` が立ち上がりで `recordRecurrence` を先に呼ぶ)と `internal/org/watch_recurrence.go:112-140`。S8 で pending への畳み込みは `foldIntoPendingAlert`(`:162-190`)に移った。テストは `TestWatch_Recurrence_AFlappingLeaderStallIsOneAlertItemUntilItIsResolved`(`watch_test.go:3126`、5 回立って件 1・recurred 4)、`TestInboxRead_RecurredCountsFiringsWithoutChangingTheState`、`TestOrgInbox_ShowsTheRecurrences` |
| AC2 resolved のあとは新しい件 | Met | `openAlertItemForKey`(`watch_recurrence.go:75-83`)は resolved の件を飛ばし、`AppendRecurred` が `ErrInboxResolved` を返せば新しい ALERT に倒れる(`:130-136`)。同じテストの後半(`watch_test.go:3197-3208`、e1 を resolve したあとに e2) |
| AC3 続きのあいだは leader に agmsg の ALERT を送らない | Met | `watch.go:1315-1317`: `recordRecurrence` が true なら `sendAlert` と `OnSemanticTrigger` の前に戻る。S8 (1) で、leader が送信のあとに動いた pending は、畳み込む前に消える(`foldIntoPendingAlert` が `leaderActedSinceAlert`(`watch.go:1760-1775`)を呼ぶ。`checkDeadman` も `:1713` で同じ関数を使う)。そのときの再発は新しい ALERT になる。これは D1 の「pending が leader の動きで消えたら続きは終わり」のとおりで、AC3 の「続きのあいだ」には入らない。テスト `TestWatch_Recurrence_AFiringAfterTheLeaderActedOnTheSeatsPendingAlertIsANewAlert`(`:3551`)は 3 つの場合を見る。manifest の `sent` と、agmsg の履歴の leader の行では ALERT が 2 通になり、何もしなければ 1 通で recurrences 1。manifest が読めないときは知らせる側に倒れる(`watch_recurrence.go:172-178`、テスト `:3650`) |
| AC4 deadman の前の再発は pending を増やさず、escalate した件に回数が残る | Met | `watch_recurrence.go:187`(pending の `Recurrences++`)、`watch.go:1887-1895`(件を記録した直後に回数つきの `recurred`)。テスト `TestWatch_Recurrence_ASeatsAlertFiringAgainBeforeTheDeadmanCarriesItsCountToTheItem`(`:3225`)と、`:3551` の「leader did nothing」の場合(30 分後に recurrences 1 の件が 1 つ) |
| AC5 watch の立て直しで記録が無くても、同じ鍵の片付いていない件に `recurred` を足す | Met | 立ち上がりの側は受信箱を読む(`watch_recurrence.go:118-139`)。escalate の側は `watch.go:1880` と `escalateAsRecurrence`(`watch_recurrence.go:202-214`)。S8 (4) で `alertItemWithRecurredID`(`:60-67`)の確かめが鍵の照合の前(`watch.go:1875`)に入った。pending の alert_id が、どれかの件の(状態を問わず)`recurred` の event にもうあれば、何も書かず何も送らない。テストは `:3299`、`:3392`、`:3699`(件が open と resolved の 2 つ)と、`TestWatch_AlertEpisodeLookupsMatchTheKeyExactly`(`:3832`、`recurred` の event だけを見ることを確かめる)。S9 の C2-1 は、立ち上がりの側の同じ確かめを消した。立ち上がりの alert_id は周期ごとの `now` から作るので、本番では当たらなかった。残る窓は V11 (a) |
| AC6 片付いていない QUESTION・BLOCKED・RESULT の件があるあいだ、leader の stall は立たず受信箱にも入らない | Met | cycle 2 で変わっていない。`watch.go:1257`、`:1155-1195`(`leaderIsWaiting`、`orgHasOpenLeaderItem`)。テスト `:3975`、`:4027` |
| AC7 ほかの座席が `working` で文字列が変わるあいだは立たず、変わらなくなれば今までどおり立つ | Met | cycle 2 で変わっていない。`watch.go:1036-1052`(leader を最後に評価)、`seatBusy`(`:1197`)。テスト `:4076`、`:4119`、`:4151` |
| AC8 予約の中だけと、自分の新しい計画と報告だけの変更では立たない | Met | `watch.go:1274-1277`、`watch_scope.go:343-364`(`dropOwnRecords`)、`:124`(`ownRecordCandidate`)。merge でない commit の扱いは `--cc` で変わらない。テスト `watch_scope_test.go:390`、`:428` |
| AC9 予約の外の変更で立ち、本文に外のパスが出る。外から中への移動を 1 周期で commit しても移動元が出る | Met | `watch_scope.go:70-87`(rename の両側)、`:289-303`(`git log --no-show-signature --cc --no-renames ...`)。S8 (2) で merge commit 自身の編集も数えるようになった。テスト `:476`(7 つの場合)と、新しい `TestWatch_ReservedScope_AConflictResolvedOutsideInAMergeRaisesScopeChange`(`:655`)。後者は実際の git で予約の外の衝突を解決して merge し、clean な worktree で、ALERT が `other/x.go` だけを名指すことを `slices.Equal` で確かめる(upstream から来た `other/new.go` は出ない) |
| AC10 予約のない org は今までどおり | Met | cycle 2 で変わっていない。`watch.go:1279-1287`。テスト `watch_scope_test.go:1216` |
| AC11 除く規則が役割 prompt の書かせる `docs/` のパスを覆う | Met | cycle 2 で変わっていない。`TestOwnRecord_CoversTheDocsPathsTheRolePromptsWrite`(`watch_scope_test.go:1294`) |
| AC15 rebase で入った upstream の commit では立たない | Met | `watch_scope.go:292`(`... <prev>..<head> --not <ref> --`)。S8 (3) で `scopeDefaultBranchRef`(`:254-272`)は origin/HEAD の指す先が commit を指すかを確かめ、指さなければ origin/main、origin/master、`git diff` の順に落ちる。前は、指す先の無い origin/HEAD で `git log` が毎回失敗し、L1 の据え置きで前の HEAD が動かなくなっていた(triage の 3)。テストは `:595`(rebase と merge)、`:635`、新しい `:687`(指す先の無い origin/HEAD。origin/main があれば upstream の merge は立たず、自分の commit だけが出る。main も master も無ければ diff に落ちる)、`RunsOnlyTheGitCallsItNeeds`(`:1072`、指す先の確かめを含む git の呼び出しの並び) |
| AC14 分岐点にあった記録の変更と、分岐点が分からないときは立つ | Met | `dropOwnRecords` は ref が無いか merge-base が読めなければ何も除かない(`watch_scope.go:353-359`)。S8 のあとは、指す先の無い origin/HEAD で main も master も無いときも「分からない」になる。テスト `:529`(8 つの場合)と `:687` の 2 つ目の sub-test |
| AC12 古い watch-status と `recurred` の無い受信箱を読む | Met | S8 と S9 は watch-status に欄を足していない(`pendingAlertForKey` の返り値の変更は package の中だけ)。テスト `watch_test.go:3745`、`:3801`、`watch_scope_test.go:950`、`inbox_test.go:859` |
| AC13 help、agent-messaging(2 つの写し)、仕様の注記、RC3 の行 | Partially met | help(`internal/cli/org_watch.go:83-92`)の「still waits for the leader to respond」と、agent-messaging の 2 つの写し(`cmp` で同じ)の「still waits for the leader to respond」は、S8 (1) の規則とも合う。leader が動いた pending は返事を待っていないので、その再発は新しい ALERT になる。一方、仕様の FR-3 の注(82 行)と tech-debt の RC3 の注(375 行)は commit のパスを `git log --no-renames` とだけ書き、S8 の `--cc`(merge commit 自身の編集を数える)が入っていない(V9、V10)。merge でない commit については今も正しい |

計画の決定との照合: D1・D2 は AC1〜AC5、D3・D4 は AC6・AC7、D5 は AC8・AC11・AC14、D6(改めた形)は AC9・AC15 で確かめた。S8 の 4 つの直しは、どれも D1、D2、D6 の意図(続きは leader の動きで終わる、同じ escalation を二重に記録しない、org 自身の commit だけを数える)の中にある。新しい AC は要らない。

self-review の直しの照合:

- C2-1(S9): `recordRecurrence` の alert_id の確かめとそのテストが消え、`alertItemWithRecurredID` の呼び出しは `escalateToInbox` の 1 か所(`watch.go:1875`)だけになった。消えたテストは 1 本(32 行の削除だけ、`git show 5393a316 -- internal/org/watch_test.go`)
- C2-2(S9): `leaderActivityEventCount` と `historyLeaderLineCount` の doc の 3 か所が `leaderActedSinceAlert` を指す。番兵の理由の段落は `leaderActedSinceAlert` の doc に移った
- C2-3(S9 と `3bb1828b`): テストのコメントは 4 か所とも triage の番号を使わない。`grep -n 'finding [0-9]' internal/org/watch*.go` が出すのは、別の PR の指摘を指す `watch.go:329`、`:997`、`watcher_test.go:78`、`:252` だけ
- C2-4(S9): `foldIntoPendingAlert(status, orgID, seatID, condType)` は `recordRecurrence`(`watch_recurrence.go:115`)の 1 か所からだけ呼ばれる
- C2b-1(`3bb1828b`): `watch_test.go:3691-3693` のコメントが「found by the cycle 1 cross-review of plan 2026-11-12-watch-alert-dedupe, the escalation replay」になった。差分は 3 追加、2 削除で、コメントだけ

## Static analysis

| Command | Result | Notes |
| --- | --- | --- |
| `./scripts/run-static-verify.sh`(changed-language scope、golang) | PASS(exit 0) | gofmt ok、`go vet ./...` は出力なし、golangci-lint `0 issues.`、staticcheck は出力なし(go 1.27.3) |
| 同じ実行の中の local verifier | PASS | shellcheck と `sh -n`(hooks の 2 つの写し)、guard の awk、`settings.json` の jq、Codex の hook 検査 3 つ |
| `scripts/check-sync.sh` | PASS | DRIFTED 0、ROOT_ONLY 0 |
| `scripts/check-pipeline-sync.sh`、`check-skill-sync.sh`、`check-template-purity.sh` | PASS | skill は 14 本とも同じ |
| tech-debt README の plan 参照の検査 | PASS | |
| branch の secret scan | PASS | `028a8f6d..3bb1828b` を origin/main に対して clean |
| `./scripts/plan-visual.sh digest` | `e5a18ef9477a` | `- Approved:` の行と一致 |
| `git diff --check 2bebc1ac HEAD -- ':!docs/evidence'` | rc 0 | |
| `2bebc1ac` のあとに変わった 10 ファイルの U+2028・U+2029・U+FFFD | 0 件 | バイト列で数えた |

## Documentation drift

| Doc / contract | In sync? | Notes |
| --- | --- | --- |
| `ralph org watch --help` | Yes | S8 (1) の規則と合う |
| `.claude/rules/ralph/agent-messaging.md` と templates/base の写し | Yes | 同じ中身。「still waits for the leader to respond」は S8 (1) と合う |
| `/org` skill(4 つの写し) | Yes(INFO I6) | 367 行、370 行、617〜650 行の 3 か所とも S8 のあとも正しい |
| README.md(131 行、278 行)、AGENTS.md の `internal/org/` の行、`docs/quality/quality-gates.md` の 2 つの写し | Yes | merge commit にも origin/HEAD にも触れない要約で、S8 で間違いにはならない |
| `docs/recipes/worktrees.md:61-66` と templates/base の写し | Yes(INFO I7) | |
| 仕様 `2026-10-07-org-multi-org-director.md` の FR-3 の注(82 行) | 一部 | V9 |
| 仕様の FR-6 の注(116 行) | Yes | 「leader の動きを待っている(escalate していない)あいだ」は S8 (1) と合う |
| `docs/tech-debt/README.md` | 一部 | V10〜V13 |
| 計画の Progress checklist | 遅れ | V14 |
| 計画の D6 の本文と Scope の 21 行 | 記録されたずれ | INFO I8 |
| self-review の報告(`d3a34921`) | 記録の時点のずれ | INFO I9 |

### 所見(cycle 2)

ID は cycle 1 の V1〜V8、I1〜I5 の続きにした。どれも /sync-docs に回す。コードの直しは含まない。

- V9(LOW): 仕様 `docs/specs/2026-10-07-org-multi-org-director.md:82` の FR-3 の注は、commit のパスを「前の周期から増えた commit のうち origin の既定のブランチに無いもの(`git log --no-renames`)」と書く。S8 (2) のあと、コマンドは `--cc` を含み、merge commit からは merge 自身の編集(予約の外の衝突の解決など)だけを数え、merge が upstream から運ぶ変更は数えない。この 1 句が無い。もう 1 点、「origin の既定のブランチが分からなければ `git diff`」は今も正しいが、origin/HEAD の指す先が無いとき(既定のブランチの名前が upstream で変わり、古い remote-tracking branch が消えた)は origin/main、origin/master を試してから `git diff` に落ちる(`scopeDefaultBranchRef`)。仕様に足すかは任意で、足すなら「origin の既定のブランチ」の決め方に 1 句でよい
- V10(LOW): tech-debt の RC3 の RESOLVED の注(`docs/tech-debt/README.md:375`)も同じく「`git log --no-renames`, so the upstream commits a rebase brings in do not count」とだけ書く。merge commit 自身の編集を数えること(`--cc`)と、upstream の変更は merge で入っても数えないことを足す。origin/HEAD の指す先の確かめは V9 と同じく任意
- V11(LOW): tech-debt の既知の穴の行(462 行、「Six gaps that fix/watch-alert-dedupe leaves」)。Trigger の欄(4 つ目のセル)を python で切り出して読んだ
  - (a) C2-1 の残る窓が入っていない。`recordRecurrence` の受信箱の側(`watch_recurrence.go:118-139`)は、`now` から作る新しい alert_id で `recurred` を足す。足したあと watch-status を保存する前に watch が止まると、立て直した watch には鍵の Active の記録が無いので、まだ立っている条件を立ち上がりと読み、同じ件に `recurred` をもう 1 つ足す。件の回数が 1 多くなり、状態と期限は変わらない。計画の Progress の S9(163 行)と self-review の Tech debt identified の (7) が、tech-debt に足すと書いている。escalate の打ち直しの側は S8 (4) で閉じた(AC5 の行)
  - (b) M48 の Trigger が、この PR の中で当たった。(6) の Trigger は「the next change to ... `scopeDefaultBranchRef`: add a test each (... an origin/HEAD that points at a ref outside `refs/remotes/origin/` is not taken as the default branch)」と書く。S8(`907afde3`)はこの関数に `namesCommit` を足したが、そのテストは足していない。新しいテスト(`watch_scope_test.go:687`、`:1072`)の origin/HEAD は、どれも `refs/remotes/origin/` の下(`gone`、`trunk`)を指すので、`HasPrefix` の確かめを外しても落ちない形はおそらく変わらない(変異は走らせていない。未確認)。(6) に引き継ぎの注を 1 文足す
  - (c) (5) の Trigger「the next edit of the doc of `evaluateReservedScope`」も当たった。S8 は doc の 1 つ目の項(`watch_scope.go:374-381`、`--cc` と merge commit)を書き換えたが、N5 の言い過ぎ(「While the item of the earlier ALERT is open, that is a recurrence on that item」、`:400-402`)は残っている。(5) に引き継ぎの注を足す
  - (d) 行全体の Trigger「The next change to the `scope_change`, `stall`, or recurrence code of `internal/org/watch.go`, `internal/org/watch_recurrence.go`, or `internal/org/watch_scope.go`」は、S8 と S9 で当たった。(1)〜(3) は払われていない。とくに (2) の窓(件を記録してから回数の `recurred` を足すまで)は、S8 が `escalateToInbox` を変えたあとも同じである(`watch.go:1887-1895`)。この PR の中で当たり、引き継いだことを 1 文で書く
  - (e) 行の見出しの数。(7) に C2-1 の窓が入れば「Seven gaps」になる。self-review(`d3a34921`)が案に出した (8)(コメントの取り残し)は、その中の C2b-1 を `3bb1828b` が直したので、入れるとしても残りの折り返し(`watch_recurrence.go:148` は 91 桁など)だけになる。入れるかは /sync-docs が決める
- V12(LOW): tech-debt の 210 行(止める頼みが周期の終わりまで待つ)の Debt の末尾は、予約のある org の座席の git を「the lookups of origin's default branch, at most three ... at most nine in all ... up to about 90 s per such seat」と数える。S8 (3) のあと、`scopeDefaultBranchRef` は symbolic-ref、指す先の rev-parse、origin/main、origin/master の最大 4 回になった(`watch_scope.go:254-272`)。git は最大 10 回、git が止まれば 1 座席でおよそ 100 秒になる。また S8 (1) の `foldIntoPendingAlert` は、pending に畳み込む再発 1 つごとに manifest を読み、leader の probe(herdr の `agent get`、agmsg の履歴)を最大 2 回呼ぶ。どれも `callWithTimeout` で 10 秒に切られる(`watch.go:1569-1575`、`:1613-1619`)。同じ周期の `checkDeadman` もこれを呼ぶので、止める待ちの呼び出しの数え方に足す。Trigger(打ち切り方の変更)には当たらない
- V13(LOW): Trigger に当たった、ほかの行
  - 357 行(`watch.go` と `watch_test.go` の大きさ。Trigger「The next change to the watch's escalation or to its tests」)。cycle 2 で S8 が `escalateToInbox`(`alertItemWithRecurredID` の確かめ)と `checkDeadman`(判定を `leaderActedSinceAlert` に移した)を変え、`watch_test.go` に 3 本のテストを足した(S9 で 1 本消えた)。引き継ぎの注の数は `3506f99a` で数えたものと書いてあり、その時点については正しい。HEAD `3bb1828b` では `watch.go` 2081 行、`watch_test.go` 4188 行、`watch_recurrence.go` 214 行、`watch_scope.go` 461 行(`wc -l`)。注の「(147 lines) and (449)」を含め、数を今の値にするか、cycle 2 でも当たったことを足す
  - 457 行の (f)(org-seat-resume の変異。Trigger「The next change to ... `leaderActivityEventCount` ...」)。S9 は `leaderActivityEventCount` の doc の 2 か所を書き換えた(コメントだけ)。S8 で、この関数は `foldIntoPendingAlert` から `leaderActedSinceAlert` を通しても呼ばれるようになった。もし `resumed` を leader の動きと数えれば、deadman の取り消しだけでなく、続きの終わり(再発が新しい ALERT になる)も起きる。(f) の Impact が少し広がった。コメントだけの変更を当たったとみなす前例(179 行、cycle 1 の 355 行と 258 行)があるので、注を足すかは /sync-docs が決める
- V14(LOW): 計画の Progress checklist は S9(163 行)で終わっていて、cycle 2 の self-review の 2 回目(`d3a34921`: Merge: yes、LOW 1 件 C2b-1)と、それを直した `3bb1828b`(テストのコメントだけ)が無い。計画を `3bb1828b` と `d3a34921` で grep すると 0 件だった。この verify の行も無い

### INFO(cycle 2)

- I6: `/org` skill(4 つの写し)は S8 のあとも正しい。367 行の `watch` の行と 618 行の「leader の返事待ち(まだ escalate していない)」は、leader が動いた pending を含まない。637 行の「予約の外を触る commit のときに立ち」は、予約の外の衝突を解決した merge commit も含むと読める。「origin の既定のブランチにある commit(rebase や merge で入ったもの)も数えない」も正しい。661 行の「同じ alert_id の件がもう受信箱にあれば … 新しい件は書かない」は、件自身の alert_id の場合だけを書く。S8 (4) で、alert_id が件の `recurred` の event にあるときも何も書かず送らない(同じ ID で人に送ることもしない)。この 1 文を足すかは任意
- I7: `docs/recipes/worktrees.md:61-66`(と templates/base の写し)の「changed in the worktree and in commits since the previous cycle」は、rebase や merge で入る upstream の commit を数えないことを書かない要約である。間違いではない
- I8: 計画の D6 の本文(64 行)、Scope(21 行)、Implementation outline の 4(89 行)は `--no-renames` だけを書く。Progress の S8 の行(160 行)が「本文は変えずにここに記録する」と書いているので、記録されたずれとして扱う。計画の本文は digest に入るので、変えない
- I9: self-review の報告(`d3a34921`)は C2b-1 を残りとし、tech-debt に (8) を足す案を出す。そのあとの `3bb1828b` が C2b-1 を直した。報告は書いた時点では正しい。/sync-docs は (8) に C2b-1 を入れない(V11 (e))
- I10: tech-debt の 303 行(`newTestWatchRun` の `os.MkdirTemp`)は、cycle 2 の `watch_test.go` の変更でもう一度当たった。呼び出しは 29 のままで(`newTestWatchRun(` を含む行は定義を入れて `2bebc1ac` でも HEAD でも 30)、注の「29 callers」は今も正しい。行数の 3975 は `3506f99a` で数えたと書いてある。直すものはない

## Observational checks

- S8、S9、`3bb1828b` の diff を全部読み、triage の 4 件(`docs/reports/cross-review-triage-watch-alert-dedupe.md`)と self-review の C2-1〜C2-4、C2b-1 のそれぞれについて、直しの道とテストの名前と assert を突き合わせた(上の表と「self-review の直しの照合」)
- doc コメントの主張を 3 つ確かめた。(1) `foldIntoPendingAlert` の「A pending alert about the leader takes the recurrence with no test, as the deadman sweep escalates it in the cycle that sent it」は、`checkDeadman` が leader の pending を同じ周期で escalate する(`watch.go:1715`、`:1727`)ので合う。(2) `leaderActivityEventCount` の doc の「this function's two production call sites are exactly a baseline/recount pair」は、本番の呼び出しが `sendAlert`(`:1416`)と `leaderActedSinceAlert`(`:1761`)の 2 つなので合う。(3) `scopeCommittedPaths` の doc の `--cc` の説明(merge が upstream から運ぶ変更は出ない)は、`git log` の `--cc` が、どの親とも違うパスだけを並べることによる。merge のテスト(`watch_scope_test.go:655`)がこれを実際の git で確かめる形になっている(実行は /test)
- S8 (1) の境界を 1 つ読んだ。`foldIntoPendingAlert` が pending を消して false を返したあとも、`recordRecurrence` は受信箱を読む。同じ鍵の片付いていない件がそこにあれば、その件の `recurred` になる。D1 では、片付いていない件があるあいだは続きなので、決定どおりである
- tech-debt の Trigger の欄を python で切り出し、cycle 2 の hunk の関数名(`git diff -U0 2bebc1ac HEAD -- '*.go'` の `@@` の見出しと `func` の行)とファイル名で照合した。候補は 12 行で、Trigger を読んで V11〜V13 と I10 を残した。179 行、238 行、262 行、338 行、356 行は、名前が引き継ぎの注か別の条件の中にあるだけで、当たっていない。118 行と 211 行は「pulse」という一般の語で拾っただけである。353 行(`sendAlert`)は、hunk の見出しが `sendAlert` でも、変わったのはその後ろの `leaderActivityEventCount` の doc なので当たっていない
- origin/main は `028a8f6d` から動いていない(`git ls-remote origin refs/heads/main`)

## Coverage gaps

- Go のテストは走らせていない(/test)。表のテストの欄は、テストがあり assert が AC を言っていることまでである。とくに S8 の 4 本の新しいテスト(`watch_test.go:3551`、`:3650`、`:3699`、`watch_scope_test.go:655`、`:687`)と、`RunsOnlyTheGitCallsItNeeds` に足した 2 つの周期は、読んだだけで、実行の結果は /test が見る
- `git log --cc --name-only` が merge commit について、どの親とも違うパスだけを出すことは、git の文書とテストの形から読んだ。手元と CI の git の版での振る舞いは、実際の git を使うテストを /test が走らせて確かめる
- M48(origin/HEAD が `refs/remotes/origin/` の外を指す場合)は、S8 の新しいテストでも固定されていないと読んだ。変異は走らせていない(V11 (b))
- 実機の走行は無い。機能の org を新しい binary で走らせ、受信箱の件を数え直すことは RC3 の注のとおり残っている
- cycle 1 の「決定的な検査が無いもの」(skill の `--json` の欄の一覧が `org.InboxItem` の JSON の欄に追いつかないこと)は今も無い。最小の検査は、`InboxItem` の `json:"..."` の名前のうち基本の欄を除いたものが `.claude/skills/org/SKILL.md` の `inbox` の行にすべて出ることを見る Go のテスト 1 本である

## Verdict

pass

- Verified: AC1〜AC12、AC14、AC15 は、コードの道とそれを assert するテストがあることを確かめた。S8 の 4 つの直し(leader が動いたあとの再発は新しい ALERT、merge commit 自身の編集を `--cc` で数える、指す先の無い origin/HEAD を飛ばす、`recurred` の event に載った alert_id の escalation の打ち直しで何も足さない)は triage の 4 件と合い、それぞれにテストがある。S9 と `3bb1828b` は C2-1〜C2-4 と C2b-1 を直した。静的解析はすべて通った(gofmt、go vet、golangci-lint、staticcheck、shellcheck、同期の検査 4 つ、template purity、tech-debt の plan 参照、secret scan)。計画の digest は承認の行と同じ。base との衝突はない
- Partially verified: AC13。help と agent-messaging の 2 つの写しは S8 のあとも合う。仕様の FR-3 の注と RC3 の注は merge commit の扱い(`--cc`)が入っていない(V9、V10)
- Not verified: Go のテストの結果と `run-test.sh`(/test)、M48 の変異、実機での件の数え直し
- 所見(cycle 2): LOW 6 件(V9〜V14)、INFO 5 件(I6〜I10)。どれも文書と記録のずれで、merge を止めない。lead が挙げた既知のずれ(仕様の 82 行、RC3 の 375 行、既知の穴の行の C2-1 の窓と M48 の注)は V9〜V11 で確かめた。V11 (c)・(d)、V12、V13、V14 はこの verify で新しく見つけた
- /sync-docs に回すもの: V9(仕様の 82 行)、V10(tech-debt の 375 行)、V11(462 行の (7)、(6) と (5) の注、行全体の注、見出しの数)、V12(210 行の数)、V13(357 行の数と cycle 2 の当たり、457 行 (f) の判断)、V14(計画の Progress に `d3a34921`、`3bb1828b`、この verify)

## Cycle 1 の記録

cycle 1(HEAD `f45133f0`、commit `b8058948`)の判定は pass で、所見は LOW 8 件(V1〜V8)と INFO 5 件(I1〜I5)だった。V1〜V8 は cycle 1 の /sync-docs(`45e84b3e`、`984761f8`、`0c727338`、`2bebc1ac`)がすべて処理した(`docs/reports/sync-docs-2026-11-12-watch-alert-dedupe.md`)。

| ID | 中身(cycle 1) | 処理 |
| --- | --- | --- |
| V1 | `/org` skill の 4 つの写しの `watch` の行、`inbox` の行の `--json` の欄、「見張りの ALERT の件」の節に今回の挙動が無い。計画の Rollout の「skill は変えない」 | `45e84b3e`。Rollout は計画の Progress の注(`2bebc1ac`) |
| V2 | README の 131 行と 278 行 | `45e84b3e` |
| V3 | `docs/quality/quality-gates.md` の 2 つの写しの Watchdog pulse layer の行 | `45e84b3e` |
| V4 | `docs/recipes/worktrees.md` の 2 つの写しの「recorded worktree と比べる」 | `45e84b3e` |
| V5 | AGENTS.md の `internal/org/` の行 | `45e84b3e` |
| V6 | tech-debt の register。(a) 462 行の Trigger に `watch_recurrence.go` が無い。(b) N5 が入っていない。(c) Trigger に当たって注の無い 357・303・211 (a)・353 行。(d) コメントだけで当たった 355・258 行。(e) 210 行の Item に、予約のある org の座席が 1 周期に git を最大 9 回打つことを足す | `984761f8`。(e) の数は S8 のあと 10 回になった(V12) |
| V7 | self-review の報告の U+2028 | `0c727338` |
| V8 | 計画の AC の箱と Progress | `2bebc1ac` |

INFO: I1 は help の `seatBusy` の要約、I2 は仕様の FR-3 の注が L2 のあとの規則の要約であること、I3 は `/director` skill の 967 行が件の回数を書かないこと(計画の Non-goals で #307b、今も残る)、I4 は `internal/org/reserve.go:26` と `/org` skill の 460〜462 行が今の挙動と合うこと、I5 は計画の Affected areas の `watch_all_test.go` と `reserve.go` が変わっていないこと。
