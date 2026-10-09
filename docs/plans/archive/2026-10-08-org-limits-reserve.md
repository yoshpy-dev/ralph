# org-limits-reserve

- Status: Approved
- Approved: 2026-10-08 sha256:1a165903b5df
- Owner: Claude Code
- Date: 2026-10-08
- Related request: 機能ごとの org と director の仕様(`docs/specs/2026-10-07-org-multi-org-director.md`)の 3 段目(FR-3 上限と担当範囲の予約)。ユーザーは「続けてください」と言い、予約は「任意にする」を選んだ(2026-10-08)
- Related issue: N/A
- Type: feat
- Branch: feat/org-limits-reserve

## Objective

全 org をまたぐ上限 `[org].max_orgs`(既定 10)と `[org].max_total_seats`(既定 30)を、座席を立てるときに台帳の共通のロックの下で守らせる。あわせて、org を立てるときに担当範囲(ディレクトリの接頭辞と明示したファイル)を予約できるようにし、走っている他の org の予約と重なれば立てるのを拒否する。disband すると予約は解ける。

今は org ごとの上限 `[org].max_seats`(既定 5)しかなく、org の数と全体の座席の数には上限がない。`--scope` は自由文の説明で、spawned の詳細に残るだけで、org どうしの担当の重なりは誰も見ていない。

## Scope

- 設定(3 面を揃える): `[org].max_orgs`(既定 10)と `[org].max_total_seats`(既定 30)を `internal/config/config.go` に足し、1 以上で検証する。`templates/base/ralph.toml`、`scripts/ralph-config.sh`(`RALPH_ORG_MAX_ORGS`・`RALPH_ORG_MAX_TOTAL_SEATS`)、`internal/config/defaults_sync_test.go` の同期の検査を揃える
- 「走っている org」の判定(`internal/org/`): org の最後の `disbanded` より後の記録だけを見て、次のどれかに当たる org を走っているとする。
  - 動いている座席がある(`Roster` の Active)
  - 作ったまま閉じていない workspace がある
  - 予約(新しいイベント `scope_reserved`)がある

  `rejected` だけの org、座席が全部止まって workspace も予約もない org は数えない
- 上限の判定(`internal/org/envelope.go` と `internal/org/spawn.go`): `Spawn` がロックを持ったまま台帳を読み直す今の区間で、`ValidateSpawnCapacity` に並べて 2 つを判定する。
  - max_orgs: まだ走っていない org_id への spawn で、走っている org の数が `max_orgs` 以上なら拒否する
  - max_total_seats: 全 org の動いている座席の数が `max_total_seats` 以上なら拒否する

  拒否は今の `reject` で `rejected` イベントと receipt を書く。すでに立っている座席への spawn は、今どおり上限の判定の前に返す(idempotent)。ただし予約を渡した leader の spawn は、返す前に予約を判定する(下の「予約」の項)
- 全体の上限の読み元(Codex plan advisory の指摘 2): `--config` がなく、台帳の置き場所を git の main worktree から決めたとき、`max_orgs` と `max_total_seats` は、その main worktree のルートの `ralph.toml` から読む(なければ既定値)。打った場所の `./ralph.toml` やサブディレクトリによって、同じ台帳に違う上限が掛からないようにする。`--config` を渡したとき、台帳を flag か env で指定したとき、git の外では、今どおり打った場所の設定を使う。org ごとの `max_seats` とほかの設定の読み方は変えない
- 予約(`internal/org/`): `SpawnParams` に予約するパスの一覧を足す。
  - パスの規則: repo のルートからの相対で書く。末尾が `/` ならディレクトリの接頭辞、そうでなければファイル。`.` は repo 全体。絶対パス、`..` を含むもの、空は、ロックを取る前の入力検査で拒否する(今の入力検査と同じく `rejected` は書かない)
  - 重なり: 2 つのディレクトリはどちらかが他方の接頭辞なら重なる。ディレクトリとファイルは、ファイルがそのディレクトリの下にあれば重なる。2 つのファイルは同じなら重なる。比べるのはパスの区切りの単位(`internal/auth/` と `internal/authz/` は重ならない)
  - 受け付けるのは leader の座席の spawn だけ。leader 以外の座席に予約を渡したら拒否する
  - 予約は org 単位で、その org にまだ予約がないときに書く。すでに予約がある org に同じ一覧を渡したら通し(新しい記録は書かない)、違う一覧なら拒否する
  - すでに立っている leader への spawn に予約を渡したとき(Codex plan advisory の指摘 3): 既存の座席を返す前に予約を判定する。まだ予約がなければ重なりを調べて予約し、同じ一覧なら通し、違う一覧なら座席の状態を変えずに拒否する。予約のない再試行は今どおり既存の座席を返す
  - ロックを持ったまま、走っている他の org の予約と重なるかを調べ、重なれば拒否する(相手の org_id と重なったパスを理由に書く)。重ならなければ org のイベント `scope_reserved`(詳細に正規化したパスの一覧)を書いてから、今の spawn の手順に進む
  - spawn がそのあと失敗しても予約は残る(disband で解く)
  - 予約を渡した spawn は、autonomous の permission で `--scope` を求めるゲートを満たしたとみなす
