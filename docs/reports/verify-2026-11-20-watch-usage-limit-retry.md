# Verify report: watch-usage-limit-retry

- Date: 2026-11-20(パイプラインの cycle 2。`cycle-count.json` は `cycle: 2`)。cycle 1 の run は下の「## Cycle 1」に残した。中身は変えず、見出しを 1 段下げ、cycle 1 の `## Verdict` を `### Verdict(cycle 1)` にした
- Plan: `docs/plans/active/2026-11-20-watch-usage-limit-retry.md`。`./scripts/plan-visual.sh digest` は今も `70b063e49ad8` で、計画の `- Approved:` の行と一致する
- Verifier: verifier サブエージェント(`/verify`、cycle 2)
- Scope: HEAD `a5ec5464`、base `main` `a672b72a`(`git ls-remote` で見た origin/main も同じ)。cycle 1 の HEAD `1b9dfa4a` のあとの Go の commit は 3 つ。`29b5d5d6` は cycle 1 の V1 と V3 の直し(help の文とコメント)。`51fe7ddf` は cross-review cycle 1 の ACTION_REQUIRED #1 の直しで、3 つ目の読み方が、入力欄の上の最後のブロックが claude の状態行の記号(`*·✢✳✶✻✽` のどれか)と、空白か文の終わりで始まることも求めるようにした(`claudeStatusLineStart`)。記号の組は `send_screen.go` の `claudeStatusSymbols` の 1 か所に置き、send の 2 つの正規表現もそこから組む。`a5ec5464` は cycle 2 の self-review の L7・L10・L11 で、コメントと help の改行だけ。ほかは文書と記録の commit(`/sync-docs` の `4ae843df`・`48cffdb9`・`d73be884`・`46b6b4b1`・`944a7820`、cross-review の `bd55da61`、self-review の `9d5ae3a4`、origin/main の merge `f4b3a7c4`)。AC1〜AC6 はブランチ全体(`git diff a672b72a..HEAD`、31 ファイル)で見直した。テストは走らせていない(`/test` の担当)
- Evidence: `docs/evidence/verify-2026-11-20-watch-usage-limit-retry.log` の末尾に「# Cycle 2」の節(C2-1〜C2-10)を足した。それまでの 31287 バイトは変えていない(`head -c` と `cmp` で確かめた)。`.gitignore` の `docs/evidence/*.log` に当たり、commit されない。ラッパーは自分の記録として `docs/evidence/verify-2026-11-20-095641.log` も書いた

## Cycle 2(HEAD a5ec5464)

### Spec compliance(cycle 2)

| Acceptance criterion | Status | Evidence |
| --- | --- | --- |
| AC1 | Met | fixture `claude-weekly-limit-retrying.txt` の入力欄の上の最後のブロックは、7 行目の `✻` 1 字の行から始まる。`resumeScreenBlocks` は行を trim して空白 1 つでつなぐ(`internal/org/resume_screens.go:601`)ので、ブロックの文は `✻ Weekly …` になり、`claudeStatusLineStart`(`internal/org/usage_limit_screen.go:207`)は真を返す。`51fe7ddf` が足した早い return(`:178`)はこの画面を通す。テストは `TestUsageLimitScreen_Issue353RetryWaitIsAUsageLimit`(`internal/org/usage_limit_screen_test.go:554`)と fixture の表で、cycle 1 から変わっていない |
| AC2 | Met | 見張りのテスト(`internal/org/watch_usage_limit_test.go:731`)は同じ fixture を読む(`:735`)。見張りが画面を読む道(`watch.go`)は cycle 2 で変わっていない |
| AC3 | Met | resume のテスト(`internal/org/resume_test.go:383`)と CLI のテスト(`internal/cli/org_resume_test.go:360`)も同じ fixture を読む。`resume.go` は `29b5d5d6` でコメントだけ変わった |
| AC4 | Met | 7 つの形の行は `TestUsageLimitScreen_RetryWaitShapes` の中で 11 行ずれただけ。(1) `:672`、(2) `:677`、(3) `:682`、(4) `:692`(止まった読み方の形は `:698`)、(5) と (6) `:713`、(7) `:724`。`51fe7ddf` は偽を返す道を 1 つ足しただけなので、否定の行は否定のまま。足した否定の行は返答の引用の 3 つ(`:766`・`:771`・`:776`)。肯定の行は、最後のブロックがどれも `retry`(`usageLimitRetryStatus`、`✻` で始まる、`:533`)か `✶` の行(`:651`)なので、始まりの条件を満たす |
| AC5 | Met(注つき) | codex は `UsageLimitScreen` の `:108` で偽のまま。ブランチ全体で既存のテストから消えた行は cycle 1 と同じ 3 種(evidence の C2-5)。`1b9dfa4a` のあとに消えた行は、このブランチが足した `TestUsageLimitScreen_RetryWaitShapes` の doc の 4 行だけ。send の `sendClaudeBusyLine` と `sendClaudeTurnEnd` は、`claudeStatusSymbols` から組んだ正規表現の文字列が base とバイト単位で同じ(python で照合、C2-7)なので、send の画面の判定は変わらない。「通る」ことは `/test` の側 |
| AC6 | Met | agent-messaging.md と director skill は cycle 2 で変わっていない。director skill の 4 写しと agent-messaging の 2 写しは `cmp` で同じ(C2-6)。514 行の注もある。`check-sync.sh`(DRIFTED 0)と `check-skill-sync.sh`(14 skill)はラッパーの中で通った |
| issue #353 の期待 2 点 | Met | AC2 と AC3 と同じ |

