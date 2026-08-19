package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/yoshpy-dev/ralph/internal/scaffold"
	"github.com/yoshpy-dev/ralph/internal/upgrade"
)

// commitAllForAdoptTest stages and commits every file under target so
// adopt's git-clean precondition (checkGitCleanForDestructiveOp) passes.
// executeInit (init.go) runs `git init` but never commits, so every
// adopt-path test that needs a clean tree calls this first.
func commitAllForAdoptTest(t *testing.T, target string) {
	t.Helper()
	runMigrationTestGit(t, target, "add", "-A")
	runMigrationTestGit(t, target, "-c", "user.email=test@example.com", "-c", "user.name=Test", "commit", "-m", "v2 fixture snapshot")
}

// --- eject ---

// TestRunEjectIO_CoreClean_ForksRecorded is handoff test 1: ejecting an
// unmodified owner=core path records owner=fork with forked_from_version
// set to the manifest's recorded version and a disk-content hash equal to
// the (unmodified) template hash, with zero writes to any other file.
func TestRunEjectIO_CoreClean_ForksRecorded(t *testing.T) {
	target := initV2Project(t, gen1(), "1.0.0-test")

	before := snapshotDirHashes(t, target)

	var out bytes.Buffer
	if err := runEjectIO(target, "scripts/run-verify.sh", &out); err != nil {
		t.Fatalf("runEjectIO: %v", err)
	}
	if !strings.Contains(out.String(), "scripts/run-verify.sh") {
		t.Errorf("eject summary must name the path:\n%s", out.String())
	}

	m := readManifestV2(t, target)
	entry, ok := m.Files["scripts/run-verify.sh"]
	if !ok {
		t.Fatal("manifest entry for scripts/run-verify.sh must still exist after eject")
	}
	if entry.Owner != scaffold.OwnerFork {
		t.Errorf("Owner = %q, want %q", entry.Owner, scaffold.OwnerFork)
	}
	if entry.ForkedFromVersion != "1.0.0-test" {
		t.Errorf("ForkedFromVersion = %q, want %q", entry.ForkedFromVersion, "1.0.0-test")
	}
	wantHash := scaffold.HashBytes([]byte(v1RunVerify))
	if entry.DiskHash != wantHash {
		t.Errorf("DiskHash = %q, want %q (unmodified template hash)", entry.DiskHash, wantHash)
	}

	// Zero writes to any other file. manifest.toml itself is excluded
	// (eject's only permitted write) by comparing the rest of the tree.
	after := snapshotDirHashes(t, target)
	before = filterOutPath(before, ".ralph/manifest.toml")
	after = filterOutPath(after, ".ralph/manifest.toml")
	if !slicesEqualStrings(before, after) {
		t.Errorf("eject must write zero files other than the manifest; before=%v after=%v", before, after)
	}
}

// TestRunEjectIO_DriftedCore_ForksRecordedWithDriftedHash is handoff test 2:
// eject also applies to a drifted owner=core path (disk diverges from the
// recorded hash, no prior fork record — FR-4's resolution route), and
// records the *drifted* disk content's hash, not the stale recorded one.
func TestRunEjectIO_DriftedCore_ForksRecordedWithDriftedHash(t *testing.T) {
	target := initV2Project(t, gen1(), "1.0.0-test")

	drifted := "#!/bin/sh\necho drifted-by-user\n"
	writeMigrationDiskFile(t, target, "scripts/run-verify.sh", drifted)

	var out bytes.Buffer
	if err := runEjectIO(target, "scripts/run-verify.sh", &out); err != nil {
		t.Fatalf("runEjectIO: %v", err)
	}

	m := readManifestV2(t, target)
	entry := m.Files["scripts/run-verify.sh"]
	if entry.Owner != scaffold.OwnerFork {
		t.Fatalf("Owner = %q, want %q", entry.Owner, scaffold.OwnerFork)
	}
	wantHash := scaffold.HashBytes([]byte(drifted))
	if entry.DiskHash != wantHash {
		t.Errorf("DiskHash = %q, want %q (drifted disk content hash)", entry.DiskHash, wantHash)
	}

	gotContent := mustReadFile(t, filepath.Join(target, "scripts", "run-verify.sh"))
	if string(gotContent) != drifted {
		t.Errorf("eject must never touch disk: got %q, want %q", gotContent, drifted)
	}
}

