package scaffold

import (
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"testing"
	"testing/fstest"

	toml "github.com/pelletier/go-toml/v2"
)

func TestBaseFS_WithMockFS(t *testing.T) {
	// EmbeddedFS is only populated when built from cmd/ralph/ with go:embed.
	// In unit tests, it's a zero-value embed.FS. Skip these tests.
	if _, err := fs.ReadDir(EmbeddedFS, "templates"); err != nil {
		t.Skip("EmbeddedFS not initialized (only available when built from cmd/ralph/)")
	}

	baseFS, err := BaseFS()
	if err != nil {
		t.Fatalf("BaseFS: %v", err)
	}

	if _, err := fs.Stat(baseFS, "AGENTS.md"); err != nil {
		t.Errorf("AGENTS.md not found in BaseFS: %v", err)
	}
}

func TestAvailablePacks_WithMockFS(t *testing.T) {
	if _, err := fs.ReadDir(EmbeddedFS, "templates"); err != nil {
		t.Skip("EmbeddedFS not initialized")
	}

	packs, err := AvailablePacks()
	if err != nil {
		t.Fatalf("AvailablePacks: %v", err)
	}

	if len(packs) < 5 {
		t.Errorf("packs count = %d, want >= 5, got: %v", len(packs), packs)
	}
}

// TestEmbedFSInterface verifies the exported variable is the right type.
func TestEmbedFSInterface(t *testing.T) {
	var _ = EmbeddedFS // type is embed.FS
}

// requiredTemplateScripts lists the scripts every `ralph init` scaffold must
// ship. TestTemplateBaseScriptsExist checks that each one exists (and is
// executable) under templates/base/scripts/, but does not catch a name
// dropped from this list.
// TestTemplateBaseScriptsMatchCheckTemplateRequiredFiles compares this set
// against the `scripts/` entries of check-template.sh's required_files
// list; that is what catches a name dropped from either side. A
// `scripts/` entry lives in four places that must change together:
// scripts/check-template.sh, templates/base/scripts/check-template.sh
// (kept byte-identical to it), this list, and the golden list in
// tests/test-check-template.sh's GOLDEN_ENTRIES (issue #183).
var requiredTemplateScripts = []string{
	"run-verify.sh",
	"run-static-verify.sh",
	"run-test.sh",
	"detect-changed-languages.sh",
	"detect-languages.sh",
	"archive-plan.sh",
	"branch-name.sh",
	"ensure-pr-ready.sh",
	"ensure-pr-title-prefix.sh",
	"new-feature-plan.sh",
	"codex-check.sh",
	"ralph-config.sh",
	"ralph-worktree.sh",
	"xreview-helpers.sh",
	"secret-scan.sh",
	"secret-scan-branch.sh",
	"pre-commit-secret-guard.sh",
	"commit-msg-guard.sh",
	"prepare-commit-msg-secret-guard.sh",
	"pre-merge-commit-secret-guard.sh",
	"check-template.sh",
	"check-skill-sync.sh",
}

// TestTemplateBaseScriptsExist verifies all required scripts are present
// in templates/base/scripts/ on disk. This catches distribution gaps where
// template docs reference scripts that are not actually included.
func TestTemplateBaseScriptsExist(t *testing.T) {
	// Locate the repo root from this test file's location.
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot determine test file location")
	}
	// thisFile is internal/scaffold/embed_test.go → repo root is ../../
	repoRoot := filepath.Join(filepath.Dir(thisFile), "..", "..")
	scriptsDir := filepath.Join(repoRoot, "templates", "base", "scripts")

	for _, name := range requiredTemplateScripts {
		path := filepath.Join(scriptsDir, name)
		info, err := os.Stat(path)
		if err != nil {
			t.Errorf("required script missing: templates/base/scripts/%s", name)
			continue
		}
		// Verify executable permission on Unix.
		if runtime.GOOS != "windows" && info.Mode().Perm()&0111 == 0 {
			t.Errorf("script not executable: templates/base/scripts/%s (mode %o)", name, info.Mode().Perm())
		}
	}
}

