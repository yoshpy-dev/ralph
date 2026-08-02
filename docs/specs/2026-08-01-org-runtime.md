# org runtime — LLM 主導組織ランタイムへの実行面全面刷新

## Summary

Ralph Loop(/loop)の自律実行系を撤去し、Lead LLM が herdr(実行・観測)と agmsg(メッセージング)を使って常駐 LLM 座席(scout / impl / reviewer / QA / watchdog)による組織を自律編成する新しい実行面「org runtime」を導入する。標準フロー(開発ハーネス: `/spec` `/plan` `/work` `/self-review` `/verify` `/test` `/sync-docs` `/cross-review` `/pr`)は廃止せず、org runtime と並存する開発ハーネスとして存続する(適用範囲は FR-6・FR-11 参照)。org runtime に関する人間の役割はエンベロープ定義(ralph.toml)・Lead 起動・エスカレーション裁定の3つに縮小し、決定論的スクリプトは機構(`ralph org` 動詞)と保証(品質ゲート・ハードリミット・監査証跡)に退く。

## Background and problem

### Current state

- 実行面が標準フロー(セッション内サブエージェント)と Ralph Loop(`ralph-orchestrator.sh` + `ralph-pipeline.sh` によるバッチ `claude -p` 並列)の二軸に分かれ、タスクごとに人間がフローを選択している。
- Ralph Loop はバックグラウンドサブシェル + PID ファイル + 5秒ポーリングで動き、稼働中エージェントへのライブチャネルがない。エージェント間通信はサイドカーファイルのみで、スライス間の契約変更を伝える手段がない。
- `run_agent` の `claude -p` 呼び出しにはターン単位のタイムアウトがなく、ハングはスライス全体のタイムアウト(既定30分)まで検出されない。進捗検知は `check_stuck()`(3イテレーション無コミット)のみ。
- 組織編成(何体を、どのモデルで、どう分担するか)がスクリプトの静的構造に固定されており、タスク特性に応じた柔軟な編成ができない。
- `codex exec` はモデル引数を無視するため、Codex 側座席のモデルを ralph から指定できない。

### Desired state

- 人間はエンベロープ(使用可能モデルプール・座席上限・予算)を ralph.toml に定義して Lead を起動するだけ。Lead がタスクを分類し、herdr pane 上の常駐対話セッションとして座席を spawn し、agmsg で統括する。
- 座席は対話セッションなので、走行中のステアリング・herdr の blocked 検知・人間の pane 介入・座席内サブエージェント fan-out が全て可能になる。
- モデル選定は「人間がプールを定義し、Lead がプール内でタスクごとに動的選択」の二層。対話セッション起動なので `codex --model` も効く。
- 品質ゲート(verify / test / cycle cap / budget)は hook と動詞バリデーションで LLM 迂回不能に執行され、全編成・全判断が org manifest と receipts に自動記録される。
- LLM を全部止めても、機構層だけで安全停止と状態説明ができる。

## Requirements

### Functional requirements

