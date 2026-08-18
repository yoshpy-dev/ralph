package cli

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"github.com/yoshpy-dev/ralph/internal/scaffold"
)

// This file resolves the plan's Open question ("移行分類器の配置" —
// docs/plans/active/2026-08-18-overlay-scaffold-v2-p4.md) in favor of
// internal/cli: the classifier needs both internal/cli's ownerForScaffoldPath
// (to assign the v3 ownership attribute a migrated path should carry) and
// internal/scaffold (manifest types, hashing). internal/upgrade cannot depend
// on internal/cli's path->owner mapping without inverting that package
// boundary (see ReplaceOptions.OwnerForPath's doc comment in
// internal/upgrade/replaceplan.go), so the classifier lives here instead,
// alongside the CLI-layer legacy-detection code in upgrade.go that will call
// it (slice 3).
//
// ClassifyMigration and RenderMigrationPreview are pure: no file writes,
// only reads (manifest paths on disk, to compare recorded hashes against
// current content). Executing the resulting MigrationPlan is slice 3's job.

// pathClaudeMD, pathAgentsMD, pathGitignore, pathSettings, and
// pathCodexOverride are the "special face" paths (spec FR-8) that get
// migration treatment different from the generic ownership-relocation
// rules below.
const (
	pathClaudeMD      = "CLAUDE.md"
	pathAgentsMD      = "AGENTS.md"
	pathGitignore     = ".gitignore"
	pathSettings      = ".claude/settings.json"
	pathCodexOverride = ".codex/AGENTS.override.md"
	pathBaselineDir   = ".ralph/baseline"
)

// legacyStatePartial is the ManifestFile.State value pre-Phase-3 `ralph`
// versions wrote when the (now-removed) interactive conflict resolution
// flow's "keep local file" outcome was chosen: the file is managed, but its
// content is a user edit the tool deliberately preserved rather than the
// template's. No named constant exists for it in scaffold/manifest.go
// (nothing in the current codebase writes it anymore — see that file's
// State field doc), so it is defined locally here, at the one remaining call
// site that needs to recognize it.
const legacyStatePartial = "partial"

// legacyRalphHookCommands lists the direct-invocation hook commands every
// pre-dispatcher `ralph` template generation shipped in .claude/settings.json
// (superseded by the fan-out dispatcher added in commit 24a611a,
// ./.claude/hooks/ralph-dispatch.sh). A migrated settings.json whose content
// has been modified by the user must have these entries pruned before
// handing off to the v2 3-way settings merge (settingsmerge.go): otherwise
// the direct-invocation commands survive the migration side by side with the
// dispatcher, and every event fires the underlying hook script twice.
//
// This list is fixed to the 8 scripts that were ever wired as direct
// top-level hook commands (excludes ralph-dispatch.sh itself, lib_json.sh,
// and every *.d/ drop-in — none of those were ever referenced directly from
// settings.json). Kept sorted for deterministic PrunedHookCommands output.
var legacyRalphHookCommands = []string{
	"./.claude/hooks/check_mojibake.sh",
	"./.claude/hooks/post_edit_verify.sh",
	"./.claude/hooks/post_tool_failure_feedback.sh",
	"./.claude/hooks/pre_bash_guard.sh",
	"./.claude/hooks/precompact_checkpoint.sh",
	"./.claude/hooks/prompt_gate.sh",
	"./.claude/hooks/session_end_summary.sh",
	"./.claude/hooks/session_start_context.sh",
}

// LegacyEntryState is the outcome of comparing a legacy (v1/v2) manifest
// entry against current disk content — the "LegacyEntryState contract"
// (plan Scope B, Codex advisory 4).
type LegacyEntryState int

const (
	// LegacyUnmodified means disk content matches the entry's recorded
	// hash: the file has not changed since the last legacy ralph
	// operation touched it.
	LegacyUnmodified LegacyEntryState = iota
	// LegacyModified means disk content diverges from the recorded hash,
	// or the comparison basis was ambiguous enough that treating it as
	// modified is the safer default (see classifyLegacyEntryState).
	LegacyModified
	// LegacyUnmanaged means the legacy entry recorded managed=false (the
	// old interactive "skip" outcome): the file was never ralph's to
	// touch, so migration inherits it as a fork unconditionally.
	LegacyUnmanaged
)

