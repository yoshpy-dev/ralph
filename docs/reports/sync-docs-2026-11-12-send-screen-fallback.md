# sync-docs report: send-screen-fallback

- Date: 2026-11-12
- Plan: `docs/plans/active/2026-11-12-send-screen-fallback.md`(承認の digest `123428594289`)。issue #308
- Branch: `fix/send-screen-fallback`。base は origin/main `d2e5c6e9`(`c31780b4` で取り込み済み)。run 開始時の HEAD は `d7086a68`
- Pipeline run: 1 回目(`cycle-count.json` は `cycle` 1。この節)と 2 回目(末尾の「Cycle 2」)
- 先行 report: self-review `71e8bfe3`(Merge: yes、LOW 7 件 L2・L4・L5・L6・L7・L9・L10 が残る)、verify `67b6a551`(pass。LOW 3 件 V1〜V3、INFO 3 件 I1〜I3)、test `d7086a68`(pass。Test gaps 1〜7、生き残った変異 M02)。test のコードは `88742506`(cli のテスト 2 ファイル)
- 見た範囲: `git diff origin/main...HEAD`(72 ファイル)。verify の HEAD `71e8bfe3` からあとで変わったのは cli のテスト 2 ファイルと report 2 つだけで、コードは変わっていない。そのため help の文は verify が build した binary で読んだ結果を使い、build し直していない
- 制約: 計画の本文は変えていない。変えたのは AC の 11 個と Progress の 4 個の箱で、`./scripts/plan-visual.sh digest` は編集の前後とも `123428594289`(`git diff --numstat` は 15/15 行)。Go のコードとテストは変えていない
- 差分の大きさ(`/pr` が walkthrough を決める材料。sync-docs の 2 つの commit の前の数): `git diff origin/main...HEAD` は 72 ファイル、+6,277/-176(計画、report、evidence、insights を含む)。docs 以外は 65 ファイルで +4,874/-176、そのうちテストと testdata を除くと 21 ファイルで +1,624/-174(skill と recipe の写しを含む)。walkthrough は書いていない

## Summary

verify の V1 は spec に 2026-11-12 の改訂の項を足して直した。V2 は verify が挙げた 6 行に、掃引で見つけた 8 行を足した計 14 行の Trigger に note を足した。V3 と test の Test gaps は、tech-debt の末尾に新しい 2 行を足した。I1 は AC と Progress の箱を付けた。I2 は直さず、I3 は発火とみなさなかった。ほかの文書(help、org skill と director skill の 4 つずつの写し、recipe の双子、leader の雛形、README、AGENTS.md、quality-gates、repo-map、agent-messaging.md)は実装と合っていた。

## Changes made

| File | Change |
|------|--------|
| `docs/specs/2026-08-01-org-runtime.md` | 「2026-11-12 改訂(fix/send-screen-fallback)」の項を足した((a)〜(g)、2026-11-07 改訂の次、運用ノートの前)。2026-11-07 改訂 (b) のロックの下の待ちの文と FR-3 の文に、改訂の印を 1 つずつ足した。ほかの本文は変えていない。下の「V1」 |
| `docs/tech-debt/README.md` | 14 行(180、188、192、226、262、337、365、371、380、427、457、460、461、464)の Trigger の末尾に note、Related の末尾に計画の path を足した。挿入だけの編集。末尾に新しい 2 行(472、473)を足した。下の「V2」「V3」 |
| `docs/plans/active/2026-11-12-send-screen-fallback.md` | AC1〜AC11 と、Progress の `Implementation started`・`Review artifact created`・`Verification artifact created`・`Test artifact created` を `[x]` に。`Branch created` と `PR created` は `[ ]` のまま。下の「I1」 |
| `docs/reports/sync-docs-2026-11-12-send-screen-fallback.md` | この report |
| `docs/insights/events/2026-11-12-send-screen-fallback.jsonl` | `sync_docs` の行(cycle 1、pass)。この file は run の開始時に未追跡で、先行の 4 行(self-review 2 回、verify、test)も、この report の commit で一緒に入る |

commit は 2 つ。内容の commit `181ecbb3`(spec、tech-debt、計画)と、この report と insights の commit。

## verify が挙げた drift

### V1: org runtime の仕様に改訂の項が無い

`docs/specs/2026-08-01-org-runtime.md` に「### 2026-11-12 改訂(fix/send-screen-fallback)」を足した。中身は次の 7 項で、仕様の決まり(本文の履歴は書き換えず、改訂の項で足す)に沿った書き方にした。

- (a) 画面のフォールバックの対象(claude と codex)、既定で有効、`--no-screen-fallback`、確かめをしない `--timeout-ms`(Enter の前の待ちが既定の 750 ms で 20250 未満)
- (b) 流れ(herdr の最初の待ちから確かめの分を引く、時間切れのときだけ `agent get` と 2 回の読み、入力待ちの条件、入力待ちでなければ herdr の待ちに戻る、確かめを半分の時間で区切る)
- (c) ロックの下で herdr の待ちの代わりに画面を読み直すこと、Enter の前と後の画面の確かめ、`submit_unconfirmed=true`、`sent` の Details の `via=screen herdr_state=<状態>`
- (d) 入力待ちでないときのエラーの文(`screen check:`、状態、判定と理由、3 つのコマンド)
- (e) `ralph org status --live`(herdr のサーバーを起動しない、`--live` なしは台帳だけ、見張りは変えていない)
- (f) `resumeCodexFooterLine` の codex 0.166.0 対応(resume の判定の規則は変えていない)
- (g) 実機の herdr での再現ができず、画面の道は偽物でだけ通したこと(tech-debt の Known gaps の行 (a) を指す)

