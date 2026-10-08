# Verify report: org-limits-reserve

- Date: 2026-10-08
- Plan: docs/plans/active/2026-10-08-org-limits-reserve.md
- Verifier: verifier subagent (Claude Opus 5.5)、パイプライン cycle 1(`cycle-count.json` は 1)
- Scope: 仕様への適合(AC1〜AC15、仕様 `docs/specs/2026-10-07-org-multi-org-director.md` の FR-3 と受け入れ条件 96〜97 行)、静的解析、文書のずれ。対象は `git diff 51855166...HEAD`(HEAD 8dd19634、30 ファイル、+2948/-74)。コードの commit は 620458d7(設定)、11fc2261 と 8fe95acd(org の層)、78e46f36(CLI)、fd3e3b47(文書)、8dd19634(self-review F-1 と F-3 の文言の直し)。テストは実行していない(`/test` の担当)。テストは中身を読んで、各 AC の場合を押さえているかを判断した
- Evidence: `docs/evidence/verify-2026-10-08-org-limits-reserve.log`(`docs/evidence/*.log` は gitignore の対象なので手元にだけ残る)。runner 自身のログは `docs/evidence/verify-2026-10-08-114200.log`

## Spec compliance

plan の承認: `./scripts/plan-visual.sh digest` は `1a165903b5df` を返し、plan の `- Approved: 2026-10-08 sha256:1a165903b5df`(`:4`)と一致した。

