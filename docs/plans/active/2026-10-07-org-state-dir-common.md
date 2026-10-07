# org-state-dir-common

- Status: Approved
- Approved: 2026-10-07 sha256:d8a7f8ae89c1
- Owner: Claude Code
- Date: 2026-10-07
- Related request: org runtime を「機能ごとの org + herdr の外の director」に組み直す系列の 1 段目。council の結論(`~/.cache/council/20261007-0324-org-of-orgs/conclusion.md` の論点 1 と論点 9 の 1 段目)で、5 体すべてが最初に直すべきだとした
- Related issue: N/A
- Type: fix
- Branch: fix/org-state-dir-common

## Objective

linked worktree の中から打った `ralph org` / `ralph status` / `ralph insights` が、main のチェックアウトと同じ org の台帳(`.harness/state/org/`)を読み書きするようにする。

今は `internal/org/statedir.go` の `ResolveOrgStateDir` が、flag と env がないとき `git rev-parse --show-toplevel` に `.harness/state/org` を付けて置き場所を決める。linked worktree の中では show-toplevel が worktree 自身のルートを返すので、worktree ごとに別の manifest・receipts・watch の状態ができる。機能ごとに worktree を分ける後続の段(全 org の上限、担当範囲の予約、受信箱、横断の watch と stop)は、全 org が同じ台帳を見ることを前提にしている。

## Scope

- `internal/org/statedir.go`: 3 段目の解決を「main worktree のルート」起点に変え、そこに `.harness/state/org` を付ける。source のタグは `git-main-worktree`。main worktree のルートは次の順で決める。flag と env の優先順は変えない
  - `git rev-parse --path-format=absolute --git-dir --git-common-dir` の 2 つが同じなら、cwd は main worktree の中にあるので、show-toplevel をルートにする
  - 違うなら(linked worktree の中)、`git worktree list --porcelain` の先頭の記録をルートにする。その記録が bare のとき、パスが common dir そのものを指すとき、ディレクトリが存在しないとき、コマンドが失敗したときは、今の show-toplevel に戻す(タグは `git-toplevel` のまま)
  - git の外なら今の cwd に戻す
- 古い台帳の扱い: 解決結果が `git-main-worktree` で、cwd が linked worktree の中にあり、その worktree の `<show-toplevel>/.harness/state/org/manifest.jsonl` が存在するとき、それを「古い台帳」として扱う。台帳を移したり混ぜたりはしない
  - 古い台帳に動いている座席があるとき、台帳を書き換える動詞(`ralph org spawn` / `start` / `send` / `stop` / `disband` / `watch`)は終了コード 1 で止まる。メッセージに、古い台帳のパス、新しい台帳のパス、`--state-dir` でどちらかを選ぶ方法を書く
  - 動いている座席がないとき、または読むだけの動詞(`ralph status`、`ralph org status` / `read` / `wait` / `report`、`ralph insights`)では、stderr に 1 回だけ注意を出して続ける。JSON の stdout は変えない
  - flag か env で置き場所を決めたときは、どちらもしない
- codex の座席: permission mode が edits か autonomous(`--sandbox workspace-write`)の codex 座席で、解決した台帳のディレクトリが座席の cwd の下にないとき、起動の引数に `--add-dir <台帳のディレクトリ>` を足す。cwd の下にあるとき、guarded のとき、claude の座席では足さない
- 呼び出し元: `internal/cli/org.go`(:103、:526、:938 の 3 か所)、`internal/cli/status.go:48`、`internal/cli/insights.go:48` で、上の拒否と注意を呼ぶ。解決そのものは関数の中で変わるので、呼び出し元の解決の書き方は変えない
- 文言: `--state-dir` フラグのヘルプ文 2 か所(`internal/cli/org.go:42`、`internal/cli/status.go:53`)、`statedir.go` と `org.go`・`status.go` のコメントの tier 名、`/org` skill の「前提」節(`.claude/skills/org/SKILL.md` と、`scripts/sync-skills.sh` で再生成する `.agents/skills/org/SKILL.md`、`templates/base/` の 2 面、計 4 面)、`docs/recipes/codex-seat-permissions.md`(root と `templates/base/` の 2 面。`--add-dir` を足すことと、その理由)
- テスト: `internal/org/statedir_test.go`、`internal/org/permissions_test.go`(または spawn のテスト)、`internal/cli` のテスト

## Non-goals

