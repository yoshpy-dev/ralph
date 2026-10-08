# Verify report: org-limits-reserve

- Date: 2026-10-09 JST(ファイル名は計画の日付)
- Plan: docs/plans/active/2026-10-08-org-limits-reserve.md
- Verifier: verifier subagent (Claude Opus 5.5)、パイプライン cycle 2(`cycle-count.json` は 2。既定の上限の最後の回)
- Scope: AC1〜AC15 を HEAD a919082a で見直した。cycle 1 の verify(fbb04f83)からの差分は、c3a95c48(テスト)、76d1cf1c(skill・quality-gates・help・仕様・`AGENTS.md`)、196205f8(tech-debt の行と plan のチェック)、6600b3a9(cross-review の triage)、975df92b と afcbc6c2(動いていない leader の予約の前に max_orgs を判定する修正と、その判定の `validateMaxOrgs` への切り出し)、710de10d(plan の進捗)、a919082a(self-review cycle 2)。重点は AC1 と AC14、self-review C2-2 の判断、文書のずれ(skill の 4 面、76d1cf1c の help、`templates/base/ralph.toml` のコメント(C2-4)、`AGENTS.md`(C2-7)、tech-debt の行 173(C2-6))。テストは実行していない(`/test` の担当)
- Evidence: `docs/evidence/verify-2026-10-08-org-limits-reserve.log` の末尾の「cycle 2」の節(`docs/evidence/*.log` は gitignore の対象なので手元にだけ残る)。runner 自身のログは `docs/evidence/verify-2026-10-08-153233.log`
- 番号の付け方: cycle 1 の V-1〜V-9 は付録 A に原文のまま残す。tech-debt の行 144・170・172・174 と `internal/org/verbs_test.go:3084` のコメントがその番号を指しているため。この回の新しい指摘は V2-1 から振る

## Spec compliance

plan の承認: `./scripts/plan-visual.sh digest` は `1a165903b5df` を返し、plan の `- Approved:` の行(`:4`)と一致した。cycle 1 のあとの plan の差分は、進捗の節とチェックボックスだけで、どちらも digest の対象の外にある(`scripts/plan-visual.sh:271-283` の `digest_body`)。

| Acceptance criterion | Status | Evidence |
| --- | --- | --- |
| AC1: `max_orgs` に達すると、まだ走っていない org_id への spawn(start を含む)を拒否して `rejected` を残す。走っている org は影響を受けない | Met(`rejected` を書かない経路が 1 つある。下の「C2-2 の判断」と V2-1) | 新しい座席の spawn の経路は変わらない。`ValidateOrgWideCapacity`(`envelope.go:92`)は max_orgs の半分を `validateMaxOrgs`(`:107`)に移しただけで、条件、エラー文、max_orgs から max_total_seats への順は fbb04f83 と同じ(`git diff fbb04f83..HEAD -- internal/org/envelope.go` で確認)。cross-review の修正で、すでに `spawned` の leader に `--reserve` を渡した再試行も、座席が Active でなければ `validateMaxOrgs` を通る(`spawn.go:1100-1104`)。渡す events は Phase 1(`:606`)でも Phase 2(`:762`)でもロックの中で読み直したもので、`o.Config` は CLI の `newOrgSpawnRuntime`(`cli/org.go:121`)が main worktree の上限に差し替えた設定。テストは `TestOrgSpawn_Reserve_InactiveLeaderChecksMaxOrgs`(`spawn_test.go:4116`)で、Phase 1 と Phase 2、max_orgs 1 と 2、別の座席で org が走っている場合の 5 通りを見る。拒否の文が新しい org のときと同じ文字列であること、記録・receipt・agmsg の呼び出しがないこと、座席が `spawned` のまま inactive であることを確かめている。実バイナリでも同じ結果になった(Observational checks)。ほかの経路の証拠は cycle 1 のとおり(付録 A) |
| AC2: `max_total_seats` に達すると新しい座席を拒否して `rejected` を残す。立っている座席への spawn は既存を返す | Met | 判定は `ValidateOrgWideCapacity` に残り、文言も同じ。c3a95c48 は、pane の補償が上限を 1 つ超える今の挙動を固定するテスト `TestOrgCloseDeferredSelfPane_CloseFails_ReactivationCanExceedMaxTotalSeats`(`verbs_test.go:3092`)を足した。挙動を変えるテストではない |
| AC3: 走っている org の数え方 | Met | `internal/org/reserve.go` は fbb04f83 から変わっていない(`git diff fbb04f83..HEAD -- internal/org/reserve.go` は空)。cycle 1 の証拠のとおり |
| AC4: 同時の spawn が上限を超えず、重なる予約は片方だけ通る | Met(テストの中身とコードで確認。実行は /test) | 975df92b の判定も、ロックの中で読み直した events から行う。同時実行の 3 本のテストは変わらない |
| AC5〜AC11、AC13、AC15 | Met | `reserve.go`、`verbs.go`、`verbs_all.go`、`internal/config/`、`scripts/`、`templates/base/ralph.toml`、`README.md` は fbb04f83 から変わらず、`internal/cli/org.go` の差分はヘルプ文の 3 か所だけ。cycle 1 の証拠のとおり(付録 A)。`spawn.go` の 1078 行より後は、cycle 1 の行番号から 2〜18 行ずれたので、関数名で探す |
| AC12: `/org` skill(4 面)と README が説明し、`check-skill-sync.sh` と `check-sync.sh` が通る | Met | 4 面の sha256 はすべて `771be761…` で同じ。2 つの検査は runner の中で通った。cycle 1 の V-1(bare リポジトリの列挙)と V-2(補償の例外)は 76d1cf1c で skill に入った(下の「cycle 1 の指摘の現況」) |
| AC14: 立っている leader への `--reserve`。予約がなければ予約し、同じなら通し、違えば座席を変えずに拒否する。予約なしの再試行は既存を返す | Met | `idempotentRespawn`(`spawn.go:1098-1116`)。Active な leader は `if !seat.Active` で max_orgs を飛ばし、これまでどおり `reservationDecision` に進む。動いていない leader は max_orgs、予約の順に判定する。予約のない再試行は `len(p.Reserve) > 0` の外なので、既存の座席を返す。拒否はどれも `rejected` と receipt を書かない。c3a95c48 の `TestOrgSpawn_Reserve_ExistingLeader_Phase2`(`spawn_test.go:4072`)は、Phase 2 の idempotent の戻りでも予約が記録され、2 つ目の saga が始まらないことを見る。既存の `TestOrgSpawn_Reserve_ExistingLeader`(`:3982`)は、予約なし・同じ一覧・違う一覧・重なり・予約なしの再試行を見る |

