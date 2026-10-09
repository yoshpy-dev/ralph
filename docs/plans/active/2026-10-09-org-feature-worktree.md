# org-feature-worktree

- Status: Approved
- Approved: 2026-10-09 sha256:56e435bfa976
- Owner: Claude Code
- Date: 2026-10-09
- Related request: 機能ごとの org と director の仕様(`docs/specs/2026-10-07-org-multi-org-director.md`)の 4 段目(FR-4 機能ごとの org、FR-11 の `/org` skill の書き換え)。ユーザーは「続けてください」と言った(2026-10-09)
- Related issue: N/A
- Type: feat
- Branch: feat/org-feature-worktree

## Objective

承認済みの分割計画から、機能ごとに worktree とブランチを 1 つずつ作り、その中で headless の leader を立てる `ralph org start --plan <分割計画> --feature <slug>` を足す。担当範囲は引数ではなく分割計画から読み、分割計画の本文が承認のあとに変わっていれば start を拒否する。

あわせて、編成パターン(Solo・Leaded・Parallel)を廃止する。leader の雛形と `/org` skill を「機能ごとの org」(leader・implementer 1 席・reviewer 1 席、worktree・ブランチ・PR を 1 つずつ)に書き換え、小さい変更は org を使わない標準フロー(`/plan` → `/implement`)に回す。

今の `ralph org start <task>` は `--cwd` で渡した場所に leader を立てるだけで、worktree もブランチも作らない。担当範囲は `--scope` / `--reserve` の引数で、どの機能のための org かは台帳に残らない。

## Scope

- 分割計画(新しいファイル `internal/org/split.go`):
  - 置き場所は台帳の下の `splits/<id>.md`(`.harness/state/org/splits/<id>.md`、git の外)。`--plan` のパスがここを指さなければ拒否する。`<id>` はファイル名から `.md` を除いたもの
  - 形式: 先頭に `- Status:` と `- Approved: <日付> sha256:<digest>` の行、見出し `## Features` の下に機能ごとの `### <slug>` の節を並べる。各節は `- Type:`(省略時は feat)、`- Reserve:`(必須、カンマ区切り、3 段目の予約のパスの規則)、`- Depends on:`(任意、同じ分割計画の slug か `none`)と、残りの本文(目的と受け入れ条件)を持つ
  - 読み込みの検査: `## Features` がない、機能が 0 個、slug の形が違う(英小文字・数字・`-`)、slug の重複、`Type` がブランチの型の一覧にない、`Reserve` がない・パスの規則に反する、`Depends on` が分割計画にない slug を指す、のどれでも拒否する
  - 承認の検査: `Status` が `Approved` で、`- Approved:` の `sha256:` の値が今の本文の digest と一致すること。digest は `scripts/plan-visual.sh digest` と同じ規則(`- Status:` / `- Approved:` / `- Branch:` の行と `## Progress checklist` の節を除き、行頭の `- [x]` を `- [ ]` として読む SHA-256 の先頭 12 桁)を Go で実装する
  - digest が読まない書き方は分割計画では拒否する(Codex plan advisory の指摘 2): 先頭の 1 行ずつ以外の `- Status:` / `- Approved:` の行、すべての `- Branch:` の行、`## Progress checklist` の見出し、行頭の `- [x]` / `- [X]`。こうしておけば、承認のあとに書き換えられる本文は digest に必ず入る。承認の digest は `scripts/plan-visual.sh digest` のまま使える
