# Test report: org-recheck

- Date: 2026-10-26(JST 02:33〜02:59 と 12:24、UTC 2026-10-25 17:33〜17:59 と 2026-10-26 03:24)
- Plan: `docs/plans/active/2026-10-24-org-recheck.md`(Test plan の Unit と Regression。通しの実行そのものの結果は evidence の側にある)
- Tester: tester サブエージェント(Claude Opus 5.5)。パイプラインの 1 回目(cycle 1 / 2)
- Scope: HEAD f5bc474b(base origin/main 98d38391、189cd55d で取り込み済み)。差分は記録と手順書だけで、Go とシェルのコードは変わっていない。behavioral test だけを実行した(静的解析は /verify の担当)。環境は macOS(Darwin 25.5.0、arm64、12 CPU)、git 2.49.0、go1.26.0 と go1.25.8(`GOTOOLCHAIN=go1.25.8`、module cache にある toolchain)
- Evidence: run-test.sh が書いた `docs/evidence/verify-2026-10-25-173332.log`(`docs/evidence/*.log` は gitignore の対象なので、コミットしない)。Go の 3 回の出力は scratchpad に置いた
- 追跡しているファイルは、この報告と insight event のほかは変えていない。実行の前とあとで `git status --porcelain --ignored`(`.harness/cache/` を除く)を比べ、増えたのは上の verify の log の 1 行だけだった。本物の herdr・agmsg・osascript・claude・codex・tmux・gh・network を打つものは実行していない。Go の実行には `GOPROXY=off` を付け、module の取得が network に出ないようにした。利用者の `go env GOENV` のファイルは、実行のあとも存在しない(書かれていない)

## Test execution

| # | Suite / Command | Tests | Passed | Failed | Skipped | rc と時間 |
| --- | --- | --- | --- | --- | --- | --- |
| 1 | `./scripts/run-test.sh` | shell 44 本(`tests/test-*.sh` のすべて) | 44 本 | 0 | shell 1 件(下) | rc 0、697 秒(02:33:31〜02:45:08)。Language scope は `changed (docs_only)`、language pack は選ばれず、Go は走っていない |
| 2 | `GOPROXY=off go test -count=1 -timeout 30m ./...`(go1.26.0) | テストのある 8 パッケージ | 8 | 0 | - | rc 0、493 秒(02:45:09〜02:53:22)。`internal/cli` 334.205s、`internal/config` 1.794s、`internal/insights` 2.078s、`internal/org` 487.519s、`internal/org/driver` 2.190s、`internal/org/protocol` 2.458s、`internal/scaffold` 2.134s、`internal/upgrade` 1.783s |
| 3 | 2 と同じコマンドを `GOTOOLCHAIN=go1.25.8` で(`go version` は go1.25.8 darwin/arm64) | 8 パッケージ | 8 | 0 | - | rc 0、396 秒(02:53:22〜02:59:58)。`internal/cli` 309.676s、`internal/config` 2.093s、`internal/insights` 1.523s、`internal/org` 386.901s、`internal/org/driver` 2.020s、`internal/org/protocol` 1.964s、`internal/scaffold` 2.369s、`internal/upgrade` 3.912s |
| 4 | `go test -count=1 -timeout 30m -v ./internal/insights/ ./internal/cli/ -run '<下の 5 本>'`(go1.26.0) | top-level 5 本(`--- PASS` 18 行、subtest を含む) | 18 | 0 | 0 | rc 0、4 秒(12:24:15〜12:24:19) |

- 2 と 3 のどちらも、`github.com/yoshpy-dev/ralph` と `cmd/ralph` は `[no test files]`。リードの依頼のコマンドは `go test -timeout 30m ./...` で、`-count=1` は cache の結果で時間が出なくなるのを避けるために足した。`GOPROXY=off` は上に書いた理由で足した。どちらも依頼より厳しい側の変更である
- 1 の shell の 44 本は、どれも最後の行が `OK`。Skipped は前からある `tests/test-secret-scan-branch.sh` の 1 件(git 2.49.0 なので「2.41 より古い本物の git」の事例を飛ばし、stub の古い git がその分岐を見る)
- load average(1 分)は、1 の始まりが 51.37、02:35 に 126.72、02:36 に 118.44、終わりが 6.29。2 は始まり 6.11、終わり 94.45、3 は始まり 94.45、終わり 5.99。高い値は、テストと別に動いていた Xcode の `swift-frontend` と iOS Simulator のプロセスによる(02:35 に `ps` で見た)。この負荷のもとで、時間に依存するテストを含めて落ちたものはない。`docs/tech-debt/README.md` の既知の flake(`TestOrgWatchDetach_StartDetachedProcess_NewSessionDevNullStdinAppendedLog` など)も出ていない
- 1〜4 のあと、`ralph-org`、`herdr`、`agmsg` の名前を持つプロセスは残っていない

## 変更された文書を読むテスト

差分の文書(手順書の 2 写し、`docs/tech-debt/README.md`、仕様、根拠の表、evidence と写し)を実行のときに読むテストを、`tests/` と `*_test.go` で探した。

