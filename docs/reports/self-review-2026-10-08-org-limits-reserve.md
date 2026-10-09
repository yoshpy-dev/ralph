# Self-review report: org-limits-reserve

- Date: 2026-10-09 JST(ファイル名は計画の日付)
- Plan: docs/plans/active/2026-10-08-org-limits-reserve.md
- Branch: feat/org-limits-reserve(HEAD 75b009f3、base 51855166)
- Reviewer: reviewer subagent (Claude)、パイプライン 4 回目(cycle 4)。ユーザーが上限を 4 に上げた回で、`cycle-count.json` は /cross-review の上限引き上げの決まりで 2 のままなので、この報告と insight event に cycle 4 と明記する(`insights-append.sh --cycle 4`)
- Scope: diff の品質だけ。cycle 3 の self-review(f19df087)以降の差分 `git diff f19df087..HEAD -- ':!docs'` の 6 ファイル(+407/-39)が対象。コードは `internal/org/verbs.go` と `verbs_test.go` の 2 つで、中身は 83aec44e(`reserveAgain` を `releasedReservation` と `startsOrg` で書き直し、pane の経路からも呼ぶ)と、cycle 3 の `/test` が足した 066bf282(テスト)。残りの 4 ファイルは cycle 3 の `/sync-docs`(736ce19d)が直した `/org` skill の 4 面。重点は、Step A(close の前の写しで、戻す予約を決める)と Step B(ロックの下の読み直しで、書くかを決める)が別の run の予約を戻さないか、`len(now) <= d || now[d] != before[d]` の判定、pane の経路が予約を書くようになって通常の `stop` が何も書かないままか、`reservationBeforeLastDisband` が本番で使われているか、doc comment とコードの一致、`ActiveReservation(now) != nil` の位置づけ。仕様への適合、テストの網羅、文書のずれは見ていない(`/verify`、`/test`、`/sync-docs` の担当)。テスト、linter、型検査、formatter、mutation は実行していない
- 番号の付け方: cycle 1 の finding は F-1〜F-11(付録 A に原文)、cycle 2 は C2-1〜C2-7(付録 B に原文)、cycle 3 は C3-1〜C3-4(付録 C に原文)、この回は `C4-` を付ける。tech-debt の行 170〜179 と verify・test・plan が F、C2、C3 の番号で指しているので、報告を上書きしても番号は付け替えない

## Evidence reviewed

- `git diff f19df087..HEAD -- ':!docs'` の全行。親子は `git log --format='%h parent=%p'` で確かめ、一直線(f19df087 → 8b83abeb → 066bf282 → 104be58d → 736ce19d → 7981e8fb → 83aec44e → 75b009f3)。cycle 3 の `/verify`・`/test`・`/sync-docs` の報告と skill の編集(736ce19d)は 83aec44e より前の tree に対するもので、83aec44e はまだどの検査も通っていない(この報告も読みだけ)
- 全文を読んだ関数: `reserveAgain`、`releasedReservation`、`startsOrg`、`compensateUnderLock`、`reactivateSeat`、`reopenWorkspace`、`CloseDeferredSelfPane`、`CloseDeferredSelfWorkspace`、`Stop`、`Disband`、`disbandOwnLast`、`closeOrgWorkspace`、`lastDisbandedIndexes`、`currentOrgLives`、`ActiveReservation`、`Roster`、`closeDeferredSelf`(`internal/cli/org.go`)。`reserveAgain` を呼ぶのは `verbs.go:1048`(pane)と `:1136`(workspace)の 2 か所
- 呼び出しの確認: `grep -rn 'reservationBeforeLastDisband\|releasedReservation\|startsOrg\|reserveAgain(' internal cmd`。`reservationBeforeLastDisband` は定義(`reserve.go:311`)とテスト(`reserve_test.go:323`)にしかない
- `ActiveReservation(now) != nil` の冗長さは、`currentOrgLives` の条件(real、org 単位、最後の `disbanded` より後の `scope_reserved`)と `startsOrg` の条件(real、同じ org_id、`scope_reserved` を含む)を並べて確かめた。前者の集合は後者に含まれる(下の C4-2)
- 台帳を書き換える本番の経路: `internal/org` で `os.Rename`、`os.WriteFile`、`O_TRUNC` を grep した。`manifest.jsonl` を作り直すものはない(`report.go:59`、`spawn.go:1642`、`watch.go:310` は別のファイル)
- 追加テストの全文: pane 側(`ReservationOnlyAfterDisband` の 3 ケース)、workspace 側(`a later run without --reserve`、`ReservationUnlessStartedAgain` の 5 ケース、`ReservationAppendFails_NamesIt`)。helper(`deferOwnLeaderWorkspaceCloseFailing`、`spawnLeaderIn`、`ownPaneEnv`、`closeHookHerdr`、`eventsWithDetailsPrefix`、`assertRejectedRecorded`)は既存
- 機械的な確認: `git diff --check f19df087..HEAD` は空。追加行に U+FFFD、`fmt.Print`、`println`、TODO、FIXME、finding の ID(`AR#`、`C3-`、`WC#`)、cycle 番号の引用はない。コミットメッセージに attribution の行はない。skill の 4 面は `cmp` で同一で、追加した文に `internal/` や `docs/reports` などの出荷物に出せない語はない
- tech-debt の行 172、179、plan の進捗(159〜161 行)、cycle 3 の `/test` 報告の G4・G5・Q1・Q2、cross-review の triage(cycle 3)を HEAD の内容と突き合わせた

## 依頼された点の結論

1. Step A と Step B が別の run の予約を戻すか: 通常の入力では戻さない。下の表のとおり、`disbanded` の後ろに立ち上げ(`scope_reserved`、`spawn_started`、`spawned`)があれば Step A が何も返さず、待つ間に立ち上げがあれば Step B が止める。`releasedReservation` は「直前の立ち上げの時点で `ActiveReservation(events[:i+1])` が何を持っていたか」を見るので、前の run の予約は、後の run が予約なしで始まれば引き継がれない(最後の `disbanded` が `events[:i+1]` の中で窓を切る)。戻しうる入力が 1 つ残る。すでに止まっている座席への `stop` の打ち直し(C4-1)

   | 入力 | Step A(close の前の写し) | Step B(ロックの下の読み直し) | 結果 | テスト |
   | --- | --- | --- | --- | --- |
   | disband の自分の close が失敗しただけ | その run の予約 R | 通る | R を戻す | `ReservationOnlyAfterDisband` の 1 つめ、`..._ReservationRestored` の `restored` |
   | 写しの時点で `disbanded` が 2 つ続く(打ち直しのあと) | 最後の `disbanded` の直前の立ち上げの時点の R | 通る | R を戻す | `disband run again before the close read the manifest` |
   | close を待つ間に同じ org の `disband` が打ち直された | R | `now[d]` は同じで、後ろに立ち上げなし | R を戻す | `only disband run again` |
   | 待つ間に `--reserve` 付きで立て直された | R | `scope_reserved` と `spawn_started` | 戻さず skip を報告 | 既存の `org started again with another reservation` |
   | 待つ間に `--reserve` なしで立て直された | R | `spawn_started` | 戻さず skip を報告 | `started again without --reserve` |
   | 写しの時点ですでに `--reserve` なしで立て直し済み(pane) | 何も返さない(`disbanded` の後ろに立ち上げ) | 呼ばれない | 何も書かず、何も報告しない | `ordinary stop in a run started again without --reserve` |
   | 前の run が予約し、次の run は予約なしで、その run を disband | 何も返さない(直前の立ち上げの時点で予約なし) | 呼ばれない | 書かない | `a later run without --reserve` |
   | 一度も disband していない org の通常の `stop` | 何も返さない(`disbanded` なし) | 呼ばれない | 書かない | `ordinary stop in an org never disbanded` |
   | 台帳が消えた、または前に 1 イベント足して置き換えられた | R | `len(now) <= d`、`now[d] != before[d]` | 戻さず skip を報告 | `manifest removed`、`manifest replaced` |
   | `disband --force` のあと、止まっている座席に `stop` を打ち直し、close が失敗 | R(`stopped` は立ち上げでない) | 通る | R を戻す(C4-1) | なし |

2. `len(now) <= d || now[d] != before[d]`: 短絡評価の順で、添字の範囲外を踏まない。台帳が `before` の接頭辞でなくなったときは戻さない側に倒れる。本番にこの状態を作る処理は、上の grep のとおりない(`before` と `now` の間で台帳を書き換えるのは `appendEvent` だけで、台帳は伸びるだけ)ので、守るのは手で編集された台帳だけになる。2 つのテストが半分ずつ固定する(`manifest removed` は `len`、`manifest replaced` は `!=`)。finding にしない。`now[d] != before[d]` は、添字 d のイベントが同じかを構造体の比較で見る。台帳が伸びるだけなら常に等しいので、置き換えを検出する目的にだけ働く
3. pane の経路が予約を書くようになったあとの通常の `stop`: 動いている座席の通常の `stop` は何も書かない。`Roster` は `disbanded` より前の状態イベントしか持たない座席を非 active にするので、`disbanded` の後ろで動いている座席は、その後ろに `spawn_started` か `spawned` を持ち、`releasedReservation` は何も返さない(spawn の途中に `disband` が入った競合で `spawn_step` だけが後ろに来る場合は例外で、読みによる推測)。表のテストが 2 つの入力で固定する。通常の `stop` が予約を書くのは、止まっている座席に対する `stop` の打ち直し(C4-1)
4. `reservationBeforeLastDisband`: 本番で使われていない(C4-3)
5. doc comment とコード: `compensateUnderLock` の追加 2 文、`reserveAgain`、`releasedReservation`、`startsOrg` の doc は規則とコードが合う。合わない所は、C4-1(doc の言い切り)、C4-2(独立した条件としての記述)、C4-4(4 点)
6. 実装者のメモ(`ActiveReservation(now) != nil` は他の条件が偽のとき常に偽): 正しい。式は残っていても結果を変えず、doc が独立した条件として書くので、読み手を誤らせる(C4-2)

## Findings

<!-- Area recommended values: naming, readability, unnecessary-change, typo,
     null-safety, debug-code, secrets, exception-handling, security, maintainability -->

