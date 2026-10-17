# sync-docs report: org-auto-merge

- Date: 2026-10-18(JST。計画の日付は 2026-10-17)
- Plan: `docs/plans/active/2026-10-17-org-auto-merge.md`(この run で計画は触っていない)
- Pipeline cycle: 1。base は `15d22b1a`、branch は `feat/org-auto-merge`。run 開始時の HEAD は `29afd327`(run 開始時の値で、この report を含む commit の SHA ではない)
- 先行 report: `docs/reports/self-review-2026-10-17-org-auto-merge.md`(初版は `29469c0e`、S6・S7 の再レビューの追記は `d93bdce1`。どちらも Merge: yes)、`docs/reports/verify-2026-10-17-org-auto-merge.md`(`c0ad3f37`。pass。LOW の V1・V2 と情報の I2〜I4 をこの run で扱う)、`docs/reports/test-2026-10-17-org-auto-merge.md`(`6e43569a`。pass。T1・T2 は `29afd327` で直っている。Failure analysis の揺れを変更 6 の行にする)
- コードとテストは変えていない。Edit で直した Markdown は 7 ファイル(`/director` skill と `/org` skill の `.claude` 側、それぞれ root と template の 4 面、`README.md`、仕様、tech-debt)、`scripts/sync-skills.sh` で再生成した skill の `.agents` 側の写し 4 面、この report

## Summary

verify の LOW 2 件(V1 と V2)と情報の I2、I4 を直し、test の揺れを tech-debt の 1 行にした。V1 は `/director` skill の `ralph org merge` の打ち方が Claude Code の分しか書かず、Codex の director が `--wait-checks 60m` の手順 2 を打つと、ツールの時間切れで止まりうるという指摘。V2 は tech-debt 239 行の最初の欄が runner を 3 つ、影響の欄が 4 つと 1 つの読み方で挙げていた食い違いで、コードに当てると同じ形の runner は 6 つあった。

`git diff 15d22b1a...HEAD` の振る舞い(`ralph org merge`、`--wait-checks`、`pr_lookup`、`[org.director].auto_merge`、`pr_merged`、`regate.lock` の共有と待ちの間の解放、gh の 0 件の文の読み取り)を、README、AGENTS.md、仕様の FR-8 の注、`/org` skill と `/director` skill、`docs/quality/`、`.claude/rules/ralph/`、`docs/recipes/`、template の文書に当てた。ほかに食い違いが 4 つあり(README の `ralph status` の行、README の regate の段落、仕様の gh の括弧、`/org` skill の `merge` の行)、直した。`ralph org merge --help` はコードなので直さず、文書と突き合わせた。ずれは見つからなかった。

## Changes made

