# verify cycle 2 の証跡: org-merge-guarantee(2026-10-14)

計画 `2026-10-14-org-merge-guarantee.md` の post-implementation pipeline の 2 回目の /verify で取った証跡。
binary は HEAD 95e97635 から `GOTOOLCHAIN=go1.25.8 go build -o <scratch>/ralph ./cmd/ralph` で作り、
対照の binary は fix の前の fb954403 を `git archive` で scratch に出して同じように作った。
`<scratch>` は session の scratchpad の下のディレクトリ。台帳はどれも `RALPH_ORG_STATE_DIR` で
`<scratch>` の下に向け、repo の台帳(`.harness/state/org/`)には何も書いていない。

手で書いたのはこの説明と見出しだけで、コードブロックの中は scratch のファイルの中身をそのまま入れた
(scratch の path を `<scratch>` に置き換えた)。`run-static-verify.sh` が自分で書いた生の出力は
`docs/evidence/verify-2026-10-14-124031.log`(gitignore の対象、名前の時刻は UTC)。

## 1. 本物の gh で cli/cli(読むだけ)

`git init` した空の repo に `git remote add origin https://github.com/cli/cli.git` だけをした場所で、
HEAD の binary の doctor を本物の gh 2.102.0 で打った。gh は読み取りの GET だけを打つ。
doctor の終了コードは 0。

```
$ RALPH_ORG_STATE_DIR=<scratch>/cli-cli/ledger <scratch>/ralph doctor | grep -i 'merge guarantee'
  ℹ Merge guarantee: info — unknown: the branch protection of trunk, the default branch of origin's repository cli/cli, requires 3 required status checks, but whether it requires branches to be up to date before merging can be read only with admin access to the repository (gh api --hostname github.com repos/cli/cli/branches/trunk/protection/required_status_checks failed: exit status 1: gh: Not Found (HTTP 404)); cached in <scratch>/cli-cli/ledger/merge-guarantee.json

$ RALPH_ORG_STATE_DIR=<scratch>/cli-cli/ledger <scratch>/ralph status | head -3
no org runtime state found (state-dir=<scratch>/cli-cli/ledger (source: env)) — run `ralph org spawn` to start one.
org watch: not running
merge guarantee: unknown (trunk of github.com/cli/cli, checked 2026-10-14T12:40:52Z): the branch protection of trunk, the default branch of origin's repository cli/cli, requires 3 required status checks, but whether it requires branches to be up to date before merging can be read only with admin access to the repository (gh api --hostname github.com repos/cli/cli/branches/trunk/protection/required_status_checks failed: exit status 1: gh: Not Found (HTTP 404))
```

doctor の行を `docs/evidence/merge-guarantee-livefire-2026-10-14.md` の 63 行目と `cmp` で比べると同じ
(`SAME_LINE`)。404 `Not Found` の文は fix で変わっていない。

## 2. 偽の gh で fix の 2 点を binary で確かめる(ネットワークに出ない)

PATH は `<scratch>/fakebin:/usr/bin:/bin` で、`fakebin` には下の偽の `gh` と、本物の git への symlink
だけを置いた。origin は `https://github.com/acme/app.git`。`FAKE_MODE` で strict の endpoint の答え
(`esc` は最初の repository の呼び出しの失敗)を選ぶ。doctor の終了コードの 1 は、絞った PATH に
claude がないことによる `✗ Claude Code CLI: fail` で、Merge guarantee の検査は `pass` か `info` を返す。
ESC と BEL の数え方の対照: `printf 'a\033b\007c' | LC_ALL=C tr -cd '\033\007' | wc -c` は 2。

### 偽の gh(`<scratch>/fakebin/gh`)

