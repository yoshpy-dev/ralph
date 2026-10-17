# Verify report: org-auto-merge

- Date: 2026-10-18
- Plan: docs/plans/active/2026-10-17-org-auto-merge.md(`Status: Approved`。`./scripts/plan-visual.sh digest` は 305d97070f38 で、`- Approved:` の digest と同じ。仕様 `docs/specs/2026-10-07-org-multi-org-director.md` の FR-8 のうち 8a-3a)
- Verifier: verifier subagent (Claude)。パイプライン cycle 2 の verify(`.harness/state/standard-pipeline/cycle-count.json` の cycle は 2)。HEAD fd5e6c51
- Scope: `git diff 15d22b1a...HEAD`(59 ファイル、+8685/-318)。重点は cycle 1 の verify の HEAD b82f9dee からの差分 `b82f9dee..HEAD`(35 ファイル、+2309/-1121)で、sync-docs b5efa956、test の直し 29afd327、cross-review の直し S8(9506ec64、b2f0b534、320b4d75)、self-review cycle 2 の直し S9(e5831d14、7d0d08dd)と計画の更新を含む。`mergeorder.go`、`mergeguarantee.go`、`seat.go`、`reserve.go`、`internal/config/config.go`、`internal/cli/status.go`、`org_director.go`、`org.go` はこの差分で変わっていない(`internal/config/config_test.go` の注だけ)。コードと文書を読み、HEAD の binary を偽の gh と手元の bare repo で打った。テスト(`go test`、`./scripts/run-test.sh`)は走らせていない(/test の受け持ち)。gh、ネットワーク、herdr、osascript、tmux、端末の app には触れていない。行番号は HEAD fd5e6c51 のもの
- Evidence: wrapper のログ `docs/evidence/verify-2026-10-17-192645.log`(gitignore の対象。名前の日時は UTC)。今回の指示は report のほかのファイルを書かないことなので、cycle 1 の probe の記録 `docs/evidence/verify-2026-10-17-org-auto-merge.log` は書き換えていない。probe のスクリプト(`fake-gh.sh`、`probe2.sh`、cycle 1 の `probe.sh` の読み直しの fixture に `headRefOid` を足した写し)と出力は session の scratchpad に置き、要点を下の「Observational checks」に写した
- Cycle 1: 前の verify は c0ad3f37(PASS、LOW の V1・V2、情報 I1〜I7)。V1・V2 は sync-docs b5efa956 で直っていることを、この cycle で確かめた(「Findings」の下の情報 I3)

## Spec compliance

