# Walkthrough: org-ssh-alias-host

- Date: 2026-11-20
- Plan: docs/plans/archive/2026-11-19-org-ssh-alias-host.md (#340)
- Author: hiroki-yoshioka

## What changed

When origin was an ssh alias (a `Host github.com.emu` in `~/.ssh/config`
with `HostName github.com`), ralph passed the alias itself to gh as
`--hostname` and `-R <host>/<owner>/<repo>`. gh cannot reach such a host, so
`ralph doctor`'s Merge guarantee was `unknown` and `ralph org merge`,
`ralph org cleanup`, and a regate with `[org.gate].regate_checks = "ci"`
refused with nothing done, although gh itself (and git) work with the alias.

Now an ssh origin (`ssh://` or scp-like) is resolved the way gh resolves a
remote (go-gh's `pkg/ssh` Translator): `ssh -G <host>` with origin's host as
its URL writes it, the last `hostname` line, lowercased, then
`ssh.github.com` read as `github.com`. https, http, and `git://` run no ssh.
One function does it for every caller (`resolveMergeAPIHost` in
`internal/org/mergeorigin.go`), used by `ReadMergeOrigin` (merge, cleanup,
regate ci) and by doctor's `readOrigin`. When ssh -G does not resolve the
host (ssh missing, a failure, its own 5 s bound, no or an unusable hostname),
ralph goes on with origin's host as before, and the refusal of a gh that
fails names why and the `git config core.sshCommand` fix.

## Reading order

1. `internal/org/mergeorigin.go`: `resolveMergeAPIHost`, `mergeSSHHost`
   (the host as origin writes it), `mergeSSHHostName` (the plain-host check
   for both the host ssh gets and the hostname it prints),
   `mergeSSHResolveRuns` (the one predicate doctor shares), and
   `mergeSSHAliasHint`. `ReadMergeOrigin` takes an ssh runner; nil means
   `realMergeDetachedExec`, so a timeout or ctx kills ssh's whole process
   group, `Match exec` children included.
2. `internal/org/mergeorder.go`: `mergeOriginFromURLAsWritten`, the parse
   without lowercasing; `mergeOriginFromURL` keeps its values.
3. `internal/org/merge.go`, `regate_ci.go`, `internal/cli/org_cleanup.go`:
   the callers pass their ssh runner; merge's `ghAccess`/`ghMissing` add the
   hint.
4. `internal/org/mergeguarantee.go`: doctor resolves with the same function,
   the cache keeps `host` as origin names it (lowercased; `ralph status`
   compares it with origin's without ssh or gh) and adds `api_host`, and an
   alias Detail ends with a clause naming both hosts.
5. `internal/cli/doctor.go`, `doctor_merge_guarantee.go`: the Merge
   guarantee check runs under `signal.NotifyContext` (as the Node toolchain
   probe does), so Ctrl-C, SIGTERM, or SIGHUP stops ssh -G's group, and
   doctor prints an `Interrupted:` line, runs no later check, and exits 1.
6. Tests: `internal/org/mergeorigin_test.go` (the resolver table, real
   process tests with a fake `ssh` on PATH), `mergeguarantee_alias_test.go`,
   `merge_ssh_alias_test.go`, `regate_ci_ssh_alias_test.go`,
   `internal/cli/doctor_merge_guarantee_signal_test.go` (a real SIGINT case).
7. Docs: README (the doctor row and the merge paragraph), the org skill's
   four copies, the spec's 7c-2 note, the tech-debt register (the row of the
   alias limit resolved, a row for the director skill's origin check, a row
   for the LOW findings left), and
   `docs/evidence/ssh-alias-host-livefire-2026-11-19.md`.

## How the pipeline shaped it

The pipeline ran three times (the cap was raised to 3 for this task).

- The plan advisory (codex) found that killing only ssh on its timeout
  leaves `Match exec` children: ssh runs through `realMergeDetachedExec`.
- The first cross-review found that doctor ran the check under
  `context.Background()`, so a Ctrl-C left ssh's own group running (fixed in
  240324fe).
- The second cross-review found that ssh got the lowercased host, while
  OpenSSH matches `Host` with case and git and gh pass the host as written
  (fixed in 664d4771 and 04161b43; reproduced with OpenSSH_10.2p1 and an
  isolated `-F` config).
- The third cross-review had no findings.

## What was checked live, and what was not

Checked on this machine (OpenSSH_10.2p1, gh 2.103.3): with origin
`ssh://git@github.com.emu/yoshpy-dev/ralph.git`, ralph 8.2.1's doctor is
`unknown` and the new binary's is `github_protection`, with `host`
`github.com.emu` and `api_host` `github.com` in the cache, and `ralph status`
not `origin_changed`; non-alias origins print the same line as 8.2.1; with
no ssh on PATH the detail names the `core.sshCommand` fix. The verify and
test runs re-ran these with the HEAD binary, and sent signals to doctor's pid.

Not run live: `ralph org merge`, `ralph org cleanup`, and a ci regate (they
need an org; covered by fake tests), an ssh config with a real `Match exec`
(a fake ssh that starts a child stands in), and Ctrl-C typed in a terminal.
