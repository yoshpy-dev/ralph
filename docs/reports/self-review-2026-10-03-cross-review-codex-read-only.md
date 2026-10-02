# Self-review report: cross-review-codex-read-only

- Date: 2026-10-03
- Plan: docs/plans/active/2026-10-03-cross-review-codex-read-only.md
- Branch: security/cross-review-codex-read-only(HEAD a96c0ae5、実装のコミットは 4fd7bf5f)
- Reviewer: reviewer subagent (Claude)、cycle 1
- Scope: diff の品質だけ(正しさ、安全性、保守性、不要な変更、文の正確さ)。仕様への適合、テストの網羅、文書のずれは /verify・/test・/sync-docs の担当。対象は `git diff main...HEAD` から `docs/plans/` を除いた 14 ファイル

## Evidence reviewed

- 4 面の cross-review の 2 か所(`.claude/skills/cross-review/SKILL.md:58` の fenced block と `:167` の表)と、4 面の `/plan` の 1 か所(`.claude/skills/plan/SKILL.md:71`)の word diff。足したのは `-c sandbox_mode=read-only`(`exec` の前)と `--ignore-rules`(`exec review` / `exec` の直後)、それぞれの理由の 1 文(`:60`、`plan/SKILL.md:73`)だけ
- `.agents/` と `templates/base/` の写しを `.claude/` と `diff` で比べた。違いは cross-review の frontmatter の `allowed-tools` 行(既知の差)だけ。`.codex/config.toml` と `docs/recipes/codex-setup.md` は root と template が一致
- `tests/test-codex-exec-invocation.sh` の全文。足したのは `:16-25` のコメントと、`check_invocation_line` の 3 つの `case`(`:120-133`)
- `docs/tech-debt/README.md:147-148`。`<!-- RESOLVED … Row preserved for traceability. -->` と、取り消し線に `(RESOLVED <date> in <branch>)` を付けた行の組で、`:16-17` などの既存の慣習と同じ。表の区切りの `|` は main と同じ 6 個
- `git grep -n -E 'exec review|codex exec'`(reports、plans、insights、evidence を除く)。ほかの文書は呼び出しの形を書いていない(`README.md:285`、`.codex/README.md:63`、`post-implementation-pipeline.md:24` は「`codex exec review` を呼ぶ」とだけ書く)
- `codex exec review --help` と `codex exec --help`(0.154.0 と 0.159.2)。どちらの版も `exec` と `exec review` に `--ignore-rules` がある。`exec review` に `--sandbox` はない。0.154.0 の `exec` には `--approve-for-me`(「workspace-write の sandbox で承認を自動の review に回す」)と `--add-dir` がある

### probe

scratch に git repo を作り、この branch の `.codex/config.toml` を置いて、`main` に対して差分のある branch を作った。`HOME` と `CODEX_HOME` は scratch に向け、`config.toml` にはその repo を trusted にする記述だけを書いた(認証なし)。codex は絶対パスで呼び、10 秒の watchdog を付けた。header(`sandbox:`、`approval:`)は 401 の前に出る。`~/.codex` は読んでも書いてもいない。main のチェックアウトでは codex を動かしていない。

