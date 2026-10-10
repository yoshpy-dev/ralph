# 機能ごとの org と director

## Summary

複数の機能を並行で頼むとき、機能ごとに org を 1 つ立てる。各 org は leader・implementer・reviewer の 3 席と、worktree・ブランチ・PR を 1 つずつ持つ。herdr の外にいる director が、全 org の進捗の管理、人への相談、報告を受け持つ。

ralph には LLM を使わない制御層(共通の台帳、全 org の上限、担当範囲の予約、受信箱、横断の見張りと停止)を足し、その上に director の役割・skill・起こし口を載せる。org の内側(座席、型付きのメッセージ、品質ゲート)は今の org runtime(`docs/specs/2026-08-01-org-runtime.md`)のものを使う。

この仕様は 2026-10-07 の council(5 体)の結論と、そのあとのユーザーの決定にもとづく。council の結論の全文は `~/.cache/council/20261007-0324-org-of-orgs/conclusion.md` にある(git の外。要点はこの仕様に写した)。

## Background and problem

### Current state

- org_id ごとに herdr の workspace と agmsg の team(`ralph-<org_id>`)が分かれている(`internal/org/spawn.go` の `resolveWorkspace`、`agmsgTeam`)。`ralph org` の動詞はすべて `--org-id` が必須で、全 org をまとめて見られるのは `ralph status --json` だけ(`internal/cli/status.go`)
- org の台帳の既定の置き場所は、1 段目の前は `git rev-parse --show-toplevel` から決まり、linked worktree の中では worktree ごとに別の台帳ができた(`internal/org/statedir.go`)。1 段目の PR で、main worktree のルートから決めるように変え、linked worktree からも main と同じ台帳を使う(計画は `2026-10-07-org-state-dir-common.md`、ブランチ `fix/org-state-dir-common`)
- `ralph org spawn` は worktree を作らず、座席の作業場所は `--cwd` で渡すだけ。編成パターンの Parallel は、1 つの worktree・1 本のブランチ・1 本の PR を複数の implementer で使う
- leader より上の宛先がない。leader の雛形(`internal/org/prompts/leader.md`)は「直せなければ人に上げる」と書くが、その経路は決まっていない。人に届くのは watchdog のデッドマン(`escalations.jsonl`、stderr、macOS の osascript)だけ
- 上限は org ごとの `max_seats`(既定 5)だけで、全 org を合わせた上限はない
- 編成パターン(Solo・Leaded・Parallel)は `/org` skill の文章にあり、leader が選ぶ
- `ralph org send` は宛先を座席台帳から引き、herdr の pane への入力として送る(`internal/org/verbs.go` の `Send`)。今の対話セッションを `/org` で昇格させた leader には pane がないので届かない

### Desired state

- 人が話す相手は director 1 体になる。director は人と話して分割計画を書き、承認を取り、機能ごとの org を起動し、進捗を見て、判断が要るときだけ人に聞き、終わったら報告する
- 各 org は leader・implementer・reviewer の 3 席で動き、自分の worktree・ブランチ・PR を持つ
- LLM が全部止まっても、ralph の仕組みだけで全 org の状態がわかり、止められる。director が応じない件は、期限が来たら人に届く

## Requirements

### Functional requirements

段は導入の順を表す(「Rollout」を参照)。1 段を 1 本の PR にするのを目安にする。

