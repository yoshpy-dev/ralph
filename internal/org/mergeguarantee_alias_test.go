package org

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"syscall"
	"testing"
	"time"
)

// doctor's merge guarantee on an ssh alias origin (plan
// docs/plans/active/2026-11-19-org-ssh-alias-host.md, slice 2: AC1, AC2,
// AC4, AC5 and AC5b on doctor's side): CheckMergeGuarantee resolves an ssh
// origin's host with the same resolveMergeAPIHost as ReadMergeOrigin,
// through mgExec's ssh -G, and the cache keeps origin's host as written
// beside the host gh read. No test runs the real ssh: the fakes answer it,
// and the one test of the real runner puts a fake ssh at the front of PATH.

// mgAliasOrigin is origin's URL on the ssh alias of the issue (#340), soEmu.
const mgAliasOrigin = "git@github.com.emu:acme/app.git"

// mgAliasSSH is what ssh -G github.com.emu prints for that alias, in part.
var mgAliasSSH = soLines("host github.com.emu", "user git", "hostname github.com", "port 22")

// mgAliasNote ends the Detail of a check whose ssh -G resolved
// github.com.emu to github.com.
const mgAliasNote = " (origin's host github.com.emu is an ssh alias of github.com, which ssh -G resolved and gh reads)"

// mgProtectedDetail is the Detail of mgFullExec's github_protection.
const mgProtectedDetail = mgSubject + ", requires its 1 required status check to pass on a head with the latest main before a " +
	"PR merges: the branch protection requires branches to be up to date before merging"

// TestMergeGuarantee_SSHAliasOriginIsReadOnTheHostSSHResolves covers AC1 of
// the plan: for an alias origin, scp-like or ssh:// (any case, with a port),
// the check runs one `ssh -G github.com.emu` in /repo with no env under
// mergeSSHResolveTimeout, every gh call carries --hostname github.com, and
// the kind is decided from the protection read there. Host stays
// github.com.emu, APIHost is github.com, and the Detail ends with the clause
// that names both. An alias of ssh.github.com is read as github.com, and a
// hostname in capitals is lowercased.
func TestMergeGuarantee_SSHAliasOriginIsReadOnTheHostSSHResolves(t *testing.T) {
	for _, tc := range []struct {
		name, origin, ssh string
		newExec           func() *mgExec
		kind, detail      string
		endpoints         []string
	}{
		{"scp-like alias", mgAliasOrigin, mgAliasSSH, mgFullExec, MergeGuaranteeGitHubProtection, mgProtectedDetail + mgAliasNote,
			[]string{mgRepoEP, mgRulesEP, mgBranchEP, mgStrictEP}},
		{"ssh:// alias in capitals with a port", "ssh://git@GitHub.com.EMU:22/acme/app.git", mgAliasSSH, mgFullExec,
			MergeGuaranteeGitHubProtection, mgProtectedDetail + mgAliasNote, []string{mgRepoEP, mgRulesEP, mgBranchEP, mgStrictEP}},
		{"alias of ssh.github.com", mgAliasOrigin, soLines("hostname ssh.github.com"), mgFullExec, MergeGuaranteeGitHubProtection,
			mgProtectedDetail + mgAliasNote, []string{mgRepoEP, mgRulesEP, mgBranchEP, mgStrictEP}},
		{"hostname in capitals", mgAliasOrigin, soLines("hostname GitHub.COM"), mgFullExec, MergeGuaranteeGitHubProtection,
			mgProtectedDetail + mgAliasNote, []string{mgRepoEP, mgRulesEP, mgBranchEP, mgStrictEP}},
		{"alias with no protection", mgAliasOrigin, mgAliasSSH, newMGExec, MergeGuaranteePRChecksOnly,
			mgSubject + ", does not make a PR's required status checks pass on a head with the latest main before it merges: no " +
				"status check is required and nothing requires branches to be up to date before merging" + mgFix + mgAliasNote,
			[]string{mgRepoEP, mgRulesEP, mgBranchEP}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			e := tc.newExec()
			e.origin = tc.origin + "\n"
			e.ssh = &mgReply{out: tc.ssh}
			start := time.Now()
			c := mgCheck(e)
			end := time.Now()
			if c.Kind != tc.kind || c.Host != soEmu || c.APIHost != "github.com" || c.Repo != "acme/app" {
				t.Fatalf("check = %s host %q api_host %q repo %q (%s), want %s host %s api_host github.com repo acme/app", c.Kind,
					c.Host, c.APIHost, c.Repo, c.Detail, tc.kind, soEmu)
			}
			if c.Detail != tc.detail {
				t.Errorf("detail = %q\nwant %q", c.Detail, tc.detail)
			}
			if got := e.endpoints(); !slices.Equal(got, tc.endpoints) {
				t.Errorf("gh endpoints = %q, want %q", got, tc.endpoints)
			}
			e.assertGHCalls(t, "github.com")
			ssh := e.sshSnapshot()
			if len(ssh) != 1 || !slices.Equal(ssh[0].args, []string{"-G", soEmu}) || ssh[0].dir != "/repo" || ssh[0].env != nil ||
				ssh[0].deadline.Before(start.Add(mergeSSHResolveTimeout)) || ssh[0].deadline.After(end.Add(mergeSSHResolveTimeout)) {
				t.Errorf("ssh calls = %+v, want one ssh -G %s in /repo with no env and a deadline 5 s after it (%v..%v)", ssh, soEmu,
					start, end)
			}
		})
	}
}

