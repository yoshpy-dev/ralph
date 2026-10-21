# Self-review report: test-stability

- Date: 2026-10-22
- Plan: `docs/plans/active/2026-10-21-test-stability.md`(パイプライン cycle 1 / 2)
- Reviewer: reviewer サブエージェント(`/self-review`)
- Scope: `git diff 293dff17...HEAD`(HEAD 818bd4b1、16 ファイル、+1017/-35)。S1(`internal/org/regate_test.go`)、S2(Go pack の `verify.sh` 2 か所、pack の README 2 か所、`tests/test-golang-pack-test-timeout.sh`)、S3(`tests/test-check-sync.sh`)、S4(`internal/cli/doctor.go`、`main_test.go`、`cli_test.go`、`doctor_scaffold_test.go`)、S5(`internal/org/verbs_test.go`)、S6(tech-debt、根拠の文書、spec)、計画の進捗メモ

## Evidence reviewed

- 16 ファイルの diff を全部読んだ。コードとテストは `git diff 293dff17...HEAD -- internal/ packs/ templates/`、2 つの新しい shell suite は全文、記録は `docs/tech-debt/README.md` の変更 5 行(コメント行と表の行の組)、根拠の文書、spec の diff を読んだ。テスト、gofmt・vet、doc drift の掃引は範囲外で、実行していない。背後で `./scripts/run-verify.sh` が動いていたので、ファイルは直していない
- S1 の stub と marker。`regateBoundedGit`(`internal/org/regate_git.go:173-210`)と `regateRemoteGit`(145-147、`ownGroup=false`)を読んだ。fetch の git は ralph の process group にいて、bound が殺すのは git だけである。コメントの「git only, in ralph's group」は実装どおり。stub の stdout は git が読む pipe、stderr と stdin は `/dev/null` なので、git が死んだあとに ralph の capture pipe を持つものはない。runner が期限切れを exit code より先に読む(`regate_git.go:197-200`)ので、marker がなければ「ran past」だけでは、bound で止めた場合と stub の終わりまで待った場合を見分けられない。コメントの理由はこの実装と合う
- S1 の marker の検査の順序。`newRegateFixture` が `t.TempDir()` を先に登録する(`regate_test.go:55`)。marker の `t.Cleanup` は `setup` で後から登録するので、LIFO で temp dir の削除より前に走る。`fileOrAbsent` と `"<absent>"` の比較は `escalate_test.go`、`deadline_test.go` と同じ書き方で、`shellQuote` は `internal/org/feature.go:408` にある。marker は stub の開始から 20 s 後に書かれる。fetch のあとに subtest が走らせるのは snapshot の git 3 回と stat 数回なので、20 s は通常の負荷の 2 桁上にあたる。20 s を超える停止が起きると、orphan の stub が marker を書き、「ran to its end」で誤って落ちるが、所見にはしない
- S2 の `run_tests`。POSIX の構文だけ(`${goflags:+ $goflags}`、代入の前置)で、前置の代入は単語分割されない。`GOFLAGS` の値の引用符はそのまま Go 自身の分割に渡り、引用つきの項目は両方の suite(fake の go と本物の go)で確かめている。`goflags=$(go env GOFLAGS 2>/dev/null) || goflags="${GOFLAGS:-}"` は `set -e` の下で `||` に守られる。go がない場合は pack が前段(`verify.sh:62`)の `command -v go` で exit 1 するので、この行には届かない。`go env` が失敗する場合は「go env fails」の事例が覆う
- S2 の「Go が受ける形ならどれでも」。cmd/go のソースを読んだ。`SetFromGOFLAGS`(`base/goflags.go`)は `--` を `-` に直し、項目を順に `flags.Set` するので、あとのものが勝つ。`test.timeout` は `cf.Var(cf.Lookup(name).Value, "test."+name, "")`(`test/testflag.go:78`)で同じ値の別名なので、`-test.timeout=45m` も `--timeout=45m` もあとに来れば勝つ。コメントと README の文は実装と合う
- S2 の pack の 2 つのコピー。`packs/languages/golang/verify.sh` と `templates/packs/golang/verify.sh`、2 つの README は `cmp` で同一
- 2 つの shell suite の移植性。どちらも `#!/usr/bin/env sh`、配列・`[[`・`local` なし、`tr -c`、`grep -Fqx --`、`mktemp -d テンプレート`、`pwd -P` だけで、BSD と GNU の違いに触れる箇所はない。macOS の `TMPDIR` の symlink は `pwd -P` で吸収している(fake の go が出す `pwd` と比べるため)。両方 755、`scripts/verify.local.sh:286-292` の `tests/test-*.sh` の glob に拾われるので、登録の漏れはない。関数 `run_real` と `expect_deadline` を `else` の中で定義しているが、`sh` では有効
- S2 の suite の数字。`run_fake` が 6 回(assert 6)、`expect_go_test` が 5 回(3 × 5 = 15)、static の 3 つで、fake の go は 24。本物の go は、scratch の GOENV の 2 つと `run_real` の 4 回、`expect_deadline` の 4 回で 10。合計 34 で、tech-debt の行と進捗メモの「34 checks、24 と 10」と一致する。`case_file` の名前に衝突はない(fake は `fake ` を前置、本物は `real go, ` を前置)
- S3 の suite。`check-sync.sh` の `ROOT_ONLY_EXCLUSIONS`(`review-`、`org-manifest-`、`self-review-`)と `is_excluded` の前方一致(`"${pattern}"*`)を読み、事例の期待と合わせた。`reviewer.md`(`review-` に合わない)、`x-review-y.md`(先頭でない)、`docs/old/docs/reports/review-x.md`(先頭でない)はどれも ROOT_ONLY になる。check の数は 1 + 3 + 4 + 2 + 2 + 2 + 2 = 16 で、行の「16 checks」と一致する。`check-sync.sh` は `$0` から repo root を決めるので、scratch の tree が root になる
- S4。`probeBinary` の呼び出しは `doctor.go:271`、`289` と `cli_test.go` の 3 か所だけ。`internal/cli` に `t.Parallel` はなく、`doctorProbeTimeout` を書くのは `TestMain` と新しい 1 つの試験(`t.Cleanup` で戻す)だけなので、競合はない。エラー文は `%s` に `time.Duration` を渡すので、既定の 5 s は従来の「5s」と同じ字になる。名前は `doctorShellAliasEnv` など隣の `doctor*` の seam と合う。`captureStdoutText` は同じファイルにある。`/bin/sleep` の絶対パスは、PATH が shim だけのためとコメントに書いてあり、release の対象(darwin と linux)にある
- S5。`truncateForDetails` が dry-run の枝(`verbs.go:345-348`)と実際の送信(469)の両方にあることを確かめた。新しい試験は、実際の送信の試験(1350 行)の本文と 4 つの検査を写し、spawn を省き(dry-run の枝は `findSeat` の前で返る、と計画にある)、driver の呼び出しがないことを足している
- S6 の記録。commit の SHA(S1 3f8ccd9b、S2 db1e5ac7、S3 dc58715a、S4 439ae909、S5 43df3185)は `git log` と一致する。tech-debt の新しい 5 行は、`(?<!\\)\|` で数えた区切りが 6 個(5 欄)で、base の 254 行と同じ。HTML コメントは 5 行とも `-->` で閉じる。`Row preserved for traceability` を使うコメントは base に 52 行あり、形は同じ。根拠の表の判定の数は、`[ ]` を FR-2・FR-3・FR-4・FR-5・FR-6・FR-9・FR-10・安全の不変条件・ツール非依存・受け入れ条件 2・5 と数えて 11 で、spec の `[ ]` も 11。PR の表は 29 本で、spec の Status と「NFR 入れ方」の 26(= 29 − 3)と合う。`ralph org gate` と `regate` が `scripts/run-test.sh`(`internal/org/gate.go:36`)から `run-verify.sh` を通って pack の `verify.sh` に行く道は、`run-verify.sh:172` で確かめた。pack 以外に `go test` を打つ runner はない(`scripts/` を grep した)
- TestMain のコメントの数「4 つの試験、3 つが `claude --version timed out after 5s` を出した」。tech-debt の行の 2 つ(merge guarantee と codex sandbox)と、`docs/reports/test-2026-10-13-org-plan-approval.md:265` の `HerdrAgmsgAbsent` で、3 つ。`CorruptManifest` は出さなかった(行に「未確認」とある)ので数は合う
- 差分に、デバッグ出力、TODO、秘密情報、ホームの絶対パス、U+FFFD はない(追加行を grep した)
- 確かめていない: 2 つの suite と Go のテストの実行結果、mutant の失敗数(計画と tech-debt の行が書く数)、`gh pr view` で 29 本が `MERGED` であること(`#246` は `main` の merge commit で見えるが、ほかの 28 本は 2026-10-21 の確認に頼る)、spec の `[x]` が 21 で根拠の表の 20 と 1 つ違うこと(base でも 20 と 19 で、この PR が作った差ではない。`/verify` の範囲)

