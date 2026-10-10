# sync-docs report: org-inbox-escalate

## Cycle 1

- Date: 2026-10-11(JST)
- Plan: `docs/plans/active/2026-10-10-org-inbox-escalate.md`(承認済み、digest `b09718e048cd`。編集の前後で同じ値。本文は触っていない)
- Pipeline cycle: 1(上限 2)。差分は origin/main(`382c18c8`)から branch HEAD `8ed32d16`(feat/org-inbox-escalate)まで
- 先行 report(それぞれを追加した commit):
  `docs/reports/self-review-2026-10-10-org-inbox-escalate.md`(`6d803bed`、`39cec1b0` で再実行の結果に更新。Merge 可、CRITICAL 0・HIGH 0・MEDIUM は M2 の 1 件、LOW は開いたもの 7 件)、
  `docs/reports/verify-2026-10-10-org-inbox-escalate.md`(`0c8fb404`。pass、K-1〜K-4 と V-1)、
  `docs/reports/test-2026-10-10-org-inbox-escalate.md`(`983ed93b`、E13 の追記は `8ed32d16`。pass、Test gaps あり)
- 内容の commit: `30d7158c`(台帳・skill・仕様・規則)。この report と insight event は、その次の commit に入れる

## Summary

古くなっていたのは、tech-debt の org-feature-worktree の行の (f) と (j)、`/org` skill の 453 行目の幅と `ralph.toml` が読めない escalate の記述、仕様 FR-5 の 5 段目の注記だった。この計画の tech-debt の行は台帳になかったので 2 行足した。README と AGENTS.md はコードと合っていて変更していない。`agent-messaging.md` には、`ralph org escalate` が同じ検証を通すことを 1 段落足した。Go のコードは触っていない。

差分の大きさ(`git diff origin/main...HEAD --shortstat`、`30d7158c` 時点、この report の commit を含まない): 28 files changed, 5389 insertions(+), 109 deletions(-)。`30d7158c` 単体は 8 files changed, 100 insertions(+), 34 deletions(-)。walkthrough は書いていない(`/pr` が決める)。

## Changes made

