# 機能ごとの org の実機確認(4 段目の AC14)

- Date: 2026-10-09
- Plan: docs/plans/active/2026-10-09-org-feature-worktree.md(AC14、S5)
- 目的: 承認済みの分割計画から `ralph org start --plan --feature` で立てた org で、leader が機能の worktree の中で動き、implementer と reviewer を 1 席ずつ立てることを確かめる

## 環境

| 項目 | 値 |
|---|---|
| ralph | このブランチをビルドしたもの(`go build ./cmd/ralph`、run 3 は af138af6) |
| herdr | 0.7.5。隔離した `herdr server`(`XDG_CONFIG_HOME` / `XDG_STATE_HOME` / `HERDR_SOCKET_PATH` を `/tmp/rs5` の下に向けた)。利用者の herdr サーバーと `~/.config/herdr/` には触れていない |
| Claude Code | 2.1.289。leader は `--model sonnet`、implementer と reviewer は leader が `--model haiku` で立てた(分割計画の本文で指定) |
| repo | `/tmp/rs5/repo`。`git init -b main` のあと `ralph init --yes` で scaffold し、remote なし |
| 分割計画 | `.harness/state/org/splits/s5.md`。機能 `hello`(`Type: docs`、`Reserve: docs/hello.md`)。digest は `scripts/plan-visual.sh digest` で出し、`- Approved:` に書いた |

## Run 1: pane の ralph が古い版だった

herdr サーバーを `ZDOTDIR` だけ空のディレクトリに向けて起動した。pane のシェルは利用者の zsh の設定を読み、PATH を組み直したので、pane の中の `ralph` は Homebrew の v5.1.0(`/opt/homebrew/bin/ralph`)に解決された。shell の alias(`claude --model opus --effort xhigh ...`)も展開され、座席は effort xhigh で動いた。

- `ralph org start --plan` は今回のバイナリで打ったので、worktree(`.claude/worktrees/org-hello`、ブランチ `docs/hello`)と leader の記録は main の台帳に入った
- leader が pane の中で打った最初の `ralph org spawn`(v5.1.0)は、1 段目より前の動きで、worktree の中の `.harness/state/org/manifest.jsonl` に implementer と reviewer を書いた
- leader はそれに気づき、2 つの座席を止めて disband し、`RALPH_ORG_STATE_DIR` で main の台帳を指して立て直した

これは未 release のバイナリを試したことから来るずれ。ただ、`start` を打った ralph と leader が pane の中で呼ぶ ralph の版が違うと台帳が分かれる、という穴は残る(tech-debt に送る)。

## Run 2: 座席の `--cwd .` が herdr サーバーの cwd で解決された

herdr サーバーを `SHELL=/bin/bash` で起動し直した。bash のログインシェルでは、`ralph` は今回のバイナリ、`claude` は alias なしの本体に解決された(effort の表示は low)。

leader は worktree で動き、雛形のとおり `ralph org spawn ... --cwd .` で implementer と reviewer を立てた。座席は 3 つ立ったが、`herdr pane get` で見ると pane の cwd は次のとおりだった。

| pane | cwd |
|---|---|
| leader(w2:p2) | `/private/tmp/rs5/repo/.claude/worktrees/org-hello` |
| implementer(w2:p3) | `/private/tmp/rs5`(herdr サーバーの cwd) |
| reviewer(w2:p4) | `/private/tmp/rs5` |

ralph は `--cwd .` を herdr にそのまま渡し、herdr がそれをサーバーの cwd から解決していた。台帳の `spawned` の `worktree` も `.` のままだった(`ralph org watch` は座席の worktree で `git status` を打つので、watch を打った場所が基準になる)。この穴は 4 段目の前からあり、`/org` skill の例の `--cwd .` でも起きる。af138af6 で、spawn の入口で相対の `--cwd` を打った場所から絶対パスにしてから herdr に渡し、台帳にも絶対パスで記録するよう直した。座席は worktree の外に何も書く前に止めた。

## Run 3: 期待どおりに動いた

Run 2 と同じ herdr サーバーで、af138af6 のバイナリに作り直して打った。

```
$ ralph org start --plan .harness/state/org/splits/s5.md --feature hello --driver claude --model sonnet
spawned seat "leader" (org_id=hello driver=claude model=sonnet pane_id=w3:p2 dry_run=false)
worktree: /private/tmp/rs5/repo/.claude/worktrees/org-hello
branch: docs/hello
hint: ralph org status --org-id hello ; attach with herdr to observe the leader pane
```

`ralph org status --org-id hello`(3 席がそろったとき):

```
SEAT_ID      ROLE         DRIVER  MODEL   STATE             PANE_ID
implementer  implementer  claude  haiku   spawned (active)  w3:p3
leader       leader       claude  sonnet  spawned (active)  w3:p2
reviewer     reviewer     claude  haiku   spawned (active)  w3:p4
reserved: docs/hello.md
feature: s5/hello branch docs/hello worktree /private/tmp/rs5/repo/.claude/worktrees/org-hello
```

- 台帳は main の `.harness/state/org/manifest.jsonl` だけで、worktree の中には台帳ができなかった
- 3 席の `spawned` の `worktree` と、`herdr pane get` の pane の cwd は、どちらも機能の worktree の絶対パスだった
- 予約の記録: `paths=docs/hello.md split=s5 feature=hello digest=3de9af9f3dac branch=docs/hello`(`worktree` フィールドに worktree の絶対パス)

