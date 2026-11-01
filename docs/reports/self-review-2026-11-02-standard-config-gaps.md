# Self-review report: standard-config-gaps

- Date: 2026-11-02
- Plan: `docs/plans/active/2026-11-02-standard-config-gaps.md`(PR #291 の Known gaps、S1〜S3、D1〜D5、AC1〜AC6)
- Reviewer: reviewer サブエージェント(`/self-review`、パイプラインの cycle 1)
- Scope: `git diff origin/main...HEAD`(base d217c575、HEAD 2066d854)。コードは S1 b71354d0、S2 7b9dc257、S3 2d4baa2d の 3 コミット(`internal/cli/org.go`、新しい `org_start_plan_config_test.go`、`org_feature_reviewer_test.go`、`org_feature_test.go`、`tests/test-ralph-config.sh`)。diff の質だけを見た。テスト、静的解析、仕様適合、文書の drift は見ていない。依頼で個別に判断を求められた 3 点は「依頼された 3 点の判断」に書いた。
- ID: Run 1 の所見は L1〜L7、Run 2(S4)の所見は R2-L1〜R2-L4。branch 全体で LOW 11 件(Run 1 が 7、Run 2 が 4)、CRITICAL、HIGH、MEDIUM はない。

## Evidence reviewed

- `internal/cli/org.go` の diff を全部読み、変更した関数の周り(`newOrgSpawnRuntimeAt`、`withMainWorktreeOrgLimits`、`loadMainWorktreeConfig`、`newOrgRuntimeAt`、`newOrgRuntimeFromConfig`、`wireOrgInbox`、`resolveOrgConfig`、`resolveConfig`、`resolveOrgStartConfig`、`runOrgStartPlan`)を現物で読んだ。
- 設定を読む呼び出し元を grep した。`config.Load(` の非テストの呼び出しは `config.go:101` と `doctor.go:106` だけで、どちらも org の読み込みではない。`resolveOrgConfig` / `resolveConfig` / `loadMainWorktreeConfig` を通る org の読み込みは、すべて `loadConfigFile` を通る(`org.go:361,676`)。
- `newOrgSpawnRuntimeAt` の呼び出し元は `org.go:893`(spawn)と `:1099`(start <task>)の 2 つ。`newOrgRuntimeAt` の非テストの呼び出し元は 10 か所(grep で数えた。plan は 11 と書く)で、変わっていない。
- 新しいテスト `org_start_plan_config_test.go` を全部読み、使っているヘルパー(`newFeatureRepo`、`startOrgFeature`、`runLedgerTestGit`、`setupOrgStubPATH`、`readLogLines`、`readManifestEvents`、`eventTypes`、`writeTestFile`)の定義を確かめた。spawn の双子のテスト `TestOrgSpawn_InFlightSeatRefusal_StopHintRepeatsStateDir`(`org_test.go:2549`)と見比べた。
- S1 の移動: `b71354d0` で `org_feature_test.go` から消えた行と、新しい `org_feature_reviewer_test.go` の行を `sort` して `comm` で突き合わせた。旧にだけある行は 0。新にだけある行は `package`、`import`、括弧、空行だけ。
- `tests/test-ralph-config.sh` の diff と、引用のない単語の並びを回す `for` の箇所(161、177、188、317、322、326、406、458、483 行)を読んだ。
- 消した `resolveOrgStartReviewer` の残りの参照を `docs`、`.claude`、`templates` で grep した。残るのは過去の plan・report・evidence と、`docs/tech-debt/README.md` の 148 行(plan が /sync-docs の担当と記録済み)。

## Findings

<!-- Area recommended values: naming, readability, null-safety,
     exception-handling, security, maintainability -->

| Severity | Area | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- |
| LOW | maintainability | L1: `InboxHintStateDir` を設定する行が 2 か所に分かれた。「start --plan の runtime は spawn の runtime と同じ初期化を受ける」という不変条件が、コードでなくテスト 1 本(`TestOrgStartPlan_InFlightLeaderRefusal_StopHintRepeatsStateDir`)で守られている。`newOrgSpawnRuntimeAt` に初期化が増えたとき、start --plan に写し忘れる。plan advisory の所見 1 が突いたのと同じ型。 | `org.go:169`(`newOrgSpawnRuntimeAt`)と `org.go:1208`(`runOrgStartPlan`)に同じ `rt.InboxHintStateDir = orgInboxHintStateDir(cmd, resolvedStateDir)`。doc は `org.go:154-159` と `:1190` が「同じ値を設定する」と言葉で結んでいる。 | `newOrgRuntimeFromConfig` に `InboxHintStateDir` を設定する薄い包み(例: `newOrgSpawnRuntimeFromConfig(cmd, resolvedStateDir, orgCfg)`)を 1 つ足し、`newOrgSpawnRuntimeAt` は「読む → 上限を当てる → 包みを呼ぶ」にして、両方がそれを呼ぶ。上限を当てる順が runtime の作成より前になるが、`AgmsgHome` は上限の影響を受けないので結果は同じ。直さないなら `org.go:169` に「`runOrgStartPlan` が同じ設定を繰り返す」と 1 行書く。 |
| LOW | readability | L2: 構築の場所が `newOrgRuntimeAt` から `newOrgRuntimeFromConfig` に移ったのに、2 つの doc が前の名前のまま。同じ関数の `wireOrgInbox` の doc だけが新しい名前に直っていて、そろっていない。 | `org.go:441`(「copied into every org.Org this package constructs (newOrgRuntimeAt)」)、`org.go:466`(「InboxPollInterval (newOrgRuntimeAt)」)。実際にコピーするのは `org.go:407` と `:416-418` で、`newOrgRuntimeFromConfig` の中。start --plan はこの 2 つの変数を `newOrgRuntimeAt` を通らずに受ける。 | 2 か所とも `newOrgRuntimeFromConfig` に直す。 |
| LOW | readability | L3: doc の向きと言い回しが 3 か所で読みにくい。(a) `resolveConfig` の doc は「`resolveOrgConfig` に全部の table を足した形」と言うが、コードは逆で `resolveOrgConfig` が `resolveConfig` の包み(`org.go:659-665`)。(b) `wireOrgInbox` の doc の「`newOrgRuntimeFromConfig` (newOrgRuntimeAt's)」は何を指すか 1 度では読めない。(c) `newOrgSpawnRuntimeAt` の最初の文「(spawn と start <task>)、which with start --plan are the only ones that check the org-wide limits: it also replaces …」は、`it` が関数に戻る前に `the only ones` で切れる。 | (a) `org.go:667-670`。(b) `org.go:430`。(c) `org.go:145-150`。 | (a) 「`resolveOrgConfig` が [org] だけを取り出す元」と向きを直す。(b) 「`newOrgRuntimeFromConfig`(`newOrgRuntimeAt` が呼ぶ)」。(c) 「spawn と start <task> が使う。`start --plan` も上限を検査する(別の経路、下を参照)。この関数は …を置き換える」のように 2 文に分ける。 |
| LOW | maintainability | L4: `resolveOrgStartConfig` の doc は「reading each file once」と言うが、cwd が main worktree の root(普通に打つ場所)のとき、同じ `ralph.toml` を `resolveConfig` と `loadMainWorktreeConfig` の 2 回読む。plan の AC2 はこれを許しているが、doc とテストのどちらにも書かれていない。テストの cwd は subdirectory だけで、2 回になる場合は固定されていない。決まる値は壊れない(承認方式と `[reviewer]` は main の 1 回の読み込みから取る)が、「1 回」の読者は 409 行の元の苦情(2 回読む)が普段の場所では残ると気づけない。 | doc は `org.go:684-695`。テストは `org_start_plan_config_test.go:86,92-94`(`want` は `sub/ralph.toml` が 1、main の `ralph.toml` が 1)。cwd = root の case なし。plan の AC2 の括弧書きは `docs/plans/active/2026-11-02-standard-config-gaps.md:79`。 | doc に「cwd が main worktree の root なら同じファイルを別の役で 2 回読む」を 1 文足す。テストに cwd = root の case を 1 つ足し、`want` を 2 にして、3 回目の読み込みが入ったら落ちるようにする。 |
| LOW | readability | L5: `TestOrgStartPlan_ReadsEachConfigFileOnce` の case の `ledger string`("main" か "outside")は 2 値だけで、比較が 2 か所に文字列で書かれている。`else if tc.ledger == "main"` は、`"outside"` でない case がすべて "main" なので条件が常に真。綴りを間違えると、コンパイルが通ったまま別の分岐に入る。 | `org_start_plan_config_test.go:64,77,92`。 | `outsideLedger bool` にして `else if` を `else` にする。 |
| LOW | naming | L6: `TestOrgStartPlan_UnreadableConfigStartsNothing` の case は権限で読めないファイルでなく、壊れた TOML(`"[org\nmax_seats = 5\n"`)。決定的で root でも落ちるので選び方は良いが、名前と doc の「読めない」は実際の入力と合っていない。`--config` の case は先頭 `org: load config: ` だけを見るので、原因の文が付いているかは固定されない。 | `org_start_plan_config_test.go:109-122,130,147`。`config.Load` は壊れた `ralph.toml` の TOML エラーにパスを付けない(`internal/config/config.go:891-893`)ので、`--config` の case のエラーはファイル名を含まない。AC3 の文面は「エラーで止まり何も作らない」までなので足りている。 | 名前か doc に「壊れた TOML で代用する」と書く。`--config` の case に `strings.Contains(err.Error(), "toml")` を足すと、別の理由で同じ接頭辞のエラーが出ても通らない。 |
| LOW | maintainability | L7: start --plan の `--state-dir` 案内のテストは `--state-dir` の枝だけ。spawn の双子(`org_test.go:2549`)には、台帳が個人設定の `[org].state_dir` から来たとき `--state-dir` を付けない案内を確かめる 2 つ目の subtest がある。start --plan で「常に `--state-dir` を付ける」に壊れても、新しいテストは通る。テストの網羅の判断は /test の担当なので、書き方の指摘にとどめる。 | `org_start_plan_config_test.go:187-222`、`org_test.go:2565-2582`(`usePersonalStateDir`)。 | 双子と同じ `[org].state_dir` の subtest を足す(`usePersonalStateDir` が使える)。 |

### 依頼された 3 点の判断

1. `resolveOrgStartReviewer` を消して `resolveOrgStartConfig` にした形: 読みやすい。読む順(cwd か `--config` が先、main が後)が doc と一致し、エラーは 2 回とも早期 return で、どちらも読めなかった値を名指す。`!ok` のときの `cfg.Org, cfg.Reviewer` と、`ok` のときの `applyMainWorktreeOrgLimits(cfg.Org, mainCfg.Org), mainCfg.Reviewer` の分岐は 1 画面に収まる。3 つの戻り値は許容。2 つをまとめる構造体を作るほどではない。決まる値は元と同じで、元の `withMainWorktreeOrgLimits` が `rt.Config`(= `resolveOrgConfig` の結果)に当てていた値と、`applyMainWorktreeOrgLimits` が `cfg.Org` に当てる値が同じ。
2. `newOrgRuntimeFromConfig` と薄い包み `newOrgRuntimeAt` の分け方: 妥当。`newOrgRuntimeFromConfig` に失敗する道がないので error を返さず、`newOrgRuntimeAt` の呼び出し元は変わらない。名前は `newOrgRuntimeAt(resolvedStateDir, configPath)` と並べて `(resolvedStateDir, orgCfg)` の順がそろっている。残る点は L1(初期化の 2 か所)と L2(古い名前の doc)。
3. `org.go` の doc コメント: start の読み方について、doc 同士が食い違う所はない。`withMainWorktreeOrgLimits`(`org.go:200-202`)、`loadMainWorktreeConfig`(`:347-348`)、`resolveOrgStartConfig`(`:684-695`)、`runOrgStartPlan`(`:1184-1197`)は、「cwd か `--config` を 1 回、main を当てはまるとき 1 回、上限・承認方式・`[reviewer]` は main から」で一致している。残るのは L2(古い名前)、L3(言い回し)、L4(「各ファイル 1 回」の例外)。

### S2 の `loadConfigFile` の差し替えの安全

- 本番は `config.Load` のまま(`org.go:682`)。差し替えるのはテストの `countConfigReads` だけで、`orig` を取って `t.Cleanup` で戻す。
- 差し替えるテストは `t.Chdir` を使い、`t.Parallel` を呼べない(Go が panic する)ので、他のテストと競合しない。数える map は 1 つの goroutine だけが書く(監視は別 process)。
- 差し替えが効く範囲は `resolveConfig`(`resolveOrgConfig` 経由で `status.go:308`、`org_director.go:395`、`org_director_headless.go:303`、`newOrgRuntimeAt` の呼び出し元)と `loadMainWorktreeConfig` で、doc が書く 2 つより広いが、全部 org の読み込みで害はない。`config.go:101` と `doctor.go:106` は通らない。
- 名前は `absPath` など、この repo の他の seam と同じ型(変数に関数を入れる)。

## Positive notes

- 上限と承認方式を main の値で上書きする規則が `applyMainWorktreeOrgLimits` の 1 か所に集まり、spawn(`withMainWorktreeOrgLimits`)と start --plan(`resolveOrgStartConfig`)が同じ関数を呼ぶ。上書きする 3 つの値を足したり減らしたりするとき、片方だけ直す余地がない。
- main の `ralph.toml` が読めないときの文が 1 つになり、読もうとした 4 つの値(`max_orgs`、`max_total_seats`、`[org.approval].mode`、`[reviewer]`)を全部名指す。spawn の文(3 つの値)とは名指す値だけが違い、書き方がそろっている。
- 回数のテストは、main の ledger、`--config`、main の外の ledger の 3 つで「読まない」側(`--config` では main を、外の ledger では main を読まない)も固定している。
- 読めない設定のテストは、worktree、branch、`ralph-worktree.sh` の記録、manifest、herdr の 5 つについて「作らない」を確かめ、壊れた `ralph.toml` を commit して checkout を clean に保ち、別の理由で止まった場合を接頭辞の検査で除いている。
- S1 は純粋な移動。消えた行で新ファイルにない行は 0、新ファイルにだけある行は `package`、`import`、括弧、空行だけ。
- S3 の 1 行は、理由(zsh は引用のない `$var` を `sh_word_split` のときだけ分ける)と、実行の仕方(`zsh -f tests/test-ralph-config.sh`)を同じ場所に書いている。zsh のときだけ入るので、sh・bash・dash の動きは変わらない。
- 秘密、デバッグの残り、範囲外のファイルは見つからなかった。

## Tech debt identified

| Debt item | Impact | Why deferred | Trigger to pay down | Related plan/report |
| --- | --- | --- | --- | --- |
| なし(L1〜L7 は PR の中で直せる LOW) | - | - | - | - |

_(`docs/tech-debt/README.md` の 148 行が、消した `resolveOrgStartReviewer` を名指している。plan が /sync-docs の担当と記録済み。406・409・410 行の書き換えも /sync-docs。)_

## Coverage gaps

- 回数のテストが cwd = main worktree の root を打たない(L4)。root の case が、元のコードで落ちるか、新しいコードで 2 回になるかは未確認。
- `loadConfigFile` の差し替えが効く範囲(上の節)は、`rg` で呼び出し元を見た範囲。動的に呼ぶ箇所は見ていない。
- `zsh -f` での実行、`shellcheck`、sh・bash・dash の結果は、この review では打っていない(plan の S3 の記録を信頼する)。/verify の担当。

## Recommendation

- Merge: yes(branch 全体の判定。Run 1 と Run 2 を合わせて CRITICAL 0、HIGH 0、MEDIUM 0、LOW 11。L1〜L7 と R2-L1〜R2-L4 はどれも merge を止めない)
- Follow-ups:
  - L2、L3 は doc の直しだけで、cycle 1 のうちに `org.go` を 1 コミットで直せる。直すなら /verify の前。
  - L1 は 1 つの包みか 1 行のコメント。L4 は doc 1 文とテストの case 1 つ。
  - L5、L6、L7 はテストの書き方で、任意。L7 は /test の判断に回してよい。

## Run 2(S4、bd41041a)

- Date: 2026-11-02
- Reviewer: reviewer サブエージェント(`/self-review`、パイプラインの cycle 1 のまま。/test のあとに足した S4 の分)
- Scope: `git show bd41041a`。`internal/cli/org_start_plan_config_test.go` に `TestOrgStartPlan_OrgWideLimits_ReadFromMainWorktreeRalphToml` を足し、ファイル冒頭のコメントを 1 句書き換えただけ(+105 -2)。Run 1 の対象(S1〜S3、F1)は見直していない。diff の質だけを見た。テスト、静的解析、仕様適合は見ていない。
- ID: 所見は R2-L1〜R2-L4 で、すべて LOW。CRITICAL、HIGH、MEDIUM はない。

### Evidence reviewed

- コミットの diff 全部と、テストが呼ぶ helper の定義(`newFeatureRepoSub`、`startOrgLeader`、`startOrgFeature`、`writeFeatureSplitPlan`、`approvedFeatureSplitPlan`、`lastSeatEvent`、`pathMissing`、`readLogLines`、`readManifestEvents`、`runLedgerTestGit`、`setupOrgStubPATH`)。使い方は定義の引数どおり。
- 双子の `TestOrgStart_OrgWideLimits_ReadFromMainWorktreeRalphToml`(`org_reserve_test.go:368`)。名前の型、doc の書き出し(`X is Y for start --plan`)、「対照のケース」を表の 1 行にする作りが同じ。
- 同じファイルの `TestOrgStartPlan_UnreadableConfigStartsNothing`(`:137-181`)。「何も作らない」の確かめの並びを見比べた。
- 断りの文面は `internal/org/envelope.go:97,114` の書式と一致する(`max_orgs 1 reached: org_id "b" is not running and 1 orgs are (org-a)`、`max_total_seats 1 reached: 1 seats are active across all orgs`)。
- 断りの場所は `internal/org/feature.go:238` の `spawnPrecheckErr` で、worktree の作成(`:253` の `Ensure`)と manifest への書き込みの前。同じファイルの冒頭の doc(`feature.go:47-48`)が「6 より前は manifest に何も書かない」と言う。テストの「b について何も記録されない」はこの順序を固定している。
- `config.Default()` の `MaxOrgs` は 10、`MaxTotalSeats` は 30(`config.go:759-760`)。対照のケースの値の読みに使った。

### Findings

| Severity | Area | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- |
| LOW | readability | R2-L1: 表の行が位置指定の 6 列。対照の行 `{"--config", "[org]\nmax_orgs = 1\n", "", true, false, ""}` は、3 列目の `""`(subToml を書かない)、4・5 列目の `true, false`(config、outsideLedger)、末尾の `""`(b が起動する)を、フィールド宣言と数えて合わせないと読めない。空文字列が意味を持つ(`""` で「書かない」「起動する」)ので、省略でなく書いた値に見える。同じファイルの `ReadsEachConfigFileOnce`(4 列)から 2 列増えて、位置指定の限度を超えた。 | `org_start_plan_config_test.go:270-282`(フィールド宣言と 4 行)。 | 行にフィールド名を付ける。対照の 2 行は `{name: "--config", mainToml: "[org]\nmax_orgs = 1\n", config: true}` のように使う列だけ書けば、空の `subToml` と `refusal` が「書かない」「起動する」として読める。 |
| LOW | maintainability | R2-L2: 同じ fixture と確かめの複製がこのファイルで増えた。(a) 「何も作らない」の確かめ(worktree、branch `fix/b`、`.git/ralph/worktrees`)の 9 行が `UnreadableConfigStartsNothing` の 167-175 と新テストの 338-346 で同じ。(b) 台帳を main の外に置く 4 行(`EvalSymlinks(t.TempDir())`、`filepath.Join(tmp, "state")`)が 87、210、301 の 3 か所。(c) main に `ralph.toml` を commit する 2 行が 148、295。新テストは 89 行で、`t.Run` の本体が 70 行(285-355)あり、起動する側と断られる側が `if tc.refusal == "" { …; return }` で 1 つの本体に入っている。 | 上記の行番号。(a) は worktree のパスの組み立て、`exec.Command("git", …)` の呼び方、`records` のパスまで一致する。 | (a) を `requireNoFeatureStart(t, root, "b")` のような helper に出して 2 つのテストから呼ぶ。manifest と herdr の確かめは条件が違う(前者は全体が空、後者は b の分と herdr の増分)ので helper の外に残す。(b) は `outsideLedgerDir(t)`。直さないと、次にこのファイルへ足すテストが 4 回目の複製になる。 |
| LOW | readability | R2-L3: doc の 2 文がテストの中身より強く言っている。(a)「Both ralph.toml files are committed」は 4 件中 3 件の話で、`--config` の行は `pkg/sub/ralph.toml` を書かず、`--config` のファイルは commit しない(gitignore された `.harness/` の下)。(b)「`--config` … keep the caller's values, which let b start」は、対照が示す範囲より広い。2 つの対照の caller の値(99)は `Default()` の 10 と 30 でも b が起動するので、対照が固定するのは「main の 1 が当たらない」ことで、caller の値が使われたことではない。 | doc は `org_start_plan_config_test.go:258-267`。行は `:281-282`(`max_orgs = 99` の `--config` ファイルは `:315`、`max_total_seats` の対照はない)。既定値は `internal/config/config.go:759-760`。 | (a) は「commit した `ralph.toml` があるぶんは clean のまま」と数を言わずに書く。(b) は「`--config` と外の台帳では main の `ralph.toml` が当たらない(b が起動する)」に直す。caller の値が使われたことまで固定するなら、対照の値を `Default()` の外(例 `max_orgs = 2`)にする手もあるが、`ReadsEachConfigFileOnce` がファイルの読みを固定済みなので必須ではない。 |
| LOW | readability | R2-L4: 「b について何も記録されない」が双子と逆に見えるのに、理由が書いていない。双子 `TestOrgStart_OrgWideLimits_ReadFromMainWorktreeRalphToml` は断られた org の leader が `rejected` で記録されることを確かめる(`org_reserve_test.go:452`)。start --plan は `spawnPrecheckErr` で worktree と manifest の前に断るので、記録は残らない(`feature.go:47-48,238`)。doc の「records nothing for b」は事実だが、2 つのテストを並べて読む人は、片方の assert の抜けと取る。冒頭のコメントの書き換えは、1 行目だけ短い(66 字)まま折り返しが揃っておらず、段落の途中が不揃いに見える。 | 双子は `org_reserve_test.go:452`(`last.Event != org.EventRejected`)。新テストは `org_start_plan_config_test.go:347-351`(manifest に b の event があれば失敗)。冒頭は `:15-21`(`that read still repeats … Like the` が 66 字、前後は 75〜77 字)。 | doc に「(start --plan の断りは worktree と manifest の前なので、`rejected` の event も残らない)」と 1 句足す。冒頭のコメントは段落を折り返し直す。 |

### 依頼された点の判断

- テストの名前: 双子の `TestOrgStart_…` に `Plan` を足した形で、grep で対になって出る。サブテスト名は `max_orgs`、`max_total_seats`(設定のキー)と `--config`、`a ledger outside the main worktree`(状況)が混ざるが、ファイルの他のテストも同じ書き方で、読みを損なわない。
- fixture の作り方: `newFeatureRepoSub`、`approvedFeatureSplitPlan`、`writeFeatureSplitPlan`、`startOrgFeature` を使い、新しい helper を作っていない。2 つの `ralph.toml` を commit して checkout を clean に保つ点は、起動が通る対照のケースに必要で、担う理由を doc に書いている。org-a は `--state-dir` を明示して起動し、b は台帳を省く作りで、既定の台帳と org-a の台帳が同じ前提が、断られるケースの成立で自然に検証される(食い違えば断りが出ず、テストが落ちる)。
- 何も作らないことの確かめ: worktree、branch、`ralph-worktree.sh` の記録、b の manifest の event、herdr の呼び出し(org-a の起動後の件数との差)の 5 つで、`UnreadableConfigStartsNothing` と同じ強さ。herdr は「ログ全体が空」でなく「増えていない」で見るので、org-a の起動を含んでも正しい。複製は R2-L2。
- 対照のケースが空振りしないか: 空振りしない。`--config` と外の台帳の対照は、断られる 2 行と同じ構図(main の `max_orgs = 1`、org-a が同じ台帳で起動中)で、cwd の違いを除いて main の 1 が当たれば断られる。当たらないから起動する、という対応が成り立つ。`max_total_seats` の対照がないのは、main を読まない経路が 2 つの対照で通っているので不要。読みの範囲は R2-L3(b)。
- 既存の helper の使い方: 引数と戻り値の取り方は定義どおり。`lastSeatEvent` は起動する側にだけ使い、断られる側は manifest を走査する。理由は、断られた b に leader の event がなく `lastSeatEvent` が Fatal になるため。選び方は正しい。
- ファイル冒頭のコメント: 4 つ目の句を足した内容は正しい。折り返しは R2-L4。

### Positive notes

- 変異 m1(`applyMainWorktreeOrgLimits` を承認方式だけにする)で落ちる形になっている。`max_orgs` と `max_total_seats` の 2 行が別々の値を固定し、1 つだけを外す変異も捕まえる。
- 断りの文面を `strings.Contains` で書き、`envelope.go` の書式とそろえた。別の理由で同じ種類のエラーが出ても、max_orgs と max_total_seats を取り違えない。
- 並行しない理由(`t.Chdir`)は冒頭のコメントに既にあり、新しいテストも同じ制約を守る。
- 秘密、デバッグの残り、範囲外のファイルは見つからなかった。

### Tech debt identified

| Debt item | Impact | Why deferred | Trigger to pay down | Related plan/report |
| --- | --- | --- | --- | --- |
| なし(R2-L1〜R2-L4 は PR の中で直せる LOW) | - | - | - | - |

### Coverage gaps

- 新しいテストの実行結果(4 つのサブテストが通ること、m1 で落ちること)は、この review では見ていない。plan の S4 の verify と /test の担当。
- `lines[herdrCalls:]`(`:353`)は、ログが縮むと範囲外になる書き方だが、stub のログは追記だけなので起きない。確かめてはいない。
