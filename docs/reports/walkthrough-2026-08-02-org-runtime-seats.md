# Walkthrough: org-runtime-seats (PR②)

- Date: 2026-08-02
- Plan: docs/plans/archive/2026-08-02-org-runtime-seats.md(アーカイブ後)
- Spec: docs/specs/2026-08-01-org-runtime.md(PR 系列②)
- Diff: 42 files, +5,444/-228(main...HEAD)

## この PR の性格

PR①(機構層)が「スタブに対して正しい」だったのに対し、PR② は**実機(herdr v0.7.5 / agmsg v1.1.13)に対して正しい**ことを保証する段。実機スモークが 5 回の試行で 4 つの統合バグ・誤仮定を検出し、それぞれを修正して完走した。この検出→修正の履歴自体が本 PR の主要な成果物である(plan の "Implementation notes (deviations)" に全記録)。

## 読む順序(推奨)

1. **docs/plans/archive/2026-08-02-org-runtime-seats.md** — スコープ・AC・実機検証で判明した逸脱 5 件(必読)。
2. **internal/org/driver/agmsg.go** — 全面書き直し。実 agmsg はスクリプト群(`bash <home>/scripts/send.sh <team> <from> <to> <msg>` 等)。home 解決は env > `[org] agmsg_home` > 既定。`AgmsgAvailable` は npm ブートストラッパー誤検出を避けるため `scripts/send.sh` の存在で判定。
3. **internal/org/spawn.go** — saga の agmsg ステップが `ensureLeadJoined`(冪等)→ 座席 `join.sh` → HELLO send に拡張。実機起因の修正 3 点: herdr JSON エンベロープ解析(`workspace_id`/`pane_id` 抽出)、複数行プロンプトのファイルポインタ渡し(herdr は複数行 argv を拒否)、`agent_pane_busy` の有界リトライ(タブ直後のシェル初期化待ち)。
4. **internal/org/identifier.go** — ID charset `^[a-z][a-z0-9-]{0,29}$`(herdr のエージェント名制約に実機プローブで整合)。`_` をセパレータとして予約し、`<org>_<seat>` を無曖昧に(cross-review 指摘の衝突 `a-b`+`c` vs `a`+`b-c` を構造的に排除)。結合長 ≤32 検証。
5. **internal/org/protocol/** — typed message バリデータ(TYPE 列挙・TASK_ID 必須集合・本文 2,000 字上限)。`ralph org send` が既定検証、`--raw` でバイパス(manifest に記録)。規約文書: `.claude/rules/agent-messaging.md`。
6. **internal/org/prompts/** — reviewer / qa の役割プロンプト(go:embed)。`spawn --role` で適用、変数展開({{ORG_ID}} 等)、スター型規約とプロトコルを本文に内包。
7. **internal/org/verbs.go** — stop/disband の実在座席前提条件(phantom 防止)、`leave.sh` による roster 除去(despawn.sh は join 参加メンバーに no-op と実機で判明)、stopped イベントへの座席メタデータ引き継ぎ。
8. **docs/evidence/org-seats-smoke-2026-08-02.txt** — 実機スモークの一次証拠(attempt 1〜5)。

## 実機スモークが検出した統合バグ(修正コミット付き)

| # | 検出 | 修正 |
|---|---|---|
| 1 | herdr CLI は JSON エンベロープを返す(PR① は trimmed stdout=ID と仮定) | 376a9ca |
| 2 | herdr は複数行 agent 引数を拒否 → 役割プロンプトを argv で渡せない | e043fde(ファイルポインタ方式) |
| 3 | タブ作成直後の pane は `agent_pane_busy` | 1d1825b(有界リトライ) |
| 4 | `despawn.sh` は join 参加メンバーに no-op → 除去は `leave.sh` | cd1452d |
| 5 | herdr エージェント名は `小文字始まり・[a-z0-9_-]・≤32`(cross-review 起因のプローブ) | f0cbf11(`_` セパレータ+charset 制限) |

## 品質ゲート履歴

| Cycle | ゲート | 結果 |
|---|---|---|
| 1 | self-review | MEDIUM 5(env 優先順位反転・パストラバーサル・ensureLeadJoined 不在ほか)→ 1520b96 で修正 |
| 1 | verify / test | PASS / PASS(coverage: internal/org 86.7%) |
| 1 | cross-review (Codex) | ACTION_REQUIRED 1(ハイフン連結の曖昧性)→ f0cbf11 で修正 |
| 2 | self-review | CRITICAL/HIGH/MEDIUM 0、LOW 7 → 安価な 4 件を 011a0a7 で修正、残りは tech-debt |
| 2 | verify / test | PASS / PASS |
| 2 | cross-review (Codex) | ACTION_REQUIRED 1(announce 失敗時の roster 残留)→ cap 到達につき Known gaps へ |

## Known gaps(意図的繰延、いずれも tech-debt 登録済み)

- announce(HELLO)失敗時に join 済み座席の agmsg roster エントリが残留(失敗パス限定・手動回復可)— PR③
- herdr エージェント名が derived で manifest に永続化されていない(名前形式変更で旧座席が孤児化。リリース済みビルドへの影響なし)— PR③
- 座席の permission-mode 設定(実機スモークで座席が権限確認ダイアログで blocked になることを確認)— PR③ の Lead 編成と同時
- 並行 spawn TOCTOU(PR① から継続)— PR③