verify が挙げた 2 か所には、2026-11-05 の改訂が使った形の印を足した。

- 2026-11-07 改訂 (b) の「ロックを取ったあと、座席が `idle`/`done` かをもう一度待つ」の文の終わりに `**(2026-11-12 改訂で変更: …画面で入った send は、ここで herdr の待ちをやり直す代わりに画面をもう一度読み、入力待ちのときだけ打つ。2026-11-12 改訂 (c))**`
- FR-3 の「追加指示は `herdr agent wait --until idle` を挟んだ `send` 動詞経由とする。」の次に `**(2026-11-12 改訂で変更: herdr の待ちが時間切れになったとき、claude か codex の座席は画面が入力待ちと確かめられれば打つ。2026-11-12 改訂 (a)〜(c))**`

(b) の印は、画面の道で変わる文の直後(`per-plan` の文の前)に置いた。そのあとの計画の承認の確かめと director の wake の文は、画面の道でも同じ(`pane_lock.go` の doc も、待ちか画面の読みのあとに最後の承認の確かめをすると言う)。`/natural-japanese` の lint は、足した項に「和欧文間の半角空白」を 1 件(7 行にまたがる)指した。この spec は全体が英字の前後に空白を入れる書き方で(同じ日の他の項も同じ)、この項だけ詰めると不揃いになるので、そのまま残した。

### V2: tech-debt の Trigger(verify が挙げた 6 行 + 掃引で 8 行)

行を開いて Trigger を読んだ。note は Trigger セルの末尾に足し(`. 2026-11-12: fix/send-screen-fallback (#308) hit …`)、Related の末尾に `docs/plans/active/2026-11-12-send-screen-fallback.md` を足した。`/pr` の `archive-plan.sh` が `archive/` に書き換える。`archive/` とは書いていない。数は、`git show` と `wc -l` と `grep -o` で測り直した。

verify が挙げた 6 行。

| 行(先頭の語) | 当たった所 | 書いたこと |
|----|----|----|
| 226「Small findings of org-plan-approval」(b) | `Send` とその doc(4 回目) | `approval_test.go` の `TestCheckTaskPlanApproved_StaleOrMistypedDigest` の doc の `approval gets` だけの行は今もある(ファイルに触れていない)。持ち越し。(c) は当たらない |
| 337「A Codex leader that waits for its seats」 | `Send` の idle の待ち | 払っていない。前面で待つ Codex leader の画面には動作中の行が出て、確かめが受け付けないので、その手番が終わるまで届かない(leader の雛形と `/director` skill がそう書く。Codex の leader では動かしていない)。agmsg と見張りに渡す案は Scope の外。持ち越し |
| 365「`internal/org/verbs.go` is 2277 lines」 | `Send` と doc の新しい節 | `verbs.go` が 2501 から 2597 行(第 1 列の 2277 に 300 を足した 2577 を超えた)、`spawn.go` は 3093 から 3101。`deddebc9` で測った。テストは新しい file に入ったので、`verbs_test.go` ほか 3 つは変わっていない。以後の +300 は `verbs.go` の 2597 から数える。持ち越し |
| 457「Seven mutants of `ralph org resume`」 | `defaultResumePaneLockWait` の doc、`resumeCodexFooterLine` | `findResumeInputBox` を通して `ClassifyResumeScreen` と `ResumeInputHolds` の結果が変わるので、R6 の前例どおり当たりと数えた。(d)(e) の fixture は足していない(複数行の画面は文の終わりを見せるので (e) にあたらない)。持ち越し |
| 460「The recognition of codex's footer」 | `resumeCodexFooterLine` | **大部分を払った**。(c) の起きる場面が codex 0.166.0 で実際に起きた。画面は evidence の 6 節と codex の fixture に、形は正規表現に入り、comment は見た形(0.165.0 と 0.166.0)と binary の文字列から取った形を分ける。払っていないのは (b)(`% left` の形は保存した画面に無い)と、理由に footer の形を出すこと。行は開いたまま |
| 461「Five test gaps from the cycle 2 `/test`」 | `resumeCodexFooterLine` | (b) を一部払った。`TestResumeCodexFooterLine` の下書き 2 つが、F2(終わりの錨)と F3(前に別の項目を要る条件)を固定するように読める。F6(`% context left` の終わりの錨)の下書きは無い。赤になるかは読んで出した見込みで、`/test` の変異は footer の正規表現に当てていない(Test gaps 6)。持ち越し |

掃引で見つけた 8 行(verify も handoff も挙げていない)。

