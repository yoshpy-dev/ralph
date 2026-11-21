# Verify report: org-human-decision

- Date: 2026-11-22(JST。UTC では 2026-11-21。cycle 1 の verify、`cycle-count.json` は `cycle: 1`)
- Plan: `docs/plans/active/2026-11-21-org-human-decision.md`(issue #366)。`- Approved:` の digest `fe3344aa0cac` は、HEAD で `./scripts/plan-visual.sh digest` を打ち直しても一致する
- Verifier: verifier サブエージェント(`/verify`)
- Scope: `git diff 09d27450...HEAD`(HEAD `fc09a923`、24 ファイル、+1903 −64)。merge-base は origin/main の `09d27450`。self-review のあとの直し `fc09a923`(M1、M2、L1、L2、L4)を、静的検査と spec の確かめに含めた。origin/main は `9cd63d60`(#367、gate.lock だけ)に進んだが、取り込まずに確かめた。テストは走らせていない(`/test` の仕事)
- Evidence: `docs/evidence/verify-2026-11-21-org-human-decision.log`(`.gitignore` の `docs/evidence/*.log` に当たるので commit されない)。`run-static-verify.sh` の出力の後ろに、HEAD から build した binary の help、計画の digest、写しの `cmp`、行数を足した。wrapper 自身のログは `docs/evidence/verify-2026-11-21-153227.log`

## Spec compliance

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

## Static analysis

| Command | Result | Notes |
| --- | --- | --- |
| `./scripts/run-static-verify.sh`(changed-language scope) | Pass(rc=0) | shellcheck と `sh -n`(hook)、`jq -e`(settings)、Codex の hook の 3 つの guard、`check-sync.sh`(DRIFTED 0、TEMPLATE_ONLY 10、KNOWN_DIFF 6)、`check-pipeline-sync.sh`、`check-skill-sync.sh`(14 skill)、`check-template-purity.sh`、tech-debt の計画の参照 |
| golang verifier(wrapper の中) | Pass | `gofmt -l` は空、`go vet ./...`、`golangci-lint run ./...` は `0 issues.`、`staticcheck ./...` は出力なし |
| `scripts/secret-scan-branch.sh`(wrapper の中) | Pass | `09d27450..fc09a923` を origin/main に対して読み、clean |
| `go build ./cmd/ralph`(HEAD、`$TMPDIR`)と `ralph org send --help`、`ralph org reseat --help` | 読んだ | send の help の ORIGIN の段落はコードと合う。reseat の help は V2 のとおり人の決定を挙げない |
| `./scripts/plan-visual.sh digest` | `fe3344aa0cac` | 計画の `- Approved:` と一致 |
| 追加した行の `fmt.Print`・`TODO`・`FIXME` の grep | なし | |

## Documentation drift

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

### Findings

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

## Observational checks

- HEAD `fc09a923` を `$TMPDIR` に build し、`ralph org send --help` の ORIGIN の段落と `ralph org reseat --help` の進捗の列挙を読んだ(log の末尾)
- director skill の 4 写し、agent-messaging.md の 2 写しは `cmp` で同じ
- `sent` を読む本番のコード(`grep EventSent`)は、`reseat_progress.go`、`approve_resolve.go:225`、`watch.go:1635` の 3 つ。`report.go` は event の種類を問わずすべて読む(V5)
- fc09a923 の差分(9 ファイル、+134 −29)を読んだ。`reseatProgressLead` の例外の文は括弧の中にあり、`(` より前の節の名前は変わらない。断りの文の `%q` への変更と、それに合わせたテストの期待の値は一致する

## Coverage gaps

- テストは走らせていない。AC1〜AC3 と AC7 は、ソースとテストの中身で確かめた。テストが通るかは `/test` が確かめる
- 実機の leader に `ORIGIN: human` の DECISION を送り、承認待ちのまま決定を進めて承認を待ち続けるかは、誰も見ていない。prompt の文とテストが固定する文だけ
- 本物の herdr での送信は見ていない(テストは偽の herdr)
- compaction のあとに leader が人の決定を読み直す手段はない(tech-debt の L3 の行、受け入れ条件の外)
- 座席の pane の中からも `ORIGIN: human` を付けて leader に送れる。ralph は書き手を確かめない(計画が受け入れた前提)
- いちばん小さく足せる確かめ: 進捗の節の項目の名前(`reseatProgressHumanDecisionsLabel` と既存の 8 項目)を、`ralph org reseat --help`、`/org` skill、`/director` skill の列挙と突き合わせるテスト。tech-debt の行 521 の (h) が求めているもので、列挙が遅れたのはこの branch で 5 回目になる。もう 1 つは、`/org` skill の `send` の行が `ORIGIN: human` を含むことを固定する 1 項目

## Verdict

- Verified: AC1〜AC7 と issue #366 の受け入れ条件 2 つ(ソースとテストの中身で)、静的解析(`run-static-verify.sh` rc=0)、写しの一致、send の help、計画の digest
- Partially verified: 文書のずれ(V1〜V6)。どれも挙動は変えず、`/sync-docs` で直せる。V1 と V2 は計画の Verify plan が挙げた面か利用者に見える help なので MEDIUM にした
- Not verified: テストの実行(`/test`)、実機の leader の振る舞い、本物の herdr での送信
- 判定: **pass**(CRITICAL と HIGH はない。MEDIUM 2、LOW 4)
