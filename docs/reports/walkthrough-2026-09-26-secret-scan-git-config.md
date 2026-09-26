# Walkthrough: secret-scan-git-config

- Date: 2026-09-26
- Plan: docs/plans/archive/2026-09-25-secret-scan-git-config.md(PR 作成時に active から移動)
- Issue: #176(Closes)。#169(PR #177)の最終 cross-review の残件 2 件に、plan 作成時の調査で再現した同じ種類の見落とし 4 つを加えた(ユーザー判断で範囲を拡大)
- Branch: fix/secret-scan-git-config(base: main @ c1785b8)
- 差分規模: 22 files / +2,802 −98。コードは `scripts/secret-scan.sh`(+198 前後、317 行)と `scripts/secret-scan-branch.sh`(+73 前後)、`templates/base/scripts/` の同一コピー、テスト 2 ファイル(`tests/test-secret-scan.sh` 6 → 86 件、`tests/test-secret-scan-branch.sh` 96 → 108 件)。残りは plan、pipeline のレポート、tech-debt の行、quality-gates / `/pr` skill / repo-map の文言

## 何が変わったか

`scripts/secret-scan.sh` は hook(`--staged`、`--file`、merge 中の `--range`)、CI(`verify.yml` の `--range`)、`/pr` の push 前の関門(`secret-scan-branch.sh` 経由の `--range`)が共通で使う scanner です。ローカルの結果は、既定の設定で動く CI と同じでなければなりません。

1. **range scan の設定からの独立**: `git log -p` が出す追加行を変えるローカルの設定を、git の既定に固定する。色、`diff.relative`、textconv、外部 diff、rename / copy 検出とその上限、diff のアルゴリズム、`core.bigFileThreshold`、ユーザー単位の attributes ファイル、`attr.tree`、root commit の diff、submodule の diff、replace ref、署名の表示。すべて git 2.8 で動く書き方(`-c` と古くからあるオプション)。
2. **失敗を clean にしない**: `git log` の出力をファイルに取り、終了状態を確かめてから解析する。git の失敗、範囲の終わりが commit に解決できない場合、`--file` のファイルがない・読めない場合、staged の行が解析できない・blob が読めない場合は exit 3(could not scan)。signal は 129 / 130 / 143 で止まる。
3. **属性の読み元**: 範囲が HEAD で終わるとき(CI、branch scan)は HEAD の tree の属性を読む(`GIT_ATTR_SOURCE`、git 2.41 以上)。それ以外(merge 中の guard の `HEAD..<merge head>`)は main と同じく作業ツリー、つまり merge 結果の属性を読む。
4. **staged scan**: `git diff --cached --raw` の blob id を `git cat-file blob` で読む。ファイル名はラベルだけに使うので、非 ASCII や引用が必要な名前でも飛ばさない。
5. **diff の解析**: `+++ ` を読み飛ばすのは diff の見出しの中だけ。hunk の中の `+` 行は、内容が `++ ` で始まっても読む。色のエスケープは行頭だけ外す。
6. **branch scan**: symlink の `.gitallowed` のリンク先をバイト列どおりに読み(末尾の改行を落とさない)、改行を含むリンク先は解決しない。非 ASCII のリンク先も解決する。

## 読む順番

1. `scripts/secret-scan.sh` のヘッダー(固定するもの、固定できない差、exit code)
2. `scan_range`: 範囲の終わりの解決 → HEAD かどうかで属性の読み元を選ぶ subshell → 固定の一覧(コメントと呼び出しが 1 対 1)→ 終了状態の確認
3. `scan_diff_stream` の awk(見出しと hunk の状態)
4. `scan_staged` と `is_object_id`
5. `scripts/secret-scan-branch.sh` の `.gitallowed` の読み取り(sentinel、`core.quotePath=false`)
6. `tests/test-secret-scan.sh` の hermetic な部分(再現の表の各経路、実際の merge と本物の guard の実行)

## コミット単位

| SHA | 内容 |
|---|---|
| 7da850a, 0904a25 | plan。Codex plan advisory の HIGH 2 件(パイプで `git log` の失敗が見えない、`core.bigFileThreshold`)を反映 |
| 014ab77 | range: 設定の固定、失敗の検出、`--diff` の色の除去 |
| 3060815 | staged: blob id で読む |
| a8adfc7 | branch: リンク先をバイト列どおりに読む |
| aecee05 | self-review cycle 1(`diff.renameLimit`、古い git への対応、`--file` と signal の exit、id の検証 ほか) |
| ca8a212 | cross-review cycle 1 の AR-1: 色の除去で内容の `++ ` が見出し扱いになる後退を、見出しの状態を持つ解析で直す(既存の `++ ` の見落としも閉じる) |
| 443ffb7 | self-review cycle 2(未コミットの `.gitattributes`、文言、tech-debt の訂正) |
| 004d99e, 0e136fc | cross-review cycle 2 の AR-2(merge guard の後退)と、その最初の修正が逆向きの後退を生んだ self-review run 3 の HIGH。最終形は「HEAD で終わる範囲だけ HEAD の属性」 |
| その他 | plan の記録、各レポート(3 回分)、insight events、tech-debt の行、quality-gates / `/pr` skill / repo-map の文言 |

## 設計判断

- CI と同じ行を読むことを基準にする。検出を減らす方向(fail-open)も、増やす方向(CI が通す branch をローカルだけが止める)も直す対象。rename 検出を切らずに既定に固定したのはこのため。
- 失敗は clean ではなく exit 3。hook と `run-verify.sh` と `/pr` はどれも非 0 で止まるので、呼び出し側の変更は要らない。
- merge 中の guard は merge 結果の属性を読む。範囲の片側の属性を読むと、どちらかの向きで見逃す(cross-review と self-review が 1 回ずつ実証)。

## 注意して見てほしい点

- `scan_range` で exit 0 に至るのは、git log が成功して解析が終わった場合だけであること。
- 固定の一覧のコメントと、実際の `-c` / オプションが一致していること。
- 既定の設定での結果が main と変わらないこと(この repo の全履歴で、取り出す追加行が main と一致することを implementer と tester が確認。変わるのは hunk の中の `++ ` で始まる行を新たに読むことだけ)。

## Known limitations

- 固定できないローカルの差が 2 つある: `.git/info/attributes` の `-diff` / `binary`、コミット済みの driver に対するローカルの `diff.<driver>.binary=true`(tech-debt に記録)。
- CI の pull_request は PR の merge commit を checkout するので、branch の作成後に base 側で `.gitattributes` が変わると、CI とローカルの結果がずれうる(main からある差。再現済み、tech-debt に記録)。
- CI と共通の穴: コミット済みの `-diff` / `binary` とバイナリファイル、merge commit の diff(tech-debt に記録)。
- `A...B` の範囲で B が HEAD のとき、A 側だけから到達する commit も HEAD の属性で読む(main と同じ。呼び出し元は使っていない)。
- テストがないもの: HUP / INT の trap(TERM はある)、行頭の色の除去の形、`log.showSignature=false`。git 2.41 / 2.42 での `GIT_ATTR_SOURCE` は未実行(2.40.4 は無視、2.43.7 は反映を確認)。
- `secret-scan-branch.sh` 自身の git 呼び出しは replace ref に従う(`GIT_NO_REPLACE_OBJECTS=1` で閉じられる。再現済み、tech-debt に記録)。