- 古い台帳(worktree の中の `.harness/state/org/`)の自動移行や統合。止めるか注意を出すだけにする
- bare リポジトリの worktree で台帳を 1 つにすること。今と同じく worktree ごとに分かれる(show-toplevel に戻す)
- `--separate-git-dir` で git dir の名前を `.git` にしたリポジトリの linked worktree。git 自身が git dir の親を main と返し、本当の作業ツリーはリポジトリのどこにも記録されないので、ralph からは区別できない。この場合の台帳は git dir の親の下になる(`statedir.go` のコメントに書く)
- `scripts/insights-append.sh` など、org 以外の `.harness/state/` の置き場所(standard pipeline の状態は worktree ごとに分かれているのが正しい)
- 座席の環境への `RALPH_ORG_STATE_DIR` の注入(council の D 案)
- 古いバイナリの挙動を変えること。戻すときの手順は「Rollout or rollback notes」に書く
- 後続の段(横断 status・stop、上限、担当範囲の予約、受信箱、director)。新しい spec(並行して `/spec` で書く)で扱う

## Assumptions

- `git worktree list --porcelain` の先頭の記録は main worktree で、bare リポジトリなら `bare` の行が付く(git の仕様)。ただし `--separate-git-dir` で作ったリポジトリでは、先頭の記録は実際の作業ツリーではなく、git dir の親になる(git 2.49 で実装中に確認。git は common dir から `/.git` を除いた値を main とみなす)。このため main worktree の中では先頭の記録を使わず、show-toplevel を使う
- 非 bare の通常のリポジトリの main のチェックアウトでは、先頭の記録のパスは show-toplevel と同じなので、今の解決結果と同じパスになる。既存の利用者の台帳は動かない(source のタグだけが `git-toplevel` から `git-main-worktree` に変わる)
- `source` のタグは診断用の表示(`ralph status` の `state-dir: ... (source: ...)` 行と `--json` の `state_dir_source`、`ralph org watch` の状態)で、外部の契約ではない
- 台帳を書き換えるのは leader 座席と人(将来は director)だけで、implementer と reviewer の座席は agmsg で送る(`internal/org/prompts/implementer.md`、`reviewer.md` に `ralph org` の動詞は出てこない)。codex の座席の `--add-dir` が効くのは、主に codex で動く headless leader
- codex-cli 0.160.0 の `codex --help` に `--add-dir <DIR>`(Additional directories that should be writable alongside the primary workspace)がある。sandbox の中で実際に書けるかの実機確認は、`codex_verified` と同じく利用者のマシンで行う(recipe に手順を書く)

## Affected areas

- `internal/org/statedir.go`、`internal/org/statedir_test.go`
- `internal/org/permissions.go`、`internal/org/spawn.go`(codex の `--add-dir`)と、そのテスト
- `internal/cli/org.go`、`internal/cli/status.go`、`internal/cli/insights.go`(拒否と注意、ヘルプ文・コメント)と、そのテスト
- `.claude/skills/org/SKILL.md` と 3 つの写し(`.agents/skills/org/SKILL.md`、`templates/base/.claude/skills/org/SKILL.md`、`templates/base/.agents/skills/org/SKILL.md`)
- `docs/recipes/codex-seat-permissions.md`、`templates/base/docs/recipes/codex-seat-permissions.md`

## Visual review

- ページ: `.harness/state/plan-visual/org-state-dir-common.html`(図 1 全体、図 2 どの台帳を読み書きするか、図 3 置き場所を決める順番と古い台帳の扱い)
- セルフチェック: 全体と全体図(`--fragment overview`)を `plan-visual.sh shot` で撮って確認した。初版では図 3 の S2 バッジのラベルへの接触と、右下のノードの文字のはみ出しを直した。Codex の指摘を反映した版(S3・S4 の追加、`git worktree list` 方式、古い台帳の扱い)も撮り直し、重なり・はみ出し・矢印の外れがないことを確かめた

## Design decisions

Critical forks: None

既定の判断で決めたこと(どれも 1 スライス以内で戻せる):

- main worktree は、main の中なら show-toplevel、linked worktree の中なら `git worktree list --porcelain` の先頭の記録で決める。最初の案(common dir の名前が `.git` なら親)は、bare リポジトリを `/x/project/.git` に置いた場合や、別置きの git ディレクトリが `.git` という名前の場合に、関係のない親を選んでしまう(Codex plan advisory の指摘 3)。先頭の記録だけで決める 2 つ目の案は、`--separate-git-dir` のリポジトリの main の中で git dir の親を選んでしまう(S1 の実装中に git 2.49 で確認した計画からの逸脱。承認をやり直した)
- source のタグは `git-main-worktree` を足し、戻り先の `git-toplevel` と `cwd` はそのまま残す
- 古い台帳は移さない。移行は座席の pane や agmsg の team を伴い、壊れたときに戻しにくい。代わりに、動いている座席が古い台帳にあるときは書き換える動詞を止める。注意だけだと、空の台帳に対する `disband` が成功して、実際の座席が動き続ける(Codex plan advisory の指摘 2、`internal/org/verbs.go` の `Disband`)
- codex の座席には `--add-dir` で台帳のディレクトリを書けるようにする。workspace-write の sandbox は cwd の外に書けないので、linked worktree にいる codex の leader が共通の台帳に書けなくなる(Codex plan advisory の指摘 1、`internal/org/permissions.go:66`)

