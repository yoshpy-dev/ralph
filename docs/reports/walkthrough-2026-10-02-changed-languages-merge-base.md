# Walkthrough: changed-languages-merge-base (#190)

- Date: 2026-10-02
- Plan: docs/plans/active/2026-10-01-changed-languages-merge-base.md
- Branch: fix/changed-languages-merge-base(base main a3adc13)
- Diff: plan・報告・insight を除くと 6 files、+751 / -22。うち 563 行が 2 本のテスト

## 何を直したか

`scripts/detect-changed-languages.sh` は、`RALPH_VERIFY_BASE` が空のとき `@{upstream}` を base にしていた。コミットのたびに push する運用では、作業 branch の upstream が HEAD と同じコミットを指す。そのため差分が空になり、`scope=changed reason=no_changes docs_only=true` を返していた。`changed` を既定にする `run-static-verify.sh`(`/verify`)と `run-test.sh`(`/test`)は、Go を変えた branch でも言語 pack を 1 つも走らせないまま通っていた。`run-verify.sh` の既定は `full` なので、この不具合の影響は受けない(issue の記述を plan で訂正した)。

この PR の変更点は次のとおり。

- base は default branch との merge-base にした。`@{upstream}` は読まない。
- 候補を探す順番は、origin の default branch、追跡先の remote の default branch、ローカルの main / master。
- merge-base には完全な ref 名を渡す。
- base が今の branch そのものになり、しかも upstream が設定されているときは、full に倒す。
- URL の形をした remote の値は出力に出さない。

## 読む順

1. `scripts/detect-changed-languages.sh`(template と byte 一致)
   - 先頭のコメント: base の決め方、full に倒す条件、残る制約(remote も upstream もない main の上では、commit 済みの変更が見えない)。
   - `remote_default_ref`: remote ごとに次の順で探し、完全な ref 名を返す。どれもなければ何も返さない。
     - `refs/remotes/<remote>/HEAD` の指す先。そのリモートの名前空間の中にあって、実在するときだけ使う。
     - `refs/remotes/<remote>/main`
     - `refs/remotes/<remote>/master`
   - base を決めるところ: 3 つの変数を使い分ける。
     - `configured_remote`: `branch.<b>.remote` の生の値。guard の判定に使う。
     - `tracked_remote`: step 2 が見る remote 名。設定済みの remote 名のときだけ入る。
     - `remote_label`: reason に出してよい形。`.`、remote 名、固定の `url` のどれか。
   - 候補の順は次のとおり。
     1. origin
     2. 追跡先の remote(origin と違う名前のときだけ)
     3. ローカルの main / master
   - guard: 3 の結果が今の branch そのもので、`configured_remote` が空でなければ、`no_remote_default:<remote_label>` で full にする。
2. `tests/test-detect-changed-languages.sh`: 既存の 9 ケースはそのままで、ケース 10〜35 を足した(83 assertion)。
   - push 済みの branch(10)。
   - base を明示した場合。存在する base(11)と、存在しない base(12)。
   - `origin/HEAD` の扱い。`trunk` を指す(13)、存在しない ref を指す(14)、remote の外を指す(31)。
   - default branch の上にいる場合。未 push の commit がない(15)、ある(16)。
   - origin ではない remote(17)。
   - detached HEAD(18)と remote `.`(19)。
   - base がない場合(20)。
   - 同じ短い名前を持つ branch や tag(21、22)。
   - 未 fetch の remote(23〜25)。
   - origin を古い fork より先に見る(26)。
   - master の段と、main と master の順番(27〜30)。
   - ローカルの branch を追跡する main(32)、URL の値(33)、設定の消えた remote(34)、`.` と `/` を含む remote 名(35)。
   - remote を使う fixture は、bare の remote をテストの一時ディレクトリに作る。テストの冒頭で HOME と `GIT_CONFIG_*` を固定し、開発者の git の設定から切り離している。
3. `tests/test-run-verify-scope.sh`: 本物の `run-static-verify.sh` を使うケースを 2 つ足した。push 済みの feature branch と、`central` を追跡する main の上の未 push の commit。どちらも golang の pack が呼ばれることを確かめる。
4. `docs/quality/quality-gates.md`(+ template): changed scope の差分の base と、残る制約を 2 文で書いた。

