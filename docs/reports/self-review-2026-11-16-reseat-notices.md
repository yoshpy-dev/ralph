# Self-review report: reseat-notices

- Date: 2026-11-16(cycle 1。`cycle-count.json` は `cycle: 1`)
- Plan: `docs/plans/active/2026-11-16-reseat-notices.md`
- Branch: `fix/reseat-notices`、HEAD `f0cc9071`、base は `origin/main` の `30e122a1`
- Reviewer: reviewer サブエージェント(`/self-review`)
- Scope: `git diff origin/main...HEAD` の 19 ファイル(+685 −84)の diff の質だけ。コードは S1 `b21c75c7`(internal/org)と S2 `72cc7d72`(internal/cli)。文書とテストは S3 `0d7dfe1f` と S4 `f18738bd`。計画書と記録の commit(`6eb82ceb`、`f806f416`、`f0cc9071`)は、Progress の説明が差分と合うかだけ見た。テスト、静的解析、spec 適合、文書のずれの検査は行っていない(`/verify` と `/test` の担当)

## Evidence reviewed

- コード: `internal/org/reseat.go`(`ReseatParams.OnPlan`、`Reseat` の 2 つの呼び出し、doc comment)、`internal/cli/org_reseat.go`(callback、`printOrgReseatCodexHookTrustNotices`、help の連結、`runOrgReseat` の doc)、`internal/cli/codex_hook_trust_notice.go`、`internal/cli/scaffold_version_notice.go`
- callback が読む先: `doctorCodexSandboxEnv`(`codexSandboxEnvFromOS`)、`codexHookTrustFor`、`codexHookSpellings`、`codexHookCheckoutOf`(`codex_hook_trust.go`)。exec は 1 つもない
- 呼び出し元の前提: `PlanReseat` の束縛の検査(`reseat.go:420`)、`FeatureBinding.Complete`(`reserve.go:125`)、seat の cwd(`reseat.go:513`)
- 文書: `/director` skill の 4 写し(差分どうし、全文どうしを `cmp` と `diff` で比べた)、`internal/org/prompts/director.md`、`.codex/README.md` の 2 写し、仕様の #318 の項、tech-debt の 478 行(語単位の差分)
- テスト(読んだだけで、走らせていない): `internal/org/reseat_test.go` の 2 つ、`internal/cli/codex_hook_trust_notice_test.go` の 4 つ、`org_reseat_test.go`、`scaffold_version_notice_cli_test.go`、`internal/org/prompts_notice_test.go`
- 周辺の言い回しの一覧: `git grep` で、hook の行と雛形の版の行を出す動詞を並べた文(Go の comment、help、README、`docs/quality`、`docs/recipes`、org の skill、leader の prompt)

## 依頼された 5 点の確認

| 問い | 結果 | 根拠 |
| --- | --- | --- |
| `OnPlan` の comment は `Reseat` の呼び出し場所と合うか | 合う | dry run は `PlanReseat` のあと `err == nil` のときだけ呼ぶ(`reseat.go:1125-1128`)。本物は `PlanReseat` が通り、印を書いたあと、`reseatRun` を作る前に 1 度呼ぶ(`:1152-1154`)。「1 度まで」「呼ばれない場合」の 4 つ(flag の誤り、別の reseat のロック、`PlanReseat` の断り、印の書き込みの失敗)は、`:1135-1150` の return と 1 つずつ合う。`Reseat` の doc の手順 2 も合う(`:1086`)。契約の「計画を読むだけ」と callback の実際のずれは L1 |
| CLI の callback はロックの中で読むだけか | 読むだけ。コメントの言い方に LOW 1 件(L1) | `printCodexHookTrustNotice` は環境変数と home、`hooks.json`、worktree の `.git`、codex の設定を読み、座席ごとに stderr へ 1 行書く。コマンドの実行も待ちもない。write のエラーは捨てる(`_, _ =`)ので、読み手のいない stderr でも止まらない |
| nil `Binding` の guard | 到達しない枝。直さなくてよい | `PlanReseat` は `binding == nil \|\| !binding.Complete()` を断り(`reseat.go:420`)、`Complete` は `Worktree != ""` を求める(`reserve.go:125`)。callback に渡る計画は、dry run でも本物でも、束縛と空でない worktree を持つ。そのため、`dir` が空になって `filepath.Abs("")` が作業ディレクトリの hooks を読む形も起きない。guard が防ぐのは nil の参照だけで、空の worktree は防がないが、`Complete` が先に断る。この枝を通るテストはない。comment が「does not get here」と理由を言っている |
| 日本語の skill と prompt の言い回しは 4 写しで揃い、テストの断片と合うか | 揃い、合う | skill の 4 写しの差分は 1 行も違わない。`.claude` と `templates/base/.claude`、`.agents` と `templates/base/.agents` は `cmp` で一致。`directorNoticeRules` の変えた断片 4 つと足した断片 2 つ、`TestDirectorSkill_ReseatPointsAtTheHumanNotes` の断片、`directorPromptNoticePhrases` の 1 つめを、skill と `director.md` の本文と順に突き合わせた(空白と改行を除いた比較で、`withoutSpaces` と同じ)。数は「4 つの動詞」(start、reseat、attach、director start)と、テストの comment の「five verbs」(status を足した数)が合う |
| 動詞を並べた comment は正確か | 3 か所を直している。ほかに足りない所は見つからなかった。言い回しに LOW 3 件(L3、L4、L5) | `codexHookTrustAgent`、`orgCodexHookTrustNoticeHelp`、`orgScaffoldVersionNoticeHelp` と、テストの doc 4 つに reseat がある。`org.go:1296`、`org_director_headless.go:316` の comment は、その動詞だけの話なので直す所がない。`README.md:133` は動詞の総覧で、hook の行を並べていない |

