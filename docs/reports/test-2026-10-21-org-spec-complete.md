# Test report: org-spec-complete

- Date: 2026-10-21(JST 12:43〜13:45)
- Plan: `docs/plans/active/2026-10-21-org-spec-complete.md`(機能ごとの org の仕様に Status の節を足し、FR・NFR・受け入れ条件の印を根拠の表から付ける。docs だけの変更)
- Tester: tester subagent(Claude Opus 5.5)。パイプラインの 1 回目
- Scope: HEAD f16e90f8(base 765bcc46)。behavioral test だけを実行した(静的解析は /verify の担当)。コードは base と同じで、`git diff --stat 765bcc46 HEAD -- '*.go' go.mod go.sum scripts tests templates .claude .agents .codex` は空だった。環境は macOS 26.5.2(arm64)、`/bin/sh` は bash 3.2.57、git 2.49.0、go1.26.0 と `GOTOOLCHAIN=go1.25.8`。load average(1 分)は 1 と 2 の間が 100〜220(iOS simulator の起動とほかの session)、3 の開始時が 21、4 から 8 は 7〜14
- Evidence: `docs/evidence/test-2026-10-21-org-spec-complete.log`(下の 1〜8 の生の出力、照合に使った script と結果。`$HOME` は `~`、session の scratchpad は `<scratchpad>` に伏せた)と、run-test.sh が書いた `docs/evidence/verify-2026-10-21-034415.log`。どちらも `docs/evidence/*.log` の gitignore の対象で、コミットしない
- 追跡しているファイルは変えていない。`git status --porcelain` は最後まで 0 行で、`git status --porcelain --ignored` の一覧は 2 を始める前と 5 のあとで同じだった。1 と 2 は 12:55〜13:18 に重ねて走らせた。3 から 8 は 1 つずつ順に打った

## Test execution

| Suite / Command | Tests | Passed | Failed | Skipped | Duration |
| --- | --- | --- | --- | --- | --- |
| 1. `GOFLAGS=-timeout=30m ./scripts/run-test.sh`(changed。detector は `reason=docs_only` を返し、言語の pack は選ばれず Go は走らない) | shell 41 本 | 41 本すべて OK | 0 | 1(下) | 2039 秒、rc 0 |
| 2. `go test -count=1 -timeout 30m ./...`(go1.26.0) | 8 パッケージ | 7 | 1(`internal/cli` の 1 本。Failure analysis) | 0 | 1391 秒、rc 1。cli 1252.7 秒、org 1313.5 秒 |
| 3. 2 で落ちたテストを単独で `-count=5 -v` | 5 回 | 5 | 0 | 0 | 3.5 秒、rc 0。1 回 0.54〜0.72 秒 |
| 4. `GOTOOLCHAIN=go1.25.8 go test -count=1 -timeout 30m ./...` | 8 パッケージ | 8 | 0 | 0 | 771 秒、rc 0。cli 670.3 秒、org 763.8 秒 |
| 5. `go test -count=1 -timeout 30m ./internal/cli`(go1.26.0、単独) | 1 パッケージ | 1 | 0 | 0 | 463 秒、rc 0 |
| 6. 根拠の表のテスト名の照合(`git grep -n 'func <名前>(' -- '*.go'`) | 名前 66 個(表の 103 か所) | 66 個とも 1 件。パスと組の 100 か所はすべてそのパスにある | 0 | - | 数秒 |
| 7. PR の状態(`gh pr view <n> -R yoshpy-dev/ralph --json state,mergedAt`、読むだけ) | 28 本 | 28 本とも `MERGED` | 0 | - | 数秒 |
| 8. 表のテスト 66 本を `-run` で選び `-v` で走らせる(go1.26.0、`./internal/cli ./internal/config ./internal/org`) | 66 本と subtest 430 本 | 496 | 0 | 0 | 71 秒、rc 0。cli 70.4 秒、org 30.6 秒 |

