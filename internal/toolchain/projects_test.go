package toolchain

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"time"
)

// packageJSONs returns a reader of the package.json contents in files (by
// tracked path); a path not in files reads as an error. reads counts the
// reads per path.
func packageJSONs(files map[string]string, reads map[string]int) func(string) ([]byte, error) {
	return func(rel string) ([]byte, error) {
		if reads != nil {
			reads[rel]++
		}
		data, ok := files[rel]
		if !ok {
			return nil, errors.New("no such file")
		}
		return []byte(data), nil
	}
}

func TestLayoutFromFiles_FindsPlacesAndTheirPackageManager(t *testing.T) {
	js := func(dir, pm, from string) Place {
		return Place{Dir: dir, JS: true, PackageManager: pm, PackageManagerFrom: from}
	}
	tests := []struct {
		name         string
		files        []string
		packageJSONs map[string]string
		want         Layout
	}{
		{
			name:  "root only",
			files: []string{"package.json", "pnpm-lock.yaml", "src/index.ts"},
			want:  Layout{Places: []Place{js(".", "pnpm", "pnpm-lock.yaml")}, HasJS: true},
		},
		{
			name: "no root markers, apps/a on pnpm and apps/b on yarn",
			files: []string{"README.md", "apps/a/package.json", "apps/a/pnpm-lock.yaml",
				"apps/b/package.json", "apps/b/yarn.lock"},
			want: Layout{Places: []Place{
				js(".", "npm", "default"),
				js("apps/a", "pnpm", "apps/a/pnpm-lock.yaml"),
				js("apps/b", "yarn", "apps/b/yarn.lock"),
			}, HasJS: true},
		},
		{
			name:         "version files only in subdirectories",
			files:        []string{"package.json", "web/.nvmrc", "web/package.json", "tools/.tool-versions"},
			packageJSONs: map[string]string{"package.json": `{"packageManager":"pnpm@9.0.0"}`, "web/package.json": `{"name":"web"}`},
			want: Layout{Places: []Place{
				js(".", "pnpm", "package.json packageManager"),
				js("tools", "pnpm", "package.json packageManager"),
				js("web", "pnpm", "package.json packageManager"),
			}, HasJS: true},
		},
		{
			name: "pnpm workspace: the packages' package.json make no place",
			files: []string{"package.json", "pnpm-lock.yaml", "pnpm-workspace.yaml",
				"packages/a/package.json", "packages/b/package.json", "packages/b/src/x.ts"},
			want: Layout{Places: []Place{js(".", "pnpm", "pnpm-lock.yaml")}, HasJS: true},
		},
		{
			name:  "pnpm-workspace.yaml alone counts as pnpm for the places under it",
			files: []string{"package.json", "pnpm-workspace.yaml", "packages/a/.node-version"},
			want: Layout{Places: []Place{
				js(".", "pnpm", "pnpm-workspace.yaml"),
				js("packages/a", "pnpm", "pnpm-workspace.yaml"),
			}, HasJS: true},
		},
		{
			name:  "no JS at all: the root alone, not a JS place",
			files: []string{"go.mod", "main.go", "tools/.tool-versions", "tools/mise.toml"},
			want:  Layout{Places: []Place{{Dir: "."}}, HasGo: true},
		},
		{
			name:  "a go.mod in a subdirectory still measures go",
			files: []string{"tools/go.mod", "tools/main.go"},
			want:  Layout{Places: []Place{{Dir: "."}}, HasGo: true},
		},
		{
			name:         "the nearest lockfile above beats the place's own packageManager",
			files:        []string{"yarn.lock", "package.json", "apps/x/package.json", "apps/x/.nvmrc"},
			packageJSONs: map[string]string{"package.json": `{}`, "apps/x/package.json": `{"packageManager":"pnpm@10.0.0"}`},
			want: Layout{Places: []Place{
				js(".", "yarn", "yarn.lock"),
				js("apps/x", "yarn", "yarn.lock"),
			}, HasJS: true},
		},
		{
			name:  "the place's own packageManager beats the root's",
			files: []string{"package.json", "apps/x/package.json", "apps/x/.node-version", "apps/y/.mise.toml"},
			packageJSONs: map[string]string{
				"package.json":        `{"packageManager":"yarn@4.1.0"}`,
				"apps/x/package.json": `{"packageManager":"pnpm@10.0.0+sha512.abc"}`,
			},
			want: Layout{Places: []Place{
				js(".", "yarn", "package.json packageManager"),
				js("apps/x", "pnpm", "apps/x/package.json packageManager"),
				js("apps/y", "yarn", "package.json packageManager"),
			}, HasJS: true},
		},
		{
			name: "a broken or missing packageManager falls back to npm",
			files: []string{"package.json", "apps/a/package.json", "apps/a/.nvmrc", "apps/b/package.json", "apps/b/.nvmrc",
				"apps/c/package.json", "apps/c/.nvmrc", "apps/d/package.json", "apps/d/.nvmrc", "apps/e/package.json", "apps/e/.nvmrc"},
			packageJSONs: map[string]string{
				"package.json":        `{"packageManager":`,
				"apps/a/package.json": `{"packageManager":42}`,
				"apps/b/package.json": `{"packageManager":"bun@1.1.0"}`,
				// apps/c/package.json cannot be read.
				"apps/d/package.json": `{"packageManager":"pnpm"}`,
				"apps/e/package.json": `["not an object"]`,
			},
			want: Layout{Places: []Place{
				js(".", "npm", "default"),
				js("apps/a", "npm", "default"),
				js("apps/b", "npm", "default"),
				js("apps/c", "npm", "default"),
				js("apps/d", "npm", "default"),
				js("apps/e", "npm", "default"),
			}, HasJS: true},
		},
		{
			name:  "several lockfiles in one directory: pnpm, then yarn, then npm",
			files: []string{"package-lock.json", "yarn.lock", "pnpm-lock.yaml", "b/package-lock.json", "b/yarn.lock"},
			want: Layout{Places: []Place{
				js(".", "pnpm", "pnpm-lock.yaml"),
				js("b", "yarn", "b/yarn.lock"),
			}, HasJS: true},
		},
		{
			name:  "paths under node_modules are ignored",
			files: []string{"package.json", "node_modules/foo/package-lock.json", "node_modules/foo/.nvmrc", "a/node_modules/b/yarn.lock"},
			want:  Layout{Places: []Place{js(".", "npm", "default")}, HasJS: true},
		},
		{
			name: "paths with a control character or invalid UTF-8 are skipped and counted",
			files: []string{"package.json", "bad\ndir/yarn.lock", "bad\tdir/.nvmrc", "\xffdir/pnpm-lock.yaml",
				"esc\x1bdir/.node-version", "", "ok dir/.nvmrc"},
			want: Layout{Places: []Place{
				js(".", "npm", "default"),
				js("ok dir", "npm", "default"),
			}, HasJS: true, SkippedPaths: 5},
		},
		{
			name:  "no tracked files",
			files: nil,
			want:  Layout{Places: []Place{{Dir: "."}}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := LayoutFromFiles(tt.files, packageJSONs(tt.packageJSONs, nil))
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("LayoutFromFiles(%q)\n got %+v\nwant %+v", tt.files, got, tt.want)
			}
		})
	}
}