// classifyLegacyEntryState implements the LegacyEntryState contract exactly
// as documented in the plan:
//
//   - managed=false (legacy skip)             -> LegacyUnmanaged,
//     unconditionally — this is checked before anything else, since an
//     unmanaged entry never had ralph-owned content to compare hashes for.
//   - state="partial" (legacy "keep local" outcome) -> LegacyModified,
//     unconditionally — the tool already recorded this content as a
//     deliberate user edit; there is no hash comparison that could turn it
//     back into "unmodified".
//   - otherwise, compare disk_hash (falling back to hash, for v1
//     compatibility) against the current disk content hash:
//   - comparison hash empty (a legacy "heal" target, i.e. a v1 entry that
//     never recorded a hash) -> LegacyModified. A mis-fork (keeping a
//     file that happened to be genuinely unmodified) is recoverable
//     later; a mis-delete is not.
//   - disk file absent -> LegacyModified, for the same reason: there is
//     nothing to compare against, so treat it like an edit rather than
//     silently dropping the path's migration record. (Top-level caller
//     ClassifyMigration handles the one case where "absent" instead
//     means "a prior interrupted migration already relocated this path"
//     — see its rerun-stability pre-check — before this function is
//     ever consulted for that path.)
//   - hashes equal -> LegacyUnmodified.
//   - hashes differ -> LegacyModified.
func classifyLegacyEntryState(entry scaffold.ManifestFile, diskHash string, hasDisk bool) LegacyEntryState {
	if !entry.Managed {
		return LegacyUnmanaged
	}
	if entry.State == legacyStatePartial {
		return LegacyModified
	}
	comparisonHash := entry.DiskHash
	if comparisonHash == "" {
		comparisonHash = entry.Hash
	}
	if comparisonHash == "" || !hasDisk {
		return LegacyModified
	}
	if diskHash == comparisonHash {
		return LegacyUnmodified
	}
	return LegacyModified
}

// MigrationOpKind identifies the treatment ClassifyMigration decided for a
// single legacy path.
type MigrationOpKind int

const (
	// OpDeleteOldPath deletes the old path outright: either it was
	// unmodified and relocated (the chained v2 upgrade creates the new
	// path), unmodified and retired from the template set, or a
	// relocation whose destination already resolved the move (rerun
	// stability / collision-matrix case (a)/(b)).
	OpDeleteOldPath MigrationOpKind = iota
	// OpKeepInPlace leaves an unmodified, non-relocated path untouched;
	// the chained v2 upgrade converges its content in the ordinary way.
	OpKeepInPlace
	// OpForkRelocate moves modified (or unmanaged) content to its new
	// (relocated) path, recorded as a fork.
	OpForkRelocate
	// OpForkInPlace records modified (or unmanaged) content as a fork at
	// its existing path; the file itself is never written.
	OpForkInPlace
	// OpReplaceWithTemplate replaces an unmodified special-face path
	// (CLAUDE.md, AGENTS.md, .gitignore, settings.json) with the v2
	// template content.
	OpReplaceWithTemplate
	// OpUntouched leaves a path completely alone: a modified special
	// face (CLAUDE.md, AGENTS.md, .gitignore), the always-untouched
	// .codex/AGENTS.override.md re-attribution, or a managed entry
	// recorded in the manifest but absent from disk.
	OpUntouched
	// OpSettingsPrune marks a modified settings.json for legacy
	// direct-invocation hook pruning before the v2 3-way settings merge
	// runs.
	OpSettingsPrune
	// OpDeleteDir marks a leftover directory (only .ralph/baseline/
	// today) for removal.
	OpDeleteDir
)