// TestRunEjectIO_ErrorMatrix is handoff test 3: every rejection path leaves
// the manifest byte-unchanged and performs zero disk writes.
func TestRunEjectIO_ErrorMatrix(t *testing.T) {
	newTarget := func(t *testing.T) string {
		t.Helper()
		return initV2Project(t, gen1(), "1.0.0-test")
	}

	tests := []struct {
		name    string
		setup   func(t *testing.T, target string)
		path    string
		wantErr string
	}{
		{
			name:    "untracked path",
			path:    "not-a-real-tracked-file.txt",
			wantErr: "not tracked",
		},
		{
			name: "already fork",
			setup: func(t *testing.T, target string) {
				var out bytes.Buffer
				if err := runEjectIO(target, "scripts/run-verify.sh", &out); err != nil {
					t.Fatalf("pre-eject: %v", err)
				}
			},
			path:    "scripts/run-verify.sh",
			wantErr: "already owner=fork",
		},
		{
			name:    "owner=seed",
			path:    "docs/notes.md",
			wantErr: "owner=seed",
		},
		{
			name:    "owner=block (AGENTS.md, also a v2 exception face)",
			path:    "AGENTS.md",
			wantErr: "v2 exception face",
		},
		{
			name:    "owner=block (.gitignore, also a v2 exception face)",
			path:    ".gitignore",
			wantErr: "v2 exception face",
		},
		{
			name:    "v2 exception face: settings.json",
			path:    ".claude/settings.json",
			wantErr: "user-editable directly",
		},
		{
			name:    "v2 exception face: settings snapshot",
			path:    upgrade.SettingsSnapshotRelPath,
			wantErr: "merge-baseline snapshot",
		},
		{
			name: "disk missing",
			setup: func(t *testing.T, target string) {
				if err := os.Remove(filepath.Join(target, "scripts", "run-verify.sh")); err != nil {
					t.Fatalf("removing disk file: %v", err)
				}
			},
			path:    "scripts/run-verify.sh",
			wantErr: "missing on disk",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			target := newTarget(t)
			if tc.setup != nil {
				tc.setup(t, target)
			}

			manifestPath := filepath.Join(target, ".ralph", "manifest.toml")
			manifestBefore := mustReadFile(t, manifestPath)
			diskBefore := snapshotDirHashes(t, target)

			var out bytes.Buffer
			err := runEjectIO(target, tc.path, &out)
			if err == nil {
				t.Fatalf("runEjectIO(%q): expected an error, got nil", tc.path)
			}
			if !strings.Contains(err.Error(), tc.wantErr) {
				t.Errorf("err = %v, want it to contain %q", err, tc.wantErr)
			}

			manifestAfter := mustReadFile(t, manifestPath)
			if !bytes.Equal(manifestBefore, manifestAfter) {
				t.Errorf("manifest must be byte-unchanged on a rejected eject:\nbefore: %s\nafter:  %s", manifestBefore, manifestAfter)
			}
			diskAfter := snapshotDirHashes(t, target)
			if !slicesEqualStrings(diskBefore, diskAfter) {
				t.Errorf("a rejected eject must write zero files; before=%v after=%v", diskBefore, diskAfter)
			}
		})
	}
}

// TestRunEjectIO_LegacyManifest_FailsClosed covers the legacy-manifest arm
// of the error matrix (handoff test 3): eject on a pre-v2 manifest is
// refused with the same fail-closed sentinel `ralph pack add` uses.
func TestRunEjectIO_LegacyManifest_FailsClosed(t *testing.T) {
	target := buildLegacyProject(t)

	manifestPath := filepath.Join(target, ".ralph", "manifest.toml")
	manifestBefore := mustReadFile(t, manifestPath)
	diskBefore := snapshotDirHashes(t, target)

	var out bytes.Buffer
	err := runEjectIO(target, "AGENTS.md", &out)
	if err != errLegacyLayoutFailClosed {
		t.Errorf("err = %v, want errLegacyLayoutFailClosed", err)
	}

	manifestAfter := mustReadFile(t, manifestPath)
	if !bytes.Equal(manifestBefore, manifestAfter) {
		t.Error("manifest must be byte-unchanged on a legacy-manifest eject refusal")
	}
	diskAfter := snapshotDirHashes(t, target)
	if !slicesEqualStrings(diskBefore, diskAfter) {
		t.Errorf("a legacy-manifest eject refusal must write zero files; before=%v after=%v", diskBefore, diskAfter)
	}
}

