package org

import (
	"bytes"
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/yoshpy-dev/ralph/internal/toolchain"
)

// The tool versions a gate measures before its scripts (plan
// docs/plans/active/2026-11-12-gate-tool-versions.md, AC1, AC2, AC4, and the
// gate's part of AC10): the summary the record and the CLI get, the log's
// lines, the record's keys, and Gate and Regate on real git repositories,
// with the real probe (toolchain.Probe) and fake tools on the caller's PATH,
// or with a probe of the test's.

// mFound, mMissing, and mUnknown are measurements of tool in dir.
func mFound(dir, tool, version, path string) toolchain.Measurement {
	return toolchain.Measurement{Dir: dir, Tool: tool, Status: toolchain.StatusFound, Version: version, Path: path,
		Shim: toolchain.ShimOf(path)}
}

func mMissing(dir, tool string) toolchain.Measurement {
	return toolchain.Measurement{Dir: dir, Tool: tool, Status: toolchain.StatusMissing}
}

func mUnknown(dir, tool, reason, path string) toolchain.Measurement {
	return toolchain.Measurement{Dir: dir, Tool: tool, Status: toolchain.StatusUnknown, Reason: reason, Path: path}
}

// TestGateToolsSummary_OneValuePerToolInTheFixedOrder covers AC2's values:
// per tool, the version every place shares, missing or unknown when every
// place says so, and mixed for any difference between places, versions or
// statuses; a version that is not one token is unknown; the tools come in the
// order node, pnpm, yarn, npm, python3, go whatever the measurements' order,
// and a tool not measured is left out.
func TestGateToolsSummary_OneValuePerToolInTheFixedOrder(t *testing.T) {
	cases := []struct {
		name  string
		dirs  int
		ms    []toolchain.Measurement
		want  []GateTool
		mixed bool
	}{
		{"one place", 1, []toolchain.Measurement{
			mFound(".", "node", "v24.22.1", "/n"), mFound(".", "pnpm", "10.10.0", "/p"), mFound(".", "python3", "3.12.1", "/py"),
		}, []GateTool{{"node", "v24.22.1"}, {"pnpm", "10.10.0"}, {"python3", "3.12.1"}}, false},
		{"several places with the same version", 3, []toolchain.Measurement{
			mFound(".", "node", "v24.22.1", "/a"), mFound("apps/a", "node", "v24.22.1", "/b"), mFound("apps/b", "node", "v24.22.1", "/c"),
		}, []GateTool{{"node", "v24.22.1"}}, false},
		{"several places with different versions", 2, []toolchain.Measurement{
			mFound(".", "node", "v24.22.1", "/a"), mFound("apps/a", "node", "v22.1.0", "/a"),
		}, []GateTool{{"node", "mixed"}}, true},
		{"missing everywhere", 2, []toolchain.Measurement{mMissing(".", "yarn"), mMissing("apps/a", "yarn")},
			[]GateTool{{"yarn", "missing"}}, false},
		{"unknown everywhere", 2, []toolchain.Measurement{
			mUnknown(".", "npm", "exit 1", "/n"), mUnknown("apps/a", "npm", "no output", ""),
		}, []GateTool{{"npm", "unknown"}}, false},
		{"found in one place and missing in another", 2, []toolchain.Measurement{
			mFound(".", "pnpm", "10.10.0", "/p"), mMissing("apps/a", "pnpm"),
		}, []GateTool{{"pnpm", "mixed"}}, true},
		{"missing in one place and unknown in another", 2, []toolchain.Measurement{
			mMissing(".", "node"), mUnknown("apps/a", "node", "not measured", ""),
		}, []GateTool{{"node", "mixed"}}, true},
		{"a version that is not one token", 1, []toolchain.Measurement{mFound(".", "go", "go 1.27", "/g")},
			[]GateTool{{"go", "unknown"}}, false},
		{"the fixed order", 1, []toolchain.Measurement{
			mFound(".", "go", "go1.27.3", "/g"), mFound(".", "python3", "3.12.1", "/py"), mFound(".", "npm", "10.9.2", "/n"),
			mFound("a", "yarn", "4.5.0", "/y"), mFound("b", "pnpm", "10.10.0", "/p"), mFound(".", "node", "v24.22.1", "/n"),
		}, []GateTool{{"node", "v24.22.1"}, {"pnpm", "10.10.0"}, {"yarn", "4.5.0"}, {"npm", "10.9.2"}, {"python3", "3.12.1"},
			{"go", "go1.27.3"}}, false},
		{"nothing measured", 1, nil, nil, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := gateToolsSummary(tc.dirs, tc.ms)
			if want := (&GateTools{Dirs: tc.dirs, Tools: tc.want}); !reflect.DeepEqual(got, want) {
				t.Fatalf("gateToolsSummary = %+v\nwant %+v", got, want)
			}
			if got.Mixed() != tc.mixed {
				t.Errorf("Mixed() = %v; want %v", got.Mixed(), tc.mixed)
			}
		})
	}
	if (*GateTools)(nil).Mixed() {
		t.Error("a nil GateTools is Mixed")
	}
}

