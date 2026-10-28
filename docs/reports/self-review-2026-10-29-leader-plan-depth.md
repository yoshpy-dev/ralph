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
