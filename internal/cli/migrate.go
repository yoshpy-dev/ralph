package cli

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/yoshpy-dev/ralph/internal/scaffold"
	"github.com/yoshpy-dev/ralph/internal/upgrade"
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
	// path), unmodified and retired from the template set, or an
	// unmodified relocation whose destination already resolved the move
	// (rerun stability / collision-matrix case (a)/(b)).
	OpDeleteOldPath MigrationOpKind = iota
	// OpDeleteOldPathAdoptFork deletes the old path (a no-op when it is
	// already gone) and records the relocation destination as a fork
	// (Owner=scaffold.OwnerFork + ForkedFromVersion): a modified or
	// unmanaged source whose relocation destination already holds that
	// same content -- either because the live collision-matrix check
	// found the destination already resolved (case (a) with a modified
	// source), or because a rerun's pre-check found the old path already
	// gone and the destination diverging from the current template.
	// Distinct from OpDeleteOldPath, which never carries Owner/
	// ForkedFromVersion, so the two stay easy to tell apart by kind
	// alone (self-review HIGH-1,
	// docs/reports/self-review-2026-08-18-overlay-scaffold-v2-p4.md).
	OpDeleteOldPathAdoptFork
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
	// PrunedHookCommands is populated only for OpSettingsPrune. Before
	// execution (classification time) it holds the candidate list --
	// every legacyRalphHookCommands entry found as a substring of the
	// current settings.json content -- used to seed the exact-match
	// removal set executeMigrationEntries applies. executeMigrationEntries
	// overwrites it in place with the commands actually removed (an
	// exact-match subset of the candidates: an argument-carrying variant
	// like "./.claude/hooks/pre_bash_guard.sh --verbose" is a candidate
	// but is never actually removed -- see PrunedHookNearMisses), so by
	// the time renderMigrationReport reads plan.Entries the report
	// reflects reality rather than the looser candidate list (self-review
	// MEDIUM-4).
	PrunedHookCommands []string
	// PrunedHookNearMisses is populated only for OpSettingsPrune, by
	// executeMigrationEntries: legacy hook commands left in place because
	// they reference a known script but do not match it exactly (an
	// argument-carrying variant). These are deliberately treated as user
	// customizations, not pruned; renderMigrationReport surfaces them so
	// an operator does not mistake survival for an oversight.
	PrunedHookNearMisses []string
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
			// so the old path is gone. Classify the relocation
			// destination from the legacy entry's recorded state rather
			// than dropping the entry outright -- see
			// classifyRerunRelocatedDestination's doc for why a plain
			// "nothing left to plan" skip loses fork attribution for a
			// modified source (self-review HIGH-1,
			// docs/reports/self-review-2026-08-18-overlay-scaffold-v2-p4.md).
			me, err := classifyRerunRelocatedDestination(entry, cleanOld, newPath, targetDir, legacyManifest.Meta.Version, desired)
			if err != nil {
				return MigrationPlan{}, fmt.Errorf("classifying rerun-relocated %s: %w", cleanOld, err)
			}
			if me != nil {
				plan.Entries = append(plan.Entries, *me)
			}
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