| Acceptance criterion | Status | Evidence |
| --- | --- | --- |
| AC1: `max_orgs` に達すると、まだ走っていない org_id への spawn(start を含む)を拒否して `rejected` を残す。走っている org は影響を受けない | Met | 判定は `envelope.go:92-106`(`ValidateOrgWideCapacity`)。走っていない org_id で `len(runningOrgs) >= MaxOrgs` のときだけ拒否する。呼ぶのは `spawnCapacityErr`(`spawn.go:1063`)で、本番はロックの中の `checkCapacityAndStart`(`:1031-1036`)、dry-run は `:500` から呼ぶ。拒否は `o.reject` を通るので `rejected` と receipt が残る。テストは `TestOrgSpawn_MaxOrgs_NewOrgRejectedAtLimit`(`spawn_test.go:3684`)。拒否、driver を呼ばないこと、走っている org-a の 2 席目が通ること、disband で枠が空くことを見る。start 経由の拒否と、共有の台帳に `rejected` が残ることは CLI の `TestOrgStart_OrgWideLimits_ReadFromMainWorktreeRalphToml`(`org_reserve_test.go:364`)が見る。境界の値は `TestValidateOrgWideCapacity`(`envelope_test.go:34`)の「running org over max_orgs」などで押さえている |
| AC2: `max_total_seats` に達すると新しい座席を拒否して `rejected` を残す。立っている座席への spawn は既存を返す | Met | 判定は `envelope.go:101-104`。idempotent の戻りは上限の判定より前にある(`spawn.go:595-606`、`:756-763`)。`TestOrgSpawn_MaxTotalSeats_Boundary`(`spawn_test.go:3731`)は、走っている org と新しい org の両方の拒否、上限ちょうどでの respawn がイベントを増やさないこと、stop で枠が空くことを見る |
| AC3: 走っている org の数え方(3 条件、最後の `disbanded` より後だけ) | Met | `RunningOrgs`(`reserve.go:233-251`)は、Roster の Active と `currentOrgLives`(`:187-224`)の開いている workspace と予約から決める。`currentOrgLives` は最後の本物の `disbanded` 以前を `i <= d` で飛ばす(`:202`)。`TestRunningOrgs`(`reserve_test.go:133`)の 19 件に、`rejected` だけ、全席 stop で何も開いていない、古い ralph の disband が workspace を残した、dry-run だけ、dry-run の disbanded は効かない、の各場合が入っている |
| AC4: 同時の spawn が `max_orgs`・`max_total_seats` を超えない。重なる予約は片方だけ通る | Met(テストの中身とコードで確認。実行は /test) | 3 つの判定は、どれも `withManifestLock` の中で読み直した events から `spawnCapacityErr` で行う(Phase 1 は `spawn.go:575` のロックの中の `:700`、Phase 2 は `:749` のロックの中の `:768`)。予約の記録は `spawn_started` の前、ロックを放す前に書く(`:1037-1042`)。テストは `TestOrgSpawn_ConcurrentSpawns_MaxOrgsNeverExceeded`(`:3790`)、`..._MaxTotalSeatsNeverExceeded`(`:3813`)、`TestOrgSpawn_ConcurrentReservations_OnlyOneOfOverlappingWins`(`:3835`)の 3 本。形は既存の `TestOrgSpawn_ConcurrentSpawns_MaxSeatsNeverExceeded`(`:2058`)と同じで、goroutine を同じ `*Org` に向けて走らせる。flock は open ごとに掛かるので、同じプロセスの中でも直列になる(`lockfile.go:50-67`)。ただし自分の close の失敗の補償は上限を見ない(V-2) |
| AC5: org A の `internal/auth/` に対し、`internal/auth/token.go`・`internal/`・`.` は拒否されて相手と重なったパスが出る。`internal/authz/` は通る | Met | 重なりは `reservePathsOverlap`(`reserve.go:90-104`)で、ディレクトリは末尾の `/` 込みで接頭辞を比べる。拒否の文は `reservationDecision`(`:302-303`)。テストは `TestReservePathsOverlap`(`reserve_test.go:77`、両方の引数の順)、`TestOrgSpawn_Reserve_OverlapWithRunningOrgRejected`(`spawn_test.go:3923`)、CLI の `TestOrgStart_Reserve_OverlapWithRunningOrgRejected`(`org_reserve_test.go:154`)。実バイナリの dry-run でも 3 つが拒否され、`internal/authz/` が通った(Observational checks) |
| AC6: パスの規則。不正な入力はロックの前に拒否する | Met | `normalizeReservePath`(`reserve.go:61-82`)。入力検査はマニフェストを読む前に行う(`spawn.go:430-442`)。`TestNormalizeReservePaths` と `..._Rejects`(`reserve_test.go:13`、`:46`)に、`.`、`./`、末尾の `/`・`/.`、`//`、絶対パス、`..` の位置 4 通り、空、カンマ、空白、制御文字が入っている。`TestOrgSpawn_Reserve_InputRejectedBeforeAnyRecord`(`spawn_test.go:3882`)は、本番と dry-run の両方でイベント、receipt、driver の呼び出しがどれも 0 件であることを見る |
| AC7: 同じ一覧は記録なしで通り、違う一覧は拒否する。予約なしの spawn は予約のある org にも立つ。leader 以外の `--reserve` は拒否する | Met | `reservationDecision`(`reserve.go:283-292`)と `spawn.go:430-436`。`TestOrgSpawn_Reserve_SameSetPassesDifferentSetRejected`(`spawn_test.go:3949`)は、書き方を変えた同じ一覧で `scope_reserved` が 1 件のままであること、違う一覧の `rejected`、予約なしの seat-1 が立つことを見る。leader 以外は `spawn_test.go:3882` と CLI の `TestOrgSpawn_Reserve_LeaderOnly`(`org_reserve_test.go:116`)。カンマを含む 1 値が分割されないことは `TestOrgReserveFlag_ValueIsOnePath`(`:82`)が見る |
| AC8: disband のあと、ほかの org が同じパスを予約できる。予約だけの org も `disband --all` の対象 | Met | `orgsToDisband` に `EventScopeReserved` を足した(`verbs_all.go:319`)。テストは `TestOrgSpawn_Reserve_DisbandReleasesIt`(`spawn_test.go:4067`)、`TestOrgsToDisband_Definition` に足した 4 件(`verbs_all_test.go:418`)、`TestOrgDisbandAll_ReservationOnlyOrgDisbanded`(`:570`)。最後の 1 本は、spawn が workspace の作成で失敗して予約だけが残った org を作り、`--all` で driver を呼ばずに解散できることを見る |
| AC9: `--reserve` を渡した autonomous の spawn は `--scope` なしで通る | Met | `autonomousScopeGateErr`(`spawn.go:1186-1193`)の条件に `len(p.Reserve) == 0` が入った。`TestOrgSpawn_Reserve_SatisfiesAutonomousScopeGate`(`:4085`、本番と dry-run)と、CLI の `TestOrgStart_Reserve_RepeatedFlagRecordsOneReservation`(`org_reserve_test.go:53`、`--scope` なしの start)が見る |
| AC10: `ralph org status` が予約を表示する | Met | `orgReservation` と表と JSON の出力(`cli/org.go:1065-1071`、`:1108-1110`、`:1059`)。`TestOrgStatus_ShowsReservation`(`org_reserve_test.go:253`)と `..._ReservationWithoutSeatsAndAfterDisband`(`:301`)は、席がないときの行、disband 後と dry-run は出さないこと、予約がない org の JSON のキーが元の 2 つのままであることを見る |
| AC11: 既定 10 と 30。0 以下は読み込みで拒否。3 面が同期の検査で揃う | Met | `config.go:50-57`、`:161-162`、`:314-319`。`templates/base/ralph.toml:41-47`、`scripts/ralph-config.sh` の `RALPH_ORG_MAX_ORGS` / `RALPH_ORG_MAX_TOTAL_SEATS`(`templates/base/scripts/ralph-config.sh` と `cmp` で同一)。`defaults_sync_test.go:153-156` に 2 つの `check` を足した。`TestLoad_OrgFleetLimitsBoundary`(`config_test.go:579`)は 0 と -1 の拒否、1 が通ること、片方だけ書いたとき他方が既定値になることを見る |
| AC12: `/org` skill(4 面)と README が説明し、`check-skill-sync.sh` と `check-sync.sh` が通る | Met(文書の正確さに LOW の指摘が残る。V-1、V-2) | 4 面の sha256 はすべて `f47919e1…`。skill の新しい節は `.claude/skills/org/SKILL.md:174-227`、README は `:124` と `:247`。2 つの検査は runner の中で通った(下の表) |
| AC13: `--config` なしでは main worktree の `ralph.toml` の `max_orgs = 1` が、サブディレクトリからも、`ralph.toml` の違う linked worktree からも効く。`--config` はその設定を使う | Met | `newOrgSpawnRuntime` と `withMainWorktreeOrgLimits`(`cli/org.go:121-163`)、`MainWorktreeRoot`(`statedir.go:81-93`)。CLI の `TestOrgStart_OrgWideLimits_ReadFromMainWorktreeRalphToml`(`org_reserve_test.go:364`)は、サブディレクトリと linked worktree で拒否、`--config`、`--state-dir`、`RALPH_ORG_STATE_DIR` では通る、の 5 通りを見る。`TestOrgSpawn_MainWorktreeRalphToml_OnlyOrgWideLimitsTaken`(`:431`)は、main から取るのが 2 つの上限だけで `max_seats` は取らないことを見る。`TestOrgStart_MainWorktreeRalphTomlLoadError`(`:462`)は、main の `ralph.toml` が壊れているとき既定値に落とさず止まることを見る。`TestMainWorktreeRoot`(`statedir_test.go:238`)は、git-main-worktree 以外の 4 つの source で root を返さないことを見る |
| AC14: 立っている leader への `--reserve`。予約がなければ予約し、同じなら通し、違えば座席を変えずに拒否する。予約なしの再試行は既存を返す | Met | `idempotentRespawn`(`spawn.go:1085-1098`)。Phase 1 と Phase 2 の両方の idempotent の戻りで、ロックの中から呼ぶ(`:604`、`:760`)。拒否は `rejected` を書かない(理由は doc の `:1076-1084`)。`TestOrgSpawn_Reserve_ExistingLeader`(`spawn_test.go:3982`)の 5 通りは、events・receipts・herdr・agmsg の件数の比較と `assertSeatUnchanged` で、状態を変えないことを固定している |
| AC15: 自分の workspace の close の失敗で補償された org は、`disbanded` の前の予約も持ち直す | Met | `CloseDeferredSelfWorkspace` が `reserveAgain` を呼ぶ(`verbs.go:1111`、本体は `:1225-1237`)。読み戻しは `reservationBeforeLastDisband`(`reserve.go:311-317`)。`TestOrgCloseDeferredSelfWorkspace_CloseFails_ReservationRestored`(`verbs_test.go:2988`)は、書き戻し、エラー文、ほかの org の拒否、`--force` では書かないこと、予約がなければ書かないことを見る。plan のリスクの窓は `..._RiskWindowClearedByRetry`(`:3048`)が作り、打ち直しの disband で解けることを見る |

