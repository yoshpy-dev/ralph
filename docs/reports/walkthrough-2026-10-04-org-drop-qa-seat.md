# Walkthrough: org-drop-qa-seat

- Date: 2026-10-04
- Plan: docs/plans/archive/2026-10-04-org-drop-qa-seat.md(この PR の最後のコミットで archive に移す)
- Branch: refactor/org-drop-qa-seat(base main 4ee080f5)
- Diff: plan・報告・insight を除くと 49 files、+2532 / -958。うちテスト以外の Go が 12 files で +542 / -250、テストが 15 files で +1612 / -384、埋め込みの雛形が 4 files で +74 / -117

## 何を変えたか

org runtime の座席を、leader(指示役)・implementer(実装役)・reviewer(レビュー役)の 3 つにした。中身は次の 3 つに分かれる。

1. 指示役の識別子を `lead` から `leader` に改めた。役割名、seat id、agmsg の宛先、herdr のエージェント名、雛形のファイル名(`prompts/leader.md`)、CLI のフラグ(`--leader-driver`)が変わる。
2. qa の雛形を撤去した。qa がしていた決定論ゲート(`run-static-verify.sh` / `run-test.sh`)の再実行は、reviewer の最初の手順になった。ゲートが落ちたら、reviewer は差分を読まずに BLOCKED を返す。
3. 旧名が残っていたら、黙って別の動きをせずに、改名や移行先を案内して止める。

## 読む順

コミットの順に読むと、改名(挙動は変えない)と挙動の変更を分けて見られる。

1. `eb30b172` 改名。挙動は変えていない
   - `LeadIdentity = "lead"` を `LeaderIdentity = "leader"` にし、`lead` を含む Go の識別子をすべて改名した。`prompts/lead.md` は `leader.md` に移した。
   - 残したものが 3 つある。watch の state に書く JSON のタグ `lead_agent_get` / `history_lead_lines`(更新の前に書いた state を読めるようにするため。`watch.go:230-231` にコメントがある)、insights の過去の receipts のフィクスチャ、コメントの中の過去の計画やレポートのファイル名。
   - manifest の Details に書く診断用の文字列(`leader_self=true`、`agmsg_leader_joined` など)も改名した。読むのは Go のテストだけ。
2. `e4d2c39a`、`3f30641d`、`23f824e6` 旧名の拒否
   - `internal/org/prompts.go` の `retiredRoles` の表に、改名した `lead` と撤去した `qa` をまとめた。
   - `internal/org/spawn.go`: `--role lead` と `--id lead` は、識別子の検証と同じ位置(manifest を読む前)で拒否する。ralph.toml の `[org.roles].lead` / `[org.permissions.roles].lead` は、idempotent return の直後で拒否する。spawn 済みの座席を同じ引数で再実行したときは、何もせずに返す。どちらの拒否も manifest と receipts に何も書かない。
   - `internal/cli/org.go`: `ralph org spawn` は `--role` / `--id` の旧名を `--model` の fallback より前に拒否し、Spawn の拒否と同じく stdout に `rejected:` を出す。`--lead-driver` は `--leader-driver` の非推奨の別名として残した。
   - `ralph doctor` は ralph.toml の旧キーに warn を出す。
   - stop、disband、status などの動詞は旧名を拒否しない。古いバイナリで立てた org も片付けられる。
3. `90488a06` qa の雛形の削除
   - `prompts/reviewer.md`: ゲートを最初に実行し、`GATE: fail`(チェックが落ちた)か `GATE: unrunnable`(権限や環境で実行できない)なら、差分レビューに進まずに BLOCKED を返す。通れば差分と受け入れ基準を見て、`GATE: pass` の RESULT を返す。`GATE:` はヘッダ行(最初の空行より前)に書く。
   - `prompts/leader.md`: `GATE: fail` は implementer に差し戻す。`GATE: unrunnable` は implementer に戻さず、leader が環境や権限を直す。直せなければ人に上げる。
   - テストはミッション節だけを見て、`GATE: fail` と `GATE: unrunnable` の指示をそれぞれの項目の中で確かめる。