## Findings

この run の指摘は L1 から振る。どれも LOW で、CRITICAL、HIGH、MEDIUM はない。

| ID | Severity | Area | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- | --- |
| L1 | LOW | maintainability | `OnPlan` の契約と、CLI の callback の説明がずれる。契約は「ロックの中で動くので、計画を読むだけで、ralph の動詞を打たない」。CLI 側の comment は「It only reads files, as Reseat requires of a call made under the reseat lock」で、Reseat が求めているのは「計画を読む」ことなので、借りた文が合わない。callback が実際にするのは、計画のほかに `hooks.json`(root と各層)、worktree の `.git` ファイル、codex の user config を読むことと、codex の座席ごとに stderr へ 1 行書くこと。実害はない(exec も待ちもなく、ロックを長く持たない)。ただ、`OnPlan` に次の仕事を足す人は、契約の文から「小さな設定ファイルの読み取りと 1 行の書き込みは許される」と読み取れず、許される範囲を callback の実装から逆算することになる。計画 Risks の「関数は計画を読むだけにし」も、実装では緩んでいる | `internal/org/reseat.go:81-82`(契約)、`internal/cli/org_reseat.go:264-265`(借りた文)、`internal/cli/codex_hook_trust.go:617-631`(読む先) | `OnPlan` の文を、「速く終わり、ralph の動詞を打たない」に直し、CLI は小さな設定ファイルを読んで行を stderr に書く、と例を添える。CLI 側の「as Reseat requires」は、直した契約に合う言い方にする |
| L2 | LOW | readability | reseat の `--help` に足した hook 段落は、spawn と start と director start のために書いた文のままで、座席 1 つ、行 1 つの言い方をする。「the agent the command starts」「one note on stderr gives the counts」と言う。reseat は立て直す codex の座席ごとに 1 行を、座席を止める前に出し、`--dry-run` でも出す。同じ変更の skill(4 写し)と `.codex/README.md`(2 写し)は「座席ごとに 1 行」と書き、`printCodexHookTrustNotice` の doc も「once per seat」と書く。利用者が最初に読む `--help` だけが、座席が 3 つなら行も 3 つ出ることを言わない。`--dry-run` の段落は「It only reads whether the cross-org watch and the herdr server run」と言い、hook の行のために `hooks.json` と codex の設定を読むことを言わない | `internal/cli/codex_hook_trust_notice.go:110-122`、`internal/cli/org_reseat.go:95-99`(`--dry-run` の段落)、`:132-134`(連結) | 共有の段落は変えずに、その前に reseat だけの 1 文を足す。例: 「reseat prints one such note for each codex seat it starts, before it stops any seat, with --dry-run too」。`TestOrgReseatHelp_...` は `HasSuffix` で末尾の 2 段落だけを固定するので、前に足す文は壊さない |
| L3 | LOW | readability | `orgScaffoldVersionNoticeHelp` の doc が、自分の中で食い違う。「`ralph org start --help`、…、`ralph org reseat --help` **end with** (all but attach with orgCodexHookTrustNoticeHelp after it)」と書く。start、reseat、director start の Long は hook 段落で終わるので、雛形の版の段落で「end with」は偽で、括弧がそれを打ち消している。「end with」は #310 のときから残った言い方で、この差分が同じ文の書き換えを触っている。test の doc(`scaffold_version_notice_cli_test.go:267-274`)は「have ... as a paragraph of its own」と直してあり、2 つの言い方が割れた | `internal/cli/scaffold_version_notice.go:107-113`、`internal/cli/scaffold_version_notice_cli_test.go:267-274` | 「have it as a paragraph of its own (all but attach put orgCodexHookTrustNoticeHelp after it)」の形にして、test の doc と合わせる |
| L4 | LOW | readability | `runOrgReseat` の doc は、関数がする順に並べた一覧で、hook の行を「model_pool と雛形の版の行」と「runtime」のあいだに入れた。hook の行が出るのは `rt.Reseat` の中で、runtime の組み立て、2 つの gate、`FeatureRepoRoot` のあとなので、一覧の順と実際の順が違う。挿入のために折り返しも崩れ、`// (newOrgRuntimeFromConfig) with` が 1 行だけ残って、次の行が `InboxHintStateDir,` で新しい断片を始める | `internal/cli/org_reseat.go:188-198` | 項を「PlanReseat runs both before it stops anything」の文のあとに移し、折り返しを直す |
| L5 | LOW | readability | `printCodexHookTrustNotice` の doc に足した reseat の文が、「with the driver they resolved.」と、引数を説明する「verb names the command …, seat …, and dir …」のあいだに入った。`dir` と `verb` が説明の前に reseat の文で使われ、「from the OnPlan of its org.ReseatParams (…), so before it stops any seat, in a dry run as well」は、`so` のあとが句で、主語がない | `internal/cli/codex_hook_trust_notice.go:9-23` | reseat の文を、「It does nothing unless driverName is "codex" …」の段落のあとに、独立した段落か 2 文に分けて置く |

