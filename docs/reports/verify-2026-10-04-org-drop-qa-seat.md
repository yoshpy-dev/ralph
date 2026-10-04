# Verify report: org-drop-qa-seat

- Date: 2026-10-04
- Plan: docs/plans/active/2026-10-04-org-drop-qa-seat.md
- Verifier: verifier subagent (Claude)、cycle 1
- Scope: `git diff 4ee080f5...HEAD`(HEAD 3c174e98、branch refactor/org-drop-qa-seat、origin/main を 28cc057c で取り込み済み)の 52 ファイル。AC-1〜AC-15 の充足、文書のずれ、静的解析。テストは実行していない(/test の担当)
- Evidence: `docs/evidence/verify-2026-10-04-org-drop-qa-seat.log`(gitignore 済みでローカルのみ。`run-static-verify.sh` 自身のログは `docs/evidence/verify-2026-10-04-114450.log`)

## Spec compliance

テストの存在と中身はコードを読んで確かめた。テストが通るかどうかは /test で確かめる。

| Acceptance criterion | Status | Evidence |
| --- | --- | --- |
| AC-1 qa 雛形の撤去 | PASS | `git ls-files internal/org/prompts/` は `implementer.md` / `leader.md` / `reviewer.md` の 3 つだけ。`TestRenderRolePrompt_QA_NoTemplate`(`internal/org/prompts_test.go:196`)が `ok=false`、空文字列、err なしを検査している |
| AC-2 reviewer のミッション節 | PASS | `internal/org/prompts/reviewer.md` のミッション節に (a) 項目 1 の 2 本のスクリプト、(b) 項目 2 の `GATE: fail` と「差分レビューに進まない」、(c) 項目 3 の `GATE: unrunnable` と「差分レビューに進まない」、(d) 項目 4 の差分品質・受け入れ基準・ゲートの結果の統合・`GATE: pass` がある。`QA 座席` / `qa 座席` は雛形 3 つのどこにもない(grep rc=1)。テストは `TestRenderRolePrompt_Reviewer_MissionRunsGateFirstAndBlocksWithoutReviewing`(`prompts_test.go:313`)で、`markdownSection` と `markdownItem` でミッション節の項目ごとに検査する。mutation の結果は計画の「実装中の逸脱」(Slice A、90488a06)に 9 通り記録されている。私は再実行していない(テストの実行になるため)。代わりに、mutation の対象の文言がミッション節に何回出るかを数え、どれもテストが見る項目にしかないことを確かめた(下の Observational checks) |
| AC-3 leader のミッション節 | PASS | `internal/org/prompts/leader.md` のミッション節の冒頭がレビューと検証(ゲートの再実行を含む)を reviewer に委譲し、項目 5 の下で `GATE: fail` を implementer に差し戻し、`GATE: unrunnable` を「implementer には戻さず」leader が直すか「人に上げる」。テストは `TestRenderRolePrompt_Leader_MissionRoutesGateBlocked`(`prompts_test.go:356`) |
| AC-4 `--role qa` の拒否 | PASS | `TestOrgSpawn_RemovedRole_WithoutPrompt_RejectedBeforeAnyManifestWrite`(`internal/org/spawn_test.go:2958`)が real と dry-run の両方で、エラー文の `reviewer` / `--prompt` / `qa`、driver の呼び出しなし、manifest の event なし、receipts なし、ゼロ値の `ModelReceipt` を検査する。`TestOrgSpawn_RemovedRole_WithPrompt_StartsWithPromptOnly`(`:3008`)が、`--prompt` を付けると起動し、初期プロンプトが `--prompt` だけになること(inline、prompt file、dry-run)を検査する |
| AC-5 qa の grep | PASS | 下の「AC-5 の grep」のとおり、12 ファイルが許可リストと一致する |
| AC-6 `/org` skill の 4 面 | PASS | 4 面は `cmp` で一致。Leaded 行(`SKILL.md:154`)は reviewer だけ、fan-out の例(`:165`、`:172-173`)は implementer / reviewer、役割リスト(`:180-190`)は leader / implementer / reviewer で、reviewer の説明にゲートの再実行と `guarded` の注意がある。「上記 3 役割」(`:192`)、`--role qa` の拒否(`:198-199`) |
| AC-7 quality-gates / ralph.toml / tech-debt | PASS | `docs/quality/quality-gates.md:79` と template の `:78` が「impl exit checks → reviewer (re-runs … first) → leader arbitration」と「driven by the role prompt templates, not enforced mechanically」を書いている(template は計画どおり `(see docs/tech-debt)` を外している)。`templates/base/ralph.toml:81-85` の `reviewer = "guarded"` の例に注記があり、`:13` は "Leader autonomy"。`docs/tech-debt/README.md:152` に hook 強制が未実装である行がある |
| AC-8 spec の改訂 | PASS | `docs/specs/2026-08-01-org-runtime.md:15-22` に 2026-10-04 改訂の節がある。(a) が 2026-09-16 改訂 (b) の「4 種」を置き換え、(c) が `lead` から `leader` への改名を書く。FR-4(`:59`)、FR-7(`:62`)、AC(`:83`、計画を書いた時点の 74 行)に改訂の印がある |
| AC-9 reviewer 雛形の smoke | DEFERRED(/test) | `claude -p` の smoke は /test で行う。この verify の判定には含めない |
| AC-10 `go test ./...` と `run-verify.sh` | 静的な部分は PASS、テストは /test | `RALPH_VERIFY_SCOPE=full ./scripts/run-static-verify.sh` は exit 0。`go test ./...` と `run-verify.sh` のテストの部分は /test で確かめる |
| AC-11 leader の識別子 | PASS | `internal/org/spawn.go:61` は `const LeaderIdentity = "leader"`。`leader.md` はあり、`lead.md` はない。テスト: `TestLeaderIdentity_ConstantValue`(`spawn_test.go:1631`)、`ralph org start` が seat_id と role の両方を `leader` にする `TestOrgStart_HappyPath_SpawnsLeaderSeat_SingleAgmsgJoin_NoHello`(`internal/cli/org_test.go:2391`)、agmsg の登録 `TestOrgSpawn_EnsureLeaderJoined_*`(`spawn_test.go:1637`、`:1655`)、watchdog の ALERT の宛先 `TestNewWatchdogHooks_AbnormalVerdict_SendsAlertToLeader`(`org_test.go:2883`、`watchdog leader TYPE: ALERT` を検査)。本番側は `watch.go:701` が `LeaderIdentity` に送る |
| AC-12 Go の識別子の grep | PASS | 下の「AC-12 の grep」のとおり、残るのは永続化された JSON のタグとそのテストのフィクスチャだけ |
| AC-13 旧名の拒否 | PASS | (a) `TestOrgSpawn_RetiredLeaderName_RejectedBeforeAnyManifestWrite`(`spawn_test.go:2795`)が `--role lead`、`--prompt` 付きの `--role lead`、`--id lead`、`--id lead --role leader --prompt` を real と dry-run の両方で検査し、エラー文の `leader`、manifest と receipts に何もないことを確かめる。CLI 経由は `TestOrgSpawn_RetiredLeaderRoleAndID_RejectedThroughCLI`(`org_test.go:3133`)。(b) 同じテストの 2 つのキーのケース、`TestOrgSpawnAndStart_RetiredLeaderConfigKey_Rejected`(`org_test.go:3037`)、status / stop / disband が動く `TestOrgCleanupVerbs_RetiredLeaderConfigKey_StillWork`(`:3088`)、doctor の warn の `TestCheckOrgRetiredRoleKeys` と `TestRunDoctorOpts_RetiredRoleKey_WarnsWithoutFailing`(`internal/cli/doctor_org_test.go:1134`、`:1197`)。(c) `TestOrgSpawn_DeprecatedDriverFlagAlias`(`org_test.go:3293`)が、旧フラグだけ・新フラグだけ・両方で同じ値・両方で違う値(順序 2 通り)を検査し、stderr の deprecation の通知と、矛盾したときに manifest と herdr に何もないことを確かめる |
| AC-14 lead の grep | PASS | 下の「AC-14 の grep と分類」のとおり、115 行がすべて計画の分類のどれかに入る |
| AC-15 leader を使う文書と同期 | PASS | `agent-messaging.md` の 2 面は `cmp` で一致し、`TO: leader` が各 2 か所。`AGENTS.md`、`.ralph/core/AGENTS.core.md`(+ template、`cmp` で一致)、`README.md`、`codex-seat-permissions.md`(2 面、`cmp` で一致)、`quality-gates.md`(2 面)が `leader` を使い、AC-14 の grep に掛かる行はない。`./scripts/check-skill-sync.sh` と `./scripts/check-sync.sh` は exit 0 |