- `ralph org start --plan <path> --feature <slug>`(`internal/cli/org.go` と新しいファイル `internal/org/feature.go`):
  - `--plan` と `--feature` はそろって使う。どちらか一方だけ、または `--cwd` / `--scope` / `--reserve` / `--allow-unscoped` / 位置引数の task との併用は拒否する。`--plan` なしの `ralph org start <task>` は今のまま残す
  - org_id は既定で機能の slug。`--org-id` を渡せばその値を使う
  - 手順: (1) 入力と分割計画の検査(副作用なし)。(2) 台帳をロックなしで読み、spawn と同じ判定(`max_orgs`、`max_total_seats`、予約の重なり、下の結びつきの検査)を先にかける。通らなければ worktree を作らずに拒否する(`rejected` は書かない)。(3) main worktree のルートで `scripts/ralph-worktree.sh ensure --id org-<org_id> --kind org --branch <type>/<slug> --path .claude/worktrees/org-<org_id> --canonical-ref split:<id>#<slug> --cleanup-policy manual` を実行する。ensure は clean な default branch から作り、同じ id・path・branch の記録があれば既存の worktree を返す。(4) leader 座席を spawn する(cwd は worktree、予約は機能の `Reserve`、scope の説明は分割計画の id と slug と予約のパス、task は機能の本文に分割計画のパス・digest・worktree・ブランチ・依存を添えたもの)。spawn はロックの下で (2) の判定をやり直す
  - worktree の使い回しの検査(Codex plan advisory の指摘 1): (3) の前に `ralph-worktree.sh state-path org-<org_id>` の記録を読み、記録があれば、`canonical_ref` が `split:<id>#<slug>` で、`worktree_path` とブランチが今回と同じで、その worktree が実際にそのブランチをチェックアウトしているときだけ使い回す。どれかが違えば拒否し、`./scripts/ralph-worktree.sh cleanup --id org-<org_id>` で消すか別の org_id を使うよう案内する。disband のあとも worktree は残るので、別の分割計画が同じ slug で古い worktree とそのコミットを引き継がないようにする。同じ分割計画の同じ機能は、承認し直したあとでも使い回せる
  - (2) を通ったあと (4) が競合で拒否されたときは、worktree とブランチが残る。エラー文に、打ち直せば同じ worktree を使うことと、要らなければ `./scripts/ralph-worktree.sh cleanup --id org-<org_id>` で消せることを書く
  - ensure が失敗したとき(main のチェックアウトが default branch でない・clean でない、スクリプトや jq がない)は、スクリプトのエラー文に「main worktree を clean な default branch にして打ち直す」を添えて終了コード 1。台帳には何も書かない
- 機能と org の結びつき(`internal/org/reserve.go`、`internal/org/spawn.go`):
  - `--plan` で立てた org の予約の記録 `scope_reserved` に、結びつき(分割計画の id、機能の slug、digest、ブランチ)を `paths=a/,b/ split=<id> feature=<slug> digest=<hex> branch=<b>` の形で足し、worktree のパスはイベントの `Worktree` フィールドに入れる。`key=value` の形でない最初の語(補償の注記 `restored: …`)で読むのを止める。今の `reservedPathsFromDetails` は最初の空白で切るので、古いバイナリも予約のパスを今どおり読める
  - 読み口は `ActiveReservation` と並ぶ関数を 1 つ足す(後の段で結びつきだけを読むため)
  - 判定(ロックの下の `reservationDecision`): 結びつきを渡した spawn は、その org に予約があれば、パスと結びつき(worktree を含む)がすべて同じときだけ通し(記録は増やさない)、違えば拒否する。予約がなく、その org が走っている(動いている座席か閉じていない workspace がある)ときも拒否する。昇格したセッションの leader の org、`--plan` なしで立てた org、`--reserve` だけで予約した org がここで止まる。エラー文に、別の org_id を使うか `ralph org disband --org-id <id>` してから立て直すことを書く
  - 結びつきを持つ org に、結びつきなしで予約を渡した spawn(`ralph org spawn --id leader --reserve …`)も、パスが同じでも拒否する
  - 自分の pane か workspace の close が失敗したときの補償(`internal/org/verbs.go` の `reserveAgain` と `releasedReservation`): 予約を書き戻すとき、パスだけでなく結びつきと worktree も写す