| File | Change |
|------|--------|
| `.claude/skills/director/SKILL.md`(「`ralph org merge` の打ち方は、auto_merge の値で変わる。」の節) | V1。(1) `auto_merge = true` の `--wait-checks 60m` の打ち方に、Codex の文を足した: Codex でもそのドライバの background の実行で同じように走らせ、background にできなければ前景で打ち、ツールの timeout を待ちの上限の 60 分に、そのあとの確かめと merge にかかる時間を足した長さにする。background の書き方は `/plan` と `/cross-review` の「Codex driver: its own background run」に、前景の timeout の書き方は regate の「前景で打つなら、ツールの timeout を regate にかかる時間より長くする」に合わせた。(2) `auto_merge = false` の前景の merge の「Claude Code では Bash のツールの timeout を 10 分にする」に、「Codex でもコマンドを打つツールの timeout を 10 分にする」を足した。pin の断片(`directorSkillAutoMergeSteps` の 7 つ)は手順 2 と末尾の箇条にあり、触れていない |
| `templates/base/.claude/skills/director/SKILL.md` | 同じ 2 か所(root と同じ文字列で Edit した) |
| `.agents/skills/director/SKILL.md`、`templates/base/.agents/skills/director/SKILL.md` | `scripts/sync-skills.sh` で再生成した(root は引数なし、template は `CLAUDE_ROOT=templates/base/.claude/skills CODEX_ROOT=templates/base/.agents/skills`)。差分は上の 2 か所だけ |
| `docs/tech-debt/README.md`(239 行の `merge-order --json` の行) | V2。コードに当てた runner は 6 つ: `realGitStatus`(`watch.go`、`gitStatusWaitDelay`)、gate の script runner(`gate_run.go`、`gateKillGrace`)、`realMergeOrderExec`(`mergeorder.go`、`mergeOrderWaitDelay`。`realMergeDetachedExec` は同じ body の `runMergeOrderCommand` を別の process group で動かす入口)、`regateRemoteGit`(本体は `regate_git.go` の `regateBoundedGit`、`regateGitWaitDelay`)、`runDirectorLaunchCmd`(`director_cmd.go`、`defaultDirectorLaunchWaitDelay`)、`ExecRunner.Run`(`internal/org/driver/driver.go`、`execWaitDelay`)。6 つ目はこの行が数えていなかったが、`realGitStatus` と `regateBoundedGit` のコメントが手本にしている、同じ形の runner である。最初の欄の「three times」を「six times」にして 6 つを挙げ、影響の欄は「each of the six runners」と 6 つの名前(`runDirectorLaunchCmd` を「読み方」から runner に、`ExecRunner.Run` を足した)、Why deferred の欄は 6 つのファイル、Related の欄は足りなかった 3 つのファイルを足した。取り消し線の部分と、Trigger の欄の経緯(「fourth runner」の文)は、そのときの記述として変えていない |
| 同(表の末尾に 1 行) | test report の Failure analysis。`TestRegate_FetchFailures/the_fetch_runs_past_the_bound`(`internal/org/regate_test.go`)が、/test で `internal/org` の全体を 3 本並べて走らせた 1 回だけ落ちた。fetch が 300 ms の上限で止められず `exit 128`(`fatal: Could not read from remote repository.`)で返った。単独 10/10、16 本の busy loop を足して 5/5、mutant の overlay で 3/3 通り、wrapper と go1.25.8 の全体でも通った。この PR は regate の fetch の道(`regateBoundedGit`)に触れていない。原因は未特定。Trigger は「CI か手元で再び落ちたら、単独で打ち直してから、`exit 128` の前に git が何を出したかと、上限より前に返ったかを調べる」。5 つの欄で、区切りの縦線は 6 つ |
| `README.md`(`ralph org merge` の段落) | I2。gh を固定する書き方を、実装に合わせた。`-R <host>/<owner>/<repo>` だけだった括弧を、origin の repository を `git remote get-url origin` から読み、`gh pr` に `-R <host>/<owner>/<repo>`、`gh repo view` に repository を引数で、`gh api` に `--hostname <host>` を渡す、と書いた(help、`/org` skill の `merge` の行、仕様と同じ) |
| `README.md`(regate の段落) | 「One regate runs per ledger at a time (`regate.lock`)」を「One regate or merge runs per ledger at a time (`regate.lock`, which `ralph org merge` shares)」にした。merge の間は regate が断られる(help の `internal/cli/org_regate.go` は、この PR で同じ言い方に変わっている) |
| `README.md`(`ralph status` の行) | `director:` の行の説明に、この PR が足した末尾の句を書いた。`attached` と `not attached` は `[org.director].auto_merge`(`enabled` と同じ規則で読む)の意味を言う: false なら merge は人に任せる、true なら director が `merge-order` の `ready` の PR を `ralph org merge` で人に聞かずに merge する。`disabled` は `auto_merge = true` が効かないと言う。`--json` には `director.auto_merge` が false でも出る(`orgDirectorLine` と `ralph status --help` に合わせた) |
| `docs/specs/2026-10-07-org-multi-org-director.md`(FR-8 の 8a-3a の注) | gh の固定の括弧に「`gh repo view` は引数」を足した(README の I2 と同じ) |
| `.claude/skills/org/SKILL.md`(`merge` の行) | I4。`--wait-checks` が待たない場合の「`fail`・`cancel`・`skipping` だけの検査と、ほかの断りでは待たない」を、「`fail` か `cancel` が 1 つでもある検査(`pending` が残っていても)と、`skipping` だけの検査と、ほかの断りでは待たない」に直した。`readChecks`(`internal/org/merge.go`)は `pending` と `fail` が混ざれば `mergeChecksNotPassed` で、待たない |
| `templates/base/.claude/skills/org/SKILL.md`、`.agents/skills/org/SKILL.md`、`templates/base/.agents/skills/org/SKILL.md` | 同じ 1 か所。`.claude` の 2 面は Edit、`.agents` の 2 面は `scripts/sync-skills.sh` |

