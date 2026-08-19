package cli

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/yoshpy-dev/ralph/internal/scaffold"
	"github.com/yoshpy-dev/ralph/internal/upgrade"
)

// ErrUpgradeDriftRemaining is returned by runUpgradeV2 when the upgrade
// otherwise completes successfully but one or more paths were left with
// unresolved drift (disk content diverges from both the recorded and new
// template hash, with no fork record to explain it). cmd/ralph/main.go maps
// this to a dedicated exit code (3) distinct from a genuine execution error
// (1), per docs/specs/2026-08-17-overlay-scaffold-v2.md FR-4 and the plan's
// exit-code decision — this keeps drift machine-detectable ahead of
// `doctor --strict` (Phase 5).
var ErrUpgradeDriftRemaining = errors.New("upgrade completed with one or more paths left in unresolved drift")

// v2SettingsPath, v2SettingsSnapshotPath, and the two block surface paths are
// the "exception faces" excluded from the core replace planner and handled
// by their own dedicated mechanisms (3-way settings merge, managed block
// update). They are still present as keys in the desired-state map (so their
// template hash can be recorded), just never planned as plain
// create/update/delete ops.
const v2SettingsPath = ".claude/settings.json"

// v2SkipPaths returns the set of manifest-relative paths the v2 core replace
// planner must never classify: settings.json (3-way merge target) and the
// two managed-block surfaces (AGENTS.md, .gitignore). The settings snapshot
// itself (.ralph/core/settings.ralph.json, see internal/upgrade/snapshot.go)
// is also excluded: it is written only after the settings merge succeeds, as
// its own explicit step outside the planner's ordinary core-replace op
// ordering (see runUpgradeV2's 2-phase settings update).
func v2SkipPaths() map[string]bool {
	skip := map[string]bool{v2SettingsPath: true}
	for _, bs := range blockSurfaces {
		skip[bs.path] = true
	}
	skip[upgrade.SettingsSnapshotRelPath] = true
	return skip
}

