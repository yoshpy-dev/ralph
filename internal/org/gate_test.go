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
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/yoshpy-dev/ralph/internal/config"
)

// `ralph org gate` at the org layer (plan
// docs/plans/active/2026-10-13-org-gate.md, AC1 to AC7 and the env half of
// AC11): Gate on real git repositories, with stub gate scripts that report to
// a directory outside every repository. The run with the shipped
// run-verify.sh is in gate_runverify_test.go.

// gateTestClock is the fixtures' fixed clock, so a gate log of org-a is
// gateTestLog.
var gateTestClock = time.Date(2026, 10, 13, 12, 0, 0, 0, time.UTC)

const gateTestLog = "org-a-20261013T120000Z.log"

// gateStub is a stub gate script (fmt verb: its name). It appends its name
// to $GATE_TEST_OUT/calls, saves its environment and working directory
// there, prints 25 numbered lines and one line on stderr, sources
// $GATE_TEST_OUT/hook-<name> when it exists, and exits with the number in
// $GATE_TEST_OUT/exit-<name> (0 when there is none).
const gateStub = `#!/bin/sh
name=%s
printf '%%s\n' "$name" >> "$GATE_TEST_OUT/calls"
env > "$GATE_TEST_OUT/env-$name"
pwd -P > "$GATE_TEST_OUT/pwd-$name"
i=1
while [ "$i" -le 25 ]; do
  echo "$name line $i"
  i=$((i + 1))
done
echo "$name to stderr" >&2
if [ -f "$GATE_TEST_OUT/hook-$name" ]; then
  . "$GATE_TEST_OUT/hook-$name"
fi
exit "$(cat "$GATE_TEST_OUT/exit-$name" 2>/dev/null || echo 0)"
`

// gateStubTail is what GateResult.Tail holds when the stub called name fails
// with no hook: its last 19 numbered lines and its stderr line.
func gateStubTail(name string) []string {
	var tail []string
	for i := 7; i <= 25; i++ {
		tail = append(tail, fmt.Sprintf("%s line %d", name, i))
	}
	return append(tail, name+" to stderr")
}

// gateFixture is org-a bound in per-plan mode to its feature worktree on
// feat/auth, with its leader spawned. The repository's one commit, on main,
// has a .gitignore for .harness/, docs/evidence/*.log and ignored/, a
// README.md, and what populate writes (writeGateStubs by default). It has an
// origin remote that is never contacted: the gate reads only local refs, so
// refs/remotes/origin/main is set to main and refs/remotes/origin/HEAD
// points to it, as a clone leaves them. With linked set, the repository is
// the main checkout (main, on main) and the feature worktree a linked
// worktree of it made from main, as scripts/ralph-worktree.sh makes one;
// otherwise the repository itself is the feature worktree, switched to
// feat/auth, and main is empty (one worktree add less per test). The stubs
// report to out, outside every repository, and env is the caller
// environment Gate gets: the test process's without USER and HERDR_PANE_ID,
// then USER=alice and GATE_TEST_OUT=<out>.
type gateFixture struct {
	o        *Org
	stateDir string
	main     string
	worktree string
	f        *FeatureBinding
	out      string
	env      []string
}

// gateFixtureOptions are newGateFixtureWith's choices (see gateFixture).
type gateFixtureOptions struct {
	linked   bool
	populate func(t *testing.T, dir string) // nil: writeGateStubs
}

func newGateFixture(t *testing.T) *gateFixture {
	t.Helper()
	return newGateFixtureWith(t, gateFixtureOptions{})
}

func newGateFixtureWith(t *testing.T, opts gateFixtureOptions) *gateFixture {
	t.Helper()
	isolateGitEnv(t)
	tmp := resolved(t, t.TempDir())
	repo := filepath.Join(tmp, "repo")
	runGit(t, "init", "--quiet", "-b", "main", repo)
	writeGateFile(t, repo, ".gitignore", ".harness/\ndocs/evidence/*.log\nignored/\n", 0o644)
	writeGateFile(t, repo, "README.md", "gate fixture\n", 0o644)
	if opts.populate == nil {
		opts.populate = writeGateStubs
	}
	opts.populate(t, repo)
	runGit(t, "-C", repo, "add", ".")
	gateCommit(t, repo, "init")
	runGit(t, "-C", repo, "remote", "add", "origin", filepath.Join(tmp, "origin.git"))
	runGit(t, "-C", repo, "update-ref", "refs/remotes/origin/main", "main")
	runGit(t, "-C", repo, "symbolic-ref", "refs/remotes/origin/HEAD", "refs/remotes/origin/main")

	fx := &gateFixture{stateDir: filepath.Join(tmp, "state"), worktree: repo, out: filepath.Join(tmp, "out")}
	if opts.linked {
		fx.main, fx.worktree = repo, filepath.Join(tmp, "wt-auth")
		runGit(t, "-C", repo, "worktree", "add", "--quiet", "-b", "feat/auth", fx.worktree)
	} else {
		runGit(t, "-C", repo, "switch", "--quiet", "-c", "feat/auth")
	}
	mkdirAll(t, fx.out)

	fx.f = testFeature("split-1", "auth", "0123456789ab")
	fx.f.Worktree = fx.worktree
	fx.o = &Org{Manifest: NewManifestStoreAtPath(ManifestPathIn(fx.stateDir)), Now: func() time.Time { return gateTestClock }}
	if err := fx.o.Manifest.Append(boundReserveEvent("org-a", fx.f, "internal/auth/")); err != nil {
		t.Fatal(err)
	}
	seedSeat(t, fx.o, "org-a", LeaderIdentity, LeaderIdentity, "pane-1")

	for _, kv := range os.Environ() {
		if name, _, _ := strings.Cut(kv, "="); name != "USER" && name != herdrPaneIDEnv && name != "GATE_TEST_OUT" {
			fx.env = append(fx.env, kv)
		}
	}
	fx.env = append(fx.env, "USER=alice", "GATE_TEST_OUT="+fx.out)
	return fx
}

// writeGateStubs writes the two stub gate scripts (gateStub) in dir.
func writeGateStubs(t *testing.T, dir string) {
	t.Helper()
	writeGateFile(t, dir, gateStaticScript, fmt.Sprintf(gateStub, "static"), 0o755)
	writeGateFile(t, dir, gateTestScript, fmt.Sprintf(gateStub, "test"), 0o755)
}

// writeGateFile writes content at rel in dir with mode, making the parents.
func writeGateFile(t *testing.T, dir, rel, content string, mode os.FileMode) {
	t.Helper()
	path := filepath.Join(dir, filepath.FromSlash(rel))
	mkdirAll(t, filepath.Dir(path))
	if err := os.WriteFile(path, []byte(content), mode); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(path, mode); err != nil {
		t.Fatal(err)
	}
}

// gateCommit commits what is staged in dir, or an empty commit.
func gateCommit(t *testing.T, dir, msg string) {
	t.Helper()
	runGit(t, "-C", dir, "-c", "user.name=t", "-c", "user.email=t@example.com", "-c", "commit.gpgsign=false",
		"commit", "--quiet", "--allow-empty", "-m", msg)
}

// advance points ref, in the fixture's repository, to a new empty commit on
// top of its tip, without touching any checkout: the default branch moving
// on after the feature branch left it.
func (fx *gateFixture) advance(t *testing.T, ref string) {
	t.Helper()
	sha := gitOut(t, fx.worktree, "-c", "user.name=t", "-c", "user.email=t@example.com", "commit-tree", "-p", ref, "-m", "moves on", ref+"^{tree}")
	runGit(t, "-C", fx.worktree, "update-ref", ref, sha)
}