// TestGateToolsLogLines covers AC1's log lines: the line that says how the
// tools were measured first; a line per measurement with its value, the path
// the shell resolved, and the shim it is; the lines of `mise current` and how
// it ended; the places and paths left out; and why the probe did not finish.
func TestGateToolsLogLines(t *testing.T) {
	layout := toolchain.Layout{Places: []toolchain.Place{{Dir: "."}, {Dir: "apps/my dir"}}, Truncated: 3, SkippedPaths: 2}
	rep := toolchain.Report{
		Measurements: []toolchain.Measurement{
			mFound(".", "node", "v24.22.1", "/home/u/.local/share/mise/shims/node"),
			mFound(".", "pnpm", "10.10.0", "/home/u/.asdf/shims/pnpm"),
			mFound(".", "python3", "3.12.1", "/usr/bin/python3"),
			mUnknown(".", "go", "exit 2: boom", "/usr/local/go/bin/go"),
			mMissing("apps/my dir", "node"),
			mUnknown("apps/my dir", "yarn", "not measured: timed out after 15s", ""),
		},
		Mise:    &toolchain.MiseCurrent{Path: "/usr/bin/mise", Lines: []string{"node 24.22.1", "python 3.12.1"}},
		Failure: "timed out after 15s",
	}
	got := gateToolsLogLines(layout, nil, rep)
	want := []string{
		gateToolsExplainLine,
		"# tool node (.): v24.22.1 (/home/u/.local/share/mise/shims/node) (mise shim)",
		"# tool pnpm (.): 10.10.0 (/home/u/.asdf/shims/pnpm) (asdf shim)",
		"# tool python3 (.): 3.12.1 (/usr/bin/python3)",
		"# tool go (.): unknown (exit 2: boom) (/usr/local/go/bin/go)",
		"# tool node (apps/my dir): missing",
		"# tool yarn (apps/my dir): unknown (not measured: timed out after 15s)",
		"# mise current: node 24.22.1",
		"# mise current: python 3.12.1",
		"# tools: 3 more place(s) with a lockfile or a version file were not measured (the limit is " + strconv.Itoa(toolchain.MaxPlaces) +
			"); 2 tracked path(s) were skipped (a control character or invalid UTF-8 in the name)",
		"# tools: the probe did not finish (timed out after 15s): the tools it did not reach are unknown",
	}
	if !slices.Equal(got, want) {
		t.Fatalf("gateToolsLogLines\n got %q\nwant %q", got, want)
	}
	if !strings.Contains(gateToolsExplainLine, "/bin/sh") || !strings.Contains(gateToolsExplainLine, "shim") {
		t.Errorf("the explaining line %q does not name /bin/sh and the shim", gateToolsExplainLine)
	}

	for _, tc := range []struct {
		name string
		mise *toolchain.MiseCurrent
		want []string
	}{
		{"mise exited non-zero", &toolchain.MiseCurrent{Lines: []string{"mise ERROR untrusted"}, ExitCode: 1},
			[]string{"# mise current: mise ERROR untrusted", "# mise current: exited 1"}},
		{"mise did not finish", &toolchain.MiseCurrent{ExitCode: -1}, []string{"# mise current: did not finish"}},
		{"mise printed nothing", &toolchain.MiseCurrent{}, []string{"# mise current: printed nothing"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := gateToolsLogLines(toolchain.RootLayout(), nil, toolchain.Report{Mise: tc.mise})
			if !slices.Equal(got[1:], tc.want) {
				t.Errorf("mise lines = %q; want %q", got[1:], tc.want)
			}
		})
	}

	t.Run("the tracked files could not be listed", func(t *testing.T) {
		rep := toolchain.Report{Measurements: []toolchain.Measurement{mFound(".", "python3", "3.12.1", "/usr/bin/python3")}}
		got := gateToolsLogLines(toolchain.RootLayout(), errors.New("git ls-files in /r: exit status 128:\nfatal"), rep)
		want := []string{
			gateToolsExplainLine,
			"# tools: the tracked files could not be listed (git ls-files in /r: exit status 128:?fatal): python3 is measured at the root only",
			"# tool python3 (.): 3.12.1 (/usr/bin/python3)",
		}
		if !slices.Equal(got, want) {
			t.Errorf("gateToolsLogLines\n got %q\nwant %q", got, want)
		}
	})
}

