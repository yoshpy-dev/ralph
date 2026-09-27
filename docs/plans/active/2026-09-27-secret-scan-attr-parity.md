# secret-scan-attr-parity

- Status: In progress
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

1. **`diff.<driver>.binary` の固定**(`secret-scan.sh` の `scan_range`): `git config --get-regexp '^diff\..*\.binary$'` で見つかった key ごとに、その key をそのまま使って `-c <key>=false` を付ける(CI には driver の config がないので、どの driver も text 扱いになる)。driver 名を切り出さない(`.` を含む driver 名、例 `diff.review.driver.binary`、を取りこぼさないため)。key は git の config が返したものをそのまま渡し、git が受け付けなければ `git log` の失敗として exit 3 になる。あわせて、driver ごとの `diff.<driver>.algorithm` が `--diff-algorithm=default` で上書きされることを、algorithm で追加行が変わる fixture(#176 の AC-5 のテストと同じ「繰り返し行を越える移動」)で確かめる。上書きされないなら同様に `-c diff.<driver>.algorithm=myers` を付ける
2. **`.git/info/attributes` の検知**(`scan_range`): `git rev-parse --git-path info/attributes` のファイルが存在し、コメントと空行以外の行があれば、scan せずに exit 3 と理由(CI はこのファイルを読まない。規則を `.gitattributes` に移すか、ファイルを消す)を出す。属性の意味は解釈しない(macro や `!diff` の見落としを避けるため、規則の行があれば止める)
3. **merge 結果の属性で scan**(`secret-scan-branch.sh` と `scan_range`): `scan_range` は、範囲が HEAD で終わるときに環境変数 `RALPH_SECRET_SCAN_ATTR_SOURCE`(tree-ish)があればそれを `GIT_ATTR_SOURCE` に使う(解決できなければ exit 3。範囲が HEAD で終わらないときは無視)。`secret-scan-branch.sh` は次の順で属性の読み元を決める。(a) merge-base と base の先端の間で `.gitattributes`(root と下位ディレクトリのすべて)に変更がなければ、merge 結果の属性は HEAD のものと同じなので、従来どおり HEAD の属性で scan する(merge-tree は使わない)。(b) 変更があるときだけ、ローカルの merge driver の config(`merge.default`、`merge.<name>.driver`)がないことを確かめてから `git merge-tree --write-tree <base_ref_full> HEAD` を実行し、rc 0 ならその tree を渡す。(c) rc 1(衝突)のときは HEAD の属性で scan し、その旨を通知する(衝突があると CI の `pull_request` は走らないので、そこで止めない)。(d) それ以外、つまりローカルの merge driver の config がある、merge-tree がそれ以外の rc で失敗した、git が `--write-tree` に対応しない、git が `GIT_ATTR_SOURCE` に対応しない(2.41 未満)場合は、CI と同じ属性を保証できないので `--strict` では exit 3 と理由を出し、既定の mode では通知して HEAD の属性で scan する
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
- **Codex plan advisory(2026-09-27、HIGH 3、ユーザー決定: 対応案で plan を更新)**: (1) driver 名を `[^.]+` で切り出すと `.` を含む driver 名を取りこぼす → key 全体をそのまま `-c` に渡す。(2) merge-tree はローカルの merge driver(`merge.default`、`merge.<name>.driver`)を実行するので merge 結果の `.gitattributes` が CI と違いうる → base が分岐後に `.gitattributes` を変えていなければ merge-tree を使わず HEAD の属性(同じ結果)で済ませ、変えているときだけ merge-tree を使い、その前にローカルの merge driver の config があれば strict で exit 3。(3) merge-tree の非 0 をすべて衝突扱いにすると権限不足などの失敗でも clean を返せる → rc 1(衝突)だけを通知付きの fallback にし、それ以外の失敗・非対応の git は strict で exit 3
- Critical forks: None

## Acceptance criteria

- [ ] AC-1: コミット済みの `.gitattributes` が `diff=<driver>` を指し、ローカルに `diff.<driver>.binary=true` がある repo で、range scan が fixture を検出する(exit 1)。driver が 2 つあっても両方固定される。driver 名が `.` を含む(`diff=review.driver`、config の key は `diff.review.driver.binary`)場合も固定される
- [ ] AC-2: driver ごとの `diff.<driver>.algorithm`(histogram / patience)があっても、range scan の結果は既定の設定と同じ(algorithm で追加行が変わる fixture で確認)
- [ ] AC-3: `.git/info/attributes` に規則の行がある repo では、range scan が exit 3 と理由を出し、clean を出さない。コメントと空行だけなら scan する
- [ ] AC-4: `secret-scan-branch.sh --strict` は、base が分岐後に `.gitattributes` で `-diff` を足した branch で fixture を検出せず(CI と同じ、exit 0 clean)、base が `-diff` を外した branch では検出する(exit 1)。どちらも merge 結果の属性で scan したことが分かる。base が `.gitattributes` を変えていない branch では merge-tree を呼ばず、従来どおり HEAD の属性で scan する(下位ディレクトリの `.gitattributes` の変更も「変えた」に数える)
- [ ] AC-5: base が `.gitattributes` を変え、かつ base と HEAD が衝突する branch では、`secret-scan-branch.sh --strict` は HEAD の属性で scan し、通知を 1 行出す(exit は scan の結果どおり)
- [ ] AC-5b: base が `.gitattributes` を変えていて、ローカルに `merge.default` または `merge.<name>.driver` の config がある repo では、`--strict` は exit 3 と理由を出し clean を出さない。既定の mode は通知して HEAD の属性で scan する
- [ ] AC-5c: base が `.gitattributes` を変えていて、merge-tree が rc 1 以外で失敗する(例: object の書き込み先を読み取り専用にする)、または git が `--write-tree` か `GIT_ATTR_SOURCE` に対応しない(stub の git で模擬してよい)場合、`--strict` は exit 3 と理由を出す。既定の mode は通知して HEAD の属性で scan する
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
- Edge cases: driver 名に `.` を含む config の key、info/attributes が symlink、merge-tree が同じ tree を返す(base がすでに HEAD に含まれる)、下位ディレクトリの `.gitattributes` だけが base で変わった、`GIT_ATTR_SOURCE` を無視する古い git(stub)
- Evidence to capture: red/green(固定・検知・merge-tree を 1 つずつ外すと対応するテストが落ちる)

## Risks and mitigations

- `.git/info/attributes` を使っている環境では、ローカルの scan が止まるようになる(理由と対処を出す。CI と同じ結果にするための意図した挙動)
- merge-tree は base の先端を読むので、`git fetch` が古いと CI と違う merge になる。`secret-scan-branch.sh` は既存の base の解決(origin 優先)をそのまま使う
- `-c <key>=false` は config が返した key をそのまま使うので、取りこぼしは起きない。git が key を受け付けなければ `git log` が失敗して exit 3
- merge-tree を使うのは base が `.gitattributes` を変えたときだけなので、通常の branch では挙動が変わらない

## Rollout or rollback notes

- scanner 2 本とテスト、文書に閉じる。revert すれば #176 の状態に戻る

## Open questions

- なし

## Deviation notes

- 2026-09-27 plan: Codex plan advisory の HIGH 3 件を反映(Design decisions を参照)。ユーザー決定(AskUserQuestion): 対応案で plan を更新
- 2026-09-27 work: Slice A(7e0c529)、B(beb19f1)、C(ad8c475)は implementer に委譲(scanner は関門なので opus)。A: driver の binary は `git config --name-only --get-regexp` で列挙した key をそのまま `-c <key>=false`(`=` を含む key は `--config-env` で。git 2.31 未満は exit 3)。driver ごとの algorithm は `--diff-algorithm=default` が上書きすることを fixture で確認したので固定は足さずテストのみ。`.git/info/attributes` は git の parser と同じ規則(空行と `#` 以外)で規則の行があれば exit 3、読めなければ exit 3、dangling symlink は scan。`RALPH_SECRET_SCAN_ATTR_SOURCE` は `^{tree}` で解決し、範囲が HEAD で終わるときだけ使う。ヘッダーに Environment 節。B: merge-base と base の間で `:(top,glob)**/.gitattributes` に変更がなければ従来どおり HEAD の属性。変更があれば、ローカルの merge driver の config → git 2.41 未満 → merge-tree の順に判定し、rc 0 なら tree を渡して通知、rc 1(衝突)は HEAD の属性と通知、それ以外と `--write-tree` 非対応は strict で exit 3(既定は通知)。この diff 自体の失敗も strict で exit 3(逸脱、fail-closed)。CI の実行では merge-base が base の先端なので merge-tree は走らない。C: quality-gates 2 コピー、tech-debt の 2 行(plan の参照は archive のパスに)、`/pr` skill の exit 3 の句(4 面。1 行形式なので折り返しはしない)。テストは 74 → 123 件、108 → 163 件。修正前の scanner で新規 19 件、branch script で 36 件が落ちる。全履歴の比較で main と一致(219,578 行)。orchestrator は HEAD 一致・porcelain 空・差分・template の byte 一致、probe(3 経路とも期待どおり)、テスト 3 本、`run-verify.sh`、strict scan を確認。範囲外の発見: merge-tree に対する `merge.renames` などの設定は未固定(probe で差は出ず、tech-debt に未再現として記録)。tech-debt の別の 2 行が archive 済み plan の active のパスを参照している(sync-docs で直す)
## Progress checklist

- [x] Plan reviewed
- [x] Branch created
- [x] Implementation started
- [ ] Review artifact created
- [ ] Verification artifact created
- [ ] Test artifact created
- [ ] PR created

## Readiness checklist

- [x] 3 つの差を fixture repo で再現し、固定できるかどうかを確かめた(orchestrator)
- [x] critical fork なし
- [x] Codex plan advisory(HIGH 3 件、対応案で plan を更新)
- [x] AC は hermetic な fixture repo の shell テストで決定的に確認できる
