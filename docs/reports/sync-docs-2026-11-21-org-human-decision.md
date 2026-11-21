# sync-docs report: org-human-decision

- Date: 2026-11-22(JST。UTC では 2026-11-21。パイプラインの cycle 2。`cycle-count.json` は `cycle: 2`。cycle 1 の report は、下の「Run 1」に 1 行も変えずに残す)
- Plan: `docs/plans/active/2026-11-21-org-human-decision.md`(承認の digest `fe3344aa0cac`。この run の前後とも同じ)。issue #366
- Branch: `feat/org-human-decision`。run 開始時の HEAD は `2211a00b`(origin/main の `10e9322d`、#368 の取り込み済み。merge-base も `10e9322d`)
- 先行 report(どれも先頭が 2 周目): self-review(cycle 2 の run 2 と run 3)、verify(Run 2、pass、LOW 4: V7〜V10)、test(Run 2、pass)。1 周目の sync-docs の内容のコミットは `477e9e5f`〜`a9e19ff5`、その report は `61c7594c`
- 見た範囲: 1 周目の sync-docs のあとにこの branch で変わったもの。cross-review の ACTION_REQUIRED #1 の直し(`1a3a940a`、`9290150f`)、self-review run 2 の直し(`d78f379a`)、verify の V8 の直し(`6dc7c22f`)、self-review run 3 の直し(`a9b160eb`)。Go は `internal/cli/org.go`(人の決定の send の失敗の note、`ralph org send --help` の 1 文)、`internal/org/verbs.go`(`SendResult.HumanOrigin`)とそのテスト。文書は director skill の 4 写し、`agent-messaging.md` の 2 写し、`internal/org/prompts/leader.md`、`internal/org/prompts/director.md`
- 制約: `./scripts/run-test.sh` は回していない(依頼の指示)。この run が変えた Go はテストの 1 ファイル(`internal/org/prompts_leader_human_decision_test.go`)だけで、本番のコードは変えていない。計画の本文(digest に入る節)は変えず、Progress の 1 行だけ足した。tracked のファイルは Edit の道具で直し、一時ファイルは scratchpad に書いた
- 差分の大きさ(`/pr` が walkthrough を決める材料。この run の report と plan と insights の commit の前の数): `git diff 10e9322d...HEAD` は 41 ファイル、+3,503/-103。内訳は Go のコード +392/-40、Go のテスト +1,712/-8、report・plan・insights +893/-0、そのほか(skill の 4 写し、規則、仕様、README、AGENTS.md、tech-debt)+506/-55。walkthrough を書くかどうかは `/pr` が決めるので、この report では書かない

## Run 2 (cycle 2)

### Summary

verify Run 2 の V7・V9・V10 と、R3 の implementer の気づき(director skill が引く note の文)を直した。V8(`ralph org send --help` の 1 文)は `6dc7c22f` と `a9b160eb` で直っているので、この run は触れず、仕様に書いただけである。直したのは、director skill の 4 写し(「人の決定を送る」の 2 の 4 つ目の項)、それを固定するテスト、仕様 2 つの director の決まりと CLI の note、tech-debt の 9 行(RC6 の行の訂正を含む)、計画の Progress、insight の 1 行である。

自分の掃引で、依頼の一覧にない行を 4 つ扱った(行 202、228、488、510)。verify が「当たらない」とした行 202 は、1 周目の注が同じ項目を「機能の org の節の編集」と数えていたので、同じ読みで 2 回目の注を足した(下の「register の Trigger の走査」)。

足さなかったものは、下の「残したこと」に理由つきで書いた。

### Changes made

| File | Change | Commit |
|------|--------|--------|
| `.claude/skills/director/SKILL.md` とその 3 つの写し | 「人の決定を送る」の 2 の 4 つ目の項(決定が送られた入力として出ていたとき)の note の文を、`a9b160eb` のあとの note に合わせた(下の「R3 の implementer の気づき」) | `43a50909` |
| `internal/org/prompts_leader_human_decision_test.go` | `TestDirectorSkill_HumanDecisionQuotesTheSendNotes` と定数、`TestDirectorSkill_SendsAHumanDecisionWithTheOriginHeader` の断片と doc を skill の文に合わせた | `43a50909` |
| `docs/specs/2026-08-01-org-runtime.md` | 2026-11-21 改訂の (e) に 2026-11-22 追記(終了コード 0 の send だけが記録する、失敗した send を Enter で回復しない、見えた画面ごとの手順)、新しい項 (f)(`HumanOrigin`、専用の note、help の 1 文) | `1911f881` |
| `docs/specs/2026-10-07-org-multi-org-director.md` | FR-9 の「2026-11-21 に決めたこと(issue #366)」に 2026-11-22 追記(同じ決まりと note) | `1911f881` |
| `docs/tech-debt/README.md` | 行 385(1 周目の注の括弧の訂正と cycle 2 の当たり)、行 190・267・370(cycle 2 の当たりと数)、行 202・228・488・510(掃引で見つけた当たり)の注、行 548(test の穴 7・8 を (d)(e) に) | `1911f881` |
| `docs/plans/active/2026-11-21-org-human-decision.md` | Progress に sync-docs cycle 2 の sub-bullet 1 つ。`git diff --numstat` は 1/0、digest は `fe3344aa0cac` のまま | この report の commit |
| `docs/reports/sync-docs-2026-11-21-org-human-decision.md` | この report(先頭に Run 2、1 周目は下に残す) | この report の commit |
| `docs/insights/events/2026-11-21-org-human-decision.jsonl` | `sync_docs` の行(cycle 2、pass) | この report の commit |

skill の 4 つの写しは、`.claude/skills/director/SKILL.md` と `templates/base/.claude/skills/director/SKILL.md` に同じ Edit をして、`./scripts/sync-skills.sh` で `.agents/skills` を、`CLAUDE_ROOT=templates/base/.claude/skills CODEX_ROOT=templates/base/.agents/skills ./scripts/sync-skills.sh` で `templates/base/.agents/skills` を作った。4 つを `cmp` で確かめた。

### 依頼の項目ごと

| 項目 | したこと |
|------|----------|
| V7 | 仕様 2 つに足した。org runtime の仕様は、2026-11-21 改訂の (e) に、人の決定を記録するのは終了コード 0 の send だけで(`sent` は Enter を押したあとに書かれる)、director は終了コード 1 で `note:` の行が出た send を 2026-11-13 改訂(fix/send-input-box-marker)の (c) の Enter で回復しないこと、キーを押さずに leader の pane を読んで skill の 4 つの項を上から順に見ること、headless director の role prompt も同じことを足した。(f) を新しく立て、`HumanOrigin` と、失敗した人の決定の send の専用の note(手で Enter を押さず文も打たない、規則の節と skill の項を指す、送り直さないとは言わない。Enter が押されて `sent` だけが書けなかった send は今までの note のまま)と、help の 1 文を書いた。multi-org director の仕様は FR-9 の #366 の項に同じことを足した。前の改訂 (c) は日付つきの履歴なので変えていない。事実は `internal/cli/org.go` の `orgSendHumanDecisionFailureNote` と help、`internal/org/verbs.go` の `Send` と `SendResult.HumanOrigin`、skill の 4 つの項を読んで確かめた |
| V8 | `6dc7c22f`(help の 1 文、専用の note の元)と `a9b160eb`(note を「してはいけないこと」と規則の節の名指しに絞った)で直っている。この run は触れていない。help の文は `ralph org send --help` に「Only a send that exits 0 records the decision: one submitted by hand in the pane after a failed send is not recorded.」とあり、仕様の (f) に書いた |
| V9 | 行 385(RC6)の 1 周目の注の括弧「with the same handling of a timeout, of `(text typed but not submitted)`, and of a DECISION left in the input box」を、今の手順に直した。受け継ぐのは待ちの時間切れと、終了コード 0 のあと入力欄に残った DECISION だけで、終了コード 1 で `note:` の行が出た send(`(text typed but not submitted)` を含む)は、Enter を使わない 4 つの項で扱う。そのうえで cycle 2 の当たり(`1a3a940a`、`d78f379a`、`43a50909`)を注の終わりに足した。払っていない(ほかの配達の印は名前を付けていない、`ralph org read` は変えていない) |
| V10 | 行 190(`leader.md` 1,654 → 1,660 行、`wc -l` は HEAD `43a50909`。cycle 2 は運用規律の人の決定の項に、全文が同じ決定への扱いを足した。`RolePromptVars` と `{{SCOPE}}` は変わらない)、行 370(下の表)、行 267(2 周目の編集は「答える順」の 2 と「人の決定を送る」の 2。`director_skill_sync_test.go` は変わらず、固定は `prompts_leader_human_decision_test.go`)に cycle 2 の当たりを足した。どれも払っていない |
| R3 の implementer の気づき | 下の「R3 の implementer の気づき」 |

行 370 の数(`wc -l`、1 周目の注の数は `b97a336c` のもの、今回は最後の Go の commit `43a50909`):

| ファイル | 1 周目の注 | 今回 | 備考 |
|------|------|------|------|
| `internal/org/verbs.go` | 2,678 | 2,700 | +22。`SendResult.HumanOrigin` と doc、`Send` の doc の 1 文、名前つきの戻り値、`defer` 1 つ。+300 の基準 2,597 から +103 |
| `internal/org/prompts/leader.md` | 1,654 | 1,660 | プロンプトでコードではない |
| `internal/org/send_origin_test.go` | 349 | 429 | |
| `internal/org/prompts_leader_human_decision_test.go` | 290 | 499 | `d78f379a` で 456、この run が +43(人の note の語の確かめ) |
| `internal/org/prompts_send_unsubmitted_test.go` | 113 | 123 | |
| `internal/org/spawn_test.go` | 6,448 | 6,508 | +60 は origin/main の側(#368)。基準 6,406 から +102 |
| `reseat_progress_decision_test.go`、`prompts_leader_reseat_test.go`、`verbs_test.go`、`prompts_test.go`、`spawn.go`、`director_skill_sync_test.go` | 394、782、5,893、2,727、3,122、1,318 | 同じ | 触れていない |

### R3 の implementer の気づき

director skill の「人の決定を送る」の 2 の 4 つ目の項は、「send の note は送り直さないように言う(Enter を送ったが herdr が受け取りを返さなかった send の `do not send the message again`、Enter を押した send の `Do not send the message again.`)が、人の決定では、記録のためのこの送り直しを note より優先する」と書いていた。`a9b160eb` のあと、人の決定の send が失敗したときに pane に文か Enter が届いたかもしれない 3 つの状態(`send-text` の失敗、文を打って Enter を押さなかった、Enter を herdr が受け取らなかった)は専用の note になり、その note は「送り直さない」と言わない(元の note の `do not send the message again` は、人の決定でない send にだけ残る)。Enter を押して `sent` だけが書けなかった send は、人の決定でも今までの note で、`Do not send the message again.` と `only the sent history event could not be recorded` を言う。

skill の項を次のように書き換えた(4 写し)。

- 人の決定の send が失敗したときの note は、Enter を押して台帳の書き込みだけが失敗した send を除き、人の決定のための note になる。その note は、手で Enter を押さず文も打たないことと、`agent-messaging.md` の節とこの項を指すことを言い、送り直さないとは言わない。
- Enter を押した send の note は、人の決定でない send と同じで、`Do not send the message again.` と `only the sent history event could not be recorded` を言う。人の決定では、記録のための送り直しをこの note より優先する。残りの文(2 回届く、同じ note で終わったらもう送らない)は変えていない。

テストの直し(`internal/org/prompts_leader_human_decision_test.go`):

- 定数 `directorSkillNoteEnterUnacknowledged` を消し、`directorSkillHumanNoteRuleFile`(`` `agent-messaging.md` ``)を足した。
- `TestDirectorSkill_SendsAHumanDecisionWithTheOriginHeader` の断片を、上の新しい文に置き換えた。doc に、専用の note と Enter を押した send の note の関係を書いた。
- `TestDirectorSkill_HumanDecisionQuotesTheSendNotes` は、`internal/cli/org.go` を読む形のまま、2 つを確かめる。(1) `case org.SendProgressEnterPressed:` の note に、skill が引く 2 つの語が今もあること。切り出しは `orgSendHumanDecisionFailureNote` の doc の注意のとおり、ファイルの最初の `case org.SendProgressEnterUnacknowledged:` から読む(その関数は `newOrgSendCmd` のあとにあり、同じ文字の case を持つ)。(2) `orgSendHumanDecisionFailureNote` と定数 `orgSendHumanDecisionNote` の文字列に、「do not press Enter in the pane or type the message into it by hand」、`agent-messaging.md`、「人の決定を送る」がまだあり、「do not send the message again」がないこと。skill が「送り直さないとは言わない」と書くので、記録のための送り直しが note と衝突しないことの確かめになる。文字列は空白を除いて比べるので、note の折り返しを変えても落ちない。

確かめ: `go test -count=1 -v ./internal/org/ -run 'DirectorSkill_HumanDecision|DirectorSkill_SendsAHumanDecision|DirectorSkill_Reseat|AgentMessagingRule'` が通る。`internal/cli/org.go` を scratchpad の写し(追跡ファイルの写し。テストは disk から読むので `-overlay` では届かない)で書き換えた変異が 4 つとも落ちた。M1(人の note に「Do not send the message again.」を足す)、M2(「do not press Enter ...」を消す)、M3(規則の名指しを消す)、M4(Enter を押した send の note の「Do not send the message again.」を別の文に変える)。何も変えない対照(M5)は通った。この run の前の skill の文(HEAD `2211a00b` の写し)に戻すと `TestDirectorSkill_SendsAHumanDecisionWithTheOriginHeader` が落ちる。

### register の Trigger の走査

`docs/tech-debt/README.md` の 5 列の表を ` | ` で分け、Trigger の列(4 つ目、添字 3)から取り消し線の文を除いて、cycle 2 が変えたファイルと関数の名前(`leader.md`、`director.md`、`verbs.go`、`Send`、`SendResult`、`HumanOrigin`、`org.go`、`newOrgSendCmd`、`orgSendHumanDecisionFailureNote`、`/director` skill、`agent-messaging.md`、`ralph org send` の help、変えたテストの名前)で照らし、当たった行の Trigger を読んだ(最初の掃引は Trigger でなく Why deferred の列を読んでいたので、列を直してやり直した。結果には響いていない)。

注を足した行(行の最初の語で引く。行番号は `1911f881` の値):

| 行(先頭の語) | 当たった Trigger | 注に書いたこと |
|------|------|------|
| 190(What a leader is told about its reservation) | (a) leader の prompt | `leader.md` 1,654 → 1,660。`RolePromptVars` と `{{SCOPE}}` は変わらない |
| 202(Small findings of org-feature-worktree) | (g) と、済みの (f)(j) | 掃引で見つけた。1 周目の注は運用規律の箇条を「機能の org の節の編集」と数えたので、同じ箇条を直した cycle 2 も同じ読みで 2 回目。払うものはない(固定は `assertPhrasesInOrder`) |
| 228(Small findings of org-plan-approval) | (b) `Send` の次の変更 | 9 回目。名前つきの戻り値、`defer`、`HumanOrigin` の doc。`approval_test.go` は触れていない |
| 267(Three test gaps of org-director-attach) | 受信箱の処理の節の次の変更 | 「答える順」の 2 と「人の決定を送る」の 2。(1)〜(3) は残る |
| 370(`verbs.go` is 2277 lines) | `Send` と leader の雛形のテスト | 上の表 |
| 385(RC6) | `/director` skill の送って確かめる手順 | 1 周目の注の括弧を訂正し、cycle 2 の当たりを足した |
| 488(Seven small findings of fix/send-screen-fallback) | (L4) `newOrgSendCmd` の `RunE` の次の変更 | 掃引で見つけた。`RunE` に人の決定の失敗の note を足した(1 周目は help の `Long` だけだったので当たらなかった)。`HintStateDir`、`orgInboxHintStateDir`、`stateDirFlagSet` は変わらず、L4 は残る。Related にこの計画を足した |
| 510(Known gaps of fix/send-claude-paste) | (g) `ralph org send` の help の次の変更 | `6dc7c22f` が同じ段落に 1 文を足した。`--raw --dry-run` と印のテストはない |
| 548(この branch の test の穴の行) | (この branch の行) | test report Run 2 の穴 7 を (d)、穴 8 を (e) に足した。穴 6(狭まった)は (a) が今の固定を言っていたので文は変えず、冒頭と Related に書いた |

当たったが注を足さなかった行: 470(`/director` skill と `director.md` の次の変更が Trigger だが、1 つ目のセルが取り消し線の RESOLVED で、Trigger に取り消し線がない行。1 周目と同じく、この branch の範囲ではないので触れていない)。

読んで外した行: 139(`Send` の seat lookup と `DefaultSendEnterDelayMS`。変えていない)、182(`/org` skill のコマンド表の編集。cycle 2 は表を変えていない)、213(Trigger に `newOrgSendCmd` の名はなく、(a)〜(d) は別のもの)、342・343(`Send` の idle wait ではなく、変えていない)、382・389・431・432(`leader.md` の機能の org の節、手順 14 と止める規則、見出しの追加、`### cross-review`。cycle 2 は運用規律の箇条だけ)、466(`Send` の approval の確かめ。変えていない)、477・509・513(DECISION の新しい読み手は足していない)、518・524(`sendTypedMarkerLineBox` などの送信の画面の規則と codex の貼り付け。変えていない)、521・525・529・530(進捗の節、`orgReseatHelp`、`Send` の Details の語。変えていない。`defer` は結果の欄を足すだけで Details は変わらない)、528(「agmsg の受信箱の運用」の節。変えていない)、386・544・547(別のもの)。一般語(`next touch`、`spec`、`skill` など)だけで当たった行と、取り消し済みか RESOLVED の行も、Trigger の元の文を読んで、cycle 2 の変更に当たらないことを確かめた。

### 検証

| 確かめたこと | 結果 |
|------|------|
| `gofmt -l internal/org internal/cli`、`go vet ./internal/org/` | 出力なし |
| `go test -count=1 ./internal/org/ -run 'Skill\|DirectorSkill\|HumanDecision\|DocsMatch'` | ok、1.07 s |
| `go test -count=1 ./internal/cli/ -run 'OrgSend\|Help'` | ok、77 s |
| 変えたテストと skill の組 | 変異 4 個が落ち、対照が通る。1 周目の skill の文では落ちる(上の「R3 の implementer の気づき」) |
| skill の 4 つの写し | `cmp` で同じ |
| `./scripts/check-skill-sync.sh` | 14 skill が同じ |
| `./scripts/check-sync.sh` | DRIFTED 0、ROOT_ONLY 0、TEMPLATE_ONLY 10、KNOWN_DIFF 6、PASS |
| `HARNESS_VERIFY_MODE=static ./scripts/verify.local.sh` | rc 0(tech-debt の編集のあと、2 つ目のコミットの前と、report・plan・insights の編集のあとの 2 回。check-sync、check-pipeline-sync、check-skill-sync、check-template-purity、tech-debt の計画の参照を含む) |
| skill の折り返し | 表示幅(全角 2)の最大は 77。上限の 78 以内 |
| tech-debt の編集 | HEAD と比べて、変わった行は 190、202、228、267、370、385、488、510、548 の 9 行だけ。列は 5 のまま、バッククォートと `~~` は列ごとに偶数、`file:line` の参照は足していない。追記だけで、例外は意図した 2 つ(行 385 の括弧の訂正、行 548 の冒頭の 1 文)と、Related の末尾の ` |` の前への追記 |
| `./scripts/plan-visual.sh digest docs/plans/active/2026-11-21-org-human-decision.md` | `fe3344aa0cac`(計画の `- Approved:` の行と同じ) |

回していないもの: `./scripts/run-test.sh`(依頼の指示)、`internal/org` と `internal/cli` の全テスト(上の絞った run だけ。この run が変えた Go は 1 つのテストファイルで、skill と `internal/cli/org.go` を disk から読む)。natural-japanese の lint は、skill の本文で「和欧文間の半角空白」を warn する。この skill のファイル全体が日本語と英数字のあいだに空白を置く書き方なので、そのままにした。

### 残したこと

- V8 の help の 1 文: 直っている(`6dc7c22f`)。この run は触れていない
- verify の INFO の 1 つ目(R2-L2 の案の半分): skill の 1 で日時を時刻まで書く案は採っていない。日付だけの決定を記録のために送り直すと leader が QUESTION を上げうるが、director がその QUESTION を受けることを skill は書いていない。INFO であり、skill の追加の編集になるので、足していない
- 人が席にいない間の入力欄: 人に入力欄を空けてもらう道では、人が戻るまで入力欄に残った決定が、その leader への後の send(受信箱の件への DECISION を含む)を止める。self-review run 2 の Known gaps と verify の INFO にある。triage が受け入れた選び方(押されない決定を記録しない)の裏側で、負債の行にはしていない。`/pr` の Known gaps に書く(下)
- 行 470: 上のとおり、1 周目と同じ
- 行数の観察: `internal/cli/org.go` は 2,471 行、`internal/cli/org_test.go` は 3,950 行(origin/main は 2,385 と 3,700)で、`internal/org` の 800 行の目安の行(370)は `internal/cli` を数えていない。この branch の前からの状態で、行は足していない
- insight の `self_review` の行: cycle 2 の行(17:38Z、MEDIUM 1、LOW 5)は run 2 の数で、run 3(HEAD `6dc7c22f`。MEDIUM 1、LOW 2。`a9b160eb` で直した)の数は入っていない。reviewer が足していないので、この run は足していない。調整役が足すかを決める
- `/pr` へ: PR の本文の Known gaps に、1 周目の 4 つ(実機の leader が `ORIGIN: human` の DECISION を承認や上限の答えと取り違えないことを見ていないこと、本物の herdr の pane での送信と立て直しを見ていないこと、compaction のあとに leader が人の決定を読み直す手段がないこと、ralph が書き手を確かめないこと)に加えて、(a) 動作中の leader の pane に Ctrl+C を送ったときの動きを実機で確かめていないこと(規則の記述に頼る)、(b) 上の人が席にいない間の入力欄、(c) 人の決定の note を読んだ director が実機の pane で正しく動くか(test の穴 7)を書く。cycle 2 のコード `6dc7c22f` と `a9b160eb` が verify Run 2(`d78f379a` まで)のあとにあることも書く。test Run 2(`b337156e`)は走ったが、self-review run 3 が見たのは `6dc7c22f` だけで、`a9b160eb` は見ていない。また、この run の `43a50909` がテストの 1 ファイルを test Run 2 のあとに変えている(絞った run だけ通した)。post-implementation pipeline をもう一度回すかは調整役が決める。pipeline の cycle は 2/2 で、この run は記録だけの例外を使っていない

### Insight event

`./scripts/insights-append.sh --slug org-human-decision --flow standard --phase sync_docs --verdict pass --cycle auto --source skill` で 1 行を足した(`ts` は 2026-11-21T20:45:08Z、`cycle: 2`)。`git status --short` に出たのは、既にある jsonl の `M` だけで、別の日付のファイルは作られていない。この report と同じ commit に入れる。同じファイルにある phase は、cycle 1 が self_review、verify、test、sync_docs、cross_review、cycle 2 が self_review、verify、test、(この行の)sync_docs である。

---

## Run 1 (cycle 1、履歴)

以下は 1 周目(HEAD `5c118d45`、commit `61c7594c`)の report を、先頭の見出しと Date の行から 1 行も変えずに残したもの。見出しの階層も変えていないので、下の `# ` と `## ` は 1 周目の節である。この report の中の数(行 370 の行数、`leader.md` の行数、`prompts_leader_human_decision_test.go` の行数)と行 385 の注の括弧は、上の Run 2 で更新した。

# sync-docs report: org-human-decision

- Date: 2026-11-22(JST。UTC では 2026-11-21。パイプラインの cycle 1。`cycle-count.json` は `cycle: 1`)
- Plan: `docs/plans/active/2026-11-21-org-human-decision.md`(承認の digest `fe3344aa0cac`。この run の前後とも同じ)。issue #366
- Branch: `feat/org-human-decision`。run 開始時の HEAD は `5c118d45`(origin/main との merge-base は `09d27450`)
- 先行 report: self-review(Merge: yes。M1・M2・L1・L2・L4 は `fc09a923` で直し、L3 は tech-debt の行)、verify `d172eff5`(pass、MEDIUM 2、LOW 4)、test `5c118d45`(pass、LOW の穴 6)。引いた SHA は各 report を含む commit で、report の中の `HEAD:` の行の値ではない
- 見た範囲: verify のあとのコードの commit は `b97a336c` だけ(verify の V2 の直し。`ralph org reseat --help` の 1 文とその pin)。test は `b97a336c` で走ったが、verify は見ていない。この run は Go のファイルを変えていない(`git diff --name-only 5c118d45...HEAD` に `.go` は 0 件)
- 制約: `./scripts/run-test.sh` は回していない(依頼の指示)。計画の本文(digest に入る節)は変えず、AC の印と Progress だけを直した
- 差分の大きさ(`/pr` が walkthrough を決める材料。この run の report と insights と plan の commit の前の数): `git diff 09d27450...HEAD` は 36 ファイル、+2,143/-92。内訳は Go のコード +295/-38、Go のテスト +1,159/-7、report・plan・insights +378/-0、そのほか(skill の 4 写し、規則、仕様、README、AGENTS.md、tech-debt)+311/-47。walkthrough を書くかどうかは `/pr` が決めるので、この report では書かない

## Summary

verify の文書の指摘 V1・V3〜V6 を直した。V2(`ralph org reseat --help`)は `b97a336c` で直っていたので触れていない。直したのは、`/org` skill の `send` の行と座席の立て直しの段落(4 写し)、AGENTS.md の 2 か所、仕様 3 つの日付つきの追記、tech-debt の L3 の行の訂正と carry-over の注 15 行、test report の穴の 1 行、計画の AC の印と Progress である。

足さなかったものは 1 つある。進捗の節の項目名を help と 2 つの skill の列挙と突き合わせるテスト(verify の項目 7)で、理由は下の「依頼の項目ごと」に書いた。

自分の掃引で、依頼の一覧にない行を 4 つ扱った。行 202、510、519、530 で、202 は `leader.md` の編集のたびに注を足してきた行、510 は `ralph org send` の help の編集に、519 は `orgReseatHelp` の編集に、530 は `Send` の Details の新しい語に当たった。

## Changes made

| File | Change | Commit |
|------|--------|--------|
| `.claude/skills/org/SKILL.md` とその 3 つの写し | コマンド表の `send` の行に ORIGIN の規則 2 文、「座席の立て直し」の段落の進捗の節の列挙に人の決定 | `477e9e5f` |
| `AGENTS.md` | `internal/org/` の地図に `send_origin.go` の 1 句、`reseat_progress.go` の列挙に人の決定 | `c1fc6da3` |
| `docs/specs/2026-08-01-org-runtime.md` | 2026-11-21 改訂(feat/org-human-decision)の (a)〜(e) と、本文を書き換えないことの注 | `c1fc6da3` |
| `docs/specs/2026-11-12-org-reseat.md` | FR-9 の末尾に 2026-11-21 追記(進捗の節の「人の決定」の項) | `c1fc6da3` |
| `docs/specs/2026-10-07-org-multi-org-director.md` | FR-9 に「2026-11-21 に決めたこと(issue #366)」(#318、#354 の項の次) | `c1fc6da3` |
| `docs/tech-debt/README.md` | 行 546(この branch の L3 の行)の誤りの訂正と、carry-over の注 14 行(182、190、228、267、370、382、385、432、510、519、521、525、529、530) | `8775106b` |
| `docs/tech-debt/README.md` | 行 202 の注(掃引で見つけた) | `7e4a2b09` |
| `docs/tech-debt/README.md` | test report の Test gap 1・4・5・6 を 1 行に(末尾に追加、5 列) | `a9e19ff5` |
| `docs/plans/active/2026-11-21-org-human-decision.md` | AC1〜AC7 に `[x]`、`Review`・`Verification`・`Test artifact created` に `[x]` とポインタ、sub-bullet 3 つ(verify のあとのコード、sync-docs の commit、足さなかったテスト)。`git diff --numstat` は 13/10 で、digest は `fe3344aa0cac` のまま | この report の commit |
| `docs/reports/sync-docs-2026-11-21-org-human-decision.md` | この report | この report の commit |
| `docs/insights/events/2026-11-21-org-human-decision.jsonl` | `sync_docs` の行(cycle 1、pass)。同じファイルに self_review、verify、test の行が既にあり、欠けた phase はない | この report の commit |

skill の 4 つの写しは、`.claude` を編集し、`./scripts/sync-skills.sh` で `.agents/skills` を作り、`templates/base/.claude/skills/org/SKILL.md` と `templates/base/.agents/skills/org/SKILL.md` へ `cp` した(`sync-skills.sh` は templates を作らない)。4 つを `cmp` で確かめた。

## 依頼の項目ごと

| 項目 | したこと |
|------|----------|
| V1 | `send` の行は、PLAN の承認の断りの次、「本文を入力してから 750ms 待って」の前に 2 文を足した。1 文目は、TYPE が `DECISION` で宛先が leader のときだけ `ORIGIN: human` を 1 行付けられること、ほかの ORIGIN(別の TYPE、`human` 以外の値、leader 以外の宛先、2 行目、16 KiB を超える本文)は `--raw`(1 行目が `TYPE:` のとき)と `--dry-run` も含めて終了コード 1 で断られ、何も記録せず pane にも何も送らないこと。2 文目は、送れたものは `sent` の Details が `origin=human` で始まり(`raw=true` のあと)、本物の送信は全文を `text` に残して、立て直した leader の経過の節に載せること(ralph は書き手が人かを確かめない)。立て直しの段落は、列挙の最後に「`ORIGIN: human` を付けて leader に送られた人の決定(全文と送った時刻)」を足し、折り返しを表示幅 72 以内に収めた。表の行全体は 4,280 → 4,683 文字(`stop` 1,208、`disband` 1,219、`reseat` 3,111 は変わらない)。F-8 の行が引く `stop` と `disband` のセルには触れていない |
| V2 | `b97a336c` で直っていたので、この run は触れていない。`ralph org reseat --help` の列挙には人の決定があり、`TestOrgReseatHelp_SaysWhatItDoesAndHowToSwitchAccounts` が固定している |
| V3 | `send_origin.go` を、`send_paste.go` の項と `ralph org status --live` の項のあいだに 1 句で足した(DECISION だけが ORIGIN を持てること、ほかは何も打たず記録せず断ること、`sent` の Details の `origin=human` と全文)。`reseat_progress.go` の列挙は「前の reseat」の次に人の決定(全文)を足した |
| V4 | org runtime の仕様に 2026-11-21 改訂を足した((a) ORIGIN の確かめ、(b) `sent` の印と `text`、(c) 進捗の節の項、(d) leader の role prompt、(e) director の決まり、本文は書き換えないこと、残る穴)。reseat の仕様の FR-9 の末尾に 2026-11-21 追記を、multi-org director の仕様の FR-9 に「2026-11-21 に決めたこと(issue #366)」を足した。この 3 つの事実は、`send_origin.go`、`manifest.go`、`reseat_progress.go`、`leader.md` を読んで確かめた(16 KiB は `sendOriginHumanMaxBytes`、キーは大文字小文字を区別するので `Origin:` は読まない) |
| V5 | 行 546 の「`ralph org report` does not list `sent` events」を、「`ralph org report` lists a `sent` event in its event timeline with its Details only, the marks and the first 200 bytes, not the whole text」に直した。読み手の列挙に `report.go` を足した(`report.go` は `ev.Details` をすべての event について書く。`grep EventSent` が挙げる読み手は `reseat_progress.go`、`approve_resolve.go`、`watch.go`)。org runtime の仕様の「残る穴」は、この行の最初の語で引いた |
| V6 | 下の「register の Trigger の走査」 |
| 項目 7 | 足していない。進捗の節の項目名と help と 2 つの skill の列挙を突き合わせるテストは Go のコードで、足すと verify と test をやり直すことになる。この run は `./scripts/run-test.sh` を回さない。3 つの列挙は言葉が違う(help は英語、skill は日本語で、節の項目名は日本語)ので、突き合わせの断片を決める設計も要る。行 521 の (h) の注に、この branch が (h) に当たったことと、verify が挙げた最小のテストの形を書いて残した |
| Plan | AC1〜AC7 に `[x]`。AC6(agent-messaging.md の段落)は、test report にテストがなく、verify report の Met による。`Review`・`Verification`・`Test artifact created` に `[x]` と report のポインタ。`b97a336c` の Progress は、`Verification artifact created` の sub-bullet に書いた。`./scripts/plan-visual.sh digest` は `fe3344aa0cac`(計画の `- Approved:` の行と同じ) |

## register の Trigger の走査

`docs/tech-debt/README.md` の 5 列の表を ` | ` で分け、Trigger の列から取り消し線の文を除いて、この branch が変えたファイルと関数の名前(`Send`、`verbs.go`、`send_origin`、`reseat_progress`、`leader.md`、`/director`・`/org` の skill、help、`ManifestEvent`、`AGENTS.md`、仕様の名前、変えたテストの名前)で照らした。候補 134 行の Trigger を読み、この pass 自身の編集(`/org` skill の `send` の行と立て直しの段落、AGENTS.md、仕様、tech-debt)でも照らした。verify が挙げた行番号は、README の行が増えていないので HEAD でも同じだった。

注を足した行(行の最初の語で引く。行番号は `7e4a2b09` の値):

| 行(先頭の語) | 当たった Trigger | 注に書いたこと |
|------|------|------|
| 182(Two documentation findings of org-stop-all) | (a) コマンド表の次の編集 | `send` の行の表の行全体が 4,280 → 4,683 文字。`stop` と `disband` は変わらず、表の作り直しは計画の範囲外 |
| 190(What a leader is told about its reservation) | (a) leader の prompt | `leader.md` 1,626 → 1,654 行。`RolePromptVars` と `{{SCOPE}}` は変わらない |
| 202(Small findings of org-feature-worktree) | (g) と、済みの (f)(j) | 掃引で見つけた。前の枝と同じ形で「払うものはない」。pin は `assertPhrasesInOrder`(空白を畳む) |
| 228(Small findings of org-plan-approval) | (b) `Send` の次の変更 | 8 回目。`approval_test.go` は触れていない |
| 267(Three test gaps of org-director-attach) | 受信箱の処理の節の次の変更 | `### 人の決定を送る` を足した。(1)〜(3) は残る |
| 370(`verbs.go` is 2277 lines) | `Send` と leader の雛形のテスト | `verbs.go` 2655 → 2678、`leader.md` 1626 → 1654、`prompts_leader_reseat_test.go` 777 → 782。新しいテストは 3 つの新しいファイル(349、394、290 行)。`spawn.go` と `spawn_test.go` は origin/main の値(3122、6448)で、前の注の 3109、6408 から動いたのは main の側 |
| 382(RC4) | 機能の org の手順 | 手順 4 に `ORIGIN: human` の文。RC4 は範囲外 |
| 385(RC6) | `/director` skill の送って確かめる手順 | `### 人の決定を送る` の 2 と、「座席を立て直す」の最後の段落 |
| 432(Seven test gaps of org-cross-review) | (1)(4)(7) `### cross-review` の節 | 上限の答えの読み手に `ORIGIN: human` の文。`TestRenderRolePrompt_Leader_HumanDecisionIsNotTheAnswerItWaitsFor` が固定 |
| 510(Known gaps of fix/send-claude-paste) | (g) `ralph org send` の help の次の変更 | 掃引で見つけた。help に ORIGIN の段落を足した。`--raw --dry-run` と印の CLI のテストはない |
| 519(Fourteen small findings of docs/spec-org-reseat) | (f)(j)(m) `orgReseatHelp` の次の変更 | 掃引で見つけた。`b97a336c` が help の列挙と pin を変えた。(f)(j)(m) は残る |
| 521(Test gaps of docs/spec-org-reseat) | (h) skill と help の reseat の文の次の編集 | director skill、`/org` skill、help の 3 つの列挙を直した。突き合わせるテストはなく、(h) は残る |
| 525(Small findings of feat/reseat-leader-progress) | `reseat_progress.go` の次の変更 | 602 → 703 行。(a) `reseatProgressSection` 107 → 123 行。(c) `reseatProgressEscape` が呼ぶたびに replacer を作る(2 つ目の場所) |
| 529(`TestReseatProgress_TheDetailsPrefixes...`) | `Send` の Details の次の変更 | `origin=human ` は定数 `sendOriginHumanDetails` を `Send` と 4 つ目の読み手 `sentDetailsHumanOrigin` が共有する。`raw=true ` と `submit_unconfirmed=true ` は文字列のまま |
| 530(`checkApprovalDecision`) | `Send` の Details の語 | 掃引で見つけた。4 つ目の読み手を足した。次の変更はそれも覆う |

各行の Related に、進行中の計画 `docs/plans/active/2026-11-21-org-human-decision.md` を足した(`archive-plan.sh` が `/pr` で `archive/` に書き換える)。

当たったが注を足さなかった行:

- 186 の C2-5 は、Trigger が補償の関数(`CloseDeferredSelfPane` など)で、この branch は `Send` の別の場所を編集した。verify が INFO にした行
- 470(`/director` skill)は、1 つ目のセルが取り消し線の RESOLVED で、Trigger に取り消し線がない行。前の枝の report と同じく、この branch の範囲ではないので触れていない(足すなら Trigger に `Done in` を付けて閉じる)

読んで外した行: 139、185 の (c)、193、342、343、466(ORIGIN の確かめは貼り付けの印の確かめの直後で、座席の探し方、idle の待ち、承認の確かめより前。そこは変えていない)。367(`manifest.go` の roster ではなく `ManifestEvent` の `Text`)。389、431(`leader.md` に節は足していない。足したのは文と項)。396、203、205、206、436、437、522(`StartFeature` は doc コメントを直しただけで、Trigger が名指す checks や呼び出しは変わらない)。488(`newOrgSendCmd` の `RunE` と `20250` は変わらない。変えたのは help の `Long`)。513 の (c)(`HOOK_SEAT:` の除外の隣に `ORIGIN: human` の除外を足しただけで、DECISION の新しい読み手は足していない)。512、360、188、406(`/org` skill の受信箱の手順と台帳の場所の項は編集していない)。518、524、544(agent-messaging.md の別の節)。519 の (a)〜(e)(g)(h)(k)(l)。520(`reseat.go` と `startFeatureInput` は変わらない)。527 の (a)(`startProgressSection` の呼び出しと `lastDisbandedIndexes` は変わらない。新しいテスト `TestReseatProgress_AStartAfterADisbandGivesNoDecisionOfTheOldLife` は同じ「今の命の始まり」を、決定について確かめる)。528、533(再立て直しの規則の座席の確認の項は変わらない。変えたのは規則の頭)。179、201、247、248、257、264、266、274、323、324、327、386、395、402、426、427、477(director の段階の移動の注か、ほかの動詞とファイルの Trigger)。一般語(`next touch`、`spec`、`skill` など)だけで掃引に当たった行(31、88、99、108、110、117、118、136、152、154、157、159、160、163、164、170、181、197、213、226、243、246、252、259、260、279〜282、285、291、292、308、366、387、388、393、420、442、445、447、448、457、463、467、507)と、取り消し済みか RESOLVED の行(133、251、461)も、Trigger の元の文を読んで、この branch の変更に当たらないことを確かめた。

## 検証

| 確かめたこと | 結果 |
|------|------|
| `/org` skill を読む 59 テスト(`go test -count=1 ./internal/org/ -run '^(<名前 59 個>)$'`) | ok、0.9 s(V1 の編集のあと) |
| `go test -count=1 ./internal/cli/ -run 'Agents\|Readme\|Docs\|Skill\|Director\|Reseat\|Help'` | ok、437 s(ほかのレーンが重なった) |
| skill の 4 つの写し | `cmp` で同じ |
| `./scripts/check-skill-sync.sh` | 14 skill が同じ |
| `./scripts/check-sync.sh` | ROOT_ONLY 0、TEMPLATE_ONLY 10、KNOWN_DIFF 6、PASS |
| `HARNESS_VERIFY_MODE=static ./scripts/verify.local.sh` | rc 0(最後の tech-debt の編集と plan の編集のあと。check-sync、check-pipeline-sync、check-skill-sync、check-template-purity、tech-debt の計画の参照を含む) |
| `./scripts/plan-visual.sh digest` | `fe3344aa0cac`(計画の `- Approved:` の行と同じ) |
| tech-debt の編集 | HEAD と比べて、変わった行は 546(訂正)、14 行(注)、202(注)、末尾の 1 行(追加)。列は 5 のまま、バッククォートは列ごとに偶数、`file:line` の参照は足していない。546 以外は追記だけ。新しい行の Related は進行中の計画の `active/` を指す |
| 計画の差分 | `git diff --numstat` は 13/10(AC 7 行、artifact 3 行の置き換えと、sub-bullet 3 行の追加) |

回していないもの: `./scripts/run-test.sh`(依頼の指示)。この run は Go のコードを変えていない。`b97a336c` は verify のあとで test の前なので、verify は見ておらず、test は走った。`/org` skill の `send` の行の文(終了コードや断る形)は、`ralph org send --help` の ORIGIN の段落と `checkSendOrigin` の読みと突き合わせたが、本物の ralph を動かしての確かめはしていない。

## 残したこと

- 項目 7 のテスト: 上のとおり足していない。行 521 の (h) が残る
- AC6 の `[x]`: test report にテストがなく、verify report の Met による。test report の Test gap 6 は、新しい行(`a9e19ff5`)にした
- 行 470: 上のとおり
- `/pr` へ: PR の本文の Known gaps に、実機の leader が `ORIGIN: human` の DECISION を承認や上限の答えと取り違えないことを見ていないこと、本物の herdr の pane での送信と立て直しを見ていないこと、compaction のあとに leader が人の決定を読み直す手段がないこと(tech-debt の L3 の行)、ralph が書き手を確かめないこと(計画の Non-goals)を書く。`b97a336c` が verify のあとのコードであることも書く

## Insight event

`./scripts/insights-append.sh --slug org-human-decision --flow standard --phase sync_docs --verdict pass --cycle auto --source skill` で 1 行を足した。この report と同じ commit に入れる。