// runUpgradeV2 is the non-interactive upgrade flow for v2-layout (overlay
// scaffold) projects: it wires the Phase 1 engine primitives (core replace
// planner + commit barrier, managed block engine, settings.json 3-way merge,
// advisory diff, upgrade report) end to end. No prompts, no stdin reads —
// every branch is deterministic given the manifest, disk state, and embedded
// templates. now is passed in by the caller (rather than calling time.Now()
// here) so report timestamps are reproducible in tests.
func runUpgradeV2(absDir, manifestPath string, oldManifest *scaffold.Manifest, opts upgradeOptions, out, errOut io.Writer, colorize bool, now time.Time) error {
	// Step 0: read+cache the settings snapshot before any writes happen, so
	// a partial failure later can never lose the oldOwned side of the 3-way
	// merge. A missing snapshot (Phase 2 init generation) falls back to "{}"
	// — see settingsmerge.go's MergeOwnedSettings doc and design decision in
	// docs/plans/active/2026-08-18-overlay-scaffold-v2-p3.md.
	oldOwnedSnapshot, snapshotFound, err := upgrade.LoadSettingsSnapshot(absDir)
	if err != nil {
		return fmt.Errorf("reading settings snapshot: %w", err)
	}
	oldOwned := oldOwnedSnapshot
	if !snapshotFound {
		oldOwned = []byte("{}")
	}

	baseFS, err := scaffold.BaseFS()
	if err != nil {
		return fmt.Errorf("loading templates: %w", err)
	}

	var notes []string
	desired, preservePrefixes, retainedPacks, buildNotes, err := buildDesiredStateV2(baseFS, oldManifest, errOut)
	if err != nil {
		return fmt.Errorf("building desired state: %w", err)
	}
	notes = append(notes, buildNotes...)

	planOpts := upgrade.ReplaceOptions{
		SkipPaths:        v2SkipPaths(),
		PreservePrefixes: preservePrefixes,
		OwnerForPath:     ownerForScaffoldPath,
	}
	plan, err := upgrade.PlanCoreReplaceDesired(oldManifest, absDir, desired, planOpts)
	if err != nil {
		return fmt.Errorf("planning upgrade: %w", err)
	}

	if opts.DryRun {
		return renderUpgradeV2Preview(plan, desired, absDir, Version, out, errOut, colorize, opts, oldOwned, oldOwnedSnapshot, snapshotFound)
	}

	if err := upgrade.ApplyOps(absDir, plan); err != nil {
		return fmt.Errorf("applying upgrade: %w", err)
	}

	blockOutcomes, blockNotes, err := applyBlockUpdatesV2(absDir, desired, errOut, true)
	if err != nil {
		return fmt.Errorf("updating managed blocks: %w", err)
	}
	notes = append(notes, blockNotes...)

	currentSettings, err := readFinalDiskContent(absDir, v2SettingsPath)
	if err != nil {
		return fmt.Errorf("reading %s: %w", v2SettingsPath, err)
	}
	newOwnedSettings := desired[v2SettingsPath]

	mergeResult, err := upgrade.MergeOwnedSettings(currentSettings, oldOwned, newOwnedSettings)
	if err != nil {
		return fmt.Errorf("merging %s: %w", v2SettingsPath, err)
	}
	finalSettingsContent := currentSettings
	if mergeResult.Changed {
		if err := writeFileV2(absDir, v2SettingsPath, mergeResult.Content); err != nil {
			return fmt.Errorf("writing %s: %w", v2SettingsPath, err)
		}
		finalSettingsContent = mergeResult.Content
	}

	// Phase 2 of the settings snapshot update: only write the new snapshot
	// content once the settings merge above has fully succeeded. If a
	// failure happens between here and manifest write, the snapshot on disk
	// (if it advances) already reflects newOwnedSettings; a re-run recomputes
	// the same merge (idempotent — see settingsmerge.go) and simply
	// completes the remaining steps. If this write itself fails, oldOwned
	// was never lost (it was cached in step 0, before any write), so a
	// re-run still has a correct 3-way merge input.
	finalSnapshotContent := oldOwnedSnapshot
	snapshotNeedsWrite := !snapshotFound || !bytes.Equal(oldOwnedSnapshot, newOwnedSettings)
	if !snapshotFound {
		notes = append(notes, "settings snapshot (.ralph/core/settings.ralph.json) was missing (pre-Phase-3 init); used \"{}\" as the oldOwned fallback, so stale ralph-owned settings entries were not pruned this run")
	}
	if snapshotNeedsWrite {
		if err := writeFileV2(absDir, upgrade.SettingsSnapshotRelPath, newOwnedSettings); err != nil {
			return fmt.Errorf("writing settings snapshot: %w", err)
		}
		finalSnapshotContent = newOwnedSettings
	}

	// AR#2 (cycle-2 cross-review, docs/reports/cross-review-triage-overlay-scaffold-v2-p3.md):
	// a converged tree (nothing left for the core replace plan, both block
	// surfaces already match, the settings merge is a no-op, and the
	// snapshot already reflects the current owned settings) must produce a
	// byte-for-byte no-op re-run — no dated report, no manifest rewrite, no
	// git-hook reinstall attempt. Every write above this point is already
	// individually conditional (ApplyOps has nothing to do; block/settings/
	// snapshot writes only fire when their own outcome changed), so the only
	// remaining unconditional writes are the report, the manifest commit
	// barrier, and installManagedGitHooks — gated below.
	//
	// The oldManifest.Meta.Version == Version check is C3-2 (cycle 3,
	// docs/reports/self-review-2026-08-18-overlay-scaffold-v2-p3.md): NFR-1
	// promises a no-op only for a re-run on the *same binary*. Without this
	// guard, a Go-only release whose templates are byte-identical to the
	// last one would converge on Ops/blocks/settings/snapshot yet skip the
	// manifest rebuild that would otherwise advance meta.version, leaving
	// doctor's version check permanently unsatisfiable.
	if oldManifest.Meta.Version == Version && isFullyConvergedV2(plan, blockOutcomes, mergeResult, snapshotNeedsWrite) {
		return finishNoOpUpgradeV2(absDir, Version, plan, out, errOut)
	}

	advisories, err := upgrade.RenderAdvisoryDiffs(absDir, desiredFS(desired), Version, plan.Advisories)
	if err != nil {
		return fmt.Errorf("rendering advisory diffs: %w", err)
	}

	reportData := upgrade.UpgradeReportData{
		TemplateVersion:      Version,
		GeneratedAt:          now.Format(time.RFC3339),
		UnresolvedDrift:      plan.Drift,
		Advisories:           advisories,
		LegacySkipped:        plan.LegacySkipped,
		ManifestRefreshPaths: manifestRefreshPaths(plan.ManifestRefresh),
		Notes:                notes,
	}
	for _, op := range plan.Ops {
		switch op.Kind {
		case upgrade.OpDelete:
			reportData.DeletedPaths = append(reportData.DeletedPaths, op.Path)
		case upgrade.OpCreate:
			reportData.CreatedPaths = append(reportData.CreatedPaths, op.Path)
		case upgrade.OpUpdate:
			reportData.UpdatedPaths = append(reportData.UpdatedPaths, op.Path)
		}
	}

	reportRelPath := upgrade.UpgradeReportRelPath(Version, now.Format("2006-01-02"))
	if err := upgrade.WriteUpgradeReport(absDir, reportRelPath, upgrade.RenderUpgradeReport(reportData)); err != nil {
		return fmt.Errorf("writing upgrade report: %w", err)
	}

	if err := rebuildManifestV2(manifestPath, oldManifest, Version, retainedPacks, desired, plan, blockOutcomes, finalSettingsContent, finalSnapshotContent); err != nil {
		return fmt.Errorf("writing manifest: %w", err)
	}

	// Reinstall/refresh managed git hooks now that every file write and the
	// manifest barrier have succeeded — same placement semantics as the
	// legacy engine (immediately after manifest.Write, before the summary)
	// and as init.go's own call. installManagedGitHooks is idempotent and
	// best-effort: failures are reported on errOut, never fail the upgrade.
	// Only reached on the non-no-op path (see isFullyConvergedV2 above) —
	// writeWrappedGitHook rewrites its guard/wrapper files on every call
	// even when their content is already identical, so calling it
	// unconditionally would itself violate a converged tree's zero-writes
	// contract (AR#2, cycle 2).
	installManagedGitHooks(absDir, out, errOut)

	if err := removeLegacyBaselineIfPresent(absDir); err != nil {
		return err
	}

	writef(out, "Upgrade complete: %s\n", Version)
	writef(out, "  created: %d, updated: %d, deleted: %d\n", len(reportData.CreatedPaths), len(reportData.UpdatedPaths), len(reportData.DeletedPaths))
	writef(out, "  report: %s\n", reportRelPath)

	if len(plan.Drift) > 0 {
		writeDriftGuidanceV2(errOut, plan.Drift, "left untouched; see report for detail")
		return fmt.Errorf("%w (%d path(s)); see %s", ErrUpgradeDriftRemaining, len(plan.Drift), reportRelPath)
	}

	return nil
}