## Acceptance criteria

- [ ] AC1: flag も env もない状態で、通常のリポジトリの main のチェックアウトのルートとサブディレクトリから `ResolveOrgStateDir("", false)` を呼ぶと、`<main のルート>/.harness/state/org` と source `git-main-worktree` が返る
- [ ] AC2: 同じリポジトリの linked worktree のルートとサブディレクトリから呼ぶと、AC1 と同じパス(worktree のパスではない)と source `git-main-worktree` が返る
- [ ] AC3: bare リポジトリ(名前が `.git` のものを含む)の linked worktree から呼ぶと、今と同じく show-toplevel 起点のパスと source `git-toplevel` が返る。`git init --separate-git-dir <別の場所>/.git <作業ツリー>` で作ったリポジトリでは、作業ツリーの下のパスと source `git-main-worktree` が返る(別の場所の親ではない)
- [ ] AC4: flag と env の優先順は変わらない(既存の `TestResolveOrgStateDir_*` がそのまま通る)。git の外では source `cwd` が返る
- [ ] AC5: linked worktree の中の古い台帳に動いている座席があるとき、`ralph org spawn` / `start` / `send` / `stop` / `disband` / `watch` は終了コード 1 で止まり、メッセージに古い台帳のパス・新しい台帳のパス・`--state-dir` が出る。台帳と herdr には何もしない。`--state-dir` か `RALPH_ORG_STATE_DIR` で置き場所を選ぶと止まらない
- [ ] AC6: 古い台帳に動いている座席がないとき、または読むだけの動詞(`ralph status`、`ralph org status` / `read` / `wait` / `report`、`ralph insights`)では、stderr に古い台帳のパスと `--state-dir` の使い方を含む注意を 1 回出して続ける。`--json` の stdout は変わらない。古い台帳がないとき、main のチェックアウトにいるとき、flag か env で置き場所を決めたときは出さない
- [ ] AC7: linked worktree から打った `ralph status --json` の `state_dir` が main 側の `.harness/state/org` を指す(CLI レベルのテスト)
- [ ] AC8: permission mode が edits か autonomous の codex 座席で、台帳のディレクトリが座席の cwd の下にないとき、起動の引数に `--add-dir <台帳のディレクトリ>` が入る。cwd の下にあるとき、guarded のとき、claude の座席では入らない(引数を組み立てる部分の単体テスト)
- [ ] AC9: `--state-dir` のヘルプ文 2 か所、`/org` skill の「前提」節(4 面)、`docs/recipes/codex-seat-permissions.md`(2 面)が新しい解決順と `--add-dir` を書いている。`./scripts/check-skill-sync.sh` と `./scripts/check-sync.sh` が通る

## Implementation outline

1. S1(`internal/org/statedir.go` と `statedir_test.go`): main worktree 起点の解決と戻り先。古い台帳を見つけて、動いている座席の数を返す関数(manifest の `Roster` / `ActiveSeatCount` と同じ判定を使う)。テストは一時ディレクトリに `git init`・空コミット・`git worktree add`、bare、`--separate-git-dir` で作る(AC1〜AC4)
2. S2(`internal/cli/org.go`、`status.go`、`insights.go` とテスト): 書き換える動詞の前で止める判定と、読むだけの動詞の注意を、小さな helper にまとめて呼ぶ。ヘルプ文とコメントを直す(AC5〜AC7)
3. S3(`internal/org/permissions.go` / `spawn.go` とテスト): codex の workspace-write の座席に `--add-dir` を足す(AC8)
4. S4(文言): `/org` skill の「前提」節を直して写しを作り直し、codex の recipe の 2 面に `--add-dir` の説明を足す(AC9)

## Verify plan

- Static analysis checks: `./scripts/run-static-verify.sh`(go vet、gofmt、shellcheck、check-sync、check-skill-sync を含む)
- Spec compliance criteria to confirm: AC1〜AC9。flag と env の優先順が変わっていないこと。台帳を移す処理が入っていないこと。止める動詞と注意だけの動詞の分け方が Scope のとおりであること
- Documentation drift to check: `git grep -n 'git-toplevel\|toplevel .harness/state/org\|リポジトリルートの'` で、新しい解決順と食い違う説明が残っていないか(`docs/plans/archive/`・`docs/reports/`・`docs/tech-debt/` の RESOLVED 行は履歴なので除く)
- Evidence to capture: verify のレポート(`docs/reports/verify-2026-10-07-org-state-dir-common.md`)

