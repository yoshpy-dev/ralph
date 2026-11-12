# Verify report: 見張りの ALERT を 1 件にまとめ、leader の stall と予約の中の scope_change を上げない(#307 の前半)

- Date: 2026-11-12
- Plan: `docs/plans/active/2026-11-12-watch-alert-dedupe.md`(digest `e5a18ef9477a`、`./scripts/plan-visual.sh digest` の値と `- Approved:` の行が一致)
- Verifier: verifier subagent(Claude Code)、パイプラインの run 4 (cycle-count 3)。`cycle-count.json` は `{"cycle":3,"cap":3}` で、cap を 3 に上げたとき cycle を進めなかったため、run の数と cycle の数が 1 ずれる。lead の指示どおり insight event は `--cycle 4` で足した。run 3 の報告(`10a86e28`)をこの報告で上書きした。前の run の所見は末尾の「Run 3 の記録」「Cycle 2 の記録」「Cycle 1 の記録」に ID ごとに残した。tech-debt の行が指す V6、V11、V12、V15、I12 もそこにある
- Scope: branch 全体は `git diff 028a8f6d...HEAD`(HEAD `abf95ceb`、34 ファイル、+5550/-174)。run 4 で新しく読んだのは、コードの 1 commit(S12 `14bc17a3`、cycle 3 の cross-review の 1 件。`watch.go` +16/-7、`watch_recurrence.go` +60/-1、`watch_test.go` +275/-0)と、記録の commit(`7f375027` の計画の Progress、`abf95ceb` の self-review run 4)。`10a86e28` のあとの /test(`24ca3804`)、/sync-docs(`baee33db`、`ad5beef6`)、triage(`e4b3ecb0`)は、run 3 の所見の処理を確かめるのにだけ使った。origin/main は `028a8f6d` のまま(`git ls-remote`)なので、base との衝突はない
- Evidence: `docs/evidence/verify-2026-11-12-watch-alert-dedupe.log`(git が無視する)。run 3 までの内容のあとに、run 4 の静的解析、probe、tech-debt の Trigger の照合を足した。足す前の 35445 バイトが先頭にそのまま残ることを `head -c | cmp` で確かめた。スクリプト自身の写しは `docs/evidence/verify-2026-11-12-033620.log`

## Spec compliance

S12 が変えたのは、`evaluateCycle` の初め(座席の評価の前)に `reconcilePendingAlertsWithInbox` を呼ぶことだけである。受信箱がすでに持つ pending(件の `alert_id` か、件の `recurred` の event の `alert_id` と同じもの)を `escalateAlert` に通し、pending から消す。続きの扱い(AC1〜AC5)と古い形の読み込み(AC12)を中心に読み直した。`watch_scope.go`、`inbox.go`、`internal/cli/` は `10a86e28` のあと変わっていないので、AC8〜AC11、AC14、AC15 は道が同じことだけを確かめた。`watch.go` の行は 1036 行より後で 8 行、1800 行より後で 9 行ずれたので、下の表は HEAD の行番号で書いた。テストの欄は、テストがあり assert が AC を言っていることまでを読んで確かめた。テストは走らせていない(/test の役目)。