- [ ] **FR-1 `ralph org` 動詞セット**: `spawn` / `send` / `wait` / `read` / `stop` / `status` / `disband` を冪等な決定論実装で提供する。`spawn` は 1 コマンドで herdr pane 作成 → worktree 用意 → agmsg チーム参加 → 役割プロンプト投入 → エンベロープ検証を完了する。プール外モデル・座席上限超過・スコープ違反は動詞のバリデーションで拒否し、receipts に記録する。
- [ ] **FR-2 `[org]` エンベロープ設定**: `model_pool`(`{ driver, model }` の組。model は `claude --model <名>` / `codex --model <名>` に渡す CLI ネイティブ名)、`driver_pool`、`[org.roles]`(役割別許可プール)、`max_seats`、budget(座席別/タスク全体の wall-clock 上限、fix ラウンド上限)を定義できる。config.go(struct / `Default()` / `Load()`)・`templates/base/ralph.toml`・`scripts/ralph-config.sh` のロックステップを維持し、`defaults_sync_test.go` を更新する。`ralph doctor` は全プールエントリを起動プローブし、無効なモデル ID を警告する。
- [ ] **FR-3 座席 = 常駐対話セッション**: 座席は herdr pane 内の対話 CLI セッション(`claude --model <名>` / `codex --model <名>` 直接起動)。初期プロンプトは起動引数で渡し、追加指示は `herdr agent wait --until idle` を挟んだ `send` 動詞経由とする。座席内部での Task サブエージェント fan-out は座席の裁量。
- [ ] **FR-4 agmsg スター型プロトコル**: identity は `lead` / `scout-<n>` / `impl-<slug>` / `reviewer` / `qa` / `watchdog`。typed message(`TYPE: TASK | RESULT | QUESTION | REVIEW | DECISION | BLOCKED | CONTRACT | HEARTBEAT | STOP`、`TASK_ID`、`EVIDENCE` はコミット SHA / ファイルパス / レポートへのポインタのみ)。impl 間の直接通信は CONTRACT のみ、lead 経由。プロトコルは `/org` skill と `.claude/rules/` に明文化する。
- [ ] **FR-5 Lead**: 現セッション昇格(`/org` skill)と headless 起動(`ralph org start "<task>"`)の両対応。エンベロープ内で完全自律編成(人間の編成承認なし)。実装は原則座席へ委譲(火消しは可)。解散判断・レビュー裁定・最終確認・最終責任を所掌。stop / replan の実行判断は Lead が持つ。座席 0 のソロ実行(herdr / agmsg を使わない自明な小修正)を許容する。
- [ ] **FR-6 開発ハーネス `/plan` の存続と Lead 自律計画立案の並存**(適用範囲確定 — PR⑤ 計画時のユーザー確定判断。FR-11 の撤去範囲注記と整合): `/plan` スキルは廃止しない。標準フロー(開発ハーネス)の一部として存続し、人間対話による曖昧要求の言語化(`/spec`)からタスク分解・計画作成(`/plan`)までを従来通り担う。org runtime 経由で自律編成されるタスクでは、`/plan` を介さず Lead 自身がタスク分解・計画立案を行う — これは `/plan` の廃止ではなく、org runtime という別実行面における Lead 自身の業務である。Lead が立てた計画アーティファクト(目標・分解・受け入れ条件・座席割当)は監査用に自動生成され `docs/plans/` に記録される(人間承認なし)。`/spec` は曖昧な要求の言語化フェーズとして両実行面で共通に存続する。
- [ ] **FR-7 新品質パイプライン(4フェーズ)**: ① impl 座席が退出チェック(スコープ内・コミット境界・verify / test のローカル通過)付きで RESULT 報告 → ② QA 座席が `run-static-verify.sh` / `run-test.sh` を実行・解釈し QA レポート作成 → ③ reviewer 座席が独立レビュー(diff 品質+仕様適合を統合。入力は要求・AC・diff・QA レポートのみ。逆 CLI 要件は撤廃し `[org.roles].reviewer` プールから選定)→ ④ lead 裁定(fix ラウンド既定 2、上限はエンベロープが執行)→ doc-maintainer を on-demand spawn → lead 最終確認 → PR。旧 6 フェーズ(self-review / verify / test / sync-docs / cross-review / pr)はこの新順序に置換される。ゲートは hook で LLM 迂回不能とする。
- [ ] **FR-8 Watchdog 二層**: パルス層(決定論タイマー、既定 30 秒: heartbeat 途絶 / プロセス生存 / budget / スコープ外変更 / ラウンド cap。ハードリミット超過は判断を経由せず自動遮断)+ ウォッチャー層(パルス層トリガーのオンデマンド LLM 判定: 意味判定トリガー検知時のみ `claude -p` を非同期 single-flight で 1 回起動し、パルスを塞がない。循環議論・役割逸脱・偽進捗の意味判定)。常駐 LLM 座席ではなくオンデマンド起動とすることで、常駐コストゼロ・判定器ハングは次回起動で回復する(PR④ `feat/org-runtime-watchdog` の設計決定、`docs/plans/active/2026-08-02-org-runtime-watchdog.md` の Design decisions 参照)。通知は Lead 宛。デッドマン条項: 異常主体が Lead 自身、または Lead が通知に N 分無応答の場合のみ人間へエスカレーション(端末 + PushNotification)。権限は PAUSE / REPLAN 要求 / 通知のみ。
- [ ] **FR-9 監査証跡**: org manifest(`.harness/state/org/manifest.jsonl`)へ全 spawn / send / stop / disband / 遮断イベントを動詞が自動追記(Lead の自己申告に依存しない)。全イベントに `org_id`(実行単位の名前空間)/ `seat_id` / `worktree` を必須フィールド化し、`max_seats` 集計・status・disband は同一 `org_id` 内に限定する。dry-run イベントは `dry_run: true` を刻印し既定で集計から除外する。spawn は saga(`spawn_started` → `spawned` / `spawn_failed` + 補償記録)として記録する。完了時に編成履歴を `docs/reports/` へ成果物化。model receipts は `commanded_model` / `reported_effective_model` / `honored: true|false|unknown` の三値とし、`true` はドライバ観測による確認がある場合のみ記録する。insights スキーマを拡張(リードタイム、初回 CI 成功率、レビュー往復数、人間介入数、座席数、停滞率)。
- [ ] **FR-10 `/org` skill**: 動詞の使い方、編成パターン(Solo / Leaded / Parallel の型)、agmsg プロトコル、スター型規約、budget 作法、役割プロンプト雛形を収録。`.agents/skills/` ミラーを `sync-skills.sh` で同期し、Codex 座席からも参照可能にする。
- [ ] **FR-11 Ralph Loop 自律実行系の完全撤去**(段階移行なし・PR 系列の最終 PR で一括削除。**適用範囲確定(PR⑤ 計画時のユーザー確定判断)**: 撤去対象は Ralph Loop の自律実行系のみ。標準フロー開発ハーネス skill 群(`/spec` `/plan` `/work` `/self-review` `/verify` `/test` `/sync-docs` `/cross-review` `/pr`)は org runtime と並存する開発ハーネスとして存続する — org の reviewer/qa 座席がこれらの検証スクリプト/skill を実行する関係でもあるため): `ralph-orchestrator.sh`、`ralph-pipeline.sh`、loop-init、旧 shell CLI の loop 系コマンド、`/loop` スキル(4 面ミラー)、loop 系テンプレート・レシピ・rules 節、`internal/ui` の Bubble Tea TUI(ライブビューは herdr に委譲)、`internal/state` のスライス/checkpoint リーダー、`internal/action` retry/abort。`ralph status` は org manifest を読むテキスト座席ビューに書き換える。`run-static-verify.sh` / `run-test.sh` / `ralph-worktree.sh` / driver 起動ロジックは標準フロー・QA 座席・動詞側の双方で存続利用する。

