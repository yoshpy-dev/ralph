# sync-docs report: org-merge-guarantee

- Date: 2026-10-14(JST)
- Plan: `docs/plans/active/2026-10-14-org-merge-guarantee.md`(承認済み。`./scripts/plan-visual.sh digest` は `d2923a8bf359` で、`- Approved:` の行と同じ。この run が変えたのは checkbox の印と Progress checklist の 3 行の注記だけ)
- Pipeline cycle: 1(`--cycle auto`。insight event は cycle 1)。差分の base は `a16d7c4f`、branch は `feat/org-merge-guarantee`。run 開始時の HEAD は `49cc80cb`(run 開始時の値で、この report を含む commit の SHA ではない)
- 先行 report: `docs/reports/self-review-2026-10-14-org-merge-guarantee.md`(Merge: yes、CRITICAL・HIGH 0、MEDIUM 1、LOW 6)、`docs/reports/verify-2026-10-14-org-merge-guarantee.md`(pass。文書のずれは tech-debt の 226 行目だけ)、`docs/reports/test-2026-10-14-org-merge-guarantee.md`(pass、失敗 0)
- 項目の番号: M1 と L1 は self-review の Findings の ID。verify の「Documentation drift」の表の「tech-debt の 226 行目」は M1 と同じ指摘
- 内容の commit: この report、tech-debt、`internal/org/mergeguarantee.go` のコメント、計画の checkbox を含む 1 つの commit(`docs: add cycle-1 sync-docs report for org-merge-guarantee`)。insight event の jsonl は次の 1 commit に分けた(`docs: record the cycle-1 sync-docs insight for org-merge-guarantee`)
- 中断: 1 回目の実行は利用上限で止まった。再開したとき、作業ツリーの未コミットの変更(計画、tech-debt、`mergeguarantee.go` の 3 ファイル)を `git diff --numstat` で数え直した(計画 13 / 13、tech-debt 1 / 1、`mergeguarantee.go` 2 / 2)。計画の checkbox の Edit は 13 行すべてが入っていて、やり直した手順はない

## Summary

self-review の M1(verify の drift と同じ指摘)で、tech-debt の 226 行目(feature org の leader が `ralph org gate` を通さずに PR を出せる、という行)を書き換えた。行は閉じていない。この PR の 7c-2 が入っても、gate を通していない PR を止める仕組みは ralph にない。Trigger の列から「Stage 7c-2」を外し、Impact の列に、doctor、status、merge-order が出すのは既定のブランチの保護で、どれも gate なしの PR を止めないという文を足した。

L1(`mergeOriginName` の doc コメントの文)をコメントだけ直した。コードは変えていない。

計画は、verify が Met とした AC1 から AC10 の 10 個の checkbox と、Progress checklist の Review・Verification・Test の artifact 3 行を `- [x]` にした(3 行には報告のパスを添えた)。`PR created` は付けていない。

そのほかに、この差分(doctor の Check 11c、`merge-guarantee.json`、status と merge-order の `merge guarantee:` の行、JSON の `merge_guarantee`)が触れる文書を掃いた。README、AGENTS.md の repo map、`/org` skill の 4 面、仕様 FR-7 の 7c-2 の注、tech-debt の追加行、`docs/architecture/repo-map.md`、`docs/quality/`、`.claude/rules/ralph/` は、コードと合っていたので、上の 3 件のほかは変えていない。

## Changes made

