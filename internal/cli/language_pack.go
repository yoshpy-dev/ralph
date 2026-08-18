package cli

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/yoshpy-dev/ralph/internal/scaffold"
)

// packRenderResult bundles the output of renderPackInto so callers can use
// it to update a manifest and print summaries without re-implementing the
// same book-keeping in both init.go and pack.go.
type packRenderResult struct {
	// result is the raw RenderResult from scaffold.RenderFS (pack payload files
	// only, not the rule.md control file).
	result *scaffold.RenderResult
	// hashes maps manifest keys (namespaced with packRelDir(lang)) to SHA256
	// hashes for all written files (payload + rule).
	hashes map[string]string
}

// renderPackInto renders a language pack into its canonical subdirectory
// (packs/languages/<lang>/) inside targetDir and handles the rule.md control
// file (→ .claude/rules/ralph/<lang>.md).
//
// The returned packRenderResult contains all namespaced hashes ready to be
// merged into the project manifest.
//
// This helper is shared by executeInit (init.go) and addPack (pack.go) so the
// two code paths cannot diverge.
func renderPackInto(targetDir, lang string, force bool) (*packRenderResult, error) {
	packFS, err := scaffold.PackFS(lang)
	if err != nil {
		return nil, fmt.Errorf("language pack %q not found: %w", lang, err)
	}

	packDir := filepath.Join(targetDir, packRelDir(lang))
	packResult, packHashes, err := scaffold.RenderFS(packFS, scaffold.RenderOptions{
		TargetDir: packDir,
		Overwrite: force,
		SkipPaths: packRenderSkipPaths,
	})
	if err != nil {
		return nil, fmt.Errorf("rendering pack %s: %w", lang, err)
	}

	out := &packRenderResult{
		result: packResult,
		hashes: make(map[string]string),
	}

	// Namespace hashes with the pack subdirectory prefix.
	packPrefix := packRelDir(lang)
	for k, v := range packHashes {
		out.hashes[filepath.Join(packPrefix, k)] = v
	}

	// Handle the rule.md control file: it renders to .claude/rules/ralph/<lang>.md
	// instead of packs/languages/<lang>/rule.md.
	ruleContent, ok, err := packRuleContent(packFS)
	if err != nil {
		return nil, fmt.Errorf("reading pack %s rule: %w", lang, err)
	}
	if ok {
		rulePath := packRuleRelPath(lang)
		ruleResult, ruleHash, err := renderMappedFile(targetDir, rulePath, ruleContent, force)
		if err != nil {
			return nil, fmt.Errorf("rendering pack %s rule: %w", lang, err)
		}
		out.hashes[rulePath] = ruleHash
		mergeRenderResult(packResult, ruleResult)
	}

	return out, nil
}

const packRuleSourcePath = "rule.md"

var packRenderSkipPaths = map[string]bool{
	packRuleSourcePath: true,
}

func packRelDir(pack string) string {
	return filepath.Join("packs", "languages", pack)
}

// packNamespacePrefix is the root namespace for all language pack entries in
// the manifest. Keys under this prefix are pack-scoped.
const packNamespacePrefix = "packs/languages/"

// packPrefixFor returns the namespace prefix used for a specific pack's files
// in the project manifest (e.g. "packs/languages/golang/").
func packPrefixFor(pack string) string {
	return packNamespacePrefix + pack + "/"
}

// packRuleRelPath returns the manifest key / render path for a pack's
// rule.md control file. It uses path.Join, not filepath.Join, because the
// result feeds manifest keys — which init.go's ownerForScaffoldPath docs as
// "always fs.FS slash paths" — not a raw OS filesystem call; filepath.Join
// would emit "\"-separated keys on Windows and silently diverge from
// ralph init's slash-keyed manifest entries for the same logical path.
func packRuleRelPath(pack string) string {
	return path.Join(".claude", "rules", "ralph", pack+".md")
}

func packRuleContent(src fs.FS) ([]byte, bool, error) {
	content, err := fs.ReadFile(src, packRuleSourcePath)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	return content, true, nil
}

func renderMappedFile(targetDir, relPath string, content []byte, overwrite bool) (*scaffold.RenderResult, string, error) {
	result := &scaffold.RenderResult{}
	hash := scaffold.HashBytes(content)
	target := filepath.Join(targetDir, relPath)
	absTargetDir, err := filepath.Abs(targetDir)
	if err != nil {
		return nil, "", fmt.Errorf("resolving target dir: %w", err)
	}
	absTarget, err := filepath.Abs(target)
	if err != nil {
		return nil, "", fmt.Errorf("resolving path %s: %w", relPath, err)
	}
	if !isInsideDir(absTargetDir, absTarget) {
		return nil, "", fmt.Errorf("template path %q escapes target directory", relPath)
	}

	// Lstat (not Stat) is used deliberately, mirroring scaffold.RenderFS: Stat
	// follows symlinks, so a *dangling* symlink at relPath would stat as
	// absent, get classified as a create below, and os.WriteFile would then
	// write straight through the link to wherever it resolves -- outside
	// targetDir, defeating the boundary check above. Lstat inspects the
	// directory entry itself, so any existing entry (regular file, valid
	// symlink, or dangling symlink) counts as "exists" and is skipped here in
	// non-force mode.
	if _, statErr := os.Lstat(target); statErr == nil {
		if !overwrite {
			result.Skipped = append(result.Skipped, relPath)
			return result, hash, nil
		}
		result.Overwritten = append(result.Overwritten, relPath)
	} else {
		result.Created = append(result.Created, relPath)
	}

	if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
		return nil, "", fmt.Errorf("creating parent dir for %s: %w", relPath, err)
	}
	if err := os.WriteFile(target, content, scaffold.FilePerm(relPath)); err != nil {
		return nil, "", fmt.Errorf("writing %s: %w", relPath, err)
	}
	return result, hash, nil
}

func isInsideDir(absDir, absPath string) bool {
	return absPath == absDir || strings.HasPrefix(absPath, absDir+string(filepath.Separator))
}

func mergeRenderResult(dst, src *scaffold.RenderResult) {
	if dst == nil || src == nil {
		return
	}
	dst.Created = append(dst.Created, src.Created...)
	dst.Overwritten = append(dst.Overwritten, src.Overwritten...)
	dst.Skipped = append(dst.Skipped, src.Skipped...)
}
