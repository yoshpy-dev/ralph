# Verify report: org-feature-worktree

- Date: 2026-10-09(cycle 1)、2026-10-10(cycle 2)
- Plan: docs/plans/active/2026-10-09-org-feature-worktree.md(承認済み、digest 56e435bfa976)
- Verifier: verifier subagent (Claude Opus 5.5)。pipeline cycle 1(`cycle-count.json` は 1、上限 2)と、cross-review(cycle 1)の ACTION_REQUIRED 2 件を直したあとの cycle 2(上限の回)。現況は末尾の「Cycle 2」の節と「Verdict」
- Scope: 仕様への適合(AC1〜AC14、仕様の FR-4、FR-11 の `/org` skill の部分、受け入れ条件)、静的解析、文書のずれ。cycle 1 の対象は `git diff 765da6bd...HEAD`(HEAD a49d38a1、14 コミット、29 ファイル、+6457/-374)。cycle 2 の対象は `git diff 183cb190 HEAD`(HEAD c322e7b3、5 コミット、17 ファイル、+678/-105)で、全体は `git diff 765da6bd...HEAD`(23 コミット、38 ファイル、+7597/-441)。テストは実行していない(`/test` の担当)。テストについては、何を固定しているかを読んで確かめた
- Evidence: `docs/evidence/verify-2026-10-09-org-feature-worktree.log`(`docs/evidence/*.log` は gitignore の対象なので手元にだけ残る。cycle 2 の出力は末尾の `===== cycle 2` の区切りのあとに足した)。runner 自身のログは cycle 1 が `docs/evidence/verify-2026-10-09-111530.log`、cycle 2 が `docs/evidence/verify-2026-10-09-164603.log`(runner の時刻は UTC)

「Spec compliance」から「Coverage gaps」までの節は cycle 1(HEAD a49d38a1)の記録で、書き換えていない。cycle 2 の確認は「Cycle 2」の節にある。

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

## Cycle 2(d49bbc34・07d38e6d・c2a1f8c4・9c1d447f・c322e7b3 のあと)

対象は `git diff 183cb190 HEAD`。`git log --oneline 183cb190..HEAD` は d49bbc34(leader に台帳を渡す、分割計画のコードフェンス)、07d38e6d(`--state-dir` が main の台帳を指すときの全体の上限)、c2a1f8c4(計画の進捗)、9c1d447f(その help の文)、c322e7b3(self-review の cycle 2)の 5 つ。作業ツリーは clean で、`origin/feat/org-feature-worktree` と同じ位置。

### cross-review の ACTION_REQUIRED の確認

| # | 指摘 | 直し(コード) | 固定するテスト(読んで確認) | 判定 |
| --- | --- | --- | --- | --- |
| 1 | `--state-dir` か env で立てた start の leader に台帳の場所が渡らず、leader の座席が別の台帳に入る | d49bbc34: `StartFeature` が解決済みの `p.StateDir` を `featureLeaderTask` に渡し(`feature.go:320`)、task の最後のヘッダ行に `- 台帳: <path>(ralph org のコマンドには必ず --state-dir '<path>' を付ける)` を足す(`:347`、引用は `shellQuote` `:358`)。雛形 `leader.md:65-70` が spawn・send・wait・read・status・stop・report・disband のすべてに付けるよう書き、skill の 4 面の 396〜404 行が同じことを書く。07d38e6d: leader の spawn は source が `flag` になるので、`withMainWorktreeOrgLimits`(`internal/cli/org.go:160`)が `org.LedgerMainWorktreeRoot`(`statedir.go:105`)で、指した台帳が打った場所の repository の main worktree の `.harness/state/org` と同じかを見て、同じなら main の `ralph.toml` の上限を使う。9c1d447f: `orgWideLimitsHelp`(`internal/cli/org.go:375`)をこの挙動に合わせた | `TestStartFeature_StartsLeaderInFeatureWorktree`(spawn された leader のプロンプトに、台帳の行が `st.stateDir` で入る)、`TestFeatureLeaderTask`(空白と `'` を含むパス)、`TestFeatureLeaderTask_StateDirIsOneShellWord`(5 つの値を `sh` に通して 1 引数になる)、`TestRenderRolePrompt_Leader_FeatureOrgProcedure`(雛形の導入の段落)、`TestLedgerMainWorktreeRoot`(肯定 4 件と否定 5 件を 3 つの cwd で、git の外)、`TestOrgStart_OrgWideLimits_ReadFromMainWorktreeRalphToml`(linked worktree の `ralph.toml` が `max_orgs = 99` でも、絶対と相対の `--state-dir` と env で main の台帳を指すと拒否、別の台帳を指すと許可)、`TestOrgSpawnAndStartHelp_OrgWideLimitsSource` | コードとテストの上では解消。leader が実際にこの行に従うことは実機で走らせていない(Coverage gaps) |
| 2 | 機能の本文のコードフェンスの中の `## Usage` などで節が閉じ、承認は通るのに leader に渡る本文が短くなる | d49bbc34: `splitFence`(`split.go:377`)、`read`(`:385`)、`fenceRun`(`:414`)、`unclosedErr`(`:402`)。`parseSplitPlan` はフェンスの中の行を見出しにもフィールドにもせず(`:315`)、機能の中なら本文に足す。閉じていないフェンスは開いた行を指して拒否する。`rejectDigestSkippedLine` はフェンスの判定より前に全行へ掛かるので、digest が読まない行はフェンスの中でも拒否される。`PlanDigest` は cycle 2 の差分にない | `TestLoadSplitPlan_CodeFences`(14 件と、フェンスの中の `## Features` で header が終わらないこと)、`TestLoadSplitPlan_Rejects` に足した 13 件、`TestLoadSplitPlan_FencedBodyIsApproved`(`CheckApproved` が通り、ブロックごと本文に入り、後ろの機能も読まれ、フェンスの中の行の編集で digest が変わる)、`TestPlanDigest_MatchesScript` に足した入力 `code fences`(`split_test.go:791`) | 解消。開閉の規則は、読んで確かめた範囲で CommonMark の fenced code block と合う(字下げは空白 3 つまででタブは不可、バッククォートの info string にバッククォートがあれば開かない、閉じる行は同じ文字で開いた長さ以上、後ろは空白とタブだけ) |