// isFullyConvergedV2 reports whether this run produced zero content changes
// across every mechanism the v2 upgrade flow drives: the core replace plan
// (creates/updates/deletes, hash-only manifest refreshes, and stale-manifest
// removals — new seed files are always represented as OpCreate entries in
// plan.Ops, so "no seeds to create" is already covered by the Ops check),
// both managed-block surfaces, the settings.json 3-way merge, and the
// settings snapshot.
//
// Unresolved drift, legacy-skipped paths, fork advisories, and preserved-pack
// namespaces are deliberately excluded from this check: those categories are
// permanently left untouched by design (see ReplacePlan's doc comments), so
// their continued presence on a re-run never implies a pending write — see
// AR#2 (cycle 2) in docs/reports/cross-review-triage-overlay-scaffold-v2-p3.md.
// Fork advisories specifically can never clear (classifyFork appends one for
// every fork entry unconditionally, and rebuildManifestV2 carries fork
// entries over verbatim), so excluding them here is not an oversight.
//
// A pending seed advisory, by contrast, is not excluded: it is one-shot
// precisely because the manifest rebuild it would otherwise trigger advances
// TemplateHash and clears it (see SetFileOwned). Skipping that rebuild here
// would make the advisory permanent and invisible — see C3-1 (cycle 3) in
// docs/reports/self-review-2026-08-18-overlay-scaffold-v2-p3.md.
func isFullyConvergedV2(plan upgrade.ReplacePlan, blockOutcomes map[string]blockUpdateOutcome, mergeResult upgrade.SettingsMergeResult, snapshotNeedsWrite bool) bool {
	if len(plan.Ops) != 0 || len(plan.ManifestRefresh) != 0 || len(plan.ManifestRemove) != 0 {
		return false
	}
	for _, a := range plan.Advisories {
		if a.Owner == scaffold.OwnerSeed {
			return false
		}
	}
	for _, bs := range blockSurfaces {
		oc, ok := blockOutcomes[bs.path]
		if !ok || oc.outcome != upgrade.BlockUnchanged {
			return false
		}
	}
	if mergeResult.Changed {
		return false
	}
	return !snapshotNeedsWrite
}

// finishNoOpUpgradeV2 is the true-no-op tail of runUpgradeV2, reached only
// when isFullyConvergedV2 holds: every file-content write already happened
// conditionally above (and fired zero times), so the only remaining steps
// are the ones this function deliberately skips — no dated report, no
// manifest rewrite (the on-disk manifest already matches; rebuildManifestV2
// would only advance Meta.Updated), and no installManagedGitHooks call
// (whose guard/wrapper writes are unconditional even when content is
// unchanged). A leftover .ralph/baseline directory, if present, is still
// removed — its presence is independent of plan convergence and removal
// itself is idempotent and stat-gated (removeLegacyBaselineIfPresent).
//
// Unresolved drift can persist forever in an otherwise-converged tree (drift
// paths are permanently left untouched), so a no-op run still returns the
// ErrUpgradeDriftRemaining sentinel when drift remains — but without a
// report-path reference, since no report was written this run.
func finishNoOpUpgradeV2(absDir, version string, plan upgrade.ReplacePlan, out, errOut io.Writer) error {
	if err := removeLegacyBaselineIfPresent(absDir); err != nil {
		return err
	}

	writef(out, "Upgrade no-op: %s (already up to date, zero writes)\n", version)
	if len(plan.Drift) == 0 {
		return nil
	}

	writef(out, "  drift (untouched): %d files\n", len(plan.Drift))
	writeDriftGuidanceV2(errOut, plan.Drift, "left untouched; no report written this run — tree already converged")
	return fmt.Errorf("%w (%d path(s))", ErrUpgradeDriftRemaining, len(plan.Drift))
}

// removeLegacyBaselineIfPresent removes a leftover .ralph/baseline directory
// (Phase 3 removed the baseline mechanism entirely — see the plan's 後始末
// step) without attempting any write when the directory is already absent:
// callers, including the true-no-op path (finishNoOpUpgradeV2), must not
// perform an unconditional RemoveAll on every run.
func removeLegacyBaselineIfPresent(absDir string) error {
	cleanBaseline, err := scaffold.CleanLocalRelPath(".ralph/baseline")
	if err != nil {
		return fmt.Errorf("resolving .ralph/baseline path: %w", err)
	}
	full := filepath.Join(absDir, cleanBaseline)
	if _, statErr := os.Stat(full); statErr != nil {
		if errors.Is(statErr, os.ErrNotExist) {
			return nil
		}
		return fmt.Errorf("checking .ralph/baseline: %w", statErr)
	}
	if err := os.RemoveAll(full); err != nil {
		return fmt.Errorf("removing .ralph/baseline: %w", err)
	}
	return nil
}

