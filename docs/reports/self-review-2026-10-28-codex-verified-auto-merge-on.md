# Self-review report: codex-verified-auto-merge-on

- Date: 2026-10-29
- Plan: `docs/plans/active/2026-10-28-codex-verified-auto-merge-on.md`(issue #274、D1〜D5、AC1〜AC7、Progress の S1〜S4)
- Reviewer: reviewer サブエージェント(`/self-review`、パイプラインの cycle 1)
- Scope: `git diff ae64da70...HEAD`(9 コミット、32 ファイル、+701/-263)。中心は S1 29fd179e(`internal/config/defaults_sync_test.go` の表のテスト)、S2 cb8d258e と ce6e972d(既定を true に、`internal/config` と `internal/cli` のテスト)、S3 920a4dcb(Go のコメント)、S4 f99ac82d(skill・recipe・README・spec・tech-debt)。diff quality だけを見た。実行したのは `go test ./internal/config/ -count=1`(通過、0.6 s)の 1 回だけで、変異と静的検査は回していない。
- ID: M1、L1〜L4

## Evidence reviewed

- `internal/config/defaults_sync_test.go` の追加部分の全体。`TestOrgTemplateDrift_EachKeyIsComparedOnBothSides`(`:402-495`、94 行のうち表が 35 行)、`tomlKeys`、`editTemplateKey`。
  - 表の行は 19 で、`orgTemplateProbe`(`:114-142`)の toml のタグの数と合う。`tomlKeys` は struct を表として下り、`*[]string` や `*[]OrgModelPoolEntry` はポインタなので葉になる。probe の表はすべて非ポインタの struct なので取りこぼさない。
  - `editTemplateKey` は、見出しの正規表現が `# [org.permissions.roles]` のようなコメントに当たらず(行頭 `^\[`)、`[[...]]` はテンプレートにない。`keyLine` は `^key\s*=` なので `wake` が `wake_cmd` に当たらない。`found != 1` が改名と二重定義を Fatal にする。`model_pool` の複数行は括弧の深さで消すが、テンプレートの該当 10 行(`templates/base/ralph.toml:33-42`)の中に `[` `]` を含む文字列やコメントはない。
  - bool の行はテンプレート側の値を `!Default()` から作るので、S2 の前後どちらでも vacuous にならない。(i) `Default()` のコピーの 1 field、(ii) テンプレートの値、(iii) テンプレートのキーの削除、の 3 つがどれも「そのキーを名指す 1 行」を求めるので、同じ既定値のキーの取り違えが両側で落ちる。
- `internal/config/config_test.go` の追加と書き換え。`TestLoad_CodexVerifiedAndAutoMergeDefaultToTrue`、`TestLoad_TemplateWithBothKeysFalse`(`reflect.DeepEqual` は同じテンプレートを `Load` した 2 つなので、2 field だけの差を見る)、`TestLoad_OrgPermissionsCodexVerifiedRoundTrip` の false/true 両方、`TestLoad_OrgDirectorRoundTrip` の false への入れ替え。default-flip の定番(新しい既定を明示して読み返すだけの vacuous なテスト)は、false の半分が残っているため、どの組も実質の確認を失っていない。
- `internal/cli` のテスト 9 ファイル。D2 の規則(false の動きのテストは false を明示して残し、既定そのものを見るテストだけ true にする)は守られている。false を明示して残したもの: `writeCodexOrgConfig` の `codex_verified = false`、`TestOrgDirector_StatusShowsAutoMerge` と `TestStatusCmd_DirectorLineShowsAutoMerge` の false の組、`TestOrgDirector_AutoMergeFromTheMainWorktreeRalphToml` の「worktree が false → 既定の true」への組み替え、`directorNotAttachedOffLine` / `directorDisabledOffLine` の新設。
- `internal/config/config.go`、`internal/org/permissions.go`、`internal/cli/doctor_shell_alias.go`、`internal/cli/org.go` のコメント。`templates/base/ralph.toml` の 2 つのキーの前のコメント。
- skill 2 つ(`.claude/skills/org/SKILL.md`、`.claude/skills/director/SKILL.md`)、recipe `codex-seat-permissions.md` と `codex-setup.md`、`README.md:124`、spec の FR-8・受け入れ条件・Security の印、`docs/tech-debt/README.md:144` の語単位の diff。写し(`.agents/`、`templates/base/`)はルート側と `diff` が一致する(recipe と org skill を確認)。
- 残った false 既定の記述の探索。`git grep -E 'codex_verified|auto_merge'` を Go 以外のファイルに掛け、`fail-closed|opt.in|unlock|既定|default` と組み合わせた。Go は `git grep -nE "codex_verified|CodexVerified|AutoMerge|auto_merge" -- 'internal/*.go' ':!*_test.go'` の全行を読んだ。
- `internal/cli/status.go:258-296` の読み込み失敗時の fallback と、`/director` skill の「merge と auto_merge」節(`:813-830`)。
- 見ていないもの: `docs/tech-debt/README.md:136` の「An operator who has already set `codex_verified = true`」(進捗欄のとおり /verify で見る)、仕様との適合、テストの網羅。

## Findings

| ID | Severity | Area | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- | --- |
| M1 | MEDIUM | exception-handling / security | `ralph status` は `ralph.toml` が読めないとき、読み取りだけなので失敗させず `config.Default()` の値で director の行を出す。`auto_merge` の既定が false から true に変わったため、この fallback が安全側から開いた側に反転した。人が `auto_merge = false` と書いた file が壊れていても、`ralph status` の `director:` の行は「人に聞かずに merge する(`auto_merge = true`)」と出し、知らせは stderr の warning 1 行だけになる。`/director` skill は merge の前にこの行を読んで決める。 | `internal/cli/status.go:261-265`(`fallback := config.Default().Org` と `fallback.Director.AutoMerge`)、`:290-295`(main worktree の `ralph.toml` が読めないときも `config.Default().Org.Director`)。テストが新しい値を固定している: `status_test.go:1164`、`:1186`(`auto_merge (true)`)。`/director` skill `:813-821` は値を `ralph status` の行か `ralph org director status` の `auto_merge:` の行から読む。2 つの食い違いの規則は `--config` を使う headless の場合だけで、warning の扱いは書いていない。一方 `ralph org director status` は読めない `ralph.toml` をエラーにする(`org_director.go:394-401`)ので、同じ状況で 1 つはエラー、もう 1 つは true と答える。 | 2 つの fallback で `AutoMerge` だけ false に固定し(`enabled` は既定の true のまま。warning の文も `auto_merge (false)` にする)、`status_test.go:1164`、`:1186` の期待を合わせる。コードを触らないなら、`/director` skill に「`ralph status` が `warning: status: load ralph.toml` を出したら `auto_merge` は false として扱い、人に伝える」の 1 文と PR の Notes を足す。skill は LLM が stderr を読むことに頼るので、前者を勧める。どちらにするかは orchestrator が決める。 |
| L1 | LOW | maintainability(コメントの主張が根拠を超える) | `permissions.go` の `permissionArgsForDriver` の doc comment が「既定は true。その検証があるから」と書く。検証は 2026-09-18 に 1 台、codex-cli 0.154.0 で行ったもので、置き換えた前の文は「検証が足りないから false なのではなく、flag が利用者の `~/.codex/config.toml` を引き継ぐから false」と、検証は理由でないと明言していた。既定を true にした理由は、計画の Assumptions のとおり 2026-10-27 の人の決定(#274)で、引き継ぎの性質は変わっていない。 | `internal/org/permissions.go:119-121`「The default is true because of that verification (issue #274).」。計画 `Assumptions` の「既定の値は 2026-10-27 にユーザーが決めた(issue #274)」。`config.go:119-123` は「and the default is true (issue #274)」と並べるだけで、この主張はない。 | 「The default is true by the maintainer's decision (issue #274); the verification above is the evidence for the mapping on one machine.」のように、決定と証拠を分ける。 |
| L2 | LOW | maintainability(テストの前提が暗黙) | `internal/org` のテストは false の動きを `config.OrgConfig{}` と `testOrgConfig()` の zero value で確かめている。動作は変わらず通るが、コメントとテスト名が「既定」の動きとして書いてあり、この PR のあとは偽になる。D2 の「false の動きは false を明示する」にも当たらない。`testOrgConfig()` は `MaxOrgs` だけ `config.Default()` から取っているので、同じ形で `Permissions` も `Default()` から取る変更が入ると、これらのテストが原因の分かりにくい形で落ちる。 | `internal/org/envelope_test.go:10-25`(`Permissions` を設定しない)。`spawn_test.go:3643-3648`「a codex seat under the default (autonomous) config is rejected」、`:3720`「fail-closed under the default autonomous mode」、`:3912-3916`「under codex's fail-closed autonomous default」。テスト名 `TestOrgSpawn_Codex_AutonomousDefault_RejectedFailClosed_WithReceipt`。`permissions_test.go:92` は名前が「when codex_verified is false」なので誤りではないが、`config.OrgConfig{}` に頼る。 | `testOrgConfig()` に `Permissions: config.OrgPermissionsConfig{CodexVerified: false}` と「zero value であり `Default()` の true ではない。codex の fail-closed のテストが頼る」の 1 行を足す。3 つのコメントの「the default」を「codex_verified false」に直す。テスト名の変更は任意。 |
| L3 | LOW | readability(利用者に見える文) | doctor の「Shell aliases」の Detail が、codex の flag の説明に「(modes a codex seat gets only once `[org.permissions].codex_verified = true`)」と書く。既定が true になったので、既定の project でも読み手には「まだ true にしていない」と読める。文は条件として偽ではない。S3 は同じ関数の上の doc comment(`:571-579`)を直したが、この 2 つの出力の文は残っている。 | `internal/cli/doctor_shell_alias.go:714-715`、`:718-719`。テストはこの括弧の文言を固定していない(`git grep "gets only once"` は実装の 2 行のみ)。 | 「modes a codex seat gets unless `codex_verified = false`」のように、既定の true を前提にした書き方に直す。S3 の範囲を超えるなら、PR の Notes に残して次に doctor を触るときに直す。 |
| L4 | LOW | maintainability(コメント) | 新しいテストが数えているのは reflect の key の集合なのに、キーの数「19」がコメントに 3 回残っている。キーを足すと表の行は強制されるが、この数は黙って古くなる(計画が閉じたかった「数がコメントにしかない」の残り)。また新しいテストの doc comment は、既定値が同じキーの組を 3 つ挙げて全部のように読めるが、`ack_minutes`(10)は `max_orgs` と `deadman_minutes` の組に入り、`interval_seconds` と `max_total_seats` は 30 で同じ。 | `internal/config/defaults_sync_test.go:10`、`:109`、`:147`(「19 [org] keys」)。`:397-399`(「Two keys with the same default (max_orgs and deadman_minutes, ...)」)。 | 「19」は消して「every [org] key of orgTemplateProbe」と書く(進捗欄の「19 は残した」を、この PR で閉じるか台帳に送る)。同じ既定値の組は「for example」を付けるか、組を数えずに「keys that share a default」とだけ書く。 |

## 依頼された観点への答え

- 表のテストの論理(`TestOrgTemplateDrift_EachKeyIsComparedOnBothSides`、`tomlKeys`、`editTemplateKey`): 欠陥は見つからなかった。上の Evidence に確かめた境界を書いた。L4 はコメントだけ。
- D2: 守られている。false の動きのテストは消えずに、false を明示する形で残っている。既定そのものを見る期待だけが true になった。`internal/org` のテストは別の問題(L2)。
- 既定 true の記述の正確さ: skill、recipe、README、template の comment、spec の印、`tech-debt:144` に false 既定の言い方は残っていない。残っていたのは L1(理由の書き方)と L3(利用者に見える文)で、どちらも偽の既定を書いてはいない。
- 既知として再提起しないもの: `permissions.go` の実行時エラー文(`codex_verified=true` を求める文)、spec の旧文の履歴、D4 の「違えば false」の規則。M1 は D4 の規則が覆わない別の経路(読み込み失敗の fallback)。

## Positive notes

- 表のテストは、比較の「どちらの側の取り違え」かを別々の変異で落とす。bool の行を `!Default()` から作る点は、この PR が既定を 2 つ反転しても通るので S1 を先に足す順が活きた。
- `editTemplateKey` が「表の範囲内で 1 回だけ」でなければ Fatal にするので、テンプレートの書き換えで行が vacuous にならない。
- `TestLoad_TemplateWithBothKeysFalse` は、戻し方の手順(2 行を手で false にする)をそのまま再現し、2 field 以外の値が変わらないことまで見る。
- テンプレートの comment と `config.go` の field comment が、既定、false にする条件、キーがないときの値を同じ順で書いていて、読み比べやすい。

## Tech debt identified

| Debt item | Impact | Why deferred | Trigger to pay down | Related plan/report |
| --- | --- | --- | --- | --- |
| なし(この cycle では台帳に足さない) | — | M1、L1〜L4 はいずれもこの PR の中で直せる。直さずに受け入れるものがあれば、PR の Notes か、最後の cycle で 1 行にまとめて足す。 | — | — |

## Recommendation

- Merge: yes(CRITICAL 0、HIGH 0、MEDIUM 1、LOW 4)
- Follow-ups: (1) M1 は直すか、skill と PR の Notes に書いて受け入れるかを /verify の前に決める。直すなら 2 つの fallback と `status_test.go` の 2 か所の期待だけで済み、`ralph org merge` の動きは変わらない。(2) L2 は `testOrgConfig()` の 1 行と 3 つのコメント。(3) L1、L3、L4 は文言だけ。直したら、コード・コメントの変更なので /self-review から回し直す。直しが記録だけに収まる場合は `post-implementation-pipeline.md` の例外を使える。
