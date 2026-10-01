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
  - `RALPH_VERIFY_BASE` が空のときの base を、次の順で最初に実在する ref に決める。`@{upstream}` そのものは base に使わない(upstream は push 先で、base ではない)。
    1. `origin` の `refs/remotes/origin/HEAD` の指す branch(指す先の ref が実在するときだけ)→ `origin/main` → `origin/master`
    2. 今の branch の追跡先の remote(`git config branch.<branch>.remote`。空、`.`、`origin` のとき、と detached HEAD のときは飛ばす)の HEAD の指す branch → `<remote>/main` → `<remote>/master`
    3. ローカルの `main` → `master`
  - `RALPH_VERIFY_BASE` の明示は従来どおり最優先。
  - ヘッダのコメントに base の決め方を書く。
- `tests/test-detect-changed-languages.sh`: 次のケースを足す。bare の remote を使う。
  - push 済みの branch(upstream == HEAD)で Go の commit → `languages=golang`。
  - 同じ状態で `RALPH_VERIFY_BASE=origin/feature`(== HEAD)→ `no_changes`(明示が勝つ)。
  - default branch が main / master ではない(`trunk`)remote で、`origin/HEAD` が `origin/trunk` を指す → `languages=golang`。
  - `origin/HEAD` が存在しない ref を指す → `origin/main` / `main` に落ちて `languages=golang`。
  - default branch の上で、未 push の commit も作業ツリーの変更もない → `no_changes`(従来どおり)。
  - remote が `central` だけで、`main` が `central/main` を追跡し、main の上に未 push の Go の commit がある → `languages=golang`(Codex plan advisory の反例。`origin` を前提にすると `no_changes` になる)。
- `tests/test-run-verify-scope.sh`: 次の 2 つで、`run-static-verify.sh` の既定が golang の pack を呼ぶケースを足す(end-to-end)。
  - push 済みの feature branch(upstream == HEAD)で Go を変えた commit がある。
  - `central` を追跡する main の上に未 push の Go の commit がある。
- `docs/quality/quality-gates.md`(+ template): changed scope の段落に、差分の base(`RALPH_VERIFY_BASE`、なければ default branch との merge-base)を 1 文で足す。

## Non-goals

- `run-verify.sh` の出力や既定の scope の変更(既定の `full`、wrapper の `changed` はそのまま)。
- CI(`.github/workflows/verify.yml`)の変更。
- `ralph-common.sh` / `xreview-helpers.sh` の helper との共通化(この検出器は単体で動く必要がある)。
- 積み重ねた branch(B を A の上に作り、upstream を `origin/A` にしたもの)で A の分を除くこと。default branch との merge-base を使うので A の変更も含まれる。検査が増える方向なので許容する。
- `docs_only` の判定や言語の分類の変更。

## Assumptions

- remote の名前は `origin` とは限らない。今の実装も upstream 経由で他の名前の remote を扱っているので、追跡先の remote を 2 番目の候補として見る(Codex plan advisory)。
- `git clone` した checkout には `refs/remotes/origin/HEAD` がある。ない場合は `origin/main` 以下に落ちる。

## Affected areas

- `scripts/detect-changed-languages.sh`、`templates/base/scripts/detect-changed-languages.sh`
- `tests/test-detect-changed-languages.sh`、`tests/test-run-verify-scope.sh`
- `docs/quality/quality-gates.md`、`templates/base/docs/quality/quality-gates.md`

## Design decisions

- base は default branch との merge-base にし、`@{upstream}` は使わない。issue で決めた方針で、upstream を残す案(同名の remote branch のときだけ飛ばす)は、upstream が別の branch を指す形で同じ問題が残りうるので採らない。差分は広がる方向にしか変わらないので、検査が漏れる方向の後退はない。
- 候補の順は remote を先にし(既存の `origin/main` → `main` の順を保つ)、`origin` を追跡先の remote より先に見る。fork に push する形(`origin` が本家、upstream が `fork/feature`)で、古いかもしれない fork の main を base にしないため。`origin` がなければ追跡先の remote の default branch を使う。どの remote でも、`HEAD` の指す先は ref が実在するときだけ使う。
- ローカルの `main` / `master` は最後の候補にする。default branch の上でローカルの main を base にすると merge-base が HEAD になり、未 push の commit が差分から消えるため。
- Critical forks: None

