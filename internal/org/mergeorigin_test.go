package org

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"
)

// The resolution of an ssh origin's host with `ssh -G` (plan
// docs/plans/active/2026-11-19-org-ssh-alias-host.md, slice 1: AC3, AC5,
// AC5b, AC6): resolveMergeAPIHost and ReadMergeOrigin with a fake ssh
// (soExec), and, for AC5b, with a fake ssh script at the front of PATH run
// through realMergeDetachedExec. No test runs the real ssh, so no developer's
// ~/.ssh/config is read.

// soExec is a fake MergeOrderExec for the ssh -G of resolveMergeAPIHost:
// every call is recorded with its ctx's deadline; during, when set, runs
// first inside the call; with block the call waits for its ctx; a call whose
// ctx is done fails with ctx's error, else it answers out and err.
type soExec struct {
	mu     sync.Mutex
	calls  []mxCall
	out    string
	err    error
	block  bool
	during func()
}

func (e *soExec) run(ctx context.Context, dir string, env []string, name string, args ...string) (string, error) {
	deadline, _ := ctx.Deadline()
	e.mu.Lock()
	e.calls = append(e.calls, mxCall{dir: dir, env: slices.Clone(env), name: name, args: slices.Clone(args), deadline: deadline})
	e.mu.Unlock()
	if e.during != nil {
		e.during()
	}
	if e.block {
		<-ctx.Done()
	}
	if err := ctx.Err(); err != nil {
		return "", err
	}
	return e.out, e.err
}

func (e *soExec) snapshot() []mxCall {
	e.mu.Lock()
	defer e.mu.Unlock()
	return slices.Clone(e.calls)
}

// soLines is ssh -G output: each line and a newline after it.
func soLines(lines ...string) string { return strings.Join(lines, "\n") + "\n" }

// soEmu is the ssh alias of the issue (#340): ~/.ssh/config's
// `Host github.com.emu` with `Hostname github.com`.
const soEmu = "github.com.emu"

// soEmuSSH is what ssh -G github.com.emu prints for that alias, in part.
var soEmuSSH = soLines("host github.com.emu", "user git", "hostname github.com", "port 22", "identityfile ~/.ssh/id_emu")