// TestGateToolsTreeLines covers the comparison of D7: the same status before
// and after is no line; a line in only one of them is listed; a status that
// could not be read says the worktree was not compared.
func TestGateToolsTreeLines(t *testing.T) {
	head := "# tools: the probe changed the worktree; git status --porcelain --untracked-files=all differs in:"
	for _, tc := range []struct {
		name          string
		before, after []string
		beforeErr     error
		afterErr      error
		want          []string
		changed       int
	}{
		{"clean before and after", nil, nil, nil, nil, nil, 0},
		{"the same lines", []string{"?? a"}, []string{"?? a"}, nil, nil, nil, 0},
		{"a file written", nil, []string{"?? pinned", " M package.json"}, nil, nil,
			[]string{head, "#   ?? pinned", "#    M package.json"}, 2},
		{"a change gone", []string{"?? a", "?? b"}, []string{"?? a"}, nil, nil, []string{head, "#   ?? b"}, 1},
		{"no status before", nil, nil, errors.New("org: gate: git status: boom"), nil,
			[]string{"# tools: the worktree was not compared before and after the probe: org: gate: git status: boom"}, 0},
		{"no status after", nil, nil, nil, errors.New("boom"),
			[]string{"# tools: the worktree was not compared before and after the probe: boom"}, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, changed := gateToolsTreeLines(tc.before, tc.after, tc.beforeErr, tc.afterErr)
			if !slices.Equal(got, tc.want) || changed != tc.changed {
				t.Errorf("gateToolsTreeLines = %q, %d; want %q, %d", got, changed, tc.want, tc.changed)
			}
		})
	}
}

// TestGateRecordDetails_Tools covers AC2's record: tool_dirs and the tool
// keys come after the load keys and max_load, right before log=, and read
// back on a pass and a failure; a value that is not one token is written as
// unknown, so the record stays readable; a Details of the old shape, with no
// tool key, reads with none; a tool_dirs that is not a number above 0 reads
// as 0, and a tool key ralph does not know is skipped.
func TestGateRecordDetails_Tools(t *testing.T) {
	sha1, sha1b := strings.Repeat("a", 40), strings.Repeat("b", 40)
	r := gateRecord{Head: sha1, Base: sha1b, BaseRef: "refs/remotes/origin/main", Branch: "feat/auth", By: "alice", Via: gateViaRegate,
		Log: `/my state/a " tool_dirs=9 log=y.log`}.withLoad(&GateLoad{Start: "1.00", Highest: "2.00", End: "1.50", NCPU: 8, MaxLoad: 4})
	r = r.withTools(&GateTools{Dirs: 2, Tools: []GateTool{{"node", "mixed"}, {"pnpm", "10.10.0"}, {"python3", "3.12.1"},
		{"go", "missing"}, {"deno", "2.0.0"}}})
	want := "head=" + sha1 + " base=" + sha1b + " base_ref=refs/remotes/origin/main branch=feat/auth by=alice herdr_pane=unset " +
		"via=regate load_start=1.00 load_max=2.00 load_end=1.50 ncpu=8 max_load=4 " +
		"tool_dirs=2 tool_node=mixed tool_pnpm=10.10.0 tool_python3=3.12.1 tool_go=missing log=" + strconv.Quote(r.Log)
	if got := r.details(); got != want {
		t.Fatalf("details = %q\nwant      %q", got, want)
	}
	for _, reason := range []string{"", GateReasonTestFailed} {
		r.Reason = reason
		if got, ok := gateRecordFromDetails(r.details()); !ok || got != r {
			t.Errorf("details %q read back as %+v (%v); want %+v", r.details(), got, ok, r)
		}
	}
	r.Reason = ""

	bad := r
	bad.Tools[slices.Index(gateToolNames[:], "node")] = "v24 (from mise)"
	if d := bad.details(); !strings.Contains(d, " tool_node=unknown tool_pnpm=") {
		t.Errorf("details with a value that is not one token = %q; want tool_node=unknown", d)
	} else if got, ok := gateRecordFromDetails(d); !ok || got.Tools[0] != "unknown" {
		t.Errorf("details %q read back as %+v (%v); want it read with node unknown", d, got, ok)
	}

	old := gateRecord{Head: sha1, Base: sha1b, BaseRef: "refs/remotes/origin/main", Branch: "feat/auth", By: "alice", Log: "/l"}
	if d := old.details(); strings.Contains(d, "tool_") {
		t.Errorf("details of a record with no tools = %q; want no tool key", d)
	}
	oldShape := "head=" + sha1 + " base=" + sha1b + " base_ref=refs/remotes/origin/main branch=feat/auth by=alice herdr_pane=unset " +
		"load_start=1.00 load_max=1.00 load_end=1.00 ncpu=8 log=\"/l\""
	if got, ok := gateRecordFromDetails(oldShape); !ok || got.ToolDirs != 0 || got.Tools != [len(gateToolNames)]string{} || got.Log != "/l" {
		t.Errorf("%q read as %+v (%v); want it read with no tools", oldShape, got, ok)
	}
	odd := strings.Replace(r.details(), "tool_dirs=2", "tool_dirs=two tool_deno=2.0.0", 1)
	if got, ok := gateRecordFromDetails(odd); !ok || got.ToolDirs != 0 || got.Tools != r.Tools {
		t.Errorf("%q read as %+v (%v); want tool_dirs 0, the known tools, and no deno", odd, got, ok)
	}
}

