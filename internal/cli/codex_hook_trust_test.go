package cli

import (
	"errors"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"
)

// Tests for codex_hook_trust.go (plan 2026-11-12-codex-hook-trust-check,
// issue #310). None of them reads the real ~/.codex: each passes its own
// config path. None runs in parallel (newLegacyLedgerRepo chdirs and sets
// environment variables).

// codexHookTrustTempDir is t.TempDir with its symlinks resolved (macOS keeps
// temp dirs under /var, a symlink to /private/var), so a fixture's lexical
// and resolved roots agree unless the test adds a symlink of its own.
func codexHookTrustTempDir(t *testing.T) string {
	t.Helper()
	dir, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatalf("eval symlinks: %v", err)
	}
	return dir
}

// writeCodexHookTrustFile writes content to path, creating its directories.
func writeCodexHookTrustFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

// mkdirCodexHookTrust creates path and its parents.
func mkdirCodexHookTrust(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", path, err)
	}
}

// makeCodexHookTrustCheckout makes dir look like a main checkout to codex: a
// .git directory holding HEAD. No git command runs.
func makeCodexHookTrustCheckout(t *testing.T, dir string) {
	t.Helper()
	writeCodexHookTrustFile(t, filepath.Join(dir, ".git", "HEAD"), "ref: refs/heads/main\n")
}

// symlinkCodexHookTrust makes link point at target, skipping the test where
// symlinks are not available.
func symlinkCodexHookTrust(t *testing.T, target, link string) {
	t.Helper()
	if err := os.Symlink(target, link); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
}

// chdirCodexHookTrustAs makes dir the working directory with $PWD spelled as
// dir, as a shell that ran cd dir leaves it. os.Getwd then returns dir even
// when it goes through a symlink, so a relative path such as doctor's "." is
// made absolute in that spelling, not the real path. It fails the test when
// os.Getwd does not return dir.
func chdirCodexHookTrustAs(t *testing.T, dir string) {
	t.Helper()
	t.Chdir(dir)
	t.Setenv("PWD", dir)
	if wd, err := os.Getwd(); err != nil || wd != dir {
		t.Fatalf("os.Getwd() = %q, %v; want the spelling %s", wd, err, dir)
	}
}

// codexHookTrustStateTable renders one [hooks.state."<key>"] table.
func codexHookTrustStateTable(key, body string) string {
	return "[hooks.state." + strconv.Quote(key) + "]\n" + body + "\n"
}

// codexHookTrustTails lists the Tail of each item.
func codexHookTrustTails(items []codexHookTrustItem) []string {
	tails := []string{}
	for _, item := range items {
		tails = append(tails, item.Tail)
	}
	return tails
}

func TestCodexHookTrust_EventLabelsAreCodexs14(t *testing.T) {
	want := map[string]string{
		"PreToolUse": "pre_tool_use", "PermissionRequest": "permission_request",
		"PostToolUse": "post_tool_use", "PreCompact": "pre_compact", "PostCompact": "post_compact",
		"PreModelRequest": "pre_model_request", "PostModelRequest": "post_model_request",
		"SessionStart": "session_start", "SessionEnd": "session_end",
		"UserPromptSubmit": "user_prompt_submit", "SubagentStart": "subagent_start",
		"SubagentStop": "subagent_stop", "Stop": "stop", "Interrupt": "interrupt",
	}
	if len(want) != 14 {
		t.Fatalf("the expectation itself must list 14 events, has %d", len(want))
	}
	if !maps.Equal(codexHookEventLabels, want) {
		t.Errorf("codexHookEventLabels = %v, want %v", codexHookEventLabels, want)
	}
}

