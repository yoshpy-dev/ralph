# codex-verified-auto-merge-on

- Status: Approved
- Approved: 2026-10-28 sha256:85ba56ccca6a
- Owner: Claude Code
- Date: 2026-10-28
- Related request: codex_verified と auto_merge の既定を true にする
- Related issue: #274
- Type: feat
- Branch: feat/codex-verified-auto-merge-on

## Objective

`[org.permissions].codex_verified` と `[org.director].auto_merge` の既定を false から true にする。`ralph.toml` にキーを書かないときの値(`internal/config/config.go` の `Default()`)と、`ralph init` が書く `templates/base/ralph.toml` の値をそろえて変え、「既定 false」と書いている文書を直す。あわせて、#273(PR #281)の Known gaps (a)(b) を閉じる照合のテストを足す。

## Scope

- `internal/config/config.go`: `Default()` の `CodexVerified` と `AutoMerge` を true にし、2 つの field のコメントと `OrgConfig` の説明の「既定 false」を直す。
- `templates/base/ralph.toml`: `codex_verified = true` と `auto_merge = true` にし、2 つのキーの前のコメント(78〜95 行、150〜161 行付近)を新しい既定に合わせる。
- `internal/config/defaults_sync_test.go`: 照合する全キーの表のテストを足す(D3)。キーごとに、`Default()` 側の値の変更、テンプレート側の値の変更、テンプレート側のキーの削除の 3 つで、`orgTemplateDrift` がそのキーを名指しする 1 行だけを返すことを確かめる。表のキーの集合が `orgTemplateProbe` の toml のキーの集合と一致することも reflect で確かめる(照合のキーの数がコメントにしかない、の解消)。既存の変異のテストの名前にある「whose default is false」を直す。
- `internal/config` のテスト: 新しいテンプレートの 2 行を `false` に書き換えた `ralph.toml` が、2 つとも false で読めることを確かめる(戻し方の手順の裏づけ、AC7)。
- 古い既定に頼っていたテスト: D2 の方針で直す。`Load` は `Default()` から始めて上書きする(`internal/config/config.go:638-649`)ので、影響はキーを書かない `ralph.toml` か `ralph.toml` のない場合を読むテストに出る。`internal/org` のテストは手組みの `testOrgConfig()`(`internal/org/envelope_test.go:10-25`)を使い、`Default()` の 2 つの値を読まないので、影響は `internal/config` と `internal/cli` に集まる(consult の確認、実装で確かめる)。
- Go のコメントで「default false」と書いている箇所: `internal/org/permissions.go`、`internal/cli/doctor_shell_alias.go`(574 行付近)、`internal/cli/org.go`(`withMainWorktreeDirectorEnabled` の「(true and false)」)など、実装のときに `git grep` で洗い出したもの。
- 文書:
  - `/org` skill と `/director` skill(`.claude/skills/` と `templates/base/.claude/skills/` を直し、`./scripts/sync-skills.sh` で `.agents/skills/` の 2 つの写しを作り直す): `/org` の 132 行付近「既定の false では」と 1541〜1546 行付近「既定では guarded 以外を…拒否」「検証してから有効にする」、`/director` の「merge と auto_merge」節の「`auto_merge = false`(既定)」。
  - `docs/tech-debt/README.md:144` の (b)「falls back to `codex_verified = false` … pass」は、既定が true になると偽になるので、その文だけを直す(表の行は増やさない)。
  - `docs/recipes/codex-seat-permissions.md` とその写し `templates/base/docs/recipes/codex-seat-permissions.md`: 冒頭の「defaults to `false`」と、確かめの手順のあとに「true にする」と書いた流れを、「既定は true。確かめて通らなければ false にして codex の座席を guarded に戻す」に組み替える。
  - `docs/recipes/codex-setup.md` とその写し(204 行付近)、`README.md` の 124 行付近「(default false)」。
  - `docs/specs/2026-10-07-org-multi-org-director.md`: FR-8「既定では人が merge する」、受け入れ条件の「`auto_merge = false`(既定)」、308 行付近「`auto_merge` は既定で無効にする」に、#274 で既定を true にした印を付ける(本文の履歴は消さない)。