// gitOut runs git -C dir args and returns its stdout, trimmed.
func gitOut(t *testing.T, dir string, args ...string) string {
	t.Helper()
	out, err := exec.Command("git", append([]string{"-C", dir}, args...)...).Output()
	if err != nil {
		t.Fatalf("git -C %s %v: %v", dir, args, err)
	}
	return strings.TrimSpace(string(out))
}

// run is Gate of orgID from dir with the fixture's environment.
func (fx *gateFixture) run(ctx context.Context, orgID, dir string) (GateResult, error) {
	return fx.o.Gate(ctx, GateParams{OrgID: orgID, Dir: dir, Env: fx.env})
}

// gate is run of org-a from the worktree, failing the test on an error.
func (fx *gateFixture) gate(t *testing.T) GateResult {
	t.Helper()
	res, err := fx.run(context.Background(), "org-a", fx.worktree)
	if err != nil {
		t.Fatalf("gate: %v", err)
	}
	return res
}

// setExit makes the stub called name exit with code.
func (fx *gateFixture) setExit(t *testing.T, name string, code int) {
	t.Helper()
	writeGateFile(t, fx.out, "exit-"+name, strconv.Itoa(code)+"\n", 0o644)
}

// hook makes the stub called name source body before it exits.
func (fx *gateFixture) hook(t *testing.T, name, body string) {
	t.Helper()
	writeGateFile(t, fx.out, "hook-"+name, body, 0o644)
}

// calls returns the stubs that ran, in order: nil when none did.
func (fx *gateFixture) calls(t *testing.T) []string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(fx.out, "calls"))
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		t.Fatal(err)
	}
	return strings.Fields(string(data))
}

// gateEvents returns the manifest's gate_passed and gate_failed events.
func (fx *gateFixture) gateEvents(t *testing.T) []ManifestEvent {
	t.Helper()
	var out []ManifestEvent
	for _, ev := range mustReadEvents(t, fx.o) {
		if ev.Event == EventGatePassed || ev.Event == EventGateFailed {
			out = append(out, ev)
		}
	}
	return out
}

// head returns the worktree's HEAD and the tip of refs/remotes/origin/main.
func (fx *gateFixture) head(t *testing.T) (head, base string) {
	t.Helper()
	return gitOut(t, fx.worktree, "rev-parse", "HEAD"), gitOut(t, fx.worktree, "rev-parse", "refs/remotes/origin/main")
}

// assertGateRefused checks a refusal before the scripts: no error, Refused
// starting "org: gate: " with every want and no notWant, every other field
// empty, the manifest as before, no gates dir, and no stub run.
func (fx *gateFixture) assertGateRefused(t *testing.T, res GateResult, err error, before string, want, notWant []string) {
	t.Helper()
	if err != nil {
		t.Fatalf("gate = %v; want a refusal, not an error", err)
	}
	if !strings.HasPrefix(res.Refused, "org: gate: ") {
		t.Fatalf("gate = %+v; want a refusal starting \"org: gate: \"", res)
	}
	for _, w := range want {
		if !strings.Contains(res.Refused, w) {
			t.Errorf("refusal %q does not say %q", res.Refused, w)
		}
	}
	for _, w := range notWant {
		if strings.Contains(res.Refused, w) {
			t.Errorf("refusal %q says %q", res.Refused, w)
		}
	}
	if !reflect.DeepEqual(res, GateResult{Refused: res.Refused}) {
		t.Errorf("refused gate = %+v; want only Refused set", res)
	}
	if after := fileOrAbsent(t, fx.o.Manifest.Path()); after != before {
		t.Errorf("a refused gate changed the manifest:\nbefore %s\nafter  %s", before, after)
	}
	if _, err := os.Stat(filepath.Join(fx.stateDir, gateLogDir)); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("a refused gate made %s (stat: %v); want no log", gateLogDir, err)
	}
	if calls := fx.calls(t); calls != nil {
		t.Errorf("a refused gate ran %v; want no script run", calls)
	}
}

// TestGate_PassRecordsGatePassed covers AC1: in a clean worktree of a running
// feature org whose HEAD contains origin/main, with both scripts passing,
// Gate runs the static script and then the test script in the worktree's
// root, records one org-level gate_passed with the head, the base, the base
// ref, the branch, who ran it and the log, and writes the log under
// <state-dir>/gates/ and nothing in the worktree, not even an ignored file.
func TestGate_PassRecordsGatePassed(t *testing.T) {
	fx := newGateFixtureWith(t, gateFixtureOptions{linked: true})
	head, base := fx.head(t)
	res := fx.gate(t)

	logPath := filepath.Join(fx.stateDir, gateLogDir, gateTestLog)
	want := GateResult{Passed: true, Head: head, Base: base, BaseRef: "refs/remotes/origin/main", Branch: "feat/auth", LogPath: logPath}
	if !reflect.DeepEqual(res, want) {
		t.Fatalf("gate = %+v\nwant   %+v", res, want)
	}
	if calls := fx.calls(t); !slices.Equal(calls, []string{"static", "test"}) {
		t.Fatalf("scripts run = %v; want static then test", calls)
	}
	for _, name := range []string{"static", "test"} {
		if pwd := strings.TrimSpace(fileOrAbsent(t, filepath.Join(fx.out, "pwd-"+name))); pwd != fx.worktree {
			t.Errorf("%s ran in %s; want the worktree root %s", name, pwd, fx.worktree)
		}
	}

	events := fx.gateEvents(t)
	if len(events) != 1 {
		t.Fatalf("gate events = %+v; want one", events)
	}
	wantDetails := "head=" + head + " base=" + base + " base_ref=refs/remotes/origin/main branch=feat/auth by=alice herdr_pane=unset log=" +
		strconv.Quote(logPath)
	if ev := events[0]; ev.Event != EventGatePassed || ev.OrgID != "org-a" || ev.SeatID != "" || ev.TS != "2026-10-13T12:00:00Z" ||
		ev.Details != wantDetails || ev.DryRun {
		t.Fatalf("gate event = %+v\nwant gate_passed of org-a with Details %q", ev, wantDetails)
	}
	if last := lastEvent(t, fx.o); last.Event != EventGatePassed {
		t.Fatalf("last event = %+v; want the gate_passed", last)
	}

	log := fileOrAbsent(t, logPath)
	for _, w := range []string{
		"# head: " + head, "# base: refs/remotes/origin/main " + base,
		"==> ralph org gate: running ./scripts/run-static-verify.sh\n", "static line 1\n", "static to stderr\n",
		"==> ralph org gate: ./scripts/run-static-verify.sh exited 0\n",
		"==> ralph org gate: running ./scripts/run-test.sh\n", "test line 25\n", "==> ralph org gate: ./scripts/run-test.sh exited 0\n",
	} {
		if !strings.Contains(log, w) {
			t.Errorf("log does not have %q:\n%s", w, log)
		}
	}
	if st := gitOut(t, fx.worktree, "status", "--porcelain", "--untracked-files=all", "--ignored"); st != "" {
		t.Fatalf("the gate wrote in the worktree:\n%s", st)
	}
}

