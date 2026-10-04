# Test report: subagent-model-defaults

- Date: 2026-10-04
- Plan: docs/plans/active/2026-10-04-subagent-model-defaults.md
- Tester: tester subagent (Claude)、pipeline cycle 1
- Scope: branch chore/subagent-model-defaults の HEAD bea18f4b と base main 11602fed の差分。実装のコミットは 173cf76d(Slice A)と de01c50f(Slice B)。behavioral test だけを実行した(static analysis は /verify で済んでいる)。plan の AC-3 と、AC-6 のうち test の半分を確かめた
- Evidence: `docs/evidence/test-2026-10-04-subagent-model-defaults.log`(`docs/evidence/*.log` は gitignore 対象なので commit しない)。`run-test.sh` 自身が書いたログは `docs/evidence/verify-2026-10-04-095132.log`

## Test execution

| Suite / Command | Tests | Passed | Failed | Skipped | Duration |
| --- | --- | --- | --- | --- | --- |
| `sh tests/test-agent-models.sh`(macOS の `/bin/sh` は bash 3.2 の posix モード、awk は BSD awk 20200816) | 47 | 47 | 0 | 0 | 6.1 s |
| `dash tests/test-agent-models.sh` | 47 | 47 | 0 | 0 | 5.7 s |
| 同じテストを ubuntu:24.04 の docker で(`sh` = dash、`dash`、`bash`。awk は mawk) | 47 × 3 | 47 × 3 | 0 | 0 | - |
| `RALPH_VERIFY_SCOPE=full ./scripts/run-test.sh` | shell 33 ファイル(1,523 件)、Go 8 パッケージ | すべて | 0 | 0(`[no test files]` の 2 パッケージを除く) | 279 s |
| `go test -count=1 ./...`(キャッシュなしで再実行) | 8 パッケージ | 8 | 0 | 0 | `internal/cli` 62 s、ほかは 0.5〜13 s |
| 独立の mutation harness(下の 3 節。macOS の sh) | 35 ケース | 35 | 0 | 0 | - |
| 同じ harness を dash で、ubuntu:24.04(dash + mawk)で | 35 × 2 | 35 × 2 | 0 | 0 | - |

- `run-test.sh` の中の `go test ./...` は `internal/org` 以外がキャッシュの結果だった。`templates/base/` は `internal/scaffold` に go:embed されるので、キャッシュに頼らず `-count=1` で全パッケージを流し直し、すべて `ok` を確かめた。
- `tests/test-agent-models.sh` は `scripts/verify.local.sh:213` の `tests/test-*.sh` の glob で拾われ、full run の最初のファイルとして走った。この glob は scope に関係なく回るので、CI の `./scripts/run-verify.sh` でもこのテストが走る。
- harness の 35 ケースの「Passed」は、ケースごとに私が事前に決めた期待(落ちる、または通る)どおりだったかを数えた。最初の実行で期待と違った 2 ケース(P1、P8)は、既知の tech-debt と設計どおりの挙動だったので、下の表には実際の挙動を書いた。

## AC-3 の red の証拠(implementer とは独立に作った mutation)

手順: `git archive HEAD` で入力 5 種(root と template の `agents/*.md`、2 つの `model-routing.md`、`tests/test-agent-models.sh`)を scratchpad の空白を含むディレクトリに展開して原本にした。ケースごとに原本を丸ごと複製し、perl で 1 か所を書き換え、`cmp` で書き換えが効いたことを確かめてから、複製の中で `sh tests/test-agent-models.sh` を実行した。テストは自分の位置から PROJECT_ROOT を求めるので、複製の中のスクリプトを使った。原本の shasum は実行の前後で一致し、worktree の `git status --porcelain` は空のままだった。テストの self-test の mutator(`mut_*`)は使っていない。

real tree の検査で FAIL が出ると self-test は SKIP される(Slice A の逸脱 (1))。そのため mutation のケースの合計は 47 ではなく 16〜17 になる。