- 1 の Skipped は前からある `tests/test-secret-scan-branch.sh` の 1 件で、git 2.49.0 なので 2.41 より古い本物の git の場合を飛ばしている。`tests/test-secret-scan.sh`(ほかの session と同時に走ると落ちることがある)も通った
- 1 は、仕様の checkbox を読むテストがないことの確認を兼ねる(計画の Integration tests)。`git grep -l org-multi-org-director -- '*_test.go' tests/ scripts/ .github/` は 0 件で、Go のコードで仕様の名前を出す 14 ファイルの参照は、どれもコメントの行だった。新しい `docs/evidence/org-spec-evidence-2026-10-21.md` は `tests/test-no-loop-references.sh` の除外(`docs/specs/`、`docs/tech-debt/README.md` など)の外にあり、この suite に走査されて PASS した
- 2 の `internal/org` 全体(load 100〜220)と 4 の `internal/org` 全体で、記録にある load に弱いテスト(試行の F9 の `TestRegate_FetchFailures/the_fetch_runs_past_the_bound`、`TestStopDirectorProcess_AProcessThatIgnoresSIGTERMIsNotEndedInTime`、`TestWakeDirector_AFailedWakeIsNotTriedAgainForFiveMinutes/no_return_in_time`)は落ちなかった
- 6 の照合は、表の各行で backtick の中の `_test.go` のパスを覚え、そのあとの `Test` で始まる名前と組にする perl の script で打った(evidence の 6 に本文)。組にならない 3 か所は試行の所見の表の F9 の行(2 か所)と G1 の行(1 か所)で、`TestRegate_FetchFailures`(`internal/org/regate_test.go:712`、subtest の名前 `the fetch runs past the bound` は 750 行)と `TestReportNames_DocsMatchConstants`(`internal/org/report_names_sync_test.go:95`)に 1 件ずつ見つかった。照合の感度は写しで確かめた。名前を 1 つ `…GONE` に変え、パスを 1 つ別のファイルに替えた写しでは、件数 0 と、パスの不一致の 2 か所(替えたパスに続く 2 つの名前)が出た
- verify の報告(AC6 の行)は表のテスト名を「103 か所(67 個)」と書く。6 では 103 か所は同じだったが、異なる名前は 66 個だった。backtick の中の `Test` で始まる文字列を subtest の部分ごと数えても、地の文の名前を数えても 66 個になる。67 になる数え方は見つからず、差の理由は分からない。名前がすべて見つかるという verify の結論は変わらない。verify の報告は、この報告のあとのコミットで「66 個」に直した
- 6 では、表がコードの根拠として挙げるテストでない名前も確かめた。`StopAll`(`internal/org/verbs_all.go:133`)、`activeEvents`(`internal/org/seat.go:134`。`spawn_failed` を含まない)、`TotalActiveSeats` と `RunningOrgs`(`internal/org/reserve.go`)、`SendWatchdogAlert` と `sendAlert`(`internal/org/watch.go`)、`callWithTimeout` と `truncateForDetails`(`internal/org/verbs.go`)は、表が書くファイルにある。tech-debt に足した行の「`callWithTimeout`(既定 10 秒)」は `defaultDriverCallTimeout = 10 * time.Second`(`internal/org/verbs.go:744`)と合い、「送れなくても deadman を登録する」は `sendAlert` の本文(送信の失敗を stderr に書いたあと `PendingAlerts` に入れる)と合う
- 7 の 28 本は、仕様の Status の節と根拠の表の末尾の一覧に出る PR 番号の和集合(どちらも同じ 28 本)である。verify の報告の結果と同じだった
- 8 の 66 本はすべて `--- PASS` で、`--- SKIP` は 0 件だった。表が根拠に挙げるテストは、この Mac で飛ばされずに走っている

## Coverage

- Statement: 測っていない。docs だけの変更で、コードの行は 1 行も変わっていない
- Branch: 同上
- Function: 同上
- Notes: この変更で確かめる対象は文書がコードを正しく指しているかどうかで、6(名前)、7(PR)、8(表のテストが走って通る)で確かめた。表のテストが項目の文のどこまでを見るか(根拠の強さ)は self-review と verify の担当で、/test では見ていない

## Failure analysis

