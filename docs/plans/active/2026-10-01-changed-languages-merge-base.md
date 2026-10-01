# changed-languages-merge-base

- Status: Draft
- Owner: Claude Code
- Date: 2026-10-01
- Related request: `scripts/detect-changed-languages.sh` は `RALPH_VERIFY_BASE` が空のとき `@{upstream}` を base にする。push 済みの作業 branch では upstream が HEAD と同じコミットを指すので差分が空になり、`scope=changed reason=no_changes docs_only=true languages=` を返す。`changed` を既定にする `run-static-verify.sh`(`/verify`)と `run-test.sh`(`/test`)は、Go を変えた branch でも言語 pack の検査を 1 つも走らせないまま通る。issue #190
- Related issue: 190
- Type: fix
- Branch: fix/changed-languages-merge-base

## Objective

`RALPH_VERIFY_BASE` を指定しないときの base を「default branch との merge-base」にする。push 済みの branch でも、default branch から分かれた後の変更に応じた言語 pack が選ばれるようにする。

## 調査で確認したこと(2026-10-01、main a3adc13)

- base の決め方(`scripts/detect-changed-languages.sh` 119〜133 行、template と byte 一致): `RALPH_VERIFY_BASE` → `@{upstream}` → `origin/main` → `main` → `origin/master` → `master`。見つからなければ `no_diff_base` で full、merge-base が取れなければ `no_merge_base:<ref>` で full。差分は merge-base..HEAD、未ステージ、ステージ済み、未追跡の和。空なら `no_changes`(docs_only=true)。
- scratch の fixture(bare の remote に main と feature を push、feature に Go の commit、upstream == HEAD)で再現: 既定は `reason=no_changes languages=`、`RALPH_VERIFY_BASE=main` なら `languages=golang`。push だけでは `origin/HEAD` は作られない。
- issue の記述の訂正: `run-verify.sh` の既定の scope は `full` で、この検出器を読まない(`docs/quality/quality-gates.md` 39〜42 行も同じ説明)。影響を受けるのは、`changed` を既定にする `run-static-verify.sh` と `run-test.sh`、つまり `/verify` と `/test`。implementer が slice の後に走らせる `run-verify.sh` は影響を受けない。CI は `RALPH_VERIFY_SCOPE=full` なので影響を受けない。
- `run-verify.sh` は changed scope のとき `==> Language scope: changed (<reason>)` と `==> Language packs selected: none` をすでに出す。issue の「少なくとも 1 行出す」の案は、base を直せば不要になる。
- 既存の default branch の helper(`scripts/ralph-common.sh` の `default_branch`、`scripts/xreview-helpers.sh` の `detect_base_branch`)は、どちらも `origin/HEAD` の先を見てからローカルの main / master を返す。この検出器は単体で動くスクリプトで、`tests/test-run-verify-scope.sh` の fixture もこれだけをコピーするので、helper を source しない。
- この meta-repo の main のチェックアウトでは `refs/remotes/origin/HEAD` が `refs/remotes/origin/main` を指している(`git clone` が作る)。

## Scope

- `scripts/detect-changed-languages.sh`(+ template、byte 一致):
  - `RALPH_VERIFY_BASE` が空のときの base を、`origin/HEAD` の指す remote の branch(ref が実在するときだけ)→ `origin/main` → `main` → `origin/master` → `master` の順で決める。`@{upstream}` は使わない(upstream は push 先で、base ではない)。
  - `RALPH_VERIFY_BASE` の明示は従来どおり最優先。
  - ヘッダのコメントに base の決め方を書く。
- `tests/test-detect-changed-languages.sh`: 次のケースを足す。bare の remote を使う。
  - push 済みの branch(upstream == HEAD)で Go の commit → `languages=golang`。
  - 同じ状態で `RALPH_VERIFY_BASE=origin/feature`(== HEAD)→ `no_changes`(明示が勝つ)。
  - default branch が main / master ではない(`trunk`)remote で、`origin/HEAD` が `origin/trunk` を指す → `languages=golang`。
  - `origin/HEAD` が存在しない ref を指す → `origin/main` / `main` に落ちて `languages=golang`。
  - default branch の上で、未 push の commit も作業ツリーの変更もない → `no_changes`(従来どおり)。
- `tests/test-run-verify-scope.sh`: push 済みの feature branch(upstream == HEAD)で Go を変えた commit があるとき、`run-static-verify.sh` の既定が golang の pack を呼ぶケースを足す(end-to-end)。
- `docs/quality/quality-gates.md`(+ template): changed scope の段落に、差分の base(`RALPH_VERIFY_BASE`、なければ default branch との merge-base)を 1 文で足す。

## Non-goals

- `run-verify.sh` の出力や既定の scope の変更(既定の `full`、wrapper の `changed` はそのまま)。
- CI(`.github/workflows/verify.yml`)の変更。
- `ralph-common.sh` / `xreview-helpers.sh` の helper との共通化(この検出器は単体で動く必要がある)。
- 積み重ねた branch(B を A の上に作り、upstream を `origin/A` にしたもの)で A の分を除くこと。default branch との merge-base を使うので A の変更も含まれる。検査が増える方向なので許容する。
- `docs_only` の判定や言語の分類の変更。

## Assumptions

- remote の名前は `origin`(既存の実装と同じ前提)。
- `git clone` した checkout には `refs/remotes/origin/HEAD` がある。ない場合は `origin/main` 以下に落ちる。

