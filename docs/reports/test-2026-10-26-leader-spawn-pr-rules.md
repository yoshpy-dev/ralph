# Test report: leader-spawn-pr-rules

- Date: 2026-10-26(JST 15:35〜16:20)
- Plan: `docs/plans/active/2026-10-26-leader-spawn-pr-rules.md`(AC1b、AC4、AC6、Test plan)
- Tester: tester サブエージェント(Claude Opus 5.5)。パイプラインの 1 回目(cycle 1 / 2)
- Scope: HEAD 2947c524(base 619cf5f8)。差分のコードは `featureLeaderTask` の 1 行(`internal/org/feature.go:412`)と leader の雛形 `internal/org/prompts/leader.md`、テストは `internal/org/feature_test.go` と `internal/org/prompts_test.go`。behavioral test だけを実行した(静的解析は /verify の担当)。環境は macOS(Darwin 25.5.0、arm64)、go1.26.0 と go1.25.8(`GOTOOLCHAIN=go1.25.8`。`GOPROXY=off go version` が go1.25.8 を返したので、toolchain は手元にあり取得していない)
- Evidence: `docs/evidence/test-2026-10-26-leader-spawn-pr-rules.log`(run-test.sh の出力)、同じ場所の `-go126.log`、`-go1258.log`、`-named-local.log`、`-named-go1.25.8.log`。run-test.sh 自身も `docs/evidence/verify-2026-10-26-063507.log` を書いた。`docs/evidence/*.log` は gitignore の対象なので、コミットしない。mutant と probe の写しと出力は scratchpad に置いた
- 追跡しているファイルは、この報告と insight event のほかは変えていない。mutant の前とあとで `git status --porcelain --ignored` は同じだった。実行のあと、`ralph-org`・`herdr`・`agmsg` の名前を持つプロセスは残っていない。利用者の `go env GOENV` のファイル(`~/Library/Application Support/go/env`)は実行のあとも存在しない。Go の出力に `go: downloading` の行はない

## Test execution

| # | Suite / Command | Tests | Passed | Failed | Skipped | rc と時間 |
| --- | --- | --- | --- | --- | --- | --- |
| 1 | `./scripts/run-test.sh`(scope は `changed (changed_languages)`、language pack は golang) | shell 44 本(`tests/test-*.sh` のすべて)と `go test ./...` | 44 本、Go はテストのある 8 パッケージ | 0 | shell 1 件(下) | rc 0、904 秒(15:35:06〜15:50:10)。`internal/cli` 323.314s、`internal/org` 403.448s、`internal/config` 0.488s。ほかの 5 パッケージは `(cached)` |
| 2 | `go test -count=1 -timeout 30m ./internal/org/... ./internal/cli/...`(go1.26.0) | 4 パッケージ | 4 | 0 | - | rc 0、919 秒(15:50:50〜16:06:09)。`internal/org` 917.785s、`internal/cli` 846.786s、`internal/org/driver` 1.465s、`internal/org/protocol` 1.031s |
| 3 | 2 と同じコマンドを `GOTOOLCHAIN=go1.25.8` で | 4 パッケージ | 4 | 0 | - | rc 0、448 秒(16:06:18〜16:13:46)。`internal/org` 442.618s、`internal/cli` 370.377s、`internal/org/driver` 1.349s、`internal/org/protocol` 1.509s |
| 4 | `go test -count=3 -v -run '^(TestFeatureLeaderTask\|TestFeatureLeaderTask_ScopeIsOneShellWord\|TestRenderRolePrompt_Leader_FeatureOrgProcedure)$' ./internal/org/`(go1.26.0) | 3 本 × 3 回 | 9 | 0 | 0 | rc 0、1 秒 |
| 5 | 4 と同じコマンドを `GOTOOLCHAIN=go1.25.8` で | 3 本 × 3 回 | 9 | 0 | 0 | rc 0、2 秒 |

