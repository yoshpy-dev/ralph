# Verify report: leader-plan-depth

- Date: 2026-10-29
- Plan: `docs/plans/active/2026-10-29-leader-plan-depth.md`(issue #267、AC1〜AC8、D1〜D7。Progress に記録された逸れは 2 つ: 計画助言を手順 1 の最後ではなく手順 16 の後ろの `### 計画助言` に移した(self-review L6)、codex の MCP の止め方に `--disable plugins` を足した(D7))
- Verifier: verifier サブエージェント(`/verify`、パイプラインの cycle 1)
- Scope: `git diff origin/main...HEAD`(base 9837f744、HEAD 060d6c1c、21 コミット、18 ファイル、+1929/-55)。self-review の Run 3(Merge yes、LOW の L8 は /sync-docs に渡す)のあとの全体。
- Evidence: `docs/evidence/verify-2026-10-29-leader-plan-depth.log`(gitignore の対象でコミットしない。`run-static-verify.sh` の全出力、digest、skill の写しの `cmp`、指示文と claude の打ち方の照合、AC8 の数、`go run -overlay` で展開した leader の雛形の `### 計画助言`、文書の grep)。wrapper 自身の保存先は `docs/evidence/verify-2026-10-28-234928.log`(UTC の日付)。
- Insight event: `docs/insights/events/2026-10-28-leader-plan-depth.jsonl` に 1 行(phase verify、cycle 1、verdict pass、low 1)。UTC の日付が変わる 3 分前だったので、self-review の 3 行と同じ file に入るよう、この report を書く前に追記した(skill の順では report のあと)。判定と所見の数は追記の時点で決まっていた。

## Spec compliance

| Acceptance criterion | Status | Evidence |
| --- | --- | --- |
| AC1: 計画を書く前に AGENTS.md・`.claude/rules/`・計画が触るコードを読む | 満たす | `internal/org/prompts/leader.md:144-147`(手順 1 の書き出し。CLAUDE.md、Codex では `.codex/AGENTS.override.md` も)。`.codex/AGENTS.override.md` はルートと `templates/base/` の両方にある。テストは `internal/org/prompts_leader_plan_test.go:144`。 |
| AC2: 分かれ目を `## Design decisions` に書き、per-plan では `FORK` の行ですべて載せ、split では推奨を採って記録する。例に推奨つきの行、2 往復目は `(決定済み)` | 満たす | 3 条件、機能の内側に限ること、2〜4 の案と推奨と理由、`Critical forks: None`、per-plan と split の書き分けは `leader.md:166-178`。手順 3 の `FORK` の形(推奨はいつも a、数で打ち切らない、理由は書かない、2,000 字に収まらなければ言い直して行は削らない)は `:204-213`、例の 2 行は `:221-222`、2 往復目は `:227-231`、手順 4 の直しの依頼は `:247-255`。テストは `:164`、`:194`、`:250`、`:295`。 |
| AC3: director の skill に、`FORK` を聞く手順、notify、推奨と違う答えで approve を打たない手順 | 満たす | `.claude/skills/director/SKILL.md:764-772`(手順 1: `ADVISORY:` の要約も伝える、人が会話にいなければ AskUserQuestion を使わず `ralph org inbox notify`)、`:773-792`(手順 2: `(決定済み)` は聞かず数えない、1 行 1 問で a が最初、3 つまでは承認と 1 回、4 つ以上は 4 問ずつ先に聞いて承認は最後、答えがそろうまで approve を打たない、推奨と違えば手順 6 の直しの DECISION)、`:793`(手順 3)、`:807`(手順 6)。4 つの写しは `cmp` で一致。テストは `internal/org/director_skill_sync_test.go` の `TestDirectorSkill_PinsTheApprovalForkSteps`。 |
| AC4: `ralph config get org.review.plan_advisory`、`false` なら呼ばない、終了コードの扱い | 満たす | `leader.md:491-506`。0 で `true` なら頼み `false` なら記録もせずに進む、1 で `unknown command "config"` なら off とみなして記録、ほかの 1・2・3・0 で真偽値でない値は BLOCKED。`internal/cli/config.go` の help と向きが同じ(テスト `:459` が help の文字列リテラルを読んで照合する)。 |
| AC5: `pick_reviewer` で相手を選び、読むだけで呼び、MCP を止め、止められなければ呼ばず、git の前後を比べ、呼べなければ飛ばして記録 | 満たす | 選び方は `leader.md:508-516`。AC の `pick_reviewer "$RALPH_PRIMARY_CLI"` は self-review L5 の直しで `"${RALPH_PRIMARY_CLI:-<あなたの CLI>}"` に広がった(Progress に記録あり、V3)。codex は `:535-558`(`--disable plugins`、server ごとの `-c mcp_servers.<名前>.enabled=false`、同じ引数の一覧で有効が 0 のときだけ `exec --sandbox read-only --ignore-rules -o`)。claude は `:560-572`。終わりの読み分けと所見の扱い(HIGH と MEDIUM は反映か理由、要約は per-plan で `ADVISORY:`、split で `## Design decisions`)は `:574-596`。テストは `:510`、`:559`、`:592`、`:628`、`:666`、`:694`、stand-in を bash と zsh で打つ `:954` と `:1073`。 |
| AC8: codex の MCP の止め方をこの machine で打ち、`enabled` が 0 になる証跡(モデルは呼ばない) | 満たす(打ち直した) | 雛形の `leader.md:552-554` を byte 単位で写した script を bash と zsh で打った。数は下の「Observational checks」。 |
| AC6: 雛形のテスト、`check-skill-sync.sh`、`check-sync.sh` が通る | 一部(静的な部分のみ) | `check-skill-sync.sh` は rc 0(14 skill)、`check-sync.sh` は rc 0(DRIFTED 0)。`go vet ./...` はテストのファイルも含めて compile を通した。`go test` は /test の担当で、ここでは打っていない。 |
| AC7: `[org.review]` のコメント、doc comment、`/org` skill が、leader が `plan_advisory` を読むことを書く | 満たす | `templates/base/ralph.toml:255-258`、`internal/config/config.go:559-561`、`.claude/skills/org/SKILL.md:51-53`(4 つの写しは `cmp` で一致)。reviewer の `cross_review` は「後の版から」のまま(#269)。 |

issue #267 の受け入れ条件との対応:

| issue の受け入れ条件 | Status | Evidence |
| --- | --- | --- |
| leader の雛形に、rules とコードを読む指示がある | 満たす | AC1 と同じ。 |
| QUESTION の例に推奨つきの分かれ目の選択肢、director の skill にそれを 1 回の AskUserQuestion で聞く手順 | 満たす(D3 の外れあり) | 例は `leader.md:221-222`。分かれ目が 3 つまでなら 1 回で聞く。4 つ以上は AskUserQuestion の 4 問の上限があるので分けて聞く(D3)。 |
| 計画助言が off のとき、leader は呼ばない | 満たす(雛形の文) | `leader.md:500`(`false` なら頼まずに次へ進む)。実際の leader の動きは見ていない。 |
| on のとき、claude の leader は codex を、codex の leader は claude を呼ぶ(呼べなければ飛ばして記録) | 満たす(雛形の文と stand-in のテスト) | AC5 と同じ。選び方のテスト `:510` は 5 通り(pane が claude、codex、pane と leader が食い違う、pane に値なしで claude と codex)を bash と zsh で打つ。 |
| 雛形のテストと skill の同期の検査が通る | 一部 | 同期の検査は通った。テストの実行は /test。 |

記録された外れ: issue の「答えを DECISION に入れる」は、D1 で「推奨どおりなら approve、違えば approve せずに直しの DECISION」に変わった。承認の DECISION の本文が固定の `approved` で、承認のあとに計画を変えると digest が合わなくなるため。issue の「1 回の AskUserQuestion」は、D3 で「3 つまでは 1 回、4 つ以上は分ける」に変わった。どちらも計画の Design decisions に理由があり、PR の本文でも触れるとよい(V5)。

## Static analysis

| Command | Result | Notes |
| --- | --- | --- |
| `./scripts/run-static-verify.sh` | pass(rc 0、44.9 秒) | scope は `full fallback (unclassified:templates/base/ralph.toml)` で golang の検査が走った。gofmt `ok`、go vet は出力なし、golangci-lint 2.15.0 は `0 issues.`、staticcheck は出力なし(`command -v staticcheck` で入っていることを確かめた)。shellcheck、`sh -n`、settings.json の jq、Codex の hook の検査、check-sync(IDENTICAL 172、DRIFTED 0、ROOT_ONLY 0、TEMPLATE_ONLY 10、KNOWN_DIFF 6)、check-pipeline-sync、check-skill-sync、check-template-purity、tech-debt の参照も通った。branch の secret scan は `9837f744..060d6c1c` で clean。1 回で通り、打ち直していない。 |
| `./scripts/check-skill-sync.sh`(単独) | pass(rc 0) | `14 skill(s) in lock-step`。 |
| `cmp`(director と org の skill、それぞれ 4 つの写し) | 一致 | `.claude`、`templates/base/.claude`、`.agents`、`templates/base/.agents`。 |
| `git diff --check origin/main...HEAD` | pass(rc 0) | 空白の誤りなし。 |
| `./scripts/plan-visual.sh digest <計画>` | 一致 | `fac3038a3706` で、`- Approved:` の行と同じ。承認のコミット fd7bb290 から HEAD までの計画の変更は `## Progress checklist` の中だけ。 |

## Documentation drift

| Doc / contract | In sync? | Notes |
| --- | --- | --- |
| leader の `FORK` の行と director の手順 | yes | 1 往復目の形(`leader.md:204-206`)と `(決定済み)` の形(`:227-231`)は、director の手順 2(`SKILL.md:773-778`)の 2 つの形と同じ。テストが両方を固定する。director の直しの DECISION は、推奨を選んだものも含めてすべての分かれ目を `FORK <番号>: <記号> <案>` で書くので、leader の「その依頼で決まった分かれ目を `(決定済み)` にする」で全部が `(決定済み)` になり、2 往復目は承認だけを聞けばよい。番号の参照(手順 1→2、2→3 と 6、3→2、「件の処理」の 3 の notify は `:652-653`)は合っている。`/org` skill の `:917-929` の「上の 5」は同じ節の人の手順 5(直してほしいとき)を指す。 |
| 計画助言の指示文と `/plan` の手順 11.c | yes | 計画のパスを除いた 990 byte が一致(`cmp`)。テスト `:559` も同じ照合をする。 |
| claude の打ち方と #266 | yes | `RALPH_WIP_CHECKPOINT=off command claude -p` から `--output-format text` までが、`origin/feat/review-switches-skills`(67410c6f)の `.claude/skills/plan/SKILL.md:96` と一致(`cmp`)。git の状態を取るコマンドも #266 の形と同じ文字列。 |
| `[org.review]` のコメント、doc comment、`/org` skill(AC7) | yes | 上の表の AC7。 |
| 「later release」の残り | yes | `templates/base/ralph.toml:29` は `[standard]` の話(#266 の範囲)、`:258` は reviewer の話(#269)。 |
| pane の `RALPH_PRIMARY_CLI` の用途 | no(既知、/sync-docs に渡す) | `README.md:259`、`.claude/skills/org/SKILL.md:31-34`(4 つの写し)、`docs/recipes/isolated-herdr-server.md:241-249`(テンプレートの写しも)は、用途を座席の cross-review としか書かない。`internal/org/driver/paneenv.go:39-43` と警告文(`:69-71`)は leader の計画助言も書いている。`/org` skill の `:40-41`(`--env` のない herdr のときの影響)も cross-review だけを書くが、leader は pane に値がなければ自分の CLI を使うので、影響するのは herdr の server の環境に値があるときだけ。こちらは足すかどうかを /sync-docs が決めてよい。 |
| `/org` skill の「機能の計画の承認」と `ADVISORY:` | no(既知、/sync-docs に渡す) | `.claude/skills/org/SKILL.md:917-929` は `FORK` を書くが `ADVISORY:` の行に触れない。director の手順 1 は触れている。 |
| `paneenv.go:42`(self-review L8) | 小さな食い違い(既知) | 「only when the variable is unset」は、`:-` が空のときも leader の CLI に落ちることを言い当てていない。害はない。 |
| spec の dated の段落 | no(新規、V1) | `docs/specs/2026-10-07-org-multi-org-director.md:129` の 2026-10-29 の段落(#265)は「この版では設定を読む座席はまだない(leader は #267、reviewer は #269)」と書く。計画の Verify plan は dated の段落を書き換えず、必要なら 2026-10-29 の行を足すと書いている。#267 の行はまだない。 |
| README の承認の段落 | 一部(情報) | `README.md:263` は per-plan の QUESTION のヘッダを書くが、本文の `FORK` の行と計画助言には触れない。計画は README の変更を求めていない。足すかどうかは /sync-docs の判断。 |
| 計画の AC のチェック | 遅れあり(情報、V4) | AC1〜AC8 は `[ ]` のまま。 |

## Observational checks

AC8(codex-cli 0.163.0。`leader.md:552-554` を byte 単位で写した script を、scratchpad の中で bash と zsh から打った。モデルは呼んでいない。server の名前と設定の中身は出していない。名前の一時 file は数えたあとに消した):

| 確かめたこと | bash | zsh |
| --- | --- | --- |
| 上書きなしの `codex mcp list --json` の server の数 | 20 | (同じ一覧) |
| そのうち `enabled` が `false` でないもの | 18 | (同じ一覧) |
| `--disable plugins` を付けた一覧の数(名前を読む元) | 14 | 14 |
| `-c mcp_servers.<名前>.enabled=false` を 14 個付けた後確認の有効な数 `n` | 0 | 0 |
| 雛形の判定(`n` が 0 なら codex を呼ぶ) | 呼ぶ(rc 0、stderr 0 byte) | 呼ぶ(rc 0、stderr 0 byte) |

- 対照: `--disable plugins` を付けず、上書きなしの一覧で有効な 18 個に `-c ...enabled=false` を付けると、`codex mcp list --json` は rc 1 で stdout が空、stderr に `invalid transport` が出た。`n` は空になり、雛形の判定は `mcp: not off (enabled: )` で呼ばない側に倒れる。D7 の逸れ(plugin の server は `-c` で止まらないので `--disable plugins`)の理由と合う。
- 引数の並び: `codex --disable plugins -m stub-model -c model_reasoning_effort=xhigh exec --sandbox read-only --ignore-rules -o <file> --help` は rc 0 で exec の help を出し、`-o` の file は作られなかった。雛形の exec の前の引数の並びを、この版の codex が受け付ける。
- 展開した雛形: scratchpad の `main.go` を `go run -overlay` で module の中に見せ、`org.RenderRolePrompt("leader", ...)` を OrgID `org-a` で展開した(worktree には何も書いていない。あとの `git status --porcelain` は空)。`### 計画助言` は `## 機能ごとの org` の後ろ、`## タスク` の前にあり、`{{` の残りは 0、BLOCKED の TASK_ID は `org-a` に置き換わる。展開した節を evidence に写した。
- `.harness/state/` はルートと `templates/base/` の `.gitignore` の 47 行目で無視される(`git check-ignore -v` で、計画助言の出力と git の状態の file 名を確かめた)。手順 12 の gate が作業ツリーの汚れとして断らない、という雛形の説明と合う。
- この repo の `ralph.toml` は `[org.review].plan_advisory = true` で、`go run ./cmd/ralph config get org.review.plan_advisory` は `true`(rc 0)を返した。新しい binary でこの repo の機能の org を立てると、leader は承認の QUESTION の前に計画助言を頼む(最長 20 分)。

## Coverage gaps

- テストは打っていない(`go test` と `run-test.sh` は /test の担当)。AC2〜AC6 の「テストが固定する」は、テストの中身を読んで確かめただけで、通るかは /test の結果で決まる。stand-in の codex と claude を bash と zsh で打つテスト(`:954`、`:1073`)は、とくに /test の結果を見る必要がある。
- 変異は打ち直していない。Progress の S1(新しい 6 本が HEAD の雛形で落ちる)、S3(20 件と 2 件)、S2(11 本と変異 7 つ)の結果は implementer の記録で、こちらでは見ていない。
- 実物の leader と director の通しは見ていない(計画の Non-goals)。codex の leader の sandbox から `claude -p` が通るかも見ていない(Non-goals。通らなければ飛ばして記録する)。AskUserQuestion で 4 問を 1 回に出す動きも実物では見ていない。
- AC8 の数は、この machine の codex の設定(server 20 個、そのうち `--disable plugins` で一覧から外れるもの 6 個)での結果。`plugins` の機能名を知らない古い codex で飛ばす道は、stand-in のテスト(`STUB_OLD`)だけが確かめる。
- PR の本文(release のノート、D1 と D3 の外れ)はまだないので未確認。
- V1(LOW): spec の `docs/specs/2026-10-07-org-multi-org-director.md:129` は、#265 の段落で「設定を読む座席はまだない(leader は #267)」と書いたまま。dated の段落なので書き換えず、#267 で leader が `plan_advisory` を読むようになったこと、承認の QUESTION に `FORK` と `ADVISORY:` の行が入ること、director が分かれ目を承認と一緒に聞くことを、2026-10-29 の新しい行として足すとよい。計画の Verify plan が名指ししている確かめで、/sync-docs に渡す。
- V2(情報): leader の手順 4(`leader.md:247-249`)と `/org` skill(`:926-927`)の「本文が `approved` の承認は、すべての分かれ目が推奨の案で決まったことも表す」は、2 往復目で `(決定済み)` の分かれ目が推奨と違う案で決まっていたときに文字どおりには当たらない。どちらの場合も「`## Design decisions` を書き換えない」という動きは正しいので、害はない。
- V3(情報): AC5 の文の `pick_reviewer "$RALPH_PRIMARY_CLI"` と雛形の `"${RALPH_PRIMARY_CLI:-<あなたの CLI>}"` の違いは、self-review L5 と L7 の直しで、Progress に記録がある。AC の中身(相手のモデルを選ぶ)は満たす。
- V4(情報): 計画の AC1〜AC8 のチェックが `[ ]` のまま。
- V5(情報): issue #267 の文と D1・D3 の違い(上の「記録された外れ」)。PR の本文で触れると、issue を閉じるときに迷わない。
- 既知で /sync-docs に渡すもの(この PR の所見として数えていない): `README.md:259`、`/org` skill の `:31-34`(4 つの写し)、`docs/recipes/isolated-herdr-server.md:241-249`(テンプレートの写しも)、`/org` skill の「機能の計画の承認」の `ADVISORY:`、`paneenv.go:42` の L8。

所見の数: LOW 1(V1)、情報 4(V2〜V5)。CRITICAL、HIGH、MEDIUM はない。

最も安く確信を上げる追加の確認: /test で `go test ./internal/org/ -run 'RenderRolePrompt_Leader|DirectorSkill'` を通すこと。とくに `TestRenderRolePrompt_Leader_FeatureOrgPlanAdvisoryCodexCommandRuns`(MCP の止め方、残った server、`enabled` の欄なし、古い codex、書き込み、hang の 6 通りを bash と zsh で打つ)と `TestDirectorSkill_PinsTheApprovalForkSteps` が、AC5 と AC3 を直接固定する。

release のノートの材料(/pr 向け):

- `[org.review].plan_advisory = true` で、機能の org の leader が承認の QUESTION の前(split では implementer の spawn の前)に、相手のモデル(claude の leader は codex、codex の leader は claude)に読むだけの計画助言を頼む。既定は off。助言者の MCP の server は止め、止められなければ呼ばずに飛ばして記録する。最長 20 分。
- 承認の QUESTION の本文に、機能の内側の分かれ目が `FORK` の行で載り、計画助言の要約が `ADVISORY:` の行で載る。director は分かれ目を承認と一緒に聞く(3 つまでは 1 回、4 つ以上は分けて先に聞く)。
- 推奨と違う案を選ぶと、approve は打たれず直しの DECISION になり、leader が計画を直して承認を取り直す(2 往復)。split では leader が推奨の案を採り、計画に記録する。
- leader の役割の雛形は go:embed なので、新しい binary で立てた leader から効く。director の skill は、新しい skill を読んだ director から効く。この repo の `ralph.toml` は `[org.review].plan_advisory = true` なので、ここで立てる機能の org の leader は計画助言を頼むようになる。

## Verdict

pass

- Verified: AC1、AC2、AC3、AC4、AC5、AC7 は雛形と skill の文を読み、テストがそれを固定していることを確かめた。AC8 は雛形の 3 行を bash と zsh で打ち直し、後確認の有効な数が 0 であることを確かめた。指示文は `/plan` の手順 11.c と、claude の打ち方と git の状態の取り方は #266 と一致した。4 つの写しは一致し、静的検査はすべて rc 0、計画の digest は承認のときのまま。
- Partially verified: AC6(同期の検査は通った。`go test` は /test)。issue #267 の「1 回の AskUserQuestion」と「答えを DECISION に入れる」は、D3 と D1 の記録された外れのとおり。
- Not verified: テストの実行、変異の打ち直し、実物の leader と director の通し、codex の sandbox からの `claude -p`、PR の本文。
