package upgrade

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/yoshpy-dev/ralph/internal/scaffold"
)

// OpKind identifies the kind of file operation a FileOp represents.
type OpKind int

const (
	// OpDelete removes a file from disk.
	OpDelete OpKind = iota
	// OpCreate writes a new file that does not currently exist on disk.
	OpCreate
	// OpUpdate overwrites an existing file's content.
	OpUpdate
)

// String returns a lowercase op name, used in error messages.
func (k OpKind) String() string {
	switch k {
	case OpDelete:
		return "delete"
	case OpCreate:
		return "create"
	case OpUpdate:
		return "update"
	default:
		return fmt.Sprintf("OpKind(%d)", int(k))
	}
}

// FileOp is a single file-level operation produced by PlanCoreReplace.
type FileOp struct {
	Kind OpKind
	Path string
	// Content is the new file content. Empty/nil for OpDelete.
	Content []byte
	// NewHash is the content hash the caller should record in the manifest
	// for Path once the operation is applied. Empty for OpDelete.
	NewHash string
}

// ManifestRefreshEntry records a path whose disk content already equals the
// new template content but whose manifest-recorded hash is stale. No file
// write is required; the caller only needs to advance the manifest's
// recorded hash to Hash. This is what keeps re-planning over a partially
// applied tree stable: files that already landed on disk in a prior
// (interrupted) run are recognized as settled instead of re-planned or
// misclassified as drift.
type ManifestRefreshEntry struct {
	Path string
	Hash string
}

// DriftEntry records a path where disk content diverges from both the
// manifest-recorded state and the new template content, with no fork record
// to explain the divergence. PlanCoreReplace never plans a write for a
// drifted path (non-destructive default); resolving drift is an eject/adopt
// decision left to later phases.
type DriftEntry struct {
	Path         string
	RecordedHash string
	DiskHash     string
	// NewHash is the new template hash, or empty if the template no longer
	// has this path.
	NewHash string
}

// AdvisoryEntry records a path that PlanCoreReplace intentionally leaves
// untouched but surfaces to the operator: every fork path produces an
// advisory entry, even when its content is byte-identical to the new
// template — that case renders in the report as a "_No differences._"
// section, not a hidden or omitted one — plus seed paths whose template
// side changed since last applied. This only carries the path and hashes;
// rendering the actual diff is advisory.go's job.
type AdvisoryEntry struct {
	Path     string
	Owner    string
	DiskHash string
	NewHash  string
}

// ReplacePlan is the ordered, deterministic outcome of PlanCoreReplace.
type ReplacePlan struct {
	// Ops is ordered: all deletes, then all creates, then all updates, each
	// group sorted by Path.
	Ops             []FileOp
	ManifestRefresh []ManifestRefreshEntry
	Drift           []DriftEntry
	Advisories      []AdvisoryEntry
	// LegacySkipped lists paths whose manifest entry has owner=block or is a
	// legacy (unattributed, ManifestFile.IsLegacyOwner()) entry. These are
	// left entirely alone; classification is a later phase's responsibility.
	LegacySkipped []string
	// ManifestRemove lists, in sorted order, the paths whose manifest entry
	// the caller must drop once ApplyOps has returned a nil error (i.e.
	// after the commit barrier — see ApplyOps' doc comment). This covers
	// every owner=core path the template no longer ships: both the path
	// paired with an OpDelete op (disk still had the unmodified file) and
	// the path with no op at all (disk was already absent, so there is
	// nothing to delete but the stale manifest entry still needs to go).
	// Without this signal a caller has no way to notice the second case and
	// the manifest accumulates entries for paths that exist nowhere.
	ManifestRemove []string
	// Preserved lists, in sorted order, manifest-tracked paths that matched
	// a ReplaceOptions.PreservePrefixes entry and had no corresponding
	// desired-state content. These paths are left completely untouched: no
	// op, no ManifestRemove entry, no Drift entry. See ReplaceOptions'
	// PreservePrefixes doc comment.
	Preserved []string
}

