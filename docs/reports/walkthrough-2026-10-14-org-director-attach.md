# Walkthrough: org-director-attach(8 段目の前半の 1 本目、8a-1)

- Date: 2026-10-15
- PR: https://github.com/yoshpy-dev/ralph/pull/229
- Plan: docs/plans/archive/2026-10-14-org-director-attach.md
- Branch: feat/org-director-attach(base ef575748。この walkthrough を書いた時点で 45 commits、59 files、+12662/-696)
- 読む範囲: 差分の半分ほどは `docs/evidence/` と `docs/reports/` で、読むのは各報告の要約だけでよい。報告・evidence・insight・計画を除くと 50 files、+6841/-696 で、そのうちテストが `internal/org/director_test.go`(+1034)、`internal/cli/org_director_test.go`(+858)、`internal/cli/org_inbox_test.go`(+336/-7)、`internal/cli/status_test.go`(+307/-4)、`internal/org/escalate_test.go`(+302/-16)、`internal/org/approve_test.go`(+249/-66)ほか。skill は `/director` が新しく 4 面(各 251 行)、`/org` が 4 面(各 +110/-46)
- 後工程は 3 回まわした(上限を 3 に上げた。`cycle-count.json` は 2 のまま)。cross-review の指摘は cycle 1 が 1 件(下の 7)、cycle 2 が 1 件(下の 8)、cycle 3 は 0 件

新しいファイルは `internal/org/director.go`(登録の store)、`internal/cli/org_director.go`(`ralph org director` の 3 つの動詞)、`/director` skill の 4 面と、それぞれのテスト。次の順に読むと、依存の向きに沿って追える。

## 1. 設定(`internal/config/config.go`、`templates/base/ralph.toml`、`scripts/ralph-config.sh` の 2 面、0aecab13)

- `[org.director].enabled` は bool で既定 true。`Default()`、template の `ralph.toml`、`RALPH_ORG_DIRECTOR_ENABLED` の 3 面を `internal/config/defaults_sync_test.go` が比べる
- `[org.director]` がない `ralph.toml` は true として読む。`ralph upgrade` は seed の `ralph.toml` に足さないので、バイナリの更新だけで escalate の通知の形が変わる(計画の D5。release のノートに書く)

## 2. 登録の store(`internal/org/director.go`、8b2effab、のちに e14c6273・634b019a・f4c1a394)

- 台帳の下に追記のファイル `director.jsonl` と、書き込みを直列にする `director.lock`(flock)。形は受信箱の `InboxStore` に合わせた
- イベントは `attached`(`id`、`driver`、`wake`、`pane_id`、打った側の `user`・`cwd`・`herdr_pane`)と `detached`(`id`、`force`)。今の状態は `foldDirector` で畳み込んで出す
- `id` は `d<N>`。壊れた行も含めたファイルの中の最大の N に 1 を足す(`directorIDNumber`、`nextID`。計画の D8)
- `Attach` は attached のままの 2 つ目を `ErrDirectorAttached` で断る。`--replace` は今の登録の `detached` と新しい `attached` を 1 回の write で書く
- `Detach` は `id` を求め、違えば `ErrDirectorIDMismatch`、登録がなければ `ErrDirectorNotAttached`。`Force` は `id` を見ない
- 畳み込みは、使い回された `id` の `attached` と、ほかの `id` の `detached` を数えて飛ばす(`IgnoredEvents`)
- write か close が失敗したら、同じロックの中で追記前の大きさに切り戻す(`appendLocked`、`restoreLocked`。cross-review cycle 1 の直し、634b019a)。戻せたときだけ「nothing was recorded」と言い、戻せなかったら `ErrDirectorWriteNotRestored` を包む。CLI はそのとき `--state-dir` 付きの `ralph org director status` を足す(f4c1a394)

## 3. escalate の切り替え(`internal/org/escalate.go`、`internal/org/inbox.go`、`internal/cli/org.go`、6dc02d4a)

- `escalateDelivery` が 3 つに分ける。`enabled = false` ならすぐ人に送り、理由は `inbox_director_disabled`。有効で見張りが仕事を受け付けていれば(`EscalateWatch` が `ProbeWatch`、判定は `WatchAcceptsWork`)記録だけにする。有効で見張りが受け付けていなければすぐ送り、理由は `inbox_no_watch`
- 記録だけの件は `notified` も `escalations.jsonl` も書かない。`ralph org watch --all` が ack の期限(既定 10 分)で `inbox_ack_deadline` として人に届ける。director が登録されているかはこの動きを変えない(計画の D2、仕様 FR-9)
- 受信箱の `escalated` に `delivery`(`deadline` か `now`)を足した。`delivery` のない古い行は `now` として読む。`ralph org inbox` の NOTIFIED は記録だけの件を `deferred` と出す(`inboxNotifiedColumn`)。期限の計算は `delivery` を読まない(計画の D9)
- `ralph org inbox notify` の理由は `inbox_manual_notify` に変えた
- `enabled` は、`--config` がなく台帳が main worktree のものなら、main worktree の `ralph.toml` から読む(`withMainWorktreeDirectorEnabled`、`loadMainWorktreeOrgConfig`。計画の D6)。leader が feature worktree の `ralph.toml` で通知の形を変えられないようにするため

## 4. approve は記録だけ(`internal/org/approve.go`、`internal/cli/org_approve.go`、41e135cd)