| Severity | Area | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- |
| C4-1 LOW | maintainability | pane の経路の doc の「after an ordinary stop the org has no `disbanded` after its latest start」は、止まっている座席への `stop` の打ち直しでは成り立たない。`Stop` は台帳に座席があれば状態を見ずに進み(`seatFromEvents` のあと `Active` を見ない)、止まっている座席でも pane が自分のものなら deferred にする。たとえば `disband --force` で自分の pane の close が失敗すると、座席は `stopped`、org は `disbanded` のまま、pane のプロセスは生きている(`--force` は補償しない)。その pane から `ralph org stop --seat <自分>` を打つと、`stopped` が `disbanded` の後ろに足され、close がまた失敗すれば、`reactivateSeat` が座席を戻し、`reserveAgain` は `disbanded` の後ろに立ち上げがない(`stopped` は `startsOrg` に入らない)と見て、forced disband が解いた予約を書き戻す。先のコマンドの `disband` が解いた予約を、後のコマンドの補償が戻す形になる。`releasedReservation` は「最後の `disbanded` がこのコマンドのものか」を「その後ろに立ち上げがないか」で代用していて、`stop` の打ち直しでこの代用が外れる。座席が動いている扱いに戻るので、予約も戻す結果は筋が通り、誤動作とは言えない。誤りは doc の言い切りと、この入力のテストがないこと。コードを読んだ結論で、実行はしていない | `verbs.go:1010-1012`(doc)、`:844`(`Stop` の座席の引き方)、`:1047-1048`、`:1348-1359`、`verbs_test.go:2899`、`:2903`(表の 2 ケースはどちらも `disbanded` が最新の立ち上げの後ろにない)、`Disband`(`disbandOwnLast` が `stopped` を書いてから `appendDisbanded`) | 戻す挙動を残すなら、doc を「after an ordinary stop of an active seat」に絞り、打ち直しの扱いを 1 文書いて、3 つめのケース(`disband --force` のあとの `stop` の打ち直し)を表に足す。戻さない挙動にするなら、pane の経路だけ「座席の `stopped` が `disbanded` より前」を条件にする(`Disband` は座席の `stopped` を `disbanded` より先に書く。`lastSeatOnPane` が返すイベントの添字を `reserveAgain` に渡す)。最後の回なので、どちらも tech-debt に送る |
| C4-2 LOW | maintainability | `reserveAgain` の `ActiveReservation(now, orgID) != nil` は、同じ条件式の `startsOrg` の走査に含まれ、単独では結果を変えない。`ActiveReservation(now)` が nil でなくなるのは、real で org 単位の `scope_reserved` が `now` の最後の `disbanded` より後にあるとき。`now[d] == before[d]` が真なら、`now` の最後の `disbanded` の添字は d 以上なので、その `scope_reserved` は `now[d+1:]` に入る。`startsOrg` は `scope_reserved` を含むので、走査が先に真になる。実装者のメモのとおり、この式だけが真になる入力はない。それでも doc(`:1313-1314`「and gives the org no reservation」)は独立した 3 つめの条件として書き、読み手は予約の有無を別に確かめていると読む。cycle 3 の `/test` 報告の G4(この式を外す: red)と G5(`slices.Equal` を外す: red)、これを引く tech-debt 172 の(a)の閉じ方(「dropping either condition ... turns them red」)は 83aec44e の前の式についての記録で、HEAD では G5 の式がなく、G4 の式は外しても落ちるテストがない(読みによる) | `verbs.go:1330`、`:1313-1314`、`:1364-1373`(`startsOrg`)、`reserve.go:187-224`(`currentOrgLives`。`:215` が `scope_reserved` の枝)、`docs/reports/test-2026-10-08-org-limits-reserve.md:63-64`、`docs/tech-debt/README.md:172`(a) | 式と doc の「and gives the org no reservation」を外し、`now[d+1:]` の走査に任せる。残すなら「implied by the scan below」と書いて、172 の閉じ方を HEAD の条件(`startsOrg` の走査と、`started again without --reserve`、`org started again with another reservation` の 2 ケース)に書き直す。最後の回なので tech-debt に送る |
| C4-3 LOW | maintainability | `reservationBeforeLastDisband` は本番の呼び出しがなくなった。呼ぶのは `TestReservationBeforeLastDisband` だけで、そのテストの「held none in its last life」(`[予約, disbanded, disbanded]` に nil を期待する)は、83aec44e が変えた規則(続けて打った `disband` は前の予約を放さない)と逆の答えを守っている。同じ入力を `releasedReservation` に渡すと R が返る。同じ入力に逆の答えを返す 2 つの関数のうち、古い方だけがテストを持ち、新しい方(`releasedReservation`、`startsOrg`)は `verbs_test.go` の補償のケースを通してしか動かない。「解いた予約」を引く次の変更が古い方を使うと、83aec44e が直した欠陥(打ち直しの `disband` で予約が戻らない)が戻る | `reserve.go:309-317`、`reserve_test.go:306-328`(`:318`)、`verbs.go:1348-1359` | 関数とテストを消し、`releasedReservation` の表のテストを `reserve_test.go` に足す。ケースは、予約のある run が 1 つ `disbanded` で終わる、`disbanded` が続く、`disbanded` の後ろに立ち上げがある、予約なしの run が後に続く、`disbanded` の前に立ち上げがない、の 5 つ(純粋な関数なので、台帳も fake も要らない)。最後の回なので tech-debt に送る |
| C4-4 LOW | readability | 83aec44e の comment に小さい残りが 4 つある。(a) `CloseDeferredSelfWorkspace` の doc の 1 行(`verbs.go:1102`)が 41 桁で、前後は 70 桁台。挿入のあとの再整形の漏れ。(b) `CloseDeferredSelfPane` の doc で、予約の文(`:1007-1012`)が「the error says the seat is recorded active again」と「It decides that from the manifest read again under the manifest lock」の間に入り、`that` の指すものが座席の補償から予約を含む全体に変わる。続く「when the seat has a newer state event ... it appends nothing」は座席だけを書き、予約の skip は「the same rule as in CloseDeferredSelfWorkspace」の 1 か所に任せる。(c) `compensateUnderLock` は `compensate` に台帳を 1 回だけ読んで渡し、`reserveAgain` はこれに依存する。`reactivateSeat` が足す `spawned` は `startsOrg` に数えられるので、各ステップが自分の append のあとで読み直すと、`reserveAgain` は座席の復帰を「立て直された」と読んで skip する。この前提(同じ呼び出しの append は `now` に入らない)はどの doc にもなく、`reserveAgain` の「now, the manifest read under the lock」は一度読んだ写しだと言わない。座席と予約を一緒に戻す 3 つのテストが間接的に固定する。(d) 「`d` のあとに org が立ち上げられたか」の式(`slices.ContainsFunc(..., startsOrg)`)が `:1331` と `:1350` に同じ形で 2 つあり、Step A と Step B が同じ規則であることが 2 つのコピーの一致でしか保たれない | `verbs.go:1102`、`:1007-1013`、`:1206-1221`(`compensateUnderLock` の doc)、`:1130-1136`(呼び出しの順)、`:1331`、`:1350` | (a) 再整形。(b) `that` を名詞に直し、予約の文を段落の末へ移す。(c) `compensateUnderLock` の doc に「compensate gets one snapshot, read before any of its appends; reserveAgain relies on it, since the `spawned` that reactivateSeat appends counts as a start」を足す。(d) `startedAfter(events []ManifestEvent, d int, orgID string) bool` に切り出す。コードの変更を含むので、最後の回は tech-debt に送る |

その他に数えない指摘が 3 つある。(1) `releasedReservation` の戻り値 `d` は、`paths` が nil のとき 0 で、先頭の添字と区別できない。呼び出しの `reserveAgain` は `paths == nil` を先に見るので正しく、doc も `paths` が nil の場合を列挙している。(2) `startsOrg` の名前は、立ち上げ(`spawn_started`)と継続(`spawned`、補償の `scope_reserved`)の両方に当たるが、doc がそれを書いている。(3) 待つ間に同じ org の座席に `rejected` だけが足された入力では、座席は `current != seat` で戻さず、予約は `startsOrg` に `rejected` が入らないので戻す。動いている座席のない org が予約だけを持つ形になる。`rejected` が書かれるのは spawn の限度や重なりの判定が落ちたときで、予約を戻す側がそれらを見ない点は行 172 の(d)の既存の窓と同じ。読みによる推測で、この入力のテストはない。

## cycle 3 の finding の現況

| C3 | 重さ | この回の判定 | 根拠 |
| --- | --- | --- | --- |
| C3-1 | LOW | 未修正(行 179 の(a)) | `compensateUnderLock` の doc は予約の規則を 2 文足したが、先頭の「appends only what is still missing」(`verbs.go:1211-1212`)は残り、ロックの外の書き込みが窓に入ることも書かない |
| C3-2 | LOW | 未修正(行 179 の(b)) | `selfCompensation`、`add`、`errorFor` の doc は 83aec44e の差分にない |
| C3-3 | LOW | 未修正(行 179 の(c)) | `internal/cli/org.go` の `stop` と `disband` の help(`:826-833`、`:1140-1151`)、`closeDeferredSelf` と `Disband` の doc は差分にない。pane の経路が予約も戻すようになったので、help が `disband` の補償に予約を挙げない点は workspace の経路だけでなく pane の経路にも当たる |
| C3-4 | LOW | 直った | (a)`--reserve` なしの立て直しは古い予約を戻さない(Step A の `startsOrg` と Step B の走査。`started again without --reserve`)。(b)`before` にすでにあった立て直しは、Step A が何も返さないので「while the close waited」と報告しない。規則の差が出ていた 2 つの入力(`disband` の打ち直し、`--reserve` なしの立て直し)は、座席、workspace、予約で同じ向きになった。残る差は「その他」の(3) |

cycle 1 の F-10(pane の経路が予約を戻さない)は 83aec44e で直った(`verbs.go:1048`)。行 172 の(b)は閉じられる。cycle 2 の C2-1〜C2-3 は行 177 に載ったまま(`spawn.go` は差分にない)。

## Positive notes

- 83aec44e は、cycle 3 の cross-review が挙げた 2 つの入力(`disband` の打ち直し、pane の経路)に加えて、C3-4 の 2 つの入力も 1 つの規則で解いた。座席、workspace、予約の 3 つの補償は、待つ間に同じ org が立て直された入力で同じ向きに揃った(決め方の違い、座席は自分の最新の状態イベント、予約は org の立ち上げの有無、は残る)。`ActiveReservation(events[:i+1])` の窓の切り方も正しい(`events[:i+1]` の中の最後の `disbanded` で前の run が切れる)
- ロックの扱いは cycle 3 のまま。`releasedReservation` と `reserveAgain` は herdr も `withManifestLock` も呼ばず、台帳への書き込みは `appendEvent` の 1 回だけ。Step A はロックの外(`before` の写し)、Step B はロックの中(`now`)で、入れ子はない
- 追加テストは、外すと落ちるケースを持つ(読みによる。mutation は回していない): `started again without --reserve` は cycle 3 の式(`ActiveReservation(now)` が nil で、`disbanded` の前の予約が同じなら書く)に戻すと落ち、pane の表の 2 ケース目は `releasedReservation` の「`disbanded` の後ろの立ち上げ」の走査を外すと落ち、`a later run without --reserve` は `events[:i+1]` で窓を切らずに `d` の前の最後の `scope_reserved` を取る形にすると落ち、`manifest removed` と `manifest replaced` は判定の半分ずつを固定する
- pane の表のテストは `disband` の経路と通常の `stop` の経路を同じ fake で通し、エラー文の末尾(`HasSuffix`)とイベント名の並び(`[spawned, scope_reserved]`)の両方を見る。復元した予約が他の org の spawn を実際に拒む(`assertRejectedRecorded`)ところまで確かめる
- `reserveAgain` の doc は、限度と重なりを見ない窓を残している(行 172 の(c)(d)と一致する)。skill の 4 面は同一。ただし、予約を戻すのは workspace の close の失敗だけと読める文が 1 か所残る(下の `/sync-docs` の項)

## Coverage gaps

- テスト、mutation、gofmt、vet、`check-sync.sh`、`check-skill-sync.sh` は実行していない(指示による)。83aec44e は cycle 4 の `/verify` と `/test` が初めて通す。上の「外すと落ちる」「外しても落ちない」の結論は読みによるので、`/test` に次の 3 つの mutation を頼む: `ActiveReservation(now) != nil` を外す(落ちなければ C4-2 の裏づけ)、`startsOrg` の `EventScopeReserved` の枝を外す、`releasedReservation` の `slices.ContainsFunc(events[d+1:], ...)` を外す(pane の表の 2 ケース目が落ちるはず)
- `startsOrg` の `EventScopeReserved` の枝は、`--reserve` 付きの立て直しが `spawn_started` も書くので、この枝がなくても `--reserve` の立て直しのテストは通る(読み)。違いが出るのは `scope_reserved` だけが `disbanded` の後ろに足された並び(補償の `reserveAgain` だけが書いた場合など)で、その並びのテストは見当たらない
- `releasedReservation` の `i` を逆順に探すループが何も見つけない枝(最後の `return 0, nil`。`disbanded` があるのに、その前に立ち上げがない台帳)にテストはない。古い ralph の台帳で届く可能性がある。何も返さない側に倒れるので害はない
- C4-1 の入力(`disband --force` のあとの `stop` の打ち直し)のテストはない。fake herdr で組めるので、`/test` が実際の結果を確かめられる
- ロックのケースは同じプロセスの goroutine で、別プロセスの競合は見ていない。本物の herdr で補償の窓を作った実測はない(cycle 1 から同じ。行 171 の(c)、行 176 の(e))
- skill の 4 面と help 文は読んだだけで、`ralph org stop --help` と `disband --help` は出していない

