package toolchain

import (
	"context"
	"errors"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"
)

// writeExec writes an executable script at p.
func writeExec(t *testing.T, p, script string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
}

// probeEnvNames are the six variables ProbeEnv sets, in its order.
var probeEnvNames = []string{
	"MISE_AUTO_INSTALL", "MISE_NOT_FOUND_AUTO_INSTALL", "COREPACK_ENABLE_AUTO_PIN",
	"COREPACK_ENABLE_NETWORK", "COREPACK_ENABLE_DOWNLOAD_PROMPT", "NPM_CONFIG_UPDATE_NOTIFIER",
}

func TestProbeEnv_SetsTheSixVariablesLastOverCallerValues(t *testing.T) {
	in := []string{
		"PATH=/x/bin",
		"COREPACK_ENABLE_AUTO_PIN=1",
		"MISE_AUTO_INSTALL=1",
		"MISE_AUTO_INSTALL_EXTRA=keep",
		"NPM_CONFIG_UPDATE_NOTIFIER",
		"COREPACK_ENABLE_NETWORK=1",
		"HOME=/h",
		"COREPACK_ENABLE_AUTO_PIN=true",
	}
	orig := slices.Clone(in)
	got := ProbeEnv(in)
	want := []string{
		"PATH=/x/bin",
		"MISE_AUTO_INSTALL_EXTRA=keep",
		"HOME=/h",
		"MISE_AUTO_INSTALL=0",
		"MISE_NOT_FOUND_AUTO_INSTALL=0",
		"COREPACK_ENABLE_AUTO_PIN=0",
		"COREPACK_ENABLE_NETWORK=0",
		"COREPACK_ENABLE_DOWNLOAD_PROMPT=0",
		"NPM_CONFIG_UPDATE_NOTIFIER=false",
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("ProbeEnv\n got %q\nwant %q", got, want)
	}
	if !reflect.DeepEqual(in, orig) {
		t.Errorf("ProbeEnv changed its input: %q", in)
	}
	for _, name := range probeEnvNames {
		n := 0
		for _, kv := range got {
			if k, _, _ := strings.Cut(kv, "="); k == name {
				n++
			}
		}
		if n != 1 {
			t.Errorf("%s appears %d times in ProbeEnv's result, want once", name, n)
		}
	}
	if got := ProbeEnv(nil); len(got) != 6 {
		t.Errorf("ProbeEnv(nil) = %q, want the six variables", got)
	}
}

// rec is one probe record line.
func rec(kind, dir, label, value string) string {
	return recordPrefix + "\t" + kind + "\t" + dir + "\t" + label + "\t" + value + "\n"
}

