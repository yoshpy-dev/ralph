# Verify report: allow-go-and-repo-commands

- Date: 2026-04-17
- Plan: `docs/plans/active/2026-04-17-allow-go-and-repo-commands.md`
- Verifier: verifier subagent
- Scope: AC1–AC12 spec compliance + static analysis + doc drift for commit `7295c69` on branch `chore/allow-go-and-repo-commands`
- Evidence: `docs/evidence/verify-2026-04-17-allow-go-and-repo-commands.log`, `docs/evidence/verify-2026-04-17-051819.log`

## Spec compliance

| Acceptance criterion | Status | Evidence |
| --- | --- | --- |
| AC1 — All 15 Go toolchain prefixes present in `.claude/settings.json/permissions.allow` | Verified | `jq any(. == $e)` returned `true` for all 15 entries (`go build:*`, `go test:*`, `go vet:*`, `go run:*`, `go mod:*`, `go get:*`, `go install:*`, `go tool:*`, `go generate:*`, `go list:*`, `go env:*`, `go fmt:*`, `go version`, `go clean:*`, `go doc:*`). See `.claude/settings.json:43–57`. |
| AC2 — `gofmt:*`, `golangci-lint:*`, `staticcheck:*`, `goimports:*`, `shellcheck:*` present | Verified | All 5 → `true`. See `.claude/settings.json:58–62`. |
| AC3 — `./ralph:*`, `./bin/ralph:*`, `bin/ralph:*` present | Verified | All 3 → `true`. See `.claude/settings.json:63–65`. |
| AC4 — `./tests/*`, `bash -n:*` present | Verified | Both → `true`. See `.claude/settings.json:41,66`. |
| AC5 — `sh:*`, `bash:*`, `xargs:*` absent (Codex [HIGH]) | Verified | `jq` select returned `[]`. |
| AC6 — No existing `allow` entries removed (pure addition) | Verified | main has 35 entries, HEAD has 60, all 35 present on HEAD (`comm -23` empty). Diff shows only additions; the single `-` line is the old trailing-comma adjustment on `"Bash(claude:*)"` which preserves the entry. No duplicates (`unique | length == length` → `true`). |
| AC7 — `hooks` section unchanged | Verified | `diff <(git show main:.claude/settings.json \| jq -S .hooks) <(jq -S .hooks .claude/settings.json)` produced no output. |
| AC8 — `jq -e . .claude/settings.json` exits 0 | Verified | Exit 0 on both `.claude/settings.json` and `templates/base/.claude/settings.json`. |
| AC9 — Runtime canary (per-plan conservative entry-string substitute) | Verified (entry-string level) | All 8 canary shapes map to an allow entry via prefix match (see evidence log AC9 section). True interactive prompt verification with `settings.local.json` disabled is out of band for this verifier session — see Coverage gaps. |
| AC10 — `./scripts/run-verify.sh` exits 0 | Partially verified | `HARNESS_VERIFY_MODE=static ./scripts/run-static-verify.sh` exit 0 (static portion of run-verify). Behavioral `run-verify.sh` full invocation is `/test`'s responsibility per skill boundaries. |
| AC11 — `docs/evidence/verify-<ts>.log` generated | Verified | `docs/evidence/verify-2026-04-17-051819.log` created by the static verifier; `docs/evidence/verify-2026-04-17-allow-go-and-repo-commands.log` created by this run. |
| AC12 — No allow-entry enumerations in AGENTS.md / CLAUDE.md / `.claude/rules/` | Verified | `grep -R "Bash(go "` and `grep -R "settings.json"` against those paths produced no matches. Extended grep for the full set of added prefixes only hits the plan file (`docs/plans/active/2026-04-17-allow-go-and-repo-commands.md:152`), which is expected — plans are allowed to describe the entries they introduce. |

## Static analysis

| Command | Result | Notes |
| --- | --- | --- |
| `HARNESS_VERIFY_MODE=static ./scripts/run-static-verify.sh` | PASS (exit 0) | gofmt: ok / 0 issues (golangci-lint) / `go vet` all 9 `internal/*` packages ok. Log auto-saved to `docs/evidence/verify-2026-04-17-051819.log`. |
| `jq -e . .claude/settings.json` | PASS (exit 0) | JSON syntax valid. |
| `jq -e . templates/base/.claude/settings.json` | PASS (exit 0) | JSON syntax valid; template copy is byte-identical. |
| `jq '.permissions.allow \| length as $l \| (unique \| length) == $l' .claude/settings.json` | PASS (`true`) | No duplicate entries in 60-element allow list. |
| `diff .claude/settings.json templates/base/.claude/settings.json` | PASS (no output) | Root and template copies byte-identical, as required by template sync. |
| `./scripts/check-template.sh` | PASS (exit 0) | Template structure OK. |
| `./scripts/check-pipeline-sync.sh` | PASS (exit 0) | All 8 pipeline-order reference sites in sync. |
| `./scripts/check-sync.sh` | Expected non-zero (exit 1) | `DRIFTED=0`, `IDENTICAL=105`. The only non-identical item is `ROOT_ONLY: docs/plans/active/2026-04-17-allow-go-and-repo-commands.md`, which is the active plan file and is resolved by `/pr` at plan archival. Not a regression. |

