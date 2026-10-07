# Self-review report: org-state-dir-common

- Date: 2026-10-07
- Plan: docs/plans/active/2026-10-07-org-state-dir-common.md
- Branch: fix/org-state-dir-common(origin/main を 8018e4bf で取り込み済み、HEAD 8c1ac257)
- Reviewer: reviewer subagent (Claude)、cycle 1
- Scope: diff の品質だけを見た(命名、読みやすさ、不要な変更、コメントの正確さ、null と境界の扱い、エラー処理、安全性)。対象は `git diff origin/main...HEAD`(19 ファイル、+1627/-95、plan 自身を除くと 18 ファイル)。`.agents/skills/` と `templates/base/` はコピーなので、root 側を読み、コピーとの差は `diff` で見た。仕様への適合、テストの網羅、文書のずれは見ていない(`/verify`、`/test`、`/sync-docs` の担当)。リポジトリのテスト、linter、vet は実行していない。

## Evidence reviewed

- `internal/org/statedir.go` の差分を全行読んだ。`gitMainWorktree` の分岐を、main の中(git dir と common dir が同じ)、linked worktree の中、bare、`--separate-git-dir`、git の失敗の順に追った。どの失敗も `ok=false` になって `git-toplevel` 層へ落ちるので、旧挙動より悪くならない。`parseMainWorktreeRecord` は最初の空行で読むのをやめ、bare、`worktree` 行なし、相対パスを弾く。
- `LegacyWorktreeStateDir` の返り値を、source が `git-main-worktree` 以外、manifest なし、stat 失敗、main のチェックアウト(`samePath`)、manifest が読めない場合について読んだ。`ErrNotExist` 以外の stat 失敗は error になり、書き換える動詞は止まる側に倒れる。
- `internal/cli/org_legacy_ledger.go` と、`org.go`、`status.go`、`insights.go` の呼び出しを読んだ。`ResolveOrgStateDir` と `newOrgRuntime(` の呼び出しを repo 全体で grep し、`.go` / `.sh` / `.json` / `.toml` に未対応の呼び出し元が残っていないことを確かめた(`newOrgRuntime` は 9 か所すべてが `access` を渡している。`insights --receipts` を明示したときは台帳を解決しないので、注意を出さないのは正しい)。
- `internal/org/permissions.go` の `codexWritableRootArgs` と `spawn.go:761` の呼び出しを読んだ。`withManifestLock` が `os.MkdirAll` で台帳のディレクトリを先に作る(`lockfile.go:51`)ので、`--add-dir` の判定時にはディレクトリが存在する。
- 存在しないパスでの `filepath.EvalSymlinks` の挙動を scratchpad の Go で確かめた(L-3)。`/var/folders` は `/private/var/folders` に解決され、存在しないパスは error で空文字を返す。
- テストを読んだ(`statedir_test.go`、`org_legacy_ledger_test.go`、`permissions_test.go`、`spawn_test.go`)。git 環境の隔離(`GIT_CONFIG_GLOBAL`、`GIT_DIR` 系の unset)、`t.Chdir`、symlink 解決済みパスでの比較、manifest が「ディレクトリ」のときの読み取り失敗は、root でも通る作りになっている。デバッグ出力、TODO、固定の認証情報は diff にない。
- `docs/` の変更は、typo と実装との食い違い(コメントや文言が実際のコードを正しく述べているか)だけを見た。`/org` skill の 4 面は `diff` で完全に一致し、recipe の 2 面も一致した。recipe の `--add-dir` の位置の説明は `spawn.go:761` の並び(sandbox フラグ、`--add-dir`、`--model`)と合う。
- `docs/tech-debt/README.md` を diff と両方向で突き合わせた(L-5)。

## Findings

<!-- Area recommended values: naming, readability, unnecessary-change, typo,
     null-safety, debug-code, secrets, exception-handling, security, maintainability -->