// writeFakeTool writes an executable shell script named name in dir.
func writeFakeTool(t *testing.T, dir, name, body string) {
	t.Helper()
	writeGateFile(t, dir, name, "#!/bin/sh\n"+body, 0o755)
}

// gateToolsPATH makes a bin directory and a mise shims directory under a temp
// dir and returns them with the PATH that finds them and nothing else, so no
// tool or mise of the machine is measured. bin holds links to the env and cat
// the stub gate scripts run.
func gateToolsPATH(t *testing.T) (bin, shims, path string) {
	t.Helper()
	base := resolved(t, t.TempDir())
	bin, shims = filepath.Join(base, "bin"), filepath.Join(base, "home", ".local", "share", "mise", "shims")
	mkdirAll(t, bin)
	mkdirAll(t, shims)
	for _, name := range []string{"env", "cat"} {
		target, err := exec.LookPath(name)
		if err != nil {
			t.Fatalf("look up %s: %v", name, err)
		}
		if err := os.Symlink(target, filepath.Join(bin, name)); err != nil {
			t.Fatal(err)
		}
	}
	return bin, shims, shims + ":" + bin
}

// withEnv returns env with each of set (NAME=value) in place of the entries
// of that name.
func withEnv(env []string, set ...string) []string {
	names := map[string]bool{}
	for _, kv := range set {
		name, _, _ := strings.Cut(kv, "=")
		names[name] = true
	}
	var out []string
	for _, kv := range env {
		if name, _, _ := strings.Cut(kv, "="); !names[name] {
			out = append(out, kv)
		}
	}
	return append(out, set...)
}

// writeJSFiles writes a package.json and a pnpm-lock.yaml in dir: the root is
// then a JS place whose package manager is pnpm.
func writeJSFiles(t *testing.T, dir string) {
	t.Helper()
	writeGateStubs(t, dir)
	writeGateFile(t, dir, "package.json", `{"name": "fixture"}`+"\n", 0o644)
	writeGateFile(t, dir, "pnpm-lock.yaml", "lockfileVersion: '9.0'\n", 0o644)
}

// envLines returns the lines of the env dump at path.
func envLines(t *testing.T, path string) []string {
	t.Helper()
	return strings.Split(fileOrAbsent(t, path), "\n")
}