- disband: `disbanded` より前の予約は効かなくなる(イベントから導く)。`orgsToDisband`(`internal/org/verbs_all.go`)は、最後の `disbanded` のあとに `scope_reserved` がある org も対象に含める
- 自分の workspace の close が失敗したときの補償(Codex plan advisory の指摘 1): 2 段目の `CloseDeferredSelfWorkspace` は、close が失敗すると workspace と自分の座席を「動いている」に戻す。このとき、その org が `disbanded` の前に持っていた予約も `scope_reserved` で書き戻す。`disbanded` から補償までの数十秒に、ほかの org が枠や範囲を取ることは防がない(下のリスクに書く)
- CLI(`internal/cli/org.go`): `ralph org start` と `ralph org spawn` に `--reserve <path>`(複数回指定できる)を足す(spawn は leader の座席のときだけ受け付ける)。上限で拒否したときのエラー文に、終わった org を `ralph org disband --org-id <id>` か `--all` で片付けると枠が空くことを書く。`ralph org status` に、その org の予約を 1 行出す
- 文書: `/org` skill(4 面)の動詞の表と運用の節、`README.md`、`templates/base/ralph.toml` の説明

## Non-goals

- 座席が予約の外へ書くのを止めること。範囲の外への変更は、今の watchdog の scope_change(worktree の `git status` の変化で ALERT を出す)で知らせる。watchdog は変えない
- 予約を必須にすること(ユーザーが「任意にする」を選んだ)。4 段目の `ralph org start --plan` では、分割計画から必ず予約する
- 予約を変える動詞(`ralph org reserve` など)。変えたいときは disband して立て直す
- 分割計画と、そこから予約を読むこと(4 段目)
- `ralph status`(全 org の表示)に予約を出すこと。director の段(8 段目)で必要になれば足す
- director を数から外すこと。director はまだなく、台帳の座席だけを数えるので、仕様の「director は数えない」は今は自然に満たされる

## Assumptions

- 1 段目で台帳は main worktree 起点の共通の場所になり、`withManifestLock` は state dir の `manifest.lock` を flock するので、全 org の判定を 1 つのロックで直列にできる
- `rejected` は `Roster` が見る状態のイベントに入る(`internal/org/seat.go`)。そのため「最後の disbanded のあとに状態のイベントがある」で数えると、上限で拒否された org_id が枠を食う。上の 3 条件で数える(consult の指摘)
- 古い ralph の `disband` は `disbanded` を書いたが workspace を閉じなかった。最後の `disbanded` より後の記録だけを見るので、そうした org は走っている org に数えない(2 段目の `disband --all` は、こうした workspace を閉じる対象にしている)
- `internal/config` の読み込みは知らないキーを無視するので、新しい設定を書いた `ralph.toml` を古いバイナリで読んでも失敗しない
- `scope_reserved` は org のイベント(座席 id なし)で、状態のイベントではない。`Roster` の座席の判定は変わらない

## Affected areas