仕様 FR-3 との突き合わせは cycle 1 から変わらない。76d1cf1c が仕様の `:41` に足した「3 段目で決めたこと」の行は、予約を任意にしたこと、受け付ける動詞、走っている org の数え方でコードと合う。上限の読み元の書き方は V2-2 を参照。

### C2-2 の判断(AC1 の文言と、動いていない leader の拒否)

AC1 は「まだ走っていない org_id への spawn は拒否され、台帳に `rejected` が残る」と書く。975df92b の枝は、走っていない org への `--reserve` つきの spawn を max_orgs で拒否する。ここで言う org は、古い台帳で leader の `spawned` のあとに `stopped` なしで `disbanded` が来たもの。この拒否は `rejected` も receipt も書かず、実バイナリでも拒否の前後で台帳はバイト単位で同じだった。AC1 の文字どおりには満たしていない。

書かないほうが正しいと判断した。`rejected` は状態のイベントなので(`seat.go:59-67`)、leader の座席 id で書くと、その座席の最新の状態が `spawned` から `rejected` に変わる。すると次の予約なしの再試行は idempotent の戻りに入らず、新しい spawn に進む。これは AC14 の「予約のない再試行は既存の座席を返す」と、plan の Scope(38 行)の「座席の状態を変えずに拒否する」に反する。この座席は Roster の上では「すでに立っている」ので、plan の進捗(155 行)の決定「すでに立っている leader への予約の拒否は `rejected` を書かない」がそのまま当たる。古い台帳のこの座席では AC1 と AC14 の両方が当てはまり、コードは AC14 を取った。計画を書いたときには想定していなかった場合で、cross-review で見つかった。

そのため、挙動の欠陥ではなく計画の文の穴として扱う(V2-1)。AC1 の本文を直すと承認の digest が変わるので、本文は直さず、digest の外の進捗の節に記録することを勧める。

### cycle 1 の判定の訂正

cycle 1 の Documentation drift の表は `templates/base/ralph.toml:41-47` を「Yes」とし、「state dir が main worktree の `.harness/state/org` のとき」と書くので bare リポジトリの場合も正しく外れる、とした。これは正確ではなかった。`--state-dir` か `RALPH_ORG_STATE_DIR` に main の `.harness/state/org` を渡しても、state dir は main worktree のものになる。しかし source は `flag` か `env` になるので、コードは main の `ralph.toml` を読まない(`MainWorktreeRoot` は `git-main-worktree` 以外で ok=false を返す。`statedir.go:81-93`)。self-review C2-4 の指摘のとおりで、V2-2 に入れた。付録 A の原文は直していない。

## Static analysis

