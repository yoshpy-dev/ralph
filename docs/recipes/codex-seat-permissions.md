# Codex 座席の permission mode を実機検証して `codex_verified` を有効にする

`[org.permissions].codex_verified` は既定 `false` で、codex 座席は `guarded`(codex CLI の既定の対話モード)以外を fail-closed で拒否する。`true` にすると autonomous → `--sandbox workspace-write --ask-for-approval never`、edits → `--sandbox workspace-write` のマッピングが有効になる。このマッピングの挙動は codex CLI のバージョンとユーザー config に依存するので、**マシンごとに**この手順で検証してから `true` にする。メタリポでの検証結果は `docs/evidence/codex-seat-permissions-2026-09-18.md`(codex-cli 0.154.0、herdr 0.7.5、agmsg 1.1.13)。

## 前提

- `ralph doctor` で codex / herdr / agmsg が pass していること
- herdr の socket が上がっていること(TUI を開くか `herdr server` をヘッドレスで起動)
- **shell alias に注意**: `~/.zshrc` 等で `alias codex="codex -m ..."` のようにモデル指定を含む alias を定義していると、herdr は pane の対話シェルにコマンドをそのまま送るため alias が展開され、`ralph org spawn --model` と衝突して `error: the argument '--model <MODEL>' cannot be used multiple times` で座席が起動しない(`spawn_failed`、`agent_start` の timeout)。検証中は alias を外すか、alias を読まない HOME / rc で herdr を起動する(`claude` の alias も同様)
- **agmsg の DB を writable root に足す**: `--sandbox workspace-write` の writable root は cwd と `/tmp` 系だけで、agmsg の SQLite DB(`~/.agents/skills/agmsg/db/`)は外にある。この設定がないと座席は `attempt to write a readonly database` で RESULT を送れない。`~/.codex/config.toml` に次を追加する:

  ```toml
  [sandbox_workspace_write]
  writable_roots = ["<your home directory>/.agents/skills/agmsg/db"]
  ```

  絶対パスで書く(`~` は展開されない。`echo $HOME/.agents/skills/agmsg/db` の出力を使う)。

- 検証用の cwd は **`/tmp` 配下に置かない**(`/tmp` は writable root なので「cwd 外」の否定テストにならない)。`$HOME` 配下の使い捨てディレクトリを `git init` して使う

## スクラッチ設定

`ralph.toml` を本番の設定と分けて用意する(`--config` で渡す)。

```toml
# autonomous 用
[org]
driver_pool = ["claude", "codex"]
max_seats = 2

[org.permissions]
default = "autonomous"
codex_verified = true
```

edits 用は上に `[org.permissions.roles] reviewer = "edits"` を足す。state も `--state-dir` で使い捨てにする。

## 手順

各モードで 1 座席ずつ spawn し、次を観測する。実際の CLI 引数は manifest には残らない(`spawned` の Details は `permission_mode=<mode>` まで)ので、`ps -axo args= | grep 'codex --sandbox'` で子プロセスのコマンドラインを控える。

1. **autonomous**

   ```sh
   ralph org spawn --org-id perm-auto --id reviewer --role reviewer --driver codex \
     --model <slug> --cwd <scratch> --scope "<scratch>/**" \
     --config <scratch>/ralph-autonomous.toml --state-dir <scratch>/state-auto
   ```

   - 子プロセス引数に `--sandbox workspace-write --ask-for-approval never` があること
   - 起動直後の pane(`herdr pane read <pane> --lines 40`)に承認 / trust / login のダイアログが出ていないこと。モデル退役の通知ダイアログ(「Try new model / Use existing model」)は出ることがある(承認とは別。選択は config に永続化される)
   - `ralph org send --to reviewer` で TASK を送る。座席が idle のとき本文が入力欄に残ることがあるので、数秒後に `herdr pane send-keys <pane> Enter` を一度送る
   - TASK の内容: cwd 内にファイルを 1 つ作る → `$HOME` 直下の使い捨てパスへ書く(拒否されるはず。**再試行や回避をしないよう明示**)→ RESULT を agmsg で送る(座席は skill の `send.sh` を使う)
   - 期待: 承認プロンプトなし、cwd 外は `operation not permitted`、RESULT が lead に届く(`bash ~/.agents/skills/agmsg/scripts/history.sh ralph-perm-auto lead 20`)。`$HOME` 直下のファイルが存在しないことを lead 側で独立に確認

2. **edits**

   - 子プロセス引数が `--sandbox workspace-write` のみ(`--ask-for-approval` なし)であること
   - 同じ TASK で cwd 内の編集が承認なしで完了し、cwd 外が拒否されること
   - 「拒否されたら sandbox 外実行の承認を要求せよ」と指示した追加 TASK で、pane に `Would you like to run the following command? … 1. Yes, proceed (y) / 2. Yes, and don't ask again … / 3. No … (esc)` が出ること。Esc で拒否し、座席に RESULT を送らせる
   - **注意**: approval policy を省略した edits は `~/.codex/config.toml` の `approval_policy` を継承する。`"never"` になっていると承認プロンプトは一切出ず、autonomous と同じ挙動になる。検証時の `approval_policy` / `sandbox_mode` / `writable_roots` の値を evidence に控える
   - **注意**: プロンプトが出るのは model が escalation を要求したときだけ(`on-request`)。「cwd 外は必ず問い合わせる」ではなく「cwd 外は sandbox が常に拒否し、model が要求すれば問い合わせる」

3. **後始末**: `ralph org stop --seat reviewer` → `ralph org disband` → `ralph org status --org-id <id>` に active なし → `herdr agent list` が空 → pane を close。`$HOME` 直下の使い捨てファイルが残っていないことを確認

## 判定と opt-in

- autonomous: 承認プロンプトなし + cwd 外拒否 + RESULT 到達 → **pass**
- edits: cwd 内が承認なし + cwd 外拒否 + (escalation 要求時に)承認プロンプト → **pass**。`approval_policy = "never"` を継承してプロンプトが出ない場合は **partial**(edits を使う意味がない構成なので、autonomous を使うか config を見直す)
- どちらかが **inconclusive**(座席が tool call に到達しない、pane が読めない)なら `codex_verified` は `false` のまま

pass したら、自分の `ralph.toml` で `[org.permissions] codex_verified = true` にする。この opt-in は「検証時の codex バージョン + `~/.codex/config.toml` の実効設定」に対するものなので、codex を更新したとき、`approval_policy` / `sandbox_mode` / `writable_roots` を変えたときは再検証する。`ralph doctor` は codex_verified の妥当性を検査しない。

## メタリポでの結果(2026-09-18)

codex-cli 0.154.0 で autonomous = pass、edits = pass(機構)。加えて、CLI の `--sandbox workspace-write` はユーザー config の `sandbox_mode = "danger-full-access"` を上書きした。詳細と判明した環境依存の問題(alias 衝突、`ralph org send` の Enter、agmsg DB の writable root、退役モデルの自動移行)は evidence を参照。