| Acceptance criterion | Status | Evidence |
| --- | --- | --- |
| AC1 merge-order の `pr_lookup` | 満たす(binary でも確認) | `mergeorder.go` はこの差分で変わっていない。cycle 1 の probe を HEAD の binary で打ち直し、`found`・`none`・`failed`・`skipped`(gh を PATH から外すと `gh_available: false`)が cycle 1 と同じに出た |
| AC2 `[org.director].auto_merge` | 満たす(binary でも確認) | 設定と表示のコードはこの差分で変わっていない。probe の config を打ち直し、main worktree の `auto_merge = true` が linked worktree の `false` に勝って `AUTO MERGE: true` と `auto_merge=true` の `pr_merged` になり、`"yes"` は終了コード 1 だった |
| AC3 断りの 14 の場合 | 満たす(binary でも確認) | cycle 1 の 25 件の断りを打ち直し、どれも終了コード 1、`gh pr merge` なし、`pr_merged` は 0 行。走っている org の断り(`runningRefusal`、`internal/org/merge.go:496-512`、呼び出しは `:268`)は、cycle 1 では binary で打っていなかったが、今回 probe の org-nowait で打った。gh は `pr list` の 1 本だけで、文は「受信箱の RESULT を処理し、org が disband してから打ち直す」を言う |
| AC4 対象の repository | 満たす(binary でも確認) | probe の happy で、動詞が打った gh の 10 本は `gh pr` がすべて `-R github.com/o/r`、`gh api` がすべて `--hostname github.com`、`gh repo view` は引数 `github.com/o/r`。読み直しも `gh pr view 7 -R github.com/o/r --json state,headRefOid,mergeCommit,autoMergeRequest` だった。foreign(別の host の PR、ローカルのパスの origin)は cycle 1 と同じく断った |
| AC5 保護 | 満たす(binary でも確認) | S8a: `enforceClassic`(`internal/org/mergeenforce.go:281-303`)は打つ本人が admin かどうかを受け取らず、毎回 `.../protection/enforce_admins` を読む。`readEnforcementRepository`(`:167-182`)は `permissions.admin` を読まない(Go のソースで `permissions.admin` が出るのは注の 2 か所 `:165`、`:278` と test の fixture だけ)。probe の classic で 6 通りを打ち、`enforce_admins` が `false` なら非 admin、admin、`permissions` なしのどれも断り、`true` ならどれも merge した。ruleset の `always`・`pull_requests_only`、GraphQL と rules の queue は cycle 1 と同じく断った。doctor 側の `mergeguarantee.go` はこの差分で変わっていない |
| AC6 通る道 | 満たす(binary でも確認) | probe の happy は `pr merge 7 --merge --match-head-commit <gate の head> -R github.com/o/r` を 1 回だけ打ち、`--auto`・`--admin`・`--delete-branch` はなかった。6 行を出して終了コード 0、`pr_merged` の Details は `branch=feat/a pr=7 head=<head> method=merge merge_commit=<sha> auto_merge=false`。method の squash も cycle 1 と同じ |
| AC7 読み直しで MERGED でない、別の head、中断 | 満たす(binary でも確認) | S8b: `mergeReadBackFields`(`internal/org/merge_run.go:36`)に `headRefOid` が入り、`mergeAndRecord`(`:331-361`)は MERGED で head が gate の head と違えば(`:339`)`otherHeadMerged`(`:390-400`)で Failed にする。`Merged` は立たず、CLI の `printMergeResult`(`internal/cli/org_merge.go:228-245`)は `Merged` のときだけ行を出す。probe の other-head(別の head)と no-head(`headRefOid` なし)は、stdout が空(`MERGED:` なし)、`pr_merged` 0 行、終了コード 1 で、stderr は merge された head(no-head では「with no head」)と、確かめる `gh pr view ... --json state,headRefOid,mergeCommit` と、merge-order の打ち直しを言った。予約の外し 2 通り、読み直しの失敗(確かめるコマンドに `headRefOid` が入った)、`gh pr merge` の途中の SIGINT(6 行と `NOTE:`、終了コード 1)は cycle 1 と同じ。中断のあとに別の head の MERGED を読んだ場合は、`mergeInterruptedAfterStart`(`:458-468`)が Failed の文を使うことをコードで読んだ(binary では打っていない) |
| AC8 `gh pr merge` の失敗 | 満たす(記録の失敗はコードで確認) | merge-fails-merged(gh が終了コード 1、読み直しが gate の head の MERGED)は記録して `NOTE:` を出し終了コード 0。merge-fails-open は記録なしで終了コード 1。fail-other-head(gh が `Head branch was modified` で終了コード 1、読み直しが別の head の MERGED)は、記録せず `MERGED:` も出さず、`; gh said:` に gh の末尾を添えて終了コード 1。記録の失敗は `recordMerge`(`:439-452`)を読んだ |
| AC9 `pr_merged` の扱い | 満たす(コードとテストのコードで確認) | `seat.go`、`reserve.go`、`verbs_all.go`、`watch.go` はこの差分で変わっていない。`TestPRMerged_IsNotASeatAndDoesNotRun`(`internal/org/merge_run_test.go:561`)は S9b の移動のあとも残っている(走らせていない) |
| AC10 ロックの共有 | 満たす(Spawn はコードで確認) | probe の lock を打ち直し、ロックを持たれている間の merge と、merge の `gh pr merge` の間の regate はどちらも断られた。検査の待ち(wait-pass)と org の待ち(wait-org)の間は、perl の flock が `regate.lock` を取れた。待つ間に外すことを言う注 3 か所(`internal/org/lockfile.go:136-138`、`internal/org/regate.go:95-97`、`internal/org/spawn.go:1326`)は、S9a で「`merge_wait.go` のすべての待ち」と書き、`merge_wait.go:37-40` がそれを受ける |
| AC11 `/director` skill | 満たす | `.claude/skills/director/SKILL.md:463-539`。false は `:463-465`、true の 4 段は `:466-477`、止め方は `:491-493`、`--wait-checks 60m` の打ち方と org の disband の待ちは `:505-526`、時間切れの断りと「director は disband を打たない」は `:528-533`。4 面は `cmp` で同じ。`TestDirectorSkill_PinsTheAutoMergeSteps`(`internal/org/director_skill_sync_test.go:123`)の 9 つの断片(S8c で 2 つ足した)は、空白を落とすと 4 面のどれにも順に入っていて、古い文はない(python で同じ規則を確かめた。テストは走らせていない) |
| AC12 文書 | 一部(LOW の V3・V4) | 下の「Documentation drift」。S8 の 3 つの変更は help、README、`/org` と `/director` の skill、仕様の注、`AGENTS.md`、tech-debt に入っている。ただし、gate の記録だけのブランチの例外(V3)と、tech-debt 243 行の classic の条件(V4)が古い |
| AC13 `--wait-checks` | 満たす(binary でも確認。情報 I1) | `internal/org/merge_wait.go`。待つ種類は `mergeRefusalKind.waits()`(`internal/org/merge.go:616-618`)の OrgRunning、ChecksPending、ChecksZero。org の待ちは `orgWaitEnd`(`merge_wait.go:250-264`)で台帳だけを読む。probe の wait-org: org が走る台帳で `--wait-checks 2m` を打ち、待ちの開始の行が出たあと 35 秒の間に gh の呼び出しは 1 本(待つ前の `pr list`)のまま増えず、台帳に `disbanded` を足すと次の読みで待ちを終え、確かめを最初から打って merge した(終わりの行は「the org was disbanded; running every check again」)。wait-org-timeout: `--wait-checks 1s` で待ち、時間切れのあとの断りに `waited 1s for org_id "a" to be disbanded` と `ralph org status --org-id a` が付いた(`waitedOut`、`:134-141`)。org-other-feature: 結びつきの org がほかの機能(`feat/b`)で走っていると、`--wait-checks 1m` でも待ちの行を出さずにすぐ断った。検査の待ち(wait-pass、wait-zero、wait-nowait)と範囲の検査(`-1s`、`6h0m1s`、`6h`)は cycle 1 と同じ。org の待ちの中断は `TestMerge_WaitChecksForTheOrgEnds`(`internal/org/merge_wait_test.go:414`)の「interrupted while it sleeps」を読んだ(走らせていない) |