func TestLayoutFromFiles_ReadsOnlyTrackedPackageJSONsOnce(t *testing.T) {
	reads := map[string]int{}
	files := []string{"package.json", "apps/a/package.json", "apps/a/.nvmrc", "apps/a/sub/.nvmrc", "apps/b/.nvmrc"}
	contents := map[string]string{
		"package.json":        `{}`,
		"apps/a/package.json": `{}`,
		"apps/b/package.json": `{"packageManager":"pnpm@10.0.0"}`, // not tracked
	}
	l := LayoutFromFiles(files, packageJSONs(contents, reads))
	if want := map[string]int{"package.json": 1, "apps/a/package.json": 1}; !reflect.DeepEqual(reads, want) {
		t.Errorf("package.json reads = %v, want %v", reads, want)
	}
	for _, p := range l.Places {
		if p.PackageManager != ToolNpm {
			t.Errorf("place %s: package manager %q, want npm (apps/b/package.json is not tracked)", p.Dir, p.PackageManager)
		}
	}
}

func TestLayoutFromFiles_CapsPlacesAndCountsTheRest(t *testing.T) {
	t.Run("MaxPlaces lowered to 3 keeps the root and the two shallowest", func(t *testing.T) {
		old := MaxPlaces
		MaxPlaces = 3
		t.Cleanup(func() { MaxPlaces = old })
		files := []string{"package.json", "a/x/y/.nvmrc", "c/.nvmrc", "a/x/.nvmrc", "b/.nvmrc", "a/.nvmrc"}
		l := LayoutFromFiles(files, nil)
		var dirs []string
		for _, p := range l.Places {
			dirs = append(dirs, p.Dir)
		}
		if want := []string{".", "a", "b"}; !reflect.DeepEqual(dirs, want) {
			t.Errorf("places = %q, want %q", dirs, want)
		}
		if l.Truncated != 3 {
			t.Errorf("Truncated = %d, want 3 (c, a/x, a/x/y)", l.Truncated)
		}
	})
	t.Run("the default cap is 10 places, the root included", func(t *testing.T) {
		files := []string{"package.json"}
		for i := range 12 {
			files = append(files, "pkg"+strconv.Itoa(i)+"/yarn.lock")
		}
		l := LayoutFromFiles(files, nil)
		if len(l.Places) != 10 || l.Truncated != 3 {
			t.Errorf("places = %d, Truncated = %d; want 10 and 3", len(l.Places), l.Truncated)
		}
		if l.Places[0].Dir != RootDir {
			t.Errorf("first place = %q, want the root", l.Places[0].Dir)
		}
	})
}