| probe | 実行 | 結果 |
| --- | --- | --- |
| R1 | 0.154.0、main の形(sandbox の指定なし)の `exec review` | `sandbox: danger-full-access`、`approval: never`(project の設定のトップレベルは `approval_policy = "on-request"`) |
| R2 | 0.154.0 と 0.159.2、新しい cross-review の形 | どちらも `sandbox: read-only`、`approval: never` |
| R3 | 0.154.0 と 0.159.2、新しい `/plan` の形(`exec --sandbox read-only --ignore-rules -o … "say hi"`) | どちらも `sandbox: read-only` |
| R4 | ユーザー設定に `default_permissions = ":danger-full-access"`、または `":workspace"` | 新しい形は `read-only`。override なしは `danger-full-access` |
| R5 | project の設定の `sandbox_mode` を `default_permissions = ":danger-full-access"` に置き換えた | override なしは `danger-full-access`、新しい形は `read-only` |
| R6 | ユーザー設定に旧式の `profile = "wide"` と `[profiles.wide] sandbox_mode = "danger-full-access"` | codex がエラーで終わる(rc 1、「legacy `profile` … is no longer supported」)。reviewer は incomplete になる |
| R7 | `$CODEX_HOME/wide.config.toml` に `sandbox_mode = "danger-full-access"`、`-p wide -c sandbox_mode=read-only exec review …` | `read-only`(`-c` が勝つ)。`exec review` の後ろに置いた `-p` は unexpected argument |
| R8 | `-s read-only` を `exec review` の前(root のオプション)に置いた | `read-only` |
| R9 | 新しい cross-review の形に `--yolo` を足した(0.154.0、0.159.2)。`/plan` の形に `--yolo` を足した | どれも `sandbox: danger-full-access`、`approval: never`。help に出ない別名で、`-c sandbox_mode=read-only` と `--sandbox read-only` のどちらにも勝つ |
| R10 | `--approve-for-me` を `exec review` の前(root)に置いた | `sandbox: workspace-write [workdir, /tmp, $TMPDIR]`、`approval: on-request`。`exec review` の後ろでは unexpected argument、`exec --sandbox read-only` と組むと「cannot be used with」で終わる |
| R11 | `exec review --full-auto`、`exec --add-dir <scratch>`、`-c 'sandbox_permissions=["disk-full-write-access"]'` | `--full-auto` は unexpected argument。残りの 2 つは header が `read-only` のまま |
| R12 | ユーザー設定に `[mcp_servers.probe]`(command は scratch のファイルへの `touch`)。新しい形で起動 | ファイルができた(reviewer の session が MCP server を起動した)。`-c 'mcp_servers={}'` を足してもできた。`-c mcp_servers.probe.enabled=false` か `--ignore-user-config` ではできなかった。`--ignore-user-config` のときは project の設定の warning も出なくなった(trust がユーザー設定にあるので、project の設定も読まれない) |
| R13 | `codex sandbox -P :read-only`(組み込みの read-only の permission profile) | 書き込みは `Operation not permitted`。workdir の外にある scratch のファイルの `cat` は rc 0。`curl` は名前解決に失敗 |
| mut | テストとその入力を scratch に写し、変異を入れて `sh` で実行 | 変更なしは 132 / 132。`.claude/skills/cross-review/SKILL.md:58` に `--yolo` を足しても 132 / 132。`/plan` の行に `--approve-for-me` を足しても 132 / 132 |

R4 と R5 は header だけを見ている。R13 は組み込みの `:read-only` が `sandbox_mode=read-only` と同じ制限だと仮定している(未確認)。実際の書き込みの拒否は、plan の AC-3b の記録(利用者の認証で実行)に頼っている。probe の前後で、worktree と main のチェックアウトの `git status --porcelain` は空のまま。

## Findings

