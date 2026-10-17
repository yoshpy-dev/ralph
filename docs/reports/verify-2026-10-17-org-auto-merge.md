# Verify report: org-auto-merge

- Date: 2026-10-17
- Plan: docs/plans/active/2026-10-17-org-auto-merge.md(`Status: Approved`。`./scripts/plan-visual.sh digest` は abefab9fdedd で、`- Approved:` の digest と同じ。仕様 `docs/specs/2026-10-07-org-multi-org-director.md` の FR-8 のうち 8a-3a)
- Verifier: verifier subagent (Claude)。パイプライン cycle 1 の verify(`.harness/state/standard-pipeline/cycle-count.json` の cycle は 1)
- Scope: `git diff 15d22b1a...HEAD`(HEAD b82f9dee、54 ファイル、+7496/-317)。コードと文書を読み、HEAD の binary と base(15d22b1a)の binary を、偽の gh と手元の bare repo を origin にした台帳で打った。テスト(`go test`、`./scripts/run-test.sh`、`./scripts/run-verify.sh`)は走らせていない(/test の受け持ち)。gh は実行していない(gh 2.102.0 の binary を `strings` で読んだだけ)。ネットワーク、herdr、osascript、tmux、端末の app には触れていない。行番号は HEAD b82f9dee のもの
- Evidence: `docs/evidence/verify-2026-10-17-org-auto-merge.log`(digest、run-static-verify の全出力、Go の検査、行数、skill の 4 面と pin の断片、gh の binary の strings、probe のスクリプトと出力、`ralph org merge --help`。`$S` は scratchpad、`$P` はその下の probe のディレクトリ)。wrapper の全ログは `docs/evidence/verify-2026-10-17-144222.log`。どちらも gitignore の対象

## Spec compliance