func TestLayoutTargets_ListToolsPlaceByPlace(t *testing.T) {
	l := LayoutFromFiles([]string{"go.mod", "apps/a/pnpm-lock.yaml", "apps/b/yarn.lock"}, nil)
	wantAll := []Target{
		{".", "node"}, {".", "npm"}, {".", "python3"}, {".", "go"},
		{"apps/a", "node"}, {"apps/a", "pnpm"},
		{"apps/b", "node"}, {"apps/b", "yarn"},
	}
	if got := l.Targets(); !reflect.DeepEqual(got, wantAll) {
		t.Errorf("Targets() = %v, want %v", got, wantAll)
	}
	wantNode := []Target{{".", "node"}, {".", "npm"}, {"apps/a", "node"}, {"apps/a", "pnpm"}, {"apps/b", "node"}, {"apps/b", "yarn"}}
	if got := l.NodeTargets(); !reflect.DeepEqual(got, wantNode) {
		t.Errorf("NodeTargets() = %v, want %v", got, wantNode)
	}

	noJS := LayoutFromFiles([]string{"main.py"}, nil)
	if got, want := noJS.Targets(), []Target{{".", "python3"}}; !reflect.DeepEqual(got, want) {
		t.Errorf("no JS, no go: Targets() = %v, want %v", got, want)
	}
	if got := noJS.NodeTargets(); len(got) != 0 {
		t.Errorf("no JS: NodeTargets() = %v, want none", got)
	}
	if got, want := RootLayout().Targets(), []Target{{".", "python3"}}; !reflect.DeepEqual(got, want) {
		t.Errorf("RootLayout().Targets() = %v, want %v", got, want)
	}
}