```sh
#!/bin/sh
# Fake gh for the cycle-2 verify probe of org-merge-guarantee.
# Answers only `gh api --hostname <host> [--paginate] <path>`; never reaches the network.
# FAKE_MODE picks the strict endpoint's answer (or the repository call's failure).
printf '%s\n' "$*" >> "${FAKE_LOG:?}"
path=""
for a in "$@"; do path="$a"; done
esc=$(printf '\033')
case "$path" in
repos/acme/app)
  if [ "$FAKE_MODE" = esc ]; then
    printf 'gh: %s[31mBad credentials%s[0m\a (HTTP 401)\n' "$esc" "$esc" >&2
    exit 1
  fi
  printf '{"default_branch":"main","full_name":"acme/app"}\n' ;;
repos/acme/app/rules/branches/main)
  printf '[]\n' ;;
repos/acme/app/branches/main)
  printf '{"name":"main","protected":true,"protection":{"enabled":true,"required_status_checks":{"enforcement_level":"everyone","contexts":["ci"],"checks":[{"context":"ci","app_id":null}]}}}\n' ;;
repos/acme/app/branches/main/protection/required_status_checks)
  case "$FAKE_MODE" in
  f403)
    printf '{"message":"Resource not accessible by personal access token","documentation_url":"https://docs.github.com/rest","status":"403"}\n'
    printf 'gh: Resource not accessible by personal access token (HTTP 403)\n' >&2
    exit 1 ;;
  f404nf)
    printf '{"message":"Not Found","documentation_url":"https://docs.github.com/rest","status":"404"}\n'
    printf 'gh: Not Found (HTTP 404)\n' >&2
    exit 1 ;;
  f404bnf)
    printf '{"message":"Branch not found","documentation_url":"https://docs.github.com/rest","status":"404"}\n'
    printf 'gh: Branch not found (HTTP 404)\n' >&2
    exit 1 ;;
  *)
    printf '{"url":"x","strict":true,"contexts":["ci"]}\n' ;;
  esac ;;
*)
  printf 'gh: unexpected path %s\n' "$path" >&2
  exit 2 ;;
esac
```

### 打ち方(`<scratch>/probe.sh`)

```sh
#!/bin/sh
# Cycle-2 verify probe: HEAD binary + fake gh (no network) + scratch repo and ledger.
set -u
S=$(cd "$(dirname "$0")" && pwd)
chmod +x "$S/fakebin/gh"
ln -sf /opt/homebrew/bin/git "$S/fakebin/git"
P="$S/fakebin:/usr/bin:/bin"
out="$S/probe.out"
: > "$out"
for mode in strict f403 f404nf f404bnf esc; do
  d="$S/p-$mode"
  rm -rf "$d"
  mkdir -p "$d/work" "$d/ledger"
  (cd "$d/work" && git init -q && git remote add origin https://github.com/acme/app.git)
  log="$d/gh-calls.log"
  : > "$log"
  {
    echo "### mode=$mode"
    (cd "$d/work" && FAKE_MODE=$mode FAKE_LOG="$log" PATH="$P" RALPH_ORG_STATE_DIR="$d/ledger" "$S/ralph" doctor > "$d/doctor.out" 2>&1; echo "doctor rc=$?")
    grep -a 'Merge guarantee' "$d/doctor.out" | sed "s#$S#<scratch>#g"
    printf 'raw ESC bytes in the doctor line: '
    grep -a 'Merge guarantee' "$d/doctor.out" | LC_ALL=C tr -cd '\033\007' | wc -c | tr -d ' '
    echo "gh calls:"
    sed 's/^/  gh /' "$log"
    (cd "$d/work" && PATH="$P" RALPH_ORG_STATE_DIR="$d/ledger" "$S/ralph" status 2>&1 | grep -a '^merge guarantee:' | sed "s#$S#<scratch>#g")
    printf 'raw ESC bytes in the status line: '
    (cd "$d/work" && PATH="$P" RALPH_ORG_STATE_DIR="$d/ledger" "$S/ralph" status 2>&1 | grep -a '^merge guarantee:' | LC_ALL=C tr -cd '\033\007' | wc -c | tr -d ' ')
    echo
  } >> "$out"
done
```

### HEAD 95e97635 の binary(`<scratch>/probe.out`)