| テスト | 読むもの | 結果 |
| --- | --- | --- |
| `tests/test-no-loop-references.sh` | 本物の木を走査する。`docs/recipes/`、`docs/evidence/`、`templates/base/` を含み、`docs/specs/`、`docs/reports/`、`docs/insights/`、`docs/plans/active/`、`docs/tech-debt/README.md` は除く(`EXCLUDE_REGEX`、56 行目) | PASS(1 で実行) |
| `tests/test-template-purity.sh` の A | `scripts/check-template-purity.sh` を本物の `templates/` に打つ。手順書の `templates/base/` の写しを含む | PASS(1 で実行)。写しに `yoshpy-dev` は 0 件 |
| `tests/test-check-sync.sh` | scratch の木で `check-sync.sh` の規則を見る。本物の 2 写しは読まない | 16 / 16(1 で実行)。本物の木での一致は /verify の `check-sync.sh`(IDENTICAL 172、DRIFTED 0)と `cmp` が見ている |
| Go のテスト | 該当なし。`readRepoFileOrSkip` などで repo のファイルを読むテストは、skill、雛形、`docs/recipes/codex-seat-permissions.md`、`ralph.toml` などを読み、変わった文書はどれも読まない。templates を go:embed する root のパッケージに依存するのは `github.com/yoshpy-dev/ralph` と `cmd/ralph` だけで(`go list -deps -test` で確かめた)、どちらもテストがない | 2 と 3 は変わっていないコードの回帰として回した |

リードの依頼の 3(run-test.sh が打たなかった文書のテスト)は、該当がない。run-test.sh の test の mode は `tests/test-*.sh` をすべて打ち(`scripts/verify.local.sh` の 292 行目)、44 本の名前は 1 の出力の `==> tests/` の行と一致した。

## 計画の Test plan の項目

| 区分 | 項目 | テスト |
| --- | --- | --- |
| Unit | 新しいテストはない(記録と手順書の PR) | - |
| Unit | `./scripts/run-test.sh`、Go は `-timeout 30m`、`GOTOOLCHAIN=go1.25.8` でも | 1、2、3 |
| Integration | 通しの実行そのもの | テストのファイルではなく evidence(`docs/evidence/org-recheck-2026-10-24.md`)。この報告では打ち直していない |
| Regression | `./scripts/run-test.sh` を最後のブランチで | 1(HEAD f5bc474b)。Go の側は 2 と 3 |
| Regression | 2 つの機能の PR(#255、#256) | それぞれの org の gate と CI が確かめた。このブランチでも、2 本が足したテスト 5 本を 4 で走らせた(下の Regression checks) |
| Edge | spawn の失敗(F4)、見張りの escalation(F6・F13)、`ralph org merge` の断り(D7) | 実機で起きたときだけ確かめられる項目で、evidence の Fix effects の表が扱う。テストでは見ていない |

## Coverage

- Statement: 今回は計っていない。差分は Go のコードもシェルのコードも変えていないので、statement coverage は差分で動かない。最後の記録は `internal/org` 95.0%(`docs/reports/test-2026-10-23-busy-leader.md`)、`internal/cli` 88.4%(`docs/reports/test-2026-10-22-seat-spawn-close.md`)
- Branch: 計っていない(Go の標準の道具にない)
- Function: 計っていない
- Notes: 2 つの機能の PR のコードは main に入っていて、このブランチは 189cd55d でそれを取り込んでいる。そのテストは 2〜4 に含まれる

## Failure analysis

| Test | Error | Root cause | Proposed fix |
| --- | --- | --- | --- |
| なし | - | - | - |

## Regression checks

| Previously broken behavior | Status | Evidence |
| --- | --- | --- |
| tech-debt 145 行: backfill が `Reviewer status: incomplete` の cross-review の cycle を `pass` にしていた(#255 で直した) | このブランチでも直っている | 4 で `TestParseReport_CrossReview_IncompleteReviewerIsNA`、`TestParseReport_CrossReview_CompleteReviewerZeroCountsIsPass`、`TestParseReport_CrossReview_OnlyLaterCycleIncomplete` が PASS(SKIP 0) |
| tech-debt 126 行: doctor が文字列でない matcher を見逃し、`[hooks]` の子が `state` だけでも併存の警告を出していた(#256 で直した) | このブランチでも直っている | 4 で `TestValidateCodexHooksJSON_MatcherType`、`TestCheckCodexEffectiveConfig_HooksTableCoexistence` が PASS(SKIP 0) |
| 既知の flake(doctor の stub の PATH、`TestRegate_FetchFailures` など) | 出ていない | 2 と 3 が高い負荷(1 分の load average が最大 94〜127)のもとで rc 0 |

## Test gaps

1. 変わった文書の中身を見る Go のテストはない(上の表)。手順書の文、tech-debt の行、仕様の印、evidence の表は、テストではなく /self-review と /verify の突き合わせ(evidence の `sh` ブロック 29 個の再実行、`check-sync.sh`、`cmp`)で確かめられている
2. 手順書のコマンドを頭から打つことはテストにない。本物の herdr と claude が要り、依頼の規則で打たない。手順書が書いてあるとおりに通るかの根拠は、evidence の実機の実行である
3. 2 つの写しが同じバイト列であることは `check-sync.sh`(静的解析)が見る。`tests/test-check-sync.sh` は規則を scratch の木で見るだけで、本物の写しを比べない
4. coverage は今回計っていない(上の Coverage)

## Verdict

- Pass: pass。HEAD f5bc474b で `./scripts/run-test.sh` が rc 0(697 秒、shell 44 本、scope は docs_only で Go なし)、`go test -count=1 -timeout 30m ./...` が go1.26.0 で rc 0(493 秒、8 パッケージ)、`GOTOOLCHAIN=go1.25.8` で rc 0(396 秒、8 パッケージ)。#255 と #256 が足したテスト 5 本も PASS。落ちたテストも再実行もない。変わった文書を読む Go のテストはない(穴 1)
- Fail: なし
- Blocked: なし