| # | Severity | Area | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- | --- |
| M-1 | MEDIUM | テスト(回帰の検査の取りこぼし) | 広い sandbox を禁じる検査(`tests/test-codex-exec-invocation.sh:130-133`)は `danger-full-access`、`workspace-write`、`--dangerously-bypass` の 3 つの文字列しか見ない。bypass の別名の `--yolo` と、root に置いた `--approve-for-me` は、read-only の指定より強いのに通る。コメント(`:24-25`)は「広い sandbox や bypass の flag を書いてはいけない」と書き、plan の Deviation notes はこの検査を「後ろから広い sandbox で上書きする改変を止めるため」と説明しているので、検査の範囲が主張より狭い。起こりうること: reviewer にテストを動かさせたい人が `--yolo` を足すと、reviewer は `danger-full-access` と `approval: never` に戻り、テストは緑のまま | R9、R10、mut(どちらの変異も 132 / 132) | `case` に `*'--yolo'*` と `*'--approve-for-me'*` を足す。`--add-dir`、`-P` / `--permission-profile`、`default_permissions` も候補に入れる。禁止の一覧を増やす代わりに、既知のトークンを取り除いたあとに `-` で始まるオプションが残れば落とす許可の一覧の形にすると、codex の新しい別名にも効く。test report に、この 2 つの変異で落ちることを記録する |
| M-2 | MEDIUM | 文の正確さと残るリスクの記録 | 新しい文は境界を全体として書いている。`.claude/skills/cross-review/SKILL.md:60`(4 面)は「`-c sandbox_mode=read-only` keeps the reviewer from writing」、`docs/recipes/codex-setup.md:93-101`(+ template)も同じ読み方になる。read-only の sandbox が止めるのは、codex が seatbelt の下で動かすコマンドと編集の書き込みで、次の 3 つは範囲の外にある。(a) ユーザー設定の MCP server は reviewer の session でも起動する(R12)。MCP の tool は server のプロセスで動くので、sandbox は掛からない。`approval: never` のときに review の model が承認なしで tool を呼べるかは未確認(認証が要る)。(b) 読み込みは止まらない(R13)。利用者が読めるファイルの中身が reviewer の出力に入れば、`docs/reports/cross-review-triage-*.md` を通ってコミットされ、push される。その間にあるのは、形で見つける secret scan だけ。(c) 逆向きの claude の reviewer(`--permission-mode auto`)は plan の Open questions にしか書かれていない。plan は `/pr` で archive され、どれも register にも issue にも残らない。起こりうること: 書き込みのできる MCP server をユーザー設定に入れている下流の利用者が、この文を読んで reviewer は何も書けないと判断する | R12、R13、`.claude/skills/cross-review/SKILL.md:60`、`docs/recipes/codex-setup.md:93-101`、plan の Open questions | 4 面と recipe の文を「reviewer が動かすコマンドと編集を書き込みのできない sandbox に入れる」の範囲に絞る。(a)〜(c) を 1 つの tech-debt の行(または follow-up の issue)に記録する。見つかった手段も添える: server ごとの `-c mcp_servers.<name>.enabled=false` は効き、`-c 'mcp_servers={}'` は効かない。`--ignore-user-config` は MCP server を止めるが、project の設定と trust も読まなくなり、ユーザー設定の model provider なども外れるので、設計の判断が要る(R12)。merge は止めない |
| L-1 | LOW | 配る設定のコメント | `.codex/config.toml:19-20` は「danger-full-access は approval_policy と組んで破壊的なコマンドを止める」、`:25` は「on-request … (default for ralph)」と書く。`codex exec` は project の `approval_policy = "on-request"` があっても `approval: never` で動く(R1)。この PR は recipe に「`codex exec` never asks for approval」(`docs/recipes/codex-setup.md:98`)と書き、同じ設定ファイルも編集したので、二つの文が同じ PR の中で食い違う。コメントはこの diff より前からあり、template から下流に届く | R1、`.codex/config.toml:19-27`、`docs/recipes/codex-setup.md:98` | `:19-20` に「対話の session のときだけ。`codex exec` は承認を求めないので、ralph の `exec` の呼び出しは read-only の sandbox を渡す」の趣旨の一文を足す(root と template)。この PR で直さないなら M-2 の tech-debt の行に含める |
| L-2 | LOW | register の追跡 | 解決のコメント(`docs/tech-debt/README.md:147`)は根拠を「plan AC-3, AC-3b」とだけ書き、パスがない。`:148` の Related の列は PR #198 の report のままで、この PR の plan も report も指していない。認証を使った実行と AC-3b の再現の記録は、plan の Deviation notes にしかない。`/pr` のあと、その plan は `docs/plans/archive/2026-10-03-cross-review-codex-read-only.md` に移る | `docs/tech-debt/README.md:147-148`、plan の Deviation notes | コメントか Related の列に、archive 後の plan のパスとこの report のパスを書く |

CRITICAL と HIGH はない。

## 依頼された観点への回答

- read-only から抜ける経路のうち、#197 の範囲のもの: `-c` の並び(後ろの `-c sandbox_mode=…` は M-1 の検査が `danger-full-access` と `workspace-write` の文字列で止める)、project の設定(R5: `default_permissions` を書いても `-c` が勝つ)、ユーザー設定の `default_permissions`(R4)、v2 の profile(R7)、旧式の `profile`(R6: エラーで止まる)、execpolicy の allow のルール(`--ignore-rules`、AC-3b)。どれも新しい形で塞がっている。flag で上書きする経路は 2 つ残る(`--yolo`、root の `--approve-for-me`)。今の skill の行にはないので、M-1 として回帰の検査の取りこぼしとして扱った
- `approval_policy`: `exec` は project の `on-request` を無視して `never` で動く(R1)。read-only と組むと、承認の要るコマンドは承認を求めずに拒否される。抜け道にはならない。L-1 は、この事実と設定のコメントが食い違うことの指摘
- ネットワーク: read-only のコマンドは名前解決に失敗した(R13)。読んだ中身をコマンドで外に出す経路は塞がっている。残るのは reviewer 自身の出力(M-2 (b))と、model の側の `web_search = "cached"`(`.codex/config.toml:31`)。後者は検索の問い合わせが OpenAI に行くだけで、model の provider と同じ相手になる
- MCP: ユーザー設定の server は reviewer でも起動する(R12)。#197 の範囲の外の残るリスクとして M-2 (a) に書いた
- `shell_environment_policy`: この repo の `.codex/config.toml` には書かれていない。環境変数を読めても、外に出す経路は (b) と同じ出力だけなので、M-2 (b) に含まれる
- `--ignore-rules` の置き場所: 0.154.0 と 0.159.2 の両方で、`exec` と `exec review` のオプションとして受け付けられる(R2、R3)。skill の行の並びは plan の Deviation notes に記録された実際の実行と同じ。`--ignore-rules` は deny や prompt のルールも読まなくなるが、read-only でネットワークもない状態では、それで失われる守りはない
- 新しいテストの substring の検査: 検査は呼び出しの行だけを見る(`grep -n 'codex' | grep ' exec '` で選んだ行)。`:60` の理由の文は `codex exec` が backtick で閉じているので選ばれず、文書の文が禁止の検査に掛かることはない。read-only の検査はどちらの書き方もどちらの skill で受け付けるが、`exec review` の後ろの `--sandbox` は codex が拒否し、root の `-s read-only` は効く(R8)ので、穴にはならない。取りこぼしは M-1 の 2 つ
- 文書の正確さ: `.codex/config.toml:62-64` のコメントと recipe の呼び出しの説明は実際の行と合う。範囲を広く書きすぎた 1 文は M-2、設定のコメントの食い違いは L-1
- tech-debt の行: 慣習どおり。根拠の参照先は L-2
- read-only の reviewer がリポジトリの外を読めること: 記録する価値がある(R13)。出力がコミットされる report に流れるので、M-2 の tech-debt の行に (b) として入れることを勧める

