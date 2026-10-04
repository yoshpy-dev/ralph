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


## Addendum (follow-up diff 28cc057c..23f824e6)

- Date: 2026-10-04
- Reviewer: reviewer subagent (Claude)、cycle 1 の指摘に対する修正の確認
- 対象: `git diff 28cc057c..23f824e6`(3f30641d、23f824e6 の 2 コミット、16 ファイル)。28cc057c(origin/main の取り込み)は対象外。親子関係は `git log --format='%h parent=%p'` で 23f824e6 → 3f30641d → 28cc057c と一直線であることを確認した
- ID の付け方: 上の表の M-1〜M-4、L-1〜L-7 は cycle 1 の ID のまま。この節の新しい指摘は C2-1 から振る(計画の「実装中の逸脱」が M-/L- を裸で引いているため、同じ接頭辞で重ねない)

### Evidence reviewed

- 読んだ diff: `spawn.go`(doc の番号付き一覧、Spawn 本体の先頭、dry-run 分岐、locked closure、`RetiredRoleInputErr` / `retiredRoleConfigErr`)、`cli/org.go`、`prompts.go`、`watch.go`、`reviewer.md`、`ralph.toml`、`/org` skill 4 面、新しいテスト(`spawn_test.go` の 2 本、`org_test.go` の 1 本)、test の定数化(`doctor_org_test.go`、`prompts_test.go`、`insights_test.go`)、計画の追記
- Spawn の順序は diff ではなくファイル全体(`spawn.go:276-695`)を読んで、manifest・receipt・driver への書き込みが旧キーの検査より前にないことを確かめた。`withManifestLock` の中身(`lockfile.go:50-70`)も読んだ
- 掃引: 旧 identifier(`retiredRoleSpawnErr`、`RetiredRoleKind`、`RetiredRoleRenamed`、`RetiredRoleRemoved`、`RetiredRole`)の `git grep` は docs/reports と archive を除いて 0 件。Spawn の doc の段番号を指す参照(`doc comment, item`、`step N`)を `internal/` で探した。`GATE:` と「許可待ち」「permission prompt」を docs 含めて探した。`/org` skill の 4 面は `cmp` で一致
- 実行していないもの: テスト、gofmt、vet、AC の grep。テストが落ちるかどうかは、コードを読んで mutation を頭の中で当てはめた範囲の判断で、実行した結果ではない

### Cycle 1 の指摘の状態

| ID | 状態 | 確認した内容 |
| --- | --- | --- |
| M-1 | 解消 | 旧キーの検査は `spawn.go:534`。idempotent return(`:515-525`)の直後で、stale 検出(`:607`)、envelope、manifest への書き込みより前。dry-run 側は `:403` で最初。doc の 1 番目と 2 番目、`:492-493`、`permArgs` の注記(`:464-473`)も新しい位置に合わせて直っている。新しいテスト 2 本(`spawn_test.go:3159`、`:3219`)は、旧位置(manifest 読み込みの前)に戻すと前者が落ちる。後者は stale 検出の return の後ろへ動かすと落ちる |
| M-2 | 解消(コメント案) | `prompts.go:119-127` が「理由の文と seat-id 規則は 2 件分を書いたもので、表から導いていない。名前を足すときは見直す」と書き、「whole change」の断定を外した |
| M-3 | 解消 | `reviewer.md:55-57`、`:66-69` で `GATE:` が `TASK_ID:` の直後、空行の上にある。`protocol.Parse`(`protocol.go:111-126`)は最初の空行までを `Fields` に入れるので、`Fields["GATE"]` として読める。`:50` に置き場所の 1 文もある。`leader.md:23` の「ヘッダ」と一致した |
| M-4 | 解消 | `watch.go:230-231` に、タグを残す理由(古い state を読むと値がゼロになる、`-1` の番兵を失う)が書かれた。`-1` の意味は `:960` と `:989-990` の読み側と合う |
| L-1 | 一部 | `ralph org spawn` の `--role` / `--id` は `cli/org.go:332` で `resolveModelOrWarn`(`:345`)より前に拒否される。`ralph org start` と、旧キーがある状態の `spawn`(旧名でない role)は、警告のあとで拒否される順序のまま。C2-4 を参照 |
| L-2 | 解消 | 型と定数 4 つが小文字になった。外からの参照は元々なく、掃引も 0 件 |
| L-3 | 解消 | `demoSeat` / `acmeSeat` に改め、エラー文の「demo/lead」と合う |
| L-4 | 解消 | `doctor_org_test.go` の 5 か所と `prompts_test.go` の `"qa"` 3 か所が定数に変わった。コメントは「literal を繰り返さない」に直った。残る `"lead"` の直書きは過去 receipts のフィクスチャ(`internal/insights/insights_test.go`、`internal/cli/insights_test.go`)で、`org_test.go:3003-3007` のコメントがその理由を述べている。`status_test.go` の `"qa"` は計画の Non-goals |
| L-5 | 解消 | `reviewer.md` に 80 字を超える行はない。`ralph.toml:82-85` も同じ。`SKILL.md` の追加行は最大 48 字で、4 面は `cmp` で一致 |
| L-6 | 解消(C2-3 を新規に起票) | 例外は `reviewer.md:12-14` と `:78-79` の 2 か所に入った |
| L-7 | 解消 | `ralph.toml:82-85` と `SKILL.md:189-190`(4 面)が claude と codex を書き分けた。同じ主張を他の文書が持っていないか「許可待ち」「permission prompt」で探したが、見つからなかった |