| ID | Mutation | rc | PASS / total | FAIL の出力(ファイルと agent) |
| --- | --- | --- | --- | --- |
| M1 | root の `.claude/agents/verifier.md` を `model: sonnet` に | 1 | 15 / 17 | `.claude/agents/verifier.md (agent verifier): frontmatter model sonnet but .claude/rules/ralph/model-routing.md tier table says opus`、比較側の `... (agent verifier): frontmatter model sonnet (root) != opus (template)` |
| M2 | template だけ `templates/base/.claude/agents/reviewer.md` を `model: opus` に | 1 | 15 / 17 | `templates/base/.claude/agents/reviewer.md (agent reviewer): frontmatter model opus but templates/base/.claude/rules/ralph/model-routing.md tier table says sonnet`、比較側の `sonnet (root) != opus (template)` |
| M3a | template の `model-routing.md` の「Review and doc seats」行を `` `opus` `` に | 1 | 15 / 17 | template の `reviewer.md (agent reviewer)` と `doc-maintainer.md (agent doc-maintainer)` が `tier table says opus` |
| M3b | template の「Implementation and verification seats」行を `` `sonnet` `` に | 1 | 14 / 17 | template の `implementer.md`、`tester.md`、`verifier.md` が `tier table says sonnet` |
| M4 | root の tier 表から `` (`tester`) `` を消す | 1 | 16 / 17 | `.claude/agents/tester.md (agent tester): not listed in any "## Tier table" row of .claude/rules/ralph/model-routing.md` |
| M5 | root の pin の記述を `` `model: sonnet` pinned in frontmatter `` に戻す | 1 | 16 / 17 | `.claude/rules/ralph/model-routing.md: pin sentence says model sonnet but .claude/agents/implementer.md frontmatter model is opus` |
| X1 | template だけ `verifier.md` を `sonnet` に((a) の template 側) | 1 | 15 / 17 | template の `verifier.md (agent verifier)` と比較側 |
| X2 | root の `reviewer.md` を `opus` に | 1 | 15 / 17 | `.claude/agents/reviewer.md (agent reviewer)` と比較側 |
| X3 | root の「Review and doc seats」行を `` `opus` `` に((b) の root 側) | 1 | 15 / 17 | root の `doc-maintainer.md` と `reviewer.md` |
| X4 | template の tier 表から `` (`tester`) `` を消す((c) の template 側) | 1 | 16 / 17 | `templates/base/.claude/agents/tester.md (agent tester): not listed ...` |
| X5 | template の pin の記述を `model: sonnet` に((d) の template 側) | 1 | 16 / 17 | `templates/base/.claude/rules/ralph/model-routing.md: pin sentence says model sonnet but templates/base/.claude/agents/implementer.md frontmatter model is opus` |
| X6 | template の tier 表から `` (`reviewer`) `` を消す | 1 | 16 / 17 | `templates/base/.claude/agents/reviewer.md (agent reviewer): not listed ...` |
| X7 | 両側の `implementer.md` を `sonnet` に(表と pin は `opus` のまま) | 1 | 13 / 17 | 両側で、表との不一致と pin の不一致の 2 行ずつ、計 4 行 |

依頼された 5 通り(M1〜M5)と、AC-3 の (a)〜(d) を root と template の両側に当てた組み合わせは、すべて exit 1 で、落ちたファイルと agent 名を出した。template 側だけを変えたケース(M2、M3a、M3b、X1、X4、X5、X6)も落ちる。AC-3 を満たす。

## Edge cases(plan の Test plan)

| ID | ケース | 期待 | 結果 |
| --- | --- | --- | --- |
| E1a | root の `doc-maintainer.md` から `model:` を消す | 落ちる | rc 1。`frontmatter has no model: value (omitting it means inherit, which model-routing.md forbids)` と、比較側の `<none> (root) != sonnet (template)` |
| E1b | `reviewer.md` の `model:` を両側で消す | 落ちる | rc 1、14 / 16。両側の FAIL 2 行。比較側は PASS 行を出さない(Slice B の F-5 の修正どおり) |
| E1c | root の `implementer.md` から `model:` を消す | 落ちる | rc 1。上の 2 行に加えて `pin sentence cannot be checked because .claude/agents/implementer.md has no frontmatter model` |
| E1d | `reviewer.md` の `model:` を frontmatter の外(本文の先頭)に移す | 落ちる | rc 1。frontmatter に `model:` がないものとして扱われる |
| E2a | root の表に agent でない backtick 語を足す: `planner`、`design`、`opus`、`sonnet`、`foo.bar`、`../agents/x`、haiku 行の `Explore`、Orchestrator 行の `reviewer` | 通る | rc 0、47 / 47 |
| E2b | template の haiku 行に `general-purpose` を足す | 通る | rc 0、47 / 47 |
| E3a | root の haiku 行にも `tester` を書く | 落ちる | rc 1。`.claude/agents/tester.md (agent tester): listed in 2 "## Tier table" rows of .claude/rules/ralph/model-routing.md (must be exactly one)` |
| E3b | template に `sonnet` の行を 1 行足し、そこにも `reviewer` を書く(2 行のモデルは同じ) | 落ちる | rc 1。`listed in 2 ... rows`。モデルが一致していても重複を見逃さない |
| E3c | root の同じ行に `tester` を 2 回書く | 通る(1 回と数える) | rc 0、47 / 47 |

