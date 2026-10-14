# sync-docs report: org-director-attach

- Date: 2026-10-15(JST)
- Plan: `docs/plans/active/2026-10-14-org-director-attach.md`(承認済み。`./scripts/plan-visual.sh digest` は `27414f27525b` で、`- Approved:` の行と同じ。この run 前後で変わらない)
- Pipeline cycle: 1。差分の base は `ef575748`、branch は `feat/org-director-attach`。run 開始時の HEAD は `80da7892`(run 開始時の値で、この report を含む commit の SHA ではない)
- 先行 report: `docs/reports/self-review-2026-10-14-org-director-attach.md`(Merge: yes、CRITICAL・HIGH 0、MEDIUM 2、LOW 10。直しは 57c551ff)、`docs/reports/verify-2026-10-14-org-director-attach.md`(pass。文書のずれ V1〜V4)、`docs/reports/test-2026-10-14-org-director-attach.md`(pass、失敗 0)
- 項目の番号: V1〜V4 は verify の「Documentation drift」の表の番号。S6 は 57c551ff(self-review の直し)の slice の名前で、依頼の項目 3 がその slice で変わった `ralph status` の git の呼び出しを指す。「指示の 5」は依頼の項目 5(tech-debt の列の数)
- 内容の commit: この report、直した文書、Go の help と test の期待値、計画の Progress の 1 項目を含む 1 つの commit(`docs: sync docs for org-director-attach`)。insight event は、指示どおり記録していない

## Summary

依頼の 5 項目のうち、文書を直したのは V1(項目 1)と S6(項目 3)の 2 件、Go の help を直したのは V4(項目 2)の 1 件。V2(項目 4)と、tech-debt の列の数(項目 5)は直さない。

あわせて、見張りなしで使える動詞を挙げる文書を掃いて、`docs/quality/quality-gates.md` とその template に `director` の動詞を足した。`/org` skill の同じ一覧(1202 行)は、この PR が `director` を入れていた。

README、`docs/architecture/repo-map.md`、`/org` skill と `/director` skill の 4 面、`.claude/rules/ralph/` とその template、leader の雛形、`templates/base/ralph.toml`、`docs/recipes/`、tech-debt の変更行は、コードと合っていたので変えていない。skill は 1 字も触っていないので、`scripts/sync-skills.sh` は打っていない。

計画は、Progress checklist に 1 項目の note を足しただけで、digest は変わらない。

## Changes made

| File | 項目 | Change |
|------|------|--------|
| `AGENTS.md:90`(repo map の `internal/org/` の項) | V1 | 「`ralph org escalate` with its interim human notification (`escalate.go`)」を「`ralph org escalate` (`escalate.go`: with `[org.director].enabled` and a cross-org watch accepting work it only records the item, for the director and the ack deadline; otherwise it also sends it to the human at once)」に直した。「interim」は、director を登録できなかった 5 段目の形(全件をすぐ送る)を指していた。今は、すぐ送るのは `enabled = false` か見張りがないときの常設の経路で、既定は記録だけになる。分岐は `escalateDelivery`(`internal/org/escalate.go`)から取った。行の他の部分は変えていない。AGENTS.md は 1 行の書き換えだけで、行数は増えていない |
| `docs/specs/2026-10-07-org-multi-org-director.md:84`(FR-9 の 8a-1 の注) | S6 | `enabled` の読み方を述べた文(「leader はそれを書き換えられるため。」)の次に 1 文を足した。内容は、`ralph status` も `director:` の行のために `enabled` を同じ規則で読むので、`--state-dir` か `RALPH_ORG_STATE_DIR` で台帳を指すときは、その台帳が main worktree のものかを調べるローカルの `git rev-parse`(連結した worktree からは `git worktree list --porcelain` も)を、キャッシュが新しいかによらず打つこと。7c-2 の注の「origin の照合で打つのはローカルの `git remote get-url origin` だけ」は origin の照合の話で別であること。status が gh を打たず、ネットワークに出ないことは変わらないこと。7c-2 の注(74 行)は変えていない。判断の理由は、その一文が origin の照合を主語にしていて今も正しく、8a-1 の注が `enabled` の読み方を持つ場所だから。`README.md:125` と `ralph status --help` は 57c551ff で同じことを言っているので、仕様だけが言っていなかった |
| `docs/quality/quality-gates.md:80`、`templates/base/docs/quality/quality-gates.md:79`(「Cross-org watch gate」の行の On failure 欄) | 掃き出し | 「`stop`, `disband`, `escalate`, `approve`, `gate`, `merge-order`, and the inbox verbs work with no watch」に、`the director verbs` と、`director attach` は登録したうえで stderr に警告を出し、escalate はそのとき件をすぐ送る、という括弧を足した。`/org` skill の「見張りがなくても使える」一覧(1202 行)と、`wireOrgWatchGate` の「spawn、start、send だけが関門を呼ぶ」に合わせた。2 つのファイルは元から別の内容(root の Quality pipeline gate の行に「see docs/tech-debt」がある)なので、同じ 1 文を両方に足した |
| `internal/cli/org_director.go`(`ralph org director status` の `Long`)、`internal/cli/org_director_test.go`(`TestOrgDirectorHelp_DescribesTheRegistration` の status の行) | V4 | help は state の名前に `disabled` を挙げていたが、`enabled = false` のときの意味を言わず、登録の欄も「director が attached のとき」としか言わなかった。実際は `enabled = false` で登録が残っていると、state は `disabled`、行は `registration d1 is still recorded`、欄(`id` など)も出る(`printOrgDirectorReport`、`TestOrgDirector_StatusDisabledNamesALeftRegistration`)。help に「With enabled false the state is disabled whether or not a registration is recorded; one that is still recorded is named and its fields are shown.」を足した。help を固定するテストに同じ文を足した(`strings.Fields` で空白をそろえて比べるので、折り返しの位置は問わない)。動きは変えていない。1 回目の edit で字下げを 1 段多くして `gofmt -l` に指されたので、直して通した |
| `docs/plans/active/2026-10-14-org-director-attach.md`(`## Progress checklist`) | 記録 | `- [x] Test artifact created` の次に、この run の note を 1 項目足した(上の 4 件の要約)。ほかの節は触っていない。`plan-visual.sh digest` は `27414f27525b` のまま |