D1〜D13 と実装(この cycle で変わったもの):

- D5: 結びつきの org が同じ機能で走っている断りは `mergeRefusalOrgRunning`、ほかの機能で走っている断りは `mergeRefusalOther`(`merge.go:496-512`)。後者の文は今までどおり RESULT の処理に触れない
- D6: 読み直しは `state,headRefOid,mergeCommit,autoMergeRequest`。gate の head の MERGED だけを記録し、別の head と空の head は記録も `MERGED:` の行もなしに終了コード 1(AC7 の行)
- D11: classic の保護は admin かどうかによらず `enforce_admins` を求める(AC5 の行)。断りの文は admin とカスタムロールの両方を言う
- D13: 待ちは `mergeLocked`(`merge_run.go:172-218`)が種類で決め(`:197-202`)、`Merge`(`:144-162`)の loop が `w.wait` に manifest のパスを渡す。待つ間はロックを持たない。2 つの待ちは `mergeWait` の 1 つの予算を分け合う(`started` と `deadline` は最初の待ちで決まる、`merge_wait.go:153-156`)。型の名前は計画のとおり断りの種類(`mergeRefusalKind`)になり、古い名前 `mergeChecksKind` と `mergeChecks{Other,Pending,Zero,NotPassed,Unread}` は `.go` に残っていない
- D1〜D4、D7〜D10、D12: この差分では名前の付け替えのほかに変わっていない。cycle 1 の確かめのまま
- Verify plan の API の field 名: `ruleset_id`(`mergeguarantee.go` の `ghBranchRule.RulesetID`)、`current_user_can_bypass`(`mergeenforce.go:245`)、`mergeQueue`(`mergeenforce.go:32-33`、`:209`)は計画の Assumptions と同じ。`permissions.admin` は読まない

