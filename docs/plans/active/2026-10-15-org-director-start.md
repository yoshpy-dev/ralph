# org-director-start

- Status: Approved
- Approved: 2026-10-15 sha256:f65d7c59746c
- Owner: Claude Code
- Date: 2026-10-15
- Related request: 機能ごとの org と director の仕様(`docs/specs/2026-10-07-org-multi-org-director.md`)の 8 段目の前半(8a)の 2 本目(8a-2)。FR-9 の起動 (B)(`ralph org director start` で headless の director を立てる。既定は herdr の中の専用の workspace)、役割の雛形 `internal/org/prompts/director.md`、「headless の director の permission mode は `[org.permissions.roles].director` で決める(既定は `[org.permissions].default`)」、受け入れ条件「Given `ralph org director start --driver codex --model <slug>`、when 起動する、then director が herdr の専用 workspace で動き、台帳に登録される」。8a-1 は PR #229(9cbf7080)で merge 済み。範囲と分かれ目は consult(consult-plan-dirstart)の判定と Codex の計画助言による。ユーザーは「すべてが完了するまで次のプランを自律的に進めて」と言った(2026-10-11)
- Related issue: N/A
- Type: feat
- Branch: feat/org-director-start

## Objective

人が端末を開いて `/director` で昇格させなくても director を立てられるようにする。`ralph org director start --driver <claude|codex> --model <model>` は、先に `director.jsonl` に登録してから、herdr に director 専用の workspace を作る。その pane で claude か codex を役割の雛形 `prompts/director.md` 付きで起動し、作った workspace と pane の id を登録に書き足す。止めるのは `ralph org director stop --id <id>` で、pane と workspace の名前がその起動の token のものであることを確かめてから閉じ、止めたことを記録する。director は org の座席ではないので、manifest には何も書かない。

## Scope

- 登録の拡張(`internal/org/director.go`、D2)
  - `attached` に `launch`(start なら `herdr`。attach では書かない)、`token`(start ごとの 12 桁の hex、D9)、`model`、`permission_mode` を足す。`pane_id` は 8a-1 からある field で、start の `attached` では空のまま
  - 新しいイベント
    - `workspace_created`(`id`、`workspace_id`): start が workspace を作った直後
    - `pane_created`(`id`、`pane_id`): start が tab を作った直後
    - `launched`(`id`): agent start が通り、今の登録がまだその `id` だったとき
    - `stopped`(`id`): start の後始末か `director stop` が、記録した pane と workspace を閉じたとき(閉じ済みも含む)。登録が今の登録なら、同じ write で `detached` も書く
  - どのイベントにも打った側の `ts`・`user`・`cwd`・`herdr_pane` を付ける(8a-1 と同じ)
  - 畳み込みは、今の登録に加えて、`launch = herdr` で `stopped` のない登録(今の登録でも、detach で外れたものでも)を「閉じていない headless の登録」として出す。それぞれの状態は `starting`(`launched` がない)か `running`(`launched` がある)で、記録した `workspace_id`・`pane_id`・`token`・`model`・`permission_mode` を持つ。`director stop` の対象はこれ
  - 古い binary の畳み込みは知らないイベントを数えて飛ばす(`directorFold.apply` の `default`)ので、新しいイベントの行が残っても 8a-1 の読み方は壊れない
