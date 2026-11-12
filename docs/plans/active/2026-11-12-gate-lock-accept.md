# gate を台帳ごとに 1 本ずつ流し、負荷を残し、director が受け入れた失敗を記録する(#309)

- Status: Approved
- Approved: 2026-11-12 sha256:1d8736effdd9
- Owner: Claude Code
- Date: 2026-11-12
- Related request: https://github.com/yoshpy-dev/ralph/issues/309(下流で feature org の gate が同時に流れ、負荷で差分の外の job が落ちた。director が受け入れた失敗を台帳に残せず、merge-order が failed を出し続けた)
- Related issue: 309
- Type: feat
- Branch: feat/gate-lock-accept

## Objective

`ralph org gate` と、scripts を走らせる `ralph org regate` を、台帳ごとに 1 本ずつ流す。gate の間の機械の負荷(load average)を log と記録に残し、任意の設定 `[org.gate].max_load` で負荷が高いときは始めるのを待たせる。director が人の承認を得て gate の失敗を受け入れたことを `ralph org gate accept` で台帳に記録し、`ralph org merge-order` は `accepted` として出し、`ralph org merge` はそれを受け付ける。issue #309 の受け入れ条件 1〜3 をこの PR で満たす。

## Scope

- 台帳の state dir の `gate.lock`(新設)。`ralph org gate` と、`[org.gate].regate_checks = "local"` の `ralph org regate` が持つ。待ちは ctx で切れ、待つ間は stderr に、ロックを持つ gate の動詞・org(regate は branch)・開始の時刻を出す
- ctx で切れる、上限のない flock の待ち(`internal/org/lockfile.go`)
- load average の読み取り(darwin は `sysctl -n vm.loadavg`、linux は `/proc/loadavg`)と、scripts の間 30 秒ごとの 1 分値の最大。gate の log の見出しと結び、`gate_passed` と `gate_failed` の Details(`log=` の前)、CLI の出力の `LOAD:` の行
- `[org.gate].max_load`(整数、既定 0 = 切)。config、検証、既定値のテスト、`templates/base/ralph.toml` の注釈
- `ralph org gate accept --branch <b> --log <path> --reason <text>` と、org-level の新しい event `gate_accepted`(`--log` は受け入れる失敗の gate log。D5)
- merge-order の新しい class `accepted`、JSON の `gate.accepted`、merge が `accepted` を受け付けること、merge の出力と `pr_merged` の `gate=accepted`
- director skill と org skill(それぞれ 4 つの写し)、reviewer の役割 prompt の gate の TASK の節、`ralph org gate` の help、AGENTS.md の internal/org の行、`docs/specs/2026-10-07-org-multi-org-director.md` の FR-7 の注記、`docs/tech-debt/README.md`

## Non-goals

