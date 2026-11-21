# Verify report: org-human-decision

- Date: 2026-11-22(JST。cycle 2 の verify、`cycle-count.json` は `cycle: 2`。cycle 1 の run 1 は、下の「Run 1」に、見出しの階層を 1 つ下げた以外は変えずに残す)
- Plan: `docs/plans/active/2026-11-21-org-human-decision.md`(issue #366)。`- Approved:` の digest `fe3344aa0cac` は、HEAD で `./scripts/plan-visual.sh digest` を打ち直しても一致する。計画は `7e8ac5b1` のあと変わっていない
- Verifier: verifier サブエージェント(`/verify`)
- Scope: `git diff 9cd63d60...HEAD`(HEAD `d78f379a`、40 ファイル、+2736 −100)。merge base は main の `9cd63d60`。重点は 1 周目の cross-review のあとの変更 `git diff 7e8ac5b1..HEAD`(14 ファイル、+502 −29)で、cross-review の ACTION_REQUIRED #1 の直し `1a3a940a`(director skill、`agent-messaging.md`、`leader.md`、テスト)と `9290150f`(headless director の role prompt)、2 周目の self-review の R2-M1 と R2-L1〜L5 の直し `d78f379a`。この範囲で変わった Go はテストの 2 ファイル(`internal/org/prompts_leader_human_decision_test.go`、`internal/org/prompts_send_unsubmitted_test.go`)だけで、本番の Go は 1 周目の verify のときと同じ。テストは走らせていない(`/test` の仕事)
- Evidence: `docs/evidence/verify-2026-11-21-org-human-decision-run2.log`(`.gitignore` の `docs/evidence/*.log` に当たるので commit されない)。`run-static-verify.sh` の出力のあとに、写しの `cmp`、計画の digest、skill が引く note の語の grep、行数を足した。wrapper 自身のログは `docs/evidence/verify-2026-11-21-175703.log`。1 周目の log は書き換えていない

## Run 2 (cycle 2)

CRITICAL、HIGH、MEDIUM はない。LOW 4(V7〜V10)。ID は 1 周目の V1〜V6 に続ける。

### Spec compliance

| Acceptance criterion | Status | Evidence |
| --- | --- | --- |
| AC1〜AC3、AC7 | Met(ソース。1 周目のとおり) | `git diff --name-only 7e8ac5b1..HEAD -- '*.go'` はテストの 2 ファイルだけで、`send_origin.go`、`verbs.go`、`manifest.go`、`reseat_progress.go` は変わっていない。1 周目の根拠がそのまま当てはまる |
| AC4 | Met | `internal/org/prompts/leader.md` の運用規律の人の決定の項に、`d78f379a` が 2 つの文を足した(全文が同じ人の決定は記録のための送り直し、日付だけのときなど迷うときは QUESTION を上げる)。承認待ちと上限の答え待ちのどちらの答えともみなさない文(`:1648-1651`)は残る。2 周目のテストの差分は追加と、固定の文 1 つの延長だけで、AC4 の固定(`TestRenderRolePrompt_Leader_HumanDecisionIsNotTheAnswerItWaitsFor`)は消えていない |
| AC5 | Met | director skill の 4 写しは `cmp` で同じ。「答える順」の 2 に、人の決定にはその Enter を送らない 1 文(`.claude/skills/director/SKILL.md:1157-1158`)。「人の決定を送る」の 2 に、記録は終了コード 0 の send だけ、`note:` の行が出た send は Enter で回復しない、見えた画面に合わせて上から 4 つの項(`:1220-1253`)。「してはいけないこと」の項(`:2532-2536`)は 1 周目のまま。`check-skill-sync.sh`(14 skill)と `check-sync.sh`(DRIFTED 0)は wrapper の中で通った |
| AC6 | Met | `.claude/rules/ralph/agent-messaging.md` の `## A human's decision` の節に、記録は終了コード 0 の send だけで、`note:` の行が出たあとの director は Enter を押さず skill の項に従う、という文が入った(`:178-189`)。「How send types a message」の Enter の段落の最後に、人の決定はその Enter で回復しない 1 文(`:162-163`)。雛形の写しと `cmp` で同じ |
| issue #366 受け入れ条件 1・2 | Met | 1 周目のとおり。2 周目の直しは、記録の残らない配達(手で押した Enter)を手順から外す向きなので、受け入れ条件 2(後から人の決定だと分かる)を弱めない |
| cross-review cycle 1 の ACTION_REQUIRED #1 | 直った(文面で) | director が人の決定に手で Enter を押す道は、skill の「答える順」の 2、「人の決定を送る」の 2、headless director の role prompt(`internal/org/prompts/director.md:44-48`)の 3 か所で閉じた。`sent` を書くのは `typeAndSubmit` が Enter を押したあとの 1 か所(`internal/org/verbs.go:700-704`)で、ほかの 3 つの進み具合(`:653-692`)は書く前に返ることをコードで確かめた。dry run の `sent` は `Text` を持たず、進捗の節は dry run を読まないので、「終了コード 0 の send だけが記録する」は必要条件として正しい |
| self-review run 2 の R2-M1、R2-L1〜L5 | 直った(R2-L2 は一部を別の形で) | 下の表 |

self-review run 2 の指摘を HEAD `d78f379a` で確かめた結果:

| ID | 状態 | 確かめたこと |
| --- | --- | --- |
| R2-M1 | 直った | 「ほかの画面(選択の画面、動作中の行、別の文)」の項が 1 つ目になり、そのときは入力欄を空けるようにも人に頼まない(`SKILL.md:1228-1231`)。`TestDirectorSkill_SendsAHumanDecisionWithTheOriginHeader` が 4 つの項をこの順で固定する |
| R2-L1 | 直った | `director.md` の項は「見えたものに合わせて skill の「人の決定を送る」の 2 の項のとおりにする(同じファイルを送り直すか、人に伝える。…)」で終わる。`agent-messaging.md` の文は「after a send that exits 1 with a `note:` line」に範囲を絞った |
| R2-L2 | 一部(別の形で) | `leader.md` に「全文が同じ」の根拠(director が本文に日時を書く)と、日付だけのときなどに QUESTION を上げる手が入った。skill の 1 に「日時は時刻まで書く」を足す案は採っていない(下の INFO) |
| R2-L3 | 直った | `docs/tech-debt/README.md` の行 548(「Test gaps of feat/org-human-decision (#366)」)の (a) は、固定された文と、固定されていない文(ヘッダ、断られる形、16 KiB、`origin=human`、書き手を確かめない文)を分けて書く。`TestAgentMessagingRule_OriginHumanIsRecordedOnlyByASendThatExits0` の中身と合う |
| R2-L4 | 直った | 2 つ目の項は、空いたと聞いたら leader の pane をもう一度読み、入力欄が空でほかの画面も出ていないときだけ送り直す |
| R2-L5 | 直った | 4 つ目の項は、記録のための送り直しを note の「送り直さない」より優先すると書き、`EnterPressed` の note のあとの送り直しがまた同じ note で終わったら送らない、と書く。skill が引く 3 つの語(`do not send the message again`、`Do not send the message again.`、`only the sent history event could not be recorded`)は `internal/cli/org.go` の note の case にある(`:1502`、`:1508-1509`)。新しいテスト `TestDirectorSkill_HumanDecisionQuotesTheSendNotes` が case ごとに固定する |

### Static analysis

| Command | Result | Notes |
| --- | --- | --- |
| `./scripts/run-static-verify.sh`(changed-language scope) | Pass(rc=0) | shellcheck と `sh -n`(hook)、`jq -e`(settings)、Codex の hook の 3 つの guard、`check-sync.sh`(IDENTICAL 172、DRIFTED 0、TEMPLATE_ONLY 10、KNOWN_DIFF 6)、`check-pipeline-sync.sh`、`check-skill-sync.sh`(14 skill)、`check-template-purity.sh`、tech-debt の計画の参照 |
| golang verifier(wrapper の中) | Pass | `gofmt: ok`、`go vet ./...`、`golangci-lint run ./...` は `0 issues.`、`staticcheck ./...` は出力なし(Skipping の行はない) |
| `scripts/secret-scan-branch.sh`(wrapper の中) | Pass | `9cd63d60..d78f379a` を origin/main に対して読み、clean |
| `./scripts/plan-visual.sh digest` | `fe3344aa0cac` | 計画の `- Approved:` と一致 |
| 写しの `cmp` | 同じ | director skill の 4 写し、`agent-messaging.md` の 2 写し、org skill の 4 写し |

### Documentation drift

| Doc / contract | In sync? | Notes |
| --- | --- | --- |
| director skill の 4 写し | Yes | 上の AC5 の行 |
| `internal/org/prompts/director.md` | Yes | skill の 4 つの項を「同じファイルを送り直すか、人に伝える」とまとめ、leader の手番の途中は人に頼まないと書く。項の中身は skill を指す |
| `internal/org/prompts/leader.md` | Yes | 送り直しの扱いは skill の 4 つ目の項の「leader は 2 回目を新しい決定として扱わない」と合う |
| `agent-messaging.md` と雛形の写し | Yes | 上の AC6 の行。言い直しは skill の 4 つの項と食い違わない |
| `/org` skill の `send` の行 | Yes | `(text typed but not submitted)` のとき「director は `/director` skill の「答える順」の 2 に従う」と言い、その 2 が人の決定の例外を持つ |
| `ralph org reseat --help`、README、AGENTS.md | Yes | 2 周目は触れていない。1 周目の V2 と V3 の直しは HEAD でも残っている |
| `ralph org send --help` と note | 一部(V8) | 下の表 |
| 仕様 2 つ | 一部(V7) | 下の表 |
| `docs/tech-debt/README.md` | 一部(V9、V10) | 下の表。行番号は HEAD のもので、行の頭の語も書く |
| 計画の Progress | 一部(INFO) | 下の INFO |

#### Findings

| ID | Severity | 場所 | 中身 | `/sync-docs` への渡し |
| --- | --- | --- | --- | --- |
| V7 | LOW | `docs/specs/2026-08-01-org-runtime.md:149`(2026-11-21 改訂の (e))、`docs/specs/2026-10-07-org-multi-org-director.md:168`(FR-9 の「2026-11-21 に決めたこと(issue #366)」) | どちらも 2 周目に決めた director の決まりを書いていない。人の決定は終了コード 0 の send でだけ記録される。`note:` の行が出た send のあと、director は Enter を押さずに pane を読み、空の入力欄に送り直すか(入力欄に残っていれば人に空けてもらう)、人に伝える。同じ仕様の前の改訂 (c)(`:113`、`(text typed but not submitted)` のときの director の Enter)には、この例外ができた | (e) と FR-9 の項に 1 文ずつ足す。前の改訂 (c) は日付の付いた履歴なので変えない |
| V8 | LOW | `internal/cli/org.go:1417-1428`(`ralph org send --help` の ORIGIN の段落)、`:1486-1490`(TextTyped の note) | help は「a real send keeps the whole text in the event」と言うが、記録が残るのは終了コード 0 の send だけで、手で押した Enter は何も記録しないことを言わない。TextTyped の note は「clear or submit it there before sending again」と言う。端末から `ORIGIN: human` を付けて送った人が note のとおり手で送ると、決定は届いても記録が残らない。rule と `/director` skill はそう書くが、CLI は書かない。triage はコードを変えない直し方を選んだ | 直すなら help に 1 文(「Only a send that exits 0 records it; an Enter pressed by hand records nothing.」)と、help のテストの固定。Go の文字列とテストを変えるので、cycle の上限 2/2 では pipeline の回し直しが要る。直さないなら `/pr` の known gap にする |
| V9 | LOW | `docs/tech-debt/README.md:385`(「RC6 (org recheck 2026-10-24)」) | 1 周目の sync-docs が足した注の「with the same handling of a timeout, of `(text typed but not submitted)`, and of a DECISION left in the input box」は、`1a3a940a` と `d78f379a` のあと誤りになった。「人の決定を送る」の 2 が「答える順」の 2 から受け継ぐのは、待ちの時間切れと、終了コード 0 のあと入力欄に残った DECISION だけで、`note:` の行が出た send(`(text typed but not submitted)` を含む)は Enter を使わない 4 つの項で扱う。2 周目は send して確かめる手順をもう一度変えたので、この行の Trigger にまた当たった | 注の括弧を今の手順に直し、「2026-11-22: cycle 2(`1a3a940a`、`d78f379a`)が人の決定の送り直しの 4 つの項を足して、また当たったが払っていない」を足す |
| V10 | LOW | `docs/tech-debt/README.md:190`(「What a leader is told about its reservation」)、`:370`(「`internal/org/verbs.go` is 2277 lines」)、`:267`(「Three test gaps of the org-director-attach PR」) | 2 周目の差分がまた Trigger に当たった行で、1 周目の注の数が古いか、2 周目の当たりを書いていない。190 の (a)(leader の prompt): 注は「1,626 to 1,654 lines … at HEAD b97a336c」。`1a3a940a` と `d78f379a` が運用規律の項を変え、`d78f379a` で 1,660 行。`RolePromptVars` と `{{SCOPE}}` は変わっていない。370(leader の雛形のテスト): 注は `prompts_leader_human_decision_test.go` を 290 行とするが、HEAD で 456 行。`prompts_send_unsubmitted_test.go` も 113 から 123 行になった(注に名前がない)。どちらも 800 行の目安の内で、`verbs.go` は 2,678 行のまま。267(`受信箱の処理`): 注は新しい小節のことだけを書くが、2 周目は「答える順」の 2 と「人の決定を送る」の 2 を変えた。`director_skill_sync_test.go` は変わっておらず、(1)〜(3) は開いたまま | 各行の注の数を HEAD のものに直し、「cycle 2 でまた当たったが払っていない」を足す。202 の (g) と 528 は当たらない(人の決定の項は `## 運用規律`(`leader.md:1545`)にあり、`## 機能ごとの org`(`:118-1343`)でも `## agmsg の受信箱の運用`(`:1491-1544`)でもない) |

INFO(直さなくてよいもの):

- R2-L2 の案の半分(skill の 1 で日時を時刻まで書く)は採っていない。`leader.md` は、日付だけのときなど同じ全文の別の決定かもしれないときに QUESTION を上げる。そのため、日付だけの決定を記録のために送り直すと(4 つ目の項)、leader が QUESTION を上げることがある。skill の「leader は 2 回目を新しい決定として扱わない」は、この場合も文としては正しい(新しい決定として進めず、聞く)が、director がその QUESTION を受けることは書いていない
- 計画の Progress に 2 周目の記録がない。cross-review の triage(`dd71d9a7`、ACTION_REQUIRED 1)、直し `1a3a940a`・`9290150f`・`d78f379a`、self-review の run 2(`572444c7`)。このコミットでは頼まれたとおり「Verification artifact created」にだけ 2 周目の 1 文を足す。残りは `/sync-docs` で足す
- 行 513(「Test gaps of docs/director-notice-handling (#318), cycle 2」)の (c) は「leader.md に DECISION の読み手が足されたとき」が Trigger で、1 周目の運用規律の項がそれに当たる。項は `HOOK_SEAT:` の DECISION と同じ扱いだと書き、テストが固定するので、(c) の求めは満たしているとも読める。記録の注はない。1 周目の当たりで、2 周目の差分は当たらない
- 入力欄を人に空けてもらう道は、人が席にいない間、入力欄に残った決定がその leader への後の send を止める。self-review run 2 の Known gaps と同じで、triage が受け入れた選び方(押されない決定を記録しない)の裏側

### Observational checks

- `Send` の `sent` を書く位置と、4 つの `SendProgress` で返る道(`internal/org/verbs.go:653-720`)、CLI の note(`internal/cli/org.go:1470-1512`)を、rule と skill の文(記録は終了コード 0 だけ、note の語)と突き合わせた
- 新しいテストの断片は `squashSpace`(空白をすべて除く)で比べるので、日本語の折り返しは固定に影響しない。テストの実行は `/test`
- `git diff 7e8ac5b1..HEAD` のテストの差分は追加と、固定の文 1 つの延長(「届いたことを確かめる」に括弧を足した)だけで、固定を消したり弱めたりしていない
- `prompts/director.md` に足した文は `{{IF_HERDR}}` と `{{IF_CMD}}` の外にあり、herdr と launch_cmd の両方の描画に入る

### Coverage gaps

- テストは走らせていない。2 周目の固定(4 つの項の順、note の語、`agent-messaging.md` の 2 写し)が通るかは `/test` が確かめる
- 実機の director と leader の pane で 4 つの項を通した人はいない。codex の `[Pasted Content <N> chars]` を Ctrl+C 1 回で消せるかは確かめられていない(`agent-messaging.md` の「How send types a message」)
- いちばん小さく足せる確かめ: `agent-messaging.md` の節の残りの文(ヘッダ、断られる形、16 KiB)を両方の写しで固定するテスト(行 548 の (a))。もう 1 つは V8 の help の 1 文と、その固定

## Verdict

- Verified: AC1〜AC7 と issue #366 の受け入れ条件 2 つ(本番のコードは 1 周目と同じで、文書は HEAD で)、cross-review の ACTION_REQUIRED #1 と self-review run 2 の 6 件の直し、静的解析(`run-static-verify.sh` rc=0)、写しの一致、計画の digest、skill が引く note の語
- Partially verified: 文書のずれ(V7〜V10)。どれも挙動は変えない。V7、V9、V10 は `/sync-docs` で直せる。V8 は Go の help の文なので、直すなら pipeline の回し直しが要る
- Not verified: テストの実行(`/test`)、実機の director と leader の振る舞い
- 判定: **pass**(cycle 2。CRITICAL、HIGH、MEDIUM はない。LOW 4: V7〜V10)
- cycle 1 の判定(pass、MEDIUM 2、LOW 4)は、下の「Run 1」の「Verdict」にそのまま残す

---

## Run 1 (cycle 1、履歴)

この下は run 1(2026-11-22、HEAD `fc09a923`)の report を、見出しの階層を 1 つ下げた以外は変えずに残したもの。

- Date: 2026-11-22(JST。UTC では 2026-11-21。cycle 1 の verify、`cycle-count.json` は `cycle: 1`)
- Plan: `docs/plans/active/2026-11-21-org-human-decision.md`(issue #366)。`- Approved:` の digest `fe3344aa0cac` は、HEAD で `./scripts/plan-visual.sh digest` を打ち直しても一致する
- Verifier: verifier サブエージェント(`/verify`)
- Scope: `git diff 09d27450...HEAD`(HEAD `fc09a923`、24 ファイル、+1903 −64)。merge-base は origin/main の `09d27450`。self-review のあとの直し `fc09a923`(M1、M2、L1、L2、L4)を、静的検査と spec の確かめに含めた。origin/main は `9cd63d60`(#367、gate.lock だけ)に進んだが、取り込まずに確かめた。テストは走らせていない(`/test` の仕事)
- Evidence: `docs/evidence/verify-2026-11-21-org-human-decision.log`(`.gitignore` の `docs/evidence/*.log` に当たるので commit されない)。`run-static-verify.sh` の出力の後ろに、HEAD から build した binary の help、計画の digest、写しの `cmp`、行数を足した。wrapper 自身のログは `docs/evidence/verify-2026-11-21-153227.log`

### Spec compliance

| Acceptance criterion | Status | Evidence |
| --- | --- | --- |
| AC1: `ORIGIN: human` の DECISION を leader へ送れ、`sent` の Details が `origin=human ` で始まる(`raw=true ` のあと)。dry run も同じ | Met(ソース) | `checkSendOrigin`(`internal/org/send_origin.go:62`)が human を返すと、`Send` が `markPrefix` に `origin=human ` を足す(`internal/org/verbs.go:504-510`)。dry run の `sent`(`:513-516`)と本物の `sent`(`:694-703`)の両方が `markPrefix` で始まる。画面の道では `raw=true origin=human via=screen … submit_unconfirmed=true ` の順(`verbs.go:694`)。テスト `TestOrgSendOrigin_HumanDecisionToLeader_MarksDetailsAndKeepsText`、`_DryRun_MarksDetailsWithoutText`、`_ScreenPath_DetailsOrder`、CLI の `TestOrgSend_OriginHumanDecision_SentToTheLeaderAndRecorded` がソースにある |
| AC2: TYPE が DECISION でない、値が `human` でない、宛先が leader でない、のどれかなら何も打たず記録せずに断り、理由を言う | Met(ソース) | 確かめは貼り付けの印の確かめの直後、watch gate・承認の確かめ・dry run の分岐・座席を探すことより前(`verbs.go:486-491`)。断りの文は見つけたこと、規則(`sendOriginRule`)、`nothing was sent` を言う(`send_origin.go:40-49`)。2 行の ORIGIN と 16 KiB 超も断る(計画の S1 のずれ (1)(2))。テスト `TestOrgSendOrigin_Refused_NothingTypedNothingRecorded`、`TestCheckSendOrigin_Table`、`_EveryTypeButDecisionRefused`、CLI の `TestOrgSend_OriginHumanToASeat_RefusedWithNothingTypedOrRecorded` |
| AC3: `ORIGIN` のない送信と記録は今のまま | Met(ソース)。既存の send のテストが通るかは `/test` | ORIGIN の行がなければ `checkSendOrigin` は `false, nil` を返し、`markPrefix` は前の `rawPrefix` と同じ値、`Text` は `""`(omitempty で JSON に出ない)。テスト `TestOrgSendOrigin_NoOriginHeader_RecordedAsBefore` |
| AC4: leader の prompt が、承認待ち(手順 4)と cross-review の上限の答え待ちのどちらでも、`ORIGIN: human` の DECISION を答えとみなさず待ち続けると書き、テストが両方の文を固定する | Met | `internal/org/prompts/leader.md:491-495`(手順 4)、`:1182-1184`(`### cross-review` の上限の答え)、`:1637-1654`(運用規律の項)、`:154-163`(立て直しの経過の「人の決定」だけは指示として読む例外)。`TestRenderRolePrompt_Leader_HumanDecisionIsNotTheAnswerItWaitsFor` が 2 か所を `HOOK_SEAT:` の文のすぐ後ろの順で固定する |
| AC5: director skill の 4 写しが件のない人の決定の送り方を書き(「してはいけないこと」と「答える順」)、skill のテストと `check-skill-sync.sh`・`check-sync.sh` が通る | Met | `.claude/skills/director/SKILL.md:1187`(`### 人の決定を送る`。「答える順」と「計画の承認」の間の独立した小節)、`:2496-2500`(してはいけないこと)、`:1719-1724`(座席を立て直すの最後の段落。fc09a923 で `ORIGIN: human` に寄せた)。4 写しは `cmp` で同じ。`check-skill-sync.sh`(14 skill)と `check-sync.sh`(DRIFTED 0)は wrapper の中で通った。skill のテストの実行は `/test` |
| AC6: agent-messaging.md に `ORIGIN: human` の段落がある | Met | `.claude/rules/ralph/agent-messaging.md:164`(`## A human's decision (`ORIGIN: human`)`)。雛形の写しと `cmp` で同じ |
| AC7: 200 バイトを超える `ORIGIN: human` の DECISION を送ったあと立て直すと(reseat と `start --plan`)、進捗の節に全文と時刻が載る。disband の前の命の決定と、`ORIGIN` のない leader 宛ての送信は載らない | Met(ソース) | `readReseatProgress` は最後の `disbanded` の次から読み、leader 宛ての `sent` のうち印と空でない `Text` の両方があるものだけを拾う(`internal/org/reseat_progress.go:212-233`)。dry run は飛ばす。`reseatProgressSection` が `人の決定` の項を全文、2 字下げ、制御文字のエスケープで載せ(`:471-486`)、この項だけでも節ができる。テスト `TestReseatProgress_AReseatGivesTheRestartedLeaderTheHumanDecisionsWhole`(reseat)、`_AStartThatRestartsTheLeaderGivesItTheHumanDecisions`(start --plan)、`_AStartAfterADisbandGivesNoDecisionOfTheOldLife`、`_AHumanDecisionBeforeTheLastDisbandIsNotListed`、`_OnlyARealHumanDecisionToTheLeaderIsListed`。決定の文は `sendDetailsMaxLen`(200)より長いことをテストが確かめる |
| issue #366 受け入れ条件 1: 人の決定を受信箱の件なしで leader に送る手順が skill に書いてある | Met | AC5 と同じ箇所 |
| issue #366 受け入れ条件 2: その送信が台帳に記録され、後から人の決定だと分かる | Met(ソース) | `sent` の Details の先頭の `origin=human ` と、本物の送信の `text`(`ManifestEvent.Text`、`internal/org/manifest.go:62-70`)。ralph が人であることを確かめないことは、計画の Non-goals と Risks が受け入れた前提で、help と agent-messaging.md もそう書く |

AC7 の文「org が disband されたあとの命の決定は載らない」は読み方が 2 つある。実装とテストの名前は「最後の disband より前の決定は載らない」と読んでおり、計画の意図はこちらだと判断した。

### Static analysis

| Command | Result | Notes |
| --- | --- | --- |
| `./scripts/run-static-verify.sh`(changed-language scope) | Pass(rc=0) | shellcheck と `sh -n`(hook)、`jq -e`(settings)、Codex の hook の 3 つの guard、`check-sync.sh`(DRIFTED 0、TEMPLATE_ONLY 10、KNOWN_DIFF 6)、`check-pipeline-sync.sh`、`check-skill-sync.sh`(14 skill)、`check-template-purity.sh`、tech-debt の計画の参照 |
| golang verifier(wrapper の中) | Pass | `gofmt -l` は空、`go vet ./...`、`golangci-lint run ./...` は `0 issues.`、`staticcheck ./...` は出力なし |
| `scripts/secret-scan-branch.sh`(wrapper の中) | Pass | `09d27450..fc09a923` を origin/main に対して読み、clean |
| `go build ./cmd/ralph`(HEAD、`$TMPDIR`)と `ralph org send --help`、`ralph org reseat --help` | 読んだ | send の help の ORIGIN の段落はコードと合う。reseat の help は V2 のとおり人の決定を挙げない |
| `./scripts/plan-visual.sh digest` | `fe3344aa0cac` | 計画の `- Approved:` と一致 |
| 追加した行の `fmt.Print`・`TODO`・`FIXME` の grep | なし | |

### Documentation drift

| Doc / contract | In sync? | Notes |
| --- | --- | --- |
| `ralph org send --help`(`internal/cli/org.go:1417-1428`) | Yes | ORIGIN を許す形、断る 4 つの形と 16 KiB 超、`--raw` と `--dry-run`、`origin=human` の印、全文、ralph が書き手を確かめないこと。build した binary の出力で確かめた |
| README(`README.md:282` と `:304`) | Yes | 282 行目の段落に人の決定の 1 文、304 行目の reseat の段落の進捗の列挙に人の決定 |
| `.claude/rules/ralph/agent-messaging.md` と雛形の写し | Yes | 新しい節が 1 つ。写しは同じ。「`TYPE: DECISION` の次の行」という書き方は director が書く形の指定で、send はヘッダの塊のどの行でも受け付ける(下の INFO) |
| director skill の 4 写し(`### 人の決定を送る`、してはいけないこと、座席を立て直す) | Yes | fc09a923 で M2 が直り、座席を立て直すの最後の段落も `ORIGIN: human` で送る形になった |
| leader の prompt(`internal/org/prompts/leader.md`) | Yes | 手順 4、`### cross-review`、運用規律、立て直しの経過の例外。`reseatProgressLead` の例外の文(fc09a923、M1)と leader.md の文は同じ言葉 |
| `/org` skill の 4 写し | No(V1) | `send` の行に ORIGIN がない。座席の立て直しの段落の進捗の列挙に人の決定がない |
| `ralph org reseat --help`(`internal/cli/org_reseat.go:108-113`) | No(V2) | 進捗の列挙に人の決定がない |
| AGENTS.md(`AGENTS.md:92`) | No(V3) | `reseat_progress.go` の列挙に人の決定がなく、`send_origin.go` が地図にない |
| 仕様 3 つ | No(V4) | 2026-11-21 の改訂の記録がない |
| `docs/tech-debt/README.md` | 一部(V5、V6) | 新しい行の事実の誤りが 1 つ。この branch が当てた Trigger に記録がない |

#### Findings

| ID | Severity | 場所 | 中身 | `/sync-docs` への渡し |
| --- | --- | --- | --- | --- |
| V1 | MEDIUM | `.claude/skills/org/SKILL.md:377`(`send` の行)、`:1930-1935`(座席の立て直しの最後の段落)。4 写しとも同じ | `send` の行は、貼り付けの印、TASK の見張り、`PLAN:` の承認の断りをそれぞれ書くが、ORIGIN の規則(DECISION を leader へ送るときの `ORIGIN: human` の 1 行だけ、ほかは断る、`origin=human` の印と全文)を書かない。座席の立て直しの段落は進捗の節の中身を「前の reseat」まで挙げて終わり、人の決定がない。計画の Verify plan は org skill の 4 写しを文書のずれの対象に挙げ、S3 の注意 (1) も同じ所を指していた | `send` の行に 1〜2 文足す(ほかの断りと同じ形で「`--raw` と `--dry-run` も含めて終了コード 1 で断られ、何も記録しない」)。立て直しの段落の列挙に「`ORIGIN: human` で leader に送った人の決定の全文」を足す。director skill の `:1712` が同じ列挙をすでに持つ。`scripts/sync-skills.sh` と雛形の 2 写し。command table の行を変えるので tech-debt の行 182 の (a) と 521 の (h) に当たる |
| V2 | MEDIUM | `internal/cli/org_reseat.go:108-113`。固定するテストは `internal/cli/org_reseat_test.go:899` | `ralph org reseat --help` は、立て直した leader が受け取るものを「when it holds any」として、承認から前の reseat までの 8 項目で挙げる。人の決定がない。決定しかない org にも節ができる(`TestReseatProgress_AHumanDecisionIsListedWhole` の doc)ので、help の列挙からは節が予測できない場合がある | 列挙の最後に「and the human decisions sent to the leader with `ORIGIN: human`, in full」を足し、`org_reseat_test.go:899` の固定の文も合わせる。前の PR(reseat-leader-progress の V11)と同じ直し方 |
| V3 | LOW | `AGENTS.md:92` | `the progress section of a restarted leader's task (reseat_progress.go: …, and the previous reseat, …)` の列挙に人の決定がない。同じ行が `send_paste.go`、`send_fallback.go` を挙げるのに、`send_origin.go`(ORIGIN の確かめと `sent` の印の読み)がない | 列挙に人の決定を足し、send の項の近くに `send_origin.go` を 1 句足す |
| V4 | LOW | `docs/specs/2026-08-01-org-runtime.md:123-126`、`docs/specs/2026-11-12-org-reseat.md:95`、`docs/specs/2026-10-07-org-multi-org-director.md:166-167` | org runtime の仕様は send と進捗の節の変更ごとに「### <日付> 改訂(<branch>)」を足してきた(#319、#326、#312c-2)。2026-11-15 改訂の (a) は進捗の節の中身を列挙する。reseat の仕様の FR-9 も中身を列挙する。director の仕様の FR-9 は director の決まりの変更を「<日付> に決めたこと(issue #N)」で残してきた(#318、#354)。この branch はどれも足していない | org runtime の仕様に「### 2026-11-21 改訂(feat/org-human-decision)」(send の ORIGIN の確かめ、`origin=human` と `text`、進捗の節の人の決定)。reseat の仕様の FR-9 に 2026-11-21 の追記。director の仕様の FR-9 に「2026-11-21 に決めたこと(issue #366)」(件のない人の決定を `ORIGIN: human` で送ってよい、人が決めたときだけ) |
| V5 | LOW | `docs/tech-debt/README.md:546`(この branch が足した L3 の行) | 「`ralph org report` does not list `sent` events」は誤り。`BuildOrgReport` は org のすべての event を Event timeline の表に書き(`internal/org/report.go:206-221`)、`sent` も Details(印と本文の先頭 200 バイト)つきで載る。載らないのは `Text`(全文)だけ。同じ文の「The `sent` events are read only by …」も `report.go` を落としている | 「`ralph org report` lists them with their Details only (the marks and the first 200 bytes), not the whole text」に直し、読み手の列挙に `report.go` を足す |
| V6 | LOW | `docs/tech-debt/README.md` の次の行(行番号は HEAD のもの) | この branch が Trigger に当てたのに、記録がない行。370(`Send` の変更と leader の雛形のテスト。`verbs.go` 2655 → 2678 行で、基準の 2597 から +81、+300 の内。新しいテスト `prompts_leader_human_decision_test.go` 290 行、`prompts_leader_reseat_test.go` 777 → 782)、525(`reseat_progress.go` 602 → 703 行。(a) `reseatProgressSection` に部分が 1 つ増えた。(c) 新しい `reseatProgressEscape` も replacer を呼ぶたびに作る)、529(`sent` の Details に `origin=human ` が加わった。新しい語は定数 `sendOriginHumanDetails` を書き手と読み手が共有するが、`raw=true ` は `verbs.go:506` と、4 つ目の読み手 `sentDetailsHumanOrigin`(`send_origin.go:107`)の両方で文字列のまま)、228 の (b)(`Send`)、190 の (a)(leader の prompt)、382 RC4(機能の org の手順 4)、432 の (1)(4)(7)(`### cross-review` の上限の答えの段落)、267(`## 受信箱の処理` に `### 人の決定を送る` が入った)、385 RC6(送って確かめる手順が 1 つ増えた。`### 人の決定を送る` の 2)、521 の (h)(「座席を立て直す」の最後の段落、1daf1591 と fc09a923) | 各行に「2026-11-21: feat/org-human-decision (#366) hit … and did not pay it」の記録を足す。202 の (g) は済みの項なので払うものはない(INFO)。186 の C2-5(`verbs.go` の編集)は前例が記録していない(INFO)。V1 の直しは 182 の (a) と 521 の (h) にもう一度当たる |

INFO(直さなくてよいもの):

- AC5 と issue の「答える順に書き足す」は、「答える順」の中ではなく、すぐ後ろの独立した小節 `### 人の決定を送る` で満たした。計画の Progress の S3 にその置き場所の記録がある。「答える順」の本文から新しい小節を指す文はないが、「してはいけないこと」の項と小節の冒頭の文が両方の使い分けを言う
- agent-messaging.md と README は ORIGIN の行を「`TYPE: DECISION` の次の行」と書く。send はヘッダの塊のどの行にあっても受け付ける(`protocol.HeaderValues`)。director が書く形の指定として読めるので、契約とのずれとはしない
- 計画の Rollout は「台帳の形は変えない」と書くが、`sent` に省略可能な `text` の key が加わった。前の版の ralph は `json.Unmarshal`(`internal/org/manifest.go:134`)で知らない key を読み飛ばすので、revert しても台帳は読める
- キーは大文字小文字を区別するので、`Origin: human` は確かめられず、印も付かずに送られる(S1 のずれ (2)、protocol の規則と同じ)。help はこの点を書かない
- 計画の受け入れ条件のチェックボックスはすべて `[ ]` のまま、Progress の「Review artifact created」も `[ ]` のまま(self-review の報告はある)。`/pr` の前に付ける
- 新しい `Details` の語 `origin=human ` は、既存の読み手 `reseatProgressSentTask`(`reseat_progress.go:307`)と `sentSubmitUnconfirmed`(`approve_resolve.go:241`)がどちらも `key=value ` の語を一般に読み飛ばすので、読み違えは起きない

### Observational checks

- HEAD `fc09a923` を `$TMPDIR` に build し、`ralph org send --help` の ORIGIN の段落と `ralph org reseat --help` の進捗の列挙を読んだ(log の末尾)
- director skill の 4 写し、agent-messaging.md の 2 写しは `cmp` で同じ
- `sent` を読む本番のコード(`grep EventSent`)は、`reseat_progress.go`、`approve_resolve.go:225`、`watch.go:1635` の 3 つ。`report.go` は event の種類を問わずすべて読む(V5)
- fc09a923 の差分(9 ファイル、+134 −29)を読んだ。`reseatProgressLead` の例外の文は括弧の中にあり、`(` より前の節の名前は変わらない。断りの文の `%q` への変更と、それに合わせたテストの期待の値は一致する

### Coverage gaps

- テストは走らせていない。AC1〜AC3 と AC7 は、ソースとテストの中身で確かめた。テストが通るかは `/test` が確かめる
- 実機の leader に `ORIGIN: human` の DECISION を送り、承認待ちのまま決定を進めて承認を待ち続けるかは、誰も見ていない。prompt の文とテストが固定する文だけ
- 本物の herdr での送信は見ていない(テストは偽の herdr)
- compaction のあとに leader が人の決定を読み直す手段はない(tech-debt の L3 の行、受け入れ条件の外)
- 座席の pane の中からも `ORIGIN: human` を付けて leader に送れる。ralph は書き手を確かめない(計画が受け入れた前提)
- いちばん小さく足せる確かめ: 進捗の節の項目の名前(`reseatProgressHumanDecisionsLabel` と既存の 8 項目)を、`ralph org reseat --help`、`/org` skill、`/director` skill の列挙と突き合わせるテスト。tech-debt の行 521 の (h) が求めているもので、列挙が遅れたのはこの branch で 5 回目になる。もう 1 つは、`/org` skill の `send` の行が `ORIGIN: human` を含むことを固定する 1 項目

### Verdict

- Verified: AC1〜AC7 と issue #366 の受け入れ条件 2 つ(ソースとテストの中身で)、静的解析(`run-static-verify.sh` rc=0)、写しの一致、send の help、計画の digest
- Partially verified: 文書のずれ(V1〜V6)。どれも挙動は変えず、`/sync-docs` で直せる。V1 と V2 は計画の Verify plan が挙げた面か利用者に見える help なので MEDIUM にした
- Not verified: テストの実行(`/test`)、実機の leader の振る舞い、本物の herdr での送信
- 判定: **pass**(CRITICAL と HIGH はない。MEDIUM 2、LOW 4)
