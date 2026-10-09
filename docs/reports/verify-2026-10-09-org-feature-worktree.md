# Verify report: org-feature-worktree

- Date: 2026-10-09
- Plan: docs/plans/active/2026-10-09-org-feature-worktree.md(承認済み、digest 56e435bfa976)
- Verifier: verifier subagent (Claude Opus 5.5)、pipeline cycle 1(`cycle-count.json` は 1、上限 2)
- Scope: 仕様への適合(AC1〜AC14、仕様の FR-4、FR-11 の `/org` skill の部分、受け入れ条件)、静的解析、文書のずれ。対象は `git diff 765da6bd...HEAD`(HEAD a49d38a1、14 コミット、29 ファイル、+6457/-374)。テストは実行していない(`/test` の担当)。テストについては、何を固定しているかを読んで確かめた
- Evidence: `docs/evidence/verify-2026-10-09-org-feature-worktree.log`(`docs/evidence/*.log` は gitignore の対象なので手元にだけ残る)。runner 自身のログは `docs/evidence/verify-2026-10-09-111530.log`

## Spec compliance

計画の承認: `./scripts/plan-visual.sh digest` は HEAD でも、計画を触った 4 つのコミット(61b05d3d、944daf6f、d849e73d、a49d38a1)のどれでも `56e435bfa976` を返し、`- Approved:` の値と一致した。承認のあとの書き込みは `## Progress checklist` の中だけ。

仕様の受け入れ条件は、依頼にある 99〜100 行が HEAD では 100〜101 行にある(この差分が FR-4 の下に「4 段目で決めたこと」を 1 行足したため)。100 行(worktree とブランチができ、leader が headless で動き、implementer と reviewer が 1 席ずつ立つ)は AC3 と AC14、101 行(承認後に本文が変わったら拒否)は AC4 で確かめた。

