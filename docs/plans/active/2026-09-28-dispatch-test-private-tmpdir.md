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
- 再現 fixture: テスト所有の「模擬共有 TMPDIR」(`$workdir/shared-tmp`)を環境の `TMPDIR` にして、第 2 の dispatcher(別の fixture repo、別の event 名の遅い hook)をそこで動かし、その一時ファイルがある状態で case I の検査を通す。第 2 の dispatcher は case I の終わりに TERM で止め、自分が持ち込んだファイルが消えたことを確認する。ホストの共有 `${TMPDIR:-/tmp}` は `workdir` の作成にだけ使い、`ralph-dispatch-*` の列挙も削除もしない。
- ヘッダーのケース一覧(`i.`)の更新と、`docs/tech-debt/README.md` の該当行の削除(閉じた gap)。

## Non-goals

- `ralph-dispatch.sh` 自体の変更(`TMPDIR` はすでに尊重している)。
- (2026-09-29 に改訂、Slice C で対応済み)case I の `pgrep -f "PreCompact.d/10-slow.sh"` / `pkill -f` による子プロセスの検査が、同じスイートの並走(別セッションが同時に同じテストを走らせる)と衝突する件。当初は記録にとどめる予定だったが、並走 2 本で 6 組中 4 組が落ちることを実測し、修正が hook 名の一意化の数行だったので対象に含めた(Deviation notes 参照)。
- 他のケース(A〜H)の隔離の強化。専用 `TMPDIR` をスイート全体に適用する案は実装の選択として許す(下記)が、A〜H の assertion は変えない。
- `docs/plans/archive/2026-09-25-doctor-shell-alias-rc-types.md` の記録は履歴なので触らない。

## Assumptions

- `TMPDIR` を dispatcher の環境に渡せば、その配下に一時ファイルが作られる(調査で確認)。
- 第 2 の dispatcher は模擬共有 TMPDIR(`$workdir` 配下)で動かす。実機の失敗(別セッションの hook が同じディレクトリに一時ファイルを作る)と同じ構造を、ホストの共有領域に触れずに再現できる。
- CI(ubuntu、`TMPDIR` 未設定 → `/tmp`)でも同じ手順で動く。

## Affected areas

- `tests/test-ralph-dispatch.sh`(case I とヘッダーのケース一覧)
- `docs/tech-debt/README.md`(case I の行を削除)

## Design decisions

- 隔離の方式: 専用 `TMPDIR`(`$workdir` 配下)を dispatcher に渡し、検査はそのディレクトリを列挙する。PID や時刻でのフィルタは採らない(他のプロセスの同名ファイルを区別できない)。共有ディレクトリの集合差と `rm -f` は削除する。
- 実装の選択(実装者に委ねる): (a) case I の起動サブシェルだけで `TMPDIR` を設定する、(b) `workdir` 作成直後に `export TMPDIR="$workdir/tmp"` としてスイート全体を専用ディレクトリにする。(b) は 1 行で A〜H も隔離されるが、共有の一時ディレクトリは fixture 用に先に `host_tmpdir` として保存する。どちらでも AC は同じ。
- 再現 fixture は「本物の第 2 の dispatcher」で行う(ディレクトリにファイルを `touch` するだけの模擬より、失敗した実機の状況に近い)。hook の名前と event は case I のものと変え、`pgrep -f` の検査に写らないようにする。
- 模擬共有 TMPDIR(Codex advisory HIGH 1 への対応): 第 2 の dispatcher と、修正前の検査の再現は、`$workdir/shared-tmp` を環境の `TMPDIR` として動かす。旧 case I の `rm -f` が他セッションのファイルを消す経路をなくし、テストがホストの `${TMPDIR:-/tmp}` の `ralph-dispatch-*` を列挙・削除する箇所を残さない。case I の対象 dispatcher には別の専用ディレクトリ(`$workdir/case-i-tmp`)を渡す。
- 再現の順序(Codex advisory MEDIUM 2 への対応): 旧検査は開始時にスナップショットを取るので、第 2 の dispatcher はその後に起動しなければ差分に写らない。修正前の再現は「事前スナップショット → 第 2 の dispatcher 起動と started marker の確認 → 対象へ TERM → 事後検査 → fixture 停止」の順を marker で同期し、落ちた対象が模擬共有 dir の中の fixture 所有のファイルであることを確認する。修正後のテストは「第 2 の dispatcher 起動と started marker の確認 → 模擬共有 dir に `ralph-dispatch-*` があることの確認 → case I(専用 dir)→ 専用 dir が空であることの確認 → fixture 停止 → 模擬共有 dir から消えたことの確認」の順。
- Critical forks: None

## Acceptance criteria