| Command | Result | Notes |
| --- | --- | --- |
| `./scripts/run-static-verify.sh`(scope changed) | rc 0 | `scripts/ralph-config.sh` が言語を判定できない変更なので full fallback になり、golang の pack が選ばれた |
| shellcheck、`sh -n`(hooks 20 本)、`jq -e`(settings.json 2 本) | OK | |
| `scripts/check-sync.sh` | PASS | IDENTICAL 164、DRIFTED 0、ROOT_ONLY 0、KNOWN_DIFF 5 |
| `check-pipeline-sync.sh`、`check-skill-sync.sh`、`check-template-purity.sh`、tech-debt README plan references | OK | 13 skill が一致 |
| gofmt、go vet、golangci-lint、staticcheck | `gofmt: ok`、出力なし、`0 issues.`、出力なし | Skipping の行がないので 4 つとも走った |
| `secret-scan-branch.sh`(runner の中) | clean | 51855166..a919082a、origin/main に対して |
| `git diff --check 51855166...HEAD` | rc 2 | `docs/reports/self-review-2026-10-08-org-limits-reserve.md:131: new blank line at EOF.`(V2-5)。runner と CI はこの検査を回さない |
| U+FFFD の検索(`git diff 51855166...HEAD`) | 0 件 | |

## Documentation drift

| Doc / contract | In sync? | Notes |
| --- | --- | --- |
| `/org` skill の上限と予約の節(4 面) | Yes | 補償の例外(`SKILL.md:190-195`)、上限の読み元の列挙(`:203-211`)、末尾の `/` のないパス(`:223-225`)、`{{SCOPE}}`(`:233-236`)はコードと合う。`:186` の「拒否された新しい座席の `spawn` は `rejected` を台帳に書き」は「新しい座席」に絞っているので、V2-1 の枝とも食い違わない |
| `ralph org spawn --help`、`ralph org start --help` | 内容は合う。文の組み立ては LOW(V2-2) | `orgWideLimitsHelp`(`cli/org.go:363`)。後ろの「In every other case (...)」が flag、env、git toplevel、git の外を並べるので、条件はコードと合う |
| `ralph org status --help` | Yes | 76d1cf1c の `Long`(`cli/org.go:1005`)が `reserved:` の行と JSON の `reservation` を書く |
| 永続フラグ `--config` の説明 | Yes | spawn と start の上限の読み元は `ralph org spawn --help` を見るよう案内する(`cli/org.go:46`) |
| `templates/base/ralph.toml:41-47` | No(V2-2) | |
| 仕様 `:41` の「3 段目で決めたこと」 | ほぼ合う(V2-2) | |
| `docs/quality/quality-gates.md:77`、`templates/base/docs/quality/quality-gates.md:76` | ほぼ合う(V2-1) | 2 つは同じ文になった。「Spawn rejected, recorded in manifest」は、すでに立っている leader への `--reserve` の拒否には当たらない |
| `AGENTS.md:90` | No(V2-3) | |
| tech-debt の行 | 一部ずれ(V2-4) | |
| `README.md:124`、`:247` | Yes | 変わっていない |
| plan の進捗のチェックボックス | Yes | `:151-153` がチェック済み |

### V2-1: 動いていない leader の予約を max_orgs で拒否するとき `rejected` を書かないことが、計画に書かれていない(LOW、self-review C2-2)

挙動は上の「C2-2 の判断」のとおり AC14 に合わせた選択で、直す必要はない。ただ、AC1 の「台帳に `rejected` が残る」と、進捗の 156 行の「AC1 にコードを合わせる直し」だけを読むと、この枝も `rejected` を書くように読める。起きるのは、古い ralph が `stopped` なしで `disband` した台帳だけ。今の `Disband` は先に `stopped` か `stop_failed` を書く(cross-review の triage)。拒否は stderr と終了コード 1 にだけ出て、台帳と receipt には残らない。

記録のしかたとして次を勧める。

- /sync-docs で plan の進捗の節に 1 行足す。趣旨は「975df92b の max_orgs の拒否は、AC14 の拒否と同じく `rejected` を書かない(書くと予約のない再試行が既存の座席を返さなくなる)。AC1 の `rejected` は新しい座席の spawn に当たる」。進捗の節は digest の外なので承認は保たれる。AC1 の本文は直さない
- self-review が提案した tech-debt のまとめの行(C2-1〜C2-7)の (a) に、doc の理由の文(「showing it inactive」がこの枝に当てはまらないこと)と一緒に入れる
- quality-gates の行の「recorded in manifest」には、入力検査の拒否と同じく例外がある。直すなら 2 面とも「(a refusal of `--reserve` for an already-spawned leader is not recorded)」を足す。任意

なお、max_orgs に空きがあると予約は記録され、CLI は `seat "leader" already spawned (org_id=org-a driver=claude model=sonnet pane_id=legacy-pane)` と、古い pane の座席を返す。その org は、動く座席がないまま予約で枠を 1 つ持ち、`disband` で解ける。plan のリスクの「spawn が失敗して予約だけが残る」と同じ形で、`ralph org status` に `reserved:` が出るので見える。finding にはしない。

### V2-2: `templates/base/ralph.toml` のコメントが、flag か env で main の台帳を指した場合も main の `ralph.toml` を読むように読める(LOW、self-review C2-4)