## Acceptance criteria

- [ ] AC-1: upstream が HEAD と同じ push 済みの branch で、Go のファイルを変えた commit があると、既定の `./scripts/detect-changed-languages.sh` が `scope=changed`、`languages=golang` を返す(テストあり)。
- [ ] AC-2: `RALPH_VERIFY_BASE` を明示した場合の挙動は変わらない。明示した base が HEAD と同じなら `no_changes` を返す(テストあり)。既存の 9 ケースは green のまま。
- [ ] AC-3: `origin/HEAD` が main / master 以外(`trunk`)を指す repo で、default branch から分かれた Go の commit が `golang` として検出される。`origin/HEAD` が存在しない ref を指すときは `origin/main` / `main` に落ちる(どちらもテストあり)。
- [ ] AC-4: default branch の上で、未 push の commit も作業ツリーの変更もないときは、従来どおり `no_changes` を返す(テストあり)。
- [ ] AC-8: remote が `central` だけで `main` が `central/main` を追跡し、main の上に未 push の Go の commit があるとき、既定の検出器が `languages=golang` を返し、`run-static-verify.sh` の既定が golang の pack を呼ぶ(どちらもテストあり)。
- [ ] AC-5: push 済みの feature branch(upstream == HEAD)で Go を変えた commit があるとき、`run-static-verify.sh` の既定(`changed`)が golang の pack を呼ぶ(`tests/test-run-verify-scope.sh` のケース)。
- [ ] AC-6: mutation: `@{upstream}` を最初に見る形に戻すと AC-1 と AC-5 のケースが落ちる。`origin/HEAD` の段を外すと AC-3 の `trunk` のケースが落ちる。ref の実在の確認を外すと、存在しない ref のケースが落ちる。追跡先の remote の段を外すと AC-8 のケースが落ちる。
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
- Edge cases: upstream なし、detached HEAD(追跡先の remote の段を飛ばす)、`branch.<branch>.remote` が `.`(ローカルの branch を追跡)、`origin` がない repo(ローカルの main に落ちる)、main も master もない repo(`no_diff_base` で full)、`RALPH_VERIFY_BASE` に存在しない ref(従来どおり `no_merge_base:<ref>` で full)。
- Evidence to capture: test report(件数、mutation 表)。

## Risks and mitigations

- 差分が広がって changed scope の実行時間が延びる: 対象は default branch から分かれた後の変更だけで、CI の full より狭い。push の前と後で結果が変わらなくなる。
- 古い `origin/main` を base にして差分が広がる: 広がる方向なので検査が漏れることはない。必要なら `RALPH_VERIFY_BASE` で狭められる。

## Rollout or rollback notes

検出器の base の選び方とテストと文書だけ。問題があれば 1 コミットを revert する。

## Open questions

なし。

## Deviation notes