| Severity | Area | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- |
| MEDIUM (M-1) | security | `--add-dir <台帳のディレクトリ>` が、役割に関係なく workspace-write の codex 座席すべてに付く。台帳を書く必要があるのは leader だけだと plan 自身が前提に置いているので、implementer と reviewer の座席にも、全 org の manifest、receipts、他座席の role prompt への書き込み権限が広がる。 | `spawn.go:761` は `codexWritableRootArgs(p.Driver, resolvedPermMode, p.Cwd, stateDir)` を呼び、`p` の Role を渡していない。plan の Assumptions は「台帳を書き換えるのは leader 座席と人だけで、implementer と reviewer の座席は agmsg で送る。`--add-dir` が効くのは、主に codex で動く headless leader」と書く。state dir には `manifest.jsonl`、`model-receipts.jsonl` のほか、`<state-dir>/prompts/<org_id>_<seat_id>.md`(`spawn.go:748` のコメント)が入る。`autonomous` は `--ask-for-approval never` なので、書き込みに承認の確認が入らない。leader の起動は `Role: org.LeaderIdentity`(`org.go` の `start`)で、役割で見分ける手段はある。AC8 は役割を限定していないので、spec 違反ではなく設計の選択。 | 付ける条件に「役割が leader」を足す(`p.Role == LeaderIdentity`)。または、全座席に付けると決めるなら、その代償(どの座席も台帳を書き換えられる)を recipe と `/org` skill の該当箇条に 1 文で書き、tech-debt に行を 1 つ足す。どちらにするかは判断が要る。 |
| LOW (L-1) | readability | コメントの tier 名が、番号の振り直しと実装の変更に追いついていない箇所が 2 つある。(a) `statedir.go:45`「`this is the same path tier 4 used to return`」。tier 4 は今も `git-toplevel` として動いているので、「used to return」は「今は返さない」と読める。(b) `statedir_test.go:36`、`initGitRepo` のコメントが「ResolveOrgStateDir's git-toplevel tier」のまま。この helper が作るリポジトリは今は `git-main-worktree` 層で解決される。 | `statedir.go:44-46`、`statedir_test.go:36-37`。(b) の helper は diff に含まれていないが、直したテストの前提になっている。 | (a) は「main のチェックアウトでは、`git-toplevel` 層が返していたのと同じパスで、tag だけが違う」のように番号を使わず書く。(b) は「git-main-worktree 層」に直す。 |
| LOW (L-2) | readability | 書き直したコメントの中に、折り返しが残った行が 2 つある。(a) `org.go:178` は 82 桁で、前後は 75 桁前後に折り直されている。(b) `status.go:212` は 92 桁。gofmt と vet は検出しない。 | `org.go:178`「`// default manifest. An env-resolved or default-resolved state dir is deliberately`」、`status.go:212`「`// every production call site, so an operator debugging "why did \`ralph status\` read from an`」。いずれも `git diff` で追加行として出ており、同じブロックの前後の行は折り直されている。 | 前後と同じ幅に折り直す。 |
| LOW (L-3) | maintainability | `codexWritableRootArgs` のコメントは「symlink を解決できるパスは解決して比べる(macOS の `/var` は `/private/var`)」と言うが、`resolvedOrClean` は全か無かで、パスの途中までしか存在しないと何も解決しない。cwd が `/var/...`(解決すると `/private/var/...`)で、台帳のディレクトリがまだないとき、解決済みの cwd と未解決の台帳パスを比べて「外」と判定し、不要な `--add-dir` を足す。 | scratchpad の Go で確認: `EvalSymlinks("/var/folders")` は `/private/var/folders`、`EvalSymlinks("/var/folders/zz-nonexistent/.harness/state/org")` は `lstat /private/var/folders/zz-nonexistent: no such file or directory` で空文字。現状は `spawn.go:498` の `withManifestLock` が先にディレクトリを作る(`lockfile.go:51`)ので届かない。テスト(`permissions_test.go`)は台帳のディレクトリをすべて作ってから呼んでいる。害は、すでに書ける場所を二重に書き込み可にするだけ。 | コメントに「台帳のディレクトリが存在すること」を前提として書くか、存在する最も深い祖先まで解決する形にする。前者で十分。 |
| LOW (L-4) | maintainability | recipe の「Checked on 2026-10-07 with codex-cli 0.160.0 on macOS」に対応する `docs/evidence/` のファイルがない。同じ recipe の前段の確認(2026-09-18)は `docs/evidence/codex-seat-permissions-2026-09-18.md` に残っている。今回の確認は plan の進捗の 1 行にしかない。 | `docs/recipes/codex-seat-permissions.md:96` の「Checked on」と、同ファイル 12 行目の evidence への参照。`ls docs/evidence` に 2026-10-07 の codex 関連ファイルはない。plan は `/pr` で archive されるので、記録は残るが、読者が recipe から辿れない。 | `docs/evidence/codex-add-dir-2026-10-07.md` に実行したコマンドと出力を残して recipe から参照する(`$HOME` は `~` に置き換える)。または「plan の進捗に記録」と書く程度に主張を弱める。 |
| LOW (L-5) | maintainability | `docs/tech-debt/README.md:144` の open 行は、state dir の順を「`--state-dir`、`RALPH_ORG_STATE_DIR`、git の toplevel、cwd」と書き、再点検の trigger に「`ResolveOrgStateDir` の変更」を挙げている。この PR がその trigger を満たしたが、行は更新されていない。さらに、linked worktree では config が worktree 側の `./ralph.toml`、台帳が main 側になり、行が述べる「config と state dir で解決規則が違う」ずれが worktree をまたぐ形に広がった。plan は register の扱いを `/sync-docs` に回している。 | `docs/tech-debt/README.md:144`。plan の進捗の最終行に「`docs/tech-debt/README.md` の現役の行(State dir の順)は /sync-docs で扱う」とある。 | `/sync-docs` で、state dir の順を新しい順に直し、worktree をまたぐずれ(config は worktree の `ralph.toml`、台帳は main)を 1 文足す。ここは見落としを防ぐための記録。 |

