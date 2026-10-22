# Self-review report: seat-spawn-close

- Date: 2026-10-22
- Plan: `docs/plans/active/2026-10-22-seat-spawn-close.md`(パイプライン cycle 1)
- Reviewer: reviewer サブエージェント(`/self-review`)
- Scope: `git diff 5850f89b...HEAD`(HEAD 4ebc315e、21 ファイル、+2240/-247)。S1 80780740(失敗した spawn が pane を閉じる)、S2 97782369(座席ごとの spawn のロックと `compensateStale` の確かめ)、S3 abb11e8c(`--timeout-ms` の help と期限の案内、skill 4 写し)、S4 57a124a4(実機の evidence)、S5 a7133901(tech-debt・仕様・根拠の表)、計画の 2 コミット。diff の質だけを見た

## Evidence reviewed

- Go の diff を全部読んだ。`internal/org/spawn.go`(`Spawn` のロックと Phase 1/2、`failStepWithNote`、`closeSpawnPane`、`compensateLeave`、`compensateStale`、`refuseStaleRespawn`、`rejectUnlessInFlight`、`inFlightSeatErr`、`spawnPrecheckErr`)、`seat.go`、`verbs.go`(`labelMismatchError`、`confirmSeatPane`)、`lockfile.go`、`internal/cli/org.go`。テストは `spawn_test.go` の追加分(fake の変更と新しい 18 本)、`spawn_lock_test.go`、`verbs_test.go`、`verbs_all_test.go`、`internal/cli/org_test.go`、`org_spawn_timeout_test.go` を読んだ。テスト、gofmt・vet、spec 突き合わせの実行は範囲外で、していない
- 依頼の観点ごとの確かめ方。pane とロックの漏れは、`Spawn` の `return` を `grep -n 'SpawnOutcomeFailed\|failStep'` で全部出し、tab_create の後の道を 1 つずつ読んだ。補償の期限は、`spawn.go` で `context.Background()` を探し(残りは 673 行の spawn 全体の ctx だけ)、`callWithTimeout` の呼び出しを数えた。ロックの解放は `defer` の位置と、`os.OpenFile` が `O_CLOEXEC` で開くこと(herdr と agmsg の子が fd を持ち越さない)から読んだ
- `failStep`・`failStepWithNote` の呼び出し 10 か所と `compensateLeave`・`closeSpawnPane` の呼び出し元を全部見た。旧名 `compensatePane`・`compensatePaneCtx` はコードに残っていない(`grep`)
- 新しい event を列挙する読み手の掃引。`EventSpawnFailed`・`EventStopFailed` を `internal/` の非テストで引き、event 名を並べて読む場所は `seat.go` の `stateEvents`、`watch.go:1328` の `leaderActivityEventCount` の 2 つと分かった。`spawn_close_failed` は前者に入れず(意図どおり)、後者には入れていない(L2)
- skill の 4 写し(`.claude/skills/org`、`.agents/skills/org`、`templates/base/.claude/skills/org`、`templates/base/.agents/skills/org`)を `cmp` で比べた。バイト一致。追加した文は `spawn`・`start` の表の欄と「無期限待機は避ける」の項で、コードの振る舞い(ロックの断り方、10 秒、`spawn_close_failed`、label が違うときは触らない)と合う
- 記録の突き合わせ。tech-debt の diff は 4 hunk で、`git diff --check` は空。新しい行 3 つ(338〜340)は、区切りの `|` を `re.split(r'(?<!\\)\|', ...)` で割ると 7 要素(5 欄)で、base の行と同じ形。F4 の行(320)は Debt と Trigger を取り消し、RESOLVED のコメント(319)と取り消し線、`(RESOLVED ...)` の 3 点がそろう。F3 の行(318)は `fix/seat-agmsg-self-name` を `#248` に直してある。org-stop-all の行(173)は (b) を Debt・Impact・Trigger の 3 欄で取り消した。行 340 が挙げる `return` は `spawn.go` の 974、1077、1100、1116、1145、1168 行に実在する
- 追加行を `/Users/`、`hiroki`、token 形の文字列、`TODO`・`FIXME`、`fmt.Print`・`println` で grep した。当たりなし。evidence の文書は `~` と `<D>`・`<pane>` に置き換えてある
- 確かめていない: Go のテストの実行結果(`/test` の範囲)、`go vet` と `gofmt`、仕様の `[x]` の数(26/6、25/6)と根拠の表の突き合わせ(`/verify` の範囲)、実機の evidence の数字(時刻、行数 51 など)の再現