func TestParseProbe_ReadsEachKindOfOutcome(t *testing.T) {
	out := "Welcome to the login shell\n" + "ralph-toolchain but not a record\n" +
		rec("path", ".", "node", "/home/u/.local/share/mise/shims/node") +
		rec("begin", ".", "node", "") + "v24.22.1\r\n" + rec("exit", ".", "node", "0") +
		rec("path", ".", "npm", "/usr/bin/npm") +
		rec("begin", ".", "npm", "") + "npm WARN config something\n10.9.2\n" + rec("exit", ".", "npm", "0") +
		rec("path", ".", "python3", "/usr/bin/python3") +
		rec("begin", ".", "python3", "") + "a\nb\nc\nPython 3.12.1\n" + rec("exit", ".", "python3", "0") +
		rec("path", ".", "go", "/usr/local/go/bin/go") +
		rec("begin", ".", "go", "") + "\x1b[31mboom\n" + rec("exit", ".", "go", "2") +
		rec("path", "apps/a", "node", "/home/u/.asdf/shims/node") +
		rec("begin", "apps/a", "node", "") + rec("exit", "apps/a", "node", "0") +
		rec("path", "apps/a", "pnpm", "/opt/odd\tdir/pnpm") +
		rec("begin", "apps/a", "pnpm", "") + "hello world\n" + rec("exit", "apps/a", "pnpm", "0") +
		rec("nocd", "apps/b", "", "") +
		rec("missing", "apps/c", "node", "") +
		rec("path", "apps/c", "yarn", "/usr/bin/yarn") +
		rec("begin", "apps/c", "yarn", "") + "mise ERROR yarn@4 is not installed\n" + rec("exit", "apps/c", "yarn", "1") +
		rec("path", ".", "mise-current", "/usr/bin/mise") +
		rec("begin", ".", "mise-current", "") + "1\n2\n\n3\n4\n5\n6\n" + rec("exit", ".", "mise-current", "0") +
		rec("end", ".", "", "")
	targets := []Target{
		{".", "node"}, {".", "npm"}, {".", "python3"}, {".", "go"},
		{"apps/a", "node"}, {"apps/a", "pnpm"},
		{"apps/b", "node"}, {"apps/b", "npm"},
		{"apps/c", "node"}, {"apps/c", "yarn"},
		{"apps/d", "node"},
	}
	rep := ParseProbe([]byte(out), targets)
	want := []Measurement{
		{Dir: ".", Tool: "node", Status: StatusFound, Path: "/home/u/.local/share/mise/shims/node", Shim: ShimMise, Version: "v24.22.1"},
		{Dir: ".", Tool: "npm", Status: StatusFound, Path: "/usr/bin/npm", Version: "10.9.2"},
		{Dir: ".", Tool: "python3", Status: StatusUnknown, Path: "/usr/bin/python3", Reason: "no version in the output: a"},
		{Dir: ".", Tool: "go", Status: StatusUnknown, Path: "/usr/local/go/bin/go", Reason: "exit 2: ?[31mboom"},
		{Dir: "apps/a", Tool: "node", Status: StatusUnknown, Path: "/home/u/.asdf/shims/node", Shim: ShimAsdf, Reason: "no output"},
		{Dir: "apps/a", Tool: "pnpm", Status: StatusUnknown, Path: "/opt/odd?dir/pnpm", Reason: "no version in the output: hello world"},
		{Dir: "apps/b", Tool: "node", Status: StatusUnknown, Reason: "cannot cd into the directory"},
		{Dir: "apps/b", Tool: "npm", Status: StatusUnknown, Reason: "cannot cd into the directory"},
		{Dir: "apps/c", Tool: "node", Status: StatusMissing},
		{Dir: "apps/c", Tool: "yarn", Status: StatusUnknown, Path: "/usr/bin/yarn", Reason: "exit 1: mise ERROR yarn@4 is not installed"},
		{Dir: "apps/d", Tool: "node", Status: StatusUnknown, Reason: "not measured"},
	}
	if !reflect.DeepEqual(rep.Measurements, want) {
		for i := range max(len(want), len(rep.Measurements)) {
			var g, w Measurement
			if i < len(rep.Measurements) {
				g = rep.Measurements[i]
			}
			if i < len(want) {
				w = want[i]
			}
			if g != w {
				t.Errorf("measurement %d\n got %+v\nwant %+v", i, g, w)
			}
		}
	}
	wantMise := &MiseCurrent{Path: "/usr/bin/mise", Lines: []string{"1", "2", "3", "4", "5"}, ExitCode: 0}
	if !reflect.DeepEqual(rep.Mise, wantMise) {
		t.Errorf("Mise = %+v, want %+v", rep.Mise, wantMise)
	}
	if !rep.Complete {
		t.Error("Complete = false, want true (the end record came)")
	}
}

func TestParseProbe_StopsAtAMissingEndRecord(t *testing.T) {
	out := rec("path", ".", "python3", "/usr/bin/python3") + rec("begin", ".", "python3", "") + "Python 3.12.1\n" +
		rec("exit", ".", "python3", "0") + rec("path", ".", "node", "/usr/bin/node") + rec("begin", ".", "node", "") +
		rec("path", ".", "mise-current", "/usr/bin/mise") + rec("begin", ".", "mise-current", "") + "node 24\n"
	rep := ParseProbe([]byte(out), []Target{{".", "python3"}, {".", "node"}})
	if rep.Complete {
		t.Error("Complete = true without an end record")
	}
	if m := rep.Measurements[0]; m.Status != StatusFound || m.Version != "3.12.1" {
		t.Errorf("python3 = %+v, want found 3.12.1", m)
	}
	if m := rep.Measurements[1]; m.Status != StatusUnknown || m.Reason != "not measured" || m.Path != "/usr/bin/node" {
		t.Errorf("node = %+v, want unknown, not measured, with its path", m)
	}
	if rep.Mise == nil || rep.Mise.ExitCode != -1 || !reflect.DeepEqual(rep.Mise.Lines, []string{"node 24"}) {
		t.Errorf("Mise = %+v, want the line so far and exit code -1", rep.Mise)
	}

	missing := ParseProbe([]byte(rec("missing", ".", "mise-current", "")+rec("end", ".", "", "")), nil)
	if missing.Mise != nil {
		t.Errorf("mise not on PATH: Mise = %+v, want nil", missing.Mise)
	}
}