## Surfaces checked for drift

| Surface | 結果 |
|---------|------|
| `README.md`(動詞の表 124 行、`ralph status` の行 125、escalate と受信箱の段落、`watch --all` の段落、director の段落、approve の段落) | 変更なし。escalate の 3 つの届き方(理由 `inbox_no_watch`・`inbox_director_disabled`、delivery `deadline`)、NOTIFIED の `deferred`、approve が resolve しないこと、send を打ってから resolve すること、`director` の 3 つの動詞、`enabled = false` で attach が断られること、`ralph upgrade` が `enabled` を足さないことが、コードと合っている。125 行は `git rev-parse` と `git worktree list` の文を 57c551ff の時点で持つ |
| `AGENTS.md` の repo map | V1 のほかは変更なし。`director.go` の足しは、`director.jsonl` と `director.lock` の説明がコードと合っている |
| `docs/architecture/repo-map.md` | 変更なし。`.claude/skills/director/` の行が足されている。74 行の `.harness/state/org/` の行は manifest・saga・receipts・watchdog の状態だけを挙げ、`inbox.jsonl` や `splits/`、gate の記録も挙げない書き方なので、`director.jsonl` は足さない |
| `/org` skill(4 面)、`/director` skill(4 面) | 変更なし。`/director` は全文、`/org` は director・escalate・approve・inbox・watch の節を読み、`ORG ESCALATION RECORDED` の banner、3 つの理由、`delivery`、approve の send と resolve の順、見張りなしで使える動詞の一覧がコードと合うことを確かめた。`cmp` で `.claude/` と `.agents/` と template の 2 面が同一(`check-skill-sync.sh` は 14 skill が lock-step)。skill を触っていないので `sync-skills.sh` は打っていない |
| `.claude/rules/ralph/ralph-workflow.md` と template | 変更なし。skill の一覧と `/director` の 1 項目が両方にあり、同一 |
| `.claude/rules/ralph/agent-messaging.md` と template | 変更なし。escalate の節は検査(`Parse`、`Validate`、2,000 字、`--raw` なし、TYPE)だけを書き、届き方は書かない |
| `.claude/rules/ralph/` のほかの rule、`CLAUDE.md` | 変更なし。`/director` は自動で起動する skill で、`CLAUDE.md` の「手動の skill は `/release` だけ」と合う |
| `internal/org/prompts/leader.md` | 変更なし。受信箱の段落(3 つの届き方、`ORG ESCALATION RECORDED`)と手順 4 の言い換えがコードと合う |
| `templates/base/ralph.toml`、`scripts/ralph-config.sh` と template | 変更なし。`[org.director].enabled` のコメントと `RALPH_ORG_DIRECTOR_ENABLED` が、`Default()` の true と同じ値で lock-step |
| `docs/quality/definition-of-done.md` と template | 変更なし。org の記述がない |
| `docs/recipes/` | 変更なし。escalate・inbox・approve・director を扱う記述がない(`codex-seat-permissions.md` の「escalation」は Codex の承認の prompt) |
| `docs/specs/2026-10-07-org-multi-org-director.md` のほかの注 | 変更なし。55 行(5 段目、`inbox_no_director`)と 7 段目前半の注は、8a-1 の注を指す但し書き付きの履歴で、verify の AC8 が確かめた |
| `docs/tech-debt/README.md` 202 行(V2) | 変更なし。取り消し線の行で、`statusDeadmanMinutes` は、解決する前の姿の説明に出る名前。行は「RESOLVED 2026-10-12」を持ち、その関数が今あるとは言っていない。履歴なので残す |
| `docs/tech-debt/README.md` の列の数 | 変更なし。5 列でない行は、ef575748 でも今でも同じ 4 行(Left open)だけで、この PR が変えた行と足した行はどれも 5 列(backtick の外の `|` を数える script を scratchpad に置いた) |