`templates/base/ralph.toml:42-43` は「Without --config, when the state dir is the main worktree's .harness/state/org, ralph reads it from the main worktree's ralph.toml.」と書く。コードが main の `ralph.toml` を読むのは、`--config` がなく、`ResolveOrgStateDir` が既定の順で台帳を main worktree に決めたとき(source が `git-main-worktree`)だけ(`withMainWorktreeOrgLimits`、`cli/org.go:137-163`)。`--state-dir` か `RALPH_ORG_STATE_DIR` に main の `.harness/state/org` を渡すと、state dir は同じでも読まない。このファイルは `ralph init` で配られる。/sync-docs で「Without --config, when neither --state-dir nor RALPH_ORG_STATE_DIR is set and the state dir resolves to the main worktree's .harness/state/org, ...」の形に直すことを勧める。値は変えない。コメントの文言を固定するテストはない(`git grep` で 0 件)。

同じ形の省略がほかに 2 つある。

- 仕様 `:41` の「`--config` がなければ、全 org の上限は main worktree のルートの `ralph.toml` から読む」は、詳細を計画に送る要約なので、直さなくてよい
- `orgWideLimitsHelp` の「Only when --config is not given and the ledger is the main worktree's, ...」は、後ろの「In every other case (--config, a ledger chosen with --state-dir or RALPH_ORG_STATE_DIR or found from the git toplevel, no git repository)」で条件が閉じるので、内容は合う。英文の組み立て(C2-4)はコードの文字列なので、self-review の tech-debt のまとめの行の (c) に入れる

### V2-3: `AGENTS.md` の repo map が、全 org の上限の判定も `reserve.go` にあるように読める(LOW、self-review C2-7)

`AGENTS.md:90` は「envelope validation (per-org `max_seats`; org-wide `max_orgs` / `max_total_seats` and scope reservations in `reserve.go`)」と書く。判定の関数 `ValidateOrgWideCapacity` と `validateMaxOrgs` は `envelope.go:92` と `:107` にある。`reserve.go` にあるのは数え方(`RunningOrgs` `:233`、`TotalActiveSeats` `:265`)と予約(`NormalizeReservePaths`、`reservationDecision` など)。repo map は grep で場所にたどり着くための文書なので、/sync-docs で「org-wide `max_orgs` / `max_total_seats` in `envelope.go`, their counts and scope reservations in `reserve.go`」の形に直すことを勧める。

### V2-4: tech-debt の行 173 の中の食い違いと、行 176 の古いカバレッジの値(LOW、self-review C2-6、C2-5)

- 行 173 の (a) は F-9 を「... and nothing says so」と書く。一方、同じ行の判断の欄は「the help text and the `/org` skill state them」と書き、skill は 76d1cf1c から「末尾に `/` がないパスはファイルなので、`--reserve internal/auth` が守るのは `internal/auth` という名前のファイルだけ」と書いている(`SKILL.md:223-225`)。言いたいのは入力したときに警告が出ないことなので、「and nothing warns when the path is typed」に直すと食い違わない
- 行 176 の (b) の「`idempotentRespawn` (87.5%)」は、975df92b が枝を足す前の値。/test の cycle 2 の値に差し替えるか、関数名だけにする
- 行 144・170・172・174 は cycle 1 の V-2、V-4、V-5、V-7、V-8 を指す。この報告は付録 A に同じ番号で残したので、指す先は変わらない。`verbs_test.go:3084` のコメントの「verify V-2」も同じ
- self-review が提案した C2-1〜C2-7 のまとめの行は、まだ台帳にない。/sync-docs が足す前提で、V2-1 の記録もそこに入れる

### V2-5: self-review の報告の末尾に空行が 1 つ増えた(LOW、記録だけ)

a919082a の `docs/reports/self-review-2026-10-08-org-limits-reserve.md` は改行 2 つで終わるので、`git diff --check 51855166...HEAD` が「new blank line at EOF」で rc 2 を返す。runner と CI はこの検査を回さないので、merge は止めない。直すなら最後の空行を消すだけで、判定の行には触れない。/sync-docs のコミットで消せる。

### cycle 1 の指摘の現況

