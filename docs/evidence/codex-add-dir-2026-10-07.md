# codex の `--add-dir` 実機確認(workspace-write で cwd の外に書く)

- Date: 2026-10-07
- Plan: docs/plans/archive/2026-10-07-org-state-dir-common.md(PR 作成時に active から移動)
- 目的: `--sandbox workspace-write` の codex が cwd の外のディレクトリに書けるのは `--add-dir` を渡したときだけか、を確かめる。ralph はこの結果を前提に、台帳のディレクトリが cwd の外にある codex の leader 座席へ `--add-dir <台帳のディレクトリ>` を渡す(`internal/org/permissions.go` の `codexWritableRootArgs`)
- 利用者のマシンで確かめ直す手順: `docs/recipes/codex-seat-permissions.md` の「The org ledger and `--add-dir`」節

## 環境

| 項目 | 値 |
|---|---|
| codex | codex-cli 0.160.0 |
| OS | macOS |
| model | gpt-5.6-luna(モデルの選択は結果に関係しない) |
| ディレクトリ | `P=$HOME/.cache/ralph-adddir-probe`。`P/cwd`(`git init` 済みの空リポジトリ)と `P/state` を作った。どちらも `/tmp` と `$TMPDIR` の外にあり、どのリポジトリにも含まれない |

## Run 1: `--add-dir` なし

```sh
command codex -m gpt-5.6-luna exec --sandbox workspace-write --skip-git-repo-check -C $P/cwd "Run this exact shell command once and report its output verbatim, do not try alternatives: touch $P/state/probe-noflag && echo WROTE" </dev/null
```

- rc=0
- `P/state/probe-noflag` は作られなかった
- codex のログに出た行: `touch: /Users/<user>/.cache/ralph-adddir-probe/state/probe-noflag: Operation not permitted`

## Run 2: `--add-dir $P/state` あり

Run 1 のコマンドに `--add-dir $P/state` を足し、書き込み先を `probe-flag` に変えた。

- rc=0
- `P/state/probe-flag` が作られた

## 結論

workspace-write の sandbox では、cwd の外のディレクトリに書けたのは `--add-dir` で渡したときだけだった。2 回とも rc=0 だったので、書けたかどうかは終了コードでは分からない。ファイルができたかで判定する。確認のあと、probe のディレクトリは削除した。