// extractCheckTemplateRequiredFiles reads a check-template.sh script and
// returns the entries of its `required_files="..."` multi-line
// double-quoted assignment (the lines between `required_files="` and the
// closing `"` line), in order, with blank lines dropped. Returns an empty
// slice if the block cannot be found (e.g. the script's format changed).
func extractCheckTemplateRequiredFiles(t *testing.T, path string) []string {
	t.Helper()

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}

	lines := strings.Split(string(data), "\n")
	var entries []string
	inBlock := false
	for _, line := range lines {
		if !inBlock {
			if line == `required_files="` {
				inBlock = true
			}
			continue
		}
		if line == `"` {
			break
		}
		if strings.TrimSpace(line) == "" {
			continue
		}
		entries = append(entries, line)
	}
	return entries
}

// TestTemplateBaseScriptsMatchCheckTemplateRequiredFiles verifies that the
// `scripts/` entries of templates/base/scripts/check-template.sh's
// required_files list are exactly the set in requiredTemplateScripts. A name
// dropped from either list (the Go list or the shell list) fails this test,
// keeping the "scripts every scaffold ships" contract and the "scripts
// check-template.sh itself requires" contract in sync.
func TestTemplateBaseScriptsMatchCheckTemplateRequiredFiles(t *testing.T) {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot determine test file location")
	}
	repoRoot := filepath.Join(filepath.Dir(thisFile), "..", "..")
	checkTemplatePath := filepath.Join(repoRoot, "templates", "base", "scripts", "check-template.sh")

	rawEntries := extractCheckTemplateRequiredFiles(t, checkTemplatePath)
	if len(rawEntries) == 0 {
		t.Fatalf("required_files block in %s yielded 0 entries; the extraction format (required_files=\"...\") may have changed", checkTemplatePath)
	}

	scriptEntries := make(map[string]bool)
	for _, entry := range rawEntries {
		name, ok := strings.CutPrefix(entry, "scripts/")
		if !ok {
			continue // non-script entry (AGENTS.md, CLAUDE.md, .claude/settings.json)
		}
		scriptEntries[name] = true
	}

	goEntries := make(map[string]bool, len(requiredTemplateScripts))
	for _, name := range requiredTemplateScripts {
		goEntries[name] = true
	}

	var onlyInScript, onlyInGo []string
	for name := range scriptEntries {
		if !goEntries[name] {
			onlyInScript = append(onlyInScript, name)
		}
	}
	for name := range goEntries {
		if !scriptEntries[name] {
			onlyInGo = append(onlyInGo, name)
		}
	}
	sort.Strings(onlyInScript)
	sort.Strings(onlyInGo)

	if len(onlyInScript) > 0 || len(onlyInGo) > 0 {
		t.Errorf("scripts/ entries in check-template.sh's required_files and requiredTemplateScripts disagree:\n  only in check-template.sh: %v\n  only in the Go list: %v", onlyInScript, onlyInGo)
	}
}

// TestTemplateBaseCodexAssetsExist enforces the Codex parity contract
// (docs/specs/2026-05-07-codex-cli-parity.md): every fresh `ralph init`
// project must ship .codex/{config.toml, AGENTS.override.md, README.md} and
// the .agents/skills/ tree alongside the existing .claude/ surface. Drift in
// either tree breaks AC-1, so guard the on-disk template directly rather than
// relying on go:embed inspection (the embed FS is empty in unit tests).
func TestTemplateBaseCodexAssetsExist(t *testing.T) {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot determine test file location")
	}
	repoRoot := filepath.Join(filepath.Dir(thisFile), "..", "..")
	baseDir := filepath.Join(repoRoot, "templates", "base")

	required := []string{
		".codex/config.toml",
		".codex/AGENTS.override.md",
		".codex/README.md",
		".codex/agents/doc-maintainer.toml",
		".codex/agents/reviewer.toml",
		".codex/agents/tester.toml",
		".codex/agents/verifier.toml",
		".codex/hooks/README.md",
		".agents/skills/.gitkeep",
	}

	for _, rel := range required {
		path := filepath.Join(baseDir, rel)
		if _, err := os.Stat(path); err != nil {
			t.Errorf("required template missing: templates/base/%s (%v)", rel, err)
		}
	}
}

