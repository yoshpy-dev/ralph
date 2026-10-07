# Self-review report: org-state-dir-common

- Date: 2026-10-07
- Plan: docs/plans/active/2026-10-07-org-state-dir-common.md
- Branch: fix/org-state-dir-common(HEAD c85e5f59、origin/main は 8018e4bf で取り込み済み)
- Reviewer: reviewer subagent (Claude)、pipeline cycle 1 の self-review 2 回目(前回の M-1 と L-1〜L-4 の修正後)
- Scope: diff の品質だけを見た(命名、読みやすさ、不要な変更、コメントの正確さ、null と境界の扱い、エラー処理、安全性)。対象は `git diff origin/main...HEAD`(22 ファイル、+1786/-103。plan と前回の本レポートを含む)。前回のレビュー(HEAD 8c1ac257)以降に増えたコードは修正 commit c85e5f59 だけなので、そこを全行読み、それ以外のコードは `git diff origin/main...HEAD` で読み直した。`.agents/skills/` と `templates/base/` はコピーなので、root 側を読み、コピーとの差は `diff` で見た。仕様への適合、テストの網羅、文書のずれは見ていない(`/verify`、`/test`、`/sync-docs` の担当)。リポジトリのテスト、linter、vet は実行していない。

## 前回からの変化

前回(477cbf6b)は MEDIUM 1 件(M-1)と LOW 5 件(L-1〜L-5)だった。

| 前回 | 状態 | 確かめたこと |
| --- | --- | --- |
| M-1(`--add-dir` が全役割の codex 座席に付く) | 解決 | `codexWritableRootArgs(role, driver, mode, cwd, stateDir)` が `role != LeaderIdentity` で nil を返す(`permissions.go:170`)。呼び出しは `spawn.go:762` の 1 か所で、`p.Role` を渡している。`add-dir` を repo 全体で grep し、「全座席に付く」と読める記述が残っていないことを確かめた(root と `templates/base/` の `/org` skill 4 面、recipe 2 面、`spawn.go` と `permissions.go` のコメント、plan の Scope と AC8)。`internal/org/prompts/` を grep すると、`ralph org` の動詞が出るのは `leader.md` だけで、`implementer.md` と `reviewer.md` には出ない。「台帳を書くのは leader だけ」という新しいコメントと文書の根拠はこれ |
| L-1(tier 名のコメント) | 解決 | `statedir.go:44-46` は番号を使わず「`git-toplevel` tier が返すのと同じパス」と書く。`statedir_test.go:36-37` の `initGitRepo` は「git-main-worktree tier」に直った。`tier [0-9]` を grep して残るのは `statedir.go:49` の「tier 3」だけで、同じコメントの番号付きリストの 3 番を指していて正しい |
| L-2(折り返し) | 前回の 2 か所は解決。同じ型が新しい行で再発(L-6) | `org.go:178` 付近と `status.go:212` 付近は前後と同じ幅に折り直された |
| L-3(`resolvedOrClean` の前提) | 解決 | `codexWritableRootArgs` のコメントが「stateDir がすでに存在すること」を前提に書き、Spawn が `withManifestLock` のあとに呼ぶことと、存在しない場合に `/var` と `/private/var` で食い違うことまで述べている。`withManifestLock` が先にディレクトリを作る点(`lockfile.go`)は前回確認済みで、この commit で呼び出しの順は変わっていない |
| L-4(evidence の欠落) | 解決 | `docs/evidence/codex-add-dir-2026-10-07.md` が加わり、recipe の「Checked on 2026-10-07」の直後から参照している。evidence の `Plan:` 行が archive 先を指す形は `codex-seat-permissions-2026-09-18.md` と同じ書き方。ユーザー名は `/Users/<user>/` に伏せてあり、実際の `$HOME` は diff にない |
| L-5(`docs/tech-debt/README.md:144`) | 未解決(plan が /sync-docs に回している) | 144 行目は今も「`--state-dir`、`RALPH_ORG_STATE_DIR`、git の toplevel、cwd」の順のまま。この commit は register に触っていない |

この修正が新しく持ち込んだ問題は L-6 の 1 件だけ。

## Evidence reviewed

