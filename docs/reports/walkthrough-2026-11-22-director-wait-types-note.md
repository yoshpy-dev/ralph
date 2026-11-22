# Walkthrough: director-wait-types-note

- Date: 2026-11-22
- Plan: docs/plans/active/2026-11-22-director-wait-types-note.md
- Issue: #374(PR は `Refs #374`。原因は実データで確かめていない)
- Base: main 64c3bfc5

差分は 24 ファイル、+1,687 行、-79 行(この walkthrough の commit の前の数)。そのうち 523 行は報告、計画、insight で、Go は 7 ファイル(+874 行、-23 行、うちテストが +781 行)。テスト以外の Go は 3 ファイルの +93 行、-9 行だけ。残りは `/director` と `/org` の skill(どちらも 4 写し。director は 1 写しあたり +70 行、-12 行)、仕様の 1 項目、tech-debt の台帳。下の順に読むと早い。

## 1. 何が報告されたか

director のセッションが `ralph org wait --inbox --director d1 --timeout-ms 1500000` を 2 本重ねて動かすことが、2026-11-22 の未明から朝に 5 回あった。後から始めた wait が `director-wait.json` を取ると、前の wait は `another ralph org wait --inbox --director … took over` の行を出して待ち続ける。

そのうち 1 回(03:42〜04:10 JST ごろ)は、重なっているあいだに leader の stall の ALERT が 2 件(e144、e145)入った。ところが 2 本ともどちらも件を返さず、`org: wait --inbox --director: no open inbox item arrived within 1500000 ms` で終わった。2 件は `NOTIFIED: yes` になっていた。ralph は 8.2.1。

issue が求めたのは、後の wait が前の wait を終わらせること、または重なっていても新しい件は少なくとも 1 本の wait が返すこと。director の skill に「前の wait が動いていないことを確かめてから張る」と書くことも挙げている。

## 2. 調べて分かったこと

重なりは、件が返らない原因ではなかった。

- 乗っ取られた wait も、毎回(既定は 1 秒ごと)受信箱の open の件を読んで返す。`loop` は毎回 `openInboxItems(w.p.Types)` を呼び(`internal/org/director_wait.go:530`)、`w.lost` を見ない。乗っ取りで止まるのは `director-wait.json` の書き込みだけで、`save` の `if w.lost { return }`(`internal/org/director_wait.go:670`)がそれ。`sayLost` が行を出して `w.lost` を立てる(`internal/org/director_wait.go:716`)
- 計画の前に、別プロセス 2 つの wait でも実測した。どちらも件を返し、終了コード 0 で終わった
- scratchpad の probe(`probe-374`、commit していない)は 5 つの場面を見た
  - S1: wait 1 本、`--types` なし。見張りが director に任せた ALERT を返す
  - S2: 2 本が重なり、`--types` なし、件は director に任せたもの。乗っ取られた wait も持ち主の wait も件を返す
  - S3: S2 と同じで、見張りが仕事を受け付けておらず、件が人へすぐ送られる場合。両方とも返す。notified は件の状態を変えない
  - S4: 2 本が重なり、`--types QUESTION,BLOCKED,RESULT` 付き。どちらも issue と同じ時間切れの文で終わり、ALERT の件は open のまま残る。見張りの ack の期限(probe は 10 分)が過ぎると、件は人へ送られて `NOTIFIED yes` になる。この文は `--types` なしの wait の時間切れと 1 文字も違わない
  - S5: wait が読む前に ack された件は返らず、同じ時間切れの文で終わる

issue の症状がすべて合うのは S4 で、wait に `--types QUESTION,BLOCKED,RESULT` が付き、件が leader の stall の ALERT だった場合。そのとき時間切れの文には、数えなかった件の手がかりが何もない。ただし症状が合う道はほかにもある(件が届いたときに読む wait がなかった、届いてすぐ ack か resolve された)。実データの台帳は別の repo にあるので、どの道かは確かめていない(8 節)。

そこでこの PR は 3 つをした。時間切れの文に数えなかった件を足す(3 節)。重なった 2 本がどちらも件を返すことをテストで固める(6 節)。skill に wait を重ねない手順を足す(4 節)。issue の 1 つ目の案(後の wait が前の wait を終わらせる)は採っていない。件が返らない原因に当たらず、新しい終了コードと skill の扱いが要るため。

