package mutex

import "sync/atomic"

const (
	unlocked = false
	locked   = true
)

type Mutex struct {
	state atomic.Bool
}

func (m *Mutex) Lock() {
	// Pure spin: burn the CPU instead of yielding, unlike spinlock-runtime-call
	for !m.state.CompareAndSwap(unlocked, locked) {
	}
}

func (m *Mutex) Unlock() {
	m.state.Store(unlocked)
}
