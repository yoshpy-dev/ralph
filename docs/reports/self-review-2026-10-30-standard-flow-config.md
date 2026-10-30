# Self-review report: standard-flow-config

- Date: 2026-10-30
- Plan: `docs/plans/active/2026-10-30-standard-flow-config.md`(issue #276 の B1、S1〜S6、D1〜D8、AC1〜AC8)
- Reviewer: reviewer サブエージェント(`/self-review`、パイプラインの cycle 1)
- Scope: `git diff origin/main...HEAD`(merge-base 0351800d、HEAD a8d8bd9e、50 ファイル、+2714/−739)。依頼で重く見たのは 6 か所。`scripts/ralph-config.sh` の `ralph_setting`、`internal/config/config.go` の `validateReviewer` と `reviewerWordPattern`、`/cross-review` の上限の規則と `cap` を書く jq、`internal/org/feature.go` の `featureLeaderTask`・`featureReviewer` と `internal/cli/org.go` の `resolveConfig`、`internal/cli/doctor.go` の `retiredEnvs`、`internal/org/prompts/leader.md` の計画助言のコマンド。
- ID: 所見は L1〜L5 で、すべて LOW。

## Evidence reviewed

- `ralph_setting`(`scripts/ralph-config.sh:58-94`。template の写しと `cmp` で一致)
  - 終了コード: 0 で値が空でなければその値を出す。黙って既定値に戻るのは、127 と、rc 1 で stderr に `unknown command "config"` があるときだけ。ほか(2、3、ほかの 1、126、シグナルの 130、0 で空)は stderr に 1 行(キー、rc、ralph の stderr の 1 行目、既定値)を出して既定値に戻る。表にないキーは何も出さず 2 を返す。
  - `set -e` の呼び出し側: 失敗しうるコマンドは、どれも `if` の条件か `|| _rs_rc=$?` の中にある。関数が 0 以外を返すのは表にないキーのときだけ。`set -u` でも、`$1` は `${1-}` の確かめのあとでしか使わない。
  - 一時ディレクトリ: `mktemp -d` が失敗したら warn して既定値に戻る。中身は ralph の stdout と stderr だけで、`return` の前に `rm -rf` する。trap を置かないので、呼び出し側の trap を上書きしない。シグナルで止まると残るが、中身に秘密はない。
  - POSIX sh: `local` を使わず `_rs_` の名前にしている。ほかは `${1-}`、`printf '%s\n'`、`sed -n 1p`、`grep -q`。POSIX にないのは `mktemp -d` だけで、その失敗にも道がある。
- `reviewerWordPattern`(`config.go:752`)
  - Go の `$` は m フラグなしでは文末にだけ当たるので、末尾の改行も断る。
  - 許す 7 つの記号はどれも、sh・bash・zsh の二重引用符の中で展開されない。展開される `$`、`` ` ``、`\`、`"`、`!` は入っていない。
  - skill は値を `"$model"` の変数で渡すので、値は 2 度目の解釈を受けない。leader は TASK の値を二重引用符の中に写すが、展開される文字がない。
  - 先頭の `-` は通る。codex(clap)は `-m -x` を値のない `-m` として断り、claude(commander の必須引数)は次の語を値として取る。どちらでも別のフラグとしては読まれない(ライブラリの既定の振る舞いから推した。未確認です)。
  - `claude_effort` は 5 つの値に限られる。
- `/cross-review` の上限(`.claude/skills/cross-review/SKILL.md:44`、`:181-185`)
  - 1.d は persisted-identity mode でだけ、設定の値と `cap` の大きいほうを取る。fallback mode では ∞ のまま。
  - jq の `a > tmp && mv || rm -f tmp; jq -c . f` は、jq が失敗すると tmp を消し、元のファイルを残す。`cycle` が文字列なら jq が失敗する。
  - CAP_REACHED は `cycle >= max(設定, cap)` なので、`cycle + 1` を書いても上限は下がらない。
  - `cycle-count.json` を読むのは `insights-append.sh`(`.cycle` だけ)と `/pr` の `rm -f` だけで、`cap` のキーが増えて壊れるものはない。
- `featureLeaderTask`(`feature.go:462-483`)と leader.md(`:526-536` と codex・claude のコマンド)
  - 2 行は `- 台帳:` の行のあとに、定数の頭で書かれる。
  - 値は shellQuote を通らず、雛形の二重引用符の中に入る。
  - leader は実行時に設定を読まない。
- `retiredEnvs`(`doctor.go:894-909`)の 4 行と `checkRetiredEnv`(`:911-935`)、cli の TestMain の unset。
- `docs/tech-debt/README.md` を両方向で突き合わせた。この diff は register に触れていない。
  - 161 行の (b) は「上限を上げた回は `cycle` を増やさない」と書き、Why deferred に「step 9 が `cycle-count.json` に書くものを変えることになる」とある。この PR は step 9 が書くもの(`cap`)を変えたが、`cycle` は今も増やさないので、(b) の事実は残る。計画が /sync-docs に回した 1 文の追記で足りる。
  - 162 行の `RALPH_STANDARD_MAX_PIPELINE_CYCLES` は過去の出来事の記述で、直さなくてよい。
- 確かめていないこと: テストと静的検査は、skill の規則どおり走らせていない。shell の挙動はコードを読んで判断し、probe していない。

## Findings

| Severity | Area | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- |
| LOW | exception-handling | L1 doctor の 4 行は「ralph no longer reads it」と書き、unset を勧める。しかし 4 つの変数を読んでいたのは binary ではなく、プロジェクトに置かれた skill と `scripts/ralph-config.sh` だった。`ralph upgrade` をまだ打っていないプロジェクトでは、古い skill が今も変数を読む。そこで warn のとおりに値を `ralph.toml` に書いて unset すると、古い skill は `[reviewer]` を読まずに自分の既定値(claude は `opus`)を使う。A1・A2 の 2 行は binary が読んでいた変数なので、この文がそのまま成り立っていた。doc の「none of the variables changes a value ralph uses」も同じ前提で書かれている。doctor の版の食い違いの行(`doctor.go:701`、run 'ralph upgrade')があるので、害は小さい。 | `internal/cli/doctor.go:894-899`、`:918-920`、`:931`。`internal/cli/init.go:469-484`(`scripts/` と `.claude/skills/` は core で、upgrade が入れ替える) | 4 行の `effect` に「`ralph upgrade` で skill と `scripts/ralph-config.sh` が入れ替わるまでは、それらがまだ読む」の旨を足す。あるいは `instead` を「`ralph upgrade` のあとで」から始める。 |
| LOW | security | L2 TASK の reviewer の 2 行が shell で安全なことは、別の package の `config.Load` の検証だけに頼っている。export された `StartFeature` は `p.Reviewer` をそのまま TASK に書き、`featureReviewer` は空のフィールドを埋めるだけで値を見ない。同じファイルの承認方式は、org の側でも `validateFeatureBinding` が断る(`feature.go:364-368`)。reviewer には org の側の確かめがない。本番の呼び出しは `runOrgStartPlan` の 1 か所で、Load の結果を渡すので、今は突ける道がない。 | `internal/org/feature.go:98-104`、`:381-396`、`:453-461`(doc が Load を根拠に挙げる)、`internal/org/reserve.go:157` | `config` に `ValidateReviewer`(または `ReviewerConfig` のメソッド)を export し、`StartFeature` が worktree を作る前に断る。軽く済ませるなら、`featureReviewer` が pattern に合わない値も既定値で置き換える。 |
| LOW | maintainability | L3 `runOrgStartPlan` は、`newOrgSpawnRuntimeAt` が `resolveOrgConfig` で読んだのと同じパスを、`resolveConfig` でもう一度読む。`config.Load` は呼ぶたびに `config.LocalPath` → `gitroot.MainWorktree` で git を打つ。この repo は、同じ理由で 2 回の解決を 1 回に直したことがある(`org.go:344-350` の注釈「each resolution shells out to git」)。計画の S4 は「runtime は `[org]` しか持たない」を理由に挙げているが、1 回の Load の結果から両方を取れる。 | `internal/cli/org.go:1131`、`:1144`、`:352`、`:628`、`:638-644`。`internal/config/local.go:42-54`、`:65-70` | `resolveConfig` を先に呼び、`cfg.Org` を受け取る runtime の作り方を足して 1 回にする。残すなら、2 回読む理由を `runOrgStartPlan` の doc に 1 文書く。 |
| LOW | maintainability | L4 1.d の有効な上限は、4 つの条件(JSON の数、整数、1 以上、設定の値との大きいほう)を文だけで書き、読むコマンドがない。書く側(step 9)には jq のコマンドがあるので、読む側だけ agent の目に任される。step 9 は「printed line must show the new cap」と書くが、出なかったとき(jq がない、`cycle` が数でない)にどうするかを書いていない。 | `.claude/skills/cross-review/SKILL.md:44`、`:181-185`(4 つの写しが同じ) | 1.d に `jq -r '.cap \| select(type == "number" and . >= 1 and . == floor)' "$f"` のような 1 行を置く(出力がなければ `cap` なし)。step 9 に「`cap` が出なければ止まってユーザーに伝える」を足す。 |
| LOW | readability | L5 leader.md は「どちらの行も、`:` のあと括弧の前にモデルと effort の 2 語」と書く。この PR はモデル名に `:` を許した(Bedrock の `…-v1:0`、ARN)。そのため行は `- reviewer の claude: us.anthropic.claude-…-v1:0 xhigh(…)` のように `:` を 2 つ以上含みうる。最後の `:` のあとを読んでも `0` と `xhigh` のちょうど 2 語になるので、規則の字面だけでは読み違いを防げない。 | `internal/org/prompts/leader.md:527-529`、`internal/org/feature.go:478-482`、`internal/config/config.go:752` | 「行の頭の `- reviewer の codex:`(`- reviewer の claude:`)のあと」と書く。この文は `prompts_leader_plan_test.go` が固定しているので、テストの文も合わせて直す。 |

## Notes(所見ではない)

- 依頼の 4 点目、`[reviewer]` と `[org.approval]` の読む元の違い
  - 安全の面では問題にならない。`[org.approval]` を主 worktree から読むのは、leader が書き換えられる feature worktree の `ralph.toml` に承認方式を決めさせないためだった(`org.go:178-182`)。reviewer の値は start が TASK に書くので、leader があとで自分の worktree の `ralph.toml` を書き換えても効かない。Load の検証を通った値しか入らない。
  - 違いが出るのは、start を主 worktree の root の外で打ったときだけ。linked worktree なら、その branch の `ralph.toml` の値になる。subdirectory なら `./ralph.toml` がなく、個人用のファイルのほかは既定値になる(`config.go:800-814`)。`[org]` の `model_pool` などと同じ扱いで、org skill の新しい段落もそう書いている。`ralph org start --help` が書いていない点は、計画が /sync-docs に回している。
- leader.md の「タスクに相手の行がなければ(その行を書かない古い ralph が起こした org)」の分岐には、今の作りでは届かない。古い ralph は古い雛形も一緒に描くからで、害はない。
- 計画の S2 から S4 にかけて一時的に入れ、あとで外した照合(skill の fallback の `opus`)は、HEAD に残っていない。

## Positive notes

- 既定値は `Default()`、helper の表、template の注釈の 3 か所にあり、`defaults_sync_test.go` が 3 か所を互いに照合する。template の注釈の行は「注釈としてちょうど 1 回、`Default()` と同じ値」で見ている。
- `ralph_setting` は黙る場合を 2 つに絞り、ほかの失敗は 1 行で知らせる。そのため `config get --help` が約束する「exit 2 は止まって知らせる」と食い違わない。
- leader に渡す値を TASK に固定したので、pane の PATH にある ralph の版にも、プロジェクトの `scripts/ralph-config.sh` の版にも左右されない。

## Tech debt identified

| Debt item | Impact | Why deferred | Trigger to pay down | Related plan/report |
| --- | --- | --- | --- | --- |

この報告では register に行を足していない。L1〜L5 は今の PR で直せる大きさで、まだ先送りと決まっていない。最後の cycle で直さずに残すときは、1 行にまとめて register に入れる。

## Recommendation

- Merge: yes(CRITICAL・HIGH・MEDIUM はない)
- Follow-ups:
  - L1〜L5 は任意。直すなら、L1 と L5 は文だけの変更で、L5 はテストの文も一緒に直す。
  - /sync-docs に回すもの(計画の S6 の一覧と同じ): `docs/tech-debt/README.md:161` の (b) への 1 文、`ralph org start --help` の `[reviewer]` の読む元。