- gate の job ごとの負荷(job は repo の scripts の中にあり、ralph からは見えない)
- gate の列の順番の保証(flock は先着順を約束しない。待つ gate が 3 本以上のとき、後から来た gate が先に取ることがある)
- `[org.gate].regate_checks = "ci"` の regate の直列化(scripts を走らせないので gate.lock を取らない)
- 座席の shell で解決されるツールの版を残すこと(#311、この PR の merge のあと)
- director が人に聞かずに accept を打つ運用(D3。PR 本文で緩める選択肢として示す)
- `ralph org gate` の結果の形(`GATE:`・`HEAD:`・`BASE:`・`LOG:`)の変更。`LOAD:` の行を足すだけ

## Assumptions

- 座席の codex は sandbox で state dir にしか書けない(`internal/org/permissions.go` の `codexWritableRootArgs`)。gate.lock は state dir に置くので、codex の reviewer も取れる
- gate の記録の Details を古い binary が読むとき、知らない鍵は読み飛ばされる(`internal/org/gate_record.go` の `gateRecordFromDetails`)。値に空白を入れなければ、新しい鍵は古い binary の読み取りを壊さない
- `pr_merged` の Details の読み手(`internal/org/followup.go` の `recordedMergeBranch` と `recordedMergeHead`)は、最初の `branch=` と最初の `head=` だけを読む。末尾に鍵を足しても読み取りは変わらない
- `ralph config get` が出せる値は bool・int・string だけ(`internal/cli/config.go` の `configValue`)。`max_load` は整数にする
- 下流の 7 件の gate_failed は、機能の org が走っている間に reviewer 座席が打った `ralph org gate` の記録(issue の本文の「director は push を許した」から推測。台帳は見ていない)。accept はこの記録に効く

## Affected areas

- `internal/org/lockfile.go`: ctx で切れる上限のない待ち
- `internal/org/gate_lock.go`(新): gate.lock と、その中身(保持者の印)、待つ間の stderr の行
- `internal/org/gate_load.go`(新): load average の読み取り、30 秒ごとの最大、max_load の待ち
- `internal/org/gate.go`、`internal/org/gate_run.go`、`internal/org/gate_record.go`: ロックの取り方、待ったあとの検査のやり直し、log の行、記録の鍵
- `internal/org/regate.go`: local の regate が gate.lock を regate.lock の前に取る
- `internal/org/gate_accept.go`(新): `GateAccept` と `gate_accepted` の Details
- `internal/org/approve.go`: 座席の pane を断る確かめを、approve と accept で共有する形にする
- `internal/org/seat.go`: `EventGateAccepted` と、その説明
- `internal/org/mergeorder.go`: accept を枝に結ぶ、class `accepted`、JSON
- `internal/org/merge.go`、`internal/org/merge_run.go`: `accepted` を受け付ける、`pr_merged` の `gate=accepted`
- `internal/org/watch.go`: `leaderActivityEventCount` の説明(数えない理由)
- `internal/org/prompts/reviewer.md`: gate の TASK で、ほかの org の gate を待つことがある旨
- `internal/config/config.go`、`internal/config/config_test.go`、`internal/config/defaults_sync_test.go`、`templates/base/ralph.toml`
- `internal/cli/org_gate.go`、`internal/cli/org_gate_accept.go`(新)、`internal/cli/org_regate.go`、`internal/cli/org_merge.go`、`internal/cli/org_merge_order.go`
- `.claude/skills/director/SKILL.md`、`.claude/skills/org/SKILL.md` と、それぞれの `.agents/skills/` と `templates/base/` の写し
- `AGENTS.md`、`docs/specs/2026-10-07-org-multi-org-director.md`、`docs/tech-debt/README.md`

## Visual review

- ページ: `.harness/state/plan-visual/gate-lock-accept.html`(やさしい図 2 枚: overview、easy-accept。折りたたみの中に詳しい図 4 枚: detail-overview、detail-locks、detail-records、detail-classify)
- lint: `./scripts/plan-visual.sh lint` が exit 0
- self-check: 全体と 6 枚の図をそれぞれ `shot` で撮って読んだ。ロックの順の図で矢印のラベルが矢じりに重なっていたので、ラベルを矢印の上に移して撮り直した。ほかは重なり・はみ出しなし

## Design decisions

critical forks は 4 つあり、どれも consult(consult-plan-gatelock)に聞いて推す案を自律で選んだ。ユーザーの 2026-11-12 の指示(計画の分かれ目も自分で選び、Design decisions に「自律で選んだ」と理由を書く)に従う。

- D1 ロックの置き方と順(自律で選んだ): Gate は今の検査で早く断ってから gate.lock を待ち、取ったら org の読み取り(`gateOrgLife`)と worktree の検査(`gateCheckWorktree`)をやり直して、その時点の HEAD で scripts を走らせる。待つ間に HEAD が動いたことによる head_moved や、待つ間に disband された org の scripts を 15 分走らせてから断ることを避けるため。local の regate は、引数の確かめのあと、regate.lock の前に gate.lock を取る。regate.lock を持ったまま gate.lock を待つと、待つ間の merge・cleanup・ほかの regate が 1 秒で断られ、director skill は断られたブランチを打ち直さない決まりなので止まる。順は gate.lock → regate.lock → manifest.lock で、Gate は gate.lock(と記録の manifest.lock)だけ、merge と cleanup は regate.lock だけを取るので循環はない。代わりに、scripts を走らせない regate(up_to_date、push_only)も前の gate を待つ。
- D2 max_load を超えたまま(自律で選んだ): 30 秒ごとに読み直して最大 30 分待ち、それでも超えていれば始める。そのことを stderr と log に書き、記録に `max_load=<値>` を残す(`load_start` と比べれば超えて始めたと分かる)。無期限に待つと台帳の全 gate が止まり、断ると leader の打ち直しが増えるだけになる。負荷が座席そのものから来るとき(issue の下流がそう)は、待っても下がらない。待ちは ctx で切れる。負荷が読めなければ待たずに始め、stderr に書く。
- D3 accept を打つ人(自律で選んだ): 人か、人がはっきり受け入れると答えたときの director が打つ(`ralph org approve` と同じ決まり)。既定の `auto_merge = true` では、merge は accepted を受け付ける(ユーザーの決定)ので、accept が merge と同じ意味になる。issue の症状は director が自分の判断で通したことで、`--reason` の文は証拠にならない。止めるのは skill の文で、`ralph org gate accept` は座席の pane だけを断る(herdr の中の headless の director は通る。approve と同じ)。PR 本文に、緩める選択肢(director が log の負荷と落ちた job を確かめたら自分で打つ)を書く。
- D4 途中の負荷(自律で選んだ): scripts の間、30 秒ごとに 1 分値を読み、最大を `load_max` に残す。issue の 80〜157 は途中の値で、始めと終わりだけでは見えないことがある。読み取りの goroutine は 1 本で、ticker と done のチャネルで閉じ、scripts が終わったら必ず止める。
- D5 accept が効く範囲: accept は、枝の最新の gate 記録が読める `gate_failed` か `gate_ci_failed` で、後ろに壊れた行がなく、reason が `static_failed`・`test_failed`・`ci_failed` のときだけ記録する。`head_moved` と `tree_changed` は scripts が通っているので「gate を打ち直す」で断る。人が見て受け入れると答えた失敗を取り違えないよう、`--log <path>` を必須にする(計画助言の HIGH)。gate の log は 1 回ごとに O_EXCL で作る別のファイルなので、失敗を一意に指す。accept は manifest lock の中で台帳を読み、枝の最新の gate 記録の log が `--log` と同じときだけ、同じ lock のうちに追記する。人の承認を待つ間に gate が打ち直されて別の失敗が最新になっていれば断り、新しい失敗の log を見て聞き直すよう言う。merge-order の JSON の `gate` に `log` を足し、`ralph org gate` が書いた失敗の detail にも log を出す。記録は失敗の記録の head・base・base_ref・branch・event・reason・ts・log に結ぶ。merge-order は、最新の gate 記録が失敗で、その後ろに結びが一致する `gate_accepted` があるとき、`classifyGatePassed` と同じ判定をし、`ready` になるところを `accepted` にする(`rebase`・`regate`・`no_remote`・`wrong_base` はそのまま)。後ろに新しい gate 記録が来れば accept は効かない(regate で head が変われば消える、を満たす。head が同じでも、gate を打ち直せば消える)。regate の merge 経路の失敗の head は戻された merge commit で、origin に無いので、accept しても `regate` のままになる。merge-order はその detail で、accept はその head にしか効かないと言う。
- D6 `leaderActivityEventCount` は `gate_accepted` を数えない。leader ではなく人か director が打つ記録で、数えると保留中の ALERT の基準線が upgrade と rollback の前後でずれる(gate_ci_* を数えない理由と同じ)。
- D7 `max_load` は整数(`ralph config get` が小数を出せないため)。0 は切、負は設定エラー。
- D8 gate.lock を scripts に継がせる(計画助言の MEDIUM): scripts は自分のプロセスグループで動き、止めるのは親の役目なので、親の ralph だけが SIGKILL で死ぬと、scripts が走ったままロックが外れて次の gate と重なる。gate.lock の開いた file を scripts に `ExtraFiles` で渡し、scripts とその子が fd を持つ間は flock が残るようにする(flock は開いた file description に付き、複製した fd の 1 つでも開いていれば残る)。正常の終わりでは親が LOCK_UN を打つので、group の外に逃げた子が fd を持ち続けても、ロックは外れる(LOCK_UN は同じ file description の lock を外す)。待つ側の行は、保持者の pid がもう無いとき「その gate の scripts がまだロックを持っている」と言う。

## Acceptance criteria

- [x] AC1 同じ台帳で 2 つの `ralph org gate` が同時に流れると、後のほうは前の gate の scripts と記録が終わるまで scripts を始めない。待つ間、stderr に前の gate の動詞・org・開始の時刻が出て、後の gate の log の見出しに待った時間と前の gate が出る(issue の受け入れ条件 1)
- [x] AC2 gate.lock の待ちを ctx で切ると(SIGINT など)、何も記録せず、log も作らない
- [x] AC3 待ったあとの Gate は org と worktree を読み直す。待つ間に HEAD が動けば新しい HEAD で走り、head_moved にならない。待つ間に org が disband されれば、scripts を走らせずに断る
- [x] AC4 local の `ralph org regate` は gate.lock を regate.lock より前に取る。gate.lock を待つ間、`ralph org merge` と cleanup は regate.lock で断られない。ci の regate は gate.lock を取らない
- [x] AC5 gate の log の見出しに始めの load average(1・5・15 分値)と CPU 数、結びに終わりの値と scripts の間の 1 分値の最大が出る。`gate_passed` と `gate_failed`(gate と local の regate)の Details に `load_start`・`load_max`・`load_end`・`ncpu` が `log=` の前に入り、`gateRecordFromDetails` が読める。負荷が読めなければ値は `unknown`。新しい鍵の無い前の形の Details も今までどおり読める(issue の受け入れ条件 2)
- [x] AC6 `[org.gate].max_load` は既定 0 で待たない。負の値は設定エラー。0 より大きいとき、ロックを取ったあと 1 分値が上限を超えていれば 30 秒ごとに読み直して待ち、stderr にそう出す。30 分たっても超えていれば始め、stderr と log にそう書き、記録に `max_load=<値>` が入る。待ちは ctx で切れる
- [x] AC7 `ralph org gate accept --branch <b> --log <path> --reason <text>` は、manifest lock の中で、枝の最新の gate 記録が読める失敗(D5 の reason)で、その log が `--log` と同じとき、同じ lock のうちに `gate_accepted` を 1 件記録し、記録の head・base・base_ref・branch・event・reason・ts・log と、by・herdr_pane・理由の文を Details に持つ。断るのは: 枝の記録が無い、最新が失敗でない、読めない、後ろに壊れた行がある、reason が head_moved か tree_changed、最新の失敗の log が `--log` と違う(打ち直された別の失敗)、同じ失敗がもう受け入れ済み、`--log` か `--reason` が空、`--org-id` が付いている、HERDR_PANE_ID が稼働中の座席の pane のとき。断りは何も書かない。同じ失敗への 2 つの accept が同時に走っても記録は 1 件
- [x] AC14 gate.lock は scripts に継がれる。gate の親の process だけを SIGKILL しても、走っている script が終わるまで gate.lock は持たれたままで、次の gate はその script と重ならない(実際の process で確かめる)。正常の終わりでは、script が残した子が fd を持っていても、親が離せば次の gate が取れる
- [x] AC8 merge-order は、受け入れた失敗の head が origin の枝の tip で、base が origin の既定のブランチの tip のとき、その枝を `accepted` として `ready` の次に並べ、detail に失敗の reason、受け入れた人、時刻、理由の文を出す。JSON の `gate` に `log` と `accepted`(by・ts・reason)が入る。accept の後ろに新しい gate 記録があれば accept は効かない。via=regate で head が origin に無い記録は `regate` のままで、detail が accept はその head にしか効かないと言う(issue の受け入れ条件 3)
- [x] AC9 `ralph org merge` は `accepted` の枝を `ready` と同じく merge まで進める。PR の head が gate の head であること、checks、保護、origin の読み直しの確かめは変わらない。出力に受け入れの行が出て、`pr_merged` の Details の末尾に `gate=accepted` が入る
- [x] AC10 `leaderActivityEventCount` は `gate_accepted` を数えない(テストで確かめる)
- [x] AC11 `config.Default()` の `max_load`、`templates/base/ralph.toml` の注釈、`defaults_sync_test.go` が一致する
- [x] AC12 director skill(4 つの写し)が、accept は人か人がはっきり答えたときの director が打つこと、打つ前に確かめること、`auto_merge` の手順で `accepted` も merge すること、gate が前の gate を待つことを書く。org skill(4 つの写し)の動詞リファレンスに `gate accept`、gate.lock の待ち、`max_load`、`accepted` が入る。reviewer の役割 prompt の gate の TASK の節が、ほかの org の gate を待つことと、Codex の前面の timeout をそのぶん長くすることを書く。`./scripts/check-skill-sync.sh` と `./scripts/check-sync.sh` が通る
- [x] AC13 `ralph org gate --help` と `ralph org gate accept --help` が、待ち、負荷、max_load、accept を書く。AGENTS.md の internal/org の行、仕様の FR-7 の注記、tech-debt の行がこの挙動に合う

## Implementation outline

1. S1 gate.lock: `lockfile.go` に ctx で切れる上限のない待ち、`gate_lock.go` に gate.lock の印と待つ間の stderr の行(待ちの始め、保持者が変わったとき、5 分ごと、取れたとき)。Gate は検査のあとに取り、取ったら `gateOrgLife` と `gateCheckWorktree` をやり直す。local の regate は regate.lock の前に取り、最初の検査のあとに印へ org を書く。`GateParams` と `RegateParams` の Stderr、CLI からの受け渡し、log の見出しの待ちの行。scripts への gate.lock の fd の受け渡し(D8)。AC1〜AC4、AC14 — verify: `test -z "$(gofmt -l internal/org internal/cli)" && go vet ./internal/org/ ./internal/cli/ && GOFLAGS="-timeout=30m $(go env GOFLAGS)" go test ./internal/org/ -run 'Gate|Regate|Lock|Merge' && go test ./internal/cli/ -run 'Gate|Regate'`
2. S2 負荷と max_load: `gate_load.go`(読み取り、30 秒ごとの最大、max_load の待ち)、`gate_run.go` の見出しと結び、`gate_record.go` の鍵、`[org.gate].max_load` の config と検証と既定値のテストとテンプレート、CLI の `LOAD:` の行。AC5、AC6、AC11 — verify: `test -z "$(gofmt -l internal/org internal/cli internal/config)" && go vet ./internal/org/ ./internal/cli/ ./internal/config/ && GOFLAGS="-timeout=30m $(go env GOFLAGS)" go test ./internal/config/ && go test ./internal/org/ -run 'Gate|Regate|Load' && go test ./internal/cli/ -run 'Gate|Regate|Config'`
3. S3 accept と merge-order と merge: `gate_accept.go`、座席の pane の確かめの共有、`EventGateAccepted`、`collectMergeTargets` と `classifyMergeTarget` の `accepted`、JSON、merge の受け付けと `pr_merged` の `gate=accepted`、`leaderActivityEventCount` の説明とテスト、CLI の `ralph org gate accept` と merge・merge-order の出力。AC7〜AC10 — verify: `test -z "$(gofmt -l internal/org internal/cli)" && go vet ./internal/org/ ./internal/cli/ && GOFLAGS="-timeout=30m $(go env GOFLAGS)" go test ./internal/org/ -run 'Accept|MergeOrder|Merge|LeaderActivity|Approve' && go test ./internal/cli/ -run 'Gate|Merge|Approve'`
4. S4 skill と prompt と help: director と org の skill(`.claude/skills/` を直して `./scripts/sync-skills.sh` と templates の写し)、reviewer の役割 prompt、`ralph org gate` と `gate accept` の help。AC12、AC13 の help — verify: `./scripts/check-skill-sync.sh && ./scripts/check-sync.sh && GOFLAGS="-timeout=30m $(go env GOFLAGS)" go test ./internal/org/ -run 'Prompt' && go test ./internal/cli/ -run 'Help|Gate'`
5. S5 地図と仕様と tech-debt: AGENTS.md の internal/org の行、仕様の FR-7 の注記、`docs/tech-debt/README.md` の行(列の順番を約束しないこと、scripts を走らせない regate も待つこと、regate の merge 経路の失敗に accept が効かないこと)。AC13 の残り — verify: `./scripts/check-sync.sh && git diff --check`

## Verify plan

- Static analysis checks: `./scripts/run-static-verify.sh`(gofmt、go vet、golangci-lint、staticcheck、skill と templates の同期)
- Spec compliance criteria to confirm: AC1〜AC13。issue #309 の受け入れ条件 1〜3 が AC1、AC5、AC8 で覆われていること。ユーザーの決定 2 点(merge は accepted を受け付け、PR の check・保護・head の一致の確かめはそのまま。regate で head が変われば accept は消える)が AC8、AC9 と D5 に出ていること
- Documentation drift to check: `ralph org gate --help`、`ralph org gate accept --help`、`ralph org merge-order --help` の class の一覧、director と org の skill の 4 写し、reviewer の役割 prompt、AGENTS.md、仕様の FR-7、`templates/base/ralph.toml`、`.claude/rules/ralph/agent-messaging.md` に触れる必要がないこと
- Evidence to capture: verify の report に run-static-verify の結果と、AC ごとの根拠の場所

## Test plan

- Unit tests:
  - gate.lock: 1 本目が持つ間、2 本目は待ち、1 本目が離すと取れる。待ちの stderr に保持者の動詞・org・開始の時刻が出る。保持者が変わると行が出る。ctx で切ると何も取らずに戻る。印が空・壊れているときの行
  - 負荷: `sysctl` と `/proc/loadavg` の形の解析(読めない形を含む)、最大の取り方、読み取りの差し替え口
  - max_load: 0 で待たない、超えて下がれば始める、30 分(テストでは短くする)で始めて log と記録に残す、ctx で切れる、読めなければ待たない
  - 記録の鍵: 新しい鍵を含む Details の往復、前の形の Details の読み取り、値に `unknown`
  - accept: AC7 の記録と断りのそれぞれ、座席の pane の断り(approve の断りが変わらないこと)。人の承認を待つ間に gate が打ち直されて最新の失敗が変わった場合(古い log の accept は断られる)、同じ失敗への 2 つの accept を goroutine で同時に走らせた場合(1 件だけ記録)
  - merge-order: accepted になる場合、rebase・regate・no_remote になる場合、後ろの新しい gate 記録で効かなくなる場合、壊れた accept の行、別の枝や別の head の accept は効かない場合、JSON
  - merge: accepted で進み、ほかの確かめで断られる場合は今までどおり。`pr_merged` の `gate=accepted`
  - `leaderActivityEventCount` が `gate_accepted` を数えない
  - config: `max_load` の既定、負の値の設定エラー、テンプレートとの一致
- Integration tests:
  - D8: テストの binary を子の process として起こし、その子が gate.lock を取って script(sleep してから終わりの印を書く)を走らせる。script が走り出したら子だけを SIGKILL し、script が終わるまで gate.lock が持たれたまま(`fileLockHeld`)で、終わったあとに外れることを確かめる
  - 実際の git の worktree で 2 つの Gate を goroutine で同時に走らせ、scripts の実行が重ならないこと(scripts が開始と終了の時刻をファイルに書く)
  - 待つ間に HEAD を動かして、新しい HEAD で記録されること。待つ間に disband して断られること
  - local の regate が gate.lock を待つ間に、regate.lock が空いていること
  - CLI の `ralph org gate accept` と `ralph org merge-order` の出力
- Regression tests: 既存の gate・regate・merge-order・merge・approve・watch のテストがそのまま通ること
- Edge cases: 印のない gate.lock(前の版の binary が持つことはない。新設なので無い)、空の印、ロックを持った process が死んだあと(OS が離す)、`--reason` の改行や制御文字(記録は quote、表示は escape)、`max_load` が設定され負荷が読めない機械
- Evidence to capture: test の report に `./scripts/run-test.sh` の結果と、上のテストの名前

## Risks and mitigations

- R1 gate.lock の待ちで reviewer 座席の gate が長くなり、Codex の前面の timeout で切られる → reviewer の prompt に、待つことと timeout をそのぶん長くすることを書く。切られた gate は何も記録しない(今と同じ)
- R2 1 本の gate が止まったまま(scripts が終わらない)だと、台帳の全 gate が待ち続ける → 待つ間の stderr の行(5 分ごと)に保持者の pid と開始の時刻を出し、人が見て止められるようにする。tech-debt に残す
- R9 D8 で scripts に渡した fd を、親が SIGKILL で死んだあとに group の外へ逃げた子(setsid した daemon など)が持ち続けると、gate.lock が外れない → 待つ側の行が「保持者の pid が無い」と言うので、人が `fileLockHeld` の相手(lsof など)を止める。正常の終わりでは親の LOCK_UN で外れるので、親が生きていれば起きない。tech-debt に残す
- R3 scripts を走らせない regate(up_to_date、push_only)も前の gate を待つ → D1 の代わりに受け入れる。tech-debt に残す
- R4 flock は先着順を約束しないので、3 本以上待つと後から来た gate が先に走ることがある → Non-goals と tech-debt に書く
- R5 accept で、gate が失敗した PR が auto_merge で merge される → D3 で人の承認を要する決まりにし、merge の出力と `pr_merged` に `gate=accepted` を残して追えるようにする
- R6 regate の merge 経路の失敗には accept が効かない → D5 のとおり merge-order の detail で言う。tech-debt に残す
- R7 レーン D の #312b が `sameFeature` の意味を変える → 呼び出しの形(`gate.go` の recordGate、`regate.go` の regateBranchChanged)は触らない。merge の前に origin/main を取り込み、gate と regate のテストを打ち直す
- R8 ほかのレーンが同じファイル(config.go、templates/base/ralph.toml、既定値のテスト、director と org の skill、AGENTS.md、tech-debt)を変える → PR を作る前と merge の前に origin/main を取り込み、衝突は両方を残して解く

## Rollout or rollback notes

- 新しい event `gate_accepted` は古い binary が読み飛ばす。rollback すると merge-order は accept を見ず、その枝を failed と出す(知らせる側に倒れる)
- gate の記録の新しい鍵は古い binary が読み飛ばす
- gate.lock は新しいファイルで、古い binary は見ない。新旧の binary が混ざると、古い binary の gate は直列にならない
- `max_load` を書いた ralph.toml を古い binary が読むと、鍵は読み飛ばされる(`config.Load` は strict でない `toml.Unmarshal` で読む)
- release のノートに: gate が台帳で 1 本ずつになること、`max_load`、`gate accept` と `accepted`、director skill の更新(新しい skill は新しい binary が要る)

## Open questions

- なし(critical forks は Design decisions で決めた)

## Progress checklist

- [x] Plan reviewed — plan advisory: ran with codex。指摘 2 件を計画に反映した。(1) [HIGH] 人が受け入れると答えた失敗と別の失敗を accept できる → `--log <path>` を必須にし、manifest lock の中で最新の失敗の log と照らして同じ lock のうちに追記する(D5、AC7、Test plan に承認待ちの間の打ち直しと同時の accept)。(2) [MEDIUM] gate の親だけが SIGKILL されると scripts が走ったままロックが外れる → gate.lock の fd を scripts に継がせる(D8、AC14、R9、Integration tests に子の process を SIGKILL する試験)。図解ページの記録の形と「気をつけること」も直した。consult(consult-plan-gatelock)の判定「直してから進める」も反映済み(D1 のロックの順、D3 の人の承認、D5 の regate の merge 経路の制限)
- [x] Plan approved — ユーザーの 2026-10-11 と 2026-11-12 の指示(作った計画は自動承認)に従って承認を記録した
- [x] Branch created
- [x] Implementation started
  - S1 3e027e09(implementer): gate.lock、待ちの行、Gate の再検査、regate の順、scripts への fd の受け渡し。検証コマンド rc=0(internal/org 559 s、internal/cli 90 s)、golangci-lint 0 件、変異 4 つがすべて新しいテストで落ちた。既存テスト 2 本(`TestRegate_Interrupted`、`TestRegate_AnInterruptBeforeTheRecordUndoesTheMerge`)は、marker の時刻で時計の読み取りが 1 回増えたので、しきい値を 2 から 3 にした。自己点検 5 項目すべて clean。範囲外の気づき: `internal/cli/org_regate.go` 217 行目のコメントを S4 で「gate.lock or regate.lock」に直す
  - 検証コマンドの直し: 計画の `GOFLAGS="-timeout=30m …" go test A && go test B` は 2 つ目の `go test` に timeout が掛からない。S2 から各 `go test` に `-timeout=30m` を付ける
  - S2 7525c36b(implementer): 負荷の読み取り(`/usr/sbin/sysctl` を絶対パスと `LC_ALL=C` で、linux は `/proc/loadavg`)、30 秒ごとの最大、`max_load` の待ち(上限 30 分)、log の見出しと結び、記録の `load_start`・`load_max`・`load_end`・`ncpu`・`max_load`、CLI の `LOAD:` の行、config と既定値のテストとテンプレート。検証コマンド exit 0(config 14 s、org 487 s、cli 88 s)、staticcheck と golangci-lint 0 件、変異 3 つがすべて落ちた。Details と結果を全文で比べる既存テスト(gate・regate・CLI で計 21 本)を、固定した負荷の鍵に合わせて直した。CLI の e2e は実機の負荷を読むので、値の形を確かめてから置き換えて比べる。自己点検 5 項目すべて clean。申し送り: S4 で `ralph org gate --help` に `LOAD:` と max_load を足すとき、`org_gate_test.go` の help のピンも直す。S5 の tech-debt に、scripts を走らせない local の regate も max_load を待つことを足す
  - S3 6e4c5f48(implementer): `ralph org gate accept`(`--branch`・`--log`・`--reason`)と `gate_accepted`、座席の pane の断りの共有(`seatPaneTripwire`、approve の文言は変えない)、merge-order の `accepted` と JSON の `gate.log`・`gate.accepted`、merge の受け付けと出力の `GATE: accepted` の行と `pr_merged` の `gate=accepted`、`leaderActivityEventCount` の説明とテスト。検証コマンド exit 0 を 2 回(org 207 s と 271 s、cli 287 s)、golangci-lint 0 件、変異 10 個がすべて落ちた。log をまだ書いていない失敗の detail すべてに log を足した(既存の固定文字列 5 か所を更新)。自己点検: 誤字 2 件を直した、ほかは clean。申し送り: `internal/cli/org.go` の `--org-id` の説明の一覧と Long の「lists as ready」、director と status の「merge-order lists as ready」の文(固定のテストあり)を S4 で直す
  - origin/main(#316、d4bafaa7)を取り込んだ(ae372953、衝突なし、`go vet` は通る)
  - S4 d23ba892(implementer): director skill に「gate の失敗の受け入れ」節(確かめること、`--log` と `--reason`、断られたとき、leader への DECISION の例、`accepted` の merge)、件の処理と stall の注記、auto_merge の手順、してはいけないこと。org skill の動詞表(`gate`・`regate`・`merge-order`・`merge` の行、新しい `gate accept` の行)と「gate の記録」「merge してよい順」。reviewer の役割 prompt(待ち、Codex の timeout、`GATE: fail` の本文に `LOAD:`)。`ralph org gate`・`regate` の help、`--org-id` の一覧、「ready」を「ready or accepted」に直した 7 か所。skill の文を固定するテスト 2 本を足した。検証コマンドは 3 回目に exit 0(1 回目は `--org-id` の一覧の固定で落ち、直した。2 回目は範囲外の `TestOrgSend_CtxExpiresDuringEnterDelay_NotesTextTypedOnStderr` が load 19 前後で落ち、単独の `-count=3` で通った)。自己点検: 長すぎる行と regate の log の行の文を直した、ほかは clean。テストへの追記を一度 shell の heredoc でしたので、その範囲を Edit で書き直して hook を通した(U+FFFD は 0 件)。申し送り: `internal/config/config.go` の AutoMerge のコメント、README.md の `ralph status` の行、AGENTS.md の「ready」を S5 で直す
  - S5 4b1eafaa(implementer、sonnet): AGENTS.md の internal/org の行、README.md の「ready or accepted」3 か所、config.go の AutoMerge のコメント、仕様の FR-7 の注記、tech-debt の 5 行(a〜e)。tech-debt の行は `check_tech_debt_plan_refs` が実在しないパスを落とすので active のパスで書いた(`/pr` の archive-plan.sh が archive に書き換える)。検証コマンド exit 0、`HARNESS_VERIFY_MODE=static ./scripts/verify.local.sh` も通る。自己点検 5 項目すべて clean。申し送り: `templates/base/ralph.toml` の auto_merge のコメントの「lists as ready」と、README.md の merge-order の class の一覧(`accepted` が無い)と `ralph org` の動詞の一覧(`gate accept` が無い)を sync-docs で扱う
  - 全 slice の検証コマンドが通った(S1〜S5)
- [x] Review artifact created — `docs/reports/self-review-2026-11-12-gate-lock-accept.md`(Merge: yes、LOW 8)
- [x] Verification artifact created — `docs/reports/verify-2026-11-12-gate-lock-accept.md`(pass、LOW V1〜V4)
- [x] Test artifact created — `docs/reports/test-2026-11-12-gate-lock-accept.md`(pass、生き残った変異 C1・C2・C8・D1 は LOW)
  - /sync-docs 1 回目(cycle 1): verify の V1〜V4、self-review の L3〜L5、test の穴を直した。README の gate・merge-order・merge の段落と動詞の表、`templates/base/ralph.toml` と `OrgDirectorConfig` の auto_merge の注釈、`quality-gates.md` の 2 写し、director と org の skill の `--log` の例(4 写しずつ。director skill の固定のテストも直した)、`ci_failed` の受け入れが merge を通さない旨(director skill と `gate accept --help`、その固定)、tech-debt(条件に当たった 24 行の持ち越しの記録、gate.lock の行の広げ、test の穴の新しい行)。報告は `docs/reports/sync-docs-2026-11-12-gate-lock-accept.md`
- [ ] PR created
