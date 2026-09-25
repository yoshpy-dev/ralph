# secret-scan-git-config

- Status: In progress
- Owner: Claude Code
- Date: 2026-09-25
- Related request: #169(PR #177)の最終 cross-review(cycle 2、cap 到達)で残った 2 件(`docs/reports/cross-review-triage-local-branch-secret-scan.md` の AR-3 / AR-4)。AR-3: git の設定に `color.ui=always` があると scanner の `git log -p` の追加行が色のエスケープで始まり、`+` で始まる行だけを読む parser が読み飛ばす。AR-4: symlink の `.gitallowed` のリンク先の末尾改行がコマンド置換で落ち、別のファイルに解決される。plan 作成時の調査で同じ種類の見落としをさらに 4 つ再現し、ユーザー判断(AskUserQuestion)ですべてこの issue で直す
- Related issue: 176
- Type: fix
- Branch: fix/secret-scan-git-config

## Objective

`scripts/secret-scan.sh` の range scan と staged scan が、ローカルの git 設定やファイル名の引用に左右されず、既定の設定の CI と同じ行を scan する。`scripts/secret-scan-branch.sh` は symlink の `.gitallowed` のリンク先をバイト列どおりに扱う。

## 調査で再現した見落とし(2026-09-25、fixture repo)

| 経路 | 条件 | 結果 |
|---|---|---|
| range | `color.ui=always`(repo の設定) | 追加行を読み飛ばして clean(AR-3、Codex が再現、orchestrator も再現) |
| range | `diff.relative=true` でサブディレクトリから実行 | サブディレクトリ外の追加行が出ず clean |
| range | コミット済みの `.gitattributes` の diff driver にローカルの textconv | 変換後の内容を scan して clean |
| range | `diff.renames=copies` で base のファイルを複製 | 複製が copy と判定され追加行が出ず clean |
| staged | 既定の設定で非 ASCII のファイル名 | `--name-only` が引用符付きの名前を返し、`git show ":<name>"` が失敗して黙って飛ばす |
| branch | symlink の `.gitallowed` のリンク先が改行で終わる | 改行が落ちて別のファイルに解決(AR-4) |
| range | `git log` 自体の失敗(存在しない `diff.orderFile`、不正な範囲など) | パイプの左側の失敗が見えず、追加行 0 件で clean(Codex plan advisory が再現) |
| range | `core.bigFileThreshold` が小さい | 閾値を超えたテキストがバイナリ扱いになり本文の diff が出ない(Codex plan advisory が再現) |

## Scope

1. **range scan の設定からの独立**(`scan_range`): `git log -p` の出力が、色(`color.ui` / `color.diff`)、`diff.relative`、textconv、rename / copy 検出の設定、diff のアルゴリズムの設定に左右されないようにする。`core.bigFileThreshold` は git の既定値(512m)に固定する。rename 検出は git の既定(rename は検出、copy は検出しない)に固定する(CI と同じ行を出すため。検出を切ると CI より多く scan して、CI が通す branch をローカルで止める)。コマンドラインのオプションで上書きできるものはオプションで、できないものは `-c` で上書きする
1b. **range scan の失敗を clean にしない**: `git log` の出力を一時ファイルに取り、終了状態を確かめてから解析する(POSIX sh のパイプでは左側の失敗が見えない)。`git log` が失敗したら、出力の前でも途中でも、「検出」(exit 1)とは別の非 0 で終わり、理由を出す。`secret-scan-branch.sh` は既存の「scanner failed with exit <rc>」の経路でそれを伝え、clean を出さない
2. **staged scan の設定と引用からの独立**(`scan_staged`): 非 ASCII、空白、タブ、引用符、バックスラッシュを含むファイル名も scan する(`-z` などで引用を避ける)。`diff.relative` に左右されず、サブディレクトリから実行しても repo 全体の staged を scan する。一覧に出たパスの blob が読めないときは黙って飛ばさず失敗にする(fail-closed)。ただし内容のない gitlink(submodule、mode 160000)は飛ばしてよい
3. **`--diff` モード**(`scan_diff_stream`): 呼び出し側が色付きの diff を渡しても追加行を読む(判定の前に色のエスケープを外す)。range scan は 1 で色を出さないので、これは外部入力の備え
4. **AR-4**(`secret-scan-branch.sh`): symlink の `.gitallowed` のリンク先をバイト列どおりに読む(コマンド置換の末尾改行の削除を避ける)。リンク先に改行が含まれるときは解決せず、空の allowlist と既存の通知に落とす(fail-closed)。`git ls-tree` の出力は `core.quotePath` で非 ASCII の名前が引用されるので、非 ASCII のリンク先も解決できるようにする(引用が残る名前は従来どおり fail-closed)
5. `templates/base/scripts/` のコピーを byte 一致に保つ