// TestGate_PassesFromASubdirectoryAndWithIgnoredFiles covers the two
// passing edges of AC2: run from a directory under the worktree, the gate
// checks and runs in the worktree's root, and ignored files, there before the
// gate or written by a script, do not count as a dirty worktree.
func TestGate_PassesFromASubdirectoryAndWithIgnoredFiles(t *testing.T) {
	fx := newGateFixture(t)
	writeGateFile(t, fx.worktree, "ignored/before.txt", "x\n", 0o644)
	writeGateFile(t, fx.worktree, ".harness/state/x", "x\n", 0o644)
	fx.hook(t, "test", "mkdir -p ignored docs/evidence && echo y > ignored/during.txt && echo z > docs/evidence/verify.log\n")
	res, err := fx.run(context.Background(), "org-a", filepath.Join(fx.worktree, "scripts"))
	if err != nil || !res.Passed || res.Refused != "" {
		t.Fatalf("gate from scripts/ with ignored files = %+v, %v; want a pass", res, err)
	}
	if pwd := strings.TrimSpace(fileOrAbsent(t, filepath.Join(fx.out, "pwd-static"))); pwd != fx.worktree {
		t.Fatalf("static ran in %s; want the worktree root %s", pwd, fx.worktree)
	}
	if events := fx.gateEvents(t); len(events) != 1 || events[0].Event != EventGatePassed {
		t.Fatalf("gate events = %+v; want one gate_passed", events)
	}
}

// TestGate_RefusedBeforeTheScripts covers AC2: each check before the scripts
// refuses with its reason and one next step, runs no script, records
// nothing, and writes no log.
func TestGate_RefusedBeforeTheScripts(t *testing.T) {
	appendEvent := func(t *testing.T, fx *gateFixture, ev ManifestEvent) {
		t.Helper()
		if err := fx.o.Manifest.Append(ev); err != nil {
			t.Fatal(err)
		}
	}
	cases := []struct {
		name    string
		orgID   string
		linked  bool
		dir     func(fx *gateFixture) string
		setup   func(t *testing.T, fx *gateFixture)
		want    func(t *testing.T, fx *gateFixture) []string
		notWant []string
	}{
		{name: "an invalid org_id", orgID: "Org_A",
			want: func(*testing.T, *gateFixture) []string { return []string{`invalid org_id "Org_A"`} }},
		{name: "an org the ledger does not have", orgID: "org-z",
			want: func(*testing.T, *gateFixture) []string {
				return []string{`org_id "org-z" is not running and holds no split plan feature binding`, "ralph org status lists them"}
			}},
		{name: "an org started without --plan", orgID: "org-b",
			setup: func(t *testing.T, fx *gateFixture) {
				seedSeat(t, fx.o, "org-b", LeaderIdentity, LeaderIdentity, "pane-2")
			},
			want: func(*testing.T, *gateFixture) []string {
				return []string{`org_id "org-b" holds no split plan feature binding`, "run ./scripts/run-static-verify.sh and ./scripts/run-test.sh yourself"}
			}},
		{name: "an incomplete binding", orgID: "org-c",
			setup: func(t *testing.T, fx *gateFixture) {
				f := *fx.f
				f.Digest = ""
				appendEvent(t, fx, boundReserveEvent("org-c", &f, "internal/c/"))
			},
			want: func(*testing.T, *gateFixture) []string {
				return []string{`org_id "org-c" is bound to`, "read from an incomplete record", "disband it and start it again"}
			}},
		{name: "a disbanded org",
			setup: func(t *testing.T, fx *gateFixture) {
				appendEvent(t, fx, ManifestEvent{TS: fx.o.now(), OrgID: "org-a", Event: EventDisbanded})
			},
			want: func(*testing.T, *gateFixture) []string {
				return []string{`org_id "org-a" was disbanded and has not started again`, "start it with ralph org start --plan"}
			}},
		{name: "a feature worktree that is gone", orgID: "org-d",
			setup: func(t *testing.T, fx *gateFixture) {
				f := *fx.f
				f.Feature, f.Worktree = "docs", filepath.Join(fx.out, "gone")
				appendEvent(t, fx, boundReserveEvent("org-d", &f, "docs/"))
			},
			want: func(_ *testing.T, fx *gateFixture) []string {
				return []string{`org_id "org-d"'s feature worktree ` + filepath.Join(fx.out, "gone") + " cannot be read", "restore it"}
			}},
		{name: "run from the main checkout", linked: true, dir: func(fx *gateFixture) string { return fx.main },
			want: func(_ *testing.T, fx *gateFixture) []string {
				return []string{"the gate ran in the git worktree " + fx.main + `, not in org_id "org-a"'s feature worktree ` + fx.worktree,
					"cd into " + fx.worktree}
			}},
		{name: "run outside git", dir: func(fx *gateFixture) string { return fx.out },
			want: func(_ *testing.T, fx *gateFixture) []string {
				return []string{fx.out + " is not in a git worktree", "cd into org_id \"org-a\"'s feature worktree " + fx.worktree}
			}},
		{name: "another branch checked out",
			setup: func(t *testing.T, fx *gateFixture) { runGit(t, "-C", fx.worktree, "switch", "--quiet", "-c", "other") },
			want: func(_ *testing.T, fx *gateFixture) []string {
				return []string{fx.worktree + " has the branch other checked out, not the feature branch feat/auth", "run git switch feat/auth"}
			}},
		{name: "a detached HEAD",
			setup: func(t *testing.T, fx *gateFixture) { runGit(t, "-C", fx.worktree, "switch", "--quiet", "--detach") },
			want: func(_ *testing.T, fx *gateFixture) []string {
				return []string{"HEAD is detached in " + fx.worktree + ", not on the feature branch feat/auth", "run git switch feat/auth"}
			}},
		{name: "an untracked file",
			setup: func(t *testing.T, fx *gateFixture) { writeGateFile(t, fx.worktree, "new.txt", "x\n", 0o644) },
			want: func(_ *testing.T, fx *gateFixture) []string {
				return []string{"the feature worktree " + fx.worktree + " has 1 uncommitted change(s) or untracked file(s)", "commit or remove them",
					":\n  ?? new.txt"}
			}},
		{name: "a modified file",
			setup: func(t *testing.T, fx *gateFixture) { writeGateFile(t, fx.worktree, "README.md", "changed\n", 0o644) },
			want:  func(*testing.T, *gateFixture) []string { return []string{"has 1 uncommitted", "\n   M README.md"} }},
		{name: "a staged file",
			setup: func(t *testing.T, fx *gateFixture) {
				writeGateFile(t, fx.worktree, "staged.txt", "x\n", 0o644)
				runGit(t, "-C", fx.worktree, "add", "staged.txt")
			},
			want: func(*testing.T, *gateFixture) []string { return []string{"has 1 uncommitted", "\n  A  staged.txt"} }},
		{name: "many untracked files",
			setup: func(t *testing.T, fx *gateFixture) {
				for i := range 25 {
					writeGateFile(t, fx.worktree, fmt.Sprintf("new/%02d.txt", i), "x\n", 0o644)
				}
			},
			want: func(*testing.T, *gateFixture) []string {
				return []string{"has 25 uncommitted", "?? new/19.txt\n  ... and 5 more"}
			},
			notWant: []string{"new/20.txt"}},
		{name: "HEAD does not contain origin/main",
			setup: func(t *testing.T, fx *gateFixture) { fx.advance(t, "refs/remotes/origin/main") },
			want: func(t *testing.T, fx *gateFixture) []string {
				head, base := fx.head(t)
				return []string{"HEAD " + head + " of feat/auth does not contain refs/remotes/origin/main's tip " + base,
					"run `git fetch origin main` and `git rebase refs/remotes/origin/main`, then run the gate again"}
			}},
		{name: "HEAD does not contain local main, with no origin",
			setup: func(t *testing.T, fx *gateFixture) {
				runGit(t, "-C", fx.worktree, "remote", "remove", "origin")
				fx.advance(t, "refs/heads/main")
			},
			want: func(*testing.T, *gateFixture) []string {
				return []string{"does not contain refs/heads/main's tip", "run `git rebase refs/heads/main`, then run the gate again"}
			},
			notWant: []string{"git fetch"}},
		{name: "no default branch",
			setup: func(t *testing.T, fx *gateFixture) {
				runGit(t, "-C", fx.worktree, "remote", "remove", "origin")
				runGit(t, "-C", fx.worktree, "branch", "--quiet", "-D", "main")
			},
			want: func(_ *testing.T, fx *gateFixture) []string {
				return []string{"no default branch found in " + fx.worktree, "neither main nor master exists", "run git fetch origin"}
			}},
		{name: "the static script is missing",
			setup: func(t *testing.T, fx *gateFixture) {
				runGit(t, "-C", fx.worktree, "rm", "--quiet", gateStaticScript)
				gateCommit(t, fx.worktree, "drop the static script")
			},
			want: func(_ *testing.T, fx *gateFixture) []string {
				return []string{filepath.Join(fx.worktree, "scripts", "run-static-verify.sh") + " does not exist", "add it, commit it"}
			}},
		{name: "the test script is missing",
			setup: func(t *testing.T, fx *gateFixture) {
				runGit(t, "-C", fx.worktree, "rm", "--quiet", gateTestScript)
				gateCommit(t, fx.worktree, "drop the test script")
			},
			want: func(_ *testing.T, fx *gateFixture) []string {
				return []string{filepath.Join(fx.worktree, "scripts", "run-test.sh") + " does not exist"}
			}},
		{name: "the static script is not executable",
			setup: func(t *testing.T, fx *gateFixture) {
				if err := os.Chmod(filepath.Join(fx.worktree, filepath.FromSlash(gateStaticScript)), 0o644); err != nil {
					t.Fatal(err)
				}
				runGit(t, "-C", fx.worktree, "add", gateStaticScript)
				gateCommit(t, fx.worktree, "the static script loses its x bit")
			},
			want: func(_ *testing.T, fx *gateFixture) []string {
				return []string{filepath.Join(fx.worktree, "scripts", "run-static-verify.sh") + " is not executable",
					"run chmod +x scripts/run-static-verify.sh"}
			}},
		{name: "the test script is a directory",
			setup: func(t *testing.T, fx *gateFixture) {
				runGit(t, "-C", fx.worktree, "rm", "--quiet", gateTestScript)
				writeGateFile(t, fx.worktree, gateTestScript+"/keep", "x\n", 0o644)
				runGit(t, "-C", fx.worktree, "add", ".")
				gateCommit(t, fx.worktree, "the test script becomes a directory")
			},
			want: func(_ *testing.T, fx *gateFixture) []string {
				return []string{filepath.Join(fx.worktree, "scripts", "run-test.sh") + " is not a regular file"}
			}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			fx := newGateFixtureWith(t, gateFixtureOptions{linked: tc.linked})
			if tc.setup != nil {
				tc.setup(t, fx)
			}
			orgID, dir := "org-a", fx.worktree
			if tc.orgID != "" {
				orgID = tc.orgID
			}
			if tc.dir != nil {
				dir = tc.dir(fx)
			}
			want := tc.want(t, fx)
			before := fileOrAbsent(t, fx.o.Manifest.Path())
			res, err := fx.run(context.Background(), orgID, dir)
			fx.assertGateRefused(t, res, err, before, want, tc.notWant)
		})
	}
}

