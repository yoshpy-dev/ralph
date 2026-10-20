# Test report: org-trial

- Date: 2026-10-20(UTC 09:39〜09:59)
- Plan: docs/plans/active/2026-10-20-org-trial.md(仕様 `docs/specs/2026-10-07-org-multi-org-director.md` の 9 段目、FR-12。2 つの機能の org の試行と、既定の見直しの記録)
- Tester: tester subagent (Claude Opus 5.5)。パイプラインの 1 回目
- Scope: HEAD 029aa209(merge-base と origin/main はどちらも a1ce47d8)。`git diff --name-only origin/main...HEAD` の 28 ファイルはすべて `docs/` の下にあり、Go のコード、script、skill、template は変わっていない。029aa209 は verify のあとのコミットで、証拠の Snapshot の表の 1 行、F11、F14、仕様の FR-12 の注に 1 文ずつ足し、計画の進み具合の印を 2 つ付けた。環境は macOS(arm64、Darwin 25.5.0)、`/bin/sh` は bash 3.2.57、jq 1.8.2、git 2.49.0。load average(1 分)は開始時 11、run-test.sh の途中で 30
- Evidence: `docs/evidence/test-2026-10-20-org-trial.log`(下の各実行の生の出力)と、run-test.sh が書いた `docs/evidence/verify-2026-10-20-094029.log`。どちらも `docs/evidence/*.log` の gitignore の対象で、コミットしない
- 追跡しているファイルのうち、書いたのはこの報告と insight の 1 行だけ。比べるための写しと harness は scratchpad に置いた

## Test execution

| Suite / Command | Tests | Passed | Failed | Skipped | Duration |
| --- | --- | --- | --- | --- | --- |
| 1. `GOFLAGS=-timeout=30m ./scripts/run-test.sh` | shell の suite 41 本(数えた assertion は 4,258) | 41 本すべて OK | 0 | 1(下) | 639 秒、rc 0 |
| 2. 証拠の `sh` block 17 個を worktree の root で 1 つずつ `/bin/sh -c` で走らせ、直上の表の本文か直下の `text` block と 1 行ずつ比べる | 17 | 17 | 0 | 0 | 数秒、rc 0 |
| 2b. 2 の対照。期待値を 2 か所変えた写し(371 行の 423 を 424、421 行の `notified events: 0` を 1)に同じ harness をかける | 17 | 15 | 2(狙いどおり) | 0 | rc 1 |
| 3a. tech-debt の行の形の awk | 1 | 1 | 0 | 0 | `22: 7`、`32: 5`、`278: 10` だけ |
| 3b. `./scripts/plan-visual.sh digest docs/plans/active/2026-10-20-org-trial.md` | 1 | 1 | 0 | 0 | `5b82ee1e30a1`、rc 0 |
| 3c. `./scripts/check-sync.sh` | 1 | 1 | 0 | 0 | rc 0、IDENTICAL 171、DRIFTED 0 |
| 追加 A. snapshot の承認済みの計画 3 つを `plan-visual.sh digest` にかけ、`- Approved:` 行と比べる | 3 | 3 | 0 | 0 | rc 0 |
| 追加 B. block の隣にない本文の数と主張を snapshot と code で照らす(029aa209 の追加を含む) | 19 | 18 | 1(T1) | 0 | - |