## Non-goals

- `.gitattributes` の `-diff` / `binary` 属性、または実際のバイナリファイルで `git log -p` が「Binary files differ」になり内容を scan しない件(CI も同じ挙動。CI 側の設計の穴として tech-debt に記録する)
- merge commit の diff(`git log -p` は既定で merge の diff を出さない。CI も同じ。tech-debt に記録する)
- scanner のパターン、`.gitallowed` の内容、`.github/workflows` の変更
- 改行を含むファイル名の完全な扱い(POSIX sh で NUL 区切りを読む手段が限られるため。扱えない場合は fail-closed にし、その旨を記録する)
- Windows

## Assumptions

- CI(`verify.yml`)は既定の git 設定で `./scripts/secret-scan.sh --range` を実行している。既定の設定での scan 結果は変更の前後で同じでなければならない(回帰の基準)
- `git show ":<path>"` はこの git(2.49.0)では textconv を適用しない(orchestrator が確認)。blob の取得は `git cat-file blob` でも可
- scaffold 先の git は `--no-relative`(2.28 以降)より古いことがある。`-c` での上書きならどの版でも効く

## Affected areas

- `scripts/secret-scan.sh`、`templates/base/scripts/secret-scan.sh`
- `scripts/secret-scan-branch.sh`、`templates/base/scripts/secret-scan-branch.sh`
- `tests/test-secret-scan.sh`(新しいケースは HOME / `GIT_CONFIG_GLOBAL` を隔離する。既存のケースは現状のまま)、`tests/test-secret-scan-branch.sh`
- `docs/tech-debt/README.md`(Non-goals の 2 件)

## Design decisions