- 1 の shell の 44 本は `tests/test-*.sh` の数(44)と同じで、`FAIL:` の行はどれも `FAIL: 0`。Skipped は前からある `tests/test-secret-scan-branch.sh` の 1 件(git 2.49.0 なので「2.41 より古い本物の git」の事例を飛ばし、stub の古い git がその分岐を見る)
- 1 の Go は `-count=1` がないので、5 パッケージ(`internal/insights`、`internal/org/driver`、`internal/org/protocol`、`internal/scaffold`、`internal/upgrade`)は cache の結果を使った。差分のある `internal/org`、それに依存する `internal/cli`、`internal/config` の 3 つは打ち直された。`internal/org` と `internal/cli` は 2 と 3 で `-count=1` を付けて打ち直している
- load average(1 分)は、1 の始まりが 11.14、終わりが 7.60。2 の途中の 16:00 ごろに 108.60 まで上がり、終わりは 9.54(5 分の平均は 38.62)。3 は始まり 9.54、終わり 5.63。108 のときの `ps` では、テストと別の Xcode の `swift-frontend`、iOS Simulator、WebKit のプロセスが上位にいた。2 の `internal/org` と `internal/cli` が 3 のおよそ 2 倍かかったのは、この負荷による
- この負荷のもとで、落ちたテストはない。slice 2 のときに落ちた `TestOrgRegate_InterruptUndoesTheMergeAndRecordsNothing/SIGTERM` も、1〜3 のどれでも落ちていない(下の「既知の不安定」)

## AC1b のテストの事例と mutant

### 事例の読み取り

`TestFeatureLeaderTask_ScopeIsOneShellWord`(`internal/org/feature_test.go:593`)は 7 つの予約を試し、各パスが `normalizeReservePath` をそのまま通ることを先に確かめてから、`- 予約したパス:` の行の `--scope` の引数を `/bin/sh -c 'set -- <引数>; printf ...'` に読ませ、引数が 1 つで元の値(`, ` でつないだもの)と同じかを見る。

| 依頼の場合 | 事例 | 判定 |
| --- | --- | --- |
| `$` | `app/routes/$slug.tsx`、7 行目の `app/$HOME/` | ある |
| バッククォート | ``a/`id`/``、7 行目の ``d"e`f`/`` | ある |
| 単一引用符 | `it's/x.md`、7 行目の `b'c/` | ある。`TestFeatureLeaderTask` も `it's/auth.md` の行を `'it'\''s/auth.md'` の字面で固定している |
| 二重引用符 | `say"hi"/`、7 行目の ``d"e`f`/`` | ある |
| 複数のパス | `docs/auth/, docs/x.md`、7 行目の 5 パス | ある。`TestFeatureLeaderTask` も 2 パスの行を字面で固定している |
| (依頼の外)バックスラッシュ、glob | 7 行目の `g\h`、`internal/*.go` | ある |

依頼の 5 つの場合はすべて事例がある。足りないと言えるのは次の 3 点で、どれも LOW とする。

1. `$(...)`、`~`、`!`、先頭か末尾の `'`(`'lead/`、`trail'`)は、`normalizeReservePath` が通すのに事例にない。throwaway の probe(下)で、どれも `/bin/sh`、`/bin/zsh`、`/bin/bash` で 1 つの元の値になった。`shellQuote` はすべての文字を単一引用符で包むので、`$` とバッククォートの事例が通る実装はこれらも通す。固定のために足す必要は低い。`{b,c}` はカンマを含むので `normalizeReservePath` が断る(probe で確かめた)
2. 追跡しているテストが読ませる shell は `/bin/sh` だけである。leader が spawn を打つ shell は driver と端末の設定によって zsh か bash でありうる。probe の 9 事例 × 3 shell の 27 通りはすべて一致した
3. `/bin/sh` が構文の誤りで終わると `t.Fatalf`(`feature_test.go:626`)でテストが止まり、後ろの事例は走らない。下の m1 では `it's/x.md` で止まり、7 行目は報告されなかった。落ちることに変わりはないので、テストの判定には影響しない

### mutant(`go test -overlay`、追跡しているファイルは変えていない)

