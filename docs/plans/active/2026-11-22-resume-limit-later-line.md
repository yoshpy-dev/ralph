# resume-limit-later-line

- Status: Approved
- Approved: 2026-11-22 sha256:9683a080e69b
- Owner: Claude Code
- Date: 2026-11-22
- Related request: issue #373(`ralph org resume` が、上限の文が候補のまとまりの先頭にない画面を stopped と見分けない)
- Related issue: 373
- Type: fix
- Branch: fix/resume-limit-later-line

## Objective

claude の座席が上限(5 時間、週など)に当たって止まり、上限の文が入力欄の上の候補のまとまりの先頭ではなく、2 行目以降の `⎿` の行(ツールの結果、背景の知らせの結果)にある画面を、`ralph org resume` が stopped と見分け、見張りの usage_limited もその座席を上限と読むようにする。今は `unknown screen: no known stop sentence above the input box` で skipped になり、上限が解けても resume で続けられない。

## Scope

- S1 `internal/org/resume_screens.go` の `ClassifyResumeScreen` の手順 4 に (c2) を足す: 候補のまとまり(プロンプトのまとまりは除く)の 2 行目以降のうち、driver の結果の記号(claude の `⎿`)と空白で始まる行の**最後のもの**が、driver の `UsageLimit` の文で始まり、かつ**その項目の頭がツールか知らせの形**のとき stopped。項目の頭は、上限の `⎿` の行から上へ、`⎿` の行と記号のない続きの行(折り返し)を飛ばして最初に当たる、`⏺` で始まる行かまとめた行。まとまりの先頭まで頭がなく、先頭が `⎿` の行(プロンプトのすぐ下の結果のまとまり)なら、CLI の結果として数える。ツールか知らせの形は、claude では `⏺ Background command "`、`⏺ Monitor "`(既存の fixture `claude-weekly-limit-retrying.txt` の形)、`⏺ <ツール名>(`(`⏺ Bash(…)` など、英字で始まる名前と開き括弧)、まとめた行(`Ran <n> shell command`、`Read <n> file`。記号のある形とない形)で、driver の表の新しい欄に置く。頭が返信の文の行(`⏺ The seat showed:` など)なら、返信が画面を引用しただけかもしれないので数えない(unknown のまま。codex の計画助言と consult の指摘)。`Line` はその行からまとまりの終わりまでをつないだもの(記号と上限の文で始まるので、`UsageLimitScreen` の `matchedStopSentence(res.Line)` がそのまま読める)。API Error の文は今どおり候補の先頭でだけ数える。最後の結果の行が上限の文でなければ(`⎿  Interrupted`、ツールの出力)当たらない
- 結果の記号とツールか知らせの行の形は driver の表の新しい欄にする(claude は `⎿` と上の形、codex は空)。codex の画面は実機で確かめていないので、(c2) は claude だけに掛ける。まとめた行の言葉は、issue と既存の fixture で見た `Ran` と `Read` だけにし、見ていない言葉(`Searched` など)は入れない(入らない画面は今どおり unknown)
- 手順 4 の画面モデルのコメント(`resume_screens.go` 74-99 行付近)、`resumeBlock` に行を持たせる
- 既存のテストのうち、今の読み方を前提にした 2 件(`resume_screens_test.go` の「only the first ⎿ line under a prompt starts a block, the next ones join it」、`usage_limit_screen_test.go` の「a limit in the turn above a turn-end status line, with no retry」)の期待を、新しい読み方に合わせて直す。前者は 2 つ目の `⎿` を API Error にして「まとまりがつながる」を確かめる意図を残し、上限の形(プロンプトのすぐ下の結果のまとまり、頭なしで先頭が `⎿`)は stopped を期待する別の行にする。後者(`⏺ Monitor "…"` の知らせの下の上限)は stopped と上限の行を期待する
- S2 文書: `docs/specs/2026-11-07-org-seat-resume.md`(画面モデルの手順 4)、`.claude/rules/ralph/agent-messaging.md` と写し(usage_limited の段落の「`ralph org resume` が読む上限の文」の言い方)、必要なら `/director` skill の resume の項

## Non-goals

- プロンプトのまとまりを読むこと(issue の案 2)。プロンプトのまとまりに `⎿` で始まる行は入らない(最初の `⎿` で別のまとまりに分かれる)ので要らない
- codex の画面の (c2)(実機の画面がない)
- 上限の文の照合を typographic apostrophe(`You’ve`)にも広げること(今もある弱点で、別の話)

## Assumptions

- 調べ(scratchpad の probe373-red.txt と probe373-green.txt)のとおり、今のコードは issue の画面 1 と 2 を unknown にし、画面 3 は issue の文字どおりなら stopped、プロンプトの下に空行がある実機の形なら unknown になる。どちらも (c2) で stopped になる
- `UsageLimitScreen` は stopped のときだけ上限を読むので、(c2) で見張りの usage_limited も同じ画面を読むようになる
- `/usage-credits to finish what you’re working on.` の行は上限の行の続きとして `Line` に入るだけで、判定には効かない

