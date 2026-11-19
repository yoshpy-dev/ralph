package cli

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"testing"

	"github.com/yoshpy-dev/ralph/internal/scaffold"
)

const (
	golangReadmeKey  = "packs/languages/golang/README.md"
	golangVerifyKey  = "packs/languages/golang/verify.sh"
	golangRuleKey    = ".claude/rules/ralph/golang.md"
	golangPackDirRel = "packs/languages/golang"
)

// golangPackFiles is every file the gen1/gen2 fixtures' golang pack writes.
var golangPackFiles = []string{golangReadmeKey, golangVerifyKey, golangRuleKey}

func runRemovePack(t *testing.T, target, lang string, dryRun bool) (string, error) {
	t.Helper()
	var out bytes.Buffer
	err := removePack(target, lang, dryRun, &out)
	return out.String(), err
}

// golangManifestKeys lists the manifest keys in the golang pack's three
// places, written out here rather than taken from the code under test.
func golangManifestKeys(m *scaffold.Manifest) []string {
	var keys []string
	for k := range m.Files {
		if strings.HasPrefix(k, "packs/languages/golang/") || k == golangRuleKey || k == ".claude/rules/golang.md" {
			keys = append(keys, k)
		}
	}
	sort.Strings(keys)
	return keys
}

func readManifestBytes(t *testing.T, target string) []byte {
	t.Helper()
	return mustReadFile(t, filepath.Join(target, ".ralph", "manifest.toml"))
}

func lstatExists(t *testing.T, p string) bool {
	t.Helper()
	_, err := os.Lstat(p)
	if err == nil {
		return true
	}
	if !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("lstat %s: %v", p, err)
	}
	return false
}

func assertGolangPackGoneFromManifest(t *testing.T, target string) {
	t.Helper()
	m := readManifestV2(t, target)
	if slices.Contains(m.Meta.Packs, "golang") {
		t.Errorf("Meta.Packs = %v, want no golang", m.Meta.Packs)
	}
	if keys := golangManifestKeys(m); len(keys) != 0 {
		t.Errorf("manifest still has golang pack entries: %v", keys)
	}
}

func upgradeToGen2(t *testing.T, target string) {
	t.Helper()
	scaffold.EmbeddedFS = gen2().build()
	Version = "1.1.0-test"
	var out, errOut bytes.Buffer
	if err := runUpgradeIOWithOptions(target, upgradeOptions{Pager: pagerNever}, strings.NewReader(""), &out, &errOut, false); err != nil {
		t.Fatalf("runUpgradeIOWithOptions: %v\nstderr:\n%s", err, errOut.String())
	}
}

// TestRemovePack_UnmodifiedPack_DeletesFilesEntriesAndPacksDir covers AC-1.
func TestRemovePack_UnmodifiedPack_DeletesFilesEntriesAndPacksDir(t *testing.T) {
	target := initV2Project(t, gen1(), "1.0.0-test")
	for _, rel := range golangPackFiles {
		if !lstatExists(t, filepath.Join(target, rel)) {
			t.Fatalf("precondition: %s must exist after init", rel)
		}
	}

	out, err := runRemovePack(t, target, "golang", false)
	if err != nil {
		t.Fatalf("removePack: %v\n%s", err, out)
	}
	if !strings.Contains(out, "✓ Pack golang removed (3 deleted, 0 kept, 0 already missing)\n") {
		t.Errorf("output must summarize 3 deleted files:\n%s", out)
	}
	if strings.Contains(out, "left in place") {
		t.Errorf("no directory is left, so the output must not say one was:\n%s", out)
	}

	assertGolangPackGoneFromManifest(t, target)
	for _, rel := range golangPackFiles {
		if lstatExists(t, filepath.Join(target, rel)) {
			t.Errorf("%s must be deleted", rel)
		}
	}
	if lstatExists(t, filepath.Join(target, "packs")) {
		t.Errorf("packs/ held only the golang pack, so it must be removed")
	}

	assertAllScaffoldChecksPass(t, target)
	for _, r := range checkInstalledPacks(target) {
		if r.Name == "Pack: golang" {
			t.Errorf("checkInstalledPacks still reports the removed pack: %+v", r)
		}
	}
}

