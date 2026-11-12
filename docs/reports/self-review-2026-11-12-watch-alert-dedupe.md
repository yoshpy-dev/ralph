# Self-review report: watch-alert-dedupe

- Date: 2026-11-12
- Plan: `docs/plans/active/2026-11-12-watch-alert-dedupe.md`(issue #307 の前半)
- Branch: `fix/watch-alert-dedupe`、base は `origin/main` の `028a8f6d`。レビューした HEAD は `7f375027`。`14bc17a3` が cycle 3 の cross-review の指摘の修正(S12)で、`7f375027` はその記録(計画の Progress)
- Reviewer: reviewer サブエージェント(`/self-review`)。パイプラインの 4 回目(`cycle-count.json` は cycle 3、cap 3 で、これが cap 3 の最後の run)。insight event は `--cycle 4` で足した
- Scope: `14bc17a3` の diff(`watch.go`、`watch_recurrence.go`、`watch_test.go`)を全部読み、cross-review の指摘が直ったか、`reconcilePendingAlertsWithInbox` が watch を黙らせる道があるか、`checkDeadman` と `escalateAlert` との順序、二重の送りを確かめた。全体の diff は前の run で読んでいるので軽く見た(不可視 Unicode の走査、comment の長さ)。それに加えて、watch が外へ書いてから周期の終わりの保存までの間に止まった場合の一覧を作った(下の「止まる窓の一覧」)。diff の質だけを見た。テストの実行、静的解析、仕様適合、文書の drift は見ていない
- ID の付け方: 今回の新しい指摘は `C4-1` から `C4-3`。前の run の ID(cycle 1 の `M1`、`L1` から `L6`、`N1` から `N5`、cycle 2 の `C2-1` から `C2-4`、`C2b-1`、cycle 3 の `C3-1`、`C3-2`)は、tech-debt の行と計画が指すので「Earlier runs」にそのまま残した

## Evidence reviewed

cross-review(cycle 3)の指摘が直ったかの確認

| # | triage の指摘 | 直し(`14bc17a3`) | この run の確認 |
| --- | --- | --- | --- |
| 1 | watch が件を作った直後、状態の保存前に止まると古い pending が残る。件が resolve されたあとの再発がそこに畳まれ、`checkDeadman` の replay で pending ごと消えて、新しい異常が送られも記録もされない | `evaluateCycle` が座席の評価の前に `reconcilePendingAlertsWithInbox` を呼ぶ。受信箱が件の `alert_id` か recurred event の `alert_id` で持つ pending を、`escalateAlert` で replay してから消す | 直っている。下の 1 から 6 で確かめた(コードを読んで追った。テストは走らせていない) |

1. 指摘の道を追い直した。止まったあとの file は pending A を持ち、受信箱は A の件 e1 を持つ。次の watch の周期の初めに、reconcile が e1 の `AlertID == A` で A を held と読んで `escalateAlert` に通す。`escalateToInbox` は `escalateRecordedItem` で replay し(新しい件も recurred event も足さず、stderr に 1 行)、A は `Escalated` に入って `PendingAlerts` から消える。そのあと座席が同じ key の発火を読むと、`foldIntoPendingAlert` に pending が無いので `recordRecurrence` の受信箱の枝に進み、件の今の状態で決まる(open か acked なら再発、resolved なら新しい ALERT と新しい pending)。指摘の道に、再発が消える先は残っていない。
2. 新しいテスト 4 本(`watch_test.go`、+275 / -0 で既存の行は消えていない)を読んだ。
   - `TestWatch_Recurrence_APendingAlertAStoppedWatchEscalatedTakesNoNewFiring`: e1 が resolved の形と open の形。前の実装では畳み込み、`checkDeadman` の replay、何も出ない、の順になり、resolved の形で 2 通目の ALERT が出ない。直しのあとは resolved で 2 通目の ALERT と新しい pending、open で件への再発 1 つと ALERT なし。stderr の 1 行、件の数、通知の数、`Escalated` まで固定している
   - `TestWatch_Recurrence_APendingAlertOnARecurredEventIsDroppedBeforeTheSeats`: recurred event 側の形。deadman まで遠い(1 分、設定 30 分)のに周期の初めに落ちることを見る
   - `TestWatch_Recurrence_APendingAlertTheInboxDoesNotHoldIsKept`: 同じ key で alert_id の違う件があっても pending を残す(key ではなく alert_id で見る)
   - `TestWatch_Recurrence_AnInboxThatCannotBeReadLeavesThePendingAlertsAsTheyAre`: 受信箱が読めなければ何もしない
   - 使う helper(`escalateAlertItem`、`recurredEvents`、`inboxItemIDs`、`inboxItemByID`、`assertEscalatedOnce`、`watchdogAlerts`、`mustReadInbox`、`stallCycle`、`readJSONLFile`)と import(`reflect`、`slices`、`errors`、`os`、`strings`)は既存の側にある。コンパイルと実行は見ていない
3. 黙る道の洗い出し。
   - held の判定が正しいか。受信箱に watch の alert_id を書く所は、`escalateToInbox` の `Escalate` と運んだ回数の `AppendRecurred`、`escalateAsRecurrence`、`recordRecurrence` の `AppendRecurred` の 4 つだけ(`AlertID:` と `AppendRecurred(` を `internal` 全体で grep)。最後の 1 つの alert_id は firing ごとの `watchAlertID(key, now)` で、`recordRecurrence` が true を返す周期では pending にならない。escalate していない pending を held と読んで消す道は、通常は無い
   - 例外が 1 つある。`AppendRecurred` が書き終えたあとの `f.Close()` の失敗(`appendLocked`)で `recordRecurrence` が false を返すと、同じ周期の `sendAlert` が同じ `now` で同じ alert_id の pending を作り、次の周期の reconcile がそれを held と読んで消す。ALERT は leader に送ってあり、件は open のまま自分の ack と resolve の期限を持つので、人に届かなくなる道ではない。書き込み後の close の失敗が要るので、記録だけにして指摘にはしない
   - 消す前に必ず `escalateAlert` を通る。通るのは replay か、受信箱が受けられないときの `escalateWithoutInbox`(banner、通知、`escalations.jsonl`)で、何も送らず記録もせずに pending だけを消す道は無い
   - reconcile は `Conditions` を触らない。key が Active かどうかは、前と同じく保存された file が決める
   - 受信箱が nil か、周期の初めに読めないとき: 何もしない。残る穴は下の W9
4. 順序。`evaluateCycle` は `pruneRetiredConditions`、reconcile、座席、`rr2` の読み直し、`checkDeadman`、保存の順。reconcile が変えるのは `PendingAlerts` と `Escalated` だけで、座席の評価が読む `Conditions`、`SeatSnapshots`、周期の `rr` と `now` は変えない。reconcile が人への送りで長引いても、`checkDeadman` は `rr2` を読み直す。held の pending は `checkDeadman` の range に入らないので、同じ周期で 2 回 escalate されない。`SweepInboxDeadlines`(`cycle()` の最初)は pulse の前に走るので、sweep が送って `notified` を書いた件は replay が送らず、sweep の送りが失敗して event の無い件は replay がもう一度送る(2026-10-23 の D4 と AC3 の設計で、重複ではなく再試行)
5. 二重の送り。replay が書くのは、送る場合の `notified` event だけ(`sendInboxItemToHuman` の最後が `AppendNotified`)。1 つの件に pending が 2 つ(件の `AlertID` が X1、recurred が X2)ついても、送る枝は X1 の replay だけで、`alertItemWithRecurredID` の枝は送らない。`escalateToInbox` は alert ごとに受信箱を読み直す(reconcile の読みと合わせて N+1 回)が、受信箱は追記だけなので、reconcile が held と読んだものは読み直しても held
6. そのほか見たもの。branch 全体の追加行の不可視 Unicode(U+0080-009F、U+2028/2029、ゼロ幅、方向制御、BOM、U+FFFD)の走査は一致なし。80 桁を超える comment 行は `watch.go` 7、`watch_recurrence.go` 1 で、`14bc17a3` の前後で変わらない。`foldIntoPendingAlert` の doc の「unless the inbox could not be read」は上のテストと合う。「cap N of N」や「the pipeline's last run」の文は、`14bc17a3` と `7f375027` の追加行に無い。新しい comment とテストの「cycle 3 cross-review of plan 2026-11-12-watch-alert-dedupe」は番号を使わず場面を書く形で、`3bb1828b` が採った形と同じ

## 止まる窓の一覧

`/sync-docs` が tech-debt に写す表。watch の状態の保存(`status.save`)は一時 file と rename(`writeJSONFileAtomic`)なので、半端な file は残らない。窓はすべて「外への書き込み」(agmsg、受信箱、通知、`escalations.jsonl`)と「周期の終わりの保存」の間にある。表の内容はコードを読んで追ったもので、再起動の実行はしていない。

| # | 書き込みの境目 | 止まったあとの watch | 向き | 受け持ち |
| --- | --- | --- | --- | --- |
| W1 | `sendAlert`: agmsg で leader に送る、周期の終わりの保存の前 | file に key の Active も pending も無い。まだ発火していれば立ち上がりと読み直し、別の alert_id で 2 通目の ALERT を送る(key に open の件があれば、その件の再発に +1)。1 通目の pending は残らないので、deadman は 2 通目だけを追う | 二重(leader に 2 通) | S12 の外。テストなし。tech-debt に行なし。この branch より前からある。(9) に足す |
| W2 | `escalateToInbox`: `Escalate` が件を書く、運んだ回数の `AppendRecurred` の前 | 件はあり、recurred event は無い。file の pending が回数 n を持つ。次の周期の初めに S12 が件の `AlertID` で held と読み、replay して pending を消す。そのあとの発火は件の状態で決まる(open か acked なら再発、resolved なら新しい ALERT) | 黙る向きは S12 で閉じた。回数は n が失われる(低い) | 黙る向き: S12 とテスト `..._APendingAlertAStoppedWatchEscalatedTakesNoNewFiring`(回数 0 の形)。回数: tech-debt の (2)、テストなし |
| W3 | `escalateToInbox`: 運んだ回数の `AppendRecurred` の後、保存の前 | W2 と同じ replay。件には回数が届いている。止まった周期でその key の立ち上がりを pending に畳み(n+1 を件に運んだ)、止まったあとも発火が続くと、保存されなかった Active の代わりに立ち上がりをもう一度読み、件の再発に +1 する | 回数が 1 多い | S12(黙る向き)。この +1 のテストなし、tech-debt なし(C4-3)。S12 の前は再発が pending に畳まれて件の回数は合ったが、件が resolved のときは再発が黙って失われた |
| W4 | `escalateAsRecurrence`: 件への recurred(alert_id X、1+n)の後、保存の前 | `alertItemWithRecurredID` で held と読み、replay して pending を消す。W3 と同じ +1 の枝がある | 回数が 1 多い(W3 と同じ条件) | S8 のテスト `..._AnEscalationAlreadyOnARecurredEventRecordsNothingNew` と S12 のテスト `..._APendingAlertOnARecurredEventIsDroppedBeforeTheSeats`。+1 は W3 と同じく C4-3 |
| W5 | `recordRecurrence`: open な件への recurred(`watchAlertID(key, now)`、1)の後、保存の前 | Active が無いので、発火が続いていれば立ち上がりをもう一度読み、同じ件にもう 1 つ recurred を足す。件が resolved なら新しい ALERT | 回数が 1 多い | tech-debt の (7)。テストなし(S9 が、周期の時刻から作る alert_id の確認を、本番では当たらないので消した) |
| W6 | 件の記録、人への送り(`sendAlertItemToHuman`)、`notified` event | 件が open で `notified` が無ければ、replay が周期の初めに人へ送る(S12 の前は `checkDeadman` の中)。osascript が通ったあと event を書く前に止まったなら、2 回目の通知が出る | 二重(デスクトップ通知) | 2026-10-23 の D4 と AC4 のテスト `TestWatch_Escalation_AnAlertAlreadyInTheInboxGetsNoSecondItem`(送る形と送らない形)。S12 は replay の時点を前に出しただけ |
| W7 | `escalateWithoutInbox`(受信箱が受けられないとき): banner、通知、`escalations.jsonl`、保存の前 | pending が file に残り、もう一度 escalate する。受信箱が直っていれば件を作り、まだなら 2 回目の通知と jsonl の 1 行 | 二重 | 2026-10-23 の D5。この branch の外。再起動のテストなし、tech-debt に行なし。(9) に足す |
| W8 | pending を消すだけの境目(`checkDeadman` の活動での消去、`foldIntoPendingAlert` の episode の終わり)、保存の前 | 外への書き込みが無い。pending が file に残るだけで、manifest から同じ判定を取り直す。probe の text が戻っていて pending が残れば、あとで escalate される | なし(残るなら知らせる向き) | 対応不要 |
| W9 | 周期の初めに受信箱が読めない(止まりではなく、S12 の残り) | reconcile は何もしない。held の pending が残り、発火を畳むことがある。そのあとの escalate が同じ失敗で `escalateWithoutInbox` に進めば、人に通知が出る。黙るのは、周期の初めの読みだけが失敗し、`checkDeadman` の読みが通り、件が resolved のときだけ。`InboxStore.Read` は lock を取らない `os.ReadFile` なので、1 周期の中の一時的な失敗は考えにくい | 黙る向き(極小) | S12 の方針(読めなければ何もしない)とテスト `..._AnInboxThatCannotBeReadLeavesThePendingAlertsAsTheyAre` が固定。tech-debt に行なし。(9) に足す |

向きでまとめると、黙るのは W9 だけで、周期の中の一時的な読み失敗が要る。二重は W1、W6、W7。回数が違うのは W2(低い)と W3 から W5(1 多い)。S12 が閉じたのは W2 から W4 の「件の記録のあとで止まる」窓の黙る向きの枝で、W3 と W4 では黙る向きを閉じた代わりに +1 が出る(C4-3)。

## Earlier runs

cycle 1 は 3 回、cycle 2 は 2 回、cycle 3 は 2 回(この run の前)。ID は run ごとに分けてある。

### cycle 3 の 2 回目(commit 521edd5b、HEAD `bfbf0a2a`)

CRITICAL・HIGH・MEDIUM なし、LOW 1(`C3-2` の持ち越し)。`C3-1` が `e90c85da` で直っていることを、本番の引数の順の実 git 2.49.0 と 2.26.3(docker の `alpine:3.12`)で確かめた。`-c diff.relative=false` は 2 つのコマンドの両方で、`diff.relative=true` と下のディレクトリの cwd の下でも root からの相対のパスを返し、2.26.3 でも終了コード 0。2.28 の境目は打っていない。

| ID | 指摘 | 処理 |
| --- | --- | --- |
| C3-2 (LOW) | tech-debt の「Eight gaps」の行が、(6) と (8) を直さない理由を「cap 2 of 2」と書くが、cap は 3 に上がった | `baee33db`(cycle 3 の `/sync-docs`)で直った。この run で `docs/tech-debt/README.md` に「cap 2 of 2」が 0 件であることを確かめた。解消 |

### cycle 3 の 1 回目(commit 65a82b01、HEAD `72d7d6ad`)

CRITICAL・HIGH・MEDIUM なし、LOW 2。cycle 2 の cross-review の ACTION_REQUIRED(`diff.relative=true` と下のディレクトリの cwd で、commit のパスが cwd からの相対になる)は `ec538952` で直っていることを確かめた。

| ID | 指摘 | 処理 |
| --- | --- | --- |
| C3-1 (LOW) | `ec538952` の `--no-relative` は git 2.28 以降の option で、同じ設定にこの repo が使ってきた `-c diff.relative=false` と形が違う。古い git では 2 つのコマンドが引数の誤りで失敗し、呼び出し元は「commit のパスを取れなかった」と読んで前の HEAD を据え置き、stderr に何も出さないので、`scope_change` が commit からは毎周期、黙って働かない。git 2.26.3 で `log --no-relative` が終了コード 128、`-c diff.relative=false` が 0 | `e90c85da` で `-c diff.relative=false` に替えた。cycle 3 の 2 回目で解消を確認 |
| C3-2 (LOW) | tech-debt の「Eight gaps」の行が、(6) と (8) を直さない理由を「cap 2 of 2」と書くが、cap は 3 に上がった(行を書いた `74e6b1db` のあとで `7aee125e` が上げた) | `baee33db` で直った。cycle 3 の 2 回目では未処理で、この run(cycle 3 の 4 回目)で解消を確認 |

### cycle 2 の 2 回目(commit d3a34921、HEAD `7ac1e5c1`)

CRITICAL・HIGH・MEDIUM なし、LOW 1。`C2-1` から `C2-4` のうち 3 件が直り、`C2-3` は 4 件中 3 件が直った。

| ID | 指摘 | 処理 |
| --- | --- | --- |
| C2b-1 (LOW) | `AnEscalationAlreadyOnARecurredEventRecordsNothingNew` の doc が、折り返された行で「(cross-review / finding 4)」と triage を番号で指す。`C2-3` の指摘は行ごとの grep で数えたので、2 行にまたがるこの 1 か所を拾わなかった | `3bb1828b` で「found by the cycle 1 cross-review of plan 2026-11-12-watch-alert-dedupe, the escalation replay」に直った(cycle 3 の run で grep し直して確認)。解消 |

注として、`5393a316` が書き換えた comment の 4 行が 80 桁を超えること、`watch_recurrence.go:145-148` で「test」が 2 回出て読みにくいことを書いた。重さは付けていない。`3bb1828b` は `watch_test.go` の comment だけを変えたので、この 2 点は今も同じ。

### cycle 2 の 1 回目(commit f853d973、HEAD `d3a63398`)

CRITICAL・HIGH・MEDIUM なし、LOW 4。cross-review の ACTION_REQUIRED 4 件(`907afde3`)は、次のとおり確かめて、4 件とも直っていた。

| 番号 | triage の指摘 | 確認 |
| --- | --- | --- |
| 1 | leader の活動が初めて見える周期に再発した firing を古い pending に畳むと、同じ周期の `checkDeadman` がその pending を消して firing が失われる | `leaderActedSinceAlert` を取り出して共有し、`foldIntoPendingAlert` が畳む前に同じ判定を使う。直っている |
| 2 | merge commit の `git log --name-only` が、merge の中での衝突解決のパスを出さない | `--cc` を足した。git 2.49.0 の scratch repo の 7 つの形(新しい内容で解決、解決と別の編集、upstream の内容、`--ours`、衝突なし、merge のあとの通常 commit、rebase の形)で、doc の文と合うことを確かめた。直っている |
| 3 | `origin/HEAD` が消えた ref を指すと、`git log ... --not <ref>` が毎周期失敗して前の HEAD が据え置かれる | `namesCommit` で指す先を確かめ、通らなければ main、master、最後に `git diff` に落とす。直っている。残りの穴(M48)は tech-debt の (6) |
| 4 | recurred を足したあと status を保存する前に止まると、立て直した watch が 2 回数える | escalate の打ち直しは `alertItemWithRecurredID` で直った。立ち上がり側に足した同じ確認は本番では当たらなかった(`C2-1`) |

| ID | 指摘 | 処理 |
| --- | --- | --- |
| C2-1 (LOW) | `recordRecurrence` の alert_id の確認は、周期ごとに変わる `now` から alert_id を作るので、止めた時計のテストだけで当たる。doc は「status を保存する前に止まった watch の再実行を見分ける」と書いていた | `5393a316` で確認とテストを消し、doc を escalate の打ち直しだけに合わせた。解消。残る窓は tech-debt の (7) |
| C2-2 (LOW) | 活動の判定を `leaderActedSinceAlert` に移したあとも、`checkDeadman` のものとして書いた comment が 4 か所に残った | `5393a316` で 4 か所を直した。解消 |
| C2-3 (LOW) | 4 つのテストの comment が、`/cross-review` がその場で書き直す triage を、番号(「cross-review finding N」)だけで指す | 4 件のうち 3 件を直し、1 件は C2-1 で消えた。折り返された 1 件が残った(`C2b-1`、`3bb1828b` で直った) |
| C2-4 (LOW) | `foldIntoPendingAlert` の引数に string が 5 つ並び、`orgID` と `org` が近い | `5393a316` で署名を 4 つに減らした。解消 |

### cycle 1 の 1 回目(commit 27a76047、HEAD `0b2c0bce`)

CRITICAL・HIGH なし、MEDIUM 1、LOW 6。`f4f1fb93` と `880a3fa0` で処理し、2 回目で現物を読んで確かめた。

| ID | 指摘 | 処理 |
| --- | --- | --- |
| M1 (MEDIUM) | `watchPendingAlert.LastRecurredTS` が書くだけで読まれず、件の「最後の時刻」が escalate の時刻になる | 欄を消し、コメントを実装に合わせた。解消。残り: `show` の `(last <ts>)` が pending のあいだの再発では escalate の時刻になることを、利用者に見える所に書いていない(`/sync-docs` に渡した) |
| L1 | commit の一覧の失敗で `ScopeHead` を進め、その周期の commit パスを失う | 失敗かつ git が前の HEAD を持つなら据え置く(`scopeHeadKnown`)。解消。残り: 失敗が続くと stderr に何も出ない。porcelain の失敗を黙って飛ばす既存の作りと揃っている |
| L2 | 署名の差で、パスが減ったときと `git add` でも立つ | 「前の周期に無かったパスか、予約の外を触る commit」に変更。解消 |
| L3 | 計画の Risks は「HEAD が変わった周期だけ」git を打つとしたが、実装は候補の記録がある周期ごとに打つ | 受け入れ。理由は計画の Progress(`880a3fa0`)。本文を変えると承認の digest が変わる |
| L4 | `watch.go` の再発の仕組み約 180 行が 1 つのまとまり | `watch_recurrence.go` に移した。解消 |
| L5 | `busySeats` は `len() > 0` でしか読まれない | `anotherSeatWorking bool` にした。解消 |
| L6 | `GitStatus` と `GitRun` の 2 つの注入口 | 受け入れ。doc が使い分けを書いている |

### cycle 1 の 2 回目(commit 6025f3cb、HEAD `880a3fa0`)

CRITICAL・HIGH・MEDIUM なし、LOW 4。`eb596f49` で 4 件とも直し、3 回目で確かめた。

| ID | 指摘 | 処理 |
| --- | --- | --- |
| N1 (LOW) | `printableScopePath` の doc コメントに U+2028 の文字そのものが入り、escape の例が欠けて見える | バックスラッシュと u2028 の 6 文字の escape にした。解消 |
| N2 (LOW) | `ScopeSeen` の key が `scope_outside_paths_seen` で欄名と食い違い、コメントが main に無い build を指す | 欄を `ScopeOutsidePathsSeen` に改名し、コメントに前の形と今の扱いを書いた。解消 |
| N3 (LOW) | `committedOutside` の項が何を足すか、doc にも計画にも書かれていない | `evaluateReservedScope` の doc に理由を足した。足した文の言い過ぎが N5 |
| N4 (LOW) | `recurred: 1 times (last ...)` | 回数 1 のとき `1 time`。解消 |

### cycle 1 の 3 回目(commit 690e7235、HEAD `eb596f49`)

CRITICAL・HIGH・MEDIUM なし、LOW 1。N1 から N4 はすべて直っていた。

| ID | 指摘 | 処理 |
| --- | --- | --- |
| N5 (LOW) | N3 の直しで足した doc が、commit の項の効き方を言い過ぎている。「commit は、報告済みのパスでも条件を立てる。ALERT の件が open なら、その件の再発になる」と読めるが、`raiseOrClear` は `rec.Active` のあいだ何もしないので、ALERT を出した直後の周期の commit は再発にも ALERT にもならない。再発になるのは、間に静かな周期が入って `Active` が落ちたあとの commit だけ | 挙動は変えず、tech-debt の「Eight gaps」の行の (5) に記録した。doc の文は今も変わっていない(`watch_scope.go` の `evaluateReservedScope`、`which the director should see` の行)。cycle 3 でも、cycle 3 の 4 回目(`14bc17a3` は `watch_scope.go` を触っていない)でも未修正のまま(受け入れ) |

## Findings

<!-- Area recommended values: naming, readability, null-safety,
     exception-handling, security, maintainability -->

| Severity | Area | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- |
| LOW | maintainability | C4-1: 「受信箱がこの alert をすでに持つ」の判定が 2 か所に書かれている。`reconcilePendingAlertsWithInbox` の `held`(件の `AlertID` と `alertItemWithRecurredID`)と、`escalateToInbox` の replay の 2 つの枝(`slices.IndexFunc` と `alertItemWithRecurredID`)。`escalateToInbox` に「記録済み」の 3 つめの形が足されたとき reconcile が追わないと、その形の pending が残って再発を畳み、今回の穴がその形で戻る。今ある 2 つの形は新しいテストの 2 本が固定している | `internal/org/watch_recurrence.go` の `reconcilePendingAlertsWithInbox` の `held` を求める 3 行と、`internal/org/watch.go` の `escalateToInbox` の `if i := slices.IndexFunc(...)` と `if item, ok := alertItemWithRecurredID(...)` | 判定を 1 つの関数に寄せて両方から呼ぶ(例: `itemHoldingAlert(items, alertID) (InboxItem, bool)` で、返した件の `AlertID` が alert_id と等しいかで 2 つの枝を分ける)。寄せないなら、`escalateToInbox` の 2 つの枝の上に「`reconcilePendingAlertsWithInbox` が同じ 2 つの判定を持つ」と書く |
| LOW | readability | C4-2: この commit が書いた comment の 2 か所が、同じ commit の別の comment や実装とずれる。(a) `evaluateCycle` の「(a replay that records nothing new)」。open で `notified` の無い件の replay は人へ送って `notified` event を足す(`sendInboxItemToHuman` の最後が `AppendNotified`)。同じ commit の `reconcilePendingAlertsWithInbox` の doc は、この送りを書いている。(b) `escalateAlert` の doc は、pending を消す意味を「`checkDeadman` が range する集合から消す」とだけ書くが、range するのは reconcile も(2 つめの呼び出し元を足したときに、1 つめの名前が doc に残った形)。「同じ alert を 2 回 escalate しない」は今も正しい | `internal/org/watch.go` の `evaluateCycle` の新しい comment(「a replay that records nothing new」)と、`escalateAlert` の doc の 2 文目(「the set checkDeadman ranges over」) | (a) を「a replay that records no second item or recurrence」の形にする。(b) を「the set checkDeadman and reconcilePendingAlertsWithInbox range over」にする |
| LOW | exception-handling | C4-3: S12 が消す pending の回数について、計画の Progress の S12 の行は「失われる(既知の穴の (2) と同じ向き)」とだけ書くが、そうなるのは W2(件を作って運んだ回数を書く前に止まった)だけ。W3 と W4(回数を件に運んだあとで止まった)では回数は失われず、保存されなかった立ち上がりがもう一度数えられて、件の回数が実際より 1 多くなる。向きは知らせる側(S12 の前は、同じ形で件が resolved のとき再発が黙って失われた)で、直すには保存より前に状態を書く設計が要り、S12 の範囲の外。ただし code の doc にも tech-debt にも書かれていない | 計画の S12 の行(`docs/plans/active/2026-11-12-watch-alert-dedupe.md`)。`escalateToInbox` の `Escalate` と運んだ回数の `AppendRecurred`、`foldIntoPendingAlert` の `Recurrences++`、`evaluateCycle` の周期の終わりの保存(上の W2 から W4) | コードは変えない。W3 と W4 を表のとおり tech-debt の (2) と (7) の近くに足す。計画の S12 の行は記録なので、直すかは `/sync-docs` が決める |

CRITICAL、HIGH、MEDIUM はない。cross-review の指摘は直っている。reconcile が watch を黙らせる道、`checkDeadman` と `escalateAlert` との順序の問題、二重の送りは見つからなかった(Evidence の 3 から 5)。残る黙る向きは W9 の 1 つで、周期の中の一時的な読み失敗が要る。

## Positive notes

- 直しは、単に pending を消さず `escalateAlert` を通す形にした。消すだけだと、2026-10-23 の D4 の「記録したが人へ送っていない件を送り直す」判定が走らない。doc がその理由を書いている
- 関数は短く、受信箱が無いとき、pending が無いとき、読めないときの 3 つの早期 return が先に並ぶ。held の判定は件の `AlertID` と recurred の `alert_id` の 2 つで、key ではなく alert_id で見るので、同じ key の別の episode の pending を巻き込まない(テストが固定)
- テスト 4 本が、指摘の形(件が resolved か open か)の両方を、stderr の 1 行、件の数、通知の数、`Escalated`、pending の alert_id まで固定している。「持たない pending は残す」と「読めなければ何もしない」の否定側も 1 本ずつ置いた
- doc を同じ commit で 3 か所(`evaluateCycle`、`escalateAlert`、`foldIntoPendingAlert`)に合わせた。`foldIntoPendingAlert` の doc は、reconcile のあとに held の pending が残るのは受信箱が読めなかったときだけと書き、テストと合う

## Tech debt identified

この run は `docs/tech-debt/README.md` を編集していない(前の run と同じく、`/sync-docs` に渡す)。この run は cap 3 の最後なので、`C4-1` と `C4-2` を直さなければ deferral になる。直すと comment とコードの変更で full re-run になり、cap をもう一度上げることになる。この review は、直さずに次の 1 行にまとめて register に置くことを勧める。

| 場所 | 状態 | 直し方 |
| --- | --- | --- |
| 「Eight gaps」の行の (9)、または別の行 | 止まる窓の W1(`sendAlert` の送りのあとで止まると 2 通目の ALERT)、W3 と W4(+1)、W7(`escalateWithoutInbox` のあとで止まると 2 回目の通知)、W9(読めない周期で黙る向きの極小の残り)が、register のどの行にも無い。W2 は (2)、W5 は (7) に既にある | 上の表を W1 から W9 のまま写し、(2) と (7) から W3 から W5 を指す。Trigger は、次に `escalateToInbox`、`recordRecurrence`、`reconcilePendingAlertsWithInbox`、`sendAlert` を変えるとき |
| `reconcilePendingAlertsWithInbox` と `escalateToInbox` の held の判定(C4-1) | 同じ判定が 2 か所 | 次に `escalateToInbox` の「記録済み」の枝を変えるとき、1 つの関数に寄せる |
| `evaluateCycle` と `escalateAlert` の comment(C4-2) | 「records nothing new」と「checkDeadman が range する集合」が 2 つめの呼び出し元に合っていない | 次にこの 2 つの doc を直すとき |

## Recommendation

- Merge: yes。CRITICAL、HIGH、MEDIUM はない。cross-review の指摘は `14bc17a3` で直っている(コードを読んで追った。テストは走らせていない)。LOW が 3 件(`C4-1`、`C4-2`、`C4-3`)で、どれも黙る向きではない。cycle 1 の `M1`、`L1` から `L6`、`N1` から `N4`、cycle 2 の `C2-1` から `C2-4`、`C2b-1`、cycle 3 の `C3-1`、`C3-2` は解消のまま、`N5` は受け入れのまま
- Follow-ups:
  - `/sync-docs` は、上の「止まる窓の一覧」を tech-debt に写す(依頼どおり)。あわせて「Eight gaps」の行の cycle の段落に S12 の段を足す。Trigger は `watch.go` の `evaluateCycle`、`escalateAlert`、`foldIntoPendingAlert` の doc と `watch_recurrence.go` で当たり直した。(1) から (3) と (5) から (8) は払っていない。行の「cycle 3 of 3」「its third run went to S10 and S11」は S10 と S11 については今も正しいが、最後の run は 4 回目(S12)になった
  - 計画の S12 の行は「回数が失われる」だけなので、W3 と W4 の +1(`C4-3`)を `/sync-docs` か `/pr` の Known gaps に足す
  - `C4-1` と `C4-2` は、直す場合は full re-run になる。直さないなら上の register の行に置く
  - 新しいテスト 4 本は、この run の `/verify` と `/test` が回る前の木に入っている。実行結果とコンパイル、gofmt の整列は `/verify` と `/test`
  - この表の W3 と W4 の +1 は、コードを読んで追っただけで、再起動を模したテストでは確かめていない。`/test` で見るなら、W3 の形(止まった周期で畳んだ firing が、再起動後にもう一度立つ)の 1 本になる