### Findings (新規)

| Severity | Area | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- |
| LOW | maintainability | C2-1: M-1 の修正で Spawn の doc の段番号を 1 つずつずらしたが、段番号で Spawn の doc を指す参照が 1 か所残った。`verbs_test.go` のコメントは「Spawn's own doc comment: step 2 runs before step 4」と書く。これは envelope 検証(ValidateSpawnEnvelope)が stale 補償より先に走る、という意味で、28cc057c の番号付けでは正しかった。今は 2 番が旧キーの検査、4 番が AC-2b のゲートで、envelope は 3 番、stale 補償は 5 番になっている。テストの論理は変わらないので動作には影響しない | `internal/org/verbs_test.go:1684-1685`、`internal/org/spawn.go:290`(2 番)、`:294`(3 番)、`:300`(4 番)、`:304`(5 番) | 「step 3 runs before step 5」に直す。次の挿入でまたずれるので、番号ではなく「ValidateSpawnEnvelope が compensateStale より前」と関数名で書くほうがよい |
| LOW | readability | C2-2: 旧キーの検査を足したのに、閉じた一覧になっているコメントが書き換わっていない。(a) `spawn.go:431-433` は「idempotent/stale-in-flight-detection/envelope/permission の検査、capacity の検査、spawn_started の追記が withManifestLock の中で走る」と数え上げ、旧キーの検査が抜けている。同じ closure の見出し(`:492-493`)は直っている。(b) dry-run のコメントは「Dry-run mirrors the real path's ordering exactly: ValidateSpawnEnvelope, then …」と 4 つの検査を同じ順序として並べ(`:387-394`)、新しい先頭の検査は 12 行下に足した段落(`:399-402`)にだけ書かれている。(c) `spawn_test.go:2789-2794` は「guard sits ahead of the manifest and the receipts」と書くが、real モードの ralph.toml キーの 2 ケースでは、検査は manifest の読み込みと lock の取得のあとに走る(event と receipt より前という意味では正しい)。(d) `SpawnResult.ModelReceipt` の doc は receipt を書かない経路を「pre-manifest identifier-validation rejection」と数え上げる(`spawn.go:264`)。この一覧は PR より前からあり(26b03ce1)、cycle 1 で入れた旧名の拒否 2 種を含まない。cycle 1 で私が見落とした分で、旧キーの拒否が closure の中から返るようになったので目立つ | `internal/org/spawn.go:431-433`、`:387-394`、`:264`、`internal/org/spawn_test.go:2789-2794` | その場で書き直す。(a) に「retired-key」を足す。(b) は一覧の先頭に retiredRoleConfigErr を入れる。(c) は「ahead of any manifest event or receipt」とする。(d) は「a plain rejection that never reaches reject()(identifier validation、retired role name / key)」とする |
| LOW | readability | C2-3: L-6 の修正文が、同じ段落の中で文体と内容の両方で浮いている。`reviewer.md:12-14` は「書き込むのは `docs/reports/` 配下のレポートと、ゲートの生成物だけです。レポートとゲートの生成物は scope にかかわらず書いてよい。それ以外は scope に従う。scope に記載された対象外のファイルは変更しないでください」と続く。(1) です・ます、常体、ください、の順に文体が変わる。(2) 「それ以外は scope に従う」は、scope が許せばレポートとゲートの生成物以外にも書けると読める。直前の「コードと設定は変更しません」「だけです」(`:9-12`)と食い違う。(3) 「書いてよい」は直前の文の言い直しで、「scope に記載された対象外…」は `:78-79` の規律の節と重なる。例外が 3 か所に書かれている | `internal/org/prompts/reviewer.md:9-14`、`:78-79` | `:12-14` を「書き込むのは ... だけです(これらは scope の外でも書いてかまいません)。」の 1 文にして、scope の規則は `:78-79` に任せる |
| LOW | exception-handling | C2-4: L-1 の残り半分は計画の「実装中の逸脱」にだけあり、tech-debt の台帳にない。旧キー(`[org.roles].lead` など)が ralph.toml に残っていると、`ralph org start` は `resolveModelOrWarn`(`cli/org.go:458`)が「--model omitted; falling back …」を出してから Spawn が旧キーで拒否する。旧名でない role の `ralph org spawn`(`:345`)も同じ順序になる。計画の記述は `org start` だけを挙げていて、`spawn` が抜けている。旧キーの検査を CLI へ出せないのは M-1 の設計の帰結で、順序を変える選択肢は実質ない。実害は警告 1 行で、LOW のまま。ただ、計画は `/pr` で archive へ移り、台帳の該当行は `docs/tech-debt/README.md:152`(FR-7)の 1 行だけ。cycle の上限に達しているなら、これは「直さずに進む」ことになる | `internal/cli/org.go:345`、`:458`、`internal/org/spawn.go:534`、`docs/plans/active/2026-10-04-org-drop-qa-seat.md`(「2026-10-04(self-review の後)」の段落)、`docs/tech-debt/README.md`(該当行なし) | 上限に達しているなら、台帳に 1 行足す(例: 影響は警告 1 行、trigger は `resolveModelOrWarn` か旧キーの検査を次に触るとき、関連は計画と本レポートの C2-4)。まだ cycle が残るなら、計画の記述に `spawn` も含めるだけでよい |