// TestGate_RecordsTheToolVersionsWithTheRealProbe covers AC1, AC2, AC5's
// GateResult.Tools, and AC10 on Gate with the real probe: in a worktree with a
// package.json and a pnpm-lock.yaml, fake node, pnpm (a mise shim), python3,
// and mise on the caller's PATH, the log's header has their lines right after
// the load at the start, the gate_passed has their keys before log=, and
// GateResult.Tools their values. The fake pnpm pins itself into the worktree
// unless COREPACK_ENABLE_AUTO_PIN is 0, and the caller sets it to 1: the probe
// overrides it, so nothing is written and nothing is said on stderr, while
// the scripts still get the caller's value. The probe gets the scripts'
// environment (RALPH_VERIFY_SCOPE=full) with the seven variables of D7.
func TestGate_RecordsTheToolVersionsWithTheRealProbe(t *testing.T) {
	setGateToolsProbe(t, toolchain.Probe)
	fx := newGateFixtureWith(t, gateFixtureOptions{populate: writeJSFiles})
	bin, shims, path := gateToolsPATH(t)
	writeFakeTool(t, bin, "node", "echo v24.22.1\n")
	writeFakeTool(t, shims, "pnpm", "if [ \"${COREPACK_ENABLE_AUTO_PIN-}\" != 0 ]; then echo pinned > ./pinned-by-probe; fi\necho 10.10.0\n")
	writeFakeTool(t, bin, "python3", "env > \"$GATE_TEST_OUT/probe-env\"\necho 'Python 3.12.1'\n")
	writeFakeTool(t, bin, "mise", "[ \"$1\" = current ] && echo 'node 24.22.1'\n")
	fx.env = withEnv(fx.env, "PATH="+path, "COREPACK_ENABLE_AUTO_PIN=1", "GOTOOLCHAIN=auto", "RALPH_VERIFY_SCOPE=changed")
	var stderr lockedBuffer
	res, err := fx.o.Gate(context.Background(), GateParams{OrgID: "org-a", Dir: fx.worktree, Env: fx.env, Stderr: &stderr})
	if err != nil || !res.Passed {
		t.Fatalf("gate = %+v, %v; want a pass", res, err)
	}

	wantTools := &GateTools{Dirs: 1, Tools: []GateTool{{"node", "v24.22.1"}, {"pnpm", "10.10.0"}, {"python3", "3.12.1"}}}
	if !reflect.DeepEqual(res.Tools, wantTools) {
		t.Errorf("GateResult.Tools = %+v; want %+v", res.Tools, wantTools)
	}
	events := fx.gateEvents(t)
	if len(events) != 1 || !strings.Contains(events[0].Details,
		testGateLoadDetails+" tool_dirs=1 tool_node=v24.22.1 tool_pnpm=10.10.0 tool_python3=3.12.1 log=") {
		t.Fatalf("gate events = %+v; want one gate_passed with the tool keys before log=", events)
	}
	if r, ok := gateRecordFromDetails(events[0].Details); !ok || r.ToolDirs != 1 || r.Tools[0] != "v24.22.1" || r.Tools[4] != "3.12.1" {
		t.Errorf("record %q read as %+v (%v)", events[0].Details, r, ok)
	}
	log := fileOrAbsent(t, res.LogPath)
	wantHeader := "# load at start: 1.50 1.25 1.00 (8 CPUs)\n" + gateToolsExplainLine + "\n" +
		"# tool node (.): v24.22.1 (" + filepath.Join(bin, "node") + ")\n" +
		"# tool pnpm (.): 10.10.0 (" + filepath.Join(shims, "pnpm") + ") (mise shim)\n" +
		"# tool python3 (.): 3.12.1 (" + filepath.Join(bin, "python3") + ")\n" +
		"# mise current: node 24.22.1\n\n==> ralph org gate: running ./" + GateStaticScript + "\n"
	if !strings.Contains(log, wantHeader) {
		t.Errorf("log does not have\n%s\nin:\n%s", wantHeader, log)
	}

	if stderr.String() != "" {
		t.Errorf("stderr = %q; want nothing", stderr.String())
	}
	if st := gitOut(t, fx.worktree, "status", "--porcelain", "--untracked-files=all", "--ignored"); st != "" {
		t.Errorf("the gate wrote in the worktree:\n%s", st)
	}
	probeEnv := envLines(t, filepath.Join(fx.out, "probe-env"))
	for _, kv := range []string{"COREPACK_ENABLE_AUTO_PIN=0", "GOTOOLCHAIN=local", "MISE_AUTO_INSTALL=0", "MISE_NOT_FOUND_AUTO_INSTALL=0",
		"COREPACK_ENABLE_NETWORK=0", "COREPACK_ENABLE_DOWNLOAD_PROMPT=0", "NPM_CONFIG_UPDATE_NOTIFIER=false", "RALPH_VERIFY_SCOPE=full",
		"RALPH_LOCAL_CONFIG=off"} {
		if !slices.Contains(probeEnv, kv) {
			t.Errorf("the probe's environment lacks %s:\n%s", kv, strings.Join(probeEnv, "\n"))
		}
	}
	for _, kv := range []string{"COREPACK_ENABLE_AUTO_PIN=1", "GOTOOLCHAIN=auto", "RALPH_VERIFY_SCOPE=changed"} {
		if slices.Contains(probeEnv, kv) {
			t.Errorf("the probe's environment has the caller's %s", kv)
		}
	}
	scriptEnv := envLines(t, filepath.Join(fx.out, "env-static"))
	for _, kv := range []string{"COREPACK_ENABLE_AUTO_PIN=1", "GOTOOLCHAIN=auto"} {
		if !slices.Contains(scriptEnv, kv) {
			t.Errorf("the static script's environment lacks the caller's %s", kv)
		}
	}
	if slices.Contains(scriptEnv, "GOTOOLCHAIN=local") || slices.Contains(scriptEnv, "COREPACK_ENABLE_AUTO_PIN=0") {
		t.Errorf("the static script got the probe's variables:\n%s", strings.Join(scriptEnv, "\n"))
	}
}

