package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/yoshpy-dev/ralph/internal/scaffold"
)

// writeManifestWithPacks writes a minimal .ralph/manifest.toml with the given
// Meta.Packs list so checkInstalledPacks reads it as a real manifest.
func writeManifestWithPacks(t *testing.T, dir string, packs []string) {
	t.Helper()
	ralphDir := filepath.Join(dir, ".ralph")
	if err := os.MkdirAll(ralphDir, 0755); err != nil {
		t.Fatal(err)
	}
	m := scaffold.NewManifest("v0.0.0-test")
	m.Meta.Packs = packs
	if err := m.Write(filepath.Join(ralphDir, "manifest.toml")); err != nil {
		t.Fatal(err)
	}
}

// writePackVerifySh creates packs/languages/<lang>/verify.sh in dir so the
// on-disk existence probe passes.
func writePackVerifySh(t *testing.T, dir, lang string) {
	t.Helper()
	vDir := filepath.Join(dir, "packs", "languages", lang)
	if err := os.MkdirAll(vDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(vDir, "verify.sh"), []byte("#!/bin/sh\necho ok\n"), 0755); err != nil {
		t.Fatal(err)
	}
}

// TestCheckInstalledPacks_InstalledPackReported verifies that a pack listed in
// Meta.Packs with verify.sh present on disk produces a "pass" result.
// When the embedded FS is available (built from cmd/ralph/), it uses a real
// pack name; otherwise it uses a synthetic "testlang" pack whose verify.sh
// exists on disk and whose PackFS lookup will fail — but the result still
// exercises the Meta.Packs → named result path.
func TestCheckInstalledPacks_InstalledPackReported(t *testing.T) {
	var lang string
	packs, err := scaffold.AvailablePacks()
	if err == nil && len(packs) > 0 {
		lang = packs[0]
	}
	if lang == "" {
		// EmbeddedFS not available: use "testlang" — PackFS will return "pack
		// not found in templates" warn, which is still a named result and proves
		// the Meta.Packs path is taken (previously "none installed" was returned).
		lang = "testlang"
	}

	dir := t.TempDir()
	writeManifestWithPacks(t, dir, []string{lang})
	writePackVerifySh(t, dir, lang)

	results := checkInstalledPacks(dir)

	// With any pack in Meta.Packs, the result must NOT be "none installed".
	for _, r := range results {
		if r.Detail == "none installed" {
			t.Errorf("got 'none installed' but Meta.Packs = [%s]; pack detection is broken", lang)
		}
	}

	// There must be exactly one result for the pack.
	wantName := "Pack: " + lang
	found := false
	for _, r := range results {
		if r.Name == wantName {
			found = true
			// If PackFS resolved, verify.sh is present → pass.
			// If PackFS did not resolve (no embedded FS), warn is acceptable.
			if r.Status != "pass" && r.Status != "warn" {
				t.Errorf("Pack: %s status = %q, want pass or warn", lang, r.Status)
			}
		}
	}
	if !found {
		t.Errorf("no result named %q; got %+v", wantName, results)
	}
}

// TestCheckInstalledPacks_MissingVerifySh verifies that a pack listed in
// Meta.Packs but without verify.sh on disk produces a "warn" with the correct
// path (packs/languages/<lang>/verify.sh, not project root verify.sh).
// Only meaningful when the embedded FS is available so PackFS resolves and the
// template-level check passes (otherwise the "pack not found in templates" warn
// fires first and masks the on-disk path assertion).
func TestCheckInstalledPacks_MissingVerifySh(t *testing.T) {
	packs, err := scaffold.AvailablePacks()
	if err != nil || len(packs) == 0 {
		t.Skip("embedded FS not available; cannot verify on-disk path warning")
	}
	lang := packs[0]

	dir := t.TempDir()
	writeManifestWithPacks(t, dir, []string{lang})
	// Intentionally do NOT write verify.sh under packs/languages/<lang>/.

	results := checkInstalledPacks(dir)

	found := false
	for _, r := range results {
		if r.Name == "Pack: "+lang {
			found = true
			if r.Status != "warn" {
				t.Errorf("Pack: %s status = %q, want warn (missing verify.sh)", lang, r.Status)
			}
			// Detail must mention the namespaced path, not the project root.
			wantPath := filepath.Join("packs", "languages", lang, "verify.sh")
			if !strings.Contains(r.Detail, wantPath) {
				t.Errorf("Pack: %s detail = %q, want path containing %q", lang, r.Detail, wantPath)
			}
			// Must NOT mention root-level "verify.sh" without the namespace prefix.
			// The previous bug emitted the project root path ("<dir>/verify.sh").
			rootPath := filepath.Join(dir, "verify.sh")
			if strings.Contains(r.Detail, rootPath) {
				t.Errorf("Pack: %s detail mentions wrong root path %q", lang, rootPath)
			}
		}
	}
	if !found {
		t.Errorf("Pack: %s result not found; got %+v", lang, results)
	}
}

// TestCheckInstalledPacks_EmptyMetaPacks verifies that an empty Meta.Packs list
// produces a "pass / none installed" result (not a fallback to checkEmbeddedPacks).
func TestCheckInstalledPacks_EmptyMetaPacks(t *testing.T) {
	dir := t.TempDir()
	writeManifestWithPacks(t, dir, nil)

	results := checkInstalledPacks(dir)

	if len(results) != 1 {
		t.Fatalf("len(results) = %d, want 1; got %+v", len(results), results)
	}
	r := results[0]
	if r.Status != "pass" {
		t.Errorf("status = %q, want pass (none installed)", r.Status)
	}
	if !strings.Contains(r.Detail, "none installed") {
		t.Errorf("detail = %q, want 'none installed'", r.Detail)
	}
}

// TestCheckInstalledPacks_NoManifestFallback verifies that a missing manifest
// falls back to checkEmbeddedPacks (not an error result).
func TestCheckInstalledPacks_NoManifestFallback(t *testing.T) {
	dir := t.TempDir()
	// No .ralph/manifest.toml written.

	results := checkInstalledPacks(dir)

	// checkEmbeddedPacks lists all embedded packs; each gets a checkResult.
	// We just verify the fallback runs without panic and returns no "fail" results.
	for _, r := range results {
		if r.Status == "fail" {
			t.Errorf("fallback result status = fail; got %+v", r)
		}
	}
}