// TestMergeGuarantee_GHFailureNamesTheSSHKeyFixOnlyForAnUnresolvedAlias
// covers AC5 of the plan on doctor's side: when ssh -G does not resolve an
// ssh origin's host (it fails, is not installed, or prints no hostname) and
// gh then fails on that host, the Detail says why and names the fix with
// core.sshCommand. When ssh -G resolves the alias and gh still fails, only
// gh's access guidance remains, for the resolved host, with the alias
// clause. No such clause comes for an https origin on another host, an ssh
// origin ssh -G gives back, or a github.com origin whose ssh -G fails.
func TestMergeGuarantee_GHFailureNamesTheSSHKeyFixOnlyForAnUnresolvedAlias(t *testing.T) {
	const ghErr = "exit status 1: error connecting to github.com.emu"
	access := func(host string) string {
		return "gh api --hostname " + host + " repos/acme/app failed (" + ghErr + "): check that gh can reach " + host + " and is " +
			"logged in to it (gh auth status --hostname " + host + ")"
	}
	const keyFix = ", so gh read the host github.com.emu as origin names it: if github.com.emu is an ssh alias (a Host in " +
		"~/.ssh/config), point origin at the real host (git remote set-url origin <URL>) and choose the ssh key for this " +
		"repository with git config core.sshCommand 'ssh -i <key> -o IdentitiesOnly=yes'"
	for _, tc := range []struct {
		name, origin  string
		ssh           *mgReply // nil: ssh -G gives the host back
		host, apiHost string
		sshRuns       bool
		detail        string
	}{
		{"ssh -G fails", mgAliasOrigin, &mgReply{err: errors.New("exit status 255: no such identity")}, soEmu, soEmu, true,
			"gh api --hostname github.com.emu repos/acme/app failed (exit status 1: error connecting to github.com.emu): check " +
				"that gh can reach github.com.emu and is logged in to it (gh auth status --hostname github.com.emu); ssh -G " +
				"github.com.emu failed (exit status 255: no such identity), so gh read the host github.com.emu as origin names " +
				"it: if github.com.emu is an ssh alias (a Host in ~/.ssh/config), point origin at the real host (git remote " +
				"set-url origin <URL>) and choose the ssh key for this repository with git config core.sshCommand 'ssh -i " +
				"<key> -o IdentitiesOnly=yes'"},
		{"ssh is not installed", mgAliasOrigin, &mgReply{err: &exec.Error{Name: "ssh", Err: exec.ErrNotFound}}, soEmu, soEmu, true,
			access(soEmu) + `; ssh -G github.com.emu failed (exec: "ssh": executable file not found in $PATH)` + keyFix},
		{"ssh -G prints no hostname", mgAliasOrigin, &mgReply{out: soLines("user git", "port 22")}, soEmu, soEmu, true,
			access(soEmu) + "; ssh -G github.com.emu printed no hostname line" + keyFix},
		{"ssh -G resolves the alias and gh still fails", mgAliasOrigin, &mgReply{out: mgAliasSSH}, soEmu, "github.com", true,
			access("github.com") + mgAliasNote},
		{"https origin on another host", "https://ghe.example.com/acme/app.git", nil, "ghe.example.com", "ghe.example.com", false,
			access("ghe.example.com")},
		{"ssh origin ssh -G gives back", "git@ghe.example.com:acme/app.git", nil, "ghe.example.com", "ghe.example.com", true,
			access("ghe.example.com")},
		{"github.com origin whose ssh -G fails", mgOriginURL, &mgReply{err: errors.New("exit status 255")}, "github.com",
			"github.com", true, access("github.com")},
	} {
		t.Run(tc.name, func(t *testing.T) {
			e := newMGExec()
			e.origin = tc.origin + "\n"
			e.ssh = tc.ssh
			e.replies[mgRepoEP] = mgReply{err: errors.New(ghErr)}
			c := mgCheck(e)
			if c.Kind != MergeGuaranteeUnknown || c.Detail != tc.detail {
				t.Errorf("check = %s %q\nwant unknown %q", c.Kind, c.Detail, tc.detail)
			}
			if c.Host != tc.host || c.APIHost != tc.apiHost {
				t.Errorf("host, api_host = %q, %q, want %q, %q", c.Host, c.APIHost, tc.host, tc.apiHost)
			}
			if got := e.endpoints(); !slices.Equal(got, []string{mgRepoEP}) {
				t.Errorf("gh endpoints = %q, want the repository's only", got)
			}
			e.assertGHCalls(t, tc.apiHost)
			if n := len(e.sshSnapshot()); n != map[bool]int{true: 1, false: 0}[tc.sshRuns] {
				t.Errorf("ssh ran %d times, want it run %t", n, tc.sshRuns)
			}
		})
	}
}