func TestCodexHookTrust_SlotsCountPositionsLikeCodex(t *testing.T) {
	data := `{
  "description": "ignored",
  "hooks": {
    "SessionStart": [{"hooks": [{"type": "command", "command": "echo start"}]}],
    "PreToolUse": [
      {"matcher": "Bash", "hooks": [
        {"type": "command", "command": "  \t"},
        {"type": "command", "command": "guard.sh", "timeout": 30}
      ]},
      {"matcher": "Edit", "hooks": [
        {"type": "command", "command": "a"},
        {"type": "command", "command": "b"}
      ]}
    ],
    "PostToolUse": [{"hooks": [{"type": "command", "command": ""}]}],
    "UserPromptSubmit": [{"hooks": [{"type": "mcp_tool", "server": "s", "tool": "t"}]}],
    "NotACodexEvent": [{"hooks": [{"type": "command", "command": "x"}]}],
    "Stop": [{"hooks": []}]
  }
}`
	slots, err := parseCodexHookSlots([]byte(data))
	if err != nil {
		t.Fatalf("parseCodexHookSlots: %v", err)
	}
	var tails []string
	for _, slot := range slots {
		tails = append(tails, slot.Tail())
	}
	// The blank command at PreToolUse 0:0 gets no slot, but the next handler
	// keeps position 1; the empty PostToolUse command, the unknown event, and
	// the empty Stop group get none. A non-command handler is counted.
	want := []string{"pre_tool_use:0:1", "pre_tool_use:1:0", "pre_tool_use:1:1", "session_start:0:0", "user_prompt_submit:0:0"}
	if !slices.Equal(tails, want) {
		t.Fatalf("slot tails = %v, want %v", tails, want)
	}
	if got := codexHookKey("/r", slots[0]); got != "/r/.codex/hooks.json:pre_tool_use:0:1" {
		t.Errorf("codexHookKey = %q", got)
	}
}

func TestCodexHookTrust_SlotsOfAFileWithNoHandlers(t *testing.T) {
	for _, data := range []string{`{}`, `{"hooks": {}}`, `{"hooks": {"Stop": []}}`, `{"hooks": {"Stop": [{"hooks": []}]}}`, `{"description": "x", "hooks": null}`} {
		slots, err := parseCodexHookSlots([]byte(data))
		if err != nil {
			t.Errorf("%s: unexpected error %v", data, err)
		}
		if len(slots) != 0 {
			t.Errorf("%s: slots = %v, want none", data, slots)
		}
	}
}

func TestCodexHookTrust_SlotsRejectWhatIsNotCodexsShape(t *testing.T) {
	cases := []struct {
		data string
		want string // the start of the error text
	}{
		{`not json`, "not valid JSON (at byte "},
		{`{"hooks": {}`, "not valid JSON (at byte "},
		{`null`, "not a JSON object"},
		{`[]`, "unexpected JSON array at the top level"},
		{`"hooks"`, "unexpected JSON string at the top level"},
		{`{"hooks": []}`, `unexpected JSON array at "hooks"`},
		{`{"hooks": "x"}`, `unexpected JSON string at "hooks"`},
		{`{"hooks": {"PreToolUse": {}}}`, `unexpected JSON object at "hooks.PreToolUse"`},
		{`{"hooks": {"PreToolUse": [{"hooks": [{"type": "command", "command": 5}]}]}}`, `unexpected JSON number at "hooks.PreToolUse.0.hooks.0.command"`},
	}
	for _, tc := range cases {
		slots, err := parseCodexHookSlots([]byte(tc.data))
		if err == nil {
			t.Errorf("%s: no error, slots %v", tc.data, slots)
			continue
		}
		if !strings.HasPrefix(err.Error(), tc.want) {
			t.Errorf("%s: error %q, want it to start with %q", tc.data, err, tc.want)
		}
		if strings.Contains(err.Error(), "Go ") || strings.Contains(err.Error(), "cli.") {
			t.Errorf("%s: error %q names Go types", tc.data, err)
		}
	}
}