```
### mode=strict
doctor rc=1
  ✓ Merge guarantee: pass — github_protection: main, the default branch of origin's repository acme/app, requires its 1 required status check to pass on a head with the latest main before a PR merges: the branch protection requires branches to be up to date before merging; cached in <scratch>/p-strict/ledger/merge-guarantee.json
raw ESC bytes in the doctor line: 0
gh calls:
  gh api --hostname github.com repos/acme/app
  gh api --hostname github.com --paginate repos/acme/app/rules/branches/main
  gh api --hostname github.com repos/acme/app/branches/main
  gh api --hostname github.com repos/acme/app/branches/main/protection/required_status_checks
merge guarantee: github_protection (main of github.com/acme/app, checked 2026-10-14T12:43:43Z): main, the default branch of origin's repository acme/app, requires its 1 required status check to pass on a head with the latest main before a PR merges: the branch protection requires branches to be up to date before merging
raw ESC bytes in the status line: 0

### mode=f403
doctor rc=1
  ℹ Merge guarantee: info — unknown: the branch protection of main, the default branch of origin's repository acme/app, requires 1 required status check, but whether it requires branches to be up to date before merging is unknown: GitHub answered 403, which can come from missing admin access to the repository or from another cause, such as a rate limit or an SSO authorization the token lacks (gh api --hostname github.com repos/acme/app/branches/main/protection/required_status_checks failed: exit status 1: gh: Resource not accessible by personal access token (HTTP 403)); cached in <scratch>/p-f403/ledger/merge-guarantee.json
raw ESC bytes in the doctor line: 0
gh calls:
  gh api --hostname github.com repos/acme/app
  gh api --hostname github.com --paginate repos/acme/app/rules/branches/main
  gh api --hostname github.com repos/acme/app/branches/main
  gh api --hostname github.com repos/acme/app/branches/main/protection/required_status_checks
merge guarantee: unknown (main of github.com/acme/app, checked 2026-10-14T12:43:44Z): the branch protection of main, the default branch of origin's repository acme/app, requires 1 required status check, but whether it requires branches to be up to date before merging is unknown: GitHub answered 403, which can come from missing admin access to the repository or from another cause, such as a rate limit or an SSO authorization the token lacks (gh api --hostname github.com repos/acme/app/branches/main/protection/required_status_checks failed: exit status 1: gh: Resource not accessible by personal access token (HTTP 403))
raw ESC bytes in the status line: 0

### mode=f404nf
doctor rc=1
  ℹ Merge guarantee: info — unknown: the branch protection of main, the default branch of origin's repository acme/app, requires 1 required status check, but whether it requires branches to be up to date before merging can be read only with admin access to the repository (gh api --hostname github.com repos/acme/app/branches/main/protection/required_status_checks failed: exit status 1: gh: Not Found (HTTP 404)); cached in <scratch>/p-f404nf/ledger/merge-guarantee.json
raw ESC bytes in the doctor line: 0
gh calls:
  gh api --hostname github.com repos/acme/app
  gh api --hostname github.com --paginate repos/acme/app/rules/branches/main
  gh api --hostname github.com repos/acme/app/branches/main
  gh api --hostname github.com repos/acme/app/branches/main/protection/required_status_checks
merge guarantee: unknown (main of github.com/acme/app, checked 2026-10-14T12:43:44Z): the branch protection of main, the default branch of origin's repository acme/app, requires 1 required status check, but whether it requires branches to be up to date before merging can be read only with admin access to the repository (gh api --hostname github.com repos/acme/app/branches/main/protection/required_status_checks failed: exit status 1: gh: Not Found (HTTP 404))
raw ESC bytes in the status line: 0

### mode=f404bnf
doctor rc=1
  ℹ Merge guarantee: info — unknown: gh api --hostname github.com repos/acme/app/branches/main/protection/required_status_checks failed (exit status 1: gh: Branch not found (HTTP 404)), so whether the branch protection of main, the default branch of origin's repository acme/app, requires branches to be up to date before merging is unknown; cached in <scratch>/p-f404bnf/ledger/merge-guarantee.json
raw ESC bytes in the doctor line: 0
gh calls:
  gh api --hostname github.com repos/acme/app
  gh api --hostname github.com --paginate repos/acme/app/rules/branches/main
  gh api --hostname github.com repos/acme/app/branches/main
  gh api --hostname github.com repos/acme/app/branches/main/protection/required_status_checks
merge guarantee: unknown (main of github.com/acme/app, checked 2026-10-14T12:43:45Z): gh api --hostname github.com repos/acme/app/branches/main/protection/required_status_checks failed (exit status 1: gh: Branch not found (HTTP 404)), so whether the branch protection of main, the default branch of origin's repository acme/app, requires branches to be up to date before merging is unknown
raw ESC bytes in the status line: 0

### mode=esc
doctor rc=1
  ℹ Merge guarantee: info — unknown: gh api --hostname github.com repos/acme/app failed (exit status 1: gh: \x1b[31mBad credentials\x1b[0m\a (HTTP 401)): check that gh can reach github.com and is logged in to it (gh auth status --hostname github.com); cached in <scratch>/p-esc/ledger/merge-guarantee.json
raw ESC bytes in the doctor line: 0
gh calls:
  gh api --hostname github.com repos/acme/app
merge guarantee: unknown (github.com/acme/app, checked 2026-10-14T12:43:45Z): gh api --hostname github.com repos/acme/app failed (exit status 1: gh: \x1b[31mBad credentials\x1b[0m\a (HTTP 401)): check that gh can reach github.com and is logged in to it (gh auth status --hostname github.com)
raw ESC bytes in the status line: 0
```

