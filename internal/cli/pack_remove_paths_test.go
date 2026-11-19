package cli

// This file holds the tests of pack remove's path and symlink handling: a
// symlink or a directory where a pack file was, a user file in a pack
// directory, a symlinked pack directory or ancestor, and a manifest key that
// escapes the pack. The shared helpers are in pack_remove_test.go.

import (
	"bytes"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/yoshpy-dev/ralph/internal/scaffold"
)

// TestRemovePack_SymlinkAtAPackPath_KeptAndTargetUntouched: a symlink where
// a pack file was is kept as not a regular file, its target is not touched,
// and its entry is dropped.
func TestRemovePack_SymlinkAtAPackPath_KeptAndTargetUntouched(t *testing.T) {
	target := initV2Project(t, gen1(), "1.0.0-test")
	outside := filepath.Join(t.TempDir(), "README.md")
	if err := os.WriteFile(outside, []byte(v1GolangReadme), 0o644); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(target, golangReadmeKey)
	if err := os.Remove(link); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, link); err != nil {
		t.Fatal(err)
	}

	out, err := runRemovePack(t, target, "golang", false)
	if err != nil {
		t.Fatalf("removePack: %v\n%s", err, out)
	}
	for _, want := range []string{
		"(2 deleted, 1 kept, 0 already missing)",
		"  " + golangReadmeKey + ": not a regular file (symlink)\n",
		"packs/languages/golang/ was left in place because it still holds files.\n",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("output must contain %q:\n%s", want, out)
		}
	}
	fi, err := os.Lstat(link)
	if err != nil || fi.Mode()&fs.ModeSymlink == 0 {
		t.Errorf("the symlink must stay: lstat = %v, %v", fi, err)
	}
	if got := string(mustReadFile(t, outside)); got != v1GolangReadme {
		t.Errorf("the symlink's target changed: %q", got)
	}
	assertGolangPackGoneFromManifest(t, target)
	assertAllScaffoldChecksPass(t, target)
}

// TestRemovePack_DirectoryAtAPackPath_KeptEvenWhenEmpty: an empty
// directory where a pack file was is kept as not a regular file; the
// cleanup of empty directories does not remove it.
func TestRemovePack_DirectoryAtAPackPath_KeptEvenWhenEmpty(t *testing.T) {
	target := initV2Project(t, gen1(), "1.0.0-test")
	readme := filepath.Join(target, golangReadmeKey)
	if err := os.Remove(readme); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(readme, 0o755); err != nil {
		t.Fatal(err)
	}

	out, err := runRemovePack(t, target, "golang", false)
	if err != nil {
		t.Fatalf("removePack: %v\n%s", err, out)
	}
	if !strings.Contains(out, "  "+golangReadmeKey+": not a regular file (directory)\n") {
		t.Errorf("output must name the directory as kept:\n%s", out)
	}
	if fi, err := os.Lstat(readme); err != nil || !fi.IsDir() {
		t.Errorf("the kept directory must stay: lstat = %v, %v", fi, err)
	}
	assertGolangPackGoneFromManifest(t, target)
}

// TestRemovePack_UserFileInPackDir_KeepsTheDirectory: a file the user added
// to the pack's directory stays, so the directory stays too, while an empty
// subdirectory is removed.
func TestRemovePack_UserFileInPackDir_KeepsTheDirectory(t *testing.T) {
	target := initV2Project(t, gen1(), "1.0.0-test")
	const notes = "my notes\n"
	writeMigrationDiskFile(t, target, golangPackDirRel+"/notes.md", notes)
	if err := os.MkdirAll(filepath.Join(target, golangPackDirRel, "sub", "empty"), 0o755); err != nil {
		t.Fatal(err)
	}

	out, err := runRemovePack(t, target, "golang", false)
	if err != nil {
		t.Fatalf("removePack: %v\n%s", err, out)
	}
	if !strings.Contains(out, "(3 deleted, 0 kept, 0 already missing)") {
		t.Errorf("output must count 3 deleted files:\n%s", out)
	}
	if !strings.Contains(out, "packs/languages/golang/ was left in place because it still holds files.\n") {
		t.Errorf("output must say the directory was left in place:\n%s", out)
	}
	if got := string(mustReadFile(t, filepath.Join(target, golangPackDirRel, "notes.md"))); got != notes {
		t.Errorf("the user's notes.md changed: %q", got)
	}
	if lstatExists(t, filepath.Join(target, golangPackDirRel, "sub")) {
		t.Errorf("the empty subdirectories must be removed")
	}
	for _, rel := range golangPackFiles {
		if lstatExists(t, filepath.Join(target, rel)) {
			t.Errorf("%s must be deleted", rel)
		}
	}
	assertGolangPackGoneFromManifest(t, target)
}