| V | この回の判定 | 根拠 |
| --- | --- | --- |
| V-1 | 解消 | skill の `:207-210` が「台帳を git の toplevel から決めたとき(bare リポジトリの linked worktree)」を並べ、help も「found from the git toplevel」を書く(`cli/org.go:363`)。言い回しの残りは V2-2 |
| V-2 | 解消(文書)。挙動は tech-debt の行 172 | skill の `:190-195` が補償の例外を書く。pane の経路は c3a95c48 のテスト(`verbs_test.go:3092`)が今の挙動を固定し、行 172 の (c) と (d) が直し方を書く |
| V-3 | 解消 | quality-gates の 2 面が `max_orgs`、`max_total_seats`、予約の重なりを書く。「recorded in manifest」の例外は V2-1 |
| V-4 | 繰り延べ(tech-debt の行 174 の (b)) | `envelope_summary.go` と `doctor.go` は変わっていない |
| V-5 | 解消(文書)。コードの直しは行 174 の (a) | skill の `:233-236` が、`--reserve` だけでは `{{SCOPE}}` が既定の文言になることと、`--scope` も渡す回避を書く |
| V-6 | (1)〜(3) は解消。(4) は既存で、行 166 の (b) にある | (1) `--config` の説明から spawn と start の文が外れた。(2) `orgWideLimitsHelp` の頭が「`ralph org spawn` and `ralph org start` check」になった。(3) `status --help` に `Long` が入った。(4) start の `--scope` の help は今も「`--scope ralph org spawn --scope`」と崩れて出る |
| V-7 | 解消 | 行 170 の (a) に `reserveAgain` が入り、(c) に `verbs.go` の 1,875 行(実測と一致)が入り、きっかけは「the trigger is not spent」と書き直された |
| V-8 | 解消 | 行 144 に、`withMainWorktreeOrgLimits` の読み元と、main の `ralph.toml` が壊れているときに止まる場合が入った |
| V-9 | 解消 | plan の `:151-153` がチェック済み |

### /sync-docs に渡すもの

1. V2-1: plan の進捗の節に 1 行(digest の外)。tech-debt のまとめの行の (a) に入れる。quality-gates の例外の文は任意
2. V2-2: `templates/base/ralph.toml:42-43` のコメントに「neither --state-dir nor RALPH_ORG_STATE_DIR is set」を足す
3. V2-3: `AGENTS.md:90` の係り方
4. V2-4: 行 173 の (a) の文言、行 176 の (b) のカバレッジの値、C2-1〜C2-7 のまとめの行
5. V2-5: self-review の報告の末尾の空行

## Observational checks

実バイナリ(`go build ./cmd/ralph` で scratchpad に作ったもの)で、herdr と agmsg を PATH から外して(`PATH=/usr/bin:/bin`)確かめた。手順と出力は evidence のログにある。

- `ralph org spawn --help`、`start --help`、`status --help` の出力は、上の Documentation drift の表と V-6 の現況のとおり
- 動いていない leader への `--reserve`(本番の経路、`--dry-run` なし): scratch の state dir に、org-b の seat-1 の `spawned`、org-a の leader の `spawned`、org-a の `disbanded` の 3 行を用意した。idempotent の戻りは Phase 1 のロックの中で driver を呼ぶ前に返るので、herdr と agmsg は呼ばれない
  - `--config` で `max_orgs = 1`: `org: max_orgs 1 reached: org_id "org-a" is not running and 1 orgs are (org-b); a finished org frees its slot ...` で rc 1。台帳は拒否の前後で `cmp` が一致し、receipt のファイルはできなかった。`status --org-id org-a` は leader を `spawned` と出し、`reserved:` の行は出さなかった
  - `max_orgs = 2`: `seat "leader" already spawned (org_id=org-a driver=claude model=sonnet pane_id=legacy-pane)` で rc 0。台帳に `scope_reserved`(`paths=internal/`)が 1 行だけ増え、`status` に `reserved: internal/` が出た

## Coverage gaps

- テストは実行していない(`/test` の担当)。新しいテスト 3 本は中身を読んで判断した。mutation は回していない。self-review C2-3 の `if !seat.Active` を外す mutation も回していない(読む限り、Active な座席はその org を走っている扱いにするので、外しても結果は同じ。未確認です)
- 古い台帳は手で 3 行書いて作った。古い ralph のバイナリが書いた台帳では確かめていない
- 本物の herdr と agmsg では動かしていない。本番の経路で打ったのは、driver を呼ぶ前に返る idempotent の戻りだけ
- 別プロセスでの競合と、bare リポジトリの linked worktree から CLI を打つ場合は、cycle 1 と同じく再現していない(tech-debt の行 176 の (c) と (d))

## Verdict

- Verdict: pass
- Verified: plan の承認の digest の一致。AC1〜AC15 の実装の位置とテストの対応を HEAD で見直したこと。975df92b と afcbc6c2 の修正が、動いていない leader の予約の前に、ロックの中の events と main worktree の上限で max_orgs を判定すること。`validateMaxOrgs` への切り出しが `ValidateOrgWideCapacity` の条件と文言を変えないこと。実バイナリで、その拒否が台帳と receipt に何も書かず、空きがあれば予約を 1 行だけ書くこと。`run-static-verify.sh` が rc 0 で終わること(gofmt、go vet、golangci-lint、staticcheck、check-sync、check-skill-sync、check-pipeline-sync、check-template-purity、secret scan)。skill の 4 面が同じ内容であること。cycle 1 の V-1、V-2、V-3、V-5、V-6 の (1)〜(3)、V-7、V-8、V-9 が解消したこと
- Partially verified: AC1 の「`rejected` が残る」は、古い台帳の動いていない leader の枝では成り立たない。AC14 を優先した正しい選択と判断し、計画の進捗に記録することを勧める(V2-1)。AC4 と AC15 はテストの中身とコードで確かめ、実行は /test に任せる。文書の LOW の V2-2〜V2-5 は /sync-docs で直せる。どれも merge を止めない
- Not verified: テストの実行、mutation、別プロセスでの競合、古い ralph が実際に書いた台帳、本物の herdr と agmsg での動作

