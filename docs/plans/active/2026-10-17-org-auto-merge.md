# org-auto-merge

- Status: Approved
- Approved: 2026-10-17 sha256:d67898ab68d4
- Owner: Claude Code
- Date: 2026-10-17
- Related request: 機能ごとの org と director の仕様(`docs/specs/2026-10-07-org-multi-org-director.md`)の 8 段目の前半(8a)の 3 本目の前半(8a-3a)。FR-8「既定では人が merge する。`[org.director].auto_merge = true` にすると、director が CI の通った PR を merge する。どちらの場合も、director は『merge してよい順』を報告する」と、受け入れ条件「Given `auto_merge = false`(既定)、when PR の CI が通る、then director は merge せず、merge してよい順を報告する」。制約「director の権限は分割計画を書くことと、ralph と gh の操作に限る。`auto_merge` は既定で無効」。consult(consult-plan-automerge)の 2026-10-16 と 2026-10-17 の判定(2 回目は Codex の計画助言の HIGH 3 件の直し方)。tech-debt の merge-order と merge guarantee の行(237、238、240、242 行)
- Related issue: N/A
- Type: feat
- Branch: feat/org-auto-merge

## Objective

director が、人に聞かずに CI の通った PR を 1 本ずつ merge し、残りのブランチの gate を取り直せるようにする。既定は今までどおり人が merge を決める。新しい動詞 `ralph org merge --branch <ブランチ>` は、`ralph org merge-order` が `ready` と分類したブランチの PR だけを、検査がすべて通り、GitHub が「merge の時点で base が最新であること」を打つ本人にも強いていて、merge queue がないことを確かめてから、gate が通した head で `gh pr merge` する。どれかが確かめられなければ何もせずに断る。人に頼まれた merge も、この動詞で同じ確かめを通す。`[org.director].auto_merge = true` のときだけ、`/director` skill が merge と取り直し(`ralph org regate`)を自分から打つ。あわせて、merge-order の JSON に、ブランチごとの gh の PR の読み取りの結果を足す。

## Scope

- merge-order の JSON: entry に `pr_lookup` を足す。値は `found`(origin の repository の PR を使った)、`none`(gh が PR を返さなかった)、`failed`(gh がこのブランチで落ちた)、`limit_forks`(gh が上限の数の PR を返し、すべて fork だった)、`skipped`(gh がない、ほかの repository を読んだので全ブランチの PR を捨てた、または問い合わせる前に止まった)。text の出力は変えない
- 設定 `[org.director].auto_merge`(bool、既定 false)。`enabled` と同じ規則で読む(`--config` がなく台帳が main worktree のものなら main worktree の `ralph.toml`)。`ralph status` の `director:` の行と `--json`、`ralph org director status` に値を出す
- 動詞 `ralph org merge --branch <b>`(`internal/org/merge.go` と CLI)。確かめ(S3a)と merge と後始末(S3b)に分ける
  - 対象の repository を決める(D10): origin の URL から host と owner/repo を読み(読めなければ断る)、動詞の gh はすべて `-R <host>/<owner>/<repo>`(`gh api` は `--hostname <host>`)で打つ。PR の url の host と owner/repo が一致しなければ断る
  - ブランチを読む: 台帳と ls-remote は merge-order と同じ読み方で、gh は名指しのブランチだけを、決めた repository で読む。ブランチが `ready`、`pr_lookup` が `found`、PR が OPEN、PR の head が gate の head、PR の base が origin の既定のブランチのときだけ進む
  - 結びつきの org が走っていれば断る(D5)
  - PR に auto-merge の予約(`autoMergeRequest`)がすでにあれば断る(D12)
  - `gh pr checks <番号> --json bucket,name,state,workflow` を読み、検査がすべて `pass` か `skipping` で、`pass` が 1 つ以上あるときだけ進む(D7)
  - `--wait-checks <時間>`(既定 0 で待たない、上限 6 時間)を渡すと、検査が待っているか 0 件の断りのときだけ、ロックを外して 30 秒ごとに `gh pr checks` だけを読み直し、待ちが終わるか時間が切れたら、ロックを取り直して確かめを最初から打つ(D13)
  - 保護を確かめる(D11): 既定のブランチに merge queue があれば断る。up-to-date の rule が自前の required check を持ち、その rule を打つ本人が迂回できないときだけ進む
  - merge の方式は `gh repo view --json viewerDefaultMergeMethod` で読む(D2)
  - merge の直前に ls-remote を打ち直し、既定のブランチの先頭が gate の base、ブランチの先頭が gate の head のままかを確かめる(D8)
  - `gh pr merge <番号> --merge|--squash|--rebase --match-head-commit <gate の head> -R <repo>` を打つ。`--auto`、`--admin`、`--delete-branch` は付けない
  - merge のあと PR を読み直し、MERGED なら台帳に org の記録 `pr_merged` を 1 行書く。MERGED でなければ、予約があれば `--disable-auto` で外し、終了コード 1 で何も記録しない(D6)
  - 台帳ごとのロック `regate.lock` の排他を取り、regate と merge を台帳ごとに 1 つずつ走らせる(D3)