// TestResolveMergeAPIHost_ResolvesAnSSHOriginsHostAsGHDoes covers the
// resolution of plan docs/plans/active/2026-11-19-org-ssh-alias-host.md (D1,
// D6, AC3, AC5): for an ssh URL, scp-like or ssh:// (any case), it runs one
// `ssh -G <host>` in dir with no env, under a deadline timeout from the call,
// <host> as the URL writes it, case kept (cycle 2 of
// docs/reports/cross-review-triage-org-ssh-alias-host.md: OpenSSH matches a
// Host with case, and git and gh pass the host as written), and takes the
// value of the last line whose first word is exactly hostname, lowercased,
// with ssh.github.com then read as github.com; for https, http, and git:// it
// runs nothing. The host it falls back to is lowercased, and the clause names
// the command with the host ssh -G got. A host whose lowercase is a plain
// host name but which is not one as written (the Kelvin sign U+212A, which
// lowercases to k) runs nothing. A host that is no plain host name (it starts
// with "-", or holds a character such as $, (, ;, |, a backquote, a quote or
// white space, which a `Match exec` line with %h would hand to a shell) runs
// nothing and leaves origin's host with a clause that quotes it, while
// ordinary hosts (an alias with "_" or "-", an IPv6 address) run ssh -G. An
// ssh -G that fails, is not installed, or times out, output with no hostname
// line, and a hostname that is empty, starts with "-", or holds white space,
// a control character, "/", invalid UTF-8, or a character such as ;, $ or a
// quote leave origin's host, with a clause that names the command and why.
// The why of a host and of a hostname that is no plain host name uses the
// same words, ralph's rule and not gh's: it starts with "-", or it is not a
// plain host name. In every case mergeSSHResolveRuns, which doctor's
// readOrigin reads, says whether ssh -G ran (L1, L3, and L10 of the
// self-review, docs/reports/self-review-2026-11-19-org-ssh-alias-host.md).
func TestResolveMergeAPIHost_ResolvesAnSSHOriginsHostAsGHDoes(t *testing.T) {
	const alias = "git@github.com.emu:o/r.git"
	const notPlain = `which is not a plain host name (only ASCII letters, digits, ".", "_", ":" and "-")`
	notAHost := func(value string) string {
		return fmt.Sprintf("ssh -G github.com.emu printed the hostname %q, %s", value, notPlain)
	}
	notRun := func(host string) string {
		return fmt.Sprintf("ssh -G was not run for the host %q, %s", host, notPlain)
	}
	for _, tc := range []struct {
		name, url string
		ssh       *soExec
		timeout   time.Duration // 0: a second
		wantHost  string
		wantWhy   string
		wantSSH   string // the host ssh -G is run for, "" for none
	}{
		{"scp-like alias", alias, &soExec{out: soEmuSSH}, 0, "github.com", "", soEmu},
		{"ssh:// alias with a port", "ssh://git@github.com.emu:22/o/r.git", &soExec{out: soEmuSSH}, 0, "github.com", "", soEmu},
		{"SSH:// in capitals", "SSH://git@github.com.emu/o/r.git", &soExec{out: soEmuSSH}, 0, "github.com", "", soEmu},
		{"an ssh host that is no alias", "git@ghe.example.com:o/r.git", &soExec{out: soLines("hostname ghe.example.com")}, 0,
			"ghe.example.com", "", "ghe.example.com"},
		{"https runs no ssh", "https://github.com/o/r.git", &soExec{out: soEmuSSH}, 0, "github.com", "", ""},
		{"http runs no ssh", "http://ghe.example.com/o/r.git", &soExec{out: soEmuSSH}, 0, "ghe.example.com", "", ""},
		{"git:// runs no ssh", "git://github.com/o/r.git", &soExec{out: soEmuSSH}, 0, "github.com", "", ""},
		{"https on ssh.github.com is github.com", "https://ssh.github.com/o/r.git", &soExec{}, 0, "github.com", "", ""},
		{"ssh fails", alias, &soExec{err: errors.New("exit status 255: command-line line 0: garbage")}, 0, soEmu,
			"ssh -G github.com.emu failed (exit status 255: command-line line 0: garbage)", soEmu},
		{"ssh is not installed", alias, &soExec{err: &exec.Error{Name: "ssh", Err: exec.ErrNotFound}}, 0, soEmu,
			`ssh -G github.com.emu failed (exec: "ssh": executable file not found in $PATH)`, soEmu},
		{"ssh times out", alias, &soExec{block: true}, 30 * time.Millisecond, soEmu, "ssh -G github.com.emu timed out after 30ms",
			soEmu},
		{"no hostname line", alias, &soExec{out: soLines("user git", "port 22")}, 0, soEmu,
			"ssh -G github.com.emu printed no hostname line", soEmu},
		{"no output", alias, &soExec{}, 0, soEmu, "ssh -G github.com.emu printed no hostname line", soEmu},
		{"hostname with nothing after it", alias, &soExec{out: soLines("hostname")}, 0, soEmu,
			"ssh -G github.com.emu printed no hostname line", soEmu},
		{"an empty hostname", alias, &soExec{out: "hostname  \r\n"}, 0, soEmu, "ssh -G github.com.emu printed an empty hostname", soEmu},
		{"a hostname that starts with -", alias, &soExec{out: soLines("hostname -oProxyCommand=x")}, 0, soEmu,
			`ssh -G github.com.emu printed the hostname "-oProxyCommand=x", which starts with "-"`, soEmu},
		{"a hostname with a space", alias, &soExec{out: soLines("hostname github.com evil.example.com")}, 0, soEmu,
			notAHost("github.com evil.example.com"), soEmu},
		{"a hostname with a tab", alias, &soExec{out: soLines("hostname github.com\tx")}, 0, soEmu, notAHost("github.com\tx"), soEmu},
		{"a hostname with a /", alias, &soExec{out: soLines("hostname github.com/x")}, 0, soEmu, notAHost("github.com/x"), soEmu},
		{"a hostname with a control character", alias, &soExec{out: soLines("hostname git\x1bhub.com")}, 0, soEmu,
			notAHost("git\x1bhub.com"), soEmu},
		{"a hostname that is not UTF-8", alias, &soExec{out: soLines("hostname git\xffhub.com")}, 0, soEmu,
			notAHost("git\xffhub.com"), soEmu},
		{"the last hostname line wins", alias,
			&soExec{out: soLines("hostname first.example.com", "hostkeyalias other.example.com", "hostname github.com")}, 0,
			"github.com", "", soEmu},
		{"a key that only starts with hostname is not hostname", alias,
			&soExec{out: soLines("hostname github.com", "hostnamex other.example.com", "Hostname other.example.com")}, 0,
			"github.com", "", soEmu},
		{"a hostname with CRLF", alias, &soExec{out: "hostname github.com\r\n"}, 0, "github.com", "", soEmu},
		{"an alias of ssh.github.com is github.com", alias, &soExec{out: soLines("hostname SSH.GitHub.com")}, 0, "github.com", "",
			soEmu},
		{"a hostname in capitals is lowercased", "git@ghe-alias:o/r.git", &soExec{out: soLines("hostname GHE.Example.COM")}, 0,
			"ghe.example.com", "", "ghe-alias"},
		// Origin's host in capitals reaches ssh -G as written; the hostname
		// ssh -G prints and the host ralph falls back to are lowercased.
		{"an scp-like host in capitals is passed as written", "git@GitHub.COM:o/r.git", &soExec{out: soLines("hostname github.com")},
			0, "github.com", "", "GitHub.COM"},
		{"an ssh:// alias in capitals is passed as written", "ssh://git@Work-GH/o/r.git",
			&soExec{out: soLines("hostname github.com")}, 0, "github.com", "", "Work-GH"},
		{"an alias in capitals with a hostname in capitals", "git@Work-GH:o/r.git", &soExec{out: soLines("hostname GHE.Example.COM")},
			0, "ghe.example.com", "", "Work-GH"},
		{"an alias in capitals whose ssh -G fails falls back lowercased", "git@Work-GH:o/r.git",
			&soExec{err: errors.New("exit status 255")}, 0, "work-gh", "ssh -G Work-GH failed (exit status 255)", "Work-GH"},
		{"an alias in capitals with a port and no hostname line", "ssh://git@Work-GH:22/o/r.git", &soExec{out: soLines("user git")},
			0, "work-gh", "ssh -G Work-GH printed no hostname line", "Work-GH"},
		{"SSH.GitHub.com whose ssh -G fails is github.com", "git@SSH.GitHub.com:o/r.git", &soExec{err: errors.New("exit status 255")},
			0, "github.com", "ssh -G SSH.GitHub.com failed (exit status 255)", "SSH.GitHub.com"},
		{"a host in capitals that is no plain host name is quoted as written", "git@Work;GH:o/r.git", &soExec{out: soEmuSSH}, 0,
			"work;gh", notRun("Work;GH"), ""},
		{"a host whose lowercase is ASCII runs no ssh", "git@\xe2\x84\xaa.example.com:o/r.git", &soExec{out: soEmuSSH}, 0,
			"k.example.com", notRun("\xe2\x84\xaa.example.com"), ""},
		{"a host that starts with - runs no ssh", "ssh://git@-x.example.com/o/r.git", &soExec{out: soEmuSSH}, 0, "-x.example.com",
			`ssh -G was not run for the host "-x.example.com", which starts with "-"`, ""},
		{"an IPv6 host", "ssh://git@[::1]:22/o/r.git", &soExec{out: soLines("hostname ::1")}, 0, "::1", "", "::1"},
		// Hosts that are no plain host name (L1 of the self-review): origin's
		// URL can name them, and none reaches ssh -G.
		{"a host $(id) runs no ssh", "ssh://git@$(id)/o/r.git", &soExec{out: soEmuSSH}, 0, "$(id)", notRun("$(id)"), ""},
		{"a host a;touch runs no ssh", "ssh://git@a;touch/o/r.git", &soExec{out: soEmuSSH}, 0, "a;touch", notRun("a;touch"), ""},
		{"a host with a space runs no ssh", "git@a b:o/r.git", &soExec{out: soEmuSSH}, 0, "a b", notRun("a b"), ""},
		{"a host a|b runs no ssh", "git@a|b:o/r.git", &soExec{out: soEmuSSH}, 0, "a|b", notRun("a|b"), ""},
		{"a host with a backquote runs no ssh", "git@a`id`:o/r.git", &soExec{out: soEmuSSH}, 0, "a`id`", notRun("a`id`"), ""},
		{"a host with a quote runs no ssh", "ssh://git@a'b/o/r.git", &soExec{out: soEmuSSH}, 0, "a'b", notRun("a'b"), ""},
		{"a host with a double quote runs no ssh", `git@a"b:o/r.git`, &soExec{out: soEmuSSH}, 0, `a"b`, notRun(`a"b`), ""},
		{"a host with a control character runs no ssh", "git@a\x1bb:o/r.git", &soExec{out: soEmuSSH}, 0, "a\x1bb",
			notRun("a\x1bb"), ""},
		// Ordinary hosts still run ssh -G.
		{"an alias with _", "git@my_alias:o/r.git", &soExec{out: soLines("hostname github.com")}, 0, "github.com", "", "my_alias"},
		{"an alias with -", "git@gh-work:o/r.git", &soExec{out: soLines("hostname github.com")}, 0, "github.com", "", "gh-work"},
		{"an IPv4 host", "git@10.0.0.1:o/r.git", &soExec{out: soLines("hostname 10.0.0.1")}, 0, "10.0.0.1", "", "10.0.0.1"},
		// Hostnames ssh -G prints that are no plain host name.
		{"a hostname with a ;", alias, &soExec{out: soLines("hostname github.com;touch")}, 0, soEmu, notAHost("github.com;touch"),
			soEmu},
		{"a hostname with a $", alias, &soExec{out: soLines("hostname $(id).example.com")}, 0, soEmu,
			notAHost("$(id).example.com"), soEmu},
		{"a hostname with a quote", alias, &soExec{out: soLines("hostname git'hub.com")}, 0, soEmu, notAHost("git'hub.com"), soEmu},
		{"a hostname that is not ASCII", alias, &soExec{out: soLines("hostname gïthub.com")}, 0, soEmu, notAHost("gïthub.com"),
			soEmu},
	} {
		t.Run(tc.name, func(t *testing.T) {
			host, _, ok := mergeOriginFromURL(tc.url)
			if !ok {
				t.Fatalf("mergeOriginFromURL(%q) reads no repository", tc.url)
			}
			if runs := mergeSSHResolveRuns(tc.url, host); runs != (tc.wantSSH != "") {
				t.Errorf("mergeSSHResolveRuns(%q, %q) = %t; want %t, as ssh -G runs", tc.url, host, runs, tc.wantSSH != "")
			}
			timeout := cmp.Or(tc.timeout, time.Second)
			start := time.Now()
			apiHost, why := resolveMergeAPIHost(context.Background(), "/repo", tc.ssh.run, timeout, tc.url, host)
			end := time.Now()
			if apiHost != tc.wantHost || why != tc.wantWhy {
				t.Errorf("resolveMergeAPIHost(%q) = %q, %q; want %q, %q", tc.url, apiHost, why, tc.wantHost, tc.wantWhy)
			}
			calls := tc.ssh.snapshot()
			if tc.wantSSH == "" {
				if len(calls) != 0 {
					t.Errorf("ssh calls = %+v; want none", calls)
				}
				return
			}
			if len(calls) != 1 || calls[0].name != "ssh" || !slices.Equal(calls[0].args, []string{"-G", tc.wantSSH}) ||
				calls[0].dir != "/repo" || calls[0].env != nil {
				t.Fatalf("ssh calls = %+v; want one ssh -G %s in /repo with no env", calls, tc.wantSSH)
			}
			if d := calls[0].deadline; d.Before(start.Add(timeout)) || d.After(end.Add(timeout)) {
				t.Errorf("ssh -G's deadline %v is not %s after the call (%v..%v)", d, timeout, start, end)
			}
		})
	}
}

