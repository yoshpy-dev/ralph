# Verify report: org-stop-all

- Date: 2026-10-08
- Plan: docs/plans/active/2026-10-07-org-stop-all.md
- Verifier: verifier subagent (Claude Opus 5.5)、pipeline cycle 2(`cycle-count.json` は 2、既定の上限の最後の回)
- Scope: 仕様への適合(AC1〜AC16。AC16 は今回の新規で、S8 のあとも AC10・AC12・AC14 が成り立つかを確かめ直した)、静的解析、文書のずれ。対象は `git diff f423f230...HEAD`(HEAD 8bfc0f40、35 ファイル、+7094/-165)。cycle 1 の verify(de154798)以降の差分は 17 ファイル(+1400/-138)で、テスト以外のコードの変更は `internal/org/verbs.go` と `internal/cli/org.go` だけ(53807a82 の S8 と、223258eb のヘルプ文)。テスト(`./scripts/run-test.sh`、`go test`)は /test の担当なので実行していない。HEAD からビルドしたバイナリで、herdr と agmsg を呼ばない観察(`status` と `--dry-run`)を行った
- Evidence: `docs/evidence/verify-2026-10-07-org-stop-all.log` の末尾に cycle 2 の節を足した(`docs/evidence/*.log` は gitignore の対象なので手元にだけ残る)。runner 自身のログは `docs/evidence/verify-2026-10-07-162302.log`
- 構成: cycle 1 の本文は、`docs/tech-debt/README.md` の 164 行(V-2、Coverage gaps)と 167 行(Coverage gaps)が指しているので、末尾の付録に残した。見出しを 1 段下げたほかは原文のままで、行番号は 801ca77f のもの

## Spec compliance

plan の承認: `./scripts/plan-visual.sh digest` は `2f2cfde39e9f` を返し、plan の `- Approved: 2026-10-08 sha256:2f2cfde39e9f` と一致した。AC1〜AC16 のチェックボックスは付いており、Verify plan の行(`:113`)も AC1〜AC16 になっている。

cycle 1 から変わったコード: `git diff de154798 HEAD` の hunk を見ると、`Stop`、`stopSeatPane`、`Disband`、`closeOrgWorkspace`、`StopAll`、`DisbandAll`、`orgsToDisband` の本体は変わっていない(`Stop` と `Disband` は doc comment だけ)。変わったのは、後回しにした自分の pane と workspace を閉じる処理(`internal/org/verbs.go:984-1248`)、CLI の `closeDeferredSelf`(`internal/cli/org.go:890`)、`stop` と `disband` の Long。このため AC1〜AC9、AC11、AC13、AC15 は cycle 1 の判断(付録)をそのまま使い、下の表には今の行番号だけを書いた。テストは名前で引ける(f63ae025 と S8 がテストを足したので、付録の `verbs_test.go` の行番号はずれている)。

| Acceptance criterion | Status | Evidence |
| --- | --- | --- |
| AC1: stop は C-c のあと pane を閉じ、見つからなければ閉じ済み | Met(実装は cycle 1 から不変) | `stopSeatPane`(`internal/org/verbs.go:950`) |
| AC2: 閉じられなければ `stop_failed`、終了コード 1、打ち直しで `stopped` | Met(不変) | `Stop`(`verbs.go:832`、`stop_failed` は `:919-933`)。`stop_failed` は `stateEvents`(`internal/org/seat.go:52`)にない |
| AC3: disband は座席を止めてから workspace を閉じ、失敗があれば `disbanded` なし | Met(不変。V2-5 を参照) | `Disband`(`verbs.go:1694`)、`closeOrgWorkspace`(`:1780`)、`printDisbandResult`(`internal/cli/org.go:1073`) |
| AC4: `stop --all` | Met(不変) | `StopAll`(`internal/org/verbs_all.go:116`)、`printStopAllResult`(`org.go:853`) |
| AC5: `disband --all`、失敗した org は次も対象 | Met(不変) | `DisbandAll`(`verbs_all.go:264`)、`orgsToDisband`(`:307`) |
| AC6: 同時指定はエラー | Met(不変) | `rejectFlagsWithAll`(`org.go:793`) |
| AC7: `--all --dry-run` は driver を呼ばない | Met(不変) | probe P5〜P7 でも、dry-run の `stop --all` と `disband --all` が herdr なしで rc 0 で終わった |
| AC8: 記録していない pane と workspace は閉じない | Met | 後回しの close も、台帳から座席と org を引き直してから閉じる(`lastSeatOnPane` `verbs.go:1213`、`lastWorkspaceEvent` `:1239`)。記録がなければ閉じずに失敗(`:1020-1021`、`:1089-1090`) |
| AC9: skill・雛形・README・FR-2、2 つの sync check | Met(S8 の補償は skill に未記載、V2-1) | 4 面の写しは `cmp` で同一。`check-skill-sync.sh` と `check-sync.sh` は通った。cycle 1 の D-1〜D-3 は 223258eb と 2c97407c で直っている |
| AC10: `--force` は失敗を警告にして終了コード 0 | Met(S8 で後回しの close の経路も直った) | 下の「AC10・AC12・AC14 の確かめ直し」 |
| AC11: 応答しない呼び出しは期限で打ち切る | Met | 後回しの close の get と close も `callWithTimeout`(`verbs.go:631`)を通る(`:1046`、`:1119`。get は `confirmSeatPane` / `confirmOrgWorkspace` の中) |
| AC12: 自分の pane と workspace の close は最後 | Met | 下の「AC10・AC12・AC14 の確かめ直し」 |
| AC13: disband のあとの spawn は新しい workspace | Met(補償で開き直した workspace には例外がある。V2-1) | `resolveWorkspace`(`internal/org/spawn.go:1705`)は不変 |
| AC14: label が違う pane と workspace は閉じない、後回しの close も同じ確認 | Met | 下の「AC10・AC12・AC14 の確かめ直し」 |
| AC15: `ErrWaitDelay` を失敗にしない | Met(不変) | `internal/org/driver/driver.go:60` |
| AC16: 後回しの close が失敗したら台帳を戻し、打ち直しで閉じ直す。`--force` は補償なし・警告・終了コード 0。`--force` で workspace を閉じると中の pane も終わることを help と skill に書く | Met(テストの実行は /test) | 下の「AC16」 |

### AC16

