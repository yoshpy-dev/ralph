# Test report: review-switches-skills

- Date: 2026-10-29(JST 08:13〜08:45、UTC では 2026-10-28 23:13〜23:45)
- Plan: `docs/plans/active/2026-10-29-review-switches-skills.md`(AC4、AC5、AC6、AC8 と Test plan)
- Tester: tester サブエージェント(Claude Opus 5.5)。pipeline の cycle 1
- Scope: HEAD 1ebd7bf2(base 9837f744)。behavioral test だけを実行した(静的解析と同期の検査は /verify の担当)。環境は macOS(Darwin 25.5.0、arm64)、go1.27.3、git 2.50.1
- Evidence: `docs/evidence/test-2026-10-29-review-switches-skills.log`(1: `run-test.sh`、2: 名指しの `-v` 実行、3: mutant と対照、4a: driver と reviewer の snippet、4b: `/cross-review` 手順 4 の base の行、4c: 飛ばした triage の report、5: coverage)。gitignore の対象で、コミットしない
- 追跡しているファイルは、この報告と insight event のほかは変えていない。mutant は `go test -overlay` で scratchpad の写しに置き換えた。ルートの `ralph.toml` と skill を読むテストの mutant では、`repoRoot` が環境変数 `RALPH_TEST_ROOT` を読むようにした `defaults_sync_test.go` の写しを overlay し、`git archive HEAD` の写しを変えたものを読ませた

## Test execution

| # | Suite / Command | Tests | Passed | Failed | Skipped | Duration |
| --- | --- | --- | --- | --- | --- | --- |
| 1 | `./scripts/run-test.sh` | shell 45 本 + Go 9 パッケージ | shell 45 本、Go 9 パッケージ | 0 | 0(shell の既存の SKIP 行が 1 つ、下) | rc 0、1,146 秒(08:14:27〜08:33:33 JST) |
| 2 | `go test -count=1 -v -run 'TestParseReport_CrossReview' ./internal/insights/` と `-run 'TestRootRalphToml_KnownKeysAndReviewSwitchesOn\|TestDefaultsLockStep' ./internal/config/` | 7 と 2(サブテスト 3) | 7 と 2(3) | 0 | 0 | rc 0、1 秒未満 |
| 3 | mutant 7 個と対照 3 つ(下の「mutant」) | 10 実行 | 対照 3 つは通り、mutant 6 個が落ちた | 生き残り 1 個(a3、base でも生き残る) | - | 1 回 1 秒前後 |
| 4a | `/plan` 11.b と `/cross-review` 2.b の snippet を、新しい bash と zsh で driver 2 通り × `RALPH_PRIMARY_CLI` 7 通り | 28 | 28 | 0 | - | - |
| 4b | `/cross-review` 手順 4 の base の行を bash と zsh で 4 つの状況 | 8 | 8 | 0 | - | - |
| 4c | 飛ばした triage の report(scratch)を `count_triage_findings` と `ralph insights backfill` で読む | 15 + dry-run 2 回 | すべて期待どおり | 0 | - | - |

- 1 の scope は `Requested scope: changed` から `Language scope: full fallback (unclassified:.codex/config.toml)` に落ち、shell と golang の両方が走った(lead の想定どおり)
- 1 の shell: `==> tests/` の 45 本(`tests/test-*.sh` の数と同じ)がすべて `OK` で終わった。`PASS` で始まる行は 4,796 行で、0 でない `FAIL:` の要約行はない。SKIP の行は `the real git older than 2.41 case (git 2.50.1 is 2.41 or later; the stubbed old git covers it)` の 1 つで、この branch と関係のない既存の行である
- 1 の Go: `cmd/ralph` 1.955 秒、`internal/cli` 368.306 秒、`internal/config` 0.910 秒、`internal/insights` 2.372 秒、`internal/org` 476.412 秒、`internal/org/driver` 4.329 秒、`internal/org/protocol` 4.666 秒、`internal/scaffold` 3.820 秒、`internal/upgrade` 3.834 秒。`(cached)` はない
- 30 分の timeout も、既知の不安定なテスト(`TestStopDirectorProcess_AProcessThatIgnoresSIGTERMIsNotEndedInTime`、`TestOrgSpawn_HelloPastTimeoutMS_PrintsTheDeadlineHint`)の失敗も起きなかったので、打ち直しはしていない。1 の間の 1 分の load average は 3.6(始まり)、9.0(08:19)、5.2(08:29)、4.6(終わり)。2〜4 の多くは 1 と並べて走らせた
- 最初の 1 回は `( … ) &` で背景に回したため、SIGINT を無視する状態で始まった(memory の注意どおり、check-template の信号のテストが SKIP になりうる)。開始 30 秒ほどで process group ごと止め、Bash tool の `run_in_background` で始め直した。上の 1 はその 2 回目の結果である

