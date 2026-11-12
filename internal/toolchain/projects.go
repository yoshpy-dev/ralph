package toolchain

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"time"
	"unicode/utf8"
)

// The places a repository's tools are measured in (plan
// 2026-11-12-gate-tool-versions, D2): the root, and every directory that
// holds a lockfile or a version file, read from the repository's tracked
// files, each JS place with the one package manager its scripts would use.

// RootDir is the Dir of the repository root's Place.
const RootDir = "."

// The tools the probe measures. A JS place's package manager is one of
// ToolPnpm, ToolYarn, and ToolNpm.
const (
	ToolNode    = "node"
	ToolPnpm    = "pnpm"
	ToolYarn    = "yarn"
	ToolNpm     = "npm"
	ToolPython3 = "python3"
	ToolGo      = "go"
)

// MaxPlaces caps the places a Layout lists, the root included; the places
// found beyond it are counted in Layout.Truncated. Tests lower it.
var MaxPlaces = 10

// LsFilesTimeout bounds the `git ls-files` FindLayout runs.
var LsFilesTimeout = 10 * time.Second

// maxPackageJSONBytes caps the package.json FindLayout reads; a bigger one
// is treated as unreadable (no packageManager).
const maxPackageJSONBytes = 1 << 20

// Place is one directory the probe measures tools in.
type Place struct {
	// Dir is the directory, slash-separated and relative to the repository
	// root; RootDir is the root.
	Dir string
	// JS is whether node and a package manager are measured here: every
	// place is a JS place when the repository has a JS marker anywhere
	// (Layout.HasJS), and the root alone, not a JS place, otherwise.
	JS bool
	// PackageManager is ToolPnpm, ToolYarn, or ToolNpm at a JS place, and ""
	// elsewhere.
	PackageManager string
	// PackageManagerFrom says what chose PackageManager: the lockfile's path
	// ("pnpm-lock.yaml", "apps/a/yarn.lock"), a package.json's field
	// ("apps/b/package.json packageManager"), or "default" for npm when
	// nothing names one. "" when JS is false.
	PackageManagerFrom string
}

// Layout is where a repository's tools are measured.
type Layout struct {
	// Places lists the root first, then the other places by depth and then
	// by path, at most MaxPlaces of them.
	Places []Place
	// Truncated is how many places were found beyond MaxPlaces and left out
	// of Places.
	Truncated int
	// HasJS is whether any tracked file is a JS marker: package.json,
	// pnpm-lock.yaml, yarn.lock, package-lock.json, .node-version, or .nvmrc.
	HasJS bool
	// HasGo is whether any tracked file is a go.mod; go is then measured at
	// the root.
	HasGo bool
	// SkippedPaths is how many tracked paths were left out because their
	// names hold a control character or are not UTF-8, so a caller can log
	// it.
	SkippedPaths int
}

// Target is one tool the probe measures in one place.
type Target struct {
	Dir  string // a Place's Dir
	Tool string // ToolNode, a package manager, ToolPython3, or ToolGo
}

// RootLayout is the layout of a repository whose tracked files could not be
// read: the root alone, not a JS place, so only python3 is measured there.
func RootLayout() Layout {
	return Layout{Places: []Place{{Dir: RootDir}}}
}

// Targets lists every tool to measure, place by place in Places' order:
// node and the package manager at each JS place, and at the root python3
// and, when HasGo, go.
func (l Layout) Targets() []Target {
	var out []Target
	for _, p := range l.Places {
		if p.JS {
			out = append(out, Target{p.Dir, ToolNode}, Target{p.Dir, p.PackageManager})
		}
		if p.Dir == RootDir {
			out = append(out, Target{p.Dir, ToolPython3})
			if l.HasGo {
				out = append(out, Target{p.Dir, ToolGo})
			}
		}
	}
	return out
}

// NodeTargets lists node and the package manager at each JS place, in
// Places' order; it is empty when the repository has no JS place.
func (l Layout) NodeTargets() []Target {
	var out []Target
	for _, p := range l.Places {
		if p.JS {
			out = append(out, Target{p.Dir, ToolNode}, Target{p.Dir, p.PackageManager})
		}
	}
	return out
}

// The tracked file names LayoutFromFiles reads.
const (
	fileGoMod           = "go.mod"
	filePackageJSON     = "package.json"
	filePnpmLock        = "pnpm-lock.yaml"
	filePnpmWorkspace   = "pnpm-workspace.yaml"
	fileYarnLock        = "yarn.lock"
	filePackageLockJSON = "package-lock.json"
	fileNodeVersion     = ".node-version"
	fileNvmrc           = ".nvmrc"
	fileToolVersions    = ".tool-versions"
	fileMiseTOML        = "mise.toml"
	fileDotMiseTOML     = ".mise.toml"
)

