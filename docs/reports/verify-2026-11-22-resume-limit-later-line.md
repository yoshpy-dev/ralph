# Verify report: resume-limit-later-line

- Date: 2026-11-22
- Plan: `docs/plans/active/2026-11-22-resume-limit-later-line.md`(issue #373。承認の digest `9683a080e69b` は HEAD の本文から計算した値と一致)
- Verifier: verifier サブエージェント(`/verify`)。パイプライン 1 回目(`cycle-count.json` は cycle 1)
- Scope: origin/main `b263b1f3` から HEAD `d470a77a` までの `git diff origin/main...HEAD`(17 ファイル、+947/-41)。self-review の対象(`444adf11` まで)に、そのあとの直し `d470a77a`(M1・L2・L3・L4)を加えた範囲。テストは実行していない(`/test` の担当)
- Evidence: `docs/evidence/verify-2026-11-22-resume-limit-later-line.log`(git の管理外。静的検証の全出力、`go vet`、digest、rule の 2 写しの `cmp`、base と HEAD の `wc -l`、テストのファイルで消えた行、改名前の名前の grep、仕様と rule の該当文、台帳の Trigger のセル)

## Spec compliance

行番号は HEAD `d470a77a` のもの。

| Acceptance criterion | Status | Evidence |
| --- | --- | --- |
| AC1: issue の画面 1・2 を入力欄と footer つきの fixture にし、claude で stopped、`Line` が `⎿  You've hit your 5-hour limit` で始まる。プロンプトの下に空行がある画面 3、知らせの行が折り返した画面 2、`⏺ Monitor "…"` の知らせの下の上限も同じ | Met(読んで確かめた。実行は `/test`) | fixture 5 件(`internal/org/testdata/resume/claude-5h-limit-after-{shell-command,background-notice,background-notice-wrapped,read-under-prompt}.txt`、`claude-weekly-limit-after-monitor.txt`)を `TestClassifyResumeScreen_Fixtures` の 5 行が stopped と `Line` で固定する。issue の文字どおりの 3 画面(空行なし)は `TestClassifyResumeScreen_UsageLimitOnTheLastResultLine` の最初の 3 行が固定する。5 件を手でたどった: 画面 1 は `Ran 1 shell command` が `Bare` の頭、画面 2 と折り返し版は `⏺ Background command "` の頭(折り返しの続きの行は頭でも結果の行でもないので飛ばす)、画面 3 は `Read 2 files` が頭、Monitor は `⏺ Monitor "` の頭。CLI でも `TestOrgResume_DryRunResumesASeatStoppedOnALimitUnderAToolOrANotice`(`internal/cli/org_resume_test.go:426`)が `would_resume` と上限の行を固定する |
| AC2: 同じ fixture で `UsageLimitScreen` が上限と読み、リセットの時刻を読める | Met(読んで確かめた) | `usageLimitFixtureWants` に 5 行を足し(真の行は 5 → 10、fixture 86 件に 86 行)、`line` と `has`(`resets 4:40am`、`(Asia/Tokyo)`)を固定する。`UsageLimitScreen`(`usage_limit_screen.go:93`)は stopped のとき `matchedStopSentence(res.Line)` で文を決め、`UsageLimit` に入る文なら `usageLimitLine(res.Line)` を返す。(c2) の `Line` は `⎿`、空白、上限の文の順で始まるので、`findResumeStop` の glyph の規則で一致する。見張りは `readUsageLimitScreen`(`watch_usage_limit.go:128`)で同じ `UsageLimitScreen` を呼ぶ。リセットの時刻は行の中の文字として運ばれる(`usageLimitResetsIn` は 3 つ目の読み方 `usageLimitRetryingTurn` だけが使い、(c2) の経路は通らない) |
| AC3: 誤検知の画面は stopped にならない(列挙の 9 形と codex) | Met(読んで確かめた) | 表のテストの AC3 の節に、最後の `⎿` の行が API Error、上限の下の `⎿  Interrupted`、上限の下の返信のまとまり、返信の文の中の引用、印のない行の上限、古い上限の下の普通の返信、`⏺ The seat showed:` のすぐ下の上限(状態行のある形とない形)、`Searched for 1 pattern` の下の上限、codex の 3 形がある。どれも `resumeReasonNoStop` と `UsageLimitScreen` の false を期待する。ほかに self-review の M1 の 2 形(空行のあとの 2 行の `⎿` の引用、868・873 行)、頭のない普通の行、`⏺` のあとに空白がない頭、`⎿` のあとに空白がない上限も否定で固定している |
| AC4: 既存のテストは期待を直す 2 件を除いて変わらない | Met(diff で確かめた) | テストの 4 ファイルで消えた行は、`resume_screens_test.go` の「only the first ⎿ line under a prompt starts a block, the next ones join it」のコメント 2 行と画面 1 行、`usage_limit_screen_test.go` のコメント「Only the five usage limit screens are true」と `resume: ResumeScreenUnknown` の 1 行だけ。前者は 2 つ目の `⎿` を API Error にして「まとまりがつながる」意図を残し、上限の形は別の行(509 行、stopped)に移した。後者は #353 の計画の AC4(今の stopped の読み方の対象でなければ上限と読まない)と矛盾しない: その画面は (c2) で stopped になり 1 つ目の読み方が読む。3 つ目の読み方だけでは今も読まないことを `TestUsageLimitScreen_Issue373LimitUnderAMonitorNoticeIsAStop`(606 行)が固定する。`send_screen_test.go` と `org_resume_test.go` は追加だけ |
| AC5: 仕様の画面モデルと rule の usage_limited の段落が (c2) を言う。`check-sync.sh` が通る | Met(文のずれは V2・V3) | 仕様 FR-2 の 4 に 2026-11-22 の改訂(58-63 行)と Open questions の補足(205 行)がある。rule の usage_limited の段落(2 写し、`cmp` で同一)が (c2) を言う。静的検証の中の check-sync は PASS(DRIFTED 0)。ただし仕様の 59・61 行は `d470a77a` より前の文のまま(V2・V3) |

計画の Verify plan の残り 2 点: (c2) は API Error の扱いを変えない(`findResumeLimitResult` は `UsageLimit` の文だけを照合し、最後の `⎿` の行が API Error なら unknown。表のテストの AC3 の 1 行目)。プロンプトのまとまりを読まない(`findResumeLimitResult` は `block.prompt` で false を返す、`resume_screens.go:796` の直後)。

### self-review の直し(`d470a77a`)の確かめ

| ID | 直ったか | 確かめたこと |
| --- | --- | --- |
| M1 | 直った | `resumeBlock` に `resultUnderPrompt`(683 行)を足し、`resumeScreenBlocks` が prompt のあとの最初の `⎿` の行で切ったまとまりにだけ立てる(714 行。空行の flush で落ちる)。頭のないときの許しは `return block.resultUnderPrompt`(837 行)。否定の 2 行(868・873 行)が固定し、元の `resumeStartsWithAny(above[0], …)` に戻すとこの 2 行が stopped になる(読んで確かめた)。肯定の側は Edges の 509 行。区別できない P2 の形(空行のあと、まとめた行で始まる引用)は 884 行で stopped と固定し、コメントで known gap と書いた |
| L2 | 直った | `findResumeLimitResult` は上限の行とその下の行をつないだ文(803 行、`Line` と同じ文)で glyph のあとを照合する。804 行のテスト(上限の文が `You've hit your` と `5-hour limit · resets 4:40am` に割れる)が固定する。つなぐ行は、最後の結果の行より下にある、頭でも結果の行でもない行だけ(下に頭があれば先に false を返す) |
| L3 | 直った | `ResultGlyphs` の doc に「Each is also one of StopGlyphs」を足し、`TestResumeResultGlyphs_AreStopGlyphs`(`resume_screens_test.go:1102`)が glyph の包含と `matchedStopSentence` の一致を driver ごとに確かめる |
| L4 | 直った | `ResultHeads` → `ToolOrNoticeHeads`、`resumeResultHead` → `resumeToolOrNoticeHead`、`resumeResultOfTool` → `resumeResultUnderToolOrNotice`、戻り値 `tool` → `toolOrNotice`。古い名前はコードとテストに残っていない。残りは計画の Progress の S1 の行だけ(V6) |
| M2・L1 | 予定どおり未修正 | 台帳の編集と Known gaps は `/sync-docs` と `/pr` に回す(下の「`/sync-docs` への引き継ぎ」) |

## Static analysis

| Command | Result | Notes |
| --- | --- | --- |
| `RALPH_VERIFY_SCOPE=full ./scripts/run-static-verify.sh` | pass(rc=0) | full scope で選ばれた pack は golang だけ(changed scope と同じ)。hook の shellcheck と `sh -n`、settings.json の jq、Codex の hook の 3 つの guard、check-sync(DRIFTED 0)、check-pipeline-sync、check-skill-sync(14 skill)、check-template-purity、tech-debt の plan 参照、gofmt(ok)、`go vet ./...`、golangci-lint(0 issues)、staticcheck(出力なし)、branch の secret scan(`b263b1f3..d470a77a` clean)がすべて通った |
| `go vet ./internal/org/ ./internal/cli/` | pass | 上の `go vet ./...` と重なるが、変更のある 2 パッケージを単独で確かめた |
| `./scripts/plan-visual.sh digest <plan>` | `9683a080e69b` | 計画の `- Approved:` の行と一致 |
| `cmp`(rule の 2 写し) | 同一 | `.claude/rules/ralph/agent-messaging.md` と `templates/base/` の写し |

## Documentation drift

| Doc / contract | In sync? | Notes |
| --- | --- | --- |
| `resume_screens.go` の画面モデルの前文(手順 4 の (c2))と関数の doc | Yes | `d470a77a` で、つないだ文の照合、prompt のすぐ下で切ったまとまりの印、上限の行より下の頭の守りを言う。`resumeBlock.resultUnderPrompt`、`findResumeLimitResult`、`resumeResultUnderToolOrNotice` の doc もコードと合う |
| `usage_limit_screen.go` のコメント(`UsageLimitScreen` の `line`、`matchedStopSentence`) | Yes | (c2) の `Line` を名指しする。`StopGlyphs` の doc も (c2) の例外を言う |
| `docs/specs/2026-11-07-org-seat-resume.md` FR-2 の 4 の改訂(58-63 行) | 一部ずれ(V2・V3・V4) | 上限の行より下の頭の守りは言う(59 行)。prompt のすぐ下で切ったまとまりの印は括弧の中だけで、条件の文は前の形(61 行)。つないだ文の照合は言わない(59 行) |
| `.claude/rules/ralph/agent-messaging.md`(2 写し)の usage_limited の段落 | Yes(INFO V5) | (c2) を言う。挙げる頭の例が閉じた列挙に読める |
| `/director` skill の resume の項、`ralph org resume --help` | Yes | どちらも「既知の文で終わる画面」と書き、(c2) の画面にも当てはまる。計画の Progress の判断と同じ |
| `docs/specs/2026-08-01-org-runtime.md` 89 行 | Yes(INFO V7) | 「resume の判定が止まったと読み、決め手の文が利用上限の文」は (c2) のあとも正しい |
| `ralph org status --live` | 対象外 | `ClassifySendScreen` を使い、`ClassifyResumeScreen` は通らない。send の表に足した 5 行は idle を期待する |
| `docs/tech-debt/README.md` | No(V1、self-review M2) | この diff が満たす Trigger は 6 行。self-review は 4 行を挙げ、行 462 と 483 を挙げていない |
| 計画の Progress checklist | 一部ずれ(V6) | S1 の行が改名前の `ResultHeads` を使い、`d470a77a` の行がない |

## Findings

| ID | Severity | Finding | Evidence | Hand-off |
| --- | --- | --- | --- | --- |
| V1 | LOW | self-review の M2 が挙げた 4 行(490・517・542・544 の (c))のほかに、この diff は台帳の 2 行の Trigger を満たす。行 462(「Seven mutants of `ralph org resume` survive」で始まる)の Trigger は「`ClassifyResumeScreen` の次の変更」で、この PR は (c2) の呼び出しを足した(`resume_screens.go:458`)。開いている (a)・(b)・(d)・(f)・(g) はどれも払っていない((d) の enter_only と waiting の文を両方出す fixture は今もない)。行 483(「A hook that was changed after codex approved it」で始まる)の Trigger は「`resume_screens.go` の画面の規則の次の変更」で、(c2) は新しい画面の規則。"Hooks need review" の条件は足していない。どちらの行も #329・#353 で同じ形の繰り越しの文を持つ | 台帳の 4 つ目のセルを `re.split(r'(?<!\\) \| ', 行)[3]` で読んだ(evidence の末尾)、`resume_screens.go:458` | `/sync-docs`: 2 行の Trigger の末尾に「2026-11-22: fix/resume-limit-later-line (#373) hit this trigger with …, did not pay it: …. Carried over to the next change that meets this trigger」を足す |
| V2 | LOW | 仕様 61 行は、頭がないときの条件を「先頭の行が `⎿` で始まる(prompt の行のすぐ下の結果のまとまり)」と書く。`d470a77a` 以後のコードは、prompt の行のあとの最初の `⎿` の行で切ったまとまり(`resultUnderPrompt`)だけを許し、空行のあとで `⎿` から始まるまとまりは数えない。括弧の中は今のコードと合うが、条件の文は前の形で、文のとおりに読むと self-review の P1 の形(空行のあとの 2 行の `⎿` の引用)は stopped になる。テストはそれを unknown と固定している | 仕様 61 行、`resume_screens.go:714`・`:837`、`resume_screens_test.go:868`・`:873` | `/sync-docs`: 61 行を「prompt の行のあとの最初の `⎿` の行で始まったまとまり(手順 1 の区切り)である。空行のあとで `⎿` から始まるまとまり(返答が座席の画面の `⎿` の行を段落として引用した形)は数えない」に直す |
| V3 | LOW | 仕様 59 行は、最後の行そのものが「`⎿`、空白、利用上限の文の順で始まる」と書く。`d470a77a` 以後は、その行と下の行をつないだ文で照合するので、pane の幅で上限の文が割れた行も当たる(L2)。同じ行の「上限の行の下の印のない行は数えない」は最後の行を選ぶ段では今も正しいが、照合では使う | 仕様 59 行、`resume_screens.go:803`、`resume_screens_test.go:804` | `/sync-docs`: 59 行に「照合は、その行とまとまりの残りの行を空白 1 つでつないだ文で行う(pane の幅で割れた上限の文も読む)」を 1 文足す |
| V4 | LOW | 仕様の改訂は「頭が返答の文の行のときは…数えない」と言うが、引用が空行のあとでまとめた行から始まる形(self-review の P2)は、issue の画面 3 の実機の形と区別できず stopped になる。テストは 884 行でこれを known gap として固定しているが、仕様、計画の Risks、台帳のどれにも書いていない。計画の Risks の 1 つ目と 2 つ目(「Known gaps に書く」)と self-review の L1(ツールの出力の中の入れ子の `⎿` の行)も、まだどこにもない | 仕様 60 行、`resume_screens_test.go:884`、計画 83-84 行、self-review の「Tech debt identified」 | `/sync-docs`: 仕様 205 行の補足か 60 行の後ろに P2 の形を 1 文足す。台帳に self-review の行案 1〜3 を入れる(行案 1 の「M1 を直さない場合は P1」は M1 が直ったので消す)。`/pr`: Known gaps に同じ 4 点 |
| V5 | INFO | rule の usage_limited の段落は (c2) の頭を「under `Ran 1 shell command`, `⏺ Bash(...)`, or `⏺ Background command "..."`」と挙げ、閉じた列挙に読める。コードは `⏺ Monitor "`、`Read <n> file`、`⏺ <名前>(` の全般、頭のない prompt のすぐ下の結果のまとまりも読む。同じ段落の後ろは #353 の Monitor の画面の retry の場合を書くので、Monitor が抜けているのは目に付く | `.claude/rules/ralph/agent-messaging.md` 229-232 行、`resume_screens.go` の `ToolOrNoticeHeads` | 任意。台帳 544 の (c)(retry の場合を一覧のあとの別の文に移す)を払うなら、同じ編集で「such as」を足すか Monitor と `Read 2 files` を足す(2 写し) |
| V6 | INFO | 計画の Progress の S1 の行が改名前の `ResultHeads` を使い、`d470a77a`(M1・L2・L3・L4 の直し)の行がない | 計画 102 行 | `/sync-docs`: S1 の行の名前を `ToolOrNoticeHeads` に直し、self-review の行か新しい行に `d470a77a` を一文で書く(Progress は digest に入らない) |
| V7 | INFO | 仕様 `docs/specs/2026-08-01-org-runtime.md` 89 行の usage_limited の読み方は、resume の「止まった」の判定に頼ると書くので、(c2) のあとも正しい。日付つきの一文で (c2) の画面が増えたことを足してもよい | 同 89 行 | 任意 |
| V8 | INFO | 計画の受け入れ条件の印(AC1〜AC5)がすべて `[ ]` のまま。この report では 5 つとも Met | 計画 55-59 行 | `/pr` の前に付ける |

## `/sync-docs` への引き継ぎ

1. 台帳の繰り越しの文(Trigger の末尾に足す)。行は最初の語で探す
   - 「`internal/org/send_screen_test.go` (662 → 831 lines) and `internal/org/send_fallback_test.go`」: `send_screen_test.go` 885 → 893(5 行の fixture の期待)。数字の行のテストは移していない
   - 「Test gaps of fix/claude-5h-usage-limit (#329)」: `usage_limit_screen_test.go` に `TestUsageLimitScreen_Issue373LimitUnderAMonitorNoticeIsAStop` と `usageLimitFixtureWants` の 5 行を足し、`UsageLimitScreen` の doc を変えた。(a)〜(d) は足していない
   - 「`internal/org/usage_limit_screen_test.go` and `internal/org/resume_test.go` went past」: `usage_limit_screen_test.go` 960 → 1013。#353 の塊の中の `TestUsageLimitScreen_RetryWaitShapes` の 1 行の期待を直したが、塊は移していない
   - 「Three LOW findings of the cycle 1 self-review of fix/watch-usage-limit-retry (#353)」の (c): usage_limited の段落を 2 写しとも変えたが、retry の場合を別の文に移していない(V5 と一緒に払える)
   - 「Seven mutants of `ralph org resume` survive」(V1): `ClassifyResumeScreen` に (c2) を足した。(a)・(b)・(d)・(f)・(g) は払っていない
   - 「A hook that was changed after codex approved it」(V1): (c2) の画面の規則を足した。"Hooks need review" の条件は足していない
2. 台帳の新しい行
   - 大きさ: `internal/org/resume_screens.go` 705 → 902 行、`internal/org/resume_screens_test.go` 781 → 1119 行(self-review の 882・1056 は `d470a77a` の前の数)。分け方は self-review の行案 4 のとおり
   - known gap: self-review の「Tech debt identified」の行案 1〜3(V4)
3. 仕様 `docs/specs/2026-11-07-org-seat-resume.md`: 59 行(V3)、61 行(V2)、P2 の形の一文(V4)
4. 計画の Progress: S1 の行の `ResultHeads`、`d470a77a` の行(V6)。ほかに Verification の行はこの report のコミットで付けた
5. 任意: rule の頭の例(V5、2 写し)、`docs/specs/2026-08-01-org-runtime.md` 89 行(V7)

## Observational checks

- 実機の画面は見ていない。fixture は issue #373 の画面の文をそのまま使い、ブロックのあいだに空行を足したもの(既存の Claude Code 2.1.316 の取り込みに合わせた)。issue のとおり空行のない形は表のテストの最初の 3 行が持つ。空行を足した形が実機と同じかは確かめていない
- 振る舞いはコードを読んでたどった。テストも probe も走らせていない

## Coverage gaps

- テストは走らせていない。新しい表のテスト、fixture の 5 行、CLI のテスト、期待を直した 2 件が通るかは `/test` で確かめる
- `d470a77a` の直しの変異(`resultUnderPrompt` を前の `above[0]` の規則に戻す、照合を 1 行に戻す)が落ちることは、どの行が落とすかを読んで確かめただけ。計画の Progress の変異 15 個は `d470a77a` の前の S1 のもの
- 見ていない頭(`Searched for …`、MCP の `(MCP)(`、`Web Search(`)の下の上限と、上限の文が割れた行のうち頭が見つからない形は unknown のまま(計画の Risks の 1 つ目)。実機の画面がないので、正しい頭の一覧かは分からない
- 決定的な検査で足りないもの: 仕様の日付つきの改訂の文を固定するテストはなく、V2・V3 はそのため `d470a77a` のあとも残った。足すなら、仕様 61 行の文と前文の (c2) の段落が同じ条件(「the first ⎿ line after a prompt line」)を言うことを `TestClassifyResumeScreen_*` の近くで 1 つの fragment で確かめるのが最小

## Verdict

- Result: pass(LOW 4、INFO 4。どれも merge を止めない)
- Verified: AC1〜AC5 をコード、fixture、テストの行、仕様と rule の文と照らした。self-review の M1・L2・L3・L4 の直しが入ったこと。見張りの `UsageLimitScreen` が (c2) の `Line` を `matchedStopSentence` で読むこと。静的検証 rc=0、rule の 2 写しの一致、計画の digest の一致
- Partially verified: AC5(仕様の文が `d470a77a` の直しの 2 点を言わない。V2・V3)
- Not verified: テストの実行(`/test` の担当)、実機の Claude Code の画面での (c2) の判定、`ralph org resume` と見張りの実機での動き
