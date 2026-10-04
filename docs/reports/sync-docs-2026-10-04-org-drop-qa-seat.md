# sync-docs report: org-drop-qa-seat

## Cycle 1

- Date: 2026-10-04
- Plan: `docs/plans/active/2026-10-04-org-drop-qa-seat.md`
- Pipeline cycle: 1 of 2。差分は `origin/main` の `4ee080f5` から branch HEAD `aa1ac717`(refactor/org-drop-qa-seat)まで
- 先行 report(それぞれを追加した commit):
  `docs/reports/self-review-2026-10-04-org-drop-qa-seat.md`(`00ecdbfc`、addendum `3c174e98`。merge 可)、
  `docs/reports/verify-2026-10-04-org-drop-qa-seat.md`(`6d9c1a12`。pass)、
  `docs/reports/test-2026-10-04-org-drop-qa-seat.md`(`aa1ac717`。pass)

## Summary

Slice C で更新した文書、`internal/cli/org.go` の help、`reviewer.md` / `leader.md` の雛形、`internal/org/prompts.go` の撤去・改名した役割の表は、互いに一致している。旧名の `lead` と `qa` が非履歴の文書に残っている箇所は、`/org` skill の 4 面の「撤去・改名した役割」の案内だけだった。

直したのは次の 4 点。

1. `docs/recipes/codex-seat-permissions.md` の TASK の例。recipe は `--role reviewer` を権限モードの検査の乗り物に使っている。reviewer 雛形は今回「最初にゲートを再実行し、落ちるか実行できなければ BLOCKED で返す」に変わったので、検査用の TASK にゲートの指示がないと、座席が検査の手順より先にゲートへ向かうおそれがある。
2. spec の 2026-10-04 改訂の節の 2 文。1 つは「Summary の『QA』を含む履歴の行に印を付ける」と読めるのに、Summary には印がない点。もう 1 つは `lead` の読み替えが FR-4 の例だけに限られている点。
3. `docs/tech-debt/README.md` の FR-7 の行が、`/pr` で archive に移る plan を `docs/plans/active/` で参照している点。
4. plan の Status 行。

## Changes made

| File | Change |
|------|--------|
| `docs/recipes/codex-seat-permissions.md`、`templates/base/docs/recipes/codex-seat-permissions.md` | (1) 「Write the TASK」の TASK の例の先頭に「This is a sandbox probe, not a review: do not run the gate scripts and do not review a diff.」の段落を足した。(2) コードブロックの直後に、`--role reviewer` で spawn する理由(`ralph-edits.toml` の `[org.permissions.roles] reviewer` を効かせるため)と、reviewer 雛形がゲートの再実行から始まるので先頭の段落を残すこと、を書いた。2 面は `cmp` で byte 一致 |
| `docs/specs/2026-08-01-org-runtime.md` | 2026-10-04 改訂の (c) の末尾に「FR-5 以降の本文にある Lead(指示役を指す語)も同じ座席のことで、識別子や宛先としては `leader` である」を足した。最後の bullet を「本文の履歴の記述は書き換えない。FR-4、FR-7、FR-11、AC、Open questions の該当行には改訂の印を付けた。Summary の『QA』と 2026-09-16 改訂 (b) の『4 種』には印を付けない。上の (a) が置き換えを述べている」に直した。履歴の本文と印の付いた行は触っていない |
| `docs/tech-debt/README.md` | FR-7 の hook 強制が未実装、の行の関連の列を `docs/plans/active/2026-10-04-org-drop-qa-seat.md` から `docs/plans/archive/2026-10-04-org-drop-qa-seat.md (archived by /pr)` に変えた。前の plan(`subagent-model-defaults`)の sync-docs と同じく、`/pr` の archive 後の場所を前方参照にした。この行だけを直した |
| `docs/plans/active/2026-10-04-org-drop-qa-seat.md` | Status を `In progress` から `Pipeline at sync-docs → cross-review (self-review, verify, test: pass; PR not created)` に変えた。ほかの節とチェックボックスは触っていない |
| `docs/reports/sync-docs-2026-10-04-org-drop-qa-seat.md` | この report |

recipe に足した段落(英語、recipe の本文に合わせた):

> The seat is spawned with `--role reviewer` so that the `[org.permissions.roles] reviewer` override in `ralph-edits.toml` applies to it. The reviewer role template starts a task by re-running the deterministic gate (`run-static-verify.sh` / `run-test.sh`) and returns BLOCKED, without doing the work, when the gate fails or cannot run. The first paragraph of the TASK keeps that out of this probe, so leave it in.

この recipe の変更は、生きた codex 座席では確かめていない。reviewer 雛形の記述(`internal/org/prompts/reviewer.md` のミッション節の項目 1〜3)から導いた。次に recipe を実行したときに、TASK の先頭の段落で座席がゲートを飛ばして検査の手順に進むかを見る。

