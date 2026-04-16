package action

import (
	"os"
	"path/filepath"
	"testing"

	tea "charm.land/bubbletea/v2"
)

func TestValidateSliceName(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{"valid simple", "slice-1", false},
		{"valid with dash", "my-slice-name", false},
		{"valid with numbers", "slice123", false},
		{"empty", "", true},
		{"path traversal", "../../etc/passwd", true},
		{"double dot", "slice..name", true},
		{"slash", "slice/name", true},
		{"backslash", "slice\\name", true},
		{"semicolon", "slice;rm -rf /", true},
		{"pipe", "slice|cat", true},
		{"ampersand", "slice&echo", true},
		{"dollar", "slice$HOME", true},
		{"backtick", "slice`cmd`", true},
		{"space", "slice name", true},
		{"tab", "slice\tname", true},
		{"newline", "slice\nname", true},
		{"single quote", "slice'name", true},
		{"double quote", "slice\"name", true},
		{"parentheses", "slice(name)", true},
		{"brackets", "slice[name]", true},
		{"braces", "slice{name}", true},
		{"angle brackets", "slice<name>", true},
		{"exclamation", "slice!name", true},
		{"hash", "slice#name", true},
		{"tilde", "slice~name", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateSliceName(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateSliceName(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			}
		})
	}
}

func TestNewExecutor(t *testing.T) {
	t.Run("valid repo root", func(t *testing.T) {
		dir := t.TempDir()
		ralphPath := filepath.Join(dir, "scripts", "ralph")
		if err := os.MkdirAll(filepath.Dir(ralphPath), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(ralphPath, []byte("#!/bin/sh\n"), 0o755); err != nil {
			t.Fatal(err)
		}

		exec, err := NewExecutor(dir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if exec.RalphPath() != ralphPath {
			t.Errorf("RalphPath() = %q, want %q", exec.RalphPath(), ralphPath)
		}
		if exec.RepoRoot() != dir {
			t.Errorf("RepoRoot() = %q, want %q", exec.RepoRoot(), dir)
		}
	})

	t.Run("missing ralph script", func(t *testing.T) {
		dir := t.TempDir()
		_, err := NewExecutor(dir)
		if err == nil {
			t.Fatal("expected error for missing ralph script")
		}
	})
}

func TestBuildCommand(t *testing.T) {
	dir := t.TempDir()
	ralphPath := filepath.Join(dir, "scripts", "ralph")
	if err := os.MkdirAll(filepath.Dir(ralphPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(ralphPath, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}

	exec, err := NewExecutor(dir)
	if err != nil {
		t.Fatal(err)
	}

	cmd := exec.BuildCommand("retry", "slice-1")
	if cmd.Path != ralphPath && cmd.Args[0] != ralphPath {
		t.Errorf("expected command to use ralph at %q", ralphPath)
	}
	if len(cmd.Args) < 3 {
		t.Fatalf("expected at least 3 args, got %d: %v", len(cmd.Args), cmd.Args)
	}
	if cmd.Args[1] != "retry" || cmd.Args[2] != "slice-1" {
		t.Errorf("args = %v, want [<path> retry slice-1]", cmd.Args)
	}
	if cmd.Dir != dir {
		t.Errorf("Dir = %q, want %q", cmd.Dir, dir)
	}
}

func TestRunAsync(t *testing.T) {
	dir := t.TempDir()
	ralphPath := filepath.Join(dir, "scripts", "ralph")
	if err := os.MkdirAll(filepath.Dir(ralphPath), 0o755); err != nil {
		t.Fatal(err)
	}
	// Create a script that echoes its arguments
	script := "#!/bin/sh\necho \"called with: $@\"\n"
	if err := os.WriteFile(ralphPath, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}

	exec, err := NewExecutor(dir)
	if err != nil {
		t.Fatal(err)
	}

	var gotOutput string
	var gotErr error
	cmd := exec.RunAsync(func(output string, err error) tea.Msg {
		gotOutput = output
		gotErr = err
		return nil
	}, "status", "--json")

	// Execute the command synchronously for testing
	cmd()

	if gotErr != nil {
		t.Errorf("unexpected error: %v", gotErr)
	}
	if gotOutput == "" {
		t.Error("expected non-empty output")
	}
	expected := "called with: status --json\n"
	if gotOutput != expected {
		t.Errorf("output = %q, want %q", gotOutput, expected)
	}
}
