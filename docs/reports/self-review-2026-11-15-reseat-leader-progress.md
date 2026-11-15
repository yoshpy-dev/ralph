# Self-review report: reseat-leader-progress

- Date: 2026-11-16(run 3、cycle 2 の cross-review のあと)。cycle 2 の最初の run は同じ日、cycle 1 は 2026-11-15
- Plan: `docs/plans/active/2026-11-15-reseat-leader-progress.md`(issue #312 の #312c-2)
- Branch: `feat/reseat-leader-progress`、base は `origin/main` の `df186374`(main は `48a8a661` まで進み、branch は 3 回マージ済み)
- Reviewer: reviewer サブエージェント(`/self-review`)。**Cycle: 2、run 3**(`cycle-count.json` は `cycle: 2`、`cap: 3`。cycle 2 の cross-review の ACTION_REQUIRED 3 件を受けて cap を 2 から 3 に上げた。この run が cap の最後の回)。新しい run を先頭に置き、前の run は履歴にした
- **Merge: yes**(run 3: CRITICAL、HIGH、MEDIUM はなし。LOW 2 件。cross-review の ACTION_REQUIRED 3 件は `d85fc0ef` で閉じ、直しが持ち込んだ欠陥は見つからなかった)

## Run 3 (cycle-count 2, cap 3): HEAD `d85fc0ef`

Scope: `d85fc0ef`(cycle 2 の cross-review の ACTION_REQUIRED 3 件の直し。`internal/org/prompts/leader.md`、`internal/org/reseat_progress.go` と、その test 2 つ)。4 ファイルの差分を全部読み、`leader.md` は座席の確認・gate の 2 項目と手順 12・14 を全文で読んだ。確認に使った読み物: `protocol.Parse` と `truncateForDetails`、`Send` が `sent` の Details を作る箇所(本文が最後に付き、後ろに足す語がないこと)、`gh pr list --json` の項目名(`gh` に未知の項目名を渡して出る一覧に `headRefOid` がある)、agmsg の `history.sh`、cross-review の triage report、`docs/tech-debt/README.md` の行 519〜524。見なかったもの: テストの実行、仕様との突き合わせ、文書のずれの詳細(`/verify`、`/test`、`/sync-docs` の担当)。

### cross-review の 3 件が閉じたか

| # | 内容 | 状態 | 根拠 |
| --- | --- | --- | --- |
| 1 | gate の項目が PR の state だけを見て、gate を通したコミットが push されていない OPEN の PR を完了と報告する | 閉じた | `gh pr list … --json number,state,url,headRefOid` で PR の head を読む(`headRefOid` は `gh` の項目名にある。`merge.go`・`mergeorder.go` も同じ項目を使う)。PR の集まりは、(A)head が gate の head と同じ OPEN か MERGED、(B)A がなく head が違う OPEN、(C)A も B もなく MERGED、(D)CLOSED だけ、(E)PR なし、のどれか 1 つに入り、A は手順 15 へ、B は `wip:` の確かめと push のあと再読して一致なら手順 15・不一致なら BLOCKED、C と D は BLOCKED、E は手順 14 へ進む。B の push は force なしで、手順 14 の `wip:` の確かめと push を指すだけで、コマンドは書き写していない(手順の順を読むテストが push の最初の出現を手順 14 に置くため。テストがそれを固定する)。head が先に進んだ PR への push は non-fast-forward で失敗し、手順 14 のとおり BLOCKED になる |
| 2 | 座席の確認が、履歴で見つけた返事をすべて「今届いたもの」として扱い、前の leader が扱った失敗を繰り返す | 閉じた | 返事の時刻と、節の最後の TASK の全行の時刻を比べ、あとの行が 1 つでもあれば扱い直さず、座席は空いたものとしてほかの座席へ進む。あとの行がなければ従来どおり今届いたもの。出口は両方にある。時刻の書き方の主張(history.sh の `[` `]` の中が UTC の `YYYY-MM-DDTHH:MM:SSZ`、節の行の時刻と同じ書き方)は、台帳の `ts` が UTC の RFC3339 であること、既定の sqlite の driver が秒までの `%SZ` を書くことと合う |
| 3 | `truncateForDetails` が切った TASK の TASK_ID が `-` や切れ目の印つきで出て、返事と結びつかない | 閉じた | `reseatProgressSentTask` は、本文が切れ目の印で終わり、かつ TASK_ID が空か印を含むときだけ「不明」にする。TASK_ID の行が 200 byte より後に始まる、跨ぐ、ちょうど終わる(`\n` の前で切れたか値が続くか見分けられないので不明側に倒す)の 3 形は不明、TASK_ID が切れ目より前に全部ある形は後ろの行が切れても残る。切れ目の印は `truncateForDetails` の末尾にだけ付き、`Send` は本文のあとに何も足さないので `HasSuffix` が当たる。規則側は不明の行で履歴を読まず、同じ TASK_ID の送り直しもせず、implementer と reviewer のそれぞれに出口がある |

### 直しが持ち込んだ欠陥の確認

見つからなかった。確認した型と結果:

- 手順に到達できない・矛盾する指示: gate の項目の 5 つの場合(上の表)は互いに素で、`PR がなければ手順 14` が残りを受ける。座席の項目の入力待ちは、行なし・不明の行・不明でない行(返事あり=あとの行なし/あり、返事なし)の順に分かれ、どれも次の動作で終わる。
- 規則が読む値を節が出しているか: 不明の行の形 `- <座席>: 不明 (記録が切れている、<時刻>)` は Go の `Sprintf` と test が同じ文字列で固定し(`TestRenderRolePrompt_Leader_ReseatRuleSeatCheckUnknownTaskID` が節の出力から行の形を作って規則の文と照らす)、`不明` の定数を test が固定する。時刻は、不明でない行は括弧の中、不明の行は `、` のあとと規則が書き分ける。規則が節の行の時刻を比べる相手は、静的な節(タスクに入った文)で、leader がこれから送る TASK の行は含まない。
- コマンドの形: `gh pr list --head <ブランチ> --state all --json number,state,url,headRefOid` は有効。push は手順 14 の `git push -u origin <ブランチ>` を指す。
- 不明の判定の解析: `Parse` は TASK_ID の行を最後の値で上書きし、ヘッダの形でない行で止まる。切れ目が TASK_ID のキーの途中なら、その行は `:` がなく解析が止まって TASK_ID は空になり、印で終わるので不明。切れ目が `TASK_ID: foo\n` の直後なら 3 行目が印だけの行になり、TASK_ID は `foo` のまま残る(正しい)。`value()` を通る前の値で判定するので、400 rune の切り詰めは影響しない。
- 既存 test の弱め: 変わった期待は、直した文に合わせて言い換えたもの(「行があれば」→「TASK_ID が不明でない行があれば」など)と、署名の変更への追従だけで、検査の対象を外したものはない。古い読み(「あれば今届いたものとして扱います」、`--json number,state,url` の読み、「OPEN か MERGED の PR があれば PR は作りません」)が消えたことを見る否定の検査が足されている。
- 切れた TASK の test が本物の切り取りを通っている: `progressSentEvent` は `truncateForDetails` を呼んで Details を作る。

確認したが指摘にしないもの:

- 不明の行に至るのは、TASK_ID より前にヘッダがある TASK、TASK_ID が約 180 byte を超える TASK、TASK_ID のない raw の TASK に限る。`leader.md:1315` が `TASK_ID:` の次の行に `PLAN:` を置くと決めているので、標準の TASK は TASK_ID が先頭 2 行目で切れ目の前に入る。到達の少ない形に規則の文が増えているが、cross-review の指摘で、triage が「Real」とした範囲なので、この run では取り上げない。
- 不明の行の implementer の出口(「まだ済んでいない最初のスライス」)は、全スライスが済んでいるときの行き先を書かない。同じ項目の不明でない行の出口(「計画の残りのスライスから次の TASK を送り」)も同じで、直しが持ち込んだものではない。
- agmsg の driver が optional の `sqlite-sync` のときは、`history.sh` の時刻が `%fZ`(ミリ秒つき)になる。規則は時刻の大小を読むだけなので、比べ方は変わらない。
- push のあとの `gh pr list` の再読で、GitHub 側の更新が遅れると一致しないことがありうるが、その場合は BLOCKED になって人に渡り、誤って完了を報告する向きではない。

### Findings

この run の指摘は R3-1 から振る(C2-n と混ぜないため)。

| ID | Severity | Area | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- | --- |
| R3-1 | LOW | docs-in-register | `d85fc0ef` が触った 3 つのファイルについて、register の行が書いた数と「最後の commit」が古くなった。行 524: `reseat_progress_test.go` は「869 行、`wc -l` は `b8a9bd38` で、最後に変えた commit」と書くが、今は 969 行で最後は `d85fc0ef`(同じ行の「`reseat_progress.go` 551」「`prompts_leader_reseat_test.go` 516」も 582 と 615)。行 519 の (a): `reseatProgressSection` は「103 行」と書くが 107 行。行 520 の (a)(b) の取り消し線つきの「RESOLVED」は、`ec10ba6f` の時点の読み(`--json number,state,url`、「OPEN か MERGED の PR があれば PR は作らない」)をそのまま書き、今の規則(PR の `headRefOid` と gate の head を比べる)と読み比べると食い違う。行 519・523・524 の「次に触るとき」の条件は、この直しがファイルと `reseatProgressSentTask` を変えたので、また満たされている(支払いはされていない) | `docs/tech-debt/README.md` の行 519、520、524(`wc -l` は `internal/org/reseat_progress_test.go` 969、`reseat_progress.go` 582、`prompts_leader_reseat_test.go` 615)、行 523 の条件(`reseatProgressSentTask`) | `/sync-docs` が数を直し、行 520 の (a)(b) に「`d85fc0ef` で PR の head を gate の head と比べる形に変わった」と 1 句足す。条件が満たされたことは、cycle 2 の run で行 519 に書いた型(支払いなしの訂正)に揃える |
| R3-2 | LOW | naming | `reseat_progress.go` の `reseatProgressUnknownTaskID` は、不明の行に出す語 `不明` の定数だが、同じ struct の bool は `TaskIDUnknown` で、語と旗の名前が語順だけ違う。定数の doc コメントは「the first word after the colon is the marker」と書くが、その 6 行上の `reseatProgressTruncatedMarker` も「the marker」で、同じ語が別のものを指す。test の `TestReseatProgress_TheTruncatedMarkerIsTheOneTruncateForDetailsWrites` は、名前が切れ目の印だけを言うのに、`不明` の定数と 2 語の行の形も固定する | `reseat_progress.go:72-91`(`TaskIDUnknown`、`reseatProgressTruncatedMarker`、`reseatProgressUnknownTaskID`)、`reseat_progress_test.go` の同 test | 定数を `reseatProgressUnknownTaskIDWord` にし、doc コメントは「the word」と書く。test は名前を 2 つに分けるか、名前に行の形を足す。挙動は変わらない |

### Positive notes

- `reseatProgressSentTask` の doc コメントが、切れ目で TASK_ID が残る条件と不明になる条件(切れ目の印、TASK_ID が空か印を含む)を書き、「TASK_ID が切れ目より前に全部ある形は、後ろの行が切れても残る」まで言う。テストの 7 つの形(後ろに切れ目、跨ぐ、ちょうど終わる、短い TASK_ID と長い本文、短い TASK_ID と長い PLAN の行、切れた raw、切れていない raw)が、doc の場合分けと 1 対 1 に並ぶ。
- 規則の文が、gate の項目の B で push のコマンドを書き写さず手順 14 を指すこと、その理由(順を読む test)が test の中に書いてあること。
- 秘密、デバッグの残り、範囲外のファイルは差分に見当たらない。

## Tech debt identified

この run は cap 3 の最後の回なので、直さずに進める LOW は、`/sync-docs` か `/pr` が register に書かないと残らない。次の 1 行を足す(R3-1 は文書の直しなので `/sync-docs` がその場で直せる)。

| Debt item | Impact | Why deferred | Trigger to pay down | Related plan/report |
| --- | --- | --- | --- | --- |
| R3-1(register の行 519・520・524 の数と「最後の commit」が `d85fc0ef` のあとの状態でない)、R3-2(`reseatProgressUnknownTaskID` と `TaskIDUnknown` の名前、定数の doc の「the marker」、test の名前) | LOW。R3-1 は register を読む人に古い数を見せるだけ。R3-2 は読み手がどちらが語でどちらが旗かを名前から取れない | 最後の回で、どれも挙動を壊さない LOW(今から Go を直すと cap をもう一度上げて `/self-review` から回し直す) | R3-1 は `/sync-docs` の run で直す。R3-2 は次に `reseat_progress.go` の `reseatProgressTask` か不明の行を触るとき | この報告書、`docs/reports/cross-review-triage-reseat-leader-progress.md` |

## Recommendation

- Merge: yes(CRITICAL、HIGH、MEDIUM はなし。LOW 2 件)
- Follow-ups:
  - R3-1 は `/sync-docs` の担当で、Go も `leader.md` も触らないので、この run の結果で pipeline の再実行は要らない。
  - R3-2 は名前の直しで、挙動は変わらない。直すなら定数の改名と test の名前だけで、cap を上げた pipeline がもう 1 回回る。直さずに上の 1 行を register に書いて進めてもよい。
  - `/sync-docs` への引き継ぎ(指摘ではない): この直しが変えた 2 点(gate の項目が PR の head を比べること、「座席ごとの最後の TASK」の行に `不明 (記録が切れている、<時刻>)` の形があること)を、節の中身を列挙する文書(`README.md:303`、`AGENTS.md:92`、`docs/specs/2026-11-12-org-reseat.md`、`.claude/skills/org/SKILL.md` と `director/SKILL.md` とその mirror)が行ごとの形まで書いているかを、cycle 2 の run が書いた引き継ぎと合わせて見る。`.claude/skills/org/SKILL.md:1836-1850` は節の中身を名前で並べるだけで、行の形は書かない。

## Cycle 2: HEAD `bd4dcb9b`(cap 2 の run。履歴)

この節の「Tech debt identified」と「Recommendation」は、cap 2 の最後の回だった当時のもの。C2-1 から C2-5 の扱いは、cross-review の triage と `d85fc0ef` のあとの register に引き継がれている。

Scope: cycle 1 の self-review のあとに入った `291ac942`(N4・N5 の直し)、`e483b52a`(lint)、`77a45022`(sync-docs)、`ec10ba6f`(cross-review の直し)、`bd4dcb9b`(計画の記録)。`ec10ba6f` は 4 ファイル全部を読んだ(`internal/org/reseat_progress.go` は全文、`leader.md` は座席の確認・計画・gate の 3 項目と手順 8・12・14、「計画助言」「cross-review」の小節、「agmsg の受信箱の運用」の節、`prompts_leader_reseat_test.go` と `reseat_progress_test.go` の差分)。`77a45022` は Go の変更(`internal/cli/org_reseat.go` と test)と、節の中身を列挙している文書の箇所を読んだ。確認に使った読み物: `verbs.go` の `Send`(`sent` の Details を作る 3 か所)、`send_fallback.go` の `details()`、`protocol.Parse`、`truncateForDetails`、`docs/tech-debt/README.md` の行 519〜521、cross-review の triage report。見なかったもの: テストの実行、仕様との突き合わせ、文書のずれの詳細(`/verify`、`/test`、`/sync-docs` の担当。ただし `77a45022` が書いた箇所が最終の挙動を述べているかは、依頼により読んだ)。

読んで問題なしとしたもの:

- `sent` の Details の読み取り(`reseatProgressSentTask`)。Send が前に付ける語は `raw=true `、`via=screen herdr_state=<state> `、`submit_unconfirmed=true ` の順で、`truncateForDetails` が本文を 200 byte で切る。読みは `=` を含み改行を含まない先頭の語を落とし、`TYPE: <値>` で始まる本文を `protocol.Parse` に渡す。TYPE と TASK_ID は先頭 2 行なので 200 byte の切れ目より前にある(`TASK_ID:` の次の行に `PLAN:` を書く手順 5 の決まりとも合う)。raw で TYPE が先頭行にない文、TYPE が TASK でない文、dry run、leader 宛て、ほかの org は数えない。Send の `isTask`(`parseErr == nil && Type == TASK`)と同じ判定で、`--raw` の TASK も数える。
- 今の life の境目。Tasks は最後の本物の `disbanded` の次のイベントから(添字)、Results は `disbanded` の時刻より後に上がった件(時刻の比較。同じ秒は前の life 側)。同じ規則を Inbox も使い、時刻が読めない件は注つきで残る。
- 空の規則。`reseatProgressSection` は Tasks の行と RESULT の行を `add` で書くので、TASK か RESULT しかない org でも節ができる。ほかの org の RESULT と TASK、dry run の TASK、TASK でない typed message は節を作らない(`TestReseatProgress_AnOrgWithNoProgressHasNoSection` がこれらを固定する)。
- 安全。新しい行の値は `value()` か `reseatProgressText` を通り、制御文字と U+2028・U+2029 が置き換わる。行頭は `- ` かラベルで、囲みの外に出る行はない。QUESTION の `PLAN:` の値は `strings.Fields` で空白を 1 つにまとめる。
- 規則の新しい分岐が読む値は、すべて節が出す。計画の項目の QUESTION の `PLAN <パス> sha256:<digest>`、座席の項目の「座席ごとの最後の TASK」の行、gate の項目の「上げた RESULT」の `HEAD`(手順 14 の RESULT のヘッダ `HEAD:` は gate の出力の HEAD で、gate の行の head と同じ値)。PR の状態は `gh pr list --json number,state,url` が出す。
- 計画助言の判定は「計画助言」の小節と合う。per-plan は手順 2 の前に済み、要約は QUESTION の `ADVISORY:` の行に、split は `## Design decisions` に書く。`plan_advisory` が off のときは記録を残さないので、どちらの印もない場合に「小節のとおり自分で決める」は、off なら何もしない形になる。

### Findings

この cycle の指摘は C2-1 から振る(cycle 1 の M・L・N と混ぜないため)。

| ID | Severity | Area | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- | --- |
| C2-1 | LOW | consistency | 同じ `leader.md` の中で、leader が送った TASK が agmsg の履歴に入るかについて、2 か所が逆を言う。直しの座席の項目は「送った TASK は台帳に残り、agmsg の履歴には残りません」と書き、テストは座席の項目に「あなたが送った TASK も 50 件に入る」が戻らないことを見る。一方「agmsg の受信箱の運用」の節は今も「`history.sh` は既読の印を変えません。あなたが送った TASK も 50 件に入るので、足りなければ数を増やします」と書き、別のテストがその文を固定する。`Send` は herdr の `send-text` で pane に打つだけで agmsg には何も投げない(cross-review の triage が `verbs.go` で確認した事実で、`Send` の呼び出しにも agmsg はない)ので、直った側が正しく、残った側が誤り。cross-review の finding 1 は報告された箇所だけを直し、同じ誤りを持つ兄弟の文が残った | `leader.md:182-184`(新しい文)、`leader.md:1405-1406`(誤りが残る文)、`prompts_leader_inbox_test.go:107`(誤った文を固定)、`prompts_leader_reseat_test.go:325`(座席の項目だけで禁止) | 1406 の「あなたが送った TASK も 50 件に入るので、」を落として「足りなければ数を増やします」だけにし、`prompts_leader_inbox_test.go:107` の固定を直す。`prompts_leader_reseat_test.go:325` の禁止語を、この項目に限らず `leader.md` 全体にかける |
| C2-2 | LOW | maintainability | 入力待ちで RESULT がない分岐(「その TASK に答えないまま入力待ちです」)の 2 つの出口が、implementer のスライスの TASK だけを前提に書かれている。(a)「branch の commit の行がその TASK のスライスをもう済んだと示していれば、座席は空いた」は、reviewer のレビューの TASK には当たらない: スライスの commit はレビューの前にあるので、返事のないレビューの TASK が、commit があることを理由に「済んだ」と読めてしまい、レビューを飛ばして次のスライスへ進める。(b)「同じ TASK_ID でもう一度送ります(中身は計画のそのスライスから書きます)」は、`gate-<n>` の TASK(中身は手順 12 の gate のコマンド)と `xreview-…` の TASK(中身は「cross-review」の小節の形)には書き方の出どころが違う。同じ項目の 1 つ前の分岐は、TASK_ID の頭で手順 12・小節・手順 8 に分けており、この分岐だけ分けていない(cycle 1 の N5 と同じ型)。手順 8 と 12 は、通らなかった TASK を新しい TASK_ID で送り直すと書くので、「同じ TASK_ID」を選んだ理由(RESULT を TASK_ID で突き合わせるため)も項目に書かれていない。到達は、返事を返さないまま入力待ちになった reviewer(利用上限や API エラーのあとの再開など)に限られ、まれ | `leader.md:191-195`(分岐)、`:189-191`(前の分岐の TASK_ID 別の行き先)、手順 8 の `新しい TASK_ID で、CHECKS: の行を付けたレビューの TASK を送り直す`、手順 12 の `新しい TASK_ID で送り直す`、`prompts_leader_reseat_test.go` の `そうでなければ、その TASK を同じ TASK_ID でもう一度送ります` | 出口を TASK_ID の頭で分ける 1 文を足す(implementer のスライスの TASK は今の文のとおり、`gate-` は手順 12 の gate の TASK を同じ TASK_ID で、`xreview-` は小節の TASK を同じ TASK_ID で、reviewer のレビューの TASK は commit の行を根拠にせず同じ TASK_ID で)。「同じ TASK_ID」にする理由を 1 句添える |
| C2-3 | LOW | readability | 新しい「座席ごとの最後の TASK」の行は `- <座席>: <TASK_ID>(<時刻>)` で、規則が比べる値の TASK_ID に時刻の括弧が空白なしでつく。承認の行は、規則が digest を空白で区切った 1 語として比べるので `sha256:<digest> (承認 <時刻>)` に直した(cycle 1 の L3)。TASK_ID は座席の項目が RESULT の TASK_ID と突き合わせる値なのに、同じ型が残った。register の行 519 の (d)(gate の行の括弧)が「規則が比べるのは head と base だけで、どちらも空白で区切れるので、判断はこれに依らない」を理由にしており、この行にはその理由が当たらない | `reseat_progress.go:311`(`"- %s: %s(%s)"`)、`reseat_progress_test.go` の `"- implementer: slice-1(2026-11-15T01:20:00Z)"`(固定)、`docs/tech-debt/README.md` 行 519 の (d) | `"- %s: %s (%s)"` にして、テストが空白で切った語を見る(`TestReseatProgress_TheApprovalLineGivesTheDigestAsOneWord` と同じ形)。直さないなら、行 519 の (d) に「座席ごとの最後の TASK の行(`reseat_progress.go`)にも同じ括弧があり、こちらは規則が比べる値」を足し、理由の文を直す |
| C2-4 | LOW | test | `TestReseatProgress_TheDetailsPrefixesAreTheOnesSendWrites` は、doc コメントが「`reseatProgressSentTask` が落とす語を、Send が本文の前に付ける語と合わせておく」と言うが、語は `"raw=true "`、`"via=screen herdr_state=idle "`、`"submit_unconfirmed=true "` の文字列の写しで、Send の側(`verbs.go:479` の `"raw=true "`、`:666` の `"submit_unconfirmed=true "`)とは結びついていない。両側に共有の定数がなく、Send が `=` のない語や空白を含む値の語を足すと、読みは語の途中で止まって `protocol.Parse` が失敗し、その TASK は節から黙って落ちる(座席の項目は「その座席の行がなければ TASK を受けていない」と読み、実際に届いている座席へ TASK を送り直す)。落ちる向きは前の挙動(行なし)と同じなので害は小さいが、doc コメントの約束は守られていない。同じ test の隣の `TestReseatProgress_TheLastTaskPerSeatComesFromTheSentEvents` は `(&sendScreenPath{…}).details()` を使い、画面の語についてだけ本物の出力に結びついている | `reseat_progress_test.go:731-746`、`verbs.go:479`、`:666`、`send_fallback.go:60-65` | Send の 2 つの語を定数(`sendDetailsRawPrefix`、`sendDetailsSubmitUnconfirmed`)にして、test と `reseatProgressSentTask` の doc がそれを使う。または test が `Org.Send` の fake herdr を通して `sent` を作り、その Details を読む |
| C2-5 | LOW | docs-in-code | `ralph org reseat --help` の節の説明は、中身を閉じた列挙で書く(「最新の plan approval、最後の gate の記録、open と acked の受信箱の件、branch の commit、この reseat が leader のあとに起動する座席、前の reseat」)。`ec10ba6f` のあとで節は 2 つの行(座席ごとの最後の TASK、今の life で上げた RESULT)と QUESTION の `PLAN:` を持つが、`--help` とその test の固定は 6 項目のままで、`/sync-docs` が `77a45022` で書いた文書の列挙も同じ 6 項目。`--help` は Go の文字列で test が固定するので、文書の直しとは別に、次の fix commit の対象になる | `internal/cli/org_reseat.go:104-109`、`internal/cli/org_reseat_test.go:873-875`。同じ列挙を持つ文書(`/sync-docs` への引き継ぎを参照) | 2 項目を足す(「the last TASK sent to each seat, the RESULT items it raised」)。文書は `/sync-docs` の cycle 2 で足す |

### cycle 1 の指摘と cross-review の確認

| ID | 内容 | 状態 | 根拠 |
| --- | --- | --- | --- |
| N4 | gate の例外が座席の確認の入力待ちの分岐に掛かっていない | 閉じた | 座席の項目の頭に gate の例外がある(`leader.md:165-168`: 「gate の項目に当たるとき…はこの確認の残りはせず、どの座席にも TASK を送らず」)。作業中の分岐の括弧も残る |
| N5 | 入力待ちの分岐の「手順 8 のとおり」が implementer と reviewer のレビューにしか合わない | 閉じた(RESULT を扱う側) | `leader.md:190-191`: `gate-` は手順 12、`xreview-` は「cross-review」の小節、それ以外は手順 8。TASK を出す側の同じ型は C2-2 |
| cross-review 1 | leader が送った TASK を agmsg の履歴から探している | 直った。同じ誤りを持つ兄弟の文が 1 か所残る(C2-1) | `sent` の Details から座席ごとの最後の TASK を読み、TASK_ID で RESULT・BLOCKED を突き合わせる。到達する形と境目は上の「問題なしとしたもの」 |
| cross-review 2 | PR を作ったあと RESULT の前に止まった leader が、手順 14 の RESULT を上げない | 直った | 「上げた RESULT」の行(今の life、全状態、新しい 5 件)と、gate の項目の RESULT の確認。RESULT の `HEAD` は gate の行の head と同じ値(手順 14 のヘッダ)。5 件の上限は、RESULT を上げるのが手順 14 だけなので、実際には当たらない |
| cross-review 3 | `--state all` が merge なしで閉じた PR も返す | 直った | gate の項目が `OPEN`、`MERGED`、`CLOSED` を分け、`CLOSED` だけなら BLOCKED。PR なし、OPEN か MERGED、CLOSED だけの 3 つで次の手順がそろっている |
| cross-review 4 | 承認なしの計画で、計画助言が済んだかが分からない | 直った | QUESTION の行の `PLAN`(パスと digest)で判定し、split は `## Design decisions` の要約で判定する。どちらの印もなければ「計画助言」の小節のとおりに自分で決める |

### Positive notes

- `reseatProgressSentTask` の doc コメントが、Send の前の語、200 byte の切れ目、残るのが TYPE と TASK_ID の先頭 2 行であること、raw の扱いを順に書いている。コードは doc のとおりに読める。
- `Tasks` を map で集めて座席の id で並べ直すので、節の行の順が実行ごとに変わらない。
- テストが、各列挙値(prefix、切れた長い本文、dry run、ほかの org、leader 宛て、別の TYPE が後から来る)を 1 件ずつ持ち、期待を行の列で比べる。RESULT の 5 件の上限は、5 件ちょうどと 8 件の 2 つを見る。
- 規則のテストは、新しい分岐の文の順と、古い文(「で PR があるかを見ます」など)が消えたことを固定している。手順 14 の RESULT の文を引いて、gate の項目が上げる RESULT の出どころを固定する点も良い。
- 秘密、デバッグの残り、範囲外のファイルは差分に見当たらない。

## Tech debt identified

cycle 2 は `max_pipeline_cycles` の 2 回目で、最後の回。C2-1 から C2-5 を直さず進めるなら、次の 1 行を register に足す(`/sync-docs` か `/pr` の担当)。

| Debt item | Impact | Why deferred | Trigger to pay down | Related plan/report |
| --- | --- | --- | --- | --- |
| C2-1(`leader.md:1406` の「送った TASK も 50 件に入る」が座席の項目と逆)、C2-2(RESULT なしの分岐の出口が implementer のスライス前提)、C2-3(座席ごとの最後の TASK の行の括弧)、C2-4(prefix の test が Send に結びつかない)、C2-5(`--help` の列挙が 2 項目足りない) | LOW。C2-1 は leader が誤った一文を読むだけで、座席の項目の指示は正しい。C2-2 は、返事のない reviewer の TASK という到達のまれな形。C2-3 は行を空白で切る読み手にだけ。C2-4 は落ちる向きが前の挙動と同じ。C2-5 は表示のみ | 最後の回で、どれも挙動を壊さない LOW(今から直すと cap を上げて pipeline が `/self-review` から回り直す) | 次に `leader.md` の座席の確認か「agmsg の受信箱の運用」の節、`reseat_progress.go` の Tasks の行、`sent` の Details を作る `Send` を触るとき | この報告書、`docs/reports/cross-review-triage-reseat-leader-progress.md` |

## Recommendation

- Merge: yes(CRITICAL、HIGH、MEDIUM はなし。LOW 5 件)
- Follow-ups:
  - C2-1 と C2-3 は 1 行ずつで、テストの固定を直すだけで済む。C2-1 は兄弟の誤りが同じ PR の前の直しと矛盾している点で、5 件のうち先に直す価値がある。C2-2 は 1 文。直せば pipeline は最後の回を超えるので、cap を上げて `/self-review` から回し直すか、上の 1 行を register に書いて進む。
  - **`/sync-docs` への引き継ぎ**(指摘ではない): `ec10ba6f` が節に足した 2 行(座席ごとの最後の TASK、上げた RESULT)と QUESTION の `PLAN` を、`77a45022` が書いた中身の列挙のうち次の箇所が持たない: `README.md:303`、`AGENTS.md:92`(`reseat_progress.go` の説明)、`docs/specs/2026-11-12-org-reseat.md:80`(FR-9 の列挙)、`docs/specs/2026-08-01-org-runtime.md:125`、`.claude/skills/org/SKILL.md` と `.claude/skills/director/SKILL.md`(とその `.agents/` と `templates/base/` の mirror 6 つ)。`77a45022` の時点では正しく、cross-review の直しがあとから列挙を古くした。
  - **register の行**(`/sync-docs`): 行 520 の (a)(`--state all` が閉じた PR を数える)、(b)(PR のあと RESULT の前に止まる)、(c)(計画助言が済んだか分からない)は `ec10ba6f` が直したので、行から外すか「`ec10ba6f` で直した」と書く。(d)(live の確認)は残る。行 519 の (a) の「86 行」は `reseatProgressSection` が 103 行になって数が違う(L5 の中身は今回の再提起ではなく、数だけの訂正)。行 519 の (d) は C2-3 の新しい箇所を足す。

## Cycle 1(履歴): 3 回の run、すべて直して閉じた

HEAD `8ac1487c`、`d4d1ca93`、`a2494fab` の 3 回。報告書の全文は `git show f57e37f4:docs/reports/self-review-2026-11-15-reseat-leader-progress.md`。

| Run | HEAD | 指摘 | 直した commit |
| --- | --- | --- | --- |
| 1 | `8ac1487c` | MEDIUM 1(M1: leader だけ、または implementer を動かしたままの立て直しが規則のどの項目にも当たらない)、LOW 7(L1: `git log` に `--no-show-signature` がない、L2: 「最初の N 件」の向き、L3: digest の後ろの括弧、L4: Go のコメントが `docs/plans/active/` を指す、L5: 関数の長さ・`inboxSummary` との二重・`NewReplacer`、L6: 計画の項目の行き先、L7: gate の項目が座席を待たない) | `71ddcb30`(L5 と L3 の gate の行は register 行 519 へ) |
| 2 | `d4d1ca93` | MEDIUM 1(N1: 座席の確認の「待つ」に出口がない)、LOW 2(N2: 項目の語の範囲、N3: コメントの折り返し) | `2a20a1f2` |
| 3 | `a2494fab` | MEDIUM 1(N4: 入力待ちの分岐に gate の例外が掛からない)、LOW 1(N5: 「手順 8 のとおり」が TASK_ID の頭で分かれない) | `291ac942` |

cycle 1 の cross-review(4 件の ACTION_REQUIRED)は `ec10ba6f` で直り、上の表で確かめた。