`internal/org/feature.go` の写しを scratchpad に置き、`- 予約したパス:` の行だけを変えて差し替えた。対象は `-run '^(TestFeatureLeaderTask.*|TestRenderRolePrompt_Leader_FeatureOrgProcedure)$'`(go1.26.0)。

| mutant | 変更 | rc | 落ちたテスト | AC1b のテストで落ちた事例 |
| --- | --- | --- | --- | --- |
| m0(対照) | 変えない写し | 0 | なし | - |
| m1 | `shellQuote` を外す(`--scope %s`) | 1 | 3 本すべて | 2 パス(`$#` が 2)、`$slug`(展開されて `app/routes/.tsx`)、バッククォート(`id` が走り `$#` が 3)、`it's/x.md`(sh が exit 2、ここで止まる) |
| m2 | `"…"` で囲む(`--scope "%s"`) | 1 | 3 本すべて | `$slug`、バッククォート、`say"hi"/`(`sayhi/` になる)、7 行目(sh が exit 2) |
| m3 | 前の形の行(`"- 予約したパス: " + reserve`) | 1 | 3 本すべて | 1 行目で「行がない」 |
| m4 | 最初のパスだけを引用する | 1 | `TestFeatureLeaderTask`、AC1b のテスト | 2 パスと 7 行目 |
| m5 | `,` でつなぐ(空白なし) | 1 | `TestFeatureLeaderTask`、AC1b のテスト | 2 パスと 7 行目 |
| m6 | `'%s'` で囲み、中の `'` を `'\''` にしない | 1 | `TestFeatureLeaderTask`、AC1b のテスト | `it's/x.md`(sh が exit 2) |

依頼の 2 つの写し(`shellQuote` を外す m1、`"…"` で囲む m2)は、どちらも AC1b のテストを落とした。m1 と m2 の下では、``a/`id`/`` の事例が本物の `/bin/sh` で `id` を走らせた。害のないコマンドなので問題はないが、事例のコマンドは今後も害のないものにしておく。

### 雛形の文の mutant(AC4)

`internal/org/prompts/leader.md` の写しを `-overlay` で差し替えた(go:embed のファイルも差し替わる)。対象は `-run '^TestRenderRolePrompt_'`。

| mutant | 変更 | rc | 落ちた pin |
| --- | --- | --- | --- |
| pc(対照) | 変えない写し | 0 | なし |
| p0 | base(619cf5f8)の雛形 | 1 | 新しい pin 6 つ(ミッションの 2、手順 5 の 3 句、手順 3、手順 14)。既存の pin(`--id`、`/pr` skill そのものは実行しない、など)は落ちない |
| p1 | ミッションの 2 の `--scope` の文を消す | 1 | `prompts_test.go:494` |
| p2 | 手順 3 の flag の文を消す | 1 | `:577` の手順 3 の pin |
| p3 | 手順 5 の `--scope` の文を消す | 1 | `:577` の手順 5 の 3 句 |
| p4 | 手順 14 を前の文に戻す | 1 | `:577` の手順 14 の pin |
| p5 | 手順 5 の「書き換えずに付ける」を「引用符で囲んで付ける」にする | 1 | `:577` の手順 5 の 1 句 |

p0 は、既存の pin が残り、新しい pin が新しい文だけを見ていることも示す。

### probe(throwaway、追跡しない)

`internal/org/zz_probe_test.go`(存在しないパス)を `-overlay` で足し、9 事例の `--scope` の引数を `/bin/sh`、`/bin/zsh`(zsh 5.9)、`/bin/bash`(3.2.57)に読ませた。事例は追跡しているテストの 4 つに `a/$(id)/`、`'lead/` と `trail'`、`~/x` と `a/!b/` と `a/{b,c}/` を足したもの。go1.26.0 と go1.25.8 のどちらも rc 0、27 通りすべて一致した。

## Coverage

- Statement: `go test -run '^TestFeatureLeaderTask' -coverprofile` で、`featureLeaderTask`(`feature.go:401`)と `shellQuote`(`feature.go:448`)はどちらも 100.0%。パッケージ全体の coverage は今回計っていない(最後の記録は `internal/org` 95.0%、`docs/reports/test-2026-10-23-busy-leader.md`)
- Branch: 計っていない(Go の標準の道具にない)
- Function: 上の 2 関数のみ
- Notes: 雛形の文は coverage に出ない。上の p0〜p5 で、文を戻すと pin が落ちることを確かめた