// TestGate_ProbeThatWritesInTheWorktreeIsReported covers AC10's other half:
// a fake tool that writes into the worktree whatever the variables say makes
// the log list the change and stderr say so in one line, the scripts run as
// before, and the gate records tree_changed.
func TestGate_ProbeThatWritesInTheWorktreeIsReported(t *testing.T) {
	setGateToolsProbe(t, toolchain.Probe)
	fx := newGateFixture(t)
	bin, _, path := gateToolsPATH(t)
	writeFakeTool(t, bin, "python3", "echo x > ./written-by-python3\necho 'Python 3.12.1'\n")
	fx.env = withEnv(fx.env, "PATH="+path)
	var stderr lockedBuffer
	res, err := fx.o.Gate(context.Background(), GateParams{OrgID: "org-a", Dir: fx.worktree, Env: fx.env, Stderr: &stderr})
	if err != nil || res.Reason != GateReasonTreeChanged || !slices.Equal(res.Dirty, []string{"?? written-by-python3"}) {
		t.Fatalf("gate = %+v, %v; want tree_changed with the file the probe wrote", res, err)
	}
	if calls := fx.calls(t); !slices.Equal(calls, []string{"static", "test"}) {
		t.Errorf("scripts run = %v; want static then test", calls)
	}
	want := "org: gate: the version probe changed the worktree (1 line(s) of git status, listed in the gate log): the scripts run " +
		"as before, and a pass is recorded as tree_changed while the change is there\n"
	if stderr.String() != want {
		t.Errorf("stderr = %q\nwant     %q", stderr.String(), want)
	}
	if log := fileOrAbsent(t, res.LogPath); !strings.Contains(log, "\n# tool python3 (.): 3.12.1 ("+filepath.Join(bin, "python3")+")\n"+
		"# tools: the probe changed the worktree; git status --porcelain --untracked-files=all differs in:\n#   ?? written-by-python3\n\n==> ") {
		t.Errorf("log does not list the change after the tool lines:\n%s", log)
	}
	if events := fx.gateEvents(t); len(events) != 1 || events[0].Event != EventGateFailed ||
		!strings.HasPrefix(events[0].Details, "reason=tree_changed ") || !strings.Contains(events[0].Details, " tool_python3=3.12.1 log=") {
		t.Fatalf("gate events = %+v; want one gate_failed tree_changed with the tool keys", events)
	}
}

// setGateToolsTimes shortens the probe's budget and the grace of its group's
// stop until the test ends.
func setGateToolsTimes(t *testing.T, budget, grace time.Duration) {
	t.Helper()
	oldBudget, oldGrace := gateToolsTimeout, toolchain.KillGrace
	t.Cleanup(func() { gateToolsTimeout, toolchain.KillGrace = oldBudget, oldGrace })
	gateToolsTimeout, toolchain.KillGrace = budget, grace
}

// TestGate_ProbePastItsBudgetLeavesTheToolsUnknown covers AC4: a fake node
// that sleeps past the (shortened) budget is stopped with its group, the
// tools the probe did not reach are unknown in the log, the record, and
// GateResult.Tools, the log says why, and the gate passes all the same.
func TestGate_ProbePastItsBudgetLeavesTheToolsUnknown(t *testing.T) {
	setGateToolsProbe(t, toolchain.Probe)
	setGateToolsTimes(t, time.Second, 300*time.Millisecond)
	fx := newGateFixtureWith(t, gateFixtureOptions{populate: writeJSFiles})
	bin, _, path := gateToolsPATH(t)
	writeFakeTool(t, bin, "node", "/bin/sleep 30\necho v24.22.1\n")
	writeFakeTool(t, bin, "python3", "echo 'Python 3.12.1'\n")
	fx.env = withEnv(fx.env, "PATH="+path)
	begin := time.Now()
	res := fx.gate(t)
	if took := time.Since(begin); took > 20*time.Second {
		t.Errorf("the gate took %s; want the probe stopped after its 1s budget", took)
	}
	want := &GateTools{Dirs: 1, Tools: []GateTool{{"node", "unknown"}, {"pnpm", "unknown"}, {"python3", "unknown"}}}
	if !res.Passed || !reflect.DeepEqual(res.Tools, want) {
		t.Fatalf("gate = %+v (Tools %+v); want a pass with Tools %+v", res, res.Tools, want)
	}
	log := fileOrAbsent(t, res.LogPath)
	for _, w := range []string{
		"\n# tool node (.): unknown (not measured: timed out after 1s) (" + filepath.Join(bin, "node") + ")\n",
		"\n# tool python3 (.): unknown (not measured: timed out after 1s)\n",
		"\n# tools: the probe did not finish (timed out after 1s): the tools it did not reach are unknown\n",
	} {
		if !strings.Contains(log, w) {
			t.Errorf("log does not have %q:\n%s", w, log)
		}
	}
	if events := fx.gateEvents(t); len(events) != 1 ||
		!strings.Contains(events[0].Details, " tool_dirs=1 tool_node=unknown tool_pnpm=unknown tool_python3=unknown log=") {
		t.Fatalf("gate events = %+v; want one gate_passed with the tools unknown", events)
	}
}