// TestMergeSSHHostName: a plain host name is not empty, does not start with
// "-", and holds only ASCII letters and digits, ".", "_", ":" and "-"; every
// other byte, a shell character, white space, a control character, "/", a
// bracket, "%", a byte that is not ASCII, and invalid UTF-8, makes it none.
func TestMergeSSHHostName(t *testing.T) {
	for _, s := range []string{"github.com", "GitHub.COM", "github.com.emu", "my_alias", "gh-work", "a-", "a", "::1",
		"10.0.0.1", "fe80::1", "ABCxyz0129._:-"} {
		if !mergeSSHHostName(s) {
			t.Errorf("mergeSSHHostName(%q) = false; want true", s)
		}
	}
	for _, s := range []string{"", "-", "-x", "--help", "-oProxyCommand=x", "$(id)", "a;touch", "a b", "a\tb", "a\nb", "a|b",
		"a&b", "a`id`", "a'b", `a"b`, "a\\b", "a>b", "a<b", "a*b", "a?b", "a~b", "a!b", "a=b", "a,b", "a+b", "a@b", "a#b",
		"a/b", "[::1]", "fe80::1%en0", "a\x1bb", "a\x00b", "a\x7fb", "gïthub.com", "git\xffhub.com"} {
		if mergeSSHHostName(s) {
			t.Errorf("mergeSSHHostName(%q) = true; want false", s)
		}
	}
}

