# 見張りの ALERT を 1 件にまとめ、leader の stall と予約の中の scope_change を上げない(#307 の前半)

- Status: Approved
- Approved: 2026-11-12 sha256:e5a18ef9477a
- Owner: Claude Code
- Date: 2026-11-12
- Related request: https://github.com/yoshpy-dev/ralph/issues/307(下流の運用で 1 日に ALERT が 345 件たまり、対応が要る QUESTION と BLOCKED が埋もれた)
- Related issue: 307
- Type: fix
- Branch: fix/watch-alert-dedupe

## Objective

横断の見張り(`ralph org watch --all`)が受信箱に入れる ALERT を、同じ org・座席・条件ごとに 1 件にまとめる。あわせて、待っているだけの leader の stall と、予約の中だけの変更の scope_change を上げないようにする。issue #307 の受け入れ条件のうち 1〜3 をこの PR で満たす。4(`ralph org wait --inbox` で ALERT を除いて待つ)と通知のまとめは後半の PR(#307b)に回す。

## Scope

- 受信箱の新しい event `recurred` と、件の回数・最後の時刻の畳み込み。`ralph org inbox show` と `--json` での表示
- 見張りの「続いている ALERT」の扱い: 同じ (org, 座席, 条件) の ALERT がまだ片付いていないあいだに条件がまた立ったら、新しい件も新しい agmsg の ALERT も作らず、回数だけを足す
- leader の stall を上げない 2 つの場合: その org に片付いていない QUESTION・BLOCKED・RESULT の件があるとき、leader 以外の座席が手番の中で動いているとき
- 予約を持つ org の scope_change: 予約の外のパスが変わったときだけ上げる。`git status --porcelain=v1 -z --untracked-files=all` の解析(rename の両側)と、前の周期から増えた commit の変更ファイル(`--no-renames` で rename の両側。rebase で入った upstream の commit は除く、D6)。その org の branch が新しく作った計画と報告(`docs/plans/` と `docs/reports/` の下で、既定のブランチとの分岐点に無かったファイル。`templates/` の下は除かない)は除く
- `ralph org watch --help`、`.claude/rules/ralph/agent-messaging.md`(と templates/base の写し)、`docs/specs/2026-10-07-org-multi-org-director.md` の決定の注記、`docs/tech-debt/README.md` の RC3 の行

## Non-goals