- 表示: `ralph org status` に、結びつきがあれば `feature: <split id>/<slug> branch <branch> worktree <path>` の行を `reserved:` の行の次に出す。`--json` には `feature` のキー(split、feature、digest、branch、worktree)を足す
- leader の雛形(`internal/org/prompts/leader.md`): 既定の編成を implementer 1 席・reviewer 1 席とし、編成パターンの記述を外す。機能ごとの org の進め方を書く: 機能の計画を worktree の `docs/plans/active/` に書く → implementer と reviewer を立てて進める → reviewer が通したら、`scripts/secret-scan-branch.sh` があれば `--strict` を通してから push し、`gh pr create` で PR を 1 本作る(`/pr` skill は使わない。worktree と local branch を消すため)→ worktree とブランチは消さない → report → disband
- `/org` skill(4 面: `.claude/skills/org/SKILL.md`、`.agents/skills/org/SKILL.md`、`templates/base/` の 2 面): 「編成パターン」の節を「機能ごとの org」に書き換える(分割計画の形式と例、承認の記録のしかた、`start --plan`、merge のあとの worktree の後始末、小さい変更は標準フロー)。動詞の表の `start` の行、「Leader 運用 2 経路」、統括のサイクルの 1 番、座席内 fan-out の「編成パターンの座席数」の文を直す
- 文書: `README.md` の org の節、`AGENTS.md` の repo map の `internal/org/` の行、仕様の FR-4 の下に「4 段目で決めたこと」

## Non-goals

- 結びつきのある org に、active な leader 座席がないまま非 leader の座席を spawn するのを断ること(consult の判定で 7 段目に送る)。この場合でも director からの `send --to leader` は pane のない leader で失敗するので、director 側の取り違えは起きない。7 段目の「承認まで implementer の spawn を断る」と同じ判定点で入れる
- 承認方式(`[org.approval].mode`)と、機能の計画の承認まで implementer の spawn と実装の TASK を断ること(7 段目)。4 段目の leader は機能の計画を書くが、人の承認を待たずに進む。7 段目までは split に近い動きになる
- 分割計画を書く・承認を記録する動詞や skill(8a 段目の director)。4 段目では、人が形式に沿って書き、`scripts/plan-visual.sh digest` の値を `- Approved:` に書く
- `Depends on` の順を start で強制すること。leader の task に依存を書くだけにする
- 既定の編成を機構で強制すること(仕様は「既定」)。leader の雛形と skill に書く
- disband や PR の merge で worktree とブランチを自動で消すこと。merge のあとに人が `ralph-worktree.sh cleanup --id org-<org_id>` で消す
- PR の前に main へ rebase してゲートを取り直すこと、検証した head と base の SHA を残すこと(7 段目)
- `ralph status`(全 org の表示)に結びつきを出すこと
- `--plan` なしの `ralph org start <task>` と、現セッションを昇格させる経路を廃止すること。どちらも director の配下に入らない org として残す
- codex の headless leader が sandbox の中から `git push` と `gh pr create` を打てるかの確認(未確認のまま known gap に書く)

## Assumptions

- 1 段目で台帳は main worktree 起点の共通の場所になったので、linked worktree の中の leader も、main のチェックアウトから打つ start も、同じ台帳と同じ `splits/` を見る
- `scripts/ralph-worktree.sh` と `scripts/plan-visual.sh` は `templates/base/scripts/` にもあり、`ralph init` した repo にも入っている
- `ralph-worktree.sh ensure` は cwd のチェックアウトの今のブランチが default branch で clean であることを確かめる(`validate_clean_base`)。そのため Go から呼ぶときは main worktree のルートを cwd にする
- 昇格したセッションの leader は台帳に `leader` 座席の `spawned` を持たない。その org が走っていることは、ほかの座席か workspace で分かる
- `scope_reserved` は org のイベントで状態のイベントではないので、Details を足しても座席の判定は変わらない

## Affected areas

- 新しいファイル: `internal/org/split.go`(分割計画の読み込み・検査・digest)とテスト、`internal/org/feature.go`(start --plan の手順と worktree の作成)とテスト
- `internal/org/reserve.go`(結びつきの記録と読み口、判定)、`internal/org/spawn.go`(`SpawnParams` に結びつき、ロックの下の判定、dry-run の予測)、`internal/org/verbs.go`(補償で結びつきを写す)とテスト
- `internal/cli/org.go`(`--plan` / `--feature`、引数の排他、status の表示)とテスト
- `internal/org/prompts/leader.md`
- `.claude/skills/org/SKILL.md` と 3 つの写し、`README.md`、`AGENTS.md`、仕様の FR-4

## Visual review