## Findings

<!-- Area recommended values: naming, readability, unnecessary-change, typo,
     null-safety, debug-code, secrets, exception-handling, security, maintainability -->

| Severity | Area | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- |
| LOW (L1) | exception-handling | 新しい復旧の案内 4 か所が `ralph org stop --org-id <o> --seat <s>` を書き、`--state-dir` を持たない。feature org の leader は ralph が `--state-dir` を pane に渡さないので、`ralph org` の全コマンドに `--state-dir <台帳>` を付ける決まり。leader がエラーの案内をそのまま打つと、既定の台帳を読む。同じ名前の org と座席が別の台帳にあれば、別の座席を止めうる。なければ座席が見つからないだけ。`herdr pane close <id>` も書く 2 か所は、そちらが台帳によらず通る。stop だけを書く 2 か所(`inFlightSeatErr`、自分の pane の断り)は、この新しい状態から出る唯一の道が台帳違いになる。同じ文が 4 か所に手書きされているのも、案内を直すときの抜けやすさに効く | `internal/org/spawn.go:2180`(`inFlightSeatErr`)、`2449`(`failStepWithNote`)、`2541`(自分の pane)、`2576`(`refuseStaleRespawn`)。先例は `internal/org/escalate.go:104-112`(`InboxCommand` は `--state-dir` を明示のときだけ足す)と `internal/cli/org.go:569`(`orgReadCommandHint`)。`internal/org/prompts/leader.md:70-76` が `--state-dir` を付ける理由を書いている | `SpawnParams` に、CLI が `--state-dir` を明示されたときだけ入れる値(`EscalateParams` の `stateDir` と同じ)を足し、4 か所の文を 1 つの関数(`stopSeatHint(orgID, seatID, stateDir)`)にまとめる。足さないなら、各文に「spawn に渡した `--state-dir` を付ける」の 1 句を入れる |
| LOW (L2) | maintainability | leader の活動を数える `leaderActivityEventCount` は、leader が打つ verb の event を数える集合(`spawned`、`spawn_started`、`stopped`、`stop_failed`、`disbanded`、`rejected`)を持ち、doc は「herdr が答えない間 stop を打ち直す leader は動いているとして数える」と理由を書く。この PR で、leader の打った spawn が数えられない道が 3 つ増えた。(1) 途中の座席への検査拒否は、以前 `rejected`(数える)を書き、今は何も書かない(D6)。(2) `compensateStale` の拒否は `spawn_close_failed` だけを書き、`spawn_started` を書かない。(3) ロックによる断りは何も書かない。計画の Risks は「`spawn_started` がすでに同じ spawn を数える」を理由に `spawn_close_failed` を数えないとしたが、これが成り立つのは `failStepWithNote` の道だけで、(1)(2) には当たらない。結果として、止まりの ALERT が保留中に leader が断られる spawn を打ち直し続けても、deadman は人への escalation まで進みうる。断られる spawn が「止まり」なのか「動いている」のかは決めてよいが、決めた記録がどこにもない | `internal/org/watch.go:1256`(doc の stop 打ち直しの文)、`1328`(集合)。`internal/org/spawn.go:2164`(`rejectUnlessInFlight`)、`2579`(`refuseStaleRespawn` が書く event)。計画 `docs/plans/active/2026-10-22-seat-spawn-close.md` の Risks「新しい event が…」の行 | 決めて 1 つ記録する。(a) `EventSpawnCloseFailed` を集合に入れる(`stop_failed` の先例どおり。古い watch が走る間は基準より 1 多く読む、という `stop_failed` の注意が同じく当てはまるので、`watch.go` の doc に 1 文足す)、(b) 数えないと決め、`leaderActivityEventCount` の doc に「断られた spawn は数えない」と書き、tech-debt に 1 行足す |
| LOW (L3) | readability | `failStepWithNote` の引数が 7 つになり、`error` が 2 つ(`cause`、`deadlineErr`)と `string` が 3 つ(`paneID`、`team`、`note`)並ぶ。呼び出しは `nil` を 5 か所、`""` を位置の真ん中に 2 か所渡す。`paneID`・`team`・`note` はどれも `string` なので、入れ替えても型は通る。`deadlineErr` は「step が返った直後に呼び出し側が `ctx.Err()` を読む」という約束で、doc にしか書かれていないので、次の呼び出しが `ctx.Err()` を Leave のあとで読む形にしても型では防げない(HELLO の道は、まさにそのためにローカル変数を作っている) | `internal/org/spawn.go:2438`(宣言)、`995`、`1043`、`1046`、`2240`、`2252`(`nil`)、`1060`(`""`)、`1139`(`team` と `note` の 2 引数)、`2380`(`failStep`) | `deadlineErr error` を `deadlineHit bool`(呼び出し側が `errors.Is(ctx.Err(), context.DeadlineExceeded)` で作る)にすると `cause` と取り違えない。`team` と `note` は小さい struct か、`failStepWithNote` の前に `note` を作る形に寄せる |
| LOW (L4) | naming | `compensateStale` が書く `compensation=` の語が揃っていない。`ctrl_c=skipped pane=already closed`(2549)、`ctrl_c=skipped pane=left untouched, labelled differently: <err>`(2551)に対し、確かめられない道だけ `ctrl_c=skipped: pane not confirmed pane=not closed: <err>`(2553)で、`skipped` の直後にコロンが付く。`ctrl_c=failed: <err>` ではコロンの後ろが誤りの文なので、`skipped:` の後ろも誤りに読める。doc の語の一覧(2512-2530)にもこの語がない。Details は `ralph org report` の timeline で人が読むので、`key=value` の形がずれると grep しにくい | `internal/org/spawn.go:2549-2553`、doc は `2512-2530`、テストの固定は `internal/org/spawn_test.go:2161` ほか 5 件(`pane=not closed: ...`) | `ctrl_c=skipped pane=not confirmed, not closed: <err>` のように、`ctrl_c=skipped pane=<状態>` の形に揃える。doc の一覧に 1 行足し、テスト 6 件の期待文を合わせる |
| LOW (L5) | maintainability | org-stop-all の行(173)は、この PR が (b) を Debt・Impact・Trigger の 3 欄で取り消したが、Why deferred の欄が「(b)、(c)、(d) は main から引き継いだもので、(b) は計画の Non-goal」と (b) を先送りの理由に挙げたままで、直した項目を先送りしたと読める | `docs/tech-debt/README.md:173`(欄 4。`split` で 7 要素のうち index 3) | 欄の文から (b) を外す(「(c) と (d) は…」にする)か、その文を取り消す |
| LOW (L6) | maintainability | F4 の行(320)は RESOLVED だが、Debt 欄の取り消しの外に、まだ分からないこと(「agmsg がなぜ遅かったかは、まだ分からない。join が 17 〜 81 s かかった」)を書いている。開いている行でこれを追うのは F6 の ALERT の行(335)で、その Trigger は「`sendAlert` の次の変更か、負荷の下で watch を動かすとき」に `send.sh` の 10 s を調べるとしか書かず、spawn の join と HELLO の遅れ(F4)を挙げていない。取り消した行に残った問いは、今の Trigger のどれにも当たらない | `docs/tech-debt/README.md:320`(欄 1 の末尾の文)、`335`(欄 4) | 行 335 の Trigger に「F4 の spawn の join と HELLO の遅れ(17 〜 81 s)も同じ負荷の下で測る」を 1 文足し、320 の文から 335 を指す |
| LOW (L7) | maintainability | `spawn.go` が 2357 行から 2634 行(+277)、`spawn_test.go` が 4688 行から 5713 行(+1025)になった。800 行の目安(code review の規約)をすでに超えていたファイルを、さらに伸ばした。この repo は、PR がファイルを 800 行の先へ進めたとき tech-debt に行を足してきた(282、283、296、309 行)が、`spawn.go` と `spawn_test.go` の行はない(`grep` で確かめた)。新しいテストは 1469〜2405 行の連続した 936 行で、`spawn_lock_test.go` が示すように fake を共有したまま別ファイルに置ける | `wc -l`、`docs/tech-debt/README.md:282-309`、`internal/org/spawn_test.go:1469-2405` | テストの 936 行を `spawn_close_test.go` に移す(移すだけ、コードは変えない)。移さないなら、`spawn.go`・`spawn_test.go` の大きさの行を 1 つ足す |
| LOW (L8) | maintainability | 新しいテスト 3 本(サブテスト含め 5 ケース)が `TimeoutMS = 50` で、前の step(workspace と tab の作成、agent 起動、leader と座席の join、manifest の追記 6 回前後)を 50 ms の中で終えることに頼る。間に合わないと失敗が前の step に落ち、step 名の検査が「期待と違う step」で赤になる。同じ 50 ms の先例が `spawn_test.go:3994` にあり、隣の `HelloFailsInsideDeadline` は 500 ms を使う。余裕は測っていない(テストを走らせない範囲) | `internal/org/spawn_test.go:1615`、`1650`、`1901`(50 ms)、`1990`(500 ms)、`3994`(先例) | 200〜300 ms に上げる。止める fake は `ctx.Done()` まで待つので、期限を伸ばしても検査の中身は変わらず、1 ケースあたり 0.2 s ほど増えるだけ |
| LOW (L9) | naming | `appendLegacyRejectedRetry` は名前が「足す」だけを言うが、前半で `r`(`Spawn` の戻り)の outcome・receipt と座席が `spawn_step` に残ったことを検査してから、古い binary の `rejected` を足す。3 か所の呼び出しは `o.Spawn(retry)` を引数に入れて渡すので、検査している行為(spawn)が引数の中に隠れる | `internal/org/verbs_test.go:4582`(宣言)、`4981`、`5046`、`5094` | `r := o.Spawn(retry)` を呼び出し側に出し、検査と追記を 2 つの関数(`assertPlainInFlightRefusal`、`appendLegacyRejected`)に分ける |
| LOW (L10) | maintainability | 「補償の呼び出しは 1 回 10 秒」が 5 か所に書かれ、固定しているのは help の文字列 1 か所だけ。定数は `internal/org` の非公開の `defaultDriverCallTimeout`(`verbs.go:758`)で、help の `10s per call` は `internal/cli/org.go:716` に直書き、skill(4 写し)、仕様の FR-9、tech-debt の行 319 も「10 秒」と書く。定数を変えても help のテストは通り、文字列だけが古くなる。コメント(`org.go:707-713`)が結びつきを認めているが、テストは結ばない。`Org.DriverCallTimeout` を設定する非テストの経路はなく、本番では定数の値で固定される | `internal/org/verbs.go:758`、`internal/cli/org.go:714-717`、`internal/cli/org_spawn_timeout_test.go:15-31` | `org.DefaultDriverCallTimeout` を公開し、help の文字列にそれを書式化して入れる。help のテストはその値から期待文を作る |