## 付録 A: cycle 1 の verify レポート(fbb04f83 時点の原文)

cycle 1 の本文を、見出しを 1 段下げただけでそのまま残す(`## Verdict` が 2 つにならないようにするため)。tech-debt の行と `verbs_test.go` のコメントは、ここの V 番号を指している。`file:line` は fbb04f83 時点のもの。`templates/base/ralph.toml` の判定の訂正は、上の「cycle 1 の判定の訂正」に書いた。

- Date: 2026-10-08
- Plan: docs/plans/active/2026-10-08-org-limits-reserve.md
- Verifier: verifier subagent (Claude Opus 5.5)、パイプライン cycle 1(`cycle-count.json` は 1)
- Scope: 仕様への適合(AC1〜AC15、仕様 `docs/specs/2026-10-07-org-multi-org-director.md` の FR-3 と受け入れ条件 96〜97 行)、静的解析、文書のずれ。対象は `git diff 51855166...HEAD`(HEAD 8dd19634、30 ファイル、+2948/-74)。コードの commit は 620458d7(設定)、11fc2261 と 8fe95acd(org の層)、78e46f36(CLI)、fd3e3b47(文書)、8dd19634(self-review F-1 と F-3 の文言の直し)。テストは実行していない(`/test` の担当)。テストは中身を読んで、各 AC の場合を押さえているかを判断した
- Evidence: `docs/evidence/verify-2026-10-08-org-limits-reserve.log`(`docs/evidence/*.log` は gitignore の対象なので手元にだけ残る)。runner 自身のログは `docs/evidence/verify-2026-10-08-114200.log`

### Spec compliance

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

### Static analysis

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

### Documentation drift

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

#### V-1: main の `ralph.toml` を読まない場合の列挙に、bare リポジトリの linked worktree が抜けている(LOW)

skill の `:199-202` は、打った場所の設定を使う場合を「`--config` を渡したときと、台帳を flag か env で決めたとき、git の外では」と並べる。コードは、source が `git-main-worktree` 以外ならすべて main を読まない(`statedir.go:81-83`、`cli/org.go:148-155`)。ここには `git-toplevel`(main worktree を決められない bare リポジトリの linked worktree、`statedir.go:47-51`、`:69-70`)も入る。`TestMainWorktreeRoot` の「git-toplevel」の場合がこれを固定している。help の `orgWideLimitsHelp` の括弧書き「(no --state-dir or RALPH_ORG_STATE_DIR)」(`cli/org.go:366-367`)も同じ穴を持つ。/sync-docs で、skill の列挙に「台帳を git の toplevel から決めたとき(bare リポジトリの linked worktree)」を足すことを勧める。help は文字列なのでコードの変更になる。tech-debt に送るか、cross-review の判断に回す。

#### V-2: 「同時に打った spawn でも超えない」は、自分の close の失敗の補償には当たらない(LOW)

skill の `:185` は「どちらも台帳のロックの下で判定するので、同時に打った `spawn` でも超えない」と書く。spawn どうしについては正しい(AC4)。ただし、補償は上限も予約の重なりも見ずに書き戻す。

- `CloseDeferredSelfWorkspace` の補償(`reopenWorkspace`、`reactivateSeat`、`reserveAgain`、`verbs.go:1082-1111`)は、`disbanded` から補償までの間にほかの org が枠か同じ範囲を取ると、`max_orgs` を 1 つ超え、予約が重なる。`reserveAgain` の doc(`:1215-1224`)、plan のリスク、テスト `..._RiskWindowClearedByRetry` はこの点を書いているが、skill には書いていない
- `CloseDeferredSelfPane`(`verbs.go:1010-1034`)の `reactivateSeat` も、止めた座席を上限を見ずに戻す。そのため、窓の間に別の座席が立つと `max_total_seats` を 1 つ超えうる(`max_seats` も同じで、これは前からある)。plan のリスクが挙げるのは disband の経路だけで、こちらはコードを読んで分かったことでテストはない。workspace は開いたままなので `max_orgs` は影響を受けない

/sync-docs で、skill の `:185` の後ろに「ただし、自分の pane や workspace の最後の close が失敗して座席や workspace を戻したときは、上限と予約を見ずに戻すので、打ち直しの `disband` まで 1 つ超えるか重なることがある」の 1 文を足すことを勧める。

