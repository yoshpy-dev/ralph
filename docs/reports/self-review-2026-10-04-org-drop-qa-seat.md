# Self-review report: org-drop-qa-seat

- Date: 2026-10-04
- Plan: docs/plans/active/2026-10-04-org-drop-qa-seat.md
- Branch: refactor/org-drop-qa-seat(HEAD 059cbe01、base main 11602fed)
- Reviewer: reviewer subagent (Claude)、cycle 1
- Scope: diff の品質だけ(命名、読みやすさ、不要な変更、誤字、null 安全、デバッグコード、秘密情報、エラー処理、セキュリティ、保守性)。仕様への適合、テストの網羅、文書のずれは /verify・/test・/sync-docs の担当。対象は `git diff 11602fed...HEAD` の 50 ファイルのうち、`docs/plans/` を除いたもの

## Evidence reviewed

- 挙動を変えるコード: `internal/org/prompts.go`(撤去・改名した役割の表と `RetiredRoleConfigKeys`)、`internal/org/spawn.go`(`retiredRoleSpawnErr` と Spawn の先頭への差し込み)、`internal/cli/org.go`(`resolveLeaderDriver`、`--lead-driver` の別名)、`internal/cli/doctor.go`(`checkOrgRetiredRoleKeys`)の全 hunk
- 改名のコミット(eb30b172)の Go 側: `watch.go`、`watcher.go`、`verbs.go`、`permissions.go`、`envelope_summary.go`、`statedir.go`、`config.go`、`doctor_codex_writable_root.go` の hunk と、テスト側の rename hunk
- 改名の取りこぼしと巻き込みの確認: `git grep -n -w -i lead -- '*.go' 'internal/org/prompts/*'` で残るのは、旧名の拒否のコードとテスト、insights の過去の receipts、過去の計画・レポートのファイル名と引用、永続化された JSON のタグだけだった。追加行の `leader` を含む識別子を数え(`grep -o -i '[a-z_]*leader[a-z_]*' | sort | uniq -c`)、`Leaderer` のような二重置換や、関係のない語への置換がないことを確かめた
- 雛形: `reviewer.md`、`leader.md`、`implementer.md` の diff と、削除した `qa.md`
- 新しいテスト: `spawn_test.go` の `TestOrgSpawn_RetiredLeaderName_*` / `TestOrgSpawn_RemovedRole_*`、`prompts_test.go` の `markdownItem` と mission 節のテスト、`org_test.go` の CLI テスト、`doctor_org_test.go` の 2 本
- 文書: `/org` skill(`.claude` 側)、`agent-messaging.md`、`README.md`、spec、`quality-gates.md`、`codex-seat-permissions.md`、`templates/base/ralph.toml`、`docs/tech-debt/README.md` の追加行(区切りの `|` は 6 個で列は崩れていない)
- 秘密情報、デバッグ出力、コメントアウトしたコード、TODO は追加行に見当たらない。`panic(err)` は `MarkDeprecated` の直後の 1 か所で、直前の行で登録したフラグが見つからないときにだけ起きる

## Findings