- [ ] **FR-1 共通の台帳(1 段目)**: flag と env がないとき、org の台帳の置き場所を main worktree のルートから決める。ルートは、main worktree の中では `git rev-parse --show-toplevel`、linked worktree の中では `git worktree list --porcelain` の先頭の、bare でない記録とする。linked worktree の中から打った動詞も、main と同じ台帳を使う。詳細は 1 段目の計画(`2026-10-07-org-state-dir-common.md`)に書いた
- [ ] **FR-2 横断の status と stop(2 段目)**: `ralph org stop --all` と `ralph org disband --all` を足す。`--org-id` なしで全 org の座席に効く。一部の座席を止められなかったときは、止められなかった座席を並べて終了コード 1 で終わる。`ralph status` は今のまま全 org を表示する。2 段目で、1 座席の `stop` と 1 org の `disband` も同じ止め方にすると決めた。`stop` は C-c のあと座席の herdr の pane を閉じてプロセスを終わらせ、`disband` は座席を止めたあと org の herdr の workspace を閉じる。pane を閉じられなかった座席は台帳に `stop_failed` を書いて動いているまま残すので、打ち直せば拾える。herdr に繋がらないまま片付けたいときは `--force` で `stopped` を書く。詳細は 2 段目の計画(`2026-10-07-org-stop-all.md`)に書いた
- [ ] **FR-3 上限と担当範囲の予約(3 段目)**:
  - `[org].max_orgs`(既定 10)と `[org].max_total_seats`(既定 30)を、台帳の共通のロックの下で強制する。director は herdr の座席ではないので数えない
  - org の start のときに担当範囲を予約する。担当範囲はディレクトリの接頭辞と明示したファイルで書く。走っている他の org の予約と重なれば start を拒否する。org を disband したら予約を解く
  - 予約が保証するのは「割り当てが重ならないこと」までで、座席が実際に範囲の外へ書くことは止めない。範囲の外への変更は、今の watchdog の検知(ALERT)で知らせる
  - 3 段目で決めたこと: 予約は任意で、付けない org は誰とも重ならない扱いにする。予約は `ralph org start` と leader の `ralph org spawn --id leader` の `--reserve` だけが受け付け、org ごとに 1 つ持つ(同じ一覧なら通り、違う一覧は拒否、変えるときは disband して立て直す)。走っている org は、最後の `disbanded` より後に、動いている座席・閉じていない workspace・予約のどれかがある org とする。`--config` がなければ、全 org の上限は main worktree のルートの `ralph.toml` から読む。詳細は 3 段目の計画(`2026-10-08-org-limits-reserve.md`)に書いた