| File | Change |
|------|--------|
| `docs/tech-debt/README.md`(org-feature-worktree の小さな指摘の行。K-1、M2) | (f) と (j) を、Debt・Impact・Why deferred・Trigger の 4 列で閉じた。この台帳の閉じ方(`~~元の文~~ Closed in <branch> (<commit>): ...`、Impact は `(closed)`、Trigger は `Done in <branch> (commit ...)`)に合わせた。直したのは feat/org-inbox-escalate の `b0fc4c59`(S4)で、`internal/org/prompts_test.go` の `squashSpace` と `containsPhrase`、`TestRenderRolePrompt_Leader_FeatureOrgProcedure` の 3 つの pin(「手順 1 の」の句、`--id` の文、`--state-dir` の文)が対象。Trigger が提案した「`strings.Fields` を空白 1 つで結ぶ」ではなく、空白をすべて除く形で直したことを書いた(テンプレートは日本語の文字の間でも折るので、空白 1 つでは合わない)。`TestContainsPhrase_IgnoresLineBreaksAndIndentation` が helper を固定していることも書いた。Why deferred に「(f) と (j) は、同じテストを書き換えた feat/org-inbox-escalate の中で直った」を足し、Related に計画の active パスを足した。(g) と (h)・(i)・(k)・(l) は開いたまま |
| `docs/tech-debt/README.md`(新しい行 1。K-3 の L4・L5・L8・N2・N3 の残り・test report の参照メモ) | org-inbox-escalate の小さな指摘を 1 行にした(self-review の L4・L5・L8、再実行の N2・N3、test report の参照メモ)。(a)(b) L4(`Body` が 2 つの意味、`Escalate` が 2 つ)、(c) L5(`internal/cli/org.go` が 1,646 行から 2,131 行。`newOrgEscalateCmd` から `yesNo` までを `org_inbox.go` へ)、(d)(e)(f) L8(`manifestLockTimeout` の名前と doc、`Org.Inbox` の doc、`RunWatch` が `EscalationsPathIn` を使わないこと。`watch.go` の `RunWatch` が今も `filepath.Join(p.StatusDir, EscalationsRelName)` であることを読んで確かめた)、(g) N2(文字列の並び、`orgInboxHintStateDir` の二重の解決、`shellQuote` と `shellQuoteIfNeeded` の差)、(h) N3 の残り(`PrintableInboxText` の置き場所、`newOrgEscalateFallbackRuntime` の 3 行の写し。写しの `EscalationsPath` の配線は `f347842d` のテストが固定するようになったことも書いた)、(i) 追記の open の失敗でエラー文にパスが 2 回出ること。行はコードを関数名・テスト名で指し、`file:line` は使っていない。行数の数字(1,646、2,131)は測った commit を書いた |
| `docs/tech-debt/README.md`(新しい行 2。test report の Test gaps) | 別々のプロセスから同時に打つ escalate の確認は、ロックを外した mutant を 8 並列で 4 回に 1 回ほどしか落とせず(X3 は 0 回)、感度が低い。プロセスをまたぐロックの保証は goroutine のテストと、manifest と同じ flock であることに頼っている、と書いた。直し方の案: 子プロセスで `inbox.lock` を `syscall.Flock` で持たせ、`ralph org escalate` と `inbox ack` が待つことを見る(`TestInboxLock_IsSeparateFromTheManifestLockAndSerializesWrites` の形)。トリガーは重複した ID か失われた ack の報告、または `withFileLock` の次の変更 |
| `.claude/skills/org/SKILL.md` と 3 つの写し(K-2、L7) | 機能ごとの org の節の `--state-dir` の段落で、104 列だった行(`pane に渡さないので、leader の側では当てにできない。付けないと leader の座席が別の台帳に入り、予約と上限`)を折り直した。後ろの行と合わせて 3 行に組み直した(表示幅 68〜75、前後の行は 71〜75)。編集は `.claude/skills/org/SKILL.md` に 1 回だけ行い、`./scripts/sync-skills.sh` で `.agents/skills/org/SKILL.md` を作り、2 つを `templates/base/` の 2 つに `cp` した |
| 同上(K-4) | `escalate` の行と「受信箱」節の「受信箱に書けなかったとき」の項に、`ralph.toml` を読めない escalate も NOT RECORDED(stderr の表示とデスクトップ通知、終了コード 1)になること、そのとき人に届くのは検査に通ったメッセージだけで、断られたメッセージは何も書かないことを足した。「escalate が何も記録せずに終了コード 1 で返ったとき」の項の例の括弧にも `ralph.toml` を読めない場合を足した(雛形と `--help` の文言に合わせた) |
| 同上(K-4 の例) | 「受信箱」節に、メッセージのファイルの置き場所と書き方の項を足した。`.harness/state/` の下に置く(git が無視する)、新しい worktree にはないことがあるので先に `mkdir -p .harness/state`、書くのは Write ツール(Codex では apply_patch)か `<<'EOF'` の heredoc、`echo '...'` と `--text '...'` は使わない(エラー文の `'` で引用符が閉じる)。文言は `internal/org/prompts/leader.md` の「件を上げる」の節に合わせた。動詞の表の `escalate` の例は `--text "$(cat blocked.txt)"` から `--text "$(cat .harness/state/escalate-X.txt)"` にし、行の中に「ファイルの置き場所と書き方は「受信箱」節」と足した |
| `docs/specs/2026-10-07-org-multi-org-director.md`(FR-5 の 5 段目の注。K-4) | 「受信箱に書けなかった escalate も」を「受信箱に書けなかった escalate と、`ralph.toml` が読めない escalate も」にし、読めないときに人へ届くのは検査に通ったメッセージだけであることを括弧で足した |
| `.claude/rules/ralph/agent-messaging.md` と `templates/base/` の写し(verify の参考の 1 つ目) | 「Escalation to the org inbox」の節を足した。`ralph org escalate --text` は agmsg のメッセージではなく受信箱に上げる経路なので星型の規則は当たらないこと、`ralph org send` と同じ message shape・`Parse`・`Validate`・2,000 文字の上限で `--raw` の抜け道がないこと、TYPE は `QUESTION`・`BLOCKED`・`RESULT` の 3 つで `BLOCKED` と `RESULT` は TASK_ID が要ること、断られたら終了コード 1 で何も書かないこと。`ralph org escalate --help` と leader の雛形がこの規則を message の形の参照先にしているので、規則の側が escalate に触れていない状態を直した。verify は 8a 段目で決めればよいとしたが、参照されている文書なので今入れた |
| `docs/insights/events/2026-10-10-org-inbox-escalate.jsonl` | `sync_docs` の event を 1 行足した(`--slug org-inbox-escalate`、cycle 1、verdict pass)。既存の event ファイルへの追記で、新しいファイルはできていない(`git status --short` で確認) |