仕様 FR-3 との突き合わせ:

- 上限を台帳の共通のロックの下で強制する: 上の AC1、AC2、AC4 のとおり。「director は数えない」は、director がまだないので空で成り立つ(plan の Non-goals)。8dd19634 で、出荷物から director の語を消した(`git grep -nw director` で残るのは、仕様・計画・レポート以外では tech-debt の行と無関係のテスト文字列だけ)
- start のときの予約、重なりの拒否、disband で解く: AC5〜AC8 のとおり。予約を任意にしたのはユーザーの選択で、plan の Design decisions に記録がある
- 予約の外への書き込みは止めず、watchdog の ALERT で知らせる: watchdog は変わっていない(`watch.go:70` の `condScopeChange`)。skill の `:223-225` と `reserve.go:22-23` がこの点を書いている
- 仕様の受け入れ条件(96〜97 行)の 2 件は、AC1(CLI の start での拒否と記録)と AC5 で満たす

## Static analysis

| Command | Result | Notes |
| --- | --- | --- |
| `./scripts/run-static-verify.sh`(scope changed) | rc 0 | `scripts/ralph-config.sh` が言語を判定できない変更なので full fallback になり、golang の pack が選ばれた |
| shellcheck、`sh -n`(hooks 20 本)、`jq -e`(settings.json 2 本) | OK | |
| `scripts/check-sync.sh` | PASS | IDENTICAL 164、DRIFTED 0、ROOT_ONLY 0 |
| `check-pipeline-sync.sh`、`check-skill-sync.sh`、`check-template-purity.sh` | OK | 13 skill が一致。templates に meta-repo の参照なし |
| tech-debt README plan references | OK | |
| gofmt、go vet、golangci-lint、staticcheck | `gofmt: ok`、出力なし、`0 issues.`、出力なし | pack の `verify.sh:91-104`。Skipping の行がないので 4 つとも走った。`go vet ./internal/org/ ./internal/cli/ ./internal/config/` も手元で単独に通した |
| `secret-scan-branch.sh`(runner の中) | clean | 51855166..8dd19634 |
| `git diff --check 51855166...HEAD`、U+FFFD の検索 | 空、0 件 | |

