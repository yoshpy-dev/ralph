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

## Run 2: Run 1 の直し(d52b6175)のあと

- Date: 2026-10-29
- Reviewer: reviewer サブエージェント(`/self-review`、パイプラインの cycle 1 の 2 回目)
- Scope: `git diff ecfa05dd..d52b6175`(Run 1 の L1〜L6 の直し、9 ファイル、+520/-342)。依頼の HEAD 34f0e437 は、その上に計画の Progress へ直しの記録を 2 行足しただけ。直しのコミットが触ったのは leader の雛形、`prompts_leader_plan_test.go`、`/director` skill の 4 つの写し、`director_skill_sync_test.go`、`internal/org/driver/paneenv.go`、`herdr_test.go` で、どれも L1〜L6 のどれかに当たる。
- ID: Run 2 の新しい所見は L7(LOW)。Run 1 の L1〜L6 と番号は重ならない。

### Run 1 の所見のその後

| ID | 状態 | 確かめたこと |
| --- | --- | --- |
| L1 | 直った | `internal/org/prompts/leader.md:550` と `:552` の 2 つの jq が、どちらも `select(.enabled != false)` になった。`:543-545` に、欄がない server と `null` の server も有効と数えることを書いた。stand-in の `STUB_NOFIELD=1` の case(`prompts_leader_plan_test.go:1011`)は、欄のない `bare` と `null` の `blank` を出し、確かめの呼び出しに 2 つの `-c mcp_servers.<名前>.enabled=false` が載ることと、`mcp: not off (enabled: 2)` を見る。直す前の 2 行なら `bare` と `blank` はどちらの一覧にも入らず、n が 0 になって exec が走るので、この case は落ちる。名前の行だけを戻すと確かめの引数の検査で落ち、数の行だけを戻すと `enabled: 2` が出なくなって落ちる。2 行がそれぞれ固定されている。 |
| L2 | 直った | `/director` skill の「計画の承認」の手順 1(`.claude/skills/director/SKILL.md:767-770`)に、本文の `ADVISORY:` の要約(飛ばしたときはそのことと理由)を計画の要点と一緒に人に伝える文が入った。root と `templates/base/` の写しは `cmp` で一致し、`.claude` と `.agents` は frontmatter を除いて一致する。`directorSkillApprovalForkSteps` の step 1 にその文の断片があり(`internal/org/director_skill_sync_test.go:808`)、4 つの写しのすべてで照合される。 |
| L3 | 直った | (a) 雛形の全文に対する `つまで`・`つ目から` の禁止をやめ、`leaderForkCountCap`(`prompts_leader_plan_test.go:182`)を手順 1 の分かれ目の段落と手順 3 にだけ当てる。正規表現そのものを、打ち切る言い方 3 つと普通の言い方 3 つ(「返事を待つまで」「案が 3 つ以上なら」「1 つ 1 行で」)で先に確かめている。(b) `goStringLiterals`(`:427`)が `go/scanner` で `internal/cli/config.go` の文字列リテラルの値を取り出し、空白を除いて約束の文と照合する。引用符と改行の位置が違う 2 つの例で、helper 自身も確かめている。 |
| L4 | 直った | コード塊の切り出しは `fencedBlockAfter`(`:100`)1 つになり、手順 3 の例(字下げあり)と「計画助言」の 3 つのコード塊(字下げなし)の両方に使う。例の QUESTION を escalate と `ParsePlanHeader` に通す確かめは `leaderQuestionExample`(`:125`)に移り、2 本の test が共有する。節の外に出ない語の 2 本は `TestRenderRolePrompt_Leader_PlanWordingStaysInTheFeatureOrgSection`(`:313`)の 2 行の表になり、節の全文が 1 回だけ現れる確かめは両方の語の表に効くようになった。直しの前と後の test から文字列リテラルを取り出して比べると、消えたものは、場所が変わった言い回し(「手順 1 の」から「この節の終わりの」へ、Go のリテラルのままの help から空白を除いた文へ)と、まとめた helper の中身だけで、固定していた約束で抜けたものはない。 |
| L5 | 直った(L7 を参照) | `leader.md:510-514` が `pick_reviewer "${RALPH_PRIMARY_CLI:-<あなたの CLI>}"` と、`<あなたの CLI>` を自分の CLI に書き換えることを書く。`internal/org/driver/paneenv.go:68-70` の警告文の後ろに `and so may a feature org leader's plan advisory if the leader does not name its own CLI` が付き、`herdr_test.go:429-431` の定数が全文を、`internal/cli/org_runtime_env_test.go:174` が前半を固定する。`PrimaryCLIEnv` の doc comment(`paneenv.go:34-42`)も「計画助言」の小見出しを指す。test は pane の値と leader の CLI の 5 通りを bash と `zsh -f` で打つ。 |
| L6 | 直った | 計画助言は `### 計画助言`(`leader.md:491`)になり、手順 16 の後ろ、`## タスク` の前に置かれた。手順 1 は `:180-181` の案内の 1 文で終わり、手順 3 は `:210-211` で「この節の終わりの「計画助言」」を指す。手順 1 から小見出しまでの字下げのない行が、どれも番号つきの項目の始まりであることを `TestRenderRolePrompt_Leader_PlanAdvisoryIsASubsectionAfterTheSteps`(`:350`)が見るので、手順は 1 つの list のまま。番号は振り直していない。計画の本文の「手順 1 の最後」は digest を変えないために直さず、Progress に読み替えを書いてある。 |

