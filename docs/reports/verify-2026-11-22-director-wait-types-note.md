# Verify report: director-wait-types-note

- Date: 2026-11-22
- Plan: `docs/plans/active/2026-11-22-director-wait-types-note.md`(issue #374)
- Verifier: verifier サブエージェント(`/verify`)。パイプライン 1 回目(`cycle-count.json` は cycle 1)
- Scope: `git diff origin/main...HEAD`(merge-base は origin/main b263b1f3、確かめた HEAD は 7e3448b7)。コードは `internal/org/escalate.go`・`director_wait.go`・`internal/cli/org.go` の help、テストは新しい `director_wait_types_note_test.go`・`director_skill_wait_types_note_test.go` と既存の 2 ファイルの更新、文書は `/director` skill(4 写し)と仕様の 1 項目。self-review の直し 7e3448b7 が M1、M2、L1〜L6 を入れたかも見た。テストは走らせていない(`/test` の役目)
- Evidence: `docs/evidence/verify-2026-11-22-director-wait-types-note.log`(gitignore の対象)。静的検査の全出力、7e3448b7 から作った binary の `ralph org wait --help` と時間切れの文の probe、skill の 4 写しの md5、行数の base と HEAD。wrapper 自身の log は `docs/evidence/verify-2026-11-22-065140.log`

## Spec compliance

| Acceptance criterion | Status | Evidence |
| --- | --- | --- |
| AC1: `--types` 付きの wait(`WaitInbox` と `WaitInboxForDirector`)が ALERT だけの受信箱で時間切れになると、文が `no open inbox item arrived within <ms> ms` で始まり、数えた種類と外の件の ID・TYPE が続く。`--types` なしと外の件なしは今の文のまま。多いときは切って残りの数 | met | `WaitInbox` の時間切れ(`escalate.go` 335〜337 行)と director の `loop` の時間切れ(`director_wait.go` 541〜544 行)が、どちらも `inboxWaitTimeoutNote(types, readInboxWaitOtherTypesText(types))` を文の後ろに足す。`readInboxWaitOtherTypesText` は types がないと何も読まず `""` を返し、読めないときも `""`。`inboxWaitOtherTypesText` は 5 件(`inboxWaitOtherTypesShown`)まで名指し、残りを `, and <n> more` にし、TYPE は `PrintableInboxText` でエスケープする。director の印の `detail` は `directorWaitTimeoutDetail(types, timeout, others)` で ` (open items of other TYPEs: …)` だけを足す。テストは `TestInboxWaitOtherTypesText`、`TestInboxWaitTimeoutNote`、`TestReadInboxWaitOtherTypesText_ReadsTheOpenItemsOrNothing`、`TestWaitInbox_TypesTimeoutNamesTheOpenItemsOfOtherTypes`、`TestWaitInbox_TimeoutKeepsItsTextWithNoOpenItemOfAnotherType`、`TestWaitInboxForDirector_TypesTimeoutNamesTheOpenItemsOfOtherTypes`、`TestWaitInboxForDirector_TimeoutKeepsItsTextWithNoOpenItemOfAnotherType`、`TestDirectorWaitTimeoutDetail` の新しい行、CLI の `TestOrgWaitInbox_TypesKeepWaitingPastAnAlertItem`(完全一致に強めた)。binary の probe でも、7 件の ALERT で `(counted only QUESTION,BLOCKED,RESULT; open items of other TYPEs: e1 ALERT, …, e5 ALERT, and 2 more)`、空の受信箱では句なしを見た(Observational checks) |
| Verify plan: 時間切れの終了コードと `inboxWaitTimeoutPhrase` の位置が変わらない | met | `inboxWaitTimeoutPhrase` の定数は diff に出ない。2 つの時間切れの文は `"…%s %d ms%s"` で、句は ms の後ろにだけ付く。probe の終了コードは 3 通りとも 1。`TestDirectorSkill_NamesTheInboxWaitTimeoutPhrase` は base のまま(`director_skill_sync_test.go` は origin/main と同じ) |
| AC2: 2 つの director wait が重なり、後の wait が `director-wait.json` を取ったあとに件が入ると、どちらも件を返す | met(静的な読み) | `loop` は毎周 `openInboxItems(w.p.Types)` を読み、`w.lost` を見ない。`save` は `w.lost` のとき書かずに戻るだけ(`director_wait.go` 670〜673 行)。テストは `TestWaitInboxForDirector_TwoOverlappingWaitsBothReturnTheWatchsItems`(見張りが受け付けているときと、いないときの 2 つ)と `TestWaitInboxForDirector_TwoOverlappingTypesWaitsBothNameTheOpenAlerts`。どちらも wait ごとに別の `Org` と PID を持ち、A の stderr に `took over` が出てから見張りの形で escalate する。通るかは `/test` で見る |
| AC3: skill の「wait で待つ」が、前の wait の終わりを確かめてから張ること、乗っ取りの行が出たら古いほうを止めずに任せ、動いている wait があるあいだは次を張らないこと、外の件の扱いを言う。古い Monitor を止めると言わない。4 写しがそろう。テストで固定 | met(V1 は LOW) | `### wait で待つ` の 1132〜1162 行(1 本ずつ張る段落と、重ねたときの段落)と 1171〜1186 行(`--types` の時間切れ)。4 写しは md5 が同じで、`check-skill-sync.sh` は 14 skill が lock-step。`TestDirectorSkill_RunsOneInboxWaitAtATime` が 23 の fragment を順に照合し、`directorSkillStopsAWait` で止める言い回しがないことを見る。`TestDirectorSkill_ReadsTheUncountedItemsOfATypesTimeout` は skill の例の句を `inboxWaitTimeoutNote` の出力から作る。引用の文は `sayLost` の行(`director_wait.go` 724 行)、`wait:` の行の `(pid N) ended at`(`internal/cli/org_director_wait_line.go` 82 行)、30 秒の印(`DirectorWaitMarkInterval`)と合う |
| AC4: 仕様と `ralph org wait --help` が足した句を言う。`check-sync.sh` が通る | met | 仕様は FR-9 の末尾の 2026-11-22(issue #374)の項目(174 行)。help は binary で `A wait with --types that times out names the open items of other TYPEs it did not count (their IDs and TYPEs) after its timeout text, and a --director wait writes them in its mark's detail too.` を出した。`check-sync.sh` は DRIFTED 0 |
| self-review の M1、M2、L1〜L6 の直し(7e3448b7) | 入った | M1: 時間切れの打ち直し 3 か所(1043〜1045、1055〜1056、1101〜1102 行)に「ほかの wait がまだ走っているあいだは打たない」が付き、fragment で固定。M2: 新しいピンは `internal/org/director_skill_wait_types_note_test.go`(313 行)に移り、`director_skill_sync_test.go` は origin/main と同じ 1320 行。L1: 「新しい wait がまだ走っているとは限らない」と、終わりの知らせか `wait:` の行の `ended at` で分かる、を足した。L2: `directorSkillStopsAWait` が語尾と否定を読み、表は 38 行。L3: `context.WithCancel` と `sync.WaitGroup` を `t.Cleanup` で待つ。L4: helper の doc が `pendingLeaderAlert` を名指す。L5: `readInboxWaitOtherTypesText` に改名し、doc が wait の最後の読みと区別する。L6: doc が試していない窓を言う |

## Static analysis

| Command | Result | Notes |
| --- | --- | --- |
| `RALPH_VERIFY_SCOPE=full ./scripts/run-static-verify.sh` | pass(rc=0) | hook の shellcheck と `sh -n`、guard の awk、settings の `jq -e`、Codex の hook の guard 3 つ、`check-sync.sh`(IDENTICAL 172、DRIFTED 0)、`check-pipeline-sync.sh`、`check-skill-sync.sh`(14 skill)、`check-template-purity.sh`、tech-debt の plan の参照、golang(gofmt ok、`go vet`、golangci-lint 0 issues、staticcheck)、branch の secret scan(b263b1f3..7e3448b7、clean) |
| `go build ./cmd/ralph`(scratch へ) | pass | help と時間切れの文の probe に使った |

## Documentation drift

| Doc / contract | In sync? | Notes |
| --- | --- | --- |
| `/director` skill `### wait で待つ`(4 写し) | ほぼ | コードとの食い違いはない。打ち直しの文 3 つに条件がない(V1) |
| 仕様 `docs/specs/2026-10-07-org-multi-org-director.md` FR-9 | yes | 「上の 8b-2b の注」は 165 行にあり、174 行の項目より上。8b-2b の注の「持ち主でなくなった wait は … 印には二度と書かずに待ち続ける」とも合う。置き場所は V5 |
| `ralph org wait --help` | yes | binary で確かめた |
| `/org` skill のコマンド表の `wait` の行 | 句がない | `--types` を help に沿って詳しく書くが、時間切れの句は言わない(V3) |
| `ralph status`・`ralph org director status` | yes | 句は文の行に出ず、`--json` の `wait.detail`・`director.wait.detail` に出る。skill もそう書く |
| `README.md`、`internal/org/prompts/director.md`、`docs/recipes/director-wake.md` | yes | 時間切れの文も打ち直しの条件も書いていない |
| `docs/tech-debt/README.md` | 注がない | 行 267・386 の Trigger に当たり、行 324 の数が古くなった(V2) |
| 計画の Progress | 古い名前 | V4 |

## Observational checks

7e3448b7 から作った binary で、手で書いた ALERT 7 件の台帳と空の台帳に `ralph org wait --inbox` を打った(scratch、出力は evidence の log)。

- `--types QUESTION,BLOCKED,RESULT --timeout-ms 300`、ALERT 7 件: `org: wait --inbox: no open inbox item arrived within 300 ms (counted only QUESTION,BLOCKED,RESULT; open items of other TYPEs: e1 ALERT, e2 ALERT, e3 ALERT, e4 ALERT, e5 ALERT, and 2 more)`、終了コード 1
- 同じ `--types`、空の台帳: `org: wait --inbox: no open inbox item arrived within 300 ms`、終了コード 1
- `--types` なし、空の台帳: 同じ文、終了コード 1
- `--types ALERT`、ALERT 7 件: 件を出して返った(log の `rc=0` は `head` の終了コードなので、根拠にしていない)

director の wait(登録と動いている見張りが要る)は binary で打っていない。

## Findings

| ID | Severity | Finding | For |
| --- | --- | --- | --- |
| V1 | LOW | `### wait で待つ` に、条件のない打ち直しの文が 3 つ残る。実装者の挙げた 2 つ(終了コード 4 の項 1078 行「wait を打ち直す」、Monitor の期限切れの段落 1121 行「wait を打ち直し」と 1125 行「同じように打ち直し」)のほか、Claude Code の箇条の終了コード 0(1039 行「件を処理し、また待つ」)もそうである。1 本ずつ張る段落(1132 行)は例外なくすべての新しい wait にかかり、重ねたときの段落の最後の箇条もあるので、文として食い違うわけではない。ただ M1 で時間切れの 3 か所にだけ条件を付けたので、残りは対比で「条件なし」と読める。M1 と同じく、director が局所の 1 文で動く心配がある。いちばん起きやすいのは終了コード 0 で、重なった片方が件を返し、もう片方の次の読み(1 秒ごと)の前に director が ack すると、もう片方は走り続ける。終了コード 4 は両方の wait が 5 秒ごとに同じ見張りを確かめるので、ほぼ同時に終わる。Monitor の期限切れは `timeout_ms` を渡し忘れたときだけ | /sync-docs: 3 か所に「ほかの wait がまだ走っているあいだは打たない(下の、wait を 1 本ずつ張る段落)」を足す(4 写し、`scripts/sync-skills.sh`)。1125 行は、固定された fragment `あとからその Monitor の知らせが届いたら、そこで同じように打ち直し`(`directorSkillWaitSubsectionRules`)の後ろに足せば、ピンは崩れない。足さないなら理由を 1 行残す |
| V2 | LOW | 台帳の行がこの branch で当たったのに注がない。(a) 行 267(Trigger「`受信箱の処理` か `WaitInbox` の次の変更」): `### wait で待つ` は `## 受信箱の処理` の下にあり、`WaitInbox` の時間切れの文も変わった。(1) は払われず、同じ形の `t.Skipf` が新しいファイルに 2 つ増えた(`director_skill_wait_types_note_test.go` 218、300 行)。(2) の `TestDirectorSkill_NamesTheInboxWaitTimeoutPhrase` は触っていない。(3) も足していない。(b) 行 386、RC9(Trigger「`director_wait.go` の wait の loop か終了コード、または skill の終了コードの部分の次の変更」): loop の時間切れの枝と、終了コードの表の 2 項(`wait --inbox --director: ` の行と 1)が変わった。`director.jsonl` を 4 の前に読み直す直しも、テストも入っていない。(c) 行 324: Trigger の 2 つ目の条件と 2026-11-12 の注が `director_wait.go` と `director_wait_test.go` の大きさを追っていて、注の「821 and 824 now」は 833 と 829 になった。self-review の Tech debt の表は、`director_wait.go` の大きさの行が台帳にないとして新しい行を勧めるが、行 324 がそれである(self-review の grep は 261・327・386 を挙げ、324 を落とした)。(d) 行 327(INFO): Trigger の「`director_wait.go` の文」に `directorWaitTimeoutDetail` と時間切れの文が当たる。求める「触った塊ごとのテスト」はこの branch にあり、G6 の塊は触っていない。当たらないもの: 行 395 と行 274 (b)(M2 のあと `director_skill_sync_test.go` の差分は空)、行 426・427・261・262・182 | /sync-docs: 行 267 と 386 に日付つきの繰り越しの注、行 324 に 833・829 の数の注。新しい行は起こさない。行 327 の注は任意 |
| V3 | LOW | `/org` skill のコマンド表の `wait` の行は `--types` を help に沿って詳しく書く(ALERT も ack の期限に見張りが人へ送る、まで)が、時間切れの文が数えなかった件を名指すことは言わない | /sync-docs(任意): 1 句足す。この行を直すと行 182 (a)(コマンド表の次の編集)に当たるので、その注も要る |
| V4 | LOW | 記録の古い名前。計画の Progress の S1 の行が `inboxWaitOtherTypes` と書く(7e3448b7 で `readInboxWaitOtherTypesText` に改名)。Progress には 7e3448b7 の行がない(M1、M2、L1〜L6 の直し、ピンを `director_skill_wait_types_note_test.go` へ移したこと)。self-review の report の `inboxWaitOtherTypes` と `TestDirectorSkillStopsAWaitRe` は、その Scope の 61dd8ced では正しい名前で、56 行の台帳の文案と Tech debt の表は M2 の直しと行 324 で不要になった | /sync-docs: Progress に 7e3448b7 の行を 1 つ足し、改名を書く。self-review の report は 61dd8ced の記録なので書き換えない |
| V5 | INFO | 仕様の項目は FR-9 の注の末尾(#364 の項目の次)にある。計画の Scope S2 は「8b-2b の項に」と書くが、Progress の S2 の行は「FR-9 の末尾」と書き、director の規則の変更を FR-9 に置く今までの書き方とも合う | なし |
| V6 | INFO | skill の「`--director` の wait は、終わった印の detail にも同じ件を書く」と help の「a --director wait writes them in its mark's detail too」は条件を言わないが、印を引き継がれた wait は終わりを書かない。仕様の項目はそう書き、skill の重ねたときの段落も「やめるのは、印を書くことだけ」と言うので、合わせて読める | なし |
| V7 | INFO | 計画の AC1〜AC4 の checkbox は `[ ]` のまま | /pr か /sync-docs |
| V8 | INFO | Progress の「変異 10 個」「変異 28 個がすべて落ちた」の出力は repo にない(self-review も同じことを言う) | /test |

## Items for /sync-docs

1. V1: skill の打ち直しの文 3 つ(1039、1078、1121・1125 行)に条件を足すか、足さない理由を残す
2. V2: 台帳の行 267・386 に繰り越しの注、行 324 に `director_wait.go` 833 行・`director_wait_test.go` 829 行の注。self-review の勧める新しい行は起こさない
3. V3(任意): `/org` の `wait` の行に句。直したら行 182 (a) の注
4. V4: 計画の Progress に 7e3448b7 の行(改名と、ピンの移動)

## Coverage gaps

- テストは走らせていない。AC1〜AC3 のテストが通るか、重なりのテストが負荷の高い機械で安定するかは `/test` で見る
- director の wait の時間切れの文と `detail` は、binary では見ていない(登録と動いている見張りが要る)。コードとテストで見た
- issue #374 の原因(下流の台帳の e144・e145 の events と、Monitor のコマンド行の `--types`)は確かめていない。計画どおり `Refs #374` にし、ユーザーに確かめてもらう
- 足りない決定的な検査: なし。打ち直しの文の条件(V1)を固定するなら、`directorSkillWaitOneAtATimeSteps` に足した文の fragment を足すのがいちばん小さい

## Verdict

- Verdict: pass
- Verified: AC1〜AC4(コードとテストの読み、binary の help と時間切れの文、check-sync と check-skill-sync)。時間切れの終了コード 1 と `inboxWaitTimeoutPhrase` が先頭のままであること。self-review の M1、M2、L1〜L6 が 7e3448b7 に入ったこと。静的検査(full)が通ったこと
- Partially verified: AC2 と AC3 のテストの中身(読んだが走らせていない)
- Not verified: テストの実行(`/test`)、director の wait の binary での時間切れ、issue #374 の本当の原因
- Findings: LOW 4 件(V1〜V4、どれも /sync-docs へ)、INFO 4 件(V5〜V8)
- Insight event: `./scripts/insights-append.sh --slug director-wait-types-note --flow standard --phase verify --verdict pass --cycle auto --source skill` で 1 行(cycle 1)を足し、`--low` を渡し忘れた `low` の 0 を 4 に直して、このレポートと同じコミットに入れた