## Affected areas

- `internal/org/resume_screens.go`、`internal/org/resume_screens_test.go`、`internal/org/usage_limit_screen_test.go`、新しい fixture(`internal/org/testdata/resume/` に issue の 3 画面と変形)
- `docs/specs/2026-11-07-org-seat-resume.md`、`.claude/rules/ralph/agent-messaging.md` と `templates/base/` の写し

## Visual review

None (1 つの関数の画面の読み方に 1 つの規則を足すだけで、部品の間のやり取りの順や、記録や状態の移り変わりは変えない)

## Design decisions

Critical forks: None

- (c2) は claude の結果の記号 `⎿` だけに掛ける。codex の `■`・`⚠` の画面は実機で見ていないので、今の (c) のまま
- 上限の文のあとの `resets <time>` を必須にする守りは入れない。claude の上限の行には `· resets` が付くが、ツールの出力が上限の文で始まる誤検知(手番が黙って終わった場合)はこの守りでも防げず、上限の言い方が変わったときに取りこぼす向きの危険を増やす
- 最後の結果の行だけを見る。上限の行の下に `⎿  Interrupted` やほかの結果がある画面は、上限のあとに続きがあったので数えない

## Acceptance criteria

- [x] AC1: issue の画面 1(ツールの実行のあと)と画面 2(背景の知らせのあと)を、入力欄と footer つきの fixture にし、`ClassifyResumeScreen` が claude で stopped、`Line` が `⎿  You've hit your 5-hour limit` で始まると固定する。プロンプトの下に空行がある画面 3 の形、知らせの行が折り返した画面 2 の形、`⏺ Monitor "…"` の知らせの下の上限(既存の fixture の形)も同じ
- [x] AC2: 同じ fixture で `UsageLimitScreen` が上限と読み、リセットの時刻を読める
- [x] AC3: 誤検知の画面は stopped にならない: 最後の `⎿` の行が API Error(候補の先頭でない)、上限の行の下に `⎿  Interrupted`、上限の行の下に返信のまとまり、返信の文の中で上限の文を引用、記号のない行の上限の文、古い上限のまとまりの下に普通の返信、返信の文の行(`⏺ The seat showed:`)のすぐ下に `⎿  You've hit your weekly limit · resets …` を引用して終わった画面(状態行のある形とない形、retry の文なし。codex の計画助言の指摘)、見ていないまとめた行(`Searched for 1 pattern`)の下の上限の行。codex の表では (c2) が働かない
- [x] AC4: 今の読み方の既存のテスト(stopped、waiting、enter_only、unknown の各場合)は、期待を直す 2 件を除いて変わらない。直す 2 件は計画のとおり意図を残す
- [x] AC5: 仕様の画面モデルと rule の usage_limited の段落が (c2) を言う。`./scripts/check-sync.sh` が通る

## Implementation outline

1. S1 `ClassifyResumeScreen` の (c2)、表の結果の記号の欄、fixture とテスト(AC1〜AC4) — verify: `test -z "$(gofmt -l internal/org)" && go vet ./internal/org/ && GOFLAGS="-timeout=30m $(go env GOFLAGS)" go test -count=1 ./internal/org/ -run 'Resume|UsageLimit|SendScreen|Watch'`
2. S2 仕様と rule(AC5) — verify: `./scripts/check-sync.sh && GOFLAGS="-timeout=30m $(go env GOFLAGS)" go test -count=1 ./internal/org/ -run 'DocsMatch|AgentMessaging|Resume'`

## Verify plan

- Static analysis checks: `./scripts/run-static-verify.sh`
- Spec compliance criteria to confirm: AC1〜AC5。(c2) が API Error の扱いを変えないこと、プロンプトのまとまりを読まないこと
- Documentation drift to check: 仕様の画面モデル、rule の usage_limited の段落、`/director` skill の resume の項、`ralph org resume --help`
- Evidence to capture: fixture のテストの出力、変異の確かめ

## Test plan

- Unit tests: `ClassifyResumeScreen` の (c2) の表のテスト(claude と codex)、`UsageLimitScreen` の上限とリセットの時刻
- Integration tests: `ralph org resume --dry-run` の fake の画面で、issue の画面が `skipped` でなく続ける対象になること(既存の resume のテストの形で)
- Regression tests: resume_screens_test.go と usage_limit_screen_test.go の既存の全件、send の画面の読み(`SendScreen`)、見張りの usage_limited
- Edge cases: 上限の行が折り返されて `(Asia/Tokyo)` が次の行にある、`/usage-credits` の行が続く、状態行の前に空行がない、`⎿` のあとが U+00A0、結果の行が 1 行だけの候補(今の (c) と同じ答え)
- Evidence to capture: `go test` の出力、変異(新しい規則を外す、最後でなく最初の結果の行を見る、記号の欄を codex にも入れる)で落ちること

