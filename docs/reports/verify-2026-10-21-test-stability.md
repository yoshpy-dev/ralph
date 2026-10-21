# Verify report: test-stability

- Date: 2026-10-22
- Plan: `docs/plans/active/2026-10-21-test-stability.md`(Status: Approved。`./scripts/plan-visual.sh digest` は `edae494b2ed1` で、`- Approved:` 行と一致)
- Verifier: verifier サブエージェント(Claude Code、パイプライン cycle 1 / 2)
- Scope: `git diff 293dff17...HEAD`(HEAD 44c9f87a、18 ファイル、+1086/-35)。S1〜S6 の slice、self-review の LOW の直し 6d990dfc、self-review の報告と insight event を含む。判定は HEAD の tree に対して行った。self-review の報告 `docs/reports/self-review-2026-10-21-test-stability.md` は Merge: yes
- Evidence: `docs/evidence/verify-2026-10-21-test-stability.log` と、run-static-verify.sh が書いた `docs/evidence/verify-2026-10-21-194413.log`。どちらも `docs/evidence/*.log` の gitignore によりコミットしない生の出力で、ホームのパスは `~` に置き換えた

## Spec compliance

テストの実行は `/test` の担当なので、AC1〜AC5 はコードとテストの本文を読んで確かめた。計画と tech-debt の行が書く実行結果(20/20、10/10、mutant の失敗数)は implementer の記録で、ここでは打ち直していない。mutant の失敗数は、コードとテストの本文から期待される数を数え、記録と一致することだけを確かめた。