## 注意して見てほしいところ

- **差分が広がる方向への変更:** 以前は push 済みの branch では差分が空だった。今後は default branch から分かれた後の変更がすべて入る。検査が漏れる方向には変わらない。積み重ねた branch(upstream が別の feature branch)では、下の branch の変更も含まれる。
- **full に倒す条件:** 3 つがそろったときだけ倒す。
  - `branch.<b>.remote` が設定されている(`.` と URL を含む)
  - どの remote にも default branch の ref がない
  - ローカルの base が今の branch そのもの

  feature branch、detached HEAD、ref のある origin を追跡する main では倒れない。reviewer がこのことを確かめた。remote も upstream もない repo の main(既存のケース 1〜9)は、以前と同じ結果になる。
- **URL の扱い:** `git push -u <URL> main` を実行すると、`branch.main.remote` に URL がそのまま入る。URL は userinfo を含みうるので、reason には固定の `url` だけを出す。step 2 も、設定済みの remote 名ではない値は git のコマンドに渡さない。git のエラー文に ref 名が出るのを避けるため。
- **この branch 自身での見え方:** この PR は検出器そのものを変えるので、この branch では `shared:scripts/detect-changed-languages.sh` の full になる。changed scope が実際にどう選ばれるかは、次に検出器以外を変える branch で見ることになる。

## 検証

| 項目 | 結果 |
|---|---|
| `tests/test-detect-changed-languages.sh`(sh、dash) | 83 / 83 |
| `tests/test-run-verify-scope.sh`(sh、dash) | 19 / 19 |
| `RALPH_VERIFY_SCOPE=full ./scripts/run-verify.sh` / `run-test.sh` / `run-static-verify.sh` | green |
| shellcheck `-S warning`、check-sync(159 IDENTICAL)、purity、cmp | clean / pass |
| end-to-end(本物の wrapper と stub の pack、push 済みの branch) | 新版は golang の pack を呼ぶ。旧版は 1 つも呼ばない(#190 の再現) |
| URL の値の probe(実行時に組み立てた 8 通り) | どれも `no_remote_default:url`。出力、ログ、state に URL は出ない |
| mutation(slice ごとに計 30 種以上) | 等価な 2 種と、reason のラベルだけが変わる 1 種を除いて、すべて該当ケースで落ちる |
| 古い git、Linux の dash、busybox | 未確認(macOS の git 2.49 と dash で確認) |

## pipeline の履歴

- plan:
  - scratch の fixture で不具合を再現した。issue の「`run-verify.sh` も影響を受ける」は誤りだったので訂正した。
  - Codex advisory の MEDIUM 1 件(origin 以外の remote で、default branch の上の未 push の commit が漏れる)を受け、ユーザーの決定で、追跡先の remote の段と AC-8 を足した。
- cycle 1:
  - Slice A で base の決め方を入れた。
  - self-review は MEDIUM 1 / LOW 5。M-1 は、短い名前を merge-base に渡すと同じ名前の tag や branch に負ける問題。L-1 は、未 fetch の remote を追跡する main が `no_changes` になる問題。Slice B で全件を直した。
  - verify は pass(LOW 1 は V-1、文書で扱う)。
  - test は pass。塞げる穴 4 件と、今回の変更で生じた後退 O-1(ローカルの branch を追跡する main)を、Slice C で塞いだ。
  - Slice B と C の差分を reviewer が見直し、LOW 2 件を指摘した。A-1 は reason に URL が出うる問題。Slice D で直した。
  - Slice C と D の差分を tester が見直し、塞げる穴 2 件を指摘した。Slice E のテストで塞いだ。
  - sync-docs で V-1 の 1 文を足した。cross-review は指摘 0 件。

## 残る gap

- remote も upstream もない repo の main の上(またはその先端で detached HEAD)では、commit 済みの変更が見えない。以前と同じ挙動で、文書に書いた。`RALPH_VERIFY_BASE` か `RALPH_VERIFY_SCOPE=full` で補える。
- 古い git、Linux の dash、busybox の sh では確かめていない。CI(ubuntu)で 2 本のテストが走る。
