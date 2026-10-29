# sync-docs report: leader-plan-depth

- Date: 2026-10-29
- Plan: `docs/plans/active/2026-10-29-leader-plan-depth.md`(issue #267。承認の digest `fac3038a3706` はこの run の前後で同じ。`./scripts/plan-visual.sh digest` で確かめた。この run が変えたのは AC1〜AC8 の checkbox と `## Progress checklist` の 4 行だけ)
- Pipeline cycle: 1(`.harness/state/standard-pipeline/cycle-count.json` の値)。base は `origin/main` の `7522835c`、branch は `feat/leader-plan-depth`。run 開始時の HEAD は `cc57bbeb`(この report を含む commit の SHA ではない)
- 先行 report: `docs/reports/self-review-2026-10-29-leader-plan-depth.md`(Run 5 が Merge: yes。LOW の L11 は Known gaps に回した)、`docs/reports/verify-2026-10-29-leader-plan-depth.md`(打ち直しが pass。LOW の V1・V6、情報の V2〜V8)、`docs/reports/test-2026-10-29-leader-plan-depth.md`(再実行が pass)
- 差分の大きさ(`git diff origin/main...HEAD --shortstat`、この run の前): 20 ファイル、+2555/-54。walkthrough を書くかどうかは `/pr` が決める
- 制約: 別の session が CPU を使う CI を回しているので、`go test`・`go build`・`golangci-lint`・`run-*.sh` は打っていない。打ったのは `check-sync.sh`、`check-skill-sync.sh`、`check-template-purity.sh`、`sync-skills.sh`、`plan-visual.sh digest`、`sh -n`、`git grep` で、`secret-scan-branch.sh --strict` は commit の前に打つ

## Summary

この branch は issue #267 の対応で、機能の org の leader が計画を書く前に AGENTS.md・`.claude/rules/`・計画が触るコードを読み、機能の内側の分かれ目を `FORK <番号>: <問い> / a <案>(推奨) / b <案>` の行で承認の QUESTION にすべて載せるようにした。`[org.review].plan_advisory` が true なら、承認の QUESTION の前(split では implementer の spawn の前)に、相手のモデルへ読むだけの計画助言を頼み、要約を `ADVISORY:` の行に書く。director は分かれ目を承認と一緒に人に聞き、推奨と違う案が選ばれたら approve を打たずに直しの DECISION を送る。

verify が渡した文書のずれ(V1、V6、V7 と、pane の `RALPH_PRIMARY_CLI` の用途、`ADVISORY:` の行)は、8 件とも直した。sweep で見つけた 4 件も直した: README の承認の段落、tech-debt の `count_triage_findings` の行番号、tech-debt の MCP の行(コードの leader の形が (a) を閉じていること)、tech-debt の 800 行の行(この branch の新しい大きなファイル)。Go のコード、計画の本文、`internal/org/prompts/leader.md`、skill の `director` は触れていない。

## Changes made

| File | Change |
|------|--------|
| `README.md:266` | 依頼 1。pane の `RALPH_PRIMARY_CLI=<driver>` の用途を、座席の cross-review だけでなく機能の org の leader の計画助言も `pick_reviewer` で相手の CLI を選ぶ、に直した。leader は変数がないとき自分の CLI を使うので「claude とみなす」は cross-review の話になる、と括弧で足した |
| `README.md:270` | sweep で見つけた分。`[org.approval].mode` の段落が per-plan の QUESTION のヘッダしか書かず、本文の `FORK` の行、director が承認と一緒に聞くこと、推奨と違う案のときの直しの DECISION と 2 往復目の `(決定済み)`、`plan_advisory` と `ADVISORY:` の行、split では推奨を採って `## Design decisions` に記録すること、を書いていなかった。README は動きを書く文書なので 1 か所にまとめた(計画は README の変更を求めていない。不要なら外せる) |
| `.claude/skills/org/SKILL.md:32-35` と写し 3 つ | 依頼 2。3 つ目の値の用途に、機能の org の leader の計画助言を足した |
| `.claude/skills/org/SKILL.md:43-45` と写し 3 つ | 依頼 2。`--env` のない herdr の影響。leader の計画助言は変数がなければ leader 自身の CLI で相手を選ぶので、取り違えるのは herdr の server の環境に `claude` か `codex` の同じ変数があって pane が引き継ぐときだけ、と足した(verify の「足すかどうかは /sync-docs」の項目) |
| `.claude/skills/org/SKILL.md:934-942` と写し 3 つ | 依頼 2。「機能の計画の承認」に、`plan_advisory` が on のときの計画助言(承認の QUESTION の前、split では implementer の spawn の前。1 回、最長 20 分。助言者の MCP の server は止め、止められなければ飛ばす。呼ぶ前後で git の状態が変われば BLOCKED)と、要約を `ADVISORY:` の行に書き director が人に伝えること(split では `## Design decisions`)を足した。`.claude/` を直して `templates/base/.claude/` に `cp` し、`.agents/` の 2 つは `./scripts/sync-skills.sh`(既定と `CLAUDE_ROOT=templates/base/.claude/skills CODEX_ROOT=templates/base/.agents/skills`)で作り直した。4 つの写しは md5 が同じ |
| `docs/recipes/isolated-herdr-server.md:245-248` と `templates/base/docs/recipes/isolated-herdr-server.md` | 依頼 3。README と同じ言い方に直した。2 つの写しは同じ文 |
| `docs/specs/2026-10-07-org-multi-org-director.md:130` | 依頼 4。#265 の段落(`:129`)は書き換えず、後ろに 2026-10-29(issue #267)の行を足した。leader が `plan_advisory` を読んで何をするか、director が何を聞くか、reviewer の `cross_review` は #269 で FR-7 の印は変えないこと、テストが固定した文と、実機の通しはしていないことを書いた |
| `scripts/ralph-config.sh:2-4`、`:11-21`、`:38`、`:44-46` と `templates/base/scripts/ralph-config.sh` | 依頼 5。冒頭の説明、「two live consumers」を「three」にして 3 つ目に feature org の leader の role prompt(ralph の repo の source で、binary に go:embed される)を足した。`RALPH_CLAUDE_REVIEWER_MODEL` と codex の 2 つの変数のコメントにも leader の計画助言を足した。コメントだけで、`sh -n` は通る。2 つの写しは `cp` |
| `.claude/rules/ralph/model-routing.md:74-78` と `templates/base/.claude/rules/ralph/model-routing.md` | 依頼 6。cross-review sync note に、feature org の leader の計画助言が同じ 3 つの変数を同じ fallback で読むことを足した(role prompt は scaffold した project にない、と括弧で書いた) |
| `.claude/rules/ralph/model-routing.md:152`(root だけ) | 依頼 6 の任意の項目。「Where the values live」に `internal/org/prompts_leader_plan_test.go` を足した。codex の fallback は `TestRenderRolePrompt_Leader_FeatureOrgPlanAdvisoryCodexTurnsMCPOff`(`RALPH_CODEX_REVIEWER_MODEL` と `RALPH_CODEX_REASONING_EFFORT`)、claude の fallback は `TestRenderRolePrompt_Leader_FeatureOrgPlanAdvisoryClaudeReadsOnly` が `scripts/ralph-config.sh` の値と突き合わせる。どちらも該当の関数を読んで確かめた |
| `scripts/xreview-helpers.sh:58-60`、`:64-70` と `templates/base/scripts/xreview-helpers.sh` | 依頼 7。`pick_reviewer` の見出しの呼び手に leader の計画助言を足した。「Both skills pass the driver they resolved」を「All three callers」にし、3 つ目の driver を leader 自身の CLI、参照先を `internal/org/prompts/leader.md`(ralph の repo の source)とした。コメントだけ。2 つの写しは `cp` |
| `docs/tech-debt/README.md:31`(`pick_reviewer` の行) | 依頼 8。Impact 欄の「the two callers」を「the three callers」にして一覧に leader の計画助言を足し、Trigger 欄の括弧に leader も入れた(Trigger は当たっていない。3 つの呼び手とも `claude`/`codex` に絞ってから渡す)。行は開いたまま。この run の `scripts/xreview-helpers.sh` の変更で `pick_reviewer` が 70-78 行から 74-82 行に動いたので、Debt 欄と Related 欄の `:70-78` は関数名の引用に直した |
| `docs/tech-debt/README.md:32`(`count_triage_findings` の行) | 前の sync-docs が「次に触るときに関数名の引用に直す」と残した行番号 `scripts/xreview-helpers.sh:86-` を、Debt 欄と Related 欄の両方で関数名の引用に直した(関数は base の 89 行目から、この run の変更で 94 行目に動く) |
| `docs/tech-debt/README.md:154`(codex reviewer の残りの危険の行) | Trigger 欄の終わりに Note を足した。feature org の leader の計画助言は 3 つ目の codex の呼び出しで、(a) の MCP を `--disable plugins`、server ごとの `enabled=false`、読み直した一覧が 0 のときだけ呼ぶ、で閉じている(codex-cli 0.163.0 で、モデルを呼ばずに確かめた)。(a) を閉じるときに使える形として残した。Related に計画の D7 を足した |
| `docs/tech-debt/README.md:360`(800 行を超えるファイルの行) | Trigger 欄に 2026-10-29 の carry note を足し、Related に 3 つの参照を足した。この branch は leader の雛形のテストを新しい `internal/org/prompts_leader_plan_test.go`(最初から 1179 行)に置き、`internal/org/director_skill_sync_test.go` を 795 から 897 行にした。`internal/org/prompts/leader.md` は 675 から 831 行(prompt で、コードではない)。`prompts_test.go`(2670)と行の 4 つのファイルは変わっていない。どの report もこの大きさを挙げていなかった。直さない理由は、行の既存の理由(動きと別の move-only の分割は計画の Scope にない)と同じ |
| `docs/plans/active/2026-10-29-leader-plan-depth.md` | AC1〜AC8 を `[x]` にした(verify が AC1〜AC5・AC7・AC8 を「満たす」と判定した。AC6 は verify が一部、/test の打ち直しで雛形のテストが通ったので、2 つを合わせて満たす)。`## Progress checklist` の「Review artifact created」「Verification artifact created」「Test artifact created」を `[x]` にし、「Sync-docs report created」の行を足した(前の計画の sync-docs と同じ)。本文の節は触れていない。digest は同じ |
| `docs/reports/sync-docs-2026-10-29-leader-plan-depth.md` | この report |
| `docs/insights/events/2026-10-28-leader-plan-depth.jsonl` | `sync_docs` の行を足した(下の「Insight event」) |

## Surfaces checked for drift

| Surface | 結果 | 根拠 |
|---------|------|------|
| `AGENTS.md`・`.ralph/core/AGENTS.core.md`・`templates/base/AGENTS.md` の org の行 | 変更なし | 機構(`approve.go`、`gate.go` など)の一覧で、leader の計画の中身は書かない。`AGENTS.md` は地図のままにする |
| `.claude/rules/ralph/agent-messaging.md`(と写し) | 変更なし | `ralph org send` と `escalate` が検査するもの(TYPE、TASK_ID、header、2,000 字、`PLAN:`)だけを書く。`FORK`・`ADVISORY:` は本文の行で、`CHECKS:`・`VERIFY:` と同じく役割の雛形と `/org` skill が持つ |
| `docs/quality/quality-gates.md:81`(Feature plan approval)と写し | 変更なし | 承認の機械的な検査(`plan_approved` の記録、digest、`- Approved:` の行)だけを書く。分かれ目と計画助言は prompt の指示で、ここには載せない |
| `docs/recipes/codex-setup.md:36`・`:99` と写し | 変更なし | 「claude を呼ぶ計画助言と cross-review が走らない」「`codex exec` を Bash tool から打つときの `</dev/null`」は、leader の codex の塊にも当てはまる一般の話で、偽ではない |
| `docs/recipes/codex-seat-permissions.md` | 変更なし | `--role reviewer` の probe だけを使う。この branch は leader の雛形の変更で、reviewer の mission は変えていない |
| `templates/base/ralph.toml:245-260` | 変更なし(S4 で直済み) | `[org.review]` のコメントが「機能の org の leader は `plan_advisory` を読み、reviewer は `cross_review` を後の版から」と言う。`ralph config get` の終了コードの扱いは leader の雛形(0・`unknown command "config"` の 1・127・ほか)と合う |
| `internal/config/config.go` の `OrgReviewConfig`、`internal/org/driver/paneenv.go` | 変更なし(直済み) | doc comment と警告文が leader の計画助言を言う。L8(「unset」は空を含まない)は main の取り込みで直済み |
| `.claude/skills/director/SKILL.md` と写し 3 つ | 変更なし | S3 が書いた「計画の承認」の手順 1〜7 が、leader の `FORK`・`(決定済み)`・`ADVISORY:` の形と合う(verify が確かめた)。4 つの写しは `cmp` で一致 |
| leader の雛形の手順番号を引く文書 | 変更なし | `git diff origin/main...HEAD -- internal/org/prompts/leader.md` に手順の番号の行(`^[-+]N. `)は出ない。tech-debt の「step 5」「steps 12 and 14」などは前のまま合う |
| `docs/tech-debt/README.md` の plan 参照 | 通った | `verify.local.sh` の `check_tech_debt_plan_refs` と同じ `grep -o -E 'docs/plans/(active\|archive)/[A-Za-z0-9._-]+'` を打ち、存在しない参照は 0 件。新しい参照は `docs/plans/active/2026-10-29-leader-plan-depth.md`(`/pr` の archive-plan.sh が書き換える) |
| 「later release」「後の版」の `git grep` | 残りなし | 現在形で leader がまだ読まないと言う文はない。残るのは reviewer の `cross_review`(#269)の話(`/org` skill の `:57`、`templates/base/ralph.toml:258`)と、別の話題 |
| 文を固定するテスト | なし | 直した skill・recipe・コメントの文を `tests/`・`internal/` で `git grep` した。`ralph-config.sh` を読むテスト(`defaults_sync_test.go`、`prompts_leader_plan_test.go`、`tests/test-codex-exec-invocation.sh`)は、行頭の `VAR="${VAR:-default}"` を読むだけで、コメントは対象外 |
| `internal/cli/config.go` の `ralph config get --help` | 変更なし(情報) | 127 の行はない(ralph 自身は 127 を返さない)。leader の雛形は「127 のほかは help の約束」と書く。足すなら `internal/cli/config_test.go` の固定も直す別の変更で、前の sync-docs が残した項目のまま |
| `docs/reports/`・`docs/plans/archive/`・`docs/evidence/` の過去の記録 | 洗っていない | 当時の状態の記録 |

## Decisions on the reports' remaining items

| 項目 | 扱い | 理由 |
|------|------|------|
| verify V1(spec の `:129` に #267 の行がない) | 直した | 上の `:130` |
| verify V6(tech-debt `:31` の「the two callers」) | 直した | 上の `:31`。Trigger は当たっていない |
| verify V7(`model-routing.md:148` が leader の fallback のテストを挙げない) | 直した | 上の `:152` |
| verify V4(AC の checkbox が `[ ]`) | 直した | 計画 |
| pane の `RALPH_PRIMARY_CLI` の用途(README、`/org` skill、recipe)と `/org` skill の `ADVISORY:` | 直した | 上の表 |
| verify V2(2 往復目で `(決定済み)` の分かれ目が推奨と違う案のとき、`approved` の DECISION が「すべて推奨で決まった」意味を持つとは言えない) | 残した(情報) | leader の手順 4 と `/org` skill(`:926-931`)の文は、どちらの場合も「`## Design decisions` を書き換えない」という動きは正しい。害はなく、文の直しは雛形(`leader.md`)の変更を伴うので /sync-docs の範囲を超える |
| verify V3(AC5 の `pick_reviewer "$RALPH_PRIMARY_CLI"` と雛形の `"${RALPH_PRIMARY_CLI:-<あなたの CLI>}"`) | 残した(情報) | self-review L5・L7・L9・L10 の直しで広がったもので、Progress に記録がある。計画の本文は digest を変えないため直さない |
| verify V5(issue #267 の文と D1・D3 の違い) | PR 本文に回す | 下の「Notes for reviewers」 |
| verify V8(計画の本文の古い S2・D5・AC4・AC5 の文) | 残した(情報) | 計画の Progress の `:180` に読み替えが書いてある。本文は digest のため直さない |
| self-review L11(run の表に codex の leader で `claude` の行がない) | Known gaps に回す | test report が直す 1 行を確かめている(`{driver.PrimaryCLIEntry("claude"), "codex", "codex"}` の行) |

## Plan drift notes

- 計画の本文(Scope の S2、D5、AC4・AC5、Test plan、Edge cases)の古い動き(`unknown command "config"` の 1 だけを off、`pick_reviewer "$RALPH_PRIMARY_CLI"`、変数のない pane は claude とみなす)は、Progress の `:180` の読み替えのとおり。雛形はその読み替えのとおりで、verify が 36 通りで確かめた。
- 計画の本文の S2 の「手順 1 の最後に置く」は、`### 計画助言` の小見出し(手順 16 の後ろ)に読み替える(Progress の self-review Run 1 の直し)。
- この run では計画の本文を変えていない。

## Release notes

(PR 本文の「リリースノート」の材料。`ralph.toml` は seed なので、`ralph upgrade` は既存の project の `ralph.toml` に `[org.review]` の表を足さない。#265 の release のノートと合わせて読む。)

1. `[org.review].plan_advisory`(既定 false。この repo の `ralph.toml` は true)を true にすると、機能の org の leader が、承認の QUESTION の前(`[org.approval].mode = "split"` では implementer を spawn する前)に、相手のモデルへ読むだけの計画助言を頼む。claude の leader は codex、codex の leader は claude を呼ぶ。最長 20 分待ち、頼むのは 1 回で、承認のやり直しで頼み直さない。書いていない project と、古い ralph(`ralph config get` がない)は off として動く。
2. 承認の QUESTION の本文が変わる。機能の内側の分かれ目(案によってリスク・費用・戻しやすさが大きく違い、コード・rules・既定値で決まらず、途中で変えると 1 スライスより高くつくもの)が `FORK <番号>: <問い> / a <案>(推奨) / b <案>` の行で、すべて載る。計画助言の要約は `ADVISORY:` の行で載る(off のときは行がない)。leader は計画を書く前に AGENTS.md・`.claude/rules/`・計画が触るコードを読む。
3. director は分かれ目を承認と一緒に人に聞く。3 つまでは 1 回の AskUserQuestion(Codex では番号つきの選択肢)、4 つ以上は 4 問ずつ分けて先に聞き、答えがそろってから承認を聞く。人が会話にいなければ、聞かずに `ralph org inbox notify` で人に送る。
4. 推奨と違う案が選ばれると、director は approve を打たず、選んだ案を書いた直しの DECISION を送る。leader は計画をその案で直し、決まった分かれ目を `(決定済み)` の行にした新しい QUESTION を上げる。承認までに 2 往復になるのは、推奨と違う案を選んだときだけ。`approved` の DECISION は固定の文字列で、承認のあとに計画を変えると digest が合わず implementer の TASK が断られるので、この形にした。split では leader は人に聞かずに推奨の案を採り、計画の `## Design decisions` に記録する。
5. 助言者は読むだけで呼ばれ、MCP の server は止められる。codex は `--sandbox read-only --ignore-rules` に加えて `--disable plugins` と server ごとの `-c mcp_servers.<名前>.enabled=false` を付け、同じ引数で読み直した一覧に有効な server が残るときは呼ばずに飛ばして記録する(codex-cli 0.163.0 で、モデルを呼ばずに確かめた)。claude は `--restricted --strict-mcp-config --disallowedTools "mcp__*" --tools Read,Grep,Glob --permission-mode dontAsk` で、`RALPH_WIP_CHECKPOINT=off` を付ける。呼ぶ前と後で git の状態(HEAD、index、作業ツリー、未追跡)を比べ、変わっていれば BLOCKED を上げて止まる。相手の CLI がない、呼べない、時間切れ、出力が空のときは飛ばして理由を記録する。
6. 設定と相手の選び方は、標準フローの `/plan`(#266)の手順 11.a・11.b と同じ。`ralph config get org.review.plan_advisory` の終了コードは、0 で値を使い、`unknown command "config"` の 1(古い ralph)と 127(`ralph` が PATH にない)は off として記録し、ほかの失敗は BLOCKED を上げる。相手の CLI は leader 自身の CLI の反対で、pane の `RALPH_PRIMARY_CLI` が `claude` か `codex`(大文字小文字を問わない)のときだけ、それが leader の CLI に代わる。worktree に `scripts/xreview-helpers.sh` と `scripts/ralph-config.sh` がない project では、計画助言を飛ばして記録する。
7. leader の役割の雛形は binary に go:embed されるので、新しい binary で立てた leader から効く(今の v6.0.0 の binary には入っていない)。director の skill は、新しい skill を読んだ director から効く(`ralph upgrade` が `/director` と `/org` の skill を更新する)。新しい binary の leader と古い skill の director を組み合わせたとき、director が `FORK` の行を分かれ目として聞かないことは、おそらくそうなる。未確認です。

自動の changelog(`.goreleaser.yml` は `^docs:`・`^test:`・`^chore:` を除く)に載る commit は、次の 7 本と merge commit である。

| commit | 題 |
|--------|----|
| `95ad1e11` | `feat: have the feature leader read the rules and list forks in its plan` |
| `837cb430` | `feat: have the director ask plan forks together with the approval` |
| `ce5bdbd1` | `feat: have the feature leader ask the other model for a plan advisory` |
| `6099187d` | `feat: say that the leader skips the advisory when codex has no plugins feature` |
| `d52b6175` | `fix: tighten the leader plan advisory and its tests from the self-review` |
| `cf0e5836` | `fix: say the pane value of RALPH_PRIMARY_CLI wins over the leader CLI` |
| `1aaadb12` | `fix: align the leader plan advisory switch and driver reading with /plan` |
| `d59966a9` | `Merge remote-tracking branch 'origin/main' into feat/leader-plan-depth`(prefix がなく、filter に掛からない) |

`fix:` の 3 本は、この branch が足した未 release の動きを直したもので、release 済みの不具合の修正ではない。changelog には「fix」として出るので、release のノートの 5 と 6 を先に読ませるか、手で整理する。`6099187d` は題が `feat:` でも、5 の「plugins を知らない古い codex では飛ばす」の 1 文で、動きの追加は小さい。`92554f77`(`test:`)と `docs:` の commit は載らない。この run の commit も `docs:` である。

## Notes for reviewers(PR 本文に入れる点)

- issue #267 の文との違い(V5): 「答えを DECISION に入れる」は、D1 で「推奨どおりなら approve、違えば approve せずに直しの DECISION」に変わった。「1 回の AskUserQuestion」は、D3 で「3 つまでは 1 回、4 つ以上は分ける」に変わった。理由は計画の Design decisions にある。
- Known gaps: self-review L11(run の表に、codex の leader で `RALPH_PRIMARY_CLI=claude` の行がなく、driver の `case` の `claude` の枝を外す変異が生き残る。test report が直す 1 行を確かめている)。
- 実行していない確かめ: 実物の leader と director の通し、codex の leader の sandbox から `claude -p` が通るか(通らなければ飛ばして記録する)、AskUserQuestion に 4 問を 1 回で出す動き、`--restricted` の claude を leader の助言者として実際に呼ぶこと(`/plan` の形は #266 が claude 2.1.304 で確かめた)、CI(ubuntu)に zsh があるか(なければ zsh の case は飛ぶ)。
- hang の case は負荷に弱かった(全体の上限 20 秒、load 約 15 で 64 回中 2 回落ちた)ので、92554f77 で上限を 120 秒にした。再実行では 34 回すべて通った(中央値 12.3 秒)。
- この repo の `ralph.toml` は `[org.review].plan_advisory = true` なので、新しい binary でここの機能の org を立てると、leader が承認の QUESTION の前に最長 20 分待つ。
- 800 行を超えるファイル: 新しいテストファイルが 1179 行(tech-debt の `:360` に carry note)。

## Left open

- 800 行を超える 2 つのテストファイルの分割(tech-debt `:360`)。
- feature org の leader の計画助言の形(MCP を止める)を `/plan` と `/cross-review` の codex の呼び出しに移すこと(tech-debt `:154` の (a))。
- V2・V3・V8(情報)と L11。
- `ralph config get --help` に 127 の行がないこと(前の sync-docs が残した項目)。

## Verification

| Check | Result |
|-------|--------|
| `./scripts/check-sync.sh` | rc 0。`PASS: all files in sync.`(IDENTICAL 172、DRIFTED 0、ROOT_ONLY 0、TEMPLATE_ONLY 10、KNOWN_DIFF 6)。root と `templates/base/` の写しは、同じ文で直したので byte 一致のまま |
| `./scripts/check-skill-sync.sh` | rc 0。`[ok] check-skill-sync: 14 skill(s) in lock-step` |
| `./scripts/check-template-purity.sh` | rc 0。`PASS: no meta-repo-specific references found in templates.` |
| `./scripts/sync-skills.sh`(既定と `CLAUDE_ROOT=templates/base/.claude/skills CODEX_ROOT=templates/base/.agents/skills`) | どちらも rc 0。`/org` skill の 4 つの写しは md5 が同じ。`/director` skill の 4 つの写しは `cmp` で一致 |
| `sh -n scripts/ralph-config.sh`、`sh -n scripts/xreview-helpers.sh` | rc 0。コメントだけの変更 |
| `git diff --check` | rc 0、出力なし |
| 変更した 16 ファイルの U+FFFD | なし(`org/SKILL.md` は shell の python で書いたので手で確かめた。ほかは Edit tool) |
| tech-debt の変更した 4 行(31、32、154、360) | どれも 5 列(32 は元から 4 列)で、列ごとの backtick は偶数、`~~` は 0。変わったのはその 4 行だけ(`git show HEAD:` と比較)。plan 参照は存在しないものが 0 件 |
| `./scripts/plan-visual.sh digest docs/plans/active/2026-10-29-leader-plan-depth.md` | `fac3038a3706`(`- Approved:` の行と同じ。AC の checkbox と Progress の編集の前後で同じ) |
| 打っていない | `go test`、`go build`、`golangci-lint`、`run-verify.sh`、`run-static-verify.sh`、`run-test.sh`(依頼で止められた)。この run が変えたのは文書、skill の文、シェルのコメント、計画の checkbox、この report、insight event で、コードの動きは変わらない。コメントを変えた `scripts/ralph-config.sh` を読むテストは行頭の代入だけを読む(上の表) |

## Insight event

`docs/insights/events/2026-10-28-leader-plan-depth.jsonl` の末尾に `sync_docs` の 1 行を足した(phase `sync_docs`、cycle 1、verdict `pass`、findings は 0、ts 2026-10-29T10:05:57Z)。`./scripts/insights-append.sh` は UTC の日付で `2026-10-29-leader-plan-depth.jsonl` を新しく作るので、`--events-dir` で scratchpad に書かせた 1 行を、同じ形のまま既存の file に足した(self-review・verify・test の行と同じ file。test と verify の打ち直しの行も同じやり方)。`--cycle auto` は `.harness/state/standard-pipeline/cycle-count.json` の 1 を読んだ。file は 10 行になり、全行が JSON として読める。