| Acceptance criterion | Status | Evidence |
| --- | --- | --- |
| AC1 open か acked のあいだは件が 1 つで、立つたびに `recurred` が 1 つ増え、show と `--json` に回数と最後の時刻が出る | Met | 通常の道は S12 のあとも同じ。`internal/org/inbox.go:398-420`(`AppendRecurred`)、`watch.go:1315-1330`(`raiseOrClear`)、`watch_recurrence.go:168-196`(`recordRecurrence`)、`internal/cli/org_inbox.go:482-502`。テスト `watch_test.go:3126`(5 回立って件 1、recurred 4)、`TestInboxRead_RecurredCountsFiringsWithoutChangingTheState`、`TestOrgInbox_ShowsTheRecurrences`。止まる窓では件の回数が 1 多くなりうる(self-review の W3〜W5、C4-3、tech-debt の (7))が、AC は止まらない運転の回数を言うので Met のまま |
| AC2 resolved のあとは新しい件 | Met | S12 で、件を記録したあと状態ファイルを保存する前に watch が止まった場合にも成り立つようになった。前は、残った pending A が resolved のあとの発火を畳み、`checkDeadman` の replay で A ごと消えて、新しい ALERT が出なかった。今は `evaluateCycle`(`watch.go:1036-1041`)が座席の前に `reconcilePendingAlertsWithInbox`(`watch_recurrence.go:119-139`)を呼んで A を消す。そのあとの発火は `recordRecurrence` の受信箱の枝(`:174-195`)に進み、resolved の件は `openAlertItemForKey`(`:75-83`)に拾われないので `sendAlert` になる。テスト `TestWatch_Recurrence_APendingAlertAStoppedWatchEscalatedTakesNoNewFiring`(`watch_test.go:3750`)の resolved の形は、leader への ALERT が 2 通、新しい pending が 1 つで回数 0、e1 の recurred が 0 であることを見る。`TestWatch_Recurrence_APendingAlertOnARecurredEventIsDroppedBeforeTheSeats`(`:3878`)の resolved の形も同じ向き。前からのテスト `:3197-3208` |
| AC3 続きのあいだは leader に agmsg の ALERT を送らない | Met | S12 のあとも、open の件への発火は件の再発になり、ALERT は送らない。`:3750` の open の形は ALERT が 1 通のままで、e1 に新しい firing の alert_id の recurred が count 1 で 1 つ足されること、`:3878` の open の形は ALERT が 0 通であることを見る。通常の道は `watch.go:1323-1325`、`foldIntoPendingAlert`(`watch_recurrence.go:221-249`)、`leaderActedSinceAlert`(`watch.go:1768-1783`)。テスト `:3551`、`:3650` |
| AC4 deadman の前の再発は pending を増やさず、escalate した件に回数が残る | Met | reconcile は alert_id で照らし、key では見ないので、まだ escalate していない pending を消さない。`TestWatch_Recurrence_APendingAlertTheInboxDoesNotHoldIsKept`(`:3952`)は、同じ key の open な件が別の alert_id の recurred を持っていても、pending がそのまま残り、`Escalated`、受信箱、stderr が変わらないことを見る。受信箱に alert_id を書くのは `watch.go:1896`(`Escalate`)、`:1901`(運んだ回数)、`watch_recurrence.go:186`(firing ごとの新しい alert_id)、`:263`(`escalateAsRecurrence`)の 4 か所だけ(`git grep`)。回数の道は `watch_recurrence.go:246`、`watch.go:1900-1905`。テスト `:3225`、`:3551` |
| AC5 watch の立て直しで記録が無くても、同じ鍵の片付いていない件に `recurred` を足す | Met | 立て直しの 2 つの形がそろった。記録が無い形(前から)は `watch_recurrence.go:174-195`、`watch.go:1880-1891`、`escalateAsRecurrence`(`watch_recurrence.go:261-273`)。テスト `:3299`、`:3392`、`:3699`、`:4107`。古い記録が残る形(S12)は、周期の初めに replay して消し、そのあとの発火は open の件の `recurred` になる(`:3750` と `:3878` の open の形)。受信箱が読めないときは何もしない(`TestWatch_Recurrence_AnInboxThatCannotBeReadLeavesThePendingAlertsAsTheyAre`、`:3985`。self-review の W9)。残る窓は tech-debt の (2)、(7) と self-review の W3、W4 |
| AC6 片付いていない QUESTION・BLOCKED・RESULT の件があるあいだ、leader の stall は立たず受信箱にも入らない | Met | 変わっていない。`watch.go:1163-1203`(`leaderIsWaiting`)、`:1265`。テスト `watch_test.go:4250`、`:4302` |
| AC7 ほかの座席が `working` で文字列が変わるあいだは立たず、変わらなくなれば今までどおり立つ | Met | 変わっていない。`watch.go:1043-1061`(leader 以外を先に、leader を最後に評価)、`seatBusy`(`:1205`)。reconcile はこの並びの前に入り、`Conditions`、`SeatSnapshots`、周期の `rr` と `now` を変えない。テスト `:4351`、`:4394`、`:4426` |
| AC8 予約の中だけと、自分の新しい計画と報告だけの変更では立たない | Met | 変わっていない。`watch.go:1282-1285`、`watch_scope.go:349-370`(`dropOwnRecords`)、`:124`、`:296-298`(`-c diff.relative=false`)。テスト `watch_scope_test.go:397`、`:435`、`:748` |
| AC9 予約の外の変更で立ち、本文に外のパスが出る。外から中への移動を 1 周期で commit しても移動元が出る | Met | 変わっていない。`watch_scope.go:295-305`。テスト `watch_scope_test.go:483`、`:662`、`:748` |
| AC10 予約のない org は今までどおり | Met | 変わっていない。`watch.go:1287-1295`(`gitStatus`)。テスト `watch_scope_test.go:1285` |
| AC11 除く規則が役割 prompt の書かせる `docs/` のパスを覆う | Met | 変わっていない。`TestOwnRecord_CoversTheDocsPathsTheRolePromptsWrite`(`watch_scope_test.go:1363`) |
| AC15 rebase で入った upstream の commit では立たない | Met | 変わっていない。`watch_scope.go:298`。テスト `watch_scope_test.go:602`、`:642`、`:694`、`:1141` |
| AC14 分岐点にあった記録の変更と、分岐点が分からないときは立つ | Met | 変わっていない。`watch_scope.go:359-365`、`:331`。テスト `:536`、`:694` |
| AC12 古い watch-status と `recurred` の無い受信箱を読む | Met | S12 は watch-status と受信箱の形を変えていない(`14bc17a3` の diff に `json:"` の行が 0)。reconcile が読み書きするのは既存の `PendingAlerts` と `Escalated` だけ。テスト `watch_test.go:4020`、`:4076`、`watch_scope_test.go:1007`、`inbox_test.go:859` |
| AC13 help、agent-messaging(2 つの写し)、仕様の注記、RC3 の行 | Met | help(`internal/cli/org_watch.go:83-95`)、agent-messaging の 2 つの写し(同じバイト列、`cmp`)、仕様の FR-3 と FR-6 の注、RC3 の注(tech-debt 375 行)は、S12 のあとも合う。S12 は書かれた規則(「件が resolved になれば、次の発火は新しい ALERT」)を変えず、watch が止まったあとでもそれが成り立つようにした。どれも replay がいつ走るかは書いていない(INFO I15、I16) |

