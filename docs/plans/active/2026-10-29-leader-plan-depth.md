# leader-plan-depth

- Status: Approved
- Approved: 2026-10-29 sha256:fac3038a3706
- Owner: Claude Code
- Date: 2026-10-29
- Related request: org: leader の計画に rules とコードの読み込み、分かれ目の選択肢、相手のモデルの計画助言を足す
- Related issue: #267
- Type: feat
- Branch: feat/leader-plan-depth

## Objective

機能の org の leader が書く計画を、標準フローの `/plan` に近づける。leader は計画を書く前に AGENTS.md・`.claude/rules/`・計画が触るコードを読み、機能の内側の分かれ目を推奨つきの選択肢として承認の QUESTION に載せ、`[org.review].plan_advisory` が on なら相手のモデルに計画助言を頼む。director は、その分かれ目を人に承認と一緒に聞く(3 つまでなら 1 回の問いで)。

## Scope

- S1 leader の雛形(`internal/org/prompts/leader.md`)の計画の手順
  - 「機能ごとの org」の手順 1 の前に、AGENTS.md(あれば CLAUDE.md)・`.claude/rules/`(Codex なら `.codex/AGENTS.override.md` も)・計画が触るコードを読む手順を足す
  - 手順 1 に、機能の内側の分かれ目を探す手順を足す。分かれ目は `/plan` の手順 8 の 3 条件(案によってリスク・費用・戻しやすさが大きく違う、コードや rules や妥当な既定値で決まらない、途中で変えると 1 スライスより高くつく)をすべて満たすもの。機能をまたぐ分かれ目は扱わない(分割計画で director と人が決める。#271)
  - 分かれ目ごとに、2〜4 の選択肢と推奨と理由を計画の `## Design decisions` に書く。計画は推奨の案で書く
  - per-plan: 手順 3 の QUESTION の本文に、分かれ目を 1 つ 1 行で `FORK <番号>: <問い> / a <案>(推奨) / b <案>` の形で、すべて載せる(数で打ち切らない。D3)。本文の 2,000 字に収まらなければ、各行を短くし、理由は `## Design decisions` を指す。QUESTION の例を、分かれ目のある形に変える
  - per-plan で 2 往復目の QUESTION を上げるとき(推奨と違う答えで計画を直したとき)は、決まった分かれ目を `FORK <番号>: <問い> → <決まった案>(決定済み)` の形で載せ、選択肢を並べない
  - split: 人に聞かずに推奨の案を採り、`## Design decisions` にそう記録する
  - DECISION の扱い(手順 4): 今の「本文が `approved` で始まるものは承認、それ以外は直しの依頼」のまま。推奨と違う答えは直しの依頼の DECISION で届くので、計画を直してから手順 2 と 3 をやり直す(D1)
- S3 director の skill と `/org` skill
  - `.claude/skills/director/SKILL.md` の「計画の承認」に、QUESTION の本文に `FORK` の行があれば、分かれ目の問い(1 行 1 問、推奨の案を最初に)と承認の問い 1 つを AskUserQuestion(Codex では番号つきの選択肢)で聞く手順を足す。分かれ目が 3 つまでなら承認と合わせて 1 回(4 問まで)で聞く。4 つ以上なら 4 問ずつに分けて先に分かれ目をすべて聞き、全部の答えがそろってから承認を聞く(答えがそろうまで approve を打たない。D3)。人が会話にいなければ、「件の処理」の 3 のとおり `ralph org inbox notify` で人に送る。`(決定済み)` の行は聞かない
  - すべての分かれ目に推奨の案が選ばれ、承認されたら、今までどおり `ralph org approve` に進む。推奨と違う案が 1 つでもあれば approve を打たず、手順 5 のとおり、選んだ案を本文に書いた直しの DECISION(`approved` で始めない)を送る(D1)
  - `.claude/skills/org/SKILL.md` の「機能の計画の承認」に、leader が分かれ目を QUESTION に載せることと計画助言のことを足す
  - 雛形の写し(`templates/base/.claude/skills/`)を同じに直し、`./scripts/sync-skills.sh` で `.agents/skills/` の写しを作り直す
- S4 文書の「まだ読まない」の文を直す
  - `templates/base/ralph.toml` の `[org.review]` のコメント(「They start reading them in a later release.」)、`internal/config/config.go` の `OrgReviewConfig` の doc comment(「they start to in issues #267 and #269」)、`/org` skill の「座席が読み始めるのは後の版から」: leader が `plan_advisory` を読むようになったこと(reviewer の `cross_review` は #269)を書く
- S2 leader の雛形の計画助言(最後のスライス。入る前に #266 の claude の打ち方が確定したかを B に確かめる)
  - 手順 1 のあと、per-plan では手順 2 の前に、split では implementer を spawn する前に、worktree で `ralph config get org.review.plan_advisory` を打つ。終了コードは `ralph config get` の help の約束どおりに扱う(0 は値を使う。1 で stderr に `unknown command "config"` なら off とみなしてそれを記録する。ほかの 1 と 2・3 は BLOCKED を上げて止まる。D5)
  - `false` なら計画助言を呼ばない
  - `true` なら、`. scripts/xreview-helpers.sh; pick_reviewer "$RALPH_PRIMARY_CLI"` で相手の CLI を選ぶ
    - codex: `/plan` の手順 11.c と同じ打ち方(`command codex -m … -c … exec --sandbox read-only --ignore-rules -o <ファイル>`、`</dev/null`、20 分の watchdog)に、MCP の server を止める `-c` を足す(D7)。`command codex mcp list --json` で `enabled` の server の名前を読み、名前ごとに `-c mcp_servers.<名前>.enabled=false` を付ける(1 行ずつ読んで引数の並びに足し、shell の語の分け方に頼らない)。付けたあと同じ `-c` で `codex mcp list --json` を打ち、`enabled` の server が 0 でなければ(jq がない、一覧が読めないときも)計画助言を飛ばして記録する
    - claude: 下の形の `--strict-mcp-config`(`--mcp-config` を渡さないので MCP の server を立てない)と `--disallowedTools "mcp__*"` で MCP を止める
    - claude: #266 と同じ読むだけの打ち方(D2)。予定の形は `RALPH_WIP_CHECKPOINT=off claude -p --model "${RALPH_CLAUDE_REVIEWER_MODEL:-opus}" --restricted --strict-mcp-config --disallowedTools "mcp__*" --tools Read,Grep,Glob --permission-mode dontAsk </dev/null` に、`/plan` の手順 11.c と同じ計画レビューの指示文を渡すもの。20 分の watchdog
    - どちらも、呼ぶ前と後で `git rev-parse HEAD`、`git status --porcelain`(未追跡を含む)、`git diff --cached` を比べ、変わっていたら BLOCKED を上げて止まる
  - 終わったとみなすのは、終了コード 0 で出力が空でないときだけ。相手の CLI がない、呼べない、時間切れ、出力が空のときは飛ばし、飛ばしたことと理由を記録して進む
  - 所見は leader が判断して計画に反映する。HIGH と MEDIUM は反映するか、反映しない理由を書く(D4)。所見と扱いの要約は、per-plan では QUESTION の本文に 1〜2 行で、split では `## Design decisions` に書く
  - `scripts/xreview-helpers.sh` か `scripts/ralph-config.sh` がない project では、計画助言を飛ばして記録する

## Non-goals

- 機能をまたぐ分かれ目を上流の分割計画で詰める手順(#271)
- reviewer 座席の cross-review(#269)
- 標準フローの `/plan` の計画助言を相手のモデルにすること(#266。セッション B が担当)
- ralph のコード(Go)で設定を読んで leader のタスクに入れること。leader は `ralph config get` で自分で読む
- claude と codex の flag を実機で確かめること(#266 の S2 が確かめる)。codex の座席の sandbox から `claude -p` が通るかも確かめない(通らなければ飛ばして記録する)
- headless の director が人に聞く方法を変えること(今の「件の処理」の手順のまま)

## Assumptions

- #265(PR #284)で、座席(leader を含む)の pane に `RALPH_PRIMARY_CLI=<driver>` が入り、`ralph config get <key>` で `[org.review].plan_advisory` を読める(`internal/cli/config.go`、`internal/org/driver/paneenv.go`)
- leader の cwd は機能の worktree で、そこにリポジトリの `ralph.toml`(なければ組み込みの既定値)、`scripts/xreview-helpers.sh`、`scripts/ralph-config.sh` がある(`templates/base/scripts/` にもあるので、scaffold した project にもある)
- `ralph org approve` が出す承認の DECISION の本文は固定の `approved` で(`internal/org/approve.go` の `ApprovalDecisionText`)、approve は digest を台帳に記録する。承認のあとに計画を変えると digest が合わず、implementer の spawn と TASK が断られる
- 座席の typed message と `ralph org escalate` の本文は 2,000 字まで(`.claude/rules/ralph/agent-messaging.md`)
- AskUserQuestion は 1 回に 4 問まで
- leader の雛形のテストは `internal/org/prompts_test.go` などにあり、`renderRolePrompt` の出力の文を確かめる

## Affected areas

- `internal/org/prompts/leader.md`
- `internal/org/prompts_leader_plan_test.go`(新規。名前は実装で決める)
- `.claude/skills/director/SKILL.md`、`templates/base/.claude/skills/director/SKILL.md`、`.agents/skills/director/SKILL.md`、`templates/base/.agents/skills/director/SKILL.md`
- `.claude/skills/org/SKILL.md`、`templates/base/.claude/skills/org/SKILL.md`、`.agents/skills/org/SKILL.md`、`templates/base/.agents/skills/org/SKILL.md`
- `templates/base/ralph.toml`、`internal/config/config.go`(doc comment だけ)

## Visual review

- Page: `.harness/state/plan-visual/leader-plan-depth.html`(易しい図 2 つ: `overview`、`easy-forks`。fold の中に `detail-overview`、`detail-approval`、`detail-advisory`)
- Lint: `./scripts/plan-visual.sh lint` が exit 0
- Self-check: ページ全体と 5 つの図を `shot` で撮って読み、崩れがないことを確かめた

## Design decisions

- D1(分かれ目に推奨と違う答え): 推奨どおりで承認されたときだけ `ralph org approve` を打つ。1 つでも違えば approve せず、選んだ案を書いた直しの DECISION を送り、leader が計画を直して承認を取り直す。承認の DECISION の本文は固定の `approved` で、承認のあとに計画を変えると digest が合わず implementer が断られるため。leader の手順 4 の「それ以外の DECISION は直しの依頼」と director の手順 5 をそのまま使える。2 往復になるのは推奨と違う案を選んだときだけ
- D2(codex の leader が claude を呼ぶ打ち方): #266(標準フロー)と同じ読むだけの形にそろえる。`--permission-mode auto` は書き込みを止めず、計画の digest の前に助言者が worktree に書けるため。#266 の S2 で flag が確定する前に S2 を書くときは #266 の計画の形で書き、PR の Known gaps に「#266 の確定で flag をそろえる」と書く。テストは flag の完全一致ではなく、`claude -p`、model の変数、読むだけの道具、完了の条件、呼ぶ前後の git の比べを確かめる
- D3(分かれ目の数): 数で打ち切らず、すべて QUESTION に載せて人に聞く。3 つまでなら承認の問いと合わせて 1 回の AskUserQuestion(4 問まで)で聞ける。4 つ以上なら director が 4 問ずつに分けて先に分かれ目をすべて聞き、答えがそろってから承認を聞く。初めの案(4 つ目から推奨を採る)は、分かれ目の定義(途中で変えると 1 スライスより高くつく)と食い違い、いちばん重い判断が 4 つ目に並んでも人に聞かずに決めてしまうため、Codex の計画助言(MEDIUM)を受けて変えた
- D7(助言者の MCP): Codex の計画助言(HIGH)を受けて、助言者の MCP の server を止める。`read-only` の sandbox と `--ignore-rules` は MCP に及ばず、書ける MCP の server があると、承認の前に外部のサービスを変えられ、git の比べでは気づけないため。codex は `-c mcp_servers.<名前>.enabled=false` を server ごとに付け(`-c mcp_servers={}` では止まらないことを `codex-cli 0.163.0` で確かめた)、止めたあとの一覧で `enabled` が 0 のときだけ呼ぶ。claude は `--strict-mcp-config` と `--disallowedTools "mcp__*"`(#266 と同じ)。止められなければ呼ばずに飛ばして記録する
- D4(所見の扱い): HIGH と MEDIUM は計画に反映するか、反映しない理由を書く。LOW は leader の判断。per-plan では QUESTION に要約を載せて人に見せる
- D5(設定を読めないとき): `ralph config get` の help の約束どおり、2・3 と、`unknown command "config"` でない 1 は BLOCKED を上げて止まる。古い ralph の `unknown command "config"` だけ off とみなして記録する
- D6(split でも計画助言を走らせる): 承認方式を問わず、on なら呼ぶ。split は人が計画を見ないので、助言の価値はむしろ高い。per-plan は QUESTION の前、split は implementer の spawn の前
- Critical forks: D1〜D6 は consult(consult-plan-leaderplan)に聞いて推奨案を選んだ(ユーザーの指示)。D3 の見直しと D7 は、Codex の計画助言の HIGH と MEDIUM を聞かずに反映したもの(ユーザーの指示)

## Acceptance criteria

- [ ] AC1: leader の雛形の「機能ごとの org」に、計画を書く前に AGENTS.md・`.claude/rules/`・計画が触るコードを読む手順がある
- [ ] AC2: leader の雛形が、機能の内側の分かれ目(3 条件)を推奨つきの選択肢で `## Design decisions` に書き、per-plan では承認の QUESTION の本文に `FORK` の行ですべて載せ、split では推奨の案を採って記録することを書く。承認の QUESTION の例に、推奨つきの分かれ目の行がある。2 往復目の QUESTION は決まった分かれ目を `(決定済み)` で載せる
- [ ] AC3: director の skill の「計画の承認」に、`FORK` の行を AskUserQuestion(Codex では番号つきの選択肢)で聞く手順(3 つまでなら承認と合わせて 1 回、4 つ以上なら 4 問ずつ分けて全部の答えがそろうまで approve を打たない)、人が会話にいないときは notify する手順、推奨と違う答えなら approve を打たずに直しの DECISION を送る手順がある
- [ ] AC4: leader の雛形が、計画助言の前に `ralph config get org.review.plan_advisory` を打ち、`false` なら計画助言を呼ばないこと、終了コードの扱いを書く
- [ ] AC5: leader の雛形が、`true` なら `pick_reviewer "$RALPH_PRIMARY_CLI"` で相手の CLI を選び、claude の leader は codex を、codex の leader は claude を読むだけの形で呼ぶこと、どちらも助言者の MCP の server を止め、止められなければ呼ばないこと、呼ぶ前後の git の状態を比べること、呼べなければ飛ばして記録することを書く
- [ ] AC8: codex の MCP の server を止める打ち方(`-c mcp_servers.<名前>.enabled=false` を server ごとに付け、止めたあとの `codex mcp list --json` で `enabled` が 0)を、この machine の codex で実際に打って確かめ、証跡を残す(モデルは呼ばない)
- [ ] AC6: 雛形のテスト(新しいテストを含む)、`./scripts/check-skill-sync.sh`、`./scripts/check-sync.sh` が通る
- [ ] AC7: `[org.review]` のコメントと doc comment と `/org` skill が、leader が `plan_advisory` を読むことを書く

## Implementation outline

1. S1: leader の雛形の読み込み・分かれ目・split の記録・QUESTION の例・2 往復目の QUESTION(AC1・AC2) — verify: `test -z "$(gofmt -l internal/org)" && go vet ./internal/org/ && GOFLAGS="-timeout=30m $(go env GOFLAGS)" go test ./internal/org/ -run 'RenderRolePrompt_Leader'`
2. S3: director の skill と `/org` skill(AC3、AC6 の同期) — verify: `./scripts/sync-skills.sh && ./scripts/check-skill-sync.sh && ./scripts/check-sync.sh && GOFLAGS="-timeout=30m $(go env GOFLAGS)" go test ./internal/org/ -run 'Skill'`
3. S4: 文書の「まだ読まない」の文(AC7) — verify: `test -z "$(gofmt -l internal/config)" && go vet ./internal/config/ && GOFLAGS="-timeout=30m $(go env GOFLAGS)" go test ./internal/config/ && ./scripts/check-sync.sh && ./scripts/check-skill-sync.sh`
4. S2: leader の雛形の計画助言(AC4・AC5・AC8)。入る前に #266 の確定を B に確かめる。AC8 の確かめ(codex の MCP の止め方を打つ)もこのスライスで行う — verify: `test -z "$(gofmt -l internal/org)" && go vet ./internal/org/ && GOFLAGS="-timeout=30m $(go env GOFLAGS)" go test ./internal/org/ -run 'RenderRolePrompt_Leader'`

## Verify plan

- Static analysis checks: `./scripts/run-static-verify.sh`(gofmt、go vet、golangci-lint、check-sync、check-skill-sync を含む)
- Spec compliance criteria to confirm: AC1〜AC7 と issue #267 の受け入れ条件 5 つ
- Documentation drift to check: `git grep -n "plan_advisory"`、`git grep -n "#267"`、`git grep -n "later release"` で、leader がまだ読まないと書く現在形の文が残っていないこと。`docs/specs/2026-10-07-org-multi-org-director.md` の dated の段落は書き換えず、必要なら 2026-10-29 の行を足す
- Evidence to capture: `run-static-verify.sh` の出力、`renderRolePrompt` で展開した leader の雛形の該当の節

## Test plan

- Unit tests(leader の雛形の文):
  - 手順 1 の前に、AGENTS.md・`.claude/rules/`・計画が触るコードを読む手順がある
  - 分かれ目の 3 条件、機能の内側に限ること、2〜4 の選択肢と推奨を `## Design decisions` に書くことがある
  - per-plan の QUESTION の例に `FORK 1:` の行と `(推奨)` がある。数で打ち切らない。2 往復目は `(決定済み)`
  - split では推奨を採って記録する
  - `ralph config get org.review.plan_advisory` と終了コードの扱い(0、`unknown command "config"` の 1、それ以外)、`false` なら呼ばない
  - `pick_reviewer "$RALPH_PRIMARY_CLI"`、codex の打ち方(`--sandbox read-only --ignore-rules -o`、20 分、`mcp_servers.<名前>.enabled=false` と止めたあとの確かめ)、claude の打ち方(`claude -p`、model の変数、読むだけの道具、`--strict-mcp-config`、20 分)、呼ぶ前後の git の比べ、終わったとみなす条件、飛ばして記録すること
- Manual check(AC8): この machine の codex で、MCP の server を止める snippet を打ち、`codex mcp list --json` の `enabled` が 0 になることを確かめる。結果を Progress checklist と verify の報告に書く(server の名前と数だけ。設定の中身は写さない)
  - 機能ごとでない org の節にはこれらが現れない(今の雛形の書き分けを崩さない)
- Unit tests(skill の文): director の skill の「計画の承認」に、`FORK` の行、AskUserQuestion、4 つ以上は分けて全部の答えがそろうまで approve しないこと、notify、推奨と違う答えで approve を打たないことがある。既存の `director_skill_sync_test.go` の形にならう
- Integration tests: なし(雛形の文と skill の文の変更。leader と director の実機の通しは Non-goals)
- Regression tests: 既存の `TestRenderRolePrompt_Leader_*`(特に `FeatureOrgPlanApproval` と `FeatureOrgProcedure`)、`director_skill_sync_test.go`、`check-skill-sync.sh`
- Edge cases: 分かれ目がない計画(QUESTION に `FORK` の行を書かない)。分かれ目が 4 つ以上の計画(director が分けて聞く)。MCP の server を止められない環境(計画助言を飛ばす)。`RALPH_PRIMARY_CLI` がない pane(`pick_reviewer` は claude とみなして codex を選ぶ)
- Evidence to capture: `go test` の出力、`run-test.sh` の結果

## Risks and mitigations

- 推奨と違う答えを承認の DECISION に入れると、承認のあとに計画が変わり、implementer の TASK が断られる: 直しの DECISION として扱い、承認を取り直す(D1)
- 計画助言の claude の打ち方が #266 とずれる: #266 の形にそろえ(D2)、S2 を最後に回して B の確定を確かめてから書く。未確定なら Known gaps に書く
- 助言者が worktree に書く: 読むだけの打ち方にし、呼ぶ前後の git の状態を比べて、変わっていたら BLOCKED で止まる
- 計画助言が 20 分かかり、承認の QUESTION が遅れる: 設定の既定は off。on のときだけ待つ
- leader が所見を都合よく捨てる: HIGH と MEDIUM は反映するか理由を書き、QUESTION の本文で人に見せる
- QUESTION の本文が 2,000 字を超える: 分かれ目は 1 つ 1 行の短い形にし、計画助言は 1〜2 行の要約にし、詳しくは計画の `## Design decisions` を指す
- 助言者が MCP の server を通して外部のサービスを変える: 助言者の MCP の server を止め、止めたことを確かめられないときは呼ばない(D7)
- codex の MCP の止め方が codex の版で変わる: 止めたあとの一覧の確かめで気づき、計画助言を飛ばす(止まらないまま呼ぶことはない)
- director が 2 往復目でも同じ分かれ目を聞く: 2 往復目は `(決定済み)` の行にし、director はその行を聞かない

## Rollout or rollback notes

- leader の雛形は go:embed で binary に入り、spawn のたびに展開される。新しい binary で立てた leader から効く。director の skill は、新しい skill を読んだ director から効く
- 戻すときはこの PR を revert する。台帳と設定の形は変えない
- release のノート: `[org.review].plan_advisory = true` で leader が相手のモデルに計画助言を頼むようになる(既定は off)。承認の QUESTION に分かれ目の選択肢が載り、director が承認と一緒に聞く

## Open questions

- なし(分かれ目は Design decisions に書いた)

## Progress checklist

- [x] Plan reviewed(consult-plan-leaderplan: 判定「直してから進める」。F1〜F5 は案のとおり。D2 を #266 の形に書き換え、S2 を最後に回し、呼ぶ前後の git の比べ、2 往復目の `(決定済み)`、headless の director の notify を足した。#266 の担当(セッション B、ralph-6d)に claude の打ち方を問い合わせ、`--restricted --strict-mcp-config --disallowedTools "mcp__*" --tools Read,Grep,Glob --permission-mode dontAsk` と `RALPH_WIP_CHECKPOINT=off` の形を聞いた。B の S2 で確定したら知らせてもらう。Codex の計画助言: HIGH 1 件(助言者の MCP が止まらない)と MEDIUM 1 件(4 つ目からの分かれ目を自動で決める)を聞かずに反映した(D7 と D3 の見直し、AC8)。codex の MCP は `-c mcp_servers={}` では止まらず、`-c mcp_servers.<名前>.enabled=false` なら止まることを `codex-cli 0.163.0` の `codex mcp list --json` で確かめた(server 1 つで試した。全部を止める確かめは S2 の AC8))
- [x] Plan approved(自動承認(ユーザーの指示)。2026-10-29、digest sha256:fac3038a3706)
- [x] Branch created
- [x] Implementation started
  - #266 の claude の計画助言の形が確定した(B から、2026-10-29、B のコミット 93dec2cc)。`RALPH_WIP_CHECKPOINT=off command claude -p --model "${RALPH_CLAUDE_REVIEWER_MODEL:-opus}" --restricted --strict-mcp-config --disallowedTools "mcp__*" --tools Read,Grep,Glob --permission-mode dontAsk --output-format text "<prompt>" </dev/null > <出力> 2> <log>`、20 分の watchdog、rc 0 かつ出力が空でないときだけ完了。git の確かめは `{ git rev-parse HEAD; git status --porcelain=v1 --untracked-files=all; git diff --cached; git diff; git ls-files -z --others --exclude-standard | xargs -0 git hash-object --; }` を前後で取って cmp する。S2 はこの形で書く
  - S1(95ad1e11、implementer): 手順 1 の書き出しに読み込み(AGENTS.md と CLAUDE.md、`.claude/rules/`、Codex は `.codex/AGENTS.override.md`、計画が触るコード)、手順 1 の最後に分かれ目の段落(3 条件、機能の内側、`## Design decisions` に 2〜4 の案と推奨と理由、分かれ目がなければ `Critical forks: None`、split は推奨を採って記録)、手順 3 に `FORK` の行(推奨はいつも a、理由は書かない、数で打ち切らない、2,000 字に収まらなければ言い直す)と例の 2 行、2 往復目の `(決定済み)`、手順 4 に「`approved` はすべて推奨で決まった意味も持ち、`## Design decisions` は書き換えない。推奨と違う答えは直しの依頼」。手順の番号は振り直していない。新しいテスト 6 本。verify rc=0(9s)。`go test -overlay` で leader.md を HEAD の版にすると新しい 6 本がすべて落ちる。self-check 5 項目 clean
  - S3(837cb430、implementer): director の「計画の承認」に手順 2(FORK)を足し、元の 2〜6 を 3〜7 に振り直した。`(決定済み)` は聞かず数えない、1 行 1 問で a を最初、3 つまでは承認と 1 回、4 つ以上は 4 問ずつ先に聞いて答えがそろってから承認、推奨どおりなら approve、違えば手順 6 の直しの DECISION。直しの DECISION の本文には、推奨を選んだものも含めてすべての分かれ目の答えを `FORK <番号>: <記号> <案>` で書く(2 往復目で全部が `(決定済み)` になり、承認だけを聞けばよい)。人が会話にいないときの notify は手順 1 に置いた(分かれ目のない承認にも効く)。`/org` skill の「機能の計画の承認」に 1 段落。4 つの写しは cmp で一致。`director_skill_sync_test.go` の pin を手順 4 に移し、新しいテスト `TestDirectorSkill_PinsTheApprovalForkSteps`。verify rc=0(15s)、check-sync DRIFTED 0。HEAD の版の skill で 20 件、FORK の形を変えた leader.md で 2 件落ちる。self-check: 誤字とコピペ fixed(例の FORK の番号)、ほか clean
    - consult(consult-plan-leaderplan)の低いリスク: 手順 1 の「問いを出して答えを待たずに」は、AskUserQuestion を出してよいとも読め、人のいない pane ではその道具で止まる。道具を使わないと分かる言い方に S4 で直す
- [ ] Review artifact created
- [ ] Verification artifact created
- [ ] Test artifact created
- [ ] PR created