// TestMergeGuarantee_CacheKeepsOriginsHostAndTheHostGHRead covers AC4 of the
// plan: the cache of an alias origin holds host github.com.emu, as origin's
// URL names it, and api_host github.com. ReadMergeGuarantee, whose git still
// reads the alias URL, gives the cached kind with both hosts (no
// origin_changed), runs one git and neither gh nor ssh, and --json carries
// api_host. Pointed at the real host, origin is another origin until doctor
// runs again; a cache with no api_host, as an older ralph wrote it, still
// reads, with none.
func TestMergeGuarantee_CacheKeepsOriginsHostAndTheHostGHRead(t *testing.T) {
	stateDir := mgStateDir(t)
	e := mgFullExec()
	e.origin = mgAliasOrigin + "\n"
	e.ssh = &mgReply{out: mgAliasSSH}
	c := mgCheck(e)
	if c.Kind != MergeGuaranteeGitHubProtection {
		t.Fatalf("check = %s (%s), want github_protection", c.Kind, c.Detail)
	}
	mgSave(t, stateDir, c)
	data, err := os.ReadFile(MergeGuaranteePathIn(stateDir))
	if err != nil {
		t.Fatal(err)
	}
	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatal(err)
	}
	if raw["host"] != soEmu || raw["api_host"] != "github.com" || raw["version"] != float64(1) {
		t.Errorf("cache = %s, want version 1, host %s and api_host github.com", data, soEmu)
	}

	read := newMGExec()
	read.origin = mgAliasOrigin + "\n"
	read.replies = nil
	g := ReadMergeGuarantee(context.Background(), stateDir, mgReadParams(read, mgClock))
	want := MergeGuarantee{
		Kind: MergeGuaranteeGitHubProtection, Detail: c.Detail, Host: soEmu, APIHost: "github.com", Repo: "acme/app",
		APIRepo: "acme/app", DefaultBranch: "main", CheckedAt: mgCheckedAt,
	}
	if g != want {
		t.Fatalf("read = %+v\nwant %+v", g, want)
	}
	if text := g.Text(); text != "github_protection (main of github.com.emu/acme/app, checked "+mgCheckedAt+"): "+c.Detail {
		t.Errorf("text = %q", text)
	}
	if js := mgJSON(g); !strings.Contains(js, `"host":"github.com.emu"`) || !strings.Contains(js, `"api_host":"github.com"`) {
		t.Errorf("--json = %s, want host github.com.emu and api_host github.com", js)
	}
	if git, gh, ssh := read.callsOf("git"), read.callsOf("gh"), read.sshSnapshot(); len(git) != 1 || len(gh) != 0 || len(ssh) != 0 {
		t.Errorf("read ran git %+v, gh %+v, ssh %+v; want one git and no gh or ssh", git, gh, ssh)
	}

	moved := ReadMergeGuarantee(context.Background(), stateDir, mgReadParams(newMGExec(), mgClock))
	if want := "unknown (doctor checked github.com.emu/acme/app; origin is now github.com/acme/app)"; moved.Reason !=
		MergeGuaranteeReasonOriginChanged || moved.Text() != want || moved.APIHost != "github.com" {
		t.Errorf("read with origin on the real host = %+v, text %q, want origin_changed %q", moved, moved.Text(), want)
	}

	old := mgStateDir(t)
	const noAPIHost = `{"version":1,"kind":"github_protection","detail":"d","host":"github.com","repo":"acme/app",` +
		`"api_repo":"acme/app","default_branch":"main","checked_at":"` + mgCheckedAt + `"}`
	mgWriteRawCache(noAPIHost)(t, old)
	g = ReadMergeGuarantee(context.Background(), old, mgReadParams(newMGExec(), mgClock))
	if g.Kind != MergeGuaranteeGitHubProtection || g.Reason != "" || g.Host != "github.com" || g.APIHost != "" {
		t.Errorf("read of a cache with no api_host = %+v, want the cached kind with no APIHost", g)
	}
	if js := mgJSON(g); strings.Contains(js, "api_host") {
		t.Errorf("--json of a cache with no api_host = %s, want no api_host", js)
	}
}