## Positive notes

- 失敗の倒し方が一貫している。`gitMainWorktree` は git のどの失敗でも旧 tier に落ち、`LegacyWorktreeStateDir` は manifest が読めないときに書き換える動詞を止め、読むだけの動詞は注意を出して続ける。
- 台帳を移さない、混ぜない方針がコードに現れている。`guardLegacyOrgStateDir` は拒否と注意だけで、書き込みを持たない。拒否は manifest、herdr、agmsg に触る前に返り、テストが 3 つとも呼び出し回数 0 で確かめている。
- `shellQuoteIfNeeded` を再利用して、空白を含むパスでも案内のコマンドがそのまま貼れる。
- `ralph insights` は `--state-dir` を持たないことを `cmd.Flags().Lookup("state-dir")` で判定して、案内を `RALPH_ORG_STATE_DIR` に切り替えている。
- 旧コメントの `docs/plans/active/2026-08-02-org-runtime-watchdog.md` を `archive/` に直した変更は、ファイルの実在を確かめた(`docs/plans/archive/2026-08-02-org-runtime-watchdog.md`)。この PR が触るコメントの中なので、不要な変更にはあたらない。
- テストの fixture が、root 権限や macOS の `/var` symlink に依存せず、manifest の読み取り失敗を「ディレクトリ」で作っている。

## Tech debt identified

| Debt item | Impact | Why deferred | Trigger to pay down | Related plan/report |
| --- | --- | --- | --- | --- |
| (候補、M-1 を直さずに先送りする場合のみ)`--add-dir <台帳>` が workspace-write の codex 座席すべてに付く。leader 以外の座席にも台帳への書き込み権限が広がる | prompt injection を受けた codex 座席が、manifest、receipts、他座席の role prompt を書き換えられる | plan の AC8 が役割を限定していない。実機の leader は codex で動くのが主だが、役割ごとの境界は未検証 | leader 以外の codex 座席で台帳への書き込みが必要になる、または座席への prompt injection の報告 | `docs/plans/active/2026-10-07-org-state-dir-common.md`(AC8、Assumptions) |

_(M-1 を直す場合は行を足さない。先送りする場合だけ、上の候補行を `docs/tech-debt/README.md` に足す。今回の commit では README を変更していない。)_

## Recommendation

- Merge: 可。CRITICAL と HIGH はない。MEDIUM 1 件(M-1)は、`/verify` に進む前に方針(leader だけに付ける、または全座席に付けて代償を文書に書く)を決めたい。LOW 5 件(L-1 から L-5)は止める理由にならない。
- Follow-ups:
  - M-1: `--add-dir` を leader だけにするか、全座席に付けると決めて文書に代償を書くか。
  - L-1 から L-3 はコメントの修正で、1 commit にまとめられる。
  - L-4 は evidence の文書を足すか、recipe の主張を弱める。
  - L-5 は `/sync-docs` で扱う(plan が回している)。
