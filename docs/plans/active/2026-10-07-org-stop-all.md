# org-stop-all

- Status: Approved
- Approved: 2026-10-07 sha256:d9d26e5e1211
- Owner: Claude Code
- Date: 2026-10-07
- Related request: 機能ごとの org と director の仕様(`docs/specs/2026-10-07-org-multi-org-director.md`)の 2 段目(FR-2 横断の status と stop)。ユーザーが「pane も閉じる」を選んだ(2026-10-07)
- Related issue: N/A
- Type: feat
- Branch: feat/org-stop-all

## Objective

`ralph org stop --all` と `ralph org disband --all` を足し、`--org-id` なしで全 org の座席を止められるようにする。あわせて、座席を止めるときに herdr の pane を閉じてプロセスを終わらせ、disband のときに org の herdr workspace を閉じる。1 座席の `stop` と 1 org の `disband` も同じ動きにする。

今の `stop` は C-c を 1 回送り、agmsg から外して台帳に `stopped` を書くだけ。Claude Code の C-c は、動いている処理を中断し、何もしていないときは入力欄を消すだけで、2 回目で終了する(https://code.claude.com/docs/en/interactive-mode の Keyboard shortcuts)。このため座席の処理は止まるが、プロセスは pane に残り、片付けは人が herdr で行う前提になっている(`/org` skill の完了条件)。

## Scope

- herdr の driver(`internal/org/driver/herdr.go`): `PaneClose(ctx, paneID)`(`herdr pane close <pane_id>`)と `WorkspaceClose(ctx, workspaceID)`(`herdr workspace close <workspace_id>`)を足す。herdr が `pane_not_found` / `workspace_not_found` のコードで返すエラーは、呼び出し側が「もう閉じている」と判定できるように区別して返す。`internal/org/spawn.go` の `HerdrClient` に 2 つのメソッドを足す
- 呼び出しの期限: `Stop` と `Disband` が呼ぶ herdr と agmsg の操作(C-c、pane を閉じる、agmsg から外す、workspace を閉じる)に、1 回ずつ期限(既定 10 秒)を付ける。今は `context.Background()` で期限がない。期限切れは「閉じられなかった」として扱い、`--all` は残りを続ける
- `Stop`(`internal/org/verbs.go`): 今の C-c のあと、pane を閉じる。順番は C-c → pane を閉じる → agmsg から外す → 台帳に `stopped` を書く。pane が見つからないときは閉じ済みとして扱う。閉じられなかったとき(herdr に繋がらない、期限切れなど)は `stopped` を書かず、状態を変えない新しいイベント `stop_failed`(詳細つき)を書いてエラーを返す。座席は台帳の上で「動いている」のまま残るので、`stop --seat` でも `stop --all` でも打ち直せば拾う
- `--force`(`stop` と `disband`): herdr に繋がらないまま片付けたいとき用。閉じられなかった座席にも `stopped` を書き、詳細に `pane=close failed (forced): …` を残す。失敗は stderr に出すが、終了コードは 0
- `Disband`: 動いている座席を `Stop` で止めたあと、その org の herdr workspace を閉じ、閉じたら org の新しいイベント `org_workspace_closed` を書く。workspace が見つからないときは閉じ済みとして扱う。止められなかった座席か閉じられなかった workspace が 1 つでもあれば `disbanded` を書かずにエラーを返す(`--force` のときは書く)。`DisbandResult` は止められた座席と止められなかった座席(理由つき)を分けて返す(今は失敗した座席も `StoppedSeats` に入る)
- 自分の pane を最後に閉じる: herdr は pane の中のプロセスに `HERDR_PANE_ID` と `HERDR_WORKSPACE_ID` を渡す。コマンドを打った pane が閉じる対象に入っているとき(headless の leader が自分で `disband` や `stop --seat leader` を打つ場合など)は、その pane と、それを含む workspace を閉じる操作を最後に回す。台帳への記録と出力をすべて済ませてから閉じる。閉じた時点でコマンドのプロセスも終わる
- workspace の作り直し(`internal/org/spawn.go` の `resolveWorkspace`): 今は最初の `org_workspace_created` の id を無条件に使い回す。そのあとに `org_workspace_closed` があれば、新しい workspace を作る
- 全 org の操作(`internal/org/verbs.go`): 全 org の動いている座席を止める関数と、まだ disband していない全 org を disband する関数を足す。どちらも 1 つの失敗で止めず、最後まで進めて、止められなかった座席と閉じられなかった workspace をまとめて返す
- CLI(`internal/cli/org.go`): `ralph org stop --all` と `ralph org disband --all`、`--force`。`--all` のときは `--org-id` を要らなくする。`--all` と `--org-id`、`stop` の `--all` と `--seat` は同時に指定できない(エラー)。止めた座席は `stopped seat <org_id>/<seat_id>` を stdout に、止められなかったものは理由つきで stderr に出し、1 つでもあれば終了コード 1(`--force` のときは 0)。`--dry-run` は `--all` でも使え、herdr を呼ばずに記録だけする。古い台帳の判定(1 段目)は今までどおり書き換えの動詞として通す
- leader の雛形(`internal/org/prompts/leader.md`)と `/org` skill: 締めの順を「座席を stop → `report` → `disband`」に変え、`disband` がそのセッションの最後のコマンドになる(自分の pane が閉じる)と書く
- 文書: `/org` skill の動詞の表と完了条件(4 面)、`README.md` の Commands の表、仕様の FR-2 に「1 座席の stop も pane を閉じる」を足す、`docs/evidence/` に herdr の確認の記録

## Non-goals

- `ralph org watch` のプロセスを止めること(6 段目の横断の見張りで扱う)
- spawn が失敗したときの補償(今の C-c だけ)を変えること
- agmsg の team の削除(今の Leave のまま)
- ralph が作っていない herdr の workspace や pane を閉じること。閉じるのは台帳に記録した pane と workspace だけ
- `herdr pane close` / `workspace close` がない古い herdr を見分けること。その場合は閉じる操作が失敗し、`stop_failed` になる(`--force` で片付けられる)。版の検査は `ralph doctor` の別の作業にする
- 閉じる前に pane の画面のログを保存すること(必要なら先に `ralph org read` で読む、と文書に書く)
- `ralph status` の変更(今のまま全 org を表示する)

## Assumptions

- herdr 0.7.5 の隔離したサーバーで確かめた(2026-10-07): `herdr pane close <id>` は `{"result":{"type":"ok"}}` を返し、pane の中のプロセス(`sleep`)は終わる。pane が 1 つだけの tab は tab ごと消える。見つからない id には終了コード 1 と `{"error":{"code":"pane_not_found",...}}` を返す。`herdr workspace close <id>` も同じ形で、見つからないときは `workspace_not_found`。pane の中のプロセスには `HERDR_ENV=1`、`HERDR_PANE_ID`、`HERDR_TAB_ID`、`HERDR_WORKSPACE_ID` が渡る。`herdr pane current` は pane の外ではフォーカス中の pane を返すので、自分の位置を知るには使わない。これらは evidence の記録として残す
- herdr のサーバーが止まっていると、そのサーバーの pane のプロセスも動いていない(サーバーが端末を持っているため。推測、未確認)。このときの閉じる操作は接続エラーになり、「閉じられなかった」として報告する
- codex の C-c の動きは確かめていない。pane を閉じればプロセスは終わるので、結果は driver によらない
- `stop_failed` と `org_workspace_closed` は状態のイベントではない。`Roster` は状態のイベントだけを見る(`internal/org/manifest.go` の `Roster`、`isStateEvent`)ので、座席の状態の判定は変わらない

## Affected areas

- `internal/org/driver/herdr.go`、`internal/org/driver/herdr_test.go`
- `internal/org/spawn.go`(`HerdrClient` のメソッド、`resolveWorkspace`)
- `internal/org/verbs.go`(`Stop`、`Disband`、全 org の操作)と、そのテスト(`internal/org/verbs_test.go`、`internal/org/spawn_test.go` の `fakeHerdr`)
- `internal/org/prompts/leader.md`(締めの順)と、雛形のテスト
- `internal/cli/org.go`(`stop` と `disband` の `--all` と `--force`)と、そのテスト
- `.claude/skills/org/SKILL.md` と 3 つの写し、`README.md`、`docs/specs/2026-10-07-org-multi-org-director.md`
- `docs/evidence/herdr-pane-close-2026-10-07.md`(新規)

## Visual review

- ページ: `.harness/state/plan-visual/org-stop-all.html`(図 1 全体、図 2 1 座席の stop の前と後、図 3 disband --all の流れ)
- セルフチェック: 全体と全体図(`--fragment overview`)を `plan-visual.sh shot` で撮って確認した。consult と Codex の指摘を入れた版も撮り直し、図 2 の右下のノードの文字のはみ出し、全体図の `spawn.go` の説明のはみ出し、文書の列のノードが枠に接していたのを直して、もう一度確かめた

## Design decisions

- 座席の止め方: C-c で中断したあと pane を閉じる。1 座席の stop と `--all` で動きを揃える(ユーザーが「pane も閉じる」を選んだ。2026-10-07)
- 閉じられなかった座席には `stopped` を書かない(consult の指摘を受けて変えた)。`stopped` を書くと、台帳の `stopped` が「プロセスが終わった」を意味しなくなり、「ralph の仕組みだけで停止できる」の根拠が崩れる。herdr が生きていて期限切れになった場合はプロセスが残る。代わりに状態を変えない `stop_failed` を書き、打ち直しで拾えるようにする。座席が「動いている」のまま残ると上限(3 段目)を塞ぐが、異常のときは新しい起動を断るという council の合意(fail-closed)と合う。herdr に繋がらないまま片付けたいときは `--force` を使う
- `--all` は 1 つの失敗でも、1 つの呼び出しが応答しなくても止めない(呼び出しごとの期限)。全 org を止めたいときに、1 座席のせいで残りが動き続けるのを防ぐ(Codex plan advisory の指摘 2)
- 自分の pane は最後に閉じる。拒否する案もあるが、headless の leader が自分の org を片付けられなくなる。記録を済ませてから閉じれば、台帳は正しく残る(Codex plan advisory の指摘 3)
- workspace を閉じたら `org_workspace_closed` を記録し、そのあとの spawn は新しい workspace を作る。記録しないと、同じ org_id の spawn が閉じた workspace に tab を作ろうとして失敗する(Codex plan advisory の指摘 4)

Critical forks: 止め方の 1 件だけで、ユーザーが決めた。ほかは既定の判断で、1 スライス以内で戻せる

## Acceptance criteria

- [ ] AC1: `ralph org stop --org-id X --seat Y` は、座席の pane に C-c を送ったあと、その pane を閉じる。台帳の `stopped` の詳細に閉じた結果が残る。pane が見つからないときは閉じ済みとして成功する
- [ ] AC2: pane を閉じられなかったとき(herdr のエラーか期限切れ)、`stop` は台帳に `stopped` を書かず、`stop_failed`(詳細つき)を書いて、終了コード 1 と理由を返す。座席は `ralph org status` で動いているまま表示される。herdr が戻ったあと `stop --seat` と `stop --all` を打ち直すと、その座席を閉じて `stopped` を書き、終了コード 0 になる
- [ ] AC3: `ralph org disband --org-id X` は、動いている座席をすべて止めたあと、その org の herdr workspace を閉じ、`org_workspace_closed` と `disbanded` を書く。止められなかった座席か閉じられなかった workspace があれば、`disbanded` を書かず、それを一覧にして終了コード 1。止められなかった座席を「stopped」と表示しない。打ち直すと残りを止めにいく
- [ ] AC4: `ralph org stop --all` は、`--org-id` なしで、全 org の動いている座席をすべて止める。1 つ止められなくても残りを止め、止められなかったものを `<org_id>/<seat_id>` と理由で stderr に並べて終了コード 1。全部止まれば終了コード 0
- [ ] AC5: `ralph org disband --all` は、まだ disband していない全 org を disband する(座席を止め、workspace を閉じ、`disbanded` を書く)。失敗の扱いは AC4 と同じ。disband に失敗した org は、次の `disband --all` でまた対象になる
- [ ] AC6: `--all` と `--org-id`、`stop` の `--all` と `--seat` を同時に指定するとエラーになり、何もしない。`--all` を付けない `stop` と `disband` は、今までどおり `--org-id` が要る
- [ ] AC7: `--all --dry-run` は herdr も agmsg も呼ばず、台帳に dry-run の記録だけを書く
- [ ] AC8: ralph が台帳に記録していない herdr の workspace と pane は閉じない(テストで、記録していない id に閉じる呼び出しが行かないことを確かめる)
- [ ] AC9: `/org` skill の動詞の表・締めの順・完了条件(4 面)、leader の雛形、`README.md`、仕様の FR-2 が新しい動きを書いている。`./scripts/check-skill-sync.sh` と `./scripts/check-sync.sh` が通る
- [ ] AC10: `--force` を付けた `stop` / `disband` は、閉じられなかった座席にも `stopped`(詳細に forced)を書き、`disband` は `disbanded` も書く。失敗は stderr に出し、終了コードは 0
- [ ] AC11: herdr か agmsg の呼び出しが応答しないとき(テストでは、取り消されるまで返らない偽の driver)、1 回の呼び出しは期限で打ち切られ、その座席は「閉じられなかった」になり、`--all` は残りの org の座席を止め、結果の一覧を出して終わる
- [ ] AC12: `HERDR_PANE_ID` か `HERDR_WORKSPACE_ID` が閉じる対象の pane か workspace を指すとき、その pane と workspace を閉じる呼び出しは、ほかのすべての閉じる呼び出しと台帳への記録のあとに行われる(偽の driver の呼び出し順で確かめる)
- [ ] AC13: spawn → disband(workspace を閉じる)→ 同じ org_id で spawn すると、2 回目の spawn は新しい workspace を作る(閉じた workspace の id を使わない)

## Implementation outline

1. S1(driver): `PaneClose` と `WorkspaceClose`、見つからないときのエラーの区別、`HerdrClient` と `fakeHerdr` の追加(AC1・AC3 の土台)
2. S2(1 座席の stop): `Stop` が pane を閉じる、`stop_failed`、`--force` の扱い、呼び出しごとの期限、自分の pane を最後に閉じる(AC1・AC2・AC8・AC10・AC11・AC12 の stop 側)
3. S3(1 org の disband と workspace): `Disband` が workspace を閉じて `org_workspace_closed` を書く、失敗を分ける、自分の workspace を最後に閉じる、`resolveWorkspace` の作り直し(AC3・AC10・AC12・AC13 の disband 側)
4. S4(全 org の動詞): 全 org の止める・disband する関数(AC4・AC5・AC7・AC11 の全 org 側)
5. S5(CLI): `--all` と `--force` のフラグ、同時指定の検査、出力と終了コード、ヘルプ文(AC4〜AC7、AC10)
6. S6(文書と雛形): leader の雛形、`/org` skill、`README.md`、仕様、evidence(AC9)

## Verify plan

- Static analysis checks: `./scripts/run-static-verify.sh`
- Spec compliance criteria to confirm: AC1〜AC13。仕様の FR-2(横断の stop、一部の失敗の一覧と終了コード 1)と、非機能要件の「全 LLM を止めても ralph の仕組みだけで停止できる」
- Documentation drift to check: `git grep -n 'C-c\|disband\b.*report\|herdr workspace / agmsg' -- .claude/skills/org internal/org/prompts docs/specs README.md` で、止め方と締めの順の古い説明が残っていないか
- Evidence to capture: verify のレポート

## Test plan

- Unit tests: driver の `PaneClose` / `WorkspaceClose` の引数と、見つからないときのエラーの区別(偽の runner)。`Stop` の呼び出し順(C-c → close → Leave → 記録)、見つからないときの成功、閉じられないときの `stop_failed` と終了、`--force`。`Disband` の workspace を閉じる処理、`org_workspace_closed`、失敗の分け方。応答しない偽の driver で期限が効くこと。`HERDR_PANE_ID` / `HERDR_WORKSPACE_ID` を設定したときの呼び出し順
- Integration tests: `internal/cli` で `stop --all` / `disband --all` / `--force` の出力と終了コード、同時指定のエラー、`--dry-run`。失敗 → 偽の driver を回復させる → 同じ `--all` を打ち直して回収できること
- Regression tests: `./scripts/run-test.sh`。今の `stop` / `disband` のテストを新しい動きに合わせて直し、意図を弱めていないことをレポートに書く。spawn → disband → 同じ org_id で spawn
- Edge cases: pane_id のない座席、workspace の記録のない org、動いている座席のない org、空の台帳、disband 済みの org を `--all` が飛ばすこと、disband に失敗した org を `--all` が拾い直すこと
- Evidence to capture: test のレポート

## Risks and mitigations

- 止めた座席の画面のログが消える: 閉じる前に読みたいときは `ralph org read` を使う、と `/org` skill に書く
- `disband --all` で全部が止まる: 閉じるのは台帳に記録した pane と workspace だけ(AC8)。ユーザーが自分で作った herdr の workspace は閉じない
- 今の `stop` の動きが変わる: 今までは pane が残っていた。PR の本文と `/org` skill に書く
- headless の leader が自分で disband すると、そのセッションが終わる: 締めの順を「report → disband」に変え、雛形と skill に書く(AC9・AC12)
- herdr が止まっているときは閉じる操作が失敗する: `stop_failed` を書いて終了コード 1 で知らせ、座席は動いているまま残る。片付けたいときは `--force` を使う
- `stop_failed` と `org_workspace_closed` が新しいイベントの種類になる: `Roster` は状態のイベントだけを見るので、座席の状態の判定は変わらない。`ralph org report` の時系列には出る

## Rollout or rollback notes

- バイナリの更新で効く。`/org` skill の文言は `ralph upgrade` で届く
- 戻すときはこの PR を revert する。台帳の形は変えない(イベントの種類と詳細の文字列が増えるだけ)。ただし古いバイナリは `org_workspace_closed` を読まないので、この PR のバイナリで disband した org_id で、古いバイナリから spawn すると、閉じた workspace に tab を作ろうとして失敗する。その場合は別の org_id を使う

## Open questions

- なし

## Progress checklist

- [x] Plan reviewed
- [x] Plan approved
- [x] Branch created
- [x] Implementation started
- [ ] Review artifact created
- [ ] Verification artifact created
- [ ] Test artifact created
- [ ] PR created
- 2026-10-07: consult(plan)は「直してから進める」。閉じられなかった座席に `stopped` を書く最初の案を、`stop_failed` で動いているまま残し `--force` で片付ける形に変えた
- 2026-10-07: Codex plan advisory の 4 件(打ち直しで拾えること、応答しない呼び出し、headless の leader が自分を閉じること、閉じた workspace の使い回し)は、ユーザーが「計画を直す」を選び、Scope・AC・Design decisions・Rollout に反映した。スライスは 6 本にした
- 2026-10-07: S1(0f2e9dfd)driver の close と `IsNotFound`。S2(815f5fc8)Stop: details の C-c の結果は `pane=` から `ctrl_c=` に名前を変えた、閉じられなかったときは agmsg から外さない(座席は動いているので届く状態を保つ)、not-found は driver の型に足した `NotFound()` を org 側の小さなインターフェースで読む(org から driver への import は入れない)。S3(7853d8a9)Disband: 座席が 1 つでも止まらなければ workspace は閉じない、`--force` では閉じられない workspace にも `org_workspace_closed` を書く、自分の workspace は記録を済ませてから最後に閉じる、台帳が読めないときは `disbanded` を書かない、ExecRunner に `WaitDelay` を足した。S4(47f36dad)全 org: disband の対象は「最後の `disbanded` のあとに座席か workspace の記録がある org と、開いた workspace が残る org」。自分の座席・org は、ほかがすべて成功したときだけ最後に処理する
