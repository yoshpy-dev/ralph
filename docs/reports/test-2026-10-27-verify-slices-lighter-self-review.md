# Test report: verify-slices-lighter-self-review

- Date: 2026-10-27
- Plan: `docs/plans/active/2026-10-27-verify-slices-lighter-self-review.md`(Test plan と AC7)
- Tester: tester サブエージェント(`/test`)。パイプラインの cycle 2 / 2。この report は cycle 1 の report(869dcab4、HEAD 395b9bd8)を置き換える
- Scope: `git diff 44c14b97...HEAD`(HEAD 9c7186ba)。cycle 1 のあとに振る舞いの文とテストを変えたのは dba1c20f(止まったスライスを resume として出し直す、implementer が引き継ぐ例外、「narrowest」の needle)、d33bb358(resume の前に秘密情報を worktree から出す、戻すコマンド、秘密情報の値を報告に書かない、handoff の表の Resume の行、手順 6 の例外)、6741f9d6(model-routing の順序、Codex の README の 5 項目の並びを固める)と、sync-docs の f2f0333c。Go のコードの変更はない。`./scripts/run-test.sh` は changed scope を頼んだが、`.codex/agents/implementer.toml` を分類できず full に落ち、Go のテストも打った
- Evidence: `docs/evidence/test-2026-10-27-verify-slices-lighter-self-review.log`(gitignore。`run-test.sh` の全出力 5,179 行のあとに、`(cached)` だった Go パッケージの打ち直しと、変異の各回の要約を足した。wrapper 自身の写しは `docs/evidence/verify-2026-10-27-135249.log`)

## Test execution

| Suite / Command | Tests | Passed | Failed | Skipped | Duration |
| --- | --- | --- | --- | --- | --- |
| `./scripts/run-test.sh`(full fallback: `unclassified:.codex/agents/implementer.toml`) | shell 45 本 + Go 9 パッケージ | 45 本 OK、Go 9 パッケージ ok | 0 | 1(`test-secret-scan-branch.sh` の「git 2.41 より古い実物の git」の行。手元は 2.49.0 で、古い git は stub の行が見る) | 929 秒(22:52:49〜23:08:18 JST、開始時の load 4.1〜4.7、終了時 6.5) |
| └ Go: `internal/org` / `internal/config` / `internal/scaffold` | — | ok | 0 | 0 | 358.8 秒 / 0.96 秒 / 1.0 秒 |
| └ Go: `cmd/ralph`、`internal/{cli,insights,org/driver,org/protocol,upgrade}` | — | ok(`(cached)`) | 0 | 0 | — |
| `go test -count=1` で上の `(cached)` の 6 パッケージを打ち直し | 6 パッケージ | 6 ok | 0 | 0 | 493 秒(23:09:44〜23:17:57、load 5.0〜6.6)。`internal/cli` 489.4 秒、ほかは 1.1〜3.3 秒 |
| `sh tests/test-slice-verification.sh` | 150 | 150 | 0 | 0 | 4 秒 |
| `sh tests/test-self-review-scope.sh` | 206 | 206 | 0 | 0 | 4 秒 |
| `sh tests/test-new-feature-plan.sh` | 21 | 21 | 0 | 0 | 1 秒 |
| `sh tests/test-agent-models.sh` | 47 | 47 | 0 | 0 | 10 秒 |
| `sh tests/test-check-skill-sync.sh` | 13 | 13 | 0 | 0 | 5 秒 |
| `sh tests/test-sync-skills.sh` | 22 | 22 | 0 | 0 | 11 秒 |

名前を挙げた 6 本は、`run-test.sh` を後ろで回している間に HEAD で単独でも打った。6 本とも `run-test.sh` の中の local verifier でも OK だった(`tests/test-*.sh` の 45 本と、log の `OK` の 45 行が一致する)。

`run-test.sh` の Go の 9 パッケージのうち 6 つが `(cached)` だった。`cmd/ralph` はルートのパッケージ経由で `templates/` を embed しており、このブランチは `templates/base/` の文書を変えているので、`-count=1` で打ち直した。6 つとも ok。

cycle 1 と比べると、slice のテストは 130 件から 150 件に、scope のテストは 198 件から 206 件に増えた。dba1c20f、d33bb358、6741f9d6 が足した検査の分で、ほかの 4 本の件数は変わっていない。

## Coverage

- Statement: 計測なし(shell のテストに計測の道具はない。Go のコードは変わっていないので、Go の coverage は取らなかった)
- Branch: 計測なし
- Function: 計測なし
- Notes: 新しい検査の効き方は、下の「Regression checks」の変異で確かめた

## Failure analysis

| Test | Error | Root cause | Proposed fix |
| --- | --- | --- | --- |
| なし | — | — | — |