- `/director` skill の 4 面: auto_merge のときの手順(merge-order → merge → merge-order → 取り直しの regate → …)と、auto_merge でないときの手順(今までどおり報告し、人に頼まれたら `ralph org merge` で 1 本)。merge は動詞でだけ打ち、`gh pr merge` を直接打たない。動詞が断ったら理由を人に伝えて止まる。「まだないもの」から自動の merge を外す
- `/org` skill の 4 面の動詞リファレンスに `merge` の行、「merge してよい順」に動詞の案内。仕様 FR-8 の注、README の動詞の表、`AGENTS.md` の repo-map、`docs/quality/quality-gates.md` の 2 面(見張りなしで使える動詞)、tech-debt、`templates/base/ralph.toml` の `[org.director]` のコメント、CLI の help

## Non-goals

- 押して起こす仕組み(push の起こし口)と director の生存(8b)
- merge のあとのブランチの削除と worktree の後始末(`--delete-branch` を付けない。merge-order の merged の見分けと cleanup の手順を変えない)
- merge queue のある branch への merge(動詞は断る)
- up-to-date の保護がない repo、保護を読めない repo、打つ本人が保護を迂回できる repo への merge(動詞は断り、人が自分で merge する)
- origin が ssh の別名の host(`~/.ssh/config` の `Host`)を指す repo(gh が host を解けず断りになる。tech-debt 240 行と同じ制約)
- 保護を迂回する merge(`--admin`)と、GitHub の自動 merge(`--auto`)
- merge-order の分類と、merge-order 自体の gh の打ち方の変更(`pr_merged` を読んで merged と分類しない。merge-order の gh に `-R` を付けない)
- `ralph doctor` の Merge guarantee の判定と、保証のキャッシュの形の変更(D11 は別の関数にする)
- leader の雛形(`internal/org/prompts/leader.md`)と、headless の director の雛形(`internal/org/prompts/director.md`)。雛形は `/director` skill に従う作りのまま
- 外部のコマンドを走らせる関数の重複(tech-debt 238 行の後半)
- merge の方式の設定(`merge_method`)。repo の既定の方式で足りる。求められたら足す

## Assumptions

- gh 2.102 の `gh pr checks --json` は `bucket`(pass・fail・pending・skipping・cancel)を持ち、検査が待っているときは終了コード 8 で終わる。終了コードによらず stdout の JSON を読む
- gh 2.102 の `gh pr merge` は `-R [HOST/]OWNER/REPO`、`--match-head-commit`、`--disable-auto` を持つ。merge queue を要る branch では、検査が終わっていないと `--auto` なしでも auto-merge を有効にする(`gh pr merge --help` で確認済み)
- `gh pr view --json` は `state`、`mergeCommit`、`autoMergeRequest` を持つ。merge queue に入ったかを表す field はない
- `gh repo view --json viewerDefaultMergeMethod` は MERGE・SQUASH・REBASE のどれかを返す。この repo は MERGE
- 2026-10-17 にこの repo で読み取りだけの gh を打って確かめた: `gh api repos/<o>/<r>/rules/branches/<b>` の要素は `ruleset_id` を持つ。`gh api repos/<o>/<r>/rulesets/<id>` は `current_user_can_bypass` を持つ(この repo は `never`)。GraphQL の `repository.mergeQueue(branch:)` は queue がなければ null を返す。`gh api repos/<o>/<r>` は `permissions.admin` を持つ。`gh pr view <n> -R github.com/<o>/<r> --json state,autoMergeRequest,mergeCommit` は 3 つを返す。`current_user_can_bypass` のほかの値(`always`・`pull_requests_only` など)と、queue があるときの `mergeQueue` の値は確かめていない。動詞は `never` 以外と null でない値をすべて断るので、知らない値は断る側に倒れる
- up-to-date の rule が効いていると、base が進んで head が遅れた PR の merge を GitHub が断る(gh は非 0 で終わる)。本物の GitHub の挙動は文書の根拠として書き、テストは偽の gh の振る舞いで行う
- director は gh を直接打てるので、動詞で auto_merge を強いても防げない。auto_merge は skill が読む印で、動詞は auto_merge によらず打てる

## Affected areas

- `internal/org/mergeorder.go`(`pr_lookup`、ブランチ 1 本の読み取りを repository を固定して動詞から呼べる形に)、`internal/org/mergeorder_test.go`
- `internal/org/mergeguarantee.go`(rules の読み取りを `CheckMergeEnforcement` と共有し、`ruleset_id` を残す。`CheckMergeGuarantee` と doctor の出力とキャッシュの形は変えない)、`internal/org/mergeguarantee_test.go`
- `internal/org/mergeenforce.go`(新規、D11 の `CheckMergeEnforcement`。S3a で mergeguarantee.go に書き、self-review の M1 を受けて S6 で移す)、`internal/org/mergeenforce_test.go`(新規、同じく移す)
- `internal/org/merge.go`(新規、S3a の確かめ)、`internal/org/merge_test.go`(新規)、`internal/org/merge_run.go`(新規、S3b の `Merge`: ロック、merge、読み直し、記録。merge.go が 800 行を超えないように分ける)、`internal/org/merge_run_test.go`(新規)、`internal/org/merge_wait.go`(新規、S7 の検査の待ち、D13)、`internal/org/merge_wait_test.go`(新規)
- `internal/org/seat.go`(event `pr_merged`)、`internal/org/regate.go` と `internal/org/spawn.go`(regate.lock の注と断りの文を「regate か merge」に)
- `internal/config/config.go`(`AutoMerge`)、`internal/config/config_test.go`、`templates/base/ralph.toml`、`scripts/ralph-config.sh` と `templates/base/scripts/ralph-config.sh`(既定値の写し `RALPH_ORG_DIRECTOR_AUTO_MERGE`)、`internal/config/defaults_sync_test.go`(3 面の一致)
- `internal/cli/org_merge.go`(新規)、`internal/cli/org_merge_test.go`(新規)、`internal/cli/org.go`(動詞の登録、main worktree の設定の読み方)、`internal/cli/status.go`、`internal/cli/org_director.go`、`internal/cli/org_legacy_ledger.go`、`internal/cli/org_merge_order.go`(help に `pr_lookup`)
- `.claude/skills/director/SKILL.md` と写し 3 面、`.claude/skills/org/SKILL.md` と写し 3 面、`internal/org/director_skill_sync_test.go`(skill の文を pin する)
- `README.md`、`AGENTS.md`、`docs/specs/2026-10-07-org-multi-org-director.md`、`docs/quality/quality-gates.md` と `templates/base/docs/quality/quality-gates.md`、`docs/tech-debt/README.md`

