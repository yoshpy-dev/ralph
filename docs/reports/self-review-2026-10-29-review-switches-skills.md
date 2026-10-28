# Self-review report: review-switches-skills

- Date: 2026-10-29
- Plan: `docs/plans/active/2026-10-29-review-switches-skills.md`(issue #266、D1〜D5、AC1〜AC9、Progress の S1〜S4)
- Reviewer: reviewer サブエージェント(`/self-review`、パイプラインの cycle 1)
- Scope: `git diff 9837f744...HEAD`(HEAD 4a2200c4、9 コミット、55 ファイル、+662/-256)。中心は S1 07cf95f8(backfill の `skipped`、ルートの `ralph.toml` の test)、S2 93dec2cc(`/plan` 手順 11)、S3 9e233825(`/cross-review` 手順 0、`/pr`、`/implement`、Codex の上書き)、S4 f88561b8(文書とコメント)。計画ファイルと Progress だけのコミットは読んだが所見の対象にしていない。命名、読みやすさ、null の扱い、例外処理、security、保守性の 6 項目で見た。機械的な項目(不要な変更、誤字、デバッグ残り、秘密情報、slice 外のファイル)は実装側が見る範囲なので探していない。
- 既定の判断(依頼どおり欠陥として扱わない): `/cross-review` の claude の道が `--permission-mode auto` のままであること、codex の道が利用者の MCP server を止めないこと、`templates/base/ralph.toml:28-29` の「in a later release」と `docs/tech-debt/README.md:31`・`:154` が /sync-docs 待ちであること、この環境の PATH の `ralph` が v6.0.0 で `config get` を持たないこと。
- ID: 所見は L1〜L6(すべて LOW)。

## Evidence reviewed

- `/plan` 手順 11(a〜h)と `/cross-review` 手順 0〜6・9・「CLI execution modes」・「Insight event」・「What /cross-review does NOT do」の差分全体。`/pr` 手順 5 の追加と雛形、`/implement` 13.e、`.codex/AGENTS.override.md`、triage の雛形。
- 4 つの写しの一致: `plan`・`cross-review`・`pr`・`implement` の `SKILL.md` と `pr/template.md` について、`.claude/` と `templates/base/.claude/`、`.agents/` と `templates/base/.agents/` を `cmp` で突き合わせた(すべて一致)。`.claude/` と `.agents/` の差は `cross-review` の frontmatter の `allowed-tools` 1 行だけで、`9837f744` の時点から同じ。差分に出た `templates/base/` 側のファイルを、root 側の同じ path と全部 `cmp` した結果、違うのは `AGENTS.md`(管理ブロックの別物)と `model-routing.md`(org の節が元から別)だけで、どちらも今回の差分の外の違い。
- 設定の読み方の文: `/plan` 11.a と `/cross-review` 手順 0 は、終了コードの 4 分岐(0 の `true`/`false`、1 と `unknown command "config"`、127、それ以外は止まる)と、記録の文言(`off: [standard].<key>`、`ralph without config get; treated as off`、`ralph not found; treated as off`、`<reviewer> not available`)が、キー名以外は同じ。driver の判別(11.b と手順 2.a)も一語違わず同じ。
- 旧い ralph の挙動: PATH の `ralph`(v6.0.0)で `ralph config get standard.plan_advisory` は exit 1 と stderr `unknown command "config" for "ralph"`。skill の分岐と一致する。
- git の状態の保存コマンド(`{ git rev-parse HEAD; git status --porcelain=v1 --untracked-files=all; git diff --cached; git diff; git ls-files -z --others --exclude-standard | xargs -0 git hash-object --; }`)を、scratchpad の一時 repo で試した。未追跡のファイルがないとき、BSD の `xargs` は `git hash-object` を呼ばず、rc 0 で HEAD だけ書く。未追跡が 1 つあるとき、その blob の hash が出る。空白を含む名前も `-z`/`-0` で読める。未追跡の dangling symlink があると `git hash-object` が `fatal: could not open` で落ち、同じ呼び出しに入った後ろのファイルの hash が出ない(Notes)。`git hash-object --` に引数も stdin もないときは rc 0 で何も出さないので、GNU の `xargs`(空入力でも 1 回呼ぶ)でも同じ出力になる。
- claude の起動形(`rm -f …; RALPH_WIP_CHECKPOINT=off command claude … </dev/null > file 2> log & cpid=$!; sleep 1200 & spid=$!; ( while kill -0 … ) & wpid=$!; wait "$cpid"; rc=$?; …; echo "claude rc=$rc"`)を、偽の `claude` を PATH の先頭に置いて bash・zsh・sh で動かした。3 つとも `claude rc=0` を出し、出力は `-last.md` に、`RALPH_WIP_CHECKPOINT=off` は子に渡り、argv は `-p --model opus …` の順で届く。`ps` で見ると `$!` は env 前置きと `command` を通しても claude 自身の pid で、watchdog の `kill "$cpid"` は claude に届く。`sleep 1200` の掃除も 3 つとも残さなかった。
- `claude --help`(2.1.304)に `--restricted`・`--strict-mcp-config`・`--disallowedTools`・`--tools`・`--permission-mode dontAsk`・`--output-format` があり、`--restricted` の説明(code-running tools を外す、user・project・local の settings を無視、managed settings と `--settings` は効く、file tool を cwd に閉じる)は skill の注釈と合う。
- `internal/insights/backfill.go` の差分全体と、追加した 2 つのテスト。`reviewerStatusRe` は先頭の語だけを取り、`strings.ToLower` のあとで `incomplete`・`skipped` の prefix を見る。`Skipped` の大文字も通る。変数名 `noReview` は 2 つの理由を束ねたもので、コメントも両方を言っている。
- `internal/config/root_ralph_toml_test.go` の全体。ゼロ値の `Config` に `DisallowUnknownFields` で読むので、4 つの true はファイルが自分で書いたものだけになる。`StrictMissingError` から key と行を出す。ファイルがなければ `Fatalf`。`repoRoot` は同じ package の既存の道具。
- `defaults_sync_test.go` の差分(`{"plan", "RALPH_CLAUDE_REVIEWER_MODEL"}` の 1 行とコメント)。plan skill の CLI 表は fallback を持たないので、コメントの「表は fallback を書かない。出現は 11.c の codex と claude の行」と合う。
- `git grep -nEi` で旧い言い方の残り(`silently skip`、`Codex plan advisory`、`Codex advisory`、`Codex not available`、`skip(ping)? to /pr`、`auto, optional`)を、reports・plans・specs・tech-debt・evidence・`.agents`・`templates` を除いて探した。当たりは過去の PR の番号つきコメント(`internal/org/*`、`internal/cli/migrate*.go` の「Codex advisory 3」など)と、`.codex/README.md:157` の Codex の hook の話だけで、今回の skill・rules・README・AGENTS.md には残っていない。
- `scripts/xreview-helpers.sh`・`scripts/ralph-config.sh`・`internal/org/driver/paneenv.go`・`.codex/config.toml`・`tests/test-codex-exec-invocation.sh` のコメントの差分。
- 確かめていない: テスト、静的検査、変異の実行(skill の規則と依頼どおり、走らせていない)。claude の呼び出しを実機で最後まで通した結果(AC1 の小さな実行)は計画の Progress の S2 にあるものを読んだだけで、再実行していない。`--restricted` は settings を読まないので、settings 経由の認証(`apiKeyHelper` や settings の `env` で Bedrock・Vertex を選ぶ構成)を使う環境で claude が認証に失敗するかは未確認。失敗したときは rc が 0 にならず「incomplete」に落ちる作りなので、黙って通ることはない。

## Findings

<!-- Area recommended values: naming, readability, null-safety,
     exception-handling, security, maintainability -->

| Severity | Area | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- |
| LOW | maintainability | L1 この PR で事実と違うか、時制が合わなくなる文が、/sync-docs の待ちの一覧(`templates/base/ralph.toml:28-29`、`tech-debt:31`・`:154`)に入っていない。(a) `internal/config/config.go:30-31` の「The skills read them with `ralph config get <key>` (they start to in issue #266)」は、この PR で skill が読むようになったので「they start to」が過去の話になる(#265 の self-review L6 と同じ型)。(b) `templates/base/ralph.toml:23-25` の「An older ralph without that command fails with exit 1 …; the skills treat that, and only that, as off」は、D1 で exit 127(ralph が PATH にない)も off になった今、「only that」が実際より狭い。同じ言い方が `internal/cli/config.go:60-62` の help と `config_test.go:228` の固定にもあるが、そちらは「exit 1 の中で」の話とも読めるので、読み手しだい。(c) `internal/org/driver/paneenv.go:66-67` の警告の条項「a seat's cross-review may pick the wrong CLI … (pick_reviewer treats the seat's driver as claude)」は、skill が driver を CLI 自身から決めて `pick_reviewer` に渡す(D5)ので、`RALPH_PRIMARY_CLI` のない座席でも当てはまらなくなる。文字列を test が固定している(S4 の報告)ので、直すなら test も動く。 | `config.go:31`、`templates/base/ralph.toml:23-25`、`internal/cli/config.go:60-62`、`paneenv.go:66-67` と `plan/SKILL.md` 11.b の driver の文 | (a) は「The skills read them with …」と現在形にする。(b) は template の注に「`ralph` が PATH にないとき(exit 127)も、同じく off として記録する」の 1 文を足す(28-29 行目の直しと同じ /sync-docs の範囲)。help と test は #265 の動詞に属するので、触らないなら Known gaps か tech-debt の 1 行にする。(c) は直すなら警告と test を一緒に「`RALPH_PRIMARY_CLI` は skill の driver の上書き」と合わせる。触らないなら (b) と同じ 1 行にまとめる。 |
| LOW | exception-handling | L2 手順 2.b の `REVIEWER=$(pick_reviewer "$DRIVER")` は、`$DRIVER` を定義する行がどこにもない。skill 自身が「Bash tool は呼び出しの間で shell の状態を持たない」と書いている(11.c)ので、driver を決めた agent が別の Bash 呼び出しでこの 1 行だけを打つと、`$DRIVER` は空になる。`pick_reviewer ""` は `${1:-${RALPH_PRIMARY_CLI:-claude}}` で claude に倒れ、`codex` を返す。D5 が直そうとした「Codex が driver なのに codex を相手にする」誤りが、黙って同じ形で起きる。この 1 行は旧い版からの写しで、旧い版は 2.a の自動判別が同じ呼び出しに入る前提だったが、今は判別が散文だけになり、結び付ける手がない。`.codex/AGENTS.override.md` が `RALPH_PRIMARY_CLI=codex` を渡すので Codex では二重に守られているが、それは env を渡した場合に限る。 | `plan/SKILL.md` 11.b と `cross-review/SKILL.md` 手順 2.b、`scripts/xreview-helpers.sh:71`(`pick_reviewer` の既定)、手で: `DRIVER= ; pick_reviewer "$DRIVER"` は `RALPH_PRIMARY_CLI` が未設定なら `codex` | 例を `REVIEWER=$(pick_reviewer <claude|codex>)` と、引数に決めた driver をそのまま書く形にするか、`DRIVER=<claude|codex>` を同じ呼び出しに入れる。2 つの skill で同じ直しを入れる。 |
| LOW | exception-handling | L3 `/cross-review` の飛ばし方のうち、「差分が空」(手順 4 の 1 つ目の bullet)だけが、report を書かずに `/pr` へ進む。この PR は「飛ばしたときも triage の report を書く」(D3)を契約にし、`description`・`post-implementation-pipeline.md`・`definition-of-done.md`・README が「記録を残す」と言い切っているので、この経路だけ契約の外に残る。`/pr` は `not recorded` と書くので壊れはしないが、理由が残らず、insight event も出ない。差分が空になるのは branch に commit がないときで、まず起きないが、起きたときは記録のない飛ばしになる。 | `cross-review/SKILL.md:59`「if exit 0 (no diff), skip with a note and proceed to /pr」、`:5-12`(description)、`post-implementation-pipeline.md:13`、`definition-of-done.md:30` | その bullet を「skip, with the reason `no diff against <BASE>`: leave out step 5 and write the skipped report in step 6」に直し、手順 0 の「A skip does not call…」と手順 9 の「Skipped review」に並べる。記録に落とさないと決めるなら、文書側の「off か相手の CLI がないとき」の限定を、手順 4 の例外として 1 語足す。 |
| LOW | readability | L4 記録の語彙が 3 か所でそろっていない。(a) `/pr` の雛形は `Plan advisory: <ran \| incomplete (<reason>) \| skipped (<reason>)>` と 3 値だが、記録元は `/plan` が `no findings (<reviewer>)` と `ran with <reviewer>`、`/cross-review` が `Reviewer status: complete` で、どれを `ran` と書くかが `/pr` の文にない(`no findings` を `ran` と書くか、`not recorded` とするか迷う)。(b) `/plan` 手順 11 は結果を計画の「`Plan reviewed` の項目」に書くと言うが、`Plan reviewed` の box を付ける手順はどの skill にもない(`Plan approved` は 12.e が付ける)。skill の指示どおりに動くと、`[ ]` のままの項目の下に `plan advisory: …` が残る。今の計画では orchestrator が手で付けたので見えない。 | `pr/template.md:33-34`、`pr/SKILL.md:34`、`plan/SKILL.md:73`(`Plan reviewed` の語はこの 1 行だけ)、`plan/template.md:67` | (a) `/pr` の 34 行目に「`no findings`・`ran with …` と `complete` は `ran` と書く」の 1 文を足す。(b) 11 の冒頭に「その項目に記録し、飛ばした場合も box を付ける(助言の有無でなく、助言の段が済んだことを表す)」と書くか、項目を使わず「Progress checklist の末尾に 1 行」と書き直す。 |
| LOW | maintainability | L5 `/cross-review` の claude の道は、この PR で env の前置きと git の状態の確かめが入った行なのに、`/plan` 手順 11.c の claude の形との差が `--permission-mode auto` と道具だけとして記録されている。ほかに、`command` を付けない `claude`(alias の回避がない)、`</dev/null` なし、20 分の watchdog なし、出力を `-o` 相当の file に取らない(tool の標準出力の JSON)、がある。codex の道と `/plan` の claude の道は、この 4 つをそろえてあるので、同じ skill の 3 つの外部呼び出しのうち 1 つだけが、止まらない形になる。driver が Codex の `/cross-review` で `claude -p` が stdin を待つと、`rc=` も `-o` もないので、完了の契約の「incomplete」に落ちる前に固まる。旧い版からの形で、この PR が作った欠陥ではない。 | `cross-review/SKILL.md:69` と表(「CLI execution modes」の claude の行)、`plan/SKILL.md` 11.c の claude の形、`tests/test-codex-exec-invocation.sh` は codex の行だけを固定 | PR の Known gaps の 1 行(S3 の報告が「道具は今のまま、差を書く」と言っているもの)に、`command`・`</dev/null`・watchdog・出力の file も入れる。直すなら別の issue で、`/cross-review` の claude の道を `/plan` の形に寄せる(道具の制限は reviewer が差分を読むので別)。`tech-debt:154` の (c) を直す /sync-docs の行に合わせてもよい。 |
| LOW | readability | L6 `scripts/xreview-helpers.sh:64-66` のコメントを、文の途中で差し替えたため、折り返しが不揃いになった(59 字、44 字、81 字の順)。81 字の行は同じ file の 78 字前後の行より長く、`templates/base/scripts/xreview-helpers.sh` にも同じ形で写っている。 | `scripts/xreview-helpers.sh:64-66`(`awk` で 59 / 44 / 81) | 「which `RALPH_PRIMARY_CLI` (case-insensitive) overrides (see `.claude/skills/plan/SKILL.md` step 11.b and `.claude/skills/cross-review/SKILL.md` Step 2).」から折り直す。 |

## Positive notes

- `/plan` 11.a と `/cross-review` 手順 0 の終了コードの規則と記録の文言が、キー名以外は同じ。off、古い ralph、ralph がない、相手の CLI がない、設定の誤りの 5 つに、それぞれ別の記録または別の動き(止まる)がある。
- git の状態の確かめが 4 つの要素(HEAD、index、作業ツリー、未追跡の中身)に分かれていて、`git diff` と `git hash-object` を足した理由(porcelain だけでは、すでに変更済みか未追跡のファイルへの編集が見えない)が本文に書かれている。`/cross-review` は同じコマンドを `/plan` 11.c への参照で済ませずに、自分の file 名で書き直している。
- claude の起動形の注釈が、`--tools` と `--disallowedTools` が値を複数取るため prompt を `--output-format text` の直後に置く、という落とし穴を明記している。flag の説明は `claude --help` の 2.1.304 と合う。
- backfill の変更が、名前(`incomplete` → `noReview`)・コメント・テスト(単独の `skipped`、cycle 2 だけが `Skipped` の混在)まで一貫している。ルートの `ralph.toml` のテストは、ゼロ値の `Config` に読む理由(ファイル自身が 4 つを true にしている証拠にする)と、ファイルがないと `Fatalf` にする理由がコメントに書かれている。
- 手順の番号を変えず(`/cross-review` に 0 を足す形)、番号を名指す他の文書(insights の README、`paneenv.go`、`/plan` 11.c)を壊さない選択をしている。

## Notes(所見にしないもの)

- 未追跡の dangling symlink があると、`git ls-files … | xargs -0 git hash-object --` は `fatal: could not open '<name>' for reading` で落ち、同じ呼び出しに入った後ろのファイルの hash が出ない(一時 repo で確認)。before と after は同じ出力になるので誤って止まることはなく、後ろの未追跡ファイルへの編集が見えなくなるだけ。最後の守りの穴で、作業ツリーに未追跡の dangling symlink があるのは稀なので、所見にしない。
- git の状態が変わって「止まる」(11.d、手順 4)と、`diff` を見せたあとの手順が書かれていない。`.codex/config.toml` が codex の実行の前後で書き換わる未特定の事象(memory の `project_codex_config_rewrite`)は、この確かめが初めて捕まえる。起きたとき、人が `diff` を見て戻す前提で書かれているので、足りないとは言わない。
- `ralph config get` は cwd の `./ralph.toml` を読むので、skill が repo の root から打つ前提(両方の skill に書いてある)が要る。task worktree の root でなく main の checkout で打つと、ほかの branch の `ralph.toml` を読む。skill の括弧書き(「the task worktree root」)が止めている。

## Tech debt identified

| Debt item | Impact | Why deferred | Trigger to pay down | Related plan/report |
| --- | --- | --- | --- | --- |
| L5: `/cross-review` の claude の道が `command`・`</dev/null`・watchdog・出力の file を持たず、`/plan` 11.c の claude の形とそろっていない | driver が Codex の `/cross-review` で `claude -p` が止まると、完了の契約に届かず固まる | 旧い版からの形。reviewer が差分を読むので道具の制限は別で、形をそろえる変更は `/cross-review` の claude の道を実機で確かめる必要がある | この道で固まった報告、または #276 でこの道の flag を触るとき | `docs/plans/active/2026-10-29-review-switches-skills.md`(S3、Known gaps) |

_(If any rows were added above, also append them to `docs/tech-debt/`.)_ 上の 1 行は、/sync-docs が `tech-debt:154` の (c) を直す行と同じ項目なので、新しい行は足さず、この report に残した。直さずに PR の Known gaps にも書かない場合だけ、`docs/tech-debt/README.md` に足す。

## Recommendation

- Merge: OK(CRITICAL・HIGH・MEDIUM なし、LOW 6 件)
- Follow-ups:
  - 直すと安上がりなもの: L2(2 つの skill の 1 行ずつ、4 つの写しは `sync-skills.sh`)、L3(1 bullet)、L4(2 文)、L6(コメントの折り直し)。L2 と L3 は skill の動作に効くので、直すなら先にやる。
  - /sync-docs の待ちに足すもの: L1 の (a) `internal/config/config.go:30-31` と (b) `templates/base/ralph.toml:23-25`。(c) と help の「only that」は直さないなら Known gaps の 1 行。
  - L5 は PR の Known gaps に書くのが最小。
  - 直したときは skill の写しの一致(`check-skill-sync.sh`、`check-sync.sh`)と、skill の文を固定する test を通し直す。
