# Self-review report: org-stop-all

- Date: 2026-10-07
- Plan: docs/plans/active/2026-10-07-org-stop-all.md
- Branch: feat/org-stop-all(HEAD e2189d79)
- Reviewer: reviewer subagent (Claude)、cycle 1
- Scope: diff の品質だけ(命名、読みやすさ、不要な変更、typo、null 安全、デバッグ用コード、秘密情報、例外処理、安全性、保守性)。対象は `git diff origin/main...HEAD`(26 ファイル、+4513/-159)。このコードは herdr の pane と workspace を閉じ、台帳に `stopped` / `disbanded` を書くので、(1) 台帳に記録していない pane や workspace を閉じうる経路、(2) 自分の pane と workspace を後回しにする順序、(3) 期限が効かず固まりうる呼び出し、(4) プロセスが動いているのに `stopped` / `disbanded` を書く経路、を重点に読んだ。仕様への適合、テストの網羅、文書のずれは見ていない(`/verify`、`/test`、`/sync-docs` の担当)。テスト、linter、型検査は実行していない。

## Evidence reviewed

- `internal/org/verbs.go` の `Stop`、`stopSeatPane`、`CloseDeferredSelfPane`、`CloseDeferredSelfWorkspace`、`Disband`、`disbandOwnLast`、`closeOrgWorkspace`、`recordWorkspaceClosed` を全行読んだ。`Stop` の呼び出し順(C-c → close → Leave → 観測 → 台帳)と、close 失敗時に Leave を飛ばして `stop_failed` を書く分岐、`--force` の分岐、自分の pane を `selfPane` で除外する分岐を、台帳に書く値と照らして追った。
- `internal/org/verbs_all.go` を全行読んだ(`StopAll`、`DisbandAll`、`orgsToDisband`、`splitOwnOrgs`、`holdsCaller`、`leaveOwnOrg`)。自分の pane・workspace を持つ org を最後に回す条件が、`Disband` の中の同じ判定と一致していることを確かめた。
- `internal/org/driver/herdr.go` の `PaneClose`、`WorkspaceClose`、`checkHerdrCloseResult`、`findHerdrErrorEnvelope`、`IsNotFound`、`NotFound` を読んだ。not-found になるのは、JSON の envelope として読めて code が `pane_not_found` / `workspace_not_found` のときだけで、本文に文字列が含まれるだけの場合、壊れた JSON、接続エラー、期限切れはどれも失敗として返ることを、`herdr_test.go` の表と照らして確かめた。
- `internal/org/driver/driver.go` の `ExecRunner.Run` に入った `cmd.WaitDelay` を読み、`go doc os/exec ErrWaitDelay` で成功時の挙動を確かめた(F-2)。
- `internal/org/spawn.go` の `resolveWorkspace` と `openOrgWorkspaces` を読んだ。dry-run のイベントを除く条件は、`org_workspace_created` を書く箇所(`resolveWorkspace` の 1 か所、`DryRun` なし)と矛盾しない。
- `internal/cli/org.go` の `stop` / `disband` の `RunE`、`rejectFlagsWithAll`、`withPrefixOnce`、`printSeatFailure`、`printOtherErrs`、`printStopAllResult`、`closeDeferredSelf`、`printDisbandResult`、`printDisbandAllResult` を読んだ。`closeDeferredSelf` は出力をすべて済ませてから最後に呼ばれ、`--all` の `stop` は workspace を渡さず pane だけを渡すことを確かめた。`root.go` が `SilenceUsage` / `SilenceErrors` を立てているので、終了時のエラー表示は二重にならない。
- `Stop` と `Disband` を呼ぶ箇所を `grep` した。非テストの呼び出し元は `internal/cli/org.go` と `verbs_all.go` だけで、watchdog など自動で呼ぶ経路はない。`stop_failed` と `org_workspace_closed` は `stateEvents` にないので `Roster` を動かさず、`watch.go` の `leaderActivityEventCount` の対象にも入らない。
- herdr の実機の挙動は、`docs/evidence/herdr-pane-close-2026-10-07.md` と、手元の `herdr --help` / `herdr pane close --help` / `herdr workspace --help`、`herdr` バイナリの文字列で確かめた。herdr のサーバーには触れていない(F-1 の根拠)。
- テストの新規・変更ファイルは、デバッグ出力、`TODO` / `FIXME`、`/Users/` などの固定パス、実時間の `time.Sleep` を `grep` した。ヒットは `fakeHerdr` の遅延の注入だけで、本体の追加行にはない。HERDR_PANE_ID と HERDR_WORKSPACE_ID は `setupOrgStubPATH` と `Org.Getenv` で固定されていて、herdr の中で `go test` を回しても結果が変わらない。
- ミラー: `.claude/skills/org/SKILL.md`、`.agents/skills/org/SKILL.md`、`templates/base/` の 2 面の差分は、追加・削除の行の md5 が 4 面とも同じ。追加行に `internal/`、`cmd/`、`issue #`、`PR #` はない。`docs/evidence/` に `$HOME` の絶対パスはない。
- `docs/tech-debt/README.md` は差分に含まれていない。この diff が新しく作った負債の行はない(下の「Tech debt identified」)。

