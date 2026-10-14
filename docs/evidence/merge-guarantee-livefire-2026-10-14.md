# merge guarantee の実機確認(2026-10-14)

計画 `2026-10-14-org-merge-guarantee.md`(7c-2)の Verify plan にある実機の証跡。
c56c1f11 の binary を `go build -o <scratch>/ralph ./cmd/ralph` で作り、本物の
gh 2.102.0(github.com に認証済み、どの repository にも admin でない)で打った。
台帳は `RALPH_ORG_STATE_DIR` で scratch のディレクトリに向け、repo の台帳には何も
書いていない。github/docs・cli/cli・ssh の別名は、clone せずに `git init` した空の
repo に `git remote add origin <URL>` だけをした場所で打った(doctor が読むのは
origin の URL と gh api だけ)。`<scratch>` は session の scratchpad。detail の
長い文は `...` で略した。

## この repo(yoshpy-dev/ralph、main は strict のない ruleset だけ)

```
$ RALPH_ORG_STATE_DIR=<scratch>/ledger ralph doctor | grep -i "merge guarantee"
  ⚠ Merge guarantee: warn — pr_checks_only: main, the default branch of origin's repository yoshpy-dev/ralph, does not make a PR's required status checks pass on a head with the latest main before it merges: no status check is required and nothing requires branches to be up to date before merging; set Require branches to be up to date before merging with at least one required status check, in a ruleset or the branch protection of the branch, or add a merge queue with at least one required status check to a ruleset; cached in <scratch>/ledger/merge-guarantee.json

$ RALPH_ORG_STATE_DIR=<scratch>/ledger ralph status | head -3
no org runtime state found (state-dir=<scratch>/ledger (source: env)) — run `ralph org spawn` to start one.
org watch: not running
merge guarantee: pr_checks_only (main of github.com/yoshpy-dev/ralph, checked 2026-10-14T08:42:14Z): main, the default branch of origin's repository yoshpy-dev/ralph, does not make ...

$ RALPH_ORG_STATE_DIR=<scratch>/ledger ralph org merge-order
merge guarantee: pr_checks_only (main of github.com/yoshpy-dev/ralph, checked 2026-10-14T08:42:14Z): main, the default branch of ...
note: no feature branches in the ledger: a branch is listed once ralph org start --plan binds an org to it or ralph org gate records a gate on it
(rc=0)
```

書かれたキャッシュは次のとおり。

```json
{
  "version": 1,
  "kind": "pr_checks_only",
  "detail": "main, the default branch of origin's repository yoshpy-dev/ralph, does not make ...",
  "host": "github.com",
  "repo": "yoshpy-dev/ralph",
  "api_repo": "yoshpy-dev/ralph",
  "default_branch": "main",
  "checked_at": "2026-10-14T08:42:14Z"
}
```

同じキャッシュを、origin の違う場所と repo の外から読んだ status の行。

```
$ cd <origin が github/docs の repo> && RALPH_ORG_STATE_DIR=<scratch>/ledger ralph status | grep "merge guarantee"
merge guarantee: unknown (doctor checked github.com/yoshpy-dev/ralph; origin is now github.com/github/docs)
$ cd <scratch> && RALPH_ORG_STATE_DIR=<scratch>/ledger ralph status | grep "merge guarantee"
merge guarantee: unknown (origin cannot be read here (git remote get-url origin failed: exit status 128: fatal: not a git repository (or any of the parent directories): .git); doctor checked github.com/yoshpy-dev/ralph)
```

## github/docs(ruleset に strict と必須の検査)

```
  ✓ Merge guarantee: pass — github_protection: main, the default branch of origin's repository github/docs, requires its 37 required status checks to pass on a head with the latest main before a PR merges: a ruleset requires branches to be up to date before merging; cached in <scratch>/github-docs/ledger/merge-guarantee.json
merge guarantee: github_protection (main of github.com/github/docs, checked 2026-10-14T08:42:16Z): main, the default branch of origin's repository github/docs, requires its 37 required status checks to pass on a head with the latest main before a PR merges: a ruleset requires branches to be up to date before merging
```

## cli/cli(classic の保護に必須の検査、strict は admin でないと読めない)

```
  ℹ Merge guarantee: info — unknown: the branch protection of trunk, the default branch of origin's repository cli/cli, requires 3 required status checks, but whether it requires branches to be up to date before merging can be read only with admin access to the repository (gh api --hostname github.com repos/cli/cli/branches/trunk/protection/required_status_checks failed: exit status 1: gh: Not Found (HTTP 404)); cached in <scratch>/cli-cli/ledger/merge-guarantee.json
merge guarantee: unknown (trunk of github.com/cli/cli, checked 2026-10-14T08:42:19Z): the branch protection of trunk, ... can be read only with admin access to the repository (...)
```

## ssh の別名の host(origin `git@github.com.emu:acme/app.git`)

```
  ℹ Merge guarantee: info — unknown: gh api --hostname github.com.emu repos/acme/app failed (exit status 1: error connecting to github.com.emu check your internet connection or https://githubstatus.com): check that gh can reach github.com.emu and is logged in to it (gh auth status --hostname github.com.emu); if github.com.emu is an ssh alias (a Host in ~/.ssh/config), doctor cannot resolve it: point origin at the real host; cached in <scratch>/alias/ledger/merge-guarantee.json
merge guarantee: unknown (github.com.emu/acme/app, checked 2026-10-14T08:42:21Z): gh api --hostname github.com.emu repos/acme/app failed (...)
```

## 台帳のディレクトリがないとき

```
$ RALPH_ORG_STATE_DIR=<scratch>/none ralph doctor | grep -i "merge guarantee"
  ⚠ Merge guarantee: warn — pr_checks_only: main, ...; not cached: there is no org ledger yet at <scratch>/none
$ RALPH_ORG_STATE_DIR=<scratch>/none ralph status | grep "merge guarantee"
merge guarantee: unknown (no org ledger yet)
$ ls <scratch>/none
No such file or directory
```

doctor は台帳のディレクトリを作らない。