// MigrationEntry is a single classified path in a MigrationPlan.
type MigrationEntry struct {
	// OldPath is always set: the legacy manifest path (or, for
	// OpDeleteDir, the directory being removed).
	OldPath string
	// NewPath is set when migration keeps or moves content somewhere:
	// equal to OldPath for same-path outcomes (KeepInPlace,
	// ForkInPlace, ReplaceWithTemplate, Untouched special faces), the
	// relocation target for ForkRelocate and a relocated DeleteOldPath,
	// and empty for a retired DeleteOldPath / OpDeleteDir (nothing left
	// to point at).
	NewPath string
	Kind    MigrationOpKind
	State   LegacyEntryState
	// Owner is the v3 ownership attribute (scaffold.OwnerCore/Fork/Seed/
	// Block) this path should carry once the migration's v3 manifest is
	// written (slice 3). Empty when the path disappears entirely
	// (OpDeleteOldPath, OpDeleteDir) — there is nothing left to own.
	Owner string
	// ForkedFromVersion is set alongside Owner=scaffold.OwnerFork
	// (OpForkRelocate, OpForkInPlace): the legacy manifest's recorded
	// template version, carried into the v3 fork record.
	ForkedFromVersion string
	// PrunedHookCommands is populated only for OpSettingsPrune: the
	// subset of legacyRalphHookCommands actually found referenced in the
	// current settings.json content, sorted (legacyRalphHookCommands is
	// itself sorted, so this stays sorted by construction).
	PrunedHookCommands []string
	// SnapshotCreate is set on the settings.json OpReplaceWithTemplate
	// entry: besides replacing the file, a fresh settings snapshot
	// (.ralph/core/settings.ralph.json) must be (re)created so the
	// result matches a clean v2 init.
	SnapshotCreate bool
	// Reason is a short human-readable explanation, surfaced in the
	// migration preview and report.
	Reason string
}

// MigrationCollision records a relocation whose destination already exists
// with content that diverges from both the source and (when relevant) the
// new template — collision-matrix case (c). Slice 3 aborts with zero writes
// whenever MigrationPlan.Collisions is non-empty, listing these for the
// operator to resolve manually before re-running.
type MigrationCollision struct {
	OldPath string
	NewPath string
	Reason  string
}

// MigrationPlan is the pure output of ClassifyMigration.
type MigrationPlan struct {
	// Entries is sorted by OldPath.
	Entries []MigrationEntry
	// Collisions is sorted by OldPath.
	Collisions []MigrationCollision
	// Packs carries over the legacy manifest's installed pack list
	// (meta.packs), sorted, so slice 3 can continue writing it into the
	// v3 manifest (AC-11's "Meta.Packs が v3 manifest に継承される").
	Packs []string
}

// ClassifyMigration builds a MigrationPlan from a legacy (v1/v2) manifest, the
// current on-disk content under targetDir, and the desired v2 state (the same
// template-relative-path -> content shape internal/upgrade.PlanCoreReplace
// uses), which the caller assembles to include the base template plus every
// pack in legacyManifest.Meta.Packs.
//
// ClassifyMigration only reads targetDir; it never writes. Every legacy
// manifest path is classified independently (see classifyLegacyPath); the
// .ralph/baseline/ directory left behind by pre-Phase-3 ralph versions (not a
// manifest entry — see internal/cli/upgrade_v2.go's
// removeLegacyBaselineIfPresent) is detected separately and always marked
// for removal when present.
func ClassifyMigration(legacyManifest *scaffold.Manifest, targetDir string, desired map[string][]byte) (MigrationPlan, error) {
	if legacyManifest == nil {
		return MigrationPlan{}, errors.New("nil legacy manifest")
	}

	var plan MigrationPlan
	if len(legacyManifest.Meta.Packs) > 0 {
		plan.Packs = append([]string(nil), legacyManifest.Meta.Packs...)
		sort.Strings(plan.Packs)
	}

	paths := make([]string, 0, len(legacyManifest.Files))
	for p := range legacyManifest.Files {
		paths = append(paths, p)
	}
	sort.Strings(paths)

	for _, oldPath := range paths {
		entry := legacyManifest.Files[oldPath]

		cleanOld, err := cleanMigrationPath(oldPath)
		if err != nil {
			return MigrationPlan{}, fmt.Errorf("legacy manifest path %q: %w", oldPath, err)
		}

		diskContent, hasDisk, err := readDiskFileForMigration(targetDir, cleanOld)
		if err != nil {
			return MigrationPlan{}, fmt.Errorf("reading %s: %w", cleanOld, err)
		}

		newPath, relocated := relocatedRulePath(cleanOld, desired)
		if relocated && !hasDisk {
			// Rerun stability: a prior interrupted migration already
			// finished moving this path (or the operator/tooling did),
			// so the old path is gone. Nothing remains to plan for it —
			// see classifyLegacyEntryState's doc for why this check must
			// happen before hash-based state classification, not inside
			// it.
			continue
		}

		var diskHash string
		if hasDisk {
			diskHash = scaffold.HashBytes(diskContent)
		}
		state := classifyLegacyEntryState(entry, diskHash, hasDisk)

		me, collision, err := classifyLegacyPath(legacyClassifyInput{
			oldPath:       cleanOld,
			newPath:       newPath,
			relocated:     relocated,
			state:         state,
			diskContent:   diskContent,
			hasDisk:       hasDisk,
			legacyVersion: legacyManifest.Meta.Version,
			targetDir:     targetDir,
			desired:       desired,
		})
		if err != nil {
			return MigrationPlan{}, err
		}
		if collision != nil {
			plan.Collisions = append(plan.Collisions, *collision)
			continue
		}
		plan.Entries = append(plan.Entries, me)
	}

	if hasBaselineDir(targetDir) {
		plan.Entries = append(plan.Entries, MigrationEntry{
			OldPath: pathBaselineDir,
			Kind:    OpDeleteDir,
			Reason:  "legacy baseline cache directory removed by migration (Phase 3 retired the baseline mechanism)",
		})
	}

	sort.Slice(plan.Entries, func(i, j int) bool { return plan.Entries[i].OldPath < plan.Entries[j].OldPath })
	sort.Slice(plan.Collisions, func(i, j int) bool { return plan.Collisions[i].OldPath < plan.Collisions[j].OldPath })

	return plan, nil
}