計画の決定との照合: S12 は D1 の「件が resolved になるか、pending が leader の動きで消えたら続きは終わり」と D2 の中にある。受信箱がすでに持つ pending は件の側で続きが決まるので、pending を消して件の状態に任せる形である。2026-10-23 の計画の D4(同じ alert_id の件の replay と、open で `notified` の無い件の人への送り)は、`escalateAlert` を通すことで残した。計画の本文(digest の内側)は変わっておらず、S12 は Progress の 176〜177 行に記録されている。新しい AC は要らない。

cross-review と self-review の照合:

- cycle 3 の cross-review の 1 件(止まった watch が残した pending が、件の resolve のあとの発火を畳んで黙らせる): S12 で直っている。self-review の手順 1 と同じ道(file に pending A、受信箱に A の件 e1)をコードで追い直し、`escalateAlert` → `escalateToInbox` → `escalateRecordedItem`(`watch.go:1955-1964`)の replay で A が消え、そのあとの発火が `recordRecurrence` の受信箱の枝に進むことを確かめた。新しいテスト 4 本の assert は上の表のとおり
- self-review の C4-1〜C4-3(どれも LOW、コードは変えず register に置く案)を読んで確かめた。C4-2 (a) は、replay で `sendInboxItemToHuman`(`internal/org/escalate.go:435`)の最後が `AppendNotified`(`:450`)を書くので、`evaluateCycle` の comment の「a replay that records nothing new」が言い過ぎになることを確かめた。C4-2 (b) は、`escalateAlert` の doc(`watch.go:1785-1789`)が今も「the set checkDeadman ranges over」とだけ書くことを確かめた。どちらも comment で、文書の drift には数えない。tech-debt への記録は V18 に含めた