## Failure analysis

| Test | Error | Root cause | Proposed fix |
| --- | --- | --- | --- |
| なし | - | - | - |

## Regression checks

| Previously broken behavior | Status | Evidence |
| --- | --- | --- |
| RC2: 雛形どおりの spawn が `--scope` なしで `autonomous` に断られた | 雛形とタスクの行は直っていて、テストが固定している。実機の spawn では確かめていない(計画の Non-goals、R3) | 4・5 と mutant の m1〜m6、p1〜p3、p5 |
| RC5: 機能の org の PR の題と本文が英語だった | 雛形の文は直っていて、テストが固定している。leader が実際に日本語で書くかは確かめていない(R3) | 4・5 と mutant の p4 |
| `- 台帳:` の行の `--state-dir` が 1 つの shell の語であること(`TestFeatureLeaderTask_StateDirIsOneShellWord`) | 変わらず通る | 1〜3 |

### 既知の不安定(`TestOrgRegate_InterruptUndoesTheMergeAndRecordsNothing/SIGTERM`)

- 今回は 1〜3 のどれでも落ちなかった。2 は 1 分の load average が 108 まで上がった中で通っている。落ちていないので、単独の打ち直しはしていない
- `docs/tech-debt/README.md` に、このテストの不安定の行はない。`grep -n "Interrupt"` が当たるのは 234 行と 235 行(org-gate の cycle 2〜4 の小さな所見で、`TestGate_InterruptAfterTheScriptsRecordsNothing` などを挙げる行)だけで、`InterruptUndoes` と `OrgRegate_Interrupt` は 0 件。`regate` を含み不安定や負荷を書く行は、別のテストの行(283 行の `TestStopDirectorProcess_AProcessThatIgnoresSIGTERMIsNotEndedInTime`、308 行の `TestRegate_FetchFailures` の RESOLVED)と、`go test` の時間切れの RESOLVED の行(314 行)で、このテストの行ではない。テスト名が出てくる文書は、この計画と `docs/reports/test-2026-10-16-org-regate.md`(AC6 の対応表)だけである
- tech-debt は書き換えていない。もう一度落ちたら、行を足すかを決める材料になるのは、slice 2 の 1 回(load average 約 12、`out/started did not appear within 10s`、`internal/cli/org_regate_test.go:528` の待ち)と今回の 0 回である

## Test gaps

1. 実機での確かめはない。leader が `--scope` を書き換えずに写すか、PR を `/pr` skill の決まりどおり日本語で書くかは、本物の herdr と claude が要る。FR-4 と受け入れ条件 5 は `[ ]` のままで、次の実行で確かめる(計画の R3)
2. AC1b のテストは `/bin/sh` だけで読み、`$(...)`、`~`、`!`、先頭・末尾の `'` の事例を持たない。probe では zsh と bash を含めてすべて一致したので、LOW とする
3. AC1b のテストは `/bin/sh` が構文の誤りで終わると `t.Fatalf` で止まり、後ろの事例を報告しない。判定には影響しない
4. パッケージ全体の coverage は計っていない

## Verdict

- Pass: pass。HEAD 2947c524 で `./scripts/run-test.sh` が rc 0(904 秒、shell 44 本、Go 8 パッケージ)。`go test -count=1 -timeout 30m ./internal/org/... ./internal/cli/...` が go1.26.0 で rc 0(919 秒)、`GOTOOLCHAIN=go1.25.8` で rc 0(448 秒)。名指しの 3 本は両方の版で `-count=3` の 9 回すべて PASS。AC1b は依頼の 5 つの場合の事例を持ち、`shellQuote` を外す写しと `"…"` で囲む写しを含む 6 つの mutant をすべて落とした。AC4 の雛形の mutant 6 つもすべて落ちた。既知の不安定の regate のテストは今回落ちていない
- Fail: なし
- Blocked: なし