// --- adopt ---

// TestRunAdoptIO_SingleFork_ResetToTemplate is handoff test 4.
func TestRunAdoptIO_SingleFork_ResetToTemplate(t *testing.T) {
	target := initV2Project(t, gen1(), "1.0.0-test")

	var ejectOut bytes.Buffer
	if err := runEjectIO(target, "scripts/run-verify.sh", &ejectOut); err != nil {
		t.Fatalf("eject: %v", err)
	}
	writeMigrationDiskFile(t, target, "scripts/run-verify.sh", "#!/bin/sh\necho user-owned-fork-content\n")
	commitAllForAdoptTest(t, target)

	var out bytes.Buffer
	err := runAdoptIO(target, adoptOptions{Path: "scripts/run-verify.sh", Yes: true}, strings.NewReader(""), &out, &out)
	if err != nil {
		t.Fatalf("runAdoptIO: %v\n%s", err, out.String())
	}

	gotContent := mustReadFile(t, filepath.Join(target, "scripts", "run-verify.sh"))
	if string(gotContent) != v1RunVerify {
		t.Errorf("disk content = %q, want the current template %q", gotContent, v1RunVerify)
	}

	m := readManifestV2(t, target)
	entry := m.Files["scripts/run-verify.sh"]
	if entry.Owner != scaffold.OwnerCore {
		t.Errorf("Owner = %q, want %q", entry.Owner, scaffold.OwnerCore)
	}
	if entry.ForkedFromVersion != "" {
		t.Errorf("ForkedFromVersion = %q, want empty (fork record cleared)", entry.ForkedFromVersion)
	}
	wantHash := scaffold.HashBytes([]byte(v1RunVerify))
	if entry.TemplateHash != wantHash || entry.DiskHash != wantHash {
		t.Errorf("TemplateHash/DiskHash = %q/%q, want both %q", entry.TemplateHash, entry.DiskHash, wantHash)
	}
}

// TestRunAdoptIO_DriftedCore_ResetToTemplate is handoff test 5: adopt
// applies directly to a drifted owner=core path with no prior fork record.
func TestRunAdoptIO_DriftedCore_ResetToTemplate(t *testing.T) {
	target := initV2Project(t, gen1(), "1.0.0-test")

	writeMigrationDiskFile(t, target, "scripts/run-verify.sh", "#!/bin/sh\necho drifted\n")
	commitAllForAdoptTest(t, target)

	var out bytes.Buffer
	err := runAdoptIO(target, adoptOptions{Path: "scripts/run-verify.sh", Yes: true}, strings.NewReader(""), &out, &out)
	if err != nil {
		t.Fatalf("runAdoptIO: %v\n%s", err, out.String())
	}

	gotContent := mustReadFile(t, filepath.Join(target, "scripts", "run-verify.sh"))
	if string(gotContent) != v1RunVerify {
		t.Errorf("disk content = %q, want the current template %q", gotContent, v1RunVerify)
	}
	m := readManifestV2(t, target)
	if m.Files["scripts/run-verify.sh"].Owner != scaffold.OwnerCore {
		t.Errorf("Owner = %q, want %q", m.Files["scripts/run-verify.sh"].Owner, scaffold.OwnerCore)
	}
}