### AC-5 の grep

`git grep -n -w -i qa -- . ':!docs/plans' ':!docs/reports' ':!docs/evidence' ':!docs/research'`(rc=0)。spec と tech-debt の行は数千字あるので 160 字で切った。切っていない出力は evidence のログにある。

```
.agents/skills/org/SKILL.md:198:- `qa` の雛形は撤去した(ゲートの再実行は reviewer に移った)。`--role qa` は
.claude/skills/org/SKILL.md:198:- `qa` の雛形は撤去した(ゲートの再実行は reviewer に移った)。`--role qa` は
docs/insights/events/2026-10-04-org-drop-qa-seat.jsonl:1:{"schema":1,"ts":"2026-10-04T10:58:52Z","slug":"org-drop-qa-seat","flow":"standard","phase":"self_revie …(truncated)
docs/specs/2026-08-01-org-runtime.md:5:Ralph Loop(/loop)の自律実行系を撤去し、Lead LLM が herdr(実行・観測)と agmsg(メッセージング)を使って常駐 LLM 座席(scout / impl / reviewer / QA / watchdog) …(truncated)
docs/specs/2026-08-01-org-runtime.md:10:- (b) 役割は lead / implementer / reviewer / qa の 4 種(FR-4 の identity 例 `impl-<slug>` は `implementer-<n>` 等の任意 seat id で運用) …(truncated)
docs/specs/2026-08-01-org-runtime.md:15:### 2026-10-04 改訂(refactor/org-drop-qa-seat)
docs/specs/2026-08-01-org-runtime.md:17:- (a) 役割は leader / implementer / reviewer の 3 種にする(2026-09-16 改訂 (b) の「4 種」を置き換える)。qa 座席の雛形(`internal/org/prompts/qa.md` …(truncated)
docs/specs/2026-08-01-org-runtime.md:20:- (d) `--role qa` は `--prompt` がなければ拒否し、reviewer を案内する。`--prompt` があれば、雛形のない独自の役割として起動する。
docs/specs/2026-08-01-org-runtime.md:22:- 本文の履歴の記述(Summary の「QA」、FR-4 の `lead` / `qa`、FR-7 の QA 座席、AC の QA 座席)は書き換えず、該当行に改訂の印を付ける。
docs/specs/2026-08-01-org-runtime.md:59:- [ ] **FR-4 agmsg スター型プロトコル**: identity は `lead` / `scout-<n>` / `impl-<slug>` / `reviewer` / `qa` / `watchdog`。typed m …(truncated)
docs/specs/2026-08-01-org-runtime.md:62:- [ ] **FR-7 新品質パイプライン(4フェーズ)**: ① impl 座席が退出チェック(スコープ内・コミット境界・verify / test のローカル通過)付きで RESULT 報告 → ② QA 座席が `run-stati …(truncated)
docs/specs/2026-08-01-org-runtime.md:66:- [ ] **FR-11 Ralph Loop 自律実行系の完全撤去**(段階移行なし・PR 系列の最終 PR で一括削除。**適用範囲確定(PR⑤ 計画時のユーザー確定判断)**: 撤去対象は Ralph Loop の自律実行系のみ。標 …(truncated)
docs/specs/2026-08-01-org-runtime.md:83:- [ ] Given impl 座席の RESULT、when QA 座席のゲートが fail、then reviewer 座席にレビューが渡らず impl に差し戻される(lead 経由)。 **(2026-10-04 改訂で変更: ゲ …(truncated)
docs/specs/2026-08-01-org-runtime.md:177:- 座席の worktree 割当粒度(impl 座席のみ worktree 必須か、QA / reviewer は統合ブランチ read-only チェックアウトで足りるか)。
docs/tech-debt/README.md:130:| ~~org-implementer-seat-envelope: deferred LOW findings batch (cosmetic) — (1) `codexModelsCachePath` discards `os.UserHomeDir`'s  …(truncated)
docs/tech-debt/README.md:152:| org runtime FR-7 quality gate is prompt-instructed, not hook-enforced. The reviewer seat re-runs `run-static-verify.sh` / `run-te …(truncated)
docs/tech-debt/README.md:153:| CLI `--model` fallback runs before the ralph.toml retired-key check. With an `[org.roles].lead` or `[org.permissions.roles].lead` …(truncated)
internal/cli/org_test.go:3174:const removedRoleName = "qa"
internal/cli/status_test.go:30:// org-b has one seat (qa active).
internal/cli/status_test.go:42:		{TS: "2026-08-01T00:00:00Z", OrgID: "org-b", SeatID: "qa", Event: org.EventSpawned, Role: "qa", Driver: "claude", Model: "sonne …(truncated)
internal/cli/status_test.go:62:		"leader", "reviewer", "qa",
internal/cli/status_test.go:64:		"active 1/1", // org-b: qa active
internal/cli/status_test.go:87:	if strings.Contains(out, "qa") {
internal/cli/status_test.go:104:		{TS: "2026-08-01T00:01:00Z", OrgID: "org-c", SeatID: "shadow", Event: org.EventSpawned, Role: "qa", Driver: "codex", Model: "s …(truncated)
internal/org/prompts.go:131:	"qa":   {Successor: "reviewer", Kind: retiredRoleRemoved},
internal/org/prompts_test.go:197:	// The qa seat template was retired: its deterministic-gate re-run moved
internal/org/prompts_test.go:198:	// into the reviewer template, so "qa" is an ordinary role with no template.
internal/org/prompts_test.go:204:		t.Fatalf("RenderRolePrompt: expected no error for the retired qa role, got %v", err)
internal/org/prompts_test.go:207:		t.Fatal("expected ok=false: the qa seat template no longer exists")
internal/org/prompts_test.go:210:		t.Fatalf("expected empty text for the retired qa role, got %q", text)
internal/org/prompts_test.go:348:	// The retired qa seat is not a collaborator any more.
internal/org/prompts_test.go:349:	for _, banned := range []string{"QA 座席", "qa 座席"} {
internal/org/prompts_test.go:386:	// The retired qa seat is not a delegation target any more.
internal/org/prompts_test.go:387:	for _, banned := range []string{"qa 座席", "QA 座席"} {
internal/org/prompts_test.go:461:const removedRoleName = "qa"
internal/org/spawn_test.go:3013:		p.Prompt = "custom qa instructions"
internal/org/spawn_test.go:3031:		p.Prompt = "custom qa instructions\nsecond line"
internal/org/spawn_test.go:3057:		p.Prompt = "custom qa instructions"
internal/org/spawn_test.go:3105:		p.Prompt = "custom qa instructions"
templates/base/.agents/skills/org/SKILL.md:198:- `qa` の雛形は撤去した(ゲートの再実行は reviewer に移った)。`--role qa` は
templates/base/.claude/skills/org/SKILL.md:198:- `qa` の雛形は撤去した(ゲートの再実行は reviewer に移った)。`--role qa` は
```

ファイルの一覧(`git grep -l`)は 12 個で、計画の許可リストと 1 対 1 で対応する。`internal/org/prompts.go`(表)、`internal/org/prompts_test.go`、`internal/org/spawn_test.go`、`internal/cli/status_test.go`(Non-goals)、`internal/cli/org_test.go`(Slice D で追加)、`/org` skill の 4 面、spec、`docs/tech-debt/README.md`、`docs/insights/events/2026-10-04-org-drop-qa-seat.jsonl`。表は `prompts.go` にあるので、読み替えは要らない。

### AC-12 の grep

`git grep -n -P 'Lead(?!e)|\blead[A-Z_]' -- '*.go'`(rc=0)

```
internal/org/watch.go:230:	LeaderAgentGet     string `json:"lead_agent_get"`     // tag predates the leader rename so older state decodes; renaming it would zero the value
internal/org/watch_test.go:947:// escalating. The fixture's "lead_agent_get": "" and "history_lead_lines":
internal/org/watch_test.go:1009:      "lead_agent_get": "",
internal/org/watch_test.go:1896:      "lead_agent_get": "",
```

4 行とも、計画が例外とした永続化された JSON のタグ(`watch.go:230`)と、それを使うテストのフィクスチャとコメント。`history_lead_lines`(`watch.go:231`)は、`_` が単語の文字なので `\blead` に掛からない。旧名の拒否のための Go の識別子は `deprecatedLeaderDriverFlag` のように `Leader` を含む名前なので、この正規表現には掛からない。

補足として、単語境界に頼らない検索(`git grep -n -i -P 'lead(?!er|ed\b|s\b|ing)'` を履歴の場所を除いて実行し、`-w lead` に掛かる行を除いたもの)も行った。残ったのは `watch.go:230-231` と `watch_test.go:947,1009-1010,1896-1897` の 7 行だけで、どれも同じ JSON のタグだった。`<org_id>_lead` のような herdr 名のフィクスチャは残っていない。

### AC-14 の grep と分類

`git grep -n -w -i lead -- . ':!docs/plans' ':!docs/reports' ':!docs/evidence' ':!docs/research'`(rc=0、115 行)。docs と insights 以外の 34 行は次のとおり(160 字で切った)。

```
.agents/skills/org/SKILL.md:200:- 指示役の旧名 `lead` は `leader` に改めた。`--role lead`、`--id lead`、
.agents/skills/org/SKILL.md:201:  `ralph.toml` の `[org.roles].lead` / `[org.permissions.roles].lead` は spawn
.agents/skills/org/SKILL.md:202:  で拒否され(`ralph doctor` も warn を出す)、`--lead-driver` は
.claude/skills/org/SKILL.md:200:- 指示役の旧名 `lead` は `leader` に改めた。`--role lead`、`--id lead`、
.claude/skills/org/SKILL.md:201:  `ralph.toml` の `[org.roles].lead` / `[org.permissions.roles].lead` は spawn
.claude/skills/org/SKILL.md:202:  で拒否され(`ralph doctor` も warn を出す)、`--lead-driver` は
internal/cli/insights_test.go:244:		SeatID:         "lead",
internal/cli/insights_test.go:257:	if !strings.Contains(out, "ORG demo") || !strings.Contains(out, "SEAT lead") {
internal/cli/insights_test.go:299:// contract from the plan (AC-3): "ORG demo  SEAT lead  commanded=opus
internal/cli/insights_test.go:315:			SeatID:         "lead",
internal/cli/insights_test.go:327:	want := "ORG demo  SEAT lead  commanded=opus  honored: true=3 false=1 unknown=2  rate=75% (unknown 2 excluded)"
internal/cli/org.go:281:const deprecatedLeaderDriverFlag = "lead-driver"
internal/cli/org.go:286:// --leader-driver claude (the default value) next to --lead-driver codex is
internal/cli/org.go:412:// (*org.Org).Spawn, per the plan's design decision ("`org start` = lead 座席
internal/cli/org.go:413:// の spawn 糖衣", docs/plans/active/2026-08-02-org-runtime-lead.md). It
internal/cli/org_test.go:3008:const oldLeaderName = "lead"
internal/cli/org_test.go:3292:// --lead-driver codex); the same value on both is fine.
internal/org/prompts.go:130:	"lead": {Successor: LeaderIdentity, Kind: retiredRoleRenamed},
internal/org/prompts.go:136:// "[org.permissions.roles].lead" and RenameTo
internal/org/prompts.go:151:// `lead = "guarded"` that no longer applies would silently run the leader
internal/org/prompts_test.go:456:const oldLeaderName = "lead"
internal/org/report.go:11:// (AC-4, FR-9 後半 -- see docs/plans/active/2026-08-02-org-runtime-lead.md).
internal/org/spawn.go:59:// docs/plans/active/2026-08-02-org-runtime-lead.md) without a duplicate
internal/org/spawn.go:583:		// fix (docs/reports/cross-review-triage-org-runtime-lead.md,
internal/org/spawn.go:818:	// docs/plans/active/2026-08-02-org-runtime-lead.md, "Design decisions").
internal/org/spawn.go:1004:// `lead = "guarded"` ignored that way would run the leader with the full
internal/org/spawn.go:1074:// docs/tech-debt/README.md, "lead identity is a bare 'lead' string literal
internal/org/spawn_test.go:545:// test for cross-review-triage-org-runtime-lead.md ACTION_REQUIRED #1: under
templates/base/.agents/skills/org/SKILL.md:200:- 指示役の旧名 `lead` は `leader` に改めた。`--role lead`、`--id lead`、
templates/base/.agents/skills/org/SKILL.md:201:  `ralph.toml` の `[org.roles].lead` / `[org.permissions.roles].lead` は spawn
templates/base/.agents/skills/org/SKILL.md:202:  で拒否され(`ralph doctor` も warn を出す)、`--lead-driver` は
templates/base/.claude/skills/org/SKILL.md:200:- 指示役の旧名 `lead` は `leader` に改めた。`--role lead`、`--id lead`、
templates/base/.claude/skills/org/SKILL.md:201:  `ralph.toml` の `[org.roles].lead` / `[org.permissions.roles].lead` は spawn
templates/base/.claude/skills/org/SKILL.md:202:  で拒否され(`ralph doctor` も warn を出す)、`--lead-driver` は
```

残りの 81 行は file:line だけ挙げる(行全体は evidence のログにある)。`docs/insights/events/2026-08-02-org-runtime-lead.jsonl:1-8`、`docs/specs/2026-08-01-org-runtime.md:5,10,19,22,46,48,59-64,78-81,83,84,91,93,94,101,146,147,158`、`docs/tech-debt/README.md:59-61,65-69,71-74,76,78-83,96,129,130,153`、`internal/insights/insights_test.go`(18 行)、`internal/insights/testdata/receipts.jsonl:1-6,9`。

| 分類 | 行数 | ファイルと理由 |
| --- | --- | --- |
| 旧名の拒否のコードとテスト | 9 | `internal/cli/org.go:281,286`(`--lead-driver` の非推奨の別名)、`internal/cli/org_test.go:3008,3292`(拒否と別名のテスト)、`internal/org/prompts.go:130,136,151`(撤去・改名した役割の表と `RetiredRoleConfigKeys` の doc)、`internal/org/prompts_test.go:456`(テストの定数)、`internal/org/spawn.go:1004`(`retiredRoleConfigErr` の doc) |
| 過去のデータのフィクスチャ | 30 | `internal/insights/testdata/receipts.jsonl`(7)、それを読む `internal/insights/insights_test.go`(18)、過去の receipts の形の行を作る `internal/cli/insights_test.go`(5) |
| 履歴の記録 | 55 | `docs/insights/events/2026-08-02-org-runtime-lead.jsonl`(8)。spec(25): 19 と 22 は今回の改訂の節、59 / 62 / 83 は改訂の印を足した履歴の行、残りは書き換えていない履歴の記述。`docs/tech-debt/README.md` の既存の行(22。この branch の tech-debt の diff は追加 2 行だけで、既存の行は変えていない) |
| 過去の計画・レポートのファイル名と文言の引用 | 8 | `internal/cli/org.go:412-413`、`internal/org/report.go:11`、`internal/org/spawn.go:59,583,818,1074`、`internal/org/spawn_test.go:545`。計画が Slice 0 の時点の行番号で挙げた箇所と同じで、spawn.go の 534 / 769 / 950 が 583 / 818 / 1074 に、org.go の 369-370 が 412-413 にずれた(`git show eb30b172:internal/org/spawn.go` の 950 行目が今の 1074 行目と同じ文であることを確かめた) |
| 永続化された state のキー | 0 | `-w lead` には掛からない(AC-12 の節を参照) |
| 文書の中の旧名の案内 | 13 | `/org` skill の 4 面の 200-202 行(12)と、`docs/tech-debt/README.md:153`(Slice D の後に足した行で、ralph.toml の旧キー `[org.roles].lead` を名指しする) |

合計は 9 + 30 + 55 + 8 + 0 + 13 = 115 で、grep の行数と一致する。`docs/tech-debt/README.md:153` は「既存の行」ではないので「文書の中の旧名の案内」に入れた。旧キーの名前を説明する行で、旧名を現役の名前として使ってはいない。

## Static analysis

| Command | Result | Notes |
| --- | --- | --- |
| `RALPH_VERIFY_SCOPE=full ./scripts/run-static-verify.sh` | PASS (exit 0) | shellcheck、`sh -n`(hooks 2 面)、`jq -e`(settings 2 面)、Codex の hook のガード 3 本、`check-sync.sh`、`check-pipeline-sync.sh`、`check-skill-sync.sh`、`check-template-purity.sh`、golang verifier(`gofmt: ok`、golangci-lint `0 issues.`)、branch secret scan(`scanned 4ee080f5..3c174e98 against origin/main: clean`)。最後の行は `==> All verifiers passed.` |
| `./scripts/check-skill-sync.sh` | PASS (exit 0) | `[ok] check-skill-sync: 13 skill(s) in lock-step` |
| `./scripts/check-sync.sh` | PASS (exit 0) | IDENTICAL 159、DRIFTED 0、ROOT_ONLY 0、TEMPLATE_ONLY 11、KNOWN_DIFF 5、`PASS: all files in sync.`。`quality-gates.md` は以前から KNOWN_DIFF で、今回の行の差は template から `(see docs/tech-debt)` を外した分だけ |
| `go vet ./...` | PASS (exit 0) | 出力なし |
| `gofmt -l internal cmd` | PASS (exit 0) | 出力なし。計画の `gofmt -l .` も出力なし |
| `go build ./...` | PASS (exit 0) | 補足として実行した。`git status` は実行の前後とも clean |

## Documentation drift

| Doc / contract | In sync? | Notes |
| --- | --- | --- |
| `AGENTS.md` の Repo map(`internal/org/`) | Yes | 「role prompt templates (go:embed)」は役割の数を書いていないので、直す必要はない。Primary loop の org runtime の行は `leader` |
| `CLAUDE.md` | Yes | `lead` / `qa` への言及はない |
| `.claude/rules/ralph/model-routing.md` の org runtime の節 | Yes | 役割名を挙げていない。AC-5 と AC-14 の grep にも掛からない |
| `docs/recipes/` の org 関連 | Yes | `codex-seat-permissions.md` は 2 面とも `leader` で、`cmp` で一致。ほかの recipe に `lead` / `qa` は残っていない(補足の検索を含む) |
| `/org` skill の動詞の例 | Yes | `read` / `stop` の例は `reviewer-1`、`start` の例は `spawn --role leader` / `leader.md`、トポロジの節は `TO: leader`(`SKILL.md:235`)。description も "Leader's operating manual" |
| `ralph org start` の help(`internal/cli/org.go`) | Yes | seat id と role の `"leader"`、`internal/org/prompts/leader.md` を書いている |
| `agent-messaging.md`(2 面) | Yes | `TO: leader`、`ensureLeaderJoined`、seat id の例は `implementer` / `reviewer` |
| spec の FR-11(`:66`)と Open questions(`:177`) | 軽微なずれ | 「reviewer/qa 座席」「QA 座席」が改訂の印なしに残る。AC-8 が印を求めるのは FR-4・FR-7・AC だけなので AC には反する点はない。ただ、改訂の節の `:22` は書き換えない履歴の記述として Summary・FR-4・FR-7・AC だけを挙げていて、この 2 か所が抜けている。LOW。直すなら `:22` に「FR-11、Open questions」を足す |
| 計画の Progress checklist | 軽微なずれ | self-review のレポート(`docs/reports/self-review-2026-10-04-org-drop-qa-seat.md`)はあるが、「Review artifact created」が未チェックのまま。Status も `In progress`。AC のチェックは実態と合っている(AC-9 だけ未チェック) |
| `docs/tech-debt/README.md:152` の関連の列 | 注意 | `docs/plans/active/2026-10-04-org-drop-qa-seat.md` を指している。`/pr` で計画が archive に移るとリンクが古くなる。同じファイルの既存の行も 31 か所 `docs/plans/active/` を指しているので、この台帳の慣習には合っている。直すかどうかは /sync-docs か /pr の判断 |

## Observational checks

- AC-2 / AC-3 の mutation(静的に確かめた範囲): ミッション節を awk で切り出し、mutation の対象の文言の出現回数を数えた。reviewer: `./scripts/run-static-verify.sh` 1、`./scripts/run-test.sh` 1、`GATE: fail` 1、`GATE: unrunnable` 1、`差分レビューに進まない` 2(項目 2 と 3 に 1 つずつ)、`GATE: pass` 1。leader: `reviewer 座席へ委譲` 1、`GATE: fail` 1、`GATE: unrunnable` 1、`implementer 座席に差し戻` 1、`implementer には戻さず` 1、`人に上げる` 1。`markdownItem` は、マーカーを含む最初の行から次の `- ` か番号付きの項目の手前までを返す。どの文言もテストが見る項目の中に 1 回しかないので、1 つ消せばその検査が落ちる、とコードを読んで判断した。実行はしていない
- self-review の Addendum の C2-1〜C2-4 は 14922f64 に入っている。`verbs_test.go:1684` は段番号ではなく `ValidateSpawnEnvelope` の名前で書かれ、`spawn.go:264`、`:433`、`:470`、`:495` が旧キーの検査を一覧に含め、`reviewer.md:9-12` は 1 文になり、`docs/tech-debt/README.md:153` が C2-4 を記録している
- 計画の Regression tests にある既存のテスト `TestOrgSpawn_UnknownRole_NoTemplateApplied`(`internal/cli/org_test.go:1492`)と、reviewer の雛形の後ろに `--prompt` を連結するテスト `TestOrgSpawn_RoleTemplate_PromptFlagAppendedAfterTemplate`(`spawn_test.go:1215`、`p.Role = "reviewer"`)は残っている

## Coverage gaps

- テストは実行していない。AC-1〜AC-4、AC-11、AC-13 の判定は、テストがあることと中身を読んだ範囲のもので、通るかどうかは /test で確かめる
- AC-2 / AC-3 の mutation は再実行していない。計画の記録(9 通り)と、上の文言の出現回数の確認に頼っている
- AC-9(`claude -p` の smoke)は /test に回した
- 計画の Open questions(codex の `guarded` の reviewer がゲートを許可なしに実行できるか)は、この PR では確かめない。文書(`SKILL.md:189-190`、`ralph.toml:82-85`)は「未確認」と書いている
- 決定論的なチェックの不足: AC-5 / AC-12 / AC-14 の grep は手で実行するもので、CI では回らない。次の改名で `lead` / `qa` が戻ってもどこでも止まらない。足すなら、AC-12 の `git grep -P` を JSON のタグの 2 行だけを許して実行する Go のテストか `scripts/` のチェックが一番小さい

## Verdict

- Verified: AC-1、AC-2、AC-3、AC-4、AC-5、AC-6、AC-7、AC-8、AC-11、AC-12、AC-13、AC-14、AC-15(テストの中身を読んだ範囲を含む)。静的解析 6 本がすべて exit 0
- Partially verified: AC-10(静的な部分だけ。`go test ./...` と `run-verify.sh` のテストの部分は /test)
- Not verified: AC-9(/test に回した)
- Overall: PASS。spec の FR-11 / Open questions の印、計画の Progress checklist、tech-debt の行のリンクの 3 点は LOW のずれで、マージを止めるものではない
