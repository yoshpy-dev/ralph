# org-inbox-escalate

- Status: Approved
- Approved: 2026-10-10 sha256:b09718e048cd
- Owner: Claude Code
- Date: 2026-10-10
- Related request: 機能ごとの org と director の仕様(`docs/specs/2026-10-07-org-multi-org-director.md`)の 5 段目(FR-5 受信箱、FR-11 の leader の雛形の「人に上げる」の置き換え)。ユーザーは「続けてください」と言った(2026-10-10)
- Related issue: N/A
- Type: feat
- Branch: feat/org-inbox-escalate

## Objective

leader が director(後の段で入る)に件を上げる経路として、`ralph org escalate` と共通の台帳の受信箱を足す。件は open → acked → resolved と進み、`ralph org inbox` の動詞で読んで進める。director が待つための `ralph org wait --inbox` も足す。

5 段目の時点では director を登録する手段(8a 段目)も、受信箱の期限を見る見張り(6 段目)もない。そこで暫定として、仕様の「director が無効のとき」と同じく、escalate は受信箱に記録したうえで、すぐに人への経路(`escalations.jsonl`、stderr、osascript)にも送る。人に送ったことは受信箱に `notified` として残し、6 段目の見張りが同じ件をもう一度鳴らさないようにする。

あわせて、leader の雛形の「人に上げる」を `ralph org escalate` に置き換える。

## Scope

- 受信箱(新しいファイル `internal/org/inbox.go`)
  - 置き場所は台帳の下の `inbox.jsonl`。追記だけのイベントで、種類は `escalated`・`notified`・`acked`・`resolved` の 4 つ。件の状態は畳み込みで出す
  - 書き込みは台帳の下の `inbox.lock` の flock の下で行う(manifest のロックとは別。escalate は manifest に書かない)
  - 件の ID は連番 `e1`、`e2`、…(人が打ちやすいため)。次の ID は、ロックの下で、読めない行も含めてファイルの中に見つかった最大の `e<N>` の次にする。壊れた行の ID を使い回さないため(Codex plan advisory の指摘 1)
  - 読めない行は飛ばして数え、`inbox` の出力で注意を出す(manifest と同じ扱い)
  - 追記の前に、ファイルが改行で終わっていなければ改行を足す。途中で切れた最後の行に新しいイベントが混ざらないようにするため(同じ指摘)
- `ralph org escalate --org-id <id> --text <型付きのメッセージ>`
  - 本文は `internal/org/protocol` で検証する。受け付ける TYPE は `QUESTION`(判断がほしい)、`BLOCKED`(進めない)、`RESULT`(終わった、PR を作った)の 3 つ。`BLOCKED` と `RESULT` は TASK_ID が必須なので、leader は org の件には org_id を TASK_ID に入れる(雛形に書く)。本文は 2,000 字まで
  - 検査に通らなければ終了コード 1 で、何も書かない
  - 通れば `escalated` を書いて ID を出力し、続けて人への経路に送る。`escalations.jsonl` に件の ID・org_id・理由(`inbox_no_director`)の 1 行、stderr の表示、osascript(darwin のときだけ、失敗しても escalate は失敗にしない)。送ったことと、osascript の結果を `notified` として書く
  - 記録はできたが通知の記録(`escalations.jsonl` か `notified`)が書けなかったときは、ID を出したうえで終了コード 1 にし、「記録は済み、通知は未完了」と `ralph org inbox notify <id>` で送り直せることを示す(Codex plan advisory の指摘 2)。escalate を打ち直すと別の件が増えるので、打ち直しは勧めない
  - 受信箱に書けなかったとき(台帳が読めない・書けない、ロックが取れない)も、stderr の表示と osascript で人に知らせてから終了コード 1 にする(指摘 3)。台帳が壊れた場面でも、leader の件が人に届くようにするため
- `ralph org inbox notify <id>`: その件の人への経路をもう一度送り、`notified` を書く。`notified` のない件は `inbox` の一覧で「未通知」と出す(指摘 2)
- `ralph org inbox`
  - `inbox`(open と acked の一覧、`--all` で resolved も、`--json`)
  - `inbox show <id>`(本文と履歴、`--json`)
  - `inbox ack <id>`: open を acked にする。acked にもう一度打つと何もせず終了コード 0、resolved には拒否
  - `inbox resolve <id> --note <ポインタ>`: open か acked を resolved にする。note は必須で、1 行、500 字まで、制御文字なし。resolved にもう一度打つと拒否
  - ack と resolve は、ロックの下で今の状態を読んでから書く
