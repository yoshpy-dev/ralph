# sync-docs report: send-codex-paste

- Date: 2026-11-15
- Plan: `docs/plans/active/2026-11-15-send-codex-paste.md`(承認の digest `f17371b28b9f`)。issue #326
- Branch: `fix/send-codex-paste`。base は origin/main `df186374`(#318 の PR #334 を含む。`dd5e28a6` で取り込み済み。`git merge-base --is-ancestor origin/main HEAD` は真)。run 開始時の HEAD は `dd5e28a6`
- Pipeline run: 1 回目(`cycle-count.json` は `cycle` 1)
- 先行 report: self-review `69179e35`(LOW 7 件。L1、L2、L4、L5、L7 は `7ecd4498` で直し、L3、L6 は残った)、verify `5c274b0a`(partial-pass。V1〜V8)、test `3598f296`(pass。変異 14 個のうち 12 個を殺した。G1、G2)。引いた commit は各 report を含む commit で、report の中の `HEAD:` の行の値ではない
- 見た範囲: `git diff origin/main...HEAD`。Go のコードとテストは `ad29a286` と `7ecd4498` の 2 commit で、このあとは変わっていない
- 制約: テストの全体は回していない(handoff の指示)。回したのは `go test ./internal/org/ -run 'OrgSkill|SkillSync' -count=1` だけ。Go のファイルは変えていない。計画の本文は変えず、Progress の 3 つの箱と sub-bullet だけを変えた(`./scripts/plan-visual.sh digest` は編集の前後とも `f17371b28b9f`、`git diff --numstat` は 11/3 行で、消えた 3 行は `- [ ]` の行)
- 差分の大きさ(`/pr` が walkthrough を決める材料。sync-docs の commit の前の数): `git diff origin/main...dd5e28a6` は 30 ファイル、+1,357/-96(計画、report、evidence、insights を含む)。Go は 11 ファイルで +573/-71、そのうちテストを除くと 4 ファイル(`send_paste.go`、`send_screen.go`、`send_fallback.go`、`verbs.go`)で +111/-47。fixture は 5 ファイルで +115。walkthrough は書いていない

## Summary

verify の V1〜V6 は直した。V7(計画の Affected areas に `pane_lock_test.go` があるが、変える物がなかった)は INFO で、計画の本文に触れないので直していない。V8(`AGENTS.md` の衝突)は、`dd5e28a6` の取り込みで解けていた(`git merge-tree` は rc 0、`AGENTS.md` の行に reseat の一節と今回の「claude or codex」が両方ある)。self-review の L3(Known gaps の行)と L6(`send_screen.go` が 788 行)は、新しい tech-debt の行に入れた。test の G1、G2 も同じ行の (g) に入れた。

掃引では、verify が挙げた 7 行(180、226、365、460、483、485、511。行番号は merge で動いた)の Trigger を読み、すべてこの branch が満たしていた。ほかに当たる行は見つからなかった。ほかの文書(README、docs/quality、repo-map、recipe、`/director` skill、リーダーとディレクターの prompt、`ralph org send --help`)に、claude だけが貼り付けで打たれると書いた所は無かった。

## Changes made

| File | Change |
|------|--------|
| `AGENTS.md` | `internal/org/` の行の bracketed paste の一節に 1 句(codex は長い貼り付けを `[Pasted Content <N> chars]` と出し、send の画面の確かめは N を文の文字数と比べる)。ほかの字は変えていない。下の「V1」 |
| `docs/tech-debt/README.md` | 新しい行(518 行目、「Known gaps of fix/send-codex-paste (#326)」)、#319 の行の (e) に日付つきの注、#322 の行の (f) を RESOLVED、持ち越しの注 7 行(180、226、365、460、483、485、511)。既存の字は消していない。下の「V2」「V5」 |
| `docs/specs/2026-08-01-org-runtime.md` | 「2026-11-15 改訂(fix/send-codex-paste)」の項((a)〜(c)と、本文は書き換えない旨の項)。下の「V3」 |
| `.claude/rules/ralph/agent-messaging.md`、`templates/base/.claude/rules/ralph/agent-messaging.md` | 「How send types a message」の Ctrl+C の文。`cmp` で同じ。下の「V4」 |
| `.claude/skills/org/SKILL.md`、`.agents/skills/org/SKILL.md`、`templates/base/.claude/skills/org/SKILL.md`、`templates/base/.agents/skills/org/SKILL.md` | `send` の行の「N が本文の文字数」を「N が見出しを含む文全体の文字数」に。4 つの写しは sha256 が同じ(`715edebd…`)。下の「V6」 |
| `docs/plans/active/2026-11-15-send-codex-paste.md` | Progress の 3 つの箱(`Review artifact created`、`Verification artifact created`、`Test artifact created`)を `[x]` に、Implementation started の下に sub-bullet 2 つ。AC の箱は `bafce96c` で済み。`PR created` は `/pr` が付ける |
| `docs/reports/sync-docs-2026-11-15-send-codex-paste.md` | この report |
| `docs/insights/events/2026-11-15-send-codex-paste.jsonl` | `sync_docs` の行(cycle 1、pass)。ts は 2026-11-15T14:22:15Z |

commit は 2 つ。内容の commit `79ad7302`(上の表の `AGENTS.md` から skill の写しまで)と、この report、計画の Progress、insights の commit。

## verify が挙げた drift

### V1: `AGENTS.md`(AC6)

`internal/org/` の行の bracketed paste の一節の括弧の終わり、「`--raw` and `--dry-run` included」のあとに「; codex shows a long paste as `[Pasted Content <N> chars]`, and send's screen check compares N with the message's characters」を足した。これで AC6 の「codex の置き換えの形を書いている」が `AGENTS.md` の行でも満たされる。行のほかの字は変えていない(`git diff` は 1 行 1 か所の挿入)。`scripts/check-sync.sh` は PASS。`AGENTS.md` の `internal/org/` の行は `templates/base/AGENTS.md` には無い(root だけの行)ので、写しはない。

### V2: tech-debt の Known gaps の行、#319 の (e)、#322 の (f)

行の位置は verify の時点から動いた(#319 の行は 503 から 505、#322 の行は 507 から 512)ので、行頭の語で探した。

- 新しい行は、前の最後の行(reseat の `ApprovalDecisionCommand` の行)の後ろに 1 つの Edit で足した。5 つのセル。Debt の (a)〜(h) は handoff のとおり((a) 版、(b) 畳み始める大きさ、(c) 符号位置と書記素、(d) 途中で切れた貼り付けと前の置き換えが残る入力欄、(e) 本文の上限を超える `--raw`、(f) 画面の道を通しで走らせていない、(g) test の G1 と G2、(h) `send_screen.go` の 788 行)。Impact、Why deferred、Trigger も (a)〜(h) の順に書いた。Related は計画(`docs/plans/active/...`。`archive-plan.sh` が `/pr` で書き換える)、証拠 2 つ、self-review、verify、test の report
- (i) は handoff の一覧にない項目で、足した。verify の V4(Ctrl+C が codex の `[Pasted Content <N> chars]` を空にするかは未確認)と V2 の「8 ビットの終わりの印」(#319 の行の (f) は claude の記述で、codex では未確認)。どちらも (a)〜(h) のどれにも入らないため、1 つの項目にまとめた。不要なら (i) を消せば済む
- (g) の G1 は、`TestSendInputHolds_CodexPlaceholderRefuses` に claude の形(`[Pasted text #1 +2016 lines]`)で数が合う行がないこと(M11)。G2 は `TestOrgResume_StoppedCodexSeat_TypesTheSentence` が打った文を見ないこと(M13)。test report の言い方に合わせ、生き残った変異の名前も書いた
- #319 の行の (e): Debt の (e) の文のあとに「(2026-11-15, fix/send-codex-paste (#326): a codex seat gets the paste too since this change, so the refusal now protects both pastes.)」。古い文は書き換えていない。Impact と Trigger の (e) は今も成り立つので触れていない
- #322 の行の (f): 4 つのセルすべてで `~~(f) …~~` にした。Debt のセルは行 474(#308 の (c)。#319 が閉じた)と同じ形で `(RESOLVED 2026-11-15 in fix/send-codex-paste (#326, commit ad29a286): …)` を続けた。Why deferred の (f) も取り消し線にした

### V3: spec の改訂の項

`### 2026-11-15 改訂(fix/send-codex-paste)` を、最後の改訂の項(`### 2026-11-13 改訂(fix/send-input-box-marker)`)の次、`### 運用ノート` の前に足した。(a) codex の座席にも貼り付けで打つこと(5340 バイトの素の文が 750 ms 後の Enter で送られなかった X1、5 つの大きさがどれも送られバイト単位で同じに届いたこと、実機の 2000 字の DECISION。実機は herdr の道を通ったと書いた)、(b) 画面の道の 4 つ目の規則と Enter のあとの確かめ(N は見出しの行を含む文全体の符号位置の数で、p4 は本文 2000 字で N が 2016。UTF-16 の 2017、バイトの 4925 ではない。通さない形。書記素の違いは未確認)、(c) 変えていないもの(印の断りは driver によらず、`sent` と dry-run の Details と画面の確かめは印のない本文、`ResumeText` は素の文、Enter の前の待ち、本文の上限を超える `--raw`、送信が失敗したあとの手順は codex にも、codex の Ctrl+C は未確認)。

本文は書き換えない旨の項に、置き換えて読む 3 か所を書いた: fix/send-claude-paste の改訂 (a) の「codex の座席の長い本文が … この改訂の外である」(codex についてはこの改訂の (a)。空とほかの値は今のまま)、同じ改訂の (c) の「claude の座席への send が … 失敗したとき」(codex にも)、fix/send-input-box-marker の改訂 (a) の「今の 3 つの規則」(4 つ目を足して 4 つ)。FR-3 と FR-9 は、前の改訂の注と同じ形で足して読むと書いた。FR の行には印を付けていない(前の send の 2 つの改訂も付けていない)。

### V4: agent-messaging.md の Ctrl+C

「The human presses Ctrl+C once in the pane (…), which clears the input box.」の `, which clears the input box` を外し、「One Ctrl+C cleared a box that held text in Claude Code 2.1.316 and in codex-cli 0.166.0 (the ralph repository's probe of 2026-11-13); clearing a codex `[Pasted Content <N> chars]` placeholder that way was not checked.」に直した。claude だけの `Press Ctrl-C again to exit` の文はそのまま。観測の根拠は `docs/evidence/org-send-marker-probe-2026-11-13.md` で、codex の Ctrl+C のあとは 4 つとも入力欄が空に戻り(109 行目)、claude も空の入力欄に戻った(148 行目)。この rule の文書は scaffold で下流の project に入り、`docs/evidence/` を引く文が 1 つもない(`grep` で確かめた)ので、evidence の経路は書かず「the ralph repository's probe」とした(「issue #319 in the ralph repository」と同じ言い方)。2 つの写しは `cmp` で同じ。

### V5: Trigger に当たった行

| 行(先頭の語) | 当たった所 | 書いたこと |
|----|----|----|
| 180「Two documentation findings of org-stop-all」(a) | `/org` skill の command table の `send` の行(この run でも編集した) | `send` の行が 4,086 から 4,280 字(表の行全体を `len` で数えた。origin/main と `3c75e2e0` は 4,086、`79ad7302` の前の HEAD は 4,273、V6 の直しが +7)。`stop` と `disband` は 1,208 と 1,219 字で変わらない。持ち越し |
| 226「Small findings of org-plan-approval」(b) | `Send` の doc の 2 段落と本体のコメント 1 つ(6 回目) | `approval_test.go` の `approval gets` だけの行は今もあり、ファイルに触れていない。(c) は当たらない。持ち越し |
| 365「`internal/org/verbs.go` is 2277 lines」 | `Send` の変更 | `verbs.go` は origin/main の 2635 から 2639(+4。`3c75e2e0` では 2615 から 2619)。`dd5e28a6` で測り、この branch が最後に変えた commit は `7ecd4498`。`verbs_test.go`、`prompts_test.go`、`spawn.go`、`spawn_test.go`、`director_skill_sync_test.go` は触れていない。新しいコードは `send_screen.go`(735 から 788)と `send_paste.go`(48 から 54)、テストは新しい `send_codex_paste_test.go`(389 行)。持ち越し |
| 460「The recognition of codex's footer」 | 「the next probe of codex's screens」(probe と実機で codex の画面を 5 つ fixture にした) | footer は前の evidence と同じ形(`GPT-6-Luna · low · Context 7% used · main · No cha…`、その下に `⚠ 11 · f2`)で、`% left` の形は出ていない。`resumeCodexFooterLine` は変えていない。(b) と footer の形の命名は開いたまま |
| 483「Seven small findings of fix/send-screen-fallback」(L2、L7、L9、L10) | `send_screen.go`(規則 4、`sendCodexPasteHolds`、`sendInputHasPastePlaceholder` の `switch`)と `send_fallback.go`(`sendScreenSubmitChecks` の doc) | 5 つの項目のどれにも触れていない。driver が画面の規則を持つかを聞く場所は増えていないので、L7 に 6 つ目はない。新しい規則は `resumeMarkerLineBox` を呼ばない(L11)。持ち越し |
| 485「`send_screen_test.go` (662 → 831 lines)」 | `send_screen_test.go` の fixture の表に 5 行(867 から 878)と、`send_fallback_test.go` の失敗の文言 2 行(835 のまま) | 数え方は `wc -l`。codex の期待は `send_paste_test.go` の `typedForTest`(324 から 330)と `send_fallback_marker_test.go`(2 行、53 のまま)で直した。新しいテストは新しい file。持ち越し |
| 511「Test gaps of fix/claude-5h-usage-limit」 | `usageLimitFixtureWants` に 5 行(598 から 606) | 「next test added to `usage_limit_screen_test.go`」の部分に当たる。(a)〜(d) は fixture の行ではない。持ち越し |

どの行も、Trigger のセルの末尾に note、Related のセルの末尾に `docs/plans/active/2026-11-15-send-codex-paste.md`(`/pr` の `archive-plan.sh` が書き換える。`archive/` とは書いていない)を足した。行 460 の Related には証拠のファイルも足した。#319 の行と #322 の行の Related にも、この計画を足した。

verify の挙げた 7 行の番号と今の番号は次のとおり。verify の時点の 482 は今の 483、484 は 485、506 は 511(#318 と #312c の merge で行が入った)。

### V6: `/org` skill の `send` の行

「N が本文の文字数」を「N が見出しを含む文全体の文字数」に直した(p4 は本文 2000 字で N が 2016)。ほかの字は変えていない。編集は `.claude/skills/org/SKILL.md` にだけ手でして、`./scripts/sync-skills.sh` で `.agents/skills/` を作り直し、`templates/base/.claude/skills/org/SKILL.md` は `cp`、その `.agents` 側は `CLAUDE_ROOT=templates/base/.claude/skills CODEX_ROOT=templates/base/.agents/skills ./scripts/sync-skills.sh` で作った。4 つの写しの sha256 は `715edebdc094a4d21350baa41c18849e51f2920f8101eb798e754fcb8bd69bfe` で同じ。`internal/org/org_skill_send_sync_test.go` の `orgSkillSendRowRules` の 6 つの断片は触れていない(`go test ./internal/org/ -run 'OrgSkill|SkillSync' -count=1` は `ok`)。

### self-review の L3、L6 と test の G1、G2

L3(Known gaps の行がない)は V2 の新しい行で足した。L6(`send_screen.go` が 788 行)は新しい行の (h)、G1 と G2 は (g)。L1、L2、L4、L5、L7 は `7ecd4498` で直っていて、verify が確かめている。

## 掃引

Trigger 列を python で切り出し(`(?<!\\)\|` で分けた 4 つ目のセル、全体が取り消し線の行は外す)、この branch が変えた file、関数、test、skill、rule、doc の語 45 個で 1 回目の候補を出した(77 行)。2 回目は Trigger ではなく行全体で、`send_screen.go`、`sendInputHolds`、`sendTextForDriver`、`SendScreenSubmitted`、`testdata/resume` など 10 語、`agent-messaging`、`org-runtime.md`、`Ctrl+C`、`bracketed` など 6 語を探した。`probe`、`fixture`、`placeholder` のような広い語は読む行を増やしたが、原文の Trigger を 1 つずつ読んだ。

当たらないとした行と理由。

| 行(先頭の語) | 判断 |
|----|----|
| 88「6 batchable LOW findings」 | 「corresponding file is touched」だが、項目が名指す file(`report_test.go`、`spawn_test.go`、`watch_test.go`、`watch.go`、`status.go`、`xreview-helpers.sh`)は変えていない |
| 137「`Send` … no unit test for its `o.findSeat`」 | Trigger は seat lookup と `DefaultSendEnterDelayMS`。どちらも変えていない |
| 337・338 | Trigger は `Send` の idle の待ち。待ちは変えていない |
| 357 | Debt が `usage_limit_screen_test.go` を挙げるが、Trigger は watch の escalation |
| 457・461 | 457 は `pane_lock_test.go` に足したテスト(この branch は触れていない、verify の V7)と `ResumeInputHolds` などの変更。461 は `Send` の approval check と `resume_screens.go` の `empty` の規則で、どちらも変えていない |
| 474「Known gaps of fix/send-screen-fallback」 | (a) は実機の seat の症状、(b) は Claude Code か codex の upgrade か画面の判定のエラー。どれも起きていない。Debt の (b) の「the only screens captured」は版についての文で、今も成り立つ |
| 484「Test gaps of fix/send-screen-fallback」 | 取り消し線でない Trigger は (2)(3)(4)(5)(6)(7)(8)(9) で、`checkSendScreen`、`ClassifySendScreen`、`sendTypedInputBox`、`sendCodexHasFooter` などを挙げる。どれも変えていない。`sendInputHolds` と `SendScreenSubmitted` は Debt の本文に出るだけ |
| 505(#319 の行)の (g)、512(#322 の行)の (a)〜(e)、(g) | (g) は `ralph org send` の flags か help(変えていない)。#322 の (g) は `sendTypedMarkerLineBox`、`sendTextInWant`、`TestSendInputHolds_MarkerLineRefuses`(変えていない)。(d) の Trigger は、前の文が入力欄に残って何も打たない send の報告。Ctrl+C の文(`agent-messaging.md`)は直したが、その報告は来ていない |
| 509・511 以外の 5h limit の行 | fixture を足す条件は本物の画面。この branch の fixture は codex の貼り付けで、claude の画面ではない |
| 150・151・159・186・249・355・377・401・507・508 | `/org` skill の別の節(model の表、品質のゲート、inbox の手順、split-plan の例)の編集を待つ。この branch は `send` の行の 1 語だけで、それらの節に触れていない。180 は command table の編集そのものなので当たる |
| 340・382・435・441・459・466・513〜515 | `org-runtime.md` や `agent-messaging` を Debt に挙げるが、Trigger の関数と節(`reject`、`HerdrServerStatus`、`confirmPaneTab`、`resumeOne`、`checkManifestVersion`、`PlanReseat` ほか)は変えていない |
| 55・98 | `AGENTS.md` を挙げるが、解決済みの行(取り消し線) |

#322 の行の (a) の Trigger は「A Claude Code or codex upgrade, or a send that stops …」で、この branch は codex の画面を新しく取ったが、版は同じ 0.166.0 なので、upgrade には当たらない。

## 確かめたこと(文書が実装と合っていた所)

- `ralph org send --help`(`internal/cli/org.go`)の 1 文「A text that holds a bracketed paste marker … is refused with or without --raw.」は、driver によらない断りと合う。貼り付けで打つ driver は help に出ない(plan も求めていない)
- `/org` skill の `send` の行は、「driver が claude か codex の座席には … 挟んで 1 回で打つ(claude は素で打つと本文の頭が欠け、codex は長い本文が Enter で送られなかったため)」と書き、`sendTextForDriver` と合う。画面の道の文は N の定義だけを V6 で直した
- `agent-messaging.md` の 113〜127 行目は、`sendTextForDriver`(claude か codex)、`checkNoPasteMarkers`(driver によらない)、codex の `[Pasted Content <N> chars]` と合う。送信が失敗したあとの段落は claude と codex の両方に読める
- `README.md`、`docs/architecture/repo-map.md`、`docs/quality/`、`docs/recipes/`、`.claude/skills/director/SKILL.md`、`internal/org/prompts/` は、send の貼り付けにも `Pasted` の置き換えにも触れていない(`git grep` で確かめた)。`.claude/skills/`、`internal/org/prompts`、`docs/recipes`、`README.md`、`AGENTS.md`、`templates/base/docs` に Ctrl+C で入力欄が空になるという文は無い
- `templates/base/AGENTS.md` に `internal/org/` の行は無く、`check-sync.sh` の block-aware 比較は PASS

## 残っていること(lead へ)

- 決めてほしい点: (1) 新しい tech-debt の行の (i)(Ctrl+C と 8 ビットの終わりの印)は handoff の一覧にない。verify の V2 と V4 から足した。(2) V4 の文が `docs/evidence/` の経路を引かず「the ralph repository's probe of 2026-11-13」と書いたこと(上の「V4」の理由)
- V7(INFO): 計画の Affected areas に `internal/org/pane_lock_test.go` があるが、変える物がなかった。計画の本文を直すと digest が変わるので直していない。`/pr` の本文に 1 行あれば足りる
- 払っていないもの: 持ち越しの注を足した 7 行は、どれも「Carried over to the next change that meets this trigger」。新しい行の (a)〜(i) もすべて開いている
- `docs/tech-debt/README.md`、`AGENTS.md`、4 つの skill の写しは、ほかの session も編集しうる。この run の tech-debt の変更は新しい行 1 つと 9 行への挿入(1 つの行は取り消し線の追加)で、既存の字は 1 字も消していない。merge の衝突は末尾の行(518)と、180 と 365 の Trigger の末尾で起きやすい
- 観察: 計画の Progress の `Implementation started` の下には、self-review の直し(`7ecd4498`)の sub-bullet がなかったので足した。足した 2 つの sub-bullet は、気に入らなければ計画の Progress だけを戻せば済む(digest は変わらない)

## Verification

- `git show HEAD:docs/tech-debt/README.md` との比較(`tdcheck.py`、scratchpad)で、変わった行は 180、226、365、460、483、485、505、511、512 と、足した 518 の 10 行だけ。消えた字は 0。列は 5(7 つに分かれる)、バッククォートと `~~` は列ごとに偶数。足した文に `file:line` は 0
- 計画の参照: `docs/tech-debt/README.md` の `docs/plans/(active|archive)/...` を `verify.local.sh` の `check_tech_debt_plan_refs` と同じ式で調べて、無い参照が 0 件
- `./scripts/plan-visual.sh digest docs/plans/active/2026-11-15-send-codex-paste.md` は編集の前後とも `f17371b28b9f`
- `./scripts/check-sync.sh` は `PASS: all files in sync.`(KNOWN_DIFF 6)、`./scripts/check-skill-sync.sh` は 14 skill で ok、`cmp` で `agent-messaging.md` の 2 つの写しは同じ、`shasum` で `/org` skill の 4 つの写しは同じ。変えたファイルの U+FFFD は 0 件
- `go test ./internal/org/ -run 'OrgSkill|SkillSync' -count=1`: `ok  github.com/yoshpy-dev/ralph/internal/org 0.868s`
- `/natural-japanese` の lint(`uv run lint.py`)を spec に当てると、既存の文を含めて「和欧文間の半角空白」(92 行)と、251 行の語 1 つの 2 件。251 行はこの run の外の文で、半角空白はこの spec が前の項から取る書き方なので、どちらも残した
- 実行していないもの: `./scripts/run-test.sh`、`go test ./...`(handoff の指示)、`run-verify.sh`、`secret-scan-branch.sh`(push しないので)。Go のコードは変えていない

## Insight event

```
./scripts/insights-append.sh --slug send-codex-paste --flow standard --phase sync_docs \
  --verdict pass --cycle auto --source skill
```

終了コード 0。`docs/insights/events/2026-11-15-send-codex-paste.jsonl` に足した行は cycle 1、pass、ts は 2026-11-15T14:22:15Z。この時点の `git status --short` は、この jsonl と計画の `M` だけで、迷子のファイルは無かった。
