# Test report: org-feature-worktree-ownership

- Date: 2026-10-10(JST。実行は 16:46〜17:01)
- Plan: docs/plans/active/2026-10-10-org-feature-worktree-ownership.md(承認済み、digest feadcc05c21f)
- Tester: tester subagent (Claude Opus 5.5)。pipeline cycle 1(`cycle-count.json` は 1、上限 2)
- Scope: HEAD 9bd90993(base origin/main be050681)。behavioral test だけを実行した(静的解析は /verify の担当)。重点は 3 つ。verify が求めた 3 つの mutant、verify が Coverage gaps に挙げた「既定の org_id での `ensure` との競合を本物のスクリプトで通すテストがない」件、PR #216 の `StartFeature`・`OrgStartPlan` のテストが通り続けること
- Evidence: `docs/evidence/test-2026-10-10-org-feature-worktree-ownership.log`(`docs/evidence/*.log` は gitignore の対象で、手元にだけ残る。`git check-ignore -v` で `.gitignore:58` に当たることを確かめた)。`run-test.sh` 自身のログは `docs/evidence/verify-2026-10-10-074658.log`(時刻は UTC)
- 足したテスト: `internal/org/feature_test.go` に `TestStartFeature_RealWorktreeScript_DefaultOrgIDOtherPlan` を 1 本(+79 行、テストのファイルだけ)。理由は「Coverage gap への対応」の節

## Test execution

| Suite / Command | Tests | Passed | Failed | Skipped | Duration |
| --- | --- | --- | --- | --- | --- |
| `./scripts/run-test.sh`(HEAD、変更言語の scope) | shell 40 本(PASS の行 3,922)、Go 8 パッケージ | すべて | 0 | shell 1(下の注) | 7 分 31 秒、rc 0 |
| `go test ./internal/org/ ./internal/cli/ -count=1 -v -coverprofile`(HEAD) | top-level 967(org 469、cli 498)、subtest 1,093 | すべて | 0 | 0 | 88 s(org 20.9 s、cli 85.3 s)、rc 0 |
| `go test ./internal/org/ -count=1 -v -coverprofile`(テストを足したあと) | top-level 470、subtest 752 | すべて | 0 | 0 | 26.2 s、rc 0 |
| `go test -race -count=1 ./internal/org/...`(足したあと) | 3 パッケージ | 3 | 0 | 0 | 28.3 s、rc 0 |
| `TestStartFeature*`・`TestEnsureFailureMessages_InWorktreeScript`・`TestSpawnPrecheckErr_MatchesSpawn` を `-count=10`(足したあと) | top-level 240(24 本 × 10)、subtest 640 | すべて | 0 | 0 | 77.0 s、rc 0 |
| mutation(`go test -overlay`、HEAD のテストで `internal/org` 全体) | 15 件 | red 15 件 | 生き残り 0 | - | 1 件 18〜36 s |
| 同じ 15 件を、足したテスト 1 本だけで(`-run`) | 15 件 | red 9 件 | 生き残り 6 件(下の表) | - | 1 件 3〜10 s |

- shell の件数は suite ごとに PASS・FAIL・SKIP の行を数えた(`PASS: 29` や `SKIP: 0` のような集計行は除いた)。40 本すべてが `OK` で終わり、FAIL の行は 0。Skipped の 1 件は `tests/test-secret-scan-branch.sh` の「the real git older than 2.41 case」(手元の git は 2.49.0)
- shell の PASS の行は、前の計画の cycle 2(`docs/reports/test-2026-10-09-org-feature-worktree.md`、同じ数え方で 3,797)より 125 多い。増えたのは `tests/test-pre-bash-guard.sh`(+120)と `tests/test-check-template.sh`(+5)だけで、どちらも PR #216 の merge の前に main から入った変更の分。この差分は shell のテストに触れていない
- `run-test.sh` の中の `go test ./...` は `internal/org` だけを実行し(18.6 s)、ほかはキャッシュの結果だった。そのため 2 行目で `internal/org` と `internal/cli` を `-count=1` で流し直した。go は 1.26.0
- `run-test.sh` と 2 行目は、テストを足す前の HEAD の tree で走らせた。足すテストは、その間 scratchpad の `zz_probe_test.go` に置き、`-overlay` で worktree に足さずに確かめた。足したあとの `internal/cli` は、足したテストが `internal/org` のテストのファイルだけなので、HEAD と同じテストを走らせることになる
- 実行中、ほかのセッションの負荷で load average は 6〜20(12 コア)だった。既知の flaky なテスト(MEMORY.md の 3 本)はどれも落ちなかった