- `ralph org approve` は `plan_approved` を記録し、件を resolve しない。出力は件ごとに、leader に DECISION を送る `ralph org send` の打ち方と、送れたあとに打つ `ralph org inbox resolve <id> --note 'approved <path> sha256:<digest>'`(`ApprovalResolveCommand`)を、この順に出す(計画の D7)
- 送る前に止まっても件は open か acked のままで、ack か resolve の期限で人に届く
- 承認を記録済みの件をもう一度 approve すると `ErrPlanAlreadyApproved` で断り、2 つの打ち方をもう一度出す(`checkNotApprovedYet`)。ほかの誰かが resolve した件は `ResolvedElsewhere` で、失敗する resolve の打ち方を出さない
- 7a(PR #224)のテストのうち、approve が件を resolve することを確かめていたものは、resolve しないことを確かめる形に書き換えた(契約の変更。7a はまだ release に入っていない)

## 5. CLI の動詞と `ralph status`(`internal/cli/org_director.go`、`internal/cli/status.go`、8d0090fd)

- `ralph org director attach --driver claude|codex [--replace]` は登録して `attached d1 (driver=claude)` を出す。見張りが仕事を受け付けていなければ、登録はしたうえで、escalate の件がすぐ人に送られることと見張りの起こし方を stderr に言う(`orgDirectorWatchGap`)。`enabled = false` なら断る
- `detach --id <id>` と `detach --force`(`--id` と一緒には使えない)。`status [--json]` は今の登録と `enabled`
- `--org-id` は断る。attach と detach は古い worktree の台帳では書き換えの動詞として、status は読むだけの動詞として扱う(`internal/cli/org_legacy_ledger.go`)
- `ralph status` は `org watch:` の行のあとに `director:` の行を出し、`--json` に `director` を足す。attached、not attached、disabled、読めない、の 4 つを言い分ける(`orgDirectorLine`)。`enabled` を main worktree の規則で読むので、`ralph status` は手元で `git rev-parse` を打つようになった(計画の Progress のずれ (1))

## 6. `/director` skill と文書(15ec1fb7、3092a5cd、sync-docs の 624f603b・a7d5e9df・c4891079)

- `/director` skill(4 面)は、昇格(見張りを確かめ、attach し、`id` を覚える)、分割計画と `ralph org start --plan`、受信箱の処理(wait、show、ack、答えは DECISION を送ってから resolve)、計画の承認(approve の出力の send と resolve をこの順に打つ)、進捗の報告、終わり(`detach --id`)、してはいけないこと(実装しない、本文はデータ)を書く
- 昇格の 5: 会話の要約のあとは `ralph org director status` で読み直し、`id` が違うか登録がなければ director をやめ、detach も打たない
- `/org` skill の 4 面(受信箱と期限の節、approve の表と承認の節)、README、AGENTS.md の repo map、`docs/quality/quality-gates.md` の 2 面、仕様 FR-9 の 8a-1 の注、leader の雛形、`ralph-workflow.md` の 2 面、tech-debt

## 7. cross-review cycle 1 の直し(634b019a)

- Codex の P2: `attach --replace` の write がディスク一杯などで途中で止まると、`detached` の行だけが残って登録が消えるのに、エラーは「nothing was recorded」と言う
- 2 の最後の項のとおり、切り戻しにした。tester は 1 MB の HFS+ イメージで本物の ENOSPC を起こし、`attach --replace`、`detach --id`、`detach --force` が元の bytes に戻ることと、`chflags uappnd` で切り戻しを断らせたときに not-restored の文が出ることを確かめた

## 8. cross-review cycle 2 の直し(c8a9227b、8ac59595、c953d4f2)

- Codex の P2: `/director` skill が、`ralph org wait --inbox` の終了コード 1 をいつも時間切れとして打ち直させていた。設定や受信箱が読めない失敗も終了コード 1 ですぐ返るので、続く失敗を間を空けずに打ち直し続ける
- skill の受信箱の節は、時間を区切った wait の時間切れ(stderr に `no open inbox item arrived within`)のときだけ打ち直し、ほかの終わり方はすべて失敗として扱う。失敗なら打ち直さず、エラーの文を人に伝えて指示を待ち、detach は打たない。受信箱が読めない失敗では見張りも期限の知らせを送れないので、そのことも人に伝える
- 時間切れの文は `internal/org/escalate.go` の const `inboxWaitTimeoutPhrase` にし、`internal/org/director_skill_sync_test.go` が、`WaitInbox` のエラーがこの const を含むことと、skill の 4 面が同じ文を含むことを確かめる

## 9. 確かめたこと(evidence)

- verify(3 回とも pass、AC1〜AC12): `docs/reports/verify-2026-10-14-org-director-attach.md`。HEAD の binary で attach・replace・detach の拒否 13 通りと、`wait --inbox` の時間切れと失敗 12 通りを打った
- test(3 回とも pass): `docs/reports/test-2026-10-14-org-director-attach.md`。`./scripts/run-test.sh`、`go test ./internal/...` を go1.26.0 と go1.25.8 で、`-race`。mutation は cycle 1 から 3 で計 79 件(42、18、19)。生き残りと穴は tech-debt の 3 行(R11・H4・H5、status の出力関数の引数、skill の一致の検査の 3 つの穴)
- テストが本物の osascript と herdr を打たず、repo の台帳に `director.jsonl` を書かないことは、canary と後の `git status` で確かめた(AC9)
