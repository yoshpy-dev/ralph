# sync-docs report: trial-docs-wrapup

- Date: 2026-10-23
- Plan: `docs/plans/active/2026-10-23-trial-docs-wrapup.md`(承認の digest `932bafc59d5c` はこの run の前後で同じ。`./scripts/plan-visual.sh digest` で確かめた。この run は計画を触れていない)
- Pipeline cycle: 1(`.harness/state/standard-pipeline/cycle-count.json` の値。上限は 2)。base は `3c1cd14b`、branch は `docs/trial-docs-wrapup`。run 開始時の HEAD は `c8f17b07`(この report を含む commit の SHA ではない)
- 先行 report: `docs/reports/self-review-2026-10-23-trial-docs-wrapup.md`(Merge yes。cycle 1 の LOW 8 件と INFO 3 件、再レビューの N1〜N3 は直った。最後の再レビューは新しい LOW 以上がなく、INFO の N4 だけが残る)、`docs/reports/verify-2026-10-23-trial-docs-wrapup.md`(pass。INFO 4 件は tester の前に直した)、`docs/reports/test-2026-10-23-trial-docs-wrapup.md`(pass。穴 1 件は skill の 3 か所の「数十分」を固定するテストがないこと)
- 差分の大きさ(`git diff 3c1cd14b...HEAD --shortstat`、この run の前): 21 ファイル、+1802/-87。walkthrough を書くかどうかは `/pr` が決める

## Summary

`3c1cd14b...c8f17b07` の差分は、分離した headless の herdr server で org runtime を動かす手順を recipe にし(`docs/recipes/isolated-herdr-server.md` と、`templates/base/` の同じ中身の写し)、`/org` skill の前提に folder trust の項目と recipe への指し先を足し、`/director` skill に trust と bypass の画面へキーを送らない禁止を足し、`/org` skill の RESULT の例を reviewer の雛形に合わせてテストで固定し、gate と regate の「数分」を「repo のテストの量とマシンの負荷によっては数十分かかる」にし、仕様・根拠の表・tech-debt で試行の所見の系列を閉じる。

この run が変えたのは 2 か所で、足したものは insight event と、この report である。`docs/architecture/repo-map.md` の recipe の列挙に新しい recipe を足し、`docs/tech-debt/README.md` に self-review の N4(template の recipe が `docs/evidence/` を出典にする)の行を足した。そのほかの面は突き合わせて、変える必要がなかった。

## Changes made

| File | Change |
|------|--------|
| `docs/architecture/repo-map.md:37` | `docs/recipes/` の行の列挙の最後に「running the org runtime in an isolated headless herdr server」を足した。この行は recipe を 1 つずつ挙げていて(Codex setup、Codex seat permissions、agent teams、language packs、worktrees、`launch_cmd`、`wake_cmd`)、新しい recipe だけが抜けていた。このファイルは `check-sync.sh` の `ROOT_ONLY_EXCLUSIONS` にあり、template に写しはない |
| `docs/tech-debt/README.md:360`(表の最後の行) | 新しい行。`templates/base/docs/recipes/` の 5 つの recipe(`isolated-herdr-server.md` が 5 件、`codex-seat-permissions.md` が 2 件、`codex-setup.md`・`director-wake.md`・`director-launch-cmd.md` が 1 件ずつ、合わせて 10 件のファイル)が、scaffold に届かない `docs/evidence/` のファイルを出典にする。recipe の手順はそのファイルを読まないので、動かなくなる手順はない。直し方は、2 つの写しで出典を「ralph repository の記録」と書き直すか、`templates/base/docs/evidence/` に出典を配るか。recipe は変えていない(依頼どおり) |
| `docs/insights/events/2026-10-23-trial-docs-wrapup.jsonl` | `sync_docs` の event を 1 行足した(`--cycle auto` で cycle 1、`./scripts/insights-append.sh` の rc 0) |
| `docs/reports/sync-docs-2026-10-23-trial-docs-wrapup.md` | 新規(この report) |

依頼は `codex-seat-permissions.md`・`director-wake.md`・`director-launch-cmd.md` の 3 つを既存の同じ形と書いた。`templates/base/docs/recipes/*.md` の `docs/evidence/` を `grep` すると `codex-setup.md:194` も `docs/evidence/codex-config-rewrite-2026-09-30.md` を挙げていたので、行に足した。出典 10 件はどれも root に存在する(`[ -f ]` で確かめた)。

