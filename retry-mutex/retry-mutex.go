package retrymutex

import (
	"runtime"
	"sync/atomic"
)

const (
	unlocked = false
	locked   = true
)

const retries = 3

type Mutex struct {
	state atomic.Bool
}

func (m *Mutex) Lock() {
	retryNum := retries
	for !m.state.CompareAndSwap(unlocked, locked) {
		retryNum--
		if retryNum <= 0 {
			// If we have retried enough times, yield the processor to allow other goroutines to run
			runtime.Gosched()
			retryNum = retries // Reset retry count after yielding
		}
	}
}

func (m *Mutex) Unlock() {
	m.state.Store(unlocked)
}