## Visual review

- Page: `.harness/state/plan-visual/org-auto-merge.html`(かんたん版の図 2 つ: `overview`、`easy-checks`。折りたたみに `detail-overview`、`detail-sequence`、`detail-records`、`detail-director`)
- Lint: `./scripts/plan-visual.sh lint` が exit 0
- Self-check: ページ全体と 6 つの図を `shot` で撮って読んだ。Codex の指摘のあと、`easy-checks`(確かめの 7 項目)、承認の前に、`detail-overview`(`mergeguarantee.go` と S3a・S3b)、`detail-sequence`(14 段)、`detail-director` を描き直して撮り直し、どれもきれいになった
- 2026-10-17(S6・S7 を足したとき): `overview`、`easy-checks`、承認の前の箱、`detail-overview`(`mergeenforce.go`、`merge_wait.go`)、`detail-sequence`(7 の待ち)、`detail-director`(`--wait-checks 60m`)を直し、lint が exit 0、5 つの図を撮り直して読み、どれもきれいになった
- Codex plan advisory: rc=0、`-o` は空でない。HIGH 3 件(別の repository、merge queue の予約、検証していない base)を D6・D10・D11・D12 と AC4・AC5・AC7・AC8 で直した。直し方は consult(consult-plan-automerge)の 2 回目の判定と、足された穴 4 つに従った

## Design decisions