## Findings

<!-- Area recommended values: naming, readability, unnecessary-change, typo,
     null-safety, debug-code, secrets, exception-handling, security, maintainability -->

| Severity | Area | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- |
| LOW (L1) | exception-handling | 2 つの新しい suite は `trap cleanup EXIT HUP INT TERM` で、`cleanup` は `rm -rf "$workdir"` だけをして exit しない。POSIX sh は信号の trap のあとで script を再開するので、Ctrl-C や TERM のあと、workdir がない状態で残りの事例が走り、`go test`(part B)も走る | `tests/test-check-sync.sh:76`、`tests/test-golang-pack-test-timeout.sh:92`。同じ形で probe した(`trap cleanup EXIT HUP INT TERM`、`kill -TERM $$`):`cleanup ran`、`RESUMED; dir exists: no`、削除済みの dir への書き込みが失敗、`rc=0`。この形は既存の suite にもある(`tests/test-branch-name.sh:16`、`test-gc-artifacts.sh:16`、`test-run-verify-scope.sh:52` など約 10 本)。一方、`tests/test-agent-models.sh:62-63`、`test-archive-plan.sh:39-40`、`test-plan-visual.sh:25-26` は `trap 'exit 1' HUP INT TERM` を別に置く | `trap cleanup EXIT` と `trap 'exit 1' HUP INT TERM` に分ける。既存の約 10 本と同じ形なので、この PR で直さず、`tests/` をまとめて直すときでもよい |
| LOW (L2) | maintainability | S5 の新しい試験は、実際の送信の試験の検査の塊(UTF-8、U+FFFD、`...(truncated)` の suffix、prefix と 200 byte の上限)と、本文の文字列 `strings.Repeat("日本語の本文を記録する。", 10)` を 2 つ目として写している。上限や marker の文言を変えるとき、直す場所が 2 つになる | `internal/org/verbs_test.go:1350-1384` と `1392-1426`。検査の塊(`details := last.Details` から prefix の検査まで)は 1370-1383 と 1412-1425 の 14 行ずつで、違うのは spawn の有無、`DryRun`、driver 呼び出しの検査、`last.DryRun` の 4 点だけ | `assertDetailsCutAtACharacter(t, msg, details)` のような helper に塊を出し、本文も 1 つの変数にする。2 つの試験の意図(実際の送信と dry-run)は変わらない。任意 |
| LOW (L3) | naming | 2 つの suite で、同じ値を指す名前が 2 つずつある。`run_fake` は大域の `FAKE_LOG` に path を置き、stub に渡す環境変数 `FAKE_GO_LOG` に写す。`run_real` は `DEADLINE_FILE` と `DEADLINE_OUT` で同じことをする。大域の側は `expect_go_test` と `expect_deadline` が暗黙に読む。名前が似ていて、片方だけを直すと stub が別の file に書く | `tests/test-golang-pack-test-timeout.sh:149,158`(`FAKE_LOG` と `FAKE_GO_LOG`)、`:253,269`(`DEADLINE_FILE` と `DEADLINE_OUT`)、`:174-177`、`:281` | 大域を `fake_log`、`deadline_file` のように小文字にして env の名前と分けるか、`expect_*` に path を引数で渡す。任意 |