- `ralph org director start --driver <claude|codex> [--model <model>] [--timeout-ms <ms>]`(org の層 `StartDirector`、CLI `internal/cli/org_director_headless.go`)
  - 断るとき(終了コード 1、何も書かない、herdr を打たない): `enabled = false`、attached の登録がある(headless なら `director stop --id`、そうでなければ `detach --id` の打ち方を出す。start に `--replace` は付けない)、`--org-id`、`--driver` が claude と codex 以外、model が `[org.roles].director` で許されない(spawn と同じ `resolveModelOrWarn`、省略時は警告付きの既定)、permission の規則で通らない(codex で `codex_verified = false` かつ autonomous か edits)、古い worktree の台帳(書き換えの動詞として扱う)、herdr の pane の中(`HERDR_PANE_ID` がある)から打たれたとき(出力で herdr の外の端末から打つよう言う)
  - 見張り: `ralph org start` と同じく、見張りが動いていなければ `ralph org watch --all --detach` と同じ形で起動して待つ。止まりかけ・印が古い・`--once` の見張りには断られる(D5)
  - 起動の順(D3、図 2)
    1. `director.lock` の下で読み直し、attached がなければ `attached`(`launch = herdr`、新しい `id` と `token`、`model`、`permission_mode`)を書く。これで登録が先に確定し、ほかの attach と start は断られる
    2. ロックを持たずに herdr の workspace を作る(label `ralph_director_<token>`)。ロックの下で `workspace_created` を書く
    3. tab を作る(label `director_<token>`)。ロックの下で `pane_created` を書く
    4. 役割の雛形を台帳の `prompts/director_<token>.md` に書く
    5. `herdr agent start director_<token> --kind <driver> --pane <pane> -- <permission の引数> --model <model> <雛形を指す 1 文>`
    6. ロックの下で読み直し、今の登録がまだ `id` なら `launched` を書く
  - 失敗の扱い
    - 2〜5 のどこかで失敗: 作った pane と workspace を(token の名前を確かめてから)閉じ、`stopped` と `detached` を書いて終了コード 1。閉じられなければ `detached` だけを書き(記録した id は登録に残る)、`director stop --id <id>` での片付けと、閉じるための herdr の打ち方を出す
    - 6 で今の登録が別の `id` か登録なし(人の `attach --replace` か `detach --force` が 1〜6 のあいだに入った): 自分の pane と workspace を閉じ、`stopped` を書いて(`launched` は書かない)終了コード 1。後から入った登録はそのまま残る
    - 1 のあとで start のプロセスが落ちた: 登録は `starting` のまま残る。start と attach はそれを断って `director stop --id` を案内し、stop は記録のある分だけ閉じる(何も記録がなければ閉じるものなしで `stopped` と `detached` を書く)。期限で自動に外すことはしない(8a-1 の D4 と同じく、生存の印は 8b)
  - 出力: `started d<N> (driver=..., model=..., pane=..., workspace=...)`、herdr で director の pane に入って話すこと、止めるときの `ralph org director stop --id d<N>`
  - permission(D6): `[org.permissions.roles].director`、なければ `[org.permissions].default` を `permissionArgsForDriver` で driver の引数に直す。codex は `codex_verified` の規則のまま。座席の担当範囲の検査(`autonomousScopeGateErr`)は当てない
  - cwd: 台帳の main worktree のルート(`LedgerMainWorktreeRoot`)。台帳が main worktree のものでないときは start を打った場所で、codex には `--add-dir <台帳>` を足す
  - `enabled`、`[org.permissions]`、`[org.roles]`、`model_pool` は、`--config` がなく台帳が main worktree のものなら main worktree の `ralph.toml` から読む(8a-1 の D6 と同じ規則)
  - `--config` を渡した start は、その絶対パスを雛形に入れ、director が打つ `ralph org` のコマンドに `--config <path>` を付けて描く。director が start と同じ設定を読むため(cross-review cycle 1 の指摘 1)
- `ralph org director stop --id <id> [--force]`(org の層 `StopDirector`、`internal/org/director_stop.go`、D4、D11)
  - 対象は閉じていない headless の登録(`starting` でも `running` でも、今の登録でも detach で外れたものでも)。`--id` がない、その `id` の閉じていない headless の登録がない、すでに `stopped` があるときは断る(終了コード 1、何も書かない)。昇格したセッションの登録(`launch` がない)なら `detach --id` を案内する
  - director の pane か workspace の中から打たれたら断る(`HERDR_PANE_ID` が登録の pane と同じとき。または label を確かめた workspace が `HERDR_WORKSPACE_ID` と同じとき。閉じると打ったセッションが終わり、記録が書けないため)
  - 手順: 記録した pane があれば、tab の label が `director_<token>` で workspace の label が `ralph_director_<token>` であることを確かめ(`confirmSeatPane` と同じ形)、C-c を送って閉じる → 記録した workspace があれば label を確かめて閉じる → `director.lock` の下で `stopped`(今の登録なら `detached` も)を書く。herdr が pane か workspace を知らなければ閉じ済みとして進む。label が違うときは C-c も close も打たない。確かめられない、閉じられないときは何も書かずに終了コード 1 で、理由と、人が閉じるための herdr の打ち方を出す
  - `--force`: label が違う、確かめられない、閉じられないときも、C-c も close も(まだ打っていない分は)打たずに `force` 付きの `stopped`(今の登録なら `detached` も)を書き、閉じていないことと、label `ralph_director_<token>` の workspace を herdr で探して閉じる打ち方を出す。label が合えば `--force` があっても通常どおり閉じる。herdr が状態を失って id を振り直したあと、記録を片付ける手段にする(D11)
