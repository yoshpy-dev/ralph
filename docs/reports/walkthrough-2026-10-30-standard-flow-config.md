# Walkthrough: standard-flow-config(標準フローの上限と reviewer のモデル・effort を `ralph.toml` で決め、4 つの環境変数を読まなくする)

- Date: 2026-10-30
- Plan: docs/plans/archive/2026-10-30-standard-flow-config.md
- Issue: #276(`Refs`。#276 を 3 本に分けた 3 本目 B の前半 B1。1 本目 A1 は #287、2 本目 A2 は #290 で merge 済み。B2 がスクリプトの個人用の値を受け持ち `Closes #276`)
- PR: #291(https://github.com/yoshpy-dev/ralph/pull/291)
- Branch: feat/standard-flow-config(この walkthrough の前の HEAD は dee8a9e6。31 commits で、merge commit はない)
- Base: 0351800d(origin/main、「Merge pull request #290」= org-state-dir-local)。分岐元も同じ
- 読む範囲: `git diff origin/main...HEAD` は 62 files、+4006/-787。内訳は Go の本番コード 5 files(+325/-74)、leader の雛形 1 file(+19/-6)、Go のテスト 11 files(+1345/-193)、helper 2 files(+170/-132。root と template の同じ写し)、shell のテスト 3 files(+295/-204)、skill の写し 16 files(+192/-100。4 つの skill × 4 つの写し)、rules・文書・設定 14 files(+197/-78)、記録(計画・report・evidence・insight event)10 files(+1463)。動きを決めるのは `internal/config/config.go`(+123/-25)、`internal/org/feature.go`(+77/-15)、`internal/cli/org.go`(+76/-24)、`internal/cli/doctor.go`(+40/-9)、`scripts/ralph-config.sh`(+85/-66)、`leader.md`(+19/-6)、skill 4 本。テストの差分の大半は、`defaults_sync_test.go`(+397/-122。shell の変数との照合を、helper の表・template の注釈・skill の本文との照合に書き換えた)、`tests/test-ralph-config.sh`(+232/-129。偽の ralph で書き直した)、`internal/org/feature_test.go`(+194/-17)、`internal/config/config_test.go`(+193/-7)、`internal/cli/org_feature_test.go`(+172)
- 後工程: パイプラインは 1 回(`cycle-count.json` は 1)。plan advisory(codex)は MEDIUM 2 件で、D5 と D8 に入れた。self-review は LOW 5 で Merge 可(L1・L2・L4・L5 は S7 で直し、L3 は tech-debt に入れた。S7 と S8・S9 の見直しに新しい所見はない)。verify は pass(S8・S9 の追補を含む。LOW の V1 は S8 で、V2 は sync-docs で直した)。test は pass。sync-docs は 1 回。cross-review(codex)は所見 0 件。記録だけの修正の例外は使っていない。CI の verify・check・GitGuardian は HEAD dee8a9e6 で pass

```
何を変えるか → 設定のキーと検証(S1) → helper(S2) → skill(S3) → leader の TASK の 2 行と読む元(S4、S9) → doctor(S5) → 文書と help(S6、S8、sync-docs) → レビューで足した直し(S7) → スライスと AC → 確かめたことと確かめていないこと
```

## 1. この PR が何か

- これまで:
  - 標準フローの値は、環境変数 `RALPH_STANDARD_MAX_PIPELINE_CYCLES`、`RALPH_CLAUDE_REVIEWER_MODEL`、`RALPH_CODEX_REVIEWER_MODEL`、`RALPH_CODEX_REASONING_EFFORT` で決めた。既定値は `scripts/ralph-config.sh` が持ち、skill は `${VAR:-既定値}` で読んだ。
  - claude の reviewer には effort を渡さなかった。上限に当たったときの「上げて回し直す」は、`export RALPH_STANDARD_MAX_PIPELINE_CYCLES=…` を人に頼む手順だった。
- この PR から:
  - 値は `ralph.toml`(自分の機械だけなら `ralph.local.toml`)の 5 つのキーで決める。環境変数は読まない。

    | キー | 既定 |
    |---|---|
    | `[standard].max_pipeline_cycles` | 2 |
    | `[reviewer].claude_model` | fable(これまでは opus) |
    | `[reviewer].claude_effort` | xhigh(新設。claude に `--effort` を渡す) |
    | `[reviewer].codex_model` | gpt-6-astra |
    | `[reviewer].codex_effort` | xhigh |

  - skill は内部用の helper `ralph_setting <key>` で読む。helper は `ralph config get` を打ち、読めないときは自分の表の既定値で続ける。
  - 上限に当たって 1 つの作業だけ上げるときは、skill がその作業の `cycle-count.json` に `cap` を書く。有効な上限は設定の値と `cap` の大きいほう。
  - feature org の leader の計画助言は、`ralph org start --plan` が TASK に書いた 2 行の値を使う。leader は設定を読まない。
  - `ralph doctor` は、4 つの変数がまだ設定されていると warn を出す。
- 計画の判断(D1〜D8)のうち、読むときの前提になるもの:
  - D1 B を B1(この PR)と B2(スクリプトの個人用の値)に分ける。前例は A1・A2。
  - D2 値の読み方は helper に集める。黙って既定値に戻るのは「ralph が PATH にない(127)」と「`config get` のない古い ralph」だけで、それ以外(exit 2・3 など)は stderr に 1 行知らせてから既定値で続ける。
  - D3 `[reviewer]` を共有の表にする(`/plan`・`/cross-review`・leader の 3 か所が同じ値を使う)。template の `ralph.toml` には注釈で書く。有効な行で書くと、seed の `ralph.toml` は `ralph upgrade` が上書きしないので、将来 `Default()` を変えても配布先に古い値が残る。
  - D4 claude の reviewer に `--effort` を渡す。`claude_effort` は claude が受け付ける 5 つの値だけ。モデル名と `codex_effort` は `[A-Za-z0-9._:@/+-]+` だけで、skill と leader のコマンドに 1 語で写るので引用符や `$` を断る。
  - D5 leader の値は `ralph org start` が TASK に書く。leader が打つ `ralph` は PATH で決まり、雛形を埋め込んだ binary と同じ版とは限らない(計画助言の指摘 1)。
  - D6 `COVERAGE_THRESHOLD` と `RALPH_TERRAFORM_INIT_VALIDATE` は環境変数のまま残す。`RALPH_PRIMARY_CLI` も、人が好みで決める値ではないので残す。
  - D7 撤去した 4 つの変数は、値として読まず、doctor が warn を出すだけにする。
  - D8 1 つの作業だけ上限を上げる値は `ralph.local.toml` に書かず、その作業の `cycle-count.json` に書く。`ralph.local.toml` は主 worktree の root の 1 つのファイルなので、ほかの作業にも効き、元に戻らない(計画助言の指摘 2)。
- issue の文との違い: D6 の 3 つの例外は、issue の「残した環境変数はすべて内部の受け渡し」の字句には合わない。PR 本文の Out of scope に理由を書き、読み替えるかどうかの判断を頼んである。全変数を分けた一覧(B2 の分も含む)は計画の冒頭にある。
- release のノート: PR 本文の「release のノートに書くこと」。要点は、4 つの変数を読まなくなること、claude の reviewer の既定が fable と xhigh になること、上限は `/cross-review` が `cap` を書くこと。skill と helper を使うには新しい binary が要る。
- 戻し方: この PR を revert する。設定ファイルの形式はキーが増えるだけで、release 済みの 6.0.0 は新しいキーを書いた `ralph.toml` を warn なしで読む(verify の I3)。

## 2. 設定のキーと検証(S1、f66c4a4a・4f3cc3d4)

- 型と既定:
  - `StandardConfig.MaxPipelineCycles`(`internal/config/config.go:53`、`toml:"max_pipeline_cycles"`)。
  - `ReviewerConfig`(`:70`)と `Config.Reviewer`(`:19`)。
  - 既定は `Default()` の `:669-675`。
- 検証:
  - 上限が 1 未満なら、`validateLoaded`(`:863`)の末尾が `[standard].max_pipeline_cycles must be >= 1` のエラーにする(`:1034-1036`)。
  - `ValidateReviewer`(`:1045`)が、claude_model、claude_effort、codex_model、codex_effort の順に確かめる。`claude_effort` は `reviewerClaudeEfforts`(`:746`、low・medium・high・xhigh・max)のどれか。ほかの 3 つは `reviewerWordPattern`(`:752`)に合う 1 語で、`checkReviewerWord`(`:1060`)が空もエラーにする。
  - エラーはキーと値を名指す。`agmsg_home` のように空を既定値で埋めることはしない。
  - `ValidateReviewer` を export したのは、`StartFeature` が TASK に写す前に同じ検査を打つため(S7 の L2、5 節)。
- 許す文字の広げ方(4f3cc3d4): 初めは `[A-Za-z0-9._-]` だった。完全なモデル ID(Bedrock の `…-v1:0`、ARN)を断らないよう、`:`・`@`・`/`・`+` を足した。この 4 つは shell が裸でも二重引用符の中でも展開しない。計画の Scope・D4・AC1 を直し、承認の digest を af46dcbdd0f8 から d8e03ef62de9 に取り直した(自動承認、ユーザーの指示)。
- `ralph config get`: reflection で引くので配線は足していない。テストで 5 つのキーの既定値、上書き、exit 3 を固定した(`internal/cli/config_test.go` の `TestConfigGet_PrintsThePipelineCapAndReviewerValues`、`TestConfigGet_InvalidPipelineCapOrReviewerValueExits3`)。
- template(`templates/base/ralph.toml:34-72`): `[standard]` に `# max_pipeline_cycles = 2` と `cap` の説明、`[reviewer]` は有効な空の表(`:68`)にして 4 つのキーを注釈にした。空の表は今の Load も旧 binary も無視する。
- 照合(`internal/config/defaults_sync_test.go`):
  - `commentedTemplateDrift`(`:236`)は、注釈にしたキーが template に set されていない、注釈がちょうど 1 回、値が `Default()` と同じ、を見る。`TestCommentedTemplateDrift_EachKeyIsComparedOnBothSides`(`:778`)と `TestTemplateDrift_EveryStandardAndReviewerKeyIsChecked`(`:818`)が、各キーが両側で比べられていることを固定する。
  - `[org.review].max_cross_review_cycles` の既定と `[standard].max_pipeline_cycles` の既定の一致は、Go の中で見る(`TestDefaultsLockStep` の `org_review_cycles`、`:363`)。これまでは shell の変数と照合していた。
- テスト(`internal/config/config_test.go`): `TestDefault_PipelineCyclesAndReviewer`(`:1967`)、`TestLoad_PipelineCyclesAndReviewer`(`:1982`)、`TestLoad_StandardMaxPipelineCyclesBoundary`(`:2038`)、`TestLoad_ReviewerValidation`(`:2078`)。個人用のファイルが上書きすることは `local_test.go` の `TestLoad_LocalFileOverridesPipelineCyclesAndReviewer`。
- 検証: config と cli の `Config` のテストが rc 0。変異 5 つを、テストが落とした。

## 3. helper(S2、27d3cbfc)

- `scripts/ralph-config.sh`(と `templates/base/scripts/` の同じ写し)は、`ralph_setting <key>` と `ralph_setting_default <key>` の 2 つの関数だけを定義する。source しても変数を設定せず、環境変数は読まない。
- 既定値の表は `ralph_setting_default`(`:44`)の `<key>) printf '%s\n' '<value>' ;;` の 5 行だけ。`defaults_sync_test.go` の `parseShellSettingDefaults`(`:82`)がこの形を正規表現で読んで `Default()` と照合し、`tests/test-ralph-config.sh` がこの関数を呼ぶ。行の形を変えるときは両方のテストを変える。
- `ralph_setting`(`:58`)の分け方(`command ralph config get <key>` の終了コードで決める):
  - 0 で値があれば、その値を出す。0 で空なら失敗として扱う。
  - 127(ralph が PATH にない)と、exit 1 で stderr に `unknown command "config"`(`:73`、`config get` より古い ralph)は、黙って既定値を出す。
  - それ以外(2 = 知らないキー、3 = 読めない・正しくない設定、ほかの 1)は、stderr に 1 行(キー、終了コード、ralph が書いた stderr の 1 行目、使う既定値)を出して既定値を出す(`:89-91`)。
  - `mktemp -d` が失敗したときも、1 行出して既定値で続ける(`:64`)。
  - どの場合も戻り値は 0 なので、`set -e` の下でも続く。表にないキーだけは、何も出さず stderr に 1 行書いて 2 を返す。
- 変数は `_rs_` で始まる(POSIX sh に local がないため)。一時ディレクトリは必ず消す。
- `tests/test-ralph-config.sh`(370 項目)は、偽の `ralph` を PATH の先頭に置き、上の分岐を 9 通り、sh・bash・dash・zsh の 4 つで打つ。4 つの変数に別の値を入れても出力が変わらないこと、一時ファイルが残らないことも見る。
- 検証: 370/370。S2 の変異 5 つと表の書き換えをテストが落とした。test の変異(127 の分岐を外す、古い ralph の判定を広げる)も落ちた。

## 4. skill(S3、2ee62950)

- `/plan` の手順 11.c(`.claude/skills/plan/SKILL.md:84-98`):
  - codex の呼び出しは `. ./scripts/ralph-config.sh; model=$(ralph_setting reviewer.codex_model); effort=$(ralph_setting reviewer.codex_effort)` で読み、`command codex -m "$model" -c "model_reasoning_effort=$effort" exec …` に渡す。
  - claude の呼び出しは `--model "$model" --effort "$effort"` を `--restricted` の前に置く。
  - 値の出どころと終了コードの扱いは 11.c に書いた。stderr の 1 行は、ユーザーに伝える。
- `/cross-review`(`.claude/skills/cross-review/SKILL.md`):
  - 手順 4(`:70`・`:78`)の codex と claude は、11.c と同じ読み方。claude の道(`--permission-mode auto`)にも `--effort` を足した。codex の `-c` は `exec review` の前に置く(後に置くと codex が前の `-c` を捨てるため)。
  - 手順 1.d(`:44-47`)が上限を決める。persisted-identity mode でだけ、設定の `standard.max_pipeline_cycles` と、`cycle-count.json` の正の整数(JSON の数)の `cap` の大きいほうを使う。fallback mode は今までどおり上限なし。
  - 手順 8(`:142`)は `CAP_REACHED = (cycle >= cap)`。
  - 手順 9 の Cap-reached Option 1(`:184-189`)は、skill 自身が `f=.harness/state/standard-pipeline/cycle-count.json; jq -c '.cap = .cycle + 1' "$f" > "$f.tmp" && mv "$f.tmp" "$f" || rm -f "$f.tmp"` を打つ。`cycle` は増やさない。`/pr` の成功でファイルごと消えるので、上げた上限は残らない。環境変数を人に頼む手順はなくなった。
- `/implement`(`:29-30`、`:59`): 手順 3.c は `cycle-count.json` を保つときに `cap` も保つ。別の plan のときの reset は `cap` を引き継がない。13.e の上限の文も直した。
- 4 つの写し(`.claude/skills`、`.agents/skills`、`templates/base/` の 2 つ)は、元 2 つを直して `sync-skills.sh` で作り直した。
- テスト:
  - `tests/test-codex-exec-invocation.sh`(160 項目)は、codex の呼び出しが `-m "$model" -c "model_reasoning_effort=$effort"` の形であること、`command codex` の前に helper を source していること、4 つの写しに `${RALPH_…}` の読み出しがないことを固定する。
  - `defaults_sync_test.go` の `checkSkillSettings`(`:489`)は、4 つの写しの helper のキーの使い方、claude の `--effort`(行数も)、上限の規則の文と jq のコマンド、implement の `cap` の文を照合する。
  - `tests/test-insights-append.sh` の 8a2(`:348`)は、`cap` のある `cycle-count.json` でも `--cycle auto` が `.cycle` を読むことを見る。
- 検証: cap のコマンドを sh・bash・zsh で実地に打ち、`{"cycle": 2}` が `"cap":3` になり、壊れた JSON では元のファイルが残ることを見た。変異 4 つ(env の fallback を戻す、`--effort` を消す、cap の文を消す、jq を固定値にする)をテストが落とした。

## 5. leader の TASK の 2 行と読む元(S4 54119962、S9 e430ac4c)

- TASK に書く(`internal/org/feature.go`):
  - `StartFeatureParams.Reviewer`(`:107`)が `[reviewer]` の表を受け取る。
  - `featureReviewer`(`:387`)は、空のフィールドを `Default()` の値で埋める(`Load` を通らずに作った呼び出しのため)。
  - `featureLeaderTask`(`:469`)は、`- 台帳:` の行の次に 2 行を書く。頭の文字列は定数(`:429-430`)。
    - `- reviewer の codex: <model> <effort>(…-m と model_reasoning_effort に使う)`
    - `- reviewer の claude: <model> <effort>(…--model と --effort に使う)`
  - 値は `shellQuote` を通さない。雛形が値を二重引用符で囲むので、単一引用符が値の一部になってしまう。`ValidateReviewer` を通る文字は二重引用符の中で展開されない。
  - `StartFeature` は、`checkSpawnInput` の直後、watch gate の前(`:227`)に、埋めた値を `config.ValidateReviewer` にかけて `refuse` する。worktree も spawn もまだ作っていない。
- 雛形(`internal/org/prompts/leader.md`「計画助言」`:496-`):
  - モデルと effort は、TASK の 2 行の頭の文字列のあと、括弧の前の 2 語(`:527-540`)。モデルが `:` を含みうる(`us.anthropic.claude-x-v1:0`)ので、行の頭のあとから読み、ほかの `:` では区切らない(S7 の L5)。
  - codex の呼び出し(`:578`)は `-m "<reviewer の codex のモデル>" -c "model_reasoning_effort=<…の effort>"`、claude の呼び出し(`:592`)は `--model "…" --effort "…"`。
  - `scripts/ralph-config.sh` を source せず、`ralph config get reviewer.*` も打たない。`xreview-helpers.sh` があるかの確かめ(`:517`)は残した。
  - TASK に 2 行がなければ(行を書かない古い ralph が起こした org)、相手を呼ばず計画助言を incomplete とする(`:538`)。
- 読む元(`internal/cli/org.go`、S9):
  - `runOrgStartPlan`(`:1165`)が `resolveOrgStartReviewer`(`:665`)を呼び、結果を `Reviewer`(`:1190`)に渡す。
  - `--config` がなく、台帳が主 worktree のものなら、主 worktree の `ralph.toml`(`loadMainWorktreeConfig`、`:337`)から読む。それ以外は `--config` か `./ralph.toml`(`resolveConfig`、`:648`)から読む。どちらも `ralph.local.toml` が上に重なる。
  - これは承認方式 `[org.approval].mode` と上限を読む規則(`withMainWorktreeOrgLimits`)と同じ。S9 の前は `[reviewer]` だけが cwd の `./ralph.toml` から読まれ、サブディレクトリや linked worktree から打つと、repo の値が使われず、エラーも出なかった(S8 の報告で見つかった)。
  - runtime は `[org]` しか持たないので、同じファイルを 2 回読む(self-review の L3、直さないと決めた。tech-debt に入れた)。
  - `ralph org start --help`(`:1004`)と `--config` の usage(`:60`)に、この読み方を書いた。
- テスト:
  - org: `TestFeatureLeaderTask_Reviewer`、`TestStartFeature_LeaderTaskCarriesTheReviewer`、`TestStartFeature_RefusesReviewerValuesThatSplitTheShell`(spawn の前に断られる)。
  - cli: `TestOrgStartPlan_LeaderTaskCarriesTheReviewerSettings`、`TestOrgStartPlan_ReviewerFromMainWorktree`(サブディレクトリ、linked worktree、主に `[reviewer]` なし、`ralph.local.toml`、`--config`、`--config` と `ralph.local.toml` の 6 通り)。
  - 雛形: `…FeatureOrgPlanAdvisoryTakesTheReviewerFromTheTask`(`prompts_leader_plan_test.go:716`)は、雛形に 4 つの変数名と `ralph-config.sh`、`ralph config get reviewer` がないことを見る。`…CodexCommandRuns`(`:1112`)と `…ClaudeCommandRuns`(`:1239`)は、TASK の 2 行から値を取り出して雛形のコマンドを偽の codex・claude で bash・zsh で動かし、撤去した変数に別の値を入れても効かないことを見る。
- 検証: org の `Leader|Prompt|Feature` が 211 本、cli の `OrgStart|Feature` が 50 本で rc 0(S4)。S9 の変異(`resolveConfig` に戻す)は、主 worktree の 4 件を落とした。

## 6. doctor(S5、10159037)

- `retiredEnvs`(`internal/cli/doctor.go:886`)に 4 行を足した。helper `retiredSettingEnv(name, key, more)`(`:914`)で作る。
  - `RALPH_STANDARD_MAX_PIPELINE_CYCLES` は `[standard].max_pipeline_cycles` を名指し、1 つの作業だけ上げるときは `/cross-review` が `cap` を書くことも添える(`:895-897`)。
  - 残る 3 つは `[reviewer].claude_model`、`codex_model`、`codex_effort` を名指す(`:898-900`)。
  - 値の形の説明は付けない。`config.Load` が合わない値のキーを名指すため。
- warn の文は `checkRetiredEnv`(`:930`)が組む。`set to "<値>", but ralph no longer reads it<effect>; <instead>, and unset <変数>`。この 4 つの `effect` は `retiredSettingEffect`(`:909`)で、「`ralph upgrade` がプロジェクトの skill と `scripts/ralph-config.sh` を入れ替えたあとは」と添える。binary は元からこの 4 つを読まず、読んでいたのは skill と helper なので、入れ替えるまでは今までどおり読む(S7 の L1)。
- 空の値と未設定では何も出さない。
- テスト(`internal/cli/doctor_retired_standard_env_test.go`): `TestCheckRetiredEnv_StandardFlowVariables`、`TestCheckRetiredEnv_AllSixInTableOrder`、`TestRunDoctorOpts_RetiredStandardFlowEnv`。`main_test.go` の `TestMain` が 4 つを unset する。表から 1 行を消す変異を、新しい 3 本が落とした。

## 7. 文書と help(S6 3cdd5a2c、S8 745a5407、sync-docs 2db81d65)

- rules(`.claude/rules/ralph/`):
  - `model-routing.md` は root と template が KNOWN_DIFF なので、別々に直した(template には Go のパスを書かない)。sync note(`[reviewer]` の 4 キー、helper、leader の TASK の 2 行、env は効かない)、full ID の pin(`[reviewer].claude_model` で、許す文字を添えた)、codex の項(doctor が見るのは `[org].model_pool` の slug で、`codex_model` は見ない)、effort(subagent は既定の high、reviewer は `[reviewer]` で既定 xhigh)、「Where the values live」。
  - `post-implementation-pipeline.md`(`:116-118`)と `subagent-policy.md`(`:101-104`)は、上限を「設定と `cap` の大きいほう」にし、上げる手順は skill が `cap` を書く、とした。
- `docs/quality/definition-of-done.md`、`docs/recipes/codex-setup.md`(と写し): `export RALPH_STANDARD_MAX_PIPELINE_CYCLES` を名指していた文を直した。`docs/architecture/repo-map.md` は `ralph-config.sh` の説明を `ralph_setting <key>` にした。
- `/org` skill の 4 つの写し: タスクの reviewer の 2 行の段落と、上限を主 worktree から読む規則の段落(sync-docs で、読む元を S9 の規則に合わせた)。
- README: `ralph config get` の行(止まるのは、skill や seat が自分で打つ switch の場合。`standard.max_pipeline_cycles` と `[reviewer]` は `ralph_setting` が 1 行知らせて既定値で続ける)と、`ralph.local.toml` の節の末尾の段落(5 つのキーの置き場所、撤去した 4 つの変数、doctor の warn、`cap`)。段落は A1・A2 の前例に合わせて足した。
- help(Go の文字列):
  - `ralph config get --help`(`internal/cli/config.go`): 終了コードの案内に、`ralph_setting` が止まらない扱いの段落を足した(S8、verify の V1)。
  - `ralph org start --help`(`org.go:1004`): `[reviewer]` を同じ `ralph.toml` から読み、TASK の 2 行に写すこと(S8)。
  - `ralph org` の `--config` の usage(`org.go:60`): `start --plan` が `[org.approval].mode` と `[reviewer]` を読むこと(sync-docs、verify の V2)。`org_director_test.go` に pin を 1 項目足した。
- `docs/evidence/claude-effort-probe-2026-10-30.md`: `claude -p --effort` が効くことの確かめ。claude 2.1.308、`--model haiku`、git の外、`RALPH_WIP_CHECKPOINT=off`。正しくない値(`bogus`)は rc 0 で「Unknown --effort value … using the default effort」の warning が出る。同じ質問で thinking のトークンは `low` 0、`max` 1066。
- `config.go` と `config_test.go` のコメント 3 か所は、もうない `PipelineConfig` を指していたので、文字列の既定を `""` のとき埋める書き方(`agmsg_home`)を名指す文にした。意味は変えていない。
- `docs/tech-debt/README.md`: 161 行の (b) に「`cap` を書くが `cycle` は増やさない」の 1 文。Trigger に当たった 7 行に持ち越しの注。新しい 3 行(800 行を超えた 2 ファイル、`start --plan` の 2 回読み、test gaps 3 つ)。

## 8. レビューで足した直し(S7、S8、S9)

- S7(759550eb、self-review の LOW 4 件):
  - L1 doctor の「no longer reads」が、upgrade 前のプロジェクトでは事実と違う。4 行の `effect` に `retiredSettingEffect` を足した(6 節)。
  - L2 TASK の値の検証が `config.Load` にしかなかった。`ValidateReviewer` を export し、`StartFeature` が spawn の前に断る(2 節、5 節)。
  - L4 有効な上限を読む手順が文だけだった。cross-review 1.d に `jq -r '.cap | select(type == "number" and . >= 1 and . == floor)'` を足し、手順 9 には「表示に新しい `cap` がなければ回し直さずに止まる」規則を足した。`defaults_sync_test.go` で固定。
  - L5 leader に「`:` のあとの 2 語」と書いてあり、`:` を含むモデル名で読み違える。行の頭のあとから読む、に直した(5 節)。
- S8(745a5407): verify の V1。`config get --help` と `org start --help` の文(7 節)。HEAD の help に戻すとテストが落ちることを overlay で確かめた。
- S9(e430ac4c): `[reviewer]` の読む元を、承認方式と上限の読む元にそろえた(5 節)。
- L3(`start --plan` で `ralph.toml` を 2 回読む)は、start 1 回につき git が 1 回増えるだけなので直さないと決め、tech-debt に入れた。

## 9. スライスと受け入れ条件(AC)

| スライス | commit | 当たる AC | 内容 |
|---|---|---|---|
| S1 | f66c4a4a 4f3cc3d4 | AC1 | `[standard].max_pipeline_cycles`、`[reviewer]` の 4 つのキー、検証、template の注釈、照合。許す文字を広げた |
| S2 | 27d3cbfc | AC2 AC3 | `ralph_setting` と既定値の表、`tests/test-ralph-config.sh` の書き直し |
| S3 | 2ee62950 | AC2 AC4 AC5 | `/plan`・`/cross-review`・`/implement` の読み方、`--effort`、上限の規則と `cap` |
| S4 | 54119962 | AC2 AC4 | `StartFeatureParams.Reviewer`、TASK の 2 行、雛形 |
| S5 | 10159037 | AC6 | `retiredEnvs` の 4 行 |
| S6 | 3cdd5a2c c87f92e3 d1ac47d1 | AC4 AC7 | rules、definition-of-done、codex-setup、repo-map、effort の evidence |
| S7 | 759550eb | AC4 AC5 AC6 | self-review の L1・L2・L4・L5 |
| S8 | 745a5407 | AC7 | `config get --help` と `org start --help` の文 |
| S9 | e430ac4c | AC4 | `[reviewer]` の読む元を主 worktree の `ralph.toml` に |
| sync-docs | 2db81d65 ace208e0 | AC7 | README、`/org` skill、`--config` の usage、コメント、tech-debt |

AC1〜AC7 は、verify が HEAD から build した binary と一時の git repo で打った probe で満たした(AC4 の leader の側は、コードとテストの存在まで)。AC8(build、gofmt、vet、golangci-lint、`check-sync`、`check-skill-sync`、`run-verify.sh`)は個別の commit に当たらない。`run-verify.sh` の既定モード(all)は 1 回では打たず、その 2 つの mode である verify の `run-static-verify.sh` と test の `run-test.sh` が、どちらも rc 0 だったことで満たしたと判断した(plan の Progress に書いてある)。

## 10. 確かめたことと確かめていないこと

- 確かめたこと:
  - `./scripts/run-static-verify.sh` rc 0(verify。`scripts/ralph-config.sh` が分類されず、Go 全体が走った。gofmt、go vet、staticcheck、golangci-lint 0 件、shellcheck `--severity=warning`、`check-sync`(DRIFTED 0、KNOWN_DIFF 6)、`check-skill-sync`(14 本)、tech-debt の plan 参照、branch の secret scan)
  - `./scripts/run-test.sh` rc 0(test。1,587 秒。shell 45 本、Go 10 パッケージ。`internal/org` 1,331 本、`internal/cli` 926 本、`internal/config` 100 本)。`tests/test-ralph-config.sh` は sh・bash・`zsh --emulate sh -f` で 370/370。`-count=3` の 2 本は 3 回とも同じ結果。変異 5 つ(helper の 127 の分岐、古い ralph の判定、doctor の表のキー、`featureLeaderTask` の claude の行、`codex_effort` の検証)はすべてテストが落とした
  - カバレッジ(test): `ValidateReviewer` 100%、`featureLeaderTask` 100%、`retiredSettingEnv` 100%、`checkRetiredEnv` 100%、`loadMainWorktreeConfig` 100%。打たれていないのは `resolveOrgStartReviewer` のエラーの return 2 つと `runOrgStartPlan` のその呼び出し(先に `newOrgSpawnRuntimeAt` が同じ設定を読むので届かない、という判断で、テストでは確かめていない)
  - HEAD の binary の probe(verify): ファイルがないとき 5 つの既定値。`ralph.toml` の値と `ralph.local.toml` の上書き。検証のエラーが exit 3 でキーを名指すこと、完全なモデル ID が通ること。4 つの変数に別の値を入れても helper の出力が変わらないこと。cross-review の jq。doctor の 4 回の実行の diff
  - I3(verify): PATH の ralph 6.0.0 の `doctor` を、5 つのキーを有効な行で書いた `ralph.toml` に打つと rc 0、`All checks passed.`、warn も parse error もなし
  - sync-docs 後: `go build ./...`、`go vet`、`gofmt -l`、`go test ./internal/config/`、`go test ./internal/cli/ -run 'Help|Config|OrgStart|Director'`、`go test ./internal/org/ -run Skill`、`check-sync`、`check-skill-sync`、plan の digest が d8e03ef62de9 のまま
  - cross-review(codex)は所見 0 件。CI の verify・check・GitGuardian は HEAD dee8a9e6 で pass
- 確かめていないこと・残っていること:
  - 実物の claude と codex を、helper が読んだ値で起こす通し
  - feature org の leader が、TASK の 2 行の値で codex と claude を起こす通し(herdr・agmsg が要る)
  - 実物の v6.0.0 に skill を回させること。偽の ralph で、exit 1 と `unknown command "config"` の経路は見ている
  - `--effort` の確かめは haiku だけ。既定の fable と、medium・high は確かめていない
  - cross-review の reviewer(codex)は、read-only の sandbox で Go のテストを打てなかった。テストの結果は `/test` による
  - sync-docs の 2 commit(2db81d65、ace208e0)のあと、`run-test.sh` 全体は打っていない。Go の変更はコメント 3 か所、`--config` の usage の 1 文、その pin の 1 項目だけ
  - Known gaps(この PR では直していない): leader が PATH の `ralph` で switch(`org.review.plan_advisory`)と `ralph org` の動詞を打つこと(計画の Non-goals)、B2 の 5 つの変数、`ralph_setting` と review switch で exit 2・3 の扱いが分かれること(README と `config get --help` に書いた)、tech-debt に足した 3 行
- 注意して読んでほしい所:
  - `ralph_setting` の分岐(3 節)。黙るのは 127 と `unknown command "config"` の exit 1 だけで、ほかの失敗は 1 行知らせる。0 で空の値も失敗として扱う
  - cross-review 1.d の `cap` の読み(4 節)。fallback mode では使わないこと、JSON の数で 1 以上の整数だけを採ること。手順 9 の書き込みが `.cycle` から計算し、失敗すると元のファイルが残ること
  - `featureLeaderTask` が値を `shellQuote` に通さない理由(5 節)。二重引用符の中で展開されない文字だけを `ValidateReviewer` が通す、という 2 つの前提が揃って成り立つ
  - 雛形が `:` を含むモデルをどう読むか(5 節)。行の頭の文字列のあと、括弧の前の 2 語
  - `StartFeature` の検証が watch gate の前にあること(何も作らずに断る)
  - template の `[reviewer]` は有効な空の表で、キーは注釈であること(2 節)。配布先の `ralph.toml` は `ralph upgrade` が上書きしないので、既定値を変えたいときは人が書き足す
  - claude の reviewer の既定が opus から fable に変わること。Codex が主の CLI のとき、使うモデルが変わる。`[reviewer].claude_model` で戻せる