## Static analysis

| Command | Result | Notes |
| --- | --- | --- |
| `./scripts/run-static-verify.sh`(changed-language scope、golang) | PASS(exit 0) | gofmt ok、`go vet ./...` は出力なし、golangci-lint `0 issues.`、staticcheck は出力なし(go 1.27.3。どちらにも「Skipping」の行は無い) |
| 同じ実行の中の local verifier | PASS | shellcheck と `sh -n`(hooks の 2 つの写し)、guard の awk、`settings.json` の jq、Codex の hook 検査 3 つ |
| `scripts/check-sync.sh` | PASS | DRIFTED 0、ROOT_ONLY 0 |
| `scripts/check-pipeline-sync.sh`、`check-skill-sync.sh`、`check-template-purity.sh` | PASS | skill は 14 本とも同じ |
| tech-debt README の plan 参照の検査 | PASS | |
| branch の secret scan | PASS | `028a8f6d..abf95ceb` を origin/main に対して clean |
| `./scripts/plan-visual.sh digest` | `e5a18ef9477a` | `- Approved:` の行と一致 |
| `git diff --check 10a86e28 HEAD -- ':!docs/evidence'` | rc 0 | |
| `10a86e28` のあとに変わった 11 ファイルの U+2028、U+2029、U+FFFD、ゼロ幅、BOM、C1 制御、方向制御 | 0 件 | python で文字ごとに数えた |

## Documentation drift

| Doc / contract | In sync? | Notes |
| --- | --- | --- |
| `ralph org watch --help` | Yes | S12 で変わっていない。replay の時点には触れない |
| `.claude/rules/ralph/agent-messaging.md` と templates/base の写し | Yes | 「Once the item is resolved, the next firing is a new `ALERT`」は、S12 のあと止まった watch の後でも成り立つ(I16) |
| `/org` skill(4 つの写し) | Yes | 4 つは同じ(`cmp`、`check-skill-sync.sh`)。661〜668 行の replay の説明は S12 のあとも正しい(I16) |
| 仕様の FR-3 と FR-6 の注 | Yes(INFO I15) | run 3 の I11 は `baee33db` で入った |
| `docs/tech-debt/README.md` | 一部 | V18(462 行)、V19(357 行)。303 行は INFO I17 |
| 計画の Progress checklist | 遅れ | V20 |
| cross-review triage の rationale | 要約 | INFO I19 |

### 所見(run 4)

ID は run 3 の V15〜V17、I11〜I14 の続きにした。どれも /sync-docs に回す。コードの直しは含まない。

- V18(LOW): tech-debt の既知の穴の行(462 行、「Eight gaps that fix/watch-alert-dedupe leaves」)
  - (a) lead が挙げた既知のずれ。self-review の「止まる窓の一覧」のうち W1(`sendAlert` のあとで止まると leader に 2 通)、W3 と W4(運んだ回数を件に書いたあとで止まると件の回数が 1 多い、C4-3)、W7(受信箱の外への escalate のあとで止まると 2 回目の通知)、W9(周期の初めの読みだけが失敗し、`checkDeadman` の読みが通り、件が resolved のときだけ黙る)は行に無い。C4-1(held の判定が `reconcilePendingAlertsWithInbox` と `escalateToInbox` の 2 か所)と C4-2(comment の 2 か所)も無い。S12 の周期の初めの reconcile も書かれていない。W2 は (2) と、W5 は (7) と同じ窓で、どちらの文も S12 のあとも正しい
  - (b) この verify で見つけた点。S12 は行全体の Trigger(「the recurrence code of `internal/org/watch.go`, `internal/org/watch_recurrence.go`」)にもう一度当たった。項目の Trigger では、(8) の「the next change to `leaderActedSinceAlert` or `foldIntoPendingAlert`」に名前で当たった。S12 は `foldIntoPendingAlert` の doc に 3 行を足し、コードは変えていない。(8) の 2 本のテストは足されていないので払われていない。(2) の Trigger の直し(運ぶ回数を件と同じ書き込みで書く)も払われていない。cycle 2 と cycle 3 の段落に続けて、S12 の当たりと引き継ぎを 1 文で足す
  - (c) Why deferred の (8) の「(cycle 3 of 3: the cap of 2 was raised to 3 for this task, and its third run went to S10 and S11)」と、Trigger の欄の「they are code changes after the pipeline's last run」は、S10 と S11 の run を最後の run としている。そのあとに run 4 (cycle-count 3) があり、S12 に使われた。run の数え方(4 回目、`cycle-count.json` は cycle 3)を足す