## Documentation drift

| Doc / contract | In sync? | Notes |
| --- | --- | --- |
| `/org` skill の上限と予約の節(4 面) | ほぼ合う(V-1、V-2、V-5) | 予約の規則、重なり、leader 限定、scope のゲート、status の行、走っている org の数え方はコードと合う |
| `ralph org start/spawn --help` | ほぼ合う(V-1、V-6) | `--reserve` は `StringArray` で、usage に backtick がない(`cli/org.go:372-378`) |
| `ralph org status --help` | 書いていない(V-6) | Short の「Show org seat roster」だけ |
| `templates/base/ralph.toml:41-47` | Yes | 「state dir が main worktree の `.harness/state/org` のとき」と書くので、bare リポジトリの場合も正しく外れる |
| `README.md:124`、`:247` | Yes | |
| `docs/quality/quality-gates.md:77`(templates の写しは `:76`) | No(V-3) | |
| `envelope_summary.go:36`、`doctor.go:788` | 判断が要る(V-4) | |
| tech-debt の行 170 と 144 | 更新が要る(V-7、V-8) | |
| plan の進捗のチェックボックス | 遅れている(V-9) | |

### V-1: main の `ralph.toml` を読まない場合の列挙に、bare リポジトリの linked worktree が抜けている(LOW)