// buildDesiredStateV2 composes the full desired-state path→content map for a
// v2 upgrade: base templates plus every installed language pack (payload
// under packs/languages/<lang>/, rule.md remapped to
// .claude/rules/ralph/<lang>.md). Packs that cannot be loaded — the pack
// itself no longer exists in the embedded templates, or reading it fails —
// are warned about on errOut and their entire namespace is added to
// preservePrefixes instead of desired content, so the replace planner leaves
// them completely alone (AC-9). notes carries the same warning text for the
// upgrade report.
func buildDesiredStateV2(baseFS fs.FS, oldManifest *scaffold.Manifest, errOut io.Writer) (desired map[string][]byte, preservePrefixes []string, retainedPacks []string, notes []string, err error) {
	desired = make(map[string][]byte)
	err = fs.WalkDir(baseFS, ".", func(p string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			return nil
		}
		content, rerr := fs.ReadFile(baseFS, p)
		if rerr != nil {
			return fmt.Errorf("reading base template %q: %w", p, rerr)
		}
		desired[filepath.ToSlash(p)] = content
		return nil
	})
	if err != nil {
		return nil, nil, nil, nil, fmt.Errorf("walking base templates: %w", err)
	}

	installedPacks := oldManifest.Meta.Packs
	availablePacks, apErr := scaffold.AvailablePacks()
	available := make(map[string]bool, len(availablePacks))
	if apErr == nil {
		for _, p := range availablePacks {
			available[p] = true
		}
	}

	preserve := func(pack, reason string) {
		msg := fmt.Sprintf("pack %q: %s — preserving its disk/manifest entries untouched", pack, reason)
		writef(errOut, "Warning: %s\n", msg)
		notes = append(notes, msg)
		preservePrefixes = append(preservePrefixes, packPrefixFor(pack), packRuleRelPath(pack), legacyPackRuleRelPath(pack))
		retainedPacks = append(retainedPacks, pack)
	}

	for _, pack := range installedPacks {
		if apErr != nil {
			preserve(pack, fmt.Sprintf("unable to list available packs: %v", apErr))
			continue
		}
		if !available[pack] {
			preserve(pack, "no longer exists in the embedded templates")
			continue
		}

		packFS, pErr := scaffold.PackFS(pack)
		if pErr != nil {
			preserve(pack, fmt.Sprintf("load failed: %v", pErr))
			continue
		}

		// Load this pack's entries into a local map first, rather than
		// straight into desired: PreservePrefixes only suppresses the
		// template-absent outcome for paths that were never added to
		// desired in the first place (see ReplaceOptions.PreservePrefixes'
		// doc comment). If a load step below fails, packEntries is
		// discarded entirely and preserve() is called instead — so a
		// mid-load failure genuinely leaves every one of the pack's
		// disk/manifest entries alone, matching the warning it prints.
		packEntries := make(map[string][]byte)
		packPrefix := packPrefixFor(pack)
		walkErr := fs.WalkDir(packFS, ".", func(p string, d fs.DirEntry, dwErr error) error {
			if dwErr != nil {
				return dwErr
			}
			if d.IsDir() || packRenderSkipPaths[p] {
				return nil
			}
			content, rerr := fs.ReadFile(packFS, p)
			if rerr != nil {
				return fmt.Errorf("reading pack %s file %q: %w", pack, p, rerr)
			}
			packEntries[filepath.ToSlash(filepath.Join(packPrefix, p))] = content
			return nil
		})
		if walkErr != nil {
			preserve(pack, fmt.Sprintf("walk failed: %v", walkErr))
			continue
		}

		ruleContent, ok, rErr := packRuleContent(packFS)
		if rErr != nil {
			preserve(pack, fmt.Sprintf("rule read failed: %v", rErr))
			continue
		}
		if ok {
			packEntries[packRuleRelPath(pack)] = ruleContent
		}

		for path, content := range packEntries {
			desired[path] = content
		}
		retainedPacks = append(retainedPacks, pack)
	}

	return desired, preservePrefixes, retainedPacks, notes, nil
}

// blockUpdateOutcome records the per-surface result of a v2 managed-block
// update, keyed by manifest-relative path in the caller's map. ok is false
// when the surface was left untouched for a reason other than "already
// matches" (missing entirely, a symlink, a non-regular file, or a malformed
// existing block) — the manifest rebuild step carries the previous entry
// over unchanged in that case instead of computing a new disk hash.
type blockUpdateOutcome struct {
	outcome upgrade.BlockOutcome
	content []byte
	ok      bool
}