| Severity | Area | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- |
| MEDIUM | maintainability | M-1: 旧名の検査が idempotent な早期 return より前にあり、Spawn が明文化している順序の不変条件を破っている。`retiredRoleSpawnErr` は manifest を読む前に走るので、spawn 済みの座席の再実行(本来は `SpawnOutcomeIdempotent`)でも、ralph.toml に `[org.roles].lead` などがあれば拒否される。旧版のバイナリで立てた org を新版で操作する場合(計画の R8)と、org の途中で ralph.toml を書き換えた場合に起きる。Spawn の doc は「1. Idempotent early return: … with no validation attempted at all」と書き、closure 内のコメントは「An idempotent no-op must not be able to fail validation」と書いている。この順序の誤りは PR① と org-runtime-lead の cross-review ACTION_REQUIRED #1 で 2 回直している(`spawn.go:540-553`)。先に走る識別子の検証は spawn 済みの座席の id を拒否できないので、この不変条件を実際に破るのは新しい検査が初めてになる。差し込み箇所のコメント(`spawn.go:342-347`)も、設定キーの分岐に触れていない | `internal/org/spawn.go:279-284`、`:348`、`:496-497`、`:540-553`、`:957-968` | どちらかにする。(a) 設定キーの分岐を、locked closure の idempotent return の直後に、manifest に書かない早期 return として移す(dry-run には idempotent の場合がないので、dry-run 側は今の位置のままでよい)。(b) 今の位置のままにするなら、Spawn の doc の 1 番目と `:496-497` に「旧名の検査だけは例外で、spawn 済みの座席の再実行も拒否する」と書き、`:342-347` に設定キーの分岐を足す |
| MEDIUM | maintainability | M-2: `retiredRoles` の表は汎用だと書かれているが、拒否の文言と規則は qa と lead に固定されている。`prompts.go:118-121` は「adding a name here is the whole change」と書く。一方 `retiredRoleSpawnErr` は、Removed のどの名前にも「its deterministic-gate re-run moved to the %q role」(qa の事情)を出し、Renamed のどの名前にも「`--id` を拒否する。it was the coordinator's agmsg identity」(lead の事情)を当てはめる。たとえば `"impl": {Successor: "implementer", Kind: RetiredRoleRenamed}` を足すと、`--id impl` が事実と違う理由で拒否される | `internal/org/prompts.go:118-125`、`internal/org/spawn.go:925-934`、`:944-956` | 理由の文と `--id` を拒否するかどうかを表の項目に持たせる(例: `Reason string`、`ReservedSeatID bool`)。今は 2 件しかないので、表のコメントを「名前を足すときは `retiredRoleSpawnErr` の文言も直す」に改めるだけでもよい |
| MEDIUM | maintainability | M-3: `GATE:` をヘッダと呼んでいるが、雛形の例では本文に置かれている。`leader.md:23` は「reviewer の BLOCKED は `GATE:` ヘッダで扱いを分ける」と書き、計画の Design decisions も「`GATE:` は typed protocol が許す任意のヘッダ行」としている。ところが `reviewer.md` の RESULT と BLOCKED の例では、`GATE: pass` / `GATE: fail` が `TASK_ID: t-42` の後の空行より下(`:55`、`:67`)にある。agent-messaging.md の規則では、ヘッダは最初の空行までなので、`protocol.Parse` はこれを本文として扱う。今は LLM が読むだけなので動作は変わらない。ただ、tech-debt に足した「FR-7 のゲートを hook で強制する」仕組みがヘッダで判定するようになると、例に従った座席の `GATE:` を見つけられない | `internal/org/prompts/leader.md:23`、`internal/org/prompts/reviewer.md:53-55`、`:65-67`、`.claude/rules/ralph/agent-messaging.md`(Message shape) | 2 つの例で `GATE:` を `TASK_ID:` の直後(空行の上)に移す。本文に置く既存の慣習(`STATUS:` など)に合わせるなら、`leader.md` の「ヘッダ」を「本文の `GATE:` 行」に改める |
| MEDIUM | maintainability | M-4: `watchPendingAlert` のフィールド名と JSON のタグが食い違っていて、理由がコードに書かれていない。`LeaderAgentGet` のタグは `json:"lead_agent_get"`、`HistoryLeaderLines` のタグは `json:"history_lead_lines"` のまま。残した理由(更新の前に書いた state を読むと値が黙ってゼロになり、`-1` の番兵が失われる)は計画の「実装中の逸脱」にしかなく、計画は `/pr` で archive に移る。AC-12 の grep(`Lead(?!e)\|\blead[A-Z_]`)もこの 2 行に毎回掛かる。後から読む人や次の一括置換が改名の漏れと見て、タグを直してしまうおそれがある | `internal/org/watch.go:225-232` | 2 つのフィールドに 1 行のコメントを付ける(例: `// tag kept as "lead_…": persisted watch state written before the leader rename must still decode; renaming it would zero the -1 sentinel`) |
| LOW | exception-handling | L-1: CLI は Spawn の旧名の検査より前に `resolveModelOrWarn` を呼ぶ。`--model` を省くと、`ralph org spawn --role qa`(`--prompt` なし)や `--role lead` で「falling back to first [org].model_pool entry permitted for role qa …」という警告が出てから拒否される。`[org.roles].lead` を別の driver のモデルだけに絞っていて、`--role lead --driver claude` を `--model` なしで実行すると、`DefaultModelForDriverAndRole` の「no [org].model_pool entry … for role "lead"」が先に返り、改名の案内まで届かない。`ralph org start` も、旧キーがあるときは警告を出してから拒否する | `internal/cli/org.go:248-260`、`:334`、`:447`、`internal/org/envelope_summary.go:75-82` | 旧名の検査を export して CLI の `resolveModelOrWarn` より前でも呼ぶか、`resolveModelOrWarn` を Spawn の検査の後に回す |
| LOW | naming | L-2: export する必要のない型と定数が export されている。`RetiredRole`、`RetiredRoleKind`、`RetiredRoleRenamed`、`RetiredRoleRemoved` は `internal/org` の外で使われていない。表の `retiredRoles` は unexport なので、外からは値を作っても渡す先がない。外で使うのは `RetiredRoleConfigKeys` と、その戻り値の `RetiredRoleConfigKey` だけ | `git grep 'RetiredRole\b\|RetiredRoleKind\|RetiredRoleRenamed\|RetiredRoleRemoved'` の非テストの結果は `internal/org/` の中だけ | 4 つを小文字にする |
| LOW | unnecessary-change | L-3: insights のテストの局所変数を改名したため、変数名と中身が合わなくなった。`demoLead` → `demoLeader`、`acmeLead` → `acmeLeader` は、過去の receipts のフィクスチャにある seat `"lead"` を指す。エラー文は「demo/lead」のまま。計画はこのファイルを「過去のデータとして残す箇所の確認だけ」としていた。AC-12 の正規表現に掛かるのを避けた結果と思われる(未確認) | `internal/insights/insights_test.go:386-397`、`:465-476` | `demoSeat` / `acmeSeat` のような、AC-12 に掛からず中身とも矛盾しない名前にする |
| LOW | readability | L-4: 「the literal appears in one place」というコメントが事実と違う。`internal/cli` では `org_test.go` に `oldLeaderName` があるのに、同じ package の `doctor_org_test.go` は `const old = "lead"`、`"[org.permissions.roles].lead"`、TOML の `lead = \"guarded\"`、`"[org.roles].leader"` を直接書いている。`internal/org` でも `TestRenderRolePrompt_QA_NoTemplate` が `"qa"` を 3 回直接書いている | `internal/cli/org_test.go:3003-3005`、`internal/cli/doctor_org_test.go:1134`、`:1158`、`:1164`、`:1235`、`internal/org/prompts_test.go:200-202`、`:454-461` | 定数(と `org.LeaderIdentity`)を使うか、コメントの主張を外す |
| LOW | readability | L-5: 一部だけ折り返し直した跡が残っている。`reviewer.md:105` は 161 字の 1 行で、ほかの行は 40〜50 字で折り返している。`/org` skill の `:158`(53 字)と `:171`(116 字)も同じで、4 面すべてに写っている。`templates/base/ralph.toml:82` は 127 字で、このファイルで 80 字を超える行はここだけ | `internal/org/prompts/reviewer.md:105`、`.claude/skills/org/SKILL.md:158`、`:171`、`templates/base/ralph.toml:82` | 周りの幅に合わせて折り返す(skill は `.claude` 側を直してから `sync-skills.sh` と template への写し) |
| LOW | readability | L-6: reviewer 雛形の書き込み範囲の規則が 2 つ並び、どちらが優先か書かれていない。新しく書いた「書き込むのは `docs/reports/` 配下のレポートと、ゲートのスクリプトが自分で作る生成物 … だけです」の直後に、前からある「scope に記載された対象外のファイルは変更しないでください」が続く。スコープ規律の節も「scope の範囲外は読むだけに留め、書き込みは行わないでください」と書く。scope が `internal/org/**` のように狭いと、レポートもゲートの生成物も scope の外になる。字義どおりに読む座席は、ゲートを `GATE: unrunnable` にするか、レポートを書かずに終わるかもしれない | `internal/org/prompts/reviewer.md:8-13`、`:75` | 「レポートとゲートの生成物は scope にかかわらず書いてよい」と 1 文で書く |
| LOW | readability | L-7: 「guarded にすると許可待ちで止まる」を driver を問わず断定している。claude の guarded は CLI の対話の既定なので、計画のとおり止まる。codex の guarded はフラグを付けず、codex 自身の設定を受け継ぐ(`permissions.go:84-89`)。scaffold する `.codex/config.toml` は `sandbox_mode = "danger-full-access"`、`approval_policy = "on-request"` なので、codex の guarded の reviewer はおそらく止まらない(未確認)。計画の Open questions もこの点を未確認として挙げている | `templates/base/ralph.toml:82`、`.claude/skills/org/SKILL.md:186`、`templates/base/.codex/config.toml:26`、`:35`、計画の Open questions | 「claude の guarded では許可待ちで止まる。codex の挙動は codex の設定による」と書き分ける |

