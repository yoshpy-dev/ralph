# `ralph org send` の Enter timing の実機確認(2026-09-19)

issue #163 の受け入れ条件「idle 座席への複数行 `ralph org send` が追加操作なしで submit される」の実機 evidence。修正前(main)と修正後(本ブランチ)のバイナリを、同じ座席・同じ本文で比較した。

## 環境

- macOS(darwin 25.5.0)、herdr 0.7.5、codex-cli 0.154.0(model `gpt-6-astra`)、claude 2.1.274(model `haiku`)
- ralph: 修正前 = main `f006200`、修正後 = `c990363`(テキスト送信と Enter の間に既定 750ms の待ち、Enter の後に herdr の状態で submit を確認、Enter の再送なし)
- 座席の permission mode は guarded(`[org.permissions] default = "guarded"`)。作業ディレクトリはスクラッチの git リポジトリ
- 計測スクリプト: 座席が idle / done になるのを待って 2 秒置き、9 行の typed TASK(本文 5 行 + 1 語で答える指示)を `ralph org send` で送る。送信後 8 秒間、herdr の状態が working / blocked になるかを 0.5 秒間隔で見る。ならなければ画面に応答が出ていないかを確認し、それもなければ「submit されなかった」と判定して Enter を手で 1 回送る(次の計測を空の入力欄から始めるため)

## 結果

| 座席 | バイナリ | 追加操作なしで submit | 手動の Enter が必要 | `sent` の Details |
|---|---|---|---|---|
| codex | 修正前 | 2 / 5 | 3 / 5 | 接頭辞なし(旧形式) |
| codex | 修正後(待ち 750ms) | 5 / 5 | 0 / 5 | 5 件とも接頭辞なし(submit を確認)。stderr の注意なし |
| claude | 修正前 | 5 / 5 | 0 / 5 | 接頭辞なし(旧形式) |
| claude | 修正後(待ち 750ms) | 5 / 5 | 0 / 5 | 5 件とも接頭辞なし(submit を確認)。stderr の注意なし |

- 不具合は codex の TUI で、毎回ではなく timing に依存して起きる。上の 5 回とは別に、4 行の短い本文を修正前のバイナリで 1 回送ったときは submit された。#155 の記録(P2)と同じ「本文が入力欄に残り、Enter を追加すると submit される」状態を 3 / 5 で再現した
- claude の TUI では修正前でも再現しなかった。修正後も退行はなく、submit の確認(working の観測)も 5 / 5 で成立した
- 既定の待ち 750ms で codex は 5 / 5。既定値の変更は不要と判断した

## 承認ダイアログ表示中の状態(Enter を再送しない根拠)

guarded の codex 座席に、sandbox の外への `touch` を 1 つだけ実行する TASK を送った。

- 送信後 2 秒・4 秒は `working`、6 秒で `blocked` になった
- 画面は承認ダイアログで、選択肢 1「Yes, proceed (y)」が選択済み、末尾に「Press enter to confirm or esc to cancel」と表示されていた
- つまり、submit 済みの座席に Enter をもう 1 回送ると、この承認を通してしまう。plan の Design decisions(Codex plan advisory HIGH、ユーザー決定)のとおり、`Send` は Enter を 1 回しか送らない。`blocked` は「submit 済み」として扱う
- ダイアログは Escape で拒否した。状態は `done` に戻り、対象のファイルは作られていない

claude 座席でも、フォルダの信頼確認とファイル読み取りの許可のダイアログ表示中、herdr は `blocked` を返した。

## 分離の方法と、途中で起きたこと

ユーザー自身の herdr server(別プロジェクトの workspace を持つ)が動いていたため、それには触れず、専用の server を別の socket で起動した。

- herdr は `HOME` を変えても設定・状態ディレクトリを切り替えない。1 回目は `HOME` と `HERDR_SOCKET_PATH` だけを変えて起動したところ、専用 server がユーザーの `~/.config/herdr/session.json` を読んでユーザーの workspace を復元し、ユーザーの `herdr-server.log` に起動ログを 8 行追記した。保存状態を上書きさせないよう、この server は SIGKILL で止めた。`session.json` は更新時刻・ハッシュとも不変、ユーザーの server と pane も不変であることを確認した。残った影響は上のログ 8 行と、agent 検出キャッシュ(`~/.local/state/herdr/agent-detection/status.toml`)の更新
- 正しい分離は `XDG_CONFIG_HOME` / `XDG_STATE_HOME` / `HERDR_SOCKET_PATH` の 3 つを専用の場所に向けること。以降はこの形で起動し、開いているファイルが専用ディレクトリだけであること、ユーザーの `session.json` とログが変わらないことを起動のたびに確認した。unix socket のパス長の制限があるため、socket と XDG のディレクトリだけは `/tmp/r163/` に置き、終了後に削除した
- codex 座席: 専用 server を偽 HOME(alias のない rc)で起動し、`~/.codex/auth.json` を偽 HOME に一時複製した(shell alias が `--model` を重複させて座席が起動しない問題の回避。#155 と同じ)
- claude 座席: 偽 HOME では macOS の keychain を参照できず「Not logged in」になったため、実 HOME で起動し直した。その際、この作業をしている session の環境変数が pane に漏れないよう、`env -i` で最小限の環境変数だけを渡した。実 HOME なのでユーザーの shell alias(`claude --model fable …`)が効いたが、claude は同じフラグの重複を受け付けて後ろの値が勝つので、座席は ralph が指定した `haiku` で動いた(ステータス行で確認)

## 後始末

- 両座席とも `ralph org stop` → `disband` → `status` で stopped を確認。専用 server は `herdr server stop` で停止。動いている herdr server はユーザーのものだけ
- 偽 HOME に複製した `auth.json` は削除。`/tmp/r163/` は削除。sandbox の外への `touch` の対象ファイルは存在しない
- ユーザーの `~/.zshrc`、`~/.config/zsh/.zshrc`、`~/.codex/config.toml`、`~/.codex/auth.json` は変更していない
- 変更が残ったもの: `~/.claude.json` に、スクラッチの作業ディレクトリをフォルダとして信頼した記録が 1 件(claude の信頼確認ダイアログで承認したため)。ユーザーの `herdr-server.log` に上記の 8 行