## Tech debt identified

この回は、ユーザーが上限を 4 に上げた最後の回なので、C4-1〜C4-4 は直さなければ繰り延べにあたる。4 件ともコードのコメント、式 1 つ、未使用の関数とテストで、直すとパイプラインの再実行になる。行 179(cycle 3 の LOW)の続きとして 1 行にまとめる案を次に書く。足すのは `/sync-docs` の担当か、オペレーターの判断。

> org-limits-reserve cycle 4 の LOW(self-review の C4-1〜C4-4)。(a) pane の経路の doc の「after an ordinary stop the org has no `disbanded` after its latest start」が、止まっている座席への `stop` の打ち直し(`disband --force` で自分の pane の close が失敗したあと)では成り立たず、その補償が先の `disband` の予約を戻す。戻す挙動のまま doc を絞ってテストを足すか、座席の `stopped` が `disbanded` より前のときだけ戻す形にするかを決める。(b) `reserveAgain` の `ActiveReservation(now, orgID) != nil` は `startsOrg` の走査に含まれ、結果を変えない。式と doc の「gives the org no reservation」を外し、行 172 の(a)の閉じ方(G4、G5)を HEAD の条件に直す。(c) `reservationBeforeLastDisband` は本番の呼び出しがなく、そのテストは `releasedReservation` と逆の答えを守る。関数とテストを消し、`releasedReservation` の表のテストを足す。(d) 83aec44e の comment の残り(`verbs.go:1102` の再整形、`CloseDeferredSelfPane` の `that`、`compensateUnderLock` の「同じ呼び出しの append は `now` に入らない」の前提、`startedAfter` への切り出し)。トリガー: `reserveAgain`、`releasedReservation`、`startsOrg`、`compensateUnderLock` の次の変更、または補償が戻した予約が先のコマンドの `disband` の結果と食い違った報告

C3-1〜C3-3(行 179 の(a)〜(c))は未修正のまま、同じ行が持つ。

## /sync-docs に渡すもの

文書のずれは見ていないので、位置と 83aec44e との関係だけを書く。83aec44e は cycle 3 の `/sync-docs`(736ce19d)のあとに入ったので、次の行と文は修正前の挙動を書いている。

- 行 172 の(b)(F-10): 83aec44e が直した。`CloseDeferredSelfPane` が `reserveAgain` を呼ぶ(`verbs.go:1048`)。(b)を取り消し線にし、見出しの「has two gaps left, (b) and (d)」を数え直し、Trigger の「(b): call `reserveAgain` from `CloseDeferredSelfPane`」を外す。「the doc of `reserveAgain` names only the workspace path」も直った。(a)の閉じ方(RESOLVED の括弧)は、a94c914f の式(`ActiveReservation(now)` と `slices.Equal`)と G4・G5 を挙げるが、HEAD の式は 83aec44e で替わった(C4-2)。HEAD の条件とテストに書き直す
- 行 179 の(d)(C3-4、V3-1): 83aec44e が直した。`disband` の打ち直しは 3 つとも戻し、`--reserve` なしの立て直しは座席も予約も戻さず、`before` にあった立て直しは黙る。テストは `..._RulesDiffer` から `..._ReservationUnlessStartedAgain` に改名された。(d)と、Trigger の「(d) Decide the rule first」、Related の「Mutation ... Q1, Q2」の語を外す。(a)〜(c)(C3-1〜C3-3)と(e)(T3-4〜T3-7)は残る。行の冒頭の「Wording findings and test gaps」の括弧の「C3-1 to C3-4」は「C3-1 to C3-3」になる
- `/org` skill の 4 面(`.claude/skills/org/SKILL.md:192` と 3 つの写し): 「workspace の close が失敗したときは `disbanded` の前の予約も戻す」は、`disband` が自分の pane を deferred にして close が失敗した場合も戻すようになったので、「pane でも workspace でも」の趣旨に直す。`stop` の行は、通常の `stop` が予約を戻さないままなので変えなくてよい(C4-1 の入力は別)
- plan: 進捗の 161 行目「(a) pane だけを後回しにした disband の補償は、予約を戻さない」は 83aec44e で解けた。160 行目が 83aec44e を書くので、161 行目に「83aec44e で解決」の 1 文を足せば足りる(進捗の行は approval digest に入らない)
- cycle 3 の `/test` 報告の G4・G5・Q1・Q2 と `RulesDiffer`、`/verify` の V3-1 は fc35fee6 時点の tree の記録で、cycle 4 の `/verify` と `/test` が HEAD で取り直す。ここでは判定しない
- 行 179 に C4-1〜C4-4 の上の案を足す

## Recommendation

- Merge: 可。CRITICAL、HIGH、MEDIUM はない。LOW は 4 件(C4-1〜C4-4)で、doc の言い切り 1 件、結果を変えない式 1 つ、未使用の関数とそのテスト、comment の残り。83aec44e は、cycle 3 の cross-review の 2 件(`disband` の打ち直し、pane の経路)と C3-4 の 2 入力を 1 つの規則でそろえた。別の run の予約を戻しうる入力として、止まっている座席への `stop` の打ち直しが 1 つ残る(C4-1)
- Follow-ups: C4-1〜C4-4 は最後の回なので直さず、上の 1 行にまとめて tech-debt へ送るのが現実的。C4-1 は挙動を決める必要があるので、送る前にオペレーターが「戻す」か「戻さない」かを選ぶとよい。`/test` に C4-1 の入力のテストと、上の 3 つの mutation を頼む。`/sync-docs` に渡すもの: 行 172(b)、行 179(d)、skill の 4 面の 1 文、plan の 161 行目
- cycle 3 の C3-4 と cycle 1 の F-10 は直り、C3-1〜C3-3 と cycle 2 の C2-1〜C2-3 は行 179 と 177 に載ったまま

## 付録 A: cycle 1 の finding、Coverage gaps、Tech debt(d8ec84da 時点の原文)

cycle 1 の self-review の本文のうち、tech-debt の行 172〜176 と verify、計画が F 番号と Coverage gaps の名前で指している部分を、そのまま残す。`file:line` は d8ec84da 時点のもので、`spawn.go` はこの回の差分で行がずれている(関数名で探す)。cycle 1 の `Recommendation` の節は、この回の判定と食い違わないように載せない。

### Findings

<!-- Area recommended values: naming, readability, unnecessary-change, typo,
     null-safety, debug-code, secrets, exception-handling, security, maintainability -->