- D1 auto_merge の効かせ方: 動詞は auto_merge によらず打てる。auto_merge は `/director` skill が読む「人に聞かずに merge と取り直しをしてよいか」の印で、`ralph status` の `director:` の行と `ralph org director status` と動詞の出力に値を出す。director は gh を直接打てるので、動詞で止めても強制にならない。人に頼まれた merge もこの動詞で打てば、素の `gh pr merge` より確かめが増える(consult)。受け入れ条件(auto_merge が false なら merge しない)は skill の文で満たし、skill の文を Go のテストで pin する
- D2 merge の方式: 設定を足さず、`gh repo view <repo> --json viewerDefaultMergeMethod` の方式で打つ(`gh repo view` は `-R` を持たず、repository を引数で受ける)。GitHub の画面で merge する人が既定で得る方式と同じで、変えたい人は repo の設定で変えられる。repo や ruleset が許さない方式なら GitHub が断る。知らない値なら動詞が断る
- D3 排他: 動詞は `regate.lock` の排他を取る(取れなければ 1 秒まで打ち直し、それでも取れなければ断る)。取った直後に印を空にし、merge の間は印を書かない。merge は worktree に触らないので、Spawn は空の印を読んで進む(regate の D9 の規則のまま)。これで regate と merge が台帳ごとに 1 つずつになり、regate が merge 中のブランチに push することもない
- D4 記録: gh が MERGED を返したあとに、manifest のロックの下で org の記録 `pr_merged` を 1 行書く。org_id は結びつきの org(結びつきが読めないブランチでは、gate の記録の org)、Details は `branch=`、`pr=`、`head=`、`method=`、`merge_commit=`、`auto_merge=`。仕様のユーザーストーリー 5(全 org の起動・承認・相談・merge を台帳と reports で追う)のため。断りと失敗は書かない。`pr_merged` は座席の状態の記録ではなく、走っている org の判定、`stop --all` と `disband --all` の対象、`max_orgs` の数え方、見張りの leader の動きの数え方に入らない
- D5 走っている org: D4 の org が `RunningOrgs` に入っていれば断る(その org が今ほかの機能を走らせていても断る)。leader は stop のあとに push と PR を打ち、RESULT を上げてから disband するので、走っている間の merge は leader の push と競る。断りの文は「受信箱の RESULT の件を処理し、org が disband してから打ち直す」と言う
- D6 merge のあとの読み直し: `gh pr merge`(60 秒で区切る)を打ったら、終了コードによらず、`gh pr view <番号> --json state,mergeCommit,autoMergeRequest -R <repo>` を 1 回読む。gh が非 0 で終わったか時間切れでも、GitHub の側では merge か予約が済んでいることがあるため。MERGED なら D4 を書いて終了コード 0(gh が非 0 だったら、その stderr の末尾を注として出す)。MERGED でなければ、`autoMergeRequest` があれば(D11 の確かめのあとに queue ができた場合など、gh が merge の代わりに予約した)`gh pr merge <番号> --disable-auto -R <repo>` で外し、予約を外したことを言って終了コード 1。外す gh が落ちたら「予約が残っているかもしれない」と、外すコマンドを出して終了コード 1。予約がなければ、gh が非 0 ならその stderr の末尾を、0 なら「gh は 0 で終わったが PR は merge されていない」を言って終了コード 1。読み直しが落ちたら、PR の状態がわからないことと確かめるコマンドを言って終了コード 1。MERGED のとき以外は記録しない。merge の呼び出しが始まったあとに中断されても、読み直しと外しと記録は `context.WithoutCancel` と gh の timeout(記録は manifest のロックの待ち時間)の下で 1 回打ち、中断されたことを言って終了コード 1。前の版の「QUEUED: で終了コード 0」は捨てた(Codex の指摘 2)。前の版の「終了コード 0 のときだけ読み直す」は、gh の時間切れのあとに merge された PR を記録し損ねるので、S3b の前に改めた
- D7 検査: `gh pr checks` の bucket がすべて `pass` か `skipping` で、`pass` が 1 つ以上のときだけ進む。`pending`・`fail`・`cancel` が 1 つでもあるか、検査が 0 件なら断り、名前と bucket を並べる
- D8 merge の直前の ls-remote: 検査と保護を読んだあと、merge の直前に ls-remote を打ち直し、既定のブランチの先頭が gate の base、ブランチの先頭が gate の head のままかを確かめる。動いていれば何もせずに断る。`--match-head-commit` は PR の head を守り、base は D11 の up-to-date の保護が GitHub 側で守るので、この確かめは早い断りにあたる
- D9 merge-order の `pr_lookup`: entry に 5 つの値の field を足す。`failed` の理由の文は今までどおり notes に置く。動詞は `found` のときだけ進む
- D10 対象の repository(Codex の指摘 1): gh は `-R` がないと cwd の remote から repository を選び、`upstream` の remote や `gh repo set-default` があればそちらを選びうる。動詞は origin の URL から host と owner/repo を読み(`mergeOriginFromURL`、host は doctor と同じ `mergeGuaranteeAPIHost`)、読めなければ断る。gh はすべてこの repository に固定し(`-R <host>/<owner>/<repo>`、`gh api --hostname <host>`)、PR の url の host と owner/repo が一致することも確かめる。merge-order の gh は変えず、動詞は名指しのブランチだけを固定した gh で読み直す
- D11 保護(Codex の指摘 3、consult): 動詞は merge の前に、新しい関数 `CheckMergeEnforcement`(`internal/org/mergeenforce.go`。読み手は `mergeguarantee.go` と共有)で保護を live で読む。キャッシュは使わない。`CheckMergeGuarantee` と同じ読み手(rules、classic の保護)を共有し、doctor の判定と出力とキャッシュの形は変えない。進むのは次のどちらかのときだけ:
  - ruleset の up-to-date の rule(`strict_required_status_checks_policy`)が自前の required check を持ち、その rule の ruleset を `gh api repos/<o>/<r>/rulesets/<id>` で読むと `current_user_can_bypass` が `never`(`always` と `pull_requests_only` は断る。`pull_requests_only` は PR の merge でこそ迂回が効く)
  - classic の保護が strict で required check を持ち、打つ本人が repository の admin でない(`gh api repos/<o>/<r>` の `permissions.admin` が false)か、`.../protection/enforce_admins` が enabled
  - merge queue があれば断る: GraphQL の `repository.mergeQueue(branch:)` を主に(classic の保護の queue も拾う)、rules の `merge_queue` を補助に読み、どちらかが queue を示すか、どちらかが読めなければ断る
  - どれかが読めなければ断る。classic の strict は admin でないと読めないので、admin でない director は classic の保護の repo で merge できない。断りの文は「ruleset で up-to-date の rule を設定する」と言う
  - 動詞は D1 のとおり auto_merge によらずこれを強いる(consult: 印で外せる形では Codex の指摘に答えたことにならない)
