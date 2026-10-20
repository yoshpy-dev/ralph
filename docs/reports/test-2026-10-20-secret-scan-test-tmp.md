# Test report: secret-scan-test-tmp

- Date: 2026-10-20(JST 11:01〜11:36)
- Plan: docs/plans/active/2026-10-20-secret-scan-test-tmp.md(tech-debt 170 行。`tests/test-secret-scan.sh` の capture を実行ごとの dir に移す)
- Tester: tester subagent (Claude Opus 5.5)。パイプラインの 1 回目
- Scope: HEAD 74598215(base e333200c)。コードの変更は `tests/test-secret-scan.sh` だけ。behavioral test だけを実行した(静的解析は /verify の担当)。環境は macOS(arm64、Darwin 25.5.0)、`/bin/sh` は bash 3.2.57、git 2.49.0、go1.26.0。load average(1 分)は開始時 26、run-test.sh の間は 8〜10
- Evidence: `docs/evidence/test-2026-10-20-secret-scan-test-tmp.log`(下の各実行の生の出力。`mktemp` の suffix と `$HOME` は伏せた)と、run-test.sh が書いた `docs/evidence/verify-2026-10-20-021809.log`。どちらも `docs/evidence/*.log` の gitignore の対象で、コミットしない
- 追跡しているファイルは変えていない。旧版と変えた写しは scratchpad に置いた。`git status --porcelain` は実行の前もあとも空だった。各実行は順に 1 つずつ打ち、重ねていない

## Test execution

| Suite / Command | Tests | Passed | Failed | Skipped | Duration |
| --- | --- | --- | --- | --- | --- |
| 1. 単独(AC3)`sh tests/test-secret-scan.sh` | 123 | 123 | 0 | 0 | 178 秒、rc 0 |
| 2. 2 つ同時(AC4)。計画の書き方(PID を取り、`wait` を別々に)を 3 回 | 3 回 × 2 本 | 6 本とも 123 | 6 本とも 0 | 0 | 154 / 121 / 72 秒。3 回とも `ra=0 rb=0` |
| 3. 2 の書き方の確かめ。片方を `sh -c 'exit 3'` に置き換える | 1 回 | script の側 123 | 0 | 0 | 59 秒。`rc=0 rd=3` |
| 4. 旧版の対照。e333200c の script を、2 本目を 0.2 秒遅らせて 2 回 | 2 回 × 2 本 | 先の本は 123、後の本は 121(2 回とも) | 後の本が 2(2 回とも) | 0 | 2 回で 144 秒。2 回とも `ra=0 rb=1` |
| 5. TERM。`TMPDIR=<空の scratch dir>` で走らせ、3 秒後に `kill -TERM`(t3 だけ 8 秒後)を 3 回 | 3 回 | - | - | - | 終了コードは 1 / 1 / 128。3 回とも、終わった直後と 2 秒後に scratch の TMPDIR は空 |
| 6. `GOFLAGS=-timeout=30m ./scripts/run-test.sh`(changed。`tests/test-secret-scan.sh` は言語に分類されず full に落ち、golang を選んだ) | shell 41 本、Go 8 パッケージ | shell 41 本すべて OK、Go 8 パッケージすべて ok | 0 | shell 1(下) | 749 秒、rc 0 |
| 追加 A。HEAD の script の写しで 3 つの check をわざと落とす(失敗の時の出力) | 123 | 120 | 3(狙いどおり) | 0 | 59 秒、rc 1 |
| 追加 B。相対の `TMPDIR=rt`(verify の V4)で、HEAD と base を 1 本ずつ | 2 本 | - | 2 本とも途中で終わる | - | 合わせて 2 秒、どちらも rc 1 |

