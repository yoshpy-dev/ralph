# Walkthrough: 機能の org の座席を disband せずに止めて立て直す `ralph org reseat`(org-reseat)

- 計画: `docs/plans/archive/2026-11-12-org-reseat.md`(PR の作成のあとに archive へ移す)
- issue: #312 の c の 1 本目(#312c-1)。経過の節と `leader.md` の規則(#312c-2、仕様の D7・FR-9・FR-10・AC-13・AC-14)は 2 本目で、issue は開けたままにする
- 差分: `git diff 7433ce84...HEAD`(37 ファイル、+9,360 / −91。walkthrough を足す前に数えた)。Go の本体は 8 ファイルで +2,195 / −34(新規の `internal/org/reseat.go` が 1,537 行、`internal/cli/org_reseat.go` が 521 行)、Go のテストは 4 ファイルで +3,446、残りの 25 ファイルは +3,719 / −57(skill の 8 つの写し、報告・実機の記録・計画・仕様)
- origin/main は 8 回取り込んだ(26f3ef44、f0c53cde、9a295a84、8fbd7118、8460bf64、46dfc5a8、c712526e、26f2e169。後ろの 4 つが取り込んだのは e902705c、bbe2807d、c53bf39c、7433ce84)。main 側の中身はどれも 7433ce84 の祖先なので差分に入らない。衝突を解いた箇所(README、tech-debt、skill など)の結果だけが、この branch の変更として入っている

## 1. Reseat の流れ(読む価値が一番高い)

`internal/org/reseat.go` の `(*Org).Reseat`。`--dry-run` は `PlanReseat` だけを呼んで返す。

- 本当の reseat は、フラグの検査、`reseat.lock`(待たずに 1 回取る。取れなければ持っている reseat の org を言って断る)、`PlanReseat`、ロックの marker への reseat id の書き込み、`stopAll`、全部止まれば `startAll`、`finish` の順に進む。
- `stopAll`: leader 以外を立てる順の逆に止め、最後に leader を止める。座席ごとに、止める直前に台帳を読み直す。その座席の最新の state event が検査のときと違えば、止めずにそこで終える(`ReseatResult.Changed`)。`Stop` が失敗(`stop_failed`)したときもそこで終え、どの座席も立てない。
- `startAll`: leader を `StartFeature`(`start --plan` と同じ道)で最初に立て、ほかの座席を `Spawn` で元の spawn の順に立てる。全座席を止めたときは、leader が新しい workspace の最初の pane になる。立てるのに失敗した座席で終え、あとの座席は立てない。
- `finish`: 座席を 1 つでも止めるか立てようとしたときだけ、org レベルの `reseated` を 1 件書く(`id=<id> seats=<座席>:<結果>,...`。`--driver`・`--model` で変えた座席は `changed=`)。途中で終えたときも書く。検査で断られた reseat と、最初の読み直しで終えた reseat は何もしていないので書かない。
- 変えていないもの: 計画の承認(`plan_approved`)、予約、機能との結びつき。`Stop` は org の命を終えないので、止めて立てても引き継がれ、立て直した implementer の spawn と TASK は承認を求め直さない。reseat は worktree も作らない。

## 2. 止めたあとにほかのコマンドと重なる場面

`reseat.lock` が止めるのは別の reseat だけで、`Stop` と `Spawn` は座席のロックを互いに取らない(前からの型)。