### Non-functional requirements

- [ ] **安全不変条件**: 全 LLM を停止しても、機構層(動詞 + manifest + receipts)だけで全座席の状態説明と安全停止ができる。
- [ ] **予算執行の決定論性**: budget の床は wall-clock・座席数・ラウンド数で機構が厳格執行。トークン量は receipts / セッションログから best-effort 計測し insights に記録(遮断条件にはしない)。
- [ ] **依存関係**: herdr / agmsg は ralph の必須依存とし `ralph doctor` で検査。座席 0 のソロ実行のみ両ツールなしで動作。
- [ ] **パリティ維持**: `templates/base/` と ルート設定、`.codex/` と `templates/base/.codex/`、`.claude/skills/` と `.agents/skills/` の各同期ゲート(`check-sync.sh` / `check-skill-sync.sh`)を org runtime 対応後も green に保つ。
- [ ] **ドキュメント整合**: AGENTS.md / CLAUDE.md / README / `docs/quality/definition-of-done.md` / rules / recipes を新アーキテクチャに全面改稿し、旧フロー参照をゼロにする。

## Acceptance criteria

- [ ] Given エンベロープにないモデル、when Lead が `ralph org spawn` を実行、then 機構がエラーで拒否し、receipts に `honored=false` 相当の拒否記録が残る。
- [ ] Given `max_seats` に達した状態、when Lead が追加 spawn、then 拒否され manifest に記録される。
- [ ] Given 座席の wall-clock budget 超過、when パルス層が検知、then LLM の判断を経由せず座席が遮断され、manifest に遮断イベントが記録され、Lead に通知される。
- [ ] Given Lead が watchdog 通知に N 分無応答、when デッドマン条項発動、then 人間へ通知が届く。
- [ ] Given `codex` driver の座席、when spawn 時にプールのモデル名を指定、then `codex --model <名>` で起動され receipts の `effective_model` が一致する(`honored=true`)。
- [ ] Given impl 座席の RESULT、when QA 座席のゲートが fail、then reviewer 座席にレビューが渡らず impl に差し戻される(lead 経由)。
- [ ] Given fix ラウンドが上限到達、when reviewer が追加 findings を報告、then 自動再ラウンドは発生せず lead 裁定に移る。
- [ ] Given 全 LLM 停止、when `ralph org status`、then 機構層だけで全座席の状態と証跡が表示される。
- [ ] Given 最終 PR マージ後、when `grep -r "ralph-orchestrator\|ralph-pipeline\|RALPH_LOOP_DRIVER" --include="*.sh" --include="*.go" --include="*.md"`、then 参照ゼロ(履歴・アーカイブ除く)。
- [ ] Given `ralph doctor`、when プールに無効なモデル ID がある、then 警告が出る。

