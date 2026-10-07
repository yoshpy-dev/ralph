# sync-docs report: org-state-dir-common

## Cycle 1

- Date: 2026-10-07
- Plan: `docs/plans/active/2026-10-07-org-state-dir-common.md`
- Pipeline cycle: 1 of 2。差分は merge base `1c4cea5a` から branch HEAD `5672ff92`(fix/org-state-dir-common)まで
- 先行 report(それぞれを追加した commit):
  `docs/reports/self-review-2026-10-07-org-state-dir-common.md`(`477cbf6b` で追加、`4961d0f5` で更新。Merge 可)、
  `docs/reports/verify-2026-10-07-org-state-dir-common.md`(`43896feb`。pass、LOW 3 件 V-1〜V-3)、
  `docs/reports/test-2026-10-07-org-state-dir-common.md`(`5672ff92`。pass、mutation 32 件中 30 件が red)

## Summary

この差分で古くなった文書は、tech-debt の State dir の行(L-5、D-1)、director spec の 3 か所(D-2)、`/org` skill の git の版の扱い、recipe の折り返し(V-3)だった。依頼のあった 1〜5 はそのとおり直した。6 は worktrees recipe に 1 文足した。7 の掃引では、`docs/architecture/repo-map.md` の `.harness/state/org/` の行と、plan の進捗に手を入れた。AGENTS.md の Repo map、README、`docs/quality/`、`.claude/rules/ralph/` は、台帳の置き場所の解決順を書いていないので変更していない。

tech-debt の既存の行 1 件を更新し(open のまま)、新しい行 2 件を足した。

## Changes made

| File | Change |
|------|--------|
| `docs/tech-debt/README.md` | (1) 144 行目の「config と state dir が別の規則で解決される」行を更新。state dir の順を「flag、env、main worktree のルート、git の toplevel、cwd」に直し、main worktree のルートの決め方を括弧で足した。linked worktree のルートから打つと config は worktree 側の `./ralph.toml`、台帳は main 側になり、ずれが広がったことを 1 文足した(`max_seats` の検査が台帳の動いている座席に対して行われることからの推論で、実行はしていない)。「Why deferred」に、この plan が台帳の置き場所だけを変え config の探し方は範囲外なので open のまま残す、と書いた。trigger を「`resolveOrgConfig` の変更、または `ResolveOrgStateDir` のさらなる変更」にし、Related に plan と verify の D-1 を足した。(2) 末尾に 2 行を追加(下記) |
| `docs/specs/2026-10-07-org-multi-org-director.md` | D-2 の 3 か所。Current state の台帳の記述を「1 段目の前は show-toplevel から決まり、worktree ごとに別の台帳ができた。1 段目の PR で main worktree のルートから決めるように変え、linked worktree からも同じ台帳を使う」に。FR-1 に、ルートは main worktree の中では show-toplevel、linked worktree の中では `git worktree list --porcelain` の先頭の記録、と足した。Research findings の `--add-dir` を「leader の座席にだけ」に。加えて Open questions の `--add-dir` の項を、`codex exec` の sandbox では codex-cli 0.160.0(macOS)で確かめ済み(evidence と recipe を指す)、herdr で起動した leader 座席では未確認、に更新した。FR-1 のチェックボックスと Rollout の「進行中」は、マージ前なのでそのまま |
| `.claude/skills/org/SKILL.md` と 3 つの写し | 前提節の state dir の箇条に 1 文(4 行)を足した。git が 2.31 より古く `--path-format=absolute` を解釈できない環境では main worktree を取れず show-toplevel に落ち、変更前と同じく worktree ごとに台帳が分かれ、古い台帳の注意も拒否も出ない。`scripts/sync-skills.sh` で `.agents/skills/org/SKILL.md` を作り直し、`cp` で `templates/base/` の 2 面を揃えた |
| `docs/recipes/codex-seat-permissions.md` と `templates/base/` の写し | V-3。「The org ledger and `--add-dir`」節の最後の 5 行を 75 桁以内に折り直した(91 桁の行は 74 桁になった)。文は変えていない。2 面は `cp` で byte 一致 |
| `docs/recipes/worktrees.md` と `templates/base/` の写し | 「Org runtime」節に 1 文足した。flag も env もなければ台帳は main worktree にあり、linked worktree から `ralph org` を打っても全 task worktree が 1 つの台帳を共有する。例外は `/org` skill の前提節を指す。この recipe は manifest の場所を書いていたので、誤りを直したのではなく、worktree をまたぐ性質を足した |
| `docs/architecture/repo-map.md` | `.harness/state/org/` の行に「main worktree に 1 つ、linked worktree と共有」を足した。root だけのファイルで、template に写しはない |
| `docs/plans/active/2026-10-07-org-state-dir-common.md` | 進捗の節だけを変えた。AC6 以外の AC(1〜5、7〜9)と Review / Verification / Test artifact にチェックを付け、進捗の行を 2 件足した。AC6 は verify の V-1(main のチェックアウトで台帳のディレクトリが読めないとき、注意が linked worktree のものとして出る)が残るのでチェックなし。`Progress checklist` の節とチェックボックスは承認 digest の対象外で、`./scripts/plan-visual.sh digest` は変更後も `79224e28032a`(plan の Approved 行と一致)。「PR created」は付けていない |
| `docs/insights/events/2026-10-07-org-state-dir-common.jsonl` | `sync_docs` の event を 1 行追記(verdict pass、cycle 1) |
| `docs/reports/sync-docs-2026-10-07-org-state-dir-common.md` | この report |