| AC16 の要件 | 実装 | テスト |
| --- | --- | --- |
| 自分の pane の close が失敗したら、座席を active に戻す | `CloseDeferredSelfPane`(`verbs.go:1010`)は、`closeSelfPane`(`:1039`。確認のあとに close)が失敗を返すと `reactivateSeat`(`:1175`)を呼ぶ。`reactivateSeat` は、座席の最後の状態が `stopped` で pane が同じときだけ(`:1177`)、直前の `stopped` の項目と元の `spawned` の `HerdrAgentName` を写した `spawned`(`reactivated: …`)を書く | `TestOrgCloseDeferredSelfPane_CloseFails_SeatReactivated`(`internal/org/verbs_test.go:2689`。`assertReactivatedCopy` が欄ごとに比べる)、`TestOrgCloseDeferredSelfPane_Outcomes`(`:2166`) |
| 自分の workspace の close が失敗したら、workspace を開き、座席を active に戻す | `CloseDeferredSelfWorkspace`(`:1079`)は `reopenWorkspace`(`:1198`。最後の記録が `org_workspace_closed` のときだけ)で `org_workspace_created`(`reopened: …`)を書き、`HERDR_PANE_ID` が同じ org の座席の pane なら `reactivateSeat` も呼ぶ(`:1103-1107`) | `TestOrgCloseDeferredSelfWorkspace_CloseFails_ReopenedAndReactivated`(`:2831`。pane を閉じた seat-2 は戻さないことも見る)、`TestOrgCloseDeferredSelfWorkspace_Outcomes`(`:745`) |
| `ralph org status` でも active | `Roster` は、`disbanded` より後に入った座席の状態イベントを active と読む(`internal/org/manifest.go:203`)。補償の `spawned` は `disbanded` の後に入る | CLI の `TestOrgStopDisband_OwnCloseFails_LedgerRestoredForRetry`(`internal/cli/org_stop_all_test.go:721`)が `status` の `spawned (active)` を見る。probe P6・P7 |
| 終了コード 1 | `closeDeferredSelf`(`org.go:890`)が `errors.Join(runErr, closeErr)` を返す(`:907`) | `org_stop_all_test.go:721` |
| 同じ `stop` / `disband` の打ち直しで閉じ直す | 座席が active に戻るので、同じ pane で打った `stop` は自分の pane としてまた後回しにする。workspace は `openOrgWorkspaces`(`spawn.go:1729`)に戻るので、`disband` もまた後回しにする | `TestOrgCloseDeferredSelfPane_CloseFails_RetryClosesIt` の "stop again in the same pane"(`verbs_test.go:2718`)、`TestOrgCloseDeferredSelfWorkspace_CloseFails_RetryClosesIt` の "disband again in the same pane"(`:2876`) |
| ほかの pane からの `stop --all` / `disband --all` で閉じ直す | `stop --all` は active な座席として、確認、C-c、close を通す。`orgsToDisband`(`verbs_all.go:307`)は、`disbanded` の後の `spawned` か `org_workspace_created` があれば org を対象に戻す | `:2718` の残り 2 例(`stop --all`、`disband`)、`TestOrgCloseDeferredSelfPane_AfterDisband_ReactivatedPastDisbanded`(`:2786`、pane だけを後回しにした disband)、`:2876` の "disband --all from another pane"、CLI の `:721` の 2 例。probe P6・P7 |
| `--force` は補償を書かず、警告を出し、終了コード 0 | 補償の前に返る(`verbs.go:1028-1030`、`:1097-1099`)。`closeDeferredSelf` は force のとき `warning:` を出して `runErr` を返す(`org.go:903-905`) | `TestOrgCloseDeferredSelf_Force_NoCompensation`(`verbs_test.go:2942`)、CLI の `TestOrgStopDisband_Force_OwnCloseFails_WarnsExitsZero`(`org_stop_all_test.go:794`) |
| 補償も書けないときは、手で閉じる herdr のコマンドを出す | `selfCompensation.errorFor`(`verbs.go:1151`)が、追記の失敗のあとに `close it by hand: herdr …` を付ける。台帳が読めないとき、記録がないとき、`--force` のときも付く(`:1017`、`:1021`、`:1029`、`:1086`、`:1090`、`:1098`) | `TestOrgCloseDeferredSelf_NothingRestored_NamesTheManualClose`(`:2983`、4 例) |
| `disband --help` と `/org` skill に、`--force` でも確かめた workspace は閉じ、中の pane も終わると書く | `org.go:1025-1027`(probe P8 で表示も見た)、`.claude/skills/org/SKILL.md:168`(4 面で同じ) | 文書なのでテストはない |

### AC10・AC12・AC14 の確かめ直し