#### V-3: quality-gates の envelope validation の行が `max_seats` だけを書く(LOW)

`docs/quality/quality-gates.md:77` と `templates/base/docs/quality/quality-gates.md:76` は、`ralph org spawn` が見る項目を「model pool / role pool / `max_seats`」と書く。今の spawn は、このほかに `max_orgs`、`max_total_seats`、予約の重なりを見て、拒否を `rejected` で記録する(`spawn.go:1063-1074`)。この 2 ファイルは `check-sync.sh` の KNOWN_DIFF なので、2 つとも直す必要がある。/sync-docs で直せる。

#### V-4: leader のプロンプトの `{{ENVELOPE}}` と doctor の要約は `max_seats` だけを出す(LOW、判断が要る)

`envelope_summary.go:36` と `doctor.go:788` は `max_orgs` と `max_total_seats` を出さない。leader は、自分の org の外の枠をプロンプトから知らず、拒否の文で初めて知る。足すならコードの変更なので、/sync-docs では tech-debt の候補として記録するのがよい。

#### V-5: `--reserve` だけを渡した leader のプロンプトの `{{SCOPE}}` が空になる(LOW、self-review F-4、plan の進捗の (c))

`spawn.go:809` と `:1715` は役割のプロンプトに `p.Scope` だけを渡す。`internal/org/prompts/leader.md:4` の `scope:` は、`--reserve` だけのときは既定の文言になる。skill の `:222` は「`--reserve` を渡した spawn は、autonomous の `--scope` 必須のゲートを満たす」とだけ書き、プロンプトに範囲が出ないことは書いていない。/sync-docs で tech-debt の行に送り、skill のその行に 1 文足すことを勧める。

#### V-6: help の文言(LOW、self-review F-11 は未対応)

- 永続フラグ `--config` の説明(`cli/org.go:46`)に spawn と start だけに関わる文が入り、`status`、`stop` などすべての org の動詞の help に出る
- `orgWideLimitsHelp`(`:363`)は「spawn checks」で始まり、`ralph org start --help` にもそのまま出る
- `ralph org status --help` は `reserved:` の行と JSON の `reservation` に触れない。skill(`:167`、`:226-227`)には書いてある
- start の `--scope` の help が「`--scope ralph org spawn --scope`」と崩れて出るのは前からある(plan の進捗の (d)。usage の backtick を pflag が値の名前と読むため)

どれもコードの文字列なので、tech-debt に送るか cross-review の判断に回す。

#### V-7: tech-debt の行 170 の見直しのきっかけが、この PR で来ている

行 170(`docs/tech-debt/README.md:170`)のきっかけは「The next change to `CloseDeferredSelfPane`, `CloseDeferredSelfWorkspace`, `reactivateSeat`, or `reopenWorkspace`」で、この PR は `CloseDeferredSelfWorkspace` を変えた(`verbs.go:1111`)。(a) C2-1(補償が driver の呼び出しの前に読んだ台帳の写しで決める)は、新しい `reserveAgain` にも当たる(self-review F-5。同じ org_id を窓の間に `--reserve` で立て直すと、古い予約が新しい予約を上書きする)。(c) C2-5(`verbs.go` を 800 行の目安に収めるため、最後の close のまとまりを別のファイルに移す)はされず、`verbs.go` は base の 1,847 行から 1,875 行になった。/sync-docs で、行 170 の (a) に `reserveAgain` を足し、きっかけを次の変更に掛け直すことを勧める。

#### V-8: tech-debt の行 144 の一部が狭まり、別の影響が 1 つ増えた

行 144(`:144`)は、config と state dir の解決の規則が違うことを書く。この PR で、spawn と start は、台帳が main worktree のものなら `max_orgs` と `max_total_seats` を main の `ralph.toml` から読むようになった。行の残り(`max_seats`、`model_pool`、`[org.roles]`、teardown の動詞)は変わらない。一方、main の `ralph.toml` が読み込みに失敗すると、linked worktree 側の設定が正しくても spawn と start が止まる(`cli/org.go:157-160`、テスト `TestOrgStart_MainWorktreeRalphTomlLoadError`)。行の「壊れた `ralph.toml`」の影響の範囲が広がった。行のきっかけ(`resolveOrgConfig` か `ResolveOrgStateDir` の変更)は、文字どおりには来ていない(どちらも変わらず、`MainWorktreeRoot` は隣に足した新しい関数)。/sync-docs で行 144 に、この 2 点を 1 文ずつ足すことを勧める。

#### V-9: plan の進捗のチェックボックス

`:151` の「Review artifact created」は未チェックだが、self-review のレポートは d8ec84da で作られている。「Verification artifact created」(`:152`)はこのレポートで満たされる。/sync-docs か /pr で直せる。

#### /sync-docs に渡す tech-debt の候補