// TestRunAdoptIO_RetiredFork is handoff test 6: a fork whose path the
// current templates no longer ship is rejected (single-path) or listed as
// skipped (--all).
func TestRunAdoptIO_RetiredFork(t *testing.T) {
	target := initV2Project(t, gen1(), "1.0.0-test")

	var ejectOut bytes.Buffer
	if err := runEjectIO(target, "scripts/old-tool.sh", &ejectOut); err != nil {
		t.Fatalf("eject scripts/old-tool.sh: %v", err)
	}
	commitAllForAdoptTest(t, target)

	// gen2 omits scripts/old-tool.sh entirely (removed upstream) — see
	// v2FixtureGen's oldToolSh doc comment.
	scaffold.EmbeddedFS = gen2().build()
	Version = "1.1.0-test"

	t.Run("single path rejected", func(t *testing.T) {
		var out bytes.Buffer
		err := runAdoptIO(target, adoptOptions{Path: "scripts/old-tool.sh", Yes: true}, strings.NewReader(""), &out, &out)
		if err == nil {
			t.Fatal("expected retired-fork rejection, got nil")
		}
		if !strings.Contains(err.Error(), "retired") {
			t.Errorf("err = %v, want it to mention retired", err)
		}
		gotContent := mustReadFile(t, filepath.Join(target, "scripts", "old-tool.sh"))
		if string(gotContent) != v1OldTool {
			t.Errorf("a rejected adopt must not touch disk: got %q", gotContent)
		}
	})

	t.Run("--all lists it as skipped", func(t *testing.T) {
		var out bytes.Buffer
		err := runAdoptIO(target, adoptOptions{All: true, Yes: true}, strings.NewReader(""), &out, &out)
		if err != nil {
			t.Fatalf("runAdoptIO --all: %v\n%s", err, out.String())
		}
		if !strings.Contains(out.String(), "scripts/old-tool.sh") || !strings.Contains(out.String(), "retired") {
			t.Errorf("--all output must list the retired fork as skipped:\n%s", out.String())
		}
		gotContent := mustReadFile(t, filepath.Join(target, "scripts", "old-tool.sh"))
		if string(gotContent) != v1OldTool {
			t.Errorf("a skipped retired fork must not be touched: got %q", gotContent)
		}
	})
}

// TestRunAdoptIO_DirtyGitTree_ZeroWrites is handoff test 7: a dirty work
// tree aborts before confirmation, with zero writes.
func TestRunAdoptIO_DirtyGitTree_ZeroWrites(t *testing.T) {
	target := initV2Project(t, gen1(), "1.0.0-test")
	// executeInit runs `git init` but never commits, so the tree starts
	// entirely untracked/dirty — no extra edit is needed to dirty it. Eject
	// first so target resolution succeeds (a legitimate adopt target must
	// exist) before the git-clean precondition gets a chance to fire.
	var ejectOut bytes.Buffer
	if err := runEjectIO(target, "scripts/run-verify.sh", &ejectOut); err != nil {
		t.Fatalf("eject: %v", err)
	}

	manifestPath := filepath.Join(target, ".ralph", "manifest.toml")
	manifestBefore := mustReadFile(t, manifestPath)
	diskBefore := snapshotDirHashes(t, target)

	err := runAdoptIO(target, adoptOptions{Path: "scripts/run-verify.sh", Yes: true}, strings.NewReader(""), &bytes.Buffer{}, &bytes.Buffer{})
	if err == nil {
		t.Fatal("adopt on a dirty git work tree: expected an error, got nil")
	}
	if !strings.Contains(err.Error(), "git work tree") {
		t.Errorf("err = %v, want a git-work-tree preflight error", err)
	}

	manifestAfter := mustReadFile(t, manifestPath)
	if !bytes.Equal(manifestBefore, manifestAfter) {
		t.Error("manifest must be byte-unchanged when the dirty-tree precondition fails")
	}
	diskAfter := snapshotDirHashes(t, target)
	if !slicesEqualStrings(diskBefore, diskAfter) {
		t.Errorf("a dirty-tree refusal must write zero files; before=%v after=%v", diskBefore, diskAfter)
	}
}

// filterOutPath removes any "path:hash" entry whose path segment equals
// relPath from a snapshotDirHashes-style slice.
func filterOutPath(entries []string, relPath string) []string {
	out := make([]string, 0, len(entries))
	prefix := filepath.ToSlash(relPath) + ":"
	for _, e := range entries {
		if strings.HasPrefix(filepath.ToSlash(e), prefix) {
			continue
		}
		out = append(out, e)
	}
	return out
}