## Coverage

- Statement: `internal/org` 93.9%(HEAD と、テストを足したあとで同じ)、`internal/cli` 85.5%(HEAD)。ほかの 6 パッケージは測り直していない(`run-test.sh` ではキャッシュの結果で ok)
- 足したテストで `internal/org` の関数ごとの値は 1 つも変わらなかった(`go tool cover -func` の 3 列を比べて差分が空)。足したテストは、fake でだけ通っていた枝を本物のスクリプトと git で通し直すもので、行の coverage は増やさない
- Function(差分が変えた、または足した関数):

| 関数 | 位置 | 値 | 通らない文 |
| --- | --- | --- | --- |
| `featureCanonicalRef` | `feature.go:152` | 100.0% | なし |
| `StartFeature` | `feature.go:175` | 92.5% | leader の入力の `checkSpawnInput` が失敗する枝と台帳の読み込みの失敗。分割計画の検査を通った入力では起きない防御の枝で、PR #216 のときと同じ |
| `ensureFailureErr` | `feature.go:267` | 100.0% | なし |
| `checkFeatureWorktreeReuse` | `feature.go:397` | 100.0% | なし |
| `checkEnsuredFeatureWorktree`(新規) | `feature.go:443` | 100.0% | なし |
| `otherRecordFix`(新規) | `feature.go:481` | 100.0% | なし |
| `featureWorktreeMismatch`(新規) | `feature.go:502` | 100.0% | なし |
| `scriptFeatureWorktrees` の `Lookup` / `Ensure` / `CurrentBranch` | `feature.go:556`、`:582`、`:597` | 78.6% / 87.5% / 75.0% | この差分は触れていない。PR #216 の report と同じ値 |

- Branch: Go の標準ツールには branch coverage がないので測っていない
- Notes: 新しい 3 関数は行では 100% だが、行の coverage は「条件を外しても気づくか」を示さない。そのため mutation で確かめた(次の節)

## 依頼された mutation

`scratchpad/ofwo-test/mut.py` が `internal/org/feature.go` の写しを 1 か所だけ書き換え、`go test -overlay` で `internal/org` 全体を走らせる。worktree のファイルは書き換えていない。どの mutant も、書き換える文字列が `feature.go` に 1 回だけあることを確かめてから当てた。