## lead が名指ししたテスト

| テスト | 場所 | 結果 |
| --- | --- | --- |
| `TestParseReport_CrossReview_SkippedReviewerIsNA` | insights(2 の `-v`) | PASS |
| `TestParseReport_CrossReview_OnlyLaterCycleSkipped` | insights | PASS |
| 既存の backfill のテスト: `…_IncompleteReviewerIsNA`、`…_OnlyLaterCycleIncomplete`、`…_CompleteReviewerZeroCountsIsPass`、`…_ZeroCounts`、`…_MultiCycle` | insights | PASS(5 本とも) |
| `TestRootRalphToml_KnownKeysAndReviewSwitchesOn` | config | PASS |
| `TestDefaultsLockStep`(`org`、`standard`、`shell_and_skills`。`shell_and_skills` は plan skill の `RALPH_CLAUDE_REVIEWER_MODEL` の照合を含む) | config | PASS(3 つとも) |
| `tests/test-xreview-helpers.sh` | 1 の中 | PASS 29、FAIL 0 |
| `tests/test-codex-exec-invocation.sh` | 1 の中 | 144/144 |
| `tests/test-check-skill-sync.sh` | 1 の中 | PASS 13、FAIL 0 |
| `tests/test-slice-verification.sh` | 1 の中 | 162/162 |
| `tests/test-pr-owner-lookup.sh` | 1 の中 | 20 passed、0 failed |
| `tests/test-skill-insight-cycle.sh` | 1 の中 | 20 passed、0 failed |

## AC ごとの確認

| AC | テストと mutant | 結果 |
| --- | --- | --- |
| AC4 | `TestParseReport_CrossReview_SkippedReviewerIsNA` と `…_OnlyLaterCycleSkipped`。`skipped` の判定を消す mutant a1 で 2 本とも落ちる。4c で、HEAD の binary は飛ばした report 6 つ(理由 5 通りと cycle 2 だけ飛ばした report)の飛ばした cycle をすべて `n/a` と読み、base 9837f744 の binary は同じ report を `pass` と読んだ | 満たす |
| AC5 | `TestRootRalphToml_KnownKeysAndReviewSwitchesOn`。写しで `[standard]` の `cross_review` を `cross_reviw` に改名すると、`standard.cross_reviw (line 11)` とキーと行を名指しして落ちる。`false` にした写し、ファイルを消した写しでも落ちる。変えていない写しの対照は通る | 満たす |
| AC6 | `TestDefaultsLockStep/shell_and_skills` の `{"plan", "RALPH_CLAUDE_REVIEWER_MODEL"}` の行。写しで plan skill の fallback を `sonnet` にすると `plan/SKILL.md RALPH_CLAUDE_REVIEWER_MODEL fallback = "sonnet", want "opus"` で落ちる。同じ写しでその行を表から消すと通るので、落としたのはこの行である | 満たす |
| AC8 | テストの部分: `go test ./internal/config/ ./internal/insights/` は 1 と 2 で通った。skill の文を固定する shell のテスト(`test-check-skill-sync.sh`、`test-slice-verification.sh`、`test-codex-exec-invocation.sh`、`test-pr-owner-lookup.sh`、`test-skill-insight-cycle.sh`、`test-xreview-helpers.sh`)は 1 で通った。`check-skill-sync.sh` と `check-sync.sh` は静的な検査なので走らせていない(verify の報告 `docs/reports/verify-2026-10-29-review-switches-skills.md` が f0290dc4 の時点で rc 0 を記録し、その後のコミット 1ebd7bf2 は計画の記録だけを変えた)。1 の中では、2 つのスクリプト自体のテスト(`tests/test-check-sync.sh`、`tests/test-check-skill-sync.sh`)が通った | テストの部分は満たす |

## mutant

| id | 変更 | 結果 | 落としたテスト |
| --- | --- | --- | --- |
| a1 | backfill の `\|\| strings.HasPrefix(status, "skipped")` を消す | 落ちる | `…_SkippedReviewerIsNA`、`…_OnlyLaterCycleSkipped` |
| a2 | `status := strings.ToLower(m[1])` を `status := m[1]` にする | 落ちる | `…_OnlyLaterCycleIncomplete`、`…_OnlyLaterCycleSkipped`(どちらも先頭が大文字の status を使う) |
| a3 | `After triage:` の行を読んだあとの `pendingNoReview = false` を消す | 生き残る | - |
| b1 | ルートの `ralph.toml` の写しで `[standard]` の `cross_review` を `cross_reviw` にする | 落ちる | `TestRootRalphToml_…`(`has keys no Config field reads: standard.cross_reviw (line 11)`) |
| b2 | 写しで `[org.review]` の `cross_review` を `false` にする | 落ちる | `TestRootRalphToml_…`(`org.review.cross_review is not true`) |
| b3 | 写しから `ralph.toml` を消す | 落ちる | `TestRootRalphToml_…`(`read the repository's ralph.toml: open …: no such file or directory`) |
| c | 写しの `.claude/skills/plan/SKILL.md` で `${RALPH_CLAUDE_REVIEWER_MODEL:-opus}` を `:-sonnet` にする(1 か所) | 落ちる | `TestDefaultsLockStep/shell_and_skills` だけ |