仕様との整合:

- FR-8「既定では人が merge する」と受け入れ条件「Given `auto_merge = false`、when PR の CI が通る、then director は merge せず、merge してよい順を報告する」: skill の `:463-465` と pin のテストが支えるのは cycle 1 と同じ。既定値の 3 面(Go、`templates/base/ralph.toml:153`、`scripts/ralph-config.sh`)はこの差分で変わっていない
- 仕様の FR-8 の注(仕様 `:78`)は、S8 の 3 つ(classic の `enforce_admins`、読み直しの head、org の disband の待ち)を書く。gate の記録だけのブランチの例外は書かない(V3)
- 制約「director の権限は分割計画を書くことと、ralph と gh の操作に限る」: skill は `gh pr merge` を直接打たないと書き(`:487`)、org の時間切れの断りでも「director は disband を打たない」と書く(`:532`)

## Static analysis

| Command | Result | Notes |
| --- | --- | --- |
| `GOTOOLCHAIN=go1.25.8 ./scripts/run-static-verify.sh` | pass(rc 0) | scope は changed。`scripts/ralph-config.sh` が分類外なので full fallback になり、言語は golang。hook の shellcheck と `sh -n`、awk の構文、`jq -e` の 2 面、Codex の hook の検査 3 つ、`check-sync.sh`(IDENTICAL 170、DRIFTED 0、TEMPLATE_ONLY 11、KNOWN_DIFF 5)、`check-pipeline-sync.sh`、`check-skill-sync.sh`(14 skill)、`check-template-purity.sh`、tech-debt の plan の参照、`gofmt: ok`、golangci-lint `0 issues.`、`secret-scan-branch: scanned 15d22b1a..fd5e6c51 against origin/main: clean` |
| `gofmt -l internal cmd` | pass(rc 0) | 出力なし |
| `GOTOOLCHAIN=go1.25.8 go build ./...` | pass(rc 0) | go.mod の Go |
| `GOTOOLCHAIN=go1.25.8 go vet ./...` | pass(rc 0) | 出力なし |
| `go vet ./...`(go1.26.0) | pass(rc 0) | 出力なし |
| `golangci-lint run ./internal/... ./cmd/...` | pass | `0 issues.`(wrapper の中の実行も同じ) |
| `./scripts/check-skill-sync.sh` | pass(rc 0) | 14 skill が lock-step |
| `./scripts/check-sync.sh` | pass(rc 0) | DRIFTED 0 |
| `git diff --check 15d22b1a...HEAD` | pass(rc 0) | 出力なし |
| `./scripts/plan-visual.sh digest <plan>` | 一致 | 305d97070f38 |
| skill の 8 面の `cmp` | 一致 | `/director` と `/org` のそれぞれ 4 面 |
| S9b の移動の突き合わせ | 一致 | 7d0d08dd で 3 つの test から消えた 648 行と `merge_fake_test.go` に入った 662 行を並べ替えて比べた。違いは新しいファイルの package と import と先頭の注、消えた側の古いファイルごとの注 6 行だけ |