重大度の数は CRITICAL 0、HIGH 0、MEDIUM 0、LOW 10。

## Positive notes

依頼で挙がった観点ごとの結果を書く。

- pane の漏れ。tab_create の後で `return` する道のうち、`failStep`・`failStepWithNote` を通らないのは manifest の追記が失敗した道だけで(`spawn.go:974`、`1077`、`1100`、`1116`、`1145`、`1168`)、計画の Non-goals と tech-debt の行 340 が同じ 6 か所を挙げている。`tab_created` の追記が失敗する道だけは、次の spawn の `compensateStale` も拾えない(記録に pane の id がない)と行 340 が書いている。ほかの step は、閉じられなかったときも `spawn_close_failed` が `spawn_step` の pane_id を残すので、stop・`stop --all`・次の spawn の補償が拾える
- ロックの解放。`defer func() { _ = lock.release() }()` は取得の直後(`spawn.go:614` の後)にあり、`Spawn` の `return` がいくつあっても(クロージャの中の `return nil` を含めて 41 行)、解放は `defer` 1 つに頼り、途中で解放を飛ばす道はない。ロックの fd は Go の `os.OpenFile` が `O_CLOEXEC` で開くので、herdr と agmsg の子プロセスに引き継がれず、`Spawn` が返ったあと子がロックを持ち続けることはない。取得に失敗した道(`errFileLockHeld` とその他)はロックを持たずに返る。ロックの順は「座席のロック、次に manifest のロック」の一方向で、`StartFeature`(`feature.go:246`)は manifest のロックの外から `Spawn` を呼ぶ。`withManifestLock` の中から `Spawn` を呼ぶ道は見つからなかった
- 補償の期限。新しい呼び出しは `closeSpawnPane` の 2 回、`compensateLeave` の 1 回、`compensateStale` の `confirmSeatPane` の 3 回で、すべて `callWithTimeout`(1 回 `driverCallTimeout`)を通る。spawn の期限が切れた後の補償が、切れた ctx で打たれる道はない。HELLO の失敗の最悪は Leave 10 s + C-c 10 s + close 10 s、`compensateStale` の最悪は get 3 回 + C-c + close の 50 s で、計画の D4 と doc の書き方に合う。確かめの get が 1 回目で失敗した場合は 10 s で断る
- ロックのテストが固めるもの。`TestOrgSpawn_SeatLock_RefusesASecondSpawnOfTheSeat` は、1 つ目を座席の join で止め、2 つ目が errFileLockHeld で断られ、event・receipt・herdr と agmsg の呼び出しの数が変わらず、1 つ目の pane に C-c も close も届かないことを数える。ロックがなければ、2 つ目は stale と見て `pane_get` から打つので赤になる。`OtherSeatsOrgsAndDryRunsDoNotWait` は、`!p.DryRun` の条件を外すと dry run が断られて赤になる。`LockThatCannotBeTaken` は `create spawn lock dir` の文を見て、held の断りと区別する
- 期限の案内のテストが固めるもの。`HelloFailsInsideDeadline_LeavePastIt` は、期限を `ctx.Err()` で Leave の後に読む実装だとヒントが付いて赤になる(Leave が期限の 300 ms 後まで眠る)。`agent start busy` の例は、`errors.Is(err, DeadlineExceeded)` で決める実装を弾く(原因が期限を包まない)。ヒントが Details に出ないことも見ている
- 補償の fake の変更。`fakeHerdr` と `fakeAgmsg` に足した ctx の記録(`ctxAtCall`)は、`assertLiveBoundedCtxs` が「生きていて期限を持つ」を要求するので、補償が `context.Background()` を使う旧実装(`compensateLeave(context.Background())`)なら赤になる。`fakeAgmsg.Send` と `Join` の mutex を `defer` から手で解放する形に直した点は、待っている間に他の呼び出しを止めない(`holdJoin` が成り立つ)ためで、読み書きの順は保たれている
- 検査の拒否が何も書かない件。`TestOrgSpawn_SeatLeftOpen_AfterRefusedRetry` は 3 つの続き(stop、`stop --all`、正しい再試行)で記録した pane の id だけが閉じられることを見る。D6 を戻して `rejected` を書けば、roster が座席を inactive にして stop が拾わないので赤になる。`spawnPrecheckErr` と `Spawn` の文が一致することも専用のテストがある
- label の確かめの追加。`errLabelMismatch` は `labelMismatchError` の型の `Is` で判定し、`confirmSeatPane` の外側の `%w` でも通る(workspace の不一致を含む)。ほかの呼び出し(`stopSeatPane`、`confirmDirectorResources`)は `err != nil` しか見ないので、テキストが変わっていないことで影響を受けない
- 不要な変更はない。デバッグ出力、TODO、コメントアウトしたコードはない。skill は 4 写しで一致している。`org-stop-all` の行の `compensatePaneCtx` の記述(「`compensatePaneCtx` is gone」)は、コードにその名前が残っていないことと合う