## Surfaces checked for drift

| Surface | 結果 |
|---------|------|
| `README.md` の org の動詞の表(124 行)と「Org runtime」節の escalate の段落(251 行前後) | escalate・inbox・`wait --inbox` の説明はコードと合っている。`ralph.toml` が読めない場合の記述は粒度が細かいので足していない。変更なし |
| `AGENTS.md` の repo map の `internal/org/` の行 | `inbox.go` と `escalate.go` が入っている。変更なし |
| `docs/quality/` | escalate と inbox を述べた箇所なし(`grep -i -E 'escalat\|inbox'` は quality-gates の watchdog の行だけに当たり、無関係)。変更なし |
| `docs/recipes/` | `codex-seat-permissions.md` の escalate は権限の話で無関係。変更なし |
| `.claude/rules/ralph/agent-messaging.md` | 上の表のとおり 1 節足した。root と `templates/base/` は `cmp` で同一、`check-sync.sh` は DRIFTED 0 |
| `/org` skill の stop と disband の表のセル(tech-debt の F-8 の行が長さを数えている) | 触っていない。差分は `escalate` の行、受信箱の節、機能ごとの org の節の 1 段落だけ |
| `/org` skill の 4 面 | `cmp` で同一 |
| 計画の Progress checklist(verify の V-1) | 今の HEAD では `Review artifact created` と `Verification artifact created` と `Test artifact created` が `[x]` で、V-1 は `495aded1` で直っている。`PR created` は `/pr` の分。計画は触っていない |

## Found but left

- test report の Test gaps のうち、台帳に入れたのはプロセスをまたぐ escalate の 1 件だけ(依頼の範囲)。残り(本物の osascript は動かしていない、`appendLocked` の open・write・close の失敗と `Inbox == nil` の防御と CLI の受信箱の読み込みの失敗はテストが通らない、herdr の上の leader の実機は未確認)は、test report の Test gaps にだけある。台帳に足すかは lead の判断
- tech-debt の新しい行 1 の (a) の `body` の JSON key は、リリースに入ると変えにくい。self-review が「いまのうちに決める」としたもので、台帳の Trigger に「受信箱を出荷する次のリリースの前」と書いた。決めるのは lead
- cross-review の結果で code の fix が入ると、行数の数字(`internal/cli/org.go` の 2,131 行)は動く。行は測った commit を書いてあるので、そのときは数字だけ測り直す
- 計画の本文は触っていない(進捗の記録は lead)

## Checks run