- D12 auto-merge の予約: PR に `autoMergeRequest` がすでにあれば、merge の前に断る。人や前の中断が残した予約が、動詞の merge と別に進むのを人に見せるため。断りの文は外すコマンド(`gh pr merge <番号> --disable-auto -R <repo>`)を言う
- D13 検査の待ち(self-review の M2、consult-plan-checkswait): leader は gate のあとに PR を出してすぐ disband し、regate は push で CI をやり直させるので、merge-order が `ready` と出す時点では検査がまだ走っていることが多い。Claude Code の director は受信箱を時間切れなしで待つので、決まった間隔で起きて打ち直す「回」がない。そこで待ちを動詞に置く。`ralph org merge --wait-checks <時間>`(Go の時間の書き方、既定 0 で今の動き、負の値と 6 時間を超える値は断る)は、確かめ(D7)が「検査が待っている(`fail` と `cancel` がなく `pending` がある)」か「0 件」で断るときだけ、断りを返さずにロックを外し、30 秒ごとに名指しの PR の `gh pr checks` だけを読み直す。待っている検査がなくなる(すべて終わる、または fail か cancel が出る)か、gh が落ちるか、時間が切れたら、ロックを取り直して確かめを最初から打つ。時間が切れたあとの確かめがまた検査待ちか 0 件なら、待った時間を添えて今の断りで終える。待ちに入るかどうかは確かめの断りの文ではなく、確かめが返す種類(待っている、0 件、ほか)で決める。待つ間は regate.lock を持たないので、regate とほかの merge は走れる。待つ間に head や base が動いても、最後の確かめが拾う。待つ間の中断は何も merge せず終了コード 1。待ちに入るときと終えるときに stderr に 1 行ずつ出す。CI のない repo では、0 件のまま時間が切れるまで待つ。skill は `auto_merge = true` のとき `--wait-checks 60m` を付けて background で打ち、断りはすべて今の文のとおり止める。skill に待ちと回数を数えさせる案(LLM が数える、0 件で `gh pr checks --watch` がすぐ返る)と、止まるのを想定の動きとする案(FR-8 の「CI の通った PR を merge する」がほぼ効かない)は捨てた
- Critical forks: consult-plan-automerge の 2026-10-17 の 2 つの判定で、D1(助言だけ)、D2(gh の既定の方式)、D3(regate.lock を共有、印は書かない)、D4(`pr_merged` を書く)、D5(断る)、D9(5 つの値)を選び、Codex の HIGH 3 件の直し方として D6(queue の道を捨て、予約は外して終了コード 1)、D10(repository を固定)、D11(保護と迂回を live で読み、常に強いる。doctor と別の関数)、D12 を選んだ。S3 は S3a と S3b に分けた。D13 は self-review の M2 を受けた consult-plan-checkswait(2026-10-17)の判定で選んだ

## Acceptance criteria

- [ ] AC1 merge-order の `--json` の entry に `pr_lookup` があり、gh の結果ごとに `found`・`none`・`failed`・`limit_forks`・`skipped` になる。gh がないときと、ほかの repository を読んで PR を捨てたときは、すべての entry が `skipped`。text の出力は変わらない
- [ ] AC2 `[org.director].auto_merge` は既定 false で、`true` を読める。真偽値でない値は設定の検証エラー。台帳が main worktree のものなら、feature worktree の `ralph.toml` ではなく main worktree の `ralph.toml` から読む。`ralph status` の `director:` の行と `--json`、`ralph org director status` に値が出る
- [ ] AC3 `ralph org merge --branch <b>` は次のどれでも、`gh pr merge` を打たず、何も記録せず、理由と次の手を出して終了コード 1 で終わる: ブランチの名指しがない、`--org-id` を渡した、ブランチが台帳にない、merge-order の class が `ready` でない(class と Detail を出す)、`pr_lookup` が `found` でない、PR が OPEN でない、PR の head が gate の head と違う、PR の base が既定のブランチでない、結びつきの org が走っている、PR に auto-merge の予約がある、検査が待っているか落ちたか取り消されたか 0 件、merge の方式が読めないか知らない値、ほかの regate か merge が台帳のロックを持っている、merge の直前の ls-remote で既定のブランチかブランチの先頭が動いた
- [ ] AC4 対象の repository: origin の URL が host と owner/repo を持たなければ断る。動詞が打つ gh はすべて `-R <host>/<owner>/<repo>`(`gh api` は `--hostname <host>`)を持つ。PR の url の host か owner/repo が origin と違えば断る(同じ owner/repo で host が違う場合を含む)
- [ ] AC5 保護: 次のどれでも断る(AC3 と同じ終わり方): merge queue がある(GraphQL か rules のどちらかが示す)、queue の有無が読めない、up-to-date の rule がない(`pr_checks_only` にあたる)、up-to-date の rule が自前の required check を持たない、ruleset の `current_user_can_bypass` が `never` でない、classic の strict の保護で打つ本人が admin で `enforce_admins` が無効、保護のどれかが読めない。進む 2 つの場合(ruleset の never、classic の非 admin か enforce_admins)は通る。doctor の Merge guarantee の判定とキャッシュの形は変わらない(既存のテストが通る)
- [ ] AC6 通る道: `gh pr merge <番号> <方式の flag> --match-head-commit <gate の head> -R <repo>` を 1 回打ち、`--auto`・`--admin`・`--delete-branch` を付けない。PR が MERGED になれば `pr_merged`(D4 の Details)を書き、`MERGED:`(PR の url)、`METHOD:`、`HEAD:`、`MERGE COMMIT:`、`AUTO MERGE:`、`PROTECTION:`(D11 で進んだ理由)の行を出して終了コード 0
- [ ] AC7 merge のあと、読み直しで MERGED でないとき(D6): 予約があれば `--disable-auto` を打って終了コード 1、外す gh が落ちたら外すコマンドを出して終了コード 1、予約がなければ終了コード 1、読み直しが落ちたら確かめるコマンドを出して終了コード 1。どれも記録しない。merge の呼び出しのあとの中断でも、読み直しと外し(MERGED なら記録)を 1 回打ち、終了コード 1
- [ ] AC8 `gh pr merge` が落ちたか時間切れでも、読み直しを 1 回打つ。MERGED でなければ(最後の確かめのあとに base が進み、GitHub が断った場合を含む。偽の gh で試す)、gh の stderr の末尾を出し、何も記録せずに終了コード 1。MERGED なら記録し、gh の stderr の末尾を注として出して終了コード 0。読み直しが MERGED を返したあとに記録が落ちたら、merge したことと記録できなかったことを言って終了コード 1
- [ ] AC9 `pr_merged` は、走っている org の判定、`stop --all` と `disband --all` の対象、`max_orgs` の数え方、見張りの leader の動きの数え方に入らない。`ralph org report` の時系列の表に出る
- [ ] AC10 merge が `regate.lock` を持つ間は regate が断られ、regate が持つ間は merge が断られる。merge の間も同じ org_id の Spawn は断られない
- [ ] AC11 `/director` skill: auto_merge が false なら、merge も regate も人に頼まれたときだけ打ち、merge は `ralph org merge` で打つ。true なら、merge-order の先頭の `ready` を `ralph org merge`(`--wait-checks 60m` を付けて background で打ち、終わってから出力を読む)で 1 本 merge し、merge-order を打ち直し、regate を名指すブランチに 1 本ずつ regate を打ち、`ready` に戻れば merge する。どちらでも `gh pr merge` を直接打たず、動詞の断りと失敗はそこで止めて人に伝え、打ち直さない。この文を Go のテストが pin する
- [ ] AC13 `--wait-checks <d>`(D13): 確かめが検査待ちか 0 件で断るときだけ、何も記録せずにロックを外し、30 秒ごとに `gh pr checks` だけを読む。待っている検査がなくなるか、gh が落ちるか、時間が切れたら、ロックを取り直して確かめを最初から打つ。fail・cancel・skipping だけの断りと、ほかのすべての断りでは待たない。時間が切れたあとの確かめがまだ検査待ちか 0 件なら、待った時間を添えた断りで終了コード 1。待つ間の中断は何も merge せず終了コード 1。待つ間は regate.lock を持たない(regate が走れる)。`0` と省略は今の動き、負の値と 6 時間を超える値は gh を打つ前に断る。help と skill がこれを書く
- [ ] AC12 文書: `/org` skill の動詞リファレンス、仕様 FR-8 の注、README、`AGENTS.md`、quality-gates の 2 面、tech-debt(237、238、240、242 行)、`templates/base/ralph.toml`、CLI の help が、動詞と設定と `pr_lookup` と保護の条件を書く

