# sync-docs report: org-plan-approval

## Latest run: cycle 2 (HEAD b75f6f93)

- Date: 2026-10-13(JST)
- Plan: `docs/plans/active/2026-10-12-org-plan-approval.md`(承認済み、digest `951cf09bd7bc`。今回の変更は Progress checklist の Notes への追記と、AC・checklist の checkbox の印だけで、digest は変わらない)
- Pipeline cycle: 2(上限は既定の 2 で、これが最後の run)。差分の base は `225dff27`、branch は `feat/org-plan-approval`。この run が始まった時点の HEAD は `b75f6f93`(`HEAD:` の行は run 開始時の値で、この report を含む commit の SHA ではない)
- 先行 report:
  `docs/reports/self-review-2026-10-13-org-plan-approval.md`(cycle 2 の 3 回目、Merge 可、CRITICAL・HIGH・MEDIUM 0、LOW 6)、
  `docs/reports/verify-2026-10-13-org-plan-approval.md`(cycle 2 の 2 回目、pass。「Documentation drift」の D6〜D12)、
  `docs/reports/test-2026-10-13-org-plan-approval.md`(cycle 2 の 2 回目、pass、失敗 0)
- 項目の番号: D6〜D12 は verify の cycle 2 の「Documentation drift」の表の番号、C2-N は self-review の番号。D1〜D5 は付録の cycle 1 の番号と同じ
- 内容の commit: この report、insight event、直した文書を含む 1 つの commit(`docs: sync docs for org-plan-approval (cycle 2)`)

### Summary

verify の D6〜D12 を、文書だけで直した。Go のコード、テスト、`--help` の文字列、leader の雛形(`internal/org/prompts/leader.md`、go:embed)は触っていない。

- D6(C2-2): template の `docs/quality/quality-gates.md` の「Feature plan approval」の行が、古い `PLAN:` の digest を直す条件を「whose current version is approved」のままにしていた。root は 3563129d で「whose current version is its latest approval and passes the check, never to an earlier approved version」に変わっている。`ralph init` が配る面なので、この句だけを root と合わせた。この行は `scripts/check-sync.sh` の KNOWN_DIFF(ファイルごと照合の外)なので、rc 0 のままでは気づけない
- D7: `/org` skill の「機能の計画の承認」の 2 項目を直した(4 面)。(a) 古い `PLAN:` の案内の項目は、入力を「承認し直す前の digest」とだけ書いていた。承認が 1 回だけの計画に打ち間違えた digest の TASK も、同じ枝(今の digest が最新の承認で、先頭の 2 行が未記入)に入る(`checkTaskPlanApproved` の doc が edee787f でこの入力を挙げた)。(b) send の項目は、宛先の座席の手が空くのを待ったあと、打つ前にもう一度照合することを書いていなかった。コードで確かめた点: 照合の 2 回目は `AgentWait` の後、`PaneSendText` の前にあり(`Send` の本体)、`--dry-run` は待つ手前で戻るので 1 回だけ照合する
- D8: 仕様 FR-7 の「7 段目の前半で決めたこと」の注に、send は宛先の座席が空くのを待つので、待ったあと打つ前に承認をもう一度照合する(待つ間に作業中の座席が計画を書き換えうるため)、という 1 文を足した
- D10・D11: tech-debt の README に、self-review cycle 2 の LOW を反映した。C2-6・C2-7・C2-10 は「Small findings of org-plan-approval」の 1 行にまとめた。C2-8・C2-9 は N4 の行の Impact・Why deferred・Trigger・Related の列に足した
- D9・D12: 計画の Notes に 3 行足した(edee787f、self-review cycle 2 の 3 回分、/test cycle 2 と Q07・Q08 を 6a7d35b0 で閉じたこと、/verify の 2 回、2 回目の検査で断られる TASK は herdr の `agent wait` のあとになること、この sync-docs)
- D5(cycle 1 から持ち越し): AC1〜AC14 と、Progress checklist の「Review artifact created」「Verification artifact created」「Test artifact created」に印を付けた。「PR created」は付けていない(/pr の担当)

### Changes made