func TestCodexHookTrust_RootFollowsCodexsWalk(t *testing.T) {
	tmp := codexHookTrustTempDir(t)
	mainRoot := filepath.Join(tmp, "main")
	makeCodexHookTrustCheckout(t, mainRoot)
	mkdirCodexHookTrust(t, filepath.Join(mainRoot, "a", "b"))
	writeCodexHookTrustFile(t, filepath.Join(mainRoot, "a", "file.txt"), "x")
	// A .git directory without HEAD is passed over.
	mkdirCodexHookTrust(t, filepath.Join(mainRoot, "nested", ".git"))
	mkdirCodexHookTrust(t, filepath.Join(mainRoot, "nested", "x"))
	// A .git file whose gitdir is not under worktrees (a submodule's).
	submodule := filepath.Join(tmp, "submodule")
	writeCodexHookTrustFile(t, filepath.Join(submodule, ".git"), "gitdir: "+filepath.Join(tmp, "main", ".git", "modules", "sub")+"\n")
	// A .git file pointing into worktrees of a main checkout that has no .git.
	orphan := filepath.Join(tmp, "orphan")
	writeCodexHookTrustFile(t, filepath.Join(orphan, ".git"), "gitdir: "+filepath.Join(tmp, "gone", ".git", "worktrees", "orphan")+"\n")
	// A relative gitdir is taken from the checkout.
	relative := filepath.Join(tmp, "relative")
	writeCodexHookTrustFile(t, filepath.Join(relative, ".git"), "gitdir: ../main/.git/worktrees/relative\n")
	// A .git file that is not a gitdir line.
	garbled := filepath.Join(tmp, "garbled")
	writeCodexHookTrustFile(t, filepath.Join(garbled, ".git"), "not a gitdir line\n")
	noGit := filepath.Join(tmp, "nogit", "sub")
	mkdirCodexHookTrust(t, noGit)

	cases := []struct {
		name, dir, want string
	}{
		{"main checkout", mainRoot, mainRoot},
		{"subdirectory of the main checkout", filepath.Join(mainRoot, "a", "b"), mainRoot},
		{"file in the main checkout", filepath.Join(mainRoot, "a", "file.txt"), mainRoot},
		{"directory that does not exist", filepath.Join(mainRoot, "a", "nope"), mainRoot},
		{".git directory without HEAD", filepath.Join(mainRoot, "nested", "x"), mainRoot},
		{".git file not under worktrees", submodule, submodule},
		{".git file whose main has no .git", orphan, orphan},
		{".git file with a relative gitdir", relative, mainRoot},
		{".git file without a gitdir line", garbled, garbled},
		{"no git at all", noGit, noGit},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := codexHookRoot(tc.dir)
			if err != nil {
				t.Fatalf("codexHookRoot(%s): %v", tc.dir, err)
			}
			if got != tc.want {
				t.Errorf("codexHookRoot(%s) = %s, want %s", tc.dir, got, tc.want)
			}
		})
	}
}

func TestCodexHookTrust_RootOfARelativeDir(t *testing.T) {
	tmp := codexHookTrustTempDir(t)
	mainRoot := filepath.Join(tmp, "main")
	makeCodexHookTrustCheckout(t, mainRoot)
	mkdirCodexHookTrust(t, filepath.Join(mainRoot, "a", "b"))
	t.Chdir(filepath.Join(mainRoot, "a"))
	got, err := codexHookRoot("b")
	if err != nil {
		t.Fatalf("codexHookRoot: %v", err)
	}
	if got != mainRoot {
		t.Errorf("codexHookRoot(b) = %s, want %s", got, mainRoot)
	}
}

func TestCodexHookTrust_RootOfALinkedWorktreeMadeByGit(t *testing.T) {
	r := newLegacyLedgerRepo(t)
	sub := filepath.Join(r.worktree, "sub")
	mkdirCodexHookTrust(t, sub)
	for _, dir := range []string{r.worktree, sub} {
		got, err := codexHookRoot(dir)
		if err != nil {
			t.Fatalf("codexHookRoot(%s): %v", dir, err)
		}
		if got != r.mainRoot {
			t.Errorf("codexHookRoot(%s) = %s, want the main checkout %s", dir, got, r.mainRoot)
		}
	}
	// Through a symlink the worktree still keys under the real main
	// checkout: its .git file spells the gitdir with the real path (evidence
	// section 2).
	link := filepath.Join(codexHookTrustTempDir(t), "link")
	symlinkCodexHookTrust(t, filepath.Dir(r.worktree), link)
	roots, err := codexHookRootCandidates(filepath.Join(link, filepath.Base(r.worktree)))
	if err != nil {
		t.Fatalf("codexHookRootCandidates: %v", err)
	}
	if !slices.Equal(roots, []string{r.mainRoot}) {
		t.Errorf("roots through the symlink = %v, want [%s]", roots, r.mainRoot)
	}
}

