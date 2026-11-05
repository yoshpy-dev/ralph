# Cross-review triage report: org-herdr-autostart

- Date: 2026-11-06
- Plan: docs/plans/active/2026-11-06-org-herdr-autostart.md
- Base branch: main
- Driver: claude
- Reviewer: codex
- Triager: Claude Code (main context)
- Reviewer status: complete
- Self-review cross-ref: yes
- Cycle: 1/2
- Total reviewer findings: 0
- After triage: ACTION_REQUIRED=0, WORTH_CONSIDERING=0, DISMISSED=0

## Triage context

- Active plan: docs/plans/active/2026-11-06-org-herdr-autostart.md(承認の digest 9bda99914bec)
- Self-review report: docs/reports/self-review-2026-11-06-org-herdr-autostart.md(Merge: yes。MEDIUM 1、LOW 5)
- Verify report: docs/reports/verify-2026-11-06-org-herdr-autostart.md(pass。LOW 1、INFO 3)
- Implementation context summary: S1〜S5 で、`herdr status server --json` を読む driver のメソッド、Org の herdr のゲート、CLI の自動起動と環境の整理、手順書、分離した herdr での実機の確かめを入れた。test report(docs/reports/test-2026-11-06-org-herdr-autostart.md)は pass で、変異 41 個のうち C21 だけが残った。

codex(`codex exec review --base main`、`-c sandbox_mode=read-only`)の出力は次の 1 文だけだった。

> No actionable regressions were identified. Diff whitespace and Go formatting checks passed; targeted tests could not run because the read-only sandbox prevented Go from creating its build directory.

指摘が 0 件なので、cross-review としては Case C(/pr に進める)にあたる。レビューの前後で git の状態(HEAD、index、作業ツリー、未追跡のファイル)は変わらなかった。

### パイプラインのほかのレポートに残った所見と、この run のあとの扱い

cross-review の指摘ではないが、/pr の前に扱いを決めた。consult-advisor に意見を聞き、cycle 2 で次を直して全パイプラインを回し直すことにした。

- self-review M1(MEDIUM): Spawn の herdr のゲートが冪等の early return より前にあるのに、Spawn の doc の step 1 と PR① の前例のコメントがそれを言っていない → doc に意図として書き、有効な座席への打ち直しで失敗するゲートが `failed` を返して台帳に何も書かないことをテストで固定する。
- test C21: start --plan を本物のゲートで通すテストが無い → tester の案(起動 1 回、状態の読み取り 3 回、start の言葉の 1 行)をテストに入れる。
- verify I1: 実機の記録の AC2 の表の `HERDR_TAB_ID` の行は、herdr がペインごとに値を付け直すので、落としたことの根拠にならない → 文言を直す。
- verify I3: `--help` が失敗を「refused」と書くが、spawn は `spawn failed:` を出す → 文言を直す。
- self-review L5: `await` の `left` の更新と `ensureOrgHerdrServer` の `switch` の意図 → コメントを足す。

次は直さず、tech-debt と PR の既知の穴に書く。

- self-review L1(同時に起動したときの負けた側の誤った失敗や誤った 1 行): 実機で見ておらず、`afterExit` を締め切りまで読み続ける形にすると、本当に起動できないときの失敗が最大 10 秒遅れる。doc に、負けた側が失敗したら打ち直すことを書く。
- verify V1(別のソケットを指す端末からの director start で director が 2 人になりうる): コードを読んだ推論で、確かめていない。直すには、登録に herdr のソケットを記録するなど範囲が広がる。tech-debt の、ralph が herdr のサーバーに届く道の行に注を足す。
- self-review L2(`CODEX_*` を丸ごと残す、仕様 FR-A3 の決定)、L3(`HerdrServerStatus.Compatible` が本番で未使用)、L4(10 秒が 3 か所)。

## ACTION_REQUIRED

| # | Reviewer finding | Triage rationale | Affected file(s) |
|---|-------------------|------------------|-------------------|

## WORTH_CONSIDERING

| # | Reviewer finding | Triage rationale | Affected file(s) |
|---|-------------------|------------------|-------------------|

## DISMISSED

| # | Reviewer finding | Dismissal reason | Category |
|---|-------------------|------------------|----------|

Categories: false-positive, already-addressed, style-preference, out-of-scope, context-aware-safe
