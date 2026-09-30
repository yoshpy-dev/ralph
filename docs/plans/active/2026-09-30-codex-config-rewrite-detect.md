# codex-config-rewrite-detect

- Status: Draft
- Owner: Claude Code
- Date: 2026-09-30
- Related request: main のチェックアウトにある追跡ファイル `.codex/config.toml`(`templates/base/.codex/config.toml` と byte 一致が `scripts/check-sync.sh` の要件)が、値は同じままコメントをすべて剥がされ、末尾に `[shell_environment_policy]`(`inherit = "core"` と `[shell_environment_policy.set]` の `CLAUDE_CODE_DISABLE_ADAPTIVE_THINKING = "1"`)を足された状態に書き換わることがある(2026-09-17、09-18)。その状態では `scripts/ralph-worktree.sh ensure` が「base branch 'main' has uncommitted changes」で止まり、次の plan の worktree が作れない。issue #185
- Related issue: 185
- Type: fix
- Branch: fix/codex-config-rewrite-detect

## Objective

原因の調査結果を記録し、書き換えが起きたときに `ralph-worktree.sh` が理由と戻し方を示して止まるようにする(issue の案 (c))。症状と戻し方を recipe に書く。

## 調査で確認したこと(2026-09-30、main 5efd3d6)

- issue と memory の仮説は「Claude Code の codex プラグイン(codex-companion)が `codex exec` の起動時に書き込む」だったが、ソースを読む限り当たらない:
  - openai-codex プラグイン(`~/.claude/plugins/cache/openai-codex/codex/1.0.2/scripts/`)の `writeFileSync` は state / job / broker / pid の JSON とログだけで、`config.toml` にも `shell_environment_policy` にも触れない。hooks は SessionStart / SessionEnd / Stop の 3 つ。
  - everything-claude-code プラグインの `scripts/sync-ecc-to-codex.sh` と `scripts/codex/merge-codex-config.js` は codex の config を扱うが、`shell_environment_policy` を含まない。hooks.json に codex の登録はない。
  - ralph 自身の Go コードは `.codex/config.toml` を読むだけ(`internal/cli/doctor.go`)で、書かない。