| Command | Result |
|---------|--------|
| 編集の前の anchor の数(scratchpad の `anchors.py`) | skill 5 つ、仕様 1 つ、台帳 11 つ+Related の 1 つ+追記の 1 つが、すべて 1 回だけ出た。台帳の 1 つは最初 2 回出たので、直前の語まで延ばして 1 回にした |
| `./scripts/sync-skills.sh` | `done: 13 skill(s) mirrored to .agents/skills` |
| `cmp .claude/skills/org/SKILL.md` と、`.agents/`、`templates/base/.claude/`、`templates/base/.agents/` の 3 つ | すべて同一 |
| `./scripts/check-skill-sync.sh` | `13 skill(s) in lock-step` |
| `./scripts/check-sync.sh` | PASS(DRIFTED 0、ROOT_ONLY 0、TEMPLATE_ONLY 11、KNOWN_DIFF 5)。規則の `cp` のあとにも実行 |
| 追加した skill の行の表示幅(全角を 2 と数える) | 最大 77(`width.py`。既存の近隣の行は 71〜76) |
| tech-debt の表の形(scratchpad の `tdcheck.py`、`git show HEAD:` と比べた) | 変わった既存の行は 193 だけ、足した行は 199 と 200。3 行とも 7 つの区切り(5 列)、列ごとのバッククォートと `~~` は偶数、`file:line` の参照なし。`docs/plans/(active\|archive)/` の参照はすべて存在する |
| `./scripts/plan-visual.sh digest docs/plans/active/2026-10-10-org-inbox-escalate.md` | `b09718e048cd`(`- Approved:` の行と一致) |
| `HARNESS_VERIFY_MODE=static RALPH_VERIFY_SCOPE=full ./scripts/run-verify.sh`(`30d7158c` の内容の編集がすべて終わった状態) | rc 0、「All verifiers passed.」。`gofmt: ok`、`0 issues.`、tech-debt の plan 参照 OK、`check-skill-sync` 13 skill。分岐の secret scan は `382c18c8..8ed32d16` が clean(その時点で commit していなかったので、今回の commit は範囲に入っていない)。evidence は `docs/evidence/verify-2026-10-10-170435.log`(時刻は UTC。gitignore の対象) |
| `go test ./internal/org/ -run 'TestRenderRolePrompt\|TestSendDefaults\|TestContainsPhrase' -count=1` | ok(`TestSendDefaults_DocsMatchConstant` が skill の 4 面を読む) |
| `go test ./internal/scaffold/ ./internal/upgrade/ ./internal/config/ ./internal/cli/ -count=1` | 4 パッケージとも ok(templates の変更が scaffold と upgrade のテストに触れないことの確認) |

## Not verified

- `ralph.toml` が読めない escalate の記述は、`internal/cli/org.go` の `newOrgEscalateCmd` と `newOrgEscalateFallbackRuntime` を読んで、verify の K-4 と self-review の L1 に合わせたもの。本物の `ralph` で壊れた `ralph.toml` を置いて打ってはいない(test report の `TestOrgEscalate_BrokenConfig_AlertsTheHumanAndExitsOne` が固定する)
- agent-messaging の新しい節の「同じ `Parse` と `Validate`、`--raw` なし」は、`internal/org/escalate.go` の `validateEscalation`(`protocol.Parse`、`protocol.Validate(m, protocol.DefaultMaxBodyChars)`、`escalateTypes` の順)と `newOrgEscalateCmd` の flag(`--text` だけで `--raw` がない)を読んで確かめた。実際に打ってはいない(test report の `TestEscalate_RefusalsWriteNothingAndNotifyNoOne` と `TestOrgEscalate_RefusalsWriteNothing` が固定する)
- この report と insight event を入れる commit を含む range の secret scan は、commit の前には流せない。commit のあとに `./scripts/secret-scan-branch.sh --strict` を流し、結果は lead への報告に書く
- この report を足したあとの `run-verify.sh` の再実行は、結果を lead への報告に書く(この report は実行の前に書いたので、結果の行は持たない)