- rename 検出は切らずに git の既定に固定する。検出を切るとローカルだけが余分に止める(CI と結果が食い違う)
- 読めない staged の blob は失敗にする。黙って飛ばしたことが非 ASCII のファイル名の見落としを隠していた
- Critical forks: None(範囲はユーザーが決定済み。方式は「CI と同じ結果」という #169 の契約から決まる)

## Acceptance criteria

- [ ] AC-1: range scan は、repo の設定に `color.ui=always` があっても、`color.diff=always` があっても fixture を検出する(exit 1)
- [ ] AC-2: range scan は、`diff.relative=true` の repo でサブディレクトリから実行しても、サブディレクトリの外の fixture を検出する
- [ ] AC-3: range scan は、コミット済みの `.gitattributes` の diff driver にローカルの textconv が設定されていても、元の内容を scan して fixture を検出する
- [ ] AC-4: range scan は、`diff.renames=copies` の repo で base のファイルを複製した branch の fixture を検出する。rename だけの branch では、既定の設定と同じ結果になる(rename 検出は維持)
- [ ] AC-5: range scan は diff のアルゴリズムを git の既定(myers)に固定し、`diff.algorithm` の設定に左右されない。設定で追加行が変わる fixture を作れればテストで固定し、作れなければ固定したことだけを確認して test report にその旨を記録する(未確認の経路。防御的な固定)
- [ ] AC-6: 既定の設定での range scan の結果は変わらない(既存のテストがそのまま pass)
- [ ] AC-7: staged scan は、非 ASCII、空白、タブ、引用符、バックスラッシュを含む名前の staged ファイルの fixture を検出する
- [ ] AC-8: staged scan は、`diff.relative=true` の repo でサブディレクトリから実行しても、repo 全体の staged ファイルを scan する
- [ ] AC-9: staged scan は、一覧に出たパスの blob が読めないとき exit 0 にならない(理由を出す)。staged の gitlink は失敗にしない
- [ ] AC-10: `--diff` モードは、色付きの diff の追加行を検出する
- [ ] AC-11: `secret-scan-branch.sh --strict` は、symlink の `.gitallowed` のリンク先が改行で終わる(改行なしの名前のファイルと両方コミットされている)とき、改行なしのファイルを allowlist として読まず、空の allowlist と通知で scan して fixture を検出する
- [ ] AC-12: `secret-scan-branch.sh --strict` は、非 ASCII の名前のリンク先を解決して allowlist として読む
- [ ] AC-13: `secret-scan-branch.sh --strict` は、repo の設定に `color.ui=always` があっても fixture を検出する(end-to-end)
- [ ] AC-15: range scan は、`git log` が出力の前に失敗したとき(存在しない `diff.orderFile` の設定、存在しない範囲)も、出力の途中で失敗したとき(範囲の途中の object が壊れている)も、exit 0 にも exit 1 にもならず、理由を出す
- [ ] AC-16: range scan は、`core.bigFileThreshold` がそのファイルより小さい repo でも、テキストファイルの fixture を検出する
- [ ] AC-17: `secret-scan-branch.sh --strict` は、scanner が AC-15 の失敗で終わったとき clean を出さず、scanner の終了コードで終わる(既存の「scanner failed with exit <rc>」の経路)
- [ ] AC-14: `scripts/` と `templates/base/scripts/` のコピーが byte 一致。`./scripts/run-verify.sh` green、この branch 自身の `secret-scan-branch.sh --strict` が clean

## Implementation outline

1. Slice A: `secret-scan.sh` の range と `--diff`(AC-1〜AC-6、AC-10、AC-15、AC-16)とテスト
2. Slice B: `secret-scan.sh` の staged(AC-7〜AC-9)とテスト
3. Slice C: `secret-scan-branch.sh` のリンク先の読み取りと `ls-tree` の引用(AC-11〜AC-13、AC-17)とテスト
4. tech-debt の 2 行(Non-goals)は sync-docs で記録

## Verify plan

- Static analysis checks: `./scripts/run-static-verify.sh`(shellcheck)、`sh -n`
- Spec compliance criteria to confirm: AC-1〜AC-14 をコードとテストに対応付ける。Non-goals(パターン、`.gitallowed`、workflows を触らない)
- Documentation drift to check: `docs/quality/quality-gates.md`(2 コピー)、`/pr` skill の secret scan の記述、scanner のヘッダーコメント
- Evidence to capture: 修正前の scanner で上の表の 6 経路が見逃すこと、修正後に検出すること

## Test plan

- Unit tests: `tests/test-secret-scan.sh` に range(色 2 種、`diff.relative`、textconv、copies、rename だけ、アルゴリズム)、staged(名前 5 種、サブディレクトリ、読めない blob、gitlink)、`--diff` の色付き入力
- Integration tests: `tests/test-secret-scan-branch.sh` に AR-4 の fixture、非 ASCII のリンク先、`color.ui=always` での end-to-end。`tests/test-run-verify-branch-secret-scan.sh` は現状のまま pass
- Regression tests: 既存のテストすべて、既定の設定での結果の一致
- Edge cases: 改行を含むファイル名(扱えなければ fail-closed であること)、先頭が `-` のファイル名、空の staged
- Evidence to capture: red/green(各オプションや `-c` を 1 つずつ外すと対応するテストが落ちる)、fixture の値は実行時に断片から組み立てる

## Risks and mitigations

- 読めない staged の blob で失敗にすると、今まで通っていたコミットが止まる場合がある(例: 想定外の index の状態)。理由を出して止まるので、黙って通るよりは扱いやすい。gitlink は除外する
- `git log` のオプションが古い git にないと CI やローカルで scanner 自体が失敗する。AC-15 により失敗は clean ではなく失敗として出る(push は止まる)。`-c` の上書きを優先し、オプションは長く存在するもの(`--no-color`、`--no-textconv`、`--no-ext-diff`、`--diff-algorithm`)に限る
- CI の結果が変わると、既存の branch の CI が突然落ちる。AC-6 で既定の設定の結果が変わらないことを固定する

## Rollout or rollback notes

- 変更は scanner 2 本とテストに閉じる。revert すれば従来の挙動に戻る。scaffold 先は `ralph upgrade` で受け取る

## Open questions

- なし

## Deviation notes

- 2026-09-25 plan: issue の 2 件に加え、調査で再現した 4 経路(`diff.relative`、textconv、`diff.renames=copies`、staged の非 ASCII のファイル名)を範囲に入れた。ユーザー決定(AskUserQuestion): すべて #176 で直す
- 2026-09-25 plan: Codex plan advisory の HIGH 2 件を反映。(1) `git log | scan_diff_stream` のパイプでは `git log` の失敗が見えず、存在しない `diff.orderFile` などで追加行 0 件の clean になる → 出力を一時ファイルに取り終了状態を確かめる(Scope 1b、AC-15、AC-17)。(2) `core.bigFileThreshold` が小さいとテキストがバイナリ扱いになり本文の diff が出ない → 512m に固定(Scope 1、AC-16)。ユーザー決定(AskUserQuestion): 対応案で plan を更新
- 2026-09-25 work: Slice A(014ab77)、B(3060815)、C(a8adfc7)は implementer に委譲(security の関門なので model-routing の規則どおり opus)。A: `scan_range` は `git log` の出力を一時ファイルに取り終了状態を確かめてから解析、失敗は exit 3。固定は `--no-color` / `--no-textconv` / `--no-ext-diff` / `--diff-algorithm=default` / `--no-show-signature` と `-c diff.relative=false` / `diff.renames=true` / `core.bigFileThreshold=512m`。implementer が再現した上で 4 つ追加: `core.attributesFile=/dev/null`(ユーザー単位の `-diff`)、`log.showRoot=true`(無関係な履歴を merge した branch の root commit)、`--no-replace-objects`(replace ref による差し替え)、`--submodule=short`(`diff.submodule=diff` で CI より多く scan)。`scan_diff_stream` は色のエスケープを外す。AC-5 は myers と histogram / patience で追加行が変わる fixture(繰り返し行を越える移動)を作れたので固定した。B: staged は `git diff --cached --raw --no-abbrev --no-renames --diff-filter=d` の blob id を `git cat-file blob` で読み、パスはラベルだけ。gitlink と 0 の id は飛ばし、読めない blob は exit 3。改行を含むファイル名も scan する。C: symlink のリンク先は sentinel 付きで読み、改行を含むリンク先は解決しない。`ls-tree` は `core.quotePath=false`。`git show` を `git cat-file blob` に置き換え。scanner の exit 3 は既存の「scanner failed with exit <rc>」の経路で伝わる。テストは 6 → 50 件、96 → 108 件。修正前の scanner では新しいテストの大半が落ちる(A 20 件、B 13 件、C は両方修正前で 12 件)。固定を 1 つずつ外す mutation で対応するテストが落ちる(例外: `--no-color` は色の除去と二重の防御なので単独では落ちない、`--no-show-signature` はテストなし)。orchestrator は HEAD 一致・porcelain 空・差分・template の byte 一致、最初の probe の 5 経路がすべて検出に変わること、テスト 3 本、`run-verify.sh`、strict scan を確認。範囲外の発見(tech-debt に記録する): `.git/info/attributes` の `-diff`(config ではないので `-c` で消せない)、コミット済みの `.gitattributes` が指す driver へのローカルの `diff.<driver>.binary=true`、作業ツリーの `.gitattributes` の未コミットの変更(推測、未確認)、`diff.renameLimit`(未確認)、`secret-scan-branch.sh` 自身の git 呼び出しの replace ref。Non-goals の 2 件(コミット済みの `-diff` / `binary`、merge commit の diff)も同じく記録する

## Progress checklist

- [x] Plan reviewed
- [x] Branch created
- [x] Implementation started
- [ ] Review artifact created
- [ ] Verification artifact created
- [ ] Test artifact created
- [ ] PR created

## Readiness checklist

- [x] 6 経路を fixture repo で再現した(orchestrator)
- [x] critical fork なし(範囲はユーザー決定済み)
- [x] Codex plan advisory(HIGH 2 件、対応案で plan を更新)
- [x] AC は hermetic な fixture repo の shell テストで決定的に確認できる
