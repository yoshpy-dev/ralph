# Self-review report: standard-config-gaps

- Date: 2026-11-02
- Plan: `docs/plans/active/2026-11-02-standard-config-gaps.md`(PR #291 の Known gaps、S1〜S3、D1〜D5、AC1〜AC6)
- Reviewer: reviewer サブエージェント(`/self-review`、パイプラインの cycle 1)
- Scope: `git diff origin/main...HEAD`(base d217c575、HEAD 2066d854)。コードは S1 b71354d0、S2 7b9dc257、S3 2d4baa2d の 3 コミット(`internal/cli/org.go`、新しい `org_start_plan_config_test.go`、`org_feature_reviewer_test.go`、`org_feature_test.go`、`tests/test-ralph-config.sh`)。diff の質だけを見た。テスト、静的解析、仕様適合、文書の drift は見ていない。依頼で個別に判断を求められた 3 点は「依頼された 3 点の判断」に書いた。
- ID: 所見は L1〜L7 で、すべて LOW。CRITICAL、HIGH、MEDIUM はない。

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

- Merge: yes(CRITICAL、HIGH、MEDIUM はない。L1〜L7 は LOW で、どれも merge を止めない)
- Follow-ups:
  - L2、L3 は doc の直しだけで、cycle 1 のうちに `org.go` を 1 コミットで直せる。直すなら /verify の前。
  - L1 は 1 つの包みか 1 行のコメント。L4 は doc 1 文とテストの case 1 つ。
  - L5、L6、L7 はテストの書き方で、任意。L7 は /test の判断に回してよい。