// TestResolveMergeAPIHost_DefaultsAndInterruption: a timeout of zero or less
// is mergeSSHResolveTimeout (5 s); a ctx with an earlier deadline bounds ssh
// -G by it; and ctx done while ssh -G runs leaves origin's host with a clause
// that says it was interrupted and why, not that it timed out.
func TestResolveMergeAPIHost_DefaultsAndInterruption(t *testing.T) {
	const alias = "git@github.com.emu:o/r.git"
	if mergeSSHResolveTimeout != 5*time.Second {
		t.Errorf("mergeSSHResolveTimeout = %s; want 5s", mergeSSHResolveTimeout)
	}
	for _, timeout := range []time.Duration{0, -time.Second} {
		t.Run(fmt.Sprintf("timeout %s", timeout), func(t *testing.T) {
			e := &soExec{out: soEmuSSH}
			start := time.Now()
			if host, why := resolveMergeAPIHost(context.Background(), "", e.run, timeout, alias, soEmu); host != "github.com" || why != "" {
				t.Fatalf("resolveMergeAPIHost = %q, %q; want github.com", host, why)
			}
			end := time.Now()
			calls := e.snapshot()
			if len(calls) != 1 || calls[0].deadline.Before(start.Add(mergeSSHResolveTimeout)) ||
				calls[0].deadline.After(end.Add(mergeSSHResolveTimeout)) {
				t.Errorf("ssh calls = %+v; want one with a deadline 5 s after the call (%v..%v)", calls, start, end)
			}
		})
	}
	t.Run("ctx's earlier deadline", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		want, _ := ctx.Deadline()
		e := &soExec{out: soEmuSSH}
		resolveMergeAPIHost(ctx, "", e.run, time.Minute, alias, soEmu)
		if calls := e.snapshot(); len(calls) != 1 || !calls[0].deadline.Equal(want) {
			t.Errorf("ssh calls = %+v; want one with ctx's deadline %v", calls, want)
		}
	})
	t.Run("interrupted", func(t *testing.T) {
		ctx, cancel := context.WithCancelCause(context.Background())
		defer cancel(nil)
		e := &soExec{block: true, during: func() { cancel(errors.New("test stopped")) }}
		host, why := resolveMergeAPIHost(ctx, "", e.run, time.Minute, alias, soEmu)
		if host != soEmu || why != "ssh -G github.com.emu was interrupted (test stopped)" {
			t.Errorf("resolveMergeAPIHost = %q, %q; want origin's host and the interruption", host, why)
		}
	})
}

