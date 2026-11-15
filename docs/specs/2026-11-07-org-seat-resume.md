# org の座席を API エラーから続ける: `ralph org resume` と再試行の設定

## Status

2026-11-07 に人が承認した。`/plan` に渡して実装の計画を作る。

同じ日、計画の他モデルの助言(codex)の HIGH 2 件を受けて改訂した。Desired state と FR-8 の待ちの長さを目安と書き直し、打つ直前の割り込みを防ぐ FR-5a と、AC-14・AC-15 を足した。人は推奨案を自動で承認すると伝えている。

実装のあと、cross-review の cycle 1 の指摘 4 件を受けて、FR-2 の 4(入力欄が空であること、止まった文が一致する場所)と FR-5a(`Send` がロックのあとにもう一度待つこと、director の push の wake が同じロックを取ること)を厳しくした。改訂の文は FR-5a の「改訂(2026-11-07、cross-review cycle 1)」にある。この改訂と計画の承認し直しは、上の方針に従い、承認の確認を出さずに行った。

そのあと、cross-review の cycle 2 の指摘 3 件を受けて FR-5a をもう一度改訂した。`Send` は計画の承認を、ロックの中の待ち直しのあと、打つ直前にも確かめる。herdr の director に設定された `wake_cmd` も、同じロックの中で走らせる。止まった文は、印、空白、文の順で始まるときだけ一致とする。改訂の文は FR-5a の「改訂(2026-11-07、cross-review cycle 2)」にある。パイプラインの回数の上限は、人の方針で 2 から 3 に上げた。この改訂と計画の承認し直し(digest `104ee2d3ca75`)も、上の方針に従い、承認の確認を出さずに行った。

## Summary

通信断や利用上限で止まった herdr の座席を、人が director に「続けて」と言うだけで続けられるようにする。claude の座席(leader と headless の director を含む)の pane には Claude Code の再試行の設定を足し、止まる回数そのものを減らす。それでも止まった座席は、新しい動詞 `ralph org resume` が決まった規則で見つけ、続行の文を打つ。director は、人に頼まれたときにだけ resume を打つ。

## Background and problem

org runtime の座席(herdr の pane の中の Claude Code か Codex)が、次のようなエラーでターンを終えて止まることがある。

```text
API Error: Can't reach the API server — check your internet or DNS (ENOTFOUND)
You've hit your session limit · resets 3:45pm
```

いまは、通信が戻るか上限がリセットされたあと、人が止まった pane を 1 つずつ開いて続行の指示を打っている。人はこれを director に代わってほしいと求めた。director に一言いえば、すべての org の止まった座席が続く形である。

### Current state

