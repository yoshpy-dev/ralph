# Self-review report: org-human-decision

- Date: 2026-11-22(cycle 2、run 2 と run 3。run 3 は 2 周目の verify のあとの 1 コミット(V8 の直し)だけを見た。cycle 1 の run 1 は、下の「Run 1」にそのまま残す)
- Plan: `docs/plans/active/2026-11-21-org-human-decision.md`
- Branch: `feat/org-human-decision`、base `main` の `9cd63d60`、HEAD `9290150f`(issue #366)。この 4 行は run 2 の時点のもので、run 3 の HEAD と範囲は「Run 3」の節に書く
- Reviewer: reviewer サブエージェント(`/self-review`)
- Scope: `git diff 9cd63d60..HEAD` 全体の diff の質。重点は cross-review cycle 1 の ACTION_REQUIRED #1(手で押した Enter で回復した人の決定が台帳に残らない)の 2 つの直し。`1a3a940a`(director skill の「人の決定を送る」の 2 と「答える順」の 2、`agent-messaging.md` とその雛形の写し、`leader.md`、`agent-messaging.md` のテスト)と、`9290150f`(headless director の role prompt `internal/org/prompts/director.md` とそのテスト)。テスト、静的解析、spec 適合、文書のずれの検査は行っていない(`/verify`、`/test`、`/sync-docs` の担当)。

## Run 3 (cycle 2, V8 fix)

- Date: 2026-11-22
- Scope: `git diff c6ec3427..6dc7c22f`(コミット `6dc7c22f` の 1 つ、5 ファイル +270 −13)。2 周目の verify の指摘 V8(`docs/reports/verify-2026-11-21-org-human-decision.md` の Run 2)の直し。`org.SendResult.HumanOrigin` を足し、人の決定(leader 宛ての `ORIGIN: human` の DECISION)の send が失敗したとき、`internal/cli/org.go` の 3 つの note(TextUnacknowledged、TextTyped、EnterUnacknowledged)の終わりに一文を足し、`ralph org send --help` に「記録は終了コード 0 の send だけ」の一文を足した
- HEAD: `6dc7c22f`
- Run 2 の指摘 R2-M1 と R2-L1 から R2-L5 は、この run の範囲(`c6ec3427` より前)にあるので、状態を見直していない。ただし R2-M1 の直し(`d78f379a`)が skill に置いた場合分けは、R3-M1 の比べる相手として読んだ

CRITICAL と HIGH はない。MEDIUM 1、LOW 2。run 2 の ID(R2-)と取り違えないよう、この run の ID には `R3-` を付ける。テストと静的解析は走らせていない(コードと文面を読んだだけ)。

### Evidence reviewed

- `git diff c6ec3427..6dc7c22f` の 5 ファイル全部(`internal/cli/org.go`、`internal/org/verbs.go`、`internal/cli/org_send_help_test.go`、`internal/cli/org_test.go`、`internal/org/send_origin_test.go`)
- `Send` の全体(`internal/org/verbs.go:484-739`)。`return` のすべてと、`result` という名前の変数がほかにないこと(名前付きの戻り値と衝突しないこと)。`checkSendOrigin`(`internal/org/send_origin.go:60-85`)の、DECISION 以外の ORIGIN を断る分岐
- CLI の `switch result.Progress`(`internal/cli/org.go:1492-1534`)の 4 つの note と、足した一文の組み合わせ
- skill の「人の決定を送る」の 2 の場合分け(`.claude/skills/director/SKILL.md:1220-1259`)と、`agent-messaging.md` の「A human's decision」(`:165-195`)、「How send types a message」の Ctrl+C の段落(`:135-150`)。CLI の note をこの 2 つと比べた
- 新しいテスト 3 つ(`TestOrgSendHumanDecisionNote_Text`、`TestOrgSend_HumanDecisionFails_NotesNotToSubmitByHand`、`TestOrgSend_HumanDecisionEnterPressedNotRecorded_NoteUnchanged`)と `TestOrgSendOrigin_SendResultHumanOrigin`、help のテストの 1 文。stub の `ORG_STUB_FAIL` が `$1:$2` で `pane:send-text` と `pane:send-keys` を拾うこと(`org_test.go:209`)、時間切れの場合の `1.1 s` / `2500 ms` / `1800 ms` が既存の `TestOrgSend_CtxExpiresDuringEnterDelay_NotesTextTypedOnStderr` と同じこと

### 確かめて問題がなかったこと

- 名前付きの戻り値と `defer` の形は正しい。`return SendResult{...}` は `result` に代入してから `defer` が走るので、`HumanOrigin` だけが足され、`Err`・`PaneID`・`Progress` は変わらない。`defer` は ORIGIN の断りの `return` の後ろにあるので、protocol 検証、貼り付けマーカー、ORIGIN の 3 つの断りは `false` のまま。それより後ろの `return`(dry run、seat なし、pane の lock、待ちの失敗、時間の足りない断り、3 つの失敗の進み具合、`sent` が書けなかった `EnterPressed`)は `true`。あとで置かれる `defer cancel()` と `defer unlock()` とは後入れ先出しで干渉しない
- `HumanOrigin` の doc の「watch gate と approval の断りは TASK だけで、TASK は `ORIGIN: human` を持たない」は、`checkSendOrigin` が DECISION 以外の ORIGIN を断るので成り立つ(`isTask` は `msg.Type == TASK`、`humanOrigin` は DECISION)
- 「記録は終了コード 0 の send だけ」は `Send` のコードと合う。`sent` を書く `appendEvent` は `typeAndSubmit` が `SendProgressEnterPressed` を返したあとの 1 か所(`verbs.go:718-737`)で、ほかの 3 つの進み具合は書く前に返る。足した一文を付けない `EnterPressed` の note(`sent` だけが書けなかった)は、元の「Do not send the message again」のままで正しく、テストが「足した文がない」ことを固定する
- 足した一文は、足す 3 つの note でだけ出る(`humanNote` は 3 つの `case` だけが使い、人の決定でないと空)。`%s` に渡すので、一文の中の `%` や `'` が書式にならない
- help の一文(`org.go:1437-1444`)は正確で、help のテストが空白を畳んだ形で固定する
- 直しを外すと落ちるか。`defer` を消すと `TestOrgSendOrigin_SendResultHumanOrigin` の `wantHuman: true` の 6 行が落ちる。`humanNote` を消すと `TestOrgSend_HumanDecisionFails_NotesNotToSubmitByHand` の 3 つの副テストが落ちる(人の決定の note が「ヘッダなしの note + 一文」と一致しない)。`EnterPressed` にも足すと `…EnterPressedNotRecorded_NoteUnchanged` が落ちる。ヘッダなしの DECISION の note が変わらないことも、同じ副テストの前半が見る。help から一文を消すと help のテストが落ちる

### Findings

| Severity | Area | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- |
| MEDIUM (R3-M1) | maintainability(言い直した規則が、skill が足した安全の条件を落とす) | 足した一文は「入力欄に文があれば、空にして、同じ文を `ralph org send` で送り直す」の 1 本道だけを言う。同じ規則の skill 側は 4 つの項を上から見る形で、R2-M1 の直し(`d78f379a`)は先頭に「ほかの画面(動作中の行、選択の画面、別の文)が出ていれば、何も送らず人に言う。入力欄を空けるようにも頼まない(手番の途中の Ctrl+C は手番を止める)」を置いた。一文にはこの前提がないので、leader が動作中の画面(send が文を打ったあとに Monitor の知らせで手番が始まる。skill 自身が書く道)でも「if the text is in the input box, clear the box instead」と読める。note は stderr の最初に読まれる面なので、skill の長い場合分けより先に従われやすい。同じ一文には、さらに 2 つの欠けがある。(a) 誰が空けるかを言わない。skill と rule は director がキーを押さず人が空けるとする(`agent-messaging.md:139` の「Clearing the box is a step for the human」)が、字面の「clear the box」は director が `herdr pane send-keys … ctrl+c` を自分で押す読みも許す。(b) 空けたあとに pane を読み直して入力欄が空なことを確かめる条件(skill の 2 つ目の項、R2-L4 の直し)がなく、「then send the same text again」が続く(TextUnacknowledged と TextTyped の元の note の「Check … before sending again」は、空けたあとの確認とは別)。EnterUnacknowledged で Enter が届いていた場合(leader が動作中)は、元の「do not send the message again」が再送を止めるが、決定は届いたのに台帳にないという、skill の 4 つ目の項の知らせは note のどこにもない。R2-M1、R2-L1、R2-L5 と同じ形(規則を 1 つの項に言い直して、ほかの項と、元の規則が持つ条件を落とす)で、直したばかりの skill と食い違う面が CLI の note に 1 つ増えた | `internal/cli/org.go:1356-1359`(定数)、`:1488-1491`(足す条件)、`:1493-1525`(3 つの note)。`.claude/skills/director/SKILL.md:1228-1232`(1 つ目の項)、`:1233-1239`(2 つ目の項)、`:1246-1259`(4 つ目の項)、`:1164-1167`(手番が始まる道)。`.claude/rules/ralph/agent-messaging.md:135-150` | 一文に条件を足す。例: 「…clear the box instead (a human does it: Ctrl+C stops a turn in progress, so only while the pane shows no working line or dialog), read the pane again, and send the same text again only into an empty box; otherwise press nothing and tell the human.」。または一文を短くして、場合分けの置き場所を指す(`agent-messaging.md` の「A human's decision」)。どちらも `orgSendHumanDecisionNote` の文字列と、それを固定する 2 つのテスト(`TestOrgSendHumanDecisionNote_Text` の写しと、連結を見る `…NotesNotToSubmitByHand`)が変わる |
| LOW (R3-L1) | readability(同じ行の中の 2 つの文が逆) | TextTyped の元の note は「clear or submit it there before sending again」、EnterUnacknowledged の元の note は「Only if the text is still sitting in the input box, submit or clear it there」と言う。人の決定では、同じ行の終わりに「do not submit it in the pane by hand: … clear the box instead」が付く。あとの文が「instead」で上書きするので読めるが、1 行に「submit してよい」と「submit するな」が両方あり、前半だけを読む人(長い note の途中で止まる、指示を拾う LLM)は元の指示に従う。テストが「ヘッダなしの note + 空白 + 一文」の連結を固定するので、元の note の「or submit」を人の決定のときだけ落とす直しは、テストの形も変える | `internal/cli/org.go:1509-1514`(TextTyped)、`:1518-1525`(EnterUnacknowledged)、`internal/cli/org_test.go` の `want := plainNote + " " + orgSendHumanDecisionNote` | R3-M1 を直すときに合わせて、人の決定のときは元の note の「submit」の語を出さない(別の文面を選ぶか、元の文を一文で置き換える)。直さないなら、一文の最初を「Unlike the above, …」の形にして、上書きであることを字面に出す |
| LOW (R3-L2) | readability(doc の言い回し) | `SendResult.HumanOrigin` の doc の最後の文は 1 文が 6 行で、「tell the sender of a send that failed after it typed that only a send that returns no Err records the decision」は主語と目的語がたどりにくい。また「false for every other text and for every refusal」は、続く列挙(protocol 検証、貼り付けマーカー、ORIGIN の断り、TASK の watch gate と approval)のとおり「ORIGIN の確認より前の断り」だけを指すが、`Send` の doc と `sendSeatNotFoundErr`・pane の lock・時間の足りない断りなど、確認より後の断りは `true` を返す。「every refusal」のままだと、あとの断り(「nothing was typed」と言うもの)も `false` と読める | `internal/org/verbs.go:252-264`。`TestOrgSendOrigin_SendResultHumanOrigin` の doc(「a later error such as a leader not found」は `true`)は正しいので、doc 同士が食い違う | 「false for every other text, and for a refusal that comes before the check (…)」と書き、最後の文を「A caller reads it on an error to decide what to tell the sender」のように 2 文に分ける |

### Positive notes

- 直しは `Send` の 20 か所ほどの `return` に触れず、名前付きの戻り値と 1 つの `defer` で全部に足した。`defer` の置き場所(ORIGIN の断りのすぐあと)が「断りは `false`」と一致し、`Send` と `HumanOrigin` の両方の doc がその理由を言う。テストの表が、断り 4 種と、確認より後の失敗(seat なし)の両側を 1 つの表で固定する
- note の文は定数 1 つに集め、`humanNote` を 3 つの `case` で使う。`EnterPressed` に足さない理由(`sent` だけが書けなかった)が定数の doc とテストの両方にある
- CLI のテストは、ヘッダなしの DECISION と人の決定を同じ台帳の同じ leader に送り、note の差が「空白 + 一文」だけであることを見る。note の他の語を変えても、人の決定の差だけが残る形で、元の note の文言の変更に強い
- 時間切れの副テストは、既存のテストと同じ秒数(`1.1`、`2500`、`1800`)で、同じ余裕(typing の前の待ちが 700 ms を超えると何も打たずに断られ、note の行が 0 本になって落ちる)。2 回送るので余裕は 2 回分だが、既存のテストと同じ前提で、この run の指摘にはしていない

### Tech debt identified

| Debt item | Impact | Why deferred | Trigger to pay down | Related plan/report |
| --- | --- | --- | --- | --- |
| R3-M1、R3-L1(CLI の note の一文が skill の場合分けを 1 本道に言い直す。元の note の「or submit」と同じ行で逆を言う) | headless director や端末の人が、動作中の leader の入力欄を空けようとして手番を止めうる。決定が届いたのに記録がない場合の知らせが note にない | cycle の上限が 2/2。直すのは `internal/cli/org.go` の文字列と 2 つのテストで、コードの変更なので pipeline の回し直しが要る | `orgSendHumanDecisionNote` の次の変更、または人の決定の送り直しで手番が止まった報告 | `docs/reports/verify-2026-11-21-org-human-decision.md` の V8、この report の R3-M1 |

_(直さないと決めたら、上の行を `docs/tech-debt/README.md` に足す。この run は report と plan の 2 つだけのコミットを頼まれているので、register は触っていない。R3-M1 を直す cycle で R3-L1 と R3-L2 も直せば、この行はいらない。)_

### Recommendation

- Merge: yes(CRITICAL と HIGH はない)。V8 の直しは目的(手で押した Enter では記録が残らないことを、失敗した send の note と help に出す)を果たし、`defer` の形と `HumanOrigin` の値は正しく、テストは直しを外すと落ちる。R3-M1 は note の一文に条件を足せば直る
- Follow-ups:
  - R3-M1 と R3-L1 は `orgSendHumanDecisionNote` の文字列とその 2 つのテストを変える Go の直し。cycle の上限は 2/2 なので、直すなら上限を上げて `/self-review` から回し直す(`post-implementation-pipeline.md`)。直さないなら `/pr` で known gap に残す。R2 の R2-L1 から R2-L5 も同じ扱いで、同じ commit に入れられる
  - R3-L2 は doc の言い直しだけで、コードの動きは変わらない
- Known gaps: note の一文が、実機の動作中の leader の pane に対して正しいか(Ctrl+C の動き)は、`agent-messaging.md` の記述に頼って確かめていない。run 2 と同じ

---

## Run 2 (cycle 2)

CRITICAL と HIGH はない。MEDIUM 1、LOW 5。run 1 の指摘の ID(M1、M2、L1 から L4)と取り違えないよう、この run の ID には `R2-` を付ける。

### Evidence reviewed

- `git show 1a3a940a` と `git show 9290150f` の全文(skill、rule、`leader.md`、`director.md`、テスト)
- `Send`(`internal/org/verbs.go`)の `sent` を書く位置(`appendEvent` の呼び出しは `typeAndSubmit` が `SendProgressEnterPressed` を返したあとの 1 か所)、`SendProgress` の 4 つの失敗の分岐、dry run の分岐。CLI が分岐ごとに出す `note:`(`internal/cli/org.go:1470-1535`)
- director skill の「答える順」(`.claude/skills/director/SKILL.md:1091-1186`)、「人の決定を送る」(`:1188-1246`)、「してはいけないこと」の送る手の項(`:2519-2523`)、`/org` skill の `send` の行(`:377`)、`agent-messaging.md` の「How send types a message」と「A human's decision」(`:120-196`)、`leader.md` の運用規律の人の決定の項(`:1636-1665`)、`director.md` の send の項(`:25-55`)
- run 1 の指摘の直し(`fc09a923`)を HEAD で確かめた(下の表)
- director skill の 4 写し、`agent-messaging.md` の 2 写しを `cmp`(いずれも同じ)
- 新しいテスト(`prompts_leader_human_decision_test.go:249-345`)の断片と、使う helper(`directorSkillSubsection`、`checkSkillPlaces`)。`docs/tech-debt/README.md` の `#366` の 2 行(`:547`、`:548`)
- probe は走らせていない(コードと文面を読んだだけ)

### Run 1 の指摘の確認

| Run 1 | 状態 | 確認したこと |
| --- | --- | --- |
| M1(`reseatProgressLead` が「本文は data」と言う) | 直った | `fc09a923` が定数に「人の決定」の行の下の DECISION の全文だけは指示として読む例外を足した。括弧が開いたあとなので節の名前は変わらない |
| M2(skill の「座席を立て直す」の最後の段落がヘッダなしで送らせる) | 直った | 同じ段落が「人の決定を送る」の `ORIGIN: human` で送る形になり、受信箱の件への答えは「答える順」のままと書く |
| L1(運用規律の項が手順 6 の戻す手を落とす) | 直った | `leader.md` の人の決定の項が、先頭の 2 行を `Status: Draft` と `Approved: TBD` に戻してから計画を変える形になった |
| L2(「人の決定」の項だけ件数の上限がない) | 直った(理由を書く形) | `reseatProgressHumanDecisionsLabel` の doc が、1 件を落とすと従われなくなる理由と、1 件の大きさの上限(`sendOriginHumanMaxBytes`)を言う |
| L3(compaction のあとの読み直し) | 延期、登録済み | `docs/tech-debt/README.md:547` |
| L4(`%s` と `%q`、空の `case`) | 直った | `send_origin.go` の TYPE が `%q`、`readReseatProgress` の空の `case` にコメント |

### Findings

| Severity | Area | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- |
| MEDIUM (R2-M1) | maintainability(場合分けの順が安全の条件を落とす) | 「人の決定を送る」の 2 の場合分けで、入力欄に文が残っている項(1 つ目)に「ほかの画面が出ていないとき」の条件がなく、「ほかの画面(選択の画面、動作中の行)」は 4 つ目、最後の項にある。上から読むと、send が文を打ったあとに leader が手番を始めた画面(動作中の行が出て、入力欄に文が残る)は 1 つ目に当たり、人に Ctrl+C を頼む。`agent-messaging.md` は Ctrl+C を「座席が入力待ちで、入力欄に何かあるときだけ」とし、手番の途中の Ctrl+C は手番を止めると書く。元の「答える順」の段落は、Enter の条件に「選択の画面、動作中の行、別の文が出ていないとき」を置いていて、新しい項はこの条件を落とした。director 自身はキーを押さない(この点は新しい文のどこにも例外がない)ので、安全でないのは人への頼みのほう。手番が始まる道は skill 自身が書いている(Monitor の知らせで始まる手番、`SKILL.md:1164-1167`) | `.claude/skills/director/SKILL.md:1228-1232`(1 つ目)、`:1240`(4 つ目)、`:1152-1155`(元の条件)。`.claude/rules/ralph/agent-messaging.md:146-150`(Ctrl+C の条件)。1 つ目の `(Ctrl+C を 1 回。…「How send types a message」)` の注意には、その節を開いた director だけが届く。`TestDirectorSkill_SendsAHumanDecisionWithTheOriginHeader`(`prompts_leader_human_decision_test.go:249`)は 4 つの項を今の順で固定する | 「ほかの画面(選択の画面、動作中の行)が出ていれば、何も送らず、人に言う」を 1 つ目に移す。または 1 つ目の条件に「動作中の行も選択の画面も出ていなくて」を足す。4 写しは `scripts/sync-skills.sh` で同じにし、上のテストの断片の順も合わせる |
| LOW (R2-L1) | maintainability(言い直した規則が元の枝を落とす) | skill の場合分けは 4 つ(空けてもらう、空なら送り直す、届いていたら人に伝える、ほかは人に言う)。同じ規則を言い直す 2 か所は 1 つ目の道だけを言う。(a) `director.md` は「キーを押さずに人に leader の入力欄を空けてもらい、同じファイルを送り直す」と言い切る。「のとおり」で skill を指すが、近い文が 1 つの道に言い切っているので、leader が決定を受け取っていた画面でも、空の入力欄でも、人に空けてもらって送り直し、同じ決定が 2 回届く。(b) `agent-messaging.md` の「it presses no key」は範囲が書かれておらず、skill が許す、終了コード 0 のあと入力欄に残った DECISION への Enter(`SKILL.md:1213-1214` の「扱いも同じ」)と字面で食い違う | `internal/org/prompts/director.md:44-47`、`.claude/rules/ralph/agent-messaging.md:178-184`(と雛形の写し)、`.claude/skills/director/SKILL.md:1213-1214,1228-1240`。run 1 の M2 と同じ形。`9290150f` が足したテストの断片の上にある doc(`prompts_send_unsubmitted_test.go:67-77`)は「それ以外は Enter を押さず、件を resolve せず、人に伝える」だけを言い、足した人の決定の文を言わない | `director.md` は末尾の句を「キーを押さずに、見えた画面に合わせて同じファイルを送り直すか人に伝える」の形にする。`agent-messaging.md` は「a send that exits 1 with a note」に範囲を絞り、場合分けは skill を指す |
| LOW (R2-L2) | maintainability(規則の根拠が片方にしかない) | `leader.md` の「全文が同じ人の決定がもう 1 度届いたら、記録のために送り直されたもの」は、同じ全文を同じ決定と読む。これが成り立つのは、skill が本文に「人が決めた日時」を書かせるので別の決定の全文は同じにならないから。leader の prompt はその前提を言わず、skill の側も「日時」とだけ言う。日付だけが書かれた同じ文が同じ日に 2 回届く(「gate をやり直す」を 2 回)と、2 回目が送り直しと読まれて何もされない。また leader が compaction で 1 回目を忘れる(run 1 の L3 の穴)と、送り直しが新しい決定として読まれ、済んだ手が繰り返される。後者は規則の限界で、前者は 1 文で防げる | `internal/org/prompts/leader.md:1640-1643`、`.claude/skills/director/SKILL.md:1198-1200`、`docs/tech-debt/README.md:547` | `leader.md` に「同じ」の根拠を書く(「本文に人が決めた日時があるので、全文が同じなら同じ決定です」)。skill の 1 は「日時は時刻まで書く」にする |
| LOW (R2-L3) | maintainability(register の行を直しが偽にした) | `docs/tech-debt/README.md` の `Test gaps of feat/org-human-decision (#366)` の (a) は「`agent-messaging.md` の「A human's decision」の節を固定するテストはなく、段落を消しても言い直しても何も落ちない」、トリガは「段落の次の変更」。`1a3a940a` が段落を変え、`TestAgentMessagingRule_OriginHumanIsRecordedOnlyByASendThatExits0` で両方の写しの一部(記録は終了コード 0 の send だけ、director は Enter を押さず送り直す)を固定した。(a) の文は偽になり、トリガは半分だけ払われた。ヘッダ、断られる形、16 KiB の文は固定されていない | `docs/tech-debt/README.md:548`(a)、`prompts_leader_human_decision_test.go:313-345` | (a) を「ヘッダ、断られる形、16 KiB の文だけが固定されていない」に狭め、トリガも同じにする |
| LOW (R2-L4) | maintainability(送り直しの前に確かめない) | 1 つ目の項は「空いたと聞いたら、同じファイルを送り直し、pane をもう一度読む」で、送る前に pane を読み直さない。残った文のあとに 2 本目が打たれると、Enter で 1 つのメッセージとして送られる(`Send` の doc の TextTyped の段落と、`agent-messaging.md` の「read the pane with `ralph org read` before sending again」)。leader には全文が 2 倍の決定が届き、「全文が同じ」の規則にも当たらず、台帳には 1 本分の全文が残る。人の「空いた」だけが根拠だと、codex の `[Pasted Content <N> chars]` を Ctrl+C で消せるかが確かめられていない(`agent-messaging.md:146-147`)ので、その場合に起こりうる | `.claude/skills/director/SKILL.md:1228-1232`、`.claude/rules/ralph/agent-messaging.md:135-147`、`internal/org/verbs.go:430-433` | 「空いたと聞いたら、leader の pane を読んで入力欄が空なことを確かめてから、同じファイルを送り直す」 |
| LOW (R2-L5) | maintainability(CLI の note と skill が逆を言う) | `ralph org send` の note は、`EnterUnacknowledged` と `EnterPressed` のとき「送り直さない」と言う(`do not send the message again`)。3 つ目の項は、leader が決定を受け取っていたら、人が記録を望めば送り直すと言うが、note を上書きすることも、上書きの理由(leader が 2 回目を新しい決定として扱わない)も書いていない。`EnterPressed` の note は「sent の記録だけが書けなかった」で、書けない原因が続いている(台帳の書き込みの失敗)なら、送り直しは同じ決定を 2 回届けたうえで記録も残せない | `internal/cli/org.go:1493-1511`、`.claude/skills/director/SKILL.md:1235-1239`、`internal/org/verbs.go:700-719`(`appendEvent` の失敗が `EnterPressed` の error になる) | 3 つ目の項に、人の決定では note の「送り直さない」より記録のための送り直しを優先すると書く。error が `the sent event could not be recorded` のときは、台帳が書けないことを人に伝え、送り直さない |

### 質問への答え

- 直しは穴を塞ぐか。塞ぐ。director が人の決定に手で Enter を押す道は 3 か所(skill の「答える順」の 2、「人の決定を送る」の 2、headless director の role prompt)で止まった。台帳に残らない配達で残るのは、`EnterPressed` で台帳の書き込みが失敗した 1 つで、3 つ目の項が人に伝える形で扱う。終了コード 0 のあと入力欄に残った DECISION への Enter は残るが、`sent` は終了コード 0 の時点で書かれているので記録は残る(submit を確かめられなかった send も終了コード 0 で `sent` を書く。`verbs.go:700-720`、`org.go:1524`)
- 場合分けは完全か、安全か。`Send` が終了コード 1 で note を出す 4 つの状態(`org.go:1470-1512`)は、TextUnacknowledged が 1 つ目か 2 つ目、TextTyped が 1 つ目(画面が入力欄でなければ 4 つ目)、EnterUnacknowledged が 1 から 4 のどれか、EnterPressed が 3 つ目に当たる。note を出さない終了コード 1(何も打っていない)は、「答える順」の 2 の送り直しに任せる。穴は順だけ(R2-M1)で、director がキーを押さない点には例外がない
- leader の「全文が同じ人の決定は新しい決定ではない」は妥当か。向きは安全(受け取り済みを繰り返さない側)で、skill が本文に日時を入れさせるので、送り直し以外で全文が同じになることはまずない。弱点は R2-L2 の 2 つ
- `Send` について誤った記述はあるか。ない。`sent` は `typeAndSubmit` が `SendProgressEnterPressed` を返したあとの 1 か所(`verbs.go:700-704`)で書かれ、ほかの 3 つの進み具合(`:653-692`)は書く前に返る。dry run は `sent` を `Text` なしで書き(`:512-519`)、`sentHumanText` を通るのは本物の送信だけなので、「終了コード 0 の send だけが決定を記録する」は本物の送信について正しい。「手で押した Enter は何も記録しない」は、`herdr pane send-keys` が台帳に触れないことと合う

### Positive notes

- 文面の直しは、skill(2 か所)、rule、leader の prompt、headless director の prompt の 5 つの面に同じ向きで入り、4 つの skill 写しと 2 つの rule 写しは `cmp` で同じ
- 新しいテストは実ファイルを読み、断片を順に固定する(定数と自分を比べない)。`TestAgentMessagingRule_…` は root と雛形の両方を見る。`9290150f` のテストは、既存の `TestRenderDirectorPrompt_SendTextTypedNotSubmittedReadsTheLeadersScreen` の断片に 2 つを足す形で、描画の組み合わせごとに走る
- cross-review の候補(打ち直さずに記録する道をコードに足す)を採らず、文面だけで塞いだ。入力欄に残ったまま押されない決定を記録してしまう害を避けている(triage の理由)
- run 1 の M1、M2、L1、L2、L4 の直しは、HEAD でも保たれている(上の表)

### Tech debt identified

| Debt item | Impact | Why deferred | Trigger to pay down | Related plan/report |
| --- | --- | --- | --- | --- |
| R2-L1 から R2-L5(場合分けを言い直した 2 か所が 1 つの道だけを言う。「全文が同じ」の根拠が leader の prompt にない。register の (a) の文が偽。送り直しの前に pane を読み直さない。CLI の note と skill が送り直しで逆) | 文面の食い違いだけで、動作は変わらない。R2-L1 と R2-L5 は、headless director が決定を 2 回届けたり、記録のない送り直しをしたりする | cycle の上限が 2/2。直すのは skill、rule、prompt の文面で、post-implementation pipeline の回し直しが要る | skill の「人の決定を送る」の次の変更、または人の決定の送り直しで同じ決定が 2 回届いた報告 | この report の R2-L1 から R2-L5 |

_(直さないと決めたら、上の行を `docs/tech-debt/README.md` に足す。この run は report と insight event の 2 つだけのコミットを頼まれているので、register は触っていない。R2-M1 を直す cycle で R2-L1 から R2-L5 も直せば、この行はいらない。)_

### Recommendation

- Merge: yes(CRITICAL と HIGH はない)。R2-M1 は skill の項の順を入れ替えるだけで直せる。cycle の上限は 2/2 なので、直すなら上限を上げて full pipeline を回し直す(`post-implementation-pipeline.md`)。直さないなら、`/pr` で known gap に残す
- Follow-ups:
  - R2-M1 は 1 か所の順の入れ替え(skill の 4 写しとテストの断片の順)。R2-L1 から R2-L5 は 1 文ずつの直しで、同じ commit に入れられる
  - 直す場合、R2-L3 の register の行は文書だけの直し。R2-L2 は `leader.md` を変えるので、テストの断片(`TestRenderRolePrompt_Leader_HumanDecisionBulletSaysWhatTheLeaderDoes`)も変わる
- Known gaps: 実機の leader の pane で、動作中の leader に Ctrl+C を送ったときの動きは、`agent-messaging.md` の記述に頼って確かめていない。「キーを押さず人に空けてもらう」ので、人が席にいない間は、入力欄に残った決定が、その leader への後の send(受信箱の件への DECISION を含む)を止める。これは triage が受け入れた選び方(押されない決定を記録しない)の裏側で、この run の指摘にはしていない

---

## Run 1 (cycle 1、履歴)

この下は run 1(2026-11-22、HEAD `91ad93a1`)の report を、見出しの階層を 1 つ下げた以外は変えずに残したもの。

- Date: 2026-11-22(cycle 1、run 1)
- Plan: `docs/plans/active/2026-11-21-org-human-decision.md`
- Branch: `feat/org-human-decision`、base は `origin/main` の `09d27450`、HEAD `91ad93a1`(issue #366)
- Reviewer: reviewer サブエージェント(`/self-review`)
- Scope: `git diff origin/main...HEAD` の 21 ファイル(+1721 −54)のうち、コードと文書の diff の質だけ。コードは S1 `fd7e9d9e`(`internal/org/send_origin.go`、`verbs.go`、`manifest.go`、`internal/cli/org.go` の help)、S2 `e1a9ad18`(`internal/org/reseat_progress.go`、`feature.go` のコメント)、S3 `1daf1591`(`internal/org/prompts/leader.md`、director skill の 4 写し、`agent-messaging.md` と雛形の写し、README)。テストファイルは、本番コードの seam と合うかだけ見た。テスト、静的解析、spec 適合、文書のずれの検査は行っていない(`/verify`、`/test`、`/sync-docs` の担当)。

### Evidence reviewed

- `internal/org/send_origin.go` 全体(108 行)と、`protocol.Parse`・`HeaderValues`・`splitHeaderLine`(`internal/org/protocol/protocol.go:100-182`)との突き合わせ。`Send`(`verbs.go:480-505`)の中の呼び出し位置と、dry run の分岐、`Details` の組み立て(`verbs.go:694-704`、`truncateForDetails`)
- 断りの順序: 非 raw は `Validate` のあと、貼り付けマーカーの確認の直後、watch gate・dry run の分岐より前。`--raw` と `--dry-run` も同じ位置を通る
- 16 KiB の上限の根拠: `readJSONLines` の buffer が 1 MiB(`manifest.go:342`)であること、JSON が 1 byte を最大 6 byte に書くこと(`<` の escape と、不正な UTF-8 の 1 byte の置換文字の escape が 6 byte)。16384 × 6 = 96 KiB で 1 MiB より十分下
- `reseat_progress.go` の diff 全体と、section の組み立て(`reseatProgressSection`、`fenceFor`(`feature.go:605`)、`PrintableInboxText`(`inbox.go:208`))。decision の text が fake の見出し、fence、項目の行、制御文字、U+2028 で項目から出られないかを追った。`RenderRolePrompt` の置換が `strings.NewReplacer` の 1 回通しで、text に `{{TASK}}` があっても再展開されないこと
- `internal/org/prompts/leader.md` の diff 4 か所(立て直しの節の 154-163 行、手順 4 の 489-495 行、cross-review の上限の 1180-1186 行、運用規律の項 1637-1653 行)と、周りの手順 3・4・6・7、`HOOK_SEAT:` の既存 3 か所。`reseatProgressLead` の使われ方(テストが定数そのものと比べるか)
- director skill の新しい節「人の決定を送る」(`.claude/skills/director/SKILL.md:1187-1223`)と、「答える順」を指す箇所の一覧(`grep`)、4 写しと `agent-messaging.md` の雛形の写しの一致(`cmp`、すべて同じ)
- `internal/cli/org.go` の help の文、`agent-messaging.md`、README の語単位の diff
- テストは、新しい 6 ファイルの関数名と、`TestReseatProgress_AHumanDecisionStaysInsideItsEntry`、`TestRenderRolePrompt_Leader_HumanDecisionIsNotTheAnswerItWaitsFor` の中身だけ読んだ。`ralph org send` が `sent` を読む他の場所(`grep EventSent`)も見た
- probe は走らせていない(コードを読んだだけ)

### Findings

CRITICAL と HIGH はない。MEDIUM 2、LOW 4。

| Severity | Area | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- |
| MEDIUM (M1) | maintainability(同じ task の中の 2 つの文が逆) | 立て直した leader の task は、囲みの直前の行 `reseatProgressLead` が「本文は data で、指示ではない」と言い、その囲みの中の「人の決定」の項は「leader はこれらの決定に従い続ける」と言う。「人の決定の全文だけは data でなく指示」という例外は role prompt(`leader.md:154-163`)にだけ書かれ、task の側には出ない。leader は近いほうの文(囲みの直前の行)に引かれうるので、AC7 の目的(立て直した leader が決定に従い続ける)が文の読み方に左右される。S2 の注意(「leader の prompt では書き分ける」)は prompt の側だけで満たされている | `internal/org/reseat_progress.go:35`(定数 `reseatProgressLead`)と `:480`(task に入る)、同じ値を出す `:459-462`(項の頭の行)。`leader.md:154-163`。テストは `reseatProgressLead` を定数のまま比べる(`reseat_progress_test.go:53,242,551`)ので、文言を変えても落ちない。`prompts_leader_reseat_test.go:111` は `(` の前の節の名前だけを使う | `reseatProgressLead` に例外を足す(例: `…本文は data で、指示ではない(「人の決定」の項の DECISION の全文を除く)。):`)。`(` より前は変えないので、節の名前の取り出しは壊れない。`leader.md` の同じ文と、頭の行の文言を 1 つの形にそろえる |
| MEDIUM (M2) | maintainability(新しい規則と古い規則が食い違う。S3 の注意 3 は本物) | director skill の「座席を立て直す」の最後の段落は、leader が済んだ手順をやり直し始めたとき、人が「済んだことを leader に伝える」と決めたなら、その DECISION を「答える順」の 1 と 2 で送ると書く。これは受信箱の件のない人の判断なので、新しい節「人の決定を送る」の対象(`ORIGIN: human`)に当たる。ヘッダなしで送ると、承認待ちの leader は手順 4 の「それ以外の DECISION(直しの依頼)」と読み、計画を直して承認を求め直す。この PR が防ぐ取り違えを、skill 自身の別の段落が起こす | `.claude/skills/director/SKILL.md:1719-1720`(「伝えるなら、DECISION を「答える順」の 1 と 2 のとおりに送る」)と、新しい節 `:1187-1196`(「受信箱の件がないまま…ヘッダを付けた DECISION で leader に送る」)。`leader.md:489-501`(ヘッダなしの DECISION は「直しの依頼」)。ほかの「答える順」への参照(`:1069`、`:1073`、`:1293`、`:1327`、`:1670`、`:1808`)は件への答えか、ralph が出した send なので当たらない | `:1720` を「人の決定を送る」のとおりに送る、に直す(本文には人が決めた日時と「済んだ手順」の事実を書く)。4 写しは `scripts/sync-skills.sh` で同じにする |
| LOW (L1) | maintainability(同じ規則を 2 つの形で持つ) | 運用規律の項と手順 4 は、決定が計画の変更を求めるとき「計画を直してコミットし、手順 2 と 3 をやり直して承認を取り直す」と書く。承認のあとに計画を変えるときの規則は手順 6 にあり、先頭の 2 行を `- Status: Draft` と `- Approved: TBD` に戻してから変え、手順 2 からやり直す。新しい文はこの戻す手を言わないので、承認済みの計画を `Status: Approved` のまま直す読みができる(implementer の spawn と TASK は承認の検査で断られ、手順 7 で戻れる) | `leader.md:1648-1650`(運用規律)、`:493-495`(手順 4。こちらは承認前なので影響しない)、`:545-551`(手順 6) | 運用規律の項の該当の文を「手順 6 のとおり先頭の 2 行を戻して計画を直し、手順 2 からやり直す」に直す。テスト(`TestRenderRolePrompt_Leader_HumanDecisionBulletSaysWhatTheLeaderDoes`)が固定する文も合わせて変わる |
| LOW (L2) | maintainability | 立て直しの経過の「人の決定」の項だけ、件数の上限がない。受信箱の件(20)、RESULT(5)、commit(20)は上限と「残りは件数」の形を持つ(`reseatProgressMaxInbox` の組)。1 件は最大 16 KiB で、人の決定が増えるほど task が伸びる。実際は数件だが、規則が 1 つの項だけ違う | `internal/org/reseat_progress.go:459-473`(上限なしのループ)、`:55-60`(他の 3 つの上限の定数) | 新しい N 件(例 10)に切って「ほか n 件(古いもの)」の行を足す。または上限を置かない理由(決定は落とすと守られなくなる)を `reseatProgressDecisionMaxBytes` の隣に 1 文書く。後者なら、合計の大きさの目安も書くとよい |
| LOW (L3) | maintainability(S3 の注意 4 は本物。deferred) | compaction のあとに leader が人の決定を読み直す手段がない。`sent` を読む場所は立て直しの進捗の節、`approve_resolve`、`watch` だけで、leader が打てる動詞に決定を出すものはない(`ralph org report` は `sent` を載せない)。leader.md は他の所で「compaction のあとも同じものが出る」ように値を git から読ませている(`:662`、`:1138`)が、人の決定は、座席の立て直しのときしか戻らない。1 回きりの決定(座席の設定を変える)は届いたときに済むので害は小さく、「今後は X しない」のような続く決定が compaction で落ちる | `internal/org/prompts/leader.md:1651-1653`(「立て直されたあとは…載ります」)。`grep EventSent internal/org` は `approve_resolve.go`、`reseat_progress.go`、`verbs.go`、`watch.go` の 4 つ | この PR では直さない(issue の範囲は届けることと立て直しの引き継ぎ)。tech-debt に 1 行足す。直し方の候補: leader が届いたとき `.harness/state/human-decision-{{ORG_ID}}.txt` に要点を書き、compaction のあとに読み直す(cross-review の上限のファイル `.harness/state/xreview-{{ORG_ID}}-cap` と同じ形)。または決定を返す読み取りの動詞 |
| LOW (L4) | readability | 小さな 2 点。(a) `checkSendOrigin` の TYPE の断りの文は `msg.Type` を `%s` で出す。同じ関数の ORIGIN の値と宛先は `%q` で、`--raw` の TYPE は利用者の文字をそのまま持てる。(b) `readReseatProgress` の `switch` の最初の `case ev.Event != EventSent:` は本体が空で、読み手は「何もしない、あとの `continue` に落ちる」と推し量る | `internal/org/send_origin.go:74`(`%s`)、`:76,79`(`%q`)。`internal/org/reseat_progress.go:212-216` | (a)は `%q` にそろえる。(b)は 1 行のコメント(`// not a sent event: nothing to read`)か、`if ev.Event == EventSent { … }` の形にする |

### S3 の注意の判断

- 注意 3(director skill の「座席を立て直す」の最後の段落)は本物で、M2 にした。人が決めた「済んだことの伝達」は件のない人の決定なので、ヘッダを付ける側に寄せるのが一貫する。逆に、ralph が reseat の最後に出す DECISION(`:1665-1671` の 6。leader に今の TASK を送り直すよう頼む)は人の決定ではないので、ヘッダなしのままでよい
- 注意 4(compaction のあとの読み直し)は本物で、L3 にした(deferred、tech-debt に 1 行)。この PR の受け入れ条件の外で、AC7 は立て直しだけを求めている

### Positive notes

- ヘッダの読み方は `protocol.HeaderValues` を使い、新しい解析を持たない。キーの大文字小文字、前後の空白、`ORIGIN :`、ヘッダの外(本文)の `ORIGIN:` の扱いが `Parse` と必ず同じ。`Parse` が失敗する場合も、`msg.Type` が `""` になって TYPE の確認で断る側に倒れる(`send_origin.go:70-75` のコメントがその理由を言う)
- 断りは貼り付けマーカーの確認の直後、watch gate と dry run の分岐より前なので、何も打たず何も記録せず、dry run も同じ断りを予測する。`--raw` も `TYPE:` が 1 行目のときは同じ規則で読み、そうでない本文は「ヘッダの塊がない」として確かめない(コメントと help と `agent-messaging.md` がその条件を言う)
- 16 KiB の根拠(buffer 1 MiB、1 byte が最大 6 byte)を確かめたら正しかった(96 KiB)。非 raw でもヘッダの長さは `Validate` が見ないので、本文の 2,000 文字の上限だけでは足りず、この上限が要る
- 立て直しの進捗の節で、decision の text は 1 行ずつ 2 字下げ、制御文字と U+2028/2029 はエスケープ、囲みの fence は body の中の backtick の最長より長い(`fenceFor`)。`## ` の行、`- ` の行、7 個の backtick の行、`\x1b`、行区切りを含む text が項目から出ないことを `TestReseatProgress_AHumanDecisionStaysInsideItsEntry` が固定する。text は `strings.NewReplacer` の 1 回通しで prompt に入るので、`{{TASK}}` を含んでも再展開されない
- 「人の決定」と判定するのが `Details` の印と空でない `Text` の両方で、`--raw` の本文が `origin=human ` で始まるだけの偽の印は読まれない(`sentDetailsHumanOrigin` のコメントと `TestReseatProgress_TheHumanOriginMarkIsReadWhereSendWritesIt`)
- leader の振り分けは、既存の `HOOK_SEAT:` の扱いに倣って手順 4 と cross-review の上限の両方に入り、テストが 2 か所とも固定する。`ORIGIN: human` が承認でも直しの依頼でもないことと、承認が `ralph org approve` の台帳の記録に結びついていること(ヘッダを偽っても承認にはならない)が両立する
- 写しの一致: director skill の 4 写しと `agent-messaging.md` の雛形の写しが、いずれも `cmp` で同じ

### Tech debt identified

| Debt item | Impact | Why deferred | Trigger to pay down | Related plan/report |
| --- | --- | --- | --- | --- |
| L3: compaction のあとに leader が人の決定を読み直す手段がない(進捗の節に載るのは座席を立て直したときだけ。`sent` を返す動詞がない) | 「今後は X しない」のような続く人の決定が、leader の compaction で落ちうる。1 回きりの決定は届いたときに済むので影響が小さい | issue #366 の範囲は届けることと立て直しの引き継ぎ。読み直しの手段は読み取りの動詞か leader の手順の追加で、別の設計 | 続く人の決定が compaction で失われた報告があったとき、または人の決定が org の日常の運用になったとき | `docs/plans/active/2026-11-21-org-human-decision.md`(S3 の注意 4)、この report の L3 |

_(上の行を `docs/tech-debt/README.md` にも足した。)_

### Recommendation

- Merge (run 1): yes(CRITICAL と HIGH はない)。ただし M1 と M2 は、この branch の中で直せる文面の食い違いで、直すなら先に直す
- Follow-ups:
  - M1(`reseatProgressLead` の例外)はコードを変える直しなので、直すなら post-implementation pipeline を `/self-review` から回し直す(cycle は 1、上限は 2)。M2・L1 は skill と prompt の文面で、同じく文書だけでも `/verify` 以降を回し直す(`post-implementation-pipeline.md`)
  - L2 と L4 は 1 か所ずつの直し。L2 は直さないなら、上限を置かない理由を 1 文書く
  - L3 は tech-debt の行のとおり(この PR では直さない)
  - 計画の Progress の「Review artifact created」は、実装側が足す
- Known gaps: `ralph org send` の ORIGIN を実機の leader に送って、leader が計画の承認待ちのまま決定を進め、承認を待ち続けるかは、この run では見ていない(prompt の文面とテストが固定する文だけ)。`.agents/skills/` と `templates/base/` の写しの内容は `cmp` で同じことだけ見た(`check-skill-sync.sh` と `check-sync.sh` は `/verify` の担当)。誰が `ralph org send` を打ったかを ralph が確かめないこと(seat の pane の中からも `ORIGIN: human` を付けて leader に送れる)は、計画の Non-goals と Risks が受け入れた前提で、director skill の「人が決めたときだけ」で守る