| File | D / C | Change |
|------|-------|--------|
| `templates/base/docs/quality/quality-gates.md:80`(「Feature plan approval」の行) | D6 / C2-2 | 「of a plan whose current version is approved」を、root `docs/quality/quality-gates.md:81` と同じ句「of a plan whose current version is its latest approval and passes the check, never to an earlier approved version」にした。行のほかの部分は変えていない(template と root で元から違う KNOWN_DIFF) |
| `.claude/skills/org/SKILL.md`(「機能の計画の承認」の send の項目と、案内の項目の 2 つ目)と、`.agents/skills/org/SKILL.md`、`templates/base/.claude/skills/org/SKILL.md`、`templates/base/.agents/skills/org/SKILL.md` の同じ箇所 | D7 | send の項目に「send は宛先の座席の手が空くのを待ち、待ったあと、打つ前にもう一度照合する(`--dry-run` は待たないので 1 回だけ)。待つ間に計画が変わっていれば、打たずに同じように断り、何も記録しない」を足した。案内の項目は、入力を「承認し直す前の digest か、承認が 1 回だけの計画に打ち間違えた digest」にした。ほかの項目は変えていない。折り返しは表示幅 76 桁以内(最長 76) |
| `docs/specs/2026-10-07-org-multi-org-director.md:72`(FR-7「7 段目の前半で決めたこと」) | D8 | 「(改訂すれば digest か全文のハッシュが変わり、承認し直すまで断る)。」の直後に、「send は宛先の座席の手が空くのを待つので、待ったあと、打つ前に承認をもう一度照合する(待つ間に作業中の座席が計画を書き換えうるため)。」を足した。ほかの文は変えていない |
| `docs/tech-debt/README.md`(N4 の行、「The org-plan-approval tests keep their approval helpers twice」) | D11 / C2-8, C2-9 | Impact に「再承認の手書きが `approval_enforce_test.go` に 4 か所」と「Progress のメモの編集 6 か所、計画の本文の編集 8 か所(6a7d35b0 が 2 か所足した)」を足した。Why deferred に cycle 2 の分(C2-5、C2-8、C2-9)も上限のため残したことを足した。Trigger に `enforceFixture.reapprove` と、計画の形を変えるときに編集の helper を 1 つにすることを足した。Related に C2-5・C2-8・C2-9 の場所と、コミット 3563129d・edee787f・6a7d35b0 を足し、既存の「N4 in its run-2 appendix」を「N4 in its cycle 1 run-2 appendix」にした。Debt の列は変えていない |
| `docs/tech-debt/README.md`(新しい行、末尾) | D10 / C2-6, C2-7, C2-10 | 「Small findings of org-plan-approval; none changes behavior」の 1 行を足した。(a) `Send` の本体のコメントの「budget check right after that wait」が、承認の確かめ直しが間に入って古い。(b) 折り返しが途中で止まったコメント 2 か所(`TestCheckTaskPlanApproved_StaleOrMistypedDigest` の doc と `Send` の doc)。(c) `TestOrgSend_ApprovalCheck_LatestApprovalIsPerPlanPath` の (b) の 2 つ目の assertion が 1 つ目より先に落ちない。C2-2 は直したので行に入れていない。関数名とテスト名で指し、行番号は使っていない |
| `docs/plans/active/2026-10-12-org-plan-approval.md`(`## Progress checklist` の `### Notes` の末尾) | D9・D12 | 3 行足した(上の Summary)。計画のほかの部分は変えていない |
| `docs/plans/active/2026-10-12-org-plan-approval.md`(AC1〜AC14、checklist の 3 項目) | D5 | `[ ]` を `[x]` にした。digest は変わらない(`951cf09bd7bc`) |
| `docs/insights/events/2026-10-13-org-plan-approval.jsonl` | - | `sync_docs` の event を 1 行足した(`--slug org-plan-approval`、`--cycle auto`) |
| この report | - | cycle 2 の節を先頭に置き、cycle 1 の内容は「付録: cycle 1」に残した |

skill の 4 面は、`.claude/skills/org/SKILL.md` を Edit で直し、`./scripts/sync-skills.sh` で `.agents/skills` を作り直し、`templates/base/` の 2 つには `.claude/skills/org/SKILL.md` と `.agents/skills/org/SKILL.md` を `cp` で写した(`sync-skills.sh` は `templates/base/` を作り直さない)。写す前に、4 面が同じ sha256(`e9e4b1d5…`)であることを確かめた。