| Severity | Area | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- |
| F-1 MEDIUM | maintainability | scaffold が配る `templates/base/ralph.toml` の説明と、`config.go` の doc comment が「the director is not a seat」と書く。director は `docs/specs/` と計画にしかなく、scaffold した project にも、このリポジトリの出荷物(README、skill、rules、コード)にも存在しない語で、読み手は何のことか分からない。計画の Non-goals も「director はまだなく」と書いていて、まだない機能について「数えない」と断る文になっている。director が入る 8 段目で数え方が変わると、この一文が先に古くなる | `templates/base/ralph.toml:46`、`internal/config/config.go:53`、`:58`。出荷物での `director` の語(単語境界)の grep は、この 3 か所だけ | 3 か所とも「The director is not a seat and is not counted」の文を消す(`max_orgs` は走っている org、`max_total_seats` は動いている座席、で完結している)。director を数える・数えないは、director を入れる段の計画で書く。scaffold の `ralph.toml` なので、`/verify` の前に直せば cycle は増えない |
| F-2 LOW | readability | `spawn.go` の中で `reserve` が 2 つの型を指す。入力検査(`:437`)では `NormalizeReservePaths` の結果の `[]string`、dry-run の分岐(`:500`)、`checkCapacityAndStart`(`:1032`)、`spawnCapacityErr` の戻り値(`:1063`)、`dryRunSpawn` の引数(`:1680`)では「予約を記録するか」の `bool`。隣に `p.Reserve`(`[]string`)もある。`scopeReservedEvent("", p.OrgID, p.Reserve, "", true)`(`:1686`)と `scopeReservedEvent(o.now(), orgID, paths, "restored: "+why, false)`(`verbs.go:1233`)は、文字列 2 つと `bool` を位置で渡すので、呼び出し側だけでは ts と note、dry-run の向きが読めない | `spawn.go:437`、`:500`、`:1032`、`:1063`、`:1680`、`:1686`、`verbs.go:1233`、`reserve.go:151` | `bool` 側を `recordReservation` にする。dry-run の呼び出し(`ts` が空)は、`dryRunSpawn` が `steps[i].TS` を上書きするので、`scopeReservedEvent` の引数から `ts` を外して呼び出し側で足す形でもよい |
| F-3 LOW | readability | autonomous の scope ゲートのエラー文が「requires --scope or --reserve」で、`--reserve` を使えない座席(leader 以外)にも勧める。その座席が従うと、次は「only the leader seat reserves paths ... (spawn it without reserve paths)」で拒否され、2 回目の拒否が 1 回目の案内と食い違う。`autonomousScopeGateErr` の条件は `p.SeatID` を見ない(`Reserve` を渡せるのが leader だけなので、条件としては合っている) | `spawn.go:1189`、`:430-441`、`:1177-1193` | 文を seat によって変えるか、「requires --scope (the leader seat may pass --reserve instead)」のように leader 限定を文に入れる。`autonomousScopeGateErr` は `p` を持っているので、`p.SeatID == LeaderIdentity` で分けられる |
| F-4 LOW | maintainability | 予約だけを渡した autonomous の leader は、scope ゲートを通るのに、役割プロンプトの `{{SCOPE}}` は `p.Scope` だけで描かれ、既定の「未指定(読み取り中心で、リポジトリ規約に従うこと)」になる。ゲートの目的(autonomous の座席が担当範囲を持つこと)を `--reserve` で満たしたことにしながら、座席に範囲が渡らない。計画の進捗の(c)として実装者が見つけて送る予定の項目と同じ。直す場合は `Scope` が空のとき `strings.Join(p.Reserve, ", ")` を渡す 1 行で、予約と `--scope` が一致する | `spawn.go:809`、`:1715`(`Scope: p.Scope`)、`prompts.go:21-26`、`:82`、`internal/org/prompts/leader.md:4`、計画の進捗 156 行 | 今直すか tech-debt に送るかを決める。送るなら、`/org` skill と README の「`--reserve` は `--scope` のゲートを満たす」の隣に、座席のプロンプトには範囲が出ないことを 1 文足す |
| F-5 LOW | exception-handling | `reserveAgain` に渡す `rr.Events` は、`CloseDeferredSelfWorkspace` の冒頭(`:1087` 付近)で読んだ snapshot で、最大 3 回 10 秒の herdr の呼び出しの前のもの。窓の間に同じ org_id を `ralph org start --reserve` で立て直した場合、snapshot は「予約なし」を示し、`reserveAgain` は古い予約を新しい予約の後ろに追記する。`currentOrgLives` は最後の `scope_reserved` を採るので、新しい予約が古いものに入れ替わる。重なりの検査もしない。計画のリスクに書いた窓(ほかの org が枠を取る)とは別の、同じ org_id の組み合わせ。`reactivateSeat` も同じ snapshot を使う既存の形なので、新しく持ち込んだというより同じ形を踏襲している | `verbs.go:1111`(呼び出し)、`:1215-1237`(`reserveAgain`)、`reserve.go:218`(最後が勝つ) | 追記の直前に `o.Manifest.Read()` で読み直し、そのとき `ActiveReservation` が nil の場合だけ書く。直さないなら `reserveAgain` の doc の窓の説明に、同じ org_id の立て直しも含めて書く |
| F-6 LOW | maintainability | 同じ導出が 2 つある。`currentOrgLives` の workspace の畳み込み(`DeleteFunc` してから `Created` なら追記)は `openOrgWorkspaces` と同じ処理を、最後の `disbanded` より後に限って書き直したもの。`orgLife.workspaces` は `len(...) > 0` の判定にしか使われず、「oldest first」の順序も誰も読まない。2 つが食い違うと、`disband --all` の対象(`orgsToDisband` が使う `openOrgWorkspaces`)と max_orgs の「走っている」が別々に動く。いまは `PaneID == ""`、dry-run、org 単位の 3 条件が同じで合っている | `reserve.go:159-164`、`:206-214`、`:241`、`spawn.go:1833-1850` | `currentOrgLives` が org ごとに `events[d+1:]` を `openOrgWorkspaces` へ渡す形にするか、少なくとも `openOrgWorkspaces` の doc に「`currentOrgLives` と同じ規則」と足す。`workspaces` は `[]string` のまま持つ理由がなければ `open bool` でよい |
| F-7 LOW | maintainability | `newOrgSpawnRuntime` は `newOrgRuntime` の 2 手順(`ResolveOrgStateDir`、`guardLegacyOrgStateDir`)をそのまま写している。`newOrgRuntime` に手順が増える(別の guard や警告)と、spawn と start だけが取り残される。写した側の doc は「`newOrgRuntime` と同じ」としか言わず、写しであることが読み手に見えにくい | `cli/org.go:108-114`、`:121-134` | 解決と guard を 1 つの関数(`resolveOrgLedger(cmd, stateDir, access) (dir, source string, err error)` など)に切り出して両方が呼ぶか、`newOrgRuntime` に限度の読み元を差し替える引数を足す |
| F-8 LOW | readability | 読めない `scope_reserved` を「repo 全体」として扱う選択は fail-closed で、doc comment(`reserve.go:183-186`)にも書いてある。ただし読み戻した結果は本物の `.` と区別できない。`ralph org status` は `reserved: .` と出し、`reservationDecision` の拒否は「already reserves ., so a different reservation ... is refused」と言い、`reserveAgain` はそれを `paths=.` の明示の記録として書き直す。利用者から見ると、自分が予約していない repo 全体の予約が現れ、原因の台帳の記録の破損は出ない | `reserve.go:216-219`、`:256-262`、`:289-292`、`verbs.go:1226-1233`、`cli/org.go:1109` | 影響は壊れた記録がある org の中だけなので、直さなくてもよい。直すなら、`ActiveReservation` が「読めなかった」を返し、`status` と拒否の文に「(unreadable scope_reserved record, treated as the whole repo)」を足す |
| F-9 LOW | readability | ディレクトリを末尾の `/` なしで書く(`--reserve internal/auth`)と、規則どおりファイルの予約になり、`internal/auth/token.go` や `internal/auth/` の予約と重ならない。拒否もされず、予約が付いたように見えて何も守らない。規則は計画の AC6 のとおりで、help と skill にも書いてあるが、書き間違えたときに何も知らせない。CLI は `--cwd` を持っているので、パスが存在するディレクトリなら気づける | `reserve.go:90-102`(`default: return a == b`)、`cli/org.go:372-378`(`orgReserveFlagUsage`) | CLI で、`filepath.Join(cwd, path)` が既存のディレクトリで末尾の `/` がないとき、stderr に「`internal/auth` is a directory; write `internal/auth/` to reserve its contents」を 1 行出す(予約の判定は変えない)。出さないなら tech-debt に残す |
| F-10 LOW | maintainability | 補償は workspace の経路(`CloseDeferredSelfWorkspace`)にしか予約を戻さない。`CloseDeferredSelfPane` は `reactivateSeat` だけで、`disbanded` で解けた予約を戻さない(計画の進捗の(a)と同じ)。その経路の org は、座席が動いている扱いに戻って走っている一方、範囲は誰にも守られない。AC15 は workspace の経路に限っているので計画どおりだが、同じ種類の欠けが片方に残る | `verbs.go:1010-1034`(`CloseDeferredSelfPane`)、`:1111`(`CloseDeferredSelfWorkspace` の `reserveAgain`)、計画の進捗 156 行 | 計画どおり tech-debt に送る。`reserveAgain` の doc に、pane の経路は戻さないことを 1 文足すと、次に読む人が経路の非対称を見落とさない |
| F-11 LOW | readability | help 文の 2 点。(1) `--config` の persistent flag の説明に、spawn と start だけに関わる文が入り(「without --config, spawn and start read ...」)、`--config` 自身の説明の中で「without --config」と言うので読みにくい。`stop` や `status` の help にも出る。(2) `orgWideLimitsHelp` は「Before starting a seat, spawn checks ...」で始まり、`ralph org start --help` にもそのまま出るので、start の読み手には動詞の `spawn` が何を指すか曖昧 | `cli/org.go:46`、`:361-370`、`:395`、`:531` | (1)は `--config` の説明を元に戻し、限度の読み元は `spawn` と `start` の Long にだけ書く(`orgWideLimitsHelp` に既にある)。(2)は「Before starting a seat, `ralph org spawn` and `ralph org start` check ...」にする |


### Coverage gaps

- テスト、mutation、`check-sync.sh`、`check-skill-sync.sh` は実行していない(指示による)。`/org` skill の 4 面が同じ内容であることは `diff -q` だけで見た
- 本物の herdr で、補償の窓(`disbanded` から補償までの数十秒)を作った実測はしていない。窓の挙動は `verbs_test.go` の fake で固定されたものを読んだだけ
- 同時実行のテスト(`TestOrgSpawn_ConcurrentSpawns_*`、`ConcurrentReservations_*`)は名前と構成を見ただけで、flock の下で本当に直列化されているかは再現していない
- macOS の標準の大文字小文字を区別しないファイルシステムでは、`Internal/` と `internal/` は別のパスとして比べられ、重ならない。パスの正規化は大文字小文字を畳まない。予約は助言であり(範囲外への書き込みは止めない)、計画に書かれていない扱いなので、finding にはせず、`/verify` と `/sync-docs` が判断する材料として残す

### Tech debt identified

新しい行は `/sync-docs` で足す。計画の進捗が挙げた 4 件のうち、(a)は F-10、(c)は F-4 と同じ。(b)(予約のパスの `*` はそのままファイル名として扱う)は skill に書いてあり、コードの挙動は `normalizeReservePath` が `*` を弾かないことと合う。(d)(start の `--scope` の help の表示崩れ)は既存。このほか、F-5、F-8、F-9 は直さない場合に tech-debt の候補。

`/sync-docs` に渡すもの(文書のずれは見ていないので、位置だけ):

- `docs/quality/quality-gates.md:77` は envelope validation を `max_seats` だけで書いている
- `internal/org/envelope_summary.go:36`(leader のプロンプトの `{{ENVELOPE}}`)と `internal/cli/doctor.go:788`(doctor の `[org]` の要約)は `max_seats` だけを出し、`max_orgs` と `max_total_seats` は出さない。出す・出さないは判断が要る
- `.claude/skills/org/SKILL.md:201` の「`--config` を渡したときと、台帳を flag か env で決めたとき、git の外では」は、`ResolveOrgStateDir` の 4 段目(`git-toplevel`、bare repository の linked worktree)も、main の `ralph.toml` を読まない場合に入ることを書いていない(`internal/org/statedir.go:38-44` と `withMainWorktreeOrgLimits` は `source != "git-main-worktree"` のすべてで読まない)
- 同じ SKILL の「どちらも台帳のロックの下で判定するので、同時に打った `spawn` でも超えない」は、自分の workspace の close が失敗した補償の窓では超える(計画のリスクとテスト `..._RiskWindowClearedByRetry` が固定している)ことを書いていない

## 付録 B: cycle 2 の self-review(a919082a 時点、HEAD 710de10d の原文)

cycle 2 の self-review の本文を、そのまま残す。tech-debt の行 177、verify の cycle 2、cross-review の triage が C2 番号と、cycle 1 の F 番号の現況表で指している。`file:line` は 710de10d 時点のもので、`internal/org/verbs.go` は a94c914f で行がずれた(関数名で探す)。`spawn.go`、`reserve.go`、`internal/cli/org.go` は a919082a 以降変わっていない。見出しは 1 段下げ、先頭のメタデータの行はそのまま次に置く。

- Date: 2026-10-09 JST(ファイル名は計画の日付)
- Plan: docs/plans/active/2026-10-08-org-limits-reserve.md
- Branch: feat/org-limits-reserve(HEAD 710de10d、base 51855166)
- Reviewer: reviewer subagent (Claude)、パイプライン 2 回目(cycle 2、既定の上限の最後の回)
- Scope: diff の品質だけ。cycle 1 の self-review(d8ec84da)以降の差分 `git diff d8ec84da..HEAD -- ':!docs'` の 13 ファイル(+348/-81)が対象。コードは 8dd19634(文言)、975df92b と afcbc6c2(cross-review の ACTION_REQUIRED の修正)、c3a95c48(テスト)、文書とヘルプ文は 76d1cf1c(`/sync-docs` の入力)。重点は、`validateMaxOrgs` が `ValidateOrgWideCapacity` の挙動と文言を保つか、動いていない leader の予約が上限の判定なしに記録される経路がほかにあるか、そこを plain rejection にした選択、76d1cf1c のヘルプ文とコードの一致。仕様への適合、テストの網羅、文書のずれは見ていない(`/verify`、`/test`、`/sync-docs` の担当)。テスト、linter、型検査、formatter、mutation は実行していない
- 番号の付け方: cycle 1 の finding は F-1〜F-11(付録 A に原文を残す。tech-debt の行 172〜176 と verify・plan が F 番号で指している)。この回の finding は `C2-` を付ける。同じ報告の上書きで番号が付け替わらないようにするため

### Evidence reviewed