// legacyClassifyInput bundles a single legacy path's classification inputs.
type legacyClassifyInput struct {
	oldPath       string
	newPath       string
	relocated     bool
	state         LegacyEntryState
	diskContent   []byte
	hasDisk       bool
	legacyVersion string
	targetDir     string
	desired       map[string][]byte
}

// classifyLegacyPath applies the special-face rules (FR-8) first, then falls
// back to the generic ownership-relocation rules (FR-7).
func classifyLegacyPath(in legacyClassifyInput) (MigrationEntry, *MigrationCollision, error) {
	switch in.oldPath {
	case pathClaudeMD:
		return classifyClaudeMD(in.state), nil, nil
	case pathAgentsMD, pathGitignore:
		return classifyBlockFace(in.oldPath, in.state), nil, nil
	case pathSettings:
		return classifySettingsFace(in.state, in.diskContent), nil, nil
	case pathCodexOverride:
		return MigrationEntry{
			OldPath: in.oldPath,
			NewPath: in.oldPath,
			Kind:    OpUntouched,
			State:   in.state,
			Owner:   ownerForScaffoldPath(in.oldPath),
			Reason:  "codex override reattributed to owner=seed; content never touched by migration",
		}, nil, nil
	}

	if in.state == LegacyModified || in.state == LegacyUnmanaged {
		return classifyForkCandidate(in)
	}
	return classifyUnmodifiedGeneric(in)
}

// classifyClaudeMD implements FR-8's CLAUDE.md face: unmodified content is
// replaceable with the minimal v2 seed (ralph guidance now lives in
// .claude/rules/ralph, so the seed's content is sufficient either way);
// modified content is left completely untouched.
func classifyClaudeMD(state LegacyEntryState) MigrationEntry {
	owner := ownerForScaffoldPath(pathClaudeMD)
	if state == LegacyUnmodified {
		return MigrationEntry{
			OldPath: pathClaudeMD, NewPath: pathClaudeMD,
			Kind: OpReplaceWithTemplate, State: state, Owner: owner,
			Reason: "unmodified CLAUDE.md replaced with the minimal v2 seed",
		}
	}
	return MigrationEntry{
		OldPath: pathClaudeMD, NewPath: pathClaudeMD,
		Kind: OpUntouched, State: state, Owner: owner,
		Reason: "modified CLAUDE.md left byte-for-byte untouched; ralph guidance is served from .claude/rules/ralph regardless",
	}
}

