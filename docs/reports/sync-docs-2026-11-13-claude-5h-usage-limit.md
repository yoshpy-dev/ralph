# sync-docs report: claude-5h-usage-limit

- Date: 2026-11-15
- Plan: `docs/plans/active/2026-11-13-claude-5h-usage-limit.md`(承認の digest `be05f6884990`)。issue #329
- Branch: `fix/claude-5h-usage-limit`。run 開始時の HEAD は `abd687f2`、origin/main との merge-base は `bbe2807d`
- Pipeline run: 1 回目(`cycle-count.json` は `cycle` 1)
- 先行 report: self-review `276f7a47`(再確認 `6fd0d2a4`、再確認 2 `8aebe087`。Merge: yes、未解決は受け入れ済みの L4 だけ)、verify `70120d4f`(pass。LOW 5 件 V1〜V5、INFO 3 件)、test `abd687f2`(pass。変異 7 個すべて死んだ。Test gaps 1〜8)。引いた commit は各 report を含む commit で、report の中の `HEAD:` の行の値ではない
- 見た範囲: `git diff bbe2807d...HEAD`(この commit の前で 22 ファイル、+1,101/-79)。walkthrough は書いていない(`/pr` が決める)
- 制約: 計画の本文と AC の箱は変えていない。変えたのは `## Progress checklist` だけで、`./scripts/plan-visual.sh digest` は編集の前後とも `be05f6884990`、`git diff --numstat` は 4/2(箱 2 つと足した 2 行)。Go のコードとテストは変えていない。テストの全体は回していない(handoff の指示)
- 差分の大きさ(`/pr` が walkthrough を決める材料。sync-docs の commit の前の数): 上の 22 ファイルのうち、docs 以外は 9 ファイルで +445/-56。そのうちテストと fixture を除くと Go の 4 ファイルで +142/-39(`internal/cli/org_watch.go` +8/-7、`internal/org/resume_screens.go` +34/-1、`usage_limit_screen.go` +75/-19、`watch_usage_limit.go` +25/-12)

## Summary

verify の V1〜V5 をすべて扱った。V1 は `/director` skill の 2 文(4 つの写し)、V4 と V5 は org runtime の仕様と tech-debt の 495 行目、V3 は tech-debt の Trigger に当たった行への持ち越しの note。V2 は handoff の指示どおり `/org` skill に触れていない。test の Test gaps 3〜5 と 7 は新しい 1 行(497 行目)にした。Gaps 1 と 2 は既存の行(496、452)、Gap 6 は 487 行目の (c)、Gap 8 は行にしない。

handoff の 8 行のほかに、掃引で 3 行見つけて note を足した。262 行目と 464 行目は、V1 の skill の編集がこの 2 行の Trigger に当たる(自分の編集)。357 行目は、行が引く 3 つのファイルの行数がこの branch で動いた。

## Changes made

| File | Change |
|------|--------|
| `.claude/skills/director/SKILL.md` と、`.agents/skills/director/SKILL.md`、`templates/base/.claude/skills/director/SKILL.md`、`templates/base/.agents/skills/director/SKILL.md` | `### 件の処理` の ALERT の項の 2 文を、上限の解除を待つ座席にも当てはまる文にした(V1)。4 つの写しは `cmp` で同じ(sha256 の先頭は `bd2adb83`) |
| `docs/specs/2026-08-01-org-runtime.md` | 2026-11-13 改訂(fix/claude-5h-usage-limit)の (c) の 1 句(V4)と、最後の項(V5)。同じ 1 つの節の 2 か所 |
| `docs/tech-debt/README.md` | 既存の 12 行を編集し、新しい 1 行を末尾に足した(497 行目)。11 行は Trigger に note と Related に計画の path(262、357、457、464、477、484、485、487、488、489、492)。488 行目は Debt に 1 文。495 行目は Why deferred の 1 句(V4)。下の「V3」「V4」「Test gaps 3〜7」と「掃引」 |
| `docs/plans/active/2026-11-13-claude-5h-usage-limit.md` | `Verification artifact created` と `Test artifact created` を `[x]` にし、`Implementation started` の下に 2 行(verify と test の commit、sync-docs の director skill のずれ)を足した |
| `docs/reports/sync-docs-2026-11-13-claude-5h-usage-limit.md` | この report |
| `docs/insights/events/2026-11-15-claude-5h-usage-limit.jsonl` | `sync_docs` の行(cycle 1、pass)。1 行から 2 行 |