- ページ: `.harness/state/plan-visual/org-feature-worktree.html`(図 1 全体、図 2 start --plan の順序と止まる場所、図 3 分割計画の形式と台帳の記録、図 4 org_id の状態ごとの結果)
- セルフチェック: 全体と全体図(`--fragment overview`)を `plan-visual.sh shot` で撮って確かめた。図 1 の「作る」の矢印が worktree の見出しを横切り、「cwd=worktree」の文字が leader の箱に触れていたので直した。図 2 の `ralph-worktree.sh` の見出しが箱からはみ出していたので箱を広げ、撮り直した。Codex plan advisory の 2 件を入れて、図 2 の②と④の拒否の文を直し、撮り直した

## Design decisions

- 結びつきは予約の記録 `scope_reserved` に入れる(consult の判定、案 Y)。別のイベントにすると、3 段目で 4 回往復した自分の close の補償に、もう 1 種類の記録を足すことになる。予約と同じ記録にすれば、補償が予約を書き戻すときに結びつきも戻る。ただし今の `reserveAgain` はパスから Details を作り直しているので、結びつきも写すように直す
- 拒否の判定は、`start --plan` を打つ時点の検査(結びつきのない走っている org には立てない)だけにする(consult の判定)。仕様と council の結論が指すのは「登録時に検査する」で、4 段目の登録は `start --plan`
- 分割計画は台帳の下の `splits/` に限る。id が一意に決まり、台帳には id だけを書けばよく、後の段の director が一覧できる
- 結びつきが同じかどうかは、分割計画の id、slug、digest、ブランチ、worktree、予約のパスがすべて同じかで決める。分割計画を書き直して承認し直したあとに同じ org を立て直すときは、disband してから start する
- PR は leader が `gh pr create` で作る(consult の判定)。`/pr` skill は worktree と local branch を消すので、worktree の中で動く leader には使わせない
- digest は Go で実装し、テストで `scripts/plan-visual.sh digest` の出力と突き合わせる。start のたびにスクリプトを呼ぶ形にすると、awk と shasum が要る環境にさらに依存する
- digest が読まない書き方は、分割計画では拒否する(Codex plan advisory の指摘 2、ユーザーが「計画を直す」を選んだ)。分割計画だけ別の digest の規則にする案は、人が承認の値を出す道具を別に用意する必要があるので採らない
- worktree を使い回すのは、ralph-worktree.sh の記録の `canonical_ref`・パス・ブランチと、実際のチェックアウトが今回と同じときだけにする(Codex plan advisory の指摘 1)。disband で worktree を消す案は、未 push のコミットを消しうるので採らない
- 先に台帳をロックなしで読んで判定するのは、通らない start で worktree とブランチを作らないため。最後の判定は spawn のロックの下で行うので、この先読みは結果を変えない

Critical forks: None(どの選択も 1 スライスの手戻りで戻せる。`start <task>` を残すこと、分割計画の置き場所、PR を作る主体は承認のときに確かめてもらう)

## Acceptance criteria