## Findings

| ID | Severity | 種類 | Finding | Evidence | 勧め |
| --- | --- | --- | --- | --- | --- |
| V3 | LOW | 文書(help、README、skill、仕様) | `--wait-checks` は「ほかの機能で走っている org は待たない」と、どの面も例外なしに書く。実際は、結びつきのない(gate の記録だけの)ブランチでは `orgRunsOn` が立たないので、その org がほかの機能を走らせていても step 8 は待つ種類で断り、`--wait-checks` は org の disband か時間切れまで待つ(`runningRefusal` と `orgWaitEnd` の注、tech-debt 298 行が書く限界)。probe の gate-only で、org b が `feat/b` で走り、`feat/a` に org b の gate の記録だけがある台帳では、`--wait-checks 1s` が待ちの行を出し、時間切れのあとに「its leader may still push」と `ralph org status --org-id b` を添えて断った。安全側に倒れる動きで、ずれは文書だけ | help: `internal/cli/org_merge.go:115`(「until the org is disbanded, runs another feature, or the ledger cannot be read」)と `:121`(「An org that runs another feature now ... are not waited for」)。README: `README.md:266` の「until the org is disbanded or runs another feature」と「an org that runs another feature now is not waited for」。`/org` skill: `.claude/skills/org/SKILL.md:189` の「org が disband するか、ほかの機能で走り出すか、台帳が読めなくなるまで待つ」と「ほかの機能で走っている org、…では待たない」(4 面とも同じ行)。`/director` skill: `.claude/skills/director/SKILL.md:513` の「ほかの機能で走り出した org は待たない」(4 面とも同じ行)。仕様: `docs/specs/2026-10-07-org-multi-org-director.md:78` の「ほかの機能で走り出した org は待たない」 | 8 か所(help 2、README 2、`/org` 2、`/director` 1、仕様 1)に「結びつきのあるブランチでだけ見分ける。gate の記録だけのブランチでは、その org がほかの機能を走らせていても disband か時間切れまで待つ」の趣旨を短く足す(/sync-docs)。`/director` の `:513` の文は pin の断片の外(直前の断片は `:511-513` の「org の disband を待つ」で終わる)なので、その断片の中を書き換えなければ pin のテストは落ちない。skill は 4 面を同じに保つ(`scripts/sync-skills.sh`) |
| V4 | LOW | 文書(tech-debt) | 243 行(`github_protection` の意味の行)の 4 つ目の欄(Trigger)は、`ralph org merge` が進む classic の条件を「a strict classic protection with required checks has a caller who is not an admin of the repository or `enforce_admins` enabled」と、S8a の前の規則で書く。S8a のあとは、admin かどうかによらず `enforce_admins` が有効なことを求める(AC5、D11)。この欄は続けて「so the gap does not reach its merges」と言うので、S8a の前の規則のままだと、cross-review #1 が指摘した非 admin のカスタムロールの穴が塞がっていないように読める。この文は S5(71c0582e)で入り、S8a で直っていない | `docs/tech-debt/README.md:243`(4 つ目の欄)。同じ PR の 296 行と、help・README・`/org` skill・仕様の注は S8a の規則に直っている | 「or a strict classic protection with required checks has `enforce_admins` enabled, whether or not the caller is an admin」に直す(/sync-docs) |

情報(ずれに数えない):

