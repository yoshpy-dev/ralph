# secret-scan-attr-parity

- Status: Draft
- Owner: Claude Code
- Date: 2026-09-27
- Related request: #176(PR #179)で `scripts/secret-scan.sh` の range scan はローカルの git 設定を git の既定に固定し、CI と同じ追加行を読むようになったが、属性(attributes)の読み元に固定できない差が 3 つ残った(`docs/tech-debt/README.md` に記録)。いずれも「ローカルの `--strict` は clean、CI は検出」の向き。(1) `.git/info/attributes` の `-diff` / `binary`(config の key ではないので `-c` でも `GIT_ATTR_SOURCE` でも消えない)。(2) コミット済みの `.gitattributes` が指す diff driver に、ローカルで `diff.<driver>.binary=true` があると binary 扱い。(3) CI の `pull_request` は PR の merge commit を checkout するので、branch の作成後に base 側で `.gitattributes` が変わると CI とローカルの結果がずれる(issue #181)
- Related issue: 181
- Type: fix
- Branch: fix/secret-scan-attr-parity

## Objective

secret scan の属性の読み元について、固定できる差は固定し、固定できない差は clean と言わずに exit 3(could not scan)で止める。`/pr` の strict scan は、CI が checkout する PR の merge commit と同じ属性で scan する。

## 調査で確認したこと(2026-09-27、fixture repo、git 2.49)

| 差 | 結果 |
|---|---|
| (2) ローカルの `diff.<driver>.binary=true` | `-c diff.<driver>.binary=false` を付けると追加行が見える。固定できる |
| (1) `.git/info/attributes` の `-diff` | `GIT_ATTR_SOURCE=HEAD` を付けても隠れたまま。無効化の手段がない。検知するしかない |
| (3) base が分岐後に `-diff` を足した / 外した | `git merge-tree --write-tree <base> HEAD` の tree を `GIT_ATTR_SOURCE` に渡すと、両向きとも CI と同じ結果になる。衝突があると rc 1(tree は出る)。git 2.38 以降 |

## Scope

