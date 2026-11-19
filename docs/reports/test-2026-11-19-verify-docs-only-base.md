# Test report: verify-docs-only-base

- Date: 2026-11-19。cycle 1(`.harness/state/standard-pipeline/cycle-count.json` は `cycle: 1`)
- Plan: `docs/plans/active/2026-11-19-verify-docs-only-base.md`(issue #341)
- Tester: tester サブエージェント(`/test`)
- Scope: HEAD `ddc1f0d6`(base `a8c5e289`、worktree は clean)。`./scripts/run-test.sh` を 1 回、worktree のルートから。scope は wrapper の既定の changed だが、変更に共有の script があるので full に落ちた(`==> Language scope: full fallback (shared:scripts/detect-changed-languages.sh)`)。計画どおり。local verifier は scope によらず `tests/test-*.sh` を全部回す。そのあと `GITHUB_BASE_REF=main` を export して、指定の shell の 6 本と Go の 2 つの絞った run を打ち直した
- Evidence: `docs/evidence/test-2026-11-19-verify-docs-only-base.log`(5,828 行。`.gitignore` の `docs/evidence/*.log` に当たるので commit されない)。run-verify.sh は自分の evidence を `docs/evidence/verify-2026-11-19-132952.log` にも書いた(同じく追跡外)

## Test execution

| Suite / Command | Tests | Passed | Failed | Skipped | Duration |
| --- | --- | --- | --- | --- | --- |
| `./scripts/run-test.sh` | 下の 2 行の合計 | 全部 | 0 | 2(前からある条件つきの skip) | 2026-11-19 22:29:51 〜 22:59:05 JST、1,754 s、rc 0 |
| local verifier(`tests/test-*.sh` 46 本) | 46 本、各 suite の集計行の合計で 5,361 件 | 46 本(`OK` 46 行) | 0 | `test-plan-visual.sh` の browser detection(`/Applications` にブラウザーがある)、`test-secret-scan-branch.sh` の「git 2.41 より古い本物の git」の場合(手元は git 2.49.0) | 約 13 分 |
| `go test ./...`(テストのある 11 パッケージ) | 11 | 11 | 0 | なし。`(cached)` のパッケージもない | 下の表 |

変更が触った shell の suite の件数(full run の中):

| Suite | 結果 |
| --- | --- |
| `tests/test-detect-changed-languages.sh` | 236 / 236 |
| `tests/test-run-verify-docs-only.sh`(新規) | 66 / 66 |
| `tests/test-run-verify-scope.sh` | 25 / 25 |
| `tests/test-run-verify-concurrency.sh` | 37 / 37 |
| `tests/test-run-verify-branch-secret-scan.sh` | 55、FAIL 0 |
| `tests/test-ralph-dispatch.sh` | 33、FAIL 0 |

| パッケージ | 結果 | 時間 |
| --- | --- | --- |
| `cmd/ralph` | ok | 1.011 s |
| `internal/cli` | ok | 724.197 s |
| `internal/config` | ok | 19.095 s |
| `internal/gitroot` | ok | 5.027 s |
| `internal/insights` | ok | 1.577 s |
| `internal/org` | ok | 853.533 s |
| `internal/org/driver` | ok | 4.978 s |
| `internal/org/protocol` | ok | 3.294 s |
| `internal/scaffold` | ok | 3.701 s |
| `internal/toolchain` | ok | 27.979 s |
| `internal/upgrade` | ok | 5.256 s |

実行の条件:

- 負荷(`sysctl -n vm.loadavg`): 始まり `31.49 32.84 30.17`、終わり `13.24 19.26 26.52`。shell の段の途中で 1 分値が 93.16 まで上がった(ほかのセッション)。Go の段の読みは 12〜33。Go の段の間、ほかのセッションの `go test ./internal/org/ -run 'Approv|Inbox|Resolve|Send|Manifest'` が並んで動いていた。自分の run は Go の段に重ねていない
- `GOFLAGS` は足していない。golang の pack が `-timeout=30m` を入れる。`internal/org` の 853.5 s は上限の 47%、`internal/cli` の 724.2 s は 40%
- 起動: `run_in_background` は使わず、perl の fork と `setsid` で runner(`$TMPDIR` の下の scratch)を切り離した。INT、HUP、QUIT、TERM は DEFAULT に戻し、標準入出力は `/dev/null`。runner は始まりと終わりの時刻、負荷、rc を meta file に書く。待つのは `caffeinate -w <pid>` を 175 s ずつ区切って繰り返した。shell の SKIP は上の 2 件だけで、signal を使う suite は skip していない
- 走らせる前に `pgrep` で、この machine に別の `run-test.sh` も `run-verify.sh` もないことを確かめた。full run は 1 回だけ

落ちたテストがないので、full run の打ち直しはしていない。

## `GITHUB_BASE_REF=main` での打ち直し

CI は `GITHUB_BASE_REF=main` を本当に設定する。その値を export した shell で、full run のあとに次を順に走らせた(worktree のルートから、ほかの run とは重ねていない)。

| Command | 結果 | 時間 |
| --- | --- | --- |
| `sh tests/test-detect-changed-languages.sh` | rc 0、236 / 236 | 51 s |
| `sh tests/test-run-verify-docs-only.sh` | rc 0、66 / 66 | 24 s |
| `sh tests/test-run-verify-scope.sh` | rc 0、25 / 25 | 13 s |
| `sh tests/test-run-verify-concurrency.sh` | rc 0、37 / 37 | 17 s |
| `sh tests/test-run-verify-branch-secret-scan.sh` | rc 0、55、FAIL 0 | 39 s |
| `sh tests/test-ralph-dispatch.sh` | rc 0、33、FAIL 0 | 15 s |
| `go test ./internal/org/ -run 'TestGateScriptEnv\|TestGate_RealRunVerify\|TestGate_ScriptEnvironmentIsFixed' -count=1 -v` | ok(10.280 s)。`TestGate_RealRunVerify_FullScopeRunsTheLanguageVerifier`、`TestGate_ScriptEnvironmentIsFixed`、`TestGateScriptEnv` の 3 つが PASS | 11 s |
| `go test ./internal/cli/ -run OrgGate -count=1 -v` | ok(22.960 s)。top-level 20(`go test -list OrgGate` も 20)、subtest 39 が PASS、FAIL と SKIP は 0 | 24 s |

`tests/test-ralph-dispatch.sh` の shebang は bash で、full run は shebang で走らせた。指示どおり `sh` でも走らせ、macOS の `/bin/sh`(POSIX モードの bash)で同じ 33 件が通った。

指定の 6 本のほかに、scripts の名前を出す shell の suite も同じ env で走らせた。`test-agent-phase-boundaries.sh` 44 / 44、`test-check-template.sh` 60 passed(bash の shebang なので `./` で)、`test-post-edit-verify.sh` 24、`test-secret-scan.sh` 123、`test-self-review-scope.sh` 206 / 206、`test-slice-verification.sh` 162 / 162、`test-secret-scan-branch.sh` 263。どれも FAIL 0。打ち直しのあとも `git status --porcelain` は空だった。

### unset の行が効いているかの確かめ

打ち直しが通るだけでは、`GITHUB_BASE_REF` がもともと結果に効かない suite と区別できない。そこで `git archive HEAD` の写し(`$TMPDIR` の下)で、各 suite の `unset` の行から `GITHUB_BASE_REF` だけを抜き、env に値を入れて走らせた。対照(抜かない写し)はどれも全件通った。

| Suite(抜いた行) | `GITHUB_BASE_REF=main` | `GITHUB_BASE_REF=nosuch` |
| --- | --- | --- |
| `test-detect-changed-languages.sh:129` | FAIL 49 / 236 | 打っていない |
| `test-run-verify-docs-only.sh:31` | FAIL 12 / 66 | 打っていない |
| `test-run-verify-scope.sh:9` | FAIL 5 / 25 | 打っていない |
| `test-run-verify-concurrency.sh:34` | 全件通る | FAIL 5 / 37 |
| `test-run-verify-branch-secret-scan.sh:127` | 全件通る | FAIL 25 / 55 |
| `test-ralph-dispatch.sh:47` | 全件通る | 全件通る |
| Go: `gateRalphStubEnv` の `name != "GITHUB_BASE_REF"`(`internal/org/gate_runverify_test.go:89`、`-overlay` で抜いた) | `TestGate_RealRunVerify_FullScopeRunsTheLanguageVerifier` が FAIL | 同じく FAIL |

CI の値 `main` では、unset がなければ detector、docs-only、scope の 3 本と Go のテストが落ちる。concurrency と branch-secret-scan は fixture に `main` があるので `main` では差が出ないが、別の名前の base では落ちる。dispatch の Case H は drop-in の verifier を走らせるので `docs_only` を読まず、どちらの値でも結果は同じになる(この unset は守りのためで、結果を分けない)。

## 計画のテストと AC の対応

full run と打ち直しがどちらも通ったので、次のテストは通っている。

| AC / Test plan の項 | テスト | 結果 |
| --- | --- | --- |
| AC1(CI 風の checkout でコードの commit は exit 2、path を名指し、docs only と言わない) | `test-run-verify-docs-only.sh` の `AC1:` の節(5 件)と fixture の節(2 件) | pass |
| AC2(docs だけは exit 0) | 同 `AC2:`(4 件) | pass |
| AC3(scaffold だけは exit 0) | 同 `AC3:`(2 件)、detector の case 43〜47 | pass |
| AC4(手元の clone が full と changed で同じ exit code) | 同 `AC4:`(7 件) | pass |
| AC5(ref がない、merge-base がない、base がないとき exit 2 と理由) | 同 `AC5:`(12 件) | pass |
| AC6(mode ごとの verifier の足し方) | 同 `AC6:`(10 件) | pass |
| AC7(3 値の `docs_only`、`first_non_docs`、既存の case の scope、reason、languages、roots は同じ) | detector の case 1〜42 と 48、49 | pass |
| AC8(`GITHUB_BASE_REF` の読み方、ralph に聞かない) | detector の case 50、51 | pass |
| AC9(写しの一致、`check-sync.sh`) | `test-check-sync.sh` 16 / 16(script そのものは verify の担当) | pass |
| AC10(テストが `GITHUB_BASE_REF` を unset する) | 上の打ち直しと、unset を抜いた写しの確かめ | pass |
| AC11(4 つの listing のどれかが失敗すると `diff_failed`、両 scope で exit 2) | detector の case 52(4 つを 1 つずつ)、`test-run-verify-docs-only.sh` の `AC11:`(6 件、`--cached` の失敗を両 scope で) | pass |
| AC12(scaffold の削除と manifest の行の削除で exit 0) | `test-run-verify-docs-only.sh` の `AC12:`(6 件)、detector の case 53 | pass |
| S2c(gate が呼び手の `GITHUB_BASE_REF` を落とす) | `TestGateScriptEnv`(`GITHUB_BASE_REFX` は残す)、`TestGate_RealRunVerify_FullScopeRunsTheLanguageVerifier`、`internal/cli` の OrgGate の help のテスト | pass |
| Test plan の Edge cases: 空の `GITHUB_BASE_REF` は unset と同じ | detector の case 50 の最後 | pass |
| Test plan の Edge cases: 消した scaffold の file | AC12 の行 | pass |
| Test plan の Edge cases: untracked のコードだけ | detector の case 1(add していない `main.go` だけで `docs_only=false`、`first_non_docs=main.go`)、`test-run-verify-concurrency.sh` の Case 5(unstaged の `app.txt` で exit 2) | pass |
| Test plan の Edge cases: scaffold の path からコードの path への rename | テストがない(verify の probe だけ) | gap(G1) |

## Coverage

- Statement: shell の script には計測の道具がない。テストの範囲で測った(上の表)。Go の変更は `internal/org/gate_run.go` の `gateScriptEnv` と comment、`internal/org/gate.go` の comment、`internal/cli/org_gate.go` の help の文。`gateScriptEnv` は上の絞った run(`-coverprofile`)で 100.0%
- Branch: 測っていない
- Function: `gateScriptEnv` 100.0%、`gateEnvValue` 100.0%(同じ絞った run)
- Notes: パッケージ全体の coverage は取っていない(full run は `-cover` なし)

## Failure analysis

| Test | Error | Root cause | Proposed fix |
| --- | --- | --- | --- |
| なし | | | |

打ち直しの途中で `sh tests/test-check-template.sh` が rc 2(`syntax error near unexpected token '('`)で止まった。この suite は bash の shebang で process substitution を使うので、`sh` で呼んだ自分の誤り。`./tests/test-check-template.sh` で 60 passed。テストの失敗ではない。

## Regression checks

| Previously broken behavior | Status | Evidence |
| --- | --- | --- |
| #341: pack のない repository で、CI の full scope がコードの PR も docs only と判定して exit 0 | 直っている | `test-run-verify-docs-only.sh` の `AC1:`(CI 風の checkout で exit 2)。unset を抜いた写しで、CI の値でも結果が変わることを見た |
| 読めなかった変更の一覧が `no_changes` や docs only になる(R8) | 直っている | detector の case 52、`AC11:` の節 |
| `ralph upgrade` が消す core の file をコードと読む(R9) | 起きない | `AC12:` の節、detector の case 53 |
| gate の呼び手の `GITHUB_BASE_REF` が scripts の base を動かす(B9) | 起きない | `TestGateScriptEnv`、`TestGate_RealRunVerify_FullScopeRunsTheLanguageVerifier`(呼び手に `GITHUB_BASE_REF=feat/auth`) |
| 既存の detector の case の scope、reason、languages、roots | 変わらない | case 1〜42 の 236 件の中 |

## Test gaps

- G1: scaffold の path をコードの path へ rename する commit のテストがない。verify の probe では `docs_only=false`、`first_non_docs=scripts/new-tool.sh` だった(git の rename 検出で新しい path だけが出る)。足すなら detector の case に `git mv` の 1 件
- G2: run-verify.sh 側の AC11 は `--cached` の失敗だけを打つ。残る 3 つは detector の case 52 と、`unknown` を reason によらず 1 つの枝で扱うコードで見ている
- G3: `test-ralph-dispatch.sh:47` の unset は結果を分けない(上の確かめ)。Case H が verifier を走らせる間は守りの行のまま
- G4: 本物の GitHub Actions の checkout はこの run では見ていない。PR の CI が最初の本物の確かめになる
- G5: パッケージ全体の coverage と branch coverage は取っていない
- 記録の小さなずれ: 計画の Progress の S2c は「21 `OrgGate` CLI tests ok」と書くが、`go test ./internal/cli/ -list OrgGate` は 20 を返し、打ち直しの PASS も top-level 20

## Verdict

- Pass: `./scripts/run-test.sh`(shell 46 本、Go 11 パッケージ、rc 0、1,754 s)、`GITHUB_BASE_REF=main` での shell の 6 本と Go の 2 つの絞った run、追加の shell の 7 本
- Fail: 0
- Blocked: なし
