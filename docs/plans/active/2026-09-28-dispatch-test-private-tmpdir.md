# dispatch-test-private-tmpdir

- Status: Draft
- Owner: Claude Code
- Date: 2026-09-28
- Related request: `tests/test-ralph-dispatch.sh` の case I(SIGTERM 後に dispatcher の一時ファイルが残らないことの検査)が、共有の `${TMPDIR:-/tmp}` にある `ralph-dispatch-*` を実行前後の集合差で比べるため、同じマシンで動く別の Claude Code / Codex セッションの hook(同じ `ralph-dispatch.sh`)が作った一時ファイルを「漏れ」として拾い、不安定になる(#167 の実装中に 1 回失敗、`docs/tech-debt/README.md` に記録)。issue #182
- Related issue: 182
- Type: test
- Branch: test/dispatch-test-private-tmpdir

## Objective

case I の dispatcher の実行と漏れの検査をテスト専用の `TMPDIR` で行い、他のプロセスが共有の一時ディレクトリに作るファイルの影響を受けないようにする。別の dispatcher が同時に動いている状態を fixture で再現し、修正前の検査が落ち、修正後の検査が通ることを示す。

## 調査で確認したこと(2026-09-28、main 21321cc)

- `.claude/hooks/ralph-dispatch.sh`(template と byte 一致)は一時ファイルを `mktemp "${TMPDIR:-/tmp}/ralph-dispatch-{stdin,out,merged}.XXXXXX"` で作り、EXIT / INT / TERM / HUP の trap で消す。`TMPDIR` を渡せば専用ディレクトリで動く。
- case I(`tests/test-ralph-dispatch.sh` 353〜412 行)は `i_tmpdir_base="${TMPDIR:-/tmp}"` を `find -maxdepth 1 -name 'ralph-dispatch-*'` で実行前後に列挙し、`comm -13` の差を漏れとみなす。別のセッションの hook が同じディレクトリに同じ名前のファイルを作ると、その hook の実行が前後のスナップショットにまたがるだけで失敗する。
- 失敗時の分岐は `rm -f $i_leaked_tmp` で「漏れ」を消す。他のセッションの hook がまだ使っているファイルを消すことになるので、不安定さに加えて他のセッションの hook 出力を壊しうる。
- テストの `workdir` 自体も `${TMPDIR:-/tmp}/ralph-dispatch-test.XXXXXX` で、`ralph-dispatch-*` に一致する。同じスイートの並走もスナップショットに写る。
- `tests/test-ralph-dispatch.sh` に template のコピーはない(`git ls-files`)。

## Scope

- `tests/test-ralph-dispatch.sh` の case I: dispatcher を `$workdir` 配下の専用 `TMPDIR` で起動し、漏れの検査はそのディレクトリの中身だけを見る。共有の `${TMPDIR:-/tmp}` は読まず、消さない。
- 再現 fixture: case I の実行中に第 2 の dispatcher(別の fixture repo、別の event 名の遅い hook)を共有の `${TMPDIR:-/tmp}` で動かし、その一時ファイルがある状態で case I の検査を通す。第 2 の dispatcher は case I の終わりに TERM で止め、自分が持ち込んだファイルが消えたことを確認する。
- ヘッダーのケース一覧(`i.`)の更新と、`docs/tech-debt/README.md` の該当行の削除(閉じた gap)。

## Non-goals

- `ralph-dispatch.sh` 自体の変更(`TMPDIR` はすでに尊重している)。
- case I の `pgrep -f "PreCompact.d/10-slow.sh"` / `pkill -f` による子プロセスの検査が、同じスイートの並走(別セッションが同時に同じテストを走らせる)と衝突しうる件。記録にとどめる。
- 他のケース(A〜H)の隔離の強化。専用 `TMPDIR` をスイート全体に適用する案は実装の選択として許す(下記)が、A〜H の assertion は変えない。
- `docs/plans/archive/2026-09-25-doctor-shell-alias-rc-types.md` の記録は履歴なので触らない。

## Assumptions

- `TMPDIR` を dispatcher の環境に渡せば、その配下に一時ファイルが作られる(調査で確認)。
- 第 2 の dispatcher を共有の `${TMPDIR:-/tmp}` で動かすのは、再現のために意図してやること。終了時に trap で消えるので、他のセッションのファイルには触れない。
- CI(ubuntu、`TMPDIR` 未設定 → `/tmp`)でも同じ手順で動く。

## Affected areas

- `tests/test-ralph-dispatch.sh`(case I とヘッダーのケース一覧)
- `docs/tech-debt/README.md`(case I の行を削除)

## Design decisions

- 隔離の方式: 専用 `TMPDIR`(`$workdir` 配下)を dispatcher に渡し、検査はそのディレクトリを列挙する。PID や時刻でのフィルタは採らない(他のプロセスの同名ファイルを区別できない)。共有ディレクトリの集合差と `rm -f` は削除する。
- 実装の選択(実装者に委ねる): (a) case I の起動サブシェルだけで `TMPDIR` を設定する、(b) `workdir` 作成直後に `export TMPDIR="$workdir/tmp"` としてスイート全体を専用ディレクトリにする。(b) は 1 行で A〜H も隔離されるが、共有の一時ディレクトリは fixture 用に先に `host_tmpdir` として保存する。どちらでも AC は同じ。
- 再現 fixture は「本物の第 2 の dispatcher」で行う(共有ディレクトリにファイルを `touch` するだけの模擬より、失敗した実機の状況に近い)。hook の名前と event は case I のものと変え、`pgrep -f` の検査に写らないようにする。
- Critical forks: None

## Acceptance criteria

- [ ] AC-1: case I の dispatcher は `$workdir` 配下の専用 `TMPDIR` で起動され、漏れの検査はそのディレクトリだけを列挙する。SIGTERM 後にそのディレクトリに `ralph-dispatch-*` が残らない(既存の意図の維持)。共有の `${TMPDIR:-/tmp}` は列挙も削除もしない(`rm -f $i_leaked_tmp` は削除)。
- [ ] AC-2: 再現 fixture: case I の実行中、第 2 の dispatcher(別の fixture repo、別の event 名の遅い hook)が共有の `${TMPDIR:-/tmp}` に `ralph-dispatch-*` を持つ状態を作り、その存在をテスト自身が確認する(fixture が効いていることの sanity。存在しなければ FAIL)。この状態で case I の全 assertion が pass する。
- [ ] AC-3: 同じ fixture を修正前の case I(共有 `TMPDIR` の集合差)に当てると「left stray ralph-dispatch-* temp files」で落ちる(scratch のコピーで確認し、test report に残す)。
- [ ] AC-4: 第 2 の dispatcher は case I の終わりに TERM で止め、その一時ファイルが共有ディレクトリから消えたことを確認する。テストが途中で止まっても残らないよう、EXIT の trap でも止める。
- [ ] AC-5: `tests/test-ralph-dispatch.sh` の既存ケース A〜I がすべて pass(26 件 + 追加分)、5 回連続で pass、`shellcheck -S warning` で警告なし、`./scripts/run-verify.sh` green。
- [ ] AC-6: ヘッダーのケース一覧(`i.`)が新しい検査を説明し、`docs/tech-debt/README.md` の case I の行が削除されている。

## Implementation outline

1. Slice A(implementer、sonnet): case I の書き換え(専用 `TMPDIR`、共有ディレクトリの集合差と `rm -f` の削除)、第 2 の dispatcher の fixture(開始 → started marker 待ち → 共有ディレクトリに `ralph-dispatch-*` があることの確認 → case I → TERM → 消えたことの確認、EXIT trap での後始末)、ヘッダー更新、tech-debt の行の削除。1 コミット。red の証拠: 修正前の case I を scratch にコピーし、同じ fixture を当てて落ちることを示す。
2. pipeline: self-review → verify → test → sync-docs → cross-review → PR。

## Verify plan

- Static analysis checks: `shellcheck -S warning tests/test-ralph-dispatch.sh`、`bash -n`、`./scripts/run-static-verify.sh`。
- Spec compliance criteria to confirm: AC-1〜AC-6 を `tests/test-ralph-dispatch.sh` の該当行と実行結果で確認。共有ディレクトリを読む `find` と `rm -f $i_leaked_tmp` が残っていないこと。
- Documentation drift to check: ヘッダーのケース一覧、`docs/tech-debt/README.md`(行の削除と、他の行からの参照がないこと)。
- Evidence to capture: `docs/evidence/verify-*.log`、verify report の AC 表。

## Test plan

- Unit tests: `bash tests/test-ralph-dispatch.sh`(全ケース pass、件数)。
- Integration tests: `./scripts/run-verify.sh`(スイートを呼ぶ経路で green)。
- Regression tests: 修正前の case I に fixture を当てた red の記録(scratch)。5 回連続実行。
- Edge cases: `TMPDIR` 未設定(`/tmp`)、`TMPDIR` に末尾 `/`、第 2 の dispatcher の started marker が出ない場合の失敗の文言。
- Evidence to capture: test report(件数、red / green、反復回数)。

## Risks and mitigations

- 第 2 の dispatcher が孤児になる: EXIT trap で TERM し、`pkill -f` を安全網にする(hook の path は一意にする)。
- 共有ディレクトリに fixture のファイルを残す: 終了時に消えたことを assertion で確認する。
- 実行時間が延びる: 第 2 の dispatcher の起動待ちは started marker のポーリング(最大 5 秒)。

## Rollout or rollback notes

テストと文書だけの変更。問題があれば 1 コミットを revert する。

## Open questions

なし。

## Progress checklist

- [ ] Plan reviewed
- [x] Branch created
- [ ] Implementation started
- [ ] Review artifact created
- [ ] Verification artifact created
- [ ] Test artifact created
- [ ] PR created

## Readiness checklist

- [x] 原因(共有 `TMPDIR` の集合差)と `rm -f` の副作用をコードで確認した
- [x] critical fork なし
- [ ] Codex plan advisory
- [x] AC は決定的な shell テストで確認できる
