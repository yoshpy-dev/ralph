# org-merge-cleanup

- Status: Approved
- Approved: 2026-10-28 sha256:0f38a9718a8b
- Owner: Claude Code
- Date: 2026-10-28
- Related request: https://github.com/yoshpy-dev/ralph/issues/263(org: 機能の worktree を merge のあとに自動で片付ける)
- Related issue: 263
- Type: feat
- Branch: feat/org-merge-cleanup

## Objective

機能の PR が merge されたら、その機能の worktree、ローカルのブランチ、`ralph-worktree.sh` の記録を ralph が片付ける。`ralph org merge` で merge したときは、merge の続きで消す。人が GitHub で直接 merge したときは、新しい動詞 `ralph org cleanup` で消す。どちらも、消すと失うものがある場合と、まだ使われている場合は消さず、理由と手で消すときのコマンドを出す。

## Scope

- org 層(`internal/org/`)に、merge 済みの機能の worktree を片付ける処理を足す。確かめてから `scripts/ralph-worktree.sh cleanup --id org-<org_id> --force-branch` を打ち、台帳に `worktree_cleaned` を書く
- `ralph org merge`: `pr_merged` を記録したあと、`regate.lock` を持ったまま上の処理を呼ぶ。結果は stdout の `CLEANUP:` の行で出し、終了コードは変えない(exit 0 は記録した merge のまま)
- 新しい動詞 `ralph org cleanup [--branch <ブランチ>]`: `ralph org merge-order` が `merged` と分類するブランチ(`--all` と同じ読み方)のうち、機能の worktree の記録が残っているものを片付ける
- leader の雛形の手順 15、`/org` skill と `/director` skill の後始末の節、`ralph org start --help`、regate の断りの文言、recipe、仕様、AGENTS.md の地図を新しい動きに合わせる
- それぞれのテスト

## Non-goals

- `scripts/ralph-worktree.sh` の `cleanup` 自体の変更(今の `--force-branch` を使う)
- PR が閉じられた(merge されていない)ブランチの片付け。今までどおり人が `ralph-worktree.sh cleanup` で消す
- リモートのブランチの削除(`gh pr merge --delete-branch` は今も付けない)
- `disband` での片付け(merge の前は regate がこの worktree を使う)
- 標準フローの task worktree(`/pr` が片付ける。変えない)
- gh のない環境で merge 済みを判定する道(gh がなければ `ralph org cleanup` は判定できないと言って断る)

## Assumptions

- 機能の worktree は `ralph org start --plan` が `ralph-worktree.sh ensure --id org-<org_id> --kind org --cleanup-policy manual` で作る(`internal/org/feature.go:139-198`、`:228-231`)。worktree の絶対パスは台帳の binding(`scope_reserved` の `Worktree`)にもある(`internal/org/reserve.go:80`、`:386-388`。regate も binding から読む、`internal/org/regate.go:486`)
- `ralph org merge` は、結びつきの org が走っていないときだけ merge する(`internal/org/merge.go:500` の `runningRefusal`)。`regate.lock` を持ち(`internal/org/merge_run.go:188-198`)、`pr_merged` は `recordMerge`(`:493-505`)が manifest のロックの下で書く
- `ralph org merge-order` の `merged` は、gh で読んだ PR の state が MERGED のときだけ付く(`internal/org/mergeorder.go:988-991`)。`--all` がないと出ない(`:945`)
- `ralph-worktree.sh cleanup`(`scripts/ralph-worktree.sh:373-405`)は、worktree に未コミットの変更か未追跡のファイルがあれば何も変えずに exit 1 で止まる。`git worktree remove` は `--force` なしで打つ。ブランチは `--force-branch` で `git branch -D`、なしで `-d`。最後に記録を消す。gitignore されたファイルは止める理由にならず、worktree と一緒に消える
- squash merge のあと、リモートのブランチが消えていると `git branch -d` は「not fully merged」で止まり、worktree だけが消えて、ブランチと記録が残る(Explore が使い捨ての repo で確かめた)。そのため `--force-branch` を使う。その前に、ローカルのブランチの先頭が merge された PR の head と一致することを確かめる(push していないコミットを失わないため)

## Affected areas

- `internal/org/feature.go`(`FeatureWorktrees` に `Cleanup` を足す、実装 `scriptFeatureWorktrees`、`featureWorktreeCleanup` の案内)と、その fake(`internal/org/feature_test.go` の `fakeWorktrees`、`spyWorktrees`)
- 新しいファイル `internal/org/cleanup.go`(片付けの確かめと実行、`worktree_cleaned`)と `internal/org/cleanup_test.go`
- `internal/org/seat.go`(イベントの定数 `EventWorktreeCleaned`)
- `internal/org/merge.go`、`internal/org/merge_run.go`(`MergeParams` に片付けの opt-in、`MergeResult` に結果、`regate.lock` の marker に org_id)とそのテスト
- `internal/org/regate_git.go`(`regateBoundedGit` を片付けの確かめにも使う)、`internal/org/lockfile.go:143-149`・`internal/org/regate.go:91-98`・`internal/org/spawn.go:1456-1461`(marker の注釈)
- `internal/cli/org_merge.go`(`CLEANUP:` の行、repo root の受け渡し)とそのテスト
- 新しいファイル `internal/cli/org_cleanup.go`(動詞 `ralph org cleanup`)と `internal/cli/org_cleanup_test.go`、`internal/cli/org.go`(登録、`ralph org start --help` の文言)
- `internal/org/regate.go:553-556`(worktree がないときの案内)と `internal/org/regate_test.go:447`
- `internal/org/prompts/leader.md`(手順 15)と `internal/org/prompts_test.go:530`、`:584`
- `.claude/skills/org/SKILL.md`(動詞の表 `:204-226`、start の節 `:643-646`・`:723-727`・`:770-773`、「org の終わり方」`:1094-1106`)と 3 つの写し
- `.claude/skills/director/SKILL.md`(「merge と auto_merge」`:813-901`、「進捗の報告」`:788-811`)と 3 つの写し、`internal/org/director_skill_sync_test.go`
- `docs/recipes/isolated-herdr-server.md:529` と template の写し
- `docs/specs/2026-10-07-org-multi-org-director.md`(FR-8 に決めたことの注)、`AGENTS.md`(`internal/org/` の地図)

## Visual review

- Page: `.harness/state/plan-visual/org-merge-cleanup.html`。かんたん版の図 1 枚(`overview`)、「承認の前に」の箱、折りたたみの中の詳細 2 枚(`detail-overview` 変わるファイル、`detail-checks` 消す前の確かめの順)
- Lint: `./scripts/plan-visual.sh lint` は exit 0
- Self-check: ページ全体と 3 枚の図を `shot` で撮って読んだ。はみ出しと重なりはない

## Design decisions