### Evidence reviewed

- `git diff ecfa05dd..d52b6175` の全体、HEAD の `leader.md` の 110〜600 行(手順 1〜16 と「計画助言」)、`prompts_leader_plan_test.go` の全体(1,099 行)、`prompts_test.go` の `markdownSection`・`markdownItem`・`numberedItemStart`・`squashSpace`、`scripts/xreview-helpers.sh` の `pick_reviewer`、`internal/org/driver/herdr.go` の `TabCreate` と `warnNoPaneEnv`。
- 手で確かめたこと(worktree には何も書いていない):
  - codex の leader が書き換えた形 `pick_reviewer "${RALPH_PRIMARY_CLI:-codex}"` を、pane の値なし、`claude`、`codex` の 3 通りで bash で打った。結果は `claude`、`codex`、`claude`(L7)。
  - 直しの前と後の test の文字列リテラルを、scratchpad の `go run`(`go/scanner`)で取り出して `comm` で比べた(L4)。
  - `markdownItem` は次の `- ` か番号つきの行で項目を切るので、手順 16 の項目は今は「計画助言」の終わりまで伸びる。手順 16 の項目に「〜を含まない」を求める test はない(`prompts_test.go:456` は「最後のコマンド」を含むことだけを見る)。
  - 消えた helper と test の名前(`leaderStep1CodeBlock`、`TestRenderRolePrompt_Leader_ForkWordingStaysInTheFeatureOrgSection`、`TestRenderRolePrompt_Leader_PlanAdvisoryStaysInTheFeatureOrgSection`)は、この report のほかに repo のどこにも残っていない。
- 確かめていない: テスト、静的検査(skill の規則と依頼どおり、走らせていない)。

### Findings

| Severity | Area | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- |
| LOW | maintainability | L7 L5 の直しの `${RALPH_PRIMARY_CLI:-<あなたの CLI>}` は、pane に値があれば、leader 自身の CLI より先にその値を使う。ralph が `--env` を渡せない herdr では pane に ralph の値は入らないが、pane は herdr の server の環境を持ち、shell は login shell として起動するので、server を起動した人の環境に `RALPH_PRIMARY_CLI` があれば、その値が pane に残る(codex の recipe は、利用者にこの変数を設定するよう書いている)。そのとき codex の leader が `<あなたの CLI>` を `codex` と書き換えても、pane の `claude` が使われて codex を相手に選ぶ。Run 1 の L5 と同じく MCP を止める確かめは効き、相手のモデルの助言にならないだけで済む。Run 1 で私が勧めた形が、この抜けを持っていた。雛形の括弧の「`--env` のない herdr が作った pane にはなく」と、警告文の「if the leader does not name its own CLI」は、この場合を言い当てていない。 | `internal/org/prompts/leader.md:510-514`、`internal/org/driver/paneenv.go:68-70`、`internal/org/feature.go:388`(leader の pane は herdr の server の環境を持つ)、`docs/recipes/isolated-herdr-server.md:220`(pane の shell は login shell)、`docs/recipes/codex-setup.md:75-76`。`prompts_leader_plan_test.go:530` は pane の `codex` が leader の `claude` より先に使われる行を意図として固定している。bash の確かめ: pane の値が `claude` のとき `pick_reviewer "${RALPH_PRIMARY_CLI:-codex}"` は `codex` を出す | この cycle では文の直しにする。雛形の括弧を「`--env` のない herdr が作った pane には ralph の値がなく(herdr の server の環境の値が残っていることはある)、値がなければこの値で選ぶ」にし、警告文の後半を `and so may a feature org leader's plan advisory` で止めて条件を外す(`herdr_test.go:429-431` の定数も合わせる)。計画の AC5 の `pick_reviewer "$RALPH_PRIMARY_CLI"` と、pane の値を先に使う test の行は変えずに済む。leader は自分の CLI を必ず知っているので `pick_reviewer <あなたの CLI>` だけにする形のほうが確かだが、AC5 の字面と test の 1 行を変えるので、選ぶなら計画の Progress に逸れとして書く。 |