cross-review の ACTION_REQUIRED #1 は直っている。指摘の 3 つの形(返答が手番を終えるときに 2 行を引用する、返答のあとの段落が `⎿` から引用する、語で始まる段落が引用を含む)は、最後のブロックがそれぞれ `⏺`、`⎿`、語で始まるので `:178` で偽になる。テストは `:766`・`:771`・`:776` の 3 行と `TestUsageLimitScreen_ClaudeStatusLineStart`(`:804`)の表で、表には 7 つの記号それぞれに真の行があるので、組から記号が 1 つ落ちれば表が落ちる。triage が名指した `✻ Can't reach the… · Retrying in 4s · attempt 13/300` も表の 2 行目で真。直しは偽の側にしか効かないので、外れた画面は cycle 1 と同じく stall の ALERT に戻る。

残る取り違え(self-review Run 2 の L7): ブロックを平文で読むので、返答の段落が記号と空白で始まり(`* ` の箇条書き、`· `、引用した再試行の行そのもの)、再試行の語と時刻を持ち、同じ手番に時刻の合う上限の文があれば、今も上限と読む。`a5ec5464` がコメント(`usage_limit_screen.go:156-160`)に書いた。計画の AC4 の 7 つの形には入らないので AC の判定は変えない。ただ、記録はこのコメントにしかない(下の「/sync-docs に渡すもの」の 5)。

### Static analysis(cycle 2)

| Command | Result | Notes |
| --- | --- | --- |
| `./scripts/run-static-verify.sh` | rc=0(09:56:41〜09:57:30 UTC) | 求めた範囲は changed だが、`Language scope: full fallback (unclassified:internal/org/testdata/resume/claude-weekly-limit-retrying.txt)` で全体を走らせた。local verifier(shellcheck、hook の `sh -n`、settings の `jq -e`、Codex の hook の 3 検査、`check-sync.sh` は DRIFTED 0、`check-pipeline-sync.sh`、`check-skill-sync.sh` は 14 skill が一致、`check-template-purity.sh`、`tech-debt README plan references` は OK)、golang verifier(`gofmt: ok`、`go vet ./...` は出力なし、golangci-lint は `0 issues.`、staticcheck は出力なし)、branch の secret scan(`a672b72a..a5ec5464` が clean) |
| `./scripts/plan-visual.sh digest docs/plans/active/2026-11-20-watch-usage-limit-retry.md` | `70b063e49ad8` | `- Approved: 2026-11-20 sha256:70b063e49ad8` と一致 |
| send の 2 つの正規表現の照合(python) | 同じ | `^[` と `claudeStatusSymbols` と `]…` をつないだ文字列が、base の `sendClaudeBusyLine` と `sendClaudeTurnEnd` の文字列とバイト単位で同じ。記号の組を書いた非テストの Go は `send_screen.go:145` の 1 か所だけ |
| `ralph org watch --help` の語の照合(python) | 語の列が同じ | `a5ec5464^` と HEAD の `Long` の文字列を語に割って比べた。動いたのは改行だけで、最長の行は 73 字。この段落を固めるテストはない |
| `git ls-remote origin main` | `a672b72a` | base と同じなので、試しの merge は要らない |

### Documentation drift(cycle 2)

| Doc / contract | In sync? | Notes |
| --- | --- | --- |
| `usage_limit_screen.go` の前書きと、`UsageLimitScreen`・`usageLimitRetryWords`・`usageLimitRetryingTurn`・`claudeStatusLineStart` の doc | Yes | 4 つの条件、残る取り違え、`isClaudeStatusBlock` との違いが本体と合う。前書きと `UsageLimitScreen` の doc は「claude's retry status line」と書き、条件を列挙しないので、直しで読み方が狭まっても偽にならない |
| `send_screen.go` の `claudeStatusSymbols` と 2 つの正規表現の doc | Yes | 挙げた evidence の節(2026-11-12 の 2・3・4・6・7、2026-11-07 の 3)には、どれも書かれたとおりの画面か規則がある |
| `resume.go` の `ResumeReasonUsageLimitWait` の doc(cycle 1 の V3) | Yes | `29b5d5d6` で直った(`:77-86`) |
| `ralph org watch --help`(cycle 1 の V1) | Yes | `29b5d5d6` で 3 つ目の読み方が入った。`a5ec5464` は改行だけ |
| org runtime の仕様の 2026-11-20 改訂 | No | V7 |
| resume の仕様の FR-3 の改訂(2026-11-20) | Yes(INFO) | V7 の後半 |
| `agent-messaging.md` の 2 写し、director skill の 4 写し、README の 302 行 | Yes(INFO) | どれも「claude's retry status line」「再試行の行」と書き、2 つの語で定義していない |
| 計画の Progress | No | V8 |
| tech-debt の、`51fe7ddf` が Trigger を満たした行 | No | V9 |
| tech-debt の、cycle 1 の注の数 | No | V10(540 行は self-review Run 2 の L8) |
| tech-debt の、残る取り違えと self-review Run 1 の L3・L4・L6 | No | self-review Run 2 の L7(register の部分)と L9。HEAD でもまだどの行にもないことを確かめた |

番号は cycle 1 の V1〜V6 から続けた。どれも LOW で、文書と記録だけ。挙動と AC の判定は変わらない。

