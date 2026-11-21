# Verify report: org-xreview-own-model

- Date: 2026-11-22(JST。pipeline の 1 周目、`cycle-count.json` は `cycle: 1`)
- Plan: `docs/plans/active/2026-11-22-org-xreview-own-model.md`(issue #360)。`- Approved:` の digest `aa9ca6400c54` は、HEAD で `./scripts/plan-visual.sh digest` を打ち直しても一致する
- Verifier: verifier サブエージェント(`/verify`)
- Scope: `git diff origin/main...HEAD`(HEAD `b98b1b96`、origin/main `bd079dbb`、36 ファイル、+2039 −1817)。HEAD は origin/main(#362 と #366 の merge まで)を取り込んだ merge commit で、取り込みの衝突は `docs/tech-debt/README.md` の 1 行だけ。self-review のあとの直し `f95a08f9`(M1、L1、L3、L4)を含む。テストは走らせていない(`/test` の仕事)
- Evidence: `docs/evidence/verify-2026-11-22-org-xreview-own-model.log`(`.gitignore` の `docs/evidence/*.log` に当たるので commit されない)。`run-static-verify.sh` の出力のあとに、AC6 の確かめ、写しの `cmp`、消した語の grep と Python のポート、台帳の Trigger の照合、digest を足した。wrapper 自身のログは `docs/evidence/verify-2026-11-21-213815.log`(UTC の日付)

CRITICAL と HIGH はない。MEDIUM 1(V1)、LOW 3(V2〜V4)、INFO 4(V5〜V8)。

## Spec compliance

| Acceptance criterion | Status | Evidence |
| --- | --- | --- |
| AC1 | Met | `internal/org/prompts/reviewer.md` の「cross-review の TASK」(`:164-391`)は、座席が自分で差分と変わったファイルを読んで所見を出し(手順 3、`:247-270`)、triage する(手順 4)。`codex exec`、`claude -p` は、呼ばないと言う 1 文(`:171-172`)にだけ出る。この 1 文と、前の版の行を読まない 1 文(`:182-184`)を除くと、`pick_reviewer`、`mcp list`、`sleep 1200`、`xreview-helpers.sh`、`-out.md`、`相手`、`20 分`、`skipped` のどれも節にない(evidence の 7、Python のポートで確かめた)。手順 2 が読む前の git の状態を `-git-before.txt` に書き、手順 5 が report を書く前に `-git-after.txt` と `cmp` で比べ、違えば report を書かずに `XREVIEW: git_changed` の BLOCKED を返す(`:294-312`)。git の状態を取る式は main の版と 1 字も変わらない。固定するテストは `TestRenderRolePrompt_Reviewer_XreviewReviewsOnTheSeatsOwnModel` と `..._XreviewComparesTheGitStateBeforeTheReport` ほか。字面の「持たない」は、Scope が求める禁止の 1 文を除いて読む(V8) |
| AC2 | Met | 手順 1(`:188-197`)が確かめるのは `XREVIEW_BASE:`、`CYCLE:`、`REPORT:` の 3 行だけ。`IMPLEMENTER_DRIVER:`、`REVIEWER_CODEX:`、`REVIEWER_CLAUDE:` は読まないと書く(`:182-186`)。`diff: failed` なら `Reviewer status: incomplete (git diff failed: <要点>)` の report を書き、同じ値の RESULT を返す(`:241-245`)。`.harness/state/` は root と雛形の `.gitignore` の両方で無視される(`git check-ignore -v` で確かめた)ので、手順 2 が書く `.diff` は手順 5 の比べに入らない |
| AC3 | Met | report の見出し `# Cross-review triage report`、`Triager:`、`Reviewer status:`、`HEAD:`、`Base:`、`Cycle:`、`Total reviewer findings:`、`After triage:` の行、3 つの表(列も同じ)は main の版のまま(`:325-358`)。`- Driver: ...  Reviewer: ...` の行は `- Reviewer: reviewer 座席 {{SEAT_ID}}({{DRIVER}}、{{MODEL}})` に、`Reviewer output:` の行は `Diff:` の行に替わった。`Reviewer status:` から `skipped` が消えたのは Scope の「相手を呼ぶための skipped を消す」のとおりで、leader は前の版の reviewer のために `skipped` の扱いを残す(`leader.md:1280-1283`)。返事は `pass`、`action_required`、`incomplete`、`git_changed` の 4 つ(`:360-367`) |
| AC4 | Met | `RolePromptVars` に `Driver` と `Model`(`internal/org/prompts.go:76`、`:82`)、置き換えは `{{DRIVER}}` を `vars.Driver`、`{{MODEL}}` を `ModelOrCLIDefault(vars.Model)` に(`:146-147`、空なら `(CLI default)`)。`Spawn` と `dryRunSpawn` の 2 か所が `p.Driver` と `p.Model` を渡す(`spawn.go:1120`、`:2423`)。どちらも `AgentStart` と `--model` に渡す値と同じで、driver と model は CLI が `ResolveRoleModel` で決めてから `Spawn` に入る。雛形で置き場を持つのは `reviewer.md` の 2 行だけ(`:4`、`:329`)。`RenderRolePrompt` のほかの呼び出しは `reseat.go:920` で、空の vars で描いて文字列を捨てる。テストは `TestRenderRolePrompt_Reviewer_ShowsTheSeatsDriverAndModel`、`TestRenderRolePrompt_SeatTemplates_NoDriverOrModelPlaceholderLeft`(3 役 × model あり・なし)、`TestOrgSpawn_ReviewerPromptFileNamesTheSeatsDriverAndModel` |
| AC5 | Met | `leader.md` の `### cross-review`(`:1043`〜)は、TASK に 3 行だけを書き、implementer の driver を台帳からも記憶からも読まないと言う(`:1232-1241`)。台帳を読むコマンドと `driver: error` の扱いは消えた。小節のコードブロックを main と比べると、状態のコマンドと上限の QUESTION は 1 字も変わらず、消えたのは driver のコマンド、変わったのは TASK の例だけ(evidence の 8)。`XREVIEW: git_changed` は先に進まず BLOCKED を上げ、戻したら送り直しの上限の中で同じ `<k>` を送り直す(`:1294-1305`)。`Cross-review:` の値は最後の report の `Reviewer status:` から決め、`ran (<report の Reviewer:>)` と書く(`:1315-1320`) |
| AC6 | Met(振る舞い)。字面は満たさない(V7) | 標準フローの `/cross-review` の skill は 4 写しとも差分に入らない。`scripts/xreview-helpers.sh` と写しは差分に入るが、変わった行はすべて `#` で始まり(`#` で始まらない変更行は 0)、コメントを除いた本体は main と同じ。root と雛形の写しは `cmp` で同じ。`tests/test-xreview-helpers.sh` は差分に入らず、その対象の関数(`pick_reviewer`、`count_triage_findings`、`detect_base_branch`)も変わらない。直したのは `pick_reviewer` の doc comment の呼び出し元の一覧で、reviewer 座席を外した。残すと「4 つ目の呼び出し元」という誤りが残る。この判断は Progress の S4 に記録がある |
| AC7 | Met | `model-routing.md`(2 写し。雛形との差は既存の KNOWN_DIFF)、org と director の skill(4 写しずつ `cmp` で同じ)、recipe 2 つ(2 写しずつ同じ)、README、仕様の 2026-11-22 の項目、`config.go` と `paneenv.go` と `spawn.go` のコメント、`templates/base/ralph.toml` のコメントが新しい動きを書く。`model-routing.md` が名指すテストの名前は 9 つともある。`check-sync.sh`(DRIFTED 0)と `check-skill-sync.sh`(14 skill)は wrapper の中で通った。`ralph init` の問いの文は Scope の一覧の外で、古いまま(V1) |
| AC8 | Met | `docs/recipes/codex-seat-permissions.md:197-201` が、`workspace-write` には許すホストの一覧がなく座席はどのホストにも出られること、閉じるには `codex_network_access = false` にして座席を立て直すこと(「A change reaches only the seats started after it」を指す)を書く。雛形の写しと同じ |

ユーザーの決定(reviewer 座席が、系列にかかわらず自分のモデルでレビューする)との照合: reviewer の雛形、leader の雛形、2 つの skill、`model-routing.md`、recipe、README、`templates/base/ralph.toml` のどれも、系列で扱いを変える文を持たない。違う系列の目が欲しいときの手(`[org.role_models].reviewer` か分割計画の前文で reviewer の driver を変える)だけを書く。recipe と org skill の「既定の 2 組では implementer と reviewer が同じ系列」は `DefaultRoleModels`(`internal/config/role_models.go:83-94`)と合い、skill の例の `gpt-6.1-sol` は `defaultCodexRoleModel` と同じ。

#366 の取り込みとの照合: `leader.md` に入った人の決定の文(立て直しの規則の `:156-161`、手順 4 の `:492-496`、上限の答えの読み方の `:1188-1190`、運用規律の項の `:1615`〜)は、cross-review の上限の QUESTION を「人の決定はこの答えではない」と扱う。この branch は上限の QUESTION と、その答えの読み方を変えていないので食い違わない。cross-review の返事を待つあいだに人の決定が worktree を変えさせても、reviewer の手順 5 が `git_changed` で拾い、見ていない変更に pass は付かない。この待ちの決まりは main にもあり、この branch が作ったものではない。`Seven test gaps of org-cross-review` の行の衝突の解き方は、branch の行に main の 2 つの挿入を足した形で、どちらの変更も落ちていない(evidence の 10)。

## Static analysis

| Command | Result | Notes |
| --- | --- | --- |
| `./scripts/run-static-verify.sh` | Pass(rc=0) | 既定の changed の範囲を求めたが、分類されない `scripts/xreview-helpers.sh` があるので full に落ちた。shellcheck と `sh -n`(hook)、`jq -e`(settings)、Codex の hook の 3 つの guard、`check-sync.sh`(IDENTICAL 172、DRIFTED 0、TEMPLATE_ONLY 10、KNOWN_DIFF 6)、`check-pipeline-sync.sh`、`check-skill-sync.sh`、`check-template-purity.sh`、tech-debt の計画の参照 |
| golang verifier(wrapper の中) | Pass | `gofmt: ok`、`go vet ./...`、`golangci-lint run ./...` は `0 issues.`、`staticcheck ./...` は出力なし |
| `scripts/secret-scan-branch.sh`(wrapper の中) | Pass | `bd079dbb..b98b1b96` を origin/main に対して読み、clean |
| `./scripts/plan-visual.sh digest` | `aa9ca6400c54` | 計画の `- Approved:` と一致 |
| 写しの `cmp` | 同じ | org と director の skill の 4 写しずつ、recipe 2 つの 2 写しずつ、`xreview-helpers.sh` の 2 写し |

## Documentation drift

| Doc / contract | In sync? | Notes |
| --- | --- | --- |
| reviewer と leader の雛形 | Yes | 上の AC1〜AC5。小節の外の手順 9、12、14 と立て直しの規則(座席の確認の項)の「相手」「20 分」も直っている。`leader.md` に残る「相手」「20 分」「`pick_reviewer`」は `### 計画助言`(`:917-1042`)だけで、Non-goals のとおり |
| org と director の skill | Yes | cross-review の説明、`IMPLEMENTER_DRIVER:` の行、`pick_reviewer` の言及、待ちの例外を直した。cross-review の停滞は、画面が動いているあいだだけ待ち、10 分動かなければ上げる、で `leader.md:766-774` と合う(`f95a08f9`) |
| `model-routing.md`、README、recipe、仕様、`templates/base/ralph.toml`、Go のコメント | Yes | 上の AC7。仕様の 2026-11-01 の項と 2026-10-29 の項に残る「違う系列」「`pick_reviewer` に座席の driver を読ませる」は、日付つきの当時の記録で、2026-11-22 の項が変えたと書く |
| `ralph init` の「変更のレビュー」の問い | No(V1) | 下の表 |
| `docs/tech-debt/README.md` | 一部(V2〜V5) | この branch は 2 行に 2026-11-22 の注を足し、新しい行(計画助言の入れ子の CLI)を足した。Trigger に当たったまま注のない行が 10 ある。行番号は HEAD のもので、行の頭の語も書く |
| `internal/cli/org_runtime_env_test.go:182-185` のコメント | No(V4) | self-review の L2。/sync-docs に回してある |
| 計画の Progress | 一部(V6) | 下の表 |

## Findings

| ID | Severity | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- |
| V1 | MEDIUM | `ralph init` の「変更のレビュー」の問いが、機能の org の cross-review を「もう一方の AI のレビュー」と説明したまま。問いの答えは `[standard].cross_review` と `[org.review].cross_review` の両方に書かれる。この branch のあと、org の cross-review は reviewer 座席が自分のモデルで行い、既定の 2 組では implementer と同じ系列になるので、説明の「Claude Code reviews the changes Codex makes, Codex reviews Claude Code's」は org の側では当たらない。CLI が 1 つのとき `[org.review].cross_review` を false で書く決まり(仕様の 2026-11-02 の項)も、相手の CLI が要るという前提を失った。計画の Scope の文書の一覧に入っておらず、AC7 の字面には当たらない | `internal/cli/init_settings.go:331-336`(問い)、`:43-44`(「Both are false with one CLI」)、`:158-160`。仕様 `docs/specs/2026-10-07-org-multi-org-director.md:149` | 問いの説明に、機能の org では reviewer 座席が自分のモデルでもう一度レビューする、と 1 文足す(Go の文字列 1 つで、説明の文を固定するテストはない)。CLI が 1 つのときに org の cross-review を聞くかは製品の判断で、Non-goals(`[org.review].cross_review` の意味と既定)の外なので、直さないなら tech-debt に行を足す |
| V2 | LOW | 台帳の Trigger の照合: この差分が当たる Trigger で、この branch の注がない行が 10 ある。「org runtime FR-7 quality gate is prompt-instructed」(161、org の品質パイプラインの変更)、「What a leader is told about its reservation」(190 の (a)、`RolePromptVars` と leader の雛形)、「Three test gaps of the org-director-attach PR」(267、director skill の `## 受信箱の処理` の中の「件の処理」と「cross-review の上限」)、「`internal/org/verbs.go` is 2277 lines」(370、leader の雛形のテスト。`prompts_test.go` 2727→2787、`spawn.go` 3122→3127 で +300 には届かない)、「The recipes under `templates/base/docs/recipes/`」(376、recipe 2 つの編集。`codex-seat-permissions.md` の `docs/evidence/` の引用は 4→3 で、行の最後の注の「4 citations of 4 files」が古くなった)、「RC1 (org recheck 2026-10-24)」(377、reviewer の雛形)、「RC4」(382、leader の機能ごとの org の手順 9、12、14)、「After step 14 of the leader template」(389、手順 14 の PR の本文と RESULT の例)、「Five files went past the 800-line guide」(420、`internal/config/config.go` 1219→1221、コメントだけ)、「The section 「agmsg の受信箱の運用」」(528、立て直しの規則の座席の確認の項) | `docs/tech-debt/README.md` の各行の 4 列目。evidence の 9 | /sync-docs で、各行に「2026-11-22: fix/org-xreview-own-model (#360) hit this trigger with ...、but did not pay it」の注を足す(払うものはない。どれも Scope の外) |
| V3 | LOW | 「A Codex reviewer that runs the gate cannot be reached」(343)の 2026-11-01 の注は、Codex の reviewer が cross-review で相手の呼び出しを前面で最大 20 分走らせ、leader の雛形と 2 つの skill がその停滞を gate のときと同じに扱う、と行を広げた。この branch でその呼び出しはなくなり、`f95a08f9` で cross-review の停滞は gate と同じには扱わなくなったので、広げた分が今の動きと合わない。Trigger には当たらない | `docs/tech-debt/README.md:343` の 4 列目 | /sync-docs で、広げた分が 2026-11-22 になくなったことを 1 文足す |
| V4 | LOW | self-review の L2 が残っている。`pick_reviewer` の行(31)の 2 列目が「The fourth caller, a feature org reviewer seat's cross-review」を説明し、4 列目も呼び出し元に数える。`internal/cli/org_runtime_env_test.go:182-185` のコメントは「a reviewer seat's cross-review takes the implementer's driver from its TASK」と言う | `docs/tech-debt/README.md:31`、`internal/cli/org_runtime_env_test.go:182-185` | 計画の S4 のとおり /sync-docs で直す。行 31 は 4 つ目の呼び出し元の文を消すか、2026-11-22 に消えたと書く |
| V5 | INFO | 「Seven test gaps of org-cross-review」(432)の 4 列目で、この branch の 2026-11-22 の注が、取り込んだ #366 の 2026-11-21 の注より前に並ぶ。取り込みの衝突をこの順で解いたため。中身は両方そろっている | `docs/tech-debt/README.md:432` | 直すなら /sync-docs で 2 つの注を日付順に入れ替える |
| V6 | INFO | 計画の Progress に、self-review のあとの直し `f95a08f9`(M1、L1、L3、L4)と、origin/main の取り込み `b98b1b96`(#362 と #366、台帳の 1 行の衝突)の記録がない。Risks の「main を取り込む」の扱いは、「順番の入れ替え」の項が予定として書くだけ | `docs/plans/active/2026-11-22-org-xreview-own-model.md:125-128` | Progress に 1 行足す |
| V7 | INFO | AC6 の「(差分に入らない)」と Verify plan の「`git diff main...HEAD -- .claude/skills/cross-review scripts/xreview-helpers.sh` が空」は、字面では満たさない。差分はコメントだけで、振る舞いは変わらない(上の AC6 の行) | evidence の 2 | AC の文は変えない(変えると digest が変わる)。PR の本文で、AC6 はコメントだけの差分で振る舞いを変えずに満たしたと書く |
| V8 | INFO | AC1 の「`codex exec`、`claude -p` … を持たない」は、Scope の「`codex exec`、`claude -p`、そのほかの CLI で別のモデルを呼ばないと明記する」と字面でぶつかる。雛形はその 1 文だけに 2 語を持ち、テスト(`reviewerXreviewGone`)もその 1 文を除いてから探す | `reviewer.md:171-172`、`internal/org/prompts_xreview_reviewer_test.go:31-50` | なし(Scope が優先する読み方で満たす) |

## Observational checks

- 実機の座席での cross-review は走らせていない。Verify plan の「描いた雛形の該当の節(テストの出力)」は `/test` の出力に任せる。ここでは雛形の文を読み、Python のポート(空白をつぶし、許した 2 文を除いてから語を探す)で消した語の不在を確かめた
- `Seven test gaps` の行と `has not run inside a seat` の行の 2026-11-22 の注は、何が閉じて何が残るかを書いている。後者の Impact の列は古い文のまま残るが、注が「the impact's sentence ... no longer apply」と書く

## Coverage gaps

- テスト(`go test ./internal/org/ ./internal/cli/`、`tests/test-xreview-helpers.sh`、`tests/test-agent-models.sh`)は走らせていない。`/test` の仕事
- 座席の中で、新しい雛形の reviewer が差分を読んで report を書く流れ、新しい leader と前の版の reviewer が混ざったときの QUESTION、前の版の leader の 5 行の TASK を新しい reviewer が読まずに進むことは、文とテストの文字列でしか確かめていない(計画の Assumptions のとおり、release ノートで reseat を勧める)
- 台帳の Trigger の照合は、Trigger の列の語とこの差分のファイルと関数の名前を突き合わせたもので、文の意味から当たるかを決めた行が 3 つある(161 の「org の品質パイプライン」、370 の「leader の雛形のテスト」、528 の「座席の確認の項」)。「`placeSeatPane` (`internal/org/spawn.go`) is 89 lines」(448)の「the spawn saga」は、`Spawn` の中の雛形の vars の 1 行で当たるとは読まなかった。「Small findings of org-feature-worktree」(202)の (g) は閉じた項目で、数えなかった
- 足すとよい決定論の確かめ: `ralph init` の問いの説明と、それが書く設定の意味を突き合わせるものはない。V1 のような文のずれは、`initSettingsQuestions` の説明を README か仕様の 1 文と比べるテストで拾える

## Verdict

- Verified: AC1〜AC5、AC7、AC8(ソースと文書で)。AC6 は振る舞いで(コメントだけの差分、本体は main と同じ)。静的解析は full の範囲で pass。計画の digest は一致。skill と recipe と `xreview-helpers.sh` の写しは一致。#366 の人の決定の文と cross-review の小節は食い違わない。merge の衝突の解き方は両方の変更を保つ
- Partially verified: AC6 の字面(V7)。文書は、`ralph init` の問い(V1)と台帳の注(V2〜V4)を除いて合う
- Not verified: テストの実行と、座席の中での cross-review の実行
- Verdict: pass(MEDIUM 1、LOW 3。止める指摘はない。V1 は PR の前に直すか tech-debt に送るかを決め、V2〜V4 は /sync-docs に渡す)