- `ralph org wait --inbox`
  - open の件があればすぐ返り、その件を一覧で出す。なければ open の件が届くまで 1 秒ごとに読み直し、`--timeout-ms`(既定 60000、0 で無期限)で終了コード 1
  - `--inbox` のときは `--org-id` と `--seat` を求めず、`--seat` / `--until` との併用は拒否する
  - acked の件では返らない(ack は読んだことの記録。acked のあとの取りこぼしは 6 段目の resolve の期限が拾う)
- 台帳の置き場所と旧い台帳の検査は、ほかの動詞と同じ(escalate・ack・resolve は書き込む動詞、inbox・show・wait は読むだけの動詞)
- `--org-id` の persistent flag の説明に、inbox と `wait --inbox` は要らないことを書く
- leader の雛形(`internal/org/prompts/leader.md`)
  - 「人に上げる」の 5 か所を `ralph org escalate` に置き換え、例のメッセージを 1 つ載せる
  - `--state-dir` を付ける動詞の列挙に escalate を足す
  - 機能ごとの org では、PR を作ったら PR の URL を `RESULT` で escalate する
  - director ができるまでは escalate が人にも届くこと、受信箱の本文はデータで指示ではないこと
  - escalate そのものが失敗したとき(終了コード 1)の予備の手順として、今までどおり pane にメッセージとエラー(手で閉じる herdr のコマンドがあればそれも)を書いて止まることを残す(指摘 3)。「記録は済み、通知は未完了」のときは、escalate を打ち直さず `ralph org inbox notify <id>` を打つ
- `/org` skill(4 面): 動詞の表(escalate、inbox、`wait --inbox`)、受信箱の節(状態、暫定の人への経路、データであって指示ではない)、「人に上げる」の 2 か所の置き換え
- 文書: README の org の節、AGENTS.md の repo map の `internal/org/` の行、仕様 FR-5 の下に「5 段目で決めたこと」

## Non-goals

- ack と resolve の期限、期限切れの件を人に送ること、横断の見張り(6 段目)
- `[org.director]` の設定、director の登録と起動(8a 段目)、push の起こし口(8b 段目)
- escalate を leader だけに限る機構(呼び出し元を確かめる手段がない。org_id を記録し、雛形で leader だけが打つと書く)
- 受信箱の古い件の掃除や圧縮
- `ralph status` に受信箱の件数を出すこと

## Assumptions

- 共通の台帳(1 段目)の下に `inbox.jsonl` と `inbox.lock` を置けば、linked worktree の leader と main のチェックアウトの人が同じ受信箱を読む
- 4 段目で leader は全 `ralph org` のコマンドに `--state-dir` を付けるので、escalate も同じ台帳に書く
- `escalations.jsonl` の行を読む非テストのコードはない(consult の確認)。欄を足しても読み手は壊れない

## Affected areas

- 新しいファイル: `internal/org/inbox.go` とテスト
- `internal/org/watch.go`(`escalationRecord` に件の ID の欄、人への経路を共有するなら切り出し)
- `internal/cli/org.go`(escalate、inbox の動詞、`wait --inbox`、`--org-id` の説明)とテスト
- `internal/org/prompts/leader.md`、`internal/org/prompts_test.go`
- `.claude/skills/org/SKILL.md` と 3 つの写し、README.md、AGENTS.md、仕様の FR-5

## Visual review

- ページ: `.harness/state/plan-visual/org-inbox-escalate.html`(図 1 全体、図 2 件の状態、図 3 inbox.jsonl のイベント)
- セルフチェック: 全体と全体図(`--fragment overview`)を `plan-visual.sh shot` で撮って確かめた。図 1 の escalate の箱の説明文がはみ出していたので短くし、「使う」のラベルを矢印のそばに移して撮り直した。Codex plan advisory の 3 件を入れて、図 1 に `inbox notify` と失敗のときの扱いを、図 3 に ID の決まりを足して撮り直した