- `internal/config/config.go` とそのテスト、`internal/config/defaults_sync_test.go`、`templates/base/ralph.toml`、`scripts/ralph-config.sh`
- `internal/org/envelope.go`(上限の判定)、`internal/org/spawn.go`(`SpawnParams`、ロックの中の判定、予約の記録、既存の leader への予約)、`internal/org/seat.go`(イベントの種類)、新しいファイル `internal/org/reserve.go`(パスの規則、重なり、走っている org の判定)とそのテスト、`internal/org/verbs_all.go`(`orgsToDisband`)、`internal/org/verbs.go`(補償で予約を書き戻す)
- `internal/cli/org.go`(`--reserve`、エラー文、`ralph org status`、全体の上限の読み元)とそのテスト
- `.claude/skills/org/SKILL.md` と 3 つの写し、`README.md`

## Visual review

- ページ: `.harness/state/plan-visual/org-limits-reserve.html`(図 1 全体、図 2 座席を立てるときの判定、図 3 走っている org と予約の重なり)
- セルフチェック: 全体と全体図(`--fragment overview`)を `plan-visual.sh shot` で撮って確かめた。図 3 の「どれか」の文字が矢印と重なっていたので位置を直し、撮り直した。Codex plan advisory の 3 件を入れて、全体図に `verbs.go` と CLI の上限の読み元を足し、図 2 の「立っている」と注記を直して撮り直した

## Design decisions

- 予約は任意にする(ユーザーの選択。2026-10-08)。付けない org は誰とも重ならない。今の `ralph org start` の使い方と scripts が壊れない。機能ごとの org は 4 段目で分割計画から必ず予約する
- 予約は spawn の同じロックの中で、上限と一緒に判定して記録する。別の動詞にすると、start と予約の間に他の org が割り込める
- 「走っている org」は、最後の `disbanded` より後の、動いている座席・閉じていない workspace・予約のどれかで決める。状態のイベントで数える案は、拒否された org_id が枠を食うので採らない。動いている座席だけで数える案は、stop だけした org が予約と workspace を持ったまま枠を空けるので採らない
- 全体の上限は、`--config` がなければ台帳と同じ main worktree のルートの `ralph.toml` から読む(Codex plan advisory の指摘 2、ユーザーが「計画を直す」を選んだ)。台帳が 1 つなら上限も 1 つにする。台帳に上限の値を保存して不一致を拒否する案もあるが、最初に書いたプロセスの値が残り続け、設定を変えたときに台帳を直す手順が要るので採らない
- 自分の workspace の close が失敗したときは、補償で予約も書き戻す(Codex plan advisory の指摘 1)。close を確かめるまで枠と予約を持ち続ける状態を作る案もあるが、成功した close は台帳に書けない(プロセスが終わる)ので、時間で区切るなどの別の決まりが要る。その org は締めの disband の途中で作業を終えているので、数十秒の窓はリスクとして書くにとどめる
- 予約を受け付けるのは leader の座席だけにする。仕様は「org の start のとき」に予約するとしている。昇格したセッションの leader は `ralph org spawn --id leader` で自分を登録するので、spawn にも同じフラグを出す

Critical forks: 予約を任意にするか必須にするかの 1 件で、ユーザーが「任意にする」を選んだ

## Acceptance criteria

