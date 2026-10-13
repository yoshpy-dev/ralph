# sync-docs report: org-merge-order

- Date: 2026-10-14(JST)
- Plan: `docs/plans/active/2026-10-14-org-merge-order.md`(承認済み。`./scripts/plan-visual.sh digest` は `c0b1e9230999` で、`- Approved:` の行と同じ。この run が変えたのは checkbox の印だけ)
- Pipeline cycle: 1(`--cycle auto`。`.harness/state/standard-pipeline/cycle-count.json` は 1)。差分の base は `ac34ccf3`、branch は `feat/org-merge-order`。run 開始時の HEAD は `e20f2616`(run 開始時の値で、この report を含む commit の SHA ではない)
- 先行 report: `docs/reports/self-review-2026-10-14-org-merge-order.md`(Merge: yes、CRITICAL・HIGH・MEDIUM 0、LOW 8)、`docs/reports/verify-2026-10-14-org-merge-order.md`(pass。文書のずれ D1 と D2)、`docs/reports/test-2026-10-14-org-merge-order.md`(pass、失敗 0)
- 項目の番号: D1・D2 は verify の「Documentation drift」の表の番号
- 内容の commit: この report、直した文書、計画の checkbox を含む 1 つの commit(`docs: sync docs for org-merge-order`)。insight event の `docs/insights/events/2026-10-13-org-merge-order.jsonl` は、指示どおりコミットに入れていない(未追跡のまま)

## Summary

verify が挙げた D1(README の分類の順)と D2(tech-debt の Trigger)を文書だけで直した。動詞の表(`README.md:124`)、AGENTS.md の repo map、仕様 FR-8 の注、`/org` skill の 4 面、`docs/architecture/repo-map.md`、`.claude/rules/ralph/` は、コードと合っていたので変えていない。

そのほかに、見張りなしで動く動詞を挙げる文書を掃いて、1 件を足した。`docs/quality/quality-gates.md` の「Cross-org watch gate」の行(と template の写し)が `merge-order` を挙げていなかった。同じ文が 7a の `approve` も挙げていなかったので、一緒に足した(skill の一覧は両方を挙げ、どちらも見張りの関門を呼ばない)。

計画は、依頼どおり checkbox の印(AC1 から AC11、Review・Verification・Test の artifact)だけを変えた。Go のコード、テスト、`--help` の文字列、leader と reviewer の雛形、skill、`scripts/`、AGENTS.md、仕様、ほかの報告は触っていない。

## Changes made

| File | D | Change |
|------|---|--------|
| `README.md:262`(`ralph org merge-order` の段落の 3 文目) | D1 | 「Each branch gets the first class that applies, and the list is in this order: ...」は並びの順だけを挙げ、「最初に当てはまる」をどの順で調べるかを言わなかった。並びの順で読むと判定と食い違う(push していない gate なしの枝は `no_gate`、並びでは `no_remote` が先)。「checked in this order: `merged` or `closed`, `unreadable`, `no_gate`, `failed`, `no_remote`, `regate`, `wrong_base`, `rebase`, and `ready` for a branch none of those fits」と判定の順を先に言い、次の文で「The list is in a different order: `ready`, `rebase`, `wrong_base`, `regate`, `unreadable`, `failed`, `no_remote`, `no_gate`, and with `--all` `merged` and `closed`, which it leaves out otherwise」と並びの順を分けた。分類ごとの括弧の説明(`unreadable`、`no_remote`、`regate`、`wrong_base`、`rebase`、`merged`・`closed` が `gh` でだけ分かること)は、最初に名前が出る判定の順の文に移した。判定の順は `classifyMergeTarget` と `classifyGatePassed`(`internal/org/mergeorder.go`)、並びの順は `mergeClassOrder` から取り、skill の `merge-order` の行(`.claude/skills/org/SKILL.md:183`)と仕様 FR-8 の 7c-1 の注と同じ順になる。段落のほかの文は変えていない |
| `docs/tech-debt/README.md:231`(「Small findings of org-gate cycle 2」の行の Trigger の列) | D2 | Trigger の「and stage 7c, which reuses these files.」のあとに、この trigger が 7c-1(`feat/org-merge-order`)で発火し、払われずに持ち越されたことを足した。`internal/org/mergeorder.go` は `gate.go` から `gateRecord`・`gateRecordFromDetails`・`gitObjectPattern` を読むだけで、`gate.go`・`gate_run.go`・`lockfile.go` は 1 つも変えず(スクリプトを走らせず、ロックも取らない)、(a) から (i) のどれにも触れていない。次にそれらのファイルを変えるときは今も trigger になる。行は消していない。5 列、変わったのは Trigger の列だけ(HEAD と比べて変わった行は 231 の 1 行だけで、ほかの列は 1 字も変わらない) |
| `docs/quality/quality-gates.md:80` と `templates/base/docs/quality/quality-gates.md:79`(「Cross-org watch gate」の行の末尾の文) | 掃引 | 「`stop`, `disband`, `escalate`, `gate`, and the inbox verbs work with no watch」を「`stop`, `disband`, `escalate`, `approve`, `gate`, `merge-order`, and the inbox verbs work with no watch」にした(両面とも同じ文)。`merge-order` は見張りの関門も承認の検査もかけない(AC9。`checkWatchGate` を呼ぶのは start・spawn・send だけ)。`approve` は 7a から抜けていた(`org_approve.go` が関門を呼ばないことは verify が確かめている) |
| `docs/plans/active/2026-10-14-org-merge-order.md`(`## Acceptance criteria` と `## Progress checklist`) | - | AC1 から AC11 の 11 個と、`Review artifact created`・`Verification artifact created`・`Test artifact created` の 3 個を `- [x]` にした(`git diff --numstat` が 14 / 14)。verify の「Spec compliance」の表は AC1 から AC9 と AC11 を「満たす」とし、AC10 は README の 1 文(D1)だけがずれていた。D1 をこの run で直したので AC10 も付けた。`PR created` は /pr の担当なので付けていない。ほかの文は 1 字も変えていない |
| `docs/insights/events/2026-10-13-org-merge-order.jsonl` | - | `sync_docs` の event を 1 行足した(`--slug org-merge-order`、`--cycle auto`、cycle 1、pass)。既存のファイルへの追記で、コミットには入れない |
| この report | - | 新規 |