leader は雛形の「機能ごとの org」の順に進めた。worktree のブランチ `docs/hello` の履歴は次のとおり。

```
9545385 docs: add hello org report and archive plan
99431d0 wip: checkpoint before session end
dce1cbe wip: checkpoint before session end
f13116e docs: add hello
119a496 docs: add hello feature plan
e0c493b chore: scaffold
```

1. 機能の計画を `docs/plans/active/` に書いてコミットした(119a496)
2. implementer に TASK を送り、implementer が `docs/hello.md` を作ってコミットした(f13116e、1 行目は `hello from the hello org`)
3. reviewer に TASK を送り、reviewer がゲートを実行して GATE: pass を返した
4. implementer と reviewer を止め、`ralph org report` を打ち、計画を `scripts/archive-plan.sh` で移してコミットした(9545385)
5. `scripts/secret-scan-branch.sh --strict` のあと `git push -u origin docs/hello` を打ち、remote がないので失敗した。`gh pr create` は打たず、失敗を pane に書いて人の判断を待った(分割計画の本文に「失敗したら、その旨を pane に書いて止まる」と書いておいた)

2 つの `wip: checkpoint before session end` は、reviewer が書いた `docs/reports/self-review-2026-10-09-hello.md` をコミットしないまま止められたとき、scaffold の Stop hook がコミットしたもの。PR には入るが `wip:` の名前で残る(tech-debt に送る)。

確認のあと、外から `ralph org disband --org-id hello` を打った(3 席とも stopped、workspace は 0 個)。隔離サーバーは `herdr server stop` で止めた。

## Run 4: 既定でない台帳で、leader が `--state-dir` を付ける(2026-10-10)

cross-review の cycle 1 の ACTION_REQUIRED #1(leader に台帳の場所が渡らない)の直し(d49bbc34、07d38e6d、9c1d447f)を確かめた。バイナリは 9a2dc5ef をビルドしたもの。Run 3 と同じ形の隔離サーバー(`/tmp/rs6`、`SHELL=/bin/bash`)と、新しく scaffold した repo(`/tmp/rs6/repo`、remote なし)を使った。台帳は repo の外の `/tmp/rs6/ledger` に置き、分割計画もその `splits/s6.md` に置いた。

```
$ ralph org start --state-dir /tmp/rs6/ledger --plan /tmp/rs6/ledger/splits/s6.md --feature hello --driver claude --model sonnet
spawned seat "leader" (org_id=hello driver=claude model=sonnet pane_id=w1:p2 dry_run=false)
worktree: /private/tmp/rs6/repo/.claude/worktrees/org-hello
branch: docs/hello
```

leader は初めて開くフォルダの信頼の確認で止まったので、`herdr pane send-keys` で「Yes, I trust this folder」を選んだ。そのあとは人の操作なしに進んだ。

- `ralph org status --state-dir /tmp/rs6/ledger --org-id hello` に、leader(sonnet)、implementer と reviewer(haiku)の 3 席が `spawned (active)` で並び、`reserved:` と `feature: s6/hello ...` の行が出た。3 席の `spawned` の `worktree` は、どれも機能の worktree の絶対パスだった
- 既定の台帳は作られなかった。`/tmp/rs6/repo/.harness/state/org/` も、worktree の中の `.harness/state/org/` もなかった。leader が `--state-dir` を付けずに `ralph org spawn` を打っていれば、座席はどちらかに記録されていた
- leader は Run 3 と同じ順に、計画のコミット(58e83a8)、implementer の `docs/hello.md`(864cb22)、reviewer の GATE: pass、座席の stop、report と計画の archive のコミット(d600e37)、secret scan、push(remote がないので失敗)まで進め、失敗を pane に書いて止まった
- worktree に残った `docs/reports/org-manifest-hello-2026-10-09.md` には 3 席が並んでいた。leader の `ralph org report` も `--state-dir` の台帳を読んだ
- reviewer のレポートは、Run 3 と同じく `wip: checkpoint before session end`(b884147)でコミットされた

確認のあと、外から `ralph org disband --state-dir /tmp/rs6/ledger --org-id hello` を打ち(workspace は 0 個)、隔離サーバーを止めて `/tmp/rs6` を消した。

## 結論

- `ralph org start --plan --feature` は、機能の worktree とブランチを作り、その中に headless の leader を立てる。leader は implementer と reviewer を 1 席ずつ立て、台帳は main の 1 つに集まる(AC14)
- 相対の `--cwd` は herdr サーバーの cwd で解決されていた。af138af6 で、打った場所を基準に絶対パスにするよう直した
- 既定でない台帳で start しても、leader は task の `- 台帳:` の行に従って `ralph org` のコマンドに `--state-dir` を付け、座席と report は同じ台帳を使った(Run 4)
- 残すもの: (1) start を打った ralph と pane の中の ralph の版が違うと台帳が分かれうる。(2) reviewer のレポートが Stop hook の `wip:` コミットで入る
- 確かめていないもの: push と `gh pr create` が通る場合(remote のある repo)。codex の leader
- 副作用: Claude Code のフォルダの信頼(run 1 で `/private/tmp/rs5/repo/.claude/worktrees/org-hello` を信頼した)が利用者の Claude Code の設定に残る