- 止める前は、1 の読み直しで守る。
- 立てるときの穴は cycle 2 の cross-review(codex、ACTION_REQUIRED 1 件)が見つけた。止めたあとにほかのコマンドが同じ座席を立てると、`Spawn` は idempotent を返す。それまでの `startAll` は、これを立てた座席と数え、`--driver codex` を頼んだのに claude の座席が動いていても成功と報告した。
- `reseatIdempotentMismatchErr`(def8d2a3): 返った座席の role・driver・model(model `""` どうしは同じ)が計画と違えば、その座席を動かしたまま `start_failed` にし、error に両方の設定を書いて、あとの座席を立てずに終える。
- `--driver`・`--model` が渡っていない座席は `Changed` にして、`reseatRetryCommand` の `--seat` から外す(df8f674e)。入れると打ち直しは、動いている座席をその座席が今動いている設定で立て直すだけになる。渡されていれば入れる。help に同じことを書いた(b142944d)。
- org の disband の割り込みは cycle 3 の cross-review(ACTION_REQUIRED 1 件目)が見つけた。`Disband` は `reseat.lock` も座席のロックも取らず、leader が自分の pane で打つこともある。止めたあとに disband されると、`Spawn` は結びつきの無い org として承認の検査を飛ばして座席を立て、`StartFeature` は前の承認を持たない新しい命を作っていた。
- 計画の時点の org の最後の `disbanded` の位置を `ReseatPlan.LastDisbandedIndex` に持たせ、止める直前(`checkBeforeStop`、c4f55aef)と立てる直前(`endIfDisbanded`、84346015)に台帳を読み直して比べる。違えばそこで終え、`ReseatResult.Disbanded` を立てる。`finish` は打ち直しも DECISION も作らない(disband のあとの reseat は結びつきが無いので断られ、DECISION の宛先の leader もいない。f2ed6bdf のテストが固定する)。error は disband の時刻、`ralph org start --plan` で立て直すこと、disband の前に立てた座席を言う。出力は止めていない座席を `left running:` でなく `not stopped: <座席> (the org was disbanded meanwhile)` と出す(b568ca8c)。
- 残る窓: 読み直しと `Stop`・`Spawn`・`StartFeature` の書き込みのあいだ。tech-debt の Known gaps に書いた。

## 3. 止める前の検査: `PlanReseat` と `StartFeature` の切り出し

- `PlanReseat` は台帳に書かず、herdr も agmsg も呼ばない。順に見るのは、フラグ、座席の pane(`HERDR_PANE_ID`)からの実行、結びつきの有無、対象、独自の役割、leader の入力(`startFeatureInput`)。そのあと、台帳の写しに止める座席の `stopped` を足して、leader の `checkLedger`(leader が対象のとき)か予約・regate・worktree の検査、worktree の記録(`reseatWorktreeRecordErr`。記録が無いと断る点で `start` より厳しい)、非 leader の座席ごとの `spawnPrecheckErr`(通るたびに写しへ `spawned` を足し、上限の数え方を実行時に合わせる)。最後が見張りと herdr server のゲートで、本当の reseat ではここで必要なら起動する。
- 判定を二重に書かないため、`internal/org/feature.go` の `StartFeature` の検査を `startFeatureInput`(台帳を読まない)と `startFeatureChecked.checkLedger`(events を引数に取る)に切り出した。`StartFeature` の順序と動きは変えていない(入力の検査が見張りと server のゲートより前)。
- `--dry-run` は見張りと herdr server を起動せず、動いているかを読むだけにした(`internal/cli/org_reseat.go` の `wireOrgReseatDryRunGates`)。動いていなければ「本当の reseat が起動する」と出す。

## 4. 対象の決め方

- `reseatLifeSeats` が、台帳の今の命(最後の `disbanded` のあと)から座席ごとの最新の state、最初の `spawn_started` の位置(立てる順)、`reseat=<id>` の note の有無(`touched`)、切れた reseat の印(`cutOff`)を読む。
- 既定の対象は active な座席すべてと、途中で切れた reseat が止めた座席(最新が `reseat=<id>` つきの `stopped` で、その id の `reseated` が無い)。leader が意図して止めた座席は蘇らせない。
- `--seat` が受け付けるのは、今の命で一度 `spawned` になった座席、今 active な座席、reseat が止めたか立てようとした座席。reseat が出す打ち直しのコマンドが、自分の `--seat` で断られないための条件である。手で止めただけの座席は断る。
- 断るもの: 結びつきの無い org(`ralph org stop` と `spawn` の打ち方を出す)、役割の prompt の雛形が無い独自の役割の座席(台帳は `--prompt` を持たない)、active な座席も `--seat` も無いとき(受け付ける座席を並べる)。

## 5. CLI と出力

