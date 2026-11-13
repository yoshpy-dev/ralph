# sync-docs report: send-claude-paste

- Date: 2026-11-13
- Plan: `docs/plans/active/2026-11-13-send-claude-paste.md`(承認の digest `2ad0b91591a9`)。issue #319
- Branch: `fix/send-claude-paste`。base は origin/main `e902705c`(#312a の PR #327 を含む。`3710ab21` で取り込み済み)。run 開始時の HEAD は `f40174a9`
- Pipeline run: 1 回目(`cycle-count.json` は `cycle` 1)
- 先行 report: self-review `b065f600`(LOW 6 件。L1〜L4 は `eea6424d` で直し、L5・L6 は残る)、verify `bb79ed4f`(pass。LOW 5 件 V1〜V5)、test `f40174a9`(pass。変異 7 個すべて死んだ。Test gaps 1〜4)。引いた commit は各 report を含む commit で、report の中の `HEAD:` の行の値ではない
- 見た範囲: `git diff origin/main...HEAD`(17 ファイル)。Go のコードとテストは `331eebd3`、`eea6424d`、`433dbc6f` の 3 commit で、このあとは変わっていない
- 制約: 計画の本文は変えていない。変えたのは Progress の箱 2 つ(`Verification artifact created`、`Test artifact created`)で、`./scripts/plan-visual.sh digest` は編集の前後とも `2ad0b91591a9`(`git diff --numstat` は 2/2 行)。Go のコードとテストは変えていない。テストの全体は回していない(handoff の指示)
- 差分の大きさ(`/pr` が walkthrough を決める材料。sync-docs の commit の前の数): `git diff origin/main...HEAD` は 17 ファイル、+1,375/-21(計画、report、evidence、insights を含む)。docs 以外は 9 ファイルで +468/-20、そのうちテストを除くと 6 ファイルで +136/-12(`agent-messaging.md` の 2 つの写し、`AGENTS.md`、`internal/cli/org.go`、`send_paste.go`、`verbs.go`)。walkthrough は書いていない

## Summary

verify の V1〜V5 は直した。V6(test の Test gaps 1)は、この PR の tech-debt の行に (g) を足して書いた。掃引では、Trigger に当たった行を 1 つ見つけた(`/org` skill の command table の行。この run で `send` の行を編集したため)。ほかの文書(README、AGENTS.md、quality-gates、repo-map、recipe、`/director` skill、`ralph org send --help`)は実装と合っていた。

## Changes made

| File | Change |
|------|--------|
| `.claude/skills/org/SKILL.md`、`.agents/skills/org/SKILL.md`、`templates/base/.claude/skills/org/SKILL.md`、`templates/base/.agents/skills/org/SKILL.md` | `send` の行に、claude の座席への bracketed paste、印を含む本文の断り、`--raw` が飛ばすのは protocol の検証だけ、`sent` の記録と画面の確かめは印のない本文を使うこと、を足した。4 つの写しは sha256 が同じ(`233bdcad…`)。下の「V1」 |
| `docs/specs/2026-08-01-org-runtime.md` | 「2026-11-13 改訂(fix/send-claude-paste)」の項を足した((a)〜(c)と、本文は書き換えない旨の項)。下の「V2」 |
| `docs/tech-debt/README.md` | 5 行の編集(180、226、365、484、494)。挿入だけの編集。下の「V3」「V6」と「掃引」 |
| `.claude/rules/ralph/agent-messaging.md`、`templates/base/.claude/rules/ralph/agent-messaging.md` | 「How send types a message」の Ctrl+C の段落に、消すのは人の手順であることを足した。`cmp` で同じ。下の「V4」 |
| `docs/evidence/org-send-paste-live-2026-11-13.md` | 「本文は 40 行で」を「本文は 39 行で」に。1 字の変更。下の「V5」 |
| `docs/plans/active/2026-11-13-send-claude-paste.md` | Progress の 2 つの箱を `[x]` に。`PR created` は `/pr` が付ける |
| `docs/reports/sync-docs-2026-11-13-send-claude-paste.md` | この report |
| `docs/insights/events/2026-11-13-send-claude-paste.jsonl` | `sync_docs` の行(cycle 1、pass)。ts は 2026-11-13T05:55:14Z |

commit は 2 つ。内容の commit `7d2aad03`(skill の写し 4 つ、rule の写し 2 つ、spec、tech-debt、evidence、計画)と、この report と insights の commit。

## verify が挙げた drift

### V1: `/org` skill の `send` の行

動詞リファレンスの `| \`send\` |` の行(4 つの写し、360 行目)の `--raw` の文を次のように直した。

- 「`--raw` で検証をバイパス」を「`--raw` で、この protocol の検証だけをバイパス」に
- そのあとに、driver が claude の座席には本文を `ESC [200~` と `ESC [201~` で挟んで 1 回で打つこと(素で打つと頭が欠けたため。ほかの driver の座席にはそのまま打つ)、この印を含む本文は driver によらず `--raw` と `--dry-run` も含めて終了コード 1 で断られ何も記録せず pane にも何も送らないこと、`sent` の記録と画面の確かめには印のない本文を使うこと、を足した

編集は `.claude/skills/org/SKILL.md` にだけ手でして、`./scripts/sync-skills.sh` で `.agents/skills/` を作り直し、`templates/base/.claude/skills/org/SKILL.md` は `cp`、その `.agents` 側は `CLAUDE_ROOT=templates/base/.claude/skills CODEX_ROOT=templates/base/.agents/skills ./scripts/sync-skills.sh` で作った。`internal/org/org_skill_send_sync_test.go` の `orgSkillSendRowRules` は 6 つの断片を順に探すが、足したのは `--raw` の文の直後で、どの断片も消していない(テストは回していない。字面で確かめた)。`send_defaults_sync_test.go` の `--enter-delay-ms` と `750ms` が同じ行にある確かめも、その文を触っていない。

### V2: org runtime の spec

「2026-11-13 改訂(fix/send-claude-paste)」を足した。位置は handoff の「2026-11-12 改訂の次」ではなく、`### 2026-11-13 改訂(feat/watch-usage-limited)` の次(運用ノートの前)にした。#312a の PR #327 が取り込みで入り、2026-11-12 改訂の次にはその項が既にあって、日付の順と merge の順を保つには、その次になるため。決めてほしい点なので「残っていること」にも書く。

- (a) claude の座席への bracketed paste(1117 バイトの本文の先頭 1022 バイトが落ちた、印で挟むと 1117 バイトも 5340 バイトも届いた、実機で 2000 字の DECISION が届いた、確かめた版は Claude Code 2.1.316 と herdr 0.7.5、ほかの driver にはそのまま打つ、codex の長い本文は issue #326)
- (b) 印を含む本文の断り(driver によらず `--raw` と `--dry-run` も、protocol の検証のすぐあと、何も打たず何も記録せず終了コード 1。protocol の検証に通らない本文は protocol のエラーが先。`--raw` が飛ばすのは protocol の検証だけ)
- (c) `sent` の Details、dry-run の Details、画面の道の Enter の前と後の確かめは印のない本文。claude の座席への send が打つあいだに失敗したら、次に送る前に `ralph org read` で読む
- 本文は書き換えない旨の項: FR-3 の「追加指示は `send` 動詞経由」は (a) と (b) を、FR-9 の send の記録は (c) を足して読む

(a) の数字は `docs/evidence/org-send-paste-probe-2026-11-13.md`(10〜13 行目と 140 行目)と照らした。

### V3: Trigger に当たった 3 行

3 行とも、行頭の語で探した(行番号は verify の時点から動かなかった。226、365、484)。Trigger を読み、これまでの note の書き方(`2026-11-12: fix/send-screen-fallback (#308) …`)に合わせて、Trigger セルの末尾に note、Related セルの末尾に `docs/plans/active/2026-11-13-send-claude-paste.md`(`/pr` の `archive-plan.sh` が `archive/` に書き換える。`archive/` とは書いていない)を足した。数は `wc -l` と `git show` で測り直した。

| 行(先頭の語) | 当たった所 | 書いたこと |
|----|----|----|
| 226「Small findings of org-plan-approval」(b) | `Send` とその doc(5 回目) | `Send` の doc に貼り付けと断りの段落を足し、`checkNoPasteMarkers` と `sendTextForDriver` の呼び出しを足した。`approval_test.go` の `approval gets` だけの行(945 行目)は今もあり、ファイルに触れていない。持ち越し。(c) は当たらない |
| 365「`internal/org/verbs.go` is 2277 lines」 | `Send` の変更 | `verbs.go` は 2597 から 2615(+18。+300 は 2597 から数える)、`eea6424d` で測った(最後に変えた commit)。`spawn.go` 3101、`verbs_test.go` 5893、`prompts_test.go` 2722、`spawn_test.go` 6408、`director_skill_sync_test.go` 1307 は変わらない。新しいコードは `send_paste.go`(48 行)、テストは `send_paste_test.go`(324 行、`433dbc6f` で測った)で、どちらも新しい file。`send_fallback_test.go`(835)と `pane_lock_test.go`(619)の 4 行の期待の直しは行数を変えない。持ち越し |
| 484「`send_screen_test.go` (662 → 831 lines)」 | `send_fallback_test.go` の期待の直し | `331eebd3` が 2 つのテスト(`TestOrgSendScreen_HerdrWorkingAndScreenWaiting_TypesByTheScreen`、`TestOrgSendScreen_MultiLineMessageTallerThanTheBox_EnterPressed`)の打つ文の期待を印付きにした(4 行、835 行のまま。origin/main `e902705c` と HEAD で `wc -l`)。`send_screen_test.go`(831)は触れていない。新しいテストは新しい file に入れた。持ち越し |

### V4: Ctrl+C を押すのは誰か

`agent-messaging.md` の「How send types a message」の最後の段落で、「Pressing Ctrl+C once in the pane … clears the input box」を、押すのは人であると言う形に直した。director は座席の pane にキーを送るのを `ralph org resume` と DECISION の送り方に限られる(`/director` skill の禁止事項、2022〜2026 行目)ので、入力欄に残りを見つけた director か leader は人に伝え、人が `herdr pane send-keys <pane_id> ctrl+c` を 1 回押す、と書いた。元の事実(押すのは座席が入力待ちで入力欄に何かあるときだけ、手番の途中なら止まる、空の入力欄では `Press Ctrl-C again to exit`)は残した。2 つの写しは `cmp` で同じ。

### V5: 実機の記録の行数

`docs/evidence/org-send-paste-live-2026-11-13.md` の「本文は 40 行で」を「本文は 39 行で」に。「改行は 40 個」はそのまま(全体の数)。ほかに行数を言う所はない(`grep` で確かめた)。

### V6: test の Test gaps 1、(g)

「Known gaps of fix/send-claude-paste (#319)」の行(494、末尾)の Debt、Impact、Why deferred、Trigger に (g) を足し(「(f) の次に `. (g) …`」の形)、Related に `docs/reports/test-2026-11-13-send-claude-paste.md` (Test gaps) を足した。文は handoff のとおり。

## 掃引

Trigger 列を python で切り出し(`" | "` で分けた 4 つ目のセル、取り消し線の中を外す)、この branch が変えた file、関数、help、skill、rule(`Send`、`verbs.go`、`typeAndSubmit`、`newOrgSendCmd`、`send_fallback_test`、`pane_lock_test`、`SKILL.md`、`/org` skill、`agent-messaging`、`--raw`、`--dry-run`、`send-text`、`AGENTS.md`、`org-runtime.md` ほか 38 語)で 1 回目の候補を出し、2 回目は Debt などのセルがその file を名指して Trigger が「any of」「these files」「the five」の形の行を探した。候補は約 60 行で、Trigger を 1 つずつ読んだ。

足した 1 行。

| 行(先頭の語) | 当たった所 | 書いたこと |
|----|----|----|
| 180「Two documentation findings of org-stop-all」(a) | この run で編集した `/org` skill の command table の `send` の行(4 つの写し)。Trigger は「The next edit to the `/org` skill's command table」 | `send` の行が 3,500 から 3,758 字(表の行全体を `len` で数えた)。`stop` と `disband` は 1,208 と 1,219 字で変わらない。持ち越し。note は Trigger の末尾、Related に計画 |

当たらないとした行と理由(代表)。

| 行(先頭の語) | 判断 |
|----|----|
| 137「`Send` … has no unit test for its `o.findSeat` …」 | Trigger は `Send` の seat lookup、`DefaultSendEnterDelayMS`、`fakeHerdr` の遅延注入 hook。この branch は seat lookup の前に確かめを足し、`seat.Driver` を読むが、lookup は変えていない |
| 184「Four findings on the S8 compensation of org-stop-all」C2-5 | 「`verbs.go` を次に直すときに block を移す」を字義どおりには満たす(verify の INFO)。ただし Trigger の冒頭は、`a94c914f` と `83aec44e` が同じ形で満たしながら払わなかった前例を書き、`verbs.go` を直した org-seat-resume や fix/send-screen-fallback などの PR も note を足していない。block(`CloseDeferredSelf*`、`reactivateSeat`)は触れていないので、note は足していない。足すなら 1 文で済む |
| 337・338(Codex の leader と reviewer に `ralph org send` が届かない) | Trigger は `Send` の idle の待ちの変更。待ちは変えていない |
| 457「Seven mutants of `ralph org resume`」 | Trigger の「the next test added to `pane_lock_test.go`」は、テストを足したときの話。この branch は 2 行の期待を直しただけ |
| 461・482 | 461 の「the approval checks of `Send`」は変えていない。482 の L4 は `newOrgSendCmd` の `RunE` で、この branch が変えたのは `Long` の文字列。L2・L7・L9・L10 の `send_fallback.go`、`send_screen.go`、`status_live.go` は変えていない |
| 195・201・427(`internal/cli/org.go`) | Trigger は `newOrgStartCmd` の `RunE`、`runOrgStartPlan`、`ralph org status` の JSON。`newOrgSendCmd` の `Long` の変更はどれでもない |
| 211(`newOrgSendCmd` を名指す) | Trigger は `newOrgRuntimeAt` とその doc の変更で、`newOrgSendCmd` は doc が呼び手を挙げる例として出るだけ |
| 262・380 | `/director` skill の受信箱の節、送って確かめる手順は変えていない |
| 88・148・159・179・186・188・189・221・223・238・249・355・377・401 | 名指しの関数や節(`report_test.go`、`watch.go`、`resolveOrgConfig`、`Stop`/`Disband`、`RolePromptVars` ほか)に触れていない。`/org` skill の別の節の行は、`send` の行を直した run ではなく、その節の編集を待つ |
| 473 | この branch が (c) を取り消した行(verify の AC6 が確かめ済み)。Trigger は実機の観察と Claude Code の upgrade で、今回は当たらない |

## 確かめたこと(文書が実装と合っていた所)

- `ralph org send --help`(`internal/cli/org.go:1363`)の 1 文「A text that holds a bracketed paste marker (ESC [200~ or ESC [201~) is refused with or without --raw.」は、断りの動き(`checkNoPasteMarkers`、`Raw` と `DryRun` の前)と合う。claude の座席に貼り付けで打つこと自体は help に出ない(verify の INFO。計画は求めていない)
- `AGENTS.md` の `internal/org/` の行の `send_paste.go` の一節は、`sendTextForDriver` と `checkNoPasteMarkers` の動きと合う。`README.md` は send の打ち方に触れていない
- `docs/quality/quality-gates.md` と `templates/base/docs/quality/quality-gates.md` の `--raw` は、見張りの確かめの話(`TYPE: TASK` を 1 行目に持つ `--raw`)で、変更なし。`docs/recipes/` の `herdr pane send-text`(`isolated-herdr-server.md` の probe の依頼、`director-wake.md` の見張りの 1 行)は ralph の `send` を通らない手の打ち方で、Non-goals(見張りの起こしの 1 行)のとおり変更なし
- `.claude/skills/director/SKILL.md` の禁止事項(2022〜2026 行目)は、`agent-messaging.md` の新しい文が言うとおり。director skill は変えていない
- `docs/specs/2026-08-01-org-runtime.md` の 2026-11-12 改訂 (c) の「打った文が入力欄に入ったことを pane の読み直しで確かめる」は、印のない本文で確かめる、と新しい (c) が言う形で今も正しい

## 残っていること(lead へ)

- 決めてほしい点: (1) spec の新しい項の位置。handoff は「2026-11-12 改訂の次」と言うが、そこには #312a の項が既にあるので、その次(運用ノートの前)に置いた。前に動かすなら 1 回の移動で済む。(2) 行 184 の C2-5 に note を足さなかった(上の表)。(3) self-review の L5・L6(テストの `want` の二重の意味、8 引数の呼び出し)は動きに関わらず、計画が「残す」とした。tech-debt の行にはしていない。登録するなら末尾に 1 行
- 払っていないもの: note を足した 4 行の Trigger の大半(持ち越し)。Test gaps 2〜4 と Known gaps の (a)〜(f) はすでに 494 行目にある(gap 3 は issue #326、gap 4 の `-race` は diff が goroutine もロックも変えていないので行にしていない)
- 観察(直していない): `Send` の doc comment(`verbs.go` の 284〜292 行)は、285 行目(62 字)と 288 行目(59 字)が周りの 74〜79 字より短い。コメントだけで動きは変わらず、Go のコードを触るとパイプラインの再走になるので残した。226 行目の (b) が言う、doc の整形の指摘と同じ種類
- `docs/tech-debt/README.md`、`AGENTS.md`、4 つの skill の写しは、ほかの session も編集しうる。この run の `tech-debt` の commit は 5 行への挿入だけで、既存の字は消していない。merge の衝突は末尾の行(494)と、180 行目の Trigger の末尾で起きやすい
- `docs/plans/active/...` への参照を Related に足した。`/pr` の `archive-plan.sh` が書き換える

## Verification

- `git show HEAD~1:docs/tech-debt/README.md` との比較(`tdcheck.py`)で、変わった行は 180、226、365、484、494 の 5 行だけ。操作は `insert` のみで、消えた字は 0。列は 5、バッククォートと `~~` は偶数。足した文に `file:line` は 0(`verbs.go` の行数は件数であって位置ではない)
- 計画の参照: `docs/tech-debt/README.md` の `docs/plans/(active|archive)/...` を `verify.local.sh` の `check_tech_debt_plan_refs` と同じ式で調べて、無い参照が 0 件(125 件)
- `./scripts/plan-visual.sh digest` は編集の前後とも `2ad0b91591a9`。`git diff --numstat` は 2/2 行
- `./scripts/check-sync.sh` は `PASS: all files in sync.`(KNOWN_DIFF 6)、`./scripts/check-skill-sync.sh` は 14 skill で ok、`cmp` で `agent-messaging.md` の 2 つの写しは同じ、`sha256sum` で `/org` skill の 4 つの写しは同じ、`git diff --check` は rc 0。変えた 11 ファイル(jsonl を含む)の U+FFFD は 0 件
- `/natural-japanese` の lint(`uv run lint.py`)を spec の新しい項(7 行を scratch に抜き出した)に当てると、「和欧文間の半角空白」を 1 件(4 行にまたがる)指した。この spec は英字の前後に空白を入れる書き方で、前の項も同じなので残した
- 実行していないもの: テスト全体(`run-test.sh`、`go test ./...`。handoff の指示)、`run-verify.sh`。Go のコードは変えていない

## Insight event

```
./scripts/insights-append.sh --slug send-claude-paste --flow standard --phase sync_docs \
  --verdict pass --cycle auto --source skill
```

終了コード 0。`docs/insights/events/2026-11-13-send-claude-paste.jsonl` に足した行は cycle 1、pass、ts は 2026-11-13T05:55:14Z。この時点の `git status --short` は、この jsonl の `M` だけで、迷子のファイルは無かった。