### Positive notes

- 6 件とも、直しの前の形に戻すと落ちる test がある(L1 の `STUB_NOFIELD`、L2 の断片、L3 の正規表現と `goStringLiterals` の自己確かめ、L5 の pane の値なしの 2 行、L6 の list の確かめ)。L1 は 2 つの jq のどちらを戻しても落ちる。
- L3 の直しは、禁止の範囲を狭めただけでなく、正規表現が当たる例と当たらない例を test の中で先に確かめてから雛形に当てている。
- L6 の移動で計画の本文(digest の対象)を変えず、読み替えを Progress に書いた。手順の番号も変えていない。
- 直しのコミットは Run 1 で挙げたファイルの外に出ておらず、変えた警告文の固定(`herdr_test.go` の全文、`org_runtime_env_test.go` の前半)も合わせて直している。

### Notes(所見にしないもの)

- 手順 1 の案内は「次に進む前に」と書き、小見出しは「split では手順 5 で implementer を spawn する前に」と書く。split では手順 2〜4 を飛ばすので次の手順は 5 になり、案内のほうが少し厳しいだけで、食い違いはない。
- `TestDirectorSkill_PinsTheApprovalForkSteps` は `FORK` の行の形を leader の雛形と突き合わせるが、`ADVISORY:` の語は突き合わせない。雛形の側は `prompts_leader_plan_test.go` が `ADVISORY:` を固定しているので、語を変える人はどちらかの test で気づく。
- `/sync-docs` に渡すもの: README の director の段落(`README.md:259`)、`/org` skill の `:31-34`、`docs/recipes/isolated-herdr-server.md:240-249` は、pane の `RALPH_PRIMARY_CLI` の用途を座席の cross-review とだけ書いている(leader の計画助言も読むようになった)。`/org` skill の「機能の計画の承認」に `ADVISORY:` がないことは、計画の Progress がすでに `/sync-docs` に回している。
- `/director` skill の手順 1 は、足した文のあとの「人が会話にいなければ、」で行が短く切れている。描画は変わらない。
- この report には `Merge:` の行が 2 つある。Run 1 の行は書き換えない依頼に従って残した。判定はどちらも yes。

### Coverage gaps

- Run 1 と同じ(本物の codex と claude での通し、Codex の leader の sandbox から `claude -p` が通るか)。
- L7 の前提(`--env` のない herdr の pane に、herdr の server の環境の `RALPH_PRIMARY_CLI` が残る)は、repo の記述(`feature.go:388`、recipe の login shell の段落)から導いたもので、herdr では確かめていない。

### Recommendation(Run 2)

- Merge: yes(Run 2。CRITICAL・HIGH・MEDIUM なし。Run 1 の L1〜L6 は直った。新しい所見は L7 の LOW 1 件)
- Follow-ups: L7 は文の 2 か所(雛形の括弧と警告文、`herdr_test.go` の定数)の直しで済む。直さないなら tech-debt に 1 行送る。

## Run 3: Run 2 の L7 の直し(cf0e5836)のあと

- Date: 2026-10-29
- Reviewer: reviewer サブエージェント(`/self-review`、パイプラインの cycle 1 の 3 回目)
- Scope: `git diff 4dcaeb27..cf0e5836`(L7 の直し、4 ファイル、+17/-13)。依頼の HEAD ba66ae9c は、その上に計画の Progress へ直しの記録を 2 行足しただけ(`git diff 4dcaeb27..ba66ae9c --stat` の 5 ファイルは、この 4 ファイルと計画)。触ったのは `internal/org/prompts/leader.md`、`internal/org/driver/paneenv.go`、`internal/org/driver/herdr_test.go`、`internal/org/prompts_leader_plan_test.go` で、どれも L7 に当たる。
- ID: Run 3 の新しい所見は L8(LOW)。L1〜L7 と番号は重ならない。

