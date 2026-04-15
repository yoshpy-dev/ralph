package watcher

import "errors"

// ErrWatcherClosed is returned when the watcher has been stopped.
var ErrWatcherClosed = errors.New("watcher closed")

// ErrTailerClosed is returned when the tailer has been stopped.
var ErrTailerClosed = errors.New("tailer closed")

// StateChangedMsg is sent when a state file in .harness/state/ changes.
type StateChangedMsg struct {
	Path string // absolute path to the changed file
	Op   string // "write", "create", "remove", "rename"
}

// LogLineMsg is sent when a new line is appended to a watched log file.
type LogLineMsg struct {
	SliceName string
	Line      string
}

// WatcherErrorMsg is sent when the watcher encounters an error.
type WatcherErrorMsg struct {
	Err error
}

func (e WatcherErrorMsg) Error() string { return e.Err.Error() }