### 対照: fix の前の fb954403 の binary(`<scratch>/probe-prefix.out`)

同じ偽の gh と PATH で、`esc` と `f403` だけを打った。この出力は、端末に生の制御文字を出さないよう、
doctor の行の ESC を `E`、BEL を `B` に `tr` で置き換えてから書いた(数はその前に数えた)。

```
### fb954403 mode=esc
  ℹ Merge guarantee: info — unknown: gh api --hostname github.com repos/acme/app failed (exit status 1: gh: E[31mBad credentialsE[0mB (HTTP 401)): check that gh can reach github.com and is logged in to it (gh auth status --hostname github.com); cached in <scratch>/q-esc/ledger/merge-guarantee.json
raw ESC/BEL bytes in the doctor line: 3

### fb954403 mode=f403
  ℹ Merge guarantee: info — unknown: the branch protection of main, the default branch of origin's repository acme/app, requires 1 required status check, but whether it requires branches to be up to date before merging can be read only with admin access to the repository (gh api --hostname github.com repos/acme/app/branches/main/protection/required_status_checks failed: exit status 1: gh: Resource not accessible by personal access token (HTTP 403)); cached in <scratch>/q-f403/ledger/merge-guarantee.json
raw ESC/BEL bytes in the doctor line: 0
```

## 3. 静的解析(HEAD 95e97635)

| Command | Result |
| --- | --- |
| `./scripts/run-static-verify.sh`(changed scope、golang の pack) | rc 0。`check-sync.sh` PASS(IDENTICAL 167、DRIFTED 0)、`check-pipeline-sync.sh` ok、`check-skill-sync: 13 skill(s) in lock-step`、`check-template-purity.sh` PASS、tech-debt の plan の参照 OK、`gofmt: ok`、golangci-lint `0 issues.`、`secret-scan-branch: scanned a16d7c4f..95e97635 against origin/main: clean`、`All verifiers passed.` |
| `gofmt -l internal/ cmd/` | 出力なし、rc 0 |
| `go vet ./internal/org/... ./internal/cli/...`(go1.26.0) | 出力なし、rc 0 |
| `GOTOOLCHAIN=go1.25.8 go vet ./...` | 出力なし、rc 0 |
| `GOTOOLCHAIN=go1.25.8 go build ./...` | rc 0 |
| `./scripts/check-skill-sync.sh` | `[ok] check-skill-sync: 13 skill(s) in lock-step` |
| `./scripts/check-sync.sh` | IDENTICAL 167、DRIFTED 0、`PASS: all files in sync.` |
| `git diff --check fb954403 HEAD` | 出力なし、rc 0 |
| `./scripts/plan-visual.sh digest docs/plans/active/2026-10-14-org-merge-guarantee.md` | `d2923a8bf359`(`- Approved:` の digest と同じ) |
| `git diff --name-only fb954403 HEAD` の各ファイルで U+FFFD を grep | 0 件(grep の rc 1) |
