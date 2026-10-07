# guard-bypass-and-hygiene

- Status: Approved
- Approved: 2026-10-07 sha256:d8f86292d5f1
- Owner: Claude Code
- Date: 2026-10-07
- Related request: ハーネスの手入れ 5 件(2026-10-07 ユーザー依頼。bypass permissions のモードでも Bash の実行に Yes / No の確認が出る件は「bypass では出さない」をユーザーが選んだ。残りは PR #204 の後続候補)
- Related issue: N/A
- Type: fix
- Branch: fix/guard-bypass-and-hygiene

## Objective

次の 5 件を片付ける。

1. bypass permissions のモードで、ralph の `pre_bash_guard.sh` が `ask` を返して確認を出している。payload の `permission_mode` が `bypassPermissions` のときは `ask` を返さない。`deny` の規則はモードによらず残す。あわせて、`.git/` を含むコマンドに `2>&1` があるだけで確認が出る、`.env` を読むだけで確認が出る、という誤検知を直す。jq がない環境では、payload からコマンドを取り出す sed がエスケープされた `"` で切れ、`git commit -m "$(id)"` の deny が効いていない(2026-10-07 実測、Codex plan advisory の指摘 1)。ask を飛ばすとこの穴が表に出るので、取り出しも直す
2. `scripts/verify.local.sh` の shellcheck の対象一覧に `scripts/insights-append.sh` がない
3. `verify.local.sh` は実行権限のない `tests/test-*.sh` を黙って飛ばす。新しいテストの `chmod +x` を忘れても気づけない
4. `/sync-docs` は insight event を書く手順を持たない。過去の 50 行は担当エージェントが任意に書いたもので、#203 と #204 には 1 行もない
5. `docs/tech-debt/README.md` が archive 済みの plan 16 本を `docs/plans/active/` のパスで指している(34 か所)

## Scope

- `.claude/hooks/pre_bash_guard.sh`(root と `templates/base/`、同じ内容):
  - payload の `permission_mode` を読む(`extract_json_field`、jq がなければ sed の fallback)
  - 判定の順を「deny の 4 規則(`sudo `、force push、`git reset --hard`、ダブルクォートの `-m` の中のコマンド置換)→ ask の規則」にする。いまはコミットメッセージの deny が ask の後にあり、ask に当たるコマンドでは deny まで届かない
  - `permission_mode` が `bypassPermissions` のときは ask の規則を見ない(何も出さずに終わる)。ほかの値と、値がないときは今までどおり
  - `.git` の ask は、リダイレクト(`>`、`>>`、`>|`)の書き込み先か `tee` の引数が `.git`(worktree では `.git` はファイル)か `.git/` の下を指すときだけにする。`.env` の ask は、書き込み先の最後の要素が `.env` で始まるとき(`.env`、`.env.local`、`.envrc`)だけにする。`case` の glob では `2>&1` と書き込みを区別できないので、`grep -E` で書き込み先を取り出して見る
- `.claude/hooks/lib_json.sh`(root と template、同じ内容): jq がないときの sed fallback を `sed -E` の `"(([^"\\]|\\.)*)"` にして、値の中のエスケープを読み飛ばす。取り出した値は 1 回の置換で `\"` を `"` に、`\\` を `\` に戻す(`s/\\(["\\])/\1/g`)。`\n` や `\uXXXX` は戻さない。deny と ask のパターンはどれも ASCII の固定文字列なので、戻さなくても判定は変わらない。`lib_json.sh` を使うもう 1 つの hook、`post_edit_verify.sh` は `tool_input.file_path` だけを読むので、`\"` を含まない値の結果は変わらない
- `tests/test-pre-bash-guard.sh`: `make_payload` が `"` と `\` を JSON エスケープし、`permission_mode` を入れられるようにする。期待値を none / ask / deny の 3 値にし、出力を厳密に比べる。既存の A・B に加えて、bypass のとき ask が出ないこと、deny が残ること、bypass 以外では ask が出ること、誤検知だった形が何も返さないこと、書き込みの形が ask を返すこと、deny が ask より先に効くこと、`"` と `$(...)` を含むコミットメッセージ(単独と、ask と deny の組み合わせ)、を jq あり・なしの両方で確かめる
- `scripts/verify.local.sh`:
  - shellcheck の対象を、手で並べた `scripts/` の一覧から `scripts/*.sh` のグロブにする。`.claude/hooks/*.sh`、`templates/base/.claude/hooks/*.sh`、`tests/test-*.sh` は今のまま
  - グロブにすると warning が 1 件出る `scripts/run-test.sh` の SC2209(`HARNESS_VERIFY_MODE=test exec ...`)を直す(root と template)
  - `run_hook_tests` で、実行権限のない `tests/test-*.sh` と、git の index で mode が 100644 のものを FAIL として数える。ファイル名と直し方(`chmod +x <file>` と `git update-index --chmod=+x <file>`)を出す
- `tests/test-verify-local-hook-tests.sh`(新規): `verify.local.sh` を一時ディレクトリに写し、偽のテストを置いて `HARNESS_VERIFY_MODE=test` で走らせる。実行権限なし、index が 100644、両方そろっている、の 3 通りを確かめる
- `/sync-docs` の SKILL.md(`.claude/skills/`、`.agents/skills/`、`templates/base/` の 2 面、計 4 面)に「Insight event (best-effort)」の節を足す。コマンドは `./scripts/insights-append.sh --slug <slug> --flow standard --phase sync_docs --verdict pass --cycle auto --source skill || true`。`tests/test-skill-insight-cycle.sh` の対象に sync-docs を足す
- `docs/tech-debt/README.md`: `docs/plans/active/<file>` のうち、そのファイルが `docs/plans/archive/` にあるもの(16 本、34 か所)を `docs/plans/archive/<file>` に直す
- `scripts/archive-plan.sh`(root と template): 処理の順を「移動元の解決 → 移動先の衝突の検査 → `docs/tech-debt/README.md` の書き換え → plan の移動」にする。書き換えは、README があれば `docs/plans/active/<name>` を `docs/plans/archive/<name>` にし、一時ファイルに書いてから置き換え、書き換えた件数を出す。直後に名前の文字(英数字、`.`、`_`、`-`)が続く参照は別の plan なので書き換えない。README がなければ何もしない。途中で止まっても、やり直せば移動が走り、書き換えは置き換える参照がないので何もしない(Codex plan advisory の指摘 2)
- `/pr` の step 8(4 面): `archive-plan.sh` が README を書き換えたら、それも同じコミットに入れる、と書く
- `tests/test-archive-plan.sh`(新規): 書き換え、似た名前(`2026-10-07-foo` と `2026-10-07-foo-bar.md`)を書き換えないこと、README がないとき、移動が失敗したあとのやり直し(archive のディレクトリを書き込み不可にして失敗させ、権限を戻してやり直すと完了し、2 回目の書き換えは 0 件)、を一時ディレクトリで確かめる。root で走るときは失敗のケースを SKIP にする
- `scripts/verify.local.sh`: `docs/tech-debt/README.md` の中の `docs/plans/active/<x>` と `docs/plans/archive/<x>` の参照が、どちらかの下に実在するかを確かめる(ファイルの plan とディレクトリの plan の両方)。plan を手で移したときの参照切れも拾う(consult の提案)。Go のコメントや fixture は見ない
- ミラー: `.agents/skills/`(`scripts/sync-skills.sh`)と `templates/base/`

## Non-goals

- `auto`、`dontAsk` など bypass 以外のモードでの扱いを変えること
- Claude Code 本体が出す確認(重要なパスの `rm` など)を消すこと。hook の外にあるので変えられない
- Codex 向けに別の扱いを作ること。self-review(L-2)が、codex-cli 0.160.0 の PreToolUse の入力にも `permission_mode` があり、値に `bypassPermissions` を含むことを確かめた。そのため Codex でもこのモードでは ask を返さなくなる。同じ binary に「ask は未対応」という文言があり、Codex は ask をもともと効かせていなかったとみられる(おそらく実害はない。未確認)。Codex が ask をどう扱うかは tech-debt の 127 行目のまま未確認として残す
- コミットメッセージの guard の誤検知(tech-debt の 124 行目、`-m "$(cat <<'EOF'` の形を deny する)を直すこと
- `cp`、`mv`、`sed -i` で `.git` や `.env` に書く形を捕まえること。今の guard は、この形の後ろにたまたまリダイレクトがあるとき(`cp hook .git/hooks/pre-commit 2>&1` など)だけ確認を出していた。書き込み先の判定に絞るとこの偶然の検出もなくなる(self-review M-1 で実測)。リダイレクトのない `cp` や `mv` は今も捕まえていないので、新しい検出は別の作業として tech-debt に記録する
- Go のコメントやテストの fixture に残る `docs/plans/active/` の参照(約 25 か所)を直すこと。後続候補として記録する
- `docs/reports/` と `docs/insights/` の中の `docs/plans/active/` の参照。その時点の記録なので書き換えない

## Assumptions

- Claude Code の PreToolUse の payload は `permission_mode` を持つ(Claude Code の hook の文書による)。値は `default`、`plan`、`acceptEdits`、`auto`、`dontAsk`、`bypassPermissions`
- hook が `ask` を返すと、bypass でも確認が出る。Claude Code の permissions の文書は、hook の出力で確認を強制できる(force a prompt)と書き、bypass を除くとは書いていない(consult の確認)。ユーザーが見た挙動とも合う。`permissions.ask` の規則も managed settings も、この環境にはない(2026-10-06 確認)
- jq がない環境では、いまの guard は `git commit -m "$(id)"` に何も返さず、`rm -rf x && git commit -m "$(id)"` には ask を返す(2026-10-07、PATH から jq を外して実測)
- この repo では guard が 2 回走る(`.claude/settings.local.json` から直接と、`ralph-dispatch.sh` → `.claude/hooks/PreToolUse.d/10-pre-bash-guard.sh` から)。同じスクリプトなので判定は同じになる
- `scripts/*.sh` の 27 本に `shellcheck --severity=warning` を掛けると、warning が出るのは `scripts/run-test.sh` だけ(2026-10-07 手元で確認)。SC2209 は右辺を引用符で囲めば消える
- いまの `tests/test-*.sh` はすべて実行権限があり、index でも 100755(2026-10-07 確認)。検査を足しても既存のテストは落ちない
- `insights-append.sh` は `--phase sync_docs` と `--cycle auto` を受け付ける(#204)。`ralph insights` の表示は `sync_docs` を並びに含んでいる(`internal/cli/insights.go:111`)
- `docs/tech-debt/README.md` の `docs/plans/active/` の参照 34 か所は、すべて plan が archive にある(2026-10-07 確認、active や行方不明のものはない)
- 下流の `templates/base/docs/tech-debt/` は `.gitkeep` だけなので、下流で README がないときに `archive-plan.sh` が何もしないことを確かめる必要がある

## Affected areas

- `.claude/hooks/pre_bash_guard.sh`、`templates/base/.claude/hooks/pre_bash_guard.sh`
- `.claude/hooks/lib_json.sh`、`templates/base/.claude/hooks/lib_json.sh`(`post_edit_verify.sh` も読む)
- `tests/test-pre-bash-guard.sh`
- `scripts/verify.local.sh`(root だけ。下流には配らない)
- `scripts/run-test.sh`、`templates/base/scripts/run-test.sh`
- `tests/test-verify-local-hook-tests.sh`(新規)
- `.claude/skills/sync-docs/SKILL.md`、`.claude/skills/pr/SKILL.md` と、`.agents/skills/`・`templates/base/.claude/skills/`・`templates/base/.agents/skills/` の同じファイル
- `tests/test-skill-insight-cycle.sh`
- `docs/tech-debt/README.md`
- `scripts/archive-plan.sh`、`templates/base/scripts/archive-plan.sh`
- `tests/test-archive-plan.sh`(新規)
- ドキュメント(`/sync-docs` で確かめる): `.codex/README.md` の 116 行目(guard が止めるという説明)、`docs/tech-debt/README.md` の 127 行目(Codex の ask。この PR で guard を触ったので見直す時期に当たる)、128・129 行目(`/sync-docs` に insight event の手順がない、という記述。S3 で事実が変わる)、157 行目(shellcheck の対象一覧。S2 で解消)、self-review が提案した新しい行(`cp`・`mv`・`sed -i` での書き込みの検出、前からある取りこぼし)、`docs/quality/quality-gates.md`(verify.local.sh の説明があれば)

## Visual review

- 図解ページ: `.harness/state/plan-visual/guard-bypass-and-hygiene.html`
- 自己チェック: ヘッドレス Chrome で全体・`#overview`・`#guard-flow`・`#examples` を撮って確認。図 2 で等幅の補足が枠に触れていたのと「いいえ」が次の枠に寄っていたのを、ノードの幅と位置を変えて直した。図 3 の「今」の列は、いまの guard に同じ payload を渡した実測(15 例と、jq を外した 2 例)。Codex plan advisory と consult を反映したあと、図 1 を 3 段組みに組み直し(`lib_json.sh` と `verify.local.sh` の検査を足し、ノードは 10 個)、図 3 に jq がない環境の 2 行を足して撮り直した

## Design decisions

- **bypass では ask を返さない(ユーザー確定、2026-10-07)**。hook の中で payload の `permission_mode` を見る。採らなかった案: ask の規則をすべて消す、ask を deny に変える
- **deny を ask より先に判定する**。いまは `rm -rf` とダブルクォートのコミットメッセージを 1 つのコマンドに並べると ask で止まり、deny の検査まで届かない。bypass で ask を飛ばすようにすると、この順のままでは同じコマンドの deny も飛ばしてしまうので、順を入れ替える
- **`.git` と `.env` は書き込み先だけを見る**。`case` の glob では「`.git/` がどこかにあり、その後ろに `>` がある」としか書けず、`ls .git/ 2>&1` も当たる。`grep -E` で `>`・`>>`・`>|` の直後と `tee` の引数を取り出して見る。副作用として、`> .github/...` と `> .gitignore`(今は `> .git` の前置きで当たっている)も確認が出なくなる
- **shellcheck の対象は `scripts/*.sh` のグロブにする**。依頼は 1 本の追加だが、手で並べた一覧から 26 本が漏れていた。`run_hook_tests` がテストの一覧をやめてグロブにしたのと同じ理由(一覧のずれで黙って漏れる)。グロブにして増える warning は run-test.sh の 1 件だけなので、ここで直す
- **実行権限のないテストは FAIL にする**。warning だけでは CI が通ってしまい、気づけない。working tree の権限と index の mode の両方を見る。手元で `chmod +x` しただけで index が 100644 のままだと、CI では実行権限がなく落ちるため
- **sync-docs の verdict は `pass`**。過去の 50 行と同じ値にする。sync-docs には止める条件がないので `fail` は使わない
- **参照切れは `archive-plan.sh` で防ぐ**。16 本がずれていたのは、plan を移すときに参照を直していないため。書き換える範囲は `docs/tech-debt/README.md` だけにする。reports と insights はその時点の記録、Go のコメントと fixture は別の文脈で、テストの期待値も含むため触らない
- **jq がないときの取り出しを直す(Codex plan advisory の指摘 1、ユーザーが「Update plan」を選択)**。bypass で ask を飛ばすと、jq がない環境では ask に当たって止まっていたコミットメッセージの deny が何も出さずに通る。sed fallback をエスケープを読める形にする。採らなかった案: jq がないときは bypass でも ask を飛ばさない。jq の有無で bypass の挙動が変わる分岐が増え、ユーザーが選んだ挙動から外れる。取り出しを直してテストで縛れば足りる(consult の判定)
- **`archive-plan.sh` は README を先に書き換え、そのあと plan を移す(Codex plan advisory の指摘 2)**。移動のあとで書き換えると、書き換えで止まったときに移動元がなく、移動先があるので、やり直しが効かない。順を入れ替えると、やり直しで移動が走り、書き換えは何もしないので、再開用の分岐が要らない
- **tech-debt README の参照は `verify.local.sh` でも確かめる(consult の提案)**。`archive-plan.sh` を通さずに plan を移したときの参照切れを拾う。検査だけにすると、doc-maintainer が active の plan を指す行を書き、`/pr` が archive したあとの CI で毎回落ちるので、書き換えと両方入れる
- Critical forks: None。残った分岐(グロブにするか 1 本足すか、FAIL か warning か、archive-plan で直すか)は、どれも 1 slice 以内でやり直せる

## Acceptance criteria

- [x] AC1: `pre_bash_guard.sh`(root と template、同じ内容)は、payload の `permission_mode` が `bypassPermissions` のとき ask を返さない。deny の 4 規則はどのモードでも deny を返す。`permission_mode` がないとき、`default` や `auto` のときは今までどおり ask を返す。ask と deny の両方に当たるコマンドは deny になる。jq がない環境でも、JSON エスケープされた `git commit -m "$(id)"` は deny になる(`lib_json.sh` の sed fallback がエスケープを読む。root と template は同じ内容)。`tests/test-pre-bash-guard.sh` が、期待値を none / ask / deny の 3 値にして jq あり・なしの両方で確かめる
- [x] AC2: `.git` と `.env` の ask は書き込み先だけを見る。`ls .git/ 2>&1`、`git status 2>&1 | grep .git/`、`cat .env.example 2>/dev/null`、`grep X .env`、`cat > .github/workflows/x.yml`、`echo x > .gitignore` は何も返さない。`echo x > .git/hooks/pre-commit`、`echo x >> /abs/repo/.git/config`、`echo x >.git/x`(空白なし)、`tee -a .git/x`、`echo x > .git`、`cat > .env <<EOF`、`printf x > .env.local` は bypass 以外で ask を返す。テストが同じ形を jq あり・なしで確かめる
- [x] AC3: `verify.local.sh` の shellcheck の対象が `scripts/*.sh` のグロブで、`scripts/insights-append.sh` を含む。`scripts/run-test.sh`(root と template、同じ内容)に SC2209 が出ない。`./scripts/run-verify.sh` の shellcheck の段が通る
- [x] AC4: `run_hook_tests` は、実行権限のない `tests/test-*.sh` と index で 100644 のものを FAIL として数え、ファイル名と直し方を出し、`verify.local.sh` が非 0 で終わる。`tests/test-verify-local-hook-tests.sh` が 3 通り(実行権限なし、index が 100644、両方そろう)を確かめる
- [x] AC5: `/sync-docs` の SKILL.md(4 面)に insight event の節があり、`--phase sync_docs`、`--verdict pass`、`--cycle auto`、`|| true` がある。`tests/test-skill-insight-cycle.sh` が sync-docs を含む 5 skill × 4 面で確かめる
- [x] AC6: `docs/tech-debt/README.md` に、archive にある plan を `docs/plans/active/` で指す参照が残っていない。`archive-plan.sh`(root と template、同じ内容)は、移動先の衝突を確かめたあと README の参照を書き換えて件数を出し、それから plan を移す。直後に名前の文字が続く参照は書き換えない。README がなければ何もしない。`/pr` の step 8(4 面)に、書き換えた README も同じコミットに入れると書いてある。`tests/test-archive-plan.sh` が、書き換え・似た名前・README なし・移動が失敗したあとのやり直し、の 4 点を確かめる。`verify.local.sh` は README の plan の参照が active か archive に実在しないと FAIL を出す
- [x] AC7: `./scripts/check-skill-sync.sh`、`./scripts/check-sync.sh`、`./scripts/check-pipeline-sync.sh`、`bash scripts/check-template-purity.sh`、`./scripts/run-verify.sh` が通る

## Implementation outline

1. S1: guard の bypass と誤検知、`lib_json.sh` の sed fallback、テスト(AC1、AC2)
2. S2: `verify.local.sh` の shellcheck の対象と実行権限の検査、`run-test.sh`、テスト(AC3、AC4)
3. S3: `/sync-docs` の insight event、テストの対象(AC5)
4. S4: tech-debt のパス、`archive-plan.sh` の順と書き換え、`/pr` step 8、`verify.local.sh` の参照の検査、テスト(AC6)
5. ドキュメントは `/sync-docs` で確かめる

## Verify plan

- Static analysis checks: shellcheck(`scripts/*.sh` に広げた対象と新しいテスト 2 本)、`check-skill-sync.sh`、`check-sync.sh`、`check-pipeline-sync.sh`、`check-template-purity.sh`
- Spec compliance criteria to confirm: AC1〜AC7。とくに AC1 の「deny が先」と、AC2 の誤検知の形と書き込みの形が両方テストにあること
- Documentation drift to check: guard の挙動を説明している文書(`.codex/README.md`、`docs/tech-debt/README.md` の 124・127 行目、`git-commit-strategy.md`)が新しい挙動と矛盾しないか
- Evidence to capture: verify レポート

## Test plan

- Unit tests: `tests/test-pre-bash-guard.sh`(拡張)、`tests/test-verify-local-hook-tests.sh`(新規)、`tests/test-archive-plan.sh`(新規)、`tests/test-skill-insight-cycle.sh`(拡張)
- Integration tests: `./scripts/run-verify.sh` の全体(shellcheck の対象が広がったことと、実行権限の検査が既存のテストで落ちないこと)
- Regression tests: `./scripts/run-test.sh` の全体、`go test ./...`(`archive-plan.sh` と skill の変更が scaffold の fixture に響かないこと)
- Edge cases: jq がないときの `permission_mode` の取り出し、`permission_mode` が JSON にないとき、JSON エスケープされた `"` と `\`、jq なしでの `tool_input.file_path` の取り出し(`lib_json.sh` を直接 source)、複数行のコマンド、`2>&1` と `2>/dev/null`、fd つきの書き込み(`2> .git/x`)、空白なしのリダイレクト、絶対パス、`.git` がファイルのとき、似た名前の plan、README がないとき、移動が失敗したあとのやり直し
- Evidence to capture: test レポート

## Risks and mitigations

- bypass で ask を返さないと、`.env` への書き込みや `rm -rf` が確認なしで走る → ユーザーが選んだ挙動。deny の 4 規則と、Claude Code 本体の重要なパスの `rm` の確認は残る。PR 本文に書く
- 書き込み先の判定を狭めて、今まで捕まえていた書き込みを見逃す → テストで書き込みの形(`>`、`>>`、空白なし、絶対パス、`tee -a`、heredoc、`.git` がファイル)を確かめる。外れるのは、`.github/` と `.gitignore` など `.git` で始まる別の名前(意図した変更)と、旧 guard が後ろの `>` に偶然当たって確認を出していた形。後者には、`cp`・`mv`・`sed -i` で書く形(self-review M-1)と、`tee` の書き込み先の前に引数の読み取りが止まる文字(`&`、`<`、`\` など)がある形(`tee out 2>&1 .env > /dev/null`、`tee < in.txt .env > /dev/null`。cycle 2 の verify)がある。どちらも今の検出が偶然に頼っているので、PR 本文と tech-debt に書く
- shellcheck の対象を広げると、CI の shellcheck の版で新しい warning が出るかもしれない → PR の CI で確かめる。出たら直し、直せないものは理由を書いて除く
- `archive-plan.sh` の書き換えが README の別の参照を壊す → 名前の境界を見て、似た名前をテストで確かめる。書き換えた件数を出すので、`/pr` のコミットで差分として見える
- 実行権限の検査が、`core.fileMode=false` の環境(Windows など)で誤って FAIL を出す → CI は ubuntu で、この repo の開発環境は macOS。起きたら index の mode だけを見る形に寄せる
- 文字列の一致で guard を書くので、heredoc の本文に `> .env` と書いただけでも bypass 以外では確認が出る → 今と同じ。PR 本文に書く
- `lib_json.sh` の sed fallback を変えると、`post_edit_verify.sh` の `file_path` の取り出しにも響く → `\"` を含まない値の結果は変わらない。テストで jq なしの `file_path` の取り出しも 1 件確かめる
- jq の経路では `\n` が改行に戻り、sed の経路では文字列 `\n` のまま残るので、複数行のコマンドで 2 つの経路の判定がずれうる → 書き込み先の判定は行の頭に頼らない形にし、複数行の例をテストで両方の経路に通す
- `.git` と `.env` の書き込み先の判定を `grep -E` で書くので、BSD と GNU の差で動きが変わりうる → `\b`・`\s`・`\w` を使わず POSIX のクラスだけにする。CI(ubuntu、GNU)と手元(macOS、BSD)の両方で通ることを確かめる

## Rollout or rollback notes

- hook、スクリプト、skill、文書、テストの変更。戻すときは PR を revert する
- 下流には次のリリースの `ralph upgrade` で届く。hook と skill と `archive-plan.sh` は core として置き換わる。`verify.local.sh` は root だけのファイルで、下流には届かない

## Open questions

- なし

## Progress checklist

- [x] Plan reviewed
- [x] Plan approved
  - 2026-10-07: 承認ゲートは、ユーザーの事前の指示「以後、私は寝るので全ての確認は承認扱いで大丈夫です。起床したときにはPRがマージされている状態にしておいてください。」により Approve として記録した。Codex plan advisory の 2 件は、ユーザーが「Update plan」を選んで反映済み。consult(consult-plan-guard)の判定は「進めてよい」
- [x] Branch created
- [x] Implementation started
  - S1 完了(da200cb0、implementer/opus): guard の判定を deny → bypass なら終了 → ask の順にし、`.git`・`.env` は `grep -E` で書き込み先だけを見る。`lib_json.sh` の sed fallback は `sed -E` でエスケープを読む。テストは none / ask / deny の 3 値で 220 件(jq あり・なし)。dash、GNU grep、ubuntu:24.04 でも通過。handoff からの差分: 書き込み先の語の終わりに `<` と `>` を足し(`cat >.env<<EOF`)、`tee` の直前に `/` を許した(`/usr/bin/tee`)
  - S2 完了(9fa40302、implementer/opus): shellcheck の対象を `scripts/*.sh` に(94 ファイル、`insights-append.sh` を含む)、`run-test.sh` の SC2209 を直し、実行権限なしと index 100644 のテストを FAIL に。テストは 4 通り(handoff の 3 通りに、untracked で実行権限ありの 1 通りを足した)
  - S3 完了(4abc3566、implementer/sonnet): `/sync-docs` の 4 面に insight event の節、`test-skill-insight-cycle.sh` は 5 skill × 4 面(20 件)
  - S4 完了(3ac0ed9b、implementer/opus): tech-debt README の 34 か所を archive のパスに、`archive-plan.sh` は README を先に書き換えてから移す、`/pr` step 8 の 4 面、`verify.local.sh` の参照の検査、`tests/test-archive-plan.sh`(22 件)。plan の範囲内の判断 2 点: 文末の `.` を参照の区切りとして扱う(verify が末尾の `.` を落として調べるのと合わせるため)。verify の検査は「参照したパスそのものが実在する」で見る(Scope の「どちらかの下に実在」は AC6 の「active か archive に実在」の意味で、手で移したときの参照切れを拾うにはこの読みが要る)
  - AC7: `check-skill-sync.sh`、`check-sync.sh`、`check-pipeline-sync.sh`、`check-template-purity.sh`、`run-verify.sh` がすべて通過(2026-10-07)
  - self-review(9bda19e3、reviewer/opus): Merge 可、MEDIUM 1・LOW 5。L-2・L-3・L-4 は 989886f2 で直した(inline。コメント 2 か所と表示の分岐 1 か所で、handoff より安いため)。M-1(`cp`・`mv`・`sed -i` の後ろにリダイレクトがある書き込みを、今は偶然捕まえていた)と L-2(Codex の payload にも `permission_mode` がある)に合わせて、plan の Non-goals・Risks・Affected areas の記述を事実に直した。本文が変わったので図 3 に `cp` の行を足して撮り直し、承認の digest を c3b201489419 から c07adf402bdb に取り直した(ユーザーの事前承認の範囲)。L-1(タブ)と L-5(tech-debt の 127・157 行目)は `/sync-docs` と tech-debt に渡す
  - verify(`docs/reports/verify-2026-10-07-guard-bypass-and-hygiene.md`、verifier/opus): pass。AC4 の直し方の表示は plan の文言(`chmod +x` と `git update-index --chmod=+x`)より細かい(V-8)。989886f2(L-3)以降、untracked のテストには `git add --chmod=+x` を出し、index だけが 100644 で working tree に実行権限があるときも `chmod +x` を出す。AC4 の「ファイル名と直し方を出し、非 0 で終わる」は満たしている。plan の本文は digest の対象なので直していない
  - test(`docs/reports/test-2026-10-07-guard-bypass-and-hygiene.md`、tester/opus): pass。Test gaps 1〜9 は tech-debt に 1 行にまとめた(コードとテストは足していない)
  - sync-docs(`docs/reports/sync-docs-2026-10-07-guard-bypass-and-hygiene.md`、doc-maintainer/sonnet): V-1〜V-7、V-9 を直した。V-8 は上のとおり plan を直さない
  - cross-review cycle 1(811e1452、codex): ACTION_REQUIRED 2(jq がないとき改行の後の `tee` を見逃す、`tee` の引数が `<` を越えて入力を拾う)。ユーザーの事前承認により推奨の「Fix」を選び、cycle を 2 に上げた
  - cycle 2: 修正 0db1a97e(implementer/opus、テスト 260 件)。self-review(14d48a77)で C2-M1(`\tee` とバッククォートの中の `tee` を旧 guard より弱く見逃す)と C2-L1(`tee` の引数が `>` で止まる)が出て、ab3ee31c(implementer/opus、テスト 280 件)で直した。verify(69d1c581)は pass。旧 guard が後ろの `>` に偶然当たって確認を出していた `tee` の形が残ると分かったので、plan の Risks を直し、図 3 に行を足して、承認の digest を c07adf402bdb から d8f86292d5f1 に取り直した(ユーザーの事前承認の範囲)
- [x] Review artifact created
- [x] Verification artifact created
- [x] Test artifact created
- [ ] PR created