// TestRemovePack_UpgradeAfterwardDoesNotReinstallThePack covers AC-2.
func TestRemovePack_UpgradeAfterwardDoesNotReinstallThePack(t *testing.T) {
	target := initV2Project(t, gen1(), "1.0.0-test")
	if out, err := runRemovePack(t, target, "golang", false); err != nil {
		t.Fatalf("removePack: %v\n%s", err, out)
	}

	upgradeToGen2(t, target)

	for _, rel := range golangPackFiles {
		if lstatExists(t, filepath.Join(target, rel)) {
			t.Errorf("upgrade must not recreate %s after the pack was removed", rel)
		}
	}
	assertGolangPackGoneFromManifest(t, target)
	assertAllScaffoldChecksPass(t, target)
}

// TestRemovePack_KeepsModifiedAndEjectedFiles covers AC-3: a changed pack
// file and an ejected rule stay byte for byte, are named with their reason,
// and are no longer tracked, so doctor --strict passes and a later upgrade
// leaves them alone.
func TestRemovePack_KeepsModifiedAndEjectedFiles(t *testing.T) {
	target := initV2Project(t, gen1(), "1.0.0-test")
	const myVerify = "#!/bin/sh\necho my own checks\n"
	const myRule = "# My Go rules\n"
	writeMigrationDiskFile(t, target, golangVerifyKey, myVerify)
	writeMigrationDiskFile(t, target, golangRuleKey, myRule)
	var ejectOut bytes.Buffer
	if err := runEjectIO(target, golangRuleKey, &ejectOut); err != nil {
		t.Fatalf("runEjectIO: %v", err)
	}

	out, err := runRemovePack(t, target, "golang", false)
	if err != nil {
		t.Fatalf("removePack: %v\n%s", err, out)
	}
	for _, want := range []string{
		"✓ Pack golang removed (1 deleted, 2 kept, 0 already missing)\n",
		"  " + golangVerifyKey + ": modified since ralph wrote it\n",
		"  " + golangRuleKey + ": ejected (owner=fork)\n",
		"ralph no longer tracks the kept files",
		"Before you run ralph pack add golang again, delete or move them: it overwrites a kept file, writes through a kept symlink to its target, and fails on a kept directory.\n",
		"packs/languages/golang/ was left in place because it still holds files.\n",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("output must contain %q:\n%s", want, out)
		}
	}

	if lstatExists(t, filepath.Join(target, golangReadmeKey)) {
		t.Errorf("the untouched %s must be deleted", golangReadmeKey)
	}
	if got := string(mustReadFile(t, filepath.Join(target, golangVerifyKey))); got != myVerify {
		t.Errorf("%s changed: %q, want %q", golangVerifyKey, got, myVerify)
	}
	if got := string(mustReadFile(t, filepath.Join(target, golangRuleKey))); got != myRule {
		t.Errorf("%s changed: %q, want %q", golangRuleKey, got, myRule)
	}
	assertGolangPackGoneFromManifest(t, target)
	assertAllScaffoldChecksPass(t, target)

	upgradeToGen2(t, target)
	if got := string(mustReadFile(t, filepath.Join(target, golangVerifyKey))); got != myVerify {
		t.Errorf("upgrade changed the kept %s: %q", golangVerifyKey, got)
	}
	if got := string(mustReadFile(t, filepath.Join(target, golangRuleKey))); got != myRule {
		t.Errorf("upgrade changed the kept %s: %q", golangRuleKey, got)
	}
	assertGolangPackGoneFromManifest(t, target)
	assertAllScaffoldChecksPass(t, target)
}

