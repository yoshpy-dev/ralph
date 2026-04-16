package watcher

import (
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/fsnotify/fsnotify"
)

// Watcher monitors .harness/state/ directories for file changes and delivers
// updates as Bubble Tea messages.
type Watcher struct {
	fsWatcher    *fsnotify.Watcher
	orchDir      string
	worktreeBase string
	msgCh        chan tea.Msg
	done         chan struct{}
	closeOnce    sync.Once
	usePolling   bool
	pollInterval time.Duration
}

// New creates a Watcher for the given orchestrator and worktree directories.
// If fsnotify is unavailable, it falls back to polling at 5-second intervals.
func New(orchDir, worktreeBase string) (*Watcher, error) {
	w := &Watcher{
		orchDir:      orchDir,
		worktreeBase: worktreeBase,
		msgCh:        make(chan tea.Msg, 64),
		done:         make(chan struct{}),
		pollInterval: 5 * time.Second,
	}

	fsw, err := fsnotify.NewWatcher()
	if err != nil {
		w.usePolling = true
		go w.pollLoop()
		return w, nil
	}
	w.fsWatcher = fsw

	// Watch orchestrator directory if it exists.
	if info, err := os.Stat(orchDir); err == nil && info.IsDir() {
		_ = fsw.Add(orchDir)
	}

	// Watch worktree checkpoint files.
	w.addWorktreeWatches()

	go w.eventLoop()
	return w, nil
}

// NewWithPolling creates a polling-only Watcher (useful for testing or when
// fsnotify is unreliable).
func NewWithPolling(orchDir, worktreeBase string, interval time.Duration) *Watcher {
	w := &Watcher{
		orchDir:      orchDir,
		worktreeBase: worktreeBase,
		msgCh:        make(chan tea.Msg, 64),
		done:         make(chan struct{}),
		usePolling:   true,
		pollInterval: interval,
	}
	go w.pollLoop()
	return w
}

// Watch returns a tea.Cmd that blocks until the next state change event.
// Call this from your Bubble Tea Update to receive continuous updates.
func (w *Watcher) Watch() tea.Cmd {
	return func() tea.Msg {
		select {
		case msg, ok := <-w.msgCh:
			if !ok {
				return WatcherErrorMsg{Err: ErrWatcherClosed}
			}
			return msg
		case <-w.done:
			return WatcherErrorMsg{Err: ErrWatcherClosed}
		}
	}
}

// Stop shuts down the watcher and releases all resources.
func (w *Watcher) Stop() error {
	var err error
	w.closeOnce.Do(func() {
		close(w.done)
		if w.fsWatcher != nil {
			err = w.fsWatcher.Close()
		}
	})
	return err
}

// eventLoop reads fsnotify events and converts them to tea.Msg.
func (w *Watcher) eventLoop() {
	for {
		select {
		case event, ok := <-w.fsWatcher.Events:
			if !ok {
				return
			}
			msg := StateChangedMsg{
				Path: event.Name,
				Op:   opString(event.Op),
			}
			select {
			case w.msgCh <- msg:
			case <-w.done:
				return
			}
		case err, ok := <-w.fsWatcher.Errors:
			if !ok {
				return
			}
			select {
			case w.msgCh <- WatcherErrorMsg{Err: err}:
			case <-w.done:
				return
			}
		case <-w.done:
			return
		}
	}
}

// pollLoop periodically scans watched directories for modifications.
func (w *Watcher) pollLoop() {
	modTimes := make(map[string]time.Time)
	// Initial scan to establish baselines.
	w.scanFiles(modTimes, true)

	ticker := time.NewTicker(w.pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			w.scanFiles(modTimes, false)
		case <-w.done:
			return
		}
	}
}

// scanFiles checks for file modifications and sends messages for changed files.
func (w *Watcher) scanFiles(modTimes map[string]time.Time, initial bool) {
	paths := w.collectWatchPaths()
	seen := make(map[string]bool, len(paths))

	for _, p := range paths {
		seen[p] = true
		info, err := os.Stat(p)
		if err != nil {
			if _, known := modTimes[p]; known {
				delete(modTimes, p)
				if !initial {
					w.sendMsg(StateChangedMsg{Path: p, Op: "remove"})
				}
			}
			continue
		}
		prev, known := modTimes[p]
		modTimes[p] = info.ModTime()
		if !initial && (!known || info.ModTime().After(prev)) {
			op := "write"
			if !known {
				op = "create"
			}
			w.sendMsg(StateChangedMsg{Path: p, Op: op})
		}
	}

	// Detect files that were previously known but are no longer in the directory listing.
	if !initial {
		for p := range modTimes {
			if !seen[p] {
				delete(modTimes, p)
				w.sendMsg(StateChangedMsg{Path: p, Op: "remove"})
			}
		}
	}
}

// collectWatchPaths returns all files that should be monitored.
func (w *Watcher) collectWatchPaths() []string {
	var paths []string

	// Orchestrator directory files.
	if entries, err := os.ReadDir(w.orchDir); err == nil {
		for _, e := range entries {
			if !e.IsDir() {
				paths = append(paths, filepath.Join(w.orchDir, e.Name()))
			}
		}
	}

	// Worktree checkpoint files.
	if entries, err := os.ReadDir(w.worktreeBase); err == nil {
		for _, e := range entries {
			if e.IsDir() {
				cp := filepath.Join(w.worktreeBase, e.Name(), ".harness", "state", "pipeline", "checkpoint.json")
				paths = append(paths, cp)
			}
		}
	}

	return paths
}

// addWorktreeWatches adds fsnotify watches for worktree checkpoint directories.
func (w *Watcher) addWorktreeWatches() {
	if w.worktreeBase == "" {
		return
	}
	entries, err := os.ReadDir(w.worktreeBase)
	if err != nil {
		return
	}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		dir := filepath.Join(w.worktreeBase, e.Name(), ".harness", "state", "pipeline")
		if info, err := os.Stat(dir); err == nil && info.IsDir() {
			_ = w.fsWatcher.Add(dir)
		}
	}
}

// sendMsg sends a message to the channel without blocking if closed.
func (w *Watcher) sendMsg(msg tea.Msg) {
	select {
	case w.msgCh <- msg:
	case <-w.done:
	}
}

// opString converts an fsnotify Op to a human-readable string.
func opString(op fsnotify.Op) string {
	parts := make([]string, 0, 4)
	if op.Has(fsnotify.Write) {
		parts = append(parts, "write")
	}
	if op.Has(fsnotify.Create) {
		parts = append(parts, "create")
	}
	if op.Has(fsnotify.Remove) {
		parts = append(parts, "remove")
	}
	if op.Has(fsnotify.Rename) {
		parts = append(parts, "rename")
	}
	if len(parts) == 0 {
		return "unknown"
	}
	return strings.Join(parts, "|")
}
