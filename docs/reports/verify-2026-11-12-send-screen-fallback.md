# Verify report: send-screen-fallback

- Date: 2026-11-13 JST(UTC では 2026-11-12。ファイル名は cycle 1 のまま)
- Plan: `docs/plans/active/2026-11-12-send-screen-fallback.md`(issue #308。`- Approved:` の digest `1db244296e07` は `./scripts/plan-visual.sh digest` で出し直した値と一致した。S8 で D3 に番号付きの行の扱いを足し、承認をやり直した版)
- Verifier: verifier サブエージェント(`/verify`)。パイプライン cycle 2(`cycle-count.json` は cycle 2)。cycle 1 の report(67b6a551、pass)を上書きした。cycle 1 の ID(V1〜V3、I1〜I3)は番号を変えず、下の「cycle 1 の drift の行方」に状況を書いた。この run の新しい ID は V4〜V6 と I4〜I5
- Scope: `git diff origin/main...HEAD`(origin/main `0936aaea`。branch は 39004c89 でこれを取り込んだので merge-base も `0936aaea`。HEAD `c93f854b`。78 ファイル、6902 行追加、212 行削除)。重く見たのは cycle 1 の verify(67b6a551)のあとのコミット: 88742506(CLI から画面の道を通すテスト)、f24290b5(cross-review の ACTION_REQUIRED #1 の直しと M02 のテスト)、cea96ea4(self-review L11 の直し)、39004c89(origin/main の取り込み)、文書の 181ecbb3・1bc2f138・bcb43faa と report のコミット。静的解析は `./scripts/run-static-verify.sh` の既定の範囲で、testdata の `.txt` がどの言語にも分けられないので full に落ちた(選ばれた pack は golang だけ)。テストは走らせていない(`/test` の範囲)
- Evidence: `docs/evidence/verify-2026-11-12-send-screen-fallback.log`(git が無視する場所)。cycle 1 の 1090 行はそのまま残し、1091 行目から「Cycle 2」の節を足した。静的解析の全出力、digest、fixture と表のキーの比べ、skill の写しのハッシュ、AGENTS.md と README.md の origin/main との単語の差、binary の probe 6 本の出力、偽の herdr の台本、tech-debt の行の読み取りを入れた
- Insight event: `docs/insights/events/2026-11-12-send-screen-fallback.jsonl` に verify の 1 行(cycle 2、pass、low 3)を足した。指示どおり、この report のコミットには入れていない

## cycle 1 からの変化

| Commit | 中身 | verify の見方 |
| --- | --- | --- |
| 88742506 | `internal/cli/org_send_screen_fallback_test.go` に 3 本(画面の道で送る、busy で打たない、`--no-screen-fallback`)。`org_resume_test.go` の stub に `keeps-working` と `send-keys` を足した | cycle 1 の Coverage gaps の「CLI のテストで画面の道を通すものが無い」を埋めた。AC1、AC2、AC5 の CLI 側の証拠に加えた |
| f24290b5 | `sendTypedInputBox` を足し、Enter の前の確かめだけが、打った文の一部である番号付きの印の行を入力欄とみなす。`findResumeInputBox` を `resumeMarkerLineBox` と `resumeInputBoxShape` に分け、`ResumeInputHolds` の比べを `holdsHead` に出した。M02 を殺すテスト、番号付きの合成 fixture 2 つ | AC1、AC4、AC7、AC8、AC10 を読み直した。下の表と probe |
| cea96ea4 | `sendTypedInputBox` が `resumeMarkerLineBox` の `bool` を見る(`!ok` なら箱なし) | 今の入力では届かない枝。挙動は変わらない |
| 39004c89 | origin/main(#321)の取り込み。AGENTS.md と tech-debt が競合 | AGENTS.md と README.md は、origin/main に対してこの branch の追加だけが差(evidence の単語の差)。org skill の 4 つの写しは同じハッシュで、`via=screen` と `status --live` の行は残る。`internal/cli/org.go` は取り込みで 2 行ずれただけ |
| 181ecbb3、1bc2f138 | sync-docs: 仕様の改訂の項、tech-debt の 14 行の注記と 2 行、plan の印 | cycle 1 の V1〜V3 を受けた(下の表) |

## Spec compliance

「満たす」は、コードとテストのソースを読んで AC を満たすと判断したという意味。テストが通るかは `/test` が確かめる。「観察」は、build した binary を `PATH` の先頭に置いた偽の herdr で動かした結果。実機の herdr、claude、codex は 1 度も呼んでいない。`send_fallback.go`、`verbs.go`、`status_live.go` は cycle 1 の verify のあと変わっていないので、そこの行番号は cycle 1 と同じ。

| Acceptance criterion | Status | Evidence |
| --- | --- | --- |
| AC1 herdr が working のまま、画面が入力待ちの座席へ `--timeout-ms` のうちに打ち、`sent` に `via=screen`(番号付きの手順の TASK を含む) | 満たす。実機は未確認 | 道そのものは cycle 1 と同じ(`sendWaitForSeat`、`send_fallback.go:126`。`verbs.go:642` が `via=screen herdr_state=` を Details の頭に付ける)。番号付きの手順の場合: 入力欄の 1 行目が `❯ 7. Run the checks…` か `› 2. Read the files…   ↑` になる画面で、`sendInputHolds`(`send_screen.go:269`)は `sendTypedInputBox`(`:315`)で箱を見つけ、規則 2(文の終わり)で通る。テスト: `TestOrgSendScreen_MultiLineMessageTallerThanTheBox_EnterPressed` に番号付きの 2 行(`send_fallback_test.go:783`、`:785`)。CLI のテスト `TestOrgSendScreenPath_ScreenWaitsForInput_TypesAndRecordsViaScreen`(88742506)は本物の `ralph org send` のプロセスで `via=screen herdr_state=working` を比べる。観察(probe claude-head、codex-head): `sendNumberedStepsMsg` の本文で、HEAD の binary は終了コード 0、20 秒、Enter を 1 回押し、`sent` は `via=screen herdr_state=working TYPE: QUESTION…` で `submit_unconfirmed` が付かなかった。直す前の binary(0efeef81、コードは 88742506 と同じ)は同じ台本で文を打ったが Enter を押さず、終了コード 1 で `the typed text was not found in the input box` を返した(probe claude-base、codex-base)。cross-review の指摘はこの binary で再現し、HEAD で消えた |
| AC2 動作中の行、選択の画面、終わりの行も起動直後の形も無い、判断できない、2 回で違う、のどれかでは打たず、エラーに herdr の状態・画面の判定と理由・次の手 | 満たす | コードは cycle 1 と同じ(`checkSendScreen`、`sendReadIdleScreen`、`sendScreenWaitErr`、`sendScreenNextSteps`)。打つ前の `ClassifySendScreen`(`send_screen.go:190`)は f24290b5 で変わっておらず、番号付きの印の行を今までどおり選択の画面と読む(新しい 2 つの fixture の期待は `SendScreenChoice`)。テストは cycle 1 の `TestOrgSendScreen_ScreenNotWaiting_TypesNothingAndSaysWhy` の 7 行に、CLI の `TestOrgSendScreenPath_ScreenBusy_TypesNothingAndSaysWhy`(88742506。エラー全文の `screen check:` の部分、`sent` が無いこと)が加わった |
| AC3 ロックの下の読み直しで入力待ちでなければ打たない | 満たす | 変更なし(`sendScreenUnderLock`、`send_fallback.go:274`、`verbs.go:546`)。テスト `TestOrgSendScreen_ScreenChangedUnderThePaneLock_TypesNothing`。binary では観察していない |
| AC4 Enter の前に文が入力欄に無ければ Enter を押さず、打ったが送っていないと返す(番号付きの手順を含む) | 満たす | 番号付きの印の行を入力欄とみなすのは `sendTypedInputBox` だけで、条件は 2 つとも要る。(1)印の行の文(空白を落とし、codex は末尾の `↑` も落とす)が打った文の一部。(2)まわりが入力欄の形(claude は印の行の真上と下に罫線、`resumeInputBoxShape`。codex は footer、`sendCodexHasFooter`)。計画の D3 の文と合う。テスト: 通る側は `TestSendInputHolds_NumberedStepOnTheMarkerLine`(`-input-typed-numbered-` の fixture をすべて表に入れることも確かめる)。断る側は `TestSendInputHolds_NumberedMarkerLineRefuses`(本物の選択の画面 4 つを 2 通りの文で、打った文に無い番号付きの行、罫線や footer を外した画面)。観察(probe claude-dialog、codex-dialog): HEAD の binary で、打ったあとの画面を `claude-folder-trust.txt`、`codex-hooks-review.txt` にすると、Enter を押さずに終了コード 1 と `(text typed but not submitted)` を返し、`sent` は無かった |
| AC10 画面の道で Enter のあと herdr が working のまま文が入力欄に残れば `submit_unconfirmed=true`。入力欄が空になったか動作中の行が出たときだけ確かめ済み | 満たす | `SendScreenSubmitted`(`send_screen.go:222`)のコードは変わらず、`findResumeInputBox` を使い続ける。番号付きの行が残った箱は選択の画面と読まれ、送れたとはならない(D3 の「確かめられない側に倒す」)。テスト: `TestSendScreenSubmitted` に番号付きの 4 行(fixture 2、動作中の行つきの合成 2)、`TestSendInputHolds_NumberedStepOnTheMarkerLine` の `SendScreenSubmitted = false`。観察: probe claude-head と codex-head は、Enter のあとの画面が空の入力欄なので `submit_unconfirmed` を付けなかった(`submit_unconfirmed=true ` は Details の本文の前に入る、`verbs.go:642-645`) |
| AC11 確かめの手前で時間切れになり、画面が入力待ちでない座席が `--timeout-ms` のうちに idle か done になれば、今までの道で届く | 満たす。計画の例の尺では動かしていない | コードは cycle 1 と同じ。f24290b5 が D1 の 2 回の読みの間を固定するテスト `TestOrgSendScreen_TwoReadsAreTheReadGapApart` を足した(`sendFallbackHerdr.readTimes` で 1 回目と 2 回目の `PaneRead` の時刻の差が `SendScreenReadGap` 以上)。cycle 1 の /test が残した M02 にあたる。待ちを 0 にする変異で落ちるかは読んだだけで、走らせていない |
| AC5 `--no-screen-fallback` では herdr の待ちだけで、時間切れは今の文。idle か done の道は今と同じ | 満たす | コードは cycle 1 と同じ(`sendScreenReserve`、`send_fallback.go:101`。フラグは `internal/cli/org.go:1519`、取り込みで 2 行ずれた)。CLI のテスト `TestOrgSendScreenPath_NoScreenFallback_ReadsNoScreen`(88742506)が、herdr の待ちが `--timeout-ms` を全部使い、`agent get` も画面も読まないことを比べる。cycle 1 の origin/main との binary の比べは、この部分のコードが動いていないので繰り返していない |
| AC6 `status --live` が herdr の状態と画面の判定を出し、食い違いに印と `herdr agent explain` の案内。`--live` なしの出力は今と同じ | 満たす | 変更なし(`LiveSeatStates`、`status_live.go:68`。`printLiveMismatches` は `internal/cli/org.go:2186`、取り込みで 2 行ずれた)。`liveScreen` は `ClassifySendScreen` を呼ぶので、番号付きの印の行は選択の画面のまま |
| AC7 codex 0.166 の footer で入力欄が見つかり、入力待ちの fixture で空と読める。既存の fixture の `ClassifyResumeScreen` は変わらない | 満たす(同じ結果になることは読んで確かめた) | `resumeCodexFooterLine`(`resume_screens.go:290`)は cycle 1 から変わっていない。f24290b5 は `findResumeInputBox`(`:388`)を `resumeMarkerLineBox`(`:402`)と `resumeInputBoxShape`(`:423`)に分けた。最後の印の行を探す走査、`no input box`、番号付きなら `choice dialog`、残りの形の確かめ、の順と返り値は前と同じで、`resumeInputBoxShape` の本体は diff の文脈行のまま。`ResumeInputHolds`(`:347`)は確かめの順が入れ替わったが、どれも副作用がなく結果は同じ。`ClassifyResumeScreen` は触れていない。テスト: `TestSendInputHolds_NumberedStepOnTheMarkerLine` が番号付きの fixture で `ClassifyResumeScreen` が `unknown (choice dialog)`、`ResumeInputHolds` が false のままであることを固定する。plan の S8 は全 fixture を overlay で確かめたと書くが、ここでは走らせていない |
| AC8 入力待ちの判定が、採った画面と既存の testdata のすべてで期待どおりに分かれる | 満たす | `internal/org/testdata/resume/` のファイルと `sendScreenFixtureWants` のキーは、どちらも 63 で同じ集合(cycle 1 の 61 に番号付きの 2 つ)。新しい 2 つは中くらいの文の実機の箱の形に `sendNumberedStepsMsg` の終わりを当てた合成で、テストの comment にもそう書いてある(I4) |
| AC9 help、org skill、recipe、AGENTS.md がこの挙動に合う | 満たす | 番号付きの行の扱いは、help、org skill、recipe、AGENTS.md、仕様が書く粒度(「Enter の前に入力欄に文があることを確かめる」)より細かく、どの文とも食い違わない。下の「Documentation drift」 |

issue #308 の受け入れ条件 1 は AC1 と AC2、受け入れ条件 2 は AC6 が受け持つ(計画の D7 のとおり status で満たす)。cross-review の ACTION_REQUIRED #1 は f24290b5 で直り、HEAD と直す前の binary の比べで確かめた。

## Static analysis

| Command | Result | Notes |
| --- | --- | --- |
| `./scripts/run-static-verify.sh` | pass(exit 0、1 回目で通った) | 範囲は full に落ちた(`unclassified:internal/org/testdata/resume/claude-folder-trust-unnumbered.txt`)。gofmt ok、`go vet ./...` は何も出さず、golangci-lint は `0 issues.`、staticcheck は何も出さなかった。shellcheck と `sh -n`、`jq -e`、Codex の hook の検査 3 つ、`check-sync.sh`(DRIFTED 0)、`check-pipeline-sync.sh`、`check-skill-sync.sh`(14 skill が lock-step)、`check-template-purity.sh`、tech-debt の plan の参照、branch の secret scan(`0936aaea..c93f854b` clean)も通った。並行の golangci-lint との重なりは起きなかった |
| `./scripts/plan-visual.sh digest <plan>` | `1db244296e07` | plan の `- Approved:` の値と一致 |
| `go build ./cmd/ralph`(HEAD と 0efeef81、出力は scratch) | pass | probe に使った。0efeef81 は `git archive` で scratch に出してから build した。repo には何も残していない(`git status --porcelain` は jsonl の ` M` だけ) |
| `shasum -a 256` と `cmp`(skill の写しと recipe の双子) | 一致 | org skill の 4 つは `40fc97060239`、director skill の 4 つは `4e54ec3b108c`。`docs/recipes/isolated-herdr-server.md` と `templates/base/` の写しは同じ |
| `wc -l` | 800 行未満 | `send_screen.go` 647、`resume_screens.go` 655、`send_fallback.go` 379。`verbs.go` は 2597 で cycle 2 では変わっていない |

## Documentation drift

### cycle 1 の drift の行方

| ID | 状況 |
| --- | --- |
| V1(仕様の改訂の項) | 直った。181ecbb3 が `docs/specs/2026-08-01-org-runtime.md` に「2026-11-12 改訂(fix/send-screen-fallback)」(a)〜(g)を足し、2026-11-07 改訂 (b) と FR-3 に改訂の印を付けた。(c) の「Enter の前に打った文が入力欄に入ったことを確かめる」は f24290b5 のあとも正しい |
| V2(Trigger の注記 6 行) | 直った。181ecbb3 が 14 行に注記を足した(cycle 1 の 6 行と掃引の 8 行) |
| V3(self-review の LOW 7 件) | 直った。`Seven small findings of fix/send-screen-fallback`(478 行目)。ただし f24290b5 のあとで中身がずれた(V6) |
| I1(plan の印の遅れ) | 一部残る。AC1〜AC11 と Progress の 4 つは付いた。`Branch created` はまだ `[ ]`。`PR created` は `/pr` で付く |
| I2(director skill と README の要約) | 変わらない。判定には使わない |
| I3(137 行目の Trigger) | sync-docs が発火とみなさないと決めた。f24290b5 が `sendFallbackHerdr` に足したのは時刻の記録(`readAt`)で遅延の注入ではないので、この判断は今も変わらない |

### この run の表

| Doc / contract | In sync? | Notes |
| --- | --- | --- |
| `ralph org send --help`(source の文) | yes | 「checks that the text is in the input box before it presses Enter」は番号付きの場合も正しい |
| `ralph org status --help` | yes | cycle 2 で変わっていない |
| org skill の 4 つの写し | yes | 取り込みのあとも同じハッシュで、send と status の行はこの branch の文のまま |
| director skill の 4 つの写し | yes | 番号付きの行には触れていない |
| `docs/recipes/isolated-herdr-server.md` と双子 | yes | 「a busy line, a choice dialog, … gets nothing typed」は打つ前の判定の話で、f24290b5 のあとも正しい |
| `AGENTS.md`(internal/org の行) | yes | `sendInputHolds` for the check before Enter の書き方のままで合う。origin/main の追加(gate lock と accept)も残る |
| `README.md` | yes | 取り込みのあとも、この branch の追加だけが差 |
| `docs/specs/2026-08-01-org-runtime.md` | yes | 上の V1 |
| `docs/evidence/org-send-screen-probe-2026-11-12.md` | 変更不要 | 番号付きの画面は採っていない(I4) |
| `docs/tech-debt/README.md` | no(V4、V5、V6) | f24290b5 が払った項目と、cycle 2 のコミットが満たした Trigger に注記が無い(self-review の L12 と同じ所見に、Trigger の発火を足した) |
| plan | yes(INFO I1) | D3 は f24290b5 の実装と合う。S8 の記録もある |

### V4(LOW): 479 行目の (1) M02 は f24290b5 で払ったが、行は開いたまま

`Test gaps of fix/send-screen-fallback` の (1)「M02: no test pins the 3 s gap between the two screen reads」と Impact の「Gap (1)」は、まだ穴があると言う。f24290b5 の `TestOrgSendScreen_TwoReadsAreTheReadGapApart`(`internal/org/send_fallback_test.go:228`)と `sendFallbackHerdr.readTimes`(`:97`)は、この行の Trigger (1) が勧めた方法(各 `PaneRead` の時刻を記録し、2 回目が `SendScreenReadGap` 以上あとかを見る)と同じ。`sendFallbackHerdr` を変えたので Trigger (1) も満たしている。勧めとの違いは 2 つ。既存の `TestOrgSendScreen_HerdrWorkingAndScreenWaiting_TypesByTheScreen` ではなく新しいテストに入れた。M02 の変異は、このコミットの時点では走らせていない。/sync-docs で (1) と「Gap (1)」に払った印(`f24290b5`)を付け、変異を cycle 2 の /test が走らせたならその結果を添える。(2)〜(7) は払っていないので行は閉じない。

### V5(LOW): 457 行目の (e) は f24290b5 で払ったが、2026-11-12 の注記は「足されていない」と言う

`Seven mutants of ralph org resume` の Trigger は「the next change to … `ClassifyResumeScreen`, `ResumeInputHolds`」を挙げる。f24290b5 は `ResumeInputHolds` の本体を `holdsHead` に分け、`findResumeInputBox` も分けたので、この Trigger をもう一度満たした。同じコミットが `TestResumeInputHolds` に `claude input box holds only the head of the sentence`(`internal/org/resume_screens_test.go:573`)を足した。入力欄が文の頭だけを見せる画面で、(e) の Trigger が求めたもの。テストの comment も行の名前と (e) を引いている。全部を比べる変異(C10)では、箱の文に文全体が無いので false になり、このテストは落ちるはず(読んだだけ)。行の末尾の 2026-11-12 の fix/send-screen-fallback の注記は、まだ「none is … a screen whose input box shows only the head of the sentence (e)」と言う。/sync-docs で (e) に払った印を付け、注記のその部分を直す。(a)、(b)、(d)、(f)、(g) は払っていない。

### V6(LOW): 478 行目の L7 の数が古く、Trigger も同じ branch の中で満たした

`Seven small findings of fix/send-screen-fallback` の (L7) は、driver に画面の規則があるかを `resumeScreenSentences[driver]` で調べる箇所を「four places」と書く。f24290b5 が `sendInputHolds` の頭(`internal/org/send_screen.go:270`)に 1 つ足し、今は 5 か所(`send_screen.go:191`、`:223`、`:270`、`send_fallback.go:105`、`status_live.go:109`)。0efeef81 の `send_screen.go` には 2 か所だった。さらに、この行の Trigger「(L2, L7, L9, L10) the next change to `send_fallback.go`, `send_screen.go`, or `status_live.go`」を、f24290b5 と cea96ea4 が `send_screen.go` を変えて満たした。行を書いた 181ecbb3 より後のコミットで、L2、L7、L9、L10 はどれも払っていない(self-review cycle 2 の表)。/sync-docs で L7 の数と列挙を 5 か所に直し、持ち越しの注記を 1 文足す。

### INFO

- I4: 番号付きの手順の 2 つの fixture(`claude-input-typed-numbered-steps.txt`、`codex-input-typed-numbered-steps.txt`)は合成で、実機の CLI に番号付きの TASK を打った画面は採っていない。claude で印が文の途中の行に付き、codex で見えている先頭の行に付くことは S3b の実機の画面で見ている。そこまでは採った事実と合うが、番号付きの行そのものは実機で見ていない。`Known gaps of fix/send-screen-fallback`(473 行目)の (b) は「the only screens captured」の規則を言うだけなので、番号付きの場合が合成の fixture だけに頼ることを 1 句足すかは /sync-docs の判断に任せる
- I5: 479 行目の Related は `docs/reports/verify-2026-11-12-send-screen-fallback.md (Coverage gaps)` をコミットなしで引く。この report は cycle 2 で上書きした。この版の Coverage gaps も実機の herdr と既定の `--timeout-ms` の穴を持つので引用は今も読めるが、cycle 1 の版(67b6a551)にだけあった「CLI のテストで画面の道を通すものが無い」は 88742506 で埋まった。/sync-docs で版のコミットを添えると、次の上書きでも読める
- I1 と I2 は上の「cycle 1 の drift の行方」のとおり

## Observational checks

偽の herdr は scratch の `probe2/bin/herdr` に置いた(台本は evidence にある)。呼び出しをすべて記録し、`agent wait` は `--timeout` の分だけ眠って失敗し、`agent get` は working を返す。`pane read` は、send-text の前は `<driver>-idle-after-reply.txt`、send-text のあとは打ったあとの画面(probe ごとに替える)、send-keys のあとは `<driver>-submitted-multiline-long.txt` を返す。台帳は `spawned` の 1 行だけを手で書いた(seat-1、pane-1)。本文は `sendNumberedStepsMsg` の 1 行目を `TYPE: QUESTION` にしたもの(TASK は見張りの確かめにかかるため)。`--timeout-ms 21000` で、最初の待ちはどれも `--timeout 14000` だった。

| Probe | Binary | 打ったあとの画面 | 結果 |
| --- | --- | --- | --- |
| claude-head | HEAD c93f854b | `claude-input-typed-numbered-steps.txt` | 終了コード 0、20 秒。呼び出しは wait、get、read 2 回、ロックの下の read、send-text、Enter の前の read、send-keys、Enter のあとの read。`sent` は `via=screen herdr_state=working TYPE: QUESTION…` |
| codex-head | HEAD | `codex-input-typed-numbered-steps.txt` | claude-head と同じ |
| claude-base | 0efeef81 | 同上(claude) | 終了コード 1、20 秒。send-text のあとの read で止まり、send-keys は無い。`check before Enter … the typed text was not found in the input box, so Enter was not pressed (text typed but not submitted)` と、打った文を確かめる note。`sent` は無い |
| codex-base | 0efeef81 | 同上(codex) | claude-base と同じ |
| claude-dialog | HEAD | `claude-folder-trust.txt` | 終了コード 1、18 秒。Enter を押さず、claude-base と同じエラー。`sent` は無い |
| codex-dialog | HEAD | `codex-hooks-review.txt` | claude-dialog と同じ |

## Coverage gaps

- 実機の herdr が手番の終わった座席を working と言い続ける場面で、送信は 1 度も通していない(AC1、AC2、AC11)。probe の evidence の 9 節で症状が再現せず、473 行目の (a) にある。cycle 1 と変わらない
- 番号付きの手順の場合は、合成の fixture と偽の herdr でだけ通した(I4)。実機の claude と codex の入力欄に番号付きの TASK を打った画面は無い
- codex の番号付きの箱が受け入れられる道は、2 つの fixture とも footer の下に `⚠ 11 · f2` があるので `sendCodexFooterInInput` の枝だけを通る。`codexFooterStart` が footer を見つける枝(`sendCodexHasFooter` の `box.footerStart < len(lines)`)で受け入れる場合は、テストも probe も無い(self-review cycle 2 の観察 (a)。網羅は `/test` の担当)
- AC7 の「既存の fixture の `ClassifyResumeScreen` は変わらない」は、リファクタの前後を読んで確かめた。plan の S8 が書く overlay での全 fixture の確かめは、ここでは繰り返していない
- 既定の `--timeout-ms` 30000 での動きは観察していない。probe は 21000、CLI のテストは 24000 と 18300
- テストの実行結果(`/test` の範囲)。M02 を殺すテストが変異で落ちるか、C10 の変異が新しい `TestResumeInputHolds` の行で落ちるかは、読んだだけ
- 確かめを最も増やす小さな次の一手: 分離した herdr server(recipe)で claude と codex の座席に `sendNumberedStepsMsg` を `pane send-text` で打ち、入力欄の画面を採って、今の合成の 2 つと差し替える(または並べる)。番号付きの場合だけが合成の fixture に頼っている

## Verdict

- Verified: AC1〜AC11 はコードとテストのソースから満たす。cross-review の ACTION_REQUIRED #1(番号付きの印の行で Enter が押されない)は f24290b5 で直り、直す前の binary では再現し、HEAD では消えたことを、偽の herdr で動かした binary の比べで観察した。本物の選択の画面は HEAD でも Enter を押さなかった。cea96ea4 は今の入力では挙動を変えない。origin/main の取り込みは、send と status の文書とコードを変えていない。静的解析は 1 回目ですべて通った。plan の digest は一致した
- Partially verified: AC7 の同じ結果になることは読んだだけ。AC11 は縮めた尺のテストのソースだけ。AC3 は単体テストのソースだけ。番号付きの場合は合成の fixture だけ
- Not verified: 実機の herdr での画面の道。実機の番号付きの入力欄の画面。テストの実行結果。既定の `--timeout-ms` での時間の配分
- Drift(/sync-docs へ): V4 479 行目 (1) M02 を払った印。V5 457 行目 (e) を払った印と、2026-11-12 の注記の直し。V6 478 行目の L7 を 5 か所に直し、`send_screen.go` の変更で満たした Trigger の持ち越しの注記。INFO は I4(合成の fixture)、I5(479 行目の report の引用の版)、I1(`Branch created` の印)
- Verdict: pass