// applyBlockUpdatesV2 updates the two v2 managed-block surfaces (AGENTS.md,
// .gitignore) in place. AGENTS.md's managed content is the raw
// .ralph/core/AGENTS.core.md file (already exactly the block interior, no
// marker stripping needed); .gitignore's managed content is extracted from
// the interior of the new .gitignore template's own block (mirrors
// init.go's reconcileBlockSurfaces). A malformed existing block, or a
// symlinked/non-regular/missing surface, is left untouched with a warning
// (AC-12) rather than aborting the upgrade.
//
// write controls whether a computed update is actually persisted to disk.
// The real upgrade path calls this with write=true; the --dry-run preview
// (renderUpgradeV2Preview, AR#1, cycle 2, in docs/reports/cross-review-triage-overlay-scaffold-v2-p3.md)
// calls it with write=false to compute the same outcomes purely in memory,
// against the current on-disk content, so the preview can name pending
// block updates without ever touching the tree.
func applyBlockUpdatesV2(absDir string, desired map[string][]byte, errOut io.Writer, write bool) (map[string]blockUpdateOutcome, []string, error) {
	outcomes := make(map[string]blockUpdateOutcome, len(blockSurfaces))
	var notes []string

	agentsManaged, hasAgentsTemplate := desired[".ralph/core/AGENTS.core.md"]
	gitignoreTemplate, hasGitignoreTemplate := desired[".gitignore"]

	for _, bs := range blockSurfaces {
		var managed []byte
		switch bs.path {
		case "AGENTS.md":
			// Mirrors the .gitignore guard below and init.go's
			// reconcileBlockSurfaces stance (a missing managed-block source
			// means nothing to reconcile): a missing or empty
			// .ralph/core/AGENTS.core.md must never be treated as "empty
			// interior" and written, which would silently blank the user's
			// AGENTS.md managed block.
			if !hasAgentsTemplate || len(agentsManaged) == 0 {
				msg := fmt.Sprintf("%s: managed block source (.ralph/core/AGENTS.core.md) is missing from the desired state; left untouched", bs.path)
				writef(errOut, "Warning: %s\n", msg)
				notes = append(notes, msg)
				continue
			}
			managed = agentsManaged
		case ".gitignore":
			if !hasGitignoreTemplate {
				continue
			}
			interior, err := extractBlockInterior(gitignoreTemplate, bs.surface, bs.style)
			if err != nil {
				return nil, nil, fmt.Errorf("extracting managed block from template %s: %w", bs.path, err)
			}
			managed = interior
		default:
			continue
		}

		oc, note, err := updateOneBlockV2(absDir, bs.path, bs.surface, bs.style, managed, write, errOut)
		if err != nil {
			return nil, nil, err
		}
		outcomes[bs.path] = oc
		if note != "" {
			notes = append(notes, note)
		}
	}

	return outcomes, notes, nil
}

// updateOneBlockV2 updates a single managed-block surface on disk. It
// refuses to follow symlinks or operate on non-regular files (mirroring
// init.go's reconcileBlockSurfaces containment guard) and leaves a malformed
// existing block untouched, reporting both cases via a warning and a report
// note instead of failing the upgrade. write=false computes the same
// outcome against current on-disk content without ever calling
// os.WriteFile — see applyBlockUpdatesV2's doc comment.
func updateOneBlockV2(absDir, relPath, surface string, style upgrade.BlockMarkerStyle, managed []byte, write bool, errOut io.Writer) (blockUpdateOutcome, string, error) {
	full := filepath.Join(absDir, relPath)
	info, err := os.Lstat(full)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return blockUpdateOutcome{}, "", nil
		}
		return blockUpdateOutcome{}, "", fmt.Errorf("reading %s: %w", relPath, err)
	}
	if info.Mode()&os.ModeSymlink != 0 {
		msg := fmt.Sprintf("%s: is a symlink; left untouched", relPath)
		writef(errOut, "Warning: %s\n", msg)
		return blockUpdateOutcome{}, msg, nil
	}
	if !info.Mode().IsRegular() {
		msg := fmt.Sprintf("%s: is not a regular file; left untouched", relPath)
		writef(errOut, "Warning: %s\n", msg)
		return blockUpdateOutcome{}, msg, nil
	}

	current, err := os.ReadFile(full)
	if err != nil {
		return blockUpdateOutcome{}, "", fmt.Errorf("reading %s: %w", relPath, err)
	}

	result := upgrade.UpdateManagedBlockStyled(current, surface, managed, style)
	switch result.Outcome {
	case upgrade.BlockUpdated, upgrade.BlockAppended:
		if write {
			if err := os.WriteFile(full, result.Content, scaffold.FilePerm(relPath)); err != nil {
				return blockUpdateOutcome{}, "", fmt.Errorf("writing %s: %w", relPath, err)
			}
		}
		return blockUpdateOutcome{outcome: result.Outcome, content: result.Content, ok: true}, "", nil
	case upgrade.BlockMalformed:
		msg := fmt.Sprintf("%s: existing ralph managed block is malformed (%s); left untouched", relPath, result.Reason)
		writef(errOut, "Warning: %s\n", msg)
		return blockUpdateOutcome{outcome: result.Outcome}, msg, nil
	default: // BlockUnchanged
		return blockUpdateOutcome{outcome: result.Outcome, content: current, ok: true}, "", nil
	}
}