// TestGate_ScriptFailuresAndChangesRecordGateFailed covers AC3: a failing
// static script records static_failed and runs no test script, a failing test
// script records test_failed, both with the failing script's last 20 output
// lines in Tail; a commit during the scripts records head_moved with the head
// from before, and a file left in the worktree records tree_changed with the
// porcelain lines in Dirty. Each writes one gate_failed whose Details start
// with the reason and refuses nothing.
func TestGate_ScriptFailuresAndChangesRecordGateFailed(t *testing.T) {
	cases := []struct {
		name      string
		setup     func(t *testing.T, fx *gateFixture)
		reason    string
		calls     []string
		tail      []string
		dirty     []string
		logExited string
	}{
		{name: "the static script fails", setup: func(t *testing.T, fx *gateFixture) { fx.setExit(t, "static", 3) },
			reason: GateReasonStaticFailed, calls: []string{"static"}, tail: gateStubTail("static"),
			logExited: "./scripts/run-static-verify.sh exited 3"},
		{name: "the test script fails", setup: func(t *testing.T, fx *gateFixture) { fx.setExit(t, "test", 1) },
			reason: GateReasonTestFailed, calls: []string{"static", "test"}, tail: gateStubTail("test"),
			logExited: "./scripts/run-test.sh exited 1"},
		{name: "HEAD moves during the scripts", setup: func(t *testing.T, fx *gateFixture) {
			fx.hook(t, "test", "git -c user.name=t -c user.email=t@example.com -c commit.gpgsign=false commit --quiet --allow-empty -m moved\n")
		}, reason: GateReasonHeadMoved, calls: []string{"static", "test"}, logExited: "./scripts/run-test.sh exited 0"},
		{name: "a script leaves a file in the worktree", setup: func(t *testing.T, fx *gateFixture) {
			fx.hook(t, "static", "echo x > untracked.txt\n")
		}, reason: GateReasonTreeChanged, calls: []string{"static", "test"}, dirty: []string{"?? untracked.txt"},
			logExited: "./scripts/run-test.sh exited 0"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			fx := newGateFixture(t)
			tc.setup(t, fx)
			head, base := fx.head(t)
			res := fx.gate(t)
			logPath := filepath.Join(fx.stateDir, gateLogDir, gateTestLog)
			want := GateResult{Reason: tc.reason, Head: head, Base: base, BaseRef: "refs/remotes/origin/main", Branch: "feat/auth",
				LogPath: logPath, Tail: tc.tail, Dirty: tc.dirty}
			if !reflect.DeepEqual(res, want) {
				t.Fatalf("gate = %#v\nwant   %#v", res, want)
			}
			if calls := fx.calls(t); !slices.Equal(calls, tc.calls) {
				t.Fatalf("scripts run = %v; want %v", calls, tc.calls)
			}
			events := fx.gateEvents(t)
			wantDetails := "reason=" + tc.reason + " head=" + head + " base=" + base + " base_ref=refs/remotes/origin/main branch=feat/auth " +
				"by=alice herdr_pane=unset log=" + strconv.Quote(logPath)
			if len(events) != 1 || events[0].Event != EventGateFailed || events[0].SeatID != "" || events[0].Details != wantDetails {
				t.Fatalf("gate events = %+v\nwant one gate_failed with Details %q", events, wantDetails)
			}
			if log := fileOrAbsent(t, logPath); !strings.Contains(log, "==> ralph org gate: "+tc.logExited+"\n") {
				t.Fatalf("log does not say %q:\n%s", tc.logExited, log)
			}
		})
	}
}