1 件目について、ほかに確かめたこと:

- state dir の source で挙動が変わる所は、`guardLegacyOrgStateDir`(`flag` では何もしない)、`withMainWorktreeOrgLimits`、`FeatureRepoRoot`(start だけ)、`watch` と `status` の表示の 4 か所だけ(`git grep stateDirSource`)。leader が `--state-dir` を付けても、旧台帳の検査が外れるほかに変わる点はない。この版の ralph は feature worktree の中に旧台帳を作らない
- agmsg の team は org_id だけから決まる(`spawn.go:1424` の `agmsgTeam`)。implementer と reviewer の雛形には `ralph org` のコマンドがないので、台帳の行が要るのは leader だけ
- skill の「pane の ralph が古い版で台帳を別の場所に決める場合にも、start と同じ台帳を使わせる」は、v5.1.0 の `org` にも persistent flag の `--state-dir` があること(`git show v5.1.0:internal/cli/org.go` の 41 行)まで確かめた。古い版で実際に打つことはしていない

### cycle 2 の差分が触れる AC の再確認

| Acceptance criterion | Status | Evidence |
| --- | --- | --- |
| AC1、AC2 | 満たす | フェンスは読み込みの規則に足しただけで、`Rejects` の既存のケースは変わっていない(差分は追加だけ)。AC2 の「digest が読まない書き方の拒否」はフェンスの中でも続く(上の表の 2) |
| AC3 | 満たす | leader の task に台帳の行が足された。機能の本文はフェンスごと入る(`TestStartFeature_StartsLeaderInFeatureWorktree`、`TestLoadSplitPlan_FencedBodyIsApproved`) |
| AC4 | 満たす | 承認のあとにフェンスの中へ `- Branch:` を足しても拒否される(`Rejects` の「branch inside a fence」) |
| AC5〜AC11 | 満たす(cycle 1 のまま) | `spawn.go`・`reserve.go`・`verbs.go` は cycle 2 の差分にない。`internal/cli/org.go` の変更は `withMainWorktreeOrgLimits` の読み口と help の文字列だけ |
| AC12 | 満たす | `leader.md` の新しい段落は台帳の説明だけで、`Solo`・`Leaded`・`Parallel`・`編成パターン` は 0 件のまま |
| AC13 | 満たす | skill の 4 面は `cmp` で同一。`check-skill-sync.sh` は 13 skill が lock-step、`check-sync.sh` は DRIFTED 0。`grep -rnw director` は skill・`templates/base`・`internal/org/prompts`・README・AGENTS.md で 0 件 |
| AC14 | 満たす(cycle 1 の観測のまま) | 実機は af138af6 のバイナリで、そのあと `leader.md` が変わった(`git diff af138af6 HEAD --stat -- internal/org/prompts/` は +18/-6)。Coverage gaps |

計画の承認: `./scripts/plan-visual.sh digest` は HEAD、c2a1f8c4、183cb190 のどれでも `56e435bfa976`。cycle 2 で計画に入った変更は `## Progress checklist` の 1 行(169 行)だけ。