## Design decisions

- 5 段目の escalate は、受信箱に記録したうえで、すぐ人への経路にも送る(consult の判定)。director を登録する手段がまだないので、仕様の「director が無効のとき」と同じ扱いにする。受信箱に書くだけにすると、6 段目が入るまで誰も読まない件になり、今の「人に上げる」(人が pane を見る)より悪くなる。6 段目で、登録された director がいるときは期限で送る形に切り替える
- 人に送ったことは `notified` のイベントとして受信箱に残す。6 段目の見張りが同じ件を重ねて鳴らさないためと、仕様 FR-6 の「通知を試したこと、人が受け取ったことは別々に記録する」に合わせるため
- 受信箱は manifest と別のファイルと別のロックにする。escalate は manifest に書かないので、2 つのロックを同時に取る経路はできない
- ID は連番(`e<N>`)。人が ack や resolve で打つため。次の ID は読めない行も含めた最大の次にし、追記の前に切れた最後の行を改行で分ける(Codex plan advisory の指摘 1、ユーザーが「計画を直す」を選んだ)。壊れた受信箱に書き込みを拒む案は、escalate が人に届かなくなるので採らない
- 記録と通知を分けて扱う。記録はできたが通知が終わっていない件は、終了コードと一覧で見分け、`inbox notify` で同じ ID のまま送り直す(指摘 2)。受信箱に書けないときも stderr と osascript で知らせる(指摘 3)
- `wait --inbox` はポーリング(1 秒)。ファイルの変更通知は入れない(依存が増え、待つ間の負荷は小さい)
- escalate の TYPE は 3 つに絞る。上げる件は「判断がほしい」「進めない」「終わった」の 3 つで足り、ほかの型は座席と leader の間の型

Critical forks: 人に届く経路(上の 1 つ目)。consult が案 A を判定し、承認のときにユーザーに確かめてもらう

## Acceptance criteria

- [x] AC1: escalate は、TYPE が QUESTION・BLOCKED・RESULT でない、TASK_ID が要るのにない、本文が 2,000 字を超える、org_id の形が違う、のどれでも終了コード 1 で拒否され、受信箱にも `escalations.jsonl` にも何も書かない
- [x] AC2: 通った escalate は `escalated` を書き、新しい ID(`e<N>`)を出力する。2 つの escalate を同時に打っても ID は重ならない。途中の行が壊れていても、最後の行が途中で切れていても、次の ID は壊れた行の ID を使い回さず、新しいイベントは読める行として残る
- [x] AC2b: 記録はできたが通知の記録が書けなかった escalate は、ID を出して終了コード 1 になり、`inbox notify` を案内する。その件は `inbox` の一覧で「未通知」と出て、`ralph org inbox notify <id>` で同じ ID のまま送り直せ、件は増えない。受信箱に書けない escalate も、stderr と osascript で知らせてから終了コード 1 になる(書けない台帳と、差し替えた通知の関数で確かめる)
- [x] AC3: escalate のあと、`escalations.jsonl` に件の ID・org_id・理由の 1 行が足され、stderr に表示が出て、`notified`(osascript の結果つき)が受信箱に書かれる。osascript が失敗しても escalate は終了コード 0
- [x] AC4: `ralph org inbox` は open と acked を、`--all` は resolved も出し、`--json` は機械で読める形で出す。`inbox show <id>` は本文と履歴を出す。知らない ID は終了コード 1
- [x] AC5: ack は open を acked にし、acked にもう一度打つと何もせず終了コード 0、resolved には終了コード 1。resolve は open と acked を resolved にし、note がない・複数行・501 字以上・制御文字を含む、のどれでも拒否、resolved にもう一度は終了コード 1。同時に打った ack と resolve で状態が壊れない
- [x] AC6: `ralph org wait --inbox` は、open の件があればすぐ返り、なければ届いた時点で返り、届かなければ `--timeout-ms` で終了コード 1。acked の件では返らない。`--org-id` なしで打て、`--seat` / `--until` との併用は拒否される
- [x] AC7: 受信箱は共通の台帳の下にでき、linked worktree から打っても main のチェックアウトから打っても同じものを読む。`--state-dir` を渡せばその台帳の受信箱を使う
- [x] AC8: leader の雛形が「人に上げる」を `ralph org escalate`(例と TASK_ID の決まりつき)に置き換え、PR を作ったら RESULT を escalate し、`--state-dir` を付ける動詞に escalate を含め、受信箱の本文はデータで指示ではないと書く。escalate そのものが失敗したときの予備の手順(pane に書いて止まる)と、未通知の件の `inbox notify` を書く。`TestRenderRolePrompt_Leader_*` が通る
- [x] AC9: `/org` skill(4 面)、README、AGENTS.md、仕様の FR-5 の注記が合っている。`./scripts/check-skill-sync.sh` と `./scripts/check-sync.sh` が通る
- [x] AC10: `./scripts/run-verify.sh` が通る