// classifyRerunRelocatedDestination handles ClassifyMigration's
// rerun-stability pre-check for a relocatable path whose old location is
// already gone from disk. Two legitimate causes reach here: (1) a normal
// unmodified relocation whose old path migration already deleted, but the
// chained v2 upgrade that creates the new path has not run yet this pass
// (destination absent); (2) a prior migration/upgrade run relocated a
// modified source (write destination, then delete old path -- see
// relocateMigrationFile) but a later entry's write failed before the
// manifest barrier committed, so a rerun sees the old path already gone
// and the destination already holding the (possibly forked) content.
//
// Returns nil, nil when there is nothing to record: either the destination
// does not exist yet (case 1 -- the chained v2 upgrade has not created it,
// so there is nothing pending for this path), or the destination already
// holds exactly the current template's content (an ordinary converged
// relocation -- buildMigratedManifest's generic desired-state sweep
// already records this correctly as owner=core without any entry here).
//
// Otherwise the destination's content diverges from the template, so --
// per the LegacyEntryState contract's "a mis-fork is recoverable, a
// mis-delete is not" philosophy (see classifyLegacyEntryState's doc) -- it
// is classified from the legacy entry's recorded state (via
// classifyLegacyEntryState with hasDisk=false, which always yields
// LegacyUnmanaged or LegacyModified, never LegacyUnmodified) rather than
// silently falling through to owner=core with a phantom disk hash. This
// closes self-review HIGH-1 item (1)
// (docs/reports/self-review-2026-08-18-overlay-scaffold-v2-p4.md).
func classifyRerunRelocatedDestination(entry scaffold.ManifestFile, oldPath, newPath, targetDir, legacyVersion string, desired map[string][]byte) (*MigrationEntry, error) {
	destContent, destExists, err := readDiskFileForMigration(targetDir, newPath)
	if err != nil {
		return nil, err
	}
	if !destExists {
		return nil, nil
	}
	if templateContent, ok := desired[newPath]; ok && scaffold.HashBytes(destContent) == scaffold.HashBytes(templateContent) {
		return nil, nil
	}
	return &MigrationEntry{
		OldPath: oldPath, NewPath: newPath,
		Kind:  OpDeleteOldPathAdoptFork,
		State: classifyLegacyEntryState(entry, "", false),
		Owner: scaffold.OwnerFork, ForkedFromVersion: legacyVersion,
		Reason: "rerun: old path already gone and relocation destination " + newPath + " diverges from the current template; destination adopted as a fork from the legacy entry's recorded state",
	}, nil
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
				Reason: "unmodified and path unchanged; left in place, the chained v2 upgrade converges it per its ownership attribute",
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
	switch kind {
	case OpForkRelocate:
		entry.Owner = scaffold.OwnerFork
		entry.ForkedFromVersion = in.legacyVersion
		entry.Reason = "modified content relocated to " + in.newPath + " as a fork"
	case OpDeleteOldPathAdoptFork:
		// Collision-matrix case (a) with a modified/unmanaged source: the
		// destination already holds this content (a prior interrupted
		// migration run, or otherwise). Record it as a fork rather than
		// letting it fall through to owner=core — see
		// docs/reports/self-review-2026-08-18-overlay-scaffold-v2-p4.md
		// HIGH-1.
		entry.Owner = scaffold.OwnerFork
		entry.ForkedFromVersion = in.legacyVersion
		entry.Reason = "already relocated: destination content matches the modified source; destination adopted as a fork, old path deleted"
	default:
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
//     interrupted migration run, or otherwise). An unmodified source's old
//     path is simply deleted (the destination already holds exactly what
//     the chained v2 upgrade would have produced). A modified/unmanaged
//     source's destination is additionally adopted as a fork (Owner=fork,
//     OpDeleteOldPathAdoptFork): letting it fall through to owner=core
//     instead would permanently lose the fork attribution and turn the
//     user's content into unresolved drift on the very next upgrade — see
//     docs/reports/self-review-2026-08-18-overlay-scaffold-v2-p4.md HIGH-1.
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
		if sourceUnmodified {
			return OpDeleteOldPath, ""
		}
		return OpDeleteOldPathAdoptFork, ""
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
		{"Delete (relocated, fork adopted)", OpDeleteOldPathAdoptFork},
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

// ---------------------------------------------------------------------
// Migration execution (plan Scope C, slice 3).
//
// runMigrateLegacy is the entry point `runUpgradeIOWithOptions` (upgrade.go)
// calls for any manifest whose meta.layout is not v2: git preflight ->
// preview -> confirm -> preflight-validated execution -> v3 manifest (commit
// barrier) -> chained v2 upgrade -> migration report. Every step before the
// commit barrier only reads; the barrier is the single point after which the
// legacy manifest is considered migrated.
// ---------------------------------------------------------------------

// migrationReportDir mirrors internal/upgrade's own report directory
// constant (unexported there): every migration report is written under
// docs/reports/, validated by the same upgrade.WriteUpgradeReport
// containment checks the v2 upgrade report uses.
const migrationReportDir = "docs/reports"

// runMigrateLegacy migrates a legacy (v1/v2, non-overlay) manifest to the v2
// overlay layout, then chains into the ordinary v2 upgrade engine
// (runUpgradeV2) so dispatcher/.ralph/core/new-file convergence happens in
// the same run. now is threaded through (rather than calling time.Now here)
// so report timestamps stay reproducible in tests, matching runUpgradeV2's
// own convention.
func runMigrateLegacy(absDir, manifestPath string, oldManifest *scaffold.Manifest, opts upgradeOptions, in io.Reader, out, errOut io.Writer, colorize bool, now time.Time) error {
	if err := checkGitCleanForMigration(absDir); err != nil {
		return err
	}

	baseFS, err := scaffold.BaseFS()
	if err != nil {
		return fmt.Errorf("loading templates: %w", err)
	}
	desired, _, _, _, err := buildDesiredStateV2(baseFS, oldManifest, errOut)
	if err != nil {
		return fmt.Errorf("building desired state: %w", err)
	}

	plan, err := ClassifyMigration(oldManifest, absDir, desired)
	if err != nil {
		return fmt.Errorf("classifying migration: %w", err)
	}

	writef(out, "%s\n", RenderMigrationPreview(plan))

	if len(plan.Collisions) > 0 {
		writef(errOut, "Migration blocked: %d relocation destination(s) diverge from both the source and the new template; resolve manually (see the collisions listed above) and re-run.\n", len(plan.Collisions))
		return fmt.Errorf("migration blocked by %d collision(s); no files were changed", len(plan.Collisions))
	}

	if opts.DryRun {
		writef(out, "\n--dry-run: preview only, no files were changed.\n")
		return nil
	}

	confirmed, err := confirmMigration(in, out, opts.Yes)
	if err != nil {
		return fmt.Errorf("reading migration confirmation: %w", err)
	}
	if !confirmed {
		writef(out, "\nMigration aborted; no files were changed.\n")
		return nil
	}

	for _, e := range plan.Entries {
		if err := validateMigrationOp(absDir, e); err != nil {
			return fmt.Errorf("migration preflight check failed for %s: %w; no files were changed — fix the underlying issue and re-run", e.OldPath, err)
		}
	}

	if err := executeMigrationEntries(absDir, desired, plan.Entries); err != nil {
		return fmt.Errorf("migration failed partway through: %w; the legacy manifest was not advanced (the commit barrier), so the tree is only partially updated — inspect `git status`/`git diff` (git is the migration's rollback mechanism; no backups are made) to restore or fix forward, then re-run `ralph upgrade` to complete the remaining work", err)
	}

	newManifest, err := buildMigratedManifest(Version, oldManifest, absDir, desired, plan)
	if err != nil {
		return fmt.Errorf("building v3 manifest after migration: %w", err)
	}
	if err := newManifest.Write(manifestPath); err != nil {
		return fmt.Errorf("writing v3 manifest: %w", err)
	}

	writef(out, "Migration complete: legacy layout -> v2 overlay layout (%s)\n", Version)

	if reportContent, rerr := renderMigrationReport(plan, desired, absDir, now); rerr != nil {
		writef(errOut, "Warning: could not render migration report: %v\n", rerr)
	} else {
		reportRelPath := migrationReportRelPath(now)
		if werr := upgrade.WriteUpgradeReport(absDir, reportRelPath, reportContent); werr != nil {
			writef(errOut, "Warning: could not write migration report: %v\n", werr)
		} else {
			writef(out, "  report: %s\n", reportRelPath)
		}
	}

	// Chain into the v2 upgrade engine so dispatcher/.ralph/core/new-file
	// convergence happens in the same run. Per the plan's design decision, a
	// failure here is a warning, not fatal: the migration itself already
	// committed (the manifest barrier above succeeded), and a subsequent
	// `ralph upgrade` re-run converges the rest through the ordinary v2
	// path. ErrUpgradeDriftRemaining is the one exception: it is not a
	// genuine execution error but the chained engine's ordinary "completed,
	// but N paths are left in unresolved drift" outcome, which
	// cmd/ralph/main.go maps to exit code 3 (spec FR-4). Swallowing it here
	// as a warning would make the migration run -- the run most likely to
	// surface drift -- exit 0 anyway, hiding the machine-detectable signal
	// for exactly one run (self-review MEDIUM-1,
	// docs/reports/self-review-2026-08-18-overlay-scaffold-v2-p4.md).
	if err := runUpgradeV2(absDir, manifestPath, newManifest, opts, out, errOut, colorize, now); err != nil {
		if errors.Is(err, ErrUpgradeDriftRemaining) {
			return err
		}
		writef(errOut, "Warning: migration completed, but the chained v2 upgrade reported an issue: %v\n", err)
		writef(errOut, "  re-run `ralph upgrade` to complete convergence via the v2 engine.\n")
	}

	return nil
}

// checkGitCleanForMigration enforces the plan's "非 git ターゲットは移行拒否"
// design decision: git is the migration's only rollback mechanism (no
// backup directory is made), so absDir must be a git work tree with no
// uncommitted changes before any migration write happens.
func checkGitCleanForMigration(absDir string) error {
	gitBin, err := exec.LookPath("git")
	if err != nil {
		return fmt.Errorf("migration requires git (it is the migration's rollback mechanism; no backups are made): git was not found in PATH; no files were changed")
	}

	treeCmd := exec.Command(gitBin, "-C", absDir, "rev-parse", "--is-inside-work-tree")
	treeOut, err := treeCmd.Output()
	if err != nil || strings.TrimSpace(string(treeOut)) != "true" {
		return fmt.Errorf("migration requires a git work tree (git is the migration's rollback mechanism; no backups are made): %s is not inside a git work tree; no files were changed", absDir)
	}

	statusCmd := exec.Command(gitBin, "-C", absDir, "status", "--porcelain")
	var statusErr bytes.Buffer
	statusCmd.Stderr = &statusErr
	statusOut, err := statusCmd.Output()
	if err != nil {
		if msg := strings.TrimSpace(statusErr.String()); msg != "" {
			return fmt.Errorf("checking git status before migration: %w: %s", err, msg)
		}
		return fmt.Errorf("checking git status before migration: %w", err)
	}
	if strings.TrimSpace(string(statusOut)) != "" {
		return fmt.Errorf("migration requires a clean git work tree (git is the migration's rollback mechanism; no backups are made): %s has uncommitted changes; commit or stash them first; no files were changed", absDir)
	}
	return nil
}

// confirmMigration implements the plan's confirm UX: autoYes (--yes) skips
// the prompt; otherwise a y/N prompt is read from in. A non-interactive EOF
// (no input available) reads as an empty line, which is treated as "no" —
// aborting safely rather than blocking or erroring.
func confirmMigration(in io.Reader, out io.Writer, autoYes bool) (bool, error) {
	if autoYes {
		return true, nil
	}
	writef(out, "Proceed with migration? [y/N] ")
	line, err := bufio.NewReader(in).ReadString('\n')
	if err != nil && !errors.Is(err, io.EOF) {
		return false, err
	}
	line = strings.TrimSpace(strings.ToLower(line))
	return line == "y" || line == "yes", nil
}

// validateMigrationOp implements the plan's AC-14/AC-16 preflight: every
// planned file operation is validated (clean path, real parent chain, leaf
// is a regular file or absent) before any migration write happens. Called
// once per plan.Entries item, in a batch pass that must fully complete
// before executeMigrationEntries writes anything.
//
// The real-parent-chain check (upgrade.ValidateRealParentChain) is hoisted
// here, ahead of the per-kind dispatch, for every kind that names a path via
// OldPath: this covers delete kinds (OpDeleteOldPath,
// OpDeleteOldPathAdoptFork, via validateMigrationLeaf) and the baseline
// directory removal (OpDeleteDir, via validateMigrationDirOp), neither of
// which validated it on their own before -- a symlinked intermediate
// directory could route os.Remove/os.RemoveAll through the link and delete
// outside absDir (self-review MEDIUM-2,
// docs/reports/self-review-2026-08-18-overlay-scaffold-v2-p4.md). Write-target
// kinds (OpForkRelocate's NewPath, OpReplaceWithTemplate, OpSettingsPrune)
// already validate their own chain via validateMigrationWriteTarget; the
// hoisted call here additionally covers OpForkRelocate's OldPath (the
// relocation source), which validateMigrationWriteTarget never saw. Placing
// the check at this single dispatch point, ahead of the switch, means no
// future MigrationOpKind can be added without it.
//
// OpDeleteOldPathAdoptFork additionally validates NewPath via
// validateMigrationForkAdoptionTarget: executeMigrationEntries never writes
// NewPath for this kind (only OldPath is deleted), so NewPath's existing
// disk content is trusted as-is and recorded into the v3 manifest as the
// adopted fork (buildMigratedManifest's forkByPath handling) -- a symlinked
// NewPath parent or a non-regular NewPath leaf must be refused before
// OldPath is deleted, the same way a write target is refused (AR#1,
// docs/reports/cross-review-triage-overlay-scaffold-v2-p4.md).
func validateMigrationOp(absDir string, e MigrationEntry) error {
	if err := upgrade.ValidateRealParentChain(absDir, e.OldPath); err != nil {
		return err
	}
	switch e.Kind {
	case OpDeleteOldPath, OpDeleteOldPathAdoptFork:
		if err := validateMigrationLeaf(absDir, e.OldPath, false /* mustExist */); err != nil {
			return err
		}
		if e.Kind == OpDeleteOldPathAdoptFork {
			return validateMigrationForkAdoptionTarget(absDir, e.NewPath)
		}
		return nil
	case OpForkRelocate:
		if err := validateMigrationLeaf(absDir, e.OldPath, true /* mustExist */); err != nil {
			return err
		}
		return validateMigrationWriteTarget(absDir, e.NewPath)
	case OpReplaceWithTemplate:
		if err := validateMigrationWriteTarget(absDir, e.NewPath); err != nil {
			return err
		}
		if e.SnapshotCreate {
			return validateMigrationWriteTarget(absDir, upgrade.SettingsSnapshotRelPath)
		}
		return nil
	case OpSettingsPrune:
		return validateMigrationWriteTarget(absDir, e.OldPath)
	case OpDeleteDir:
		return validateMigrationDirOp(absDir, e.OldPath)
	default: // OpKeepInPlace, OpForkInPlace, OpUntouched: no write.
		return nil
	}
}

// validateMigrationLeaf validates relPath's own Lstat state: regular file or
// absent is always accepted; anything else (symlink, directory, other
// non-regular entry) is refused. mustExist additionally refuses an absent
// leaf.
func validateMigrationLeaf(absDir, relPath string, mustExist bool) error {
	clean, err := scaffold.CleanLocalRelPath(relPath)
	if err != nil {
		return fmt.Errorf("%s: %w", relPath, err)
	}
	full := filepath.Join(absDir, clean)
	fi, err := os.Lstat(full)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			if mustExist {
				return fmt.Errorf("%s: expected to exist but is missing", relPath)
			}
			return nil
		}
		return fmt.Errorf("%s: lstat: %w", relPath, err)
	}
	if !fi.Mode().IsRegular() {
		return fmt.Errorf("%s: refusing to operate on a non-regular file (mode %s)", relPath, fi.Mode())
	}
	return nil
}