重大度の数は CRITICAL 0、HIGH 0、MEDIUM 0、LOW 3。

## Positive notes

- S1 は、原因(stub が git の読む pipe を閉じるので、git が 0.2 s ほどで exit 128 になり、300 ms の bound と競る)をコメントに書き、直し(stdout を開けたまま眠る)と、直しだけでは見えない欠陥(bound が git を止めない場合も「ran past」と読まれる)を marker で見分ける。marker の検査は、setup の中の `t.Cleanup` が LIFO で temp dir の削除より前に走る順序に乗っていて、コメントもそう書く
- S1 の stub は git が死んだあとも最大 20 s 眠って残る(前の stub も 3 s 残った)。git の stdout を持つので ralph の pipe は握らず、`go test` の出力を止めない。`exec sleep` にはできない(marker を書く行があるため)ので、これは受け入れてよい
- S2 は、fix を `GOFLAGS` の先頭に置いて呼び出し側の timeout が勝つ形にし、コマンドラインに置く案(呼び出し側を上書きする)を退けた理由をコメントに書いた。fake の go で渡された値を、本物の go で `t.Deadline()` を、両方で確かめる。scratch の `GOENV` を使い、利用者の `go.env` を読まず書かない。`GOLANG_VERIFY` と `CHECK_SYNC` で対象の script を差し替えられるので、mutant を同じ事例で流せる
- S4 は、5 s を利用者向けの値のまま残し、test だけが `TestMain` で上げる。timeout の経路は、100 ms と 2 s の shim(`exec /bin/sleep 2`)で別に守り、変数を無視する実装を落とす。shim が exec する理由(kill が sleep に届く)をコメントが言う。エラー文は束縛した値を名指す
- S6 の記録は、5 行とも打ち消しと RESOLVED のコメントの形が先例(base の 52 行)と同じで、数(16、34 = 24 + 10、11、29、26)を再計算して合う。F12・F14 の「直す PR」の欄を `#246` に直したのは、根拠の文書の 16 行目の新しい規則(merge した PR は番号で書く)に従う
- 追加行にデバッグ出力、TODO、ホームの絶対パス、秘密情報、U+FFFD はない。コミットの型(`test:`、`fix:`、`docs:`)も内容と合う。pack の変更だけが `fix:` で、release のノートに残る