## User stories

1. As a 開発者, I want to 一行のタスクを Lead に渡すだけで最適な編成が自律的に組まれる, so that フロー選択・編成承認・進捗監視のタスクから解放される。
2. As a 開発者, I want to herdr で全座席のライブ pane を観測し必要なら介入する, so that ブラックボックスなしに自律実行を信頼できる。
3. As a テックリード(人間), I want to ralph.toml でモデルプールと予算の上限を定める, so that コストと使用モデルの最終統制を保ったまま Lead に編成を委譲できる。
4. As a Lead LLM, I want to 冪等な `ralph org` 動詞で座席を操作する, so that pane 操作の失敗モードを再発明せず編成判断に集中できる。
5. As a 監査者, I want to org manifest と receipts で「誰が・何を・どのモデルで・何を根拠に」を追跡する, so that 事後に全判断を説明できる。

## Constraints

### In scope

- `ralph org` 動詞セット、`[org]` エンベロープ、常駐座席、agmsg プロトコル、Lead 自律編成、新品質パイプライン、Watchdog 二層、org manifest / receipts / insights 拡張、`/org` skill、旧系撤去とドキュメント全面改稿。
- driver は `claude` / `codex` の 2 種。

### Out of scope

- トークン厳格計測・コスト(USD)換算による予算執行(best-effort 記録のみ)。
- claude / codex 以外のドライバー(gemini 等。herdr は 20 種対応だが将来拡張)。
- マルチリポジトリ組織、Web ダッシュボード、Slack 通知(将来拡張)。
- agmsg / herdr 自体への上流コントリビュート。
- `/spec` スキルの改修(存続、現行のまま)。

## Impact

| Target | Impact | Severity |
|--------|--------|----------|
| `scripts/ralph-orchestrator.sh`(1722行)/ `ralph-pipeline.sh`(1420行)/ loop-init / 旧 shell CLI loop 系 | 削除 | CRITICAL |
| `scripts/ralph-cli-driver.sh` | 起動ロジックを org 動詞へ転用・改修 | HIGH |
| `scripts/ralph-config.sh` | `RALPH_LOOP_*` / `RALPH_MAX_*` → `[org]` 系へ全面置換 | HIGH |
| `internal/config` | Loop 構造体 → Org 構造体、ロックステップテスト書き換え | HIGH |
| `internal/ui`(Bubble Tea TUI)/ `internal/state` スライスリーダー / `internal/action` | 削除(`ralph status` は座席ビューに書き換え) | HIGH |
| `internal/watcher` / `internal/insights` | org manifest 監視・新スキーマ対応 | MEDIUM |
| `.claude/skills/`(work / loop / cross-review / self-review ほか)+ `.agents/skills/` ミラー | 削除・`/org` 新設 | HIGH |
| `.claude/agents/` pipeline サブエージェント定義 | 削除(役割プロンプトへ移行) | MEDIUM |
| `.claude/rules/`(post-implementation-pipeline / subagent-policy / model-routing) | 全面改稿 | HIGH |
| AGENTS.md / CLAUDE.md / README / recipes / templates / `docs/quality/` | 全面改稿 | HIGH |
| `templates/base/` + `.codex/` パリティ | `[org]` 設定・`/org` skill の追従 | MEDIUM |

## Dependencies

- **herdr**(必須依存): pane / workspace 管理、`agent start/wait/get`、`pane read/send-text/wait-output`。scriptable CLI を確認済み。
- **agmsg**(必須依存): SQLite ベースのチームメッセージング。monitor / turn / both 配送。Codex monitor bridge は β(孤児スレッド既知問題)のため Codex 座席は turn を既定とする。
- `gh` CLI、既存の `run-static-verify.sh` / `run-test.sh` / `ralph-worktree.sh`。

## Research findings

### Codebase analysis

- 全エージェント呼び出しは `run_agent`(`ralph-cli-driver.sh:245`)に集約されており、org 動詞への転用点になる。`claude -p` にターン単位タイムアウトはない。
- オーケストレーター↔スライス間の通信はファイルシステム + PID 生存確認のみ(`check_slice_status`)。スライス間通信は存在しない。
- 設定は config.go / `templates/base/ralph.toml` / `ralph-config.sh` の 3 面ロックステップで `defaults_sync_test.go` が執行。shell エントリポイントは toml を読まないため env が唯一の普遍チャネル。
- 観測面は完全にファイルポーリング(`internal/state` + fsnotify)。TUI はスライス前提の 5 ペイン構成。
- receipts(requested / effective / honored)と insights イベントの流儀が確立済みで、org でもそのまま拡張できる。

