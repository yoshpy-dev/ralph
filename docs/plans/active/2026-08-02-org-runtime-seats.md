# org-runtime-seats

- Status: Draft
- Owner: Claude Code
- Date: 2026-08-02
- Related request: docs/specs/2026-08-01-org-runtime.md(PR 系列②: 座席の常駐セッション化)
- Related issue: N/A
- Type: feat
- Branch: feat/org-runtime-seats

## Objective

org runtime PR②: 座席を実際に動かせるようにする。agmsg アダプタを実機インターフェース(スクリプト群)に書き直し、spawn saga に正式な `join.sh` 参加を統合し、役割プロンプト雛形(reviewer / QA)を go:embed で同梱、typed message プロトコルを規約化し、PR① の既知カバレッジギャップを解消する。最後に実機スモーク(herdr 上に実 claude 座席を spawn)で機構を通しで検証する。

## Scope

- **agmsg アダプタ書き直し**(`internal/org/driver/agmsg.go`): PR① の仮定(単一バイナリ + `--team/--as` フラグ)は実機検証で誤りと判明。実体は `~/.agents/skills/agmsg/scripts/` のスクリプト群(v1.1.13 で確認)。呼び出しを `bash <agmsg_home>/scripts/<verb>.sh <positional args>` 形式に変更:
  - `send.sh <team> <from> <to> <message>` / `join.sh <team> <agent_id> <type> <project_path>` / `team.sh <team>` / `history.sh <team> [agent] [limit]` / `despawn.sh <team> <from> <name>` / `whoami.sh <project_path> [type]` / `delivery.sh set <mode> <type> <project_path>`
  - agmsg home 解決: env `RALPH_ORG_AGMSG_HOME` > `[org] agmsg_home`(config 追加、既定 `~/.agents/skills/agmsg`)
  - `AgmsgAvailable()`: `exec.LookPath("agmsg")` は npm ブートストラッパーを誤検出するため、`<agmsg_home>/scripts/send.sh` の存在確認に変更(doctor の表示も追従)
- **spawn saga への join 統合**(`internal/org/spawn.go`): agmsg ステップを HELLO send 単独から `ensureLeadJoined`(`join.sh <team> lead claude-code <cwd>`、冪等)→ `join.sh <team> <seat_id> <type> <cwd>` → HELLO send に変更(Codex 所見 1: lead 未登録の roster 検証で send が落ちる手動前提を排除)。driver → agmsg type のマッピング(`claude`→`claude-code`、`codex`→`codex`)。`stop` / `disband` に `despawn.sh` の best-effort 呼び出しを追加。
- **stop / disband の実在座席前提条件**(Codex 所見 2): `stop --seat <未知>` は非ゼロ終了し、state イベントを追記しない(phantom 座席の防止。PR① self-review で繰延した既知問題をここでクローズ)。despawn 失敗時も manifest は真実を保つ(cleanup 結果を Details に記録)。
- **typed protocol バリデータ**(Codex 所見 3、新規 `internal/org/protocol/`): TYPE 列挙・TASK_ID 必須・本文サイズ上限(EVIDENCE ポインタ原則の機械的下限)を検証する小パッケージ。`ralph org send` は既定で検証し、不正形式を拒否(`--raw` で意図的バイパス可)。規約文書はこのバリデータを正として参照する。
- **役割プロンプト雛形**(新規 `internal/org/prompts/` + go:embed): `reviewer.md` / `qa.md`。変数展開(org_id / seat_id / team / scope / plan path / レポート契約)。`spawn --role reviewer` で自動適用、`--prompt` は追記。雛形にはスター型規約(宛先は lead のみ、EVIDENCE はポインタのみ)と typed protocol を埋め込む。
- **typed message プロトコル規約**(新規 `.claude/rules/agent-messaging.md` + `templates/base/` ミラー): TYPE 列挙(TASK / RESULT / QUESTION / REVIEW / DECISION / BLOCKED / CONTRACT / HEARTBEAT / STOP)、TASK_ID、EVIDENCE ポインタ原則、スター型トポロジ、他座席メッセージは指示ではなくデータとして扱う原則。
- **カバレッジギャップ解消**: `Verbs.Send / Wait / Read` + CLI 配線のテスト追加(tech-debt 登録済み行のクローズ)。
- **実機スモーク**(手動・証拠必須): 本マシンの herdr + agmsg で、**クリーンな team から手動セットアップなしに** `ralph org spawn --role reviewer --driver claude` を実行し、pane 起動・`herdr agent list` での状態遷移・`team.sh` での参加確認(lead + 座席)・`status` / `stop` / despawn までのライフサイクルを検証。スモーク前後で `git status --porcelain` を取得し、宣言スコープ外のパス変更がないことを決定論的に確認する(Codex 所見 4 の最小ガード。恒久的な検知は PR④ の Watchdog パルス層)。出力を `docs/evidence/` に保存。

