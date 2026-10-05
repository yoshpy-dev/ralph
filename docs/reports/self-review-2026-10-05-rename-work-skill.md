# Self-review report: rename-work-skill

- Date: 2026-10-05
- Plan: docs/plans/active/2026-10-05-rename-work-skill.md
- Branch: refactor/rename-work-skill(base main 4a5d7071、HEAD 7bfe76bc)
- Reviewer: reviewer subagent (Claude)、cycle 1
- Scope: diff の品質だけ(命名、読みやすさ、不要な変更、過剰置換、root と template の不整合、コメントの正確さ、Go の `detectFlow` と追加テスト)。対象は `git diff main...HEAD`(71 ファイル、+389/-161)。plan の記録コミット(797d0551、7bfe76bc)は plan 本文を読んだが、差分の指摘対象にしていない。仕様への適合、テストの網羅、文書のずれは /verify・/test・/sync-docs の担当。テストと静的解析は実行していない

## Evidence reviewed

- 置換した語の集計: `git diff main...HEAD -U0 --word-diff=porcelain`(plan を除く)で、変わった語を一意に数えた。置換の組は `/work`→`/implement`(単独 51、句読点つき 15 ほか)、`$work`→`$implement`、`spec/plan/work`→`spec/plan/implement`、`Plan/work`→`Plan/implement`、`name: work`→`name: implement`、`.claude/skills/work`→`.claude/skills/implement`、`Work`→`Implement`(AGENTS 系 4 面の `3. Work` と README の `**Work**`)だけで、ほかの語は変わっていない
- 過剰置換の有無: 追加行にある `implement` を含む語を全部数えた。`implement`、`implementation`、`implementer`、`implements`、`implementSkillRe`、`spec/plan/implement`、パス表記しか出てこない。`implementtree` や `implementflow` のような壊れた語はない。削除行に出る `worktree`(33)、`workflow`(8)、`worktree_state_id`(8)、`ralph-worktree` は、同じ行の別の語だけが変わったもの
- skill 4 面: `.claude/skills/implement/SKILL.md`、`.agents/skills/implement/SKILL.md`、`templates/base/` 側の 2 面は、`git rev-parse HEAD:<path>` がすべて `bfd9f9f3`(同一 blob)。frontmatter は `name: implement`。どの面にも `work/` ディレクトリが残っていない(`ls` で確認)
- root と template の対: 両方にあるファイルについて、追加・削除行の集合を比べた。一致しないのは `AGENTS.md`(root にだけ meta-repo 向けの「Org runtime pointers」節があり、そこに `spec/plan/work` の 1 行がある)と `.claude/rules/ralph/model-routing.md`(org runtime 節の文面が root と template で元から違う)の 2 組。どちらも `/work`→`/implement` の置換は両側で同じ
- plan の AC2 の grep を、そのまま実行した。出力は 0 行(再現)
- AC2 の grep が見ない形を別に探した: `plan, work`、`plan and work`、`work flows`、`(work)`、`"work"`、`work/SKILL`、`work.md`、`work skill`、`work step` など。結果は F-1。`.codex/config.toml:64` の `[profiles.work]` と `internal/cli/doctor_codex_writable_root_test.go:1154,1160` の `profile "work"` は codex の profile 名、`implement/SKILL.md:5,58` の "Work from the active plan" は英語の動詞で、どれも直す対象ではない
- 履歴文書: `docs/plans/archive`、`docs/reports`、`docs/evidence`、`docs/insights/events` は差分にない。`docs/tech-debt/README.md` の差分は 150 行目 1 行だけで、16〜17 行目の RESOLVED 行は変わっていない。`docs/specs/2026-08-01-org-runtime.md` は 2026-10-05 改訂節の追加と、FR-11 行(71 行)末の印の追加だけ。印の書式は 2026-10-04 改訂の印と同じ
- Go: `internal/insights/backfill.go` の import(`regexp` は既存)、`detectFlow` と呼び出し元(`:90`)、`TestDetectFlow` の 7 ケースの期待値を手で追った。`/implementer` を含む 2 ケースは、`\b` を外して素の部分一致に戻すと落ちる。`internal/cli/init_v2_test.go` の差分のうち、キーの値の変更以外の 8 行(`:88-95`)は、長くなったキーに合わせた gofmt の整列で、意味は変わらない
- 秘密情報とローカルパス: diff 全体に `/Users/`、`$HOME`、`tmp/claude` の文字列はない

