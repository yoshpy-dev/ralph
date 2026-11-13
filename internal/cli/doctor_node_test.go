package cli

import (
	"context"
	"errors"
	"io/fs"
	"os"
	"os/signal"
	"path/filepath"
	"reflect"
	"strings"
	"syscall"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/yoshpy-dev/ralph/internal/toolchain"
)

// Tests of Check 14, "Node toolchain" (doctor_node.go; plan
// 2026-11-12-gate-tool-versions, AC6 and the doctor part of AC10).

// testDoctorNodeProbe is the doctorNodeProbe TestMain pins: it runs no shell
// and measures nothing, and its report says why.
func testDoctorNodeProbe(context.Context, toolchain.ProbeOptions) (toolchain.Report, error) {
	return toolchain.Report{Failure: "the probe runs no shell in internal/cli's tests (TestMain)"}, nil
}

// setDoctorNodeProbe makes doctorNodeProbe probe until the test ends, when
// TestMain's pin comes back.
func setDoctorNodeProbe(t *testing.T, probe func(context.Context, toolchain.ProbeOptions) (toolchain.Report, error)) {
	t.Helper()
	old := doctorNodeProbe
	t.Cleanup(func() { doctorNodeProbe = old })
	doctorNodeProbe = probe
}

// setDoctorNodeShell makes the check read shell as $SHELL until the test
// ends.
func setDoctorNodeShell(t *testing.T, shell string) {
	t.Helper()
	old := doctorNodeGetenv
	t.Cleanup(func() { doctorNodeGetenv = old })
	doctorNodeGetenv = func(name string) string {
		if name == "SHELL" {
			return shell
		}
		return os.Getenv(name)
	}
}

// fakeNodeProbe records each call and answers each target with its entry in
// measure, or missing. report's Failure and TimedOut are returned as they
// are; err, when set, is returned instead of a report.
type fakeNodeProbe struct {
	calls   []toolchain.ProbeOptions
	measure map[toolchain.Target]toolchain.Measurement
	report  toolchain.Report
	err     error
}

func (f *fakeNodeProbe) probe(_ context.Context, opts toolchain.ProbeOptions) (toolchain.Report, error) {
	f.calls = append(f.calls, opts)
	if f.err != nil {
		return toolchain.Report{}, f.err
	}
	rep := toolchain.Report{Failure: f.report.Failure, TimedOut: f.report.TimedOut, Complete: f.report.Failure == ""}
	for _, tg := range opts.Targets {
		m, ok := f.measure[tg]
		if !ok {
			m = toolchain.Measurement{Dir: tg.Dir, Tool: tg.Tool, Status: toolchain.StatusMissing}
		}
		rep.Measurements = append(rep.Measurements, m)
	}
	return rep, nil
}

// useFakeNodeProbe installs a fakeNodeProbe answering ms, with $SHELL
// /bin/zsh.
func useFakeNodeProbe(t *testing.T, ms ...toolchain.Measurement) *fakeNodeProbe {
	t.Helper()
	f := &fakeNodeProbe{measure: map[toolchain.Target]toolchain.Measurement{}}
	for _, m := range ms {
		f.measure[toolchain.Target{Dir: m.Dir, Tool: m.Tool}] = m
	}
	setDoctorNodeProbe(t, f.probe)
	setDoctorNodeShell(t, "/bin/zsh")
	return f
}

// nodeFound is a tool found at dir with version, resolved to bin.
func nodeFound(dir, tool, version, bin string) toolchain.Measurement {
	return toolchain.Measurement{Dir: dir, Tool: tool, Status: toolchain.StatusFound, Version: version, Path: bin, Shim: toolchain.ShimOf(bin)}
}

// nodeNonGitDir returns a new directory git does not see as part of a
// repository, even when a directory above the temp dir is one.
func nodeNonGitDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("GIT_CEILING_DIRECTORIES", filepath.Dir(dir))
	return dir
}