## Findings

<!-- Area recommended values: naming, readability, unnecessary-change, typo,
     null-safety, debug-code, secrets, exception-handling, security, maintainability -->

| Severity | Area | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- |
| F-1 MEDIUM | security | 台帳に記録した id を、それが今も ralph の作ったものを指しているか確かめずに閉じる。herdr の id は `w1` や `w1:p2` のような短い通し番号で、herdr は名前付きセッションを持ち(`herdr session`、バイナリに `HERDR_SESSION`)、セッションを復元する処理もある。実機の記録は、pane 1 つと workspace 1 つを閉じただけで、閉じたあとに別の workspace を作って id が再利用されるか、サーバーを再起動して id が振り直されるかは確かめていない(未確認。再利用されないと分かれば、この指摘は閉じる)。再利用されるなら、次の 3 経路で ralph が作っていない workspace や pane を閉じる。(a) `disband --all` と、その org の `disband` は、古い ralph の disband が workspace を閉じずに残した org も対象にする(`verbs_all.go` の `orgsToDisband` のコメントに明記)。古い ralph の利用者は、skill の「herdr workspace に残留がないか確認」に従って workspace を手で閉じてきたはずで、その記録の id を最初の `disband --all` が一斉に `workspace close` する。(b) `Stop` は `seat.Active` を見ないので、`stop --seat X` を `stopped` 済みの座席に打つと、記録された古い pane id に C-c を送り、閉じる。これまでは C-c だけだった。(c) 起動していた herdr のセッションと別のセッション(`HERDR_SESSION` が違う shell)から打つと、同じ id が別の pane を指す。AC8 のテストは「記録していない id に閉じる呼び出しが行かない」ことしか見ていないので、記録が古い場合は防げない | `internal/org/verbs.go:744`(`seatFromEvents` は `Active` を見ない)、`:874`(`PaneClose`)、`:1448`(`WorkspaceClose`)。`internal/org/verbs_all.go:294-296`、`:320`(古い disband が残した workspace も対象にする条件)。`internal/org/spawn.go:1697`(`WorkspaceCreate(ctx, p.Cwd, p.OrgID)`、label は org_id)。`docs/evidence/herdr-pane-close-2026-10-07.md` の Run 2・3(閉じたあとの再作成なし) | まず隔離した herdr サーバーで、workspace を作って閉じ、もう 1 つ作って id が繰り返されるか、サーバーを再起動して復元したあとの id がどうなるかを確かめ、結果を evidence に足す。繰り返されるなら、閉じる前に `herdr workspace get <id>` で label が org_id と一致することを確かめ、一致しなければ閉じずに「もう ralph のものではない」と報告する(workspace の label は spawn が org_id で付けている)。あわせて、`orgsToDisband` の「古い disband が残した workspace」の枝は `--org-id` を明示した disband に限る(`--all` では対象にしない)か、`--all` の help に書く。`stop` は `Active` でない座席の pane を閉じない(C-c だけ、または何もしない)ようにする案もある |
| F-2 LOW | exception-handling | `ExecRunner.Run` に入れた `cmd.WaitDelay` は、期限切れのときだけでなく、成功した(exit 0 の)コマンドにも効く。プロセスは終わったが孫プロセスが stdout / stderr を握ったままのとき、これまでは孫が終わるまで `Run` が戻らなかった。今は 2 秒後に pipe を閉じて `exec.ErrWaitDelay` を返す。`Run` はこれを失敗として返すので、成功したコマンドが失敗に見え、`out` も途中で切れる。herdr と agmsg の shell script の全呼び出し(spawn、send、wait を含む)に効く変更だが、コメントとテストは期限切れの側しか扱っていない。孫を残す呼び出しが今あるかは未確認(あれば、これまで固まっていたはずなので、実害は 2 秒以上で終わる孫だけ) | `internal/org/driver/driver.go:51`。追加テスト `TestExecRunner_Run_TimeoutNotHeldByGrandchild` は期限切れのみ。`go doc os/exec ErrWaitDelay`: 「プロセスが成功して終わったが、pipe が WaitDelay 内に閉じられなかったとき」に返る | `cmd.Run()` の直後で `errors.Is(err, exec.ErrWaitDelay) && ctx.Err() == nil` のときは、プロセスが成功しているので `err = nil` として扱う(出力は読めた分を返す)。コメントに成功側の扱いを 1 文足し、孫が 2 秒を超えて pipe を握る成功ケースのテストを 1 つ足す |
| F-3 LOW | maintainability | `--all` の全体に上限がない。1 回の呼び出しは 10 秒で切れるが、herdr が接続を受けて返事をしない状態では、1 座席が C-c と close で約 20 秒かかり(close が失敗すれば Leave は飛ばす)、座席ごとに繰り返す。座席の上限 30(仕様の FR-3)なら約 10 分、codex の座席は観測の最大 8 秒が加わる。AC11 の「1 回の呼び出しは期限で切れる」は満たしているが、「herdr が止まっている」ときの `--all` が分単位で無言になる。接続拒否なら即座に失敗するので、返事をしない場合だけの問題。出力は最後にまとめて出す作りで、進行が見えない | `internal/org/verbs.go:603`(`defaultDriverCallTimeout = 10s`)、`:858-883`(`stopSeatPane`)。`internal/org/verbs_all.go:113-145`(座席ごとの直列ループ)。`internal/cli/org.go` の `printStopAllResult` は最後に出力 | 受け入れてよいなら、`stop --all` の help に「herdr が応答しないと 1 座席あたり約 20 秒かかる」と 1 行足す。直すなら、連続して期限切れになった回数が一定を超えたら残りの座席を `stop_failed` にせず「herdr 応答なし」で打ち切る、または座席ごとの結果をその場で stderr に出す |
| F-4 LOW | exception-handling | `Disband` は台帳を 1 回読んだあとで座席を止め、最後に `disbanded` を書く。`disbanded` は、それより前に書かれたその org のすべての座席を非 active にする(`Roster`)。読んだあと、`disbanded` を書くまでの間に別プロセスの `spawn` が座席を足すと、その座席は止められないまま非 active になり、プロセスは動き続ける。この diff より前からある経路だが、`Stop` が pane の close と期限(座席あたり最大約 20 秒)を含むようになり、窓が広がった | `internal/org/verbs.go:1370`(1 回の `Manifest.Read`)、`:1407`(`appendDisbanded`)。`internal/org/manifest.go:184`(`disbanded` が前の座席を非 active にする) | `disbanded` を書く直前に台帳を読み直し、最初の読みになかった active な座席があれば `disbanded` を書かずに失敗として返す。直さないなら、台帳に 1 行足す(spawn と disband の競合) |
| F-5 LOW | maintainability | `withPrefixOnce` は、エラー文字列が `"org: stop: "` で始まるかを見て接頭辞を足す。原因は `Stop` の error の包み方がそろっていないことで、台帳への追記失敗だけが接頭辞なしで返る(`result.Err = err`)のに対し、`stop_failed` の追記失敗は `org: stop: record stop_failed: %w` で包まれている。CLI で文字列の先頭を見て補うと、`Stop` の文言を変えたときに二重表示か接頭辞なしに戻る | `internal/cli/org.go:797-802`、`internal/org/verbs.go:845`(`result.Err = err`)と `:841`(`org: stop: record %s: %w`) | `Stop` の `result.Err = err` を `fmt.Errorf("org: stop: record %s: %w", event, err)` にそろえ、`withPrefixOnce` をやめて `fmt.Errorf("%w")` 1 つか、そのまま返す形に戻す |
| F-6 LOW | maintainability | (a) `Stop` は 734〜850 行の約 117 行(コメントと空行を除くと 81 行)で、この diff で約 31 行増えた(元は 86 行)。agmsg の Leave を `stopSeatPane` と同じ形の関数に出せる。(b) 「自分の pane / workspace か」の判定が、`Stop`(`selfPane`)、`Disband`(座席の loop と workspace の loop)、`StopAll`、`holdsCaller` の 4 か所に別々に書かれている。1 か所だけ直すと、自分の pane を後回しにする順序が静かに壊れ、呼び出した側が途中で閉じる。今は 4 か所が一致している。(c) `DisbandResult.recordStop` / `failSeat` と `StopAllResult.recordStop` / `failSeat` が同じ分岐の写し | `internal/org/verbs.go:734-850`、`:758`、`:1382`、`:1394`。`internal/org/verbs_all.go:131`、`:344-353`。`internal/org/verbs.go:1279-1296` と `internal/org/verbs_all.go:73-93` | (b) は `o.ownPaneID()` / `o.ownWorkspaceID()` のような読み取りと比較の 1 組の関数にまとめる。(a)(c) は急がない |
| F-7 LOW | maintainability | `NewHerdrError` は、別パッケージのテスト(`internal/org` の `fakeHerdr`)が「閉じ済み」の応答を作るためだけに、本番の API として export されている。コメントに理由は書いてあるが、利用者向けの面に test 専用の生成子が残る | `internal/org/driver/herdr.go:61` | 急がない。直すなら、`fakeHerdr` が返すエラーを `NotFound() bool` を持つ小さな型にして、`driver` の export を減らす |
| F-8 LOW | readability | `/org` skill の表の `stop` と `disband` の行が、1 つのセルに 8〜10 の挙動(順序、失敗時、`--all`、`--force`、`--dry-run`、自分の pane)を詰めていて、表のセルとして読みにくい(4 面の写しで同じ) | `.claude/skills/org/SKILL.md` の `stop` と `disband` の行(167・168 行目) | 表のセルは 1〜2 文にして、詳細は表の下の節に出す。4 面の写しは `scripts/sync-skills.sh` で同期する |