## Surfaces checked for drift

| Surface | Finding |
|---------|---------|
| `git grep -n -w -i -E 'lead\|qa'`(`*.go`、履歴の dir、`docs/specs/2026-08-01-org-runtime.md`、`docs/tech-debt/README.md` を除く) | `/org` skill の 4 面(`:198-203`)の旧名の案内と、`internal/insights/testdata/receipts.jsonl`(過去のデータのフィクスチャ)だけ。どちらも plan が残すと決めたもの |
| 単語境界に掛からない形(`_lead`、`lead_`、`lead.md`、`lead-driver`、`lead[A-Z]`) | `/org` skill の `--lead-driver`(非推奨の別名の案内)と、`docs/tech-debt/README.md` の履歴の行だけ |
| 「4 役割」「4 種」「4 フェーズ」「impl → qa」の言い回し | 非履歴の文書にはない。残りは spec の履歴の記述と、FR-7 / FR-11 の印の付いた行 |
| `README.md`(org の節、Commands の表 `:124`、Features) | `leader` / 3 役割に更新済み。Commands の表は `--leader-driver` などのフラグを書いていないので変更の対象外 |
| `AGENTS.md`、`.ralph/core/AGENTS.core.md`、`templates/base/AGENTS.md`、`templates/base/.ralph/core/AGENTS.core.md`、`CLAUDE.md` | 4 つの AGENTS 系は `leader`。Repo map の `internal/org/`(「role prompt templates」)は役割の数を書いていない。`CLAUDE.md` に旧名はない |
| `.claude/rules/ralph/agent-messaging.md`(+ template) | `TO: leader`、`ensureLeaderJoined`、seat id の例が `implementer` / `reviewer`。`ensureLeaderJoined` は `internal/org/spawn.go:54,63,100` のコメントにある名前と一致 |
| `.claude/rules/ralph/model-routing.md`、`subagent-policy.md`、`post-implementation-pipeline.md`、`ralph-workflow.md` | org の役割名を挙げていない。ずれなし |
| `docs/quality/quality-gates.md`(+ template) | Quality pipeline gate の行が新しい順序で、`GATE: fail` / `GATE: unrunnable` を書いている。root は `(see docs/tech-debt)` あり、template は元から参照先がないので外してある(plan の逸脱に記録済み)。`check-sync.sh` の KNOWN_DIFF の範囲 |
| `docs/quality/definition-of-done.md`(+ template)、`docs/architecture/repo-map.md`、`docs/insights/README.md`(+ template) | org の節は spec と `agent-messaging.md` を指すだけで、役割名を書いていない |
| `docs/recipes/` | `codex-seat-permissions.md` は `leader` に更新済み。ほかに org を扱うのは `worktrees.md:55`(manifest の worktree path の説明)で、役割名なし。上記の TASK の変更が 1 点 |
| `templates/base/ralph.toml` | `Leader autonomy` と reviewer の `guarded` の注記を確認。`[org.roles]` の例は `implementer` / `reviewer`。ルートに `ralph.toml` はない |
| `.claude/skills/org/SKILL.md`(+ 3 面) | 動詞リファレンスの `--leader-driver`、`start` の `leader.md`、役割の節(3 役割と撤去・改名の案内)、Leaded 行、fan-out の例が、`reviewer.md` / `leader.md` / `prompts.go` の表と一致。`--role qa` の拒否は「`--prompt` がなければ」、`--role lead` / `--id lead` / 旧キーは「spawn で拒否」で、`retiredRoleRemoved` / `retiredRoleRenamed` の説明と合う |
| `ralph org` の help(`internal/cli/org.go`) | `spawn` の `--leader-driver` と非推奨の `--lead-driver`、`start` の Short / Long / hint、`wait` / `watch` の Long が `leader`。`--role` のフラグ説明は役割を列挙しない。ずれなし |
| `ralph doctor` の新しい Check(`Org retired role keys`、`internal/cli/doctor.go`) | 文書で Check を 1 つずつ挙げている面はない(README `:127` は種別の説明、`/org` skill は「Org codex model slugs」と「Shell aliases」と「Codex sandbox」だけ)。足す場所がない。旧キーの warn は `/org` skill と spec 改訂 (c) に書いてある |
| `scripts/`、`tests/`、`.claude/hooks/`、`.codex/`、`packs/`、`.github/` | `lead` / `qa` の role 参照なし |
| `docs/specs/2026-08-01-org-runtime.md` | 印の付いた行は FR-4(`:59`)、FR-7(`:62`)、FR-11(`:66`)、AC(`:83`)、Open questions(`:177`)。Summary(`:5`)と 2026-09-16 改訂 (b)(`:10`)は印なしで、改訂の節がそれを書くようにした |
| AC-5 / AC-12 / AC-14 の grep の再実行 | 編集の後も、AC-5 の `qa` の 12 ファイル(skill 4 面、insight event、spec、tech-debt、Go 5 ファイル)と、AC-12 の例外 1 行(`watch.go:230` の JSON タグ)は plan の許可リストのまま。AC-14 の `lead` の 17 ファイルにも新しいファイルはない |