// TestRemovePack_AfterUpgrade_ComparesWithTheRecordedDiskHash: an upgrade
// records DiskHash on the pack's entries; removePack reads it before Hash,
// as ralph upgrade does, so a stale Hash does not keep an unchanged file.
func TestRemovePack_AfterUpgrade_ComparesWithTheRecordedDiskHash(t *testing.T) {
	target := initV2Project(t, gen1(), "1.0.0-test")
	upgradeToGen2(t, target)

	m := readManifestV2(t, target)
	for _, key := range golangPackFiles {
		entry, ok := m.Files[key]
		if !ok || entry.DiskHash == "" || entry.Owner != scaffold.OwnerCore {
			t.Fatalf("precondition: %s after upgrade = %+v (present %v), want an owner=core entry with DiskHash", key, entry, ok)
		}
		entry.Hash = "sha256:stale"
		m.Files[key] = entry
	}
	if err := m.Write(filepath.Join(target, ".ralph", "manifest.toml")); err != nil {
		t.Fatalf("writing manifest: %v", err)
	}

	out, err := runRemovePack(t, target, "golang", false)
	if err != nil {
		t.Fatalf("removePack: %v\n%s", err, out)
	}
	if !strings.Contains(out, "(3 deleted, 0 kept, 0 already missing)") {
		t.Errorf("every upgraded pack file is unchanged, so all 3 must be deleted:\n%s", out)
	}
	for _, rel := range golangPackFiles {
		if lstatExists(t, filepath.Join(target, rel)) {
			t.Errorf("%s must be deleted", rel)
		}
	}
	assertGolangPackGoneFromManifest(t, target)
	assertAllScaffoldChecksPass(t, target)
}

// TestRemovePack_NonCoreOwnersAndStaleDiskHash_AreKept: only owner=core
// files whose content matches the recorded hash are deleted. An entry with
// owner=seed or with no owner is kept, and so is a file whose DiskHash
// differs even when its Hash matches.
func TestRemovePack_NonCoreOwnersAndStaleDiskHash_AreKept(t *testing.T) {
	target := initV2Project(t, gen1(), "1.0.0-test")
	m := readManifestV2(t, target)
	readme := m.Files[golangReadmeKey]
	readme.Owner = scaffold.OwnerSeed
	m.Files[golangReadmeKey] = readme
	verify := m.Files[golangVerifyKey]
	verify.Owner = ""
	m.Files[golangVerifyKey] = verify
	rule := m.Files[golangRuleKey]
	rule.Hash = scaffold.HashBytes(mustReadFile(t, filepath.Join(target, golangRuleKey)))
	rule.DiskHash = "sha256:other"
	m.Files[golangRuleKey] = rule
	if err := m.Write(filepath.Join(target, ".ralph", "manifest.toml")); err != nil {
		t.Fatalf("writing manifest: %v", err)
	}

	out, err := runRemovePack(t, target, "golang", false)
	if err != nil {
		t.Fatalf("removePack: %v\n%s", err, out)
	}
	for _, want := range []string{
		"(0 deleted, 3 kept, 0 already missing)",
		"  " + golangReadmeKey + ": owner=seed (only owner=core files are deleted)\n",
		"  " + golangVerifyKey + ": no recorded owner (only owner=core files are deleted)\n",
		"  " + golangRuleKey + ": modified since ralph wrote it\n",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("output must contain %q:\n%s", want, out)
		}
	}
	for _, rel := range golangPackFiles {
		if !lstatExists(t, filepath.Join(target, rel)) {
			t.Errorf("%s must be kept", rel)
		}
	}
	assertGolangPackGoneFromManifest(t, target)
}