## Affected areas

- `scripts/detect-changed-languages.sh`、`templates/base/scripts/detect-changed-languages.sh`
- `tests/test-detect-changed-languages.sh`、`tests/test-run-verify-scope.sh`
- `docs/quality/quality-gates.md`、`templates/base/docs/quality/quality-gates.md`

## Design decisions

- base は default branch との merge-base にし、`@{upstream}` は使わない。issue で決めた方針で、upstream を残す案(同名の remote branch のときだけ飛ばす)は、upstream が別の branch を指す形で同じ問題が残りうるので採らない。差分は広がる方向にしか変わらないので、検査が漏れる方向の後退はない。
- 候補の順は remote を先にする(既存の `origin/main` → `main` の順を保つ)。`origin/HEAD` は ref が実在するときだけ使う。
- Critical forks: None

## Acceptance criteria

- [ ] AC-1: upstream が HEAD と同じ push 済みの branch で、Go のファイルを変えた commit があると、既定の `./scripts/detect-changed-languages.sh` が `scope=changed`、`languages=golang` を返す(テストあり)。
- [ ] AC-2: `RALPH_VERIFY_BASE` を明示した場合の挙動は変わらない。明示した base が HEAD と同じなら `no_changes` を返す(テストあり)。既存の 9 ケースは green のまま。
- [ ] AC-3: `origin/HEAD` が main / master 以外(`trunk`)を指す repo で、default branch から分かれた Go の commit が `golang` として検出される。`origin/HEAD` が存在しない ref を指すときは `origin/main` / `main` に落ちる(どちらもテストあり)。
- [ ] AC-4: default branch の上で、未 push の commit も作業ツリーの変更もないときは、従来どおり `no_changes` を返す(テストあり)。
- [ ] AC-5: push 済みの feature branch(upstream == HEAD)で Go を変えた commit があるとき、`run-static-verify.sh` の既定(`changed`)が golang の pack を呼ぶ(`tests/test-run-verify-scope.sh` のケース)。
- [ ] AC-6: mutation: `@{upstream}` を最初に見る形に戻すと AC-1 と AC-5 のケースが落ちる。`origin/HEAD` の段を外すと AC-3 の `trunk` のケースが落ちる。ref の実在の確認を外すと、存在しない ref のケースが落ちる。
- [ ] AC-7: root と template の `detect-changed-languages.sh`、`quality-gates.md` が byte 一致。`shellcheck -S warning` で警告なし、`RALPH_VERIFY_SCOPE=full ./scripts/run-verify.sh`、`./scripts/check-sync.sh` が green。

## Implementation outline

1. Slice A(implementer、sonnet): Scope を 1 コミットで。red の証拠は AC-6。
2. pipeline: self-review → verify → test → sync-docs → cross-review → PR(`Closes #190`)。

## Verify plan

- Static analysis checks: `sh -n` と `shellcheck -S warning scripts/detect-changed-languages.sh tests/test-detect-changed-languages.sh tests/test-run-verify-scope.sh`、`RALPH_VERIFY_SCOPE=full ./scripts/run-static-verify.sh`、`./scripts/check-sync.sh`。
- Spec compliance criteria to confirm: AC-1〜AC-7 を該当行と実行結果で確認。
- Documentation drift to check: `docs/quality/quality-gates.md`(root と template)、`.claude/agents/verifier.md` / `tester.md`(scope の説明)、`docs/recipes/adding-a-language-pack.md`。
- Evidence to capture: `docs/evidence/verify-*.log`、verify report の AC 表。

## Test plan

- Unit tests: `sh tests/test-detect-changed-languages.sh`、`sh tests/test-run-verify-scope.sh`(sh と dash)。
- Integration tests: `RALPH_VERIFY_SCOPE=full ./scripts/run-verify.sh`。この branch を push した後に、worktree で `./scripts/detect-changed-languages.sh` を実行し、`no_changes` にならないことを確かめる(この変更は検出器そのものを変えるので `shared:` の full になるはず)。
- Regression tests: AC-6 の mutation。
- Edge cases: upstream なし、detached HEAD、`origin` がない repo(ローカルの main に落ちる)、main も master もない repo(`no_diff_base` で full)、`RALPH_VERIFY_BASE` に存在しない ref(従来どおり `no_merge_base:<ref>` で full)。
- Evidence to capture: test report(件数、mutation 表)。

## Risks and mitigations

- 差分が広がって changed scope の実行時間が延びる: 対象は default branch から分かれた後の変更だけで、CI の full より狭い。push の前と後で結果が変わらなくなる。
- 古い `origin/main` を base にして差分が広がる: 広がる方向なので検査が漏れることはない。必要なら `RALPH_VERIFY_BASE` で狭められる。

## Rollout or rollback notes

検出器の base の選び方とテストと文書だけ。問題があれば 1 コミットを revert する。

## Open questions

なし。

## Deviation notes

## Progress checklist

- [x] Plan reviewed
- [x] Branch created
- [ ] Implementation started
- [ ] Review artifact created
- [ ] Verification artifact created
- [ ] Test artifact created
- [ ] PR created

## Readiness checklist

- [x] 不具合を scratch の fixture で再現した
- [x] issue の「run-verify.sh も影響を受ける」は誤りで、影響は wrapper の 2 つだけと確認した
- [x] critical fork なし
- [ ] Codex plan advisory
