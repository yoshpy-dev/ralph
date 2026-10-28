# Verify report: codex-verified-auto-merge-on

- Date: 2026-10-29(静的検査の実行は 2026-10-28T16:00:25Z、UTC)
- Plan: `docs/plans/active/2026-10-28-codex-verified-auto-merge-on.md`(issue #274、AC1〜AC7、Progress の S1〜S4 と修正スライス F1、Deviation M1)
- Verifier: verifier サブエージェント(`/verify`、パイプラインの cycle 1)
- Scope: `git diff ae64da70...HEAD`(HEAD 7ca29c41、14 コミット、38 ファイル、+872/-317)。静的検査は `./scripts/run-static-verify.sh` の既定(changed)で打った。`templates/base/ralph.toml` が言語に分類されないので full に落ち、golang の verifier が走った。テストは走らせていない(`/test` の役目)。
- Evidence: `docs/evidence/verify-2026-10-28-codex-verified-auto-merge-on.log`(gitignore 対象。run-verify 自身の記録は `docs/evidence/verify-2026-10-28-160025.log`)

## Spec compliance

| Acceptance criterion | Status | Evidence |
| --- | --- | --- |
| AC1: キーを書かないときの既定が `codex_verified = true` と `auto_merge = true`。キーのない `ralph.toml` と `ralph.toml` のない場合の両方を `internal/config` のテストで確かめる | 満たす(静的に確認。実行は `/test`) | `internal/config/config.go:566`(`CodexVerified: true`)、`:576`(`AutoMerge: true`)。`TestLoad_CodexVerifiedAndAutoMergeDefaultToTrue`(`internal/config/config_test.go:123`)が、`ralph.toml` なし、`[org]` なし、表はあるがキーなし、の 3 つで 2 つとも true を求める。`Default()` の値は `TestDefault_OrgPermissions`(`:527`)と `TestDefault_OrgDirector`(`:913`)、表のない場合は `TestLoad_OrgPermissionsMissingSectionBackfillsDefault`(`:574`)と `TestLoad_OrgDirectorMissingSectionBackfillsDefault`(`:948`)が見る。`go vet ./...` がテストファイルもコンパイルして通った |
| AC2: テンプレートが 2 つとも true を書き、前のコメントが新しい既定を default と説明する。`TestDefaultsLockStep` が通る | 満たす(`TestDefaultsLockStep` の実行は `/test`) | `templates/base/ralph.toml:97`(`codex_verified = true`)、`:163`(`auto_merge = true`)。コメントは `:81-85`(codex の座席は true の間すべての mode、false で guarded だけ)、`:88-96`(「true (default)」「An absent codex_verified loads as true」)、`:155-162`(「auto_merge = true (default)」「An absent auto_merge loads as true」)。tomllib で読むと 2 つとも `True` |
| AC3: skill、recipe、`README.md`、テンプレート、`docs/tech-debt/README.md`、Go のコメントに「既定 false」の類いが残らない | 満たす | 計画の語(既定 false、既定の false、検証してから有効、default false、defaults to false、false (default)、false(既定))で組んだ `git grep -nE`(コマンドの全文は evidence の log の「AC3 grep」)を、skill の 4 つの写し、recipe の 2 つの写し、`README.md`、テンプレート、tech-debt、`internal/`・`cmd/` の Go に掛けた。当たりは 4 行で、どれも別の設定(`internal/cli/status.go:384` の `IncludeDryRun`、`internal/config/config_test.go:20`、`:23`、`:34` の `require_*_cli`)。「既定では」の当たりは `/org` skill `:1286`(merge-order の `--all`)だけで、2 つの設定とは関係ない。広げた確認は「Observational checks」に書いた |
| AC4: PR の本文に release のノートの項がある | 確認できない | PR はまだない。項に使う文は計画の Rollout notes(`:117-118`)にある |
| AC5: 照合の表のテストが全キーで 3 つの変異を当て、表のキーの集合が probe の toml のキーの集合と一致することを確かめる | 満たす(静的に確認。変異の実行は `/test`) | `TestOrgTemplateDrift_EachKeyIsComparedOnBothSides`(`internal/config/defaults_sync_test.go:404`)。表は 19 行(`:420-454`)、キーの集合の比較は `tomlKeys`(`:501`)で取った probe の葉と(`:460-463`)、3 つの変異は `:482`(`Default()` 側)、`:488`(テンプレートの値)、`:494`(テンプレートのキーの削除)。probe の toml のタグは 24 個で、葉 19 個と表 5 個(`org`、`permissions`、`watchdog`、`director`、`approval`)。bool の行のテンプレート側の値は `!Default()` から作るので、既定を反転しても空振りしない。`editTemplateKey`(`:526`)を python3 -I に写して実テンプレートに当てると、19 キーとも表の中に 1 回だけあり、書き換えた写しと消した写しはどちらも tomllib で読め、実テンプレートとの差はそのキー 1 つだけだった(19/19)。計画の 4 つの変異の実行は `/test` の分。S1 の implementer が `-overlay` で m1〜m6 を当てた記録は Progress にある |
| AC6: `go test` の 3 パッケージ、`check-sync.sh`、`check-skill-sync.sh` が通る。false の動きのテストが残る | 一部(verify の分は満たす) | `./scripts/check-sync.sh` は単独で rc=0(DRIFTED 0)、`./scripts/check-skill-sync.sh` は rc=0(14 skill)。`go test` は `/test`。false の動きのテストは残っている。codex: `internal/cli/org_test.go:1257`(`writeCodexOrgConfig` が `codex_verified = false` を書く)を `TestOrgSpawn_CLI_EnvelopeRejection_NoModelWarning`(`:1445`)が使う。`internal/org/envelope_test.go:28`(`testOrgConfig()` の `CodexVerified: false`)、`internal/org/spawn_test.go:3921`(`codexGuardedOrg`)、`internal/org/permissions_test.go:80`、`:92`(「when codex_verified is false」)。auto_merge: `internal/cli/org_director_test.go:47-58`(`directorNotAttachedOffLine`、`directorDisabledOffLine`)、`TestOrgDirector_StatusShowsAutoMerge`(`:580`)の false の組、`internal/cli/status_test.go:862`、`:873`。round trip は `config_test.go:599`(false と true)と `:980`(全キー、`auto_merge = false`) |
| AC7: テンプレートの 2 行を false に書き換えた `ralph.toml` が 2 つとも false で読める。戻し方の手順がこの書き換えを使う | 満たす(静的に確認。実行は `/test`) | `TestLoad_TemplateWithBothKeysFalse`(`internal/config/config_test.go:160`)。置き換える `\ncodex_verified = true\n` と `\nauto_merge = true\n` はテンプレートにそれぞれ 1 回だけある(python3 -I で数えた)。置き換えた写しを tomllib で読むと `False False`。計画の戻し方(`:118`)の (1) がこの書き換え |
| Deviation M1: `ralph status` は `ralph.toml` が読めないとき auto_merge を false と出す | 満たす(静的に確認) | `internal/cli/status.go:266`(`statusOrgConfig` の fallback)と `:299`(`statusDirectorConfig` の fallback)が `AutoMerge = false` を入れ、warning(`:267-270`、`:302-304`)が「auto_merge as false (not the default), because … could not be read」と理由を言う。`Enabled` は既定の true のまま。テストは `TestStatusCmd_DirectorEnabledFromTheMainWorktree`(`status_test.go:1146`)と `TestStatusCmd_DirectorFallsBackToTheDefaultWhenRalphTomlDoesNotLoad`(`:1190`)で、行、`--json` の `auto_merge`、stderr を固定する |

issue #274 の受け入れ条件との対応: (1) キーがないときの既定が true(`internal/config` のテスト)→ AC1。(2) テンプレートの値とコメント → AC2。(3) skill と recipe に「既定 false」が残らない → AC3(grep の当たりのうち 2 つの設定を言うものは 0 件)。(4) release のノートに既存の project への影響と直し方 → AC4、PR の本文で確かめる。

## Static analysis

| Command | Result | Notes |
| --- | --- | --- |
| `./scripts/run-static-verify.sh` | pass(rc=0) | 2026-10-28T16:00:25Z、HEAD 7ca29c41。shellcheck、hook の `sh -n` 20 本、guard の awk 2 本、`settings.json` の jq 2 本、Codex の hook 系 3 本、`check-sync.sh`、`check-pipeline-sync.sh`、`check-skill-sync.sh`(14 skill)、`check-template-purity.sh`、tech-debt の plan 参照がすべて OK。golang: `gofmt: ok`、golangci-lint `0 issues.`、go vet と staticcheck は成功時に何も出さない(Skipping の行はなく、どちらも走った)。branch の secret scan は `ae64da70..7ca29c41` で clean |
| `./scripts/check-sync.sh`(単独) | pass(rc=0) | IDENTICAL 172、DRIFTED 0、ROOT_ONLY 0、TEMPLATE_ONLY 11、KNOWN_DIFF 5 |
| `./scripts/check-skill-sync.sh`(単独) | pass(rc=0) | 14 skill が一致 |
| `./scripts/check-template-purity.sh`(単独) | pass(rc=0) | テンプレートに meta-repo 固有の参照なし |
| `git diff --check ae64da70...HEAD` | pass(rc=0) | 出力なし |
| `cmp`(recipe 2 本のルートとテンプレート) | pass | `codex-seat-permissions.md` と `codex-setup.md` はどちらも byte 一致 |

## Documentation drift

| Doc / contract | In sync? | Notes |
| --- | --- | --- |
| `docs/tech-debt/README.md:136` | no(LOW D-1、D-2) | 下の D-1、D-2 |
| `docs/tech-debt/README.md:144` | yes | (b) は「falls back to the built-in defaults (`codex_verified = true` since #274) and the check judges those instead of the project's `ralph.toml`」に直っている |
| `docs/tech-debt/README.md:260` | yes | テンプレートの `[org.permissions].default` が autonomous であることと、codex の director が `codex_verified` によらず guarded に限られることは、どちらもこの PR で変わらない |
| `docs/tech-debt/README.md:268`、`:270` | yes | `:268` の「whatever `[org.permissions].codex_verified` says」は今も正しい(`internal/org/director_start.go:1005-1009` が `permissionArgsForDriver` の前に断る)。`:270` は「With `codex_verified = true`」の条件つきの文 |
| `docs/tech-debt/README.md:241-249`(8a-3a の注) | yes | どれも「`auto_merge = true` のとき」の条件で書かれ、既定が false であることを前提にした文はない |
| `/director` skill「merge と auto_merge」(`.claude/skills/director/SKILL.md:813-826`、写し 3 つ) | yes | 「(既定)」が true の側に移った。D4 の「2 つの値が違えば false」は変わっていない |
| `/org` skill(`.claude/skills/org/SKILL.md:131-134`、`:225`、`:1541-1546`、写し 3 つ) | yes | 既定 true、false にすると guarded だけ、と書く |
| `docs/recipes/codex-seat-permissions.md:1-10`、`:314-334`(写し 1 つ) | yes | 既定 true、確かめて通らなければ false。`ralph init` の前の版が false を書き、`ralph upgrade` はその行を変えないことも書いてある |
| `docs/recipes/codex-setup.md:200-204`(写し 1 つ)、`README.md:124` | yes | 「(default `true`)」「(default true)」 |
| spec `docs/specs/2026-10-07-org-multi-org-director.md:129`、`:132`、`:193`、`:310` | yes(履歴の印) | 旧文は残し、2026-10-28 の印と「2026-10-28 に決めたこと」の注で新しい既定を言う。AC3 の除外に当たる |
| `internal/config/config.go:100-105`、`:113-123`、`:181-188` | yes | field のコメントは既定 true と false にする条件を書く |
| `internal/org/permissions.go:64-67`、`:105-125` | yes | fail-closed は false のときだけ。既定 true の理由は #274 の決定、と書く |
| `internal/org/permissions.go:147`、`:152`(実行時のエラー文) | yes(INFO D-4) | 下の D-4 |
| `internal/cli/doctor_shell_alias.go:573-579`、`:714-719`、`internal/cli/org.go:211` | yes | Detail の文は「only while `[org.permissions].codex_verified` is true, the default」 |
| `ralph status --help`(`internal/cli/status.go:66-71`) | yes(INFO D-3) | 下の D-3 |
| 計画の AC と Progress のチェックボックス | INFO D-5 | 下の D-5 |

LOW の細目(どちらも文書だけで、動作には関係しない。`/sync-docs` で直す):

- D-1: `docs/tech-debt/README.md:136` の Debt item の欄は、doctor の Detail の文を「("… once codex_verified = true … a guarded seat runs with the alias's value …")」と引用している。F1(L3)で実装の文は「(modes a codex seat gets only while [org.permissions].codex_verified is true, the default)」に変わった(`internal/cli/doctor_shell_alias.go:714-715`、`:718-719`)ので、前半の引用は実装に合わない。後半の「a guarded seat runs with the alias's value」はこの PR の前から言い換えだった(実装は「a guarded seat silently runs with the alias's sandbox」)。勧める直し方は、引用の括弧を次に差し替えること: `("… only while [org.permissions].codex_verified is true, the default … a guarded seat silently runs with the alias's sandbox …")`。
- D-2: 同じ行の Impact の欄の書き出し「An operator who has already set `codex_verified = true` and assigned a `guarded` role in `[org.permissions].roles` gets no configuration-specific warning;」は、true が人の書く opt-in だった頃の言い方で、キーを書かない project(既定で true)が抜ける。勧める文: 「An operator whose `[org.permissions].codex_verified` is `true` (the default since #274, whether written or left out) and who assigned a `guarded` role in `[org.permissions].roles` gets no configuration-specific warning;」。同じ欄の残りと Debt item の「nor whether `[org.permissions].codex_verified` is really `true` on this project」は、false を書いた project があるので今も正しい。

INFO の細目(どれもマージを止めない):

- D-3: `ralph status --help`(`internal/cli/status.go:66-71`)は auto_merge を「read by the same rule as enabled」と書き、読めないときに false と出すことは書いていない。warning の文が理由を言うので足りる。self-review の Follow-ups (2) と同じで、足すかは `/sync-docs` が決める。
- D-4: `internal/org/permissions.go:147`、`:152` のエラー文「requires [org.permissions].codex_verified=true; only guarded is allowed until then (fail-closed; verify the codex flag mapping on this machine first …)」は、false の project でだけ出る条件つきの文で、偽ではない。S3 で残すと決め、PR の Notes に書く予定(Progress)。「until then」と「first」は opt-in の頃の言い方なので、直すなら「only guarded is allowed while it is false」の形になる。
- D-5: 計画の AC1〜AC7 はすべて `[ ]` のまま、Progress の「Review artifact created」も `[ ]` だが、self-review の report は b49fd6c2 と 7ca29c41 にある。記録の遅れで、PR の前に orchestrator が付ける。

## Observational checks

- `editTemplateKey` の写し: python3 -I と tomllib で写し、19 キーそれぞれについて、表の中の出現回数(すべて 1)、書き換えた写しと消した写しの読み込み、実テンプレートとの差(すべてそのキー 1 つ)を確かめた。出力は evidence の log の末尾。
- AC3 を広げた確認: `codex_verified` か `auto_merge` が出る行のうち、前後 180 文字に「default」「既定」「opt-in」「unlock」「until」「fail-closed」「人が merge」などがあるものを、tech-debt、README、recipe、skill、rules、`docs/quality`、`AGENTS.md`、`.ralph/core`、role prompt、spec から拾った。どれも既定 true を言うか、値を条件にした文か、spec の印つきの履歴だった。role prompt(`internal/org/prompts/`)と `.claude/rules/ralph/` には 2 つの設定の既定を言う文がない。
- doctor の hermetic のテスト(`internal/cli/doctor_codex_writable_root_test.go:1770`)が求める文「~/.codex/config.toml does not exist, so no writable root covers the agmsg store」は、実装の文(`internal/cli/doctor_codex_writable_root.go:834`、`:840` の `"%s does not exist, so no writable root covers the agmsg store %s%s…"`)と合う。実行は `/test`。
- `EnvelopeSummary`(`internal/org/envelope_summary.go:19`)は `config.Default().Org` を読むテスト(`internal/org/prompts_test.go:260`)があるが、`codex_verified` も `auto_merge` も出さないので、既定の反転の影響を受けない。

## Coverage gaps

- `go test ./internal/config/ ./internal/cli/ ./internal/org/` は走らせていない。`/test` が走らせる。AC5 の 4 つの変異(比較の 1 行の削除、同じ既定値の 2 キーの `Default()` 側とテンプレート側の取り違え、probe の toml タグの取り違え)の実行も `/test`。
- AC4(PR の本文の release のノート)は PR の作成のあとでしか確かめられない。
- この repo にはルートの `ralph.toml` がないので、この PR のあと、この repo の `ralph org` は auto_merge true、codex の座席は autonomous で動く。計画の Assumptions が受け入れた動きで、実機では試していない。
- テンプレートの既定のままの project で `ralph doctor` の「Codex sandbox (agmsg writable root)」が warn になること(R5)は、hermetic のテストの期待として読んだだけで、実行していない。

## Verdict

- Verdict: pass
- Verified: AC1・AC2・AC7 の値と、それを確かめるテストの中身(`go vet` のコンパイルを含む)、AC3(計画の語の `git grep` の当たり 4 行はどれも別の設定、広げた確認も当たりなし)、AC5 のテストの構造と `editTemplateKey` の写しでの空振りのなさ、AC6 のうち `check-sync.sh` と `check-skill-sync.sh` と false の動きのテストが残っていること、Deviation M1 の実装とテスト、静的検査の全体(`run-static-verify.sh` rc=0、単独の 3 本も rc=0)、issue #274 の受け入れ条件 1〜3
- Partially verified: AC6(テストの実行は `/test`)、AC5(変異の実行は `/test`)、文書のずれ(LOW 2 件 D-1、D-2 は `/sync-docs` で直す)
- Not verified: AC4 と issue #274 の受け入れ条件 4(PR の本文がまだない)、`go test` の実行結果、AC5 の変異の実行結果
