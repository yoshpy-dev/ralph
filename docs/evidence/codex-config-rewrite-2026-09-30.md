# 調査記録: `.codex/config.toml` の外部書き換え

- Date: 2026-09-30
- Related plan: `docs/plans/active/2026-09-30-codex-config-rewrite-detect.md`
- Related issue: #185
- Status: 原因未特定。open のまま残す。

## 症状

main のチェックアウトにある追跡ファイル `.codex/config.toml`(`templates/base/.codex/config.toml`
と byte 一致が `scripts/check-sync.sh` の要件)が、値は同じままコメントと空行を
すべて剥がされ、末尾に `[shell_environment_policy]`(`inherit = "core"`)と
`[shell_environment_policy.set]`(`CLAUDE_CODE_DISABLE_ADAPTIVE_THINKING = "1"`)
を足された状態に書き換わることがある。観測日は 2026-09-17、09-18 の 2 回。
その状態では `scripts/ralph-worktree.sh ensure`(`/plan`、`/spec` から呼ばれる)
が `validate_clean_base` の一律の「base branch 'main' has uncommitted changes」
で止まり、次の plan の worktree が作れない。

## 否定した仮説

### Claude Code の codex プラグイン (codex-companion)

memory と issue に記録されていた最有力の仮説は「`codex exec` の起動時に
codex-companion が書き込む」だったが、ソースを読む限り当たらない。

- `openai-codex` プラグイン(`~/.claude/plugins/cache/openai-codex/codex/1.0.2/scripts/`
  以下の `codex-companion.mjs`、`app-server-broker.mjs`、`session-lifecycle-hook.mjs`、
  `stop-review-gate-hook.mjs`)の `writeFileSync` 呼び出しは、state / job /
  broker / pid の JSON とログファイルだけを対象にしている。`config.toml` にも
  `shell_environment_policy` にも触れない。プラグインが登録する hook は
  SessionStart / SessionEnd / Stop の 3 つで、いずれも Codex の設定ファイルを
  書く処理を持たない。
- `everything-claude-code` プラグイン(`~/.claude/plugins/cache/everything-claude-code/`)
  の `scripts/sync-ecc-to-codex.sh` と `scripts/codex/merge-codex-config.js` は
  Codex 向けの設定をマージするが、対象は ECC 自身のフック定義であり、
  `shell_environment_policy` は生成しない。プラグインの `hooks.json` には
  codex 関連のイベント登録がない。

### ralph 自身のコード

ralph の Go コードで `.codex/config.toml` を参照しているのは
`internal/cli/doctor.go` の読み取り専用パス(`ralph doctor` の wiring チェック)
のみで、書き込みは一切行わない。

### codex の通常の設定変更サブコマンド

追加される table の中身(`CLAUDE_CODE_DISABLE_ADAPTIVE_THINKING=1`)は
ユーザーの `~/.claude/settings.json` とこのリポジトリの `.claude/settings.json`
の `env` に存在する値と一致する。2026-08-24 の plan(commit `38612ef`、
`codex-hooks-multi-event`)の Non-goals には「`shell_environment_policy` の
ユーザレベル移動 → メンテナ環境の main 差分ゼロ化(ローカル作業)」という
記述がある。つまりこの table はもともとメンテナが project の
`.codex/config.toml` に手で入れていたもので、その後ユーザーレベルの
config に移した。観測される書き換えは「ユーザーレベルに移した table が、
コメントを落とした再シリアライズとともに project 側に戻ってくる」形をしている。

これが codex の通常の設定変更(`codex features enable` などの config
書き込みサブコマンド)によるものかどうかを、隔離した環境で確認した。

**probe 手順**(実際の `~/.codex` は使わない):

1. scratch ディレクトリに偽の `HOME` / `CODEX_HOME` を用意する。
2. 偽ユーザーレベルの config に `shell_environment_policy` テーブルと
   コメントを持つ内容を置く。
3. 別の scratch ディレクトリに project を模した `.codex/config.toml`
   (このリポジトリの template 相当のコメント付き内容)を置く。
4. project ディレクトリの中で、config を書き込む codex のサブコマンド
   (`codex features enable <feature>`)を実行する。

**結果**: 書き込まれたのはユーザーレベルの config のみだった。コメントは
残ったまま、`[features]` テーブルが追記されただけで、`shell_environment_policy`
には触れなかった。project 側の `.codex/config.toml` は変化しなかった。
→ codex の通常の設定変更コマンドは、観測された書き換えの原因ではない。

### `codex exec` そのもの

2026-09-27〜30 のこの session 中、worktree から `codex exec` /
`codex exec review` を 15 回以上実行したが、書き換えは一度も発生しなかった。
→ `codex exec` の起動・終了そのものは原因ではない。

## 残る候補(未確認)

自動での再現が難しい対話的な操作が候補として残る。

- codex の TUI のダイアログで設定を保存する経路(project の trust 確認、
  hook の trust 確認、model 移行の確認など)。
- Codex の desktop app または IDE 拡張の設定 UI。
- codex のバージョン更新時に走る設定移行処理。

観測日の 2026-09-17〜18 は、codex の TUI 座席で実機確認をしていた時期
(issue #155、#162 の作業期間)と重なる。この符合は確度の高い手がかりでは
あるが、因果関係は未確認。

## 次に発生したときに取る証拠

書き換えに気づいた時点で、`git checkout -- .codex/config.toml` で戻す前に
以下を記録する。

```sh
# 書き換わったファイルの最終更新時刻
stat -f '%Sm %N' .codex/config.toml

# そのとき動いていた codex 関連プロセスと親プロセス
ps -axo pid,ppid,lstart,command | grep -i codex

# その時刻付近の codex のログ(オペレーターが確認・保管する。
# 実際の ~/.codex 以下を自動ツールで読み取らないこと)
#   $CODEX_HOME/log/ もしくは ~/.codex/log/
```

加えて、直前に答えた codex の対話的ダイアログ(trust、hook trust、model
移行の確認など)があれば、その内容と回答をメモする。これらの証拠が
揃えば、残る候補のどれが原因かを切り分けられる可能性が高い。

## この調査が変えたこと

原因そのものは特定できなかったが、`scripts/ralph-worktree.sh` の
`validate_clean_base` に検知を追加し、書き換えの形(コメント・空行の削除、
および/または末尾への `[shell_environment_policy]` テーブルの追加)に一致する
ときだけ、専用の理由と戻し方を表示して止まるようにした。原因不明のまま
再発しても、影響(worktree 作成の停止)と復旧手順はその場で分かるようにする
ための対応。詳細は `docs/plans/active/2026-09-30-codex-config-rewrite-detect.md`
と `docs/recipes/codex-setup.md` を参照。
