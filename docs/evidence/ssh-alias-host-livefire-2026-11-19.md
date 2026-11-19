# ssh の別名の origin での doctor の実機確認(2026-11-19)

計画 `2026-11-19-org-ssh-alias-host.md`(issue #340)の AC7 の証跡。
7798b966 の binary を `go build -o <scratch>/ralph ./cmd/ralph` で作った(コードは
slice 2 の 40afa91e と同じで、そのあとのコミットは文書だけ)。比べる相手は Homebrew の
`ralph 8.2.1 (a8c5e289)`。gh は 2.103.3 で、github.com の yoshpy-dev のアカウントで
認証済み。ssh は OpenSSH_10.2p1。このマシンの `~/.ssh/config` には
`Host github.com.emu` があり、`ssh -G github.com.emu` は `hostname github.com` を返す。
doctor は clone せずに `git init` した空の repo に `git remote add origin <URL>` だけを
した場所で打ち、台帳は `--state-dir` で scratch のディレクトリに向けた。`<scratch>` は
session の scratchpad。

追記(2026-11-20): 上の「そのあとのコミットは文書だけ」は書いた時点のこと。7798b966 の
あとに 55cdb7c3(slice 5)と 3c369ff4(slice 6)がコードを変えた。binary を作り直した
verify と test の report が、同じ場面を HEAD の binary で打ち直し、この文書と同じ内容の
行を見た(path と `checked_at` は違う)。verify の 1 回目は HEAD `0a71a930` の binary で、
別名の origin の doctor、キャッシュ、`ralph status`、別名でない 2 つの origin、ssh を外した
doctor を打った(`docs/reports/verify-2026-11-19-org-ssh-alias-host.md` の Spec compliance
の AC7 と Observational checks)。test は HEAD `6a17dfad` の binary で、別名の origin の
doctor、キャッシュ、`ralph status`、ssh を外した doctor を打った
(`docs/reports/test-2026-11-19-org-ssh-alias-host.md` の「HEAD での実機の確かめ」)。

## 別名の origin(`ssh://git@github.com.emu/yoshpy-dev/ralph.git`)

8.2.1 では、別名をそのまま gh に渡して `unknown` になる(issue #340 と同じ)。

```
$ ralph doctor --state-dir <scratch>/ledger-old | grep -i "merge guarantee"
  ℹ Merge guarantee: info — unknown: gh api --hostname github.com.emu repos/yoshpy-dev/ralph failed (exit status 1: error connecting to github.com.emu check your internet connection or https://githubstatus.com): check that gh can reach github.com.emu and is logged in to it (gh auth status --hostname github.com.emu); if github.com.emu is an ssh alias (a Host in ~/.ssh/config), doctor cannot resolve it: point origin at the real host; cached in <scratch>/ledger-old/merge-guarantee.json
```

新しい binary では、`ssh -G` で `github.com` を解き、その host で保護を読む。

```
$ <scratch>/ralph doctor --state-dir <scratch>/ledger | grep -i "merge guarantee"
  ✓ Merge guarantee: pass — github_protection: main, the default branch of origin's repository yoshpy-dev/ralph, requires its 2 required status checks to pass on a head with the latest main before a PR merges: a ruleset requires branches to be up to date before merging (origin's host github.com.emu is an ssh alias of github.com, which ssh -G resolved and gh reads); cached in <scratch>/ledger/merge-guarantee.json
```

書かれたキャッシュでは、`host` が origin に書かれた別名のまま、`api_host` が gh に
渡した host になっている。

```json
{
  "version": 1,
  "kind": "github_protection",
  "detail": "main, the default branch of origin's repository yoshpy-dev/ralph, requires its 2 required status checks to pass on a head with the latest main before a PR merges: a ruleset requires branches to be up to date before merging (origin's host github.com.emu is an ssh alias of github.com, which ssh -G resolved and gh reads)",
  "host": "github.com.emu",
  "api_host": "github.com",
  "repo": "yoshpy-dev/ralph",
  "api_repo": "yoshpy-dev/ralph",
  "default_branch": "main",
  "checked_at": "2026-11-19T12:41:07Z"
}
```

`ralph status` は ssh も gh も打たずにこのキャッシュを読み、`origin_changed` には
ならない(AC4)。

```
$ <scratch>/ralph status --state-dir <scratch>/ledger | grep -i "merge guarantee"
merge guarantee: github_protection (main of github.com.emu/yoshpy-dev/ralph, checked 2026-11-19T12:41:07Z): main, the default branch of origin's repository yoshpy-dev/ralph, requires its 2 required status checks to pass on a head with the latest main before a PR merges: a ruleset requires branches to be up to date before merging (origin's host github.com.emu is an ssh alias of github.com, which ssh -G resolved and gh reads)
```

`ralph status --json` の `merge_guarantee` にも `"host": "github.com.emu"` と
`"api_host": "github.com"` が出た。

## 別名でない origin(AC3)

`git@github.com:yoshpy-dev/ralph.git` と `https://github.com/yoshpy-dev/ralph.git` で、
新しい binary と 8.2.1 の `Merge guarantee` の行(`; cached in ...` より前)を比べた。
どちらも 1 文字も違わなかった。

```
== git@github.com:yoshpy-dev/ralph.git
new:   ✓ Merge guarantee: pass — github_protection: main, the default branch of origin's repository yoshpy-dev/ralph, requires its 2 required status checks to pass on a head with the latest main before a PR merges: a ruleset requires branches to be up to date before merging
same as 8.2.1
== https://github.com/yoshpy-dev/ralph.git
new:   ✓ Merge guarantee: pass — github_protection: main, the default branch of origin's repository yoshpy-dev/ralph, requires its 2 required status checks to pass on a head with the latest main before a PR merges: a ruleset requires branches to be up to date before merging
same as 8.2.1
```

## ssh を解けないとき(AC5)

PATH を `/opt/homebrew/bin:/bin` にし、git と gh は残して ssh(`/usr/bin/ssh`)を
外した。別名の origin で gh が失敗し、案内に解けなかった理由と
`git config core.sshCommand` の手が出た。

```
$ PATH=/opt/homebrew/bin:/bin <scratch>/ralph doctor --state-dir <scratch>/l-nossh | grep "Merge guarantee"
  ℹ Merge guarantee: info — unknown: gh api --hostname github.com.emu repos/yoshpy-dev/ralph failed (exit status 1: error connecting to github.com.emu check your internet connection or https://githubstatus.com): check that gh can reach github.com.emu and is logged in to it (gh auth status --hostname github.com.emu); ssh -G github.com.emu failed (exec: "ssh": executable file not found in $PATH), so gh read the host github.com.emu as origin names it: if github.com.emu is an ssh alias (a Host in ~/.ssh/config), point origin at the real host (git remote set-url origin <URL>) and choose the ssh key for this repository with git config core.sshCommand 'ssh -i <key> -o IdentitiesOnly=yes'; cached in <scratch>/l-nossh/merge-guarantee.json
```

## 確かめていないこと

- `ralph org merge`、`ralph org cleanup`、`regate_checks = "ci"` の regate の実走。どれも
  org を立てる必要があるので、fake のテスト(`internal/org/merge_test.go`、
  `merge_ssh_alias_test.go`、`regate_ci_test.go`、`regate_ci_ssh_alias_test.go`、
  `mergeorigin_test.go`)で確かめた。別名の origin のテストは、da80502f で
  `merge_ssh_alias_test.go` と `regate_ci_ssh_alias_test.go` に移った。
- `Match exec` を書いた ssh の設定での実走。期限切れのときにプロセスグループごと止まる
  ことは、PATH に置いた偽の `ssh` を使う実プロセスのテスト(`mergeorigin_test.go`)で
  確かめた。