足した文にホームのパス、token はない。`git diff --check` は rc 0、差分の U+FFFD は 0 件。

## Surfaces checked for drift

| Surface | 結果 | 根拠 |
|---------|------|------|
| `AGENTS.md:112`(`docs/recipes/` の行: Codex setup, language packs, worktrees) | 変更なし | 例を 3 つ挙げる行で、`agent-teams`・`codex-seat-permissions`・`director-wake`・`director-launch-cmd` もすでに挙げていない。計画の Non-goals が「この行を書き直さない」と書き、AGENTS.md は地図で短く保つ決まり。recipe を 1 つずつ挙げる場所は `docs/architecture/repo-map.md` で、そちらを直した |
| `templates/base/AGENTS.md`、`.ralph/core/AGENTS.core.md`、`templates/base/.ralph/core/AGENTS.core.md` | 変更なし | 3 つとも「recipes」の語が 0 行(root の AGENTS.md は 1 行)。managed block に recipe の行はなく、`check-sync.sh` は PASS |
| `docs/recipes/README.md`、`index.md` など recipe の索引 | なし | `docs/recipes/` には recipe の 8 ファイルだけがある |
| `README.md` | 変更なし | recipe を挙げるのは director の節の 2 か所(258 行の `director-launch-cmd.md`、260 行の `director-wake.md`)で、どちらも director の文脈の指し先。recipe を 1 つずつ並べる場所はない。64 行の `codex trust .` は `/org` skill の新しい前提の文と合う |
| 「数分」「a few minutes」など所要時間の文 | 残りなし | `grep -rn -e '数分' -e 'a few minutes' -e 'few minutes' -e 'several minutes'` を、evidence・reports・plans・insights と CHANGELOG を除いた repo 全体に打って 0 件。この repo に CHANGELOG はない(release ノートは goreleaser が git log から作る) |
| `ralph org gate --help` の「They can take minutes」(`internal/cli/org_gate.go:55` 付近、`org_gate_test.go` が固定) | 変更なし | 「数分」と違い、長さを数分に限らない。試行の 1375〜1990 秒(23〜33 分)と食い違わない。直すなら help の文と固定するテストを変えるコードの変更になり、この PR は動きを変えない |
| director が trust や bypass の画面に答えてよいと読める文 | なし | `/director` skill の「してはいけないこと」(962 行)が禁じる。`/org` skill の bypass の項目は「director もこの画面にキーを送らない」と書く。`/director` skill 740 行の `herdr pane send-keys <pane> Enter` は、入力欄に残った DECISION の文に Enter を 1 回送る話で、「承認の画面など別のものを出していれば、Enter を送らない」と書く。`internal/cli/org.go` の send の警告は「dialog が出ていれば Enter を押さない」と書き、`docs/recipes/codex-seat-permissions.md` の send の段落は、入力欄に残ったときだけ Enter を押し、何も見ずに打つキーは承認の画面を確定しうると書く。どれも新しい禁止と食い違わない |
| `docs/recipes/codex-seat-permissions.md`、`codex-setup.md` の dialog の文 | 変更なし | 前者 179 行は起動直後の pane に dialog がないことの確かめ、後者 191 行は `.codex/config.toml` の書き換えの原因の候補。trust の画面に誰が答えるかは書いていない |
| `docs/specs/2026-08-01-org-runtime.md` | 変更なし | RESULT の例、trust、bypass、gate の所要時間のどれも書かない。reviewer の `GATE: pass` の RESULT(FR-7)は、新しい例の `GATE: pass` と合う |
| `.claude/rules/ralph/agent-messaging.md` | 変更なし | TYPE の表、TASK_ID の必須、メッセージの形、EVIDENCE はポインタ、2,000 文字の上限を書く。新しい例は TYPE `RESULT` と TASK_ID を持ち、EVIDENCE は `docs/reports/` のパスで、どれにも反しない。ヘッダ `GATE`・`HEAD`・`BASE` は「他の KEY: value の行」に入る |
| `docs/specs/2026-10-07-org-multi-org-director.md`、`docs/evidence/org-spec-evidence-2026-10-21.md`、tech-debt の S3 の書き換え | 変更なし | AC5 の `fix/busy-leader` を `docs/specs`・`docs/evidence`・`docs/tech-debt`・`docs/architecture`・`README.md`・`AGENTS.md`・`.claude/rules`・`internal` に打って 0 件。ほかは verify が AC5 で確かめた |
| tech-debt のファイルの大きさの行 | 変更なし | この PR で Go のファイルの行数が変わったのは `report_names_sync_test.go`(300 行、800 未満)だけ。行に載る `prompts_test.go` は 2165 行のまま(1 行の置き換え)で、`verbs.go`・`verbs_test.go`・`spawn.go`・`spawn_test.go` は触れていない |
| tech-debt の F-8 の行(skill の `stop`・`disband` の cell の長さ) | 変更なし | その行が挙げるのは `stop` と `disband` の cell で、この PR が直した `gate` の cell ではない。`gate` や `regate` の cell の長さを挙げる行は tech-debt にない |
| `/org`・`/director` skill の 4 写しずつ | 同期 | `./scripts/check-skill-sync.sh` が 14 skill で lock-step。templates の写しも `check-sync.sh` で IDENTICAL |
| `docs/quality/definition-of-done.md`、`docs/quality/quality-gates.md` と template の写し | 変更なし | recipe、trust、RESULT の例、gate の所要時間のどれも書かない |

