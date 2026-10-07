# Walkthrough: org-state-dir-common

- Date: 2026-10-07
- Plan: docs/plans/archive/2026-10-07-org-state-dir-common.md(この PR の最後のコミットで archive に移す)
- Branch: fix/org-state-dir-common(base main 1c4cea5a。途中で origin/main を 8018e4bf で取り込んだ)
- Diff(HEAD 5bbc980d、この walkthrough を除く): 31 files、+2,186 / -111。報告・insight・plan・evidence・ミラー(`.agents/`、`templates/`)を除くと 18 files、+1,401 / -89
- PR: #207

## 何を変えたか

org の台帳(`.harness/state/org/`)の既定の置き場所を、main worktree 起点にした。linked worktree の中から打った `ralph org` / `ralph status` / `ralph insights` も、main と同じ台帳を使う。linked worktree に残った古い台帳に動いている座席があるときは、書き換えの動詞を止める。codex の leader 座席には、台帳のディレクトリが cwd の外なら `--add-dir` を足す。機能ごとの org を並べる系列(`docs/specs/2026-10-07-org-multi-org-director.md`)の 1 段目にあたる。

## 読む順

1. `c8dc2e8e` 解決(`internal/org/statedir.go`)。`ResolveOrgStateDir` の 3 段目が `gitMainWorktree` になった。`git rev-parse --path-format=absolute --git-dir --git-common-dir` の 2 つが同じなら main の中なので show-toplevel を使い、違えば `git worktree list --porcelain` の先頭の記録(`parseMainWorktreeRecord`)を使う。bare、common dir そのもの、存在しないディレクトリは show-toplevel の段に戻る。`LegacyWorktreeStateDir` は、source が `git-main-worktree` のときだけ、worktree 側の古い台帳と動いている座席の数を返す
2. `c90a504e` CLI の判定(`internal/cli/org_legacy_ledger.go`、新規)。`guardLegacyOrgStateDir` を、`newOrgRuntime` の 4 つ目の引数(書き換えか読むだけか)と、自分で解決する send・watch・status・insights から呼ぶ。止めるのは、古い台帳に動いている座席があるときと、古い台帳が読めないときの書き換えの動詞だけ
3. `53aa1877` と `c85e5f59` codex の `--add-dir`(`internal/org/permissions.go` の `codexWritableRootArgs`、`spawn.go` の起動引数)。最初は全役割に付けていたが、self-review の M-1 を受けて leader だけにした
4. `bde89dc0`・`a6dc9eb8`・`8c1ac257` 文書(`/org` skill の 4 面、codex の recipe の 2 面)と、`docs/evidence/codex-add-dir-2026-10-07.md`(`c85e5f59` で追加)
5. 実装後のパイプライン
   - `477cbf6b`・`4961d0f5` self-review の 2 回(1 回目 MEDIUM 1・LOW 5、2 回目 LOW 2)、`3832595b` L-6 の折り返し
   - `43896feb` verify(pass)、`5672ff92` test(pass)
   - `54f7c187` /sync-docs(tech-debt の 3 行、仕様の FR-1 の書き方、`worktrees.md`、`repo-map.md`、git 2.31 未満の記述)
   - `5bbc980d` cross-review(Codex、指摘 0 件)

## 計画からの逸脱(すべてユーザーが承認し直した)

- `--separate-git-dir` のリポジトリでは `git worktree list` の先頭の記録が git dir の親になる(git 2.49 で確認)。main の中では先頭の記録を使わず show-toplevel を使う形にした(`e121a5f4`)
- `--add-dir` は leader の座席だけに付ける(`42bd8fa6`)

## 変えていないもの

- flag(`--state-dir`)と env(`RALPH_ORG_STATE_DIR`)の優先順
- main のチェックアウトでの台帳のパス。変わるのは `ralph status` の `source:` の表示(`git-toplevel` → `git-main-worktree`)だけ
- 古い台帳のファイル。移しも混ぜもしない
- claude の座席の起動引数、codex の guarded の座席、implementer と reviewer の座席の起動引数

## 下流への届き方

| 所有権 | ファイル | upgrade での扱い |
|---|---|---|
| バイナリ | `internal/org/`、`internal/cli/` | バイナリを更新すると効く |
| core | `/org` skill(`.claude/skills/`、`.agents/skills/`)、`docs/recipes/codex-seat-permissions.md`、`docs/recipes/worktrees.md` | 置き換わる |

更新の前に、linked worktree の中から `ralph org` を使っている org を stop / disband しておく。残したまま更新した場合は、その worktree から打つ書き換えの動詞が止まるので、`--state-dir <worktree>/.harness/state/org` を付けて片付ける。

## 確かめたこと

- `./scripts/run-verify.sh` は pass(コミット済みの状態で)。`./scripts/run-test.sh` は shell 2,047 / 2,047、Go 8 パッケージ ok
- mutation 32 件中 30 件をテストが検出した。残る 2 件は tech-debt に記録した
- ビルドしたバイナリで、書き換えの動詞が終了コード 1 で止まること、`status --json` の stdout が古い台帳の有無で変わらないことを確かめた(test の報告)
- codex-cli 0.160.0(macOS)で、`--add-dir` なしでは cwd の外に書けず、ありでは書けることを確かめた(`docs/evidence/codex-add-dir-2026-10-07.md`)

## 確かめていないこと

- herdr で起動した codex の leader 座席での `--add-dir`
- Windows のパス
- AC6 の端のケース(main のチェックアウトで台帳のディレクトリ自体が読めないときのメッセージ)は tech-debt に送った