- [ ] AC-1: case I の dispatcher は `$workdir` 配下の専用 `TMPDIR` で起動され、漏れの検査はそのディレクトリだけを列挙する。SIGTERM 後にそのディレクトリに `ralph-dispatch-*` が残らない(既存の意図の維持)。共有の `${TMPDIR:-/tmp}` は列挙も削除もしない(`rm -f $i_leaked_tmp` は削除)。
- [ ] AC-2: 再現 fixture: case I の実行中、第 2 の dispatcher(別の fixture repo、別の event 名の遅い hook)が模擬共有 TMPDIR(`$workdir/shared-tmp`、環境の `TMPDIR`)に `ralph-dispatch-*` を持つ状態を作り、その存在をテスト自身が確認する(fixture が効いていることの sanity。存在しなければ FAIL)。この状態で case I の全 assertion が pass する。
- [ ] AC-3: 同じ fixture を修正前の case I(環境の `TMPDIR` の集合差)に、旧検査の事前スナップショットの後・対象への TERM の前に起動する順で当てると、「left stray ralph-dispatch-* temp files」で落ち、落ちた対象は模擬共有 dir の中の fixture 所有のファイルである(scratch のコピーに fixture を挿入して確認し、test report に残す。ホストの共有領域は使わない)。
- [ ] AC-4: 第 2 の dispatcher は case I の終わりに TERM で止め、その一時ファイルが模擬共有 dir から消えたことを確認する。テストが途中で止まっても残らないよう、EXIT の trap でも止める。
- [ ] AC-5: `tests/test-ralph-dispatch.sh` の既存ケース A〜I がすべて pass(26 件 + 追加分)、5 回連続で pass、`shellcheck -S warning` で警告なし、`./scripts/run-verify.sh` green。
- [ ] AC-6: ヘッダーのケース一覧(`i.`)が新しい検査を説明し、`docs/tech-debt/README.md` の case I の行が削除されている。
- [ ] AC-7: テストはホストの共有 `${TMPDIR:-/tmp}` を `workdir` の作成にだけ使い、そこで `ralph-dispatch-*` を列挙も削除もしない(`find` / `rm` の対象は `$workdir` 配下だけ。静的に確認)。tester はホストの共有 dir に `ralph-dispatch-canary.*` を置いてスイートを走らせ、実行後も残っていることで動的にも確認する。

## Implementation outline

1. Slice A(implementer、sonnet): `workdir` 作成直後に模擬共有 TMPDIR を作って環境の `TMPDIR` にする。case I の書き換え(対象 dispatcher に専用 `TMPDIR`、共有ディレクトリの集合差と `rm -f` の削除、専用 dir が空であることの検査)、第 2 の dispatcher の fixture(起動 → started marker 待ち → 模擬共有 dir に `ralph-dispatch-*` があることの確認 → case I → TERM → 消えたことの確認、EXIT trap での後始末)、ヘッダー更新、tech-debt の行の削除。1 コミット。red の証拠: 修正前の case I を scratch にコピーし、事前スナップショットの直後に fixture の起動を挿入して、模擬共有 dir を `TMPDIR` にした実行で「stray」で落ち、落ちた名前が fixture のファイルであることを示す。
2. pipeline: self-review → verify → test → sync-docs → cross-review → PR。

## Verify plan

- Static analysis checks: `shellcheck -S warning tests/test-ralph-dispatch.sh`、`bash -n`、`./scripts/run-static-verify.sh`。
- Spec compliance criteria to confirm: AC-1〜AC-6 を `tests/test-ralph-dispatch.sh` の該当行と実行結果で確認。共有ディレクトリを読む `find` と `rm -f $i_leaked_tmp` が残っていないこと。
- Documentation drift to check: ヘッダーのケース一覧、`docs/tech-debt/README.md`(行の削除と、他の行からの参照がないこと)。
- Evidence to capture: `docs/evidence/verify-*.log`、verify report の AC 表。

## Test plan

- Unit tests: `bash tests/test-ralph-dispatch.sh`(全ケース pass、件数)。
- Integration tests: `./scripts/run-verify.sh`(スイートを呼ぶ経路で green)。
- Regression tests: 修正前の case I に fixture を当てた red の記録(scratch、模擬共有 dir、順序は AC-3 のとおり)。5 回連続実行。
- Edge cases: `TMPDIR` 未設定(`/tmp`)、`TMPDIR` に末尾 `/`、第 2 の dispatcher の started marker が出ない場合の失敗の文言、ホストの共有 dir の canary が残ること(AC-7)。
- Evidence to capture: test report(件数、red / green、反復回数)。

## Risks and mitigations

- 第 2 の dispatcher が孤児になる: EXIT trap で TERM し、`pkill -f` を安全網にする(hook の path は一意にする)。
- 模擬共有 dir に fixture のファイルを残す: 終了時に消えたことを assertion で確認する(`workdir` ごと EXIT で消える)。
- ホストの共有 dir に触れる: `find` / `rm` の対象を `$workdir` 配下に限る(AC-7、静的確認と canary)。
- 実行時間が延びる: 第 2 の dispatcher の起動待ちは started marker のポーリング(最大 5 秒)。

## Rollout or rollback notes

テストと文書だけの変更。問題があれば 1 コミットを revert する。

## Open questions

なし。

## Deviation notes