// isolateGit keeps git in these tests off the developer's git config and a
// GIT_DIR-style variable from a git hook.
func isolateGit(t *testing.T) {
	t.Helper()
	t.Setenv("GIT_CONFIG_GLOBAL", os.DevNull)
	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
	for _, key := range []string{"GIT_DIR", "GIT_WORK_TREE", "GIT_COMMON_DIR", "GIT_INDEX_FILE"} {
		t.Setenv(key, "")
		if err := os.Unsetenv(key); err != nil {
			t.Fatalf("unsetenv %s: %v", key, err)
		}
	}
}

func writeFile(t *testing.T, p, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestFindLayout_ReadsTrackedFilesFromGit(t *testing.T) {
	isolateGit(t)
	root := t.TempDir()
	if out, err := exec.Command("git", "-C", root, "init", "--quiet").CombinedOutput(); err != nil {
		t.Fatalf("git init: %v\n%s", err, out)
	}
	writeFile(t, filepath.Join(root, "package.json"), `{"packageManager":"yarn@4.5.0"}`)
	writeFile(t, filepath.Join(root, "apps", "web app", "package.json"), `{"name":"web"}`)
	writeFile(t, filepath.Join(root, "apps", "web app", ".node-version"), "24\n")
	writeFile(t, filepath.Join(root, "go.mod"), "module example.com/x\n")
	// Not tracked: no place.
	writeFile(t, filepath.Join(root, "apps", "untracked", "pnpm-lock.yaml"), "")
	if out, err := exec.Command("git", "-C", root, "add", "-f", "package.json", "apps/web app", "go.mod").CombinedOutput(); err != nil {
		t.Fatalf("git add: %v\n%s", err, out)
	}

	l, err := FindLayout(context.Background(), root)
	if err != nil {
		t.Fatalf("FindLayout: %v", err)
	}
	want := Layout{Places: []Place{
		{Dir: ".", JS: true, PackageManager: "yarn", PackageManagerFrom: "package.json packageManager"},
		{Dir: "apps/web app", JS: true, PackageManager: "yarn", PackageManagerFrom: "package.json packageManager"},
	}, HasJS: true, HasGo: true}
	if !reflect.DeepEqual(l, want) {
		t.Errorf("FindLayout\n got %+v\nwant %+v", l, want)
	}
}

func TestFindLayout_GitFailureReturnsRootLayoutAndAnError(t *testing.T) {
	isolateGit(t)
	missing := filepath.Join(t.TempDir(), "no-such-dir")
	l, err := FindLayout(context.Background(), missing)
	if err == nil {
		t.Fatal("FindLayout on a missing directory: no error")
	}
	if !strings.Contains(err.Error(), "git ls-files") {
		t.Errorf("error %q does not name git ls-files", err)
	}
	if !reflect.DeepEqual(l, RootLayout()) {
		t.Errorf("layout = %+v, want RootLayout() (python3 at the root)", l)
	}
	if got, want := l.Targets(), []Target{{".", "python3"}}; !reflect.DeepEqual(got, want) {
		t.Errorf("Targets() = %v, want %v", got, want)
	}
}

func TestFindLayout_LsFilesIsBoundedByLsFilesTimeout(t *testing.T) {
	bin := t.TempDir()
	writeExec(t, filepath.Join(bin, "git"), "#!/bin/sh\nexec /bin/sleep 30\n")
	t.Setenv("PATH", bin)
	oldTimeout, oldGrace := LsFilesTimeout, KillGrace
	LsFilesTimeout, KillGrace = 300*time.Millisecond, 300*time.Millisecond
	t.Cleanup(func() { LsFilesTimeout, KillGrace = oldTimeout, oldGrace })

	start := time.Now()
	l, err := FindLayout(context.Background(), t.TempDir())
	elapsed := time.Since(start)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("error = %v, want one wrapping context.DeadlineExceeded", err)
	}
	if !reflect.DeepEqual(l, RootLayout()) {
		t.Errorf("layout = %+v, want RootLayout()", l)
	}
	// 30 s is the fake git's sleep; well under it means the bound worked
	// (slack for a loaded machine).
	if elapsed > 10*time.Second {
		t.Errorf("FindLayout took %s with a 300ms bound", elapsed)
	}
}