- 役割の雛形(`internal/org/prompts/director.md`、`internal/org/prompts.go`、D7)
  - 座席の `RolePromptVars` とは別の `DirectorPromptVars`(登録の `id`、driver、台帳を指す `--state-dir` の値、cwd)で描く `RenderDirectorPrompt`
  - 中身
    - 最初に `ralph org director status --state-dir <台帳>` を打ち、今の登録が自分の `id` であることを確かめてから動く。違うか登録がなければ何もせず、そのことをこの pane に書いて止まる
    - 自分は `ralph org director start` が立てた headless の director で、登録は済んでいる。手順の全体は `/director` skill に従う。昇格の手順のうち、見張りの確かめと attach は start が済ませた。会話の要約のあとに登録を読み直す手順は守る
    - 打つ `ralph org` のコマンドにはどれも `--state-dir` を付ける
    - 人とはこの pane で話す
    - 終わりに detach を打たず、人に、herdr の外の端末から `ralph org director stop --id <id>` を打つよう頼む
  - `ralph org spawn --role director` は断る(director は座席ではない)
- 表示
  - `ralph org director status` と `ralph status` の `director:` の行は、headless の登録なら `starting` か `running`、pane、workspace、model を出す(`--json` にも)。閉じていない headless の登録が今の登録のほかにもあれば、その `id` を出す
  - `ralph org stop --all` は、閉じていない headless の登録があれば、director は止めないことと `ralph org director stop --id <id>` の打ち方を 1 行出す(終了コードは変えない)
  - `ralph org director` の help から「start はまだない」を消し、start と stop を書く
- `/director` skill(4 面): director の立て方を 2 つ(昇格と `director start`)にし、headless の director の始め方(最初に登録を確かめる)と終わり方(detach を打たず、人に stop を頼む)を書く。「まだないもの」から headless の director を消す
- 文書: README の director の段落、仕様 FR-9 の 8a-2 の注、AGENTS.md の repo map(`director.go`)、`templates/base/ralph.toml` の `[org.permissions.roles]` のコメント(director の例)、`docs/tech-debt/README.md`

## Non-goals

- 8a-2b: `[org.director].launch_cmd`(herdr の外の端末で headless の director を起動する)。仕様 FR-9 の起動 (B) の後半。consult の判定で分けた(herdr の起動は argv を `herdr agent start` に直接渡すので、起動スクリプトを使う launch_cmd と仕組みを共有しない)
- 8a-3: `[org.director].auto_merge`、ほかの PR の merge のあとの rebase と gate の取り直し、merge-order の JSON の gh の失敗の見分け
- 8b: push の起こし口、director の生存の印、`starting` のまま残った登録を期限で外すこと
- `stop --all` と `disband --all` が headless の director も止めること(director は org の座席ではない。案内の 1 行だけ出す)
- director の model receipts(`model-receipts.jsonl` は org と座席の鍵で、director は座席ではない。model と permission mode は登録に書く。D8)
- director の pane が閉じられたこと(人が herdr で閉じた、herdr が落ちた)を見張りが見ること
- start の `--replace`
- org の座席の agent 名(`<org_id>_<seat_id>`)が repo をまたいで重なること(今のまま。director だけ token にする)

## Assumptions

- herdr 0.7.5 の workspace と tab の label は自由な文字列で(`herdr workspace create --label <TEXT>`)、agent 名は herdr の server 全体で一意で `^[a-z][a-z0-9_-]{0,31}$`(`internal/org/spawn.go` の `herdrAgentName` と `maxHerdrAgentNameLen` の doc)。`director_<12 桁の hex>` は 21 字で収まる。org_id は `^[a-z][a-z0-9-]{0,29}$`(`internal/org/identifier.go`)で `_` を含まないので、label の `ralph_director_<token>` と `director_<token>` は org の workspace と座席の label と重ならない。agent 名の `director_<token>` は、org_id が `director` で seat_id が token と同じ文字列の座席の agent 名とだけ重なりうる(そのときは agent start が失敗して start の後始末になり、label が違うのでほかの pane は閉じない。tech-debt に 1 行)
- `confirmSeatPane(orgID, seatID, paneID)` と `confirmOrgWorkspace(orgID, workspaceID)`(`internal/org/verbs.go`)は manifest を読まず label の文字列だけを比べるので、token の名前で使える
- `permissionArgsForDriver`、`writePromptFile`、`PromptFilePointer`、`agentStartWithRetry` は manifest に書かない(Explore の地図)
- `[org.permissions.roles]` と `[org.roles]` は役割の名前を検証しないので、`director` の鍵は今の設定の検証のまま読める
- 台帳のロックは 5 秒で待ちを切る(`internal/org/lockfile.go` の `manifestLockTimeout`)。agent start は最大 60 秒かかるので、herdr を打つあいだ `director.lock` を持たない(D3)

## Affected areas