| Acceptance criterion | Status | Evidence |
| --- | --- | --- |
| AC1 merge-order の `pr_lookup` | 満たす(binary でも確認) | 5 つの値は `internal/org/mergeorder.go:84-106`、field は `MergeOrderEntry.PRLookup`(`:209-213`)。`lookupMergePRs` は最初に全 target を `skipped` にし(`:622-627`)、`failed`(`:644`)、`limit_forks`(`:648`)、`none`(`:651`)、`found`(`:662`)を付ける。gh がないとき(`:639-640`)と別の repository を読んだとき(`:659`)は、また全 target を `skipped` に戻す。probe の lookup で `found`・`none`・`failed`・`skipped`(gh を PATH から外すと `gh_available: false`)、otherrepo で PR の url が `github.com/x/y` のとき `skipped`。base の binary と比べると、4 つの状態のどれでも text は同じで、JSON は `pr_lookup` の行のほかは同じ。`limit_forks` は binary では打っていない(テストのコードを読んだ) |
| AC2 `[org.director].auto_merge` | 満たす(binary でも確認) | `internal/config/config.go:180`(field)、`:377`(既定 false)、`:575`(型の検査は toml の decode に任せる注)。`withMainWorktreeDirectorEnabled`(`internal/cli/org.go:218-229`)が `:227` で AutoMerge も main worktree の `ralph.toml` から取る。`statusDirectorConfig`(`internal/cli/status.go:258`)、JSON の `auto_merge`(`internal/cli/org_director.go:493`)、行の末尾の句(`:611`)、`auto_merge:` の行(`:701`)。probe の config: main の `ralph.toml` が `auto_merge = true`、linked worktree の `ralph.toml` が `false` のとき、linked worktree から打った merge は `AUTO MERGE: true` を出し、`auto_merge=true` の `pr_merged` を書いた。`ralph org director status` は句と `auto_merge: true`、`ralph status` は `director:` の行の末尾に句、`--json` は `"auto_merge": true`。main の値を `"yes"` にすると merge は `cannot decode TOML string into ... AutoMerge of type bool` で終了コード 1 |
| AC3 断りの 14 の場合 | 満たす(10 の場合は binary でも確認) | 名指しなし: CLI `internal/cli/org_merge.go:147-150`、`Merge` は `internal/org/merge.go:206-209`。`--org-id`: `org_merge.go:143-146`。台帳にない: `merge.go:242-246`。`ready` でない: `:255-258`(class と Detail を出す)。`pr_lookup` が `found` でない: `findPR`(`:404-423`)。OPEN でない、head、base: `viewPR`(`:529-538`)。org が走っている: `runningRefusal`(`:481-496`)。予約: `:541-544`。検査: `readChecks`(`:603-655`)。方式: `readMethod`(`:662-682`)。ロック: `internal/org/merge_run.go:162-171`。直前の ls-remote: `checkOriginUnmoved`(`merge.go:689-712`)。probe では、名指しなし、`--org-id`、台帳にない、`rebase`(origin の main を進めた)、予約、`pending`、0 件、空の方式、ロック、別の host の PR で断り、どれも gh の `pr merge` を打たず、`pr_merged` は 0 行だった。OPEN でない、head と base の違い、走っている org、直前の ls-remote の 5 つは binary では打っていない(コードを読んだ) |
| AC4 対象の repository | 満たす(binary でも確認。情報 I2) | `readOrigin`(`merge.go:344-373`)。`isOriginsPR`(`:124-127`)は host も比べ、`findPR`(`:418`)と `viewPR`(`:539`)が使う。probe の happy で動詞が打った gh の 10 本は、`gh pr` がすべて `-R github.com/o/r`、`gh api` がすべて `--hostname github.com`、`gh repo view` は引数 `github.com/o/r` だった(D2 のとおり `-R` を持たない)。probe の foreign: PR の url が `https://ghe.example.com/o/r/pull/7`(同じ owner/repo で別の host)なら断る。origin がローカルのパスなら、gh を 1 回も打たずに断る |
| AC5 保護 | 満たす(binary でも確認) | `enforce`(`internal/org/mergeenforce.go:120-154`)、`readMergeQueue`(`:203-235`)、`enforceRulesets`(`:242-276`)、`enforceClassic`(`:284-311`)、`notEnforced`(`:316-326`)。probe で断ったもの: `current_user_can_bypass` が `always` と `pull_requests_only`、GraphQL の `mergeQueue` が null でない、rules に `merge_queue`、classic の strict で admin かつ `enforce_admins` が false。通ったもの: ruleset の `never`(happy)、classic の非 admin。読めない場合と `pr_checks_only`、自前の check のない up-to-date の rule は、binary では打っていない(コードを読んだ)。doctor 側: `internal/org/mergeguarantee.go` の差分は `strictRulesets`、`mergeStrictRuleset`、`RulesetID`、`addStrictRuleset` を足すだけで、`CheckMergeGuarantee` と `decide` には触れない(既存のテストの実行は /test) |
| AC6 通る道 | 満たす(binary でも確認) | argv は `merge_run.go:245-251`。probe の happy は `pr merge 7 --merge --match-head-commit <gate の head> -R github.com/o/r` を 1 回だけ打ち、`--auto`・`--admin`・`--delete-branch` はなかった。出力の 6 行(`org_merge.go:232-243`)を出して終了コード 0。`pr_merged` の Details(`merge_run.go:406-408`)は D4 の順で、`branch=feat/a pr=7 head=<head> method=merge merge_commit=<sha> auto_merge=false`。probe の method で `SQUASH` なら `--squash`、`METHOD: squash`、`method=squash` |
| AC7 読み直しで MERGED でない、中断 | 満たす(binary でも確認) | `mergeAndRecord`(`merge_run.go:317-345`)、`removeAutoMerge`(`:353-364`)、`readBack`(`:301-311`)、`ghAfterStart`(`:291-293`、`context.WithoutCancel` と別のプロセスグループ)、中断の扱い(`:202-204`、`:422-432`)。probe: 予約があると `--disable-auto` を 1 回打ち、外したと言って終了コード 1。外す gh が落ちると外すコマンドを出して終了コード 1。読み直しが落ちると `gh pr view ... --json state,mergeCommit` を出して終了コード 1。どれも記録なし。interrupt: `gh pr merge` が 5 秒眠る間に ralph に SIGINT を送ると、gh は `signal: killed` で止まり、読み直しを 1 回打ち、MERGED なので記録して 6 行と `NOTE:` を出し、`interrupted (interrupt signal received) once gh pr merge had started` で終了コード 1 |
| AC8 `gh pr merge` の失敗 | 満たす(記録の失敗はコードで確認) | probe の merge-fails-merged: gh が終了コード 1 でも読み直しが MERGED なら、記録して `NOTE:` に gh の末尾 2 行を出し、終了コード 0。merge-fails-open: 読み直しが OPEN なら、`Head branch was modified` の末尾を stderr に出し、記録なしで終了コード 1。記録の失敗は `recordMerge`(`:403-416`)と `printMergeResult`(`org_merge.go:212-229`)で、MERGED の行を出してから終了コード 1(binary では打っていない) |
| AC9 `pr_merged` の扱い | 満たす(コードとテストのコードで確認) | `internal/org/seat.go:90-107`。`stateEvents` に入らない。`RunningOrgs`(`internal/org/reserve.go:464`)は Roster と workspace と予約だけ、`orgsToDisband`(`internal/org/verbs_all.go:337`)は状態の event と workspace と予約だけ、`leaderActivityEventCount`(`internal/org/watch.go:1236`)は `pr_merged` を数えない。`TestPRMerged_IsNotASeatAndDoesNotRun`(`internal/org/merge_run_test.go:692`)が 4 つの読み手、Roster、`BuildOrgReport` の時系列の行を押さえる(走らせていない) |
| AC10 ロックの共有 | 満たす(Spawn はコードで確認) | `mergeLocked` が `regate.lock` を取る(`merge_run.go:162`)。probe の lock: perl の flock がロックを持つ間は merge が断られ、merge の `gh pr merge` の間(4 秒眠る)は `ralph org regate` が `another ralph org regate or merge runs on this ledger` で断られ、flock も取れなかった。Spawn: 印を空にする(`:177`)だけで書かず、`regateSpawnErr`(`internal/org/spawn.go`)は空の印で nil を返す。`TestMerge_SharesRegateLockWithRegate`(`merge_run_test.go:754`)が押さえる(走らせていない) |
| AC11 `/director` skill | 満たす(LOW の V1) | `.claude/skills/director/SKILL.md:453-528`(false は `:463-465`、true の 4 段は `:466-477`、終わり方と報告は `:479-484`、どちらでも変わらないことは `:485-501`、打ち方は `:503-528`)。4 面は cmp で同じ。`TestDirectorSkill_PinsTheAutoMergeSteps`(`internal/org/director_skill_sync_test.go:119`)の 7 つの断片は、空白を落とすと 4 面のどれにも順に入っていて、古い文はない(python で同じ規則を確かめた。テストは走らせていない)。Codex の director への打ち方の案内がない(V1) |
| AC13 `--wait-checks` | 満たす(binary でも確認。情報 I1・I4) | `internal/org/merge_wait.go`。範囲の検査は CLI(`org_merge.go:151-153`)と `Merge`(`merge_run.go:135-137`)。待つのは `mergeChecksKind.waits()`(`merge.go:593`)の Pending と Zero だけ。probe の wait-pass: `pending` から 30 秒後に `pass` になると、待ちの開始と終わりの 2 行を stderr に出し、`gh pr list` から確かめを打ち直して merge した。待つ間は perl の flock が `regate.lock` を取れた。wait-zero: `no checks reported on the 'feat/a' branch` を 0 件と読んで待ち、`--wait-checks 1s` が切れると確かめを打ち直し、待った時間を添えて断った。別のブランチの名前の文は gh の失敗として待たずに断った。wait-nowait: `pending` と `fail` が混ざる場合と `skipping` だけの場合は、待ちの行を出さずに断った。`-1s` と `6h0m1s` は gh も git も打たずに断り、`6h` は受け付けた。待つ間の中断は binary では打っていない(テストのコードを読んだ) |
| AC12 文書 | 一部(LOW の V2。情報 I2・I3) | 下の「Documentation drift」。`/org` skill の動詞リファレンス(`.claude/skills/org/SKILL.md:189`)、仕様 FR-8 の注(仕様 `:78`)、README(`:124`、`:258`、`:266`)、`AGENTS.md:90`、quality-gates の 2 面、tech-debt(236〜243 行、292〜296 行)、`templates/base/ralph.toml:145-153`、CLI の help は、動詞と設定と `pr_lookup` と保護の条件を書く |

