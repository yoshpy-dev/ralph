# sync-docs report: org-regate

- Date: 2026-10-17(JST)
- Plan: `docs/plans/active/2026-10-16-org-regate.md`(承認済み。この run で計画は触っていない)
- Pipeline cycle: 1。base は `670ec219`、branch は `feat/org-regate`。run 開始時の HEAD は `84e258b2`(run 開始時の値で、この report を含む commit の SHA ではない)
- 先行 report: `docs/reports/self-review-2026-10-16-org-regate.md`(2 回目で Merge: yes、LOW 4 件は 0b6ba232 で直った)、`docs/reports/verify-2026-10-16-org-regate.md`(pass。LOW 1 件は下の変更 1)、`docs/reports/test-2026-10-16-org-regate.md`(pass。穴 1 は 5e4ce6cd で閉じ、穴 2〜4 は PR の既知の穴に書く)
- コードとテストは変えていない。変えたのは Markdown の 4 ファイルと、`scripts/sync-skills.sh` で再生成した skill の写し 2 面

## Summary

S5(5400264c ほか)で、動詞の表、skill の 8 面、仕様の FR-7 の注、tech-debt、`AGENTS.md`、`README.md` は揃っていた。この run で直したのは、依頼された 2 点だけだった。`docs/quality/quality-gates.md` の「見張りなしで使える動詞」の一覧に `regate` がなかったこと、`/org` skill の `regate` の行が `reset --keep` の手を出す条件の 1 つを書いていなかったこと。あわせて、品質ゲートの表の「Quality pipeline gate」の行は `ralph org gate` が disband 済みの org を断ることまでしか言わず、regate を知らなかったので、1 文足した。ほかの面は、コードと `ralph org regate --help` などの出力に当てて、ずれを見つけなかった。

## Changes made

| File | Change |
|------|--------|
| `docs/quality/quality-gates.md`(「Cross-org watch gate」の行) | 見張りなしで使える動詞の一覧に `regate` を足した(`gate`, `merge-order` の次)。regate は `wireOrgWatchGate` も `wireOrgStartWatchGate` も呼ばず、`internal/org/regate*.go` に `WatchGate` の呼び出しもない。`ralph org regate --help` も「needs no running watch and no approved plan」と言う |
| `docs/quality/quality-gates.md`(「Quality pipeline gate」の行の末尾) | org を disband したあとは `ralph org regate --branch <branch>` が gate に代わる、と 1 文足した。merge commit で取り込む(rebase も force push もしない)、同じ 2 つのスクリプトを走らせる、同じ記録を `via=regate` 付きで書く、通れば push する |
| `templates/base/docs/quality/quality-gates.md`(同じ 2 か所) | 上の 2 つを同じ文で足した。この対は `scripts/check-sync.sh` の `KNOWN_DIFFS` にあるので、既知の差(check-sync.sh と check-template.yml の行、`see docs/tech-debt` の句)は残した |
| `.claude/skills/org/SKILL.md`(`regate` の行) | `reset --keep` の手を出す条件に「そのあとに manifest の壊れた行がなく」を足した。`regateResetHint`(`internal/org/regate_merge.go`)は `g.readable` と `!b.corruptAfterGate` の両方を見るが、行は前者しか言っていなかった。同じ行の `UP_TO_DATE:` の文の「そのあとに manifest の壊れた行があるときは除く」と同じ言い方に合わせた |
| `templates/base/.claude/skills/org/SKILL.md` | 同じ直し(root と同一バイト) |
| `.agents/skills/org/SKILL.md`、`templates/base/.agents/skills/org/SKILL.md` | `scripts/sync-skills.sh` で再生成した(root は引数なし、template は `CLAUDE_ROOT=templates/base/.claude/skills CODEX_ROOT=templates/base/.agents/skills`)。差分は上の 1 行だけ |

## Surfaces checked for drift