// rebuildManifestV2 is the commit barrier: it is only reached once every
// other write (core ops, blocks, settings, snapshot, report) has already
// succeeded, and produces the single manifest.toml write that advances
// ralph's recorded state to match. Fork entries, drift-path entries,
// preserved-pack entries, and legacy-skipped entries are carried over from
// oldManifest completely unchanged, so a later re-plan classifies them
// identically; every other desired-state path gets a fresh entry with its
// ownership attribute (core/seed/block) via ownerForScaffoldPath.
func rebuildManifestV2(
	manifestPath string,
	oldManifest *scaffold.Manifest,
	version string,
	retainedPacks []string,
	desired map[string][]byte,
	plan upgrade.ReplacePlan,
	blockOutcomes map[string]blockUpdateOutcome,
	finalSettingsContent, finalSnapshotContent []byte,
) error {
	newManifest := scaffold.NewManifest(version)
	newManifest.Meta.Packs = retainedPacks
	newManifest.SetLayoutV2()

	handled := make(map[string]bool)

	for path, entry := range oldManifest.Files {
		if entry.Owner == scaffold.OwnerFork {
			newManifest.Files[path] = entry
			handled[path] = true
		}
	}

	carryOverIfTracked := func(path string) {
		if handled[path] {
			return
		}
		if old, ok := oldManifest.Files[path]; ok {
			newManifest.Files[path] = old
		}
		handled[path] = true
	}
	for _, d := range plan.Drift {
		carryOverIfTracked(d.Path)
	}
	for _, p := range plan.LegacySkipped {
		carryOverIfTracked(p)
	}
	for _, p := range plan.Preserved {
		carryOverIfTracked(p)
	}

	sortedPaths := make([]string, 0, len(desired))
	for p := range desired {
		sortedPaths = append(sortedPaths, p)
	}
	sort.Strings(sortedPaths)

	for _, path := range sortedPaths {
		if handled[path] {
			continue
		}
		templateHash := scaffold.HashBytes(desired[path])

		switch path {
		case v2SettingsPath:
			if err := newManifest.SetFileOwned(path, scaffold.OwnerCore, templateHash, scaffold.HashBytes(finalSettingsContent)); err != nil {
				return err
			}
		case upgrade.SettingsSnapshotRelPath:
			if err := newManifest.SetFileOwned(path, scaffold.OwnerCore, templateHash, scaffold.HashBytes(finalSnapshotContent)); err != nil {
				return err
			}
		case "AGENTS.md", ".gitignore":
			oc := blockOutcomes[path]
			if oc.ok {
				if err := newManifest.SetFileOwned(path, scaffold.OwnerBlock, templateHash, scaffold.HashBytes(oc.content)); err != nil {
					return err
				}
			} else if old, ok := oldManifest.Files[path]; ok {
				newManifest.Files[path] = old
			}
		default:
			owner := ownerForScaffoldPath(path)
			diskHash := templateHash
			if owner == scaffold.OwnerSeed {
				diskContent, rerr := readFinalDiskContent(manifestRootFromManifestPath(manifestPath), path)
				if rerr != nil {
					return fmt.Errorf("reading %s for manifest rebuild: %w", path, rerr)
				}
				diskHash = scaffold.HashBytes(diskContent)
			}
			if err := newManifest.SetFileOwned(path, owner, templateHash, diskHash); err != nil {
				return err
			}
		}
		handled[path] = true
	}

	return newManifest.Write(manifestPath)
}

// manifestRootFromManifestPath derives the project root (absDir) from
// manifestPath (absDir/.ralph/manifest.toml), so rebuildManifestV2 does not
// need absDir threaded through as a separate parameter purely for the one
// seed-owner disk read it performs.
func manifestRootFromManifestPath(manifestPath string) string {
	return filepath.Dir(filepath.Dir(manifestPath))
}

// readFinalDiskContent reads targetDir/relPath, returning (nil, nil) when
// the file does not exist rather than an error — callers treat "absent" as
// a legitimate, hashable ("") state.
func readFinalDiskContent(targetDir, relPath string) ([]byte, error) {
	full := filepath.Join(targetDir, filepath.FromSlash(relPath))
	data, err := os.ReadFile(full)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}
	return data, nil
}

// writeFileV2 writes content to targetDir/relPath, creating parent
// directories as needed, using the same permission heuristic as the rest of
// the scaffold/upgrade write paths.
//
// This is the write path for two of the four v2 exception faces that
// bypass ApplyOps: the settings.json 3-way merge and the settings.ralph.json
// snapshot (see v2SkipPaths' doc comment for the full set — the two block
// surfaces, AGENTS.md and .gitignore, write via updateOneBlockV2 instead,
// which carries its own leaf Lstat guard and whose root-level targets have
// no parent chain to validate). Because these two writes never
// go through ApplyOps' preflight, writeFileV2 applies the same containment
// checks itself: upgrade.ValidateRealParentChain against every existing
// parent path component, plus an Lstat of the leaf that rejects anything
// other than a regular file or an absent entry (a symlink or other
// non-regular file at the target is refused, mirroring ApplyOps' leaf
// check). Both checks run, and any write, only after they both pass.
func writeFileV2(targetDir, relPath string, content []byte) error {
	if err := upgrade.ValidateRealParentChain(targetDir, relPath); err != nil {
		return fmt.Errorf("write %s: %w", relPath, err)
	}

	full := filepath.Join(targetDir, filepath.FromSlash(relPath))
	fi, err := os.Lstat(full)
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("write %s: lstat: %w", relPath, err)
		}
	} else if !fi.Mode().IsRegular() {
		return fmt.Errorf("write %s: refusing to operate on non-regular file (mode %s)", relPath, fi.Mode())
	}

	if err := os.MkdirAll(filepath.Dir(full), 0755); err != nil {
		return err
	}
	return os.WriteFile(full, content, scaffold.FilePerm(relPath))
}