D1〜D13 と実装:

- D1: 動詞は auto_merge を読まずに打てる(`MergeParams.AutoMerge` は報告だけ、`merge.go:77-79`)。probe の config で `auto_merge = false` の linked worktree の `ralph.toml` は効かず、main の `true` が `AUTO MERGE:` と `pr_merged` に出た
- D2: `gh repo view <host>/<owner>/<repo> --json viewerDefaultMergeMethod`(`merge.go:664`)。空の値と知らない値は断る(probe の method)
- D3: 1 秒まで取り直す(`regateLockWait`、`internal/org/regate.go`)、取った直後に印を空にする(`merge_run.go:177`)
- D4: Details の順と org_id(`mergeTargetOrg`、`merge.go:470-475`)は計画のとおり
- D5: 結びつきの org がほかの機能を走らせている場合の文は「disband されるまで待って打ち直す」で、RESULT の処理には触れない(情報 I5)
- D6〜D8、D10〜D12: 上の AC6〜AC8、AC3〜AC5 の行のとおり
- D7: 0 件の文は gh 2.102.0 の binary に `no checks reported on the '%s' branch` の書式として入っている(`strings`)。`%s` に入るのが PR の head のブランチの名前かは、binary からは読めない(おそらくそう。未確認。計画は gh のソースで確かめたと書く)
- D9: 5 つの値(AC1)
- D11 の field 名: `ruleset_id`(`mergeguarantee.go` の `ghBranchRule.RulesetID`)、`current_user_can_bypass`、`mergeQueue`、`permissions.admin`、`enabled` は計画の Assumptions と同じ。gh の binary の strings に `viewerDefaultMergeMethod`、`autoMergeRequest`、`mergeQueue`、`match-head-commit`、`disable-auto` がある
- D13: 待つ間はロックを持たない(probe の wait-pass)。待ち終えると確かめを最初から打つ(probe の gh の呼び出しの順)