## Regression checks

`git archive HEAD` で作った写しを scratchpad に 5 つ置き(対照 1 つと変異 4 つ)、変異はそこでだけ入れた。worktree には触れていない。対照では 6 本のテストがすべて全件 PASS で、`check-sync.sh` と `check-skill-sync.sh` は rc 0 だった。各変異の写しは、対照と変えたファイルの分だけ違う(`diff -r -q` で 4、4、2、4 ファイル)。変異はどれも写しのすべてに同時に入れたので、写しのずれは起きず、`check-sync.sh` と `check-skill-sync.sh` はどの変異でも rc 0 のままだった。拾えるのは文言のテストだけになる。

| 変異 | 拾ったテスト | 結果 | 落ちた検査 |
| --- | --- | --- | --- |
| (a) implement の SKILL.md の写し 4 つ(`.claude/`、`.agents/`、その `templates/base/` の 2 つ)で、手順 7 の Delegated slices の文から「as a resume」を消す | `tests/test-slice-verification.sh` | 146/150、FAIL 4 | `<file> delegated gate dispatches a stopped slice again as a resume`(写しごとに 1 つ) |
| (b) implementer の定義 4 つ(`.claude/agents/implementer.md`、`.codex/agents/implementer.toml`、その `templates/base/` の 2 つ)で、「Before any edit」の段落から resume の文(「If the handoff says it resumes a slice …」)を消す | `tests/test-self-review-scope.sh` | 202/206、FAIL 4 | `<file> takes over a stopped slice's changes on a resume`(定義ごとに 1 つ) |
| (c) `.codex/README.md` と `templates/base/.codex/README.md` で、「files outside the slice」の前の「and」を消す | `tests/test-slice-verification.sh` | 148/150、FAIL 2 | `<file> implementer row lists the five items in the canonical words`(写しごとに 1 つ) |
| (d、追加) cycle 1 で残った穴: implement の SKILL.md の写し 4 つで、手順 6 の「the narrowest check that covers」を「the broadest check that covers」に変える | `tests/test-slice-verification.sh` | 146/150、FAIL 4 | `<file> step 6 takes the slice command from the plan, the narrowest check`(写しごとに 1 つ) |

どの変異でも、落ちたのはその文を受け持つテストだけで、ほかの 5 本は全件 PASS だった。(a) と (c) は slice のテストが、(b) は scope のテストが拾う。この分け方は cycle 1 の受け持ちと同じ。

(d) は cycle 1 の report の Test gaps に LOW で書いた穴で、cycle 1 では `narrowest` を `broadest` に変えても 2 本とも通った。dba1c20f で needle が文の終わり(「the narrowest check that covers the slice's files.」)まで伸び、今回は 4 つの写しのすべてで落ちた。この穴はふさがった。

## Test gaps

- d33bb358 が足したほかの needle(手順 6 の例外、止まったときに秘密情報を先に worktree から出す、外したファイルを `git restore --staged --worktree` で戻す、inline のスライスで秘密情報をファイルと行で書く、model-routing の Resume の行と resume の順序、implementer の定義の「its file and line, never its value」)は、HEAD で PASS することは確かめたが、1 つずつ変異で落としてはいない。needle は文か箇条の終わりまでの完全一致で、「there is no new commit to check」で切れるものは、続く「First take a secret out of the worktree …」の文を別の検査が固めている。inline のスライスの needle は「Then commit」で切れ、続く「with conventional format」を固めていないが、これはコミットの形式の文で、このブランチが変えた語ではない。cycle 1 の (d) のように、このブランチが変えた語の手前で切れた needle は、今回は見当たらなかった
- 新しい文を実際の `/implement` の実行で使った結果(止まったスライスを resume として出し直したとき、新しい implementer が残りの変更を引き継いで止まらないか)は、テストでは確かめられない。cycle 1 と同じ穴
- Codex の custom agent としての `.codex/agents/implementer.toml` の振る舞いは、テストでは確かめていない
- 文書の変更なので Go の coverage は取っていない。Go のテストは full fallback と `-count=1` の打ち直しで、9 パッケージとも ok だった

## Verdict

- Pass: `./scripts/run-test.sh` rc 0(full fallback、shell 45/45、Go 9 パッケージ ok、929 秒)。`(cached)` の 6 パッケージも `-count=1` で ok(493 秒)。名前を挙げた 6 本はすべて全件 PASS(150、206、21、47、13、22)。頼まれた 3 つの変異は、それぞれ受け持ちのテストが写しごとに 1 件ずつ落とした。cycle 1 の `narrowest` の穴もふさがった。AC7 のテスト側の半分を満たす
- Fail: なし
- Blocked: なし
- 判定: pass(HEAD 9c7186ba)。`/sync-docs` に進んでよい