// TestMergeSSHAliasHint: the clause is "" when ssh -G resolved the host (no
// why) and when gh read github.com, which is no alias; otherwise it says why,
// that gh read origin's host as it is, and the fix: point origin at the real
// host, and choose the ssh key with git config core.sshCommand.
func TestMergeSSHAliasHint(t *testing.T) {
	for _, tc := range []struct {
		name, originHost, apiHost, why, want string
	}{
		{"resolved", soEmu, "github.com", "", ""},
		{"resolved to the same host", "ghe.example.com", "ghe.example.com", "", ""},
		{"github.com", "github.com", "github.com", "ssh -G github.com failed (exit status 255)", ""},
		{"ssh.github.com", "ssh.github.com", "github.com", "ssh -G ssh.github.com failed (exit status 255)", ""},
		{"unresolved alias", soEmu, soEmu, "ssh -G github.com.emu timed out after 5s", "; ssh -G github.com.emu timed out after " +
			"5s, so gh read the host github.com.emu as origin names it: if github.com.emu is an ssh alias (a Host in " +
			"~/.ssh/config), point origin at the real host (git remote set-url origin <URL>) and choose the ssh key for this " +
			"repository with git config core.sshCommand 'ssh -i <key> -o IdentitiesOnly=yes'"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := mergeSSHAliasHint(tc.originHost, tc.apiHost, tc.why); got != tc.want {
				t.Errorf("mergeSSHAliasHint = %q\nwant %q", got, tc.want)
			}
		})
	}
}