## Decisions on the reports' remaining items

| 項目 | 決めたこと |
|------|------------|
| self-review の N4(INFO。template の recipe の出典が scaffold に届かない) | tech-debt に 1 行足した。recipe は変えない。同じ形が 4 つの既存の recipe にあり、直すには 5 つの recipe の 2 つの写しを書き直す必要があるため |
| test の穴(skill の 3 か所の「数十分」を固定するテストがない) | 文書には書かない。計画の Progress が「AC4 は grep で見る」と書き、leader の雛形の文は `prompts_test.go` が固定している。skill の 3 か所の文は、次に `gate` や `regate` の所要時間の文を直すときに、`rg` で残りを確かめれば足りる |

## Plan drift notes

- 計画の Progress は「PR created」を除いて `[x]`。この run が計画を触れていないので digest は承認のまま
- 計画の Affected areas が挙げない `docs/architecture/repo-map.md` を、この run が直した。Non-goals には入らない(Non-goals の AGENTS.md の行は触れていない)。tech-debt の新しい行(N4)は、Affected areas が挙げる `docs/tech-debt/README.md` への足しで、計画の範囲の中

## Left open

- 開いたまま残る行は、仕様と根拠の表が挙げる 3 つ(leader への ALERT、Codex の leader、watcher の時間切れ)に、この run が足した N4 の 1 つを足した 4 つ
- recipe は通しで打ち直していない(計画の Non-goals と「確かめていないこと」。recipe の冒頭も書く)

## Verification

| Check | Result |
|-------|--------|
| `./scripts/check-skill-sync.sh` | rc 0。`[ok] check-skill-sync: 14 skill(s) in lock-step` |
| `./scripts/check-sync.sh` | rc 0。IDENTICAL 172、DRIFTED 0、ROOT_ONLY 0、TEMPLATE_ONLY 11、KNOWN_DIFF 5。`PASS: all files in sync.` |
| `./scripts/plan-visual.sh digest docs/plans/active/2026-10-23-trial-docs-wrapup.md` | `932bafc59d5c`(承認の digest と同じ) |
| `git diff --check`、差分のホームのパスを含む追加行の数、差分の U+FFFD の数 | rc 0、0 件、0 件 |
| `cmp` の結果(`docs/recipes/` と `templates/base/docs/recipes/`) | `diff -rq` が出すのは `adding-a-language-pack.md` の違い(KNOWN_DIFF)と `.gitkeep` だけ |
| テスト | 打っていない(`/test` の範囲。この run が変えたのは文書と insight event だけで、Go とスクリプトには触れていない) |

## Insight event

追記した。`./scripts/insights-append.sh --slug trial-docs-wrapup --flow standard --phase sync_docs --verdict pass --cycle auto --source skill` が rc 0 で、`docs/insights/events/2026-10-23-trial-docs-wrapup.jsonl` の末尾の行(cycle 1)になった。