// fakeTools makes a bin directory holding the fake tools the probe tests
// run, and a mise shims directory holding pnpm, and returns the PATH that
// finds them (and nothing else: no yarn, no npm).
func fakeTools(t *testing.T) (path, shims string) {
	t.Helper()
	base := t.TempDir()
	bin := filepath.Join(base, "bin")
	shims = filepath.Join(base, "home", ".local", "share", "mise", "shims")
	// node prints another version in "apps/my dir".
	writeExec(t, filepath.Join(bin, "node"), "#!/bin/sh\ncase \"$PWD\" in\n*'/apps/my dir') echo v22.1.0 ;;\n*) echo v24.22.1 ;;\nesac\n")
	// pnpm prints two lines.
	writeExec(t, filepath.Join(shims, "pnpm"), "#!/bin/sh\nprintf '10.10.0\\nsecond line\\n'\n")
	// python3 prints no final newline.
	writeExec(t, filepath.Join(bin, "python3"), "#!/bin/sh\nprintf 'Python 3.12.1'\n")
	// go prints garbage.
	writeExec(t, filepath.Join(bin, "go"), "#!/bin/sh\necho 'something went wrong'\n")
	writeExec(t, filepath.Join(bin, "mise"), "#!/bin/sh\n[ \"$1\" = current ] && printf 'node 24.22.1\\npython 3.12.1\\n'\n")
	return shims + ":" + bin, shims
}

// probeRepo makes a repository directory with the places "apps/b" and
// "apps/my dir", and returns it with its layout.
func probeRepo(t *testing.T) (string, Layout) {
	t.Helper()
	root := t.TempDir()
	files := []string{"package.json", "pnpm-lock.yaml", "go.mod", "apps/my dir/.nvmrc", "apps/b/yarn.lock"}
	for _, f := range files {
		writeFile(t, filepath.Join(root, filepath.FromSlash(f)), "")
	}
	return root, LayoutFromFiles(files, nil)
}

func TestProbe_MeasuresFakeToolsPlaceByPlace(t *testing.T) {
	shells := [][]string{{"/bin/sh", "-c"}}
	for _, s := range [][]string{{"dash", "-c"}, {"bash", "-c"}, {"zsh", "-f", "-c"}} {
		if p, err := exec.LookPath(s[0]); err == nil {
			shells = append(shells, append([]string{p}, s[1:]...))
		}
	}
	for _, argv := range shells {
		t.Run(filepath.Base(argv[0]), func(t *testing.T) {
			path, shims := fakeTools(t)
			root, layout := probeRepo(t)
			home := t.TempDir()
			rep, err := Probe(context.Background(), ProbeOptions{
				Argv:        argv,
				Dir:         root,
				Env:         []string{"PATH=" + path, "HOME=" + home, "ZDOTDIR=" + home},
				Timeout:     20 * time.Second,
				Targets:     layout.Targets(),
				MiseCurrent: true,
			})
			if err != nil {
				t.Fatalf("Probe: %v", err)
			}
			bin := strings.SplitN(path, ":", 2)[1]
			want := []Measurement{
				{Dir: ".", Tool: "node", Status: StatusFound, Path: filepath.Join(bin, "node"), Version: "v24.22.1"},
				{Dir: ".", Tool: "pnpm", Status: StatusFound, Path: filepath.Join(shims, "pnpm"), Shim: ShimMise, Version: "10.10.0"},
				{Dir: ".", Tool: "python3", Status: StatusFound, Path: filepath.Join(bin, "python3"), Version: "3.12.1"},
				{Dir: ".", Tool: "go", Status: StatusUnknown, Path: filepath.Join(bin, "go"), Reason: "no version in the output: something went wrong"},
				{Dir: "apps/b", Tool: "node", Status: StatusFound, Path: filepath.Join(bin, "node"), Version: "v24.22.1"},
				{Dir: "apps/b", Tool: "yarn", Status: StatusMissing},
				{Dir: "apps/my dir", Tool: "node", Status: StatusFound, Path: filepath.Join(bin, "node"), Version: "v22.1.0"},
				{Dir: "apps/my dir", Tool: "pnpm", Status: StatusFound, Path: filepath.Join(shims, "pnpm"), Shim: ShimMise, Version: "10.10.0"},
			}
			if !reflect.DeepEqual(rep.Measurements, want) {
				t.Errorf("measurements\n got %+v\nwant %+v", rep.Measurements, want)
			}
			wantMise := &MiseCurrent{Path: filepath.Join(bin, "mise"), Lines: []string{"node 24.22.1", "python 3.12.1"}, ExitCode: 0}
			if !reflect.DeepEqual(rep.Mise, wantMise) {
				t.Errorf("Mise = %+v, want %+v", rep.Mise, wantMise)
			}
			if !rep.Complete || rep.TimedOut || rep.Failure != "" {
				t.Errorf("Complete=%v TimedOut=%v Failure=%q; want a complete run", rep.Complete, rep.TimedOut, rep.Failure)
			}
		})
	}
}