## Documentation drift

| Doc / contract | In sync? | Notes |
| --- | --- | --- |
| `AGENTS.md` | In sync | No enumeration of allow entries; primary-loop narrative unchanged by this diff. |
| `CLAUDE.md` | In sync | No enumeration. Default-behavior section is about skills, not permissions. |
| `.claude/rules/*.md` | In sync | `grep -R "Bash(go \|gofmt\|golangci-lint\|staticcheck\|goimports\|shellcheck\|./ralph\|bin/ralph\|./tests\|bash -n" .claude/rules/` returned no matches. |
| `templates/base/.claude/settings.json` | In sync | Byte-identical to `.claude/settings.json`. |
| `docs/plans/active/2026-04-17-allow-go-and-repo-commands.md` | In sync | Plan correctly references the added entries. `ROOT_ONLY` in `check-sync.sh` is expected for active plans and resolved by `/pr` archival. |
| `templates/base/CLAUDE.md` / `AGENTS.md` | In sync | Known-diff entries per `scripts/check-sync.sh`; not affected by this change. |

## Observational checks

- Diff shape confirmed additive. `git diff main...HEAD -- .claude/settings.json` shows 25 `+` lines and 1 `-` line (the latter is the old trailing-comma revision of `"Bash(claude:*)"`); the entry itself is preserved on the next line as `"Bash(claude:*)",`.
- The plan's Codex advisory resolution (exclude `sh:*`/`bash:*`/`xargs:*`) is visibly honored in the diff — jq search returns `[]`.
- `.claude/hooks/pre_bash_guard.sh` remains the runtime safety net; it runs before `allow` matching per Claude Code's hook order, so the relative security posture is unchanged by this diff (plan Assumption §3 holds).
- The repo state at verify time: branch `chore/allow-go-and-repo-commands`, tip `7295c69`. `git status` clean apart from an untracked self-review report (`docs/reports/self-review-2026-04-17-allow-go-and-repo-commands.md`), which is expected — it will be added in a later commit.

## Coverage gaps

- **AC9 interactive-prompt verification (likely but unverified)**: Confirming that, in a clean Claude Code session with `.claude/settings.local.json` emptied, the canary commands succeed *without a permission prompt* requires an interactive session, not a verifier pass. The plan itself substitutes an entry-string match as a conservative proxy, which this verifier performed. True permission-prompt verification belongs to `/test` or a manual smoke check, and any miss there would surface as a prompt rather than a failure.
- **AC10 full run-verify (verified in static mode only)**: The project's `./scripts/run-verify.sh` invokes both static and behavioral verifiers. Per skill boundaries, `/verify` executes only the static pass (`run-static-verify.sh`, which exited 0). Running the full `run-verify.sh` (including `go test`) is `/test`'s responsibility.
- **Prefix-matching semantics (likely but unverified)**: The plan's assumption that `Bash(<prefix>:*)` matches `<prefix> <any-args>` rests on observed behavior of the existing `Bash(./scripts/*)` entry. Not independently re-verified here; the risk of a mismatch (e.g., `Bash(go version)` being an exact-only entry) is low and would be caught by a runtime prompt, not a silent failure.

## Verdict

- Verified: AC1, AC2, AC3, AC4, AC5, AC6, AC7, AC8, AC11, AC12, plus the static-analysis slice of AC10, plus no regression in `check-template.sh`, `check-pipeline-sync.sh`, template byte-identity, and DRIFTED=0 in `check-sync.sh`.
- Partially verified: AC9 (entry-string level only), AC10 (static portion only).
- Not verified: none (no AC was left unchecked).
- **Overall verdict: PASS**. No failing criteria. Proceed to `/test`. The partial items are skill-boundary artifacts, not failures — behavioral run-verify and interactive prompt confirmation are explicitly `/test`'s scope.

## Minimal follow-up that would increase confidence most

One small behavioral check that would close the remaining gap, suitable for `/test`:

```sh
# In a clean Claude Code session (or with settings.local.json temporarily disabled):
go version && go vet ./... && gofmt -l . && bash -n scripts/ralph-pipeline.sh && ./tests/test-ralph-config.sh
```

If any of these triggers a permission prompt, the corresponding allow entry shape is wrong (e.g., `Bash(go version)` might need to be `Bash(go version:*)` — see self-review LOW note). Otherwise AC9 is fully verified. This is the single highest-value addition and fits cleanly in the `/test` step.
