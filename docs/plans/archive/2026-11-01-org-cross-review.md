# org-cross-review

- Status: Approved
- Approved: 2026-11-02 sha256:ce18b629320e
- Owner: Claude Code
- Date: 2026-11-01
- Related request: org: reviewer 座席が相手のモデルで cross-review する(設定で任意)
- Related issue: 269
- Type: feat
- Branch: feat/org-cross-review

## Objective

機能の org で `[org.review].cross_review` が on のとき、leader が reviewer のレビューのあと・sync-docs の前に、reviewer 座席に cross-review の TASK を送る。reviewer は implementer と違う系列のモデルに差分を見せ、triage した report を書いて返す。leader が裁定し、直すものを implementer に頼み、回数を `[org.review].max_cross_review_cycles` までに抑え、上限で残れば人に聞く。

## Scope

- `internal/org/prompts/reviewer.md`: cross-review の TASK(TASK_ID が `xreview-` で始まる)の節。相手の選び方、呼び方、triage、report、返事の形
- `internal/org/prompts/leader.md`: 手順 9 の rebase と sync-docs の間の段落、`### cross-review` の小節(設定の読み方、回数と上限、TASK の形、裁定と直しのループ、上限の QUESTION、飛ばしたときの記録)、ミッション 5 の xreview の BLOCKED、手順 12 と同じ停滞の扱い、手順 14 の PR の本文
- `/director` と `/org` の skill(と写し): 停滞の ALERT の扱い、上限の QUESTION の扱い、「後の版から」の文の置き換え
- 文書: `internal/config/config.go` のコメント、`templates/base/ralph.toml` のコメント、仕様の注、README、`docs/reports/README.md` とその写し
- テスト: 雛形の文と順を固定するテスト、reviewer と leader のコマンドをスタブで走らせるテスト、skill の文のテスト

## Non-goals

