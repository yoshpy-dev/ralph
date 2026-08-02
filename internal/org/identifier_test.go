package org

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidateIdentifier(t *testing.T) {
	valid := []string{"org-a", "seat-1", "reviewer", "a", "a9", "a-9-b"}
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
		// Cross-review cycle-2 fix: `_` is reserved as the org/seat join
		// separator (herdrAgentName/promptFilePath in spawn.go), so it must
		// never be legal inside a single identifier -- otherwise the join
		// stops being unambiguous. See identifierPattern's doc comment.
		"has_underscore",
		// Uppercase is rejected because herdr's own live-probed agent-name
		// pattern (`^[a-z][a-z0-9_-]{0,31}$`) is lowercase-only; ralph's
		// charset must stay inside herdr's or a valid ralph id could still
		// be rejected by herdr at spawn time.
		"UPPER",
		"Mixed-Case",
	}
	for _, v := range invalid {
		if err := ValidateIdentifier("seat_id", v); err == nil {
			t.Errorf("expected %q to be rejected as an invalid identifier", v)
		}
	}
}

// TestValidateIdentifier_LengthBoundary pins the exact 30-character cutoff:
// identifierPattern allows a leading char plus 29 more (30 total, matching
// herdr's 32-char agent-name limit minus the `_seat` a spawn always adds at
// least one character of). 31 characters must be rejected.
func TestValidateIdentifier_LengthBoundary(t *testing.T) {
	ok := "a" + strings.Repeat("b", 29) // 30 chars
	if len(ok) != 30 {
		t.Fatalf("test setup: expected a 30-char string, got %d", len(ok))
	}
	if err := ValidateIdentifier("seat_id", ok); err != nil {
		t.Errorf("expected a 30-char identifier to be valid, got error: %v", err)
	}

	tooLong := "a" + strings.Repeat("b", 30) // 31 chars
	if len(tooLong) != 31 {
		t.Fatalf("test setup: expected a 31-char string, got %d", len(tooLong))
	}
	if err := ValidateIdentifier("seat_id", tooLong); err == nil {
		t.Errorf("expected a 31-char identifier to be rejected")
	}
}

// TestOrgSpawn_TraversalSeatID_RejectedBeforeAnyFilesystemOrManifestWrite is
// the concrete path-traversal case self-review MEDIUM-2 flagged:
// promptFilePath joins state-dir/prompts/<org_id>_<seat_id>.md, so an
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

// TestHerdrAgentNameAndPromptFilePath_UnambiguousAcrossSplit is the
// cross-review cycle-2 regression: a `-` join could not distinguish
// org_id="a-b"/seat_id="c" from org_id="a"/seat_id="b-c" (both produced
// "a-b-c"). The `_` join must produce two different names/paths for these
// two distinct (org, seat) pairs.
func TestHerdrAgentNameAndPromptFilePath_UnambiguousAcrossSplit(t *testing.T) {
	o, _, _ := testOrg(t)

	nameOne := herdrAgentName("a-b", "c")
	nameTwo := herdrAgentName("a", "b-c")
	if nameOne == nameTwo {
		t.Fatalf("expected herdrAgentName(a-b, c) and herdrAgentName(a, b-c) to differ, both produced %q", nameOne)
	}
	if nameOne != "a-b_c" {
		t.Errorf("herdrAgentName(a-b, c) = %q, want a-b_c", nameOne)
	}
	if nameTwo != "a_b-c" {
		t.Errorf("herdrAgentName(a, b-c) = %q, want a_b-c", nameTwo)
	}

	pathOne, err := o.promptFilePath("a-b", "c")
	if err != nil {
		t.Fatalf("promptFilePath(a-b, c): %v", err)
	}
	pathTwo, err := o.promptFilePath("a", "b-c")
	if err != nil {
		t.Fatalf("promptFilePath(a, b-c): %v", err)
	}
	if pathOne == pathTwo {
		t.Fatalf("expected promptFilePath(a-b, c) and promptFilePath(a, b-c) to differ, both produced %q", pathOne)
	}
	if filepath.Base(pathOne) != "a-b_c.md" {
		t.Errorf("promptFilePath(a-b, c) base = %q, want a-b_c.md", filepath.Base(pathOne))
	}
	if filepath.Base(pathTwo) != "a_b-c.md" {
		t.Errorf("promptFilePath(a, b-c) base = %q, want a_b-c.md", filepath.Base(pathTwo))
	}
}

// TestOrgSpawn_CombinedIdentifierLength_RejectedBeforeAnyManifestWrite
// covers the case where org_id and seat_id are each individually legal
// (<=30 chars, matches identifierPattern) but their `_`-joined herdr agent
// name would exceed herdr's live-probed 32-character agent-name limit.
// Spawn must reject this combination up front, the same way an
// individually-invalid id is rejected, with zero driver calls and no
// manifest write.
func TestOrgSpawn_CombinedIdentifierLength_RejectedBeforeAnyManifestWrite(t *testing.T) {
	o, h, a := testOrg(t)

	// 30 + 1 (join) + 30 = 61 chars, both halves individually valid.
	orgID := "a" + strings.Repeat("b", 29)  // 30 chars
	seatID := "c" + strings.Repeat("d", 29) // 30 chars
	if err := ValidateIdentifier("org_id", orgID); err != nil {
		t.Fatalf("test setup: expected orgID to be individually valid, got %v", err)
	}
	if err := ValidateIdentifier("seat_id", seatID); err != nil {
		t.Fatalf("test setup: expected seatID to be individually valid, got %v", err)
	}

	result := o.Spawn(mustSpawnParams(orgID, seatID))
	if result.Outcome != SpawnOutcomeRejected {
		t.Fatalf("expected SpawnOutcomeRejected for an over-length combined id, got %v (err=%v)", result.Outcome, result.Err)
	}
	if result.Err == nil || !strings.Contains(result.Err.Error(), "32") {
		t.Errorf("expected rejection error to name herdr's 32-character limit, got %v", result.Err)
	}
	if len(h.calls) != 0 || len(a.calls) != 0 {
		t.Fatalf("expected zero driver calls before the combined-length check, got herdr=%v agmsg=%v", h.calls, a.calls)
	}
	if _, err := os.Stat(o.Manifest.Path()); !os.IsNotExist(err) {
		t.Fatalf("expected no manifest file to be created for a rejected over-length combined id, stat err=%v", err)
	}
}

// TestOrgSpawn_CombinedIdentifierLength_ExactlyAtLimit_Spawns documents the
// boundary just inside the limit: org_id+"_"+seat_id totalling exactly 32
// characters must spawn successfully.
func TestOrgSpawn_CombinedIdentifierLength_ExactlyAtLimit_Spawns(t *testing.T) {
	o, _, _ := testOrg(t)

	// 15 + 1 (join) + 16 = 32 chars exactly.
	orgID := "a" + strings.Repeat("b", 14)  // 15 chars
	seatID := "c" + strings.Repeat("d", 15) // 16 chars
	if got := len(orgID) + 1 + len(seatID); got != maxHerdrAgentNameLen {
		t.Fatalf("test setup: expected combined length %d, got %d", maxHerdrAgentNameLen, got)
	}

	result := o.Spawn(mustSpawnParams(orgID, seatID))
	if result.Outcome != SpawnOutcomeSpawned {
		t.Fatalf("expected an exactly-at-limit combined id to spawn, got %v (err=%v)", result.Outcome, result.Err)
	}
}