commit は 1 つ(`docs: sync docs for claude-5h-usage-limit`)。

## V1: `/director` skill の 2 文

計画の Non-goals は「director の skill の `usage_limited` の扱いを変えること」を外していた。この PR のあと、上限の解除を待つ座席も `usage_limited` の件に入る(D1)ので、次の 2 文は偽になった。計画からのずれとして、計画の Progress に理由を書いた(計画の本文は変えていない)。

1. `REASON` の説明「`usage_limit` は org の座席が利用上限で止まったこと」: 待つ座席だけの件でも `REASON` は `usage_limit`。
2. 「履歴の `recurred` の行の `note=` に座席が出ていれば、その座席も上限で止まったということで、同じく扱う」: `recurred` で足された座席が待つ座席なら、止まっていない。

新しい文は次のとおり。

```
`usage_limit` は org の座席が利用上限に当たって、
止まっているか、解除を待っていること)。
```

```
履歴の `recurred` の行の `note=` に座席が出ていれば、その座席も上限に
当たったということで、同じく扱う。解除を待っている座席の行には
`Continuing automatically at …` が出て、解除の時刻に CLI が自分で続ける
ので、その座席は `ralph org resume` を待たずに続く。
```

2 文目の「解除を待っている座席の行には…」の 2 行は、handoff の「待つ座席の行には `Continuing automatically at …` が出て、解除の時刻に CLI が自分で続けるので、director は `ralph org resume` を要しない」を skill の文にしたもので、足すか決めてほしい点として「残っていること」にも書く。1 文目は折り返しで `ralph org inbox show` の行を 1 行送っただけで、ほかの行は動かしていない。足した行の最長は 78 桁(全角を 2 と数える)。

編集は `.claude/skills/director/SKILL.md` と `templates/base/.claude/skills/director/SKILL.md` にだけ手でして、`./scripts/sync-skills.sh` で `.agents/skills/` を作り直し、`CLAUDE_ROOT=templates/base/.claude/skills CODEX_ROOT=templates/base/.agents/skills ./scripts/sync-skills.sh` で雛形の `.agents` 側を作った。2 回とも終了コード 0。

テストが古い言い方を固めていないかを `git grep --line-number` で探した。`internal/` に「org の座席が利用上限で」「その座席も上限で」「止まったということで、同じく扱う」の字面を持つファイルは無い。skill を読むテスト(`director_skill_sync_test.go`、`skill_doc_followup_test.go`、`skill_split_questions_test.go`)で `usage_limit`、`SEATS`、`recurred` を含む断片は 0 件で、「止まった」を含む断片は片付けや `SKIPPED:` の話だけ。テストは書き換えていない。

## V2: `/org` skill の `usage_limited` の項

`.claude/skills/org/SKILL.md` の「受信箱」節の `usage_limited` の項(647〜654 行目)と、見張りの ALERT の件を挙げる 612〜613 行目は、止まった座席の道だけを書く。偽ではない。レーン B の #322 が `/org` skill を編集するので、4 つの写しのどれにも触れていない。#322 の merge のあとに足すなら、同じ 1 句(待つ座席にも条件が立つこと、行に `Continuing automatically at …` が出ること)で足りる。

## V3: Trigger に当たった行への持ち越しの note

handoff の 8 行は、行頭の語で対象を確かめた(番号は verify の時点から動いていない)。これまでの note(`2026-11-13: fix/send-claude-paste (#319) …`)の形に合わせ、Trigger セルの末尾に note、Related セルの末尾に `docs/plans/active/2026-11-13-claude-5h-usage-limit.md (Scope; the carry-over note)` を足した。`archive-plan.sh` が `/pr` で `archive/` に書き換えるので、`archive/` とは書いていない。数は `wc -l` で測り直した(origin/main の `bbe2807d` と、branch の最後の Go の commit `dd3b1e58`。HEAD は同じ数)。

