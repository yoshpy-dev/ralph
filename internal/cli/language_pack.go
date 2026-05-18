package cli

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/yoshpy-dev/ralph/internal/scaffold"
)

const packRuleSourcePath = "rule.md"

var packRenderSkipPaths = map[string]bool{
	packRuleSourcePath: true,
}

func packRelDir(pack string) string {
	return filepath.Join("packs", "languages", pack)
}

func packRuleRelPath(pack string) string {
	return filepath.Join(".claude", "rules", pack+".md")
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

	if _, statErr := os.Stat(target); statErr == nil {
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
