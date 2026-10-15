# Cross-review triage report: org-director-start

- Date: 2026-10-16
- Plan: docs/plans/active/2026-10-15-org-director-start.md
- Base branch: main(merge-base f21df66c)
- Driver: claude
- Reviewer: codex
- Triager: Claude Code (main context)
- Reviewer status: complete
- Self-review cross-ref: yes
- Cycle: 3/3 (cap reached)
- Total reviewer findings: 2
- After triage: ACTION_REQUIRED=2, WORTH_CONSIDERING=0, DISMISSED=0

## Triage context

- Active plan: docs/plans/active/2026-10-15-org-director-start.md(承認済み、digest 96796452ceae)。下の「Cycle 2」の ACTION_REQUIRED 2 件は、caf1d025(D12 と D13)と 83287d39 で直した
- Self-review report: docs/reports/self-review-2026-10-15-org-director-start.md(cycle 3 は e62df32e で Merge: yes。LOW 4 件は 83287d39 で直した)
- Verify report: docs/reports/verify-2026-10-15-org-director-start.md(cycle 3 は 3504ee30 で pass。直す前の binary で cycle 2 の 2 件が再現し、HEAD で直ったことを確かめた)
- Test report: docs/reports/test-2026-10-15-org-director-start.md(cycle 3 は b0adfc63 で pass。mutant 30 個のうち 24 個が red。生き残りの 6 個のうち 4 個は等価で、穴の 2 個は sync-docs の 49cee058 で tech-debt に記録した)
- Implementation context summary:
  - `codex exec review --base main` を 49cee058 で実行した。モデルは gpt-6-astra、effort は xhigh、sandbox は read-only。`codex rc=0` で、`-o` のファイルは 1,754 バイト。
  - Codex は read-only の sandbox で Go のテストを回せていない。
  - 2 件とも、主張をコードと parse で確かめた。直し方は consult(consult-plan-xreviewc3)に聞き、判定は「進めてよい」だった。P1 は言い換えと検査のテスト(B1)、P2 は stop がロックの下で記録を比べて断る形(C)。

## ACTION_REQUIRED

| # | Reviewer finding | Triage rationale | Affected file(s) |
|---|-------------------|------------------|-------------------|
| 1 | [P1] `/director` skill の `description:` は引用符のない YAML の値で、`ways: the` のところで「コロンと空白」を含む。そのため 4 面とも YAML として読めない(`mapping values are not allowed in this context`)。base の版は読める。YAML で skill を読む loader は、interactive と headless の director が要る手順を読み込めない。値を引用符で囲むか `description: >` にして、写しを作り直すべき | 実際の問題で、直す価値もある。Ruby の `YAML.safe_load` で、4 面とも 3 行目 155 桁目で失敗することを確かめた。base(f21df66c)の版と、ほかの skill の SKILL.md はすべて読める。入れたのはこのブランチの 9fb4f635(S3)である。`scripts/check-skill-sync.sh` は frontmatter を YAML として読まず、値の引用符も剥がすので、4 回の検証で誰も気づかなかった。直し方は 2 つ。description を「コロンと空白」のない文に言い換え、sync-skills で 4 面を作り直す。あわせて、tracked のすべての SKILL.md の frontmatter について、引用符・`>`・`|` のどれでも始まらない値に「コロンと空白」か「空白と `#`」があれば落とす Go のテストを足す。Codex や Claude Code の loader が YAML をどこまで厳しく読むかは確かめていない | `.claude/skills/director/SKILL.md` と写しの 3 面、新しい frontmatter の検査のテスト、計画 |
| 2 | [P2] stop が start と重なると、動いている director を stopped と記録しうる。stop は、workspace と pane の id がどちらも記録される前に登録を読むことがある。その後、start が両方の id と `launched` を書いてから stop が再開しても、stop は元の読みに従って何も閉じない。`MarkStopped` は登録が開いていることしか確かめないので、動いている director を stopped と書く。その資源は閉じていない headless の登録から消え、次の stop は already stopped で断る。閉じる前に start の進みを止めるか、閉じた資源の記録をロックの下で比べ、変わっていたら片付け直すべき | 実際の問題で、直す価値もある。`StopDirector`(`internal/org/director_stop.go`)はロックなしで登録を読み、`markStopped`(`internal/org/director_start.go`)はロックの下で `openHeadless(id)` だけを確かめる。`recordHerdrID` は、開いた登録になら workspace と pane の id を書く。そのため「読んだときに workspace の記録がない → start が記録する → stop が stopped を書く」の順で、記録が事実と違う。D11 は、閉じ損ねたときに事実と違う記録を残す案を退けている。直し方は、`markStopped` に stop が読んだときの workspace と pane の id を渡し、ロックの下で今の記録と違えば何も書かずに断ること。新しい sentinel を使い、stop の出力では、start が途中で記録を足したのでもう一度 stop を打つよう案内する。`recordHerdrID` は記録済みの id を断るので、id は「空から値」にしか変わらない。等しいかを比べれば、新しく記録されたかが分かる。`launched` は閉じるものに影響しないので比べない。`--force` は閉じていないことを記録するので当てない。stop が workspace を閉じたあとで start の次の段が失敗すれば、start の後始末が stopped を書く。失敗しなければ director は running のまま残り、2 回目の stop が閉じる。どちらでも記録は事実と一致する | `internal/org/director_start.go`(`markStopped`)、`internal/org/director_stop.go`(`StopDirector`、`recordDirectorStop`)、CLI の stop の出力、そのテスト、計画(D14、AC9) |