skill の 4 面(`director` と `org` のそれぞれ)は `cmp` で同一バイト。

## Surfaces checked for drift

| 面 | 結果 | メモ |
|----|------|------|
| `internal/cli/org_merge.go` の help | 合っている(コードのずれなし) | 確かめの順、`--wait-checks` の待ち方(30 秒ごとに `gh pr checks` だけ、ロックを外す、時間切れの断り)、終了コード、`regate.lock` を待つ間は持たないこと、`auto_merge` が `AUTO MERGE:` と `pr_merged` にだけ出ることは、README、`/org` skill の `merge` の行、仕様の注と同じ。gh の固定の書き方も、README と仕様を help に合わせた |
| `internal/cli/org_regate.go` の help | 合っている | 「no other ralph org regate or merge runs on the ledger」。README の regate の段落を合わせた |
| `internal/cli/status.go`、`org_director.go` の help と `orgDirectorLine` | README の `ralph status` の行を直した | `/director` skill と `/org` skill の `director status` の行はすでに `auto_merge` を書いている |
| `README.md` の verb の行、director の段落、merge-order の段落、`ralph org merge` の段落 | 直したもののほか合っている | verb の行は `merge` を持つ。merge-order の段落は `pr_lookup` の 5 つの値を help と同じに書く |
| `AGENTS.md` の `internal/org/` の行 | 変更不要 | `merge.go`・`merge_run.go`・`merge_wait.go`・`mergeenforce.go` と `regate.lock` の下で打つことを持つ。見張りの門の節は spawn・start・TASK の送信だけを挙げ、merge を含まないので、見張りを要らない merge と合う |
| 仕様 FR-8 の注(8a-3a) | 括弧の 1 か所を直した | `pr_lookup` の値、`regate.lock` の共有と待ちの間の解放、`pr_merged` が running の判定に入らないこと、main の ruleset が required check を持たないので今は断ること(tech-debt の行あり)は、コードと合う。FR-8 の checkbox は付けない、と注が言う |
| `/director` skill(4 面) | 直した | V1 のほか、手順 2 の pin の断片、`auto_merge = false` で merge しないこと、`MERGED:` が出ていれば終了コード 1 でも merge 済みと読むことは、help と合う |
| `/org` skill(4 面) | 直した | 動詞の表の `merge`、「merge してよい順」、regate との排他(`1069` 行付近)、`pr_lookup`、`auto_merge` は合っている。直したのは I4 の 1 か所 |
| `docs/quality/quality-gates.md` と template の写し | 変更不要 | 見張りの門の行は、見張りなしで動く verb に `merge` を持つ(この PR が足した)。`ralph org merge` の確かめを門の行にするかどうかは、行が増える判断で、合っていない文はない。merge-order、doctor の merge guarantee も行を持たない前例に合わせた |
| `.claude/rules/ralph/`、`docs/recipes/`、`docs/specs/2026-08-01-org-runtime.md`、`.ralph/core/AGENTS.core.md` | 変更不要 | `ralph org merge`・regate・merge-order・auto_merge・`pr_merged` の語を持たない(grep で 0 件。`aggregates` の語中の一致は除く)。`agent-messaging.md` の TASK と見張りの記述は `merge` の送信を含まない。`ralph-workflow.md` の `/director` の 1 文は merge に触れず、矛盾しない |
| `templates/base/ralph.toml`、`scripts/ralph-config.sh`(root と template) | 合っている | `auto_merge = false` とその説明、`RALPH_ORG_DIRECTOR_AUTO_MERGE`。`internal/config/defaults_sync_test.go` が 3 つの既定値を突き合わせ、`check-sync.sh` が root と template を見る |
| `docs/tech-debt/README.md` のこの PR の行(`merge_test.go` などの 800 行超、本物の GitHub で試していない 3 つの読み、main の ruleset、classic の admin、gh の 0 件の文) | 変更なし | コードに当てた名前は実在し、`run-static-verify.sh` の「tech-debt README plan references」は OK |
| 絶対パスの `$HOME` | なし | 今回の差分に `/Users/` も `hiroki.yoshioka` もない。U+FFFD もない |

