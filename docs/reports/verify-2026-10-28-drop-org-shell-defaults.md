# Verify report: drop-org-shell-defaults

- Date: 2026-10-28
- Plan: `docs/plans/active/2026-10-28-drop-org-shell-defaults.md`(AC1〜AC5、issue #273 の受け入れ条件 3 つ)
- Verifier: verifier サブエージェント(`/verify`、パイプラインの cycle 1)
- Scope: `git diff 15e301c8...HEAD`(HEAD c416a49c、9 コミット、12 ファイル)。静的検査は `./scripts/run-static-verify.sh` の既定(changed)で、`scripts/ralph-config.sh` が言語に分類されないため full に落ちて golang の verifier が走った。テストは走らせていない(`/test` の役目)。
- Evidence: `docs/evidence/verify-2026-10-28-drop-org-shell-defaults.log`(gitignore 対象。run-verify 自身の記録は `docs/evidence/verify-2026-10-28-101331.log`)

## Spec compliance

| Acceptance criterion | Status | Evidence |
| --- | --- | --- |
| AC1: 2 つの `ralph-config.sh` に `RALPH_ORG_AGMSG_HOME` 以外の `RALPH_ORG_*` の代入と export がなく、byte 一致 | 満たす | `grep -n 'RALPH_ORG_' scripts/ralph-config.sh \| grep -v RALPH_ORG_AGMSG_HOME` は出力なし(rc=1)、テンプレート側も同じ。残る `RALPH_ORG_` の行は `scripts/ralph-config.sh:46`、`:56`、`:60` で、すべて `RALPH_ORG_AGMSG_HOME`。export は `:70` の標準フローの 3 変数だけ。`cmp scripts/ralph-config.sh templates/base/scripts/ralph-config.sh` は rc=0。`sh -n` は両方 rc=0。削除は 18 変数と `export RALPH_ORG_*` の 3 行で、base の 19 キーの `check(...)` と数が合う |
| AC2: `Default()` とテンプレートの `[org]` がずれると落ち、値の変更とキーの削除の両方を単体テストで固定 | 満たす(静的に確認。実行は `/test`) | 比較関数 `orgTemplateDrift`(`internal/config/defaults_sync_test.go:148-175`)が probe の 19 キーを `valueDrift`/`listDrift`(`:179-199`)で比べ、`TestDefaultsLockStep` の `org` サブテスト(`:207-223`)が呼ぶ。単体テスト `TestOrgTemplateDrift_DetectsChangedAndMissingKeys`(`:314-389`)に、実テンプレートで差なし、`max_seats` の変更(`:333-336`)、`auto_merge` の削除(`:339-342`)、`codex_verified` の削除(`:345-348`)、`wake` の削除(`:351-354`)、`model_pool` の入れ替え(`:357-362`)がある。各ケースの置き換え前の文字列はテンプレートにちょうど 1 回あり(python3 -I で数えた)、同じ置き換えを tomllib で読む写しの比較に通すと、各ケースでキー名つきのずれが 1 行だけ出て、実テンプレートでは 0 行だった。probe の 19 キーは base の 19 個の `check(...)` と同じ集合 |
| AC3: `org.agmsg_home` の 3 面の照合と SKILL.md の fallback 照合が残る | 満たす(残し方は計画の Deviation に記録済み) | shell 対 `Default()` は `defaults_sync_test.go:236-241`、テンプレート対 `Default()` は `orgTemplateDrift` の `:161`。2 つが通れば 3 面が一致する。SKILL.md の照合は `:268-274` の 5 組で、base の `:214-218` と同じ。直接の 3 者比較から 2 つのサブテストの合成に変わったことは計画 `:119` に書いてある |
| AC4: `scripts/check-sync.sh`、`tests/test-ralph-config.sh`、`go test ./internal/config/` が通る | 一部(verify の分は満たす) | `./scripts/check-sync.sh` は単独で rc=0(DRIFTED 0、KNOWN_DIFF 5)。`go vet ./...` はテストファイルもコンパイルして通った。`tests/test-ralph-config.sh` と `go test` は `/test` が走らせる。計画では AC4 に `[x]` が付いているが、根拠は implementer の報告で、この report の時点では未確認 |
| AC5: 写しを前提にしたコメントと文書が新しい形に合い、消した変数名が履歴の外に残らない。FR-2 が 2 面を述べる | 満たす(INFO 2 件) | 計画の step 3 の `git grep`(`docs/evidence`・`docs/reports`・`docs/plans`・`docs/insights` を除く)は 0 件(rc=1)。Scope の 7 か所は直っている: `config.go:28-34`、`config_test.go:89-90`、`tests/test-ralph-config.sh:7-13`、spec の注記 (a)(`docs/specs/2026-08-01-org-runtime.md:45`)、FR-2(`:74`、日付つきの印「2026-10-28 #273 で `agmsg_home` のほかは照合から外し、config.go と `templates/base/ralph.toml` の 2 面にした」)、`docs/tech-debt/README.md:149`、`.claude/rules/ralph/model-routing.md:148`。言い方の細部は下の D-1、D-2 |

issue #273 の受け入れ条件との対応: (1) 写しがない → AC1。(2) ずれると落ちる → AC2(静的に確認)。(3) `check-sync.sh` と既存のテストが通る → `check-sync.sh` は確認、テストは `/test`。issue の「`tests/` で写しを読んでいるもの」は、`git grep RALPH_ORG_ -- tests` が 0 件で、該当なし。

## Static analysis

| Command | Result | Notes |
| --- | --- | --- |
| `./scripts/run-static-verify.sh` | pass(rc=0) | 2026-10-28T10:13:31Z。shellcheck、hook の `sh -n` 20 本、guard の awk、`settings.json` の jq 2 本、Codex の hook 系 3 本、`check-sync.sh`、`check-pipeline-sync.sh`、`check-skill-sync.sh`(14 skill)、`check-template-purity.sh`、tech-debt の plan 参照がすべて OK。golang: `gofmt: ok`、golangci-lint `0 issues.`、go vet と staticcheck は成功時に何も出さない。branch の secret scan は `15e301c8..c416a49c` で clean |
| `./scripts/check-sync.sh`(単独) | pass(rc=0) | IDENTICAL 172、DRIFTED 0、ROOT_ONLY 0、TEMPLATE_ONLY 11、KNOWN_DIFF 5(`model-routing.md` を含む) |
| `./scripts/check-template-purity.sh`(単独) | pass(rc=0) | テンプレートの `ralph-config.sh` の新しいコメントに meta-repo 固有の参照はない |
| `./scripts/check-skill-sync.sh`(単独) | pass(rc=0) | 14 skill が一致 |
| `sh -n` と `cmp`(2 つの `ralph-config.sh`) | pass | `sh -n` は両方 rc=0、`cmp` は rc=0。base の時点でも byte 一致だった |
| `gofmt -l internal/config` / `git diff --check 15e301c8...HEAD` | pass | どちらも出力なし |

## Documentation drift

| Doc / contract | In sync? | Notes |
| --- | --- | --- |
| `docs/specs/2026-08-01-org-runtime.md:45`(運用注記 (a)) | yes | 2 面(`Default()` と `templates/base/ralph.toml` の `model_pool`)と、値の一致とキーの記述を検査する、と書く。テストの実装と合う |
| `docs/specs/2026-08-01-org-runtime.md:74`(FR-2) | yes | 既存の改訂の印と同じ形で、`agmsg_home` の例外を含めて 2 面を述べる |
| `docs/specs/2026-08-01-org-runtime.md:47`(注記 (c)) | yes(INFO D-3) | `ralph upgrade` が core(skill・`ralph-config.sh`)を置き換える、という文はいまも正しい |
| `docs/specs/2026-08-01-org-runtime.md:135`(Impact)、`:157`(Codebase analysis) | 履歴 | 計画の Non-goals が履歴として残すと決めた。`:157` に「3 面ロックステップ」が残る |
| `docs/specs/2026-10-07-org-multi-org-director.md:213`(In scope)、`:237`(Impact) | 履歴 | 同上。この spec に lock-step を求める FR はない(grep で確認) |
| `docs/tech-debt/README.md:149` | yes | 「two value surfaces (`Default()`, `templates/base/ralph.toml`)」 |
| `.claude/rules/ralph/model-routing.md:148`(root) | yes(INFO D-2) | テンプレート側は KNOWN_DIFF で、この行を持たない |
| `internal/config/config.go:28-34` | yes | `AgmsgHome` だけ `RALPH_ORG_AGMSG_HOME` とも比べる、と書く |
| `internal/config/config_test.go:89-90` | yes(INFO D-2) | 畳んだ確認の代わりに `TestDefaultsLockStep` を指す |
| `tests/test-ralph-config.sh:7-13` | 一部(INFO D-1) | 残る中身の列挙から `RALPH_ORG_AGMSG_HOME` が抜けた |
| `scripts/ralph-config.sh:2-4`、`:45-54`(2 つの写し) | yes | 残るのは `RALPH_ORG_AGMSG_HOME` だけで、ほかの `[org]` の既定値は `config.Default()` と scaffold される `ralph.toml` にある、と書く |
| `AGENTS.md:111` | yes | `ralph-config.sh` の説明は変わっておらず、いまも正しい |
| `/org` skill の 4 つの写し `:182` | yes | `ralph upgrade` が core(skill・`ralph-config.sh`)を置き換えるだけ、という文はいまも正しい |
| `internal/org/driver/agmsg.go:134-137`、`internal/cli/doctor.go:792`、`docs/tech-debt/README.md:66` | yes | どれも `agmsg_home` の 3 面の話で、この PR のあとも 3 面のまま |
| `internal/org/director_cmd_test.go:953-954` | 無関係 | `RALPH_ORG_TEST_CHECKIN_*` はテストの補助の環境変数 |

INFO の細目(どれも動作に関係せず、マージを止めない):

- D-1: `tests/test-ralph-config.sh:9-13` は「The surviving surface is」と書いて標準フローの 4 変数を挙げるが、`scripts/ralph-config.sh:56` に残る `RALPH_ORG_AGMSG_HOME` を挙げていない。直す前の文は「and the [org] envelope lock-step vars」で、この変数を含んでいた。このテストは `RALPH_ORG_AGMSG_HOME` を検査しないので、「このテストが見る範囲」と読めば正しい。直すなら「`RALPH_ORG_AGMSG_HOME` は `internal/config` の lock-step test が見る」と 1 文足す。
- D-2: `model-routing.md:148` の「`config.Default()`'s `[org]` values match `templates/base/ralph.toml` (each key set, same value)」と `config_test.go:89` の「sets each [org] default」は、すべての `[org]` キーを検査すると読める。実際に検査するのは 19 キーで、`[org].roles`、`[org.permissions].roles`、`[org.director].launch_cmd`、`wake_cmd` は対象外(既定は空で、テンプレートはコメントの例しか書かない。`wake_cmd` は `config_test.go:91` が別に空を確かめる)。self-review の L4(19 という数がコメントにしかない)と同じ根から来ている。
- D-3: spec の注記 (c)(`:47`)は `model_pool` を変える手順の中で「core(skill・`ralph-config.sh`)」を挙げる。`ralph-config.sh` はもう `model_pool` を持たないので例として要らなくなったが、文は誤りではない。
- D-4: 計画の Implementation outline の step 2(`:74`)は「2 行の `export RALPH_ORG_*`」と書くが、base にあったのは 3 行で、3 行とも消えている。Progress の S2(`:115`)は 3 行と書いていて、計画の中で食い違う。記録だけの差。

## Observational checks

- 変異の静的な写し: `orgTemplateDrift` の比較を python3 -I と tomllib で写し、単体テストの 5 つの置き換えと、S1 の報告にある `max_orgs` 10→11 をテンプレートの写しに当てた。実テンプレートは 0 行、各変異は該当キーの 1 行だけ(`org.max_seats: changed 6`、`org.director.auto_merge: missing`、`org.permissions.codex_verified: missing`、`org.director.wake: missing`、`org.model_pool: changed […]`、`org.max_orgs: changed 11`)。tomllib と go-toml/v2 は別の実装なので、これは Go のテストの実行の代わりにはならない。
- `OrgConfig` の toml の葉(`config.go` の tag)は 23 個で、probe の 19 個との差は上の D-2 の 4 個。4 個とも既定が空の map か nil の slice。

## Coverage gaps

- `go test ./internal/config/`(`TestDefaultsLockStep` の 2 つのサブテストと `TestOrgTemplateDrift_DetectsChangedAndMissingKeys`)と `sh tests/test-ralph-config.sh` は走らせていない。`/test` が走らせる。
- go-toml/v2 がキーのない pointer field を nil のまま残すことは、`config.go` の `orgPoolKeysPresent` が同じ慣用句を使っていることと、self-review の読みに基づく。この report では実行で確かめていない(`/test` の `auto_merge` と `codex_verified` の削除ケースが確かめる)。
- 19 キーの列挙そのものの正しさ(`OrgConfig` に 20 個目の既定値を足したとき)はどのテストも固定していない。self-review L4 で、PR の Known gaps に回る予定。
- 前からある穴(この PR は触っていない): `internal/org/driver/agmsg.go:134-137` の `defaultAgmsgHome` は「`defaults_sync_test.go` を見よ」と書くが、そのテストは `driver` パッケージのこの定数を読まない。

## Verdict

- Verdict: pass
- Verified: AC1(grep、`cmp`、`sh -n`)、AC3(照合の所在と SKILL.md の 5 組)、AC5(Scope の 7 か所と、計画の `git grep` が 0 件)、AC4 のうち `check-sync.sh`、静的検査の全体(`run-static-verify.sh` rc=0、単独の 3 本も rc=0)、issue #273 の受け入れ条件 1 と 3 の check-sync の部分
- Partially verified: AC2(比較関数と単体テストの中身を読み、変異を tomllib の写しで当てた。Go での実行はしていない)、AC4(テストの部分は `/test`)
- Not verified: `go test ./internal/config/` と `tests/test-ralph-config.sh` の実行結果、go-toml/v2 が nil を残す挙動の実行での確認