// classifyBlockFace implements FR-8's AGENTS.md/.gitignore face: unmodified
// content is replaceable with the block-carrying v2 template; modified
// content is left in place for the chained v2 upgrade's block engine to
// append its managed block onto (block-outside content is fully preserved,
// but any legacy-template-derived duplicate content outside the block is not
// removed automatically — see the plan's Design decisions).
func classifyBlockFace(oldPath string, state LegacyEntryState) MigrationEntry {
	owner := ownerForScaffoldPath(oldPath)
	if state == LegacyUnmodified {
		return MigrationEntry{
			OldPath: oldPath, NewPath: oldPath,
			Kind: OpReplaceWithTemplate, State: state, Owner: owner,
			Reason: "unmodified; replaced with the block-carrying v2 template",
		}
	}
	return MigrationEntry{
		OldPath: oldPath, NewPath: oldPath,
		Kind: OpUntouched, State: state, Owner: owner,
		Reason: "modified content kept in place; the chained v2 upgrade's block engine appends its managed block without touching existing content, but legacy-template-derived duplicate content outside the block is not removed automatically — see the migration report for cleanup guidance",
	}
}

// classifySettingsFace implements FR-8's settings.json face (a
// migration-only step — Codex advisory 1): unmodified content is replaced
// wholesale (plus a fresh settings snapshot, matching a clean v2 init).
// Modified content is marked for legacy hook pruning before the v2 3-way
// settings merge runs, so direct-invocation hook commands from before the
// dispatcher (commit 24a611a) do not survive alongside it and double-fire.
func classifySettingsFace(state LegacyEntryState, diskContent []byte) MigrationEntry {
	owner := ownerForScaffoldPath(pathSettings)
	if state == LegacyUnmodified {
		return MigrationEntry{
			OldPath: pathSettings, NewPath: pathSettings,
			Kind: OpReplaceWithTemplate, State: state, Owner: owner,
			SnapshotCreate: true,
			Reason:         "unmodified settings.json replaced wholesale with the v2 template and a fresh settings snapshot, matching a clean v2 init",
		}
	}
	return MigrationEntry{
		OldPath: pathSettings, NewPath: pathSettings,
		Kind: OpSettingsPrune, State: state, Owner: owner,
		PrunedHookCommands: prunedLegacyHookCommands(diskContent),
		Reason:             "modified settings.json: known legacy direct-invocation hook commands are pruned before handing off to the v2 3-way settings merge, to avoid double-firing hooks alongside the dispatcher",
	}
}

// prunedLegacyHookCommands returns the subset of legacyRalphHookCommands
// referenced in diskContent, preserving legacyRalphHookCommands' sorted
// order.
func prunedLegacyHookCommands(diskContent []byte) []string {
	var found []string
	for _, cmd := range legacyRalphHookCommands {
		if bytes.Contains(diskContent, []byte(cmd)) {
			found = append(found, cmd)
		}
	}
	return found
}

// classifyUnmodifiedGeneric handles a non-special-face path whose
// LegacyEntryState is LegacyUnmodified: relocate-and-delete, retire, or keep
// in place, per FR-7.
func classifyUnmodifiedGeneric(in legacyClassifyInput) (MigrationEntry, *MigrationCollision, error) {
	if !in.relocated {
		if _, ok := in.desired[in.oldPath]; ok {
			return MigrationEntry{
				OldPath: in.oldPath, NewPath: in.oldPath,
				Kind: OpKeepInPlace, State: LegacyUnmodified,
				Owner:  ownerForScaffoldPath(in.oldPath),
				Reason: "unmodified and path unchanged; content converges via the chained v2 upgrade",
			}, nil, nil
		}
		return MigrationEntry{
			OldPath: in.oldPath,
			Kind:    OpDeleteOldPath, State: LegacyUnmodified,
			Reason: "unmodified and retired from the v2 template set",
		}, nil, nil
	}

	sourceHash := scaffold.HashBytes(in.diskContent)
	destContent, destExists, err := readDiskFileForMigration(in.targetDir, in.newPath)
	if err != nil {
		return MigrationEntry{}, nil, err
	}
	kind, collisionReason := relocationOutcome(sourceHash, true, destContent, destExists, in.newPath, in.desired)
	if collisionReason != "" {
		return MigrationEntry{}, &MigrationCollision{OldPath: in.oldPath, NewPath: in.newPath, Reason: collisionReason}, nil
	}
	reason := "unmodified and relocated to " + in.newPath + "; the chained v2 upgrade creates the new path"
	if destExists {
		reason = "unmodified; relocation destination " + in.newPath + " already resolved (rerun), old path deleted"
	}
	return MigrationEntry{
		OldPath: in.oldPath, NewPath: in.newPath,
		Kind: kind, State: LegacyUnmodified,
		Reason: reason,
	}, nil, nil
}