| Acceptance criterion | Status | Evidence |
| --- | --- | --- |
| AC1 分割計画の読み込みと各拒否 | 満たす | `internal/org/split.go` の `parseSplitPlan`・`newSplitFeatureDraft`・`(*splitFeatureDraft).read`・`parseSplitReserve`(`NormalizeReservePaths` を通す)・`parseSplitDependsOn`・`finishSplitPlan`。`Type` の既定 feat は `finishSplitPlan`、型の一覧は `/plan` の 11 種(`.claude/skills/plan/SKILL.md` 26 行)と同じ。`TestLoadSplitPlan_Fields`(9 件)と `TestLoadSplitPlan_Rejects` が、AC の 7 種の拒否をそれぞれ 1 件以上持つ |
| AC2 digest の一致と digest が読まない行の拒否 | 満たす | `PlanDigest`(`split.go:111`)を `scripts/plan-visual.sh` の `digest_body` の awk と規則ごとに突き合わせた: `## ` の行での節の切り替え、`^- (Status\|Approved\|Branch):` の除外、`^[[:space:]]*- \[[xX]\]` の書き換えの位置、awk のレコードの扱い(最後の改行の有無、空のファイル)、CRLF をバイトのまま比べること。`TestPlanDigest_MatchesScript` は本物のスクリプトを 13 個の生成した入力と `docs/plans/{active,archive}` の全計画・雛形にかける。拒否は `rejectDigestSkippedLine`・`(*splitHeader).read`・ヘッダのあとの `isStatusOrApprovedLine` で、`TestLoadSplitPlan_Rejects` の AC2 の 13 件と `TestLoadSplitPlan_RejectsEditsTheDigestSkips`(digest が変わらない編集 10 件がすべて拒否される)が固定する。出荷する `templates/base/scripts/plan-visual.sh` は root と `cmp` で同一 |
| AC3 start --plan で worktree・leader・予約・結びつき | 満たす | `(*Org).StartFeature`(`feature.go:161`)。`Ensure` に渡す引数は計画の Scope(31 行)の `ensure --id org-<org_id> --kind org --branch <type>/<slug> --path .claude/worktrees/org-<org_id> --canonical-ref split:<id>#<slug> --cleanup-policy manual` と一致する。`startFeatureLeaderParams` が cwd・予約・結びつき・scope・task を組む。`TestStartFeature_StartsLeaderInFeatureWorktree` が ensure の引数、予約の Details(`paths=… split=… feature=… digest=… branch=…`)と `Worktree`、leader の cwd、プロンプトの task を見る。`TestOrgStartPlan_RealWorktreeScript` は本物の `ralph-worktree.sh` で worktree とブランチができることを見る。実機は AC14 の Run 3 |
| AC4 未承認・digest なし・承認後の変更で拒否、何も残らない | 満たす | `readStartFeature`(`feature.go:274`)が副作用の前に `CheckApproved` を呼ぶ。`TestStartFeature_RefusedBeforeAnyRecord` の 4 件(Draft、digest なし、本文の変更、承認後に足した `- Branch:`)が、worktree の呼び出しも台帳の記録もないことをスナップショットで確かめる。CLI 側は `TestOrgStartPlan_RealWorktreeScript` の「a body edited after the approval」 |
| AC5 フラグの排他と `start <task>` の互換 | 満たす | `checkOrgStartPlanInput`(`internal/cli/org.go:656`)、`ResolveSplitPlanPath`、`readStartFeature` の知らない `--feature`。`TestOrgStartPlan_FlagCombinationsRefused`(12 件、state dir も herdr の呼び出しもできないこと)。`internal/cli/org_test.go` はこの差分にないので、既存の `start <task>` のテストは変わっていない。`TestOrgStart_WithoutPlan_StillTakesExactlyOneTask` が cobra の引数検査を固定する |
| AC6 打ち直し・別の結びつき・disband のあと・チェックアウトの違い | 満たす | 冪等は `idempotentRespawnDecision` → `reservationDecision`(同じパスと `sameFeature` で記録なし)。使い回しの検査は `checkFeatureWorktreeReuse`(`feature.go:357`)。テストは `TestStartFeature_SameStartAgainReusesWorktree`、`_OtherBindingRefusedWithoutWorktree`(別の機能と、承認し直した digest)、`_AfterDisband`(別の分割計画の同じ slug は拒否、同じ機能の承認し直しは使い回し)、`_WorktreeRecordMismatchRefused`(拒否 9 件、使い回し 1 件、スクリプトがない場合 1 件)。`WorktreeRecord` の JSON のキーは `ralph-worktree.sh` の jq の書き出し(`kind`、`worktree_path`、`branch`、`canonical_ref`)と同じ名前 |
| AC7 結びつきのない走っている org への拒否 | 満たす | `reservationDecision`(`reserve.go:479`)の 2 つの分岐(予約あり・結びつきなし、予約なし・走っている)。`TestStartFeature_PrecheckRefusesWithoutWorktree` の最初の 3 件(昇格した leader の org、`start <task>` の org、`--reserve` だけの org)が、worktree の lookup より前に拒否されることを見る。結びつきなしの予約を結びつきのある org に渡す spawn は `TestOrgSpawn_Feature_PlainReserveIntoBoundOrgRefused` |
| AC8 先読みでの拒否と、ロックの下で拒否されたときの案内 | 満たす | `spawnPrecheckErr`(`spawn.go:1106`)は `Spawn` のロックの下の判定を同じ順で呼ぶ。`TestSpawnPrecheckErr_MatchesSpawn` が同じ台帳で両者のエラー文字列の一致を見る。`TestStartFeature_PrecheckRefusesWithoutWorktree` の `max_orgs`・`max_total_seats`・重なりの 3 件。後半は `TestStartFeature_SpawnRefusedAfterEnsure`(ensure の中で別の org が重なる予約を取る。エラー文の打ち直しと `cleanup` の案内を全文で固定)と `_SpawnFailureKeepsWorktree` |
| AC9 main が clean な default branch でないときの拒否 | 満たす | `ensureFailureErr`(`feature.go:248`)。照合する 6 つの文字列はどれも `scripts/ralph-worktree.sh` にある(grep で 1/3/1/1/1/1 件)。`templates/base/scripts/ralph-worktree.sh` は root と同一。終了コード 1 は `cmd/ralph/main.go:31`。`TestStartFeature_EnsureFailureRefused`(8 件)と、本物のスクリプトで main が dirty な場合(org 層と CLI 層)。default branch でない場合は fake と文字列の存在の検査だけ(Coverage gaps) |
| AC10 close の失敗の補償で結びつきと worktree も戻る | 満たす | `reserveAgain`(`verbs.go:1327`)と `releasedReservation` が `Reservation` ごと写す。`TestOrgCloseDeferredSelf_CloseFails_BindingRestored` の pane と workspace の 2 件が、戻した Details と `Worktree`、`ActiveFeature` を見る。`TestReleasedReservation_KeepsBinding` |
| AC11 status の `feature:` の行と JSON の `feature` | 満たす | `printStatusTable`(`internal/cli/org.go:1272`、`reserved:` の次)と `printStatusJSON`。`TestOrgStatus_FeatureLineOnlyForBoundReservation`、`TestOrgStatus_IncompleteFeatureBinding`。実機の Run 3 の status の出力にも同じ形の行がある |
| AC12 leader の雛形 | 満たす | `internal/org/prompts/leader.md` の 14〜24 行(既定の 2 席)と 55〜98 行(機能の計画、secret scan、`gh pr create`、`/pr` を使わない、worktree を消さない、report、disband)。`Solo`・`Leaded`・`Parallel`・`編成パターン` は grep で 0 件で、`TestRenderRolePrompt_Leader_FeatureOrgProcedure` がこの 4 語を禁じる(`prompts_test.go:497`)。締めの順(report を PR の前)は計画の進捗の S4 に記録したずれ |
| AC13 `/org` skill・README・AGENTS.md・仕様と同期ゲート | 満たす | skill の 4 面は `cmp` で同一。「機能ごとの org」(`SKILL.md:265`)、動詞の表の `start`(175 行)と `status`(170 行)、「Leader 運用 2 経路」、サイクルの 1・2・7。`README.md:249`、`AGENTS.md:90`、仕様の 48 行。`check-skill-sync.sh` は 13 skill が lock-step、`check-sync.sh` は DRIFTED 0 |
| AC14 実機 | 満たす(観測の証拠) | `docs/evidence/org-feature-worktree-live-2026-10-09.md` の Run 3: 3 席の pane の cwd と台帳の worktree が機能の worktree の絶対パスで、台帳は main の 1 つ。実機の確認は af138af6 のバイナリで行い、8aae7ce2・2c1bb13b のあとは打ち直していない。そのあとの変更のうち成功の経路に入るのは、20 文字の上限の検査(slug `hello` は 5 文字)と雛形の文(予約の外に書く手順の明記、`--id` を `implementer` と `reviewer` にする指示)。Run 3 の leader はもともとその 2 つの id で座席を立てていたので、結果は変わらないと見ている。未確認です。push は remote がないので失敗し、`gh pr create` まで進んでいない(計画の Non-goals と evidence に書いてある) |

