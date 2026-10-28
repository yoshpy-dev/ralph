# Self-review report: leader-plan-depth

- Date: 2026-10-29
- Plan: `docs/plans/active/2026-10-29-leader-plan-depth.md`(issue #267、S1〜S4、D1〜D7、AC1〜AC8、Progress の S2 の逸れ `--disable plugins`)
- Reviewer: reviewer サブエージェント(`/self-review`、パイプラインの cycle 1)
- Scope: `git diff origin/main...6099187d`(base 9837f744、12 コミット、15 ファイル、+1570/-53)。依頼の HEAD は 80d88e96 だったが、レビューの途中で 6099187d(古い codex が `--disable plugins` を知らないときに飛ばすことを雛形に書き、test に `STUB_OLD` の case を足したもの)が積まれ、雛形と test はそのあとの版を読んだ。中心は `internal/org/prompts/leader.md` の「機能ごとの org」の手順 1・3・4、新しい `internal/org/prompts_leader_plan_test.go`、`/director` skill の「計画の承認」と `/org` skill(それぞれ 4 つの写し)、`director_skill_sync_test.go`、コメントだけの `templates/base/ralph.toml`・`internal/config/config.go`・`internal/org/driver/paneenv.go`。命名、読みやすさ、null の扱い、例外処理、security、保守性の 6 項目で見た。機械的な項目(不要な変更、誤字、デバッグ残り、秘密情報、slice 外のファイル)は実装側が見る範囲なので探していない。
- ID: 所見は L1〜L6(すべて LOW)。

## Evidence reviewed

- `leader.md` の 110〜420 行(「機能ごとの org」の手順 1〜9)と、参照先の「件を上げる」「agmsg の受信箱の運用」の節。手順 1 の 3 つのコード塊(指示文、codex、claude)を 1 行ずつ読んだ。
- 手で打った確かめ(scratchpad の中だけ。worktree には何も書いていない):
  - `$!` が CLI そのものの PID か。`command fakecli ... & cpid=$!`(`if` の中)と `RALPH_WIP_CHECKPOINT=off command fakecli ... & cpid=$!` を bash、`zsh -f`、`/bin/sh` で打ち、子が書いた `$$` と `cpid` が 6 通りとも一致し、`kill "$cpid"` で rc=143、残る `sleep` はなかった。見張りの kill は相手に届く。
  - `codex --disable plugins mcp list --json`(codex-cli 0.163.0、モデルは呼ばない、出したのは数と型だけ): 14 件すべてに `enabled` があり、型は boolean だけ。名前に英数字・`_`・`-` 以外の文字はない。
  - jq の絞り込み: `enabled` のない項目は `select(.enabled)` で落ちる(L1)。
  - `claude --help`(2.1.304): `--restricted`、`--strict-mcp-config`、`--disallowedTools <tools...>`、`--tools <tools...>`、`--permission-mode` の `dontAsk`、`--output-format` がある。`--restricted` は user・project・local の設定ファイルを読まない(managed settings と `--settings` は効く)。指示文を `--output-format text` の直後に置くのは、`--tools` と `--disallowedTools` が可変長の引数をとるためで、test の doc comment の説明と合う。
- `scripts/xreview-helpers.sh` の `pick_reviewer`(引数が空なら `RALPH_PRIMARY_CLI`、それもなければ claude とみなして codex を返す)と `scripts/ralph-config.sh`(読み込むと既定値の代入と export だけで、`exit` は検証関数の中だけ)。`.gitignore` と `templates/base/.gitignore` の 47 行目に `.harness/state/` がある。
- `internal/cli/config.go:50-66` の `ralph config get` の help。手順 1 の終了コードの分け方と合う。
- `/director` skill の「件の処理」「答える順」「計画の承認」、`/org` skill の 860〜960 行。`/org` の「上の 5 の直しの DECISION」は同じ節の人の手順 5 を指していて正しい。4 つの写しは `cmp`(root と `templates/base/`)と frontmatter を除いた `diff`(`.claude` と `.agents`)で一致した。
- `prompts_leader_plan_test.go` の全体(949 行)と、使う helper(`runGit`、`gitOut`、`writeTestExecutable`、`renderSeatPrompt`、`directorSkillRepoRoot`)。stand-in の codex と claude は `t.TempDir()` の `bin` に置き、PATH の先頭に入れる。`command codex` と `command claude` はそこに当たり、本物の CLI と `~/.codex` には触れない。shell は `zsh -f` と素の `bash -c` で、zsh の rc ファイルは読まない。1 回の実行は 60 秒の context で process group ごと止め、`WaitDelay` は 10 秒。見張りの `sleep 1200` は hang の case だけ 1 秒の stub に替える。ほかの case では本物の `sleep 1200` が走るが、codex の stub がすぐ終わるので `kill "$spid"` で止まる。見張りの subshell の `sleep 1` は最大 1 秒残り、出力の pipe を持つので、`CombinedOutput` がその分だけ待つ(止まりはしない)。
- `director_skill_sync_test.go` の差分。手順の番号を 3→4 に動かした pin と、新しい `TestDirectorSkill_PinsTheApprovalForkSteps`。
- 確かめていない: テスト、静的検査(skill の規則と依頼どおり、走らせていない)。leader と director の実機の通し(計画の Non-goals)。

## Findings

| Severity | Area | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- |
| LOW | security | L1 codex の MCP の止め方の確かめが、`enabled` の欄のない一覧では「止まった」と読む。名前の行も確かめの行も `select(.enabled)` で絞るので、`enabled` が欠けた項目や null の項目は、止める対象にも数える対象にも入らない。そのとき `n` は 0 になり、その server が動いたまま `codex exec` が走る。今の codex 0.163.0 では 14 件すべてに boolean の `enabled` があるので起きない。ただ計画のリスクの表は「codex の MCP の止め方が codex の版で変わる: 止めたあとの一覧の確かめで気づき、計画助言を飛ばす(止まらないまま呼ぶことはない)」と書いている。版で欄の名前が変わる(`status` など)のは、この確かめが受け持つはずの変化そのもの。 | `internal/org/prompts/leader.md:234` と `:236`。jq の確かめ: `[{"name":"alpha"},{"name":"beta","enabled":false}]` に `[.[] \| select(.enabled)] \| length` は `0`、`select(.enabled != false)` は `1`。計画の Risks の 10 行目 | 2 つの jq をどちらも `select(.enabled != false)` にする。欄のない server も `-c ...enabled=false` を付けられ、codex が `false` と答えない限り数に残るので、止められなければ `mcp: not off` に倒れる。stand-in の codex に `enabled` を出さない case を 1 つ足すと固定できる。 |
| LOW | maintainability | L2 leader が書く `ADVISORY:` の行に、読む側の手順がない。leader の雛形は、計画助言の要約(飛ばした理由を含む)を承認の QUESTION の本文の `ADVISORY:` の行に書かせる。計画は「leader が所見を都合よく捨てる」の対策を「QUESTION の本文で人に見せる」にしている。ところが `/director` skill の「計画の承認」の手順 1 が人に伝えるのは「`PLAN:` の計画の要点」で、`ADVISORY:` には触れない。`/org` skill の「機能の計画の承認」にもない。`FORK` の行は、書く側と読む側の両方が test で固定されているのと比べて、片側だけになっている。 | `internal/org/prompts/leader.md:306-308` と `:319`、`.claude/skills/director/SKILL.md:764-769`、`git grep -n 'ADVISORY' .claude/skills` は 0 件 | 手順 1 に「本文に `ADVISORY:` の行があれば、その要約(飛ばしたときはその理由)も人に伝える」と 1 句足し、`directorSkillApprovalForkSteps` の step 1 の断片に入れる。 |
| LOW | maintainability | L3 test の否定の pin と lock-step の pin が、意図と関係のない書き換えで落ちる。(a) `TestRenderRolePrompt_Leader_FeatureOrgQuestionListsEveryFork` は雛形の全文で `つまで` と `つ目から` を禁じる。「返事を待つまで」「approve を打つまで」のような普通の言い回しも当たり、落ちたときの文は「分かれ目の数を打ち切っている」になる。`/org` skill にはすでに「打つまで」「1 つまで」があり、leader の雛形には「待つ」が 9 回ある。(b) `TestRenderRolePrompt_Leader_FeatureOrgPlanAdvisoryReadsTheSwitchFirst` は `internal/cli/config.go` の Go のソースを、文字列リテラルの引用符と `\n` と折り返しの位置まで含めて照合する(`unknown command \"config\" on stderr. Treat that, and only that, as\n` など)。help の文を折り直すだけで、約束が変わらなくても落ちる。 | `internal/org/prompts_leader_plan_test.go:117-124`、`:322-333`、`.claude/skills/org/SKILL.md:227`(「1 つまで」)と `:936`(「打つまで」) | (a) は禁止を手順 1 の分かれ目の段落と手順 3 に絞るか、`分かれ目` の近くの `\d+ ?つ(目から\|まで)` の正規表現にする。(b) は `" +\n\t\t\t"` の継ぎ目と `\n` を外してから、`Treat that, and only that, as off` のような文で照合する。 |
| LOW | maintainability | L4 新しい test file の中に同じ処理が重なっている。コード塊を切り出して字下げを外す処理が 3 か所にあり(手順 3 の例を S1 の test は自前で切り、S2 で足した `leaderQuestionExample` を使っていない)、例の QUESTION を `validateEscalation` と `ParsePlanHeader` に通す確かめも 2 つの test で同じ。「機能ごとの org の節の外に出ない」test も 2 本あり、片方だけが節の全文が 1 回だけ現れることを確かめる。手順 3 の例の形を変える人は、3 か所を同じように直すことになる。 | `internal/org/prompts_leader_plan_test.go:139-150`、`:274-284`、`:557-567`。`:152-158` と `:584-590`。`:219-237` と `:608-623` | 切り出しを 1 つの helper(`leaderStep1CodeBlock` を手順の marker で引けるようにしたもの)にまとめ、例の QUESTION の 2 本の test は `leaderQuestionExample` を共有する。節の外に出ない語の 2 本は、語の表 1 つの test にまとめる。 |
| LOW | security | L5 pane に `RALPH_PRIMARY_CLI` がないとき、codex の leader が自分と同じ codex を相手に選ぶ。`--env` のない herdr では、ralph は pane の env をすべて付けずに作り、警告だけを出す(`herdr.go:247`)。そのとき `pick_reviewer ""` は claude とみなして codex を返す。MCP を止める確かめは効くので安全は保たれるが、「相手のモデルの助言」にはならず、ADVISORY の行にもそのことは出ない。また、この PR は `PrimaryCLIEnv` の doc comment を「計画助言も読む」に直したが、同じ file の実行時の警告文は今も「a seat's cross-review may pick the wrong CLI」だけを言う。#265 の self-review の L1 で、計画助言がまだ `pick_reviewer` を使わないために外した句が、今度は要るようになった。 | `internal/org/prompts/leader.md:195-199`、`scripts/xreview-helpers.sh:69-77`、`internal/org/driver/paneenv.go:39-40`(直した doc comment)と `:65-67`(直していない警告文)、`internal/org/driver/herdr.go:243-248` | 雛形で、`RALPH_PRIMARY_CLI` が空なら自分の CLI(Claude Code なら `claude`、Codex なら `codex`)を `pick_reviewer` に渡すと書く(例: `pick_reviewer "${RALPH_PRIMARY_CLI:-<あなたの CLI>}"`)。警告文は `a seat's cross-review or a feature leader's plan advisory may pick ...` にする。`org_runtime_env_test.go` が見る部分文字列 `may pick the wrong CLI as its other model` は変わらない。 |
| LOW | readability | L6 手順 1 が 1 つの list item で 135 行になった。中身は、計画を書く手順、分かれ目の段落、計画助言の 5 段落(設定の読み方、相手の選び方、打ち方、出力の分け方、所見の扱い)と 3 つのコード塊。手順 3 と手順 4 は「手順 1 の分かれ目」「手順 1 の計画助言の要約」とこの塊の中を指し、手順 1 の中にも「下の要約」という前方参照がある。番号を振り直さない判断(Progress の S1・S2)は妥当だが、leader が手順 2 に進む前に読み終える量が、ほかの手順の 5〜10 倍ある。 | `internal/org/prompts/leader.md:143-277`(手順 2 は 278 行目から)、`:189`(「下の要約」)、`:300`、`:306` | 計画助言を「機能ごとの org」の下の小見出し(例: `### 計画助言`)に移し、手順 1 は「`plan_advisory` が on なら『計画助言』の節のとおり頼む」の 1 文にする。番号は変わらない。test の `leaderStep1CodeBlock` は節を引く形に変える。今回直さないなら、次に手順 1 に段落を足す PR で行う。 |

## Positive notes

- 見張りの形が `/plan` の手順 11.c と同じで、test がその文字列を `/plan` の SKILL.md と突き合わせる(`leaderAdvisoryWatchdog`)。`$!` が CLI の PID になることは bash・zsh・sh で手で確かめた。
- MCP の server の名前を 1 行ずつ `set -- "$@" -c ...` に積むので、名前に空白があっても 1 語のまま渡る。test の stand-in は `has space` という名前の server と plugin の server を出し、codex 0.163.0 が plugin の server の上書きで設定を読めなくなる動きも真似ている。古い codex(`--disable plugins` を知らない)、止まらない server、書き込む相手、止まる相手の 4 通りがどれも「呼ばない」か `git: changed` に倒れることを、bash と zsh で固定している。
- 確かめの失敗(jq がない、一覧が読めない、`codex` が非 0 で終わる)では `n` が空になり、`[ "$n" = 0 ]` が偽になって呼ばない側に倒れる。L1 の欄の欠け以外は fail-closed になっている。
- git の前後の状態は `.harness/state/` に書き、`git status` が無視する場所なので、比べる対象に自分のファイルが入らない。`cmp` が失敗したとき(後ろのファイルを書けなかったときを含む)は `git: changed` に倒れる。
- director の「計画の承認」の手順を 1 つ足して番号を振り直したとき、既存の test の pin(手順 3→4)と doc comment も合わせて直している。`FORK` の行の 2 つの形を、leader の雛形と skill の両方に入っていることで固定する test がある。
- stand-in の test は、`RALPH_` と `GIT_` で始まる env を外し、`git init --template=` で利用者の template の hook を避けている。

## Notes(所見にしないもの)

- `FORK` の行の区切りは ` / ` で、`/` を含む案(`internal/auth/` のようなパス)は前後に空白がないので区切りとぶつからない。test の正規表現 `[^/]*` は例の行だけに使っていて、leader が書く行を縛るものではない。
- 2 往復目の手順 4 の「すべての分かれ目が推奨の案で決まったことも表す」は、`(決定済み)` の行が推奨と違う案で決まっていても成り立つ書き方ではないが、そこから導く動き(`## Design decisions` を書き換えない)は 2 往復目でも正しい。
- 推奨どおりの答えで、承認だけを断った人の直しの依頼では、director は `FORK` の答えを本文に書かないので、leader は次の QUESTION でも同じ分かれ目を聞く。手間は増えるが、誤りにはならない。
- 承認の QUESTION の本文が言い直しても 2,000 字に収まらないときの手が書かれていない。分かれ目は 3 条件をすべて満たすものに限るので、1 行 60〜100 字で 20 行に届くことはまずない。

## Coverage gaps

- 本物の codex と claude で手順 1 のコマンドを打つ通しは、計画の Non-goals で、ここでも打っていない(AC8 の確かめは Progress に記録がある)。
- Codex の leader の sandbox から `claude -p` が通るかは確かめていない(Non-goals、通らなければ飛ばして記録する設計)。

## Recommendation

- Merge: yes(CRITICAL・HIGH・MEDIUM なし。LOW 6 件)
- Follow-ups: L1(jq の 2 行の直し)と L2(director の 1 句)は小さいので、この PR の中で直すのがよい。L3〜L6 は直すか、tech-debt に送るかを orchestrator が決める。