// TestGate_InterruptDuringTheProbeRecordsNothing covers AC4's ctx: an
// interrupt while the probe runs stops it, runs no script, records nothing,
// and says so with the log, which says the probe was interrupted; the error
// says it came before the scripts ran, not that they were stopped; gate.lock
// is free afterwards.
func TestGate_InterruptDuringTheProbeRecordsNothing(t *testing.T) {
	setGateToolsProbe(t, toolchain.Probe)
	setGateToolsTimes(t, time.Minute, 300*time.Millisecond)
	fx := newGateFixture(t)
	bin, _, path := gateToolsPATH(t)
	writeFakeTool(t, bin, "python3", ": > \"$GATE_TEST_OUT/probe-started\"\nexec /bin/sleep 30\n")
	fx.env = withEnv(fx.env, "PATH="+path)
	before := fileOrAbsent(t, fx.o.Manifest.Path())
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan gateOutcome, 1)
	go func() {
		res, err := fx.run(ctx, "org-a", fx.worktree)
		done <- gateOutcome{res, err}
	}()
	waitForFile(t, filepath.Join(fx.out, "probe-started"))
	cancel()
	got := waitGate(t, done, 10*time.Second)
	logPath := filepath.Join(fx.stateDir, gateLogDir, gateTestLog)
	if want := "org: gate: interrupted (context canceled) before the scripts ran: nothing was recorded; the output so far is in " +
		logPath; got.err == nil || got.err.Error() != want {
		t.Fatalf("gate = %+v, %v\nwant the error %s", got.res, got.err, want)
	}
	if got.res.Tools != nil || got.res.Load != nil {
		t.Errorf("gate = %+v; want no Tools and no Load", got.res)
	}
	if after := fileOrAbsent(t, fx.o.Manifest.Path()); after != before {
		t.Errorf("the interrupted gate changed the manifest:\nbefore %s\nafter  %s", before, after)
	}
	if calls := fx.calls(t); calls != nil {
		t.Errorf("the interrupted gate ran %v; want no script", calls)
	}
	if log := fileOrAbsent(t, logPath); !strings.HasSuffix(log, "# tools: interrupted\n") {
		t.Errorf("log does not end with the probe's interrupt:\n%s", log)
	}
	if held, err := fileLockHeld(gateLockPathIn(fx.stateDir)); err != nil || held {
		t.Errorf("gate.lock after the interrupted gate: held %t, %v; want free", held, err)
	}
}

// TestMeasureGateTools_WithoutTrackedFiles covers D2's fallback and the
// probe's call: in a directory git cannot list, the log says why and only
// python3 is measured at the root, and the probe gets /bin/sh -c, the
// directory, the scripts' environment, the budget, and mise current; the
// status that cannot be read says the worktree was not compared. A probe that
// fails with ctx not done (toolchain.Probe never does) leaves every tool
// unknown and is no error.
func TestMeasureGateTools_WithoutTrackedFiles(t *testing.T) {
	isolateGitEnv(t)
	root := resolved(t, t.TempDir())
	env := []string{"PATH=/x", "RALPH_VERIFY_SCOPE=full"}
	var got toolchain.ProbeOptions
	setGateToolsProbe(t, func(ctx context.Context, opts toolchain.ProbeOptions) (toolchain.Report, error) {
		got = opts
		return testGateToolsProbe(ctx, opts)
	})
	var log, stderr bytes.Buffer
	tools, err := measureGateTools(context.Background(), root, env, &log, &stderr, gateWords)
	if err != nil {
		t.Fatalf("measureGateTools: %v", err)
	}
	want := toolchain.ProbeOptions{Argv: []string{"/bin/sh", "-c"}, Dir: root, Env: env, Timeout: gateToolsTimeout,
		Targets: []toolchain.Target{{Dir: ".", Tool: "python3"}}, MiseCurrent: true}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("probe options = %+v\nwant %+v", got, want)
	}
	if !reflect.DeepEqual(tools, testGateTools()) {
		t.Errorf("tools = %+v; want %+v", tools, testGateTools())
	}
	lines := strings.Split(strings.TrimSuffix(log.String(), "\n"), "\n")
	if len(lines) != 4 || lines[0] != gateToolsExplainLine ||
		!strings.HasPrefix(lines[1], "# tools: the tracked files could not be listed (git ls-files in "+root+": ") ||
		!strings.HasSuffix(lines[1], "): python3 is measured at the root only") ||
		lines[2] != "# tool python3 (.): 3.12.1 (/usr/bin/python3)" ||
		!strings.HasPrefix(lines[3], "# tools: the worktree was not compared before and after the probe: org: gate: git status ") {
		t.Errorf("log =\n%s", log.String())
	}
	if stderr.Len() != 0 {
		t.Errorf("stderr = %q; want nothing", stderr.String())
	}

	setGateToolsProbe(t, func(context.Context, toolchain.ProbeOptions) (toolchain.Report, error) {
		return toolchain.Report{}, errors.New("no shell")
	})
	log.Reset()
	tools, err = measureGateTools(context.Background(), root, env, &log, &stderr, gateWords)
	if err != nil || !reflect.DeepEqual(tools, &GateTools{Dirs: 1, Tools: []GateTool{{"python3", "unknown"}}}) {
		t.Fatalf("measureGateTools with a failing probe = %+v, %v; want python3 unknown and no error", tools, err)
	}
	for _, w := range []string{"\n# tool python3 (.): unknown (not measured: no shell)\n",
		"\n# tools: the probe did not finish (no shell): the tools it did not reach are unknown\n"} {
		if !strings.Contains(log.String(), w) {
			t.Errorf("log does not have %q:\n%s", w, log.String())
		}
	}
}