- 1 の数は、implementation の evidence にある変更前の単独の数(123)と同じ。集計の行のほかに `  PASS:` の行が 123 あり、`  FAIL:` の行はない
- 2 では、各回の前とあとで `$TMPDIR` の下の `ralph-secret-scan*` が 0 件だった。終了時の trap が capture dir と fixture の dir をどちらも消している
- 3 では失敗する側を後ろに置いた。script を 59 秒 `wait` したあとで、先に終わっていた子の終了コード 3 を拾えている。逆の順(先に `exit 3`、後ろに `sleep 1; exit 0`)でも `ra=3 rb=0` だった。2 の書き方は、どちらの側の失敗も取りこぼさない
- 4 で落ちた check は回ごとに違い、4 件とも stderr を読む check だった。1 回目は「attribute source: an unresolved name is named in the reason」と「AC-7: a non-ASCII name is labeled unquoted」、2 回目は「AC-15: unresolved range names the unscanned range」と「staged: a renamed file is labeled by its new name」で、どれも `missing in stderr`。固定の `/tmp/ralph-secret-scan-test.err` を片方が書き換えるか消すという、row 170 の記録と同じ壊れ方である。実行の前とあとに `/tmp/ralph-secret-scan-test.out` と `.err` がないことを確かめ、ほかの session の旧版と混ざっていないことも見た。2 回とも片方が落ちたが、時間に左右される結果なので、毎回落ちるとは言えない
- 5 では、TERM の前の scratch の TMPDIR に `ralph-secret-scan-test.<suffix>`(capture dir)と `ralph-secret-scan.<suffix>`(fixture の repository)があり、t1 と t2 ではそれに scanner 自身の `ralph-secret-scan.<pid>` も加わっていた。trap は `exit` を打たないので、script は消えた working directory で次のコマンドが失敗するまで進み、`set -e` で終わる。t1 は `rm: .gitallowed: No such file or directory` で exit 1、t2 は 76 行の `mkdir "$range_dir"` の失敗で exit 1(macOS の mkdir は、ないのが親の dir のときに親の名前を出す)、t3 は `fatal: Unable to read current working directory` で exit 128。implementation の evidence(128 と 1)と同じ型で、trap に `exit` がないのは base と同じ形である。計画の Edge cases は SIGINT と書くが、非対話の sh は background の job の SIGINT を無視するので、指示どおり TERM で打った
- 6 の Go は cli(228.0 秒)、config(0.791 秒)、org(314.9 秒)を走らせ、insights、org/driver、org/protocol、scaffold、upgrade は `(cached)` だった。この差分は Go のファイルを変えていないので、cached の結果は古くならない。shell の Skipped は前からある `tests/test-secret-scan-branch.sh` の 1 件で、git 2.49.0 なので 2.41 より古い本物の git の場合を飛ばしている。run-test.sh の中の `tests/test-secret-scan.sh` も 123/0 だった。記録にある Go の揺れ(`TestRegate_FetchFailures/the_fetch_runs_past_the_bound`、`TestStopDirectorProcess_AProcessThatIgnoresSIGTERMIsNotEndedInTime`、doctor の 2 本)は、この回では出なかった
- 追加 A では 3 か所を変えた。60 行の `expect_exit` の期待する終了コード(1 を 7 に)、233 行の `expect_stderr_contains` の探す文字列(stderr にない文字列に)、234 行の `expect_stderr_not_contains` の探す文字列(stderr にある `could not scan main..feature` に)。3 つとも FAIL になり、`expect_exit` の失敗は `--- stderr ---` の下に scanner の `BLOCKED` の出力を、stderr の check の 2 つは scanner の stderr(`could not scan main..feature: git log exited with 128, ...`)を出した。1 と 2 では check が 1 つも落ちないので、失敗の時に capture を出す `cat "$capdir/out"` と `cat "$capdir/err"` の 3 行は、この回で初めて通った。実行のあと `$TMPDIR` に `ralph-secret-scan*` は残っていない
- 追加 B は Test gaps の G2 に書いた

## Coverage

- Statement: shell には instrumented な coverage の道具がない。変わった行はすべて通った。capdir を作る行と trap の行は毎回、check の出力を capture に向ける redirect の行は check のたびに、失敗の時に capture を出す `cat` の 3 行は追加 A で通った
- Branch: 測っていない。trap は EXIT(1、2、追加 A の終了時)と TERM(5)で通した。HUP と INT は通していない(G3)
- Function: 同上
- Notes: AC4 を確かめたのは suite の外の harness で、run-test.sh は 2 つ同時の実行をしない(G1)

## Failure analysis