- 置き換えは perl で 1 回だけ現れる字面を変えた(0 回か 2 回以上なら止める)。build の失敗は 0 件
- 走らせた範囲: a1〜a3 は `./internal/insights/` 全体。b1〜c は `./internal/config/` の `-run 'TestRootRalphToml_KnownKeysAndReviewSwitchesOn|TestDefaultsLockStep'`(この 2 本だけがルートの `ralph.toml` と plan skill を読む)
- 対照: (1) 変えていない `git archive HEAD` の写しを `RALPH_TEST_ROOT` で読ませると 2 本とも通る(overlay が写しを読むことの確かめ。b1〜c が落ちたことと合わせて、写しを読んでいる)。(2) c の写しのまま、表から `{"plan", "RALPH_CLAUDE_REVIEWER_MODEL"}` の行を消した overlay では通る。(3) a3 と同じ変更を base 9837f744 の `pendingIncomplete = false` に入れても、base のテストは通る
- a3 が生き残る理由: cycle 1 が飛ばした(か未完了の)report で、cycle 2 に `Reviewer status:` の行がない形のテストがない。今あるテストは、cycle 2 にも `Reviewer status:` の行があり、その行が値を上書きする。base でも同じく生き残るので、この branch で入った穴ではない。この branch で `skipped` もこの道を通るようになった

## Skill の手順の確かめ(モデルを呼ばない)

### 4a driver と reviewer の snippet

`/plan` 11.b と `/cross-review` 2.b の snippet は、8 つの写し(`.claude`、`.agents`、`templates/base` の 2 つ × 2 skill)で同じ 1 行だった(sha256 の先頭 `f11981820e577892`)。`<claude|codex>` を置き換え、`bash --noprofile --norc -c` と `zsh -f -c` で worktree の root から走らせた。期待値は snippet と別に計算した(`RALPH_PRIMARY_CLI` を小文字にして `claude` か `codex` ならそれ、ほかは driver のまま。reviewer はその反対)。

| driver | `RALPH_PRIMARY_CLI` | 出力(bash と zsh で同じ) |
| --- | --- | --- |
| claude | 未設定 / `gemini` / `claude` / `Claude` / 空 | `driver=claude reviewer=codex` |
| claude | `codex` / `CODEX` | `driver=codex reviewer=claude` |
| codex | 未設定 / `codex` / `CODEX` / `gemini` / 空 | `driver=codex reviewer=claude` |
| codex | `claude` / `Claude` | `driver=claude reviewer=codex` |

28 行とも期待どおり。比べ方が誤りを拾うことを、2 つの mutant の snippet で確かめた: 小文字にする `tr` を外すと `RALPH_PRIMARY_CLI=CODEX` で `driver=claude reviewer=codex` になり、`pick_reviewer` を引数なしで呼ぶと driver が codex で未設定のとき `reviewer=codex` になる(どちらも bash と zsh で期待と違った)。

### 4b `/cross-review` 手順 4 の base の行

`. scripts/xreview-helpers.sh; BASE=$(detect_base_branch); git diff "$BASE"...HEAD --quiet; echo "base=$BASE rc=$?"`(4 つの写しで同じ字面)を bash と zsh で走らせた。

| 状況 | 出力(bash と zsh で同じ) |
| --- | --- |
| この worktree | `base=main rc=1`(差分あり、review に進む) |
| `main` と同じ内容の branch だけの scratch の repo | `base=main rc=0`(`skipped (empty diff)` の道) |
| worktree で `RALPH_XREVIEW_BASE=HEAD` | `base=HEAD rc=0` |
| worktree で `RALPH_XREVIEW_BASE=no-such-ref` | `base=no-such-ref rc=128`(0 でないので review に進む。helper の注記どおり fail-open) |

`$?` は `git diff` の終了コードを表し、base は 1 回の呼び出しの中で決まって表示された。

### 4c 飛ばした triage の report

`docs/reports/templates/cross-review-triage-report.md` から、skill の手順 6 のとおり `Reviewer status: skipped (<理由>)`、`Total reviewer findings: 0`、`After triage: ACTION_REQUIRED=0, WORTH_CONSIDERING=0, DISMISSED=0`、`Cycle: 1/2` を書いた report を scratch に 5 つ作った(理由は `off: [standard].cross_review`、`ralph without config get; treated as off`、`ralph not found; treated as off`、`codex not available`、`empty diff`)。もう 1 つ、cycle 1 が `complete` で 1/1/1、`## Cycle 2` が `skipped` で 0/0/0 の report も作った。