- I1 gate の記録だけのブランチの待ち(V3 の動き)は、計画の S9(Implementation outline 10 と Progress)、`orgWaitEnd`(`merge_wait.go:240-249`)と `runningRefusal`(`merge.go:485-495`)の注、tech-debt 298 行が限界として書いている。298 行の書き方(step 8 が `mergeRefusalOrgRunning` で断り、待ちが disband か予算まで続き、「its leader may still push to the branch」で断る)は probe の gate-only の出力と合う。AC13 の「org がほかの機能を走らせている断りでは待たない」は、その断り(`mergeRefusalOther` の文)が出るときは待たない、という意味では満たす(probe の org-other-feature)。gate の記録だけのブランチではその断り自体が出ない
- I2 tech-debt 299 行は「The PR only rewords a refusal in `regate.go`」と書くが、この PR は `regate.go` の注も 3 か所変えている(S4 と S9a)。行が言いたい「fetch の道(`regateBoundedGit`、`internal/org/regate_git.go`)に触れない」は正しく、`regate_git.go` はこの PR で変わっていない
- I3 cycle 1 の指摘の今: V1(`/director` の Codex の打ち方)は `:518-521` と `:536-538` に入った。V2(tech-debt 239 行の runner の数)は「six times」と 6 つの一覧になり、名前はどれも今のコードにある(`realGitStatus`、gate の runner、`realMergeOrderExec`、`regateBoundedGit`、`runDirectorLaunchCmd`、`ExecRunner.Run`。`realMergeDetachedExec` は `runMergeOrderCommand` を共有するので数に入らない)。I2(README の `-R` だけの括弧)は `gh repo view` の引数と `gh api` の `--hostname` も書くようになった。I3(計画の checkbox)は 124ed5df で直った。I4(`pending` と `fail` の混在)は `/org` skill の行が「`fail` か `cancel` が 1 つでもある検査(`pending` が残っていても)」と書くようになった
- I4 classic の非 admin が通る道は、本物の GitHub では strict を admin にしか見せないので、実際にはほぼ通らない(tech-debt 296 行)。probe で非 admin が通ったのは、偽の gh が非 admin にも strict を返したため。S8a のあとは、通るにはどの caller でも `enforce_admins` が要る
- I5 S9b(7d0d08dd)の test の行数は `merge_fake_test.go` 662、`merge_test.go` 626、`merge_run_test.go` 795、`merge_wait_test.go` 712 で、tech-debt 292 行の RESOLVED の注と合う。注が名指す `mxExec`、`mxFixture`、`mrExec`、`mrFixture`、`mwClock`、`mwFixture` はどれも `merge_fake_test.go` にある
- I6 新しいテスト `TestMerge_WaitChecksForTheOrgGoesOnWhenItStartsAgain`(`merge_wait_test.go:484`)はある。起き直したら待ちを終える mutant で落ちるという計画の記録は、テストを走らせていないので確かめていない(/test の受け持ち)

## Documentation drift