## 3. 時間切れの文に足した句(読む順)

`--types` 付きの wait が時間切れになるとき、数えなかった種類の open の件があれば、文の後ろに `(counted only QUESTION,BLOCKED,RESULT; open items of other TYPEs: e144 ALERT, e145 ALERT)` の形の句が付く。

1. `internal/org/escalate.go:102` の `inboxWaitTimeoutPhrase`(`no open inbox item arrived within`)は変えていない。skill とテストが照合するので、文の先頭に残す。足したのは 2 つの定数で、`internal/org/escalate.go:108` の `inboxWaitOtherTypesPhrase`(`open items of other TYPEs:`)と、`internal/org/escalate.go:112` の `inboxWaitOtherTypesShown`(5)
2. `internal/org/escalate.go:377` の `readInboxWaitOtherTypesText` は、wait の最後の読み(時間切れのあとの読み)のあとにもう 1 回、すべての種類の open の件を読み、`--types` の外の件の文を返す。`--types` がないときは何も読まず `""` を返す。読みが失敗したときも `""` で、wait は句なしの時間切れで終わる
3. `internal/org/escalate.go:395` の `inboxWaitOtherTypesText` は、`<ID> <TYPE>` を ID の順に 5 件まで並べ、超えたら `, and <n> more` と数だけ言う。TYPE は `PrintableInboxText` で端末向けにエスケープする(手で書き換えた `inbox.jsonl` は TYPE に何でも入れられるため)
4. `internal/org/escalate.go:417` の `inboxWaitTimeoutNote` が句を組み立てる。外の件がなければ `""` で、文は今までのまま
5. `internal/org/escalate.go:336` で、`WaitInbox` の時間切れが文の後ろに句を足す。`ralph org wait --inbox` の経路はここ
6. `internal/org/director_wait.go:542` で、`--director` の wait の `loop` が時間切れのときに同じ読みをし、エラーの文に句を足す(`internal/org/director_wait.go:544`)。印の `detail` にも書く(`directorWaitTimeoutDetail`、`internal/org/director_wait.go:754`)。detail はもともと TYPE を言うので、足すのは ` (open items of other TYPEs: e144 ALERT)` だけで、`counted only` は繰り返さない。乗っ取られた wait が印を書かないのは今のまま

終了コード 1 と文の先頭は変わらない。`--types` なしのときと、外の件がないときの文も変わらない。見張りの配送の判断(`inboxDelivery`)と ack の期限には触れていない。

## 4. skill: wait を 1 本ずつ張る

`/director` skill の `### wait で待つ`(`.claude/skills/director/SKILL.md:1028`、4 写し)に足した。

- `.claude/skills/director/SKILL.md:1136`: 新しい wait を打つ前に、前の wait が終わったことを確かめる。Claude Code では、その wait を見張っていた Monitor(`run_in_background` なら終わりの知らせ)が終わりを知らせていれば終わっている。Codex では、前の wait が返ってから次を打つ
- `.claude/skills/director/SKILL.md:1143`: 出力に `another ralph org wait --inbox --director` で始まり `took over` を含む行が出たら、wait を重ねている。新しい wait がまだ走っているとは限らない。古い wait がこの行を出すのは印を書くときだけ(30 秒ごと、見張りの様子が変わったとき、終わるとき)なので、新しい wait が先に終わっていることもある。終わりは、終わりの知らせか、`ralph org director status` の `wait:` の行の `ended at` で見る
- `.claude/skills/director/SKILL.md:1155` から 1166 の 4 つの箇条: それ以上 wait を打たない。古い wait の Monitor も新しい wait も止めない。どちらかが件を返したら処理する(もう一方が同じ件を返すので、`ralph org inbox show <id>` で状態を見て 2 度処理しない)。もう一方が走っているあいだは次を打たず、2 本とも終わってから 1 本だけ打つ
- 打ち直しを言う 7 か所に「ほかの wait がまだ走っているあいだは打たない」を足した。終了コード 0 の項、終了コード 1 の時間切れ(Claude Code と Codex の箇条)、終了コード 4 の項、終了コード 1 の項、Monitor の期限切れの段落の 2 文。self-review の M1 が 3 か所、sync-docs が 4 か所。director が 1 文だけを読んで動いても、重なった片方が終わったところで 3 本目を打たないようにするため