// TestReadMergeOrigin_ResolvesAnSSHAlias covers AC6 and AC5 of the plan for
// ReadMergeOrigin: an alias origin is read on the host ssh -G names, so Arg
// is github.com/o/r and a PR whose url is on github.com is origin's while one
// on the alias is not; ssh -G runs through sshRun in dir under its own 5 s
// deadline, apart from the get-url's timeout. An ssh -G that fails reads
// origin's host as it is, with MergeOriginRead and why kept. An https origin
// runs no ssh. Origin's host in capitals reaches ssh -G as written, while Arg,
// originHost, and the host of an ssh -G that fails are lowercased. ctx done
// while ssh -G runs is MergeOriginInterrupted with the zero MergeOrigin.
func TestReadMergeOrigin_ResolvesAnSSHAlias(t *testing.T) {
	const alias = "git@github.com.emu:o/r.git"
	t.Run("resolved", func(t *testing.T) {
		git := &moExec{origin: alias}
		ssh := &soExec{out: soEmuSSH}
		start := time.Now()
		o, problem, err := ReadMergeOrigin(context.Background(), "/repo", git.run, ssh.run, 100*time.Millisecond)
		end := time.Now()
		if problem != MergeOriginRead || err != nil || o.Arg() != "github.com/o/r" || o.originHost != soEmu || o.unresolvedWhy != "" {
			t.Fatalf("ReadMergeOrigin = %+v, %d, %v; want github.com/o/r read from the alias", o, problem, err)
		}
		if !o.hasPR("https://github.com/o/r/pull/7") || o.hasPR("https://github.com.emu/o/r/pull/7") {
			t.Errorf("hasPR: a PR on github.com is %t, one on the alias %t; want true and false",
				o.hasPR("https://github.com/o/r/pull/7"), o.hasPR("https://github.com.emu/o/r/pull/7"))
		}
		if why := o.foreignPRWhy("feat/a", &MergeOrderPR{Number: 7, URL: "https://github.com/o/r/pull/7"}); why != "" {
			t.Errorf("foreignPRWhy = %q; want the PR taken as origin's", why)
		}
		calls := ssh.snapshot()
		if len(calls) != 1 || !slices.Equal(calls[0].args, []string{"-G", soEmu}) || calls[0].dir != "/repo" ||
			calls[0].deadline.Before(start.Add(mergeSSHResolveTimeout)) || calls[0].deadline.After(end.Add(mergeSSHResolveTimeout)) {
			t.Errorf("ssh calls = %+v; want one ssh -G %s in /repo with a deadline 5 s after it (%v..%v)", calls, soEmu, start, end)
		}
		if got := git.callsOf("ssh"); len(got) != 0 {
			t.Errorf("ssh ran through the git runner: %+v", got)
		}
	})
	t.Run("ssh -G fails", func(t *testing.T) {
		git := &moExec{origin: alias}
		ssh := &soExec{err: errors.New("exit status 255")}
		o, problem, err := ReadMergeOrigin(context.Background(), "/repo", git.run, ssh.run, time.Second)
		if problem != MergeOriginRead || err != nil || o.Arg() != "github.com.emu/o/r" || o.originHost != soEmu ||
			o.unresolvedWhy != "ssh -G github.com.emu failed (exit status 255)" {
			t.Fatalf("ReadMergeOrigin = %+v, %d, %v; want github.com.emu/o/r with why", o, problem, err)
		}
	})
	t.Run("a host in capitals reaches ssh -G as written", func(t *testing.T) {
		git := &moExec{origin: "git@GitHub.COM:o/r.git"}
		ssh := &soExec{out: soLines("hostname GitHub.COM")}
		o, problem, err := ReadMergeOrigin(context.Background(), "/repo", git.run, ssh.run, time.Second)
		if problem != MergeOriginRead || err != nil || o.Arg() != "github.com/o/r" || o.originHost != "github.com" ||
			o.unresolvedWhy != "" {
			t.Fatalf("ReadMergeOrigin = %+v, %d, %v; want github.com/o/r, lowercased", o, problem, err)
		}
		if calls := ssh.snapshot(); len(calls) != 1 || !slices.Equal(calls[0].args, []string{"-G", "GitHub.COM"}) {
			t.Errorf("ssh calls = %+v; want one ssh -G GitHub.COM, origin's host as written", calls)
		}
	})
	t.Run("an alias in capitals whose ssh -G fails", func(t *testing.T) {
		git := &moExec{origin: "git@Work-GH:o/r.git"}
		ssh := &soExec{err: errors.New("exit status 255")}
		o, problem, err := ReadMergeOrigin(context.Background(), "/repo", git.run, ssh.run, time.Second)
		if problem != MergeOriginRead || err != nil || o.Arg() != "work-gh/o/r" || o.originHost != "work-gh" ||
			o.unresolvedWhy != "ssh -G Work-GH failed (exit status 255)" {
			t.Fatalf("ReadMergeOrigin = %+v, %d, %v; want work-gh/o/r with why naming ssh -G Work-GH", o, problem, err)
		}
	})
	t.Run("an https origin runs no ssh", func(t *testing.T) {
		git := &moExec{origin: "https://github.com.emu/o/r.git"}
		ssh := &soExec{out: soEmuSSH}
		o, problem, err := ReadMergeOrigin(context.Background(), "/repo", git.run, ssh.run, time.Second)
		if calls := ssh.snapshot(); problem != MergeOriginRead || err != nil || o.Arg() != "github.com.emu/o/r" ||
			o.unresolvedWhy != "" || len(calls) != 0 {
			t.Fatalf("ReadMergeOrigin = %+v, %d, %v (ssh calls %+v); want github.com.emu/o/r with no ssh", o, problem, err, calls)
		}
	})
	t.Run("interrupted while ssh -G runs", func(t *testing.T) {
		ctx, cancel := context.WithCancelCause(context.Background())
		defer cancel(nil)
		git := &moExec{origin: alias}
		ssh := &soExec{block: true, during: func() { cancel(errors.New("test stopped")) }}
		o, problem, err := ReadMergeOrigin(ctx, "/repo", git.run, ssh.run, time.Second)
		if problem != MergeOriginInterrupted || err != nil || o != (MergeOrigin{}) || len(ssh.snapshot()) != 1 {
			t.Fatalf("ReadMergeOrigin = %+v, %d, %v; want the zero MergeOrigin, MergeOriginInterrupted, no error", o, problem, err)
		}
	})
}