4. `fbc9e539` `--role qa` の拒否
   - `--prompt` がなければ、reviewer を案内して拒否する。`--prompt` があれば、雛形のない独自の役割として起動する。seat id の `qa` と ralph.toml の `qa` キーは、今までどおり使える。
5. `dbb782b5`、`f075656f` 文書
   - `/org` skill(4 面)、`agent-messaging.md`(契約は `TO: leader`)、`AGENTS.md` と `AGENTS.core.md`、README、codex の権限の recipe、`quality-gates.md`、`ralph.toml` のコメントを直した。
   - spec に 2026-10-04 改訂の節を足し、FR-4・FR-7・FR-11・AC・Open questions の行に印を付けた。履歴の本文は書き換えていない。
   - tech-debt に 2 行を足した。FR-7 のゲートが hook ではなく雛形の指示で動いていること、ralph.toml に旧キーがあると `--model` の fallback の警告が拒否より先に出ること。
6. `14922f64` コメントの整合
   - Spawn の検査を段番号ではなく関数名で参照するようにし、検査を並べたコメントに新しい検査を足した。

## 変えていないもの

- 編成パターン名の `Leaded`(Solo / Leaded / Parallel)。メンテナの判断で残した。
- 権限モードの既定値(全役割 `autonomous`)。
- FR-7 のゲートを hook で強制する仕組み。tech-debt に記録した。
- 履歴の成果物(archive の計画、過去のレポート、evidence、insight event、tech-debt の既存の行、spec の履歴の本文)。

## 確かめたこと

- `go test ./... -count=1` は全パッケージで通った。`RALPH_VERIFY_SCOPE=full ./scripts/run-test.sh` と `run-verify.sh` も green。
- 計画の AC-5 / AC-12 / AC-14 の grep で、残った `qa` と `lead` がすべて許可リストか分類に入ることを確かめた。
- mutation: 雛形の指示で 9 通り、`--role qa` の拒否で 3 通り、idempotent の順序で 2 通り。どれもテストが落ちた。tester が別に入れた 34 個のうち 31 個も落ちた。
- reviewer の雛形の smoke: わざと落ちるゲートを `claude -p`(sonnet)に渡し、`TYPE: BLOCKED` と `GATE: fail` を返し、`SEVERITY:` の所見を出さないことを 1 回確かめた。
- cross-review(codex、read-only)の指摘は 0 件。

## 確かめていないこと

- herdr と agmsg で座席を実際に立てたときの動き。
- codex の reviewer 座席を `guarded` にしたとき、ゲートのスクリプトを許可なしで実行できるか。
- 雛形の指示のうち 3 つはテストで固定していない。reviewer の「最初に」「推測で結果を上書きしない」と、leader の「直ったら reviewer にもう一度ゲートから実行させる」。
- smoke で reviewer が書いたレポートは、ゲートの失敗を `SEVERITY: HIGH` の所見として載せていた。fail のときのレポートの書き方は雛形で決めていない。
- codex の権限の recipe に足した「この probe ではゲートを実行しない」の段落は、生きた codex 座席では試していない。
- AC-5 / AC-12 / AC-14 の grep は CI では回らない。

## 利用先への影響

雛形はバイナリに埋め込まれ、座席は spawn した時点の雛形で動き続ける。更新は次の順で行う。

1. 動いている org を `ralph org disband` で片付ける。
2. バイナリを更新する。
3. `ralph upgrade` で core の skill と rule、`AGENTS.md` の managed block を更新する。
4. ralph.toml の `[org.roles].lead` や `[org.permissions.roles].lead` を `leader` に改める(`ralph doctor` が知らせる)。
5. org を立て直す。

herdr のエージェント名は 32 文字までなので、leader 座席を立てられる org_id は 25 文字までになる(以前は 26 文字)。`docs/` 配下と `ralph.toml` は seed なので、利用先にある `quality-gates.md`、`codex-seat-permissions.md`、`ralph.toml` のコメントは古いまま残る。