### 印を付けた AC と、付けたかどうか

- AC1〜AC14 に `[x]` を付けた。AC1〜AC10、AC13、AC14 は verify の cycle 2 の判定(コードとテストの対応を読んで「満たす」)による。AC5 の「herdr も呼ばずに」は、呼んだ時点で拒否の理由がある TASK について成り立つと読む(2 回目の検査で断られる TASK は `agent wait` を 1 回呼んだあとになる)。この読みは計画の Notes に書いた。AC10 は逸脱 2 の読みによる(Notes に既にある)
- AC11: 付けた。/test の cycle 2 の 2 回目で、`go test ./... -count=1` は top-level 1,504 本が PASS、0 fail。前の実行から消えたテストは 0 で、足したのは 6a7d35b0 の 2 本(subtest を含め 4 つ)だけ。書き換えたテストの本数と理由は計画の Notes(AC11 の行)にある
- AC12: 付けた。この sync-docs の編集のあと、`check-sync.sh` と `check-skill-sync.sh` はどちらも rc 0(下の Verification)
- 付けなかったもの: 「PR created」(/pr の担当)

### Surfaces checked for drift

| Surface | 結果 |
|---------|------|
| `templates/base/docs/quality/quality-gates.md:80` と root `docs/quality/quality-gates.md:81` | 案内の句が同じになった(`grep -c` で同じ句が各 1 件)。行のほかの部分は KNOWN_DIFF。`check-sync.sh` は引き続き rc 0 |
| `/org` skill の 4 面のほかの箇所(動詞表の `spawn`・`send`・`approve`(168、169、180 行)、案内の項目のうち直した 1 つを除く分) | コードと合う。変更なし。4 面は `cmp` で一致 |
| `.claude/rules/ralph/agent-messaging.md` と template の写し | `cmp` で一致。「`ralph org send` refuses it, with nothing recorded」は 2 回目の検査にも成り立つ。待ったあとの確かめ直しは、この rule が書く粒度(拒否と何も記録しない)の内側にあるので、足さない |
| README(`:124` の verb 表、`:258` の承認の段落) | 「a plan changed after its approval is refused until it is approved again」は、いつ照合するかを言わない書き方で、2 回目の検査と矛盾しない。変更なし |
| AGENTS.md の repo map(`internal/org/` の行が `approve.go` と `approval.go` を挙げる) | cycle 2 は新しいファイルを足していない。変更なし |
| `docs/quality/definition-of-done.md`、`docs/architecture/repo-map.md` | 承認の検査の言い方はない(`/plan` の承認の話だけ)。変更なし |
| `internal/org/prompts/leader.md`(go:embed。触らない) | 手順 3〜7 は、拒否の目印の後ろの次の手に従う形で、待ったあとの確かめ直しで断られたときにも当てはまる。2 回目の検査のことは書かないが、書かなくても誤らない。ずれなし |
| `ralph org send --help`(`newOrgSendCmd`、Go の文字列。触らない) | 「refused with nothing recorded unless …」は 2 回目の検査にも成り立つ。待ったあとの確かめ直しは書かない。ずれとは言えない |
| 仕様 FR-7 の注の「TASK の拒否は何も記録しない(6 段目の後半の見張りの拒否と同じ)」 | 2 回目の拒否にも成り立つ。D8 の 1 文と矛盾しない |
| tech-debt の README の plan 参照(`docs/plans/(active\|archive)/…`) | すべて存在する(`check_tech_debt_plan_refs` と同じ検査で 0 件欠け)。新しい行は計画を `docs/plans/active/2026-10-12-org-plan-approval.md` と書き(`/pr` の `archive-plan.sh` が archive に書き直す)、`archive/` は先に書いていない。N4 の行は計画のパスを持たない。表の行は 5 列(エスケープしていない `\|` は 6 本)で崩れていない |
| skill の 4 面を読むテスト(`internal/org/send_defaults_sync_test.go` ほか) | `go test -count=1 ./internal/org/ -run 'Defaults\|SyncSkill\|SendDefaults'`、`./internal/config/`、`./internal/scaffold/` は ok。`./scripts/check-template-purity.sh` は rc 0 |

