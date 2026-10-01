# Verify report: changed-languages-merge-base

- Date: 2026-10-01
- Plan: docs/plans/active/2026-10-01-changed-languages-merge-base.md(issue #190)
- Verifier: verifier subagent (Claude Code)、cycle 1(pipeline cap 2 のうち 1 回目)
- Scope: `git diff main...HEAD`(HEAD `9a6d21b2`、base `main` `a3adc13b`)。`scripts/detect-changed-languages.sh` と template の複製、`tests/test-detect-changed-languages.sh`、`tests/test-run-verify-scope.sh`、`docs/quality/quality-gates.md`(root と template)。仕様への適合(AC-1〜AC-8)、静的解析、文書のずれを見た。動作テストの判定は /test の担当で、ここでは行っていない。scratch で走らせた 2 本のテストは AC-6 の mutation の基準線であり、/test の判定の代わりではない

## Deterministic checks run

| Command | Result | Notes |
| --- | --- | --- |
| `RALPH_VERIFY_SCOPE=full ./scripts/run-static-verify.sh`(前景) | PASS | exit 0。`Language scope: full`、`Language packs selected: golang`、gofmt ok、`0 issues.`、`All verifiers passed.`。check-sync は IDENTICAL 159 / DRIFTED 0 / ROOT_ONLY 0 / KNOWN_DIFF 5、check-pipeline-sync と check-skill-sync(13 件)と check-template-purity が OK、branch secret scan は `a3adc13b..9a6d21b2` が clean。log: `docs/evidence/verify-2026-10-01-084709.log` |
| `./scripts/run-static-verify.sh`(既定の scope、観測) | PASS | exit 0。`Requested scope: changed`、`Language scope: full fallback (shared:scripts/detect-changed-languages.sh)`、`Language packs selected: golang`。この branch が検出器そのものを変えるので、変更後の検出器は upstream == HEAD の push 済みの状態でも `no_changes` にならず full になる。log: `docs/evidence/verify-2026-10-01-084738.log` |
| `./scripts/detect-changed-languages.sh`(worktree、既定) | 観測 | `scope=full`、`reason=shared:scripts/detect-changed-languages.sh`、`docs_only=false`。`git rev-parse HEAD '@{upstream}'` は両方 `9a6d21b2`(upstream == HEAD)、`refs/remotes/origin/HEAD` は `refs/remotes/origin/main`。旧版なら同じ状態で `no_changes` になる状態 |
| `shellcheck -S warning scripts/detect-changed-languages.sh tests/test-detect-changed-languages.sh tests/test-run-verify-scope.sh` | PASS | exit 0、指摘なし |
| `sh -n`(検出器と 2 本のテスト)、`dash -n`(検出器) | PASS | 構文エラーなし |
| `cmp scripts/detect-changed-languages.sh templates/base/scripts/detect-changed-languages.sh` | PASS | byte 一致。4 ファイル(検出器 2、テスト 2)とも mode 755 |
| `./scripts/check-sync.sh`(単体) | PASS | `PASS: all files in sync.`、DRIFTED 0 |
| `./scripts/check-template-purity.sh`(単体) | PASS | `PASS: no meta-repo-specific references found in templates.` |
| `./scripts/secret-scan-branch.sh --strict` | PASS | exit 0。`a3adc13b..9a6d21b2 against origin/main: clean` |
| 非 ASCII と U+FFFD の確認(検出器、2 本のテスト、`quality-gates.md` の追加 4 行) | PASS | 検出器と 2 本のテストは非 ASCII 0 行。`quality-gates.md` の非 ASCII 9 行は既存の行(em dash 等)で、追加 4 行は ASCII のみ。U+FFFD は変更範囲に 0 件 |
| `git status --porcelain`(worktree) | PASS | 空(`docs/evidence/*.log` は gitignore) |

## Observational checks

### AC-by-AC

| AC | Verdict | Evidence |
| --- | --- | --- |
| AC-1 push 済みの branch で `golang` | PASS | scratch の probe P1(bare の remote に main を push、feature に go.mod の commit を push、HEAD == upstream == `9fa74ae7`、`origin/HEAD` なし): 新版は `scope=changed reason=changed_languages languages=golang golang_roots=.`、`git show main:scripts/detect-changed-languages.sh` で取った旧版は `reason=no_changes languages=`(#190 の不具合を再現)。dash でも同じ。テスト: `tests/test-detect-changed-languages.sh:278`(ケース 10)が `assert_upstream_is_head` で fixture が #190 の状態であることを先に確かめてから `scope`、`docs_only`、`languages` を見る |
| AC-2 明示した base は従来どおり | PASS | probe P1 で `RALPH_VERIFY_BASE=origin/feature`(== HEAD)は `reason=no_changes`。テスト: `:294`(ケース 11)、`:301`(ケース 12、存在しない明示の base は `no_merge_base:nope` の full)。既存のケース 1〜9 は `git diff main...HEAD` で追加と冒頭の `unset`、hermetic 化のブロックだけの差分で、アサーションは変わっていない。scratch の基準線で 60 / 60 が通る |
| AC-3 `trunk` と存在しない ref | PASS | テスト: `:307`(ケース 13、`origin/HEAD` → `origin/trunk`、main も master もない)、`:322`(ケース 14、`origin/HEAD` → 存在しない `origin/gone` で `origin/main` に落ちる)。検出器: `scripts/detect-changed-languages.sh:139-146`(HEAD の指す先を `refs/remotes/<remote>/` の下かつ実在のときだけ使う)。mutation (b) と (c) がそれぞれ該当ケースを落とす(下の AC-6) |
| AC-4 default branch 上で変更なし | PASS | probe P3(clean な main、push 済み): `reason=no_changes languages=`。テスト: `:336`(ケース 15)。`:345`(ケース 16)は未 push の commit があれば `golang` になる対照 |
| AC-5 `run-static-verify.sh` の既定が golang の pack を呼ぶ | PASS | `tests/test-run-verify-scope.sh:224-226`: push 済み feature(`assert` で upstream == HEAD を確認済み)で `local:static:changed` と `golang:static:changed:service` が呼ばれ、`python:static:changed:` は呼ばれない。mutation (a) で `golang:static:changed:service` の呼び出しが消えて落ちる。実運用の観測は上の「既定の scope」の行(この branch 自身は検出器を変えるので full) |
| AC-6 mutation | PASS | 独立に再現した(下の mutation 表)。plan が主張する 4 種(a)〜(d)が、plan の言う該当ケースを落とす。self-review の M-1 と L-1 に対応する 2 種(e)(f)も落とす |
| AC-7 byte 一致と静的解析 | PASS(範囲に注記) | 検出器は `cmp` で byte 一致。`quality-gates.md` はファイル全体では root と template が以前から異なる(`scripts/check-sync.sh:101` の KNOWN_DIFF、main でも差分 5 行で、HEAD でも同じ 5 行で行番号だけ 4 行ずれた)。追加した hunk 4 行は root と template で同一(`diff` で確認)。shellcheck と check-sync は green。plan の AC-7 にある `RALPH_VERIFY_SCOPE=full ./scripts/run-verify.sh`(テストも含む集約)のうち、静的側は全 green で、テスト側は /test の担当 |
| AC-8 `central` だけの remote | PASS | probe P2(remote は `central` だけ、main が `central/main` を追跡、main の上に未 push の go.mod の commit、`central/main..HEAD` が 1 commit): 新版は `golang`。テスト: `:353`(ケース 17)、`tests/test-run-verify-scope.sh:238-240`(`run-static-verify.sh` が `golang:static:changed:service` を呼ぶ)。mutation (d) が検出器側の 2 アサーションと scope 側の 2 アサーションを落とす |

補足: probe P2 では旧版(upstream 経由)も `golang` を返す。plan の AC-8 は、下書きの plan(upstream を使わず `origin` 前提にする案)の反例で、旧実装の不具合ではない。`central` のケースは、今回の候補順で追跡先の remote の段を落とさないための回帰を守っている。

### mutation の再現(scratch、worktree は変更していない)

検出器と `tests/` の 2 本を、`scripts/`(run-verify、run-static-verify、run-test、検出器)と `tests/` の形で scratch に複製し、1 回に 1 箇所だけ置換して `sh` で 2 本のテストを走らせた。基準線(置換なし)は検出器 60 / 60、scope 19 / 19。

| mutation | 検出器のテスト | scope のテスト | plan の主張との一致 |
| --- | --- | --- | --- |
| (a) `@{upstream}` を最初に見る | 52 / 60。`pushed branch is not docs-only`、`pushed branch selects golang`、`trunk default branch selects golang`、`dangling origin/HEAD falls through to origin/main`、`unfetched ... records no_remote_default`(2)、`unfetched remote on a feature branch`(2)が落ちる | 18 / 19。`pushed branch static wrapper runs golang pack for committed change` が落ちる | 一致(AC-1 と AC-5 が落ちる。`trunk` と存在しない ref も fixture が push 済みなので落ちる、と Deviation notes にもある) |
| (b) `origin/HEAD` の段を外す | 58 / 60。`trunk default branch uses changed scope`、`trunk default branch selects golang` | 19 / 19 | 一致 |
| (c) 実在の確認を外す | 58 / 60。`dangling origin/HEAD uses changed scope`、`dangling origin/HEAD falls through to origin/main` | 19 / 19 | 一致 |
| (d) 追跡先の remote の段を外す | 58 / 60。`central-only remote uses changed scope`、`central-only remote selects golang for unpushed commit` | 17 / 19。`central-only remote static wrapper runs golang pack for unpushed commit`、`... skips unrelated python pack` | 一致(検出器と scope の両方) |
| (e) merge-base に短い名前を渡す(self-review M-1 の旧状態) | 58 / 60。`branch named origin/main does not hide golang`、`tag named main does not hide golang` | 19 / 19 | 一致(Slice B の「21 と 22 が落ちる」) |
| (f) L-1 の guard を外す | 56 / 60。`unfetched tracked remote falls back to full`、`... records no_remote_default`、`unfetched origin falls back to full`、`... records no_remote_default` | 19 / 19 | 一致(Slice B の「23 と 24 が落ちる」) |

### self-review の指摘 6 件が HEAD で直っているか

| # | 重大度 | 状態 | 根拠 |
| --- | --- | --- | --- |
| M-1 | MEDIUM | 修正済み | `remote_default_ref` は完全な ref 名を返す(`scripts/detect-changed-languages.sh:143`、`:150`)。ローカルの段は `refs/heads/main` / `refs/heads/master` を入れる(`:177`、`:179`)。`git merge-base HEAD "$base_ref"`(`:192`)には既定では完全な名前が渡る。`RALPH_VERIFY_BASE` は利用者の指定のためそのまま。再現: probe A(`feature` を `--track origin/main` で作り、ローカルの branch `origin/main` を先端に置く。git は `refname 'origin/main' is ambiguous` を出す)は新版が `golang`。probe G(remote の名前が `heads`)も `golang`。テスト `:401`(ケース 21)、`:416`(ケース 22)があり、短い名前が HEAD に解決される fixture であることを `assert_short_name_shadowed` で確かめる。mutation (e) で落ちる |
| L-1 | LOW | 修正済み(範囲を絞った形) | `:182` の `no_remote_default:$tracked_remote`。条件は、今の branch が `.` 以外の remote を追跡している(`:163-168`)、origin にも追跡先の remote にも default branch の ref がない(`:170-173`)、ローカルの段の候補が今の branch そのもの(`:181`)。probe E(未 fetch の `central` を main が追跡、main の上に未 push の go.mod): 新版は `scope=full reason=no_remote_default:central`、旧版は `no_merge_base:@{upstream}` の full。テスト `:430`、`:444`、`:456`(ケース 23、24、25)。ケース 25 は feature branch ではローカルの main が別の branch なので changed のまま。mutation (f) で落ちる。残る場合の扱いは下の V-1 |
| L-2 | LOW | 修正済み | `:24` は「never consulted」に変わった。`:134` と先頭のコメント(`:17-20`)に、HEAD の指す先が `refs/remotes/<remote>/` の下の実在の ref のときだけ使う条件がある。probe D の形(外を指す HEAD)は本体の `case` の条件(`:140`)と一致している |
| L-3 | LOW | 修正済み | `docs/quality/quality-gates.md:43` と template の同じ行が「the merge-base of HEAD and `RALPH_VERIFY_BASE` when set, otherwise of HEAD and the default branch」となり、明示した base も merge-base を通ることが読み取れる。コードの `:192` と一致 |
| L-4 | LOW | 修正済み | 2 本のテストが `HOME`、`GIT_CONFIG_GLOBAL`、`GIT_CONFIG_SYSTEM=/dev/null`、`GIT_CONFIG_NOSYSTEM=1`、`GIT_TERMINAL_PROMPT=0` を `trap cleanup` の直後に固定している(`tests/test-detect-changed-languages.sh:87-95`、`tests/test-run-verify-scope.sh:55-63`)。`$workdir` の下の HOME なので後始末は既存の trap に入る。冒頭の `unset RALPH_VERIFY_BASE`(`:14`、scope 側は `:8` で `RALPH_VERIFY_SCOPE` も) |
| L-5 | LOW | 修正済み | ケース 18(`:362`)の fixture は、`central/main` に main より古い commit だけを置き、main の未 push の commit を python にしている。結果が `golang` だけなら、ローカルの main が base だったと分かる。ケース 19(`:380`)は名前を「still selects golang」に直した。plan の Deviation notes(Slice B)の記述と一致 |

### 文書のずれ

| 対象 | 結果 |
| --- | --- |
| `docs/quality/quality-gates.md:43-46` と `templates/base/docs/quality/quality-gates.md:43-46` | コードと一致。基準の順(`RALPH_VERIFY_BASE` → origin の default branch → 追跡先の remote の default branch → ローカルの main / master)と、merge-base 経由であること、未コミットと未追跡のファイルを足すことが、検出器の挙動と合う。追加 4 行の hunk は root と template で同一 |
| `.claude/agents/verifier.md:21`、`.claude/agents/tester.md:19-20`、`.claude/skills/verify/SKILL.md:13`、`.claude/skills/test/SKILL.md:11`(`.agents/skills/` と `templates/base/` の写しも) | 「changed-language scope by default」と `RALPH_VERIFY_SCOPE=full` の説明だけで、diff の base を述べていない。ずれなし |
| `docs/recipes/adding-a-language-pack.md:39,50,79`、`templates/base/docs/recipes/adding-a-language-pack.md:52-53` | 検出器へのパターンの足し方と複製の同期だけで、base の選び方には触れていない。ずれなし |
| `docs/quality/definition-of-done.md:12-13`(と template) | 「changed-language scope by default」だけ。ずれなし |
| `@{upstream}` と `RALPH_VERIFY_BASE` の言及(`git grep`、plan・reports・insights・tests を除く) | 検出器の先頭コメント、`quality-gates.md:43`、`docs/tech-debt/README.md:56-57` のみ。後者は `detect_base_branch`(cross-review の helper)の解決済みの行で、この検出器とは別物。ずれなし |
| `run-verify.sh` が `reason` を読む箇所(`scripts/run-verify.sh:63-98`) | 表示だけで、`no_remote_default:<remote>` の新しい値を分岐に使う読み手はない。既存の `shared:<file>`、`no_merge_base:<ref>` と同じ形 |
| plan の進捗(`## Progress checklist` と AC のチェックボックス) | AC-1〜AC-8 のボックスと「Verification artifact created」は未チェックのまま。pipeline の途中なので想定内で、verify の fail の理由にはしない。文書の遅れとして記録する |

### 指摘

| # | 重大度 | 内容 | 根拠 | 推奨 |
| --- | --- | --- | --- | --- |
| V-1 | LOW(観測、仕様違反ではない) | remote のない repo で main の上に commit した場合は、これまでどおり `no_changes` を返す。base がローカルの main になり、HEAD と同じ commit になるため、commit 済みの変更が見えない。self-review の L-1 は「少なくとも先頭のコメントに書く」を推奨し、Slice B は挙動の修正(追跡先がある場合の full)で閉じたが、remote のない場合の記述は検出器の先頭コメントにも `quality-gates.md` にもない | probe F(remote なし、main の上に go.mod の commit、作業ツリーは clean): 新版・旧版とも `reason=no_changes languages=`。`quality-gates.md:43-46` は「else local main/master」と書くだけで、HEAD がそのブランチにいる場合の帰結は読み取れない。plan の Deviation notes(self-review)は「remote のない repo の main(ケース 1〜9)は変わらない」と記録している | マージを止める理由はない。1 文を検出器の先頭コメントに足すか、`docs/tech-debt/README.md` に 1 行足す。どちらも別の変更で足りる |

CRITICAL 0 / HIGH 0 / MEDIUM 0 / LOW 1。

## Coverage gaps

- 動作テストの判定(sh と dash での 60 / 60、19 / 19、`RALPH_VERIFY_SCOPE=full ./scripts/run-verify.sh` のテスト側)は /test の担当で、ここでは判定していない。scratch の基準線として sh で 60 / 60 と 19 / 19 を得たのは mutation の比較のため
- push 済みの branch に対する `/verify` の既定の scope が、この branch では full に倒れるため(検出器自身が変更対象)、`changed_languages` の選択を既定の wrapper で実機の観測として見る機会がない。代わりに AC-1 と AC-8 を scratch の fixture と、`tests/test-run-verify-scope.sh` の end-to-end で確かめた。検出器を変えない次の branch で最初に観測できる
- 古い git(2.8 系など)と Linux の dash では確かめていない。macOS の git 2.49.0 と `/bin/sh`、`dash` で確認した。使っている `git symbolic-ref --quiet`、`git show-ref --verify --quiet`、`git config --get` は古くからあるコマンド。未確認です
- 積み重ねた branch で下位 branch の変更も対象に入る点(plan の Non-goals)は、挙動として確認していない。差分が広がる方向なので検査が漏れる側の後退ではない
- `docs/tech-debt/README.md` に #190 の行がないことは確認した。V-1 を行にするかどうかは判断が残る

## Verdict

- Verified: AC-1、AC-2、AC-3、AC-4、AC-5、AC-6、AC-8(コードの読み、scratch の probe、mutation の独立再現、テストの存在)。AC-7 の静的側(byte 一致、shellcheck、check-sync、check-template-purity、フル scope の静的 verify)。self-review の M-1 と L-1〜L-5 の 6 件すべてが HEAD で修正済み(コードの読みと、M-1・L-1 は scratch の probe と mutation)。文書は `quality-gates.md` が挙動と一致し、ほかの名指しのファイルに base の選び方についてのずれはない
- Partially verified: AC-7 の `RALPH_VERIFY_SCOPE=full ./scripts/run-verify.sh` のうちテスト側(/test の担当)。`quality-gates.md` の byte 一致は追加 hunk の同一性まで(ファイル全体は以前からの KNOWN_DIFF)
- Not verified: 古い git と Linux の dash での実行、積み重ねた branch の挙動、この branch の後に検出器を変えない状態での既定 wrapper の実機の観測
- 判定: **pass**(CRITICAL 0 / HIGH 0 / MEDIUM 0 / LOW 1。V-1 は仕様違反でも後退でもなく、マージを止めない)
