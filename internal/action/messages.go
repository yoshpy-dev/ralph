package action

// RetryResultMsg is sent when a retry operation completes.
type RetryResultMsg struct {
	SliceName string
	Err       error
	Output    string
}

// AbortResultMsg is sent when an abort operation completes.
type AbortResultMsg struct {
	SliceName string // empty string means abort-all
	Err       error
	Output    string
}

// ExternalDoneMsg is sent when an external process (pager/editor) completes.
type ExternalDoneMsg struct {
	Action string // "pager" or "editor"
	Err    error
}