- [x] AC1: 走っている org が `max_orgs` 個あるとき、まだ走っていない org_id への spawn(`ralph org start` を含む)は拒否され、台帳に `rejected` が残る。走っている org への spawn は `max_orgs` の影響を受けない
- [x] AC2: 全 org の動いている座席が `max_total_seats` 個あるとき、新しい座席の spawn は拒否され、台帳に `rejected` が残る。すでに立っている座席への spawn は、今どおり既存の座席を返す
- [x] AC3: 走っている org として数えるのは、最後の `disbanded` より後に、動いている座席・閉じていない workspace・予約のどれかがある org だけ。`rejected` だけの org、座席が全部止まり workspace も予約もない org、古い ralph が workspace を閉じずに disband した org は数えない
- [x] AC4: 同時に打った 2 つの spawn が、`max_orgs` と `max_total_seats` を超えない(今の `max_seats` の競合のテストと同じ形で確かめる)。2 つの org が同時に重なる範囲を予約しようとしても、片方だけが通る
- [x] AC5: org A が `internal/auth/` を予約しているとき、`internal/auth/token.go`、`internal/`、`.` のどれかを予約する org B の start は拒否され、理由に org A と重なったパスが出る。`internal/authz/` は通る
- [x] AC6: パスの規則。末尾の `/` はディレクトリの接頭辞、それ以外はファイル、`.` は repo 全体。絶対パス、`..` を含むもの、空は、ロックを取る前に拒否される。重なりはパスの区切りの単位で比べる
- [x] AC7: すでに予約がある org に同じ一覧を渡すと通り、記録は増えない。違う一覧を渡すと拒否される。予約のない spawn は、予約のある org にも今どおり立つ。leader 以外の座席に `--reserve` を渡すと拒否される
- [x] AC8: org を `disband` すると、ほかの org が同じパスを予約できる。予約だけを持つ org も `disband --all` の対象に入る
- [x] AC9: `--reserve` を渡した autonomous の spawn は、`--scope` がなくても通る
- [x] AC10: `ralph org status` が、その org の予約を表示する
- [x] AC11: `max_orgs` と `max_total_seats` の既定は 10 と 30 で、0 以下は設定の読み込みで拒否される。3 面(Go の既定、`templates/base/ralph.toml`、`scripts/ralph-config.sh`)が `defaults_sync_test.go` で揃っている
- [x] AC12: `/org` skill(4 面)と `README.md` が上限と予約を説明している。`./scripts/check-skill-sync.sh` と `./scripts/check-sync.sh` が通る
- [x] AC13: `--config` を渡さずに、main worktree のルートの `ralph.toml` で `max_orgs = 1` にしたとき、サブディレクトリからも、`ralph.toml` の違う linked worktree からも、2 つ目の org は拒否される(CLI のテストで確かめる)。`--config` を渡したときはその設定を使う
- [x] AC14: すでに立っている leader への spawn に `--reserve` を渡すと、その org に予約がなければ予約し(重なれば拒否)、同じ一覧なら通り、違う一覧なら座席の状態を変えずに拒否される。予約のない再試行は既存の座席を返す
- [x] AC15: 自分の workspace の close が失敗して補償で戻った org は、`disbanded` の前の予約も持ち直す(`scope_reserved` が書き戻され、ほかの org は同じ範囲を予約できない)

## Implementation outline

1. S1(設定): `max_orgs`・`max_total_seats` の 3 面と検証(AC11)
2. S2(org の層): 走っている org の判定、パスの規則と重なり、上限の判定、予約の記録と判定、既存の leader への予約、`orgsToDisband`、scope のゲート、補償で予約を書き戻すこと(AC1〜AC9、AC14、AC15)
3. S3(CLI): `--reserve`、エラー文、`ralph org status` の表示、全体の上限の読み元(AC5・AC7・AC10 の CLI 側、AC13)
4. S4(文書): `/org` skill(4 面)、README(AC12)

## Verify plan

- Static analysis checks: `./scripts/run-static-verify.sh`
- Spec compliance criteria to confirm: AC1〜AC15。仕様の FR-3 と受け入れ条件(96〜97 行)
- Documentation drift to check: `/org` skill の `start` / `spawn` の行、`ralph org start --help` と `ralph org spawn --help`、`templates/base/ralph.toml` の `[org]` の説明
- Evidence to capture: verify のレポート

## Test plan

- Unit tests: パスの正規化と拒否(絶対パス、`..`、空、`.`、末尾の `/`)、重なり(ディレクトリどうし、ディレクトリとファイル、ファイルどうし、区切りの単位)、走っている org の判定(AC3 の各場合)、上限の判定(境界の値)
- Integration tests: `Spawn` を通した拒否と `rejected` の記録、予約の記録・同じ一覧・違う一覧・leader 以外、すでに立っている leader への予約(なし・同じ・違う)、disband のあとの再予約、`disband --all` が予約だけの org を拾うこと、補償で予約が書き戻されること、同時に打った spawn の競合(goroutine)。CLI の `--reserve`、エラー文、`ralph org status` の表示、main worktree の `ralph.toml` の上限がサブディレクトリと linked worktree からも効くこと
- Regression tests: `./scripts/run-test.sh`。今の `max_seats`、scope のゲート、spawn の idempotent のテストが通ること
- Edge cases: 予約だけを残して spawn が失敗した org、古い ralph が disband した org、`max_orgs = 1`、同じ org の 2 回目の start
- Evidence to capture: test のレポート

