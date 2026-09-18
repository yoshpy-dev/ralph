# codex 座席の permission mode 実機検証(autonomous / edits)

- Date: 2026-09-18(JST 15:36〜16:19、UTC 06:36〜07:19)
- Issue: #155
- Plan: docs/plans/archive/2026-09-18-codex-seat-permission-verification.md(PR 作成時に active から移動)
- 生ログ: セッションの scratchpad(`<scratch>/codex-perm/logs/run-auto.log`、`run-edits.log`、各 org の `state-*/manifest.jsonl`)に残したが、リポジトリにはコミットしていない(pane 抜粋と manifest の要点は本ファイルに転記済み)。ホームディレクトリは `~`、scratchpad は `<scratch>` に置換し、session UUID は伏せた。

## 環境

| 項目 | 値 |
|---|---|
| codex | codex-cli 0.154.0(`~/.local/share/mise/installs/node/24.15.0/bin/codex`、npm 配布) |
| herdr | 0.7.5、protocol 17。`herdr server`(ヘッドレス)で socket を起動 |
| agmsg | 1.1.13(`~/.agents/skills/agmsg`、DB は `~/.agents/skills/agmsg/db/messages.db`) |
| ralph | worktree `docs/codex-seat-permission-verification` @ e38dc27 から `go build` した一時バイナリ(インストール済み 5.1.0 は repo より古い) |
| scratch cwd | `<scratch>/codex-perm/cwd`(`git init` 済みの空リポジトリ、本 repo 外) |
| ralph.toml(autonomous) | `[org] driver_pool = ["claude","codex"]`、`max_seats = 2`、`[org.permissions] default = "autonomous"`、`codex_verified = true` |
| ralph.toml(edits) | 上に加えて `[org.permissions.roles] reviewer = "edits"` |
| 座席の実効 codex config | 下記「実効設定」参照 |

### 実効設定(advisory HIGH-2 対応)

`~/.codex/config.toml`(検証時点)の該当キー: `model = "gpt-6-astra"`、`model_reasoning_effort = "max"`、`sandbox_mode = "danger-full-access"`、`approval_policy = "never"`、`[tui.model_availability_nux] "gpt-5.5" = 2`、`[notice.model_migrations] "gpt-5.2" = "gpt-5.4"`。`[sandbox_workspace_write]` なし。scratch cwd に `.codex/config.toml` なし。

座席は herdr server を偽 HOME(後述)で起動して動かしたため、座席の `CODEX_HOME` は `<fake-home>/.codex`。そこに置いた config は run ごとに切り替えた:

- **minimal**: `model = "gpt-5.5"` + scratch cwd の `trust_level = "trusted"`(+ Run A2 以降は `[sandbox_workspace_write] writable_roots = ["<home>/.agents/skills/agmsg/db"]`、絶対パス)。approval policy / sandbox mode は codex の既定
- **real-copy**: 上記 `~/.codex/config.toml` の完全なコピー + trust + writable_roots(Run E1)

## 前提として判明した環境依存の問題

### P1. ユーザーの shell alias が herdr pane に効き、`--model` が二重になる

`~/.zshrc`(`~/.config/zsh/.zshrc` への symlink)に `alias codex="codex -m gpt-6-astra -c '...'"` と `alias claude="claude --model fable ..."` がある。herdr は pane の対話 zsh に `codex --sandbox ... --model gpt-5.5 '<prompt>'` をそのまま送るため alias が展開され、codex が `error: the argument '--model <MODEL>' cannot be used multiple times` で即終了、`ralph org spawn` は `agent_start` の timeout(retry 3 回)で `spawn_failed` になる(Run A の 1〜3 回目)。