- [ ] **FR-4 機能ごとの org(4 段目)**:
  - `ralph org start --plan <分割計画> --feature <slug>` は、clean な default branch から worktree とブランチを作り(`scripts/ralph-worktree.sh` を使う)、その中で leader を headless の座席として立てる
  - 担当範囲は引数ではなく、承認済みの分割計画から読む。分割計画の digest が承認時の値と一致しなければ start を拒否する
  - leader は implementer 1 席と reviewer 1 席を立てる。これを既定の編成にする
  - 編成パターン(Solo・Leaded・Parallel)はこの段で廃止する。小さい変更は org を使わない標準フロー(`/plan` → `/implement`)に回す
  - director の配下に置ける leader は headless の座席だけにする。昇格させたセッションの leader を director の配下に置こうとしたら拒否する
  - 4 段目で決めたこと: 分割計画は台帳の下の `splits/<id>.md`(git が無視する場所)に置き、先頭の `- Status:` と `- Approved: <日付> sha256:<digest>`、`## Features` の下の `### <slug>` の節(`- Type:` は省略時 feat、`- Reserve:` は必須、`- Depends on:` は任意)で書く。slug は英小文字・数字・`-` で 20 文字までにした(既定の org_id になり、leader が `implementer` を立てるときの herdr の agent 名 `<org_id>_<seat_id>` の 32 文字に収めるため。`--org-id` も同じ上限)。digest は `scripts/plan-visual.sh digest` と同じ規則で Go で計算し、digest が読まない書き方(先頭の 1 行ずつ以外の `- Status:` / `- Approved:`、`- Branch:`、`## Progress checklist`、行頭の `- [x]` / `- [X]`)は読み込みで拒否する。機能との結びつき(分割計画の id、slug、digest、ブランチ、worktree)は予約の記録に入れ、`ralph org status` が `feature:` の行で出す(キーが欠けた・重複した不完全な結びつきには `(incomplete record)` を付ける)。拒否の判定は `start --plan` を打つ時点の検査だけにした(結びつきのない走っている org には立てない)。結びつきのある org に active な leader がないまま非 leader の座席を spawn するのを断る検査は 7 段目に送った。worktree は `ralph-worktree.sh` の記録の `canonical_ref`・パス・ブランチ・kind と実際のチェックアウトが今回と同じときだけ使い回す。PR は leader が `gh pr create` で作り(`/pr` skill は worktree を消すので使わない)、worktree とブランチは merge のあとに人が `ralph-worktree.sh cleanup` で消すまで残す。`--plan` なしの `ralph org start <task>` と現セッションの昇格は残し、どちらも director の配下に入らない org とする。出荷する `/org` skill には、まだない director の説明を書かない(director が入る段で足す)。cross-review のあとに 3 つ足した。(a) 分割計画のコードフェンス(`` ` `` 3 つ以上か `~` 3 つ以上で、開閉は CommonMark の fenced code block の規則に合わせた)の中の行は、見出しにもフィールドにもせず、機能の中なら本文として残す。閉じていないフェンスは開いた行を指して拒否する。digest が読まない行の拒否はフェンスの中でも続ける。(b) leader のタスクに `- 台帳:` の行(start が使った台帳の絶対パスと、それを指す `--state-dir`)を足し、leader は `ralph org` のコマンドすべてに `--state-dir` を付ける。ralph は start に渡した `--state-dir` も `RALPH_ORG_STATE_DIR` も pane に渡さず、pane の環境は start を打った環境と同じとは限らないため。(c) `--config` がなく、`--state-dir` か `RALPH_ORG_STATE_DIR` が打った場所の repository の main worktree の `.harness/state/org` を指すときも、全 org の上限は main worktree の `ralph.toml` から読む。leader が `--state-dir` を付けても、feature branch の `ralph.toml` で上限を変えられないようにするため。別の台帳を指すときは、打った場所の設定を使う。詳細は 4 段目の計画(`2026-10-09-org-feature-worktree.md`)に書いた
- [ ] **FR-5 受信箱(5 段目)**:
  - `ralph org escalate` で leader から director へ上げる。今の型付きのメッセージと同じ検証(型、本文 2,000 字、証拠はポインタ)を通し、共通の台帳の受信箱ファイルにイベント ID を付けて追記する
  - 1 件の状態は open → acked → resolved と進む。`ralph org inbox ack <id>` と `ralph org inbox resolve <id> --note <ポインタ>` で進める。ack は「読んだ」ことの記録で、解決ではない
  - `ralph org wait --inbox` は、未処理の件が届くと返る(全 org が対象)
  - director から leader へは、今の `ralph org send --to leader` を使う
  - director が無効(FR-9)のとき、escalate は受信箱に記録したうえで、人への経路(FR-6)にすぐ送る
- [ ] **FR-6 横断の見張り(6 段目)**:
  - `ralph org watch --all` を 1 プロセス立て、全 org のパルスと受信箱の期限を見る
  - ack の期限(`[org.director].ack_minutes`、既定 10)と resolve の期限(`[org.director].resolve_minutes`、既定 60)が切れた件は、今ある人への経路(`escalations.jsonl`、stderr、osascript)に直接送る
  - 見張りは 30 秒ごとに生存の印を共通の台帳に書く。印が古いとき(`deadman_minutes` より前)、`ralph org start`・`spawn`・実装の TASK の送信は断る。`ralph org start` は、見張りが動いていなければ起動する
  - 横断の stop(FR-2)は、見張りが動いていなくても使える
  - 見張りを再起動したら、受信箱を読み直して期限を計算し直す
  - 通知を試したこと、人が受け取ったこと、座席を止めたことは、別々に記録する
- [ ] **FR-7 承認と統合検証(7 段目)**:
  - 承認方式は `[org.approval].mode` で選ぶ。`"per-plan"` が既定で、`"split"` に切り替えられる
  - split では、人が承認するのは分割計画(機能ごとの slug・担当範囲・受け入れ条件・依存順)だけで、各 org の中の設計は leader に任せる
  - per-plan では、加えて各 org の leader が書いた機能の計画(その org の worktree の `docs/plans/active/`)を人が承認する。準備ができた計画はまとめて承認できる。承認されるまで、その org では implementer の spawn と実装の TASK の送信を断る。TASK の本文の `PLAN: <path> <sha256>` を、計画の `Status: Approved` と digest に照らす。TASK の再投入や計画の改訂のときも同じ検査をする
  - どちらの方式でも、承認は `/plan` と同じ図解ページと digest で記録する。director は承認を代行しない。人の答えを受けて記録するだけにする
  - 後から入る PR は、main に rebase してゲート(`run-static-verify.sh` / `run-test.sh`)を取り直してから出す。reviewer の結果には、検証した head と base の SHA を残す
  - `ralph doctor` は、GitHub の保護ブランチに「最新でないと merge できない」設定がなければ warn する。`ralph status` は merge の時点の保証の種類(GitHub の保護か、PR 作成時の検査だけか)を表示する
- [ ] **FR-8 merge**: 既定では人が merge する。`[org.director].auto_merge = true` にすると、director が CI の通った PR を merge する。どちらの場合も、director は「merge してよい順」(検証した base が今の main と一致する PR)を報告する
- [ ] **FR-9 director の役割と skill(8a 段目)**:
  - 役割の雛形 `internal/org/prompts/director.md` と `/director` skill を足す(`.claude/skills/`、`.agents/skills/`、`templates/base/` の 2 面、計 4 面)
  - 起動は 2 通りにする。(A) 任意の端末(Orca、普通のターミナル、Claude のデスクトップなど)の対話セッションを `/director` で昇格させ、`ralph org director attach` で共通の台帳に登録する。登録するのは driver、起こし方、push の送り先。(B) `ralph org director start` で headless の director を立てる。既定は herdr の中の専用の workspace(機能の workspace とは別)で動かし、`[org.director].launch_cmd` で別の端末の起動コマンドに差し替えられる
  - `[org.director].enabled` は既定で `true`。有効でも director が登録されていなければ、受信箱の件は FR-6 の期限で人に届く
  - director の仕事は、分割計画の下書き(人と話して書き、承認を取る)、org の start、受信箱の処理、進捗の報告、merge してよい順の報告、`auto_merge` のときの merge、終わったときの `ralph org report`
  - director は実装しない。許すのは、分割計画を書くこと、repo を読むこと、ralph と gh の操作に限る。headless の director の permission mode は `[org.permissions.roles].director` で決める(既定は他の役割と同じく `[org.permissions].default`)
  - driver は claude と codex の両方を使えるようにする
- [ ] **FR-10 起こし口(8b 段目)**:
  - `[org.director].wake` は既定で `["pull"]`。`["push"]` と `["pull", "push"]` も許す
  - pull では、director が `ralph org wait --inbox`(FR-5)で待つ。Claude Code は Monitor で待ち、codex は timeout 付きの wait を繰り返す
  - push では、受信箱に追記されたとき `[org.director].wake_cmd`(コマンドの雛形)を実行する。送り先は attach で登録した値を使う。headless の director が herdr にいるなら herdr の pane への入力を既定の送り方にする。Orca の `orca terminal send` を使う例を recipe に書く
  - push を含むのに `wake_cmd` も送り先もないときは、設定の検証エラーにする
  - どちらの起こし方でも、届かなかった件は FR-6 の期限で人に届く
- [ ] **FR-11 雛形と skill の書き換え**: leader の雛形の「人に上げる」を `ralph org escalate` に置き換える。`/org` skill の編成パターンの節を「機能ごとの org」に書き換える(4 段目)
- [ ] **FR-12 試行と既定値の見直し(9 段目)**: 2 つの org を同時に走らせ、人の介入回数・介入時間・承認待ちの時間・総費用・停止と再開の成否を記録する。その結果で上限と期限の既定を見直す

### Non-functional requirements

- [ ] **安全の不変条件**: 全 LLM を止めても、ralph の仕組みだけで全 org の状態の説明・停止・期限切れの件の人への通知ができる。今の org runtime の不変条件を、全 org に広げる
- [ ] **ツール非依存**: director・leader・座席は、claude と codex のどちらでも動く。Orca などの外部の端末は必須にしない
- [ ] **記録の置き場所**: 記録は共通の台帳(manifest・受信箱・receipts)と git と reports に置く。agmsg と push は通知だけに使う
- [ ] **パリティ**: `templates/base/` とルート、`.claude/skills/` と `.agents/skills/` の同期ゲート(`scripts/check-sync.sh`、`scripts/check-skill-sync.sh`)を通す
- [ ] **入れ方**: 各段は標準フロー(`/plan` → `/implement` → パイプライン → `/pr`)で入れる

## Acceptance criteria

- [ ] Given 3 つの org が linked worktree で動いている、when main のチェックアウトで `ralph status --json` を打つ、then 3 つの org の座席がすべて表示される(FR-1)
- [ ] Given 全 LLM のセッションを止めた、when `ralph org stop --all` を打つ、then 全 org の座席が止まり、止められなかった座席があれば一覧と終了コード 1 が返る(FR-2)
- [ ] Given `max_orgs` 個の org が動いている、when もう 1 つ start する、then 拒否され、台帳に記録が残る(FR-3)
- [ ] Given org A が `internal/auth/` を予約している、when `internal/auth/token.go` を含む org B を start する、then 拒否される(FR-3)
- [ ] Given 承認済みの分割計画、when `ralph org start --plan <path> --feature a` を打つ、then 新しい worktree とブランチができ、その中で leader が headless で動き、implementer と reviewer が 1 席ずつ立つ(FR-4)
- [ ] Given 分割計画の本文が承認後に変わった、when start を打つ、then 拒否される(FR-4)
- [ ] Given director が登録されていない、when leader が escalate する、then ack の期限が切れた時点で人への経路に届く(FR-5、FR-6)
- [ ] Given director が ack したが resolve しない、when resolve の期限が切れる、then 人への経路に届く(FR-6)
- [ ] Given 横断の見張りを kill した、when `ralph org start` を打つ、then 生存の印が古ければ拒否される。見張りを起動し直すと、期限切れの件が人に届く(FR-6)
- [ ] Given `mode = "per-plan"` で、ある org の計画が未承認、when leader が implementer に実装の TASK を送る、then 拒否される(FR-7)
- [ ] Given `mode = "split"`、when 分割計画を承認した、then 各 org の計画の承認なしで TASK を送れる(FR-7)
- [ ] Given `auto_merge = false`(既定)、when PR の CI が通る、then director は merge せず、merge してよい順を報告する(FR-8)
- [ ] Given `wake = ["pull", "push"]` で `wake_cmd` を設定した、when escalate が記録される、then wait が返り、かつ wake_cmd が実行される(FR-10)
- [ ] Given `wake` に push を含むが `wake_cmd` も送り先もない、when 設定を読む、then 検証エラーになる(FR-10)
- [ ] Given `ralph org director start --driver codex --model <slug>`、when 起動する、then director が herdr の専用 workspace で動き、台帳に登録される(FR-9)

## User stories

1. As 開発者, I want to 3 つの機能を director にまとめて頼む, so that 機能ごとの workspace を自分で見て回らずに済む。
2. As 開発者, I want to 外出先からスマホで director と話す, so that 判断が要る点だけを答えて開発を進められる。
3. As 開発者, I want to director が止まっても期限が来たら通知を受ける, so that 黙って止まったままにならない。
4. As テックリード, I want to 承認方式を per-plan と split から選ぶ, so that 機能の中の設計まで見るかどうかを自分で決められる。
5. As 監査する人, I want to 全 org の起動・承認・相談・merge を台帳と reports で追う, so that 事後に誰が何を決めたかを説明できる。

## Constraints

### In scope

- FR-1〜FR-12 の 9 段
- director の役割の雛形、`/director` skill、attach と start、起こし口(pull と push)
- `ralph.toml` の新しい設定(`[org].max_orgs`、`[org].max_total_seats`、`[org.approval]`、`[org.director]`)と、`internal/config` の 3 面の lock-step(`config.go` の `Default()`、`templates/base/ralph.toml`、`scripts/ralph-config.sh`)
- 関係する文書(`/org` skill、leader の雛形、recipe、AGENTS.md の主ループの説明、`docs/quality/` の該当箇所)

### Out of scope

- 金額の上限(座席数とモデルで間接的に抑える。今の org runtime と同じ)
- claude と codex 以外の driver
- 複数のリポジトリにまたがる org
- web のダッシュボード、Slack への通知
- 機能ごとに承認方式を変えること(分割計画の中での上書き)
- 古い台帳(linked worktree の中の `.harness/state/org/`)の移行
- hook による品質ゲートの強制(今の tech-debt に記録済み)
- agmsg・herdr・Orca への上流の変更

## Impact

| Target | Impact | Severity |
|--------|--------|----------|
| `internal/org/statedir.go` | 台帳の置き場所の解決(1 段目) | HIGH |
| `internal/org/verbs.go`、`internal/cli/org.go` | stop・disband の `--all`、escalate・inbox・wait の追加、start の `--plan` / `--feature` | HIGH |
| `internal/org/spawn.go`、`internal/org/manifest.go` | 全 org の上限、担当範囲の予約、worktree の作成、承認の検査 | HIGH |
| `internal/org/watch.go`、`internal/org/watcher.go` | 横断の見張り、受信箱の期限、生存の印 | HIGH |
| `internal/org/protocol/` | escalate の本文の検証、TASK の `PLAN:` の検査 | MEDIUM |
| `internal/org/prompts/` | `director.md` の追加、`leader.md` の書き換え | MEDIUM |
| `internal/config/` と `templates/base/ralph.toml`、`scripts/ralph-config.sh` | 新しい設定と既定値(lock-step のテストを含む) | MEDIUM |
| `internal/cli/doctor*.go` | 保護ブランチの設定、director の設定の検査 | MEDIUM |
| `.claude/skills/org/`、`.claude/skills/director/`(新規)と写し | 編成パターンの廃止、director の手順 | MEDIUM |
| `docs/recipes/` | push の起こし口(Orca の例)、codex の座席の書き込み先 | LOW |

## Dependencies

- herdr(必須): workspace、tab、pane、agent の操作。headless の director の専用 workspace にも使う
- agmsg(必須): org の中の座席と leader のメッセージ。director との通信には使わない
- `gh` CLI: PR の作成、保護ブランチの設定の読み取り、`auto_merge` のときの merge
- Orca(任意): push の送り先の例
- 段の依存: 2・3・5 段目は 1 段目に、4 段目は 1・3 段目に、6 段目は 5 段目に、7 段目は 4 段目に、8a 段目は 5・6・7 段目に、8b 段目は 5・8a 段目に依存する

## Research findings

### Codebase analysis

- org_id ごとの herdr workspace と agmsg team は `internal/org/spawn.go` の `resolveWorkspace` と `agmsgTeam` にある。機能ごとの org はこの単位をそのまま使える
- 全 org の集約は `internal/cli/status.go` にある。他の動詞は `--org-id` が必須(`internal/cli/org.go`)
- 台帳の置き場所の解決は `internal/org/statedir.go` の `ResolveOrgStateDir` 1 か所に集まっている
- `Send` は宛先を `findSeat` で引き、pane のない座席には送れない(`internal/org/verbs.go`)。watchdog は同じ理由で `SendWatchdogAlert` を別に持つ(`internal/org/watch.go`)
- 人への経路は `internal/org/watch.go` の `EscalateFunc`(既定は osascript)と `escalations.jsonl` で、差し替えられる
- 座席の台帳は座席ごとに agmsg の team を 1 つだけ持つ(`internal/org/manifest.go`)。director を上位の agmsg team でつなぐと、この形を変えることになる
- codex の座席は workspace-write の sandbox で動き、cwd の外には書けない(`internal/org/permissions.go`)。1 段目で、leader の座席にだけ `--add-dir` を足す

### Best practices

- Anthropic の orchestrator-worker の型: 独立した文脈で作業させ、上位が統合する。コードは調べものより並列にしにくい
- マルチエージェントの失敗の研究: 役割の逸脱、過剰な調整、終わりの条件がわからないことが主な原因。星型のトポロジー、型付きのメッセージ、期限、決定論の見張りで抑える(今の org runtime の spec の調査と同じ)
- GitHub の保護ブランチの「Require branches to be up to date before merging」と merge queue は、base が進んだ PR の検査のやり直しを強制できる

### Alternatives considered and trade-offs

| Option | Pros | Cons | Adopted |
|--------|------|------|---------|
| Orca の orchestration(run・task・gate・mailbox)に任せる | すでに作られている。モバイルで使える | 状態の記録が ralph と二重になる。Orca が必須になり、ツール非依存の原則に反する | No |
| director を各 org の座席(4 席目)にする | 今の座席の仕組みに載る | herdr の外で人と話すという目的に合わない。org ごとに 1 体要る | No |
| leader → director を上位の agmsg team でつなぐ | agmsg の配送を使える | 座席の台帳の形(team は 1 つ)を変える。正本がメッセージ側に寄る | No |
| leader → director を受信箱ファイル + 動詞で送る | 記録が台帳に残る。LLM が止まっても期限で人に届く | 起こし口を別に作る必要がある | **Yes** |
| 既存の Parallel に reviewer を足して済ませる | 安い。director が要らない | 1 本のブランチ・1 本の PR になり、機能ごとのレビュー・revert・merge の順を分けられない。leader 1 体に全機能が集まる | No |
| 担当範囲の重なりを警告だけにする | 柔軟 | 重なりの判断が LLM に移る。手戻りが作業のあとで見つかる | No(start で拒否) |
| 承認は分割計画だけ(split)を既定にする | 人が読む量と承認待ちが少ない | 同じ範囲の中の設計の違いを人が見られない | No(per-plan を既定、split は設定で選べる) |
| director を既定で無効にする | 立てない人に手間がない | 人の窓口を 1 つにするという目的から遠い | No(既定で有効。登録がなければ期限で人に届く) |

## Design decisions

- 分割計画の置き場所: 共通の台帳の下(`.harness/state/org/splits/<id>.md`、git の外)に置き、承認は digest で記録する。全 org が終わったら `ralph org report` が `docs/reports/` に写す。機能の計画は、その org の worktree の `docs/plans/active/` に置き、機能の PR で archive する。分割計画を docs の PR で main に入れる案は、機能の org を始める前に merge を 1 回待つことになるので採らない
- headless の director は herdr の専用 workspace で動かすのを既定にする。herdr の pane への入力が、そのまま push の起こし口になるから。Orca などは `launch_cmd` と `wake_cmd` で差し替える
- codex の director の pull は、timeout 付きの `ralph org wait --inbox` を繰り返す形にする(Claude Code の Monitor に当たるものが codex にあるかは未確認)
- director の権限: 実装はさせない。ralph と gh の操作と、分割計画を書くことに限る
- 9 段目の「編成の切り替え」は、ユーザーの決定で 4 段目に移した。9 段目は試行と既定値の見直しだけにする

## Rollout

1. 共通の台帳(FR-1、進行中)
2. 横断の status と stop(FR-2)
3. 全 org の上限と担当範囲の予約(FR-3)
4. 機能ごとの worktree と headless の leader、既定の編成、編成パターンの廃止(FR-4、FR-11 の `/org` skill)
5. 受信箱(FR-5、FR-11 の leader の雛形)
6. 横断の見張り(FR-6)
7. 承認と統合検証(FR-7、FR-8 の merge してよい順)
8. (8a)director の役割・skill・attach・start(FR-9、FR-8 の auto_merge)、(8b)起こし口(FR-10)
9. 試行と既定値の見直し(FR-12)

7 段目までで、人が director を兼ねる形でも使える。各段を戻すときは、その段の PR を revert する。1 段目の戻し方は、その計画の「Rollout or rollback notes」に書いた。

## Security considerations

- 越権の防止: 全 org の上限、担当範囲の予約、承認の検査は、LLM から見えない動詞の検証で強制する。director も leader もこの検証を迂回できない
- 受信箱とメッセージ経由の注入: 受信箱の本文、座席からのメッセージ、Orca や herdr から届いた入力は、director にとってデータであって指示ではない。director に指示できるのは人だけにする。今の `agent-messaging.md` の考え方を director にも広げる
- `wake_cmd` と `launch_cmd`: 利用者が `ralph.toml` に書くコマンドの雛形を ralph が実行する。受信箱の本文を雛形に埋め込むときはシェルに解釈させない(引数として渡す)。本文の代わりに件の ID だけを渡す形を既定にする
- director の権限: 分割計画を書くことと、ralph と gh の操作に限る。`auto_merge` は既定で無効にする
- 原案の投稿で、Claude の auto モードの分類器が許可を代行する操作を止めたという報告がある。director に許可を代行させない設計にしたが、分類器の挙動は確かめていない

## Open questions

- codex の director で、timeout 付きの wait を繰り返す pull が実機で使えるか
- codex の座席の `--add-dir` が sandbox の中で効くか。`codex exec` の sandbox では codex-cli 0.160.0(macOS)で効くことを確かめた(`docs/evidence/codex-add-dir-2026-10-07.md`、確かめ方は `docs/recipes/codex-seat-permissions.md`)。herdr で起動した leader の座席で共通の台帳に書けるかは未確認
- headless の director(herdr)に、外出先のスマホから話す経路。herdr の remote attach で足りるか、Orca などを使うか
- 9 段目の試行の測り方(介入回数や介入時間を誰がどう記録するか)
- 上限の既定(10 org・30 席)と期限の既定(10 分・60 分)は測った根拠がない。9 段目で見直す

## References

- council の結論: `~/.cache/council/20261007-0324-org-of-orgs/conclusion.md`(git の外)
- 原案の投稿: https://x.com/tomohisa/status/2095057357579657340 (2026-09-02。Orca に置いた設計スレッドが、herdr の中のオーケストレーション・実装・レビューのスレッドとやりとりする構成)
- 今の org runtime の仕様: `docs/specs/2026-08-01-org-runtime.md`
- 1 段目の計画: `2026-10-07-org-state-dir-common.md`(ブランチ `fix/org-state-dir-common` の `docs/plans/active/`。merge 後は `docs/plans/archive/`)
- 通信の規約: `.claude/rules/ralph/agent-messaging.md`
- GitHub の保護ブランチ: https://docs.github.com/en/repositories/configuring-branches-and-merges-in-your-repository/managing-protected-branches/about-protected-branches
- herdr: https://herdr.dev/ 、agmsg: https://github.com/fujibee/agmsg