| Doc / contract | In sync? | Notes |
| --- | --- | --- |
| `ralph org merge --help` | 一部(V3) | classic の `enforce_admins`(`internal/cli/org_merge.go:73-78`)、読み直しの head と `MERGED:` を出さない場合(`:89-97`)、org の待ちと時間切れの `ralph org status`(`:107-130`)、終了コードの「of another head than the gate's」(`:139-141`)、待つ間はロックを外すこと(`:144-145`)、flag の説明(`:197-199`)はコードと合う。gate の記録だけのブランチの例外がない |
| README | 一部(V3) | `merge` の段落(`:266`)は gh の 3 つの固定の形、classic の `enforce_admins`、別の head の merge、org の待ちを書く。動詞の表(`:124`)と `ralph status` の行(`:125`)はコードと合う |
| `/org` skill の 4 面 | 一部(V3) | 4 面は `cmp` で同じ。`merge` の行(`:189`)、「merge してよい順」の org の待ち(`:959-963`)と classic の条件(`:972-977`)、regate のロックの段落(`:1075`)はコードと合う |
| `/director` skill の 4 面 | 一部(V3) | 4 面は `cmp` で同じ。org の待ち、Codex の打ち方、時間切れで disband を打たないこと、`MERGED:` の行の読み方(`:494-496`)はコードと合う。別の head の merge(`MERGED:` なしの終了コード 1)は「終了コード 1 で止めて stderr を伝える」(`:491-493`)の扱いに入り、stderr の文がほかのブランチを続けることを言う |
| 仕様 FR-8 の注(`:78`) | 一部(V3) | S8 の 3 つは実装と合う |
| `AGENTS.md` の repo map | yes | `merge_wait.go` に「for checks still pending and for the branch's org to be disbanded」が入った(`AGENTS.md:90`)。test のファイルはほかの module と同じく名指さないので、`merge_fake_test.go` がないのはずれではない |
| `docs/tech-debt/README.md` | 一部(V4。情報 I2) | 236〜243、259、291〜299 行のセルの数はどれも 5。292 行の注と 293 行の行数(877、848、1334、318、480、1238、1152)は `wc -l` と base の `git show` に合う。296 行は S8a の規則で書く。298 行は gate だけのブランチの限界を probe のとおりに書き、名指す `orgRunsOn`、`collectMergeTargets`、`mergeRefusalOrgRunning`、`runningRefusal`、`orgWaitEnd`、`mergeTargetOrg` はどれもある。243 行の 4 つ目の欄が S8a の前の規則のまま |
| `docs/quality/quality-gates.md` の 2 面、`templates/base/ralph.toml`、`scripts/ralph-config.sh` の 2 面 | yes | この差分で変わっておらず、cycle 1 の確かめのまま |
| Go の注(`merge.go`、`merge_run.go`、`merge_wait.go`、`mergeenforce.go`、`lockfile.go`、`regate.go`、`spawn.go`) | yes | 待つ種類、読み直しの head、classic の規則、待つ間のロックの持ち方は、コードと probe の動きに合う |
| 計画の AC と Progress | yes | AC1〜AC13 の checkbox は `[x]`、AC12 と AC13 の順も直っている。Verify plan の field 名から `permissions.admin` が外れている |

## Observational checks

probe は HEAD fd5e6c51 の binary(`GOTOOLCHAIN=go1.25.8 go build -o $P/ralph ./cmd/ralph`)で打った。場合ごとに、bare の origin、`main` と `feat/a` を push した checkout、台帳を作り直した。`env -i` で `PATH` は probe の bin(git の包みと偽の gh)だけ、`HOME` は偽の場所、`GIT_CONFIG_GLOBAL=/dev/null`。git の包みは `git remote get-url origin` にだけ `https://github.com/o/r.git` を返し、ほかは Homebrew の git 2.49.0 に渡す。偽の gh はシェルの組み込みだけで答え、呼び出しを記録する。cycle 2 の偽の gh は、読み直しの `--json state,headRefOid,mergeCommit,autoMergeRequest` で答えを分ける。ロックは `/usr/bin/perl` の flock で確かめた。ネットワーク、本物の gh、herdr、osascript、tmux は打っていない。

S8 の新しい場合(`probe2.sh`):

- happy: merge して `pr_merged` を 1 行、終了コード 0
- classic の 6 通り(非 admin・admin・`permissions` なし × `enforce_admins` の `true`・`false`): `false` の 3 通りは断り(gh は `enforce_admins` まで打ち、`repo view` と `pr merge` は打たない)、`true` の 3 通りは merge した
- other-head、no-head、fail-other-head: どれも stdout が空、`pr_merged` 0 行、終了コード 1
- org-nowait: 待ちなしで今の断り。wait-org: org の disband を待って merge(待つ間は gh 0 本、ロックは空いている)。wait-org-timeout: 待った時間と `ralph org status --org-id a` を添えて断る。org-other-feature: 待たずに断る。gate-only: 待ってから断る(V3、情報 I1)

cycle 1 の場合を HEAD の binary で打ち直した(`probe.sh` の読み直しの fixture に `headRefOid` を足した写し):