- D1 片付けの本体は org 層の 1 つの関数にまとめ、`ralph org merge` と `ralph org cleanup` の両方から呼ぶ。消す操作は `ralph-worktree.sh cleanup --id org-<org_id> --force-branch` に任せる(記録の場所と worktree の消し方を script と二重に持たない)。Go の側は、消す前の確かめと台帳の記録だけを持つ
- D2 消す前の確かめ。どれかが通らなければ消さず、理由と次の手を返す。理由を直して片付けを打ち直せば続く断り(3・4・5・6)では、次の手は打ち直しだけで、手で消すコマンドは返さない。手で消すコマンド(`--force-branch` の script や `git branch -D`)は、ralph が先に進めず人の判断が要るとき(記録がない、記録が binding と合わない、ブランチがなく worktree だけがある、消す途中で失敗した)だけ返す。確かめ 5 と 6 より前の断りで `--force-branch` の script を渡すと、push していないコミットのあるブランチまで消えるため(self-review の M2)
  0. `regate.lock` を持ち、その marker に片付ける org の org_id を書く(merge は持ったまま呼ぶ。`ralph org cleanup` は merge と同じく取る。取れなければ regate か merge が走っているので断る)。merge は今 marker を空にするだけで(`internal/org/merge_run.go:203`)、`regateSpawnErr`(`internal/org/spawn.go:1464-1479`)は marker が空なら spawn と start を通すので、marker を書かないと、片付けの最中に同じ org_id の `ralph org start --plan` が worktree を作り直しにかかる(`internal/org/feature.go:216`)。「merge は worktree を変えないので marker を書かない」という注釈 3 か所(`internal/org/lockfile.go:143-149`、`internal/org/regate.go:91-98`、`internal/org/spawn.go:1456-1461`)も直す
  1. 台帳にそのブランチの binding があり、その org が走っていない(`RunningOrgs`)
  2. `ralph-worktree.sh` の記録 `org-<org_id>` があり、その branch と worktree_path と kind(`org`)が binding と一致する(同じ org_id を別の機能で立て直したとき、別の機能の worktree を消さないため)
  3. ralph を打ったディレクトリがその worktree の中にない(消す worktree の中に、打った人の shell を残さないため。ralph は script を main worktree で打つので、script を中から打ったときの動きは関わらない)
  4. worktree に未コミットの変更も未追跡のファイルもない(`git -C <worktree> status --porcelain` が空)。rebase や merge の途中でない
  5. ローカルのブランチの先頭が、merge された PR の head と一致する(`ralph org merge` では gate の head、`ralph org cleanup` では gh の `headRefOid`)。一致しなければ、push していないコミットがありうるので消さない
  6. worktree が名前のブランチ(記録の branch)にいて(`git -C <worktree> symbolic-ref --short HEAD`)、その HEAD も merge された head と一致する。detached HEAD や別のブランチに切り替えた worktree では、そこで足したコミットへの参照が worktree と一緒に消えるので、消さない(Codex の計画助言 1)

  再開(Codex の計画助言 2): script は worktree、ブランチ、記録の順に消し、途中で止まると前の段だけが消えた状態が残る。もう一度片付けを打ったときに、残った状態ごとに次のように続ける。確かめは、残っているものにだけかける
  - worktree のディレクトリがなく、ブランチと記録が残る(ブランチの削除で止まった): 3・4・6 は worktree がないのでかけず、1・2・5 だけをかけて script を打つ。git に worktree の登録だけが残っているときは、その 1 つだけを外す(`git worktree prune` はほかの worktree の登録にも効くので使わない。外し方は実装で確かめる)
  - worktree もブランチもなく、記録だけが残る: 1・2 だけをかけて script を打つ(記録を消すだけになる)
  - 記録がない(人が `ralph-worktree.sh gc --prune` などで消した): 消すものを特定できないので片付けず、残ったブランチを手で消すコマンド(ブランチの先頭を確かめてから `git branch -D`)を出す
  どの段で止まっても、もう一度の `ralph org merge` の続きか `ralph org cleanup` で片付くことをテストで確かめる。断りの文で `gc --prune` は勧めない

  確かめの git(`status --porcelain`、`rev-parse`)は、regate と同じ時間の上限つきの git(`regateBoundedGit`、`internal/org/regate_git.go:173`)で打ち、`MergeParams.Exec` は使わない。merge のテストの「打つコマンドはすべて `/repo` で origin に固定」という検査(`internal/org/merge_fake_test.go:464`)と分けるため
- D3 `ralph org merge` は、片付けが断られても失敗しても終了コードを変えない(exit 0 は記録した merge、`internal/cli/org_merge.go:236-238`)。stdout に `CLEANUP: done` か `CLEANUP: skipped (<理由>)` の 1 行を足し、skipped のときは、手で消すコマンドがあればそれを `NOTE:` と同じ形で出す(D2 のとおり、打ち直せば続く断りにはない)。skipped は片付けが終わっていないことで、断り(何も消していない)と、消す途中の失敗の両方を含む。失敗は、script が途中で止まったもの(消した分は消えたまま)か、script の前に、なくなった worktree の git の登録を外せなかったもの(何も消していない)で、どちらも次の片付けが続きから進む。人と director の次の手はどちらも同じなので、行の形は分けない(self-review の M1)。片付けは `MergeParams` で opt-in にし(repo root が空なら呼ばない)、CLI だけが渡す(org 層の今のテストは script のない `/repo` を使うため)。CLI の repo root は、`ralph org start --plan` と同じく `org.FeatureRepoRoot`(`internal/cli/org.go:1087`、`internal/org/statedir.go:128`)で求める
- D4 動詞の名前は `ralph org cleanup`。`--branch` を付けるとそのブランチだけ、付けなければ merge 済みで記録の残るすべてのブランチを片付ける。1 本ごとに `CLEANED: <ブランチ> org-<org_id>` か `SKIPPED: <ブランチ> <理由>` の行を出す。終了コードは、すべて片付いたか片付けるものがなければ 0、1 本でも残れば 1。gh がないか PR を読めないときは、merge 済みを判定できないと言って 1 で終わる。`--branch` のブランチが `merged` でなければ断る。binding がなく gate の記録だけのブランチは対象にしない(機能の worktree は `ralph org start --plan` の org にしかなく、binding がなければ消すものを特定できない)。`--branch` なしでは、片付け済みで何も残っていないブランチ(記録も worktree もローカルのブランチもない)と、同じ org_id がのちに別のブランチに結びつけ直された古い binding は、数えず行も出さない。前者は org 層の結果 `NothingLeft` で、後者は merge-order の binding の `BoundTo`(org の最新の binding が別のブランチのとき、そのブランチ)で見分ける。結びつけ直しが通った時点で古い記録と worktree はもうないので(`checkFeatureWorktreeReuse`、`internal/org/feature.go:225`)、残りうる古いローカルのブランチは `--branch` で名指ししたときだけ扱う(S3 の consult による)
- D5 片付けたら、台帳に org の記録 `worktree_cleaned`(`branch=`、`worktree=`、`head=`、`via=merge|cleanup`)を書く。`ralph org report` と人が台帳で後始末を追えるようにするため。`RunningOrgs` と Roster の状態には入らない(`pr_merged` と同じ扱い)
- D6 gitignore されたファイル(feature worktree の `.harness/state/` や `docs/evidence/` の log)は、止める理由にしない。script と `git worktree remove` が今もそう扱い、merge のあとの worktree の中の生成物だから。リスクの表に書き、skill の後始末の節にも 1 文書く