// TestRunAdoptIO_PreflightFailure_AllBatchZeroWrites is handoff test 8: one
// target's parent directory is replaced with a symlink between eject and
// adopt; --all's preflight must reject the whole batch (including the other,
// otherwise-valid fork) with zero writes.
//
// "packs/languages/golang/verify.sh" sorts BEFORE "scripts/run-verify.sh"
// (--all's targets are processed in path order, see resolveAdoptAllTargets),
// so this test corrupts the LATER-sorting target (scripts/) and leaves the
// EARLIER-sorting one (packs/languages/golang) valid — not the reverse. That
// choice is what actually distinguishes a preflight-everything-before-
// writing-anything design from a write-as-you-go one: if the corrupted
// target sorted first, a write-as-you-go implementation would also fail on
// that same first target before ever reaching the valid one, and the
// untouched-file assertion below would pass under either design (self-review
// M3, docs/reports/self-review-2026-08-19-overlay-scaffold-v2-p5.md). With
// the valid target sorting first instead, only a true preflight-first
// design leaves it un-overwritten when the batch aborts on the later,
// corrupted target.
func TestRunAdoptIO_PreflightFailure_AllBatchZeroWrites(t *testing.T) {
	target := initV2Project(t, gen1(), "1.0.0-test")

	// Modify packs/languages/golang/verify.sh's content before ejecting it,
	// so its fork content genuinely diverges from the current template.
	// Ejecting an UNMODIFIED core path records a fork whose content is
	// byte-identical to the template, which the byte-equality assertion
	// below could not tell apart from "adopt overwrote it with the
	// template" either way — a modified fork is required to make the
	// write-as-you-go distinction observable at all.
	writeMigrationDiskFile(t, target, "packs/languages/golang/verify.sh", "#!/bin/sh\necho user-owned-golang-verify\n")

	for _, p := range []string{"scripts/run-verify.sh", "packs/languages/golang/verify.sh"} {
		var out bytes.Buffer
		if err := runEjectIO(target, p, &out); err != nil {
			t.Fatalf("eject %s: %v", p, err)
		}
	}
	commitAllForAdoptTest(t, target)

	golangVerifyBefore := mustReadFile(t, filepath.Join(target, "packs", "languages", "golang", "verify.sh"))
	manifestPath := filepath.Join(target, ".ralph", "manifest.toml")

	// Corrupt scripts/'s parent chain: replace the directory with a symlink
	// so ValidateRealParentChain refuses it. Re-commit afterward so the
	// git-clean precondition still passes and the failure under test is
	// genuinely the preflight (not the unrelated git-dirty check racing
	// ahead of it).
	scriptsDir := filepath.Join(target, "scripts")
	if err := os.RemoveAll(scriptsDir); err != nil {
		t.Fatalf("removing %s: %v", scriptsDir, err)
	}
	elsewhere := t.TempDir()
	if err := os.Symlink(elsewhere, scriptsDir); err != nil {
		t.Fatalf("symlinking %s: %v", scriptsDir, err)
	}
	commitAllForAdoptTest(t, target)

	manifestBefore := mustReadFile(t, manifestPath)

	err := runAdoptIO(target, adoptOptions{All: true, Yes: true}, strings.NewReader(""), &bytes.Buffer{}, &bytes.Buffer{})
	if err == nil {
		t.Fatal("adopt --all with a symlinked target parent: expected an error, got nil")
	}
	if !strings.Contains(err.Error(), "scripts/run-verify.sh") {
		t.Errorf("err = %v, want it to name the failing target", err)
	}

	golangVerifyAfter := mustReadFile(t, filepath.Join(target, "packs", "languages", "golang", "verify.sh"))
	if !bytes.Equal(golangVerifyBefore, golangVerifyAfter) {
		t.Error("a batch preflight failure must leave the other (valid, earlier-sorting) fork untouched")
	}
	scriptsAfterInfo, statErr := os.Lstat(scriptsDir)
	if statErr != nil || scriptsAfterInfo.Mode()&os.ModeSymlink == 0 {
		t.Error("the corrupted target's directory must remain untouched (still a symlink)")
	}
	manifestAfter := mustReadFile(t, manifestPath)
	if !bytes.Equal(manifestBefore, manifestAfter) {
		t.Error("a batch preflight failure must leave the manifest byte-unchanged")
	}
}