func TestCodexHookTrust_RootCandidatesOfASymlinkedMainCheckout(t *testing.T) {
	tmp := codexHookTrustTempDir(t)
	realMain := filepath.Join(tmp, "real", "main")
	makeCodexHookTrustCheckout(t, realMain)
	mkdirCodexHookTrust(t, filepath.Join(realMain, "sub"))
	symlinkCodexHookTrust(t, filepath.Join(tmp, "real"), filepath.Join(tmp, "link"))
	linkMain := filepath.Join(tmp, "link", "main")

	cases := []struct {
		name string
		dir  string
		want []string
	}{
		{"through the symlink", filepath.Join(linkMain, "sub"), []string{linkMain, realMain}},
		{"by the real path", realMain, []string{realMain}},
		{"a path that does not exist", filepath.Join(linkMain, "nope"), []string{linkMain}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			roots, err := codexHookRootCandidates(tc.dir)
			if err != nil {
				t.Fatalf("codexHookRootCandidates(%s): %v", tc.dir, err)
			}
			if !slices.Equal(roots, tc.want) {
				t.Errorf("codexHookRootCandidates(%s) = %v, want %v", tc.dir, roots, tc.want)
			}
		})
	}
}

// doctor passes "." (relative), and a shell opened through a symlink keeps
// the symlink's spelling in $PWD. The real root must still come back second:
// a seat started under the real path keys its hooks there (AC9).
func TestCodexHookTrust_RootCandidatesOfARelativeDirInASymlinkedMainCheckout(t *testing.T) {
	tmp := codexHookTrustTempDir(t)
	realMain := filepath.Join(tmp, "real", "main")
	makeCodexHookTrustCheckout(t, realMain)
	mkdirCodexHookTrust(t, filepath.Join(realMain, "sub"))
	symlinkCodexHookTrust(t, filepath.Join(tmp, "real"), filepath.Join(tmp, "link"))
	linkMain := filepath.Join(tmp, "link", "main")

	cases := []struct {
		name string
		cwd  string // the working directory, with $PWD spelled the same way
		dir  string
		want []string
	}{
		{"dot through the symlink", linkMain, ".", []string{linkMain, realMain}},
		{"subdirectory through the symlink", linkMain, "sub", []string{linkMain, realMain}},
		{"path that does not exist through the symlink", linkMain, "nope", []string{linkMain}},
		{"dot by the real path", realMain, ".", []string{realMain}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			chdirCodexHookTrustAs(t, tc.cwd)
			roots, err := codexHookRootCandidates(tc.dir)
			if err != nil {
				t.Fatalf("codexHookRootCandidates(%s): %v", tc.dir, err)
			}
			if !slices.Equal(roots, tc.want) {
				t.Errorf("codexHookRootCandidates(%s) in %s = %v, want %v", tc.dir, tc.cwd, roots, tc.want)
			}
		})
	}
}

func TestCodexHookTrust_ReadStatesOfAMissingConfig(t *testing.T) {
	states, state, err := readCodexHookStates(filepath.Join(t.TempDir(), "no-such", "config.toml"))
	if state != codexHookSourceMissing || err != nil || states != nil {
		t.Errorf("got states=%v state=%v err=%v, want nil, missing, nil", states, state, err)
	}
}

func TestCodexHookTrust_ReadStatesOfAnUnreadableConfig(t *testing.T) {
	dir := t.TempDir()
	asDir := filepath.Join(dir, "config-dir.toml")
	mkdirCodexHookTrust(t, asDir)
	oversized := filepath.Join(dir, "big.toml")
	writeCodexHookTrustFile(t, oversized, strings.Repeat("#", codexConfigMaxBytes+1))

	cases := []struct {
		name, path, want string
	}{
		{"a directory", asDir, "not a regular file"},
		{"larger than 1 MiB", oversized, "larger than 1 MiB"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			states, state, err := readCodexHookStates(tc.path)
			if state != codexHookSourceUnreadable || err == nil || err.Error() != tc.want || states != nil {
				t.Errorf("got states=%v state=%v err=%v, want nil, unreadable, %q", states, state, err, tc.want)
			}
		})
	}
}