## 読んだが指摘にしないもの

- reseat は codex の座席ごとに 1 行を出すので、座席が 3 つなら、座席 id だけが違う 3 行が出る。skill は「行をそのまま人に伝える」と言う。計画の D1 が決めた形で、diff の質の問題ではない。人に伝える文が長くなるだけで、情報は落ちない
- dry run のあとに本物の reseat を打つと、同じ座席に同じ行が 2 度出る。S4 が skill の手順 2・4 と小節の bullet で「1 度だけ伝える」と決め、test が 3 か所を固定している。コードに dedupe を入れなかった判断は、計画の Scope の外
- `newReseatCLIOrgWith` の `if len(files) > 0` は、`range` が空なら何もしないので不要。`commitRalphTomls` の commit message が `ralph.toml` のまま `hooks.json` を commit するのは、既存の helper の名前で、この差分の前から start --plan のテストが同じ使い方をする。どちらも読みやすさへの影響は小さい
- `OnPlan` は計画を値で受けるが、`Binding`、`Start[i].Spawn`、スライスは共有される。callback が書き換えれば、そのあとの start に効く。comment が「読むだけ」と言う契約で守る形で、`WatchProbe` と同じ程度。今の callback は書き換えない
- skill の 4 写しと `.codex/README.md` の 2 写しは、root と `templates/base` で一致する。`scripts/check-sync.sh` と `check-skill-sync.sh` の実行は `/verify` の担当
- `plan 2026-11-16-reseat-notices` を Go の comment に写すのは、既存の comment(`2026-11-12-codex-hook-trust-check` など)と同じ慣習

## Positive notes

- 仕組みが小さい。`ReseatParams` に `WatchProbe` と同じ形の関数を 1 つ足し、CLI が渡す。internal/org から internal/cli への依存は増えず、nil なら何も変わらない
- 呼び出しは 2 か所で、「PlanReseat が通ったあと、どの座席も止める前」という 1 つの規則を、dry run と本物で同じにしている。`OnPlan` の comment が、呼ばれない 4 つの場合を `Reseat` の return と 1 つずつ合わせて書く
- CLI のテストが、行を書いた時点の manifest を取り(`reseatStderrAtNote`)、reseat 前と同じことで「止める前に出る」を確かめる。「出ない」ことの側(claude の座席、計画の前の断り、flag の断り)も、dry run と本物の両方で組み、positive control を持つ
- org 側のテストが、`OnPlan` が動く時点の `reseat.lock` の印と、台帳に新しい event がないことを callback の中で確かめる
- S4 の逸脱(skill の reseat 手順に「1 度だけ伝える」を足した)は、計画の Progress に理由つきで書いてあり、test で固定してある
- skill の本文は 4 写しが同一で、prompt の断片と test の断片が同じ言い回しを共有する