## Implementation outline

1. S1: merge-order の `pr_lookup`(`internal/org/mergeorder.go`)。`lookupMergePRs` がブランチごとの結果を target に残し、`mergeEntryFor` が entry に写す。ブランチ 1 本の PR を、固定した repository で読める形にする。CLI の help に field を足す
2. S2: 設定 `auto_merge`(`internal/config/config.go`)、main worktree の設定の読み方(`internal/cli/org.go`)、`ralph status` と `ralph org director status` の表示、`templates/base/ralph.toml`
3. S3a: 確かめ(`internal/org/merge.go` の前半と `internal/org/mergeguarantee.go` の `CheckMergeEnforcement`)。対象の repository、ブランチの読み直し、走っている org、予約、検査、保護と迂回、queue、方式、直前の ls-remote。gh と git は `MergeOrderExec` を通し、何も変えない。偽の gh の表で網羅する
4. S3b: merge と後始末(`internal/org/merge_run.go` と `internal/org/seat.go` の event `pr_merged`)。ロック、`gh pr merge`、読み直し、予約の外し、記録、中断
5. S4: CLI `ralph org merge`(`internal/cli/org_merge.go`)、動詞の登録と台帳を書く動詞の扱い、古い台帳の注、regate.lock の注と regate と Spawn の断りの文
6. S5: skill の 8 面、仕様、README、`AGENTS.md`、quality-gates、tech-debt、skill の文を pin するテスト
7. S6: self-review の直し。M1(`CheckMergeEnforcement` 一式とそのテストを `mergeenforce.go` と `mergeenforce_test.go` に移す。動きは変えない)と L1〜L8(使わない `mergeCommit` の要求、`pr_merged` の `auto_merge=` の注、merge のあとの gh を別のプロセスグループで、`withMainWorktreeDirectorEnabled` の注、エラーの前置き、origin の host と owner/repo の先頭の `-`、tech-debt 239 行のセル、誤記)
8. S7: `--wait-checks`(D13、`internal/org/merge_wait.go`)。確かめが断りの種類を返す形、CLI の flag と help、`/director` skill の merge の打ち方(background と `.harness/state/merge-<名前>.out`)、pin テスト、`/org` skill と README の `merge` の行

## Verify plan

- Static analysis checks: `./scripts/run-static-verify.sh`(golangci-lint、gofmt、secret scan、check-sync、check-skill-sync)、`go vet ./...`、`GOTOOLCHAIN=go1.25.8 go vet ./...`
- Spec compliance criteria to confirm: AC1〜AC12 を、コードと、HEAD から作った binary を偽の gh と手元の bare repo で打って確かめる。FR-8 の受け入れ条件は skill の文と pin するテストで確かめる。D11 の API の field 名(`ruleset_id`、`current_user_can_bypass`、`mergeQueue`、`permissions.admin`)が、計画の Assumptions に書いた実機の出力と合うことを確かめる
- Documentation drift to check: help、skill の 8 面の同一性、README の動詞の表、`AGENTS.md` の repo-map、仕様の注、quality-gates の 2 面、`templates/base/ralph.toml` と `config.Default()` の一致
- Evidence to capture: `docs/evidence/verify-2026-10-17-org-auto-merge.log`(run-static-verify の出力、probe のスクリプトと出力)