// TestRegate_ProbeThatChangesTheWorktreeSaysSoOnStderr covers the Regate side
// of D7 and AC2: a local regate passes RegateParams.Stderr down to the probe's
// check, so a probe that writes into the worktree is said there with
// "org: regate: ", the log lists it, and the gate_failed via=regate it records
// (tree_changed) holds the tool keys, as RegateResult.Tools holds the tools.
func TestRegate_ProbeThatChangesTheWorktreeSaysSoOnStderr(t *testing.T) {
	fx := newRegateFixture(t)
	setGateToolsProbe(t, func(ctx context.Context, opts toolchain.ProbeOptions) (toolchain.Report, error) {
		if err := os.WriteFile(filepath.Join(opts.Dir, "pinned-by-probe"), []byte("x\n"), 0o644); err != nil {
			return toolchain.Report{}, err
		}
		return testGateToolsProbe(ctx, opts)
	})
	var stderr lockedBuffer
	res, err := fx.o.Regate(context.Background(), RegateParams{Branch: "feat/auth", Env: fx.env, Stderr: &stderr})
	if err != nil || res.Outcome != RegateOutcomeGated || res.Reason != GateReasonTreeChanged ||
		!reflect.DeepEqual(res.Tools, testGateTools()) {
		t.Fatalf("regate = %+v (Tools %+v), %v; want a gated tree_changed with the tools", res, res.Tools, err)
	}
	if !strings.HasPrefix(stderr.String(), "org: regate: the version probe changed the worktree (1 line(s) of git status, ") {
		t.Errorf("stderr = %q; want the probe's line", stderr.String())
	}
	if log := fileOrAbsent(t, res.LogPath); !strings.Contains(log, "\n#   ?? pinned-by-probe\n") {
		t.Errorf("log does not list the file:\n%s", log)
	}
	if last := lastEvent(t, fx.o); last.Event != EventGateFailed ||
		!strings.Contains(last.Details, " via=regate"+testGateLoadDetails+testGateToolsDetails+" log=") {
		t.Fatalf("last event = %+v; want a gate_failed via regate with the tool keys", last)
	}
}

// TestRegateCI_RecordsNoTools covers AC2's ci record: a regate that reads
// the PR's checks runs no script, so it measures no tool, its gate_ci_passed
// has no tool key, and RegateResult.Tools is nil.
func TestRegateCI_RecordsNoTools(t *testing.T) {
	fx := newRCFixture(t)
	setGateToolsProbe(t, func(context.Context, toolchain.ProbeOptions) (toolchain.Report, error) {
		t.Error("a ci regate measured the tools")
		return toolchain.Report{}, nil
	})
	res, err := fx.o.Regate(context.Background(), fx.params())
	if err != nil || res.Outcome != RegateOutcomeGated || !res.Passed || res.Tools != nil {
		t.Fatalf("regate = %+v, %v; want a gated pass with no Tools", res, err)
	}
	if last := lastEvent(t, fx.o); last.Event != EventGateCIPassed || strings.Contains(last.Details, "tool_") {
		t.Fatalf("last event = %+v; want a gate_ci_passed with no tool key", last)
	}
}