## Tech debt identified

| Debt item | Impact | Why deferred | Trigger to pay down | Related plan/report |
| --- | --- | --- | --- | --- |

追加なし(このレポートは範囲外のファイルに触れない)。L2 は決めた結果によって、L7 は移さないなら、親が `docs/tech-debt/README.md` に行を足す。L5・L6 は既存の行の直し。

_(If any rows were added above, also append them to `docs/tech-debt/`.)_

## Recommendation

- Merge: マージ可(CRITICAL・HIGH・MEDIUM なし。LOW 10 件)
- Follow-ups:
  - 直すなら先に L1 と L2。L1 は復旧の案内が feature org の leader にとって正しいかどうかに関わる。L2 は deadman の数え方の決めで、コードの変更が小さくても doc と tech-debt に記録が要る。L5・L6 は register の文字の直しだけ
  - L3、L4、L8、L9 はテストと呼び出しの形の整理で、動作を変えない。L4 はテスト 6 件の期待文も動く。L7 と L10 は任意
  - パイプラインは cycle 1 で、LOW を今直すと `/verify` 以降をもう一度走らせる。cycle 2 が上限なので、直すものを選ぶなら L1 と L2 だけにして、残りは tech-debt の 1 行に束ねる案もある
  - 範囲外として扱ったこと(新しい指摘ではない): `Org.Getenv` の doc(`spawn.go:234`)が「Stop と Disband が読む」のままで、`compensateStale` が新しい読み手になった(`approve.go:123`、`director_stop.go:258` も前から読み手)。新しい binary の `compensateStale` は、記録に `spawn_step` が残る spawn を、ロックを取らない古い binary が実行中でも stale と見て閉じる(古い binary の spawn の途中で更新すると起きる。計画の D5 は新しい binary 同士だけを守る)。仕様の `[x]` の数と evidence の数字(`/verify`)、`docs/plans/active/` を指す文字(evidence の 4 行目など。`/pr` の archive で tech-debt の行は直るが evidence は直らない)