func TestCodexHookTrust_ReadStatesOfAnUndecodableConfigCarriesNoText(t *testing.T) {
	const marker = "config-content-marker-7f3a"
	cases := []struct {
		name, content string
	}{
		{"invalid TOML", "[hooks.state.\"/r/.codex/hooks.json:stop:0:0\"]\ntrusted_hash = \"" + marker + "\"\n" + marker + " = = 1\n"},
		{"a table defined twice", codexHookTrustStateTable("/"+marker, "trusted_hash = \"x\"") + codexHookTrustStateTable("/"+marker, "enabled = true")},
		{"trusted_hash of the wrong type", codexHookTrustStateTable("/"+marker, "trusted_hash = 5")},
		{"hooks that is not a table", "hooks = \"" + marker + "\"\n"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "config.toml")
			writeCodexHookTrustFile(t, path, tc.content)
			states, state, err := readCodexHookStates(path)
			if state != codexHookSourceInvalid || states != nil {
				t.Fatalf("got states=%v state=%v err=%v, want nil, invalid", states, state, err)
			}
			var decodeErr *codexConfigDecodeError
			if !errors.As(err, &decodeErr) {
				t.Fatalf("err = %T %v, want a *codexConfigDecodeError", err, err)
			}
			if strings.Contains(err.Error(), marker) {
				t.Errorf("error text %q repeats the config's content", err)
			}
		})
	}
}

func TestCodexHookTrust_ReadStatesKeepsOnlyHookStates(t *testing.T) {
	content := `model = "gpt-x"
sandbox_mode = "workspace-write"

[projects."/r"]
trust_level = "trusted"

[[hooks.PreToolUse]]
matcher = "Bash"
[[hooks.PreToolUse.hooks]]
type = "command"
command = "echo toml"

` + codexHookTrustStateTable("/r/.codex/hooks.json:pre_tool_use:0:0", `trusted_hash = "sha256:aaaa"`) +
		codexHookTrustStateTable("/r/.codex/hooks.json:session_start:0:0", "enabled = false\ntrusted_hash = \"sha256:bbbb\"") +
		codexHookTrustStateTable("/r/.codex/hooks.json:stop:0:0", "enabled = true") +
		codexHookTrustStateTable("/r/.codex/hooks.json:post_tool_use:0:0", `trusted_content_hash = "sha256:cccc"`)
	path := filepath.Join(t.TempDir(), "config.toml")
	writeCodexHookTrustFile(t, path, content)

	states, state, err := readCodexHookStates(path)
	if state != codexHookSourceRead || err != nil {
		t.Fatalf("state=%v err=%v, want read, nil", state, err)
	}
	off, on := false, true
	want := map[string]codexHookStateEntry{
		"/r/.codex/hooks.json:pre_tool_use:0:0":  {TrustedHash: "sha256:aaaa"},
		"/r/.codex/hooks.json:session_start:0:0": {TrustedHash: "sha256:bbbb", Enabled: &off},
		"/r/.codex/hooks.json:stop:0:0":          {Enabled: &on},
		"/r/.codex/hooks.json:post_tool_use:0:0": {},
	}
	if len(states) != len(want) {
		t.Fatalf("states = %v, want %d entries", states, len(want))
	}
	for key, w := range want {
		got, ok := states[key]
		if !ok {
			t.Errorf("no state for %s", key)
			continue
		}
		if got.TrustedHash != w.TrustedHash || (got.Enabled == nil) != (w.Enabled == nil) || (got.Enabled != nil && *got.Enabled != *w.Enabled) {
			t.Errorf("state of %s = %+v, want %+v", key, got, w)
		}
	}

	// A config with no [hooks] at all reads as no states.
	writeCodexHookTrustFile(t, path, "model = \"gpt-x\"\n")
	states, state, err = readCodexHookStates(path)
	if state != codexHookSourceRead || err != nil || len(states) != 0 {
		t.Errorf("no [hooks]: states=%v state=%v err=%v, want none, read, nil", states, state, err)
	}
}