// validateMigrationWriteTarget validates a path migration is about to write
// to: the path itself must be clean/local, every existing parent path
// component must be a real directory (never a symlink — closes the same gap
// upgrade.ValidateRealParentChain's doc comment describes for ApplyOps), and
// the leaf itself must be a regular file or absent.
func validateMigrationWriteTarget(absDir, relPath string) error {
	if _, err := scaffold.CleanLocalRelPath(relPath); err != nil {
		return fmt.Errorf("%s: %w", relPath, err)
	}
	if err := upgrade.ValidateRealParentChain(absDir, relPath); err != nil {
		return fmt.Errorf("%s: %w", relPath, err)
	}
	return validateMigrationLeaf(absDir, relPath, false /* mustExist */)
}

// validateMigrationForkAdoptionTarget validates NewPath for an
// OpDeleteOldPathAdoptFork entry: the path itself must be clean/local, every
// existing parent path component must be a real directory (never a symlink
// -- the same chain validateMigrationWriteTarget applies to write targets),
// and the leaf must already exist as a regular file. mustExist is true here
// (unlike validateMigrationWriteTarget): this kind never writes NewPath
// (executeMigrationEntries only deletes OldPath for it), so its content is
// read as-is and trusted as the adopted fork -- an absent or non-regular
// leaf means there is nothing safe to adopt (AR#1,
// docs/reports/cross-review-triage-overlay-scaffold-v2-p4.md).
func validateMigrationForkAdoptionTarget(absDir, relPath string) error {
	if _, err := scaffold.CleanLocalRelPath(relPath); err != nil {
		return fmt.Errorf("%s: %w", relPath, err)
	}
	if err := upgrade.ValidateRealParentChain(absDir, relPath); err != nil {
		return fmt.Errorf("%s: %w", relPath, err)
	}
	return validateMigrationLeaf(absDir, relPath, true /* mustExist */)
}

