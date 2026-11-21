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