func manifestRefreshPaths(entries []upgrade.ManifestRefreshEntry) []string {
	if len(entries) == 0 {
		return nil
	}
	paths := make([]string, len(entries))
	for i, e := range entries {
		paths[i] = e.Path
	}
	return paths
}

func sortedDriftV2(entries []upgrade.DriftEntry) []upgrade.DriftEntry {
	sorted := make([]upgrade.DriftEntry, len(entries))
	copy(sorted, entries)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].Path < sorted[j].Path })
	return sorted
}

// writeDriftGuidanceV2 prints the sorted drift list (annotating untracked
// entries) followed by the resolution guidance. Single shared body for the
// three stderr sites (real-run error path, no-op tail, --dry-run preview) so
// the wording cannot diverge (resolves the tech-debt register's C2-L4 drift-guidance duplication item; the
// markdown report keeps its own formatting in internal/upgrade/report.go).
//
// Untracked drift (RecordedHash == "": a new template core path colliding
// with a pre-existing local file that has no manifest entry) cannot be
// resolved by eject/adopt — both reject untracked paths by design (AC-3) —
// so the guidance must not send those users in a circle (AR#1, cycle 3,
// docs/reports/cross-review-triage-overlay-scaffold-v2-p5.md).
func writeDriftGuidanceV2(errOut io.Writer, drift []upgrade.DriftEntry, heading string) {
	writef(errOut, "\nUnresolved drift (%s):\n", heading)
	hasUntracked := false
	for _, d := range sortedDriftV2(drift) {
		if d.RecordedHash == "" {
			hasUntracked = true
			writef(errOut, "  ⚠ %s (untracked)\n", d.Path)
			continue
		}
		writef(errOut, "  ⚠ %s\n", d.Path)
	}
	writef(errOut, "\nResolve with `ralph eject <path>` (keep the local change, tracked as a fork) or `ralph adopt <path>` (discard the local change, revert to template).\n")
	if hasUntracked {
		writef(errOut, "Paths marked (untracked) are new template files colliding with pre-existing local files that have no manifest entry; `ralph eject`/`ralph adopt` cannot resolve them — move the local file aside (or merge its content manually), then re-run `ralph upgrade`.\n")
	}
}

// renderUpgradeV2Preview renders the --dry-run (and --diff) preview for a v2
// upgrade: op counts, per-op paths, drift, and preserved-pack namespaces.
// --diff additionally renders the full advisory diffs (fork/seed paths whose
// template side changed). Never writes to targetDir.
func renderUpgradeV2Preview(plan upgrade.ReplacePlan, desired map[string][]byte, absDir, version string, out, errOut io.Writer, colorize bool, opts upgradeOptions, oldOwned, oldOwnedSnapshot []byte, snapshotFound bool) error {
	var deletes, creates, updates int
	for _, op := range plan.Ops {
		switch op.Kind {
		case upgrade.OpDelete:
			deletes++
		case upgrade.OpCreate:
			creates++
		case upgrade.OpUpdate:
			updates++
		}
	}

	writef(out, "Upgrade preview (dry run, v2 layout)\n")
	writef(out, "  delete:            %d files\n", deletes)
	writef(out, "  create:            %d files\n", creates)
	writef(out, "  update:            %d files\n", updates)
	writef(out, "  manifest refresh:  %d files\n", len(plan.ManifestRefresh))
	writef(out, "  drift (untouched): %d files\n", len(plan.Drift))
	writef(out, "  advisories:        %d files\n", len(plan.Advisories))
	writef(out, "  preserved packs:   %d files\n", len(plan.Preserved))

	if len(plan.Ops) == 0 {
		writef(out, "\nNo core changes.\n")
	} else {
		writef(out, "\nFiles:\n")
		for _, op := range plan.Ops {
			writef(out, "  %-8s %s\n", op.Kind.String(), op.Path)
		}
	}

	if len(plan.Drift) > 0 {
		writeDriftGuidanceV2(errOut, plan.Drift, "would be left untouched")
	}

	// AR#1 (cycle-2 cross-review, docs/reports/cross-review-triage-overlay-scaffold-v2-p3.md):
	// the core replace plan above never covers the four v2 "exception
	// faces" (they are excluded via v2SkipPaths and handled by their own
	// mechanisms), so a preview that only reported plan.Ops could show "No
	// core changes" while a real run would still rewrite AGENTS.md,
	// .gitignore, settings.json, or the settings snapshot. Compute each
	// exception face's pending outcome purely in memory — write=false below
	// means applyBlockUpdatesV2 and MergeOwnedSettings never touch disk —
	// and name it here so --dry-run stays a complete preview.
	if err := renderUpgradeV2ExceptionFacePreview(desired, absDir, oldOwned, oldOwnedSnapshot, snapshotFound, out, errOut); err != nil {
		return err
	}

	if !opts.DiffPreview {
		return nil
	}

	diffs, err := upgrade.RenderAdvisoryDiffs(absDir, desiredFS(desired), version, plan.Advisories)
	if err != nil {
		return fmt.Errorf("rendering advisory diffs: %w", err)
	}
	for _, d := range diffs {
		writef(out, "\n--- %s (owner: %s) ---\n", d.Path, d.Owner)
		if d.Skipped {
			writef(out, "  (%s)\n", d.Note)
			continue
		}
		if d.Diff == "" {
			writef(out, "  (no differences)\n")
			continue
		}
		diffText := d.Diff
		if colorize {
			diffText = upgrade.Colorize(diffText)
		}
		writeDiffOutput(diffText, out, errOut, opts.Pager)
	}
	return nil
}

