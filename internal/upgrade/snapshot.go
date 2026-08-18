package upgrade

import (
	"errors"
	"os"
	"path/filepath"

	"github.com/yoshpy-dev/ralph/internal/scaffold"
)

// settingsSnapshotRelPath is the manifest-relative (and on-disk-relative)
// path of the settings.json snapshot ralph writes on every successful
// upgrade. It records the ralph-owned template content last applied, so a
// later upgrade's 3-way settings merge has an oldOwned side to diff against
// without needing network access or a second embedded template generation.
const settingsSnapshotRelPath = ".ralph/core/settings.ralph.json"

// LoadSettingsSnapshot reads the settings.json snapshot from targetDir. A
// missing snapshot (found=false, err=nil) is the expected state for
// projects scaffolded before the snapshot was introduced (Phase 2 init
// generation); callers fall back to treating oldOwned as "{}" in that case.
// The snapshot path is validated with scaffold.CleanLocalRelPath before use,
// consistent with every other path this package reads.
func LoadSettingsSnapshot(targetDir string) (content []byte, found bool, err error) {
	clean, cerr := scaffold.CleanLocalRelPath(settingsSnapshotRelPath)
	if cerr != nil {
		return nil, false, cerr
	}
	full := filepath.Join(targetDir, clean)
	data, rerr := os.ReadFile(full)
	if rerr != nil {
		if errors.Is(rerr, os.ErrNotExist) {
			return nil, false, nil
		}
		return nil, false, rerr
	}
	return data, true, nil
}