| 行(先頭の語) | 当たった所 | 書いたこと |
|----|----|----|
| 457「Seven mutants of `ralph org resume`」 | `ClassifyResumeScreen`。`6cf988d5` が footer の切り出しを `footerBlocks` に替えた(どの fixture でも結果は同じ) | (d) の fixture(enter-only の文と待ちの文の両方)は払っていない。新しい fixture は止まった文と待ちの文の組。持ち越し |
| 477「A hook that was changed after codex approved it」 | 画面の規則(5 時間の上限の文、`UsageLimitWaiting`)と、見張りの ALERT の条件(待つ座席が `usage_limited` に入る) | 「Hooks need review」の条件は足していない。`readUsageLimitScreen` は #312a のまま。持ち越し |
| 484「`send_screen_test.go` (662 → 831 lines)」 | `fddda578` が fixture の表に 1 行(831 → 839 行)。`send_fallback_test.go`(835)は触れていない | 番号つきの行のテストは動かしていない(move-only の分割は計画の Scope の外)。持ち越し |
| 485「A seat in the set of the org condition `usage_limited`」 | `recordUsageLimitRecurrence`(`dd3b1e58` が stderr の 2 行の語を変えた) | 読みが 1 回失敗した座席の扱いは触れていない。`readUsageLimitScreen`、`raiseUsageLimited`、`evaluateSeat` の stall の分岐は同じ。持ち越し |
| 487「Test gaps of feat/watch-usage-limited」 | `watch_usage_limit.go` とそのテストの両方。語の変更(`usageLimitedWhy`、`usageLimitedAlertMessage`、`usageLimitRecurNote`、stderr の 2 行)と、待つ画面のテスト 2 つ | (a) は足していない。(c) の `recur` が失敗したときの stderr の行は、語を変えたのに今もどのテストも固めていない(test の Test gaps 6)。(d) の 2 つ目の acked の行もない。Related に test の report を足した。持ち越し |
| 488「`watch_usage_limit_test.go` is 936 lines」 | 同じファイルに 2 つのテストと fixture の method(`screenText`)を足した | 936 から 999(`fddda578`)、1000(`6cf988d5`)。Debt の「this file did not grow」は古くなったので、Debt の末尾に日付つきの 1 文を足し、Trigger に数を書いた。fixture と helper は先頭の 252 行で、動かすと約 750 行になる。持ち越し |
| 489「The `recurred` note of a grown `usage_limited` condition」 | `usageLimitRecurNote`。頭の文が「stopped on a usage limit: 」(26 文字)から「hit a usage limit: 」(19 文字)に変わった(`dd3b1e58`) | 500 文字の上限で座席に使える文字が 7 つ増えただけで、切り詰めと「全部の id の場所を指す」案は払っていない。同じ commit が第 1 セルの stderr の引用を直したことも書いた。持ち越し |
| 492「Two LOW findings of the cycle 2 self-review」 | (a) `recordUsageLimitRecurrence`(stderr の語)。(b) 489 行目の編集 | (a) の read-time の節は消していないし doc にも書いていない。(b) 489 行目の Related は今も triage を「WORTH_CONSIDERING 2」と番号で引く。持ち越し |

### 掃引で足した 3 行

handoff の外で、Trigger を 1 つずつ読んで足した。足した note は、前の note と同じ形である。

| 行(先頭の語) | 当たった所 | 書いたこと |
|----|----|----|
| 262「Three test gaps of the org-director-attach PR」 | Trigger は「受信箱の処理」節の次の変更。V1 の編集は `### 件の処理`(`## 受信箱の処理` の中)の 2 文 | `director_skill_sync_test.go`、`TestDirectorSkill_NamesTheInboxWaitTimeoutPhrase`、`WaitInbox` は触れていない。(1) 〜 (3) は開いたまま。持ち越し。自分の編集が当てた行 |
| 464「The `/director` skill … do not say what a director does when it sees the scaffold version note」 | Trigger は `/director` skill の次の変更 | 同じ形で当たってきた前の PR の note(feat/watch-usage-limited も `### 件の処理` を編集)に合わせた。skill は今も scaffold、`manifest.toml`、`ralph upgrade` を名指さない(`git grep -c -i` が 0 件)。持ち越し。自分の編集が当てた行 |
| 357「`internal/org/watch.go` is 1848 lines」 | 境界。`usageLimitedWhy` と `usageLimitedAlertMessage`(escalation の本文)の語を変えた。`watch.go` と `watch_test.go`、`watch_all_test.go` には触れていない | 行が数えている 3 つのファイルが動いた: `watch_usage_limit.go` 483 行(470)、`usage_limit_screen.go` 152(96)、`usage_limit_screen_test.go` 584(402)。移す作業は払っていない。持ち越し。境界の判断なので「残っていること」にも書く |