- 修正 commit c85e5f59 のコードと文書の差分を全行読んだ(`permissions.go`、`permissions_test.go`、`spawn.go`、`spawn_test.go`、`statedir.go`、`statedir_test.go`、`org.go`、`status.go`、`/org` skill 4 面、recipe 2 面、evidence、plan)。コミットメッセージに attribution の行はない。
- `codexWritableRootArgs` の役割の判定を、呼び出し元から追った。`ralph org start` は `SeatID == Role == LeaderIdentity` で spawn する(`org.go:471`)。`ralph org spawn --role leader --id <x>` は `Role` で判定するので、leader 用の template を使う座席として `--add-dir` が付く。逆に `--id leader --role implementer` は `leaderSelfSpawn`(`SeatID` 判定、`spawn.go:833`)には当たるが、`--add-dir` は付かない。どちらも権限が増える向きの食い違いではない。`--role Leader` のように綴りが違う役割は template がなく、`--add-dir` も付かない(より少ない権限に倒れる)。
- `--dry-run` の経路(`dryRunSpawn`、`spawn.go:387-417`)は `permissionArgsForDriver` で検証するだけで、起動の引数を組み立てない。`--add-dir` の有無で dry-run と本番が食い違う箇所はない。
- テストを読んだ。`TestCodexWritableRootArgs` は役割の軸を追加し、implementer、reviewer、未知の役割、空の役割で nil を確かめる。`TestOrgSpawn_Codex_WorkspaceWrite_AddDirWhenStateDirOutsideCwd` は、leader と implementer の両方で `<sandbox> [--add-dir <dir>] --model <m> <pointer>` の並び全体を `slices.Equal` で比べる。leader と implementer はどちらも埋め込み template があり(`internal/org/prompts/leader.md` と `implementer.md` が 4.9k と 5.9k)、プロンプトファイル経由になるので、末尾が pointer になるという fixture のコメントは正しい。`promptFilePath` は `<state-dir>/prompts/<org>_<seat>.md` で、fixture が組む pointer と一致する。
- recipe の「The procedure below spawns reviewer seats」は、手順に `--role reviewer` の spawn(`ralph org spawn --org-id perm-auto --id reviewer --role reviewer ...`)があることで裏付けられる。この文は修正前の「scratch の state dir が cwd の外にあると child の引数に `--add-dir` が出る」を置き換えたもので、置き換えなければ偽になる箇所だった。
- `git diff origin/main...HEAD` から、`TODO`、`FIXME`、`fmt.Print`、`println`、`/Users/<実名>`、ユーザー名の混入を grep した。ヒットなし。
- 前回レビューで読んだ `statedir.go`、`org_legacy_ledger.go`、`org.go`、`status.go`、`insights.go` の差分は、修正 commit で変わっていない部分も含めて読み直し、新しい指摘はなかった。`gitMainWorktree` の `--path-format=absolute` は git 2.31 以降の機能だが、`internal/cli/git_hooks.go:213` と `scripts/secret-scan.sh` がすでに同じ下限に依存している。古い git では出力が 2 行にならず(`len(dirs) != 2`)、`git-toplevel` 層に落ちる。
- `docs/tech-debt/README.md` を diff と両方向で突き合わせた。M-1 は直ったので、前回の「先送りする場合の候補行」は不要。L-5 は上の表のとおり。

## Findings

<!-- Area recommended values: naming, readability, unnecessary-change, typo,
     null-safety, debug-code, secrets, exception-handling, security, maintainability -->