- 2026-09-28 plan: Codex plan advisory(gpt-6-astra、xhigh)が 2 件。HIGH 1: 修正前の再現をホストの共有 `TMPDIR` で行うと旧 case I の `rm -f` が他セッションのファイルを消しうる → 模擬共有 TMPDIR(`$workdir/shared-tmp`)を環境の `TMPDIR` にし、ホストの共有領域を列挙・削除しない AC-7 を追加。MEDIUM 2: 第 2 の dispatcher を旧検査のスナップショットより前に起動すると差分が空で red にならない → 再現の順序を marker で同期し、落ちた対象が fixture 所有であることを確認する AC-3 に改訂。ユーザー決定: 対応案で plan を更新
- 2026-09-28 work: Slice A は implementer(sonnet)に委譲(84c412f、2 ファイル、+117 / -23、push 済み)。設計は plan の (b): `workdir` 作成直後に `TMPDIR=$workdir/shared-tmp` を export(模擬共有 dir)。case I の対象 dispatcher は起動サブシェル内で `TMPDIR=$workdir/case-i-tmp`、検査はその dir だけを `find`(共有 dir のスナップショットと `rm -f $i_leaked_tmp` は削除)。第 2 の dispatcher は `$workdir/repo2` の `Stop.d/10-concurrent-slow.sh`(sleep 30)で、started marker(最大 5 秒)→ 模擬共有 dir に `ralph-dispatch-*` があることの assertion → case I → TERM + wait → 消えたことの assertion。EXIT trap でも TERM と `pkill -f`。ヘッダーの `i.` を更新、tech-debt の行を削除。red: 旧テスト(efd4ec1)の scratch コピーで事前スナップショットの直後に fixture を挿入し `TMPDIR=<scratch>/shared` で実行 → 25 PASS / 1 FAIL「left stray ralph-dispatch-* temp files: …/shared/ralph-dispatch-{merged,out,stdin}.*」(fixture 所有)。ホストの canary は残った。green: 29 / 0(26 から +3)、5 回連続、shellcheck clean、`run-verify.sh` All verifiers passed。orchestrator も 29 / 0・canary・shellcheck を確認
- 2026-09-29 self-review(cycle 1、b45326e): CRITICAL 0 / HIGH 0 / MEDIUM 1 / LOW 5、merge 可。MEDIUM: 第 2 の dispatcher を止める `pkill -f "Stop.d/10-concurrent-slow.sh"` が同一ホストの別スイート実行の hook にも一致する(dispatcher は hook を相対パスで起動するので cmdline に実行ごとの区別がない)。LOW: `sleep 30` が TERM 後に孤児化、模擬共有 dir の事前空チェックがない、専用 dir の検査が名前で絞っている、PASS 行に実行ごとのパス、コメント 4 か所のずれ。全件を in-cycle で修正
- 2026-09-29 work: Slice B は implementer に委譲(3ba2f65、2 ファイル、+92 / -31)。第 2 の dispatcher の hook を `10-concurrent-slow-$$.sh` にして `pkill -f` を 2 か所とも削除(停止は trap の `kill -TERM` + `wait`)、hook の末尾を `exec sleep 30`(TERM が sleep 自身に届く。停止後の `wait` の rc が 143 であることを assertion)、fixture 起動前に `$shared_tmp` が空であることの assertion、`find` の対象を `$shared_tmp` に、専用 dir は `-mindepth 1` で全件、PASS 行は件数だけ、コメントを修正。テスト 31 / 0(29 から +2)。implementer が並走 2 本で既存の `PreCompact.d/10-slow.sh` の `pgrep`/`pkill` が互いの hook を拾い 6 組中 4 組が落ちることを実測(旧テスト efd4ec1 でも 4 / 6、同じ失敗の型を含む)。この時点では tech-debt の行として記録
- 2026-09-29 work(非目標の改訂): Slice C は implementer に委譲(300aa85、2 ファイル、+28 / -16)。case I 自身の hook も `10-slow-$$.sh` にし、`pgrep -f` / `pkill -f` を同じ実行ごとのパスに。Slice B の tech-debt の行を削除、ヘッダーの `i.` に一意な名前の理由を 1 文。並走 2 本 × 6 組と 3 本 × 1 組はすべて rc 0、FAIL 行なし、孤児の `sleep` なし。orchestrator も並走 2 本(31 / 0 × 2)、ホストの canary、孤児プロセスなしを確認。理由: 修正が hook 名の一意化の数行で、放置すると #182 と同じ「同一ホストの別実行」型の偽 FAIL が残るため
## Progress checklist

- [x] Plan reviewed
- [x] Branch created
- [x] Implementation started
- [x] Review artifact created
- [ ] Verification artifact created
- [ ] Test artifact created
- [ ] PR created

## Readiness checklist

- [x] 原因(共有 `TMPDIR` の集合差)と `rm -f` の副作用をコードで確認した
- [x] critical fork なし
- [x] Codex plan advisory(HIGH 1 / MEDIUM 1、対応案で plan を更新)
- [x] AC は決定的な shell テストで確認できる