## Risks and mitigations

- 上限に当たって start が急に通らなくなる: 走っている org の判定を 3 条件に絞り、エラー文で `ralph org disband --org-id <id>` か `--all` を案内する
- 予約を付けない org が、他の org の範囲に書く: 予約は任意なので止められない(ユーザーの選択)。範囲の外への変更は watchdog の ALERT で知らせる。4 段目の機能ごとの org は必ず予約する
- 予約のパスを repo のルートからの相対で書くので、worktree が違っても同じ範囲として比べる: 4 段目で org ごとに worktree が分かれても、同じ repo の同じパスを指す
- spawn が失敗して予約だけが残る: org は走っている扱いになり、枠と範囲を持ち続ける。`disband` で解ける。`ralph org status` に予約が出るので見える
- 自分の disband の途中で close が失敗したときの窓: `disbanded` から補償までの数十秒(herdr の呼び出しの期限 10 秒が最大 3 回)に、ほかの org が枠か同じ範囲を取ると、補償のあとに上限を 1 つ超えるか、範囲が重なる。補償で戻る org は締めの disband の途中で作業を終えているので、打ち直しの disband で解ける。テストで窓を作り、上限を超えた状態と重なりが打ち直しで解けることを確かめる
- 全体の上限を main worktree の `ralph.toml` から読むので、linked worktree のブランチで `ralph.toml` の上限を変えても効かない: 全体の上限は台帳と同じく repo に 1 つ、と `/org` skill と `ralph.toml` の説明に書く。`--config` で明示すれば変えられる

## Rollout or rollback notes

- バイナリの更新で効く。既定の上限(10 org・30 席)は今の使い方では当たらない。`ralph.toml` に書かなければ既定が使われる
- 戻すときはこの PR を revert する。古いバイナリは `scope_reserved` を読まない(状態のイベントではないので、座席の判定は変わらない)。設定のキーは無視される

## Open questions

- なし

## Progress checklist