- 標準フローの `/cross-review` の変更
- `ralph` の Go のコマンドや設定のキーの追加(`[org.review]` のキーは #265 で入っている)
- leader の task に cross-review の設定を書くこと(D1)
- codex の reviewer 座席の network の権限を変えること(Risks に書く)
- insights の event(flow が standard 前提のため)

## Assumptions

- `ralph config get org.review.cross_review` と `org.review.max_cross_review_cycles` は #265 の binary にある
- leader の task には `- reviewer の codex:` と `- reviewer の claude:` の行がある(#291)。ない古い org は incomplete として飛ばす
- implementer の driver は、leader が TASK を作るたびに台帳の記録(`ralph org status --json` の implementer の座席の driver)から読む(手順 5 で spawn に付けた値の記憶には頼らない。F6)
- reviewer 座席は `docs/reports/` と `.harness/state/` に書ける

## Affected areas

- `internal/org/prompts/reviewer.md`、`internal/org/prompts/leader.md`
- `internal/org/prompts_test.go`、`internal/org/prompts_leader_syncdocs_test.go`、`internal/org/prompts_leader_plan_test.go`、新しい `internal/org/prompts_xreview_reviewer_test.go` と `internal/org/prompts_xreview_leader_test.go`(reviewer と leader の cross-review を固定)、新しい `internal/org/skill_xreview_test.go`(skill の文)
- `internal/org/driver/paneenv.go`(`RALPH_PRIMARY_CLI` がないときの警告の文)、`internal/org/spawn.go`(コメント)、`internal/org/driver/herdr_test.go`、`internal/cli/org_runtime_env_test.go`、`docs/recipes/isolated-herdr-server.md` と写し、`scripts/xreview-helpers.sh` と写し(コメント)、`docs/evidence/xreview-live-probe-2026-11-01.md`、`docs/recipes/codex-seat-permissions.md` と写し(network の注)、`docs/tech-debt/README.md`
- `.claude/skills/director/SKILL.md`、`.claude/skills/org/SKILL.md` と、それぞれの `.agents/skills/`・`templates/base/` の写し、`internal/org/director_skill_sync_test.go` など skill の文のテスト
- `internal/config/config.go`(コメント)、`templates/base/ralph.toml`(コメント)、`docs/specs/2026-10-07-org-multi-org-director.md`、`README.md`、`docs/reports/README.md` と `templates/base/docs/reports/README.md`

## Visual review

- Page: `.harness/state/plan-visual/org-cross-review.html`(やさしい図 `overview`・`easy-cap`、「承認の前に」の枠、折りたたみの中に `detail-overview`・`detail-leader-order`・`detail-reviewer-run`・`detail-cycles`)
- Lint: `./scripts/plan-visual.sh lint` が exit 0
- Self-check: full と各図を 1 枚ずつ撮って読んだ。直したのは、`detail-cycles` のラベルが TASK の枠に食い込んでいたこと(2 行に分けた)と、各図の下の余白。codex の計画助言の MEDIUM 1(TASK の前に上限を確かめる)を `easy-cap`・`detail-cycles`・`detail-leader-order` と「承認の前に」の枠に入れた

## Design decisions

- D1 設定の読み方: leader が手順 9 の差し込み位置で、worktree で `ralph config get org.review.cross_review` と `ralph config get org.review.max_cross_review_cycles` を打つ。終了コードの扱いは計画助言の小節と同じ(0 は値、1 で `unknown command "config"` と 127 は off とみなす、それ以外は BLOCKED)。`ralph org start --plan` が task に書く案は採らない。計画助言の switch も使う時点で読んでおり、そろえるため(consult)
- D2 差し込み位置: 手順 9 の rebase の段落と sync-docs の段落の間に 1 段落を置き、番号は振り直さない。詳しい手順は `### 計画助言` のあとの `### cross-review` の小節に書く。rebase のあとなので、相手は rebase 済みの差分を見る。#268 の計画の D1 がこの場所を空けている
- D3 回数の数え方(consult): 上限に数えるのは、相手が所見を返して triage まで終わった回だけ。数は、ブランチにコミット済みの cross-review の report のうち `- Reviewer status: complete` の行を持つものの数で、今回は その数 +1 回目。reviewer は skipped や incomplete のときも report を書き、`- After triage: ACTION_REQUIRED=0, WORTH_CONSIDERING=0, DISMISSED=0` の行を残す(標準フローの雛形と同じ形)。report の名前の連番 `c<n>` は名前の一意性のためで、既存の cross-review の report の数 +1。gate の失敗などで手順 8 → 9 に戻っても、complete の回が増えていなければ初回のまま
- D4 report の名前: `docs/reports/review-xreview-<date>-{{ORG_ID}}-c<n>.md`。`review-` で始めるので、`scripts/check-sync.sh` の除外と insights の backfill(`cross-review-triage-` を標準フローとして読む)に触れない
- D5 TASK の形: TASK_ID は `xreview-` で始める。reviewer のミッション 1 はレビューの TASK から `gate-` と `xreview-` を除く。本文の行は `IMPLEMENTER_DRIVER: <claude|codex>`、`XREVIEW_BASE: origin/<既定のブランチ>`(RESULT のヘッダ `BASE:` と名前を分ける)、`CYCLE: <n>/<上限>`、`REPORT: <report のパス>`、task の reviewer の codex と claude の model と effort。reviewer の返事は本文に `XREVIEW: pass | action_required | skipped (<理由>) | incomplete (<理由>) | git_changed` と `REPORT:` を書き、ACTION_REQUIRED があれば BLOCKED、なければ RESULT。相手を呼ぶあいだに git の状態が変わったら `XREVIEW: git_changed` の BLOCKED で、leader は計画助言の `git: changed` と同じく escalate で止める。`IMPLEMENTER_DRIVER:`・`XREVIEW_BASE:`・`CYCLE:`・`REPORT:` のどれかが欠けたら、reviewer は誰も呼ばずに QUESTION で leader に聞き、leader は行をそろえて新しい TASK_ID で送り直す(gate の TASK の QUESTION と同じ形)。incomplete にするのは `REVIEWER_*` の行が欠けたとき(task に reviewer の値がない古い org)だけ(S1 の consult)
- D6 相手の呼び方: reviewer は `pick_reviewer <IMPLEMENTER_DRIVER>` で相手を決める(引数が env の `RALPH_PRIMARY_CLI` より先)。codex は `-c sandbox_mode=read-only exec review --ignore-rules --base <XREVIEW_BASE>` に、計画助言と同じ MCP の止め方を足す。claude は、reviewer が `git diff <XREVIEW_BASE>...HEAD` を `.harness/state/xreview-{{ORG_ID}}-c<n>.diff` に書き、計画助言と同じ読むだけの形(`--restricted --strict-mcp-config --disallowedTools "mcp__*" --tools Read,Grep,Glob --permission-mode dontAsk`、`RALPH_WIP_CHECKPOINT=off`)で、そのファイルと worktree を読ませる。どちらも 20 分の watchdog、前後の git の状態の比較、完了は rc 0 かつ出力のファイルが空でないとき。相手の CLI がなければ skipped、終わらなければ incomplete
- D7 裁定とループ(consult): leader は report を読み、ACTION_REQUIRED のうち直すものを implementer に TASK で頼む(per-plan では `PLAN:`)。直さないものは理由を PR の本文に書く。直したら手順 8 のレビューから回し直し、手順 9 で次の回の cross-review を頼む。complete の回が上限に届いて ACTION_REQUIRED が残れば、`ralph org escalate` で QUESTION を上げる(1 上限を上げる、2 known gaps として PR に書いて進む、3 止める)。人が上限を上げたら、leader は `.harness/state/xreview-{{ORG_ID}}-cap` に数字を 1 つ書き、上限は config get の値とこのファイルの値の大きいほうにする(標準フローの `cycle-count.json` の `cap` と同じ考え。agmsg の DECISION は compaction のあと読み直せないため)。WORTH_CONSIDERING は leader が判断し、PR の本文に残す。上限は TASK を送る前に確かめる(codex の計画助言 MEDIUM 1): complete の回が上限に達していれば TASK を送らない。最後の complete の回に ACTION_REQUIRED が残っていれば上の QUESTION を上げる。最後の回が pass でも、そのあと gate の失敗などで直しが入り、HEAD がその report の `HEAD:` の行と違えば、やはり TASK を送らず QUESTION を上げる(1 上限を上げて見せる、2 見せていない直しを PR に書いて進む、3 止める)。reviewer は report に相手に見せた HEAD を `HEAD:` の行で書く
- D7b 実装中に決めたこと(S2 と consult、2026-11-01): report の連番 `<k>` は既存の番号のいちばん大きいもの +1(番号が飛んでも名前がぶつからない)。同じ `<k>` で送り直す TASK_ID には `-r2`、`-r3` を付ける。上限のあとの「直しが入った」は、最後の complete の report の `HEAD:` から、`docs/reports/` 直下の `.md`(生成された記録)の外が変わったかで決める(leader が report をコミットするので HEAD の比較では毎回違う。`docs/reports/templates/` は skill が読む入力なので数える。cross-review cycle 1 の ACTION_REQUIRED #1、F7)。手順 9 の rebase で upstream が入れば変わったとみなし、上限を 1 回使う。report の `HEAD:` の object が消えて差分を取れなければ、変わったとみなす。上限の回で ACTION_REQUIRED が返ったら implementer には頼まず、そのまま QUESTION に進む。上限の QUESTION は、ヘッダ `XREVIEW: cap_reached` で件を見分け、選択肢は計画の承認の `FORK` の行と同じ形で書き、人の答えは DECISION の `FORK 1: <記号> <案>` の形で受ける(director の DECISION の本文は `SUMMARY:` で始まるので、先頭の数字では選ばない)。推奨は上限を上げる案。「止める」を選ばれたら leader はそれ以上何もせず、org を止めるかどうかは人が決める
- D6b 本物の CLI での確かめ(S1b)で相手のモデルが出した所見と気づいた点(2026-11-01): codex の呼び出しに `--disable apps` を足す(`--disable plugins` では apps の機能が残り、sandbox の外の道具が使えうる。leader の計画助言の codex の呼び出しにも同じ穴があるので合わせて直す)。incomplete の理由は `<cli> rc=<n>: <log の要点の 1 行>` の形にし(claude は最初の空でない行、codex は最後の空でない行。codex の log は 1 行目が版の見出しで、エラーは末尾に出るため。S1c)、rc=143 は watchdog の 20 分の止めと書く。`IMPLEMENTER_DRIVER` の値はコマンドの中でも確かめ、claude と codex 以外なら誰も呼ばない。`scripts/xreview-helpers.sh` がなければ `skipped (scripts/xreview-helpers.sh not found)` にして、CLI がないことと分ける。claude への指示文は、diff のファイルだけが読める前提で書く。CLI があるかは `sh -c 'command -v <cli>'` で確かめる(zsh の alias を CLI があるとみなさない)。codex の `exec review` は組み込みのレビューで動き、重大度を P0〜P3 で返すので、triage は P0・P1 を HIGH、P2 を MEDIUM、P3 を LOW と読む
- D8 report のコミット(consult): cross-review の report は leader がコミットしてから sync-docs に進む(手順 9 の冒頭のコミットは cross-review より前なので、そのままでは sync-docs の前の `git status` の確かめで止まる)
- D9 停滞の ALERT: leader の雛形、`/director` と `/org` の skill の「gate を頼んでいる間」の扱いに、cross-review の TASK を足す
- D10 PR の本文: 手順 14 で、PR の雛形の `Cross-review:` の行に `ran (<相手>)`、`skipped (<理由>)`、`incomplete (<理由>)` のどれかを書き、直さなかった所見を本文に残す。off のときは `skipped (off: [org.review].cross_review)`。reviewer が skipped か incomplete を返した回は、leader が記録して sync-docs に進む(回数には数えず、やり直しも頼まない)
- Critical forks: D1(設定を使う時点で読むか、start で task に書くか)と D6 の claude の呼び方(読むだけか、標準フローと同じ auto か)。ユーザーの「計画の分かれ目は consult に聞いて自分で選ぶ」に従い、consult と相談して上のとおり選んだ

## Acceptance criteria

- [x] AC1 `[org.review].cross_review` が off のとき(config get が false、古い ralph、ralph なし)、leader は cross-review の TASK を送らず、PR の本文の `Cross-review:` に `skipped (<理由>)` を書く
- [x] AC2 on のとき、手順 9 の rebase のあと・sync-docs の前に、leader が `xreview-` の TASK を送る。TASK は implementer の driver、base、回数、report のパス、reviewer の model と effort を持つ
- [x] AC3 reviewer は `pick_reviewer <implementer の driver>` で implementer と違う系列のモデルを選び、読むだけの形で呼ぶ(スタブで走らせるテストで、claude の implementer なら codex、codex なら claude が呼ばれることを確かめる)
- [x] AC4 reviewer は triage した report を `docs/reports/review-xreview-<date>-<org>-c<n>.md` に書き、ACTION_REQUIRED があれば BLOCKED、なければ RESULT を返す。相手を呼べなければ skipped、終わらなければ incomplete の report と RESULT
- [x] AC5 leader は直すものを implementer に頼み、レビューから回し直す。complete の回が上限に届いて ACTION_REQUIRED が残れば、3 つの選択肢の QUESTION で人に聞く。人が上げた上限は `.harness/state/xreview-<org>-cap` に控える
- [x] AC5b leader は TASK を送る前に上限を確かめ、上限の回が pass でもそのあと HEAD が変われば、TASK を送らずに人に聞く(上限の回の pass → gate の失敗 → 直し → 手順 9 に戻る流れで、承認の前に TASK を送らない)
- [x] AC9 雛形から抜き出した reviewer のコマンドを、この機械の本物の claude と codex で両方向(implementer が claude なら codex、codex なら claude)に走らせ、この branch の差分を読んで complete の report の形の出力が出ることを `docs/evidence/` に残す。座席の中(権限・sandbox・network)での実行は known gap とし、PR の本文に書く
- [x] AC6 cross-review の TASK の間の reviewer の停滞の ALERT は、gate と同じく扱う(雛形と director・org の skill)
- [x] AC7 雛形のテスト、skill の写しの検査(`check-skill-sync.sh`、`check-sync.sh`、`check-template-purity.sh`)、文を固定するテストが通る
- [x] AC8 設定と skill の文書(config.go、templates/base/ralph.toml、`/org` skill、仕様、README、docs/reports/README.md)が、leader が設定を読み reviewer が implementer の driver で相手を選ぶ形を書く

## Implementation outline

1. S1 reviewer の雛形: `internal/org/prompts/reviewer.md` に cross-review の TASK の節(ミッション 1 の除外、相手の選び方と呼び方、triage、report、返事)、テスト(文と、スタブの codex・claude で呼び方と相手を確かめる)— verify: `test -z "$(gofmt -l internal/org)" && go vet ./internal/org/ && GOFLAGS="-timeout=30m $(go env GOFLAGS)" go test -count=1 ./internal/org/ -run 'Prompt|Reviewer|Xreview|ReportNames'`
1b. S1b 本物の CLI での確かめ(AC9): S1 の reviewer の雛形からコマンドを抜き出し、この branch の差分に対して本物の claude と codex で両方向に走らせ、出力・終了コード・前後の git の状態を `docs/evidence/xreview-live-probe-2026-11-01.md` に残す(token の値は書かない)— verify: `test -s docs/evidence/xreview-live-probe-2026-11-01.md && git diff --check`
2. S2 leader の雛形: `internal/org/prompts/leader.md` の手順 9 の段落、`### cross-review` の小節、ミッション 5、停滞の扱い、手順 14 の PR の本文、テスト(順の固定に cross-review の位置を足す、off のとき送らない文、上限と QUESTION、cap のファイル、スタブで config get の分岐)— verify: `test -z "$(gofmt -l internal/org)" && go vet ./internal/org/ && GOFLAGS="-timeout=30m $(go env GOFLAGS)" go test -count=1 ./internal/org/ -run 'Prompt|Leader|Xreview|SyncDocs'`
2b. S1c reviewer の直し(D6b): `internal/org/prompts/reviewer.md` と `internal/org/prompts_xreview_reviewer_test.go`(負荷で隣の行に bash のジョブの通知が入って落ちる 2 か所の直しを含む)— verify: S1 と同じ
2c. S2b leader の直し(D6b、D7b): `internal/org/prompts/leader.md`(上限の QUESTION の形、`since:` の注、計画助言の codex の `--disable apps`)と `internal/org/prompts_xreview_leader_test.go`、計画助言のテスト — verify: S2 と同じ
3. S3 skill: `/director` と `/org` の skill(停滞、上限の QUESTION(`XREVIEW: cap_reached` と FORK の行)と答えの返し方、cross-review の説明の置き換え)と写し、skill の文のテスト — verify: `./scripts/check-skill-sync.sh && ./scripts/check-sync.sh && ./scripts/check-template-purity.sh && GOFLAGS="-timeout=30m $(go env GOFLAGS)" go test -count=1 ./internal/org/ -run 'Skill|Director' && git diff --check`
4. S4 文書: `internal/config/config.go` のコメント、`templates/base/ralph.toml` のコメント、仕様の注、README、`docs/reports/README.md` と写し、`docs/recipes/codex-seat-permissions.md` の network の注 — verify: `./scripts/check-sync.sh && ./scripts/check-template-purity.sh && test -z "$(gofmt -l internal/config)" && go vet ./internal/config/ && GOFLAGS="-timeout=30m $(go env GOFLAGS)" go test -count=1 ./internal/config/ ./internal/org/ -run 'Template|Default|ReportNames|Docs' && git diff --check`

## Verify plan

- Static analysis checks: `./scripts/run-static-verify.sh`、`./scripts/check-skill-sync.sh`、`./scripts/check-sync.sh`、`./scripts/check-template-purity.sh`
- Spec compliance criteria to confirm: AC1〜AC8。特に AC3 の相手の選び方(implementer の driver)、AC5 の回数の数え方と cap のファイル、D8 の report のコミットの位置
- Documentation drift to check: `[org.review]` を説明する文(config.go、templates/base/ralph.toml、`/org` skill、README、仕様)、`docs/reports/README.md` の report 名、director の停滞の扱い
- Evidence to capture: verify の report、static verify の出力

## Test plan

- Unit tests: reviewer の xreview の節の文(ミッション 1 の除外、report 名、返事の形、skipped と incomplete)、leader の手順 9 の順(commit → fetch → rebase → cross-review → report のコミット → sync-docs)、`### cross-review` の小節(config get の終了コードの分岐、回数の定義、cap のファイル、QUESTION の 3 つの選択肢、PR の本文の行)
- Integration tests: 雛形のコマンドを抜き出してスタブで走らせる(implementer が claude なら codex、codex なら claude を呼ぶ、相手の CLI がないと skipped、出力が空だと incomplete、git の状態が変わると止める)
- Regression tests: 既存の雛形のテスト(手順の 1 行目、sync-docs の順、計画助言の小節の位置、gate の停滞の文)と skill の写しの検査が通る
- Edge cases: 古い org で task に reviewer の値がない、`IMPLEMENTER_DRIVER` が claude と codex 以外、gate の失敗で手順 9 に戻ったときの回数、上限を上げたあとの回数、上限の回の pass のあとに HEAD が変わったとき(TASK を送らず QUESTION)
- Live probe(AC9): 雛形から抜き出したコマンドを本物の claude と codex で両方向に走らせ、出力と終了コードを `docs/evidence/` に残す
- Evidence to capture: test の report、変異(相手の選び方を driver の逆でなく同じにする、off で TASK を送る、cap のファイルを読まない)が落ちること

## Risks and mitigations

- 推測: codex の reviewer 座席は `--sandbox workspace-write` で動き、network がないと相手の CLI(`claude -p` や入れ子の `codex exec`)を呼べず、毎回 skipped になりうる。実機は未確認。skipped は記録して進むので PR は止まらないが、cross-review が走らないことに気づきにくい(codex の計画助言 MEDIUM 2)。AC9 で座席の外の本物の CLI での実行を確かめ、座席の中の実行は PR の known gaps に書く。reviewer の skipped の理由に呼べなかった CLI とエラーの 1 行を入れ、PR の本文の `Cross-review:` の行で見えるようにする。`docs/recipes/codex-seat-permissions.md` に、cross-review には network が要ることの注を足す
- 上限のあとの直しが相手に見せられないまま PR に入る: TASK を送る前に上限を確かめ、HEAD が変わっていれば人に聞く(D7、AC5b)
- 相手の呼び出しが長く、reviewer の停滞の ALERT が出る: gate と同じ扱いを雛形と skill に書く(D9)
- 回数を git から数えるので、report のコミットが漏れると数え違える: D8 で leader がコミットする位置を固定し、テストで順を固定する
- `leader.md` が 975 行からさらに伸びる: 小節に分け、番号は振り直さない。大きさは tech-debt で扱う
- ルートの `ralph.toml` はすでに `cross_review = true` なので、merge 後はこの repo の機能の org で cross-review が走る: release のノートに書く

## Rollout or rollback notes

- 雛形と skill と文書の変更だけで、記録の形と Go のコマンドは変えない。戻すときはこの PR を revert する
- release のノート: `[org.review].cross_review` を on にした機能の org で、reviewer 座席が相手のモデルで cross-review する。新しい雛形は新しい binary が要る

## Open questions

- なし

## Progress checklist

- [x] Plan reviewed(consult の判定は「直してから進める」。D3 の回数を complete の report の数で数え skipped と incomplete を数えないこと、D7 の人が上げた上限を `.harness/state/xreview-<org>-cap` に控えること、D8 の report のコミットを sync-docs の前に置くこと、本文の行の名前を `XREVIEW_BASE:` にすること、codex 座席の network を Risks に書くことを直した。plan advisory: ran with codex。MEDIUM 2 件: (1) gate のやり直しで上限を越えて cross-review が走る → TASK を送る前に上限を確かめ、上限の回の pass のあとに HEAD が変われば人に聞く(D7、AC5b)、(2) 本物の条件で complete になる確かめがない → 本物の claude と codex で両方向に走らせる S1b と AC9 を足し、座席の中の実行は known gap にして recipe に network の注を足す。どちらも反映した)
- [x] Plan approved(自動承認。ユーザーの「最後まで止まらずに」の決め。図解ページは lint 0 と撮り直しを確かめた。digest c2af738f882b。S1 の途中で consult が、git が変わった回の `XREVIEW: git_changed` と、行が欠けた TASK への QUESTION を約束に足すと決めたので、D5 に書き、図解ページの Detail 3 を合わせ、lint 0 と撮り直しを確かめて承認し直した。digest 718d571f5e9f。S2 と consult の決め(D7b)と、S1b で相手のモデルが出した所見(D6b)を書き、スライス S1c と S2b を足し、図解ページの Detail 4 の QUESTION の箱を合わせ、lint 0 と撮り直しを確かめて承認し直した。digest 343b8c8dd0ee。S1c が codex の log の要点を最後の空でない行にしたので D6b の 1 文を合わせ(図に影響なし)、承認し直した。digest 46439ef27692。self-review の L7 で Affected areas のテストのファイル名と、S5 で触ったファイルを実際に合わせ(図に影響なし)、承認し直した。digest 7789ae616bd0。verify の INFO で Affected areas に recipe と tech-debt を足し(図に影響なし)、承認し直した。digest f30958e4f38f。sync-docs の指摘で Assumptions の driver の読み方を F6 の形に直し(図に影響なし)、承認し直した。digest 6e44700b98f0。cross-review cycle 1 の直し(F7)で D7b の `since:` の除外を `docs/reports/` 直下の生成された記録だけにした文に合わせ(図に影響なし)、2026-11-02 に承認し直した。digest ce18b629320e)
  - cross-review cycle 1(3aeafce1、codex、ACTION_REQUIRED 2): F7 4d289952 で、`since:` の除外を `':(exclude,glob)docs/reports/*.md'` にして `docs/reports/templates/` の変更を数える、`XREVIEW_BASE` は文字の許可リストをやめて、`origin/` で始まり `-` で始まらず `..`・`@{`・空白・制御文字・引用符・`$`・`` ` ``・`\` を含まないことを見てから `git check-ref-format --branch` に渡す(`@{-1}` を前のブランチに読み替えて通すので `@{` はここで断る)。`docs/reports/README.md` も直下の `.md` なので除かれる(記録ではないが、変わっても流れは変わらない)。変異 7 個が落ちる。機械的な確かめは 5 項目とも clean。前のセッションで渡した F7 は実装の前に止まっていたので渡し直した。`cycle-count.json` を 2 にして self-review から回し直す
- [x] Implementation started
  - S1 e3c04e18 と 586f7934(reviewer の cross-review の TASK。586f7934 は consult の決めの `git_changed` と、行が欠けた TASK への QUESTION。件名は指定の `fix:` でなく `feat:` になったが、中身は依頼どおりなので amend させない)
  - S1b(本物の CLI での確かめ): 2 方向とも rc 0・`out: not empty`・`git: same`。codex は 5 分 20 秒で所見 1 件(P1: `--disable apps` がない)、claude は 11 分 46 秒で所見 4 件(MEDIUM 1・LOW 3)。所見と気づいた点は D6b に書いた。evidence は `docs/evidence/xreview-live-probe-2026-11-01.md`
  - S2 2478e069(leader の cross-review の段。QUESTION の形と `since:` の注は consult の答えが後から届いたので S2b で直す)
  - S1c aa0a0fd5(reviewer の直し、D6b): codex に `--disable apps`(手元の codex 0.163.0 で受け付けると確かめた)、incomplete の理由に log の 1 行(claude は最初、codex は最後の空でない行)、driver の値をコマンドの中でも一重引用符で確かめる、helper がないときの別の理由、`sh -c 'command -v'` の確かめ(zsh の alias だけの場合のテストを足した)、claude への指示文、P0〜P3 の読み替え、負荷で落ちる 2 か所を行ごとの確かめに。変異 3 つが落ちる。機械的な確かめは 5 項目とも clean
  - S2b 82df1474(leader の直し、D6b・D7b): 上限の QUESTION を `XREVIEW: cap_reached` のヘッダと `FORK 1:` の行(a 上限を上げる(推奨)/ b 残りか見せていない変更を PR に書いて進む / c 止める)にし、DECISION の `FORK 1:` で答えを読む。a の新しい上限は `done` + 1(設定が下げられたときに同じ問いがくり返されないため)。`FORK 1:` がない DECISION では問いを上げ直す。c は push も disband もしない。`since:` の注 3 つ(rebase で上限を使う、HEAD の object が消えたら変わったとみなす、送り直しの回数は TASK_ID から読める)。計画助言の codex にも `--disable apps`(古い codex は `mcp: not off` で飛ばす)。機械的な確かめは 5 項目とも clean
  - S3 93d58b65(skill): `/director` に cross-review の停滞と「### cross-review の上限」(FORK の答え方、a は新しい上限を書く、c は人が止めるかを決める)、RESULT の `CROSS_REVIEW:` の報告。`/org` は前提の 2 か所を今の動きに直し、終わり方に cross-review の段、停滞、PR の `Cross-review:` の行。写し 4 つ。新しいテスト `skill_xreview_test.go`。完了条件の項は既存の固定に掛かるので変えなかった。機械的な確かめは 5 項目とも clean(長い 1 行を折り返した)
  - S4 91c3798b(文書): config.go と templates/base/ralph.toml のコメント(complete の回だけを数える)、仕様の 2026-11-01 の注、README(278 行の古い説明も)、docs/reports/README.md と写し、codex-seat-permissions.md と写しの network の注。機械的な確かめは 5 項目とも clean
  - S5 106bc425(古い記述): `RALPH_PRIMARY_CLI` がなくても座席の cross-review は TASK の driver で選ぶので、paneenv.go の警告文を leader の計画助言のことだけにし、spawn.go のコメント、recipe と写し、警告を固定するテスト 2 本を合わせた。README:278 の「instead of treating it as claude」は sync-docs に回す。機械的な確かめは 5 項目とも clean
  - self-review cycle 1(a47bd973、MEDIUM 1・LOW 7、Merge: yes)の直し。F1 522fb8cd: M1 数える report をこのブランチが足したか書き換えたもの(`git diff --diff-filter=AM origin/<base>...HEAD`)に限る(同じ org_id の前の org が main に入れた report を数えない。同じ日に org_id を使い直すと同じパスを上書きするので A でなく AM)、base が決まらなければ BLOCKED、L1 上限を上げたら状態のコマンドを打ち直して新しい `cap:` で TASK を作る、L3 sync-docs の段の頭を cross-review のあとの形に、L4 `log:` の行を 200 字で切る(UTF-8 の文字の途中で切らないよう awk で数える)。F2 a6a39a6c: L2 director は上限の件の report を機能の worktree で読み、新しい上限を report の `- Cycle:` の `<n>` で説明する、L3 の skill の側、L5 helper のコメント、L6 evidence の利用者名を含むパスを伏せた、README:278。F3 2c41af71(F2 の範囲外のメモ): 上げた上限の数は DECISION の `FORK 1:` の行の `新しい上限 <数>` からだけ読む(`SUMMARY:` の `e5` などを拾わない)。L7 は orchestrator が計画の Affected areas と tech-debt の行で直した(2632f14f)。機械的な確かめは 3 スライスとも 5 項目 clean
  - self-review run 2(15e62eb5、LOW 4、Merge: yes)の直し。F4 dbfc3ef2: R2-1 `k:` はこの org_id の全期間の report の番号 + 1(前の org のパスを使わない)にし、数えるのはこのブランチが足した report だけ(`--diff-filter=A` に戻した)、R2-3 L1 の規則のテストを雛形の文と状態のコマンドで確かめる形に、R2-4 base がない枝の前に `base.err` を空にする、L4 の残り `log:` の行の `$HOME` を `~`、worktree のパスを `.` にしてから切る(`<worktree>` は雛形の差し込み記号と同じ形でテストが埋め残しとみなすため)。R2-2 は orchestrator が tech-debt の行数を数え直した。LOW だけの直しなので self-review は回し直さず verify に進む。機械的な確かめは 5 項目とも clean
  - verify cycle 1(d0d9d3ba、pass、LOW 2)の直し。F5 62d62be0: V2 report の `- HEAD:` の値は完全な object 名(40 か 64 桁の 16 進)のときだけ `--end-of-options` のあとで `git diff` に渡す、`last head:` は printf で出す(zsh の echo が `\n` を解釈して `since:` の行を偽造できたため)、leader が report をコミットする前にパスの形と番号を確かめる、reviewer は `XREVIEW_BASE`・`REPORT:` の日付・`REVIEWER_*` の文字を確かめ、claude の `git diff` にも `--end-of-options`(直す前は `--output=` でファイルが書け、`/../` で docs/reports/ の外に書けた)。V1 返事の例の TASK_ID を `xreview-{{ORG_ID}}-c1` に。SHA-256 の repo でも試した。範囲外: 手順 9 のふつうのレビューの report のコミットは `git add --` を使えと書いていない(前からの書き方、sync-docs で tech-debt に送る)。機械的な確かめは 5 項目とも clean。AC9 の 2 回目の本物の CLI での確かめを 62d62be0 で回す
  - test cycle 1 run 1(5b6c3d42、pass。run-test.sh 1,830 秒、race 3 回、変異 9 個がすべて落ちる)。2 回目の本物の CLI での確かめ(afccca76、evidence に追記): 2 方向とも rc 0・`out: not empty`・`git: same`。相手のモデルの所見は codex の P2 1 件(`XREVIEW_BASE` が `+`・`@` を断る)と claude の MEDIUM 1・LOW 2(implementer の driver を記憶から写す、`Reviewer status` の照合が厳しい、QUESTION の送り直しに上限がない)
  - F6 17fc2b52(2 回目の確かめの所見): `XREVIEW_BASE` で `+`・`@` を受け、`@{`・`..`・先頭の `-` は断る、implementer の driver は TASK を作るたびに `ralph org status --state-dir <台帳> --json` を jq で読む別のブロックから取る(jq がなければ BLOCKED)、`Reviewer status: complete` は行末の空白を許す、QUESTION の送り直しは `-r3` まで、codex の絶対パスは report では相対に。ずれ: テストの追記を一度 `cat >>` で書いた(そのあと Edit を重ね、U+FFFD は 0 件)。機械的な確かめは 5 項目とも clean(Edit が消した空白 1 か所を直した)
  - test cycle 1 run 2(93d8aad8、pass。`./internal/org/` 全体 1,096 秒、race 3 回、変異 17 個のうち 14 個が落ちる。残った 3 個は雛形とテストの定数を一緒に変えたもので、テストの穴として sync-docs で tech-debt に送る)
  - sync-docs cycle 1(HEAD 433df4ae): 仕様の 2026-11-01 の注(`IMPLEMENTER_DRIVER:` は F6 のとおり `ralph org status --json` から読む、数えるのはこのブランチが足した report)、`/org` skill の 4 つの写し(終わり方 1 の driver の読み方と数える report、タスクの reviewer の行の段落に cross-review の `REVIEWER_*`)、`.claude/rules/ralph/model-routing.md` と写し(同期の注と effort の一覧に reviewer 座席の cross-review)、`templates/base/ralph.toml` の `[reviewer]` のコメントを直した。tech-debt はテストの穴 9 つの行と座席の中の実行の行を足し、雛形の大きさの行を 433df4ae の数に直し、13 行に注か追記を足した(Trigger に当たったのは 10 行)。手順 9 のふつうのレビューの report の `git add --` は、org-gate の小さい所見の行の (o) にすでにあるので、その行に注を足した。Assumptions の 3 つ目(driver は手順 5 の spawn のもの)は F6 で古くなったが、計画は印とこの行のほかは変えていない。`internal/config/config.go` の `ReviewerConfig` と `reviewerWordPattern` のコメントが reviewer 座席を挙げていないのはコードなので直さず、報告に回した
  - self-review cycle 2(d14ecf9a、LOW 4、Merge: yes)の直し。consult の判定は「4 件ともこの回で直す」。F8 7724f6af: C2-2 reviewer は base の確かめが `base: invalid` を出したときだけ QUESTION にし、`base: ok` も `base: invalid` も出なければ(heredoc が閉じなかった)打ち直す。C2-3 同じ `<k>` の送り直しは QUESTION のあとと `git_changed` のあとを合わせて 2 回(`-r2`・`-r3`)まで、`-r3` から先の TASK に QUESTION か `git_changed` が来たら BLOCKED で止まり、人の DECISION が言うときだけ番号を続けて(`-r4`、…)送る。`git_changed` のあとは `git status --porcelain` が空のときだけ送り直す。C2-1 tech-debt のテストの穴の行から F7 で消えた (6)(7) を外して 7 つに。C2-4 config.go と model-routing.md の折り返し、`ReviewerConfig` のコメントに reviewer 座席。検証は org・config のテスト、check-sync、agent-models が通り、変えた 4 つのテストは雛形を戻すと落ちる。F7 の件名「git が許すブランチ名を受け付ける」は言い過ぎで、引用符・`$`・`` ` ``・`\` を含む名前は断る(コミットは push 済みなので直さない)
  - verify cycle 2(58031854、pass、LOW 2): V2-1 仕様の 133 行目は sync-docs cycle 2 で直す。V2-2 origin/main が 31812a3b まで進んでいたので、4424b257 で取り込んだ(rebase はしない)。衝突は `docs/tech-debt/README.md` の 6 か所だけで、両側の注と行をすべて残した(4 行は両側の注を合わせた)。merge のあと build、vet、check-sync、check-skill-sync、`run-static-verify.sh` が通った
  - test cycle 2 run 1(b9829fa6、fail): merge で入った #271 の `TestDirectorSkill_AsksThePlanApprovalForksWithNumberedOptionsWhenTheWakePushes` が、このブランチの「cross-review の上限」の手順 2 の古い言い方で落ちた(git は別の段落なので衝突なしで混ぜた意味の衝突)。ほかは run-test.sh の shell 45 本と Go 9 パッケージが ok、変異 26 個のうち 19 個が落ちた。F9 4f46dce1: 手順 2 を #271 の言い方(Codex と、wake に push がある director は番号つきの選択肢)にし、「人への聞き方」の場面に上限の `FORK 1:` の問いを足した。テストの穴 2(常に 0 で終わる git のスタブで base の確かめの case だけの断りを固定)と 3(F8 の前の条件がないことの否定の固定)も閉じた
  - test cycle 2 run 2(e969ab5c、pass): `./internal/org/` 全体 1,525 秒、race 517 本、check-sync と check-skill-sync、写しを読む shell テスト 2 本。変異 8 個のうち 7 個が落ち、残ったのは文を足すだけの D4。既知の flaky(tech-debt の負荷で落ちる 2 本の行)が写しの対照で 1 回落ちた
  - sync-docs cycle 2(HEAD f0b0177d): 仕様の 2026-11-01 の注(`docs/specs/2026-10-07-org-multi-org-director.md` の 133 行)の上限の QUESTION の条件を、F7 のとおり「`docs/reports/` の直下の `.md`(生成された report)のほかが変わっていれば。`docs/reports/templates/` のような下のディレクトリの変更は数える」に直した(verify cycle 2 の V2-1)。README、`/director` と `/org` の skill と写し、`docs/reports/README.md`、recipe、model-routing.md と写しに同じ言い方は残っていない。F8 の送り直しの上限と reviewer の base の確かめの QUESTION の条件を書く文書は、雛形とテストと tech-debt だけで、仕様と skill は書いていない。tech-debt は、雛形の大きさの行を f0b0177d の数に直し、テストの穴の行に F8 の 2 つの規則と文を足すだけの変異 D4、F9 の分、F8 の持ち越しの注を足し、`config.go` の行の数(1190 → 1192)、recipe の行の引用の数(merge で 12 → 13)、負荷で落ちるテストの行の注を足した。計画の本文は変えていない(digest `ce18b629320e`)
  - cross-review cycle 2/2(b7fa913c、codex): 所見 0 件、Case C。consult の判定は「/pr に進んでよい」
  - PR #298 の CI(verify、ubuntu)で、`TestRenderRolePrompt_Reviewer_XreviewFindsNoCLI` の claude の 2 行が落ちた。fixture の `sh` はホストの sh へのリンクで、ubuntu の dash は見つからないコマンドの `command -v` に 127 を返す(macOS の sh は 1)。雛形は `available rc=0` かどうかしか見ないので変えず、F10 7e30b80f でテストの期待値を同じ sh の終了コード(0 以外であることも確かめる)から作るようにした。手元で sh を dash にした回で、直す前に CI と同じ失敗、直したあとに通ることを確かめた
- [x] Branch created
- [x] Review artifact created
- [x] Verification artifact created
- [x] Test artifact created
- [x] PR created(#298)