## Tech debt identified

| Debt item | Impact | Why deferred | Trigger to pay down | Related plan/report |
| --- | --- | --- | --- | --- |
| なし | - | L1・L2・L3 は任意の整理で、先送りの作業や既知の近道ではない | - | - |

_(行を足していないので、`docs/tech-debt/` への追記はない。)_

## Recommendation

- Merge: yes。CRITICAL、HIGH、MEDIUM の所見はない。LOW の 3 件(L1〜L3)は任意で、直さなくても `/verify` に進める
- Follow-ups: L1 は既存の約 10 本の suite と同じ形なので、`tests/` の trap を揃えるときにまとめて直す。L2・L3 を直す場合も、テストの意図は変わらない。どれを直しても `/self-review` の cycle 2 を要するため、cycle の上限(2)を使い切るなら、直さずに `/verify` へ進める
- 範囲外で `/verify` に渡すこと: spec の `[x]` が 21 で、根拠の表の 20 と 1 つ違う(base でも 20 と 19)。進捗メモは「spec 21 and 11」と書いており、この PR が作った差ではない

## LOW の直し(報告のあと)

- L1 と L3 は 6d990dfc で直した。2 つの新しい suite は `trap cleanup EXIT` と `trap 'exit 1' HUP INT TERM` に分けた(`tests/test-agent-models.sh:62-63` と同じ形)。`tests/test-golang-pack-test-timeout.sh` は、stub と go test が環境から読む `FAKE_GO_LOG` と `DEADLINE_OUT` の名前だけを残し、`FAKE_LOG` と `DEADLINE_FILE` をなくした。既存の約 10 本の suite の trap はこの PR では直さない
- 確かめたこと(implementer の報告): `sh -n` と `bash -n` が両方とも通り、`tests/test-check-sync.sh` は 16 件すべて PASS、`tests/test-golang-pack-test-timeout.sh` は go1.26.0 と go1.25.8 の両方で 34 件すべて PASS。専用の `TMPDIR` で起動して途中で TERM を送ると、2 本とも rc 1 で Summary の前に止まり、`TMPDIR` に何も残らなかった
- L2 は直さない。既存の試験を 1 byte も変えないことを優先した。同じ形の試験が 3 本目になったときに 3 本まとめて helper に出す
- 範囲外の項の数の差(spec の 21 と根拠の表の 20)は FR-12 の分である。根拠の表は FR-12 を入れない(表の前文)ので、spec の 32 項目に対して表は 31 項目になる
- この直しは、続く `/verify`、`/test`、`/cross-review` の差分に入る