| 行(先頭の語) | 当たった所 | 書いたこと |
|----|----|----|
| 180「Two documentation findings of org-stop-all」(a) | `/org` skill の表の `send` と `status` の行(4 つの写し) | `send` は 2,528 → 3,500 字、`status` は 689 → 1,047 字(`.claude/skills/org/SKILL.md` の表の行全体を数えた)。`stop` と `disband`(1,208 と 1,219 字)は変わらず、`send` はどちらより長く、`status` はまだ短い。持ち越し |
| 188「What a leader is told about its reservation」(a) | leader の雛形の agmsg の受信箱の節 | 1,329 → 1,332 行。`RolePromptVars` と `{{SCOPE}}` は変わらない。(b) は当たらない。持ち越し |
| 192「Test gaps left by the org-limits-reserve cycle 2 fix」(c) | `ralph org status` の `Long` | **一部を払った**。`--live` の段落とフラグの説明は `TestOrgStatus_HelpDescribesLive` が cobra の出力から確かめる(段落の 2 文とフラグの説明)。`reserved:` の行、`feature:` の行、不完全な記録の注、`--config` の説明は読んでいないので、(c) は残りについて開いたまま |
| 262「Three test gaps of the org-director-attach PR」 | `/director` skill の受信箱の節(`### 答える順`) | `director_skill_sync_test.go` と `TestDirectorSkill_NamesTheInboxWaitTimeoutPhrase` に触れていない。(1)〜(3) は開いたまま。持ち越し |
| 371「The recipes under `templates/base/docs/recipes/` cite `docs/evidence/` files」 | `isolated-herdr-server.md` の新しい節(2 つの写し) | **前の note と違い、citation が増えた**。`docs/evidence/org-send-screen-probe-2026-11-12.md` を 1 回足した。この recipe は 13 件・6 file から 14 件・7 file になった(origin/main と HEAD を `grep -o` で数えた)。Scope は新しい節で、旧い citation の言い換えは入らず、新しい 1 件だけ言い換えると recipe の中で言い方が割れるので直していない。行は開いたまま |
| 380「RC6 … the `/director` skill's check after a send」 | `/director` skill の送って確かめる手順(4 か所、4 つの写し) | 画面を読んだ send の説明を足した。スクロールで消えた入力のほかの配達の印は、まだ書いていない。`ralph org read` も変わらない。持ち越し |
| 427「Seven test gaps of org-cross-review」(6) | `ralph org status` の JSON | `--live` で、読んだ座席ごとに `live` object(`agent`・`herdr_state`・`herdr_error`・`screen`・`screen_reason`・`screen_line`・`mismatch`)が付く。`role` と `driver` の欄は同じ。`TestOrgStatus_JSON` は今も `seat_id` だけ。持ち越し |
| 464「The `/director` skill … do not say what a director does when it sees the scaffold version note」 | `/director` skill(4 つの写し) | 判断の中身(人に伝えるか、`ralph upgrade` を打つか)は Scope の外。`director.md` と 4 つの動詞の help は触っていない。持ち越し |

当たらないとした行と理由。

| 行(先頭の語) | 判断 |
|----|----|
| 137「`Send` … has no unit test for its `o.findSeat` manifest-read-error branch」 | 3 つ目の条件「次の `fakeHerdr` の遅延注入の hook」(verify の I3)は当たらない。この PR が足した `doneAfter`、`blockRead`、`blockGet` は、`fakeResumeHerdr` を包む新しい偽物 `sendFallbackHerdr` のもので、`fakeHerdr` は変えていない(diff に `fakeHerdr` の行が 0)。`Send` の seat の lookup と `DefaultSendEnterDelayMS` も変わらない。ただ、遅延を入れられる偽物が 1 つ増えたので、この行の「clamp の穴」のテストを書く人は `sendFallbackHerdr` を使える。行には足していない |
| 338「A Codex reviewer that runs the gate」 | Trigger は「上の行の直し」(`Send` が agmsg に渡す案)で、満たしていない |
| 362「A seat that a spawn left through `spawn_close_failed`」 | Trigger は `ralph status` の座席の表(`internal/cli/status.go`)と roster。この PR が変えたのは `ralph org status` の表(`org.go` の `printStatusTable`)で、別のコマンドとファイル |
| 456 | 解決済みの行(Trigger は取り消し線) |
| 450・451・452 | Trigger は実機での観察(長い沈黙、通信断の停止、利用上限の停止)で、`resume_screens.go` の次の変更ではない |
| 195・199・211 の (b)・384・372・377・381・401・426・459・466・467・239 の (o)(p) | 名指しの関数や節(`newOrgStartCmd` の `RunE`、`newOrgRuntimeAt` とその doc、leader の雛形の 14 番、`/org` skill の split-plan の例、director の wait の終了コード、README の `[org].state_dir`、新しい節の追加、など)に触れていない。leader の雛形の変更は agmsg の受信箱の節の 1 文で、feature-org の手順ではない |
| 364・443 | `spawn.go` に足したのは `Org` の 2 欄で、spawn の失敗の経路と `placeSeatPane` は変わらない |

掃引は、Trigger の列を python で切り出し(`" | "` で分けた 4 つ目のセル、取り消し線の中を外す)、変わった file、関数、help、skill、recipe、README の語に加えて、`` `/director` ``・`` `/org` ``・leader の雛形などの語(正規表現)で 3 回探した。当たった行は Trigger を 1 つずつ読んだ。

### V3: self-review の残りの LOW 7 件

tech-debt の 472 行目に 1 行にまとめた(「Seven small findings of fix/send-screen-fallback」)。L2・L4・L5・L6・L7・L9・L10 を、関数とテストの名前で書いた(新しい `file:line` は 0)。Trigger は項目ごとに分けた(L6 は実画面で箇条書きの返事の行が busy と読まれたとき、L9 は CLI が `DriverCallTimeout` か `SendScreenReserve` を設定できるようにするとき、L10 は herdr の 3 つの呼び出しを実測したとき、など)。L4 の `orgInboxHintStateDir` は、ほかの org の動詞も使う解決関数なので、Trigger は「`newOrgSendCmd` の `RunE` か `orgInboxHintStateDir` の doc の次の変更」にした(関数名だけだと広すぎる)。L6 は、self-review が勧めた「Known gaps の行 (b) に 1 文足す」の代わりに、この行に入れて「(b) と同じ種類」と書いた。