| Test | Error | Root cause | Proposed fix |
| --- | --- | --- | --- |
| なし | - | 1〜3、5、6 で失敗したテストはない。4 の旧版の `FAIL: 2` はこの計画が直す不具合を base で再現した対照で、追加 A の 3 件は失敗の時の出力を見るためにわざと出した失敗である。どちらも HEAD の不具合ではない | - |

## Regression checks

| Previously broken behavior | Status | Evidence |
| --- | --- | --- |
| 2 つ同時に走らせると、固定の capture のパスを互いに上書き・削除し、stderr を読む check が落ちる(row 170。この計画が直す不具合) | 直っている。HEAD は 3 回とも両方 exit 0 で `FAIL: 0`、base は 2 回とも片方が `FAIL: 2` | 2 と 4 |
| 単独の実行の数が変わらない(R1) | pass。123/0 で変更前と同じ | 1 |
| capture dir を `$workdir` より前に作っても、signal の check が通る(R2) | pass。「a TERM during the scan exits 143」と「a TERM during the scan still removes the temp dir」が 1、2、6 のすべての実行で PASS | 1、2、6 |
| 全体(shell 41 本、Go 8 パッケージ) | pass | 6 |

## Test gaps

- G1(LOW): capture を固定のパスに戻す変更を止める check が suite にない。AC4 を確かめたのは suite の外の harness で、run-test.sh は `tests/test-secret-scan.sh` を 1 本しか走らせない。capture を `/tmp/<固定の名前>` に戻しても、単独の実行と run-test.sh は緑のままになる。base の script が単独では 123/0 になること(implementation の evidence)がその例である。止めるなら、verify の Coverage gaps の提案と同じく、`tests/*.sh` の中の固定の `/tmp/<名前>` への redirect を grep で断る check を `scripts/verify.local.sh` に足す。2 つ同時の実行をテストにすると 1 回 1〜3 分かかるので、grep の方が安い。計画の Non-goals の調べでは、ほかの suite が使う固定のパスはない
- G2(INFO): 相対の `TMPDIR` では、HEAD も base も途中で終わって dir を残す。verify の V4 は読んだだけだったので、走らせて確かめた。HEAD は 1 つ目の check の redirect(`rt/ralph-secret-scan-test.<suffix>/out: No such file or directory`)で落ち、capture dir と fixture の dir の 2 つを残した。base は 5 つ目の check の前の `synthetic.diff` への redirect で落ち、fixture の dir を 1 つ残した。どちらも rc 1 で、相対の `TMPDIR` は base のときから支えていない。違いは、落ちる場所が前に来たことと、残る dir が 1 つ増えたことである。`scripts/*.sh` と `.github/` には相対の `TMPDIR` を渡す箇所がない(grep で `TMPDIR=` を探し、当たったのは `scripts/install.sh` の `mktemp -d` の結果を入れる 1 行だけ)
- G3(INFO): HUP と INT の trap は通していない。INT は非対話の sh の background の job では無視されるので、この harness では送れない。4 つの signal は同じ trap の行に並ぶので、TERM で確かめた消し方と同じになるとみている。未確認
- G4(INFO): Linux(GNU の `mktemp`、dash の `/bin/sh`)では走らせていない。CI の verify(`ubuntu-latest` の `./scripts/run-verify.sh`)が 1 本を走らせるが、Linux での 2 つ同時の実行は確かめていない
- G5(INFO): 3 本以上の同時の実行は打っていない。capture dir は実行ごとの `mktemp -d` なので本数に依らないとみている。未確認

## Verdict

- Verdict: pass
- Pass: 1(AC3、123/0)、2(AC4、3 回とも `ra=0 rb=0`、6 本とも `FAIL: 0`)、3(`rd=3`)、5(3 回とも scratch の TMPDIR が空)、6(run-test.sh。shell 41 本と Go 8 パッケージ、749 秒、rc 0)。4 の旧版は 2 回とも片方が `FAIL: 2` で、不具合を base で再現した。追加 A の 3 件の失敗は狙いどおり
- Fail: なし
- Blocked: なし
- 残る穴: G1(LOW)。固定のパスに戻す変更を止める check がない。HEAD の script は正しく、check がないだけなので、マージを止めるものではない