## Tech debt identified

| Debt item | Impact | Why deferred | Trigger to pay down | Related plan/report |
| --- | --- | --- | --- | --- |
| なし | - | L1 から L5 は、この PR の中で直せる comment と help の言い回し | - | - |

`docs/tech-debt/README.md` の 478 行に reseat を足した S4 の変更は、読んだ範囲で正しい。新しい行は足していない。

## Recommendation

- Merge: yes(CRITICAL、HIGH、MEDIUM はなし。LOW 5 件)
- Follow-ups:
  - L1 から L5 は、comment と help の言い回しの直し。L2 だけが利用者に見える文字列(`orgReseatHelp`)を変えるので、コードの再検査が要る。`/verify` を走らせる前なら、直しにパイプラインの再走を足さずに済む。直さずに進めるなら、PR 本文の Known gaps に書く
  - この run は、依頼された範囲(report だけの commit)に合わせて、insight event を書いていない。skill の手順のとおりに残すなら `./scripts/insights-append.sh --slug reseat-notices --flow standard --phase self_review --verdict pass --critical 0 --high 0 --medium 0 --low 5 --cycle auto --source skill`
  - `/verify` では、2 つの行を出す動詞を並べた文がほかにないかの検索(計画の Verify plan)を、`git grep` で補う。この run の検索では、直す所は見つからなかった

## Addendum: S5(`64746c2b`)の再確認

- Date: 2026-11-16(cycle 1 のまま。S5 は `56992f8e` の上に載った直しの commit)
- Scope: `git diff 56992f8e 64746c2b` の 4 ファイル(+45 −38)。変わったのは comment と、`orgCodexHookTrustNoticeHelp` の文字列 1 つだけで、コードの経路は変わらない。上の L1 から L5 の文は書き換えず、ここに結果を足す。テストと静的解析は走らせていない
- 読んだ以外に確かめたこと: 直した help の行の長さは最大 75 文字(テストの上限は 76)。`TestCodexHookTrustHelp_EachVerbDescribesTheNote` が固定する 2 つの句(`($CODEX_HOME/config.toml, else ~/.codex/config.toml)` と `The Codex hook trust check of\nralph doctor`)は、折り返しを変えたあとも 1 行に収まって残る。直した文言を固定するテストはほかにない(`git grep` で 0 件)。help の段落は 4 つの動詞(spawn、start、director start、reseat)が共有するが、「an agent」「for each such agent」は座席 1 つの動詞でも読める

### L1 から L5 の結果

| ID | 結果 | 根拠 |
| --- | --- | --- |
| L1 | 解消 | `OnPlan` の契約が「ralph の動詞も遅い処理も待ちも持たない。計画を読み、書き換えない」になり、CLI の使い方として「小さな file を数個読み、行を stderr に書く」が書いてある(`reseat.go:81-86`)。CLI 側の comment は、読む先(codex の user config、worktree の `.git` と `hooks.json`)と stderr への書き込みを挙げ、「Reseat が OnPlan に許すこと」に合うと言い直した(`org_reseat.go:263-267`)。借りた「as Reseat requires」はなくなり、実装(exec なし、待ちなし)とも合う |
| L2 | 本体は解消。dry run の半分は未解消で、N1 に移す | 共有の段落が「an agent」「that agent」「one note on stderr for each such agent」になり、reseat が座席ごとに出すことと、spawn、start、director start の「1 つ」の両方を読める(`codex_hook_trust_notice.go:112-124`)。「座席を止める前」と「dry run でも出る」は help に書かれないまま |
| L3 | 解消 | doc が「have it as a paragraph of its own; start, director start, and reseat put orgCodexHookTrustNoticeHelp after it」になり、「end with」はない。配線(`org.go:1156-1158`、`org_director_headless.go:163-165`、`org_reseat.go:132-134`、attach は `org_director.go:137` で段落が末尾)と test の doc(`scaffold_version_notice_cli_test.go:267-274`)に合う |
| L4 | 解消 | hook の行の項を一覧から出し、gate の文のあとに「In both, once PlanReseat has passed, rt.Reseat calls OnPlan, which prints …, before any seat is stopped」と置いた。`Reseat` の 2 つの呼び出し(`reseat.go:1126`、`:1152`)と合う。折り返しも直り、1 行だけ残った断片はない |
| L5 | 解消。小さな読みにくさが 1 つ残る(N2) | 最初の段落が、引数の説明(`verb`、`seat`、`dir`)を「with the driver they resolved:」の直後に戻した。reseat の文は独立した段落になり、各文に主語がある(`codex_hook_trust_notice.go:8-25`) |