07d38e6d は 3 段目の挙動を 1 つ変えた。`--config` がなく、`--state-dir` か `RALPH_ORG_STATE_DIR` で main の台帳を指したとき、これまでは打った場所の `ralph.toml` を読んでいたが、今は main の `ralph.toml` を読む。理由(leader が `--state-dir` を付けても、feature branch の `ralph.toml` で全体の上限を変えられないようにする)は 07d38e6d のメッセージと計画の進捗 169 行にある。仕様の 41 行の要約(「`--config` がなければ、全 org の上限は main worktree のルートの `ralph.toml` から読む」)は、この変更のあとも成り立つ。

### Static analysis(cycle 2)

| Command | Result | Notes |
| --- | --- | --- |
| `./scripts/run-static-verify.sh` | pass(rc 0) | shellcheck、hook の `sh -n`、`jq -e` の settings 2 つ、Codex の hook の 3 つのガード、`check-sync.sh`(IDENTICAL 164、DRIFTED 0、ROOT_ONLY 0)、`check-pipeline-sync.sh`、`check-skill-sync.sh`(13 skill)、`check-template-purity.sh`、tech-debt の計画の参照、golang(`gofmt: ok`、`golangci-lint` は `0 issues.`、`go vet` と staticcheck は出力なし)、`secret-scan-branch`(765da6bd..c322e7b3 clean) |
| `git diff --check 183cb190 HEAD` と `git diff --check 765da6bd...HEAD` | pass | 出力なし |
| 追加行の U+FFFD(`765da6bd...HEAD`) | pass | 0 件 |
| コミットメッセージの帰属の行(23 コミット) | pass | `Co-Authored-By` / `Generated with` は 0 件 |
| `./scripts/plan-visual.sh digest`(計画) | pass | `56e435bfa976` |
| `go build -o <scratch>/ralph ./cmd/ralph` と `org start --help` / `org spawn --help` | pass | 文書のずれの確認に使った。出力は evidence のログに足した |

### Documentation drift(cycle 2)

| Doc / contract | In sync? | Notes |
| --- | --- | --- |
| `ralph org spawn --help` と `ralph org start --help` の全体の上限の段落 | 一致 | 「found by default or named with --state-dir or RALPH_ORG_STATE_DIR」と「a --state-dir or RALPH_ORG_STATE_DIR naming another ledger」が `LedgerMainWorktreeRoot` の分岐と合う |
| skill の 4 面の「全体の上限」(215〜226 行) | 一致 | symlink を解決して比べることまで書いてある(`samePath`) |
| skill の台帳の段落(396〜404 行)、`leader.md:65-70`、`featureLeaderTask` | 一致 | 動詞の一覧は 3 か所とも同じ 8 つ。引用は単一引用符。「leader には届かない」の言い切りは self-review の C2-2 (c) |
| skill のフェンスの説明(334〜336 行、349〜350 行)と `parseSplitPlan` | 一致 | |
| `templates/base/ralph.toml:42-46` の `max_orgs` のコメント | 不一致 | V2-1 |
| `docs/tech-debt/README.md` の行 146・184 の (d)・185 の (c)・188 | 古い | V2-2 |
| 仕様の FR-4 の「4 段目で決めたこと」(48 行)、計画の進捗 | 参考 | V2-3 |
| `docs/recipes/worktrees.md:57`、`docs/recipes/codex-seat-permissions.md:80` | 一致 | 台帳の置き場所だけを書き、上限の読み元には触れない |
| cycle 1 の V-1・V-2・V-4 | 解消(6312b5d6) | `start --help` の `--plan` の usage に `--allow-unscoped`、Long と `--org-id` の usage に 20 文字の上限、実機の記録の Run 3 に `hint:` の行。V-3 は `internal/cli/doctor.go:731`・`:761` に残り、台帳に行がある |

### Findings(cycle 2)