// writeNodeFiles writes files (slash-separated paths relative to dir).
func writeNodeFiles(t *testing.T, dir string, files map[string]string) {
	t.Helper()
	for rel, content := range files {
		p := filepath.Join(dir, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

// nodeGitRepo writes files into a new git repository and tracks them (git
// add -f, so no global ignore rule leaves one out); nothing is committed.
func nodeGitRepo(t *testing.T, files map[string]string) string {
	t.Helper()
	dir := t.TempDir()
	runMigrationTestGit(t, dir, "init", "--quiet", "--template=")
	writeNodeFiles(t, dir, files)
	args := []string{"add", "-f", "--"}
	for rel := range files {
		args = append(args, rel)
	}
	runMigrationTestGit(t, dir, args...)
	return dir
}

// nodeLines returns the results with status.
func nodeLines(results []checkResult, status string) []string {
	var out []string
	for _, r := range results {
		if r.Status == status {
			out = append(out, r.Detail)
		}
	}
	return out
}

// nodeCheck runs checkNodeToolchain on dir and returns its lines. The tests
// that call it send no interrupt, so an interrupt it reports fails the test.
func nodeCheck(t *testing.T, dir string) []checkResult {
	t.Helper()
	results, interrupted := checkNodeToolchain(dir)
	if interrupted != nil {
		t.Errorf("checkNodeToolchain(%q) reports an interrupt with none sent: %v", dir, interrupted)
	}
	return results
}

// assertNodeResults checks that every result is a "Node toolchain" line
// that never fails, that there are wantWarns warn lines, and that the
// details hold each of wants.
func assertNodeResults(t *testing.T, results []checkResult, wantWarns int, wants ...string) {
	t.Helper()
	var all []string
	for _, r := range results {
		if r.Name != nodeToolchainCheckName {
			t.Errorf("result name = %q, want %q", r.Name, nodeToolchainCheckName)
		}
		if r.Status == "fail" {
			t.Errorf("a Node toolchain line fails (only an interrupt may change doctor's exit code, and it is a warn line): %+v", r)
		}
		all = append(all, r.Status+" | "+r.Detail)
	}
	text := strings.Join(all, "\n")
	if got := len(nodeLines(results, "warn")); got != wantWarns {
		t.Errorf("%d warn lines, want %d:\n%s", got, wantWarns, text)
	}
	for _, w := range wants {
		if !strings.Contains(text, w) {
			t.Errorf("the results lack %q:\n%s", w, text)
		}
	}
}

func TestDoctorNodeToolchain_NoJSPlaceRunsNoProbe(t *testing.T) {
	t.Run("not a git repository", func(t *testing.T) {
		f := useFakeNodeProbe(t)
		dir := nodeNonGitDir(t)
		writeNodeFiles(t, dir, map[string]string{"README.md": "x\n", "main.go": "package main\n", "web/package.json": "{}"})
		results := nodeCheck(t, dir)
		if len(f.calls) != 0 {
			t.Errorf("the probe ran %d time(s) with no Node marker in the top-level files", len(f.calls))
		}
		// The line carries why git could not list the files: here git's
		// exit, the first line of its stderr after it.
		prefix := "no Node markers (package.json, a lockfile, .node-version, .nvmrc) in the top-level files " +
			"(the tracked files could not be listed: git ls-files in " + dir + ": exit status 128: "
		if len(results) != 1 || results[0].Name != nodeToolchainCheckName || results[0].Status != "info" ||
			!strings.HasPrefix(results[0].Detail, prefix) || !strings.HasSuffix(results[0].Detail, "): skipped") {
			t.Errorf("results = %+v, want one info line that starts %q and ends \"): skipped\"", results, prefix)
		}
	})
	t.Run("a git repository with no tracked Node marker", func(t *testing.T) {
		f := useFakeNodeProbe(t)
		dir := nodeGitRepo(t, map[string]string{"go.mod": "module x\n", "cmd/x/main.go": "package main\n"})
		writeNodeFiles(t, dir, map[string]string{"package.json": "{}"}) // untracked
		results := nodeCheck(t, dir)
		if len(f.calls) != 0 {
			t.Errorf("the probe ran %d time(s) with no tracked Node marker", len(f.calls))
		}
		want := []checkResult{{Name: nodeToolchainCheckName, Status: "info",
			Detail: "no Node markers (package.json, a lockfile, .node-version, .nvmrc) in the tracked files: skipped"}}
		if !reflect.DeepEqual(results, want) {
			t.Errorf("results = %+v, want %+v", results, want)
		}
	})
}

func TestDoctorNodeToolchain_VersionFiles(t *testing.T) {
	const bin = "/opt/node/bin/node"
	cases := []struct {
		name      string
		files     map[string]string
		wantWarns int
		wants     []string
	}{
		{"exact .node-version matches", map[string]string{".node-version": "24.22.1\n"}, 0,
			[]string{"pass | /bin/zsh -lc: repo root: node v24.22.1 meets .node-version (24.22.1)"}},
		{"partial .node-version matches", map[string]string{".node-version": "24\n"}, 0,
			[]string{"pass | /bin/zsh -lc: repo root: node v24.22.1 meets .node-version (24)"}},
		{"partial .node-version does not match", map[string]string{".node-version": "24.15"}, 1,
			[]string{"warn | repo root: .node-version asks for node 24.15, but /bin/zsh -lc resolves node v24.22.1 (" + bin + "); " +
				"pin it with MISE_NODE_VERSION=24.15, or put node 24.15 first on PATH"}},
		{"exact .node-version does not match", map[string]string{".node-version": "22.11.0\n"}, 1,
			[]string{"repo root: .node-version asks for node 22.11.0, but", "MISE_NODE_VERSION=22.11.0"}},
		{".nvmrc with v and a trailing newline matches", map[string]string{".nvmrc": "v24\n"}, 0,
			[]string{"pass | /bin/zsh -lc: repo root: node v24.22.1 meets .nvmrc (v24)"}},
		{".nvmrc after a comment line does not match", map[string]string{".nvmrc": "# the CI node\nv22\n"}, 1,
			[]string{"warn | repo root: .nvmrc asks for node v22, but /bin/zsh -lc resolves node v24.22.1 (" + bin + "); " +
				"pin it with MISE_NODE_VERSION=22, or put node 22 first on PATH"}},
		{"an alias is info", map[string]string{".nvmrc": "lts/iron\n"}, 0,
			[]string{`info | /bin/zsh -lc: nothing compared; .nvmrc names the alias "lts/iron", which this check does not compare`}},
		{"a spec that is not a version is info", map[string]string{".nvmrc": "banana\n"}, 0,
			[]string{`.nvmrc holds "banana", which is not a version or a known alias; not compared`}},
		{"an empty file is info", map[string]string{".node-version": "\n"}, 0,
			[]string{".node-version holds no version; not compared"}},
		{"both files in one directory are compared", map[string]string{".node-version": "24", ".nvmrc": "22"}, 1,
			[]string{"warn | repo root: .nvmrc asks for node 22, but"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			useFakeNodeProbe(t, nodeFound(".", "node", "v24.22.1", bin))
			dir := nodeNonGitDir(t)
			writeNodeFiles(t, dir, tc.files)
			assertNodeResults(t, nodeCheck(t, dir), tc.wantWarns, tc.wants...)
		})
	}
}

func TestDoctorNodeToolchain_EnginesNode(t *testing.T) {
	cases := []struct {
		name      string
		pkg       string
		wantWarns int
		wants     []string
	}{
		{"a lower bound", `{"engines":{"node":">=22"}}`, 0, []string{`node v24.22.1 meets engines.node ">=22" in package.json`}},
		{"a caret", `{"engines":{"node":"^24.0.0"}}`, 0, []string{`meets engines.node "^24.0.0"`}},
		{"a tilde", `{"engines":{"node":"~24.22"}}`, 0, []string{`meets engines.node "~24.22"`}},
		{"a hyphen range", `{"engines":{"node":"22 - 24"}}`, 0, []string{`meets engines.node "22 - 24"`}},
		{"an x range", `{"engines":{"node":"24.x"}}`, 0, []string{`meets engines.node "24.x"`}},
		{"an or", `{"engines":{"node":"<20 || >=24.22"}}`, 0, []string{`meets engines.node "<20 || >=24.22"`}},
		{"a lower bound above", `{"engines":{"node":">=25"}}`, 1,
			[]string{`warn | repo root: engines.node ">=25" in package.json does not allow node v24.22.1 (/opt/node/bin/node) ` +
				"that /bin/zsh -lc resolves; pin a version it allows with MISE_NODE_VERSION=<version>, or put such a node first on PATH"}},
		{"a caret below", `{"engines":{"node":"^22"}}`, 1, []string{`engines.node "^22" in package.json does not allow`}},
		{"an and of two", `{"engines":{"node":">=22 <24"}}`, 1, []string{`engines.node ">=22 <24" in package.json does not allow`}},
		{"a range it cannot read is info", `{"engines":{"node":"banana"}}`, 0,
			[]string{`engines.node "banana" in package.json is not a range this check reads; not compared`}},
		{"engines.node not a string is info", `{"engines":{"node":24}}`, 0,
			[]string{"engines.node in package.json is not a string; not compared"}},
		{"engines not an object is info", `{"engines":"node24"}`, 0,
			[]string{"engines.node in package.json cannot be read: engines is not an object; not compared"}},
		{"engines null is nothing", `{"engines":null}`, 0, []string{"info | /bin/zsh -lc: nothing compared;", "resolved repo root: node v24.22.1, npm missing"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			useFakeNodeProbe(t, nodeFound(".", "node", "v24.22.1", "/opt/node/bin/node"))
			dir := nodeNonGitDir(t)
			writeNodeFiles(t, dir, map[string]string{"package.json": tc.pkg})
			assertNodeResults(t, nodeCheck(t, dir), tc.wantWarns, tc.wants...)
		})
	}
}

func TestDoctorNodeToolchain_PackageManager(t *testing.T) {
	cases := []struct {
		name      string
		pkg       string
		lockfile  string
		measured  toolchain.Measurement
		wantWarns int
		wants     []string
	}{
		{"pnpm with a hash matches", `{"packageManager":"pnpm@10.10.0+sha512.0123abcd"}`, "pnpm-lock.yaml",
			nodeFound(".", "pnpm", "10.10.0", "/opt/pnpm/bin/pnpm"), 0,
			[]string{"pnpm 10.10.0 meets packageManager pnpm@10.10.0 in package.json"}},
		{"pnpm with a hash does not match", `{"packageManager":"pnpm@10.10.0+sha512.0123abcd"}`, "pnpm-lock.yaml",
			nodeFound(".", "pnpm", "9.15.0", "/Users/u/.local/share/mise/shims/pnpm"), 1,
			[]string{"warn | repo root: packageManager pnpm@10.10.0 in package.json, but /bin/zsh -lc resolves pnpm 9.15.0 " +
				"(/Users/u/.local/share/mise/shims/pnpm, a mise shim); run corepack enable so pnpm follows packageManager, " +
				"or pin it with MISE_PNPM_VERSION=10.10.0"}},
		{"yarn matches", `{"packageManager":"yarn@4.1.0"}`, "yarn.lock",
			nodeFound(".", "yarn", "4.1.0", "/opt/yarn/bin/yarn"), 0, []string{"yarn 4.1.0 meets packageManager yarn@4.1.0 in package.json"}},
		{"yarn does not match", `{"packageManager":"yarn@4.1.0+sha224.0123abcd"}`, "yarn.lock",
			nodeFound(".", "yarn", "1.22.22", "/opt/yarn/bin/yarn"), 1,
			[]string{"resolves yarn 1.22.22 (/opt/yarn/bin/yarn); run corepack enable so yarn", "MISE_YARN_VERSION=4.1.0"}},
		{"npm does not match", `{"packageManager":"npm@10.9.0"}`, "package-lock.json",
			nodeFound(".", "npm", "10.8.2", "/opt/node/bin/npm"), 1,
			[]string{"resolves npm 10.8.2 (/opt/node/bin/npm); run corepack enable npm so npm follows packageManager", "MISE_NPM_VERSION=10.9.0"}},
		{"npm from packageManager alone matches", `{"packageManager":"npm@10.9.0"}`, "",
			nodeFound(".", "npm", "10.9.0", "/opt/node/bin/npm"), 0, []string{"npm 10.9.0 meets packageManager npm@10.9.0 in package.json"}},
		{"a name the lockfile does not use", `{"packageManager":"yarn@4.1.0"}`, "pnpm-lock.yaml",
			nodeFound(".", "pnpm", "10.10.0", "/opt/pnpm/bin/pnpm"), 1,
			[]string{"warn | repo root: packageManager yarn@4.1.0 in package.json names yarn, but the place's package manager is pnpm " +
				"(from pnpm-lock.yaml); make packageManager and the lockfile agree"}},
		{"no full version is info", `{"packageManager":"pnpm@latest"}`, "pnpm-lock.yaml",
			nodeFound(".", "pnpm", "10.10.0", "/opt/pnpm/bin/pnpm"), 0,
			[]string{`packageManager "pnpm@latest" in package.json is not <name>@<full version>; not compared`}},
		{"a manager it does not measure is info", `{"packageManager":"bun@1.1.0"}`, "",
			nodeFound(".", "npm", "10.9.0", "/opt/node/bin/npm"), 0,
			[]string{"packageManager bun@1.1.0 in package.json names a package manager this check does not measure"}},
		{"not a string is info", `{"packageManager":10}`, "pnpm-lock.yaml",
			nodeFound(".", "pnpm", "10.10.0", "/opt/pnpm/bin/pnpm"), 0,
			[]string{"packageManager in package.json is not a string; not compared"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			useFakeNodeProbe(t, nodeFound(".", "node", "v24.22.1", "/opt/node/bin/node"), tc.measured)
			dir := nodeNonGitDir(t)
			files := map[string]string{"package.json": tc.pkg}
			if tc.lockfile != "" {
				files[tc.lockfile] = ""
			}
			writeNodeFiles(t, dir, files)
			assertNodeResults(t, nodeCheck(t, dir), tc.wantWarns, tc.wants...)
		})
	}
}

// The nearest .node-version or .nvmrc at or above a place is its spec: the
// root's for apps/a, apps/b's .nvmrc (not the root's) for apps/b/c.
func TestDoctorNodeToolchain_NearestVersionFileInAParentDirectory(t *testing.T) {
	f := useFakeNodeProbe(t,
		nodeFound(".", "node", "v22.11.0", "/opt/node22/bin/node"),
		nodeFound("apps/a", "node", "v24.22.1", "/opt/node24/bin/node"),
		nodeFound("apps/b", "node", "v24.22.1", "/opt/node24/bin/node"),
		nodeFound("apps/b/c", "node", "v24.22.1", "/opt/node24/bin/node"),
	)
	dir := nodeGitRepo(t, map[string]string{
		".node-version":           "22\n",
		"apps/a/pnpm-lock.yaml":   "",
		"apps/a/package.json":     "{}",
		"apps/b/.nvmrc":           "24\n",
		"apps/b/c/pnpm-lock.yaml": "",
	})
	results := nodeCheck(t, dir)
	assertNodeResults(t, results, 1,
		"warn | apps/a: .node-version asks for node 22, but /bin/zsh -lc resolves node v24.22.1 (/opt/node24/bin/node); "+
			"pin it with MISE_NODE_VERSION=22, or put node 22 first on PATH")
	if len(f.calls) != 1 {
		t.Fatalf("the probe ran %d times, want once", len(f.calls))
	}
	var dirs []string
	for _, tg := range f.calls[0].Targets {
		dirs = append(dirs, tg.Dir+" "+tg.Tool)
	}
	if want := []string{". node", ". npm", "apps/a node", "apps/a pnpm", "apps/b node", "apps/b npm", "apps/b/c node", "apps/b/c pnpm"}; !reflect.DeepEqual(dirs, want) {
		t.Errorf("targets = %q, want %q", dirs, want)
	}

	// With apps/a on node 22 every place meets its own spec.
	useFakeNodeProbe(t,
		nodeFound(".", "node", "v22.11.0", "/opt/node22/bin/node"),
		nodeFound("apps/a", "node", "v22.11.0", "/opt/node22/bin/node"),
		nodeFound("apps/b", "node", "v24.22.1", "/opt/node24/bin/node"),
		nodeFound("apps/b/c", "node", "v24.22.1", "/opt/node24/bin/node"),
	)
	assertNodeResults(t, nodeCheck(t, dir), 0,
		"pass | /bin/zsh -lc: repo root: node v22.11.0 meets .node-version (22); apps/a: node v22.11.0 meets .node-version (22); "+
			"apps/b: node v24.22.1 meets apps/b/.nvmrc (24); apps/b/c: node v24.22.1 meets apps/b/.nvmrc (24)")
}

// A monorepo whose apps use different package managers: the one place whose
// package manager does not match gets the one warn line, naming it.
func TestDoctorNodeToolchain_MonorepoWarnsForThePlaceThatDiffers(t *testing.T) {
	const yarnShim = "/Users/u/.local/share/mise/shims/yarn"
	useFakeNodeProbe(t,
		nodeFound(".", "node", "v24.22.1", "/opt/node/bin/node"),
		nodeFound(".", "npm", "10.9.0", "/opt/node/bin/npm"),
		nodeFound("apps/a", "node", "v24.22.1", "/opt/node/bin/node"),
		nodeFound("apps/a", "pnpm", "10.10.0", "/opt/pnpm/bin/pnpm"),
		nodeFound("apps/b", "node", "v24.22.1", "/opt/node/bin/node"),
		nodeFound("apps/b", "yarn", "1.22.22", yarnShim),
	)
	dir := nodeGitRepo(t, map[string]string{
		"package.json":          `{"private":true}`,
		"apps/a/package.json":   `{"packageManager":"pnpm@10.10.0"}`,
		"apps/a/pnpm-lock.yaml": "",
		"apps/b/package.json":   `{"packageManager":"yarn@4.1.0+sha512.0123abcd"}`,
		"apps/b/yarn.lock":      "",
	})
	results := nodeCheck(t, dir)
	assertNodeResults(t, results, 1,
		"warn | apps/b: packageManager yarn@4.1.0 in apps/b/package.json, but /bin/zsh -lc resolves yarn 1.22.22 ("+yarnShim+
			", a mise shim); run corepack enable so yarn follows packageManager, or pin it with MISE_YARN_VERSION=4.1.0")
	for _, w := range nodeLines(results, "warn") {
		if strings.Contains(w, "apps/a") {
			t.Errorf("a warn line names apps/a, whose pnpm matches: %s", w)
		}
	}
}

// A place without its own package.json field reads the root's, and a
// package.json that is not JSON is info, its fields read from the root's.
func TestDoctorNodeToolchain_BrokenPackageJSONFallsBackToTheRoot(t *testing.T) {
	useFakeNodeProbe(t,
		nodeFound(".", "node", "v25.1.0", "/opt/node25/bin/node"),
		nodeFound("apps/a", "node", "v24.22.1", "/opt/node24/bin/node"),
	)
	dir := nodeGitRepo(t, map[string]string{
		"package.json":          `{"engines":{"node":">=25"}}`,
		"apps/a/package.json":   "{ not json",
		"apps/a/pnpm-lock.yaml": "",
	})
	assertNodeResults(t, nodeCheck(t, dir), 1,
		`warn | apps/a: engines.node ">=25" in package.json does not allow node v24.22.1 (/opt/node24/bin/node)`,
		"info | apps/a/package.json is not a JSON object; its engines.node and packageManager are not compared")

	for _, content := range []string{"{", "[]", "null", ""} {
		useFakeNodeProbe(t, nodeFound(".", "node", "v24.22.1", "/opt/node/bin/node"))
		dir := nodeNonGitDir(t)
		writeNodeFiles(t, dir, map[string]string{"package.json": content, ".nvmrc": "24"})
		assertNodeResults(t, nodeCheck(t, dir), 0,
			"pass | /bin/zsh -lc: repo root: node v24.22.1 meets .nvmrc (24); "+
				"package.json is not a JSON object; its engines.node and packageManager are not compared")
	}
}

// A place's own package.json field wins over the root's: apps/a's
// engines.node and packageManager, not the root's, are its requirements.
func TestDoctorNodeToolchain_PlaceOwnPackageJSONFieldWins(t *testing.T) {
	useFakeNodeProbe(t,
		nodeFound(".", "node", "v25.1.0", "/opt/node25/bin/node"),
		nodeFound(".", "pnpm", "10.10.0", "/opt/pnpm/bin/pnpm"),
		nodeFound("apps/a", "node", "v24.22.1", "/opt/node24/bin/node"),
		nodeFound("apps/a", "pnpm", "9.15.0", "/opt/pnpm9/bin/pnpm"),
	)
	dir := nodeGitRepo(t, map[string]string{
		"package.json":          `{"engines":{"node":">=25"},"packageManager":"pnpm@10.10.0"}`,
		"pnpm-lock.yaml":        "",
		"apps/a/package.json":   `{"engines":{"node":"^24"},"packageManager":"pnpm@9.15.0"}`,
		"apps/a/pnpm-lock.yaml": "",
	})
	assertNodeResults(t, nodeCheck(t, dir), 0,
		`pass | /bin/zsh -lc: repo root: node v25.1.0 meets engines.node ">=25" in package.json; `+
			"repo root: pnpm 10.10.0 meets packageManager pnpm@10.10.0 in package.json; "+
			`apps/a: node v24.22.1 meets engines.node "^24" in apps/a/package.json; `+
			"apps/a: pnpm 9.15.0 meets packageManager pnpm@9.15.0 in apps/a/package.json")
}

func TestDoctorNodeToolchain_ProbeFailureAndTimeoutWarn(t *testing.T) {
	files := map[string]string{"package.json": `{"engines":{"node":">=22"},"packageManager":"pnpm@10.10.0"}`, "pnpm-lock.yaml": ""}
	t.Run("timeout", func(t *testing.T) {
		f := useFakeNodeProbe(t,
			nodeFound(".", "node", "v24.22.1", "/opt/node/bin/node"),
			toolchain.Measurement{Dir: ".", Tool: "pnpm", Status: toolchain.StatusUnknown, Reason: "not measured: timed out after 20s"},
		)
		f.report = toolchain.Report{Failure: "timed out after 20s", TimedOut: true}
		dir := nodeNonGitDir(t)
		writeNodeFiles(t, dir, files)
		assertNodeResults(t, nodeCheck(t, dir), 1,
			"warn | /bin/zsh -lc did not finish measuring: timed out after 20s; the tools it did not reach are not compared")
	})
	t.Run("the shell fails", func(t *testing.T) {
		f := useFakeNodeProbe(t)
		f.report = toolchain.Report{Failure: "the shell exited 1 before the probe's end: .zprofile:3: command not found: brew"}
		f.measure[toolchain.Target{Dir: ".", Tool: "node"}] = toolchain.Measurement{Dir: ".", Tool: "node", Status: toolchain.StatusUnknown,
			Reason: "not measured: the shell exited 1 before the probe's end"}
		f.measure[toolchain.Target{Dir: ".", Tool: "pnpm"}] = toolchain.Measurement{Dir: ".", Tool: "pnpm", Status: toolchain.StatusUnknown,
			Reason: "not measured: the shell exited 1 before the probe's end"}
		dir := nodeNonGitDir(t)
		writeNodeFiles(t, dir, files)
		assertNodeResults(t, nodeCheck(t, dir), 1,
			"warn | /bin/zsh -lc did not finish measuring: the shell exited 1 before the probe's end: .zprofile:3: command not found: brew")
	})
	t.Run("the probe is stopped", func(t *testing.T) {
		f := useFakeNodeProbe(t)
		f.err = context.Canceled
		dir := nodeNonGitDir(t)
		writeNodeFiles(t, dir, files)
		results, interrupted := checkNodeToolchain(dir)
		assertNodeResults(t, results, 1,
			"warn | /bin/zsh -lc was interrupted (context canceled) before it finished measuring: nothing was compared")
		if !errors.Is(interrupted, context.Canceled) {
			t.Errorf("interrupted = %v, want the probe's error, %v", interrupted, context.Canceled)
		}
	})
}

func TestDoctorNodeToolchain_MissingOrUnreadableToolWarns(t *testing.T) {
	useFakeNodeProbe(t,
		toolchain.Measurement{Dir: ".", Tool: "node", Status: toolchain.StatusMissing},
		toolchain.Measurement{Dir: ".", Tool: "pnpm", Status: toolchain.StatusUnknown, Path: "/usr/local/bin/pnpm",
			Reason: "exit 1: Corepack is about to download pnpm"},
	)
	dir := nodeNonGitDir(t)
	writeNodeFiles(t, dir, map[string]string{
		"package.json":   `{"engines":{"node":">=22"},"packageManager":"pnpm@10.10.0"}`,
		"pnpm-lock.yaml": "",
		".nvmrc":         "lts/*\n",
	})
	assertNodeResults(t, nodeCheck(t, dir), 2,
		`warn | repo root: .nvmrc (lts/*) and engines.node ">=22" in package.json asks for node, but /bin/zsh -lc finds no node on PATH; `+
			"install it, or put it first on PATH for a non-interactive login shell",
		"warn | repo root: packageManager pnpm@10.10.0 in package.json asks for pnpm, but /bin/zsh -lc could not read the version of pnpm "+
			"(/usr/local/bin/pnpm): exit 1: Corepack is about to download pnpm",
		`info | .nvmrc names the alias "lts/*", which this check does not compare`)
}

// A tool the login shell resolves to an alias or a function is not run by the
// probe, and the warn line ends with what to change. It is still one warn
// line (assertNodeResults also checks that no line fails).
func TestDoctorNodeToolchain_NotAFileWarnsWithAHint(t *testing.T) {
	const hint = "; make node resolve to a file in a non-interactive login shell (an alias or function there is not measured)"
	cases := []struct{ name, path, reason string }{
		{"alias", "alias node='/x'", "an alias in the shell, not run (the probe runs files only)"},
		{"function", "node", "not a file (a shell function or builtin), not run (the probe runs files only)"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			useFakeNodeProbe(t, toolchain.Measurement{Dir: ".", Tool: "node", Status: toolchain.StatusUnknown, Path: tc.path, Reason: tc.reason})
			dir := nodeNonGitDir(t)
			writeNodeFiles(t, dir, map[string]string{".nvmrc": "24"})
			assertNodeResults(t, nodeCheck(t, dir), 1,
				"warn | repo root: .nvmrc (24) asks for node, but /bin/zsh -lc could not read the version of node ("+tc.path+"): "+tc.reason+hint)
		})
	}
}

// A node that prints a version that is not a full one ("v24") cannot be
// compared with either form, and each says so in a warn line.
func TestDoctorNodeToolchain_NodeVersionNotFullWarns(t *testing.T) {
	useFakeNodeProbe(t, nodeFound(".", "node", "v24", "/opt/node/bin/node"))
	dir := nodeNonGitDir(t)
	writeNodeFiles(t, dir, map[string]string{".nvmrc": "24", "package.json": `{"engines":{"node":">=22"}}`})
	assertNodeResults(t, nodeCheck(t, dir), 2,
		"warn | repo root: .nvmrc (24) asks for node, but the version /bin/zsh -lc resolves, v24 (/opt/node/bin/node), is not a full version: ",
		`warn | repo root: engines.node ">=22" in package.json asks for node, but the version /bin/zsh -lc resolves, v24 (/opt/node/bin/node), `+
			"is not a full version: ")
}

func TestDoctorNodeToolchain_ShellChoice(t *testing.T) {
	cases := []struct {
		shell, wantArgv0, wantDesc string
	}{
		{"", "/bin/sh", "/bin/sh -lc ($SHELL is empty)"},
		{"/opt/homebrew/bin/fish", "/bin/sh",
			"/bin/sh -lc ($SHELL /opt/homebrew/bin/fish is not an sh-compatible shell: sh, bash, zsh, dash, ksh, mksh, yash)"},
		{"/bin/bash", "/bin/bash", "/bin/bash -lc"},
		{"/usr/bin/zsh", "/usr/bin/zsh", "/usr/bin/zsh -lc"},
		{"/bin/dash", "/bin/dash", "/bin/dash -lc"},
	}
	for _, tc := range cases {
		t.Run(tc.shell, func(t *testing.T) {
			f := useFakeNodeProbe(t, nodeFound(".", "node", "v24.22.1", "/opt/node/bin/node"))
			setDoctorNodeShell(t, tc.shell)
			dir := nodeNonGitDir(t)
			writeNodeFiles(t, dir, map[string]string{".nvmrc": "24"})
			results := nodeCheck(t, dir)
			if len(f.calls) != 1 {
				t.Fatalf("the probe ran %d times, want once", len(f.calls))
			}
			if got, want := f.calls[0].Argv, []string{tc.wantArgv0, "-lc"}; !reflect.DeepEqual(got, want) {
				t.Errorf("argv = %q, want %q", got, want)
			}
			assertNodeResults(t, results, 0, "pass | "+tc.wantDesc+": repo root: node v24.22.1 meets .nvmrc (24)")

			// A warn line names the shell too.
			useFakeNodeProbe(t, nodeFound(".", "node", "v22.1.0", "/opt/node/bin/node"))
			setDoctorNodeShell(t, tc.shell)
			assertNodeResults(t, nodeCheck(t, dir), 1, "but "+tc.wantDesc+" resolves node v22.1.0")
		})
	}
}

// The doctor part of AC10: the environment the probe receives is
// toolchain.ProbeEnv's, so the nine variables that keep it from installing,
// downloading, pinning, or switching anything override the user's values.
func TestDoctorNodeToolchain_ProbeEnvForcesTheNineVariables(t *testing.T) {
	t.Setenv("COREPACK_ENABLE_AUTO_PIN", "1")
	t.Setenv("MISE_AUTO_INSTALL", "1")
	t.Setenv("GOTOOLCHAIN", "auto")
	t.Setenv("NPM_CONFIG_UPDATE_NOTIFIER", "true")
	t.Setenv("pnpm_config_manage_package_manager_versions", "true")
	t.Setenv("RALPH_NODE_TEST_KEPT", "kept")
	f := useFakeNodeProbe(t, nodeFound(".", "node", "v24.22.1", "/opt/node/bin/node"))
	dir := nodeNonGitDir(t)
	writeNodeFiles(t, dir, map[string]string{"package.json": `{"engines":{"node":">=22"}}`})
	nodeCheck(t, dir)
	if len(f.calls) != 1 {
		t.Fatalf("the probe ran %d times, want once", len(f.calls))
	}
	opts := f.calls[0]
	values := map[string][]string{}
	for _, kv := range opts.Env {
		name, value, _ := strings.Cut(kv, "=")
		values[name] = append(values[name], value)
	}
	for _, kv := range []string{
		"MISE_AUTO_INSTALL=0", "MISE_NOT_FOUND_AUTO_INSTALL=0", "COREPACK_ENABLE_AUTO_PIN=0", "COREPACK_ENABLE_NETWORK=0",
		"COREPACK_ENABLE_DOWNLOAD_PROMPT=0", "NPM_CONFIG_UPDATE_NOTIFIER=false", "GOTOOLCHAIN=local",
		"pnpm_config_manage_package_manager_versions=false", "npm_config_manage_package_manager_versions=false",
	} {
		name, value, _ := strings.Cut(kv, "=")
		if got := values[name]; !reflect.DeepEqual(got, []string{value}) {
			t.Errorf("the probe's environment has %s = %q, want only %q", name, got, value)
		}
	}
	if got := values["RALPH_NODE_TEST_KEPT"]; !reflect.DeepEqual(got, []string{"kept"}) {
		t.Errorf("the probe's environment has RALPH_NODE_TEST_KEPT = %q, want the caller's %q", got, "kept")
	}
	if opts.Dir != dir {
		t.Errorf("Dir = %q, want %q", opts.Dir, dir)
	}
	if opts.Timeout != 20*time.Second || doctorNodeTimeout != 20*time.Second {
		t.Errorf("Timeout = %v (doctorNodeTimeout %v), want 20s", opts.Timeout, doctorNodeTimeout)
	}
	if want := []toolchain.Target{{Dir: ".", Tool: "node"}, {Dir: ".", Tool: "npm"}}; !reflect.DeepEqual(opts.Targets, want) {
		t.Errorf("Targets = %+v, want %+v", opts.Targets, want)
	}
	if opts.MiseCurrent {
		t.Error("MiseCurrent is set; doctor does not run mise current")
	}
}

// Outside a git repository the places come from the directory's own
// top-level files: apps/x's yarn.lock is not read, and the line says so.
func TestDoctorNodeToolchain_NotAGitRepositoryReadsTheTopLevelFiles(t *testing.T) {
	f := useFakeNodeProbe(t,
		nodeFound(".", "node", "v24.22.1", "/opt/node/bin/node"),
		nodeFound(".", "pnpm", "10.10.0", "/opt/pnpm/bin/pnpm"),
	)
	dir := nodeNonGitDir(t)
	writeNodeFiles(t, dir, map[string]string{
		"package.json":        `{"packageManager":"pnpm@10.10.0"}`,
		"pnpm-lock.yaml":      "",
		"apps/x/package.json": `{"packageManager":"yarn@4.1.0"}`,
		"apps/x/yarn.lock":    "",
	})
	assertNodeResults(t, nodeCheck(t, dir), 0,
		"pass | /bin/zsh -lc: repo root: pnpm 10.10.0 meets packageManager pnpm@10.10.0 in package.json; "+
			"the tracked files could not be listed (git ls-files in "+dir+": exit status 128: ",
		"), so only the top-level files were read")
	if want := []toolchain.Target{{Dir: ".", Tool: "node"}, {Dir: ".", Tool: "pnpm"}}; len(f.calls) != 1 || !reflect.DeepEqual(f.calls[0].Targets, want) {
		t.Errorf("probe calls = %+v, want one with targets %+v", f.calls, want)
	}
}

// When git ls-files runs past toolchain.LsFilesTimeout, the line says so,
// and a non-git directory says git's exit: the two read differently. The
// reason stays on one line: a control character in it (here a line break in
// the directory's name) becomes "?".
func TestDoctorNodeToolchain_ListingFailureSaysWhy(t *testing.T) {
	t.Run("git ls-files times out", func(t *testing.T) {
		old := toolchain.LsFilesTimeout
		t.Cleanup(func() { toolchain.LsFilesTimeout = old })
		toolchain.LsFilesTimeout = time.Nanosecond
		useFakeNodeProbe(t, nodeFound(".", "node", "v24.22.1", "/opt/node/bin/node"))
		dir := nodeGitRepo(t, map[string]string{".nvmrc": "24"})
		assertNodeResults(t, nodeCheck(t, dir), 0,
			"the tracked files could not be listed (git ls-files in "+dir+": context deadline exceeded), so only the top-level files were read")
	})
	t.Run("a line break in the directory's name", func(t *testing.T) {
		useFakeNodeProbe(t)
		dir := filepath.Join(nodeNonGitDir(t), "a\nb")
		writeNodeFiles(t, dir, map[string]string{"README.md": "x\n"})
		results := nodeCheck(t, dir)
		want := "(the tracked files could not be listed: git ls-files in " + filepath.Dir(dir) + "/a?b: exit status 128: "
		if len(results) != 1 || !strings.Contains(results[0].Detail, want) || strings.ContainsAny(results[0].Detail, "\n\r") {
			t.Errorf("results = %q, want one line with %q", results, want)
		}
	})
}

func TestNodeErrText_OneShortLine(t *testing.T) {
	cases := []struct {
		name, in, want string
	}{
		{"as it is", "git ls-files in /r: context deadline exceeded", "git ls-files in /r: context deadline exceeded"},
		{"control characters and invalid UTF-8", "a\nb\tc\x1b[0m\xffd\u0085e", "a?b?c?[0m?d?e"},
		{"at the cap", strings.Repeat("x", maxNodeErrBytes), strings.Repeat("x", maxNodeErrBytes)},
		{"over the cap", strings.Repeat("x", maxNodeErrBytes+1), strings.Repeat("x", maxNodeErrBytes-3) + "..."},
		// 3-byte characters after one byte: the cut, which would fall inside
		// a character, moves back to that character's start.
		{"over the cap in the middle of a character", "x" + strings.Repeat("あ", 100),
			"x" + strings.Repeat("あ", (maxNodeErrBytes-3-1)/3) + "..."},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := nodeErrText(errors.New(tc.in))
			if got != tc.want {
				t.Errorf("nodeErrText(%q) = %q, want %q", tc.in, got, tc.want)
			}
			if len(got) > maxNodeErrBytes || !utf8.ValidString(got) {
				t.Errorf("nodeErrText(%q) = %q: %d bytes (cap %d), valid UTF-8 %v", tc.in, got, len(got), maxNodeErrBytes, utf8.ValidString(got))
			}
		})
	}
}

