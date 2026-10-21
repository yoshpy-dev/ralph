# Test report: test-stability

- Date: 2026-10-22(JST 05:00〜05:40)
- Plan: `docs/plans/active/2026-10-21-test-stability.md`(試行の所見 F9・F10・根拠の表の G1、doctor の試験の行、PR #246 の test report の G1)
- Tester: tester subagent(Claude Opus 5.5)。パイプラインの 1 回目
- Scope: HEAD 37c01116(base 293dff17 = origin/main)。behavioral test だけを実行した(静的解析は /verify の担当)。環境は macOS 26.5.2(arm64、12 CPU)、`/bin/sh` は bash 3.2.57、git 2.49.0、go1.26.0 と go1.25.8。各実行の load average は「load」の節に書いた
- Evidence: `docs/evidence/test-2026-10-21-test-stability.log`(各実行の出力、mutant の差分と結果、load average。ホームは `~`、session の scratchpad は `<scratch>`、Go の一時ディレクトリは `<T>`、40 桁の commit hash は `<sha>`、U+FFFD は `<U+FFFD>` と書いた)と、run-test.sh が書いた `docs/evidence/verify-2026-10-21-200109.log`。どちらも `docs/evidence/*.log` の gitignore の対象で、コミットしない
- 追跡しているファイルは、この報告と insight event のほかは変えていない。mutant は `git archive` で scratchpad に出した写し(HEAD が 2 つ、base 293dff17 が 1 つ)に入れ、1 回ごとに HEAD の中身へ戻して、写しの全ファイルの sha256 が開始時と一致することを確かめた。scratch の `GOENV` だけを使い、利用者の go env のファイル(`~/Library/Application Support/go/env`)は全実行のあとも存在しない。worktree の `git status --porcelain --ignored` は、開始時と同じく `.harness/` と `docs/evidence/*.log` だけを示す

## Test execution

| # | Suite / Command | Tests | Passed | Failed | Skipped | Duration |
| --- | --- | --- | --- | --- | --- | --- |
| 1 | `env -u GOFLAGS ./scripts/run-test.sh`(worktree の `GOCACHE` のテスト結果を `go clean -testcache` で消してから) | shell 43 本、Go 8 パッケージ | shell 43 本すべて OK、Go 8 パッケージすべて ok | 0 | shell 1(下) | 773 秒、rc 0。cli 251.7 秒、org 314.2 秒、ほかは 0.4〜1.8 秒 |
| 2 | `env -u GOFLAGS GOTOOLCHAIN=go1.25.8 go test -count=1 -timeout 30m ./...` | 8 パッケージ | 8 | 0 | 0 | 332 秒、rc 0。cli 243.7 秒、org 327.3 秒、ほかは 0.8〜2.9 秒 |
| 3 | `go test -timeout 30m -count=20 -run 'TestRegate_FetchFailures/the_fetch_runs_past_the_bound' ./internal/org/`(go1.26.0) | 20 | 20 | 0 | 0 | 24 秒、rc 0 |
| 4 | 3 と同じ(`GOTOOLCHAIN=go1.25.8`) | 20 | 20 | 0 | 0 | 21 秒、rc 0 |
| 5 | doctor の 3 試験と `TestProbeBinary_TimeoutNamesTheBound` を `-count=10`(go1.26.0) | 40 | 40 | 0 | 0 | 17 秒、rc 0 |
| 6 | 5 と同じ(`GOTOOLCHAIN=go1.25.8`) | 40 | 40 | 0 | 0 | 30 秒、rc 0 |
| 7 | `sh tests/test-check-sync.sh`(go1.26.0 と go1.25.8) | 16 × 2 | 16 × 2 | 0 | 0 | 1〜2 秒、rc 0 |
| 8 | `sh tests/test-golang-pack-test-timeout.sh`(go1.26.0 と go1.25.8 を PATH の先頭に置いて) | 34 × 2 | 34 × 2 | 0 | 0 | 5〜6 秒、rc 0 |
| 9 | 8 を `PATH=/usr/bin:/bin`(go なし)で | 24 | 24 | 0 | 本物の go の事例(SKIP の行あり) | rc 0 |
| 10 | 2 つの suite に TERM を送る(専用の `TMPDIR`、HEAD と 6d990dfc^ の版) | 5 回 | HEAD の 3 回は rc 1 ですぐ止まる。6d990dfc^ の 2 回は L1 を再現(比較のため) | 0 | 0 | 下の表 |
| 11 | AC1〜AC5 の mutant と回帰の確かめ(scratch の写し) | 対照 2 つ、比較 1 つ(N2)、mutant 22 本 | 対照と N2 は緑 | 狙いどおり赤 20 本、生き残り 2 本(m6 は等価、D3 は G1) | - | 下の表 |
| 12 | `GOFLAGS` の追加の形を本物の go で(go1.26.0 と go1.25.8) | 9 × 2 | 9 × 2(期待どおり) | 0 | 0 | 下の表 |
| 13 | 追加: base と HEAD の F9 の subtest を繰り返す(負荷を変えて) | 6 回 | HEAD は 20/20 | base の stub では 60 回のうち 36 回が FAIL(下) | 0 | 下の表 |
| 14 | 追加: doctor の 5 試験(4 つ目に `TestRunDoctorOpts_HerdrAgmsgAbsent_ExitCodeUnaffected`)を負荷の下で | 50 × 2、15 | すべて | 0 | 0 | 下の表 |

- 1 の scope は changed で、`packs/languages/golang/README.md` が共有のファイルなので full になり、golang の pack だけを選んだ(`.harness/state/verify-scope` の `reason=shared:packs/languages/golang/README.md`、`languages=golang`)。`GOFLAGS` は環境になく、利用者の `go env GOFLAGS` も空なので、pack は `GOFLAGS=-timeout=30m` で `go test ./...` を走らせた(AC7 の「pack の bound」)。テストの cache を消したので、8 パッケージすべてが実際に走った。shell の 43 本には新しい 2 本が入っており、`scripts/verify.local.sh` が一覧の編集なしで拾うことも確かめた(計画の前提)
- 1 の shell の Skipped は前からある `tests/test-secret-scan-branch.sh` の 1 件で、git 2.49.0 なので 2.41 より古い本物の git の場合を飛ばしている
- 2 は命令に `-count=1` を足した。既定の `GOCACHE` に同じ中身の結果が残っていると `(cached)` になるためである
- 7 の check-sync の suite は go を呼ばないので、2 回の違いは環境だけである。8 の go1.25.8 は、toolchain の `bin` を PATH の先頭に置いた。suite の part B は `GOTOOLCHAIN=local` で本物の go を呼ぶので、`GOTOOLCHAIN=go1.25.8` を環境に置いても part B は go1.26.0 のままになる

### 10: TERM を送ったとき

`sh <suite> &` を専用の `TMPDIR` で起動し、決めた秒数のあと `kill -TERM` を送って `wait` した。

| 版 | suite | TERM まで | rc | 止まるまで | TMPDIR に残ったもの | TERM のあと |
| --- | --- | --- | --- | --- | --- | --- |
| HEAD | check-sync | 0.4 秒 | 1 | 0.56 秒 | なし | 事例 4 つのあとで止まり、Summary を出さない |
| HEAD | pack | 1 秒 | 1 | 1.08 秒 | なし | part A の 2 つ目の事例で止まる |
| HEAD | pack | 4 秒 | 1 | 4.64 秒 | なし | part B の scratch `GOENV` の事例のあとで止まる。TERM の前は TMPDIR に 3 つあった(suite の workdir、pack の roots の一時ファイル、go の build dir と見られる) |
| 6d990dfc^ | check-sync | 0.4 秒 | 1 | 0.54 秒 | なし | 消えた workdir の上で事例を続け、`sed: …/other.out: No such file or directory` のあと `set -e` で止まる |
| 6d990dfc^ | pack | 1 秒 | 1 | 1.38 秒 | なし | 消えた workdir の上で「go env -w value」の事例まで続けて FAIL を出し、`set -e` で止まる |

self-review の L1(信号のあとで suite が再開する)は、6d990dfc^ の版で再現し、HEAD の版で直っている。

### 11: mutant と回帰の確かめ

#### AC1(F9 の subtest)

| ID | 変えたところ | 結果 |
| --- | --- | --- |
| N2 | bound を 2 s に、期待の文を「ran past 2s」に(新しい stub のまま) | 緑(2.72 秒)。marker はなく、「ran past 2s and was stopped」 |
| O2 | N2 に加えて stub を base(293dff17)のもの(`exec >/dev/null 2>&1 </dev/null`、`sleep 3`)に | 赤(1.01 秒)。「failed (exit 128) … fatal: Could not read from remote repository.」で、期待の 2 つの文がない |
| X | `regateBoundedGit` の `exec.CommandContext(gitCtx, …)` を `exec.Command(…)` に(取り消しなし) | 赤(20.99 秒)。失敗は marker の検査(`regate_test.go:769` の「the slow upload-pack ran to its end」)だけで、「ran past 300ms」の検査は通る |
| X2(追加) | runner の `case gitCtx.Err() != nil:` が `timedOut` を false で返す(期限を報告しない) | 赤。regate が断らずに `gated` まで進み、「want a refusal」で落ちる |

O2 と N2 で、2 s の bound では base の stub は git の exit 128 になり、新しい stub は bound で止まることを、負荷に頼らずに確かめた(計画の AC1)。

#### AC2(`tests/test-golang-pack-test-timeout.sh` を `GOLANG_VERIFY` で写しに向ける)

| ID | 変えたところ | 結果 |
| --- | --- | --- |
| 対照 | HEAD の `verify.sh` | 34/34 |
| m1 | 既定を `GOFLAGS` の後ろに(`"${goflags:+$goflags }-timeout=30m"`) | 赤 11 件: fake の 4 事例 × 2 root(`-count=1`、引用、`go env -w`、`go env` の失敗)と、本物の go の 45 分の 3 事例が 30 分になる |
| m2 | `go env` を使わない(`goflags="${GOFLAGS:-}"`) | 赤 3 件: fake の `go env -w` の 2 root と、本物の go の `go env -w` の事例(30 分になる) |
| m3 | base の `run_tests`(293dff17 の `verify.sh`) | 赤 11 件: fake の 5 事例 × 2 root(`GOFLAGS` がつかない)と、本物の go の既定の事例(10 分になる) |
| m4 | `-timeout` を命令行に(`go test -timeout=30m ./...`、`GOFLAGS` はそのまま) | 赤 13 件: fake の 5 事例 × 2 root と、本物の go の 45 分の 3 事例(命令行が勝って 30 分になる) |
| m5(追加) | `go env` が失敗したら空にする(`$GOFLAGS` を使わない) | 赤 2 件: 「go env fails」の 2 root |
| m6(追加) | 前置の代入をやめて `GOFLAGS` を export する | 生き残った。等価な mutant である。root ごとに `( cd … && verify_root … )` の subshell で走り、`run_tests` はその最後の処理なので、export はどこにも漏れない |

m1〜m4 の赤の数(11・3・11・13)は計画の記録と同じである。

#### AC3(`tests/test-check-sync.sh` を `CHECK_SYNC` で写しに向ける)

| ID | 変えたところ | 結果 |
| --- | --- | --- |
| 対照 | HEAD の `check-sync.sh` | 16/16 |
| m0 | F8 より前の script(9349914b^) | 赤 5 件: 2 つの report の事例の rc と、`review-x.md`・`org-manifest-x.md` に ROOT_ONLY の行が出る 4 件 |
| m1 | `"docs/reports/review-"` を消す | 赤 3 件 |
| m2 | `"docs/reports/org-manifest-"` を消す | 赤 3 件 |
| m3 | `"docs/reports/"` を足す | 赤 6 件: `other-x.md`、`reviewer.md`、`x-review-y.md` の rc と ROOT_ONLY の行 |
| m4 | 部分一致(`*"${pattern}"*`) | 赤 2 件: `docs/old/docs/reports/review-x.md` の事例だけ |
| m5(追加) | directory のあとの file 名で一致(`"${pattern%/*}"/*"${pattern##*/}"*`) | 赤 2 件: `x-review-y.md` の事例だけ |
| m6(追加) | 前方一致の末尾の `-` を外す(`"docs/reports/review"`) | 赤 2 件: `reviewer.md` の事例だけ |

m0〜m4 の赤の数(5・3・3・6・2)は計画の記録と同じである。計画の Edge cases の「file 名の一部だけに当たる前方一致」は m6 が `reviewer.md` の事例で、「名前の途中に当たる一致」は m5 が `x-review-y.md` の事例で落ちる。

#### AC4(doctor の probe)

| ID | 変えたところ | 結果 |
| --- | --- | --- |
| D1 | probe が `doctorProbeTimeout` を読まない(`5*time.Second`) | 赤: `TestProbeBinary_TimeoutNamesTheBound` だけ。「claude --version produced no output」(2 s の sleep が 5 s の bound の中で終わる) |
| D2 | timeout の文を「timed out after 5s」に固定 | 赤: `TestProbeBinary_TimeoutNamesTheBound` だけ。「timed out after 5s」で 100ms がない |
| D3 | `TestMain` が bound を上げない | 生き残った(下の G1)。doctor の 4 試験(load 22)でも、`internal/cli` の全体の 813 本(342 秒、load は 31 から 166 まで上がった)でも緑。D4 で落ちる 15 本は 1 本 0.4〜1.2 秒で通り、stub の `--version` は 5 s に届かない |
| D4(感度の確かめ) | `TestMain` が bound を 1 ns にする(`internal/cli` の全体) | 赤 15 本: stub を PATH に置く doctor の試験がすべて「timed out after 1ns」で落ちた(813 本のうち。`TestProbeBinary_TimeoutNamesTheBound` は自分の bound で通る)。`TestCheckScaffoldIntegrity_CorruptManifest_StrictFails` の失敗の文には doctor の出力が入り、落ちた check(「Claude Code CLI: fail — claude unusable: claude --version timed out after 1ns」)が読める(AC4 の 4 つ目) |

#### AC5(dry-run の記録)

| ID | 変えたところ | 結果 |
| --- | --- | --- |
| M1 | dry-run の枝で本文を切らない(`rawPrefix + p.Text`、`internal/org` の全体) | 赤: 新しい試験だけ(`verbs_test.go:1421` の `...(truncated)`)。ほかの 1116 本は通り、1 本は darwin で飛ばす前からの試験(`TestEscalate_NoDesktopNotificationOffDarwinIsSkipped`)。#246 の V10 は生き残っていたので、穴が埋まった |
| M2(追加) | dry-run の枝で 200 byte の位置で切る | 赤: `TestOrgSend_DryRun_JapaneseBodyOver200Bytes_DetailsStayValidUTF8` だけ(`-run 'TestOrgSend\|DryRun'` の 60 本のうち)。`verbs_test.go:1417` の U+FFFD と、`:1424` の 202 byte |

### 12: `GOFLAGS` の追加の形(本物の go、HEAD の pack、scratch の `GOENV`)

scratch の module の試験が `t.Deadline()` までの分と `test.count` を書く。go1.26.0 と go1.25.8 で同じ結果になった。

| 呼び出し側の設定 | go test が見た deadline | rc |
| --- | --- | --- |
| `GOFLAGS=-timeout=10m`(README が書く短い bound) | 10 分 | 0 |
| `GOFLAGS=--timeout=45m` | 45 分 | 0 |
| `GOFLAGS=-test.timeout=45m` | 45 分 | 0 |
| `GOFLAGS='-timeout=45m'`(単引用の項目) | 45 分 | 0 |
| `GOFLAGS=-timeout=45m -timeout=10m`(呼び出し側の最後が勝つ) | 10 分 | 0 |
| `GOFLAGS=-count=2 -timeout=45m`(ほかの flag が先) | 45 分、count=2 | 0 |
| `go env -w` の 45m と、環境の `GOFLAGS=-count=2` | 30 分、count=2。環境の変数が file の値を丸ごと置き換える go の決まりのとおり | 0 |
| `GOFLAGS=-timeout=0` | deadline なし | 0 |
| `GOFLAGS=notaflag` | 走らない。`go: parsing $GOFLAGS: non-flag "notaflag"` | 1 |

`GOFLAGS=notaflag` のとき、`go env GOFLAGS` は `notaflag` を出して rc 0 になる。pack は `GOFLAGS=-timeout=30m notaflag` で go test を呼び、go test が `GOFLAGS` の解析で落ちる。base の pack も `GOFLAGS` をそのまま go test に渡すので、同じ所で落ちるはずである(base では打っていない)。計画の Progress の S2 の注と同じである。

### 13: F9 の subtest を base と HEAD で繰り返す

| 版 | 回し方 | load average(開始 → 終了、1 分) | 結果 |
| --- | --- | --- | --- |
| HEAD の写しの stub だけ base に戻す(bound 300 ms) | `-count=20` | 7.94 → 8.05 | 4 PASS、16 FAIL |
| base(293dff17 の写し) | `-count=20` | 16.48 → 15.20 | 5 PASS、15 FAIL |
| base | `go test -count=1` を 10 回 | 15.20 → 26.20 | 9 PASS、1 FAIL |
| base | `-count=10` | 122.82 → 92.89 | 6 PASS、4 FAIL |
| HEAD の写し | `go test -count=1` を 10 回 | 26.20 → 23.02 | 10 PASS |
| HEAD(worktree) | `-count=10` | 153.47 → 122.82 | 10 PASS |

base の FAIL はどれも「failed (exit 128)」である。1 つの process で 20 回繰り返すと 15〜16 回落ち、別の process で 1 回ずつ回すと 10 回のうち 9 回通る。tech-debt の F9 の行が書く 2026-10-18 の記録(単独で 10/10)とは数が違う。おそらく、同じ process での 2 回目以降は git の起動が速くなり、300 ms の bound より先に EOF で終わる。未確認です。どちらにしても、base の失敗は重い負荷がなくても起き、HEAD は負荷 8 でも 150 でも落ちなかった。

### 14: doctor の試験を負荷の下で

| 回し方 | load average(開始 → 終了、1 分) | 結果 |
| --- | --- | --- |
| 5 試験 `-count=10`、go1.26.0(batch2 の全体の mutant が 2 つ走る横で) | 23.71 → 24.44 | 50/50、45 秒 |
| 同じ、go1.25.8 | 24.44 → 20.25 | 50/50、32 秒 |
| 5 試験 `-count=3`、go1.26.0(ほかの session の iOS simulator で load が 150 を超えたとき) | 160.26 → 149.19 | 15/15、14 秒。1 試験 0.7〜1.4 秒 |

load 150 でも stub の `--version` は 5 s に届かず、2026-10-19 と 10-21 の失敗の条件はこの負荷だけでは再現しなかった。上げた bound が doctor の試験に効いていることは D4 で確かめた。

## load

| 実行 | 開始時の load average(1・5・15 分) | 終了時 |
| --- | --- | --- |
| 1 | 5.24 / 5.24 / 5.94 | 4.95 / 7.14 / 7.06 |
| 2 | 4.89 / 6.98 / 7.01 | 19.15 / 11.41 / 8.78 |
| 3 | 8.30 / 7.65 / 6.94 | 8.08 / 7.64 / 6.96 |
| 4 | 8.08 / 7.64 / 6.96 | 7.01 / 7.44 / 6.91 |
| 5 | 6.25 / 7.25 / 6.85 | 6.01 / 7.13 / 6.81 |
| 6 | 6.01 / 7.13 / 6.81 | 5.19 / 6.83 / 6.71 |
| batch2(D4 と M1、357 秒と 428 秒) | 22.90 / 16.97 / 11.63 | 16.59 / 44.04 / 30.15(途中で 1 分平均が 160 まで上がった) |

1 の間(20:03〜20:13Z)に、3〜10、12 と 11 の AC1〜AC3 の分を順に走らせた(写しの subtest、2 つの suite、小さな module の build)。その間の 1 分平均の load は 6〜10 だった。1 の Go の段(cli と org、20:08 ごろから 20:14)には 5・6、8〜10、12 が重なった。2 は単独で走らせた。2 の終わりの 19 と、その後の 150 前後は、ほかの session の swift の build と iOS simulator による(20:20 と 20:26 の `ps` で CPU の上位にあった)。

## Coverage

- Statement: `go test -coverprofile` を新しい試験と F9 の試験に絞って取った。`probeBinary` は `TestProbeBinary_*` の 3 本で 69.2% で、timeout の枝(`doctor.go:255-257`)を通る。`regateBoundedGit` は `TestRegate_FetchFailures` の 4 事例で 72.0% で、期限の枝(`regate_git.go:199-200`)を通る。`Send` の dry-run の枝(`verbs.go:345-352`)は、manifest への追記が失敗したときの return(349-351 行)を除いて、新しい試験が通る
- Branch: 測っていない。分岐は mutant で確かめた。`probeBinary` の timeout の枝は D1・D2、dry-run の枝の切り方は M1・M2、runner の期限の枝は X・X2 が落ちる
- Function: 同上
- Notes: shell の suite は計測の道具がないので、事例の範囲と mutant で数えた(AC2 の 6 事例と本物の go の 4 事例、AC3 の 7 回の実行)

## Failure analysis

| Test | Error | Root cause | Proposed fix |
| --- | --- | --- | --- |
| なし | - | 1〜10、12、14 で落ちたテストはない。11 と 13 の FAIL は、確かめのためにわざと起こした失敗と base の振る舞いである | - |

## Regression checks

| Previously broken behavior | Status | Evidence |
| --- | --- | --- |
| F9: 遅い upload-pack の subtest が、bound より先に git の exit 128 で終わる | 直っている。base は 1 つの process で 20 回のうち 15 回、別の process で 10 回のうち 1 回、load 90〜150 で 10 回のうち 4 回落ち、HEAD は 3・4・13 の 60 回すべて通る | 3、4、11 の N2・O2・X・X2、13 |
| F10: Go の pack の `go test ./...` が go test の 10 分の既定で止まる | 直っている。`GOFLAGS` なしの run-test.sh が rc 0 で、本物の go の deadline は 30 分。呼び出し側の timeout は、どの形でも既定のあとに来て勝つ | 1、8、11 の m1〜m4、12 |
| G1(根拠の表): F8 の 2 つの除外を check-sync で通す suite がない | suite ができた。除外を 1 つ消しても、F8 より前の script でも落ちる | 7、11 の m0〜m6 |
| doctor の試験が負荷の下で stub の `--version` の 5 s を越えて落ちる | `TestMain` が bound を 1 分に上げ、stub を使う doctor の 15 試験がその値で走る(D4)。負荷 20〜160 で 115 回通った | 5、6、11 の D1〜D4、14 |
| #246 の G1: dry-run の `sent` の本文の切り方を確かめる試験がない | 試験ができた。切らない mutant と byte で切る mutant が、どちらもこの試験だけで落ちる | 11 の M1・M2 |
| self-review の L1: 新しい suite が信号のあとで再開する | 直っている | 10 |
| 全体(shell 43 本、Go 8 パッケージを 2 つの toolchain で、AC7 のテスト部分) | pass | 1、2 |

計画の Edge cases:

- `GOFLAGS` にほかの flag: suite の `-count=1`(fake)、12 の `-count=2 -timeout=45m`
- 引用の項目: suite の `"-timeout=45m"`(fake と本物)、12 の単引用
- `go env -w` の file にだけ timeout: suite の scratch `GOENV`(fake と本物)、12 の file と環境の両方
- `go env` の失敗: suite の fake(`FAKE_GO_ENV_FAIL=1`)、m5 がその事例で落ちる。本物の go の `go env` は不正な `GOFLAGS` でも失敗しない(12)
- static mode で `go test` を呼ばない: suite の fake の 3 件(`go vet` が走り、`go test` と `go env` が 0 行)
- file 名の一部だけに当たる前方一致: suite の `reviewer.md` と `x-review-y.md`、m5・m6

## Test gaps

- G1(LOW): `TestMain` が `doctorProbeTimeout` を上げることを確かめる試験がない。D3(上げない)は doctor の 4 試験でも `internal/cli` の全体でも生き残った。stub が 5 s を越えたときにだけ差が出るので、ふつうの実行では見えない。D4 で、`TestMain` の値が stub を使う doctor の 15 試験の bound になることは確かめた。埋めるなら、`internal/cli` に「`doctorProbeTimeout` が 5 s より大きい」ことを見る 1 本を置けば足りる。この PR を止めるものではない
- G2(INFO): doctor の試験の元の失敗(stub の `--version` が 5 s を越える)は、load 150 でも再現しなかった(1 試験 1 秒前後)。2026-10-19 と 10-21 の失敗の条件は負荷のほかにもあるとみられる。未確認です
- G3(INFO): base の F9 の subtest は、2026-10-18 の記録(単独で 10/10)より今のほうがよく落ちる(13)。理由は確かめていない。tech-debt の F9 の行は原因を stub の stdout と書いており、13 の結果はそれと合う。記録に足すかは `/sync-docs` に任せる
- G4(INFO): `TestCheckScaffoldIntegrity_CorruptManifest_StrictFails` の doctor は、利用者の `~/.codex/models_cache.json` を読む(D4 の出力の「Org codex model slugs: pass — … ~/.codex/models_cache.json」)。base からある振る舞いで、この PR の差分ではない。cache の中身で結果が変わる試験ではない
- G5(INFO): 負荷の下での live の regate と試行は打ち直していない(計画の Non-goals)。Linux では走らせていない。CI の verify(`ubuntu-latest`)が run-verify.sh を走らせる
- G6(INFO): `ralph upgrade` で pack の `verify.sh` が下流に届くことは確かめていない(verify の報告と同じ)

## Verdict

- Verdict: pass
- Pass: 1(`GOFLAGS` なしの run-test.sh。shell 43 本と Go 8 パッケージ、773 秒、rc 0、cli 251.7 秒、org 314.2 秒)、2(go1.25.8 の `./...`、332 秒、rc 0)、3 と 4(F9 の subtest 20/20 を両方の toolchain で)、5 と 6(doctor の 4 試験 40/40 を両方の toolchain で)、7〜9(2 つの suite を両方の toolchain で、go なしで SKIP)、10(TERM で rc 1、TMPDIR は空)、11(mutant 22 本のうち 20 本が狙いどおり赤、生き残りは等価の m6 と G1 の D3)、12(`GOFLAGS` の 9 つの形が両方の toolchain で設計どおり)、13 と 14(HEAD は負荷 23〜153 で F9 20/20、負荷 20〜160 で doctor 115/115)
- Fail: なし
- Blocked: なし
- 残る穴: G1(LOW)。`TestMain` の bound の引き上げを確かめる試験がない。ふつうの実行では差が出ないので、マージを止めるものではない
