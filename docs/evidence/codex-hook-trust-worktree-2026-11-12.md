# codex が linked worktree で見る hook の承認の鍵(2026-11-12)

#310 の計画の前に、task worktree で動く codex が `hooks.state` のどの鍵を見るか(main のチェックアウトのパスか、worktree のパスか)を確かめた記録。

## 環境

- codex-cli 0.166.0(macOS、手元の `/opt/homebrew` の codex)
- ユーザーの `~/.codex` と herdr には触れていない。一時ディレクトリ `<S>`(このセッションの scratchpad の `hooktrust/`)に次を作った
  - `<S>/exp1/repo`: `git init -b main` した repo。`.codex/hooks.json` に `PreToolUse`(matcher `Bash`、command `echo pre`)と `SessionStart`(command `echo start`)を 1 つずつ置いてコミット
  - `<S>/exp1/wt`: `git -C repo worktree add <S>/exp1/wt -b wt` で作った linked worktree
  - `<S>/exp1/home`: 空の `CODEX_HOME`。`config.toml` には `[projects."<S>/exp1/repo"] trust_level = "trusted"` と、下の表の `hooks.state` だけを書いた
  - `<S>/link`: `<S>/exp1` への symlink
- 読み方: `CODEX_HOME=<S>/exp1/home codex app-server` を `<S>/exp1/run` で起こし、stdio の JSON-RPC で `initialize` のあと `hooks/list`(`cwds` を渡す)を呼んだ。応答の各 hook の `key`、`sourcePath`、`trustStatus`、`enabled`、`currentHash` を書き出した(手順のスクリプトは scratchpad の `hooks_list.py`)

## 1. 鍵の候補を main のパスだけにした場合

`hooks.state` に `"<S>/exp1/repo/.codex/hooks.json:pre_tool_use:0:0"` だけを置き、`trusted_hash` はわざと 0 を 64 個並べた値にした。

| cwd | key | sourcePath | trustStatus |
|---|---|---|---|
| `<S>/exp1/repo` | `<S>/exp1/repo/.codex/hooks.json:pre_tool_use:0:0` | `<S>/exp1/repo/.codex/hooks.json` | modified |
| 同 | `<S>/exp1/repo/.codex/hooks.json:session_start:0:0` | 同 | untrusted |
| `<S>/exp1/wt` | `<S>/exp1/repo/.codex/hooks.json:pre_tool_use:0:0` | `<S>/exp1/repo/.codex/hooks.json` | modified |
| 同 | `<S>/exp1/repo/.codex/hooks.json:session_start:0:0` | 同 | untrusted |

linked worktree の codex は、main のチェックアウトの `.codex/hooks.json` を読み、鍵も main のパスで引いた。仕込んだ鍵が当たったので `modified`(hash が違う)、鍵の無い `session_start` は `untrusted` になった。`currentHash` は両方の cwd で同じ値だった(`pre_tool_use` は `sha256:3ce3…70c2`、`session_start` は `sha256:0622…388d`)。

## 2. 正しい hash と `enabled = false`

`pre_tool_use:0:0` の `trusted_hash` を 1 の `currentHash` にし、`session_start:0:0` に `enabled = false` と正しい `trusted_hash` を置いた。

| cwd | key の末尾 | trustStatus | enabled |
|---|---|---|---|
| `<S>/exp1/wt` | `pre_tool_use:0:0` | trusted | true |
| 同 | `session_start:0:0` | trusted | false |
| `<S>/link/wt`(symlink 越しの worktree) | `pre_tool_use:0:0`(鍵は `<S>/exp1/repo/...`) | trusted | true |
| `<S>/exp1/repo/sub`(main の subdirectory) | `pre_tool_use:0:0`(鍵は `<S>/exp1/repo/...`) | trusted | true |

- `enabled = false` の handler は、承認済みでも動かない(`enabled: false`)
- symlink 越しに worktree を開いても、鍵は実パスの main になった。worktree の `.git` が実パスの gitdir を指すためとみる(下のソースの読みと合う)

## 3. symlink 越しに main のチェックアウトを開いた場合

| cwd | key | trustStatus |
|---|---|---|
| `<S>/link/repo` | `<S>/link/repo/.codex/hooks.json:pre_tool_use:0:0` | untrusted |
| 同 | `<S>/link/repo/.codex/hooks.json:session_start:0:0` | untrusted |

main のチェックアウトを symlink のパスで開くと、鍵もそのパスになり、実パスで承認した鍵は当たらない。

## ソースとの突き合わせ(openai/codex、tag `rust-v0.166.0`)

- 鍵: `codex-rs/hooks/src/lib.rs` の `hook_key` が `{key_source}:{event の snake_case}:{group}:{handler}` を作る。`hook_event_key_label` が `PreToolUse` を `pre_tool_use` にする(14 種)
- 番号: `codex-rs/hooks/src/engine/discovery.rs` の `append_matcher_groups` が group と handler を `enumerate` で数える。空の command などで読み飛ばす handler も番号を使う
- 状態: 同じファイルの `hook_trust_status` は、`trusted_hash` が無ければ untrusted、あって違えば modified。`hook_enabled` は `enabled = false` のときだけ false
- worktree: `codex-rs/config/src/loader/mod.rs` の `root_checkout_hooks_folder_for_dir` が、checkout の root(`.git` を持つ最も近い祖先)と repo の root(`codex-rs/git-utils/src/trust.rs` の `resolve_root_git_project_for_trust`、main のリポジトリの root)が違うとき、hook の folder を `<repo の root>/<相対 dir>/.codex` に向ける。`merge_root_checkout_project_hooks` は worktree の config から hooks を除き、root checkout の hooks に差し替える
- 未確認: この処理が入った codex の版。0.166.0 にはある

## 結論

- ralph の doctor と spawn は、main worktree の root の `.codex/hooks.json` を読み、`<その絶対パス>:<event>:<group>:<handler>` の鍵を探せばよい。task worktree や機能の org の worktree のパスの鍵は探さない
- 承認されていない状態は 2 つ: 鍵が無いか `trusted_hash` が無い(untrusted)、`enabled = false`(止めてある)。hash の食い違い(modified)は鍵の有無では分からない
- main のチェックアウトを symlink のパスで使うと鍵が変わる、という端のケースがある