## Test plan

- Unit tests: `go test ./internal/org/ -run 'ResolveOrgStateDir|Legacy|Permission'`
- Integration tests: `go test ./internal/cli/` で、linked worktree からの `ralph status --json`、古い台帳に動いている座席があるときの `ralph org stop` の拒否と `--state-dir` での回避、注意だけのケース
- Regression tests: `./scripts/run-test.sh`(`go test ./...` と `tests/test-*.sh`)。既存の flag・env・cwd のテストと、codex・claude の permission の既存テストが通ること
- Edge cases: main のサブディレクトリ、linked worktree のサブディレクトリ、名前が `.git` の bare リポジトリ、`--separate-git-dir` の先の名前が `.git`、main worktree のディレクトリが消えている(prunable)場合、git の外、macOS の `/var` → `/private/var` の symlink(既存の `chdir` helper が EvalSymlinks した値で比べる)
- Evidence to capture: test のレポート(`docs/reports/test-2026-10-07-org-state-dir-common.md`)

## Risks and mitigations

- 動いている org の座席を見失う: linked worktree の中で `ralph org` を使っていた人は、バイナリを更新すると別の台帳を見る。書き換える動詞は止まり(AC5)、読むだけの動詞は注意を出す(AC6)ので、黙って空の台帳を操作することはない
- codex の leader が台帳に書けない: `--add-dir` を足す(AC8)。ユーザーの codex の設定によっては効かない可能性があるので、recipe に実機での確かめ方を書く
- source のタグの変化: main のチェックアウトでも `git-toplevel` が `git-main-worktree` に変わる。タグを比べている既存のテストは直す。外部の契約ではないが、PR の本文に書く
- `git worktree list` の失敗や想定外の出力: 先頭の記録を読めなければ show-toplevel に戻すので、今より悪くならない
- `GIT_DIR` や `GIT_COMMON_DIR` が設定された環境(git hook の中など): git がそれに従うので、ralph も同じ結果になる。特別な処理はしない

## Rollout or rollback notes

- 配布はリリースのバイナリと `ralph upgrade`(skill と recipe の文言)。main のチェックアウトでの置き場所は変わらないので、多くの利用者には移行の手順は要らない
- 更新の前: linked worktree の中から leader や操作者が `ralph org` を使っている org があれば、先に `stop` / `disband` しておく。残したまま更新した場合、その worktree から打つ書き換えの動詞は止まるので、`--state-dir <worktree>/.harness/state/org` を付けて片付ける
- 戻すとき(この PR を revert、または古いバイナリに戻す): 古いバイナリは、更新後に共通の台帳に作られた座席を linked worktree からは見つけられず、注意も出さない。戻す前にそれらの座席を `stop` / `disband` するか、戻したあと `--state-dir <main>/.harness/state/org`(または `RALPH_ORG_STATE_DIR`)を付けて片付ける。台帳のファイルは動かしていないので、データの戻しは要らない

## Open questions

- なし

## Progress checklist

- [x] Plan reviewed
- [x] Plan approved
- [x] Branch created
- [x] Implementation started
- [ ] Review artifact created
- [ ] Verification artifact created
- [ ] Test artifact created
- [ ] PR created
- 2026-10-07: Codex plan advisory の 3 件(sandbox の書き込み、動いている座席がある状態の切り替えと戻し、`.git` の名前だけで main と決める誤り)は、ユーザーが「計画を直す」を選び、Scope・AC・Design decisions・Rollout に反映した
- 2026-10-07: S1 を c8dc2e8e でコミットした(implementer)。実装中に `--separate-git-dir` のリポジトリでは `git worktree list` の先頭が git dir の親になるとわかり、main の中は show-toplevel を使う形に変えた。計画の Scope・Assumptions・Non-goals・Design decisions と図 2・図 3 を直し、ユーザーが承認し直した(digest d8a7f8ae89c1)
- 2026-10-07: S2 を c90a504e でコミットした(implementer)。書き換えの動詞は spawn(`--dry-run` を含む)・start・send・stop・disband・watch、読むだけの動詞は org の status・read・wait・report と `ralph status`・`ralph insights`。計画が決めていなかった 2 点は次のようにした。古い台帳が読めないとき、書き換えの動詞は止め、読むだけの動詞は注意を出す。`ralph insights` には `--state-dir` がないので、注意では `RALPH_ORG_STATE_DIR` を案内する