Critical forks: なし。issue が方針(merge の続きで消す、直接 merge には新しい動詞、未コミットと未追跡では消さない、走っている org と regate・merge の最中は消さない)を決めている。動詞の名前は issue が計画に任せた命名で、D4 に理由を書いた。ブランチを `-D` で消す前の head の照合(D2 の 5)は、issue の「PR が MERGED であることを根拠に消す」を、push していないコミットを失わない形で満たすための既定である。

## Acceptance criteria

- [ ] AC1 `ralph org merge` で merge したあと、その機能の worktree・ローカルのブランチ・`ralph-worktree.sh` の記録が消え、台帳に `worktree_cleaned` があり、stdout に `CLEANUP: done` が出る。squash merge でリモートのブランチが消えたあとも止まらない(issue の受け入れ条件 1)
- [ ] AC2 GitHub で直接 merge したブランチを `ralph org cleanup` で片付けられる。`--branch` なしなら merge 済みで記録の残るすべてのブランチ、`--branch` ならそのブランチだけ(issue の受け入れ条件 2)
- [ ] AC3 未コミットの変更か未追跡のファイルがあるとき、ローカルのブランチの先頭が merge された head と違うとき、worktree が detached HEAD か別のブランチにいるとき、記録が binding と一致しないとき、ralph をその worktree の中から打ったときは、何も消さずに理由と次の手(打ち直し、または D2 の場合だけ手で消すコマンド)を出す。`ralph org merge` の終了コードは 0 のまま(issue の受け入れ条件 3)
- [ ] AC3b 片付けが途中で止まった状態(worktree だけが消えた、記録だけが残った)から、もう一度の `ralph org cleanup` で残りが片付く。記録がないときは何も消さず、手で消すコマンドを出す
- [ ] AC4 org が走っているブランチと、regate か merge が走っているとき(`regate.lock` が取れない)は消さない(issue の受け入れ条件 4)
- [ ] AC5 `/org` と `/director` の skill の後始末の節、leader の雛形の手順 15、`ralph org start --help`、regate の案内が新しい動きに合い、写しが一致する(issue の受け入れ条件 5)
- [ ] AC6 新しいテストと既存のテストが通る(`internal/org`、`internal/cli` の該当するテスト、`scripts/check-skill-sync.sh`、`scripts/check-sync.sh`)

## Implementation outline

1. S1 org 層の片付け: `FeatureWorktrees.Cleanup`(script の `cleanup --id <id> --force-branch`、fake 2 つも)、`internal/org/cleanup.go` の確かめ(D2、git は `regateBoundedGit`)と実行と `worktree_cleaned`(D5)、`EventWorktreeCleaned`。`regate.lock` の marker に org_id を書く手続き(D2 の 0)。テストは fake の worktrees と、`worktreeScriptRepo` の実 script の repo で、片付く道と D2 の断る道を 1 つずつ。片付けの最中に同じ org_id の spawn と start が `regateSpawnErr` で断られること — verify: `test -z "$(gofmt -l internal/org)" && go vet ./internal/org/ && GOFLAGS="-timeout=30m $(go env GOFLAGS)" go test ./internal/org/ -run 'Cleanup|FeatureWorktree|StartFeature' -count=1`
2. S2 `ralph org merge` から呼ぶ: `MergeParams` に repo root(opt-in)、`MergeResult` に片付けの結果、`pr_merged` のあと `regate.lock` を持ったまま、marker に org_id を書いてから呼ぶ。marker の注釈 3 か所を直す。CLI は `org.FeatureRepoRoot` で repo root を求め、`CLEANUP:` の行を出す。テストは org の merge のテスト(片付けを呼ばない既定と、呼んで done・skipped になる道)と、CLI の merge のテスト(実 script と bare origin で squash merge のあとに消えること) — verify: `test -z "$(gofmt -l internal/org internal/cli)" && go vet ./internal/org/ ./internal/cli/ && GOFLAGS="-timeout=30m $(go env GOFLAGS)" go test ./internal/org/ -run 'Merge|Cleanup' -count=1 && GOFLAGS="-timeout=30m $(go env GOFLAGS)" go test ./internal/cli/ -run 'OrgMerge' -count=1`
3. S3 動詞 `ralph org cleanup`: `internal/cli/org_cleanup.go`(merge-order を `--all` と同じに読み、`merged` のブランチごとに S1 を呼ぶ)、`internal/cli/org.go` に登録。テストは片付く道、gh がないとき、`--branch` が merged でないとき、1 本が残るとき — verify: `test -z "$(gofmt -l internal/cli)" && go vet ./internal/cli/ && GOFLAGS="-timeout=30m $(go env GOFLAGS)" go test ./internal/cli/ -run 'OrgCleanup|OrgMergeOrder' -count=1`
4. S4 雛形と skill と案内: leader の手順 15、`/org` skill(動詞の表、start の節、「org の終わり方」)、`/director` skill(「merge と auto_merge」と「進捗の報告」で、`CLEANUP: skipped` を人に伝え、merge-order の `--all` に `merged` があれば `ralph org cleanup` を打つ)、`ralph org start --help`、regate の案内、`featureWorktreeCleanup`。4 つの写しは `scripts/sync-skills.sh` で作る。固定しているテストを直し、新しい文を固定する — verify: `test -z "$(gofmt -l internal/org internal/cli)" && go vet ./internal/org/ ./internal/cli/ && GOFLAGS="-timeout=30m $(go env GOFLAGS)" go test ./internal/org/ -run 'TestRenderRolePrompt|TestRolePrompts|OrgSkill|DirectorSkill|Regate' -count=1 && GOFLAGS="-timeout=30m $(go env GOFLAGS)" go test ./internal/cli/ -run 'OrgStart.*Help|OrgHelp' -count=1 && ./scripts/check-skill-sync.sh && ./scripts/check-sync.sh`
5. S5 文書: 仕様 `docs/specs/2026-10-07-org-multi-org-director.md` の FR-8 に決めたことの注、`AGENTS.md` の `internal/org/` の地図、`docs/recipes/isolated-herdr-server.md` と template の写し — verify: `./scripts/check-sync.sh && git diff --check`

## Verify plan

