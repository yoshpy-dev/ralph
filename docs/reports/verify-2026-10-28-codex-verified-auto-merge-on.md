# Verify report: codex-verified-auto-merge-on

- Date: 2026-10-29(cycle 2 の静的検査の実行は 2026-10-28T17:30:29Z、UTC)
- Plan: `docs/plans/active/2026-10-28-codex-verified-auto-merge-on.md`(issue #274、AC1〜AC7、Progress の S1〜S4、修正スライス F1・F2、Deviation M1)
- Verifier: verifier サブエージェント(`/verify`、パイプラインの cycle 2)。cycle 1 の report(e802b228、HEAD 7ca29c41、Verdict pass、LOW 2 件)をこの内容で置き換えた
- Scope: この branch の変更は `git diff 09dc0260...HEAD`(HEAD aa4ec8d3、27 コミット(うち merge 1)、42 ファイル、+1436/-343)。cycle 1 のあとに入ったものは、e3363a99(sync-docs、`docs/tech-debt/README.md:136`)、a91f6694(`internal/cli/org.go` の doc comment、D-6)、aea6b494(F2、`resolveOrgConfig`)、記録のコミット、aa4ec8d3(origin/main の #282 init-pack-choice の merge)。cycle 1 のあとのコードの差は 3 ファイル、+120/-30(`internal/cli/org.go`、`org_test.go`、`status_test.go`)。静的検査は `./scripts/run-static-verify.sh` の既定(changed)で打ち、`templates/base/ralph.toml` が言語に分類されないので full に落ちて golang の verifier が走った。テストは走らせていない(`/test` の役目)
- Evidence: `docs/evidence/verify-2026-10-28-codex-verified-auto-merge-on.log`(gitignore 対象。cycle 2 の内容で書き直した。run-verify 自身の記録は `docs/evidence/verify-2026-10-28-173029.log`)

## Spec compliance

cycle 1 のあとで、`templates/base/ralph.toml`、`internal/config/`、`internal/org/`、`internal/cli/status.go`、`internal/cli/doctor_shell_alias.go`、skill、recipe、spec は変わっていない(`git diff --quiet e802b228 HEAD -- <それらのパス>` が rc=0)。AC1・AC2・AC3・AC5・AC7 の根拠の行番号は cycle 1 と同じで、HEAD で引き直した。

| Acceptance criterion | Status | Evidence |
| --- | --- | --- |
| AC1: キーを書かないときの既定が `codex_verified = true` と `auto_merge = true`。キーのない `ralph.toml` と `ralph.toml` のない場合の両方を `internal/config` のテストで確かめる | 満たす(静的に確認。実行は `/test`) | `internal/config/config.go:566`(`CodexVerified: true`)、`:576`(`AutoMerge: true`)。`TestLoad_CodexVerifiedAndAutoMergeDefaultToTrue`(`internal/config/config_test.go:123`)が、`ralph.toml` なし、`[org]` なし、表はあるがキーなし、の 3 つで 2 つとも true を求める。`Default()` の値は `TestDefault_OrgPermissions`(`:527`)と `TestDefault_OrgDirector`(`:913`)、表のない場合は `:574` と `:948` が見る。F2 のあと、`--config` のない `ralph org` の動詞と `ralph status` の経路でも、`./ralph.toml` がないとき(`fs.ErrNotExist`)だけが既定になる(`TestResolveOrgConfig_UnreadableRalphTomlIsAnErrorNotTheDefaults` の最初の確認、`internal/cli/org_test.go:1133`) |
| AC2: テンプレートが 2 つとも true を書き、前のコメントが新しい既定を default と説明する。`TestDefaultsLockStep` が通る | 満たす(`TestDefaultsLockStep` の実行は `/test`) | `templates/base/ralph.toml:97`(`codex_verified = true`)、`:163`(`auto_merge = true`)。コメントは `:81-85`、`:88-96`(「true (default)」「An absent codex_verified loads as true」)、`:155-162`(「auto_merge = true (default)」「An absent auto_merge loads as true」) |
| AC3: skill、recipe、`README.md`、テンプレート、`docs/tech-debt/README.md`、Go のコメントに「既定 false」の類いが残らない | 満たす | 計画の語(既定 false、既定の false、検証してから有効、default false、defaults to false、false (default)、false(既定))で組んだ `git grep -nE`(コマンドの全文は evidence の log の「AC3 grep command」)を HEAD で打ち直した。当たりは cycle 1 と同じ 4 行で、どれも別の設定(`internal/cli/status.go:384` の `IncludeDryRun`、`internal/config/config_test.go:20`、`:23`、`:34` の `require_*_cli`)。merge で入った #282 の変更(`README.md:121`、`templates/base/docs/recipes/adding-a-language-pack.md`、`internal/cli/init.go` など)にも当たりはない |
| AC4: PR の本文に release のノートの項がある | 確認できない | PR はまだない。項に使う文は計画の Rollout notes(`:117-118`)と sync-docs の report の「Release notes」にある |
| AC5: 照合の表のテストが全キーで 3 つの変異を当て、表のキーの集合が probe の toml のキーの集合と一致することを確かめる | 満たす(静的に確認) | `TestOrgTemplateDrift_EachKeyIsComparedOnBothSides`(`internal/config/defaults_sync_test.go:404`)。表は 19 行(`:420-454`)、キーの集合の比較は `tomlKeys`(`:501`)で取った probe の葉と(`:460-463`)、3 つの変異は `:482`、`:488`、`:494`。`editTemplateKey`(`:526`)を python3 -I に写して HEAD のテンプレートに当て直すと 19/19 で「表の中に 1 回だけ、差はそのキーだけ」。計画の 4 つの変異の実行は、cycle 1 の `/test`(9d78a1c3)が overlay の変異 17 個すべてで落ちることを記録している |
| AC6: `go test` の 3 パッケージ、`check-sync.sh`、`check-skill-sync.sh` が通る。false の動きのテストが残る | 一部(verify の分は満たす) | `./scripts/check-sync.sh` は単独で rc=0(DRIFTED 0)、`./scripts/check-skill-sync.sh` は rc=0(14 skill)。`go test` は `/test`。false の動きのテストは残っている。codex: `internal/cli/org_test.go:1332`(`writeCodexOrgConfig` の `codex_verified = false`)を `TestOrgSpawn_CLI_EnvelopeRejection_NoModelWarning`(`:1520`)が使う。`internal/org/envelope_test.go:28`、`internal/org/spawn_test.go:3921`、`internal/org/permissions_test.go:80`、`:92`。auto_merge: `internal/cli/org_director_test.go:49`、`:58`(off の行の定数)、`TestOrgDirector_StatusShowsAutoMerge`(`:580`)、`internal/cli/status_test.go:862`、`:873`。F2 で、読めない file が書く `auto_merge = false` を off と出す確認(`status_test.go:1217`)と、読めない file を既定にしない確認(`org_test.go:1133`)が加わった |
| AC7: テンプレートの 2 行を false に書き換えた `ralph.toml` が 2 つとも false で読める。戻し方の手順がこの書き換えを使う | 満たす(静的に確認。実行は `/test`) | `TestLoad_TemplateWithBothKeysFalse`(`internal/config/config_test.go:160`)。置き換える 2 行はテンプレートにそれぞれ 1 回だけある(cycle 1 で python3 -I で数え、テンプレートはその後変わっていない)。計画の戻し方(`:118`)の (1) がこの書き換え |
| Deviation M1: `ralph status` は `ralph.toml` が読めないとき auto_merge を false と出す | 満たす(静的に確認) | `internal/cli/status.go:266` と `:299` が 2 つの fallback で `AutoMerge = false` を入れ、warning(`:267-270`、`:302-304`)が理由を言う。テストは `TestStatusCmd_DirectorEnabledFromTheMainWorktree`(`status_test.go:1146`)と `TestStatusCmd_DirectorFallsBackToTheDefaultWhenRalphTomlDoesNotLoad`(`:1193`)。F2 のあとは、権限のないディレクトリへの symlink の `./ralph.toml` もこの fallback に入る(下の F2) |

issue #274 の受け入れ条件との対応: (1) キーがないときの既定が true(`internal/config` のテスト)→ AC1。(2) テンプレートの値とコメント → AC2。(3) skill と recipe に「既定 false」が残らない → AC3(grep の当たりのうち 2 つの設定を言うものは 0 件)。(4) release のノートに既存の project への影響と直し方 → AC4、PR の本文で確かめる。

### F2(aea6b494)の差分

cross-review cycle 1 の ACTION_REQUIRED 1 件(`docs/reports/cross-review-triage-codex-verified-auto-merge-on.md`)への修正。triage が求めた 2 点(`fs.ErrNotExist` 以外のエラーを返す、回帰テストを足す)はどちらも入っている。

- `resolveOrgConfig`(`internal/cli/org.go:613-623`)は、`--config` がなければ `"ralph.toml"` をそのまま `config.Load` に渡す。`os.Stat` の分岐と `os` の import は消えた。`config.Load`(`internal/config/config.go:640-649`)は `os.ReadFile` の `fs.ErrNotExist` だけを既定にし、ほかのエラーは返す。`resolveOrgConfig` はそのエラーを包まずに返すので、テストの `errors.Is(err, fs.ErrPermission)` が成り立つ。`go vet ./...` は通った(`os` を消したあともコンパイルできる)。
- 呼び出し元は 4 つ。`newOrgRuntimeAt`(`org.go:347`、「org: load config: %w」で返す)、`newOrgDirectorRuntime`(`org_director.go:396`、attach と status)、`org_director_headless.go:313`、`statusOrgConfig`(`status.go:263`、warning を出して M1 の fallback)。どれも、エラーを返すか、`ralph status` では auto_merge false の fallback に入る。エラーを既定に変える経路は残っていない。
- `ralph.toml` を `os.Stat` してから既定を返す cli のコードは、ほかにない(`git grep` の結果は evidence の log)。`config.Load` の呼び出し元は `internal/cli/doctor.go:93`、`org.go:331`(`loadMainWorktreeOrgConfig`)、`org.go:618` の 3 つ。
- 変わる場合は、`os.Stat("ralph.toml")` が `fs.ErrNotExist` 以外で失敗する場合だけ(権限のないディレクトリへの symlink、輪になった symlink など)。ない file、行き先のない symlink は前と同じく既定、0o000 の file とディレクトリは前と同じくエラー。self-review run 3 の表と同じ読みになった。
- テスト: `TestResolveOrgConfig_UnreadableRalphTomlIsAnErrorNotTheDefaults`(`internal/cli/org_test.go:1133`)は、ない → 既定でエラーなし、読める false → false のまま、権限のないディレクトリへの symlink と 0o000 の file → `fs.ErrPermission`、を見る。補助の `writeRalphTomlBehindLockedDir`(`:1108`)は `t.TempDir()` のあとに chmod の戻しを `t.Cleanup` に登録するので、後入れ先出しで戻しが先に走る。`TestStatusCmd_DirectorFallsBackToTheDefaultWhenRalphTomlDoesNotLoad`(`internal/cli/status_test.go:1193`)は「parse できない」と「読めない」(`:1217`、中身は `auto_merge = false`)の 2 軸で、off の行、`--json` の `auto_merge` false、warning 1 行を固定する。root では権限の軸だけを skip する。変異の実行(古い `os.Stat` の分岐に戻すと symlink のケースと status の 2 つの subtest が落ちる)は計画の Progress にある implementer の記録で、この report では確かめていない。

## Static analysis

| Command | Result | Notes |
| --- | --- | --- |
| `./scripts/run-static-verify.sh` | pass(rc=0) | 2026-10-28T17:30:29Z、HEAD aa4ec8d3。shellcheck、hook の `sh -n` 20 本、guard の awk 2 本、`settings.json` の jq 2 本、Codex の hook 系 3 本、`check-sync.sh`、`check-pipeline-sync.sh`、`check-skill-sync.sh`(14 skill)、`check-template-purity.sh`、tech-debt の plan 参照がすべて OK。golang: `gofmt: ok`、golangci-lint `0 issues.`、go vet と staticcheck は成功時に何も出さない(Skipping の行はなく、どちらも走った)。branch の secret scan は `09dc0260..aa4ec8d3` で clean |
| `./scripts/check-sync.sh`(単独) | pass(rc=0) | IDENTICAL 172、DRIFTED 0、ROOT_ONLY 0、TEMPLATE_ONLY 11、KNOWN_DIFF 5 |
| `./scripts/check-skill-sync.sh`(単独) | pass(rc=0) | 14 skill が一致 |
| `./scripts/check-template-purity.sh`(単独) | pass(rc=0) | テンプレートに meta-repo 固有の参照なし |
| `git diff --check 09dc0260...HEAD` | pass(rc=0) | 出力なし |
| merge aa4ec8d3 の重なり | 問題なし | main(`ae64da70..09dc0260`)とこの branch の両方が変えた file は `README.md` だけ。branch 側の差は今も `:124` の 1 行(「(default true)」)で、merge で変わっていない |

## Documentation drift

| Doc / contract | In sync? | Notes |
| --- | --- | --- |
| `docs/tech-debt/README.md:136` | yes | cycle 1 の D-1、D-2 は e3363a99 で勧めた文のとおりに直った。引用は「only while [org.permissions].codex_verified is true, the default … a guarded seat silently runs with the alias's sandbox」、Impact は「whose `[org.permissions].codex_verified` is `true` (the default since #274, whether written or left out) and who assigned …」 |
| `internal/cli/org.go:218-224`(`withMainWorktreeDirectorEnabled` の doc comment、D-6) | yes | 「`ralph status` warns about before it shows the built-in Enabled with AutoMerge false」。`status.go:299` と合う |
| `internal/cli/org.go:604-612`(`resolveOrgConfig` の doc comment) | yes | 「ない」は `fs.ErrNotExist` だけ、読めない file はエラー、と書く。行き先のない symlink が既定になることは書いていない(self-review run 3 の任意の提案) |
| `--config` の help(`internal/cli/org.go:58`) | yes | 「default: ./ralph.toml if present, else built-in defaults」。ない場合だけ既定と読め、読めない file を既定にするとは書いていない |
| `ralph status` の `statusOrgConfig` の doc comment(`internal/cli/status.go:249-261`) | yes | 「./ralph.toml when present … else the built-in defaults」「a ralph.toml that does not load falls back … with one warning」。F2 で、読めない file も「does not load」の側に入る |
| `ralph org escalate` の help(`internal/cli/org_inbox.go:62-76`)、`ralph org merge` の help(`internal/cli/org_merge.go:157-163`)、`org_director_wake.go:50-56` | yes | どれも「ないときは既定、読めない(does not load)ときはエラー」の形。F2 で `./ralph.toml` の側もこの形にそろった |
| `/org` skill(`.claude/skills/org/SKILL.md:218`、`:405-413`)、`/director` skill(`.claude/skills/director/SKILL.md:252-257`) | yes | 「`ralph.toml` を読めないときは終了コード 1」「読めない `ralph.toml` はエラーになる」。F2 の前は、権限のないディレクトリへの symlink の `./ralph.toml` がこの文に反して既定になっていた |
| recipe(`docs/recipes/`)、`README.md`、spec の 2 本 | yes | `ralph.toml` が読めないときに既定を使う、と書いた文はない(`git grep` で「if present」「built-in default」「unreadable」「cannot be read」「読めない」などを確かめた) |
| `ralph status --help`(`internal/cli/status.go:66-71`) | yes(INFO D-3) | 下の D-3 |
| `internal/org/permissions.go:147`、`:152`(実行時のエラー文) | yes(INFO D-4) | 下の D-4 |
| 計画の AC と Progress のチェックボックス | yes | cycle 1 の D-5 は解消した。AC1〜AC3、AC5〜AC7 が `[x]`、AC4 は PR の前なので `[ ]` のまま |
| sync-docs の report の「Release notes」と「`/pr` に渡すこと」(`docs/reports/sync-docs-2026-10-28-codex-verified-auto-merge-on.md:53-95`) | no(LOW D-9) | 下の D-9 |

LOW の細目(記録だけで、動作には関係しない。cycle 2 の `/sync-docs` で直す):

- D-9: sync-docs の report は cycle 1(e3363a99)に書かれ、F2 と a91f6694 より前の内容のまま。AC4 の PR の本文はここから作るので、次の 3 点を直す。(1) 「Release notes」の枠の「そのほか」に、F2 の動きの変化を 1 項足す。勧める文: 「`--config` なしの `ralph org` の動詞と `ralph status` は、`./ralph.toml` があるのに読めない(権限のないディレクトリへの symlink、輪になった symlink など)とき、既定で動かずにエラーにする(`ralph status` は warning を出し、auto_merge を false と出す)。前の版は既定で動いていた。`ralph.toml` がないときは今までどおり既定。」(2) 枠の下の changelog の段落は「載るのは次の 2 件」と書くが、`aea6b494 fix: return an error for an unreadable ralph.toml instead of the defaults` も `fix:` なので載る(`.goreleaser.yml:35-39`。`git log --no-merges 09dc0260..HEAD` で `docs:`・`test:`・`chore:` でないものは cb8d258e、958d9bff、aea6b494 の 3 件)。「ほかの 16 件」の数も合わなくなった。(3) 「`/pr` に渡すこと」は D-6 を「PR の Notes に書く未修正の INFO」に挙げるが、D-6 は a91f6694 で直っている。

INFO の細目(どれもマージを止めない):

- D-3(cycle 1 から): `ralph status --help`(`internal/cli/status.go:66-71`)は auto_merge を「read by the same rule as enabled」と書き、読めないときに false と出すことは書いていない。warning の文が理由を言う。sync-docs の report は PR の Notes に回すと決めた。
- D-4(cycle 1 から): `internal/org/permissions.go:147`、`:152` のエラー文の「until then」「first」は opt-in の頃の言い方で、偽ではない。PR の Notes に書く予定(Progress の S3)。
- D-7(新規): `--config` で名指した file がないときも、`config.Load` は既定を返す(`internal/config/config.go:640-645`。`TestLoad_MissingFile`(`internal/config/config_test.go:27`)が `Load("/nonexistent/ralph.toml")` のエラーなしを固定している)。この PR の前からの契約で、F2 の対象(`./ralph.toml` が読めない場合)とは別の場合。ただ、既定が true になったので、`--config` の path の打ち間違いは、`ralph.toml` のない project と同じく codex_verified true と auto_merge true で動く。露出は計画の R1(`ralph.toml` のない project)と同じなので、この PR では直さなくてよい。残すなら PR の Notes に 1 文書く。
- D-8(新規): `ralph doctor` は `ralph.toml` が読めないとき「ralph.toml: warn, parse error: … — using defaults」を出し(`internal/cli/doctor.go:93-101`)、ほかの検査を既定で判定する。既定が true になったので、false と書いた file が読めない project では「Codex sandbox (agmsg writable root)」が要らない warn を出しうる。doctor は判定するだけで何も動かさず、外れる向きは warn が増える側なので、直さなくてよい。読めない file を「parse error」と呼ぶのはこの PR の前からある。

## Observational checks

- `editTemplateKey` の写し(python3 -I と tomllib)を HEAD のテンプレートに当て直し、19/19 で、表の中に 1 回だけあり、書き換えた写しと消した写しの差はそのキーだけだった。
- F2 の前後の比較は、self-review run 3 の端の場合の表(ない、読める、0o000、権限のないディレクトリへの symlink、輪になった symlink、ディレクトリ、行き先のない symlink)を、`config.Load`(`internal/config/config.go:640-649`)と、消えた `os.Stat` の分岐(aea6b494 の差分)を読んで確かめた。変わるのは `os.Stat` が `fs.ErrNotExist` 以外で失敗する 2 つ(権限のないディレクトリへの symlink、輪になった symlink)だけで、どちらも既定からエラーに変わる。
- cycle 1 の広げた確認(`codex_verified` か `auto_merge` が出る行のうち、前後に「default」「既定」「opt-in」などがあるもの)は、対象の file が cycle 1 のあとで変わっていないので打ち直していない。変わったのは `docs/tech-debt/README.md:136`(上の表)と merge で入った #282 の file で、#282 の file には 2 つの設定が出ない。

## Coverage gaps

- `go test ./internal/config/ ./internal/cli/ ./internal/org/` は走らせていない。F2 のテストと、F2 のあとの `internal/cli` 全体は cycle 2 の `/test` が走らせる。
- F2 の変異(古い `os.Stat` の分岐に戻すと落ちること)は implementer の記録だけで、この report では確かめていない。
- AC4(PR の本文の release のノート)は PR の作成のあとでしか確かめられない。
- この repo にはルートの `ralph.toml` がないので、この PR のあと、この repo の `ralph org` は auto_merge true、codex の座席は autonomous で動く。計画の Assumptions が受け入れた動きで、実機では試していない。
- root で走らせたときの F2 のテストの skip は、コードを読んだだけで、root では走らせていない。

## Verdict

- Verdict: pass
- Verified: AC1・AC2・AC7 の値と、それを確かめるテストの中身(`go vet` のコンパイルを含む)、AC3(HEAD で打ち直した grep の当たり 4 行はどれも別の設定、merge で入った file にも当たりなし)、AC5 のテストの構造と `editTemplateKey` の写しでの空振りのなさ、AC6 のうち `check-sync.sh` と `check-skill-sync.sh` と false の動きのテストが残っていること、Deviation M1、F2 の差分(呼び出し元 4 つ、ほかに `os.Stat` の分岐がないこと、テストの組み立て)、cycle 1 の D-1・D-2・D-5 の解消、F2 に関わる help・skill・recipe に食い違いがないこと、静的検査の全体(`run-static-verify.sh` rc=0、単独の 3 本も rc=0)、issue #274 の受け入れ条件 1〜3
- Partially verified: AC6(テストの実行は `/test`)、F2(変異の実行は implementer の記録だけ)、文書のずれ(LOW 1 件 D-9、sync-docs の report の release ノートが F2 の前のまま。cycle 2 の `/sync-docs` で直す)
- Not verified: AC4 と issue #274 の受け入れ条件 4(PR の本文がまだない)、`go test` の実行結果、root での skip