// ReplaceOptions controls how PlanCoreReplaceDesired treats specific paths
// before the normal ownership-based classification rules run.
type ReplaceOptions struct {
	// SkipPaths excludes matching paths from classification entirely: the
	// planner never emits an op, refresh, drift, or advisory entry for
	// them, and they never appear in Preserved either. Callers use this for
	// paths handled by a dedicated mechanism outside the replace planner
	// (e.g. the settings.json 3-way merge, or managed-block surfaces).
	SkipPaths map[string]bool
	// PreservePrefixes lists slash-separated path prefixes (e.g.
	// "packs/languages/golang/") for which template absence must not
	// produce a delete op, a ManifestRemove entry, or a Drift entry. A
	// manifest-tracked path under one of these prefixes that has no
	// desired-state content is left on disk and in the manifest untouched,
	// and is recorded in ReplacePlan.Preserved instead. This only
	// suppresses the "template no longer has this path" outcome: a path
	// under a preserve prefix that does have desired-state content is
	// classified normally (creates/updates/no-op still apply), since the
	// prefix exists to protect namespaces that have gone fully absent (e.g.
	// an uninstalled language pack), not to opt a still-active namespace
	// out of upgrades.
	PreservePrefixes []string
}

// hasPreservePrefix reports whether path matches one of prefixes.
func hasPreservePrefix(path string, prefixes []string) bool {
	for _, prefix := range prefixes {
		if strings.HasPrefix(path, prefix) {
			return true
		}
	}
	return false
}

// PlanCoreReplace computes an ordered file-operation plan over the union of
// template paths (walked from templateFS) and manifest-tracked paths in m.
//
// This is a thin adapter over PlanCoreReplaceDesired: it walks templateFS
// into a path→content map and delegates with a zero-value ReplaceOptions
// (no skip paths, no preserve prefixes). Callers that need to compose
// desired state from more than one fs.FS (e.g. base templates plus
// installed language packs), or that need SkipPaths/PreservePrefixes,
// should build the map themselves and call PlanCoreReplaceDesired directly.
func PlanCoreReplace(m *scaffold.Manifest, targetDir string, templateFS fs.FS) (ReplacePlan, error) {
	tmplFiles, err := collectTemplateFiles(templateFS)
	if err != nil {
		return ReplacePlan{}, err
	}
	return PlanCoreReplaceDesired(m, targetDir, tmplFiles, ReplaceOptions{})
}

// PlanCoreReplaceDesired computes an ordered file-operation plan over the
// union of desired-state paths (template-relative path → content) and
// manifest-tracked paths in m.
//
// PlanCoreReplaceDesired only reads disk (to hash existing content); it
// never writes. Every path — manifest keys and desired-state keys alike —
// is validated with scaffold.CleanLocalRelPath before use; an invalid path
// aborts planning with an error (spec AC-9).
//
// opts.SkipPaths excludes matching paths from all classification before the
// per-owner rules run. opts.PreservePrefixes exempts matching
// manifest-tracked, template-absent paths from delete/ManifestRemove/drift,
// collecting them into ReplacePlan.Preserved instead. See ReplaceOptions'
// field docs.
//
// Classification is driven by each path's manifest ownership attribute
// (core/fork/seed/block) plus the legacy case (no ownership recorded). See
// docs/specs/2026-08-17-overlay-scaffold-v2.md, section "層モデル", and the
// Phase 1 plan's slice-4 handoff for the exact per-owner rules.
func PlanCoreReplaceDesired(m *scaffold.Manifest, targetDir string, desired map[string][]byte, opts ReplaceOptions) (ReplacePlan, error) {
	if m == nil {
		return ReplacePlan{}, errors.New("nil manifest")
	}

	tmplFiles := make(map[string][]byte, len(desired))
	for rawPath, content := range desired {
		cleanPath, cerr := cleanPathKey(rawPath)
		if cerr != nil {
			return ReplacePlan{}, fmt.Errorf("desired path %q: %w", rawPath, cerr)
		}
		tmplFiles[cleanPath] = content
	}

	manifestFiles := make(map[string]scaffold.ManifestFile, len(m.Files))
	for rawPath, entry := range m.Files {
		cleanPath, cerr := cleanPathKey(rawPath)
		if cerr != nil {
			return ReplacePlan{}, fmt.Errorf("manifest path %q: %w", rawPath, cerr)
		}
		manifestFiles[cleanPath] = entry
	}

	allPaths := make(map[string]struct{}, len(tmplFiles)+len(manifestFiles))
	for p := range tmplFiles {
		allPaths[p] = struct{}{}
	}
	for p := range manifestFiles {
		allPaths[p] = struct{}{}
	}
	sortedPaths := make([]string, 0, len(allPaths))
	for p := range allPaths {
		sortedPaths = append(sortedPaths, p)
	}
	sort.Strings(sortedPaths)

	var plan ReplacePlan
	var deletes, creates, updates []FileOp

	for _, path := range sortedPaths {
		if opts.SkipPaths[path] {
			continue
		}

		tmplContent, hasTemplate := tmplFiles[path]
		entry, hasEntry := manifestFiles[path]

		if !hasTemplate && hasEntry && hasPreservePrefix(path, opts.PreservePrefixes) {
			plan.Preserved = append(plan.Preserved, path)
			continue
		}

		diskContent, hasDisk, derr := readDiskFile(targetDir, path)
		if derr != nil {
			return ReplacePlan{}, fmt.Errorf("reading disk file %q: %w", path, derr)
		}

		classifyPath(&plan, &deletes, &creates, &updates, path, entry, hasEntry, tmplContent, hasTemplate, diskContent, hasDisk)
	}

	plan.Ops = make([]FileOp, 0, len(deletes)+len(creates)+len(updates))
	plan.Ops = append(plan.Ops, deletes...)
	plan.Ops = append(plan.Ops, creates...)
	plan.Ops = append(plan.Ops, updates...)

	sort.Strings(plan.Preserved)

	return plan, nil
}