- `git diff d8ec84da..HEAD -- ':!docs'` の全行。`internal/org/envelope.go`、`internal/org/spawn.go`、`internal/config/config.go`、`internal/cli/org.go`、`AGENTS.md`、`templates/base/ralph.toml`、`templates/base/docs/quality/quality-gates.md`、skill の 4 面、`spawn_test.go` と `verbs_test.go` の追加分。コミットの親子は `git log --format='%h parent=%p'` で確かめ、一直線(8dd19634 → fbb04f83 → c3a95c48 → fff1ec23 → 76d1cf1c → 196205f8 → 6600b3a9 → 975df92b → afcbc6c2 → 710de10d)で、この回のコード変更はすべて上の範囲に入っている
- `validateMaxOrgs` の同値性: `git show d8ec84da:internal/org/envelope.go` の `ValidateOrgWideCapacity` と HEAD を並べて読んだ。条件(`len(runningOrgs) >= cfg.MaxOrgs && !slices.Contains(runningOrgs, orgID)`)、`running == ""` のときの `none`、`fmt.Errorf` の書式と引数の順、max_orgs → max_total_seats の順序、上限が 0 以下の hand-built config の扱いは同じ。変わったのは仮引数が `req.OrgID` から `orgID` になった点だけ
- 予約を記録する経路: `grep -rn 'scopeReservedEvent\|EventScopeReserved' internal cmd` で非テストの呼び出しを数えた。`checkCapacityAndStart`(`spawn.go:1040`)、`idempotentRespawn`(`:1110`)、`dryRunSpawn`(`:1704`、dry-run のイベント)、`reserveAgain`(`verbs.go:1233`、補償)の 4 つ。`idempotentRespawn` の呼び出し元は Phase 1(`:606`)と Phase 2(`:762`)の 2 つで、どちらも `Roster` の座席をそのまま渡す
- `Roster`(`manifest.go:171`)、`currentOrgLives`(`reserve.go:187`)、`RunningOrgs`(`:233`)、`reservationDecision`(`:283`)、`stateEvents` と `activeEvents`(`seat.go:59`、`:77`)、`reject`(`spawn.go:1671`)を読み、plain rejection の理由と、座席が `spawned` なのに Active でない状態を作る経路を確かめた
- 76d1cf1c のヘルプ文: `orgWideLimitsHelp` を `withMainWorktreeOrgLimits`(`cli/org.go:148`)と `ResolveOrgStateDir`(`statedir.go:59-72`)、`MainWorktreeRoot`(`:81`)に突き合わせた。`status` の `Long` は `printStatusTable` と `orgStatusJSON`(`omitempty`)に突き合わせた
- skill の差分(補償の例外、設定の読み元、末尾の `/` のないパス、`{{SCOPE}}`)をコードと突き合わせた。4 面は `cmp` で同一、`docs/quality/quality-gates.md` と `templates/base/docs/quality/quality-gates.md` の envelope validation の行は同じ文
- 機械的な確認: `git diff --check d8ec84da..HEAD` は空。追加行に U+FFFD、`fmt.Print`、`println`、`TODO`、`FIXME` はない。出荷物(`internal`、`templates/base`、`.claude`、`.agents`、`scripts`、`AGENTS.md`)で `director` の語は 0 件
- tech-debt の行 170、172〜176 を HEAD の内容と突き合わせた。cycle 1 の推奨で足した行で、この回のコミットが事実を変えたものがないかを見た

### 依頼された 4 点の結論

1. `validateMaxOrgs` は `ValidateOrgWideCapacity` と同じ挙動、同じ文言を返す(上の Evidence)。`ValidateOrgWideCapacity` の doc も、まだ合っている。
2. 動いていない leader の予約は、上限を見ずに記録される経路が残っていない。`idempotentRespawn` は座席が Active でないとき `validateMaxOrgs` を通り、`checkCapacityAndStart` は `spawnCapacityErr` の全判定を通る。残る 2 つは、`reserveAgain`(補償。判定しないことは行 172 に記録済み)と dry-run のイベント(数えられない)。座席が `spawned` のまま Active でなくなるのは、`disbanded` が `stopped` なしで続いた古い台帳だけ。現行の `Disband` は先に `stopped` を書き、`stop_failed` は状態のイベントではないので座席は Active のまま残る。
3. plain rejection(`rejected` を書かない)の選択は適切。`rejected` は状態のイベント(`stateEvents`)なので、書くと座席の最新の状態が `spawned` から `rejected` に変わり、次の予約なしの再試行が idempotent の返しを失って、新しい spawn に進む。ただし doc の理由はこの枝に当てはまらない(C2-2)。
4. ヘルプ文は内容がコードと合う。`status` の `Long` は `printStatusTable` と JSON の `reservation`(予約がなければ出ない)に合う。`orgWideLimitsHelp` の読み元の条件は `withMainWorktreeOrgLimits` と合う。直す点は文の組み立てだけ(C2-4)。

### Findings

<!-- Area recommended values: naming, readability, unnecessary-change, typo,
     null-safety, debug-code, secrets, exception-handling, security, maintainability -->

| Severity | Area | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- |
| C2-1 LOW | readability | `idempotentRespawn` の doc で「decided first」が 2 回出て、何が先か読み取れない。1 段落目は「the reservation is decided first against the same locked events」、2 段落目は「max_orgs is decided first with validateMaxOrgs」。コードの順序は max_orgs、予約、座席を返す、の順。`Spawn` の手順 1 の「the reservation is decided first, after max_orgs when the seat is not active」も、1 文の中で「first」と「after」が食い違う。3 か所とも同じ編集で書かれ、互いに矛盾して読める | `spawn.go:345-346`、`:1081`、`:1088`、`:1100-1112` | 「座席を返す前に」の意味で書き直す。`Spawn` の手順 1: 「With Reserve, max_orgs (when the seat is not active) and then the reservation are decided before the seat is returned (idempotentRespawn).」。`idempotentRespawn` は 1 段落目の「first」を「before the seat is returned」に、2 段落目を「max_orgs is decided before the reservation」にする |
| C2-2 LOW | maintainability | 新しい枝の拒否は plain rejection で、その選択は正しい。ただし理由が書かれていない。doc の理由は「a `rejected` for the seat would replace `spawned` as its latest state event (for a running leader, showing it inactive)」だが、この枝の座席はすでに inactive なので「inactive に見える」は当てはまらない。実際の理由は、`rejected` が状態のイベントなので次の予約なしの再試行が idempotent の返しをせず、新しい spawn を始めてしまうこと。計画側も、AC1 は「台帳に `rejected` が残る」、進捗(156 行)は「AC1 にコードを合わせる直し」と書き、この枝が `rejected` を書かないことには触れていない。AC1 の文字どおりには満たさない経路が、満たしたように読める | `spawn.go:1094-1097`(doc)、`:1100-1103`、`seat.go:59-67`(`stateEvents` に `EventRejected`)、`spawn.go:1671-1676`(`reject` は `SeatID: p.SeatID` で書く)、計画 89 行(AC1)と 156 行 | doc の括弧を「for an inactive seat it would also stop the next retry from being returned as idempotent」の趣旨に差し替える。計画には「この枝は AC14 の拒否と同じく `rejected` を書かない」の 1 行を足す。どちらもコメントと計画の文なので、cap のこの回では tech-debt に送る(Tech debt identified) |
| C2-3 LOW | maintainability | `if !seat.Active` は、外しても結果が変わらない。Active な座席は `RunningOrgs` の最初のループでその org を「走っている」にするので、`validateMaxOrgs` は `slices.Contains(runningOrgs, orgID)` で nil を返す。分岐が省くのは、通ると決まっている呼び出しだけで、doc の「An Active seat's org is running, so it skips this check」もそれを書いている。そのため、(1) 読み手には Active の有無で規則が分かれるように見える、(2) この条件を外す・間違える変更を落とすテストは書けない(新しいテストは、どちらの条件でも `validateMaxOrgs` が拒否か通すかを決める)、(3) 正しさが `Active` の導出と `RunningOrgs` の定義の両方に寄りかかる。コードを読んだ結論で、条件を外した mutation は回していない(未確認) | `spawn.go:1100-1104`、`reserve.go:233-239`(`RunningOrgs`)、`envelope.go:107-108` | 条件を外し、予約つきの再試行では常に `validateMaxOrgs` を通す(費用は再試行ごとの `RunningOrgs` 1 回)。残すなら、doc の「skips this check」を「skips a call that cannot fail」と言い換え、近道であることを書く |
| C2-4 LOW | readability | `orgWideLimitsHelp` の「Only when --config is not given and the ledger is the main worktree's, those two are read from ...」は、only が文頭にあるのに語順が平叙文のままで、英文として崩れている。意味の面でも、cycle 1 の版は最初の文の中で「(no --state-dir or RALPH_ORG_STATE_DIR)」と条件を絞っていたが、今の版は後ろの「In every other case (...)」で初めて、`--state-dir <main>/.harness/state/org` のように flag で main の台帳の path を指した場合を外す。コードは source が `git-main-worktree` のときだけ読む(`MainWorktreeRoot`)ので、最初の文だけ読むと、その場合も main の `ralph.toml` を読むように取れる。同じ言い回し(「when the state dir is the main worktree's .harness/state/org」)が、この差分では触れていない `templates/base/ralph.toml` にもあり、そちらは利用者に配られる | `internal/cli/org.go:363-372`、`internal/org/statedir.go:59-72`、`:81-84`、`templates/base/ralph.toml:41-44`、cycle 1 の版は `git show d8ec84da:internal/cli/org.go` の 365-367 行 | 「Those two are read from the main worktree's ralph.toml (built-in defaults when it has none) only when --config is not given and the ledger was resolved by default to the main worktree's (neither --state-dir nor RALPH_ORG_STATE_DIR is set).」の形にして、後ろの「In every other case」は残す。`ralph.toml` のコメントも「resolved by default」を足して同じ条件にする |
| C2-5 LOW | maintainability | `/verify` と `/test` はこの回、報告を同じパスに上書きする。tech-debt の行と Go のコメントは、その中の番号を指している。行 170 は verify の V-7、V-8、行 172 は V-2、行 174 は V-4、V-5、行 176 は test の Test gaps 3〜5、7、8 と mutation X18、`verbs_test.go` のコメントは「verify V-2」。この報告は付録 A で F 番号を残すが、verify と test に同じ手当てを頼まないと、番号が別の finding に付け替わる。行 176 の(b)の「`idempotentRespawn` (87.5%)」は 975df92b が枝を足す前の値で、今の関数には合わない | `docs/tech-debt/README.md:170`、`:172`、`:174`、`:176`、`internal/org/verbs_test.go:3083-3085`、`docs/reports/verify-2026-10-08-org-limits-reserve.md` の 69、82、86、99、103 行、`docs/reports/test-2026-10-08-org-limits-reserve.md` の 96、132 行 | `/verify` と `/test` の cycle 2 に、V-1〜V-9、Test gaps 1〜8、X18 の番号を残すよう頼む(旧版を付録に残し、新しい指摘は別の接頭辞で採番する)。行 176 のカバレッジの値は関数名だけにする。Go のコメントの「verify V-2」は、挙動(「the pane compensation does not check max_total_seats」)で書き直す |
| C2-6 LOW | readability | 行 173 の(a)は F-9 を「`--reserve internal/auth` is accepted and protects only a file ... and nothing says so」と書くが、同じ行の見直しの欄は「the help text and the `/org` skill state them」と書き、skill には 76d1cf1c が足した「末尾に `/` がないパスはファイルなので、`--reserve internal/auth` が守るのは `internal/auth` という名前のファイルだけ」がある。行の中で食い違う。言いたいことは「入力したときに警告が出ない」で、その文言なら食い違わない | `docs/tech-debt/README.md:173`(説明の欄と見直しの欄)、`.claude/skills/org/SKILL.md:223-225` | 「and nothing says so」を「and nothing warns when the path is typed」に変える |
| C2-7 LOW | readability | `AGENTS.md` の repo map の「envelope validation (per-org `max_seats`; org-wide `max_orgs` / `max_total_seats` and scope reservations in `reserve.go`)」は、`in reserve.go` が 3 つ全部にかかるように読める。org-wide の判定そのもの(`ValidateOrgWideCapacity`、`validateMaxOrgs`)は `envelope.go` にあり、`reserve.go` には数え方(`RunningOrgs`、`TotalActiveSeats`)と予約がある。この文書は「map」で、grep で場所に着くことが役割 | `AGENTS.md:90`、`internal/org/envelope.go:92`、`:107`、`internal/org/reserve.go:233`、`:265` | 「org-wide `max_orgs` / `max_total_seats` in `envelope.go`, their counts and scope reservations in `reserve.go`」の形にする |