| Severity | Area | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- |
| LOW (L-5、前回から継続) | maintainability | `docs/tech-debt/README.md:144` の open 行は、state dir の順を「`--state-dir`、`RALPH_ORG_STATE_DIR`、git の toplevel、cwd」と書き、再点検の trigger に「`ResolveOrgStateDir` の変更」を挙げている。この PR がその trigger を満たしたが、行は更新されていない。linked worktree では config が worktree 側の `./ralph.toml`、台帳が main 側になり、行が述べる「config と state dir で解決規則が違う」ずれが worktree をまたぐ形に広がった。plan は register の扱いを `/sync-docs` に回している | `docs/tech-debt/README.md:144`。plan の進捗の最終行 2 つに「`docs/tech-debt/README.md` の現役の行(State dir の順)は /sync-docs で扱う」とある | `/sync-docs` で、state dir の順を新しい順に直し、worktree をまたぐずれ(config は worktree の `ralph.toml`、台帳は main)を 1 文足す。見落としを防ぐための記録 |
| LOW (L-6、新規) | readability | 修正で足した文に、前後と折り返しの幅が合わない行が 2 種類ある(コード 1 行、recipe は 2 面で同じ 1 行)。前回の L-2 と同じ型で、gofmt と vet は検出しない。(a) `permissions.go:155` は 93 桁で、前後のコメント行は 73〜76 桁。(b) `docs/recipes/codex-seat-permissions.md:92` は 89 桁で、前後は 72〜77 桁。`templates/base/` 側のコピーも同じ行 | (a)「`// report to the leader over agmsg. Making the state dir writable for them would let any seat`」。この行の途中までを前の行に詰めて挿入したため、行が折り返されないまま残った。(b)「`the manifest, the receipts, and other seats' prompt files. Guarded codex seats and claude`」。同じ編集で、前の段落の末尾と「Guarded codex seats and claude seats get no `--add-dir` either.」の頭がつながっている。`awk '{print length}'` で各行の桁数を測った | (a) は 3 行に折り直す。(b) は「Only the leader gets it: ...」の段落と、「Guarded codex seats ...」から始まる段落を、前後と同じ 75 桁前後で折り直す。recipe は root と `templates/base/` の 2 面を同時に直す(`scripts/check-sync.sh` が一致を見る) |

## Positive notes

- M-1 の修正は、判定を `codexWritableRootArgs` の先頭 1 か所に置き、呼び出し元は `p.Role` を渡すだけにしてある。役割の判定が呼び出し側の `if` に散らばらないので、新しい呼び出し元が増えても同じ制約を受ける。
- テストが役割の軸を 2 段で持つ。引数の組み立ての単体テストが 6 つの非 leader ケース(implementer、reviewer の autonomous と edits、未知の役割、空の役割)を持ち、spawn のテストが implementer の起動引数全体を並びごと比べる。判定を削ると前者が、呼び出し側が役割を渡し損ねると後者が落ちる。
- 権限を絞る理由が、コメント、recipe、`/org` skill のいずれにも同じ言い方で書かれている(manifest、model receipts、他座席の role prompt を書き換えられる)。
- recipe の再確認の手順が reviewer 座席を spawn することを踏まえて、手順側の説明(child の引数に `--add-dir` が出ない)を直してある。コードだけを直して文書の例を残す、という取りこぼしがない。
- evidence に実行したコマンドと出力(rc=0 でも書けたかは終了コードで分からない、という注意まで)が残り、recipe から辿れる。
- 失敗の倒し方が一貫している。`gitMainWorktree` は git のどの失敗でも旧 tier に落ち、`LegacyWorktreeStateDir` は manifest が読めないときに書き換える動詞を止め、読むだけの動詞は注意を出して続ける。`guardLegacyOrgStateDir` は拒否と注意だけで書き込みを持たず、拒否は manifest、herdr、agmsg に触る前に返る。

## Tech debt identified

| Debt item | Impact | Why deferred | Trigger to pay down | Related plan/report |
| --- | --- | --- | --- | --- |
| (新しい行はない) | - | - | - | - |

_(今回の commit では `docs/tech-debt/README.md` を変更していない。L-5 の更新は `/sync-docs` で行う。)_

## Recommendation

- Merge: 可。CRITICAL、HIGH、MEDIUM はない。LOW は 2 件(L-5 が継続、L-6 が新規)で、どちらも止める理由にならない。前回の M-1 は解決し、L-1、L-3、L-4 も解決した。L-2 は前回の 2 か所が直り、同じ型が L-6 として再発している。
- Follow-ups:
  - L-6: コメントと recipe の 3 行(recipe は 2 面)を折り直す。コードの変更はないので、`/verify` に進む前に 1 commit にまとめるか、`/sync-docs` の commit に含めてよい。
  - L-5: `/sync-docs` で扱う(plan が回している)。