### 追加した tech-debt 行

| 行 | 内容 | 既存の行との関係 |
|----|------|------------------|
| 161 行目 | V-1: `LegacyWorktreeStateDir` が manifest を stat してから `samePath` を見るので、main のチェックアウトの台帳のディレクトリが読めないと、注意と拒否が「linked worktree の古い台帳」として出る。直し方は `samePath` を stat の前に動かし、テストを足す。同じ行に V-2(読むだけの動詞の「読めない」注意に `--state-dir` / `RALPH_ORG_STATE_DIR` の案内がない)も入れた。V-2 は依頼に含まれていなかったが、verify の LOW で、同じ箇所の文言なので同じ行にした | 重なる行はない |
| 162 行目 | test の生き残った mutation 2 件。(a) `statedir.go:104` の stat が ENOENT 以外で失敗する経路(M08b)。本番の挙動は e2e の E7 で正しいことを確かめてあるが、回帰を止めるテストがない。(b) `pathWithinDir` の `rel != ".."`(M29)。台帳が座席の cwd の直接の親になる場合で、leader の cwd が台帳の中になることはまずない。足すテストの形(`chmod 000` の例、cwd が `<台帳>/prompts` の例)まで書いた | 161 行目の V-1 と同じ関数。V-1 を直すときに (a) を一緒に足す |

どちらの行も「なぜ今直さないか」は、`/sync-docs` が文書しか変えず、コードやテストを足すと post-implementation pipeline が `/self-review` からやり直しになるため、と書いた。Related には `docs/plans/active/2026-10-07-org-state-dir-common.md` を使った(`/pr` の archive-plan.sh が `archive/` に書き換える)。

## Surfaces checked for drift