その他に数えない指摘が 1 つある。`autonomousScopeGateErr` の文(`spawn.go:1207`)は「(--reserve on the leader seat only; or --allow-unscoped to explicitly bypass)」で、セミコロンのあとの `or` が `--reserve` の注記の続きに読める。F-3 の推奨(leader 限定を文に入れる)は満たしているので、直さなくてよい。

### cycle 1 の finding の現況

| F | 重さ | この回の判定 | 根拠 |
| --- | --- | --- | --- |
| F-1 | MEDIUM | 直った | 8dd19634 が `config.go` と `templates/base/ralph.toml` から director の文を消した。出荷物で `director` は 0 件 |
| F-2 | LOW | 繰り越し(行 175 の(a))。悪化なし | 新しいコードの bool は `record`(`idempotentRespawn`)で、`reserve` の二重の意味は増えていない |
| F-3 | LOW | 直った | 8dd19634。`spawn.go:1207` が「--reserve on the leader seat only」を持つ。上の「その他」を参照 |
| F-4 | LOW | 繰り越し(行 174 の(a))。skill に断り書きが入った | 76d1cf1c が skill に `{{SCOPE}}` の説明を足し、行 174 の(a)もそれを書く |
| F-5 | LOW | 繰り越し(行 172 の(a))。悪化なし | `verbs.go` はこの回に触れていない |
| F-6 | LOW | 繰り越し(行 175 の(b))。悪化なし | `reserve.go` はこの回に触れていない |
| F-7 | LOW | 繰り越し(行 175 の(c))。悪化なし | `newOrgSpawnRuntime` はこの回に触れていない |
| F-8 | LOW | 繰り越し(行 173 の(c))。悪化なし | 同上 |
| F-9 | LOW | 繰り越し(行 173 の(a))。skill が規則を明記した | 76d1cf1c。行の文言との食い違いは C2-6 |
| F-10 | LOW | 繰り越し(行 172 の(b))。悪化なし | `CloseDeferredSelfPane` はこの回に触れていない |
| F-11 | LOW | 直った | 76d1cf1c。(1)`--config` の説明から spawn と start の文を外し、(2)`orgWideLimitsHelp` の頭を「`ralph org spawn` and `ralph org start`」にした。新しい文の組み立ては C2-4 |

### Positive notes

- 判定を `validateMaxOrgs` に切り出して、新しい枝と既存の経路が同じ関数、同じ文言を使う。テスト(`TestOrgSpawn_Reserve_InactiveLeaderChecksMaxOrgs`)の期待する文字列は `ValidateOrgWideCapacity` の書式を組み立て直したもので、2 つが食い違えば落ちる
- 新しいテストは、上限に達した場合に記録なし、receipt なし、agmsg の呼び出しなし、座席が `spawned` のまま inactive、`RunningOrgs` が org-b だけ、を確かめる。Phase 1 と Phase 2 の両方、上限 1 と 2、別の座席で org が走っている場合を含む。修正前のコードなら org-a の `scope_reserved` が増えて `lastEvent` の検査で落ちる(読みによる。mutation は回していない)
- 判定の順序が、新しい spawn(max_seats、max_orgs、max_total_seats、予約)と idempotent の枝(max_orgs、予約)で同じ向きに並ぶ。走っている org の再試行は max_orgs を通るので、上限が no-op の再試行を拒否することはない
- 補償の例外を skill に書き、pane 側の窓(`max_total_seats` を 1 つ超える)をテストで固定し、行 172 の見直しの欄に「テストと skill の文を一緒に変える」と書いてある
- skill の 4 面、`quality-gates.md` の該当行は、ミラーのどちらも同一

### Coverage gaps

- テスト、mutation、`check-sync.sh`、`check-skill-sync.sh` は実行していない(指示による)。C2-3 は読みによる結論
- ヘルプ文は `ralph org spawn --help` を出して確かめていない。Go の文字列と周辺のコードを読んだ
- dry-run は idempotent の枝を持たないので、動いていない leader(Active な leader も同じ)への `--reserve --dry-run` は max_seats と max_total_seats まで判定し、本番の枝より厳しい予測になる。cycle 1 から同じ形で、この回の差分で広がってはいない
- verify の cycle 1 の報告は、V-1(bare リポジトリの列挙)、V-2(補償の断り)、V-3(quality-gates)、V-5(`{{SCOPE}}`)、V-6(F-11)、V-9(チェックボックス)を未対応と書くが、いずれも後続のコミット(76d1cf1c、196205f8、計画の差分)で解けたように見える。この回の `/verify` が HEAD で読み直す前提で、ここでは判定しない

### Tech debt identified

この回は既定の上限の最後なので、C2-1〜C2-7 のうち直さないものは繰り延べにあたる。どれもコメント、ヘルプ文、計画、台帳の文言で、挙動は変わらない。コードのコメントを直すと全パイプラインの再実行になり、上限を超える。1 行にまとめて `docs/tech-debt/README.md` に足す案を次に書く。足すのは `/sync-docs` の担当か、オペレーターの判断。

> org-limits-reserve の文言の LOW(self-review cycle 2 の C2-1〜C2-7)。(a) `idempotentRespawn` と `Spawn` 手順 1 の doc で「decided first」が食い違い、plain rejection の理由が既に inactive な座席に当てはまらない。(b) `if !seat.Active` は `validateMaxOrgs` が org の走っている座席に通す結果と同じなので外せる。(c) `orgWideLimitsHelp` と scaffold の `ralph.toml` の読み元の条件の書き方。(d) 行 170〜176 と `verbs_test.go` が verify の V 番号、test の Test gaps、カバレッジの値を指す。(e) `AGENTS.md` の repo map の `reserve.go` の係り方。トリガー: `idempotentRespawn`、`orgWideLimitsHelp` の次の変更、または verify・test の報告が番号を付け替えたとき

C2-6(行 173 の 1 語)は、台帳の別の編集のついでに直せる。

### Recommendation

- Merge: 可。CRITICAL、HIGH、MEDIUM はない。LOW は 7 件(C2-1〜C2-7)で、いずれもコメント、ヘルプ文、台帳、計画の文言。cross-review の修正(975df92b、afcbc6c2)は挙動、エラー文、順序が正しく、`ValidateOrgWideCapacity` の既存の挙動を変えていない
- Follow-ups: C2-1〜C2-7 は cap のため直さず、上の 1 行にまとめて tech-debt へ送るのが現実的。C2-5 の番号の保持は、直す・直さないにかかわらず、`/verify` と `/test` の cycle 2 に先に頼む。`/sync-docs` に渡すもの: 行 173 の C2-6、C2-4 の `ralph.toml` のコメント、C2-7 の `AGENTS.md`
- cycle 1 の F-1(MEDIUM)、F-3、F-11 は直り、F-2、F-4〜F-10 は行 172〜176 に載っていて悪化していない

## 付録 C: cycle 3 の self-review(f19df087 時点、HEAD fc35fee6 の原文)

cycle 3 の self-review の本文を、そのまま残す。tech-debt の行 179 と、verify・test・sync-docs の cycle 3、cross-review の triage が C3 番号で指している。`file:line` は fc35fee6 時点のもので、`internal/org/verbs.go` と `verbs_test.go` は 83aec44e で行がずれた(関数名で探す)。見出しは 1 段下げ、先頭のメタデータの行はそのまま次に置く。

- Date: 2026-10-09 JST(ファイル名は計画の日付)
- Plan: docs/plans/active/2026-10-08-org-limits-reserve.md
- Branch: feat/org-limits-reserve(HEAD fc35fee6、base 51855166)
- Reviewer: reviewer subagent (Claude)、パイプライン 3 回目(cycle 3)。ユーザーが上限を 3 に上げた回で、`cycle-count.json` は /cross-review の上限引き上げの決まりで 2 のままなので、この報告と insight event に cycle 3 と明記する
- Scope: diff の品質だけ。cycle 2 の self-review(a919082a)以降の差分 `git diff a919082a..HEAD -- ':!docs'` の 4 ファイル(+382/-72)が対象。コードの変更は a94c914f(`internal/org/verbs.go` と `verbs_test.go`)だけで、`AGENTS.md` と `templates/base/ralph.toml` は cycle 2 の `/sync-docs`(27fefc47)の編集。重点は、補償のロックの範囲と順序(入れ子や herdr をまたぐ保持がないか)、`reactivateSeat` / `reopenWorkspace` / `reserveAgain` の before と now の判定規則、skip のエラー文と byHand の案内、doc comment とコードの一致、ロックの外に残る窓を doc が正直に書いているか。仕様への適合、テストの網羅、文書のずれは見ていない(`/verify`、`/test`、`/sync-docs` の担当)。テスト、linter、型検査、formatter、mutation は実行していない
- 番号の付け方: cycle 1 の finding は F-1〜F-11(付録 A に原文)、cycle 2 は C2-1〜C2-7(付録 B に原文)、この回は `C3-` を付ける。tech-debt の行 170〜178 と verify・test・plan が F 番号と C2 番号で指しているので、報告を上書きしても番号は付け替えない

### Evidence reviewed

- `git diff a919082a..HEAD -- ':!docs'` の全行。親子は `git log --format='%h parent=%p'` で確かめ、一直線(f144021d → 6ee9836b → 27fefc47 → c4edd978 → a94c914f → fc35fee6)。`/verify`・`/test`・`/sync-docs` の cycle 2 の報告は a94c914f より前の tree に対するもので、a94c914f はまだどの検査も通っていない(この報告も読みだけ)
- ロックの入れ子と保持: `withManifestLock`(`lockfile.go:50-70`)は呼び出しごとに fd を開いて `flock` するので、同じプロセスの入れ子は自分のロックを `manifestLockTimeout`(5 秒)待って timeout になる。入れ子になる経路を探したが、`compensateUnderLock` の呼び出し元は `CloseDeferredSelfPane`(`verbs.go:1040`)と `CloseDeferredSelfWorkspace`(`:1121`)だけで、この 2 つの呼び出し元は `closeDeferredSelf`(`internal/cli/org.go:984`、`:987`)だけ。`Stop` と `Disband` が戻ったあとに呼ぶので、ロックを持った呼び出し元はいない。ロックの中で動く関数(`reactivateSeat`、`reopenWorkspace`、`reserveAgain`、`lastSeatOnPane`、`o.getenv`、`o.now`)は herdr も agmsg も `withManifestLock` も呼ばず、台帳への書き込みは `appendEvent`(`appendJSONLine`、ロックを取らない)だけ
- ロックの外の書き込み: `grep -n 'appendEvent(' internal/org/*.go` の呼び出しごとにイベント名を読んだ。ロックの中: `checkCapacityAndStart`(`spawn.go:1033`、呼び出しは `:702` と `:770`、どちらも `withManifestLock` の closure の中)の `scope_reserved`、`spawn_started`、`rejected`。ロックの外: `spawn_step`(`:790`、`:893`、`:932`、`:954`、`:1268`)、`spawned`(`:976`)、`org_workspace_created`(`:1835`)、`spawn_failed`(`:1903`、`:1936`)、`Stop` の `stopped` / `stop_failed`(`verbs.go:924`)、`Disband` の `org_workspace_closed`(`:1934`)と `disbanded`(`:1944`)
- 判定規則: `Roster`(`manifest.go:171`)、`seatFromEvents`(`verbs.go:74`)、`lastWorkspaceEvent`(`:1343`)、`ActiveReservation`(`reserve.go:256`)、`reservationBeforeLastDisband`(`:311`)、`currentOrgLives`(`:187`)を読んだ。`SeatStatus` と `ManifestEvent` は文字列と bool だけの構造体で、`!=` で全フィールド(TS、Details、`Active` を含む)を比べる。`stopped` の座席は前後どちらの読みでも `Active` が false なので、`current != seat` は「最新の状態イベントが変わった」と同じ意味になる。`Roster` は状態イベントだけを見るので、`sent` などの非状態イベントで skip にはならない
- テスト: 追加した 8 ケース(`NothingRestored_NamesTheManualClose` の lock の 2 つと、`TestOrgCloseDeferredSelf_NewerRecordsWhileClosing_Stay` の 6 つ)と、`closeHookHerdr`、`eventsWithDetailsPrefix` を読んだ。helper(`deferOwn*CloseFailing`、`spawnSeatIn`、`spawnLeaderIn`、`ownPaneEnv`、`leaderParams`、`mustReadEvents`、`eventNames`)は既存で、旧シグネチャの呼び出しは `reactivateSeat(`、`reopenWorkspace(`、`reserveAgain(`、`.errorFor(` の grep でテストに残っていない
- 機械的な確認: `git diff --check a919082a..HEAD` は空。追加行に U+FFFD、`fmt.Print`、`println`、TODO、FIXME はない。コミットメッセージに attribution の行はない。追加したコメントはどの行も 80 列以下で、挿入で 1 行だけ長くなった所はない(`awk` で測った)。新しいコメントに `file:line`、段階の名前、`AR#` や finding の ID の引用はない
- 27fefc47 の 2 ファイル: `AGENTS.md:90` と `templates/base/ralph.toml:42-46` を、`withMainWorktreeOrgLimits`(`internal/cli/org.go:148`)と `ResolveOrgStateDir`(`statedir.go:59-72`)の条件、`ValidateOrgWideCapacity` と `RunningOrgs` / `TotalActiveSeats` の置き場所(`envelope.go`、`reserve.go:233`、`:265`)に突き合わせた
- tech-debt の行 170〜178、plan の AC15 と進捗の fc35fee6 の行、`docs/reports/cross-review-triage-org-limits-reserve.md`(cycle 2 の WORTH_CONSIDERING 1 件)を HEAD の内容と突き合わせた