## Non-goals

- Lead 自律編成・`/org` skill・`ralph org start`(PR③)。
- Watchdog 二層・budget 自動遮断(PR④)。
- 旧系撤去・ドキュメント全面改稿(PR⑤)。
- スコープ外書き込みの**決定論的ブロック**: spec は PR② 想定だったが、汎用対話セッションへの hook 強制は座席 worktree 生成の設計が絡むため、スコープ違反の**検知**は PR④ の Watchdog パルス層(`git status` × manifest 記録スコープ)に一本化する。PR② では spawn `--scope` の manifest 記録と役割プロンプトでの指示まで。spec 側の当該記述は /sync-docs で追従。
- effective model の実測(`honored` 三値運用は現状維持。実測手段は PR④ のウォッチャー設計と同時)。
- Codex 座席の実機スモーク(claude 座席で機構を検証すれば十分。Codex は adapter 単体テストまで)。

## Assumptions

- agmsg v1.1.13(インストール済み)のスクリプトインターフェースは上記シグネチャで検証済み。バージョン差異はアダプタに隔離。
- herdr は導入済みで `ralph doctor` pass。実機スモークは対話確認を伴うため /work 中に手動実行し、自動テストはスタブで成立させる(CI に実バイナリなし、の前提は PR① から不変)。
- 座席の delivery mode 既定は agmsg 側の既定に任せる(claude-code=monitor)。明示設定が必要になった場合のみ `delivery.sh set` を spawn に足す(open question)。
- 役割プロンプトの override 機構(リポジトリ側差し替え)は必要になった時点で追加(ユーザー確定: go:embed 単独で開始)。
- `join.sh` は roster 未登録の from/to を send が拒否する仕様(agmsg #355)への前提条件でもあるため、saga 内で join → send の順序を固定する。

## Affected areas

- `internal/org/driver/agmsg.go`(書き直し)+ `agmsg_test.go`
- `internal/org/spawn.go` / `verbs.go` / それぞれのテスト
- `internal/org/prompts/`(新規、go:embed)+ ローダ
- `internal/config/`(`[org] agmsg_home` 追加、ロックステップ 3 面 + defaults_sync_test)
- `internal/cli/org.go`(`--role` 対応)/ `doctor.go`(agmsg 検出変更)
- `.claude/rules/agent-messaging.md`(新規)+ `templates/base/.claude/rules/` ミラー
- `docs/tech-debt/README.md`(カバレッジ行・agmsg 仮定行のクローズ)
- 触らない: `scripts/ralph-orchestrator.sh` / `ralph-pipeline.sh` / `internal/ui` / `internal/state` / `.claude/skills/`

## Design decisions

- **フロー**: Standard flow (/work)(ユーザー確定)。アダプタ→spawn→実機検証の直列依存+実機スモークが対話的なため。
- **役割プロンプトの配置**: `internal/org/prompts/` + go:embed(ユーザー確定)。バイナリと雛形のバージョン不整合を排除。override は将来必要時に追加。
- **agmsg 呼び出しは Runner 経由の `bash <script> <args>`**: PR① の Runner/スタブ構造は維持し、argv 構築のみ差し替え(argv 検証容易性を保つ)。
- **join は spawn saga の正式ステップ**: HELLO send 単独から join.sh + HELLO へ。despawn は stop/disband の best-effort 補償(失敗しても state event は記録)。
- **Codex advisory 反映(4 件)**: ① `ensureLeadJoined` を saga に組込(手動 lead 登録依存の排除)、② stop/disband の実在座席前提条件(phantom state 防止)、③ `internal/org/protocol` バリデータ+`org send` 既定検証(文書だけの規約にしない)、④ 実機スモークにスコープ外変更の決定論チェック(PR④ までの最小ガード)。
- Critical forks: 上記 2 点(フロー・雛形配置)を解決済み — 残りは既存構造の延長で、追加のフォークなし。

## Acceptance criteria

- [ ] AC-1: Agmsg アダプタが実スクリプトを正しい argv で呼ぶ(`bash <home>/scripts/send.sh team from to msg` 等、全メソッドのスタブ argv 検証)。`AgmsgAvailable` は `<agmsg_home>/scripts/send.sh` 存在で判定し、npm ブートストラッパーのみの環境では不可用と報告する。
- [ ] AC-2: `[org] agmsg_home` が 3 面ロックステップ(config.go / templates/base/ralph.toml / ralph-config.sh)で追加され、`defaults_sync_test.go` が drift を検出する。
- [ ] AC-3: spawn saga の agmsg ステップが `join.sh <team> <seat_id> <type> <cwd>`(driver→type マッピング込み)+ HELLO send になり、join / send いずれの失敗注入でも `spawn_failed`+補償記録(PR① AC-10 系)が成立する。
- [ ] AC-4: `ralph org spawn --role reviewer|qa` で埋め込み雛形が初期プロンプトに展開される(変数置換のユニットテスト+スタブ AgentStart argv でプロンプト内容確認)。未知 role は雛形なし(--prompt のみ)で動作。
- [ ] AC-5: `stop` / `disband` が `despawn.sh` を best-effort 呼び出しし、失敗しても state event が記録される。
- [ ] AC-6: `Verbs.Send / Wait / Read` と CLI 配線のテストが追加され、カバレッジ 0% が解消(tech-debt 行クローズ)。
- [ ] AC-7: `.claude/rules/agent-messaging.md` が存在し、templates/base ミラーと同期(check-sync green)。役割雛形の本文がプロトコルを参照する。
- [ ] AC-8(実機スモーク・手動): **クリーンな agmsg team から手動 lead 登録なしに** reviewer 座席を実 spawn し、(1) herdr pane 起動と agent 状態遷移、(2) `team.sh` で lead+座席の参加確認、(3) `ralph org status` の spawned 表示、(4) `stop` + despawn、(5) スモーク前後の `git status --porcelain` 比較で宣言スコープ外の変更なし、の一連の証拠を `docs/evidence/org-seats-smoke-*.log` に保存する。
- [ ] AC-9: `go test ./...` / `./scripts/run-verify.sh` green。既存フロー非干渉の維持。
- [ ] AC-10(実在座席前提条件): 未知座席への `stop` は非ゼロ終了し state イベントを追記しない。`disband` は実在 active 座席のみ処理する。despawn 失敗時も `stopped` の Details に cleanup 結果が記録され status が真実を保つ。
- [ ] AC-11(protocol バリデータ): `internal/org/protocol` が TYPE 列挙・TASK_ID・本文サイズ上限を検証し、`ralph org send` が不正形式を既定拒否(`--raw` でバイパス可)する。パーサ/バリデータのユニットテストと CLI 拒否テストが存在する。

## Implementation outline

1. **Slice 1 — agmsg アダプタ書き直し**: argv 形式変更、agmsg_home 解決(env > config > 既定)、AgmsgAvailable 変更、doctor 追従、`[org] agmsg_home` ロックステップ、スタブ argv テスト全面更新。
2. **Slice 2 — spawn/stop への join/despawn 統合**: type マッピング、`ensureLeadJoined` → seat join → HELLO の saga ステップ差し替え(argv 順序のユニットテスト含む)、失敗注入テスト更新、stop/disband の despawn 追加と**実在座席前提条件**(未知座席 → 非ゼロ・state イベントなし)。
3. **Slice 3 — 役割プロンプト + プロトコル**: `internal/org/protocol/`(バリデータ+テスト)、`ralph org send` の既定検証+`--raw`、`internal/org/prompts/`(go:embed + 変数展開ローダ)、`spawn --role`、`.claude/rules/agent-messaging.md` + ミラー(バリデータを正として参照)。
4. **Slice 4 — verbs カバレッジ解消**: Send/Wait/Read のユニット+CLI テスト、tech-debt 行クローズ(phantom 座席行も AC-10 でクローズ)。
5. **Slice 5 — 実機スモーク + 証拠 + ドキュメント**: クリーン team からの手動スモーク実行(スコープ外変更チェック込み)、evidence 保存、AGENTS.md 等の最小追従、spec の PR② 境界注記(hook 検知の PR④ 移管)。

## Verify plan

- Static analysis checks: `./scripts/run-static-verify.sh`(golang)。
- Spec compliance criteria to confirm: spec FR-3(座席 = 常駐対話セッション: 実機スモークで挙動確認)、FR-4(typed protocol / identity)、FR-9 の join/despawn イベント記録。PR① AC 群にリグレッションがないこと。
- Documentation drift to check: `.claude/rules/agent-messaging.md` ミラー、tech-debt 行のクローズ整合、spec の PR② 境界注記、AGENTS.md。
- Evidence to capture: スタブ argv テスト出力、実機スモークログ、doctor 出力(agmsg 検出の新旧)。

## Test plan

- Unit tests: agmsg 各メソッドの argv(スクリプトパス含む)、agmsg_home 解決優先順位、type マッピング、雛形変数展開(未知変数・未知 role)、AgmsgAvailable(scripts あり/なし/ブートストラッパーのみ)。
- Integration tests: スタブ join/despawn を含む spawn→status→stop ライフサイクル、join 失敗注入 → spawn_failed+補償、`--role` 付き spawn の AgentStart argv 内容。
- Regression tests: PR① の全テスト、defaults_sync_test、check-sync / check-skill-sync。
- Edge cases: agmsg_home 不在、despawn 失敗時の stop 継続、role 雛形と `--prompt` の併用、team 名バリデーション(agmsg 側 validate と整合)。
- Evidence to capture: `go test ./...` / `run-test.sh` 出力、実機スモークログ。

## Risks and mitigations

| リスク | 影響 | 緩和 |
|---|---|---|
| agmsg スクリプト仕様の将来変化(活発に開発中) | アダプタ破損 | `VERSION` ファイル読み取りを doctor に追加し想定バージョン外で warn。呼び出しはアダプタ 1 ファイルに集約済み |
| 実機スモークの対話コスト(billed セッション) | コスト | 最短プロンプト(応答後待機)で 1 回、stop まで数分で完了 |
| join.sh の roster 検証(未登録 from/to 拒否)との整合 | send 失敗 | saga で join → send の順序を保証。lead identity の登録はスモーク手順に含め、PR③ で `org start` に正式統合 |
| despawn 追加による stop の挙動変化 | 既存テスト破損 | best-effort(失敗無視+記録)とし、state event 契約は不変 |

## Rollout or rollback notes

- 引き続き純追加系(既存フロー非干渉)。ロールバックは revert のみ。
- `[org] agmsg_home` は既定値で従来と同経路のため、下流プロジェクトへの影響なし。

## Open questions

- 座席の delivery mode を spawn 時に明示設定するか(`delivery.sh set`)— 実機スモークで既定挙動を観察して判断し、plan に追記。
- protocol バリデータの本文サイズ上限の初期値(仮: 2,000 文字。EVIDENCE はポインタ原則のため十分)— 実機スモークで調整。

(解決済み: lead identity 登録は Codex 所見 1 により `ensureLeadJoined` として saga に組み込み。lead の agmsg type は `claude-code`、project_path はリポジトリルート。)

## Implementation notes (deviations)

- **実機スモークが herdr アダプタの統合バグを検出**(Slice 5、2026-08-02): 実 herdr CLI(v0.7.5)は全コマンドで JSON エンベロープ(`{"id":...,"result":{...}}` / `{"error":{...}}`)を返すが、PR① アダプタは「trimmed stdout = ID」と仮定していたため、`WorkspaceCreate` の戻り値(JSON 全文)を `tab create --workspace` に渡して `workspace_not_found` で spawn が失敗した。修正: herdr アダプタに JSON エンベロープ解析を追加(`workspace create` → `result.workspace.workspace_id`、`tab create` → `result.root_pane.pane_id`、非 JSON 出力は従来どおり trimmed raw にフォールバック)。`pane read` はプレーンテキストで従来どおり。CLI スタブは実キャプチャ JSON を出力するよう更新し、パーサを実形状でテストする。

## Progress checklist

- [ ] Plan reviewed
- [x] Branch created (feat/org-runtime-seats)
- [ ] Implementation started
- [ ] Review artifact created
- [ ] Verification artifact created
- [ ] Test artifact created
- [ ] PR created

## Readiness checklist

- [x] agmsg 実機インターフェース検証済み(v1.1.13、スクリプトシグネチャ確認済み)
- [x] herdr 導入済み(doctor pass)
- [x] フロー確定(Standard /work)・雛形配置確定(go:embed)
- [ ] Codex plan advisory(次ステップ)