| ID | Severity | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- |
| V2-1 | LOW | `ralph init` で配る `templates/base/ralph.toml` の `max_orgs` のコメントが、07d38e6d のあとの挙動と逆のことを書く。コメントは「only when --config is not given, neither --state-dir nor RALPH_ORG_STATE_DIR is set, and the state dir resolves to the main worktree's .harness/state/org」で、今は `--state-dir` か env で main の台帳を指しても main の `ralph.toml` を読む。機能ごとの org の leader は `--state-dir` を必ず付けるので、コメントが外している場合がこの PR の中心の経路に当たる。self-review の C2-1 (b) と同じ | `templates/base/ralph.toml:42-46`、`internal/org/statedir.go:105`、`internal/cli/org.go:160`。help(`:375`)と skill(215〜226 行)は直っている | /sync-docs で、help と同じ条件(「--config がなく、台帳が main worktree の .harness/state/org のとき。既定の解決でも、--state-dir や RALPH_ORG_STATE_DIR で指しても」)に書き直す。値は変えない |
| V2-2 | LOW | 台帳の 4 行が cycle 2 の直しで古くなった。146 行の「`--config` or `--state-dir` avoids it」は、main の台帳を指す `--state-dir` では成り立たない。184 行の (d) の提案文は「neither --state-dir nor RALPH_ORG_STATE_DIR is set」で、今の挙動と逆。185 行の (c)(help の文を固定するテストがない)は、spawn と start の分を `TestOrgSpawnAndStartHelp_OrgWideLimitsSource` が固定した。188 行の「Nothing in `StartFeature`, in the leader prompt … hands the resolved state dir to the leader」は HEAD では偽。self-review の C2-6 の (a)〜(d) と同じ | `docs/tech-debt/README.md` の 146・184・185・188 行。cycle 2 の差分に台帳はない | /sync-docs で、self-review の C2-6 の最後の列のとおりに直す |
| V2-3 | 参考 | 仕様の FR-4 の「4 段目で決めたこと」に、フェンスの規則、leader の task の台帳の行、main の台帳を指す `--state-dir` が main の上限を使うこと、の 3 つが書かれていない。仕様は「詳細は 4 段目の計画に書いた」とし、計画の進捗 169 行に 3 つともあるので、食い違いではない。計画の進捗は 9c1d447f(help の文)を挙げていない | 仕様の 48 行、計画の 169 行 | /sync-docs が触れるなら、仕様に 1 文、計画の進捗に 9c1d447f を足す。足さなくても merge は止めない |

### Coverage gaps(cycle 2)

- テストは実行していない。上の表のテスト名は中身を読んで確かめたもので、通ることは /test で確かめる
- AC14 の実機は af138af6 のバイナリで、そのあと `leader.md` と task の文が変わった。claude の leader が台帳の行に従ってすべての `ralph org` のコマンドに `--state-dir` を付けるか、引用したパスを pane の中でそのまま打てるかは、実機で確かめていない。Run 3 の上限の結果は、既定の解決でも `--state-dir` でも main の `ralph.toml` になるので変わらないと見ている。未確認です
- pane の ralph が v5.1.0 のとき、flag があることまでは確かめたが、その版で今の台帳のイベントを読ませることはしていない
- まだない台帳を symlink を含む別名のパスで `--state-dir` に渡すと、`samePath` が文字列の比較になる場合(self-review の「finding にしないもの」)は、試していない
- `ralph org start <task> --state-dir X` の leader に台帳が渡らない件(self-review の C2-6 (g))は、計画の範囲の外なので見ていない

## Verdict

- Verdict: pass
- Verified(cycle 2): cross-review の ACTION_REQUIRED 2 件が、コード(`featureLeaderTask` の台帳の行と `LedgerMainWorktreeRoot`、`splitFence`)とテストの中身で直っていること。cycle 2 の差分が触れる AC(AC1〜AC4、AC12、AC13)を満たし、AC5〜AC11 のコードが cycle 2 の差分にないこと。計画の承認の digest が HEAD まで一致すること。`run-static-verify.sh` が rc 0 で終わること(Go の gofmt・vet・golangci-lint・staticcheck、同期ゲート、765da6bd..c322e7b3 の secret scan を含む)。help・skill の 4 面・雛形が新しい挙動と合っていること
- Partially verified: 文書は V2-1(配る `ralph.toml` のコメント)と V2-2(台帳の 4 行)が古いまま残る。どちらも挙動を変えず、/sync-docs で直せる
- Not verified: テストの実行、d49bbc34 以降のバイナリでの実機(leader が `--state-dir` を付けること)、push と `gh pr create` が通る場合、codex の leader、同時の start
- 参考(cycle 1 の判定、HEAD a49d38a1。V-1・V-2・V-4 はそのあと 6312b5d6 の sync-docs で直った): pass
  - Verified: AC1〜AC13 をコードとテストの中身で、AC14 を実機の記録で確かめた。計画の承認の digest が HEAD まで一致すること。`run-static-verify.sh` が rc 0 で終わること(Go の gofmt・vet・golangci-lint・staticcheck、同期ゲート、secret scan を含む)。`status --help` と雛形・skill・README・AGENTS.md・仕様がコードと合っていること
  - Partially verified: `start --help` は V-1(`--plan` の usage)と V-2(20 文字の上限)が残る。AC9 の default branch でない場合は fake だけ。どれも merge を止めない
  - Not verified: テストの実行、8aae7ce2 以降のバイナリでの実機、push と `gh pr create` が通る場合、codex の leader、同時の start
