# org-merge-cleanup

- Status: Approved
- Approved: 2026-10-28 sha256:25aae0c272fe
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
- D2 消す前の確かめ。どれかが通らなければ消さず、理由と手で消すときのコマンドを返す
  0. `regate.lock` を持ち、その marker に片付ける org の org_id を書く(merge は持ったまま呼ぶ。`ralph org cleanup` は merge と同じく取る。取れなければ regate か merge が走っているので断る)。merge は今 marker を空にするだけで(`internal/org/merge_run.go:203`)、`regateSpawnErr`(`internal/org/spawn.go:1464-1479`)は marker が空なら spawn と start を通すので、marker を書かないと、片付けの最中に同じ org_id の `ralph org start --plan` が worktree を作り直しにかかる(`internal/org/feature.go:216`)。「merge は worktree を変えないので marker を書かない」という注釈 3 か所(`internal/org/lockfile.go:143-149`、`internal/org/regate.go:91-98`、`internal/org/spawn.go:1456-1461`)も直す
  1. 台帳にそのブランチの binding があり、その org が走っていない(`RunningOrgs`)
  2. `ralph-worktree.sh` の記録 `org-<org_id>` があり、その branch と worktree_path と kind(`org`)が binding と一致する(同じ org_id を別の機能で立て直したとき、別の機能の worktree を消さないため)
  3. ralph を打ったディレクトリがその worktree の中にない(script は中から打つとブランチを残して記録だけ消すため)
  4. worktree に未コミットの変更も未追跡のファイルもない(`git -C <worktree> status --porcelain` が空)。rebase や merge の途中でない
  5. ローカルのブランチの先頭が、merge された PR の head と一致する(`ralph org merge` では gate の head、`ralph org cleanup` では gh の `headRefOid`)。一致しなければ、push していないコミットがありうるので消さない
  6. worktree が名前のブランチ(記録の branch)にいて(`git -C <worktree> symbolic-ref --short HEAD`)、その HEAD も merge された head と一致する。detached HEAD や別のブランチに切り替えた worktree では、そこで足したコミットへの参照が worktree と一緒に消えるので、消さない(Codex の計画助言 1)

  再開(Codex の計画助言 2): script は worktree、ブランチ、記録の順に消し、途中で止まると前の段だけが消えた状態が残る。もう一度片付けを打ったときに、残った状態ごとに次のように続ける。確かめは、残っているものにだけかける
  - worktree のディレクトリがなく、ブランチと記録が残る(ブランチの削除で止まった): 3・4・6 は worktree がないのでかけず、1・2・5 だけをかけて script を打つ。git に worktree の登録だけが残っているときは、その 1 つだけを外す(`git worktree prune` はほかの worktree の登録にも効くので使わない。外し方は実装で確かめる)
  - worktree もブランチもなく、記録だけが残る: 1・2 だけをかけて script を打つ(記録を消すだけになる)
  - 記録がない(人が `ralph-worktree.sh gc --prune` などで消した): 消すものを特定できないので片付けず、残ったブランチを手で消すコマンド(ブランチの先頭を確かめてから `git branch -D`)を出す
  どの段で止まっても、もう一度の `ralph org merge` の続きか `ralph org cleanup` で片付くことをテストで確かめる。断りの文で `gc --prune` は勧めない

  確かめの git(`status --porcelain`、`rev-parse`)は、regate と同じ時間の上限つきの git(`regateBoundedGit`、`internal/org/regate_git.go:173`)で打ち、`MergeParams.Exec` は使わない。merge のテストの「打つコマンドはすべて `/repo` で origin に固定」という検査(`internal/org/merge_fake_test.go:464`)と分けるため
