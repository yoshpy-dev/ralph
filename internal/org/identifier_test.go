package org

import (
	"os"
	"path/filepath"
	"testing"
)

func TestValidateIdentifier(t *testing.T) {
	valid := []string{"org-a", "seat-1", "reviewer", "a", "ORG_1", "a-b_c-D9"}
	for _, v := range valid {
		if err := ValidateIdentifier("seat_id", v); err != nil {
			t.Errorf("expected %q to be a valid identifier, got error: %v", v, err)
		}
	}

	invalid := []string{
		"",
		"../../../../tmp/pwn",
		"../escape",
		"a/b",
		`a\b`,
		"-leading-hyphen",
		"_leading-underscore",
		"has space",
		"has.dot",
	}
	for _, v := range invalid {
		if err := ValidateIdentifier("seat_id", v); err == nil {
			t.Errorf("expected %q to be rejected as an invalid identifier", v)
		}
	}
}

// TestOrgSpawn_TraversalSeatID_RejectedBeforeAnyFilesystemOrManifestWrite is
// the concrete path-traversal case self-review MEDIUM-2 flagged:
// promptFilePath joins state-dir/prompts/<org_id>-<seat_id>.md, so an
// unvalidated --id could normalize outside the state dir. Spawn's identifier
// check must reject it before the manifest is even read, so: (1) no
// manifest file is created at all, (2) no file is written anywhere,
// including inside the state dir's own prompts/ subdirectory.
func TestOrgSpawn_TraversalSeatID_RejectedBeforeAnyFilesystemOrManifestWrite(t *testing.T) {
	o, h, a := testOrg(t)
	stateDir := filepath.Dir(o.Manifest.Path())

	p := mustSpawnParams("org-a", "../../../../../../tmp/pwn")
	result := o.Spawn(p)

	if result.Outcome != SpawnOutcomeRejected {
		t.Fatalf("expected SpawnOutcomeRejected for a traversal seat_id, got %v (err=%v)", result.Outcome, result.Err)
	}
	if result.Err == nil {
		t.Fatal("expected a non-nil Err for a rejected traversal seat_id")
	}

	if len(h.calls) != 0 || len(a.calls) != 0 {
		t.Fatalf("expected zero driver calls before identifier validation, got herdr=%v agmsg=%v", h.calls, a.calls)
	}

	if _, err := os.Stat(o.Manifest.Path()); !os.IsNotExist(err) {
		t.Fatalf("expected no manifest file to be created for a rejected traversal seat_id, stat err=%v", err)
	}

	// stateDir itself is t.TempDir() and pre-exists the call; what matters is
	// that Spawn wrote nothing into it -- no manifest file, no prompts/
	// subdirectory, no stray file at all.
	entries, err := os.ReadDir(stateDir)
	if err != nil {
		t.Fatalf("read state dir: %v", err)
	}
	for _, e := range entries {
		t.Errorf("expected the state dir to remain empty for a rejected spawn, found %q", e.Name())
	}
	if _, err := os.Stat(filepath.Join(stateDir, "prompts")); !os.IsNotExist(err) {
		t.Fatalf("expected no prompts/ subdirectory to be created, stat err=%v", err)
	}
}

// TestOrgSpawn_InvalidOrgID_RejectedBeforeAnyManifestWrite covers the OrgID
// half of the same check: a malformed --org-id must be rejected the same
// way, with zero driver calls and no manifest write attempted.
func TestOrgSpawn_InvalidOrgID_RejectedBeforeAnyManifestWrite(t *testing.T) {
	o, h, a := testOrg(t)

	result := o.Spawn(mustSpawnParams("../escape", "seat-1"))
	if result.Outcome != SpawnOutcomeRejected {
		t.Fatalf("expected SpawnOutcomeRejected for a malformed org_id, got %v (err=%v)", result.Outcome, result.Err)
	}
	if len(h.calls) != 0 || len(a.calls) != 0 {
		t.Fatalf("expected zero driver calls before identifier validation, got herdr=%v agmsg=%v", h.calls, a.calls)
	}
	if _, err := os.Stat(o.Manifest.Path()); !os.IsNotExist(err) {
		t.Fatalf("expected no manifest file to be created for a rejected org_id, stat err=%v", err)
	}
}

// TestOrgSpawn_ValidIdentifiers_PassValidationAndSpawn documents the other
// side of the same gate: ordinary hyphenated org_id/seat_id values (the
// shape every other spawn test in this package already uses) are unaffected
// by the new identifier check and reach SpawnOutcomeSpawned exactly as
// before.
func TestOrgSpawn_ValidIdentifiers_PassValidationAndSpawn(t *testing.T) {
	o, _, _ := testOrg(t)

	result := o.Spawn(mustSpawnParams("org-a", "seat-1"))
	if result.Outcome != SpawnOutcomeSpawned {
		t.Fatalf("expected valid identifiers to spawn successfully, got %v (err=%v)", result.Outcome, result.Err)
	}
}