func TestCodexHookTrust_ClassifyUnderOneSpelling(t *testing.T) {
	slot := codexHookSlot{Label: "pre_tool_use", Group: 0, Handler: 0}
	key := codexHookKey("/r", slot)
	off, on := false, true
	cases := []struct {
		name   string
		states map[string]codexHookStateEntry
		want   codexHookTrust
	}{
		{"approved", map[string]codexHookStateEntry{key: {TrustedHash: "sha256:a"}}, codexHookApproved},
		{"approved and enabled", map[string]codexHookStateEntry{key: {TrustedHash: "sha256:a", Enabled: &on}}, codexHookApproved},
		{"no config entries", nil, codexHookNotApproved},
		{"key missing", map[string]codexHookStateEntry{codexHookKey("/other", slot): {TrustedHash: "sha256:a"}}, codexHookNotApproved},
		{"no trusted_hash", map[string]codexHookStateEntry{key: {Enabled: &on}}, codexHookNotApproved},
		{"turned off with a hash", map[string]codexHookStateEntry{key: {TrustedHash: "sha256:a", Enabled: &off}}, codexHookOff},
		{"turned off without a hash", map[string]codexHookStateEntry{key: {Enabled: &off}}, codexHookOff},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := classifyCodexHookSlot(tc.states, []string{"/r"}, slot); got != tc.want {
				t.Errorf("classifyCodexHookSlot = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestCodexHookTrust_ClassifyNeedsEverySpelling(t *testing.T) {
	slot := codexHookSlot{Label: "session_start", Group: 0, Handler: 0}
	roots := []string{"/link/main", "/real/main"}
	linkKey, realKey := codexHookKey(roots[0], slot), codexHookKey(roots[1], slot)
	approved := codexHookStateEntry{TrustedHash: "sha256:a"}
	off := false
	turnedOff := codexHookStateEntry{TrustedHash: "sha256:a", Enabled: &off}
	cases := []struct {
		name   string
		states map[string]codexHookStateEntry
		want   codexHookTrust
	}{
		{"approved only under the real path", map[string]codexHookStateEntry{realKey: approved}, codexHookNotApproved},
		{"approved only under the symlink path", map[string]codexHookStateEntry{linkKey: approved}, codexHookNotApproved},
		{"approved under both", map[string]codexHookStateEntry{linkKey: approved, realKey: approved}, codexHookApproved},
		{"approved under one, turned off under the other", map[string]codexHookStateEntry{linkKey: approved, realKey: turnedOff}, codexHookOff},
		{"turned off under one, missing under the other", map[string]codexHookStateEntry{linkKey: turnedOff}, codexHookNotApproved},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := classifyCodexHookSlot(tc.states, roots, slot); got != tc.want {
				t.Errorf("classifyCodexHookSlot = %v, want %v", got, tc.want)
			}
		})
	}
}

// codexHookTrustFixtureHooks is a hooks.json with four handlers codex asks
// approval for (pre_tool_use 0:0 and 0:1, post_tool_use 0:0, session_start
// 0:0) and a Stop handler with a blank command, which needs none.
const codexHookTrustFixtureHooks = `{
  "hooks": {
    "PreToolUse": [{"matcher": "Bash", "hooks": [
      {"type": "command", "command": "guard.sh"},
      {"type": "command", "command": "other.sh"}
    ]}],
    "PostToolUse": [{"hooks": [{"type": "command", "command": "post.sh"}]}],
    "SessionStart": [{"hooks": [{"type": "command", "command": "start.sh"}]}],
    "Stop": [{"hooks": [{"type": "command", "command": " "}]}]
  }
}`

func TestCodexHookTrustFor_SortsEveryHandlerOfTheRepo(t *testing.T) {
	tmp := codexHookTrustTempDir(t)
	repo := filepath.Join(tmp, "repo")
	makeCodexHookTrustCheckout(t, repo)
	mkdirCodexHookTrust(t, filepath.Join(repo, "sub"))
	hooksPath := filepath.Join(repo, ".codex", "hooks.json")
	writeCodexHookTrustFile(t, hooksPath, codexHookTrustFixtureHooks)
	configPath := filepath.Join(tmp, "codex-home", "config.toml")
	writeCodexHookTrustFile(t, configPath,
		codexHookTrustStateTable(hooksPath+":pre_tool_use:0:0", `trusted_hash = "sha256:a"`)+
			codexHookTrustStateTable(hooksPath+":pre_tool_use:0:1", "trusted_hash = \"sha256:b\"\nenabled = false")+
			codexHookTrustStateTable(hooksPath+":session_start:0:0", "enabled = true"))

	report := codexHookTrustFor(filepath.Join(repo, "sub"), codexSandboxEnv{ConfigPath: configPath})

	if !slices.Equal(report.Roots, []string{repo}) || report.HooksPath != hooksPath || report.ConfigPath != configPath {
		t.Fatalf("roots=%v hooks=%s config=%s, want [%s] %s %s", report.Roots, report.HooksPath, report.ConfigPath, repo, hooksPath, configPath)
	}
	if report.HooksState != codexHookSourceRead || report.HooksErr != nil || report.ConfigState != codexHookSourceRead || report.ConfigErr != nil || !report.Classified() {
		t.Fatalf("hooks=%v/%v config=%v/%v classified=%v, want both read", report.HooksState, report.HooksErr, report.ConfigState, report.ConfigErr, report.Classified())
	}
	if len(report.Slots) != 4 {
		t.Errorf("slots = %v, want 4", report.Slots)
	}
	if got := codexHookTrustTails(report.Approved); !slices.Equal(got, []string{"pre_tool_use:0:0"}) {
		t.Errorf("approved = %v", got)
	}
	if got := codexHookTrustTails(report.NotApproved); !slices.Equal(got, []string{"post_tool_use:0:0", "session_start:0:0"}) {
		t.Errorf("not approved = %v", got)
	}
	if got := codexHookTrustTails(report.Off); !slices.Equal(got, []string{"pre_tool_use:0:1"}) {
		t.Errorf("off = %v", got)
	}
	if len(report.Approved) == 1 && !slices.Equal(report.Approved[0].Keys, []string{hooksPath + ":pre_tool_use:0:0"}) {
		t.Errorf("approved keys = %v", report.Approved[0].Keys)
	}
}

func TestCodexHookTrustFor_LinkedWorktreeReadsTheMainCheckoutsHooks(t *testing.T) {
	r := newLegacyLedgerRepo(t)
	mainHooks := filepath.Join(r.mainRoot, ".codex", "hooks.json")
	writeCodexHookTrustFile(t, mainHooks, `{"hooks": {"PreToolUse": [{"hooks": [{"type": "command", "command": "guard.sh"}]}]}}`)
	// The worktree's own hooks.json is not the one codex reads.
	writeCodexHookTrustFile(t, filepath.Join(r.worktree, ".codex", "hooks.json"), codexHookTrustFixtureHooks)
	configPath := filepath.Join(t.TempDir(), "config.toml")
	writeCodexHookTrustFile(t, configPath, codexHookTrustStateTable(mainHooks+":pre_tool_use:0:0", `trusted_hash = "sha256:a"`))

	report := codexHookTrustFor(r.worktree, codexSandboxEnv{ConfigPath: configPath})

	if !slices.Equal(report.Roots, []string{r.mainRoot}) || report.HooksPath != mainHooks {
		t.Fatalf("roots=%v hooks=%s, want [%s] %s", report.Roots, report.HooksPath, r.mainRoot, mainHooks)
	}
	if got := codexHookTrustTails(report.Approved); !slices.Equal(got, []string{"pre_tool_use:0:0"}) || len(report.NotApproved) != 0 || len(report.Off) != 0 {
		t.Errorf("approved=%v not approved=%v off=%v, want only pre_tool_use:0:0 approved", got, report.NotApproved, report.Off)
	}
}

func TestCodexHookTrustFor_SymlinkedMainCheckoutNeedsBothSpellings(t *testing.T) {
	tmp := codexHookTrustTempDir(t)
	realRepo := filepath.Join(tmp, "real", "repo")
	makeCodexHookTrustCheckout(t, realRepo)
	writeCodexHookTrustFile(t, filepath.Join(realRepo, ".codex", "hooks.json"), `{"hooks": {"Stop": [{"hooks": [{"type": "command", "command": "stop.sh"}]}]}}`)
	symlinkCodexHookTrust(t, filepath.Join(tmp, "real"), filepath.Join(tmp, "link"))
	linkRepo := filepath.Join(tmp, "link", "repo")
	linkKey := filepath.Join(linkRepo, ".codex", "hooks.json") + ":stop:0:0"
	realKey := filepath.Join(realRepo, ".codex", "hooks.json") + ":stop:0:0"
	configPath := filepath.Join(tmp, "config.toml")
	approval := `trusted_hash = "sha256:a"`

	cases := []struct {
		name     string
		config   string
		approved bool
	}{
		{"approved under the real path only", codexHookTrustStateTable(realKey, approval), false},
		{"approved under the symlink path only", codexHookTrustStateTable(linkKey, approval), false},
		{"approved under both", codexHookTrustStateTable(linkKey, approval) + codexHookTrustStateTable(realKey, approval), true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			writeCodexHookTrustFile(t, configPath, tc.config)
			report := codexHookTrustFor(linkRepo, codexSandboxEnv{ConfigPath: configPath})
			if !slices.Equal(report.Roots, []string{linkRepo, realRepo}) {
				t.Fatalf("roots = %v, want [%s %s]", report.Roots, linkRepo, realRepo)
			}
			if got := len(report.Approved) == 1 && len(report.NotApproved) == 0; got != tc.approved {
				t.Errorf("approved=%v not approved=%v, want approved %v", report.Approved, report.NotApproved, tc.approved)
			}
			if !tc.approved && (len(report.NotApproved) != 1 || !slices.Equal(report.NotApproved[0].Keys, []string{linkKey, realKey})) {
				t.Errorf("not approved = %v, want stop:0:0 with keys [%s %s]", report.NotApproved, linkKey, realKey)
			}
		})
	}
}