// validateMigrationDirOp validates a directory migration is about to
// os.RemoveAll (only .ralph/baseline/ today): absent is fine (nothing to
// do); present must be a real directory, never a symlink.
func validateMigrationDirOp(absDir, relPath string) error {
	clean, err := scaffold.CleanLocalRelPath(relPath)
	if err != nil {
		return fmt.Errorf("%s: %w", relPath, err)
	}
	full := filepath.Join(absDir, clean)
	fi, err := os.Lstat(full)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return fmt.Errorf("%s: lstat: %w", relPath, err)
	}
	if fi.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("%s: refusing to remove a symlink", relPath)
	}
	if !fi.IsDir() {
		return fmt.Errorf("%s: expected a directory", relPath)
	}
	return nil
}

// executeMigrationEntries applies every plan.Entries write in order: fork
// relocations (read source, write destination, delete source), template
// replacements (special faces, plus a fresh settings snapshot when
// SnapshotCreate is set), settings pruning, plain deletes (including
// OpDeleteOldPathAdoptFork, whose delete is identical to OpDeleteOldPath's --
// removeMigrationPath already tolerates an already-absent old path, which is
// exactly the state a rerun-detected adopt-fork entry is in), and the
// .ralph/baseline/ directory removal. OpKeepInPlace, OpForkInPlace, and
// OpUntouched entries never write — their content converges via the chained
// v2 upgrade (KeepInPlace) or is deliberately left alone (ForkInPlace,
// Untouched). Callers must run validateMigrationOp over every entry first
// (runMigrateLegacy does); this function does not re-validate.
//
// entries is indexed (not ranged by value) so the OpSettingsPrune case can
// write the actually-removed commands and near-misses back into the
// caller's plan.Entries slice (same backing array) for renderMigrationReport
// to read afterward — see MigrationEntry.PrunedHookCommands' doc.
func executeMigrationEntries(absDir string, desired map[string][]byte, entries []MigrationEntry) error {
	for i := range entries {
		e := entries[i]
		switch e.Kind {
		case OpDeleteOldPath, OpDeleteOldPathAdoptFork:
			if err := removeMigrationPath(absDir, e.OldPath); err != nil {
				return fmt.Errorf("deleting %s: %w", e.OldPath, err)
			}
		case OpForkRelocate:
			if err := relocateMigrationFile(absDir, e.OldPath, e.NewPath); err != nil {
				return fmt.Errorf("relocating %s -> %s: %w", e.OldPath, e.NewPath, err)
			}
		case OpReplaceWithTemplate:
			content, ok := desired[e.NewPath]
			if !ok {
				return fmt.Errorf("replacing %s: no desired-state content for this path", e.NewPath)
			}
			if err := writeMigrationFile(absDir, e.NewPath, content); err != nil {
				return fmt.Errorf("replacing %s: %w", e.NewPath, err)
			}
			if e.SnapshotCreate {
				if err := writeMigrationFile(absDir, upgrade.SettingsSnapshotRelPath, content); err != nil {
					return fmt.Errorf("writing settings snapshot: %w", err)
				}
			}
		case OpSettingsPrune:
			if len(e.PrunedHookCommands) == 0 {
				// Nothing classification found as even a candidate: skip
				// the read/parse/rewrite entirely rather than
				// unconditionally re-marshaling (and losing key order /
				// HTML-escaping) a settings.json that would come out
				// byte-identical in content anyway.
				continue
			}
			current, hasDisk, err := readDiskFileForMigration(absDir, e.OldPath)
			if err != nil {
				return fmt.Errorf("reading %s: %w", e.OldPath, err)
			}
			if !hasDisk {
				continue
			}
			pruned, removed, nearMisses, err := pruneLegacySettingsHooks(current, e.PrunedHookCommands)
			if err != nil {
				return fmt.Errorf("pruning legacy hooks from %s: %w", e.OldPath, err)
			}
			if err := writeMigrationFile(absDir, e.OldPath, pruned); err != nil {
				return fmt.Errorf("writing pruned %s: %w", e.OldPath, err)
			}
			entries[i].PrunedHookCommands = removed
			entries[i].PrunedHookNearMisses = nearMisses
		case OpDeleteDir:
			if err := removeMigrationDir(absDir, e.OldPath); err != nil {
				return fmt.Errorf("removing %s: %w", e.OldPath, err)
			}
		case OpKeepInPlace, OpForkInPlace, OpUntouched:
			// No write: content converges via the chained v2 upgrade
			// (KeepInPlace) or is deliberately left alone.
		}
	}
	return nil
}

