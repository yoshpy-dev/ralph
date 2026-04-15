package ui

import "github.com/yoshpy-dev/harness-engineering-scaffolding-template/internal/state"

// SliceSelectedMsg is sent when a slice is selected in the slice list.
type SliceSelectedMsg struct {
	Slice state.SliceState
}

// LogLineMsg is sent when a new log line is available.
type LogLineMsg struct {
	Line string
}

// StateUpdatedMsg is sent when the overall state has been refreshed.
type StateUpdatedMsg struct {
	Status state.FullStatus
}