// classifyPath applies the ownership-based classification rules for a single
// path and appends the outcome to plan / the op accumulators.
func classifyPath(
	plan *ReplacePlan,
	deletes, creates, updates *[]FileOp,
	path string,
	entry scaffold.ManifestFile,
	hasEntry bool,
	tmplContent []byte,
	hasTemplate bool,
	diskContent []byte,
	hasDisk bool,
) {
	if !hasEntry {
		classifyUntracked(plan, creates, path, tmplContent, hasTemplate, diskContent, hasDisk)
		return
	}

	if entry.Owner == scaffold.OwnerBlock || entry.IsLegacyOwner() {
		plan.LegacySkipped = append(plan.LegacySkipped, path)
		return
	}

	recordedHash := entry.DiskHash
	if recordedHash == "" {
		recordedHash = entry.Hash
	}

	switch entry.Owner {
	case scaffold.OwnerCore:
		classifyCore(plan, deletes, creates, updates, path, recordedHash, tmplContent, hasTemplate, diskContent, hasDisk)
	case scaffold.OwnerFork:
		classifyFork(plan, path, entry, tmplContent, hasTemplate, diskContent, hasDisk)
	case scaffold.OwnerSeed:
		// Seed's "changed since last seen" comparison is against the
		// previously recorded *template* hash (has the template evolved?),
		// not the disk-content hash used for core's modification check —
		// seed files are user-owned once created, so drift-vs-disk is never
		// relevant here.
		recordedTemplateHash := entry.TemplateHash
		if recordedTemplateHash == "" {
			recordedTemplateHash = entry.Hash
		}
		classifySeed(plan, creates, path, recordedTemplateHash, tmplContent, hasTemplate, diskContent, hasDisk)
	default:
		// Unknown/future owner values: treat like legacy — leave alone
		// rather than guess.
		plan.LegacySkipped = append(plan.LegacySkipped, path)
	}
}