CRITICAL と HIGH はない。

## Positive notes

- 撤去・改名した役割を `retiredRoles` の 1 か所にまとめ、spawn の拒否、ralph.toml のキーの検査、doctor の warn がどれもそこを読んでいる。照合は大文字小文字を区別するという既存の規則に合わせ、テストでも押さえている
- 旧名の拒否は manifest と receipts に何も書かない。実行と dry-run の両方で、ドライバの呼び出し、manifest のイベント、receipts、`ModelReceipt` がどれも空であることをテストで確かめている
- `resolveLeaderDriver` は既定値との比較ではなく `Flags().Changed` で判定している。既定値と同じ `--leader-driver claude` と `--lead-driver codex` を同時に渡した場合も、矛盾として拒否する
- 永続化された JSON のタグを残すという判断は計画に記録してあり、AC-12 と AC-14 の例外として数えている(M-4 はその理由をコードにも書く提案)
- `markdownItem` を足したので、`GATE: fail` と `GATE: unrunnable` の文言を、隣の項目に頼らずそれぞれの項目の中で検査できる。この helper 自体にもテストがある
- 改名で、関係のない語や過去の計画・レポートのファイル名の引用は書き換わっていない(`spawn.go:59`、`:543`、`:778`、`org.go:401-402`、`report.go:11`)

## Tech debt identified

| Debt item | Impact | Why deferred | Trigger to pay down | Related plan/report |
| --- | --- | --- | --- | --- |

この review で新しく先送りにしたものはない。M-1〜M-4 とどの LOW も、この PR の中で直せる。直さずに PR に進む所見があれば、その所見を `docs/tech-debt/README.md` に 1 行にまとめて足すこと(cycle の上限に達した場合も同じ)。

## Recommendation

- Merge: 条件付きで可。CRITICAL と HIGH はない。M-1(Spawn の順序の不変条件)と M-3(`GATE:` の置き場所)は、後から機構を足すときにそのまま引き継がれるので、PR の前に直すことを勧める。M-2 と M-4 はコメントの修正で済む
- Follow-ups: L-1〜L-7。M-1 を (a) で直す場合は、spawn 済みの座席の再実行が旧キーのある設定でも `SpawnOutcomeIdempotent` を返すことを確かめるテストを足すとよい(テストの判断は /test の担当)
