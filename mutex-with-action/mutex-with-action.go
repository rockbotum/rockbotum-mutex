package mutexwithaction

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

func (m *Mutex) Lock(action func() any) any {
	for !m.state.CompareAndSwap(unlocked, locked) {
		runtime.Gosched() // Yield the processor to allow other goroutines to run
	}
	return action() // Execute the provided action while holding the lock
}

func (m *Mutex) Unlock() {
	m.state.Store(unlocked)
}