- 変えるファイル: `internal/org/director.go`、`internal/org/director_test.go`、`internal/org/prompts.go`、`internal/org/prompts_test.go`、`internal/org/spawn.go`(role director の拒否)、`internal/org/verbs_all.go`(stop --all の案内の 1 行)、`internal/cli/org_director.go`、`internal/cli/org_director_test.go`、`internal/cli/org.go`、`internal/cli/org_legacy_ledger.go`、`internal/cli/status.go`、`internal/cli/status_test.go`、`internal/cli/org_stop_all_test.go`、`templates/base/ralph.toml`、`/director` skill の 4 面、README、仕様、AGENTS.md、`docs/tech-debt/README.md`
- 新しいファイル: `internal/org/prompts/director.md`、director の起動と停止の org の層のファイル(`internal/org/director_start.go`、`internal/org/director_stop.go` とそれぞれのテスト。S1b のあとで director_start.go が 949 行になり、800 行の目安を超えたので stop を分けた)、start と stop の CLI のファイル(`internal/cli/org_director_headless.go` とそのテスト。S2 のあとで org_director.go が 914 行になったので分けた)

## Visual review

- 図解ページ: `.harness/state/plan-visual/org-director-start.html`(図 1 全体図、図 2 start の順と後始末、図 3 登録の移り方)
- 自己点検: 全体と全体図(`--fragment overview`)、図 2・図 3 を `./scripts/plan-visual.sh shot` で撮って読んだ。1 回目は全体図の縦の矢印が台帳の見出しを、破線が herdr の見出しを横切り、図 2 の失敗の枠の見出しと注が lifeline に重なっていた。直して撮り直した。Codex の計画助言を受けて、図 1 の名前を token 付きに、図 2 を「先に登録」の順(① から ⑥ と、失敗か割り込みの後始末)に、図 3 を `starting`・`running`・headless の detached・`stopped` に描き直した。2 回目の自己点検で、図 2 の矢印のラベルの下地が線を半分隠していたので、下地を外してラベルを線から離した。撮り直した 3 つの図と全体に重なりはない。S1b のあとの D11 で、図 1 の StopDirector を `internal/org/director_stop.go` に、stop の CLI に `[--force]` を足し、図 3 の注と冒頭の「変わるもの」に `--force` を書いた。3 回目の自己点検で stop の CLI の下の文字が箱からはみ出していたので短くし、撮り直した全体図と全体に重なりはない

## Design decisions

