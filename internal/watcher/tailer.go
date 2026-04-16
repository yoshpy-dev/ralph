package watcher

import (
	"bufio"
	"errors"
	"io"
	"io/fs"
	"os"
	"sync"
	"time"

	tea "charm.land/bubbletea/v2"
)

const tailBufSize = 4096

// Tailer follows the tail of a log file and delivers new lines as LogLineMsg.
type Tailer struct {
	sliceName    string
	file         *os.File
	offset       int64
	msgCh        chan tea.Msg
	done         chan struct{}
	loopCancel   chan struct{} // per-loop cancel; closed by SwitchFile to stop the current readLoop/waitForFile
	closeOnce    sync.Once
	pollInterval time.Duration
	mu           sync.Mutex
}

// NewTailer opens a log file and seeks to the end. New lines appended after
// this point will be delivered as LogLineMsg via the Tail() command.
// If the file does not exist, the tailer waits for it to appear.
func NewTailer(sliceName, filepath string) (*Tailer, error) {
	t := &Tailer{
		sliceName:    sliceName,
		msgCh:        make(chan tea.Msg, 64),
		done:         make(chan struct{}),
		loopCancel:   make(chan struct{}),
		pollInterval: 200 * time.Millisecond,
	}

	f, err := os.Open(filepath)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			// File doesn't exist yet — tailer will poll until it appears.
			go t.waitForFile(filepath)
			return t, nil
		}
		return nil, err
	}

	// Seek to end so we only get new lines.
	offset, err := f.Seek(0, io.SeekEnd)
	if err != nil {
		_ = f.Close()
		return nil, err
	}
	t.file = f
	t.offset = offset

	go t.readLoop()
	return t, nil
}

// Tail returns a tea.Cmd that blocks until the next log line is available.
func (t *Tailer) Tail() tea.Cmd {
	return func() tea.Msg {
		select {
		case msg, ok := <-t.msgCh:
			if !ok {
				return WatcherErrorMsg{Err: ErrTailerClosed}
			}
			return msg
		case <-t.done:
			return WatcherErrorMsg{Err: ErrTailerClosed}
		}
	}
}

// SwitchFile changes the log file being tailed.
func (t *Tailer) SwitchFile(sliceName, filepath string) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	// Cancel the previous readLoop/waitForFile goroutine.
	close(t.loopCancel)
	t.loopCancel = make(chan struct{})

	// Close existing file.
	if t.file != nil {
		_ = t.file.Close()
		t.file = nil
	}

	t.sliceName = sliceName

	f, err := os.Open(filepath)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			go t.waitForFile(filepath)
			return nil
		}
		return err
	}

	offset, err := f.Seek(0, io.SeekEnd)
	if err != nil {
		_ = f.Close()
		return err
	}
	t.file = f
	t.offset = offset

	go t.readLoop()
	return nil
}

// Stop shuts down the tailer and releases resources.
func (t *Tailer) Stop() {
	t.closeOnce.Do(func() {
		close(t.done)
		t.mu.Lock()
		if t.file != nil {
			_ = t.file.Close()
			t.file = nil
		}
		t.mu.Unlock()
	})
}

// readLoop polls the file for new data and sends lines as messages.
func (t *Tailer) readLoop() {
	t.mu.Lock()
	cancel := t.loopCancel
	t.mu.Unlock()

	ticker := time.NewTicker(t.pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			t.readNewLines()
		case <-cancel:
			return
		case <-t.done:
			return
		}
	}
}

// readNewLines reads any new content from the file and sends line messages.
func (t *Tailer) readNewLines() {
	t.mu.Lock()
	f := t.file
	if f == nil {
		t.mu.Unlock()
		return
	}

	// Check if file has grown.
	info, err := f.Stat()
	if err != nil {
		t.mu.Unlock()
		return
	}

	if info.Size() <= t.offset {
		// File was truncated — reset to beginning.
		if info.Size() < t.offset {
			t.offset = 0
			_, _ = f.Seek(0, io.SeekStart)
		}
		t.mu.Unlock()
		return
	}

	_, _ = f.Seek(t.offset, io.SeekStart)
	reader := bufio.NewReaderSize(f, tailBufSize)
	sliceName := t.sliceName
	t.mu.Unlock()

	for {
		line, err := reader.ReadString('\n')
		if len(line) > 0 {
			// Trim trailing newline for clean display.
			trimmed := line
			if len(trimmed) > 0 && trimmed[len(trimmed)-1] == '\n' {
				trimmed = trimmed[:len(trimmed)-1]
			}
			t.mu.Lock()
			t.offset += int64(len(line))
			t.mu.Unlock()

			select {
			case t.msgCh <- LogLineMsg{SliceName: sliceName, Line: trimmed}:
			case <-t.done:
				return
			}
		}
		if err != nil {
			break
		}
	}
}

// waitForFile polls until the file appears, then starts reading.
func (t *Tailer) waitForFile(filepath string) {
	t.mu.Lock()
	cancel := t.loopCancel
	t.mu.Unlock()

	ticker := time.NewTicker(t.pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			f, err := os.Open(filepath)
			if err != nil {
				continue
			}
			t.mu.Lock()
			t.file = f
			t.offset = 0
			t.mu.Unlock()
			t.readLoop()
			return
		case <-cancel:
			return
		case <-t.done:
			return
		}
	}
}