### I1〜I3

- I1: AC1〜AC11 と Progress の 4 つを付けた。AC1・AC2・AC11 は、verify の「満たす」(コードとテストのソース、偽の herdr での binary の観察)と test の pass による。実機の herdr で working を言い続ける座席への送信は未確認のままで、tech-debt の Known gaps の行 (a) と、新しい Test gaps の行 (5) にある。`Branch created` と `PR created` は handoff が挙げていないので付けていない。`Affected areas` の `pane_lock.go`(「変えない見込み」で、doc comment は変えた)は、digest の対象で計画の本文なので触っていない。計画の後段の sub-bullet(self-review、verify、test、sync-docs の commit と結果)も、handoff が本文を変えないよう言ったので足していない
- I2: 直していない。director skill が起動直後の画面を省くことと、README が 2 回の読みが同じ画面であることを省くことは、どちらも要約で、正しい規則は help、org skill、recipe にある(verify の判断のとおり)
- I3: 発火とみなさなかった(上の 137 行)

## 新しい tech-debt の行(472、473 行)

| 行(先頭の語) | 内容 |
|----|----|
| 「Seven small findings of fix/send-screen-fallback」 | self-review の L2、L4、L5、L6、L7、L9、L10。どれも失敗側に倒れるか、doc とテストが本番の動きと合わないだけ。L6 だけ効果があり、箇条書きの返事の行が busy と読まれると画面の道が黙って効かなくなる |
| 「Test gaps of fix/send-screen-fallback」 | test の Test gaps 1〜7。(1) M02: 2 回の読みのあいだの 3 秒を固定するテストが無い。(2) 確かめが 2 回の読みのあいだで切れる道。(3) `sendHerdrState` の unknown の 2 つの道。(4) `Send` の確かめで herdr が blocked を返す場合。(5) 実機と既定の `--timeout-ms` 30000 での実行。(6) 変異 7 個しか回しておらず、残り 57 個は定義だけ(定義は `.gitignore` が外す evidence の log にあるので、clone には report の表だけが残る)。(7) 時間に頼るテストの余裕は load 13 まで。Trigger は項目ごとに、足すテストの形まで書いた(数字は読んで出したもので、回していない) |

verify の Coverage gaps は、4 番目(CLI のテストが画面の道を通さない)が `88742506` で足されたので行に入れていない。1 番目(実機の herdr)は Known gaps の行 (a) に、2 番目(既定の `--timeout-ms` で動かしていない)は Test gaps の (5) と L10 に、3 番目(`AgentGetHangs` の説明が本番と合わない)は L9 に入っている。

行の中の名前は、関数、テスト、変異の名前で、新しい `file:line` は無い。参照した名前は `git grep` で実在を確かめた(`sendFallbackHerdr`、`sendFallbackOrg`、`blockGet`、`sendScreenSubmitChecks`、`sendScreenCall`、`liveScreen`、`sendScreenText`、`sendHerdrState`、`isSendBusyLine`、`errSendTextNotInInput`、先に出たテスト 3 本ほか)。

## 確かめたこと(文書が実装と合っていた所)

- org skill の 4 つの写し: sha256 が同じ(`0921a99d…`)。director skill の 4 つの写しも同じ(`ca8fb5f7…`)。recipe と `templates/base/` の双子は `cmp` で同じ
- `ralph org send --help` と `ralph org status --help` は verify が `71e8bfe3` から build した binary で読んだ。それ以後にコードは変わっていない(`git diff --stat 71e8bfe3..HEAD` はテストと report の 4 ファイル)。`TestOrgSend_HelpDescribesTheScreenFallback` と `TestOrgStatus_HelpDescribesLive` が文を固定する
- `AGENTS.md` の `internal/org/` の行は `send_fallback.go`、`send_screen.go`、`status_live.go` と pane lock の「画面の道では読み直す」を言う。`README.md` は director から leader への send の説明に、画面の確かめと `status --live` を足している。どちらも合っている
- `docs/recipes/codex-seat-permissions.md` の「Enter once, and then checks through herdr that the seat left idle/done」は、herdr の待ちのあとに打つ道の説明として今も正しい(画面の道は herdr が working と言い続けるときだけ)。直していない
- `docs/quality/quality-gates.md`、`docs/architecture/repo-map.md`(`internal/org` の記述なし)、`.claude/rules/ralph/agent-messaging.md`(send の待ちに触れない)、`docs/specs/2026-10-07-org-multi-org-director.md` の印(Codex leader の send が届かないこと。今も正しい)は変更不要
- 計画の AC と Design decisions は verify と test が照合済み。本文は変えていない

## 残っていること(lead へ)

- 計画の本文と Go のコードは変えていない。パイプラインの再走は要らない
- 払っていないもの: note を足した 14 行の Trigger の大半(持ち越し)と、新しい 2 行の中身(どれもコードかテストの変更)。460 行目は大部分を払ったが、行は開いたまま。192 行目の (c) は `--live` の段落だけ払った
- 371 行目(recipe の citation)は、この PR の新しい節が dead な path を 1 件足した(`docs/evidence/org-send-screen-probe-2026-11-12.md`)。scaffold した project にこの file は無い。旧い 13 件と同じ扱いで残したが、新しい 1 件だけ「ralph repository の `docs/evidence/…`」と言い換えるなら、recipe の 2 つの写しの 2 行で済む。言い換えるか決めてほしい
- `docs/tech-debt/README.md` は別のセッションも編集しうる。この commit は 14 行への挿入と、末尾への 2 行の追加で、既存の行は消していない。merge の衝突は末尾の追加行で起きやすい
- `docs/plans/active/...` への参照を Related に足した。`/pr` の `archive-plan.sh` が書き換える