// writeMigrationFile writes content to absDir/relPath, creating parent
// directories as needed. Callers must have already validated relPath via
// validateMigrationWriteTarget.
func writeMigrationFile(absDir, relPath string, content []byte) error {
	clean, err := scaffold.CleanLocalRelPath(relPath)
	if err != nil {
		return err
	}
	full := filepath.Join(absDir, clean)
	if err := os.MkdirAll(filepath.Dir(full), 0755); err != nil {
		return err
	}
	return os.WriteFile(full, content, scaffold.FilePerm(relPath))
}

// removeMigrationPath removes absDir/relPath. Absent is not an error
// (matches upgrade.ApplyOps' OpDelete semantics: a path already gone —
// e.g. from a prior interrupted migration run — is simply settled).
func removeMigrationPath(absDir, relPath string) error {
	clean, err := scaffold.CleanLocalRelPath(relPath)
	if err != nil {
		return err
	}
	full := filepath.Join(absDir, clean)
	if err := os.Remove(full); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}

// relocateMigrationFile reads oldPath's content, writes it to newPath, then
// removes oldPath — in that order, so a failure between the write and the
// delete leaves both paths present rather than losing content.
func relocateMigrationFile(absDir, oldPath, newPath string) error {
	content, hasDisk, err := readDiskFileForMigration(absDir, oldPath)
	if err != nil {
		return err
	}
	if !hasDisk {
		return fmt.Errorf("source is missing at execution time (changed since the migration plan was built)")
	}
	if err := writeMigrationFile(absDir, newPath, content); err != nil {
		return err
	}
	return removeMigrationPath(absDir, oldPath)
}

// removeMigrationDir removes absDir/relPath recursively (only
// .ralph/baseline/ today). Callers must have already validated relPath via
// validateMigrationDirOp.
func removeMigrationDir(absDir, relPath string) error {
	clean, err := scaffold.CleanLocalRelPath(relPath)
	if err != nil {
		return err
	}
	return os.RemoveAll(filepath.Join(absDir, clean))
}

