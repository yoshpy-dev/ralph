package action

import tea "charm.land/bubbletea/v2"

// AbortSlice requests the orchestrator to abort a specific slice.
// It calls: scripts/ralph abort --slice <sliceName>
func (e *Executor) AbortSlice(sliceName string) tea.Cmd {
	if err := ValidateSliceName(sliceName); err != nil {
		return func() tea.Msg {
			return AbortResultMsg{SliceName: sliceName, Err: err}
		}
	}
	return e.RunAsync(func(output string, err error) tea.Msg {
		return AbortResultMsg{
			SliceName: sliceName,
			Err:       err,
			Output:    output,
		}
	}, "abort", "--slice", sliceName)
}

// AbortAll requests the orchestrator to abort all slices.
// It calls: scripts/ralph abort
func (e *Executor) AbortAll() tea.Cmd {
	return e.RunAsync(func(output string, err error) tea.Msg {
		return AbortResultMsg{
			Err:    err,
			Output: output,
		}
	}, "abort")
}