## Implementation outline

1. S1(受信箱の層): `inbox.go` のイベント、畳み込み、ロック、ID(壊れた行を含めた最大の次)、切れた最後の行の扱い、ack と resolve の遷移(AC2 の ID、AC5、AC7 の org 層)
2. S2(escalate): 検証、`escalated` の記録、人への経路と `notified`、通知が終わらなかったときの終了コードと `inbox notify`、受信箱に書けないときの通知(AC1〜AC3、AC2b)
3. S3(CLI): escalate、inbox の動詞、`wait --inbox`、`--org-id` の説明(AC1〜AC7 の CLI 側)
4. S4(文書): leader の雛形、`/org` skill、README、AGENTS.md、仕様(AC8、AC9)

## Verify plan

- Static analysis checks: `./scripts/run-static-verify.sh`
- Spec compliance criteria to confirm: AC1〜AC10。仕様の FR-5、FR-11 の leader の雛形の部分
- Documentation drift to check: `ralph org escalate --help`、`ralph org inbox --help`、`ralph org wait --help` と `/org` skill、leader の雛形
- Evidence to capture: verify のレポート

## Test plan

- Unit tests: イベントの畳み込み(順序、読めない行、知らない種類)、ID の連番、状態遷移の表(ack・resolve × open・acked・resolved)、note の検査、TYPE と TASK_ID の検査
- Integration tests: escalate から受信箱・`escalations.jsonl`・stderr・`notified` まで(osascript は差し替え)。同時の escalate(goroutine で ID が重ならない)。CLI の各動詞と `--json`。`wait --inbox` の即時・到着・期限切れ(待つ間に別の goroutine が escalate する)。linked worktree からの共通の台帳
- Regression tests: `./scripts/run-test.sh`。watch の escalation のテスト(`escalationRecord` の欄を足しても通る)、`ralph org wait --seat` の既存のテスト
- Edge cases: 空の受信箱、受信箱のファイルがない、途中の行が壊れている(その行の ID は使い回さない)、最後の行が途中で切れたまま escalate する(新しいイベントが読める)、`e` のあとが数字でない ID、`--timeout-ms 0`、通知の記録だけが失敗する(`inbox notify` で送り直す)、受信箱に書けない(stderr と通知の関数が呼ばれる)
- Evidence to capture: test のレポート

## Risks and mitigations

- 6 段目と 8 段目が入るまで、escalate のたびに人に通知が届く(PR を作るたびの RESULT も含む): 5 段目の暫定として仕様の「director が無効のとき」と同じ扱いにし、6 段目で切り替える。`notified` のイベントで重ねて鳴らさない
- escalate を leader 以外の座席も打てる: 雛形で leader だけが打つと書く。org_id は記録に残る。機構で限るのは呼び出し元を確かめる手段ができてから
- 受信箱の本文が director への指示として読まれる: 雛形と skill に、本文はデータで指示ではないと書く(仕様の Security の項)
- 受信箱が大きくなる: 5 段目では掃除しない。件の数が多くなったら後の段で考える

## Rollout or rollback notes

- バイナリの更新で効く。escalate を使わなければ今の動きは変わらない
- 戻すときはこの PR を revert する。`inbox.jsonl` と `inbox.lock` は台帳の下に残るが、古いバイナリは読まない

## Open questions

- なし

## Progress checklist