// TestRunAdoptIO_All_ZeroTargets_NoOp is handoff test 9.
func TestRunAdoptIO_All_ZeroTargets_NoOp(t *testing.T) {
	target := initV2Project(t, gen1(), "1.0.0-test")
	commitAllForAdoptTest(t, target)

	manifestPath := filepath.Join(target, ".ralph", "manifest.toml")
	manifestBefore := mustReadFile(t, manifestPath)
	diskBefore := snapshotDirHashes(t, target)

	var out bytes.Buffer
	err := runAdoptIO(target, adoptOptions{All: true, Yes: true}, strings.NewReader(""), &out, &out)
	if err != nil {
		t.Fatalf("runAdoptIO --all with zero targets: %v\n%s", err, out.String())
	}
	if !strings.Contains(out.String(), "Nothing to adopt") {
		t.Errorf("stdout must explicitly say there was nothing to adopt:\n%s", out.String())
	}

	manifestAfter := mustReadFile(t, manifestPath)
	if !bytes.Equal(manifestBefore, manifestAfter) {
		t.Error("--all with zero targets must not write the manifest")
	}
	diskAfter := snapshotDirHashes(t, target)
	if !slicesEqualStrings(diskBefore, diskAfter) {
		t.Errorf("--all with zero targets must write zero files; before=%v after=%v", diskBefore, diskAfter)
	}
}

// TestRunAdoptIO_ConfirmationDeclined_ZeroWrites is handoff test 11.
func TestRunAdoptIO_ConfirmationDeclined_ZeroWrites(t *testing.T) {
	target := initV2Project(t, gen1(), "1.0.0-test")

	var ejectOut bytes.Buffer
	if err := runEjectIO(target, "scripts/run-verify.sh", &ejectOut); err != nil {
		t.Fatalf("eject: %v", err)
	}
	commitAllForAdoptTest(t, target)

	manifestPath := filepath.Join(target, ".ralph", "manifest.toml")
	manifestBefore := mustReadFile(t, manifestPath)
	diskBefore := snapshotDirHashes(t, target)

	var out bytes.Buffer
	err := runAdoptIO(target, adoptOptions{Path: "scripts/run-verify.sh"}, strings.NewReader("n\n"), &out, &out)
	if err != nil {
		t.Fatalf("declining confirmation must not be an error: %v\n%s", err, out.String())
	}
	if !strings.Contains(out.String(), "aborted") {
		t.Errorf("stdout must announce the abort:\n%s", out.String())
	}

	manifestAfter := mustReadFile(t, manifestPath)
	if !bytes.Equal(manifestBefore, manifestAfter) {
		t.Error("a declined confirmation must not write the manifest")
	}
	diskAfter := snapshotDirHashes(t, target)
	if !slicesEqualStrings(diskBefore, diskAfter) {
		t.Errorf("a declined confirmation must write zero files; before=%v after=%v", diskBefore, diskAfter)
	}
}

// TestRunAdoptIO_LegacyManifest_FailsClosed covers adopt's legacy-manifest
// arm of the error matrix (handoff test 3).
func TestRunAdoptIO_LegacyManifest_FailsClosed(t *testing.T) {
	target := buildLegacyProject(t)

	manifestPath := filepath.Join(target, ".ralph", "manifest.toml")
	manifestBefore := mustReadFile(t, manifestPath)
	diskBefore := snapshotDirHashes(t, target)

	err := runAdoptIO(target, adoptOptions{Path: "AGENTS.md", Yes: true}, strings.NewReader(""), &bytes.Buffer{}, &bytes.Buffer{})
	if err != errLegacyLayoutFailClosed {
		t.Errorf("err = %v, want errLegacyLayoutFailClosed", err)
	}

	manifestAfter := mustReadFile(t, manifestPath)
	if !bytes.Equal(manifestBefore, manifestAfter) {
		t.Error("manifest must be byte-unchanged on a legacy-manifest adopt refusal")
	}
	diskAfter := snapshotDirHashes(t, target)
	if !slicesEqualStrings(diskBefore, diskAfter) {
		t.Errorf("a legacy-manifest adopt refusal must write zero files; before=%v after=%v", diskBefore, diskAfter)
	}
}