| Acceptance criterion | Status | Evidence |
| --- | --- | --- |
| AC1: stub が stdout を開けたまま stderr を `/dev/null` に送り、20 s 眠ってから marker を書く。marker があれば subtest が落ちる。20/20 と、2 s の bound での base と新しい stub の結果、`exec.Command` の mutant | コードは満たす。実行は `/test` | `internal/org/regate_test.go:765` の stub は `exec 2>/dev/null </dev/null`、`sleep 20`、`: > <marker>` で、fd 1 は閉じない。同じ case の `t.Cleanup`(769 行)が `fileOrAbsent(t, ended) != "<absent>"` で `t.Errorf` する。`newRegateFixture` の `t.TempDir()`(55 行)が先に登録されるので、LIFO で marker の確認は temp dir の削除より前に走る。runner は `gitCtx.Err()` を exit code より前に読む(`internal/org/regate_git.go:199`)ので、marker がなければ bound で止めた fetch と stub の終わりで終わった fetch を見分けられない。コメントの理由はこの実装と合う。記録された結果はコードから導ける。base の stub では git が約 0.2 s で exit 128 になり、2 s の bound より先に「failed (exit 128)」になる。`exec.Command` の mutant では git が stub の 20 s を待ち、そのあいだに期限が切れるので「ran past」は通るが marker が残る。`regate_git.go` は diff で変わっていない(Non-goals のとおり) |
| AC2: fake の `go` で 5 つの `GOFLAGS` の形と static mode、本物の `go` で 4 つの期限、2 つの `verify.sh` が同一 | 満たす。suite の実行は `/test` | `packs/languages/golang/verify.sh:114-115` は `goflags=$(go env GOFLAGS 2>/dev/null) \|\| goflags="${GOFLAGS:-}"` のあと、`GOFLAGS="-timeout=30m${goflags:+ $goflags}" go test ./...` を走らせる。前置の代入なので、`GOFLAGS` は `go test` にだけ渡る。`tests/test-golang-pack-test-timeout.sh` は AC2 の各事例を 1 つずつ持つ。no GOFLAGS(181 行)、`-count=1`(184)、引用つきの項目(187)、`go env -w` の値(190)、`go env` の失敗(193)、static mode の go test なしと go env なし(196-198)、go がなければ `SKIP: real go cases`、本物の go の 30・45・45・45(288-298)。check の数は fake 24(6 + 5 × 3 + 3)と本物 10(2 + 4 + 4)で計 34 になり、記録と合う。mutant m1〜m4 の失敗数 11・3・11・13 も、事例ごとに数えると同じになる(m1: fake 8 と本物 3、m2: fake 2 と本物 1、m3: fake 10 と本物 1、m4: fake 10 と本物 3。m1 は、既定を後ろに置き、`GOFLAGS` が空なら既定だけになる形として数えた)。2 つの `verify.sh` と README は `cmp` で同一、`diff -r packs/languages/golang templates/packs/golang` は rc 0、check-sync は DRIFTED 0 |
| AC3: scratch の tree で `review-x.md`・`org-manifest-x.md` が rc 0、`other-x.md` を足すと rc 1 で ROOT_ONLY、`reviewer.md`・`x-review-y.md` も ROOT_ONLY。base の script から除外を 1 つ消すと suite が落ちる | 満たす。suite の実行は `/test` | `tests/test-check-sync.sh` は 7 回の実行で 16 check(1 + 3 + 4 + 2 + 2 + 2 + 2)。`check-sync.sh` は `find` で走査し(git を使わない)、`is_excluded` は先頭の前方一致(`"${pattern}"*`)なので、事例の期待はこの実装と合う。script の複製は `scripts/check-sync.sh` の除外に入るので、tree に template の複製はいらない。記録された mutant の失敗数も事例ごとに数えると同じになる。`review-` を消すと 3、`org-manifest-` を消すと 3、`docs/reports/` を足すと 6、部分一致では nested の 2、F8 より前の script(9349914b^)では 5 |
| AC4: `probeBinary` が `doctorProbeTimeout`(既定 5 s)を読み、timeout の文が bound を言う。`TestMain` が上げる。100 ms の bound で「timed out after 100ms」。`CorruptManifest` の試験が doctor の出力を出す。3 つの試験が単独で 10/10 | コードは満たす。10/10 は `/test` | `internal/cli/doctor.go:235` `var doctorProbeTimeout = 5 * time.Second`、252 行の `context.WithTimeout(..., doctorProbeTimeout)`、256 行の `"%s --version timed out after %s"`。`time.Duration` の `String()` は 5 s を `5s`、100 ms を `100ms` と書くので、既定の文は前と同じになる。`internal/cli/main_test.go:113` が `m.Run()`(131 行)より前で 1 分に上げる。`TestProbeBinary_TimeoutNamesTheBound`(`cli_test.go:391`)は `exec /bin/sleep 2` の shim と 100 ms の bound で、変数を無視する probe なら sleep が終わって timeout の文が出ない。`doctor_scaffold_test.go:457`・`461` が `doctor output:` を添える。`internal/cli` に `t.Parallel()` はない(Assumptions のとおり) |
| AC5: 200 byte を超え、200 byte 目が字の途中にある日本語の本文を `DryRun: true` で送り、dry-run の `sent` の Details が有効な UTF-8 で U+FFFD を含まず `...(truncated)` で終わり、残す前半が 200 byte 以下。mutant で落ちる | コードは満たす。mutant の実行は `/test` | `TestOrgSend_DryRun_JapaneseBodyOver200Bytes_DetailsStayValidUTF8`(`internal/org/verbs_test.go:1392`)。本文は 385 byte で、200 byte 目は `0x82`(続きの byte)なので前提の検査を通る(python で数えた)。dry-run の枝(`internal/org/verbs.go:345-353`)は `truncateForDetails` を呼んで `findSeat` より前に返るので、spawn なしで動く。driver の呼び出しがないこと、最後の event が `sent` で `DryRun` が真であること、4 つの Details の検査がある |
| AC6: tech-debt の 5 行、根拠の表の F9・F10・G1・F12・F14、仕様の印 21 と 11、Status、FR-7、根拠の表の 20 と 11、PR の一覧 29 本、「NFR 入れ方」の 26、#246 が MERGED | 満たす | 下の「記録の確かめ」 |
| AC7: `./scripts/run-verify.sh` が rc 0(`GOFLAGS=-timeout=30m`)、`GOTOOLCHAIN=go1.25.8 go test -timeout 30m ./...` が通る、`env -u GOFLAGS ./scripts/run-test.sh` が rc 0 | 静的な部分だけ確かめた。3 つとも `/test` に残す | `GOFLAGS=-timeout=30m ./scripts/run-static-verify.sh` が rc 0、`GOTOOLCHAIN=go1.25.8 go vet ./internal/...` と go1.26.0 の `go vet ./...` が rc 0(vet はテストファイルもコンパイルする)。run-verify のテスト部分、go1.25.8 の `go test`、`GOFLAGS` なしの `run-test.sh` は回していない。orchestrator の `run-verify.sh` が背後で走っていたが、その結果は読んでいない |