// TestRemovePack_NotInstalled_RefusesWithoutWriting covers AC-4.
func TestRemovePack_NotInstalled_RefusesWithoutWriting(t *testing.T) {
	target := initV2Project(t, gen1(), "1.0.0-test")
	before := snapshotTreeHashesExcluding(t, target)
	manifestBefore := readManifestBytes(t, target)

	out, err := runRemovePack(t, target, "typescript", false)
	if err == nil {
		t.Fatalf("removePack of a pack that is not installed: want an error, got nil\n%s", out)
	}
	if !strings.Contains(err.Error(), "not installed") || !strings.Contains(err.Error(), "golang") {
		t.Errorf("error must say typescript is not installed and name the installed golang: %v", err)
	}
	if after := snapshotTreeHashesExcluding(t, target); !slices.Equal(before, after) {
		t.Errorf("the tree changed:\nbefore=%v\nafter=%v", before, after)
	}
	if got := readManifestBytes(t, target); !bytes.Equal(got, manifestBefore) {
		t.Errorf("the manifest changed:\nbefore:\n%s\nafter:\n%s", manifestBefore, got)
	}
}

// TestRemovePack_NotInstalled_NoPacksSaysNone: with no pack installed the
// error says so instead of listing an empty set.
func TestRemovePack_NotInstalled_NoPacksSaysNone(t *testing.T) {
	target := initV2Project(t, gen1(), "1.0.0-test")
	if out, err := runRemovePack(t, target, "golang", false); err != nil {
		t.Fatalf("removePack: %v\n%s", err, out)
	}
	_, err := runRemovePack(t, target, "golang", false)
	if err == nil || !strings.Contains(err.Error(), "installed packs: none") {
		t.Errorf("a second removal must fail with \"installed packs: none\", got %v", err)
	}
}

// TestRemovePack_MalformedName_RefusesBeforeReadingAnything: a name that
// is not one path segment is refused before the manifest is read, so even a
// directory with no manifest reports the name, and nothing is written.
func TestRemovePack_MalformedName_RefusesBeforeReadingAnything(t *testing.T) {
	target := initV2Project(t, gen1(), "1.0.0-test")
	before := snapshotTreeHashesExcluding(t, target)
	empty := t.TempDir()

	for _, name := range []string{"", ".", "..", "a/b", `a\b`, "../golang"} {
		for _, dir := range []string{target, empty} {
			_, err := runRemovePack(t, dir, name, false)
			if err == nil {
				t.Errorf("removePack(%q) in %s: want an error, got nil", name, dir)
				continue
			}
			if !strings.Contains(err.Error(), "invalid language pack name "+fmt.Sprintf("%q", name)) {
				t.Errorf("removePack(%q) in %s: error must name the invalid value, got %v", name, dir, err)
			}
		}
	}
	if after := snapshotTreeHashesExcluding(t, target); !slices.Equal(before, after) {
		t.Errorf("the tree changed:\nbefore=%v\nafter=%v", before, after)
	}
}