// classifyForkCandidate handles a non-special-face path whose
// LegacyEntryState is LegacyModified or LegacyUnmanaged: fork in place, or
// fork-relocate, per FR-7.
func classifyForkCandidate(in legacyClassifyInput) (MigrationEntry, *MigrationCollision, error) {
	if !in.hasDisk {
		// Recorded in the legacy manifest (managed+modified, or
		// managed=false/skip) but the file is simply gone from disk, and
		// it is not a relocatable path (the relocated-and-absent case is
		// handled by ClassifyMigration's rerun-stability pre-check before
		// this function is reached). There is nothing to fork or move.
		return MigrationEntry{
			OldPath: in.oldPath, NewPath: in.oldPath,
			Kind: OpUntouched, State: in.state,
			Reason: "recorded in the legacy manifest but absent from disk; nothing to fork or move",
		}, nil, nil
	}

	if !in.relocated {
		return MigrationEntry{
			OldPath: in.oldPath, NewPath: in.oldPath,
			Kind: OpForkInPlace, State: in.state,
			Owner: scaffold.OwnerFork, ForkedFromVersion: in.legacyVersion,
			Reason: "modified content recorded as a fork in place; the chained v2 upgrade surfaces ongoing advisories against the new core content",
		}, nil, nil
	}

	sourceHash := scaffold.HashBytes(in.diskContent)
	destContent, destExists, err := readDiskFileForMigration(in.targetDir, in.newPath)
	if err != nil {
		return MigrationEntry{}, nil, err
	}
	kind, collisionReason := relocationOutcome(sourceHash, false, destContent, destExists, in.newPath, in.desired)
	if collisionReason != "" {
		return MigrationEntry{}, &MigrationCollision{OldPath: in.oldPath, NewPath: in.newPath, Reason: collisionReason}, nil
	}

	entry := MigrationEntry{OldPath: in.oldPath, NewPath: in.newPath, Kind: kind, State: in.state}
	if kind == OpForkRelocate {
		entry.Owner = scaffold.OwnerFork
		entry.ForkedFromVersion = in.legacyVersion
		entry.Reason = "modified content relocated to " + in.newPath + " as a fork"
	} else {
		entry.Reason = "already relocated: destination content matches the source; only the old path is deleted"
	}
	return entry, nil, nil
}

// relocationOutcome resolves the relocation-destination collision matrix
// (Codex advisory 3): given a source that is about to move to newPath, and
// newPath's current disk state, decide the outcome.
//
//   - dest absent: unmodified sources delete the old path outright (the
//     chained v2 upgrade creates newPath); modified/unmanaged sources fork-
//     relocate.
//   - (a) dest exists, content == source: already relocated (by a prior
//     interrupted migration run, or otherwise) — delete the old path only.
//   - (b) dest exists, content == the new template, AND the source is
//     unmodified: the destination already holds exactly what a fresh v2
//     upgrade would have written there anyway — delete the old path only.
//     This does not apply to a modified/unmanaged source: silently
//     discarding the user's fork content just because the destination
//     happens to match the template would lose that content, so it falls
//     through to the collision case below instead.
//   - (c) otherwise (divergent): collision — the caller must abort with
//     zero writes and report it.
func relocationOutcome(sourceHash string, sourceUnmodified bool, destContent []byte, destExists bool, newPath string, desired map[string][]byte) (kind MigrationOpKind, collisionReason string) {
	if !destExists {
		if sourceUnmodified {
			return OpDeleteOldPath, ""
		}
		return OpForkRelocate, ""
	}
	destHash := scaffold.HashBytes(destContent)
	if destHash == sourceHash {
		return OpDeleteOldPath, ""
	}
	if sourceUnmodified {
		if templateContent, ok := desired[newPath]; ok && destHash == scaffold.HashBytes(templateContent) {
			return OpDeleteOldPath, ""
		}
	}
	return 0, "relocation destination " + newPath + " exists with content diverging from the source (and the new template, if unmodified); manual resolution required before migration can proceed"
}