func TestProbe_PlaceItCannotEnterIsUnknown(t *testing.T) {
	path, _ := fakeTools(t)
	root := t.TempDir()
	rep, err := Probe(context.Background(), ProbeOptions{
		Argv:    []string{"/bin/sh", "-c"},
		Dir:     root,
		Env:     []string{"PATH=" + path},
		Timeout: 20 * time.Second,
		Targets: []Target{{"gone/it's here", "node"}, {".", "python3"}},
	})
	if err != nil {
		t.Fatalf("Probe: %v", err)
	}
	if m := rep.Measurements[0]; m.Status != StatusUnknown || m.Reason != "cannot cd into the directory" {
		t.Errorf("node in a missing place = %+v, want unknown: cannot cd", m)
	}
	if m := rep.Measurements[1]; m.Status != StatusFound || m.Version != "3.12.1" {
		t.Errorf("python3 at the root = %+v, want found 3.12.1", m)
	}
	if !rep.Complete || rep.Mise != nil {
		t.Errorf("Complete=%v Mise=%+v; want a complete run without mise", rep.Complete, rep.Mise)
	}
}

// listFiles returns every path under root, relative to it.
func listFiles(t *testing.T, root string) []string {
	t.Helper()
	var out []string
	err := filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(root, p)
		out = append(out, rel)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return out
}

func TestProbe_ChangesNothingWithShimsThatWouldPinAndInstall(t *testing.T) {
	base := t.TempDir()
	shims := filepath.Join(base, "mise", "shims")
	markers := filepath.Join(base, "markers")
	if err := os.MkdirAll(markers, 0o755); err != nil {
		t.Fatal(err)
	}
	// A shim that pins its package manager into the place's package.json
	// (corepack's auto pin) and installs a missing version (mise's auto
	// install) unless the variables say not to.
	shim := "#!/bin/sh\n" +
		"if [ \"${COREPACK_ENABLE_AUTO_PIN-}\" != 0 ]; then printf 'pinned\\n' > ./pinned-by-probe; fi\n" +
		"if [ \"${MISE_AUTO_INSTALL-}\" != 0 ]; then printf 'x\\n' > \"$PROBE_MARKERS/installed-${0##*/}\"; fi\n" +
		"echo 10.10.0\n"
	writeExec(t, filepath.Join(shims, "pnpm"), shim)
	writeExec(t, filepath.Join(shims, "node"), strings.Replace(shim, "10.10.0", "v24.22.1", 1))
	root, layout := probeRepo(t)
	env := []string{"PATH=" + shims, "PROBE_MARKERS=" + markers, "COREPACK_ENABLE_AUTO_PIN=1", "MISE_AUTO_INSTALL=1"}
	targets := layout.NodeTargets()

	// The control: run with the caller's environment as it is, and the
	// shims write.
	if _, err := Run(context.Background(), []string{"/bin/sh", "-c"}, BuildScript(targets, false), root, env, 20*time.Second); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if got := listFiles(t, markers); len(got) != 3 {
		t.Fatalf("control run: markers = %q, want the two shims' install markers (the fake shims do not write)", got)
	}
	if _, err := os.Stat(filepath.Join(root, "apps", "my dir", "pinned-by-probe")); err != nil {
		t.Fatalf("control run: the fake shim did not pin: %v", err)
	}

	// The probe, from fresh directories.
	root, layout = probeRepo(t)
	if err := os.RemoveAll(markers); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(markers, 0o755); err != nil {
		t.Fatal(err)
	}
	before := listFiles(t, root)
	rep, err := Probe(context.Background(), ProbeOptions{
		Argv: []string{"/bin/sh", "-c"}, Dir: root, Env: env, Timeout: 20 * time.Second, Targets: layout.NodeTargets(),
	})
	if err != nil {
		t.Fatalf("Probe: %v", err)
	}
	if after := listFiles(t, root); !reflect.DeepEqual(after, before) {
		t.Errorf("the probe changed the repository\nbefore %q\n after %q", before, after)
	}
	if got := listFiles(t, markers); len(got) != 1 {
		t.Errorf("the probe left install markers: %q", got)
	}
	for _, m := range rep.Measurements {
		if m.Status != StatusFound && m.Tool != ToolYarn {
			t.Errorf("%s at %s = %+v, want found", m.Tool, m.Dir, m)
		}
	}
}