### 当たらないとした行(代表)

| 行(先頭の語) | 判断 |
|----|----|
| 88「6 batchable LOW findings」 | Trigger の項目が名指すのは `status.go`、`spawn_test.go`、`watch.go` の `SEAT:` の header など。この branch は `internal/cli/org_watch.go` の `--help` と `watch_usage_limit.go` だけで、どれにも触れていない |
| 118・180 | `quality-gates.md` の雛形と `/org` skill の command table は、この branch が触れていない |
| 353「The watch's ALERTs to the leaders failed 10 times」 | Trigger は `sendAlert` と `SendWatchdogAlert`。どちらも変えていない |
| 450・451 | 450 は `Waiting` の文を `ralph status` に出す段階 2 の話。451 は codex の長い待ち。どちらも `Waiting` の文そのものを変えていない(新しい欄 `UsageLimitWaiting` は印) |
| 460・461・473・483 | 460 と 461 は `resumeCodexFooterLine`、`StopGlyphs`、`empty` の placeholder の規則。473(b) は Claude Code か codex の upgrade で fixture を足す話で、この fixture は同じ版 2.1.316。483 は `ClassifySendScreen` などで、send の画面の判定は変えていない |
| 462「Eleven gaps that fix/watch-alert-dedupe leaves」 | Trigger の `scope_change`、stall、recurrence の code(`watch.go`、`watch_recurrence.go`、`watch_scope.go`)と、`sendAlert` などは変えていない |
| 486「The reason a screen could not be read」 | Trigger は `usageLimitUnreadNote`、`readUsageLimitScreen`、`watchEscalationBody` の stall の分岐。どれも変えていない(`UsageLimitScreen` の中身は変わったが、Trigger は呼ぶ側を名指す) |
| 380・381 | 380 は `/director` skill の send と確かめの手順、381 は終了コードの部分。V1 の 2 文はどちらでもない |
| 206〜217・354(`org_watch.go` を挙げる行) | 挙げるのは `ensureOrgWatch`、`launchAndAwaitOrgWatch`、`newWatchdogHooks` など。この branch が変えたのは `newOrgWatchCmd` の `Long` の文字列だけ |
| 452・495・496 | 452 は S2 が #329 の注を足した行。495 と 496 はこの branch の行 |

## V4: resume の判定が待ちの文を数える場所

org runtime の仕様の改訂 (c) と、tech-debt の 495 行目(Why deferred)の「どのブロックでも数える」(`any block`)を、`ClassifyResumeScreen` が読む場所に直した。`resume_screens.go` の画面の model の 4 の (a) と (b) は「入力欄のすぐ上のブロック、候補、footer のブロック」で待ちの文を探す。候補は claude で手番の終わりの行の 1 つ上のブロック。それより上の古いブロックは数えない。

- 仕様の (c): 「どのブロックでも数える」を「入力欄のすぐ上のブロック、候補のブロック(claude では手番の終わりの行の 1 つ上)、footer のブロックで数える(`resume_screens.go` の画面の model の 4 の (a) と (b)。それより上の古いブロックは数えない)」に。続く「返事が文を引用しただけでも」は「返事の最後の段落が」にした(最後の段落は候補になるため)
- 495 行目: 「counts a waiting sentence in any block」を「in the last block above the input box, the candidate (the block before a claude status line), and the footer blocks (`ClassifyResumeScreen`; step 4 (a) and (b) of the screen model in `internal/org/resume_screens.go`)」に。「a seat that only quotes them」は「in its last paragraph」を足した

結論(入力欄の上を数えると、引用だけで stall の知らせが止まる)は変わらない。計画の D2 と Progress も同じ言い方だが、計画の本文は digest の対象なので直していない(verify の指示)。

## V5: 仕様の改訂の最後の項

2026-11-13 改訂(fix/claude-5h-usage-limit)の最後の項に、次を足した。