// lockfilePrecedence is the order in which a directory's lockfiles choose
// its package manager when it has more than one; pnpm-workspace.yaml counts
// as pnpm's.
var lockfilePrecedence = []struct{ file, manager string }{
	{filePnpmLock, ToolPnpm},
	{filePnpmWorkspace, ToolPnpm},
	{fileYarnLock, ToolYarn},
	{filePackageLockJSON, ToolNpm},
}

// dirMarkers is what LayoutFromFiles found in one directory.
type dirMarkers struct {
	files       map[string]bool // the marker file names found here
	place       bool            // a lockfile or a version file is here
	packageJSON bool
}

// LayoutFromFiles builds the layout from files, the repository's tracked
// paths (slash-separated, relative to the root, as `git ls-files` prints
// them). readPackageJSON reads a tracked package.json by that path; a read
// error, a file that is not a JSON object, and a packageManager that is not
// a string naming pnpm, yarn, or npm all count as no packageManager.
//
// The places are the root, when any JS marker is tracked, and every
// directory holding a lockfile (pnpm-lock.yaml, yarn.lock,
// package-lock.json) or a version file (.node-version, .nvmrc,
// .tool-versions, mise.toml, .mise.toml); all of them are JS places. With no
// JS marker the root alone is a place, not a JS one. A package.json makes no
// place by itself (a workspace's packages share the root's tools). A JS
// place's package manager is the nearest lockfile's at or above it, else its
// own package.json's packageManager, else the root package.json's, else npm.
// Paths under a node_modules directory are ignored, and paths holding a
// control character or invalid UTF-8 are counted in SkippedPaths.
func LayoutFromFiles(files []string, readPackageJSON func(rel string) ([]byte, error)) Layout {
	var l Layout
	dirs := map[string]*dirMarkers{}
	marker := func(dir string) *dirMarkers {
		m := dirs[dir]
		if m == nil {
			m = &dirMarkers{files: map[string]bool{}}
			dirs[dir] = m
		}
		return m
	}
	for _, f := range files {
		if !safePathName(f) {
			l.SkippedPaths++
			continue
		}
		p := path.Clean(f)
		if p == RootDir || p == ".." || strings.HasPrefix(p, "../") || path.IsAbs(p) || underNodeModules(p) {
			continue
		}
		dir, base := path.Dir(p), path.Base(p)
		switch base {
		case fileGoMod:
			l.HasGo = true
		case filePackageJSON:
			l.HasJS = true
			marker(dir).packageJSON = true
		case filePnpmLock, fileYarnLock, filePackageLockJSON, fileNodeVersion, fileNvmrc:
			l.HasJS = true
			m := marker(dir)
			m.files[base] = true
			m.place = true
		case fileToolVersions, fileMiseTOML, fileDotMiseTOML:
			m := marker(dir)
			m.files[base] = true
			m.place = true
		case filePnpmWorkspace:
			marker(dir).files[base] = true
		}
	}
	if !l.HasJS {
		l.Places = RootLayout().Places
		return l
	}

	var others []string
	for dir, m := range dirs {
		if m.place && dir != RootDir {
			others = append(others, dir)
		}
	}
	sort.Slice(others, func(i, j int) bool {
		di, dj := strings.Count(others[i], "/"), strings.Count(others[j], "/")
		if di != dj {
			return di < dj
		}
		return others[i] < others[j]
	})
	keep := max(MaxPlaces, 1) - 1
	if len(others) > keep {
		l.Truncated = len(others) - keep
		others = others[:keep]
	}

	fields := map[string]string{} // package.json path -> its packageManager's manager
	managerField := func(dir string) string {
		m := dirs[dir]
		if m == nil || !m.packageJSON {
			return ""
		}
		rel := path.Join(dir, filePackageJSON)
		if v, ok := fields[rel]; ok {
			return v
		}
		v := packageManagerOf(rel, readPackageJSON)
		fields[rel] = v
		return v
	}
	for _, dir := range append([]string{RootDir}, others...) {
		pm, from := placeManager(dir, dirs, managerField)
		l.Places = append(l.Places, Place{Dir: dir, JS: true, PackageManager: pm, PackageManagerFrom: from})
	}
	return l
}