// setDoctorNodeSignalContext makes doctorNodeSignalContext f until the test
// ends.
func setDoctorNodeSignalContext(t *testing.T, f func(context.Context) (context.Context, context.CancelFunc)) {
	t.Helper()
	old := doctorNodeSignalContext
	t.Cleanup(func() { doctorNodeSignalContext = old })
	doctorNodeSignalContext = f
}

// An interrupt while the probe runs stops it: the probe's process group,
// which a terminal's Ctrl-C does not reach, gets SIGTERM, the check returns
// at once with one warn line that says the probe was interrupted and why and
// with the interrupt's cause as its error, and nothing of the group runs on
// (the fake node, stopped in its sleep, never writes the file it writes after
// it). Through the seam, the context ends; with SIGINT to the process, the
// default context (signal.NotifyContext) does. The fake shell runs the script with
// /bin/sh -c, as in TestDoctorNodeToolchain_RealShellProbe.
func TestDoctorNodeToolchain_InterruptStopsTheProbe(t *testing.T) {
	cases := []struct {
		name  string
		cause string
		// setup runs before the check and returns the interrupt.
		setup func(t *testing.T) (interrupt func())
	}{
		{"the context ends", "test stopped", func(t *testing.T) func() {
			stopped := make(chan struct{})
			setDoctorNodeSignalContext(t, func(parent context.Context) (context.Context, context.CancelFunc) {
				ctx, cancel := context.WithCancelCause(parent)
				go func() {
					select {
					case <-stopped:
						cancel(errors.New("test stopped"))
					case <-ctx.Done():
					}
				}()
				return ctx, func() { cancel(nil) }
			})
			return func() { close(stopped) }
		}},
		{"SIGINT to the process", "interrupt signal received", func(t *testing.T) func() {
			return func() {
				// A channel of the test's own is registered first, so a
				// SIGINT the check does not catch cannot end the test binary.
				caught := make(chan os.Signal, 1)
				signal.Notify(caught, syscall.SIGINT)
				t.Cleanup(func() { signal.Stop(caught) })
				if err := syscall.Kill(os.Getpid(), syscall.SIGINT); err != nil {
					t.Fatalf("send SIGINT to the test process: %v", err)
				}
			}
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			oldGrace := toolchain.KillGrace
			t.Cleanup(func() { toolchain.KillGrace = oldGrace })
			toolchain.KillGrace = 300 * time.Millisecond
			binDir, marks := t.TempDir(), t.TempDir()
			started, late := filepath.Join(marks, "started"), filepath.Join(marks, "late")
			writeNodeExec(t, filepath.Join(binDir, "node"), "#!/bin/sh\n: > '"+started+"'\n/bin/sleep 2\n: > '"+late+"'\necho v24.22.1\n")
			shell := filepath.Join(t.TempDir(), "sh")
			writeNodeExec(t, shell, "#!/bin/sh\n[ \"$1\" = -lc ] || exit 97\nexec /bin/sh -c \"$2\"\n")
			setDoctorNodeProbe(t, toolchain.Probe)
			setDoctorNodeShell(t, shell)
			t.Setenv("PATH", binDir+":/usr/bin:/bin")
			t.Setenv("BASH_ENV", "")
			t.Setenv("ENV", "")
			dir := nodeNonGitDir(t)
			writeNodeFiles(t, dir, map[string]string{".nvmrc": "24"})
			interrupt := tc.setup(t)

			type checked struct {
				results     []checkResult
				interrupted error
			}
			done := make(chan checked, 1)
			go func() {
				results, interrupted := checkNodeToolchain(dir)
				done <- checked{results, interrupted}
			}()
			waitForGateCLIFile(t, started)
			startedAt := time.Now()
			interrupt()
			var got checked
			select {
			case got = <-done:
			case <-time.After(10 * time.Second):
				t.Fatal("the check did not return within 10s of the interrupt")
			}
			want := []checkResult{{Name: nodeToolchainCheckName, Status: "warn",
				Detail: shell + " -lc was interrupted (" + tc.cause + ") before it finished measuring: nothing was compared"}}
			if !reflect.DeepEqual(got.results, want) {
				t.Errorf("results = %+v, want %+v", got.results, want)
			}
			if got.interrupted == nil || got.interrupted.Error() != tc.cause {
				t.Errorf("interrupted = %v, want the interrupt's cause %q", got.interrupted, tc.cause)
			}
			// Past the fake node's sleep, with room for a loaded machine.
			time.Sleep(time.Until(startedAt.Add(4 * time.Second)))
			if _, err := os.Stat(late); !errors.Is(err, fs.ErrNotExist) {
				t.Errorf("the fake node wrote %s after the interrupt (stat: %v): the probe's group ran on", late, err)
			}
		})
	}
}