func classifyCore(
	plan *ReplacePlan,
	deletes, creates, updates *[]FileOp,
	path, recordedHash string,
	tmplContent []byte,
	hasTemplate bool,
	diskContent []byte,
	hasDisk bool,
) {
	if hasTemplate {
		templateHash := scaffold.HashBytes(tmplContent)
		switch {
		case !hasDisk:
			*creates = append(*creates, FileOp{Kind: OpCreate, Path: path, Content: tmplContent, NewHash: templateHash})
		default:
			switch diskHash := scaffold.HashBytes(diskContent); diskHash {
			case templateHash:
				if recordedHash != templateHash {
					plan.ManifestRefresh = append(plan.ManifestRefresh, ManifestRefreshEntry{Path: path, Hash: templateHash})
				}
				// else: fully settled, true no-op.
			case recordedHash:
				*updates = append(*updates, FileOp{Kind: OpUpdate, Path: path, Content: tmplContent, NewHash: templateHash})
			default:
				plan.Drift = append(plan.Drift, DriftEntry{Path: path, RecordedHash: recordedHash, DiskHash: diskHash, NewHash: templateHash})
			}
		}
		return
	}

	// Template no longer has this path: a candidate for deletion, but never
	// destroy a modified file.
	if !hasDisk {
		// Already gone from disk; nothing to delete, but the manifest entry
		// is now stale and must still be dropped by the caller.
		plan.ManifestRemove = append(plan.ManifestRemove, path)
		return
	}
	diskHash := scaffold.HashBytes(diskContent)
	if diskHash == recordedHash {
		*deletes = append(*deletes, FileOp{Kind: OpDelete, Path: path})
		plan.ManifestRemove = append(plan.ManifestRemove, path)
		return
	}
	plan.Drift = append(plan.Drift, DriftEntry{Path: path, RecordedHash: recordedHash, DiskHash: diskHash, NewHash: ""})
}

func classifyFork(
	plan *ReplacePlan,
	path string,
	entry scaffold.ManifestFile,
	tmplContent []byte,
	hasTemplate bool,
	diskContent []byte,
	hasDisk bool,
) {
	var newHash string
	if hasTemplate {
		newHash = scaffold.HashBytes(tmplContent)
	}
	diskHash := entry.DiskHash
	if hasDisk {
		diskHash = scaffold.HashBytes(diskContent)
	}
	plan.Advisories = append(plan.Advisories, AdvisoryEntry{Path: path, Owner: scaffold.OwnerFork, DiskHash: diskHash, NewHash: newHash})
}

// classifySeed applies the owner=seed rules. recordedTemplateHash is the
// template hash last recorded for this path (not a disk-content hash — seed
// content is user-owned once created, so upgrade never compares it against
// disk).
func classifySeed(
	plan *ReplacePlan,
	creates *[]FileOp,
	path, recordedTemplateHash string,
	tmplContent []byte,
	hasTemplate bool,
	diskContent []byte,
	hasDisk bool,
) {
	if !hasDisk {
		if hasTemplate {
			*creates = append(*creates, FileOp{Kind: OpCreate, Path: path, Content: tmplContent, NewHash: scaffold.HashBytes(tmplContent)})
		}
		return
	}
	if !hasTemplate {
		return
	}
	templateHash := scaffold.HashBytes(tmplContent)
	if templateHash != recordedTemplateHash {
		diskHash := scaffold.HashBytes(diskContent)
		plan.Advisories = append(plan.Advisories, AdvisoryEntry{Path: path, Owner: scaffold.OwnerSeed, DiskHash: diskHash, NewHash: templateHash})
	}
	// else: template unchanged since last recorded application, no-op.
}

// classifyUntracked handles paths with no manifest entry at all: template
// files ralph has never recorded ownership for yet.
func classifyUntracked(
	plan *ReplacePlan,
	creates *[]FileOp,
	path string,
	tmplContent []byte,
	hasTemplate bool,
	diskContent []byte,
	hasDisk bool,
) {
	if !hasTemplate {
		// Should not happen (path came from the template-or-manifest union
		// and has no manifest entry), but guard defensively.
		return
	}
	templateHash := scaffold.HashBytes(tmplContent)
	if !hasDisk {
		*creates = append(*creates, FileOp{Kind: OpCreate, Path: path, Content: tmplContent, NewHash: templateHash})
		return
	}
	diskHash := scaffold.HashBytes(diskContent)
	if diskHash == templateHash {
		plan.ManifestRefresh = append(plan.ManifestRefresh, ManifestRefreshEntry{Path: path, Hash: templateHash})
		return
	}
	plan.Drift = append(plan.Drift, DriftEntry{Path: path, RecordedHash: "", DiskHash: diskHash, NewHash: templateHash})
}