- 2026-10-01 plan: Codex plan advisory(gpt-6-astra、xhigh、watchdog の 1 行、`codex rc=0`、`-o` 1225 バイト)は MEDIUM 1: `origin` 以外の remote で、default branch の上の未 push の commit が検査から漏れる(ローカルの main を base にすると merge-base が HEAD になる)。scratch で確認: `central/main` を追跡する main の上の未 push の Go の commit は、今の実装では `golang`、下書きの plan では `no_changes`、`central/main` を base にすれば `golang`。ユーザー決定: 対応案で plan を更新。候補に追跡先の remote の段を足し、AC-8 とテスト 2 件と mutation 1 種を追加した
- 2026-10-01 work: Slice A は implementer(sonnet)に委譲(5bf0d0ad、6 ファイル、+358 / -16、push 済み)。`remote_default_ref` で remote ごとに「HEAD の指す先(実在するときだけ)→ main → master」を返し、origin → 追跡先の remote → ローカルの main / master の順に見る。detached HEAD の判定は `--short` を使わず `git symbolic-ref --quiet HEAD` から `refs/heads/` を外す(同名の tag があると `heads/<name>` が返りうるため)。検出器のテストに 11 ケース(10〜20)を追加した。push 済み、明示の base、存在しない明示の base、`trunk`、存在しない ref を指す HEAD、default branch 上で変更なし、default branch 上の未 push の commit、`central`、detached HEAD、remote `.`、base なしの各ケースで、48 / 48。scope のテストに 2 ケースを追加して 19 / 19。sh と dash の両方で同じ結果。2 本のテストの冒頭で `RALPH_VERIFY_BASE`(scope 側は `RALPH_VERIFY_SCOPE` も)を unset する。red: (a) `@{upstream}` を先頭に戻す → push 済みのケースが落ちる(fixture が push 済みなので `trunk` と存在しない ref のケースも落ちる)、(b) HEAD の段を外す → `trunk`、(c) 実在の確認を外す → 存在しない ref のケース、(d) 追跡先の remote の段を外す → `central` のケースが検出器と scope の両方で落ちる。push 後の検出器は `scope=full reason=shared:scripts/detect-changed-languages.sh`(`no_changes` ではない)。`quality-gates.md` は root と template で同じ 1 文を足した(全体の差分は check-sync の KNOWN_DIFF のまま)。implementer は「2 本のテストは run-verify に配線されていない」と報告したが誤りで、`verify.local.sh` の test モードは実行権限のある `tests/test-*.sh` を全部走らせる。run-verify の log でも 48 / 48 と 19 / 19 を確認した。orchestrator も 48 / 48(sh)、19 / 19(dash)、cmp を確認
- 2026-10-01 self-review(cycle 1、fff03042): CRITICAL 0 / HIGH 0 / MEDIUM 1 / LOW 5、merge 可。M-1: 存在を確かめた `refs/remotes/<r>/<b>` / `refs/heads/main` ではなく短い名前を merge-base に渡すので、同名の tag やローカルの branch(`origin/main` という名前の branch など)が base になる。upstream のある経路では旧版より後退。L-1: 追跡先の remote に ref が 1 つもないまま main の上にいると、ローカルの main == HEAD になって `no_changes`。L-2: コメントの「@{upstream} is never the base」が字面どおりでない、HEAD の指す先が remote の外のときに無視する条件が書かれていない。L-3: `quality-gates.md` の文が、明示の base も merge-base を通ることを読み取れない。L-4: fixture が global / system の git 設定から切り離されていない。L-5: ケース 18 と 19 の名前が、fixture では確かめられない経路を謳っている。全件を in-cycle で修正する。L-1 は reviewer の「別 issue」案ではなく、範囲を絞って塞ぐ(orchestrator 判断): 今の branch が `.` 以外の remote を追跡していて、どの remote にも default branch の ref がなく、ローカルの段で選んだ base が今の branch そのものなら、`no_remote_default:<remote>` で full にする。remote のない repo の main(ケース 1〜9)は変わらない

## Progress checklist

- [x] Plan reviewed
- [x] Branch created
- [x] Implementation started
- [x] Review artifact created
- [ ] Verification artifact created
- [ ] Test artifact created
- [ ] PR created

## Readiness checklist

- [x] 不具合を scratch の fixture で再現した
- [x] issue の「run-verify.sh も影響を受ける」は誤りで、影響は wrapper の 2 つだけと確認した
- [x] critical fork なし
- [x] Codex plan advisory(MEDIUM 1、対応案で plan を更新)
