# Verify report: drop-gpt-5-5-default-pool

- Date: 2026-10-02
- Plan: docs/plans/active/2026-10-02-drop-gpt-5-5-default-pool.md
- Verifier: verifier subagent (Claude)、cycle 1
- Branch: chore/drop-gpt-5-5-default-pool、HEAD 92ec1659(push 済み)、base main 7dd6911c
- Scope: 仕様適合(AC-1〜AC-7 を該当行と実行結果で確認)、静的解析、文書のずれ。振る舞いのテスト suite は /test の担当なので verdict には使っていない。AC に直結するテスト 5 本だけは単発の probe として実行し、下の表に分けて書いた
- Evidence: `docs/evidence/verify-2026-10-02-044720.log`(full)、`docs/evidence/verify-2026-10-02-044740.log`(default)。どちらも `docs/evidence/*.log` が gitignore のため commit には含まれない。scratch の probe 出力は repo の外に置いた

## Deterministic checks run

| Command | Result | Notes |
| --- | --- | --- |
| `RALPH_VERIFY_SCOPE=full ./scripts/run-static-verify.sh` | PASS(rc 0) | shellcheck と `sh -n` の全項目 OK。check-sync は IDENTICAL 159 / DRIFTED 0 / ROOT_ONLY 0 / TEMPLATE_ONLY 11 / KNOWN_DIFF 5。check-pipeline-sync OK。check-skill-sync は 13 skill で lock-step。check-template-purity OK。golang は gofmt ok、golangci-lint 0 issues。branch secret scan は `7dd6911c..92ec1659` で clean |
| `./scripts/run-static-verify.sh`(default スコープ) | PASS(rc 0) | `Requested scope: changed` が `Language scope: full fallback (unclassified:scripts/ralph-config.sh)` になり、golang が選ばれて gofmt ok / 0 issues。`./scripts/detect-changed-languages.sh` の直接実行は `scope=full`、`reason=unclassified:scripts/ralph-config.sh`、`golang_roots=.`。branch の upstream が HEAD と同じ(origin/chore/drop-gpt-5-5-default-pool = 92ec1659)でも差分は見えていて、`no_changes` にはなっていない(PR #195 の merge-base 化が効いている)。golang が選ばれた直接の理由は未分類ファイルによる full fallback で、diff の Go 3 ファイル(config.go、config_test.go、org_test.go)だけで選ばれる経路は、この branch では通っていない |
| `./scripts/check-sync.sh` | PASS(rc 0) | 上と同じ集計、`PASS: all files in sync.` |
| `./scripts/check-skill-sync.sh` | PASS(rc 0) | `13 skill(s) in lock-step` |
| `./scripts/check-template-purity.sh` | PASS(rc 0) | meta-repo 固有の参照なし |
| `cmp scripts/ralph-config.sh templates/base/scripts/ralph-config.sh` | PASS(rc 0) | byte 一致 |
| `go vet ./...` | PASS(rc 0) | |
| `gofmt -l ./internal` | PASS | 出力なし |
| `sh -n scripts/ralph-config.sh` と source 後の `RALPH_ORG_MODEL_POOL` | PASS | run-static-verify は `ralph-config.sh` を `sh -n` していないので、別に実行した。source 後の要素数は 8 |
| 既定 4 面の静的比較(`config.go:141-150`、`templates/base/ralph.toml:29-38`、`scripts/ralph-config.sh:57`、org skill の表) | PASS | 順序と内容が 4 面とも同じ 8 エントリ(`claude:fable,opus,sonnet,haiku` + `codex:gpt-6-astra,gpt-5.6-sol,gpt-5.6-terra,gpt-5.6-luna`)。抽出した列を `cmp` で比べた |
| 単発 probe(verdict ではない): `go test ./internal/config -run 'TestDefaultsLockStep\|TestDefault_Org$\|TestLoad_OmittedModelPoolWithRolesNamingDroppedDefault_ErrorsUntilModelPoolIsExplicit\|TestLoad_DriverPoolOnlyOverride_Codex_KeepsOnlyCodexDefaultEntriesInOrder' -count=1`、`go test ./internal/cli -run TestOrgStatus_ConfigFlagRecoversAfterDefaultPoolDropsRoleModel -count=1` | PASS | 5 本とも pass。CLI のテストは 3 つのサブテストすべて pass。suite 全体は実行していない |

## AC-5: doctor の probe(独立に実施)

- HEAD 92ec1659 から `go build -o <scratch>/ralph ./cmd/ralph`。`env -u ZDOTDIR HOME=<scratch>/home CODEX_HOME=<scratch>/codex-home PATH=/usr/bin:/bin GIT_CONFIG_GLOBAL=/dev/null GIT_CONFIG_NOSYSTEM=1` で実行。利用者の `~/.codex`、HOME の rc、herdr には触れていない
- `<scratch>/codex-home/models_cache.json` は `writeCodexModelsCache`(`internal/cli/doctor_org_test.go:384`)と同じ形で、スラッグは `gpt-6-astra`、`gpt-5.6-sol`、`gpt-5.6-terra`、`gpt-5.6-luna` の 4 つだけ

| scratch project | doctor の「Org codex model slugs」 |
| --- | --- |
| `ralph.toml` なし | `✓ Org codex model slugs: pass — 4 codex model_pool slug(s) present in <scratch>/codex-home/models_cache.json (cache written 2026-10-02T05:16:37Z, 0m ago)` |
| `ralph.toml` あり、`model_pool` キーなし(`[org] max_seats = 5` のみ) | 同じ行(pass、4 slug) |
| `templates/base/ralph.toml` をそのまま置いた project | 同じ行(pass、4 slug)。template の明示プールからも codex が 4 つになる |
| 負の対照: main 7dd6911c のビルド、`ralph.toml` なし、同じ cache | `⚠ Org codex model slugs: warn — 1 codex model_pool slug(s) not found in <scratch>/codex-home/models_cache.json: gpt-5.5` |

HEAD は pass、main は `gpt-5.5` だけを挙げて warn になるので、probe は変更の有無を区別できる。利用者の実際の cache での確認は、plan の取り決めどおり行っていない(合否に使わない補足)。

## AC ごとの verdict

| AC | Verdict | 根拠 |
| --- | --- | --- |
| AC-1 `Default()` が 8 エントリで `gpt-5.5` を含まない | Pass | `internal/config/config.go:141-150` は claude 4 + codex 4。`TestDefault_Org`(`config_test.go:117`)が全 8 エントリを順序つきで比べ、`gpt-5.5` の不在を長さの確認より前に名前つきで確かめる(`:132-139`)。`len != 8` の確認は `:225`。単発 probe で pass。`Default()` が `gpt-5.5` を戻す mutation は self-review addendum が確認済みで、私は再現していない |
| AC-2 3 面 + template が同じ 8 エントリ、`defaults_sync_test.go` green、`ralph-config.sh` が byte 一致 | Pass | 4 面の静的比較が一致、`cmp` rc 0、`TestDefaultsLockStep`(`defaults_sync_test.go:92`)は単発 probe で pass。suite 全体での green は /test で確認する |
| AC-3 org skill の表(4 面)に `gpt-5.5` の行がなく、check-skill-sync と check-sync が green | Pass | `.claude/skills/org/SKILL.md:76-87` の表は 8 行。`.agents/skills/` と `templates/base/` の 4 面から `gpt-5.5` は 0 hit。check-skill-sync と check-sync は rc 0 |
| AC-4 `git grep` に既定のプールとしての記述が残らない | Pass | 下の分類のとおり。253 hit、46 ファイル。出荷する skill と `templates/base/ralph.toml`、`ralph-config.sh` は 0 hit。「9 エントリ」「codex 5 スラッグ」「`!= 9`」の主張は plan の経緯の文(`plan:21`、`:113`)以外にない |
| AC-5 doctor の probe | Pass | 上の節。3 パターンとも pass、4 slug、負の対照で main は warn |
| AC-6 `run-verify.sh` と `go test ./...` が green | 一部確認 | 静的な半分(full スコープの `run-static-verify.sh` が rc 0)は確認した。`go test ./... -count=1` と `run-verify.sh` / `run-test.sh` の振る舞い側は /test の担当で、ここでは実行していない |
| AC-7 org skill に復旧の段落(4 面)、CLI の回帰テストが失敗と復旧を確かめる | Pass | 段落は `.claude/skills/org/SKILL.md:115-131`(4 面が cmp で同一)。テストは `internal/cli/org_test.go:900`(`--state-dir` 経路の 2 つの直し方 + `RALPH_ORG_STATE_DIR` 経路のサブテスト)と `internal/config/config_test.go:850`。いずれも単発 probe で pass。CLI でも同じ結果を出した(下の節) |

## 復旧の段落と spec (e) の主張をコードと git に当てた結果

| 主張 | 確認 | 結果 |
| --- | --- | --- |
| `[org.roles]` のモデルが実効プールにないと `[org.roles].<role> references model "<model>" not present in [org].model_pool` で検証が通らない | `internal/config/config.go:294-300`(`validOrgModels` との照合)。`model_pool` を省略した設定は `Default()` のプールを引き継ぐ(`:241-255` は `driver_pool` だけのときの絞り込み) | 一致 |
| `ralph org` の動詞はどれも設定を読む。省略時は cwd の `./ralph.toml`、`--config` は全動詞に効く | `internal/cli/org.go:43`(persistent flag)、`:114-115`(`newOrgRuntimeAt` → `resolveOrgConfig`)、`:218-231`(`--config` 空なら `ralph.toml` があればそれ、なければ組み込みの既定)。`newOrgRuntime` / `newOrgRuntimeAt` の呼び出しは 10 動詞に 10 か所(`:302`、`:411`、`:484`、`:601`、`:639`、`:676`、`:706`、`:801`、`:842`、`:896`)。`read` は `org.go:183` のコメントのとおり設定を読むが seat 探索には使わない。「読む」の主張は成り立つ | 一致 |
| 壊れた設定では `status` / `stop` / `disband` が止まる | CLI probe。`model_pool` なし + `[org.roles] reviewer = ["gpt-5.5"]` の project で、`org status`、`org disband`、`org stop --seat seat-1 --dry-run` がいずれも rc 1、`org: load config: [org.roles].reviewer references model "gpt-5.5" not present in [org].model_pool` | 一致 |
| 直したコピーを `--config` で渡すと動く | CLI probe。同じ project、同じ `--state-dir` で `org status --config <直したコピー>` が rc 0(manifest が空なので `no seats`)。座席が見えることは単発 probe の CLI テスト(manifest に `seat-1` を入れる)が確かめる | 一致 |
| `spawn --model` / `start --model` は `org: model "<model>" not in [org].model_pool for driver "<driver>"` で拒否される | `internal/org/envelope.go:55-57`。`org.go:310`(spawn)と `:420`(start)はどちらも `rt.Spawn` を呼び、`spawn.go:375`、`:502` で `ValidateSpawnEnvelope`。CLI probe: 既定のプール(`ralph.toml` なし)で `org spawn --driver codex --model gpt-5.5 --dry-run` が rc 1、同じ文面。`start` は CLI では実行していない(コードの経路が同じ) | 一致 |
| 明示した `model_pool` は既定を置き換える | CLI probe。`model_pool = [{claude, opus}]` だけの project で `org spawn --driver claude --model sonnet --dry-run` が `org: model "sonnet" not in [org].model_pool for driver "claude"` で rc 1(`sonnet` は既定のプールにある)。skill が引く `TestLoad_OrgFullRoundTrip` は読んでいない | 一致 |
| state dir は `--state-dir`、`RALPH_ORG_STATE_DIR`、git の toplevel、cwd の順で、設定ファイルの場所では変わらない | `internal/org/statedir.go:47-58`(`ResolveOrgStateDir`)。設定のパスは引数に入らない | 一致 |
| spec (e): 最新のタグは v5.1.0 で、3f9b4a01 を含むタグはない | `git tag --sort=-v:refname` の先頭が v5.1.0、`git tag --contains 3f9b4a01` は空、`git ls-remote --tags origin` の最新も v5.1.0。3f9b4a01(2026-09-16)は HEAD の祖先 | 一致 |
| spec (e): v5.1.0 の既定は claude の 3 つだけ | `git show v5.1.0:internal/config/config.go` の `Default()` は `opus` / `sonnet` / `haiku`。v5.1.0 の `templates/base/ralph.toml` の明示プールも同じ 3 つ。v5.1.0 にも同じ roles の検証(`config.go:235`)がある | 一致 |
| spec (e): #156 の観測(2 回目 2026-09-25 に退役予告 `gpt-5.6-sol` への移行・2026-10-14、3 回目 2026-10-02 に予告が消えて "Legacy coding model.") | `gh issue view 156 --comments` の読み取り。観測 2 の `retiring: gpt-5.5 -> gpt-5.6-sol on 2026-10-14`、観測 3 の `upgrade` が `null` と "Legacy coding model." が記録と合う。外す判断自体はメンテナの指示(plan の Related request)で、issue には 10/2 時点で「今は外しません」とあり、spec (e) の「メンテナの判断で」はその後の指示を指している | 一致 |

## self-review の指摘が HEAD で直っているか

| # | 状態 | 確認 |
| --- | --- | --- |
| M-1 | 直った | skill の段落(`SKILL.md:115-131`)は日付つきの経緯と `gpt-5.5` を含まず、版に依存しない手順。spec (e)(`:23`)は `gpt-5.5` を既定に含むリリースがないことと、影響が main のソースビルドに限られることを書く。上の表で v5.1.0 とタグを自分で確認した |
| L-1 | 直った | `config_test.go:132-139` のループが長さの `t.Fatalf`(`:141`)より前にある |
| L-2 | 直った | `org_test.go` の `state dir from RALPH_ORG_STATE_DIR without --state-dir` サブテストが存在し、単発 probe で pass。git の toplevel の段は `PATH=""` のため試せない旨がテストのコメントにある。私の CLI probe は `--state-dir` 経路で、toplevel の段は未実行 |
| L-3 | 直った | skill から退役の理由が消え、state dir の順序に cwd が入った(`SKILL.md:129-131`) |
| L-4 | 直った | テスト名が `..._ErrorsUntilModelPoolIsExplicit`(`config_test.go:850`)で、後半の確認も表す |
| L-5 | 解消 | M-1 で申し送りが不要になり、plan の Non-goals(`plan:43`)に理由がある |
| A-L1 | 直った | 段落が `spawn` / `start` の拒否に「プールにあるモデルを `--model` に渡す」を足し、明示した `model_pool` が既定を置き換えること、使い続ける既定のエントリも書くことを述べる(`SKILL.md:124-127`)。置き換えは CLI probe で確認した |
| A-L2 | 直った | spec (e) は「この変更の時点で最新のタグは v5.1.0 で、3f9b4a01 を含むタグはなかった」と書き、`--model gpt-5.5` で spawn / start していた人も影響の範囲に入れ、座席の teardown が止まる話と分けた |

self-review の mutation の主張(`Default()` に戻すと落ちる、env の段を無効にすると落ちる、など)は、addendum の記録を前提にしていて、私は mutation を再実行していない。

## `git grep -n 'gpt-5\.5'` の分類(HEAD 92ec1659)

合計 253 hit、46 ファイル。

| 分類 | ファイル数 / hit | 内容 | AC-4 への影響 |
| --- | --- | --- | --- |
| 履歴(除外対象) | 28 / 61 | `docs/plans/archive/`(9 ファイル、25 hit)、`docs/reports/` の過去分(16 ファイル、21 hit)、`docs/evidence/`(3 ファイル、15 hit)。`docs/insights/events/` は 0 hit | なし |
| この task 自身の記録 | 2 / 43 | `docs/plans/active/2026-10-02-drop-gpt-5-5-default-pool.md`(29)と `docs/reports/self-review-2026-10-02-drop-gpt-5-5-default-pool.md`(14)。plan の `:21` と `:113` に旧状態の `!= 9` の経緯がある。経緯の記述であって、既定の主張ではない | なし |
| spec の記録 | 1 / 2 | `docs/specs/2026-08-01-org-runtime.md:11`(「`gpt-5.5` も含めていたが、2026-10-02 に既定から外した」)と `:23`(運用ノート (e))。どちらも除外した事実の記録 | なし(AC-4 が許す「spec の運用ノートの記録」) |
| codex CLI 自体の既定のモデル(Non-goal) | 3 / 8 | `.codex/config.toml:15,50,55`、`templates/base/.codex/config.toml:15,50,55`、`docs/specs/2026-05-07-codex-cli-parity.md:45,180`。`[org].model_pool` とは別の設定 | なし(plan の Non-goals) |
| codex の config.toml を模した fixture | 4 / 42 | `scripts/verify.local.sh`(2)、`tests/test-hook-wiring.sh`(3)、`tests/test-ralph-worktree.sh`(22)、`internal/cli/cli_test.go`(15、`writeCodexConfigToml` の `model = "gpt-5.5"` と `:43` の template fixture) | なし(既定のプールと無関係) |
| 任意の値として使う Go のテスト fixture | 7 / 96 | `internal/cli/doctor_org_test.go`(62。プールも cache も test 内で組む)、`internal/cli/org_test.go`(13。`:685` は `gpt-5.5-codex`、`:892-942` は AC-7 のテスト、`:1118-1121` は effective model の receipt の場面)、`internal/config/config_test.go`(14。`:132-136` は不在の確認、`:799` は明示プールの厳格検証、`:826-882` は AC-7)、`internal/org/verbs_test.go`(2)、`codex_session_test.go`(2、`gpt-5.5-old` / `-stale`)、`spawn_test.go`(1)、`doctor_shell_alias_test.go`(2) | なし。どれも `Default()` が `gpt-5.5` を含むことには依存しない(既定のプールに依存するテストは `TestDefault_Org` と lock-step で、8 エントリに追従済み) |
| コメントの例 | 1 / 1 | `internal/cli/doctor_shell_alias.go:627`(`-mgpt-5.5` の例) | なし |

出荷する面(`.claude/skills`、`.agents/skills`、`templates/base/.claude`、`templates/base/.agents`、`templates/base/ralph.toml`、`scripts/ralph-config.sh` と template)に、`gpt-5.5` を既定のプールとして書いた箇所はない。README、`docs/recipes/`、`docs/quality/` に既定プールの列挙はなく、`docs/tech-debt/README.md` にこの変更で無効になる行もない。

## 文書のずれ

既定のプールや復旧の手順について、コードと食い違う記述は見つからなかった。次は記録しておく項目で、いずれも verdict には影響しない。

| # | 重さ | 場所 | 内容 |
| --- | --- | --- | --- |
| D-1 | LOW(記録の更新待ち) | `docs/plans/active/2026-10-02-drop-gpt-5-5-default-pool.md:68-74`、`:126` | AC-1〜AC-7 のチェックボックスと「Verification artifact created」が未チェック。この report の後に orchestrator が更新する項目で、/verify は plan を編集しない |
| D-2 | 情報 | `docs/tech-debt/README.md` | 設定は cwd の `./ralph.toml`、state dir は git の toplevel から決まる非対称(self-review の probe C、Follow-ups)と、teardown 系の動詞が設定の検証で止まる件の行がない。この diff が作った問題ではなく、skill の記述(省略時は cwd の `./ralph.toml`)は正確。行にするかは /sync-docs かメンテナの判断 |
| D-3 | 情報(未決事項) | `docs/plans/active/2026-10-02-drop-gpt-5-5-default-pool.md:108` | `.codex/config.toml` の `model = "gpt-5.5"`(codex CLI の既定のモデル)はこの PR の対象外で、メンテナへの確認が Open questions に残っている。#156 の観測 3 では同じスラッグに "Legacy coding model." の表示が出ていた |

## Coverage gaps

- `go test ./... -count=1`、`run-verify.sh`、`run-test.sh` は実行していない(AC-6 の振る舞い側。/test の担当)。実行した 5 本の単発テストは、AC-1、AC-2、AC-7 に直結するものだけ
- `stop` と `disband` は壊れた設定で止まるところ(設定の読み込み時点で herdr / agmsg に届く前)までを実行した。設定が正しい状態での実動作(herdr と agmsg が要る)は実行していない
- `org start --model` の拒否は CLI では実行していない。`org.go:420` が `rt.Spawn` に入り、`spawn.go:375`、`:502` の検証が `spawn` と同じであることをコードで確認した
- 復旧後に座席が見えることは、CLI probe では manifest が空だったため `no seats` までで、座席の表示は単発の Go テストに依存する
- 利用者の実際の `models_cache.json`(`~/.codex`)は読んでいない
- self-review addendum の mutation probe と `.goreleaser.yml` の changelog の挙動(M-1 と L-5 の前提)は再確認していない。M-1 の結論の側(タグと v5.1.0 の既定)は、このレポートで自分で確認した
- default スコープで golang が選ばれた理由は未分類ファイルによる full fallback で、Go の差分だけで選ばれる経路は未実行

## Verdict

- Verdict: PASS(CRITICAL / HIGH / MEDIUM の指摘なし)。/test へ進める
- Verified: AC-1、AC-2、AC-3、AC-4、AC-5、AC-7。AC-6 の静的な半分。self-review の M-1、L-1〜L-5、A-L1、A-L2 が HEAD で直っていること。skill の復旧の段落と spec (e) の主張がコードと git に一致すること
- Partially verified: AC-6(静的は確認、`go test ./...` と `run-verify.sh` の振る舞い側は /test)。AC-2 の `defaults_sync_test.go` green(単発 probe では pass、suite 全体は /test)
- Not verified: 上の Coverage gaps の項目
