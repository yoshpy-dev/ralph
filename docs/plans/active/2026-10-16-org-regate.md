# org-regate

- Status: Approved
- Approved: 2026-10-16 sha256:f841204a7bc0
- Owner: Claude Code
- Date: 2026-10-16
- Related request: 機能ごとの org と director の仕様(`docs/specs/2026-10-07-org-multi-org-director.md`)の 8 段目の前半(8a)の 3 本目。FR-7 の「後から入る PR は、main に rebase してゲートを取り直してから出す」のうち、ほかの PR の merge のあとの取り直し。8a-1 の計画(`docs/plans/archive/2026-10-14-org-director-attach.md`)の D1 が、これを `auto_merge` と一緒に 3 本目に残した。consult(consult-plan-automerge、2026-10-16)の判定で 3 本目を 2 つに分け、取り直し(この計画、8a-3b)を `auto_merge`(8a-3a)より先に入れる。ユーザーは「すべてが完了するまで次のプランを自律的に進めて」と言った(2026-10-11)
- Related issue: N/A
- Type: feat
- Branch: feat/org-regate

## Objective

ほかの機能の PR が merge されたあとで、disband 済みの機能の org のブランチを、ralph の動詞 1 つで merge してよい状態(`ralph org merge-order` の `ready`)に戻せるようにする。新しい動詞 `ralph org regate --branch <ブランチ>` は、そのブランチの機能の worktree で origin の既定のブランチを取り込み、gate のスクリプトを走らせて記録し、push する。今は、機能の org が PR を出して disband すると `ralph org gate` がその org を断るので、`rebase` や `regate` になったブランチを ralph で `ready` に戻す道がない(tech-debt の merge-order の行)。

## Scope

- 動詞 `ralph org regate --branch <b>`(`internal/org/regate.go` と CLI)
  - 台帳から、ブランチ `b` を名指す最新の完全な結びつきを読む。その org が走っていれば断る
  - 結びつきの worktree で、gate と同じ確かめ(worktree、ブランチ、きれいな作業ツリー、スクリプト)をする
  - origin から既定のブランチと `b` を fetch する。HEAD が `origin/<b>` と同じであることを確かめる
  - 既定のブランチの先頭を HEAD が含まなければ、merge commit で取り込む。衝突したら merge を戻し、何も記録せずに断る
  - gate のスクリプトを走らせ、`gate_passed` か `gate_failed` を、結びつきの org の org_id で manifest に書く
  - 通れば、force なしで `origin/<b>` に push する。push が落ちたら、打ち直しは push だけをする
  - `gate_passed` を記録する前に、スクリプトが落ちたとき、中断されたとき、結びつきが変わったときは、自分が作った merge commit を戻す
- regate と Spawn の排他: regate が走っている間、同じ org_id の `ralph org spawn` と `ralph org start --plan` を断る(`internal/org/spawn.go`、`internal/org/feature.go`)。ほかの org_id は止めない
- `ralph org merge-order` の理由の欄: 結びつきの org が走っていないブランチでは、`ralph org gate --org-id` の代わりに `ralph org regate --branch <b>` を出す
- 文書: `/org` skill の 4 面(動詞リファレンスの行と「merge してよい順」)、`/director` skill の 4 面(人に頼まれたら regate を打つ手順)、仕様 FR-7 の注、tech-debt、`AGENTS.md` の repo-map、CLI の help

## Non-goals

- `[org.director].auto_merge`、ralph の merge の動詞、merge-order の JSON での gh の失敗の区別(8a-3a)。director が regate を自分から打つのも、auto_merge のとき(8a-3a)に限る
- 衝突の解消。regate は衝突を直さず、人か、立て直した org に返す
- rebase と force push。取り込みは merge commit だけにする(D3)
- 走っている機能の org のブランチ。走っている間は leader が `ralph org gate` を打つ
- leader の雛形(`internal/org/prompts/leader.md`)の手順 9〜16 と、tech-debt の leader の雛形の行(230・231 行)
- worktree の後始末(`ralph-worktree.sh cleanup`)。worktree がなければ regate は断るだけ
- merge-order の分類そのものの変更。変えるのは理由の欄の文だけ

## Assumptions