`internal/cli/org_reseat.go`。`newOrgReseatCmd` は `--seat`(繰り返せる)、`--driver`、`--model`、`--dry-run`、`--timeout-ms` を持つ。`runOrgReseat` は設定と台帳を `start --plan` と同じ関数で読み、`--config` は絶対パスにして、出力が打つコマンドへ引き継ぐ。

- 出力は `stopped:`、`started:`、失敗した行、`left running:` などの行、成功なら `reseated org_id ... (reseat=<id>)`、打ち直しのコマンド、DECISION の順。終了コードは `Reseat` の結果のまま。
- leader が対象に無く座席を立てたときは、leader へ送る DECISION の `ralph org send` の行を、leader の pane を読む `ralph org read` の行と一緒に出す。reseat は送らない(送り手の名乗りと待ちを動詞に抱えさせないため)。途中で失敗したときも、立った座席があれば出す。どちらの行も、打ち直しと同じく `--config` を絶対パスで引き継ぐ(ffd32109。cycle 3 の cross-review の 2 件目。send も read も `--config` か `./ralph.toml` を読むため)。
- 本当の reseat は `orgReseatHoldSignals` の中で動く。SIGINT・SIGTERM・SIGHUP は 1 行を stderr に出して受け止め、SIGPIPE は黙って受け止める。止めて立てる処理を最後まで進め、`reseated` を書くためである。dry-run は受け止めない。
- 変えていないもの: pane の環境。`CLAUDE_CONFIG_DIR` を pane ごとに渡さない(仕様の D6)。別のアカウントへ移すときは、人が herdr server を止めてから新しい環境の shell で reseat を打つ。

## 6. 既存コードの小さな変更

- `StopParams`・`SpawnParams`・`StartFeatureParams` に `Note` を足した(`verbs.go` の `noteDetails`)。Details の末尾に `reseat=<id>` が付く。空なら Details は今と同じ。
- `seat.go` に `EventReseated`。座席の state を変えない org レベルの event で、古い binary は読み飛ばす。
- `internal/cli/org.go` に動詞の登録と `--config` の help の追記、`org_legacy_ledger.go` はコメントに `reseat` を足しただけ。`internal/org/prompts/` は変えていない。

## 7. テスト

- 新規の 4 ファイルに、トップレベルで 64 本(`internal/org/reseat_test.go` 36、`reseat_start_test.go` 6、`reseat_disband_test.go` 9、`internal/cli/org_reseat_test.go` 13)。偽の herdr と偽の worktree で `Reseat` を回す。disband のテストは、偽の driver の上で本物の `Disband` をフックから打つ。
- 順序と記録: `TestReseat_StopsAndStartsEverySeatInOrder`、`TestReseat_RecordsItsIDOnEveryStopAndStart`、`TestReseat_LeaderIsTheFirstPaneOfANewWorkspace`、`TestReseat_SeatsWhosePanesAreGone`。
- 重なりと失敗: `TestReseat_RefusedWhileAnotherReseatHoldsTheLock`、`TestReseat_ASeatChangedBeforeItsStop`、`TestReseat_AStopThatFailsStartsNothing`、`TestReseat_RetryAfterAFailedStart`、cycle 3 の `reseat_start_test.go`(別設定で立てられた座席、同じ設定、leader、`Changed` の打ち直し)。
- 検査: `TestPlanReseat_RefusesWhatTheStartsWouldRefuse`、`TestPlanReseat_DryRunStartsNoWatchOrServer`、`TestStartFeature_InputRefusalsComeBeforeTheGates`。signal は `TestOrgReseatHoldSignals_*` が見る。
- disband の割り込みは `reseat_disband_test.go`(最初の止める前、止める途中、立てる前、leader を立てたあと、leader 以外を立てたあと、台帳が読めないとき)。
- 変異は cycle 1 の 10 個のうち 5 個が生き残り(Test gaps に残した)、cycle 2 の 11 個と 3 回目の 6 個はすべて落ちた。4 回目は 11 個のうち 9 個が落ち、生き残った 2 個は CLI の出力への配線で、CLI のテストからは割り込みを起こせないため(Test gaps に残した)。

## 8. 文書