- D3 `ralph org merge` は、片付けが断られても失敗しても終了コードを変えない(exit 0 は記録した merge、`internal/cli/org_merge.go:236-238`)。stdout に `CLEANUP: done` か `CLEANUP: skipped (<理由>)` の 1 行を足し、skipped のときは手で消すコマンドを `NOTE:` と同じ形で出す。片付けは `MergeParams` で opt-in にし(repo root が空なら呼ばない)、CLI だけが渡す(org 層の今のテストは script のない `/repo` を使うため)。CLI の repo root は、`ralph org start --plan` と同じく `org.FeatureRepoRoot`(`internal/cli/org.go:1087`、`internal/org/statedir.go:128`)で求める
- D4 動詞の名前は `ralph org cleanup`。`--branch` を付けるとそのブランチだけ、付けなければ merge 済みで記録の残るすべてのブランチを片付ける。1 本ごとに `CLEANED: <ブランチ> org-<org_id>` か `SKIPPED: <ブランチ> <理由>` の行を出す。終了コードは、すべて片付いたか片付けるものがなければ 0、1 本でも残れば 1。gh がないか PR を読めないときは、merge 済みを判定できないと言って 1 で終わる。`--branch` のブランチが `merged` でなければ断る。binding がなく gate の記録だけのブランチは対象にしない(機能の worktree は `ralph org start --plan` の org にしかなく、binding がなければ消すものを特定できない)。`--branch` なしでは、片付け済みで何も残っていないブランチ(記録も worktree もローカルのブランチもない)と、同じ org_id がのちに別のブランチに結びつけ直された古い binding は、数えず行も出さない。前者は org 層の結果 `NothingLeft` で、後者は merge-order の binding の `BoundTo`(org の最新の binding が別のブランチのとき、そのブランチ)で見分ける。結びつけ直しが通った時点で古い記録と worktree はもうないので(`checkFeatureWorktreeReuse`、`internal/org/feature.go:225`)、残りうる古いローカルのブランチは `--branch` で名指ししたときだけ扱う(S3 の consult による)
- D5 片付けたら、台帳に org の記録 `worktree_cleaned`(`branch=`、`worktree=`、`head=`、`via=merge|cleanup`)を書く。`ralph org report` と人が台帳で後始末を追えるようにするため。`RunningOrgs` と Roster の状態には入らない(`pr_merged` と同じ扱い)
- D6 gitignore されたファイル(feature worktree の `.harness/state/` や `docs/evidence/` の log)は、止める理由にしない。script と `git worktree remove` が今もそう扱い、merge のあとの worktree の中の生成物だから。リスクの表に書き、skill の後始末の節にも 1 文書く

Critical forks: なし。issue が方針(merge の続きで消す、直接 merge には新しい動詞、未コミットと未追跡では消さない、走っている org と regate・merge の最中は消さない)を決めている。動詞の名前は issue が計画に任せた命名で、D4 に理由を書いた。ブランチを `-D` で消す前の head の照合(D2 の 5)は、issue の「PR が MERGED であることを根拠に消す」を、push していないコミットを失わない形で満たすための既定である。

## Acceptance criteria

- [ ] AC1 `ralph org merge` で merge したあと、その機能の worktree・ローカルのブランチ・`ralph-worktree.sh` の記録が消え、台帳に `worktree_cleaned` があり、stdout に `CLEANUP: done` が出る。squash merge でリモートのブランチが消えたあとも止まらない(issue の受け入れ条件 1)
- [ ] AC2 GitHub で直接 merge したブランチを `ralph org cleanup` で片付けられる。`--branch` なしなら merge 済みで記録の残るすべてのブランチ、`--branch` ならそのブランチだけ(issue の受け入れ条件 2)
- [ ] AC3 未コミットの変更か未追跡のファイルがあるとき、ローカルのブランチの先頭が merge された head と違うとき、worktree が detached HEAD か別のブランチにいるとき、記録が binding と一致しないとき、ralph をその worktree の中から打ったときは、何も消さずに理由と手で消すコマンドを出す。`ralph org merge` の終了コードは 0 のまま(issue の受け入れ条件 3)
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
- [ ] Review artifact created
- [ ] Verification artifact created
- [ ] Test artifact created
- [ ] PR created
