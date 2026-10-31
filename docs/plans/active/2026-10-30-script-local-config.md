# script-local-config

- Status: Approved
- Approved: 2026-10-30 sha256:cdd1e5ba1234
- Owner: Claude Code
- Date: 2026-10-30
- Related request: #276 の最後(B2)。スクリプトが読む個人用の値(plan-visual のブラウザと opener、cross-review と branch の secret scan の base、verify の base、secret scan の skip)を `ralph.local.toml` に移し、環境変数での上書きをやめる
- Related issue: 276
- Type: feat
- Branch: feat/script-local-config

## Objective

#276 は、人が決める値を `ralph.toml` と git が無視する個人用のファイル `ralph.local.toml` だけで決め、環境変数での上書きをやめる issue です。分け方は 4 本です(docs/plans/archive/2026-10-29-local-config-org.md の D1、docs/plans/archive/2026-10-30-standard-flow-config.md の D1)。

- A1(PR #287): 個人用のファイルの仕組みを足し、`RALPH_ORG_AGMSG_HOME` を撤去した。
- A2(PR #290): `RALPH_ORG_STATE_DIR` を撤去した。
- B1(PR #291): 標準フローの値を `[standard]` と `[reviewer]` に移した。helper `ralph_setting` を作った。
- B2(この計画): 最後の 1 本で、スクリプトが読む個人用の 5 つの値を受け持ち、`Closes #276` にする。

| 変数 | 行き先 | 読むスクリプト |
|---|---|---|
| `RALPH_PLAN_VISUAL_BROWSER` | `[plan_visual].browser` | `scripts/plan-visual.sh` |
| `RALPH_PLAN_VISUAL_OPENER` | `[plan_visual].opener` | `scripts/plan-visual.sh` |
| `RALPH_XREVIEW_BASE` | `[standard].base_branch` | `scripts/xreview-helpers.sh`(`detect_base_branch`。`/cross-review` と、ローカルの `scripts/secret-scan-branch.sh`) |
| `RALPH_VERIFY_BASE` | `[verify].base` | `scripts/detect-changed-languages.sh`(changed scope の diff の base) |
| `RALPH_VERIFY_SKIP_BRANCH_SECRET_SCAN` | `[verify].skip_branch_secret_scan` | `scripts/run-verify.sh` |

5 つとも、`ralph.toml` に書けない local-only のキーにします(D2)。全変数の一覧は B1 の計画の冒頭にあります。この計画で、一覧のうち「人が決める値」はすべて設定ファイルに移ります。例外として残す 3 つ(`COVERAGE_THRESHOLD`、`RALPH_TERRAFORM_INIT_VALIDATE`、`RALPH_PRIMARY_CLI`)の理由は、B1 の D6 のとおりです。

## Scope

- **config**(`internal/config/`)
  - 新しい表を足す。
    - `[plan_visual]`: `browser`(string)、`opener`(string)
    - `[verify]`: `base`(string)、`skip_branch_secret_scan`(bool)
  - `[standard]` にキーを足す: `base_branch`(string)
  - 既定値はどれも空か false。
  - 5 つを `config.LocalOnlyKeys()` に足す。`ralph.toml` に書くと、Load がパスと `ralph.local.toml` を名指してエラーにする。キーの大文字小文字は B1 と同じ規則(go-toml の decoder と同じ)で引く。
  - 値の検証(Load。個人用のファイルの値も含む)は次のとおり(D3)。
    - `plan_visual.browser`: 空、`none`、絶対パスのどれか。`~/` は許さない。
    - `plan_visual.opener`: 空、`none`、空白・制御文字を含まず先頭が `-` でない 1 語のどれか。
    - `standard.base_branch`: 空か、`[A-Za-z0-9._/-]+` で、先頭が `-` でなく `..` を含まないもの。
    - `verify.base`: 空か、空白・制御文字を含まず先頭が `-` でないもの。`HEAD~1`・`HEAD^` は通す。
  - template の `ralph.toml` には注釈で書く。local-only なので「`ralph.local.toml` にだけ書ける」と書く。
- **helper**(`scripts/ralph-config.sh` と写し)
  - 表に 5 つのキーと既定値を足す。既定値は `''` と `false`。
  - 既定値が空のキーでは、exit 0 の空の出力を正しい値として黙って返す(D4)。既定値が空でないキーは、今のまま 1 行知らせて既定値に戻る。
  - 新しい関数 `ralph_setting_or_fail <key>` を足す(D7)。
    - 値の出し方は `ralph_setting` と同じ。127 と `unknown command "config"` のときは、黙って既定値を出して 0 を返す。
    - それ以外の失敗(exit 2・3 など)では、stderr に 1 行出し、stdout には何も出さず、0 以外を返す。
    - 既存の `ralph_setting` の約束(どの場合も 0 を返し、`set -e` の呼び出し側で止まらない)は変えない。
- **スクリプト**(どれも `templates/base/scripts/` の写しも)。環境変数は読まない。
  - `scripts/plan-visual.sh`: `resolve_opener` と `resolve_browser` は `ralph_setting plan_visual.opener` / `plan_visual.browser` で読む。`none` と空の意味は今の env と同じ。
  - `scripts/xreview-helpers.sh`: `detect_base_branch` は `ralph_setting_or_fail standard.base_branch` で読む(D7)。
    - 読み出しが失敗したら(設定の誤り)、`detect_base_branch` は stderr に理由を出して 0 以外を返し、自動の base に黙って戻らない。
    - 呼び出し側の扱い: `/cross-review` は止まって知らせる。`secret-scan-branch.sh` は exit 3(scan できない)で push を止める。
    - `ralph-config.sh` は、自分のディレクトリからの相対パスで source する。
    - 注釈「Pure function」は、`ralph config get` を打つ形に直す。
  - `scripts/secret-scan-branch.sh`: CI では `GITHUB_BASE_REF` が先に決まり、helper を打たないことを保つ。
  - `scripts/detect-changed-languages.sh`: `ralph_setting_or_fail verify.base` で読む(D7)。読み出しが失敗したら、stderr に理由を出し、changed scope ではなく full scope(すべての言語)を選ぶ。`no_changes` にはしない。
  - `scripts/run-verify.sh`: `ralph_setting verify.skip_branch_secret_scan` が `true` のときだけ、branch の secret scan を飛ばす。
- **org の gate**(`internal/org/gate_run.go`): 振る舞いは変えない。
  - スクリプトには `RALPH_LOCAL_CONFIG=off` を渡すので、個人用の値は gate に効かない。
  - `RALPH_VERIFY_*` を消すのは scope と roots のために残す。
  - 注釈のうち `RALPH_VERIFY_BASE=HEAD` を理由に挙げる文を直す。
  - 個人用のファイルの skip と base が gate に効かないことを、テストで固定する。
- **doctor**(`internal/cli/doctor.go` の `retiredEnvs`)
  - 5 行を足し、代わりのキー(`ralph.local.toml` の)を名指す。
  - B1 の `retiredSettingEffect` は「skill と ralph-config.sh が読む」と書いているので、B2 の 5 行には「スクリプトが読む」形の文を用意する。
- **文書**
  - `docs/quality/quality-gates.md`(と写し)の `RALPH_VERIFY_BASE` の案内。
  - `/cross-review` の skill:67(`$RALPH_XREVIEW_BASE` の解決順。4 つの写し)。
  - README の `ralph.local.toml` の節(5 つのキー)、`.claude/rules/ralph/` と recipe で当たるもの。
- **テスト**
  - 共通の偽の `ralph` を `tests/lib/fake-ralph.sh` に置く。キーごとの値と終了コードを、テストが渡せる形にする。
  - env で値を渡していたテストを、偽の ralph を PATH の先頭に置く形に書き換える。
    - `tests/test-plan-visual.sh`
    - `test-secret-scan-branch.sh`
    - `test-xreview-helpers.sh`
    - `test-run-verify-branch-secret-scan.sh`
    - `test-detect-changed-languages.sh`
    - `test-run-verify-concurrency.sh`・`test-run-verify-scope.sh`(unset の行)
  - 環境変数を設定しても効かないことを、各テストで 1 つずつ見る。

## Non-goals

- B1 で扱った値(`[standard].max_pipeline_cycles`、`[reviewer]`)。
- 例外として残す 3 つの環境変数(`COVERAGE_THRESHOLD`、`RALPH_TERRAFORM_INIT_VALIDATE`、`RALPH_PRIMARY_CLI`。B1 の D6)。
- 内部の受け渡しの変数(`RALPH_VERIFY_SCOPE`、`HARNESS_VERIFY_MODE`、`RALPH_VERIFY_PROJECT_ROOTS` など)。
- org の gate の振る舞い。
- CI の workflow。
- 古い `ralph` binary を入れ直すこと。

## Assumptions

- `ralph config get` は空文字の値を、空の行として exit 0 で出す(`internal/cli/config.go:98`、127-128)。
- CI には ralph の binary がない。スクリプトの `ralph config get` は 127 になり、helper は黙って既定値(空と false)に戻るので、CI の振る舞いは変わらない。
- org の gate はスクリプトに `RALPH_LOCAL_CONFIG=off` を渡す(`internal/org/gate_run.go:56`)。個人用のファイルは読まれず、local-only のキーは `ralph.toml` に書けないので、gate では 5 つとも既定値になる。
- この repo の PATH の ralph は v6.0.0 で `config get` がない。新しい binary を入れるまで、個人用の値は効かない(既定値になる)。

## Affected areas

- `internal/config/`(`config.go`、`local.go`、テスト、`defaults_sync_test.go`)
- `internal/cli/doctor.go` とテスト、`internal/org/gate_run.go`(注釈)と gate のテスト
- `scripts/{ralph-config,plan-visual,xreview-helpers,detect-changed-languages,run-verify}.sh`(と `templates/base/scripts/` の写し)、`scripts/secret-scan-branch.sh`(必要なら)
- `tests/lib/fake-ralph.sh`(新規)と上のテスト
- `docs/quality/quality-gates.md`(と写し)、`/cross-review` の skill(4 つの写し)、README、`templates/base/ralph.toml`

## Visual review

- ページ: `.harness/state/plan-visual/script-local-config.html`。かんたん版の図は `overview`(今とこれから)の 1 つ。折りたたみの中に、詳しい図が 3 つある。
  - `detail-overview`: ファイルの地図
  - `detail-flow`: 手元、CI、gate での読み方
  - `detail-keys`: 変数とキーの対応
- lint: `./scripts/plan-visual.sh lint` が exit 0。
- self-check: 全体と 4 つの図を `shot` で撮った。ファイルの地図と読み方の図の PNG を読み、崩れはなかった。かんたん版の図と対応表は B1 のページと同じ作りで、文字の幅も枠に収まる計算。

## Design decisions

分かれ目は consult(consult-plan-scriptlocal276b2)に聞き、推奨案を選んだ(ユーザーの指示)。

- D1 この PR は `Closes #276` にする。
  - 元の依頼は「merge と close まで」。
  - 例外の 3 つの env は、B1 の D6 と PR #291 に理由を書いた。PR 本文の先頭で、受け入れ条件「残した環境変数がすべて内部の受け渡し」は、この例外付きで満たすと書く。読み替えを拒まれたら、issue を開け直す(安い)。
- D2 5 つとも `config.LocalOnlyKeys` にする。
  - どれも機械ごとか、1 回だけの値。env はもともと commit されない値なので、local-only にすれば今と同じ。
  - あとで緩めるのは互換で、締めるのは非互換(B1 の計画の consult)。
  - secret scan の skip は、commit されると全員の scan が止まる。
- D3 値の検証
  - `browser` は、空・`none`・絶対パスだけ。`config get` は値をそのまま出し、`plan-visual.sh` は `[ -f "$browser" ]` で見るので、`~/` は字面のままでは必ず失敗する。
  - `opener` は 1 語(`"$opener" "$page"` で使う)。
  - `standard.base_branch` はブランチ名の文字だけ。
  - `verify.base` は、`HEAD~1`・`HEAD^` のような rev を通す。文書と過去の報告がこの形を案内しているので、断るのは先頭の `-` と空白・制御文字だけにする。
  - 値はスクリプトで二重引用符の中に入るので、`$` などは shell では展開されない。検証は、明らかな誤り(オプションに見える値、空白)を先に断るためのもの。
- D4 helper は、既定値が空のキーでは exit 0 の空を正しい値として黙って返す。
  - 設定していない人に毎回 warn が出ないようにするため。
  - 案 b(`none` のような目印を既定にする)は、キーごとに意味が変わり、スクリプトの分岐が増える。
- D5 テストは、共通の偽の ralph(`tests/lib/fake-ralph.sh`)を PATH の先頭に置いて値を渡す。B1 の `tests/test-ralph-config.sh` にある偽の ralph と、まとめられるならまとめる(S2 で判断)。
- D6 撤去した 5 つの変数は、doctor の `retiredEnvs` に足して warn を出す(B1 の D7 と同じ)。
  - スクリプトは環境を見ない。
  - 値を読まなくなったときの害はどれも既定値に戻ることで、小さい。scan の skip は scan が走る側に倒れる。
- D7 base の 2 つのキーは、設定の誤りで黙って既定値に戻らず、安全な側に倒す。
  - 計画助言の HIGH を受けて足した。
  - 設定の検証は全体で 1 つなので、別のキー(`plan_visual.browser` など)の誤りで `ralph config get verify.base` も exit 3 になる。このとき既定値の空に戻ると、自動の base になる。ローカルの main では base が main 自身になり、changed scope が `no_changes` で成功し、何も検査しない。`standard.base_branch` も、黙って戻ると review と scan の範囲が変わる。
  - そのため base の読み出しには `ralph_setting_or_fail` を使う。失敗したら、検証は full scope に倒し、`detect_base_branch` は失敗にする(cross-review は止まり、secret scan は push を止める)。
  - 黙って既定値に戻るのは、ralph がないとき(CI)と古い ralph のときだけ。このとき個人用のファイルは読めないので、設定の誤りは起きない。
  - 退けた案: `ralph_setting` 自体を失敗で 0 以外にする。B1 の skill と `set -e` の呼び出し側の約束を壊す。
  - ブラウザ・opener・scan の skip は、今の `ralph_setting` のまま。失敗しても自動検出に戻るか、scan が走る側に倒れるので、害は小さい。

## Acceptance criteria

- [x] AC1 設定のキー
  - `ralph config get` で 5 つのキーが既定値を出す: `plan_visual.browser`・`plan_visual.opener`・`standard.base_branch`・`verify.base` は空、`verify.skip_branch_secret_scan` は `false`。
  - `ralph.local.toml` に書いた値が効く。`ralph.toml` に書くと Load がエラー(exit 3)にし、パスと `ralph.local.toml` を名指す。大文字小文字を変えても同じ。
  - D3 の検証に合わない値は exit 3 で、キーと値を名指す。
- [x] AC2 helper の表に 5 つのキーがあり、`Default()` と一致する(`defaults_sync_test.go`)。既定値が空のキーでは、exit 0 の空を黙って返す。
- [x] AC3 スクリプト
  - 4 つのスクリプト(`plan-visual.sh`、`xreview-helpers.sh` の `detect_base_branch`、`detect-changed-languages.sh`、`run-verify.sh`)は、5 つの値を helper で読む。
  - 5 つの環境変数を設定しても、振る舞いは変わらない。
  - `none` と空の意味は今と同じ。
  - 偽の ralph で値を渡すテストで確かめる。
- [x] AC3b 設定の誤りでの扱い(D7)
  - `ralph config get verify.base` が exit 2・3 で失敗すると、`detect-changed-languages.sh` は stderr に理由を出し、full scope を選ぶ。`no_changes` は出さない。
  - `ralph config get standard.base_branch` が失敗すると、`detect_base_branch` は 0 以外を返す。`secret-scan-branch.sh` は exit 3 になり、`/cross-review` の skill は止まって知らせる。
  - `ralph_setting_or_fail` は、127 と `unknown command` では黙って既定値を出して 0、それ以外の失敗では stdout が空で 0 以外を返す。
- [x] AC4 CI と gate
  - ralph が PATH にないとき、4 つのスクリプトは今の既定の振る舞い(自動検出、merge の対象の branch、first-parent の base、scan を走らせる)のまま動く。
  - CI で `GITHUB_BASE_REF` があるとき、`secret-scan-branch.sh` は helper を打たない。
  - org の gate では、個人用のファイルの skip と base が効かない。
- [x] AC5 `ralph doctor` は、撤去した 5 つの変数が空でない値で設定されているとき、代わりのキーを名指す warn を出す。設定されていなければ何も出さない。
- [x] AC6 文書
  - 5 つの変数に触れるのは「もう読まない」の説明だけになる。
  - `quality-gates.md`、`/cross-review` の skill、README の `ralph.local.toml` の節が、新しいキーを案内する。
- [x] AC7 `go build ./...`、gofmt、`go vet`、golangci-lint、shellcheck、`check-sync.sh`、`check-skill-sync.sh`、`./scripts/run-verify.sh` が通る。

## Implementation outline

1. S1 config: 5 つのキー、local-only、検証、template の注釈、`ralph config get` のテスト — verify: `go test ./internal/config/ && go test ./internal/cli/ -run 'Config'`
2. S2 helper: 表に 5 つのキー、空の扱い(D4)、`ralph_setting_or_fail`(D7)、`tests/test-ralph-config.sh`、共通の偽の ralph `tests/lib/fake-ralph.sh`、`defaults_sync_test.go` の照合 — verify: `sh tests/test-ralph-config.sh && go test ./internal/config/ -run 'Sync|Drift|Default|LockStep' && ./scripts/check-sync.sh`
3. S3 plan-visual: `scripts/plan-visual.sh`(と写し)と `tests/test-plan-visual.sh` — verify: `sh tests/test-plan-visual.sh && ./scripts/check-sync.sh`
4. S4 base: `scripts/xreview-helpers.sh`(と写し)、`scripts/secret-scan-branch.sh`(必要なら)、`tests/test-xreview-helpers.sh`・`tests/test-secret-scan-branch.sh` — verify: `sh tests/test-xreview-helpers.sh && sh tests/test-secret-scan-branch.sh && ./scripts/check-sync.sh`
5. S5 verify: `scripts/detect-changed-languages.sh`・`scripts/run-verify.sh`(と写し)、`internal/org/gate_run.go` の注釈と gate のテスト、関係する `tests/test-*.sh` — verify: `sh tests/test-detect-changed-languages.sh && sh tests/test-run-verify-branch-secret-scan.sh && sh tests/test-run-verify-concurrency.sh && sh tests/test-run-verify-scope.sh && go test ./internal/org/ -run 'Gate' && ./scripts/check-sync.sh`
6. S6 doctor: `retiredEnvs` に 5 行とテスト — verify: `go test ./internal/cli/ -run 'Retired|Doctor'`
7. S7 文書: `quality-gates.md`(と写し)、`/cross-review` の skill(4 つの写し)、README、rules・recipe で当たるもの — verify: `./scripts/check-sync.sh && ./scripts/check-skill-sync.sh && ! grep -rnE '\$\{?RALPH_(PLAN_VISUAL_BROWSER|PLAN_VISUAL_OPENER|XREVIEW_BASE|VERIFY_BASE|VERIFY_SKIP_BRANCH_SECRET_SCAN)' scripts templates/base/scripts .claude/skills templates/base/.claude/skills`

## Verify plan

- 静的解析: gofmt、`go vet`、golangci-lint、shellcheck(`run-static-verify.sh`)。
- 仕様の適合: AC1〜AC6 を確かめる。
  - HEAD から build した binary の probe: 一時 repo で `ralph config get` と `ralph doctor` を打つ。
  - 4 つのスクリプトを、偽の ralph と本物の binary の両方で打つ。
  - grep とコードの行。
- 文書のずれ: 5 つの変数名の grep(記録を除く)、`check-sync.sh`、`check-skill-sync.sh`。
- evidence: probe の出力、grep の結果。

## Test plan

- 単体(Go)
  - 5 つのキーの既定値と、`ralph.local.toml` の値。
  - `ralph.toml` に書いたときのエラー(大文字小文字の変化も)。
  - D3 の検証。通る例: `HEAD~1`・`origin/main`・`/Applications/…`・`none`。断る例: 先頭の `-`、空白、`..`、相対パス、`~/`。
- 照合(Go): helper の表と `Default()`。
- shell(偽の ralph)
  - 4 つのスクリプトで、値あり・空・`none`・ralph なし(127)・古い ralph(unknown command)。
  - 環境変数を設定しても効かないこと。
  - `secret-scan-branch.sh` は `GITHUB_BASE_REF` があると helper を打たない(偽の ralph が呼ばれた記録で見る)。
  - D7 の回帰
    - 偽の ralph が `verify.base` で exit 3 を返すと、`detect-changed-languages.sh` は full scope を選ぶ(`no_changes` を出さない)。わざと失敗する言語の検証を置いた一時 repo で、`run-verify.sh` の changed scope が失敗することも見る。
    - 偽の ralph が `standard.base_branch` で exit 3 を返すと、`detect_base_branch` は 0 以外を返し、`secret-scan-branch.sh` は exit 3 になる。
    - /verify では、本物の binary で同じ形を確かめる。`ralph.local.toml` に正しい `verify.base` と、誤った `plan_visual.browser` を書く。
- gate(Go): 個人用のファイルに skip と base を書いても、gate の scan が走り、base が既定になる。
- doctor(Go): 5 行(設定あり・空・未設定)。
- 回帰: `run-test.sh` 全体。
- evidence: テストの出力、変異(helper の空の扱い、スクリプトが env を読む形に戻すなど)。

## Risks and mitigations

- この repo の PATH の ralph(v6.0.0)は `config get` がないので、新しい binary を入れるまで個人用の値は効かない。今 env で値を変えている人は、更新のあと既定値に戻る。
  - 対策: doctor の warn、release のノート、最終の報告。
- テストの書き換えが多い(shell 7 本)。env を偽の ralph に替える途中で、意図が弱まるおそれがある。
  - 対策: 共通の偽の ralph にし、各テストに「env を設定しても効かない」の行を足す。変異で確かめる。
- スクリプトが `ralph config get` を打つので、ralph がある環境では 1 回ずつ git と設定の読み込みが増える。回数は、`run-verify.sh` と `detect-changed-languages.sh` と `detect_base_branch` で 1 回ずつ。
- 3 セッションの並行で、`scripts/` の写し、skill の写し、`docs/tech-debt/README.md`、README が衝突しうる。
  - 対策: `git merge origin/main` で取り込み、両方の変更を残す。

## Rollout or rollback notes

- release のノートに書くこと:
  - スクリプトは `RALPH_PLAN_VISUAL_BROWSER`・`RALPH_PLAN_VISUAL_OPENER`・`RALPH_XREVIEW_BASE`・`RALPH_VERIFY_BASE`・`RALPH_VERIFY_SKIP_BRANCH_SECRET_SCAN` を読まなくなった。
  - 代わりに `ralph.local.toml` の `[plan_visual].browser`・`opener`、`[standard].base_branch`、`[verify].base`・`skip_branch_secret_scan` に書く。`ralph.toml` には書けない。
  - `ralph doctor` は、まだ設定されている変数に warn を出す。
  - スクリプトがこの値を読むには、新しい binary が要る。
- 戻すときは、この PR を revert する。設定ファイルの形式はキーが増えるだけ。revert した ralph は、新しいキーを知らないキーとして無視する(推測、未確認)。

## Open questions

- なし。

## Progress checklist

- [x] Plan reviewed — plan advisory: ran with codex
  - 設定と実行: switch は scratchpad の ralph で `standard.plan_advisory` = true と読んだ。PATH の ralph は v6.0.0 で `config get` がない。codex は gpt-6-astra、xhigh、read-only。モデルと effort は B1 の新しい skill どおり `ralph_setting` で読み、PATH の ralph が古いので既定値になった。前後の git の状態は同じ。
  1. [HIGH] 設定の誤りで、検証が何も確かめずに成功しうる。
     - 設定の検証は全体で 1 つなので、別のキーの誤りで `verify.base` の読み出しも exit 3 になる。helper が既定値の空に戻ると、ローカルの main では base が main 自身になり、changed scope が `no_changes` で成功する。`standard.base_branch` も、黙って戻ると review と scan の範囲が変わる。
     - ユーザーの指示(HIGH は聞かずに反映)どおり計画に入れた。D7 を足した。
       - base の 2 つのキーは `ralph_setting_or_fail` で読み、設定の誤りで安全な側に倒す。検証は full scope、`detect_base_branch` は失敗。cross-review は止まり、secret scan は exit 3。
       - 黙って既定値に戻るのは、127 と古い ralph のときだけ。
       - 既存の `ralph_setting` の約束は変えない。
     - Scope の helper とスクリプト、AC3b、Test plan(偽の ralph の回帰と、本物の binary の probe)、Implementation outline の S2、図解のページ(「気をつけること」と detail-flow の注)を直した。
- [x] Plan approved — 自動承認(ユーザーの指示)。digest sha256:cdd1e5ba1234。分かれ目は consult(consult-plan-scriptlocal276b2)に聞いて推奨案を選び、理由は Design decisions(D1〜D7)に書いた。consult は 7 つの分かれ目のうち 3(値の検証)だけ直すよう言い、browser の `~/` を外し、`verify.base` に `HEAD~1`・`HEAD^` を通すようにした。
- [x] Branch created — feat/script-local-config(worktree `.claude/worktrees/script-local-config`、state id `plan-script-local-config`、base 3d497a4f)
- [x] Implementation started
  - S1 8bb68fe2(impl-276b2-s1)
    - 足したもの: `PlanVisualConfig`(`[plan_visual]`)、`VerifyConfig`(`[verify]`)、`StandardConfig.BaseBranch`、`LocalOnlyKeys()` に 5 つ。
    - 検証: `validateLoaded` の最後で `validateScriptValues` を呼ぶ。opener と `verify.base` は共通の `isScriptWord` で見る。browser は計画に加えて制御文字も断る。
    - template: 5 つを注釈の形(`# base_branch = ""` など)で書き、B1 の `commentedTemplateProbe` に載せた。`[plan_visual]` と `[verify]` は中身が注釈だけの表。全キーの照合を 4 つの表に広げ、テストの名前を `…EveryStandardReviewerPlanVisualAndVerifyKeyIsChecked` に変えた。
    - 新しいテストは `local_script_keys_test.go` に置いた(`local_test.go` の tech-debt の Trigger を避けるため)。`config get --help` は local-only のキーを列挙していないので、変えていない。
    - 検証: config 611 件・cli 190 件 PASS、golangci-lint 0 件、check-sync rc=0。変異 3 つはすべてテストが落とした。self-check: 改行 1 つを fixed、ほかは clean。
  - S1b 67d9d80b(impl-276b2-s1b): tech-debt の行(B1 の `defaults_sync_test.go` が 800 行を超えた件)の Trigger に S1 が当たったので、移動だけのスライスを足した。
    - B1 と S1 の照合を `defaults_sync_settings_test.go` に中身を変えずに移した。
      - 移したもの: template の注釈の照合、2 つのテスト、helper と skill の照合。
      - 結果: `defaults_sync_test.go` は 977 → 620 行、新しいファイルは 381 行。
    - `go test -list` は移す前と後で同じ 104 本。移した範囲は `cmp` で一致。
    - tech-debt の行は 2 つに分けた。`defaults_sync_test.go` の分は閉じ、`org_feature_test.go` の分は残した。
    - 範囲外: `scripts/ralph-config.sh:41`(と写し)の注釈が、表を読むテストのファイルを `defaults_sync_test.go` と書いている。S2 で `defaults_sync_settings_test.go` に直す。
  - S2 e4794cd7(impl-276b2-s2)
    - helper
      - 分け方は `_ralph_setting_get` の 1 か所。`ralph_setting` は失敗で既定値を出して 0 を返す(空の既定なら「using the empty default」)。
      - `ralph_setting_or_fail` は失敗で「not using the default」を出し、stdout には何も出さずに 1 を返す(mktemp の失敗も失敗に数える)。
      - exit 0 の空は、既定値が空のキーだけ値として扱う。
      - 表に 5 つ(空と `false`)を足し、注釈のテストのファイル名を直した。
    - 偽の ralph `tests/lib/fake-ralph.sh`
      - 使い方は `fake_ralph_install`・`_value`・`_fail`・`_old`・`_calls`・`_reset`。状態は `BIN/ralph.fake/` に置き、env は読まない。
      - 答えのないキーは exit 2、`config get` 以外は exit 99。
      - `_old` の stderr は本物の v6.0.0 と同じ 1 行(`unknown command "config" for "ralph"`、`Error:` なし)で、手元で確かめた。
    - `tests/test-ralph-config.sh` は共通の偽の ralph に置き換えた。
      - 1 つの shell で複数のキーを順に呼ぶ形にした(1 ケース 1 shell では 243 秒かかった)。全キーを回すのは /bin/sh だけで、ほかの shell は代表の 3 キー。所要は約 120 秒で、元と同じ程度。
      - tech-debt の「Three test gaps of standard-flow-config」(a) の Trigger に当たったので、mktemp が失敗するケースを 4 つの shell で足した。
    - Go の照合: 期待値は `shellSettingWant(def)` にまとめ、bool は `strconv.FormatBool` で表す。
    - 検証
      - shellcheck rc 0、test-ralph-config 272/272、test-codex-exec-invocation 160/160、check-sync rc 0。gofmt・vet・go test(`shell_and_skills` は PASS)も通った。
      - 変異: shell の 7 つと Go の 3 つは、どれもテストが落とした。
    - self-check: typos を 1 つ fixed(Go の doc の `''` を言葉にした)、ほかは clean。
    - S7 / /sync-docs に回すもの
      - tech-debt のその行の (a) を閉じる。
      - `internal/cli/config.go:77-83` の `config get --help` は、`ralph_setting` を読むのは skill だけ、と書いている(`config_test.go:302` が固定)。スクリプトと `_or_fail` に合わせる。
      - template の model-routing.md の「one of those five keys」。
      - `scripts/verify.local.sh` の shellcheck が `tests/lib/*.sh` を見ていない。
  - S3 674b193c(impl-276b2-s3)
    - `plan-visual.sh`(と写し)は、新しい関数 `plan_visual_setting` で `[plan_visual].opener`・`browser` を読む。読む場所は、スクリプトのある repo の root(`scripts/` の親)で、呼んだ人の cwd には依らない。
    - 読むのは `open` で opener を 1 回、`shot` で browser を 1 回だけ。`lint`・`digest`・usage では `ralph` を打たない(呼び出し記録で確かめた)。
    - 値の意味は env と同じ(空は自動、`none` は使わない)。warn・注釈・`--help` はキー名に直した。
    - テスト
      - 偽の ralph で値を渡す形にした。env が効かないことを見るケースを足した。127・古い ralph・exit 3 は opener のケースで見る。
      - ブラウザの自動検出の 2 件は、`/Applications` に本物のブラウザがあると起動してしまうので、手元では SKIP。Linux の CI で走る。PR の CI のログで、SKIP でなく通ったかを確かめる。
    - 検証: shellcheck rc 0、test-plan-visual 272 passed(SKIP 1)、check-sync rc 0、grep の否定 rc 0、変異 3 つはテストが落とした。self-check: 改行 1 つを fixed、ほかは clean。
    - consult の注意 1(スクリプトを PATH 経由で裸の名前で呼ぶと、`ralph-config.sh` の source に失敗しうる)は直さない。PATH 経由では `$0` はスラッシュ付きで、`.` になるのは `scripts/` の中で打つときだけ。そのときは `.` が正しい。PR の Known gaps に 1 行書く。
  - S4 7803a530(impl-276b2-s4)
    - `detect_base_branch`(と写し)は、`git rev-parse --show-toplevel` の `scripts/ralph-config.sh` を subshell の中で source し、root で `ralph_setting_or_fail standard.base_branch` を打つ。
      - 失敗(設定の誤り)では、stderr に 1 行出して 0 以外を返し、自動の段に戻らない。
      - root がない、またはファイルがないときは、黙って自動の段に進む。
      - 注釈の「Pure function」を直した。
    - `secret-scan-branch.sh`: `detect_base_branch` が失敗したら `cannot_scan` で止まる(`--strict` では exit 3、既定のモードは今の cannot_scan と同じく報告だけ)。`GITHUB_BASE_REF` があるときは helper を打たない。
    - `/cross-review` の skill: 手順 4 の 3 か所を `BASE=$(detect_base_branch) || exit 1` にし、解決順の説明を `[standard].base_branch` に直した。4 つの写し。
    - 検証
      - shellcheck rc 0、test-xreview-helpers 53/53、test-secret-scan-branch 263/263、check-skill-sync・check-sync ok、go test(Sync|Settings|Skill)ok。
      - 変異 7 つは、どれもテストが落とした。
      - この repo の `secret-scan-branch.sh --strict` は、PATH の v6.0.0 で今どおり rc 0。
    - handoff の広い grep(`RALPH_XREVIEW_BASE`)は、S2 の `ralph-config.sh` の「もう読まない」の注釈 2 行だけに当たる。範囲のファイルと展開の形の grep は 0 件(orchestrator が承認)。
    - self-check は 5 項目とも clean。
    - 範囲外(S5 で直す): `tests/test-run-verify-branch-secret-scan.sh:74,91` と `test-run-verify-concurrency.sh:34` に、`unset … RALPH_XREVIEW_BASE` が残っている。
  - S5 5f201546(impl-276b2-s5 → impl-276b2-s5r)
    - 最初の implementer は、使用量の上限で途中で止まった。残っていたのは `detect-changed-languages.sh` の頭の注釈の書き換えだけ(未コミット)。`/implement` の手順 7 どおり、その変更をスライスの分として引き継ぐ再開で出し直した。
    - `detect-changed-languages.sh`
      - git の toplevel で `ralph_setting_or_fail verify.base` を読む(S4 と同じ形)。
      - 失敗したら `fallback_full config_error:verify.base` で full scope に倒す。空なら自動の base に進む。
      - 引き継いだ注釈は、コードと合っていたのでそのまま使った。
    - `run-verify.sh`: scan の分岐の中だけで `./scripts/ralph-config.sh` の `ralph_setting verify.skip_branch_secret_scan` を読み、`true` のときだけ飛ばす。repo の root で動く前提は確かめた(自分で `./scripts/*` を打つ、ラッパーと CI と gate の cwd)。
    - gate
      - `gate_run.go` は注釈だけを直した。
      - `gate_runverify_test.go`: 一時の repo に `ralph-config.sh` と偽の ralph(`RALPH_LOCAL_CONFIG=off` なら既定値、それ以外なら HEAD と true)を置いた。gate が skip を off で聞いて scan を走らせることを、呼び出し記録で見る。
      - `RALPH_LOCAL_CONFIG=off` で 5 つのキーが既定値を出すことは、cli の `TestConfigGet_PrintsTheScriptLocalOnlyValues` に足した。
    - テスト
      - detect 113/113、scan 50/0、concurrency 37/37、scope 25/25。
      - D7 の回帰は test-run-verify-scope.sh にある。main の上で、空なら `no_changes`(対照)、exit 3 なら full になり、わざと失敗させた golang の検証で落ちる。
      - unset の行から、もう読まない変数を外した。
    - 検証: shellcheck rc 0、check-sync rc 0、gofmt・vet rc 0、否定の grep rc 0、Go(org 150 秒、config、cli)ok。変異 3 つはどれもテストが落とした。
    - self-check は 5 項目とも clean。consult は「進めてよい」。古い `ralph-config.sh` を手で写した repo では、エラーの文が実態とずれる(低)。
    - 範囲外(S7 で直す): `docs/quality/quality-gates.md` の 43・49 行(と写し)の `RALPH_VERIFY_BASE`。
  - S6 a338dc66(impl-276b2-s6)
    - `retiredEnvs` に 5 行を足した。新しい helper `retiredScriptEnv(name, key, script, more)` で作る。
      - effect には、変数ごとに実際に読んでいたスクリプトを名指す(「`ralph upgrade` が `scripts/ralph-config.sh` と `scripts/<script>` を置き換えるまで、後者が読む」)。旧版の `ralph-config.sh` はこの 5 つを読んでいないので、事実に合わせた。
      - instead は `ralph.local.toml` だけを名指す。
      - SKIP の行には「`1` だった人は `true` と書く」を足した。キーは bool なので、`= 1` と書くと Load のエラーになり、D7 で base まで full scope に倒れる。
    - テスト: 新しい `doctor_retired_script_env_test.go`(5 つのサブテスト、11 個を表の順で見る、doctor の実行)。B1 の「6 つの順」のテストは 11 個の順のテストに置き換えた。TestMain で 5 つを unset した。
    - 検証: cli `-run 'Retired|Doctor'` ok(51 秒)、golangci-lint 0 件、変異(表から 1 行を消す)は 3 本が落とした。
    - self-check は 5 項目とも clean。
  - S7 7fa65455(impl-276b2-s7)
    - quality-gates.md(KNOWN_DIFF なので root と写しを別々に): `[verify].base` に直し、設定の誤りで full scope になる 1 文を足した。
    - README: `ralph.local.toml` の節に 5 つのキーの段落を足した。書いたのは、値の形、local-only、base の 2 つの D7、ralph がないときと gate、撤去した 5 つの変数と doctor の warn。`config get` の行の「唯一の local-only のキー」の古い文も直した。
    - `config get --help`: 4 つのスクリプトがどのキーをどちらの関数で読むかを書いた。テストに 2 句を足した(既存の句は消していない)。
    - model-routing.md(root と写し): helper の説明を 10 キー・2 関数に直した。root の `defaults_sync_test.go` の説明も直した。
    - tech-debt の「Three test gaps」の (a) を閉じた(S2 e4794cd7)。
    - `verify.local.sh`: shellcheck の対象に `tests/lib/*.sh` を足した。
    - 検証
      - check-sync・check-skill-sync rc 0、gofmt 空、build rc 0、cli `-run 'Config|Help'` rc 0。
      - `HARNESS_VERIFY_MODE=static ./scripts/verify.local.sh` rc 0(OK 33)、test-agent-models 47/47、S7 の否定の grep rc 0。
      - 赤の確認: help を HEAD の版に戻すとテストが落ちる。
      - grep の残りは、どれもテストか「もう読まない」の説明。
    - self-check は 5 項目とも clean。
    - /sync-docs に回すもの
      - `docs/architecture/repo-map.md:62`(`ralph-config.sh` の説明に B2 のキーと `_or_fail` がない)。
      - tech-debt の同じ行の (c)(`zsh -f`)が今も当てはまるか。
  - S1〜S7 の実装はそろった。次は /self-review(cycle 1)。
  - S7 の追い a5a38aff(impl-276b2-s7): `docs/architecture/repo-map.md:62` の `ralph-config.sh` の説明を 2 つの関数と 10 キーの形に直した。README の値の形を `templates/base/ralph.toml` の注釈にそろえた。check-sync rc 0。
  - origin/main(#292)の取り込み 2cb62e4b: `docs/tech-debt/README.md` が衝突した。両方の行を残して直した(#292 の 2 行と、この branch の S1b の RESOLVED の行と分けた行)。build・vet・check-sync・check-skill-sync rc 0。
  - /self-review cycle 1(review-276b2、59674701): CRITICAL 0、HIGH 0、MEDIUM 0、LOW 5、`Merge: yes`。
    - L1(`plan-visual.sh` が `ralph-config.sh` の有無を見ずに source する)、L2(設定の誤りの文が原因を `ralph.local.toml` に決めつける)、L3(doctor の SKIP の案内が `0` の人に `= 0` を勧める)、L5(`ralph-config.sh` の終了コードの注釈)は、安く直せて実害があるので直した。
    - L4(800 行を超えたファイル 5 つ)は /sync-docs で tech-debt に行を足す。
  - LOW の直し 00c494cc(impl-276b2-fix1)
    - L1: `plan_visual_setting` の先頭に `[ -f … ] || return 0`。L2: 3 つの文をキー名と次の動きを残して「理由は上の行、どのキーの誤りでも全キーが失敗する」に直し、テストの句も合わせた。L3: 「1 のときだけ true、ほかの値は何も書かない」にし、値 0・yes・true の case を足した。L5: 注釈を整えた。
    - 検証: shellcheck rc 0、plan-visual 277/0、xreview-helpers 53/0、detect 113/0、run-verify-scope 25/0、ralph-config 272/0、secret-scan-branch 263/0、check-sync・check-skill-sync ok、cli `-run 'Retired|Doctor|Config'` ok。変異 2 つはテストが落とした。self-check は 5 項目とも clean。consult は「進めてよい」。
    - L1 の守りで、PATH 経由で裸の名前で呼んだときの source の失敗(S3 の consult の注意 1)もなくなった。
  - 直しの見直し(review-276b2、8c5270a4、report の「Fix review (00c494cc)」): L1・L2・L3・L5 は fixed、L4 は /sync-docs へ。新しい所見は LOW 1 件(L6: doctor の SKIP の warn は固定文で、先頭の「write the value」と末尾の「1 のときだけ」が逆向きに読める)。誤った書き方は勧めなくなったので直さず、PR の Known gaps に書く。判定は `Merge: yes` のまま。
  - /verify cycle 1(verify-276b2、643a92c3): pass。AC1〜AC6 に印。本物の binary で D7 を確かめた(壊れた browser と正しい verify.base で config get は exit 3、detect は full scope と `config_error:verify.base`、scan は strict で 3・既定で 0、`detect_base_branch` は sh・bash・dash・zsh で rc 1)。文書のずれ(AGENTS.md:89 の `LocalOnlyKeys`、L4 の tech-debt の行、spec の過去の例 I1)は /sync-docs へ。I2(既定の run-verify は config の誤りで scan を飛ばしても通る)と I3(`= 1` の誤りの文にキー名がない)は PR の Known gaps に書く。
  - /test cycle 1(test-276b2、6ef9f844): pass。run-test.sh は full fallback で rc 0(shell 45 本、Go 10 パッケージ、1,801 秒)。AC7 の run-verify.sh は rc 0(1,711 秒)。落ちたテストと打ち直しはなし。変異 7 個はすべてテストが落とした。Test gaps: ブラウザ自動検出の 2 件はこの Mac で SKIP なので PR の CI のログで走ったことを確かめる、shell のテストは偽の ralph だけで値を渡す、shell の coverage はない。
  - /sync-docs cycle 1(doc-maintainer): `docs/reports/sync-docs-2026-10-30-script-local-config.md`。直したのは `AGENTS.md` の `LocalOnlyKeys` の説明、`docs/architecture/repo-map.md` の Tests 節(`tests/lib/`)、`quality-gates.md`(と写し)の secret scan の base の 1 文、tech-debt の新しい 1 行(L4: 800 行を超えた 5 ファイル)と「Three test gaps」の (c) の確認の 1 文、Trigger に当たった 4 行(118・235・277・364)への持ち越しの注。(c) の `zsh -f` は今も当てはまる。5 つの変数の名前は、「もう読まない」の説明、doctor の表、テスト、過去の記録だけに残る。コードは触れておらず、plan 本文も触れていない(digest は cdd1e5ba1234 のまま)。
- [x] Review artifact created
- [x] Verification artifact created
- [x] Test artifact created
- [ ] PR created