- AC10: cycle 1 はこれを Met としたが、自分の pane か workspace を最後に閉じる close が `--force` で失敗した場合を見ていなかった。そのときは終了コード 1 になっていた(cross-review の #3)。S8 で `closeDeferredSelf` に `force` の分岐が入り(`org.go:903-905`)、今は警告を出して終了コード 0 で終わる。cycle 1 の「Met」はこの経路を含んでいなかった、とここに書き残す(付録の本文は書き換えていない)。`stop --all --force` も同じ関数を通る(`org.go:756`)が、テストが持つのは `stop --seat` と `disband --org-id` の 2 例だけ
- AC12: 成功の経路では、`CloseDeferredSelfPane` / `CloseDeferredSelfWorkspace` は台帳を読むだけで何も書かずに close する(`verbs.go:1015`、`:1084`)。このため close は最後の操作のまま。補償を書くのは close のあとだが、close が失敗してプロセスが生きているときに限られ、AC16 がそれを求めている。順序のテスト `TestOrgStopDisband_OwnPaneOrWorkspace_ClosedLastAfterOutput`(`org_stop_all_test.go:629`)は S8 で変わっていない
- AC14: `closeSelfPane` と `closeSelfWorkspace` は、close の前に `confirmSeatPane` / `confirmOrgWorkspace` を通す(`verbs.go:1040`、`:1113`)。確認に落ちたときは close せず、補償だけを書く(pane と workspace は実際に開いたままなので、補償のあとの台帳は事実と合う)。`TestOrgCloseDeferredSelf_RecheckedBeforeTheClose`(`verbs_test.go:2566`)は、S8 で close が 0 回であることに加えて補償の記録も見るようになった。条件を弱めた変更はない(S8 のテストの削除行は、関数に `force` 引数を足した呼び出しの書き換えと、表の行の書き換えだけ)

## Static analysis

| Command | Result | Notes |
| --- | --- | --- |
| `./scripts/run-static-verify.sh` | PASS(rc 0) | Mode static、scope changed。`verify.local.sh`(shellcheck、hook の `sh -n` 20 本、settings の `jq -e` 2 本、Codex の hook guard 3 本、`check-sync.sh`、`check-pipeline-sync.sh`、`check-skill-sync.sh`、`check-template-purity.sh`、tech-debt README の plan の参照)、golang verifier(`gofmt: ok`、`go vet ./...` は出力なしで成功、golangci-lint `0 issues.`、staticcheck は出力なしで成功)、branch secret scan(`f423f230..8bfc0f40` clean) |
| `./scripts/check-sync.sh`(上の中) | PASS | IDENTICAL 164、DRIFTED 0、ROOT_ONLY 0、TEMPLATE_ONLY 11、KNOWN_DIFF 5 |
| `./scripts/check-skill-sync.sh`(上の中) | PASS | 13 skill が一致 |
| `git diff --check f423f230...HEAD` | PASS | 出力なし |
| 追加行の U+FFFD 検索 | PASS | 0 件 |
| ミラーの `cmp` | PASS | `/org` skill の 4 面、recipe の 2 面 |
| `./scripts/plan-visual.sh digest` | PASS | `2f2cfde39e9f`、plan の Approved 行と一致 |
| `go build ./cmd/ralph`(probe 用) | PASS | go 1.26.0、macOS |

## Documentation drift

plan の Verify plan の grep(`git grep -n 'C-c\|disband\b.*report\|herdr workspace / agmsg' -- .claude/skills/org internal/org/prompts docs/specs README.md`)で出るのは、新しい止め方と締めの順を書いた行だけだった(skill の stop 行、README の org 節、FR-2、leader の雛形の運用規律)。

| Doc / contract | In sync? | Notes |
| --- | --- | --- |
| `ralph org stop --help`(`org.go:724-744`) | Yes | 持ち主の確認、`stop_failed`、`--force`、自分の pane を最後に閉じること、その close が失敗したら座席を active に戻して終了コード 1、`--force` なら警告だけ、と書いてあり、コードと合う |
| `ralph org disband --help`(`org.go:1011-1034`) | 一部ずれ(V2-2) | `--force` でも確かめた workspace を閉じる 1 文(AC16)はある。最後の段落の「座席を active に、workspace を open に戻す」は、pane だけを後回しにした場合(自分の pane が org の workspace の外にある)には workspace を戻さないことを言い落としている。self-review の C2-4 と同じ |
| `/org` skill の 4 面(`.claude/skills/org/SKILL.md:167-168` ほか) | 一部ずれ(V2-1) | 4 面は同一。`stop` と `disband` の行は、自分の pane と workspace を最後に閉じることまでは書くが、その close が失敗したときの動き(座席を active、workspace を open に戻して終了コード 1、打ち直すか別の pane の `--all` で閉じ直す、`--force` なら戻さず警告で終了コード 0)を書いていない。ヘルプ文にはある |
| leader の雛形(`internal/org/prompts/leader.md:34-39`) | Yes(V2-6 の残余あり) | 手順 8 の「終了コード 1 なら打ち直す」は、S8 で後回しの close の失敗にも効くようになった(同じ `disband` がまた後回しにして閉じ直す。`verbs_test.go:2876` の "disband again in the same pane")。補償の書き込みまで失敗した場合だけは、打ち直しが何も閉じない |
| `README.md` | Yes | Commands の表と org 節は概要だけで、S8 の細部は書く必要がない |
| 仕様 FR-2 | Yes | 概要と、詳細は計画にあると書いている |
| `docs/tech-debt/README.md` の 163〜168 行 | ずれ(V2-3) | 行番号の参照 57 件のうち 27 件が別の行を指す。S8 で増えるはずの行(self-review の「Tech debt identified」の 2 行)もまだない |
| plan の進捗の行(`:156`) | ずれ(V2-4) | 「Verify plan の AC1〜AC13 は書き換えていない」と書くが、今の Verify plan(`:113`)は AC1〜AC16 に書き換わっている |
| cycle 1 の指摘 | 解消 | D-1(ヘルプの持ち主の確認)と D-2(古い ralph の残した workspace)は 223258eb、D-3(plan のチェックボックス)と D-4(tech-debt の行)は 2c97407c、V-1(古い台帳の判定の `--all` の行)と V-3(雛形の締めの順のテスト)は f63ae025 で直った。V-2 は tech-debt の 164 行 (c) に移った |

## Observational checks

HEAD 8bfc0f40 からビルドしたバイナリを、`HERDR_PANE_ID` / `HERDR_WORKSPACE_ID` / `RALPH_ORG_STATE_DIR` を外した環境で、scratchpad の一時ディレクトリの台帳に向けて動かした。herdr と agmsg を呼ぶ経路は通していない(`status` と `--dry-run` だけ)。番号は cycle 1 の P1〜P4 に続ける。

- P5(補償の前): 自分の pane と workspace を最後に閉じる直前の台帳(leader の座席は `stopped`、workspace は `org_workspace_closed`、org は `disbanded`)では、`status` が `stopped`、`stop --all --dry-run` が `no active seats`、`disband --all --dry-run` が `no orgs to disband` を返した(どれも rc 0)。cross-review の #1 が指摘した「打ち直しが何も見つけない」状態はこれ
- P6(workspace の補償のあと): P5 に `org_workspace_created`(`reopened: …`)と `spawned`(`reactivated: …`)を足すと、`status` が `spawned (active)`、`stop --all --dry-run` が `stopped seat org-a/leader`、`disband --all --dry-run` が `disbanded org org-a` を返し、座席と org の両方が対象に戻った
- P7(pane だけの補償のあと): workspace は閉じたまま、`spawned`(`reactivated: …`)だけを足した台帳でも、`status` が `spawned (active)`、`disband --all --dry-run` が `disbanded org org-a` を返した
- P8(ヘルプ): `ralph org disband --help` に、`--force` でも workspace を閉じる文と、最後の close が失敗したときの段落が出る(V2-2 の文言はここで見た)

## Findings

| ID | Severity | Finding | Recommendation |
| --- | --- | --- | --- |
| V2-1 | LOW | `/org` skill の `stop` と `disband` の行(4 面)は、後回しの close が失敗したときの動きを書いていない(上の表)。`disband` の行の最後の「解散した org_id でまた `spawn` すると新しい workspace を作る」には、補償で開き直した workspace があれば、それを使い回すという例外がある(`TestOrgCloseDeferredSelfWorkspace_CloseFails_ReopenedAndReactivated` の末尾)。開き直した workspace は herdr に実際に残っているので、使い回すのは正しい | /sync-docs で、`stop` と `disband` の行に 1 文ずつ足す。表のセルが長い問題(F-8)があるので、表の下の節に書いてもよい。4 面は `scripts/sync-skills.sh` と `templates/base/` への写しでそろえる |
| V2-2 | LOW | `disband --help` の最後の段落は「the command records the seat in the pane active and the workspace open again」と書く。pane だけを後回しにした場合は workspace を戻さない(`CloseDeferredSelfPane` は `reactivateSeat` だけ)。workspace を後回しにした場合も、座席を戻すのは `HERDR_PANE_ID` が同じ org の座席の pane のときだけ(`verbs.go:1103-1107`)。self-review の C2-4 と同じで、直すと `internal/` の変更になる | self-review の「Tech debt identified」の 2 行目のとおり tech-debt に送るか、PR の known gap に書く |
| V2-3 | LOW | tech-debt の 163〜168 行の行番号は、57 件のうち 27 件が HEAD で別の行を指す(どれも `verbs.go`、`org.go`、`verbs_test.go`)。そのうち `verbs_test.go:2215`(164 行の (a))は、行を書いた 2c97407c の時点ですでにずれていた。f63ae025 のテスト追加で 2322 行目に移り、今は 2344 行目にある。ずれの原因を S8 だけとする self-review の C2-3 には入っていない。168 行の (a) の `:1003`、`:1045`(後回しの close の台帳の読み取り)は、S8 の `TestOrgCloseDeferredSelf_NothingRestored_NamesTheManualClose` の「manifest unreadable」の 2 例が通るようになり、「テストがない」は成り立たない。全件の対応は evidence の log にある | /sync-docs で、行番号を関数名と動作に置き換え、168 行の (a) から 2 か所を外す。self-review の「Tech debt identified」の 2 行(C2-1・C2-2・C2-5・C2-6 と C2-4)を足す |
| V2-4 | LOW | plan の進捗の行(`:156`)は「Verify plan の AC1〜AC13 は S7 より前の記述で、承認 digest が変わるので書き換えていない」と書く。5f26af07 の承認のやり直しで Verify plan(`:113`)は AC1〜AC16 に書き換わったので、この行は今の plan と食い違う(`:154` は書き換えたと書いている) | /sync-docs で `:156` に「5f26af07 で AC1〜AC16 に直した」と足す。進捗の節は digest の対象外(`scripts/plan-visual.sh` の `digest_body`)なので、承認は崩れない |
| V2-5 | LOW | 後回しの close が失敗したとき、stdout にはその前に `stopped seat "<seat>"` や `disbanded org "<org>"` が出ている(AC12 のとおり、出力を済ませてから閉じるため)。そのあと stderr のエラーが「active に戻した」と言い、終了コードは 1 になる。台帳とエラーは合っているが、stdout だけを読む人やスクリプトは、止まった・解散したと受け取る。AC3 の「止められなかった座席を stopped と表示しない」は Disband の `FailedSeats` についての条件なので、AC3 の違反ではない。CLI のテストは失敗した回の stdout を見ていない | 直さなくてよい(閉じる前に出力するしかない)。/sync-docs で、skill に「最後の close が失敗したときは、先に出た stdout の行より終了コードと stderr が正しい」と 1 文足すと、leader が読み違えない |
| V2-6 | LOW | leader の雛形の手順 8 は、終了コード 1 なら打ち直すよう書く。後回しの close の失敗は、S8 の補償で打ち直しが効く。ただし補償の書き込みも失敗したとき(または後回しの段で台帳が読めないとき)は、打ち直した `disband` は閉じる対象を見つけない。台帳が書けるように戻っていれば、`disbanded` をもう一度書いて終了コード 0 で終わり、セッションは動いたまま残る。エラー文は `herdr workspace close <id>` を示すが、雛形はそれを打てとは書いていない。台帳への追記が直前に通ったあとで失敗する場合だけなので、起きにくい | 雛形は `internal/org/prompts/` にあり、直すと `internal/` の変更になる。直すなら手順 8 に「stderr が herdr のコマンドを示したら、それを打つ」と足す。直さないなら PR の known gap に書く |

## Coverage gaps

- テストは実行していない(/test の担当)。上の表の「Met」は、コードとテストの中身を読み、テストが AC の各場合を持つことを確かめた結果で、テストが通ることは確かめていない
- 本物の herdr で、自分の pane や workspace の close が失敗する場合は誰も観察していない(fake の driver と sh の stub だけ)。close が 10 秒の期限で切れたあとに herdr の側で閉じ、補償の途中でプロセスが終わる場合は、self-review が「次の stop や disband が not-found を閉じ済みと読んで直る」とコードから判断しているが、確かめていない
- self-review の C2-1(補償が driver 呼び出しの前に読んだ台帳で判断する窓)と C2-2(workspace 側の座席を pane id の一致だけで選ぶ)は、コードを読んで成り立つことを確かめたが、起こしてはいない
- `stop --all --force` で自分の pane の close が失敗したときに終了コード 0 になることは、コード(`org.go:756` と `:903-905`)で確かめただけで、テストはない
- cycle 1 の残り(codex の座席、別の herdr セッションからの not-found、`WaitDelay` のほかの呼び出しへの効き方、`leaderActivityEventCount`、Windows)は付録のとおりで、変わっていない。`leaderActivityEventCount` は tech-debt の 167 行に移った

## Verdict

- Verdict: pass
- Verified: AC1〜AC16 をコードとテストの中身で確かめた。AC16 は 9 つの要件ごとに実装とテストを対応させ、台帳の状態を `status` と `--dry-run` で観察した(P5〜P7)。S8 のあとも AC10・AC12・AC14 は成り立つ。AC10 は cycle 1 が見落とした経路(`--force` で後回しの close が失敗)が S8 で直った。静的解析は `run-static-verify.sh` が rc 0。plan の承認の digest が一致。`/org` skill の 4 面と recipe の 2 面が一致。cycle 1 の D-1〜D-4、V-1、V-3 は解消
- Partially verified: 文書は V2-1(skill に後回しの close の失敗の動きがない)、V2-2(`disband --help` の言い過ぎ)、V2-3(tech-debt の行番号と、まだ足していない 2 行)、V2-4(plan の進捗の行)が残る。V2-1、V2-3、V2-4 は /sync-docs で直せる。V2-2 と V2-6 は `internal/` の変更なので、tech-debt か PR の known gap に送る
- Not verified: テストの実行、本物の herdr での後回しの close の失敗、C2-1・C2-2 の窓、`stop --all --force` の自分の pane の失敗、cycle 1 から残る項目

## 付録: cycle 1 の verify report(原文)

`docs/tech-debt/README.md` の 164 行と 167 行が、この付録の V-2 と Coverage gaps を指している。見出しを 1 段下げた以外は de154798 の原文のまま。行番号は 801ca77f のもの。

- Date: 2026-10-07
- Plan: docs/plans/active/2026-10-07-org-stop-all.md
- Verifier: verifier subagent (Claude Opus 5.5)、pipeline cycle 1(`cycle-count.json` は 1)
- Scope: 仕様への適合(AC1〜AC15)、静的解析、文書のずれ。対象は `git diff f423f230...HEAD`(HEAD 801ca77f、28 ファイル、+5738/-165)。S7(05977322)と、self-review の残りの (b) を閉じた evidence の Run 5(801ca77f)を含む。テスト(`./scripts/run-test.sh`、`go test`)は /test の担当なので実行していない。HEAD からビルドしたバイナリを scratchpad の一時ディレクトリで動かす観察用の probe は行った(テストスイートではない。herdr と agmsg は呼んでいない)
- Evidence: `docs/evidence/verify-2026-10-07-org-stop-all.log`(`docs/evidence/*.log` は gitignore の対象なので手元にだけ残る)。runner 自身のログは `docs/evidence/verify-2026-10-07-132918.log`

### Spec compliance

plan の承認: `./scripts/plan-visual.sh digest` は `15eb2a79e2f6` を返し、plan の `- Approved: 2026-10-07 sha256:15eb2a79e2f6` と一致した。進捗の行に書かれた逸脱(details の `pane=` を `ctrl_c=` に改名、閉じられなかったときは Leave しない、座席が 1 つでも止まらなければ workspace を閉じない、`--force` では閉じられない workspace にも `org_workspace_closed` を書く、disband の対象に「開いた workspace が残る org」を含める、ExecRunner の `WaitDelay`)は、どれもその内容で実装されている。

| Acceptance criterion | Status | Evidence |
| --- | --- | --- |
| AC1: stop は C-c のあと pane を閉じ、`stopped` の詳細に結果が残る。見つからなければ閉じ済み | Met(テストの実行は /test) | `stopSeatPane`(`internal/org/verbs.go:949`)は確認 → C-c(`:964`)→ `PaneClose`(`:970`)の順で、詳細は `ctrl_c=ok pane=closed leave=ok` の形。not-found は確認の段(`:957`)と close の段(`:976`)のどちらでも `pane=already closed`。テストは `TestOrgStop_CallOrder_CtrlCThenCloseThenLeaveThenStopped`(`verbs_test.go:1738`、herdr の呼び出し順と Leave の時点で記録がまだないことを見る)、`TestOrgStop_PaneNotFound_CountsAsClosed`(`:1791`)、`TestOrgStop_PaneGoneAtCheck_RecordsAlreadyClosed`(`:2253`) |
| AC2: 閉じられなければ `stop_failed`、終了コード 1、status で active のまま、打ち直しで `stopped` | Met(テストの実行は /test) | `Stop`(`verbs.go:878-933`)は `closeErr != nil && !Force` で `EventStopFailed` を書き、Leave を飛ばし、座席と org を名指しした Err を返す。`stop_failed` は `stateEvents`(`internal/org/seat.go:52-60`)に入っていないので Roster は変わらない。CLI は Err をそのまま返す(`internal/cli/org.go:761-763`)。`TestOrgStop_CloseFails_RecordsStopFailedThenRetryStops`(`verbs_test.go:1818`)が `stop --seat` 側の打ち直しまで、`TestOrgStopAll_OneCloseFails_ExitsOneThenRerunStopsIt`(`internal/cli/org_stop_all_test.go:184`)が CLI の終了コード、`ralph org status` の `(active)`、`stop --all` の打ち直しを確かめる |
| AC3: disband は座席を止めたあと workspace を閉じ、`org_workspace_closed` と `disbanded`。失敗があれば `disbanded` なし、終了コード 1、失敗した座席を stopped と出さない、打ち直しで残りを片付ける | Met(テストの実行は /test) | `Disband`(`verbs.go:1535`)と `closeOrgWorkspace`(`:1621`)。座席が 1 つでも失敗すれば workspace は閉じずに失敗として並べる(`:1567-1568`)。`DisbandResult.recordStop`(`:1444`)は失敗した座席を `FailedSeats` に入れ、`printDisbandResult`(`org.go:1044`)は `StoppedSeats` だけを stdout に出し、`disbanded org` は `Disbanded` のときだけ出す。テストは `TestOrgDisband_ClosesSeatPanesThenWorkspace`(`verbs_test.go:232`)、`_SeatCloseFails_LeavesWorkspaceAndRetries`(`:281`)、`_WorkspaceCloseFails_NoDisbandedThenRetry`(`:340`)、CLI の `TestOrgDisband_OneOrgSeatFails_NoDisbandedLine`(`org_stop_all_test.go:322`) |
| AC4: `stop --all` は全 org の active な座席を止め、失敗を `<org_id>/<seat_id>` と理由で stderr に並べて終了コード 1、全部止まれば 0 | Met(テストの実行は /test) | `StopAll`(`internal/org/verbs_all.go:116`)は 1 回読んだ Roster の active な座席を順に `Stop` し、失敗で止めない。`printStopAllResult`(`org.go:844`)は `stopped seat <org>/<seat>` を stdout、`seat <org>/<seat> not stopped: <理由>` を stderr に出す。テストは `verbs_all_test.go:96`、`:146`、CLI の `org_stop_all_test.go:139`、`:184` |
| AC5: `disband --all` はまだ disband していない全 org を disband し、失敗した org は次の `--all` でまた対象 | Met(テストの実行は /test) | `DisbandAll`(`verbs_all.go:264`)と `orgsToDisband`(`:307`)。対象は「最後の `disbanded` のあとに座席の状態イベントか workspace の記録がある org」と「開いた workspace が残る org」(後者は進捗の行で決めた逸脱。D-2 を参照)。テストは `TestOrgsToDisband_Definition`(`verbs_all_test.go:418`)、`TestOrgDisbandAll_FailedOrgNotDisbandedThenRetried`(`:556`)、CLI の `TestOrgDisbandAll_OneOrgFails_ExitsOneAndNextRunRetriesIt`(`org_stop_all_test.go:277`) |
| AC6: `--all` と `--org-id`、`stop` の `--all` と `--seat` の同時指定はエラーで何もしない。`--all` なしは `--org-id` が要る | Met | `rejectFlagsWithAll`(`org.go:784`)は state dir を解決する前に `Changed` を見る(`:738`、`:1008`)。`TestOrgStopDisbandAll_ConflictingFlags_RejectedWithoutChanges`(`org_stop_all_test.go:436`)が 5 例で manifest が変わらないことと herdr・agmsg の呼び出しがないことを見る。probe P1 でも同じ文言と rc 1、state dir が作られないことを確かめた |
| AC7: `--all --dry-run` は herdr も agmsg も呼ばず、dry-run の記録だけ | Met | `StopAll` は `DryRun` を `Stop` に渡し、`Stop` の dry-run は driver を呼ばない(`verbs.go:852-853`)。自分の pane の判定も dry-run では空にする(`verbs_all.go:125-127`)。`DisbandAll` も同じ(`:275`)。テストは `verbs_all_test.go:224`、`:640`、CLI の `TestOrgStopDisbandAll_DryRun_RecordsWithoutDriverCalls`(`org_stop_all_test.go:481`)。probe P2 で空の台帳の `--dry-run` が rc 0 で終わることも見た |
| AC8: 台帳に記録していない pane と workspace は閉じない | Met(テストの実行は /test) | `Stop` は `seat.PaneID` だけ、`Disband` は `openOrgWorkspaces(rr.Events, p.OrgID)`(`internal/org/spawn.go:1729`)だけを閉じる。後回しの close も台帳から座席と org を引き直す(`verbs.go:1006`、`:1048`)。テストは `TestOrgStop_ClosesOnlyTheSeatsRecordedPane`(`verbs_test.go:2124`)、`TestOrgDisband_ClosesOnlyThisOrgsRecordedIDs`(`:643`) |
| AC9: `/org` skill の動詞の表・締めの順・完了条件(4 面)、leader の雛形、README、FR-2。`check-skill-sync.sh` と `check-sync.sh` | Met(ヘルプ文に D-1) | skill は `.claude/skills/org/SKILL.md:167-168`(表)、`:255-260`(締めの順)、`:327-333`(運用の締め)、`:342-347`(完了条件)。4 面は `cmp` で同一。leader の雛形は `internal/org/prompts/leader.md` の手順 6〜8 と「運用規律」。README は Commands の表と org 節の 1 文(`README.md:124`、`:245`)。FR-2 は `docs/specs/2026-10-07-org-multi-org-director.md:36`。recipe の 2 面も `cmp` で同一。どちらの check も OK |
| AC10: `--force` は閉じられなかった座席にも `stopped`(forced)、disband は `disbanded` も書き、失敗は stderr、終了コード 0 | Met(テストの実行は /test) | `notClosedNote(..., true)`(`verbs.go:736`)が `(forced)` を足す。forced の失敗は `Errs` に入らない(`:1448-1449`、`:1465`)。CLI は `printSeatFailure` / `printWorkspaceFailure` で `warning:` として stderr に出す(`org.go:810`、`:820`)。テストは `TestOrgStop_Force_CloseFails_RecordsStoppedForced`(`verbs_test.go:1899`)、`TestOrgDisband_Force_RecordsPastCloseFailures`(`:432`)、CLI の `TestOrgStopAndDisband_Force_FailuresAreWarningsExitZero`(`org_stop_all_test.go:351`) |
| AC11: 応答しない呼び出しは期限で打ち切られ、「閉じられなかった」になり、`--all` は残りを続ける | Met(テストの実行は /test) | `callWithTimeout`(`verbs.go:631`)が呼び出しごとに `defaultDriverCallTimeout`(10 秒、`:603`)の新しい期限を作る。偽の driver は `<-ctx.Done()` まで返らない(`spawn_test.go:175`、`:278`、`:292`、`:306`、`:377`)。テストは `TestOrgStop_UnansweredHerdr_EachCallTimesOut`(`verbs_test.go:1942`、C-c と close が別々の期限を持つことを経過時間で見る)、`_UnansweredLeave_`(`:1979`)、`TestOrgStopDisband_UnansweredGet_LeavesItAlone`(`:2284`)、`TestOrgDisband_UnansweredWorkspaceClose_TimesOut`、`TestOrgStopAll_UnansweredHerdrForOneOrg_OthersStillStopped`(`verbs_all_test.go:260`)、`TestOrgDisbandAll_UnansweredHerdrForOneOrg_OthersStillDisbanded`(`:687`) |
| AC12: `HERDR_PANE_ID` / `HERDR_WORKSPACE_ID` が対象を指すとき、その close はほかのすべての close と記録のあと | Met(テストの実行は /test) | `Stop` は自分の pane に C-c も close も送らず `stopped` を書いて id を返す(`verbs.go:959-960`、`:935-937`)。`Disband` と `StopAll` / `DisbandAll` は自分の座席と org を最後に回し、ほかが失敗すれば手を付けない(`verbs.go:1588`、`verbs_all.go:140-146`、`:281-287`)。CLI は出力のあとに `closeDeferredSelf`(`org.go:877`)を呼ぶ。`TestOrgStopDisband_OwnPaneOrWorkspace_ClosedLastAfterOutput`(`org_stop_all_test.go:628`)は herdr の stub と出力を 1 つのファイルに書かせ、close が最後の行であることを 5 例で見る。org 側は `verbs_test.go:502`、`:2004`、`verbs_all_test.go:321`、`:755`、`:789` |
| AC13: spawn → disband → 同じ org_id の spawn は新しい workspace を作る | Met(テストの実行は /test) | `resolveWorkspace`(`spawn.go:1705`)は `openOrgWorkspaces` の先頭だけを使い回す。`TestOrgSpawn_AfterDisbandClosedWorkspace_CreatesNewWorkspace`(`spawn_test.go:641`)は `workspace_create` が 2 回、閉じた ws-1 に tab を作らないことを見る |
| AC14: label が座席 id / org_id でない pane と workspace は閉じず、`stop_failed` / workspace の失敗で終了コード 1。`--force` でも閉じない。後回しの close も同じ確認 | Met(テストの実行は /test) | `confirmSeatPane`(`verbs.go:669`)と `confirmOrgWorkspace`(`:715`)。C-c と close の呼び出しはどれも直前にこの確認を通る(`:953`、`:1010`、`:1052`、`:1623`)。tab の not-found は `%v` で包み直し、「閉じ済み」と読ませない(`:693`)。テストは `TestOrgStop_PaneNotConfirmed_NoCtrlCNoClose`(`verbs_test.go:2165`、4 例 × `Force` の有無)、`TestOrgDisband_WorkspaceNotConfirmed_LeftOpen`(`:2340`)、`TestOrgStopDisband_OwnIDNotConfirmed_NotDeferredNotClosed`(`:2385`)、`TestOrgCloseDeferredSelf_RecheckedBeforeTheClose`(`:2435`)、`TestOrgStopAll_OwnPaneIDNotConfirmed_NotDeferred`(`verbs_all_test.go:387`)。driver の get が読めない応答を成功にしないことは `TestHerdr_PaneGet_TabGet_WorkspaceGet`(`internal/org/driver/herdr_test.go:557`) |
| AC15: 成功で終わったコマンドの孫がパイプを握っていても ExecRunner は成功を返す | Met(テストの実行は /test) | `internal/org/driver/driver.go:60` は `errors.Is(err, exec.ErrWaitDelay) && ctx.Err() == nil` を成功にする。`go doc os/exec.ErrWaitDelay` は「プロセスが成功の終了コードで終わったときだけ返る」と書いており、判断の前提と合う(go.mod は `go 1.25.8`、`WaitDelay` は 1.20 から)。テストは `TestExecRunner_Run_SuccessNotFailedByGrandchildHoldingPipes`(`driver_test.go:113`)と、期限切れ側の `TestExecRunner_Run_TimeoutNotHeldByGrandchild`(`:86`) |

Verify plan の残りの点も確かめた。1 段目の古い台帳の判定は、`--all` の 2 つの経路でも `newOrgRuntime(..., orgLedgerMutating)` を通る(`org.go:741`、`:1011`)。probe P3 で、古い台帳に active な座席がある linked worktree からの `stop --all`、`disband --all`、`stop --all --dry-run` が rc 1 で拒否されることを見た。テストはこの 3 つを持たない(V-1)。spawn の補償(`compensatePaneCtx`、`spawn.go:1838`)は変わっていない(Non-goals のとおり)。

### Static analysis

| Command | Result | Notes |
| --- | --- | --- |
| `./scripts/run-static-verify.sh` | PASS(rc 0) | Mode static、scope changed。`verify.local.sh`(shellcheck、hook の `sh -n` 20 本、settings の `jq -e` 2 本、Codex の hook guard 3 本、`check-sync.sh`、`check-pipeline-sync.sh`、`check-skill-sync.sh`、`check-template-purity.sh`、tech-debt README の plan の参照)、golang verifier(`gofmt: ok`、`go vet ./...` は出力なしで成功、golangci-lint `0 issues.`、staticcheck は出力なしで成功)、branch secret scan(`f423f230..801ca77f` clean) |
| `./scripts/check-sync.sh`(上の中) | PASS | IDENTICAL 164、DRIFTED 0、ROOT_ONLY 0、TEMPLATE_ONLY 11、KNOWN_DIFF 5 |
| `./scripts/check-skill-sync.sh`(上の中) | PASS | 13 skill が一致 |
| `git diff --check f423f230...HEAD` | PASS | 出力なし |
| 追加行の U+FFFD 検索 | PASS | 0 件 |
| ミラーの `cmp` | PASS | `/org` skill の 4 面、recipe の 2 面 |
| `./scripts/plan-visual.sh digest` | PASS | `15eb2a79e2f6`、plan の Approved 行と一致 |

### Documentation drift

plan の Verify plan の grep(`git grep -n 'C-c\|disband\b.*report\|herdr workspace / agmsg' -- .claude/skills/org internal/org/prompts docs/specs README.md`)で出るのは、新しい止め方と締めの順を書いた行だけだった(skill の表、README、FR-2、leader の雛形)。範囲をリポジトリ全体に広げ、`C-c` / `send-keys` / `close the pane` / `残留` も grep したが、「stop は C-c だけ」「pane は人が herdr で閉じる」といった古い説明は残っていない。

| Doc / contract | In sync? | Notes |
| --- | --- | --- |
| `/org` skill の 4 面 | Yes | 動詞の表、締めの順、運用の締め、完了条件。`--force` で記録したものは herdr に残りうる、記録していない pane と workspace は閉じない、と書いてある |
| leader の雛形(`internal/org/prompts/leader.md`) | Yes | report のあと disband を最後に打つ。disband が終了コード 1 のときはセッションが続くので打ち直す。実装の「ほかが失敗したら自分の pane と workspace に手を付けない」と合う |
| `README.md` | Yes | Commands の表と org 節 |
| 仕様 FR-2 | Yes | 1 座席の stop と 1 org の disband も閉じる、`stop_failed`、`--force` |
| `docs/recipes/codex-seat-permissions.md` の 2 面 | Yes | |
| `ralph org stop` / `disband` の `--help`(`org.go:724-734`、`:994-1004`) | 一部ずれ(D-1) | 「C-c を送り pane を閉じる」とだけ書き、持ち主の確認(label が違えば C-c も close もしない)と、`--force` でも確認を通らない pane と workspace は閉じないことに触れていない。`--force` の行は「pane が閉じられなくても `stopped` を書く」なので誤りではないが、herdr に pane が残ることが読み取れない。self-review が /sync-docs に引き継いだ点と同じ |
| `disband --all` の対象 | 一部ずれ(D-2) | skill の表は「まだ解散していない全 org」、ヘルプは「every org_id that still needs it」と書く。実装は、古い ralph の disband が workspace を閉じずに残した org も対象にする(`verbs_all.go:299-301`)。このため、この版に上げて最初の `disband --all` は、解散済みの古い org にも `disbanded org <id>` を出し、label が合えばその workspace を閉じる。label の確認があるので閉じすぎはないが、利用者には予想外の出力になりうる。skill の表に 1 文足すのがよい |
| plan(`docs/plans/active/2026-10-07-org-stop-all.md`) | 一部ずれ(D-3) | Verify plan の行(`:109`)が「AC1〜AC13」のまま(S7 で AC14・AC15 が増えた)。AC のチェックボックス(`:80-94`)は全部未チェック、進捗の「Review artifact created」(`:145`)も未チェック。この report をもとに /sync-docs か /pr で直せばよい |
| `docs/tech-debt/README.md` | 未反映(D-4、/sync-docs の予定) | self-review の F-3〜F-10 と、`compensateStale` が確認なしに C-c を送る既存の経路の行は、この diff にはまだない。self-review の「Tech debt identified」が /sync-docs に回している |
| `docs/evidence/herdr-pane-close-2026-10-07.md` の Plan 行 | Yes(先の参照) | `docs/plans/archive/...` を指し、「PR 作成時に active から移動」と添えてある。/pr で解消する |

### Observational checks

HEAD 801ca77f からビルドしたバイナリ(`go build ./cmd/ralph`、go 1.26.0、macOS)を、`HERDR_PANE_ID` / `HERDR_WORKSPACE_ID` / `RALPH_ORG_STATE_DIR` を外した環境で、scratchpad の一時ディレクトリに向けて動かした。herdr と agmsg を呼ぶ経路は通していない。

- P1(同時指定): `stop --all --org-id x`、`stop --all --seat y`、`disband --all --org-id x` は rc 1 で `--all cannot be combined with --<flag>`。`--all` なしの `stop --seat y` と `disband` は `--org-id is required`。どれも state dir を作らない
- P2(空の台帳): `stop --all --dry-run` は `no active seats`、`disband --all --dry-run` は `no orgs to disband`、どちらも rc 0
- P3(古い台帳): `GIT_CONFIG_GLOBAL=/dev/null` の一時リポジトリに linked worktree を作り、その `.harness/state/org/manifest.jsonl` に active な座席を 1 つ置いた。そこから打った `stop --all`、`disband --all`、`stop --all --dry-run` は rc 1 で `org: refusing to change the org ledger: this linked worktree still has an older ledger at ...`。main 側に台帳のディレクトリはできなかった
- P4(ヘルプ): `ralph org stop --help` と `disband --help` に Long の説明、`--all` / `--force` / `--dry-run` の行、`--org-id` の「stop --all と disband --all では不要」が出る(D-1 の内容はここで見た)

### Findings

| ID | Severity | Finding | Recommendation |
| --- | --- | --- | --- |
| V-1 | LOW | `stop --all` と `disband --all` が 1 段目の古い台帳の判定を通ることはコード(`org.go:741`、`:1011`)と probe P3 で確かめたが、`TestOrgLegacyLedger_MutatingVerbsRefusedWhileLegacySeatsActive`(`internal/cli/org_legacy_ledger_test.go:108`)の表は `--org-id` 付きの stop と disband しか持たない。将来 `--all` の経路が別の runtime の作り方に変わっても、テストは落ちない | /test か次の変更で、表に `stop --all` と `disband --all` の 2 行を足す |
| V-2 | LOW | `resolveWorkspace`(`spawn.go:1705`)は、台帳で開いている workspace の id を label を確かめずに使い回す(main と同じ)。herdr のセッションの保存ファイルが失われて id が振り直されると、同じ org_id の次の spawn が、その id を持つ無関係の workspace に座席の tab を作りうる。この PR の持ち主の確認と組み合わさると、その座席は workspace の label が org_id でないので `stop` が毎回 `stop_failed` になり、`--force` は記録だけで pane を閉じないため、エージェントは利用者の workspace で動き続ける。起きる条件は保存ファイルの消失と id の一致が重なる場合だけ | 直すなら、使い回す前に `confirmOrgWorkspace` を通し、確認できなければ新しい workspace を作る。直さないなら tech-debt に 1 行(self-review の `compensateStale` の行と並べる) |
| V-3 | LOW | plan の Affected areas は「leader の雛形と、雛形のテスト」と書くが、`internal/org/prompts_test.go` は変わっていない。今のテスト(`TestRenderRolePrompt_Leader_AllKnownVarsSubstituted`、`:214`)は `ralph org report` の文字列があることしか見ず、「report のあと disband を最後に打つ」という新しい締めの順はテストで固定されていない | /test で、雛形の中で `ralph org report` が `ralph org disband` より前に出ることを確かめる 1 行を足す。足さないなら known gap として PR に書く |

### Coverage gaps

- テストは実行していない(/test の担当)。上の表の「Met」は、コードとテストの中身を読み、テストが AC の各場合を持つことを確かめた結果で、テストが通ることは確かめていない
- 本物の herdr での挙動は、実装者の記録(`docs/evidence/herdr-pane-close-2026-10-07.md` の Run 1〜5)を読んだだけで、verifier は herdr を動かしていない。codex の座席の C-c と label、起動を終えた Claude Code が出す端末タイトルのあとの label は、その記録でも確かめていない
- 別の herdr セッション(`HERDR_SESSION` や `HERDR_SOCKET_PATH` の違う shell)から打った `stop` で、本物の pane が `pane_not_found` と読まれる場合(self-review の F-1 の残余 (c))は確かめていない
- ExecRunner の `WaitDelay`(`driver.go:51`)は Stop と Disband だけでなく、spawn・send・agent start など ExecRunner を使うすべての呼び出しに効く。成功で終わったコマンドの子がパイプを握る場合、今は 2 秒後に途中までの出力で成功を返す(前は子が終わるまで待った)。コードを読んだだけで、本物の herdr と agmsg では観察していない
- `ralph org watch` の deadman が数える leader の活動(`leaderActivityEventCount`、`internal/org/watch.go:816`)は `stop_failed` を数えない。leader が `stop_failed` しか出さない間に deadman が鳴りうるが、影響は確かめていない
- Windows のパスと herdr は見ていない

### Verdict(cycle 1)

- Verdict: pass
- Verified: AC1〜AC15 をコードとテストの中身で確かめた(AC6・AC7 と、`--all` での古い台帳の判定は probe でも確認)。静的解析は `run-static-verify.sh` が rc 0。plan の承認の digest が一致。`/org` skill の 4 面と recipe の 2 面が一致
- Partially verified: 文書は D-1(`stop` / `disband` のヘルプ文に持ち主の確認がない)、D-2(`disband --all` が古い ralph の残した workspace の org も対象にすることが書かれていない)、D-3(plan の AC の範囲とチェックボックス)、D-4(tech-debt の行、/sync-docs の予定)が残る。テストの固定は V-1・V-3 が欠ける
- Not verified: テストの実行、本物の herdr での close と label の確認(実装者の evidence に頼った)、codex の座席、別セッションからの not-found、`WaitDelay` のほかの呼び出しへの効き方