// TestRemovePack_LegacyManifest_FailsClosed covers AC-5 for a legacy
// manifest: the sentinel is returned as is and nothing is written.
func TestRemovePack_LegacyManifest_FailsClosed(t *testing.T) {
	setupTestEmbedFS(t)
	Version = "0.1.0-test"

	dir := t.TempDir()
	writeMigrationDiskFile(t, dir, golangReadmeKey, "# Go\n")
	legacy := scaffold.NewManifest(Version)
	legacy.SetFile(golangReadmeKey, scaffold.HashBytes([]byte("# Go\n")))
	legacy.Meta.Packs = []string{"golang"}
	if err := os.MkdirAll(filepath.Join(dir, ".ralph"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := legacy.Write(filepath.Join(dir, ".ralph", "manifest.toml")); err != nil {
		t.Fatalf("writing legacy manifest: %v", err)
	}
	manifestBefore := readManifestBytes(t, dir)

	_, err := runRemovePack(t, dir, "golang", false)
	if !errors.Is(err, errLegacyLayoutFailClosed) || err.Error() != legacyLayoutFailClosedMsg {
		t.Errorf("err = %v, want errLegacyLayoutFailClosed returned unwrapped", err)
	}
	if got := readManifestBytes(t, dir); !bytes.Equal(got, manifestBefore) {
		t.Errorf("the legacy manifest changed:\nbefore:\n%s\nafter:\n%s", manifestBefore, got)
	}
	if !lstatExists(t, filepath.Join(dir, golangReadmeKey)) {
		t.Errorf("%s must not be deleted on a legacy refusal", golangReadmeKey)
	}
}

// TestRemovePack_NoManifest_Errors covers AC-5 for a directory with no
// .ralph/manifest.toml.
func TestRemovePack_NoManifest_Errors(t *testing.T) {
	_, err := runRemovePack(t, t.TempDir(), "golang", false)
	if err == nil {
		t.Fatal("removePack without a manifest: want an error, got nil")
	}
	if !strings.Contains(err.Error(), "reading manifest") || !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("err = %v, want a \"reading manifest\" error wrapping fs.ErrNotExist", err)
	}
}

// TestRemovePack_DryRun_ChangesNothing covers AC-6: the dry run names what
// it would delete, keep, and untrack, and the tree, manifest included,
// stays the same.
func TestRemovePack_DryRun_ChangesNothing(t *testing.T) {
	target := initV2Project(t, gen1(), "1.0.0-test")
	writeMigrationDiskFile(t, target, golangVerifyKey, "#!/bin/sh\necho mine\n")
	if err := os.Remove(filepath.Join(target, golangRuleKey)); err != nil {
		t.Fatal(err)
	}
	before := snapshotTreeHashesExcluding(t, target)
	manifestBefore := readManifestBytes(t, target)

	out, err := runRemovePack(t, target, "golang", true)
	if err != nil {
		t.Fatalf("removePack --dry-run: %v\n%s", err, out)
	}
	for _, want := range []string{
		"Dry run for pack golang (1 to delete, 1 to keep, 1 already missing):\n",
		"  delete  " + golangReadmeKey + "\n",
		"  keep    " + golangVerifyKey + ": modified since ralph wrote it\n",
		"  untrack " + golangRuleKey + ": already missing\n",
		"It would remove the directories under packs/languages/golang/ that are empty afterwards, and packs/languages/ and packs/ when nothing else is in them.\n",
		"remove golang and its 3 entries from .ralph/manifest.toml, so ralph no longer tracks the kept files.\n",
		"Nothing was changed.\n",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("dry-run output must contain %q:\n%s", want, out)
		}
	}
	if strings.Contains(out, "✓ Pack") {
		t.Errorf("a dry run must not print the removal summary:\n%s", out)
	}
	if after := snapshotTreeHashesExcluding(t, target); !slices.Equal(before, after) {
		t.Errorf("the dry run changed the tree:\nbefore=%v\nafter=%v", before, after)
	}
	if got := readManifestBytes(t, target); !bytes.Equal(got, manifestBefore) {
		t.Errorf("the dry run changed the manifest:\nbefore:\n%s\nafter:\n%s", manifestBefore, got)
	}
}