// soFakeSSH puts an executable ssh, a sh script with body, in a temp dir at
// the front of PATH for the test, so realMergeDetachedExec runs it and not
// the real ssh.
func soFakeSSH(t *testing.T, body string) {
	t.Helper()
	bin := t.TempDir()
	if err := os.WriteFile(filepath.Join(bin, "ssh"), []byte("#!/bin/sh\n"+body), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
}

// soWaitFile is the content of path once it ends with a newline, waiting up
// to 10 s for it; "" when it does not by then.
func soWaitFile(path string) string {
	for end := time.Now().Add(10 * time.Second); time.Now().Before(end); time.Sleep(10 * time.Millisecond) {
		if b, err := os.ReadFile(path); err == nil && strings.HasSuffix(string(b), "\n") {
			return string(b)
		}
	}
	return ""
}

// soAssertGone fails the test unless the process pid ends within 5 s
// (kill -0 answers ESRCH), and kills it when it does not.
func soAssertGone(t *testing.T, pid int) {
	t.Helper()
	var err error
	for end := time.Now().Add(5 * time.Second); time.Now().Before(end); time.Sleep(20 * time.Millisecond) {
		if err = syscall.Kill(pid, 0); errors.Is(err, syscall.ESRCH) {
			return
		}
	}
	t.Errorf("the fake ssh's child %d still runs 5 s after resolveMergeAPIHost returned (kill -0: %v); want it killed with "+
		"ssh's process group", pid, err)
	_ = syscall.Kill(pid, syscall.SIGKILL)
}

// TestResolveMergeAPIHost_RealSSHThatHangsEndsWithItsChildren covers AC5b of
// the plan: through realMergeDetachedExec, an ssh -G that does not return,
// here a fake ssh that starts a child (as OpenSSH runs a `Match exec`
// command) and waits for it, is stopped by the ssh -G timeout and by the
// caller's ctx alike: the call returns with origin's host and why, and the
// child ends too, killed with ssh's process group.
func TestResolveMergeAPIHost_RealSSHThatHangsEndsWithItsChildren(t *testing.T) {
	for _, tc := range []struct {
		name    string
		cancel  bool
		timeout time.Duration
		wantWhy string
	}{
		{"the ssh -G timeout", false, 3 * time.Second, "ssh -G github.com.emu timed out after 3s"},
		{"the caller's ctx", true, time.Minute, "ssh -G github.com.emu was interrupted (test stopped)"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			child := filepath.Join(t.TempDir(), "child")
			soFakeSSH(t, "sleep 30 &\necho \"$!\" > "+ShellWord(child)+"\nwait\n")
			ctx, cancel := context.WithCancelCause(context.Background())
			defer cancel(nil)
			if tc.cancel {
				go func() {
					soWaitFile(child)
					cancel(errors.New("test stopped"))
				}()
			}
			start := time.Now()
			host, why := resolveMergeAPIHost(ctx, t.TempDir(), realMergeDetachedExec, tc.timeout, "git@github.com.emu:o/r.git", soEmu)
			elapsed := time.Since(start)
			if host != soEmu || why != tc.wantWhy {
				t.Errorf("resolveMergeAPIHost = %q, %q; want %q, %q", host, why, soEmu, tc.wantWhy)
			}
			if bound := min(tc.timeout, 10*time.Second) + 5*time.Second; elapsed > bound {
				t.Errorf("resolveMergeAPIHost returned after %s; want within %s", elapsed, bound)
			}
			line := soWaitFile(child)
			if line == "" && !tc.cancel {
				// The timeout killed ssh's group before the fake ssh got to start
				// its child (seen under a load average of about 100): the call
				// still returned as it should, but no child was there to check.
				t.Skipf("the fake ssh wrote no child's pid within the %s timeout (a loaded machine), so whether the "+
					"timeout kills the child too was not checked in this run", tc.timeout)
			}
			var pid int
			if _, err := fmt.Sscan(line, &pid); err != nil || pid <= 0 {
				t.Fatalf("the fake ssh wrote %q as its child's pid (%v); want a pid", line, err)
			}
			soAssertGone(t, pid)
		})
	}
}