- Static analysis checks: `./scripts/run-static-verify.sh`(gofmt、go vet、golangci-lint、staticcheck、shellcheck、同期の検査、secret scan)
- Spec compliance criteria to confirm: AC1〜AC6、issue #263 の受け入れ条件 5 つ、D1〜D6。`ralph org merge` の「exit 0 は記録した merge」が変わっていないこと。`--force-branch` の前に head の照合があること
- Documentation drift to check: `cleanup --id org-` と `cleanup-policy manual` を grep し、手で消す案内が「PR が閉じられたとき」と「片付けが断られたとき」だけに残ること。`/org` と `/director` の skill の 4 つの写し、AGENTS.md の地図、仕様の FR-8
- Evidence to capture: `docs/evidence/verify-2026-10-28-org-merge-cleanup.log`

## Test plan

- Unit tests:
  - S1 の確かめを 1 つずつ。binding がない、org が走っている、記録がない、記録の branch・path・kind が違う、worktree の中から打った、未コミット、未追跡、rebase の途中、ブランチの先頭が違う、worktree が detached HEAD でコミットを足した、worktree が別のブランチにいる。どれも何も消さず、理由と手のコマンドを返すこと
  - 再開(D2 の再開): worktree だけが消えてブランチと記録が残る、記録だけが残る、の 2 つの状態から片付くこと。記録がないときは何も消さないこと
  - 片付く道では、script の `cleanup --id org-<id> --force-branch` を 1 回だけ打ち、`worktree_cleaned` を 1 行書くこと
- Integration tests:
  - `worktreeScriptRepo` の実 script と bare origin で、機能のブランチを squash merge し、リモートのブランチを消したあとに片付くこと(`git branch -d` では止まる場面)。worktree・ブランチ・記録が消えること
  - CLI の `ralph org merge`(fake gh と実 git)で `CLEANUP: done` が出て exit 0、未追跡のファイルを置くと `CLEANUP: skipped` で exit 0 のまま、何も消えないこと
  - `ralph org cleanup` の片付く道、gh がない道、`--branch` が merged でない道
- Regression tests: org の merge のテストすべて(片付けの opt-in を渡さない既定では、今までどおりのコマンドだけを打つ)、`mrFixture` の `-d`・`--delete-branch` の検査、regate と spawn の `regate.lock` のテスト、`TestStartFeature`
- Edge cases: 同じ org_id を別の機能で立て直した記録、記録はあるが worktree のディレクトリがない、gitignore されたファイルだけがある worktree(消える。D6)、`regate.lock` を別のプロセスが持っている
- Evidence to capture: `docs/evidence/test-2026-10-28-org-merge-cleanup.log`

## Risks and mitigations

| Risk | Mitigation |
| --- | --- |
| `--force-branch` で、push していないコミットのあるブランチを消す | D2 の 5: ローカルのブランチの先頭が merge された PR の head と一致するときだけ消す。違えば理由と手のコマンドを出す |
| 同じ org_id を別の機能で立て直したとき、新しい機能の worktree を消す | D2 の 2: 記録の branch・path・kind を binding と照らし、違えば消さない |
| merge のあとの片付けの失敗で、merge が失敗に見える | D3: 終了コードは変えず、`CLEANUP:` の行で伝える。片付けは台帳の `pr_merged` のあとに呼ぶ |
| gitignore された生成物(evidence の log など)が worktree と一緒に消える | D6: 今の script と同じ扱い。skill の後始末の節に 1 文書く。報告はコミットされている |
| 片付けの途中で止まる(worktree は消えたが、ブランチと記録が残る) | script の順(worktree、ブランチ、記録)のまま。D2 の再開のとおり、残った状態ごとに残っているものにだけ確かめをかけ、もう一度の `ralph org cleanup` で片付ける。`gc --prune` は勧めない(記録だけを消し、残ったブランチが片付けの対象から外れるため) |
| detached HEAD か別のブランチで足したコミットを、worktree と一緒に失う | D2 の 6: worktree が記録のブランチにいて、その HEAD が merge された head と一致するときだけ消す |
| 古い binary の leader の雛形は「人が消す」のまま | 雛形は spawn のたびに展開される。merge のあとの片付けは merge の側が行うので、leader の文が古くても結果は同じ |

## Rollout or rollback notes

- 新しい binary の `ralph org merge` から効く。古い binary で merge したブランチは `ralph org cleanup` で片付けられる
- 戻すときはこの PR を revert する。台帳に増えるのは `worktree_cleaned` の行だけで、古い binary はそれを読まずに無視する(読み手は未知のイベントを許す、`internal/org/seat.go:30-37`)
- release ノート: `ralph org merge` が merge のあとに機能の worktree とブランチを消すようになったこと、`ralph org cleanup` が足されたこと

## Open questions

- なし(動詞の名前と終了コードは D4 で決めた)

## Progress checklist