## Left open

- Go を触った。`ralph org director status` の help の 1 文と、その help を固定するテストの期待値 1 つで、動きは変えていない。verify と test は触る前の Go を見ている。pipeline を回し直すかどうかは lead の判断で、前の cycle の sync-docs も同じ扱いだった。確かめたのは、help と `Director|Help` のテスト(go1.26.0 と go1.25.8)、`gofmt -l`、`go vet ./internal/cli/` だけで、cli と org の全体は打ち直していない
- tech-debt の列の数(指示の 5)。22 行(6 列)と 32 行(4 列)が、ef575748 の時点から他の行と違う。この PR の外なので直していない。同じ script は 125 行(0 列)と 160 行(4 列)も指した。この 2 行は列を区切る `|` が backtick の中や対にならない backtick にあるせいで数え損ねている可能性があり、本当に壊れているかは確かめていない。4 行とも ef575748 と今で出力が同じ(この PR が変えた行ではない)
- quality-gates に director の登録の行(「1 つ目の登録、detach の `id` の照合」)は足していない。D2・D4 のとおり、登録は権限を開かず escalate の動きも変えないので、この表の関門(拒否して何も記録しない決定的な検査)とは性質が違うと判断した。足すなら、`ErrDirectorAttached` と `ErrDirectorIDMismatch` の拒否を 1 行にする。手を入れていない
- `docs/architecture/repo-map.md` の `.harness/state/org/` の行は、上の理由で `director.jsonl` を足していない。ほかのファイルも挙げていないままの行を、director だけで太くしないため
- 計画の `PR created` は `/pr` が付ける。release のノート(`enabled` の既定 true で escalate の届き方が変わること、前の動きは `enabled = false`、`ralph upgrade` は `enabled` を足さないこと、approve は件を resolve しなくなったこと)は計画の Rollout に書いてあり、`/pr` と release の担当
- test report の Test gaps 1(畳み込みが違う `id` の `detached` を無視することを止めるテストがない)は、文書の話ではないので触っていない
- insight event は、指示でこの run は記録しない

## Verification

commit の前に、worktree のルートで打った。

- `./scripts/plan-visual.sh digest docs/plans/active/2026-10-14-org-director-attach.md`: `27414f27525b`(run の前も後も同じ)
- `gofmt -l internal`: 出力なし(1 回目は `internal/cli/org_director.go` を指した。edit の字下げの誤りで、直したあとの出力はない。`gofmt -w` は使っていない)
- `go vet ./internal/cli/`: 通過
- `go test ./internal/cli/ -run 'Director|Help' -count=1`(go1.26.0): `ok`(2 回目の実行は 6.9 s)
- `GOTOOLCHAIN=go1.25.8 go test ./internal/cli/ -run 'OrgDirectorHelp|Help' -count=1`: `ok`
- `go run ./cmd/ralph org director status --help`: 足した 2 行が `The registration does not tell ...` の前に出る
- `./scripts/check-skill-sync.sh`: `[ok] check-skill-sync: 14 skill(s) in lock-step`
- `./scripts/check-sync.sh`: `IDENTICAL: 169`、`DRIFTED: 0`、`ROOT_ONLY: 0`、`TEMPLATE_ONLY: 11`、`KNOWN_DIFF: 5`、`PASS: all files in sync.`
- `./scripts/run-static-verify.sh`(`scripts/ralph-config.sh` の変更で言語の範囲は full): rc 0、`==> All verifiers passed.`。`check-pipeline-sync`、`check-skill-sync`、`check-template-purity`(`PASS`)、`tech-debt README plan references`、`gofmt: ok`、`0 issues.`、`secret-scan-branch: scanned ef575748..80da7892 against origin/main: clean`(commit の前に打ったので、範囲は run 開始時の HEAD まで)。出力 425 行は scratchpad に落として読んだ。wrapper が書いたログ(`docs/evidence/verify-2026-10-14-195335.log`)は gitignore の対象で、コミットしていない
- `git diff --check`: 出力なし
- 変えた 7 ファイルの U+FFFD は 0 件
- 「inbox_no_director」「Until a director」「director を登録できない」「今後入る director」「interim」の grep(Go、md、toml、sh。history の `docs/reports/`、`docs/evidence/`、`docs/plans/` と `_test.go` を除く): 残りは互換の説明(`escalate.go:48`)、仕様 55 行の 5 段目の履歴、tech-debt 232 行の無関係な「interim manual path」だけ

## Verdict

Verdict: complete。文書と help のずれは、依頼の項目 1 から 3 で直した。項目 4 と 5 は、理由を書いて残した。静的検査(`run-static-verify.sh`、`check-skill-sync.sh`、`check-sync.sh`、`gofmt -l`、`git diff --check`)はすべて通り、help のテストは go1.26.0 と go1.25.8 で通った。計画の digest は変わっていない。確かめていないのは、cli と org のテスト全体の打ち直しと、Go を触ったあとの `/verify`・`/test`(Left open の 1 つ目)。