### Run 2 の所見のその後

| ID | 状態 | 確かめたこと |
| --- | --- | --- |
| L7 | 直った | (1) 雛形の括弧(`leader.md:512-515`)は、ralph が pane に値を入れること、`--env` のない herdr の pane には入らないこと、その場合も herdr の server の環境に同じ変数があれば pane がその値を引き継いでその値で選ぶこと、書き換えた自分の CLI が使われるのは変数がないときだけ、を順に書く。Run 2 の勧めた文のとおり。(2) 警告文(`paneenv.go:69-71`)は `and so may a feature org leader's plan advisory` で終わり、条件の句がない。`herdr_test.go:429-431` の `primaryCLIClause` と、連結した字面が同じ。`org_runtime_env_test.go:174` が固定する前半 `a seat's cross-review may pick the wrong CLI as its other model` は変わっていない。(3) `PrimaryCLIEnv` の doc comment(`paneenv.go:39-43`)は、計画助言が自分の CLI を使うのは変数がないときで、pane が server の環境から得た値はそれに勝つ、と書く(「unset」の字面は L8)。(4) test の pin(`prompts_leader_plan_test.go:515`)は雛形の括弧と字面が同じで、doc comment(`:503-509`)も「server にもない」場合を足した。pane の値が leader の CLI に勝つ行(`:531`、pane `codex`・leader `claude` → `claude`)は直し前からあり、今の文が言う動きを bash と zsh で固定している。 |

### Evidence reviewed

- `git diff 4dcaeb27..cf0e5836` の全体と、HEAD の `leader.md:490-535`、`paneenv.go:20-80`、`herdr_test.go:420-435`、`prompts_leader_plan_test.go:495-560`、`org_runtime_env_test.go:174`、`scripts/xreview-helpers.sh` の `pick_reviewer`。
- 手で確かめたこと(worktree には何も書いていない): `pick_reviewer "${RALPH_PRIMARY_CLI:-codex}"` を bash で、変数が unset のときと空のときに打つと、どちらも `claude`(leader の CLI の `codex` が使われる)。値が `claude` のときは `codex`(Run 2 の確かめと同じ)。L8 の根拠。
- `git grep -n 'does not name its own CLI'` は、計画の Progress の 2 行(`:169` は Run 1 の直しの記録、`:171` はそれが当時の文だと書いた行)と report だけに残る。コード、test、雛形、skill には残っていない。直しの前の括弧の句 `pane にはなく` は repo のどこにも残っていない。
- 確かめていない: テスト、静的検査(skill の規則と依頼どおり、走らせていない)。

### Findings

| Severity | Area | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- |
| LOW | readability | L8 `PrimaryCLIEnv` の doc comment の「only when the variable is unset」は、変数が空のときに当たらない。コマンドの `${RALPH_PRIMARY_CLI:-<あなたの CLI>}` は `:-` なので、unset のときだけでなく空のときも leader の CLI に落ちる。server を起動した人の環境に `RALPH_PRIMARY_CLI=` が空で入っている pane は、この comment からは pane の値が勝つように読めるが、実際は leader の CLI が使われる。害はない(落ちた先は leader 自身の CLI で、いつもの選び方になる)。雛形の「変数がないときだけ」は「ない」が空を含むとも読めるので、ここでは足りている。 | `internal/org/driver/paneenv.go:42`。bash の確かめ: unset と空のどちらでも `pick_reviewer "${RALPH_PRIMARY_CLI:-codex}"` は `claude`。`prompts_leader_plan_test.go:531-533` に空の値の行はない | comment を「only when the variable is unset or empty」にする。雛形は変えない(pin を動かさずに済む)。直さないなら対応は要らない。 |

### Positive notes

- 直しは Run 2 の勧めを最小の形で入れた。警告文の条件を外し、雛形の括弧に足し、comment と定数と pin を 1 つずつ合わせている。触ったのは L7 の 4 ファイルだけで、test の表は変えていない(pane の値が勝つ行は最初からあった)。
- 警告文の字面を持つ 3 か所(`paneenv.go`、`herdr_test.go` の定数、`org_runtime_env_test.go` の前半)のうち、前半だけを見る 1 か所は変わらず、全文を見る 1 か所は合わせて直っている。
- 直しの前の言い回し(`pane にはなく`、`if the leader does not name its own CLI`)は、計画の Progress の `:171` が「当時の文」と明記した 1 行を除いて残っていない。