### 記録の確かめ(AC6)

- tech-debt の 5 行(`docs/tech-debt/README.md` の 305・311・314・332・335 行と、その上の 304・310・313・331・334 行のコメント)は、隣の #244 の行(322・323)と #246 の行(325・326、328・329)と同じ形をしている。上に `<!-- RESOLVED 2026-10-22 in fix/test-stability (S<n> <sha>): … Row preserved for traceability. -->` を置き、Debt 欄は打ち消して `(RESOLVED …)` を添え、Trigger 欄は打ち消して `— RESOLVED:` を添え、Related 欄に計画を足す。5 行とも 5 欄
- コメントの SHA と commit の中身: S1 3f8ccd9b は `regate_test.go` だけ、S2 db1e5ac7 は `verify.sh` 2 つ・README 2 つ・新しい suite、S3 dc58715a は `tests/test-check-sync.sh`、S4 439ae909 は `doctor.go`・`main_test.go`・`cli_test.go`・`doctor_scaffold_test.go`、S5 43df3185 は `verbs_test.go` だけで、どれもコメントが書く内容と合う。記録は S6 58f2624f が書き、4f83c84c が FR-7 の注から AC7 の run-test.sh の節を外した。LOW の直し 6d990dfc は 2 つの suite の trap と変数名を変えただけで、assert の数と事例は変わらないので、S2・S3 のコメントの数(34 = 24 + 10、16)は今も正しい
- S4 のコメントと `TestMain` のコメントが挙げる 4 つ目の試験 `TestRunDoctorOpts_HerdrAgmsgAbsent_ExitCodeUnaffected` は、`docs/reports/test-2026-10-13-org-plan-approval.md:265` に R03 と R04 の 2 回の mutant 実行で `claude --version timed out after 5s` として出てくる。「3 つが timeout の文を出した」は、行の 3 試験のうち timeout の文を出した 2 つ(MergeGuarantee、CodexSandbox)と HerdrAgmsgAbsent を合わせた数で、合う
- 仕様の印: FR 12、NFR 5、受け入れ条件 15 の計 32 項目で、`[x]` 21、`[ ]` 11。base(293dff17)は 20 と 12 で、変わったのは FR-7 だけ。`[ ]` の 11 項目はどれも「印(2026-10-21)」の注を持つので、Status の「理由を項目の下の『印(2026-10-21)』に書いた」は今も正しい。Status は「11 項目は `[ ]` のまま残る」「29 本」「試行の所見 F12・F14 の直し: #246」を持ち、食い違う所見の一覧は F3、F4、F7、F13、D-F3、「leader への ALERT」で、F10 は消えている。この一覧は `[ ]` の項目の注が挙げる所見の和集合と一致し、根拠の表で「食い違う」が「はい」の 6 行とも一致する
- FR-7 は `[x]` で、注「印(2026-10-22)」がこのブランチの名前、pack の既定にした `-timeout=30m`、`tests/test-golang-pack-test-timeout.sh`、負荷の下の regate を打ち直していないことを書く。8a-3b の注の FR-7 の文も「F10 を直したので `[x]` にした」に変わった
- 根拠の表: 31 項目で `[x]` 20、`[ ]` 11、ほかの判定はない。数の行(79 行)の `[ ]` の一覧は表と一致し、FR-7 がない。仕様の印と 1 項目ずつ比べると、違いは表にない FR-12 だけで、印の食い違いは 0。表の前文(3 行)が FR-12 を入れないと書いているので、self-review が `/verify` に回した 21 と 20 の差はこれで説明がつく
- 所見の表: F9・F10・G1 の「直す PR」は「テストの安定(fix/test-stability で直した)」、F12・F14 は「#246 で直した」。F10 の「食い違う」は「いいえ(今は)」に変わった。表の前の文(16 行)は、merge した PR を番号で書く規則と F8・F12・F14 の例を書く
- PR の一覧: 根拠の表の「段ごとの PR」と仕様の Status の PR 番号は同じ 29 本。2026-10-22 に `gh pr view <n> -R yoshpy-dev/ralph --json state --jq .state` で 29 本すべてを読み直し(読むだけ、active account は yoshpy-dev)、すべて `MERGED` だった。「NFR 入れ方」の 26 本(9 段の PR 24 本と #244・#246)は 29 − 3(#205、#241、#242)と合い、#246 の計画と 4 つの report は HEAD にある
- tech-debt の行を名指す引用は、打ち消した行では `"~~<先頭の数語>~~"` の形で、`~~` を外すとどれも 1 行だけに当たる(V3)。根拠の表が挙げるテストの関数名 65 個と仕様が挙げる 20 個(重なりを除いて 71 個)は、どれも `func <名前>(` が repo に 1 件だけある

計画のほかの項目:

- Assumptions: `internal/cli` に `t.Parallel()` はない。`scripts/verify.local.sh:292` は `tests/test-*.sh` を glob で拾うので、2 つの suite は一覧に足さなくても走る。`ralph org gate` は `scripts/run-test.sh`(`internal/org/gate.go:36`)を打ち、`run-verify.sh:172` が pack の `verify.sh` を選ぶので、gate と regate も同じ `run_tests` を通る。`internal/upgrade/replaceplan.go` の `PreservePrefixes` の doc は、template に中身のある pack のパスを普通に分類すると書いており、「upgrade が pack の変更を届ける」という前提と合う(upgrade は打っていない)
- Non-goals: `regate_git.go`、`verbs.go`、`internal/upgrade/` は diff で変わっていない。`probeBinary` の利用者向けの 5 s も変わらない

## Static analysis

| Command | Result | Notes |
| --- | --- | --- |
| `GOFLAGS=-timeout=30m ./scripts/run-static-verify.sh`(HEAD 44c9f87a) | rc 0、`All verifiers passed.` | `Language scope: full fallback (shared:packs/languages/golang/README.md)` で、選ばれた pack は golang だけ。changed scope の指定が full に落ちたので、この run は full gate と同じ範囲になる。shellcheck(severity warning)、hook の `sh -n`、guard の awk の構文、settings.json の jq、Codex の 3 つの guard、check-sync(IDENTICAL 171、DRIFTED 0、ROOT_ONLY 0、TEMPLATE_ONLY 11、KNOWN_DIFF 5)、check-pipeline-sync、check-skill-sync(14 skill)、check-template-purity、tech-debt README の計画の参照が OK。golang は `gofmt: ok`、golangci-lint の `0 issues.`、go vet と staticcheck は無出力。secret scan は 293dff17..44c9f87a で clean。開始時の load average は 9.12 |
| `GOTOOLCHAIN=go1.25.8 go vet ./internal/...` | rc 0 | go.mod は `go 1.25.8`、手元の go は go1.26.0 |
| `go vet ./...`(go1.26.0) | rc 0 | |
| `gofmt -l` で変わった Go ファイル 6 つ | 出力なし | |
| `sh -n` と `bash -n` で `tests/test-check-sync.sh`、`tests/test-golang-pack-test-timeout.sh`、2 つの `verify.sh` | 8 回とも rc 0 | 2 つの suite と 2 つの `verify.sh` は mode 100755 |
| `shellcheck`(全 severity)で 2 つの suite と `packs/languages/golang/verify.sh` | suite は指摘なし。`verify.sh:15` に SC2329(info) | SC2329 は trap から呼ぶ `cleanup` についての info で、base の `verify.sh` でも同じ指摘が出る(この PR が足したものではない) |
| `git diff --check 293dff17...HEAD` | rc 0 | 足した行にホームのパスと EF BF BD の byte 列はない。TODO・XXX の一致は `mktemp` の `XXXXXX` と self-review の文だけ |

## Documentation drift

| Doc / contract | In sync? | Notes |
| --- | --- | --- |
| Go pack の README(`packs/languages/golang/README.md`、`templates/packs/golang/README.md`) | 合う | 「`GOFLAGS` の先頭に `-timeout=30m`、go test 自身の既定は 10 分、`GOFLAGS` か `go env -w` で決めた timeout があとに来て勝つ、`GOFLAGS=-timeout=10m` で 10 分」は `run_tests` の実装と合う。2 つは `cmp` で同一 |
| Go pack が go の既定の bound で走ると書く文書、run-verify や gate に `GOFLAGS=-timeout=30m` を付けるよう書く文書 | ない | lead の `git grep -n -E 'GOFLAGS=-timeout\|go test \./\.\.\.\|10-minute\|10 分' -- docs/recipes docs/quality README.md .claude/skills .agents/skills packs templates` で出るのは、`ack_minutes` と `deadman_minutes` の 10 分、`/director` の merge の Bash の timeout 10 分(806 行、`ralph org merge` の話で、gate のテストの bound ではない)、pack の `rule.md` の `go test ./...`(完了前に打つことを言うだけ)、README と `verify.sh` の新しい文だけ。records(reports、evidence、plans、insights、tech-debt、specs)と `tests/` の外で `-timeout=30m` を書くのは pack の README と `verify.sh` だけ。`.github/workflows` に `GOFLAGS` や timeout の指定はない |
| `docs/recipes/` の言語 pack の文書 | 合う | `docs/recipes` と `docs/quality` に golang、go test の bound を書く文はない |
| doctor の probe の 5 s を書く文書 | 合う | README、recipes、quality、specs、rules、`internal` のテスト以外で、probe の 5 s を書く文はない。利用者向けの 5 s は変わっていない |
| `templates/packs/golang/` と `packs/languages/golang/` | 同一 | `diff -r` rc 0、check-sync の pack の節も差分なし |
| 計画の Risks と Progress | Progress だけ進んでいる(V1、V2) | 下の Findings |

## Observational checks

- 計画の digest を計算し直して `edae494b2ed1` を得た。`- Approved:` 行と一致するので、承認のあとに変わったのは digest の範囲の外(Progress、AC の checkbox)だけである
- `gh pr view` を 29 回打った。書き込みの操作はしていない

## Findings

| ID | Severity | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- |
| V1 | INFO | 計画の Risks が、Progress の記録で誤りとわかった文をそのまま残している。Risks の 3 つ目は「invalid な `GOFLAGS` では `go env` が失敗する」と書くが、S2 の Progress の注は、go1.26 は `notaflag` を rc 0 で出すので Risks のその文は誤りで、動きは同じだと記録する | `docs/plans/active/2026-10-21-test-stability.md:312-313` と 350-353 | 動きは変わらないので直さなくてよい。直すなら `/sync-docs` で Risks のその文に「Progress の S2 を見よ」と 1 行足す |
| V2 | INFO | 計画の Progress checklist の「Review artifact created」が `[ ]` のまま。self-review の報告は c02647ca でコミットされ、LOW の直しの記録も 44c9f87a で入っている | 計画の 389 行 | `/sync-docs` か `/pr` でほかの項目と一緒に付ける |
| V3 | INFO | 打ち消した tech-debt の行を名指す引用は、`~~` を先頭の数語の直後で閉じる。行の最初のセルの `~~` はもっと後ろで閉じるので、字のまま前方一致させると当たらない。`~~` を外すとどれも 1 行だけに当たる(305、311、323、326、329、332 行) | 根拠の表の F8・F9・F10・F12・F14・G1 の行と FR-7 の行、仕様の FR-7 の注。evidence の log の 8 節 | #244・#246 と同じ書き方で、根拠の文書の規則(最初のセルの先頭の数語で指す)は中身として満たす。直す必要はない |
| V4 | INFO | merge のあと、根拠の文書 16 行の規則(merge した PR は番号で書く)が、F9・F10・G1 の「テストの安定(fix/test-stability で直した)」と、FR-7 の 2 つの注の「fix/test-stability」にも当てはまる。この PR が #246 の F12・F14 を番号に直したのと同じ更新が、系列の次の PR に要る | 根拠の表 28・29・36・53 行、仕様の FR-7 の注 | 今は PR の番号がないので、ずれではない。系列の次の PR の計画に入れる |

CRITICAL、HIGH、MEDIUM、LOW はない。

## Coverage gaps

- AC7 の 3 つ(`GOFLAGS=-timeout=30m` の `./scripts/run-verify.sh`、`GOTOOLCHAIN=go1.25.8 go test -timeout 30m ./...`、`env -u GOFLAGS ./scripts/run-test.sh`)は `/test` に残す。orchestrator が背後で回している `run-verify.sh` の結果は読んでいない
- AC1〜AC5 の実行結果(F9 の subtest の `-count=20` を両方の toolchain で、doctor の 3 試験の 10/10、2 つの suite の 16 と 34、2 s の bound での base と HEAD、`exec.Command` の mutant、AC3 と AC5 の mutant)は打ち直していない。mutant の失敗数(AC2 の 11・3・11・13、AC3 の 3・3・6・2・5)は、コードとテストの本文から事例ごとに数えた期待と一致することだけを確かめた
- 6d990dfc のあとの 2 つの suite は、implementer の報告(16/16、両方の toolchain で 34/34、TERM で rc 1)だけが実行の根拠で、`/test` で打ち直す
- `ralph upgrade` が pack の `verify.sh` を下流に届けることは、`replaceplan.go` の doc と計画の前提を読んだだけで、upgrade は打っていない。下流で `verify.sh` を書き換えた project にどう届くかは確かめていない
- F9 が重い負荷の下でだけ起きた理由は、記録が書くとおり説明されていない。負荷の下での regate も打ち直していない(計画の Non-goals)
- insight event: `./scripts/insights-append.sh --slug test-stability --flow standard --phase verify --verdict pass --cycle auto --source skill` で 1 行を足し、この報告と同じ commit に入れた
- 足りない決定的な verifier はない。この PR の 2 つの suite が、F10 と G1 の検査を `tests/` に足した

## Verdict

- Verdict: pass(CRITICAL、HIGH、MEDIUM、LOW はない。INFO 4 件(V1〜V4)はどれも記録の文面で、V1 と V2 は `/sync-docs` で直せ、V4 は系列の次の PR の作業)
- Verified: AC6 の記録(tech-debt の 5 行の形と 5 欄、コメントの SHA と commit の中身、仕様の 32 項目の印 21 と 11、Status の数と所見の一覧、根拠の表の 31 項目 20 と 11、仕様の印との 1 項目ずつの一致、29 本の PR が `gh` で `MERGED`、26 本の数、#246 の計画と 4 つの report、引用の行の解決、71 個のテスト名)、AC1〜AC5 のコードとテストの本文の対応、記録された check の数と mutant の失敗数がコードから数えた数と合うこと、静的解析(run-static-verify rc 0、go1.25.8 と go1.26.0 の vet、gofmt、`sh -n` と `bash -n`、shellcheck、`git diff --check`)、文書のずれ(pack の README、lead の grep、recipes、doctor の文書、2 つの pack の同一)、計画の digest
- Partially verified: AC7(静的な部分だけ)、AC1〜AC5(実行は `/test`)
- Not verified: Go のテストと 2 つの shell suite の実行、mutant の実行、`ralph upgrade` での配布、負荷の下での F9 と regate