### 依頼された点の結論

1. ロックの範囲と順序: 入れ子も、herdr をまたぐ保持もない(上の Evidence)。ロックを取るのは close が失敗したあとの 1 回で、保持の間にすることは `Manifest.Read()` 1 回と append(workspace の経路で最大 3 回)。待ちの上限は 5 秒で、取れない、または読み直せないときは何も書かずに「nothing recorded again: ...」と byHand になる(`NothingRestored_NamesTheManualClose` の lock の 2 ケースと、`manifest unreadable at the second read` のケースが固定する)。`--force` はロックを取る前に戻る(`verbs.go:1035`、`:1116`)。`before` が何も戻さないと示す場合(3 つの関数がすべて `"", "", nil`)にも毎回ロックを取って読み直すので、戻す必要のなかった失敗でも、ロックが取れなければ「nothing recorded again」と byHand が付く。誤りではなく、費用も小さいので finding にはしない
2. before と now の規則: `reactivateSeat` と `reopenWorkspace` は「最新の 1 イベントが close の前の写しと同じか」(構造体の比較)、`reserveAgain` は「予約がなく、最後の `disbanded` の前の予約が同じパスか」(導出した値の比較)で決める。規則の形が違う。テストした入力ではどちらも期待どおりに分かれ、違いが出る入力が 2 つある(C3-4)
3. skip の文面と byHand: 文は 6 ケースで事実に合う(「did not record ... again: newer records written to the manifest while the close waited stay as they are」)。byHand は skip があれば付き、pane は `herdr pane close <pane>`、workspace は「if it is the org's」つきで、herdr が id を別の workspace に振り直した場合にも誤った案内にならない。ただし `reserveAgain` だけ、`before` にすでにあった予約も「while the close waited」と書いてしまう(C3-4 の(b))。restored と skipped が混ざるときの併記は、下の「その他」の(3)
4. 残る窓を doc が正直に書いているか: 一部だけ(C3-1)。`compensateUnderLock` の doc は、ロックが止めるもの(spawn のロックの中の判定、`scope_reserved`、`spawn_started`)を正確に挙げる。ロックの外に残る書き込みを書かず、規則を「appends only what is still missing」と書くが、これは同じ commit のテストが固定する挙動と合わない

### Findings

<!-- Area recommended values: naming, readability, unnecessary-change, typo,
     null-safety, debug-code, secrets, exception-handling, security, maintainability -->