### Notes(所見にしないもの)

- 警告文の `and so may a feature org leader's plan advisory` は、前の節の理由(`pick_reviewer` が seat の driver を claude とみなす)をそのまま受けるが、計画助言が外れるのは別の理由(server の環境から残った値)による。leader は変数がないとき自分の CLI を書くので、警告が出ても server に値がなければ計画助言は外れない。「may」の範囲に入るので直さない。Run 2 で私が勧めた形でもある。
- 計画の Progress の `:171` は、括弧の文を「」で引くが逐語ではない(「書いた CLI が使われるのは変数がないときだけ」。雛形は「書き換えたあなたの CLI が使われるのは、変数がないときだけ」)。引用符の使い方の問題で、読み違えは起きない。
- 雛形の括弧は 512-515 行で、行頭が `が、` になる行がある。この file は日本語の文を行の途中で折る(`:512` も同じ)ので、描画も pin(空白を除いて照合)も影響を受けない。

### Coverage gaps

- Run 1・Run 2 と同じ(本物の codex と claude での通し、Codex の leader の sandbox から `claude -p` が通るか)。
- L7 の前提(`--env` のない herdr の pane に、herdr の server の環境の `RALPH_PRIMARY_CLI` が残る)は、Run 2 のとおり repo の記述(`feature.go:388`、recipe の login shell の段落)から導いたもので、herdr では確かめていない。雛形はこれを事実として書いている。

### Recommendation(Run 3)

- Merge: yes(Run 3。CRITICAL・HIGH・MEDIUM なし。Run 2 の L7 は直った。新しい所見は L8 の LOW 1 件)
- Follow-ups: L8 は `paneenv.go:42` の comment 1 語(`unset or empty`)の直し。直さないなら対応は要らない。`/sync-docs` に渡すもの(README の director の段落、`/org` skill の 31〜34 行、`isolated-herdr-server.md` の 240〜249 行)は Run 2 のとおり。

## Run 4: main の取り込み(d59966a9)と test の直し(92554f77)のあと

- Date: 2026-10-29
- Reviewer: reviewer サブエージェント(`/self-review`、パイプラインの cycle 1 の 4 回目)
- Scope: `git diff 436e5896..a1f0f40a`(Run 3 のあとに積まれた分)。読んだ head は依頼どおり a1f0f40a で、レビュー中に動いていない。中身は 3 つ。(1) 取り込みの merge commit d59966a9。衝突は `internal/org/driver/paneenv.go` の 1 か所だけで、`git show d59966a9 -- internal/org/driver/paneenv.go` を読み、main 側の変更(`/plan` の計画助言、`/cross-review`、`ralph config get` の説明)がこの branch の file に届いているかを、branch 固有の差分 `git diff origin/main...HEAD` と突き合わせた。(2) 92554f77 の `internal/org/prompts_leader_plan_test.go`(+96/-29)。(3) a1f0f40a の plan の Progress 5 行(読んだだけ)。main から入った file そのものの出来は見ていない。
- ID: Run 4 の新しい所見は L9・L10(どちらも LOW)。L1〜L8 と番号は重ならない。

### Run 3 の所見のその後

| ID | 状態 | 確かめたこと |
| --- | --- | --- |
| L8 | 直った | `internal/org/driver/paneenv.go:34-43` の `PrimaryCLIEnv` の doc comment は、leader の計画助言が自分の CLI を使うのは変数が「unset or empty」のときだけで、pane が herdr の server の環境から得た値はそれに勝つ、と書く。雛形の `${RALPH_PRIMARY_CLI:-<あなたの CLI>}`(`leader.md:510`)の `:-` と合う。bash で `pick_reviewer "${RALPH_PRIMARY_CLI:-codex}"` を、空のとき `claude`、`CODEX` のとき `claude` で確かめた(空でも leader の CLI に落ちる)。 |

### Evidence reviewed