- [x] Plan reviewed(consult の判定は「進めてよい」。足したのは 3 点で、`regate.lock` の marker に org_id を書くこと(D2 の 0)、確かめの git を `regateBoundedGit` で打つこと、CLI の repo root を `org.FeatureRepoRoot` で求めること。Codex の計画助言は 2 件で、どちらも反映した。HIGH は worktree の HEAD とブランチの確かめ(D2 の 6、AC3)、MEDIUM は途中で止まったあとの再開(D2 の再開、AC3b、リスクの表)。図解の詳細 2 も直し、lint と撮り直しを確かめた)
- [x] Plan approved(自動承認。2026-10-11 のユーザー指示「作成したプランは自動承認します」と、2026-10-27 の「最後まで止まらずに」による)
- [x] Branch created
- [x] Implementation started(スライスは implementer に委譲)
  - S1 e49cddb5: `internal/org/cleanup.go`。入口は 2 つで、`CleanupFeatureWorktree` は `regate.lock` を自分で取り(S3 用)、`cleanupFeatureWorktreeLocked` は lock を持った呼び手が使う(S2 用)。ほかに `FeatureWorktrees.Cleanup`(script の `--force-branch`)と `EventWorktreeCleaned` を足した
    - 確かめ 1 を manifest のロックの下で通したときに、marker に org_id を書く
    - 結果は `Refused`(確かめで止まった)・`Failed`(削除の失敗)・`Manual`(どこからでも打てる 1 行)で返す
    - 登録だけ残った worktree は、そのパスだけに `git worktree remove` を打って外す(prune は使わない)
    - `worktree_cleaned` の詳細は `branch= head= via= worktree=`。パスは quote して最後に置く
    - 検証: 指定のテストが通過(76 秒)、regate・merge・spawn の回帰も通過、変異 5 つが落ちる。機械的な確かめは 5 項目とも clean
    - consult は、判断 7 点を妥当とした
  - S2 に渡すこと: marker の注釈 3 か所と `regateSpawnErr` の断りの文(片付けの最中も「regate が走っている」と言う)。`Org.Worktrees` の注釈。`Refused` と `Failed` はどちらも `CLEANUP: skipped` にする
  - S4 に渡すこと: `featureWorktreeCleanup` に `--force-branch` を足すなら、`cleanupRun.scriptCmd` で二重にならないようにする
  - 既知の穴: `StartFeature` の `regateSpawnErr` は manifest のロックの外で打たれる(`internal/org/feature.go:208-218`)。Regate と同じ窓で、この計画では直さない
  - S2 b8209a37: `ralph org merge` は `pr_merged` を記録したあと、`regate.lock` を持ったまま `cleanupFeatureWorktreeLocked` を打つ。`MergeParams.RepoRoot` が空なら片付けない
    - 結果は `MergeResult.Cleanup`(nil は呼んでいない)と `CleanupErr` に入る。merge のエラーは片付けでは立たない
    - 記録のあと片付けの前に割り込まれたら、片付けは打たずに `Refused` に `ralph org cleanup --branch` の打ち方を入れ、exit 1 で終わる
    - CLI の行は `CLEANUP: done`、`CLEANUP: done (resumed: <kind>[; <err>])`、`CLEANUP: skipped (<理由>)`、続けて `CLEANUP MANUAL: <コマンド>`。repo root は `org.FeatureRepoRoot` で求め、見つからなければ skipped で理由を出す
    - spawn の断りの文は「regate が走っているか、merge か cleanup が片付けている」に直した。`ralph org merge --help` に片付けの段落を足した
    - 検証: 指定のコマンドが通過(internal/org 229 秒、internal/cli 70 秒)。断りの文を固定するテストの組は 1 回目に 30 分の timeout に当たり、回し直しで通過(552 秒、原因は未確認)。変異 9 つが落ちる。機械的な確かめは 5 項目とも clean
  - S4 に渡すこと(S2 の報告から):
    - `errWorktreeScriptMissing`(`internal/org/feature.go:622`)の「run start again」が片付けの断りに混ざる。文を直し、`internal/cli/org_merge_test.go` の `noScriptCleanup()` の固定は `CLEANUP: skipped (scripts/ralph-worktree.sh cannot be run in ` までにして、`CLEANUP MANUAL:` が出ないことは別に確かめる(consult の勧め)
    - `regateLockWait` の注釈が「Regate and Merge retry」のまま
    - `ralph org --help` の merge の説明に片付けがない
    - `/director` skill に「`CLEANUP: done` で始まれば片付いた」と読む 1 文(`resumed: record_only; <err>` は台帳の追記だけが失敗した形)
  - S3 の前に D4 へ 1 文足した(計画の本文の変更): 片付け済みで何も残らないブランチ(`NothingLeft`)と、結びつけ直された古い binding(`BoundTo`)は、`--branch` なしでは数えない。片付け済みのブランチも merge-order の `merged` に残り続け、S1 はそれを `Refused` で返すため、そのままでは `--branch` なしの動詞が毎回 exit 1 になる。consult の勧めで、結びつけ直しの判定は CLI でなく `collectMergeTargets` に置く(同じ秒の 2 つの binding で順序が決まらないのを避ける)。図解は「merge-order の merged のブランチごと」としか描いていないので直す図はなく、digest を取り直して自動承認した(2026-10-28 sha256:25aae0c272fe)
  - S3 87b6e72d: 動詞 `ralph org cleanup [--branch]`(`internal/cli/org_cleanup.go`)。merge-order を `--all` で読み、対象を選び(`cleanupSteps`、`--branch` は `cleanupNamedStep`)、1 本ずつ `CleanupFeatureWorktree` を打つ(`runCleanupSteps`)。行は `CLEANED:`、`SKIPPED:`、`MANUAL:`、`nothing to clean up`
    - org 層に `CleanupResult.NothingLeft`(`recordGone` の何も残らない道)と `MergeOrderBinding.BoundTo`(`collectMergeTargets`)を足した
    - `--org-id` の説明の一覧に `cleanup` を足した。approve と director の help のテストは、固定している部分にこの箇所が入らないので変えていない
    - `TestMergeOrder_DetailsOfABranchWhoseOrgRunsAnotherFeature` はエントリの JSON 全体を比べていたので、BoundTo の値を先に確かめてから空にして比べる形に直した(比べること自体は残した)
    - 検証: 指定のコマンド(計画の S3 のコマンドを org 層の `Cleanup|MergeOrder` と cli の `OrgMerge_|Help` まで広げたもの)が 2 回目で通過(1 回目は上の JSON のテストで落ちた。org 76 秒、cli 133 秒)。変異 6 つが落ちる。機械的な確かめは 5 項目とも clean。consult は判断 8 点を妥当とした
  - S4・S5 に渡すこと(S3 の報告から): `ralph org merge-order --help` は `--json` の `pr_lookup` を説明しているが `bound_to` に触れていない。AGENTS.md の地図と `/org` skill に `ralph org cleanup` がまだない
  - S4 cb7ed292: leader の手順 15、`/org` skill(動詞の表、start の節、merge してよい順、「org の終わり方」、完了条件)、`/director` skill(`CLEANUP:` の読み方、「GitHub で merge した PR の後始末」の小節、「してはいけないこと」に MANUAL のコマンドを頼まれずに打たない)、start・spawn・merge-order の help、regate の案内(merge 済みなら `ralph org cleanup --branch`、閉じた PR なら script の cleanup に `--force-branch`)
    - `featureWorktreeCleanup` は変えない(start の失敗の案内では `-d` が merge されていないコミットを守る)。regate の案内は呼ぶ側で `--force-branch` を足した。Affected areas の「`featureWorktreeCleanup` の案内」はこの判断で満たす
    - 片付けの script がないときの断りから `errWorktreeScriptMissing` の「run start again」を外した。`noScriptCleanup()` は前置きと「MANUAL の行がない」を確かめる `assertNoScriptCleanup` にした
    - 検証: 指定のコマンド(計画の S4 のコマンドを `Cleanup`、cli の `Help|OrgMerge|OrgCleanup` まで広げたもの)が通過(744 秒)。新しいテスト 4 つは壊した版 4 つで落ちる。機械的な確かめは 5 項目とも clean。consult は判断 8 点を妥当とした
  - S4b 111c62a2(consult の勧めで足したスライス): regate と merge の `regate.lock` の断り、その help と注釈に cleanup を足した(「another ralph org regate, merge, or cleanup runs」)。`internal/org/feature.go:538-539` は変えていない。同じ org_id の別の start の断りが片付けのコマンドを案内しないことを `feature_test.go` が意図して固定しているため。検証: `Regate|Merge|StartFeature|Lock` と cli の `OrgRegate|OrgMerge` が 2 回とも通過。機械的な確かめは clean(折り返しの直しを 3 回 fixed)
  - S5 に渡すこと: README.md の 252 行(merge のあとも人が script で消す)と 266・268 行(「regate or merge」)、`docs/recipes/isolated-herdr-server.md:529` と template の写し
  - 既知の穴(PR に書く): `runWorktreeScript` は script を時間の上限なしで打つ(S1 から。`git worktree remove` と `git branch -D` はローカルの操作)
  - S5 c0383ffe と 7d3e3f9f: 仕様の FR-8 に「merge のあとの後始末(#263)で決めたこと」の注(4 段目と 8a-3a の注がどう変わったかも 1 文ずつ)、AGENTS.md の地図に `cleanup.go`、README の 252・266・268 行と `ralph org cleanup` の段落(268 行の「spawn と start を止めない」も、片付けの間だけ止めると直した)、herdr の recipe とその写し。7d3e3f9f は consult の遅れた返事で、仕様の注の閉じた PR の案内を「ブランチが要らないと確かめたら `--force-branch`」の形にした(README と skill に合わせた。recipe は clone ごと捨てる文脈なので無条件のまま)
    - 検証: `./scripts/check-sync.sh && git diff --check` が両方のコミットで通過、recipe の 2 つは byte で同じ。機械的な確かめは 5 項目とも clean
    - `/org` skill に残る `cleanup --id org-` の 727・729・777 行は start の失敗の案内、1143 行は閉じた PR の案内で、直すものはない
  - self-review cycle 1(97ea508b): CRITICAL 0、HIGH 0、MEDIUM 2、LOW 6。consult の勧めで全部を直すか台帳に書く
    - 計画の本文を直した(digest を取り直して自動承認、2026-10-28 sha256:810337822bb7): D2 の頭に「打ち直せば続く断り(3・4・5・6)には手で消すコマンドを返さない」(M2)、D2 の 3 の理由を「消す worktree の中に人の shell を残さないため」に(report の Follow-ups)、D3 に「skipped は断りと script の途中の失敗の両方を含み、行は分けない」(M1)、AC3 の「手で消すコマンド」を「次の手」に。図解は overview と詳細 2 の文を「理由と次の手」に直し、lint 0、2 枚を撮って読んだ(はみ出しなし)
    - F3(inline、この記録と同じコミット): `docs/tech-debt/README.md` に 2 行(L5 の 800 行を越えたテストのファイル 4 つ、L6 の既知の穴 2 件)。機械的な確かめは 5 項目とも clean
    - F1 に渡すこと: M2(確かめ 3・4・5・6 の断りから Manual を外す。5 は「要るものを別のブランチに置き、ブランチを head に戻してから打ち直す」の文に)、L1(`cleanupFirstCheck` で org_id を `ValidateIdentifier` に照らす)、L2(断りの文の中のコマンドのパスを quote)
    - F2 に渡すこと: M1(`/director` skill の「skipped なら何も消していない」を直す)、L3(main worktree が見つからないときの文)、L4(`/director` skill の手順 1 と 3 のぶつかり)、仕様 FR-8 の注の確かめ 3 の理由
    - F1 a3af6114(M2・L1・L2): 確かめ 3・4・5・6 の断りから Manual を外した。確かめ 5 の文は場面ごとに戻し方を名指す(worktree がブランチにいれば `reset --hard`、別のブランチか detached なら `branch -f`、worktree_gone なら `branch -f`、登録だけ残るなら `update-ref`。git 2.49.0 で `branch -f` が登録の残る worktree のブランチを断ることを確かめた)。Manual を残すのは記録なし・記録違い・ブランチがなく worktree だけ・Failed。org_id を `ValidateIdentifier` に照らし、通らなければ marker を書かずに断る。文の中のコマンドを `cleanupGitCmd` と `cleanupShellWord` で quote し、空白を含む repo で文のコマンドを `sh -c` で打つテストを足した。検証: 指定のコマンドが 2 回通過、変異 4 つが落ちる。機械的な確かめは clean(注釈の typo を fixed)。consult は妥当とした
    - F2 cbaa92f4(M1・L3・L4 と文): `/director` skill の skipped の読み方(断りか途中の失敗、MANUAL がなければ打ち直し)と「GitHub で merge した PR の後始末」を 5 手順に(SKIPPED のブランチは人が直したと言うまで打ち直さない)、`/org` skill、merge と cleanup の help、README、仕様 FR-8 の注(確かめ 3 の理由、D3 の skipped)、main worktree が見つからないときの文を定数 `orgMainWorktreeMissing` に、merge の中断の断りのブランチを quote。leader の雛形は同じ趣旨の文がなく変えていない。検証: 指定のコマンドが通過(679 秒)、壊した版 7 つで落ちる。機械的な確かめは clean
    - F2b 794aee4d(consult の遅れた勧め): CLI の文の中のブランチ(`mergeCleanupRoot`、`cleanupNamedStep`)を `shellQuoteIfNeeded` で quote。検証: cli の `OrgMerge|OrgCleanup` が通過(559 秒)、壊した版で 3 行ずつ落ちる。機械的な確かめは clean。範囲外: `internal/cli/org_regate.go:203` の regate の文もブランチが素のまま(この PR の前から。触らない)
    - self-review の run 2(report は fb214c0d): CRITICAL・HIGH・MEDIUM は 0、LOW が 8(R2-L1〜R2-L8)。consult と相談し、8 つとも F3 の 1 コミットで直すことにした
    - F3 5ddf5532(R2-L1〜R2-L8): 断りの文が安全なコマンドを名指す(記録の path か kind が違うときの `git log` と `git status`、別のブランチへの退避は `git -C <root> branch <branch>-unmerged <branch>`)。PR の head が手元にないときは `git -C <root> fetch origin refs/pull/<n>/head` を先に出す(番号がなければ SHA で取る。`CleanupParams.PRNumber` を merge と cleanup から渡す)。regate の `ralph org cleanup --branch` の quote。Failed の出どころを 2 つと書いた(script の途中と、その前の `git worktree remove`)。`/director` skill が exit 1 の混在でも CLEANED の行を報告する。tech-debt の行も足した。検証: handoff のコマンドが exit 0(internal/org 1001 秒、internal/cli 394 秒)、check-skill-sync と check-sync が PASS、壊した版 8 つがすべて落ちる。機械的な確かめは、director skill の行のつながりを fixed、ほかは clean
    - 計画の本文を直した(digest を取り直して自動承認、2026-10-28 sha256:0f38a9718a8b): D2 の手で消すコマンドを返す場面の「script が途中で止まった」を「消す途中で失敗した」に、D3 の skipped の失敗に、script の前の登録の外しの失敗(何も消していない)を足した。仕様 FR-8 の注も同じ形に直した(F3 の R2-L6 に合わせる。図解ページは「途中で止まったら」のままで食い違わない)
    - self-review の run 3(report は 3303f4cb、F3 と計画の直しだけを読んだ): CRITICAL・HIGH・MEDIUM は 0、LOW が 3。run 3 の LOW は台帳に書いて verify に進む方針だが、R3-L1(確かめ 5 の文が退避に名指す `<branch>-unmerged` は名前が固定で、すでにあると失敗し、文の次の手の reset が merge されていないコミットをブランチから外す)は、ツールの文のとおりに打つとコミットが外れる道なので、consult と相談して例外として F4 で直す(名前に tip の短い SHA を入れる)。R3-L2(fetch の案内は確かめ 5 にだけある)と R3-L3(`cleanup.go` が 800 行を越えた)は台帳に 2 行を足した
    - 重い処理の一時停止(2026-10-28): 別のセッション(stoic-app)が local CI を流すあいだ、go test・go build・run-verify・run-test を控えるよう頼まれた。F4 の検証と verify・test は再開の知らせのあとに回す。同じ日の夜、F4 を渡す前に再開の知らせが届いた
    - F4 3dfd434e(R3-L1): 確かめ 5 の断りが名指す退避を `<branch>-unmerged-<tip の先頭 7 文字>` にした。起点はブランチ名のまま(断りのあとに足したコミットも退避に入り、reset で外れない)。新しいテスト `TestCleanupFeatureWorktree_MovedBranchKeptUnderItsTip` は、同じ tip のあいだは同じ名前で、退避を打ち直すと失敗しても退避がその tip にあること、次のコミットで名前が変わることを見る。検証: handoff のコマンドが exit 0(internal/org 670 秒、internal/cli 234 秒)、壊した版 2 つ(名前を固定に戻した版、起点を tip にした版)で新しいテストが落ちる。機械的な確かめは 5 項目とも clean。consult の判定どおり run 4 は回さず、self-review の report に F4 の追記を 1 段足して verify に進む
- [x] Review artifact created(`docs/reports/self-review-2026-10-28-org-merge-cleanup.md`、run 3 と F4 の追記まで)
- [x] Verification artifact created(`docs/reports/verify-2026-10-28-org-merge-cleanup.md`、pass、所見は LOW 1 つ)
- [x] Test artifact created(`docs/reports/test-2026-10-28-org-merge-cleanup.md`、pass)
- [x] Sync-docs report created(`docs/reports/sync-docs-2026-10-28-org-merge-cleanup.md`。verify の LOW-1 の列挙 2 つと Info の注釈 1 つを 7e72d11e で直した。動詞を並べる文書に `cleanup` を足した: README の動詞の表、quality-gates の watch gate の行と template の写し)
- [x] origin/main(#280〜#285)を b164744f で merge した。衝突は main が `auto_merge` の既定を true にした行と、このブランチが `cleanup` を足した行の重なり。このブランチの文に main の変更を当て、このブランチが足した CLI の merge のテストの期待を `AUTO MERGE: true` に直した。build・vet・check-skill-sync・check-sync と絞ったテストが通った(`internal/org` の `TestRegate_AnInterruptBeforeTheRecordUndoesTheMerge` が load 10〜12 の下で 1 回落ち、単独では 5 回とも通る。このブランチはこのテストを変えていない)
- [x] cross-review cycle 1(triage は a7801152): ACTION_REQUIRED 2、WORTH_CONSIDERING 0、DISMISSED 0。consult(判定: 進めてよい)と相談して、2 つとも直して cycle 2 で /self-review から回し直す
  - S-X1(指摘 1): `ralph org cleanup` の PR の読み方を origin に固定する。merge の `readOrigin` から origin の host と owner/repo を読む部分を共通の関数に切り出し、merge-order の PR の読み方に、固定した repository を受ける欄を足す。欄があると `gh pr list` に `-R` を渡し、PR の url を host と owner/repo で origin と比べる。cleanup だけがこの欄を渡し、merge-order の一覧の動きは変えない。origin が GitHub の repository を名指さないとき、cleanup は何も消さずに断る(merge と同じ)。PR を使えなかった理由は cleanup の行が自分で出す
  - S-X2(指摘 2): worktree のディレクトリがなく git の登録だけが残るとき、登録の `branch`/`detached` を読み、登録が binding のブランチにいるときだけ進む(D2 の確かめ 6 を登録に延ばす)。detached、別のブランチ、ブランチの消えた登録は、何も消さずに断る。断りの文は登録の HEAD を示して退避の打ち方を出し、ralph が先へ進めないので手で消すコマンドも出す
  - S-X1 8e46db0f: `internal/org/mergeorigin.go` に `MergeOrigin`(host と owner/repo、`Arg()` が `-R` の値)と `ReadMergeOrigin` を足し、merge の `readOrigin` と `isOriginsPR` もこれを使う。`MergeOrderParams.Origin` があると merge-order は `-R` で引き、PR の url を host と owner/repo で比べ、origin の get-url は打たない。cleanup は先に origin を読み、読めなければ何も消さずに exit 1(`cleanupOriginRefusal`)。SKIPPED の行と `--branch` の断りは、PR を使えなかった理由を括弧で出す。`ralph org cleanup --help` に固定した読み方の段落。consult は設計を「変更なし」とした。ssh の alias の origin は merge と同じく SKIPPED になる(PR の既知の穴)
  - S-X2 d3452a9d: `readState` が登録の `HEAD` と `branch`/`detached` を読み、`run` は登録の残る場合に `checkRegistration` を先に通す。detached、別のブランチ、ブランチの消えた登録、HEAD を読めない登録は、何も消さずに断る。断りは登録の HEAD と退避のコマンド(`keptBranch` の `<branch>-unmerged-<sha7>`、`branchMovedSteps` と共通)を出し、HEAD がコミットを指さない(ゼロ)ときは退避のコマンドを出さずに `git rev-parse --git-common-dir` の下の worktrees/ の logs/HEAD を指す(handoff からの逸脱、案 C)。文は「手で片付ける。登録が残るので次の cleanup も断る」で終わり、Manual は `git worktree remove <wt> && <script>`。挙動の変化: ブランチだけ消えて登録が残る場合は、これまで記録だけの再開として外していたが、今は断る。検証: handoff のコマンドが exit 0(internal/org 206 秒、internal/cli 173 秒)、`checkRegistration` を外した版で新しいテストの 4 ケースが落ちる。機械的な確かめは 5 項目とも clean。`internal/org/cleanup.go` は 935 行(R3-L3 の台帳の行)
- [ ] パイプライン cycle 2
  - self-review cycle 2(report は 04c83f5c、reviewer は opus): CRITICAL・HIGH・MEDIUM は 0、LOW が 6(C2-L1〜C2-L6)、Merge 可。consult(判定: 進めてよい)と相談して 6 つとも直し、reviewer に直しだけを読ませる短い run を回す。C2-L3 は R3-L1 と同じく、文のとおりに打つとコミットが外れる道
  - C2-L5 060102bb(inline): 台帳の R3-L2 の行に `checkRegistration` の断りを足し、R3-L3 の行を `internal/cli/org_cleanup_test.go` も 800 行を越えたと直した
  - C2-F 54fa6351(C2-L1・L2・L3・L4・L6): 登録の断りの一文を `registrationAtRisk` にまとめ、`checkRegistration` と `recordGone` の両方が使う。記録がないときも、登録が binding のブランチにいなければ(ディレクトリが残っていても消えていても)HEAD と退避の打ち方を出す(C2-L3)。HEAD がゼロの登録の案内は `rev-parse --path-format=absolute --git-common-dir` と「gitdir が `<worktree>/.git` のディレクトリ」(C2-L1)。別のブランチの登録は退避を出さず、そのブランチは残り消えるのは reflog だけと言う(C2-L6、reflog の場所も出す)。`foreignPRWhy` に「check origin (git remote get-url origin)」(C2-L2、原因は書かない)。テストの注釈 2 か所に cross-review cycle 1 と書いた(C2-L4)。検証: handoff のコマンドが exit 0(internal/org 223 秒、internal/cli 351 秒)、変更前のコードに差し替えた版で、文の変わったケースと新しいテスト `TestCleanupFeatureWorktree_RecordGoneNamesADetachedRegistrationsHead` の 2 ケースが落ちる。機械的な確かめは 5 項目とも clean
  - self-review cycle 2 run 2(report は fdb9ee67、`git diff 04c83f5c..21785882` だけを読んだ): CRITICAL・HIGH・MEDIUM は 0、LOW が 3、Merge 可。C2-L1〜C2-L6 は解消。LOW 3 つ(`recordGone` の「remove it by hand」の it、`registrationAtRisk` の注釈の届かない分かれ、台帳の行)は cycle 1 の run 3 と同じく直さず、09e1e93b で台帳に 1 行足し、R3-L2・R3-L3 の行を直した。所見にしていない穴: git 2.48 以降の `worktree.useRelativePaths=true` では gitdir が相対パスになり、C2-L1 の文の `<worktree>/.git` と文字では合わない(PR の既知の穴)
  - verify cycle 2(HEAD 09e1e93b): pass。`run-static-verify.sh`・`check-skill-sync.sh`・`check-sync.sh` が rc 0。動きと受け入れ条件の食い違いはなし。/sync-docs に渡すもの: C2-D1(cleanup の PR の読み方に origin への固定と origin の断りがない: `/org` skill :234 と写し、README :278、仕様 :133、`/director` skill :986-987、台帳 :247 の ssh の別名の行)、C2-D2(手で消すコマンドが出る場面の列挙に登録の断りがない: cleanup と merge の help、`/org` skill と `/director` skill と写し、README、仕様。固定しているテストもある)、C2-D3(確かめ 6 がディレクトリのない登録にもかかる)、C2-D4(AGENTS.md の地図に `mergeorigin.go`)
  - test cycle 2(HEAD 9a6ae47c、report は 1341a3d4): pass。cycle 2 で変えた関数の coverage は `registrationAtRisk`・`checkRegistration`・`recordGone`・`lookupMergePRs` などが 100%。変異で確かめた判定はすべてテストに固定されている
  - sync-docs cycle 2: verify の C2-D1〜C2-D4 を直した。cleanup と merge の `--help`(登録の断りが手で消すコマンドを出す場面に入ること、確かめ 6 が登録にかかること)、`/org` skill の cleanup・merge の行と「org の終わり方」(「3 つ」を「4 つ」に)、`/director` skill の手順 5、README、仕様 FR-8 の注、tech-debt の ssh の別名の行、AGENTS.md の地図(`mergeorigin.go`)と、固定するテスト。C2-D5 はこの 2 行で解消。report は `docs/reports/sync-docs-2026-10-28-org-merge-cleanup.md` の cycle 2 の節
  - origin/main(#286)を 65a7b067 で merge した(衝突なし)。build・vet・check-skill-sync・check-sync・`tests/test-xreview-helpers.sh` と、両方が触った範囲の絞ったテスト(`internal/org` の `Skill|Prompt|Cleanup`、`internal/org/driver`、`internal/config`、`internal/cli` の `Help|OrgCleanup|OrgMerge`)が通った
  - cross-review cycle 2(triage は eca287bf、上限の run): ACTION_REQUIRED 2、WORTH_CONSIDERING 0、DISMISSED 0。指摘 1(P1)は `--state-dir` か `RALPH_ORG_STATE_DIR` が機能の worktree の中の台帳を名指すと script が台帳ごと消すこと、指摘 2(P2)は最後のブランチの片付けの途中の割り込みで `ralph org cleanup` が exit 0 になること。consult(判定: 進めてよい)と相談して、2 つを S-X3 の 1 スライスで直す。新しい確かめは `run` の頭(確かめ 4 と記録がない断りより前)に置き、計画の本文の D2 は変えない。2026-10-27 の方針どおり上限を `RALPH_STANDARD_MAX_PIPELINE_CYCLES=3` に上げ、`cycle-count.json` は 2 のまま増やさずに /self-review から回し直す
  - S-X3 4d7cbfd8(指摘 1・2): `checkLedger` を `cleanupRun.run` の頭(記録を読む前)に置き、台帳のディレクトリが機能の worktree かその中なら(symlink を解いて比べる)Manual なしで断る。理由は台帳を worktree の外に動かすか外の台帳を選び、`--state-dir` か `RALPH_ORG_STATE_DIR` で打ち直すと言う(結びつきは中の台帳にあるので「動かす」も足した)。`runCleanupSteps` はループのあとに ctx を見て、割り込みのエラー(exit 1)を `p.finish()` より先に返す。`TestOrgMerge_AutoMergeComesFromTheMainWorktreeRalphToml` は fixture が main worktree を機能の worktree として結びつけていて台帳がその中にあるので、期待を台帳の断りの 1 行に変えた。検証: handoff のコマンドが通過(internal/org 569 秒、internal/cli 289 秒)、確かめを外した版・記録の lookup のあとに移した版・symlink を解かない版・CLI の直す前の版で新しいテストが落ちる。機械的な確かめは 5 項目とも clean。sync-docs に渡すこと: `ralph org cleanup --help` の「Before it removes anything, cleanup checks...」の一覧と、skill の「何も消さない場合」の一覧に台帳の場所を足す
  - self-review(cycle 2 の再実行、report は 6c3738e2): CRITICAL・HIGH は 0、MEDIUM 1、LOW 4。C3-M1 は `checkLedger` が文字列の前置で比べるので、大文字小文字を区別しない macOS の APFS で書き方の違う同じディレクトリを見落とし、全 org の台帳が消えうること。C2 の前例どおり 5 つとも S-X4 で直す
  - S-X4 9dd79b94(C3-M1・C3-L1〜L4): `pathWithin` は名前で一致しなければ祖先を `os.SameFile` でたどる。`checkLedger` と確かめ 3 の作業ディレクトリの比べがこれを使う。断りの次の手は `ledgerMoveStep`(台帳を使っていないあいだに worktree の外へ動かし、`--state-dir` か `RALPH_ORG_STATE_DIR` で打ち直す)。記録のパスの worktree が台帳を持つときは Manual を出さない(C3-L2)。割り込みの文は出した行の数で言い分ける(C3-L3、ループの中の割り込みも)。検証: handoff のコマンドが通過(internal/org 81 秒、internal/cli 113 秒)。この Mac の TempDir は大文字小文字を区別せず、大文字小文字の行は走って通った。前の比べに戻した版・名前だけで比べる版・記録の確かめを外した版・行の数の分かれを外した版で新しいテストが落ちる。機械的な確かめは 5 項目とも clean
- [ ] PR created