// renderUpgradeV2ExceptionFacePreview computes and prints the pending
// outcome of all four v2 exception faces (AGENTS.md, .gitignore,
// settings.json, the settings snapshot) without writing anything to disk.
// applyBlockUpdatesV2 is called with write=false, and MergeOwnedSettings is
// a pure function — neither touches targetDir. See AR#1's (cycle 2) fix note
// on renderUpgradeV2Preview.
func renderUpgradeV2ExceptionFacePreview(desired map[string][]byte, absDir string, oldOwned, oldOwnedSnapshot []byte, snapshotFound bool, out, errOut io.Writer) error {
	blockOutcomes, _, err := applyBlockUpdatesV2(absDir, desired, errOut, false)
	if err != nil {
		return fmt.Errorf("computing managed block preview: %w", err)
	}

	currentSettings, err := readFinalDiskContent(absDir, v2SettingsPath)
	if err != nil {
		return fmt.Errorf("reading %s: %w", v2SettingsPath, err)
	}
	newOwnedSettings := desired[v2SettingsPath]
	mergeResult, err := upgrade.MergeOwnedSettings(currentSettings, oldOwned, newOwnedSettings)
	if err != nil {
		return fmt.Errorf("merging %s: %w", v2SettingsPath, err)
	}
	snapshotWouldChange := !snapshotFound || !bytes.Equal(oldOwnedSnapshot, newOwnedSettings)

	writef(out, "\nException faces (block/settings, not counted above):\n")
	for _, bs := range blockSurfaces {
		writef(out, "  block %-14s %s\n", bs.path+":", blockPreviewLabel(blockOutcomes[bs.path]))
	}
	settingsLabel := "unchanged"
	if mergeResult.Changed {
		settingsLabel = "merge pending"
	}
	writef(out, "  %-19s %s\n", v2SettingsPath+":", settingsLabel)
	snapshotLabel := "unchanged"
	if snapshotWouldChange {
		snapshotLabel = "update pending"
		if !snapshotFound {
			snapshotLabel = "update pending (no existing snapshot; will be created)"
		}
	}
	writef(out, "  %-19s %s\n", "settings snapshot:", snapshotLabel)

	return nil
}

// blockPreviewLabel summarizes a computed block-update outcome for the
// --dry-run preview. A malformed existing block is reported distinctly from
// "unchanged" since it is a warning condition, even though neither state
// would trigger a write.
func blockPreviewLabel(oc blockUpdateOutcome) string {
	switch {
	case oc.ok && (oc.outcome == upgrade.BlockUpdated || oc.outcome == upgrade.BlockAppended):
		return "update pending"
	case oc.outcome == upgrade.BlockMalformed:
		return "left untouched (malformed)"
	default:
		return "unchanged"
	}
}

// desiredFS adapts a desired-state path→content map into an fs.FS, so it can
// be passed to internal/upgrade functions written against fs.FS (e.g.
// RenderAdvisoryDiffs) without those functions needing to know about the v2
// desired-state map type. Read-only; every path is already fully
// materialized in memory.
type desiredFS map[string][]byte

func (d desiredFS) Open(name string) (fs.File, error) {
	data, ok := d[name]
	if !ok {
		return nil, &fs.PathError{Op: "open", Path: name, Err: fs.ErrNotExist}
	}
	return &desiredFile{name: name, Reader: bytes.NewReader(data), size: int64(len(data))}, nil
}

func (d desiredFS) ReadFile(name string) ([]byte, error) {
	data, ok := d[name]
	if !ok {
		return nil, &fs.PathError{Op: "readfile", Path: name, Err: fs.ErrNotExist}
	}
	return data, nil
}

type desiredFile struct {
	*bytes.Reader
	name string
	size int64
}

func (f *desiredFile) Stat() (fs.FileInfo, error) { return desiredFileInfo{f.name, f.size}, nil }
func (f *desiredFile) Close() error               { return nil }

type desiredFileInfo struct {
	name string
	size int64
}

func (i desiredFileInfo) Name() string       { return filepath.Base(i.name) }
func (i desiredFileInfo) Size() int64        { return i.size }
func (i desiredFileInfo) Mode() fs.FileMode  { return 0444 }
func (i desiredFileInfo) ModTime() time.Time { return time.Time{} }
func (i desiredFileInfo) IsDir() bool        { return false }
func (i desiredFileInfo) Sys() any           { return nil }