- 読み替える先の feat/watch-usage-limited の改訂を、(a) だけから (a)・(b)・(c) に。件の本文の言い方は (b) に、`recurred` の note は (c) にあるため
- FR-8 の行の印「2026-11-13 改訂 (a)〜(c)」が feat/watch-usage-limited の改訂を指すこと(同じ日の改訂が 3 つになり、どの (a)〜(c) かが読み分けにくかった)
- 人と leader に届く文の変更: 件の本文の説明と leader への ALERT の本文は「hit a usage limit (stopped, or waiting for it to reset)」、`recurred` の note は「hit a usage limit: 」で始まり、どれも「stopped on a usage limit」とは言わない。`SEATS` の見出しと本文の行には待つ座席も並ぶ

handoff は 3 つとも「hit a usage limit (stopped, or waiting for it to reset)」と言うとしていた。コードを読むと、`recurred` の note の頭は括弧のない「hit a usage limit: 」(`usageLimitRecurNote` の `lead`)なので、そのとおりに書いた。FR-8 の行の本文(142 行目)は書き換えていない。

## Test gaps 3〜7(新しい tech-debt の行、497 行目)

「Test gaps of fix/claude-5h-usage-limit (#329)」の 1 行にし、(a)〜(d) の 4 項目を持たせた。

| 項目 | Test gaps | 内容 |
|----|----|----|
| (a) | 3 | 計画の edge case「プロンプトが待ちの文を引用した画面」に専用の行がない。overlay の probe で 3 つの画面が上限の画面と答えないことは確かめた。うち 2 つは `ClassifyResumeScreen` が unknown と読み待ちの道に入らず、変異 a と a2 は既存の行でも死ぬので、足す価値は小さい |
| (b) | 4 | 止まった座席と待つ座席が同じ件に入る組み合わせと、待つ座席が `recurred` で足されるときの note のテストがない(待つ座席の行は実機で 169 字。同じ `usageLimitLine` と `fitUsageLimitedSeats` を通るので、おそらく同じ。未確認) |
| (c) | 5 | `statusLine` が印の文を出す設定の座席は、waiting と読まれ `usage_limited` に入る。件はすぐ記録されるので黙る向きではない。テストはない |
| (d) | 7 | `ralph org watch --help` の `usage_limited` の段落は語を変えただけで、`internal/cli` のテストは `usage_limited` の語を 1 つも持たない(`git grep --line-number -F usage_limited -- internal/cli` は `org_watch.go` の 2 行だけ) |

ほかの gap の扱いは次のとおり。

- Test gaps 1: 496 行目(「Known gap of fix/claude-5h-usage-limit (#329, self-review L5)」)が `herdr agent get` の出力が待つあいだ変わらないかを持つ
- Test gaps 2: 452 行目が `Continuing shortly` と別の幅、版、モデルの画面を持つ
- Test gaps 6: 487 行目の (c) の「`recur` が失敗したときの stderr の行」と同じ。V3 の note で、この branch が語を変えたことと、test の Test gaps 6 を書いた
- Test gaps 8: 前の tester の report が run-test.sh の load を記録しなかった、という記録の欠け。行にしない(handoff の指示)

行の Trigger は「`usageLimitWaitingFooter`、`UsageLimitScreen`、`usageLimitLine`、`fitUsageLimitedSeats`、`newOrgWatchCmd` の段落の次の変更、`usage_limit_screen_test.go` か `watch_usage_limit_test.go` に次に足すテスト」で、(a)〜(d) をそれぞれ足す先を挙げた。

## 残っていること(lead へ)

- 決めてほしい点: (1) V1 の 2 つ目の文に「解除を待っている座席の行には `Continuing automatically at …` が出て…`ralph org resume` を待たずに続く」(2 行)を足した。handoff の「待つ座席の行には…director は resume を要しない」を skill の文にしたもの。「当たった」に直すだけで足りるなら、この 2 行は消せる(4 つの写しから同じ 2 行)。(2) 357 行目に note を足したのは境界の判断(Trigger の「escalation」の字義は `watch.go` の code で、この branch は本文の語を変えた)。足さないなら、357 行目の Trigger の末尾の文と Related の 1 句を消す。(3) 262 行目と 464 行目の note は、V1 の編集が当てた分
- 触れなかった同じ言い方: `/director` skill の 19 行目と 65 行目(「利用上限で止まった座席」「org の座席が利用上限で止まったとき(`usage_limited`)」)、同じ skill の `### 件の処理` の `usage_limited` の項(「画面が利用上限の文で止まっていると」「止まった座席は `SEATS:` の行に並ぶ」)、`/org` skill の 612〜613 行目と 2246〜2247 行目。待つ座席が抜けているが、偽ではない(verify の V1、V2)。handoff が 2 文と指定したので残した
- 観察(直していない): `internal/cli/org_watch.go` は 884 行(`bbe2807d` で 883)で 800 行を超えているが、この branch が足したのは 1 行(`--help` の文字列の入れ替え)で、新しいファイルではなく、サイズの tech-debt の行もない。`send_screen_test.go`(839)と `watch_usage_limit_test.go`(1000)は 484 行目と 488 行目で扱った
- 払っていないもの: note を足した 11 行の Trigger(持ち越し)。Test gaps 3〜5 と 7 は 497 行目で、V4・V5 は文書の直しとして済んだ
- `docs/tech-debt/README.md`、`docs/specs/2026-08-01-org-runtime.md`、`/director` skill の 4 つの写しは、レーン A の #318 とレーン B の #322 も編集しうる。この run の `tech-debt` の編集は、既存の文を消さず、末尾への挿入が中心(495 行目の 1 か所だけ語の置き換え)。merge の衝突は 262、464 行目の Trigger の末尾と、末尾の行(497)で起きやすい
- `docs/plans/active/...` への参照を Related に足した。`/pr` の `archive-plan.sh` が書き換える

## Verification

- `git show HEAD:docs/tech-debt/README.md` との比較(`tdcheck.py`)で、変わった行は 262、357、457、464、477、484、485、487、488、489、492、495 の 12 行と、末尾に足した 497 行目だけ。どれも列は 5(`(?<!\\) \| ` で分けた)、バッククォートと `~~` は各列で偶数。11 行は挿入だけ(op は `insert`)、495 行目は挿入に加えて 1 か所の置き換え。足した文に `file:line` は 0。U+FFFD は 0
- 計画の参照: `docs/tech-debt/README.md` の `docs/plans/(active|archive)/...` を `verify.local.sh` の `check_tech_debt_plan_refs` と同じ式で調べて、無い参照が 0 件(126 件)。`docs/plans/active/2026-11-13-claude-5h-usage-limit.md` を指す Related は 12 行
- `./scripts/plan-visual.sh digest` は編集の前後とも `be05f6884990`。`git diff --numstat` は 4/2 行
- `./scripts/check-sync.sh` は `PASS: all files in sync.`(KNOWN_DIFF 6)、`./scripts/check-skill-sync.sh` は 14 skill で ok、`./scripts/check-pipeline-sync.sh` と `./scripts/check-template-purity.sh` は PASS。`cmp` で `/director` skill の 4 つの写しと `agent-messaging.md` の 2 つの写しは同じ。`git diff --check` は rc 0
- 行数(size check): 変えたファイルのうち 800 行を超えるのは `internal/cli/org_watch.go`(884、base 883)、`internal/org/send_screen_test.go`(839、base 831)、`internal/org/watch_usage_limit_test.go`(1000、base 936)の 3 つ。前の 2 つは新しい超過ではない
- `/natural-japanese` の lint(`uv run lint.py`)を skill の 2 つの hunk と spec の 2 つの hunk(scratch に抜き出した)に当てると、「和欧文間の半角空白」(repo は英字の前後に空白を入れる書き方)と、短い抜き出しに対する burstiness を指した。どちらも repo の書き方で、前の文と同じなので残した
- 実行していないもの: テスト全体(`run-test.sh`、`go test ./...`。handoff の指示)、`run-verify.sh`、`run-static-verify.sh`。Go のコードとテストは変えていない

## Insight event

```
./scripts/insights-append.sh --slug claude-5h-usage-limit --flow standard --phase sync_docs \
  --verdict pass --cycle auto --source skill
```

終了コード 0。`docs/insights/events/2026-11-15-claude-5h-usage-limit.jsonl` に足した行は cycle 1、pass、ts は 2026-11-15T10:24:56Z。この時点の `git status --short` で、jsonl は既にある `M` の 1 つで、迷子の `??` の jsonl は無かった(`??` は、この report だけ)。
