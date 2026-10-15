# Cross-review triage report: org-director-start

- Date: 2026-10-15
- Plan: docs/plans/active/2026-10-15-org-director-start.md
- Base branch: main(merge-base f21df66c)
- Driver: claude
- Reviewer: codex
- Triager: Claude Code (main context)
- Reviewer status: complete
- Self-review cross-ref: yes
- Cycle: 1/2
- Total reviewer findings: 1
- After triage: ACTION_REQUIRED=1, WORTH_CONSIDERING=0, DISMISSED=0

## Triage context

- Active plan: docs/plans/active/2026-10-15-org-director-start.md(承認済み、digest 671c60416856)
- Self-review report: docs/reports/self-review-2026-10-15-org-director-start.md(cycle 1 は b50d261b で Merge: yes。CRITICAL・HIGH なし、MEDIUM 1、LOW 6。da818b4f と dbd67a58 で直した)
- Verify report: docs/reports/verify-2026-10-15-org-director-start.md(cycle 1 は ec70c739 で pass。隔離した herdr server の実機で、codex と claude の start と stop を確かめた)
- Test report: docs/reports/test-2026-10-15-org-director-start.md(cycle 1 は 8461043a で pass。mutant 86 個のうち 81 個が red。生き残りから出たテストの穴 D4・E1・E12/E13・P2 は、sync-docs の e80ccafe で tech-debt に 1 行として記録した)
- Implementation context summary:
  - `codex exec review --base main` を e683ee47 で実行した。モデルは gpt-6-astra、effort は xhigh、sandbox は read-only。結果は `codex rc=0` で、`-o` のファイルは 1,002 バイト。
  - Codex は read-only の sandbox で Go のビルド用ディレクトリを作れず、Go のテストを回せていない。テストは cycle 1 の /test が回している。
  - 指摘の主張は、コードと照らして確かめた。`startDirector`(`internal/cli/org_director_headless.go`)は `configPath` を見張りの起動(`wireOrgStartWatchGate`)にだけ渡しており、`StartDirector` には渡していない。そのため `RenderDirectorPrompt` の雛形(`internal/org/prompts/director.md`)のコマンドは `--state-dir {{STATE_DIR}}` だけを持つ。
  - `ralph org director status` は `--config` がなく、台帳が main worktree のものなら、main worktree の `ralph.toml` を読む(`withMainWorktreeDirectorEnabled`)。人が `--config` で別の設定を渡して start した場合、director が最初に打つ status と、その後の `ralph org` の動詞は、start とは別の設定を読む。

## ACTION_REQUIRED

| # | Reviewer finding | Triage rationale | Affected file(s) |
|---|-------------------|------------------|-------------------|
| 1 | [P2] `director start` に `--config` を渡しても、その設定は director の起動の雛形に入らない。director が最初に必ず打つ `ralph org director status --state-dir ...` は既定の `ralph.toml` を読む。たとえば、`--config` の設定は正しく、root の `ralph.toml` が壊れていると、start は成功を報告するが、director は自分の登録を確かめられない。`--config` を絶対パスにして雛形に入れ、設定を読む `ralph org` の動詞に付けるべき | 実際の問題で、直す価値もある。`--config` で `[org.director].enabled` や `[org.permissions]` を変えて start した人に対し、director が `enabled = false` の別の設定を読むと、status は disabled と答える。雛形の手順では、そのとき director は何もしない。壊れた `ralph.toml` なら、status はエラーで終わる。どちらも、start が成功を報告したあとで director が動けない。直し方は 2 つ。`DirectorPromptVars` に設定のパス(絶対パス、なければ空)を足し、雛形の `ralph org` のコマンドに `--config <path>` を付けて描く。`/director` skill の headless の節にも同じ規則を書く | `internal/cli/org_director_headless.go`(`startDirector`)、`internal/org/director_start.go`(`DirectorStartParams`、`RenderDirectorPrompt` の呼び出し)、`internal/org/prompts.go`(`DirectorPromptVars`)、`internal/org/prompts/director.md`、`/director` skill の 4 面 |

## WORTH_CONSIDERING

なし。

## DISMISSED

なし。

Categories: false-positive, already-addressed, style-preference, out-of-scope, context-aware-safe
