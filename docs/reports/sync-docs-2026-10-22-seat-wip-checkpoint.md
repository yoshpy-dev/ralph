# sync-docs report: seat-wip-checkpoint

- Date: 2026-10-22
- Plan: `docs/plans/active/2026-10-22-seat-wip-checkpoint.md`(承認の digest `b98f2806ab31` はこの run の前後で同じ。`./scripts/plan-visual.sh digest` で確かめた。触ったのは digest の範囲の外の Progress checklist だけ)
- Pipeline cycle: 1(`.harness/state/standard-pipeline/cycle-count.json` の値。上限は 2)。base は `c90d01b4`、branch は `fix/seat-wip-checkpoint`。run 開始時の HEAD は `62f92f42`(この report を含む commit の SHA ではない)
- 先行 report: `docs/reports/self-review-2026-10-22-seat-wip-checkpoint.md`(Merge OK、LOW 7。7 件とも a79b0f82 で直った)、`docs/reports/verify-2026-10-22-seat-wip-checkpoint.md`(pass。LOW V1・V2、INFO V3・V4)、`docs/reports/test-2026-10-22-seat-wip-checkpoint.md`(pass。G1〜G6。G1・G2 は 62f92f42 で直り、末尾の Addendum に書いた)
- この run が変えたのは、`docs/tech-debt/README.md`、`docs/evidence/org-spec-evidence-2026-10-21.md`、test report(末尾の Addendum だけ)、計画の Progress checklist、この report、insight event。コード、hook、skill、rules、仕様、先行 3 report の既存の行には触っていない
- 差分の大きさ(`git diff c90d01b4...HEAD --shortstat`、この run の前): 43 ファイル、+1708/-283。walkthrough を書くかどうかは `/pr` が決める

## Summary

`c90d01b4...62f92f42` の差分は、pane に足す値の一覧 `driver.SeatPaneEnv()` に `RALPH_WIP_CHECKPOINT=off` を 2 つ目として足し、ralph が作る herdr の pane と launch_cmd の起動スクリプトに付ける。2 つの WIP hook は、その値のとき `wip:` のコミットだけを飛ばす。leader の雛形の手順 14 は push の前に `wip:` のコミットを探して BLOCKED を上げ、運用規律に spawn の失敗の 1 項が入る。62f92f42 は /test の G1・G2 を埋めるテストで、コードは変えていない。

文書はこの branch の S4・S5 がすでに合わせていて(`git-commit-strategy.md`、`/org`・`/director` の skill の 4 写し、recipe の 2 写し、README、仕様、根拠の表、tech-debt、試行の evidence の訂正の注)、verify が突き合わせ済みである。この run は verify が残した V1・V2 を直し、62f92f42 が出した前からの振る舞い(detached HEAD の WIP コミット)を tech-debt に行にし、F11 の RESOLVED の注のテストの説明を今のテストに合わせ、test report に Addendum を足した。

## Changes made