## Positive notes

- 順序の不変条件が実装で守られている。自分の pane の座席は `Stop` が C-c も close もせず、`stopped` を書いて pane id を返すだけで、閉じるのは CLI の最後(`closeDeferredSelf`、出力のあと)になる。自分の workspace は `org_workspace_closed` を先に書いてから最後に閉じる。ほかの座席か workspace が 1 つでも失敗したときは、自分の座席と workspace に手を付けず、失敗として一覧に出すので、セッションが生きたまま失敗を読める。`--all` でも、自分を持つ org を最後に回す条件は `Disband` の中の判定と一致している。
- `stopped` を書く経路が絞られている。pane が閉じた、herdr が not-found と返した、pane_id の記録がない、自分の pane(閉じる前に記録するが close の失敗は呼び出し側が見せる)、`--force` の 5 つで、`--force` は Details に `close failed (forced)` を残し、stderr に警告を出し、`DisbandResult` では `Forced` の失敗として別に数える。`stop_failed` は状態のイベントではなく、座席が active のまま残るので、打ち直しで拾える。
- not-found の判定が閉じている。`IsNotFound` は、envelope として読めた error の code だけを見る。本文に `pane_not_found` を含むだけの文字列、壊れた JSON、接続拒否、期限切れは失敗のまま返り、`errors.Is(err, context.DeadlineExceeded)` も保たれる(`herdr_test.go` の表に 9 ケース)。
- 期限が 1 回の呼び出しごとに新しい context で効き、`WaitDelay` で孫が pipe を握る場合の固まりも塞いでいる(F-2 の成功側を除く)。台帳の flock には既に 5 秒の取得期限がある。
- 失敗した座席が「stopped」と表示されなくなった。`DisbandResult` は止められた座席と止められなかった座席(理由つき)を分け、CLI は `disbanded org` を `Disbanded` のときだけ出す。以前は失敗した座席も `StoppedSeats` に入っていた。
- `--all` と `--org-id` / `--seat` の同時指定は、状態ディレクトリの解決や台帳の読み込みより前に断る。`--org-id ""` も `Changed` で拾う。
- ミラー 4 面の差分が一致し、追加の文書に `internal/` や `PR #` の参照がない。evidence の記録に `$HOME` の絶対パスはなく、隔離したサーバーで取ったと明記している。

## Tech debt identified

| Debt item | Impact | Why deferred | Trigger to pay down | Related plan/report |
| --- | --- | --- | --- | --- |
| (この diff は台帳に行を足していない。F-3、F-4、F-6、F-7 を直さずに進める場合は、1 行にまとめて足す) | | | | |

_(If any rows were added above, also append them to `docs/tech-debt/`.)_

## Recommendation

- Merge: 可。CRITICAL と HIGH はない。F-1(MEDIUM)は herdr の id が再利用されるかが未確認で、再利用されないと確かめられれば閉じる。再利用されるなら、閉じる前の確認か `--all` の対象の絞り込みを入れてから merge することを勧める。確かめる手順は隔離したサーバーで 5 分ほどで、結果を evidence に足せば AC8 の根拠にもなる。
- Follow-ups: F-2 は 3 行とテスト 1 つで直り、全 driver 呼び出しに効く変更なので PR の前に直すことを勧める。F-3〜F-8 は blocking ではない。直すのが `internal/` の変更なら `/self-review` から回し直しになる。直さないなら、F-3、F-4、F-6 を台帳に 1 行で足す。