CRITICAL、HIGH、MEDIUM はない。新規は LOW 4 件。

### 確認して起票しなかったもの

- Spawn の順序の不変条件(依頼の確認事項 2): 実 path の入力検査(識別子、agent-name 長、`RetiredRoleInputErr`)は manifest の読み込み前、旧キーの検査は idempotent return の直後で、`Receipts.Append`、`appendEvent`、driver 呼び出し、`compensateStale` はどれもその後ろにある。旧位置と新位置で変わるのは、real path で `withManifestLock` が state dir の作成と `manifest.lock` の作成を先に行うことだけ。これは closure に入っている idempotent や envelope の検査と同じで、manifest の event ではない。state dir が書き込めない環境では、旧キーの案内より先に「manifest lock」のエラーが返る(想定が薄いので起票しない)
- dry-run の順序: 新しい座席では dry-run も real も 旧キー → envelope → permission mode → AC-2b → capacity で、最初の原因が一致する。dry-run には idempotent の分岐がないので、spawn 済みの座席に対する dry-run は旧キーで拒否され、real は idempotent で成功する。envelope や capacity で以前からある差と同じ種類で、doc の「first cause」の主張は複数の検査に落ちる要求についてのものなので、doc は偽にならない。既存のテスト `TestOrgSpawn_RetiredLeaderName_RejectedBeforeAnyManifestWrite`(`spawn_test.go:2795`)が、キーの 2 ケースを dry-run と real の両方で押さえている
- step 0 の `RetiredRoleInputErr` は、旧版で spawn 済みの座席 id `lead` の再実行も拒否する(識別子の形の検査と違い、spawn 済みの id を拒否できる)。doc の 1 番目は「config-dependent validation」と書き分けていて正確で、旧名を使う手順を新版が拒否するのは計画の意図どおり
- CLI の事前拒否(`cli/org.go:332-335`): 本来の拒否と同じ `printSpawnResult` を通り、`return err` も Spawn 後の `return result.Err` と同じ形で、stdout の `rejected:` 行と stderr の cobra のエラー行が両方出る。テスト `TestOrgSpawn_RetiredRoleOrID_RefusedBeforeModelFallback` の 4 ケースは、CLI の事前拒否を外すと case 1 と 4(警告または model_pool のエラーが先に出る)が落ち、`printSpawnResult` を外すと 4 ケースとも落ちる(実行ではなく読んだ範囲の判断)
- orchestrator が計画の trivial-edit の例外で直した 2 点(`printSpawnResult` の追加、テストの `"qa"` 直書きへの戻し)の周囲: 直上のコメント(`org.go:325-331`)は「printed the same way」に更新済み。同じ RunE の他の早期 return(必須フラグの欠落など)とは出力の形が違うが、Spawn 後の拒否とは同じ。計画の AC-5 の許可リストにも `org_test.go` が足されている
- `watch.go:230-231` の行末コメントは 148 字と 154 字で、1091 行のファイルで 120 字を超える 9 行のうちの 2 行。struct の doc(`:221-224`)が `HistoryLeaderLines` をすでに論じているので、そこへ移す手もあるが、見た目だけの話なので起票しない
- 台帳(`docs/tech-debt/README.md`)はこの range では触っていない。cycle 1 で足した FR-7 の行(`:152`)の記述は今回の修正で偽になっていない
- insight event: `docs/insights/events/2026-10-04-org-drop-qa-seat.jsonl` に `self_review` / cycle 1 がすでにあるので、この addendum では追記しない(phase と cycle が集計の key で、二重に数えるため)。cycle 2 として数えるかどうかは orchestrator の判断

### Recommendation (addendum)

- Merge: 可。CRITICAL、HIGH、MEDIUM はない。cycle 1 の M-1〜M-4 と L-2〜L-7 は解消し、L-1 は半分残った。C2-1(番号の参照)と C2-2(コメントの一覧)は PR の中で直せる。C2-4 は上限に達しているなら台帳に 1 行足してから `/pr` へ進む

### Resolution (orchestrator)

- C2-1〜C2-4 は 14922f64 で直した(コメントとテンプレートの文言と tech-debt の 1 行だけで、挙動は変えていない)。C2-4 は `docs/tech-debt/README.md` の末尾の行に記録した。L-1 の残り(`ralph org start` などで、ralph.toml の旧キーがあると `--model` の fallback 警告が先に出る)は、この tech-debt の行で扱う
