# Verify report: script-local-config

- Date: 2026-11-01(JST。UTC では 2026-10-31)
- Plan: `docs/plans/active/2026-10-30-script-local-config.md`(issue #276 の B2、AC1〜AC7、D1〜D7、Progress の S1〜S7 と self-review の直し 00c494cc)
- Verifier: verifier サブエージェント(`/verify`、パイプラインの cycle 1)
- Scope: `git diff origin/main...HEAD`(merge-base = origin/main 1cc84ffa、HEAD fc472d28、50 ファイル、+3715/−941)。2cb62e4b は origin/main(#292)の取り込みで、その分は対象にしていない。spec の準拠、静的解析、文書のずれだけを見た。Go のテストと `./scripts/run-test.sh` は /test の担当なので打っていない。probe は HEAD から build した binary(`<scratchpad>/v276b2/bin/ralph`)と、その binary の `ralph init --yes --pack golang` で作った一時 repo(bare の origin、`main` と `feat` の 2 コミット)で打った。一時 repo は scratchpad の中にあり、task worktree と main のチェックアウトには何も書いていない。
- Evidence: `docs/evidence/verify-2026-10-30-script-local-config.log`(run-static-verify の出力、probe のスクリプトと出力、helper の表、D7 の probe、`/cross-review` の手順 4 のコマンド)。`.gitignore:61` の `docs/evidence/*.log` に当たるのでコミットしない(`git check-ignore -v` で確かめた)。run-static-verify 自身のログは `docs/evidence/verify-2026-10-31-160218.log`(同じく対象外)。
- load average(`uptime`): static の直前 12.54、`go build` の直前 27.01、probe の直前 13.96、D7 の probe の直前 7.58。30 を超えたときには打っていない。

## Spec compliance

| Acceptance criterion | Status | Evidence |
| --- | --- | --- |
| AC1 5 つのキーの既定値、`ralph.local.toml` の値が効く、`ralph.toml` に書くと exit 3(大文字小文字を変えても)、D3 に合わない値は exit 3 でキーと値を名指す | 満たす | `internal/config/config.go:729`・`:737-744`(`Default()`)、`:826`(`baseBranchPattern`)、`:1129-1145`(`validateScriptValues`)、`:1148`(`isScriptWord`)、`internal/config/local.go:123`(`LocalOnlyKeys` は 6 キー)。probe の 1 節: ファイルがないとき 4 つは空、skip は `false`、どれも rc 0。5 節: `ralph.local.toml` の値(`none`、`release`、`HEAD`、`true`)がそのまま出る。6 節: 追跡される `ralph.toml` に 5 つのキーを 1 つずつ書くと、どれも exit 3 で「ralph.toml: [verify].base can be set only in ralph.local.toml, …」の形でパスとキーと `ralph.local.toml` を名指す。`[VERIFY] BASE`、`[Plan_Visual] Opener`、`[standard] Base_Branch` も同じ。7 節: 断った値は base_branch の `-x`・`a..b`・`a b`、browser の `~/bin/chrome`・`bin/chrome`、opener の `-x`・`open -a`、verify.base の `-x`・`a b` で、どれも exit 3 でキーと値を名指す。通った値は `feat/x-1.2_y`、`none`、`/Applications/Google Chrome.app/Contents/MacOS/Google Chrome`、`xdg-open`、`HEAD~1`、`HEAD^`、`origin/main`。skip の `1` と `"true"` は go-toml の decode が exit 3 で断る(I3)。テストは `internal/config/local_script_keys_test.go:31`・`:48`・`:87`・`:147`、`internal/cli/config_test.go:439`・`:474`・`:503`(存在を確かめた。実行は /test) |
| AC2 helper の表に 5 つのキーがあり `Default()` と一致する。既定値が空のキーでは exit 0 の空を黙って返す | 満たす | `scripts/ralph-config.sh:72-81`(表の 10 行)、`:115`(exit 0 の空を値にするのは既定値が空のキーだけ)。照合は `internal/config/defaults_sync_settings_test.go:67-78`(`shellSettingWant`)と `internal/config/defaults_sync_test.go:259-263`(コードを読んで確かめた。実行は /test)。helper の probe(sh、bash、dash、`zsh -f` × 2 つの関数 × `verify.base`・`standard.base_branch`・`verify.skip_branch_secret_scan`、HEAD の binary、ファイルなし): 空・空・`false` を rc 0 で出し、stderr は 24 通りとも空 |
| AC3 4 つのスクリプトは helper で読み、5 つの環境変数は効かない。`none` と空の意味は今と同じ。偽の ralph のテストで確かめる | 満たす(テストの結果は /test) | 読み手: `scripts/plan-visual.sh:157-163`(`plan_visual_setting`)・`:189`・`:233`、`scripts/xreview-helpers.sh:57-69`、`scripts/detect-changed-languages.sh:183-192`、`scripts/run-verify.sh:257-264`。写しは 6 本とも `cmp` で一致し、scaffold した repo のものとも一致した。probe の 3 節: `[verify].base = "HEAD"` で `no_changes`(対照は `golang`)、`[standard].base_branch = "release"` で `detect_base_branch` が `release`、scan が「against origin/release」、`[plan_visual].opener = "none"` で「([plan_visual].opener is none)」の注記、空なら OS 既定の opener(stub の `open`)。8 節: `skip_branch_secret_scan = true` で「==> Skipping branch secret scan ([verify].skip_branch_secret_scan = true)」。環境変数: 9 節で `RALPH_VERIFY_BASE=HEAD` は `golang` のまま、`RALPH_XREVIEW_BASE=develop` は `main` のまま、`RALPH_PLAN_VISUAL_OPENER=none` でも opener が動く。8 節の対照で `RALPH_VERIFY_SKIP_BRANCH_SECRET_SCAN=1` でも scan が走る。`RALPH_PLAN_VISUAL_BROWSER` は `shot` がブラウザを要るので probe していない(計画の S7 の否定の grep が 0 件、テストは `tests/test-plan-visual.sh:443-452`)。偽の ralph は `tests/lib/fake-ralph.sh` |
| AC3b 設定の誤りでの扱い(D7) | 満たす | probe の 2 節(lead の指定どおり、正しい `[verify].base = "HEAD~1"` と相対パスの `[plan_visual].browser = "bin/chrome"`): `ralph config get verify.base` は exit 3。`detect-changed-languages.sh` は `scope=full`、`reason=config_error:verify.base` を出し、stderr に ralph-config.sh の行と「[verify].base could not be read … taking the full scope, not the automatic base」の 2 行。`secret-scan-branch.sh --strict` は exit 3、既定モードは同じ「cannot scan」を出して exit 0。`detect_base_branch` は sh、bash、dash、`zsh -f` で stdout が空、rc 1。`plan-visual.sh open` は path を出し、1 行知らせて rc 0。D7 の場面(d7.sh、`main` の上で HEAD = origin/main): ファイルがなければ「Language scope: changed (no_changes)」で言語の検査は走らない。同じ誤りを書くと「full fallback (config_error:verify.base)」で golang の検査が走る。`/cross-review` の手順 4 のコマンドを SKILL.md から取り出して打つと、rc 1 で `base=` の行は出ない(対照は `base=main rc=1`)。helper の probe: `ralph_setting_or_fail` は誤りで stdout が空の rc 1、ralph がない(127)ときと 6.0.0(unknown command)のときは既定値を黙って rc 0 |
| AC4 ralph がないときの既定の振る舞い、CI の `GITHUB_BASE_REF`、gate で個人用のファイルが効かない | 満たす(gate はコードとテストの存在と `RALPH_LOCAL_CONFIG=off` の probe まで) | probe の 4 節: 5 つすべてを書いた `ralph.local.toml` があっても、ralph がない PATH と PATH の 6.0.0 で、`golang`(origin/main との差分)、`main`、「scanned 789a482..9e91cff against origin/main」、OS 既定の opener のまま。3c' 節: 誤った `base_branch = "-bad"` と `GITHUB_BASE_REF=main` で scan は origin/main を相手に rc 0 で、helper の stderr の行は出ない(helper を打っていない)。gate: `internal/org/gate_run.go:53` と `:75-88`(`gateScriptEnv` は `RALPH_LOCAL_CONFIG=off` と `RALPH_VERIFY_SCOPE=full` を足す)を `gate.go:177` と `regate.go:889` が使う。probe の 5 節: `RALPH_LOCAL_CONFIG=off` で 5 つとも既定値、`detect-changed-languages.sh` は `golang`、`detect_base_branch` は `main`。テストは `internal/org/gate_runverify_test.go:132` |
| AC5 `ralph doctor` が 5 つの変数(空でない値)に代わりのキーを名指す warn を出し、未設定と空では何も出さない | 満たす | `internal/cli/doctor.go:909-914`、`:942-948`(`retiredScriptEnv`)。probe の 10 節(4 回の出力の diff): 未設定と空は同じ。5 つを設定すると 5 行の warn が増え、それぞれ `[plan_visual].browser`・`opener`、`[standard].base_branch`、`[verify].base`・`skip_branch_secret_scan` の `ralph.local.toml` を名指す。`=0` は 1 行で、文は self-review の L6 のとおり「write the value … as true only if it is 1; … so write nothing」。rc は 4 回とも 1 だが、PATH を絞ったので claude と codex がない ✗ の 2 行によるもので、diff は warn の行だけ |
| AC6 5 つの変数に触れるのは「もう読まない」の説明だけ。`quality-gates.md`、`/cross-review` の skill、README の節が新しいキーを案内する | 満たす(/sync-docs に回すものは下の表) | `docs/quality/quality-gates.md:43-54`(と写し)、`.claude/skills/cross-review/SKILL.md:67-68`・`:72`(4 つの写しは check-skill-sync と check-sync で一致)、`README.md:135`・`:152`。変数名の `git grep`(記録を除く)の残りは、`internal/cli/doctor.go` の表、テスト、README:152 と `scripts/ralph-config.sh:15-21`(と写し)の「もう読まない」の説明、計画、tech-debt の RESOLVED の記録(`:56`・`:225`)、spec の決定の記録 1 行(I1)だけ |
| AC7 build、gofmt、vet、golangci-lint、shellcheck、check-sync、check-skill-sync、run-verify | 一部(静的の部分は満たす) | `./scripts/run-static-verify.sh` rc 0、`go build ./...` rc 0(下の表)。`./scripts/run-verify.sh` の既定モード(テストを含む)は /test の範囲で、ここでは打っていない。計画の AC7 のチェックは付けていない |

計画の AC のチェックは AC1〜AC6(AC3b を含む)を `[x]` にした。`./scripts/plan-visual.sh digest` は直す前も後も `cdd1e5ba1234` で、`- Approved:` の digest と同じ。

## Static analysis

| Command | Result | Notes |
| --- | --- | --- |
| `./scripts/run-static-verify.sh` | pass(rc 0) | changed の既定でも「Language scope: full fallback (shared:scripts/detect-changed-languages.sh)」で、Go は全体で走った。gofmt `ok`、golangci-lint `0 issues.`、go vet と staticcheck は無出力(Skipping の行なし)。local verifier(shellcheck `--severity=warning`、対象に足した `tests/lib/*.sh` を含む)、`sh -n`、Codex の hook の検査、check-sync `PASS`(IDENTICAL 172、DRIFTED 0、ROOT_ONLY 0、KNOWN_DIFF 6)、check-pipeline-sync、check-skill-sync(14 本)、check-template-purity、tech-debt の計画参照はすべて OK。branch の secret scan は `1cc84ffa..fc472d28` で clean(PATH の ralph は 6.0.0 なので base は自動の origin/main)。打ち直しはしていない |
| `go build ./...`、`go build -o <scratchpad>/v276b2/bin/ralph ./cmd/ralph` | pass(rc 0) | go1.27.3。前後で `git status --porcelain` は空 |
| `shellcheck <file>`(既定の severity、変わった shell 22 本) | note だけ | SC1091(source 先を追わない)が、ほかのファイルを source する 16 本に 1 件ずつ。`shellcheck -x -P SCRIPTDIR` で 5 つの読み手を source 先ごと見ると、残るのは `scripts/secret-scan-branch.sh:97` の SC2329 だけ。SC2329 は `secret-scan-branch.sh` 1 件、`verify.local.sh` 6 件で、どちらも origin/main と同じ数 |
| `./scripts/plan-visual.sh digest <plan>` | `cdd1e5ba1234` | 承認時と同じ |
| `git diff --check origin/main...HEAD`、変えたファイルの U+FFFD | どちらも空 | |
| 計画の S7 の否定の grep(`\$\{?RALPH_(…5 つ…)` を `scripts`、`templates/base/scripts`、`.claude/skills`、`.agents/skills` とその写しで) | 0 件 | `xreview-helpers.sh` の「Pure function」も 0 件 |

## Documentation drift

| Doc / contract | In sync? | Notes |
| --- | --- | --- |
| `AGENTS.md:89`(repo map の `internal/config/`) | no(V1) | `LocalOnlyKeys` を「keys only that file may set, `[org].state_dir` today」と書く。今は 6 キー(`internal/config/local.go:123-131`)。lead が先に挙げたもの |
| `README.md:135`(`ralph config get` の行)、`:152`(`ralph.local.toml` の節) | yes | local-only のキーの説明、値の形(D3)、D7 の 2 つの base、ralph がないときと gate と regate、撤去した 5 つと doctor の warn が、コードと probe の結果に合う。`:135` の末尾は 2 つの関数の使い分けを `config get --help` と同じ内容で書く |
| `docs/quality/quality-gates.md`(KNOWN_DIFF、root と写し) | yes | `[verify].base` と、読み出しの失敗で full scope になる 1 文 |
| `/cross-review` の skill(4 つの写し) | yes | 解決順の (1) と `\|\| exit 1`、止まって stderr を見せる手順。手順 4 のコマンドは probe で rc 1、`base=` なし |
| `.claude/rules/ralph/model-routing.md`(root と写し) | yes | helper の説明が 10 キーと 2 つの関数になり、`ralph_setting_or_fail` の振る舞いが `scripts/ralph-config.sh:28-57` に合う |
| `docs/architecture/repo-map.md:62` | yes | `ralph-config.sh` の 2 つの関数と 10 キー |
| `templates/base/ralph.toml:51-60`・`:84-113` | yes | 5 つは注釈で、値の形、local-only、gate の既定値の説明が Load の検証と合う |
| `ralph config get --help`(`internal/cli/config.go:84-92`) | yes | 4 つのスクリプトと 2 つの関数の割り当てが、読み手のコードと合う |
| `scripts/secret-scan-branch.sh:51-56` の exit code の説明 | yes | `--strict` の 3 に「[standard].base_branch that ralph config get cannot read」が入った |
| `docs/specs/2026-10-07-org-multi-org-director.md:123` | 記録(I1) | 7b の決定の記録が、gate が `RALPH_VERIFY_` を落とす理由に `RALPH_VERIFY_BASE=HEAD` を挙げる |
| `docs/tech-debt/README.md` の 800 行の行(self-review の L4) | /sync-docs に回す | まだない。今の行数は `tests/test-detect-changed-languages.sh` 669 → 848、`tests/test-plan-visual.sh` 857 → 1012、`tests/test-secret-scan-branch.sh` 2116 → 2217、`internal/config/config.go` 1065 → 1180、`internal/cli/doctor.go` 1483 → 1514(origin/main → HEAD)。self-review の数(998、1510)は直しの前のもの |

## Findings

| ID | Severity | Area | Finding | Recommendation |
| --- | --- | --- | --- | --- |
| V1 | LOW | doc drift | `AGENTS.md:89` が `LocalOnlyKeys` を「`[org].state_dir` today」としている。この PR で 5 キーが足され、6 キーになった | /sync-docs で「`[org].state_dir` and the five keys the scripts read」の形に直す。merge は止めない |
| I1 | INFO | doc の記録 | `docs/specs/2026-10-07-org-multi-org-director.md:123` は 7b の決定の記録で、gate が `RALPH_VERIFY_` で始まる変数を落とす例に `RALPH_VERIFY_BASE=HEAD` を挙げる。スクリプトはもうこの変数を読まないが、gate は今も同じ接頭辞を落とし、`internal/org/gate_run.go:56-63` の注釈は例を `RALPH_VERIFY_SCOPE=changed` に替えた | 任意。/sync-docs で「(B2 からは読まれない)」の注記を足すか、記録として残す |
| I2 | INFO | 振る舞い | 設定の誤りがあると、`run-verify.sh`(既定モードの scan)は branch の scan を「cannot scan」で飛ばし、「All verifiers passed」で終わる(d7.sh)。self-review の Notes と同じ観察で、AC3b の exit 3 は `--strict` の話。`/pr` と leader の push の守りは `--strict` で止まる | なし |
| I3 | INFO | エラーの文 | `skip_branch_secret_scan = 1` と `= "true"` は go-toml の decode が断り、文は「cannot decode TOML integer into struct field config.VerifyConfig.SkipBranchSecretScan of type bool」で、ドット区切りのキー名と値を名指さない。D3 の検証の対象外(`config.go:1127` の注釈どおり)で、パスは名指す。doctor の warn は `1` の人に `true` と書くよう案内する | なし |

## Observational checks

- lead が挙げた probe はすべて打った。正しい `[verify].base` と相対パスの `[plan_visual].browser` の組で 4 つの読み手(probe の 2 節)、正しい値だけで 4 つの値が効くこと(3 節、8 節)、`ralph.toml` に 5 つのキー(6 節)、`RALPH_LOCAL_CONFIG=off`(2 節の末尾、5 節)、`ralph doctor`(10 節)。
- 計画の Test plan が /verify に回した「本物の binary で同じ形を確かめる」は、probe の 2 節と d7.sh で済んだ。
- `ralph config get` のエラーの文は、`ralph.local.toml` の値の誤りでも「config get: ralph.toml: cannot be read or is invalid: … (with <path>/ralph.local.toml applied)」で始まる。A1 からの形で、この PR は変えていない。
- probe で打った ralph は、HEAD の binary(`ralph dev`)と PATH の 6.0.0 だけ。`ralph doctor` は `--state-dir <scratchpad>/v276b2/state` を付け、merge guarantee のキャッシュはそこに書かれた(origin がローカルの bare なので gh は呼ばれず「unknown」)。doctor は `~/.zshrc` と `~/.codex/models_cache.json` を読んだが、書いていない。
- `plan-visual.sh open` は stub の `open` を PATH の前に置いて打ち、本物のブラウザは開いていない。herdr、agmsg、codex、claude、gh は打っていない。
- 一時 repo と probe のスクリプトは scratchpad の `v276b2/` にある。lead の指示どおり、report を書いたあとで消す。

## Coverage gaps

- Go のテストと `tests/test-*.sh` の結果(/test の担当)。AC2 の照合、AC4 の gate のテスト、AC3 の偽の ralph のテストは、存在とコードを読むところまで。
- `RALPH_PLAN_VISUAL_BROWSER` を設定して `plan-visual.sh shot` を打つ probe(ブラウザが要る)。grep とテストの行で代えた。
- `tests/test-plan-visual.sh` のブラウザ自動検出の 2 件は、この Mac では SKIP になる(計画の Progress の S3)。PR の CI のログで、Linux で SKIP でなく通ったかを見る。
- `ralph org gate` と `ralph org regate` を本物の feature org で打つこと。gate の環境はコード、テストの存在、`RALPH_LOCAL_CONFIG=off` の probe で確かめた。
- Codex の driver で `/cross-review` を動かす道(`.agents/skills/` の写しは文字列として一致することまで)。

## Verdict

pass

- Verified: AC1、AC2、AC3、AC3b、AC4、AC5、AC6 は、コードの行と、HEAD の binary と scaffold した一時 repo の probe、helper の 4 つの shell の表、SKILL.md から取り出した手順 4 のコマンド、grep で確かめた。静的解析(gofmt、go vet、golangci-lint、staticcheck、shellcheck、check-sync、check-pipeline-sync、check-skill-sync、check-template-purity、secret scan)はすべて通り、計画の digest は承認時と同じ。
- Partially verified: AC7 は静的の部分だけ。AC4 の gate は probe とコードとテストの存在まで。
- Not verified: Go と shell のテストの結果(/test)、`shot` でのブラウザ、本物の feature org の gate、Codex の driver での skill。
- 所見: LOW 1 件(V1、AGENTS.md の `LocalOnlyKeys`)、INFO 3 件。どれも merge を止めない。
- /sync-docs に回すもの: V1、self-review の L4 の tech-debt の行(上の表の今の行数で)。任意: I1 の注記。
- Insight event: `docs/insights/events/2026-10-31-script-local-config.jsonl` に verify の行を 1 行足した(`--verdict pass --cycle auto --low 1`、`"cycle":1`)。ファイル名は UTC の日付で、self-review の行と同じファイル。