- D1 範囲: 8a-2 は herdr の中の start と stop、役割の雛形、`[org.permissions.roles].director` に絞り、`launch_cmd` は 8a-2b に分ける。consult(consult-plan-dirstart)の判定。herdr の起動は argv を `herdr agent start` に直接渡し(`internal/org/spawn.go` の agent start、`internal/org/driver/herdr.go`)、launch_cmd に要る起動スクリプト・設定の検証・ralph が止められない登録は別の仕組みになるため。8a-1 は後工程を 3 回まわしたので、1 本を小さく保つ
- D2 headless の director の居場所は `director.jsonl` に書き(`attached` の `launch`・`token`・`model`・`permission_mode`、`workspace_created`、`pane_created`、`launched`、`stopped`)、manifest には何も書かない。manifest に書くと `max_orgs`・`max_total_seats` に数えられ、`stop --all`・`disband --all` に拾われ、`approveTripwire` がその pane からの `ralph org approve` を断る(Explore の地図)。8a-1 の D3 と同じ判断
- D3 先に登録する: start は `director.lock` の下で `attached` を書いてから herdr を打ち、workspace と pane の id は分かった時点で短い追記で書き足す。herdr を打つあいだはロックを持たない。Codex の計画助言 1(HIGH)と consult の判定。最初の案(herdr のあとにロックの下で読み直して書く楽観形)は、登録が確定する前に agent が動き、負けた側の agent が受信箱の ack・resolve・send をしうる(pane を閉じても取り消せない)ので退けた。先に登録すれば attach と start が attached を断るので取り合いがなくなり、1〜6 のあいだに入れるのは人の `attach --replace` と `detach --force` だけになる。その窓は雛形の最初の手順(今の登録が自分の `id` かを確かめる)と 6 の読み直しで塞ぐ。start のロックを全体で持つ案は、同時の attach と detach が 5 秒で時間切れになる(`manifestLockTimeout`)ので退けた
- D4 止めたことは `detached` と別のイベント `stopped` で書き、`director stop --id` は今の登録かどうかによらず、閉じていない headless の登録に効く。consult の判定。headless の director が skill のとおり自分で detach しても、pane は開いたままなので、人が stop で閉じられるようにするため。`detach` の意味(記録だけ)は変えない。退けた案: `detach` が pane を閉じる(昇格したセッションが自分の pane を閉じて会話が切れうる)
- D5 start は見張りを確かめ、なければ起動する(`ralph org start` と同じ)。director の仕事は受信箱で、`enabled` の escalate は見張りが期限を回すことを前提にするため。`enabled = false` なら attach と同じく断る
- D6 permission は仕様どおり `[org.permissions.roles].director`、なければ `[org.permissions].default`。template の既定は `autonomous` なので、claude の director は `--permission-mode bypassPermissions` で動く。「実装しない」は雛形と skill の指示だけで守られる(tech-debt に 1 行)。仕様が既定を他の役割と同じにすると決めているので、director だけ変えない
- D7 役割の雛形は座席の雛形と別の関数で描き、`ralph org spawn --role director` は断る。spawn の `--role` は開いていて、`prompts/` に雛形のある役割はその雛形を使うため(Explore の地図)、断らないと座席としての director が立ててしまう
- D8 director は model receipts に書かない。receipts は org と座席の鍵で、director には org がない。model と permission mode は登録に書く
- D9 名前に起動ごとの token を入れる: start ごとに crypto/rand の 12 桁の hex を作り、agent 名は `director_<token>`、workspace の label は `ralph_director_<token>`、tab の label は `director_<token>`。止めるときも後始末のときも、tab と workspace の label がその token の名前であることを確かめてからだけ閉じる。Codex の計画助言 2(HIGH)と consult の判定。herdr の agent 名は server 全体で一意なので、台帳ごとの `d<N>` を名前にすると 2 つの repo の最初の director が同じ名前になる。herdr が状態を失って id を振り直すと、repo A の古い登録の pane id が repo B の pane を指しうるが、token が違うので label の検査で閉じない。最初の案(workspace の label `ralph_director`、tab の label `d<N>`)は退けた
- D10 `stop --all` は headless の director を止めず、閉じていない headless の登録があれば止め方を 1 行出す。consult の判定(「ralph の仕組みだけで止められる」の案内)
- D11 `director stop --force` は、label が違う・確かめられない・閉じられない登録でも、閉じずに `force` 付きの `stopped` を書く。workspace の中からの stop は、label を確かめた workspace が `HERDR_WORKSPACE_ID` と同じときに断る。S1b の実装で見つかった 2 つの穴への consult(consult-plan-dirstop)の判定。force がないと、herdr が id を振り直したあとの登録は stop が毎回断り、閉じていない headless の登録として status と `stop --all` の案内に残り続ける(手で `director.jsonl` を直すしかない)。座席の `ralph org stop --force` も、確かめられない pane には close を打たずに `stopped` を記録する(`internal/org/verbs.go` の `stopSeatPane` と `notClosedNote`)ので、同じ意味にした。close は打たないので、ほかの起動の pane を閉じないこと(AC5)は保たれる。workspace の検査を env の一致だけにしないのは、id の振り直しのあとに無関係な workspace にいる人を誤って断らないため。退けた案: workspace を閉じる前に `stopped` を書く(閉じ損ねたときの記録が事実と違う)

Critical forks: D1、D4、D10 は consult(consult-plan-dirstart)の判定で、D3 と D9 は Codex の計画助言の 2 件の HIGH を受けた consult の判定で、D11 は S1b のあとの consult(consult-plan-dirstop)の判定で決めた(ユーザーの指示「計画の分かれ目も自分で選ぶ」「Codex の修正の提案は必ず直す」)。D6 は仕様どおり。

## Acceptance criteria