| ID | 書き換え | 結果(HEAD のテスト) | 落ちたテストのうち依頼で名前の出たもの |
| --- | --- | --- | --- |
| M1 | `StartFeature` の `checkEnsuredFeatureWorktree` の呼び出しを外す | red(top-level 9 本) | `TestStartFeature_RecordChangedDuringEnsureRefused`(4 ケースとも「expected a rejection, got spawned」、つまり相手の worktree で leader が立つ)と `TestStartFeature_RealWorktreeScript_OrgIDOtherPlan`(`:1565`、同じく spawned) |
| M2 | `featureCanonicalRef(plan.Path, …)` を `featureCanonicalRef(plan.ID, …)` にする | red(top-level 10 本) | `TestStartFeature_RealWorktreeScript_OtherLedger`(`:1498`、「expected a rejection, got spawned」。台帳 B の start が台帳 A の worktree を引き継ぐ。cross-review の WORTH_CONSIDERING #1 の場面そのもの) |
| M3 | `otherRecordFix` の `if orgID == slug` を `if true` にする(明示の `--org-id` を無視して、いつも slug の案内) | red(top-level 2 本) | `TestStartFeature_OtherRecordWithOrgID`(3 ケースとも、エラー文の全文の比較 `:1070` で落ちる)。ほかに `_RealWorktreeScript_OrgIDOtherPlan` |

3 つとも、依頼された落ち方をした。M1 と M2 は、エラー文の違いではなく、拒否されるべき start が leader を立てたことで落ちている。

## 追加の mutation

同じ仕組みで 12 件を足した。

| ID | 書き換え | 結果 | 落としたテスト |
| --- | --- | --- | --- |
| M4 | `checkEnsuredFeatureWorktree` の `otherPlan` をいつも false(別の計画の記録も「打ち直す」の案内) | red | `RecordChangedDuringEnsureRefused`、`OtherRecordWithOrgID`、`_RealWorktreeScript_OrgIDOtherPlan` |
| M5 | 同じく `otherPlan` をいつも true | red | `RecordChangedDuringEnsureRefused` だけ(「the branch cannot be read」のケース) |
| M6 | 記録が消えた(`!ok`)ときに nil を返す | red | `RecordChangedDuringEnsureRefused` だけ |
| M7 | 記録が読めない(`err != nil`)ときに nil を返す | red | `RecordChangedDuringEnsureRefused` だけ |
| M8 | `ensure` のあとの拒否を `rejected` ではなく `failed` にする | red | `RecordChangedDuringEnsureRefused`、`OtherRecordWithOrgID`、`_OrgIDOtherPlan` |
| M9 | `checkFeatureWorktreeReuse` の `canonical_ref` が違う枝を外す(一般の拒否の文になる) | red | 5 本(`WorktreeRecordMismatchRefused`、`AfterDisband`、`_OtherLedger` ほか) |
| M10 | `featureWorktreeMismatch` が `canonical_ref` を比べない | red | 6 本(`_OtherLedger` を含む) |
| M11 | `otherRecordFix` が、同じブランチの記録にも slug の案内を足さない | red | `OtherRecordWithOrgID`、`_OrgIDOtherPlan` |
| M12 | `otherRecordFix` が、いつも slug の案内を足す | red | `OtherRecordWithOrgID` だけ(「before ensure, on another branch」のケース) |
| M13 | `canonical_ref` を解決前の `--plan`(`mustAbs(p.SplitPath)`)から作る | red | 10 本(`CanonicalRefIsTheResolvedPlanPath` を含む) |
| M14 | `ensure` に渡す `canonical_ref` だけを古い形 `split:<id>#<slug>` にする(照合の `want` は新しい形のまま) | red | 15 本 |
| M15 | `checkEnsuredFeatureWorktree` が、読み直した記録を自分自身と比べる(いつも一致) | red | `RecordChangedDuringEnsureRefused`、`OtherRecordWithOrgID`、`_OrgIDOtherPlan` |

M5・M6・M7・M12 は 1 本のテストだけが落とす。どれもそのテストの専用のケースが落としているので、穴ではない。ただ、そのケースを消すと気づけなくなる。

## Coverage gap への対応(既定の org_id と `ensure` の競合を本物のスクリプトで)

verify の Coverage gaps の 2 つ目(既定の org_id で `ensure` と競合する場合は fake の `Ensure` でだけ通る)に、テストを 1 本足した。`spyWorktrees` の `ensureHook` は既存のものをそのまま使えたので、足すのは安かった。

`TestStartFeature_RealWorktreeScript_DefaultOrgIDOtherPlan`(`internal/org/feature_test.go` の末尾)は、`_OrgIDOtherPlan` と同じ組み立てで `--org-id` を渡さない。

1. 2 つの台帳に、同じ id(`auth-split`)と同じ slug(`auth-core`)の分割計画を置く。台帳 B の start の `ensure` の中で、台帳 A の start が worktree を作る
2. 本物のスクリプトは台帳 A の記録を台帳 B に返す。台帳 B は `ensure` のあとの照合で拒否する。エラー文は全文で固定し、`cleanup` も `--org-id` も含まないことを見る。呼び出しは lookup、ensure、lookup の順で、台帳 B には何も書かれず、driver も呼ばれない
3. 台帳 A の記録・ブランチの commit・チェックアウトは、`last_seen_at` を除いて、A の start が作ったときのまま
4. 同じ start をもう一度打つと、使い回しの検査で拒否される(呼ばれるのは lookup だけ)。エラー文は slug の変更の案内で始まる
5. 案内どおり、台帳 B の計画の slug を `auth-store` にして承認し直すと start が通り、自分の worktree とブランチ(`fix/auth-store`)を持つ。台帳 A の状態は 3 のときから変わらない

確かめたこと:

- HEAD で通る(足す前に `-overlay` で、足したあとは tree で。`-count=10` で 10 回とも通った)
- 足したテストだけを 15 件の mutant に当てると(上の表の最後の行)、M1・M2・M4・M8・M9・M10・M13・M14・M15 の 9 件で落ちる。M1(照合を外す)・M4(別の計画の枝を外す)・M15(照合がいつも一致)は、これまで既定の org_id では fake のテストでしか落ちなかった。生き残った 6 件は `--org-id` の枝(M3・M11・M12)と、記録が消えた・読めない・ほかの点が違う場合の枝(M5・M6・M7)で、このテストが通らない経路なので、生き残るのが正しい
- HEAD のテストで生き残っていた mutant はないので、足したテストだけが落とす mutant はない。足したテストの役目は、fake が本物のスクリプトと同じく振る舞うという verify の推測(「おそらく本物でも同じ挙動になる。未確認です」)を、本物のスクリプトで確かめることにある。既定の org_id でも、`ensure_worktree` は `canonical_ref` を見ずに既存の記録を返し、ralph の側の照合がそれを拒否した

## Failure analysis

| Test | Error | Root cause | Proposed fix |
| --- | --- | --- | --- |
| なし | - | - | - |

## Regression checks

| Previously broken behavior | Status | Evidence |
| --- | --- | --- |
| PR #216 の `StartFeature`・split・leader の spawn・CLI の `start --plan` のテスト(be050681 の `internal/org/feature_test.go` 20 本、`spawn_feature_test.go`、`split_test.go`、`internal/cli/org_feature_test.go` の計 49 本) | 通る | 49 本の名前を、2 行目と 3 行目の `-v` の出力の `--- PASS:` と照合し、欠けは 0。`TestOrgStartPlan_*` の 5 本と `TestOrgStatus_Feature*`・`_Incomplete*` も含む |
| cross-review cycle 2 の WORTH_CONSIDERING #1(別の台帳の同じ名前の分割計画が、残った worktree を引き継ぐ) | 直っている | `_RealWorktreeScript_OtherLedger` が通り、M2 で落ちる(台帳 B の start が spawned になる) |
| 同 #2(別々の分割計画からの同時の start で、台帳の結びつきと worktree の記録が食い違う) | 直っている | `RecordChangedDuringEnsureRefused`(fake)、`_RealWorktreeScript_OrgIDOtherPlan`(本物、`--org-id`)、足した `_RealWorktreeScript_DefaultOrgIDOtherPlan`(本物、既定の org_id)が通り、M1 で落ちる |
| AC4: `canonical_ref` の形が変わったあとも既存のテストが通る | 通る | 2 行目と 3 行目。古い形 `split:auth-split#auth-core` を期待するのは、PR #216 の記録を拒否する `WorktreeRecordMismatchRefused` の「this feature's record from PR #216」だけで、意図した使い方 |
| AC1 の edge case: 空白を含む台帳のパス、symlink を通した `--plan` | 通る | 空白は `_OtherLedger` と `_OrgIDOtherPlan` の `ledger b`(本物のスクリプトで記録を読み戻す)、symlink は `CanonicalRefIsTheResolvedPlanPath`(fake。M13 で落ちる) |

## Test gaps

- 2 つの start を本当に並行で走らせるテストはない。どのテストも `ensureHook` の中で順に走らせている。計画の Non-goals どおりで、tech-debt の同時の start の行に残る
- symlink を通した `--plan` は fake の worktree でだけ確かめている。本物のスクリプトは `--canonical-ref` を argv から `jq --arg` で書くだけなので、おそらく同じになる。未確認です
- 大文字小文字を区別しない FS で `--state-dir` の綴りが前回と違う場合(self-review の F-3、verify の V-2)は、テストがない。拒否の側に倒れるので害は小さい
- `scriptFeatureWorktrees` のエラーの枝(`state-path` が何も出さない、記録が JSON でない、`git branch --show-current` の失敗)は、PR #216 のときから通っていない。この差分は触れていない
- 実機(herdr の上の leader)では確かめていない。計画も求めていない

## Verdict

- Verdict: pass
- Pass: `./scripts/run-test.sh` は rc 0(shell 40 本、Go 8 パッケージ)。`internal/org` と `internal/cli` を `-count=1` で流し直して 0 fail、足したテストを入れて `-race` と `-count=10` も 0 fail。依頼の 3 つを含む 15 件の mutant はすべて HEAD のテストで落ちた。verify の挙げた gap(既定の org_id での `ensure` との競合)は、本物のスクリプトのテストを 1 本足して閉じた
- Fail: 0。flake も出なかった
- Blocked: なし。Test gaps の 5 件は merge を止めない