skill の `:199-202` は、打った場所の設定を使う場合を「`--config` を渡したときと、台帳を flag か env で決めたとき、git の外では」と並べる。コードは、source が `git-main-worktree` 以外ならすべて main を読まない(`statedir.go:81-83`、`cli/org.go:148-155`)。ここには `git-toplevel`(main worktree を決められない bare リポジトリの linked worktree、`statedir.go:47-51`、`:69-70`)も入る。`TestMainWorktreeRoot` の「git-toplevel」の場合がこれを固定している。help の `orgWideLimitsHelp` の括弧書き「(no --state-dir or RALPH_ORG_STATE_DIR)」(`cli/org.go:366-367`)も同じ穴を持つ。/sync-docs で、skill の列挙に「台帳を git の toplevel から決めたとき(bare リポジトリの linked worktree)」を足すことを勧める。help は文字列なのでコードの変更になる。tech-debt に送るか、cross-review の判断に回す。

### V-2: 「同時に打った spawn でも超えない」は、自分の close の失敗の補償には当たらない(LOW)

skill の `:185` は「どちらも台帳のロックの下で判定するので、同時に打った `spawn` でも超えない」と書く。spawn どうしについては正しい(AC4)。ただし、補償は上限も予約の重なりも見ずに書き戻す。

- `CloseDeferredSelfWorkspace` の補償(`reopenWorkspace`、`reactivateSeat`、`reserveAgain`、`verbs.go:1082-1111`)は、`disbanded` から補償までの間にほかの org が枠か同じ範囲を取ると、`max_orgs` を 1 つ超え、予約が重なる。`reserveAgain` の doc(`:1215-1224`)、plan のリスク、テスト `..._RiskWindowClearedByRetry` はこの点を書いているが、skill には書いていない
- `CloseDeferredSelfPane`(`verbs.go:1010-1034`)の `reactivateSeat` も、止めた座席を上限を見ずに戻す。そのため、窓の間に別の座席が立つと `max_total_seats` を 1 つ超えうる(`max_seats` も同じで、これは前からある)。plan のリスクが挙げるのは disband の経路だけで、こちらはコードを読んで分かったことでテストはない。workspace は開いたままなので `max_orgs` は影響を受けない

/sync-docs で、skill の `:185` の後ろに「ただし、自分の pane や workspace の最後の close が失敗して座席や workspace を戻したときは、上限と予約を見ずに戻すので、打ち直しの `disband` まで 1 つ超えるか重なることがある」の 1 文を足すことを勧める。

### V-3: quality-gates の envelope validation の行が `max_seats` だけを書く(LOW)

`docs/quality/quality-gates.md:77` と `templates/base/docs/quality/quality-gates.md:76` は、`ralph org spawn` が見る項目を「model pool / role pool / `max_seats`」と書く。今の spawn は、このほかに `max_orgs`、`max_total_seats`、予約の重なりを見て、拒否を `rejected` で記録する(`spawn.go:1063-1074`)。この 2 ファイルは `check-sync.sh` の KNOWN_DIFF なので、2 つとも直す必要がある。/sync-docs で直せる。

### V-4: leader のプロンプトの `{{ENVELOPE}}` と doctor の要約は `max_seats` だけを出す(LOW、判断が要る)

`envelope_summary.go:36` と `doctor.go:788` は `max_orgs` と `max_total_seats` を出さない。leader は、自分の org の外の枠をプロンプトから知らず、拒否の文で初めて知る。足すならコードの変更なので、/sync-docs では tech-debt の候補として記録するのがよい。

### V-5: `--reserve` だけを渡した leader のプロンプトの `{{SCOPE}}` が空になる(LOW、self-review F-4、plan の進捗の (c))

`spawn.go:809` と `:1715` は役割のプロンプトに `p.Scope` だけを渡す。`internal/org/prompts/leader.md:4` の `scope:` は、`--reserve` だけのときは既定の文言になる。skill の `:222` は「`--reserve` を渡した spawn は、autonomous の `--scope` 必須のゲートを満たす」とだけ書き、プロンプトに範囲が出ないことは書いていない。/sync-docs で tech-debt の行に送り、skill のその行に 1 文足すことを勧める。

