# Verify report: resume-limit-later-line

## Run 2 (cycle 2)

- Date: 2026-11-22
- Plan: `docs/plans/active/2026-11-22-resume-limit-later-line.md`(issue #373。承認の digest `9683a080e69b` は HEAD の本文から計算した値と一致)
- Verifier: verifier サブエージェント(`/verify`)。パイプライン 2 回目(`cycle-count.json` は cycle 2)。今の判定は下の `## Verdict` で、Run 1 は見出しを 1 段下げて `---` の下に残した
- Scope: Run 1(HEAD `d470a77a`、report のコミット `836cb19f`)のあとにコードを変えた 2 コミット。`1042ccae` は cross-review cycle 1 の ACTION_REQUIRED 1 の直しで、`resumeScreenBlocks` が prompt の行のあとの項目の頭からも新しいまとまりを始める。`e9017d7f` は self-review Run 2 の M1・L1・L2 の直しで、項目の頭を 1 桁目の `⏺` の行に限り、`resumeScreenBlocks(lines, driver)` にし、send のコメントを直した。`c4e6f979` は self-review の report。`git diff b061c0c9..HEAD` は 8 ファイル、+294/-33。そのあいだの `90d5d824`(test)、`1dda5f7f`(sync-docs)、`b061c0c9`(triage)、`32440422`(origin/main の取り込み)は、Run 1 の所見の行き先を確かめるためにだけ読んだ。テストは実行していない(`/test` の担当)
- Evidence: `docs/evidence/verify-2026-11-22-resume-limit-later-line-run2.log`(git の管理外)。Run 1 の本文がその log の「末尾」を引くので、Run 1 の log には足していない。中身は静的検証の全出力、`go vet`、`gofmt -l`、digest、rule の 2 写しの `cmp`、4 時点の `wc -l`、テストのファイルで消えた行、消えた関数名の grep、fixture の走査、台帳のセル

### Spec compliance (cycle 2)

行番号は HEAD `e9017d7f` のもの。cycle 2 のコミットは受け入れ条件を足しておらず、計画の AC1〜AC5 の文も変えていない。

| Acceptance criterion | Status | Evidence |
| --- | --- | --- |
| AC1 | Met(読んで確かめた。実行は `/test`) | fixture 5 件と `TestClassifyResumeScreen_Fixtures` の 5 行は cycle 2 で変わっていない。`internal/org/testdata/` の fixture 52 件のうち、prompt の行のあと空行なしに 1 桁目の `⏺` の行が来るものは 0 件(python の走査、evidence)なので、新しい切れ目はどの fixture の答えも変えない。issue の画面 3 の文字どおりの形(prompt の行、字下げした `Read 2 files`、`⎿` の上限の行)は `⏺` を持たないので、今までどおり prompt のすぐ下の結果のまとまりとして読まれる |
| AC2 | Met(読んで確かめた) | `UsageLimitScreen` のコードは変わっていない。`usageLimitRetryingTurn` は `resumeScreenBlocks` に渡す引数だけが変わった(`usage_limit_screen.go:175`)。`usageLimitFixtureWants` の 5 行もそのまま |
| AC3 | Met(読んで確かめた) | cycle 1 の否定の行はすべて残る(下の AC4)。cycle 2 は否定を 4 行足し、どれも `UsageLimitScreen` の false も確かめる。返答の頭が prompt のすぐ下にあり、2 行の `⎿` の引用が上限で終わる形(917・922 行、cross-review の指摘)と、prompt が字下げした `⏺` の上限の行を写した形(931・936 行、self-review Run 2 の M1)である。resume だけの否定は Edges の 532・537 行(字下げした `⏺ API Error` を写した prompt) |
| AC4 | Met(diff で確かめた) | `git diff b061c0c9..HEAD -- '*_test.go'` で消えた行は 3 行だけ。`TestClassifyResumeScreen_UsageLimitOnTheLastResultLine` の doc のコメント 2 行と、`usage_limit_screen_test.go:254` の `resumeScreenBlocks` の呼び出し 1 行(引数を marker から driver に)である。期待を変えた行はない |
| AC5 | Met(手順 1 の文のずれは既知、V9) | rule の usage_limited の段落(2 写し、`cmp` で同一)は (c2) と頭の例を言い、cycle 2 のあとも正しい。仕様 FR-2 の 4 の 2026-11-22 の改訂は (c2) の 3 条件を言うが、54 行と 61 行が cycle 2 の手順 1 を言わない(V9、依頼で既知)。静的検証の中の check-sync は PASS(DRIFTED 0) |

計画の Verify plan の残り 2 点。「(c2) が API Error の扱いを変えないこと」は、`findResumeLimitResult` が変わっていないので (c2) については今も成り立つ。ただし手順 1 の新しい切れ目は、prompt の行のすぐ下(空行なし)の 1 桁目の `⏺ API Error` を独立したまとまりにし、(c) で stopped と読ませる(base では prompt_last の unknown)。(c2) の外の変化で、Edges の 522 行が対照として固定している(V18)。「プロンプトのまとまりを読まないこと」も成り立つ。`e9017d7f` のあと、prompt の字下げした続きの行は `⏺` で始まっても prompt のまとまりに残る(1045・1053 行)。

#### cycle 2 の直しの確かめ

| 指摘 | 直したコミット | 確かめたこと |
| --- | --- | --- |
| cross-review の ACTION_REQUIRED 1(prompt のすぐ下の返答の頭が、引用の `⎿` の行に `resultUnderPrompt` を付けさせる) | `1042ccae` | `resumeScreenBlocks` は prompt の行のあとの項目の頭で flush する(`resume_screens.go:743`)。そのあとの `⎿` の行は頭のまとまりに入るので、印は付かない。`TestResumeScreenBlocks_ItemHeadUnderAPrompt` の 1029 行がまとまりと印を、917・922 行が resume と見張りの答えを固定する。切れ目を外すと 917・922 行は頭のない許しで stopped になって落ちる(読んで確かめた)。肯定の側は 838 行(prompt のすぐ下の `⏺ Bash(cat notes.md)` の下の上限) |
| self-review Run 2 の M1(切れ目が字下げを見ない) | `e9017d7f` | 項目の頭の判定は、字下げを落とす前の行 `l` で行う(`resume_screens.go:743`、`resumeStartsWithAny(l, itemGlyphs)`)。`resumeScreenLines` は右の空白だけを落とす(677 行)ので、左の字下げは残る。Claude Code が項目の `⏺` を 1 桁目に描き(`docs/evidence/org-seat-resume-probe-2026-11-07.md` の 2 節)、prompt の折り返しの行を字下げする(`docs/evidence/org-seat-resume-live-2026-11-07.md` の 0 節)ことは、evidence の画面で確かめた。否定は切り方 2 行(1045・1053)、resume 2 行(532・537)、resume と見張り 2 行(931・936)で、対照は 522 行。`l` を `t` に戻すと 532 行は `⏺ API Error` で始まるまとまりになり、stopped で落ちる(読んで確かめた) |
| self-review Run 2 の L1(send のコメント) | `e9017d7f` | `send_screen.go` の画面モデルの 3(37-46 行)と `sendScreenRegion` の doc(658-665 行)が、send の prompt のまとまりは項目の頭では切れないと書く。`sendScreenRegion` は空行と `⎿` の行でだけ切り(674 行)、region が空で prompt の行があれば `sendScreenTurnCheck` は prompt_last の unknown を返す(647-648 行)。コメントの「send types nothing」はコードと合う |
| self-review Run 2 の L2(marker からの逆引き) | `e9017d7f` | `resumeScreenBlocks(lines []string, driver string)` が `resumeMarker(driver)` と `resumeScreenSentences[driver].ItemGlyphs` を引く(722-724 行)。`resumeMarkerItemGlyphs` はコードにもテストにも残っていない(計画の Progress の履歴の文だけ)。呼び出し元 5 か所(`resume_screens.go:454`・`:665`、`send_screen.go:611`、`usage_limit_screen.go:175`、`usage_limit_screen_test.go:254`)とテストの 1071 行が driver を渡す。本番の呼び出し元は、どれも先に `resumeScreenSentences[driver]` を確かめて表にない driver を返すので、逆引きを消しても答えは変わらない |
| self-review Run 2 の L3(記録 3 つ) | 未修正(cycle 2 の `/sync-docs`) | V9・V10・V11 |

send への影響はない。`sendScreenBusyCheck` が受け取る region は最後の prompt のまとまりより下の行で、footer は入力欄より下の行なので、どちらにも prompt の行がなく、新しい切れ目は働かない。

### Static analysis (cycle 2)

| Command | Result | Notes |
| --- | --- | --- |
| `./scripts/run-static-verify.sh`(changed scope) | pass(rc=0) | fixture の `.txt` を分類できず full にフォールバックし、選ばれた pack は golang だけ。hook の shellcheck と `sh -n`、settings.json の jq、check-sync(DRIFTED 0)、check-pipeline-sync、check-skill-sync(14 skill)、check-template-purity、tech-debt の plan 参照、gofmt(ok)、`go vet ./...`、golangci-lint(0 issues)、staticcheck(出力なし)、branch の secret scan(`38932c13..e9017d7f` clean) |
| `go vet ./internal/org/ ./internal/cli/` | pass | 変更のある 2 パッケージを単独で |
| `gofmt -l internal/org` | 出力なし | |
| `./scripts/plan-visual.sh digest <plan>` | `9683a080e69b` | 計画の `- Approved:` の行と一致(cycle 2 が変えたのは Progress だけ) |
| `cmp`(rule の 2 写し) | 同一 | `.claude/rules/ralph/agent-messaging.md` と `templates/base/` の写し |

### Documentation drift (cycle 2)

| Doc / contract | In sync? | Notes |
| --- | --- | --- |
| `resume_screens.go` の画面モデル(手順 1、手順 4 の (c2)、最後の段落)と、`resumeBlock.resultUnderPrompt`・`ItemGlyphs`・`resumeScreenBlocks` の doc | Yes | 1 桁目の項目の頭で切ること、字下げした行が prompt のまとまりに残ること、頭のまとまりの `⎿` の行に印が付かないことを言う。最後の段落(136-142 行)は、prompt の後ろの段落と `⎿` で始まる折り返しの行が普通のまとまりとして読まれる(base からの穴)と、主張を成り立つ範囲に狭めた |
| `send_screen.go` の画面モデルの 3 と `sendScreenRegion` の doc | Yes | 上の表の L1 |
| 仕様 `docs/specs/2026-11-07-org-seat-resume.md` FR-2 の 4 | No(V9、既知) | 54 行と 61 行 |
| `.claude/rules/ralph/agent-messaging.md`(2 写し)の usage_limited の段落 | Yes | 「at the start of the last reply, result, or error above the input box」と「but not under a reply's own text」は、cycle 2 の手順 1 のあとも正しい |
| `docs/specs/2026-08-01-org-runtime.md` 89 行 | Yes | `1dda5f7f` が足した 2026-11-22 の追記は (c2) の画面を言い、cycle 2 のあとも正しい |
| `/director` skill の resume の項、`ralph org resume --help`、`ralph org status --live` | Yes / 対象外 | Run 1 と同じ。status --live は `ClassifySendScreen` を使い、上の「send への影響はない」のとおり変わらない |
| `docs/tech-debt/README.md` | No(V10〜V17、V19) | 下の Findings |
| 計画の Progress | 一部ずれ(V20) | |

### Findings (cycle 2)

番号は Run 1 の V8 から続ける。V9〜V11 は依頼で既知とされたずれで、記録だけにした。台帳の行は 1 つ目のセルの書き出しで探せるように書き、行番号は HEAD のもの。

| ID | Severity | Finding | Evidence | Hand-off |
| --- | --- | --- | --- | --- |
| V9 | LOW(既知) | 仕様 54 行(2026-11-07 の改訂)は、まとまりの区切りを空行、prompt の行、prompt のあとの最初の `⎿` の行とし、`1042ccae`・`e9017d7f` が足した「prompt のあとの、1 桁目が `⏺` の行(項目の頭)」を言わない。61 行の「prompt の行のあとの最初の `⎿` の行で始まったもの」は、あいだに項目の頭がないことを言わない | 仕様 54・61 行、`resume_screens.go:30-47`・`:743` | `/sync-docs`: 54 行に 1 文(1 桁目の `⏺` の行は項目の頭で、そこから新しいまとまりになる。字下げした `⏺` の行は prompt のまとまりに残る)。61 行に「あいだに項目の頭がない」を足す。V18 の帰結も同じ文に入れられる |
| V10 | LOW(既知) | 台帳 570 行(「`internal/org/resume_screens.go` (705 → 902 lines)」で始まる)は「705 → 902」「781 → 1,119」と「d470a77a, the last Go commit of the branch」と書く。HEAD では 935 行と 1,261 行で、最後の Go のコミットは `e9017d7f`。この行の Trigger「The next change to either file」も `1042ccae`・`e9017d7f` が満たしたが、分割はしていない | `wc -l`(evidence)、台帳 570 行 | `/sync-docs`: 数とコミットを直し、cycle 2 で Trigger を満たして払っていないことを足す |
| V11 | LOW(既知) | 台帳 572 行(「Known gap of fix/resume-limit-later-line (#373, self-review L1」で始まる)の Trigger「the next change to `resumeScreenBlocks` or `findResumeLimitResult`」を、`1042ccae`・`e9017d7f` が `resumeScreenBlocks` を変えて満たした。繰り越しの注がない | 台帳 572 行の 4 つ目のセル | `/sync-docs`: 繰り越しの注。`e9017d7f` は生の行を読むが、572 行が求める「`resumeBlock.lines` に字下げを残す」は払っていない |
| V12 | LOW | 同じ 572 行の 1 つ目のセルの「`resumeScreenBlocks` trims every line before the block rules read it, so the deeper indent that sets a nested line apart is lost」は、`e9017d7f` のあと一部が違う。手順 1 の項目の頭の規則は、字下げを落とす前の行を読む(`:743`)。まとまりに残る `lines` は今も字下げを落とした行(`:749`)なので、(c2) が入れ子の `⎿` の行を見分けられないことは変わらない | `resume_screens.go:735`・`:743`・`:749` | `/sync-docs`: 「the blocks keep each line trimmed (`resumeBlock.lines`; step 1 reads the raw line only to tell a first-column item head, e9017d7f)」のように直す |
| V13 | LOW | `internal/org/send_screen.go` が `e9017d7f`(コメント 2 か所)で 798 行から 804 行になり、800 行の目安を越えた。台帳 524 行(「Known gaps of fix/send-codex-paste (#326)」で始まる)の (h) は「`internal/org/send_screen.go` is 798 lines ... near the 800-line guide」と書き、その Trigger「(h) The next change to `send_screen.go`: move the paste rules ...」を `e9017d7f` が満たしたが、払っておらず注もない(新しい規則は足していないので「before the next rule is added」には反しない)。同じ変更で、488 行(「Seven small findings of fix/send-screen-fallback (#308)」で始まる)の「(L2, L7, L9, L10) the next change to `send_fallback.go`, `send_screen.go`, or `status_live.go`」も満たした。cycle 1 は `send_screen.go` を変えていないので、どちらの行にもこの branch の注はない | `wc -l`(base 798、HEAD 804)、台帳 524 行の (h)、488 行の Trigger | `/sync-docs`: 524 行の (h) の数を 804 にし、800 を越えたことと繰り越しの注を足す。488 行に繰り越しの注(L2・L7・L9・L10 はどれも払っていない) |
| V14 | LOW | 台帳 544 行(「Three LOW findings of the cycle 1 self-review of fix/watch-usage-limit-retry (#353)」で始まる)の 2026-11-22 の注「(a) and (b) were not met: `usageLimitRetryingTurn`, `usageLimitResetsIn`, `usageLimitResetTime`, and `resumeTargetText` are as they were」は、HEAD では違う。`e9017d7f` は `usageLimitRetryingTurn` の本文(`resumeScreenBlocks` の呼び出し、`usage_limit_screen.go:175`)を変えた。`1042ccae`・`e9017d7f` の手順 1 は、それが読むまとまりも変えた(prompt のすぐ下の 1 桁目の `⏺` の行は、prompt のまとまりではなく `usageLimitTurnBlocks` の今の手番に入る)。(a) の Trigger「The next change to `usageLimitRetryingTurn`」を満たし、改名は払っていない | `usage_limit_screen.go:169-175`・`:221-227`、台帳 544 行 | `/sync-docs`: 注の (a) を「met by e9017d7f (its call of `resumeScreenBlocks` only), not paid」に直す |
| V15 | LOW | 台帳 574 行(「Test gaps of fix/resume-limit-later-line (#373)」で始まる)の (a) の Trigger「the next change ... to `TestClassifyResumeScreen_UsageLimitOnTheLastResultLine`」を、`1042ccae`(838・917・922 行)と `e9017d7f`(931・936 行)が満たした。codex の probe の行(prompt の行と `⎿` の 2 行)は足していないので、M4b はおそらく今も生き残る(未確認)。(b) の「The next change to step 4 (c2) in code」も、`1042ccae` が (c2) の頭のない許しの条件(`resultUnderPrompt` の印)と画面モデルの (c2) の段落を変えたので、中身は満たした。(b) が予告したとおり、仕様の文がコードに遅れた(V9) | 台帳 574 行、`resume_screens_test.go` の上の 5 行 | `/sync-docs`: 574 行に繰り越しの注 |
| V16 | LOW | `1042ccae`・`e9017d7f` の手順 1 の切れ目は、台帳 462・483・567 行の Trigger をもう一度満たした。462 行(「Seven mutants of `ralph org resume` survive」)は「the next change to `ClassifyResumeScreen`」で、読むまとまりが変わり、prompt のすぐ下の 1 桁目の `⏺ API Error` が stopped になった。483 行(「A hook that was changed after codex approved it」)は「the next change to the screen rules of `internal/org/resume_screens.go`」で、新しい画面の規則である。567 行(「A claude reviewer seat runs a feature org's gate with `run_in_background`」)は「the next change to how the watch reads a seat's screen」で、prompt のすぐ下の 1 桁目の `⏺ You've hit …` も上限と読まれ、字下げした引用は読まれなくなった。3 行の 2026-11-22 の注は (c2) と `d470a77a` だけを言う。どの行の項目も払っていない | 台帳 462・483・567 行の 4 つ目のセル | `/sync-docs`: 3 行の注に cycle 2 の 1 文を足す |
| V17 | INFO | 台帳 490・517・542 行の 2026-11-22 の注は、`d470a77a` を「the last Go commit of the branch」と呼ぶ。HEAD の最後の Go のコミットは `e9017d7f`。数(893、1013、1013)は HEAD でも同じ。542 行の「the #353 block is not moved」も正しいが、`e9017d7f` は塊の中の補助 `usageLimitRetryingLine` の 1 行(254 行)を変えた | `wc -l`(evidence)、台帳 490・517・542 行 | 任意: 「at e9017d7f, the last Go commit」に直す |
| V18 | INFO | 計画の Verify plan の「(c2) が API Error の扱いを変えないこと」は (c2) については成り立つが、cycle 2 の手順 1 は (c2) の外で答えを 1 つ変えた。prompt の行のすぐ下(空行なし)の 1 桁目の `⏺ API Error: …` は、base では prompt のまとまりに入って prompt_last の unknown だった。今は独立したまとまりになり、(c) で stopped(resume は続行の文を打つ)。同じ位置の `⏺ You've hit your weekly limit …` は見張りが上限と読み、`usageLimitRetryingTurn` の今の手番にも入る。Claude Code が描いた項目なので向きは正しく、空行のある実機の描き方(probe の evidence の 2 節)と答えがそろう。計画の Progress の 105 行(対照の 1 行)と self-review Run 2 の P5 が記録し、Edges の 522 行が固定するが、仕様は言わない | `resume_screens_test.go:522`、self-review Run 2 の probe P5 | V9 の直しで仕様に 1 文 |
| V19 | INFO | self-review Run 2 の Tech debt の 2 行目(base からの穴: prompt の空行より下の段落 P13 と、prompt の折り返しの行のうち `⎿` で始まる行 P4 は、prompt のまとまりに入らない)は台帳にない。`e9017d7f` は画面モデルの最後の段落(`resume_screens.go:136-142`)でこれを言うようになった。仕様 54 行の「利用者の prompt のまとまりは、…どの判定にも使わない」はまとまりについては正しいが、利用者が打った文がいつも数えられないとまでは言えない | self-review Run 2 の Tech debt、`resume_screens.go:136-142` | `/sync-docs`: self-review の行案どおり台帳に行を足す。仕様 54 行の直し(V9)に 1 文足してもよい |
| V20 | INFO | 計画の Progress の Implementation started の下に、`1042ccae`(cross-review の ACTION_REQUIRED 1 の直し)の行がない。106 行が self-review Run 2 の HEAD として名を挙げるだけ | 計画 102-106 行 | `/sync-docs`: 1 行足す(Progress は digest に入らない) |

### Run 1 の所見の行き先

| ID | 行き先 | HEAD での状態 |
| --- | --- | --- |
| V1 | `1dda5f7f` が台帳 462・483 行に繰り越しの注を足した | 注は正しい。cycle 2 でもう一度満たした(V16) |
| V2 | `1dda5f7f` が仕様 61 行の条件を直した | `1042ccae` のあと、あいだの項目の頭を言わない(V9) |
| V3 | `1dda5f7f` が仕様 59 行に、つないだ文での照合を足した | 正しい |
| V4 | `1dda5f7f` が仕様 60 行に P2 の文を、台帳に 571〜573 行を足した | 正しい(572 行の一部は V12)。`/pr` の Known gaps はまだ |
| V5 | `1dda5f7f` が rule の頭の例を `such as` にし、`Read 2 files` と `⏺ Monitor "..."` を足した(2 写し) | 正しい |
| V6 | `1dda5f7f` が計画 102 行の名前と 104 行の `d470a77a` を直した | 正しい |
| V7 | `1dda5f7f` が `docs/specs/2026-08-01-org-runtime.md` 89 行に 2026-11-22 の追記を足した | 正しい |
| V8 | `1dda5f7f` が AC1〜AC5 に `[x]` を付けた | 正しい |

### `/sync-docs` への引き継ぎ (cycle 2)

1. 仕様 `docs/specs/2026-11-07-org-seat-resume.md` の 54・61 行(V9)。V18 の帰結と、任意で V19 の 1 文
2. 台帳: 570 行の数とコミットと Trigger(V10)、572 行の注と 1 つ目のセル(V11・V12)、524 行の (h) と 488 行(V13)、544 行(V14)、574 行(V15)、462・483・567 行(V16)、任意で 490・517・542 行(V17)、新しい行(V19)
3. 計画の Progress(V20)

### Observational checks (cycle 2)

- 実機の画面は見ていない。1 桁目の `⏺` と字下げした prompt の行の違いは、2026-11-07 の 2 つの evidence(`org-seat-resume-probe-2026-11-07.md` の 2 節、`org-seat-resume-live-2026-11-07.md` の 0 節)の画面で確かめた
- 振る舞いはコードを読んでたどった。テストも probe も走らせていない

### Coverage gaps (cycle 2)

- テストは走らせていない。cycle 2 で足した 13 行(`TestResumeScreenBlocks_ItemHeadUnderAPrompt` の 5 行、`TestClassifyResumeScreen_UsageLimitOnTheLastResultLine` の 5 行、`TestClassifyResumeScreen_Edges` の 3 行)が通るか、上の 2 つの変異で落ちるかは `/test` で確かめる
- prompt の行のすぐ下(空行なし)の返答の頭の下に `⎿` の行が 1 行だけある形(self-review Run 2 の P6 は上限、P10 は `⎿  API Error`)は、どの行も固定していない。`1042ccae` で base の stopped から unknown に変わった形で、(c) の経路を通る。固定しているのは 2 行の `⎿` の引用(917・922 行、(c2) の経路)だけである。手順 1 を「頭のあとの `⎿` の行に印を付けないだけ」に変える変異は、917・922 行を通り、522 行で落ちる(読んで確かめた)。足すなら 922 行の隣に 2 行
- 手順 1 の切れ目は `ClassifyResumeScreen` の (a)・(b) にも効く(prompt のすぐ下の 1 桁目の項目にある enter-only や waiting の文)。その形の行はない
- 決定的な検査で足りないものは Run 1 の Coverage gaps と同じ(仕様の文を固定するテストがない。台帳 574 行の (b))。cycle 2 でも仕様の 2 行がコードに遅れた

## Verdict

- Result: pass(cycle 2。LOW 8(うち依頼で既知の 3)、INFO 4。どれも merge を止めない文書と台帳のずれで、cycle 2 の `/sync-docs` で直せる)
- Verified: `1042ccae` と `e9017d7f` が cross-review の ACTION_REQUIRED 1 と self-review Run 2 の M1・L1・L2 を直したこと(コードと、それを固定するテストの行を読んだ)。AC1〜AC5 が cycle 2 のあとも満たされること(fixture の答えが変わらないことは走査で、テストの期待が変わらないことは diff で確かめた)。send の判定が変わらないこと。静的検証 rc=0、`go vet` と `gofmt`、rule の 2 写しの一致、計画の digest の一致
- Partially verified: AC5(仕様 54・61 行が手順 1 の新しい切れ目を言わない。V9)
- Not verified: テストの実行と変異(`/test` の担当)、実機の Claude Code の画面での手順 1 と (c2) の判定、`ralph org resume` と見張りの実機での動き

---

## Run 1 (cycle 1、履歴)

以下は cycle 1(HEAD `d470a77a`)の report で、見出しを 1 段下げたほかは元のまま。今の判定は上の `## Verdict`。

- Date: 2026-11-22
- Plan: `docs/plans/active/2026-11-22-resume-limit-later-line.md`(issue #373。承認の digest `9683a080e69b` は HEAD の本文から計算した値と一致)
- Verifier: verifier サブエージェント(`/verify`)。パイプライン 1 回目(`cycle-count.json` は cycle 1)
- Scope: origin/main `b263b1f3` から HEAD `d470a77a` までの `git diff origin/main...HEAD`(17 ファイル、+947/-41)。self-review の対象(`444adf11` まで)に、そのあとの直し `d470a77a`(M1・L2・L3・L4)を加えた範囲。テストは実行していない(`/test` の担当)
- Evidence: `docs/evidence/verify-2026-11-22-resume-limit-later-line.log`(git の管理外。静的検証の全出力、`go vet`、digest、rule の 2 写しの `cmp`、base と HEAD の `wc -l`、テストのファイルで消えた行、改名前の名前の grep、仕様と rule の該当文、台帳の Trigger のセル)

### Spec compliance

行番号は HEAD `d470a77a` のもの。

| Acceptance criterion | Status | Evidence |
| --- | --- | --- |
| AC1: issue の画面 1・2 を入力欄と footer つきの fixture にし、claude で stopped、`Line` が `⎿  You've hit your 5-hour limit` で始まる。プロンプトの下に空行がある画面 3、知らせの行が折り返した画面 2、`⏺ Monitor "…"` の知らせの下の上限も同じ | Met(読んで確かめた。実行は `/test`) | fixture 5 件(`internal/org/testdata/resume/claude-5h-limit-after-{shell-command,background-notice,background-notice-wrapped,read-under-prompt}.txt`、`claude-weekly-limit-after-monitor.txt`)を `TestClassifyResumeScreen_Fixtures` の 5 行が stopped と `Line` で固定する。issue の文字どおりの 3 画面(空行なし)は `TestClassifyResumeScreen_UsageLimitOnTheLastResultLine` の最初の 3 行が固定する。5 件を手でたどった: 画面 1 は `Ran 1 shell command` が `Bare` の頭、画面 2 と折り返し版は `⏺ Background command "` の頭(折り返しの続きの行は頭でも結果の行でもないので飛ばす)、画面 3 は `Read 2 files` が頭、Monitor は `⏺ Monitor "` の頭。CLI でも `TestOrgResume_DryRunResumesASeatStoppedOnALimitUnderAToolOrANotice`(`internal/cli/org_resume_test.go:426`)が `would_resume` と上限の行を固定する |
| AC2: 同じ fixture で `UsageLimitScreen` が上限と読み、リセットの時刻を読める | Met(読んで確かめた) | `usageLimitFixtureWants` に 5 行を足し(真の行は 5 → 10、fixture 86 件に 86 行)、`line` と `has`(`resets 4:40am`、`(Asia/Tokyo)`)を固定する。`UsageLimitScreen`(`usage_limit_screen.go:93`)は stopped のとき `matchedStopSentence(res.Line)` で文を決め、`UsageLimit` に入る文なら `usageLimitLine(res.Line)` を返す。(c2) の `Line` は `⎿`、空白、上限の文の順で始まるので、`findResumeStop` の glyph の規則で一致する。見張りは `readUsageLimitScreen`(`watch_usage_limit.go:128`)で同じ `UsageLimitScreen` を呼ぶ。リセットの時刻は行の中の文字として運ばれる(`usageLimitResetsIn` は 3 つ目の読み方 `usageLimitRetryingTurn` だけが使い、(c2) の経路は通らない) |
| AC3: 誤検知の画面は stopped にならない(列挙の 9 形と codex) | Met(読んで確かめた) | 表のテストの AC3 の節に、最後の `⎿` の行が API Error、上限の下の `⎿  Interrupted`、上限の下の返信のまとまり、返信の文の中の引用、印のない行の上限、古い上限の下の普通の返信、`⏺ The seat showed:` のすぐ下の上限(状態行のある形とない形)、`Searched for 1 pattern` の下の上限、codex の 3 形がある。どれも `resumeReasonNoStop` と `UsageLimitScreen` の false を期待する。ほかに self-review の M1 の 2 形(空行のあとの 2 行の `⎿` の引用、868・873 行)、頭のない普通の行、`⏺` のあとに空白がない頭、`⎿` のあとに空白がない上限も否定で固定している |
| AC4: 既存のテストは期待を直す 2 件を除いて変わらない | Met(diff で確かめた) | テストの 4 ファイルで消えた行は、`resume_screens_test.go` の「only the first ⎿ line under a prompt starts a block, the next ones join it」のコメント 2 行と画面 1 行、`usage_limit_screen_test.go` のコメント「Only the five usage limit screens are true」と `resume: ResumeScreenUnknown` の 1 行だけ。前者は 2 つ目の `⎿` を API Error にして「まとまりがつながる」意図を残し、上限の形は別の行(509 行、stopped)に移した。後者は #353 の計画の AC4(今の stopped の読み方の対象でなければ上限と読まない)と矛盾しない: その画面は (c2) で stopped になり 1 つ目の読み方が読む。3 つ目の読み方だけでは今も読まないことを `TestUsageLimitScreen_Issue373LimitUnderAMonitorNoticeIsAStop`(606 行)が固定する。`send_screen_test.go` と `org_resume_test.go` は追加だけ |
| AC5: 仕様の画面モデルと rule の usage_limited の段落が (c2) を言う。`check-sync.sh` が通る | Met(文のずれは V2・V3) | 仕様 FR-2 の 4 に 2026-11-22 の改訂(58-63 行)と Open questions の補足(205 行)がある。rule の usage_limited の段落(2 写し、`cmp` で同一)が (c2) を言う。静的検証の中の check-sync は PASS(DRIFTED 0)。ただし仕様の 59・61 行は `d470a77a` より前の文のまま(V2・V3) |

計画の Verify plan の残り 2 点: (c2) は API Error の扱いを変えない(`findResumeLimitResult` は `UsageLimit` の文だけを照合し、最後の `⎿` の行が API Error なら unknown。表のテストの AC3 の 1 行目)。プロンプトのまとまりを読まない(`findResumeLimitResult` は `block.prompt` で false を返す、`resume_screens.go:796` の直後)。

#### self-review の直し(`d470a77a`)の確かめ

| ID | 直ったか | 確かめたこと |
| --- | --- | --- |
| M1 | 直った | `resumeBlock` に `resultUnderPrompt`(683 行)を足し、`resumeScreenBlocks` が prompt のあとの最初の `⎿` の行で切ったまとまりにだけ立てる(714 行。空行の flush で落ちる)。頭のないときの許しは `return block.resultUnderPrompt`(837 行)。否定の 2 行(868・873 行)が固定し、元の `resumeStartsWithAny(above[0], …)` に戻すとこの 2 行が stopped になる(読んで確かめた)。肯定の側は Edges の 509 行。区別できない P2 の形(空行のあと、まとめた行で始まる引用)は 884 行で stopped と固定し、コメントで known gap と書いた |
| L2 | 直った | `findResumeLimitResult` は上限の行とその下の行をつないだ文(803 行、`Line` と同じ文)で glyph のあとを照合する。804 行のテスト(上限の文が `You've hit your` と `5-hour limit · resets 4:40am` に割れる)が固定する。つなぐ行は、最後の結果の行より下にある、頭でも結果の行でもない行だけ(下に頭があれば先に false を返す) |
| L3 | 直った | `ResultGlyphs` の doc に「Each is also one of StopGlyphs」を足し、`TestResumeResultGlyphs_AreStopGlyphs`(`resume_screens_test.go:1102`)が glyph の包含と `matchedStopSentence` の一致を driver ごとに確かめる |
| L4 | 直った | `ResultHeads` → `ToolOrNoticeHeads`、`resumeResultHead` → `resumeToolOrNoticeHead`、`resumeResultOfTool` → `resumeResultUnderToolOrNotice`、戻り値 `tool` → `toolOrNotice`。古い名前はコードとテストに残っていない。残りは計画の Progress の S1 の行だけ(V6) |
| M2・L1 | 予定どおり未修正 | 台帳の編集と Known gaps は `/sync-docs` と `/pr` に回す(下の「`/sync-docs` への引き継ぎ」) |

### Static analysis

| Command | Result | Notes |
| --- | --- | --- |
| `RALPH_VERIFY_SCOPE=full ./scripts/run-static-verify.sh` | pass(rc=0) | full scope で選ばれた pack は golang だけ(changed scope と同じ)。hook の shellcheck と `sh -n`、settings.json の jq、Codex の hook の 3 つの guard、check-sync(DRIFTED 0)、check-pipeline-sync、check-skill-sync(14 skill)、check-template-purity、tech-debt の plan 参照、gofmt(ok)、`go vet ./...`、golangci-lint(0 issues)、staticcheck(出力なし)、branch の secret scan(`b263b1f3..d470a77a` clean)がすべて通った |
| `go vet ./internal/org/ ./internal/cli/` | pass | 上の `go vet ./...` と重なるが、変更のある 2 パッケージを単独で確かめた |
| `./scripts/plan-visual.sh digest <plan>` | `9683a080e69b` | 計画の `- Approved:` の行と一致 |
| `cmp`(rule の 2 写し) | 同一 | `.claude/rules/ralph/agent-messaging.md` と `templates/base/` の写し |

### Documentation drift

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

### Findings

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

### `/sync-docs` への引き継ぎ

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

### Observational checks

- 実機の画面は見ていない。fixture は issue #373 の画面の文をそのまま使い、ブロックのあいだに空行を足したもの(既存の Claude Code 2.1.316 の取り込みに合わせた)。issue のとおり空行のない形は表のテストの最初の 3 行が持つ。空行を足した形が実機と同じかは確かめていない
- 振る舞いはコードを読んでたどった。テストも probe も走らせていない

### Coverage gaps

- テストは走らせていない。新しい表のテスト、fixture の 5 行、CLI のテスト、期待を直した 2 件が通るかは `/test` で確かめる
- `d470a77a` の直しの変異(`resultUnderPrompt` を前の `above[0]` の規則に戻す、照合を 1 行に戻す)が落ちることは、どの行が落とすかを読んで確かめただけ。計画の Progress の変異 15 個は `d470a77a` の前の S1 のもの
- 見ていない頭(`Searched for …`、MCP の `(MCP)(`、`Web Search(`)の下の上限と、上限の文が割れた行のうち頭が見つからない形は unknown のまま(計画の Risks の 1 つ目)。実機の画面がないので、正しい頭の一覧かは分からない
- 決定的な検査で足りないもの: 仕様の日付つきの改訂の文を固定するテストはなく、V2・V3 はそのため `d470a77a` のあとも残った。足すなら、仕様 61 行の文と前文の (c2) の段落が同じ条件(「the first ⎿ line after a prompt line」)を言うことを `TestClassifyResumeScreen_*` の近くで 1 つの fragment で確かめるのが最小

### Verdict

- Result: pass(LOW 4、INFO 4。どれも merge を止めない)
- Verified: AC1〜AC5 をコード、fixture、テストの行、仕様と rule の文と照らした。self-review の M1・L2・L3・L4 の直しが入ったこと。見張りの `UsageLimitScreen` が (c2) の `Line` を `matchedStopSentence` で読むこと。静的検証 rc=0、rule の 2 写しの一致、計画の digest の一致
- Partially verified: AC5(仕様の文が `d470a77a` の直しの 2 点を言わない。V2・V3)
- Not verified: テストの実行(`/test` の担当)、実機の Claude Code の画面での (c2) の判定、`ralph org resume` と見張りの実機での動き