## Verification

- `git show HEAD:docs/tech-debt/README.md` との比較(`tdcheck.py`)で、変わった行は 180、188、192、226、262、337、365、371、380、427、457、460、461、464 の 14 行(どれも `difflib` の操作が `insert` のみ)と、末尾の新しい 2 行(472、473)。変わった行も新しい行も列は 5、列ごとのバッククォートと `~~` は偶数。足した文に `file:line` は 0
- 計画の参照: `docs/tech-debt/README.md` の `docs/plans/(active|archive)/...` を `verify.local.sh` の `check_tech_debt_plan_refs` と同じ式で調べて、無い参照が 0 件(119 件)
- `./scripts/plan-visual.sh digest docs/plans/active/2026-11-12-send-screen-fallback.md` は編集の前後とも `123428594289`。`git diff --numstat` は 15/15(AC 11 と Progress 4)
- `./scripts/check-sync.sh` は `PASS: all files in sync.`、`./scripts/check-skill-sync.sh` は 14 skill で ok、`git diff --check` は rc 0。変えた 3 ファイルの U+FFFD は 0 件
- `/natural-japanese` の lint(`uv run lint.py`)は、spec に足した項(`/private/tmp` に抜き出した)とこの report の両方で、「和欧文間の半角空白」を 1 件ずつだけ指した。どちらも英字の前後に空白を入れる repo の書き方で、先行の report と spec も同じなので残した
- 実行していないもの: テスト、`run-verify.sh` の全体(コードを変えていない)

## Insight event

```
./scripts/insights-append.sh --slug send-screen-fallback --flow standard --phase sync_docs \
  --verdict pass --cycle auto --source skill
```

終了コード 0。`docs/insights/events/2026-11-12-send-screen-fallback.jsonl` に足した行は cycle 1、pass、ts は 2026-11-12T14:00:40Z。この時点の `git status --short` は、計画、spec、tech-debt の `M` と、未追跡のこの jsonl だけで、迷子のファイルは無かった。

---

# Cycle 2

