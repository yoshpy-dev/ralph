# Test report: verify-slices-lighter-self-review

- Date: 2026-10-27
- Plan: `docs/plans/active/2026-10-27-verify-slices-lighter-self-review.md`(Test plan と AC7)
- Tester: tester サブエージェント(`/test`)
- Scope: `git diff 44c14b97...HEAD`(HEAD 395b9bd8、verify report のコミットを含む)。Go のコードの変更はない。文書、skill、rules、agent の定義と、shell のテスト 2 本(`tests/test-slice-verification.sh` は新規、`tests/test-self-review-scope.sh` は検査の追加)。`./scripts/run-test.sh` は changed scope を頼んだが、`.codex/agents/implementer.toml` を分類できず full に落ち、Go のテストも打った
- Evidence: `docs/evidence/test-2026-10-27-verify-slices-lighter-self-review.log`(gitignore。`run-test.sh` の全出力 5,151 行。wrapper 自身の写しは `docs/evidence/verify-2026-10-27-113448.log`)

## Test execution

| Suite / Command | Tests | Passed | Failed | Skipped | Duration |
| --- | --- | --- | --- | --- | --- |
| `./scripts/run-test.sh`(full fallback: `unclassified:.codex/agents/implementer.toml`) | shell 45 本 + Go 9 パッケージ | 45 本 OK、Go 9 パッケージ ok | 0 | 1(`test-secret-scan-branch.sh` の「git 2.41 より古い実物の git」の行。手元は 2.49.0 で、古い git は stub の行が見る) | 806 秒(20:34:48〜20:48:14 JST、開始時の load 2.4〜3.9) |
| └ Go: `internal/cli` / `internal/org` | — | ok | 0 | 0 | 232.5 秒 / 308.4 秒 |
| └ Go: `cmd/ralph`、`internal/{config,insights,org/driver,org/protocol,scaffold,upgrade}` | — | ok | 0 | 0 | 各 0.7〜3.6 秒 |
| `sh tests/test-slice-verification.sh` | 130 | 130 | 0 | 0 | 4 秒 |
| `sh tests/test-self-review-scope.sh` | 198 | 198 | 0 | 0 | 4 秒 |
| `sh tests/test-new-feature-plan.sh` | 21 | 21 | 0 | 0 | 1 秒 |
| `sh tests/test-agent-models.sh` | 47 | 47 | 0 | 0 | 8 秒 |
| `sh tests/test-check-skill-sync.sh` | 13 | 13 | 0 | 0 | 5 秒 |
| `sh tests/test-sync-skills.sh` | 22 | 22 | 0 | 0 | 10 秒 |

名前を挙げた 6 本は、`run-test.sh` を後ろで回している間に単独でも打った。6 本とも `run-test.sh` の中の local verifier でも OK だった(`tests/test-*.sh` の glob で拾われる。`tests/` の 45 本と local verifier の 45 本の OK は一致する)。

計画の Progress checklist にある数(S5 の 84 件、bc56cc4f の時点の scope 198/198)と比べると、slice のテストは 130 件に増えた。cb094ae8 と 804d05f9 が slice のテストに検査を足したためで、verify report の注記と合う。

## Coverage

- Statement: 計測なし(shell のテストに計測の道具はない。Go のコードは変わっていないので、Go の coverage は取らなかった)
- Branch: 計測なし
- Function: 計測なし
- Notes: 文言の検査の効き方は、下の「Regression checks」の変異で確かめた。新しい 2 本のテストは、基準(44c14b97)の版に戻した 6 つのファイルのすべてで落ちた

## Failure analysis

| Test | Error | Root cause | Proposed fix |
| --- | --- | --- | --- |
| なし | — | — | — |

## Regression checks

計画の Edge cases の 2 つ(`templates/base/` 側だけに残った古い文、`.agents/` の写しの作り直し忘れ)を、`git archive HEAD` で作った scratch の写しで確かめた。worktree には触れていない。変異のない写し(対照)では、`check-sync.sh` も `check-skill-sync.sh` も rc 0 だった。各変異の写しは、対照と 1 ファイルだけ違う(`diff -r -q` で確認)。

