# sync-docs report: send-screen-fallback

- Date: 2026-11-12
- Plan: `docs/plans/active/2026-11-12-send-screen-fallback.md`(承認の digest `123428594289`)。issue #308
- Branch: `fix/send-screen-fallback`。base は origin/main `d2e5c6e9`(`c31780b4` で取り込み済み)。run 開始時の HEAD は `d7086a68`
- Pipeline run: 1 回目(`cycle-count.json` は `cycle` 1)
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