// placeManager returns the package manager of the JS place dir and what
// chose it (see Place.PackageManagerFrom).
func placeManager(dir string, dirs map[string]*dirMarkers, managerField func(dir string) string) (manager, from string) {
	for d := dir; ; d = path.Dir(d) {
		if m := dirs[d]; m != nil {
			for _, lf := range lockfilePrecedence {
				if m.files[lf.file] {
					return lf.manager, path.Join(d, lf.file)
				}
			}
		}
		if d == RootDir {
			break
		}
	}
	for _, d := range []string{dir, RootDir} {
		if pm := managerField(d); pm != "" {
			return pm, path.Join(d, filePackageJSON) + " packageManager"
		}
		if d == RootDir {
			break
		}
	}
	return ToolNpm, "default"
}

// packageManagerOf reads the package.json at rel and returns the manager its
// packageManager field names when that is pnpm, yarn, or npm with an "@"
// and a version after it, and "" otherwise.
func packageManagerOf(rel string, readPackageJSON func(rel string) ([]byte, error)) string {
	if readPackageJSON == nil {
		return ""
	}
	data, err := readPackageJSON(rel)
	if err != nil {
		return ""
	}
	var pj struct {
		PackageManager any `json:"packageManager"`
	}
	if json.Unmarshal(data, &pj) != nil {
		return ""
	}
	s, ok := pj.PackageManager.(string)
	if !ok {
		return ""
	}
	name, version, ok := strings.Cut(strings.TrimSpace(s), "@")
	if !ok || version == "" {
		return ""
	}
	switch name {
	case ToolPnpm, ToolYarn, ToolNpm:
		return name
	}
	return ""
}

// safePathName reports whether p is valid UTF-8 with no control character,
// so it can go into a shell word, a record field, and a log line as it is.
func safePathName(p string) bool {
	if p == "" || !utf8.ValidString(p) {
		return false
	}
	for _, r := range p {
		if r < 0x20 || r == 0x7f || (r >= 0x80 && r < 0xa0) {
			return false
		}
	}
	return true
}

// underNodeModules reports whether the clean relative path p is inside a
// node_modules directory.
func underNodeModules(p string) bool {
	return slices.Contains(strings.Split(path.Dir(p), "/"), "node_modules")
}

// FindLayout reads root's tracked files with `git -C root ls-files -z`,
// bounded by LsFilesTimeout and ctx, and builds their layout
// (LayoutFromFiles), reading each tracked package.json from disk (a missing,
// non-regular, or larger than 1 MiB file counts as no packageManager). When
// git fails it returns RootLayout(), so the caller can still measure python3
// at the root, with an error the caller can log.
func FindLayout(ctx context.Context, root string) (Layout, error) {
	files, err := trackedFiles(ctx, root)
	if err != nil {
		return RootLayout(), err
	}
	return LayoutFromFiles(files, func(rel string) ([]byte, error) {
		return readSmallFile(filepath.Join(root, filepath.FromSlash(rel)), maxPackageJSONBytes)
	}), nil
}

// trackedFiles returns the paths `git -C root ls-files -z` lists.
func trackedFiles(ctx context.Context, root string) ([]string, error) {
	ctx, cancel := context.WithTimeout(ctx, LsFilesTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, "git", "-C", root, "ls-files", "-z")
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	cmd.WaitDelay = KillGrace
	out, err := cmd.Output()
	if err != nil {
		if ctx.Err() != nil {
			return nil, fmt.Errorf("git ls-files in %s: %w", root, context.Cause(ctx))
		}
		if line := firstLine(stderr.String()); line != "" {
			return nil, fmt.Errorf("git ls-files in %s: %w: %s", root, err, cleanText(line, maxReasonBytes))
		}
		return nil, fmt.Errorf("git ls-files in %s: %w", root, err)
	}
	var files []string
	for _, f := range strings.Split(string(out), "\x00") {
		if f != "" {
			files = append(files, f)
		}
	}
	return files, nil
}

// readSmallFile reads the regular file p, refusing one bigger than limit
// bytes; it never opens a FIFO or a device, which could block.
func readSmallFile(p string, limit int64) ([]byte, error) {
	fi, err := os.Stat(p)
	if err != nil {
		return nil, err
	}
	if !fi.Mode().IsRegular() {
		return nil, fmt.Errorf("%s is not a regular file", p)
	}
	f, err := os.Open(p)
	if err != nil {
		return nil, err
	}
	defer func() { _ = f.Close() }()
	data, err := io.ReadAll(io.LimitReader(f, limit+1))
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > limit {
		return nil, errors.New(p + " is bigger than the limit")
	}
	return data, nil
}
