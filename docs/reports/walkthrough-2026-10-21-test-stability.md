# Walkthrough: test-stability(テストの安定: F9、F10、G1、doctor の試験、dry-run の切り方)

- Date: 2026-10-22
- Plan: docs/plans/active/2026-10-21-test-stability.md(/pr で archive に移す)
- Branch: fix/test-stability(base 293dff17、この walkthrough の前の HEAD a41bad68。24 commits、22 files、+1576/-35)
- 読む範囲: コードとテストは 12 files(+610/-14)。そのうち pack の `run_tests` は 2 つの写しで 8 行ずつ、Go の本体は `internal/cli/doctor.go` の 12 行だけで、残りはテストと新しい shell の suite 2 本。記録は 3 files。残りは計画(約 420 行)、後工程の報告 5 本と triage、insight event 5 行
- 後工程は 1 回(cycle 1)で終え、cross-review の指摘は 0 件

```
何を直すか → 計画 → stub(S1) → pack の timeout(S2) → check-sync の suite(S3) → doctor(S4) → dry-run(S5) → 記録(S6) → 後工程と、その中の直し → AC の根拠 → 確かめていないこと
```

## 1. この PR が何か

- 試行の所見を直す系列の 3 本目(1 本目は仕様の印の #245、2 本目は記録の文字の #246)
- コードではなく、テストか runner が原因で落ちる失敗を止める
  - F9: `TestRegate_FetchFailures/the_fetch_runs_past_the_bound` の stub が git の読む stdout を閉じていたので、git が 0.2 秒ほどで exit 128 になり、300 ms の bound と競っていた
  - F10: Go の pack の `run_tests` に `-timeout` がなく、`GOFLAGS` のない shell から打った gate と regate が go の既定の 10 分で切れた
  - doctor の試験: `probeBinary` が stub の `--version` に 5 秒しか与えず、重い負荷の下で落ちていた
- テストの穴を 2 つ埋める
  - 根拠の表の G1: F8 を直した check-sync の 2 つの除外を走らせる suite がなかった
  - #246 の test report の G1: dry-run の `sent` の本文の切り方を確かめる試験がなかった

## 2. 読む順(commit ごと)

1. 8a7a519a 計画
   - Objective、Scope(S1〜S6)、Non-goals、Assumptions、Design decisions、AC1〜AC7 を読む。承認の digest は `edae494b2ed1` で、後工程のあいだ本文は変えていない(Progress checklist と AC の checkbox だけが動く)
   - Design decisions の要点: timeout はコマンドラインに付けず、`GOFLAGS` の先頭に置く。コマンドラインの flag は `GOFLAGS` より強いので、付けると打った側の `GOFLAGS=-timeout=60m` を上書きする。最初の案は値の中から timeout を探したが、Codex の計画の助言で、Go が受け付ける引用付きの形を見落とすことが分かり、この形にした
   - F9 の stub に marker を足したのも Codex の助言から。runner は git の終了コードより先に期限切れを読むので、stub が終わったせいで終わった fetch も「ran past」と読まれる
   - critical fork はなし
2. 3f8ccd9b S1 stub(F9)
   - `internal/org/regate_test.go:764-766`。stub は `exec 2>/dev/null </dev/null` で stdout を開けたまま 20 秒眠り、そのあと `upload-pack-ended` を書く。case の `t.Cleanup` が、Regate が戻ったあとに marker があれば subtest を落とす
   - 見ること: コメントが原因(閉じた stdout で git が EOF を読む)と、marker がなぜ要るかを言っているか。`t.TempDir()` の削除より前にこの cleanup が走る順序(fixture が先に `t.TempDir()` を登録する)
3. db1e5ac7 S2 pack の timeout(F10)
   - `packs/languages/golang/verify.sh:108` の `run_tests`。`goflags=$(go env GOFLAGS 2>/dev/null) || goflags="${GOFLAGS:-}"` で実際の値(変数、なければ `go env -w` のファイル)を読み、`GOFLAGS="-timeout=30m${goflags:+ $goflags}" go test ./...` を打つ。`templates/packs/golang/verify.sh` は同じ中身
   - 両方の README に、`-timeout=30m` が先頭に入ることと、`GOFLAGS=-timeout=10m` で短くできることを書いた
   - `tests/test-golang-pack-test-timeout.sh`: 偽の go(受け取った `GOFLAGS` と引数を書く)で 24 件、本物の go の scratch の module(テストが `t.Deadline()` を書く)で 10 件。利用者の go.env は読まず書かない(scratch の `GOENV`)
   - 見ること: static の mode では `go test` を打たないこと、`go env` が失敗したときの扱い
4. dc58715a S3 check-sync の suite(根拠の表の G1)
   - `tests/test-check-sync.sh`(16 件)。`scripts/check-sync.sh` の写しを、check-sync が通る scratch の tree で走らせる。`review-` と `org-manifest-` の report で rc 0、`other-x.md` を足すと rc 1。前方一致の境目(`reviewer.md`、`x-review-y.md`、別の directory の下)も見る
   - `CHECK_SYNC` で写す元を変えられるので、mutant を同じ事例に流せる
5. 439ae909 S4 doctor
   - `internal/cli/doctor.go:232-256`。bound を package の変数 `doctorProbeTimeout`(利用者向けは 5 秒のまま)にし、timeout のエラーはその値を書く
   - `internal/cli/main_test.go:39-47,113`。`TestMain` が 1 分に上げる。コメントは 2026-10-13 の 4 本目の失敗も挙げる
   - `internal/cli/cli_test.go:391` の `TestProbeBinary_TimeoutNamesTheBound`。stub は `exec /bin/sleep 2` で、100 ms の bound を超え、5 秒には届かない。変数を無視する probe はここで落ちる
   - `internal/cli/doctor_scaffold_test.go`: 落ちたときに doctor の出力を出す
6. 43df3185 S5 dry-run(#246 の G1)
   - `internal/org/verbs_test.go:1392` の `TestOrgSend_DryRun_JapaneseBodyOver200Bytes_DetailsStayValidUTF8`。1350 行の実際の送信の試験と同じ本文を `DryRun: true` で送る。spawn しない(dry-run の枝は `findSeat` より前で戻る)ので、driver を呼ばないことも確かめられる
7. 58f2624f、4f83c84c S6 記録
   - `docs/tech-debt/README.md`: F9、F10、doctor の試験、G1、dry-run の切り方の 5 行を、#244・#246 の行と同じ形で RESOLVED にした(上に RESOLVED のコメント、1 つ目と 4 つ目の欄を打ち消す)。doctor の行の書き出し「Two doctor tests」は 3 本を挙げていたので「Doctor tests such as」にした
   - `docs/evidence/org-spec-evidence-2026-10-21.md`: F9・F10・G1 の「直す PR」はこのブランチ、F12・F14 は #246。F10 の「食い違う」は「いいえ(今は)」。FR-7 は `[x]`。数の行は `[x]` 20、`[ ]` 11。PR の表に #246 を足して 29 本。「NFR 入れ方」は 26 本
   - `docs/specs/2026-10-07-org-multi-org-director.md`: Status に #246、29 本、11 項目。FR-7 を `[x]` にし、印の注を 2026-10-22 にした。仕様の印は `[x]` 21、`[ ]` 11(FR-12 を含む 32 項目)
   - 4f83c84c は、まだ打っていなかった AC7 の run-test.sh を根拠として書いた句を外した(consult の指摘)
8. 後工程と、その中の直し
   - c02647ca self-review(Merge: yes、LOW 3)。6d990dfc で L1(新しい suite が信号のあとも続く)と L3(1 つのパスに 2 つの名前)を直し、44c9f87a でそれを報告に書いた。L2(2 つの試験の検査の重複)は残した
   - 24df95dd verify(pass、INFO 4)
   - d4918ed9 test(pass)。G1(`TestMain` の引き上げを確かめる試験がない)を ec852215 の `TestMain_RaisesTheDoctorProbeTimeout`(`cli_test.go:415`)で埋め、d34e60c6 で報告に書いた
   - 621a815e sync-docs(tech-debt の RESOLVED のコメント 3 つを test の結果に合わせた)
   - 5e69d002 cross-review の triage(指摘 0)、a41bad68 その insight event
   - 計画の Progress: 621d5e82、c4164f68、143697c7、818bd4b1、37c01116、4416c4c9

## 3. AC の根拠

| AC | 根拠 |
|----|------|
| AC1 | S1。F9 の subtest は go1.26.0 と go1.25.8 で 20/20。bound を 2 秒にすると、base の stub は「failed (exit 128)」、新しい stub は「ran past 2s」。`exec.Command` の mutant は marker の検査で落ちる(test report の 11) |
| AC2 | S2。pack の suite 34/34 を両方の toolchain で。mutant m1〜m4 は 11・3・11・13 件落ちる |
| AC3 | S3。suite 16/16。F8 の除外を 1 つ消すと 3 件、`docs/reports/` を足すと 6 件、部分一致は 2 件、F8 より前の script は 5 件落ちる |
| AC4 | S4 と ec852215。doctor の 3 試験と timeout の試験は両方の toolchain で 40/40。D1・D2 は timeout の試験だけが落ち、D3 は ec852215 の試験だけが落ちる |
| AC5 | S5。M1(切らない)と M2(byte で切る)は、`internal/org` の全体で新しい試験だけが落ちる |
| AC6 | S6。verify が数(21・11、20・11、29、26)と RESOLVED のコメントの SHA を数え直して合った |
| AC7 | `GOFLAGS=-timeout=30m ./scripts/run-verify.sh` は 818bd4b1 で rc 0(13 分)。`env -u GOFLAGS ./scripts/run-test.sh` は 37c01116 で rc 0(773 秒)。`GOTOOLCHAIN=go1.25.8 go test -count=1 -timeout 30m ./...` は rc 0(332 秒) |

## 4. 確かめていないこと

- ec852215 と後の記録のコミットのあとに、run-verify の全体は打ち直していない。ec852215 はテスト 1 本の追加で、両方の toolchain と D3 の写しの `internal/cli` の全体(813 本が通過)で確かめた。CI の verify(ubuntu)が HEAD で全体を走らせる
- Linux では、手元で走らせていない(CI が見る)
- 実際の project での `ralph upgrade` は打っていない。pack が届くことは `internal/upgrade/replaceplan.go` とその試験で読んだ範囲
- 負荷の下での live の regate と試行は打ち直していない(計画の Non-goals)
- doctor の試験の元の失敗(stub の `--version` が 5 秒を超える)は、load 150 でも再現しなかった。負荷のほかの条件は分かっていない
- base の F9 の subtest が、1 つの process の繰り返しで落ちやすい理由は確かめていない