// TestRemovePack_LeavesTheOtherPackUntouched covers AC-7.
func TestRemovePack_LeavesTheOtherPackUntouched(t *testing.T) {
	setupTestEmbedFS(t)
	Version = "0.1.0-test"
	dir := t.TempDir()
	if err := executeInit(dir, initConfig{ProjectName: "test", Packs: []string{"golang", "typescript"}}, false); err != nil {
		t.Fatalf("executeInit: %v", err)
	}
	isTypescript := func(k string) bool {
		return strings.HasPrefix(k, "packs/languages/typescript/") || k == ".claude/rules/ralph/typescript.md"
	}
	tsEntries := func(m *scaffold.Manifest) map[string]scaffold.ManifestFile {
		out := make(map[string]scaffold.ManifestFile)
		for k, v := range m.Files {
			if isTypescript(k) {
				out[k] = v
			}
		}
		return out
	}
	tsFiles := func() []string {
		var out []string
		for _, entry := range snapshotTreeHashesExcluding(t, dir) {
			if isTypescript(filepath.ToSlash(strings.SplitN(entry, ":", 2)[0])) {
				out = append(out, entry)
			}
		}
		return out
	}
	entriesBefore := tsEntries(readManifestV2(t, dir))
	filesBefore := tsFiles()
	if len(entriesBefore) == 0 || len(filesBefore) == 0 {
		t.Fatalf("precondition: typescript entries %v and files %v must exist", entriesBefore, filesBefore)
	}

	if out, err := runRemovePack(t, dir, "golang", false); err != nil {
		t.Fatalf("removePack: %v\n%s", err, out)
	}

	m := readManifestV2(t, dir)
	if !slices.Equal(m.Meta.Packs, []string{"typescript"}) {
		t.Errorf("Meta.Packs = %v, want [typescript]", m.Meta.Packs)
	}
	if keys := golangManifestKeys(m); len(keys) != 0 {
		t.Errorf("manifest still has golang pack entries: %v", keys)
	}
	entriesAfter := tsEntries(m)
	if len(entriesAfter) != len(entriesBefore) {
		t.Errorf("typescript entries: %d after, %d before", len(entriesAfter), len(entriesBefore))
	}
	for k, v := range entriesBefore {
		if entriesAfter[k] != v {
			t.Errorf("typescript entry %s changed: before=%+v after=%+v", k, v, entriesAfter[k])
		}
	}
	if filesAfter := tsFiles(); !slices.Equal(filesBefore, filesAfter) {
		t.Errorf("typescript files changed:\nbefore=%v\nafter=%v", filesBefore, filesAfter)
	}
	if lstatExists(t, filepath.Join(dir, golangPackDirRel)) {
		t.Errorf("%s must be removed", golangPackDirRel)
	}
	if !lstatExists(t, filepath.Join(dir, "packs", "languages", "typescript")) {
		t.Errorf("the typescript pack directory must stay")
	}
}

// TestRemovePack_FilesDeletedByHand_DropsTheEntries covers AC-8.
func TestRemovePack_FilesDeletedByHand_DropsTheEntries(t *testing.T) {
	target := initV2Project(t, gen1(), "1.0.0-test")
	if err := os.RemoveAll(filepath.Join(target, golangPackDirRel)); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(filepath.Join(target, golangRuleKey)); err != nil {
		t.Fatal(err)
	}

	out, err := runRemovePack(t, target, "golang", false)
	if err != nil {
		t.Fatalf("removePack: %v\n%s", err, out)
	}
	if !strings.Contains(out, "(0 deleted, 0 kept, 3 already missing)") {
		t.Errorf("output must count the 3 missing files:\n%s", out)
	}
	assertGolangPackGoneFromManifest(t, target)
	if lstatExists(t, filepath.Join(target, "packs")) {
		t.Errorf("the empty packs/ must be removed")
	}
	assertAllScaffoldChecksPass(t, target)
}

// TestRemovePack_ManifestWriteFails_RerunFinishes covers AC-11: the files
// are deleted, the manifest write fails, the old manifest is still there,
// and running the command again finishes the removal.
func TestRemovePack_ManifestWriteFails_RerunFinishes(t *testing.T) {
	target := initV2Project(t, gen1(), "1.0.0-test")
	manifestBefore := readManifestBytes(t, target)

	saved := writePackRemoveManifest
	t.Cleanup(func() { writePackRemoveManifest = saved })
	writePackRemoveManifest = func(*scaffold.Manifest, string) error {
		return errors.New("injected write failure")
	}

	out, err := runRemovePack(t, target, "golang", false)
	if err == nil {
		t.Fatalf("removePack with a failing manifest write: want an error, got nil\n%s", out)
	}
	for _, want := range []string{
		"pack golang: deleted 3 unchanged files, but writing .ralph/manifest.toml failed, so it was not updated: ",
		"injected write failure",
		"run ralph pack remove golang again",
	} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error must contain %q: %v", want, err)
		}
	}
	if got := readManifestBytes(t, target); !bytes.Equal(got, manifestBefore) {
		t.Errorf("the manifest must stay as it was:\nbefore:\n%s\nafter:\n%s", manifestBefore, got)
	}
	m := readManifestV2(t, target)
	if !slices.Contains(m.Meta.Packs, "golang") || len(golangManifestKeys(m)) != len(golangPackFiles) {
		t.Errorf("the old manifest must still list golang and its entries: packs=%v keys=%v", m.Meta.Packs, golangManifestKeys(m))
	}

	writePackRemoveManifest = saved
	out, err = runRemovePack(t, target, "golang", false)
	if err != nil {
		t.Fatalf("re-running removePack: %v\n%s", err, out)
	}
	if !strings.Contains(out, "(0 deleted, 0 kept, 3 already missing)") {
		t.Errorf("the re-run must find the files already deleted:\n%s", out)
	}
	assertGolangPackGoneFromManifest(t, target)
	assertAllScaffoldChecksPass(t, target)
}