skill は古い Monitor を止めろとは言わない。止めた wait の終わりは「Monitor の期限切れ」以外の signal での終わりになり、skill はそれを失敗として扱って受信箱を待つのをやめ、人の指示を待つ(`.claude/skills/director/SKILL.md:1109` から 1131)。古い wait は自分の `--timeout-ms`(25 分)でひとりでに終わる。これは `/plan` の plan advisory(codex)の指摘 1 件で、計画に入れた。skill が止めろと言わないことは、テストでも確かめる(6 節)。

時間切れの文の扱いは `.claude/skills/director/SKILL.md:1175` から 1190。文に `open items of other TYPEs:` が出たら、その件を `ralph org inbox show <id>` で読み、扱いは `--types` を指示した人に従う。ALERT を期限に任せると言われていれば、ack も resolve もしない(ack した件は ack の期限で人に届かない)。印の detail は `ralph org director status --json` の `wait.detail` と `ralph status --json` の `director.wait.detail` で読め、`wait:` と `director:` の行には出ない。

## 5. 仕様、help、/org

- 仕様 `docs/specs/2026-10-07-org-multi-org-director.md:174`: FR-9 の末尾に 2026-11-22 の項目。調べて分かったこと、足した句、skill に足した 3 つ、採らなかった案を書いた
- `ralph org wait --help` の `--types` の説明に 1 文(`internal/cli/org.go:1715` から 1718)
- `/org` skill のコマンド表の `wait` の行(`.claude/skills/org/SKILL.md:387`、4 写し)に 1 文。行は 1,090 文字から 1,183 文字になった。この文を固定するテストはない

## 6. 先に読むテスト

- `internal/org/director_wait_types_note_test.go:28` の `TestInboxWaitOtherTypesText`(5 件と 6 件の境目、エスケープ)、`:66` の `TestInboxWaitTimeoutNote`、`:91` の `TestReadInboxWaitOtherTypesText_ReadsTheOpenItemsOrNothing`(acked の件は数えない、読めない受信箱は空)
- `internal/org/director_wait_types_note_test.go:135` の `TestWaitInbox_TypesTimeoutNamesTheOpenItemsOfOtherTypes` と `:156` の `TestWaitInbox_TimeoutKeepsItsTextWithNoOpenItemOfAnotherType`(`--types` なし、外の件なし、待つあいだに resolve された件で、文が今のまま)
- `internal/org/director_wait_types_note_test.go:209` と `:235`: 上の 2 つの director 版
- `internal/org/director_wait_types_note_test.go:375` の `TestWaitInboxForDirector_TwoOverlappingWaitsBothReturnTheWatchsItems`(issue の 2 つ目の期待)。wait ごとに別の `Org` と PID(A が 4101、B が 4102)を持ち、A が `took over` を出してから、見張りの escalate の形で件を入れる。乗っ取られた A も持ち主の B も件を返し、印は B のものだけが残る。見張りが director に任せた場合と、人へすぐ送った場合の両方を見る
- `internal/org/director_wait_types_note_test.go:416` の `TestWaitInboxForDirector_TwoOverlappingTypesWaitsBothNameTheOpenAlerts`(issue の症状)。`--types` 付きの 2 本がどちらも時間切れになり、文に ALERT の件が出る
- 固定を変えたテスト: `internal/org/director_wait_test.go:787` の `TestWaitInboxForDirector_TypesTimeOutWhileOnlyAnAlertIsOpen` と `:808` の `TestDirectorWaitTimeoutDetail`。`internal/cli/org_inbox_test.go:1603` の `TestOrgWaitInbox_TypesKeepWaitingPastAnAlertItem`(stderr を完全一致に強めた)と `:1689` の `TestOrgWait_HelpDescribesTypes`
- `internal/org/director_skill_wait_types_note_test.go:215` の `TestDirectorSkill_RunsOneInboxWaitAtATime` は、27 の断片(`directorSkillWaitOneAtATimeSteps`、`:57`)を skill の順に照合し、wait や Monitor を止める言い回しがないことを `directorSkillStopsAWait`(`:117`。言い回しの表は `:138` の `TestDirectorSkillStopsAWait`)で見る
- `internal/org/director_skill_wait_types_note_test.go:290` の `TestDirectorSkill_ReadsTheUncountedItemsOfATypesTimeout` は、skill の例の句を `inboxWaitTimeoutNote` の出力から作って照合し、「5 件を超えると」の 5 を `inboxWaitOtherTypesShown` と結ぶ。新しいピンは `director_skill_sync_test.go` に足さず、新しいファイルに置いた(`director_skill_sync_test.go` は origin/main と同じ)

