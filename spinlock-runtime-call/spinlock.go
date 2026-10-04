package mutex

import (
	"runtime"
	"sync/atomic"
)

const (
	unlocked = false
	locked   = true
)

type Mutex struct {
	state atomic.Bool
}

func (m *Mutex) Lock() {
	for !m.state.CompareAndSwap(unlocked, locked) {
		runtime.Gosched() // Yield the processor to allow other goroutines to run
	}
}

func (m *Mutex) Unlock() {
	m.state.Store(unlocked)
}