| Severity | Area | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- |
| C3-1 LOW | maintainability | `compensateUnderLock` の doc が、規則の言い方と、ロックの外に残る窓の 2 点で実際と合わない。(a)「each compensation step compares that copy with this read and appends only what is still missing」は、同じ commit のテストが固定する挙動と逆の場合がある。org を立て直された予約、別 pane で立て直された座席、別の org に振られた workspace id は、台帳には「まだない」のに書かない(`NewerRecordsWhileClosing_Stay` の 4 ケース)。実際の規則は「close の前の写しが戻すべきだと示し、今の読みでもその記録が最新のままなら書く」。(b) ロックが止めるものは正しく書いてある(「a spawn's locked section (its checks, scope_reserved and spawn_started)」)。座席は `spawn_started`、予約は `scope_reserved` が立て直しの最初に書かれ、どちらもロックの中なので、テストした 2 つの競合(座席の立て直しと予約の立て直し)は直列化される。一方、`withManifestLock` に入らない書き込み(`spawned`、`spawn_step`、`spawn_failed`、`org_workspace_created`、`Stop` の `stopped` / `stop_failed`、`Disband` の `org_workspace_closed` / `disbanded`)は、この read と append の間に入りうる。窓は read 1 回の長さで、herdr を待つ時間(F-5 の 10〜40 秒)は含まない。呼び出し側 3 つの doc(「decide ... from the manifest read again under the manifest lock」)、skill、テスト名「a spawn holds the manifest lock」は、ロックの下で判断するとだけ言うので、読み手は書き込み全体が直列化されたと読む。plan の Risks と tech-debt の行 172 もこの残りの窓を持たない | `verbs.go:1198-1207`(doc)、`:1008-1016`、`:1087-1094`(呼び出し側の doc)、`verbs_test.go:3392-3476`(規則を固定する 3 ケース)、`:3477-3509`(ロックのケース)。ロックの外の書き込みは上の Evidence の一覧(`spawn.go`、`verbs.go:924`、`:1934`、`:1944`) | (a)「appends a step only when the copy read before the close asked for it and this read still shows that record as the latest」の趣旨に直す。(b)「Writes made outside withManifestLock (a spawn's spawned and org_workspace_created, stop's and disband's records) can still land between this read and the appends; the window is one read, not the herdr calls」の 1 文を足す。コードのコメントなので、最後の回は直さず tech-debt の行 177 に足す |
| C3-2 LOW | readability | ロックや 2 回目の read の失敗が、「append の失敗」を数える欄に入るのに、3 か所の doc が append だけと書く。`compensateUnderLock` は `c.add("", "", fmt.Errorf("nothing recorded again: %w", err))` で、ロックを取れない、または読み直せない失敗を `failed` に積む。`selfCompensation` の doc は「the appends that failed」、`add` の doc は「err a failed append」、`errorFor` の doc は「the appends that failed」。2 つめの経路が増えたのに、コメントは 1 つめのまま(#3 の型)。`errorFor` の「When something was not recorded or failed」は失敗を含むので合うが、その手前の列挙が append に限られる | `verbs.go:1150-1152`、`:1159-1162`、`:1175-1181`、`:1218` | 3 か所の「append」を「step (an append, the lock, or the second read)」にする。コメントだけなので、最後の回は tech-debt の行 177 に足す |
| C3-3 LOW | maintainability | 補償を無条件に書く文が、a94c914f で条件つきになった挙動のまま残る。a94c914f は、台帳に新しい記録があれば戻さない経路を足した。`closeDeferredSelf` の doc は「A failed close has already recorded the seat active and the workspace open again ... so the command exits 1 and running it again retries the close」、`Disband` の doc は「CloseDeferredSelfPane / CloseDeferredSelfWorkspace record the own seat active and the own workspace open again」、`stop` と `disband` の help は「the seat is recorded active again」「records the pane's seat active again ...」、skill の `stop` と `disband` の行も「座席を active に戻して」と書く。S8 の時点で、読み取りか append の失敗では戻らなかったので、すでに無条件ではなかった。この commit は 3 つめの戻らない場合を足し、そのときのエラーは「打ち直せば閉じられる」ではなく、手で閉じるコマンドを示す。plan の AC15 も無条件の文のままで、fc35fee6 の進捗の行は skip の場合に触れない | `internal/cli/org.go:973-976`、`:831`、`:1146-1150`、`internal/org/verbs.go:1785-1787`、`.claude/skills/org/SKILL.md:168`、`:169`、`:190-195`(`.agents/skills/org/SKILL.md`、`templates/base/.claude/skills/org/SKILL.md`、`templates/base/.agents/skills/org/SKILL.md` の同じ行)、plan 103 行(AC15) | 文を「戻す。台帳にその座席・workspace・予約の新しい記録があれば戻さず、エラーが手で閉じるコマンドを示す」の形にする。Go のコメントと help は code の変更なので、最後の回は tech-debt の行 177 に足し、skill の 4 面は `/sync-docs` で直せる。plan は進捗の行に 1 文足せばよい(approval digest に入らない) |
| C3-4 LOW | maintainability | `reserveAgain` の規則だけ、兄弟 2 つと形が違い、違いが出る入力にテストも doc の文もない。`reactivateSeat` と `reopenWorkspace` は「最新のイベントが写しと同じか」で決め、`reserveAgain` は「今の `ActiveReservation` が nil で、最後の `disbanded` の前の予約が `before` と同じパスか」で決める。旧コードにあった写し側の `ActiveReservation(before) != nil` の早期 return もなくなった(行 176 の mutation X18 の対象)。(a) 同じ org_id が `--reserve` なしで立て直された場合: 予約の記録は書かれないので、座席の規則は新しい `spawn_started` を見て戻さず、予約の規則は何も新しくないと見て古い予約を書き戻す(`ActiveReservation(now)` は nil、`reservationBeforeLastDisband(now)` は変わらない)。立て直した org が頼んでいない予約を持つ。意図としては成り立つ(古い run の leader が ws-1 でまだ動いている)が、doc の「otherwise the org was started again (with --reserve) or disbanded again」は `--reserve` ありだけを書き、6 ケースのどれもこの入力ではない。(b) 立て直しの予約がすでに `before` にあった場合(`disbanded` から最初の read の間。`disband --all` で複数の org を順に閉じたあとなど): 旧コードは黙って戻り、新コードは skip と数えて「did not record the reservation ... again: newer records written to the manifest while the close waited」と byHand を出す。記録は close の前にあったので「while the close waited」は事実と違う。実害は byHand の 1 行と文言の不正確さだけ | `verbs.go:1285-1313`(`:1301`、`:1306`)、`:1222-1255`(`reactivateSeat`)、`:1257-1283`(`reopenWorkspace`)、`verbs_test.go:3392-3476`(`--reserve` ありの立て直しだけ) | 挙動は変えずに、`reserveAgain` の doc の「started again (with --reserve)」を「started again, with or without --reserve (without it the old reservation is restored)」にして意図を残すか、(a) のテストを 1 つ足して決める。(b) の文言は「while the close waited」を外した「newer records stay as they are」で足りる。最後の回なので tech-debt の行 176 の後継(テストの欠け)に足す |

その他に数えない指摘が 3 つある。(1) `reactivateSeat(before, now []ManifestEvent, ...)` は同じ型の引数が 2 つ並び、入れ替えても型検査を通る。`reopenWorkspace(now, last, why)` は `now` が先頭で、兄弟の 2 つ(`before, now` の順)と並びが違う。入れ替えは「seat spawned again in another pane」のケースが skip の文言で落とすので守られている。(2) `compensateUnderLock(c, compensate)` は `c` を引数で受け、呼び出し側の closure も `c` を捕まえるので、`c` に書く経路が 2 つある。`compensate func(now []ManifestEvent, c *selfCompensation)` にすれば 1 つで済む。(3) `errorFor` は `len(c.skipped) > 0` で byHand を足すので、workspace が戻って座席と予約だけを skip した場合(`org started again with another reservation` と `org started again and disbanded again`)は、「recorded workspace ... open again, so running the command again retries the close」の後ろに「close it by hand」が続き、「打ち直す」と「手で閉じる」が並ぶ。後者は余計なだけで、誤りではない。`errorFor` の doc の「a retry may not find what is open」は、この場合には当てはまらない。

### cycle 2 の finding の現況

| C2 | 重さ | この回の判定 | 根拠 |
| --- | --- | --- | --- |
| C2-1 | LOW | 未修正(行 177 の(a)) | `spawn.go:345`、`:1081`、`:1088` は a919082a から変わっていない(`spawn.go` はこの回の差分にない) |
| C2-2 | LOW | 未修正(行 177 の(b)) | `spawn.go:1095` も同じ。AC1 の文言と quality-gates の行も変わっていない |
| C2-3 | LOW | 未修正(行 177 の(c)) | `spawn.go:1100` の `if !seat.Active` は残っている |
| C2-4 | LOW | 半分直った | `templates/base/ralph.toml:42-46` は 27fefc47 で直り、`withMainWorktreeOrgLimits` の条件(`--config` なし、`git-main-worktree`)と合う。`internal/cli/org.go:366` の `orgWideLimitsHelp` の「Only when --config is not given and the ledger is the main worktree's」は変わっていない(行 177 の(d)) |
| C2-5 | LOW | ほぼ直った | verify と test の cycle 2 が付録 A で V 番号と Test gaps の番号を残した。行 176 の(b)のカバレッジの値は関数名にならず、`idempotentRespawn` の値(90.9%)が更新されただけで、`reserveAgain` の 75.0% は a94c914f で書き換わった関数の値。`verbs_test.go:3085` の「verify V-2」も残る |
| C2-6 | LOW | 直った | 行 173 の(a)は「nothing warns when the path is typed」になり、見直しの欄と食い違わない |
| C2-7 | LOW | 直った | `AGENTS.md:90` は「in `envelope.go`, their counts and scope reservations in `reserve.go`」になり、`envelope.go:92`、`:107` と `reserve.go:233`、`:265` に合う |

cycle 1 の F-5 は a94c914f で直った(`reserveAgain` が書く前に台帳をロックの下で読み直す)。F-10(pane の経路が予約を戻さない)は `CloseDeferredSelfPane` が `reactivateSeat` だけを呼ぶままなので、行 172 の(b)に残る。F-1、F-3、F-11 は cycle 2 のとおり直っていて、F-2、F-4、F-6〜F-9 は `reserve.go`、`spawn.go`、`internal/cli/org.go` が差分にないので変わらない。

### Positive notes

- 補償の判断がロックの下の読み直しになり、規則ごとに、外すと落ちるケースがある(読みによる。mutation は回していない)。

  | 外すもの | 落ちるケース |
  | --- | --- |
  | ロック(`withManifestLock` を外して `fn` を直接呼ぶ) | `a spawn holds the manifest lock`(200 ms の間に書かれる `docs/` を見ずに古い予約を書く)、`NothingRestored_NamesTheManualClose` の lock の 2 ケース |
  | `reactivateSeat` の `current != seat` | `seat spawned again in another pane`、`org started again with another reservation` と `org started again and disbanded again` の leader |
  | `reopenWorkspace` の `current != last` | `herdr gave the id to another org` |
  | `reserveAgain` の `ActiveReservation(now) != nil` | `org started again with another reservation`、`a spawn holds the manifest lock` |
  | `reserveAgain` の `slices.Equal(...)` | `org started again and disbanded again` |
  | `errorFor` の skipped での byHand | skip の 4 ケースの `HasSuffix` |
  | `compensateUnderLock` の read の失敗 | `manifest unreadable at the second read` |

- ロックの中は read と append だけで、herdr の呼び出しを持たない。5 秒待って取れなくても、何も書かない側に倒れる
- ロックの順序のテストは決定的: 別の goroutine が `held` を閉じてからフックが返るので、補償は解放まで読めない。200 ms の保持は結果の向きを変えない。旧コード(写しで判断、ロックなし)なら古い予約を先に書くので、このテストは落ちる
- skip と失敗のエラー文は、`HasSuffix` で byHand の位置まで固定される。`--force` の経路はロックを取らず、a94c914f は触れていない
- `reserveAgain` の「限度と重なりを見ない」の doc は残り、行 172 の(c)(d)と一致する。`lastHerdrAgentName` を `before` から `now` に替えたのは、状態イベントが変わらない座席では結果が同じで、安全
- 27fefc47 の `AGENTS.md` と `templates/base/ralph.toml` の編集は、コードの条件に合う

### Coverage gaps

- テスト、mutation、gofmt、vet、`check-sync.sh`、`check-skill-sync.sh` は実行していない(指示による)。a94c914f はこの回の `/verify` と `/test` が初めて通す。上の削除の表は読みによる結論なので、`/test` に 7 行の mutation を頼む。新しいテストは goroutine とフックを使うので、`-race` でも回したい
- C3-4 の(a)(`--reserve` なしの立て直し)と、`reserveAgain` の append の失敗の枝(`verbs.go:1309-1311`。`compensation append fails` の 2 ケースの台帳には予約がない)には、テストがない。前者の欠けは行 176 に載っていない
- ロックのケースは同じプロセスの goroutine で、別プロセスの競合は見ていない(`flock` は fd ごとなので同じ意味だが、行 176 の(c)にある欠けと同じ)
- 本物の herdr で補償の窓を作った実測はない(cycle 1 から同じ。行 171 の(c)、行 176 の(e))
- skill の 4 面と help 文は読んだだけで、`ralph org stop --help` と `disband --help` は出していない

### Tech debt identified

この回は上限を 3 に上げた最後の回なので、C3-1〜C3-4 のうち直さないものは繰り延べにあたる。4 件とも挙動を変えず、コードのコメントとテストなので、直すとパイプラインの再実行になる。行 177(cycle 2 の文言の LOW)の続きとして 1 行にまとめる案を次に書く。足すのは `/sync-docs` の担当か、オペレーターの判断。

> org-limits-reserve cycle 3 の LOW(self-review の C3-1〜C3-4)。(a) `compensateUnderLock`(`internal/org/verbs.go`)の doc が、規則を「appends only what is still missing」と書くが、立て直された予約・座席・workspace id は「まだない」のに書かない。ロックの外の書き込み(`spawn.go` の `spawned`、`spawn_step`、`spawn_failed`、`org_workspace_created`、`Stop` の `stopped` / `stop_failed`、`Disband` の `org_workspace_closed` / `disbanded`)が read と append の間に入りうることも書かない(窓は read 1 回の長さ)。(b) `selfCompensation`、`add`、`errorFor` の doc は失敗を「append」に限るが、ロックと 2 回目の read の失敗も入る。(c) 補償を無条件に書く文(`internal/cli/org.go` の `closeDeferredSelf` の doc と `stop` / `disband` の help、`Disband` の doc、`/org` skill の 4 面、plan の AC15)が、台帳に新しい記録があると戻さない挙動を書かない。(d) `reserveAgain` の規則が座席・workspace の規則(最新のイベントが同じか)と違い、`--reserve` なしの立て直しで古い予約を戻す挙動にテストと doc の文がない。写しにすでにあった予約を「while the close waited」と呼ぶ文言も不正確。トリガー: `compensateUnderLock`、`reserveAgain`、`closeDeferredSelf` の次の変更、またはロックの外の書き込みが補償と競合した報告

`/sync-docs` に渡すものを次に挙げる。文書のずれは見ていないので、位置と a94c914f との関係だけを書く。

- 行 172 の(a)(F-5): a94c914f が直した。(a) を取り消し線にして、直した場所(`reserveAgain` の `before` / `now`)とテスト(`NewerRecordsWhileClosing_Stay` の `org started again with another reservation`、`org started again and disbanded again`、`a spawn holds the manifest lock`)を書く。Risk の欄の「(a) can swap a live org's reservation for an older one」と Trigger の欄の「(a): re-read the ledger just before the append ...」も外す。(b)(c)(d)は残る(pane の経路は予約を戻さず、補償は `spawnCapacityErr` を呼ばない)。見出しの「its reservation half has three gaps」は数え直す
- 行 170 の(a)(C2-1): 半分だけ直った。「写しで決めて古い座席を戻す」は直ったが、後半(`reactivateSeat` が 3 つの理由で `""` を返し、`errorFor` が「写しは何も要らなかった」として byHand なしで元のエラーを返す)は残っている。(a)全体を取り消し線にせず、後半だけに書き直す。「org-limits-reserve added `reserveAgain` as a third reader of the same copy」の文も外す。(c)(C2-5)は `verbs.go` が 1,875 行から 1,951 行になり、ブロックに `compensateUnderLock` が加わった。Trigger の「next change to ... `reactivateSeat`, `reopenWorkspace`, or `reserveAgain` ... the trigger is not spent」の次の編集が a94c914f なので、移さなかったことを書き足す
- 行 171 の(b): 「nothing re-spawns a seat inside the compensation window」は直った(`seat spawned again in another pane`)。C2-2 の側(同じ pane id を持つ古い `stopped` の座席)のテストは残る
- 行 176 の(a)(mutation X18): 「already reserved」の早期 return は消え、同じ判定が `ActiveReservation(now) != nil` になった。読みでは `org started again with another reservation` と `a spawn holds the manifest lock` が落とす。`/test` の mutation で確かめてから取り消し線にする。(b)は、カバレッジの値(69.2%、90.9%、75.0%)を関数名だけにする(C2-5 と同じ。`reserveAgain` の値は書き換わった)。`compensateUnderLock` のロックと読み直しの失敗の枝は新しいケースが通すので、(b)に書かない。`reserveAgain` の append の失敗の枝は残る
- 行 177: C3-1〜C3-4 の上の案を足す。(c)の C2-4 は `templates/base/ralph.toml` が直ったことを書いてあり、`orgWideLimitsHelp` が残ることも合っている
- skill の 4 面(`.claude/skills/org/SKILL.md:168`、`:169`、`:190-195` と 3 つの写し)と plan の進捗の行: C3-3 のとおり、skip の場合を 1 文足す

### Recommendation

- Merge: 可。CRITICAL、HIGH、MEDIUM はない。LOW は 4 件(C3-1〜C3-4)で、コメントと doc の言い方、テストの欠け。a94c914f はロックの入れ子も herdr をまたぐ保持もなく、3 つの補償の規則はそれぞれ、外すと落ちるケースを持つ(読みによる)。cross-review の cycle 2 の指摘(予約を写しで書き戻す)を直し、pane の経路もそろえた
- Follow-ups: C3-1〜C3-4 は最後の回なので直さず、上の 1 行にまとめて tech-debt へ送るのが現実的。`/test` に 7 行の mutation(上の表)と `-race` を頼む。`/sync-docs` に渡すもの: 行 170、171、172、176、177 の更新、skill の 4 面と plan の進捗(C3-3)
- cycle 2 の C2-6、C2-7 は直り、C2-4 と C2-5 は半分、C2-1〜C2-3 は行 177 に載ったまま。cycle 1 の F-5 は直り、F-10 は行 172 の(b)に残る