- 追加される設定の出どころ: `CLAUDE_CODE_DISABLE_ADAPTIVE_THINKING=1` はユーザーの `~/.claude/settings.json` とこの repo の `.claude/settings.json` の `env` にある。2026-08-24 の plan(38612ef、`codex-hooks-multi-event`)の Non-goals に「`shell_environment_policy` のユーザレベル移動 → メンテナ環境の main 差分ゼロ化(ローカル作業)」とある。つまりこの table はもともとメンテナが project の `.codex/config.toml` に手で入れていたもので、その後ユーザーレベルに移した。書き換えは「ユーザーレベルに移した table が、コメントを落とした再シリアライズとともに project 側に戻ってくる」形をしている。
- 隔離環境(scratch の偽 `HOME` / `CODEX_HOME`、ユーザーの実際の `~/.codex` は使わない)での probe: ユーザーレベルに `shell_environment_policy` とコメントを持つ config、project に template の `.codex/config.toml` を置き、project の中で `codex features enable <feature>`(config を書くサブコマンド)を実行した。書き込まれたのはユーザーレベルの config で、コメントは残り、`[features]` が追記されただけだった。project の config は変わらなかった。codex の通常の設定変更は原因ではない。
- この session(2026-09-27〜30)では worktree から `codex exec` / `codex exec review` を 15 回以上実行したが、書き換えは一度も起きていない。`codex exec` そのものは原因ではない。
- 残る候補(未確認、自動での再現が難しい): codex の TUI のダイアログ(project の trust、hook の trust、model 移行の確認)で設定を保存する経路、Codex の desktop app / IDE 拡張の設定 UI、codex のバージョン更新時の設定移行。発生した 09-17〜18 は codex の TUI 座席で実機確認をしていた時期(#155、#162)と重なる。
- `validate_clean_base`(`scripts/ralph-worktree.sh` 89〜99 行、template と byte 一致)は `git status --porcelain` が空でなければ一律に `base branch '<base>' has uncommitted changes` で止まる。
- 追跡している `.codex/config.toml` にインラインのコメントはなく、コメントは行頭 `#` の行だけ。

## Scope

- `scripts/ralph-worktree.sh`(+ template、byte 一致): `validate_clean_base` で、dirty な項目が `.codex/config.toml` の未ステージの変更だけ(`git status --porcelain` が ` M .codex/config.toml` の 1 行だけ)で、その差分が「コメント行・空行の削除」と「末尾への `[shell_environment_policy]` / `[shell_environment_policy.<name>]` table の追加」だけのとき、専用の理由と戻し方を出して止まる。戻し方は `git -C <root> diff -- .codex/config.toml` で差分を見てから `git -C <root> checkout -- .codex/config.toml`、確認は `git -C <root> status --porcelain` が空になること(配布先には template がないので、template との `cmp` は案内しない)。それ以外(ステージ済み・部分ステージ、HEAD または作業ツリーの版に複数行文字列 `"""` / `'''` がある、など)はこれまでどおりの文言で止まる。自動では戻さない。
- `tests/test-ralph-worktree.sh`: 検知の red / green のケース。
- `docs/recipes/codex-setup.md`(+ template): 症状、確認、戻し方、調査の状況(否定した候補と残る候補)を 1 段落で。
- 調査の記録: `docs/evidence/codex-config-rewrite-2026-09-30.md`(上の確認事項、probe の手順と結果、残る候補、次に起きたときに取る証拠)。

## Non-goals

- 原因の特定そのもの(残る候補はどれも対話的な操作で、この PR の中では再現できない)。issue #185 は open のまま残し、PR は `Refs #185` とする。
- 書き換えの自動修復(作業ツリーを変える操作で、手で入れた変更を消す危険がある)。
- `check-sync.sh` への検知の追加(issue の案 (b))。書き換えが実際に止めるのは worktree の作成なので、そこで案内すれば足りる。
- `.codex/config.toml` の中身の変更(`model = "gpt-5.5"` の退役は #156 で判断する)。

## Assumptions

- 書き換えの形は memory と issue に記録されたもの(コメント行の削除と、末尾への `shell_environment_policy` の table の追加)に限る。値の変更、table の並べ替え、引用符の変更を伴う差分は、この検知の対象にしない(一般の文言で止まる。安全側)。
- 行単位の正規化は、TOML の複数行文字列(`"""` / `'''`)の中の `#` 行や空行を値として扱えない(Codex advisory)。どちらかの版に複数行文字列の区切りがあれば検知しない(一般の文言)。TOML パーサは shell から使えないので導入しない。
- `validate_clean_base` は `ensure` と `validate-clean-base` の両方から呼ばれる。

## Affected areas

- `scripts/ralph-worktree.sh`、`templates/base/scripts/ralph-worktree.sh`
- `tests/test-ralph-worktree.sh`
- `docs/recipes/codex-setup.md`、`templates/base/docs/recipes/codex-setup.md`
- `docs/evidence/codex-config-rewrite-2026-09-30.md`(新規)

## Design decisions

- 検知は `validate_clean_base` の中で、dirty の一覧が `.codex/config.toml` 1 件だけのときに限って行う。判定は「作業ツリーのファイルからコメント行と空行を除いたもの」が「HEAD の版からコメント行と空行を除いたもの」に、`shell_environment_policy` の table(見出し行とその下の `key = value` 行)を末尾に足しただけと一致するかどうか。
- 止まる挙動は変えない(exit code は非 0 のまま)。変わるのは文言だけで、呼び出し元(`/plan`、`/spec`)の分岐は影響を受けない。
- 案内は戻す前に差分を見ることを先に書く(`git -C <root> diff -- .codex/config.toml`)。
- Critical forks: None(issue を open のまま残すか閉じるかは簡単に戻せる判断で、既定は open)

## Acceptance criteria

- [ ] AC-1: `.codex/config.toml` だけが未ステージで dirty で、差分がコメント行・空行の削除と末尾の `shell_environment_policy` の table の追加だけのとき、`ralph-worktree.sh validate-clean-base` と `ensure` が非 0 で止まり、stderr に書き換えの説明、`git -C <root> diff -- .codex/config.toml`、`git -C <root> checkout -- .codex/config.toml`、`git -C <root> status --porcelain` が空になることの確認が出る。template との `cmp` は案内に含めない。テストは案内のコマンドをそのまま実行して、その後 `validate-clean-base` が pass することまで確認する。
- [ ] AC-2: 次の場合は従来の `base branch '<base>' has uncommitted changes` で止まり、専用の文言は出ない: (a) 同じ書き換えに値の変更が 1 つ混ざる、(b) `.codex/config.toml` 以外にも dirty なファイルがある、(c) `shell_environment_policy` 以外の table が追加されている、(d) 未追跡のファイルだけがある、(e) 書き換えをステージした(`M `)、部分的にステージした(`MM`)、(f) HEAD の版に複数行文字列があり、その中の `#` 行か空行が消えている(Codex advisory の反例)。
- [ ] AC-3: コメントの削除だけ(table の追加なし)の差分と、table の追加だけ(コメントは残る)の差分も AC-1 と同じ扱いになる。コメントの書き換え・追加、インデントだけの変更、CRLF 化、行末コメント付きの見出しの追加は一般の文言になる(self-review cycle 1 の MEDIUM で改訂。判定は生の行で行い、HEAD の行はコメント行と空行だけを落とせる)。
- [ ] AC-4: clean な base では従来どおり pass。`scripts/ralph-worktree.sh` と template が byte 一致。
- [ ] AC-5: `tests/test-ralph-worktree.sh` に AC-1〜AC-4 のケースがあり、mutation(検知の関数が常に偽を返す、table 名の判定を外す)で落ちる。
- [ ] AC-6: `docs/recipes/codex-setup.md`(+ template)に症状・確認・戻し方(`git status --porcelain` が空になることの確認。template との `cmp` は ralph 本体を開発するときだけ)・調査の状況の段落があり、`docs/evidence/codex-config-rewrite-2026-09-30.md` に調査の記録(否定した候補とその根拠、probe の手順と結果、残る候補、次に起きたときに取る証拠: 書き換え直後の `stat` の時刻、実行中だった codex のプロセスと親、`~/.codex/log` の該当時刻)がある。
- [ ] AC-7: `shellcheck -S warning` で警告なし、`RALPH_VERIFY_SCOPE=full ./scripts/run-verify.sh` green、`./scripts/check-sync.sh` pass。

## Implementation outline

1. Slice A(implementer、sonnet): `validate_clean_base` の検知(関数 1 つ、awk で正規化)、テスト、recipe の段落、調査の記録。1 コミット。red の証拠: AC-5 の mutation。
2. pipeline: self-review → verify → test → sync-docs → cross-review → PR(`Refs #185`)。merge 後に #185 へ調査結果のコメントを書き、open のまま残す。

## Verify plan

- Static analysis checks: `shellcheck -S warning scripts/ralph-worktree.sh tests/test-ralph-worktree.sh`、`sh -n`、`RALPH_VERIFY_SCOPE=full ./scripts/run-static-verify.sh`、`./scripts/check-sync.sh`。
- Spec compliance criteria to confirm: AC-1〜AC-7 を該当行とテストの結果で確認。
- Documentation drift to check: `docs/recipes/codex-setup.md`、`docs/recipes/` の worktree の recipe、memory ではなく repo の記録に調査が残っていること。
- Evidence to capture: `docs/evidence/verify-*.log`、verify report の AC 表。

## Test plan

- Unit tests: `sh tests/test-ralph-worktree.sh`(fixture repo に `.codex/config.toml` を追跡させ、書き換えの形を作って `validate-clean-base` を呼ぶ)。
- Integration tests: `RALPH_VERIFY_SCOPE=full ./scripts/run-verify.sh`、fixture で `ensure` 経由でも同じ文言になること。
- Regression tests: AC-5 の mutation。
- Edge cases: ファイルが削除された(` D`)、rename、HEAD に `.codex/config.toml` がない repo、`shell_environment_policy.set` のような dotted table、CRLF の改行、末尾の改行の有無、ステージ済み / 部分ステージ、複数行文字列、template のない fixture(配布先と同じ形)で案内どおりに戻して clean になること。
- Evidence to capture: test report(件数、mutation 表)。

## Risks and mitigations

- 検知が広すぎて、手で入れた変更を「外部の書き換え」と案内してしまう: 判定を「コメント・空行の削除」と「`shell_environment_policy` の table の追加」だけに絞り、値の変更や他の table の追加は一般の文言にする。案内は戻す前に diff を見ることを先に書き、自動では戻さない。
- 原因が分からないまま再発する: 検知で止まる理由が分かるようにし、次に起きたときに取る証拠を記録に書く。issue は open のまま残す。

## Rollout or rollback notes

文言の追加だけで、止まる / 通る の判定は変えない。問題があれば 1 コミットを revert する。

## Open questions

- 原因(TUI のダイアログ、desktop app、バージョン更新時の移行のどれか)。次に起きたときの証拠で判断する。

## Deviation notes

- 2026-09-30 plan: Codex plan advisory(gpt-6-astra、xhigh、#192 の新しい呼び出し形で実行、rc 0、`-o` 2943 バイト)は MEDIUM 3 件。(1) ステージ済みの書き換えでは `git diff` が空、`git checkout --` が index から戻すので dirty のまま → 検知を未ステージの ` M` 1 行だけに限定し、案内どおりに戻して clean になることまでテストする。(2) 行単位の正規化は TOML の複数行文字列の中の `#` 行・空行を値として扱えず、設定の変更を書き換えと誤判定する → 複数行文字列の区切りがあれば検知しない。(3) 配布先には template がないので `cmp` の手順が使えない → 確認は `git status --porcelain` が空になることにし、`cmp` は ralph 本体の開発時だけ recipe に書く。ユーザー決定: 対応案で plan を更新
- 2026-09-30 work: Slice A は implementer(sonnet)に委譲(cd1aadb、6 ファイル、+716 / -4、push 済み)。`codex_config_external_rewrite_only`(awk で HEAD と作業ツリーを正規化して比較、複数行文字列があれば検知しない)を、porcelain がちょうど ` M .codex/config.toml` のときだけ `validate_clean_base` から呼び、一致したら専用の文言と戻し方(diff → checkout → status が空)で止まる。template は byte 一致。`tests/test-ralph-worktree.sh` に 13 ケース(86 / 0)。逸脱: 調査の記録は `docs/reports/` ではなく `docs/evidence/codex-config-rewrite-2026-09-30.md` に置いた(check-sync は `docs/reports/` のうち pipeline の接頭辞だけを除外し、`investigation-` は ROOT_ONLY で落ちる。`docs/evidence/` は除外済みで、#155 などの調査記録も同じ場所にある)。plan 内の参照 3 箇所も更新。red: 判定を常に偽 → 一致ケース 17 assertion が落ちる、table 名を任意に → ケース 6、複数行文字列のガードを外す → ケース 10、porcelain の制限を外す → ケース 5 / 7 / 8 / 9。`RALPH_VERIFY_SCOPE=full ./scripts/run-verify.sh` green、check-sync pass。orchestrator も 86 / 0、cmp、check-sync を確認
- 2026-09-30 self-review(cycle 1、b164556): CRITICAL 0 / HIGH 0 / MEDIUM 1 / LOW 9、merge 可。MEDIUM: 判定がコメント・空行・空白・改行コードの中の差分をすべて既知の書き換えと見なす(コメントの書き換え・追加、インデント、CRLF 化でも checkout を案内する)。LOW: 行末コメント付きの見出しが shell_environment_policy の table に吸収される、HEAD が 0 バイトだと `FNR == NR` で取りこぼす、案内のパスが引用なし、temp ファイル、テストのコメントの位置と fixture の重複、調査記録の結論の強さ、plugin の記述の範囲、`stat` の書式。全件を in-cycle で修正
- 2026-09-30 work: Slice B は implementer に委譲(0bb6aae、6 ファイル、+406 / -270、push 済み)。判定を生の行の照合にした(HEAD の行はコメント行と空行だけ落とせる、ほかは byte 単位で一致、末尾は空行と `[shell_environment_policy]` / `[shell_environment_policy.<name>]` の table だけ)。HEAD の版は stdin で awk に渡し `FILENAME == "-"` で区別(temp ファイルなし、空の HEAD も正しく扱う)、案内のパスは `printf -v qroot %q`。テストは case 11(CRLF)を一般の文言に変え、6 ケース追加(コメントの書き換え、追加、インデント、`[features2] # c`、空の HEAD、空白を含むパスで案内どおりに戻す)。105 / 0。調査記録: trust 済みの偽 project でも `codex features enable` はユーザーレベルだけを書く(再 probe)、「codex exec は原因ではない」は「15 回以上で再現せず、可能性は低い」に弱めた。逸脱: 見出しの判定は括弧の内側の空白も許さない(「その形だけ」の指示の帰結)。red: 5 種の変異がそれぞれ該当ケースで落ちる(%q を外すとテストが git のエラーで中断する)。orchestrator も 105 / 0 と cmp を確認
- 2026-09-30 verify(cycle 1、8e7a678): PASS。AC-1〜AC-7 をコードとテストで確認、mutation 4 種を独立に再現。`RALPH_VERIFY_SCOPE=full ./scripts/run-static-verify.sh` green、check-sync pass
- 2026-09-30 test(cycle 1、1c503ee): PASS。`RALPH_VERIFY_SCOPE=full ./scripts/run-test.sh` green、105 / 0 を sh で 3 回・dash・空白を含む TMPDIR で。scratch clone での end-to-end: 書き換えの形で `ensure` が専用の文言で止まり worktree も state も作らない → 案内どおりに戻すと `ensure` が通る、値の変更を混ぜる・stage する → 一般の文言、サブディレクトリから実行しても案内のコマンドが動く。mutation: 常に MATCH → 8 ケース、porcelain の制限を外す → 4 ケースが落ちる。追加 table の中のコメント行と `[[shell_environment_policy]]` の見出しは、どちらの変異もテストが捕まえない(穴)
- 2026-09-30 work: Slice C は implementer に委譲(d84cdab、テストだけ、+48)。ケース 20(追加 table の中のコメント行)と 21(`[[shell_environment_policy]]`)を追加し、どちらも一般の文言。111 / 0。red: 上の 2 つの変異がそれぞれ 20 と 21 で落ちる
- 2026-09-30 sync-docs(cycle 1、9b68c43): tech-debt に原因未特定の行を追加、他は drift なし。cross-review(cycle 1、HEAD 9b68c43、watchdog の 1 行で実行、`codex rc=0`、`-o` 965 バイト): ACTION_REQUIRED 1 件(AR-1: 追加部分の見出し判定が `/^\[/` だけなので、policy の table の後のインデントされた `  [features]` が見出しと見なされず、その下の実際の設定の追加まで書き換えと判定して checkout を案内する。空白でもタブでも再現)。ユーザー決定: 修正して pipeline を cycle 2/2 として再実行
- 2026-09-30 work(cycle 2): Slice D は implementer に委譲(ee397e9、3 ファイル、+45 / -4、push 済み)。追加部分の見出し判定を `/^[ \t]*\[/` にし、インデントされた見出しも見出しとして扱う。受け入れるのはインデントのない policy の見出しだけ(`is_sep_header` は変えない)なので、インデントされた見出しは policy でも一般の文言になる(安全側)。ケース 22(空白 2 つの `[features]`)、23(タブ)、24(インデントされた policy の見出し)を追加、120 / 0。red: 判定を `/^\[/` に戻すと 22 / 23 / 24、インデントされた policy の見出しを受け入れると 24 だけが落ちる。`RALPH_VERIFY_SCOPE=full ./scripts/run-verify.sh` green。orchestrator も 120 / 0 と cmp を確認
- 2026-09-30 self-review(cycle 2、f49ceb9): CRITICAL 0 / HIGH 0 / MEDIUM 0 / LOW 5、merge 可。cycle 1 の 10 件は HEAD で全部直っている。C2-1: 既知の変更が 1 つもない差分(モードだけ、末尾の改行の有無、末尾の空行だけ)も専用の文言になる。C2-2: `%q` は bash 3.2 と UTF-8 で非 ASCII の root を貼り付けられない形にする(cycle 1 の修正による退行)。C2-3: テストの期待値が引用しない root を前提にしている、ケース 19 の `bash -c` が失敗すると集計を出さずに止まる。C2-4: ECC が書く中身の記述が不正確。C2-5: コメントの直し残し。全件を in-cycle で修正
- 2026-09-30 work(cycle 2): Slice E は implementer に委譲(a2c721d、6 ファイル、+202 / -62、push 済み)。awk に `dropped` と `saw_policy_header` を持たせ、どちらもなければ NOMATCH。root の引用は `sq="'\\''"; qroot="'${root//\'/$sq}'"` の単一引用符の形。テストは期待値を同じ引用で計算し(bash の小さなスクリプトを実行して一致させる)、ケース 19 / 28 の復旧は `set +e` で守る。ケース 25(モードだけ)、26(末尾の改行を外す)、27(空行・空白だけを足す)は一般の文言、28(日本語と `'` を含むパス)は専用の文言で、案内のコマンドが bash と zsh の両方で戻せる。143 / 0。red: 条件を外すと 25 / 26 / 27、`%q` に戻すと 1 / 2 / 3 / 18 / 28、期待値を引用なしに戻すと 1 / 2 / 3 / 18 / 28 が落ちる(ロケール en_US.UTF-8、bash 3.2.57)。最終 cycle なので再レビューはせず、orchestrator が差分を読み 143 / 0 と cmp を確認。あわせて、調査の記録と tech-debt の行が参照する plan のパスを archive 側に直した(`/pr` で archive に移るため)
- 2026-09-30 pipeline(cycle 2/2): self-review LOW 5(Slice E で修正)→ verify pass(8fc3eb3)→ test pass(3186527。143 / 0 を sh・dash で、scratch clone で `ensure` の end-to-end、日本語と `'` を含むパスで bash / zsh とも案内どおりに戻る、cycle 2 の mutation 5 種がすべて該当ケースで落ちる、gawk は未導入で未確認)→ sync-docs drift なし(074ee7f)→ cross-review(HEAD 074ee7f、watchdog の 1 行、`codex rc=0`、`-o` 199 バイト)指摘 0 件。cap 到達だが Case C なので `/pr` へ。原因は未特定のまま(Non-goals)。PR は `Refs #185`、issue は open のまま残し、調査結果をコメントする
## Progress checklist

- [x] Plan reviewed
- [x] Branch created
- [x] Implementation started
- [x] Review artifact created
- [x] Verification artifact created
- [x] Test artifact created
- [ ] PR created

## Readiness checklist

- [x] プラグイン説を否定する根拠(ソース)と、codex の通常の設定変更がユーザーレベルに入ること(隔離 probe)を確認した
- [x] critical fork なし
- [x] Codex plan advisory(MEDIUM 3、対応案で plan を更新)
- [x] AC は fixture の shell テストで確認できる