FR-4 の各項目: worktree とブランチを clean な default branch から作る(`ensure`)、担当範囲を分割計画から読み digest で拒否する(AC3、AC4)、既定の 2 席(雛形と skill、機構では強制しない。計画の Non-goals どおり)、編成パターンの廃止(AC12、AC13)はどれも満たす。「director の配下に置ける leader は headless だけ」は、4 段目では「結びつきは `start --plan` だけが作り、結びつきのない走っている org への `start --plan` を拒否する」ところまでで、director そのものは後の段。仕様の 48 行がそう記録している。FR-11 のうち 4 段目の分(`/org` skill)は満たす。雛形の「人に上げる」を `ralph org escalate` に置き換える部分は 5 段目。

## Static analysis

| Command | Result | Notes |
| --- | --- | --- |
| `./scripts/run-static-verify.sh` | pass(rc 0、20.2 秒) | shellcheck、hook の `sh -n`、`jq -e` の settings 2 つ、Codex の hook の 3 つのガード、`check-sync.sh`(IDENTICAL 164、DRIFTED 0、ROOT_ONLY 0)、`check-pipeline-sync.sh`、`check-skill-sync.sh`(13 skill)、`check-template-purity.sh`、tech-debt の計画の参照、golang(`gofmt: ok`、`go vet` は出力なし、`golangci-lint` は `0 issues.`、staticcheck は出力なし)、`secret-scan-branch`(765da6bd..a49d38a1 clean) |
| `git diff --check 765da6bd...HEAD` | pass | 出力なし |
| 追加行の U+FFFD | pass | 0 件 |
| コミットメッセージの帰属の行 | pass | `Co-Authored-By` / `Generated with` は 0 件 |
| `./scripts/plan-visual.sh digest`(計画、HEAD と計画を触った 4 コミット) | pass | すべて `56e435bfa976` |
| `go build -o <scratch>/ralph ./cmd/ralph` と `org start --help` / `org status --help` | pass | 文書のずれの確認に使った。scratch の外には何も置いていない |

## Documentation drift

| Doc / contract | In sync? | Notes |
| --- | --- | --- |
| `ralph org start --help` と skill の `start` の行・「機能ごとの org」 | ほぼ一致 | V-1、V-2 |
| `ralph org status --help` と skill の `status` の行・予約の節・コード | 一致 | `feature:` の行の形、JSON の 5 キー、`(incomplete record)` と `"incomplete": true` |
| leader の雛形と `featureLeaderTask`(`feature.go:328`) | 一致 | task の 1 行目が `- 分割計画:`、`- ブランチ:`、`- 予約したパス:` の行がある。`- 進め方:` が指す「機能ごとの org」の節は skill の 265 行にある |
| leader の雛形の手順と skill の「org の終わり方」 | 一致 | stop と report → archive とコミット → secret scan → push と `gh pr create` → disband の順 |
| `README.md:249`、`AGENTS.md:90`、仕様の 48 行 | 一致 | |
| skill と雛形が使うスクリプトの出荷 | 一致 | `templates/base/scripts/` に `ralph-worktree.sh`・`plan-visual.sh`(root と同一)、`archive-plan.sh`、`secret-scan-branch.sh` がある |
| `.claude/worktrees/` と `.harness/state/` の gitignore | 一致 | root と `templates/base/` の `.gitignore` の 42・47 行 |
| 出荷する面の director の言及 | 一致 | skill の 4 面、`templates/base/`、`internal/org/prompts/`、README、AGENTS.md で 0 件(self-review の M2) |
| `internal/cli/doctor.go` の `solo execution` | 参考 | V-3 |
| 計画の進捗のずれの記録 | 一致 | S3 の分割、締めの順、af138af6、8aae7ce2・2c1bb13b(20 文字の上限など)。self-review の N3 の進捗の部分は a49d38a1 で直った |