### V-6: help の文言(LOW、self-review F-11 は未対応)

- 永続フラグ `--config` の説明(`cli/org.go:46`)に spawn と start だけに関わる文が入り、`status`、`stop` などすべての org の動詞の help に出る
- `orgWideLimitsHelp`(`:363`)は「spawn checks」で始まり、`ralph org start --help` にもそのまま出る
- `ralph org status --help` は `reserved:` の行と JSON の `reservation` に触れない。skill(`:167`、`:226-227`)には書いてある
- start の `--scope` の help が「`--scope ralph org spawn --scope`」と崩れて出るのは前からある(plan の進捗の (d)。usage の backtick を pflag が値の名前と読むため)

どれもコードの文字列なので、tech-debt に送るか cross-review の判断に回す。

### V-7: tech-debt の行 170 の見直しのきっかけが、この PR で来ている

行 170(`docs/tech-debt/README.md:170`)のきっかけは「The next change to `CloseDeferredSelfPane`, `CloseDeferredSelfWorkspace`, `reactivateSeat`, or `reopenWorkspace`」で、この PR は `CloseDeferredSelfWorkspace` を変えた(`verbs.go:1111`)。(a) C2-1(補償が driver の呼び出しの前に読んだ台帳の写しで決める)は、新しい `reserveAgain` にも当たる(self-review F-5。同じ org_id を窓の間に `--reserve` で立て直すと、古い予約が新しい予約を上書きする)。(c) C2-5(`verbs.go` を 800 行の目安に収めるため、最後の close のまとまりを別のファイルに移す)はされず、`verbs.go` は base の 1,847 行から 1,875 行になった。/sync-docs で、行 170 の (a) に `reserveAgain` を足し、きっかけを次の変更に掛け直すことを勧める。

### V-8: tech-debt の行 144 の一部が狭まり、別の影響が 1 つ増えた

行 144(`:144`)は、config と state dir の解決の規則が違うことを書く。この PR で、spawn と start は、台帳が main worktree のものなら `max_orgs` と `max_total_seats` を main の `ralph.toml` から読むようになった。行の残り(`max_seats`、`model_pool`、`[org.roles]`、teardown の動詞)は変わらない。一方、main の `ralph.toml` が読み込みに失敗すると、linked worktree 側の設定が正しくても spawn と start が止まる(`cli/org.go:157-160`、テスト `TestOrgStart_MainWorktreeRalphTomlLoadError`)。行の「壊れた `ralph.toml`」の影響の範囲が広がった。行のきっかけ(`resolveOrgConfig` か `ResolveOrgStateDir` の変更)は、文字どおりには来ていない(どちらも変わらず、`MainWorktreeRoot` は隣に足した新しい関数)。/sync-docs で行 144 に、この 2 点を 1 文ずつ足すことを勧める。

### V-9: plan の進捗のチェックボックス

`:151` の「Review artifact created」は未チェックだが、self-review のレポートは d8ec84da で作られている。「Verification artifact created」(`:152`)はこのレポートで満たされる。/sync-docs か /pr で直せる。

### /sync-docs に渡す tech-debt の候補

self-review が挙げた F-5、F-8、F-9、F-10 と、plan の進捗の (a)〜(d)(self-review の対応づけでは (a) は F-10、(c) は F-4)。このほか self-review の Coverage gaps にある、予約のパスを大文字小文字を区別して比べる件(macOS の既定のファイルシステムでは `Internal/` と `internal/` は同じ場所を指すが、重ならないと判定される)。plan に書かれていない扱いで、予約は助言なので、tech-debt に記録する程度でよいと判断した。

## Observational checks

実バイナリ(`go build ./cmd/ralph` で scratchpad に作ったもの)で、herdr と agmsg を PATH から外し(`PATH=/usr/bin:/bin`)、scratch の state dir に対して `--dry-run` の spawn と status を打った。台帳には、org-a の leader が `internal/auth/` を予約して立っている状態を 3 行の JSONL で用意した。手順と出力は evidence のログにある。