- 機能の org の worktree とブランチは、PR の merge のあとに人が cleanup するまで残る(leader の雛形の手順 15)
- `ralph org gate` は HEAD が既定のブランチの base ref の先頭を含むことだけを見る(`gateCheckAncestor`、`git merge-base --is-ancestor`)。merge commit で取り込んだ head も通る。7b の注(仕様 FR-7)で「rebase して」は「push する head で最後の gate を取る」と読み替え済みで、この計画もその読み方に従う
- gate の記録(`gate_passed`、`gate_failed`)は org を running に数えない(`RunningOrgs` は座席、workspace、予約だけを見る。`seat.go` の注)。disband 済みの org の org_id で書いても、上限と予約と承認の検査は変わらない
- 機能の org は disband するまで予約を持つので、走っていない機能の org は disband 済みである
- commit-msg の hook(`scripts/commit-msg-guard.sh`)は `Merge ` で始まる件名を形式の検査から外す。pre-merge-commit の hook(`scripts/pre-merge-commit-secret-guard.sh`)は取り込む中身の secret scan をする。どちらも bootstrap が `.git/hooks` に入れたときだけ走るので、regate は hook が走ることを前提にしない
- テストは origin をローカルの bare repository にし、ネットワークに出ない

## Affected areas

- `internal/org/regate.go`(新規)、`internal/org/regate_test.go`(新規)
- `internal/org/gate.go`(記録の `via=regate`、確かめの関数を regate から呼べる形に。振る舞いは変えない)
- `internal/org/spawn.go`、`internal/org/feature.go` と、それぞれのテスト(regate が走っている間の同じ org_id の spawn と start を断る。D9)
- `internal/org/mergeorder.go`、`internal/org/mergeorder_test.go`(理由の欄の文)
- `internal/cli/org_regate.go`(新規)、`internal/cli/org_regate_test.go`(新規)、`internal/cli/org.go`(動詞の登録)
- `.claude/skills/org/SKILL.md`、`.claude/skills/director/SKILL.md` と、それぞれの写しの 3 面(`.agents/skills/`、`templates/base/.claude/skills/`、`templates/base/.agents/skills/`)
- `docs/specs/2026-10-07-org-multi-org-director.md`(FR-7 の注)、`docs/tech-debt/README.md`、`AGENTS.md`

## Visual review

- Page: `.harness/state/plan-visual/org-regate.html`(かんたん版の図 2 枚 `#overview`・`#easy-failure`、「承認の前に」の欄、閉じた折りたたみの中に `#detail-overview`・`#detail-paths`・`#detail-failure`)
- Lint: `./scripts/plan-visual.sh lint .harness/state/plan-visual/org-regate.html` が終了コード 0
- Self-check: `./scripts/plan-visual.sh shot` でページ全体と 5 枚の図を 1 枚ずつ撮り、PNG を読んで確かめた。文字のはみ出し、線と文字の重なり、箱の重なりはない。consult と Codex の助言を入れたあと(スライスを 5 本に、Spawn との排他、失敗の表に記録のあとの中断)も撮り直して確かめた

## Design decisions