// relocatedRulePath reports whether oldPath is a shipped-rule or pack-rule
// path from before the .claude/rules/ralph/ layout: a path directly under
// .claude/rules/ (not already nested under ralph/) whose basename matches a
// file the v2 desired state ships under .claude/rules/ralph/.
//
// desired is built by the caller to include every installed pack's rule
// file (internal/cli/language_pack.go's packRuleRelPath renders every pack
// rule to exactly this new location), so this single membership check
// mechanically covers both shipped core rules and pack rules without a
// second, separate "is this an installed pack name" lookup — anything the
// migrated project should still have under .claude/rules/ralph/ after
// migration is, by construction, already a key in desired.
func relocatedRulePath(oldPath string, desired map[string][]byte) (string, bool) {
	if path.Dir(oldPath) != ".claude/rules" {
		return "", false
	}
	newPath := path.Join(".claude/rules/ralph", path.Base(oldPath))
	if _, ok := desired[newPath]; ok {
		return newPath, true
	}
	return "", false
}

// RenderMigrationPreview renders plan as a deterministic, grouped preview:
// one section per MigrationOpKind (path counts + old->new listings), then
// collisions. Used by `ralph upgrade`'s pre-migration confirmation prompt
// (slice 3) and the migration report.
func RenderMigrationPreview(plan MigrationPlan) string {
	var b strings.Builder
	b.WriteString("Legacy -> v2 migration preview\n")

	groups := []struct {
		title string
		kind  MigrationOpKind
	}{
		{"Delete (relocated/retired)", OpDeleteOldPath},
		{"Keep in place", OpKeepInPlace},
		{"Fork - relocate", OpForkRelocate},
		{"Fork - in place", OpForkInPlace},
		{"Replace with template", OpReplaceWithTemplate},
		{"Untouched", OpUntouched},
		{"Settings prune", OpSettingsPrune},
		{"Delete directory", OpDeleteDir},
	}

	for _, g := range groups {
		entries := entriesForKind(plan.Entries, g.kind)
		fmt.Fprintf(&b, "\n%s (%d)\n", g.title, len(entries))
		for _, e := range entries {
			if e.NewPath != "" && e.NewPath != e.OldPath {
				fmt.Fprintf(&b, "  %s -> %s\n", e.OldPath, e.NewPath)
			} else {
				fmt.Fprintf(&b, "  %s\n", e.OldPath)
			}
		}
	}

	fmt.Fprintf(&b, "\nCollisions (%d)\n", len(plan.Collisions))
	for _, c := range plan.Collisions {
		fmt.Fprintf(&b, "  %s -> %s: %s\n", c.OldPath, c.NewPath, c.Reason)
	}

	return b.String()
}

// entriesForKind filters entries by kind, preserving their relative order
// (ClassifyMigration returns plan.Entries already sorted by OldPath, so this
// stays deterministic).
func entriesForKind(entries []MigrationEntry, kind MigrationOpKind) []MigrationEntry {
	var out []MigrationEntry
	for _, e := range entries {
		if e.Kind == kind {
			out = append(out, e)
		}
	}
	return out
}

// cleanMigrationPath validates and slash-normalizes a legacy manifest path,
// mirroring internal/upgrade's cleanPathKey (unexported there, so this
// package needs its own copy rather than a cross-package call).
func cleanMigrationPath(raw string) (string, error) {
	clean, err := scaffold.CleanLocalRelPath(raw)
	if err != nil {
		return "", err
	}
	return filepath.ToSlash(clean), nil
}

// readDiskFileForMigration reads targetDir/relPath, reporting
// hasDisk=false (with no error) when the file does not exist.
func readDiskFileForMigration(targetDir, relPath string) (content []byte, hasDisk bool, err error) {
	full := filepath.Join(targetDir, filepath.FromSlash(relPath))
	data, err := os.ReadFile(full)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, false, nil
		}
		return nil, false, err
	}
	return data, true, nil
}

// hasBaselineDir reports whether targetDir/.ralph/baseline exists as a
// directory.
func hasBaselineDir(targetDir string) bool {
	fi, err := os.Stat(filepath.Join(targetDir, filepath.FromSlash(pathBaselineDir)))
	return err == nil && fi.IsDir()
}