| Test | Error | Root cause | Proposed fix |
| --- | --- | --- | --- |
| `TestRunDoctorOpts_CodexSandboxCheck_HermeticByDefault`(`internal/cli/doctor_codex_writable_root_test.go:1802`、2 で 1 回、10.34 秒) | `runDoctorOpts returned error: doctor: 1 check(s) failed`。doctor の出力は `✗ Claude Code CLI: fail — claude unusable: claude --version timed out after 5s` | `probeBinary`(`internal/cli/doctor.go:242`)は `<bin> --version` に 5 秒を与える。テストが PATH に置いた stub の `claude` が、load average 100〜220 の下でその 5 秒を超えた。この PR は Go に触れていない。単独では 5 回とも 0.5〜0.7 秒で通り(3)、4(go1.25.8)と 5(go1.26.0)の `internal/cli` 全体でも通った | この PR では直さない。`docs/tech-debt/README.md` の "Two doctor tests, `TestRunDoctorFull_PrintsTheMergeGuarantee` …" の行と同じ原因で、その行が挙げない 3 本目のテストに出た。その行の trigger は「2 つのテストのどちらかが同じ timeout で 2 度目に落ちる」で、このテストは名前に入っていないので、厳密には trigger に当たらない。後続で、行の対象を「stub を置いて doctor を走らせるテストのどれか」に広げ、このテストの名前と 2026-10-21 の失敗を足すことを提案する。このテストは doctor の出力を出すので、原因の check が `Claude Code CLI` であることは出力で確かめた |

## Regression checks

| Previously broken behavior | Status | Evidence |
| --- | --- | --- |
| 全体(shell 41 本、Go 8 パッケージを 2 つの toolchain で) | pass。go1.26.0 の 1 回目に `internal/cli` で load による失敗が 1 件あり、単独(5/5)とパッケージの打ち直し(rc 0)で通った。go1.25.8 は 1 回で全部通った | 1〜5 |
| main と同じ結果(計画の Regression tests) | pass。コードは 765bcc46 と同じで、Go のテストは変わった文書を読まない(上の grep)。結果の差は 2 の load による失敗だけで、コードの差によるものではない | 2〜5 |
| 試行の F9 などの load に弱いテスト | 2 と 4 で落ちていない | 2、4 |
| 仕様の checkbox を読むテストがない(計画の Integration tests) | pass | 1、grep |

計画の Edge cases:

- 根拠の表のテスト名がすべて今のコードにある: 6 で 66 個とも `func <名前>(` が 1 件、パスも一致。8 で 66 本とも走って PASS
- PR 番号がすべて MERGED: 7 で 28 本とも `MERGED`

## Test gaps

- T1(INFO): 根拠の表のテスト名の照合は suite に入っていない。表は `docs/evidence/` の日付つきの記録なので、改名で古くなっても日付の時点の記録として読める。ただし計画のとおり系列の 2 本目以降が表を更新するので、そのたびに同じ照合を手で打つ必要がある。6 の script は evidence にあり、そのまま使える
- T2(INFO): Failure analysis の doctor の flake は、この PR の範囲(docs だけ、計画の Non-goals はコードの変更を外す)の外である。tech-debt の行を広げる提案だけを残す
- T3(INFO): Linux では走らせていない。CI の verify(`.github/workflows/verify.yml`、`ubuntu-latest`)は `RALPH_VERIFY_SCOPE: full` で run-verify.sh を走らせるので、docs だけの変更でも Go のテストを含めて走る

## Verdict

- Verdict: pass
- Pass: 1(run-test.sh。shell 41 本、2039 秒、rc 0)、3(落ちたテストを単独で 5/5)、4(Go 8 パッケージ、go1.25.8、rc 0)、5(`internal/cli`、go1.26.0、rc 0)、6(名前 66 個とパス 100 か所が一致、感度の写しで不一致を検出)、7(28 本 MERGED)、8(表のテスト 66 本と subtest 430 本が PASS、SKIP 0)
- Fail: なし。2 の rc 1 は load による失敗で、3 と 5 で通った(Failure analysis)
- Blocked: なし
- 残る穴: T1〜T3(どれも INFO)。マージを止めるものはない
