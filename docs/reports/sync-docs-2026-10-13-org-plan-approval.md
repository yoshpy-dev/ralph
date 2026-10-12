# sync-docs report: org-plan-approval

## Latest run: cycle 1 (HEAD 71e42662)

- Date: 2026-10-13(JST)
- Plan: `docs/plans/active/2026-10-12-org-plan-approval.md`(承認済み、digest `951cf09bd7bc`。Progress checklist の Notes に 1 行足しただけで、値は変わらない)
- Pipeline cycle: 1。差分の base は `225dff27`、branch は `feat/org-plan-approval`。この run が始まった時点の HEAD は `71e42662`(`HEAD:` の行は run 開始時の値で、この report を含む commit の SHA ではない)
- 先行 report:
  `docs/reports/self-review-2026-10-13-org-plan-approval.md`(Merge 可)、
  `docs/reports/verify-2026-10-13-org-plan-approval.md`(pass。「Documentation drift」の D1〜D5)、
  `docs/reports/test-2026-10-13-org-plan-approval.md`(pass)
- 項目の番号: D1〜D5 は verify の「Documentation drift」の表の番号と同じ
- 内容の commit: この report、insight event、直した文書を含む 1 つの commit(`docs: sync docs for org-plan-approval`)

### Summary

verify が挙げた D1 と D3 を、文書だけで直した。Go のコード、テスト、`--help` の文字列は触っていない。

D1 は `/org` skill の動詞表の `approve` の行である。approve が受信箱を読んだあとにほかの誰かが件を resolve していたときの括弧は、「打ち方は示さず、承認は記録してあると言う」で止まっていた。今のエラー(`resolveApproved`、`internal/org/approve.go`)は、そのあとに「その人が直しを求めたかもしれないので、leader への承認の DECISION はまだ送らず、先に `ralph org inbox show <ID>` で resolve の note を読む」とも言う。括弧にこの続きを足し、skill の 4 面を同じにした。

D3 は仕様 FR-7 の注である。要件は「TASK の本文の `PLAN:`」と書くが、実装(`protocol.HeaderValues`)はヘッダの塊の 1 行だけを読み、空行のあとの本文の行は数えない。注に、要件をヘッダの 1 行と読んだことを 1 文足した。

D2 は `--help` の文字列なので、cycle 2 の直しで扱う。D4 と D5 は記録だけである。

### Changes made

| File | D | Change |
|------|---|--------|
| `.claude/skills/org/SKILL.md:180`(動詞表の `approve`)と `.agents/skills/org/SKILL.md`、`templates/base/.claude/skills/org/SKILL.md`、`templates/base/.agents/skills/org/SKILL.md` の同じ行 | D1 | 括弧の「承認は記録してあると言う」を「承認は記録してあると言い、その人が直しを求めたかもしれないので、leader への承認の DECISION はまだ送らず、先に `ralph org inbox show <ID>` で resolve の note を読むよう言う」に直した。エラー文(`resolveApproved`)の 3 つの要素(承認は記録済み、DECISION はまだ送らない、先に `inbox show` で note を読む)と同じ順にした。ほかの文は変えていない |
| `docs/specs/2026-10-07-org-multi-org-director.md:72`(FR-7「7 段目の前半で決めたこと」) | D3 | 「TASK の `PLAN:` のヘッダが 2 行あれば断る(…)。」の直後に、「上の要件の「TASK の本文の `PLAN:`」は、ヘッダの塊の 1 行 `PLAN: <path> sha256:<digest>` と読んだ(`protocol.HeaderValues` はヘッダの塊だけを読み、空行のあとの本文の `PLAN:` の行は数えない)。」を足した。ほかの文は変えていない |
| `docs/plans/active/2026-10-12-org-plan-approval.md`(`## Progress checklist` の `### Notes` の末尾) | - | sync-docs の cycle 1 の結果を 1 行足した。計画のほかの部分は変えていない。AC と checklist の印は付けていない(cycle 2 で付ける) |
| `docs/insights/events/2026-10-12-org-plan-approval.jsonl` | - | `sync_docs` の event を 1 行足した(`--slug org-plan-approval`、`--cycle auto`)。既存のファイルへの追記 |
| この report | - | 新規 |

skill の 4 面は、`.claude/skills/org/SKILL.md` を Edit で直し、`./scripts/sync-skills.sh` で `.agents/skills` を作り直し、`templates/base/` の 2 つには `.claude/skills/org/SKILL.md` と `.agents/skills/org/SKILL.md` を `cp` で写した(`sync-skills.sh` は `templates/base/` を作り直さない)。写す前に、`templates/base/` の 2 つが HEAD の root と `cmp` で一致することを確かめた。

### Surfaces checked for drift