- V19(LOW): tech-debt の 357 行(`watch.go` と `watch_test.go` の大きさ)。Trigger は「The next change to the watch's escalation or to its tests」で、S12 はこれに当たった。`escalateAlert` に 2 つめの呼び出し元(周期の初めの reconcile)を足し、その doc を変え、escalation のテスト 4 本(`watch_test.go` +275 行)を足した。払われていない(escalation のコードを `watch_escalate.go` に移していない)。注の「At e90c85da, the last Go commit of the branch, `watch.go` is 2081 lines, `watch_test.go` 4188, `watch_recurrence.go` 214, and `watch_scope.go` 467」は古い。今の最後の Go の commit は `14bc17a3` で、`watch.go` 2090、`watch_test.go` 4463、`watch_recurrence.go` 273、`watch_scope.go` 467(`git show <commit>:<file> | wc -l`)。S12 の当たりを 1 文で足し、commit と数を直す。lead の既知の一覧には無い
- V20(LOW): 計画の Progress checklist
  - (a) self-review run 4(`abf95ceb`: Merge yes、LOW 3 件 C4-1〜C4-3、止まる窓の一覧 W1〜W9)とこの verify の行が無い(run 3 の V17 と同じ型)
  - (b) S12 の行(177 行)の「受信箱が読めないときは何もしない(知らせる側)」は、W9 の狭い黙る場合(周期の初めの読みだけが失敗し、件が resolved のとき)を除いた言い方になっている。同じ行の「消す pending が持っていた回数は失われる」は W2 だけに当たり、W3 と W4 では回数が 1 多くなる(self-review の C4-3)。どちらも記録の言い過ぎで、Progress は digest の外(`scripts/plan-visual.sh` の `digest_body` は `## Progress checklist` を飛ばす)なので、直しても承認は変わらない

### INFO(run 4)

- I15: 仕様の FR-6 の 2026-11-12 の注は、S12 の周期の初めの reconcile に触れない。同じ節の 2026-10-23 の注の「同じ alert_id の件がもう受信箱にあれば新しい件は書かず、その件が open で `notified` がなく、今の見張りの判定が delivery `now` のときだけ、同じ ID で人への経路に送る」は、S12 のあとも正しい。S12 が変えたのは replay の時点(周期の初め)と、止まったあとも「resolved の件のあとの発火は新しい ALERT」が成り立つことだけで、書かれた文と食い違わない。足すなら「見張りが止まって状態ファイルの記録が残っても、次の見張りは最初の周期の初めに、受信箱がすでに持つ ALERT を片付けてから座席を見る」の 1 文でよい。lead が挙げた既知の項目
- I16: help、agent-messaging(2 つの写し)、`/org` skill(4 つの写し)は replay がいつ走るかを書かない要約で、S12 のあとも正しい。変える必要はない
- I17: tech-debt の 303 行(`newTestWatchRun` の一時ディレクトリ)。Trigger「The next change to `watch_test.go`」に S12 が当たったが、払っていない(`newTestWatchRun` は今も `os.MkdirTemp`)。S12 の新しいテストは `newWatchInboxFixture` を使い、`newTestWatchRun(` を含む行は `10a86e28` と HEAD のどちらも 30(定義の 1 行と呼び出し 29)で、注の「29 callers」(`3506f99a` で数えた)は今も合う。branch ごとの注が branch の後の commit も覆うと読む扱いは、cycle 2 の I10 と同じ。足すかは任意
- I18: S12 の端の変化は、Progress の「deadman の前でも周期の初めに再送の判定を通る」より 1 つ広い。止まった watch が escalate した pending は、もともと deadman を過ぎていた(か対象が leader だった)ので、前との違いは主に leader の動きのほうにある。前は、そのあと leader が動いていれば `checkDeadman` が replay なしで消していた。今は leader の動きによらず replay を通る(`reconcilePendingAlertsWithInbox` の doc の「whatever the leader did since」)。replay が人に送るのは件が open で `notified` が無く、この見張りの判定が delivery `now` のときだけで、向きは知らせる側。もう 1 つ、手で書き換えて key が `""` になった pending は、`alert_id` を持たない leader の件(QUESTION など、`internal/org/escalate.go:203` は `AlertID` を渡さない)と一致し、周期の初めに `checkWatchEscalationValues` で断られて受信箱の外へ escalate される。前は deadman を過ぎてからだった。手の書き換えでしか起きず、向きは知らせる側なので、記録だけにする
- I19: cross-review triage の rationale の「so it is dropped (and noted in Escalated)」は、S12 が消す前に `escalateAlert` の replay を通すこと(人への送りがありうる)を書いていない。triage の Implementation context と、計画の Progress の 177 行、コードの doc は replay を書いている。triage は /cross-review の記録なので、変えなくてよい