// TestGate_PassThenFailOnTheSameHead covers B7's edge: a pass and then a
// failure on the same head are both recorded, in that order, each with its
// own log (the second gate in the same second gets the -2 suffix).
func TestGate_PassThenFailOnTheSameHead(t *testing.T) {
	fx := newGateFixture(t)
	if res := fx.gate(t); !res.Passed {
		t.Fatalf("first gate = %+v; want a pass", res)
	}
	fx.setExit(t, "test", 1)
	res := fx.gate(t)
	if res.Passed || res.Reason != GateReasonTestFailed || res.LogPath != filepath.Join(fx.stateDir, gateLogDir, "org-a-20261013T120000Z-2.log") {
		t.Fatalf("second gate = %+v; want test_failed with the -2 log", res)
	}
	events := fx.gateEvents(t)
	if len(events) != 2 || events[0].Event != EventGatePassed || events[1].Event != EventGateFailed {
		t.Fatalf("gate events = %+v; want gate_passed then gate_failed", events)
	}
	first, ok1 := gateRecordFromDetails(events[0].Details)
	second, ok2 := gateRecordFromDetails(events[1].Details)
	if !ok1 || !ok2 || first.Head != second.Head || first.Log == second.Log {
		t.Fatalf("records = %+v (%v), %+v (%v); want the same head and two logs", first, ok1, second, ok2)
	}
}

// TestGate_BaseResolution covers AC4 through Gate: the default branch is the
// one refs/remotes/origin/HEAD names when it exists, else main, else master,
// and the base ref is refs/remotes/origin/<name> when that exists, else
// refs/heads/<name>; the recorded base is that ref's tip.
func TestGate_BaseResolution(t *testing.T) {
	// git runs git in the fixture's repository with args.
	git := func(t *testing.T, fx *gateFixture, args ...string) {
		t.Helper()
		runGit(t, append([]string{"-C", fx.worktree}, args...)...)
	}
	cases := []struct {
		name    string
		setup   func(t *testing.T, fx *gateFixture)
		branch  string
		baseRef string
	}{
		{"origin/HEAD names trunk", func(t *testing.T, fx *gateFixture) {
			git(t, fx, "update-ref", "refs/remotes/origin/trunk", "main")
			git(t, fx, "symbolic-ref", "refs/remotes/origin/HEAD", "refs/remotes/origin/trunk")
		}, "trunk", "refs/remotes/origin/trunk"},
		{"origin/HEAD names trunk and a local branch origin/trunk exists", func(t *testing.T, fx *gateFixture) {
			git(t, fx, "update-ref", "refs/remotes/origin/trunk", "main")
			git(t, fx, "symbolic-ref", "refs/remotes/origin/HEAD", "refs/remotes/origin/trunk")
			git(t, fx, "branch", "origin/trunk", "main")
		}, "trunk", "refs/remotes/origin/trunk"},
		{"origin/HEAD names a missing branch: main", func(t *testing.T, fx *gateFixture) {
			git(t, fx, "symbolic-ref", "refs/remotes/origin/HEAD", "refs/remotes/origin/gone")
		}, "main", "refs/remotes/origin/main"},
		{"no origin/HEAD, main and master on origin: main", func(t *testing.T, fx *gateFixture) {
			git(t, fx, "update-ref", "refs/remotes/origin/master", "main")
			git(t, fx, "symbolic-ref", "--delete", "refs/remotes/origin/HEAD")
		}, "main", "refs/remotes/origin/main"},
		{"only master, on origin", func(t *testing.T, fx *gateFixture) {
			git(t, fx, "update-ref", "refs/remotes/origin/master", "main")
			git(t, fx, "symbolic-ref", "--delete", "refs/remotes/origin/HEAD")
			git(t, fx, "update-ref", "-d", "refs/remotes/origin/main")
			git(t, fx, "branch", "--quiet", "-D", "main")
		}, "master", "refs/remotes/origin/master"},
		{"only a local master", func(t *testing.T, fx *gateFixture) {
			git(t, fx, "remote", "remove", "origin")
			git(t, fx, "branch", "--quiet", "-m", "main", "master")
		}, "master", "refs/heads/master"},
		{"main only locally", func(t *testing.T, fx *gateFixture) {
			git(t, fx, "symbolic-ref", "--delete", "refs/remotes/origin/HEAD")
			git(t, fx, "update-ref", "-d", "refs/remotes/origin/main")
		}, "main", "refs/heads/main"},
		{"no origin remote", func(t *testing.T, fx *gateFixture) {
			git(t, fx, "remote", "remove", "origin")
		}, "main", "refs/heads/main"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			fx := newGateFixture(t)
			tc.setup(t, fx)
			got, found, err := resolveGateBase(context.Background(), fx.worktree)
			wantSHA := gitOut(t, fx.worktree, "rev-parse", tc.baseRef)
			if want := (gateBase{branch: tc.branch, ref: tc.baseRef, sha: wantSHA}); err != nil || !found || got != want {
				t.Fatalf("resolveGateBase = %+v, %v, %v; want %+v", got, found, err, want)
			}
			res := fx.gate(t)
			if !res.Passed || res.BaseRef != tc.baseRef || res.Base != wantSHA {
				t.Fatalf("gate = %+v; want a pass on %s %s", res, tc.baseRef, wantSHA)
			}
			if events := fx.gateEvents(t); len(events) != 1 || !strings.Contains(events[0].Details, " base="+wantSHA+" base_ref="+tc.baseRef+" ") {
				t.Fatalf("gate events = %+v; want the base %s %s recorded", events, tc.baseRef, wantSHA)
			}
		})
	}
	t.Run("none", func(t *testing.T) {
		fx := newGateFixture(t)
		git(t, fx, "remote", "remove", "origin")
		git(t, fx, "branch", "--quiet", "-D", "main")
		if got, found, err := resolveGateBase(context.Background(), fx.worktree); err != nil || found || got != (gateBase{}) {
			t.Fatalf("resolveGateBase = %+v, %v, %v; want none found", got, found, err)
		}
	})
}

// TestGate_ManifestLockNotHeldWhileScriptsRun covers AC5's first half: while
// a script runs, another writer takes the manifest lock at once and appends,
// and the gate records after it when the script ends.
func TestGate_ManifestLockNotHeldWhileScriptsRun(t *testing.T) {
	fx := newGateFixture(t)
	started, release := filepath.Join(fx.out, "started"), filepath.Join(fx.out, "release")
	fx.hook(t, "static", "touch '"+started+"'\nwhile [ ! -f '"+release+"' ]; do sleep 0.05; done\n")

	type outcome struct {
		res GateResult
		err error
	}
	done := make(chan outcome, 1)
	go func() {
		res, err := fx.run(context.Background(), "org-a", fx.worktree)
		done <- outcome{res, err}
	}()
	waitForFile(t, started)

	begin := time.Now()
	err := withManifestLock(filepath.Dir(fx.o.Manifest.Path()), func() error {
		return fx.o.Manifest.Append(ManifestEvent{TS: fx.o.now(), OrgID: "org-a", SeatID: "implementer", Event: EventSent})
	})
	if took := time.Since(begin); err != nil || took > time.Second {
		t.Fatalf("taking the manifest lock while a gate script runs = %v after %s; want it at once", err, took)
	}
	writeGateFile(t, fx.out, "release", "", 0o644)
	got := <-done
	if got.err != nil || !got.res.Passed {
		t.Fatalf("gate = %+v, %v; want a pass", got.res, got.err)
	}
	events := mustReadEvents(t, fx.o)
	if n := len(events); n < 2 || events[n-2].Event != EventSent || events[n-1].Event != EventGatePassed {
		t.Fatalf("last events = %+v; want the sent appended during the script, then the gate_passed", events)
	}
}