## 追加の probe

| ID | ケース | 結果 |
| --- | --- | --- |
| P1 | root の pin の記述を 2 行に折り返す(値は `opus` のまま) | real tree の検査は PASS。self-test の (d) と (h) が `self-test setup: mut_pin left .claude/rules/ralph/model-routing.md unchanged (mutation did not apply)`(`mut_drop_pin` も同じ)で赤になり、rc 1、45 / 47。`docs/tech-debt/README.md` の最終行 (b)(N-1)に書かれたとおりの挙動を確認した |
| P2 | 折り返したうえで `sonnet` に戻す | rc 1。`pin sentence says model sonnet`。行をつないで照合するので、折り返しても検出は外れない |
| P3 | 別の段落に 2 つめの pin の記述を `sonnet` で足す | rc 1、17 / 18。2 つめだけが FAIL になる |
| P4 | 見出し `## Tier table` を `## Tiers` に変える | rc 1。`no row with a backticked model under "## Tier table"` と、5 agent の `not listed` |
| P5 | 両側の `tester.md` を `model: "opus"`(引用符つき)にする | rc 1。`frontmatter model "opus" but ... tier table says opus`。文字列をそのまま比べるので引用符つきは赤になる |
| P6 | 両側の `tester.md` の値の後ろに空白 2 つと CR を足す | rc 0、47 / 47。末尾の空白と CR は取り除かれる |
| P7 | 表にない agent `newseat.md` を両側に足す | rc 1。両側で `not listed in any "## Tier table" row` |
| P8 | `doc-maintainer.md` を両側から消す(表の行は残す) | rc 1、37 / 38。検査本体は通り、self-test の fixture guard が `self-test fixture agent doc-maintainer not found under templates/base/.claude/agents` を出す。Slice B の F-6 で入れた明示の失敗で、設計どおり |
| P8b | fixture でない `verifier.md` を両側から消す(表の行は残す) | rc 0、44 / 44。テストのヘッダの Known limits 2 つめのとおり、通る |
| P8c | template からだけ `verifier.md` を消す | rc 1。`.claude/agents/verifier.md (agent verifier): missing from templates/base/.claude/agents` |
| P9 | 両側で doc-maintainer を表と frontmatter の両方で `haiku` に移す | rc 0、47 / 47。期待値を直書きしない設計(Design decisions)のとおり、そろえて直せば通る |
| P10 | 両側の `tester.md` を `sonnet` にする(表は `opus` のまま) | rc 1。両側で表との不一致。root と template の比較だけなら通るケースで、表との照合が拾う |
| Z0 | 両側の `agents/*.md` をすべて消す | rc 1、0 / 4。`no agent files found` が両側、`pin sentence cannot be checked` が両側 |
| S1 | `/` から絶対パスで実行し、`TMPDIR` を空白を含むディレクトリにする | rc 0、47 / 47。終了後に一時ディレクトリは残らない |
| S2 | 実行中に TERM、HUP、INT を送る(INT は perl で既定の処理に戻してから exec) | どれも rc 1 で、一時ディレクトリは残らない(実行中は 1 個あった) |

35 ケースの判定(rc と FAIL 行)は、macOS の sh と dash、ubuntu:24.04 の dash + mawk の 3 通りで同じだった。

## Coverage

- Statement / Branch / Function: shell なので計測の道具はない。代わりに検査本体の FAIL 分岐を数えた。
- Notes: 検査本体(`check_side`、`check_side_inputs`、`check_agent_models`、`check_one_agent`、`check_pin_sentence`、`check_cross`)の FAIL 分岐は 15 個ある。テストの self-test が通るのは 12 個。残りの 3 個(表に backtick のモデルの行がない、agent ファイルが 0 個、implementer に `model:` がない)は、私の harness の P4、Z0、E1c で通り、どれも期待どおりの文言で落ちた。15 個すべてが、どちらかで少なくとも 1 回は通っている。どちらも通っていないのは `emit_results` の防御的な 2 分岐(tech-debt (a) に記録済み)と、self-test の準備の失敗 2 つ(`add_fixture_agent` の「real tree に同名がある」と `mutate` の「fixture にパスがない」)。
- Go: この差分に Go の変更はない。キャッシュなしの `go test -count=1 ./...` が全パッケージ `ok` だったことだけを確かめ、カバレッジは測っていない。