## 直したもの(cycle 1、レビューのあと)

L1〜L10 と範囲外の注記について、直したか残したかを書く。コードとテストの直しはコミット dde26a8f と 70b23052、記録の直しはこの節を足したコミットと、それを直したコミットにある。

- L1: 直した(dde26a8f)。復旧の案内 4 か所の `ralph org stop --org-id <o> --seat <s>` を `stopSeatCommand` 1 つにまとめた。`Org.InboxHintStateDir` が入っていれば、`CommandWithStateDir` が `--state-dir <台帳>` を引用符つきで末尾に足す。CLI は `newOrgSpawnRuntimeAt`(`internal/cli/org.go`)で `orgInboxHintStateDir` の値を入れる。`spawn`、`start`、`start --plan` はこの関数で runtime を作るので、どれも `--state-dir` を明示したときだけ案内に付く。`InboxHintStateDir` の doc も、Spawn の案内が同じ値を繰り返すと書き足した。テストは `TestOrgSpawn_StopHints_RepeatInboxHintStateDir`(4 つの道のそれぞれで、未設定、設定、引用符の要る値の 3 通り)と、`internal/cli/org_test.go` の `TestOrgSpawn_InFlightSeatRefusal_StopHintRepeatsStateDir`(`--state-dir` では付き、`RALPH_ORG_STATE_DIR` では付かない)。`stopSeatCommand` が値を無視する変異と、CLI の代入を消す変異は、どちらも新しいテストで赤になった。`StartFeature` の事前検査(`spawnPrecheckErr`)には 70b23052 で引数 `hintStateDir` を足し、`internal/org/feature.go` が `o.InboxHintStateDir` を渡す。`TestSpawnPrecheckErr_InFlightSeatRefusal_MatchesSpawn` は、未設定と設定の 2 通りで事前検査と Spawn の文が一致することを見る。`internal/org/feature_test.go` の `TestStartFeature_PrecheckStopHintRepeatsInboxHintStateDir` は、`StartFeature` から値が渡ることを見る。`feature.go` の受け渡しを消す変異と、`spawnPrecheckErr` の中で値を捨てる変異は、どちらも赤になった
- L2: 動作は変えず、`leaderActivityEventCount` の doc に数えるものと数えないものを書いた(dde26a8f)。spawn が書く event で数えるのは `spawn_started` だけで、`spawn_failed` と `spawn_close_failed` は数えない。途中の座席への検査拒否、ロックによる断り、`compensateStale` の 2 つの断り(自分の pane、確かめも閉じもできない pane)は `spawn_started` も `rejected` も書かないので数えない。断られる spawn を打ち直すだけの leader は止まっているので、deadman が人へ escalation してよいと決めた
- L3: 残した。`failStepWithNote` の引数を変えると呼び出しのすべてに手が入り、動作は変わらない。後の整理に回す
- L4: 語は残した。確かめられない道の `ctrl_c=skipped: pane not confirmed pane=not closed: <err>` は、同じ状況で `stopSeatPane`(`internal/org/verbs.go`)が `stop_failed` に書く語と同じで、変えると `spawn_close_failed` と `stop_failed` の語がずれる。dde26a8f で一度 `ctrl_c=skipped pane=not confirmed, not closed: <err>` に変えたが、70b23052 で base の語とテストの期待文に戻した。`compensateStale` の doc の一覧にはこの形を足し、`stopSeatPane` と同じ語だと書いた(dde26a8f、70b23052)。evidence(`docs/evidence/seat-spawn-close-2026-10-22.md`)と skill の 4 写しは、どちらの語も引いていない
- L5: 直した(この節のコミット)。org-stop-all の行の Why deferred の欄で、(b) を挙げた文を取り消し線にした
- L6: 直した(この節のコミット)。ALERT の行の Trigger に、spawn の join と HELLO(試行の失敗した spawn で 17〜81 s)も同じ測り方で見ることを足した。F4 の行の「agmsg がなぜ遅かったかは分からない」という文から、ALERT の行を書き出しの語で指した
- L7: ファイルは分けず、`spawn.go` と `spawn_test.go` の大きさの行を tech-debt に足した(この節のコミット)。分けると 1000 行ほどが動くが、確かめる中身は変わらない
- L8: 直した(dde26a8f)。3 本(5 ケース)の `TimeoutMS` を 50 から 500 にし、期限の案内の期待文も 500 ms にした。どの fake(`joinBlocks`、`sendBlock`、1 時間待つ `agent start` の再試行)も spawn の期限が切れるまで待つ。期限を伸ばしても、期限が切れる step は変わらず、前の step の余裕が増えるだけになる。直しを戻すと赤になることは、コードから読んで変わらない。補償が spawn の ctx を使えば `assertLiveBoundedCtxs` が切れた ctx を見て落ち、pane を閉じなければ herdr の呼び出しの列が合わず、案内がなければ文の検査が落ちる
- L9: 直した(dde26a8f)。`appendLegacyRejectedRetry` を `checkRetryRefusedThenAppendLegacyRejected` に改めた。動作は変えていない
- L10: 残した。10 秒の上限は `defaultDriverCallTimeout` の 1 定数で、文の写しごとに固定するテストを足しても、手間に見合うものが少ない
- 範囲外の注記
  - `Org.Getenv` の doc に Spawn を足した(dde26a8f)。`compensateStale` が自分の pane を見分けるために `HERDR_PANE_ID` を読む
  - 古い binary の spawn がロックを取らない件は、tech-debt の stop とロックの行に 1 文足した(この節のコミット)。コードから読んだもので、再現はしていない
  - evidence の `docs/plans/active/` を指す文字は残した。`archive-plan.sh` が書き換えるのは `docs/tech-debt/README.md` だけで、#248 の evidence と同じ扱いにした