// TestGate_InterruptStopsTheScriptsAndRecordsNothing covers AC5's second
// half: when ctx is cancelled while a script runs, the script and the child it
// started are stopped, also when both ignore SIGTERM (SIGKILL after the
// grace), the test script never runs, nothing is recorded, and the error
// says so.
func TestGate_InterruptStopsTheScriptsAndRecordsNothing(t *testing.T) {
	grace := gateKillGrace
	gateKillGrace = 300 * time.Millisecond
	t.Cleanup(func() { gateKillGrace = grace })
	for _, ignoreTerm := range []bool{false, true} {
		t.Run(fmt.Sprintf("ignore SIGTERM=%v", ignoreTerm), func(t *testing.T) {
			fx := newGateFixture(t)
			trap := ""
			if ignoreTerm {
				trap = "trap '' TERM\n"
			}
			fx.hook(t, "static", trap+"sleep 300 &\necho $! > \"$GATE_TEST_OUT/child\"\necho $$ > \"$GATE_TEST_OUT/pid\"\n"+
				"touch \"$GATE_TEST_OUT/started\"\nwait\n")
			before := fileOrAbsent(t, fx.o.Manifest.Path())
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			done := make(chan error, 1)
			go func() {
				_, err := fx.run(ctx, "org-a", fx.worktree)
				done <- err
			}()
			waitForFile(t, filepath.Join(fx.out, "started"))
			pids := []int{gatePID(t, fx, "pid"), gatePID(t, fx, "child")}
			begin := time.Now()
			cancel()
			var err error
			select {
			case err = <-done:
			case <-time.After(10 * time.Second):
				t.Fatal("gate did not return within 10s of the interrupt")
			}
			if took := time.Since(begin); took > 3*time.Second {
				t.Errorf("gate took %s to return after the interrupt", took)
			}
			logPath := filepath.Join(fx.stateDir, gateLogDir, gateTestLog)
			if err == nil || !strings.Contains(err.Error(), "org: gate: interrupted") ||
				!strings.Contains(err.Error(), "nothing was recorded; the output so far is in "+logPath) {
				t.Fatalf("gate = %v; want the interrupt error naming the log", err)
			}
			for _, pid := range pids {
				waitProcessGone(t, pid)
			}
			if after := fileOrAbsent(t, fx.o.Manifest.Path()); after != before {
				t.Fatalf("an interrupted gate changed the manifest:\nbefore %s\nafter  %s", before, after)
			}
			if calls := fx.calls(t); !slices.Equal(calls, []string{"static"}) {
				t.Fatalf("scripts run = %v; want only static", calls)
			}
			if log := fileOrAbsent(t, logPath); !strings.Contains(log, "./scripts/run-static-verify.sh interrupted") {
				t.Fatalf("log does not say the script was interrupted:\n%s", log)
			}
		})
	}
}

// TestGate_InterruptedBeforeTheScripts: a ctx done before the checks
// finish stops the gate with an error that says nothing was recorded, and no
// script runs and no log is written.
func TestGate_InterruptedBeforeTheScripts(t *testing.T) {
	fx := newGateFixture(t)
	before := fileOrAbsent(t, fx.o.Manifest.Path())
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	res, err := fx.run(ctx, "org-a", fx.worktree)
	if err == nil || err.Error() != "org: gate: interrupted (context canceled) before the scripts ran: nothing was recorded" ||
		!reflect.DeepEqual(res, GateResult{}) {
		t.Fatalf("gate = %+v, %v; want the interrupt error and nothing else", res, err)
	}
	if after := fileOrAbsent(t, fx.o.Manifest.Path()); after != before {
		t.Fatal("an interrupted gate changed the manifest")
	}
	if _, err := os.Stat(filepath.Join(fx.stateDir, gateLogDir)); !errors.Is(err, os.ErrNotExist) || fx.calls(t) != nil {
		t.Fatalf("an interrupted gate wrote a log (stat: %v) or ran %v", err, fx.calls(t))
	}
}

// waitForFile waits up to 10s for path to exist.
func waitForFile(t *testing.T, path string) {
	t.Helper()
	deadline := time.Now().Add(10 * time.Second)
	for {
		if _, err := os.Stat(path); err == nil {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("%s did not appear within 10s", path)
		}
		time.Sleep(10 * time.Millisecond)
	}
}

// gatePID reads the pid a stub wrote to $GATE_TEST_OUT/<name>.
func gatePID(t *testing.T, fx *gateFixture, name string) int {
	t.Helper()
	pid, err := strconv.Atoi(strings.TrimSpace(fileOrAbsent(t, filepath.Join(fx.out, name))))
	if err != nil {
		t.Fatalf("read pid %s: %v", name, err)
	}
	return pid
}

// waitProcessGone waits up to 5s for pid to be gone (kill(pid, 0) answers
// ESRCH), giving init the time to reap a reparented child.
func waitProcessGone(t *testing.T, pid int) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for {
		err := syscall.Kill(pid, 0)
		if errors.Is(err, syscall.ESRCH) {
			return
		}
		if time.Now().After(deadline) {
			_ = syscall.Kill(pid, syscall.SIGKILL)
			t.Fatalf("process %d is still there 5s after the interrupt (kill 0: %v)", pid, err)
		}
		time.Sleep(20 * time.Millisecond)
	}
}

// TestGate_OrgChangedWhileTheScriptsRan: when the org is disbanded, started
// again, or bound to another feature while the scripts run, Gate records
// nothing and refuses, keeping the log of the run.
func TestGate_OrgChangedWhileTheScriptsRan(t *testing.T) {
	line := func(t *testing.T, ev ManifestEvent) string {
		t.Helper()
		data, err := json.Marshal(ev)
		if err != nil {
			t.Fatal(err)
		}
		return string(data)
	}
	cases := []struct {
		name  string
		lines func(t *testing.T, fx *gateFixture) []string
		want  string
	}{
		{"disbanded", func(t *testing.T, fx *gateFixture) []string {
			return []string{line(t, ManifestEvent{TS: "2026-10-13T12:00:01Z", OrgID: "org-a", Event: EventDisbanded})}
		}, "(it was disbanded)"},
		{"disbanded and started again with the same binding", func(t *testing.T, fx *gateFixture) []string {
			return []string{
				line(t, ManifestEvent{TS: "2026-10-13T12:00:01Z", OrgID: "org-a", Event: EventDisbanded}),
				line(t, boundReserveEvent("org-a", fx.f, "internal/auth/")),
				line(t, ManifestEvent{TS: "2026-10-13T12:00:02Z", OrgID: "org-a", SeatID: LeaderIdentity, Event: EventSpawned, Role: LeaderIdentity, PaneID: "pane-7"}),
			}
		}, "(it was disbanded and started again)"},
		{"bound to another feature", func(t *testing.T, fx *gateFixture) []string {
			f := *fx.f
			f.Feature = "auth-v2"
			return []string{line(t, boundReserveEvent("org-a", &f, "internal/auth/"))}
		}, `(it is now bound to feature "auth-v2"`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			fx := newGateFixture(t)
			fx.hook(t, "test", "cat >> '"+fx.o.Manifest.Path()+"' <<'EOF'\n"+strings.Join(tc.lines(t, fx), "\n")+"\nEOF\n")
			res, err := fx.run(context.Background(), "org-a", fx.worktree)
			if err != nil || res.Passed || !strings.Contains(res.Refused, `org: gate: nothing was recorded, since org_id "org-a" changed while the scripts ran `+tc.want) {
				t.Fatalf("gate = %+v, %v; want the record refused with %q", res, err, tc.want)
			}
			if !strings.Contains(res.Refused, "check the org with ralph org status") {
				t.Errorf("refusal %q has no next step", res.Refused)
			}
			if events := fx.gateEvents(t); len(events) != 0 {
				t.Fatalf("gate events = %+v; want none", events)
			}
			if _, err := os.Stat(res.LogPath); res.LogPath == "" || err != nil {
				t.Fatalf("log %q: %v; want the log of the run kept", res.LogPath, err)
			}
		})
	}
}