// TestMergeGuarantee_DoctorAndMergeReadAnAliasOnTheSameHost covers AC2 of the
// plan: with one fake exec for git, ssh and gh, doctor's gh calls carry as
// --hostname the host of ReadMergeOrigin's Arg, which ralph org merge pins gh
// pr to with -R, and CheckMergeEnforcement, given ReadMergeOrigin's host as
// merge passes it, reads the protection there too: github.com when ssh -G
// resolves the alias, the alias itself when it fails.
func TestMergeGuarantee_DoctorAndMergeReadAnAliasOnTheSameHost(t *testing.T) {
	for _, tc := range []struct {
		name string
		ssh  mgReply
		host string
	}{
		{"ssh -G resolves the alias", mgReply{out: mgAliasSSH}, "github.com"},
		{"ssh -G fails", mgReply{err: errors.New("exit status 255")}, soEmu},
	} {
		t.Run(tc.name, func(t *testing.T) {
			e := mgFullExec()
			e.origin = mgAliasOrigin + "\n"
			e.ssh = &tc.ssh
			c := mgCheck(e)
			o, problem, err := ReadMergeOrigin(context.Background(), "/repo", e.run, e.run, time.Second)
			if problem != MergeOriginRead || err != nil {
				t.Fatalf("ReadMergeOrigin = %+v, %d, %v; want origin read", o, problem, err)
			}
			host, _, _ := strings.Cut(o.Arg(), "/")
			if host != tc.host || c.APIHost != host || o.Arg() != host+"/acme/app" {
				t.Fatalf("merge reads %s (host %s), doctor's APIHost is %s; want both on %s", o.Arg(), host, c.APIHost, tc.host)
			}
			e.assertGHCalls(t, host)
			if ssh := e.sshSnapshot(); len(ssh) != 2 || !slices.Equal(ssh[0].args, ssh[1].args) {
				t.Errorf("ssh calls = %+v, want the same ssh -G for doctor and merge", ssh)
			}

			enf := newMEExec()
			r := CheckMergeEnforcement(context.Background(), MergeEnforcementParams{Dir: "/repo", Exec: enf.run, Host: o.apiHost,
				Owner: "acme", Name: "app", DefaultBranch: "main"})
			if !r.OK || len(enf.callsOf("gh")) == 0 {
				t.Errorf("CheckMergeEnforcement = %+v with gh calls %q, want ok", r, enf.endpoints())
			}
			meAssertGHCalls(t, enf, host, "acme", "app", "main")
		})
	}
}

// TestMergeGuarantee_InterruptedWhileSSHRunsIsUnknown: the caller's ctx done
// while the ssh -G of an alias origin runs makes the check unknown, naming
// ssh -G as what ran, with origin's host and owner/repo recorded, no
// APIHost, no alias clause, and no gh run.
func TestMergeGuarantee_InterruptedWhileSSHRunsIsUnknown(t *testing.T) {
	ctx, cancel := context.WithCancelCause(context.Background())
	defer cancel(nil)
	e := newMGExec()
	e.origin = mgAliasOrigin + "\n"
	e.block = "ssh"
	go func() {
		for end := time.Now().Add(10 * time.Second); len(e.sshSnapshot()) == 0 && time.Now().Before(end); {
			time.Sleep(time.Millisecond)
		}
		cancel(errors.New("doctor stopped"))
	}()
	c := CheckMergeGuarantee(ctx, mgParams(e))
	want := "the check was interrupted (doctor stopped) while ssh -G github.com.emu ran, so the protection of origin's " +
		"repository's default branch is unknown"
	if c.Kind != MergeGuaranteeUnknown || c.Detail != want {
		t.Errorf("check = %s %q\nwant unknown %q", c.Kind, c.Detail, want)
	}
	if c.Host != soEmu || c.Repo != "acme/app" || c.APIHost != "" {
		t.Errorf("host, repo, api_host = %q, %q, %q, want %s, acme/app and none", c.Host, c.Repo, c.APIHost, soEmu)
	}
	if gh := e.callsOf("gh"); len(gh) != 0 {
		t.Errorf("gh ran: %q", e.endpoints())
	}
}

