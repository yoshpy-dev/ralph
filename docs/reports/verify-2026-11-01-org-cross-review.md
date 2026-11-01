# Verify report: org-cross-review

- Date: 2026-11-01
- Plan: `docs/plans/active/2026-11-01-org-cross-review.md`(issue #269、AC1〜AC9 と AC5b、D1〜D10 と D6b・D7b)
- Verifier: verifier サブエージェント(`/verify`、パイプラインの cycle 1。`cycle-count.json` は `"cycle": 1`)
- Scope: `git diff origin/main...HEAD`(origin/main 4fef6b8c、HEAD d8e02c1d、38 ファイル、+4507/−195)。spec の準拠、静的解析、文書のずれを見た。`go test` と `./scripts/run-test.sh` は /test の担当なので打っていない。`sh tests/test-xreview-helpers.sh` は振る舞いのテストだが、lead の handoff が静的の検査の一覧に名指したので打った
- Evidence: `docs/evidence/verify-2026-11-01-org-cross-review.md`(静的検査の要約、render の確かめ、状態のコマンドと reviewer のコマンドを stub で打った結果)。run-static-verify 自身のログは `docs/evidence/verify-2026-11-01-001525.log`(`.gitignore` の `docs/evidence/*.log` に当たるのでコミットしない)

## Spec compliance

| Acceptance criterion | Status | Evidence |
| --- | --- | --- |
| AC1 off(false、古い ralph、ralph なし)なら TASK を送らず、PR の本文の `Cross-review:` に `skipped (<理由>)` | met | 状態のコマンドの `switch:` の行 `internal/org/prompts/leader.md:798-799`、扱い `:815-828`(off は TASK を送らず `Cross-review: skipped (<理由>)` を控えて sync-docs へ)、PR の本文 `:603-610`、最後の値 `:1019-1025`。stub の probe で、rc 0 の `false` は `switch: off (off: [org.review].cross_review)`、127 は `switch: off (ralph not found; treated as off)`、rc 1 で `unknown command "config"` は `switch: off (ralph without config get; treated as off)` を出し、ほかの rc 1・rc 2・rc 0 の `maybe` は `switch: error` になった(evidence の A〜F) |
| AC2 on なら手順 9 の rebase のあと・sync-docs の前に `xreview-` の TASK。driver、base、回数、report のパス、reviewer の model と effort を持つ | met | 手順 9 の段落 `leader.md:351-360`(rebase のあと)、sync-docs の頭 `:362`。TASK_ID `:937-944`、本文の行 `:947-959`(`IMPLEMENTER_DRIVER:`、`XREVIEW_BASE:`、`CYCLE:`、`REPORT:`、`REVIEWER_CODEX:`、`REVIEWER_CLAUDE:`)、例 `:966-977`。手順の番号は振り直していない(見出しの比較で増えたのは `### cross-review` だけ) |
| AC3 reviewer は `pick_reviewer <implementer の driver>` で違う系列を選び、読むだけの形で呼ぶ(スタブのテストで両方向) | met | 手順 2 の 1 行 `internal/org/prompts/reviewer.md:189`(driver を一重引用符で確かめ、`pick_reviewer "<IMPLEMENTER_DRIVER>"`)、理由 `:177-186`。codex `:223-251`(`-c sandbox_mode=read-only exec review --ignore-rules`、`--disable plugins --disable apps`、MCP を切る)、claude `:253-280`(diff のファイル、`--restricted --strict-mcp-config --disallowedTools "mcp__*" --tools Read,Grep,Glob --permission-mode dontAsk`、`RALPH_WIP_CHECKPOINT=off`)。スタブのテスト `TestRenderRolePrompt_Reviewer_XreviewCallsTheOtherModel`(`internal/org/prompts_xreview_reviewer_test.go:837`)は pane の `RALPH_PRIMARY_CLI` を逆の値にして両方向を確かめる。手で打った probe でも、driver `codex` と pane の `RALPH_PRIMARY_CLI=claude` で `reviewer=claude`、driver `claude` で `reviewer=codex` になった(evidence の R1〜R2b)。テストが通るかは /test |
| AC4 report を `review-xreview-<date>-<org>-c<n>.md` に書き、ACTION_REQUIRED があれば BLOCKED、なければ RESULT。呼べなければ skipped、終わらなければ incomplete | met | report の形 `reviewer.md:349-391`、返事 `:393-400`、skipped `:195-206`・`:289-292`、incomplete `:172-176`・`:293-320`。probe で、CLI がなければ `available rc=1`、出力が空なら `out: empty`、見張りに止められると `codex rc=143` になった(evidence の R5〜R7、X2、X4、Y2) |
| AC5 直すものを implementer に頼みレビューから回し直す。complete の回が上限に届き ACTION_REQUIRED が残れば 3 択の QUESTION。人が上げた上限を `.harness/state/xreview-<org>-cap` に控える | met | 回し直し `leader.md:997-1005`、上限の QUESTION `:875-879`・`:889-915`、a の cap のファイル `:917-930`、b `:931-933`、c `:934-935`。cap の読み方 `:800-802`・`:829-835`(設定とファイルの 1 行目の大きいほう)。probe で、ファイル `3` は `cap: 3`、`03`・`1`・`5 extra` は設定の 2 のまま(evidence の I) |
| AC5b TASK の前に上限を確かめ、上限の回の pass のあとに変われば送らずに聞く | met | `leader.md:871-887`(`done:` が `cap:` 以上なら送らない。`since: changed` で 2 つ目の QUESTION)、`since:` の決め方 `:858-869`。probe で、complete の c3(AR=0)で done 2 = cap 2 のときは `since: same`、そのあと gate の直しのコミットを足すと `since: changed` になった(evidence の AC5b-1・AC5b-2)。HEAD の object が消えたときも `since: changed`(M) |
| AC9 本物の claude と codex で両方向に走らせ、complete の report の形の出力を `docs/evidence/` に残す。座席の中は known gap | met(586f7934 の時点) | `docs/evidence/xreview-live-probe-2026-11-01.md`。A(claude の implementer → codex)は `codex rc=0`・`out: not empty`・`git: same`、所見 1 件 `[P1]`。B(codex の implementer → claude)は `claude rc=0`・`out: not empty`・`git: same`、所見 4 件。座席の中で走らせていないことは `:9` と結論 `:143` に書いてある。利用者名や secret の形の文字列は grep で 0 件。記録は 586f7934 の版のコマンドで、そのあと reviewer.md の節は aa0a0fd5 などで変わった(Coverage gaps) |
| AC6 cross-review の TASK の間の reviewer の停滞の ALERT は gate と同じ扱い(雛形と director・org の skill) | met | `leader.md:504-512`(手順 12 の段落に cross-review の待ちを足した)、`:979-985`。`.claude/skills/director/SKILL.md:697-701`、`.claude/skills/org/SKILL.md:1236-1238`。写し 3 つずつは `cmp` で同じ |
| AC7 雛形のテスト、`check-skill-sync.sh`・`check-sync.sh`・`check-template-purity.sh`、文の固定のテスト | partly | 静的の 3 つは通った(下の表)。新しいテスト 3 本(`prompts_xreview_leader_test.go` 1,144 行、`prompts_xreview_reviewer_test.go` 1,127 行、`skill_xreview_test.go` 485 行)がある。既存のテスト 3 本の固定の文は新しい文に合わせて書き換えてあり(`prompts_test.go`、`prompts_leader_syncdocs_test.go`、`skill_doc_followup_test.go`)、意図は弱めていない。テストが通るかは /test が確かめる |
| AC8 設定と skill の文書が、leader が設定を読み reviewer が implementer の driver で相手を選ぶ形を書く | met | `internal/config/config.go:674-701`、`templates/base/ralph.toml:330-347`、`/org` skill `.claude/skills/org/SKILL.md:33-36`・`:50-58`・`:1074-1104`、仕様 `docs/specs/2026-10-07-org-multi-org-director.md:133`、README `README.md:278`・`:284`、`docs/reports/README.md:23`(写しと同じ) |

### Design decisions

| 決定 | 状態 | Evidence |
| --- | --- | --- |
| D1 設定は手順 9 で leader が config get で読む。終了コードは計画助言と同じ | met | `leader.md:787-788`、`:798-802`、`:815-828` |
| D2 手順 9 の rebase と sync-docs の間に 1 段落、詳しくは `### 計画助言` のあとの小節。番号は振り直さない | met | `leader.md:351-360`、`### cross-review` は `:780`(`### 計画助言` `:654` のあと)。origin/main との見出しの比較で増えたのは `### cross-review` だけ |
| D3 数えるのは complete の report だけ。skipped と incomplete も report を書き 0 の `After triage:` を残す | met | `leader.md:844-857`、`reviewer.md:353-355`。probe K で skipped の c2 は `done:` に入らず `k:` に入った |
| D4 report の名前を `review-` で始める | met | `scripts/check-sync.sh:80` の除外 `docs/reports/review-`、`internal/insights/backfill.go:69-80` は `review-` を読まない |
| D5 TASK の形、ミッション 1 の除外、返事の `XREVIEW:`、`git_changed`、欠けた行の QUESTION、`REVIEWER_*` がなければ incomplete | met | ミッション 1 `reviewer.md:17-21`、手順 1 `:165-176`、`git_changed` `:283-288` と leader の扱い `leader.md:1006-1012`、QUESTION の扱い `:1013-1017` |
| D6 相手の呼び方(codex は read-only の `exec review`、claude は diff のファイルを読むだけ、20 分の見張り、git の前後の比較、rc 0 かつ出力が空でない) | met | `reviewer.md:207-280`、`:298-301`。probe X1・Y1 |
| D6b `--disable apps`(leader の計画助言にも)、incomplete の理由の形、driver の確かめ、helper がないときの skipped、`sh -c 'command -v'`、claude の指示文、P0〜P3 の読み替え | met | `reviewer.md:229-247`、`:302-320`、`:189`、`:195-199`、`:263-268`、`:323-326`。leader の計画助言 `leader.md:720-738` |
| D7 上限での QUESTION、cap のファイル、送る前の確かめ、`HEAD:` の行 | met | `leader.md:871-935`、`reviewer.md:349-352`(`HEAD:` は相手に見せた HEAD) |
| D7b `k:` は番号の最大 + 1、`-r2`・`-r3`、`docs/reports/` の外で `since:` を決める、上限の回の ACTION_REQUIRED はそのまま QUESTION、`XREVIEW: cap_reached` と `FORK 1:`、c は何もしない | met | `leader.md:853-857`、`:937-944`、`:858-869`、`:999-1002`、`:889-935`。probe J で前の org の c9 があると `k: 10` |
| D8 report は leader がコミットしてから sync-docs | met | `leader.md:352-360`、`:987-990` |
| D9 停滞の ALERT を gate と同じく | met | AC6 と同じ |
| D10 PR の本文の `Cross-review:` の行と、off のときの `skipped (off: [org.review].cross_review)` | met | `leader.md:603-613`、`:799`、`:1019-1025`。RESULT の `CROSS_REVIEW:` `:627-628`、`/director` の報告 `.claude/skills/director/SKILL.md:663-668` |

## Static analysis

| Command | Result | Notes |
| --- | --- | --- |
| `./scripts/run-static-verify.sh`(changed scope) | pass(rc=0) | `All verifiers passed.`。`scripts/xreview-helpers.sh` が分類されないので full fallback になり golang が走った。shellcheck、`sh -n`、guard の awk、`jq -e`、Codex hook の検査 3 つ、check-sync、check-pipeline-sync、check-skill-sync、check-template-purity、tech-debt の plan の参照、gofmt、golangci-lint(`0 issues.`)、secret scan(`4fef6b8c..d8e02c1d`、clean) |
| `./scripts/check-template-purity.sh` | pass(rc=0) | `PASS: no meta-repo-specific references found in templates.` |
| `./scripts/check-skill-sync.sh` | pass(rc=0) | 14 skill が一致 |
| `./scripts/check-sync.sh` | pass(rc=0) | IDENTICAL 172、DRIFTED 0、ROOT_ONLY 0 |
| `sh tests/test-xreview-helpers.sh` | pass(rc=0) | PASS 53、FAIL 0。1f「explicit arg wins over env」がある |
| `git diff --check origin/main...HEAD` | pass(rc=0、出力なし) | |
| `go build ./...` / `go vet`(`internal/org/`、`internal/org/driver/`、`internal/cli/`、`internal/config/`)/ `gofmt -l internal cmd` | pass | どれも出力なし |
| `./scripts/plan-visual.sh digest <plan>` | `7789ae616bd0` | 計画の `- Approved:` の digest と同じ |
| 写しの `cmp` | 同じ | director と org の skill の写し 3 つずつ、`scripts/xreview-helpers.sh`、`docs/reports/README.md`、2 つの recipe |

## Documentation drift

| Doc / contract | In sync? | Notes |
| --- | --- | --- |
| `internal/config/config.go:674-701` | yes | leader だけが使うときに読み、reviewer は TASK の driver で選ぶ。complete の回だけを数える |
| `templates/base/ralph.toml:330-347` | yes | 同じ中身。「the reviewer seat starts reading ... in a later release」の古い文は消えた |
| 仕様の注 `docs/specs/2026-10-07-org-multi-org-director.md:133` | yes | 雛形と合う。「数はコミット済みの report から読む」はこのブランチが足した report に限ることまでは書かないが、詳細は計画を指しているので info |
| README `:278`、`:284` | yes | `pick_reviewer` は driver がないときだけ変数を読む(`scripts/xreview-helpers.sh:109-116` と合う)。`xreview-<org_id>-c<k>`、`XREVIEW: cap_reached`、`Cross-review:` の 3 つの値 |
| AGENTS.md | yes | 機能の org の手順を並べない。触っていない |
| `docs/reports/README.md:23` と写し | yes | `review-xreview-YYYY-MM-DD-<org_id>-c<k>.md` |
| `docs/recipes/codex-seat-permissions.md:147-164` と写し | yes | network の注。座席の中は未確認と書く |
| `docs/recipes/isolated-herdr-server.md:250-255` と写し、`internal/org/driver/paneenv.go:41-47`・`:72-76`、`internal/org/spawn.go:996-1006` | yes | 警告文は leader の計画助言のことだけになった。「seat's cross-review may pick」「treats the seat's driver as claude」は過去の report のほかに残っていない(`git grep`) |
| `scripts/xreview-helpers.sh:87-105` と写し | yes | 4 つ目の呼び手(reviewer 座席)が引数を渡すこと |
| `/director` skill(4 つ)の上限の件 `:845-905` | yes | `FORK 1: a 上限を上げる(新しい上限 3)` の形、新しい上限の決め方(report の `- Cycle:` の `<n>` と leader の `done:` は同じ数になる)、report は機能の worktree `.claude/worktrees/org-<org_id>`(`internal/org/feature.go:163`)で読む |
| `/org` skill(4 つ) | yes | 前提 2 か所、終わり方の 1・4・6・7、締めの順 `:2046` |
| `docs/tech-debt/README.md:416` | yes | 行数は HEAD で数え直して同じ(leader.md 1,256、reviewer.md 526、テスト 1,144 と 1,127。origin/main は 975 と 224) |
| `#269` への前向きの参照 | なし | 仕様の `:131`(#265 の日付の付いた注)だけで、`:133` の注が受けている |
| 計画の Affected areas | info | `docs/recipes/codex-seat-permissions.md` と写し(S4 と Risks に名前がある)、`docs/tech-debt/README.md`(Risks の「tech-debt で扱う」)、`internal/org/skill_doc_followup_test.go`(固定の 1 行。「など」に入る)は Affected areas に名前がない。中身のずれではない |
| 計画の AC の checkbox | 未記入 | 指示どおり印は付けていない |

## Observational checks

- render: `go run -overlay` で `org.RenderRolePrompt` を `OrgID: "org-a"` で呼び、leader と reviewer を render した。`{{` は 0 件。下の probe で使った 4 つのブロックは render した文にそのまま入っている。worktree の `git status --porcelain` は空のまま
- 状態のコマンド(`leader.md:797-810`)と reviewer のコマンド(`reviewer.md:189`、`:242-250`、`:274-279`)を、scratch の git repo で stub の `ralph`・`codex`・`claude` を使い、`/bin/sh` と `zsh -f` で打った。2 つの shell で出力は同じ(SHA とパスのほか)。結果は AC の表と evidence の表のとおり
- bash の `/bin/sh` では、見張りを止めたときのジョブの通知(`Terminated: 15`)が出力の行のあいだに入る。判定の行は別の行なので読み分けは変わらない(S1c で直したテストの話と同じもの)

## Findings

### V1(LOW): reviewer の雛形の返事の例の TASK_ID が leader の送る形と違う

`internal/org/prompts/reviewer.md:406` と `:418` の RESULT と BLOCKED の例は `TASK_ID: xreview-1` を使う。leader は `xreview-{{ORG_ID}}-c<k>` で送り(`leader.md:937`、例 `:968`)、reviewer は受けた TASK_ID で返す(`reviewer.md:393`)ので、動きは正しい。例の `REPORT:` は `-c1.md` なので、読み手には `xreview-{{ORG_ID}}-c1` の返事に見えるほうが自然。`prompts_xreview_reviewer_test.go:509` が `xreview-1` を固定しているので、直すならテストも合わせる。

### V2(LOW): 状態のコマンドが report の `- HEAD:` の値を形を確かめずに git の引数にする

`leader.md:810` は report の `- HEAD:` の値 `h` を `git diff --quiet "$h" HEAD -- . ':(exclude)docs/reports/'` に渡す。値は reviewer が書く report から来る(雛形はほかの場所で座席の文をデータとして扱う)。`-` で始まる値は git の option になる。scratch で `h` を `--output=<file>` にすると rc=0 で 0 バイトのファイルができ、`since: same` が出た(evidence の「report の `- HEAD:` を option として読む場合」)。上限での「見せていない変更」の判定(AC5b)が、その場合だけ安全でない側に倒れる。reviewer は `-git-before.txt` の 1 行目(`git rev-parse HEAD`)を書くので、ふつうの運用では起きない。直すなら、`case "$h" in ''|*[!0-9a-f]*) echo 'since: changed' ;; ...` のように 16 進でなければ `changed` にする。reviewer の `git diff "<XREVIEW_BASE>...HEAD"`(`reviewer.md:276`)も、`XREVIEW_BASE:` の形(`origin/` で始まるか)は確かめていない(値は leader が作る)。

### Info

- 状態のコマンドで、`switch: off` と同時に `cap: error` か `base: error` が出たときにどちらを先に読むかは、文に書いていない(`leader.md:815-843`)。probe では rc 0 の `false` と cap の rc 2 の組で両方の行が出る。2 つの設定は同じ binary(#265)から読むので、ふつうは片方だけが落ちることはない
- report の `HEAD:` が存在しない object のときと、rebase で upstream が入ったときは `since: changed` になる。計画の D7b のとおり

## Coverage gaps

- AC7 のテストの結果と、計画の Test plan の変異(相手を driver と同じにする、off で TASK を送る、cap のファイルを読まない)が落ちることは /test が確かめる
- AC9 の live の記録は 586f7934 の版のコマンドで、そのあと reviewer.md の節は 63 行増え 24 行減った(aa0a0fd5 の `--disable apps`、driver の `case`、`sh -c 'command -v claude'`、`log:` の行など)。今の版のコマンドを本物の CLI で通しで走らせた記録はない。stub の probe とテストは今の版の形を確かめる。下の「ずれ」の偶然の呼び出しで、codex 0.163.0 が今の版の引数(`--disable plugins --disable apps` の付いた `mcp list` と `exec review`)を受け付けて API まで届くことは見えたが、モデルを `gpt-x` にしていたので review は走っていない
- 座席の中での実行(codex の reviewer 座席の `--sandbox workspace-write` での network、pane の環境)は、計画のとおり known gap
- leader と director が実機で上限の QUESTION を往復することは確かめていない

## Deviations

- 最初の zsh の probe を `zsh -f` でなく `/bin/zsh` で打ったため、zsh の起動ファイルが PATH の先頭を戻し、stub ではなく本物の `ralph`、`codex`(4 回)、`claude`(2 回)が scratch の repo で呼ばれた。codex の 3 回はモデル `gpt-x` を API に断られ、見張りを 2 秒にした 1 回は止められて rc=0(雛形の「見張りに止められても 0 を返すことがある」と同じ)。claude には stub 用の 1 語 `PROMPT` が渡り、読む道具だけで scratch の repo を見て「指示がない」と返した。送ったのは scratch の repo の数行の差分だけで、worktree の外にも中にも書いていない。上の結果は `zsh -f` で打ち直したもの。詳細は evidence の最後の節
- `sh tests/test-xreview-helpers.sh` は handoff の指示で打った

## Insight event

`./scripts/insights-append.sh --slug org-cross-review --flow standard --phase verify --verdict pass --cycle auto --source skill --low 2` を、この report を書く前に打った。UTC の日付が 2026-11-01 に変わっていたので、self-review の event のある `docs/insights/events/2026-10-31-org-cross-review.jsonl` ではなく、新しい `docs/insights/events/2026-11-01-org-cross-review.jsonl` に書かれた(`"cycle":1`、`"low":2`)。スクリプトが作ったままコミットする。

## Verdict

pass

- Verified: AC1・AC2・AC3・AC4・AC5・AC5b・AC6・AC8 と D1〜D10(D6b・D7b を含む)は、雛形と skill と文書の該当行にあり、状態のコマンドと reviewer のコマンドを stub で打った結果も文のとおりだった。静的の検査はすべて通り、計画の digest は承認時の `7789ae616bd0` のまま
- Partially verified: AC7(静的の部分だけ。テストは /test)、AC9(586f7934 の版で記録済み。今の版の live の通しはない)
- Not verified: Go のテストの結果、座席の中での相手の CLI の呼び出し、leader と director の実機の往復
- Findings: LOW 2 件(V1 例の TASK_ID、V2 `HEAD:` の値の形)。どちらも merge を止めない