## Findings

<!-- Area recommended values: naming, readability, unnecessary-change, typo,
     null-safety, debug-code, secrets, exception-handling, security, maintainability -->

| Severity | Area | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- |
| F-1 MEDIUM | maintainability | 散文の skill 一覧とフェーズ名に旧名 `work` が 6 行残っている。skill が存在しない名前を、利用者向けの文書が挙げている。AC2 の grep が 0 行を返すので、検証でも見つからない。`.codex/README.md` は core の所有で、`ralph init` と `ralph upgrade` で下流に届く | `README.md:29` の `On-demand skills (spec, plan, work, verify, ...)`、`README.md:111` の `Spec, plan, work, and PR artifacts`、`README.md:170` の `# on-demand workflows (plan, work, verify, ...)`、`.codex/README.md:48` と `templates/base/.codex/README.md:48` の `Spec, plan, and work flows create or resume ...`、`docs/research/approach-comparison.md:87` の `Skills for plan / work / review / verify`(この 1 行は「この repo の設計判断」の節にあるので、同じ扱いにするのが自然だが、確度は他より低い)。plan の AC2 が列挙する形は `spec/plan/work`、`Plan/work`、`**Work**`、`^[0-9]+\. Work ` で、カンマや `and` で区切った形に当たらない。Scope は「散文中のフェーズ名」を含むと書いている | 6 行を `implement` に直す。`.codex/README.md` は root と template の両方を変える(`check-sync.sh` が差を見る)。AC2 の grep に下の補足の式を足し、/verify の再実行でも同じ式を使う |
| F-2 LOW | readability | `/work` を `/implement` に変えた結果、コメントの 1 行が同じブロックの他の行より 6 文字長い。ブロックの再整形が途中で止まった形 | `internal/cli/cli_test.go:349` は 83 文字。同じコメントの他の行は 77 文字以下。`internal/cli/doctor.go:222` は 79 バイトで、元のブロックの幅(72〜79)に収まっている | このコメントブロック(`:347-351`)を 77 文字前後で折り直す。次の行も 77 文字なので、1 語だけ送ると連鎖して折り直しになる |
| F-3 LOW | maintainability | `/implement\b` の `\b` は `-` や `.` の前でも境界になる。`refactor/implement-foo` のようなブランチ名や `/implement.md` も「skill の言及」と判定する。コメントは `agents/implementer.md` を除くことしか述べておらず、ハイフンの扱いを固定するテスト行もない。影響は小さい。loop 系が撤去されたので、先頭 20 行に `loop` がない報告は標準フローで、誤判定の結果がほぼ正解と同じになる | `internal/insights/backfill.go:210`(正規表現)、`:229`(判定行)、`internal/insights/backfill_test.go` の `TestDetectFlow`(`/implementer` を含む 2 ケースはあるが、`/implement-` を含むケースはない)。旧名側の `strings.Contains(line, "/work")`(`:229`)は `.claude/worktrees/...` も拾う。こちらは既存の挙動で、plan が据え置きと決めている | 受け入れるなら、コメントに「末尾の `-` や `.` は一致する」と一言足し、そのケースをテストに 1 行足して固定する。絞るなら `/implement` の直後を空白・バッククォート・`)`・行末に限る。どちらでもよい |
| F-4 LOW | readability | `detectFlow` の doc コメントに足した 2 行が、`Returns "" when not derivable — the caller omits the field.` と同じ段落に続いていて、前の文の続きに読める。`since 2026-10-05` の日付は、コメントの中で古くなる | `internal/insights/backfill.go:212-215` | 空の `//` 行で段落を分ける。旧名を残す理由は、`:229` の `"/work"` の判定の直前に 1 行で書くほうが、残した箇所の近くに理由が置かれる |
| F-5 LOW | maintainability | plan の Evidence が、セッション限りの一時領域にあるファイルを指している。`/pr` が plan をアーカイブすると、パスだけが残って中身を開けない。AC6 の結果(Deleted・Created・Advisories・block)は同じ節に文章で書かれているので、このポインタが足す情報はない | `docs/plans/active/2026-10-05-rename-work-skill.md:151`(`証拠のコピー: scratchpad の ac/upgrade-0.0.0-rename6f1f-2026-10-05.md(セッション限りの一時領域)`) | 行を消す。upgrade レポートの原文を残したいなら、`$HOME` を `~` に直した抜粋を `docs/evidence/` に置いて、そのパスを書く |