## Observational checks

- S12 の diff を全部読み、cycle 3 の cross-review の 1 件と、self-review の C4-1〜C4-3 と止まる窓の一覧 W1〜W9 を、コードの道と突き合わせた(上の表と照合)
- 受信箱に watch の alert_id を書く所を `git grep -n 'AlertID: \|AppendRecurred('` で数え、4 か所だけであることを確かめた。leader の escalate(`escalate.go:203`)は `AlertID` を渡さない。firing ごとの alert_id(`watch_recurrence.go:186`)は、`recordRecurrence` が true を返す周期では pending にならないので、escalate していない pending を held と読む道は通常は無い(self-review の手順 3 と同じ結論。例外の close の失敗も同じく記録だけ)
- range の中で `escalateAlert` が `delete(status.PendingAlerts, alertID)` を打つのは、Go の仕様で今の key の削除は安全であることを確かめた。`checkDeadman` は reconcile の後に走り、消えた alert を range しないので、同じ alert を 1 周期で 2 回 escalate しない
- 周期の初めの受信箱の読みの失敗について、doc の「the deadline sweep at the start of the cycle already says so on stderr」は、`watchAllRun.cycle`(`watch.go:746-750`)が毎周期 `SweepInboxDeadlines` を打ち、読みの失敗を `watchdog: inbox deadlines skipped this cycle:` の 1 行にすることと合う(読みは別の時点なので、片方だけが失敗する場合は W9)
- tech-debt の Trigger の欄を python で切り出し(`re.split(r"(?<!\\) \| ", row)` の 4 つ目のセル)、S12 が触った名前(`evaluateCycle`、`escalateAlert`、`escalateToInbox`、`reconcilePendingAlertsWithInbox`、`foldIntoPendingAlert`、`checkDeadman`、`watch.go`、`watch_recurrence.go`、`watch_test.go`、escalation、deadman、status file、recurrence)で照合した。当たったのは 462 行(V18)、357 行(V19)、303 行(I17)。210 行は Trigger(「`evaluateCycle` の呼び出しの打ち切り方の変更」)に当たらない。reconcile は herdr、agmsg、git を打たず、受信箱のファイルを 1 回読むだけで、replay の送りは前と同じく `checkDeadman` の中と同じ道を通る。457 行は名前を注の中で拾っただけで、Trigger の `leaderActivityEventCount` は変わっていない。179、211、262 行も文脈の語を拾っただけだった。出力は evidence の log にある
- run 3 の所見の処理を確かめた(下の「Run 3 の記録」)
- origin/main は `028a8f6d` から動いていない(`git ls-remote origin refs/heads/main`)

## Coverage gaps

