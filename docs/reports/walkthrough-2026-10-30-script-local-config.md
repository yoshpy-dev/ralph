# Walkthrough: script-local-config(スクリプトが読む 5 つの値を `ralph.local.toml` だけに書けるキーにし、5 つの環境変数を読まなくする)

- Date: 2026-11-01(ファイル名の 2026-10-30 は計画の日付)
- Plan: docs/plans/archive/2026-10-30-script-local-config.md(書いた時点では `docs/plans/active/` にあり、`/pr` が archive に移す)
- Issue: #276(`Closes`。#276 を 4 本に分けた 4 本目 B2。A1 は #287、A2 は #290、B1 は #291 で merge 済み)
- PR: #293(https://github.com/yoshpy-dev/ralph/pull/293)
- Branch: feat/script-local-config(この walkthrough の前の HEAD は f93326f6。26 commits で、merge commit は 1 つ。2cb62e4b が origin/main の #292 の取り込み)
- Base: 1cc84ffa(origin/main、「Merge pull request #292」= leader-sync-docs)。分岐元も同じ
- 読む範囲: `git diff origin/main...HEAD` は 55 files、+4080/-948。内訳は Go の本番コード 5 files(+195/-24)、Go のテスト 8 files(+1033/-392)、shell のスクリプト 13 files(root 7 + `templates/base/scripts/` の写し 6、+515/-163)、shell のテスト 9 files(+1398/-313)、skill の写し 4 files(+16/-16)、rules・文書・設定 9 files(+111/-40)、記録(計画・report 5 本・insight event)7 files(+812)。記録を除くと 48 files、+3268/-948。動きを決めるのは `internal/config/config.go`(+123/-8)、`scripts/ralph-config.sh`(+104/-38)、`scripts/plan-visual.sh`(+55/-26)、`internal/cli/doctor.go`(+41/-10)、`scripts/xreview-helpers.sh`(+36/-6)、`scripts/detect-changed-languages.sh`(+31/-4)、`scripts/run-verify.sh`(+17/-2)、`scripts/secret-scan-branch.sh`(+13/-5)、`/cross-review` の skill 4 本。テストの差分の大半は `tests/test-ralph-config.sh`(+366/-151)、`internal/config/defaults_sync_settings_test.go`(+403。B1 の照合を `defaults_sync_test.go` から移したもの)、`internal/config/local_script_keys_test.go`(+213)、`tests/test-plan-visual.sh`(+194/-39)、`tests/test-detect-changed-languages.sh`(+198/-19)、`tests/test-xreview-helpers.sh`(+190/-31)、`internal/cli/doctor_retired_script_env_test.go`(+182)
- 後工程: パイプラインは 1 回(`cycle-count.json` は 1、cross-review は 1/2)。plan advisory(codex)は HIGH 1 件で、D7 に入れた。self-review は CRITICAL・HIGH・MEDIUM なしで LOW 5(L1〜L5)、Merge 可。L1・L2・L3・L5 は 00c494cc で直し、L4 は sync-docs で tech-debt に入れた。直しの見直しで LOW が 1 件(L6)増え、直さず Known gaps に書いた。verify は pass(LOW 1 の V1 は sync-docs で直した。INFO 3 の I1〜I3 は直さない)。test は pass。sync-docs は 1 回。cross-review(codex)は所見 0 件。記録だけの修正の例外は使っていない

```
何を変えるか → キーと検証(S1・S1b) → helper(S2) → 4 つのスクリプト(S3・S4・S5) → /cross-review の skill → org の gate → doctor(S6) → 偽の ralph とテスト → 文書と help(S7・sync-docs) → レビューで足した直し → スライスと AC → 確かめたことと確かめていないこと
```

## 1. この PR が何か

- これまで:
  - スクリプトが読む個人用の値は、環境変数 5 つで渡した。
  - 環境変数は repo にも `ralph.toml` にも残らず、機械ごとの値を置く場所が決まっていなかった。
- この PR から:
  - 5 つの値は `ralph.local.toml` だけに書けるキー(local-only)で決める。`ralph.toml` に書くと `ralph` が設定の誤りとして断る。環境変数は読まない。

    | 値 | キー | 撤去した変数 | 既定 | 読むスクリプト |
    |---|---|---|---|---|
    | plan-visual のブラウザ | `[plan_visual].browser` | `RALPH_PLAN_VISUAL_BROWSER` | 空 | `plan-visual.sh`(`shot`) |
    | plan-visual の opener | `[plan_visual].opener` | `RALPH_PLAN_VISUAL_OPENER` | 空 | `plan-visual.sh`(`open`) |
    | cross-review と secret scan の base | `[standard].base_branch` | `RALPH_XREVIEW_BASE` | 空 | `xreview-helpers.sh` の `detect_base_branch` |
    | verify の changed scope の base | `[verify].base` | `RALPH_VERIFY_BASE` | 空 | `detect-changed-languages.sh` |
    | 手元の branch secret scan を飛ばす | `[verify].skip_branch_secret_scan` | `RALPH_VERIFY_SKIP_BRANCH_SECRET_SCAN` | `false` | `run-verify.sh` |

  - 読み出しは B1 の helper `scripts/ralph-config.sh` に集める。helper は `ralph config get` を打ち、読めないときの扱いは 2 つの関数で分ける(3 節)。
  - 設定の誤りで base が読めないときは、黙って自動の base に替えない(D7)。
  - `ralph doctor` は、撤去した 5 つの変数がまだ設定されていると warn を出す。
- 計画の判断(D1〜D7)のうち、読むときの前提になるもの:
  - D1 `Closes #276`。残す 3 つの環境変数(`COVERAGE_THRESHOLD`、`RALPH_TERRAFORM_INIT_VALIDATE`、`RALPH_PRIMARY_CLI`)の理由は B1 の D6 と PR #291 にある。issue の受け入れ条件「残した環境変数がすべて内部の受け渡し」を字句どおりには満たさない。読み替えを認めない場合は #276 を開け直す(PR 本文の先頭)。
  - D2 5 つとも local-only。あとで緩めるのは互換で、締めるのは非互換。skip は commit されると全員の scan が止まる。
  - D3 値の検証は使われ方ごとに変える(2 節)。値はスクリプトで二重引用符の中に入るので展開はされない。検証は、オプションに見える値や空白を先に断るためのもの。
  - D4 既定値が空のキーでは、exit 0 の空を正しい値として黙って返す。設定していない人に毎回 warn が出ないようにするため。
  - D5 テストは共通の偽の `ralph`(`tests/lib/fake-ralph.sh`)を PATH の先頭に置いて値を渡す。
  - D6 撤去した変数は読まず、doctor が warn を出すだけにする(B1 の D7 と同じ)。
  - D7 base の 2 つのキーは、設定の誤りで黙って既定値に戻らず、安全な側に倒す。計画助言の HIGH を受けて足した。設定の検証は全体で 1 つなので、別のキーの誤りでも `ralph config get verify.base` が exit 3 になる。このとき空の既定値に戻ると自動の base になり、ローカルの main では base が main 自身になって、changed scope が `no_changes` で成功し何も検査しない。
- release のノート(計画の Rollout): スクリプトは 5 つの変数を読まなくなる。代わりに `ralph.local.toml` に書く。`ralph.toml` には書けない。doctor がまだ設定されている変数に warn を出す。スクリプトがこの値を読むには新しい binary が要る。
- 戻し方: この PR を revert する。設定ファイルの形式はキーが増えるだけ。revert した ralph が新しいキーを知らないキーとして無視するかは推測で、未確認(計画の Rollout)。

## 2. 設定のキーと検証(S1 8bb68fe2、S1b 67d9d80b)

- 型と既定(`internal/config/config.go`):
  - `PlanVisualConfig`(`Browser`、`Opener`)、`VerifyConfig`(`Base`、`SkipBranchSecretScan`)、`StandardConfig.BaseBranch`。`Config` に `PlanVisual` と `Verify` を足した。
  - `Default()` はどれも空か `false`。
- local-only(`internal/config/local.go`):
  - `LocalOnlyKeys()` に 5 つ足し、`org.state_dir` と合わせて 6 つにした。B2 は `checkLocalOnlyKeys` の仕組みを変えていない。キーを足しただけで、`ralph.toml` に書くと Load がパスと `ralph.local.toml` を名指してエラーにする(A2 からの形)。
  - キーの大文字小文字は B1 と同じ規則(go-toml の decoder と同じ)で引く。
- 検証(D3):
  - `validateLoaded` の末尾が `validateScriptValues` を呼ぶ。順は `[standard].base_branch`、`[plan_visual].browser`、`[plan_visual].opener`、`[verify].base`。エラーはキーと値を名指す。個人用のファイルの値も同じ検査を通る。

    | キー | 通す | 断る |
    |---|---|---|
    | `plan_visual.browser` | 空、`none`、絶対パス(空白を含んでよい) | 相対パス、`~/`、制御文字 |
    | `plan_visual.opener` | 空、`none`、1 語 | 空白・制御文字、先頭の `-`(`isScriptWord`) |
    | `standard.base_branch` | 空、`baseBranchPattern`(`[A-Za-z0-9._/-]+`) | 先頭の `-`、`..` |
    | `verify.base` | 空、1 語(`HEAD~1`、`HEAD^`、`origin/main`) | 空白・制御文字、先頭の `-`(`isScriptWord`) |
    | `verify.skip_branch_secret_scan` | bool | bool 以外。検査は `toml.Unmarshal` で、`validateScriptValues` には載っていない(Known gaps の I3) |

  - 見る所: `browser` が絶対パスだけなのは、`plan-visual.sh` が `[ -f "$browser" ]` で値をそのまま見るので、`~/` は字面のままでは必ず失敗するため。`verify.base` が `HEAD~1` を通すのは、文書と過去の報告がこの形を案内しているため(consult で直した点)。
- `ralph config get`: reflection で引くので配線は足していない。`--help` の文は 9 節。
- template(`templates/base/ralph.toml`): 5 つは注釈で書いた。`[plan_visual]` と `[verify]` は中身が注釈だけの表。local-only なので、有効な行で書くと `ralph.toml` として読めなくなる。注釈に「`ralph.local.toml` にコピーして使う」と書いた。
- 照合:
  - B1 の `commentedTemplateProbe` の対象を、`[standard]`・`[reviewer]` に `[plan_visual]`・`[verify]` を加えた 4 つの表に広げた。各キーが template の注釈と `Default()` の両側で比べられることを `TestCommentedTemplateDrift_EachKeyIsComparedOnBothSides` と `TestTemplateDrift_EveryStandardReviewerPlanVisualAndVerifyKeyIsChecked` が固定する(後者は B1 の `…EveryStandardAndReviewerKeyIsChecked` を改名)。
  - S1b は B1 から続く照合(`parseShellSettingDefaults`、`commentedTemplateDrift`、`checkSkillSettings` などと 2 つのテスト)を、`defaults_sync_test.go` から `defaults_sync_settings_test.go` に中身を変えずに移した移動だけのスライス。S1 で `defaults_sync_test.go` が 977 行になり、tech-debt の「800 行を超えた」行の Trigger に当たったため。`go test -list` は移す前後で同じ 104 本、移した範囲は `cmp` で一致した。S2 で、移動先の期待値を `shellSettingWant` にまとめ、5 キーを足した。
- テスト:
  - `internal/config/local_script_keys_test.go`(新規): `TestLocalOnlyKeys_HoldsTheScriptKeys`、`TestLoad_ScriptLocalOnlyKeysInRalphToml`(5 キー × 大文字小文字の綴り、サブテスト 50)、`TestLoad_LocalScriptKeys`、`TestLoad_LocalScriptValueValidation`(D3 の通る例と断る例、サブテスト 59)。大文字小文字の綴りは、本物の `toml.Unmarshal` が読む綴りだけを使う。
  - `internal/cli/config_test.go`: `TestConfigGet_PrintsTheScriptLocalOnlyValues`(値、空白つきのブラウザ、`HEAD~1`、`RALPH_LOCAL_CONFIG=off` で既定値)、`TestConfigGet_InvalidScriptLocalOnlyValueExits3`(別のキーを聞いても exit 3)、`TestConfigGet_RalphTomlThatSetsAScriptLocalOnlyKeyExits3`。
- 検証: config と cli の関係するテストが rc 0。変異 3 つをテストが落とした(S1 の報告)。

## 3. helper(S2 e4794cd7)

- `scripts/ralph-config.sh`(と `templates/base/scripts/` の byte 一致の写し)の関数は `ralph_setting_default`、`_ralph_setting_get`、`ralph_setting`、`ralph_setting_or_fail` の 4 つ。source しても変数は設定せず、環境変数は読まない。
- 表(`ralph_setting_default`)は 5 行から 10 行になった。既定値が空のキーは `''`、bool は `'false'`。行の形(`<key>) printf '%s\n' '<value>' ;;`)を `parseShellSettingDefaults` が正規表現で読んで `Default()` と照合し、`tests/test-ralph-config.sh` がこの関数を呼ぶ。行の形を変えるときは両方のテストを変える。
- 分け方は `_ralph_setting_get` の 1 か所。`command ralph config get <key>` の終了コードで決め、2 つの関数は理由の扱いだけが違う。

  | ralph の結果 | `ralph_setting` | `ralph_setting_or_fail` |
  |---|---|---|
  | exit 0 で値あり | その値 | その値 |
  | exit 0 で空、既定値が空のキー(D4) | 空を黙って返す | 空を黙って返す |
  | exit 0 で空、既定値があるキー | 失敗として扱う(下の行) | 失敗として扱う(下の行) |
  | 127(ralph が PATH にない)、exit 1 で `unknown command "config"`(古い ralph) | 既定値を黙って返す | 既定値を黙って返す |
  | exit 2・3・その他、`mktemp -d` の失敗 | stderr に 1 行(`…; using the default <値>` か `using the empty default`)、既定値を出し、戻り値 0 | stderr に 1 行(`…; not using the default`)、stdout は空、戻り値 1(D7) |
  | 表にないキー | stderr に 1 行、何も出さず、戻り値 2 | 同じ |

- 見る所:
  - `ralph_setting` の約束(どの場合も戻り値 0、`set -e` の呼び出し側が止まらない)は B1 のまま。D7 のために `ralph_setting` 自体を失敗で 0 以外にする案は退けた(B1 の skill と `set -e` の呼び出し側が壊れる)。
  - 黙って既定値に戻るのは 127 と古い ralph だけ。このとき個人用のファイルは読めないので、設定の誤りは起きない。
  - 「exit 0 の空を値にするのは既定値が空のキーだけ」の条件(`_ralph_setting_get`)。8 節の変異 M5・M6 がこの条件の両側を落とす。
  - 変数は `_rs_` で始まる(POSIX sh に local がないため)。一時ディレクトリは通常の道で必ず消し、trap は置かない(呼び出し側の trap を壊さない)。
  - ファイル頭の使い分けの文: 既定値が誤りを隠さないキーは `ralph_setting`、隠すキー(base)は `ralph_setting_or_fail`。
- テスト(`tests/test-ralph-config.sh`、272 項目): 共通の偽の ralph で、表の 10 キー、上の表の各行、`mktemp` が失敗するケースを打つ。全キーを回すのは `/bin/sh` だけで、bash・dash・zsh は代表の 3 キー(1 ケース 1 shell では 243 秒かかり、約 120 秒に収めたため)。mktemp の失敗ケースは、tech-debt の「Three test gaps of standard-flow-config」の (a) の Trigger に当たったので足し、その (a) を閉じた。
- 検証: shellcheck rc 0、272/272、`test-codex-exec-invocation.sh` 160/160、check-sync rc 0。S2 の変異は shell 7 つ、Go 3 つともテストが落とした。

## 4. 4 つのスクリプト(S3 674b193c、S4 7803a530、S5 5f201546)

読み手ごとに、何を読み、どこで読み、読めなかったときどうなるかを並べる。

| スクリプト | キーと関数 | ralph を打つ場所 | 読めなかったとき | `ralph-config.sh` がないとき |
|---|---|---|---|---|
| `plan-visual.sh`(`plan_visual_setting`) | `plan_visual.opener`(`open` で 1 回)、`plan_visual.browser`(`shot` で 1 回)。`ralph_setting` | スクリプトの隣の `ralph-config.sh` を source し、スクリプトの親(repo の root)で打つ。呼んだ人の cwd に依らない | stderr に 1 行、空の既定値(自動検出)。exit 0 のまま | 何も読まず ralph も打たない(00c494cc の L1 の直し) |
| `xreview-helpers.sh`(`detect_base_branch`) | `standard.base_branch`。`ralph_setting_or_fail` | `git rev-parse --show-toplevel` の root で、subshell の中で source して打つ | stdout 空、戻り値 1、stderr に 2 行。自動の段に進まない(D7) | その段を黙って飛ばし、origin の既定 branch → main → master |
| `secret-scan-branch.sh` | 手元の base は `detect_base_branch`。`GITHUB_BASE_REF` があれば先に使い、helper も ralph も打たない(CI に ralph がない) | 上と同じ | `cannot_scan`: `--strict` は exit 3、既定のモードは報告して exit 0 | 上と同じ |
| `detect-changed-languages.sh` | `verify.base`。`ralph_setting_or_fail` | git の toplevel で、subshell の中で source して打つ | `fallback_full "config_error:verify.base"`: full scope、`no_changes` は出さない(D7) | 読まず、自動の base |
| `run-verify.sh` | `verify.skip_branch_secret_scan`。`ralph_setting` | `HARNESS_VERIFY_MODE` が `static` か `all` の scan の分岐の中だけ。`./scripts/ralph-config.sh` を source | `false` に倒れ、scan が走る | scan が走る |

- 共通の見る所:
  - 5 つの変数は、スクリプトのどこからも読まれない(S7 の否定の grep が 0 件)。
  - `ralph-config.sh` を subshell の中で source する形(`CDPATH='' cd -- "$root" && . ./scripts/ralph-config.sh && …`)。呼び出し側の shell に関数と `_rs_` の変数が残らない(`tests/test-xreview-helpers.sh` の 3k が固定)。`xreview-helpers.sh` は source される側で、POSIX sh では自分のパスを知る手段がないため、git の toplevel を「このファイルの親」の代わりにする(ファイル頭に書いた)。
  - 読む場所が 3 通りある(スクリプトの隣、git の toplevel、cwd の `./`)。self-review の L1 はこのずれの 1 つ(`plan-visual.sh` だけ `ralph-config.sh` の有無を見なかった)を指し、00c494cc で `[ -f … ] || return 0` を足した。
  - 設定の誤りは全キーに効く。ブラウザの誤りで `verify.base` も exit 3 になる。そのため base の 2 つのエラー文は、原因を `ralph.local.toml` に決めず「理由は上の行。どのキーの誤りでも全キーが失敗する」の形にした(L2)。
- `plan-visual.sh`: `none` と空の意味は env のときと同じ(空は自動検出、`none` は使わない)。warn・ファイル頭・`--help` はキー名に直した。`lint`・`digest`・usage では ralph を打たない(呼び出し記録で確かめた)。
- `secret-scan-branch.sh`: 変えたのは base を決める所と、ファイル頭の exit code の説明。
- `detect-changed-languages.sh`: 設定の誤りで full scope に倒す理由は、ファイル頭の長い注釈(D7)に書いてある。自動の base は、リモートの既定 branch がない main の上で HEAD 自身になり、`no_changes` になりうる。
- `run-verify.sh`: このスクリプトが repo の root で動く前提(自分で `./scripts/*` を打つ、ラッパー・CI・gate の cwd)を注釈に書いた。
- 検証:
  - S3: `test-plan-visual.sh` 272 件通過(SKIP 1)、変異 3 つをテストが落とした。
  - S4: `test-xreview-helpers.sh` 53/53、`test-secret-scan-branch.sh` 263/263、変異 7 つをテストが落とした。この repo の `secret-scan-branch.sh --strict` は PATH の v6.0.0 で rc 0。
  - S5: `test-detect-changed-languages.sh` 113/113、`test-run-verify-branch-secret-scan.sh` 50/0、`test-run-verify-concurrency.sh` 37/37、`test-run-verify-scope.sh` 25/25、変異 3 つをテストが落とした。

## 5. `/cross-review` の skill(S4)

- 手順 4 の 4 行を直した。skill の 4 つの写し(`.claude/skills`、`.agents/skills`、`templates/base/` の 2 つ)は同じ内容で、`check-skill-sync.sh` が揃いを見る。
  - base を決める行: `BASE=$(detect_base_branch) || exit 1`。解決順を `[standard].base_branch` → origin の既定 branch → main か master に直した。設定の誤りでは自動の段に進まず、理由を stderr に書いて 0 以外を返し、`|| exit 1` でその Bash 呼び出しを終える。そこで止まり、stderr の行(キー名と ralph のエラー)をユーザーに見せる。
  - diff が空かを確かめる行: 同じ Bash 呼び出しの中で base を決め直す(Bash は呼び出しの間で状態を持たない)。`base=` の行を出さずに exit 1 で終わった呼び出しは、上の失敗として止まる。
  - codex の説明の行と起動コマンドの行: 起動前に base を決め直す所に `|| exit 1` を置いた。失敗すると codex は起動せず、`codex rc=` の行も出ない。
- 見る所: `$(…)` の失敗は代入文の終了コードに出る。`|| exit 1` がないと、空の `$BASE` のまま `git diff "$BASE"...HEAD --quiet` が進む。skill 自身が書くとおり、空の `$BASE` では diff が常に空に見えるので、設定の誤りが「empty diff」の skip として記録される。
- テストでは固定していない。`detect_base_branch) || exit 1` の文字列を見る Go のテストも shell のテストも、grep では見つからなかった。確かめは次のとおり。
  - verify が SKILL.md から取り出した手順 4 のコマンドを、壊れた設定で打った(rc 1、`base=` の行なし)。
  - `check-skill-sync.sh` と `check-sync.sh` が 4 つの写しの揃いを見る。

## 6. org の gate(S5)

- gate は、スクリプトに `RALPH_LOCAL_CONFIG=off` を渡して個人用のファイルを読ませない(`gateLocalConfigEnv`、`gateScriptEnv` が足す)。そのため、5 つのキーは gate のスクリプトでは既定値になる。B2 は振る舞いを変えていない。
- `internal/org/gate_run.go` は注釈だけを直した。`gateStrippedEnvPrefixes`(`RALPH_VERIFY_` と `HARNESS_VERIFY_` を落とす)の理由に挙げていた `RALPH_VERIFY_BASE=HEAD` は、もう読まれない変数なので、`RALPH_VERIFY_SCOPE=changed` の例に替えた。接頭辞を落とす処理は scope と roots のために残る。
- テスト(`internal/org/gate_runverify_test.go`): `TestGate_RealRunVerify_FullScopeRunsTheLanguageVerifier` の場面を、偽の ralph(`gateRalphStub`)に替えた。偽の ralph は `RALPH_LOCAL_CONFIG` が `off` でなければ `verify.base` に `HEAD`、`verify.skip_branch_secret_scan` に `true` を答え、`off` なら既定値を答える。呼び出しごとに `<key>:<RALPH_LOCAL_CONFIG>` を記録する。
  - 対照の実行(gate を通さない): 個人用の値で Go の検証と scan を飛ばして通る。
  - gate の実行: ralph に `verify.skip_branch_secret_scan` を `off` で 1 回だけ聞いて `false` を得て、scan が走る。full scope なので `verify.base` は聞かない。
  - `gateCallerVerifyEnv` から、もう読まれない `RALPH_VERIFY_BASE` と skip の変数を外した。
- 同じ約束を cli 側でも固定する: `TestConfigGet_PrintsTheScriptLocalOnlyValues` の後半が、`RALPH_LOCAL_CONFIG=off` で 5 キーが既定値を出すことを見る。
- 見る所: gate は「個人用のファイルを読まない」ことで守られている。base を gate に効かせる変更は入っていない。gate を本物の feature org で打つ確かめはしていない(12 節)。

## 7. doctor(S6 a338dc66)

- `retiredEnvs`(`internal/cli/doctor.go`)に 5 行を足した。新しい `retiredScriptEnv(name, key, script, more)` で作る。B1 の `retiredSettingEnv` の `effect` は「skill と `scripts/ralph-config.sh` が読む」と書くので、B2 の 5 行は変数ごとに実際に読んでいたスクリプトを名指す形にした(旧版の `ralph-config.sh` はこの 5 つを読んでいない)。
  - effect: 「`ralph upgrade` が `scripts/ralph-config.sh` と `scripts/<script>` を置き換えるまで、後者が読む」。binary は元からこの変数を読まない。
  - instead: 「`<キー>` を `ralph.local.toml`(主 worktree の root、git 無視)に書く」。キーは local-only なので `ralph.toml` は案内しない。
- `RALPH_VERIFY_SKIP_BRANCH_SECRET_SCAN` の行だけ、旧版の動きを添える。旧版は値が `1` のときだけ scan を飛ばした。そのため文は「`1` の人は `true` と書く。ほかの値は飛ばしていないので何も書かない(キーは `true`/`false` だけ)」の固定文にした。値を問わず「値を書く」と案内すると、`0` や `yes` の人が `= 0` と書いて Load が断り、D7 で base まで失敗するため(self-review L3)。固定文は読む順で迷う(L6、Known gaps)。
- 空の値と未設定では何も出さない。
- テスト(`internal/cli/doctor_retired_script_env_test.go`、新規): `TestCheckRetiredEnv_ScriptVariables`(5 行)、`TestCheckRetiredEnv_ScanSkipOtherThanOne`(`0`・`yes`・`true`)、`TestCheckRetiredEnv_AllElevenInTableOrder`(B1 の `AllSix` を置き換えた。`doctor_retired_standard_env_test.go` から 26 行が消えたのはこのため)、`TestRunDoctorOpts_RetiredScriptEnv`(設定あり・空・未設定の 3 通り)。`main_test.go` の `TestMain` が 5 つを unset する。表から 1 行を消す変異を 3 本が落とした。

## 8. 偽の ralph とシェルのテスト(S2、S3〜S5)

- `tests/lib/fake-ralph.sh`(新規 118 行): キーごとに値と終了コードを返す偽の `ralph`。
  - 関数は `fake_ralph_install`、`_value`、`_fail`、`_old`、`_calls`、`_reset`、`_clear_calls`。答え・呼び出し記録・古い ralph の印は `<bindir>/ralph.fake/` に置き、環境変数を読まない。環境を消して起動するスクリプトにも答えが届く。
  - 答えのないキーは本物と同じ文で exit 2、`config get` 以外の呼び出しは exit 99。`_old` の stderr は、手元の v6.0.0 と同じ 1 行(`unknown command "config" for "ralph"`)。ralph がない状態(127)は、bindir を PATH に置かないことで作る。
  - 呼び出し記録は `<pwd -P>|<引数>`。テストは「どこで何回読んだか」を見る。
  - 見る所: `fake_ralph_value` は `_fake_ralph_answer` が置いた `_fr_key` を暗黙に読む(self-review の Notes。呼ぶ順が固定なので今は壊れない)。
- 書き直したテストは 8 本(`test-ralph-config.sh` と、plan の D5 の 7 本)。env で値を渡す形をやめ、偽の ralph を PATH の先頭に置く。`test-plan-visual.sh`、`test-xreview-helpers.sh`、`test-secret-scan-branch.sh`、`test-detect-changed-languages.sh`、`test-run-verify-branch-secret-scan.sh` には、撤去した変数を設定しても効かない行と、呼び出し記録の行がある。`test-run-verify-concurrency.sh` と `test-run-verify-scope.sh` の変数の扱いは、unset の行の直しだけ。

  | テスト | 結果 | 見ているもの |
  |---|---|---|
  | `tests/test-ralph-config.sh` | 272/272 | 3 節 |
  | `tests/test-plan-visual.sh` | 277 通過、SKIP 1 | opener と browser の値・空・`none`、ralph なし、古い ralph、exit 3。env が効かない行 |
  | `tests/test-xreview-helpers.sh` | 53/53 | Test 3(3a〜3k): `[standard].base_branch` が勝つ、ディレクトリを下りても同じ root、exit 2・3 で戻り値 1 と stdout 空、ralph なしと古い ralph で自動の段 |
  | `tests/test-secret-scan-branch.sh` | 263 通過 | `run_with_base_branch` で base を渡す。`GITHUB_BASE_REF` があると ralph を打たない(呼び出し記録が空)、設定の誤りで `--strict` が exit 3 |
  | `tests/test-detect-changed-languages.sh` | 113/113 | `[verify].base` の値・空、exit 2・3 で `config_error:verify.base` の full、ralph なし、`ralph-config.sh` のない repo |
  | `tests/test-run-verify-branch-secret-scan.sh` | 50/0 | `true` のときだけ飛ばす。`false`・127・古い ralph・exit 3・ファイルなしでは scan が走る |
  | `tests/test-run-verify-concurrency.sh` | 37/37 | 撤去した変数の unset の行を直した分だけ |
  | `tests/test-run-verify-scope.sh` | 25/25 | D7 の回帰: main の上で、空なら `no_changes`(対照)、exit 3 なら full になり、わざと失敗させる言語の検証が走って run が落ちる |

- `scripts/verify.local.sh`: shellcheck の対象に `tests/lib/*.sh` を足した。
- 変異(test の回、`git archive HEAD` の写しに 1 か所ずつ入れる): M1 D7 の `fallback_full` を外す、M2 `RALPH_VERIFY_SKIP_BRANCH_SECRET_SCAN=1` で飛ばす形に戻す、M3 `detect_base_branch` が失敗で自動の段に進む、M4 `GITHUB_BASE_REF` があっても helper を打つ、M5 と M6 helper の空の扱いの反転、M7 `plan-visual.sh` が `RALPH_PLAN_VISUAL_OPENER` を先に読む。7 個ともテストが落とした。

## 9. 文書と help(S7 7fa65455・a5a38aff、sync-docs d90f63a3)

- `ralph config get --help`(`internal/cli/config.go`): 4 つのスクリプトがどのキーをどちらの関数で読むかを書いた(`ralph_setting_or_fail` は 127 と古い ralph のときだけ既定値、それ以外の失敗は stderr に 1 行を出して失敗し、スクリプトが止まるか full scope に倒れる)。`config_test.go` の help のテストに 2 句を足して固定した(既存の句は消していない)。
- README: `ralph config get` の行の「唯一の local-only のキー」の古い文を直した。`ralph.local.toml` の節に 5 キーの段落を足した(値の形、local-only、base の 2 つの D7、ralph がないときと gate、撤去した 5 つの変数と doctor の warn)。値の形は `templates/base/ralph.toml` の注釈にそろえた。
- `docs/quality/quality-gates.md` と `templates/base/docs/quality/quality-gates.md`(KNOWN_DIFF なので別々に): changed scope の base を `[verify].base` に直し、読めないと full scope になる 1 文を足した。sync-docs で、手元の secret scan の base が読めないと "cannot scan"(`--strict` は exit 3)で、CI は `GITHUB_BASE_REF` が先に決まって ralph を打たない、の 1 文も足した。
- `.claude/rules/ralph/model-routing.md`(root と template、KNOWN_DIFF なので別々に): helper の説明を 10 キー・2 関数に直した。root の `defaults_sync_test.go` の説明には、移した照合の置き場所(`defaults_sync_settings_test.go`)を足した。
- `AGENTS.md`(`LocalOnlyKeys` の説明に 5 つの script key を足した。verify の V1)、`docs/architecture/repo-map.md`(`ralph-config.sh` の説明と、`tests/lib/` の行)。
- `docs/tech-debt/README.md`:
  - 800 行を超えた 5 ファイルの新しい行(self-review の L4。`tests/test-detect-changed-languages.sh`、`tests/test-plan-visual.sh`、`tests/test-secret-scan-branch.sh`、`internal/config/config.go`、`internal/cli/doctor.go`)。
  - B1 の `defaults_sync_test.go` の行を S1b で閉じ、`org_feature_test.go` の分は残した。「Three test gaps」の (a) を閉じ、(c)(`zsh -f` で動かない)は今も当てはまることを確かめた。
  - Trigger に当たった 4 行への持ち越しの注。
- 触っていない文書: `docs/specs/2026-10-07-org-multi-org-director.md` の 7b の決定の記録は、gate が `RALPH_VERIFY_` を落とす例に `RALPH_VERIFY_BASE=HEAD` を挙げる。過去の決定の記録なので残した(verify の I1)。

## 10. レビューで足した直し(00c494cc)

self-review の LOW のうち、安く直せて実害があるものを 1 commit で直した。

- L1 `plan_visual_setting` の先頭に `ralph-config.sh` の有無の確かめを足した。部分的に写した repo で `open` が path を出したあと落ちる、を防ぐ。
- L2 設定を読めなかったときの 3 つの文(`xreview-helpers.sh`、`detect-changed-languages.sh`、`secret-scan-branch.sh`)を、原因を `ralph.local.toml` に決めない形に直した。固定するテストの句も 4 つのファイルで合わせた。
- L3 doctor の `RALPH_VERIFY_SKIP_BRANCH_SECRET_SCAN` の行を、`1` のときだけ `true` と書く文にした(7 節)。`0`・`yes`・`true` の case を足した。
- L5 `ralph-config.sh` の終了コードの一覧の書き方を直した(コメントだけ)。
- L4 は直さず、sync-docs で tech-debt の行にした。L6 は直さず、Known gaps に書いた。

## 11. スライスと受け入れ条件(AC)

| スライス | commit | 当たる AC | 内容 |
|---|---|---|---|
| S1 | 8bb68fe2 | AC1 | 5 つのキー、local-only、D3 の検証、template の注釈、`ralph config get` のテスト |
| S1b | 67d9d80b | — | B1 と S1 の照合を `defaults_sync_settings_test.go` に移す(移動だけ) |
| S2 | e4794cd7 | AC2 AC3b | 表に 5 キー、D4、`ralph_setting_or_fail`、`fake-ralph.sh`、`test-ralph-config.sh` |
| S3 | 674b193c | AC3 AC4 | `plan-visual.sh` |
| S4 | 7803a530 | AC3 AC3b AC4 | `detect_base_branch`、`secret-scan-branch.sh`、`/cross-review` の skill |
| S5 | 5f201546 | AC3 AC3b AC4 | `detect-changed-languages.sh`、`run-verify.sh`、gate の注釈とテスト |
| S6 | a338dc66 | AC5 | doctor の 5 行 |
| S7 | 7fa65455 a5a38aff | AC6 | README、`quality-gates.md`、`model-routing.md`、`config get --help`、repo-map |
| 直し | 00c494cc | — | self-review の L1・L2・L3・L5 |
| sync-docs | d90f63a3 | AC6 | `AGENTS.md`、repo-map、`quality-gates.md`、tech-debt |

AC1〜AC6(AC3b を含む)は、verify が HEAD から build した binary と一時 repo で打った probe で満たした(AC4 の gate は、コード・テストの存在・`RALPH_LOCAL_CONFIG=off` の probe まで)。AC7(build、gofmt、vet、golangci-lint、shellcheck、`check-sync`、`check-skill-sync`、`run-verify.sh`)は個別の commit に当たらない。verify が静的の部分、test が `run-verify.sh` の既定モードを rc 0 で通した。

## 12. 確かめたことと確かめていないこと

- 確かめたこと:
  - `./scripts/run-static-verify.sh` rc 0(verify。gofmt、go vet、staticcheck、golangci-lint 0 件、shellcheck `--severity=warning`、`check-sync`(DRIFTED 0、KNOWN_DIFF 6)、`check-skill-sync`(14 本)、`check-template-purity`、tech-debt の plan 参照、branch の secret scan)
  - `./scripts/run-test.sh` rc 0(test。1,801 秒。shell 45 本、Go 10 パッケージ。`internal/org` 1,361 本、`internal/cli` 932 本、`internal/config` 104 本)。`./scripts/run-verify.sh` の既定モードも rc 0(1,711 秒、AC7)。落ちたテストと打ち直しはなし
  - 変異 7 個(8 節)はすべてテストが落とした
  - カバレッジ(test): `validateScriptValues`・`isScriptWord`・`retiredScriptEnv`・`checkRetiredEnv`・`LocalOnlyKeys`・`Default` は 100%、`internal/config` は全体で 97.2%。シェルには計測の道具がない
  - HEAD の binary と一時 repo の probe(verify): ファイルがないとき 4 つが空で skip が `false`。個人用の値が効くこと。5 つのキーを `ralph.toml` に書くと exit 3。壊れた `browser` と正しい `[verify].base` で `config get` が exit 3、detect が full scope(`config_error:verify.base`)、scan が `--strict` で exit 3、`detect_base_branch` が sh・bash・dash・`zsh -f` で rc 1。`RALPH_LOCAL_CONFIG=off` の既定値。doctor の 4 回の実行の diff。cross-review の手順 4 のコマンドが rc 1 で `base=` を出さないこと
  - 実物の古い ralph(PATH の v6.0.0)の道: test の 2 回の run で、4 つのスクリプトが helper の行を stderr に出さずに、scan が自動の base `origin/main` で走った(AC4 の「古い ralph では今の既定の振る舞いのまま」)
  - cross-review(codex、`gpt-6-astra`、`xhigh`、read-only)は所見 0 件。設定は HEAD から build した ralph で読んだ(PATH の 6.0.0 には `config get` がない)
- 確かめていないこと・残っていること:
  - Linux の CI で、ブラウザ自動検出のテスト 2 件(6 行)が SKIP でなく通ること。`tests/test-plan-visual.sh` のこの 2 件は、`/Applications` の Chrome を検出して本物のブラウザを起動してしまうため、手元の Mac では SKIP になる。`/Applications` を隠した写しでは 283 件が通った。PR の CI のログで確かめる
  - 本物の binary でシェルのテストを打つこと。シェルのテストはどれも偽の ralph で値を渡している。本物の binary での確かめは verify の probe の分だけ
  - `ralph org gate` と `ralph org regate` を本物の feature org で打つこと(gate の環境はコード、テストの存在、probe まで)
  - `shot` をブラウザ入りで打つ probe(`[plan_visual].browser` を設定して撮る)。grep とテストの行で代えた
  - Codex の driver で `/cross-review` を動かす道。`.agents/skills/` の写しが文字列として一致することまで
  - `/cross-review` の `|| exit 1` を固定するテストはない(5 節)
  - `tests/test-ralph-config.sh` を `zsh -f` で直接動かすこと。runner が単語を sh のように分けるため動かない。tech-debt の既存の項目 (c) で、今も当てはまる
  - Known gaps(この PR では直していない。PR 本文のものと同じ):
    - L6 doctor の `RALPH_VERIFY_SKIP_BRANCH_SECRET_SCAN` の warn は固定文で、前半の「値を書く」と後半の「`1` のときだけ」が逆向きに読める。誤った書き方は勧めていない
    - I2 既定のモードの `run-verify.sh` は、設定の誤りで branch secret scan が "cannot scan" になっても通る。`--strict` の `/pr` と CI は止まる
    - I3 `= 1` のように型の違う値の誤りの文に、ドット付きのキー名が出ない
    - 関数のない古い `scripts/ralph-config.sh` が残る repo では、新しいスクリプトの base の読み出しが rc 127 で失敗し、設定を読めなかったという文を出す。スクリプトと一緒に `ralph-config.sh` も新しい版にそろえる必要がある
    - ブラウザ自動検出のテスト 2 件は、手元の Mac で SKIP になった(上の 1 つ目)
    - シェルのテストは、すべて偽の ralph で値を渡している(上の 2 つ目)
- 注意して読んでほしい所:
  - D7(3 節、4 節): base の 2 つだけを失敗にした理由は、黙って自動の base に替えると secret scan の範囲や verify の scope が意図と変わるため。ほかの 3 つは既定値に戻っても危なくないので、stderr に 1 行知らせて続ける。`ralph_setting_or_fail` が黙って既定値を返すのは 127 と古い ralph だけ
  - `_ralph_setting_get` の「exit 0 の空」の条件(3 節)。既定値が空のキーだけ値として扱い、ほかのキーでは失敗にする
  - 設定の誤りが全キーに効くこと(4 節)。ブラウザの誤りで verify の scope と cross-review の base が止まる。エラー文が原因を決めつけないのはこのため
  - `detect-changed-languages.sh` の full scope への倒れ方(4 節)。`no_changes` の経路を塞いでいる
  - gate が個人用のファイルを読まないことで守られていること(6 節)
  - PATH の ralph は 6.0.0 で `config get` がないので、新しい binary を入れるまで 5 つのキーは書いても効かず、既定値(自動検出)で動く。B1 の値も同じ
  - 上限の引き上げは B1 の `cycle-count.json` の `cap` で行う。この PR では引き上げていない(cycle 1/2 で終わった)