- [x] AC1 `ralph org director start --driver codex --model <slug>` は、偽の herdr に対して、まず `director.jsonl` に `launch = herdr`・`token`・`model`・`permission_mode` 付きの `attached` を書き、label `ralph_director_<token>` の workspace と label `director_<token>` の tab を作って、そのたびに `workspace_created` と `pane_created` を書き、`herdr agent start director_<token> --kind codex --pane <pane>` を、permission の引数、`--model <slug>`、雛形のファイルを指す 1 文の順の引数で打ち、`launched` を書いて、stdout に `id`・pane・workspace と stop の打ち方を出す。manifest にも model receipts にも何も書かない。`--driver claude` でも同じ形(仕様の受け入れ条件「director start」)
- [x] AC2 attached の登録があるとき(`starting` を含む)、`enabled = false` のとき、`--org-id`・`--driver` が違うとき、`[org.roles].director` で model が許されないとき、`HERDR_PANE_ID` があるとき、start は断り(終了コード 1)、herdr を打たず、何も書かない。attached の登録があるときは、headless なら `director stop --id`、そうでなければ `detach --id` の打ち方を出す。`starting` の登録には attach も同じく `director stop --id` を案内する
- [x] AC3 workspace の作成・tab の作成・雛形の書き込み・agent start のどれかが失敗すると、start はそれまでに作った pane と workspace を閉じ、`stopped` と `detached` を書いて終了コード 1 で終わる。閉じられなかったときは `detached` だけを書き、`director stop --id` と閉じるための herdr の打ち方を出す
- [x] AC4 `attached` のあと `launched` の前に `attach --replace` を差し込むと(テストでは agent start の偽に差し込む)、start は自分の pane と workspace を閉じ、`stopped` を書き、`launched` を書かず終了コード 1 で終わる。`--replace` の登録は今の登録のまま残る。雛形の最初の手順は、`ralph org director status --state-dir <台帳>` で今の登録が自分の `id` であることを確かめ、違えば何もしないと書く(文の固定のテスト)
- [x] AC5 2 つの台帳が 1 つの偽の herdr を使い、herdr が id を振り直したあと(A の記録した pane と workspace の id が B のものを指す)、A の `director stop --id` は label の違いで B の pane にも workspace にも C-c と close を打たず、何も書かず終了コード 1 で終わる。2 つの start の agent 名・label は token で違う
- [x] AC6 permission は `[org.permissions.roles].director` を読み、なければ `[org.permissions].default` を読む。claude で autonomous なら `--permission-mode bypassPermissions`、edits なら `acceptEdits`、guarded なら引数なし。codex は `codex_verified = false` なら guarded のほかを断る(spawn と同じ)。これらの設定と `enabled` と `[org.roles]` は、`--config` がなく台帳が main worktree のものなら main worktree の `ralph.toml` から読む
- [x] AC7 見張りが動いていなければ start は見張りを起動してから director を立てる。止まりかけ・印が古い・`--once` の見張りでは断り、herdr を打たず、何も書かない
- [x] AC8 `ralph org director stop --id <id>` は、記録した pane の tab の label が `director_<token>` で workspace の label が `ralph_director_<token>` のとき、C-c を送り、pane と workspace を閉じ、`stopped` を書く。登録が今の登録なら `detached` も同じ write で書き、`ralph org director status` は not attached になる。detach で外れたあとの headless の登録にも効き、`stopped` だけを書く。`starting` のまま残った登録(pane も workspace も記録がない、または workspace だけ記録がある)にも効き、記録のある分だけ閉じる。herdr が pane か workspace を知らなければ閉じ済みとして進む
- [x] AC9 stop は、`--id` がない、その `id` の閉じていない headless の登録がない、`stopped` がすでにある、pane の中(`HERDR_PANE_ID` が登録の pane)か workspace の中(label を確かめた workspace が `HERDR_WORKSPACE_ID`)から打たれたときに断り(終了コード 1)、何も書かない。`--force` がなければ、label が違う、閉じられないときも同じく断る。label が違うときは C-c も close も打たない。昇格したセッションの登録なら `detach --id` を案内する
- [x] AC14 `director stop --id <id> --force` は、label が違う(AC5 の台帳 A)、確かめられない、閉じられないとき、まだ打っていない C-c と close を打たずに `force` 付きの `stopped`(今の登録なら `detached` も)を書き、閉じていないことと label `ralph_director_<token>` の探し方を出す。label が合えば `--force` があっても閉じる。pane か workspace の中からの拒否は `--force` でも変わらない
- [x] AC10 `ralph org director status` と `ralph status` の `director:` の行は、headless の登録なら `starting` か `running`、pane・workspace・model を出し、`--json` にも出す。`ralph org stop --all` は、閉じていない headless の登録があれば止め方を 1 行出し、終了コードは変えない
- [x] AC11 `ralph org spawn --role director` は断られ、何も書かない。`RenderDirectorPrompt` は `{{...}}` を残さず、雛形は `/director` skill を指し、`--state-dir`、最初の登録の確かめ、stop の頼み方、detach を打たないことを書く。`--config` を渡した start の雛形は、`ralph org` のコマンドに `--config <絶対パス>` を付け、渡さない start の雛形は付けない
- [x] AC12 `/director` skill の 4 面が同じ中身で、`scripts/check-skill-sync.sh` と `scripts/check-sync.sh` が通る。help、README、仕様 FR-9 の注、AGENTS.md、`templates/base/ralph.toml` のコメント、tech-debt が 8a-2 の動きを説明し、「start はまだない」「headless の director はまだない」と書いた箇所が残っていない
- [x] AC13 テストは本物の herdr と osascript を打たず、repo の台帳に書かない。実機の確認は、人の herdr server に触れない隔離した herdr server で行う(evidence に残す)

## Implementation outline