| File | Change |
|------|--------|
| `docs/tech-debt/README.md`(318 行、F3 の RESOLVED のコメント) | V1。引いているテスト名 `TestOrgRuntimes_HerdrPaneEnvIsSelfNameOff` は残し、そのあとに `(renamed TestOrgRuntimes_HerdrPaneEnvIsSeatPaneEnv in fix/seat-wip-checkpoint, when the list of pane values grew a second entry)` を足した。#248 の時点の記録としての名前は変えず、今の名前で grep できる。リポジトリで旧名が残るのはこの行と過去の report だけ(`grep -rn` で確かめた) |
| `docs/tech-debt/README.md`(327 行、F11 の RESOLVED のコメント) | hook のテストの説明を「main and a clean tree never commit」から、「main and master never commit, whatever the value; a clean tree never commits; on a detached HEAD, which the hooks read as a feature branch, an unset value commits and `off` does not」に直し、新しい行の頭の文を指す参照を足した。テストの内容は 62f92f42 の `tests/test-wip-checkpoint-hooks.sh` の `case_commits`(unset・`on`・`OFF`・空と detached の unset)、`case_off`(feature と detached)、`case_no_commit`(main、master、きれいな木)から読み、156 件が通ることを打って確かめた。detached HEAD で `on`・`OFF`・空を打つ行はないので、書き方を「unset ならコミット、`off` ならしない」に絞った |
| `docs/tech-debt/README.md`(344 行、新規) | 「The session-end and pre-compaction WIP hooks make a `wip:` commit on a detached HEAD」の行。hook が `git rev-parse --abbrev-ref HEAD` で枝を読み、detached HEAD では `HEAD` が返って `main\|master\|unknown` に当たらないので feature ブランチ扱いになること、rule の「only on feature branches」と合わないこと、`off` では飛ばすこと、対話セッションの影響(reflog からしか辿れない commit、rebase の途中は未確認)、この PR が足さなかった理由(計画の Non-goals)、払う契機(枝の判定の次の変更)を書いた。計画のパスは `docs/plans/active/...` で、`verify.local.sh` の検査(`check_tech_debt_plan_refs`)と同じ式で存在を確かめた |
| `docs/evidence/org-spec-evidence-2026-10-21.md`(30 行、F11 の行) | V2。tech-debt の欄を `"~~The session-end WIP hook commits in a feature org's worktree~~"(RESOLVED の行)` にした。F3・F4・F8・F9・F10・F12・F14 と同じ形で、引いた句は tech-debt の 328 行の頭と一致する |
| `docs/reports/test-2026-10-22-seat-wip-checkpoint.md`(145〜152 行、新規の Addendum) | 62f92f42 が G1・G2 を埋めたこと(hook のテスト 108 → 156 件、master と detached HEAD の行、起動スクリプトを端末の値なしで打つ subtest)、W16 と O3 を sync-docs が打ち直して赤になったこと、G3〜G6 が残ること、ほかの mutant を打ち直していないことを足した。Verdict と既存の行は変えていない(diff は末尾の追加のみ) |
| `docs/plans/active/2026-10-22-seat-wip-checkpoint.md`(133〜135 行) | 「Review artifact created」「Verification artifact created」「Test artifact created」を `[x]` にし、それぞれの report と結果を書いた。「PR created」は `[ ]` のまま |
| `docs/insights/events/2026-10-22-seat-wip-checkpoint.jsonl` | sync_docs の event を 1 行足した(cycle 1、pass) |
| `docs/reports/sync-docs-2026-10-22-seat-wip-checkpoint.md` | 新規 |

足した文にホームのパス、token はない。`git diff --check` は rc 0、差分の U+FFFD は 0 件。

## Surfaces checked for drift