// TestReadMergeOrigin_RunsSSHInAProcessGroupOfItsOwnByDefault covers AC5b
// for the callers that pass no ssh runner (ralph org cleanup): ReadMergeOrigin
// with sshRun nil runs ssh -G through realMergeDetachedExec, in a process
// group of its own led by ssh, not the test's, and reads its answer. The fake
// ssh writes its pid and process group, then prints the alias's hostname.
func TestReadMergeOrigin_RunsSSHInAProcessGroupOfItsOwnByDefault(t *testing.T) {
	group := filepath.Join(t.TempDir(), "group")
	soFakeSSH(t, "echo \"$$ $(ps -o pgid= -p $$)\" > "+ShellWord(group)+"\necho 'hostname github.com'\n")
	git := &moExec{origin: "git@github.com.emu:o/r.git"}
	o, problem, err := ReadMergeOrigin(context.Background(), t.TempDir(), git.run, nil, time.Second)
	if problem != MergeOriginRead || err != nil || o.Arg() != "github.com/o/r" || o.unresolvedWhy != "" {
		t.Fatalf("ReadMergeOrigin = %+v, %d, %v; want github.com/o/r resolved by the fake ssh", o, problem, err)
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

// TestReadMergeOrigin_RealSSHGetsOriginsHostAsWritten covers cycle 2 of
// docs/reports/cross-review-triage-org-ssh-alias-host.md through
// realMergeDetachedExec: the ssh on the command line gets origin's host as
// its URL writes it, Work-GH, not work-gh. The fake ssh writes its arguments
// and, as OpenSSH matches `Host Work-GH` with case, names github.com only for
// Work-GH and gives any other host back, so a lowercased host would read
// work-gh/o/r.
func TestReadMergeOrigin_RealSSHGetsOriginsHostAsWritten(t *testing.T) {
	args := filepath.Join(t.TempDir(), "args")
	soFakeSSH(t, "printf '%s\\n' \"$*\" > "+ShellWord(args)+"\ncase \"$2\" in\nWork-GH) echo 'hostname github.com' ;;\n"+
		"*) echo \"hostname $2\" ;;\nesac\n")
	git := &moExec{origin: "git@Work-GH:o/r.git"}
	o, problem, err := ReadMergeOrigin(context.Background(), t.TempDir(), git.run, nil, time.Second)
	if problem != MergeOriginRead || err != nil || o.Arg() != "github.com/o/r" || o.originHost != "work-gh" || o.unresolvedWhy != "" {
		t.Fatalf("ReadMergeOrigin = %+v, %d, %v; want github.com/o/r resolved from Work-GH", o, problem, err)
	}
	if b, err := os.ReadFile(args); err != nil || string(b) != "-G Work-GH\n" {
		t.Errorf("the fake ssh got %q (%v); want -G Work-GH", b, err)
	}
}