- 依頼の 1(merge の解決)
  - `paneenv.go:34-43` の全文を、main の `/plan` の手順 11.b(`.claude/skills/plan/SKILL.md:79-83`)、`/cross-review` の手順 2(`.claude/skills/cross-review/SKILL.md:52-55`)、`scripts/xreview-helpers.sh:57-78` の `pick_reviewer` と突き合わせた。main 由来の前半(「`/plan` の手順 11.b と `/cross-review` の手順 2 が読み、skill が driver を `pick_reviewer` に渡す」)は、2 つの skill の書き方と合う。足した後半(leader の計画助言も読む、`unset or empty` のときだけ leader 自身の CLI)は雛形の 計画助言 と合い、見出しの名前(`## 機能ごとの org` の `### 計画助言`)も `leader.md:110` と `:491` にある。
  - 衝突の外では、`git diff 436e5896..HEAD --stat -- internal/org internal/cli` が `paneenv.go` と `prompts_leader_plan_test.go` の 2 file だけを出す。main は `leader.md`、`/org` skill、`/director` skill、`director_skill_sync_test.go`、`internal/cli/config.go` を触っていない。`internal/config` の 3 file(`config.go` の `StandardConfig` の comment、`defaults_sync_test.go`、新しい `root_ralph_toml_test.go`)と `internal/insights` は main の変更で、branch が書き換えた `OrgReviewConfig` の comment(「leader は `org.review.plan_advisory` を読む」)とは矛盾しない。雛形が `/plan` の手順 11.c の指示文と見張りを写している部分は、`prompts_leader_plan_test.go:556-582` が main の SKILL.md に当たって通る(orchestrator の記録どおり)。
  - main が書き換えた `ralph config get` の失敗の扱い(`templates/base/ralph.toml:19-29`、`README.md:135`、`/plan` の 11.a、`/cross-review` の手順 1)を、雛形の手順 `leader.md:498-506` と、branch が足した `[org.review]` の説明(`templates/base/ralph.toml:252-258`)に突き合わせた。L9。
  - 雛形が `pick_reviewer` に渡す値を、main の 11.b の `case "$override" in claude|codex) ... esac` と比べた。L10。
- 依頼の 2(92554f77)
  - 差分の全体と、HEAD の `prompts_leader_plan_test.go:752-935`(stub、`advisoryWrites`、`advisorySleepStub`、`newAdvisoryFixture`、`run`、`runWriting`)と `:985-1168`(codex と claude の 2 本)。`gitOut` は `strings.TrimSpace` を返すので、`advisoryWrites` の `M tracked.txt` は ` M tracked.txt` の先頭の空白を落とした形。
  - hang の case: stand-in は `exec /bin/sleep 240` で、`$!` はその sleep そのもの。見張りの `kill "$cpid"` は SIGTERM で、`wait` が 143 を返す。外側の期限(`advisoryRunTimeout` 120 秒)に先に当たると、`run` が process group ごと SIGKILL して `t.Fatalf` するので、`codex rc=143` はシェルが生きたまま TERM を受けたときにしか出ない。見張りの kill を外すと 120 秒で落ち、`advisorySleepStub` が効かず本物の `sleep 1200` が走っても落ちる。どちらの失敗も 120 秒+`WaitDelay` 10 秒で終わり、suite は止まらない。
  - 書き換えの case: 追跡のファイルの書き換え(`git diff` の項)、新しい未追跡のファイル(`git status` の項と hash の項)、前から未追跡のファイルの書き換え(`git status` の行は前後で同じ `?? untracked.txt` で、hash の項だけが変わる)の 3 つは、それぞれ別の項に依る。codex の塊の `cmp` を外すと codex の書き換え 6 本(3 つ × bash と zsh)が、hash の項を外すと未追跡の 2 本が落ち、claude の塊も同じ。いつも `git: changed` にすると clean の case が `git: same` の確かめで落ちる。`runWriting` は stand-in が実際に変更したことを `git status` で先に確かめるので、stand-in が書けなかった case が通ることはない。これは読みの推論で、mutation は打っていない(implementer の結果は plan の Progress `:177`)。
  - `mcp)` の枝の `exit 0` は、一覧の呼び出しが末尾の `STUB_WRITES` の書き込みに落ちないための行。外しても before の snapshot が先に取られているので test は通るとみられる(読みのみ、未確認)。実害のない冗長さで終わる。
  - 後始末: 子は `Setpgid` の別 group で、`Cancel` が group ごと SIGKILL する。fixture は `t.TempDir()` で、書き換えの case が作る `new.txt` と `untracked.txt` もその中。ほかの file に触れない。
