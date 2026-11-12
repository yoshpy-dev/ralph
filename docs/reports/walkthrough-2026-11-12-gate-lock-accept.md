# Walkthrough: gate を台帳で 1 本ずつ流し、負荷を残し、受け入れた失敗を merge できるようにする(gate-lock-accept)

- 計画: `docs/plans/archive/2026-11-12-gate-lock-accept.md`(PR の作成時に archive へ移す)
- issue: #309 の受け入れ条件 1〜3 をこの PR で満たす
- 差分: `git diff d2e5c6e9...HEAD`(79 ファイル、約 +7,100 / −550。テストを除く Go は約 +1,900 / −210、Go のテストは約 +3,550 / −120、残りは skill・help・文書・記録)

テストと skill の文が差分の大半を占める。次の順に読むと、仕組みを追いやすい。

## 1. gate.lock(S1: lockfile.go、gate_lock.go、gate.go、regate.go、gate_run.go)

- `lockfile.go` の `waitHeldFileLock` は、時間の上限のない flock の待ちで、ctx で切れる。空いているロックは ctx を見ずに取る。
- `gate_lock.go` は台帳の state dir の `gate.lock` を扱う。取ったら印(動詞、org_id か branch、取った時刻、pid)を JSON 1 行で書き、離すときは印を空にしてから LOCK_UN する。待つ間は stderr に、待ちの始め、保持者が変わったとき、5 分ごと、取れたときに 1 行ずつ出す。保持者の pid がもう無ければ、「その gate が起動した script がまだロックを持っている」と書く(D8)。
- `Gate`(gate.go)は、今の検査で早く断ってから gate.lock を待ち、取ったら manifest を読み直して `gateOrgLife` と `gateCheckWorktree` をやり直す(D1)。待つ間に HEAD が動けば新しい HEAD で走り、disband されていれば scripts を走らせずに断る。`recordGate` と `sameFeature` の呼び方は変えていない。
- `Regate`(regate.go)は、local の検査のときだけ、引数の確かめのあと regate.lock より前に gate.lock を取る。待つ間 regate.lock は空いているので、merge と cleanup は断られない。最初の検査が通ったら印に org_id を書き足す。ci の regate は gate.lock を取らない。
- `runGateScript`(gate_run.go)は gate.lock の開いた file を `ExtraFiles` で script の fd 3 に渡す。親の ralph だけが SIGKILL されても、script が終わるまでロックが残る。
- テスト: `gate_lock_test.go`(待ち、行、ctx、印、本物の process を SIGKILL する AC14)、`gate_lock_gate_test.go`(AC1〜AC3、2 本の Gate が重ならないこと)、`regate_test.go` の AC4。

## 2. 負荷と max_load(S2: gate_load.go、gate_run.go、gate_record.go、config.go)

- `gate_load.go` は load average を読む。darwin は `/usr/sbin/sysctl -n vm.loadavg`(`LC_ALL=C`)、linux は `/proc/loadavg` を使う。scripts の間は goroutine 1 本が 30 秒ごとに 1 分値を読んで最大を覚え、scripts が終われば必ず止まる(D4)。
- `[org.gate].max_load`(整数、既定 0 = 切)が 0 より大きいとき、gate.lock を取ったあと 1 分値が上限を超えていれば 30 秒ごとに読み直して待つ。30 分たっても超えていれば始め、stderr と log に書く(D2)。負荷が読めなければ待たない。local の regate は gate.lock のすぐあと、regate.lock の前で待つ。
- gate の log の見出しに始めの値(1・5・15 分)と CPU 数、結びに終わりの値と scripts の間の最大が入る。`gate_passed` と `gate_failed` の Details には `load_start`・`load_max`・`load_end`・`ncpu`(と、`max_load` が 0 より大きいときは `max_load`)が `log=` の前に入る。古い binary は知らない鍵を読み飛ばす。
- CLI の `ralph org gate` と `ralph org regate` は `LOG:` の後ろに `LOAD:` の行を出す。
- テスト: `gate_load_test.go`(2 つの形の読み取り、最大、待ち、上限、ctx)。Details を全文で比べる既存のテストは、`main_test.go` で固定した負荷に合わせて期待値を直した。

## 3. 受け入れ(S3: gate_accept.go、mergeorder.go、merge.go、merge_run.go)

- `ralph org gate accept --branch <b> --log <path> --reason <text>` は、manifest lock の中で台帳を読み、枝の最新の gate の記録が読める失敗(static_failed、test_failed、ci_failed)で、その log が `--log` と同じときだけ、同じ lock のうちに `gate_accepted` を追記する(D5)。人の承認を待つ間に gate が打ち直されて最新の失敗が変わっていれば、新しい log を名指して断る。座席の pane からは、approve と共有した `seatPaneTripwire` で断る。
- `collectMergeTargets` は、最新の gate の記録の後ろにある、head・base・base_ref・branch・event・reason・ts・log・org_id がすべて一致する `gate_accepted` を枝に付ける。gate の記録が新しく付くたびに外す。`classifyMergeTarget` は、受け入れの付いた失敗を通った gate と同じに分類し、`ready` になるところを `accepted` にする。`regate` のときは、受け入れがその head にしか効かないことを detail に書く。
- `ralph org merge` は `ready` か `accepted` で進む。PR の head、checks、保護、origin の読み直しの確かめは変えていない。出力に `GATE: accepted` の行が出て、`pr_merged` の末尾に `gate=accepted` が付く。
- `leaderActivityEventCount` は `gate_accepted` を数えない(D6、コメントとテスト)。
- テスト: `gate_accept_test.go`(記録と断り、打ち直しのあとの log の不一致、同時の 2 つの accept)、`mergeorder_test.go`、`merge_test.go`、`merge_run_test.go`。

## 4. skill、prompt、help、文書(S4、S5、sync-docs)

- director skill の「gate の失敗の受け入れ」節: accept を打つのは人か、人がはっきり受け入れると答えたときの director(D3)。聞く前に log の落ちた job、変更の外か、負荷、時間切れを確かめる。受け入れたら leader に DECISION を送る。auto_merge の手順は `accepted` も merge する。
- org skill の動詞表(`gate`、`regate`、`gate accept`、`merge-order`、`merge`)、reviewer の役割 prompt(ほかの org の gate を待つこと、Codex の timeout、`GATE: fail` の本文に `LOAD:`)、各 help。
- AGENTS.md の internal/org の行、仕様の FR-7 の注記、README、quality-gates、`templates/base/ralph.toml`、tech-debt の行(この PR の穴 6 行と、条件に当たった既存の行の持ち越し)。

## 5. 記録

- self-review、verify、test、sync-docs の report と cross-review の triage(所見なし)、insight event。