- [x] Plan reviewed
- [x] Plan approved
- [x] Branch created
- [x] Implementation started
- [x] Review artifact created
- [x] Verification artifact created
- [x] Test artifact created
- [ ] PR created
- 2026-10-08: S1(620458d7)設定。`templates/base/scripts/ralph-config.sh` も同期ゲートの写しなので同じく直した。S2(11fc2261、8fe95acd)org の層。決めたこと: 上限が 0 以下なら拒否する(`max_seats` と同じく fail-closed)。予約の記録は `paths=a/,b.go` の形で、パスにカンマ・空白・制御文字が入ると拒否する。読めない `scope_reserved` は repo 全体として扱う。すでに立っている leader への予約の拒否は `rejected` を書かない(書くと leader の最新の状態が rejected になり、動いているのに inactive と表示されるため)。autonomous の scope のゲートの文に `--reserve` を足した。S3(78e46f36)CLI。全体の上限を main の `ralph.toml` から読むのは spawn と start だけ(ほかの動詞を、main の設定の読み込みエラーで止めないため)。S4(fd3e3b47)文書
- 2026-10-08: cross-review(cycle 1)は ACTION_REQUIRED 1 件(`docs/reports/cross-review-triage-org-limits-reserve.md`)。古い台帳で動いていない leader に `--reserve` を渡すと、`max_orgs` を見ずに予約を書いていた。ユーザーが「直す」を選び、975df92b で、動いていない座席への予約の前に `max_orgs` を判定するようにした。afcbc6c2 で、その判定を `validateMaxOrgs` に切り出して `ValidateOrgWideCapacity` と共有した。AC1 にコードを合わせる直しなので、計画の本文は変えていない。パイプラインは 2 回目(上限)
- 2026-10-09: 975df92b と afcbc6c2 の、動いていない leader への `--reserve` を `max_orgs` で拒否する枝は、`rejected` を書かない。立っている leader への予約の拒否(AC14)と同じで、`rejected` を書くと座席の最新の状態が `spawned` から変わり、予約なしの再試行が既存の座席を返さなくなるため。AC1 の「`rejected` が残る」は新しい座席の spawn に当たる(verify V2-1)。計画の本文は変えていない
- 2026-10-09: cross-review(cycle 2、上限)は WORTH_CONSIDERING 1 件。自分の workspace の close が失敗したときの補償が、close の前に読んだ台帳の写しで予約を書き戻していた(self-review の F-5 と同じ)。ユーザーが「上限を 3 に上げて直す」を選んだ。a94c914f で、補償(座席を戻す、workspace を開き直す、予約を書き戻す)を台帳のロックの下で読み直してから判断するようにした。pane の経路も同じ形にそろえた。cross-review の skill の決まりで `cycle-count.json` は 2 のまま、`RALPH_STANDARD_MAX_PIPELINE_CYCLES=3` でパイプラインを 3 回目として回す
- 2026-10-09: a94c914f の 3 つの補償は、何を「新しい記録」と見るかが揃っていない(cycle 3 の verify V3-1、test T3-1、T3-2)。座席は最新の状態イベント、workspace はその id の最新の workspace イベント、予約は「今の予約がなく、最後の `disbanded` の前の予約が close の前の写しと同じか」で決める。そのため、close を待つ間に同じ org の `disband` だけが打ち直されると、座席と workspace は戻るのに予約は戻らず、org は予約なしで走っている扱いに戻る。`--reserve` なしで立て直されたときは逆向きで、座席は戻らないのに古い予約が立て直した run に書き戻される。どちらもエラーは戻さなかったものを挙げて手で閉じる herdr のコマンドで終わり、打ち直した `disband` で解ける。現状は `TestOrgCloseDeferredSelfWorkspace_NewerRecordsWhileClosing_RulesDiffer` が固定している。AC15 の本文は digest の中なので変えていない。規則を揃えるかは tech-debt に送った
- 2026-10-09: cross-review(cycle 3)は WORTH_CONSIDERING 2 件。close を待つ間の disband の打ち直しで予約が戻らない(V3-1)、pane の経路の disband の補償が予約を戻さない(進捗の (a)、F-10)。ユーザーが「上限を 4 に上げて直す」を選んだ。consult を受けて 83aec44e で予約の規則をそろえた。close 前の写しで、最後の disbanded がこのコマンドのものか(その後ろに立て直しがないか)を確かめる。読み直した台帳でも、その disbanded の後ろに立て直し(`scope_reserved`、`spawn_started`、`spawned`)がなければ、最後の立ち上げの時点の予約を戻す。打ち直しの disband だけなら戻し、立て直しなら `--reserve` の有無によらず戻さない。pane の経路も同じ規則を呼ぶ。`RulesDiffer` のテストは `..._ReservationUnlessStartedAgain` に改名した。パイプラインは 4 回目(`RALPH_STANDARD_MAX_PIPELINE_CYCLES=4`、`cycle-count.json` は 2 のまま)
- 2026-10-08: 実装中に見つけて送るもの(sync-docs で tech-debt へ)。(a) pane だけを後回しにした disband の補償は、予約を戻さない(83aec44e で解決。`CloseDeferredSelfPane` が `reserveAgain` を呼ぶ)。(b) 予約のパスの `*` はそのままファイル名として扱う。(c) 予約だけを渡したとき、役割のプロンプトの `{{SCOPE}}` は空になる。(d) start の `--scope` の help の表示崩れ(既存)
- 2026-10-09: cycle 4 の self-review C4-1 と verify V4-1。`disband --force` で自分の pane の close が失敗し(補償はしない)、同じ pane から止まっている座席へ `stop` を打ち直してその close もまた失敗すると、座席と一緒に、forced disband が解いた予約も戻る。ユーザーが今の挙動を残す(「戻す」)と決めた。`TestOrgCloseDeferredSelfPane_StopRetriedAfterForcedDisband_ReservationRestored` が固定する。この入力の窓は、リスクの節の数十秒ではなく、forced disband から打ち直しの `stop` までにオペレーターがかけた時間になる。その間にほかの org が同じ範囲か枠を取っていれば、戻したあと次の `disband` まで予約が重なるか、上限を 1 つ超える。AC15 の本文とリスクの節は digest の中なので変えていない。doc の言い切りの直しは tech-debt に送った