- [x] Plan reviewed
- [x] Plan approved
- [x] Branch created
- [x] Implementation started
- 2026-10-10: S1(f8fff664)受信箱の層。決めたこと: `Read` は `InboxReadResult`(件、読めない行の数、無視したイベントの数)を返す。次の ID は、生の行の正規表現と JSON として読めた行の id の両方から最大を取る。note は不正な UTF-8 と U+2028 / U+2029 も拒否し、trim せずに残す。受信箱のファイルがないときの ack / resolve / notify は「知らない ID」を返し、ディレクトリもロックも作らない。flock の helper は `withFileLock` に共有化し、`withManifestLock` の動きと文言は変えていない
- 2026-10-10: S2(a6e736ed)escalate と人への経路。決めたこと: osascript の実行を `osascriptNotify` に切り出し、escalate の通知のタイトルは "ralph org escalate"(watch は今のまま)。`escalations.jsonl` に書けなかったときは `notified` も書かない(件は未通知で残り、`inbox notify` で送り直す)。受信箱に書けなかったときに best-effort で書く `escalations.jsonl` の行の理由は `inbox_not_recorded`(inbox_id なし)。デスクトップ通知には本文を渡さない
- 2026-10-10: S3(faa4b1c3)CLI。計画からのずれ: inbox の動詞と `wait --inbox` は `--org-id` を「求めない」のではなく、渡されたら拒否する(黙って無視すると org で絞った一覧に見えるため)。決めたこと: 一覧の時刻は RFC3339 の ESCALATED_AT。端末に出す受信箱の文字列は制御文字をエスケープする。CLI のテストは TestMain でデスクトップ通知を stub にする
- 2026-10-10: S4(b0fc4c59)雛形・skill・文書。決めたこと: 予備の手順を、検査で断られたとき(メッセージを直して打ち直す)と、それ以外の失敗(pane に書いて止まる)に分けた。`inbox notify` も `--state-dir` を付ける動詞に入れた。雛形のテストは空白を除いて語句を比べる形にし、改行の位置に依存しないようにした(tech-debt の org-feature-worktree の行の (f) と (j) が直る)
- 2026-10-10: self-review cycle 1(6d803bed、Merge: yes、MEDIUM 2・LOW 9)の直し(a396db57)。M1: 雛形の escalate の例を、メッセージを `.harness/state/escalate-{{ORG_ID}}.txt` に書いてから `--text "$(cat …)"` で渡す 2 ブロックに分けた(単一引用符を含むエラー文で壊れないため。guard は通る)。L1: `ralph.toml` が読めないときも NOT RECORDED の banner と通知を出して終了コード 1。L2: 復旧の案内(`inbox notify` / `inbox show`)は `--state-dir` を明示したときにそれを付ける(コマンドの文字列は org 層の `InboxCommand`)。L3: `--org-id` の説明を書き直した。L6: 雛形の agmsg の受信箱と org の受信箱を言い分けた。L9: `inbox notify` は受信箱から読んだ org_id と TYPE を端末向けにエスケープし、通知には検査を通った値だけを渡す。M2 と L7 は sync-docs、L4・L5・L8 は tech-debt に回す
- 2026-10-10: self-review の再実行(39cec1b0、Merge: yes、MEDIUM 1(M2 の残り)・LOW 7)。N1 と N3 のテストの穴を f347842d で直した。雛形の書き込みの例は `mkdir -p .harness/state` と `<<'EOF'` の heredoc から始め、Write ツール(Codex では apply_patch)か heredoc で書くように書いた。guard は mkdir と heredoc を `&&`・改行・`;` のどれでつないでも通る。CLI の config 失敗のテストは escalations.jsonl の `inbox_not_recorded` の行を確かめる。N2 と N3 の残り(PrintableInboxText の置き場所、fallback の配線の重複)は tech-debt に回す
- 2026-10-10: verify(0c8fb404、pass)と test(983ed93b、pass)。test の追加の mutant 40 件のうち、E13(`WaitInbox` の `timeout > 0` を `>= 0`)だけが生き残った。`--timeout-ms 0` で、あとから届いた件を返す動きを固定するテストを a3bd5cff で足した(E13 で落ちることを確かめた)。追記の open の失敗でエラー文にパスが 2 回出ることは tech-debt に回す
- [x] Review artifact created
- [x] Verification artifact created
- [x] Test artifact created
- [ ] PR created
