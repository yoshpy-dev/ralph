# Self-review report: director-wait-types-note

- Date: 2026-11-22
- Plan: `docs/plans/active/2026-11-22-director-wait-types-note.md`(issue #374)
- Reviewer: reviewer サブエージェント(`/self-review`)。パイプライン 1 回目(`cycle-count.json` は cycle 1)
- Scope: origin/main `b263b1f3` から HEAD `61dd8ced` までの `git diff origin/main...HEAD`(13 ファイル、+1034 -27)のうち、diff の質を見るもの。コードは `internal/org/escalate.go`・`director_wait.go`、`internal/cli/org.go` の help、テストは新しい `director_wait_types_note_test.go` と `director_skill_sync_test.go` の追加分、既存テストの更新、文書は `/director` skill(4 写し)と仕様の 1 項目。仕様との突き合わせ、テストの網羅、文書のずれは見ていない

## Evidence reviewed

- `git diff origin/main...HEAD` の Go の本体(`escalate.go` 71 行、`director_wait.go` 24 行、`cli/org.go` の help 5 行)の全 hunk と、`WaitInbox`・`openInboxItems`・`WaitInboxForDirector` の `loop`・`sayLost`・`PrintableInboxText`・`foldInbox` の周りの全文
- 時間切れの文の読み手の確認: `inboxWaitTimeoutPhrase` を使う非テストの所は 2 か所の生成と skill の 4 写しだけで、全文を照合する所はない(`grep`)。終了コード 1 と先頭の句は変わらない
- ID の前提: `foldInbox` の `apply` が `inboxIDNumber`(`e<N>`、先頭の 0 なし)を通らない行を捨てるので、`inboxWaitOtherTypesText` が ID をエスケープしないのは成り立つ。TYPE は `ev.Type` のままなので `PrintableInboxText` が要り、テストが ESC を入れた値で固定している
- skill の主張の裏取り: `wait.detail`(`orgDirectorReport.Wait` の `json:"wait,omitempty"`)と `director.wait.detail`(`status.go:613` の `json:"director"`)は存在し、`Detail` は JSON の構造体にだけ入って text の行を作る所は読まない(`grep`)。引用した行 `another ralph org wait --inbox --director … took over` は `sayLost`(`director_wait.go:724`)の文と合う。skill の 4 写しは `check-skill-sync.sh` で lock-step、手元の `diff --no-index` でも同一
- skill 全文の該当節(`### wait で待つ`、1028〜1176 行)を通して読み、(a)(b)(c) と既存の終了コードの表、Monitor の期限切れの段落、push の wake の節との突き合わせ
- 新しいテスト 418 行の全文。`director_skill_sync_test.go` の追加 215 行(正規表現のテスト 1、skill のピン 2)。`directorSkillStopsAWaitRe` を別の小さな Go プログラム(`$TMPDIR` 側)で 22 の文に当てた
- `go test -count=1 ./internal/org/ -run 'WaitInbox|DirectorWait|DirectorSkill'` と `./internal/cli/ -run 'OrgWait|OrgInbox'` は ok。新しい時間依存のテストは `-race -count=8` で ok、CPU を 6 つ塞いで `GOMAXPROCS=1`・`-race -count=5` でも ok(flake は出なかった)
- `docs/tech-debt/README.md` を、diff が触れたファイル名(`director_wait.go`、`escalate.go`、`director_skill_sync_test.go`)と記号で引き、行 267・395・426 の Trigger のセルを読んだ。diff に台帳の hunk はない。`git diff --check` は rc 0

## Findings

| ID | Severity | Area | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- | --- |
| M1 | MEDIUM | maintainability | skill の「打ち直す」が 3 か所で無条件のまま、新しい「もう一方が走っているあいだは次を打たない」と食い違う。重なった 2 本のうち古いほうが先に時間切れになると、director は「終了コード 1 で時間切れの文があれば上のとおり打ち直す」を読んで 3 本目を打てる。新しい規則は「wait を重ねたとき」の段落の 4 つ目の箇条にだけあり、`wait を重ねたしるし` の行(1095〜1097)が指すのも、その段落までの案内にとどまる。重なりは、コンテキストが詰め直されたあとなどに前の wait を忘れて起きるので、そのとき director の目にあるのは例外つきの段落でなく、終了コードの表の 1 行のほうである。固定するテストも、この 3 か所と段落のつながりは見ていない | `.claude/skills/director/SKILL.md` 1042〜1043(Claude Code の箇条)、1052〜1054(Codex の箇条)、1098〜1099(終了コード 1 の箇条)と、1151〜1152(新しい規則)。`directorSkillWaitOneAtATimeSteps`(`director_skill_sync_test.go`)の 17 の fragment に「打ち直す」側の条件はない | 3 か所の「打ち直す」に「ただし、ほかの wait が走っているあいだは打たない(下の、wait を 1 本ずつ張る段落)」を足す。1 つ目の箇条が「打つ前に前の wait が終わったことを確かめる」の段落を指すだけでも足りる。足した文を `directorSkillWaitOneAtATimeSteps` の fragment に入れる。skill は 4 写し、`scripts/sync-skills.sh` |
| M2 | MEDIUM | maintainability | 台帳の行 395(「次の変更で `director_skill_sync_test.go` などからテストを出す」)と行 267(「`WaitInbox` の次の変更で、skill のピンの `t.Skipf` などを直す」)の Trigger を満たすのに、新しい 215 行をこの 1320 行のファイルへ足した(1535 行、+16%)。#271 と #365 は同じ状況で新しいピンを新しいファイル(`skill_split_questions_test.go`、`skill_issue_test.go`)へ置いて、このファイルを 52 行と 2 行の増加にとどめた。今回の追加は新しいコードだけなので、move-only の差分を作らずに分けられる。同じ PR は `director_wait.go` も 821 行から 833 行へ増やした(base から 800 行超)が、このファイルの行の大きさを載せた行は台帳にない | `wc -l`: `director_skill_sync_test.go` 1320(base)から 1535、`director_wait.go` 821 から 833、`escalate.go` 740。行 395 の末尾の 2026-11-22 の繰り越し文(#365)。`grep -n 'director_wait\.go' docs/tech-debt/README.md` の 3 件(261・327・386)は大きさの行ではない | 新しい 3 つのテストと 3 つの定数・変数(`directorSkillTakeoverLine*`、`directorSkillWaitOneAtATimeSteps`、`directorSkillStopsAWaitRe`、`directorSkillTypesTimeout*`)を `internal/org/skill_wait_overlap_test.go` のような新しいファイルへ移す(`withoutSpaces`、`directorSkillWaitSubsection`、`skillOrderedFragmentProblems` は今の場所のまま使う)。この PR で動かさないなら、sync-docs で行 395 と行 267 に日付つきの繰り越し(1320 から 1535、`WaitInbox` の時間切れの文を変えたこと、新しい 2 つの `t.Skipf` が (1) と同じ形であること)を足し、`director_wait.go` の 833 行は新しい行にする(下の Tech debt) |
| L1 | LOW | maintainability | 「wait を重ねたとき」の段落は 2 つのことを director にまかせているが、足がかりを言わない。(a) 段落の冒頭は、その行が名指す新しい wait が「走っている」と言い切る。古い wait が行を出すのは、次の印の書き込み(30 秒ごと)か終わりのときなので、新しい wait がもう終わっていることもある。(b) 「2 本とも終わってから、1 本だけ打つ」の終わりは、director が自分の Monitor か `run_in_background` の終わりの知らせで知るしかない。新しい wait が前のセッションの置き土産(skill 自身が 1044〜1048 行で、Bash の wait は `/exit` や SIGKILL のあとも 25 分走ると言う)なら、その終わりを知らせるものがなく、director は次を打たないまま待ちを止められる | skill 1136〜1152 行。新しい wait の印は `ralph org director status` の `wait:` の行(1062〜1066 行)にあり、`end` と `ended_at` を持つ(古い wait の終わりは、印を引き継がれているので残らない) | 「走っている」を「走っていることがある」に直し、自分の知らせがない wait は `ralph org director status` の `wait:` の行(`end` が出ているか)で見る、と 1 文足す。古い wait は自分の `--timeout-ms` で終わる分(25 分)を待つ、と上限も言う |
| L2 | LOW | maintainability | `directorSkillStopsAWaitRe` の doc とテストのエラー文は「wait や Monitor を止めろと言う言葉」を探すと言うが、探すのは `(Monitor|wait)[をも]…止め(る\|て\|、\|。\|ろ)` の 1 つの形だけ。手元の probe(22 文)で、通らない形: `` `Monitor` を止める``(バッククォート)、`Monitor は止める`(は)、`Monitor を停止する`、`wait を kill する`、`wait を TaskStop する`、`wait を打ち切る`、`wait を終わらせる`。逆に、禁止の言い回しを自然に書き換えた `wait を止めるな`・`止めてはいけない` は、`止め` のあとの る/て にあたって失敗になる。今の skill が書く「止めない」だけが両方を避けている | `director_skill_sync_test.go` の `directorSkillStopsAWaitRe`(683 行から)とその表の 10 行。probe は `$TMPDIR` 側の小さな Go プログラム | doc を「`止め` で終わる形の言い回しを見る」と狭く書き、表に「止めるな」「止めてはいけない」を `false`(許す)側で足すか、規則を「止めない」の 1 つに絞って、`止め` のあとが `ない` でないものを探す形にする。`TaskStop`・`停止`・`kill` の語を別に探すなら表に足す |
| L3 | LOW | maintainability | 重なりのテストは、`startOverlappingDirectorWaits` が wait の goroutine を `context.Background()` で起こし、`t.Cleanup` で止めない。`waitForMark` や `waitForStderr` が `t.Fatalf` で先に失敗すると、2 つの wait は、そのまま 5 秒(`--types` の wait は 1 秒)走り続け、`t.TempDir` の後始末と書き込みが重なる(失敗したときだけ)。既存の `startDirectorWait` は ctx を受ける。時間は実時間で、2 つの前提がある。1 つ目は、1 秒の wait が時間切れになる前に 2 つの escalate を済ませること(`len(w.aDone) > 0` の `t.Fatalf` が「machine too slow」を言う)。2 つ目は、「ALERT が wait の途中で resolve される」ケースの goroutine が、300ms の時間切れより前の 50ms 後に動くこと。上のとおり、`-race -count=8` と CPU 逼迫でも失敗は出ず、余裕は足りている | `director_wait_types_note_test.go` 306〜331 行(`context.Background()`)、401〜408 行(1 秒と `len(...) > 0`)、165〜185 行(`time.Sleep(50 * time.Millisecond)`) | `ctx, cancel := context.WithCancel(...)` を作って `t.Cleanup(cancel)` と、2 つの `done` の読み切りを足す。実時間の余裕は今のままでよい(失敗の文が原因を言う) |
| L4 | LOW | maintainability | `escalateLeaderStallAsTheWatch` は、`watchRun`(`org`・`cfg`・`stderr`・`acceptsWork`)と `watchStatusFile`・`watchPendingAlert`(`HistoryLeaderLines: -1`)を手で組み直す。`watch_test.go` の `pendingLeaderAlert`(993 行)が同じ状態を作る(`condLiveness`、`org-a` 固定、`fakeClock`)ので、watch の `escalateAlert` に状態や引数が増えると、直す場所が 2 つになる。失敗は `t.Fatalf("the watch did not escalate …")` で声が出るので、黙って通ることはない | `director_wait_types_note_test.go` 273〜296 行、`watch_test.go` 993〜1003 行 | `pendingLeaderAlert` を `condition` と `orgID` を受ける形に広げて使い回すか、この helper の doc に「`pendingLeaderAlert` の写し」と書く |
| L5 | LOW | naming | `(*Org).inboxWaitOtherTypes` は「読む」と「文にする」を持ち、返すのは文字列。名前は種類の一覧を返すように読め、純粋な整形の `inboxWaitOtherTypesText` と 1 語違いで並ぶ。doc の「one more read」は、`WaitInbox`・director の `loop` が時間切れのあとにもう 1 回読む(既存の文、`director_wait.go:286`)と、今回足した読み(`inboxWaitOtherTypes` の doc)の 2 つを同じ言葉で言う。時間切れのとき、読みは 3 回目になる | `escalate.go` 365〜384 行(doc と本体)、`director_wait.go` 283〜290 行 | `readInboxWaitOtherTypesText` か `inboxWaitOtherTypesAtTimeout` に改め、doc の「one more read」を「the read for other TYPEs, after the last read of the wait」のように言い分ける |
| L6 | LOW | readability | `TestWaitInbox_TimeoutKeepsItsTextWithNoOpenItemOfAnotherType` の doc は「resolved while the wait runs, before its timeout reads the inbox again」と言うが、`during` は 300ms の時間切れの 50ms 後に resolve するので、「最後の読みと時間切れの読みの間」の窓は試していない。試しているのは「resolve された件は名指されない」だけで、それで足りる(その窓は、計画の Edge cases の「時間切れの直前に外の件が resolve される」に当たるが、フックなしでは作れない) | `director_wait_types_note_test.go` 145〜185 行 | doc を「resolved before the timeout」にし、窓は試していないと 1 文足す |

INFO(直さなくてよい、見たこと):

- `inboxWaitOtherTypes` は 2 回目の読みの失敗を黙って捨てる(`""` を返し、時間切れの文は今のまま)。doc に書いてあり、読めない受信箱のテストもある。次の wait は最初の読みで失敗として返るので、失敗は隠れない
- 外の件の TYPE は、ID と違って長さの上限がない。手で書き換えた `inbox.jsonl` の行だけの話で、本文も同じ信頼の置き方をしているので、直さない(エスケープはある)
- 新しい 2 つのテストの `t.Skipf`(skill の写しが読めなければ skip)は、行 267 の (1) と同じ形で、既存の 6 か所に並ぶ。M2 のとおり、行へ足せば足りる

## Positive notes

- 時間切れの文は `inboxWaitTimeoutPhrase` を先頭のまま、後ろに句を足すだけで、終了コードと skill の照合(`TestDirectorSkill_NamesTheInboxWaitTimeoutPhrase`)は変わらない。`--types` なし、外の件なし、acked だけ、resolve 済み、読めない受信箱のどれも、今の文のまま(テストで固定)
- `WaitInbox` と director の wait が同じ 3 つの helper(`inboxWaitOtherTypes`・`inboxWaitOtherTypesText`・`inboxWaitTimeoutNote`)を通り、`detail` は種類をすでに言うので句を重ねない形になっている。ID は fold が `e<N>` に絞り、TYPE はエスケープする(`ALE\x1b[2JRT` で確かめている)
- 5 件で切る定数(`inboxWaitOtherTypesShown`)は skill の「5 件を超えると、6 件目から」とテストで結ばれ、skill の例の句は `inboxWaitTimeoutNote` の出力から作ってピンしている(本体を変えると、skill の例が落ちる)
- CLI のテストは、部分一致から完全一致に強めた(`org: wait --inbox: …(counted only …; open items of other TYPEs: e1 ALERT)`)
- 重なりのテストは別プロセスの形(wait ごとに `Org` と PID)を作り、`took over` が出るまで待ってから escalate し、先の読みと後の読みの両方を許す(`["e1"]` または `["e1","e2"]`)。notified が付く形と付かない形の両方を固定している
- skill が引く場所(`wait.detail`、`director.wait.detail`、`wait:` と `director:` の行に出ないこと)は、どれもコードで確かめられた

## Tech debt identified

| Debt item | Impact | Why deferred | Trigger to pay down | Related plan/report |
| --- | --- | --- | --- | --- |
| `internal/org/director_wait.go` は 833 行(base 821)で、800 行を超える。この PR が 12 行足した。台帳に大きさの行がない(M2) | 読む量が多い。印の書き込み(`begin`・`save`・`end`・`owns`)、wait の 1 周(`loop`・`probe`)、文を作る関数(`directorWait*Detail`・`*Facts`・`*State`)が 1 ファイルにある | 計画の Scope は時間切れの文で、ファイルの分割は範囲外。分割は move-only の差分になり、レビューの対象を混ぜる | `director_wait.go` の次の変更、または 900 行を超えたとき: 文を作る関数(`directorWaitItemsDetail` から `directorWaitMarkTS`、約 100 行)を `director_wait_text.go` に move-only で出す | この report の M2、`docs/plans/active/2026-11-22-director-wait-types-note.md` |
| `director_skill_sync_test.go` の 800 行超(行 395)と `t.Skipf` の 2 か所(行 267 の (1))が、この PR でも払われない場合(M2) | 1535 行のファイルに、skill の節ごとのピンが並ぶまま増える | 新しいピンを新しいファイルへ置けば払わずに済む(#271・#365 のやり方)。置かないなら、sync-docs が日付つきの繰り越しを足す | 行 395・267 の Trigger のとおり、次の変更。この PR は両方の Trigger を満たす | この report の M2 |

_台帳 `docs/tech-debt/README.md` には追記していない。依頼の範囲が report・計画の Progress・insight event の 1 コミットで、台帳の編集は sync-docs か直しのコミットに入れる(M2 の移し先が決まってから)。上の 2 行は、M2 を直さない場合に必要な文の案。_

行 395 に足す文の案(M2 を直さない場合、数は `wc -l`、HEAD `61dd8ced` のもの): `2026-11-22: fix/director-wait-types-note (#374) hit this trigger with internal/org/director_skill_sync_test.go (1,320 to 1,535 lines: TestDirectorSkillStopsAWaitRe, TestDirectorSkill_RunsOneInboxWaitAtATime, TestDirectorSkill_ReadsTheUncountedItemsOfATypesTimeout and their fragments), but did not pay it: moving the cleanup tests is a move-only change outside its Scope. Carried over to the next change that meets this trigger.` 行 267 には、`WaitInbox` の時間切れの文を変えたことと、(1) の形の `t.Skipf` が 2 か所増えたことを足す。

## Recommendation

- Merge: yes(CRITICAL 0、HIGH 0、MEDIUM 2、LOW 6)
- Follow-ups:
  - M1 は skill の 3 か所に 1 句と、`directorSkillWaitOneAtATimeSteps` に fragment を足す。skill は 4 写し(`scripts/sync-skills.sh`)で、直すなら cross-review の再実行に載せるのが安い。L1 は同じ段落の直しなので、M1 と一緒に直せる
  - M2 は新しいテストを新しいファイルへ移すだけで済み、直すなら `director_skill_sync_test.go` の diff は削除なしの 215 行がなくなる。移さないなら、sync-docs で行 395 と 267 に繰り越しを足し、`director_wait.go` の行を新しく起こす
  - L2〜L6 はテストと doc の直しで、挙動は変わらない。L5 の改名は 3 か所の呼び出しとテストで足りる
  - 範囲外で見たこと: 計画の Progress の S1 の行に「変異 10 個がすべて落ちた」とあるが、その出力は repo の中にない(`/test` の mutation で見る)
