# Walkthrough: codex-config-rewrite-detect (#185)

- Date: 2026-09-30
- Plan: docs/plans/active/2026-09-30-codex-config-rewrite-detect.md
- Branch: fix/codex-config-rewrite-detect(base main 5efd3d6)
- Diff: 14 files、+2287 / -4(plan・報告・insight を除くと 7 files、+1082 / -4。うちテストが大半)

## 何を直したか

main のチェックアウトの `.codex/config.toml` が、コメントを剥がされ、末尾に `[shell_environment_policy]` の table を足された形に書き換わることがある。そうなると `scripts/ralph-worktree.sh ensure` は「base branch 'main' has uncommitted changes」で止まり、理由が分からないまま次の plan に進めない。

この PR で入れたのは次の 3 つ。

- 書き換えがこの形だけのときに、理由と戻し方を示して止まるようにした。
- 症状と戻し方を recipe に書いた。
- 原因の調査の記録を残した。

原因はまだ特定できていない(issue #185 は open のまま残す)。

## 読む順

1. `scripts/ralph-worktree.sh`(template と byte 一致)の `codex_config_external_rewrite_only`:
   - 呼ばれる条件: `git status --porcelain` がちょうど ` M .codex/config.toml` のとき(未ステージの変更 1 件)だけ。
   - HEAD の版の渡し方: stdin で awk に渡し、`FILENAME == "-"` で作業ツリーの版と区別する。
   - 照合: 生の行のまま、空白の除去も CR の除去もしない。HEAD の行はコメント行と空行だけを落とせる。
   - 末尾に足してよい行: 空行、インデントのない `[shell_environment_policy]` / `[shell_environment_policy.<name>]` の見出し、その下の key 行だけ。インデントされた見出しも見出しとして扱い、policy のものでも受け入れない。
   - 追加の条件: HEAD の行を 1 つ以上落としたか、policy の見出しを 1 つ以上足したかのどちらかが要る。
   - 検知しない場合: どちらかの版に複数行文字列(`"""` / `'''`)があれば検知しない。
2. `validate_clean_base`: 上の判定に一致したときは、専用の文言と 3 つのコマンドを出して止まる。コマンドは diff を見る、checkout で戻す、status が空になることを確かめる、の順。root のパスは POSIX の単一引用符で囲む。日本語や `'` を含むパスでも、そのまま貼り付けられる。一致しないときは、これまでどおりの文言で止まる。どちらの場合も自動では戻さない。
3. `tests/test-ralph-worktree.sh`: 28 ケースを追加した(143 assertion)。
   - 専用の文言になるもの: 記録された形、コメントを消しただけ、table を足しただけ、空の HEAD、空白や日本語と `'` を含むパス。
   - 一般の文言になるもの: 値の変更、他の dirty なファイル、policy 以外の table、未追跡だけ、stage 済み、部分 stage、複数行文字列、CRLF 化、コメントの書き換えと追加、インデントの変更、行末コメント付きの見出し、追加した table の中のコメント、`[[...]]`、インデントされた見出し、モードだけの変更、末尾の改行、空行だけの追加。
   - 専用の文言のケースでは、案内のコマンドを実際に実行して、元に戻ることまで確かめる。
4. `docs/recipes/codex-setup.md`(template と一致): 症状、確認、戻し方(status が空になること。template との `cmp` は ralph 本体の開発時だけ)、原因の状況。
5. `docs/evidence/codex-config-rewrite-2026-09-30.md`: 調査の記録。
   - 否定した候補: Claude Code の codex プラグインと everything-claude-code プラグインはどちらも project の `.codex/config.toml` を書かない(ソースで確認)。ralph も書かない。
   - 隔離環境での probe: `codex features enable` はユーザーレベルの config だけを書き、コメントを残した(trust 済みの project でも同じ)。
   - 残る候補と、次に起きたときに取る証拠。
6. `docs/tech-debt/README.md`: 原因が未特定であることを 1 行記録した。

## 検証

| 項目 | 結果 |
|---|---|
| `sh tests/test-ralph-worktree.sh`(sh・dash) | 143 / 0 |
| `RALPH_VERIFY_SCOPE=full ./scripts/run-test.sh` / `run-static-verify.sh` | green |
| shellcheck、`bash -n`、check-sync、cmp | clean / pass |
| scratch clone で `ensure` の end-to-end | 書き換えで止まり、案内どおりに戻すと通る |
| 日本語と `'` を含むパス | bash と zsh の両方で、案内のコマンドで戻る |
| mutation(cycle 1 と 2 の計 9 種) | すべて該当ケースで落ちる |
| gawk | 未導入で未確認 |

## pipeline の履歴

- plan: 事前の調査で、issue の「プラグインが書く」という仮説をソースで否定した。Codex advisory の MEDIUM 3 件を plan に反映した(stage 済みの差分、複数行文字列、配布先に template がないこと)。
- cycle 1:
  - Slice A で検知を入れた。
  - self-review は MEDIUM 1 / LOW 9。検知が広すぎたので、Slice B で生の行の照合に狭めた。
  - test で見つかった穴 2 つは Slice C で塞いだ。
  - cross-review は AR-1(インデントされた見出しの誤判定)。ユーザー決定で修正し、再実行した。
- cycle 2:
  - Slice D で AR-1 を直した。
  - self-review は LOW 5。Slice E で、既知の変更が 1 つ以上あることの条件と、非 ASCII のパスでも壊れない引用を入れた。
  - verify、test、sync-docs はどれも pass で、cross-review の指摘は 0 件だった。

## 残る gap

- 原因は未特定。次に起きたら、調査の記録の手順で証拠を取る。
- `FILENAME == "-"` は BWK awk、mawk、BusyBox の awk で確認したが、gawk では未確認。