- 手で確かめたこと(scratchpad のみ、worktree には何も書いていない): `bash` で `. ./scripts/xreview-helpers.sh; pick_reviewer "${RALPH_PRIMARY_CLI:-codex}"` を打った結果は、`gemini` が `codex`、末尾に空白のある `claude ` が `codex`、`CODEX` が `claude`、空が `claude`。leader が claude のとき(`:-claude`)の `gemini` も `codex`。L10 の根拠。
- 確かめていない: テスト、静的検査、92554f77 の mutation(依頼どおり走らせていない)。

### Findings

| Severity | Area | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- |
| LOW | maintainability | L9 取り込んだ main は「`ralph config get` の失敗のうち off とみなすのは 2 つ(古い ralph の exit 1 と `unknown command "config"`、`ralph` が PATH にないときの shell の 127)」と決め、shipped の `templates/base/ralph.toml` の `[standard]` の説明、`README.md` の `ralph config get` の行、`/plan` の 11.a、`/cross-review` の手順 1 がそう書く。この branch が足した `[org.review]` の説明は、leader と reviewer が「exit code を `[standard]` の skill と同じに扱う」と書き、同じ段落で leader が `plan_advisory` を読むと言う。ところが leader の雛形は `ralph config get --help` の約束(`unknown command "config"` の exit 1 だけが off)に従うので、127 は「それ以外」になり、BLOCKED を上げて返事が来るまで止まる。BLOCKED を上げる `ralph org escalate` も `ralph` なので、`ralph` が PATH にない pane では上げられない。起きにくい(leader は `ralph org start` が作る pane で動く)が、利用者の `ralph.toml` に入る説明が、雛形の動きと食い違った。 | `templates/base/ralph.toml:22-25`(127 を off)と `:254-255`(「skills do with [standard] above」)、`internal/org/prompts/leader.md:498-506`(127 に当たる枝がない)、`README.md:135`、`.claude/skills/plan/SKILL.md:77`、`.claude/skills/cross-review/SKILL.md:33`。`prompts_leader_plan_test.go:459-495` は `ralph config get --help` の文だけを固定するので、どちらの動きでも通る。`internal/cli/config.go` の help は 127 に触れない(shell が返す値で、ralph は出せない)。 | 雛形の手順に「`ralph` が PATH になく shell が 127 を返したときも、off とみなして頼まず、そのことを書く」を足し、`ReadsTheSwitchFirst` の文の pin に 1 句入れる(README の「skills and seats」と `[org.review]` の説明が、そのまま正しくなる)。leader だけ厳しくするなら、`[org.review]` の説明に「leader は 127 を off とみなさない」と書く。 |
| LOW | maintainability | L10 main の `/plan` 11.b と `/cross-review` の手順 2 は、`RALPH_PRIMARY_CLI` が `claude` か `codex` 以外なら無視して、自分の CLI を driver にする(`case "$override" in claude\|codex)`)。leader の雛形は `pick_reviewer "${RALPH_PRIMARY_CLI:-<あなたの CLI>}"` で値をそのまま渡すので、別の値は `pick_reviewer` の未知の driver の枝で `codex` になる。claude の leader には合うが、codex の leader は自分と同じ codex を相手に選ぶ。ralph が pane に入れる値は `claude` か `codex` だけ(spawn が検証する)なので、この値が来るのは herdr の server の環境に残った場合に限り、害も L5 と同じ(MCP を止める確かめは効き、相手のモデルの助言にならないだけ)。ただ `PrimaryCLIEnv` の comment は「reads it as well」と、`/plan` と同じ読み方に見える書き方をし、main が直した `xreview-helpers.sh` の header は「Both skills pass the driver they resolved」と言う。leader は 3 つ目の呼び方で、どちらにも出てこない。 | `internal/org/prompts/leader.md:510`、`.claude/skills/plan/SKILL.md:79-83`、`scripts/xreview-helpers.sh:63-64` と `:70-77`、`internal/org/driver/paneenv.go:39-43`。bash の確かめは上の Evidence | 直すなら雛形の括弧に「`claude` と `codex` 以外の値は無視する」と書き、打つ形を `/plan` の 11.b の `case` に揃える(`PicksTheOtherCLI` の表に 1 行足す)。直さないなら `PrimaryCLIEnv` の comment に「別の値は `pick_reviewer` にそのまま渡り、`codex` が選ばれる」と 1 句足し、L9 と一緒に tech-debt に 1 行送る。 |