## Test plan

- Unit tests: `pr_lookup` の 5 つの値(gh がない、落ちた、0 件、上限の fork、ほかの repository、問い合わせる前の中断)。設定の既定と読み込みと検証。動詞の AC3〜AC5 の断りを 1 つずつ、AC6 の flag の並びと `-R`、AC7、AC8、検査の読み方の境目(pass だけ、pass と skipping、skipping だけ、pending を 1 つ、0 件、終了コード 8 の JSON)、方式の 3 つと知らない値、`CheckMergeEnforcement` の表(ruleset の never・always・pull_requests_only、classic の非 admin・admin と enforce_admins の有無、strict が読めない、queue が GraphQL だけ・rules だけ、どちらかが読めない)
- Integration tests: 偽の gh(`MergeOrderExec`)と手元の bare repo を origin にした台帳で、CLI の `ralph org merge` を打ち、出力の行と終了コードと manifest を確かめる。`ralph status` の `director:` の行
- Regression tests: merge-order の text の出力が変わらない(既存のテスト)。doctor の Merge guarantee の判定とキャッシュが変わらない(既存のテスト)。`pr_merged` が `RunningOrgs`、`orgsToDisband`、`max_orgs`、`leaderActivityEventCount` に入らない。regate と merge のロックの排他、merge の間の Spawn
- `--wait-checks`(S7): pending から pass に変わって merge する、0 件から pass に変わる、pending から fail に変わって待たずに断る、時間切れ、待つ間の中断、待つ間の gh の失敗、待つ間に head が動く、待つ間に regate がロックを取れる、fail と cancel と skipping だけでは待たない、負の値と 6 時間を超える値
- Edge cases: merge の直前に既定のブランチが動く、PR の head が gate の head と違う、検査が 0 件、gh が終了コード 8 で JSON を返す、merge のあとの読み直しが落ちる、予約を外す gh が落ちる、記録が落ちる、merge の呼び出しの途中の中断、`viewerDefaultMergeMethod` が空、同じ owner/repo で host が違う PR、upstream の remote がある checkout(`-R` で origin を打つ)
- Evidence to capture: `docs/evidence/test-2026-10-17-org-auto-merge.log`(`./scripts/run-test.sh`、`go test -race`、`GOTOOLCHAIN=go1.25.8`、カバレッジ)

## Risks and mitigations

- 最後の確かめと `gh pr merge` の間に既定のブランチが動く: D11 の up-to-date の保護が効いていれば、GitHub が遅れた head の merge を断るので、窓は「gh が非 0 で断る」形に変わる(AC8)。本物の GitHub の文は確かめていない。D8 の ls-remote は早い断り
- この repo の main の ruleset は required status checks を持たない(2026-10-17 に `gh api repos/yoshpy-dev/ralph/rules/branches/main` で確認、`pull_request` と `deletion` と `non_fast_forward` だけ)ので、動詞はこの repo では merge できない。dogfood(9 段目の試行)には ruleset に required check(verify)と up-to-date の rule を足す必要がある。repo の設定の変更は人に頼む
- classic の保護の strict は admin でないと読めないので、admin でない director は classic の保護の repo で merge できない(断りの文で ruleset を勧める)
- `gh pr checks` が commit status だけを書く外部の CI で 0 件になる経路は確かめていない(おそらく出る。未確認)。0 件なら断るので、merge する側には倒れない
- D11 の API の field は名前を実機で確かめたが、queue がある repo と迂回できる ruleset の値は見ていない。知らない値と読めない値は断るので、merge する側には倒れない
- 予約を外す前にプロセスが落ちると、GitHub に auto-merge の予約が残りうる。次の動詞の呼び出しは D12 で断り、外すコマンドを言う。merge-order はこの予約を見ない
- director が auto_merge を読み違えて merge するおそれ。動詞は auto_merge を強いないが、ready と検査と保護の確かめは常に通る。出力に `AUTO MERGE:` の値を出し、skill の手順は先に `ralph status` で値を読む
- merge の呼び出しの途中で中断すると、merge されたのに記録が残らないことがある。出力で merge-order の打ち直しを言う
- `--wait-checks` の待ちは 30 秒ごとに gh を 1 回打つ(60 分で 120 回)。GitHub の API の上限(認証済みで 1 時間に 5000 回)に比べて小さいが、待つ merge を並べると増える。skill は 1 本ずつ打つ
- CI のない repo では、`--wait-checks` は 0 件のまま時間が切れるまで待ち、0 件の断りで終わる。D7 が pass を 1 つ以上求めるので、そうした repo ではもともと動詞で merge できない
- `regate.lock` の名前が merge にも使われる。ファイル名は変えず、注と断りの文で「regate か merge」と言う

## Rollout or rollback notes

- 新しい動詞、設定、merge-order の JSON の field、event が増えるだけで、既存の記録の形と分類と doctor の判定は変わらない。戻すときは revert する。`pr_merged` の行は、revert したあとの ralph でも走っている org の判定に入らない(読み手は知らない org の記録を読み飛ばす)。GitHub に予約は残さない作り(D6、D12)なので、revert のあとに GitHub 側で進む merge はない
- 使う前に repo の側で要ること: 既定のブランチに、required check を持つ up-to-date の ruleset(打つ本人が迂回できないもの)を設定し、merge queue を使わない。この repo は今は満たさない
- release のノートに足すこと: `ralph org merge`、その保護の条件、`[org.director].auto_merge`(既定 false。`ralph upgrade` は seed の `ralph.toml` に足さない)、event `pr_merged`、merge-order の JSON の `pr_lookup`、regate と merge が同じ台帳のロックを使うこと