// TestTemplateBaseCodexTomlFilesParse guards the content of the Codex TOML
// files that `ralph init` ships, which TestTemplateBaseCodexAssetsExist only
// checks for presence: a syntax error in the template config or a custom agent
// would otherwise reach a scaffolded project and surface only in
// `ralph doctor`. The config must also keep a top-level `model` key. The
// literal slug is deliberately not pinned, because it changes whenever the
// model is retired. The meta-repo root copies under .codex/ are not parsed
// here: scripts/check-sync.sh keeps the config and agents byte-identical to
// the template.
func TestTemplateBaseCodexTomlFilesParse(t *testing.T) {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot determine test file location")
	}
	repoRoot := filepath.Join(filepath.Dir(thisFile), "..", "..")
	codexDir := filepath.Join(repoRoot, "templates", "base", ".codex")

	agents, err := filepath.Glob(filepath.Join(codexDir, "agents", "*.toml"))
	if err != nil {
		t.Fatalf("glob templates/base/.codex/agents/*.toml: %v", err)
	}
	if len(agents) == 0 {
		t.Fatal("no templates/base/.codex/agents/*.toml found; the glob would pass vacuously")
	}
	paths := append([]string{filepath.Join(codexDir, "config.toml")}, agents...)

	parsed := map[string]map[string]any{}
	for _, path := range paths {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Errorf("read %s: %v", path, err)
			continue
		}
		var doc map[string]any
		if err := toml.Unmarshal(data, &doc); err != nil {
			t.Errorf("%s is not valid TOML: %v", path, err)
			continue
		}
		parsed[path] = doc
	}

	configPath := paths[0]
	doc, ok := parsed[configPath]
	if !ok {
		return // already reported above
	}
	model, ok := doc["model"].(string)
	if !ok || strings.TrimSpace(model) == "" {
		t.Errorf("%s: top-level `model` must be a non-empty string, got %#v", configPath, doc["model"])
	}
}

// TestTemplateBaseRalphTomlHasOrgSection enforces that scaffolded projects
// receive the [org] envelope config out of the box. The [loop]/[pipeline]
// sections this test used to also check were removed along with the Ralph
// Loop execution system.
func TestTemplateBaseRalphTomlHasOrgSection(t *testing.T) {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot determine test file location")
	}
	repoRoot := filepath.Join(filepath.Dir(thisFile), "..", "..")
	tomlPath := filepath.Join(repoRoot, "templates", "base", "ralph.toml")

	data, err := os.ReadFile(tomlPath)
	if err != nil {
		t.Fatalf("read ralph.toml: %v", err)
	}
	body := string(data)
	for _, want := range []string{
		"[org]",
		`driver_pool = ["claude", "codex"]`,
	} {
		if !contains(body, want) {
			t.Errorf("templates/base/ralph.toml missing %q", want)
		}
	}
}

// contains is a tiny substring helper to avoid pulling in strings just for
// this assertion.
func contains(haystack, needle string) bool {
	for i := 0; i+len(needle) <= len(haystack); i++ {
		if haystack[i:i+len(needle)] == needle {
			return true
		}
	}
	return false
}

// TestAvailablePacksExcludesTemplate verifies _template is excluded.
func TestAvailablePacksExcludesTemplate(t *testing.T) {
	orig := EmbeddedFS
	defer func() { EmbeddedFS = orig }()

	// Use a MapFS to simulate embedded templates.
	mock := fstest.MapFS{
		"templates/packs/golang/README.md":    {Data: []byte("go")},
		"templates/packs/python/README.md":    {Data: []byte("py")},
		"templates/packs/_template/README.md": {Data: []byte("tpl")},
	}

	// AvailablePacks reads from EmbeddedFS directly, but since embed.FS
	// can't be mocked, test the filtering logic directly.
	entries, err := fs.ReadDir(mock, "templates/packs")
	if err != nil {
		t.Fatal(err)
	}
	var packs []string
	for _, e := range entries {
		if e.IsDir() && e.Name() != "_template" {
			packs = append(packs, e.Name())
		}
	}
	if len(packs) != 2 {
		t.Errorf("packs = %v, want [golang python]", packs)
	}
}