1. S1 org の層: 登録の拡張(新しい field とイベント、閉じていない headless の登録の畳み込み、`starting` と `running`)、`StartDirector`(偽の herdr で、先に登録、id の書き足し、後始末)、`StopDirector`(token の名前の検査)、`prompts/director.md` と `RenderDirectorPrompt`、spawn の role director の拒否、`stop --all` の案内
2. S1c org の層の続き: stop を `internal/org/director_stop.go` に移すだけのコミット(`refactor:`)と、`--force` と workspace の中からの拒否のコミット(`feat:`)に分ける(D11)
3. S2 CLI: `ralph org director start` と `stop`(`--force` を含む)(設定を main worktree の規則で読む、見張りの確かめ、古い台帳)、`director status` と `ralph status` の表示、help
4. S3 `/director` skill の 4 面と文書(README、仕様、AGENTS.md、template のコメント、tech-debt)

## Verify plan

- Static analysis checks: `./scripts/run-static-verify.sh`、`gofmt -l`、`go vet ./...`、`GOTOOLCHAIN=go1.25.8 go build ./... && go vet ./...`、`scripts/check-skill-sync.sh`、`scripts/check-sync.sh`
- Spec compliance criteria to confirm: FR-9 の起動 (B) の herdr の部分を AC1・AC7・AC8、`[org.permissions.roles].director` を AC6、受け入れ条件「director start --driver codex --model <slug>」を AC1 と実機の確認、FR-3 の「director は座席ではないので数えない」を AC1(manifest に書かない)と対応づける。launch_cmd が 8a-2b に送られたことを仕様の注に書いたか
- Documentation drift to check: 「not available yet」「まだないもの」の headless の行(grep)、`/director` skill の 4 面、README と仕様の director の段落、AGENTS.md の repo map
- Evidence to capture: 隔離した herdr server(XDG の 3 変数で分け、pane のシェルは bash)で、HEAD の binary の `ralph org director start --driver codex --model <slug>`(と claude)を打ち、`herdr workspace list` の label と `director.jsonl`、`ralph status` の `director:` の行を取り、`director stop --id` で pane と workspace が閉じることを確かめる。herdr が `_` を含む 27 字の label をそのまま返すことも見る。agent が起動しない環境なら、偽の herdr のテストと、そこまでの出力を残し、確かめていないと書く

## Test plan

- Unit tests
  - director store: 新しい field とイベントの読み書き、`stopped` の書き方(今の登録なら `detached` も 1 回の write で)、閉じていない headless の登録の畳み込み(今の登録、detach で外れたもの、`stopped` 済み、`starting` と `running`)、id の書き足しのイベントがほかの `id` を指すとき、古い行(新しい field なし)の読み、壊れた行
  - StartDirector(偽の herdr): イベントの順(attached → workspace_created → pane_created → launched)、引数の順と中身、label と agent 名の token、雛形のファイル、各段の失敗の後始末(閉じる、閉じられないときの打ち方と `detached` だけ)、`launched` の前の `--replace`(AC4)、プロセスが落ちたあとの `starting` の登録、manifest に書かないこと
  - StopDirector(偽の herdr): 成功、閉じ済み(not found)、label の違い、close の失敗、pane の中から、`stopped` 済み、pane のない登録、detach 後の登録、`starting` の登録(記録なし・workspace だけ)、2 つの台帳と id の振り直し(AC5)、`--force`(label の違い・確かめられない・close の失敗で記録だけ、label が合えば閉じる、AC14)、workspace の中からの拒否(label を確かめた workspace が `HERDR_WORKSPACE_ID` のときだけ)
  - token: 12 桁の hex、agent 名の規則に収まる、2 回の start で違う
  - 雛形: `RenderDirectorPrompt` が `{{` を残さない、必要な文を含む(最初の手順の文の固定)。spawn の role director の拒否
  - permission: role director の解決(roles、default、codex の verified)
- Integration tests: CLI の `ralph org director start|stop|status`(一時の台帳、stub の herdr、`--json`、`--org-id` の拒否、`enabled = false`、見張りの偽の印、main worktree の `ralph.toml` と feature worktree の `ralph.toml`)、`ralph status` の `director:` の行と JSON、`ralph org stop --all` の案内
- Regression tests: `go test ./internal/...` を go1.26 と go1.25.8 で、`-race`。8a-1 の attach・detach・status のテストが通ること、help の文の固定のテストの直し
- Edge cases: `USER` がない、台帳が main worktree のものでない(cwd と codex の `--add-dir`)、雛形のファイルが書けない、agent start の時間切れ、id の書き足しの追記の失敗
- Evidence to capture: `docs/evidence/test-2026-10-15-org-director-start.md`

## Risks and mitigations