## WORTH_CONSIDERING

なし。

## DISMISSED

なし。

Categories: false-positive, already-addressed, style-preference, out-of-scope, context-aware-safe

## Cycle 2

- Date: 2026-10-15
- Cycle: 2/2 (cap reached)
- Total reviewer findings: 2
- 仕分けの結果: ACTION_REQUIRED 2 件、WORTH_CONSIDERING 0 件、DISMISSED 0 件(cycle 2 のときの数。この報告の数の行は先頭の 1 行だけにしている)

### Triage context(cycle 2)

- Active plan: docs/plans/active/2026-10-15-org-director-start.md(承認済み、digest f65d7c59746c)。下の「Cycle 1」の ACTION_REQUIRED 1 件は、3243c499 と a539a765 で直した。`--config` を渡した start は、その絶対パスを雛形の `ralph org director status` のコマンドに付け、人に頼む stop の行には付けない
- Self-review report: docs/reports/self-review-2026-10-15-org-director-start.md(cycle 2 は 391b8b80 で Merge: yes。LOW は 5 件で、C2-L1〜C2-L4 は a539a765、C2-L5 は 12c7428d で直した)
- Verify report: docs/reports/verify-2026-10-15-org-director-start.md(cycle 2 は d3bf2066 で pass。直す前の binary と並べ、cycle 1 の指摘が再現して HEAD で直ったことを確かめた)
- Test report: docs/reports/test-2026-10-15-org-director-start.md(cycle 2 は 6556bce5 で pass。cycle 2 の変更の mutant 26 個のうち 24 個が red。生き残った K15 と K16 は、sync-docs の 147f2317 で tech-debt に記録した)
- Implementation context summary:
  - `codex exec review --base main` を 147f2317 で実行した。モデルは gpt-6-astra、effort は xhigh、sandbox は read-only。`codex rc=0` で、`-o` のファイルは 2,073 バイト。
  - Codex は read-only の sandbox で Go のテストを回せていない。
  - 2 件とも、主張をコードと文書で確かめた。直し方は consult(consult-plan-xreviewc2)に聞き、判定は「進めてよい」だった。P1 は案 A(断って案内する)、P2 は案 E(断る)。

### ACTION_REQUIRED(cycle 2)

| # | Reviewer finding | Triage rationale | Affected file(s) |
|---|-------------------|------------------|-------------------|
| C2-1 | [P1] `codex_verified = true` で既定の `autonomous` のとき、codex の director は `--sandbox workspace-write --ask-for-approval never` で動く。書けるディレクトリとして足されるのは、多くても台帳だけになる。codex は `.git` を守る。そのため director が打つ `ralph org start --plan`(`ralph-worktree.sh ensure` の `git worktree add` と、`.git` の下の状態の書き込み)が失敗する。`never` なので昇格もできない。動く permission の組み合わせを用意するか、案内付きで断るべき | 実際の問題で、直す価値もある。recipe(`docs/recipes/codex-seat-permissions.md`)には、「codex keeps `.git`, `.agents`, and `.codex` directories under a writable root read-only, recursively」と書かれている。`/director` skill の手順「機能ごとに org を立てる」は `ralph org start --plan` を打つ。その結果、start が成功を報告したあとで、director は自分の仕事の一部をこなせない。直し方は、権限を広げず、確かめていない挙動にも頼らない形にする。codex の director で、解決した mode が `autonomous` か `edits`(`workspace-write`)なら、`codex_verified` によらず、start が何も書かずに断る。案内には 2 つの道を書く。1 つは `[org.permissions.roles].director = "guarded"` にし、人の codex の設定で `.git` と network を許すこと。もう 1 つは claude の director を使うこと。断るので、`codexWritableRootArgs` の director の分岐は使われなくなる。分岐を消し、help、template の注記、計画を揃える。director だけを `danger-full-access` に写す案は、tech-debt に記録する | `internal/org/director_start.go`(`checkDirectorStart`)、`internal/org/permissions.go`(`codexWritableRootArgs`)、`internal/cli/org_director_headless.go`(help)、`templates/base/ralph.toml`、`/director` skill の 4 面、計画、`docs/tech-debt/README.md` |
| C2-2 | [P2] linked worktree から、相対の `[org].agmsg_home` を持つ `--config` で start すると、director の cwd は main のチェックアウトに移るが、受け取るのは設定のファイル名の絶対パスだけになる。`ResolveAgmsgHome` は相対のままにするので、director の後のコマンドは main のチェックアウトに対して解く。見張りは `orgWatchChildAgmsgHome` で元のディレクトリに対して解いた値を受け取る。そのため director と見張りが別の script の場所を使う | 実際の問題で、直す価値もある。条件は狭い。それでも director が `ralph org start` や `send` を打つと、script がない、または別の install を使う、という形で失敗するか食い違う。director の pane の env は herdr server のものなので、ralph からは env を足せない。直し方は、見張りと同じ判定(絶対パスでも `~/` でもない)で、`--config` の相対の `agmsg_home` を断ること。断るときは、絶対パスにするよう案内する。人の shell の `RALPH_ORG_AGMSG_HOME` も director の pane には届かない。座席にもある既存の穴なので、start で警告を 1 行出し、tech-debt に記録する | `internal/cli/org_director_headless.go`(`startDirector`)、そのテスト、`docs/tech-debt/README.md` |