// pruneLegacySettingsHooks removes every hooks.<Event>[].hooks[] entry whose
// "command" field exactly matches one of commands from content's "hooks"
// object, dropping now-empty matcher entries and now-empty event arrays
// along with them. Everything else in content (env, permissions, user hook
// entries, unrelated top-level keys) is preserved. Matching is exact-string
// only, deliberately: an argument-carrying variant of a known command (e.g.
// "./.claude/hooks/pre_bash_guard.sh --verbose") is treated as a user
// customization and left in place rather than pruned — reported back via
// nearMisses so an operator does not mistake its survival for an oversight
// (self-review MEDIUM-4).
//
// Returns the pruned content, the subset of commands actually removed
// (sorted, for deterministic reporting — a strict subset of the commands
// argument whenever an argument-carrying variant is present in content but
// not an exact match), and any near-miss commands left in place for that
// reason.
//
// The result is re-marshaled as 2-space-indented JSON. Key order is not
// preserved (unlike internal/upgrade's settings merge, which keeps an
// order-preserving JSON model for exactly this reason) because this output
// is a transient step: whenever the chained v2 upgrade's own 3-way settings
// merge (internal/upgrade.MergeOwnedSettings) actually changes something —
// which it does on an ordinary legacy migration, since the dispatcher
// command itself is new — it re-canonicalizes the result immediately
// afterward in the same runMigrateLegacy call. That re-canonicalization is
// conditional on mergeResult.Changed (see runUpgradeV2), not unconditional;
// executeMigrationEntries skips calling this function at all when there is
// nothing to prune (see its OpSettingsPrune case), so the transient,
// non-order-preserving output only exists when there is content to remove.
func pruneLegacySettingsHooks(content []byte, commands []string) (pruned []byte, removed []string, nearMisses []string, err error) {
	trimmed := bytes.TrimSpace(content)
	if len(trimmed) == 0 {
		return content, nil, nil, nil
	}

	var doc map[string]any
	if err := json.Unmarshal(trimmed, &doc); err != nil {
		return nil, nil, nil, fmt.Errorf("parsing settings.json: %w", err)
	}

	hooksRaw, ok := doc["hooks"]
	if !ok {
		return content, nil, nil, nil
	}
	hooksObj, ok := hooksRaw.(map[string]any)
	if !ok {
		return content, nil, nil, nil
	}

	cmdSet := make(map[string]bool, len(commands))
	for _, c := range commands {
		cmdSet[c] = true
	}
	removedSet := make(map[string]bool)
	nearMissSet := make(map[string]bool)

	for event, entriesRaw := range hooksObj {
		entries, ok := entriesRaw.([]any)
		if !ok {
			continue
		}
		var keptEntries []any
		for _, entryRaw := range entries {
			entry, ok := entryRaw.(map[string]any)
			if !ok {
				keptEntries = append(keptEntries, entryRaw)
				continue
			}
			innerRaw, ok := entry["hooks"].([]any)
			if !ok {
				keptEntries = append(keptEntries, entryRaw)
				continue
			}
			var keptInner []any
			for _, innerRawEntry := range innerRaw {
				innerEntry, ok := innerRawEntry.(map[string]any)
				if !ok {
					keptInner = append(keptInner, innerRawEntry)
					continue
				}
				cmd, _ := innerEntry["command"].(string)
				if cmdSet[cmd] {
					removedSet[cmd] = true
					continue
				}
				if legacyHookNearMiss(cmd) {
					nearMissSet[cmd] = true
				}
				keptInner = append(keptInner, innerRawEntry)
			}
			if len(keptInner) == 0 {
				continue
			}
			entry["hooks"] = keptInner
			keptEntries = append(keptEntries, entry)
		}
		if len(keptEntries) == 0 {
			delete(hooksObj, event)
		} else {
			hooksObj[event] = keptEntries
		}
	}

	if len(hooksObj) == 0 {
		delete(doc, "hooks")
	} else {
		doc["hooks"] = hooksObj
	}

	out, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return nil, nil, nil, fmt.Errorf("marshaling pruned settings.json: %w", err)
	}
	return append(out, '\n'), sortedStringSet(removedSet), sortedStringSet(nearMissSet), nil
}

// legacyHookNearMiss reports whether cmd is an argument-carrying variant of
// a known legacy direct-invocation hook command: it contains one of
// legacyRalphHookCommands as a substring but is not an exact match for it,
// so pruneLegacySettingsHooks's exact-match removal deliberately leaves it
// in place.
func legacyHookNearMiss(cmd string) bool {
	for _, known := range legacyRalphHookCommands {
		if cmd != known && strings.Contains(cmd, known) {
			return true
		}
	}
	return false
}