- Go のテストは走らせていない(/test)。表のテストの欄は、テストがあり assert が AC を言っていることまでである。S12 の新しいテスト 4 本はコンパイルも含めて読んだだけで、前の実装で落ちることも、この verify では確かめていない
- 止まる窓(W1〜W9)は、self-review と同じくコードを読んで追ったもので、watch を実際に止めて立て直す実行はしていない
- 実機の走行は無い。機能の org を新しい binary で走らせ、受信箱の件を数え直すことは RC3 の注のとおり残っている
- cycle 1 からの「決定的な検査が無いもの」(skill の `--json` の欄の一覧が `org.InboxItem` の JSON の欄に追いつかないこと)は今も無い。最小の検査は、`InboxItem` の `json:"..."` の名前のうち基本の欄を除いたものが `.claude/skills/org/SKILL.md` の `inbox` の行にすべて出ることを見る Go のテスト 1 本である
- C4-1 の 2 か所の held の判定がそろっていることを見る決定的な検査は無い。最小の検査は、`escalateToInbox` が「記録済み」として何も足さない受信箱の形の一覧を表にして、同じ表で `reconcilePendingAlertsWithInbox` が held と読むことを見るテスト 1 本である(self-review の勧めどおり、判定を 1 つの関数に寄せれば要らなくなる)

## Verdict

pass

- Verified: AC1〜AC15 は、コードの道とそれを assert するテストがあることを確かめた。S12 は cycle 3 の cross-review の 1 件を直し、watch が件を記録したあと状態ファイルを保存する前に止まった場合も、AC2(resolved のあとは新しい ALERT)と AC3・AC5(open の件には再発)が成り立つようにした。reconcile は alert_id で照らすので、まだ escalate していない pending(AC4)を消さない。watch-status と受信箱の形は変わっていない(AC12)。静的解析はすべて通った(gofmt、go vet、golangci-lint、staticcheck、shellcheck、同期の検査 4 つ、template purity、tech-debt の plan 参照、secret scan)。計画の digest は承認の行と同じ。base との衝突はない
- Not verified: Go のテストの結果と `run-test.sh`(/test)、S12 のテストが前の実装で落ちること、止まる窓の実際の再起動、実機での件の数え直し
- 所見(run 4): LOW 3 件(V18〜V20)、INFO 5 件(I15〜I19)。どれも文書と記録のずれで、merge を止めない。lead が挙げた既知のずれ(462 行の止まる窓の一覧、C4-1〜C4-3、S12 の reconcile、FR-6 の注)は V18 (a) と I15 で確かめた。V18 (b) と (c)、V19、V20、I17〜I19 はこの verify で新しく見つけた
- /sync-docs に回すもの: V18(462 行の止まる窓、C4-1〜C4-3、S12 の当たりと (8) の名前での当たり、run 4 の数え方)、V19(357 行の S12 の当たり、`14bc17a3` と 4 つの数)、V20(計画の Progress に `abf95ceb` とこの verify、S12 の行の 2 つの言い過ぎ)。I15 と I17 は任意

## Run 3 の記録

run 3(cycle 3、HEAD `521edd5b`、commit `10a86e28`)の判定は pass で、所見は LOW 3 件(V15〜V17)と INFO 4 件(I11〜I14)だった。V15〜V17 と I11 は run 3 の /sync-docs(`baee33db`、`ad5beef6`)が処理した(`docs/reports/sync-docs-2026-11-12-watch-alert-dedupe.md`)。この run で、処理された文が今も HEAD にあることを確かめた。

| ID | 中身(run 3) | 処理 |
| --- | --- | --- |
| V15 | 462 行の既知の穴の行。(a) C3-2: (8) の Why deferred と Trigger の欄の cycle 2 の段落が「cap 2 of 2」「cap of 2 runs」と書き、cap が 3 に上がったことと合わない。(b) S10 と S11 は `scopeCommittedPaths` の引数と doc を変え、行全体の Trigger(`internal/org/watch_scope.go`)にもう一度当たった。(5) と (6) の Trigger は当たっていない(hunk は `scopeCommittedPaths` の中だけ)。cycle 3 の引き継ぎを 1 文で足す。(c) (3) の「`scopeCommittedPaths` gives none」は S10 と S11 のあとも正しい | `baee33db`。`cap 2 of 2` は 0 件、cycle 3 の段落が入った。run 4 で、run の数え方が古くなった(V18 (c)) |
| V16 | 357 行の「At 3bb1828b, the last Go commit of the branch, ... `watch_scope.go` 461」が古い(`e90c85da` で 467) | `baee33db`。`e90c85da` と 467 に直った。S12 で再び古くなった(V19) |
| V17 | 計画の Progress に `521edd5b` とこの verify の行が無く、cycle の行の並びが時の順でない | `ad5beef6`(172〜175 行)。並べ替えのあとも digest は `e5a18ef9477a` |