| Surface | Finding |
|---------|---------|
| `AGENTS.md` の Repo map、`.ralph/core/AGENTS.core.md`、`CLAUDE.md`、`README.md` | `internal/org/` の行と org runtime の記述は台帳の置き場所の解決順を持たない。変更なし |
| `.claude/rules/ralph/`(`agent-messaging.md`、`git-commit-strategy.md`、`model-routing.md` ほか) | `.harness/state/org/` を指す箇所は場所の名前だけで、解決順は `model-routing.md:100` の「`ralph org` の動詞と同じ順」のみ。順を持たないので食い違わない |
| `docs/quality/definition-of-done.md`、`quality-gates.md` | `quality-gates.md:82` の manifest の言及は場所の名前だけ。変更なし |
| `docs/specs/2026-08-01-org-runtime.md` | FR-9 が manifest の場所を書くだけで、解決順はない。変更なし |
| `/org` skill の 4 面 | 前提節(解決順、古い台帳、`--separate-git-dir` の制約、git の版)と「既定の model_pool」節(「`--state-dir`、`RALPH_ORG_STATE_DIR`、main worktree のルート、git の toplevel、cwd の順」)が実装の順に一致。4 面は `cmp` と `check-skill-sync.sh` で一致 |
| `--state-dir` のヘルプ文(`internal/cli/org.go:42`、`status.go:56`)、`ralph insights` の Long と `--receipts` のヘルプ、`statedir.go` のコメント | 新しい順を書いている。verify が確認済みで、今回は変更していない |
| `docs/recipes/codex-seat-permissions.md` の `--add-dir` の節 | leader だけ、guarded と claude には付かない、とコードに一致 |
| 古い解決順の grep(`git grep -n -i 'toplevel'` ほか。archive、reports、evidence、plans の履歴を除く) | 古い順を書いたまま残る箇所は tech-debt 144 行目だけだった(上で更新)。残りは hook の `cd "$(git rev-parse --show-toplevel)"` など、台帳と無関係の用法 |
| `docs/evidence/org-watchdog-smoke-2026-08-02.txt:34`、tech-debt の RESOLVED 行(73、81 行目) | 履歴なので触っていない |

## Found but left

- spec の FR-1 のチェックボックスと Rollout の「共通の台帳(FR-1、進行中)」は変えていない。PR が作られてマージされるまで実装済みとは言えず、「進行中」を直すのはマージ後か `/pr` の側になる。
- git が 2.31 より古い環境の扱いは `/org` skill の 1 文だけにした。recipe、README、`ralph doctor` には書いていない。`internal/cli/git_hooks.go` と `scripts/secret-scan.sh` が同じ下限にすでに依存していることは self-review が確認済みだが、最低限の git の版を ralph が文書で宣言しているかは調べていない。
- verify の Coverage gaps にある、herdr で起動した codex leader が共通の台帳に書けること、名前が `.git` でない bare リポジトリのテスト、Windows のパスの扱いは、tech-debt に足していない。依頼の範囲外で、test / verify の report に残っている。
- `docs/tech-debt/README.md` の既存の行は、144 行目のほかには触っていない。

## Checks run

| Command | Result |
|---------|--------|
| `./scripts/sync-skills.sh` | 13 skill を `.agents/skills` に再生成 |
| `./scripts/check-skill-sync.sh` | PASS(13 skill が一致) |
| `./scripts/check-sync.sh` | PASS(IDENTICAL 164、DRIFTED 0、ROOT_ONLY 0、TEMPLATE_ONLY 11、KNOWN_DIFF 5) |
| `./scripts/plan-visual.sh digest docs/plans/active/2026-10-07-org-state-dir-common.md` | `79224e28032a`(plan の Approved 行と一致) |
| `./scripts/run-static-verify.sh` | PASS(rc 0。`verify.local.sh` の全項目、tech-debt README の plan 参照、gofmt、golangci-lint 0 issues、branch secret scan clean) |
| `git diff --check` | 出力なし |
| 差分内の U+FFFD 検索 | 0 件 |

注: director spec の最初の 3 か所は、`Edit` ではなく python の文字列置換で書いた(各置換元が 1 回だけ現れることを確かめてから)。`Edit` / `Write` にしか掛からない mojibake の hook を通っていないので、差分全体の U+FFFD を別に検索した(0 件)。残りの編集は `Edit` で行った。
