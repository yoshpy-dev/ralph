# Walkthrough: doctor-shell-alias-rc-types

- Date: 2026-09-25
- Plan: docs/plans/archive/2026-09-25-doctor-shell-alias-rc-types.md(PR 作成時に active から移動)
- Issue: #167(Closes)。#162(PR #168)の cross-review cycle 2 で残った WORTH_CONSIDERING 2 件(WC-3 / WC-4)
- Branch: fix/doctor-shell-alias-rc-types(base: main @ a5aafc3)
- 差分規模: 10 files / +1,135 −44。コードは `internal/cli/` の 3 ファイル(+533 −43): `doctor_shell_alias.go`(+139 −43)、`doctor_shell_alias_test.go`(+265)、`doctor_shell_alias_unix_test.go`(新規 129 行)。残りは plan と pipeline のレポート、tech-debt の 2 行

## 何が変わったか

`ralph doctor` の「Shell aliases (codex/claude)」Check が対象です。

1. **通常ファイルでない rc を開かない**: 候補の rc が存在し(symlink は辿る)、ディレクトリでも通常ファイルでもない(FIFO、デバイス、socket)とき、`scanShellAliasFile` は開かずに `not a regular file` を返す。Detail の「could not read」の節に `~/.zshrc (not a regular file)` のように出て、Check は info。以前は FIFO の `.zshrc` で `os.Open` が書き手を待って固まり、`ralph doctor` 全体が何も出さずに止まった。ディレクトリは従来どおり黙って飛ばす。
2. **相対パスの `$ZDOTDIR`**: 以前は無視していた(コメントは「zsh が拒否する」と書いていたが誤り)。zsh は相対の `ZDOTDIR` をシェルの作業ディレクトリ基準で解決する。doctor は自分の作業ディレクトリ(`shellAliasEnv.Cwd`、`os.Getwd`)で herdr の pane の cwd を近似する。`os.Getwd` が失敗したときは相対の `$ZDOTDIR` を飛ばす。
3. **`..` は OS に解決させる**: `$ZDOTDIR` 由来の候補は `filepath.Join` でなく連結(`shellAliasJoinRaw`)で作る。`Join` は `..` を字面で消すので、`ZDOTDIR=link/../rc` で `link` が symlink のとき zsh と別の rc を読んでいた(相対・絶対とも)。Detail は zsh が開くのと同じ `..` を残したパスを示す。重複の除去は `filepath.EvalSymlinks` で物理的に行う。

## 読む順番

1. `internal/cli/doctor_shell_alias.go` — `shellAliasEnv`(`Cwd`)→ `shellAliasEnvFromOS` → `shellAliasJoinRaw` / `shellAliasRawParent` / `shellAliasZdotdirBase` → `shellAliasRcCandidates` の doc comment(zsh の読み方、cwd の近似、`..` を残す理由はここに 1 回だけ書いてある)→ `scanShellAliasFile` の先頭の stat
2. `internal/cli/doctor_shell_alias_unix_test.go` — FIFO の 3 件(直接、symlink 経由、他の rc の所見との併存)。`checkShellAliasesWithin` が 5 秒で打ち切り、後始末で FIFO を書き込み側で開いて止まった goroutine を解放する
3. `internal/cli/doctor_shell_alias_test.go` — 相対 `$ZDOTDIR`、`Cwd` 空、`link/../rc`(相対・絶対)、symlink を含む `Cwd` と `..`、`$HOME` への dedup、ディレクトリの回帰、`shellAliasEnvFromOS` の `Cwd`、`shellAliasRawParent` / `shellAliasJoinRaw` の表。表示パスは完全一致で確認する

## コミット単位

| SHA | 内容 |
|---|---|
| 613b177, 380ae26 | plan。Codex plan advisory の 1 件(`..` を字面で消すと zsh と別の rc を読む)を反映 |
| daaf29f | 通常ファイルでない rc を開かずに報告。FIFO のテスト |
| b42533e | `Cwd`、相対 `$ZDOTDIR`、連結による候補、コメントの訂正。テスト |
| b611661 | self-review の LOW 6 件(root の親、末尾の区切り、コメント、表示パスの完全一致、FIFO テストの helper、HOME の固定) |
| その他 | plan の記録、各レポート、insight events、tech-debt の 2 行 |

## 設計判断

- 通常ファイルの判定は「stat して通常ファイルでなければ開かない」。#164 の `readCodexUserConfig` と同じ形。stat と open の間の差し替えは対象外(差し替えられる者はすでにアカウントを握っている)。
- 相対 `$ZDOTDIR` の基準は seam(`shellAliasEnv.Cwd`)で渡す。テストが実際の cwd に依存しない。
- 候補の一覧は over-approximate の方針のまま。`$HOME` と `$HOME/.config/zsh` は `$ZDOTDIR` があっても読む。

## 注意して見てほしい点

- `os.Open` の前に必ず stat と `IsRegular` があること。候補の stat と走査の stat はどちらも symlink を辿る。
- `$ZDOTDIR` 由来の候補の文字列が、どこでも字面で正規化されないこと(`shellAliasRawParent` は `filepath.Dir` の代わり)。
- alias の値やファイルの内容が Detail に出ないこと(従来どおり)。

## Known limitations

- `.zshrc` が `/dev/null` への symlink の環境では、Check が pass から info(`not a regular file`)に変わる。読んでいないのは事実なので受け入れた。
- 相対 `$ZDOTDIR` は doctor の cwd で解決するので、座席の cwd と違うディレクトリを読むことがある。
- `checkShellAliases` は `ralph.toml` を読まず、flag の分類だけで重大度を決める(既存の tech-debt。今回は重大度の規則を変えていない)。
- `tests/test-ralph-dispatch.sh` の case I が、同時に動いている agent の hook の一時ファイルを拾って 1 回だけ落ちた(tech-debt に記録、未確認)。
- このマシンの `/etc/zshenv` が `ZDOTDIR` を上書きするため、login zsh での実機確認はできていない。`zsh -f` で `link/../rc/.zshrc` を読ませ、doctor と同じ物理ファイルになることは確認した。