| Surface | 結果 | 根拠 |
|---------|------|------|
| `AGENTS.md`、`templates/base/AGENTS.md`、`.ralph/core/AGENTS.core.md` | 変更なし | WIP の hook、`SessionEnd`・`PreCompact` を書いた文がない。repo map の `.claude/hooks/` の行は hook の一覧を持たない |
| `README.md` | 変更なし | この branch の差分は launch_cmd の文に 2 つ目の値を足した 1 か所だけ(258 行)。環境変数の一覧の節はなく、`RALPH_WIP_CHECKPOINT` を足す先がない。利用者向けの説明は `git-commit-strategy.md` が持つ |
| `.claude/rules/ralph/git-commit-strategy.md`(と template の写し) | 変更なし。下の Left open | Org Runtime Commits と End-of-Session の 2 節は、ralph が付ける pane、launch_cmd の起動スクリプト、`--env` のない herdr の warning、手順 14 の `wip:` の確かめで揃う。「Only on feature branches (never on main/master)」の行と detached HEAD の食い違いは、新しい tech-debt の行が持つ |
| `docs/quality/quality-gates.md`、`docs/quality/definition-of-done.md`、`docs/architecture/repo-map.md` | 変更なし | WIP の hook、`RALPH_WIP_CHECKPOINT`、`tests/test-wip-checkpoint-hooks.sh` を名指す文がない。repo-map の `tests/` の行は `tests/test-*.sh` と一般に書き、新しいテストは `verify.local.sh` が拾う(verify の報告) |
| `.codex/README.md`(と template の写し)、`.codex/hooks.json` | 変更なし | Codex の `SessionEnd`・`PreCompact` は配線しないままで、`.codex/README.md` の 134 行の hook 名の文も今の hook の名前と合う。2 つの README は `cmp` で同じ |
| `.claude/settings.json`、`internal/cli/migrate.go` の hook の一覧 | 変更なし | hook のファイル名と配線は変わっていない(中身だけ) |
| `/org`・`/director` の skill 4 写し × 2、`director-launch-cmd.md` の 2 写し | 変更なし | verify が全部を突き合わせ済みで、`check-skill-sync.sh` と `check-sync.sh` が rc 0。`/org` skill の差分は前提の節(23 行付近)と手順 6(888 行付近)の 2 hunk だけで、tech-debt の F-8 の行が引く stop・disband の cell には触れていないので、長さの数字は変わらない |
| `docs/specs/2026-10-07-org-multi-org-director.md`、根拠の表(`docs/evidence/org-spec-evidence-2026-10-21.md`) | 前者は変更なし、後者は 30 行の 1 か所 | 仕様の印の数(26/6、表は 25/6)と PR の一覧(32 本)は verify が HEAD の a79b0f82 で数えた。この run は仕様に触れていない |
| `docs/tech-debt/README.md` の引用 | 新しい 3 つの句が一意 | 327 行の頭、328 行の頭、新しい 344 行の頭を `grep -F` で数えた。F11 の行の頭の句は 328 行の 1 行だけに当たり、新しい行の頭の句は 344 行(と 327 行の参照)に当たる。327 行の参照は、191 行のコメントが 328 行の頭の句を引くのと同じ形 |
| tech-debt の file size の行 | 変更なし | `internal/org/prompts_test.go`(2069 行)、`director_cmd_test.go`(1957 行)、`herdr_test.go`(1332 行)の行数を引く行はない。前からどれも 800 行を超えているので、今回は行を足していない |
| CHANGELOG と release のノート | 変更なし | repo に CHANGELOG はない。計画の Rollout(効くのは次の release の binary と `ralph upgrade` で入る hook から)は PR の本文の仕事 |
| `./scripts/check-sync.sh`、`./scripts/check-skill-sync.sh` | rc 0、rc 0 | 下の Verification |

## Decisions on the reports' remaining items

| 項目 | 決めたこと |
|------|------------|
| verify の V1(LOW)tech-debt 318 行の旧テスト名 | 直した(上の 318 行の行) |
| verify の V2(LOW)根拠の表 F11 の tech-debt の欄 | 直した(上の 30 行の行)。同じ行の「直す PR」の欄は、verify の指摘どおり前文の決まりに合うので変えていない |
| verify の V3(INFO)Org Runtime Commits の段落が `--env` のない herdr の例外を書かない | 直さない。「(see there)」で End-of-Session の節に送っていて、両節は食い違わない。detached HEAD の件と合わせて、`git-commit-strategy.md` を次に触るとき(新しい tech-debt の行の契機)に見直せばよい |
| verify の V4(INFO)Progress の「Review artifact created」 | 直した。あわせて Verification と Test も `[x]` にした |
| test の G1・G2 | 62f92f42 で直り、Addendum に書いた。G1 を埋めるテストが detached HEAD の前からの振る舞いを見つけ、tech-debt の行にした |
| test の G3〜G6 | 文書の話ではない。G4(`/org` skill の手順 6 の文と雛形の手順 14 の文を突き合わせるテストがない)は、tech-debt の行を立てていない(test report が LOW として残している) |
| verify の Coverage gaps の提案(`ralph org gate` が `origin/<default>..HEAD` の `wip:` を断る検査) | 提案のまま。この PR の範囲の外で、tech-debt の行も立てていない。下の Left open |

## Plan drift notes

- 計画の AC1〜AC9 の checkbox はすべて `[x]`。Progress に sync-docs の項はなく、`[ ] PR created` が最後。digest の範囲の外なので触ってよく、触ったのは 3 つの artifact の項だけ
- 計画の Non-goals「人が自分で打つ対話セッションの WIP コミットの振る舞い。`RALPH_WIP_CHECKPOINT` を設定しない限り今までどおり」が、detached HEAD の行を足さなかった根拠になっている(新しい tech-debt の行が引く)
- 62f92f42 は /test のあとのテストだけの commit(`tests/test-wip-checkpoint-hooks.sh` と `internal/org/director_cmd_test.go` の 2 ファイル)。コードと hook は a79b0f82 のまま。/test の Addendum に範囲を書いた。cycle は 1 のまま