// TestGate_LogDirCannotBeMade: a gates path that is a file is an error
// before any script runs, with nothing recorded.
func TestGate_LogDirCannotBeMade(t *testing.T) {
	fx := newGateFixture(t)
	writeGateFile(t, fx.stateDir, gateLogDir, "not a dir\n", 0o644)
	before := fileOrAbsent(t, fx.o.Manifest.Path())
	res, err := fx.run(context.Background(), "org-a", fx.worktree)
	if err == nil || !strings.Contains(err.Error(), "org: gate: create the gate log dir") || res.Refused != "" {
		t.Fatalf("gate = %+v, %v; want the log dir error", res, err)
	}
	if after := fileOrAbsent(t, fx.o.Manifest.Path()); after != before {
		t.Fatal("the gate changed the manifest")
	}
	if calls := fx.calls(t); calls != nil {
		t.Fatalf("scripts run = %v; want none", calls)
	}
}

// TestGate_ScriptEnvironmentIsFixed covers AC11's environment half: the
// scripts get the caller's environment without any RALPH_VERIFY_* or
// HARNESS_VERIFY_* variable and with RALPH_VERIFY_SCOPE=full, and keep the
// rest, a name that only contains RALPH_VERIFY_ included.
func TestGate_ScriptEnvironmentIsFixed(t *testing.T) {
	fx := newGateFixture(t)
	fx.env = append(fx.env, "RALPH_VERIFY_BASE=HEAD", "RALPH_VERIFY_SCOPE=changed", "RALPH_VERIFY_PROJECT_ROOTS=nowhere",
		"RALPH_VERIFY_SKIP_BRANCH_SECRET_SCAN=1", "HARNESS_VERIFY_MODE=all", "HARNESS_VERIFY_EXTRA=1", "KEEP_RALPH_VERIFY_BASE=kept")
	if res := fx.gate(t); !res.Passed {
		t.Fatalf("gate = %+v; want a pass", res)
	}
	for _, name := range []string{"static", "test"} {
		var verify []string
		vars := strings.Split(strings.TrimSpace(fileOrAbsent(t, filepath.Join(fx.out, "env-"+name))), "\n")
		for _, kv := range vars {
			if strings.HasPrefix(kv, "RALPH_VERIFY_") || strings.HasPrefix(kv, "HARNESS_VERIFY_") {
				verify = append(verify, kv)
			}
		}
		if !slices.Equal(verify, []string{"RALPH_VERIFY_SCOPE=full"}) {
			t.Errorf("%s got the verify variables %v; want only RALPH_VERIFY_SCOPE=full", name, verify)
		}
		for _, keep := range []string{"KEEP_RALPH_VERIFY_BASE=kept", "GATE_TEST_OUT=" + fx.out, "USER=alice"} {
			if !slices.Contains(vars, keep) {
				t.Errorf("%s did not get %s", name, keep)
			}
		}
	}
}

// TestGateScriptEnv pins gateScriptEnv on its own: the order of what is
// kept, the prefixes matched on the name only, and RALPH_VERIFY_SCOPE=full
// last.
func TestGateScriptEnv(t *testing.T) {
	got := gateScriptEnv([]string{"PATH=/bin", "RALPH_VERIFY_BASE=HEAD", "X=RALPH_VERIFY_BASE=1", "HARNESS_VERIFY_MODE=test",
		"RALPH_VERIFY_=x", "RALPH_VERIFYX=1", "NOEQUALS", "RALPH_VERIFY_SCOPE=changed"})
	want := []string{"PATH=/bin", "X=RALPH_VERIFY_BASE=1", "RALPH_VERIFYX=1", "NOEQUALS", "RALPH_VERIFY_SCOPE=full"}
	if !slices.Equal(got, want) {
		t.Fatalf("gateScriptEnv = %q; want %q", got, want)
	}
	if got := gateScriptEnv(nil); !slices.Equal(got, []string{"RALPH_VERIFY_SCOPE=full"}) {
		t.Fatalf("gateScriptEnv(nil) = %q", got)
	}
}

// TestGate_RecordsWhoRanIt: by= is USER (approverName: unknown when it is
// empty or has whitespace, the last USER entry winning), and herdr_pane= is
// set exactly when HERDR_PANE_ID is non-empty.
func TestGate_RecordsWhoRanIt(t *testing.T) {
	cases := []struct {
		name     string
		env      []string
		by, pane string
	}{
		{"USER and no pane", nil, "alice", "unset"},
		{"a herdr pane", []string{"HERDR_PANE_ID=pane-9"}, "alice", "set"},
		{"an empty pane id", []string{"HERDR_PANE_ID="}, "alice", "unset"},
		{"USER given again", []string{"USER=bob"}, "bob", "unset"},
		{"USER with a space", []string{"USER=bob smith"}, approverUnknown, "unset"},
		{"USER empty", []string{"USER="}, approverUnknown, "unset"},
	}
	fx := newGateFixture(t)
	env := fx.env
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			fx.env = append(slices.Clip(env), tc.env...)
			n := len(fx.gateEvents(t))
			fx.gate(t)
			events := fx.gateEvents(t)
			if len(events) != n+1 {
				t.Fatalf("gate events = %+v; want one more than %d", events, n)
			}
			last := events[n].Details
			r, ok := gateRecordFromDetails(last)
			if !ok || r.By != tc.by || r.HerdrPane != (tc.pane == "set") || !strings.Contains(last, " by="+tc.by+" herdr_pane="+tc.pane+" log=") {
				t.Fatalf("record %q read as %+v (%v); want by=%s herdr_pane=%s", last, r, ok, tc.by, tc.pane)
			}
		})
	}
}

// TestGate_NoWatchGateNoApprovalCheck covers AC7: an org bound in per-plan
// mode with no plan approved, on a ledger whose watch gate would refuse,
// still gates, and the watch gate is never called.
func TestGate_NoWatchGateNoApprovalCheck(t *testing.T) {
	fx := newGateFixture(t)
	watch := refusingWatchGate()
	fx.o.WatchGate = watch.gate
	if !fx.f.RequiresApproval() || fx.f.Approval != config.ApprovalModePerPlan {
		t.Fatalf("binding %+v; want per-plan", fx.f)
	}
	if approvals := currentPlanApprovals(mustReadEvents(t, fx.o), "org-a"); len(approvals) != 0 {
		t.Fatalf("approvals = %+v; want none", approvals)
	}
	if res := fx.gate(t); !res.Passed {
		t.Fatalf("gate = %+v; want a pass with no watch and no approval", res)
	}
	if watch.calls != 0 {
		t.Fatalf("the watch gate was called %d times; want none", watch.calls)
	}
}