- [x] AC1: 分割計画の読み込みが、`Type` の既定(feat)、`Reserve` の正規化、`Depends on` の参照を扱い、`## Features` がない・機能が 0 個・slug の形が違う・slug の重複・知らない `Type`・`Reserve` がないか規則に反する・知らない slug への `Depends on` を、それぞれ拒否する
- [x] AC2: digest の Go 実装が、`- Status:` / `- Approved:` / `- Branch:` の行、`## Progress checklist` の節、行頭の `- [x]` / `- [X]`、最後の改行の有無、CRLF の行を含む本文で、`scripts/plan-visual.sh digest` と同じ値を返す。分割計画の読み込みは、先頭の 1 行ずつ以外の `- Status:` / `- Approved:`、`- Branch:`、`## Progress checklist`、行頭の `- [x]` / `- [X]` を拒否する
- [x] AC3: 承認済みの分割計画で `ralph org start --plan <path> --feature a` を打つと、`.claude/worktrees/org-a` に `<type>/a` のブランチの worktree が default branch からでき(`ralph-worktree.sh` の記録 `org-a`)、その worktree を cwd にした leader 座席が spawn され、org `a` が機能の `Reserve` を予約し、予約の記録に結びつき(分割計画の id、slug、digest、ブランチ、worktree)が残る。leader の task に機能の本文が入る(仕様の受け入れ条件 FR-4 の前半)
- [x] AC4: `Status` が `Approved` でない、`- Approved:` に digest がない、本文が承認のあとに変わった、のどれでも start は拒否され、worktree も台帳の記録もできない(仕様の受け入れ条件 FR-4 の後半)。承認のあとに機能の本文へ `- Branch:` の行を足した場合も拒否される
- [x] AC5: `--plan` が台帳の `splits/` の下を指さない、知らない `--feature`、`--plan` と `--feature` の片方だけ、`--cwd` / `--scope` / `--reserve` / `--allow-unscoped` / task の引数との併用は、どれも拒否される。`--plan` なしの `ralph org start <task>` は今のテストのまま通る
- [x] AC6: 同じ `--plan` と `--feature` で start を打ち直すと、同じ worktree を使い、予約の記録は増えない。同じ org_id に別の機能か、承認し直した分割計画(digest が違う)で start を打つと拒否される。disband のあと、別の分割計画の同じ slug で start を打つと、残っている worktree を使い回さずに拒否される。記録のブランチと worktree の実際のチェックアウトが違うときも拒否される。同じ分割計画の同じ機能なら、disband のあと承認し直しても同じ worktree を使う
- [x] AC7: 結びつきのない走っている org の org_id への `start --plan` は拒否され、worktree も台帳の記録もできない。昇格したセッションの leader の org(leader 座席なしでほかの座席が動いている)、`--plan` なしの start の org、`--reserve` だけの org の 3 つで確かめる。結びつきのある org に結びつきなしの予約を渡した spawn も拒否される
- [x] AC8: ロックなしの先読みで `max_orgs`、`max_total_seats`、予約の重なりに当たる start は、worktree を作らずに拒否される。先読みを通ったあとロックの下で拒否されたときは、worktree が残り、エラー文が打ち直しと `ralph-worktree.sh cleanup` を示す
- [x] AC9: main のチェックアウトが default branch でないか clean でないとき、start はスクリプトのエラー文と打ち直しの案内を出して終了コード 1 になり、台帳には何も書かない
- [x] AC10: 自分の pane か workspace の close が失敗して予約を書き戻すとき、結びつきと worktree も戻る
- [x] AC11: `ralph org status` が `feature:` の行を出し、`--json` が `feature` のキーを持つ。結びつきのない org では出ない
- [x] AC12: leader の雛形が既定の編成(implementer 1 席・reviewer 1 席)と機能ごとの org の進め方(機能の計画、push の前の secret scan、`gh pr create`、`/pr` を使わない、worktree を消さない、report、disband)を書き、Solo・Leaded・Parallel に触れない
- [x] AC13: `/org` skill(4 面)の「編成パターン」の節が「機能ごとの org」になり、動詞の表・2 経路・サイクルが合っている。`README.md`、`AGENTS.md`、仕様の FR-4 が合っている。`./scripts/check-skill-sync.sh` と `./scripts/check-sync.sh` が通る
- [x] AC14: 実機で 1 回、分けた herdr のもとで、小さな repo の分割計画から claude の leader を `start --plan` で立て、leader が worktree の中で implementer と reviewer を 1 席ずつ立てることを確かめる(仕様の受け入れ条件 FR-4 の「implementer と reviewer が 1 席ずつ立つ」)

## Implementation outline

1. S1(分割計画): `split.go` の読み込み・検査・承認の検査・digest と、スクリプトとの突き合わせのテスト(AC1、AC2)
2. S2(結びつき): 予約の記録への結びつきの追加と読み口、ロックの下の判定、dry-run の予測、補償で写すこと(AC6 と AC7 の org 層、AC10)
3. S3(start --plan): `feature.go` の手順(先読み、worktree の作成、spawn)と CLI のフラグ・排他・エラー文、status の表示(AC3〜AC9、AC11)
4. S4(文書): leader の雛形、`/org` skill(4 面)、README、AGENTS.md、仕様(AC12、AC13)。S1〜S3 の挙動が固まってから書く
5. S5(実機): 分けた herdr で AC14 を確かめ、`docs/evidence/` に残す