1. **`diff.<driver>.binary` の固定**(`secret-scan.sh` の `scan_range`): `git config --get-regexp '^diff\.[^.]+\.binary$'` で見つかった driver ごとに `-c diff.<driver>.binary=false` を付ける(CI には driver の config がないので、どの driver も text 扱いになる)。driver 名は config の key から取り、`-c` の引数として安全な形に限る(想定外の文字を含む key は無視せず exit 3)。あわせて、driver ごとの `diff.<driver>.algorithm` が `--diff-algorithm=default` で上書きされることを、algorithm で追加行が変わる fixture(#176 の AC-5 のテストと同じ「繰り返し行を越える移動」)で確かめる。上書きされないなら同様に `-c diff.<driver>.algorithm=myers` を付ける
2. **`.git/info/attributes` の検知**(`scan_range`): `git rev-parse --git-path info/attributes` のファイルが存在し、コメントと空行以外の行があれば、scan せずに exit 3 と理由(CI はこのファイルを読まない。規則を `.gitattributes` に移すか、ファイルを消す)を出す。属性の意味は解釈しない(macro や `!diff` の見落としを避けるため、規則の行があれば止める)
3. **merge 結果の属性で scan**(`secret-scan-branch.sh` と `scan_range`): `scan_range` は、範囲が HEAD で終わるときに環境変数 `RALPH_SECRET_SCAN_ATTR_SOURCE`(tree-ish)があればそれを `GIT_ATTR_SOURCE` に使う(解決できなければ exit 3。範囲が HEAD で終わらないときは無視)。`secret-scan-branch.sh` は `git merge-tree --write-tree <base_ref_full> HEAD` を試み、rc 0 ならその tree を渡す。rc が 0 でない(衝突)か、`--write-tree` に対応しない git なら、HEAD の属性のまま scan し、その旨を 1 行通知する(衝突があると CI の `pull_request` は走らないので、そこで止めない)
4. 文書: `secret-scan.sh` のヘッダー(固定するもの、検知して止めるもの、環境変数)、`secret-scan-branch.sh` のヘッダー(merge の属性)、`docs/quality/quality-gates.md`(2 コピー)の例外の記述、`docs/tech-debt/README.md` の該当 2 行(解消した項目を消し、残る差があれば書き直す)
5. `templates/base/scripts/` のコピーを byte 一致に保つ

## Non-goals

- CI と共通の穴(コミット済みの `-diff` / `binary`、バイナリファイル、merge commit の diff)
- `.git/info/attributes` の属性の意味の解釈(規則があれば止める)
- `secret-scan.sh --range` を CI が呼ぶ経路の挙動の変更(CI は merge commit を checkout しているので、HEAD の属性がそのまま CI の属性)
- `secret-scan-branch.sh` 自身の git 呼び出しの replace ref(別の tech-debt。この plan の範囲外)

## Assumptions

- CI の runner の git は `merge-tree --write-tree` に対応する(ubuntu-latest は 2.43 以降)。ローカルは 2.49
- `GIT_ATTR_SOURCE` は tree の id を受け付ける(#176 で commit id を渡して確認済み。tree も tree-ish として受け付ける想定。実装時に確かめる)
- 既定の設定での range scan の結果は変えない(回帰の基準)

## Affected areas

- `scripts/secret-scan.sh`、`templates/base/scripts/secret-scan.sh`
- `scripts/secret-scan-branch.sh`、`templates/base/scripts/secret-scan-branch.sh`
- `tests/test-secret-scan.sh`、`tests/test-secret-scan-branch.sh`
- `docs/quality/quality-gates.md`、`templates/base/docs/quality/quality-gates.md`、`docs/tech-debt/README.md`

## Design decisions

- (1) は検知して exit 3。属性の意味を解釈しない(fail-closed で単純に)
- (3) の読み元は環境変数で渡す(`RALPH_SECRET_ALLOWLIST` と同じ形。CI の呼び出しは変えない)
- 衝突で merge tree が作れないときは HEAD の属性で scan して通知(そこで exit 3 にすると、CI が走らない PR をローカルだけが止める)
- Critical forks: None

## Acceptance criteria

- [ ] AC-1: コミット済みの `.gitattributes` が `diff=<driver>` を指し、ローカルに `diff.<driver>.binary=true` がある repo で、range scan が fixture を検出する(exit 1)。driver が 2 つあっても両方固定される
- [ ] AC-2: driver ごとの `diff.<driver>.algorithm`(histogram / patience)があっても、range scan の結果は既定の設定と同じ(algorithm で追加行が変わる fixture で確認)
- [ ] AC-3: `.git/info/attributes` に規則の行がある repo では、range scan が exit 3 と理由を出し、clean を出さない。コメントと空行だけなら scan する
- [ ] AC-4: `secret-scan-branch.sh --strict` は、base が分岐後に `.gitattributes` で `-diff` を足した branch で fixture を検出せず(CI と同じ、exit 0 clean)、base が `-diff` を外した branch では検出する(exit 1)。どちらも merge 結果の属性で scan したことが分かる
- [ ] AC-5: base と HEAD が衝突する branch では、`secret-scan-branch.sh --strict` は HEAD の属性で scan し、通知を 1 行出す(exit は scan の結果どおり)
- [ ] AC-6: `RALPH_SECRET_SCAN_ATTR_SOURCE` が tree-ish に解決できないとき、range scan は exit 3。範囲が HEAD で終わらないときは無視される
- [ ] AC-7: `.git/info/attributes` の規則がある repo では `secret-scan-branch.sh --strict` が「scanner failed with exit 3」で止まり clean を出さない
- [ ] AC-8: 既定の設定での range scan の結果は変わらない(この repo の全履歴で、取り出す追加行が main と一致)
- [ ] AC-9: 文書(ヘッダー、quality-gates 2 コピー、tech-debt)が最終の挙動と一致し、`scripts/` と `templates/base/scripts/` が byte 一致。`./scripts/run-verify.sh` green

## Implementation outline

1. Slice A: `secret-scan.sh`(driver の固定、info/attributes の検知、`RALPH_SECRET_SCAN_ATTR_SOURCE`)とテスト(AC-1〜3、AC-6、AC-8)
2. Slice B: `secret-scan-branch.sh`(merge-tree、通知)とテスト(AC-4、5、7)
3. Slice C: 文書(AC-9)

## Verify plan

- Static analysis checks: `./scripts/run-static-verify.sh`(shellcheck)、`sh -n`
- Spec compliance criteria to confirm: AC-1〜AC-9、Non-goals(CI の呼び出しの経路が不変)
- Documentation drift to check: 2 本のヘッダー、quality-gates 2 コピー、tech-debt、`/pr` skill の exit 3 の説明(変更が要るか)
- Evidence to capture: 修正前の scanner で 3 つの差が見逃すこと、修正後に検出か exit 3 になること

## Test plan

- Unit tests: `tests/test-secret-scan.sh` に driver の binary(1 つ・2 つ)、driver の algorithm、info/attributes(規則あり・コメントのみ)、環境変数の解決失敗と無視
- Integration tests: `tests/test-secret-scan-branch.sh` に base の `.gitattributes` の変更(両向き)、衝突、info/attributes
- Regression tests: 既存の 3 ファイルすべて、全履歴の比較
- Edge cases: driver 名に `.` や空白を含む config の key、info/attributes が symlink、merge-tree が同じ tree を返す(base がすでに HEAD に含まれる)
- Evidence to capture: red/green(固定・検知・merge-tree を 1 つずつ外すと対応するテストが落ちる)

## Risks and mitigations

- `.git/info/attributes` を使っている環境では、ローカルの scan が止まるようになる(理由と対処を出す。CI と同じ結果にするための意図した挙動)
- merge-tree は base の先端を読むので、`git fetch` が古いと CI と違う merge になる。`secret-scan-branch.sh` は既存の base の解決(origin 優先)をそのまま使う
- `-c diff.<driver>.binary=false` の driver 名の扱いに漏れがあると、その driver だけ固定されない。key の形を検査し、想定外なら exit 3

## Rollout or rollback notes

- scanner 2 本とテスト、文書に閉じる。revert すれば #176 の状態に戻る

## Open questions

- なし

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

- [x] 3 つの差を fixture repo で再現し、固定できるかどうかを確かめた(orchestrator)
- [x] critical fork なし
- [x] AC は hermetic な fixture repo の shell テストで決定的に確認できる