// TestRemovePack_SymlinkedPackDir_RefusesAndDeletesNothingOutside: when
// packs/languages/golang is a symlink to a directory outside the project,
// both the dry run and the real run refuse, nothing outside is deleted, and
// the manifest is not changed.
func TestRemovePack_SymlinkedPackDir_RefusesAndDeletesNothingOutside(t *testing.T) {
	target := initV2Project(t, gen1(), "1.0.0-test")
	outside := t.TempDir()
	copies := map[string]string{"README.md": v1GolangReadme, "verify.sh": v1GolangVerify}
	for name, content := range copies {
		if err := os.WriteFile(filepath.Join(outside, name), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	packDir := filepath.Join(target, golangPackDirRel)
	if err := os.RemoveAll(packDir); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, packDir); err != nil {
		t.Fatal(err)
	}
	manifestBefore := readManifestBytes(t, target)

	for _, dryRun := range []bool{true, false} {
		out, err := runRemovePack(t, target, "golang", dryRun)
		if err == nil {
			t.Errorf("dryRun=%v: want an error for the symlinked pack directory, got nil\n%s", dryRun, out)
			continue
		}
		if !strings.Contains(err.Error(), golangPackDirRel) {
			t.Errorf("dryRun=%v: error must name the symlinked directory: %v", dryRun, err)
		}
		if !strings.Contains(err.Error(), "pack golang: nothing was changed: ") {
			t.Errorf("dryRun=%v: error must say nothing was changed: %v", dryRun, err)
		}
	}
	for name, content := range copies {
		if got := string(mustReadFile(t, filepath.Join(outside, name))); got != content {
			t.Errorf("outside %s changed: %q", name, got)
		}
	}
	if !lstatExists(t, filepath.Join(target, golangRuleKey)) {
		t.Errorf("%s must not be deleted when the command refuses", golangRuleKey)
	}
	if got := readManifestBytes(t, target); !bytes.Equal(got, manifestBefore) {
		t.Errorf("the manifest changed:\nbefore:\n%s\nafter:\n%s", manifestBefore, got)
	}
}

// TestRemovePack_SymlinkedPacksAncestor_RemovesNoDirectoryOutside: when
// packs or packs/languages is a symlink to a directory outside the project
// and the manifest records no file under packs/languages/golang/ (only the
// rule), no key's parent check sees the symlink, so the cleanup of empty
// directories must check the ancestors itself: it removes nothing through
// the symlink, and both the dry run and the real run say why.
func TestRemovePack_SymlinkedPacksAncestor_RemovesNoDirectoryOutside(t *testing.T) {
	for _, linked := range []string{"packs", "packs/languages"} {
		t.Run(linked, func(t *testing.T) {
			target := initV2Project(t, gen1(), "1.0.0-test")
			m := readManifestV2(t, target)
			delete(m.Files, golangReadmeKey)
			delete(m.Files, golangVerifyKey)
			if err := m.Write(filepath.Join(target, ".ralph", "manifest.toml")); err != nil {
				t.Fatal(err)
			}
			if err := os.RemoveAll(filepath.Join(target, "packs")); err != nil {
				t.Fatal(err)
			}
			// outside is what linked points at, so packs/languages/golang/empty
			// resolves to emptyOutside.
			outside := t.TempDir()
			emptyOutside := filepath.Join(outside, filepath.FromSlash(strings.TrimPrefix(golangPackDirRel+"/empty", linked+"/")))
			link := filepath.Join(target, filepath.FromSlash(linked))
			for _, dir := range []string{emptyOutside, filepath.Dir(link)} {
				if err := os.MkdirAll(dir, 0o755); err != nil {
					t.Fatal(err)
				}
			}
			if err := os.Symlink(outside, link); err != nil {
				t.Fatal(err)
			}

			want := linked + " is not a real directory"
			for _, dryRun := range []bool{true, false} {
				out, err := runRemovePack(t, target, "golang", dryRun)
				if err != nil {
					t.Fatalf("dryRun=%v: removePack: %v\n%s", dryRun, err, out)
				}
				if !strings.Contains(out, want) {
					t.Errorf("dryRun=%v: output must say %q:\n%s", dryRun, want, out)
				}
			}
			if !lstatExists(t, emptyOutside) {
				t.Errorf("the empty directory outside the project, %s, must stay", emptyOutside)
			}
			if lstatExists(t, filepath.Join(target, golangRuleKey)) {
				t.Errorf("%s must still be deleted", golangRuleKey)
			}
			assertGolangPackGoneFromManifest(t, target)
		})
	}
}

// TestRemovePack_ManifestKeyEscapingThePack_Refuses: a tampered key under
// the pack's prefix whose cleaned form names another file is refused before
// anything is deleted.
func TestRemovePack_ManifestKeyEscapingThePack_Refuses(t *testing.T) {
	target := initV2Project(t, gen1(), "1.0.0-test")
	m := readManifestV2(t, target)
	agents := mustReadFile(t, filepath.Join(target, "AGENTS.md"))
	escaping := "packs/languages/golang/../../../AGENTS.md"
	if err := m.SetFileOwned(escaping, scaffold.OwnerCore, scaffold.HashBytes(agents), ""); err != nil {
		t.Fatal(err)
	}
	if err := m.Write(filepath.Join(target, ".ralph", "manifest.toml")); err != nil {
		t.Fatal(err)
	}
	before := snapshotTreeHashesExcluding(t, target)

	_, err := runRemovePack(t, target, "golang", false)
	if err == nil || !strings.Contains(err.Error(), escaping) {
		t.Errorf("err = %v, want an error naming %q", err, escaping)
	}
	if after := snapshotTreeHashesExcluding(t, target); !slices.Equal(before, after) {
		t.Errorf("the tree changed:\nbefore=%v\nafter=%v", before, after)
	}
}