- `count_triage_findings` は 5 つの report すべてで 3 つの区分とも 0 を返した(`sh`、bash、zsh の 15 回)
- `ralph insights backfill --reports-dir <scratch>/reports --events-dir <scratch>/events`(dry-run): HEAD で build した binary は飛ばした 6 つの cycle をすべて `verdict=n/a`、cycle 1 を `action_required` と読んだ。parse miss は 0。base 9837f744 で build した binary は飛ばした 6 つを `verdict=pass` と読んだ。events のディレクトリには何も書かれていない

## Test plan の端のケース

| 端のケース | テスト |
| --- | --- |
| ルートの `ralph.toml` に知らないキーがある写し | mutant b1(落ちる、キーと行を名指しする) |
| `Reviewer status: skipped` の report の cycle 2 | `TestParseReport_CrossReview_OnlyLaterCycleSkipped`、4c の 2 cycle の report |
| backfill の `incomplete` と `complete` が変わらない(回帰) | `…_IncompleteReviewerIsNA`、`…_OnlyLaterCycleIncomplete`、`…_CompleteReviewerZeroCountsIsPass` が通る。mutant a2 で `…_OnlyLaterCycleIncomplete` が落ちる |

## Coverage

- Statement: `internal/insights` 86.5%(`ParseReport` 77.8%、`parseCrossReviewAllCycles` 92.5%)、`internal/config` 97.0%。2 パッケージの合計 89.0%
- Branch: 測っていない
- Function: 上の行のとおり
- Notes: `parseCrossReviewAllCycles` で変わった行(`status` の小文字化と 2 つの接頭辞、`noReview` の受け渡し)は mutant a1 と a2 が落ちることで通ることを確かめた。shell は計測の道具がないので、範囲は上のテストの行で示した

## Failure analysis

| Test | Error | Root cause | Proposed fix |
| --- | --- | --- | --- |
| なし | - | - | - |

## Regression checks

| Previously broken behavior | Status | Evidence |
| --- | --- | --- |
| 未完了の cycle を backfill が `n/a` と読む(#265 より前からの動き) | 変わらない | `…_IncompleteReviewerIsNA`、`…_OnlyLaterCycleIncomplete` が通る |
| `Reviewer status: complete` で 0/0/0 の cycle は `pass` | 変わらない | `…_CompleteReviewerZeroCountsIsPass` が通る |
| driver の上書きが大文字でも効く(AR-2) | 変わらない | 4a の `CODEX` と `Claude` の行 |
| 空の `$BASE` で「差分なし」と誤る(F1 で直した) | 直っている | 4b: base は同じ呼び出しで決まり、`base=main` と表示される |

## Test gaps

- mutant a3 の生き残り(LOW、base からある): cycle 1 が `skipped` か `incomplete` で、cycle 2 に `Reviewer status:` の行がない report のテストがない。足すなら「cycle 1 は skipped、cycle 2 は status の行なしで 0/1/0 → cycle 2 は pass」の 1 本で a3 が落ちる
- skill の読み方の約束(`ralph config get` の終了コードの分け方)、`/pr` の Verification の行、`/cross-review` の skip の道は文で書かれた手順で、決まった結果を出すテストはない。終了コードは /verify が binary で実測した(PATH の 6.0.0 で rc 1、build した binary で `true`、知らないキー 2、型の誤り 3、ralph なし 127)。この /test では 4a〜4c の部品を確かめたが、skill を通しで動かしてはいない
- 相手のモデルを実際に呼ぶ道(`claude -p` の flag の組、codex の呼び出し)は、モデルを呼ぶので走らせていない。claude の形は S2 の implementer が haiku で 1 回確かめた(計画の Progress)
- driver の上書きの処理が skill の本文の 8 か所にある(計画の N2、PR の Known gaps)。4a は今の 1 行を 1 回確かめただけで、常設のテストではない
- 範囲外の観察: `count_triage_findings` は最初の `After triage:` の行を読む(`grep -m1`)。cycle 2 の節に別の `After triage:` の行を足す形で report を書くと、手順 6 の件数の確かめは cycle 1 の数と比べることになる。今ある複数 cycle の report は cycle 2 の節に `After triage:` の行を持たないので、この branch の前からの扱いである

## Verdict

- Pass: 1(`./scripts/run-test.sh` が rc 0、shell 45 本、Go 9 パッケージ)、2(名指しのテストすべて、SKIP 0)、3(mutant 7 個のうち 6 個が落ちた。生き残りの a3 は base からある)、4a〜4c。AC4、AC5、AC6 と AC8 のテストの部分を満たす。/pr に進んでよい
- Fail: 0
- Blocked: なし
- Verdict: pass