- V7(LOW): org runtime の仕様の 2026-11-20 改訂 (a)(`docs/specs/2026-08-01-org-runtime.md:130`)は、3 つ目の読み方を「次の 3 つがそろう」と閉じた列挙で書き、(1) を「入力欄のすぐ上のブロックが再試行の状態行(`Retrying in` と `attempt` を両方含む)である」とする。括弧が状態行を 2 つの語で定義していて、`51fe7ddf` の条件(ブロックの文が `claudeStatusSymbols` の記号のどれかと、空白か終わりで始まる)がない。仕様だけを読むと、issue の 2 行を引用した返答も対象に見える。直すなら (1) に「先頭が claude の状態行の記号(`*·✢✳✶✻✽` のどれか)と空白で始まる」を足し、(b)(`:131`)の stall に戻る形に「`⏺`、`⎿`、語で始まる返答の段落が 2 行を引用した画面」を、知られた取り違えとして「記号と空白で始まる返答の段落が引用した画面」を足す。resume の仕様の改訂(`docs/specs/2026-11-07-org-seat-resume.md:62`)は「`Retrying in … (<時刻>) · attempt n/m` の状態行」と書くだけで条件を定義しないので偽ではない。そろえるなら例に `✻` を付ける程度でよい
- V8(LOW): 計画の Progress(`docs/plans/active/2026-11-20-watch-usage-limit-retry.md:111-124`)に、cross-review cycle 1(ACTION_REQUIRED 1 件)、`51fe7ddf`、cycle 2 の self-review(L7〜L11)、`a5ec5464` の行がない。計画の Scope の 3(`:21`)は状態行を「`Retrying in` と `attempt` を両方含む」とだけ書き、Test plan にも始まりの条件はない。Scope は承認の digest に入るので書き換えず、Progress に「ずれ: 最後のブロックの先頭の記号も求めた(`51fe7ddf`)」と残すのが前例どおり。Progress は digest の外(`scripts/plan-visual.sh:42-49`)なので、承認は変わらない
- V9(LOW): `51fe7ddf` が `send_screen.go` を変えて Trigger を満たしたのに、持ち越しの注がない行が 2 つある。行番号は HEAD の `docs/tech-debt/README.md`
  - 488 行(#308 の self-review の L2・L7・L9・L10): Trigger は「the next change to `send_fallback.go`, `send_screen.go`, or `status_live.go`」。`51fe7ddf` は定数を足し、2 つの正規表現をそこから組み直した。4 項目はどれも払っていない
  - 524 行の (h): Trigger は「The next change to `send_screen.go`: move the paste rules (`sendClaudePastesHold`, `sendCodexPasteHolds`, and their two expressions) into `internal/org/send_screen_paste.go`, move-only, in a commit of its own, before the next rule is added」。移していない。1 つめのセルの (h) は「`internal/org/send_screen.go` is 788 lines」と書くが、今は 798 行(`wc -l`。base `1d05c9ac` で 788)で、800 行の目安まで 2 行
- V10(LOW): cycle 1 の `/sync-docs` が足した注の数が、`51fe7ddf` で古くなった。362 行の注は `usage_limit_screen.go` を 317 行、`usage_limit_screen_test.go` を 892 行とし、「counted with `wc -l` at 29b5d5d6, the last Go commit of the branch」と書く。今は 353 行と 960 行で、最後の Go の commit は `a5ec5464`(コードの最後は `51fe7ddf`)。517 行の注も `usage_limit_screen_test.go` を「606 to 892 lines」と書き、足したテストの列に `TestUsageLimitScreen_ClaudeStatusLineStart` と `51fe7ddf` の 4 行がない。`51fe7ddf` と `a5ec5464` は 517 行の Trigger の 3 つ(`UsageLimitScreen` の変更、`usage_limit_screen_test.go` へのテストの追加、`newOrgWatchCmd` の段落)をもう一度満たしている。540 行は self-review Run 2 の L8 と同じ

INFO(直さなくてよいもの):

- `agent-messaging.md`(`:199-204`)の例 `Retrying in 29m (Nov 21 at 3pm) · attempt 1/55` と director skill(`:1051-1057`)の例は、先頭の `✻` を書かない。文は「claude's retry status line」「再試行の行」なので偽ではない。そろえるなら `✻` を足す。写しが 2 つと 4 つあるので、直すときは `check-sync.sh` と `scripts/sync-skills.sh` も回す
- 489 行の (6)「the next change to `ClassifySendScreen`, …」には当たらないと読んだ。`ClassifySendScreen` 自体は変わらず、使う 2 つの正規表現の文字列も base と同じ。213・370・462・491・492 行も、cycle 2 が触った名前が前の注の中に出るだけか、Trigger の条件(370 行の +300 行など)に届かない
- 残る取り違えを画面の形で固めるテストの行はない。`claudeStatusLineStart` の表が `* Retrying in 4s` と `· Retrying in 4s` を真と固めるだけ
- Claude Code が返答の箇条書きを `* ` で始めて描くかは確かめていない。描き方しだいで、残る取り違えが起きうる範囲が変わる

### /sync-docs に渡すもの

1. V7: org runtime の仕様の 2026-11-20 改訂 (a)(1) に始まりの条件、(b) に引用した返答と残る取り違え。resume の仕様は任意
2. V8: 計画の Progress に、cross-review cycle 1、`51fe7ddf`(Scope の 3 からのずれとして)、cycle 2 の self-review、`a5ec5464` の行
3. V9: register の 488 行と 524 行に持ち越しの注。524 行の 1 つめのセルの 788 行を 798 行に
4. V10 と L8: register の 362 行・517 行・540 行の数を `51fe7ddf` か HEAD で数え直す(`usage_limit_screen.go` は base 152 から 353、`usage_limit_screen_test.go` は base 606 から 960)。517 行の足したテストの列と、540 行の移すものの列に `TestUsageLimitScreen_ClaudeStatusLineStart`(`usage_limit_screen_test.go:804`)を足す
5. L7(register の部分): 残る取り違え(記号と空白で始まる返答の段落が再試行の語と時刻を持ち、同じ手番に時刻の合う上限の文がある画面は、今も `usage_limited` になる)を register に書く。515 行の Impact の「none makes a wrong `usage_limited` item」は外れる側(stall に戻る形)の話なので、取り違えは別の行にするか、515 行の中で分けて書く
6. L9: self-review Run 1 の L3(`usageLimitReset*` の名前の群)、L4(`resumeTargetText` が理由の前置きで出し方を選ぶ、`internal/cli/org_resume.go:207`)、L6(`agent-messaging.md:199` の入れ子の括弧)を register の 1 行に。HEAD でもどの行にもない(464 行は org-seat-resume の L3〜L9 で、この L4 とは別)

### Observational checks(cycle 2)

- cross-review の triage の勧めと `51fe7ddf` の diff を 1 つずつ突き合わせた。`*·✢✳✶✻✽` と空白か行末で始まる条件、`⏺` で始まる返答の回帰テスト、記号のない段落の回帰テストは、どれも入っている
- 実機の 2 つの画面(issue #353 の fixture の元の断片と、`docs/evidence/org-seat-resume-probe-2026-11-07.md` の 3)は、どちらも状態行の前に空行があり、状態行が 1 つのブロックになる。前の返答と空行なしでつながる版があれば、ブロックの始まりが `⏺` になり stall に戻る(黙る向きではない)
- register の Trigger の列(5 セルの 4 つめ)を python で走査した(evidence の C2-9 と C2-10)。cycle 2 が触った名前で当たる行のうち、注がないのは 488 行と 524 行だけ
- cycle 1 の V1〜V6 の HEAD での状態は次の表のとおり

| cycle 1 | 内容 | 直した commit | HEAD での状態 |
| --- | --- | --- | --- |
| V1 | `ralph org watch --help` に 3 つ目の読み方がない | `29b5d5d6` | 直った。`a5ec5464` は改行だけ(C2-8) |
| V2 | 仕様に改訂がない | `4ae843df` | 直った。(a)(1) は `51fe7ddf` で足りなくなった(V7) |
| V3 | `ResumeReasonUsageLimitWait` の doc | `29b5d5d6` | 直った(`resume.go:77-86`) |
| V4 | 持ち越しの注がない 6 行 | `48cffdb9`(267・362・483 行にも) | 直った。362 行と 517 行の数は古くなった(V10) |
| V5 | 800 行を超えた 2 つのテストのファイル | `48cffdb9`、`46b6b4b1`(540 行) | 直った。数は古くなった(self-review Run 2 の L8) |
| V6 | AC のチェックボックス | `d73be884` | 直った |

### Coverage gaps(cycle 2)

- テストは走らせていない。`go vet ./...` が新しいテストのコンパイルまでを確かめた
- `51fe7ddf` の 3 行と表が直しの前のコードで落ちることは、self-review の手のトレースで見ただけで、変異は回していない
- 実機の画面の全体はない(cycle 1 と同じ)。返答の箇条書きの描かれ方も見ていない

確信を上げる最小の追加の確かめ: `/test` で `go test ./internal/org/ -run 'UsageLimit|SendScreen'` を回し、`usage_limit_screen.go:178` の早い return を消す一時の変異を 1 つ回す。`:766`・`:771`・`:776` の 3 行が落ちれば、cross-review の指摘がテストで守られていることが分かる。send の正規表現は文字列が同じなので、send の表は変異なしで通れば足りる。

## Verdict

- Verdict: pass
- Verified: AC1〜AC6 と issue #353 の期待 2 点が `51fe7ddf` と `a5ec5464` のあとも成り立つこと(テストの中身は読みで、静的解析は `run-static-verify.sh` の rc=0 で)。cross-review の ACTION_REQUIRED #1 の 3 つの形が偽になり、issue の画面と通信断の再試行の行が始まりの条件を通ること。send の 2 つの正規表現の文字列が base と同じこと。`ralph org watch --help` の語が `a5ec5464` の前後で同じこと。計画の digest の一致。写しの一致と `check-sync.sh`・`check-skill-sync.sh` の通過。cycle 1 の V1〜V6 がどれも直っていること
- Partially verified: AC5 の「通る」(既存のテストの変更は cycle 1 と同じ 3 種と確かめた。実行は `/test`)。文書と記録の食い違いは V7〜V10 の LOW 4 件と、self-review Run 2 から持ち越した L7(register の部分)・L8・L9 で、どれも `/sync-docs` で扱える
- Not verified: テストの実行、変異の確かめ、実機の画面の全体、返答の箇条書きの描かれ方、待ちのあいだの `herdr agent get` の出力、再試行が尽きたあとの画面
- Insight event: `./scripts/insights-append.sh --slug watch-usage-limit-retry --flow standard --phase verify --verdict pass --cycle auto --source skill` で `docs/insights/events/2026-11-20-watch-usage-limit-retry.jsonl` に 1 行を足した(cycle 2)。skill のコマンドは指摘の数を渡さないので、`findings` は LOW があっても 0 のまま(cycle 1 と同じ)

## Cycle 1(HEAD 1b9dfa4a、報告の commit f8303063)

- Date: 2026-11-20(パイプラインの cycle 1。`cycle-count.json` は `cycle: 1`)
- Plan: `docs/plans/active/2026-11-20-watch-usage-limit-retry.md`。`./scripts/plan-visual.sh digest` の値 `70b063e49ad8` は、計画の `- Approved:` の行と一致する
- Verifier: verifier サブエージェント(`/verify`)
- Scope: `git diff 1d05c9ac...HEAD`(HEAD `1b9dfa4a`、22 ファイル、+1015 −55)。S1 `10541be6`、S2 `e57ddb65`、S3 `5ddd4ef9`、self-review の直し S4 `cfe09a8f`(M1、L1)と、計画・報告の commit。issue #353 の「期待すること」の 2 点も見た。origin/main(`9e27d961`)は base のあとに `docs/specs/2026-11-20-org-reseat-restart-herdr.md` だけを足していて、`git merge-tree` で衝突はない。テストは走らせていない(`/test` の担当)
- Evidence: `docs/evidence/verify-2026-11-20-watch-usage-limit-retry.log`(`.gitignore` の `docs/evidence/*.log` に当たり、commit されない)。静的検査の出力、digest、写しの `cmp`、fixture と issue の本文の照合、行数、register の Trigger の走査、origin/main への試しの merge が入っている。ラッパーは自分の記録として `docs/evidence/verify-2026-11-20-081939.log` も書いた(これも gitignore に当たる)

### Spec compliance

| Acceptance criterion | Status | Evidence |
| --- | --- | --- |
| AC1: issue の画面を元にした fixture で `UsageLimitScreen("claude", …)` が上限と読み、行が `⎿ You've hit your weekly limit · resets Nov 21 at 3pm (Asia/Tokyo)` を含む | Met | 3 つ目の読み方は `usageLimitRetryingTurn`(`internal/org/usage_limit_screen.go:153`)で、`UsageLimitScreen`(`:90`)が footer の読み方のあとに claude だけ呼ぶ(`:107`)。テストは `TestUsageLimitScreen_Issue353RetryWaitIsAUsageLimit`(`internal/org/usage_limit_screen_test.go:554`)と fixture の表の行(`:55`)。fixture `internal/org/testdata/resume/claude-weekly-limit-retrying.txt` の 3〜5 行と 7〜9 行は、`gh issue view 353` の本文の 2 つのコードブロックとバイト単位で同じ(python で照合)。末尾 5 行は `claude-weekly-limit-under-prompt.txt` の末尾 5 行と `cmp` で同じ。Progress の S1 の記述どおり |
| AC2: stall の条件が立った座席でこの画面を読むと、stall ではなく `usage_limited` の件(`REASON: usage_limit`、`SEATS:`、本文に上限の文とリセットの時刻)を上げる | Met | 見張りは stall が立った座席の画面を、leader の待ちの除外より前に 1 回読む(`internal/org/watch.go:1325`、除外は `:1333`)。herdr の状態では分けないので、issue のように herdr が working と言う座席にも届く。テスト `TestWatchUsageLimit_AClaudeSeatRetryingUntilItsWeeklyLimitResetsIsUsageLimited`(`internal/org/watch_usage_limit_test.go:731`)は implementer と leader の 2 通りで、stall の ALERT がないこと、受信箱が `usage_limited` の 1 件だけであること、`assertUsageLimitedBody` で `CONDITION`・`REASON: usage_limit`・`SEATS`・本文の行 `<seat> (claude): ⎿ You've hit your weekly limit · resets Nov 21 at 3pm (Asia/Tokyo)` を確かめる |
| AC3: `ralph org resume --dry-run` が、herdr が working の座席でこの画面を読むと `waiting for the usage limit to reset: <行>` を出し、何も打たない | Met | `resumeLook` の working の分岐(`internal/org/resume.go:448`)が `resumeWorkingReason`(`:482`)を呼ぶ。画面が waiting(`:487`)で `UsageLimitScreen` が真なら理由を `ResumeReasonUsageLimitWait`(`:87`)と行にする。idle で waiting の画面も同じ(`:465`)。CLI は `resumeTargetText`(`internal/cli/org_resume.go:207`)でこの理由を画面の行より先に出す。テストは `TestOrgResume_UsageLimitWait_TheReasonNamesTheLimit`(`internal/org/resume_test.go:383`、dry run とそうでない run、イベントとロックのファイルがないこと)と、CLI を通す `TestOrgResume_ASeatWaitingForItsUsageLimitToResetShowsTheLimit`(`internal/cli/org_resume_test.go:360`、出力の行、pane ごとに読みが 1 回、send と `resumed` がないこと) |
| AC4: 7 つの画面を上限と読まない | Met | `TestUsageLimitScreen_RetryWaitShapes`(`usage_limit_screen_test.go:584`)の行に 1 対 1 で当たる。(1) 再試行の行だけ `:661`。(2) 前の手番にだけ上限の文 `:666`。(3) 返答の文の途中の上限の文 `:671`。(4) 再試行の行がなく手番の中に上限の文があるだけ `:681`(unknown。止まった読み方に当たる形は `:687` で、1 つ目の読み方のまま真)。(5) tool の出力の引用のあとの通信断の再試行 `:702`。(6) 同じ手番で上限の文のあとに時刻のない再試行: (5) と同じ形で、`usageLimitResetsIn` は文の出どころ(tool の出力か CLI か)を見ないので、`:702` の行が (6) も覆う。(7) prompt が画面の外で時刻が合わない `:713`。ほかに codex の行(`:743`)、時刻なし、空の時刻、`attempt` なし、`Waiting for API response`、再試行の行が最後のブロックでない画面も否定の行がある |
| AC5: codex の読み方と fixture の表の今の行は変わらない。今ある見張りと usage limit のテストは変えずに通る。working の座席の呼び出しの列を固定するテストは直す(意図した変更) | Met(注つき) | codex は footer の読み方のあと `:107` で偽を返し、base と同じ。`git diff -w` で既存のテストから消えた行は、(a) `TestOrgResume_RetryingOrWorking_Waits` の working の行の呼び出しの列(`resume_test.go:343` に `read pane-1 60` を足した、AC5 が名指す意図した変更)、(b) `TestUsageLimitScreen_EveryFixture`(`usage_limit_screen_test.go:263`)の waiting の分岐を「footer の行か、再試行の待ちの行」に広げた変更(新しい fixture を通すため。Progress の S1 に「ずれ」として記録済み。既存の fixture の判定は変わらない)、(c) コメントの「four」を「five」にした 2 か所と CLI の help のテストのコメント、の 3 種だけ。fixture の表(`resume_screens_test.go:109`、`send_screen_test.go:135`、`usage_limit_screen_test.go:55`)は行が増えただけ。「通る」ことは `/test` の側 |
| AC6: agent-messaging.md と director skill の 4 写しがこの待ちの形を書き、tech-debt の 514 行に追記があり、`check-sync.sh` と `check-skill-sync.sh` が通る | Met | `agent-messaging.md:199-204`、director skill `:1051-1057`。写しは `cmp` で同じ(director skill の 4 つ、agent-messaging の 2 つ)。514 行の Trigger の列の末尾に 2026-11-20 の注がある。2 つの検査はラッパーの中で通った(Static analysis) |
| issue #353 の期待 1: 見張りが `usage_limited` の件を上げ、`SEATS:` に座席、本文にリセットの時刻 | Met | AC2 と同じ。issue の座席は leader で、テストは leader の行も持つ |
| issue #353 の期待 2: `ralph org resume --dry-run` がその座席を上限の解除を待つ座席として出す | Met | AC3 と同じ。issue の出力 `waiting <org>/leader claude herdr reports working` は、新しい binary では `waiting … claude waiting for the usage limit to reset: ⎿ You've hit your weekly limit · resets Nov 21 at 3pm (Asia/Tokyo)` になる(CLI のテストの期待の行) |

#### 計画とのずれ(記録済み)

- S4 `cfe09a8f`(self-review M1)で、herdr が working の座席に理由を付けるのは、画面が waiting で上限と読めたときだけにした。計画の Scope の文(「`UsageLimitScreen` が上限と読めば」)より狭いが、Progress の 120 行に記録がある。M1 の勧め(`ClassifyResumeScreen` の waiting の確かめと、止まった画面の 2 行のテスト)と突き合わせると、コードは `resume.go:487`、テストは `resume_test.go:400` からの 2 行で、勧めのとおり。L1 の勧め(`watch_usage_limit.go` の 2 か所のコメントに 3 つ目の読み方を書く)も `cfe09a8f` に入っている。`cfe09a8f` は self-review の報告(`94fa8031`)のあとの commit で、self-review はこれを見ていない。この verify で上の突き合わせをした
- `TestUsageLimitScreen_EveryFixture` の分岐を広げたことは、AC5 の「変えずに通る」の字面から外れる。`testing.md` の「意図を弱めたら計画と報告に書く」は、計画の Progress の S1 で満たしている。広げた側の `usageLimitRetryingLine`(`usage_limit_screen_test.go` の helper)は時刻の一致まで独立に確かめるので、既存の fixture の確かめは弱まっていない

### Static analysis

| Command | Result | Notes |
| --- | --- | --- |
| `./scripts/run-static-verify.sh` | rc=0(08:19:38〜08:21:15 UTC) | 求めた範囲は changed だが、`Language scope: full fallback (unclassified:internal/org/testdata/resume/claude-weekly-limit-retrying.txt)` で全体を走らせた。local verifier(shellcheck、hook の `sh -n`、settings の `jq -e`、Codex の hook の 3 検査、`check-sync.sh` は DRIFTED 0、`check-pipeline-sync.sh`、`check-skill-sync.sh` は 14 skill が一致、`check-template-purity.sh`、`tech-debt README plan references` は OK)、golang verifier(`gofmt: ok`、`go vet ./...` は出力なし、golangci-lint は `0 issues.`、staticcheck は出力なし)、branch の secret scan(`1d05c9ac..1b9dfa4a` が clean) |
| `./scripts/plan-visual.sh digest docs/plans/active/2026-11-20-watch-usage-limit-retry.md` | `70b063e49ad8` | `- Approved: 2026-11-20 sha256:70b063e49ad8` と一致 |
| `cmp`(director skill の 4 写し、agent-messaging の 2 写し) | 一致 | |
| `git merge-tree --write-tree --name-only origin/main HEAD` | rc=0、衝突なし | origin/main が足したのは別の spec 1 つで、#353 の話題(`usage_limited`、`Retrying in`、resume)には触れない |

`go vet ./...` はテストのファイルもコンパイルするので、新しいテストの型と名前の誤りはここで出る。テストの実行は `/test` の担当。

### Documentation drift

| Doc / contract | In sync? | Notes |
| --- | --- | --- |
| `usage_limit_screen.go` の前書きと `UsageLimitScreen`・`usageLimitRetryingTurn`・`usageLimitResetsIn`・`usageLimitResetTime` の doc | Yes | 4 つの条件、claude だけ、最後の一致を返すこと、返す行の範囲が本体と合う |
| `resume_screens.go` の claude の表の `UsageLimitWaiting` のコメント | Yes | `Retrying in` は印を付けず、3 つ目の読み方が時刻の一致で読むと書く |
| `resume.go` の `Resume` の doc の 2 と 3、`resumeWorkingReason`、`ResumeTargetResult.Line` | Yes | `cfe09a8f` の狭め方と合う |
| `resume.go` の `ResumeReasonUsageLimitWait` の doc(`:77-86`) | 一部(V3) | |
| `ralph org resume --help`(`internal/cli/org_resume.go:54-58`)と `resumeTargetText` の doc | Yes(INFO あり) | help のテストが文を固める(`org_resume_test.go:718`) |
| `ralph org watch --help`(`internal/cli/org_watch.go:102-106`) | No | V1 |
| `watch_usage_limit.go` の前書きと `usageLimitedWhy` のコメント | Yes | `cfe09a8f` で 2 つ目と 3 つ目の読み方を書いた |
| `agent-messaging.md` の 2 写しの `usage_limited` の段落 | Yes | 入力欄のすぐ上の再試行の行、今の手番、`⎿` か `⏺`(claude の `StopGlyphs`、`resume_screens.go:255`)、同じ時刻、時刻のない再試行は読まない、のどれもコードと合う |
| director skill の 4 写しの `usage_limited` の項 | Yes(INFO あり) | 手番の中で待つ座席の画面と、resume が何も打たずに出す理由が、コードと合う |
| `AGENTS.md` の `internal/org/` の行、README の watch と resume の段落 | Yes | どちらも「止まっているか、解除を待っている」と書き、読み方を列挙しない |
| org runtime の仕様と resume の仕様 | No | V2 |
| tech-debt の 514 行(#329 の穴)と新しい 515 行(#353 の穴) | Yes | 514 行は Trigger の列に注を足した。515 行は 5 セルで、計画は active のパスか素の名前で指す(`plans/archive/` はない)。gate も OK |
| tech-debt の、この PR が Trigger を満たした行 | No | V4 |
| tech-debt の、800 行を超えたテストのファイル | No | V5 |
| 計画の AC のチェックボックス | No | V6 |

指摘はどれも LOW で、文書だけ。挙動と AC の判定は変わらない。`/sync-docs` で直すか、PR 本文の Known gaps に書くかを orchestrator が決める。

- V1(LOW): `ralph org watch --help` の段落は、stall を立てない座席を「screen shows a usage limit (a stop sentence ralph org resume reads as one, or the CLI's wait for the limit to reset in the footer below the input box)」と書く(`internal/cli/org_watch.go:102-106`)。括弧は 2 つの読み方を挙げる閉じた列挙で、入力欄の上の再試行の行で読む 3 つ目が入っていない。help を読んだ人は、手番の中で再試行している座席には stall が立つと受け取る。直すなら括弧に「or claude's retry status line above the input box whose time in parentheses is the reset time of a usage limit sentence of the current turn」のような 1 句を足す。Go の文字列の変更で、この段落を固めるテストはない(register 517 行の (d))。`/sync-docs` で直すなら `gofmt` と `go vet` を当て、直さないなら Known gaps に 1 行
- V2(LOW): 仕様に、この変更の改訂がない。(1) org runtime の仕様の 2026-11-13 改訂 (b)(`docs/specs/2026-08-01-org-runtime.md:98`)は「`Retrying in` と `Waiting for API response` は API の再試行の待ちなので印を付けず、今までどおり stall にする」と書く。日付のある改訂なのでその日の時点では正しいが、括弧の時刻が手番の上限の文のリセットの時刻と同じ `Retrying in` の座席は、今は `usage_limited` になる。前例(#329 の `2026-11-13 改訂(fix/claude-5h-usage-limit)`)と同じく、`2026-11-20 改訂(fix/watch-usage-limit-retry)` を足すのが合う。(2) resume の仕様(`docs/specs/2026-11-07-org-seat-resume.md:60`)は「見張りの `usage_limited` は、このうち `Continuing automatically at` と `Continuing shortly` が入力欄の下の footer にある画面を利用上限の画面と読む」と書く。3 つ目の読み方が抜けている。「resume の判定は変わらず、待ち中のまま打たない」は今も正しい(outcome は waiting のまま)が、理由の文が `waiting for the usage limit to reset: <行>` に変わったことはどこにもない。計画の Scope にも Non-goals にも仕様の名前はない
- V3(LOW): `ResumeReasonUsageLimitWait` の doc(`internal/org/resume.go:77-86`)は「a waiting target whose screen UsageLimitScreen reads as a usage limit, herdr reporting it working or the screen showing the CLI waiting」と書く。`cfe09a8f` のあと、herdr が working で画面が上限で止まっている座席(`UsageLimitScreen` は 1 つ目の読み方で真)はこの理由にならず、`herdr reports working` になる。doc は括弧で `Resume` の doc の 2 と 3 を指し、そちらは正しい。直すなら「herdr reporting it working with a waiting screen」のように 1 語足す。Go のコメントだけ
- V4(LOW): この PR が Trigger を満たしたが、持ち越しの注がない行が 6 つある。前例(#322・#326 が 517 行に、#329 が 490・493・494 行に)は「hit this trigger … but did not pay it … Carried over」の注を足している。行番号は HEAD の `docs/tech-debt/README.md`
  - 464 行(org-seat-resume の self-review の L3〜L9): Trigger は「the next change to the re-check of `resumeOne`, the result lines of `internal/cli/org_resume.go`, …」。この PR は `resumeTargetText`(結果の行に何を出すか)に分岐を足し、`resumeOne` の読み直しも `resumeLook` を通して working の座席の画面を 1 回読むようになった。L3〜L9 はどれも払っていない
  - 490 行(`send_screen_test.go` と `send_fallback_test.go` の大きさ): Trigger は「The next change to either file」。`send_screen_test.go` は fixture の行 1 つで 878 行から 885 行(`wc -l`、base `1d05c9ac` と HEAD)。numbered-line のテストは移していない
  - 493 行(#312a のテストの穴): Trigger は「The next change to `internal/org/watch_usage_limit.go` or its test file」。どちらも変えた(コメント 2 か所とテスト 1 つ)。(a)、(c) の `recur` の失敗の分岐、(d) の 2 つ目の acked の行は足していない
  - 494 行(`watch_usage_limit_test.go` の大きさ): Trigger は「The next change to this file」。1000 行から 1035 行。fixture と helper は移していない。行の注の最後の数(1000)は今は古い
  - 516 行(待ちのあいだ `herdr agent get` の出力が変わらないか): Trigger は「a report of a stall ALERT or of silence for such a seat」。issue #353 は、利用上限の解除を待つ claude の座席(手番の中で再試行する形)に stall の ALERT が立ったという報告で、この Trigger に当たる。stall が立ったので、少なくともこの形では、2 周期のあいだ `agent get` の出力が同じだったことになる(issue の本文から読んだもので、`watch-status-<org_id>.json` は見ていない)。この PR の見張りの道(AC2)は stall が立つ座席でしか走らないので、この点は AC2 が実機で効くかに直結する。footer の待ちの形(行の本来の対象)は今も確かめていない
  - 517 行(#329 のテストの穴): Trigger は「The next change to … `UsageLimitScreen` …, or the next test added to `usage_limit_screen_test.go` or `watch_usage_limit_test.go`」。3 つとも当たる。(a)〜(d) はどれも足していない
- V5(LOW): 2 つのテストのファイルが、この PR で 800 行の目安を超えたが、register に行がない。`internal/org/usage_limit_screen_test.go` は 606 行から 892 行(+286、`TestUsageLimitScreen_RetryWaitShapes` だけで約 185 行)、`internal/org/resume_test.go` は 800 行から 869 行。前例(490 行、494 行、520 行)は、超えたファイルと分け方の案を 1 行にしている。526〜768 行(#353 の helper と 2 つのテスト、約 240 行)を `usage_limit_screen_retry_test.go` に移せば 650 行ほどに戻る
- V6(LOW): 計画の AC1〜AC6 のチェックボックス(64〜71 行)がすべて `[ ]`。この報告をもとに `/sync-docs` か `/pr` で `[x]` にする。`plan-visual.sh` の digest は行頭の `- [x]` を `- [ ]` として読む(`scripts/plan-visual.sh:46-48`)ので、チェックを付けても承認の digest は変わらない

INFO(直さなくてよいもの):

- `/org` skill の `usage_limited` の項(`.claude/skills/org/SKILL.md:686-693`)と watch の行(`:383`)は、止まった座席の読み方しか書かない。#329 の verify の V2 から残っている穴で、この PR は読み方を 1 つ増やしたので穴が少し広がった。偽ではない
- working の座席の pane の読みが失敗すると、理由は `herdr reports working` のまま(self-review L2。計画が決めた挙動で、`resume_test.go:397` が固める)。`ralph org resume --help` と director skill の「`waiting for the usage limit to reset: <上限の行>` と出す」はこの例外を書かない
- `ResumeReasonUsageLimitWait` の doc と director skill の「この座席も CLI が自分で再試行して続ける」は、再試行が解除の時刻まで続くことを前提にしている。issue の画面は `Retrying in 29m … attempt 1/55` で、週の上限の解除が再試行の尽きるより遅ければ、手番は上限の文で終わる。そのときは 1 つ目の読み方で `usage_limited` のままなので黙らないが、座席は解除のあとに `ralph org resume` が要る。未確認。同じ画面は、README の 302 行と 2 つの recipe(`docs/recipes/isolated-herdr-server.md:263`、`docs/recipes/director-launch-cmd.md:277`)の「Claude Code は `api_wait_minutes` ほど待ってから手番を終える」が利用上限には当てはまらないかもしれないことも示す。この PR の前からの文で、確かめていない
- 再試行している座席の件の本文の行(`⎿ You've hit your weekly limit · resets Nov 21 at 3pm (Asia/Tokyo)`)は、止まった座席の行と同じ形で、件だけでは見分けられない。director skill は座席の画面を読む手順なので、画面の再試行の行で見分けられる。resume の理由でも分かる。issue がリセットの時刻を求めたことに沿った設計
- コードのコメントとテストのコメントは計画を `docs/plans/archive/2026-11-20-watch-usage-limit-retry.md` で指す。`/pr` が計画を archive に移すまでは指し先がないが、どの gate も読まない(前例どおり)。register は active のパスで指していて、`archive-plan.sh` が書き換える

### Observational checks

- `gh issue view 353 --repo yoshpy-dev/ralph` で本文を読み、「期待すること」の 2 点を AC2 と AC3 に当てた(上の表)
- fixture の issue の断片は、issue の本文の 2 つのコードブロックとバイト単位で同じ(python で行の列を照合。fixture の 3 行目と 7 行目から)
- 見張りが画面を読む道(`watch.go:1321-1334`)を読み、herdr の状態で分けず、leader の除外より前に読むことを確かめた。issue の leader は herdr が working と言う座席だったが、この道には届く
- self-review の M1 と L1 の勧めを `cfe09a8f` の diff と 1 つずつ突き合わせた(Spec compliance の「計画とのずれ」)。self-review が直さずに残した L2〜L6 は、L5 が register の 515 行に入り、ほかは計画の Progress の 120 行に記録がある
- register の Trigger の列(5 セルの 4 つめ)を python で走査し、この PR の触ったファイルと関数の名前が当たる行を読んだ(V4)。走査の出力は evidence の log の 7 節

### Coverage gaps

- テストは走らせていない。AC1〜AC5 のテストが通ることは `/test` の側。`go vet ./...` がテストのコンパイルまでを確かめた
- 実機の画面の全体はない。fixture は issue の断片と別の fixture の入力欄と footer の組み立てで、再試行の行の括弧の時刻と上限のリセットの時刻が同じことも 1 つの画面でしか見ていない(計画の Assumptions、register 515 行)。pane がもっと狭くて括弧の時刻が 2 行に割れる画面、時間帯の括弧のない上限の文のあとに別の行が続く画面は、stall のまま(黙る向きではない)
- 待ちのあいだ `herdr agent get` の出力が変わらないかは、issue の報告から推した(V4 の 516 行)。見張りの状態ファイルは見ていない
- 再試行が解除の時刻より先に尽きたときの画面と、そのときの resume の出し方は見ていない(INFO の 3 つ目)

確信を上げる最小の追加の確かめ: `/test` で計画の S2 の検証コマンドを回し、`usage_limit_screen.go:172` の時刻の一致(`strings.Contains(retry.text, "("+limit.reset+")")`)を「手番に上限の文があれば真」に替える一時の変異を 1 つ回す。`TestUsageLimitScreen_RetryWaitShapes` の `:702`(tool の出力の引用のあとの通信断の再試行)と `:713`(時刻が合わない)の行が落ちれば、計画が一番恐れた取り違え(通信断を上限と読む)がテストで守られていることが分かる。

### Verdict(cycle 1)

- Verdict: pass
- Verified: AC1〜AC6 と issue #353 の期待 2 点(テストの中身は読みで、静的解析は `run-static-verify.sh` の rc=0 で)。計画の digest の一致。fixture と issue の本文の一致。director skill の 4 写しと agent-messaging の 2 写しの一致、`check-sync.sh` と `check-skill-sync.sh` の通過。self-review の M1 と L1 の直しが勧めのとおりであること。origin/main と衝突しないこと
- Partially verified: AC5 の「通る」(既存のテストの変更は `git diff -w` で 3 種だけと確かめた。実行は `/test`)。文書の食い違いは V1〜V6 の LOW 6 件で、どれも `/sync-docs` か PR 本文で扱える
- Not verified: テストの実行、変異の確かめ、実機の画面の全体、待ちのあいだの `herdr agent get` の出力、再試行が尽きたあとの画面
- Insight event: `./scripts/insights-append.sh --slug watch-usage-limit-retry --flow standard --phase verify --verdict pass --cycle auto --source skill` で `docs/insights/events/2026-11-20-watch-usage-limit-retry.jsonl` に 1 行を足した(cycle 1)。skill のコマンドは指摘の数を渡さないので、`findings` は LOW 6 件があっても 0 のまま(前例どおり)