| File | 項目 | Change |
|------|------|--------|
| `docs/tech-debt/README.md`(「Nothing stops a feature org's leader from pushing and opening the PR without `ralph org gate` ...」の行、226 行目)の Trigger の列 | M1 | 「Stage 7c-2 (`ralph doctor`'s protected-branch warning, `ralph status`'s guarantee display), or a report of a feature PR merged without a gate」を「A report of a feature PR merged without a gate」にした。元の Trigger に 8a 段目の director の条件はなかったので、足していない |
| 同じ行の Impact の列 | M1 | 「A PR can carry a head that no gate passed, and a human who merges it sees no warning today」の後ろに 1 文足した: `ralph doctor` は origin の既定のブランチに保証がない(`pr_checks_only`)とき warn し、`ralph status` と `ralph org merge-order` は cache した種類を出すが、どれも保護を述べるもので、ralph の gate を通していない PR を止めない。元の文は変えていない(元の文は gate の有無の警告がないことを言い、足した文は既定のブランチの保護の警告を言うので、食い違わない) |
| `internal/org/mergeguarantee.go`(`mergeOriginName` の doc コメント) | L1 | 「mergeOriginName is host/owner/repo, or "no readable origin" for a cache whose check could not read origin (host and repo both empty).」に直した(2 行のまま)。関数の本体は変えていない。ファイルは 834 行のまま(tech-debt の行(a)の「about 830 lines」と合う) |
| `docs/plans/active/2026-10-14-org-merge-guarantee.md`(`## Acceptance criteria`) | - | AC1 から AC10 の 10 個を `- [x]` にした。verify の「Spec compliance」の表は 10 個とも Met。AC3 は「Met(計画からのずれの注を通して読む)」で、計画の Scope の「または merge queue」と実装(merge queue は ruleset に足すよう言う)の違いは Progress checklist の「計画からのずれ」にある。AC9 は「tech-debt の 226 行目のずれ」が M1 の直しで消えた |
| 同じ計画(`## Progress checklist`) | - | `Review artifact created`、`Verification artifact created`、`Test artifact created` を `- [x]` にし、報告のパスと判定を添えた。`PR created` は付けていない |
| `docs/insights/events/2026-10-14-org-merge-guarantee.jsonl` | - | `sync_docs` の event を 1 行足した(`--slug org-merge-guarantee`、`--cycle auto`、cycle 1、pass)。既存のファイルへの追記(`git status --short` で `M`。新しい未追跡のファイルはない)。別の commit に入れる |
| この report | - | 新規 |

## Surfaces checked for drift

| Surface | 結果 |
|---------|------|
| `README.md`(`ralph status` の行、`ralph doctor` の行、`ralph org merge-order` の段落) | 変更なし。verify が読んだとおり、行の位置、`merge_guarantee`、`unknown` の理由、gh を打たないこと、get-url は cache が新しいときだけ、`pr_checks_only` の勧めが merge queue を ruleset に足すと言うこと、doctor が `fail` にしないこと、が実装と合う。merge-order の段落の「the gate records are the only evidence it reads」は `ready` の判定の話で、保証の行は判定に使わないので、今も合う |
| `/org` skill の 4 面(`.claude/skills/org/SKILL.md`、`.agents/skills/org/SKILL.md`、`templates/base/` の 2 面) | 変更なし。動詞の表の `merge-order` の行と「merge してよい順」の節の保証の段落が、README と同じ条件(24 時間、origin の host と owner/repo、既定のブランチの違い、台帳がなければ書かない)を言う。`./scripts/check-skill-sync.sh` と `./scripts/check-sync.sh` が通る。`/org` skill は `ralph status` の `org watch:` の行を 1 か所(見張りが動いているかの確かめ方)で挙げるだけで、`merge guarantee:` の行は挙げる必要がない |
| `AGENTS.md`(repo map の `internal/org/` の行) | 変更なし。`mergeguarantee.go` の句が入っている。`internal/cli/` の行は subcommand の名前だけで、新しいファイル(`doctor_merge_guarantee.go`)を挙げない既存の書き方と同じ |
| `docs/specs/2026-10-07-org-multi-org-director.md`(FR-7 の「7c-2 で決めたこと」の注) | 変更なし。gh の 4 回の呼び出しの順、`--hostname` と `GH_HOST`、3 つの種類の決め方、doctor の状態、cache の書き方と読み方、tech-debt に送ったものが、verify の確認どおりコードと合う。FR-7 と FR-8 の checkbox は、rebase し直す役と 8a が残るので未完のまま。7c-1 の注(76 行目)の「7c-2(次の PR)に残した」は、7c-1 を書いた時点の記録で、7c-2 の注が 74 行目にあるので、そのままにした |
| `docs/tech-debt/README.md` の追加 6 行(240 行目から 245 行目)と 233 行目 | 変更なし。挙げた名前(`mergeGuaranteeAPIHost`、`decide`、`ReadMergeGuarantee`、`doctorMergeGuaranteeEnv` など)は関数と変数の名前で、`file:line` の参照は 226 行目にも新しい 6 行にもない。計画のパスは `docs/plans/active/` で、`archive-plan.sh` が `/pr` で書き換える。233 行目は「since stage 7c-2」の書き方に直っている |
| `docs/tech-debt/README.md` の 226 行目の `docs/plans/archive/2026-10-13-org-gate.md` | 合う。archive 済みの計画で、パスは今もある |
| `docs/quality/quality-gates.md` と template の写し | 変更なし。「Cross-org watch gate」の行の「work with no watch」の一覧は、7c-1 で `merge-order` を足した後のまま合う(doctor と status は org の動詞ではない)。保証の判定は gate や拒否ではなく、新しい検査も足していない |
| `docs/quality/definition-of-done.md`、`docs/architecture/repo-map.md`、`docs/architecture/design-principles.md` | 変更なし。repo-map は `internal/org/` の個別のファイルも doctor の検査も挙げない |
| `.claude/rules/ralph/`(`agent-messaging.md`、`model-routing.md` ほか)と template の写し | 変更なし。doctor の新しい検査と status・merge-order の行は agmsg のメッセージも受信箱も触らない |
| `docs/recipes/`(`codex-setup.md`、`codex-seat-permissions.md` ほか)、`.codex/README.md` | 変更なし。`ralph doctor` の検査を挙げる箇所は codex、herdr、agmsg、hooks の検査の説明で、検査の一覧を数えていない |
| `internal/cli/doctor.go` の「subprocess を増やすのは `--probe-models` だけ」の注釈と Check 11c のコメント、`ralph status` と `ralph org merge-order` の `--help` | 合う。注釈は git の `rev-parse` と `remote get-url origin`、`gh api` 最大 4 回、1 つの 20 秒の期限、gh が PATH になければ gh を打たないことを書く。verify の文書の表と同じ結論 |
| Go のコメントとテストが名指す `docs/plans/active/2026-10-14-org-merge-guarantee.md`(7 か所) | 変えていない。`/pr` の archive のあとに切れるが、`archive-plan.sh` が書き換えるのは tech-debt の README だけで、前からの書き方(Go のコメントが名指す `docs/plans/active/` のパス 19 種のうち 18 種はもうない)。verify の情報の項目と同じ判断 |
| 証跡の `$HOME` の置き換え | `docs/evidence/merge-guarantee-livefire-2026-10-14.md` と `docs/evidence/test-2026-10-14-org-merge-guarantee-*.txt` の 4 本、この PR の 3 つの報告で、`grep -c -E '/Users/[a-z]'` はどれも 0 件(`docs/quality/definition-of-done.md` が求める `~` への置き換えは済んでいる) |

## Left open

- self-review の L2(`readMergeOriginRepo` と `getMergeOriginURL` の読み取りが 2 つある。`ReadMergeGuarantee` の get-url の 10 秒が `MergeOrderParams` に出ない)、L3(doctor の Detail に gh の stderr の制御文字が残りうる)、L4(strict の 403 の文が原因を言い切る)、L5(status と merge-order の行が branch と repository を 2 回言う。直すと README と skill の例の行が変わる)、L6(gh の呼び出しの 3 行が 4 か所にある)は、Go のコードに触れる直しなので、この run では直していない。直すと `/self-review` から回し直しになる。self-review は、直さずに進めるなら L2 と L6 を 1 行にまとめて tech-debt の `mergeguarantee.go` の行(a)の近くに足すよう勧めている。足すかどうかは lead の判断に任せた
- verify の Coverage gaps と test の Test gaps(strict の endpoint の 404 で message が想定外の行がない、classic の strict true と merge queue と classic の検査の組み合わせは偽の seam だけ、`protection/required_status_checks` が `%2F` を受け付けるかが未確認、gh の stderr の制御文字が未確認)は、文書では直せない。どれも merge を止めるものではないと両方の報告が言う
- 7c-1 の注(仕様 76 行目)の「7c-2(次の PR)」の「次の PR」は、7c-1 を書いた時点の言い方のまま。7c-2 の注が 74 行目に入ったので、読み違えは起きにくいと判断して直していない
- release のノート(計画の Rollout が言う、doctor が gh で保護ブランチを読むこと(ネットワーク、最大 20 秒)、`merge-guarantee.json`、status と merge-order の `merge guarantee:` の行)と PR の本文は、この run の範囲に含めていない(`/pr`)。walkthrough は書いていない(`/pr` が決める)。差分は `git diff a16d7c4f...HEAD --stat` で 31 ファイル、+5,251 / -89(HEAD `49cc80cb` の時点。テストと証跡と報告を含む)
- 計画の `PR created` は `/pr` が付ける

## Verification

コミットの前に、worktree のルートで打った。

- `gofmt -l internal/`: 出力なし。`go vet ./internal/org/...`: 出力なし(rc 0)
- `./scripts/run-verify.sh`: `==> All verifiers passed.`、rc 0(出力 4,925 行、`FAIL` を含む行は 0。ブランチの secret scan `a16d7c4f..49cc80cb against origin/main` は clean)。出力は scratchpad に保存した。名前の付いた証跡 `docs/evidence/verify-2026-10-14-113230.log` は gitignore の対象で、コミットしない(手元のコピーだけ)
- `./scripts/check-skill-sync.sh`: `[ok] check-skill-sync: 13 skill(s) in lock-step`
- `./scripts/check-sync.sh`: `KNOWN_DIFF: 5`、`PASS: all files in sync.`
- `./scripts/plan-visual.sh digest docs/plans/active/2026-10-14-org-merge-guarantee.md`: `d2923a8bf359`(承認時の digest と同じ)
- tech-debt の 226 行目の形の検査(scratchpad の `tdcheck.py`。`git show HEAD:` との比較): 変わった行は 226 行目だけ、`|` で分けた断片は 7 つ(= 5 列、`HEAD` と同じ)、列ごとの backtick は偶数、`file:line` の参照は 0、「Stage 7c-2」は残っていない
- 編集前に、書き換える文字列が各ファイルに 1 回だけあることを `anchors.py` で数えた(17 個すべて 1)。編集は Edit だけで行った(`sed -i` や `gofmt -w` は使っていない)
- insight の追記のあと `git status --short` に新しい未追跡のファイルはなく、既存の jsonl が `M` になっただけ

## Insight event

`./scripts/insights-append.sh --slug org-merge-guarantee --flow standard --phase sync_docs --verdict pass --cycle auto --source skill` で、`docs/insights/events/2026-10-14-org-merge-guarantee.jsonl` に 1 行足した(cycle 1)。この jsonl だけの commit にした。