func TestProbe_TimeoutKeepsWhatWasMeasured(t *testing.T) {
	setKillGrace(t, 300*time.Millisecond)
	bin := t.TempDir()
	writeExec(t, filepath.Join(bin, "python3"), "#!/bin/sh\necho 'Python 3.12.1'\n")
	writeExec(t, filepath.Join(bin, "node"), "#!/bin/sh\n/bin/sleep 30\necho v24.22.1\n")
	timeout := 1500 * time.Millisecond
	start := time.Now()
	rep, err := Probe(context.Background(), ProbeOptions{
		Argv: []string{"/bin/sh", "-c"}, Dir: t.TempDir(), Env: []string{"PATH=" + bin}, Timeout: timeout,
		Targets: []Target{{".", "python3"}, {".", "node"}, {".", "go"}},
	})
	elapsed := time.Since(start)
	if err != nil {
		t.Fatalf("Probe: %v", err)
	}
	if !rep.TimedOut || rep.Complete || rep.Failure != "timed out after 1.5s" {
		t.Errorf("TimedOut=%v Complete=%v Failure=%q; want a timeout", rep.TimedOut, rep.Complete, rep.Failure)
	}
	if m := rep.Measurements[0]; m.Status != StatusFound || m.Version != "3.12.1" {
		t.Errorf("python3 = %+v, want found before the timeout", m)
	}
	for _, m := range rep.Measurements[1:] {
		if m.Status != StatusUnknown || m.Reason != "not measured: timed out after 1.5s" {
			t.Errorf("%s = %+v, want unknown, not measured: timed out", m.Tool, m)
		}
	}
	// The budget plus the grace; the slack is for a loaded machine, far
	// below the fake node's 30 s sleep.
	if limit := timeout + KillGrace + 5*time.Second; elapsed > limit {
		t.Errorf("Probe took %s, over %s", elapsed, limit)
	}
}

func TestProbe_ShimChildHoldingTheOutputDoesNotStopTheRest(t *testing.T) {
	setKillGrace(t, 300*time.Millisecond)
	bin := t.TempDir()
	pidFile := filepath.Join(t.TempDir(), "child.pid")
	writeExec(t, filepath.Join(bin, "pnpm"), "#!/bin/sh\n/bin/sleep 30 &\necho $! > '"+pidFile+"'\necho 10.10.0\n")
	writeExec(t, filepath.Join(bin, "node"), "#!/bin/sh\necho v24.22.1\n")
	start := time.Now()
	rep, err := Probe(context.Background(), ProbeOptions{
		Argv: []string{"/bin/sh", "-c"}, Dir: t.TempDir(), Env: []string{"PATH=" + bin}, Timeout: 20 * time.Second,
		Targets: []Target{{".", "pnpm"}, {".", "node"}},
	})
	elapsed := time.Since(start)
	if err != nil {
		t.Fatalf("Probe: %v", err)
	}
	if !rep.Complete || rep.TimedOut || rep.Failure != "" {
		t.Errorf("Complete=%v TimedOut=%v Failure=%q; want a complete run", rep.Complete, rep.TimedOut, rep.Failure)
	}
	if m := rep.Measurements[0]; m.Status != StatusFound || m.Version != "10.10.0" {
		t.Errorf("pnpm = %+v, want found", m)
	}
	if m := rep.Measurements[1]; m.Status != StatusFound || m.Version != "v24.22.1" {
		t.Errorf("node after pnpm = %+v, want found", m)
	}
	// Far below both the 20 s budget and the child's 30 s sleep.
	if elapsed > 10*time.Second {
		t.Errorf("Probe took %s: it waited for the child holding the output", elapsed)
	}
	waitProcessGone(t, readPID(t, pidFile))
}