## Open questions

- なし

## Progress checklist

- [x] Plan reviewed
- [x] Plan approved(2026-10-17、ユーザーの永続の指示による自動承認)
- [x] Branch created
- [x] Implementation started
  - S1 fa49d310(merge-order の `pr_lookup`)、S2 cddb608c(`auto_merge` の設定と表示)
  - 2026-10-17: Affected areas に `ralph-config.sh` の 2 面と `defaults_sync_test.go` を足した(`[org.director]` のほかの既定値と同じく 3 面で揃える。S4 で入れる)。digest を取り直して承認し直した
  - S4 に回すもの: `internal/cli/org.go` の `--config` の usage と `internal/cli/org_director.go` の group help が `[org.director].enabled` だけを名指す(auto_merge も読む)。S5 に回すもの: `docs/tech-debt/README.md` の `statusDirectorEnabled`(`statusDirectorConfig` に改名した)
  - S3a acc81fef(merge の前の 13 段の確かめと `CheckMergeEnforcement`)。GraphQL の変数は `-F` でなく `-f`(`-F` は true や数や `@` で始まる値を読み替える)
  - 2026-10-17(S3b の前): D6・AC7・AC8 を「`gh pr merge` の終了コードによらず読み直し、MERGED なら記録する」に改め、D2 の `gh repo view` の形を直し、S3b を `merge_run.go` に分けた。図 1 と図 2 を直して撮り直し、digest を取り直して承認し直した
  - S3b df8958a5(`Merge`、`pr_merged`、読み直しと予約の外しと記録、中断)。mutation 22/22。テスト用の `mergeTestHooks.afterChecks` を足した(確かめのあとの中断を起こすため)。merge_commit は git の SHA の形のときだけ使い、ほかは unknown
  - S4 に足すもの: `internal/org/lockfile.go` の `heldFileLock` の注(regate.lock を握るのは regate だけと書いている)、`internal/org/seat.go` の package doc(gh を打つのは mergeorder.go だけと書いている。merge.go と merge_run.go も打つ)
  - S5 の tech-debt に足すもの: `mergeLsRemoteRefusal` と `mergeGHSaid` が merge-order の文言と `realMergeOrderExec` の err の形(`<exit err>: <stderr>`)に頼る。`merge_test.go`(835 行)と `merge_run_test.go`(842 行)が 800 行の目安を少し超える。検査が 0 件の PR で `gh pr checks` が実際に何を出すかは確かめていない(Risks の行のまま)
  - S4 e7624d79(CLI の `ralph org merge`、`RALPH_ORG_DIRECTOR_AUTO_MERGE` の 2 面、regate の拒否の文言、lockfile・seat・spawn・regate の注)。mutation 16/16。fake gh はシェルスクリプトで、Go 側に差し込み口は足していない。結果が何も立たないときもエラーにし、exit 0 は「merge して記録した」だけになる
  - S5 71c0582e(skill の 8 面、README、`AGENTS.md`、仕様の FR-8 の注、quality-gates の 2 面、tech-debt、pin テスト `TestDirectorSkill_PinsTheAutoMergeSteps`)。pin は 6 つの断片とその順、古い文「`gh pr merge` で 1 本 merge し」がないこと。mutation 3/3。`/director` に「headless の director を `--config` で立てたとき、`director status` と `ralph status` の auto_merge が違えば false として扱う」を足した。tech-debt 238 行は外部コマンドの runner の重複が残るので、JSON の半分だけを RESOLVED にした
  - sync-docs に回すもの: tech-debt 238 行が `regateRemoteGit` を `internal/org/regate.go` に置いている(今は `regate_git.go`)
  - 2026-10-17(self-review cycle 1 のあと): report 29469c0e(MEDIUM 2、LOW 8)。M2(auto_merge = true でも検査待ちの断りで毎回止まる)を consult-plan-checkswait に聞き、動詞に待ちを入れる D13 と AC13 を足した。M1 を受けて Affected areas と D11 に `mergeenforce.go` を書いた。Implementation outline に S6(self-review の直し)と S7(`--wait-checks`)を足し、Scope・Test plan・Risks を直した。図 5 つを直して撮り直し、digest を取り直して承認し直した(ユーザーの永続の指示による自動承認)。self-review の直しは cycle を増やさず、S7 のあとに reviewer に S6・S7 の diff を見せてから verify に進む
  - S6 4a39e909(M1 の移動、L1〜L8)。M1 は消えた行と増えた行を突き合わせ、違いが L8 の doc の直しと新しいファイルの先頭だけと確かめた。merge のあとの読み直しと `--disable-auto` は `realMergeDetachedExec`(Setpgid、ctx が終わればグループごと kill)で打つ。`gh pr merge` と確かめは ralph のグループのまま。origin の host・owner・repo が `-` で始まれば merge だけが断る(merge-order は変えない)。mutation 7/7。tech-debt 238 行の `regateRemoteGit` の場所も直した
- [ ] Review artifact created
- [ ] Verification artifact created
- [ ] Test artifact created
- [ ] PR created