仕様との整合:

- FR-8「既定では人が merge する」: `auto_merge` の既定は 3 面(Go、`templates/base/ralph.toml:153`、`scripts/ralph-config.sh:101`)とも false で、`internal/config/defaults_sync_test.go:179` が揃いを押さえる。`cmp` で `scripts/ralph-config.sh` と `templates/base/scripts/ralph-config.sh` は同じ
- 受け入れ条件「Given `auto_merge = false`、when PR の CI が通る、then director は merge せず、merge してよい順を報告する」: skill の `:463-465` と「進捗の報告」節の merge-order の箇条で満たす。動詞はこれを強いない(D1)ので、skill の文と pin のテストだけが支える
- 「どちらの場合も merge してよい順を報告する」: 「進捗の報告」節は値によらず merge-order を報告に入れる。true の手順も merge と regate のたびに結果を報告する(`:479-484`)
- 制約「director の権限は分割計画を書くことと、ralph と gh の操作に限る」: skill は `gh pr merge` を直接打たないと書き(`:487`)、`internal/org/prompts/director.md` の権限の文は変わらない。director が gh を直接打てる点は D1 と計画の Assumptions が認めている

## Static analysis

| Command | Result | Notes |
| --- | --- | --- |
| `./scripts/run-static-verify.sh` | pass(rc 0) | scope は changed。`scripts/ralph-config.sh` が分類外なので full fallback になり、言語は golang。hook の shellcheck と `sh -n`、awk の構文、`jq -e` の 2 面、Codex の hook の検査 3 つ、`check-sync.sh`(IDENTICAL 170、DRIFTED 0、KNOWN_DIFF 5)、`check-pipeline-sync.sh`、`check-skill-sync.sh`(14 skill)、`check-template-purity.sh`、tech-debt の plan の参照、`gofmt: ok`、golangci-lint `0 issues.`、`secret-scan-branch: scanned 15d22b1a..b82f9dee against origin/main: clean` |
| `go build ./...`(go1.26.0) | pass(rc 0) | 出力なし |
| `gofmt -l internal/ cmd/` | pass(rc 0) | 出力なし |
| `go vet ./...`(go1.26.0) | pass(rc 0) | 出力なし |
| `GOTOOLCHAIN=go1.25.8 go build ./...` | pass(rc 0) | go.mod の Go。`strings.SplitSeq`(Go 1.24 から)を使っても通る |
| `GOTOOLCHAIN=go1.25.8 go vet ./...` | pass(rc 0) | 出力なし |
| `golangci-lint run ./internal/...` | pass(rc 0) | `0 issues.` |
| `./scripts/check-skill-sync.sh` | pass(rc 0) | 14 skill が lock-step |
| `git diff --check 15d22b1a...HEAD` | pass(rc 0) | 出力なし |
| `sh -n` と `shellcheck`(`scripts/ralph-config.sh`、`templates/base/scripts/ralph-config.sh`) | pass(rc 0) | 2 つは `cmp` で同じ |
| `./scripts/plan-visual.sh digest <plan>` | 一致 | abefab9fdedd |

