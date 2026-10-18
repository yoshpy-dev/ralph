# Walkthrough: org-auto-merge(8 段目の前半の 3 本目の前半、8a-3a)

- Date: 2026-10-18(計画の日付は 2026-10-17)
- Plan: docs/plans/active/2026-10-17-org-auto-merge.md(/pr で archive に移す)
- Branch: feat/org-auto-merge(base 15d22b1a。この walkthrough を書いた時点で 61 commits、61 files、+9173/-318)
- PR: #235
- 読む範囲: `docs/reports/`、`docs/evidence/`、`docs/insights/` と計画を除くと 53 files、+8283/-318 になる。そのうちテストは `internal/cli/org_merge_test.go`(+798)、`internal/org/merge_run_test.go`(+790)、`merge_wait_test.go`(+748)、`merge_fake_test.go`(+706)、`merge_test.go`(+626)、`mergeenforce_test.go`(+480)、`mergeorder_test.go`(+226)、`merge_readback_test.go`(+210)ほか。ほかに `/org` と `/director` の skill の 4 面ずつ、README、`AGENTS.md`、仕様 FR-8 の注、`docs/quality/quality-gates.md` の 2 面、tech-debt がある
- 後工程は 3 回まわした。cross-review の 1 回目が ACTION_REQUIRED を 3 件、2 回目が 1 件出し、2 回目の直しのために上限を 2 から 3 に上げた(下の 9)

新しいファイルは `internal/org/merge.go`、`merge_run.go`、`merge_wait.go`、`mergeenforce.go`、`internal/cli/org_merge.go` と、それぞれのテスト。次の順に読むと、`ralph org merge` が 1 回で通る順に追える。

## 1. merge-order の `pr_lookup`(`internal/org/mergeorder.go`、S1)

- entry に `pr_lookup` を足した。値は `found`(origin の repository の PR を使った)、`none`(gh が PR を返さなかった)、`failed`(gh がこのブランチで落ちた)、`limit_forks`(gh が上限の数の PR を返し、すべて fork だった)、`skipped`(gh がない、全ブランチの PR を捨てた、問い合わせる前に止まった)の 5 つ(`MergeOrderPRLookup*`)
- `lookupMergePRs` が値を決める。text の出力、分類、merge-order の gh の打ち方は変えていない(Non-goals)。動詞は `found` のときだけ進む

## 2. 設定 `[org.director].auto_merge`(`internal/config/config.go`、S2)

- `OrgDirectorConfig.AutoMerge`(既定 false)。`enabled` と同じ規則で、`--config` がなく台帳が main worktree のものなら main worktree の `ralph.toml` から読む
- 値は `ralph status` の `director:` の行と `--json`、`ralph org director status` に出る(`internal/cli/status.go`、`org_director.go`)。`scripts/ralph-config.sh` の 2 面の `RALPH_ORG_DIRECTOR_AUTO_MERGE` と `defaults_sync_test.go` が既定値を揃える
- 動詞は auto_merge によらず打てる(D1)。auto_merge は `/director` skill が読む「人に聞かずに merge と取り直しをしてよいか」の印で、merge の確かめは変わらない

## 3. merge の前の 13 段(`internal/org/merge.go`、S3a)

- `(*Org).checkMerge` が順を持つ。どれも何も書かず、ロックも取らない。1 つ目が落ちればそこで断り、後ろは打たない
  1. ブランチの名指し、2. origin の URL から host と owner/repo を読む(`readOrigin`、D10)、3. gh が PATH にある
  4. 台帳がブランチを名指す(merge-order と同じ `readMergeLedger` と `collectMergeTargets`)、5. ls-remote(`listOriginBranches`)
  6. origin の repository の PR を見つける(`findPR`)、7. merge-order の分類が `ready`(`classifyMergeTarget`)
  8. 結びつきの org(なければ gate の記録の org)が走っていない(`runningRefusal`、D4、D5)
  9. `gh pr view`: OPEN、head が gate の head、base が既定のブランチ、url が origin の repository、auto-merge の予約がない(`viewPR`、D12)
  10. `gh pr checks`: すべて pass か skipping で、pass が 1 つ以上(`readChecks`、D7)
  11. 保護(`CheckMergeEnforcement`、下の 4)、12. merge の方式(`readMethod`、D2)、13. ls-remote の打ち直し(`checkOriginUnmoved`、D8)
