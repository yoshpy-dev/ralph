package upgrade

import (
	"bytes"
	"encoding/json"
	"slices"
	"testing"
)

// canon parses s as JSON and re-renders it through the package's own
// canonical serializer, so tests can express expected results as readable
// JSON literals without hand-matching exact whitespace/formatting.
func canon(t *testing.T, s string) []byte {
	t.Helper()
	v, err := parseSettingsDoc([]byte(s))
	if err != nil {
		t.Fatalf("parseSettingsDoc(%q) error: %v", s, err)
	}
	return marshalOrdered(v)
}

func TestMergeOwnedSettings_EmptyCurrentEqualsTemplate(t *testing.T) {
	newOwned := []byte(`{
		"env": {"FOO": "bar"},
		"permissions": {"allow": ["Bash(ls)"], "deny": ["Bash(rm -rf /)"]},
		"hooks": {"PreToolUse": [{"matcher": "Bash", "hooks": [{"type": "command", "command": "echo hi"}]}]}
	}`)
	oldOwned := []byte(`{}`)

	got, err := MergeOwnedSettings(nil, oldOwned, newOwned)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := canon(t, string(newOwned))
	if !bytes.Equal(got.Content, want) {
		t.Fatalf("Content =\n%s\nwant\n%s", got.Content, want)
	}
	if !got.Changed {
		t.Fatalf("Changed = false, want true")
	}
}