### WORTH_CONSIDERING(cycle 2)

なし。

### DISMISSED(cycle 2)

なし。

## Cycle 1

- Date: 2026-10-15
- Cycle: 1/2
- Total reviewer findings: 1
- 仕分けの結果: ACTION_REQUIRED 1 件、WORTH_CONSIDERING 0 件、DISMISSED 0 件(cycle 1 のときの数。この報告の数の行は先頭の 1 行だけにしている)

### Triage context(cycle 1)

- Active plan: docs/plans/active/2026-10-15-org-director-start.md(承認済み、digest 671c60416856)
- Self-review report: docs/reports/self-review-2026-10-15-org-director-start.md(cycle 1 は b50d261b で Merge: yes。CRITICAL・HIGH なし、MEDIUM 1、LOW 6。da818b4f と dbd67a58 で直した)
- Verify report: docs/reports/verify-2026-10-15-org-director-start.md(cycle 1 は ec70c739 で pass。隔離した herdr server の実機で、codex と claude の start と stop を確かめた)
- Test report: docs/reports/test-2026-10-15-org-director-start.md(cycle 1 は 8461043a で pass。mutant 86 個のうち 81 個が red。生き残りから出たテストの穴 D4・E1・E12/E13・P2 は、sync-docs の e80ccafe で tech-debt に 1 行として記録した)
- Implementation context summary:
  - `codex exec review --base main` を e683ee47 で実行した。モデルは gpt-6-astra、effort は xhigh、sandbox は read-only。結果は `codex rc=0` で、`-o` のファイルは 1,002 バイト。
  - Codex は read-only の sandbox で Go のビルド用ディレクトリを作れず、Go のテストを回せていない。テストは cycle 1 の /test が回している。
  - 指摘の主張は、コードと照らして確かめた。`startDirector`(`internal/cli/org_director_headless.go`)は `configPath` を見張りの起動(`wireOrgStartWatchGate`)にだけ渡しており、`StartDirector` には渡していない。そのため `RenderDirectorPrompt` の雛形(`internal/org/prompts/director.md`)のコマンドは `--state-dir {{STATE_DIR}}` だけを持つ。
  - `ralph org director status` は `--config` がなく、台帳が main worktree のものなら、main worktree の `ralph.toml` を読む(`withMainWorktreeDirectorEnabled`)。人が `--config` で別の設定を渡して start した場合、director が最初に打つ status と、その後の `ralph org` の動詞は、start とは別の設定を読む。

### ACTION_REQUIRED(cycle 1)

| # | Reviewer finding | Triage rationale | Affected file(s) |
|---|-------------------|------------------|-------------------|
| C1-1 | [P2] `director start` に `--config` を渡しても、その設定は director の起動の雛形に入らない。director が最初に必ず打つ `ralph org director status --state-dir ...` は既定の `ralph.toml` を読む。たとえば、`--config` の設定は正しく、root の `ralph.toml` が壊れていると、start は成功を報告するが、director は自分の登録を確かめられない。`--config` を絶対パスにして雛形に入れ、設定を読む `ralph org` の動詞に付けるべき | 実際の問題で、直す価値もある。`--config` で `[org.director].enabled` や `[org.permissions]` を変えて start した人に対し、director が `enabled = false` の別の設定を読むと、status は disabled と答える。雛形の手順では、そのとき director は何もしない。壊れた `ralph.toml` なら、status はエラーで終わる。どちらも、start が成功を報告したあとで director が動けない。直し方は 2 つ。`DirectorPromptVars` に設定のパス(絶対パス、なければ空)を足し、雛形の `ralph org` のコマンドに `--config <path>` を付けて描く。`/director` skill の headless の節にも同じ規則を書く | `internal/cli/org_director_headless.go`(`startDirector`)、`internal/org/director_start.go`(`DirectorStartParams`、`RenderDirectorPrompt` の呼び出し)、`internal/org/prompts.go`(`DirectorPromptVars`)、`internal/org/prompts/director.md`、`/director` skill の 4 面 |

### WORTH_CONSIDERING(cycle 1)

なし。

### DISMISSED(cycle 1)

なし。