func TestProbe_ShellThatCannotStartIsAFailureNotAnError(t *testing.T) {
	rep, err := Probe(context.Background(), ProbeOptions{
		Argv: []string{filepath.Join(t.TempDir(), "no-such-shell"), "-lc"}, Dir: t.TempDir(), Timeout: 5 * time.Second,
		Targets: []Target{{".", "python3"}},
	})
	if err != nil {
		t.Fatalf("Probe: %v, want no error", err)
	}
	if !strings.HasPrefix(rep.Failure, "could not start ") || rep.Complete {
		t.Errorf("Failure=%q Complete=%v; want could not start", rep.Failure, rep.Complete)
	}
	if m := rep.Measurements[0]; m.Status != StatusUnknown || !strings.HasPrefix(m.Reason, "not measured: could not start ") {
		t.Errorf("python3 = %+v, want unknown, not measured: could not start", m)
	}
}

func TestProbe_ShellThatExitsEarlySaysSoWithItsStderr(t *testing.T) {
	// The script goes after "-c" "<command>" as $0, so the shell runs the
	// command and never the script.
	rep, err := Probe(context.Background(), ProbeOptions{
		Argv: []string{"/bin/sh", "-c", "echo 'profile: broken' >&2; exit 3"}, Dir: t.TempDir(), Timeout: 5 * time.Second,
		Targets: []Target{{".", "python3"}},
	})
	if err != nil {
		t.Fatalf("Probe: %v", err)
	}
	if want := "the shell exited 3 before the probe's end: profile: broken"; rep.Failure != want {
		t.Errorf("Failure = %q, want %q", rep.Failure, want)
	}
	if m := rep.Measurements[0]; m.Reason != "not measured: the shell exited 3 before the probe's end: profile: broken" {
		t.Errorf("python3 reason = %q", m.Reason)
	}
}

func TestProbe_ContextDoneIsAnError(t *testing.T) {
	setKillGrace(t, 300*time.Millisecond)
	bin := t.TempDir()
	writeExec(t, filepath.Join(bin, "node"), "#!/bin/sh\n/bin/sleep 30\n")
	ctx, cancel := context.WithCancel(context.Background())
	time.AfterFunc(300*time.Millisecond, cancel)
	start := time.Now()
	rep, err := Probe(ctx, ProbeOptions{
		Argv: []string{"/bin/sh", "-c"}, Dir: t.TempDir(), Env: []string{"PATH=" + bin}, Timeout: 20 * time.Second,
		Targets: []Target{{".", "node"}},
	})
	if !errors.Is(err, context.Canceled) {
		t.Errorf("Probe error = %v, want context.Canceled", err)
	}
	if rep.Measurements != nil || rep.Failure != "" {
		t.Errorf("Report = %+v, want none after an interrupt", rep)
	}
	if elapsed := time.Since(start); elapsed > 10*time.Second {
		t.Errorf("Probe took %s after the cancel", elapsed)
	}
}

func TestBuildScript_QuotesPlacesAndListsEachToolOnce(t *testing.T) {
	script := BuildScript([]Target{{".", "python3"}, {"it's here", "node"}, {".", "go"}, {"it's here", "pnpm"}}, true)
	for _, want := range []string{
		"cd './it'\\''s here' >/dev/null 2>&1 || {",
		"rtc_tool 'python3' '.' 'python3' '--version'\n",
		"rtc_tool 'go' '.' 'go' 'version'\n",
		"rtc_tool 'node' 'it'\\''s here' 'node' '-v'\n",
		"rtc_tool 'pnpm' 'it'\\''s here' 'pnpm' '-v'\n",
		"rtc_tool 'mise-current' '.' mise current\n",
		// Each version command reads no input even when the shell has some
		// (Run's stdin is already empty, so no behavior test sees this).
		"\"$rtc_t\" \"$@\" </dev/null 2>&1\n",
	} {
		if strings.Count(script, want) != 1 {
			t.Errorf("script has %d of %q, want 1\n%s", strings.Count(script, want), want, script)
		}
	}
	// The root's tools share one subshell, before the other place's.
	if i, j := strings.Index(script, "'go' '.'"), strings.Index(script, "cd './it"); i < 0 || j < 0 || i > j {
		t.Errorf("the root's go is not measured before the other place\n%s", script)
	}
	if strings.Contains(BuildScript(nil, false), "mise current") {
		t.Error("BuildScript without miseCurrent runs mise current")
	}
}