## Left open

- fix/seat-wip-checkpoint の PR 番号は PR を作るまで分からない。仕様の Status、根拠の表の F11 の行(`org-spec-evidence-2026-10-21.md` の 30 行の「直す PR」の欄)、tech-debt の 191・327・328 行は、ブランチ名で書いている。直前の系列(#249 が #248 を置き換えた)と同じく、次の PR が番号に置き換える
- `.claude/rules/ralph/git-commit-strategy.md` の「Only on feature branches (never on main/master)」は、detached HEAD で hook がコミットすることと合わない。hook の判定を変える契機が来るまで、新しい tech-debt の行(344 行)が持つ。rule の文は直していない(対話セッションの振る舞いを変える話で、この PR の Non-goals)
- 手順 14 の `wip:` の確かめは雛形の指示で、ralph の検査ではない。leader が従うかは確かめていない(test の G6)。verify が挙げた最小の検査(`ralph org gate` が `origin/<default>..HEAD` に件名 `wip:` のコミットを見つけたら断る)は、この PR の範囲の外で、tech-debt の行にしていない。立てるかどうかは orchestrator の判断
- mutant のうち、62f92f42 のあとに打ち直したのは W16 と O3 だけ。hook のテストの書き方が変わったので、W1〜W15 が今も赤かは未確認(test の Addendum に書いた)
- G3(`ralph org start --plan` の leader の pane の `--env` を CLI から見るテストがない)、G4、G5(実機の確かめを打ち直していない)、G6 は /test が LOW・INFO で残したまま
- `unknown` の枝(`git rev-parse` が失敗するとき)を打つテストの行はない(test の G1 の残り)

## Verification

| 確かめたこと | 結果 |
|--------------|------|
| `./scripts/check-sync.sh`(編集のあと) | rc 0。`PASS: all files in sync.`(IDENTICAL 171、DRIFTED 0、ROOT_ONLY 0、TEMPLATE_ONLY 11、KNOWN_DIFF 5) |
| `./scripts/check-skill-sync.sh` | rc 0。`14 skill(s) in lock-step` |
| `./scripts/plan-visual.sh digest`(計画) | `b98f2806ab31`(承認の digest と同じ) |
| tech-debt の計画の参照(`verify.local.sh` の `check_tech_debt_plan_refs` と同じ式) | 欠けた参照 0 |
| `bash tests/test-wip-checkpoint-hooks.sh`(62f92f42 の tree) | rc 0、156 passed、0 failed |
| W16 の打ち直し(hook 4 本とテストを scratchpad に写し、session_end(root)の `main\|master\|unknown` を `main\|unknown` に) | 赤。154 passed、2 failed(master の値なしと `off`) |
| O3 の打ち直し(`go test -overlay` で `director_cmd.go` の `export` の行を落とす) | 赤。`TestDirectorLaunchScript_ChecksInThenExecsTheAgent/the_terminal_passes_neither_value` だけが落ちた。worktree のファイルには書いていない |
| `gofmt -l internal/org/director_cmd_test.go`、`bash -n` と `shellcheck tests/test-wip-checkpoint-hooks.sh` | 出力なし、rc 0 |
| `git diff --check`、差分の U+FFFD | rc 0、0 件 |
| 足した行の走査(ホームのパス、token) | 漏れなし |

Go のテスト全体と `run-verify.sh` は走らせていない(この run が変えたのは文書だけ。Go のコードは /test が走らせた tree と同じで、62f92f42 が足したテストの全体の実行は orchestrator の範囲)。

## Verdict

- Verdict: pass(sync-docs に stop condition はない)
- Not verified: 手順 14 と運用規律を本物の leader が守るか、AC6 の実機の確かめの再現、W1〜W15 とほかの mutant の 62f92f42 のあとの結果、Go のテスト全体の 62f92f42 での再実行

## Insight event

`./scripts/insights-append.sh --slug seat-wip-checkpoint --flow standard --phase sync_docs --verdict pass --cycle auto --source skill` で `docs/insights/events/2026-10-22-seat-wip-checkpoint.jsonl` に 1 行足した(cycle 1)。self-review・verify・test の event と同じファイルで、日付またぎの食い違いはない。
