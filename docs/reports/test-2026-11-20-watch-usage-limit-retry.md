# Test report: watch-usage-limit-retry

- Date: 2026-11-20(パイプラインの cycle 2。`cycle-count.json` は `cycle: 2`)。cycle 1 の run は下の「## Cycle 1」に残した。中身は変えず、見出しを 1 段下げ、cycle 1 の `## Verdict` を `### Verdict(cycle 1)` にした
- Plan: `docs/plans/active/2026-11-20-watch-usage-limit-retry.md`(issue #353)
- Tester: tester サブエージェント(`/test`、cycle 2)
- Scope: HEAD `0ea15fd2`(base `main` `a672b72a`、merge base も同じ、worktree は clean)。cycle 1 の HEAD `87693818` のあとのコードの変更は 2 つ。`51fe7ddf` は cross-review cycle 1 の ACTION_REQUIRED #1 の直しで、3 つ目の読み方は、入力欄の上の最後のブロックが claude の状態行の記号と、空白か文の終わりで始まることも求める(`claudeStatusLineStart`)。記号の組は `send_screen.go` の `claudeStatusSymbols` の 1 か所に置き、send の 2 つの正規表現もそこから組む。`a5ec5464` はコメントと help の改行だけ。`./scripts/run-test.sh` は 1 回だけ、worktree のルートから回した。起動は cycle 1 と同じで、perl の fork と setsid で切り離し(INT・HUP・QUIT・TERM は既定の扱い、標準入出力は `/dev/null`)、`caffeinate -w` で終わりを待った。範囲は `- Requested scope: changed` で、cycle 1 と同じく `Language scope: full fallback (unclassified:internal/org/testdata/resume/claude-weekly-limit-retrying.txt)` になった。そのあと、AC の 2 つの絞り込み、依頼の変異 3 つ、send の判定の確かめ、base のテストを HEAD に当てる run、狭い coverage の run を足した
- Evidence: `docs/evidence/test-2026-11-20-watch-usage-limit-retry.log` の末尾に「# Cycle 2」の節を足した。cycle 1 の 716,337 バイトは変えていない(`head -c` と `cmp` で確かめた)。`.gitignore` の `docs/evidence/*.log` に当たるので commit されない。run-verify.sh は自分の記録として `docs/evidence/verify-2026-11-20-105017.log` も書いた(同じく追跡外)

## Cycle 2(HEAD 0ea15fd2)

### Test execution(cycle 2)

| Suite / Command | Tests | Passed | Failed | Skipped | Duration |
| --- | --- | --- | --- | --- | --- |
| `./scripts/run-test.sh` | 下の 2 行の合計 | 1 つを除いて全部 | 1(`internal/org` の `TestGate_TwoGatesRunTheirScriptsOneAtATime`、下の Failure analysis) | 2(cycle 1 と同じ条件つきの skip) | 2026-11-20 19:50:17 〜 20:24:25 JST、2,048 s、rc 1。開始時の 1 分 load 12.3、終了時 8.1 |
| local verifier(`tests/test-*.sh` 46 本) | 46 本、`PASS` で始まる行 5,390 | 46 本(`OK` 46 行) | 0(`FAIL:` の要約行 29 本はどれも `FAIL: 0`) | `test-plan-visual.sh` の browser detection、`test-secret-scan-branch.sh` の「git 2.41 より古い本物の git」の場合 | 約 18 分(19:50:17 〜 20:08 ごろ)。この段の 1 分 load は 20:03 に 236 まで上がった(ほかのセッションの Xcode のビルド) |
| `go test ./...`(テストのある 11 パッケージ) | 11 | 10 | 1(`internal/org` の 1 テスト) | なし。6 パッケージが `(cached)` で、`-count=1` で回し直して ok(下の表) | 約 16 分(20:08 ごろ 〜 20:24:25) |
| `go test -count=1 -v ./internal/org/ -run 'UsageLimit\|Resume\|WatchUsage\|SendScreen'` | 96 関数、`--- PASS` 627 行(subtest を含む) | 全部 | 0 | 0 | 68 s(パッケージ 65.2 s) |
| `go test -count=1 -timeout 9m -v ./internal/cli/ -run 'Resume\|OrgWatch\|Help'` | 118 関数、`--- PASS` 200 行 | 全部 | 0 | 0 | 131 s(パッケージ 129.9 s) |
| `go test -count=20 -v ./internal/org/ -run '^TestGate_TwoGatesRunTheirScriptsOneAtATime$'` | 20 | 20 | 0 | 0 | 54 s。1 分 load 7.9 から 19.7 |
| `go test -count=1 -timeout 30m ./internal/org/`(パッケージ全体の打ち直し) | パッケージ 1 つ(`-v` なし) | ok | 0 | — | 20:27:52 〜 20:39:49、717 s(パッケージ 715.576 s)。1 分 load 6.0〜13.9 |

cli の絞り込みは依頼では `-v` なしだったが、関数の数を数えるために `-v` を付けた(cycle 1 と同じ)。org の絞り込みの 96 関数は、cycle 1 の 69 関数に send の画面の 26 関数と `TestUsageLimitScreen_ClaudeStatusLineStart` を足した数。cli の 118 関数のうち `Help` に当たるのは 54 関数。

| パッケージ | 結果 | 時間 |
| --- | --- | --- |
| `cmd/ralph` | ok | 0.739 s |
| `internal/cli` | ok | 796.199 s |
| `internal/config` | ok | 11.601 s |
| `internal/gitroot` | ok(`(cached)`。`-count=1` で ok) | 1.895 s |
| `internal/insights` | ok | 1.334 s |
| `internal/org` | FAIL(`TestGate_TwoGatesRunTheirScriptsOneAtATime` だけ) | 905.699 s |
| `internal/org/driver` | ok(`(cached)`。`-count=1` で ok) | 1.259 s |
| `internal/org/protocol` | ok(`(cached)`。`-count=1` で ok) | 0.998 s |
| `internal/scaffold` | ok(`(cached)`。`-count=1` で ok) | 1.370 s |
| `internal/toolchain` | ok(`(cached)`。`-count=1` で ok) | 15.657 s |
| `internal/upgrade` | ok(`(cached)`。`-count=1` で ok) | 2.496 s |

`(cached)` の 6 パッケージは `internal/org` も `internal/cli` も import しないので、この branch の変更で結果は変わらない。それでも `-count=1` で回し直した(時間は回し直しのもの)。`internal/org` は golang pack の 1 パッケージの上限(30 分)の 50% で終わった。Go の段の 1 分 load は 9.3〜112.8 で、20:12〜20:14 に 100 を超えた。私の絞った run(AC の 2 つ、変異 3 つ、send の確かめ、残る取り違えの probe、coverage、base のテスト)はどれも 19:51〜20:07 の shell の段に重ね、Go の段には何も回していない。

### 変異(cycle 2)

どれも変異の写しと overlay の JSON を `$TMPDIR` の下に置き、`go test -count=1 -vet=off -timeout 4m -overlay <json> -v ./internal/org/ -run 'UsageLimit|Resume|WatchUsage|SendScreen'` で回した。追跡しているファイルは変えていない(`git status --porcelain` は空)。`build failed` はどの変異でも 0 件。対照の HEAD は同じ絞り込みで rc 0(上の表)。

| 変異 | 場所 | 結果 | 落ちたテスト |
| --- | --- | --- | --- |
| M1: 状態行の始まりを確かめる早い return を消す | `usageLimitRetryingTurn`(`internal/org/usage_limit_screen.go:178-180`) | rc 1(48 s) | `TestUsageLimitScreen_RetryWaitShapes` の 3 subtest だけ。返答の引用の 3 行(`usage_limit_screen_test.go:766`・`:771`・`:776`)で、どれも `("⎿ You've hit your weekly limit · resets Nov 21 at 3pm (Asia/Tokyo) ✻ Weekly li… · Retrying in 29m (Nov 21 at 3pm) · attempt 1/55", true)` を返した。期待は `("", false)` |
| M2: 時刻の一致を常に真にする(cycle 1 の変異) | `usageLimitRetryingTurn` の `if strings.Contains(retry.text, "("+limit.reset+")") {`(`:189`)を `if true {` に | rc 1(41 s) | `TestUsageLimitScreen_RetryWaitShapes` の 5 subtest。cycle 1 と同じ 5 行(`:630`・`:644`・`:713`・`:719`・`:724`) |
| M3: `claudeStatusSymbols` から `✻` を外す | `internal/org/send_screen.go:145` | rc 1(83 s) | 16 関数、42 subtest。issue の fixture の行は、`TestUsageLimitScreen_EveryFixture/claude-weekly-limit-retrying.txt`、`TestUsageLimitScreen_Issue353RetryWaitIsAUsageLimit`、`TestWatchUsageLimit_AClaudeSeatRetryingUntilItsWeeklyLimitResetsIsUsageLimited` の 2 subtest(implementer と leader)、`TestOrgResume_UsageLimitWait_TheReasonNamesTheLimit` の再試行の 4 行。ほかに `RetryWaitShapes` の肯定の 9 行、`TestUsageLimitScreen_ClaudeStatusLineStart`、send の側の `TestClassifySendScreen_EveryFixture` の claude の 10 fixture、`TestClassifySendScreen_Edges` の 3 行(`✻` の busy の行と手番の終わりの行)、`TestOrgSendScreen_*` の 8 関数 |

M1 は依頼どおり 3 行だけで落ちた。見張り(`TestWatchUsageLimit_*`)と resume(`TestOrgResume_*`)のテストは M1 で 1 つも落ちない。cycle 1 の Test gaps の 1 と同じ形で、返答の引用の否定は `UsageLimitScreen` の単体の表だけが固めている(下の Test gaps の 1)。

M3 で `TestClassifySendScreen_EveryFixture/claude-weekly-limit-retrying.txt` は落ちない。send はこの画面を `Retrying in` の待ちの文で busy と読み、記号は見ないため。M3 は usage limit の側と send の側の両方のテストで落ちるので、記号の組を 1 か所にまとめたことは両側から固められている。

### send の busy と手番の終わりの判定(cycle 2)

`51fe7ddf` のあとも変わっていない。確かめたのは次の 3 つ。

1. HEAD で send の画面のテスト 26 関数(`--- PASS` 198 行)が通る(上の org の絞り込み)
2. base `a672b72a` の `send_screen_test.go` を HEAD に `-overlay` で当てて `-run 'Probe_ClaudeRegexps|ClassifySend|SendScreen|SendProbe|SendInputHolds'` を回すと、落ちるのは `TestClassifySendScreen_EveryFixture` だけで、理由は `fixture testdata/resume/claude-weekly-limit-retrying.txt has no expectation in sendScreenFixtureWants`(新しい fixture の行が base の表にない)。ほかの fixture と `TestClassifySendScreen_Edges` を含めて `--- PASS` 296 行
3. 同じ run に overlay で足した使い捨ての probe テストで、`sendClaudeBusyLine.String()` と `sendClaudeTurnEnd.String()` が base の正規表現の文字列 `` ^[*·✢✳✶✻✽] \S+…(?:\s|$) `` と `` ^[*·✢✳✶✻✽] \S+ for \d[^·]*· done \d `` に等しいことを確かめた(PASS)。verify の python の照合と同じ結論

### base のテストを HEAD に当てる(AC5、cycle 2)

cycle 1 と同じ 6 つのテストのファイルを、base `a672b72a` の版で HEAD のコードに当てた(6 つは `1d05c9ac` と `a672b72a` で同じ)。

- org(`-run 'UsageLimit|Resume|WatchUsage|ClassifySend|SendScreen|SendProbe|SendInputHolds|FindResumeInputBox'`): `--- PASS` 668 行、落ちたのは cycle 1 と同じ 3 つ。`TestOrgResume_RetryingOrWorking_Waits/working` の呼び出しの列(AC5 が名指す意図した変更)と、`TestUsageLimitScreen_EveryFixture` と `TestClassifySendScreen_EveryFixture` の新しい fixture の期待の不足
- cli(`-run 'Resume'`): rc 0、`--- PASS` 16 行

### 残る取り違えの確かめ(cycle 2)

`a5ec5464` が `usageLimitRetryingTurn` のコメントに書いた残る取り違えが、書かれたとおりに起きるかを使い捨ての probe(overlay だけ)で見た。返答の段落が引用した再試行の行で始まり、その上の段落に時刻の合う上限の引用がある画面と、`⏺` のブロックに上限があり、そのあとの `* ` で始まる段落が再試行の語と時刻を持つ画面の 2 つ。どちらも `ClassifyResumeScreen` は waiting、`UsageLimitScreen` は上限の行と真を返した。コメントの記述と合う。この形を固めるテストの行はない(下の Test gaps の 2)。

### Coverage(cycle 2)

- Statement: 狭い run(`go test -count=1 -coverprofile -run 'UsageLimit|Resume|WatchUsage|SendScreen|ClassifySend' ./internal/org/`)の profile で測った。`51fe7ddf` が足した文は 7 つ(`usage_limit_screen.go:178-179` の if と return、`claudeStatusLineStart` の 5 つ)で、7 つとも実行された。`send_screen.go` の変更はパッケージの変数の初期化なので、profile の文に入らない
- Branch: Go の標準ツールでは測れない
- Function: `claudeStatusLineStart` 100%、`UsageLimitScreen` 100%、`usageLimitRetryingTurn` 94.7%(cycle 1 は 94.1%)、`ClassifySendScreen` 100%
- Notes: 実行されない 1 文は cycle 1 と同じ `usage_limit_screen.go:171-172`(入力欄が見つからないときの `return "", false`)

### Failure analysis(cycle 2)

| Test | Error | Root cause | Proposed fix |
| --- | --- | --- | --- |
| `TestGate_TwoGatesRunTheirScriptsOneAtATime`(`internal/org/gate_lock_gate_test.go:219`) | 待った側の gate の log の行が `# waited 2s for gate.lock: a gate whose marker is not written yet` で、期待の ` for gate.lock: ralph org gate of org_id "org-a", started ` を含まない | この branch の外にある gate.lock の待ちの競合(下の段落) | `acquireGateLock` の待ちの callback で、名前の読めた持ち主を空の marker の文で上書きしない。この branch では直さない |

先の gate は解放のとき、marker を空にしてからロックを外す(`gateLock.release`、`internal/org/gate_lock.go:153-159`)。待つ側は 20 ms ごとの poll で marker を読み直し、「最後に見た持ち主」を毎回上書きする(`acquireGateLock`、`:102-103`)。空にしてからロックを外すまでのあいだに poll が入ると、最後に見た持ち主が「marker がまだ書かれていない」になり、log にもそう書かれる。このテストが落ちたのは run-test.sh の Go の段で、1 分 load は 9.3〜112.8 だった。

この branch と関係がないと判断した根拠は 3 つある。

1. gate のコードとこのテストは base から変わっていない(`git diff --quiet a672b72a HEAD -- internal/org/gate_lock.go internal/org/lockfile.go internal/org/gate_lock_gate_test.go internal/org/gate.go internal/org/gate_run.go` が真)
2. 単独で `-count=20` は 20/20 PASS。run-test.sh のあとに回した `internal/org` のパッケージ全体の打ち直し(`go test -count=1 -timeout 30m ./internal/org/`)も ok(715.576 s、1 分 load 6.0〜13.9)
3. `release` の、marker を空にする書き込みとロックを外す呼び出しのあいだに 200 ms の sleep を overlay で入れると、`-count=5` で 5/5 が同じ文(`a gate whose marker is not written yet`)で落ちる

`docs/tech-debt/README.md` にも issue にも、この競合の記録はまだない(`grep` で `TwoGates`・`not written yet` とも 0 件)。tech-debt の行か issue にすることを勧める。

### Regression checks(cycle 2)

| Previously broken behavior | Status | Evidence |
| --- | --- | --- |
| cross-review cycle 1 の ACTION_REQUIRED #1: issue #353 の 2 行を引用した返答を、上限の解除を待つ画面と読む | 直った | 返答の引用の 3 行(`:766`・`:771`・`:776`)が PASS。M1 でこの 3 行が落ちる |
| issue #353(AC1〜AC3) | 直ったまま | AC の 2 つの絞り込みが PASS。M3 で issue の fixture の行が落ちる |
| 通信断の再試行と、時刻の合わない画面を上限と読まない | 保たれている | M2 が cycle 1 と同じ 5 行で落ちる |
| send の busy と手番の終わりの判定 | 変わらない | 上の「send の busy と手番の終わりの判定」 |
| cycle 1 の Regression checks の残りの行 | 保たれている | 同じテストが AC の絞り込みと、base のテストを当てた run で PASS |

### Test gaps(cycle 2)

1. 返答の引用の否定(M1)は、`UsageLimitScreen` の単体の表の 3 行だけが固める。見張りと resume のテストには issue の 2 行を引用した返答の画面がなく、M1 でも落ちない。両方とも `UsageLimitScreen` の答えをそのまま使うので、cycle 1 の 1 と同じ理由で今のままでよいと考える(LOW)
2. 残る取り違え(記号と空白で始まる返答の段落)を固めるテストの行はない。上の probe で、今は上限と読むことを確かめた。挙動はコメントの記述と合う。記録はコメントにしかなく、verify の報告は「/sync-docs に渡すもの」の 5 で register に書くよう勧めている(self-review Run 2 の L7)
3. cycle 1 の 2〜6 は変わらない。実機の画面の全体がない、再試行が解除の時刻より先に尽きたあとの画面、待ちのあいだの `herdr agent get` の出力、`ralph org watch --help` の句を固めるテスト、入力欄が見つからないときの return
4. `TestGate_TwoGatesRunTheirScriptsOneAtATime` は負荷の下で落ちうる(上の Failure analysis)。この branch の外の問題

## Verdict

- Pass: `./scripts/run-test.sh` の shell 46 本と Go 10 パッケージ(`(cached)` の 6 つは `-count=1` で回し直した)、`internal/org` のパッケージ全体の打ち直し、AC の 2 つの絞り込み(org 96 関数、cli 118 関数)、依頼の変異 3 つがどれも落ちること(M1 は返答の引用の 3 行、M2 は cycle 1 と同じ 5 行、M3 は issue の fixture の行と send の行)、send の判定が変わらないこと、base のテストを HEAD に当てて落ちるのが cycle 1 と同じ 3 つだけであること
- Fail: 0(run-test.sh は rc 1 だった。落ちた `TestGate_TwoGatesRunTheirScriptsOneAtATime` は、この branch が触っていない gate.lock の待ちの競合によるもので、単独の 20 回とパッケージ全体の打ち直しで通った。上の Failure analysis)
- Blocked: 0
- Verdict: pass。/sync-docs に進んでよい。run-test.sh は依頼どおり 1 回だけ回したので、rc 0 の run-test.sh はこの cycle にない。この扱いを受け入れない場合は、run-test.sh をもう 1 回回すかを人が決める

## Cycle 1(HEAD 87693818、報告の commit f246da8f)

- Date: 2026-11-20(パイプラインの cycle 1。`cycle-count.json` は `cycle: 1`)
- Plan: `docs/plans/active/2026-11-20-watch-usage-limit-retry.md`(issue #353)
- Tester: tester サブエージェント(`/test`)
- Scope: HEAD `87693818`(base `1d05c9ac`、worktree は clean)。`./scripts/run-test.sh` を 1 回、worktree のルートから回した。Bash tool の 10 分の上限を超えるので、perl の fork と setsid で切り離して起動し(INT・HUP・QUIT・TERM は既定の扱いに戻し、標準入出力は `/dev/null`)、`caffeinate -w` で終わりを待った。範囲は wrapper の既定の changed(`- Requested scope: changed`)で、fixture の `.txt` が言語に分類されないため `Language scope: full fallback (unclassified:internal/org/testdata/resume/claude-weekly-limit-retrying.txt)` になった。この repo で見つかる言語は golang だけなので、走る pack は changed のときと同じ。そのあと、計画の AC のテストを 2 つの絞り込みで回し、verify の報告が勧めた変異 1 個、base のテストのファイルを HEAD のコードに当てる run(AC5)、狭い coverage の run を足した
- Evidence: `docs/evidence/test-2026-11-20-watch-usage-limit-retry.log`(9,346 行。`.gitignore` の `docs/evidence/*.log` に当たるので commit されない)。run-test.sh の出力、AC の 2 つの run の `-v` の出力、変異の diff と出力、base のテストの run の出力、coverage の要約が入っている。run-verify.sh は自分の evidence を `docs/evidence/verify-2026-11-20-083353.log` にも書いた(同じく追跡外)

### Test execution

| Suite / Command | Tests | Passed | Failed | Skipped | Duration |
| --- | --- | --- | --- | --- | --- |
| `./scripts/run-test.sh` | 下の 2 行の合計 | 全部 | 0 | 2(条件つきの skip、下の行) | 2026-11-20 17:33:52 〜 18:04:37 JST、1,845 s、rc 0。開始時の 1 分 load 28.3、終了時 8.3 |
| local verifier(`tests/test-*.sh` 46 本) | 46 本、`PASS` で始まる行 5,390 | 46 本(`OK` 46 行) | 0(`FAIL:` の要約行 29 本はどれも `FAIL: 0`) | `test-plan-visual.sh` の browser detection(`/Applications` にブラウザーがある)、`test-secret-scan-branch.sh` の「git 2.41 より古い本物の git」の場合(手元は git 2.49.0、stub の古い git の行は通る) | 約 18 分(17:33:52 〜 17:51:42)。この段の 1 分 load は 17:47 に 224 まで上がった(ほかのセッションの Xcode のビルド) |
| `go test ./...`(テストのある 11 パッケージ) | 11 | 11 | 0 | なし。`(cached)` は 0 | 約 13 分(17:51:42 〜 18:04:37)、下の表 |
| `go test -count=1 -v ./internal/org/ -run 'UsageLimit\|Resume\|WatchUsage'` | 69 関数、`--- PASS` 424 行(subtest を含む) | 全部 | 0 | 0 | 12 s(パッケージ 10.7 s) |
| `go test -count=1 -v ./internal/cli/ -run 'Resume\|OrgWatch'` | 70 関数、`--- PASS` 148 行 | 全部 | 0 | 0 | 80 s(パッケージ 78.7 s) |

cli の絞り込みは依頼では `-v` なしだったが、関数の数を数えるために `-v` を付けた。絞り込みと `-count=1` は依頼のとおり。

| パッケージ | 結果 | 時間 |
| --- | --- | --- |
| `cmd/ralph` | ok | 0.726 s |
| `internal/cli` | ok | 651.382 s |
| `internal/config` | ok | 17.137 s |
| `internal/gitroot` | ok | 3.477 s |
| `internal/insights` | ok | 2.419 s |
| `internal/org` | ok | 755.975 s |
| `internal/org/driver` | ok | 3.767 s |
| `internal/org/protocol` | ok | 2.360 s |
| `internal/scaffold` | ok | 2.173 s |
| `internal/toolchain` | ok | 20.798 s |
| `internal/upgrade` | ok | 3.244 s |

`internal/org` は golang pack の 1 パッケージの上限(30 分)の 42% で終わった。Go の段の 1 分 load は、17:53 から 18:04 までの 5 回の測りで 8.3〜19.9。私の絞った run(AC の run、変異、base のテスト、coverage)は 17:34〜17:44 の shell の段に重なり、Go の段には何も回していない。負荷に弱いと分かっているテストを含め、落ちたテストはなく、打ち直しもしていない。

#### 計画のテストと AC の対応

| AC | テスト | 結果 |
| --- | --- | --- |
| AC1 | `TestUsageLimitScreen_Issue353RetryWaitIsAUsageLimit`、`TestUsageLimitScreen_EveryFixture/claude-weekly-limit-retrying.txt`、`TestClassifyResumeScreen_Fixtures/claude_retrying_until_its_weekly_limit_resets_is_waiting`(org) | PASS。issue の画面を元にした fixture を上限と読み、`⎿ You've hit your weekly limit · resets Nov 21 at 3pm (Asia/Tokyo)` を返す |
| AC2 | `TestWatchUsageLimit_AClaudeSeatRetryingUntilItsWeeklyLimitResetsIsUsageLimited`(org、implementer と leader の 2 subtest) | PASS。stall の ALERT を出さず、`usage_limited` の件 1 つ(`REASON: usage_limit`、`SEATS:`、本文の上限の行)を上げる |
| AC3 | `TestOrgResume_UsageLimitWait_TheReasonNamesTheLimit`(org、14 subtest)、`TestOrgResume_ASeatWaitingForItsUsageLimitToResetShowsTheLimit`(cli、`resume` と `resume --dry-run`) | PASS。14 subtest は、working と idle の再試行の画面、5 時間の上限の footer、pane の読みの失敗、tool の呼び出し中、上限で止まった画面(週と session)の 7 形を、dry run の真と偽で回す。cli は出力の行、pane ごとに読みが 1 回、send と `resumed` がないことを確かめる |
| AC4 | `TestUsageLimitScreen_RetryWaitShapes`(org、26 subtest) | PASS。AC4 の 7 形は verify の報告の対応のとおり、「an API retry status line alone」「a limit only before the last prompt, …」「a limit in the middle of a reply line …」「a limit in the turn above a turn-end status line, with no retry」(止まった読み方の対照は「a limit in the turn that is the stop, with no retry」)「a tool output that prints the limit, then a network retry with no time」(6 番目の形もこの行が覆う)「no prompt on the screen and a retry whose time differs」の行に当たる。codex の行も偽 |
| AC5 | 下の「base のテストを HEAD に当てる」、fixture の 3 つの表、`TestOrgResume_RetryingOrWorking_Waits` | PASS(注つき)。今あるテストの判定は変わらない。base のテストで落ちるのは、AC5 が名指す呼び出しの列の変更と、新しい fixture の行が base の表にないことの 2 種だけ |
| AC6 | 文書と `check-sync.sh`・`check-skill-sync.sh` | /test の範囲の外。verify の報告で 2 つの検査が通っている |

#### 変異: 時刻の一致を外す(verify の報告の勧め)

`usageLimitRetryingTurn`(`internal/org/usage_limit_screen.go:172`)の `if strings.Contains(retry.text, "("+limit.reset+")") {` を `if true {` に替え、「手番の中に上限の文があり、入力欄のすぐ上が再試行の行なら上限と読む」形にした。変異の写しと overlay の JSON は `$TMPDIR` の下に置き、`go test -count=1 -vet=off -timeout 4m -overlay <json> -v ./internal/org/ -run 'UsageLimit|Resume|WatchUsage'` で回した。対照の HEAD は同じ絞り込みで rc 0(上の表の行)。

結果は rc 1(`build failed` は 0 件)で、`TestUsageLimitScreen_RetryWaitShapes` の 5 つの subtest が落ちた。

| subtest | 変異での戻り値 | 期待 |
| --- | --- | --- |
| a tool output that prints the limit, then a network retry with no time | `("⎿ You've hit your weekly limit · resets Nov 21 at 3pm", true)` | `("", false)` |
| a retry whose time differs from the limit's reset | `("⎿ You've hit your weekly limit · resets Nov 21 at 3pm (Asia/Tokyo)", true)` | `("", false)` |
| no prompt on the screen and a retry whose time differs | 同上 | `("", false)` |
| of two limits in the turn, the one whose reset the retry shows | 最後の上限(`Nov 28 at 9am`)を返す | 再試行の時刻と一致する `Opus limit`(`Nov 21 at 3pm`)の行 |
| of two limits in one block, the first, to the block's end | 同上 | ブロックの最初の上限からブロックの終わりまで |

計画が一番恐れた取り違え(tool の出力が引用した上限の文のあとの通信断の再試行を上限と読む)と、時刻が合わない画面は、テストで守られている。見張り(`TestWatchUsageLimit_*`)と resume(`TestOrgResume_*`)のテストはこの変異で 1 つも落ちなかった。どちらも `UsageLimitScreen` の答えをそのまま使うので、否定の形は `UsageLimitScreen` の単体の表だけが固めている(Test gaps の 1)。

#### base のテストを HEAD に当てる(AC5)

この branch が変えた 6 つのテストのファイル(`internal/cli/org_resume_test.go`、`internal/org/resume_screens_test.go`、`resume_test.go`、`send_screen_test.go`、`usage_limit_screen_test.go`、`watch_usage_limit_test.go`)を base `1d05c9ac` の版に `-overlay` で差し替え、HEAD のコードに当てた。

- org(`-run 'UsageLimit|Resume|WatchUsage|ClassifySend|SendScreen|SendProbe|SendInputHolds|FindResumeInputBox'`): `--- PASS` 668 行、落ちたのは 3 つ
  - `TestOrgResume_RetryingOrWorking_Waits/working`: `herdr calls = ["get org-a_implementer" "read pane-1 60"], want ["get org-a_implementer"]`。AC5 が名指す意図した変更(working の座席の pane を 1 回読む)
  - `TestUsageLimitScreen_EveryFixture` と `TestClassifySendScreen_EveryFixture`: どちらも `fixture testdata/resume/claude-weekly-limit-retrying.txt has no expectation in …Wants` だけで、既存の fixture の subtest はすべて通る。新しい fixture をディレクトリから拾うのに、base の表にその行がないため。判定の変更ではない
- cli(`-run 'Resume'`): rc 0、`--- PASS` 16 行

今ある見張りのテスト(`watch_usage_limit_test.go` の base の 14 関数)と、止まった画面と 5 時間の上限の footer の読み方のテストは、base の版のまま通る。

### Coverage

- Statement: 狭い run の profile で測った(パッケージ全体の profile は取っていない)。`git diff -U0 1d05c9ac HEAD` で足された行に当たる文は、`usage_limit_screen.go` 51/52、`resume.go` 19/19、`internal/cli/org_resume.go` 2/2、`internal/cli/org_watch.go` 1/1。関数では `UsageLimitScreen`、`usageLimitTurnBlocks`、`usageLimitResetsIn`、`usageLimitResetTime`、`resumeLook`、`resumeWorkingReason`、`resumeTargetText` が 100%、`usageLimitRetryingTurn` が 94.1%
- Branch: Go の標準ツールでは測れない
- Function: 上の通り。変更のない関数の数字は狭い run のものなので載せない
- Notes: 実行されない 1 文は `usage_limit_screen.go:157-158`(入力欄が見つからないときの `return "", false`)。doc comment のとおり、`UsageLimitScreen` からはここに来ない(先に `ClassifyResumeScreen` が waiting と読んでいるので、入力欄はある)。直接呼ぶテストはない

### Failure analysis

| Test | Error | Root cause | Proposed fix |
| --- | --- | --- | --- |
| なし | — | — | — |

### Regression checks

| Previously broken behavior | Status | Evidence |
| --- | --- | --- |
| issue #353: 手番の中で再試行して週の上限の解除を待つ claude の座席が stall の件になり、`resume --dry-run` が `herdr reports working` と出す | 直った | AC2 と AC3 のテストが PASS |
| 通信断の再試行(2026-11-07 の evidence の 3 の形)を上限と読まない | 保たれている | 「an API retry status line alone」が PASS。時刻の一致を外す変異で、tool の出力の引用のあとの通信断の再試行の行が落ちる |
| 5 時間の上限の footer の待ち(#329)を上限と読む | 保たれている | `TestUsageLimitScreen_EveryFixture/claude-5h-limit-continuing.txt`、resume の「idle, continuing automatically when the 5-hour limit resets」の 2 行、base の `TestUsageLimitScreen_WaitingForALimitToReset` が PASS |
| herdr が working で上限で止まった画面の座席は、理由が `herdr reports working` のまま(self-review M1) | 保たれている | 「working, the screen stopped on the weekly limit」と「… session limit」の 4 行が PASS |
| working の座席の pane の読みが失敗したら、理由は `herdr reports working` のまま | 保たれている | 「working, the pane read fails」の 2 行が PASS |
| 上限と読めない画面の座席は今までどおり stall になる | 保たれている | base の `TestWatchUsageLimit_ASeatWhoseScreenIsNoUsageLimitStallsAsBefore` と `…ASeatWhoseReplyQuotesTheWaitKeepsItsStall` が PASS |
| codex の読み方 | 変わらない | 「a codex limit and a retry that shows its reset time」が偽、fixture の表の codex の行と base の表がすべて PASS |

### Test gaps

1. 否定の形(通信断の再試行、時刻が合わない画面)は `UsageLimitScreen` の単体の表だけが固める。見張りと resume のテストは肯定の形しか持たないので、時刻の一致を外す変異でも落ちない。両方とも `UsageLimitScreen` の答えをそのまま使うので、呼び出し側で否定の形を足しても新しい分岐には届かない。今のままでよいと考える(LOW)
2. 実機の画面の全体はない。fixture は issue の断片と別の fixture の入力欄と footer の組み立てで、再試行の行の括弧の時刻と上限のリセットの時刻が同じことは 1 つの画面でしか見ていない(計画の Assumptions、register 515 行)。形の違う版では今までどおり stall になる
3. 再試行が解除の時刻より先に尽きたあとの画面と、そのときの resume の出し方は試していない(verify の報告の INFO の 3 つ目)
4. 待ちのあいだ `herdr agent get` の出力が変わらないか(見張りの道が stall の立つ座席でしか走らないこと)は、issue の報告から推しただけで、テストでも実機でも見ていない(verify の V4 の 516 行)
5. verify で直した `ralph org watch --help` の新しい句を固めるテストはない(register 517 行の (d))。help の文の変更で、挙動は変わらない
6. `usageLimitRetryingTurn` の入力欄が見つからないときの return は、どのテストからも実行されない(Coverage の Notes)

### Verdict(cycle 1)

- Pass: `./scripts/run-test.sh` rc 0(shell 46 本、Go 11 パッケージ)、AC の 2 つの絞り込み(org 69 関数、cli 70 関数)、時刻の一致を外す変異が 5 つの subtest で落ちること、base のテストを HEAD に当てて落ちるのが意図した変更と新しい fixture の行の不足だけであること
- Fail: 0
- Blocked: 0
- Verdict: pass。/sync-docs に進んでよい