| Previously broken behavior | Status | Evidence |
| --- | --- | --- |
| m1: `templates/base/.claude/skills/self-review/SKILL.md` だけに 1 行足す | 検出 | `check-sync.sh` rc 1(`DRIFTED .claude/skills/self-review/SKILL.md`、`FAIL: 1 sync issue(s) found.`)。`check-skill-sync.sh` は rc 0(templates/base は見ない) |
| m1b: 同じファイルを基準の版(10 項目の古い文)に戻す | 検出 | `check-sync.sh` rc 1(同じ DRIFTED)。`test-self-review-scope.sh` も 187/198、FAIL 11 |
| m2: `.claude/skills/implement/SKILL.md` の手順 6 の「the narrowest check」を「the broadest check」に変え、`.agents/` を作り直さない | 検出 | `check-skill-sync.sh` rc 1(`FAIL: skill 'implement': SKILL.md body differs`、36 行目の diff)。`check-sync.sh` も rc 1 |
| m2b: `.agents/skills/implement/SKILL.md` を基準の版に戻す(写しの作り直し忘れ) | 検出 | `check-skill-sync.sh` rc 1、`check-sync.sh` rc 1 |
| m3(追加): `templates/base/.agents/skills/implement/SKILL.md` を基準の版に戻す | 検出 | `check-sync.sh` rc 1(`DRIFTED .agents/skills/implement/SKILL.md`)。`check-skill-sync.sh` は rc 0 |

新しい 2 本のテストの効き方(1 ファイルを基準の版に戻す):

| 戻したファイル | `test-slice-verification.sh` | `test-self-review-scope.sh` |
| --- | --- | --- |
| `.claude/skills/implement/SKILL.md` | 111/130、FAIL 19 | 198/198 |
| `docs/plans/templates/feature-plan.md` | 127/130、FAIL 3 | 198/198 |
| `.claude/skills/self-review/SKILL.md` | 130/130 | 187/198、FAIL 11 |
| `templates/base/.claude/skills/self-review/SKILL.md` | 130/130 | 187/198、FAIL 11 |
| `templates/base/.claude/agents/implementer.md` | 130/130 | 192/198、FAIL 6 |
| `.codex/agents/implementer.toml` | 130/130 | 192/198、FAIL 6 |

どの変異も、その文を受け持つほうのテストが落とした。もう一方のテストが通るのは受け持ちの分け方どおりで、穴ではない。

## Test gaps

- m2 の「the narrowest check」→「the broadest check」は、新しい 2 本のテストでは落ちなかった(130/130、198/198)。手順 6 の検査は「The exact verification commands are the slice's command from the plan's `## Implementation outline`」までで文字列が終わり、そのあとの「the narrowest check that covers the slice's files」を固めていない(`tests/test-slice-verification.sh:156-158`)。同じ手順の「pick the narrowest command …」(コマンドのない計画の場合)は固めている。写しの側の同じ変更は `check-skill-sync.sh` と `check-sync.sh` が拾うので、root の 4 つの写しを一緒に変えた場合だけ通る。LOW、後続の候補
- 新しい文を実際の `/implement` の実行で使った結果(implementer が 5 項目の結果を返すか、コマンドのない計画で orchestrator が Progress checklist に書くか)は、テストでは確かめられない。verify report の Coverage gaps と同じ
- Codex の custom agent としての `.codex/agents/implementer.toml` と `reviewer.toml` の振る舞いは、テストでは確かめていない(TOML として読めることは verify で確かめた)
- 文書の変更なので Go の coverage は取っていない。Go のテストは full fallback で打ち、9 パッケージとも ok だった

## Verdict

- Pass: `./scripts/run-test.sh` rc 0(full fallback、shell 45/45、Go 9 パッケージ ok、806 秒)。名前を挙げた 6 本はすべて全件 PASS(130、198、21、47、13、22)。計画の Edge cases の 2 つは変異で検出された(`check-sync.sh` が templates/base の古い文を、`check-skill-sync.sh` が `.agents/` の作り直し忘れを拾う)。AC7 のテスト側の半分を満たす
- Fail: なし
- Blocked: なし
- 判定: pass(HEAD 395b9bd8)。`/sync-docs` に進んでよい