// writeNodeExec writes an executable file at p.
func writeNodeExec(t *testing.T, p, content string) {
	t.Helper()
	if err := os.WriteFile(p, []byte(content), 0o755); err != nil {
		t.Fatal(err)
	}
}

func TestDoctorNodeToolchain_PlacesBeyondTheCapAreInfo(t *testing.T) {
	old := toolchain.MaxPlaces
	t.Cleanup(func() { toolchain.MaxPlaces = old })
	toolchain.MaxPlaces = 2
	useFakeNodeProbe(t, nodeFound(".", "node", "v24.22.1", "/opt/node/bin/node"), nodeFound("apps/a", "node", "v24.22.1", "/opt/node/bin/node"))
	dir := nodeGitRepo(t, map[string]string{
		".nvmrc": "24", "apps/a/pnpm-lock.yaml": "", "apps/b/pnpm-lock.yaml": "", "apps/c/yarn.lock": "",
	})
	assertNodeResults(t, nodeCheck(t, dir), 0, "not measured: 2 more place(s) beyond the first 2")
}

// The real probe: toolchain.Probe runs the script with a shell named sh on a
// PATH whose node and pnpm are fakes. The shell is a wrapper that runs the
// script with /bin/sh -c, not -lc, so a login profile (macOS's /etc/profile
// runs path_helper) cannot put a real node before the fake one. The fake
// pnpm writes what COREPACK_ENABLE_AUTO_PIN it sees, which must be ProbeEnv's
// 0, not the caller's 1.
func TestDoctorNodeToolchain_RealShellProbe(t *testing.T) {
	binDir := t.TempDir()
	envFile := filepath.Join(t.TempDir(), "pnpm-env.txt")
	writeExec := func(p, content string) {
		t.Helper()
		if err := os.WriteFile(p, []byte(content), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	writeExec(filepath.Join(binDir, "node"), "#!/bin/sh\necho v24.22.1\n")
	writeExec(filepath.Join(binDir, "pnpm"), "#!/bin/sh\nprintf '%s\\n' \"$COREPACK_ENABLE_AUTO_PIN\" > '"+envFile+"'\necho 9.15.0\n")
	shellDir := t.TempDir()
	shell := filepath.Join(shellDir, "sh")
	writeExec(shell, "#!/bin/sh\n[ \"$1\" = -lc ] || exit 97\nexec /bin/sh -c \"$2\"\n")

	setDoctorNodeProbe(t, toolchain.Probe)
	setDoctorNodeShell(t, shell)
	t.Setenv("PATH", binDir+":/usr/bin:/bin")
	t.Setenv("COREPACK_ENABLE_AUTO_PIN", "1")
	t.Setenv("BASH_ENV", "")
	t.Setenv("ENV", "")
	dir := nodeNonGitDir(t)
	writeNodeFiles(t, dir, map[string]string{
		"package.json":   `{"engines":{"node":">=24"},"packageManager":"pnpm@10.10.0+sha512.0123abcd"}`,
		"pnpm-lock.yaml": "",
		".nvmrc":         "v24.22\n",
	})
	assertNodeResults(t, nodeCheck(t, dir), 1,
		"warn | repo root: packageManager pnpm@10.10.0 in package.json, but "+shell+" -lc resolves pnpm 9.15.0 ("+
			filepath.Join(binDir, "pnpm")+"); run corepack enable so pnpm follows packageManager, or pin it with MISE_PNPM_VERSION=10.10.0")
	got, err := os.ReadFile(envFile)
	if err != nil {
		t.Fatalf("the fake pnpm did not run: %v", err)
	}
	if strings.TrimSpace(string(got)) != "0" {
		t.Errorf("the fake pnpm saw COREPACK_ENABLE_AUTO_PIN=%q, want ProbeEnv's 0", strings.TrimSpace(string(got)))
	}

	writeExec(filepath.Join(binDir, "pnpm"), "#!/bin/sh\necho 10.10.0\n")
	assertNodeResults(t, nodeCheck(t, dir), 0,
		"pass | "+shell+" -lc: repo root: node v24.22.1 meets .nvmrc (v24.22); repo root: node v24.22.1 meets engines.node \">=24\" in package.json; "+
			"repo root: pnpm 10.10.0 meets packageManager pnpm@10.10.0 in package.json")
}

// ralph doctor prints the Node toolchain line, and its warn leaves the exit
// code as it is.
func TestDoctorNodeToolchain_DoctorOutputHasTheLine(t *testing.T) {
	dir := nodeNonGitDir(t)
	writeNodeFiles(t, dir, map[string]string{"package.json": `{"engines":{"node":">=22"}}`})

	useFakeNodeProbe(t, nodeFound(".", "node", "v24.22.1", "/opt/node/bin/node"))
	var passOut string
	passErr := captureStdoutText(t, &passOut, func() error { return runDoctorOpts(dir, false) })
	if want := `  ✓ Node toolchain: pass — /bin/zsh -lc: repo root: node v24.22.1 meets engines.node ">=22" in package.json`; !strings.Contains(passOut, want) {
		t.Errorf("doctor output lacks %q:\n%s", want, passOut)
	}

	useFakeNodeProbe(t, nodeFound(".", "node", "v20.1.0", "/opt/node/bin/node"))
	var warnOut string
	warnErr := captureStdoutText(t, &warnOut, func() error { return runDoctorOpts(dir, false) })
	if want := `  ⚠ Node toolchain: warn — repo root: engines.node ">=22" in package.json does not allow node v20.1.0`; !strings.Contains(warnOut, want) {
		t.Errorf("doctor output lacks %q:\n%s", want, warnOut)
	}
	if (passErr == nil) != (warnErr == nil) || (passErr != nil && passErr.Error() != warnErr.Error()) {
		t.Errorf("doctor's error changed with the Node toolchain warn: %v with pass, %v with warn", passErr, warnErr)
	}
}

// doctorLinesButNode returns the check lines of doctor's output, without
// the Node toolchain line.
func doctorLinesButNode(out string) []string {
	var lines []string
	for _, line := range strings.Split(out, "\n") {
		if strings.HasPrefix(line, "  ") && !strings.Contains(line, " Node toolchain: ") {
			lines = append(lines, line)
		}
	}
	return lines
}

// An interrupt while the Node toolchain probe runs fails doctor. Through the
// seam, the probe's context has ended before the probe starts: doctor prints
// every other check's line as it does without the interrupt, the Node
// toolchain line is the interrupt's warn, "All checks passed." is not
// printed, and runDoctorOpts returns an error that says doctor was
// interrupted and wraps the cause. Without the interrupt, doctor ends as
// before: "All checks passed." and nil, or the count of failed checks.
func TestDoctorNodeToolchain_InterruptFailsDoctor(t *testing.T) {
	dir := nodeNonGitDir(t)
	writeNodeFiles(t, dir, map[string]string{"package.json": `{"engines":{"node":">=22"}}`})
	f := useFakeNodeProbe(t, nodeFound(".", "node", "v24.22.1", "/opt/node/bin/node"))
	// As toolchain.Probe does, the probe returns the cause once its context
	// has ended, and otherwise answers like f.
	setDoctorNodeProbe(t, func(ctx context.Context, opts toolchain.ProbeOptions) (toolchain.Report, error) {
		if ctx.Err() != nil {
			return toolchain.Report{}, context.Cause(ctx)
		}
		return f.probe(ctx, opts)
	})

	var calmOut string
	calmErr := captureStdoutText(t, &calmOut, func() error { return runDoctorOpts(dir, false) })
	if want := `  ✓ Node toolchain: pass — /bin/zsh -lc: repo root: node v24.22.1 meets engines.node ">=22" in package.json`; !strings.Contains(calmOut, want) {
		t.Errorf("without an interrupt, doctor output lacks %q:\n%s", want, calmOut)
	}
	if calmErr != nil && !strings.HasSuffix(calmErr.Error(), " check(s) failed") {
		t.Errorf("without an interrupt, doctor returned %v, want nil or the count of failed checks", calmErr)
	}
	if strings.Contains(calmOut, "All checks passed.") != (calmErr == nil) || strings.Contains(calmOut, "\nInterrupted: ") {
		t.Errorf("without an interrupt, doctor returned %v after this output:\n%s", calmErr, calmOut)
	}

	cause := errors.New("test stopped")
	setDoctorNodeSignalContext(t, func(parent context.Context) (context.Context, context.CancelFunc) {
		ctx, cancel := context.WithCancelCause(parent)
		cancel(cause)
		return ctx, func() { cancel(nil) }
	})
	var out string
	err := captureStdoutText(t, &out, func() error { return runDoctorOpts(dir, false) })
	wantErr := "doctor: interrupted (test stopped) while the Node toolchain check measured the tools"
	if calmErr != nil {
		wantErr += "; " + strings.TrimPrefix(calmErr.Error(), "doctor: ")
	}
	if err == nil || err.Error() != wantErr || !errors.Is(err, cause) {
		t.Errorf("with an interrupt, doctor returned %v, want %q wrapping the cause", err, wantErr)
	}
	if strings.Contains(out, "All checks passed.") {
		t.Errorf("with an interrupt, doctor still says all checks passed:\n%s", out)
	}
	for _, want := range []string{
		"  ⚠ Node toolchain: warn — /bin/zsh -lc was interrupted (test stopped) before it finished measuring: nothing was compared\n",
		"\nInterrupted: the Node toolchain check did not finish. Run ralph doctor again.\n",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("with an interrupt, doctor output lacks %q:\n%s", want, out)
		}
	}
	if got, want := doctorLinesButNode(out), doctorLinesButNode(calmOut); len(want) == 0 || !reflect.DeepEqual(got, want) {
		t.Errorf("with an interrupt, the other checks' lines are\n%s\nwant them as without one:\n%s",
			strings.Join(got, "\n"), strings.Join(want, "\n"))
	}
}

// TestMain's pin runs no shell: in a directory with a Node marker the line
// says the probe did not run, and in one without, the check is skipped.
func TestDoctorNodeToolchain_TestMainPinRunsNoShell(t *testing.T) {
	setDoctorNodeShell(t, "/bin/zsh")
	dir := nodeNonGitDir(t)
	writeNodeFiles(t, dir, map[string]string{".nvmrc": "24"})
	var out string
	_ = captureStdoutText(t, &out, func() error { return runDoctorOpts(dir, false) })
	if want := "  ⚠ Node toolchain: warn — /bin/zsh -lc did not finish measuring: the probe runs no shell in internal/cli's tests " +
		"(TestMain); the tools it did not reach are not compared\n"; !strings.Contains(out, want) {
		t.Errorf("doctor output lacks %q:\n%s", want, out)
	}
	if n := strings.Count(out, "⚠ Node toolchain:"); n != 1 {
		t.Errorf("doctor output has %d Node toolchain warn lines, want 1:\n%s", n, out)
	}

	empty := nodeNonGitDir(t)
	out = ""
	_ = captureStdoutText(t, &out, func() error { return runDoctorOpts(empty, false) })
	if want := "  ℹ Node toolchain: info — no Node markers"; !strings.Contains(out, want) {
		t.Errorf("doctor output lacks %q:\n%s", want, out)
	}
}
