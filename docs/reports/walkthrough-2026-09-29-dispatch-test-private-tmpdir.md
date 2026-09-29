# Walkthrough: dispatch-test-private-tmpdir (#182)

- Date: 2026-09-29
- Plan: docs/plans/active/2026-09-28-dispatch-test-private-tmpdir.md
- Branch: test/dispatch-test-private-tmpdir(base main 21321cc)
- Diff: 9 files、+762 / -27(コードは `tests/test-ralph-dispatch.sh` の +247 行と tech-debt の 1 行削除。残りは plan と報告)

## 何を直したか

`tests/test-ralph-dispatch.sh` の case I は、SIGTERM 後に dispatcher の一時ファイルが残らないことを、ホストの共有 `${TMPDIR:-/tmp}` にある `ralph-dispatch-*` の実行前後の集合差で見ていた。同じマシンで動く別セッションの hook(同じ `ralph-dispatch.sh`)が同じディレクトリに一時ファイルを作ると、それを「漏れ」として拾い、しかも `rm -f` で消していた。この 2 つをなくした。

## 読む順(`tests/test-ralph-dispatch.sh`)

1. `workdir` 作成の直後: `shared_tmp="$workdir/shared-tmp"` を `TMPDIR` として export する。スイートが起動するすべての dispatcher(A〜H を含む)がここを使う。ホストの共有 dir は `workdir` の作成にだけ使い、そこで `find` も `rm` もしない。
2. case I の先頭: 模擬共有 dir が空であることの assertion。次に第 2 の fixture repo(`$workdir/repo2`)を作り、`Stop.d/10-concurrent-slow-$$.sh`(`exec sleep 30`)を hook にして dispatcher を背景で起動し、started marker を待ち、模擬共有 dir に `ralph-dispatch-*` があることを確認する(件数だけを PASS 行に出す)。
3. case I の対象 dispatcher: 起動サブシェルで `TMPDIR="$workdir/case-i-tmp"` を設定する。started marker の直後に、専用 dir に対象の一時ファイルがあること、模擬共有 dir の件数が fixture の件数のままであることを確認する(この 2 件が専用 `TMPDIR` の pin。dispatcher の trap はどの dir でも消すので、事後の検査では区別できない)。
4. TERM の後: 既存の assertion(exit 143、3 秒以内、finished marker なし、`.first` なし)に加え、専用 dir に何も残っていないこと(`-mindepth 1` で全件)。
5. 子プロセスの検査: hook 名は `PreCompact.d/10-slow-$$.sh` で、`pgrep -f` / `pkill -f` は実行ごとのパスに一致する(同一ホストで並走する別のスイート実行の hook に一致しない)。
6. 末尾: 第 2 の dispatcher を TERM + wait で止め、rc 143 と、模擬共有 dir から消えたことを確認する。EXIT trap でも止める。

## 検証

| 項目 | 結果 |
|---|---|
| `bash tests/test-ralph-dispatch.sh` | 33 PASS / 0 FAIL(main は 26) |
| 5 回連続 | すべて 33 / 0 |
| 並走 2 本 × 6 組、3 本 × 1 組 | すべて rc 0、FAIL 行なし |
| ホストの共有 dir の canary | すべての実行で残存 |
| 変異: 対象の `TMPDIR` 上書きを外す | 31 / 2(実行中の 2 assertion が落ちる) |
| 変異: 専用 dir の検査を `true` にして漏れを植える | 元 30 / 1、変異 31 / 0(assertion は生きている) |
| 旧テスト(efd4ec1)+ fixture(模擬共有 dir) | 26 / 1「left stray ralph-dispatch-* temp files」 |
| 旧テストの並走 2 本 × 3 組 | 2 組が落ちる(stray temp files、child alive の両型) |
| `./scripts/run-verify.sh`、shellcheck | green、警告なし |

## pipeline の履歴

- plan: Codex advisory HIGH 1(再現をホストの共有 dir で行うと旧コードの `rm -f` が他セッションのファイルを消す)/ MEDIUM 1(fixture を旧検査のスナップショットより前に起動すると差分が空)→ 模擬共有 dir と順序の同期に改訂。
- self-review: MEDIUM 1(`pkill -f` が別スイート実行の hook にも一致)/ LOW 5 → Slice B で修正。並走で既存の `10-slow.sh` の `pgrep`/`pkill` も衝突すると実測 → 非目標を改訂して Slice C で修正。
- verify pass → test pass(mutation で専用 `TMPDIR` の pin が未検証と判明 → Slice D で実行中の assertion を追加)→ sync-docs drift なし → cross-review 指摘 0 件。

## 残る gap

- `exec sleep 30` を `sleep 30` に戻しても exit 143 の assertion は通る(exec 後の cmdline は `sleep 30` だけで見分けられない。コメントに記載)。