## Positive notes

- 8 か所の呼び出しと 4 面の写しが一致していて、template とのずれもない
- AC-3b は、ルールを置いたときに回り込みが起きることを先に再現してから、`--ignore-rules` で止まることを確かめている。再現できなかった場合の扱いまで AC に書いてあった
- 新しい形で失敗するときは閉じる側に倒れる。旧式の `profile`(R6)、`--full-auto`(R11)、置き場所を誤った `--sandbox` や `--approve-for-me`(R10)はどれも codex が rc 0 以外で終わり、skill の reviewer incomplete の経路に入る

## Tech debt identified

| Debt item | Impact | Why deferred | Trigger to pay down | Related plan/report |
| --- | --- | --- | --- | --- |
| read-only の sandbox の外に残る、codex の reviewer と advisory の経路。(a) ユーザー設定の MCP server は reviewer の session でも起動し、その tool には sandbox が掛からない(`approval: never` で呼べるかは未確認)。(b) 利用者が読めるものは何でも読め、その中身は出力を通って `docs/reports/cross-review-triage-*.md` に入り、コミットされる。(c) 逆向きの claude の reviewer は `--permission-mode auto` で動く | 書き込みのできる MCP server をユーザー設定に入れている利用者では、差分に紛れた指示で reviewer が MCP 経由の操作をしうる。reviewer の出力に混ざった秘密は、形で見つける secret scan を通れば push される | #197 の範囲はコマンドの sandbox と execpolicy のルールに限っていた。MCP を止める手段は、server ごとの `enabled=false`(名前が要る)か `--ignore-user-config`(project の設定と trust も読まなくなる)で、どちらも設計の判断が要る | 次に `/cross-review` か `/plan` の codex の呼び出しを変えるとき。`codex exec` の MCP の tool の承認の扱いが確かめられたとき。または、MCP の tool を使った reviewer の操作が報告されたとき | この report(M-2、R12、R13)、plan の Open questions |

_(この行は report の中だけにある。この phase では report と insights の events のほかを stage しないので、`docs/tech-debt/README.md` への追記は sync-docs か orchestrator に任せる。)_

## Known gaps

- probe は認証のない環境で、header と起動の時点までしか見ていない。model が MCP の tool を呼べるか、read-only の下で review が最後まで進むかは、plan の AC-3 の記録に頼っている
- R13 は組み込みの `:read-only` の permission profile で確かめた。`-c sandbox_mode=read-only` が同じ seatbelt の規則になるかは未確認
- `.codex/config.toml` は `[features] hooks = true` を有効にしているので、trust 済みの project の hook は reviewer の session でも動くと思われる。hook はこの repo のコードで、主の session も動かすものなので新しい権限ではないが、sandbox の中で動くかは確かめていない
- `--ignore-rules` のない古い codex では、呼び出しが unexpected argument で終わり、毎回 reviewer incomplete になる。どの版から `--ignore-rules` があるかは調べていない
- 利用者の実際の `~/.codex` の設定(MCP server、`.rules`)は読んでいないので、この環境で M-2 (a) が起きるかは分からない

## Recommendation

- Merge: 可。CRITICAL と HIGH はない。M-1 と M-2 は merge を止めないが、直す費用が小さいので、cross-review の結果と合わせて PR の前に直すことを勧める
- Follow-ups: (1) M-1 の禁止の一覧に `--yolo` と `--approve-for-me` を足すか、許可の一覧の形にする。test report に 2 つの変異を記録する。(2) M-2 の文の範囲を 4 面と recipe で絞り、上の tech-debt の行を `docs/tech-debt/README.md` に足す(sync-docs)。(3) L-1 の設定のコメントを直すか、(2) の行に含める。(4) L-2 の解決のコメントに archive 後の plan とこの report のパスを書く