- happy、method の squash、merge-fails-merged、lock の bg、wait-pass、config の wt は merge して終了コード 0
- flags(5 つ)、notready、automerge-req、checks-pending、checks-zero(2 つ)、bypass(2 つ)、queue(2 つ)、classic(2 つ)、foreign(2 つ)、method の空、merge-fails-open、automerge-held(2 つ)、readback-fails、lock の held と regate、wait-zero(2 つ)、wait-nowait(2 つ)、config の bad は終了コード 1 で、記録しなかった
- cycle 1 から変わったのは classic の非 admin(`enforce_admins` が `false`)だけで、cycle 1 の merge から S8a のとおり断りに変わった
- interrupt は merge して記録し、6 行と `NOTE:` を出して終了コード 1。lookup の 4 つの値と config の表示は cycle 1 と同じ

## Coverage gaps

- テストは走らせていない。go1.26 と `GOTOOLCHAIN=go1.25.8` の `go test ./internal/org/ ./internal/cli/ ./internal/config/` は /test で確かめる
- binary で打っていない場合: 中断のあとに別の head の MERGED を読む場合、org の待ちの間の中断、検査の待ちと org の待ちが続けて起きる場合の予算の共有、待つ間に disband して同じ org_id が同じブランチで起き直す場合、記録の失敗、merge の間の Spawn。どれもコードとテストのコードを読んだ範囲で計画どおり
- 本物の GitHub と gh の振る舞い: merge された PR の `headRefOid` が merge のときの head のままか(計画の Risks。未確認)、非 admin が `enforce_admins` の endpoint を読めるか、`current_user_can_bypass` の `never` 以外の値、queue があるときの `mergeQueue`、0 件の文。知らない値と読めない値は断る側に倒れる
- この repo の main の ruleset は required status checks を持たないので、`ralph org merge` はこの repo では断る(tech-debt 295 行)。dogfood は確かめていない
- probe の生の出力は session の scratchpad にだけあり、repo の `docs/evidence/` には残していない(report のほかを書かない指示のため)

## Verdict: PASS

- Verdict: pass(MEDIUM 以上はない。LOW 2 件。V3 は `--wait-checks` が gate の記録だけのブランチで待つ例外を 5 つの面の 8 か所が書かないこと、V4 は tech-debt 243 行の classic の条件が S8a の前のままのことで、どちらも /sync-docs で直せる文書のずれ。情報 6 件)
- Verified: 承認の digest。AC1〜AC13 のコードの突き合わせ(S8 と S9 の差分は全部読んだ)。HEAD の binary で、classic の 6 通り、読み直しの別の head と空の head と `gh pr merge` の失敗との組み合わせ、org の disband の待ち(gh を打たないこと、ロックを外すこと、disband で merge まで進むこと)、時間切れの断りの `ralph org status`、ほかの機能で走る org を待たないこと、gate の記録だけのブランチの待ち、cycle 1 の 40 あまりの場合の打ち直し。skill の 8 面の同一と pin の 9 断片の順。S9b の移動の行の一致。tech-debt の行のセルの数、行数、名前。静的解析(run-static-verify、gofmt、go1.25.8 の build と vet、go1.26 の vet、golangci-lint、check-skill-sync、check-sync、`git diff --check`)
- Partially verified: AC9 と AC10 の Spawn の部分(コードとテストのコード)、AC12(LOW の V3・V4)
- Not verified: テストの実行、本物の GitHub と gh の振る舞い、Codex の director、dogfood
- Insight event: 指示により `docs/insights/events/2026-10-17-org-auto-merge.jsonl` には書いていない。orchestrator が足すなら次の行になる(cycle は 2、手で組んだ):
  `{"schema":1,"ts":"2026-10-17T19:50:00Z","slug":"org-auto-merge","flow":"standard","phase":"verify","cycle":2,"verdict":"pass","findings":{"critical":0,"high":0,"medium":0,"low":2},"triage":{"action_required":0,"worth_considering":0,"dismissed":0},"source":"skill"}`
