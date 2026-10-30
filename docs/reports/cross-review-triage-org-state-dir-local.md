# Cross-review triage report: org-state-dir-local

- Date: 2026-10-30
- Plan: docs/plans/active/2026-10-30-org-state-dir-local.md
- Base branch: main (c03647f9)
- Driver: claude
- Reviewer: codex
- Triager: Claude Code (main context)
- Reviewer status: complete
- Self-review cross-ref: yes
- Cycle: 1/2
- Total reviewer findings: 1
- After triage: ACTION_REQUIRED=1, WORTH_CONSIDERING=0, DISMISSED=0

## Triage context

- Active plan: docs/plans/active/2026-10-30-org-state-dir-local.md(A2: `RALPH_ORG_STATE_DIR` を撤去し、台帳を `--state-dir` と個人用の `ralph.local.toml` の `[org].state_dir` で決める。AC4: `ralph.toml` に `[org].state_dir` があると Load がエラーにする)
- Self-review report: docs/reports/self-review-2026-10-30-org-state-dir-local.md(`Merge: yes`、この所見に当たる指摘はない)
- Verify report: docs/reports/verify-2026-10-30-org-state-dir-local.md(pass、AC4 は小文字の `[org].state_dir` で確かめていた)
- Implementation context summary: `checkLocalOnlyKeys`(`internal/config/local.go`)は `ralph.toml` を map に読み、`tomlDocHasKey` が区切りごとに大文字小文字を区別して引く。一方 `toml.Unmarshal` で struct に読むと、go-toml v2 はキーを大文字小文字を区別せずに対応させる。`LocalOrgStateDir` も struct で読むので、台帳を決める側は大文字小文字を区別しない。reviewer は codex(gpt-6-astra、xhigh、read-only)。sandbox がビルドのディレクトリを作れず、reviewer 自身はテストを打てなかった。
- Reproduction (triager): `go test -overlay` で一時のテストを足し(作業ツリーは変えていない)、`RALPH_LOCAL_CONFIG=off` で `[ORG]` / `STATE_DIR = "/ledger"` を書いた `ralph.toml` を `Load` すると、`err=<nil> state_dir="/ledger"` になった。

## ACTION_REQUIRED

| # | Reviewer finding | Triage rationale | Affected file(s) |
|---|-------------------|------------------|-------------------|
| 1 | [P2] local-only の検査が go-toml の大文字小文字の扱いと合わない。`ralph.toml` の `[ORG]` / `STATE_DIR = "/ledger"` は検査をすり抜けて `cfg.Org.StateDir` に入る。`ralph config get org.state_dir` は `/ledger` を答えるが、実際のコマンドは黙って既定の台帳を使う。decoder と同じキーの扱いにし、大文字小文字を混ぜた table と key の回帰テストを足す。 | 実在する(triager が overlay のテストで再現)。AC4 の約束(`ralph.toml` に書くと Load が断る)が大文字小文字を変えただけで破れ、表示と実際の台帳が黙って食い違う。`tomlDocHasKey` を区切りごとに `strings.EqualFold` で引けば直り、テストも安く足せる。ユーザーの指示どおり聞かずに直し、/self-review から回し直す。 | internal/config/local.go:143-155 |

## WORTH_CONSIDERING

| # | Reviewer finding | Triage rationale | Affected file(s) |
|---|-------------------|------------------|-------------------|

## DISMISSED

| # | Reviewer finding | Dismissal reason | Category |
|---|-------------------|------------------|----------|

Categories: false-positive, already-addressed, style-preference, out-of-scope, context-aware-safe