// sortedStringSet returns set's keys as a sorted slice, or nil when set is
// empty (keeps MigrationEntry.PrunedHookCommands/PrunedHookNearMisses zero
// on the common no-match case, matching legacyRalphHookCommands' own
// already-sorted convention).
func sortedStringSet(set map[string]bool) []string {
	if len(set) == 0 {
		return nil
	}
	out := make([]string, 0, len(set))
	for k := range set {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// buildMigratedManifest builds the v3 manifest a successful migration
// writes as its commit barrier. It is driven primarily by desired (every
// path a fresh v2 project would ship), not by plan.Entries directly, because
// the chained v2 upgrade (runUpgradeV2) that runs immediately afterward
// rebuilds every desired-state path's manifest entry from scratch anyway
// (rebuildManifestV2) except for the two categories this function must get
// right on the first pass: fork entries (carried forward verbatim by
// rebuildManifestV2's owner=fork sweep, so they must already be forks here)
// and OpKeepInPlace entries (must carry the *old* recorded hash forward
// unchanged, not a hash of the new template, so the chained call's core
// replace planner recognizes "unmodified since last recorded state, new
// template differs" and emits a real update instead of a false no-op).
//
// v2SettingsPath and upgrade.SettingsSnapshotRelPath are deliberately
// excluded: the chained v2 upgrade's rebuildManifestV2 always overwrites
// both unconditionally (its switch-case branches for these two paths do not
// gate on "already handled"), so whatever this function might record for
// them would just be discarded a moment later.
//
// For every other path this function does not otherwise handle (the generic
// desired-state sweep below), DiskHash is optimistically recorded as
// templateHash even though the file may not exist on disk yet — the chained
// v2 upgrade (runUpgradeV2) is what creates it. This is safe only because
// that chained call is expected to run in the same pass and correct the
// manifest to match reality (rebuildManifestV2 makes the same assertion, but
// after ApplyOps has already made it true). Since a chained-call failure is
// warning-only for a genuine execution error (see runMigrateLegacy), a
// failed chain can leave this optimistic hash committed until the next
// `ralph upgrade` re-run converges it — see
// docs/reports/self-review-2026-08-18-overlay-scaffold-v2-p4.md LOW (DiskHash
// optimism).
func buildMigratedManifest(version string, oldManifest *scaffold.Manifest, absDir string, desired map[string][]byte, plan MigrationPlan) (*scaffold.Manifest, error) {
	nm := scaffold.NewManifest(version)
	nm.SetLayoutV2()
	nm.Meta.Packs = plan.Packs

	forkByPath := make(map[string]MigrationEntry)
	keepByPath := make(map[string]MigrationEntry)
	for _, e := range plan.Entries {
		switch e.Kind {
		case OpForkRelocate, OpForkInPlace, OpDeleteOldPathAdoptFork:
			forkByPath[e.NewPath] = e
		case OpKeepInPlace:
			keepByPath[e.NewPath] = e
		}
	}

	sortedPaths := make([]string, 0, len(desired))
	for p := range desired {
		sortedPaths = append(sortedPaths, p)
	}
	sort.Strings(sortedPaths)

	handled := make(map[string]bool, len(desired))

	for _, path := range sortedPaths {
		if path == v2SettingsPath || path == upgrade.SettingsSnapshotRelPath {
			continue
		}

		if fe, ok := forkByPath[path]; ok {
			diskContent, _, err := readDiskFileForMigration(absDir, path)
			if err != nil {
				return nil, err
			}
			nm.SetFileFork(path, scaffold.HashBytes(diskContent), fe.ForkedFromVersion)
			handled[path] = true
			continue
		}

		if ke, ok := keepByPath[path]; ok {
			if old, ok := oldManifest.Files[ke.OldPath]; ok {
				old.Owner = ke.Owner
				// The legacy baseline mechanism (and its
				// BaselineStatus/BaselinePath bookkeeping) was removed in
				// Phase 3; a carried-forward BaselinePath can point into
				// .ralph/baseline/, the directory this same migration
				// deletes (OpDeleteDir). Every other v3 write path
				// (SetFileOwned/SetFileFork) already normalizes these
				// away, so this hand-rolled carry-forward must too — see
				// docs/reports/self-review-2026-08-18-overlay-scaffold-v2-p4.md
				// LOW (dangling baseline pointer).
				old.BaselineStatus = scaffold.BaselineStatusMissing
				old.BaselinePath = ""
				nm.Files[path] = old
			} else {
				// Defensive fallback: OpKeepInPlace is only ever produced
				// from an actual legacy manifest entry, so this should not
				// happen. Degrade to a fresh owned entry rather than
				// silently dropping the path.
				templateHash := scaffold.HashBytes(desired[path])
				if err := nm.SetFileOwned(path, ke.Owner, templateHash, templateHash); err != nil {
					return nil, err
				}
			}
			handled[path] = true
			continue
		}

		templateHash := scaffold.HashBytes(desired[path])
		owner := ownerForScaffoldPath(path)
		diskHash := templateHash
		diskExists := false
		if owner == scaffold.OwnerSeed || owner == scaffold.OwnerBlock {
			diskContent, hasDisk, err := readDiskFileForMigration(absDir, path)
			if err != nil {
				return nil, err
			}
			diskExists = hasDisk
			if hasDisk {
				diskHash = scaffold.HashBytes(diskContent)
			}
		}
		if owner == scaffold.OwnerSeed && diskExists && diskHash != templateHash {
			if _, trackedInLegacy := oldManifest.Files[path]; !trackedInLegacy {
				// Untracked-in-legacy-manifest seed path whose pre-existing
				// disk content diverges from the new template: leave it out
				// of the v3 manifest entirely instead of recording
				// SetFileOwned here. The chained v2 upgrade's
				// classifyUntracked (internal/upgrade/replaceplan.go) is what
				// raises the seed advisory for exactly this shape, but only
				// when the path has no manifest entry at all -- recording
				// TemplateHash=current here would make the chained call's
				// classifySeed see "template unchanged since last recorded
				// application" and silently no-op, swallowing the advisory
				// (AR#2, docs/reports/cross-review-triage-overlay-scaffold-v2-p4.md).
				continue
			}
		}
		if err := nm.SetFileOwned(path, owner, templateHash, diskHash); err != nil {
			return nil, fmt.Errorf("setting owner for %s: %w", path, err)
		}
		handled[path] = true
	}

	// Fork entries with no template counterpart at all: a retired path the
	// user had modified, so classifyForkCandidate forked it in place rather
	// than deleting it.
	for path, fe := range forkByPath {
		if handled[path] {
			continue
		}
		diskContent, _, err := readDiskFileForMigration(absDir, path)
		if err != nil {
			return nil, err
		}
		nm.SetFileFork(path, scaffold.HashBytes(diskContent), fe.ForkedFromVersion)
	}

	return nm, nil
}

// migrationReportRelPath returns the manifest-relative path a migration
// report for the given timestamp should be written to:
// docs/reports/ralph-migration-<date>.md.
func migrationReportRelPath(now time.Time) string {
	return path.Join(migrationReportDir, fmt.Sprintf("ralph-migration-%s.md", now.Format("2006-01-02")))
}

// renderMigrationReport renders the migration report: the classification
// listing (RenderMigrationPreview), unified diffs for every forked file
// against its new core counterpart, the settings-prune listing, guidance for
// modified block faces left for the chained upgrade's block engine to
// append onto, and a short next-steps note.
func renderMigrationReport(plan MigrationPlan, desired map[string][]byte, absDir string, now time.Time) ([]byte, error) {
	var b strings.Builder
	fmt.Fprintf(&b, "# Legacy -> v2 Migration Report\n\n")
	fmt.Fprintf(&b, "Generated: %s\n\n", now.Format(time.RFC3339))
	b.WriteString(RenderMigrationPreview(plan))
	b.WriteString("\n")

	forkEntries := entriesForKind(plan.Entries, OpForkRelocate)
	forkEntries = append(forkEntries, entriesForKind(plan.Entries, OpForkInPlace)...)
	// OpDeleteOldPathAdoptFork's NewPath is also a fork -- an already-
	// relocated modified/unmanaged source whose destination was adopted as
	// the fork record (see relocationOutcome) -- and its content is never
	// read anywhere else in this report, so omitting it here silently drops
	// the one diff an operator most needs to review (AR#3,
	// docs/reports/cross-review-triage-overlay-scaffold-v2-p4.md).
	forkEntries = append(forkEntries, entriesForKind(plan.Entries, OpDeleteOldPathAdoptFork)...)
	if len(forkEntries) > 0 {
		sort.Slice(forkEntries, func(i, j int) bool { return forkEntries[i].NewPath < forkEntries[j].NewPath })
		b.WriteString("## Fork diffs\n\n")
		b.WriteString("Forked files are never auto-updated by future `ralph upgrade` runs; they continue to surface an advisory diff against the current core template each time it changes.\n\n")
		for _, e := range forkEntries {
			diskContent, _, err := readDiskFileForMigration(absDir, e.NewPath)
			if err != nil {
				return nil, err
			}
			fmt.Fprintf(&b, "### `%s`\n\n", e.NewPath)
			newTemplate, ok := desired[e.NewPath]
			if !ok {
				b.WriteString("_No corresponding path in the new template set; content preserved as-is._\n\n")
				continue
			}
			diff := upgrade.UnifiedDiff(newTemplate, diskContent, "new core (template)", "fork (your content)")
			if diff == "" {
				b.WriteString("_No differences from the new core template._\n\n")
				continue
			}
			b.WriteString("```diff\n")
			b.WriteString(diff)
			if !strings.HasSuffix(diff, "\n") {
				b.WriteString("\n")
			}
			b.WriteString("```\n\n")
		}
	}

	if pruneEntries := entriesForKind(plan.Entries, OpSettingsPrune); len(pruneEntries) > 0 {
		e := pruneEntries[0]
		b.WriteString("## Settings prune\n\n")
		if len(e.PrunedHookCommands) == 0 {
			b.WriteString("No known legacy direct-invocation hook commands were removed from `.claude/settings.json`.\n\n")
		} else {
			b.WriteString("The following legacy direct-invocation hook commands were removed from `.claude/settings.json` before handing off to the v2 settings merge, so they do not double-fire alongside the dispatcher:\n\n")
			for _, c := range e.PrunedHookCommands {
				fmt.Fprintf(&b, "- `%s`\n", c)
			}
			b.WriteString("\n")
		}
		if len(e.PrunedHookNearMisses) > 0 {
			b.WriteString("The following commands reference a known legacy hook script but were left in place because their arguments differ from the plain invocation; they are treated as a deliberate user customization, not pruned, and may still double-fire alongside the dispatcher — review manually:\n\n")
			for _, c := range e.PrunedHookNearMisses {
				fmt.Fprintf(&b, "- `%s`\n", c)
			}
			b.WriteString("\n")
		}
	}

	if untouched := blockFaceUntouchedEntries(plan.Entries); len(untouched) > 0 {
		b.WriteString("## Block face duplicate-content guidance\n\n")
		b.WriteString("The following files were modified before migration and were left in place; the chained v2 upgrade's block engine will append the ralph-managed block onto them later in this same run, without touching existing content. Legacy-template-derived content outside the block may end up duplicated with the new block's content — review and consolidate manually:\n\n")
		for _, e := range untouched {
			fmt.Fprintf(&b, "- `%s`\n", e.OldPath)
		}
		b.WriteString("\n")
	}

	b.WriteString("## Next steps\n\n")
	b.WriteString("- Review the fork diffs above; forked files are never auto-updated by future `ralph upgrade` runs.\n")
	b.WriteString("- Commit this migration — git was the mechanism used to make it safely reversible.\n")
	b.WriteString("- Re-run `ralph upgrade` any time to converge further template changes.\n")

	return []byte(b.String()), nil
}

// blockFaceUntouchedEntries filters plan.Entries for the modified-block-face
// case (AGENTS.md/.gitignore, OpUntouched): renderMigrationReport surfaces
// these separately since they carry duplicate-content risk once the chained
// upgrade's block engine appends onto them.
func blockFaceUntouchedEntries(entries []MigrationEntry) []MigrationEntry {
	var out []MigrationEntry
	for _, e := range entries {
		if e.Kind != OpUntouched {
			continue
		}
		if e.OldPath == pathAgentsMD || e.OldPath == pathGitignore {
			out = append(out, e)
		}
	}
	return out
}