### Best practices

- Anthropic のオーケストレーター・ワーカーパターン: 独立コンテキストで探索し Lead が統合。コーディングは調査より並列化しにくい。
- Claude Code Agent Teams: Lead + 独立 teammate + 共有タスクリスト + メッセージ機構の中央集約型。
- OpenAI: 並列は読み取り中心から。複数エージェント同時書き込みは競合注意。
- マルチエージェント失敗研究: 役割逸脱・過剰調整・終了条件不認識が主要因 → スター型・typed protocol・有限ラウンド・二層 Watchdog で対処。
- agmsg は「not a task manager」(README 明記)→ 正本は manifest / git / reports に置き、agmsg は通知専用。

### Alternatives considered and trade-offs

| Option | Pros | Cons | Adopted |
|--------|------|------|---------|
| 座席 = `claude -p` バッチ(旧設計) | 再現性・再開性が高い | agmsg monitor 無意味・ステアリング不可・herdr の価値半減 | No |
| 座席 = 常駐対話セッション | ステアリング・介入・座席内 fan-out・`codex --model` 有効 | コンテキスト常時保持コスト → budget で統制 | **Yes** |
| Lead が herdr 生コマンドを直接操作 | 実装最小 | send-text の脆弱性を LLM が毎回再発明 | No |
| 決定論的動詞セット + LLM 編成判断 | 柔軟性と信頼性の両立 | 動詞セットの実装コスト | **Yes** |
| reviewer 逆 CLI 固定(現行) | モデル異種性を機械的に保証 | 独立性の本質はコンテキスト分離。プール制で人間が選択可能 | No |
| Watchdog LLM 単層 | 意味的異常も検知 | LLM 共倒れ・被説得リスク | No(二層採用) |
| 段階的移行(旧 Loop 併存) | リスク低減 | 二重保守。ユーザー判断で完全リプレース | No |
| 編成の人間事前承認 | 統制強い | 人間タスク最小化方針に反する。エンベロープ+監査+デッドマンで代替 | No |

## Security considerations

- **越権防止**: プール外モデル・座席上限は LLM から見えない機構層(動詞バリデーション)で拒否する。エンベロープの壁は判断対象にしない。スコープ外書き込みは PR② 時点では spawn `--scope` の manifest 記録+役割プロンプト指示+スモークでの前後 `git status` 検査に留め、**決定論的検知は PR④ の Watchdog パルス層に一本化**する(PR② 計画で確定した境界変更)。
- **agmsg 経由のインジェクション**: 座席は他座席からのメッセージを指示ではなくデータとして扱う規約を役割プロンプトと `/org` skill に明記。EVIDENCE はポインタのみとし、シークレットや長文コンテキストをメッセージに載せない(`git-commit-strategy.md` の safe-quoting 原則を send 動詞にも適用)。
- **自律座席の権限**: 座席の permission mode / sandbox は現行 `RALPH_PERMISSION_MODE` / codex sandbox 設定の流儀を継承し、エンベロープで指定可能にする。
- **デッドマン経路の完全性**: 人間エスカレーションはパルス層(決定論)からのみ発火し、LLM が抑止できない。

## Open questions

- `ralph org` 動詞の実装言語(Go サブコマンド vs shell)— /plan で決定。
- デッドマン条項の N 分既定値(暫定 10 分)と通知チャネルの設定面。
- agmsg メッセージスキーマの厳密なフィールド定義(プロトコルバージョニング含む)— /plan で確定。
- 座席の worktree 割当粒度(impl 座席のみ worktree 必須か、QA / reviewer は統合ブランチ read-only チェックアウトで足りるか)。

## References

- herdr: https://herdr.dev/ / https://herdr.dev/agent-guide.md / https://herdr.dev/docs/cli-reference/
- agmsg: https://github.com/fujibee/agmsg / https://github.com/fujibee/agmsg/blob/main/docs/codex-monitor-beta.md
- Anthropic multi-agent research system / Claude Code Agent Teams、OpenAI parallel agents guidance(設計議論で参照)
- 内部: `.claude/rules/post-implementation-pipeline.md`、`.claude/rules/model-routing.md`、`.claude/rules/subagent-policy.md`(いずれも本仕様で改稿対象)
- 設計議論: 2026-07-29〜2026-08-01 チャットセッション(v1 保守設計 → v3 org runtime 確定)