// --- round trip (AC-13) ---

// TestEjectAdoptRoundTrip_AC13 covers handoff test 10: eject a drifted core
// path (FR-4's "keep the change" route), confirm `ralph upgrade` treats it
// as an advisory (not drift, not exit 3) and leaves it untouched, then
// adopt it back (FR-4's "discard the change" route reused to converge), and
// confirm a subsequent `ralph upgrade` is a true no-op.
func TestEjectAdoptRoundTrip_AC13(t *testing.T) {
	target := initV2Project(t, gen1(), "1.0.0-test")

	relPath := "scripts/run-verify.sh"
	drifted := "#!/bin/sh\necho user-edited-before-eject\n"
	writeMigrationDiskFile(t, target, relPath, drifted)

	var ejectOut bytes.Buffer
	if err := runEjectIO(target, relPath, &ejectOut); err != nil {
		t.Fatalf("eject: %v", err)
	}
	m1 := readManifestV2(t, target)
	if m1.Files[relPath].Owner != scaffold.OwnerFork {
		t.Fatalf("after eject Owner = %q, want %q", m1.Files[relPath].Owner, scaffold.OwnerFork)
	}

	// Upgrade after eject must not report drift for this path (AC-2/FR-4).
	var upOut1, upErr1 bytes.Buffer
	if err := runUpgradeIOWithOptions(target, upgradeOptions{Pager: pagerNever}, strings.NewReader(""), &upOut1, &upErr1, false); err != nil {
		t.Fatalf("upgrade after eject must not fail (expected advisory, not drift): %v\nstderr:\n%s", err, upErr1.String())
	}
	gotAfterUpgrade := mustReadFile(t, filepath.Join(target, relPath))
	if string(gotAfterUpgrade) != drifted {
		t.Errorf("upgrade must leave a fork's content untouched: got %q, want %q", gotAfterUpgrade, drifted)
	}

	commitAllForAdoptTest(t, target)

	var adoptOut bytes.Buffer
	if err := runAdoptIO(target, adoptOptions{Path: relPath, Yes: true}, strings.NewReader(""), &adoptOut, &adoptOut); err != nil {
		t.Fatalf("adopt: %v\n%s", err, adoptOut.String())
	}

	m2 := readManifestV2(t, target)
	entry2 := m2.Files[relPath]
	if entry2.Owner != scaffold.OwnerCore {
		t.Fatalf("after adopt Owner = %q, want %q", entry2.Owner, scaffold.OwnerCore)
	}
	if entry2.ForkedFromVersion != "" {
		t.Errorf("adopt must clear forked_from_version, got %q", entry2.ForkedFromVersion)
	}
	gotAfterAdopt := mustReadFile(t, filepath.Join(target, relPath))
	if string(gotAfterAdopt) != v1RunVerify {
		t.Errorf("adopt must reset disk content to the current template: got %q, want %q", gotAfterAdopt, v1RunVerify)
	}

	commitAllForAdoptTest(t, target)

	before := snapshotTreeHashesExcluding(t, target)
	var upOut2, upErr2 bytes.Buffer
	if err := runUpgradeIOWithOptions(target, upgradeOptions{Pager: pagerNever}, strings.NewReader(""), &upOut2, &upErr2, false); err != nil {
		t.Fatalf("upgrade after adopt: %v\nstderr:\n%s", err, upErr2.String())
	}
	if !strings.Contains(upOut2.String(), "no-op") {
		t.Errorf("post-adopt upgrade must be a converged no-op:\n%s", upOut2.String())
	}
	after := snapshotTreeHashesExcluding(t, target)
	if !slicesEqualStrings(before, after) {
		t.Errorf("post-adopt upgrade must write zero files; before=%v after=%v", before, after)
	}
}