- desktop notification のまとめと切る設定、`ralph org wait --inbox --types`、director skill の変更(#307b)
- leader 以外の座席の stall の規則の変更。まとめる仕組みで 1 件に収まるので、規則はそのままにする
- scope_change を座席ごとでなく worktree ごとに 1 回にすること。機能の org の 3 席は 1 つの worktree を共有するので、予約の外の変更は最大 3 件になる(既知の穴として残す)
- 利用上限で止まった座席の `usage_limited`(#312a)、herdr の状態と画面の食い違い(#308)
- 見張りの周期や `stall_minutes` などの既定値の変更

## Assumptions

- 4 時間で 95 件の再発は、herdr の `agent get` の生の文字列が揺れて stall が解除され、約 2 周期後にまた立つことで起きた(`internal/org/watch.go` の `raiseOrClear` と stall の条件から推測。下流の watch-status は見ていない)。見張りが leader に送る ALERT が leader を起こし、揺れを自分で作っていた可能性もある(未確認)。この計画はどちらの場合でも件が増えない形にする
- 受信箱の畳み込みは知らない event を数えずに捨てる(`internal/org/inbox.go` の `apply`)。古い binary は `recurred` を読み飛ばす
- `git status --porcelain=v1` のパスは、作業ディレクトリによらず repo の root からの相対パスになる(git の文書の porcelain の節)。予約のパスも repo の root からの相対パスである(`internal/org/reserve.go` の `NormalizeReservePaths`)
- `.harness/state/` は git が無視するので、porcelain に出ない

## Affected areas

- `internal/org/inbox.go`(`recurred` の event、畳み込み、追記の関数)と `inbox_test.go`
- `internal/org/watch.go`(`raiseOrClear`、`sendAlert`、`escalateToInbox`、`evaluateCycle`・`evaluateSeat` の stall と scope_change、watch-status の記録)と `watch_test.go`・`watch_all_test.go`
- 新しいファイル: `internal/org/watch_scope.go`(porcelain の解析、予約に含まれるかの判定、除くパスの一覧)と `watch_scope_test.go`
- `internal/org/reserve.go`(含まれるかの判定を置くなら。重なりの判定 `reservePathsOverlap` は変えない)
- `internal/cli/org_inbox.go`(件の表示)、`internal/cli/org_watch.go`(help)と、それぞれのテスト
- `.claude/rules/ralph/agent-messaging.md`、`templates/base/.claude/rules/ralph/agent-messaging.md`
- `docs/specs/2026-10-07-org-multi-org-director.md`(FR-3 と ALERT の件の決定に注記)、`docs/tech-debt/README.md`(RC3 の行)

## Visual review

- Page: `.harness/state/plan-visual/watch-alert-dedupe.html`(やさしい図 3 枚: overview、easy-stall、easy-scope。折りたたみの中に detail-overview、detail-episode、detail-stall、detail-scope)
- Lint: `./scripts/plan-visual.sh lint` exit 0
- Self-check: 全体、overview、ほかの 6 枚をそれぞれ `shot` で撮って読んだ。detail-scope の除く一覧の文字が枠からはみ出していたので枠を広げ、detail-overview の文書の群に S5 を書き足し、撮り直して問題がないことを確かめた。計画の助言を反映したあと、easy-scope と detail-scope の除く規則の文を書き直し、lint と撮り直しで確かめた

## Design decisions

Critical forks: None。次の 6 点は、コードと既存の記録から決められるので自律で選んだ(ユーザーの 2026-10-11 の指示「計画は自動承認」に従う)。

- D1 続いている ALERT の単位: 同じ (org, 座席, 条件) について、受信箱に片付いていない(open か acked の)ALERT の件があるか、watch-status にまだ escalate していない pending の ALERT があるあいだを 1 つの「続き」とみなす。続きのあいだに条件がまた立ったら、件には `recurred` の event を足し、pending には回数を足す。新しい件も新しい agmsg の ALERT も作らない。件が resolved になるか、pending が leader の動きで消えたら続きは終わり、次に立ったときは今までどおり新しい ALERT になる。理由: issue の案のとおり。加えて、同じ件について leader へ ALERT を送り続けないので、ALERT が leader を起こして揺れを作る道も断てる
- D2 受信箱の側でも同じ鍵で重ねない: `escalateToInbox` は、同じ (org, 座席, 条件) の片付いていない件があれば、新しい件の代わりに `recurred` を足す。理由: watch を立て直して watch-status の続きの記録を失っても、件が増えないようにする
- D3 leader の stall を上げない場合の 1 つ目は、その org に片付いていない QUESTION・BLOCKED・RESULT の件があるとき。RESULT も入れる。理由: `ralph org escalate` が受け付けるのはこの 3 つで、どれも leader が上げ、人か director の手を待っている印になる(`internal/org/escalate.go` の Escalate)
- D4 2 つ目は、leader 以外の active な座席が手番の中で動いているとき。判定は、その座席の `herdr agent get` の `agent_status` が `working` で、生の文字列が前の周期から変わっていること。TASK の宛先かは見ない。座席は leader の TASK でしか動かないので、動いている座席がいれば leader はそれを待っている(reviewer が gate を流している場合も入る)。herdr が `working` のまま固まった座席(#308)は文字列が変わらないので数えず、leader の stall は今までどおり立つ。そのために、同じ周期で leader 以外の座席を先に評価する
- D5 scope_change の絞り込みは、予約を持つ org すべてに当てる(`ActiveReservation` が空でない org)。予約のない org は今のまま。予約の外でも除くのは、その org の branch が新しく作った記録だけにする。条件は、パスが `docs/plans/` か `docs/reports/` の下にあり、`docs/plans/templates/` と `docs/reports/templates/` の下になく、HEAD と origin の既定のブランチの分岐点(`git merge-base`)の tree に無いこと。役割 prompt が座席に予約の外へ書かせる機能の計画(active から archive への移動を含む)、review・verify・sync-docs の報告、`org-manifest-` の報告はこれに当たる。分岐点にあったファイル(共有のテンプレート、ほかの機能の計画、前からある報告)への変更は、予約の外なら知らせる。分岐点が分からないとき(origin の既定のブランチが無い)は除かずに知らせる側に倒す。`.harness/state/` は git が無視するので扱わない。除く規則が役割 prompt の書かせるパスを覆うことをテストで確かめる(計画の助言の指摘 2 で、`docs/plans/` と `docs/reports/` を丸ごと除く案から絞った)
- D6 予約の外の変更は、作業中の変更(porcelain)と、前の周期の HEAD から今の HEAD までの commit の変更ファイル(`git diff --no-renames --name-only -z <前> <今>`)の両方から集める。`--no-renames` で rename を削除と追加に分け、移動元と移動先の両方を取る(計画の助言の指摘 1: rename を検出すると `--name-only` は移動先だけを返す)。理由: 今の「文字列が変わったら立てる」は、編集して 1 周期のうちに commit した変更も、作業中から clean への変化として拾っていた。作業中の変更だけに絞ると、それを取りこぼす。実装中のずれ(S4 の報告で判明): leader は PR の前に `git rebase origin/<既定>` を打つので、`git diff <前> <今>` には upstream の変更が並び、予約の外の変更として誤って知らせる。そこで commit の変更ファイルは `git log --no-renames --name-only -z --format= <前>..<今> --not <origin の既定のブランチ>` で、origin の既定のブランチに無い commit からだけ取る。既定のブランチが分からないときは今の `git diff` に戻す(知らせる側に倒す)。rebase で積み直された自分の commit が予約の外を触っていれば、もう一度知らせる(続きが開いていれば回数になる)

## Acceptance criteria

- [x] AC1 同じ (org, 座席, 条件) の条件が、その件が open か acked のあいだに何度立っても、受信箱の件は 1 つのままで、立つたびに `recurred` の event が 1 つ増える。`ralph org inbox show <id>` と `ralph org inbox --json` に回数と最後の時刻が出る(issue の受け入れ条件 1)
- [x] AC2 件が resolved になったあとに同じ条件が立つと、新しい件になる
- [x] AC3 続きのあいだに条件がまた立っても、見張りは leader に agmsg の ALERT を送らない
- [x] AC4 leader 以外の座席の ALERT が deadman の期限の前にまた立っても、pending は増えず、期限で escalate した件に回数が残る
- [x] AC5 watch-status に続きの記録が無くても(watch の立て直し)、受信箱に同じ鍵の片付いていない件があれば、新しい件を作らず `recurred` を足す
- [x] AC6 その org に片付いていない QUESTION・BLOCKED・RESULT の件があるあいだ、leader の stall は立たず、受信箱にも入らない(issue の受け入れ条件 2)
- [x] AC7 leader 以外の active な座席が `working` で生の文字列が変わり続けているあいだ、leader の stall は立たない。その座席の文字列が変わらなくなり、ほかに上の場合が無ければ、`stall_minutes` を過ぎた leader の stall は今までどおり立つ
- [x] AC8 予約を持つ org で、予約の中だけの変更と、その org の branch が新しく作った計画と報告(D5)だけの変更(作業中の変更、untracked、rename、commit、計画の active から archive への移動)では scope_change が立たない(issue の受け入れ条件 3)
- [x] AC9 予約を持つ org で、予約の外のパスの変更(作業中、untracked のディレクトリの中のファイル、rename の片側、前の周期のあとの commit)があれば scope_change が立ち、ALERT の本文に予約の外のパスが出る。予約の外から予約の中へのファイルの移動を 1 周期のうちに commit した場合も、移動元が出る(実際の git で確かめる)
- [x] AC10 予約を持たない org の scope_change は今までと同じに立つ
- [x] AC11 除く規則が、役割 prompt(`internal/org/prompts/*.md`)が予約の外に書かせる `docs/` の下のパスを覆うことをテストが確かめる
- [x] AC15 予約を持つ org の worktree で `git rebase origin/<既定>` をして、upstream の commit が予約の外のパスを触っていても、scope_change は立たない(実際の git で確かめる)
- [x] AC14 予約を持つ org で、分岐点にあった `docs/reports/templates/` の下のファイル、ほかの機能の計画、前からある報告を変えたり消したりすると、予約の外なら scope_change が立つ。分岐点が分からないときは、`docs/plans/` と `docs/reports/` の下の変更も予約の外なら立つ
- [x] AC12 新しい欄の無い watch-status と、`recurred` の無い受信箱を、新しい binary が今までどおり読む
- [x] AC13 `ralph org watch --help`、agent-messaging のルール(2 つの写し)、2026-10-07 の仕様の注記、tech-debt の RC3 の行が、この挙動に合う

## Implementation outline

1. 受信箱に `recurred` を足す。`InboxEvent` はそのまま使い、event の名前、件の `Recurrences` と `LastRecurredAt` の畳み込み、`inbox.lock` の下で読んで決めて足す関数(`update` を使う)。状態は変えず、期限(`InboxDeadlineDue`)も読まない。`ralph org inbox show` と `--json` に回数と最後の時刻を出す — verify: `test -z "$(gofmt -l internal/org internal/cli)" && go vet ./internal/org/ ./internal/cli/ && GOFLAGS="-timeout=30m $(go env GOFLAGS)" go test ./internal/org/ ./internal/cli/ -run 'Inbox|Escalate'`
2. 見張りの続きの扱い(D1、D2)。`raiseOrClear` が条件の立ち上がりで、同じ鍵の片付いていない件か pending があるかを見て、あれば回数を足すだけにする。`escalateToInbox` は同じ鍵の片付いていない件に `recurred` を足す。pending の回数は escalate のときに件へ移す — verify: `test -z "$(gofmt -l internal/org)" && go vet ./internal/org/ && GOFLAGS="-timeout=30m $(go env GOFLAGS)" go test ./internal/org/ -run 'Watch|Deadman|Escalat|Inbox'`
3. leader の stall を上げない 2 つの場合(D3、D4)。`evaluateCycle` で leader 以外の座席を先に評価し、その周期の「動いている座席」を集めてから leader の stall を決める。片付いていない件は、その周期で 1 回読んだ受信箱から数える — verify: `test -z "$(gofmt -l internal/org)" && go vet ./internal/org/ && GOFLAGS="-timeout=30m $(go env GOFLAGS)" go test ./internal/org/ -run 'Watch|Stall'`
4. scope_change を予約で絞る(D5、D6)。新しい `watch_scope.go` に、porcelain v1 `-z` の解析、予約に含まれるかの判定(`.`、`/` で終わるディレクトリ、ファイルの完全一致)、その org が新しく作った記録かの判定(分岐点の tree の `docs/plans/` と `docs/reports/` の一覧を HEAD が変わったときだけ取り直す)、commit の変更ファイルの取得(`--no-renames`)を置く。watch-status の座席の記録に前の周期の HEAD と予約の外の変更の署名を持つ。本文には予約の外のパスだけを載せる — verify: `test -z "$(gofmt -l internal/org)" && go vet ./internal/org/ && GOFLAGS="-timeout=30m $(go env GOFLAGS)" go test ./internal/org/ -run 'Watch|Scope|Reserv'`
4b. rebase の誤報を直す(D6 の実装中のずれ、AC15)。commit の変更ファイルを `git log ... <前>..<今> --not <origin の既定のブランチ>` で取り、分からないときは `git diff` に戻す — verify: `test -z "$(gofmt -l internal/org)" && go vet ./internal/org/ && GOFLAGS="-timeout=30m $(go env GOFLAGS)" go test ./internal/org/ -run 'Watch|Scope|Reserv|Porcelain|OwnRecord'`
5. 文書と help。`ralph org watch --help` の pulse の説明、agent-messaging のルールの「Escalation to the org inbox」の節(2 つの写し)、2026-10-07 の仕様の注記、tech-debt の RC3 の行 — verify: `test -z "$(gofmt -l internal/cli)" && go vet ./internal/cli/ && GOFLAGS="-timeout=30m $(go env GOFLAGS)" go test ./internal/cli/ -run 'Watch' && ./scripts/check-sync.sh`

## Verify plan

- Static analysis checks: `./scripts/run-static-verify.sh`(gofmt、go vet、golangci-lint、staticcheck、同期の検査)
- Spec compliance criteria to confirm: AC1〜AC13。issue #307 の受け入れ条件 1〜3 がこの PR で満たされ、4 が #307b に残ることを PR 本文に書く
- Documentation drift to check: `ralph org watch --help` と agent-messaging のルールの ALERT の件の説明、AGENTS.md の internal/org の項(ALERT の件の言い方)、2026-10-07 の仕様の FR-3 と 112 行の決定、tech-debt の RC3
- Evidence to capture: 再発を模したテストの件数(修正前は立つたびに 1 件、修正後は 1 件と `recurred` の数)

## Test plan

- Unit tests:
  - 受信箱: `recurred` の畳み込み(回数、最後の時刻、状態を変えない、resolved の件にも足せるか)、知らない event の扱いが変わらないこと
  - 見張り: 条件が立つ→解除→立つを 5 回くり返し、件が 1 つで `recurred` が 4 つ、agmsg の ALERT が 1 回だけのこと。resolved のあとは新しい件。pending の再発で pending が増えず、escalate した件に回数が出ること
  - leader の stall: 片付いていない QUESTION・BLOCKED・RESULT のそれぞれ、acked、resolved(上げる)。動いている座席がいる/文字列が固まった `working` の座席(上げる)
  - scope_change: porcelain `-z` の解析(変更、追加、削除、untracked、rename、空白や非 ASCII を含むパス)、予約の判定(`.`、ディレクトリ、ファイル、接頭辞だけが同じ別のディレクトリ `src/a` と `src/ab/`)、予約の無い org
  - 除く規則: 分岐点に無い計画と報告(除く)、分岐点にあった `docs/reports/templates/` の下・ほかの機能の計画・前からある報告(除かない)、計画の active から archive への移動(除く)、分岐点が分からないとき(除かない)
  - 実際の git の一時 repo での回帰: 予約の外から予約の中へのファイルの `git mv` を 1 周期のうちに commit し、移動元が本文に出ること(AC9)
  - 除く規則と役割 prompt の照合(AC11)
- Integration tests: `watch_all_test.go` の流れで、leader の stall と scope_change が受信箱に何件入るかを、修正前の形(既存のテストの期待)と照らす
- Regression tests: 既存の deadman、liveness、watch の立て直し(同じ alert_id の件を二重に作らない D4 の決定)のテストがそのまま通ること
- Edge cases: 受信箱が読めないとき(今までどおり受信箱の外へ escalate)、HEAD の無い worktree、前の周期の HEAD が今の HEAD の祖先でないとき(rebase)、予約が `.` の org
- Evidence to capture: `go test ./internal/org/ ./internal/cli/` の結果、テストで数えた件数

## Risks and mitigations

- QUESTION を上げたあとに leader が本当に止まっても、stall は立たない。対策: liveness(`agent get` の失敗)は今までどおり立つ。片付いていない件そのものが受信箱にあり、ack と resolve の期限で人に届く
- 続きのあいだの再発で、新しい情報(scope_change の新しいパス)が見えにくくなる。対策: `recurred` の event に、その回の alert_id を残し、scope_change ではその回の予約の外のパスを `note` に短く残す
- herdr の `agent_status` の読み方が herdr の版で変わる。対策: `resume.go` の `herdrAgentStatus` と同じ読み方を使い、読めなければ「動いていない」とみなす(leader の stall は今までどおり立つ側に倒れる)
- commit の変更ファイルと分岐点の一覧の取得が周期ごとの負荷を増やす。対策: HEAD が変わった周期だけ `git diff`、`git merge-base`、`git ls-tree` を打ち、`callWithTimeout` の中で走らせる
- 分岐点の求め方を誤ると、共有の文書への変更を見逃すか、自分の計画まで知らせる。対策: 分岐点が分からないときは知らせる側に倒し(AC14)、どちらの向きもテストで確かめる
- 3 席が共有する worktree で、予約の外の変更が最大 3 件になる。対策: 既知の穴として PR 本文と tech-debt に書く
- watch-status の形が変わる。対策: 欄の追加だけにし、古いファイルを読むテストを足す(AC12)

## Rollout or rollback notes

- 動いている見張りには、止めて新しい binary で立て直すまで効かない。release のノートに「更新の前に見張りを止める」を入れる(過去の release と同じ)
- 戻すときは PR を revert する。受信箱に残った `recurred` の行は、古い binary が読み飛ばす
- 役割 prompt と skill は変えないので、下流の `ralph upgrade` は要らない(CLI の更新だけで効く)

## Open questions

- なし。下流の watch-status と受信箱の実物は見ていないので、再発の仕組みは推測のまま PR 本文に書く

## Progress checklist

- [x] Plan reviewed — plan advisory: ran with codex。指摘 2 件(どちらも MEDIUM)を計画に反映した。(1) commit 済みの rename の移動元を `git diff --name-only` が落とす → `--no-renames` で両側を取り、実際の git の回帰テストを AC9 に足した(D6)。(2) `docs/plans/` と `docs/reports/` を丸ごと除くと共有のテンプレートやほかの機能の計画への変更まで隠す → 除くのを分岐点に無かった記録だけに絞り、AC14 を足した(D5)
- [x] Plan approved — ユーザーの 2026-10-11 の指示(作った計画は自動承認)に従って承認を記録した
- [x] Branch created
- [x] Implementation started
  - S1 f947db71: 受信箱の `recurred`。show の event 行に `alert_id=` と `count=` を足した。resolve と recurred の note の検査を `validateInboxNoteLine` で共有した
  - S2 e9c49d0c: 続きの扱い(D1、D2、AC4)。既存のテスト 2 本(`TestWatch_Stall_AlertsOnlyNeverStops_RecoversAndRefires`、`TestWatch_EnsureWatchdogJoined_TransientFailure_RetriesUntilSuccess`)は、pending のあいだの再発で 2 通目の ALERT を期待していた。D1 と AC3 でこれは再発の記録に変わるので、期待を「再発の回数が 1 で ALERT は 1 通、leader が動いて pending が消えたあとの stall で 2 通目」に直した。テストの意図(再発で知らせが届くこと、Join の再試行)は変えていない。既知の小さな穴: watch が件を作った直後、pending の回数を件に移す前に止まると、その回数は件に残らない
  - S3 117675ea: 待っている leader の stall(D3、D4)。種類の集合は `escalate.go` の `escalateTypes` を共有した。抑止の始まりの stderr の 1 行は、watch-status に欄が要るので見送った。既存テストの期待の変更なし
  - S4 61fc63f1: 予約での scope_change(D5、D6)。AC11 のテストは、役割 prompt がすでに `docs/evidence/`(gate のログ)、`docs/new/`(git の出力の例)、`docs/recipes/`(DOC_FOLLOWUP の例)を名指ししているので、座席が書かない場所として理由つきの許可の一覧にした。既定のブランチは完全な ref 名で渡す。予約のある org の git には `--no-optional-locks` を付けた。報告で、rebase のあとに upstream の変更を予約の外として知らせる誤報が分かった → D6 にずれの注記を書き、AC15 と 4b を足した(計画の本文が変わったので、図の detail-scope を直し、lint と撮り直しのあと digest e5a18ef9477a で承認を記録し直した)。既知の穴: git が持っていない前の HEAD との差分は取れない、commit の無い repo の最初の commit は porcelain でしか拾えない
  - S4b 34b0fc87: commit の変更ファイルを origin の既定のブランチに無い commit からだけ取る(AC15)。rebase と merge の両方を実際の git で確かめた
  - a650ae2e(orchestrator の直し、1 ファイルの小さな変更): `git log` に `--no-show-signature` を足した。S4b の報告で、利用者の `log.showSignature` が gpg の行をパスとして混ぜうると分かったため。検証コマンド(4b と同じ)は通った。自己点検: 不要な変更・誤字・デバッグの残り・秘密情報・範囲外のファイルとも該当なし
  - S5 48dc5934: help(テストで文を固定)、agent-messaging の 2 つの写し、仕様の FR-3 と FR-6 の注、tech-debt の RC3 を RESOLVED にし、既知の穴の行を足した。報告で、README.md、org と director の skill、`docs/quality/quality-gates.md`、`internal/org/reserve.go` のコメントに scope_change の古い説明が残ると分かった(/sync-docs で判断)
- [x] Review artifact created — `docs/reports/self-review-2026-11-12-watch-alert-dedupe.md`(cycle 1 の 1 回目、27a76047): CRITICAL・HIGH なし、MEDIUM 1・LOW 6
  - S6 f4f1fb93: M1(読まれない `LastRecurredTS` を消した)、L1(commit の一覧の失敗で前の HEAD を据え置く。git が持っていない HEAD なら進める)、L2(立てる条件を「前の周期に無かった予約の外のパスか、予約の外を触る commit」に変えた。外のファイルが減ったときや stage しただけでは立たない。AC9 の「予約の外のパスの変更があれば立つ」は満たす)、L4(再発の仕組みを `internal/org/watch_recurrence.go` に移した)、L5(bool にした)。watch-status の欄の名前を変え、このブランチの前のビルドが書いたファイルでも誤って立たないようにした
  - L3(merge-base と ls-tree を「HEAD が変わった周期だけ」でなく「計画と報告の候補がある周期だけ」打つ): 計画の Risks の文と違うが、候補のある周期は少なく、HEAD が同じでも候補は変わるので、実装のほうを採る。L6(GitStatus と GitRun の 2 つの差し替え口)は doc があるので受け入れる
  - self-review を S6 のあとに回し直す
  - 2 回目(6025f3cb): LOW 4(N1〜N4)。S7 eb596f49 で直した(doc コメントの U+2028 の文字、欄の名前、commit で立てる理由の doc、`1 time`)
  - 3 回目(690e7235): Merge: yes、LOW 1(N5)。`evaluateReservedScope` の doc は、予約の外の commit がいつも続きの回数になるように読めるが、ALERT を立てた直後の周期の commit は `raiseOrClear` の Active で抑えられ、回数にもならない。挙動は変えず、doc の言い過ぎとして tech-debt の既知の穴の行に足す(/sync-docs)。self-review の回し直しはここで止める
- [x] Verification artifact created — `docs/reports/verify-2026-11-12-watch-alert-dedupe.md`(cycle 1。pass。LOW 8 件 V1〜V8、INFO 5 件 I1〜I5。計画の digest は承認の行と同じ)
- [x] Test artifact created — `docs/reports/test-2026-11-12-watch-alert-dedupe.md`(cycle 1。pass。`run-test.sh` rc 0。変異 61 個のうち 57 個が既存のテストで死に、4 個が生き残った。M30・M50 は LOW、M48・M56 は INFO)
  - sync-docs(cycle 1。報告 `docs/reports/sync-docs-2026-11-12-watch-alert-dedupe.md`。文書は `45e84b3e`、tech-debt は `984761f8`、self-review の報告の U+2028 は `0c727338`): V1〜V8 と N5 を処理した。AC の 15 個の箱を `[x]` にした(digest は `e5a18ef9477a` のまま)。`/org` skill の本文(4 つの写し)を変えた。コードは変えていない
  - Rollout の「skill は変えないので `ralph upgrade` は要らない」は、この sync-docs が `/org` skill の文面を足したので、文面については当たらなくなった。挙動は CLI が持つので、CLI の更新だけで効く。下流の skill の文面は `ralph upgrade` でしか新しくならない(文書だけの差)。計画の本文は変えていない
- [ ] PR created
- cycle 1 の cross-review(codex、18bb7424): ACTION_REQUIRED 4(どれも P2)。2026-10-11 の指示どおり聞かずに直し、cycle を 2 に進めた
  - S8 907afde3: (1) leader が動いた周期の再発を古い pending に畳まず、新しい ALERT にする(`leaderActedSinceAlert` を checkDeadman と共有)、(2) `git log` に `--cc` を足し、merge 自身の編集(衝突の解決)を数え、upstream から入る変更は数えない、(3) 指す先の無い `origin/HEAD` を確かめて main・master・`git diff` に落とす、(4) recurred の event に載った alert_id の escalation の打ち直しで何も足さない。D6 の本文の `git log` のコマンドには `--cc` と `origin/HEAD` の確かめが書かれていないが、D6 の意図(org 自身の commit だけを数える)の中の実装の細部として、本文は変えずにここに記録する
  - cycle 2 として self-review から回し直す
- cycle 2 の self-review(f853d973): Merge: yes、cycle 1 の 4 件は直っていると確認、LOW 4(C2-1〜C2-4)
  - S9 5393a316: C2-1 本番では当たらない `recordRecurrence` の打ち直しの確かめを消した(escalation の打ち直しの確かめは残す。pending の alert_id は立て直しても変わらないので当たる)、C2-2 判定の helper を指すコメント、C2-3 テストのコメントの番号、C2-4 引数の整理。実装者は消したテスト 1 本の削除だけ python で行った(リポジトリの規則では Edit を使う)。差分が 32 行の削除だけで U+FFFD が無いことを確かめて受け入れた。残る穴: 再発を足した直後に watch が止まると、立て直した watch は新しい alert_id で再発をもう 1 つ足し、回数が 1 多くなる(tech-debt の既知の穴の行に足す)
- cycle 2 の self-review の 2 回目(d3a34921): Merge: yes、LOW 1(C2b-1)。C2-1〜C2-4 のうち 3 件は直り、C2-3 は 4 件中 1 件(折り返された行の、triage を番号で指すテストのコメント)が残った
  - 3bb1828b(orchestrator の直し、テストのコメントだけの 1 ファイルの変更): C2b-1 のコメントを、番号でなく case を言葉で書く形(ほかの 3 件と同じ)にした。コードは変えていない
- cycle 2 の verify(0a957745): pass。計画の digest は承認の行と同じ。LOW 6 件(V9〜V14)、INFO 5 件(I6〜I10)で、どれも文書と記録のずれ。AC13 は「一部」(仕様の FR-3 の注と RC3 の注に `--cc` が無い)
- cycle 2 の test(fe4eff60): pass。`run-test.sh` は rc 0。S8・S9 の変更への変異 23 個のうち 19 個が既存のテストで死に、cycle 1 で死んだ 57 個も HEAD でまた死んだ。生き残りのうち LOW は M30・M50・C24・C04 で、どれも probe 1 本で死ぬ。穴は tech-debt の既知の穴の行に足した(この sync-docs)
- cycle 2 の sync-docs(報告 `docs/reports/sync-docs-2026-11-12-watch-alert-dedupe.md`。仕様と `/org` skill は `0cc32e6d`、tech-debt は `74e6b1db`): V9〜V14 を処理した。仕様の FR-3 の注と tech-debt の RC3 の注に `--cc`(merge commit 自身の編集を数える)と origin の既定のブランチの決め方を足した。既知の穴の行は「Eight gaps」になり、(7) 再発を足した直後に止まる窓と (8) C24・C04 を足した。`/org` skill(4 つの写し)に、再発の event に載った alert_id の escalation の打ち直しは何も書かず送らないことを 1 文足した。コードは変えていない。計画の本文は変えていない(digest `e5a18ef9477a` のまま)
- cycle 2 の cross-review(codex、7aee125e): ACTION_REQUIRED 1(P2: 利用者の `diff.relative=true` と下のディレクトリの cwd で、commit のパスが cwd からの相対になり、予約の判定を両向きに誤る)。上限 2 に届いたので、2026-10-11 の指示どおり `cycle-count.json` の `cap` を 3 に上げ(cycle の数は 2 のまま)、直して cycle 3 として回し直す
  - S10 ec538952: `git log` と予備の `git diff` に `--no-relative` を足し、`diff.relative=true` と `internal/org` の cwd の実際の git で、予約の中の commit で立たず、外の commit で立つことを確かめた。実装者は最初の 1 か所だけ python で書き換えた(リポジトリの規則では Edit を使う)。U+FFFD が無いことを確かめて受け入れた
- cycle 3 の self-review(65a82b01): Merge: yes、cycle 2 の 1 件は直っていると確認、LOW 2。C3-1 `--no-relative` は git 2.28 以降にしか無く、古い git では commit の一覧が失敗し続けて検知が黙って止まる → S11 e90c85da で、この repo の慣例どおり `-c diff.relative=false` に替えた(スタブの振り分けも `-c` を飛ばすように直し、変異で 2 つのコマンドがそれぞれのテストで落ちることを確かめた)。C3-2(tech-debt の「cap 2 of 2」の文が cap 3 と食い違う)は /sync-docs で直す
- cycle 3 の self-review の 2 回目(521edd5b): Merge: yes、LOW 1(C3-2 の持ち越し)。C3-1 は S11 で直っていると、実際の git 2.49.0 と 2.26.3 で確かめた。C3-2 は /sync-docs で直す
- cycle 3 の verify(10a86e28): pass。計画の digest は承認の行と同じ。LOW 3 件(V15〜V17)、INFO 4 件(I11〜I14)で、どれも文書と記録のずれ。V15 は tech-debt の既知の穴の行の cap の文(C3-2)と cycle 3 の引き継ぎ、V16 は 357 行の数と「the last Go commit」、V17 はこの Progress の行と並び
- cycle 3 の test(24ca3804): pass。`run-test.sh` は rc 0。S10・S11 への変異 5 個はすべて死に、前の cycle で死んだ 43 個も HEAD でまた死んだ。Test gaps 12 と 13 は INFO(新しいテストの 2 つの向きは同じ git の設定から出ること、2.28 ちょうどの git は打っていないこと)
- cycle 3 の sync-docs(報告 `docs/reports/sync-docs-2026-11-12-watch-alert-dedupe.md`。仕様と tech-debt は `baee33db`): V15〜V17 と I11 を処理した。tech-debt の既知の穴の行の「cap 2 of 2」と「cap of 2 runs」を、cap を 3 に上げたことに合わせて直し、cycle 3 の引き継ぎを 1 文足した(Trigger には S10・S11 が `scopeCommittedPaths` で当たり、(5) と (6) は当たらず、(1)〜(3) と (5)〜(8) は払っていない)。357 行の数は最後の Go の commit `e90c85da` のもの(`watch_scope.go` は 467 行)に直した。仕様の FR-3 の注と tech-debt の RC3 の注に、commit のパスは `-c diff.relative=false` で root からの相対になることを足した。この Progress の cycle の行を時の順に並べ直した(「PR created」の箱は元の位置)。I12〜I14 は記録のまま。コードは変えていない。計画の本文は変えていない(digest `e5a18ef9477a` のまま)
- cycle 3 の cross-review(codex、e4b3ecb0): ACTION_REQUIRED 1(P2: watch が件を作った直後、状態の保存前に落ちると古い pending が残り、件が resolve されたあとの再発がそこに畳まれて知らせが失われる。この branch の畳み込みが持ち込んだ黙る向きの穴)。2026-10-11 の指示どおり直し、4 回目として回し直す(`cycle-count.json` は cycle 3、cap 3)。consult(consult-stuck-cycles)の判定: 今回の変更が持ち込んだ黙る向きの穴は窓が狭くても直し、前からある型や回数のずれ(知らせる側に倒れる)は WORTH_CONSIDERING にして tech-debt と PR の Known gaps に残す。落ちうる窓の一覧を tech-debt に置く
  - S12 14bc17a3: 周期の初め、座席の評価の前に、受信箱がすでに持つ pending(件の alert_id か recurred の event の alert_id)を `escalateAlert` で片付ける(`reconcilePendingAlertsWithInbox`)。単に消すと 2026-10-23 の D4 の人への送り直しが走らないので、既存の escalate の再送の判定を通す。受信箱が読めないときは何もしない(知らせる側)。端の変化: 落ちた watch が残した pending は、deadman の前でも周期の初めに再送の判定を通る。既存テストの期待の変更なし。消す pending が持っていた回数は失われる(既知の穴の (2) と同じ向き)