self-review が挙げた F-5、F-8、F-9、F-10 と、plan の進捗の (a)〜(d)(self-review の対応づけでは (a) は F-10、(c) は F-4)。このほか self-review の Coverage gaps にある、予約のパスを大文字小文字を区別して比べる件(macOS の既定のファイルシステムでは `Internal/` と `internal/` は同じ場所を指すが、重ならないと判定される)。plan に書かれていない扱いで、予約は助言なので、tech-debt に記録する程度でよいと判断した。

### Observational checks

実バイナリ(`go build ./cmd/ralph` で scratchpad に作ったもの)で、herdr と agmsg を PATH から外し(`PATH=/usr/bin:/bin`)、scratch の state dir に対して `--dry-run` の spawn と status を打った。台帳には、org-a の leader が `internal/auth/` を予約して立っている状態を 3 行の JSONL で用意した。手順と出力は evidence のログにある。

- org-b の leader の `--reserve internal/auth/token.go`、`internal/`、`.` は、どれも `reservation overlaps the reservation of running org_id "org-a": <path> overlaps internal/auth/` で rc 1。`internal/authz/` は通り、dry-run の trail が `scope_reserved`(dry_run)、`spawn_started` の順で始まった
- `/etc/`、`internal/../x`、空、`a,b` は入力検査の文で rc 1 になり、台帳に何も書かなかった。leader 以外の `--reserve` も同じだった
- `--scope` も `--reserve` もない autonomous の leader は「requires --scope or --reserve (--reserve on the leader seat only; …)」で拒否された。8dd19634 の F-3 の直しが出ている
- org-a の leader に違う一覧(`docs/`)を渡した dry-run は「already reserves internal/auth/ … ralph org disband --org-id org-a」で拒否され、dry-run の `rejected` が 1 行増えた。本番の経路では `idempotentRespawn` が `rejected` を書かない。dry-run には idempotent の戻りがない(`spawn.go:378-381` の doc のとおり)。dry-run の行は Roster に数えないので、座席の状態には効かない
- `status --org-id org-a` は表の下に `reserved: internal/auth/` を出し、`--json` は `"reservation": ["internal/auth/"]` を出した。予約のない org-b は `no seats` だけを出した
- `--config` で `max_orgs = 1` にすると、org-c の dry-run は `max_orgs 1 reached: org_id "org-c" is not running and 1 orgs are (org-a); a finished org frees its slot … ralph org disband --all for every org` で拒否された。`max_total_seats = 1` でも同じ形で拒否された。`max_orgs = 0` は `org: load config: [org].max_orgs must be >= 1, got 0` で、spawn の前に止まった
- `ralph org start --help` と `ralph org spawn --help` の出力は V-1 と V-6 のとおり

### Coverage gaps

- テストは実行していない(`/test` の担当)。各 AC のテストは中身を読んで判断した。mutation は再現していない
- AC4 の同時実行のテストは、同じプロセスの goroutine で flock を競わせる。別プロセスの競合は再現していない(既存の `max_seats` のテストと同じ範囲)
- V-2 の pane の経路(`CloseDeferredSelfPane` の補償で `max_total_seats` を 1 つ超える)は、コードを読んで分かったことで、テストはない
- 本物の herdr と agmsg では動かしていない。本番の spawn(dry-run でないもの)は、Go のテストの fake を読んだだけで確かめた
- bare リポジトリの linked worktree で、main の `ralph.toml` を読まないことは、`TestMainWorktreeRoot` の単体の場合とコードで確かめた。CLI からは打っていない

### Verdict

- Verdict: pass
- Verified: AC1〜AC15 の実装の位置とテストの対応(上の表)。plan の承認の digest の一致。`run-static-verify.sh` が rc 0 で終わること(gofmt、go vet、golangci-lint、staticcheck、check-sync、check-skill-sync、check-pipeline-sync、check-template-purity、secret scan)。`/org` skill の 4 面が同じ内容であること。self-review の F-1(director の語)と F-3(scope のゲートの文)が 8dd19634 で直っていること。実バイナリの dry-run で、重なりの拒否、入力検査、scope のゲート、上限の拒否と案内の文、status の表示が仕様のとおりに出ること
- Partially verified: AC4 と AC15 はテストの中身とコードで確かめ、実行は /test に任せる。AC12 の文書は、V-1(bare リポジトリの場合の列挙の抜け)と V-2(補償の窓の断り)が残る。そのほかの文書は V-3(quality-gates)、V-5(`{{SCOPE}}`)、V-7・V-8(tech-debt の行)、V-9(チェックボックス)を /sync-docs で直せる。V-4 と V-6 はコードの文字列なので、tech-debt に送るか cross-review で判断する。どれも merge を止めない
- Not verified: テストの実行、mutation、別プロセスでの競合、V-2 の pane の経路の再現、本物の herdr と agmsg での動作