## Verify plan

- Static analysis checks: `./scripts/run-static-verify.sh`
- Spec compliance criteria to confirm: AC1〜AC14。仕様の FR-4、FR-11 の `/org` skill の部分、受け入れ条件(98〜99 行)
- Documentation drift to check: `ralph org start --help`、`/org` skill の `start` の行と「機能ごとの org」の節、leader の雛形、README の org の節、AGENTS.md の repo map
- Evidence to capture: verify のレポート、実機の記録(`docs/evidence/`)

## Test plan

- Unit tests: 分割計画の読み込みと各拒否、承認の検査、digest(固定の入力と、スクリプトとの突き合わせ)、結びつきの Details の書き出しと読み戻し(注記つき、古い形)、`reservationDecision` の結びつきの判定(同じ・違う・予約なしで走っている・結びつきなしの予約を渡す)
- Integration tests: `StartFeature` を fake の herdr / agmsg と fake の worktree 作成で通し、spawn の引数(cwd、予約、scope、task)と台帳の記録を確かめる。先読みでの拒否で worktree 作成が呼ばれないこと。打ち直しの冪等。補償で結びつきが戻ること。CLI の `--plan` / `--feature` の排他、status の `feature:` と JSON。一時的な git の repo に本物の `ralph-worktree.sh` を置き、worktree とブランチができること、main が dirty なら失敗して台帳に何も書かないこと
- Regression tests: `./scripts/run-test.sh`。3 段目の予約と補償、`start <task>`、spawn の idempotent のテストが通ること
- Edge cases: CRLF の分割計画、`- [X]`、`## Progress checklist` が最後の節、機能の本文の中の `- Branch:` と 2 つ目の `- Status:`、`Depends on: none`、slug と同じ名前のブランチがすでにある、disband のあとの別の分割計画の同じ slug、worktree の中で別のブランチをチェックアウトした場合、`--org-id` で slug と違う org_id を使う、予約の Details に注記と結びつきの両方がある
- Evidence to capture: test のレポート

## Risks and mitigations

- main のチェックアウトで別の作業をしていると start が通らない(ensure が clean な default branch を求める): エラー文で main を clean にして打ち直すよう案内する。director は main のチェックアウトから打つ前提にする
- 先読みとロックの下の判定の間に別の org が枠か範囲を取ると、worktree とブランチだけが残る: 打ち直せば同じ worktree を使う。要らなければ `ralph-worktree.sh cleanup` で消せるとエラー文に書く
- digest の規則が Go とスクリプトでずれる: 突き合わせのテストを CI で回す。どちらかを直したらテストが落ちる
- 承認のあとに、digest が読まない行で leader への指示を書き換えられる: その書き方を分割計画の読み込みで拒否する(AC2、AC4)
- disband のあとに残った worktree を、別の分割計画の同じ slug が引き継ぐ: 記録の `canonical_ref`・パス・ブランチと実際のチェックアウトを照合し、違えば拒否する(AC6)。ralph-worktree.sh の記録の形が将来変わると読めなくなるので、読めない記録も拒否として扱う
- 予約と結びつきを 1 つの記録にまとめたので、予約の判定を変えると結びつきにも響く: 読み口を 1 つにし、判定のテストで両方を確かめる
- 既定の編成は雛形に書くだけなので、leader が座席を増やすことがある: `max_seats` が上限になる。9 段目の試行で見る
- 分割計画は git の外にあるので、`.harness/state/` を消すと失われる: 台帳の結びつきに id と digest が残る。分割計画を書き直すのは 8a 段目の director の仕事として残す
- leader が機能の計画を人の承認なしに進める(7 段目まで): 7 段目までの間の動きとして rollout に書く
- codex の leader が sandbox のせいで push か PR を作れない: 未確認。雛形には、push か `gh pr create` に失敗したら人に上げるよう書く

## Rollout or rollback notes