// ApplyOps executes plan.Ops against targetDir in order (deletes, then
// creates, then updates, each sorted by path), stopping at the first
// failure. Ops after a failure are not attempted.
//
// Before executing any op, ApplyOps validates every op's Path with
// cleanPathKey; on the first invalid path it returns an error naming that
// path and performs no filesystem operations at all (validate-all-upfront,
// so a plan that fails validation never leaves partial-failure semantics to
// reason about). This guards against a hand-built ReplacePlan carrying a
// path that would resolve outside targetDir.
//
// In that same validate-all-upfront pass, ApplyOps also os.Lstat's every
// op's target path (update and delete targets, and create targets in case a
// path collides with something already there): if the entry exists and is
// not a regular file — a symlink (including a dangling one, since Lstat
// reports the symlink's own mode without following it) or a directory —
// ApplyOps returns an error naming the path and performs no filesystem
// operations at all. A missing entry (os.ErrNotExist) is fine; that is the
// ordinary create case.
//
// ApplyOps never reads or writes the manifest — that is the commit barrier:
// callers must only advance manifest state (recorded hashes for creates,
// updates, and ReplacePlan.ManifestRefresh entries, plus dropping the
// entries listed in ReplacePlan.ManifestRemove — the one entry in this list
// that removes rather than advances a manifest hash) after ApplyOps returns
// a nil error. A non-nil error means the on-disk tree may be partially
// updated but the manifest must not be advanced; re-planning over that tree
// with PlanCoreReplace produces a plan that completes the remaining work
// without misclassifying the already-applied paths as drift.
func ApplyOps(targetDir string, plan ReplacePlan) error {
	for _, op := range plan.Ops {
		if _, cerr := cleanPathKey(op.Path); cerr != nil {
			return fmt.Errorf("%s %s: invalid path: %w", op.Kind, op.Path, cerr)
		}
	}

	for _, op := range plan.Ops {
		full := filepath.Join(targetDir, filepath.FromSlash(op.Path))
		fi, err := os.Lstat(full)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				continue
			}
			return fmt.Errorf("%s %s: lstat: %w", op.Kind, op.Path, err)
		}
		if !fi.Mode().IsRegular() {
			return fmt.Errorf("%s %s: refusing to operate on non-regular file (mode %s)", op.Kind, op.Path, fi.Mode())
		}
	}

	for _, op := range plan.Ops {
		full := filepath.Join(targetDir, filepath.FromSlash(op.Path))
		switch op.Kind {
		case OpDelete:
			if err := os.Remove(full); err != nil && !errors.Is(err, os.ErrNotExist) {
				return fmt.Errorf("%s %s: %w", op.Kind, op.Path, err)
			}
		case OpCreate, OpUpdate:
			if err := os.MkdirAll(filepath.Dir(full), 0755); err != nil {
				return fmt.Errorf("%s %s: %w", op.Kind, op.Path, err)
			}
			if err := os.WriteFile(full, op.Content, scaffold.FilePerm(op.Path)); err != nil {
				return fmt.Errorf("%s %s: %w", op.Kind, op.Path, err)
			}
		default:
			return fmt.Errorf("unknown op kind %v for %s", op.Kind, op.Path)
		}
	}
	return nil
}

// collectTemplateFiles walks templateFS and returns a map of validated,
// slash-normalized relative paths to file content.
func collectTemplateFiles(templateFS fs.FS) (map[string][]byte, error) {
	files := make(map[string][]byte)
	err := fs.WalkDir(templateFS, ".", func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		clean, cerr := cleanPathKey(p)
		if cerr != nil {
			return fmt.Errorf("template path %q: %w", p, cerr)
		}
		data, rerr := fs.ReadFile(templateFS, p)
		if rerr != nil {
			return fmt.Errorf("reading template file %q: %w", p, rerr)
		}
		files[clean] = data
		return nil
	})
	if err != nil {
		return nil, err
	}
	return files, nil
}

// cleanPathKey validates raw through scaffold.CleanLocalRelPath and
// normalizes the result to forward slashes, so template-walk paths and
// manifest keys resolve to the same canonical form regardless of platform.
func cleanPathKey(raw string) (string, error) {
	clean, err := scaffold.CleanLocalRelPath(raw)
	if err != nil {
		return "", err
	}
	return filepath.ToSlash(clean), nil
}

// readDiskFile reads targetDir/path, reporting hasDisk=false (with no error)
// when the file does not exist.
func readDiskFile(targetDir, path string) (content []byte, hasDisk bool, err error) {
	full := filepath.Join(targetDir, filepath.FromSlash(path))
	data, err := os.ReadFile(full)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, false, nil
		}
		return nil, false, err
	}
	return data, true, nil
}