- 1 の scope: run-test.sh は `RALPH_VERIFY_SCOPE=changed` で run-verify.sh を test mode で呼ぶ。`detect-changed-languages.sh` は merge-base(a1ce47d8)からの差分の 28 ファイルをすべて docs に分類し(`docs/*` と `*.md` の case)、`scope=changed reason=docs_only docs_only=true languages=` を返した。そのため言語の pack は 1 つも選ばれず(`Language packs selected: none`)、Go のテストは走っていない。走ったのは root の `scripts/verify.local.sh` の test mode で、`tests/test-*.sh` の 41 本である。Go を走らせないので、`GOTOOLCHAIN=go1.25.8` の実行も要らない
- 1 の Skipped は前からある `tests/test-secret-scan-branch.sh` の 1 件で、git 2.49.0 なので 2.41 より古い本物の git の場合を飛ばしている。`tests/test-secret-scan.sh` は 123/0、`tests/test-pre-bash-guard.sh` は 2227/0。assertion の数は suite の要約の行(要約がない suite は `PASS:` の行)を足したもので、出力の形が suite ごとに違うので目安である。OK の印は 41 本すべてにあり、`FAIL:` の行は 0 件
- 2 の結果(block の番号は証拠の中の順、行は ```` ```sh ```` の行)は下の表のとおり。17 個とも rc 0 で、stderr は空だった。verify は 8e1d5c40 で同じ比べ方をしており、029aa209 のあとも 17 個とも一致している

| Block | 行 | 比べた先 | 行数 | 結果 |
| --- | --- | --- | --- | --- |
| 1 | 87 | label の表(72〜85 行) | 12 | 一致 |
| 2 | 167 | model の表(158〜165 行) | 6 | 一致 |
| 3 | 241 | 2 つの org の表(235〜239 行) | 3 | 一致 |
| 4 | 249 | 承認の行の `text`(253 行) | 3 | 一致 |
| 5 | 279 | merge の表(274〜277 行) | 2 | 一致 |
| 6 | 359 | 介入の表(348〜357 行) | 8 | 一致 |
| 7 | 373 | 承認待ちの表(368〜371 行) | 2 | 一致 |
| 8 | 399 | 受信箱の扱いの表(390〜397 行) | 6 | 一致 |
| 9 | 412 | 受信箱の event の `text`(417 行) | 4 | 一致 |
| 10 | 444 | 見張りの escalation の表(429〜442 行) | 12 | 一致 |
| 11 | 461 | shim の通知の表(452〜459 行) | 6 | 一致 |
| 12 | 479 | watch.log と receipt の `text`(484 行) | 9 | 一致 |
| 13 | 518 | 費用の表(501〜516 行) | 14 | 一致 |
| 14 | 568 | gate の表(559〜566 行) | 6 | 一致 |
| 15 | 602 | 見張りの止め直しの `text`(609 行) | 7 | 一致 |
| 16 | 657 | 止め直しの manifest の表(644〜655 行) | 10 | 一致 |
| 17 | 664 | doctor-escape の受信箱と承認の `text`(669 行) | 6 | 一致 |

- 2b は harness の感度の確かめである。変えた 2 か所は block 7 と block 9 の不一致として出て、ほかの 15 個は一致のまま、rc 1 だった
- 3a の 3 行は origin/main の `docs/tech-debt/README.md` でも同じ行番号、同じ中身で出る。この branch の register の変更(20 行足し 7 行消し)は、形の崩れた行を増やしていない
- 追加 A は 3 つとも一致した(report-escape 4d7f7b10b330、doctor-escape c55693d7f59c、分割計画 0ffd32661606)。snapshot を作るときの置き換えは、この 3 つの承認された中身を変えていない
- 追加 B で照らした 19 件: manifest の 91 行、受信箱の e1〜e6、escalation の 30 件、shim の 49 件、`director.jsonl` に `detached` がないこと、`director-wait.json` の 06:37:35Z〜07:02:35Z と `end` timeout、`watch-alive.json` の running、merge guarantee の 06:10:08Z(`checked_at`)、介入の 7 件、1h の cache write(opus 491,330、sonnet 382,876)、9370 秒 = 2 時間 36 分 10 秒、1302 秒 = 21 分 42 秒、e1 の承認から resolve までの 888 秒、見張りの停止から次の wait までの 43 秒、029aa209 の F14 の数(`sent` は 11 件ですべて改行を持ち、report-escape の 5 件はすべて報告にあり、doctor-escape の 6 件のうち報告にあるのは 4 件、残りは 05:00:57Z と 05:37:13Z)、doctor-escape の報告の event が 04:44:51Z で終わること、report-escape の報告が 95 行であること、F11 の「disband writes no report」(`Org.Disband`、`internal/org/verbs.go:2066` は報告を書かない)、Snapshot の表の report-escape の報告の終わり。最後の 1 件だけが合わない(T1)

## Coverage

- Statement: この branch は記録だけを変えたので、測る code がない。run-test.sh は Go を走らせず、shell には instrumented な coverage の道具がない
- Branch: 該当なし
- Function: 該当なし
- Notes: 記録について確かめた範囲は、表と `text` block のすべて(17 個の block)、block の隣にない本文の数と主張の 19 件、計画の digest、承認された 3 つの計画の digest、register の行の形、check-sync である。block の隣にない本文の数は、全部は照らしていない(G2)

## Failure analysis

| Test | Error | Root cause | Proposed fix |
| --- | --- | --- | --- |
| T1(MEDIUM): 追加 B、`docs/evidence/org-trial-2026-10-20.md:40` | Snapshot の表は「report-escape's timeline ends at 05:01:18Z ... so neither shows the gates, the PRs, or the disband」と書く。snapshot の `reports/org-manifest-report-escape-2026-10-20.md` の Event timeline の最後の行(73 行)は `2026-10-20T05:56:21Z` の `gate_passed` で、report-escape の報告は自分の org の唯一の gate を示している。manifest の report-escape の event は 05:01:18Z(`sent`)の次が 05:56:21Z(`gate_passed`)、05:56:46Z(implementer の `stopped`)なので、報告は 05:56:21Z から 05:56:46Z の間に書かれている。F11 の WIP commit(05:56:51Z)とも合う | 029aa209 で足した文。05:01:18Z は report-escape の報告の最後の `sent` の行で、Event timeline の最後の行ではない。おそらく `sent` の行だけを見て時刻を取った。未確認 | 40 行の文を「report-escape's timeline ends at 05:56:21Z (its gate_passed) and doctor-escape's at 04:44:51Z (before its first gate), so neither shows the seats' stops, the PRs, or the disband, and doctor-escape's shows none of its gates」の趣旨に直す。表、AC の判定、verdict の行は変わらない。証拠は `docs/reports/` の記録ではないので、直すと record-only の例外の外になり、パイプラインを `/self-review` から回し直すことになる |

run-test.sh、17 個の block、3a〜3c、追加 A で失敗したものはない。2b の 2 件はわざと出した不一致である。

## Regression checks

| Previously broken behavior | Status | Evidence |
| --- | --- | --- |
| verify(8e1d5c40)で一致した 17 個の block が、そのあとの 029aa209 の文の追加で崩れる | 崩れていない。029aa209 でも 17 個とも一致 | 2 |
| 計画の進み具合の印を付けると承認の digest が変わる | 変わらない。`5b82ee1e30a1` で `- Approved:` 行と同じ(印の行は digest の外) | 3b |
| register の行の形(1 行 5 欄)を壊す | 壊していない。基準と違う 3 行は origin/main と同じ | 3a |
| snapshot の置き換えが承認された計画の中身を変える | 変えていない。3 つとも digest が承認の行と同じ | 追加 A |
| verify の V1(doctor-escape の報告が disband の前に書かれたという書き方)を直した 029aa209 | doctor-escape の側と F14 の数は直っている。report-escape の側に新しい誤り(T1) | 追加 B |
| shell の suite 全体 | pass(41 本、FAIL 0) | 1 |

## Test gaps

- G1(INFO): 17 個の block が確かめるのは、証拠の表とコミットした snapshot が合っていることまでである。snapshot が試行の台帳(`/tmp/ot` は 07:18:53Z に消えた)と同じであることは、もう確かめられない。置き換えを通っても変わらないことを確かめられたのは、追加 A の 3 つの計画だけ
- G2(INFO): block の隣にない本文の数は、19 件を照らしただけで全部ではない。director の手元の記録だけが出どころの数(`--wait-checks` の 3m39s、load average、各コマンドの終了コード)は、コミットした出どころがないので照らせない。T1 はこの照らし合わせで見つかったので、ほかの本文の数にも同じ種類の誤りが残っている可能性はある。未確認
- G3(INFO): Go のテストは走らせていない。merge-base からの差分に Go のファイルはなく、changed scope は docs_only を選んだ。2 本の機能の PR(#241、#242)の Go の変更は、それぞれの gate と CI(証拠の Gates の表)で確かめられたもので、この回では走らせていない
- G4(INFO): 計画の Test plan の Edge cases(D6 の 2 つ、D8 の断られた場合)は試行の中で一度だけ起きたことで、走らせ直せない。この回は証拠のその表を snapshot から計算し直しただけ(block 5、15〜17)
- G5(INFO): block を走らせたのは jq 1.8.2、macOS の awk・sort・grep、bash 3.2 の `/bin/sh` だけで、GNU の道具では走らせていない。証拠の 18 行も同じ道具で試したと書いている

## Verdict

- Verdict: pass
- Pass: 1(run-test.sh、shell 41 本、FAIL 0、639 秒、rc 0)、2(17 個の block がすべて一致)、2b(感度の対照、狙いどおり 2 件の不一致)、3a(`22: 7`、`32: 5`、`278: 10` だけで、origin/main と同じ)、3b(`5b82ee1e30a1`)、3c(rc 0)、追加 A(3/3)、追加 B(19 件中 18 件)
- Fail: なし。テストの失敗はない。T1(MEDIUM)は証拠の 1 文の誤りで、表と判定は変わらないが、`/pr` の前に直すことを勧める
- Blocked: なし