func TestCodexHookTrustFor_ReportsEachInputApart(t *testing.T) {
	const marker = "config-content-marker-91c2"
	cases := []struct {
		name            string
		hooks           string // "" leaves hooks.json missing; "<dir>" makes it a directory
		config          string // "" leaves config.toml missing; "<dir>" makes it a directory
		hooksState      codexHookSourceState
		configState     codexHookSourceState
		classified      bool
		notApprovedTail []string
	}{
		{"hooks.json missing", "", `model = "x"`, codexHookSourceMissing, codexHookSourceRead, false, nil},
		{"hooks.json not JSON", "{", `model = "x"`, codexHookSourceInvalid, codexHookSourceRead, false, nil},
		{"hooks.json a directory", "<dir>", `model = "x"`, codexHookSourceUnreadable, codexHookSourceRead, false, nil},
		{"config missing", codexHookTrustFixtureHooks, "", codexHookSourceRead, codexHookSourceMissing, true,
			[]string{"post_tool_use:0:0", "pre_tool_use:0:0", "pre_tool_use:0:1", "session_start:0:0"}},
		{"config a directory", codexHookTrustFixtureHooks, "<dir>", codexHookSourceRead, codexHookSourceUnreadable, false, nil},
		{"config not decodable", codexHookTrustFixtureHooks, marker + " = = 1", codexHookSourceRead, codexHookSourceInvalid, false, nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			tmp := codexHookTrustTempDir(t)
			repo := filepath.Join(tmp, "repo")
			makeCodexHookTrustCheckout(t, repo)
			hooksPath := filepath.Join(repo, ".codex", "hooks.json")
			configPath := filepath.Join(tmp, "codex-home", "config.toml")
			for path, content := range map[string]string{hooksPath: tc.hooks, configPath: tc.config} {
				switch content {
				case "":
				case "<dir>":
					mkdirCodexHookTrust(t, path)
				default:
					writeCodexHookTrustFile(t, path, content)
				}
			}

			report := codexHookTrustFor(repo, codexSandboxEnv{ConfigPath: configPath})

			if report.HooksPath != hooksPath || report.HooksState != tc.hooksState || (report.HooksErr != nil) != (tc.hooksState == codexHookSourceUnreadable || tc.hooksState == codexHookSourceInvalid) {
				t.Errorf("hooks: %s %v %v, want %s %v", report.HooksPath, report.HooksState, report.HooksErr, hooksPath, tc.hooksState)
			}
			if report.ConfigState != tc.configState || (report.ConfigErr != nil) != (tc.configState == codexHookSourceUnreadable || tc.configState == codexHookSourceInvalid) {
				t.Errorf("config: %v %v, want %v", report.ConfigState, report.ConfigErr, tc.configState)
			}
			if report.ConfigErr != nil && strings.Contains(report.ConfigErr.Error(), marker) {
				t.Errorf("config error %q repeats the config's content", report.ConfigErr)
			}
			if report.Classified() != tc.classified {
				t.Errorf("classified = %v, want %v", report.Classified(), tc.classified)
			}
			if got := codexHookTrustTails(report.NotApproved); !slices.Equal(got, tc.notApprovedTail) {
				t.Errorf("not approved = %v, want %v", got, tc.notApprovedTail)
			}
			if !tc.classified && (len(report.Approved) != 0 || len(report.Off) != 0) {
				t.Errorf("approved=%v off=%v, want none when not classified", report.Approved, report.Off)
			}
		})
	}
}