// TestRemovePack_RetiredPack_DeletesByRecordedHash: a pack the embedded
// templates no longer ship is removed by the hashes the manifest recorded.
func TestRemovePack_RetiredPack_DeletesByRecordedHash(t *testing.T) {
	target := initV2Project(t, gen1(), "1.0.0-test")
	retired := gen2()
	retired.includeGolang = false
	scaffold.EmbeddedFS = retired.build()
	Version = "1.1.0-test"
	if _, err := scaffold.PackFS("golang"); err == nil {
		t.Fatal("precondition: the golang pack must be missing from the embedded templates")
	}

	out, err := runRemovePack(t, target, "golang", false)
	if err != nil {
		t.Fatalf("removePack: %v\n%s", err, out)
	}
	if !strings.Contains(out, "(3 deleted, 0 kept, 0 already missing)") {
		t.Errorf("output must count 3 deleted files:\n%s", out)
	}
	for _, rel := range golangPackFiles {
		if lstatExists(t, filepath.Join(target, rel)) {
			t.Errorf("%s must be deleted", rel)
		}
	}
	assertGolangPackGoneFromManifest(t, target)
}

// TestNewPackCmd_RemoveTakesOneLanguageAndDryRun checks the wiring: ralph
// pack has a remove subcommand that takes exactly one argument and a
// --dry-run flag that reaches removePack.
func TestNewPackCmd_RemoveTakesOneLanguageAndDryRun(t *testing.T) {
	sub, rest, err := newPackCmd().Find([]string{"remove"})
	if err != nil || len(rest) != 0 || sub.Name() != "remove" {
		t.Fatalf("find remove = %v, %v, %v; want the remove subcommand", sub, rest, err)
	}
	flag := sub.Flags().Lookup("dry-run")
	if flag == nil || flag.DefValue != "false" {
		t.Fatalf("remove --dry-run = %+v, want a bool flag that defaults to false", flag)
	}
	for _, args := range [][]string{nil, {"golang", "typescript"}} {
		if err := sub.Args(sub, args); err == nil {
			t.Errorf("remove with args %v: want an error", args)
		}
	}
	if err := sub.Args(sub, []string{"golang"}); err != nil {
		t.Errorf("remove golang: %v", err)
	}

	target := initV2Project(t, gen1(), "1.0.0-test")
	t.Chdir(target)
	root := NewRootCmd()
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetArgs([]string{"pack", "remove", "golang", "--dry-run"})
	if err := root.Execute(); err != nil {
		t.Fatalf("ralph pack remove golang --dry-run: %v\n%s", err, out.String())
	}
	if !strings.Contains(out.String(), "Nothing was changed.") {
		t.Errorf("--dry-run output must say nothing was changed:\n%s", out.String())
	}
	if strings.Contains(out.String(), "kept files") {
		t.Errorf("nothing is kept, so --dry-run must not mention kept files:\n%s", out.String())
	}
	for _, rel := range golangPackFiles {
		if !lstatExists(t, filepath.Join(target, rel)) {
			t.Errorf("--dry-run must not delete %s", rel)
		}
	}
}