- `ZDOTDIR` を差し替えた環境で `herdr server` を起動しても効かない(pane の zsh は `/etc/zshenv` の `ZDOTDIR=$HOME/.config/zsh` を読む)
- 回避: `herdr server` を **偽 HOME**(alias なしの `.zshrc`、`.config/zsh/.zshrc` も alias なし、`.config/herdr` は実体への symlink で socket を共有、`.agents` / `.local` は実体への symlink、`.codex` に `auth.json` と最小 config を複製)で起動した。ユーザーの dotfile は変更していない(`~/.zshrc` は backup と byte 一致を確認)
- claude 座席も同じ alias で同じ衝突が起きるはず(未検証。#155 の範囲外)

### P2. `ralph org send` の本文が codex の入力欄に残り送信されないことがある

座席が idle のときに `ralph org send` で複数行の TASK を送ると、本文は貼り付けられるが送信されず、`herdr pane send-keys <pane> Enter` をもう一度送ると submit された(Run A の t-auto-2)。座席が busy のときは "Queued follow-up inputs" に入り、前のターン終了後に自動送信された(t-auto-1)。ralph は `pane send-text` の直後に `send-keys Enter` を送っている(`internal/org/verbs.go` Send)ので、codex TUI の貼り付け処理との timing 問題。以降の run では send の 3 秒後に Enter を追加で送った。

### P3. workspace-write 下の codex 座席は agmsg に書けない

`--sandbox workspace-write` の writable root は cwd と `/tmp` 系のみ。agmsg の SQLite DB(`~/.agents/skills/agmsg/db/messages.db`)は外にあるため、座席の `send.sh` が `Runtime error near line 1: attempt to write a readonly database (8)` で失敗し、RESULT が lead に届かない(Run A、t-auto-2。座席は代わりに BLOCKED を pane に印字した)。codex config に `[sandbox_workspace_write] writable_roots = ["<home>/.agents/skills/agmsg/db"]`(実際は絶対パス。`~` 表記は未検証)を足すと送信できる(Run A2 以降)。**`codex_verified = true` で autonomous / edits を有効にしても、この設定なしでは typed RESULT が返らない。**

### P4. scratchpad が `/tmp` 配下だと「cwd 外」テストにならない

Run A の t-auto-1 で座席の `$HOME`(偽 HOME、`/private/tmp/...` 配下)への書き込みが成功したのは、`/tmp` が writable root だから(advisory HIGH-1 のとおり)。以降は実ホーム `~/codex-perm-outside-*.txt` を対象にした。

### P5. gpt-5.5 の退役ダイアログ / 自動移行

minimal config では起動直後に「GPT-5.5 retires on October 14, 2026 … 1. Try new model / 2. Use existing model」の対話ダイアログが出る(herdr の `agent wait --until idle` はこれを idle と判定する)。「2」を選ぶと codex は `[notice.model_migrations] "gpt-5.5" = "gpt-5.6-sol"` と `[tui.model_availability_nux]` を config に書き、以後はダイアログなし。real-copy config(Run E1)ではダイアログなしでステータス行が `gpt-5.6-sol · low` になり、`--model gpt-5.5` は実効モデルとして尊重されなかった(codex 側の退役移行)。ralph の receipts が追う `honored` はこのケースを拾う対象。

## Run A / A2: autonomous(`--sandbox workspace-write --ask-for-approval never`)

- org: `codex-perm-auto-0918e`(A)、`codex-perm-auto-0918f`(A2)。role reviewer、`--model gpt-5.5`、config minimal
- manifest `spawned` Details: `scope=<scratch>/codex-perm/cwd/** permission_mode=autonomous`(実引数は記録されない)
- 子プロセス(`ps`): `node ~/.local/share/mise/installs/node/24.15.0/bin/codex --sandbox workspace-write --ask-for-approval never --model gpt-5.5 '役割指示を読み込んで従ってください: <state>/prompts/<org>_reviewer.md'`
- 起動直後の pane: P5 のダイアログのみ(承認 / trust / login のダイアログなし)
- TASK t-auto-1(A): `printf 'ok\n' > hello.txt` → 成功、`shasum -a 256 hello.txt` → `dc51b8c9…fc22`(lead 側で同一ハッシュを独立確認)、偽 HOME への書き込み → 成功(P4)。承認プロンプトなし。RESULT は pane に印字のみ(P3 の前段)
- TASK t-auto-2(A): 実ホーム `~/codex-perm-outside-auto-2026-09-18.txt` への書き込み → `zsh:1: operation not permitted`(sandbox 拒否、承認プロンプトなし)。agmsg send → readonly database(P3)。座席は BLOCKED を印字
- TASK t-auto-3(A2、writable_roots 追加後): 同じ実ホームパス → `operation not permitted`(承認プロンプトなし)。**RESULT が agmsg で lead に届いた**(`TYPE: RESULT / TASK_ID: t-auto-3 / SEVERITY: LOW / EVIDENCE: command … outcome: zsh:1: operation not permitted … / SUMMARY: No approval prompt appeared; the write did not succeed.`、07:13:02Z、送信から 50 秒以内)
- 事後確認: `~/codex-perm-outside-auto-2026-09-18.txt` は存在しない。`stop` → `disband` → `status` は `stopped`、`herdr agent list` は空、pane close 済み

**判定: pass**(承認ダイアログで止まらない、sandbox は cwd 外を拒否する、TASK → RESULT の E2E が agmsg で通る)。

## Run E1: edits(`--sandbox workspace-write` のみ)、real-copy config

- org: `codex-perm-edits1-0918`。manifest Details `permission_mode=edits`
- 子プロセス: `codex --sandbox workspace-write --model gpt-5.5 '…'`(`--ask-for-approval` なし)
- 座席の実効 config は `approval_policy = "never"`、`sandbox_mode = "danger-full-access"` を継承
- TASK t-edits-1: `notes.md` 作成 → 承認なしで成功。実ホーム `~/codex-perm-outside-edits-2026-09-18.txt` → `operation not permitted`(承認プロンプトなし)。RESULT は agmsg で到達(07:14:29Z)
- 事後確認: 実ホームにファイルなし。cleanup 完了

**判定: partial(継承により承認プロンプトは出ない)**。ただし重要な肯定所見: CLI の `--sandbox workspace-write` は config の `sandbox_mode = "danger-full-access"` を上書きし、cwd 外は拒否される。また E1 では退役移行で実効モデルが `gpt-5.6-sol` になった(P5)。

## Run E2 / E2b: edits、minimal config(approval policy は codex 既定 = on-request)

- org: `codex-perm-edits2-0918`。子プロセス引数は E1 と同じ
- TASK t-edits-2(「拒否されたら回避しない」指示): `notes.md` → 承認なしで成功。実ホーム `~/codex-perm-outside-edits2-2026-09-18.txt` → `operation not permitted`、承認プロンプトなし(model が escalation を要求しなかった)。RESULT は agmsg で到達
- TASK t-edits-3(「拒否されたら sandbox 外実行の承認を要求せよ」指示): sandbox 拒否のあと **承認プロンプトが pane に表示された**:

  ```
  Would you like to run the following command?
  Environment: local
  Reason: Do you want to allow writing 'outside' to ~/codex-perm-outside-edits2-2026-09-18.txt outside the sandbox, as requested for t-edits-3?
  $ printf outside > …
  › 1. Yes, proceed (y)
    2. Yes, and don't ask again for commands that start with `printf outside > …` (p)
    3. No, and tell Codex what to do differently (esc)
  ```

  lead が Esc で拒否 → "Conversation interrupted" → 追加指示で RESULT を agmsg 送信(`SUMMARY: An approval prompt appeared after the sandbox denial and was denied by lead, so the outside write did not succeed.`)
- 事後確認: 実ホームにファイルなし。cleanup 完了

**判定: pass(機構として)**。edits では sandbox 外の実行を model が要求したときだけ承認プロンプトが出る。`on-request` は「model が必要と判断したとき」なので、プロンプトが出るかどうかは model の判断に依存し、「cwd 外は必ず問い合わせる」とは言えない。cwd 外の書き込み自体は常に sandbox が拒否する。

## 結論(issue #155 のやること 1〜2 に対して)

| 観点 | 結果 |
|---|---|
| autonomous 座席がツール使用時に承認ダイアログで止まらない | **成立**(Run A / A2) |
| edits 座席がファイル編集は自動承認する | **成立**(cwd 内。Run E1 / E2) |
| edits 座席がそれ以外を問い合わせる | **条件付き**: sandbox 外実行を model が要求したときにプロンプトが出る(E2b)。ユーザー config の `approval_policy = "never"` を継承すると出ない(E1)。cwd 外の書き込み自体は sandbox が常に拒否 |
| herdr pane 内の codex 座席で TASK → RESULT の E2E | **成立**(A2 / E1 / E2。ただし agmsg DB を writable root に足す必要あり = P3) |
| `--sandbox workspace-write` は config の `danger-full-access` を上書きする | **成立**(E1) |

`codex_verified` の既定は false のまま(ユーザー決定)。opt-in の条件と手順は `docs/recipes/codex-seat-permissions.md`。

## 後始末の記録

- 実 `~/.codex/config.toml` に一時追記した scratch cwd の `trust_level` は backup から復元(byte 一致を確認)
- 偽 HOME に複製した `auth.json` は削除。`~/.zshrc` は無変更(backup と byte 一致)
- `herdr server stop` 済み。`herdr agent list` 空、検証 pane はすべて close
- 実ホームの `codex-perm-outside-*` は存在しない(3 回とも sandbox 拒否)