## Found but left

- `/org` skill(4 面)と `templates/base/ralph.toml` の注記「claude の座席を `guarded` にすると許可待ちで止まる」: 理由づけが少し粗い。`.claude/settings.json`(root と template は同じ内容)の allow に `Bash(./scripts/*)` があるので、このプロジェクトの設定を読む座席では、ゲートのスクリプト自体は許可待ちにならない可能性がある。`guarded` の reviewer は `docs/reports/` へレポートを書く Write でも許可待ちになるので、「止まる」という結論は変わらないと見ている。ただし生きた座席で確かめていない(AC-9 の smoke は `--allowedTools` を絞った `claude -p` で、`guarded` の座席ではない)。確かめずに 5 面の文言を変えると、別の未確認の主張に置き換わるので、そのままにした。plan の R4 と Open questions が同じ未確認を持っている。
- `ralph org` 親コマンドの Short(`internal/cli/org.go:32`)は「spawn, send, wait, read, stop, status, disband」のままで、`start`、`report`、`watch` がない。この変更の前からある。README の Commands の表は 10 個すべてを挙げている。Go のコードは対象外なので触っていない。
- `docs/tech-debt/README.md` には `docs/plans/active/` を指す行が他にも多く(verify の report は 31 か所と数えている)、archive 済みの plan を指すものがある。この task が触る行ではないので、掃除はしていない。
- spec の Summary(`:5`)と 2026-09-16 改訂 (b)(`:10`)の「QA」「4 種」には印を付けていない。plan が「履歴の記述(Summary など)は書き換えない」とし、改訂の (a) が置き換えを述べているため。印を付ける案もあるが、改訂の節の書き方を直すほうが履歴への手入れが少ない。
- plan の Progress notes / 逸脱の節に sync-docs の行は足していない。この commit の SHA が要るので orchestrator が足す。
- insight event(`./scripts/insights-append.sh`)は追記していない。指示に含まれず、`docs/insights/events/2026-10-04-org-drop-qa-seat.jsonl` には self_review / verify / test の 3 行だけがある。
- recipe の TASK の変更は生きた codex 座席で確かめていない(上記)。`docs/evidence/codex-seat-permissions-2026-09-18.md` は変更前の TASK での実行記録で、履歴なので触っていない。

## Checks run

| Command | Result |
|---------|--------|
| `./scripts/check-sync.sh` | exit 0。IDENTICAL 159、DRIFTED 0、ROOT_ONLY 0、TEMPLATE_ONLY 11、KNOWN_DIFF 5(編集の前後で同じ)。`PASS: all files in sync.` |
| `./scripts/check-skill-sync.sh` | exit 0。13 skill が一致 |
| `cmp docs/recipes/codex-seat-permissions.md templates/base/docs/recipes/codex-seat-permissions.md` | exit 0。byte 一致 |
| `./scripts/check-template-purity.sh` | exit 0。template に meta-repo 固有の参照なし |
| `./scripts/check-template.sh` | exit 0。`Template structure looks good.` |
| `go test -count=1 ./internal/scaffold/ ./internal/config/` | ok(埋め込み template の検査) |
| `./scripts/secret-scan.sh --file`(この report)と `git diff \| ./scripts/secret-scan.sh --diff`(作業ツリーの編集) | どちらも exit 0。`--staged` は commit する orchestrator が実行する |
| `./scripts/secret-scan-branch.sh --strict` | 同上。push 前に orchestrator が実行する |

## Diff size (for /pr)

`git diff 4ee080f5...HEAD`(この pass の編集を含まない HEAD `aa1ac717`)は 54 files changed、3283 insertions、958 deletions。テストを除く Go は 542 行追加、250 行削除。テストを含む `docs/` 以外は 2488 行追加、947 行削除。walkthrough を書くかどうかは `/pr` が決める。この step では書いていない。

## Files changed in this pass

- `docs/recipes/codex-seat-permissions.md`、`templates/base/docs/recipes/codex-seat-permissions.md`
- `docs/specs/2026-08-01-org-runtime.md`
- `docs/tech-debt/README.md`
- `docs/plans/active/2026-10-04-org-drop-qa-seat.md`
- `docs/reports/sync-docs-2026-10-04-org-drop-qa-seat.md`(この report)