- PR の本文の release のノート: 既存の project の `ralph.toml` には false が書いてあるので変わらない。変えたい人は手で true にする。キーを書いていない `ralph.toml` と `ralph.toml` のない project では、新しい既定(true)になる。テンプレートの既定(`[org].driver_pool` に codex、`[org.permissions].default = "autonomous"`)のままの project では、agmsg の DB が codex の writable root にないと `ralph doctor` の「Codex sandbox (agmsg writable root)」が warn を出すようになる。

## Non-goals

- `codex_verified` の名前を変えること(issue の注意のとおり)。
- 既存の project の `ralph.toml` を書き換えること。`ralph.toml` は seed で、`ralph upgrade` は advisory を出すだけ(`internal/upgrade/replaceplan.go`)。
- codex の headless の director の扱い(`codex_verified` によらず guarded のまま)。
- `ralph doctor` の検査の追加や変更。「Codex sandbox (agmsg writable root)」の検査は今のまま残す。
- この repo のルートに `ralph.toml` を作ること(#264・#265 の担当)。
- `ralph org merge` の動き。`auto_merge` は `/director` skill が読む印で、verb は報告するだけ(今のまま)。

## Assumptions

- 既定の値は 2026-10-27 にユーザーが決めた(issue #274)。蒸し返さない。
- `ralph org merge` は `auto_merge` によらず PR、checks、既定のブランチの保護を確かめてから merge する(issue の背景、`internal/org/merge.go`)。
- この repo にはルートの `ralph.toml` がないので、この repo の `ralph org` もこの PR から新しい既定で動く。この repo では CI 通過後の自動 merge がもともと承認済みで、codex の座席の扱いもユーザーの決めた既定に従う。

## Affected areas

- `internal/config/config.go`、`internal/config/config_test.go`、`internal/config/defaults_sync_test.go`
- `templates/base/ralph.toml`
- `internal/org/`、`internal/cli/` の、古い既定に頼るテストと「default false」のコメント
- `.claude/skills/org/SKILL.md`、`.claude/skills/director/SKILL.md` と 3 つずつの写し
- `docs/recipes/codex-seat-permissions.md`、`docs/recipes/codex-setup.md` と `templates/base/docs/recipes/` の写し
- `README.md`、`docs/specs/2026-10-07-org-multi-org-director.md`

## Visual review

None (2 つの設定の既定値を入れ替え、文書をそれに合わせる変更で、図にする仕組みの変化がない)

## Design decisions

- D1 テンプレートには新しい値を明示して書く(`codex_verified = true`、`auto_merge = true`)。`TestDefaultsLockStep/org` がテンプレートと `Default()` の一致とキーの有無を見るので、キーを省く形は取らない。
- D2 古い既定に頼っていたテストの直し方: false のときの動き(codex の座席を guarded に限る、director が merge を人に任せる)を確かめるテストは、false を明示して書く形に直し、その動きのテストを残す。既定の値そのものを確かめるテストだけ、期待を true に変える。期待を true にひっくり返すだけで、false の動きのテストを消すことはしない。具体的には(consult の洗い出し、行は main の ae64da70):
  - `ralph.toml` を書かないテストが共有する文言の定数(`internal/cli/org_director_test.go:26,45,61` の `directorAutoMergeOff` と、それを畳んだ `directorNotAttachedLine` など)は、新しい既定の文言に差し替える。`ralph.toml` のない wake、wait、check-line、status のテスト(`org_director_wake_test.go:53,63`、`org_director_wait_line_test.go:226,550`、`org_director_check_line_test.go:209,282`、`status_test.go:726`)はこれで新しい既定を確かめる形になる。false を名乗るテストだけ `auto_merge = false` を書く。
  - `status_test.go:1157,1179` の warn の文「auto_merge (false)」は、表示が `Default()` を読むので true の文に直す。
  - `org_director_test.go:636` の「main の ralph.toml がなく、worktree が true と書く → false」は、既定が true になると main 優先の判別ができなくなるので、「worktree が false と書く → true」に組み替える。
  - codex: `writeCodexOrgConfig`(`internal/cli/org_test.go:1244`)は `codex_verified` を書かないので、`TestOrgSpawn_CLI_EnvelopeRejection_NoModelWarning`(1442 行)の拒否が起きなくなる。そのテストの `ralph.toml` に `codex_verified = false` を明示する。`TestRunDoctorOpts_CodexSandboxCheck_HermeticByDefault`(`doctor_codex_writable_root_test.go:1770`)はテンプレートから init した project で pass か「not needed」を期待しているが、新しいテンプレートでは 692 行のテストと同じ warn になるので、期待を warn に変える(hermetic であることは保つ)。
  - 上の一覧は静的な洗い出しで、S2 の検証で落ちたテストも同じ規則で直す。
- D3 照合の表のテストは、照合する全キーの行を持ち、行ごとに 3 つの変異を当てる。(i) `Default()` のコピーの 1 field だけを変える、(ii) テンプレートの文字列のそのキーの値だけを変える(表の見出しを見て、そのキーの属する表の中の行だけを書き換える)、(iii) テンプレートの文字列からそのキーの行だけを消す。どれも、そのキーを名指しする 1 行だけが返ることを確かめる。行のキーの集合と `orgTemplateProbe` の toml のタグの集合(表の見出しを含む点つきの名前)が一致することを reflect で確かめ、数をコメントに頼らない。(i) は比較の `Default()` 側の取り違えを、(ii)(iii) はテンプレート側の取り違え(比較の相手や toml のタグ)を捕まえるので、既定値が同じキー(例: この PR のあとは `codex_verified` と `auto_merge` がどちらも true、`watcher_enabled` と `director.enabled` もどちらも true)の取り違えも、どちらの側で起きても落ちる(Codex の計画助言 2 を反映)。
- D4 `/director` skill の「2 つの値が違えば `false` として扱い」の規則(headless の director の `--config` と `ralph status` の読む `ralph.toml` が食い違うとき)は変えない。食い違いのときに安全な側に倒す規則で、既定の値とは別の話。
- D5 recipe `codex-seat-permissions.md` は、確かめの手順を残し、「確かめてから true にする」を「既定は true。手順で確かめ、通らなければ false にする」に組み替える。手順の中身(codex の flag の対応、agmsg の DB の writable root)は変えない。
- Critical forks: None(既定の値と範囲は issue で決まっており、残る選択はどれも 1 スライスの範囲で戻せる)。

## Acceptance criteria

- [x] AC1: `ralph.toml` にキーを書かないときの既定が `codex_verified = true` と `auto_merge = true` になる。`internal/config` のテスト(`TestDefault` か新しいテスト)が、キーのない `ralph.toml` と `ralph.toml` のない場合の両方で確かめる。
- [x] AC2: `templates/base/ralph.toml` が `codex_verified = true` と `auto_merge = true` を書き、2 つのキーの前のコメントが新しい既定を「default」として説明する。`TestDefaultsLockStep` が通る。
- [x] AC3: `git grep` で、skill(4 つの写し)、recipe(2 つの写し)、`README.md`、`templates/base/ralph.toml`、`docs/tech-debt/README.md`、Go のコメントに、2 つの設定を「既定 false」「既定の false」「既定では」(guarded 以外を拒否、merge を人に任せる、の意味で)「検証してから有効」「default false」「defaults to false」「(default)」(false に付いたもの)と書いた箇所が残っていない(`docs/plans/archive/`、`docs/reports/`、`docs/insights/`、`docs/evidence/`、spec の印つきの履歴の文は除く)。
- [ ] AC4: PR の本文に release のノートの項があり、既存の project への影響(書いてある false は変わらない)と直し方(手で true にする)が書いてある。
- [x] AC5: 照合の表のテストが、照合する全キーで D3 の 3 つの変異((i) `Default()` 側の値、(ii) テンプレート側の値、(iii) テンプレート側のキーの削除)のそれぞれについて、そのキーを名指しする 1 行だけが返ることを確かめ、表のキーの集合が `orgTemplateProbe` の toml のキーの集合と一致することを確かめる。/test で次の変異を当てて、このテストが落ちることを確かめる: 比較の 1 行の削除、既定値が同じ 2 キーの比較の `Default()` 側の取り違え、同じ 2 キーの比較のテンプレート側の取り違え、probe の 2 つの field の toml のタグの取り違え。
- [x] AC6: `go test ./internal/config/ ./internal/org/ ./internal/cli/`、`./scripts/check-sync.sh`、`./scripts/check-skill-sync.sh` が通る。D2 のとおり、false の動きを確かめるテストが残っている。
- [x] AC7: 新しいテンプレートの `codex_verified = true` と `auto_merge = true` を `false` に書き換えた `ralph.toml` が、2 つとも false で読める(`internal/config` のテスト)。戻し方の手順(Rollout or rollback notes)がこの書き換えを使う。

## Implementation outline

1. S1 照合の表のテスト(変更の前に足し、今の既定で通ることを確かめる): `internal/config/defaults_sync_test.go` に AC5 のテスト(D3)を足す。既存の変異のテストの名前の「whose default is false」は S2 で直す。 — verify: `test -z "$(gofmt -l internal/config)" && go vet ./internal/config/ && GOFLAGS="-timeout=30m $(go env GOFLAGS)" go test ./internal/config/`
2. S2 既定を true にする: `internal/config/config.go` の `Default()` と 2 つの field のコメント、`templates/base/ralph.toml` の値とコメント、`internal/config` のテスト(`TestDefault` など、変異のテストの名前、AC7 のテスト)、古い既定に頼っていた `internal/cli` のテストを D2 で直す。`internal/org` のテストは `Default()` の 2 つの値を読まないので、このスライスの検証に入れず、全体は /test で回す。 — verify: `test -z "$(gofmt -l internal/config internal/cli)" && go vet ./internal/config/ ./internal/cli/ && GOFLAGS="-timeout=30m $(go env GOFLAGS)" go test ./internal/config/ ./internal/cli/`
3. S3 Go のコメント: `internal/org` と `internal/cli` で 2 つの設定を「default false」と書いたコメントを直す(`git grep -nE 'default(s to)? ?\(?false|false \(default\)|\(true and false\)'` などで洗い出す)。コメントだけで、コードは変えない。 — verify: `test -z "$(gofmt -l internal/org internal/cli)" && go vet ./internal/org/ ./internal/cli/ && go build ./...`
4. S4 文書: skill の 2 つ(`.claude/skills/` と `templates/base/.claude/skills/` を直し `./scripts/sync-skills.sh`)、recipe の 2 つとその写し、`README.md`、spec の印。 — verify: `./scripts/check-skill-sync.sh && ./scripts/check-sync.sh && GOFLAGS="-timeout=30m $(go env GOFLAGS)" go test ./internal/org/ -run 'TestDirectorSkill|TestOrgSkill|TestSendDefaults|TestReportNames'`

## Verify plan

- Static analysis checks: `./scripts/run-static-verify.sh`(gofmt、go vet、golangci-lint、staticcheck、shellcheck、check-sync、check-skill-sync、check-template-purity)。
- Spec compliance criteria to confirm: AC1〜AC6。issue #274 の受け入れ条件 4 つ(既定、テンプレート、skill と recipe、release のノート)。
- Documentation drift to check: AC3 の `git grep`。`/director` skill の「merge と auto_merge」節と `/org` skill の codex の座席の段落が新しい既定と食い違わないこと。`docs/tech-debt/README.md` の codex の座席と director の行(136、260、268 行付近)が今も正しいこと。
- Evidence to capture: `run-static-verify.sh` の出力、AC3 の `git grep` の結果。

## Test plan

- Unit tests: `TestDefault`(既定が true)、照合の表のテスト(AC5)、`TestDefaultsLockStep`、`TestOrgTemplateDrift_DetectsChangedAndMissingKeys`。
- Integration tests: `internal/cli` の status、director、merge、doctor のテストのうち、`ralph.toml` のない場合やキーのない場合を読むもの。
- Regression tests: D2 で false を明示した、codex の座席を guarded に限るテスト(`internal/org/permissions_test.go` など)と、`auto_merge = false` の表示のテスト。
- Edge cases: キーのない `ralph.toml` と `ralph.toml` のない場合、`codex_verified = false` を明示した `ralph.toml`(guarded に限られる)、`auto_merge = false` を明示した `ralph.toml`。AC5 の変異(比較の 1 行の削除、`codex_verified` と `auto_merge` の比較の取り違え)。
- Evidence to capture: `./scripts/run-test.sh` の出力、AC5 の変異の結果。

## Risks and mitigations

- R1: `ralph.toml` のない project とキーを書いていない project で、codex の座席が `--sandbox workspace-write --ask-for-approval never` で動き、director が聞かずに merge するようになる。軽減: ユーザーの決めた既定。release のノートに書く。`ralph org merge` は PR、checks、保護を確かめてから merge する。agmsg の DB が writable root にないときは `ralph doctor` の検査が warn する。
- R2: 古い既定に頼るテストが `internal/cli` と `internal/org` に散っていて、S2 の検証で初めて分かる。軽減: S2 の検証コマンドが 3 つのパッケージ全体を回す。D2 の方針で、false の動きのテストを消さない。
- R3: `internal/org` と `internal/cli` のテストは時間がかかり、同じ Mac のほかのセッションと CPU を分け合う。軽減: 30 分の timeout か既知の不安定なテストで落ちたら 1 回だけ打ち直して記録する。
- R4: skill の 4 つの写しと README は、ほかのセッションの PR と衝突しやすい。軽減: 衝突したら両方の変更を残し、`./scripts/sync-skills.sh` で写しを作り直して、check-skill-sync と check-sync を打ち直す。
- R5: テンプレートの既定のままの project とこの repo で、`ralph doctor` が agmsg の writable root を warn するようになる。軽減: issue がこの検査を残すと決めている。release のノートに書く。warn は doctor の終了コードを変えず、CI と scripts は doctor を呼ばない。
- R6: PR を revert しても、新しい版の `ralph init` が書いた `ralph.toml` には `true` が残り、起動済みの codex の座席は spawn のときの flag のまま動く(Codex の計画助言 1)。軽減: 戻し方の手順を下に書き、AC7 で書き換えのあとの読み込みを確かめる。

## Rollout or rollback notes

- release のノートの項(PR の本文): 「`[org.permissions].codex_verified` と `[org.director].auto_merge` の既定を true にした。既存の project の `ralph.toml` には false が書いてあるので変わらない。新しい既定にしたいときは、その 2 行を手で true にする」。`ralph upgrade` はテンプレートの変化を advisory の diff として出す。R5 の warn も 1 文書く。
- 戻し方: コードはこの PR を revert する。ただし revert では、新しい版の `ralph init` が作った project の `ralph.toml` の `true` は戻らない(seed なので `ralph upgrade` も書き換えない)。その project で元の動きに戻すには、(1) `ralph.toml` の `codex_verified` と `auto_merge` を手で `false` にする、(2) codex の座席は spawn のときに flag が決まるので、動いている座席を `ralph org stop` で止めて spawn し直す(false では、codex の座席は guarded の役でしか spawn できない。以前と同じ)、(3) `ralph org director status` の `auto_merge:` の行と、`ralph org report` に出る座席の `permission_mode`(manifest の spawned の記録、`internal/org/spawn.go:1722`)で、効いている値を確かめる。director は merge の前に毎回 `auto_merge` を読むので、(1) のあとの merge から効く。PR の本文の release のノートにもこの手順を書く。

## Open questions

- なし

## Progress checklist

- [x] Plan reviewed — consult-plan-defaultstrue: 直してから進める。D2 に共有定数・status の warn・main 優先のテスト・codex の 2 テストの扱いを書き、Scope に `/org` skill 1541 行付近と tech-debt:144 を足し、S2 の検証を `internal/config` と `internal/cli` に絞り、AC3 の grep の語を足し、release のノートに doctor の warn を足した。Codex plan advisory: MEDIUM 2 件をどちらも反映(1: revert で `ralph.toml` の true が戻らない → 戻し方の手順、R6、AC7。2: `Default()` 側だけの変異ではテンプレート側の取り違えを捕まえない → D3 と AC5 を 3 つの変異に)。
- [x] Plan approved — 自動承認(ユーザーの指示)。digest は Approved の行。Critical forks: None(consult も同意、ユーザーに聞く分かれ目なし)
- [x] Branch created (feat/codex-verified-auto-merge-on, worktree .claude/worktrees/codex-verified-auto-merge-on)
- [x] Implementation started
  - S1 29fd179e(impl-274-s1): `TestOrgTemplateDrift_EachKeyIsComparedOnBothSides`(19 行、3 つの変異)、`tomlKeys`(reflect で probe の点つきの名前)、`editTemplateKey`(表の範囲の中で 1 行だけ書き換えか削除、複数行の配列も扱う)。表の名前と素のキーは点つきの名前から作る。bool の行のテンプレート側の値は `!Default()` から作るので、S2 のあとも直さずに通る。`go test -overlay` の写しで変異 m1〜m6(`Default()` 側とテンプレート側の取り違え、比較の 1 行の削除、toml タグの取り違え、表の 1 行の削除)がすべて落ちることを確かめた。指定の検証コマンドは通り、orchestrator も `-run 'EachKey|DetectsChanged|LockStep' -count=1` で確かめた。コメントの「19」は残した(S2 か後の直しで扱う)。
  - S2 cb8d258e(impl-274-s2): `Default()` とテンプレートの 2 つの値を true に、コメントを新しい既定に。`internal/config` に `TestLoad_CodexVerifiedAndAutoMergeDefaultToTrue`(AC1: ralph.toml なし、[org] なし、表はあってキーなしの 3 つ)と `TestLoad_TemplateWithBothKeysFalse`(AC7)を足した。`internal/cli` のテスト 9 本を D2 で直した。false を明示して残したもの: `writeCodexOrgConfig`(`codex_verified = false`)、`TestOrgDirector_StatusShowsAutoMerge` と `TestStatusCmd_DirectorLineShowsAutoMerge` の false の場合、main 優先のテストを「worktree が false → true」に組み替え、config の round trip。期待を true にしたもの: 共有の定数と、ralph.toml のないテスト、status の warn の文、merge の 3 本、doctor の hermetic のテスト(warn に。hermetic の確認は残る)。disabled 系のテストは既定の文(has no effect 付き)に変え、false を名乗る 2 件(status_test の "disabled, auto_merge = false" と StatusShowsAutoMerge の disabledOff)だけ `auto_merge = false` を明示した(consult-plan-defaultstrue が 14:56Z に同意)。検証コマンドは rc 0(14:30Z 以降に実行、orchestrator の確かめではキャッシュの通過)。
  - ce6e972d(impl-274-s2): S1 のテストの doc comment の「既定値が同じキーの例」に `codex_verified` と `auto_merge` を足した(consult の任意の提案)。コメントだけの別コミット。
  - S2 の検証: 指定のコマンドを 14:46Z に実行し rc 0(`internal/config` 0.6 s、`internal/cli` 577 s)。D2 の一覧にない `org_merge_test.go`(ralph.toml のない merge の 3 本)と `org_director_headless_test.go`(`auto_merge: true` の 2 か所)も同じ規則で直した。`-overlay` で、AutoMerge の既定を false に戻すと AC1 のテストと `TestDefault_OrgDirector`・`TestDefaultsLockStep` が落ち、`writeCodexOrgConfig` の `codex_verified = false` を消すと EnvelopeRejection が落ちることを確かめた。
  - S3 920a4dcb(impl-274-s3): `internal/org/permissions.go`(CodexVerified の説明、fail-closed は false のときだけ)、`internal/cli/doctor_shell_alias.go`(default true、既定の project では edits と autonomous も起動する)、`internal/cli/org.go`(「(both true)」)のコメント。検証コマンド rc 0。実行時の文 `internal/org/permissions.go` のエラー文「requires … codex_verified=true; only guarded is allowed until then (fail-closed; verify … first …)」は、false を明示した人だけが見る条件つきの文で偽ではないので変えていない(PR の Notes に書く)。
  - S4 f99ac82d(impl-274-s4): `/org` と `/director` の skill(4 つの写し、`sync-skills.sh`)、recipe `codex-seat-permissions.md`(題を「opt out if it fails」に、既定 true、通らなければ false)と `codex-setup.md`(各 2 つの写し)、`README.md:124`、spec の FR-8・受け入れ条件・Security の印と「2026-10-28 に決めたこと」の注、`docs/tech-debt/README.md:144` の (b)。AC3 の grep は一致なし。`/org` skill の 225 行の「既定 false」も同じ grep に掛かったので直した。check-skill-sync、check-sync、check-template-purity、skill の文のテストが通った。気づき: `docs/tech-debt/README.md:136` の「An operator who has already set `codex_verified = true`」は /verify で見る。
- [x] Review artifact created — self-review 7ca29c41(2 回目、Merge: yes、所見 0。1 回目の M1・L1〜L4 は F1 で閉じた)
  - self-review b49fd6c2(cycle 1 の 1 回目): Merge: yes、MEDIUM 1(M1)、LOW 4(L1〜L4)。consult-plan-defaultstrue の判定に従い、修正スライス F1 で全部直してから self-review をもう 1 回回し、/verify に進む(まだ /verify 以降を走らせていないので、増えるのは self-review の 1 回だけ。cycle-count は増やさない)。
  - Deviation(M1): `ralph status` は `ralph.toml` が読めないとき `Default()` で director の行を出していた。既定が true になると、人が false と書いた file が壊れていても「人に聞かずに merge する」と出る。F1 で、`statusOrgConfig` と `statusDirectorConfig` の 2 つの fallback で `AutoMerge` だけ false にし、stderr の warning が「ralph.toml を読めなかったので false と出す(既定ではない)」と理由を言うようにする。`Enabled` は既定の true のまま。`ralph org merge` と `ralph org director status` は読めない file でエラーになるので変わらない。`/director` skill は変えない(表示が false になるので D4 の規則で足りる)。skill だけで直す代案は、`--json` の `director.auto_merge` と stderr を読まない側に効かないので取らない。
  - F1 958d9bff(impl-274-f1): M1(`status.go` の 2 つの fallback で `AutoMerge = false`、warning が「not the default, because … could not be read」と言う、doc comment、`status_test.go` の 2 つの fallback のテストが false の行と JSON と stderr を固定)、L1(`permissions.go` のコメントで、既定 true の理由は #274 の人の決定、2026-09-18 の検証は flag の対応の証拠と分けた)、L2(`testOrgConfig()` と `codexGuardedOrg` に `CodexVerified: false` を明示、`spawn_test.go` のコメント)、L3(doctor の出力を「gets only while … codex_verified is true, the default」に、固定していたテスト 4 か所)、L4(「19」を消し、同じ既定値の組の例を直した。ack_minutes は 10、interval_seconds は 30 で同じでないので、ack_minutes を 10 の組に、max_total_seats と interval_seconds を 30 の組にした)。検証コマンドは 2 回 rc 0、`-overlay` で 2 つの fallback の `AutoMerge = false` を 1 行ずつ消すとそれぞれのテストが落ちることを確かめた。範囲外の気づき: `internal/org/director_start_test.go:370-378` のケース名「codex autonomous before codex_verified」と `feature_test.go:1639`「an unverified codex leader」は false を明示したテストの名前で、偽ではないので残す。
- [x] Verification artifact created — verify e802b228(cycle 1、pass。`run-static-verify.sh` rc 0、golangci-lint 0 件。AC1・AC2・AC3・AC7 を満たす、AC5・AC6 は静的の部分、AC4 は PR で)。LOW の D-1・D-2(`docs/tech-debt/README.md:136` の doctor の文の引用と「already set」の言い方)は /sync-docs で直す。INFO の D-3(`ralph status --help` に読めないときの false がない)は /sync-docs で任意、D-4(`permissions.go` のエラー文の「until then … first」)は PR の Notes に書く。
- [x] Test artifact created — test 9d78a1c3(cycle 1、pass。`run-test.sh` rc 0 を 1 回で、shell 45 本 4,768 件、Go 9 パッケージ、cli 559 s・org 654 s。AC5 の overlay の変異 17 個がすべて落ちた。(b) と #273 で生き残った M1・M3・M5 を落とすのは新しい表のテストだけ。false の動きのテストは名指しの実行で通った)
- [ ] PR created