## Risks and mitigations

- 見ていない形のツールの行(`Searched for …` など)の下で上限に当たった画面は、今どおり unknown で skipped になる: 取りこぼす向きで、今より悪くはならない。実機で見た形が増えたら表に足す。Known gaps に書く
- ツールの出力の 1 行目が上限の文で始まり(`cat` した文書など)、そのまま手番が黙って終わった座席を stopped と読む: resume は続行の文を 1 行打ち、見張りは stall の代わりに usage_limited を上げる。本物の上限がツールのあとに来るときは最後の結果の行として足されるので、誤検知は手番が黙って終わったときに限る。続行の文は「止まったところから続けて」という害の小さい文で、usage_limited は人に知らせる。Known gaps に書く
- 既存の 2 件のテストの期待を変える: 前者は意図(まとまりがつながる)を残して別の文で確かめ、後者は #353 の計画の AC4(今の stopped の読み方の対象でなければ上限と読まない)と矛盾しないことを verify で確かめる

## Rollout or rollback notes

- 新しい binary が要る。動いている見張りは、立て直すまで前の読み方
- revert すると、上限の文が候補の先頭にある画面だけを読む前の形に戻る

## Open questions

なし

## Progress checklist

- [x] Plan reviewed — plan advisory: ran with codex(gpt-6-astra、xhigh)。指摘 1 件、計画に入れた。[MEDIUM] 1: 新しい規則は、普通に終わった返信が上限の行を `⎿` の形で引用しただけの画面(`⏺ The seat showed:` の下に `⎿  You've hit your weekly limit · resets …`)も stopped と読む(leader や director は座席の画面を引用することがある) → (c2) を、上限の `⎿` の行のすぐ上がツールか知らせの行(`⏺ Background command "`、`⏺ <ツール名>(`、まとめた行 `Ran <n> shell command`・`Read <n> file`)のときだけに絞り、その引用の画面を状態行のある形とない形で否定の fixture にした(Scope S1、Design decisions、AC3、Risks)。git の状態は助言の前後で同じ。consult(consult-plan-373-374): 判定「直してから進める」。表に `⏺ Monitor "` がない(既存の fixture と `usageLimitMonitorLimit` の形)、「すぐ上の行」は知らせの行の折り返しで取りこぼす、試作のプロンプトの下の結果のまとまりのテストと合わない → 規則を「上限の `⎿` の行から上へ `⎿` の行と続きの行を飛ばした項目の頭」に変え、`⏺ Monitor "` を足し、頭なしで先頭が `⎿` のまとまり(プロンプトのすぐ下)は数える、とした(Scope S1、AC1)
- [x] Plan approved — 系列の決まり(推奨案で自動承認、2026-10-11 のユーザーの指示)で承認
- [x] Branch created
- [x] Implementation started
  - S1 2c418694(最初の implementer は範囲外の `send_screen_test.go` が全 fixture を読むテストで落ちるのでコミットせずに止まり、範囲を足した resume で仕上げた): `findResumeLimitResult` と表の欄 `ResultGlyphs`・`ItemGlyphs`・`ToolOrNoticeHeads`(claude だけ。2c418694 の名前は `ResultHeads` で、d470a77a が `ToolOrNoticeHeads` に改名した)。計画にない守りを 1 つ足した: 上限の `⎿` の行より下に項目の頭があれば (c2) を当てない(上限のあとに続きがあった画面を数えない、Design decisions の趣旨)。fixture 5 件、否定 14 件、codex 3 件、`ralph org resume --dry-run` の統合テスト(直す前のコードでは issue と同じ `skipped … unknown screen` で落ちることを確かめた)。既存 2 件の期待を計画どおり直した(#353 の AC4 と矛盾しない)。変異 15 個のうち 14 個が落ち、残る 1 個は等価
  - S2 90a93764: 仕様 FR-2 の 4 に 2026-11-22 の改訂((c2) の 3 条件、まとめた行の言葉は `Ran`・`Read` だけ、codex に当てない、見張りも同じ画面を読む)、Open questions の補足、rule(2 写し)の usage_limited の段落の例に 1 つ足した、`usage_limit_screen.go` のコメント 2 か所。`/director` skill と `ralph org resume --help` は「既知の文で終わる画面」と書いていて (c2) でも正しいので変えていない
  - d470a77a(self-review の M1・L2・L3・L4 の直し): M1 は `resumeBlock` に `resultUnderPrompt` を足し、`resumeScreenBlocks` が prompt の行のあとの最初の `⎿` の行で切ったまとまりにだけ立てて、頭がないときの許しをその印だけにした(空行のあとで `⎿` から始まる引用は数えない。否定のテスト 2 行)。空行のあとにまとめた行で始まる引用(P2)は issue の画面 3 と区別できず stopped のままで、テストが known gap として固定している。L2 は `findResumeLimitResult` が上限の行とその下の行をつないだ文で照合するようにした(pane の幅で割れた上限の文も読む)。L3 は `ResultGlyphs` が `StopGlyphs` に含まれることを doc に書き、`TestResumeResultGlyphs_AreStopGlyphs` で固定した。L4 は `ResultHeads` → `ToolOrNoticeHeads`、`resumeResultHead` → `resumeToolOrNoticeHead`、`resumeResultOfTool` → `resumeResultUnderToolOrNotice`、戻り値 `tool` → `toolOrNotice` の改名。M2(台帳)は /sync-docs で台帳に入れ、L1(入れ子の `⎿` の行)と P2 は台帳の行と /pr の Known gaps に回した
  - cycle 2 self-review の M1・L1・L2 の直し(この行を足したコミット): M1 は、`resumeScreenBlocks` が prompt のあとで項目の頭と読む行を、字下げを落とす前の行が `⏺` で始まる行(1 桁目)に限った。座席の画面を写した prompt の字下げした `  ⏺ …` の行は prompt のまとまりに残る(否定のテストは切り方 2 行、resume 2 行、resume と見張り 2 行。対照に、1 桁目の `⏺ API Error` を stopped とする 1 行)。L2 は `resumeScreenBlocks` の引数を marker から driver にし、`resumeMarkerItemGlyphs` を消した。L1 は `send_screen.go` の 2 か所のコメントに、send の prompt のまとまりは項目の頭では切れないと書いた
- [x] Review artifact created — `docs/reports/self-review-2026-11-22-resume-limit-later-line.md`(cycle 1、Merge: yes、MEDIUM 2・LOW 4。M1: 頭のない結果のまとまりの許しが prompt のすぐ下かを見ず、空行のあとの 2 行以上の `⎿` の引用を stopped と読む。M2: 台帳の行 490・517・542・544 の Trigger を満たしたが繰り越しがなく、800 行を越えた 2 ファイルの行もない)。cycle 2 は同じ report の `## Run 2 (cycle 2)`(HEAD `1042ccae`、Merge: yes、MEDIUM 1・LOW 3。M1: `1042ccae` の切れ目が字下げを見ず、`⏺` で始まる prompt の続きの行を項目の頭と読む。座席の画面を写した prompt のあと中断した画面が stopped になる。L3: 台帳の 570・572 行と仕様 54 行が古くなった)
- [x] Verification artifact created — `docs/reports/verify-2026-11-22-resume-limit-later-line.md`(cycle 1、pass。LOW 4・INFO 4。AC1〜AC5 は Met、静的検証 rc=0。V1: 台帳の行 462・483 の Trigger も満たす。V2・V3: 仕様 FR-2 の改訂の 59・61 行が `d470a77a` のつないだ文の照合と prompt のすぐ下の印を言わない。V4: P2 の形と Risks の 2 つが Known gaps にない。どれも `/sync-docs` へ)
- [x] Test artifact created — `docs/reports/test-2026-11-22-resume-limit-later-line.md`(cycle 1、pass。`RALPH_VERIFY_SCOPE=full ./scripts/run-test.sh` rc 0、1,349 s、SKIP は条件つきの 2 つだけ。足したテストは `-count=1 -v` で SKIP なし。変異 7 個のうち 6 個が落ち、codex の表に `⎿` を包含つきで足す M4b が残った(LOW、codex の行にプロンプトの下の結果のまとまりがない))
- [x] Sync-docs artifact created — `docs/reports/sync-docs-2026-11-22-resume-limit-later-line.md`(cycle 1、HEAD `32440422` の木。1 コミット)。verify の引き継ぎ 5 項目と test の Test gap 1 を片づけた。台帳は Trigger を満たした 7 行(self-review・verify の 6 行と、掃き出しで見つけた行 567)に繰り越しの注を足し、新しい行を 5 つ(大きさ、P2、入れ子の `⎿`、見ていない頭、この branch のテストの穴)足した。仕様 `2026-11-07-org-seat-resume.md` の 59・60・61 行を直し(つないだ文の照合、P2 の 1 文、prompt のすぐ下で切ったまとまり)、rule(2 写し)の頭の例を `such as` にして `Read 2 files` と `⏺ Monitor "..."` を足し、`2026-08-01-org-runtime.md` の 89 行に 1 文足した。`check-sync.sh` は PASS、狭い Go テスト(org と cli)は ok、digest は前後とも同じ。AC1〜AC5 に `[x]`
- [ ] PR created
