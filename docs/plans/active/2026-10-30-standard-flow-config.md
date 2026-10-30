# standard-flow-config

- Status: Approved
- Approved: 2026-10-30 sha256:d8e03ef62de9
- Owner: Claude Code
- Date: 2026-10-30
- Related request: #276 の 3 本目(B)の前半(B1)。標準フローの値(パイプラインの回し直しの上限、reviewer のモデルと effort)を `ralph.toml` と `ralph.local.toml` に移し、環境変数での上書きをやめる
- Related issue: 276
- Type: feat
- Branch: feat/standard-flow-config

## Objective

#276 は、人が決める値を `ralph.toml` と git が無視する個人用のファイル `ralph.local.toml` だけで決め、環境変数での上書きをやめる issue です。docs/plans/archive/2026-10-29-local-config-org.md の D1 で 3 本に分けました。

- A1(PR #287): 個人用のファイルの仕組みを足し、`RALPH_ORG_AGMSG_HOME` を撤去した。
- A2(PR #290): `RALPH_ORG_STATE_DIR` を撤去した。

残りの B は、この計画でさらに 2 本に分けます(D1)。

- **B1(この計画、`Refs #276`)**: 標準フローの値を受け持つ。
  - `RALPH_STANDARD_MAX_PIPELINE_CYCLES` → `[standard].max_pipeline_cycles`(既定 2)
  - `RALPH_CLAUDE_REVIEWER_MODEL` → `[reviewer].claude_model`(既定を opus から fable に変える)
  - claude の reviewer の effort(新設)→ `[reviewer].claude_effort`(既定 xhigh)
  - `RALPH_CODEX_REVIEWER_MODEL` → `[reviewer].codex_model`(既定 gpt-6-astra)
  - `RALPH_CODEX_REASONING_EFFORT` → `[reviewer].codex_effort`(既定 xhigh)
- **B2(次の計画、`Closes #276`)**: スクリプトが読む個人用の値を受け持つ。対象は `RALPH_PLAN_VISUAL_BROWSER`、`RALPH_PLAN_VISUAL_OPENER`、`RALPH_XREVIEW_BASE`、`RALPH_VERIFY_BASE`、`RALPH_VERIFY_SKIP_BRANCH_SECRET_SCAN`。

この計画の最初に、issue が求める「全変数を分けた一覧」を置きます(B2 の分も含む)。

## 全変数の一覧(issue の規則で分ける)

規則は issue のとおりです。人が決める値は `ralph.toml`(と個人用のファイル)だけで決め、環境変数での上書きはやめます。ralph やスクリプトが内部で受け渡す変数は残します。出典は scratchpad の 276b-env-inventory.md(main 0351800d、file:line 付き)です。

| 変数 | 誰が設定するか | 分け方 | 行き先・扱い |
|---|---|---|---|
| `RALPH_STANDARD_MAX_PIPELINE_CYCLES` | 人 | 人が決める値 | B1: `[standard].max_pipeline_cycles` |
| `RALPH_CLAUDE_REVIEWER_MODEL` | 人 | 人が決める値 | B1: `[reviewer].claude_model` |
| (新設)claude の reviewer の effort | 人 | 人が決める値 | B1: `[reviewer].claude_effort` |
| `RALPH_CODEX_REVIEWER_MODEL` | 人 | 人が決める値 | B1: `[reviewer].codex_model` |
| `RALPH_CODEX_REASONING_EFFORT` | 人 | 人が決める値 | B1: `[reviewer].codex_effort` |
| `RALPH_PLAN_VISUAL_BROWSER`、`RALPH_PLAN_VISUAL_OPENER` | 人(テスト) | 人が決める値 | B2: 個人用のファイル |
| `RALPH_XREVIEW_BASE`、`RALPH_VERIFY_BASE` | 人(テスト) | 人が決める値 | B2: 個人用のファイル |
| `RALPH_VERIFY_SKIP_BRANCH_SECRET_SCAN` | 人(テスト) | 人が決める値 | B2: 個人用のファイル |
| `RALPH_ORG_AGMSG_HOME`、`RALPH_ORG_STATE_DIR` | 人 | 人が決める値 | A1・A2 で撤去済み(doctor が warn、`RALPH_ORG_STATE_DIR` は台帳を変えるコマンドの拒否にだけ読む) |
| `HARNESS_VERIFY_MODE` | `run-static-verify.sh`・`run-test.sh` | 内部の受け渡し | 残す |
| `RALPH_VERIFY_SCOPE` | CI の workflow、org の gate、wrapper | 内部の受け渡し | 残す |
| `RALPH_VERIFY_PROJECT_ROOTS` | `run-verify.sh` がパックに渡す | 内部の受け渡し | 残す |
| `RALPH_WIP_CHECKPOINT` | ralph が pane と `claude -p` に渡す | 内部の受け渡し | 残す |
| `RALPH_PRIMARY_CLI` | ralph が座席の pane に渡す。Codex のセッションは `.codex/AGENTS.override.md` の指示で自分で設定する | 内部の受け渡し(例外の注あり) | 残す。値は「いま動いている CLI」で、人が好みで決める値ではない。Codex の場合も、AGENTS.override.md が CLI 自身に設定させている |
| `RALPH_DIRECTOR_TOKEN` | director の起動スクリプト | 内部の受け渡し | 残す |
| `RALPH_LOCAL_CONFIG` | org の gate・regate、テスト。人が `off` にもする | 設定ファイル自体の切り替え | 残す。個人用のファイルを読むかどうかの切り替えなので、そのファイルには書けない |
| `RALPH_SECRET_ALLOWLIST`、`RALPH_SECRET_SCAN_ATTR_SOURCE`、`RALPH_SECRET_SCAN_BINARY_AUTO` | `secret-scan-branch.sh`・`secret-scan.sh` | 内部の受け渡し | 残す |
| `RALPH_COMMON_SOURCED` | `ralph-common.sh` 自身 | 内部(source の印) | 残す |
| `RALPH_GUARD` | 生成する git hook の中の shell 変数 | env ではない | 対象外 |
| `RALPH_TEST_*`、`RALPH_ORG_TEST_*`、`RALPH_CLI_TEST_*`、`RALPH_DRIVER_TEST_INHERITED`、`HARNESS_VERIFY_EXTRA`、`RALPH_PURITY_SCAN_ROOT`、`HOOK_REPO_ROOT`、`CLAUDE_ROOT`、`CODEX_ROOT` | テスト | テスト用 | 残す |
| `RALPH_TERRAFORM_INIT_VALIDATE` | 人(terraform のパックを使う人、CI でも) | 人が決める値(例外) | 残す。理由は D6 |
| `COVERAGE_THRESHOLD` | CI の workflow(`"80"`) | CI が渡す値(例外) | 残す。理由は D6 |
| `INSTALL_DIR` | 人(`scripts/install.sh`、ralph を入れる前) | 外部の道具の設定 | 残す。ralph がまだない段階で読む |
| `GITHUB_BASE_REF`、`GOCACHE`、`STATICCHECK_CACHE`、`GOFLAGS`、`CODEX_HOME` ほか | GitHub Actions・Go・codex など | 外部の道具の env | 対象外 |

## Scope

- **config**(`internal/config/`)
  - `[standard].max_pipeline_cycles`(int、既定 2、1 以上)を足す。
  - 新しい表 `[reviewer]` を足す。
    - `claude_model`(既定 "fable")
    - `claude_effort`(既定 "xhigh"。low・medium・high・xhigh・max のどれか)
    - `codex_model`(既定 "gpt-6-astra")
    - `codex_effort`(既定 "xhigh")
    - どの値も、空ならエラーにする。`claude_model`・`codex_model`・`codex_effort` は `[A-Za-z0-9._:@/+-]+` に限る。値は skill と leader の shell のコマンドに 1 語で写るので、引用符や `$` で割れないようにする。`:`・`@`・`/`・`+` は完全なモデル ID(Bedrock の ID や ARN)に入りうる文字で、shell はそのまま扱う(D4)。
  - 値の誤りは Load がキーを名指してエラーにする(`ralph config get` は終了コード 3)。
  - `ralph config get <key>` は reflection で引くので、キーを足すだけで読める。テストで固定する。
  - `[org.review].max_cross_review_cycles` の既定と `[standard].max_pipeline_cycles` の既定の一致は、Go の中で照合する(今は shell の変数と照合している)。
- **helper**(`scripts/ralph-config.sh` と `templates/base/scripts/` の写し)
  - 人が決める値を持たない、内部用の読み出しにする(D2)。
  - `ralph_setting <key>` が `ralph config get <key>` を打ち、終了コードで次のように分ける。
    - 0 なら、その値を出す。
    - 127(ralph がない)と、exit 1 で `unknown command "config"`(古い ralph)は、黙って既定値を出す。
    - exit 2(知らないキー。#265〜B1 のあいだの binary など)、exit 3(読めない・正しくない設定)、そのほかの失敗は、stderr にキー名・終了コード・使う既定値を 1 行出し、既定値を出す。
  - 既定値は helper の中の 1 つの表にだけ置く。呼び出し側はキーだけを渡す。
  - 環境変数は読まない。`validate_numeric` などの関数は、要らなくなれば消す。
- **skill**(`.claude/skills/` の plan・cross-review・implement と、`templates/base/.claude/skills/` の写し。`.agents/skills/` の 2 つは `sync-skills.sh` で作り直す)
  - `/plan` の手順 11 と `/cross-review` の手順 4 の reviewer の呼び出しは、helper でモデルと effort を読む。
  - claude の reviewer には `--effort <claude_effort>` を渡す(D4)。
  - `/cross-review` の手順 1 は、上限を次の規則で決める(D8)。
    - 設定の上限は `standard.max_pipeline_cycles` で読む。
    - task worktree の `.harness/state/standard-pipeline/cycle-count.json` に正の整数の `cap` があれば、有効な上限は設定の値と `cap` の大きいほう。なければ設定の値。整数でない `cap` は、無いものとして扱う。
  - 上限に当たったときの「上げて回し直す」(Cap-reached Option 1、`export RALPH_STANDARD_MAX_PIPELINE_CYCLES=…` を頼んでいた手順)は、skill 自身が cycle-count.json に `cap: <cycle + 1>` を書く形にする。`cycle` は増やさない。
    - `/pr` の成功でこのファイルは消える(今と同じ)ので、上げた上限は残らない。
    - 中止したときは残り、再開したその作業だけが上げた上限で続く。
  - `/implement` の手順 3 は、cycle-count.json を保つときに `cap` も保つ。上限の名前も合わせる。
- **leader の雛形**(`internal/org/prompts/leader.md`)と feature org の起こし方(`internal/org/feature.go`、`internal/cli/` の `ralph org start`)(D5)
  - `ralph org start --plan` は、読み込んだ設定の `[reviewer]` の 4 つの値を `StartFeatureParams` で受け取る。
  - `featureLeaderTask` は、受け取った値を名前付きの 2 行で TASK に書く。今の `- 台帳:` の行と同じ作り。
    - `- reviewer の codex: <model> <effort>`
    - `- reviewer の claude: <model> <effort>`
  - 計画助言の codex・claude の呼び出しは、この 2 行の値を使う。claude には `--effort` を渡す。
  - 雛形は `scripts/ralph-config.sh` を source せず、`ralph config get reviewer.*` も打たない。leader の PATH にある ralph の版にも、プロジェクトの `scripts/ralph-config.sh` の版にも頼らない。
- **doctor**(`internal/cli/doctor.go` の `retiredEnvs`): 撤去する 4 つの変数を足し、それぞれ代わりのキーを名指す warn を出す。
- **文書**
  - `.claude/rules/ralph/model-routing.md`(「Rules」の full ID の pin と effort、「Cross-review sync note」、「Where the values live」)と templates/base の写し。
  - `post-implementation-pipeline.md`(上限の名前と、「Raise the cap」を cycle-count.json の `cap` に)、`subagent-policy.md`、`docs/quality/definition-of-done.md`、`docs/recipes/codex-setup.md`(と写し)。どれも `export RALPH_STANDARD_MAX_PIPELINE_CYCLES` を名指している箇所を直す。
  - `templates/base/ralph.toml`: `[standard]` と `[reviewer]` の新しいキーを注釈で書く(D3)。`[org.review]` の注釈の変数名も直す。
  - `internal/config/config.go` の注釈。
  - `claude -p --effort` を確かめた結果を `docs/evidence/` に置く。
- **テスト**
  - `tests/test-ralph-config.sh` を helper のテストに書き直す。偽の `ralph` を PATH に置き、終了コード 0、127、unknown command、2、3 と、env が効かないことを見る。
  - `tests/test-codex-exec-invocation.sh`(skill の fallback の照合を helper の呼び出しの照合に)。
  - `internal/config/defaults_sync_test.go`(helper の表と `Default()`、template の注釈)。
  - `internal/org/prompts_leader_plan_test.go`。
  - config・cli のテスト。

## Non-goals

- B2 の 5 つの変数(plan-visual のブラウザと opener、cross-review の base、verify の base、secret scan の skip)。B2 の計画で扱う。
- `[standard]` と `[org.review]` の review の on/off の値と、その読み方(#265・#266 で決めた)。
- subagent の frontmatter の `model:` と、`.claude/agents/*.md` の tier の表。
- `RALPH_TERRAFORM_INIT_VALIDATE` と `COVERAGE_THRESHOLD` の移動(D6)。
- `RALPH_PRIMARY_CLI` の扱い(driver の上書きの重複は #266 の残り)。
- leader が PATH の `ralph` を打つこと自体。`ralph config get org.review.plan_advisory`(leader.md の計画助言の switch)と `ralph org` の動詞は、これまでどおり PATH の ralph を打つ。新しい binary を絶対パスで起こし、PATH に古い ralph がある場合のずれは残る(計画助言の指摘 1 のうち、この計画が直すのは reviewer の値だけ)。
- 古い `ralph` binary を入れ直すこと。

## Assumptions

- `ralph config get` は `config.Load` で `ralph.toml` と `ralph.local.toml` を合わせて読む(A1)。キーを足せば読める(`internal/cli/config.go` の `configValue`)。
- skill と `scripts/ralph-config.sh` は、配布先では `ralph upgrade` で同じ版がそろう。この repo では両方が同じ tree にある。
- leader の雛形は binary に埋め込まれているので、leader が打つ `ralph` は雛形と同じ版を持つ(`ralph org start` が起こす)。
- `claude -p --effort` は効く。2026-10-30 に claude 2.1.308(haiku)で確かめた。
  - 正しくない値は「Unknown --effort value … using the default effort」の warning が出る。
  - 同じ質問で、thinking のトークンは `low` 0、`max` 1066 だった。
  - 結果は S6 で `docs/evidence/` に置く。
- この repo の PATH の ralph は v6.0.0 で `config get` がない。helper は既定値に戻る。

## Affected areas

- `internal/config/`(`config.go`、テスト、`defaults_sync_test.go`)
- `internal/cli/`(`config.go` のテスト、`doctor.go` の `retiredEnvs` とテスト)
- `internal/org/prompts/leader.md`、`internal/org/prompts_leader_plan_test.go`、`internal/org/feature.go`(`StartFeatureParams`、`featureLeaderTask`)とテスト、`internal/cli/` の `ralph org start --plan`(設定の受け渡し)
- `tests/test-insights-append.sh`(`cap` のある cycle-count.json)
- `scripts/ralph-config.sh`(と `templates/base/scripts/` の写し)
- `.claude/skills/{plan,cross-review,implement}/SKILL.md`(と 3 つの写し)
- `.claude/rules/ralph/{model-routing,post-implementation-pipeline,subagent-policy}.md`(と templates/base の写し)
- `docs/quality/definition-of-done.md`、`docs/recipes/codex-setup.md`(と写し)、`templates/base/ralph.toml`
- `tests/test-ralph-config.sh`、`tests/test-codex-exec-invocation.sh`

## Visual review

- ページ: `.harness/state/plan-visual/standard-flow-config.html`。かんたん版の図は `overview`(今とこれから)の 1 つ。折りたたみの中に、詳しい図が 3 つある。
  - `detail-overview`: ファイルの地図
  - `detail-read`: `ralph_setting` の読み方と終了コード
  - `detail-keys`: 変数とキーの対応
- lint: `./scripts/plan-visual.sh lint` が exit 0。
- self-check: 全体と 4 つの図を `shot` で撮り、PNG を読んだ。はみ出し、重なり、線の行き違いはない。
- 計画助言を受けて直した箇所: D5 と D8 に合わせて、ファイルの地図の leader の箱、2 つの注、「承認の前に」の S4 と気をつけることの 2 項目を書き直した。lint(exit 0)のあと、全体・`detail-overview`・`detail-read` を撮り直して確かめた。

## Design decisions

分かれ目は consult(consult-plan-standardcfg276b)に聞き、推奨案を選んだ(ユーザーの指示)。

- D1 B を 2 本に分ける。B1(この計画、標準フローの値、`Refs #276`)と B2(スクリプトの個人用の値、`Closes #276`)。分けないと、差分は 1,500 行前後になりそう(見積もり、未確認)。書き換えるテストも、shell 9 本と Go 3 本に広がる。B1 と B2 は触るスクリプトがほとんど重ならない(B1 は ralph-config.sh と skill、B2 は plan-visual.sh・xreview-helpers.sh・detect-changed-languages.sh・run-verify.sh)。前例は A1・A2。
- D2 値の読み方: `scripts/ralph-config.sh` を、`ralph config get` で読む内部用の helper `ralph_setting <key>` に作り替える(案 a)。
  - 既定値は helper の中の 1 つの表だけに置き、`Default()` との一致は `defaults_sync_test.go` で見る。
  - 黙って既定値に戻るのは、127 と `unknown command "config"` の exit 1 だけ。
  - exit 2・3 とそのほかの失敗は、stderr に 1 行出して既定値に戻る。`ralph config get --help` と template の注釈が「exit 2 は止まって知らせる」と約束しており、キーの打ち間違いや改名を見えるようにするため(consult の指摘)。
  - 退けた案は 2 つ。各 skill に `ralph config get` を直に書く案は、既定値と分岐が 4 つの写しと雛形に散らばる。shell で TOML を読む案は壊れやすい。
- D3 キーの置き場所と書き方
  - 上限は `[standard].max_pipeline_cycles`。
  - reviewer のモデルと effort は、新しい共有の表 `[reviewer]`(`claude_model`、`claude_effort`、`codex_model`、`codex_effort`)。`/plan`・`/cross-review`・feature org の leader の 3 か所が同じ値を使うので、`[standard]` に置いて leader が `standard.*` を読む形は避けた。
  - `templates/base/ralph.toml` には注釈で書く。有効な行で書くと、ralph.toml は seed で `ralph upgrade` が上書きしないため、`Default()` を変えても配布先に古い値が残る。
  - この repo のルートの `ralph.toml` は、テンプレートとの違いだけを書く形のままにする(#265)。
  - 4 つのキーは local-only にしない。モデルは repo で決める値で、自分の機械だけで変えたいときは個人用のファイルが上書きする。
- D4 claude の reviewer に `--effort` を渡す。値は `[reviewer].claude_effort`。`-p` で効くことを確かめた(Assumptions)。
  - Load は `claude_effort` を claude の 5 つ(low・medium・high・xhigh・max)に限る。claude は正しくない値を warning を出して無視するので、先に断る。
  - codex の effort は、モデルによって受け付ける値が違う。そのため形だけを見る。
  - モデル名と codex の effort は `[A-Za-z0-9._:@/+-]+` に限る。skill と leader のコマンドに 1 語で写るので、引用符や `$` で shell が割れないようにする(consult の指摘)。完全なモデル ID は `:`・`@`・`/`・`+` を含みうるので、これらは許す。
- D5 leader の reviewer の値は、`ralph org start` が TASK に書く。
  - 計画助言の指摘 1(MEDIUM)を受けて変えた。leader が打つ `ralph` は PATH で決まり、雛形を埋め込んだ binary と同じ版とは限らない。
  - 雛形を埋める binary(StartFeature)は `[reviewer]` を知っている版なので、その値を TASK に名前付きの 2 行で書く。leader は実行時に読まない。
  - 値は org を起こしたときのものに固定され、org の寿命のあいだ変わらない。
  - 雛形の「`scripts/ralph-config.sh` があるか」の確かめは外し、`xreview-helpers.sh` の確かめは残す。
  - 退けた案:
    - `ralph config get reviewer.*` を雛形で打つ(初めの案)。PATH の古い ralph では exit 2 になる。
    - `os.Executable` のパスを雛形に渡し、すべての ralph の呼び出しに使う。範囲が広い。
- D6 表にない 2 つの変数は残し、一覧と PR 本文に例外として理由を書く。
  - `COVERAGE_THRESHOLD`: CI の workflow が渡す値で、CI には ralph がない。`ralph.toml` に移すと、CI では黙って既定の 80 になる。
  - `RALPH_TERRAFORM_INIT_VALIDATE`: terraform のパックの選択で、CI で使うことがある。理由は同じ。
  - issue の受け入れ条件「残した環境変数がすべて内部の受け渡し」は、字句どおりには満たさない。`RALPH_PRIMARY_CLI` も同じ書き方で注を付ける(consult の指摘)。
- D7 撤去した 4 つの変数がまだ設定されているとき: 値として読まない。doctor の `retiredEnvs` に足して warn を出すだけにする(A1 の D8 と同じ)。値を読まなくなったときの害は、どれも既定値に戻ることなので小さい。helper は環境を見ない。
- D8 その作業だけ上限を上げるときは、task worktree の cycle-count.json に `cap` を書く。
  - 計画助言の指摘 2(MEDIUM)を受けて変えた。`ralph.local.toml` は主 worktree の root の 1 つのファイルなので、そこに書くと、ほかの linked worktree の作業にも効く。完了しても中止しても元に戻らない。
  - `.harness/state` は worktree ごとにあり、`/pr` の成功で cycle-count.json は消える。だから上げた上限はその作業の中だけで効き、残らない。
  - 有効な上限は、設定の値と正の整数の `cap` の大きいほう。上げる手順では上限は下がらない。整数でない `cap` は無いものとして扱う。
  - `cycle` は今どおり増やさない。`insights-append.sh` は `.cycle` だけを読むので、`cap` が増えても壊れない。
  - `[standard].max_pipeline_cycles` は、「この repo(個人用のファイルならこの機械)の既定の上限」として残す。

## Acceptance criteria

- [ ] AC1 既定値と上書き
  - `ralph config get` の既定値は次のとおり。
    - `standard.max_pipeline_cycles`: `2`
    - `reviewer.claude_model`: `fable`
    - `reviewer.claude_effort`: `xhigh`
    - `reviewer.codex_model`: `gpt-6-astra`
    - `reviewer.codex_effort`: `xhigh`
  - `ralph.toml` の値と、それを上書きする `ralph.local.toml` の値が効く。
  - 次の値では、Load がキーを名指してエラーにする(`config get` は exit 3)。
    - `max_pipeline_cycles` が 1 未満
    - `claude_effort` が 5 つの外
    - 4 つの文字列のどれかが空
    - `claude_model`・`codex_model`・`codex_effort` に `[A-Za-z0-9._:@/+-]` 以外の文字(引用符、`$`、空白など)がある
- [ ] AC2 skill(plan、cross-review、implement の 4 つの写し)は、この 5 つの値を設定から読む。leader の雛形は、`ralph org start` が TASK に書いた reviewer の 2 行の値を使う(AC4)。
  - `RALPH_STANDARD_MAX_PIPELINE_CYCLES`、`RALPH_CLAUDE_REVIEWER_MODEL`、`RALPH_CODEX_REVIEWER_MODEL`、`RALPH_CODEX_REASONING_EFFORT` を設定しても、使われる値は変わらない(helper のテストで env を設定して確かめる)。
  - skill、スクリプト、雛形の本文に `${RALPH_…:-…}` の形の読み出しが残らない(grep の否定)。
- [ ] AC3 helper の `ralph_setting <key>` の振る舞い
  - exit 0 ではその値を出す。
  - 127 と `unknown command "config"` では、黙って既定値を出す。
  - exit 2・3 とそのほかの失敗では、stderr に 1 行(キー、終了コード、既定値)を出して既定値を出す。
  - 表にないキーでは、空を出して失敗する。
  - helper の既定値の表は `Default()` と一致する(`defaults_sync_test.go`)。
- [ ] AC4 claude の reviewer の呼び出しに `--effort` が付く。
  - 対象は `/plan` の計画助言、`/cross-review`、leader の計画助言の 3 つで、値は `reviewer.claude_effort`。
  - leader の分は、`featureLeaderTask` が設定の値で `- reviewer の codex: <model> <effort>` と `- reviewer の claude: <model> <effort>` の 2 行を書き、雛形がその値を使う。雛形には `scripts/ralph-config.sh` の source も `ralph config get reviewer` もない。
  - `claude -p --effort` を確かめた結果が `docs/evidence/` にある。
- [ ] AC5 `/cross-review` の上限
  - 有効な上限は、設定の `standard.max_pipeline_cycles` と、task worktree の cycle-count.json の正の整数の `cap` の大きいほう。`cap` がない、または整数でないときは設定の値。
  - 上限に当たって上げるときは、skill が cycle-count.json に `cap: <cycle + 1>` を書き、`cycle` は増やさない。環境変数も `ralph.local.toml` も頼まない。
  - `/implement` の手順 3 は `cap` を保つ。`/pr` の成功でファイルごと消える(今と同じ)。
- [ ] AC6 `ralph doctor` は撤去した 4 つの変数が空でない値で設定されているとき、それぞれ代わりのキーを名指す warn を出す。設定されていなければ何も出さない。
- [ ] AC7 文書
  - model-routing.md の full ID の pin、effort、sync note、「Where the values live」を新しい置き場所と既定値に合わせる。post-implementation-pipeline.md、subagent-policy.md、definition-of-done.md、codex-setup.md、template の ralph.toml の注釈も同じ。
  - 4 つの変数に触れるのは「もう読まない」の説明だけになる。
- [ ] AC8 `go build ./...`、gofmt、`go vet`、golangci-lint、`check-sync.sh`、`check-skill-sync.sh`、`./scripts/run-verify.sh` が通る。

## Implementation outline

1. S1 config: `[standard].max_pipeline_cycles` と `[reviewer]` の 4 つのキー、検証、`Default()`、template の注釈、`standardTemplateDrift` と template の照合の直し、`[org.review]` の上限との Go の中の照合、`ralph config get` のテスト — verify: `go test ./internal/config/ && go test ./internal/cli/ -run 'Config'`
2. S2 helper: `scripts/ralph-config.sh` を `ralph_setting` に作り替え(写しも)、`tests/test-ralph-config.sh` を書き直し、`defaults_sync_test.go` で helper の表と `Default()` を照合 — verify: `sh tests/test-ralph-config.sh && go test ./internal/config/ -run 'Sync|Drift|Default' && ./scripts/check-sync.sh`
3. S3 skill: plan・cross-review・implement の 2 つの元を直し(reviewer の値、claude の `--effort`、上限の規則と cycle-count.json の `cap`)、`sync-skills.sh` で写しを作り直し、`tests/test-codex-exec-invocation.sh` を合わせる — verify: `./scripts/check-skill-sync.sh && ./scripts/check-sync.sh && sh tests/test-codex-exec-invocation.sh && sh tests/test-insights-append.sh && go test ./internal/config/ -run 'Sync'`
4. S4 leader: `StartFeatureParams` で `[reviewer]` を受け取り(`ralph org start` が読み込んだ設定から)、`featureLeaderTask` に 2 行、雛形はその値を使い claude に `--effort`、`prompts_leader_plan_test.go` と feature の TASK のテスト — verify: `go test ./internal/org/ -run 'Leader|Prompt|Feature' && go test ./internal/cli/ -run 'OrgStart|Feature'`
5. S5 doctor の `retiredEnvs` に 4 行とテスト — verify: `go test ./internal/cli/ -run 'Retired|Doctor'`
6. S6 文書と evidence: model-routing.md・post-implementation-pipeline.md・subagent-policy.md(と写し)、definition-of-done.md、codex-setup.md(と写し)、`config.go` の注釈、`docs/evidence/` に effort の確かめ — verify: `./scripts/check-sync.sh && sh tests/test-agent-models.sh && ! grep -rnE '\$\{RALPH_(STANDARD_MAX_PIPELINE_CYCLES|CLAUDE_REVIEWER_MODEL|CODEX_REVIEWER_MODEL|CODEX_REASONING_EFFORT)' .claude scripts templates internal --include='*.md' --include='*.sh' --include='*.go'`

## Verify plan

- 静的解析: gofmt、`go vet`、golangci-lint、shellcheck(`run-static-verify.sh` に含まれる)。
- 仕様の適合: AC1〜AC7 を、HEAD から build した binary の probe(一時 repo の `ralph config get`、`ralph doctor`)と、grep とコードの行で確かめる。
- 文書のずれ: 4 つの変数の名前の grep(「もう読まない」の説明と、記録の docs/plans・docs/reports・docs/specs を除く)。`check-sync.sh`(写しの一致)、`check-skill-sync.sh`。
- `standardTemplateDrift` の直し方を確かめる。注釈にしたキーは「template に set されている」照合から外し、注釈の値が `Default()` と同じであることを別に照合する。
- evidence: probe の出力、grep の結果。

## Test plan

- 単体(Go)
  - 5 つのキーの既定値、`ralph.toml` の値、`ralph.local.toml` の上書き。
  - 検証のエラー: 上限 0 と負、claude_effort の外の値、空の文字列。どれもキーを名指す。
  - `ralph config get` の 5 つのキー(exit 0)。
  - doctor の 4 行(設定あり・空・未設定)。
- 照合(Go)
  - helper の既定値の表と `Default()`。
  - `[org.review].max_cross_review_cycles` と `[standard].max_pipeline_cycles` の既定。
  - template の注釈と `Default()`。
  - skill と雛形の本文が helper か `ralph config get` で読むこと(`${RALPH_…:-…}` がないこと)。
- shell
  - `tests/test-ralph-config.sh`: 偽の `ralph` を PATH に置き、exit 0・127・unknown command・2・3 と表にないキーを確かめる。env を設定しても値が変わらないことも見る。
  - `tests/test-codex-exec-invocation.sh`: skill の codex の呼び出しの形(`-c` が exec の前、`</dev/null` など)と helper の使い方。
- 雛形と feature org
  - `prompts_leader_plan_test.go` で、leader の計画助言の codex・claude の呼び出しが TASK の reviewer の 2 行の値を使い、claude に `--effort` が付くことを見る。雛形に `ralph-config.sh` と `ralph config get reviewer` がないことも見る。
  - `featureLeaderTask` の 2 行が、`ralph.toml` と `ralph.local.toml` で決めた値になることを見る(cli の `ralph org start --plan` の経路で)。
- 上限(skill の本文の照合と、cycle-count.json の形)
  - 有効な上限の規則(設定と `cap` の大きいほう、整数でない `cap` は無視)が cross-review の本文にあること。
  - 上げる手順が `cap` を書き、`cycle` を増やさないこと。
  - `/implement` の手順 3 が `cap` を保つこと。
  - `tests/test-insights-append.sh` に `cap` のある cycle-count.json の行を足し、`--cycle auto` が `.cycle` を読み続けることを見る。
- 回帰: `run-test.sh` 全体。
- 端の場合
  - #265〜B1 のあいだの binary(`config get` はあるがキーがない、exit 2)で、warn を出して既定値に戻る。
  - 個人用のファイルが壊れている(exit 3)ときも、warn を出して既定値に戻る。
- evidence: テストの出力、変異(helper の分岐を変えるとテストが落ちる)。

## Risks and mitigations

- この repo の PATH の ralph(v6.0.0)は `config get` がないので、helper は常に既定値に戻る。
  - 上限を上げる手順は cycle-count.json の `cap` を使うので、古い binary でも効く(D8)。
  - 対策: release のあとに新しい binary を入れるよう、PR 本文と最終の報告に書く。
  - この PR 自身のパイプラインは main のチェックアウトの skill(この PR の前の版)で回るので、上限は今の手順で上げる。
- leader は、reviewer の値を TASK で受け取る。ただし switch(`org.review.plan_advisory`)と `ralph org` の動詞は、これまでどおり PATH の ralph を打つ(Non-goals)。
- ユーザーの今の進め方の指示(上限に当たったら `export RALPH_STANDARD_MAX_PIPELINE_CYCLES=3`)は、この PR のあとの skill では効かない。代わりに cycle-count.json の `cap` を使う。最終の報告で伝える。
- 既定のモデルが opus から fable に変わる。Codex が driver で claude が reviewer のとき、使うモデルが変わる。
  - 対策: release のノートに書く。ralph.toml か ralph.local.toml の `[reviewer].claude_model` で戻せる。
- 4 つの変数を設定していた人は、更新のあと黙って既定値に戻る。
  - 対策: doctor の warn と release のノート。害は小さい(既定値に戻るだけ)。
- `ralph upgrade` をしていないプロジェクトで、新しい binary の leader が古い `scripts/ralph-config.sh` に出会う。
  - 対策: D5。leader は helper に頼らない。
- 3 セッションの並行で、skill の 4 つの写し、template の ralph.toml、`docs/tech-debt/README.md`、model-routing.md が衝突しうる。
  - 対策: `git merge origin/main` で取り込み、両方の変更を残す。skill は元の 2 つを直して `sync-skills.sh` で作り直す。

## Rollout or rollback notes

- release のノートに書くこと:
  - ralph は、`RALPH_STANDARD_MAX_PIPELINE_CYCLES`、`RALPH_CLAUDE_REVIEWER_MODEL`、`RALPH_CODEX_REVIEWER_MODEL`、`RALPH_CODEX_REASONING_EFFORT` を読まなくなった。代わりに `ralph.toml` に書く。自分の機械だけで変えるなら `ralph.local.toml` に書く。置き場所は `[standard].max_pipeline_cycles` と `[reviewer]` の 4 つのキー。
  - claude の reviewer の既定は fable と xhigh になり、`--effort` を渡すようになった。
  - パイプラインの上限に当たって 1 つの作業だけ上げるときは、`/cross-review` がその作業の cycle-count.json に `cap` を書く。環境変数で上げる手順はなくなった。
  - feature org の leader の計画助言は、`ralph org start` のときの `[reviewer]` の値を使う。
  - `ralph doctor` は、まだ設定されている 4 つの変数に warn を出す。
  - skill と helper を使うには、新しい binary が要る。古い binary では既定値になる。
- 戻すときは、この PR を revert する。設定ファイルの形式は、キーが増えるだけ。revert した ralph は `[reviewer]` を知らないキーとして無視する(A1 の Load は知らないキーを無視する。推測、未確認)。env での上書きが戻る。

## Open questions

- なし。B2 の範囲は Objective のとおり。

## Progress checklist

- [x] Plan reviewed — plan advisory: ran with codex
  - 設定と実行: switch は scratchpad の ralph(main 7522835c から build)で `standard.plan_advisory` = true と読んだ。PATH の ralph は v6.0.0 で `config get` がない。codex は gpt-6-astra、xhigh、read-only。前後の git の状態は同じ。
  - 指摘 2 件(どちらも MEDIUM)は、ユーザーの指示(HIGH と MEDIUM は聞かずに反映)どおり計画に入れた。直し方は consult(consult-plan-standardcfg276b)に聞き、「変更なし」の判定を受けた。
  1. [MEDIUM] leader が打つ `ralph` は PATH で決まり、雛形を埋め込んだ binary と同じ版とは限らない。古い版では `reviewer.*` がなく、計画助言が incomplete になる。
     - D5 を変えた。`ralph org start` が読み込んだ `[reviewer]` を `StartFeatureParams` で受け取り、`featureLeaderTask` が名前付きの 2 行で TASK に書く。雛形はその値を使い、実行時には読まない。
     - consult の指摘で 2 つ足した。モデル名と codex の effort を `[A-Za-z0-9._-]+` に限る(D4、AC1)。switch と `ralph org` の動詞が PATH の ralph を打つことは Non-goals に書いた。
  2. [MEDIUM] 「上限を一時的に上げる」を `ralph.local.toml` に書くと、主 worktree の 1 つのファイルなので、ほかの作業にも効き、元に戻らない。
     - D8 を足した。上げる手順では skill が task worktree の cycle-count.json に `cap: <cycle + 1>` を書く。有効な上限は、設定の値と `cap` の大きいほう。`/pr` の成功でファイルが消えるので、上げた値は残らない。
     - Scope、AC5、Test plan(`tests/test-insights-append.sh` の `cap` の行を含む)、Risks、Rollout と図を直した。
- [x] Plan approved — 自動承認(ユーザーの指示)。digest sha256:af46dcbdd0f8。分かれ目は consult(consult-plan-standardcfg276b)に聞いて推奨案を選び、理由は Design decisions(D1〜D8)に書いた。
  - `claude -p --effort` の確かめ(2026-10-30、claude 2.1.308、haiku、git の外の scratchpad で `RALPH_WIP_CHECKPOINT=off`):
    - `--effort bogus` は rc 0 で、stderr に「Unknown --effort value 'bogus' — ignoring it and using the default effort. Valid values: low, medium, high, xhigh, max.」が出た。
    - 同じ質問で、thinking のトークンは `--effort low` 0、`--effort max` 1066 だった。
    - 別の短い質問では、既定(bogus)58、`xhigh` 236 だった。
    - `-p` でも効くと判断した。S6 で `docs/evidence/` に置く。
  - 全変数の一覧の出典: scratchpad の 276b-env-inventory.md(Explore の agent が main 0351800d で作った、137 行)。
- [x] Branch created — feat/standard-flow-config(worktree `.claude/worktrees/standard-flow-config`、state id `plan-standard-flow-config`、base 0351800d)
- [x] Implementation started
  - S1 f66c4a4a + 4f3cc3d4(impl-276b1-s1)
    - 足したもの: `StandardConfig.MaxPipelineCycles`(`max_pipeline_cycles`、既定 2)と `Config.Reviewer`(`ReviewerConfig`)の 4 つのキー。
    - 検証: `validateLoaded` の末尾で上限が 1 未満ならエラーにし、新しい `validateReviewer` で claude_effort を 5 つの値に限り、モデル名と codex_effort を `reviewerWordPattern` で見る。どれもキーと値を名指す。
    - template: `[standard]` に `# max_pipeline_cycles = 2`。`[reviewer]` は有効な空の表にし、4 つのキーを注釈で書いた。空の表は今の Load も旧 binary も無視する。
    - 照合: `defaults_sync_test.go` に、注釈のキーと `Default()` の照合(有効な行・注釈なし・2 回・値の違いを drift にする)、全キーがどちらかの照合に載る固定、`org_review_cycles`(Go の中で上限を照合)を足した。shell と SKILL.md の照合は S2・S3 まで残す。
    - 検証: config 100 本・cli `-run Config` 65 本 rc=0、golangci-lint 0 件、check-sync rc=0、変異 5 つはすべてテストが落とした。self-check は 5 項目とも clean。
    - 計画とのずれ(orchestrator の判断): モデル名と codex_effort に許す文字を `[A-Za-z0-9._-]` から `[A-Za-z0-9._:@/+-]` に広げた(4f3cc3d4)。完全なモデル ID(Bedrock の `…-v1:0`、ARN)を断らないため。`:`・`@`・`/`・`+` は shell の 1 語の中でも展開されない。計画の Scope・D4・AC1 を直し、digest を af46dcbdd0f8 から d8e03ef62de9 に取り直した(自動承認、ユーザーの指示)。図には正規表現が出てこないので撮り直していない。
    - consult(consult-plan-standardcfg276b)の注意 2 点:
      1. template と doc コメントが S2〜S4 の挙動を先に書いているので、S4 の終わりに文と実装を突き合わせる。
      2. PR 本文に許す文字(`A-Z a-z 0-9 . _ : @ / + -`)を 1 行書く。
    - 範囲外: `internal/config/config.go:875`・`:927` と `config_test.go:419` のコメントが、もうない `PipelineConfig` を指している(前から)。`docs/recipes/codex-setup.md:93` は S6。
  - S2 27d3cbfc(impl-276b1-s2)
    - `scripts/ralph-config.sh`(と写し)を作り替えた。既定値の表は `ralph_setting_default <key>` の `<key>) printf '%s\n' '<value>' ;;`、読み出しは `ralph_setting <key>`。
    - 振る舞い:
      - 表にないキーは 2 を返す。
      - exit 0 なら値を出す。
      - 127 と `unknown command "config"` は黙って既定値。
      - そのほかは stderr に 1 行(例 `ralph-config.sh: ralph config get reviewer.codex_model exited 2 (Error: …); using the default gpt-6-astra`)を出して既定値。
      - 環境変数は読まない。`validate_*` は使う所がないので消した。
    - テスト: `tests/test-ralph-config.sh` を偽の ralph で書き直した。sh・bash・dash・zsh `-f` で全ケースを回し、一時ファイルが消えることも見る。
    - 一時の照合(S3・S4 で書き直す前提。どのコミットでもテストが通るように範囲に入れた):
      - `defaults_sync_test.go` の `shell_and_skills` は helper の表と `Default()` を照合し、skill の codex の fallback を表と照合する。
      - claude のモデルの fallback(skill と leader の `opus`)は、S1 で既定を fable にしたので表と合わない。literal の opus で照合し、`prompts_leader_plan_test.go` は ralph-config.sh との照合の行を消した(orchestrator が承認)。
      - `tests/test-codex-exec-invocation.sh` も表を読む形にした。
    - 検証:
      - `test-ralph-config.sh` 370/370、`test-codex-exec-invocation.sh` 144/144、check-sync rc=0。
      - config `-run 'Sync|Drift|Default|LockStep'` 73 本、org `-run 'Leader|Prompt'` 280 本 rc=0(load 17)、golangci-lint 0 件。
      - 変異 5 つと表の書き換えは、すべてテストが落とした。
      - 素の shellcheck は `tests/test-codex-exec-invocation.sh` の SC2016(info)3 件で rc=1。HEAD の版でも同じで、`--severity=warning`(repo の verify と同じ)では rc=0。
    - self-check は 5 項目とも clean。
    - S3 までの一時の挙動: cross-review の上限は skill の本文どおり `${RALPH_STANDARD_MAX_PIPELINE_CYCLES:-2}` を読み、reviewer の呼び出しも skill の fallback を使う。
    - 注意: `zsh -c` は `-f` がないとユーザーの zshenv を読み、PATH の古い ralph を拾う。
  - S3 2ee62950(impl-276b1-s3)
    - 変えた skill: plan・cross-review・implement の元 2 つを Edit で直し、sync-skills.sh で写し 2 つを作り直した(15 ファイル)。
    - reviewer の呼び出し
      - `/plan` 11.c と `/cross-review` 手順 4 は `. ./scripts/ralph-config.sh; model=$(ralph_setting reviewer.<…>); effort=$(ralph_setting reviewer.<…>)` で読む。
      - codex には `-m "$model" -c "model_reasoning_effort=$effort"` を渡す(`-c` は exec の前)。
      - claude には `--model "$model" --effort "$effort"` を `--restricted` の前に置く。cross-review の claude の道(`--permission-mode auto`)にも `--effort` を足した。
      - /plan 11.c に値の出どころと終了コードの扱いを書いた(stderr の 1 行はユーザーに伝える)。cross-review はそこを参照する。
    - 上限の規則と上げ方
      - cross-review 1.d は、persisted-identity mode でだけ、設定の値と cycle-count.json の正の整数(JSON の数)の `cap` の大きいほうを使う。fallback mode は ∞ のまま。
      - Cap-reached Option 1 は、skill が `f=.harness/state/standard-pipeline/cycle-count.json; jq -c '.cap = .cycle + 1' "$f" > "$f.tmp" && mv "$f.tmp" "$f" || rm -f "$f.tmp"` で書く(`.cycle` から計算し、失敗なら元が残る)。
      - implement 3.c は `cap` も保つ。13.e の上限の文も直した。
    - テスト
      - `test-codex-exec-invocation.sh` は skill が `ralph_setting` を使い `${RALPH_…:-…}` がないことを照合する(S2 の一時の照合は消した)。
      - `defaults_sync_test.go` は 4 つの写しに env の読み出しがないこと、helper のキーの使い方、claude の `--effort`(行数も)、上限の規則の文と jq のコマンド、implement の cap の文を照合する。
      - `test-insights-append.sh` に `cap` のある cycle-count.json の行(8a2)を足した。
    - 検証
      - handoff の 7 つのコマンドはすべて rc=0(load 6〜8)。codex-exec 160/160、insights-append 52、ralph-config 370/370、config の照合 73 本。
      - 変異 4 つ(env の fallback を戻す、`--effort` を消す、cap の文を消す、jq を `.cap = 3` に)は、すべてテストが落とした。
      - cap のコマンドを sh・bash・zsh -f で実地に打ち、`{"cycle": 2}` が `"cap":3` になり、壊れた JSON では元のファイルが残った。
    - self-check は 5 項目とも clean。consult(consult-plan-standardcfg276b)はコミットのあとで 3 点とも「変更なし」。orchestrator の確かめ: 4 つの写しに `${RALPH_…` の読み出しがないことを grep の否定で見た。
    - 範囲外: `scripts/ralph-config.sh:42-43`(と写し)の注釈が「tests/test-codex-exec-invocation.sh calls this function」と書くが、S3 でもう呼ばない。S4 で直す。rules・definition-of-done・codex-setup の export の文は S6。
  - S4 54119962(impl-276b1-s4)
    - `StartFeatureParams.Reviewer`(`config.ReviewerConfig`)を足した。
      - `featureLeaderTask` は `- 台帳:` の直後に 2 行を書く。
        - `- reviewer の codex: <model> <effort>(… -m と model_reasoning_effort に使う)`
        - `- reviewer の claude: <model> <effort>(… --model と --effort に使う)`
      - 頭の文字列は定数(`featureReviewerCodexLead`・`featureReviewerClaudeLead`)。
      - 空のフィールドは `featureReviewer` が `config.Default().Reviewer` で埋める。
      - shellQuote は使わない。雛形が値を二重引用符で囲むので、単一引用符が値の一部になってしまう。Load の検証を通る文字は二重引用符の中で展開されない。
    - cli の受け渡し
      - `resolveOrgConfig` の読み込みを `resolveConfig(configPath) (config.Config, error)` に分けた。
      - `runOrgStartPlan` がそれを呼び、`Reviewer: cfg.Reviewer` を渡す(`--config` か `./ralph.toml` に `ralph.local.toml` を重ねたもの)。
      - 設定は runtime と合わせて 2 回読む。runtime は `[org]` しか持たないため。
      - 読む元は、`[org.approval]`(主 worktree の `ralph.toml`)と違う。self-review で見てもらう。
    - leader.md
      - codex・claude の起動から `. ./scripts/ralph-config.sh;` を外し、TASK の値の置き場所を使う。claude には `--effort` を渡す。
      - `ralph-config.sh` の存在の確かめを外し、`xreview-helpers.sh` の確かめは残した。
      - TASK に行がなければ incomplete と記録する文を足した。
    - テスト
      - 雛形の照合: 4 つの変数名と `ralph-config.sh`・`ralph config get reviewer` がないこと、`${RALPH_` の当たりがちょうど `${RALPH_PRIMARY_CLI` の 1 件であること。
      - コマンドを動かすテストは、TASK の 2 行から値を取り出し、bash・zsh で動かす。撤去した変数に別の値を入れても効かないことも見る。
      - feature の TASK の 2 行と空の扱い。cli で ralph.toml と ralph.local.toml の値が TASK に出ること。
    - `scripts/ralph-config.sh`(と写し)の注釈のテストの名指しを直した。
    - 検証
      - org `-run 'Leader|Prompt|Feature'` 211 本(215 秒)、cli `-run 'OrgStart|Feature'` 50 本、check-sync rc=0、test-ralph-config 370/370、golangci-lint 0 件。
      - 4 つの変数に絞った grep の否定は rc=0。handoff の広い grep(`${RALPH_`)は、残す `RALPH_PRIMARY_CLI` の 517 行の 1 件だけに当たる。handoff の grep が広すぎたので、orchestrator が承認して絞った。
      - 変異 2 つは、どちらもテストが落とした。
    - self-check: unnecessary changes(意味のない loop を消した)と typos(gofmt が見つけた空白、テストのコメント)が fixed、ほかは clean。
    - コミット本文の「reads RALPH_* variables」は不正確(正しくは reviewer の変数を読まない)。amend しないので残した。
    - template の注釈と S2〜S4 の実装の突き合わせ(consult の注意 1): 読めないときの扱い、許す文字、leader が start のときの値を使うこと、はどれも合っている。`max_pipeline_cycles` の注釈に cycle-count.json の `cap` の 1 文がないので、S6 で足す。
    - 範囲外(S6 で扱う)
      - model-routing.md の 78・86・90・148・152 行、codex-setup.md:105 の ralph-config.sh の古い説明。
      - org skill の 697 行あたりのタスクの行の説明。
      - `ralph org start --help` の reviewer の行。
  - S5 10159037(impl-276b1-s5)
    - `retiredEnvs` に 4 行を足した。helper `retiredSettingEnv(name, key, more)` で作り、effect は空、値の形の説明は付けない(Load が合わない値のキーを名指すため)。
    - 上限の行の文: `set to "3", but ralph no longer reads it; write the value in [standard].max_pipeline_cycles of ralph.toml, or of ralph.local.toml (…) for this machine only, or, to raise the cap of one task only, have /cross-review write cap in that task's .harness/state/standard-pipeline/cycle-count.json when it reaches the cap, and unset RALPH_STANDARD_MAX_PIPELINE_CYCLES`。
    - ほかの 3 行は `[reviewer].claude_model`・`codex_model`・`codex_effort` を名指す。
    - 表の doc コメントの出典を 3 つに直し、「~/ の形」の文は org の 2 行だけに掛かる書き方にした。cli の TestMain で 4 つを unset した(org は読まないので足していない)。
    - テストは新しい `doctor_retired_standard_env_test.go`。
    - 検証: cli `-run 'Retired|Doctor'` 40 本 rc=0、golangci-lint 0 件。表から 1 行を消す変異は新しい 3 本が落とした。
    - self-check: typos を 1 つ fixed(TestMain のコメント)、ほかは clean。consult は「変更なし」(コミットのあと)。
  - S6 3cdd5a2c + c87f92e3 + d1ac47d1(impl-276b1-s6)
    - 3cdd5a2c(17 ファイル)
      - model-routing.md は KNOWN_DIFF なので、root と template を別々に直した。template には Go のパスを書かない。
        - sync note: `[reviewer]` の 4 キー、helper、leader の TASK の 2 行、env は効かない。
        - full ID の pin: `[reviewer].claude_model` で、許す文字を添えた。
        - codex の項: doctor が見るのは `[org].model_pool` の slug で、`codex_model` は見ないとコードで確かめた。
        - effort: subagent は既定 high、reviewer は `[reviewer]` で既定 xhigh。
        - 「Where the values live」: `ralph.toml` の項を足し、helper は内部用とした。テストの説明は今の照合に合わせた。
      - post-implementation-pipeline.md と subagent-policy.md(と写し): 上限は設定と `cap` の大きいほう、上げる手順は skill が `cap` を書く。
      - definition-of-done.md と codex-setup.md(と写し)、template の ralph.toml の `cap` の 1 文。
      - org の skill(4 つの写し)にタスクの reviewer の 2 行を足した。
      - repo-map.md の `ralph-config.sh` の説明を直した。
      - evidence `docs/evidence/claude-effort-probe-2026-10-30.md`。
      - `ralph org start --help` はタスクの行を列挙していないので触らない。
    - c87f92e3: evidence のコマンドを、orchestrator が渡した正確な 3 行(2 つの質問の文)に直した。
    - d1ac47d1: `config.go` の `MaxPipelineCycles` の doc コメントに `cap` の 1 文を足した。計画の S6 にあり、handoff に入れ忘れていた分。
    - 検証
      - check-sync rc=0(KNOWN_DIFF 6)、check-skill-sync rc=0、test-agent-models 47/47、test-self-review-scope 206/206、test-slice-verification 162/162。
      - cli `-run 'Init|OrgStart|Help'`・org `-run 'Skill|Feature'`・insights・config は ok。
      - 4 つの変数名の grep は 6 行に当たり、どれも「もう読まない、doctor が warn」の説明。
    - self-check: 作業中にできた `.new` のファイルを消した、テストの説明の言い回しを直した(どちらも fixed)、ほかは clean。
    - consult
      - S6 は「変更なし」。
      - 「codex-setup.md:94 の `$cross-review` を `/cross-review` に」の指摘は採らなかった。この recipe は Codex の `$skill-name` の形でそろっている。
      - 「`ralph upgrade` が helper を入れ替える」は、`ownerForScaffoldPath`(`internal/cli/init.go:469`)が `scripts/` を core に分けるので正しいと orchestrator が確かめた。
    - /sync-docs に回すもの
      - `docs/tech-debt/README.md:161` の行 (b) に「step 9 は `cap` を書くが `cycle` は増やさない」の 1 文。
      - `ralph org start --help` が `[reviewer]` の読む元を書いていない。
      - `config.go:875`・`:927` と `config_test.go:419` の古い `PipelineConfig` の参照(前から)。
      - B2 の個人用の値のための README の `ralph.local.toml` の節(B2 で)。
  - S1〜S6 の実装はそろった。次は /self-review(cycle 1)。
  - /self-review(review-276b1、3c165805): CRITICAL・HIGH・MEDIUM 0、LOW 5、`Merge: yes`。LOW の扱い:
    - L1(doctor の「no longer reads」が upgrade 前のプロジェクトでは事実と違う)、L2(TASK の値の検証が config にしかない)、L4(有効な上限を読む手順が文だけ)、L5(leader に「`:` のあとの 2 語」と書いてあり、`:` を含むモデル名で読み違える)は、安く直せて実害があるので S7 で直した。
    - L3(`ralph org start` で設定を 2 回読む)は直さない。start 1 回につき git が 1 回増えるだけで、ほかの org のコマンドと同じ読み方。
  - S7 759550eb(impl-276b1-s7)
    - L5: leader の文を「行の頭の `- reviewer の codex:`(claude)のあと括弧の前の 2 語」にし、モデルは `:` を含みうるので頭の文字列のあとから読む、と足した。テストの TASK と `advisoryReviewer` に `:` 入りのモデル名を使う。
    - L4
      - cross-review 1.d に `jq -r '.cap | select(type == "number" and . >= 1 and . == floor)' "$f"` を足した(空なら設定の値だけ)。
      - 手順 9 に、表示に新しい `cap` がなければ回し直さずに止まる規則を足した。
      - `defaults_sync_test.go` で固定した。
    - L2
      - `config.ValidateReviewer` を export した。
      - `StartFeature` は `checkSpawnInput` の直後、watch gate より前に、埋めた値を検証して `refuse` する。文は `org: the leader's task copies the [reviewer] values into its shell commands: …`。
      - テストで 5 通りの値が spawn の前に断られることを見る。
    - L1: 4 行の `effect` に「`ralph upgrade` がプロジェクトの skill と `scripts/ralph-config.sh` を入れ替えるまでは、それらが読む」の 1 句を足した。
    - 検証
      - check-skill-sync・check-sync PASS、config ok、org `-run 'Leader|Prompt|Feature'` ok(247 秒)、cli `-run 'Retired|Doctor|OrgStart|Feature'` ok(119 秒)、golangci-lint 0 件。
      - 変異 2 つは、どちらもテストが落とした。
      - jq の 1 行は jq 1.7.1 で、cap なし → 空、3 → 3、"3" → 空、2.5 → 空。
    - self-check は 5 項目とも clean。consult は 4 点とも「変更なし」(コミットのあと)。
    - 範囲外(直さない): rules の文書(model-routing の「Where the values live」、post-implementation-pipeline)の「ralph no longer reads」。rules は skill と同じく `ralph upgrade` で一緒に入れ替わるので、文書と skill の版はずれない。ずれうるのは binary の doctor で、L1 はそれを直した。
  - S7 の見直し(review-276b1、f1685e87): L1・L2・L4・L5 は直った。新しい所見はなく、`Merge: yes`。
  - /verify(verify-276b1、6848f496): Verdict pass。
    - probe(HEAD の binary と一時の repo)で AC1〜AC7 を満たした。
      - AC1: 既定値、上書き、検証の exit 3、完全な ID が通ること。
      - AC2・AC3: env が効かない。helper は 9 通りを sh・bash・dash・zsh で確かめた。
      - AC5: jq を実地に打った。
      - AC6: doctor の warn。
    - AC4 の leader の側は、コードとテストの存在まで。AC8 は静的の部分が rc 0。
    - V1(LOW): `ralph config get --help` が exit 2・3 で「stop and report」と案内し、`ralph_setting` の扱いと食い違う。
    - I3: release 済みの 6.0.0 は、新しいキーを書いた ralph.toml を warn なしで読む(Rollout の「未確認」を確かめた)。
  - S8 745a5407(impl-276b1-s8): help の文。
    - `config get --help` に、switch は止まるが標準フローの値は `ralph_setting` が 1 行知らせて既定値で続ける、と足した(V1)。
    - `org start --help` に `[reviewer]` の読む元と TASK の 2 行を足した。
    - テストを足し、HEAD の help に戻すと落ちることを overlay で確かめた。cli `-run 'Config|OrgStart|Help'` ok、golangci-lint 0 件。self-check は 5 項目とも clean。
  - S9 e430ac4c(impl-276b1-s9): `ralph org start --plan` の `[reviewer]` の読む元を、承認方式と上限の読む元にそろえた。
    - 直す前は、`[reviewer]` だけが cwd の `./ralph.toml` から読まれた。そのため、サブディレクトリや linked worktree から打つと repo の値を黙って無視した(S8 の報告で見つかった)。
    - 変えたこと
      - `loadMainWorktreeConfig`(全体を返す)を足し、`loadMainWorktreeOrgConfig` はその包みにした。
      - `resolveOrgStartReviewer` は、主 worktree の設定が読めればその Reviewer を使い、そうでなければ `resolveConfig(--config)` の Reviewer を使う。
      - help の文は「the same ralph.toml as max_orgs and max_total_seats」に直した。
    - テスト `TestOrgStartPlan_ReviewerFromMainWorktree` の 6 通り: サブディレクトリ、linked、主に `[reviewer]` なし、`ralph.local.toml`、`--config`、`--config` と `ralph.local.toml`。
    - 2 回読みの回数は変わらない(二重に読むファイルが cwd から主に移っただけ。L3 の判断どおり)。
    - 検証: cli `-run 'OrgStart|Feature|MainWorktree|Limits|Approval|Help'` 98 本 ok、golangci-lint 0 件。変異(`resolveConfig` に戻す)は主 worktree の 4 件が落とした。
    - self-check: doc の 1 語を fixed、ほかは clean。consult は「変更なし」(主の ralph.toml が読めないときは、上限の読み込みが先にエラーで止める)。
  - /sync-docs に回すもの(S8・S9 まで)
    - `docs/tech-debt/README.md:161` の (b)
    - `README.md:135` の `config get` の行(V1 と同じずれ)
    - org skill の 712〜713 行あたり(4 つの写し)の reviewer の読む元(S9 で承認方式と同じ規則になった)
    - 古い `PipelineConfig` の参照(`config.go:883`・`:935`、`config_test.go:419`)
    - verify の I3 を Rollout の記録か PR 本文に書く
  - /sync-docs(cycle 1、doc-maintainer): `docs/reports/sync-docs-2026-10-30-standard-flow-config.md`。上の 5 項目はすべて直した(README の `config get` の行、org skill の 4 つの写し、`--config` の usage とテストの pin、`PipelineConfig` のコメント 3 か所、tech-debt の 161 行)。tech-debt は、Trigger に当たった 7 行に持ち越しの注、新しい 3 行(800 行を超えた 2 ファイル、`ralph org start --plan` の 2 回読み、test gaps 3 つ)。README の `ralph.local.toml` の節に、標準フローの値と撤去した 4 つの変数の段落を足した。コードの変更はコメントと help の文字列と pin の 1 行だけ。plan 本文は触れておらず、digest は d8e03ef62de9 のまま。
- [ ] Review artifact created
- [ ] Verification artifact created
- [ ] Test artifact created
- [ ] PR created