- gh はすべて origin の repository に固定する。`gh pr` は `-R <host>/<owner>/<repo>`、`gh api` は `--hostname <host>`、`gh repo view` は引数。`GH_PROMPT_DISABLED=1` と `GIT_TERMINAL_PROMPT=0` を付ける
- 断りの文は `org: merge: ` で始まり、見つけたこと、何も merge していないこと、次の手を 1 つ言う。台帳や gh から来た制御文字は `PrintableInboxText` で escape する
- 検査が 0 件の PR では、gh 2.102.0 の `gh pr checks --json` は JSON を出さずに終了コード 1 で終わり、`no checks reported on the '<branch>' branch` と言う。この文を 0 件として読む(`mergeNoChecksSentence`)
- 断りの種類は `mergeRefusalKind` で返す。`waits()` が true になるのは、同じ機能で走っている org(`mergeRefusalOrgRunning`)、検査の待ち(`ChecksPending`)、0 件(`ChecksZero`)の 3 つだけで、`--wait-checks` はこの 3 つだけを待つ(下の 6)

## 4. 保護の確かめ(`internal/org/mergeenforce.go`、D11)

- `CheckMergeEnforcement` は、doctor の `mergeguarantee.go` の読み手を使って保護を live で読む。キャッシュ(`merge-guarantee.json`)は使わない
- 既定のブランチに merge queue があれば断る(GraphQL の `repository.mergeQueue(branch:)` と、ruleset の `merge_queue` の rule の両方を見る)
- ruleset では、up-to-date の rule が自前の required check を持ち、`current_user_can_bypass` が `never` のときだけ進む(`enforceRulesets`)
- classic の保護では、`strict` と required check に加えて、`enforce_admins` が有効なことを、打つ本人が admin かどうかによらず求める(`enforceClassic`、cross-review cycle 1 の #1。カスタムロールが迂回の権限を持ちうるため、`permissions.admin` は読まない)
- 読めない値と知らない値は断る

## 5. merge と読み直しと記録(`internal/org/merge_run.go`、S3b)

- `(*Org).Merge` は台帳ごとの `regate.lock` の排他を取る(取れなければ 1 秒まで打ち直す、D3)。regate と merge は台帳ごとに 1 つずつ走る。merge は worktree に触らないので、Spawn は空の印を読んで進む
- `mergeLocked` が `checkMerge` を通したあと、`gh pr merge <n> --merge|--squash|--rebase --match-head-commit <gate の head> -R <repo>` を 1 回打つ(60 秒で区切る)。`--auto`、`--admin`、`--delete-branch` は付けない
- 終了コードによらず、`gh pr view --json state,headRefOid,baseRefName,mergeCommit,autoMergeRequest` で 1 回読み直す(`readBack`、D6)。読み直しと `--disable-auto` は別のプロセスグループで打ち、ctx が終わればグループごと止める(`realMergeDetachedExec`)
- `mergeAndRecord` の分かれ方
  - MERGED で head が gate の head、base が既定のブランチ: manifest のロックの下で `pr_merged` を 1 行書く(`recordMerge`、D4)。Details は `branch=`、`pr=`、`head=`、`method=`、`merge_commit=`、`auto_merge=`
  - MERGED で head が違う(空を含む): 記録せず `MERGED:` も出さずに終了コード 1(`otherHeadMerged`、cross-review cycle 1 の #3)
  - MERGED で base が違う(空を含む): 同じく記録しない(`otherBaseMerged`、cross-review cycle 2 の #1)。merge-order はそのブランチを `merged` と並べることも Failed で言う。head の文が先に来る
  - MERGED でない: 予約があれば `--disable-auto` で外し、何も記録せずに終了コード 1(`removeAutoMerge`、D12)
  - 読み直しが読めない: merge されたか分からないと言い、確かめるコマンドを出す
- `gh pr merge` の途中の中断は、読み直しを 1 回打ってから終わる(`mergeInterruptedAfterStart`)
- `pr_merged` は座席でない org の event で、走っている org の判定に入らない(`internal/org/seat.go` の `EventPRMerged`)

## 6. `--wait-checks`(`internal/org/merge_wait.go`、S7、S8c、D13)

- 既定 0 で待たない。上限は 6 時間。範囲の検査は CLI と `Merge` の 2 か所にある
- 待つ間は `regate.lock` を外す。この file の待ちだけが、Merge が途中でロックを外す場所である(file の先頭の注)
- 検査の待ちは `gh pr checks` だけを 30 秒ごとに読む(`checksWaitEnd`)。org の待ちは台帳だけをロックなしで読み、org が disband されたか、ほかの機能を走らせ始めたら終わる(`orgWaitEnd`)。どちらも 1 つの予算を分け合う
- 待ちが終わったらロックを取り直し、13 段を最初から打つ。時間が切れたら、待たないときと同じ断りに、どれだけ待ったかを足す。org の待ちでは `ralph org status --org-id <org>` を名指す(disband は人の判断のため)
- stderr には、待ちの始まりと終わりに 1 行ずつ出す
- `mergeWait.wait` は待つ kind ごとに case を持ち、`default:` は誤りを返す。`TestMergeWait_EveryKindThatWaitsHasACase` が、`waits()` に kind を足して case を足し忘れる変更を落とす
- 結びつきのない gate だけのブランチでは、org がほかの機能を走らせ始めても見分けられず、予算まで待つ(安全側。tech-debt の行)

## 7. CLI(`internal/cli/org_merge.go`、S4)

- `ralph org merge --branch <b> [--wait-checks <時間>]`。merge して記録したときだけ、stdout に 6 行(`MERGED:` ほか)を出して終了コード 0。断り、失敗、記録しない MERGED は stderr に理由を出して終了コード 1
- help は保護の条件、記録の条件、待ちの規則を書く。`TestOrgMerge_Help` が 45 個の断片を固定する
- fake gh はシェルスクリプトで、PATH を fixture の bin に置き換える

## 8. skill と文書(S5、sync-docs)

- `/director` skill の 4 面: auto_merge が true のときは merge-order → merge(`--wait-checks 60m` を background で打つ)→ merge-order → 残りのブランチの regate → … の順で進める。false のときは今までどおり報告し、人に頼まれたら動詞で 1 本 merge する。merge は動詞でだけ打ち、動詞が断ったら理由を人に伝えて止まる。director は disband を打たない。`TestDirectorSkill_PinsTheAutoMergeSteps`(`internal/org/director_skill_sync_test.go`)が 9 つの断片とその順を固定する
- `/org` skill の 4 面の動詞リファレンスに `merge` の行。README の動詞の表、`AGENTS.md` の repo-map、仕様 FR-8 の注、`docs/quality/quality-gates.md` の 2 面、`templates/base/ralph.toml` のコメント
- tech-debt: merge-order の JSON の半分を RESOLVED にした。新しい行は、merge の文が merge-order の文に頼ること、本物の GitHub で試していない 3 つの読み取り、この repo の ruleset、classic の保護と admin、gh の 0 件の文、gate だけのブランチの待ち、merge-order が MERGED を base を見ずに `merged` とすること、`TestRegate_FetchFailures` の揺れ

## 9. 後工程の 3 回の直し

- cycle 1: self-review の MEDIUM 2(保護の関数を `mergeenforce.go` へ、検査の待ちがないと auto_merge でも毎回止まる → D13 と S7)と LOW 8。cross-review の 3 件(classic の `enforce_admins`、読み直しの head、org の disband の待ち)を S8 で直した
- cycle 2: self-review の LOW 6 を S9 で、test の LOW 1 を 425a8543 で直した。cross-review の 1 件(確かめのあとに PR の base を付け替えられると、別のブランチへの merge を記録してしまう)を S10 で直した
- cycle 3(上限を 3 に上げた回し直し。`cycle-count.json` は 2 のまま): self-review の LOW 7 を S11 で、test の LOW 2(既定のブランチが main でないときの読み直し、merge-order の分類の pin)を d099bab0 で直した。cross-review は 0 件

## 10. 試していないこと

- 本物の GitHub での `baseRefName` と `headRefOid` の答え、base を付け替えた PR の merge、merge queue のある repo、迂回できる ruleset の値、`enforce_admins` の endpoint
- この repository の main の ruleset には required status checks がないので、ここでは動詞はすべて断る。9 段目の trial の前に人が設定する