- D1 動詞と単位: `ralph org regate --branch <b>`。merge-order の単位がブランチなので、ブランチで名指す。`--org-id` は断る(結びつきはブランチから読む)。台帳は他の動詞と同じ規則で決め、台帳を書く動詞として扱う(`orgLedgerMutating`)。新しい作業を始めないので、見張りも承認された計画も要らない
- D2 結びつきと org の状態: ブランチ `b` の結びつきと最新の gate の記録は、merge-order と同じ `readMergeLedger` と `collectMergeTargets` で、manifest のロックの下で読む(ロックなしの読みは追記中の最終行を壊れた行に数える。tech-debt の行)。結びつきは、`b` を名指す完全な結びつきのうち manifest で最後のものを使う。その org_id が `RunningOrgs` に入っていれば断る(leader がその worktree を使っている。org_id を別の機能で立て直した場合も、記録が今の leader の動きに数えられないよう断る)。結びつきがなければ断る。最新の gate の記録が読めないか、そのあとに壊れた行がある(merge-order の `unreadable`)ときは、その前の成功の記録を使わない(AC3 と D4 の例外に当てない)
- D3 取り込みは merge commit: `git merge --no-ff --no-edit -m 'Merge <base_ref の短い名前> <base の 12 桁> into <b>' <base の SHA>`。rebase と force push をしない。GitHub の「Update branch」と同じ操作で、PR の差分は変わらない(consult の判定。B' の rebase と force push は、この repo の決まりと合わず、push 先を壊すおそれがある)。取り込む前に、HEAD が既定のブランチの先頭を含んでいれば merge しない
- D4 push する head を確かめる: fetch のあと、HEAD が `refs/remotes/origin/<b>` と同じでなければ断る(push していない作業を巻き込まない、誰かが push した先を上書きしない)。例外は push の打ち直しで、最新の gate の記録が読める `gate_passed`(D2)、その head が HEAD、base が今の既定のブランチの先頭、`origin/<b>` が HEAD の祖先のときだけ、merge も scripts も飛ばして push だけをする。例外に当たらずに断るとき、最新の記録が regate の書いた(`via=regate`)読める記録で、その head が HEAD で、HEAD の第 1 親が `origin/<b>` の先頭(regate 自身の、push していない merge commit)なら、`git -C <worktree> reset --keep refs/remotes/origin/<b>` のあとに regate を打ち直す手を出す。それ以外では人の push していない作業を消しうるので、その手は出さない
- D5 戻し方: regate が作った merge commit は、`gate_passed` を記録する前に限り、スクリプトが落ちたとき(`gate_failed` を記録したあと)、中断されたとき、結びつきや org が変わったときに、`git reset --keep <merge の前の HEAD>` で戻す。`gate_passed` を記録したあと(push の前と途中)の中断は push の失敗と同じに扱い、記録と merge commit を残す(origin が merge commit を受け取ったあとに応答が届かなかった場合に、戻すと origin と worktree が食い違い、打ち直しで直らないため)。打ち直しは fetch で push が届いたかを見て、届いていれば AC3、届いていなければ D4 の例外になる。戻すのは、HEAD がまだ regate の merge commit で、作業ツリーがきれいなときだけにする。スクリプトが走る間に人が commit した(`head_moved`)か、スクリプトがファイルを残した(`tree_changed`)ときは戻さない(`reset --keep` は人の commit を消し、汚れた作業ツリーでは止まるか変更が混ざる)。戻したときは、出力で「merge は戻した」と言い、LOG の行を出す(戻した merge commit はどのブランチからも届かなくなる)。戻さないときと戻せなかったときは、今の状態と戻すコマンドを出して終了コード 1 で終わる。失敗した merge を worktree に残す案は取らない(残すと D4 に「間の commit が regate の merge だけ」の例外が要り、その判定が regate 本体より重い。人が直すときは `git merge origin/<既定>` を自分で打てば同じ状態に戻る)
- D6 記録の確かめ: `recordGate` の life の確かめは、走っている org 用のままにし、regate 用に別の確かめを足す。manifest のロックの下で読み直し、ブランチ `b` の最後の完全な結びつきが同じもの(org_id と結びつきの中身)で、その org が走っていないときだけ書く。記録の Details は gate と同じ形に `via=regate` を足す。読み手の `gateRecordFromDetails` は知らないキーを飛ばすが、` log=` のあとを丸ごと log の値として読むので、`via` は `log` より前に置く
- D7 同時に走らせない: 台帳ごとのロック `regate.lock`(flock の排他)を、最初の確かめから push か巻き戻しの終わりまで持つ。Spawn が D9 で一瞬だけ共有で持つのと重ならないよう、取れなければ 1 秒まで短く打ち直し、それでも取れなければ断る。merge と巻き戻しが 2 つ重なると、一方の巻き戻しがもう一方の merge を消しうるため。台帳の書き込みは manifest のロックが守り、leader の `ralph org gate` は別の worktree で動くので、ブランチごとのロックは要らない(director は 1 本ずつ打つ)
- D8 ネットワーク: fetch は既定のブランチと `b` の 2 回に分け、それぞれ明示の refspec(`+refs/heads/<x>:refs/remotes/origin/<x>`)で取る。`remote.origin.fetch` の設定に頼らず、`b` が origin にないこと(`couldn't find remote ref`)を既定のブランチの失敗と分けて言うため。fetch と push には `GIT_TERMINAL_PROMPT=0` を付け、1 回 120 秒で区切る。fetch が落ちたら何も記録せずに断る
- D9 Spawn との排他: 新しい life を作る記録(`spawn_started`、`scope_reserved`、`org_workspace_created`)は Spawn の `withManifestLock` の中で書かれ、機能の worktree は `.claude/worktrees/org-<org_id>` で org_id と 1 対 1 である。そこで regate は、`regate.lock` の排他を取ったあと manifest のロックの下で、ロックファイルの中身に印(org_id、branch、worktree、pid)を書き、そのまま org が走っていないことを確かめる。Spawn は `withManifestLock` の判定の中で、冪等な打ち直しの判定のあとに `regate.lock` の共有を待たずに試す。取れれば regate は走っていないので、すぐ放して続ける。取れなければ印を読み、印の org_id が自分の org_id と同じときだけ、何も記録せずに断る。印がまだないか読めないときは続ける(regate が manifest のロックを待っている間で、そのあと regate が走っている org を確かめて断る)。regate は排他を取った直後に印を空にし(manifest のロックを待つ間に、前の regate が残した印で、その org の Spawn を断らないため)、manifest のロックの下で新しい印を書く。終わりには印を消さず、排他を放すだけにする(共有が取れれば Spawn は印を見ないので、残った印は害がない。終わりに消すと、push か巻き戻しの前に Spawn が空の印を読んで進みうる)。`ralph org start --plan` は手順 2(ロックなしの `spawnPrecheckErr`)でも同じ確かめをし、ensure の前に断る(ensure は既存の worktree を読むだけなので、間に regate が始まっても続く Spawn が断る)。regate の間にすべての start と spawn を断る案は、走っている別の org の leader の spawn まで止めるので取らない
- Critical forks: D2〜D5、D7、D9 は consult(consult-plan-automerge)の判定と、この計画への確認で決めた。D9 と D5 の push の扱い、D2 の壊れた記録の扱いは Codex の計画の助言(HIGH 1 件、MEDIUM 2 件)を受けて足した

## Acceptance criteria

- [ ] AC1 断る(何も記録せず、worktree も origin も変えず、終了コード 1、理由と次の手を stderr に出す): `--branch` がない、`--org-id` を付けた、`b` を名指す完全な結びつきがない、その org が走っている、worktree がない、worktree のブランチが `b` でない、detached HEAD、rebase か merge の途中、作業ツリーが汚れている、スクリプトがない、ほかの regate が走っている(D7)
- [ ] AC2 fetch が落ちる、origin に `b` がない(merge のあとに消された。理由でそう言う)、HEAD が `origin/<b>` と違う(D4 の例外を除く)ときも、AC1 と同じく断る。fetch と push に `GIT_TERMINAL_PROMPT=0` が付く
- [ ] AC3 HEAD が既定のブランチの先頭を含み、最新の gate の記録が読める、その HEAD と base の `gate_passed` なら、何も走らせず、記録も push もせず、そう言って終了コード 0。最新の記録が読めないか、そのあとに壊れた行があるときは、前の成功を使わずに AC9 の道で取り直し、そのあと merge-order がそのブランチを `ready` と出す(壊れた gate の Details と壊れた JSON 行の 2 つの場合)
- [ ] AC4 取り込みの道: HEAD が先頭を含まなければ、D3 の merge commit を作り、スクリプトを走らせ、`gate_passed`(head は merge commit、base と base_ref は取り込んだ先頭、branch は `b`、`via=regate`、org_id は結びつきの org)を記録し、force なしで `origin/<b>` に push する。stdout に `GATE: pass`、`MERGED:`、`HEAD:`、`BASE:`、`LOG:`、`PUSHED:` の行を出して終了コード 0。そのあと merge-order はそのブランチを `ready` と出す
- [ ] AC5 merge commit を作れない: 衝突、pre-merge-commit の hook が止めた、`user.name` か `user.email` がない、のどれでも、merge の途中なら `git merge --abort` で戻し、HEAD と作業ツリーが前のままであることを確かめ、何も記録せず、理由(衝突なら衝突したファイル)と次の手を出して終了コード 1
- [ ] AC6 スクリプトが落ちたら、`gate_failed`(head は merge commit)を記録し、merge commit を戻し(D5)、push せず、「merge は戻した」と LOG を出して終了コード 1。`gate_passed` を記録する前の中断(SIGINT、SIGTERM、SIGHUP)ならスクリプトを止め、何も記録せず、merge commit を戻す。`head_moved` と `tree_changed` では gate と同じく `gate_failed` を記録し、戻さず、push せず、今の状態と戻すコマンドを出す
- [ ] AC7 push が落ちたとき、または `gate_passed` を記録したあと(push の前か途中)に中断されたときは、記録と merge commit をそのままにして終了コード 1 になり、打ち直しを案内する。打ち直すと、push が届いていなければ D4 の例外で push だけをし、届いていれば(origin が受け取ったあと応答が届かなかった)AC3 になる。既定のブランチがその間に進んだとき、または fetch と push の間に誰かが `origin/<b>` に push した(push は non-fast-forward で落ち、`origin/<b>` が HEAD の祖先でなくなる)ときは、例外に当たらず断り、D4 の条件のときだけ `reset --keep` の手を出す。最新の記録が読めず HEAD が `origin/<b>` と違うときも、同じ条件で断る
- [ ] AC8 スクリプトが走る間に、その org が立て直されたか、ブランチの結びつきが変わったら、何も記録せず、merge commit を戻して終了コード 1
- [ ] AC9 gate だけの道: HEAD が `origin/<b>` と同じで既定の先頭を含むが、その head の通った記録がない(merge-order の `regate` や `failed` のあと、人が直して push した)なら、merge せずにスクリプトを走らせて記録する。push は要らない
- [ ] AC10 merge-order の理由の欄: 結びつきの org が走っていないブランチでは、`ralph org gate --org-id <org>` と「(the gate runs only while the feature org runs)」の代わりに `ralph org regate --branch <b>` を出す。走っている org が今結びついているブランチは今のまま。org_id が別の機能で立て直されて走っているときは、その org が別のブランチで走っていることと、disband のあとに `ralph org regate --branch <b>` を打つことを言う。最新の記録が regate の `gate_failed`(`via=regate`)のときは、その head が regate の merge commit であることと log のパスを理由で言う。`regate`(origin の先頭が gate の head と違う)の理由は、走っていない org のブランチでは「gate のあとに push された、または gate の head が push されていない」と言う(regate の push が落ちたあとの状態を読み違えないため)
- [ ] AC13 Spawn との排他(D9): regate が最初の確かめから終わりまでの間、同じ org_id の `ralph org spawn` と `ralph org start --plan` は何も記録せずに断られ(start は ensure の前に断る)、別の org_id の spawn は通る。regate が排他を取れないときは 1 秒まで打ち直してから断る。テストは、regate を最初の確かめのあとと記録の直後で止める test hook で確かめる
- [ ] AC11 文書: `/org` skill の 4 面に動詞リファレンスの `regate` の行を足し、「merge してよい順」の人の手順を regate に置き換える。`/director` skill の 4 面に、人に頼まれたら regate を 1 本ずつ打ち、衝突と失敗は人に伝える手順を書く。仕様 FR-7 の注、tech-debt の merge-order の行、`AGENTS.md` の repo-map、CLI の help
- [ ] AC12 テストはネットワークに出ず、本物の herdr、osascript、端末を打たない。Go のテストは go1.26 と `GOTOOLCHAIN=go1.25.8` の両方で通る

## Implementation outline

1. S1 regate の芯(`internal/org/regate.go`): 結びつきと gate の記録をロックの下で読む(D2)、worktree の確かめ(gate の関数を使う)、ロックと印(D7、D9 の regate の側)、fetch と HEAD の確かめ(D4、D8)、gate だけの道(AC9)と何もしない道(AC3、壊れた記録を含む)、記録の確かめ(D6)。テストは bare origin の fixture で AC1〜AC3、AC8、AC9
2. S2 Spawn との排他(`internal/org/spawn.go`、`internal/org/feature.go`): `withManifestLock` の中と start の手順 2 の確かめ(D9)。テストは AC13
3. S3 取り込みと push(`regate.go` に足す): merge(D3)、merge commit を作れないとき(AC5)、スクリプトの失敗と中断の巻き戻し(D5、AC6)、push と打ち直しと記録のあとの中断(AC7)。テストは AC4〜AC7
4. S4 CLI と merge-order: `ralph org regate`(出力、終了コード、シグナル)、merge-order の理由の欄(AC10)。テストは CLI と merge-order
5. S5 文書: skill の 8 面、仕様、tech-debt、`AGENTS.md`(AC11)

## Verify plan

- Static analysis checks: `./scripts/run-static-verify.sh`(gofmt、golangci-lint、shellcheck、secret scan)、`./scripts/check-skill-sync.sh`、`./scripts/check-sync.sh`
- Spec compliance criteria to confirm: FR-7 の「取り直してから出す」を、push する head の gate で満たすこと。FR-9 の「director は ralph と gh の操作に限る」を、regate が ralph の動詞であることで満たすこと。8a-3a に残すもの(auto_merge)を入れていないこと
- Documentation drift to check: skill の 4 面ずつの一致、CLI の help と skill の動詞リファレンスの一致、merge-order の理由の欄の文と skill の説明の一致、tech-debt の行の書き換え
- Evidence to capture: AC ごとのテスト名、偽でない git(ローカルの bare origin)で regate を打った出力

## Test plan

- Unit tests: `internal/org/regate_test.go`(AC1〜AC9。bare origin、結びつきの記録、偽の gate のスクリプト、止める test hook)、`internal/org/spawn_test.go` と `internal/org/feature_test.go`(AC13)、`internal/org/mergeorder_test.go`(AC10)
- Integration tests: `internal/cli/org_regate_test.go`(CLI の出力と終了コード、`--org-id` の拒否、台帳の決め方)
- Regression tests: `ralph org gate` の既存のテストがそのまま通ること(記録の形に `via` を足すのは regate の記録だけ)。merge-order の既存のテストのうち、走っている org の理由の欄は変わらないこと。Spawn と start の既存のテストがそのまま通ること(regate が走っていなければ共有のロックはすぐ取れて放す)
- Edge cases: org_id を別の機能で立て直して走っている、同じブランチの結びつきが 2 つある(後のものを使う)、`origin/<b>` が消えている、既定のブランチが HEAD の子孫(機能のコミットがない)、merge の途中で止まっている worktree、pre-merge-commit の hook が落ちる、`user.name` がなく merge commit を作れない、hook が入っていない repo、スクリプトが走る間に人が commit する(`head_moved`)かファイルが残る(`tree_changed`)、fetch と push の間に誰かが `origin/<b>` に push する、push の打ち直しの間に既定のブランチが進む、巻き戻しの `reset --keep` が落ちる、`via` を足した記録を今の `gateRecordFromDetails` が読める、push を origin が受け取ったあと regate が中断される(テストでは手で push してから打ち直す)、記録と push の間の中断のあとの打ち直し、regate が落ちて古い印が残っている、Spawn の共有と regate の排他が一瞬重なる
- Evidence to capture: `go test ./internal/org/ ./internal/cli/` の結果(go1.26 と go1.25.8)、mutant の赤

## Risks and mitigations

- regate は利用者の名前で merge commit を作り、push する。人の知らないうちにブランチが進まないよう、director は人に頼まれたときだけ打つ(8a-3a までは自分から打たない)。出力は作った commit と push 先を言う
- 巻き戻しの `git reset --keep` が、regate のあとに人が足したものを消すおそれ。D7 のロックと、記録の前の HEAD と作業ツリーの確かめ(`gateCheckAfter`)で、regate が作った commit が HEAD で作業ツリーがきれいなときだけ戻す(D5)。`head_moved` と `tree_changed` では戻さず、戻すコマンドを人に渡す
- fetch と push は利用者の資格情報で origin に出る。`GIT_TERMINAL_PROMPT=0` で入力を待たず、120 秒で区切る。ssh の鍵の入力は止めない(`core.sshCommand` を上書きしない)
- 衝突した PR は ralph では `ready` に戻らない。人が worktree で直して push し、regate を打ち直す(AC9 の道)か、org を立て直す。skill にそう書く
- org_id を同じ名前で立て直すと、古いブランチの regate はその org が走る間断られる。disband のあとに打つよう理由で言う。逆に regate の間は、同じ org_id の start と spawn が断られる(D9)。理由で regate が終わってから打ち直すよう言う
- regate が排他を取ってから印を空にするまでの一瞬に、前の regate が残した印の org_id の Spawn が断られうる。打ち直せば通るので、理由で打ち直しを言う
- Spawn の変更は、すべての spawn の `withManifestLock` の中に、待たない flock を 1 回足す。regate が走っていなければ取ってすぐ放すだけで、記録の形と判定は変わらない
- 既定のブランチの決め方は gate と同じく `refs/remotes/origin/HEAD` を読む。merge-order は ls-remote の symref を読むので、既定のブランチを変えた repo では食い違いうる(gate と同じ割り切り。tech-debt の既存の行)

## Rollout or rollback notes

- 新しい動詞と、merge-order の理由の欄の文と、Spawn の排他の確かめが増えるだけで、既存の記録の形と分類は変わらない。戻すときは revert する。regate が書いた記録(`via=regate` 付きの gate の記録)は、revert したあとの merge-order も読める
- release のノートに足すこと: `ralph org regate`、記録の `via=regate`、merge-order の理由の欄の変更、regate の間は同じ org_id の start と spawn が断られること。古い ralph の Spawn は regate.lock を見ないので、regate を使う前に全員の ralph を揃える

## Open questions

- なし(衝突の自動の解消は Non-goals)

## Progress checklist

- [x] Plan reviewed(consult-plan-automerge で 3 回。Codex の計画の助言 3 件は HIGH 1 件(D9)と MEDIUM 2 件(D5 の push の扱い、D2 の壊れた記録)で、すべて入れた)
- [x] Plan approved(ユーザーの「作成したプランは自動承認します」(2026-10-11)により自動承認)
- [x] Branch created
- [x] Implementation started
  - S1(7b6894ea): regate の芯。implementer の相談(consult-plan-automerge)で、排他を取った直後に印を空にする直しが入り、D9 の文を合わせて承認し直した(0e719e0a、sha256:7306f0d06797)。計画にない足し: fetch の env の `LC_ALL=C`(「couldn't find remote ref」を英語で読むため)、D6 の結びつきの位置の比較、test hook `afterRecord`、remote を叩く git の 1 回ごとの 120 秒の上限
  - S1 で残したもの: regate の gate のログの見出しが `# ralph org gate` のまま(`createGateLog` を共有)、共有の helper の誤りの接頭辞が `org: gate:` のまま、merge-order の JSON に `via` がない(S4 で扱う)
  - S2(458c7728): `regateSpawnErr` を `checkCapacityAndStart` の先頭と `StartFeature` の ensure の前に置いた。dry-run は新しい life を作らないので外した(consult で可)。範囲外: 古い ralph の台帳(disbanded の前に stopped がない)では、Reserve 付きの leader の spawn が冪等の分岐で `scope_reserved` を書き、この確かめを通らない。新しい座席は起動せず、regate の記録は `recordRegate` が断る。S5 で tech-debt に 1 行足す。stale な seat の補償は `spawn_failed` を書いてから Phase 2 で断られる
  - S3(5c7f585f): 取り込み、push、戻し方、push だけの打ち直し。計画からのずれ: `reset --keep` の手の条件に、最新の記録が `via=regate` であることを足した(leader の gate が通った head を手で落とさないため)。D4 の文を合わせて承認し直した(sha256:0d63d9df1288)。push だけの例外には `via` を求めない(leader が通した head はそのまま push する)。Outcome に `merged` と `push_only` を足し、push の失敗は誤りでなく `PushFailed` で返す
  - S4(2c3874e2、1ace199b): CLI の `ralph org regate`、merge-order の理由の欄(AC10)と JSON の `via`、共有の helper の接頭辞。1ace199b は、S4 の implementer が push 済みの 2c3874e2 を amend した中身を、reset --soft で新しいコミットとして積み直したもの(force push なし)。regate のログの見出し、`tree_changed` の退避の一言、push が落ちたあと push だけで済む e2e のテストを含む
  - S4b(7602b00d、d8675039、8222d6bd): merge-order の help と台帳の注、org_id を別の機能で立て直して走っている org のブランチの Detail と regate の断りの文(AC10 の文を「今結びついているブランチ」に直し、承認し直した。sha256:f841204a7bc0)、`regate.go` から merge と push を `regate_merge.go` へ移した(中身は変えない)。`regate.go` は 851 行で、gate だけの道と共有する関数まで動かさないと 800 行を切らないので止めた
  - 範囲外(この PR では直さない): `internal/org/director_cmd_test.go:300-302` が sh の trap を 100 ms の sleep で待つので、負荷が高いと go1.25.8 の全体の実行で 1 回落ちた(単独では 5 回通る)。S5 で tech-debt に 1 行足す
- [ ] Review artifact created
- [ ] Verification artifact created
- [ ] Test artifact created
- [ ] PR created