- バイナリの更新で効く。`--plan` を使わなければ今の動きは変わらない
- 7 段目が入るまで、`start --plan` で立てた leader は機能の計画を人の承認なしに進める
- 戻すときはこの PR を revert する。古いバイナリは予約の Details の後ろの語を読まない(最初の空白で切る)ので、結びつきのある記録もパスの予約として読む。worktree は `ralph-worktree.sh cleanup --id org-<org_id>` で消す

## Open questions

- なし

## Progress checklist

- [x] Plan reviewed
- [x] Plan approved
- [x] Branch created
- [x] Implementation started
- 2026-10-09: S1(a321d1a7)分割計画の読み込み・検査・digest。決めたこと: `SplitPlansDirIn` を足した。`LoadSplitPlan` もファイル名(`<id>.md`)を検査する。`ResolveSplitPlanPath` はファイル自体の symlink も解決し、`splits/` の外を指すものを拒否する。`- Approved:` の digest の形が違えば読み込みでは拒否せず、`CheckApproved` が「digest がない」と返す(Draft の `- Approved: TBD` を読めるように)。slug `none`、`Depends on` の空の項目や重複、最初の `### <slug>` より前のフィールドの行も拒否する。digest は本物のスクリプトと 114 個の入力で一致を確かめた
- 2026-10-09: S2(9d7a85ae)予約の記録に結びつき。決めたこと: 一部だけ壊れた結びつきの記録は「どれとも一致しない結びつき」として読み、その org への予約は disband まで拒否する。結びつきの worktree は `filepath.Clean` で正規化する。入力検査は制御文字も拒否する
- 2026-10-09: S3 は org 層の S3a(f27188d7)と CLI の S3b(216b2cdd)の 2 つのコミットに分けた。S3a: 先読みは Spawn の入力検査(`checkSpawnInput`)と既存の leader の判定(`idempotentRespawnDecision`)を切り出して共有し、`TestSpawnPrecheckErr_MatchesSpawn` で先読みと Spawn のエラーが一致することを確かめる。RepoRoot は symlink を解決して使う。worktree の使い回しの検査は kind も比べ、記録の worktree のディレクトリがないときも拒否する。implementer が未追跡の新規テストファイルの 1 行を `sed -i` で直した(追跡済みのファイルではなく、U+FFFD の検査は 0 件)。S3b: `org.FeatureRepoRoot(dir, source)` を足し、`newOrgSpawnRuntime` を解決と構築に分けた。`--plan` / `--feature` の空白の値も拒否する。start の `--scope` の usage にあったバッククォートで `--help` が崩れていたので直した(3 段目の進捗の (d))
- 2026-10-09: S4(aaaef61a)雛形・skill・文書。計画からのずれ: org の締めの順を「座席を stop → report → 機能の計画を archive → コミット → secret scan → push → `gh pr create` → disband」にした(`ralph org report` は worktree の `docs/reports/` に書くので、PR のあとに打つと未追跡のファイルが残り、`ralph-worktree.sh cleanup` が拒否する。implementer の指摘)。雛形に「`/plan` skill は使わない」を足し、skill に `cleanup` で `--force-branch` が要る場合を書いた
- 2026-10-09: S5(実機、`docs/evidence/org-feature-worktree-live-2026-10-09.md`)。run 2 で、leader が雛形どおり `--cwd .` で立てた implementer と reviewer の pane が、worktree ではなく herdr サーバーの cwd で動いた(ralph が相対の `--cwd` を herdr にそのまま渡していた。4 段目の前からある穴)。計画からのずれとして、af138af6 で spawn の入口で相対の `--cwd` を絶対パスにしてから herdr・agmsg・台帳に渡すよう直した(consult の判定、案 A。雛形の `--cwd .` はこの直しで正しくなるので変えない)。run 3 で、台帳は main の 1 つ、3 席の pane の cwd と台帳の worktree はどれも機能の worktree の絶対パスになり、leader は計画・実装・レビュー・report・archive・push(remote がないので失敗)まで雛形の順に進めた
- 2026-10-09: self-review(cycle 1)の MEDIUM 2 件と LOW を直した(8aae7ce2、2c1bb13b)。計画からのずれ: (a) 機能の slug と `start --plan` の org_id を 20 文字までにした(`maxFeatureOrgIDLen` = herdr の agent 名の上限 32 − 1 − `implementer` の 11)。それより長いと、leader が立ったあとに implementer の spawn が拒否されるため(M1)。雛形の手順 2 に、座席の `--id` を `implementer` と `reviewer` にすることを足した。(b) 出荷する `/org` skill から、まだない director の説明を外した(M2)。(c) `ensure` の失敗の案内を、スクリプトの文で原因ごとに分けた(L1)。(d) `ralph org status` は壊れた結びつきの記録に `(incomplete record)`、`--json` に `"incomplete": true` を付ける(L4)。`FeatureBinding.Complete` は型と同じ `reserve.go` に置いた。self-review をかけ直して Merge 可(1ffa4f15)
- 2026-10-10: cross-review(cycle 1)は ACTION_REQUIRED 2 件(`docs/reports/cross-review-triage-org-feature-worktree.md`)。ユーザーが「直す」を選んだ。d49bbc34: (1) leader の task に台帳の絶対パスの行 `- 台帳:` を足し、leader は `ralph org` のコマンドすべてに `--state-dir` を付ける(既定の台帳でも付ける。`start` に渡した `--state-dir` や `RALPH_ORG_STATE_DIR` は herdr の pane に届かず、pane の ralph が古い版でも台帳がそろう)。(2) 分割計画のコードフェンスの中の行は見出しにもフィールドにもせず本文に残し、閉じていないフェンスは拒否する(digest が読まない行の拒否はフェンスの中でも続ける)。07d38e6d: `--state-dir` で指した台帳が main worktree の既定の台帳と同じなら、全体の上限は今どおり main の `ralph.toml` から読む(`org.LedgerMainWorktreeRoot`。leader が `--state-dir` を付けても、3 段目の「feature branch の `ralph.toml` で全体の上限を変えられない」を保つため)。パイプラインは 2 回目(上限)
- 2026-10-09: 実装中に見つけて送るもの(sync-docs で tech-debt へ)。(1) start を打った ralph と、leader が pane の中で呼ぶ ralph の版が違うと台帳が分かれうる(run 1 で pane の `ralph` が Homebrew の v5.1.0 に解決され、worktree の中に台帳ができた)。(2) reviewer が書いたレポートをコミットしないまま止められると、scaffold の Stop hook が `wip: checkpoint before session end` でコミットし、PR に `wip:` の名前で入る。(3) `start <task>` の leader や昇格した leader が task worktree の中で `ralph org report` を打つと、同じく未追跡のファイルが残る(S4 の implementer の指摘)。(4) 同じ機能の `start --plan` を同時に 2 つ打つと、両方が先読みを通って `ensure` まで進み、後の方のエラーに「main を clean にして打ち直す」の案内が付く(この場合は的外れ)。`ralph-worktree.sh` にはロックがない
- 2026-10-09: sync-docs(cycle 1)。計画からのずれ: verify の V-1・V-2 に合わせて `internal/cli/org.go` の help の文字列を直した(`--plan` の usage に `--allow-unscoped`、start の Long と `--org-id` の usage に org_id の 20 文字の上限。あわせて spawn と start の `--cwd` の usage に、相対パスは打った場所から絶対パスにすると足した。af138af6 の挙動で、skill にだけ書いてあった)。実機の記録の Run 3 の出力に、抜けていた `hint:` の行を足した(V-4)。仕様の「4 段目で決めたこと」に、20 文字の上限、status の `feature:` の行と `(incomplete record)`、worktree の使い回しの検査の kind、skill に director の説明を書かないことを足した。`docs/tech-debt/README.md` は、上の「実装中に見つけて送るもの」の (1)〜(4) を 4 行、self-review の L5・N1・N2 と verify の V-3 と help の 20 を 1 行、test report の T-1〜T-7 を 1 行に足し、行 182・184・185 の関数名(`idempotentRespawnDecision`)、C2-1 の解消、F-2・F-6 のトリガーが満たされたまま残る旨を直した。(2) の「Stop hook」は、実際には `SessionEnd` の `session_end_summary.sh` で、台帳の行はこちらの名前で書いた
- [x] Review artifact created
- [x] Verification artifact created
- [x] Test artifact created
- [ ] PR created