## Findings

| ID | Severity | 種類 | Finding | Evidence | 勧め |
| --- | --- | --- | --- | --- | --- |
| V1 | LOW | 文書(skill) | `/director` skill の `ralph org merge` の打ち方は、`--wait-checks 60m` の background の打ち方(`run_in_background` と `.harness/state/merge-<名前>.out`)も、前景で打つときのツールの timeout(10 分)も、Claude Code の分しか書かない。skill は Codex の director にも使われ(attach の `--driver codex`、`:71`)、受信箱の待ちには Codex の行がある(`:341-344`)。regate の打ち方には「前景で打つなら、ツールの timeout を regate にかかる時間より長くする」という、どちらにも効く文がある(`:545-546`)。merge の打ち方にはそれもないので、Codex の director が手順 2 を最大 60 分の前景のコマンドとして打つと、ツールの時間切れで止まりうる(Codex のツールの既定の時間は確かめていない) | `.claude/skills/director/SKILL.md:503-528`(4 面は同じ)、`:334-344`、`:538-546` | 4 面に Codex の 1 文を足す。たとえば「Codex では、ツールの timeout を `--wait-checks` の時間より長くして前景で打つ」。前景の merge の「Claude Code では Bash のツールの timeout を 10 分にする」にも同じ趣旨の Codex の文を足す(/sync-docs。pin の断片には触れない) |
| V2 | LOW | 文書(tech-debt) | 239 行の最初の欄は、外部のコマンドの runner が「now written three times」で 3 つ(`realGitStatus`、gate の runner、`realMergeOrderExec`)と書く。同じ行の影響の欄は、この PR で書き直されて、5 つ(4 つの runner と `runDirectorLaunchCmd` の `exec.ErrWaitDelay` の読み方)を挙げる。最初の欄の「Separately,」を「Still open:」に替えたときに、数と一覧は残った。base の最初の欄も「three times」で、`regateRemoteGit`(#234)と `runDirectorLaunchCmd`(#232)が入ったときに直っていない。base の影響の欄は runner を挙げていなかったので、同じ行の中の食い違いはこの PR の書き直しで生じた | `docs/tech-debt/README.md:239`。base は `git show 15d22b1a:docs/tech-debt/README.md` の 238 行 | 最初の欄を「four times ... `realMergeOrderExec` ..., and `regateRemoteGit` (`internal/org/regate_git.go`, `regateGitWaitDelay`); `runDirectorLaunchCmd` (`internal/org/director_cmd.go`) reads `exec.ErrWaitDelay` the same way」に直す(/sync-docs) |

情報(ずれに数えない):

- I1 断りに足す「waited <時間>」は、最初の待ちの始まりから断りまでを測るので、待ちのあとの確かめの時間も入る。probe の wait-zero では、終わりの行が `after 1s`、断りが `ralph org merge waited 2s for the checks (--wait-checks 1s)` だった。`waited` の doc(`merge_wait.go:99-101`)のとおりの値で、`60m` なら数秒の差にとどまる
- I2 AC4 は「動詞が打つ gh はすべて `-R`(`gh api` は `--hostname`)」と書くが、`gh repo view` は repository を引数で受ける(D2 と D10 がそう書き、help と `/org` skill もそう書く)。README の `:266` は「pins every `gh` call to origin's repository (`-R <host>/<owner>/<repo>`, ...)」と括弧に `-R` だけを書く。どれも振る舞いは同じで、gh はすべて origin の repository に固定されている(probe の happy)
- I3 計画の受け入れ条件の checkbox は AC1〜AC13 のどれも `[ ]` のまま。AC13 が AC12 の前に並び、Verify plan の節は「AC1〜AC12」と書く(S7 で AC13 を足したとき直していない)。/sync-docs か /pr で、この report をもとに checkbox を `[x]` にし、並びと Verify plan の文を直すとよい
- I4 `/org` skill の `merge` の行と AC13 の「`fail`・`cancel`・`skipping` だけの断りでは待たない」は、`pending` と `fail` が混ざる場合を言葉にしない。D13(「`fail` と `cancel` がなく `pending` がある」)と help(「A check that failed or was cancelled ... are not waited for」)ははっきりしていて、コードも待たない(probe の wait-nowait)
- I5 結びつきの org が今ほかの機能を走らせている場合の断り(`merge.go:485-487`)は「disband されるまで待って打ち直す」と言い、D5 の「受信箱の RESULT の件を処理し」を言わない。その org の RESULT はこの機能のものではないので、この文の方が合う
- I6 classic の非 admin で通る道(`mergeenforce.go:291-292`)は、偽の gh が非 admin に strict を返したので通った。本物の GitHub は strict を admin にしか見せないので、実際にはこの道は通らない。計画の Risks と tech-debt 295 行が書いている
- I7 self-review の再レビューの R1〜R4 の直し(97a1a062)は、読んだ範囲でコードに入っている。ロックの注の 3 か所(`internal/org/lockfile.go:132-141`、`internal/org/regate.go:90-97`、`internal/org/spawn.go:1324-1329`)、待ちの開始行の丸め(`merge_wait.go:128`)、`poll` の注(`:157`)、`waitFor`(`merge_run.go:144`)、`mergeInterrupted` の再利用(`merge_wait.go:134`)

## Documentation drift

| Doc / contract | In sync? | Notes |
| --- | --- | --- |
| `/director` skill の 4 面 | 一部(V1) | 4 面は cmp で同じ。手順、断りで止めること、`gh pr merge` を打たないことはコードと合う。Codex の打ち方がない |
| `/org` skill の 4 面 | yes(情報 I4) | 4 面は同じ。`merge` の行(`:189`)、台帳を書き換える動詞の一覧(`:57`)、見張りなしで使える動詞(`:1275`)、「merge してよい順」の節(`:940`、merge の箇条は `:954-961`、保護の箇条は `:966-975`)、regate のロックの段落(`:1069`)はコードと合う |
| `ralph org merge --help` | yes | 確かめの順は `checkMerge` の 13 段とロックの順に合う。時間(ロック 1 秒、`gh pr merge` 60 秒、gh 15 秒)、`--wait-checks` の範囲と 30 秒の間隔、終了コード、出力の行も合う |
| `ralph org` の help と `--org-id`・`--config` の usage | yes | `merge` が `--org-id` を断る動詞に入り、`--config` の usage が auto_merge の読み手を名指す |
| `ralph org director status` と `ralph status` の help | yes | probe の config の出力と合う |
| `ralph org merge-order --help` の `pr_lookup` | yes | 5 つの値と 1000 の上限は `mergeOrderGHLimit` と合う |
| README | yes(情報 I2) | 動詞の表(`:124`)、director の段落(`:258`)、merge-order の `pr_lookup`、`merge` の段落(`:266`) |
| `AGENTS.md` の repo map | yes | `merge.go`、`merge_run.go`、`merge_wait.go`、`mergeenforce.go` を名指す(`:90`) |
| 仕様 FR-8 の注(`:78`) | yes | 確かめの一覧、auto_merge の効かせ方、`pr_merged`、`pr_lookup`、`--wait-checks` は実装と合う |
| `docs/quality/quality-gates.md` の 2 面 | yes | 見張りなしで使える動詞に `merge` がある。2 面の違いは KNOWN_DIFF のもの |
| `docs/tech-debt/README.md` | 一部(V2) | 800 行の行(292 行)の数(876、841、926、877、848、1334、326、452、1238、1152)は `wc -l` と合う。238 行の RESOLVED の注と 239 行の取り消し線は `pr_lookup` と合う。0 件の文の行(296 行)は `mergeNoChecksSentence` と合う。236、237、240〜243 行の 8a-3a の追記はコードと合う |
| `templates/base/ralph.toml` と `scripts/ralph-config.sh` の 2 面 | yes | `auto_merge = false`(`:153`)、`RALPH_ORG_DIRECTOR_AUTO_MERGE` の既定 false(`:101`)、`Default()` も false |
| Go の注(`seat.go` の `EventPRMerged`、`lockfile.go`、`regate.go`、`spawn.go`、`mergeorder.go`) | yes | 書き手と読み手の一覧、ロックの持ち方(待つ間は外す)はコードと合う |
| `internal/org/prompts/director.md` と `leader.md` | yes | 非目標のとおり変えていない。director の雛形は skill に従う作りで、merge を人の仕事と言い切る文はない |
| 計画の AC と Progress | 一部(情報 I3) | checkbox がすべて空、AC13 の位置、Verify plan の「AC1〜AC12」 |

## Observational checks

probe は、HEAD b82f9dee の binary(`go build -o $P/ralph ./cmd/ralph`)と、base 15d22b1a を `git archive` で scratchpad に出して作った binary で打った。場合ごとに、bare の origin、`main` と `feat/a` を push した checkout、台帳(org a の結びつき、`feat/a` の先頭を `main` の先頭で通した `gate_passed`、`disbanded` の 3 行)を作り直した。`env -i` で `PATH` は probe の bin(git の包みと偽の gh)だけ、`HOME` は偽の場所、`GIT_CONFIG_GLOBAL=/dev/null`。git の包みは `git remote get-url origin` にだけ `https://github.com/o/r.git` を返し、ほかは Homebrew の git 2.49.0 に渡す。偽の gh はシェルの組み込みだけで答え、呼び出しを記録し、名前ごとの回数で答えを変えられる。ロックの確かめは `/usr/bin/perl` の flock で、中断は ralph の pid に SIGINT を 1 回送った。ネットワーク、本物の gh、herdr、osascript、tmux は打っていない。

- happy、method の squash、classic の非 admin、merge-fails-merged、lock の bg、wait-pass、config の wt は merge して `pr_merged` を 1 行書き、終了コード 0
- flags(5 つ)、notready、automerge-req、checks-pending、checks-zero(2 つ)、bypass(2 つ)、queue(2 つ)、classic の admin、foreign(2 つ)、method の空、merge-fails-open、automerge-held(2 つ)、readback-fails、lock の held と regate、wait-zero、wait-nowait(2 つ)、config の bad は終了コード 1 で、記録しなかった
- interrupt は merge して記録し、6 行と `NOTE:` を出して終了コード 1
- lookup と compare-text で、merge-order の text は base と同じ、JSON は `pr_lookup` のほかは同じ

## Coverage gaps

- テストは走らせていない。go1.26 と `GOTOOLCHAIN=go1.25.8` の `go test ./internal/org/ ./internal/cli/ ./internal/config/` は /test で確かめる
- binary で打っていない場合: `limit_forks`、PR が OPEN でない、PR の head と base の違い、走っている org、直前の ls-remote の動き、保護の読み取りの失敗、`pr_checks_only`、自前の check のない up-to-date の rule、記録の失敗、待つ間の中断、merge の間の Spawn。どれもコードとテストのコードを読んだ範囲で計画どおり
- 本物の GitHub と gh の振る舞い: 0 件の文の `%s` の中身、`current_user_can_bypass` の `never` 以外の値、queue があるときの `mergeQueue`、up-to-date の保護が遅れた head の merge を断ること、`--match-head-commit` の効き方。計画の Assumptions と tech-debt 294 行のとおり、知らない値は断る側に倒れる
- Codex の director での `--wait-checks 60m` の打ち方(V1)と、Codex のツールの既定の時間
- この repo の main の ruleset は required status checks を持たないので、`ralph org merge` はこの repo では断る(計画の Risks、tech-debt 295 行)。dogfood は確かめていない

## Verdict: PASS

- Verdict: pass(MEDIUM 以上はない。LOW 2 件。V1 は `/director` skill の Codex の打ち方、V2 は tech-debt 239 行の数のずれで、どちらも /sync-docs で直せる文書のずれ。情報 7 件)
- Verified: 承認の digest。AC1〜AC11 と AC13 のコードの突き合わせ。HEAD の binary で、通る道の gh の argv と固定した repository、`pr_merged` の Details、断り 25 件の終了コードと記録なし、`gh pr merge` の失敗と読み直しの 4 つの終わり方、予約の外し、`gh pr merge` の途中の SIGINT、ロックの共有、`--wait-checks` の待ちと時間切れと待たない場合、main worktree の auto_merge の読み方と型違いの値、`ralph status` と `director status` の表示、`pr_lookup` の 4 つの値と base との text の比較。skill の 4 面の同一と pin の断片。静的解析(run-static-verify、go build、gofmt、go vet、go1.25.8 の build と vet、golangci-lint、check-skill-sync、`git diff --check`、ralph-config.sh の `sh -n` と shellcheck と cmp)
- Partially verified: AC9 と AC10 の Spawn の部分(コードとテストのコード)、AC12(LOW の V2)、AC11(LOW の V1)
- Not verified: テストの実行、本物の GitHub と gh の振る舞い、Codex の director、dogfood
- Insight event: `docs/insights/events/2026-10-17-org-auto-merge.jsonl` には書き足していない(orchestrator が commit する指示で、書いてよいのは report と evidence だけのため)。scratch の `--events-dir` で作った行は次のとおりで、orchestrator がこの行を足せばよい:
  `{"schema":1,"ts":"2026-10-17T15:04:26Z","slug":"org-auto-merge","flow":"standard","phase":"verify","cycle":1,"verdict":"pass","findings":{"critical":0,"high":0,"medium":0,"low":2},"triage":{"action_required":0,"worth_considering":0,"dismissed":0},"source":"skill"}`