### Positive notes

- 衝突の解決は main の文を土台にし、足したのは leader の計画助言と `unset or empty` の 2 点だけ。main 側が直した前半(「skill が driver を `pick_reviewer` に渡す」)を壊していない。`herdr_test.go` と `org_runtime_env_test.go` が固定する警告文は、この merge で動いていない。
- 92554f77 は、/test が挙げた 3 つの穴(hang の case の 20 秒の上限、claude の塊の `git: changed`、未追跡のファイルの hash)をそれぞれ別の case で埋めた。hang の case は時間の上限を外し、見張りが TERM で止めたことを rc の値で見る形に替えたので、負荷で遅くなっても落ちず、見張りが欠ければ 120 秒で落ちる。
- `runWriting` が stand-in の書き込みを `git status` で先に確かめるので、書けなかった case が「`git: changed` になるはず」の側で偽に通る道がない。`advisoryWrites` の comment が、未追跡のファイルの書き換えだけ `git status` の行が変わらず hash の項だけが見ることを書いている。
- 変更は `prompts_leader_plan_test.go` の中だけで、共有の helper や他の test に触れていない。renamed した subtest(claude の `bash` / `zsh` が `bash/a clean review` などに)を名指しする file は repo にない(`git grep CommandRuns` で確認)。

### Notes(所見にしないもの)

- `paneenv.go:38-39` の main の文「`pick_reviewer` called without a driver ... picks codex when it is unset」は、同じ `:-` の展開で空のときも `claude` になる(上の bash の確かめ)。同じ comment の中で、この文は「unset」、足した文は「unset or empty」と書き分けている。前の文は main の書き方で、害はない。
- `advisoryWrites`(case の表)と `advisoryStubWrites`(stub の末尾の shell)は名前が 1 語違いで、後者は stub の末尾そのものなので、`advisoryStubTail` のほうが探しやすい。どちらも comment で説明されているので所見にはしない。
- stand-in は `Setpgid` で test 本体と別の group にある。test の binary が途中で殺された(`go test -timeout` の panic、Ctrl-C)ときは、走っていた run の本物の `sleep 1200` が最大 20 分残る。hang の case の `sleep 240` は見張りが 1 秒後に止めるので残らない。前からある性質で、92554f77 で増えたものではない。
- `/test` の report(`docs/reports/test-2026-10-29-leader-plan-depth.md:129-131`)の Test gaps 3 件は、92554f77 が埋めた。report はその前の版のまま「pass(…時間の上限が負荷に弱いことを Test gaps に残した)」と書く。/test を回し直すなら自然に更新される。回し直さないなら、/pr の known gaps に写す前に閉じた旨を 1 行足す。
- `/sync-docs` に渡すもの(Run 2・3 の 3 か所に加えて): `scripts/ralph-config.sh:10-18`(「two live consumers」として `/cross-review` と `/plan` だけを挙げる)と `.claude/rules/ralph/model-routing.md` の「Cross-review sync note」は、reviewer の model と effort の 3 変数の読み手に、機能の org の leader の雛形(`leader.md` の 計画助言、fallback は `prompts_leader_plan_test.go:650-680` が固定)を数えていない。

### Coverage gaps

- Run 1〜3 と同じ(本物の codex と claude での通し、Codex の leader の sandbox から `claude -p` が通るか、herdr の server の環境に `RALPH_PRIMARY_CLI` が残る pane)。
- 92554f77 の mutation と stand-in の実行は、依頼どおり打っていない。implementer の記録(`-count=5` で 120 本 PASS、変異 3 つ)は plan の Progress `:177` にある。

### Recommendation(Run 4)

- Merge: yes(Run 4。CRITICAL・HIGH・MEDIUM なし。Run 3 の L8 は直った。新しい所見は L9・L10 の LOW 2 件)
- Follow-ups: L9 は雛形の 1 文と pin の 1 句の足しで、同じ PR で直すのがよい。L10 は直さないなら comment 1 句。どちらも直さないなら、`docs/tech-debt/README.md` に 1 行(この PR は register を触っていない)。直しは雛形と test を動かすので、記録だけの例外には当たらず、`/self-review` からの再実行になる。cap の 2 回に収まるかどうかは orchestrator が決める。