| Surface | 結果 |
|---------|------|
| `/org` skill の 4 面で、受信箱の件がほかで resolve されたときの扱いを書く箇所(`resolve していた`、`ほかの誰か` で grep) | 動詞表の `approve` の行だけ。D1 で直した。ほかの節(`機能の計画の承認`、人の側の手順)はこの場合に触れない |
| README、AGENTS.md、`docs/quality/quality-gates.md`、`internal/org/prompts/leader.md`、`internal/cli/org.go` で同じ場合を書く箇所 | なし(README の `already resolved` は別の話) |
| `approve` と `[org.approval]` と `plan_approved` を挙げる文書(`git grep` で `internal/`、`docs/reports`、`docs/plans`、`docs/evidence`、skill を除いた全部) | `.claude/rules/ralph/agent-messaging.md` と template の写し(`cmp` で一致)、AGENTS.md の repo map(`approve.go` と `approval.go` を挙げる)、README の verb 表と段落、`docs/quality/quality-gates.md` と template の写し(承認の行は同じ。`check-sync.sh` の KNOWN_DIFF だけが違う)、`scripts/ralph-config.sh` と template の写し(`cmp` で一致)、`templates/base/ralph.toml`、仕様。どれもコードと合っていて、変更なし |
| `docs/architecture/repo-map.md`(`:47` の `/org` skill、`:73` の `.harness/state/org/`、`scripts/` の一覧) | 承認はこの 3 か所の言い方に収まる(`plan_approved` は manifest の event で、新しい script も skill もない)。変更なし |
| `docs/tech-debt/README.md` の 213〜221 行 | 220 行目は approve のエラー文と承認の効き方の食い違い(承認は DECISION なしで効く)を書く。D1 は文書をエラー文に合わせただけで、行の内容は変わらない。変更なし |
| 4 面の skill を読むテスト(`internal/org/send_defaults_sync_test.go` の `<N>ms` の照合ほか) | 括弧の変更は照合する語に触れない。`go test -count=1 ./internal/org/ -run 'Defaults|SyncSkill|SendDefaults'`、`./internal/config/`、`./internal/scaffold/` は ok |

### Left open

- D2(`ralph org spawn --help` と `ralph org send --help` の文字列、`internal/cli/org.go` の `newOrgSpawnCmd` と `newOrgSendCmd`): どちらも見張りの拒否は書くが、per-plan の implementer の spawn と TASK の拒否、leader のいない org の spawn の拒否は書かない。`internal/` の文字列なので、この run では触らない。cycle 2 の直しで扱う
- D4(計画の A5・AC6・Verify plan が書く仕様の行番号「107・108 行」「108 行」が、今は 109・110 行): 計画の本文は承認の digest の対象なので直さない。記録だけ
- D5(計画の Progress checklist の `- [ ] Review artifact created` と、AC1〜AC14 のチェックボックス): 今回は付けない。cycle 2 で付ける
- `/org` skill の人の側の手順 4(「approve は leader に知らせない。出力に出た `ralph org send` のコマンドを打って、DECISION … を leader に送る」)は、approve が成功した通常の経路を書く。件がほかで resolve されていたときは approve が終了コード 1 で終わり、エラー文が DECISION を送らず note を先に読むよう言う。この例外は動詞表の括弧(D1)に書いたので、手順 4 には足していない。手順 4 に 1 文足すかどうかは、cycle 2 で決めてよい
- 仕様の Security considerations(`director も leader もこの検証を迂回できない`)の節そのものには、FR-7 の注への参照がない(verify は任意とした)。この run では足していない

### Verification

コミットの前に、worktree のルートで打った。

- `./scripts/check-skill-sync.sh`: rc 0、`13 skill(s) in lock-step`
- `./scripts/check-sync.sh`: rc 0、`DRIFTED: 0`、`ROOT_ONLY: 0`、`TEMPLATE_ONLY: 11`、`KNOWN_DIFF: 5`、`PASS: all files in sync.`
- `./scripts/plan-visual.sh digest docs/plans/active/2026-10-12-org-plan-approval.md`: `951cf09bd7bc`(`- Approved:` の行と同じ。計画を変える前と後で変わらない)
- `shasum -a 256` で skill の 4 面(`.claude/skills/org`、`.agents/skills/org`、`templates/base/.claude/skills/org`、`templates/base/.agents/skills/org`): 4 面とも `b0b9547ee547…`
- `git diff --stat`(report と insight の行を足す前): 6 ファイル、+6/-5。skill の 4 面は各 1 行、仕様 1 行、計画 +1 行
- `go test -count=1 ./internal/org/ -run 'Defaults|SyncSkill|SendDefaults'`、`go test -count=1 ./internal/config/ ./internal/scaffold/`: すべて ok

`./scripts/run-verify.sh` は打っていない。この run が変えたのは文書(skill の 4 面、仕様、計画の Notes)、report、insight の行だけである。