### Left open

- 付録の cycle 1 の D4(計画の A5・AC6・Verify plan が書く仕様の行番号「107・108 行」「108 行」が今は 109・110 行): 計画の本文は digest の対象なので直さない。記録だけ
- C2-6・C2-7・C2-10 は Go のコメントとテストの書き方なので、この run では直さず tech-debt の行にした(上限が 2 で、`.go` の編集は /self-review からの回し直しになる)
- `ralph org send --help` と leader の雛形は、待ったあとの確かめ直しに触れない。どちらも書かなくて誤らないので、ずれとして扱わない
- verify の Coverage gaps(2 回目の検査のバイナリでの確認、leader の図解ページの実機、Linux)は、文書では埋まらない
- release のノートと PR の本文は、この run の範囲に含めていない(/pr)

### Verification

コミットの前に、worktree のルートで打った。

- `./scripts/check-skill-sync.sh`: rc 0、`13 skill(s) in lock-step`
- `./scripts/check-sync.sh`: rc 0、`IDENTICAL: 167`、`DRIFTED: 0`、`ROOT_ONLY: 0`、`TEMPLATE_ONLY: 11`、`KNOWN_DIFF: 5`、`PASS: all files in sync.`
- `./scripts/plan-visual.sh digest docs/plans/active/2026-10-12-org-plan-approval.md`: `951cf09bd7bc`(`- Approved:` の行と同じ。Notes と印を足したあとも変わらない)
- `shasum -a 256` で skill の 4 面(`.claude/skills/org`、`.agents/skills/org`、`templates/base/.claude/skills/org`、`templates/base/.agents/skills/org`): 4 面とも `9869f963cddcbf38be70ec8f5b7b33612ac769ffc35b1f1c77cc94be3280ff9e`(直す前は `e9e4b1d5469c…`)
- `grep -n 'current version is' docs/quality/quality-gates.md templates/base/docs/quality/quality-gates.md`: root の 81 行と template の 80 行が、どちらも「whose current version is its latest approval and passes the check, never to an earlier approved version」を含む
- `git diff --stat`(report と insight の行を足す前): 8 ファイル、+68/-48。skill の 4 面は各 +11/-7、template の quality-gates と仕様は各 1 行、tech-debt は 1 行直して 1 行足し、計画は Notes +3 行と印 17 か所
- `go test -count=1 ./internal/org/ -run 'Defaults|SyncSkill|SendDefaults'`、`go test -count=1 ./internal/config/ ./internal/scaffold/`: すべて ok。`./scripts/check-template-purity.sh`: rc 0

`./scripts/run-verify.sh` は打っていない。この run が変えたのは文書(skill の 4 面、template の quality-gates、仕様、tech-debt、計画の Notes と印)、report、insight の行だけである。

## 付録: cycle 1 (HEAD 71e42662 の時点)

cycle 1 の report の本文を、そのとき書いたまま残す。`## Latest run` の見出しを `## 付録` に改め、小見出しの名前に「(cycle 1)」を付けた。本文の D1〜D5 は、verify の cycle 1 の表の番号である。「cycle 2 の直しで扱う」と書いた D2(`--help`)は 5e809ac6 で直り、D5(印)は上の cycle 2 で付けた。

- Date: 2026-10-13(JST)
- Plan: `docs/plans/active/2026-10-12-org-plan-approval.md`(承認済み、digest `951cf09bd7bc`。Progress checklist の Notes に 1 行足しただけで、値は変わらない)
- Pipeline cycle: 1。差分の base は `225dff27`、branch は `feat/org-plan-approval`。この run が始まった時点の HEAD は `71e42662`(`HEAD:` の行は run 開始時の値で、この report を含む commit の SHA ではない)
- 先行 report:
  `docs/reports/self-review-2026-10-13-org-plan-approval.md`(Merge 可)、
  `docs/reports/verify-2026-10-13-org-plan-approval.md`(pass。「Documentation drift」の D1〜D5)、
  `docs/reports/test-2026-10-13-org-plan-approval.md`(pass)
- 項目の番号: D1〜D5 は verify の「Documentation drift」の表の番号と同じ
- 内容の commit: この report、insight event、直した文書を含む 1 つの commit(`docs: sync docs for org-plan-approval`)

