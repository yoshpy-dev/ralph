package action

import tea "charm.land/bubbletea/v2"

// RetrySlice requests the orchestrator to retry a failed or stuck slice.
// It calls: scripts/ralph retry <sliceName>
func (e *Executor) RetrySlice(sliceName string) tea.Cmd {
	if err := ValidateSliceName(sliceName); err != nil {
		return func() tea.Msg {
			return RetryResultMsg{SliceName: sliceName, Err: err}
		}
	}
	return e.RunAsync(func(output string, err error) tea.Msg {
		return RetryResultMsg{
			SliceName: sliceName,
			Err:       err,
			Output:    output,
		}
	}, "retry", sliceName)
}