## Left open

- verify の I3: 計画の受け入れ条件の checkbox が AC1〜AC13 とも `[ ]` のまま、AC13 が AC12 の前に並び、Verify plan の節が「AC1〜AC12」と書くこと。計画は orchestrator が直すので、この run では触れていない(この report の verify の結果をもとに、`/pr` の前に `[x]` と並びを直す)
- README は `enabled` について「`ralph upgrade` does not add the key, so an existing `ralph.toml` gets the default true」と言うが、`auto_merge` には書いていない。`templates/base/ralph.toml` のコメントは「An absent auto_merge loads as false」と言う。既存の `ralph.toml` に `auto_merge` を足すのは人で、release ノートに書く項目になる(7 段目の `[org.approval]` と同じ)。README には足していない
- `/director` skill の Codex の文(変更の V1)は、Codex の shell ツールの timeout の上限と background の実行の有無を確かめていない。regate の前景の文と `/plan`・`/cross-review` の「its own background run」の書き方に合わせただけで、Codex の director での実機の確かめは、この PR の範囲に入っていない
- test report の揺れ(変更の tech-debt の 1 行)の原因は未特定。再び落ちたらこの行の Trigger に従う

## Verification

| Command | Result |
|---------|--------|
| `./scripts/sync-skills.sh`(root と template) | 14 skill と 13 skill を再生成。変わったのは `director` と `org` の `.agents` の 4 面だけ(`.claude` 側を Edit したあと) |
| `./scripts/check-skill-sync.sh` | 終了コード 0。`[ok] check-skill-sync: 14 skill(s) in lock-step` |
| `./scripts/check-sync.sh` | 終了コード 0。PASS(KNOWN_DIFF 5) |
| `GOTOOLCHAIN=go1.25.8 go test ./internal/org/ -run 'DirectorSkill' -count=1 -timeout 30m` | 終了コード 0。`ok  github.com/yoshpy-dev/ralph/internal/org` |
| `./scripts/run-static-verify.sh` | 終了コード 0。tech-debt の plan の参照 OK、gofmt ok、`0 issues.`、`secret-scan-branch: scanned 15d22b1a..29afd327 against origin/main: clean`。出力は scratch に保存(repo の外) |
| `git diff --check` | 出力なし、終了コード 0 |
| `director` と `org` の skill の 4 面 | `cmp` で同一バイト(`.claude` と `.agents`、root と template) |
| `docs/tech-debt/README.md` の 239 行と表の末尾の行 | 区切りの縦線はどちらも 6 つ(5 列)。末尾の行が挙げる名前(`TestRegate_FetchFailures`、`regateBoundedGit`)は実在する。239 行が挙げる 6 つの runner の関数と定数は、`internal/org/` と `internal/org/driver/` で grep して実在を確かめた |
| 変更した Markdown の U+FFFD、差分の `/Users/` と `hiroki.yoshioka` | どちらも 0 件 |

`./scripts/verify.local.sh` と Go のテストの全体は走らせていない。変えたのは文書と skill の写しだけで、`run-static-verify.sh` が同じ範囲の検査(skill の lock-step、template の一致、tech-debt の plan の参照、secret scan)を含んで通り、skill の文を固定するテストも通っている。

## Verdict

- Verdict: pass(sync-docs に stop condition はない)
- Not verified: Codex の director が `--wait-checks 60m` を打つ実機の動き(上の「Left open」)。`TestRegate_FetchFailures` の揺れの原因

## Insight event

orchestrator が足す行(この run では jsonl に書いていない)。`ts` は足す時刻、ファイルは `docs/insights/events/<UTC の日付>-org-auto-merge.jsonl`:

```
./scripts/insights-append.sh --slug org-auto-merge --flow standard --phase sync_docs --cycle 1 --verdict pass --source skill --critical 0 --high 0 --medium 0 --low 0
```

```
{"schema":1,"ts":"<UTC>","slug":"org-auto-merge","flow":"standard","phase":"sync_docs","cycle":1,"verdict":"pass","findings":{"critical":0,"high":0,"medium":0,"low":0},"triage":{"action_required":0,"worth_considering":0,"dismissed":0},"source":"skill"}
```