- org-b の leader の `--reserve internal/auth/token.go`、`internal/`、`.` は、どれも `reservation overlaps the reservation of running org_id "org-a": <path> overlaps internal/auth/` で rc 1。`internal/authz/` は通り、dry-run の trail が `scope_reserved`(dry_run)、`spawn_started` の順で始まった
- `/etc/`、`internal/../x`、空、`a,b` は入力検査の文で rc 1 になり、台帳に何も書かなかった。leader 以外の `--reserve` も同じだった
- `--scope` も `--reserve` もない autonomous の leader は「requires --scope or --reserve (--reserve on the leader seat only; …)」で拒否された。8dd19634 の F-3 の直しが出ている
- org-a の leader に違う一覧(`docs/`)を渡した dry-run は「already reserves internal/auth/ … ralph org disband --org-id org-a」で拒否され、dry-run の `rejected` が 1 行増えた。本番の経路では `idempotentRespawn` が `rejected` を書かない。dry-run には idempotent の戻りがない(`spawn.go:378-381` の doc のとおり)。dry-run の行は Roster に数えないので、座席の状態には効かない
- `status --org-id org-a` は表の下に `reserved: internal/auth/` を出し、`--json` は `"reservation": ["internal/auth/"]` を出した。予約のない org-b は `no seats` だけを出した
- `--config` で `max_orgs = 1` にすると、org-c の dry-run は `max_orgs 1 reached: org_id "org-c" is not running and 1 orgs are (org-a); a finished org frees its slot … ralph org disband --all for every org` で拒否された。`max_total_seats = 1` でも同じ形で拒否された。`max_orgs = 0` は `org: load config: [org].max_orgs must be >= 1, got 0` で、spawn の前に止まった
- `ralph org start --help` と `ralph org spawn --help` の出力は V-1 と V-6 のとおり

## Coverage gaps

- テストは実行していない(`/test` の担当)。各 AC のテストは中身を読んで判断した。mutation は再現していない
- AC4 の同時実行のテストは、同じプロセスの goroutine で flock を競わせる。別プロセスの競合は再現していない(既存の `max_seats` のテストと同じ範囲)
- V-2 の pane の経路(`CloseDeferredSelfPane` の補償で `max_total_seats` を 1 つ超える)は、コードを読んで分かったことで、テストはない
- 本物の herdr と agmsg では動かしていない。本番の spawn(dry-run でないもの)は、Go のテストの fake を読んだだけで確かめた
- bare リポジトリの linked worktree で、main の `ralph.toml` を読まないことは、`TestMainWorktreeRoot` の単体の場合とコードで確かめた。CLI からは打っていない

## Verdict

- Verdict: pass
- Verified: AC1〜AC15 の実装の位置とテストの対応(上の表)。plan の承認の digest の一致。`run-static-verify.sh` が rc 0 で終わること(gofmt、go vet、golangci-lint、staticcheck、check-sync、check-skill-sync、check-pipeline-sync、check-template-purity、secret scan)。`/org` skill の 4 面が同じ内容であること。self-review の F-1(director の語)と F-3(scope のゲートの文)が 8dd19634 で直っていること。実バイナリの dry-run で、重なりの拒否、入力検査、scope のゲート、上限の拒否と案内の文、status の表示が仕様のとおりに出ること
- Partially verified: AC4 と AC15 はテストの中身とコードで確かめ、実行は /test に任せる。AC12 の文書は、V-1(bare リポジトリの場合の列挙の抜け)と V-2(補償の窓の断り)が残る。そのほかの文書は V-3(quality-gates)、V-5(`{{SCOPE}}`)、V-7・V-8(tech-debt の行)、V-9(チェックボックス)を /sync-docs で直せる。V-4 と V-6 はコードの文字列なので、tech-debt に送るか cross-review で判断する。どれも merge を止めない
- Not verified: テストの実行、mutation、別プロセスでの競合、V-2 の pane の経路の再現、本物の herdr と agmsg での動作