### F-1 の補足: 6 行だけに当たる grep

次の式は、AC2 の許容リストと同じ除外条件で、F-1 の 6 行だけを返す(実行して確認した)。

```sh
git grep -nP '\b[Pp]lan(,| and| /) (and )?work\b|\bwork flows\b|, work,' -- . \
  ':!docs/plans/archive' ':!docs/reports' ':!docs/evidence' ':!docs/specs' \
  ':!docs/insights/events' ':!docs/plans/active/2026-10-05-rename-work-skill.md'
```

直す先の文面の案: `(spec, plan, implement, verify, ...)`、`Spec, plan, implement, and PR artifacts`、`(plan, implement, verify, ...)`、`Spec, plan, and implement flows`、`Skills for plan / implement / review / verify`。

## Positive notes

- 置換は機械的に一貫している。語の集計で、組になった置換以外の変更がない。`worktree`、`workflow`、`/workspace` のような近い綴りの語は 1 件も壊れていない
- 履歴の扱いが plan どおり。archive、reports、evidence、events、tech-debt の RESOLVED 行は触っておらず、org-runtime spec は本文を書き換えずに日付つきの改訂節と印で足している。2026-10-04 改訂と同じ流儀
- `docs/recipes/codex-setup.md` の `$spec`/`$plan`/`$implement` の行末コメントを揃え直したのは、`$implement` が長くなった分の整列で、必要な変更
- `detectFlow` は旧名を消さずに残した(既存の報告が flow なしになるのを避ける)。正規表現をパッケージ変数にして、呼び出しごとの再コンパイルを避けている(同じファイルの他の正規表現は関数内で毎回コンパイルしている)。`TestDetectFlow` は backtick つき、行末、旧名、`implementer` のパス、`/implementer`、loop、マーカーなしを押さえ、境界の判定が効いていることを `/implementer` の 2 ケースが見張っている
- root と template の対、skill 4 面の blob が一致していて、片側だけの編集がない

## Tech debt identified

| Debt item | Impact | Why deferred | Trigger to pay down | Related plan/report |
| --- | --- | --- | --- | --- |
| なし | - | - | - | - |

_(行を足していないので `docs/tech-debt/` への追記はしていない。F-1 を直さずに先へ進める場合は、F-1 を 1 行の tech-debt として登録する)_

## Recommendation

- Merge: 可(CRITICAL・HIGH なし)。F-1 は利用者向けの文書が存在しない skill 名を挙げる問題で、下流にも届くので、この branch で直してから /pr に進める。修正は 5 ファイル 6 行
- Follow-ups:
  - F-1 を小さなコミットで直し、AC2 の grep に補足の式を足す。直したあとの /verify は、足した式で再確認する
  - F-2〜F-5 は任意。F-3 だけは、受け入れる場合もコメントかテストで挙動を固定しておくと、次に正規表現を触る人が迷わない
- Known gaps:
  - ビルド、テスト、静的解析は実行していない。`detectFlow` とテストの期待値は目で追っただけ
  - skill の本文 4 面は、blob の一致で同一性を確かめた。本文の意味の確認(手順の番号や参照が変わっていないか)は diff の置換語の範囲で、全文の再読はしていない
  - `/implement` が Claude Code・Codex の組み込みコマンドと衝突しないことは、仕様側の確認事項で、ここでは見ていない
  - insight event(`scripts/insights-append.sh --phase self_review`)は追記していない。呼び出し側の指示で、このコミットは報告だけにしている
