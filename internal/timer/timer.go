package timer

import (
	"sync"
	"time"
)

// Timer manages a Pomodoro countdown with focus/break phases.
type Timer struct {
	mu        sync.Mutex
	duration  int // total seconds for current phase
	remaining int
	running   bool
	isBreak   bool

	focusDur int // configured focus duration in seconds
	breakDur int // configured break duration in seconds

	ticker *time.Ticker
	stop   chan struct{}

	// OnTick is called after every tick and on state transitions.
	OnTick func(remaining int, isBreak bool)
}

// New creates a Timer with the given focus and break durations (in minutes).
func New(focusMin, breakMin int) *Timer {
	t := &Timer{
		focusDur: focusMin * 60,
		breakDur: breakMin * 60,
	}
	t.duration = t.focusDur
	t.remaining = t.focusDur
	return t
}

// SetDurations updates focus/break durations (in minutes). Safe to call when paused.
func (t *Timer) SetDurations(focusMin, breakMin int) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.focusDur = focusMin * 60
	t.breakDur = breakMin * 60
	if !t.running {
		if t.isBreak {
			t.duration = t.breakDur
			t.remaining = t.breakDur
		} else {
			t.duration = t.focusDur
			t.remaining = t.focusDur
		}
	}
}

// Start begins (or resumes) the countdown.
func (t *Timer) Start() {
	t.mu.Lock()
	if t.running {
		t.mu.Unlock()
		return
	}
	t.running = true
	t.stop = make(chan struct{})
	t.mu.Unlock()

	// Goroutine drives the tick loop; it exits when stopped or when remaining hits zero.
	go func() {
		t.ticker = time.NewTicker(time.Second)
		defer t.ticker.Stop()
		for {
			select {
			case <-t.stop:
				return
			case <-t.ticker.C:
				t.mu.Lock()
				if t.remaining > 0 {
					t.remaining--
				}
				rem := t.remaining
				brk := t.isBreak
				done := rem == 0
				t.mu.Unlock()

				if t.OnTick != nil {
					t.OnTick(rem, brk)
				}

				if done {
					t.advance()
					return
				}
			}
		}
	}()
}

// Pause halts the countdown without resetting.
func (t *Timer) Pause() {
	t.mu.Lock()
	defer t.mu.Unlock()
	if !t.running {
		return
	}
	t.running = false
	close(t.stop)
}

// Reset stops and restores the current phase to its full duration.
func (t *Timer) Reset() {
	t.mu.Lock()
	wasRunning := t.running
	if wasRunning {
		t.running = false
		close(t.stop)
	}
	if t.isBreak {
		t.duration = t.breakDur
	} else {
		t.duration = t.focusDur
	}
	t.remaining = t.duration
	rem := t.remaining
	brk := t.isBreak
	t.mu.Unlock()

	if t.OnTick != nil {
		t.OnTick(rem, brk)
	}
}

// Skip immediately moves to the next phase.
func (t *Timer) Skip() {
	t.mu.Lock()
	wasRunning := t.running
	if wasRunning {
		t.running = false
		close(t.stop)
	}
	t.mu.Unlock()

	t.transition()

	if wasRunning {
		t.Start()
	}
}

// advance is called internally when remaining hits zero; transitions phases and auto-starts.
func (t *Timer) advance() {
	t.transition()
	t.Start()
}

// transition flips focus<->break and resets remaining to the new phase duration.
func (t *Timer) transition() {
	t.mu.Lock()
	t.isBreak = !t.isBreak
	if t.isBreak {
		t.duration = t.breakDur
	} else {
		t.duration = t.focusDur
	}
	t.remaining = t.duration
	t.running = false
	rem := t.remaining
	brk := t.isBreak
	t.mu.Unlock()

	if t.OnTick != nil {
		t.OnTick(rem, brk)
	}
}

// State returns a snapshot of the current timer state.
func (t *Timer) State() (remaining, duration int, running, isBreak bool) {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.remaining, t.duration, t.running, t.isBreak
}