### Summary(cycle 1)

verify が挙げた D1 と D3 を、文書だけで直した。Go のコード、テスト、`--help` の文字列は触っていない。

D1 は `/org` skill の動詞表の `approve` の行である。approve が受信箱を読んだあとにほかの誰かが件を resolve していたときの括弧は、「打ち方は示さず、承認は記録してあると言う」で止まっていた。今のエラー(`resolveApproved`、`internal/org/approve.go`)は、そのあとに「その人が直しを求めたかもしれないので、leader への承認の DECISION はまだ送らず、先に `ralph org inbox show <ID>` で resolve の note を読む」とも言う。括弧にこの続きを足し、skill の 4 面を同じにした。

D3 は仕様 FR-7 の注である。要件は「TASK の本文の `PLAN:`」と書くが、実装(`protocol.HeaderValues`)はヘッダの塊の 1 行だけを読み、空行のあとの本文の行は数えない。注に、要件をヘッダの 1 行と読んだことを 1 文足した。

D2 は `--help` の文字列なので、cycle 2 の直しで扱う。D4 と D5 は記録だけである。

### Changes made(cycle 1)

| File | D | Change |
|------|---|--------|
| `.claude/skills/org/SKILL.md:180`(動詞表の `approve`)と `.agents/skills/org/SKILL.md`、`templates/base/.claude/skills/org/SKILL.md`、`templates/base/.agents/skills/org/SKILL.md` の同じ行 | D1 | 括弧の「承認は記録してあると言う」を「承認は記録してあると言い、その人が直しを求めたかもしれないので、leader への承認の DECISION はまだ送らず、先に `ralph org inbox show <ID>` で resolve の note を読むよう言う」に直した。エラー文(`resolveApproved`)の 3 つの要素(承認は記録済み、DECISION はまだ送らない、先に `inbox show` で note を読む)と同じ順にした。ほかの文は変えていない |
| `docs/specs/2026-10-07-org-multi-org-director.md:72`(FR-7「7 段目の前半で決めたこと」) | D3 | 「TASK の `PLAN:` のヘッダが 2 行あれば断る(…)。」の直後に、「上の要件の「TASK の本文の `PLAN:`」は、ヘッダの塊の 1 行 `PLAN: <path> sha256:<digest>` と読んだ(`protocol.HeaderValues` はヘッダの塊だけを読み、空行のあとの本文の `PLAN:` の行は数えない)。」を足した。ほかの文は変えていない |
| `docs/plans/active/2026-10-12-org-plan-approval.md`(`## Progress checklist` の `### Notes` の末尾) | - | sync-docs の cycle 1 の結果を 1 行足した。計画のほかの部分は変えていない。AC と checklist の印は付けていない(cycle 2 で付ける) |
| `docs/insights/events/2026-10-12-org-plan-approval.jsonl` | - | `sync_docs` の event を 1 行足した(`--slug org-plan-approval`、`--cycle auto`)。既存のファイルへの追記 |
| この report | - | 新規 |

skill の 4 面は、`.claude/skills/org/SKILL.md` を Edit で直し、`./scripts/sync-skills.sh` で `.agents/skills` を作り直し、`templates/base/` の 2 つには `.claude/skills/org/SKILL.md` と `.agents/skills/org/SKILL.md` を `cp` で写した(`sync-skills.sh` は `templates/base/` を作り直さない)。写す前に、`templates/base/` の 2 つが HEAD の root と `cmp` で一致することを確かめた。

### Surfaces checked for drift(cycle 1)