## Failure analysis

| Test | Error | Root cause | Proposed fix |
| --- | --- | --- | --- |
| なし | - | - | - |

どのスイートにも失敗はない。harness で最初の期待と違った P1 と P8 は製品の失敗ではない。P1 は tech-debt (b)(N-1)に書かれた self-test の制約で、P8 は fixture agent が消えたことを明示する設計どおりの失敗。

## Regression checks

| Previously broken behavior | Status | Evidence |
| --- | --- | --- |
| agent の `model:` を検査するテストがなかった(plan の調査) | 解消 | `tests/test-agent-models.sh` 47 / 47。M1〜M5、X1〜X7 で red |
| template 側の `model-routing.md` の表は `check-sync.sh` の KNOWN_DIFF で守られていなかった(Codex plan advisory の MEDIUM) | 解消 | M2、M3a、M3b、X1、X4〜X6 で template だけの変更が落ちる |
| 既存の shell テスト 32 ファイル | 通過 | full run で 33 ファイルすべて `OK`。agent 定義を読む `test-agent-phase-boundaries.sh` は 44 / 44、`test-self-review-scope.sh` は 64 / 64 |
| Go のテスト | 通過 | `go test -count=1 ./...` で 8 パッケージ `ok` |

## Test gaps

- G-1(LOW、既知): tech-debt (b)(N-1)を P1 で再現した。pin の記述を折り返すと、検査本体は正しく動くが self-test の (d) と (h) が `mutation did not apply` で赤になる。今の pin の記述は 1 行に収まっているので、現状の結果には影響しない。
- G-2(LOW、既知): tech-debt (a)。self-test は FAIL 分岐 3 つと `emit_results` の 2 つを通らない。3 つは harness で外から通し、検査本体が正しく落ちることは確かめた。足りないのは、それを将来も守る self-test のケースだけ。
- G-3(LOW、観察): self-test の fixture agent(`TARGET_AGENT="tester"`、`CROSS_FIXTURE_AGENT="doc-maintainer"`、`tests/test-agent-models.sh:38` と `:40`)を将来消したり改名したりすると、検査本体が通っていてもテストは赤になる(P8)。出る文言が原因を名指しするので、気づかずに困ることはないと考える。
- G-4(LOW、観察): `model: "opus"` のように引用符で囲むと赤になる(P5)。Claude Code が引用符つきの値を受け付けるかは確かめていない。今の 10 ファイルはどれも引用符なしなので、影響はない。
- G-5(LOW、観察): agent ファイルが 1 つもないとき(Z0)と、両側から `implementer.md` を消したとき、pin の検査は `implementer.md has no frontmatter model` と出す。ファイルそのものがない場合も同じ文言になるので、原因の説明として少し不正確。どちらの場合もテストは落ちる。
- G-6(範囲外、未解消を確認): verify の D-1(`.codex/README.md:74` と template の同じ行が implementer を「the `sonnet` tier」と書いている)は HEAD bea18f4b でも残っている。/sync-docs の担当で、`tests/test-agent-models.sh` は `model-routing.md` しか読まないので、このテストでは検出できない。
- 証拠の範囲: AC-3 の red は scratch の複製で確かめた。実際の Claude Code が新しい frontmatter の値でサブエージェントを起動するかは、この phase では確かめていない(plan の Progress notes には、Slice B の implementer を `opus` を明示して起動した記録がある)。
- insight event(`./scripts/insights-append.sh`)は追記していない。追記先の `docs/insights/events/` は commit 対象で、今回は report だけを commit するよう指示されているため。self-review と verify の phase もこの slug の event を書いていない。

## Verdict

- Pass: `tests/test-agent-models.sh` は sh と dash で 47 / 47(ubuntu の dash、bash、mawk でも 47 / 47)。`RALPH_VERIFY_SCOPE=full ./scripts/run-test.sh` は exit 0(shell 33 ファイル 1,523 件、Go 8 パッケージ)。キャッシュなしの `go test -count=1 ./...` も全 `ok`。AC-3 の red は M1〜M5 と X1〜X7 のすべてで exit 1 になり、ファイルと agent 名を出した。plan の edge case 3 種(`model:` なし、agent でない backtick 語、同じ agent が 2 行)は期待どおり。AC-3 と、AC-6 の test の半分を満たす
- Fail: なし
- Blocked: なし。/sync-docs に進んでよい