- skill: `/org` の動詞の表に `reseat` の行、「座席の立て直し」の節(順序、重なり、切れた reseat、独自の役割)。`/director` に「座席を立て直す」の節(人が頼んだときだけ打つ、先に `--dry-run`、アカウントを移すときは人が `herdr server stop`)。どちらも 4 つの写し。
- `README.md` の動詞の行と段落、`AGENTS.md` の `internal/org/` の行、`docs/quality/quality-gates.md` と `templates/base/` の写しの見張りと herdr server のゲートの行(real reseat を足した)。
- 仕様 `docs/specs/2026-11-12-org-reseat.md`(新規)と、`docs/specs/2026-08-01-org-runtime.md` の 2026-11-12 改訂 (a)〜(d)。
- `docs/tech-debt/README.md`: 既存の行に持ち越しの注を足し、新しい行を 5 つ足した(小さな指摘 14 件、800 行を超えたファイル、テストの穴、`Disband` が `reseat.lock` を取らない窓、`ApprovalDecisionCommand` と `orgReadCommandHint` の doc の `--config`)。

## 9. 実機の確認(AC-12 は一部だけ)

`docs/evidence/org-reseat-live-2026-11-12.md`。分離した herdr の上に、leader と reviewer の機能の org を立てて 2 回打った。

- AC-1: reviewer、leader の順に止め、leader、reviewer の順に立てた(終了コード 0、15 秒)。leader は新しい workspace の最初の pane に入り、`scope_reserved` は増えず、`reseat=<id>` と `reseated` が付いた。
- AC-8: herdr server を止めて立て直すと、0.7.5 は pane の記録だけを戻し、端末もプロセスも無かった。reseat は Ctrl-C の失敗(`pane_not_found`)を記録して pane を閉じ、新しい workspace に 2 席を立て直した(16 秒)。
- 開いた問い: 全座席が agmsg の team から Leave しても、team と履歴は消えなかった。
- 確かめていないもの: implementer と承認の検査、`--seat`・`--driver`・`--model`、途中の失敗、ロックの競合。binary は dd204370 で、切れた reseat の座席、独自の役割、SIGPIPE、cycle 2・3 の直しより前のものである。これらはテストだけで確かめている。仕様の AC-12 は、この限りを書いたうえで `[x]` にした。verify の報告は AC-12 を一部の確認とし、計画の AC-12 は `[ ]` のまま残した。

## 10. パイプラインの経過

4 回走らせた(cycle cap は 2 から 3 に上げた。3 回目の cross-review の再実行は、skill の非上限の手順で cycle を 3 にした)。報告は `docs/reports/{self-review,verify,test,sync-docs}-2026-11-12-org-reseat.md`。

- cycle 1: cross-review の ACTION_REQUIRED 3 件(後の失敗のあとの打ち直しと通知、絶対パスの `--config`、`--seat` の受け付ける座席)を adc3e205、e9461b79、0468d96f で直した。
- cycle 2: cross-review の ACTION_REQUIRED 1 件が 2 の穴。cap を 3 に上げ、def8d2a3、df8f674e、b142944d で直した。
- 3 回目: self-review は blocking 0、verify と test は pass。test の Go の段は負荷で 30 分の時間切れが出て、同じ上限で打ち直して通った。cross-review の ACTION_REQUIRED 2 件(disband の割り込み、DECISION の send と read の `--config`)を ffd32109、84346015、c4f55aef、b568ca8c で直し、f2ed6bdf で DECISION が出ないことのテストを足した。
- 4 回目: self-review は blocking 0(LOW は L26 まで)、verify は pass、test は pass(run-test.sh の 1 回で Go の全パッケージが時間切れなしで通った)。sync-docs は ee784d9c で、計画の digest が変わったので承認を 19fb585e で記録し直した。origin/main を c712526e(PR #330)と 26f2e169(PR #331)で取り込み、cross-review(cycle 3/3)は指摘なし。
- 残ること: 2 本目(経過の節と `leader.md`)、`CLAUDE_CONFIG_DIR` を org ごとに持たせる案(別の issue)、tech-debt の Known gaps。