### `--dry-run` の文は誤解を招くか

文そのものは偽ではない。「It only reads whether the cross-org watch and the herdr server run」の「only」は、2 つの gate の扱いに掛かる。本物の reseat が watch と herdr server を起動するのに対し、dry run は動いているかを読むだけ、という対比で、後ろの「a note says when a real reseat would start them」も同じ話をしている。この段落は、dry run がすでに読む台帳、分割計画、worktree、`ralph.toml`、`.ralph/manifest.toml`(雛形の版の行)も挙げていない。したがって、hook の行が加わっても、この文が新しく偽になったとは言えない。

ただし `--help` 全体では、dry run が hook の行を出すことが読み取れない。共有の段落は「an agent the command starts」と言い(dry run は何も起動しない)、dry run の段落は「writes nothing」「only reads」と言う。AC2 は dry run の行を契約にしていて、`/director` skill の手順 2 も、dry run の stderr の行に従うよう director に言う。文書の側が dry run の行を前提にしているのに、利用者が最初に読む help は黙っている。

### 新しい指摘

| ID | Severity | Area | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- | --- |
| N1 | LOW | readability | `ralph org reseat --help` は、`--dry-run` が codex の座席ごとに hook の行を stderr に出すことを言わない。dry run の段落は「It stops, starts, and writes nothing. It only reads whether the cross-org watch and the herdr server run」で終わり、共有の hook 段落は「an agent the command starts」と言うので、読み手は dry run が `hooks.json` も codex の設定も読まず、行も出さないと取りうる。L2 の後半の持ち越し | `internal/cli/org_reseat.go:95-99`(dry run の段落)、`internal/cli/codex_hook_trust_notice.go:112-124`(共有の段落)、`internal/cli/org_reseat_test.go:884-885` | dry run の段落の末尾に 1 文足す。例: 「It prints the notes below as a real reseat does, the codex hook note for each codex seat it would start.」。test が固定するのは「…writes nothing.」までなので、後ろに足しても落ちない。`--dry-run` のフラグ説明(`org_reseat.go:170`)は今のままでよい |
| N2 | LOW | readability | `printCodexHookTrustNotice` の doc の 3 つめの段落が「It reads codex's user config …」で始まる。S5 で reseat の段落が間に入ったため、直前の主語は reseat で、「It」を reseat と読める。関数の話は 1 つめの段落で終わっている。小さな読みにくさで、直しても挙動は変わらない | `internal/cli/codex_hook_trust_notice.go:8-35` | 「printCodexHookTrustNotice reads codex's user config …」と主語を書く |

### 読んだが指摘にしないもの(S5)

- `OnPlan` の「waits for nothing」は、stderr への書き込みが、読み手のいない詰まった pipe でブロックしうる点では厳密でない。ただ、callback は書き込みのエラーを捨てるだけで、`orgReseatThroughSignals` が SIGPIPE を受ける形になっている。ロックを長く持つ実害は見つからなかった
- 共有の段落の「so if one is, or that config does not exist」は、元の「so when one is」より 1 語だけ読みにくい(「one」が hook を指す)。意味は変わらず、直さなくてよい
- S5 の commit message は「word the reseat hook-note comments and help for several seats」で、中身(comment 4 つと help 1 つ)と合う

### 更新後の判定

- **Merge: yes**(CRITICAL、HIGH、MEDIUM はなし。L1 から L5 は解消。開いているのは LOW 2 件の N1 と N2)
- Follow-ups:
  - N1 は help の文字列を変える(コード)。N2 は comment だけ。どちらも `/verify` を走らせる前なら、パイプラインの再走を足さずに直せる。直さずに進めるなら、PR 本文の Known gaps に書く
  - insight event の件数は、この addendum を足した時点で LOW 2(5 件が解消、2 件が開いている)。残す場合の `--low` は 2