// TestGateRecordDetails_RoundTrip covers the record's Details: every reason
// and a pass read back as written, with a log path holding spaces, quotes
// and " log=", and SHA-256 object names; event() names the event; and a
// record that cannot stand as a gate result does not read.
func TestGateRecordDetails_RoundTrip(t *testing.T) {
	sha1, sha1b := strings.Repeat("a", 40), strings.Repeat("b", 40)
	records := []gateRecord{
		{Head: sha1, Base: sha1b, BaseRef: "refs/remotes/origin/main", Branch: "feat/auth", By: "alice", Log: "/state/gates/org-a.log"},
		{Head: strings.Repeat("c", 64), Base: strings.Repeat("d", 64), BaseRef: "refs/heads/main", Branch: "fix/a=b", By: approverUnknown,
			HerdrPane: true, Log: `/my state/a "quoted" log=x.log`},
	}
	for _, reason := range []string{GateReasonStaticFailed, GateReasonTestFailed, GateReasonHeadMoved, GateReasonTreeChanged} {
		r := records[0]
		r.Reason = reason
		records = append(records, r)
	}
	for _, r := range records {
		got, ok := gateRecordFromDetails(r.details())
		if !ok || got != r {
			t.Errorf("details %q read back as %+v (%v); want %+v", r.details(), got, ok, r)
		}
		if want := map[bool]string{true: EventGatePassed, false: EventGateFailed}[r.Reason == ""]; r.event() != want {
			t.Errorf("event() of %+v = %s; want %s", r, r.event(), want)
		}
	}
	if d := records[2].details(); !strings.HasPrefix(d, "reason=static_failed head=") {
		t.Errorf("gate_failed Details %q do not start with the reason", d)
	}

	good := records[0].details()
	for _, tc := range []struct{ name, details string }{
		{"head given twice", "head=" + sha1 + " " + good},
		{"a token without =", "stray " + good},
		{"a short head", strings.Replace(good, "head="+sha1, "head=abc", 1)},
		{"an upper-case base", strings.Replace(good, "base="+sha1b, "base="+strings.ToUpper(sha1b), 1)},
		{"a base_ref outside refs/", strings.Replace(good, "base_ref=refs/remotes/origin/main", "base_ref=main", 1)},
		{"no branch", strings.Replace(good, " branch=feat/auth", "", 1)},
		{"an unknown reason", "reason=flaky " + good},
		{"an empty reason", "reason= " + good},
		{"empty", ""},
	} {
		if r, ok := gateRecordFromDetails(tc.details); ok {
			t.Errorf("%s: %q read as %+v; want it refused", tc.name, tc.details, r)
		}
	}
	for _, tc := range []struct{ name, details, log string }{
		{"an unknown key", "extra=1 " + good, "/state/gates/org-a.log"},
		{"no log", strings.Split(good, " log=")[0], ""},
		{"a log that is not quoted", strings.Split(good, " log=")[0] + " log=/plain", ""},
	} {
		if r, ok := gateRecordFromDetails(tc.details); !ok || r.Log != tc.log || r.Head != sha1 {
			t.Errorf("%s: %q read as %+v (%v); want it read with log %q", tc.name, tc.details, r, ok, tc.log)
		}
	}
}

// TestGateEvents_AreNotASeat covers AC6's second half: a manifest with
// gate_passed and gate_failed lines reads as before for the roster, the
// status, the report's roster, the running orgs, and disband --all's
// targets.
func TestGateEvents_AreNotASeat(t *testing.T) {
	leader := ManifestEvent{TS: "2026-10-13T11:00:00Z", OrgID: "org-a", SeatID: LeaderIdentity, Event: EventSpawned, Role: "leader", PaneID: "p-1"}
	r := gateRecord{Head: strings.Repeat("a", 40), Base: strings.Repeat("b", 40), BaseRef: "refs/remotes/origin/main", Branch: "feat/auth", By: "alice", Log: "/l"}
	passed := ManifestEvent{TS: "2026-10-13T12:00:00Z", OrgID: "org-a", Event: EventGatePassed, Details: r.details()}
	r.Reason = GateReasonTestFailed
	failed := ManifestEvent{TS: "2026-10-13T12:01:00Z", OrgID: "org-a", Event: EventGateFailed, Details: r.details()}
	withLines := []ManifestEvent{leader, passed, failed}

	for _, opts := range []RosterOptions{{}, {IncludeDryRun: true}} {
		roster := Roster(withLines, opts)
		if len(roster) != 1 || roster[0].SeatID != LeaderIdentity || !roster[0].Active || roster[0].Event != EventSpawned {
			t.Fatalf("Roster(%+v) = %+v, want only the active leader", opts, roster)
		}
	}
	if got := RunningOrgs([]ManifestEvent{passed, failed}); len(got) != 0 {
		t.Fatalf("RunningOrgs = %q, want gate events alone not to make an org run", got)
	}
	if got := orgsToDisband([]ManifestEvent{passed, failed}); len(got) != 0 {
		t.Fatalf("orgsToDisband = %q, want gate events alone not to make a target", got)
	}
	report := BuildOrgReport(withLines, nil, "org-a", "2026-10-13", 0)
	rosterSection := report[strings.Index(report, "## Roster"):strings.Index(report, "## Event timeline")]
	if rows := strings.Count(rosterSection, "\n| ") - 1; rows != 1 || !strings.Contains(rosterSection, "| "+LeaderIdentity+" |") {
		t.Fatalf("report roster has %d seat rows, want only the leader:\n%s", rows, rosterSection)
	}
	if !strings.Contains(report, "- active seats: 1\n") {
		t.Fatalf("report counts a phantom seat:\n%s", report)
	}

	fx := newApprovalFixture(t)
	fx.appendEvents(t, withLines...)
	st, err := fx.o.Status("org-a", true)
	if err != nil || len(st.Seats) != 1 || st.Seats[0].SeatID != LeaderIdentity || !st.Seats[0].Active || st.CorruptLines != 0 {
		t.Fatalf("Status = %+v, %v; want only the active leader", st, err)
	}
}

// TestLeaderActivityEventCount_GateEvents covers AC6's first half: each
// gate_passed and gate_failed of the watched org counts one, also one whose
// Details hold reason=watchdog_ (no gate record is a watchdog write), and
// another org's does not.
func TestLeaderActivityEventCount_GateEvents(t *testing.T) {
	gateEvent := func(orgID, event, details string) ManifestEvent {
		return ManifestEvent{TS: "2026-10-13T12:00:00Z", OrgID: orgID, Event: event, Details: details}
	}
	leader := ManifestEvent{TS: "2026-10-13T11:00:00Z", OrgID: "org-a", SeatID: LeaderIdentity, Event: EventSpawned, Role: "leader"}
	base := leaderActivityEventCount([]ManifestEvent{leader}, "org-a")
	tests := []struct {
		name string
		ev   ManifestEvent
		want int
	}{
		{"gate_passed in the watched org counts", gateEvent("org-a", EventGatePassed, "head=a"), 1},
		{"gate_failed in the watched org counts", gateEvent("org-a", EventGateFailed, "reason=test_failed head=a"), 1},
		{"a gate event whose Details hold reason=watchdog_ counts", gateEvent("org-a", EventGateFailed, `log="/reason=watchdog_x"`), 1},
		{"gate_passed in another org does not count", gateEvent("org-b", EventGatePassed, "head=a"), 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := leaderActivityEventCount([]ManifestEvent{leader, tt.ev}, "org-a") - base; got != tt.want {
				t.Errorf("leaderActivityEventCount adds %d, want %d (event %+v)", got, tt.want, tt.ev)
			}
		})
	}
}
