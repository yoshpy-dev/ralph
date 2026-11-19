package cli

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"

	"github.com/spf13/cobra"

	"github.com/yoshpy-dev/ralph/internal/scaffold"
	"github.com/yoshpy-dev/ralph/internal/upgrade"
)

func newPackRemoveCmd() *cobra.Command {
	var dryRun bool
	cmd := &cobra.Command{
		Use:   "remove <language>",
		Short: "Remove a language pack from the project",
		Long: `Removes a language pack that ralph init or ralph pack add installed.
ralph deletes the pack's files under packs/languages/<language>/ and its
rule .claude/rules/ralph/<language>.md when their content is still what
ralph wrote, removes the directories that become empty, and drops the pack
and every one of its entries from .ralph/manifest.toml, so ralph upgrade no
longer installs it.

A file you changed, a file you ejected with ralph eject, and a path that is
not a regular file (a symlink, a directory) are kept and named with the
reason. ralph no longer tracks them: delete them yourself if you do not
need them. ralph pack add <language> would overwrite them.

A project with the legacy scaffold layout must run ralph upgrade first.
--dry-run prints what would be deleted and kept and changes nothing. When a
removal stops partway, running the same command again finishes it.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return removePack(".", args[0], dryRun, cmd.OutOrStdout())
		},
	}
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "print what would be deleted and kept without changing anything")
	return cmd
}

// writePackRemoveManifest writes the manifest at the end of removePack.
// Tests replace it to make that last step fail.
var writePackRemoveManifest = func(m *scaffold.Manifest, path string) error {
	return m.Write(path)
}

// packRemoveKept is a file of the pack that removePack leaves on disk, with
// the reason it is kept.
type packRemoveKept struct {
	path   string
	reason string
}

// packRemovePlan is the outcome of classifying every manifest entry of a
// pack before anything is written. keys holds every entry (all three
// groups); deletes holds cleaned slash paths for upgrade.ApplyOps.
type packRemovePlan struct {
	keys    []string
	deletes []string
	kept    []packRemoveKept
	missing []string
}

// removePack uninstalls the language pack lang from the v2 project rooted
// at targetDir. It deletes the pack's files whose content still matches the
// hash the manifest recorded for an owner=core entry, keeps every other file
// (changed, ejected, not owned by core, or not a regular file), drops all of
// the pack's manifest entries, and removes lang from Meta.Packs. The decision
// reads only the manifest and the disk, never the embedded templates, so a
// pack the current binary no longer ships can be removed too.
//
// Files are deleted before the manifest is written (the same commit barrier
// as ralph upgrade): when the manifest write fails, the manifest still lists
// the pack, and running removePack again treats the deleted files as already
// missing and finishes the job. With dryRun, removePack prints the plan and
// writes nothing.
func removePack(targetDir, lang string, dryRun bool, out io.Writer) error {
	if err := validatePackRemoveName(lang); err != nil {
		return err
	}
	absDir, err := filepath.Abs(targetDir)
	if err != nil {
		return err
	}

	manifestPath := filepath.Join(absDir, ".ralph", "manifest.toml")
	manifest, err := scaffold.ReadManifest(manifestPath)
	if err != nil {
		return fmt.Errorf("reading manifest: %w", err)
	}
	if manifest.Meta.Layout != scaffold.LayoutV2 {
		return errLegacyLayoutFailClosed
	}
	if !slices.Contains(manifest.Meta.Packs, lang) {
		return packNotInstalledError(lang, manifest.Meta.Packs)
	}

	plan, err := planPackRemove(absDir, lang, manifest)
	if err != nil {
		return err
	}
	if dryRun {
		printPackRemoveDryRun(out, lang, plan)
		return nil
	}

	if err := applyPackRemoveDeletes(absDir, plan.deletes); err != nil {
		return fmt.Errorf("removing pack %s: %w (.ralph/manifest.toml was not changed; run ralph pack remove %s again after fixing this)", lang, err, lang)
	}
	leftReason := removeEmptyPackDirs(absDir, lang, plan.kept, out)

	for _, key := range plan.keys {
		delete(manifest.Files, key)
	}
	manifest.Meta.Packs = slices.DeleteFunc(manifest.Meta.Packs, func(p string) bool { return p == lang })
	if err := writePackRemoveManifest(manifest, manifestPath); err != nil {
		return fmt.Errorf("the files of pack %s were removed, but writing .ralph/manifest.toml failed: %w; run ralph pack remove %s again to finish", lang, err, lang)
	}

	printPackRemoveSummary(out, lang, plan, leftReason)
	return nil
}

// validatePackRemoveName refuses a name that is not one path segment.
// packPrefixFor joins the name into a path as is, and the only other check
// on the name is that Meta.Packs lists it, which a tampered manifest can
// make true for ".." or "a/b".
func validatePackRemoveName(lang string) error {
	if lang == "" || lang == "." || lang == ".." || strings.ContainsAny(lang, `/\`) {
		return fmt.Errorf("invalid language pack name %q: a pack name is one path segment, not empty, \".\", or \"..\"", lang)
	}
	return nil
}

func packNotInstalledError(lang string, installed []string) error {
	list := "none"
	if len(installed) > 0 {
		list = strings.Join(installed, ", ")
	}
	return fmt.Errorf("pack %q is not installed in this project (installed packs: %s)", lang, list)
}

// isPackRemoveKey reports whether the slash path p is one of the places a
// pack's files are recorded: under packs/languages/<lang>/, the rule
// .claude/rules/ralph/<lang>.md, or the rule's pre-v2 place
// .claude/rules/<lang>.md that a migration may have kept.
func isPackRemoveKey(p, lang string) bool {
	return strings.HasPrefix(p, packPrefixFor(lang)) || p == packRuleRelPath(lang) || p == legacyPackRuleRelPath(lang)
}

// packRemoveKeys returns the manifest keys of the pack lang, sorted.
func packRemoveKeys(m *scaffold.Manifest, lang string) []string {
	var keys []string
	for key := range m.Files {
		if isPackRemoveKey(filepath.ToSlash(key), lang) {
			keys = append(keys, key)
		}
	}
	sort.Strings(keys)
	return keys
}

// planPackRemove classifies every manifest entry of the pack. Any path that
// fails validation stops the command before anything is written, so a dry
// run never promises a deletion the real run would refuse.
func planPackRemove(absDir, lang string, m *scaffold.Manifest) (packRemovePlan, error) {
	plan := packRemovePlan{keys: packRemoveKeys(m, lang)}
	for _, key := range plan.keys {
		rel, err := validatePackRemovePath(absDir, key, lang)
		if err != nil {
			return packRemovePlan{}, err
		}
		fi, err := os.Lstat(filepath.Join(absDir, filepath.FromSlash(rel)))
		if errors.Is(err, fs.ErrNotExist) {
			plan.missing = append(plan.missing, key)
			continue
		}
		if err != nil {
			return packRemovePlan{}, fmt.Errorf("%s: %w", key, err)
		}
		reason, err := packRemoveKeepReason(absDir, rel, fi, m.Files[key])
		if err != nil {
			return packRemovePlan{}, fmt.Errorf("%s: %w", key, err)
		}
		if reason != "" {
			plan.kept = append(plan.kept, packRemoveKept{path: key, reason: reason})
			continue
		}
		plan.deletes = append(plan.deletes, rel)
	}
	return plan, nil
}

// validatePackRemovePath checks a manifest key the same way upgrade.ApplyOps
// checks an op's path: a local relative path whose existing parent
// directories are all real directories, never a symlink. It also refuses a
// key whose cleaned form leaves the pack's places (such as
// "packs/languages/<lang>/../other"). It returns the cleaned slash path.
func validatePackRemovePath(absDir, key, lang string) (string, error) {
	clean, err := scaffold.CleanLocalRelPath(key)
	if err != nil {
		return "", fmt.Errorf("manifest path %q: %w", key, err)
	}
	rel := filepath.ToSlash(clean)
	if !isPackRemoveKey(rel, lang) {
		return "", fmt.Errorf("manifest path %q resolves to %s, outside the files of pack %s", key, rel, lang)
	}
	if err := upgrade.ValidateRealParentChain(absDir, rel); err != nil {
		return "", fmt.Errorf("%s: %w", key, err)
	}
	return rel, nil
}

// packRemoveKeepReason returns why the existing path rel is kept, or "" when
// it may be deleted: a regular file of an owner=core entry whose content
// hash equals the recorded one (DiskHash, else Hash, as ralph upgrade reads
// it).
func packRemoveKeepReason(absDir, rel string, fi fs.FileInfo, entry scaffold.ManifestFile) (string, error) {
	if !fi.Mode().IsRegular() {
		return packRemoveNonRegularReason(fi.Mode()), nil
	}
	switch entry.Owner {
	case scaffold.OwnerCore:
	case scaffold.OwnerFork:
		return "ejected (owner=fork)", nil
	case "":
		return "no recorded owner (only owner=core files are deleted)", nil
	default:
		return fmt.Sprintf("owner=%s (only owner=core files are deleted)", entry.Owner), nil
	}
	content, err := os.ReadFile(filepath.Join(absDir, filepath.FromSlash(rel)))
	if err != nil {
		return "", err
	}
	recorded := entry.DiskHash
	if recorded == "" {
		recorded = entry.Hash
	}
	if scaffold.HashBytes(content) != recorded {
		return "modified since ralph wrote it", nil
	}
	return "", nil
}

func packRemoveNonRegularReason(mode fs.FileMode) string {
	switch {
	case mode&fs.ModeSymlink != 0:
		return "not a regular file (symlink)"
	case mode.IsDir():
		return "not a regular file (directory)"
	default:
		return fmt.Sprintf("not a regular file (mode %s)", mode)
	}
}

// applyPackRemoveDeletes deletes the files through upgrade.ApplyOps, which
// checks every path, parent chain, and leaf before it deletes the first one.
func applyPackRemoveDeletes(absDir string, deletes []string) error {
	ops := make([]upgrade.FileOp, 0, len(deletes))
	for _, rel := range deletes {
		ops = append(ops, upgrade.FileOp{Kind: upgrade.OpDelete, Path: rel})
	}
	return upgrade.ApplyOps(absDir, upgrade.ReplacePlan{Ops: ops})
}

// removeEmptyPackDirs removes the directories under packs/languages/<lang>/
// that hold nothing anymore, deepest first, then packs/languages/ and packs/
// when each is a real directory that is empty now. filepath.WalkDir does not
// follow symlinks, and a symlinked packs/languages/<lang> is not walked at
// all. A kept path that is a directory is left alone, empty or not. It
// returns why packs/languages/<lang> is still there, or "" when it is gone.
// A directory it fails to remove is reported on out, not as an error: the
// pack's files are already deleted by then.
func removeEmptyPackDirs(absDir, lang string, kept []packRemoveKept, out io.Writer) string {
	keptPaths := make(map[string]bool, len(kept))
	for _, k := range kept {
		keptPaths[filepath.Join(absDir, filepath.FromSlash(k.path))] = true
	}
	packDir := filepath.Join(absDir, packRelDir(lang))
	if isRealDir(packDir) {
		var dirs []string
		_ = filepath.WalkDir(packDir, func(p string, d fs.DirEntry, err error) error {
			if err != nil || !d.IsDir() {
				return nil
			}
			if keptPaths[p] {
				return fs.SkipDir
			}
			dirs = append(dirs, p)
			return nil
		})
		for i := len(dirs) - 1; i >= 0; i-- {
			removeDirIfEmpty(dirs[i], absDir, out)
		}
	}
	for _, rel := range []string{filepath.Join("packs", "languages"), "packs"} {
		if dir := filepath.Join(absDir, rel); isRealDir(dir) {
			removeDirIfEmpty(dir, absDir, out)
		}
	}

	if _, err := os.Lstat(packDir); err != nil {
		return ""
	}
	if isRealDir(packDir) {
		return "it still holds files"
	}
	return "it is not a directory"
}

func isRealDir(p string) bool {
	fi, err := os.Lstat(p)
	return err == nil && fi.IsDir()
}

func removeDirIfEmpty(dir, absDir string, out io.Writer) {
	entries, err := os.ReadDir(dir)
	if err == nil && len(entries) > 0 {
		return
	}
	if err == nil {
		err = os.Remove(dir)
	}
	if err != nil {
		writef(out, "⚠ Could not remove the directory %s: %v\n", packRemoveDisplayPath(absDir, dir), err)
	}
}

func packRemoveDisplayPath(absDir, p string) string {
	rel, err := filepath.Rel(absDir, p)
	if err != nil {
		return p
	}
	return filepath.ToSlash(rel)
}

func printPackRemoveDryRun(out io.Writer, lang string, plan packRemovePlan) {
	writef(out, "Dry run for pack %s (%d to delete, %d to keep, %d already missing):\n", lang, len(plan.deletes), len(plan.kept), len(plan.missing))
	for _, rel := range plan.deletes {
		writef(out, "  delete  %s\n", rel)
	}
	for _, k := range plan.kept {
		writef(out, "  keep    %s: %s\n", k.path, k.reason)
	}
	for _, key := range plan.missing {
		writef(out, "  untrack %s: already missing\n", key)
	}
	writef(out, "It would also remove %s and its %d entries from .ralph/manifest.toml, so ralph no longer tracks the kept files.\n", lang, len(plan.keys))
	writef(out, "Nothing was changed.\n")
}

func printPackRemoveSummary(out io.Writer, lang string, plan packRemovePlan, leftReason string) {
	writef(out, "✓ Pack %s removed (%d deleted, %d kept, %d already missing)\n", lang, len(plan.deletes), len(plan.kept), len(plan.missing))
	if len(plan.kept) > 0 {
		writef(out, "Kept:\n")
		for _, k := range plan.kept {
			writef(out, "  %s: %s\n", k.path, k.reason)
		}
		writef(out, "ralph no longer tracks the kept files: delete them yourself if you do not need them, and note that ralph pack add %s overwrites them.\n", lang)
	}
	if leftReason != "" {
		writef(out, "%s/ was left in place because %s.\n", filepath.ToSlash(packRelDir(lang)), leftReason)
	}
}
