# sync-docs report: send-input-box-marker

- Date: 2026-11-15
- Plan: `docs/plans/active/2026-11-13-send-input-box-marker.md`(承認の digest `6dc6132083c1`)。issue #322
- Branch: `fix/send-input-box-marker`。run 開始時の HEAD は `bffc6934`(origin/main の `c53bf39c`、#329 の PR #330 を取り込んだ merge)、origin/main との merge-base も `c53bf39c`
- Pipeline run: 1 回目(`cycle-count.json` は `cycle` 1)
- 先行 report: self-review `40d7a240`(MEDIUM 1、LOW 6。M1 と L1〜L5 は S3 `0c41fbf8` で直した)、verify `9784fcbf`(pass。V1〜V5)、test `6ebadc63`(pass。変異 19 個のうち 14 個が死んだ。Test gaps 1〜7)。引いた commit は各 report を含む commit で、report の中の `HEAD:` の行の値ではない
- 見た範囲: `git diff origin/main...HEAD`(この commit の前で 35 ファイル、+1,830/-26)。walkthrough は書いていない(`/pr` が決める)
- 制約: 計画の本文と AC の箱は変えていない。変えたのは `## Progress checklist` だけで、`./scripts/plan-visual.sh digest` は編集の前後とも `6dc6132083c1`、`git diff --numstat` は 4/2(足した 4 行と、箱 2 行の書き換え)。Go のコードとテストは変えていない。テストの全体は回していない(handoff の指示)
- 差分の大きさ(`/pr` が walkthrough を決める材料。sync-docs の commit の前の数): 上の 35 ファイルのうち、docs 以外は 28 ファイルで +1,029/-26。そのうちテストと fixture と prompt と skill を除いた Go の本体は 1 ファイル、`internal/org/send_screen.go` の +110/-22 だけ

## Summary

verify の V1〜V4 と、evidence の fixture 一覧を直した。V5(INFO)は Go の comment なので、handoff のとおり触れていない。

- V1: org runtime の仕様に `### 2026-11-13 改訂(fix/send-input-box-marker)` を足した((a) Enter の前の確かめ、(b) 変えていないもの、(c) `(text typed but not submitted)` の手順、閉じの項)
- V2: tech-debt に `Known gaps of fix/send-input-box-marker (#322)` の 1 行(498 行目)を足した。項目は handoff の (a)〜(g)
- V3: handoff の 9 行(180、188、262、380、460、464、482、483、484)に持ち越しの note を足した
- V4: 482 行目 (L11) と 483 行目 (9) に、日付つきの note を足した。古い文は書き換えていない
- evidence: fixture 11 の一覧と sha256 を足した

handoff の 9 行のほかに、掃引で 3 行(365、473、497)に note を足し、`agent-messaging.md`(2 つの写し)に 1 段落を足した。

## Changes made

| File | Change |
|------|--------|
| `docs/tech-debt/README.md` | 既存の 12 行に挿入だけをし、新しい 1 行を末尾に足した(498 行目)。12 行は 180、188、262、365、380、460、464、473、482、483、484、497。文の書き換えと削除は 0(下の「Verification」) |
| `docs/specs/2026-08-01-org-runtime.md` | `### 2026-11-13 改訂(fix/send-input-box-marker)` の節。2026-11-13 の最後の節(fix/send-claude-paste)の次、`### 運用ノート` の前 |
| `docs/evidence/org-send-marker-probe-2026-11-13.md` | 「fixture の一覧と sha256」の節。11 fixture と、写した画面と、sha256 |
| `.claude/rules/ralph/agent-messaging.md` と `templates/base/.claude/rules/ralph/agent-messaging.md` | 「How send types a message」に、`(text typed but not submitted)` の手順の 1 段落。2 つの写しは `cmp` で同じ |
| `docs/plans/active/2026-11-13-send-input-box-marker.md` | `Verification artifact created` と `Test artifact created` を `[x]` にし、`Implementation started` の下に 2 行(verify と test の commit、sync-docs の計画からのずれ)を足した |
| `docs/reports/sync-docs-2026-11-13-send-input-box-marker.md` | この report |
| `docs/insights/events/2026-11-15-send-input-box-marker.jsonl` | `sync_docs` の行(cycle 1、pass)。1 行から 2 行 |

commit は 1 つ(`docs: sync docs for send-input-box-marker`)。

## V1: 仕様の改訂の節

`2026-11-13 改訂(fix/send-claude-paste)` の次に足した。#308 と #319 の節と同じ形で、(a)〜(c) と閉じの項から成る。

- (a) Enter の前の確かめが、打った文の中の印の行を入力欄の頭と取り違えなくなったこと。D2 の 5 つの条件を、コード(`sendTypedMarkerLineBox`)と同じ順で書いた。条件を満たさない画面と、claude でも codex でもない driver は今までの探し方に戻ること、選択の画面などで Enter を押さない向きが変わらないこと、実機の画面と fixture 11 の場所も書いた
- (b) 変えていないもの。Enter のあとの確かめ(`SendScreenSubmitted`)の読み方、resume の画面の型の関数 3 つと `ClassifySendScreen` のコード、前の send の残りの文が入力欄にあるときの扱い(画面の道は何も打たず終了コード 1)
- (c) `(text typed but not submitted)` で終わった send の手順。leader は pane を読んで BLOCKED を上げ、キーを送らない。director は leader の画面を読み、DECISION の全文が残っているときだけ Enter を 1 回。どちらも打ち直さない。note の `clear or submit it there` に leader は従わないこと
- 閉じの項: 本文は書き換えない。2026-11-12 改訂 (c) の、Enter の前の確かめが番号付きの行を入力欄に入ったと読む条件は、この改訂の (a) の道を足して読む

verify が言うとおり、2026-11-12 改訂 (c) の説明は今も正しいので、書き換えていない。その節に印は付けていない(閉じの項が読み方を示す)。

`/natural-japanese` の lint(`uv run lint.py`)を、節を scratch に抜き出して当てた。指摘は「和欧文間の半角空白」1 件だけで、spec の周りの節と同じ書き方(英字の前後に空白を入れる)なので残した。

## V2: Known gaps の行(498 行目)

`Known gaps of fix/send-input-box-marker (#322)` の 1 行。5 つの列に (a)〜(g) を持たせた。#329 の行(495〜497)の次、末尾に足した。

| 項目 | 内容(handoff のとおり) | 足したもの |
|----|----|----|
| (a) | 頭が 0 桁目で、続きが 2 字下げという形への依存 | Claude Code 2.1.316、codex-cli 0.166.0、幅 53 桁。ほかの版で頭が字下げされると、新しい道は働かず、今までどおり `text typed but not submitted` |
| (b) | claude の返事のあとの画面を見ていない | probe の claude の手番が 5 時間の上限で止まった。Enter の 0.3 秒後の画面だけで Enter のあとの確かめを固めた。外れたときの影響は `submit_unconfirmed=true`(Enter は押し直さない) |
| (c) | 画面の道を ralph で通しで走らせていない | herdr が `working` と言い続ける状態を作れない。#308 の (a) と同じ |
| (d) | 打つ前の確かめ(`ClassifySendScreen`)は、字下げされた印の行を持つ残りの文のある入力欄を見つけない | 新しい fixture の行(`sendScreenFixtureWants`)が `input box has text`、`no input box`、`choice dialog` と読むことを書いた。send は何も打たず終了コード 1(知らせる側)。Trigger に、`ClassifySendScreen` には打った文がないので D2 の条件 2 が使えないことを書いた |
| (e) | 印だけの行(`❯` か `›` の後ろに何もない)を含む文は新しい道に入らず、前と同じに止まる | self-review L4。`sendTextInWant` は空の文をどの文の一部ともみない |
| (f) | codex の座席には素の文を打つ | bracketed paste は #326 |
| (g) | test の report の Test gaps 1〜4 | `head < 0` の return を通るテストがない(外すと index -1 で panic、overlay の probe で確かめた)。字下げ 1 字の行と 2 字の行を分ける行がない(条件 3)。skill の D3 の文は 4 つの写しの一致でしか固まっていない。prompt のテストの `sendTypedNotSubmittedMark` は `verbs.go` のエラーの文とは別に書かれた文字列 |

Related は、計画(`docs/plans/active/...`)、evidence、self-review の L4、verify の V2、test の Test gaps 1〜4。`archive-plan.sh` が `/pr` で `archive/` に書き換えるので、`archive/` とは書いていない。

test の Test gaps 5(実機の側)は (a)(b)、6(Non-goals の 2 つ)は (d)(f)、7(`-race` を回していない。この diff は goroutine もロックも変えていない)は行にしない。変異のうち M4a、M4g、M5b は知られた画面では等価なので、行にしていない。

## V3: Trigger に当たった行への持ち越しの note

handoff の 9 行は、行頭の語で対象を確かめた(番号は verify の時点から動いていない。#329 の merge で足された行は 495 行目以降だけ)。これまでの note(`2026-11-13: fix/send-claude-paste (#319) …`)の形に合わせ、Trigger セルの末尾に note を足した。note の日付は、足した日(2026-11-15)。Related セルの末尾に `docs/plans/active/2026-11-13-send-input-box-marker.md (Scope; the carry-over note)` を足した。

| 行(先頭の語) | 当たった所 | 書いたこと |
|----|----|----|
| 180「Two documentation findings of org-stop-all」(a) | `/org` skill の `send` の行。表の行は 3,758 から 4,086 字(`wc` で測り直した。4 つの写し) | `stop`(1,208 字)と `disband`(1,219 字)は変わらず、表の組み替えは計画の Scope の外。持ち越し |
| 188「What a leader is told about its reservation」(a) | `leader.md` の typed protocol の節に 1 段落。1,332 から 1,344 行(origin/main の `c53bf39c` と HEAD で `wc -l`) | `RolePromptVars` と `{{SCOPE}}` は変わらない。(b) は当たらない。持ち越し |
| 262「Three test gaps of the org-director-attach PR」 | `/director` skill の `### 答える順`(`## 受信箱の処理` の中)の手順 2 に 1 段落 | `director_skill_sync_test.go` と `TestDirectorSkill_NamesTheInboxWaitTimeoutPhrase` は触れていない。(1) 〜 (3) は開いたまま。持ち越し |
| 380「RC6 … the `/director` skill's check after a send」 | 同じ段落。send と確かめの手順の変更 | 入力欄から流れた DECISION を見分ける別の印は書いていない。`ralph org read` は変わらない。持ち越し |
| 460「The recognition of codex's footer」 | この PR の probe が codex-cli 0.166.0 の画面を取り直した(「次の probe」の字義に当たる) | footer の形は 2026-11-12 の evidence と同じ(`Context 0% used · main · No chan…` と `⚠ 11 · f2`)で、`% left` の形は出なかった。`resumeCodexFooterLine` は変えていない。(b) と、reason に footer の形を書くことは開いたまま。Related に evidence を足した |
| 464「The `/director` skill … do not say what a director does when it sees the scaffold version note」 | `/director` skill(4 つの写し)と `director.md`(ミッションの箇条)の編集 | skill と prompt の 5 ファイルに `scaffold`、`manifest.toml`、`ralph upgrade` の語は 0 件(`git grep -c -i`、終了コード 1)。持ち越し |
| 482「Seven small findings of fix/send-screen-fallback」(L2、L7、L9、L10)と (L11) | `send_screen.go` の変更。`sendTypedMarkerLineBox` は `resumeMarkerLineBox` の 3 つ目の呼び手で、`sendTypedInputBox` も変わった | 5 つとも払っていない。新しい関数は driver を `switch` で見るので、(L7) の `resumeScreenSentences[driver]` の数は増えない(`send_screen.go` で 3 のまま)。3 つ目の呼び手も `marker` と `first` だけを読む。持ち越し |
| 483「Test gaps of fix/send-screen-fallback」(8)(9) | `sendTypedInputBox` と `sendCodexHasFooter` の変更 | (8) の行は足していない(番号付きの道の shape の条件は前のまま、panic で固まっている)。(9) の codex の画面は足していない。新しい codex の fixture(箱に文がある 4 つ)は、`⚠ 11 · f2` の行が footer の下にあるので `sendCodexFooterInInput` の道を通る。新しい関数には、assert で固めた行がある。持ち越し |
| 484「`internal/org/send_screen_test.go` (662 → 831 lines)」 | `send_screen_test.go` に fixture の表の 11 行と `TestSendScreenSubmitted` の 2 行(+28 行)。S1 でテストも足して 1,160 行にしたが、`0c41fbf8`(self-review の M1)が新しいテストを新しいファイルに移した | 番号付きの行のテストは動かしていない(move-only の分割は計画の Scope の外)。持ち越し。数は下の「行数」 |

### 行数(484 行目の note)

handoff は「`send_screen_test.go` は 859 行(831 に既存の表の 28 行)」と書いた。859 は、この branch が origin/main に取り込む前の数(`0c41fbf8`、base は `bbe2807d`)である。HEAD では #329 の merge(`fddda578` が表に 1 行、8 行)が入っているので、`wc -l` は 867 になる。note には、4 つの数を並べて書いた。

| 時点 | `send_screen_test.go` |
|----|----|
| `bbe2807d`(#308 のあとの base) | 831 |
| origin/main の `c53bf39c`(#329 の merge のあと) | 839(+8) |
| `0c41fbf8`(この branch の最後の Go の commit。base は `bbe2807d`) | 859(+28) |
| HEAD の `bffc6934`(merge のあと。この commit の前) | 867 |

`send_fallback_test.go` は 835 行で、`bbe2807d` から変わっていない(HEAD と origin/main で同じ。`git diff origin/main...HEAD` に入らない)。新しいテストは `send_screen_marker_test.go`(337 行)と `send_fallback_marker_test.go`(53 行)に置いた。

### 掃引で足した 3 行

handoff の外で、Trigger を 1 つずつ読んで足した(`td_sweep` で名前を拾い、当たった行を目で読んだ)。note の形は前の note と同じ。どれも判断が分かれるので、「残っていること」にも書く。

| 行(先頭の語) | 当たった所 | 書いたこと |
|----|----|----|
| 365「`internal/org/verbs.go` is 2277 lines」 | Trigger は「`Send` か leader template tests の次の変更」。この PR の leader の prompt のテストは新しいファイル(`prompts_send_unsubmitted_test.go`、113 行)にある。feat/leader-plan-depth(2026-10-29)が、同じ形(新しいファイルに leader template のテスト)で当てた前例 | `verbs.go`(2615)、`verbs_test.go`(5893)、`prompts_test.go`(2722)、`spawn.go`(3101)、`spawn_test.go`(6408)、`director_skill_sync_test.go`(1307)は `wc -l` で変わらない。`Send` も変えていない。`leader.md` は 1,332 から 1,344 行。move-only の分割は Scope の外。持ち越し |
| 473「Known gaps of fix/send-screen-fallback (#308)」(b) | Trigger の「番号付きの TASK を本物の claude か codex の座席に送り、箱の画面を取って、合成の 2 つの fixture の代わりか隣に置く」。この PR の probe は codex に番号付きの行を含む文を送り、`codex-input-typed-marker-numbered.txt` を隣に置いた。verify の INFO が挙げた行 | `sendNumberedStepsMsg` ではない文で、本物の画面では番号付きの行(`  › 2. explain this step`)が頭の下に字下げされて出た。合成の fixture は番号付きの行が頭にある形なので、本物の画面は取れていない。合成の 2 つは残す。(b) は開いたまま。Related に evidence を足した |
| 497「Test gaps of fix/claude-5h-usage-limit (#329)」 | Trigger の「次に `usage_limit_screen_test.go` に足すテスト」。この PR は `usageLimitFixtureWants` に 11 行(新しい fixture は上限なし。`TestUsageLimitScreen_EveryFixture` が全 fixture の行を求める)を足した | (a)〜(d) は fixture の行ではないので払っていない。Trigger の字義に当たる境界の判断 |

### 当たらないとした行(代表)

| 行(先頭の語) | 判断 |
|----|----|
| 226 | Trigger は `Send` の doc と `approval_test.go`。この PR は `Send` を変えていない |
| 337・338 | Trigger は `Send` の idle の待ちの変更。変えていない |
| 357「`internal/org/watch.go` is 1848 lines」 | Trigger は watch の escalation のコードとテスト。この PR は watch に触れていない(`usage_limit_screen_test.go` に 14 行足したが、Trigger はそのファイルを名指さない) |
| 426「The role prompts grew with org-cross-review」 | Trigger は「`leader.md` か `reviewer.md` に節を足す変更」。この PR は typed protocol の節の中に 1 段落を足しただけで、節を足していない |
| 457・461・477 | 457 の Trigger は `ClassifyResumeScreen`、`ResumeInputHolds` などの変更で、この PR は両方のコードを変えていない(`TestClassifyResumeScreen_Fixtures` に fixture の行を足しただけ。テストの行だけでは「次の変更」に当たらない)。461 は `resumeCodexFooterLine`、`StopGlyphs`、`empty` の placeholder の規則、477 は画面の規則と watch の ALERT の条件で、どれも変えていない |
| 177・195・199・206〜217・244〜252 | director の stage 8a の話(自動 merge や headless の director)。この PR の director の編集は skill の手順と prompt の箇条 1 つ |
| 200・384・377・426 の `leader.md` の節 | 200 は feature-org の節、384 は step 14、377 は feature-org の step。この PR の段落は typed protocol の節 |
| 372・374・376・379・367 | leader template の spawn の step、reviewer の prompt、watch の `scope_change`、PR の step、gate の変更。どれも変えていない |
| 180 以外の `/org` skill の行(186・340・355・401) | 186 は `reserveAgain` などの関数、340 は解決済み、355 は `InboxDeadlineDue`、401 は `checkRetiredOrgStateDirEnv`。`send` の行の編集はどれにも当たらない |
| 450〜452 | `Waiting` の文と、待ちの画面の話。この PR は `resume_screens.go` の表を変えていない(Go の本体は `send_screen.go` だけ) |

## V4: 482 行目 (L11) と 483 行目 (9) の古い文

古い文は書き換えず、各項目の末尾に日付つきの note(括弧)を足した。

- 482 行目 (L11): 「both callers, `findResumeInputBox` and `sendTypedInputBox`」は 3 つになった(`sendTypedMarkerLineBox` が加わった)。3 つ目は半分だけ埋まった箱から `marker` と `first` だけを読み、`resumeInputBoxShape` には新しく作った箱を渡すので、「`text` に半端な箱が届かない」は今も成り立つ。数だけが古い
- 483 行目 (9): `if box.footerStart < len(lines) { return true }` は S3 `0c41fbf8` の書き換えでなくなった。`sendCodexHasFooter` は `sendCodexFooterStart` を呼び、それは `sendCodexFooterInInput` の footer を先に返し、そのあとで `box.footerStart, box.footerStart < len(lines)` を返す。(9) の変異は、この最後の return を `0, false` に替えるものになる。test の report は走らせていない(M4g はその 2 つの順を入れ替えるもので、知られた画面では等価として生き残った)

## evidence: fixture 11 の一覧

`docs/evidence/org-send-marker-probe-2026-11-13.md` の「この確かめから決めたこと」と「確かめで残ったもの」の間に節を足した。fixture 11 と、写した画面(evidence の名前: codex の short、last、medium、numbered と Enter から 0.3 秒後、claude の two-nl、one-nl、no-nl、short、折り返しの Enter の前と Enter から 0.3 秒後)と、sha256 を表にした。

- 対応は、計画の Affected areas と verify の AC6 の対応を、scratchpad の probe の画面(`probe322/out/` の 43 ファイル)と `cmp` で確かめ直した。11 すべてバイト単位で一致した(`codex-input-typed-marker-line` は `x1-short.typed.txt`、`-line-last` は `x2-last`、`-line-scrolled` は `x3-medium`、`-numbered` は `x4-numbered`、`claude-input-typed-marker-line` は `c5-two-nl`、`-one-nl` は `c6-one-nl`、`-no-nl` は `c7-no-nl`、`claude-input-typed-paste-short` は `c1-short`、`-line-wrapped` は `cs-submit.typed.txt`、送れた 2 画面は `cs-submit.enter03.txt` と `xs-submit.enter03.txt`)。probe の生の画面は scratchpad にだけあり、repo には入れていない。表の sha256 は fixture の中身から取ったので、あとから repo の中だけで写しのままかを確かめられる
- 表の sha256 は、`hashlib` で fixture 11 のバイト列から取り直した値と 11 すべて一致した(`evidence_hash_check`)

## 掃引で足した文書の直し: `agent-messaging.md`

verify の INFO が「足すかは `/sync-docs` が決めてよい」とした。「How send types a message」は、打つあいだの失敗(届いたか分からない場合)だけを扱い、`(text typed but not submitted)`(打ち終わって Enter を押していない)の手順は持たなかった。手順は role prompt と skill にあるが、rule は seat と director が読む契約の文書なので、1 段落を足した(2 つの写し)。

- 同じ send を打ち直さない(2 本目は残った文のあとに打たれ、1 回の Enter で 1 つのメッセージとして送られる)
- leader は `ralph org read` で座席の pane を読み、キーを送らず、`ralph org escalate` で BLOCKED を上げる
- director は leader の pane を読み、DECISION の全文が箱にあり、ほかの画面が出ていないときだけ Enter を 1 回(DECISION の送り方の一部)。それ以外は何も押さず、人に伝える

前の段落(「leftover を見つけた director と leader は人に伝える」)は書き換えていない。計画の `## Affected areas` にない文書なので、計画の Progress に「計画からのずれ」として書いた。

## V5(INFO)

`internal/org/send_screen.go` 先頭の comment の「claude shows a long piece of it as a paste placeholder (`[Pasted text #1 +17 lines]`; evidence section 10)」は、#319 より前の素の打ち方の画面の説明である。この PR の evidence では、bracketed paste で打った claude は 67 バイト・改行 3 つの短い文でも placeholder に畳んだ。Go の comment なので、handoff の指示どおり触れていない。規則の動きには関係しない。`send_screen.go` を次に変えるときに、「改行が 3 つ以上の文は畳まれる」を足すとよい。tech-debt の行にはしていない(comment の言い回しで、挙動の穴ではないため)。

## 残っていること(lead へ)

- 決めてほしい点: (1) 掃引で足した 3 行(365、473、497)の note は境界の判断(365 は「leader template tests」の字義、497 は「次に足すテスト」の字義、473 は Trigger の字義に近い probe)。足さないなら、各行の Trigger の末尾の note の文と、Related の 1 句を消す。(2) `agent-messaging.md` の段落(2 つの写し)は計画の外。要らなければ 2 か所の同じ段落を消す。(3) 484 行目の行数は、handoff の 859 ではなく 867 を書いた(上の「行数」)
- 触れなかった同じ言い方: `/org` skill の `send` の行の「note の指示に従う」と、leader の段落の「note の `clear or submit it there` に従わない」の関係は、self-review の L1 のとおり、leader の段落が優先すると書いてある(S3 で直した)。ここでは変えていない
- 払っていないもの: note を足した 12 行のうち、Known gaps の行以外はすべて持ち越し。V5 は残す
- `docs/tech-debt/README.md` と `docs/specs/2026-08-01-org-runtime.md` は、ほかのレーンも編集しうる。この run の tech-debt の編集は、既存の文を消さず、セルの末尾への挿入が中心(文の書き換えは 0)。merge の衝突は、末尾の行(498)と、262、464 行目の Related の末尾(#329 と同じ場所に足した)で起きやすい。仕様の節は、2026-11-13 の最後の節(fix/send-claude-paste)の次に足した。同じ日付の節が別の PR から足されたときは、順番を直す
- `docs/plans/active/...` への参照を Related に足した。`/pr` の `archive-plan.sh` が書き換える

## Verification

- `git show HEAD:docs/tech-debt/README.md` との比較(`td_diffcheck`、文字単位の `difflib`)で、変わった行は 180、188、262、365、380、460、464、473、482、483、484、497 の 12 行と、末尾に足した 498 行目だけ。12 行はどれも挿入だけ(`replace` と `delete` は 0 ブロック)。どの行も列は 5(`(?<!\\) \| ` で分けた)、バッククォートと `~~` は各列で偶数。足した文に `file:line` は 0(行が元から持つものも 0)。U+FFFD は 0
- 計画の参照: `docs/tech-debt/README.md` の `docs/plans/(active|archive)/...` を python で調べて、無い参照が 0 件(126 件)。`docs/plans/active/2026-11-13-send-input-box-marker.md` を指す Related は 11 行(180、188、262、365、380、464、482、483、484、497、498)
- 表の sha256: 11 行を python の `hashlib` で fixture から取り直して、evidence の表と 11 行すべて一致
- `./scripts/plan-visual.sh digest` は編集の前後とも `6dc6132083c1`。`git diff --numstat` は 4/2 行
- `./scripts/check-sync.sh` は `PASS: all files in sync.`(KNOWN_DIFF 6)、`./scripts/check-skill-sync.sh` は 14 skill で ok、`./scripts/check-pipeline-sync.sh` は ok、`./scripts/check-template-purity.sh` は PASS。`cmp` で `agent-messaging.md` の 2 つの写しは同じ。`git diff --check` は rc 0
- `/natural-japanese` の lint を仕様の新しい節(scratch に抜き出した)に当てた。指摘は「和欧文間の半角空白」だけで、周りの節と同じ書き方なので残した
- 実行していないもの: テスト全体(`run-test.sh`、`go test ./...`。handoff の指示)、`run-verify.sh`、`run-static-verify.sh`。Go のコードとテストは変えていない

## Insight event

```
./scripts/insights-append.sh --slug send-input-box-marker --flow standard --phase sync_docs \
  --verdict pass --cycle auto --source skill
```

終了コード 0。`docs/insights/events/2026-11-15-send-input-box-marker.jsonl` に足した行は cycle 1、pass、ts は 2026-11-15T11:12:22Z。この時点の `git status --short` は `M` が 7 つで、迷子の `??` の jsonl は無かった(`??` は、この report だけになる)。