| Surface | 結果 |
|---------|------|
| `/org` skill の 4 面で、受信箱の件がほかで resolve されたときの扱いを書く箇所(`resolve していた`、`ほかの誰か` で grep) | 動詞表の `approve` の行だけ。D1 で直した。ほかの節(`機能の計画の承認`、人の側の手順)はこの場合に触れない |
| README、AGENTS.md、`docs/quality/quality-gates.md`、`internal/org/prompts/leader.md`、`internal/cli/org.go` で同じ場合を書く箇所 | なし(README の `already resolved` は別の話) |
| `approve` と `[org.approval]` と `plan_approved` を挙げる文書(`git grep` で `internal/`、`docs/reports`、`docs/plans`、`docs/evidence`、skill を除いた全部) | `.claude/rules/ralph/agent-messaging.md` と template の写し(`cmp` で一致)、AGENTS.md の repo map(`approve.go` と `approval.go` を挙げる)、README の verb 表と段落、`docs/quality/quality-gates.md` と template の写し(承認の行は同じ。`check-sync.sh` の KNOWN_DIFF だけが違う)、`scripts/ralph-config.sh` と template の写し(`cmp` で一致)、`templates/base/ralph.toml`、仕様。どれもコードと合っていて、変更なし |
| `docs/architecture/repo-map.md`(`:47` の `/org` skill、`:73` の `.harness/state/org/`、`scripts/` の一覧) | 承認はこの 3 か所の言い方に収まる(`plan_approved` は manifest の event で、新しい script も skill もない)。変更なし |
| `docs/tech-debt/README.md` の 213〜221 行 | 220 行目は approve のエラー文と承認の効き方の食い違い(承認は DECISION なしで効く)を書く。D1 は文書をエラー文に合わせただけで、行の内容は変わらない。変更なし |
| 4 面の skill を読むテスト(`internal/org/send_defaults_sync_test.go` の `<N>ms` の照合ほか) | 括弧の変更は照合する語に触れない。`go test -count=1 ./internal/org/ -run 'Defaults|SyncSkill|SendDefaults'`、`./internal/config/`、`./internal/scaffold/` は ok |

### Left open(cycle 1)

- D2(`ralph org spawn --help` と `ralph org send --help` の文字列、`internal/cli/org.go` の `newOrgSpawnCmd` と `newOrgSendCmd`): どちらも見張りの拒否は書くが、per-plan の implementer の spawn と TASK の拒否、leader のいない org の spawn の拒否は書かない。`internal/` の文字列なので、この run では触らない。cycle 2 の直しで扱う
- D4(計画の A5・AC6・Verify plan が書く仕様の行番号「107・108 行」「108 行」が、今は 109・110 行): 計画の本文は承認の digest の対象なので直さない。記録だけ
- D5(計画の Progress checklist の `- [ ] Review artifact created` と、AC1〜AC14 のチェックボックス): 今回は付けない。cycle 2 で付ける
- `/org` skill の人の側の手順 4(「approve は leader に知らせない。出力に出た `ralph org send` のコマンドを打って、DECISION … を leader に送る」)は、approve が成功した通常の経路を書く。件がほかで resolve されていたときは approve が終了コード 1 で終わり、エラー文が DECISION を送らず note を先に読むよう言う。この例外は動詞表の括弧(D1)に書いたので、手順 4 には足していない。手順 4 に 1 文足すかどうかは、cycle 2 で決めてよい
- 仕様の Security considerations(`director も leader もこの検証を迂回できない`)の節そのものには、FR-7 の注への参照がない(verify は任意とした)。この run では足していない

### Verification(cycle 1)

コミットの前に、worktree のルートで打った。

- `./scripts/check-skill-sync.sh`: rc 0、`13 skill(s) in lock-step`
- `./scripts/check-sync.sh`: rc 0、`DRIFTED: 0`、`ROOT_ONLY: 0`、`TEMPLATE_ONLY: 11`、`KNOWN_DIFF: 5`、`PASS: all files in sync.`
- `./scripts/plan-visual.sh digest docs/plans/active/2026-10-12-org-plan-approval.md`: `951cf09bd7bc`(`- Approved:` の行と同じ。計画を変える前と後で変わらない)
- `shasum -a 256` で skill の 4 面(`.claude/skills/org`、`.agents/skills/org`、`templates/base/.claude/skills/org`、`templates/base/.agents/skills/org`): 4 面とも `b0b9547ee547…`
- `git diff --stat`(report と insight の行を足す前): 6 ファイル、+6/-5。skill の 4 面は各 1 行、仕様 1 行、計画 +1 行
- `go test -count=1 ./internal/org/ -run 'Defaults|SyncSkill|SendDefaults'`、`go test -count=1 ./internal/config/ ./internal/scaffold/`: すべて ok

`./scripts/run-verify.sh` は打っていない。この run が変えたのは文書(skill の 4 面、仕様、計画の Notes)、report、insight の行だけである。