- headless の director が既定で bypassPermissions で動き、実装しないことが指示だけで守られる: 仕様どおりとし、tech-debt に 1 行、README と template のコメントで `[org.permissions.roles].director` を書く
- 人が herdr で director の pane を閉じると、登録は attached のまま残る(生存の印は 8b): `director stop --id` は閉じ済みとして `stopped` と `detached` を書けるので、それで片付ける。status の help に書く
- start のプロセスが 1 のあとで落ちると、`starting` の登録が残り、attach と start が断られる: どちらも `director stop --id` を案内し、stop は記録のある分だけ閉じる。workspace を作ってから `workspace_created` を書くまでに落ちたときは、herdr に記録のない workspace が残る(label `ralph_director_<token>` で見つけられる。stop の出力に label を出す。tech-debt に 1 行)
- 雛形の最初の手順で登録を確かめたあとに `attach --replace` が入る窓が残る: 昇格の director と同じ窓で、`/director` skill の「会話の要約のあとに読み直す」と同じ扱いにする
- herdr が状態を失って id を振り直す: token の名前の検査(AC5)で、ほかの起動の pane と workspace を閉じない。stop は断り、人の打ち方を出す
- 実機の agent start が環境によって通らない(モデルの退役の確認画面、ログインなど): 偽の herdr のテストを主の根拠にし、実機の結果は evidence に書く
- 8a-1 のテストの help の文の固定(「not available yet」)を直す: テストの意図の変更ではなく文の更新として review の報告に書く

## Rollout or rollback notes

- 新しい動詞(`director start`、`director stop`)、イベント(`workspace_created`、`pane_created`、`launched`、`stopped`)、`attached` の field を足す。8a-1 の登録の読み方は変わらない(新しい field のない行はそのまま読める)。戻すときは PR を revert する。revert のあとの binary は新しいイベントを知らないイベントとして数えて飛ばし、start が今の登録のときに書いた `stopped` には同じ write の `detached` があるので、登録の状態は正しく読める
- release のノートに足すこと: `ralph org director start` と `stop`、新しいイベントと `attached` の field、`[org.permissions.roles].director`、headless の director は既定で `[org.permissions].default`(template の既定は autonomous)、`ralph org spawn --role director` を断る、`stop --all` の案内の行

## Open questions

- herdr が `ralph_director_<token>` の label をそのまま返すか(実機で確かめる)

## Progress checklist

- [x] Plan reviewed
- [x] Plan approved
- [x] Branch created
- [x] Implementation started
  - S1a b9861932(登録の拡張、雛形 `prompts/director.md` は他の役割と同じく日本語、spawn の role director の拒否)。S1b 8528515f(StartDirector と StopDirector、`codexWritableRootArgs` の director、StopAll の閉じていない headless の登録)。S1b の報告の 2 つの穴を consult(consult-plan-dirstop)に聞き、D11 と AC14 を足して計画を承認し直した(2026-10-15)
  - S1c b5ce8dea(stop を director_stop.go へ移すだけ)と a2d5fc01(stop の `--force`、director の workspace の中からの stop の拒否)。S2 d4da1c45(CLI の start・stop・status、stop --all の案内、main worktree の ralph.toml の読み方)。S2b 725fff10(org_director.go が 914 行になったので、start と stop の CLI を `internal/cli/org_director_headless.go` へ移すだけ)。新しいファイルを Affected areas に書き足し、承認し直した(digest d59311f57d8e、図は CLI のファイル名を描いていないので変えていない)
  - S3 9fb4f635(`/director` skill の 4 面、README、仕様 FR-9 の注、AGENTS.md、template の ralph.toml の注、tech-debt 5 行)。S3 の報告にあった範囲外の文書のずれ(`/org` skill の director の動詞の行、`ralph-workflow.md` の `/director` の項目が headless に触れない)は /sync-docs に渡す
  - self-review(cycle 1)b50d261b: CRITICAL・HIGH 0、MEDIUM 1、LOW 6。S4 da818b4f で 7 件を直し、dbd67a58 で改名のあとのコメント 4 行を直した。verify ec70c739 は pass(隔離した herdr server の実機で codex と claude の start と stop を確かめた)。verify の AC7 の一部未確認(見張りの印が古い場合)は S5 ca98c248 のテストで固定した(見張りを起動せずに断る)。verify の V4 で、Evidence to capture の label の字数を 22 から 27 に直し、承認し直した(digest 671c60416856、図は字数を描いていない)
  - test 8461043a は pass。origin/main(f21df66c)を 1cdddbbb で merge した(tech-debt の末尾の行が衝突し、両方を残した)。sync-docs e80ccafe。cross-review(cycle 1)54cab8a2 は ACTION_REQUIRED 1 件: `--config` を渡した start の設定が director の雛形に入らない。Scope と AC11 に `--config` の 1 文を足し、承認し直した(digest f65d7c59746c、図は設定のパスを描いていない)。cycle 2 で直し、/test のテストの穴 D4・E1・E12/E13・P2 のテストも足す
- [x] Review artifact created
- [x] Verification artifact created
- [x] Test artifact created
- [ ] PR created