| 面 | 結果 | メモ |
|----|------|------|
| `README.md`(動詞の表の `ralph org` の行、`merge-order` の段落) | 合っている | `regate` が動詞の並びに入っている。段落の記述(merge commit、rebase と force push をしない、`via=regate`、失敗した gate では merge commit を戻し `head_moved` と `tree_changed` では戻さない、`regate.lock`、見張りも承認も要らない、`--org-id` を断る、同じ org の spawn と start を断る)は `ralph org regate --help` と `printRegateResult` の出力に合う。merge-order の「merges nothing」は regate が別の動詞なので正しいまま |
| `AGENTS.md`(`internal/org/` の行) | 合っている | `regate.go`、`regate_merge.go`、`regate_git.go` の 3 つの役割分担と `regate.lock` の句。実在するファイルと一致。managed block(`.ralph/core/AGENTS.core.md`)には org の行がなく、変更不要 |
| `/org` skill(動詞リファレンスの `regate` の行、「merge してよい順」、gate の記録の節、見張りなしの動詞の一覧、古い台帳の守りの一覧) | 直した 1 か所を除いて合っている | 120 秒、1 秒のロック待ち、`UP_TO_DATE:` / `PUSH_ONLY:` / `MERGED:` / `PUSHED:` の行は `regateLockWait` と `regateGitTimeout` と `printRegateLines` に合う。書き換える動詞の一覧(57 行目)と見張りなしの動詞の一覧(1255 行目)には `regate` が入っている。読むだけの動詞の一覧には入っていない(正しい) |
| `/director` skill(「merge のあとの取り直し」、「まだないもの」、してはいけないこと) | 合っている | 人に頼まれたときだけ 1 本ずつ打つ、終了コード 1 は regate の文をそのまま伝える、衝突を直さない、force push をしない。「まだないもの」は「merge のあとの regate を director が自分から打つこと」で、仕様の「auto_merge と一緒に 8a-3a で決める」と同じ |
| `/org` と `/director` の skill の 4 面ずつ | 合っている | `check-skill-sync.sh` が 14 skill で lock-step。org の本文(frontmatter の 5 行目以降)は root の `.claude` と両方の `.agents` で同一、director は変更なし |
| `docs/specs/2026-10-07-org-multi-org-director.md`(FR-7 の 8a-3b の注) | 合っている | 注が挙げるテスト 2 本(`TestRegate_MergesTheTipInAndPushes`、`TestOrgRegate_MergesPushesAndThenIsUpToDate`)は存在する。FR-7 の checkbox を付けない理由も、7c-2 の注から続けて読める。FR-9 の 8a-1 の注の「3 本目(8a-3)は `auto_merge` と取り直しを持つ」は当時の記録で、FR-7 の 8a-3b の注が 2 本に分けたことを書いている |
| `docs/tech-debt/README.md` | 合っている | 「regate」を含む表の行は 10 本(この PR の RESOLVED と打ち消し線の行、前からある行を含む)で、すべて 5 列。`docs/plans/(active|archive)/...` の参照 67 件はすべて実在(`verify.local.sh` の `check_tech_debt_plan_refs` と同じ規則を手で当てた)。この PR の行は計画を `plans/active/` で指す(`/pr` の `archive-plan.sh` が書き換える)。追加した行に `file:line` はない |
| `docs/quality/quality-gates.md` と template の写し | 直した | 上の 2 点 |
| `docs/recipes/`、`docs/architecture/repo-map.md`、`docs/quality/definition-of-done.md` | 変更不要 | gate、merge-order、regate に触れていない。repo-map に `internal/org/` のファイルの一覧はない。`codex-seat-permissions` の recipe は reviewer と leader の雛形の使命を試す recipe で、この PR は雛形を変えていない |
| `internal/org/prompts/`(`leader.md`、`reviewer.md`、`director.md`) | 変更不要 | 計画の Non-goals(leader の雛形の手順 9〜16)。`director.md` は merge と gate に触れず `/director` skill に任せる |
| コマンドのヘルプ(`ralph org --help`、`ralph org regate --help`、`ralph org merge-order --help`、`--org-id` の説明) | 合っている | HEAD の binary を scratchpad に build して読んだ。`org` の説明、`regate` の動詞の一覧の行、`--org-id` を断る動詞の一覧に `regate` がある。merge-order のヘルプの `regate` の分類と「For a branch whose feature org is not running」の段落は、`/org` skill の「merge してよい順」と README に合う。分類の並びは変わっていない |
| `.claude/rules/ralph/`(`agent-messaging.md` ほか) | 変更不要 | regate は座席にも message にも触れない |
| 絶対パスの `$HOME` | なし | 今回の差分と変更した文書に `/Users/` も `hiroki.yoshioka` もない |
| 計画の Progress | 変更不要 | S1〜S5、self-review 2 回、verify、test、test のあとの 5e4ce6cd まで記録済み。AC1〜AC13 は付いている。`PR created` は /pr が付ける。依頼どおり計画の本文には触れていない |

## Left open

- `ralph org spawn --help` と `ralph org start --help`、`/org` skill の `spawn` と `start` の行は、regate が走っている間の同じ org_id の spawn と start の断りに触れていない。断りは regate のヘルプ、「機能ごとの org」節(skill)、README、断りの文自体が言う。見張りの関門の断りのように spawn と start の行にも書くなら、コードのヘルプ文の変更が要るので、この run では直さなかった。INFO
- test report の穴 2〜4(fetch と push の途中の中断、push・merge・`git var`・reset の 120 秒の上限、`merge --abort` 自体の失敗)は、計画の Progress のとおり PR の既知の穴に書く。tech-debt にはまだ行がない
- `internal/org/director_cmd_test.go` の sh の trap を 100 ms の sleep で待つテストが、負荷が高い go1.25.8 の全体の実行で 1 回落ちたこと(tech-debt に行がある)。この PR では直さない

## Verification

| Command | Result |
|---------|--------|
| `./scripts/sync-skills.sh`(root と template) | 14 skill と 13 skill を再生成。変わったのは `org` の 2 面だけ |
| `./scripts/check-skill-sync.sh` | `[ok] check-skill-sync: 14 skill(s) in lock-step` |
| `./scripts/check-sync.sh` | PASS(IDENTICAL 170、DRIFTED 0、ROOT_ONLY 0、TEMPLATE_ONLY 11、KNOWN_DIFF 5) |
| `md5` で `/org` skill の root と template、本文の 3 面 | root と template の `.claude` は同一。3 つの本文(`.claude` の 5 行目以降、2 つの `.agents`)は同一 |
| `docs/tech-debt/README.md` の plan 参照 67 件の存在、regate の行の列の数 | 欠けなし、すべて 5 列 |
| `git diff --check`(作業ツリー)と `git diff --check 670ec219...HEAD` | どちらも出力なし、終了コード 0 |
| HEAD の binary のヘルプ | 上の表のとおり |

`./scripts/verify.local.sh` は走らせていない。変えたのは文書と skill の写しだけで、`check-skill-sync.sh` と `check-sync.sh` が通っていれば、その 2 つの検査は同じ範囲を見る。

## Verdict

- Verdict: pass(sync-docs に stop condition はない)
- Not verified: Go のテストとリント(コードを変えていないので走らせていない。verify と test の report が先行している)