// TestMergeGuarantee_SSHRunsThroughExecOrInAProcessGroupOfItsOwn covers AC5b
// of the plan for doctor: with no Exec, git and gh run through
// realMergeOrderExec and ssh -G through realMergeDetachedExec
// (newMergeGuaranteeRun); with Exec set, git and ssh -G both run through it.
// With no Exec the check runs the real `git remote get-url origin` in Dir and
// ssh -G in a process group of its own led by ssh, here a fake ssh at the
// front of PATH that writes its pid and process group and names github.com
// for the alias; gh is not on PATH, so the check is unknown, with both hosts
// recorded.
func TestMergeGuarantee_SSHRunsThroughExecOrInAProcessGroupOfItsOwn(t *testing.T) {
	ctx := context.Background()
	g := newMergeGuaranteeRun(ctx, ctx, time.Second, MergeGuaranteeParams{}, &MergeGuaranteeCheck{})
	if reflect.ValueOf(g.run).Pointer() != reflect.ValueOf(realMergeOrderExec).Pointer() ||
		reflect.ValueOf(g.sshRun).Pointer() != reflect.ValueOf(realMergeDetachedExec).Pointer() ||
		reflect.ValueOf(g.lookPath).Pointer() != reflect.ValueOf(exec.LookPath).Pointer() {
		t.Error("newMergeGuaranteeRun with no Exec: git and gh do not run through realMergeOrderExec, ssh -G not through " +
			"realMergeDetachedExec, or gh is not looked up with exec.LookPath")
	}
	e := newMGExec()
	g = newMergeGuaranteeRun(ctx, ctx, time.Second, MergeGuaranteeParams{Exec: e.run}, &MergeGuaranteeCheck{})
	if _, err := g.sshRun(ctx, "/repo", nil, "ssh", "-G", soEmu); err != nil {
		t.Errorf("ssh -G through Exec: %v", err)
	}
	if _, err := g.run(ctx, "/repo", nil, "git", "remote", "get-url", "origin"); err != nil {
		t.Errorf("git through Exec: %v", err)
	}
	if len(e.sshSnapshot()) != 1 || len(e.callsOf("git")) != 1 {
		t.Errorf("Exec ran ssh %+v and git %+v, want one of each", e.sshSnapshot(), e.callsOf("git"))
	}

	isolateGitEnv(t)
	root := resolved(t, t.TempDir())
	runGit(t, "init", "--quiet", root)
	runGit(t, "-C", root, "remote", "add", "origin", mgAliasOrigin)
	group := filepath.Join(t.TempDir(), "group")
	soFakeSSH(t, "echo \"$$ $(ps -o pgid= -p $$)\" > "+ShellWord(group)+"\necho 'hostname github.com'\n")
	c := CheckMergeGuarantee(ctx, MergeGuaranteeParams{Dir: root, LookPath: mgNotFound, Now: func() time.Time { return mgClock }})
	wantDetail := `gh is not on PATH (exec: "gh": executable file not found in $PATH), so the protection of the default branch ` +
		"of origin's repository acme/app was not read" + mgAliasNote
	if c.Kind != MergeGuaranteeUnknown || c.Detail != wantDetail || c.Host != soEmu || c.APIHost != "github.com" {
		t.Fatalf("check = %s %q host %q api_host %q\nwant unknown %q host %s api_host github.com", c.Kind, c.Detail, c.Host,
			c.APIHost, wantDetail, soEmu)
	}
	b, err := os.ReadFile(group)
	var pid, pgid int
	if _, scanErr := fmt.Sscan(string(b), &pid, &pgid); err != nil || scanErr != nil {
		t.Fatalf("the fake ssh wrote %q, %v (%v)", b, err, scanErr)
	}
	if test := syscall.Getpgrp(); pgid != pid || pgid == test {
		t.Errorf("ssh (pid %d) ran in the process group %d; want its own, %d (the test's is %d)", pid, pgid, pid, test)
	}
}