INFO(run 3): I11 は、仕様の FR-3 の注と tech-debt の RC3 の注が commit の一覧の `-c diff.relative=false` に触れないこと(`baee33db` で両方に「利用者の `diff.relative` によらず root からの相対」が入った)。I12 は、`evaluateReservedScope` の doc(`watch_scope.go:380-393`)が 2 つのコマンドを `-c diff.relative=false` なしで要約し、`scopeCommittedPaths` を指すこと。直すとコードの変更になり、tech-debt 462 行の (5) の Trigger(N5 の文を同じ編集で直す)にも当たるので、今は直さない(462 行の cycle 3 の段落が「verify I12」として指す。S12 は `watch_scope.go` を変えておらず、今も同じ)。I13 は、計画の D6、Scope、Implementation outline の 4 が `--no-renames` だけを書く記録されたずれ。I14 は、`--no-relative` が git 2.28 からという doc を release notes とは照らしていないこと。

## Cycle 2 の記録

cycle 2(HEAD `3bb1828b`、commit `0a957745`)の判定は pass で、所見は LOW 6 件(V9〜V14)と INFO 5 件(I6〜I10)だった。AC13 は「一部」(V9、V10)。V9〜V14 は cycle 2 の /sync-docs(`0cc32e6d`、`74e6b1db`、`cd4a9268`)がすべて処理した(`docs/reports/sync-docs-2026-11-12-watch-alert-dedupe.md`)。

| ID | 中身(cycle 2) | 処理 |
| --- | --- | --- |
| V9 | 仕様の FR-3 の注(82 行)に S8 の `--cc`(merge commit 自身の編集を数える)が無い。origin の既定のブランチの決め方は任意 | `0cc32e6d`。両方が入った |
| V10 | tech-debt の RC3 の注(375 行)に同じく `--cc` が無い | `74e6b1db`。両方が入った |
| V11 | 462 行の既知の穴の行。(a) C2-1 の残る窓、(b) M48 の Trigger が S8 で当たった、(c) (5) の Trigger が S8 で当たった、(d) 行全体の Trigger の引き継ぎ、(e) 見出しの数 | `74e6b1db`。(a) は (7)、C24 と C04 は (8)、(b)〜(d) は Trigger の欄の cycle 2 の段落、見出しは「Eight gaps」。cap の文は run 3 で直った(V15) |
| V12 | 210 行の git の呼び出しの数(S8 で最大 10 回、およそ 100 秒)と、`foldIntoPendingAlert` の呼び出し | `74e6b1db`。S10、S11、S12 のあとも正しい |
| V13 | 357 行の数と cycle 2 での当たり、457 行の (f) の判断 | `74e6b1db`。357 行の注の「the last Go commit」は run 3 と run 4 で古くなった(V16、V19)。457 行の (f) は注を足した |
| V14 | 計画の Progress に `d3a34921`、`3bb1828b`、cycle 2 の verify が無い | `cd4a9268` |

INFO(cycle 2): I6 は `/org` skill の 4 つの写しが S8 のあとも正しく、escalation の打ち直しの 1 文は任意(`0cc32e6d` で足された)。I7 は `docs/recipes/worktrees.md` の要約。I8 は計画の D6 の本文などが `--no-renames` だけを書くこと(I13 に続く)。I9 は self-review の報告(`d3a34921`)の C2b-1 が `3bb1828b` で直ったこと。I10 は tech-debt 303 行の `newTestWatchRun` の呼び出しの数が正しいこと(S12 のあとも同じ。I17)。

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