- ralph は座席の pane を読み(`ralph org read`、`internal/org/verbs.go` の `Read`)、文字を打てる(`ralph org send`、同じファイルの `Send`)。`Send` は、herdr が座席を `idle` か `done` と報告するのを待ち、本文を打ち、750ms 待って Enter を 1 回押し、送信を herdr の状態で確かめる。
- director の手順書(`.claude/skills/director/SKILL.md`)は、leader に DECISION を送る手順だけを持つ。座席に続行の文を打つ手順はない。「してはいけないこと」は、座席の pane のフォルダ信頼の画面と bypass の承諾の画面にキーを送ることを禁じている。
- 横断の見張り(`ralph org watch --all`)の stall の条件(`internal/org/watch.go` の `evaluateSeat`)は、`herdr agent get` の出力と座席の最後のイベントが `[org.watchdog].stall_minutes`(既定 15)変わらないと leader に ALERT を送る。エラーで止まった座席はおそらくこれで ALERT になるが、ralph は「API エラーで止まった」とは判定しない。
- ralph が作る pane には環境変数を足す仕組みがある(`internal/org/driver/paneenv.go` の `SeatPaneEnv` と、座席ごとの `PrimaryCLIEntry`)。
- Claude Code には、上限のリセット後に自分で続ける機能がある(v2.1.234 以降、claude.ai の契約の対話セッションで既定 on)。ただし、誰も attach していない pty では発火しない不具合(anthropics/claude-code#91535)と、複数のセッションが同時に上限に当たると続かない不具合(#92418、#92004)が開いたままである。herdr のヘッドレスのサーバーの org は両方の条件に当たる。人が手で続けている理由はおそらくこれだが、確かめてはいない。
- 通信断では、Claude Code は既定で 10 回再試行してからターンを終える。herdr の外の実測では 183 秒だった(`docs/evidence/org-seat-resume-probe-2026-11-07.md` の 1)。Codex は通信断ではターンを終えず、回復を待つ(同じ記録の 4)。Codex には上限のあとに自動で続ける機能がない。

### Desired state

- claude の座席は、通信断と上限でおよそ `[org].api_wait_minutes`(既定 30 分)まで待つ。そのあいだに直れば、続行の指示なしに続く。上限と混雑(429/529)の待ちはこの長さで区切られる。通信断は再試行の回数で区切るので目安になり、応答が返らない種類の通信断では 1 回の試行が最大で約 10 分かかるため、長くなりうる(FR-8)。
- それを超えて止まった座席は、いまと同じく見張りの stall の ALERT から受信箱を通って director に届く。director は人に「API エラーで止まっています。解決したら『続けて』と言ってください」と伝える。
- 人が director に「続けて」と言うと、director が `ralph org resume` を打ち、止まった座席だけに続行の文が入る。director は座席ごとの結果を人に報告する。director が止まっているときは、人が自分の shell から `ralph org resume` を打てる。

## Requirements

### Functional requirements

- [x] FR-1: `ralph org resume` を足す。対象は、既定で台帳のすべての動いている org の座席と、herdr の中の headless の director(登録が pane を持つもの)である。`--org-id <id>` で 1 つの org の座席に絞れる(このときは director を含めない)。`--dry-run` は何も打たず何も記録せずに、各座席への判定を出す。`--state-dir` はほかの `ralph org` の動詞と同じ。
- [x] FR-2: 座席ごとに次の順で判定する。どれかで外れたら、その座席には何も打たず、理由を付けて飛ばす。
  1. 台帳で動いていて、pane の id がある。
  2. 呼び出したプロセスの pane(`HERDR_PANE_ID`)ではない。
  3. `herdr agent get` の状態が `idle` か `done` である。`working` と `blocked` は飛ばす。
  4. `herdr pane read` の末尾(行数は計画で決める)を、折り返された行をつないでから読む。最後の入力欄より下の footer は読み飛ばす。入力欄が空で、その直前のまとまりが既知の「止まった」文である。エラー文と入力欄のあいだに、claude の `✻ … · done …` のような状態行が 1 行あってもよい。

     改訂(2026-11-07、実機の確認で見つけた点): Claude Code は上限の文を、利用者の prompt の行のすぐ下に空行なしで `⎿` の行として出す(`docs/evidence/org-seat-resume-live-2026-11-07.md`)。そこで、まとまりは空行のほかに、prompt の行(claude の `❯`、codex の `›` のあとに空白)と、prompt のあとの最初の `⎿` の行でも区切る。利用者の prompt のまとまりは、止まった・Enter だけ・待ち中のどの判定にも使わない。

     改訂(2026-11-07、cross-review cycle 1): 4 の「入力欄が空」と「既知の止まった文」は、FR-5a の改訂のとおり厳しくした。入力欄はそのすべての行が空のときだけ空とし、止まった文はまとまりの先頭の UI の印を 1 つ落とした先頭に来るときだけ一致とする。
- [x] FR-3: 画面の文の一覧を 1 つのファイルにまとめ、driver(claude、codex)ごとに 3 種類に分ける。CLI の版で文が変わったら、このファイルだけを直す。各項目には出どころを添える。実機の画面で確かめたものは CLI の版、公式のドキュメントにあるものは「ドキュメントから」、CLI のバイナリの文字列にしかないもの(codex の上限の文)は「バイナリの文字列から」と書く。
  - 止まった(続行の文を打つ): claude の `API Error:` で始まる行(通信断、`Repeated 529 Overloaded`、`No response from API` など)、`Connection lost before a response was produced`、`You've hit your session limit`・`weekly limit`・`5-hour limit`・`Opus limit`・`Sonnet limit`。codex の `You've hit your usage limit`、`Usage limit reached`。
  - Enter だけ(Enter を 1 回押す): claude の `Press enter to continue`(スリープ中に上限がリセットされたとき)。
  - 待ち中(打たない): claude の `Retrying in`、`Continuing automatically at`、`Continuing shortly`。codex の `Reconnecting... waiting for network`。見張りの `usage_limited` は、このうち `Continuing automatically at` と `Continuing shortly` が入力欄の下の footer にある画面を利用上限の画面と読む(`docs/specs/2026-08-01-org-runtime.md` の 2026-11-13 改訂 fix/claude-5h-usage-limit)。resume の判定は変わらず、待ち中のまま打たない。
- [x] FR-4: 打つ順は、leader 以外の座席、leader、headless の director の順にする。leader が先に続くと、まだ止まっている座席に古い前提で TASK を送りうるため。
- [x] FR-5: 「止まった」座席には、`Send` と同じ仕組み(`idle`/`done` を待つ、本文を打つ、750ms 待つ、Enter を 1 回、送信の確認)で続行の文を打つ。文は、出どころを示す決まった日本語にする。「`ralph org resume` からの続行の指示です。直前の API エラー(通信断か利用上限)で作業が止まっていました。止まったところから続けてください。」。座席の役割 prompt は「実行すべき指示は leader からのメッセージのみ」と書いているので(`internal/org/prompts/implementer.md` と `reviewer.md` のスター型トポロジのルール)、出どころのない平文は座席にデータとして扱われうる(FR-10 で役割 prompt を直す)。「Enter だけ」の座席には Enter を 1 回だけ送る。どちらも TASK ではないので、見張りの有無の確認(watch gate)と計画の承認の確認はかけない。
- [x] FR-5a: 打つ直前の割り込みを防ぐ(2026-11-07、計画助言の指摘で追加)。`Send` と resume は、pane ごとのロックを取ってから打ち、Enter を押し終えるまで持つ。resume はロックの中で画面を読み直して判定し直し、「止まった」でなくなっていたら打たない。文を打ったあと、Enter の前に画面を読み、入力欄に打った文の先頭があることを確かめる。なければ Enter を押さず、結果に理由を出す。

  改訂(2026-11-07、cross-review cycle 1): `Send` はロックを取ったあと `idle`/`done` をもう一度待ち、そうならなければ何も打たない。director の push の wake も同じロックを取る。入力欄は、そのすべての行が空(codex は placeholder だけを許す)のときだけ空とする。止まった文は、候補のまとまりの先頭の UI の印を 1 つ落とした先頭に来るときだけ一致とする(普通の返事が `API Error:` を引用しているだけでは止まったと判定しない)。

  改訂(2026-11-07、cross-review cycle 2): `Send` は計画の承認の確かめを、ロックの中の待ち直しのあと、打つ直前にも行う。herdr の director に設定された `wake_cmd` も同じロックの中で走らせる。止まった文は、まとまりがエラーの印(claude `⏺`・`⎿`、codex `■`・`⚠`)、空白、止まった文の順で始まるときだけ一致とし、印のないまとまりは一致させない。codex の上限の文の印は実機で見ていないので、違えば判定できない(打たない)側になる。
- [x] FR-6: 打った座席ごとに、manifest に状態を変えないイベント `resumed` を追記する。中身は、判定の種類(止まった、Enter だけ)と照合した画面の行である。`--dry-run` は記録しない。
- [x] FR-7: 結果を座席ごとに 1 行で出す。org、座席、結果(続けた、Enter を押した、待ち中、飛ばした)、理由か照合した行。上限のリセット時刻を含む行(`resets 3:45pm`、`try again at …`)は、解釈せずにそのまま出す。終了コードは、打とうとした送信がすべて成功したか、打つ座席がなかったとき 0、送信が 1 つでも失敗したか台帳を読めなかったとき 1 にする。
- [x] FR-8: `[org].api_wait_minutes`(整数、既定 30)を足す。1 以上のとき、driver が claude の座席(leader を含む)と、driver が claude の headless の director(herdr の中と `launch_cmd` の起動スクリプト)の環境に次の 3 つを足す。0 のときは何も足さない(この仕様より前の挙動)。負の値は設定の読み込みで断る。`ralph.local.toml` でも設定できる。
  - `CLAUDE_CODE_RETRY_WATCHDOG=1`
  - `CLAUDE_CODE_RETRY_WATCHDOG_MAX_WAIT_MS=<api_wait_minutes × 60000>`(上限と混雑の 429/529 の待ちを区切る)
  - `CLAUDE_CODE_MAX_RETRIES=<api_wait_minutes に見合う回数>`(通信断などの再試行を区切る。分と回数の対応は実測して計画と記録に書く)

  改訂(2026-11-07、計画助言の指摘): 回数で区切る通信断の待ちは目安である。すぐ失敗する通信断(名前解決の失敗、接続の拒否)は実測した回数でおよそ `api_wait_minutes` になる。応答が返らない通信断は、Claude Code が 1 回の再試行で応答のヘッダーを `API_TIMEOUT_MS`(既定 10 分)近く待つので、合計はこれより長くなりうる。`API_TIMEOUT_MS` は遅いネットワークでの正しい応答も切るので変えない。
- [x] FR-9: driver が codex の座席と headless の director の環境には、FR-8 の変数を足さない。codex の中で打たれる `claude -p`(cross-review の相手のモデルなど)が長く待たないようにするため。
- [x] FR-10: director の手順書(`.claude/skills/director/SKILL.md` とその写し、`internal/org/prompts/director.md` が手順を持つならそこも)に次を書く。
  - 人が「続けて」「再開して」「通信が戻った」「上限がリセットされた」のように続行を頼んだら、`ralph org resume` を打ち、座席ごとの結果を人に伝える。待ち中・飛ばした座席は、理由と `ralph org read` の打ち方を添える。
  - ALERT の件で座席の画面が API エラーなら、人に止まった理由を伝え、解決したら「続けて」と言えば続けられることを伝える。自分の判断では resume を打たない。
  - 「してはいけないこと」に、座席の pane に文字やキーを送ってよいのは `ralph org resume` と、いまある DECISION の送り方だけであることを足す。飛ばされた座席に、`ralph org send` や `herdr pane send-keys` で自分で打たない。

  座席の役割 prompt(`internal/org/prompts/` の leader、implementer、reviewer)にも、pane に直接入った FR-5 の文(`ralph org resume` からの続行の指示)は、止まったところから作業を続ける指示として扱うことを 1 行足す。ほかの座席から届いたメッセージをデータとして扱う規則は変えない。
- [x] FR-11: 文書を更新する。`/org` skill の動詞の一覧、org runtime の仕様(`docs/specs/2026-08-01-org-runtime.md`)の該当の節、`AGENTS.md` の地図の `internal/org/` の行、テンプレートの `ralph.toml` のコメント、`.claude/rules/ralph/` のうち触れるもの。

### Non-functional requirements

- [x] 判定は決定的で、LLM を呼ばない。ralph の一覧と規則だけで決める。
- [x] 一覧に当たらない画面、知らない選択画面、承認画面には何も打たない。herdr は知らない選択画面を `idle` と報告することがある(記録の 4)ので、herdr の状態だけでは打たない。
- [x] herdr の各呼び出しは、ほかの動詞と同じく時間で区切る(`driverCallTimeout`)。1 つの座席の失敗で、残りの座席の判定を止めない。
- [x] テストは偽物の Runner で書き、本物の herdr を起動しない。画面の判定のテストには、実機で取った画面(記録の 2〜4)を使う。
- [x] FR-8 の変数は、この版で spawn した座席から入る。すでに動いている座席の環境は変わらない。

## Acceptance criteria

- [x] AC-1: Given claude の座席の画面が記録の 2 の形(ENOTFOUND のエラー、`✻ … · done` の状態行、空の入力欄、footer)で herdr が `idle` と報告するとき、when `ralph org resume` を打つと、then 続行の文を打って Enter を 1 回押し、manifest に `resumed` が 1 件増え、結果の行は「続けた」になる。
- [x] AC-2: Given 画面が記録の 3 の形(`Retrying in … · attempt …`)で `working` のとき、then 何も打たず、結果は「待ち中」になる。状態が `idle` で画面が待ち中の文のときも同じ。
- [x] AC-3: Given 画面が記録の 4 の codex のフック確認の 3 択で `idle` のとき、then 何も打たず、結果は「飛ばした(画面が一覧にない)」になる。
- [x] AC-4: Given 画面の最後が `Press enter to continue` のとき、then Enter だけを 1 回送り、文は打たない。
- [x] AC-5: Given 呼び出したプロセスの `HERDR_PANE_ID` が座席の pane と同じとき、then その座席は飛ばす。
- [x] AC-6: Given `--dry-run` のとき、then herdr の `send-text` と `send-keys` を 1 回も呼ばず、manifest に何も追記せず、各座席の判定を出す。
- [x] AC-7: Given 止まった座席が leader と implementer にあるとき、then implementer に打ってから leader に打つ。
- [x] AC-8: Given 既定の設定で claude の座席を spawn すると、then その pane の作成の `--env` に FR-8 の 3 つが入る。codex の座席には入らない。`api_wait_minutes = 0` なら claude の座席にも入らない。headless の director(herdr と `launch_cmd`)にも、その driver について同じ規則で入る(claude なら入り、codex なら入らない)。
- [x] AC-9: Given `api_wait_minutes` が負のとき、then 設定の読み込みが断る。既定値はテンプレートの `ralph.toml` のコメントと一致し、既定値の一致を確かめるテストが通る。
- [ ] AC-10: 分離した herdr で、claude の座席を通信断で止め(例: `HTTPS_PROXY` に向けたプロキシを止める)、通信を戻したあと `ralph org resume` を打つと、その座席が続きを最後まで進める。実機の記録を `docs/evidence/` に残す。
  - 一部だけ確認した(2026-11-07 時点)。実機で確かめたのは、通信断で座席が止まること、`--dry-run` が `would_resume` を出すこと、resume が文を打って Enter を 1 回押し `resumed` を記録すること、座席が API へ要求を出し直すことまで(`docs/evidence/org-seat-resume-live-2026-11-07.md` の 1)。座席が作業を最後まで進めることは、claude のアカウントが週の利用上限(日本時間 2026-11-07 23 時にリセット)で要求を `rate_limit` で返したため確かめていない。やり直す手順は tech-debt の AC-10 の行にある。
- [x] AC-11: 30 分に見合う `CLAUDE_CODE_MAX_RETRIES` の回数を実測し、計画と記録に書く。
- [x] AC-12: director の手順書に FR-10 の手順と禁止があり、skill の写しの一致の検査(`scripts/check-skill-sync.sh`)とテンプレートの一致の検査が通る。
- [x] AC-13: leader、implementer、reviewer の役割 prompt に FR-10 の 1 行があり、役割 prompt のテストがそれを確かめる。
- [x] AC-14: Given 2 つの resume(か resume と `Send`)が同じ pane に同時に打とうとするとき、then 打つのは 1 つずつで、2 つ目はロックの中で画面を読み直して「止まった」でなければ打たない。Given 判定のあとに画面が承認画面に変わったとき、then 文も Enter も送らない。Given 文を打ったあと入力欄に文がないとき、then Enter を送らない。
- [x] AC-15: Given 分離した herdr で `api_wait_minutes` を小さく(例: 2)して claude の座席を接続の拒否で止めるとき、then 座席がターンを終えるまでの時間と、見張りの stall の ALERT が届くまでの時間を測り、記録に残す。

## User stories

1. As 人, I want to director に「続けて」と言うだけで止まった座席がすべて続くこと, so that 通信が戻るたびに pane を 1 つずつ開かなくてよい。
2. As 人, I want to 短い通信断や上限の待ちでは座席が自分で待って続くこと, so that 続行の指示そのものが要らない場面が増える。
3. As 人, I want to 30 分を超えて止まったら知らせが届くこと, so that org が止まったまま気づかない時間を短くできる。
4. As director, I want to 止まった座席だけを ralph が決まった規則で見つけること, so that 承認画面や知らない画面に誤って打たずに済む。

## Constraints

### In scope

- `ralph org resume` の動詞、画面の判定、文の一覧のファイル
- `[org].api_wait_minutes` と、claude の座席と headless の director の環境への 3 つの変数
- director の手順書の手順と禁止
- 文書と、既定値の一致のテスト
- 分離した herdr での実機の確認

### Out of scope

段階 2 の候補として残す。

- 見張りが止まった座席を見つけて自分で resume を打つこと(既定 off にする想定)
- 上限のリセット時刻の解釈と、リセット前には打たないこと
- 再試行中の座席を `ralph status` や見張りの情報として出すこと。FR-8 の待ちのあいだ、claude の座席は `working` で `herdr agent get` の出力が変わり続けるので、stall の ALERT が出ない(記録の 3)
- Claude Code の StopFailure の hook で止まったことを記録し、判定に使うこと
- promoted の director(人が話している対話のセッション)自身の環境。ralph が起動したものではないので、FR-8 の変数は足さない
- `launch_cmd` の director の pane への resume。herdr の外にあるので届かない
- codex の再試行の設定(`request_max_retries` など)の変更

## Impact

| Target | Impact | Severity |
|--------|--------|----------|
| `internal/org/`(新しい resume の動詞と文の一覧のファイル) | 新しい動詞、manifest の新しいイベント `resumed` | Medium |
| `internal/org/driver/paneenv.go`、`internal/org/spawn.go`、`internal/org/director_start.go`、`internal/org/director_cmd.go` | claude の座席と headless の director の pane の環境に変数が増える | Medium |
| `internal/config/`(`[org].api_wait_minutes`)、`templates/base/ralph.toml`、既定値の一致のテスト | 設定のキーが 1 つ増える | Low |
| `internal/cli/`(`ralph org resume` の cobra の定義) | 新しい CLI | Low |
| `.claude/skills/director/SKILL.md` とその写し、`.claude/skills/org/SKILL.md` とその写し | 手順と禁止が増える | Medium |
| `internal/org/prompts/`(leader、implementer、reviewer の役割 prompt)とそのテスト | 続行の指示の扱いの 1 行が増える | Low |
| `docs/specs/2026-08-01-org-runtime.md`、`AGENTS.md` | 文書の更新 | Low |
| 既存の org | この版の前に spawn した座席の環境は変わらない。resume はどの座席にも使える | Low |

## Dependencies

- herdr 0.7.5 の `agent get`、`pane read`、`pane send-text`、`pane send-keys`(ralph がすでに使っている)
- Claude Code v2.1.295 以降(`CLAUDE_CODE_RETRY_WATCHDOG_MAX_WAIT_MS`)。手元は 2.1.312。それより古い Claude Code はこの変数を読まず、429/529 の待ちに上限が付かない
- 分離した herdr の手順(`docs/recipes/isolated-herdr-server.md`)

## Research findings

### Codebase analysis

- 打つ仕組みは `Send`(`internal/org/verbs.go`)にそろっている。`idle`/`done` の待ち、Enter の前の待ち、送信の確認、途中で失敗したときの状態(`SendProgress`)まである。resume はこれを使い、protocol の検査を飛ばす(`Raw` と同じ扱い)。
- `Read` は `herdr pane read --source recent --lines N --format text` を打つ(`driver/herdr.go` の `PaneRead`)。
- 見張りの stall の条件は `herdr agent get` の出力の文字列を前の周回と比べる(`watch.go` の `evaluateSeat`)。
- headless の director の pane は、登録(`director.jsonl`)の `pane_id` にある(`internal/org/director.go`)。
- pane の環境の変数は `SeatPaneEnv`(すべての pane)と `PrimaryCLIEntry`(座席ごと、`spawn.go`)で足している。`launch_cmd` の起動スクリプトは `SeatPaneEnv` の各項を export する(`director_cmd.go`)。

### Best practices

- Claude Code の `CLAUDE_CODE_RETRY_WATCHDOG=1` は、無人のセッション向けに 429 と 529 を無期限に再試行し、レスポンスにリセット時刻があればそこまで待つ。ほかの一時的なエラー(server error、timeout、接続断)の再試行を 300 回(約 3 時間)にする。`CLAUDE_CODE_RETRY_WATCHDOG_MAX_WAIT_MS` は 429/529 の待ちだけを区切る(公式の env-vars と errors のページ)。実測では、DNS の失敗も watchdog の再試行の対象だった(記録の 1)。
- Claude Code の StopFailure の hook は、API エラーでターンが終わったときに `error`(rate_limit、server_error など)と `last_assistant_message`(エラーの文)を渡す。出力は無視される(記録と通知だけ)。
- herdr のドキュメントは、誤った `idle` を信じた自動化が承認ダイアログに打ち込みうると警告している。
- 同じ問題に対するコミュニティの道具(tmux の中の claude の画面を読んで上限を見つけ、リセット後に `continue` を送るもの)がある。この仕様の resume は、人の指示を起点にする点が違う。

### Alternatives considered and trade-offs

| Option | Pros | Cons | Adopted |
|--------|------|------|---------|
| A. claude の座席に再試行の設定を足す(予防) | 変更が小さい。短い通信断と上限の待ちで続行が要らなくなる | 待ちのあいだ stall の ALERT が出ない。codex には効かない | 採用(上限 30 分つき) |
| B. `ralph org resume`(決まった規則で判定して打つ) | 決定的でテストできる。承認画面に打たない。人も直接打てる | 文の一覧が CLI の版で変わる。一覧にない画面は拾えない | 採用 |
| C. 見張りが自動で resume を打つ | 人の手が要らない | 誤判定で打ち続ける。リセット時刻の解釈が要る | 段階 2 の候補 |
| D. director が画面を読んで判断し `ralph org send --raw` で打つ | コードの変更がない | 承認画面への誤入力。判断がぶれる。座席の数だけ手間がかかる | 不採用 |

## Security considerations

- 続行の文は決まった 1 文で、画面や受信箱から読んだ文字を含めない。画面の行は判定と結果の表示にだけ使い、コマンドの文字列に入れない。
- 打つのは、一覧に当たる「止まった」画面で入力欄が空のときだけにする。承認・信頼・選択の画面には打たない(director の手順書の禁止と同じ考え方)。
- FR-8 の変数に秘密は含まれない。herdr の argv(`ps` で見える)に出てもよい値である。

## Open questions

- 上限(429)に当たったときの claude と codex の画面と、herdr の状態。claude は 2026-11-07 の実機で見た(`docs/evidence/org-seat-resume-live-2026-11-07.md` の 3)。FR-8 の待ちのあいだは `working` で、状態行は `Retrying in … · attempt 1/8` と数え下げ、待ちの終わりに `rate_limit` でターンを終えて、prompt の行の下の `⎿  You've hit your weekly limit · resets 11pm` で止まった(画面の文の一覧は、この文の出どころを `seen:` に直した)。claude の session・Opus・Sonnet の上限の画面と、codex の上限の画面は見ていない。
- prompt の行の下の最初の `⎿` の行が上限の文でないとき、その下の 2 つ目の `⎿` の行にある上限の文は照合しない(cross-review cycle 1 の直し以後。後ろの `⎿` の行は最初のまとまりに入り、先頭が文ではなくなるため、判定は「不明」になって打たない)。そういう画面が実機で出るかは見ていない。
- codex が通信断を待ち続ける時間の上限。15 分を超えると、見張りの stall の ALERT が出る見込み(記録の 4)。director の手順書に、codex の `waiting for network` の ALERT は待つと書くかどうかは計画で決める。
- codex のフック確認の画面が `idle` になる件は 1 回しか見ていない。
- Claude Code の自動の続行(`Continuing automatically at …`)と FR-8 の待ちが重なったときの画面。FR-3 では待ち中として打たない。

## References

- 実機の記録 `docs/evidence/org-seat-resume-probe-2026-11-07.md`
- Claude Code: [Wait for a usage limit to reset](https://code.claude.com/docs/en/interactive-mode#wait-for-a-usage-limit-to-reset)、[Errors](https://code.claude.com/docs/en/errors)、[Environment variables](https://code.claude.com/docs/en/env-vars)、[Hooks](https://code.claude.com/docs/en/hooks)
- Claude Code の不具合: [#91535](https://github.com/anthropics/claude-code/issues/91535)、[#92418](https://github.com/anthropics/claude-code/issues/92418)、[#92004](https://github.com/anthropics/claude-code/issues/92004)、[#103591](https://github.com/anthropics/claude-code/issues/103591)
- Codex: [Config reference](https://learn.chatgpt.com/docs/config-file/config-reference)、[Feature Request: Auto-Resume after Usage Limit](https://community.openai.com/t/feature-request-auto-resume-after-usage-limit/1397124)
- herdr: [agents](https://herdr.dev/docs/0.7.5/agents/)
- 関連する仕様: `docs/specs/2026-08-01-org-runtime.md`、`docs/specs/2026-10-07-org-multi-org-director.md`
