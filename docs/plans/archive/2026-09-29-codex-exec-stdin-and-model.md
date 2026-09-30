# codex-exec-stdin-and-model

- Status: PR created (#192), awaiting CI and merge
- Owner: Claude Code
- Date: 2026-09-29
- Related request: `/plan` の Codex plan advisory と `/cross-review` の `codex exec review` を Bash ツールから起動すると、codex が「Reading additional input from stdin...」で stdin の EOF を待ち続けて止まることがある(2026-09-17 の #153 で約 4 時間)。回避策の `</dev/null` は毎回手で付けており、skill 本体には入っていない。また shell alias を迂回して起動すると、`.codex/config.toml` の `model` とユーザー config の `model_reasoning_effort` の組み合わせで API が 400 を返して review が中断した(2026-09-18、#162)。`-m` と `-c model_reasoning_effort=...` を明示すれば確実。issue #184
- Related issue: 184
- Type: fix
- Branch: fix/codex-exec-stdin-and-model

## Objective

`/plan` と `/cross-review` の skill 本文(4 面)の codex の呼び出しを、stdin を閉じ(`</dev/null`)、model と reasoning effort を `scripts/ralph-config.sh` の設定値から明示する形にし、その形が崩れたらテストが落ちるようにする。

## 調査で確認したこと(2026-09-29、main 6e26aaf)

- 呼び出し箇所は 2 つ: `.claude/skills/plan/SKILL.md` の step 11.c(`codex exec --sandbox read-only "<adversarial prompt> docs/plans/active/<plan-file>"`)と `.claude/skills/cross-review/SKILL.md` の step 4(`codex exec review --base "$BASE"`)および末尾の CLI execution modes 表(同じ形)。どちらも `</dev/null` がなく、model / effort も付いていない。
- 4 面の関係: root の `.claude/skills/` が source。`.agents/skills/` は `scripts/sync-skills.sh` が frontmatter を変換して生成し(`CLAUDE_ROOT` / `CODEX_ROOT` で template 側にも使える)、`scripts/check-skill-sync.sh` が drift を検査する。`templates/base/.claude/skills/` は root と byte 一致で `scripts/check-sync.sh` が検査する(`release` skill だけ repo 専用として除外)。
- `scripts/ralph-config.sh`(template と byte 一致)は `RALPH_CLAUDE_REVIEWER_MODEL="${RALPH_CLAUDE_REVIEWER_MODEL:-opus}"` を持ち、`internal/config/defaults_sync_test.go` が cross-review SKILL.md の `${RALPH_CLAUDE_REVIEWER_MODEL:-opus}` の fallback と shell の既定の一致を検査している(`parseShellDefaults` は `^NAME="${NAME:-default}"` の行を読む)。codex 側の model / effort の設定値はまだない。
- これまでの issue 処理(2026-09-18 以降)では `command codex -m gpt-6-astra -c 'model_reasoning_effort=xhigh' exec review --base main </dev/null` をバックグラウンドで起動し、出力ファイルの `^codex$` 以降の最終ブロックを読む形で安定して動いている。`command` は shell alias(`-m` を足すものがあり、codex は同じフラグの重複を拒否する)を迂回するため。
- `.codex/config.toml`(project 設定)の `model = "gpt-5.5"` は 2026-10-14 に退役予定(#156 の観測対象)。skill が `-m` を明示すれば `exec` の呼び出しはこの値に依存しなくなる。
- `docs/recipes/codex-setup.md` の `codex exec` への言及は hook の承認の話で、呼び出し形は書いていない。`README.md` と `.claude/rules/ralph/post-implementation-pipeline.md` は「`codex exec review` を呼ぶ」とだけ書いている。`.claude/rules/ralph/model-routing.md` は「Cross-review sync note」と「Where the values live」で `RALPH_CLAUDE_REVIEWER_MODEL` に触れている。

## Scope

- `scripts/ralph-config.sh`(+ template、byte 一致): `RALPH_CODEX_REVIEWER_MODEL`(既定 `gpt-6-astra`)と `RALPH_CODEX_REASONING_EFFORT`(既定 `xhigh`)を `RALPH_CLAUDE_REVIEWER_MODEL` と同じ形で定義し export する。
- `/plan` と `/cross-review` の skill 本文(4 面): codex の呼び出しを `command codex -m "${RALPH_CODEX_REVIEWER_MODEL:-gpt-6-astra}" -c "model_reasoning_effort=${RALPH_CODEX_REASONING_EFFORT:-xhigh}" exec ... -o <last-message-file> </dev/null` にし、先に `scripts/ralph-config.sh` を source すること、`</dev/null` と `command` の理由を 1〜2 文で書く。完了の contract(Codex advisory への対応): 数分かかりうるのでバックグラウンドで起動し(Claude Code は Bash の `run_in_background`、Codex driver は自身の background 実行)、その task の完了通知を待ってから、exit code が 0 で、`-o` の最終応答ファイルが新しく書かれて空でないことを確認してから結果を読む。exit が非 0、上限時間(20 分)までに完了しない、最終応答ファイルが空か古い、のいずれかは「review 未完了」として扱い、cross-review では triage report の header に `Reviewer status: incomplete (<reason>)` を書いて Case C(指摘なし)には進まない(再実行するか、`/pr` に known gap として記録するかを AskUserQuestion / 番号選択で選ぶ)。plan advisory では「Codex advisory: incomplete」と記録して先に進む。中断(timeout)時は残った process を止める。最終応答の抽出は人間向けログの `^codex$` ブロックではなく `-o` のファイルを読む。
- 回帰テスト: `tests/test-codex-exec-invocation.sh`(新規、100755)。4 面 × 2 skill の `codex ... exec` を含む行がすべて `</dev/null` と明示の `-m` / `-c model_reasoning_effort=` を持つこと、skill の fallback が `ralph-config.sh` の既定と一致することを検査する。`internal/config/defaults_sync_test.go` の SKILL.md fallback 検査を表駆動にして codex の 2 変数(plan と cross-review の両方)も対象にする。`tests/test-ralph-config.sh` に 2 変数の export と上書きの assertion。
- 文書: `.claude/rules/ralph/model-routing.md`(sync note と値の所在)、`docs/recipes/codex-setup.md`(agent の Bash から `codex exec` を呼ぶときの規則を短く)。

## Non-goals

- `/cross-review` の codex 出力の解釈や triage の変更。
- org runtime の codex 座席の起動(`ralph org spawn`)。別の機構で model を指定している。
- `.codex/config.toml` の `model = "gpt-5.5"` の更新(#156 の観測で判断)。`exec` の呼び出しは本 issue で `-m` 明示になるので影響を受けない。
- `scripts/codex-check.sh` の変更。
- codex の `--sandbox` や承認の設定。

## Assumptions

- codex は `-m <model>` と `-c model_reasoning_effort=<level>` を `exec` / `exec review` の両方で受け付ける(2026-09-18 以降の実績)。
- `defaults_sync_test.go` の `parseShellDefaults` の書式(`NAME="${NAME:-default}"`)に合わせれば、新しい変数も同じ仕組みで読める。Go の `config.Default()` に対応する項目は不要(`RALPH_CLAUDE_REVIEWER_MODEL` も shell 専用)。
- `check-sync.sh` は `.claude/skills/`(`release` を除く)と `scripts/ralph-config.sh` の root / template の一致を検査する(実装時に pass で確認)。

## Affected areas

- `scripts/ralph-config.sh`、`templates/base/scripts/ralph-config.sh`
- `.claude/skills/plan/SKILL.md`、`.claude/skills/cross-review/SKILL.md` とそのミラー 3 面(`.agents/skills/`、`templates/base/.claude/skills/`、`templates/base/.agents/skills/`)
- `tests/test-codex-exec-invocation.sh`(新規)、`tests/test-ralph-config.sh`、`internal/config/defaults_sync_test.go`
- `.claude/rules/ralph/model-routing.md`(+ template があれば同様)、`docs/recipes/codex-setup.md`

## Design decisions

- 設定値の置き場は `scripts/ralph-config.sh`(既存の `RALPH_CLAUDE_REVIEWER_MODEL` と対称)。skill 本文は `${VAR:-default}` で fallback を持ち、fallback と shell の既定の一致をテストで守る(既存の仕組みの延長)。
- 既定値は `gpt-6-astra` / `xhigh`。2026-09-18 以降の cross-review と plan advisory で使ってきた組み合わせで、org の `model_pool` 既定の先頭とも一致する。effort の既定を `high` にしない理由は、review と advisory は 1 回の呼び出しで結論を出す用途で、これまで `xhigh` で問題がなかったこと。
- `command codex` で alias を迂回する。alias が `-m` を足すと codex はフラグの重複を拒否するため、明示のフラグと共存させない。
- 回帰テストは grep 型(`tests/test-no-loop-references.sh` と同じ流儀)。skill 本文の「呼び出し形」は実行できないので、行の形を検査するのが最も安い。
- Critical forks: None

## Acceptance criteria

- [x] AC-1: `scripts/ralph-config.sh` と template が byte 一致で、`RALPH_CODEX_REVIEWER_MODEL="${RALPH_CODEX_REVIEWER_MODEL:-gpt-6-astra}"` と `RALPH_CODEX_REASONING_EFFORT="${RALPH_CODEX_REASONING_EFFORT:-xhigh}"` を定義して export する。`tests/test-ralph-config.sh` が既定値と環境からの上書きを確認する。
- [x] AC-2: `/plan`(step 11.c)と `/cross-review`(step 4 と CLI execution modes 表)の codex の呼び出しが 4 面すべてで `command codex -m "${RALPH_CODEX_REVIEWER_MODEL:-gpt-6-astra}" -c "model_reasoning_effort=${RALPH_CODEX_REASONING_EFFORT:-xhigh}" exec ... -o <file> </dev/null` の形で、先に `scripts/ralph-config.sh` を source すること、バックグラウンド起動と完了の contract(完了通知を待つ、exit 0、`-o` の最終応答ファイルが新しく空でない、20 分の上限、中断時の停止)、`</dev/null` と `command` の理由が書かれている。`./scripts/check-skill-sync.sh` と `./scripts/check-sync.sh` が pass。
- [x] AC-3: `tests/test-codex-exec-invocation.sh`(100755)が、4 面 × 2 skill の `codex ... exec` 行の `</dev/null`、`-m "${RALPH_CODEX_REVIEWER_MODEL:-`、`model_reasoning_effort=${RALPH_CODEX_REASONING_EFFORT:-`、`-o`(または `--output-last-message`)の存在と、fallback が `ralph-config.sh` の既定と等しいことを検査する。mutation: 1 面から `</dev/null` を外す、`-o` を外す、fallback を変える、`ralph-config.sh` の既定を変える、のそれぞれで落ちる。
- [x] AC-4: `internal/config/defaults_sync_test.go` が SKILL.md の fallback 検査を表駆動にし、`RALPH_CLAUDE_REVIEWER_MODEL`(cross-review)に加えて `RALPH_CODEX_REVIEWER_MODEL` と `RALPH_CODEX_REASONING_EFFORT`(plan と cross-review)を検査する。`go test ./internal/config/...` と `TMPDIR=/tmp go test ./internal/config/...` が green。mutation: skill の fallback を変えると落ちる。
- [x] AC-5: `.claude/rules/ralph/model-routing.md` の「Cross-review sync note」と「Where the values live」に codex の 2 変数が入り、`docs/recipes/codex-setup.md` に agent の Bash から `codex exec` を呼ぶ規則(`</dev/null`、明示の model / effort、バックグラウンドと出力ファイル)が短く書かれている。`.claude/skills`、`.agents/skills`、`templates/base`、`docs/recipes`、`README.md`、`.claude/rules` に `</dev/null` のない `codex exec` の呼び出し例が残っていない(「`codex exec review` を呼ぶ」という名前だけの言及は可)。
- [x] AC-6: 実機確認: `. scripts/ralph-config.sh; command codex -m "$RALPH_CODEX_REVIEWER_MODEL" -c "model_reasoning_effort=$RALPH_CODEX_REASONING_EFFORT" exec --sandbox read-only -o <file> 'Reply with the single word ok' </dev/null` が 1 分以内に exit 0 で返り、`<file>` の内容(前後の空白を除く)が `ok` に完全一致する(test report に記録。codex がなければその旨を記録)。plan 時の probe(2026-09-29): 同じ形(`-o` なし)で 11 秒、exit 0、出力の末尾が `ok`。
- [x] AC-7: `RALPH_VERIFY_SCOPE=full ./scripts/run-verify.sh` green(#190 のため full を明示)、`shellcheck -S warning` で新規テストに警告なし。
- [x] AC-8: `/cross-review` の skill 本文(4 面)に「reviewer 未完了」の経路がある: exit 非 0 / 上限時間 / 最終応答ファイルが空か古い、のとき triage report の header に `Reviewer status: incomplete (<reason>)` を書き、Case C には進まず、再実行か `/pr`(known gap として記録)かを選ばせる。`/plan` は「Codex advisory: incomplete」と記録して進む。`tests/test-codex-exec-invocation.sh` が cross-review の 4 面に `Reviewer status: incomplete` の文言があることを検査する(mutation: 1 面から消すと落ちる)。

## Implementation outline

1. Slice A(implementer、sonnet): `ralph-config.sh`(2 コピー)の 2 変数、skill 本文 2 つの書き換え(呼び出し形、完了の contract、未完了の経路)と `sync-skills.sh` による `.agents` 側の再生成(root と template の両方)、template の `.claude/skills` への反映、`tests/test-codex-exec-invocation.sh`、`tests/test-ralph-config.sh` の追加 assertion、`defaults_sync_test.go` の表駆動化、`model-routing.md` と `codex-setup.md` の文書。1 コミット。red の証拠: AC-3 / AC-4 / AC-8 の mutation。実機確認は AC-6。
2. pipeline: self-review → verify → test → sync-docs → cross-review(この skill の新しい呼び出し形で実行する)→ PR。

## Verify plan

- Static analysis checks: `shellcheck -S warning tests/test-codex-exec-invocation.sh`、`bash -n` / `sh -n`、`gofmt -l internal/config`、`go vet ./internal/config/...`、`RALPH_VERIFY_BASE=main ./scripts/run-static-verify.sh`、`./scripts/check-skill-sync.sh`、`./scripts/check-sync.sh`。
- Spec compliance criteria to confirm: AC-1〜AC-7 を該当行と実行結果で確認。4 面の呼び出し行が同一であること。
- Documentation drift to check: `model-routing.md`、`codex-setup.md`、`README.md`、`post-implementation-pipeline.md`、`docs/quality/`。
- Evidence to capture: `docs/evidence/verify-*.log`、verify report の AC 表。

## Test plan

- Unit tests: `sh tests/test-codex-exec-invocation.sh`、`bash tests/test-ralph-config.sh`、`go test ./internal/config/... -count=1`。
- Integration tests: `RALPH_VERIFY_SCOPE=full ./scripts/run-verify.sh`、AC-6 の実機確認。cross-review 自体を新しい呼び出し形で実行する(pipeline の中で自然に検証される)。
- Regression tests: AC-3 / AC-4 の mutation(`</dev/null` の削除、fallback の変更、既定の変更)。
- Edge cases: `RALPH_CODEX_REVIEWER_MODEL` を環境で上書きしたときに skill の形がそのまま使えること、`ralph-config.sh` を source していない shell でも fallback で動くこと、alias が定義された shell で `command codex` が alias を迂回すること(テストでは alias を定義した fixture で `command -v` / 実行行の確認)、`-o` のファイルが存在しない・空・古い場合を「未完了」と判定する手順が skill に書かれていること。
- Evidence to capture: test report(件数、mutation 表、実機確認の出力)。

## Risks and mitigations

- 既定の model が退役する: 値は `ralph-config.sh` の 1 箇所に集まり、`ralph doctor` の退役表示(#165)と #156 の観測で更新する。環境変数で上書きできる。
- 4 面の drift: `check-skill-sync.sh` と `check-sync.sh` が CI で止める。新規テストも 4 面を個別に読む。
- skill 本文が長くなる: 理由は 1〜2 文にとどめ、詳しい経緯は recipe に置く。

## Rollout or rollback notes

skill 本文・設定・テスト・文書の変更のみ。問題があれば 1 コミットを revert する。既存の環境変数の意味は変えない。

## Open questions

- `.codex/config.toml` の `model = "gpt-5.5"`(退役予定)を `gpt-6-astra` に更新するかは #156 の観測で判断する。
- codex 側の reviewer の既定 effort を `xhigh` のままにするか `high` に下げるかは、コストの実測がないので現状維持。

## Deviation notes

- 2026-09-29 plan: Codex plan advisory(gpt-6-astra、xhigh)は MEDIUM 1: バックグラウンド実行に完了・失敗の contract がなく、失敗した reviewer が「指摘なし → PR」に落ちうる → `-o`(`--output-last-message`、`exec` と `exec review` の両方で利用可と確認)で最終応答をファイルに取り、完了通知・exit 0・新しく空でないファイル・20 分の上限・中断時の停止を contract として AC-2 に入れ、未完了の経路を AC-8 として追加、AC-6 を「exit 0 かつ完全一致」に締めた。ユーザー決定: 対応案で plan を更新。plan 時の probe で `-m` / `-c` 付きの `codex exec ... </dev/null` が 11 秒で `ok` を返すことを確認
- 2026-09-29 work: Slice A は implementer(sonnet)に委譲(076b546、19 ファイル、+423 / -65、push 済み)。`ralph-config.sh`(2 コピー)に `RALPH_CODEX_REVIEWER_MODEL`(gpt-6-astra)と `RALPH_CODEX_REASONING_EFFORT`(xhigh)を定義・export。`/plan` 11.c と `/cross-review` step 4・表を `command codex -m … -c … exec … -o <file> </dev/null` にし、バックグラウンド起動・完了通知待ち・exit 0・新しく空でない `-o` ファイル・20 分・停止の contract と、cross-review の `Reviewer status: incomplete (<reason>)` の経路(再実行 / `/pr` に known gap / 中止)を追加。triage report の template に `Reviewer status: complete` 行。4 面は `sync-skills.sh` で再生成し `check-skill-sync.sh` / `check-sync.sh` pass(`model-routing.md` の既知差分は維持)。`tests/test-codex-exec-invocation.sh`(100755、84 assertion)、`tests/test-ralph-config.sh` +4、`defaults_sync_test.go` を表駆動(5 組、全出現を検査)、`model-routing.md` と `codex-setup.md`(2 コピー)。逸脱: skill 本文の「background execution」を「background run」に言い換え(テストの行検出 `codex` + ` exec` が「execution」に誤反応するため。パターンは緩めない)。red: (i) `</dev/null` 除去、(ii) `-o` 除去、(iii) fallback 変更(shell と Go の両方)、(iv) `ralph-config.sh` の既定変更、(v) `Reviewer status: incomplete` 除去、すべて該当ファイルを名指しで落ちる。AC-6: `-o` 付きで rc 0、11 秒、ファイル内容 `ok`。`RALPH_VERIFY_SCOPE=full ./scripts/run-verify.sh` green。orchestrator も 84 / 84、19 / 19、`go test`、check-skill-sync、check-sync を確認
- 2026-09-29 self-review(cycle 1、c9384a8): CRITICAL 0 / HIGH 0 / MEDIUM 5 / LOW 10、merge 可。MEDIUM: 20 分の上限を効かせる仕組みがない(完了通知は hang では来ない)、`-o` の鮮度を確かめられない(固定名、`<scratch>` 未定義)、未完了の経路が Step 4 / 8 で途切れる(insight は pass、backfill も 0/0/0 を pass と読む、Step 9 に再実行の分岐なし)、claude 経路の「常に complete」は誤り、テストが 1 行前提で行分割を見落とす。LOW: ` exec` が「execution」に一致、`command` 未検査、`set -f`、Go の `t.Skipf` が loop 内、plan パス参照、config ヘッダー、到達しない「Codex driver」、理由の重複、`$BASE` の持ち越し、template と文書の表現。14 件を in-cycle で修正し、1 件は据え置き(下の verify の行を参照)
- 2026-09-30 work: Slice B は implementer に委譲(173e8cc、19 ファイル、+180 / -96、push 済み)。設計の逸脱(実機で発見): handoff の watchdog `( sleep 1200; kill "$cpid" ) &` は wrapper を kill しても子の `sleep` が孤児になり、TERM された codex は rc 0 で終わる(codex-cli 0.154.0、1 秒と 5 秒の timeout で確認)。最終形は `codex` と `sleep 1200` を直接の兄弟として background にし、`kill -0 "$spid"` のポーリングで期限切れを検知して codex を kill、`wait "$cpid"` の後に `sleep` とループを kill する。完了の判定は「`codex rc=0` かつ `-o` ファイルが空でない」で、timeout は `-o` の欠落で捕まえる(rc 0 だけでは完了と見なさない)。skill 本文にこの理由を書き、実機で plan advisory(282 秒)と cross-review(290 秒)が rc 0・孤児なし・`.codex/config.toml` 変更なしで通ることを確認。他: `rm -f` と slug / cycle 付きのファイル名、`<scratch>` の定義、`$BASE` と config の source を同じ Bash 呼び出しに、claude 経路も exit 0 かつ非空のときだけ complete、未完了は insight `--verdict n/a` と cycle 据え置き、backfill が 0/0/0 を pass と読む件は tech-debt に記録、テストは ` exec ` と `command codex ` を検査し呼び出し行数(plan 1、cross-review 2)を assert、`set -f`、Go は dir の有無だけで skip、参照は #184。96 / 96、19 / 19、`go test` ok、check-skill-sync / check-sync pass。red: 行分割と `command` 除去で落ちる。orchestrator も 96 / 96 と sync 系を確認
- 2026-09-30 verify(cycle 1、3a15d60): PASS。AC-1〜AC-8 を確認。watchdog の 1 行を `sh -n` で検査し、記述(兄弟の `sleep`、`kill -0` のポーリング、`wait` の後の後始末)と一致。`RALPH_VERIFY_SCOPE=full ./scripts/run-static-verify.sh` green。verifier の指摘: self-review の重複の LOW は半分だけ直っている。理由の一文は /plan への参照に置き換えた(修正済み)。完了 contract の段落は cross-review 側にも残した。この重複は設計判断で残す: 各 skill は単独で読まれて実行され、cross-review の contract には /plan にない未完了時の経路(triage report の `Reviewer status` 行、triage の省略、Step 8 の代替の選択肢)が要るため。tech-debt には記録しない
- 2026-09-30 test(cycle 1、b045ccb): PASS。`RALPH_VERIFY_SCOPE=full ./scripts/run-test.sh` green、96 / 96、19 / 19、`go test` ok。skill の 1 行をそのまま実機で: ok の経路(rc 0、約 15 秒、`-o` が `ok`)、timeout の経路(`sleep 3`、rc 0、`-o` なし = contract の判定どおり)、cross-review の形(rc 0、約 291 秒、`-o` 190 バイト)、dash と `bash --posix` でも完走、孤児なし、`.codex/config.toml` 変更なし。mutation 6 種のうち 5 種が red(行分割は件数ではなく行内容の assertion で落ちる)、`sleep 12000` への変更はどのテストも捕まえない(情報)
- 2026-09-30 sync-docs(cycle 1、88c50b1): drift なし。cross-review(cycle 1、HEAD 88c50b1): この branch の新しい step 4 の形(watchdog の 1 行、`-o`)でそのまま実行。`codex rc=0`、`-o` 189 バイトで contract 上 complete、指摘 0 件。孤児なし、`config.toml` 変更なし。Case C なので `/pr` へ
## Progress checklist

- [x] Plan reviewed
- [x] Branch created
- [x] Implementation started
- [x] Review artifact created
- [x] Verification artifact created
- [x] Test artifact created
- [x] PR created (#192)

## Readiness checklist

- [x] 呼び出し箇所、ミラーの生成と検査の仕組み、既存の fallback 検査をコードで確認した
- [x] critical fork なし
- [x] Codex plan advisory(MEDIUM 1、対応案で plan を更新)
- [x] AC は決定的なテストと 1 回の実機確認で確認できる