## Surfaces checked for drift

| Surface | 結果 |
|---------|------|
| `/org` skill の 4 面(`.claude/skills/org/SKILL.md`、`.agents/skills/org/SKILL.md`、`templates/base/` の 2 面) | 変更なし。4 面の `shasum -a 256` は 1 種類。動詞の表の `merge-order` の行(`:183`)の分類の順(merged・closed、unreadable、no_gate、failed、no_remote、regate、wrong_base、rebase、ready)と並びの順は、コードの `classifyMergeTarget`・`classifyGatePassed`・`mergeClassOrder` と合い、README の直した文とも同じ。見張りなしで使える動詞の一覧(`:1097` から `:1103`)は `approve`・`gate`・`merge-order` を挙げる |
| `README.md:124`(動詞の表) | 変更なし。`merge-order` が動詞の列と 1 文の説明にある |
| `AGENTS.md:90`(repo map の `internal/org/` の行) | 変更なし。`mergeorder.go` を挙げる。`internal/cli/` の行は `org` でまとめており、新しいファイル名を挙げない既存の書き方と同じ |
| `docs/specs/2026-10-07-org-multi-org-director.md`(FR-8 の「7c-1 で決めたこと」の注) | 変更なし。分類の順(merged・closed、unreadable、no_gate、failed、no_remote、regate、wrong_base、rebase、ready)、`ready` の定義、7c-2 と 8a への送りが実装と合う。FR-7 と FR-8 の checkbox は 7c-2 と 8a が残るので未完のまま |
| `docs/quality/quality-gates.md` と template の写し(「Cross-org watch gate」以外の行) | 変更なし。merge-order は gate の記録を読むだけで、新しい検査や拒否を足さない。「Quality pipeline gate」の行は `ralph org gate` の拒否を書き、merge-order を述べない |
| `docs/quality/definition-of-done.md`、`docs/architecture/repo-map.md` | 変更なし。definition-of-done の org の節は仕様と rule を指すだけ。repo-map は `.harness/state/org/` の要約で、merge-order は台帳に何も書かず、動詞も script も足さない |
| `.claude/rules/ralph/`(`agent-messaging.md`、`post-implementation-pipeline.md`、`subagent-policy.md`、`model-routing.md`、`git-commit-strategy.md`)と template の写し | 変更なし。merge-order は agmsg のメッセージを送らず、受信箱も読まない(B2。verify が `mergeorder.go` を grep して確かめた)ので、TYPE の表にも承認の規則にも影響しない |
| `docs/recipes/`(`codex-seat-permissions.md` とその template の写し、`worktrees.md`) | 変更なし。reviewer と leader の雛形は触っておらず、recipe の TASK の前提は変わらない |
| `docs/tech-debt/README.md` の追加 4 行(`:233` から `:236`)と変更 1 行(`:226`) | 変更なし。挙げた名前(`gateOrgLife`、`StartFeature`、`classifyGatePassed`、`listOriginBranches`、`realMergeOrderExec`、`lookupMergePRs`、`readMergeLedger`、`readJSONLines` など)は verify が今のコードで確かめた。計画のパスは `docs/plans/active/` で、`archive-plan.sh` が `/pr` で書き直す。`run-static-verify.sh` の `tech-debt README plan references` は OK |
| `docs/tech-debt/README.md:229`(gate が feature ブランチのスクリプトを走らせる行)の Trigger「stage 7c's merge order wanting a stronger guarantee」 | 発火していない。merge-order は gate の記録だけを読み、保証を強めていない(verify の情報の項目と同じ判断)。変更なし |
| `internal/org/gate.go:27`・`:164` のコメント(「Stage 7c's merge order reads that record」) | 合う。7c-1 が読むので今も正しい。触らない |
| `ralph org --help`、`ralph org merge-order --help`(`internal/cli/org.go`、`org_merge_order.go`) | 変更なし。verify が binary で確かめた。文字列は変えていない |
| `docs/reports/`、`docs/evidence/` | 編集していない(この report を除く)。`docs/evidence/` の org-merge-order の 5 ファイルで、`grep -c '/Users/[a-z]'` はどれも 0 件(`docs/quality/definition-of-done.md` が求める `$HOME` の `~` への置き換えは済んでいる) |

## Left open

- self-review の L1 から L8 と、test の Test gaps(M05、C12、M03、C06・C07、M09。どれも 1 行の probe が用意されている)は、コードとテストと skill に触れる直しなので、この run では直していない。直すと `/self-review` から回し直しになる(cycle 2)。self-review は、直さずに進めるなら L2(PR の lookup の失敗が entry の側で区別できない)と L4(外のコマンドの実行関数の 4 つ目の写し)を 1 行にまとめて tech-debt に足すよう勧めている。cycle 2 で直すかどうかが決まっていないので、この run では tech-debt に足していない。cross-review のあとも直さずに `/pr` へ進むなら、足すか、PR の本文に known gaps として書く(orchestrator の判断)
- verify の Coverage gaps のうち、ls-remote の 30 秒の timeout と ssh の passphrase は未確認のまま(tech-debt の `:234` に記録済み)。本物の GitHub と `gh` の確認は、test の real-origin の evidence で済んでいる
- release のノート(計画の Rollout が言う、新しいサブコマンド `ralph org merge-order`(`--json`、`--all`)、`git ls-remote` でリモートを読むこと、`gh` があれば merge 済みと閉じた PR を外すこと)と PR の本文は、この run の範囲に含めていない(`/pr`)
- 計画の `PR created` は `/pr` が付ける

## Verification

コミットの前に、worktree のルートで打った。

- `./scripts/check-sync.sh`: `IDENTICAL: 167`、`DRIFTED: 0`、`ROOT_ONLY: 0`、`TEMPLATE_ONLY: 11`、`KNOWN_DIFF: 5`、`PASS: all files in sync.`(quality-gates の root と template を同じ文で変えたあと)
- `./scripts/check-skill-sync.sh`: `[ok] check-skill-sync: 13 skill(s) in lock-step`。skill は触っておらず、4 面の `shasum -a 256` は 1 種類
- `./scripts/check-template-purity.sh`: `PASS: no meta-repo-specific references found in templates.`(`templates/base/docs/quality/quality-gates.md` を変えたので打った)
- `./scripts/run-static-verify.sh`(changed の範囲): rc 0、`==> All verifiers passed.`。`check-sync.sh`、`check-skill-sync.sh`、`check-template-purity.sh`、`tech-debt README plan references`、`gofmt: ok`、`0 issues.`、`secret-scan-branch: scanned ac34ccf3..e20f2616 against origin/main: clean`。wrapper が書いたログは gitignore の対象で、コミットしていない
- `./scripts/plan-visual.sh digest docs/plans/active/2026-10-14-org-merge-order.md`: `c0b1e9230999`(checkbox の印を付けたあとも、`- Approved:` の行と同じ)
- 計画の差分: `git diff --numstat` が 14 / 14 で、変わった行はすべて `- [ ]` から `- [x]` への 1 字だけ
- tech-debt の行: HEAD と比べて変わったのは `:231` の 1 行だけ。5 列のまま、変わった列は Trigger だけ、列ごとの backtick は偶数、`file:line` の形は 0 件、U+FFFD は 0 件、計画のパスは `docs/plans/archive/2026-10-13-org-gate.md` の 1 つのまま(Python。スクリプトは scratchpad)
- `git diff --check`: 出力なし
- `./scripts/insights-append.sh --slug org-merge-order ...` のあとの `git status --short`: 既存の `docs/insights/events/2026-10-13-org-merge-order.jsonl` が未追跡のまま、ほかに新しい `??` のファイルはない