func TestMergeOwnedSettings_EmptyCurrentAndEmptyTemplatesIsNoop(t *testing.T) {
	got, err := MergeOwnedSettings(nil, []byte(`{}`), []byte(`{}`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Changed {
		t.Fatalf("Changed = true, want false ({} vs {} is a no-op)")
	}
	if !bytes.Equal(got.Content, canon(t, `{}`)) {
		t.Fatalf("Content = %s, want {}", got.Content)
	}
}

func TestMergeOwnedSettings_UserAddedPermissionPreserved(t *testing.T) {
	current := `{"permissions": {"allow": ["Bash(ls)", "Bash(git status)"]}}`
	old := `{"permissions": {"allow": ["Bash(ls)"]}}`
	newT := `{"permissions": {"allow": ["Bash(ls)"]}}`

	got, err := MergeOwnedSettings([]byte(current), []byte(old), []byte(newT))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := canon(t, current)
	if !bytes.Equal(got.Content, want) {
		t.Fatalf("Content =\n%s\nwant\n%s", got.Content, want)
	}
	if got.Changed {
		t.Fatalf("Changed = true, want false (nothing ralph-owned changed)")
	}
}

func TestMergeOwnedSettings_StaleRalphPermissionRemoved(t *testing.T) {
	current := `{"permissions": {"allow": ["Bash(ls)", "Bash(rm -rf /)"]}}`
	old := `{"permissions": {"allow": ["Bash(ls)", "Bash(rm -rf /)"]}}`
	newT := `{"permissions": {"allow": ["Bash(ls)"]}}`

	got, err := MergeOwnedSettings([]byte(current), []byte(old), []byte(newT))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := canon(t, `{"permissions": {"allow": ["Bash(ls)"]}}`)
	if !bytes.Equal(got.Content, want) {
		t.Fatalf("Content =\n%s\nwant\n%s", got.Content, want)
	}
	if !got.Changed {
		t.Fatalf("Changed = false, want true")
	}
}

func TestMergeOwnedSettings_RalphPermissionAdded(t *testing.T) {
	current := `{"permissions": {"allow": ["Bash(ls)"]}}`
	old := `{"permissions": {"allow": ["Bash(ls)"]}}`
	newT := `{"permissions": {"allow": ["Bash(ls)", "Bash(git status)"]}}`

	got, err := MergeOwnedSettings([]byte(current), []byte(old), []byte(newT))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := canon(t, `{"permissions": {"allow": ["Bash(ls)", "Bash(git status)"]}}`)
	if !bytes.Equal(got.Content, want) {
		t.Fatalf("Content =\n%s\nwant\n%s", got.Content, want)
	}
}

// TestMergeOwnedSettings_UserPermissionIdenticalToRemovedRalphEntry documents
// known behavior: entry identity is structural (deep equality), not
// provenance-tracked. If a user's own entry happens to be byte-for-byte
// identical to an entry that was ralph-owned in oldOwned, it is
// indistinguishable from the ralph entry and is removed along with it when
// newOwned drops it — even though, informally, "the user added it too".
func TestMergeOwnedSettings_UserPermissionIdenticalToRemovedRalphEntry(t *testing.T) {
	current := `{"permissions": {"allow": ["Bash(rm -rf /)"]}}`
	old := `{"permissions": {"allow": ["Bash(rm -rf /)"]}}`
	newT := `{"permissions": {"allow": []}}`

	got, err := MergeOwnedSettings([]byte(current), []byte(old), []byte(newT))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := canon(t, `{}`)
	if !bytes.Equal(got.Content, want) {
		t.Fatalf("Content =\n%s\nwant\n%s (permissions should be pruned entirely)", got.Content, want)
	}
}

func TestMergeOwnedSettings_HooksEventArrayAddUpdateRemove(t *testing.T) {
	current := `{"hooks": {
		"PreToolUse": [{"matcher": "Bash", "hooks": [{"type": "command", "command": "old.sh"}]}],
		"PostToolUse": [{"matcher": "Write", "hooks": [{"type": "command", "command": "post.sh"}]}]
	}}`
	old := `{"hooks": {
		"PreToolUse": [{"matcher": "Bash", "hooks": [{"type": "command", "command": "old.sh"}]}],
		"PostToolUse": [{"matcher": "Write", "hooks": [{"type": "command", "command": "post.sh"}]}]
	}}`
	newT := `{"hooks": {
		"PreToolUse": [{"matcher": "Bash", "hooks": [{"type": "command", "command": "new.sh"}]}],
		"SessionStart": [{"matcher": "", "hooks": [{"type": "command", "command": "start.sh"}]}]
	}}`

	got, err := MergeOwnedSettings([]byte(current), []byte(old), []byte(newT))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := canon(t, `{"hooks": {
		"PreToolUse": [{"matcher": "Bash", "hooks": [{"type": "command", "command": "new.sh"}]}],
		"SessionStart": [{"matcher": "", "hooks": [{"type": "command", "command": "start.sh"}]}]
	}}`)
	if !bytes.Equal(got.Content, want) {
		t.Fatalf("Content =\n%s\nwant\n%s", got.Content, want)
	}
}

func TestMergeOwnedSettings_UserAddedHookEntryPreserved(t *testing.T) {
	current := `{"hooks": {"PreToolUse": [
		{"matcher": "Bash", "hooks": [{"type": "command", "command": "ralph.sh"}]},
		{"matcher": "Write", "hooks": [{"type": "command", "command": "user.sh"}]}
	]}}`
	old := `{"hooks": {"PreToolUse": [
		{"matcher": "Bash", "hooks": [{"type": "command", "command": "ralph.sh"}]}
	]}}`
	newT := old

	got, err := MergeOwnedSettings([]byte(current), []byte(old), []byte(newT))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := canon(t, current)
	if !bytes.Equal(got.Content, want) {
		t.Fatalf("Content =\n%s\nwant\n%s (user's own hook entry must survive)", got.Content, want)
	}
	if got.Changed {
		t.Fatalf("Changed = true, want false")
	}
}

func TestMergeOwnedSettings_UnknownTopLevelKeyPreserved(t *testing.T) {
	current := `{"customField": {"foo": "bar"}, "env": {"FOO": "old"}}`
	old := `{"env": {"FOO": "old"}}`
	newT := `{"env": {"FOO": "new"}}`

	got, err := MergeOwnedSettings([]byte(current), []byte(old), []byte(newT))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := canon(t, `{"customField": {"foo": "bar"}, "env": {"FOO": "new"}}`)
	if !bytes.Equal(got.Content, want) {
		t.Fatalf("Content =\n%s\nwant\n%s", got.Content, want)
	}
}

func TestMergeOwnedSettings_EnvTemplateUpdateAndUserKeyPreserved(t *testing.T) {
	current := `{"env": {"FOO": "old", "USER_KEY": "keep-me"}}`
	old := `{"env": {"FOO": "old"}}`
	newT := `{"env": {"FOO": "new"}}`

	got, err := MergeOwnedSettings([]byte(current), []byte(old), []byte(newT))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := canon(t, `{"env": {"FOO": "new", "USER_KEY": "keep-me"}}`)
	if !bytes.Equal(got.Content, want) {
		t.Fatalf("Content =\n%s\nwant\n%s", got.Content, want)
	}
}

func TestMergeOwnedSettings_DuplicateEntriesDeduplication(t *testing.T) {
	current := `{"permissions": {"allow": ["Bash(ls)", "Bash(ls)", "Bash(git status)"]}}`
	old := `{"permissions": {"allow": ["Bash(ls)"]}}`
	newT := `{"permissions": {"allow": ["Bash(ls)"]}}`

	got, err := MergeOwnedSettings([]byte(current), []byte(old), []byte(newT))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := canon(t, `{"permissions": {"allow": ["Bash(ls)", "Bash(git status)"]}}`)
	if !bytes.Equal(got.Content, want) {
		t.Fatalf("Content =\n%s\nwant\n%s", got.Content, want)
	}
}

func TestMergeOwnedSettings_DeterministicDoubleMergeIsNoop(t *testing.T) {
	current := `{
		"customField": "keep me",
		"env": {"FOO": "old", "USER_KEY": "keep-me"},
		"permissions": {
			"allow": ["Bash(ls)", "Bash(rm -rf /)", "Bash(git status)"],
			"deny": ["Bash(curl)"]
		},
		"hooks": {
			"PreToolUse": [
				{"matcher": "Bash", "hooks": [{"type": "command", "command": "old.sh"}]},
				{"matcher": "Write", "hooks": [{"type": "command", "command": "user.sh"}]}
			]
		}
	}`
	old := `{
		"env": {"FOO": "old"},
		"permissions": {
			"allow": ["Bash(ls)", "Bash(rm -rf /)"],
			"deny": ["Bash(curl)"]
		},
		"hooks": {
			"PreToolUse": [{"matcher": "Bash", "hooks": [{"type": "command", "command": "old.sh"}]}]
		}
	}`
	newT := `{
		"env": {"FOO": "new"},
		"permissions": {
			"allow": ["Bash(ls)", "Bash(git status)"],
			"deny": ["Bash(curl)", "Bash(wget)"]
		},
		"hooks": {
			"PreToolUse": [{"matcher": "Bash", "hooks": [{"type": "command", "command": "new.sh"}]}],
			"SessionStart": [{"matcher": "", "hooks": [{"type": "command", "command": "start.sh"}]}]
		}
	}`

	first, err := MergeOwnedSettings([]byte(current), []byte(old), []byte(newT))
	if err != nil {
		t.Fatalf("first merge: unexpected error: %v", err)
	}
	if !first.Changed {
		t.Fatalf("first.Changed = false, want true")
	}

	second, err := MergeOwnedSettings(first.Content, []byte(old), []byte(newT))
	if err != nil {
		t.Fatalf("second merge: unexpected error: %v", err)
	}
	if second.Changed {
		t.Fatalf("second.Changed = true, want false (merge of own output must be a no-op)")
	}
	if !bytes.Equal(second.Content, first.Content) {
		t.Fatalf("second.Content =\n%s\nwant (same as first)\n%s", second.Content, first.Content)
	}
}

func TestMergeOwnedSettings_MalformedCurrentJSONReturnsError(t *testing.T) {
	current := []byte(`{"env": `) // truncated
	_, err := MergeOwnedSettings(current, []byte(`{}`), []byte(`{}`))
	if err == nil {
		t.Fatalf("expected error for malformed current JSON, got nil")
	}
}

func TestMergeOwnedSettings_MalformedTemplateJSONReturnsError(t *testing.T) {
	t.Run("old owned", func(t *testing.T) {
		_, err := MergeOwnedSettings([]byte(`{}`), []byte(`{not json`), []byte(`{}`))
		if err == nil {
			t.Fatalf("expected error for malformed oldOwned JSON, got nil")
		}
	})
	t.Run("new owned", func(t *testing.T) {
		_, err := MergeOwnedSettings([]byte(`{}`), []byte(`{}`), []byte(`{not json`))
		if err == nil {
			t.Fatalf("expected error for malformed newOwned JSON, got nil")
		}
	})
}

func TestMergeOwnedSettings_NonObjectRootRejected(t *testing.T) {
	_, err := MergeOwnedSettings([]byte(`[1,2,3]`), []byte(`{}`), []byte(`{}`))
	if err == nil {
		t.Fatalf("expected error for non-object root, got nil")
	}
}

// TestOwnedSettingsPaths_AnchorsMergeBehavior is a deliberate change-detector
// on the declared OwnedSettingsPaths list (edit the merge handlers and this
// test's `want` literal together) plus a separate regression test that an
// un-owned top-level key a template ships outside the owned paths is never
// introduced into the merged result. It does not verify that each handler
// implements its corresponding owned path.
func TestOwnedSettingsPaths_AnchorsMergeBehavior(t *testing.T) {
	want := []string{"env", "permissions.allow", "permissions.deny", "hooks"}
	if got := OwnedSettingsPaths[:]; !slices.Equal(got, want) {
		t.Fatalf("OwnedSettingsPaths = %v, want %v (update the merge handlers and this test together)", got, want)
	}

	current := []byte(`{"model": "user-choice"}`)
	newOwned := []byte(`{"model": "template-choice", "outputStyle": "x", "env": {"A": "1"}}`)
	res, err := MergeOwnedSettings(current, []byte(`{}`), newOwned)
	if err != nil {
		t.Fatalf("MergeOwnedSettings: %v", err)
	}
	var out map[string]any
	if err := json.Unmarshal(res.Content, &out); err != nil {
		t.Fatalf("unmarshal result: %v", err)
	}
	if out["model"] != "user-choice" {
		t.Errorf("un-owned key 'model' = %v, want user's value preserved", out["model"])
	}
	if _, ok := out["outputStyle"]; ok {
		t.Error("un-owned template key 'outputStyle' must not be introduced by the merge")
	}
	if _, ok := out["env"]; !ok {
		t.Error("owned path 'env' from the template must be merged in")
	}
}