### Findings

| ID | Severity | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- |
| V-1 | LOW | `--plan` の usage が併用できないものを「the task argument, --cwd, --scope and --reserve」と並べ、`--allow-unscoped` を落としている。Long の説明、skill、`checkOrgStartPlanInput` は `--allow-unscoped` も拒否する | `internal/cli/org.go:635`。`ralph org start --help` の Flags の `--plan` の行 | usage に `--allow-unscoped` を足す |
| V-2 | LOW | `start --plan` の org_id の 20 文字の上限が、`start --help` の Long にも `--org-id` の usage にも書かれていない。skill(「分割計画」と「start が拒否するもの」)とエラー文には書いてある。self-review の N3 の help の部分がまだ残っている | `internal/cli/org.go:44`、`:555` | Long の「The org_id is the feature's slug unless --org-id is given」に上限を 1 文足す |
| V-3 | LOW(参考) | `ralph doctor` の herdr と agmsg の info が `(solo execution unaffected)` と書く。廃止した Solo の頃(844df054)の言い方で、skill の対応する文(「座席 0 のソロ実行のみ」)はこの差分で「org を使わない標準フロー」に直った。意味は今も大きくは外れていない | `internal/cli/doctor.go:731`、`:761` | 計画の範囲の外。直すなら「standard flow unaffected」など。後の段に回してよい |
| V-4 | 参考 | 実機の記録の Run 3 の start の出力が、`runOrgStartPlan` が出す 3 行目の `hint:` の行を省いている。af138af6 の時点でもこの行は出ていたので、抜粋とみなした | `docs/evidence/org-feature-worktree-live-2026-10-09.md` の Run 3、`internal/cli/org.go` の `runOrgStartPlan` | 抜粋と書くか、そのままでよい |

V-1〜V-4 はどれも挙動を変えず、merge を止めない。self-review の残り(L5、N1、N2 は diff の品質、L2 の台帳の行と計画の進捗にある 4 件の tech-debt は /sync-docs)は /verify の対象外なので、ここでは状態だけを確かめた。

## Observational checks

- 実機の記録(AC14)を読み、Run 1 と Run 2 で見つかった 2 つの穴(pane の中の ralph の版、相対の `--cwd`)が、後者は af138af6 で直され、前者は tech-debt に送る項目として計画の進捗(169 行の (1))にあることを確かめた
- `ralph org start --help` と `ralph org status --help` の出力を、このブランチのビルドで取った(scratch に保存。V-1、V-2 の根拠)

## Coverage gaps

- テストは実行していない。上の表のテスト名は、何を固定しているかを読んで確かめたもので、通ることは /test で確かめる
- AC9 の「main のチェックアウトが default branch でない」は、fake の ensure と、スクリプトにその文があることの検査だけで見ている。本物のスクリプトで通すのは dirty の場合だけ
- AC14 の実機は af138af6 のバイナリで、8aae7ce2・2c1bb13b のあとは打ち直していない。push と `gh pr create` が通る場合(remote のある repo)と codex の leader は未確認(計画の Non-goals と evidence の結論に書いてある)
- 同じ機能の `start --plan` を同時に 2 つ打つ場合(計画の進捗の (4))は見ていない
- digest の一致は規則ごとに読んで確かめた。114 個の入力での一致(計画の進捗の S1)は /test の実行で確かめる

## Verdict

- Verdict: pass
- Verified: AC1〜AC13 をコードとテストの中身で、AC14 を実機の記録で確かめた。計画の承認の digest が HEAD まで一致すること。`run-static-verify.sh` が rc 0 で終わること(Go の gofmt・vet・golangci-lint・staticcheck、同期ゲート、secret scan を含む)。`status --help` と雛形・skill・README・AGENTS.md・仕様がコードと合っていること
- Partially verified: `start --help` は V-1(`--plan` の usage)と V-2(20 文字の上限)が残る。AC9 の default branch でない場合は fake だけ。どれも merge を止めない
- Not verified: テストの実行、8aae7ce2 以降のバイナリでの実機、push と `gh pr create` が通る場合、codex の leader、同時の start