- Date: 2026-11-13 JST(UTC では 2026-11-12。file 名は cycle 1 のまま)
- Pipeline run: 2 回目(`cycle-count.json` は `cycle` 2)。上の節は cycle 1 のままで、書き換えていない
- 先行 report(cycle 2 の版): self-review `c93f854b`(LOW 8 件: L2・L4・L5・L6・L7・L9・L10・L12。L11 は `cea96ea4` で直り、後半は観察)、verify `89585818`(pass。drift V4・V5・V6、INFO I4・I5・I1)、test `dd5e08ff`(pass。変異 5 個すべて死んだ。Test gaps 1〜8、8 は新しい)。cross-review の triage は cycle 1 の `0efeef81`(ACTION_REQUIRED 1)。引いた commit は各 report を含む commit で、report の中の `HEAD:` の行の値ではない
- 見た範囲: cycle 1 の内容の commit `181ecbb3` のあとのコードの変更は 3 つ。`f24290b5`(番号付きの入力欄の行を選択の画面と見分ける `sendTypedInputBox`、M02 のテスト、合成 fixture 2 つ)、`cea96ea4`(`sendTypedInputBox` が `resumeMarkerLineBox` の `bool` を見る)、`39004c89`(origin/main `0936aaea`、#321 の取り込み)。run 開始時の HEAD は `dd5e08ff`
- 制約: 計画の本文は変えていない。変えたのは Progress の `Branch created` の箱 1 つで、`./scripts/plan-visual.sh digest` は編集の前後とも `1db244296e07`(`git diff --numstat` は 1/1 行)。cycle 1 の節にある `123428594289` は、S8 が D3 に番号付きの行の扱いを足して承認をやり直す前の値。Go のコードとテストは変えていない
- 差分の大きさ(`/pr` が walkthrough を決める材料。sync-docs の commit の前の数): `git diff origin/main...HEAD` は 78 ファイル、+6,906/-212(計画、report、evidence、insights を含む)。docs 以外は 67 ファイルで +5,266/-196、そのうちテストと testdata を除くと 21 ファイルで +1,748/-194(skill と recipe の写しを含む)。cycle 1 は 72 ファイル、+6,277/-176 だった。walkthrough は書いていない

## Summary

verify の V4・V5・V6 と self-review の L12 は、すべて tech-debt の行の直しで、3 行(479、457、478)を直した。L11 の後半は 478 行目に足した。I4 は 473 行目、I5 は 479 行目の Related、I1 は計画の Progress。spec の 2026-11-12 改訂の (c) に、番号付きの入力欄の行の扱いを 1 文足した。掃引で、行 365 の `send_screen.go` の行数が古いこと、行 461 の cycle 1 の note が test report の番号を指すこと、test の 2 ファイルが 800 行を超えたことを見つけ、前の 2 つは追記で直し、後の 1 つは新しい 480 行目にした。help、org skill、director skill、recipe、AGENTS.md、README は、直さなかった。

## Changes made

| File | Change |
|------|--------|
| `docs/tech-debt/README.md` | 365、457、461、473、478、479 行目の編集(挿入が主で、消した字は意図した所だけ)と、末尾に新しい 480 行目。下の「handoff の項目」と「掃引で足したもの」 |
| `docs/specs/2026-08-01-org-runtime.md` | 2026-11-12 改訂 (c) の「入っていなければ Enter を押さず、…pane を返す。」のあとに、番号付きの手順が入力欄の先頭の行になる場合の文を 1 つ(その行の文が打った文の一部で、まわりが入力欄の形のときだけ入ったと読む。打つ前の判定と Enter のあとの確かめは、今までどおり選択の画面と読む)。ほかは変えていない |
| `docs/plans/active/2026-11-12-send-screen-fallback.md` | Progress の `Branch created` を `[x]` に。下の「I1」 |
| `docs/reports/sync-docs-2026-11-12-send-screen-fallback.md` | この report(cycle 1 の節はそのまま、冒頭に pointer を 1 行、末尾に「Cycle 2」) |
| `docs/insights/events/2026-11-12-send-screen-fallback.jsonl` | `sync_docs` の行(cycle 2、pass)。この file は run の開始時に既に `M` で、cycle 2 の self-review、verify、test の 3 行(未コミット)も、この report の commit で一緒に入る |

commit は 2 つ。内容の commit `39ad3434`(tech-debt、spec、計画)と、この report と insights の commit。

## handoff の項目

### V4 と L12 の (a): 479 行目の (1) M02

(1) の本文と Impact の「Gap (1)」を `~~` で消し、「Closed in fix/send-screen-fallback (f24290b5)」と書いた。cycle 2 の test が M02(`waitOrCtxDone(ctx, 0)`)を当て直して、`TestOrgSendScreen_TwoReadsAreTheReadGapApart` が落とした(2 回の読みが数 µs しかあいていない)ので、「払った」と書ける。Trigger の (1) も取り消して「Done」にした。残りの gap は、test report の cycle 2 の版(1〜8)に合わせて内容を更新した。

- 番号は cycle 1 のままにした。test report の cycle 2 の版は残りを 1〜8 に付け替え(その 1〜6 がこの行の (2)〜(7))、self-review、verify、test の各 report が行を「(1)」「(e)」などと引くので、付け替えると読めなくなる。対応は行の冒頭に 1 文で書いた(新しい 7 と 8 はこの行の (8) と (9))。付け替えたほうがよければ、そう言ってほしい
- (8)(新): test report の Test gaps 7。`sendTypedInputBox` の claude の形の条件が、assert ではなく `resumeInputBox.text` の範囲外の panic で固定されていること。条件を外すと、`resumeInputBoxShape` が返す空の box が `holdsHead` を通って `lines[1:0]` で panic し、最初の panic のあとの行は変異の下で回らない。`text` が空の box を許すように変わると N2 は生き残る
- (9)(新): test report の Test gaps 8。`sendCodexHasFooter` の `box.footerStart < len(lines)`(`codexFooterStart` が見つけた footer を受け入れる枝)を通るテストが無い。codex の 3 つの fixture は footer の下に `⚠ 11 · f2` の行があって、`sendCodexFooterInInput` の枝だけを通る。overlay の probe で production は正しく true を返すと分かっている。この枝を外した変異が落ちるかは読んだだけ(変異は 5 個までと決められていた)
- (6): cycle 2 の変異 5 個(M02 と新しい N1・N2・N3・R1)がすべて死んだこと、cycle 1 の 57 個は未実行のままであること、定義の log が `docs/evidence/test-2026-11-12-send-screen-fallback.cycle1.log` に移ったこと(ディスクにあることを確かめた)を書いた。AC8 の「画面の規則」は R1 が `findResumeInputBox` の順だけ覆う、と書き分けた
- (7): load 23 までの org の全体の回で通り、`TestOrgSendScreen_TwoReadsAreTheReadGapApart` は `-count=30` で 30/30(load 7〜18)、余裕は約 100 ms。Trigger に、self-review が勧めた「間隔を 100 ms に下げる」を足した(読んだだけ、走らせていない)
- (5): 合成 fixture 2 つの文を 1 つ足した(473 行目の (b) を指す)
- Related: test report は cycle 1 の版 `d7086a68` と cycle 2 の版 `dd5e08ff` を、verify report は cycle 1 の版 `67b6a551` と cycle 2 の版 `89585818` を添えた(I5)。cycle 1 の版にだけあった「CLI のテストで画面の道を通すものが無い」は `88742506` で埋まったことも書いた

### V5 と L12 の (b): 457 行目の (e)

(e) の本文を `~~` で消し、「Closed in fix/send-screen-fallback (f24290b5)」と書いた。`TestResumeInputHolds` の `claude input box holds only the head of the sentence` の行が、入力欄が文の頭だけを見せる画面で true を期待し、`holdsHead` が文の頭 12 字だけを比べるので、全部を比べる変異(C10)では落ちる(`holdsHead` を読んで確かめた)。C10 そのものは cycle 2 の変異 5 個に入っておらず、「read, not run」と書いた。

- Impact の「(d), (e)」は「(d) …。(e)、同じ種類のもの、は閉じた」に。Trigger の「for (e), a screen whose input box …」は `~~` と「done in fix/send-screen-fallback, f24290b5」に
- 2026-11-12 の cycle 1 の note が言う「(e) の画面は足されていない」は、同じ branch の自分の note なので、その場で直した。「(d) の fixture も、(e) の画面もない」を、「(d) の fixture は無い。送りの確かめの複数行の画面は打った文の終わりを見せるので (e) にもあたらず、(e) の画面は同じ branch の `f24290b5` で足された(最後の note を見る)」に変えた。足した本文は消していない
- 末尾に cycle 2 の日付つきの文を足した。`f24290b5` が `ResumeInputHolds` の比べを `holdsHead` に分け、`findResumeInputBox` を `resumeMarkerLineBox` と `resumeInputBoxShape` に分けたので、この Trigger に再び当たった。(e) は払い、(a)(b)(d)(f)(g) は払っていない(コミットは send の確かめの番号付きの行の話で、resume の Scope は footer の形のまま)。分割の前後で全 fixture の結果が同じであること(`TestClassifyResumeScreen_Fixtures` と `TestResumeInputHolds` が通る。順を入れ替える変異 R1 が 3 つのテストで落ちる)も書いた。R1 は 457 行目の cross-review の fix R1 と取り違えないよう「その branch の test report の変異 R1」と書いた

### V6 と L12 の (c)、L11 の後半: 478 行目

- L7: 「four places」を「five places」にして、`sendInputHolds` を列挙に足した(`ResumeInputHolds` がしていた確認を `f24290b5` が引き取った、と添えた)
- Trigger「(L2, L7, L9, L10) the next change to `send_fallback.go`, `send_screen.go`, or `status_live.go`」は、`f24290b5` と `cea96ea4` が `send_screen.go` を変えて満たしたので、持ち越しの note を足した(L2、L7、L9、L10 は開いたまま。コミットは D3 の番号付きの行の話で、これらの項目ではない)。`send_fallback.go` と `status_live.go` は cycle 2 で変わっていない(`git diff 181ecbb3 HEAD --stat` で確かめた)
- L11 の後半: self-review が観察として残している(`resumeMarkerLineBox` が `marker` と `first` だけを埋めた box を返す。本番の呼び出し 2 つは返り値をそのまま `resumeInputBoxShape` に渡すので、今は害がない)ので、(L11) として Debt、Impact、Why deferred、Trigger に足した。行の名前「Seven small findings」は、report や plan が引くので変えず、8 件目の追加は行頭の文に書いた。Related の self-review には、cycle 1 の版 `71e8bfe3` と cycle 2 の版 `c93f854b` を添えた

### I4: 473 行目の Known gaps (b)

Debt に、`claude-input-typed-numbered-steps.txt` と `codex-input-typed-numbered-steps.txt` は合成で、実機の claude と codex に番号付きの TASK を打った画面ではないことを書いた。実機の画面で見たのは、番号付きの手順が claude では文の途中の行の印に、codex では見えている先頭の行に付くことまで(S3b の画面。verify の I4 のとおり)。Impact には、実画面が合成と違うと、番号付きの TASK は打たれたまま送られず、Enter の前の確かめのエラーになる(`f24290b5` の前と同じ側)ことを足した。Trigger (b) には、分離した herdr server の recipe で claude と codex の座席に `sendNumberedStepsMsg` を打って入力欄の画面を採り、2 つの fixture と差し替えるか並べる、を足した(verify の「小さな次の一手」)。

### I1: 計画の Progress

`Branch created` を `[x]` にした。1 行だけの編集で、digest は `1db244296e07` のまま。`PR created` は `/pr` が付ける。

## doc drift の確認(`f24290b5`、`cea96ea4`)

| 文書 | 結果 |
|------|------|
| `docs/specs/2026-08-01-org-runtime.md` の 2026-11-12 改訂 | (c) が Enter の前の確かめを書くので、番号付きの入力欄の行の 1 文を足した。ほかの (a)〜(g) は、`f24290b5` のあとも合う(打つ前の判定、Enter のあとの確かめ、resume の判定は変わっていない) |
| `ralph org send --help`(`internal/cli/org.go`) | 変えない。「checks that the text is in the input box before it presses Enter」は番号付きの場合も正しい。粒度は help のほうが粗く、help の固定テスト(`TestOrgSend_HelpDescribesTheScreenFallback`)も触れない |
| `/org` skill の send の行、`/director` skill、recipe `isolated-herdr-server.md` と `templates/base/` の双子 | 変えない。「入力欄に残った文や積まれた文には何も入力しない」は打つ前の判定の話で、番号付きの行の扱い(打ったあとの Enter の前)とは別。「Enter の前には本文が入力欄に入ったことを確かめ」も合う。sha256 は cycle 1 のあと、origin/main の取り込みで変わった(org skill と director skill の写し)が、4 つの写しは同じ値(verify の cycle 2 が読んだ) |
| `AGENTS.md` の `internal/org/` の行、`README.md` | 変えない。`sendInputHolds` for the check before Enter の書き方のままで合う |
| `docs/quality/quality-gates.md`、`docs/architecture/repo-map.md`、`.claude/rules/ralph/agent-messaging.md`、leader と reviewer の雛形 | 変更不要(send の待ちや確かめの内容に触れていない) |

## 掃引で足したもの

Trigger 列を python で切り出し(`" | "` で分けた 4 つ目のセル、取り消し線の中を外す)、`f24290b5`、`cea96ea4`、`88742506` が変えた関数、file、fixture、test の名前(`ResumeInputHolds`、`findResumeInputBox`、`resumeInputBoxShape`、`resumeMarkerLineBox`、`holdsHead`、`sendTypedInputBox`、`sendCodexHasFooter`、`SendScreenSubmitted`、`sendInputHolds`、`sendFallbackHerdr`、`resume_screens.go`、`send_screen.go`、`testdata/resume` ほか)で探した。

| 行 | 判断 |
|----|------|
| 365 | 足した。cycle 1 の note は `send_screen.go (570)` と書いたが、`f24290b5` と `cea96ea4` のあとは 647 行(`wc -l`)。日付つきの追記で 647 に直し、`send_fallback.go` 379 と `status_live.go` 156 は変わらないこと、`verbs.go` 2597、`spawn.go` 3101、`verbs_test.go` 5893、`spawn_test.go` 6408 も cycle 2 で動いていないことを書いた。merge で `prompts_test.go` は 2722、`director_skill_sync_test.go` は 1307 になったが、origin/main の変更で、この branch が編集したものではない(`git log --no-merges 181ecbb3..HEAD` で確かめた) |
| 461 | 足した。cycle 1 の note の「(its Test gaps 6)」は、test report の cycle 2 の版では Test gaps 5 になった。cycle 1 の版 `d7086a68` と cycle 2 の版 `dd5e08ff` を添えた。F2・F3・F6 が未変異のままであることも、cycle 2 の版が言う |
| 460 | 当たらない。`resumeCodexFooterLine` と `resumeReasonInputHasText` は cycle 2 で動いていない(`resume_screens.go` の差分は `ResumeInputHolds` と `findResumeInputBox` だけ)。`testdata/resume` の新しい fixture は合成の入力欄で、Trigger の「codex の画面の次の probe」ではない |
| 448、449 | 当たらない。448 は取り消し線の RESOLVED の行。449 の Trigger は `kind=enter_only` の観察で、`ClassifyResumeScreen` の結果は変わっていない |
| 450〜452 | cycle 1 と同じ判断。Trigger は実機の観察で、`resume_screens.go` の変更ではない |
| 457 | 上の「V5」で足した |
| 480(新) | 追加した。`send_screen_test.go`(662 → 831)と `send_fallback_test.go`(786 → 835)が `f24290b5` で 800 行を超えた。verify の行数の確認は test でない file だけ(`send_screen.go` 647、`resume_screens.go` 655、`send_fallback.go` 379)を見るので、どの report も挙げていない。handoff に無い追加で、不要なら 480 行目と 365 行目の「(the row that starts with those two file names)」の参照を落とせば済む。Trigger は、(8)(9) のテストを足すときに先に、番号付きの行のテストと時間のテストを別の file に移す、と提案した形 |

## Verification

- `git show HEAD:docs/tech-debt/README.md`(`39ad3434` の前の `dd5e08ff`)との比較(`tdcheck.py`)で、変わった行は 365、457、461、473、478、479 の 6 行と、末尾に足した 480 行目。変わった行も新しい行も列は 5、列ごとのバッククォートと `~~` は偶数。足した文に `file:line` は 0。`difflib` で、行の中の消えた字は意図した所だけ(457 の Impact の `, (e)` と cycle 1 の note の (e) の文、478 の `four`、479 の Related の `;`)
- 計画の参照: `docs/tech-debt/README.md` の `docs/plans/(active|archive)/...` が 120 件、無い参照 0 件
- `./scripts/plan-visual.sh digest` は編集の前後とも `1db244296e07`。`git diff --numstat` は 1/1 行
- `./scripts/check-sync.sh` は `PASS: all files in sync.`(KNOWN_DIFF 6)、`./scripts/check-skill-sync.sh` は 14 skill で ok、`git diff --check` は rc 0。変えた 4 ファイル(tech-debt、spec、計画、jsonl)の U+FFFD は 0 件
- `/natural-japanese` の lint: spec の改訂の項(10 行を scratch に抜き出した)は「和欧文間の半角空白」1 件(7 行にまたがる)。cycle 1 と同じで、この spec が英字の前後に空白を入れる書き方なので残した
- 実行していないもの: テストと `run-verify.sh` の全体(Go のコードを変えていない)。C10 や、`box.footerStart < len(lines)` を外す変異(読んだだけ)

## 残っていること(lead へ)

- 計画の本文と Go のコードは変えていない。パイプラインの再走は要らない。この run の変更は docs だけ(tech-debt、spec、計画の 1 つの箱、この report、insights)
- 決めてほしい点: (1) 479 行目の番号を cycle 1 のままにして、test report の cycle 2 の版との対応を冒頭に書いた。付け替えるなら 479 行目だけの編集で済む。(2) 480 行目は handoff に無い追加。(3) 371 行目(recipe の `docs/evidence/` の引用)は cycle 1 から、新しい 1 件を言い換えるか決まっていない
- cycle 1 の節の値が、今は古い所: 行番号(472、473)は origin/main の取り込み(`39004c89`)で 478、479 になった。digest `123428594289` は `1db244296e07`。「I1: `Branch created` は付けていない」は付けた。「Test gaps 1〜7」と「L7: four places」は cycle 2 の run が更新した。cycle 1 の節は書き換えず、ここに書いた
- `docs/tech-debt/README.md` は別のセッションも編集しうる。この run の commit は、6 行への挿入と、末尾への 1 行の追加で、既存の行は消していない。merge の衝突は末尾の追加行で起きやすい
- `docs/plans/active/...` への参照を Related に足した。`/pr` の `archive-plan.sh` が書き換える

## Insight event

```
./scripts/insights-append.sh --slug send-screen-fallback --flow standard --phase sync_docs \
  --verdict pass --cycle auto --source skill
```

終了コード 0。`docs/insights/events/2026-11-12-send-screen-fallback.jsonl` に足した行は cycle 2、pass、ts は 2026-11-12T15:54:56Z。この時点の `git status --short` は、jsonl、計画、spec、tech-debt の `M` だけで、迷子のファイルは無かった。