## 7. 確かめたこと

- `RALPH_VERIFY_SCOPE=full ./scripts/run-test.sh` は rc 0(test report の HEAD は ea088c24、1,403 秒、shell 46 本と Go 11 パッケージ)。足した 12 関数と変えた 4 関数は `-count=1 -v` で PASS、SKIP 0。重なった wait のテストは `-race -count=5` で 10/10、data race 0
- tester の変異 8 個はすべて落ちた(`WaitInbox` と director の句を外す、detail に足さない、読みを `--types` の種類だけにする、エスケープを外す、5 件の境目を変える、5 件で切るのをやめる、skill に止める指示を足す)。implementer の変異 38 個(S1 で 10、S2 で 28)も落ちたと計画の Progress にあるが、その出力は repo にない。穴を測る変異 2 個(止める指示の言い換え)は落ちない(8 節)
- `RALPH_VERIFY_SCOPE=full ./scripts/run-static-verify.sh` は pass。verify が HEAD 7e3448b7 で rc 0、origin/main の取り込みと sync-docs のあと(HEAD 9568d1d2)にも cross-review の前に rc 0
- verify は 7e3448b7 から作った binary で `ralph org wait --inbox` を 3 通り打った。ALERT 7 件と `--types QUESTION,BLOCKED,RESULT` では `(counted only QUESTION,BLOCKED,RESULT; open items of other TYPEs: e1 ALERT, …, e5 ALERT, and 2 more)` が付き、空の受信箱と `--types` なしでは句がない。終了コードはどれも 1。help の追記も binary で見た
- `--director` の時間切れは、CLI の層(cobra)で 1 回、commit しない probe で確かめた。終了コード 1、エラーの文と印の detail が期待どおりで、`ralph org director status --json` に detail が出た
- self-review は CRITICAL 0、HIGH 0、MEDIUM 2、LOW 6。すべて 7e3448b7 で直した。cross-review(codex、1 周、HEAD 9568d1d2)は指摘なし

## 8. 残したもの

- 原因は確かめていない。PR は `Refs #374`。ユーザーに 2 つを見てもらう。`ralph org inbox show e144` の notified の reason(`inbox_ack_deadline` なら `--types` の道、すぐ人へ送られていれば見張りが仕事を受け付けていなかった道)と、director の Monitor のコマンド行の `--types`。合えば issue を閉じ、違えば追いの issue にする
- 止める指示を探す検査(`directorSkillStopsAWait`)は言い換えを見つけない。「古いほうの Monitor に TaskStop をかけて終える」と「古いほうは TaskStop で片づける」を skill に足しても落ちない。今の skill にそう書いた文はなく、4 写しは `check-skill-sync.sh` でそろう。tech-debt に LOW の行を起こした(`The /director skill's check that no sentence of` で始まる行)
- wait の最後の読みと、句のための読みのあいだに、外の件が resolve される窓は試していない。テストが手を入れる場所がない。句がないか、resolve された件が載るかのどちらかで、時間切れの扱いは変わらない
- ビルドした binary での `--director` の時間切れは回していない(登録と、動いている見張りが要る)
- `/org` skill の `wait` の行の新しい 1 文を固定するテストはない
- sync-docs で skill を直したあと、`./scripts/run-test.sh` の通しは回していない。絞ったテストと静的検査は通った。通しは CI が回す
- 台帳の既存の 4 行に繰り越しの注を足した(受信箱の処理の test gap の行、RC9 の行、`director_start.go` の大きさの行、`/org` の表の行)。`director_wait.go` は 833 行(base は 821 行)で、800 行を超えたまま
